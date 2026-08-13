package identity

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/challenge"
	"github.com/jackc/pgx/v5"
)

const (
	providerStateInvited     = "INVITED"
	providerStateActive      = "ACTIVE"
	providerStateDisabled    = "DISABLED"
	providerStateSuspended   = "SUSPENDED"
	providerStateDeactivated = "DEACTIVATED"
)

// ProvisionProviderInvitation creates the account, invitation, profile, and
// authority mirror in one transaction. The raw invitation token is returned
// only to the caller that immediately queues the dedicated auth email.
func (store *PostgresStore) ProvisionProviderInvitation(
	ctx context.Context,
	input InvitationInput,
	profile ProviderProfileInput,
	authority ProviderAuthorityInput,
) (AccountSnapshot, InvitationSnapshot, error) {
	email, err := NormalizeIdentifier(IdentifierEmail, input.Email)
	if err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, err
	}
	if err := validateProviderProfile(profile); err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, err
	}
	if err := validateProviderAuthorityInput(authority, false); err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, err
	}
	subjectID, err := NewSubjectID()
	if err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, err
	}
	invitationID, err := newPrefixedID("inv_")
	if err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, err
	}
	rawToken, tokenHash, err := newRandomTokenHash()
	if err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, err
	}
	now := store.clock().UTC()
	expiresAt := now.Add(24 * time.Hour)
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, fmt.Errorf("begin provider provisioning: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_identity.accounts(subject_id, state, email_verified, auth_revision, created_at, updated_at)
		VALUES ($1, 'invited', false, 1, $2, $2)`, subjectID, now); err != nil {
		if isDuplicate(err) {
			return AccountSnapshot{}, InvitationSnapshot{}, ErrDuplicateIdentifier
		}
		return AccountSnapshot{}, InvitationSnapshot{}, fmt.Errorf("insert provider account: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_identity.identifiers(subject_id, identifier_type, normalized_value, created_at)
		VALUES ($1, 'email', $2, $3)`, subjectID, email.Normalized, now); err != nil {
		if isDuplicate(err) {
			return AccountSnapshot{}, InvitationSnapshot{}, ErrDuplicateIdentifier
		}
		return AccountSnapshot{}, InvitationSnapshot{}, fmt.Errorf("insert provider email identifier: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_identity.invitations(invitation_id, subject_id, token_hash, state, issued_at, expires_at)
		VALUES ($1, $2, $3, 'issued', $4, $5)`, invitationID, subjectID, tokenHash[:], now, expiresAt); err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, fmt.Errorf("insert provider invitation: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_identity.provider_profiles(subject_id, display_name, given_name, family_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)`, subjectID, profile.DisplayName, profile.GivenName, profile.FamilyName, now); err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, fmt.Errorf("insert provider profile: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_identity.application_authorities(
			subject_id, membership_id, organization_id, role, state, membership_revision, created_at, updated_at
		) VALUES ($1, $2, $3, $4, 'INVITED', $5, $6, $6)`, subjectID,
		authority.MembershipID, authority.OrganizationID, authority.Role,
		authority.ResultingRevision, now); err != nil {
		if isDuplicate(err) {
			return AccountSnapshot{}, InvitationSnapshot{}, ErrProviderRevisionConflict
		}
		return AccountSnapshot{}, InvitationSnapshot{}, fmt.Errorf("insert provider authority: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, fmt.Errorf("commit provider provisioning: %w", err)
	}
	return AccountSnapshot{
		SubjectID: subjectID, Email: email.Normalized, State: AccountInvited,
		AuthRevision: 1, CreatedAt: now, UpdatedAt: now,
	}, InvitationSnapshot{SubjectID: subjectID, Token: rawToken, IssuedAt: now, ExpiresAt: expiresAt}, nil
}

func (store *PostgresStore) ObserveProviderAuthority(ctx context.Context, subjectID string) (ProviderAuthority, error) {
	return scanProviderAuthority(ctx, store.pool, subjectID)
}

func (store *PostgresStore) ListProviderAuthorities(ctx context.Context, first, limit int, search string) ([]ProviderAuthority, bool, error) {
	if first < 0 || limit < 1 || limit > 100 {
		return nil, false, ErrInvalidIdentifier
	}
	search = strings.TrimSpace(search)
	rows, err := store.pool.Query(ctx, `
		SELECT a.subject_id,
		       COALESCE(email.normalized_value, ''),
		       p.display_name, p.given_name, p.family_name,
		       a.state, a.email_verified, a.auth_revision,
		       aa.membership_id, aa.organization_id, aa.role, aa.state,
		       aa.membership_revision,
		       EXISTS (SELECT 1 FROM auth_identity.mfa_factors mf WHERE mf.subject_id = a.subject_id AND mf.enabled),
		       (a.state = 'locked' OR (a.locked_until IS NOT NULL AND a.locked_until > $1)),
		       aa.updated_at
		FROM auth_identity.accounts a
		JOIN auth_identity.application_authorities aa ON aa.subject_id = a.subject_id
		JOIN auth_identity.provider_profiles p ON p.subject_id = a.subject_id
		LEFT JOIN auth_identity.identifiers email
		  ON email.subject_id = a.subject_id AND email.identifier_type = 'email'
		WHERE ($2 = '' OR email.normalized_value ILIKE '%' || $2 || '%' OR p.display_name ILIKE '%' || $2 || '%')
		ORDER BY aa.updated_at, a.subject_id
		OFFSET $3 LIMIT $4`, store.clock().UTC(), search, first, limit+1)
	if err != nil {
		return nil, false, fmt.Errorf("list provider authority: %w", err)
	}
	defer rows.Close()
	items := make([]ProviderAuthority, 0, limit)
	for rows.Next() {
		var authority ProviderAuthority
		if err := rows.Scan(
			&authority.SubjectID, &authority.Email, &authority.DisplayName,
			&authority.GivenName, &authority.FamilyName, &authority.State,
			&authority.EmailVerified, &authority.AuthRevision,
			&authority.MembershipID, &authority.OrganizationID, &authority.Role,
			&authority.State, &authority.MembershipRevision, &authority.MFAEnrolled,
			&authority.Locked, &authority.UpdatedAt,
		); err != nil {
			return nil, false, fmt.Errorf("scan provider authority: %w", err)
		}
		// The provider authority state is authoritative for the projection; the
		// account state is intentionally not copied into this field.
		if len(items) < limit {
			items = append(items, authority)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate provider authority: %w", err)
	}
	return items, len(items) == limit, nil
}

func (store *PostgresStore) UpdateProviderAuthority(ctx context.Context, subjectID string, input ProviderAuthorityInput) (ProviderAuthority, error) {
	if err := validateProviderAuthorityInput(input, true); err != nil {
		return ProviderAuthority{}, err
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return ProviderAuthority{}, fmt.Errorf("begin provider authority update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var currentRevision int64
	if err := tx.QueryRow(ctx, `SELECT membership_revision FROM auth_identity.application_authorities WHERE subject_id = $1 FOR UPDATE`, subjectID).Scan(&currentRevision); errors.Is(err, pgx.ErrNoRows) {
		return ProviderAuthority{}, ErrProviderNotFound
	} else if err != nil {
		return ProviderAuthority{}, fmt.Errorf("lock provider authority: %w", err)
	}
	if currentRevision != input.ExpectedRevision {
		return ProviderAuthority{}, ErrProviderRevisionConflict
	}
	now := store.clock().UTC()
	if _, err := tx.Exec(ctx, `
		UPDATE auth_identity.application_authorities
		SET organization_id = $2, role = $3, membership_revision = $4, updated_at = $5
		WHERE subject_id = $1 AND membership_revision = $6`, subjectID,
		input.OrganizationID, input.Role, input.ResultingRevision, now, input.ExpectedRevision); err != nil {
		return ProviderAuthority{}, fmt.Errorf("update provider authority: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.accounts SET auth_revision = auth_revision + 1, updated_at = $2 WHERE subject_id = $1`, subjectID, now); err != nil {
		return ProviderAuthority{}, fmt.Errorf("advance authority auth revision: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ProviderAuthority{}, fmt.Errorf("commit provider authority update: %w", err)
	}
	if err := store.revokeSessions(ctx, subjectID); err != nil {
		return ProviderAuthority{}, err
	}
	return store.ObserveProviderAuthority(ctx, subjectID)
}

func (store *PostgresStore) SetProviderAuthorityState(ctx context.Context, subjectID, target string, expectedRevision, resultingRevision int64) (ProviderAuthority, error) {
	target = strings.ToUpper(strings.TrimSpace(target))
	if !validProviderState(target) || expectedRevision < 1 || resultingRevision != expectedRevision+1 {
		return ProviderAuthority{}, ErrProviderRevisionConflict
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return ProviderAuthority{}, fmt.Errorf("begin provider state update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var currentRevision int64
	if err := tx.QueryRow(ctx, `SELECT membership_revision FROM auth_identity.application_authorities WHERE subject_id = $1 FOR UPDATE`, subjectID).Scan(&currentRevision); errors.Is(err, pgx.ErrNoRows) {
		return ProviderAuthority{}, ErrProviderNotFound
	} else if err != nil {
		return ProviderAuthority{}, fmt.Errorf("lock provider state: %w", err)
	}
	if currentRevision != expectedRevision {
		return ProviderAuthority{}, ErrProviderRevisionConflict
	}
	now := store.clock().UTC()
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.application_authorities SET state = $2, membership_revision = $3, updated_at = $4 WHERE subject_id = $1 AND membership_revision = $5`, subjectID, target, resultingRevision, now, expectedRevision); err != nil {
		return ProviderAuthority{}, fmt.Errorf("update provider authority state: %w", err)
	}
	accountState := mapProviderStateToAccountState(target)
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.accounts SET state = $2, auth_revision = auth_revision + 1, updated_at = $3 WHERE subject_id = $1`, subjectID, accountState, now); err != nil {
		return ProviderAuthority{}, fmt.Errorf("update provider account state: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ProviderAuthority{}, fmt.Errorf("commit provider state update: %w", err)
	}
	if err := store.revokeSessions(ctx, subjectID); err != nil {
		return ProviderAuthority{}, err
	}
	return store.ObserveProviderAuthority(ctx, subjectID)
}

// ActivateWithInvitation consumes the latest invitation and mutates the
// password, account state, authority state, and auth revision in one
// transaction. A consumed invitation is never replayable.
func (store *PostgresStore) ActivateWithInvitation(ctx context.Context, subjectID, token string, newPassword []byte) (AccountSnapshot, error) {
	if err := store.prevalidateInvitation(ctx, subjectID, token); err != nil {
		return AccountSnapshot{}, err
	}
	current, err := loadAccount(ctx, store.pool, subjectID, false)
	if err != nil {
		return AccountSnapshot{}, mapAccountReadError(err)
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
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return AccountSnapshot{}, fmt.Errorf("begin provider activation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var invitationState string
	var tokenHash []byte
	var expiresAt time.Time
	var consumedAt *time.Time
	if err := tx.QueryRow(ctx, `SELECT i.token_hash, i.state, i.expires_at, i.consumed_at FROM auth_identity.invitations i WHERE i.subject_id = $1 ORDER BY i.issued_at DESC, i.invitation_id DESC LIMIT 1 FOR UPDATE`, subjectID).Scan(&tokenHash, &invitationState, &expiresAt, &consumedAt); errors.Is(err, pgx.ErrNoRows) {
		return AccountSnapshot{}, ErrInvitationNotFound
	} else if err != nil {
		return AccountSnapshot{}, fmt.Errorf("load provider activation invitation: %w", err)
	}
	provided := DigestSecret(token)
	if len(tokenHash) != len(provided) || subtle.ConstantTimeCompare(tokenHash, provided[:]) != 1 || invitationState != "issued" || consumedAt != nil {
		return AccountSnapshot{}, ErrInvitationNotFound
	}
	now := store.clock().UTC()
	if !now.Before(expiresAt) {
		_, _ = tx.Exec(ctx, `UPDATE auth_identity.invitations SET state = 'expired', invalidated_at = $2 WHERE subject_id = $1 AND state = 'issued'`, subjectID, now)
		return AccountSnapshot{}, ErrInvitationExpired
	}
	locked, err := loadAccount(ctx, tx, subjectID, true)
	if err != nil {
		return AccountSnapshot{}, mapAccountReadError(err)
	}
	if locked.authRevision != current.authRevision || locked.state != AccountInvited {
		return AccountSnapshot{}, ErrRevisionConflict
	}
	var authoritySubject string
	if err := tx.QueryRow(ctx, `SELECT subject_id FROM auth_identity.application_authorities WHERE subject_id = $1 FOR UPDATE`, subjectID).Scan(&authoritySubject); errors.Is(err, pgx.ErrNoRows) {
		return AccountSnapshot{}, ErrProviderNotFound
	} else if err != nil {
		return AccountSnapshot{}, fmt.Errorf("lock provider activation authority: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.invitations SET state = 'consumed', consumed_at = $2 WHERE subject_id = $1 AND state = 'issued'`, subjectID, now); err != nil {
		return AccountSnapshot{}, fmt.Errorf("consume provider invitation: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.accounts SET state = 'active', email_verified = true, password_hash = $2, auth_revision = auth_revision + 1, updated_at = $3 WHERE subject_id = $1 AND auth_revision = $4`, subjectID, hash, now, current.authRevision); err != nil {
		return AccountSnapshot{}, fmt.Errorf("activate provider account: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE auth_identity.application_authorities SET state = 'ACTIVE', updated_at = $2 WHERE subject_id = $1`, subjectID, now); err != nil {
		return AccountSnapshot{}, fmt.Errorf("activate provider authority: %w", err)
	}
	updated, err := loadAccount(ctx, tx, subjectID, true)
	if err != nil {
		return AccountSnapshot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AccountSnapshot{}, fmt.Errorf("commit provider activation: %w", err)
	}
	if err := store.revokeSessions(ctx, subjectID); err != nil {
		return AccountSnapshot{}, err
	}
	return updated.snapshot(), nil
}

// prevalidateInvitation is the cheap invitation-owned boundary. It performs
// no password policy or Argon2 work and never authorizes the later mutation.
func (store *PostgresStore) prevalidateInvitation(ctx context.Context, subjectID, token string) error {
	var tokenHash []byte
	var state string
	var expiresAt time.Time
	var consumedAt *time.Time
	err := store.pool.QueryRow(ctx, `SELECT token_hash, state, expires_at, consumed_at FROM auth_identity.invitations WHERE subject_id = $1 ORDER BY issued_at DESC, invitation_id DESC LIMIT 1`, subjectID).Scan(&tokenHash, &state, &expiresAt, &consumedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvitationNotFound
	}
	if err != nil {
		return fmt.Errorf("prevalidate provider invitation: %w", err)
	}
	provided := DigestSecret(token)
	if len(tokenHash) != len(provided) || subtle.ConstantTimeCompare(tokenHash, provided[:]) != 1 || state != "issued" || consumedAt != nil {
		return ErrInvitationNotFound
	}
	if !store.clock().UTC().Before(expiresAt) {
		return ErrInvitationExpired
	}
	return nil
}

// ResetPasswordWithChallenge consumes the subject-bound recovery challenge and
// changes the password under one database transaction.
func (store *PostgresStore) ResetPasswordWithChallenge(ctx context.Context, subjectID, purpose, token string, newPassword []byte) (AccountSnapshot, error) {
	purposeValue := challenge.Purpose(purpose)
	if _, err := store.challenges.Prevalidate(ctx, subjectID, purposeValue, token); err != nil {
		return AccountSnapshot{}, ErrInvalidRecovery
	}
	current, err := loadAccount(ctx, store.pool, subjectID, false)
	if err != nil {
		return AccountSnapshot{}, mapAccountReadError(err)
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
	var updated dbAccount
	err = store.challenges.WithMutation(ctx, subjectID, purposeValue, token, func(ctx context.Context, tx pgx.Tx, handle challenge.MutationHandle) error {
		locked, err := loadAccount(ctx, tx, subjectID, true)
		if err != nil {
			return mapAccountReadError(err)
		}
		if locked.authRevision != current.authRevision || locked.state != AccountActive {
			return ErrRevisionConflict
		}
		now := store.clock().UTC()
		if err := handle.Consume(ctx); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO auth_identity.password_history(subject_id, history_revision, password_hash, created_at) VALUES ($1, $2, $3, $4)`, subjectID, current.authRevision, current.passwordHash, now); err != nil {
			return fmt.Errorf("record transactional password history: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE auth_identity.accounts SET password_hash = $2, auth_revision = auth_revision + 1, failed_login_count = 0, locked_until = NULL, updated_at = $3 WHERE subject_id = $1 AND auth_revision = $4`, subjectID, newHash, now, current.authRevision); err != nil {
			return fmt.Errorf("apply transactional password recovery: %w", err)
		}
		updated, err = loadAccount(ctx, tx, subjectID, true)
		return err
	})
	if errors.Is(err, challenge.ErrInvalidChallenge) || errors.Is(err, challenge.ErrChallengeExpired) || errors.Is(err, challenge.ErrChallengeUsed) || errors.Is(err, challenge.ErrChallengeLocked) {
		return AccountSnapshot{}, ErrInvalidRecovery
	}
	if err != nil {
		return AccountSnapshot{}, err
	}
	if err := store.revokeSessions(ctx, subjectID); err != nil {
		return AccountSnapshot{}, err
	}
	return updated.snapshot(), nil
}

func (store *PostgresStore) AdvanceAuthRevision(ctx context.Context, subjectID string) (uint64, error) {
	var revision uint64
	err := store.pool.QueryRow(ctx, `UPDATE auth_identity.accounts SET auth_revision = auth_revision + 1, updated_at = $2 WHERE subject_id = $1 RETURNING auth_revision`, subjectID, store.clock().UTC()).Scan(&revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrAccountNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("advance auth revision: %w", err)
	}
	if err := store.revokeSessions(ctx, subjectID); err != nil {
		return 0, err
	}
	return revision, nil
}

func scanProviderAuthority(ctx context.Context, queryer rowQuerier, subjectID string) (ProviderAuthority, error) {
	var authority ProviderAuthority
	var accountState string
	err := queryer.QueryRow(ctx, `
		SELECT a.subject_id, COALESCE(email.normalized_value, ''), p.display_name,
		       p.given_name, p.family_name, a.state, a.email_verified, a.auth_revision,
		       aa.membership_id, aa.organization_id, aa.role, aa.state,
		       aa.membership_revision,
		       EXISTS (SELECT 1 FROM auth_identity.mfa_factors mf WHERE mf.subject_id = a.subject_id AND mf.enabled),
		       (a.state = 'locked' OR (a.locked_until IS NOT NULL AND a.locked_until > now())),
		       aa.updated_at
		FROM auth_identity.accounts a
		JOIN auth_identity.application_authorities aa ON aa.subject_id = a.subject_id
		JOIN auth_identity.provider_profiles p ON p.subject_id = a.subject_id
		LEFT JOIN auth_identity.identifiers email
		  ON email.subject_id = a.subject_id AND email.identifier_type = 'email'
		WHERE a.subject_id = $1`, subjectID).Scan(
		&authority.SubjectID, &authority.Email, &authority.DisplayName,
		&authority.GivenName, &authority.FamilyName, &accountState,
		&authority.EmailVerified, &authority.AuthRevision,
		&authority.MembershipID, &authority.OrganizationID, &authority.Role,
		&authority.State, &authority.MembershipRevision, &authority.MFAEnrolled,
		&authority.Locked, &authority.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProviderAuthority{}, ErrProviderNotFound
	}
	if err != nil {
		return ProviderAuthority{}, fmt.Errorf("observe provider authority: %w", err)
	}
	authority.State = strings.ToUpper(strings.TrimSpace(authority.State))
	_ = accountState
	return authority, nil
}

func validateProviderProfile(profile ProviderProfileInput) error {
	if strings.TrimSpace(profile.DisplayName) == "" || len(profile.DisplayName) > 200 || len(profile.GivenName) > 100 || len(profile.FamilyName) > 100 {
		return ErrInvalidIdentifier
	}
	return nil
}

func validateProviderAuthorityInput(input ProviderAuthorityInput, update bool) error {
	if strings.TrimSpace(input.MembershipID) == "" || strings.TrimSpace(input.OrganizationID) == "" || strings.TrimSpace(input.Role) == "" || !validProviderState(strings.ToUpper(strings.TrimSpace(input.State))) {
		if !update && strings.TrimSpace(input.State) == "" {
			// Provisioning always starts in INVITED and does not need the caller
			// to repeat that state in the request.
		} else {
			return ErrInvalidIdentifier
		}
	}
	if input.ExpectedRevision < 0 || input.ResultingRevision <= 0 || input.ResultingRevision != input.ExpectedRevision+1 {
		return ErrProviderRevisionConflict
	}
	if err := validateProviderAuthority(strings.TrimSpace(input.OrganizationID), strings.TrimSpace(input.Role)); err != nil {
		return err
	}
	return nil
}

func validateProviderAuthority(organizationID, role string) error {
	if organizationID == "" || role == "" {
		return ErrInvalidIdentifier
	}
	if organizationID == "CAA" {
		switch role {
		case "admin", "inspector", "leadInspector", "manager", "gm", "finance", "executiveDirector":
			return nil
		}
	} else if role == "auditee" {
		return nil
	}
	return ErrInvalidIdentifier
}

func validProviderState(state string) bool {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case providerStateInvited, providerStateActive, providerStateDisabled, providerStateSuspended, providerStateDeactivated:
		return true
	default:
		return false
	}
}

func mapProviderStateToAccountState(state string) AccountState {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case providerStateActive:
		return AccountActive
	case providerStateDisabled:
		return AccountDisabled
	case providerStateSuspended:
		return AccountSuspended
	case providerStateDeactivated:
		return AccountDeletionPending
	default:
		return AccountInvited
	}
}
