package mfa

import (
	"context"
	"errors"
	"fmt"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/challenge"
	"github.com/jackc/pgx/v5"
)

// ResetWithChallenge consumes the MFA recovery challenge, removes the factor,
// and advances auth_revision in one transaction. It is deliberately separate
// from the general Reset method so browser recovery cannot split consumption
// and mutation across two commits.
func (store *PostgresStore) ResetWithChallenge(ctx context.Context, subjectID, purpose, token string) error {
	purposeValue := challenge.Purpose(purpose)
	if _, err := store.challenges.Prevalidate(ctx, subjectID, purposeValue, token); err != nil {
		return ErrRecoveryInvalid
	}
	err := store.challenges.WithMutation(ctx, subjectID, purposeValue, token, func(ctx context.Context, tx pgx.Tx, handle challenge.MutationHandle) error {
		var factorSubject string
		if err := tx.QueryRow(ctx, `SELECT subject_id FROM auth_identity.mfa_factors WHERE subject_id = $1 FOR UPDATE`, subjectID).Scan(&factorSubject); errors.Is(err, pgx.ErrNoRows) {
			return ErrFactorNotFound
		} else if err != nil {
			return fmt.Errorf("lock MFA factor for recovery: %w", err)
		}
		now := store.clock().UTC()
		if err := handle.Consume(ctx); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM auth_identity.mfa_factors WHERE subject_id = $1`, factorSubject); err != nil {
			return fmt.Errorf("delete MFA factor during recovery: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE auth_identity.accounts SET auth_revision = auth_revision + 1, updated_at = $2 WHERE subject_id = $1`, subjectID, now); err != nil {
			return fmt.Errorf("advance MFA auth revision: %w", err)
		}
		return nil
	})
	if errors.Is(err, challenge.ErrInvalidChallenge) || errors.Is(err, challenge.ErrChallengeExpired) || errors.Is(err, challenge.ErrChallengeUsed) || errors.Is(err, challenge.ErrChallengeLocked) {
		return ErrRecoveryInvalid
	}
	if err != nil {
		return err
	}
	if store.sessionRevoker != nil {
		if err := store.sessionRevoker.RevokeAllSessions(ctx, subjectID); err != nil {
			return fmt.Errorf("revoke provider sessions after MFA recovery: %w", err)
		}
	}
	return nil
}
