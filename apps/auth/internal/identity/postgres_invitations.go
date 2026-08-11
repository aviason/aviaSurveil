package identity

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (store *PostgresStore) VerifyInvitation(ctx context.Context, subjectID, token string) (AccountSnapshot, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return AccountSnapshot{}, fmt.Errorf("begin invitation verification: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var state, invitationState string
	var revision uint64
	var tokenHash []byte
	var expiresAt time.Time
	var consumedAt *time.Time
	if err := tx.QueryRow(ctx, `
		SELECT a.state, a.auth_revision, i.token_hash, i.state, i.expires_at, i.consumed_at
		FROM auth_identity.accounts a
		JOIN auth_identity.invitations i ON i.subject_id = a.subject_id
		WHERE a.subject_id = $1
		ORDER BY i.issued_at DESC, i.invitation_id DESC
		LIMIT 1
		FOR UPDATE OF a, i`, subjectID).Scan(&state, &revision, &tokenHash, &invitationState, &expiresAt, &consumedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AccountSnapshot{}, ErrInvitationNotFound
		}
		return AccountSnapshot{}, fmt.Errorf("load invitation: %w", err)
	}
	providedHash := DigestSecret(token)
	if len(tokenHash) != len(providedHash) || subtle.ConstantTimeCompare(tokenHash, providedHash[:]) != 1 || invitationState != "issued" || state != string(AccountInvited) || consumedAt != nil {
		return AccountSnapshot{}, ErrInvitationNotFound
	}
	now := store.clock()
	if !now.Before(expiresAt) {
		if _, err := tx.Exec(ctx, `UPDATE auth_identity.invitations SET state = 'expired', invalidated_at = $2 WHERE subject_id = $1 AND state = 'issued'`, subjectID, now); err != nil {
			return AccountSnapshot{}, fmt.Errorf("expire invitation: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return AccountSnapshot{}, fmt.Errorf("commit expired invitation: %w", err)
		}
		return AccountSnapshot{}, ErrInvitationExpired
	}
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.invitations SET state = 'consumed', consumed_at = $2 WHERE subject_id = $1 AND state = 'issued'`, subjectID, now); err != nil {
		return AccountSnapshot{}, fmt.Errorf("consume invitation: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.accounts SET email_verified = true, auth_revision = auth_revision + 1, updated_at = $2 WHERE subject_id = $1 AND auth_revision = $3`, subjectID, now, revision); err != nil {
		return AccountSnapshot{}, fmt.Errorf("verify invited account: %w", err)
	}
	updated, err := loadAccount(ctx, tx, subjectID, true)
	if err != nil {
		return AccountSnapshot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AccountSnapshot{}, fmt.Errorf("commit invitation verification: %w", err)
	}
	return updated.snapshot(), nil
}

func (store *PostgresStore) ResendInvitation(ctx context.Context, subjectID string) (InvitationSnapshot, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return InvitationSnapshot{}, fmt.Errorf("begin invitation resend: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var state, invitationState string
	var resendCount int
	var expiresAt, issuedAt time.Time
	if err := tx.QueryRow(ctx, `
		SELECT a.state, i.state, i.resend_count, i.issued_at, i.expires_at
		FROM auth_identity.accounts a
		JOIN auth_identity.invitations i ON i.subject_id = a.subject_id
		WHERE a.subject_id = $1
		ORDER BY i.issued_at DESC, i.invitation_id DESC
		LIMIT 1
		FOR UPDATE OF a, i`, subjectID).Scan(&state, &invitationState, &resendCount, &issuedAt, &expiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InvitationSnapshot{}, ErrInvitationNotFound
		}
		return InvitationSnapshot{}, fmt.Errorf("load invitation for resend: %w", err)
	}
	if state != string(AccountInvited) || invitationState != "issued" || resendCount >= 3 {
		if resendCount >= 3 {
			return InvitationSnapshot{}, ErrInvitationResendLimit
		}
		return InvitationSnapshot{}, ErrInvitationNotFound
	}
	_ = expiresAt
	rawToken, tokenHash, err := newRandomTokenHash()
	if err != nil {
		return InvitationSnapshot{}, err
	}
	now := store.clock()
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.invitations SET state = 'cancelled', invalidated_at = $2 WHERE subject_id = $1 AND state = 'issued'`, subjectID, now); err != nil {
		return InvitationSnapshot{}, fmt.Errorf("invalidate prior invitation: %w", err)
	}
	invitationID, err := newPrefixedID("inv_")
	if err != nil {
		return InvitationSnapshot{}, err
	}
	newExpiry := now.Add(24 * time.Hour)
	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_identity.invitations(invitation_id, subject_id, token_hash, state, resend_count, issued_at, expires_at)
		VALUES ($1, $2, $3, 'issued', $4, $5, $6)`, invitationID, subjectID, tokenHash[:], resendCount+1, now, newExpiry); err != nil {
		return InvitationSnapshot{}, fmt.Errorf("insert resent invitation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return InvitationSnapshot{}, fmt.Errorf("commit invitation resend: %w", err)
	}
	return InvitationSnapshot{SubjectID: subjectID, Token: rawToken, IssuedAt: now, ExpiresAt: newExpiry, ResendCount: resendCount + 1, InvalidatedAt: issuedAt}, nil
}
