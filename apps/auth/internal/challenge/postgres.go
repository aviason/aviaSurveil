package challenge

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore keeps identity challenges durable across auth-process restarts.
// Sensitive state transitions lock the one hashed-token row before evaluating
// expiry, attempt budgets, or consumption.
type PostgresStore struct {
	pool  *pgxpool.Pool
	clock Clock
}

func NewPostgresStore(pool *pgxpool.Pool, configuration Config) (*PostgresStore, error) {
	if pool == nil {
		return nil, errors.New("PostgreSQL challenge store requires a pool")
	}
	if configuration.Clock == nil {
		configuration.Clock = time.Now
	}
	return &PostgresStore{pool: pool, clock: configuration.Clock}, nil
}

func (store *PostgresStore) Issue(ctx context.Context, subject string, purpose Purpose, ttl time.Duration, maxAttempts int) (Issued, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return Issued{}, fmt.Errorf("begin identity challenge issue: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	issued, err := store.IssueTx(ctx, tx, subject, purpose, ttl, maxAttempts)
	if err != nil {
		return Issued{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Issued{}, fmt.Errorf("commit identity challenge issue: %w", err)
	}
	return issued, nil
}

func (store *PostgresStore) IssueTx(ctx context.Context, tx pgx.Tx, subject string, purpose Purpose, ttl time.Duration, maxAttempts int) (Issued, error) {
	if tx == nil {
		return Issued{}, ErrInvalidChallenge
	}
	if strings.TrimSpace(subject) == "" || !validPurpose(purpose) || ttl <= 0 || ttl > 72*time.Hour || maxAttempts < 1 || maxAttempts > 10 {
		return Issued{}, ErrInvalidChallenge
	}
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return Issued{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	hash := hashToken(token)
	now := store.clock().UTC()
	expires := now.Add(ttl)
	command, err := tx.Exec(ctx, `
		INSERT INTO auth_identity.identity_challenges(
			token_hash, subject_id, purpose, state, attempt_count, max_attempts,
			expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, 'active', 0, $4, $5, $6, $6)`, hash[:], subject, purpose, maxAttempts, expires, now)
	if err != nil {
		if isForeignKeyViolation(err) {
			return Issued{}, ErrInvalidChallenge
		}
		return Issued{}, fmt.Errorf("store identity challenge: %w", err)
	}
	if command.RowsAffected() != 1 {
		return Issued{}, ErrInvalidChallenge
	}
	return Issued{Subject: subject, Purpose: purpose, Token: token, Expires: expires}, nil
}

func (store *PostgresStore) Consume(ctx context.Context, subject string, purpose Purpose, token string) error {
	return store.WithMutation(ctx, subject, purpose, token, func(ctx context.Context, _ pgx.Tx, handle MutationHandle) error {
		return handle.Consume(ctx)
	})
}

func (store *PostgresStore) RejectAttempt(ctx context.Context, subject string, purpose Purpose, token string) error {
	return store.WithMutation(ctx, subject, purpose, token, func(ctx context.Context, _ pgx.Tx, handle MutationHandle) error {
		return handle.RejectAttempt(ctx)
	})
}

type Validation struct {
	Subject  string
	Purpose  Purpose
	Expires  time.Time
	Attempts int
	Maximum  int
}

type MutationHandle struct {
	store    *PostgresStore
	tx       pgx.Tx
	hash     []byte
	state    string
	attempts int
	maximum  int
}

// Prevalidate is a cheap, read-only check intended to run before password
// policy and Argon2 work. It never authorizes a mutation.
func (store *PostgresStore) Prevalidate(ctx context.Context, subject string, purpose Purpose, token string) (Validation, error) {
	if strings.TrimSpace(subject) == "" || !validPurpose(purpose) || strings.TrimSpace(token) == "" {
		return Validation{}, ErrInvalidChallenge
	}
	hash := hashToken(token)
	var validation Validation
	var validationState string
	validation.Subject, validation.Purpose = subject, purpose
	err := store.pool.QueryRow(ctx, `SELECT expires_at, attempt_count, max_attempts, state FROM auth_identity.identity_challenges WHERE token_hash = $1 AND subject_id = $2 AND purpose = $3`, hash[:], subject, purpose).Scan(&validation.Expires, &validation.Attempts, &validation.Maximum, &validationState)
	if errors.Is(err, pgx.ErrNoRows) {
		return Validation{}, ErrInvalidChallenge
	}
	if err != nil {
		return Validation{}, fmt.Errorf("prevalidate identity challenge: %w", err)
	}
	if err := store.usable(durableRecord{state: validationState, attempts: validation.Attempts, maxAttempts: validation.Maximum, expires: validation.Expires}); err != nil {
		return Validation{}, err
	}
	return validation, nil
}

// WithMutation provides the only transaction-bound challenge handle. The
// callback owns the protected mutation and may commit only after final
// revalidation through this package.
func (store *PostgresStore) WithMutation(ctx context.Context, subject string, purpose Purpose, token string, mutation func(context.Context, pgx.Tx, MutationHandle) error) error {
	if mutation == nil {
		return ErrInvalidChallenge
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin challenge mutation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	record, err := store.lock(ctx, tx, subject, purpose, token)
	if err != nil {
		return err
	}
	if err := store.usable(record); err != nil {
		return err
	}
	handle := MutationHandle{store: store, tx: tx, hash: append([]byte(nil), record.hash...), state: record.state, attempts: record.attempts, maximum: record.maxAttempts}
	mutationErr := mutation(ctx, tx, handle)
	if mutationErr != nil && !errors.Is(mutationErr, ErrInvalidChallenge) && !errors.Is(mutationErr, ErrChallengeLocked) {
		return mutationErr
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit challenge mutation: %w", err)
	}
	if mutationErr != nil {
		return mutationErr
	}
	return nil
}

func (handle MutationHandle) Consume(ctx context.Context) error {
	if handle.store == nil || handle.tx == nil || handle.state != "active" {
		return ErrInvalidChallenge
	}
	now := handle.store.clock().UTC()
	if _, err := handle.tx.Exec(ctx, `UPDATE auth_identity.identity_challenges SET state = 'consumed', consumed_at = $2, updated_at = $2 WHERE token_hash = $1 AND state = 'active'`, handle.hash, now); err != nil {
		return fmt.Errorf("consume identity challenge: %w", err)
	}
	return nil
}

func (handle MutationHandle) RejectAttempt(ctx context.Context) error {
	if handle.store == nil || handle.tx == nil || handle.state != "active" {
		return ErrInvalidChallenge
	}
	attempts := handle.attempts + 1
	state := "active"
	if attempts >= handle.maximum {
		state = "locked"
	}
	if _, err := handle.tx.Exec(ctx, `UPDATE auth_identity.identity_challenges SET attempt_count = $2, state = $3, updated_at = $4 WHERE token_hash = $1 AND state = 'active'`, handle.hash, attempts, state, handle.store.clock().UTC()); err != nil {
		return fmt.Errorf("record rejected challenge attempt: %w", err)
	}
	if state == "locked" {
		return ErrChallengeLocked
	}
	return ErrInvalidChallenge
}

func (store *PostgresStore) Invalidate(ctx context.Context, subject string, purpose Purpose) (int, error) {
	if strings.TrimSpace(subject) == "" || !validPurpose(purpose) {
		return 0, ErrInvalidChallenge
	}
	command, err := store.pool.Exec(ctx, `UPDATE auth_identity.identity_challenges SET state = 'invalidated', invalidated_at = $3, updated_at = $3 WHERE subject_id = $1 AND purpose = $2 AND state = 'active'`, subject, purpose, store.clock().UTC())
	if err != nil {
		return 0, fmt.Errorf("invalidate identity challenges: %w", err)
	}
	return int(command.RowsAffected()), nil
}

// ActiveTx reports whether a non-expired active challenge exists inside the
// caller's transaction. Challenge ownership stays in this package so callers
// do not need to query identity_challenges directly.
func (store *PostgresStore) ActiveTx(ctx context.Context, tx pgx.Tx, subject string, purpose Purpose, at time.Time) (bool, error) {
	if tx == nil || strings.TrimSpace(subject) == "" || !validPurpose(purpose) {
		return false, ErrInvalidChallenge
	}
	var active bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM auth_identity.identity_challenges WHERE subject_id = $1 AND purpose = $2 AND state = 'active' AND expires_at > $3)`, subject, purpose, at.UTC()).Scan(&active); err != nil {
		return false, fmt.Errorf("check active identity challenge: %w", err)
	}
	return active, nil
}

// InvalidateTx invalidates every active challenge inside a caller-owned
// transaction, including rows whose expiry has passed. Recovery coordination
// uses this immediately before issuing a replacement so the partial unique
// active-challenge index cannot retain an expired row.
func (store *PostgresStore) InvalidateTx(ctx context.Context, tx pgx.Tx, subject string, purpose Purpose, at time.Time) (int64, error) {
	if tx == nil || strings.TrimSpace(subject) == "" || !validPurpose(purpose) {
		return 0, ErrInvalidChallenge
	}
	command, err := tx.Exec(ctx, `UPDATE auth_identity.identity_challenges SET state = 'invalidated', invalidated_at = $3, updated_at = $3 WHERE subject_id = $1 AND purpose = $2 AND state = 'active'`, subject, purpose, at.UTC())
	if err != nil {
		return 0, fmt.Errorf("invalidate identity challenge: %w", err)
	}
	return command.RowsAffected(), nil
}

func (store *PostgresStore) Cleanup(ctx context.Context, at time.Time, limit int) (int, error) {
	if limit < 1 || limit > 1000 {
		return 0, ErrInvalidChallenge
	}
	command, err := store.pool.Exec(ctx, `
		WITH stale AS (
			SELECT token_hash FROM auth_identity.identity_challenges
			WHERE expires_at <= $1 OR state <> 'active'
			ORDER BY updated_at, token_hash
			LIMIT $2
		)
		DELETE FROM auth_identity.identity_challenges challenge
		USING stale
		WHERE challenge.token_hash = stale.token_hash`, at.UTC(), limit)
	if err != nil {
		return 0, fmt.Errorf("cleanup identity challenges: %w", err)
	}
	return int(command.RowsAffected()), nil
}

type durableRecord struct {
	hash        []byte
	state       string
	attempts    int
	maxAttempts int
	expires     time.Time
}

func (store *PostgresStore) lock(ctx context.Context, tx pgx.Tx, subject string, purpose Purpose, token string) (durableRecord, error) {
	if strings.TrimSpace(subject) == "" || !validPurpose(purpose) || strings.TrimSpace(token) == "" {
		return durableRecord{}, ErrInvalidChallenge
	}
	hash := hashToken(token)
	var record durableRecord
	record.hash = append([]byte(nil), hash[:]...)
	err := tx.QueryRow(ctx, `SELECT state, attempt_count, max_attempts, expires_at FROM auth_identity.identity_challenges WHERE token_hash = $1 AND subject_id = $2 AND purpose = $3 FOR UPDATE`, hash[:], subject, purpose).Scan(&record.state, &record.attempts, &record.maxAttempts, &record.expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return durableRecord{}, ErrInvalidChallenge
	}
	if err != nil {
		return durableRecord{}, fmt.Errorf("lock identity challenge: %w", err)
	}
	return record, nil
}

func (store *PostgresStore) usable(record durableRecord) error {
	switch record.state {
	case "consumed", "invalidated":
		return ErrChallengeUsed
	case "locked":
		return ErrChallengeLocked
	case "active":
	default:
		return ErrInvalidChallenge
	}
	if !store.clock().Before(record.expires) {
		return ErrChallengeExpired
	}
	if record.attempts >= record.maxAttempts {
		return ErrChallengeLocked
	}
	return nil
}

func validPurpose(purpose Purpose) bool {
	switch purpose {
	case PurposeEmailVerification, PurposePasswordReset, PurposeMFARecovery, PurposeAdminRecovery:
		return true
	default:
		return false
	}
}

func isForeignKeyViolation(err error) bool {
	var pgError *pgconn.PgError
	return errors.As(err, &pgError) && pgError.Code == "23503"
}
