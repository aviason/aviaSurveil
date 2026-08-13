package identity

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/password"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/throttle"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool           *pgxpool.Pool
	hasher         *password.Hasher
	passwordPolicy password.Policy
	limiter        Limiter
	sessionRevoker SessionRevoker
	trustedProxies []netip.Prefix
	clock          Clock
}

// SetSessionRevoker wires provider credential/session revocation after both
// durable stores have been initialized. Security-state mutations fail closed
// when this boundary reports an unavailable provider session store.
func (store *PostgresStore) SetSessionRevoker(revoker SessionRevoker) {
	store.sessionRevoker = revoker
}

func NewPostgresStore(pool *pgxpool.Pool, configuration Config) (*PostgresStore, error) {
	if pool == nil || configuration.Hasher == nil || configuration.Limiter == nil {
		return nil, errors.New("PostgreSQL identity store requires pool, hasher, and limiter")
	}
	if configuration.PasswordPolicy.MinBytes < 1 || configuration.PasswordPolicy.MaxBytes < configuration.PasswordPolicy.MinBytes {
		return nil, errors.New("PostgreSQL identity password policy is invalid")
	}
	if configuration.Clock == nil {
		configuration.Clock = time.Now
	}
	return &PostgresStore{
		pool:           pool,
		hasher:         configuration.Hasher,
		passwordPolicy: configuration.PasswordPolicy,
		limiter:        configuration.Limiter,
		sessionRevoker: configuration.SessionRevoker,
		trustedProxies: append([]netip.Prefix(nil), configuration.TrustedProxies...),
		clock:          configuration.Clock,
	}, nil
}

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type dbAccount struct {
	subjectID      string
	state          AccountState
	emailVerified  bool
	passwordHash   string
	authRevision   uint64
	failedAttempts int
	lockedUntil    time.Time
	createdAt      time.Time
	updatedAt      time.Time
	email          string
	username       string
}

func (store *PostgresStore) ProvisionInvitation(ctx context.Context, input InvitationInput) (AccountSnapshot, InvitationSnapshot, error) {
	email, err := NormalizeIdentifier(IdentifierEmail, input.Email)
	if err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, err
	}
	username := ""
	if strings.TrimSpace(input.Username) != "" {
		normalizedUsername, usernameErr := NormalizeIdentifier(IdentifierUsername, input.Username)
		if usernameErr != nil {
			return AccountSnapshot{}, InvitationSnapshot{}, usernameErr
		}
		username = normalizedUsername.Normalized
	}
	subjectID, err := NewSubjectID()
	if err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, err
	}
	invititationID, err := newPrefixedID("inv_")
	if err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, err
	}
	rawToken, tokenHash, err := newRandomTokenHash()
	if err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, err
	}
	now := store.clock()
	expiresAt := now.Add(24 * time.Hour)
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, fmt.Errorf("begin invitation transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if _, err := transaction.Exec(ctx, `
		INSERT INTO auth_identity.accounts(subject_id, state, email_verified, auth_revision, created_at, updated_at)
		VALUES ($1, 'invited', false, 1, $2, $2)`, subjectID, now); err != nil {
		if isDuplicate(err) {
			return AccountSnapshot{}, InvitationSnapshot{}, ErrDuplicateIdentifier
		}
		return AccountSnapshot{}, InvitationSnapshot{}, fmt.Errorf("insert invited account: %w", err)
	}
	for _, identifier := range []Identifier{{Type: IdentifierEmail, Normalized: email.Normalized}, {Type: IdentifierUsername, Normalized: username}} {
		if identifier.Normalized == "" {
			continue
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO auth_identity.identifiers(subject_id, identifier_type, normalized_value, created_at)
			VALUES ($1, $2, $3, $4)`, subjectID, identifier.Type, identifier.Normalized, now); err != nil {
			if isDuplicate(err) {
				return AccountSnapshot{}, InvitationSnapshot{}, ErrDuplicateIdentifier
			}
			return AccountSnapshot{}, InvitationSnapshot{}, fmt.Errorf("insert canonical identifier: %w", err)
		}
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO auth_identity.invitations(invitation_id, subject_id, token_hash, state, issued_at, expires_at)
		VALUES ($1, $2, $3, 'issued', $4, $5)`, invititationID, subjectID, tokenHash[:], now, expiresAt); err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, fmt.Errorf("insert invitation: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, fmt.Errorf("commit invitation transaction: %w", err)
	}
	return AccountSnapshot{SubjectID: subjectID, Email: email.Normalized, Username: username, State: AccountInvited, AuthRevision: 1, CreatedAt: now, UpdatedAt: now}, InvitationSnapshot{SubjectID: subjectID, Token: rawToken, IssuedAt: now, ExpiresAt: expiresAt}, nil
}

func (store *PostgresStore) RegisterPublic(context.Context, InvitationInput) error {
	return ErrPublicRegistrationDisabled
}

func (store *PostgresStore) SetEmailVerified(ctx context.Context, subjectID string, expectedRevision uint64) (AccountSnapshot, error) {
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return AccountSnapshot{}, fmt.Errorf("begin email verification: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	current, err := loadAccount(ctx, transaction, subjectID, true)
	if err != nil {
		return AccountSnapshot{}, mapAccountReadError(err)
	}
	if current.authRevision != expectedRevision {
		return AccountSnapshot{}, ErrRevisionConflict
	}
	if current.state == AccountDeleted || current.state == AccountDeletionPending {
		return AccountSnapshot{}, ErrInvalidTransition
	}
	if _, err := transaction.Exec(ctx, `UPDATE auth_identity.accounts SET email_verified = true, auth_revision = auth_revision + 1, updated_at = $2 WHERE subject_id = $1`, subjectID, store.clock()); err != nil {
		return AccountSnapshot{}, fmt.Errorf("verify email: %w", err)
	}
	updated, err := loadAccount(ctx, transaction, subjectID, true)
	if err != nil {
		return AccountSnapshot{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return AccountSnapshot{}, fmt.Errorf("commit email verification: %w", err)
	}
	snapshot := updated.snapshot()
	if err := store.revokeSessions(ctx, subjectID); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

func (store *PostgresStore) Activate(ctx context.Context, subjectID string, expectedRevision uint64, newPassword []byte) (AccountSnapshot, error) {
	current, err := loadAccount(ctx, store.pool, subjectID, false)
	if err != nil {
		return AccountSnapshot{}, mapAccountReadError(err)
	}
	if current.authRevision != expectedRevision {
		return AccountSnapshot{}, ErrRevisionConflict
	}
	if current.state != AccountInvited || !current.emailVerified {
		return AccountSnapshot{}, ErrEmailNotVerified
	}
	history, err := loadPasswordHistory(ctx, store.pool, subjectID)
	if err != nil {
		return AccountSnapshot{}, err
	}
	if err := store.passwordPolicy.Validate(newPassword, store.hasher, current.passwordHash, history); err != nil {
		return AccountSnapshot{}, err
	}
	hash, err := store.hasher.Hash(newPassword)
	if err != nil {
		return AccountSnapshot{}, err
	}
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return AccountSnapshot{}, fmt.Errorf("begin account activation: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	locked, err := loadAccount(ctx, transaction, subjectID, true)
	if err != nil {
		return AccountSnapshot{}, mapAccountReadError(err)
	}
	if locked.authRevision != expectedRevision || locked.state != AccountInvited || !locked.emailVerified {
		return AccountSnapshot{}, ErrRevisionConflict
	}
	if _, err := transaction.Exec(ctx, `UPDATE auth_identity.accounts SET state = 'active', password_hash = $2, auth_revision = auth_revision + 1, updated_at = $3 WHERE subject_id = $1 AND auth_revision = $4`, subjectID, hash, store.clock(), expectedRevision); err != nil {
		return AccountSnapshot{}, fmt.Errorf("activate account: %w", err)
	}
	updated, err := loadAccount(ctx, transaction, subjectID, true)
	if err != nil {
		return AccountSnapshot{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return AccountSnapshot{}, fmt.Errorf("commit account activation: %w", err)
	}
	snapshot := updated.snapshot()
	if err := store.revokeSessions(ctx, subjectID); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

func (store *PostgresStore) Authenticate(ctx context.Context, request AuthenticationRequest) (AuthenticationResult, error) {
	identifier, identifierErr := DetectIdentifier(request.Identifier)
	clientIP, ipErr := throttle.ResolveClientIP(request.Source, store.trustedProxies)
	if ipErr != nil {
		return AuthenticationResult{}, ErrAuthenticationUnavailable
	}
	keys := []string{throttle.Key("ip", clientIP.String()), throttle.Key("identifier", request.Identifier)}
	if strings.TrimSpace(request.DeviceKey) != "" {
		keys = append(keys, throttle.Key("device", request.DeviceKey))
	}
	if err := store.limiter.Allow(ctx, keys...); err != nil {
		if errors.Is(err, throttle.ErrRateLimited) {
			return AuthenticationResult{}, ErrAuthenticationRateLimited
		}
		return AuthenticationResult{}, ErrAuthenticationUnavailable
	}
	var current dbAccount
	lookupErr := ErrAccountNotFound
	if identifierErr == nil {
		current, lookupErr = loadAccountByIdentifier(ctx, store.pool, identifier.Normalized)
	}
	encodedHash := store.hasher.DummyHash()
	if lookupErr == nil {
		encodedHash = current.passwordHash
	}
	verified, verifyErr := store.hasher.Verify(encodedHash, request.Password)
	if verifyErr != nil {
		return AuthenticationResult{}, ErrAuthenticationFailed
	}
	if lookupErr != nil {
		return AuthenticationResult{}, ErrAuthenticationFailed
	}
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return AuthenticationResult{}, ErrAuthenticationUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	locked, err := loadAccount(ctx, transaction, current.subjectID, true)
	if err != nil {
		return AuthenticationResult{}, ErrAuthenticationUnavailable
	}
	if locked.authRevision != current.authRevision || locked.passwordHash != current.passwordHash || !verified || !canLoginAt(locked, store.clock()) {
		willLock := locked.state == AccountActive && locked.failedAttempts+1 >= 5
		if err := recordPostgresFailure(ctx, transaction, locked, store.clock()); err != nil {
			return AuthenticationResult{}, ErrAuthenticationUnavailable
		}
		if err := transaction.Commit(ctx); err != nil {
			return AuthenticationResult{}, ErrAuthenticationUnavailable
		}
		if willLock && store.sessionRevoker != nil {
			if err := store.sessionRevoker.RevokeAllSessions(ctx, locked.subjectID); err != nil {
				return AuthenticationResult{}, ErrAuthenticationUnavailable
			}
		}
		return AuthenticationResult{}, ErrAuthenticationFailed
	}
	if locked.state == AccountLocked {
		if _, err := transaction.Exec(ctx, `UPDATE auth_identity.accounts SET state = 'active', auth_revision = auth_revision + 1, failed_login_count = 0, locked_until = NULL, updated_at = $2 WHERE subject_id = $1`, locked.subjectID, store.clock()); err != nil {
			return AuthenticationResult{}, ErrAuthenticationUnavailable
		}
	} else if _, err := transaction.Exec(ctx, `UPDATE auth_identity.accounts SET failed_login_count = 0, locked_until = NULL, updated_at = $2 WHERE subject_id = $1`, locked.subjectID, store.clock()); err != nil {
		return AuthenticationResult{}, ErrAuthenticationUnavailable
	}
	updated, err := loadAccount(ctx, transaction, locked.subjectID, true)
	if err != nil {
		return AuthenticationResult{}, ErrAuthenticationUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return AuthenticationResult{}, ErrAuthenticationUnavailable
	}
	return AuthenticationResult{Account: updated.snapshot()}, nil
}

func (store *PostgresStore) ChangePassword(ctx context.Context, subjectID string, expectedRevision uint64, currentPassword, newPassword []byte) (AccountSnapshot, error) {
	current, err := loadAccount(ctx, store.pool, subjectID, false)
	if err != nil {
		return AccountSnapshot{}, mapAccountReadError(err)
	}
	if current.authRevision != expectedRevision || current.state != AccountActive {
		return AccountSnapshot{}, ErrRevisionConflict
	}
	validCurrent, err := store.hasher.Verify(current.passwordHash, currentPassword)
	if err != nil || !validCurrent {
		return AccountSnapshot{}, ErrAuthenticationFailed
	}
	history, err := loadPasswordHistory(ctx, store.pool, subjectID)
	if err != nil {
		return AccountSnapshot{}, err
	}
	if err := store.passwordPolicy.Validate(newPassword, store.hasher, current.passwordHash, history); err != nil {
		return AccountSnapshot{}, err
	}
	newHash, err := store.hasher.Hash(newPassword)
	if err != nil {
		return AccountSnapshot{}, err
	}
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return AccountSnapshot{}, fmt.Errorf("begin password change: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	locked, err := loadAccount(ctx, transaction, subjectID, true)
	if err != nil {
		return AccountSnapshot{}, mapAccountReadError(err)
	}
	if locked.authRevision != expectedRevision || locked.passwordHash != current.passwordHash {
		return AccountSnapshot{}, ErrRevisionConflict
	}
	if _, err := transaction.Exec(ctx, `INSERT INTO auth_identity.password_history(subject_id, history_revision, password_hash, created_at) VALUES ($1, $2, $3, $4)`, subjectID, expectedRevision, current.passwordHash, store.clock()); err != nil {
		return AccountSnapshot{}, fmt.Errorf("record password history: %w", err)
	}
	if _, err := transaction.Exec(ctx, `UPDATE auth_identity.accounts SET password_hash = $2, auth_revision = auth_revision + 1, failed_login_count = 0, locked_until = NULL, updated_at = $3 WHERE subject_id = $1 AND auth_revision = $4`, subjectID, newHash, store.clock(), expectedRevision); err != nil {
		return AccountSnapshot{}, fmt.Errorf("update password: %w", err)
	}
	updated, err := loadAccount(ctx, transaction, subjectID, true)
	if err != nil {
		return AccountSnapshot{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return AccountSnapshot{}, fmt.Errorf("commit password change: %w", err)
	}
	snapshot := updated.snapshot()
	if err := store.revokeSessions(ctx, subjectID); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

// LookupEmail is deliberately narrow for provider-owned recovery initiation.
// The caller must still return an enumeration-resistant response when this
// lookup reports no account.
func (store *PostgresStore) LookupEmail(ctx context.Context, email string) (AccountSnapshot, error) {
	identifier, err := NormalizeIdentifier(IdentifierEmail, email)
	if err != nil {
		return AccountSnapshot{}, ErrAccountNotFound
	}
	current, err := loadAccountByIdentifier(ctx, store.pool, identifier.Normalized)
	if err != nil {
		return AccountSnapshot{}, mapAccountReadError(err)
	}
	return current.snapshot(), nil
}

// ResetPassword changes an active account's password after an independently
// consumed, subject-bound recovery challenge. It preserves password-history
// policy and revokes existing provider sessions just like a password change.
func (store *PostgresStore) ResetPassword(ctx context.Context, subjectID string, expectedRevision uint64, newPassword []byte) (AccountSnapshot, error) {
	current, err := loadAccount(ctx, store.pool, subjectID, false)
	if err != nil {
		return AccountSnapshot{}, mapAccountReadError(err)
	}
	if current.authRevision != expectedRevision || current.state != AccountActive {
		return AccountSnapshot{}, ErrRevisionConflict
	}
	history, err := loadPasswordHistory(ctx, store.pool, subjectID)
	if err != nil {
		return AccountSnapshot{}, err
	}
	if err := store.passwordPolicy.Validate(newPassword, store.hasher, current.passwordHash, history); err != nil {
		return AccountSnapshot{}, err
	}
	newHash, err := store.hasher.Hash(newPassword)
	if err != nil {
		return AccountSnapshot{}, err
	}
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return AccountSnapshot{}, fmt.Errorf("begin password reset: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	locked, err := loadAccount(ctx, transaction, subjectID, true)
	if err != nil {
		return AccountSnapshot{}, mapAccountReadError(err)
	}
	if locked.authRevision != expectedRevision || locked.passwordHash != current.passwordHash || locked.state != AccountActive {
		return AccountSnapshot{}, ErrRevisionConflict
	}
	if _, err := transaction.Exec(ctx, `INSERT INTO auth_identity.password_history(subject_id, history_revision, password_hash, created_at) VALUES ($1, $2, $3, $4)`, subjectID, expectedRevision, current.passwordHash, store.clock()); err != nil {
		return AccountSnapshot{}, fmt.Errorf("record password reset history: %w", err)
	}
	if _, err := transaction.Exec(ctx, `UPDATE auth_identity.accounts SET password_hash=$2, auth_revision=auth_revision+1, failed_login_count=0, locked_until=NULL, updated_at=$3 WHERE subject_id=$1 AND auth_revision=$4`, subjectID, newHash, store.clock(), expectedRevision); err != nil {
		return AccountSnapshot{}, fmt.Errorf("reset password: %w", err)
	}
	updated, err := loadAccount(ctx, transaction, subjectID, true)
	if err != nil {
		return AccountSnapshot{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return AccountSnapshot{}, fmt.Errorf("commit password reset: %w", err)
	}
	snapshot := updated.snapshot()
	if err := store.revokeSessions(ctx, subjectID); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

func (store *PostgresStore) Transition(ctx context.Context, subjectID string, expectedRevision uint64, target AccountState) (AccountSnapshot, error) {
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return AccountSnapshot{}, fmt.Errorf("begin account transition: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	current, err := loadAccount(ctx, transaction, subjectID, true)
	if err != nil {
		return AccountSnapshot{}, mapAccountReadError(err)
	}
	if current.authRevision != expectedRevision {
		return AccountSnapshot{}, ErrRevisionConflict
	}
	if !allowedTransition(current.state, target) || (target == AccountActive && (!current.emailVerified || current.passwordHash == "")) {
		return AccountSnapshot{}, ErrInvalidTransition
	}
	if _, err := transaction.Exec(ctx, `UPDATE auth_identity.accounts SET state = $2, auth_revision = auth_revision + 1, updated_at = $3 WHERE subject_id = $1 AND auth_revision = $4`, subjectID, target, store.clock(), expectedRevision); err != nil {
		return AccountSnapshot{}, fmt.Errorf("transition account state: %w", err)
	}
	updated, err := loadAccount(ctx, transaction, subjectID, true)
	if err != nil {
		return AccountSnapshot{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return AccountSnapshot{}, fmt.Errorf("commit account transition: %w", err)
	}
	snapshot := updated.snapshot()
	if err := store.revokeSessions(ctx, subjectID); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

func (store *PostgresStore) revokeSessions(ctx context.Context, subjectID string) error {
	if store.sessionRevoker == nil {
		return nil
	}
	if err := store.sessionRevoker.RevokeAllSessions(ctx, subjectID); err != nil {
		return fmt.Errorf("%w: %v", ErrSessionRevocationUnavailable, err)
	}
	return nil
}

func (store *PostgresStore) Snapshot(ctx context.Context, subjectID string) (AccountSnapshot, error) {
	record, err := loadAccount(ctx, store.pool, subjectID, false)
	if err != nil {
		return AccountSnapshot{}, mapAccountReadError(err)
	}
	return record.snapshot(), nil
}

func loadAccount(ctx context.Context, queryer rowQuerier, subjectID string, forUpdate bool) (dbAccount, error) {
	lock := ""
	if forUpdate {
		lock = " FOR UPDATE"
	}
	var record dbAccount
	var state string
	var passwordHash *string
	var lockedUntil *time.Time
	if err := queryer.QueryRow(ctx, `SELECT subject_id, state, email_verified, password_hash, auth_revision, failed_login_count, locked_until, created_at, updated_at FROM auth_identity.accounts WHERE subject_id = $1`+lock, subjectID).Scan(&record.subjectID, &state, &record.emailVerified, &passwordHash, &record.authRevision, &record.failedAttempts, &lockedUntil, &record.createdAt, &record.updatedAt); err != nil {
		return dbAccount{}, err
	}
	record.state = AccountState(state)
	if passwordHash != nil {
		record.passwordHash = *passwordHash
	}
	if lockedUntil != nil {
		record.lockedUntil = lockedUntil.UTC()
	}
	rows, err := queryer.Query(ctx, `SELECT identifier_type, normalized_value FROM auth_identity.identifiers WHERE subject_id = $1`, subjectID)
	if err != nil {
		return dbAccount{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind, value string
		if err := rows.Scan(&kind, &value); err != nil {
			return dbAccount{}, err
		}
		switch IdentifierType(kind) {
		case IdentifierEmail:
			record.email = value
		case IdentifierUsername:
			record.username = value
		}
	}
	if err := rows.Err(); err != nil {
		return dbAccount{}, err
	}
	return record, nil
}

func loadAccountByIdentifier(ctx context.Context, queryer rowQuerier, normalized string) (dbAccount, error) {
	var subjectID string
	if err := queryer.QueryRow(ctx, `SELECT subject_id FROM auth_identity.identifiers WHERE normalized_value = $1`, normalized).Scan(&subjectID); err != nil {
		return dbAccount{}, err
	}
	return loadAccount(ctx, queryer, subjectID, false)
}

func loadPasswordHistory(ctx context.Context, queryer rowQuerier, subjectID string) ([]string, error) {
	rows, err := queryer.Query(ctx, `SELECT password_hash FROM auth_identity.password_history WHERE subject_id = $1 ORDER BY history_revision DESC LIMIT 5`, subjectID)
	if err != nil {
		return nil, fmt.Errorf("load password history: %w", err)
	}
	defer rows.Close()
	var history []string
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return nil, err
		}
		history = append(history, hash)
	}
	return history, rows.Err()
}

func recordPostgresFailure(ctx context.Context, transaction pgx.Tx, record dbAccount, now time.Time) error {
	nextCount := record.failedAttempts + 1
	if record.state == AccountActive && nextCount >= 5 {
		_, err := transaction.Exec(ctx, `UPDATE auth_identity.accounts SET state = 'locked', failed_login_count = $2, locked_until = $3, auth_revision = auth_revision + 1, updated_at = $4 WHERE subject_id = $1`, record.subjectID, nextCount, now.Add(15*time.Minute), now)
		return err
	}
	_, err := transaction.Exec(ctx, `UPDATE auth_identity.accounts SET failed_login_count = $2, updated_at = $3 WHERE subject_id = $1`, record.subjectID, nextCount, now)
	return err
}

func canLoginAt(record dbAccount, now time.Time) bool {
	if !record.emailVerified || (record.state != AccountActive && record.state != AccountLocked) {
		return false
	}
	return record.lockedUntil.IsZero() || !record.lockedUntil.After(now)
}

func (record dbAccount) snapshot() AccountSnapshot {
	return AccountSnapshot{SubjectID: record.subjectID, Email: record.email, Username: record.username, State: record.state, EmailVerified: record.emailVerified, AuthRevision: record.authRevision, FailedAttempts: record.failedAttempts, LockedUntil: record.lockedUntil, CreatedAt: record.createdAt, UpdatedAt: record.updatedAt}
}

func mapAccountReadError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAccountNotFound
	}
	return err
}

func isDuplicate(err error) bool {
	var pgError *pgconn.PgError
	return errors.As(err, &pgError) && pgError.Code == "23505"
}

func newPrefixedID(prefix string) (string, error) {
	subject, err := NewSubjectID()
	if err != nil {
		return "", err
	}
	return prefix + strings.TrimPrefix(subject, "usr_"), nil
}
