package mfa

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/challenge"
	"github.com/jackc/pgx/v5"
)

// ResetWithChallenge consumes the MFA recovery challenge, removes the factor,
// and advances auth_revision in one transaction. It is deliberately separate
// from the general Reset method so browser recovery cannot split consumption
// and mutation across two commits.
func (store *PostgresStore) ResetWithChallenge(ctx context.Context, subjectID, purpose, token string) error {
	hash := challenge.DigestToken(token)
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transactional MFA recovery: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var storedHash []byte
	var state string
	var attempts, maxAttempts int
	var expiresAt time.Time
	if err := tx.QueryRow(ctx, `SELECT token_hash, state, attempt_count, max_attempts, expires_at FROM auth_identity.identity_challenges WHERE token_hash = $1 AND subject_id = $2 AND purpose = $3 FOR UPDATE`, hash[:], subjectID, purpose).Scan(&storedHash, &state, &attempts, &maxAttempts, &expiresAt); errors.Is(err, pgx.ErrNoRows) {
		return ErrRecoveryInvalid
	} else if err != nil {
		return fmt.Errorf("lock MFA recovery challenge: %w", err)
	}
	if len(storedHash) != len(hash) || subtle.ConstantTimeCompare(storedHash, hash[:]) != 1 || state != "active" || attempts >= maxAttempts || !store.clock().UTC().Before(expiresAt) {
		return ErrRecoveryInvalid
	}
	var factorSubject string
	if err := tx.QueryRow(ctx, `SELECT subject_id FROM auth_identity.mfa_factors WHERE subject_id = $1 FOR UPDATE`, subjectID).Scan(&factorSubject); errors.Is(err, pgx.ErrNoRows) {
		return ErrFactorNotFound
	} else if err != nil {
		return fmt.Errorf("lock MFA factor for recovery: %w", err)
	}
	now := store.clock().UTC()
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.identity_challenges SET state = 'consumed', consumed_at = $2, updated_at = $2 WHERE token_hash = $1`, hash[:], now); err != nil {
		return fmt.Errorf("consume MFA recovery challenge: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM auth_identity.mfa_factors WHERE subject_id = $1`, factorSubject); err != nil {
		return fmt.Errorf("delete MFA factor during recovery: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.accounts SET auth_revision = auth_revision + 1, updated_at = $2 WHERE subject_id = $1`, subjectID, now); err != nil {
		return fmt.Errorf("advance MFA auth revision: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transactional MFA recovery: %w", err)
	}
	if store.sessionRevoker != nil {
		if err := store.sessionRevoker.RevokeAllSessions(ctx, subjectID); err != nil {
			return fmt.Errorf("revoke provider sessions after MFA recovery: %w", err)
		}
	}
	return nil
}
