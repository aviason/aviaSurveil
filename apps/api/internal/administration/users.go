package administration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	administrationstore "github.com/MarlonJD/aviaSurveil360/apps/api/internal/administration/store/postgres"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/notifications"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/idempotency"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserLifecycleAction string

const (
	UserLifecycleProvision            UserLifecycleAction = "PROVISION"
	UserLifecycleUpdateRoles          UserLifecycleAction = "UPDATE_ROLES"
	UserLifecycleSuspend              UserLifecycleAction = "SUSPEND"
	UserLifecycleReactivate           UserLifecycleAction = "REACTIVATE"
	UserLifecycleDeactivate           UserLifecycleAction = "DEACTIVATE"
	UserLifecycleTransferOrganization UserLifecycleAction = "TRANSFER_ORGANIZATION"
	UserLifecycleResendInvitation     UserLifecycleAction = "RESEND_INVITATION"
	UserLifecycleResetPassword        UserLifecycleAction = "RESET_PASSWORD"
	UserLifecycleResetMFA             UserLifecycleAction = "RESET_MFA"
	UserLifecycleForceLogout          UserLifecycleAction = "FORCE_LOGOUT"
)

type UserLifecycleStatus string

const (
	UserLifecyclePending         UserLifecycleStatus = "PENDING"
	UserLifecycleRunning         UserLifecycleStatus = "RUNNING"
	UserLifecycleSuccess         UserLifecycleStatus = "SUCCEEDED"
	UserLifecycleFailed          UserLifecycleStatus = "FAILED"
	UserLifecycleFailedRetryable UserLifecycleStatus = "FAILED_RETRYABLE"
	UserLifecycleFailedPermanent UserLifecycleStatus = "FAILED_PERMANENT"
	UserLifecycleManualReview    UserLifecycleStatus = "MANUAL_REVIEW"
)

type UserLifecycleRequest struct {
	ID                          string              `json:"id"`
	SubjectID                   string              `json:"subjectId"`
	Action                      UserLifecycleAction `json:"action"`
	Roles                       []identity.Role     `json:"roles"`
	OrganizationID              string              `json:"organizationId"`
	Email                       string              `json:"email,omitempty"`
	DisplayName                 string              `json:"displayName,omitempty"`
	Status                      UserLifecycleStatus `json:"status"`
	IdempotencyKey              string              `json:"idempotencyKey"`
	ExpectedMembershipRevision  int64               `json:"expectedMembershipRevision"`
	ResultingMembershipRevision int64               `json:"resultingMembershipRevision"`
	MembershipID                string              `json:"membershipId,omitempty"`
	Reason                      string              `json:"reason"`
	EffectiveAt                 *time.Time          `json:"effectiveAt,omitempty"`
	ProviderFailureClass        string              `json:"providerFailureClass,omitempty"`
	ProviderAcknowledgedAt      *time.Time          `json:"providerAcknowledgedAt,omitempty"`
	AttemptCount                int                 `json:"attemptCount"`
	RequestedBy                 string              `json:"requestedBySubjectId"`
	OutboxMessageID             string              `json:"outboxMessageId"`
	FailureReason               string              `json:"failureReason,omitempty"`
	CreatedAt                   time.Time           `json:"createdAt"`
	UpdatedAt                   time.Time           `json:"updatedAt"`
}

type RequestUserLifecycleCommand struct {
	OperationID                string
	IdempotencyKey             string
	SubjectID                  string
	Action                     UserLifecycleAction
	Roles                      []identity.Role
	OrganizationID             string
	Email                      string
	DisplayName                string
	Reason                     string
	ExpectedMembershipRevision int64
	EffectiveAt                *time.Time
}

type UserServiceDependencies struct {
	Clock       func() time.Time
	IDGenerator func(string) string
}

type UserService struct {
	pool        *database.Pool
	clock       func() time.Time
	idGenerator func(string) string
}

func NewUserService(pool *database.Pool, dependencies UserServiceDependencies) *UserService {
	clock := dependencies.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := dependencies.IDGenerator
	if idGenerator == nil {
		idGenerator = randomUserID
	}
	return &UserService{pool: pool, clock: clock, idGenerator: idGenerator}
}

func (service *UserService) RequestLifecycle(
	ctx context.Context,
	actor identity.Principal,
	command RequestUserLifecycleCommand,
) (UserLifecycleRequest, error) {
	if !CanManageUsers(actor) {
		return UserLifecycleRequest{}, ErrForbidden
	}
	command = normalizeLifecycleCommand(command)
	if err := validateLifecycleCommand(command); err != nil {
		return UserLifecycleRequest{}, err
	}
	roles := make([]string, len(command.Roles))
	for index, role := range command.Roles {
		roles[index] = string(role)
	}
	semanticHash, err := idempotency.SemanticHash(struct {
		IdempotencyKey             string              `json:"idempotencyKey"`
		SubjectID                  string              `json:"subjectId"`
		Action                     UserLifecycleAction `json:"action"`
		Roles                      []string            `json:"roles"`
		OrganizationID             string              `json:"organizationId"`
		Email                      string              `json:"email,omitempty"`
		DisplayName                string              `json:"displayName,omitempty"`
		Reason                     string              `json:"reason"`
		ExpectedMembershipRevision int64               `json:"expectedMembershipRevision"`
		EffectiveAt                *time.Time          `json:"effectiveAt,omitempty"`
	}{
		command.IdempotencyKey,
		command.SubjectID,
		command.Action,
		roles,
		command.OrganizationID,
		command.Email,
		command.DisplayName,
		command.Reason,
		command.ExpectedMembershipRevision,
		command.EffectiveAt,
	})
	if err != nil {
		return UserLifecycleRequest{}, err
	}
	scope := actor.SubjectID + ":user_lifecycle"
	var output UserLifecycleRequest
	err = database.WithinTransaction(ctx, service.pool, func(ctx context.Context, transaction pgx.Tx) error {
		if _, err := transaction.Exec(
			ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
			scope+":idempotency:"+command.IdempotencyKey,
		); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", scope+":"+command.OperationID); err != nil {
			return err
		}
		if command.Action == UserLifecycleProvision {
			if _, err := transaction.Exec(
				ctx,
				"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
				"user_lifecycle:provision_email:"+command.Email,
			); err != nil {
				return err
			}
		}
		var storedHash string
		var responseBody []byte
		err := transaction.QueryRow(ctx, `
			SELECT semantic_hash, response_body
			FROM idempotency_responses
			WHERE scope = $1 AND operation_id = $2
		`, scope, command.OperationID).Scan(&storedHash, &responseBody)
		if err == nil {
			if storedHash != semanticHash {
				return idempotency.ErrOperationIDReuse
			}
			return json.Unmarshal(responseBody, &output)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if _, err := administrationstore.New(transaction).GetUserLifecycleRequestByIdempotencyKey(
			ctx, command.IdempotencyKey,
		); err == nil {
			return idempotency.ErrOperationIDReuse
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		membershipID, err := validateMembershipExpectation(
			ctx,
			transaction,
			command,
		)
		if err != nil {
			return err
		}
		if command.Action == UserLifecycleProvision {
			var existingRequestID string
			err := transaction.QueryRow(ctx, `
				SELECT id
				FROM user_lifecycle_requests
				WHERE requested_action = 'PROVISION'
				  AND lower(requested_email) = lower($1)
				  AND status IN ('PENDING', 'RUNNING', 'SUCCEEDED')
				LIMIT 1
			`, command.Email).Scan(&existingRequestID)
			if err == nil {
				return ErrConflict
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}

		now := service.clock().UTC()
		if command.Action == UserLifecycleTransferOrganization &&
			(command.EffectiveAt == nil || !command.EffectiveAt.After(now)) {
			return ErrInvalid
		}
		requestID := service.idGenerator("user-lifecycle")
		outboxID := service.idGenerator("outbox-user-lifecycle")
		output = UserLifecycleRequest{
			ID: requestID, SubjectID: command.SubjectID, Action: command.Action,
			Roles: append([]identity.Role(nil), command.Roles...), OrganizationID: command.OrganizationID,
			Email: command.Email, DisplayName: command.DisplayName,
			Status: UserLifecyclePending, IdempotencyKey: command.IdempotencyKey,
			ExpectedMembershipRevision: command.ExpectedMembershipRevision,
			MembershipID:               membershipID, Reason: command.Reason,
			EffectiveAt: command.EffectiveAt,
			RequestedBy: actor.SubjectID, OutboxMessageID: outboxID,
			CreatedAt: now, UpdatedAt: now,
		}
		responseBody, err = json.Marshal(output)
		if err != nil {
			return err
		}
		auditID := service.idGenerator("audit-user-lifecycle")
		actorRole := identity.RoleAdmin
		if _, err := transaction.Exec(ctx, `
			INSERT INTO audit_events (
				event_id, occurred_at, actor_subject_id, actor_role, organization_id,
				action, entity_type, entity_id, entity_version, after_status,
				operation_id, correlation_id, request_id, details
			) VALUES (
				$1, $2, $3, $4, NULLIF($5, ''), $6, 'USER_LIFECYCLE_REQUEST',
				$7, 1, 'PENDING', $8, $8, $8,
				jsonb_build_object(
					'reason', $9::text,
					'expectedMembershipRevision', $10::bigint,
					'membershipId', NULLIF($11::text, '')
				)
			)
		`, auditID, now, actor.SubjectID, string(actorRole), command.OrganizationID,
			lifecycleAuditAction(command.Action), requestID, command.OperationID,
			command.Reason, command.ExpectedMembershipRevision, membershipID); err != nil {
			return fmt.Errorf("append user lifecycle audit event: %w", err)
		}
		var changeSequenceID int64
		if err := transaction.QueryRow(ctx, `
			INSERT INTO authorized_sync_changes (
				subject_id, organization_id, kind, entity_id, entity_revision,
				payload, changed_at, operation_id, correlation_id
			) VALUES ($1, NULLIF($2, ''), 'USER_LIFECYCLE_REQUEST', $3, 1, $4, $5, $6, $6)
			RETURNING sequence_id
		`, actor.SubjectID, command.OrganizationID, requestID, responseBody, now,
			command.OperationID).Scan(&changeSequenceID); err != nil {
			return fmt.Errorf("append user lifecycle change: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO outbox_messages (
				id, topic, aggregate_type, aggregate_id, payload, available_at,
				idempotency_key, operation_id, correlation_id
			) VALUES (
				$1, 'identity.user-lifecycle.requested', 'USER_LIFECYCLE_REQUEST',
				$2, $3, $4, $5, $6, $6
			)
		`, outboxID, requestID, responseBody, lifecycleAvailableAt(command, now),
			"command:"+scope+":"+command.OperationID, command.OperationID); err != nil {
			return fmt.Errorf("enqueue user lifecycle job: %w", err)
		}
		var subjectID *string
		if command.SubjectID != "" {
			subjectID = &command.SubjectID
		}
		organizationID := command.OrganizationID
		outboxMessageID := outboxID
		var email *string
		var displayName *string
		var storedMembershipID *string
		if command.Email != "" {
			email = &command.Email
			displayName = &command.DisplayName
		}
		if membershipID != "" {
			storedMembershipID = &membershipID
		}
		requestedEffectiveAt := pgtype.Timestamptz{}
		if command.EffectiveAt != nil {
			requestedEffectiveAt = pgtype.Timestamptz{
				Time:  command.EffectiveAt.UTC(),
				Valid: true,
			}
		}
		if _, err := administrationstore.New(transaction).CreateUserLifecycleRequest(ctx,
			administrationstore.CreateUserLifecycleRequestParams{
				ID: requestID, SubjectID: subjectID, RequestedAction: string(command.Action),
				RequestedRoles: roles, RequestedOrganizationID: &organizationID,
				RequestedEmail: email, RequestedDisplayName: displayName,
				IdempotencyKey:             command.IdempotencyKey,
				ExpectedMembershipRevision: command.ExpectedMembershipRevision,
				Reason:                     command.Reason,
				RequestedEffectiveAt:       requestedEffectiveAt,
				RequestedBySubjectID:       actor.SubjectID,
				OutboxMessageID:            &outboxMessageID, MembershipID: storedMembershipID,
			},
		); err != nil {
			return fmt.Errorf("persist user lifecycle request: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO idempotency_responses (
				scope, operation_id, semantic_hash, response_status,
				response_headers, response_body, created_at
			) VALUES ($1, $2, $3, 202, '{}'::jsonb, $4, $5)
		`, scope, command.OperationID, semanticHash, responseBody, now); err != nil {
			return fmt.Errorf("store user lifecycle idempotency response: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO command_transaction_links (
				operation_id, idempotency_scope, audit_event_id,
				change_sequence_id, outbox_message_id, created_at
			) VALUES ($1, $2, $3, $4, $5, $6)
		`, command.OperationID, scope, auditID, changeSequenceID, outboxID, now); err != nil {
			return fmt.Errorf("link user lifecycle command: %w", err)
		}
		return nil
	})
	return output, err
}

func (service *UserService) GetLifecycle(
	ctx context.Context,
	actor identity.Principal,
	requestID string,
) (UserLifecycleRequest, error) {
	if !CanManageUsers(actor) {
		return UserLifecycleRequest{}, ErrForbidden
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return UserLifecycleRequest{}, ErrInvalid
	}
	var output UserLifecycleRequest
	var subjectID, organizationID, email, displayName, outboxID, failure *string
	var membershipID, providerFailureClass *string
	var resultingRevision *int64
	var providerAcknowledgedAt, effectiveAt *time.Time
	var action, status string
	var roles []string
	err := service.pool.QueryRow(ctx, `
		SELECT id, subject_id, requested_action, requested_roles,
		       requested_organization_id, requested_email,
		       requested_display_name, status, idempotency_key,
		       expected_membership_revision, reason, requested_effective_at,
		       membership_id,
		       resulting_membership_revision, provider_failure_class,
		       provider_acknowledged_at, requested_by_subject_id,
		       outbox_message_id, failure_reason,
		       COALESCE((
		           SELECT attempt_count
		           FROM outbox_messages
		           WHERE id = user_lifecycle_requests.outbox_message_id
		       ), 0),
		       created_at, updated_at
		FROM user_lifecycle_requests
		WHERE id = $1
	`, requestID).Scan(
		&output.ID,
		&subjectID,
		&action,
		&roles,
		&organizationID,
		&email,
		&displayName,
		&status,
		&output.IdempotencyKey,
		&output.ExpectedMembershipRevision,
		&output.Reason,
		&effectiveAt,
		&membershipID,
		&resultingRevision,
		&providerFailureClass,
		&providerAcknowledgedAt,
		&output.RequestedBy,
		&outboxID,
		&failure,
		&output.AttemptCount,
		&output.CreatedAt,
		&output.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return UserLifecycleRequest{}, ErrNotFound
	}
	if err != nil {
		return UserLifecycleRequest{}, err
	}
	if subjectID != nil {
		output.SubjectID = *subjectID
	}
	if organizationID != nil {
		output.OrganizationID = *organizationID
	}
	if email != nil {
		output.Email = *email
	}
	if displayName != nil {
		output.DisplayName = *displayName
	}
	if outboxID != nil {
		output.OutboxMessageID = *outboxID
	}
	if failure != nil {
		output.FailureReason = *failure
	}
	if membershipID != nil {
		output.MembershipID = *membershipID
	}
	if resultingRevision != nil {
		output.ResultingMembershipRevision = *resultingRevision
	}
	if providerFailureClass != nil {
		output.ProviderFailureClass = *providerFailureClass
	}
	output.ProviderAcknowledgedAt = providerAcknowledgedAt
	output.EffectiveAt = effectiveAt
	output.Action = UserLifecycleAction(action)
	output.Status = UserLifecycleStatus(status)
	output.Roles = make([]identity.Role, len(roles))
	for index, role := range roles {
		output.Roles[index] = identity.Role(role)
	}
	return output, nil
}

func (service *UserService) ReconcileActivatedMembership(
	ctx context.Context,
	subjectID string,
	expectedMembershipRevision int64,
	requiredActions []string,
	providerMFAEnrolled bool,
) error {
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" || expectedMembershipRevision < 1 ||
		len(requiredActions) != 0 {
		return ErrInvalid
	}
	now := service.clock().UTC()
	var outcome error
	err := database.WithinTransaction(
		ctx,
		service.pool,
		func(ctx context.Context, transaction pgx.Tx) error {
			if _, err := transaction.Exec(
				ctx,
				"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
				"desired-membership:"+subjectID,
			); err != nil {
				return err
			}
			var membershipID, state, organizationID, requestedBy string
			var revision int64
			var roles []string
			if err := transaction.QueryRow(ctx, `
				SELECT membership_id, revision, membership_state,
				       organization_id, roles, requested_by_subject_id
				FROM desired_membership_versions
				WHERE subject_id = $1
				ORDER BY revision DESC
				LIMIT 1
			`, subjectID).Scan(
				&membershipID,
				&revision,
				&state,
				&organizationID,
				&roles,
				&requestedBy,
			); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return ErrMembershipRevisionConflict
				}
				return err
			}
			if state == "ACTIVE" &&
				revision == expectedMembershipRevision+1 {
				var consumed bool
				if err := transaction.QueryRow(ctx, `
					SELECT EXISTS (
						SELECT 1
						FROM identity_action_facts
						WHERE membership_id = $1
						  AND action_kind = 'INVITATION'
						  AND state = 'CONSUMED'
					)
				`, membershipID).Scan(&consumed); err != nil {
					return err
				}
				if consumed {
					return nil
				}
			}
			if revision != expectedMembershipRevision {
				return ErrMembershipRevisionConflict
			}
			if state != "INVITED" {
				return fmt.Errorf(
					"%w: membership state %q is not invited",
					ErrConflict,
					state,
				)
			}
			var requestID, invitationReason, invitationState string
			var factSequence, deliveryAttempt int
			var expiresAt *time.Time
			if err := transaction.QueryRow(ctx, `
				SELECT request_id, fact_sequence, state, delivery_attempt,
				       expires_at, reason
				FROM identity_action_facts
				WHERE membership_id = $1
				  AND action_kind = 'INVITATION'
				ORDER BY created_at DESC, fact_sequence DESC, id DESC
				LIMIT 1
			`, membershipID).Scan(
				&requestID,
				&factSequence,
				&invitationState,
				&deliveryAttempt,
				&expiresAt,
				&invitationReason,
			); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return fmt.Errorf(
						"%w: no invitation delivery fact",
						ErrConflict,
					)
				}
				return err
			}
			if invitationState != string(
				notifications.IdentityDeliveryAccepted,
			) {
				return fmt.Errorf(
					"%w: invitation state %q is not delivery-accepted",
					ErrConflict,
					invitationState,
				)
			}
			if expiresAt == nil || !expiresAt.After(now) {
				if err := notifications.ValidateIdentityDeliveryTransition(
					notifications.IdentityDeliveryState(invitationState),
					notifications.IdentityDeliveryExpired,
				); err != nil {
					return err
				}
				if _, err := transaction.Exec(ctx, `
					INSERT INTO identity_action_facts (
						id, request_id, fact_sequence, membership_id,
						subject_id, action_kind, state, delivery_attempt,
						expires_at, reason, created_at
					) VALUES (
						$1, $2, $3, $4, $5, 'INVITATION', 'EXPIRED',
						$6, $7, $8, $9
					)
				`, service.idGenerator("identity-action-fact"), requestID,
					factSequence+1, membershipID, subjectID, deliveryAttempt,
					expiresAt, invitationReason, now); err != nil {
					return fmt.Errorf("append expired invitation fact: %w", err)
				}
				outcome = ErrInvitationExpired
				return nil
			}
			activationReason :=
				"Provider required actions completed at first login."
			nextRevision := revision + 1
			if err := notifications.ValidateIdentityDeliveryTransition(
				notifications.IdentityDeliveryState(invitationState),
				notifications.IdentityDeliveryConsumed,
			); err != nil {
				return err
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO desired_membership_versions (
					membership_id, subject_id, revision, membership_state,
					organization_id, roles, requested_by_subject_id, reason,
					source_request_id, requested_at, effective_at,
					observed_provider_enabled, observed_organization_id,
					observed_roles, observed_at, drift_state
				) VALUES (
					$1, $2, $3, 'ACTIVE', $4, $5, $2, $6, $7, $8, $8,
					true, $4, $5, $8, 'IN_SYNC'
				)
			`, membershipID, subjectID, nextRevision, organizationID, roles,
				activationReason, requestID, now); err != nil {
				return fmt.Errorf("append activated membership version: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
				UPDATE desired_membership_sync
				SET desired_revision = $2,
				    observed_provider_enabled = true,
				    observed_organization_id = $3,
				    observed_roles = $4,
				    observed_at = $5,
				    drift_state = 'IN_SYNC'
				WHERE membership_id = $1
			`, membershipID, nextRevision, organizationID, roles, now); err != nil {
				return fmt.Errorf("synchronize activated membership: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO identity_action_facts (
					id, request_id, fact_sequence, membership_id, subject_id,
					action_kind, state, delivery_attempt, expires_at,
					provider_acknowledged_at, reason, created_at
				) VALUES (
					$1, $2, $3, $4, $5, 'INVITATION', 'CONSUMED',
					$6, $7, $8, $9, $8
				)
			`, service.idGenerator("identity-action-fact"), requestID,
				factSequence+1, membershipID, subjectID, deliveryAttempt,
				expiresAt, now, activationReason); err != nil {
				return fmt.Errorf("append consumed invitation fact: %w", err)
			}
			actorRole := "auditee"
			if len(roles) > 0 {
				actorRole = roles[0]
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO audit_events (
					event_id, occurred_at, actor_subject_id, actor_role,
					organization_id, action, entity_type, entity_id,
					entity_version, after_status, request_id, details
				) VALUES (
					$1, $2, $3, $4, $5, 'USER_ACTIVATION_RECONCILED',
					'DESIRED_MEMBERSHIP', $6, $7, 'ACTIVE', $8,
					jsonb_build_object(
						'expectedMembershipRevision', $9::bigint,
						'providerMfaEnrolled', $10::boolean,
						'requestedBySubjectId', $11::text
					)
				)
			`, service.idGenerator("audit-user-activation"), now, subjectID,
				actorRole, organizationID, membershipID, nextRevision,
				requestID, expectedMembershipRevision, providerMFAEnrolled,
				requestedBy); err != nil {
				return fmt.Errorf("append activation audit event: %w", err)
			}
			return nil
		},
	)
	if err != nil {
		return err
	}
	return outcome
}

func validateMembershipExpectation(
	ctx context.Context,
	transaction pgx.Tx,
	command RequestUserLifecycleCommand,
) (string, error) {
	var organizationExists bool
	if err := transaction.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM organizations
			WHERE id = $1
			  AND tombstoned_at IS NULL
			  AND status = 'ACTIVE'
		)
	`, command.OrganizationID).Scan(&organizationExists); err != nil {
		return "", err
	}
	if !organizationExists {
		return "", ErrInvalid
	}
	if command.Action == UserLifecycleProvision {
		if command.ExpectedMembershipRevision != 0 {
			return "", ErrMembershipRevisionConflict
		}
		return "", nil
	}
	if _, err := transaction.Exec(
		ctx,
		"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
		"desired-membership:"+command.SubjectID,
	); err != nil {
		return "", err
	}
	var membershipID, state, organizationID string
	var revision int64
	var roles []string
	err := transaction.QueryRow(ctx, `
		SELECT membership_id, revision, membership_state, organization_id, roles
		FROM desired_membership_versions
		WHERE subject_id = $1
		ORDER BY revision DESC
		LIMIT 1
	`, command.SubjectID).Scan(
		&membershipID,
		&revision,
		&state,
		&organizationID,
		&roles,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrMembershipRevisionConflict
	}
	if err != nil {
		return "", err
	}
	if revision != command.ExpectedMembershipRevision {
		return "", ErrMembershipRevisionConflict
	}
	var conflictingRequest bool
	if err := transaction.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM user_lifecycle_requests
			WHERE membership_id = $1
			  AND expected_membership_revision = $2
			  AND status IN ('PENDING', 'RUNNING', 'FAILED_RETRYABLE')
		)
	`, membershipID, revision).Scan(&conflictingRequest); err != nil {
		return "", err
	}
	if conflictingRequest {
		return "", ErrMembershipRevisionConflict
	}
	commandRoles := make([]string, len(command.Roles))
	for index, role := range command.Roles {
		commandRoles[index] = string(role)
	}
	if command.Action != UserLifecycleTransferOrganization &&
		command.OrganizationID != organizationID {
		return "", ErrConflict
	}
	if command.Action != UserLifecycleUpdateRoles &&
		strings.Join(commandRoles, "\x00") != strings.Join(roles, "\x00") {
		return "", ErrConflict
	}
	switch command.Action {
	case UserLifecycleResendInvitation:
		if state != "INVITED" {
			return "", ErrConflict
		}
	case UserLifecycleReactivate:
		if state != "SUSPENDED" && state != "DEACTIVATED" {
			return "", ErrConflict
		}
	case UserLifecycleDeactivate:
		if state == "DEACTIVATED" {
			return "", ErrConflict
		}
	}
	return membershipID, nil
}

func validateLifecycleCommand(command RequestUserLifecycleCommand) error {
	if command.OperationID == "" || command.IdempotencyKey == "" ||
		command.OrganizationID == "" || strings.TrimSpace(command.Reason) == "" ||
		command.ExpectedMembershipRevision < 0 {
		return ErrInvalid
	}
	switch command.Action {
	case UserLifecycleProvision, UserLifecycleUpdateRoles, UserLifecycleSuspend,
		UserLifecycleReactivate, UserLifecycleDeactivate,
		UserLifecycleTransferOrganization, UserLifecycleResendInvitation,
		UserLifecycleResetPassword, UserLifecycleResetMFA,
		UserLifecycleForceLogout:
	default:
		return ErrInvalid
	}
	if command.Action == UserLifecycleProvision {
		if command.SubjectID != "" || command.Email == "" ||
			command.DisplayName == "" ||
			command.ExpectedMembershipRevision != 0 {
			return ErrInvalid
		}
	} else if command.SubjectID == "" || command.Email != "" ||
		command.DisplayName != "" ||
		command.ExpectedMembershipRevision == 0 {
		return ErrInvalid
	}
	if command.Action == UserLifecycleTransferOrganization {
		if command.EffectiveAt == nil {
			return ErrInvalid
		}
	} else if command.EffectiveAt != nil {
		return ErrInvalid
	}
	if len(command.Roles) != 1 {
		return ErrInvalid
	}
	if err := identity.ValidateApplicationAuthority(
		command.OrganizationID,
		command.Roles,
	); err != nil {
		return ErrInvalid
	}
	return nil
}

func normalizeLifecycleCommand(command RequestUserLifecycleCommand) RequestUserLifecycleCommand {
	command.OperationID = strings.TrimSpace(command.OperationID)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.SubjectID = strings.TrimSpace(command.SubjectID)
	command.OrganizationID = strings.TrimSpace(command.OrganizationID)
	command.Email = strings.ToLower(strings.TrimSpace(command.Email))
	command.DisplayName = strings.TrimSpace(command.DisplayName)
	command.Reason = strings.TrimSpace(command.Reason)
	if command.EffectiveAt != nil {
		effectiveAt := command.EffectiveAt.UTC()
		command.EffectiveAt = &effectiveAt
	}
	return command
}

func lifecycleAvailableAt(
	command RequestUserLifecycleCommand,
	now time.Time,
) time.Time {
	if command.Action == UserLifecycleTransferOrganization &&
		command.EffectiveAt != nil {
		return command.EffectiveAt.UTC()
	}
	return now
}

func lifecycleAuditAction(action UserLifecycleAction) string {
	switch action {
	case UserLifecycleProvision:
		return "USER_PROVISION_REQUESTED"
	case UserLifecycleUpdateRoles:
		return "USER_ROLE_UPDATE_REQUESTED"
	case UserLifecycleSuspend:
		return "USER_SUSPENSION_REQUESTED"
	case UserLifecycleReactivate:
		return "USER_REACTIVATION_REQUESTED"
	case UserLifecycleDeactivate:
		return "USER_DEACTIVATION_REQUESTED"
	case UserLifecycleTransferOrganization:
		return "USER_ORGANIZATION_TRANSFER_REQUESTED"
	case UserLifecycleResendInvitation:
		return "USER_INVITATION_RESEND_REQUESTED"
	case UserLifecycleResetPassword:
		return "USER_PASSWORD_RESET_REQUESTED"
	case UserLifecycleResetMFA:
		return "USER_MFA_RESET_REQUESTED"
	case UserLifecycleForceLogout:
		return "USER_FORCE_LOGOUT_REQUESTED"
	default:
		return "USER_LIFECYCLE_REQUESTED"
	}
}

func randomUserID(prefix string) string {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic(fmt.Sprintf("secure user lifecycle identifier generation failed: %v", err))
	}
	return prefix + "-" + hex.EncodeToString(bytes[:])
}
