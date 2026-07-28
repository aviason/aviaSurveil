package administration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/telemetry"
	"github.com/jackc/pgx/v5"
)

type UserLifecycleIdentityProvider interface {
	ProvisionUser(context.Context, identity.KeycloakUser) (string, error)
	ReconcileProvisionedUser(
		context.Context,
		identity.KeycloakUser,
	) (string, bool, error)
	UpdateUserAuthority(context.Context, string, string, []identity.Role) error
	DisableUser(context.Context, string) error
	EnableUser(context.Context, string) error
}

type UserLifecycleWorkerDependencies struct {
	Clock         func() time.Time
	IDGenerator   func(string) string
	WorkerID      string
	LeaseDuration time.Duration
	RetryDelay    time.Duration
	Issuer        string
}

type UserLifecycleWorker struct {
	pool          *database.Pool
	provider      UserLifecycleIdentityProvider
	clock         func() time.Time
	idGenerator   func(string) string
	workerID      string
	leaseDuration time.Duration
	retryDelay    time.Duration
	issuer        string
}

type claimedUserLifecycle struct {
	RequestID      string
	SubjectID      string
	Action         UserLifecycleAction
	Roles          []identity.Role
	OrganizationID string
	Email          string
	DisplayName    string
	RequestedBy    string
	OutboxID       string
	OperationID    string
	CorrelationID  string
	TraceParent    string
	AvailableAt    time.Time
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
	return &UserLifecycleWorker{
		pool: pool, provider: provider, clock: clock,
		idGenerator: idGenerator, workerID: workerID,
		leaseDuration: leaseDuration, retryDelay: retryDelay,
		issuer: strings.TrimSpace(dependencies.Issuer),
	}
}

func (worker *UserLifecycleWorker) ProcessNext(
	ctx context.Context,
) (processed bool, resultErr error) {
	claimed, ok, err := worker.claimNext(ctx)
	if err != nil || !ok {
		return ok, err
	}
	jobContext, span := telemetry.StartPersistedJob(
		ctx,
		claimed.TraceParent,
		claimed.CorrelationID,
		"identity",
		"keycloak",
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
			"keycloak",
			resultErr,
		)
	}()
	subjectID, err := worker.applyProviderAction(jobContext, claimed)
	if err != nil {
		if recordErr := worker.recordFailure(jobContext, claimed, err); recordErr != nil {
			return true, errors.Join(err, recordErr)
		}
		return true, err
	}
	if err := worker.finalizeSuccess(jobContext, claimed, subjectID); err != nil {
		return true, err
	}
	return true, nil
}

func (worker *UserLifecycleWorker) claimNext(
	ctx context.Context,
) (claimedUserLifecycle, bool, error) {
	var claimed claimedUserLifecycle
	var rawAction string
	var rawRoles []string
	var subjectID, organizationID, email, displayName *string
	var operationID, correlationID *string
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
				       outbox.id,
				       outbox.operation_id,
				       outbox.correlation_id,
				       COALESCE(outbox.traceparent, ''),
				       outbox.available_at
				FROM user_lifecycle_requests request
				JOIN outbox_messages outbox
				  ON outbox.id = request.outbox_message_id
				WHERE outbox.topic = 'identity.user-lifecycle.requested'
				  AND request.status IN ('PENDING', 'FAILED')
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
				&claimed.OutboxID,
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
			claimed.Action = UserLifecycleAction(rawAction)
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

func (worker *UserLifecycleWorker) applyProviderAction(
	ctx context.Context,
	claimed claimedUserLifecycle,
) (string, error) {
	switch claimed.Action {
	case UserLifecycleProvision:
		firstName, lastName := splitDisplayName(claimed.DisplayName)
		user := identity.KeycloakUser{
			Email: claimed.Email, FirstName: firstName, LastName: lastName,
			OrganizationID: claimed.OrganizationID,
			Roles:          append([]identity.Role(nil), claimed.Roles...),
		}
		subjectID, err := worker.provider.ProvisionUser(ctx, user)
		if !errors.Is(err, identity.ErrKeycloakDuplicateEmail) {
			return subjectID, err
		}
		reconciledSubjectID, matched, reconcileErr :=
			worker.provider.ReconcileProvisionedUser(ctx, user)
		if reconcileErr != nil {
			return "", reconcileErr
		}
		if !matched {
			return "", err
		}
		return reconciledSubjectID, nil
	case UserLifecycleUpdateRoles:
		return claimed.SubjectID, worker.provider.UpdateUserAuthority(
			ctx,
			claimed.SubjectID,
			claimed.OrganizationID,
			claimed.Roles,
		)
	case UserLifecycleSuspend:
		return claimed.SubjectID, worker.provider.DisableUser(
			ctx,
			claimed.SubjectID,
		)
	case UserLifecycleReactivate:
		return claimed.SubjectID, worker.provider.EnableUser(
			ctx,
			claimed.SubjectID,
		)
	default:
		return "", fmt.Errorf(
			"unsupported user lifecycle action %q",
			claimed.Action,
		)
	}
}

func (worker *UserLifecycleWorker) finalizeSuccess(
	ctx context.Context,
	claimed claimedUserLifecycle,
	subjectID string,
) error {
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" {
		return errors.New("identity provider omitted subject ID")
	}
	now := worker.clock().UTC()
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
			var membershipID string
			var membershipRevision int64
			if err := transaction.QueryRow(ctx, `
				SELECT COALESCE(
					(
						SELECT membership_id
						FROM desired_membership_versions
						WHERE subject_id = $1
						ORDER BY revision DESC
						LIMIT 1
					),
					$2
				),
				COALESCE(MAX(revision), 0) + 1
				FROM desired_membership_versions
				WHERE subject_id = $1
			`, subjectID, "membership-"+claimed.RequestID).Scan(
				&membershipID,
				&membershipRevision,
			); err != nil {
				return fmt.Errorf("allocate desired membership revision: %w", err)
			}
			membershipState := "ACTIVE"
			providerEnabled := true
			if claimed.Action == UserLifecycleSuspend {
				membershipState = "SUSPENDED"
				providerEnabled = false
			}
			rawRoles := make([]string, len(claimed.Roles))
			for index, role := range claimed.Roles {
				rawRoles[index] = string(role)
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
				claimed.OrganizationID, rawRoles, claimed.RequestedBy,
				string(claimed.Action), claimed.RequestID,
				claimed.AvailableAt, now, providerEnabled); err != nil {
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
				claimed.OrganizationID, rawRoles, now); err != nil {
				return fmt.Errorf("synchronize desired membership observation: %w", err)
			}

			statusPayload, err := json.Marshal(UserLifecycleRequest{
				ID: claimed.RequestID, SubjectID: subjectID,
				Action: claimed.Action, Roles: claimed.Roles,
				OrganizationID: claimed.OrganizationID,
				Email:          claimed.Email, DisplayName: claimed.DisplayName,
				Status: UserLifecycleSuccess, RequestedBy: claimed.RequestedBy,
				OutboxMessageID: claimed.OutboxID, UpdatedAt: now,
			})
			if err != nil {
				return err
			}
			if _, err := transaction.Exec(ctx, `
				UPDATE user_lifecycle_requests
				SET subject_id = $2,
				    status = 'SUCCEEDED',
				    failure_reason = NULL,
				    updated_at = $3
				WHERE id = $1
			`, claimed.RequestID, subjectID, now); err != nil {
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
					NULLIF($7, ''), NULLIF($8, ''), $6, '{}'::jsonb
				)
			`, worker.idGenerator("audit-user-lifecycle-completed"), now,
				claimed.RequestedBy, claimed.OrganizationID,
				lifecycleSuccessAuditAction(claimed.Action), claimed.RequestID,
				claimed.OperationID, claimed.CorrelationID); err != nil {
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
	providerErr error,
) error {
	now := worker.clock().UTC()
	failure := "identity provider operation failed"
	return database.WithinTransaction(
		ctx,
		worker.pool,
		func(ctx context.Context, transaction pgx.Tx) error {
			result, err := transaction.Exec(ctx, `
				UPDATE user_lifecycle_requests
				SET status = 'FAILED',
				    failure_reason = $2,
				    updated_at = $3
				WHERE id = $1
			`, claimed.RequestID, failure, now)
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
				    last_error = $3
				WHERE id = $1
				  AND lease_owner = $4
				  AND delivered_at IS NULL
			`, claimed.OutboxID, now.Add(worker.retryDelay), failure, worker.workerID)
			if err != nil {
				return err
			}
			if result.RowsAffected() != 1 {
				return errors.New("user lifecycle failure lease changed")
			}
			return nil
		},
	)
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
	default:
		return "USER_LIFECYCLE_SUCCEEDED"
	}
}
