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
	command, err := store.pool.Exec(ctx, `
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
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin challenge consume: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	record, err := store.lock(ctx, tx, subject, purpose, token)
	if err != nil {
		return err
	}
	if err := store.usable(record); err != nil {
		return err
	}
	now := store.clock().UTC()
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.identity_challenges SET state = 'consumed', consumed_at = $2, updated_at = $2 WHERE token_hash = $1`, record.hash, now); err != nil {
		return fmt.Errorf("consume identity challenge: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit challenge consume: %w", err)
	}
	return nil
}

func (store *PostgresStore) RejectAttempt(ctx context.Context, subject string, purpose Purpose, token string) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin challenge rejection: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	record, err := store.lock(ctx, tx, subject, purpose, token)
	if err != nil {
		return err
	}
	if err := store.usable(record); err != nil {
		return err
	}
	attempts := record.attempts + 1
	state := "active"
	if attempts >= record.maxAttempts {
		state = "locked"
	}
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.identity_challenges SET attempt_count = $2, state = $3, updated_at = $4 WHERE token_hash = $1`, record.hash, attempts, state, store.clock().UTC()); err != nil {
		return fmt.Errorf("record rejected challenge attempt: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit challenge rejection: %w", err)
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

func (store *PostgresStore) Cleanup(ctx context.Context, at time.Time) (int, error) {
	command, err := store.pool.Exec(ctx, `DELETE FROM auth_identity.identity_challenges WHERE expires_at <= $1 OR state <> 'active'`, at.UTC())
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
