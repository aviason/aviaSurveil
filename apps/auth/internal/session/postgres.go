package session

import (
	"context"
	"crypto/hmac"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore is the durable provider-session adapter. Rotation locks the
// family and session rows in one transaction, so only one presentation of a
// current refresh token can advance a family.
type PostgresStore struct {
	pool                  *pgxpool.Pool
	authorizer            Authorizer
	clients               ClientRegistry
	fingerprintKey        []byte
	clock                 Clock
	idleTTL               time.Duration
	absoluteTTL           time.Duration
	maxFamiliesPerSubject int
}

func NewPostgresStore(pool *pgxpool.Pool, configuration Config) (*PostgresStore, error) {
	if pool == nil || configuration.Authorizer == nil || configuration.Clients == nil || len(configuration.FingerprintKey) < 32 {
		return nil, errors.New("PostgreSQL session store requires pool, authorizer, client registry, and 32-byte fingerprint key")
	}
	if configuration.Clock == nil {
		configuration.Clock = time.Now
	}
	if configuration.IdleTTL <= 0 || configuration.AbsoluteTTL <= configuration.IdleTTL || configuration.MaxFamiliesPerSubject < 1 || configuration.MaxFamiliesPerSubject > 32 {
		return nil, errors.New("PostgreSQL session expiry and family limits are invalid")
	}
	return &PostgresStore{
		pool:                  pool,
		authorizer:            configuration.Authorizer,
		clients:               configuration.Clients,
		fingerprintKey:        append([]byte(nil), configuration.FingerprintKey...),
		clock:                 configuration.Clock,
		idleTTL:               configuration.IdleTTL,
		absoluteTTL:           configuration.AbsoluteTTL,
		maxFamiliesPerSubject: configuration.MaxFamiliesPerSubject,
	}, nil
}

type durableFamily struct {
	familyID          string
	sessionID         string
	subjectID         string
	clientID          string
	fingerprintHash   [32]byte
	authRevision      uint64
	currentTokenHash  [32]byte
	generation        uint64
	familyState       FamilyState
	sessionState      SessionState
	issuedAt          time.Time
	lastUsedAt        time.Time
	idleExpiresAt     time.Time
	absoluteExpiresAt time.Time
}

func (store *PostgresStore) Issue(ctx context.Context, input IssueInput) (IssuedRefresh, error) {
	if strings.TrimSpace(input.SubjectID) == "" || input.AuthRevision == 0 || !store.clients.Validate(input.ClientID, input.RedirectURI) {
		return IssuedRefresh{}, ErrInvalidClient
	}
	if authorized, err := store.authorizer.Authorize(ctx, input.SubjectID, input.AuthRevision); err != nil || !authorized {
		return IssuedRefresh{}, ErrSubjectUnauthorized
	}
	fingerprint := DeriveFingerprint(store.fingerprintKey, input.Fingerprint)
	now := store.clock()
	sessionID, err := newPrefixedID("ses_")
	if err != nil {
		return IssuedRefresh{}, err
	}
	familyID, err := newPrefixedID("fam_")
	if err != nil {
		return IssuedRefresh{}, err
	}
	refreshToken, tokenHash, err := newRefreshToken()
	if err != nil {
		return IssuedRefresh{}, err
	}

	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return IssuedRefresh{}, fmt.Errorf("begin provider session issue: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var state string
	var emailVerified bool
	var revision uint64
	if err := tx.QueryRow(ctx, `
		SELECT state, email_verified, auth_revision
		FROM auth_identity.accounts
		WHERE subject_id = $1
		FOR UPDATE`, input.SubjectID).Scan(&state, &emailVerified, &revision); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return IssuedRefresh{}, ErrSubjectUnauthorized
		}
		return IssuedRefresh{}, fmt.Errorf("lock session subject: %w", err)
	}
	if state != "active" || !emailVerified || revision != input.AuthRevision {
		return IssuedRefresh{}, ErrSubjectUnauthorized
	}
	activeFamilies, err := lockedActiveFamilies(ctx, tx, input.SubjectID)
	if err != nil {
		return IssuedRefresh{}, fmt.Errorf("lock active provider families: %w", err)
	}
	for len(activeFamilies) >= store.maxFamiliesPerSubject {
		oldest := activeFamilies[0]
		for _, candidate := range activeFamilies[1:] {
			if candidate.issuedAt.Before(oldest.issuedAt) {
				oldest = candidate
			}
		}
		if err := revokeFamilyTx(ctx, tx, oldest.familyID, FamilyRevoked, SessionRevoked, now); err != nil {
			return IssuedRefresh{}, fmt.Errorf("bound provider families: %w", err)
		}
		filtered := activeFamilies[:0]
		for _, candidate := range activeFamilies {
			if candidate.familyID != oldest.familyID {
				filtered = append(filtered, candidate)
			}
		}
		activeFamilies = filtered
	}
	idleExpiresAt := now.Add(store.idleTTL)
	absoluteExpiresAt := now.Add(store.absoluteTTL)
	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_identity.provider_sessions(
			session_id, subject_id, family_id, client_id, fingerprint_hash,
			auth_revision, state, issued_at, last_used_at, idle_expires_at,
			absolute_expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, 'active', $7, $7, $8, $9)`,
		sessionID, input.SubjectID, familyID, input.ClientID, fingerprint[:], input.AuthRevision,
		now, idleExpiresAt, absoluteExpiresAt); err != nil {
		return IssuedRefresh{}, fmt.Errorf("insert provider session: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_identity.refresh_families(
			family_id, session_id, current_token_hash, generation, state,
			issued_at, last_used_at, idle_expires_at, absolute_expires_at
		) VALUES ($1, $2, $3, 1, 'active', $4, $4, $5, $6)`,
		familyID, sessionID, tokenHash[:], now, idleExpiresAt, absoluteExpiresAt); err != nil {
		return IssuedRefresh{}, fmt.Errorf("insert refresh family: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return IssuedRefresh{}, fmt.Errorf("commit provider session issue: %w", err)
	}
	return IssuedRefresh{
		SessionID: sessionID, FamilyID: familyID, RefreshToken: refreshToken,
		AuthRevision: input.AuthRevision, IssuedAt: now,
		IdleExpiresAt: idleExpiresAt, AbsoluteExpiresAt: absoluteExpiresAt,
	}, nil
}

func (store *PostgresStore) Rotate(ctx context.Context, input RotateInput) (RotationResult, error) {
	if strings.TrimSpace(input.RefreshToken) == "" || !store.clients.Validate(input.ClientID, input.RedirectURI) {
		return RotationResult{}, ErrInvalidClient
	}
	tokenHash := HashRefreshToken(input.RefreshToken)
	fingerprint := DeriveFingerprint(store.fingerprintKey, input.Fingerprint)
	now := store.clock()
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return RotationResult{}, fmt.Errorf("begin refresh rotation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	family, err := loadCurrentFamily(ctx, tx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		var familyID string
		historyErr := tx.QueryRow(ctx, `SELECT family_id FROM auth_identity.refresh_token_history WHERE token_hash = $1`, tokenHash[:]).Scan(&familyID)
		if errors.Is(historyErr, pgx.ErrNoRows) {
			return RotationResult{}, ErrSessionNotFound
		}
		if historyErr != nil {
			return RotationResult{}, fmt.Errorf("look up refresh history: %w", historyErr)
		}
		if err := lockFamily(ctx, tx, familyID); err != nil {
			return RotationResult{}, err
		}
		return RotationResult{}, finishRevocation(ctx, tx, familyID, FamilyReuseDetected, SessionRevoked, now, ErrRefreshReuse)
	}
	if err != nil {
		return RotationResult{}, fmt.Errorf("load current refresh family: %w", err)
	}
	if family.familyState != FamilyActive {
		return RotationResult{}, stateError(family.familyState)
	}
	if family.sessionState != SessionActive {
		return RotationResult{}, ErrSessionRevoked
	}
	if family.clientID != input.ClientID {
		return RotationResult{}, finishRevocation(ctx, tx, family.familyID, FamilyRevoked, SessionRevoked, now, ErrInvalidClient)
	}
	if !hmac.Equal(family.fingerprintHash[:], fingerprint[:]) {
		return RotationResult{}, finishRevocation(ctx, tx, family.familyID, FamilyRevoked, SessionRevoked, now, ErrFingerprintMismatch)
	}
	if !now.Before(family.absoluteExpiresAt) || !now.Before(family.idleExpiresAt) {
		return RotationResult{}, finishRevocation(ctx, tx, family.familyID, FamilyExpired, SessionExpired, now, ErrRefreshExpired)
	}
	authorized, authorizeErr := store.authorizer.Authorize(ctx, family.subjectID, family.authRevision)
	if authorizeErr != nil || !authorized {
		return RotationResult{}, finishRevocation(ctx, tx, family.familyID, FamilyRevoked, SessionRevoked, now, ErrAuthRevisionStale)
	}
	refreshToken, nextHash, err := newRefreshToken()
	if err != nil {
		return RotationResult{}, err
	}
	idleExpiresAt := minTime(now.Add(store.idleTTL), family.absoluteExpiresAt)
	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_identity.refresh_token_history(token_hash, family_id, used_at)
		VALUES ($1, $2, $3)`, tokenHash[:], family.familyID, now); err != nil {
		return RotationResult{}, fmt.Errorf("record consumed refresh token: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE auth_identity.refresh_families
		SET current_token_hash = $2, generation = generation + 1,
			last_used_at = $3, idle_expires_at = $4
		WHERE family_id = $1 AND state = 'active' AND current_token_hash = $5`,
		family.familyID, nextHash[:], now, idleExpiresAt, tokenHash[:]); err != nil {
		return RotationResult{}, fmt.Errorf("advance refresh family: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE auth_identity.provider_sessions
		SET last_used_at = $2, idle_expires_at = $3
		WHERE session_id = $1 AND state = 'active'`, family.sessionID, now, idleExpiresAt); err != nil {
		return RotationResult{}, fmt.Errorf("advance provider session: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return RotationResult{}, fmt.Errorf("commit refresh rotation: %w", err)
	}
	return RotationResult{
		SessionID: family.sessionID, FamilyID: family.familyID, RefreshToken: refreshToken,
		AuthRevision: family.authRevision, Generation: family.generation + 1,
		IdleExpiresAt: idleExpiresAt, AbsoluteExpiresAt: family.absoluteExpiresAt,
	}, nil
}

func (store *PostgresStore) RevokeFamily(ctx context.Context, subjectID, familyID string) error {
	now := store.clock()
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin family revocation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var storedSubject string
	if err := tx.QueryRow(ctx, `
		SELECT s.subject_id
		FROM auth_identity.refresh_families f
		JOIN auth_identity.provider_sessions s ON s.session_id = f.session_id
		WHERE f.family_id = $1
		FOR UPDATE OF f, s`, familyID).Scan(&storedSubject); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("load family for revocation: %w", err)
	}
	if storedSubject != subjectID {
		return ErrSessionNotFound
	}
	if err := revokeFamilyTx(ctx, tx, familyID, FamilyRevoked, SessionRevoked, now); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit family revocation: %w", err)
	}
	return nil
}

func (store *PostgresStore) RevokeAll(ctx context.Context, subjectID string) int {
	count, _ := store.RevokeAllSessions(ctx, subjectID)
	return count
}

func (store *PostgresStore) RevokeAllSessions(ctx context.Context, subjectID string) (int, error) {
	now := store.clock()
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin all-family revocation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		SELECT f.family_id
		FROM auth_identity.refresh_families f
		JOIN auth_identity.provider_sessions s ON s.session_id = f.session_id
		WHERE s.subject_id = $1 AND f.state = 'active'
		ORDER BY f.issued_at
		FOR UPDATE OF f, s`, subjectID)
	if err != nil {
		return 0, fmt.Errorf("lock subject families: %w", err)
	}
	var familyIDs []string
	for rows.Next() {
		var familyID string
		if err := rows.Scan(&familyID); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan subject family: %w", err)
		}
		familyIDs = append(familyIDs, familyID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("read subject families: %w", err)
	}
	rows.Close()
	for _, familyID := range familyIDs {
		if err := revokeFamilyTx(ctx, tx, familyID, FamilyRevoked, SessionRevoked, now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit all-family revocation: %w", err)
	}
	return len(familyIDs), nil
}

func (store *PostgresStore) Cleanup(ctx context.Context, at time.Time) (int, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin session cleanup: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		SELECT f.family_id
		FROM auth_identity.refresh_families f
		WHERE f.state = 'active' AND ($1 >= f.absolute_expires_at OR $1 >= f.idle_expires_at)
		FOR UPDATE`, at)
	if err != nil {
		return 0, fmt.Errorf("select expired families: %w", err)
	}
	var familyIDs []string
	for rows.Next() {
		var familyID string
		if err := rows.Scan(&familyID); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan expired family: %w", err)
		}
		familyIDs = append(familyIDs, familyID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("read expired families: %w", err)
	}
	rows.Close()
	for _, familyID := range familyIDs {
		if err := revokeFamilyTx(ctx, tx, familyID, FamilyExpired, SessionExpired, at); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit session cleanup: %w", err)
	}
	return len(familyIDs), nil
}

func (store *PostgresStore) Snapshot(ctx context.Context, familyID string) (Snapshot, error) {
	var snapshot Snapshot
	var sessionState, familyState string
	var fingerprint []byte
	var generation int64
	var authRevision int64
	err := store.pool.QueryRow(ctx, `
		SELECT s.session_id, f.family_id, s.subject_id, s.client_id,
			s.auth_revision, s.state, f.state, f.generation,
			s.issued_at, s.last_used_at, s.idle_expires_at,
			s.absolute_expires_at, s.fingerprint_hash
		FROM auth_identity.refresh_families f
		JOIN auth_identity.provider_sessions s ON s.session_id = f.session_id
		WHERE f.family_id = $1`, familyID).Scan(
		&snapshot.SessionID, &snapshot.FamilyID, &snapshot.SubjectID, &snapshot.ClientID,
		&authRevision, &sessionState, &familyState, &generation,
		&snapshot.IssuedAt, &snapshot.LastUsedAt, &snapshot.IdleExpiresAt,
		&snapshot.AbsoluteExpiresAt, &fingerprint)
	if errors.Is(err, pgx.ErrNoRows) {
		return Snapshot{}, ErrSessionNotFound
	}
	if err != nil {
		return Snapshot{}, fmt.Errorf("read provider session snapshot: %w", err)
	}
	if len(fingerprint) != len(snapshot.FingerprintHash) {
		return Snapshot{}, errors.New("stored provider fingerprint has invalid length")
	}
	copy(snapshot.FingerprintHash[:], fingerprint)
	snapshot.AuthRevision = uint64(authRevision)
	snapshot.SessionState = SessionState(sessionState)
	snapshot.FamilyState = FamilyState(familyState)
	snapshot.Generation = uint64(generation)
	return snapshot, nil
}

func lockedActiveFamilies(ctx context.Context, tx pgx.Tx, subjectID string) ([]lockedFamily, error) {
	rows, err := tx.Query(ctx, `
		SELECT f.family_id, f.issued_at
		FROM auth_identity.refresh_families f
		JOIN auth_identity.provider_sessions s ON s.session_id = f.session_id
		WHERE s.subject_id = $1 AND f.state = 'active'
		ORDER BY f.issued_at
		FOR UPDATE OF f, s`, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var families []lockedFamily
	for rows.Next() {
		var family lockedFamily
		if err := rows.Scan(&family.familyID, &family.issuedAt); err != nil {
			return nil, err
		}
		families = append(families, family)
	}
	return families, rows.Err()
}

type lockedFamily struct {
	familyID string
	issuedAt time.Time
}

func loadCurrentFamily(ctx context.Context, tx pgx.Tx, tokenHash [32]byte) (durableFamily, error) {
	var family durableFamily
	var fingerprint []byte
	var currentHash []byte
	var authRevision, generation int64
	var familyState, sessionState string
	err := tx.QueryRow(ctx, `
		SELECT f.family_id, f.session_id, s.subject_id, s.client_id,
			s.fingerprint_hash, s.auth_revision, f.current_token_hash,
			f.generation, f.state, s.state, f.issued_at, f.last_used_at,
			f.idle_expires_at, f.absolute_expires_at
		FROM auth_identity.refresh_families f
		JOIN auth_identity.provider_sessions s ON s.session_id = f.session_id
		WHERE f.current_token_hash = $1
		FOR UPDATE OF f, s`, tokenHash[:]).Scan(
		&family.familyID, &family.sessionID, &family.subjectID, &family.clientID,
		&fingerprint, &authRevision, &currentHash, &generation,
		&familyState, &sessionState, &family.issuedAt, &family.lastUsedAt,
		&family.idleExpiresAt, &family.absoluteExpiresAt)
	if err != nil {
		return durableFamily{}, err
	}
	if len(fingerprint) != len(family.fingerprintHash) || len(currentHash) != len(family.currentTokenHash) {
		return durableFamily{}, errors.New("stored provider token or fingerprint has invalid length")
	}
	copy(family.fingerprintHash[:], fingerprint)
	copy(family.currentTokenHash[:], currentHash)
	family.authRevision = uint64(authRevision)
	family.generation = uint64(generation)
	family.familyState = FamilyState(familyState)
	family.sessionState = SessionState(sessionState)
	return family, nil
}

func lockFamily(ctx context.Context, tx pgx.Tx, familyID string) error {
	var locked string
	if err := tx.QueryRow(ctx, `SELECT family_id FROM auth_identity.refresh_families WHERE family_id = $1 FOR UPDATE`, familyID).Scan(&locked); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("lock refresh family: %w", err)
	}
	if _, err := tx.Exec(ctx, `SELECT session_id FROM auth_identity.provider_sessions WHERE family_id = $1 FOR UPDATE`, familyID); err != nil {
		return fmt.Errorf("lock provider session: %w", err)
	}
	return nil
}

func revokeFamilyTx(ctx context.Context, tx pgx.Tx, familyID string, familyState FamilyState, sessionState SessionState, at time.Time) error {
	if _, err := tx.Exec(ctx, `
		UPDATE auth_identity.refresh_families
		SET state = $2, revoked_at = $3
		WHERE family_id = $1`, familyID, familyState, at); err != nil {
		return fmt.Errorf("revoke refresh family: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE auth_identity.provider_sessions
		SET state = $2, revoked_at = $3
		WHERE family_id = $1`, familyID, sessionState, at); err != nil {
		return fmt.Errorf("revoke provider session: %w", err)
	}
	return nil
}

func finishRevocation(ctx context.Context, tx pgx.Tx, familyID string, familyState FamilyState, sessionState SessionState, at time.Time, reason error) error {
	if err := revokeFamilyTx(ctx, tx, familyID, familyState, sessionState, at); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: commit family revocation: %v", reason, err)
	}
	return reason
}
