package session

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	identitystore "github.com/aviason/aviaSurveil/internal/identity/store/postgres"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	idleDuration                  = 30 * time.Minute
	absoluteDuration              = 8 * time.Hour
	loginStateTTL                 = 10 * time.Minute
	providerLogoutTicketTTL       = 2 * time.Minute
	loginBindingTTL               = loginStateTTL
	authorityObservationHeartbeat = 30 * time.Second
	authorityObservationMaxAge    = 60 * time.Second
	authorityDenialDeadline       = 120 * time.Second
	// Provider observations are written immediately before the session row is
	// read again during login. Allow a small bounded clock skew between those
	// operations without weakening the stale-observation or denial deadlines.
	authorityObservationClockSkew = 5 * time.Second
)

var (
	ErrUnauthenticated  = errors.New("unauthenticated")
	ErrCSRF             = errors.New("csrf validation failed")
	ErrLoginRateLimited = errors.New("OIDC login admission denied")
)

type authenticationFailureError struct {
	diagnostic string
	cause      error
}

func (failure authenticationFailureError) Error() string { return failure.cause.Error() }
func (failure authenticationFailureError) Unwrap() error { return failure.cause }

func authenticationFailure(diagnostic string, cause error) error {
	switch diagnostic {
	case "missing-token", "session-not-found", "session-read", "expired-or-revoked",
		"unbound-authority", "revocation-pending", "missing-membership",
		"authority-read", "authority-mismatch", "invalid-observation-time",
		"provider-unavailable", "provider-drift", "observation-refresh",
		"identity-reference", "profile-read", "invalid-role", "idle-refresh",
		"transaction", "create-authority-read", "create-authority-mismatch",
		"create-unbound-authority", "create-identity-reference",
		"create-profile-read", "create-observation-refresh", "create-authority-state",
		"create-authority-claims", "create-authority-revisions", "create-desired-revisions",
		"create-desired-state":
		return authenticationFailureError{diagnostic: diagnostic, cause: cause}
	default:
		return authenticationFailureError{diagnostic: "internal", cause: cause}
	}
}

// AuthenticationFailureDiagnostic returns a fixed, privacy-safe stage label.
// It never includes a subject, token, provider payload, SQL text, or raw error.
func AuthenticationFailureDiagnostic(err error) string {
	var failure authenticationFailureError
	if errors.As(err, &failure) {
		if failure.diagnostic != "internal" {
			// A request context can be cancelled while a browser page is
			// navigating or closing. WithinTransaction preserves the stage
			// label, so inspect the cause before reporting it as a database
			// transaction failure.
			if failure.diagnostic == "transaction" &&
				(errors.Is(failure.cause, context.Canceled) || errors.Is(failure.cause, context.DeadlineExceeded)) {
				return "context-expired"
			}
			return failure.diagnostic
		}
		err = failure.cause
	}
	if errors.Is(err, ErrUnauthenticated) {
		return "unauthenticated"
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "context-expired"
	}
	if strings.HasPrefix(err.Error(), "resolve authenticated department authority:") {
		return "department-assignment"
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		switch {
		case postgresError.Code == "42501":
			for _, privilege := range []struct {
				fragment   string
				diagnostic string
			}{
				{fragment: "schema public", diagnostic: "postgres-privilege-public-schema"},
				{fragment: "parameter \"avia.traceparent\"", diagnostic: "postgres-privilege-trace-context"},
				{fragment: "sequence audit_events_sequence_id_seq", diagnostic: "postgres-privilege-audit-sequence"},
			} {
				if strings.Contains(postgresError.Message, privilege.fragment) {
					return privilege.diagnostic
				}
			}
			for _, relation := range []struct {
				name       string
				diagnostic string
			}{
				{name: "session_references", diagnostic: "postgres-privilege-session-references"},
				{name: "identity_references", diagnostic: "postgres-privilege-identity-references"},
				{name: "user_profiles", diagnostic: "postgres-privilege-user-profiles"},
				{name: "desired_membership_versions", diagnostic: "postgres-privilege-membership-versions"},
				{name: "desired_membership_sync", diagnostic: "postgres-privilege-membership-sync"},
				{name: "oidc_login_states", diagnostic: "postgres-privilege-login-states"},
				{name: "audit_events", diagnostic: "postgres-privilege-audit-events"},
			} {
				if postgresError.TableName == relation.name ||
					strings.Contains(postgresError.Message, " "+relation.name) {
					return relation.diagnostic
				}
			}
			return "postgres-insufficient-privilege"
		case strings.HasPrefix(postgresError.Code, "23"):
			return "postgres-integrity"
		case strings.HasPrefix(postgresError.Code, "08"):
			return "postgres-connection"
		default:
			return "postgres"
		}
	}
	return "internal"
}

type freshAuthorityObservationContextKey struct{}

func RequireFreshAuthorityObservation(ctx context.Context) context.Context {
	return context.WithValue(
		ctx,
		freshAuthorityObservationContextKey{},
		true,
	)
}

func requiresFreshAuthorityObservation(ctx context.Context) bool {
	required, _ := ctx.Value(
		freshAuthorityObservationContextKey{},
	).(bool)
	return required
}

type ActivationReconciler interface {
	ReconcileActivatedMembership(
		context.Context,
		string,
		int64,
		[]string,
		bool,
	) error
}

type ManagerDependencies struct {
	Clock                             func() time.Time
	IDGenerator                       func(string) string
	RandomBytes                       func(int) ([]byte, error)
	LoginBindingKey                   []byte
	AuthorityObserver                 identity.AuthorityObserver
	ActivationReconciler              ActivationReconciler
	RequireProviderAuthorityRevisions bool
	LoginAdmissionPolicy              LoginAdmissionPolicy
}

type LoginAdmissionPolicy struct {
	Window                    time.Duration
	GlobalLimit               int
	BrowserLimit              int
	OutstandingLimit          int
	BootstrapOutstandingLimit int
}

func DefaultLoginAdmissionPolicy() LoginAdmissionPolicy {
	return LoginAdmissionPolicy{
		Window:                    time.Minute,
		GlobalLimit:               600,
		BrowserLimit:              60,
		OutstandingLimit:          100,
		BootstrapOutstandingLimit: 20,
	}
}

type Manager struct {
	pool                              *database.Pool
	aead                              cipher.AEAD
	clock                             func() time.Time
	idGenerator                       func(string) string
	randomBytes                       func(int) ([]byte, error)
	loginBindingKey                   []byte
	authorityObserver                 identity.AuthorityObserver
	activationReconciler              ActivationReconciler
	requireProviderAuthorityRevisions bool
	loginAdmission                    LoginAdmissionPolicy
}

func NewManager(pool *database.Pool, encryptionKey []byte, dependencies ManagerDependencies) (*Manager, error) {
	if pool == nil {
		return nil, fmt.Errorf("session PostgreSQL pool is required")
	}
	if len(encryptionKey) != 32 {
		return nil, fmt.Errorf("session encryption key must contain exactly 32 bytes")
	}
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create session token cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create session token AEAD: %w", err)
	}
	clock := dependencies.Clock
	if clock == nil {
		clock = time.Now
	}
	idGenerator := dependencies.IDGenerator
	if idGenerator == nil {
		idGenerator = randomIdentifier
	}
	randomBytes := dependencies.RandomBytes
	if randomBytes == nil {
		randomBytes = secureRandomBytes
	}
	bindingKey := dependencies.LoginBindingKey
	if len(bindingKey) == 0 {
		// Direct package fixtures can use the session key. The deployed API
		// supplies the shared first-party OIDC client secret explicitly.
		bindingKey = encryptionKey
	}
	loginAdmission := dependencies.LoginAdmissionPolicy
	if loginAdmission.Window <= 0 {
		loginAdmission = DefaultLoginAdmissionPolicy()
	}
	if loginAdmission.BootstrapOutstandingLimit <= 0 {
		loginAdmission.BootstrapOutstandingLimit = DefaultLoginAdmissionPolicy().BootstrapOutstandingLimit
	}
	if loginAdmission.GlobalLimit < 1 || loginAdmission.BrowserLimit < 1 || loginAdmission.OutstandingLimit < 1 || loginAdmission.BootstrapOutstandingLimit < 1 {
		return nil, fmt.Errorf("OIDC login admission policy is invalid")
	}
	return &Manager{
		pool: pool, aead: aead, clock: clock, idGenerator: idGenerator,
		randomBytes:                       randomBytes,
		loginBindingKey:                   deriveLoginBindingKey(bindingKey),
		authorityObserver:                 dependencies.AuthorityObserver,
		activationReconciler:              dependencies.ActivationReconciler,
		requireProviderAuthorityRevisions: dependencies.RequireProviderAuthorityRevisions,
		loginAdmission:                    loginAdmission,
	}, nil
}

type CreateInput struct {
	SubjectID          string
	Issuer             string
	DisplayName        string
	Email              string
	OrganizationID     string
	Roles              []identity.Role
	ProviderSessionID  string
	MembershipID       string
	MembershipRevision int64
	AuthRevision       uint64
	ProviderTokens     identity.ProviderTokens
}

type BrowserSession struct {
	ID                string
	Token             string
	CSRFToken         string
	ExpiresAt         time.Time
	AbsoluteExpiresAt time.Time
	Principal         identity.Principal
}

type desiredSessionAuthority struct {
	MembershipID            string
	Revision                int64
	State                   string
	OrganizationID          string
	Roles                   []identity.Role
	EffectiveAt             time.Time
	SyncRevision            int64
	ObservedProviderEnabled bool
	ObservedOrganizationID  string
	ObservedRoles           []identity.Role
	ObservedAt              time.Time
	DriftState              string
}

func (manager *Manager) Create(ctx context.Context, input CreateInput) (BrowserSession, error) {
	if strings.TrimSpace(input.SubjectID) == "" || strings.TrimSpace(input.Issuer) == "" || strings.TrimSpace(input.DisplayName) == "" || strings.TrimSpace(input.OrganizationID) == "" || len(input.Roles) == 0 {
		return BrowserSession{}, fmt.Errorf("subject, issuer, display name, organization, and roles are required")
	}
	if err := identity.ValidateApplicationAuthority(
		input.OrganizationID,
		input.Roles,
	); err != nil {
		return BrowserSession{}, authenticationFailure("create-authority-mismatch", ErrUnauthenticated)
	}
	if manager.authorityObserver == nil {
		return BrowserSession{}, authenticationFailure("create-authority-read", ErrUnauthenticated)
	}
	observation, err := manager.authorityObserver.ObserveUserAuthority(
		ctx,
		input.SubjectID,
	)
	if err != nil || observation.SubjectID != input.SubjectID {
		return BrowserSession{}, authenticationFailure("create-authority-read", ErrUnauthenticated)
	}
	if !observation.Enabled || observation.Locked || len(observation.RequiredActions) != 0 {
		return BrowserSession{}, authenticationFailure("create-authority-state", ErrUnauthenticated)
	}
	if !identity.EqualApplicationAuthority(
		input.OrganizationID,
		input.Roles,
		observation.OrganizationID,
		observation.Roles,
	) {
		return BrowserSession{}, authenticationFailure("create-authority-claims", ErrUnauthenticated)
	}
	if (input.MembershipID != "" && input.MembershipID != observation.MembershipID) ||
		(input.MembershipRevision > 0 && input.MembershipRevision != observation.MembershipRevision) ||
		(input.AuthRevision > 0 && input.AuthRevision != observation.AuthRevision) {
		return BrowserSession{}, authenticationFailure("create-authority-revisions", ErrUnauthenticated)
	}
	if manager.requireProviderAuthorityRevisions &&
		(input.MembershipID == "" || input.MembershipRevision <= 0 || input.AuthRevision == 0 ||
			observation.MembershipID != input.MembershipID ||
			observation.MembershipRevision != input.MembershipRevision ||
			observation.AuthRevision != input.AuthRevision) {
		return BrowserSession{}, authenticationFailure("create-authority-revisions", ErrUnauthenticated)
	}
	now := manager.clock().UTC()
	authority, err := loadDesiredSessionAuthority(
		ctx,
		manager.pool,
		input.SubjectID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return BrowserSession{}, authenticationFailure("create-unbound-authority", ErrUnauthenticated)
	}
	if err != nil {
		return BrowserSession{}, fmt.Errorf(
			"read desired session authority before login: %w",
			err,
		)
	}
	if manager.requireProviderAuthorityRevisions &&
		(authority.MembershipID != input.MembershipID ||
			authority.Revision != input.MembershipRevision) {
		return BrowserSession{}, authenticationFailure("create-desired-revisions", ErrUnauthenticated)
	}
	if authority.State == "INVITED" {
		if manager.activationReconciler == nil ||
			authority.EffectiveAt.After(now) ||
			authority.SyncRevision != authority.Revision ||
			!authority.ObservedProviderEnabled ||
			authority.DriftState != "IN_SYNC" ||
			!identity.EqualApplicationAuthority(
				authority.OrganizationID,
				authority.Roles,
				authority.ObservedOrganizationID,
				authority.ObservedRoles,
			) ||
			!identity.EqualApplicationAuthority(
				authority.OrganizationID,
				authority.Roles,
				input.OrganizationID,
				input.Roles,
			) ||
			!identity.EqualApplicationAuthority(
				authority.OrganizationID,
				authority.Roles,
				observation.OrganizationID,
				observation.Roles,
			) {
			return BrowserSession{}, authenticationFailure("create-unbound-authority", ErrUnauthenticated)
		}
		if err := manager.activationReconciler.ReconcileActivatedMembership(
			ctx,
			input.SubjectID,
			authority.Revision,
			observation.RequiredActions,
			observation.MFAEnrolled,
		); err != nil {
			return BrowserSession{}, authenticationFailure("create-authority-mismatch", ErrUnauthenticated)
		}
		now = manager.clock().UTC()
	}
	rawToken, err := manager.opaqueToken(32)
	if err != nil {
		return BrowserSession{}, fmt.Errorf("generate session token: %w", err)
	}
	rawCSRF, err := manager.opaqueToken(32)
	if err != nil {
		return BrowserSession{}, fmt.Errorf("generate CSRF token: %w", err)
	}
	providerCiphertext, err := manager.encryptProviderTokens(input.ProviderTokens)
	if err != nil {
		return BrowserSession{}, err
	}
	sessionID := manager.idGenerator("session")
	idleExpiresAt := now.Add(idleDuration)
	absoluteExpiresAt := now.Add(absoluteDuration)
	err = database.WithinTransaction(ctx, manager.pool, func(ctx context.Context, transaction pgx.Tx) error {
		authority, err := loadDesiredSessionAuthority(
			ctx,
			transaction,
			input.SubjectID,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return authenticationFailure("create-unbound-authority", ErrUnauthenticated)
		}
		if err != nil {
			return fmt.Errorf("read desired session authority: %w", err)
		}
		if authority.State != "ACTIVE" ||
			authority.EffectiveAt.After(now) ||
			authority.SyncRevision != authority.Revision ||
			!authority.ObservedProviderEnabled ||
			authority.DriftState != "IN_SYNC" ||
			!identity.EqualApplicationAuthority(
				authority.OrganizationID,
				authority.Roles,
				authority.ObservedOrganizationID,
				authority.ObservedRoles,
			) ||
			!identity.EqualApplicationAuthority(
				authority.OrganizationID,
				authority.Roles,
				input.OrganizationID,
				input.Roles,
			) ||
			!identity.EqualApplicationAuthority(
				authority.OrganizationID,
				authority.Roles,
				observation.OrganizationID,
				observation.Roles,
			) ||
			manager.requireProviderAuthorityRevisions &&
				(authority.MembershipID != input.MembershipID ||
					authority.Revision != input.MembershipRevision ||
					observation.MembershipID != authority.MembershipID ||
					observation.MembershipRevision != authority.Revision ||
					observation.AuthRevision != input.AuthRevision) {
			return authenticationFailure("create-authority-mismatch", ErrUnauthenticated)
		}
		var persistedSubjectID, persistedIssuer string
		if err := transaction.QueryRow(ctx, `
			UPDATE identity_references
			SET display_name = $3,
			    email = COALESCE(NULLIF(lower(trim($4)), ''), email)
			WHERE subject_id = $1
			  AND issuer = $2
			  AND tombstoned_at IS NULL
			  AND deactivated_at IS NULL
			RETURNING subject_id, issuer
		`, input.SubjectID, input.Issuer, input.DisplayName,
			input.Email).Scan(
			&persistedSubjectID,
			&persistedIssuer,
		); errors.Is(err, pgx.ErrNoRows) {
			return authenticationFailure("create-identity-reference", ErrUnauthenticated)
		} else if err != nil {
			return fmt.Errorf("refresh authenticated identity reference: %w", err)
		}
		var profileOrganizationID *string
		if err := transaction.QueryRow(ctx, `
			SELECT organization_id
			FROM user_profiles
			WHERE subject_id = $1
			  AND tombstoned_at IS NULL
		`, input.SubjectID).Scan(&profileOrganizationID); errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return authenticationFailure("create-profile-read", ErrUnauthenticated)
		} else if err != nil {
			return fmt.Errorf("read authenticated user profile: %w", err)
		}
		if profileOrganizationID == nil ||
			*profileOrganizationID != authority.OrganizationID {
			return authenticationFailure("create-profile-read", ErrUnauthenticated)
		}
		observationResult, err := transaction.Exec(ctx, `
			UPDATE desired_membership_sync
			SET observed_provider_enabled = true,
			    observed_organization_id = $2,
			    observed_roles = $3,
			    observed_at = $4,
			    drift_state = 'IN_SYNC'
			WHERE membership_id = $1
			  AND desired_revision = $5
		`, authority.MembershipID, observation.OrganizationID,
			rolesToStrings(observation.Roles), now,
			authority.Revision)
		if err != nil {
			return fmt.Errorf("refresh provider authority observation: %w", err)
		}
		if observationResult.RowsAffected() != 1 {
			return authenticationFailure("create-observation-refresh", ErrUnauthenticated)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO session_references (
				id, subject_id, organization_id, provider_session_id, expires_at, created_at,
				session_token_hash, csrf_token_hash, last_seen_at, absolute_expires_at, roles,
				provider_tokens_ciphertext, membership_id, membership_revision,
				provider_auth_revision, authority_observed_at, authority_state
			) VALUES (
				$1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $6, $9, $10,
				$11, $12, $13, NULLIF($14, 0), $6, 'ACTIVE'
			)
		`, sessionID, input.SubjectID, authority.OrganizationID,
			input.ProviderSessionID, idleExpiresAt, now,
			hashToken(rawToken), hashToken(rawCSRF), absoluteExpiresAt,
			rolesToStrings(authority.Roles), providerCiphertext,
			authority.MembershipID, authority.Revision, input.AuthRevision); err != nil {
			return fmt.Errorf("persist browser session: %w", err)
		}
		return nil
	})
	if err != nil {
		return BrowserSession{}, err
	}
	principal := identity.Principal{
		SubjectID: input.SubjectID, DisplayName: input.DisplayName,
		OrganizationID: observation.OrganizationID,
		Roles:          append([]identity.Role(nil), observation.Roles...),
		SessionID:      sessionID,
	}
	return BrowserSession{
		ID: sessionID, Token: rawToken, CSRFToken: rawCSRF, ExpiresAt: idleExpiresAt,
		AbsoluteExpiresAt: absoluteExpiresAt, Principal: principal,
	}, nil
}

func (manager *Manager) Authenticate(ctx context.Context, rawToken string) (identity.Principal, error) {
	if strings.TrimSpace(rawToken) == "" {
		return identity.Principal{}, authenticationFailure("missing-token", ErrUnauthenticated)
	}
	now := manager.clock().UTC()
	var principal identity.Principal
	var outcome error
	err := database.WithinTransaction(ctx, manager.pool, func(ctx context.Context, transaction pgx.Tx) error {
		tokenHash := hashToken(rawToken)
		record, err := identitystore.New(transaction).GetSessionForAuthentication(ctx, &tokenHash)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return authenticationFailure("session-not-found", ErrUnauthenticated)
			}
			return authenticationFailure("session-read", fmt.Errorf("read browser session: %w", err))
		}
		if !record.ExpiresAt.Valid || !record.AbsoluteExpiresAt.Valid || record.RevokedAt.Valid ||
			!now.Before(record.ExpiresAt.Time) || !now.Before(record.AbsoluteExpiresAt.Time) {
			return authenticationFailure("expired-or-revoked", ErrUnauthenticated)
		}
		if record.MembershipID == nil ||
			record.MembershipRevision == nil ||
			record.OrganizationID == nil ||
			record.AuthorityState == nil ||
			(*record.AuthorityState != "ACTIVE" &&
				*record.AuthorityState != "REVOCATION_PENDING") {
			if err := manager.denySessionAuthority(
				ctx,
				transaction,
				record.ID,
				record.SubjectID,
				record.OrganizationID,
				record.Roles,
				record.MembershipRevision,
				"UNBOUND_SESSION_AUTHORITY",
				now,
			); err != nil {
				return err
			}
			outcome = authenticationFailure("unbound-authority", ErrUnauthenticated)
			return nil
		}
		if *record.AuthorityState == "REVOCATION_PENDING" {
			if !record.AuthorityObservedAt.Valid ||
				!now.Before(
					record.AuthorityObservedAt.Time.Add(
						authorityDenialDeadline,
					),
				) {
				if err := manager.denySessionAuthority(
					ctx,
					transaction,
					record.ID,
					record.SubjectID,
					record.OrganizationID,
					record.Roles,
					record.MembershipRevision,
					"PROVIDER_OBSERVATION_DEADLINE",
					now,
				); err != nil {
					return err
				}
			}
			outcome = authenticationFailure("revocation-pending", ErrUnauthenticated)
			return nil
		}
		authority, err := loadDesiredSessionAuthority(
			ctx,
			transaction,
			record.SubjectID,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			if err := manager.denySessionAuthority(
				ctx,
				transaction,
				record.ID,
				record.SubjectID,
				record.OrganizationID,
				record.Roles,
				record.MembershipRevision,
				"MISSING_DESIRED_MEMBERSHIP",
				now,
			); err != nil {
				return err
			}
			outcome = authenticationFailure("missing-membership", ErrUnauthenticated)
			return nil
		}
		if err != nil {
			return authenticationFailure("authority-read", fmt.Errorf("read current session authority: %w", err))
		}
		sessionRoles := rolesFromStrings(record.Roles)
		if authority.MembershipID != *record.MembershipID ||
			authority.Revision != *record.MembershipRevision ||
			authority.State != "ACTIVE" ||
			authority.EffectiveAt.After(now) ||
			authority.SyncRevision != authority.Revision ||
			!authority.ObservedProviderEnabled ||
			authority.DriftState != "IN_SYNC" ||
			!identity.EqualApplicationAuthority(
				authority.OrganizationID,
				authority.Roles,
				*record.OrganizationID,
				sessionRoles,
			) ||
			!identity.EqualApplicationAuthority(
				authority.OrganizationID,
				authority.Roles,
				authority.ObservedOrganizationID,
				authority.ObservedRoles,
			) {
			if err := manager.denySessionAuthority(
				ctx,
				transaction,
				record.ID,
				record.SubjectID,
				record.OrganizationID,
				record.Roles,
				record.MembershipRevision,
				"SESSION_AUTHORITY_MISMATCH",
				now,
			); err != nil {
				return err
			}
			outcome = authenticationFailure("authority-mismatch", ErrUnauthenticated)
			return nil
		}
		if !record.AuthorityObservedAt.Valid ||
			now.Add(authorityObservationClockSkew).Before(
				record.AuthorityObservedAt.Time,
			) {
			if err := manager.denySessionAuthority(
				ctx,
				transaction,
				record.ID,
				record.SubjectID,
				record.OrganizationID,
				record.Roles,
				record.MembershipRevision,
				"INVALID_PROVIDER_OBSERVATION_TIME",
				now,
			); err != nil {
				return err
			}
			outcome = authenticationFailure("invalid-observation-time", ErrUnauthenticated)
			return nil
		}
		observationAge := now.Sub(record.AuthorityObservedAt.Time)
		if observationAge >= authorityObservationHeartbeat ||
			requiresFreshAuthorityObservation(ctx) {
			observation, observationErr :=
				manager.observeSessionAuthority(
					ctx,
					record.SubjectID,
					now,
				)
			if observationErr != nil {
				if observationAge >= authorityDenialDeadline {
					if err := manager.denySessionAuthority(
						ctx,
						transaction,
						record.ID,
						record.SubjectID,
						record.OrganizationID,
						record.Roles,
						record.MembershipRevision,
						"PROVIDER_OBSERVATION_DEADLINE",
						now,
					); err != nil {
						return err
					}
					outcome = authenticationFailure("provider-unavailable", ErrUnauthenticated)
					return nil
				}
				reason := "PROVIDER_UNAVAILABLE"
				if observationAge > authorityObservationMaxAge {
					reason = "STALE_PROVIDER_OBSERVATION"
				}
				if err := manager.markSessionAuthorityPending(
					ctx,
					transaction,
					record.ID,
					record.SubjectID,
					record.OrganizationID,
					record.Roles,
					record.MembershipRevision,
					authority.MembershipID,
					reason,
					now,
				); err != nil {
					return err
				}
				outcome = authenticationFailure("provider-unavailable", ErrUnauthenticated)
				return nil
			}
			if record.ProviderAuthRevision != nil &&
				(observation.AuthRevision == 0 || uint64(*record.ProviderAuthRevision) != observation.AuthRevision) {
				if err := manager.denySessionAuthority(
					ctx, transaction, record.ID, record.SubjectID, record.OrganizationID,
					record.Roles, record.MembershipRevision, "AUTH_REVISION_MISMATCH", now,
				); err != nil {
					return err
				}
				outcome = authenticationFailure("authority-mismatch", ErrUnauthenticated)
				return nil
			}
			if manager.requireProviderAuthorityRevisions &&
				(observation.MembershipID == "" ||
					observation.MembershipID != *record.MembershipID ||
					observation.MembershipRevision != *record.MembershipRevision) ||
				!observation.Enabled ||
				observation.Locked ||
				len(observation.RequiredActions) != 0 ||
				!identity.EqualApplicationAuthority(
					authority.OrganizationID,
					authority.Roles,
					observation.OrganizationID,
					observation.Roles,
				) {
				if err := manager.recordProviderAuthorityDrift(
					ctx,
					transaction,
					authority.MembershipID,
					observation,
					now,
				); err != nil {
					return err
				}
				if err := manager.denySessionAuthority(
					ctx,
					transaction,
					record.ID,
					record.SubjectID,
					record.OrganizationID,
					record.Roles,
					record.MembershipRevision,
					"PROVIDER_AUTHORITY_DRIFT",
					now,
				); err != nil {
					return err
				}
				outcome = authenticationFailure("provider-drift", ErrUnauthenticated)
				return nil
			}
			if err := manager.refreshSessionAuthorityObservation(
				ctx,
				transaction,
				record.ID,
				authority.MembershipID,
				authority.Revision,
				observation,
				now,
			); err != nil {
				return authenticationFailure("observation-refresh", err)
			}
		}
		principal.SessionID = record.ID
		principal.SubjectID = record.SubjectID
		identityReference, err := identitystore.New(transaction).GetIdentityReference(ctx, record.SubjectID)
		if err != nil {
			return authenticationFailure("identity-reference", fmt.Errorf("read authenticated identity reference: %w", err))
		}
		principal.DisplayName = identityReference.DisplayName
		profile, err := identitystore.New(transaction).GetProfile(ctx, record.SubjectID)
		if err == nil {
			principal.DisplayName = profile.DisplayName
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return authenticationFailure("profile-read", fmt.Errorf("read authenticated user profile: %w", err))
		}
		if record.OrganizationID != nil {
			principal.OrganizationID = *record.OrganizationID
		}
		principal.Roles = make([]identity.Role, len(record.Roles))
		for index, role := range record.Roles {
			principal.Roles[index] = identity.Role(role)
			if !validRole(principal.Roles[index]) {
				return authenticationFailure("invalid-role", ErrUnauthenticated)
			}
		}
		nextIdleExpiry := now.Add(idleDuration)
		if nextIdleExpiry.After(record.AbsoluteExpiresAt.Time) {
			nextIdleExpiry = record.AbsoluteExpiresAt.Time
		}
		if _, err := transaction.Exec(ctx, `
			UPDATE session_references SET last_seen_at = $2, expires_at = $3 WHERE id = $1
		`, principal.SessionID, now, nextIdleExpiry); err != nil {
			return authenticationFailure("idle-refresh", fmt.Errorf("refresh browser session idle expiry: %w", err))
		}
		return nil
	})
	if err != nil {
		var failure authenticationFailureError
		if errors.As(err, &failure) {
			return identity.Principal{}, err
		}
		return identity.Principal{}, authenticationFailure("transaction", err)
	}
	if outcome != nil {
		return identity.Principal{}, outcome
	}
	if principal.HasRole(identity.RoleDepartmentManager) {
		assignments, assignmentErr := identity.ResolveEffectiveDepartmentAssignments(
			ctx, manager.pool, principal.SubjectID, now,
		)
		if assignmentErr != nil {
			return identity.Principal{}, fmt.Errorf("resolve authenticated department authority: %w", assignmentErr)
		}
		principal.DepartmentAssignments = assignments
	}
	return principal, nil
}

func (manager *Manager) observeSessionAuthority(
	ctx context.Context,
	subjectID string,
	now time.Time,
) (identity.AuthorityObservation, error) {
	if manager.authorityObserver == nil {
		return identity.AuthorityObservation{}, ErrUnauthenticated
	}
	observation, err := manager.authorityObserver.ObserveUserAuthority(
		ctx,
		subjectID,
	)
	if err != nil {
		return identity.AuthorityObservation{}, err
	}
	if observation.SubjectID != subjectID {
		return identity.AuthorityObservation{}, ErrUnauthenticated
	}
	observation.ObservedAt = now
	return observation, nil
}

func (manager *Manager) refreshSessionAuthorityObservation(
	ctx context.Context,
	transaction pgx.Tx,
	sessionID,
	membershipID string,
	membershipRevision int64,
	observation identity.AuthorityObservation,
	now time.Time,
) error {
	result, err := transaction.Exec(ctx, `
		UPDATE desired_membership_sync
		SET observed_provider_enabled = $3,
		    observed_organization_id = $4,
		    observed_roles = $5,
		    observed_at = $6,
		    drift_state = 'IN_SYNC'
		WHERE membership_id = $1
		  AND desired_revision = $2
	`, membershipID, membershipRevision, observation.Enabled,
		observation.OrganizationID, rolesToStrings(observation.Roles), now)
	if err != nil {
		return fmt.Errorf("refresh desired membership observation: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrUnauthenticated
	}
	if _, err := transaction.Exec(ctx, `
		UPDATE session_references
		SET authority_observed_at = $2,
		    authority_state = 'ACTIVE'
		WHERE id = $1
		  AND revoked_at IS NULL
	`, sessionID, now); err != nil {
		return fmt.Errorf("refresh session authority observation: %w", err)
	}
	return nil
}

func (manager *Manager) recordProviderAuthorityDrift(
	ctx context.Context,
	transaction pgx.Tx,
	membershipID string,
	observation identity.AuthorityObservation,
	now time.Time,
) error {
	if _, err := transaction.Exec(ctx, `
		UPDATE desired_membership_sync
		SET observed_provider_enabled = $2,
		    observed_organization_id = $3,
		    observed_roles = $4,
		    observed_at = $5,
		    drift_state = 'DRIFTED'
		WHERE membership_id = $1
	`, membershipID, observation.Enabled, observation.OrganizationID,
		rolesToStrings(observation.Roles), now); err != nil {
		return fmt.Errorf("record provider authority drift: %w", err)
	}
	return nil
}

func (manager *Manager) markSessionAuthorityPending(
	ctx context.Context,
	transaction pgx.Tx,
	sessionID,
	subjectID string,
	organizationID *string,
	roles []string,
	membershipRevision *int64,
	membershipID,
	reason string,
	now time.Time,
) error {
	if reason == "PROVIDER_UNAVAILABLE" {
		if _, err := transaction.Exec(ctx, `
			UPDATE desired_membership_sync
			SET drift_state = 'PROVIDER_UNAVAILABLE'
			WHERE membership_id = $1
		`, membershipID); err != nil {
			return fmt.Errorf(
				"record unavailable provider observation: %w",
				err,
			)
		}
	} else {
		if _, err := transaction.Exec(ctx, `
			UPDATE desired_membership_sync
			SET drift_state = 'STALE'
			WHERE membership_id = $1
		`, membershipID); err != nil {
			return fmt.Errorf("record stale provider observation: %w", err)
		}
	}
	if _, err := transaction.Exec(ctx, `
		UPDATE session_references
		SET authority_state = 'REVOCATION_PENDING',
		    provider_tokens_ciphertext = NULL
		WHERE id = $1
		  AND revoked_at IS NULL
	`, sessionID); err != nil {
		return fmt.Errorf("mark session authority revocation pending: %w", err)
	}
	actorRole := ""
	if len(roles) > 0 {
		actorRole = roles[0]
	}
	entityVersion := int64(1)
	if membershipRevision != nil && *membershipRevision > 0 {
		entityVersion = *membershipRevision
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO audit_events (
			event_id, occurred_at, actor_subject_id, actor_role,
			organization_id, action, entity_type, entity_id,
			entity_version, after_status, details
		) VALUES (
			$1, $2, $3, NULLIF($4, ''), $5,
			'SESSION_AUTHORITY_REVOCATION_PENDING', 'SESSION', $6,
			$7, 'REVOCATION_PENDING',
			jsonb_build_object('reasonCode', $8::text)
		)
	`, manager.idGenerator("audit-session-authority"), now, subjectID,
		actorRole, organizationID, sessionID, entityVersion, reason); err != nil {
		return fmt.Errorf(
			"audit session authority revocation pending: %w",
			err,
		)
	}
	return nil
}

type sessionAuthorityQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadDesiredSessionAuthority(
	ctx context.Context,
	querier sessionAuthorityQuerier,
	subjectID string,
) (desiredSessionAuthority, error) {
	var output desiredSessionAuthority
	var roles, observedRoles []string
	err := querier.QueryRow(ctx, `
		SELECT version.membership_id, version.revision,
		       version.membership_state, version.organization_id,
		       version.roles, version.effective_at,
		       sync.desired_revision, sync.observed_provider_enabled,
		       sync.observed_organization_id, sync.observed_roles,
		       sync.observed_at, sync.drift_state
		FROM desired_membership_versions version
		JOIN desired_membership_sync sync
		  ON sync.membership_id = version.membership_id
		WHERE version.subject_id = $1
		ORDER BY version.revision DESC
		LIMIT 1
	`, subjectID).Scan(
		&output.MembershipID,
		&output.Revision,
		&output.State,
		&output.OrganizationID,
		&roles,
		&output.EffectiveAt,
		&output.SyncRevision,
		&output.ObservedProviderEnabled,
		&output.ObservedOrganizationID,
		&observedRoles,
		&output.ObservedAt,
		&output.DriftState,
	)
	if err != nil {
		return desiredSessionAuthority{}, err
	}
	output.Roles = rolesFromStrings(roles)
	output.ObservedRoles = rolesFromStrings(observedRoles)
	return output, nil
}

func (manager *Manager) denySessionAuthority(
	ctx context.Context,
	transaction pgx.Tx,
	sessionID,
	subjectID string,
	organizationID *string,
	roles []string,
	membershipRevision *int64,
	reason string,
	now time.Time,
) error {
	if _, err := transaction.Exec(ctx, `
		UPDATE session_references
		SET revoked_at = COALESCE(revoked_at, $2),
		    provider_tokens_ciphertext = NULL,
		    authority_state = 'DENIED_STALE_AUTHORITY'
		WHERE id = $1
	`, sessionID, now); err != nil {
		return fmt.Errorf("deny stale session authority: %w", err)
	}
	actorRole := ""
	if len(roles) > 0 {
		actorRole = roles[0]
	}
	entityVersion := int64(1)
	if membershipRevision != nil && *membershipRevision > 0 {
		entityVersion = *membershipRevision
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO audit_events (
			event_id, occurred_at, actor_subject_id, actor_role,
			organization_id, action, entity_type, entity_id,
			entity_version, after_status, details
		) VALUES (
			$1, $2, $3, NULLIF($4, ''), $5,
			'SESSION_AUTHORITY_DENIED', 'SESSION', $6,
			$7, 'DENIED_STALE_AUTHORITY',
			jsonb_build_object('reasonCode', $8::text)
		)
	`, manager.idGenerator("audit-session-authority"), now, subjectID,
		actorRole, organizationID, sessionID, entityVersion, reason); err != nil {
		return fmt.Errorf("audit stale session authority denial: %w", err)
	}
	return nil
}

func rolesFromStrings(values []string) []identity.Role {
	roles := make([]identity.Role, len(values))
	for index, value := range values {
		roles[index] = identity.Role(value)
	}
	return roles
}

func rolesToStrings(values []identity.Role) []string {
	roles := make([]string, len(values))
	for index, value := range values {
		roles[index] = string(value)
	}
	return roles
}

func (manager *Manager) ValidateCSRF(ctx context.Context, sessionID, rawCSRF string) error {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(rawCSRF) == "" {
		return ErrCSRF
	}
	var storedHash string
	var expiresAt, absoluteExpiresAt time.Time
	var revokedAt *time.Time
	if err := manager.pool.QueryRow(ctx, `
		SELECT csrf_token_hash, expires_at, absolute_expires_at, revoked_at
		FROM session_references WHERE id = $1
	`, sessionID).Scan(&storedHash, &expiresAt, &absoluteExpiresAt, &revokedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCSRF
		}
		return fmt.Errorf("read session CSRF authority: %w", err)
	}
	now := manager.clock().UTC()
	actualHash := hashToken(rawCSRF)
	if revokedAt != nil || !now.Before(expiresAt) || !now.Before(absoluteExpiresAt) || subtle.ConstantTimeCompare([]byte(storedHash), []byte(actualHash)) != 1 {
		return ErrCSRF
	}
	return nil
}

func (manager *Manager) Revoke(ctx context.Context, sessionID string) (string, error) {
	if strings.TrimSpace(sessionID) == "" {
		return "", ErrUnauthenticated
	}
	now := manager.clock().UTC()
	var providerLogoutTicket string
	err := database.WithinTransaction(ctx, manager.pool, func(ctx context.Context, transaction pgx.Tx) error {
		var subjectID string
		var organizationID *string
		var roles []string
		var providerTokensCiphertext []byte
		if err := transaction.QueryRow(ctx, `
			SELECT subject_id, organization_id, roles, provider_tokens_ciphertext
			FROM session_references
			WHERE id = $1
			  AND revoked_at IS NULL
			FOR UPDATE
		`, sessionID).Scan(
			&subjectID,
			&organizationID,
			&roles,
			&providerTokensCiphertext,
		); errors.Is(err, pgx.ErrNoRows) {
			return ErrUnauthenticated
		} else if err != nil {
			return fmt.Errorf("read browser session for revocation: %w", err)
		}
		providerIDToken := ""
		if len(providerTokensCiphertext) > 0 {
			providerTokens, err := manager.decryptProviderTokens(
				providerTokensCiphertext,
			)
			if err != nil {
				return err
			}
			providerIDToken = strings.TrimSpace(providerTokens.IDToken)
		}
		ticket, ticketErr := manager.encryptProviderLogoutTicket(
			providerIDToken,
			now.Add(providerLogoutTicketTTL),
		)
		if ticketErr != nil {
			return ticketErr
		}
		providerLogoutTicket = ticket
		result, err := transaction.Exec(ctx, `
			UPDATE session_references
			SET revoked_at = $2,
			    provider_tokens_ciphertext = NULL
			WHERE id = $1
			  AND revoked_at IS NULL
		`, sessionID, now)
		if err != nil {
			return fmt.Errorf("revoke browser session: %w", err)
		}
		if result.RowsAffected() != 1 {
			return ErrUnauthenticated
		}
		actorRole := ""
		if len(roles) > 0 {
			actorRole = roles[0]
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO audit_events (
				event_id, occurred_at, actor_subject_id, actor_role, organization_id,
				action, entity_type, entity_id, entity_version, after_status, details
			) VALUES (
				$1, $2, $3, NULLIF($4, ''), $5,
				'SESSION_REVOKED', 'SESSION', $6, 1, 'REVOKED', '{}'::jsonb
			)
		`, manager.idGenerator("audit-session"), now, subjectID, actorRole, organizationID, sessionID); err != nil {
			return fmt.Errorf("append session revocation audit event: %w", err)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return providerLogoutTicket, nil
}

func (manager *Manager) RedeemProviderLogout(
	ctx context.Context,
	rawTicket string,
) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	ticket, err := manager.decryptProviderLogoutTicket(rawTicket)
	if err != nil || !manager.clock().UTC().Before(ticket.ExpiresAt) {
		return "", ErrUnauthenticated
	}
	return strings.TrimSpace(ticket.IDToken), nil
}

type LoginRequest struct {
	State             string
	Nonce             string
	PKCEChallenge     string
	ReturnTo          string
	BrowserBinding    string
	SetBrowserBinding bool
}

type LoginState struct {
	Nonce        string
	PKCEVerifier string
	ReturnTo     string
}

func (manager *Manager) NewLoginState(ctx context.Context, returnTo, currentBinding string) (LoginRequest, error) {
	state, err := manager.opaqueToken(32)
	if err != nil {
		return LoginRequest{}, fmt.Errorf("generate OIDC state: %w", err)
	}
	nonce, err := manager.opaqueToken(32)
	if err != nil {
		return LoginRequest{}, fmt.Errorf("generate OIDC nonce: %w", err)
	}
	verifier, err := manager.opaqueToken(32)
	if err != nil {
		return LoginRequest{}, fmt.Errorf("generate PKCE verifier: %w", err)
	}
	binding, setBinding, err := manager.loginBinding(currentBinding)
	if err != nil {
		return LoginRequest{}, fmt.Errorf("generate OIDC browser binding: %w", err)
	}
	returnTo = safeReturnTo(returnTo)
	now := manager.clock().UTC()
	bindingHash := hashBrowserBinding(binding)
	tx, err := manager.pool.Begin(ctx)
	if err != nil {
		return LoginRequest{}, fmt.Errorf("begin OIDC login state admission: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := manager.admitLoginState(ctx, tx, bindingHash, setBinding, now); err != nil {
		return LoginRequest{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO oidc_login_states (state_hash, nonce, pkce_verifier, return_to, expires_at, created_at, browser_binding_hash, browser_binding_bootstrap)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, hashToken(state), nonce, verifier, returnTo, now.Add(loginStateTTL), now, bindingHash, setBinding); err != nil {
		return LoginRequest{}, fmt.Errorf("persist OIDC login state: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return LoginRequest{}, fmt.Errorf("commit OIDC login state: %w", err)
	}
	challengeHash := sha256.Sum256([]byte(verifier))
	return LoginRequest{
		State: state, Nonce: nonce, PKCEChallenge: base64.RawURLEncoding.EncodeToString(challengeHash[:]), ReturnTo: returnTo,
		BrowserBinding: binding, SetBrowserBinding: setBinding,
	}, nil
}

func (manager *Manager) ConsumeLoginState(ctx context.Context, rawState, rawBinding string) (LoginState, error) {
	_, err := decodeBrowserBinding(rawBinding, manager.loginBindingKey)
	if strings.TrimSpace(rawState) == "" || err != nil {
		return LoginState{}, ErrUnauthenticated
	}
	var state LoginState
	err = manager.pool.QueryRow(ctx, `
		DELETE FROM oidc_login_states
		WHERE state_hash = $1 AND browser_binding_hash = $2 AND expires_at > $3
		RETURNING nonce, pkce_verifier, return_to
	`, hashToken(rawState), hashBrowserBinding(strings.TrimSpace(rawBinding)), manager.clock().UTC()).Scan(&state.Nonce, &state.PKCEVerifier, &state.ReturnTo)
	if errors.Is(err, pgx.ErrNoRows) {
		return LoginState{}, ErrUnauthenticated
	}
	if err != nil {
		return LoginState{}, fmt.Errorf("consume OIDC login state: %w", err)
	}
	return state, nil
}

func (manager *Manager) loginBinding(current string) (string, bool, error) {
	if _, err := decodeBrowserBinding(current, manager.loginBindingKey); err == nil {
		return strings.TrimSpace(current), false, nil
	}
	rawBinding, err := manager.randomBytes(32)
	if err != nil {
		return "", false, err
	}
	if len(rawBinding) != 32 {
		return "", false, fmt.Errorf("random source returned %d bytes, expected 32", len(rawBinding))
	}
	return encodeBrowserBinding(rawBinding, manager.loginBindingKey), true, nil
}

func deriveLoginBindingKey(raw []byte) []byte {
	hash := sha256.New()
	_, _ = hash.Write([]byte("as360-oidc-login-binding-key-v1\x00"))
	_, _ = hash.Write(raw)
	return hash.Sum(nil)
}

func encodeBrowserBinding(raw []byte, key []byte) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("as360-oidc-login-binding-v1\x00"))
	_, _ = mac.Write(raw)
	return base64.RawURLEncoding.EncodeToString(raw) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func decodeBrowserBinding(raw string, key []byte) ([]byte, error) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 2 {
		return nil, ErrUnauthenticated
	}
	value, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || len(value) != 32 {
		return nil, ErrUnauthenticated
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(signature) != sha256.Size {
		return nil, ErrUnauthenticated
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("as360-oidc-login-binding-v1\x00"))
	_, _ = mac.Write(value)
	if subtle.ConstantTimeCompare(signature, mac.Sum(nil)) != 1 {
		return nil, ErrUnauthenticated
	}
	return value, nil
}

func hashBrowserBinding(raw string) string {
	hash := sha256.Sum256([]byte("as360-oidc-login-binding-v1\x00" + raw))
	return hex.EncodeToString(hash[:])
}

func (manager *Manager) admitLoginState(ctx context.Context, tx pgx.Tx, bindingHash string, bootstrap bool, now time.Time) error {
	type admissionState struct {
		key     string
		started time.Time
		count   int
		limit   int
	}
	states := make([]admissionState, 0, 2)
	keys := []struct {
		key   string
		limit int
	}{
		{key: hashToken("as360-oidc-login-global"), limit: manager.loginAdmission.GlobalLimit},
		{key: bindingHash, limit: manager.loginAdmission.BrowserLimit},
	}
	for _, rule := range keys {
		if _, err := tx.Exec(ctx, `
			INSERT INTO oidc_login_admission(rule_key, window_started, request_count, updated_at)
			VALUES ($1, $2, 0, $2) ON CONFLICT (rule_key) DO NOTHING
		`, rule.key, now); err != nil {
			return fmt.Errorf("create OIDC login admission bucket: %w", err)
		}
		var state admissionState
		state.key, state.limit = rule.key, rule.limit
		if err := tx.QueryRow(ctx, `SELECT window_started, request_count FROM oidc_login_admission WHERE rule_key = $1 FOR UPDATE`, rule.key).Scan(&state.started, &state.count); err != nil {
			return fmt.Errorf("lock OIDC login admission bucket: %w", err)
		}
		if !now.Before(state.started.Add(manager.loginAdmission.Window)) {
			state.started, state.count = now, 0
		}
		if state.count >= state.limit {
			return ErrLoginRateLimited
		}
		states = append(states, state)
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('oidc_login_states:outstanding'))`); err != nil {
		return fmt.Errorf("lock outstanding OIDC login-state cap: %w", err)
	}
	var outstanding int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM oidc_login_states WHERE expires_at > $1`, now).Scan(&outstanding); err != nil {
		return fmt.Errorf("count outstanding OIDC login states: %w", err)
	}
	if outstanding >= manager.loginAdmission.OutstandingLimit {
		return ErrLoginRateLimited
	}
	if bootstrap {
		var bootstrapOutstanding int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM oidc_login_states WHERE browser_binding_bootstrap = true AND expires_at > $1`, now).Scan(&bootstrapOutstanding); err != nil {
			return fmt.Errorf("count bootstrap OIDC login states: %w", err)
		}
		if bootstrapOutstanding >= manager.loginAdmission.BootstrapOutstandingLimit {
			return ErrLoginRateLimited
		}
	}
	for _, state := range states {
		if _, err := tx.Exec(ctx, `UPDATE oidc_login_admission SET window_started = $2, request_count = $3, updated_at = $2 WHERE rule_key = $1`, state.key, state.started, state.count+1); err != nil {
			return fmt.Errorf("update OIDC login admission bucket: %w", err)
		}
	}
	return nil
}

func (manager *Manager) CleanupLoginState(ctx context.Context, limit int) (int64, error) {
	if limit < 1 || limit > 1000 {
		return 0, fmt.Errorf("OIDC login-state cleanup limit is invalid")
	}
	command, err := manager.pool.Exec(ctx, `
		WITH expired AS (
			SELECT state_hash FROM oidc_login_states
			WHERE expires_at <= $1 ORDER BY expires_at, created_at, state_hash LIMIT $2
		)
		DELETE FROM oidc_login_states state USING expired
		WHERE state.state_hash = expired.state_hash
	`, manager.clock().UTC(), limit)
	if err != nil {
		return 0, fmt.Errorf("cleanup OIDC login states: %w", err)
	}
	return command.RowsAffected(), nil
}

func (manager *Manager) CleanupLoginAdmission(ctx context.Context, limit int) (int64, error) {
	if limit < 1 || limit > 1000 {
		return 0, fmt.Errorf("OIDC login-admission cleanup limit is invalid")
	}
	command, err := manager.pool.Exec(ctx, `
		WITH stale AS (
			SELECT rule_key FROM oidc_login_admission
			WHERE updated_at <= $1 ORDER BY updated_at, rule_key LIMIT $2
		)
		DELETE FROM oidc_login_admission admission USING stale
		WHERE admission.rule_key = stale.rule_key
		  AND admission.updated_at <= $1
	`, manager.clock().UTC().Add(-manager.loginAdmission.Window), limit)
	if err != nil {
		return 0, fmt.Errorf("cleanup OIDC login admission: %w", err)
	}
	return command.RowsAffected(), nil
}

func (manager *Manager) encryptProviderTokens(tokens identity.ProviderTokens) ([]byte, error) {
	plaintext, err := json.Marshal(tokens)
	if err != nil {
		return nil, fmt.Errorf("encode provider tokens: %w", err)
	}
	nonce, err := manager.randomBytes(manager.aead.NonceSize())
	if err != nil {
		return nil, fmt.Errorf("generate provider-token nonce: %w", err)
	}
	if len(nonce) != manager.aead.NonceSize() {
		return nil, fmt.Errorf("provider-token nonce has invalid length")
	}
	return manager.aead.Seal(append([]byte(nil), nonce...), nonce, plaintext, nil), nil
}

func (manager *Manager) decryptProviderTokens(ciphertext []byte) (identity.ProviderTokens, error) {
	nonceSize := manager.aead.NonceSize()
	if len(ciphertext) < nonceSize+manager.aead.Overhead() {
		return identity.ProviderTokens{}, fmt.Errorf("provider-token ciphertext has invalid length")
	}
	nonce := ciphertext[:nonceSize]
	plaintext, err := manager.aead.Open(nil, nonce, ciphertext[nonceSize:], nil)
	if err != nil {
		return identity.ProviderTokens{}, fmt.Errorf("decrypt provider tokens: %w", err)
	}
	defer func() {
		for index := range plaintext {
			plaintext[index] = 0
		}
	}()
	var tokens identity.ProviderTokens
	if err := json.Unmarshal(plaintext, &tokens); err != nil {
		return identity.ProviderTokens{}, fmt.Errorf("decode provider tokens: %w", err)
	}
	return tokens, nil
}

type providerLogoutTicket struct {
	IDToken   string    `json:"idToken"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (manager *Manager) encryptProviderLogoutTicket(
	idToken string,
	expiresAt time.Time,
) (string, error) {
	plaintext, err := json.Marshal(providerLogoutTicket{
		IDToken:   idToken,
		ExpiresAt: expiresAt.UTC(),
	})
	if err != nil {
		return "", fmt.Errorf("encode provider logout ticket: %w", err)
	}
	defer func() {
		for index := range plaintext {
			plaintext[index] = 0
		}
	}()
	nonce, err := manager.randomBytes(manager.aead.NonceSize())
	if err != nil {
		return "", fmt.Errorf("generate provider logout nonce: %w", err)
	}
	if len(nonce) != manager.aead.NonceSize() {
		return "", fmt.Errorf("provider logout nonce has invalid length")
	}
	sealed := manager.aead.Seal(append([]byte(nil), nonce...), nonce, plaintext, nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (manager *Manager) decryptProviderLogoutTicket(
	rawTicket string,
) (providerLogoutTicket, error) {
	ciphertext, err := base64.RawURLEncoding.DecodeString(
		strings.TrimSpace(rawTicket),
	)
	if err != nil {
		return providerLogoutTicket{}, ErrUnauthenticated
	}
	nonceSize := manager.aead.NonceSize()
	if len(ciphertext) < nonceSize+manager.aead.Overhead() {
		return providerLogoutTicket{}, ErrUnauthenticated
	}
	plaintext, err := manager.aead.Open(
		nil,
		ciphertext[:nonceSize],
		ciphertext[nonceSize:],
		nil,
	)
	if err != nil {
		return providerLogoutTicket{}, ErrUnauthenticated
	}
	defer func() {
		for index := range plaintext {
			plaintext[index] = 0
		}
	}()
	var ticket providerLogoutTicket
	if err := json.Unmarshal(plaintext, &ticket); err != nil || ticket.ExpiresAt.IsZero() {
		return providerLogoutTicket{}, ErrUnauthenticated
	}
	return ticket, nil
}

func (manager *Manager) opaqueToken(size int) (string, error) {
	bytes, err := manager.randomBytes(size)
	if err != nil {
		return "", err
	}
	if len(bytes) != size {
		return "", fmt.Errorf("random source returned %d bytes, expected %d", len(bytes), size)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hashToken(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func safeReturnTo(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") || strings.Contains(raw, "\\") {
		return "/"
	}
	return parsed.RequestURI()
}

func validRole(role identity.Role) bool {
	switch role {
	case identity.RoleInspector, identity.RoleLeadInspector, identity.RoleDepartmentManager,
		identity.RoleGeneralManager, identity.RoleFinance, identity.RoleExecutiveDirector,
		identity.RoleAuditee, identity.RoleAdmin:
		return true
	default:
		return false
	}
}

func secureRandomBytes(size int) ([]byte, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return nil, err
	}
	return value, nil
}

func randomIdentifier(prefix string) string {
	value, err := secureRandomBytes(16)
	if err != nil {
		panic(fmt.Sprintf("generate session identifier: %v", err))
	}
	return prefix + "-" + hex.EncodeToString(value)
}
