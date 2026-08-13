package administration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/notifications"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/platform/telemetry"
	"github.com/jackc/pgx/v5"
)

type UserLifecycleIdentityProvider interface {
	ProvisionUser(context.Context, identity.ProviderUser) (string, error)
	ReconcileProvisionedUser(context.Context, identity.ProviderUser) (string, bool, error)
	UpdateUserAuthority(context.Context, string, string, []identity.Role) error
	DisableUser(context.Context, string) error
	EnableUser(context.Context, string) error
	IssueExecuteActionsEmail(context.Context, string, []string, int) error
	ResetUserMFA(context.Context, string) error
	ForceUserLogout(context.Context, string) error
}

type UserLifecycleWorkerDependencies struct {
	Clock          func() time.Time
	IDGenerator    func(string) string
	WorkerID       string
	LeaseDuration  time.Duration
	RetryDelay     time.Duration
	MaxAttempts    int
	MaxRetryWindow time.Duration
	Issuer         string
}

type UserLifecycleWorker struct {
	pool           *database.Pool
	provider       UserLifecycleIdentityProvider
	clock          func() time.Time
	idGenerator    func(string) string
	workerID       string
	leaseDuration  time.Duration
	retryDelay     time.Duration
	maxAttempts    int
	maxRetryWindow time.Duration
	issuer         string
}

type claimedUserLifecycle struct {
	RequestID                  string
	SubjectID                  string
	Action                     UserLifecycleAction
	Roles                      []identity.Role
	OrganizationID             string
	Email                      string
	DisplayName                string
	RequestedBy                string
	OutboxID                   string
	OperationID                string
	CorrelationID              string
	TraceParent                string
	AvailableAt                time.Time
	ExpectedMembershipRevision int64
	MembershipID               string
	Reason                     string
	EffectiveAt                *time.Time
	Attempt                    int
}

type providerActionResult struct {
	SubjectID              string
	ActionKind             notifications.IdentityActionKind
	ProviderAcknowledgedAt time.Time
	InvitationExpiresAt    time.Time
}

func NewUserLifecycleWorker(
	pool *database.Pool,
	provider UserLifecycleIdentityProvider,
	dependencies UserLifecycleWorkerDependencies,
) *UserLifecycleWorker {
	clock := dependencies.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := dependencies.IDGenerator
	if idGenerator == nil {
		idGenerator = randomUserID
	}
	workerID := strings.TrimSpace(dependencies.WorkerID)
	if workerID == "" {
		workerID = "identity-lifecycle-worker"
	}
	leaseDuration := dependencies.LeaseDuration
	if leaseDuration <= 0 {
		leaseDuration = time.Minute
	}
	retryDelay := dependencies.RetryDelay
	if retryDelay <= 0 {
		retryDelay = 5 * time.Second
	}
	maxAttempts := dependencies.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	maxRetryWindow := dependencies.MaxRetryWindow
	if maxRetryWindow <= 0 {
		maxRetryWindow = 30 * time.Minute
	}
	return &UserLifecycleWorker{
		pool: pool, provider: provider, clock: clock,
		idGenerator: idGenerator, workerID: workerID,
		leaseDuration: leaseDuration, retryDelay: retryDelay,
		maxAttempts: maxAttempts, maxRetryWindow: maxRetryWindow,
		issuer: strings.TrimSpace(dependencies.Issuer),
	}
}

func (worker *UserLifecycleWorker) ProcessNext(
	ctx context.Context,
) (processed bool, resultErr error) {
	if expired, err := worker.expireNextIdentityAction(ctx); err != nil || expired {
		return expired, err
	}
	claimed, ok, err := worker.claimNext(ctx)
	if err != nil || !ok {
		return ok, err
	}
	jobContext, span := telemetry.StartPersistedJob(
		ctx,
		claimed.TraceParent,
		claimed.CorrelationID,
		"identity",
		"identity-provider",
	)
	telemetry.RecordPersistedOutboxReadyAge(
		jobContext,
		"identity",
		"identity",
		claimed.AvailableAt,
		worker.clock().UTC(),
	)
	defer func() {
		telemetry.FinishPersistedJob(
			jobContext,
			span,
			"identity",
			"identity-provider",
			resultErr,
		)
	}()
	if err := worker.validateClaimedMembershipExpectation(
		jobContext,
		claimed,
	); err != nil {
		if !errors.Is(err, ErrMembershipRevisionConflict) {
			return true, err
		}
		if recordErr := worker.recordFailure(
			jobContext,
			claimed,
			providerActionResult{SubjectID: claimed.SubjectID},
			err,
		); recordErr != nil {
			return true, errors.Join(err, recordErr)
		}
		return true, err
	}
	if claimed.Action == UserLifecycleResetPassword ||
		claimed.Action == UserLifecycleResetMFA {
		if err := worker.invalidateRecoverySessions(
			jobContext,
			claimed,
		); err != nil {
			return true, err
		}
	}
	providerResult, err := worker.applyProviderAction(jobContext, claimed)
	if err != nil {
		if recordErr := worker.recordFailure(
			jobContext,
			claimed,
			providerResult,
			err,
		); recordErr != nil {
			return true, errors.Join(err, recordErr)
		}
		return true, err
	}
	if err := worker.finalizeSuccess(
		jobContext,
		claimed,
		providerResult,
	); err != nil {
		return true, err
	}
	return true, nil
}

func (worker *UserLifecycleWorker) invalidateRecoverySessions(
	ctx context.Context,
	claimed claimedUserLifecycle,
) error {
	now := worker.clock().UTC()
	return database.WithinTransaction(
		ctx,
		worker.pool,
		func(ctx context.Context, transaction pgx.Tx) error {
			result, err := transaction.Exec(ctx, `
				UPDATE session_references
				SET revoked_at = COALESCE(revoked_at, $2),
				    provider_tokens_ciphertext = NULL
				WHERE subject_id = $1
				  AND revoked_at IS NULL
			`, claimed.SubjectID, now)
			if err != nil {
				return fmt.Errorf("invalidate recovery sessions: %w", err)
			}
			if result.RowsAffected() == 0 {
				return nil
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO audit_events (
					event_id, occurred_at, actor_subject_id, actor_role,
					organization_id, action, entity_type, entity_id,
					entity_version, after_status, operation_id,
					correlation_id, request_id, details
				) VALUES (
					$1, $2, $3, 'admin', NULLIF($4, ''), $5,
					'USER_LIFECYCLE_REQUEST', $6, 2,
					'SESSIONS_REVOKED', NULLIF($7, ''),
					NULLIF($8, ''), $6,
					jsonb_build_object('reason', $9::text)
				)
			`, worker.idGenerator("audit-recovery-session-invalidation"), now,
				claimed.RequestedBy, claimed.OrganizationID,
				"USER_"+string(claimed.Action)+"_SESSIONS_REVOKED",
				claimed.RequestID, claimed.OperationID,
				claimed.CorrelationID, claimed.Reason); err != nil {
				return fmt.Errorf("append recovery session audit: %w", err)
			}
			return nil
		},
	)
}

func (worker *UserLifecycleWorker) expireNextIdentityAction(
	ctx context.Context,
) (bool, error) {
	now := worker.clock().UTC()
	expired := false
	err := database.WithinTransaction(
		ctx,
		worker.pool,
		func(ctx context.Context, transaction pgx.Tx) error {
			var requestID, membershipID, subjectID, actionKind, state, reason string
			var factSequence, deliveryAttempt int
			var expiresAt time.Time
			err := transaction.QueryRow(ctx, `
				SELECT fact.request_id, COALESCE(fact.membership_id, ''),
				       COALESCE(fact.subject_id, ''), fact.fact_sequence,
				       fact.action_kind, fact.state, fact.delivery_attempt,
				       fact.expires_at, fact.reason
				FROM identity_action_facts fact
				WHERE fact.expires_at IS NOT NULL
				  AND fact.expires_at <= $1
				  AND fact.state IN ('ISSUED', 'DELIVERY_ACCEPTED')
				  AND NOT EXISTS (
				      SELECT 1
				      FROM identity_action_facts terminal
				      WHERE terminal.request_id = fact.request_id
				        AND terminal.fact_sequence > fact.fact_sequence
				        AND terminal.state IN (
				            'CONSUMED',
				            'CANCELLED',
				            'EXPIRED'
				        )
				  )
				ORDER BY fact.expires_at, fact.request_id, fact.fact_sequence DESC
				LIMIT 1
			`, now).Scan(
				&requestID,
				&membershipID,
				&subjectID,
				&factSequence,
				&actionKind,
				&state,
				&deliveryAttempt,
				&expiresAt,
				&reason,
			)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			if _, err := transaction.Exec(
				ctx,
				"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
				"identity-action-expiry:"+requestID,
			); err != nil {
				return err
			}
			var nextSequence int
			var alreadyTerminal bool
			if err := transaction.QueryRow(ctx, `
				SELECT COALESCE(MAX(fact_sequence), 0) + 1,
				       COALESCE(bool_or(
				           state IN ('CONSUMED', 'CANCELLED', 'EXPIRED')
				       ), false)
				FROM identity_action_facts
				WHERE request_id = $1
			`, requestID).Scan(&nextSequence, &alreadyTerminal); err != nil {
				return err
			}
			if alreadyTerminal {
				return nil
			}
			if err := notifications.ValidateIdentityDeliveryTransition(
				notifications.IdentityDeliveryState(state),
				notifications.IdentityDeliveryExpired,
			); err != nil {
				return err
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO identity_action_facts (
					id, request_id, fact_sequence, membership_id, subject_id,
					action_kind, state, delivery_attempt, expires_at,
					reason, created_at
				) VALUES (
					$1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6,
					'EXPIRED', $7, $8, $9, $10
				)
			`, worker.idGenerator("identity-action-fact"), requestID,
				nextSequence, membershipID, subjectID, actionKind,
				deliveryAttempt, expiresAt, reason, now); err != nil {
				return fmt.Errorf("append expired identity action fact: %w", err)
			}
			var requestedBy, organizationID string
			if err := transaction.QueryRow(ctx, `
				SELECT requested_by_subject_id,
				       COALESCE(requested_organization_id, '')
				FROM user_lifecycle_requests
				WHERE id = $1
			`, requestID).Scan(&requestedBy, &organizationID); err != nil {
				return err
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO audit_events (
					event_id, occurred_at, actor_subject_id, actor_role,
					organization_id, action, entity_type, entity_id,
					entity_version, after_status, request_id, details
				) VALUES (
					$1, $2, $3, 'admin', NULLIF($4, ''),
					'IDENTITY_ACTION_EXPIRED', 'USER_LIFECYCLE_REQUEST',
					$5, 2, 'EXPIRED', $5,
					jsonb_build_object(
						'actionKind', $6::text,
						'membershipId', NULLIF($7::text, '')
					)
				)
			`, worker.idGenerator("audit-identity-action-expired"), now,
				requestedBy, organizationID, requestID, actionKind,
				membershipID); err != nil {
				return fmt.Errorf("append identity action expiry audit: %w", err)
			}
			expired = true
			return nil
		},
	)
	return expired, err
}

func (worker *UserLifecycleWorker) claimNext(
	ctx context.Context,
) (claimedUserLifecycle, bool, error) {
	var claimed claimedUserLifecycle
	var rawAction string
	var rawRoles []string
	var subjectID, organizationID, email, displayName *string
	var operationID, correlationID *string
	var effectiveAt *time.Time
	found := false
	now := worker.clock().UTC()
	err := database.WithinTransaction(
		ctx,
		worker.pool,
		func(ctx context.Context, transaction pgx.Tx) error {
			err := transaction.QueryRow(ctx, `
				SELECT request.id,
				       request.subject_id,
				       request.requested_action,
				       request.requested_roles,
				       request.requested_organization_id,
				       request.requested_email,
				       request.requested_display_name,
				       request.requested_by_subject_id,
				       request.expected_membership_revision,
				       COALESCE(request.membership_id, ''),
				       request.reason,
				       request.requested_effective_at,
				       outbox.id,
				       outbox.attempt_count,
				       outbox.operation_id,
				       outbox.correlation_id,
				       COALESCE(outbox.traceparent, ''),
				       outbox.available_at
				FROM user_lifecycle_requests request
				JOIN outbox_messages outbox
				  ON outbox.id = request.outbox_message_id
				WHERE outbox.topic = 'identity.user-lifecycle.requested'
				  AND request.status IN ('PENDING', 'FAILED', 'FAILED_RETRYABLE')
				  AND outbox.delivered_at IS NULL
				  AND outbox.terminal_state IS NULL
				  AND outbox.available_at <= $1
				  AND (
				      outbox.lease_expires_at IS NULL
				      OR outbox.lease_expires_at <= $1
				  )
				ORDER BY outbox.available_at, outbox.created_at, outbox.id
				FOR UPDATE OF request, outbox SKIP LOCKED
				LIMIT 1
			`, now).Scan(
				&claimed.RequestID,
				&subjectID,
				&rawAction,
				&rawRoles,
				&organizationID,
				&email,
				&displayName,
				&claimed.RequestedBy,
				&claimed.ExpectedMembershipRevision,
				&claimed.MembershipID,
				&claimed.Reason,
				&effectiveAt,
				&claimed.OutboxID,
				&claimed.Attempt,
				&operationID,
				&correlationID,
				&claimed.TraceParent,
				&claimed.AvailableAt,
			)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			found = true
			if subjectID != nil {
				claimed.SubjectID = *subjectID
			}
			if organizationID != nil {
				claimed.OrganizationID = *organizationID
			}
			if email != nil {
				claimed.Email = *email
			}
			if displayName != nil {
				claimed.DisplayName = *displayName
			}
			if operationID != nil {
				claimed.OperationID = *operationID
			}
			if correlationID != nil {
				claimed.CorrelationID = *correlationID
			}
			claimed.EffectiveAt = effectiveAt
			claimed.Action = UserLifecycleAction(rawAction)
			claimed.Attempt++
			claimed.Roles = make([]identity.Role, len(rawRoles))
			for index, role := range rawRoles {
				claimed.Roles[index] = identity.Role(role)
			}
			if _, err := transaction.Exec(ctx, `
				UPDATE user_lifecycle_requests
				SET status = 'RUNNING',
				    failure_reason = NULL,
				    updated_at = $2
				WHERE id = $1
			`, claimed.RequestID, now); err != nil {
				return err
			}
			result, err := transaction.Exec(ctx, `
				UPDATE outbox_messages
				SET claimed_at = $2,
				    lease_owner = $3,
				    lease_expires_at = $4,
				    attempt_count = attempt_count + 1,
				    last_error = NULL
				WHERE id = $1
				  AND delivered_at IS NULL
				  AND terminal_state IS NULL
			`, claimed.OutboxID, now, worker.workerID, now.Add(worker.leaseDuration))
			if err != nil {
				return err
			}
			if result.RowsAffected() != 1 {
				return errors.New("user lifecycle outbox claim changed")
			}
			return nil
		},
	)
	return claimed, found, err
}

func (worker *UserLifecycleWorker) validateClaimedMembershipExpectation(
	ctx context.Context,
	claimed claimedUserLifecycle,
) error {
	if claimed.Action == UserLifecycleProvision {
		if claimed.ExpectedMembershipRevision != 0 || claimed.MembershipID != "" {
			return ErrMembershipRevisionConflict
		}
		return nil
	}
	var membershipID string
	var revision int64
	if err := worker.pool.QueryRow(ctx, `
		SELECT membership_id, revision
		FROM desired_membership_versions
		WHERE subject_id = $1
		ORDER BY revision DESC
		LIMIT 1
	`, claimed.SubjectID).Scan(&membershipID, &revision); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMembershipRevisionConflict
		}
		return err
	}
	if membershipID != claimed.MembershipID ||
		revision != claimed.ExpectedMembershipRevision {
		return ErrMembershipRevisionConflict
	}
	return nil
}

func (worker *UserLifecycleWorker) applyProviderAction(
	ctx context.Context,
	claimed claimedUserLifecycle,
) (providerActionResult, error) {
	result := providerActionResult{SubjectID: claimed.SubjectID}
	switch claimed.Action {
	case UserLifecycleProvision:
		firstName, lastName := splitDisplayName(claimed.DisplayName)
		membershipID := "membership-" + claimed.RequestID
		user := identity.ProviderUser{
			Email: claimed.Email, FirstName: firstName, LastName: lastName,
			MembershipID:   membershipID,
			OrganizationID: claimed.OrganizationID,
			Roles:          append([]identity.Role(nil), claimed.Roles...),
		}
		var subjectID string
		var err error
		if revisioned, ok := worker.provider.(identity.RevisionedProviderAdmin); ok {
			subjectID, err = revisioned.ProvisionUserAtRevision(ctx, user, 0, 1)
		} else {
			subjectID, err = worker.provider.ProvisionUser(ctx, user)
		}
		result.SubjectID = subjectID
		if err == nil {
			if err := worker.requireUnboundProviderSubject(
				ctx,
				subjectID,
			); err != nil {
				return result, err
			}
			return worker.issueIdentityActions(
				ctx,
				result,
				notifications.IdentityActionInvitation,
				[]string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
				24*time.Hour,
			)
		}
		if !errors.Is(err, identity.ErrProviderDuplicateEmail) {
			return result, err
		}
		reconciledSubjectID, matched, reconcileErr :=
			worker.provider.ReconcileProvisionedUser(ctx, user)
		if reconcileErr != nil {
			return result, reconcileErr
		}
		if !matched {
			return result, err
		}
		result.SubjectID = reconciledSubjectID
		if err := worker.requireUnboundProviderSubject(
			ctx,
			reconciledSubjectID,
		); err != nil {
			return result, err
		}
		return worker.issueIdentityActions(
			ctx,
			result,
			notifications.IdentityActionInvitation,
			[]string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
			24*time.Hour,
		)
	case UserLifecycleUpdateRoles, UserLifecycleTransferOrganization:
		var err error
		if revisioned, ok := worker.provider.(identity.RevisionedProviderAdmin); ok {
			err = revisioned.UpdateUserAuthorityAtRevision(ctx, claimed.SubjectID, claimed.OrganizationID, claimed.Roles, claimed.MembershipID, claimed.ExpectedMembershipRevision, claimed.ExpectedMembershipRevision+1)
		} else {
			err = worker.provider.UpdateUserAuthority(ctx, claimed.SubjectID, claimed.OrganizationID, claimed.Roles)
		}
		if err != nil {
			return result, err
		}
		if err := worker.provider.ForceUserLogout(
			ctx,
			claimed.SubjectID,
		); err != nil {
			return result, err
		}
		return result, nil
	case UserLifecycleSuspend, UserLifecycleDeactivate:
		if revisioned, ok := worker.provider.(identity.RevisionedProviderAdmin); ok {
			state := "SUSPENDED"
			if claimed.Action == UserLifecycleDeactivate {
				state = "DEACTIVATED"
			}
			return result, revisioned.SetUserStateAtRevision(ctx, claimed.SubjectID, state, claimed.ExpectedMembershipRevision, claimed.ExpectedMembershipRevision+1)
		}
		return result, worker.provider.DisableUser(ctx, claimed.SubjectID)
	case UserLifecycleReactivate:
		var err error
		if revisioned, ok := worker.provider.(identity.RevisionedProviderAdmin); ok {
			err = revisioned.SetUserStateAtRevision(ctx, claimed.SubjectID, "INVITED", claimed.ExpectedMembershipRevision, claimed.ExpectedMembershipRevision+1)
		} else {
			err = worker.provider.EnableUser(ctx, claimed.SubjectID)
		}
		if err != nil {
			return result, err
		}
		return worker.issueIdentityActions(
			ctx,
			result,
			notifications.IdentityActionInvitation,
			[]string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
			24*time.Hour,
		)
	case UserLifecycleResendInvitation:
		var resendCount int
		if err := worker.pool.QueryRow(ctx, `
			SELECT count(*)
			FROM user_lifecycle_requests
			WHERE membership_id = $1
			  AND requested_action = 'RESEND_INVITATION'
			  AND created_at >= $2
		`, claimed.MembershipID, worker.clock().UTC().Add(-24*time.Hour)).Scan(
			&resendCount,
		); err != nil {
			return result, err
		}
		if resendCount > 3 {
			return result, ErrInvitationResendLimit
		}
		return worker.issueIdentityActions(
			ctx,
			result,
			notifications.IdentityActionInvitation,
			[]string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
			24*time.Hour,
		)
	case UserLifecycleResetPassword:
		result.ActionKind = notifications.IdentityActionRecovery
		if err := worker.provider.ForceUserLogout(
			ctx,
			claimed.SubjectID,
		); err != nil {
			return result, err
		}
		return worker.issueIdentityActions(
			ctx,
			result,
			notifications.IdentityActionRecovery,
			[]string{"UPDATE_PASSWORD"},
			15*time.Minute,
		)
	case UserLifecycleResetMFA:
		result.ActionKind = notifications.IdentityActionMFAReset
		if err := worker.provider.ForceUserLogout(ctx, claimed.SubjectID); err != nil {
			return result, err
		}
		if err := worker.provider.ResetUserMFA(ctx, claimed.SubjectID); err != nil {
			return result, err
		}
		result.ProviderAcknowledgedAt = worker.clock().UTC()
		return result, nil
	case UserLifecycleForceLogout:
		if err := worker.provider.ForceUserLogout(
			ctx,
			claimed.SubjectID,
		); err != nil {
			return result, err
		}
		result.ProviderAcknowledgedAt = worker.clock().UTC()
		return result, nil
	default:
		return result, fmt.Errorf(
			"unsupported user lifecycle action %q",
			claimed.Action,
		)
	}
}

func (worker *UserLifecycleWorker) requireUnboundProviderSubject(
	ctx context.Context,
	subjectID string,
) error {
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" {
		return fmt.Errorf(
			"identity provider omitted subject ID: %w",
			identity.ErrProviderManualReview,
		)
	}
	var retained bool
	if err := worker.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM identity_references
			WHERE subject_id = $1
		)
	`, subjectID).Scan(&retained); err != nil {
		return fmt.Errorf("check retained provider subject: %w", err)
	}
	if retained {
		return identity.ErrProviderDuplicateSubject
	}
	return nil
}

func (worker *UserLifecycleWorker) issueIdentityActions(
	ctx context.Context,
	result providerActionResult,
	actionKind notifications.IdentityActionKind,
	actions []string,
	lifespan time.Duration,
) (providerActionResult, error) {
	result.ActionKind = actionKind
	result.InvitationExpiresAt = worker.clock().UTC().Add(lifespan)
	if err := worker.provider.IssueExecuteActionsEmail(
		ctx,
		result.SubjectID,
		actions,
		int(lifespan/time.Second),
	); err != nil {
		return result, err
	}
	result.ProviderAcknowledgedAt = worker.clock().UTC()
	return result, nil
}

func (worker *UserLifecycleWorker) finalizeSuccess(
	ctx context.Context,
	claimed claimedUserLifecycle,
	providerResult providerActionResult,
) error {
	subjectID := providerResult.SubjectID
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" {
		return errors.New("identity provider omitted subject ID")
	}
	now := worker.clock().UTC()
	providerAcknowledgedAt := providerResult.ProviderAcknowledgedAt
	if providerAcknowledgedAt.IsZero() {
		providerAcknowledgedAt = now
	}
	return database.WithinTransaction(
		ctx,
		worker.pool,
		func(ctx context.Context, transaction pgx.Tx) error {
			var leaseOwner string
			if err := transaction.QueryRow(ctx, `
				SELECT COALESCE(lease_owner, '')
				FROM outbox_messages
				WHERE id = $1
				  AND delivered_at IS NULL
				FOR UPDATE
			`, claimed.OutboxID).Scan(&leaseOwner); err != nil {
				return err
			}
			if leaseOwner != worker.workerID {
				return errors.New("user lifecycle outbox lease changed")
			}
			if claimed.Action == UserLifecycleProvision {
				if worker.issuer == "" {
					return errors.New("identity provider issuer is required")
				}
				if _, err := transaction.Exec(ctx, `
					INSERT INTO identity_references (
						subject_id, issuer, display_name, email
					) VALUES ($1, $2, $3, lower($4))
					ON CONFLICT (subject_id) DO UPDATE
					SET email = EXCLUDED.email
					WHERE identity_references.issuer = EXCLUDED.issuer
					  AND identity_references.display_name = EXCLUDED.display_name
					  AND identity_references.tombstoned_at IS NULL
				`, subjectID, worker.issuer,
					claimed.DisplayName, claimed.Email); err != nil {
					return fmt.Errorf("persist provider identity reference: %w", err)
				}
				var storedIssuer, storedDisplayName, storedEmail string
				if err := transaction.QueryRow(ctx, `
					SELECT issuer, display_name, COALESCE(email, '')
					FROM identity_references
					WHERE subject_id = $1
				`, subjectID).Scan(
					&storedIssuer,
					&storedDisplayName,
					&storedEmail,
				); err != nil {
					return err
				}
				if storedIssuer != worker.issuer ||
					storedDisplayName != claimed.DisplayName ||
					storedEmail != strings.ToLower(claimed.Email) {
					return errors.New("provider subject conflicts with retained identity")
				}
				if _, err := transaction.Exec(ctx, `
					INSERT INTO user_profiles (
						subject_id, display_name, organization_id
					) VALUES ($1, $2, $3)
					ON CONFLICT (subject_id) DO NOTHING
				`, subjectID, claimed.DisplayName, claimed.OrganizationID); err != nil {
					return fmt.Errorf("persist provider user profile: %w", err)
				}
				var profileDisplayName string
				var profileOrganizationID *string
				if err := transaction.QueryRow(ctx, `
					SELECT display_name, organization_id
					FROM user_profiles
					WHERE subject_id = $1
				`, subjectID).Scan(
					&profileDisplayName,
					&profileOrganizationID,
				); err != nil {
					return err
				}
				if profileOrganizationID == nil ||
					*profileOrganizationID != claimed.OrganizationID ||
					profileDisplayName != claimed.DisplayName {
					return errors.New(
						"provider subject conflicts with retained profile",
					)
				}
				if _, err := transaction.Exec(ctx, `
					INSERT INTO user_settings (
						subject_id, notification_preferences, locale, timezone
					) VALUES ($1, '{}'::jsonb, 'en', 'UTC')
					ON CONFLICT (subject_id) DO NOTHING
				`, subjectID); err != nil {
					return fmt.Errorf("persist provider user settings: %w", err)
				}
			}
			if _, err := transaction.Exec(
				ctx,
				"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
				"desired-membership:"+subjectID,
			); err != nil {
				return fmt.Errorf("lock desired membership: %w", err)
			}
			membershipID := claimed.MembershipID
			membershipRevision := claimed.ExpectedMembershipRevision
			membershipState := ""
			currentOrganizationID := claimed.OrganizationID
			currentRoles := make([]string, len(claimed.Roles))
			for index, role := range claimed.Roles {
				currentRoles[index] = string(role)
			}
			if claimed.Action == UserLifecycleProvision {
				membershipID = "membership-" + claimed.RequestID
				membershipRevision = 1
				membershipState = "INVITED"
			} else {
				if err := transaction.QueryRow(ctx, `
					SELECT membership_state, organization_id, roles
					FROM desired_membership_versions
					WHERE membership_id = $1
					  AND revision = $2
					  AND subject_id = $3
				`, membershipID, claimed.ExpectedMembershipRevision, subjectID).Scan(
					&membershipState,
					&currentOrganizationID,
					&currentRoles,
				); err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						return ErrMembershipRevisionConflict
					}
					return err
				}
			}
			appendMembership := membershipAuthorityAction(claimed.Action)
			providerEnabled := membershipState != "SUSPENDED" &&
				membershipState != "DEACTIVATED"
			if appendMembership && claimed.Action != UserLifecycleProvision {
				membershipRevision++
				switch claimed.Action {
				case UserLifecycleSuspend:
					membershipState = "SUSPENDED"
					providerEnabled = false
				case UserLifecycleDeactivate:
					membershipState = "DEACTIVATED"
					providerEnabled = false
				case UserLifecycleReactivate:
					membershipState = "INVITED"
					providerEnabled = true
				}
				if claimed.Action == UserLifecycleUpdateRoles {
					currentRoles = make([]string, len(claimed.Roles))
					for index, role := range claimed.Roles {
						currentRoles[index] = string(role)
					}
				}
				if claimed.Action == UserLifecycleTransferOrganization {
					currentOrganizationID = claimed.OrganizationID
				}
			}
			if appendMembership {
				effectiveAt := now
				if claimed.Action == UserLifecycleTransferOrganization &&
					claimed.EffectiveAt != nil {
					effectiveAt = claimed.EffectiveAt.UTC()
				}
				if _, err := transaction.Exec(ctx, `
				INSERT INTO desired_membership_versions (
					membership_id, subject_id, revision, membership_state, organization_id,
					roles, requested_by_subject_id, reason, source_request_id,
					requested_at, effective_at, observed_provider_enabled,
					observed_organization_id, observed_roles, observed_at,
					drift_state
				) VALUES (
					$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
					$12, $5, $6, $11, 'IN_SYNC'
				)
			`, membershipID, subjectID, membershipRevision, membershipState,
					currentOrganizationID, currentRoles, claimed.RequestedBy,
					claimed.Reason, claimed.RequestID,
					claimed.AvailableAt, effectiveAt, providerEnabled); err != nil {
					return fmt.Errorf("append desired membership version: %w", err)
				}
				if _, err := transaction.Exec(ctx, `
				INSERT INTO desired_membership_sync (
					membership_id, subject_id, desired_revision,
					observed_provider_enabled, observed_organization_id,
					observed_roles, observed_at, drift_state
				) VALUES ($1, $2, $3, $4, $5, $6, $7, 'IN_SYNC')
				ON CONFLICT (membership_id) DO UPDATE
				SET desired_revision = EXCLUDED.desired_revision,
				    observed_provider_enabled = EXCLUDED.observed_provider_enabled,
				    observed_organization_id = EXCLUDED.observed_organization_id,
				    observed_roles = EXCLUDED.observed_roles,
				    observed_at = EXCLUDED.observed_at,
				    drift_state = EXCLUDED.drift_state
			`, membershipID, subjectID, membershipRevision, providerEnabled,
					currentOrganizationID, currentRoles, now); err != nil {
					return fmt.Errorf("synchronize desired membership observation: %w", err)
				}
			}
			if claimed.Action == UserLifecycleDeactivate {
				if _, err := transaction.Exec(ctx, `
					UPDATE identity_references
					SET deactivated_at = COALESCE(deactivated_at, $2)
					WHERE subject_id = $1
				`, subjectID, now); err != nil {
					return fmt.Errorf("retain deactivated identity tombstone: %w", err)
				}
			}
			if claimed.Action == UserLifecycleTransferOrganization {
				if _, err := transaction.Exec(ctx, `
					UPDATE user_profiles
					SET organization_id = $2,
					    revision = revision + 1,
					    updated_at = $3
					WHERE subject_id = $1
				`, subjectID, currentOrganizationID, now); err != nil {
					return fmt.Errorf("update transferred user profile: %w", err)
				}
			}
			if invalidatesSessions(claimed.Action) {
				if _, err := transaction.Exec(ctx, `
					UPDATE session_references
					SET revoked_at = COALESCE(revoked_at, $2),
					    provider_tokens_ciphertext = NULL
					WHERE subject_id = $1
					  AND revoked_at IS NULL
				`, subjectID, now); err != nil {
					return fmt.Errorf("invalidate target sessions: %w", err)
				}
			}
			if err := worker.appendIdentityActionFacts(
				ctx,
				transaction,
				claimed,
				providerResult,
				membershipID,
				subjectID,
				now,
			); err != nil {
				return err
			}

			statusPayload, err := json.Marshal(UserLifecycleRequest{
				ID: claimed.RequestID, SubjectID: subjectID,
				Action: claimed.Action, Roles: claimed.Roles,
				OrganizationID: claimed.OrganizationID,
				Email:          claimed.Email, DisplayName: claimed.DisplayName,
				Status: UserLifecycleSuccess, RequestedBy: claimed.RequestedBy,
				ExpectedMembershipRevision:  claimed.ExpectedMembershipRevision,
				ResultingMembershipRevision: membershipRevision,
				MembershipID:                membershipID, Reason: claimed.Reason,
				ProviderAcknowledgedAt: &providerAcknowledgedAt,
				AttemptCount:           claimed.Attempt,
				OutboxMessageID:        claimed.OutboxID, UpdatedAt: now,
			})
			if err != nil {
				return err
			}
			if _, err := transaction.Exec(ctx, `
				UPDATE user_lifecycle_requests
				SET subject_id = $2,
				    status = 'SUCCEEDED',
				    failure_reason = NULL,
				    provider_failure_class = NULL,
				    membership_id = $4,
				    resulting_membership_revision = $5,
				    provider_acknowledged_at = $6,
				    updated_at = $3
				WHERE id = $1
			`, claimed.RequestID, subjectID, now, membershipID,
				membershipRevision, providerAcknowledgedAt); err != nil {
				return err
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO audit_events (
					event_id, occurred_at, actor_subject_id, actor_role,
					organization_id, action, entity_type, entity_id,
					entity_version, after_status, operation_id, correlation_id,
					request_id, details
				) VALUES (
					$1, $2, $3, 'admin', NULLIF($4, ''), $5,
					'USER_LIFECYCLE_REQUEST', $6, 2, 'SUCCEEDED',
					NULLIF($7, ''), NULLIF($8, ''), $6,
					jsonb_build_object(
						'reason', $9::text,
						'membershipId', $10::text,
						'expectedMembershipRevision', $11::bigint,
						'resultingMembershipRevision', $12::bigint,
							'providerAcknowledgedAt', $13::timestamptz,
							'sessionInvalidated', $14::boolean,
							'identityActionKind', NULLIF($15::text, ''),
							'subjectId', $16::text,
							'roles', to_jsonb($17::text[]),
							'organizationId', $4::text
					)
				)
			`, worker.idGenerator("audit-user-lifecycle-completed"), now,
				claimed.RequestedBy, claimed.OrganizationID,
				lifecycleSuccessAuditAction(claimed.Action), claimed.RequestID,
				claimed.OperationID, claimed.CorrelationID, claimed.Reason,
				membershipID, claimed.ExpectedMembershipRevision,
				membershipRevision, providerAcknowledgedAt,
				invalidatesSessions(claimed.Action),
				string(providerResult.ActionKind), subjectID,
				currentRoles); err != nil {
				return fmt.Errorf("append user lifecycle success audit: %w", err)
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO authorized_sync_changes (
					subject_id, organization_id, kind, entity_id,
					entity_revision, payload, changed_at, operation_id,
					correlation_id
				) VALUES (
					$1, NULLIF($2, ''), 'USER_LIFECYCLE_REQUEST', $3, 2,
					$4, $5, NULLIF($6, ''), NULLIF($7, '')
				)
			`, claimed.RequestedBy, claimed.OrganizationID,
				claimed.RequestID, statusPayload, now,
				claimed.OperationID, claimed.CorrelationID); err != nil {
				return fmt.Errorf("append user lifecycle success change: %w", err)
			}
			result, err := transaction.Exec(ctx, `
				UPDATE outbox_messages
				SET delivered_at = $2,
				    lease_owner = NULL,
				    lease_expires_at = NULL,
				    last_error = NULL
				WHERE id = $1
				  AND lease_owner = $3
				  AND delivered_at IS NULL
			`, claimed.OutboxID, now, worker.workerID)
			if err != nil {
				return err
			}
			if result.RowsAffected() != 1 {
				return errors.New("user lifecycle outbox completion changed")
			}
			return nil
		},
	)
}

func (worker *UserLifecycleWorker) recordFailure(
	ctx context.Context,
	claimed claimedUserLifecycle,
	providerResult providerActionResult,
	providerErr error,
) error {
	now := worker.clock().UTC()
	failureClass := identity.ClassifyProviderError(providerErr)
	failureCode := identity.ProviderFailureReasonCode(providerErr)
	if errors.Is(providerErr, ErrMembershipRevisionConflict) {
		failureClass = identity.ProviderFailurePermanent
		failureCode = "STALE_MEMBERSHIP_REVISION"
	} else if errors.Is(providerErr, ErrInvitationResendLimit) {
		failureClass = identity.ProviderFailurePermanent
		failureCode = "INVITATION_RESEND_LIMIT"
	}
	return database.WithinTransaction(
		ctx,
		worker.pool,
		func(ctx context.Context, transaction pgx.Tx) error {
			var attemptCount int
			var createdAt time.Time
			if err := transaction.QueryRow(ctx, `
				SELECT attempt_count, created_at
				FROM outbox_messages
				WHERE id = $1
				  AND lease_owner = $2
				  AND delivered_at IS NULL
				FOR UPDATE
			`, claimed.OutboxID, worker.workerID).Scan(
				&attemptCount,
				&createdAt,
			); err != nil {
				return err
			}
			retryable := failureClass == identity.ProviderFailureRetryable &&
				attemptCount < worker.maxAttempts &&
				now.Sub(createdAt) < worker.maxRetryWindow
			status := UserLifecycleFailedPermanent
			terminalState := string(status)
			nextAvailableAt := now
			if retryable {
				status = UserLifecycleFailedRetryable
				terminalState = ""
				nextAvailableAt = now.Add(worker.retryBackoff(attemptCount))
			} else if failureClass == identity.ProviderFailureManualReview ||
				failureClass == identity.ProviderFailureRetryable {
				status = UserLifecycleManualReview
				terminalState = string(status)
				if failureClass == identity.ProviderFailureRetryable {
					failureClass = identity.ProviderFailureManualReview
					failureCode = "PROVIDER_RETRY_EXHAUSTED"
				}
			}
			result, err := transaction.Exec(ctx, `
				UPDATE user_lifecycle_requests
				SET status = $2,
				    failure_reason = $3,
				    provider_failure_class = $4,
				    updated_at = $5
				WHERE id = $1
			`, claimed.RequestID, string(status), failureCode,
				string(failureClass), now)
			if err != nil {
				return err
			}
			if result.RowsAffected() != 1 {
				return errors.New("user lifecycle failure state changed")
			}
			result, err = transaction.Exec(ctx, `
				UPDATE outbox_messages
				SET available_at = $2,
				    lease_owner = NULL,
				    lease_expires_at = NULL,
				    last_error = $3,
				    terminal_state = NULLIF($5, '')
				WHERE id = $1
				  AND lease_owner = $4
				  AND delivered_at IS NULL
			`, claimed.OutboxID, nextAvailableAt, failureCode, worker.workerID,
				terminalState)
			if err != nil {
				return err
			}
			if result.RowsAffected() != 1 {
				return errors.New("user lifecycle failure lease changed")
			}
			if providerResult.ActionKind != "" {
				state := notifications.IdentityDeliveryTerminalFailure
				if retryable {
					state = notifications.IdentityDeliveryRetryableFailure
				}
				if err := worker.appendIdentityFailureFact(
					ctx,
					transaction,
					claimed,
					providerResult,
					state,
					now,
				); err != nil {
					return err
				}
			}
			if !retryable {
				severity := "CRITICAL"
				if status == UserLifecycleFailedPermanent {
					severity = "WARNING"
				}
				if _, err := transaction.Exec(ctx, `
					INSERT INTO identity_lifecycle_alerts (
						id, request_id, severity, failure_class,
						reason_code, created_at
					) VALUES ($1, $2, $3, $4, $5, $6)
				`, worker.idGenerator("identity-lifecycle-alert"),
					claimed.RequestID, severity, string(failureClass),
					failureCode, now); err != nil {
					return fmt.Errorf("append identity lifecycle alert: %w", err)
				}
			}
			if _, err := transaction.Exec(ctx, `
				INSERT INTO audit_events (
					event_id, occurred_at, actor_subject_id, actor_role,
					organization_id, action, entity_type, entity_id,
					entity_version, after_status, operation_id, correlation_id,
					request_id, details
					) VALUES (
						$1, $2, $3, 'admin', NULLIF($4, ''),
						$5, 'USER_LIFECYCLE_REQUEST', $6, 2, $7,
						NULLIF($8, ''), NULLIF($9, ''), $6,
						jsonb_build_object(
							'failureClass', $10::text,
							'reasonCode', $11::text,
							'attempt', $12::integer,
							'retryScheduled', $13::boolean
						)
					)
				`, worker.idGenerator("audit-user-lifecycle-failed"), now,
				claimed.RequestedBy, claimed.OrganizationID,
				fmt.Sprintf(
					"USER_LIFECYCLE_PROVIDER_ATTEMPT_%d_FAILED",
					attemptCount,
				),
				claimed.RequestID, string(status), claimed.OperationID,
				claimed.CorrelationID, string(failureClass),
				failureCode, attemptCount, retryable); err != nil {
				return fmt.Errorf("append user lifecycle failure audit: %w", err)
			}
			return nil
		},
	)
}

func (worker *UserLifecycleWorker) retryBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := worker.retryDelay
	for index := 1; index < attempt && delay < 5*time.Minute; index++ {
		delay *= 2
	}
	if delay > 5*time.Minute {
		return 5 * time.Minute
	}
	return delay
}

func (worker *UserLifecycleWorker) appendIdentityActionFacts(
	ctx context.Context,
	transaction pgx.Tx,
	claimed claimedUserLifecycle,
	providerResult providerActionResult,
	membershipID,
	subjectID string,
	now time.Time,
) error {
	actionKind := providerResult.ActionKind
	states := []notifications.IdentityDeliveryState{}
	if claimed.Action == UserLifecycleResendInvitation {
		if err := worker.appendPendingIdentityActionCancellation(
			ctx,
			transaction,
			membershipID,
			subjectID,
			[]string{"INVITATION"},
			false,
			claimed.Reason,
			now,
		); err != nil {
			return err
		}
	} else if claimed.Action == UserLifecycleSuspend ||
		claimed.Action == UserLifecycleDeactivate {
		if err := worker.appendPendingIdentityActionCancellation(
			ctx,
			transaction,
			membershipID,
			subjectID,
			[]string{"INVITATION", "RECOVERY"},
			true,
			claimed.Reason,
			now,
		); err != nil {
			return err
		}
	}
	switch {
	case claimed.Action == UserLifecycleResendInvitation:
		actionKind = notifications.IdentityActionInvitation
		states = append(
			states,
			notifications.IdentityDeliveryIssued,
			notifications.IdentityDeliveryAccepted,
		)
	case actionKind == notifications.IdentityActionMFAReset:
		states = append(
			states,
			notifications.IdentityDeliveryResetPending,
			notifications.IdentityDeliveryResetCompleted,
		)
	case actionKind != "":
		states = append(
			states,
			notifications.IdentityDeliveryIssued,
			notifications.IdentityDeliveryAccepted,
		)
	}
	var priorSequence int
	if err := transaction.QueryRow(ctx, `
		SELECT COALESCE(MAX(fact_sequence), 0)
		FROM identity_action_facts
		WHERE request_id = $1
	`, claimed.RequestID).Scan(&priorSequence); err != nil {
		return err
	}
	for index, state := range states {
		if err := worker.validateIdentityActionTransition(
			ctx,
			transaction,
			claimed.RequestID,
			state,
		); err != nil {
			return err
		}
		var expiresAt any
		if !providerResult.InvitationExpiresAt.IsZero() &&
			(state == notifications.IdentityDeliveryIssued ||
				state == notifications.IdentityDeliveryAccepted) {
			expiresAt = providerResult.InvitationExpiresAt
		}
		var acknowledgedAt any
		if state == notifications.IdentityDeliveryAccepted ||
			state == notifications.IdentityDeliveryResetCompleted {
			acknowledgedAt = providerResult.ProviderAcknowledgedAt
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO identity_action_facts (
				id, request_id, fact_sequence, membership_id, subject_id,
				action_kind, state, delivery_attempt, expires_at,
				provider_acknowledged_at, reason, created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
			)
		`, worker.idGenerator("identity-action-fact"), claimed.RequestID,
			priorSequence+index+1, membershipID, subjectID,
			string(actionKind), string(state),
			claimed.Attempt, expiresAt, acknowledgedAt, claimed.Reason,
			now.Add(time.Duration(index)*time.Nanosecond)); err != nil {
			return fmt.Errorf("append identity action fact: %w", err)
		}
	}
	return nil
}

func (worker *UserLifecycleWorker) appendPendingIdentityActionCancellation(
	ctx context.Context,
	transaction pgx.Tx,
	membershipID string,
	subjectID string,
	actionKinds []string,
	cancelAll bool,
	reason string,
	now time.Time,
) error {
	type pendingAction struct {
		requestID       string
		actionKind      string
		state           notifications.IdentityDeliveryState
		deliveryAttempt int
		expiresAt       *time.Time
	}
	limit := 1
	if cancelAll {
		limit = 0
	}
	rows, err := transaction.Query(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (request_id)
			       request_id, action_kind, state, delivery_attempt,
			       expires_at, created_at, fact_sequence, id
			FROM identity_action_facts
			WHERE membership_id = $1
			  AND action_kind = ANY($2::text[])
			ORDER BY request_id, fact_sequence DESC
		)
		SELECT request_id, action_kind, state, delivery_attempt, expires_at
		FROM latest
		WHERE state IN ('ISSUED', 'DELIVERY_ACCEPTED')
		ORDER BY created_at DESC, fact_sequence DESC, id DESC
		LIMIT NULLIF($3, 0)
	`, membershipID, actionKinds, limit)
	if err != nil {
		return err
	}
	pending := []pendingAction{}
	for rows.Next() {
		var action pendingAction
		if err := rows.Scan(
			&action.requestID,
			&action.actionKind,
			&action.state,
			&action.deliveryAttempt,
			&action.expiresAt,
		); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, action)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for index, action := range pending {
		if err := notifications.ValidateIdentityDeliveryTransition(
			action.state,
			notifications.IdentityDeliveryCancelled,
		); err != nil {
			return err
		}
		var nextSequence int
		if err := transaction.QueryRow(ctx, `
			SELECT COALESCE(MAX(fact_sequence), 0) + 1
			FROM identity_action_facts
			WHERE request_id = $1
		`, action.requestID).Scan(&nextSequence); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO identity_action_facts (
				id, request_id, fact_sequence, membership_id, subject_id,
				action_kind, state, delivery_attempt, expires_at, reason,
				created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, 'CANCELLED', $7, $8, $9, $10
			)
		`, worker.idGenerator("identity-action-fact"), action.requestID,
			nextSequence, membershipID, subjectID, action.actionKind,
			action.deliveryAttempt, action.expiresAt, reason,
			now.Add(time.Duration(index)*time.Microsecond)); err != nil {
			return fmt.Errorf("append cancelled identity action fact: %w", err)
		}
	}
	return nil
}

func (worker *UserLifecycleWorker) appendIdentityFailureFact(
	ctx context.Context,
	transaction pgx.Tx,
	claimed claimedUserLifecycle,
	providerResult providerActionResult,
	state notifications.IdentityDeliveryState,
	now time.Time,
) error {
	if err := worker.validateIdentityActionTransition(
		ctx,
		transaction,
		claimed.RequestID,
		state,
	); err != nil {
		return err
	}
	var nextSequence int
	if err := transaction.QueryRow(ctx, `
		SELECT COALESCE(MAX(fact_sequence), 0) + 1
		FROM identity_action_facts
		WHERE request_id = $1
	`, claimed.RequestID).Scan(&nextSequence); err != nil {
		return err
	}
	var subjectID any
	if providerResult.SubjectID != "" {
		var exists bool
		if err := transaction.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM identity_references
				WHERE subject_id = $1
			)
		`, providerResult.SubjectID).Scan(&exists); err != nil {
			return err
		}
		if exists {
			subjectID = providerResult.SubjectID
		}
	}
	var membershipID any
	if claimed.MembershipID != "" {
		membershipID = claimed.MembershipID
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO identity_action_facts (
			id, request_id, fact_sequence, membership_id, subject_id,
			action_kind, state, delivery_attempt, expires_at,
			provider_acknowledged_at, reason, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, NULL, $10, $11
		)
	`, worker.idGenerator("identity-action-fact"), claimed.RequestID,
		nextSequence, membershipID, subjectID, string(providerResult.ActionKind),
		string(state), claimed.Attempt, nil, claimed.Reason, now); err != nil {
		return fmt.Errorf("append identity action failure fact: %w", err)
	}
	return nil
}

func (worker *UserLifecycleWorker) validateIdentityActionTransition(
	ctx context.Context,
	transaction pgx.Tx,
	requestID string,
	next notifications.IdentityDeliveryState,
) error {
	var previous string
	err := transaction.QueryRow(ctx, `
		SELECT state
		FROM identity_action_facts
		WHERE request_id = $1
		ORDER BY fact_sequence DESC
		LIMIT 1
	`, requestID).Scan(&previous)
	if errors.Is(err, pgx.ErrNoRows) {
		previous = ""
	} else if err != nil {
		return err
	}
	if err := notifications.ValidateIdentityDeliveryTransition(
		notifications.IdentityDeliveryState(previous),
		next,
	); err != nil {
		return fmt.Errorf("validate identity action fact: %w", err)
	}
	return nil
}

func membershipAuthorityAction(action UserLifecycleAction) bool {
	switch action {
	case UserLifecycleProvision, UserLifecycleUpdateRoles,
		UserLifecycleSuspend, UserLifecycleReactivate,
		UserLifecycleDeactivate, UserLifecycleTransferOrganization:
		return true
	default:
		return false
	}
}

func invalidatesSessions(action UserLifecycleAction) bool {
	switch action {
	case UserLifecycleUpdateRoles, UserLifecycleSuspend,
		UserLifecycleReactivate, UserLifecycleDeactivate,
		UserLifecycleTransferOrganization, UserLifecycleResetPassword,
		UserLifecycleResetMFA, UserLifecycleForceLogout:
		return true
	default:
		return false
	}
}

func splitDisplayName(displayName string) (string, string) {
	parts := strings.Fields(displayName)
	if len(parts) < 2 {
		return displayName, "."
	}
	return strings.Join(parts[:len(parts)-1], " "), parts[len(parts)-1]
}

func lifecycleSuccessAuditAction(action UserLifecycleAction) string {
	switch action {
	case UserLifecycleProvision:
		return "USER_PROVISION_SUCCEEDED"
	case UserLifecycleUpdateRoles:
		return "USER_ROLE_UPDATE_SUCCEEDED"
	case UserLifecycleSuspend:
		return "USER_SUSPENSION_SUCCEEDED"
	case UserLifecycleReactivate:
		return "USER_REACTIVATION_SUCCEEDED"
	case UserLifecycleDeactivate:
		return "USER_DEACTIVATION_SUCCEEDED"
	case UserLifecycleTransferOrganization:
		return "USER_ORGANIZATION_TRANSFER_SUCCEEDED"
	case UserLifecycleResendInvitation:
		return "USER_INVITATION_RESEND_SUCCEEDED"
	case UserLifecycleResetPassword:
		return "USER_PASSWORD_RESET_SUCCEEDED"
	case UserLifecycleResetMFA:
		return "USER_MFA_RESET_SUCCEEDED"
	case UserLifecycleForceLogout:
		return "USER_FORCE_LOGOUT_SUCCEEDED"
	default:
		return "USER_LIFECYCLE_SUCCEEDED"
	}
}
