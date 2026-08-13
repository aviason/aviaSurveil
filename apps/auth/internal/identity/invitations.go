package identity

import (
	"context"
	"crypto/hmac"
	"time"
)

// VerifyInvitation consumes the one-time invitation credential and advances
// the account's authentication revision. The raw token is accepted only at
// this boundary and is never returned from a later snapshot.
func (store *Store) VerifyInvitation(ctx context.Context, subjectID, token string) (AccountSnapshot, error) {
	store.mu.Lock()
	account := store.accounts[subjectID]
	invitation := store.invitations[subjectID]
	providedHash := DigestSecret(token)
	if account == nil || invitation == nil || !hmac.Equal(invitation.tokenHash[:], providedHash[:]) {
		store.mu.Unlock()
		return AccountSnapshot{}, ErrInvitationNotFound
	}
	now := store.clock()
	if invitation.consumedAt.After(time.Time{}) {
		store.mu.Unlock()
		return AccountSnapshot{}, ErrInvitationNotFound
	}
	if !now.Before(invitation.expiresAt) {
		store.mu.Unlock()
		return AccountSnapshot{}, ErrInvitationExpired
	}
	if account.state != AccountInvited {
		store.mu.Unlock()
		return AccountSnapshot{}, ErrInvalidTransition
	}
	invitation.consumedAt = now
	account.emailVerified = true
	account.authRevision++
	account.updatedAt = now
	snapshot := account.snapshot()
	store.mu.Unlock()
	if err := store.revokeSessions(ctx, subjectID); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

func (store *Store) ResendInvitation(_ context.Context, subjectID string) (InvitationSnapshot, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	account := store.accounts[subjectID]
	invitation := store.invitations[subjectID]
	if account == nil || invitation == nil || account.state != AccountInvited {
		return InvitationSnapshot{}, ErrInvitationNotFound
	}
	if invitation.resendCount >= 3 {
		return InvitationSnapshot{}, ErrInvitationResendLimit
	}
	rawToken, tokenHash, err := newRandomTokenHash()
	if err != nil {
		return InvitationSnapshot{}, err
	}
	now := store.clock()
	invitation.invalidatedAt = now
	invitation.issuedAt = now
	invitation.expiresAt = now.Add(24 * time.Hour)
	invitation.resendCount++
	invitation.tokenHash = tokenHash
	snapshot := invitation.snapshot()
	snapshot.Token = rawToken
	return snapshot, nil
}
