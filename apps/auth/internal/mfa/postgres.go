package mfa

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore persists the privileged MFA state separately from the normal
// application database. Each mutation locks the factor row so a TOTP counter
// or recovery code cannot be consumed concurrently more than once.
type PostgresStore struct {
	pool                *pgxpool.Pool
	key                 []byte
	sessionRevoker      SessionRevoker
	clock               Clock
	period              time.Duration
	window              int
	digits              int
	maxRecoveryFailures int
}

func NewPostgresStore(pool *pgxpool.Pool, configuration Config) (*PostgresStore, error) {
	if pool == nil {
		return nil, errors.New("PostgreSQL MFA store requires a pool")
	}
	if err := validateConfig(&configuration); err != nil {
		return nil, err
	}
	return &PostgresStore{
		pool: pool, key: append([]byte(nil), configuration.EncryptionKey...), sessionRevoker: configuration.SessionRevoker, clock: configuration.Clock,
		period: configuration.Period, window: configuration.Window, digits: configuration.Digits,
		maxRecoveryFailures: configuration.MaxRecoveryFailures,
	}, nil
}

func (store *PostgresStore) SetSessionRevoker(revoker SessionRevoker) {
	store.sessionRevoker = revoker
}

func (store *PostgresStore) Enroll(ctx context.Context, subjectID, issuer, accountLabel string) (Enrollment, error) {
	if strings.TrimSpace(subjectID) == "" || strings.TrimSpace(issuer) == "" || strings.TrimSpace(accountLabel) == "" {
		return Enrollment{}, ErrInvalidCode
	}
	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		return Enrollment{}, err
	}
	ciphertext, err := store.encrypt(subjectID, secretBytes)
	if err != nil {
		return Enrollment{}, err
	}
	now := store.clock().UTC()
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return Enrollment{}, fmt.Errorf("begin MFA enrollment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var existingEnabled bool
	err = tx.QueryRow(ctx, `SELECT enabled FROM auth_identity.mfa_factors WHERE subject_id = $1 FOR UPDATE`, subjectID).Scan(&existingEnabled)
	if errors.Is(err, pgx.ErrNoRows) {
		var lockedSubject string
		if err := tx.QueryRow(ctx, `SELECT subject_id FROM auth_identity.accounts WHERE subject_id = $1 FOR KEY SHARE`, subjectID).Scan(&lockedSubject); errors.Is(err, pgx.ErrNoRows) {
			return Enrollment{}, ErrFactorNotFound
		} else if err != nil {
			return Enrollment{}, fmt.Errorf("verify MFA subject: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO auth_identity.mfa_factors(subject_id, secret_ciphertext, enabled, last_used_counter, recovery_failures, created_at, updated_at) VALUES ($1, $2, false, -1, 0, $3, $3)`, subjectID, ciphertext, now); err != nil {
			return Enrollment{}, fmt.Errorf("insert MFA factor: %w", err)
		}
	} else if err != nil {
		return Enrollment{}, fmt.Errorf("lock MFA factor: %w", err)
	} else if existingEnabled {
		return Enrollment{}, errors.New("MFA factor already enabled")
	} else {
		if _, err := tx.Exec(ctx, `UPDATE auth_identity.mfa_factors SET secret_ciphertext = $2, enabled = false, last_used_counter = -1, recovery_failures = 0, updated_at = $3 WHERE subject_id = $1`, subjectID, ciphertext, now); err != nil {
			return Enrollment{}, fmt.Errorf("replace pending MFA factor: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM auth_identity.mfa_recovery_codes WHERE subject_id = $1`, subjectID); err != nil {
			return Enrollment{}, fmt.Errorf("clear pending MFA recovery codes: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Enrollment{}, fmt.Errorf("commit MFA enrollment: %w", err)
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)
	return Enrollment{SubjectID: subjectID, Secret: secret, OTPAuthURI: "otpauth://totp/" + urlEscape(accountLabel) + "?secret=" + secret + "&issuer=" + urlEscape(issuer) + "&algorithm=SHA1&digits=" + fmt.Sprint(store.digits) + "&period=" + fmt.Sprint(int64(store.period/time.Second))}, nil
}

func (store *PostgresStore) ConfirmEnrollment(ctx context.Context, subjectID, code string) error {
	return store.verify(ctx, subjectID, code, true)
}

func (store *PostgresStore) Verify(ctx context.Context, subjectID, code string) error {
	return store.verify(ctx, subjectID, code, false)
}

func (store *PostgresStore) verify(ctx context.Context, subjectID, code string, confirming bool) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin MFA verification: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var ciphertext []byte
	var enabled bool
	var lastCounter int64
	err = tx.QueryRow(ctx, `SELECT secret_ciphertext, enabled, last_used_counter FROM auth_identity.mfa_factors WHERE subject_id = $1 FOR UPDATE`, subjectID).Scan(&ciphertext, &enabled, &lastCounter)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrFactorNotFound
	}
	if err != nil {
		return fmt.Errorf("lock MFA factor: %w", err)
	}
	if !confirming && !enabled {
		return ErrFactorDisabled
	}
	secret, err := store.decrypt(subjectID, ciphertext)
	if err != nil {
		return ErrMFAUnavailable
	}
	counter, ok := store.matchCounter(secret, code, store.counter(store.clock()))
	if !ok {
		return ErrInvalidCode
	}
	if !confirming && counter <= lastCounter {
		return ErrCodeReplayed
	}
	now := store.clock().UTC()
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.mfa_factors SET enabled = true, last_used_counter = $2, updated_at = $3 WHERE subject_id = $1`, subjectID, counter, now); err != nil {
		return fmt.Errorf("advance MFA counter: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit MFA verification: %w", err)
	}
	return nil
}

func (store *PostgresStore) GenerateRecoveryCodes(ctx context.Context, subjectID string, count int) ([]string, error) {
	if count < 1 || count > 20 {
		return nil, errors.New("recovery code count is invalid")
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin MFA recovery generation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var enabled bool
	err = tx.QueryRow(ctx, `SELECT enabled FROM auth_identity.mfa_factors WHERE subject_id = $1 FOR UPDATE`, subjectID).Scan(&enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFactorDisabled
	}
	if err != nil {
		return nil, fmt.Errorf("lock MFA factor: %w", err)
	}
	if !enabled {
		return nil, ErrFactorDisabled
	}
	if _, err := tx.Exec(ctx, `DELETE FROM auth_identity.mfa_recovery_codes WHERE subject_id = $1`, subjectID); err != nil {
		return nil, fmt.Errorf("replace MFA recovery codes: %w", err)
	}
	codes := make([]string, 0, count)
	for len(codes) < count {
		raw := make([]byte, 10)
		if _, err := rand.Read(raw); err != nil {
			return nil, err
		}
		code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
		hash := hashRecoveryCode(code)
		command, err := tx.Exec(ctx, `INSERT INTO auth_identity.mfa_recovery_codes(subject_id, code_hash, created_at) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, subjectID, hash[:], store.clock().UTC())
		if err != nil {
			return nil, fmt.Errorf("store MFA recovery code: %w", err)
		}
		if command.RowsAffected() == 1 {
			codes = append(codes, code)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.mfa_factors SET recovery_failures = 0, updated_at = $2 WHERE subject_id = $1`, subjectID, store.clock().UTC()); err != nil {
		return nil, fmt.Errorf("reset MFA recovery failures: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit MFA recovery generation: %w", err)
	}
	return codes, nil
}

func (store *PostgresStore) ConsumeRecoveryCode(ctx context.Context, subjectID, code string) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin MFA recovery consume: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var enabled bool
	var failures int
	err = tx.QueryRow(ctx, `SELECT enabled, recovery_failures FROM auth_identity.mfa_factors WHERE subject_id = $1 FOR UPDATE`, subjectID).Scan(&enabled, &failures)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrFactorDisabled
	}
	if err != nil {
		return fmt.Errorf("lock MFA factor: %w", err)
	}
	if !enabled {
		return ErrFactorDisabled
	}
	if failures >= store.maxRecoveryFailures {
		return ErrRecoveryLocked
	}
	hash := hashRecoveryCode(code)
	command, err := tx.Exec(ctx, `DELETE FROM auth_identity.mfa_recovery_codes WHERE subject_id = $1 AND code_hash = $2`, subjectID, hash[:])
	if err != nil {
		return fmt.Errorf("consume MFA recovery code: %w", err)
	}
	if command.RowsAffected() != 1 {
		failures++
		if _, err := tx.Exec(ctx, `UPDATE auth_identity.mfa_factors SET recovery_failures = $2, updated_at = $3 WHERE subject_id = $1`, subjectID, failures, store.clock().UTC()); err != nil {
			return fmt.Errorf("record MFA recovery failure: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit MFA recovery failure: %w", err)
		}
		if failures >= store.maxRecoveryFailures {
			return ErrRecoveryLocked
		}
		return ErrRecoveryInvalid
	}
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.mfa_factors SET recovery_failures = 0, updated_at = $2 WHERE subject_id = $1`, subjectID, store.clock().UTC()); err != nil {
		return fmt.Errorf("reset MFA recovery failures: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit MFA recovery consume: %w", err)
	}
	return nil
}

func (store *PostgresStore) Reset(ctx context.Context, subjectID string) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin MFA reset: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	command, err := tx.Exec(ctx, `DELETE FROM auth_identity.mfa_factors WHERE subject_id = $1`, subjectID)
	if err != nil {
		return fmt.Errorf("reset MFA factor: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrFactorNotFound
	}
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.accounts SET auth_revision = auth_revision + 1, updated_at = $2 WHERE subject_id = $1`, subjectID, store.clock().UTC()); err != nil {
		return fmt.Errorf("advance MFA reset auth revision: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit MFA reset: %w", err)
	}
	if store.sessionRevoker != nil {
		if err := store.sessionRevoker.RevokeAllSessions(ctx, subjectID); err != nil {
			return fmt.Errorf("revoke provider sessions after MFA reset: %w", err)
		}
	}
	return nil
}

// ResetAtAuthRevision makes the provider-admin MFA mutation conditional on
// the account security revision observed by the caller.
func (store *PostgresStore) ResetAtAuthRevision(ctx context.Context, subjectID string, expected uint64) (uint64, error) {
	if expected < 1 {
		return 0, ErrRevisionConflict
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin revisioned MFA reset: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var current uint64
	if err := tx.QueryRow(ctx, `SELECT auth_revision FROM auth_identity.accounts WHERE subject_id = $1 FOR UPDATE`, subjectID).Scan(&current); errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrFactorNotFound
	} else if err != nil {
		return 0, fmt.Errorf("lock account for revisioned MFA reset: %w", err)
	}
	if current != expected {
		return 0, ErrRevisionConflict
	}
	command, err := tx.Exec(ctx, `DELETE FROM auth_identity.mfa_factors WHERE subject_id = $1`, subjectID)
	if err != nil {
		return 0, fmt.Errorf("reset revisioned MFA factor: %w", err)
	}
	if command.RowsAffected() != 1 {
		return 0, ErrFactorNotFound
	}
	resulting := expected + 1
	now := store.clock().UTC()
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.accounts SET auth_revision = $2, updated_at = $3 WHERE subject_id = $1 AND auth_revision = $4`, subjectID, resulting, now, expected); err != nil {
		return 0, fmt.Errorf("advance revisioned MFA auth revision: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit revisioned MFA reset: %w", err)
	}
	if store.sessionRevoker != nil {
		if err := store.sessionRevoker.RevokeAllSessions(ctx, subjectID); err != nil {
			return 0, fmt.Errorf("revoke provider sessions after revisioned MFA reset: %w", err)
		}
	}
	return resulting, nil
}

func (store *PostgresStore) Snapshot(ctx context.Context, subjectID string) (Snapshot, error) {
	var snapshot Snapshot
	err := store.pool.QueryRow(ctx, `SELECT f.subject_id, f.enabled, f.last_used_counter, COUNT(c.code_hash) FROM auth_identity.mfa_factors f LEFT JOIN auth_identity.mfa_recovery_codes c ON c.subject_id = f.subject_id WHERE f.subject_id = $1 GROUP BY f.subject_id, f.enabled, f.last_used_counter`, subjectID).Scan(&snapshot.SubjectID, &snapshot.Enabled, &snapshot.LastUsedCounter, &snapshot.RecoveryCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return Snapshot{}, ErrFactorNotFound
	}
	if err != nil {
		return Snapshot{}, fmt.Errorf("load MFA snapshot: %w", err)
	}
	return snapshot, nil
}

func (store *PostgresStore) CurrentCodeForTesting(ctx context.Context, subjectID string, at time.Time) (string, error) {
	var ciphertext []byte
	err := store.pool.QueryRow(ctx, `SELECT secret_ciphertext FROM auth_identity.mfa_factors WHERE subject_id = $1`, subjectID).Scan(&ciphertext)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrFactorNotFound
	}
	if err != nil {
		return "", fmt.Errorf("load MFA factor for test: %w", err)
	}
	secret, err := store.decrypt(subjectID, ciphertext)
	if err != nil {
		return "", ErrMFAUnavailable
	}
	return generateCode(secret, store.counter(at), store.digits), nil
}

func (store *PostgresStore) counter(at time.Time) int64 {
	return at.Unix() / int64(store.period/time.Second)
}

func (store *PostgresStore) matchCounter(secret []byte, code string, current int64) (int64, bool) {
	code = strings.TrimSpace(code)
	for offset := -store.window; offset <= store.window; offset++ {
		counter := current + int64(offset)
		if counter >= 0 && constantTimeString(generateCode(secret, counter, store.digits), code) {
			return counter, true
		}
	}
	return 0, false
}

func (store *PostgresStore) encrypt(subjectID string, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(store.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, []byte("as360-mfa-v1\x00"+subjectID)), nil
}

func (store *PostgresStore) decrypt(subjectID string, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(store.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(ciphertext) < gcm.NonceSize() {
		return nil, ErrMFAUnavailable
	}
	nonce, payload := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, payload, []byte("as360-mfa-v1\x00"+subjectID))
}

func validateConfig(configuration *Config) error {
	if len(configuration.EncryptionKey) != 32 {
		return errors.New("MFA store requires a 32-byte encryption key")
	}
	if configuration.Clock == nil {
		configuration.Clock = time.Now
	}
	if configuration.Period == 0 {
		configuration.Period = 30 * time.Second
	}
	if configuration.MaxRecoveryFailures == 0 {
		configuration.MaxRecoveryFailures = 5
	}
	if configuration.Period < 15*time.Second || configuration.Period > 5*time.Minute || configuration.Window < 0 || configuration.Window > 2 || (configuration.Digits != 0 && configuration.Digits != 6 && configuration.Digits != 8) || configuration.MaxRecoveryFailures < 1 || configuration.MaxRecoveryFailures > 20 {
		return errors.New("MFA timing or recovery policy is invalid")
	}
	if configuration.Digits == 0 {
		configuration.Digits = 6
	}
	return nil
}
