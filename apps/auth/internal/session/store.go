package session

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidClient       = errors.New("OIDC client or redirect is invalid")
	ErrSubjectUnauthorized = errors.New("subject is not authorized for a provider session")
	ErrSessionNotFound     = errors.New("provider session not found")
	ErrSessionRevoked      = errors.New("provider session is revoked")
	ErrRefreshReuse        = errors.New("refresh token reuse detected")
	ErrRefreshExpired      = errors.New("refresh token expired")
	ErrAuthRevisionStale   = errors.New("provider session authorization revision is stale")
	ErrFingerprintMismatch = errors.New("provider session fingerprint mismatch")
	ErrInvalidSessionInput = errors.New("provider session input is invalid")
)

type Clock func() time.Time

type Authorizer interface {
	Authorize(context.Context, string, uint64) (bool, error)
}

type ClientRegistry interface {
	Validate(string, string) bool
}

type StaticClientRegistry map[string]map[string]struct{}

func (registry StaticClientRegistry) Validate(clientID, redirectURI string) bool {
	redirects, ok := registry[strings.TrimSpace(clientID)]
	if !ok || strings.TrimSpace(redirectURI) == "" {
		return false
	}
	_, ok = redirects[redirectURI]
	return ok
}

type Config struct {
	Authorizer            Authorizer
	Clients               ClientRegistry
	FingerprintKey        []byte
	Clock                 Clock
	IdleTTL               time.Duration
	AbsoluteTTL           time.Duration
	MaxFamiliesPerSubject int
}

type Store struct {
	mu                    sync.Mutex
	authorizer            Authorizer
	clients               ClientRegistry
	fingerprintKey        []byte
	clock                 Clock
	idleTTL               time.Duration
	absoluteTTL           time.Duration
	maxFamiliesPerSubject int
	sessions              map[string]*providerSession
	families              map[string]*refreshFamily
	tokenFamilies         map[[32]byte]string
}

type providerSession struct {
	SessionID         string
	SubjectID         string
	FamilyID          string
	ClientID          string
	FingerprintHash   [32]byte
	AuthRevision      uint64
	State             SessionState
	IssuedAt          time.Time
	LastUsedAt        time.Time
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
	RevokedAt         time.Time
}

type refreshFamily struct {
	FamilyID          string
	SessionID         string
	SubjectID         string
	ClientID          string
	FingerprintHash   [32]byte
	AuthRevision      uint64
	CurrentToken      [32]byte
	Generation        uint64
	State             FamilyState
	IssuedAt          time.Time
	LastUsedAt        time.Time
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
	UsedTokens        map[[32]byte]time.Time
}

type SessionState string
type FamilyState string

const (
	SessionActive  SessionState = "active"
	SessionRevoked SessionState = "revoked"
	SessionExpired SessionState = "expired"

	FamilyActive        FamilyState = "active"
	FamilyRevoked       FamilyState = "revoked"
	FamilyReuseDetected FamilyState = "reuse-detected"
	FamilyExpired       FamilyState = "expired"
)

type IssueInput struct {
	SubjectID    string
	AuthRevision uint64
	ClientID     string
	RedirectURI  string
	Fingerprint  FingerprintInput
}

type IssuedRefresh struct {
	SessionID         string
	FamilyID          string
	RefreshToken      string
	AuthRevision      uint64
	IssuedAt          time.Time
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
}

type RotateInput struct {
	RefreshToken string
	ClientID     string
	RedirectURI  string
	Fingerprint  FingerprintInput
}

type RotationResult struct {
	SessionID         string
	FamilyID          string
	RefreshToken      string
	AuthRevision      uint64
	Generation        uint64
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
}

type FingerprintInput struct {
	UserAgent string
	ClientIP  string
}

func NewStore(configuration Config) (*Store, error) {
	if configuration.Authorizer == nil || configuration.Clients == nil || len(configuration.FingerprintKey) < 32 {
		return nil, errors.New("session store requires authorizer, client registry, and 32-byte fingerprint key")
	}
	if configuration.Clock == nil {
		configuration.Clock = time.Now
	}
	if configuration.IdleTTL <= 0 || configuration.AbsoluteTTL <= configuration.IdleTTL || configuration.MaxFamiliesPerSubject < 1 || configuration.MaxFamiliesPerSubject > 32 {
		return nil, errors.New("session expiry and family limits are invalid")
	}
	return &Store{
		authorizer:            configuration.Authorizer,
		clients:               configuration.Clients,
		fingerprintKey:        append([]byte(nil), configuration.FingerprintKey...),
		clock:                 configuration.Clock,
		idleTTL:               configuration.IdleTTL,
		absoluteTTL:           configuration.AbsoluteTTL,
		maxFamiliesPerSubject: configuration.MaxFamiliesPerSubject,
		sessions:              make(map[string]*providerSession),
		families:              make(map[string]*refreshFamily),
		tokenFamilies:         make(map[[32]byte]string),
	}, nil
}

func (store *Store) Issue(ctx context.Context, input IssueInput) (IssuedRefresh, error) {
	if strings.TrimSpace(input.SubjectID) == "" || input.AuthRevision == 0 || !store.clients.Validate(input.ClientID, input.RedirectURI) {
		return IssuedRefresh{}, ErrInvalidClient
	}
	authorized, err := store.authorizer.Authorize(ctx, input.SubjectID, input.AuthRevision)
	if err != nil {
		return IssuedRefresh{}, ErrSubjectUnauthorized
	}
	if !authorized {
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
	family := &refreshFamily{
		FamilyID: familyID, SessionID: sessionID, SubjectID: input.SubjectID,
		ClientID: input.ClientID, FingerprintHash: fingerprint, AuthRevision: input.AuthRevision,
		CurrentToken: tokenHash, Generation: 1, State: FamilyActive, IssuedAt: now,
		LastUsedAt: now, IdleExpiresAt: now.Add(store.idleTTL), AbsoluteExpiresAt: now.Add(store.absoluteTTL),
		UsedTokens: make(map[[32]byte]time.Time),
	}
	session := &providerSession{
		SessionID: sessionID, SubjectID: input.SubjectID, FamilyID: familyID, ClientID: input.ClientID,
		FingerprintHash: fingerprint, AuthRevision: input.AuthRevision, State: SessionActive,
		IssuedAt: now, LastUsedAt: now, IdleExpiresAt: family.IdleExpiresAt, AbsoluteExpiresAt: family.AbsoluteExpiresAt,
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.revokeOldestIfBoundedLocked(input.SubjectID, now)
	store.sessions[sessionID] = session
	store.families[familyID] = family
	store.tokenFamilies[tokenHash] = familyID
	return IssuedRefresh{SessionID: sessionID, FamilyID: familyID, RefreshToken: refreshToken, AuthRevision: input.AuthRevision, IssuedAt: now, IdleExpiresAt: family.IdleExpiresAt, AbsoluteExpiresAt: family.AbsoluteExpiresAt}, nil
}

func (store *Store) Rotate(ctx context.Context, input RotateInput) (RotationResult, error) {
	if strings.TrimSpace(input.RefreshToken) == "" || !store.clients.Validate(input.ClientID, input.RedirectURI) {
		return RotationResult{}, ErrInvalidClient
	}
	tokenHash := HashRefreshToken(input.RefreshToken)
	fingerprint := DeriveFingerprint(store.fingerprintKey, input.Fingerprint)
	now := store.clock()
	store.mu.Lock()
	defer store.mu.Unlock()
	familyID, known := store.tokenFamilies[tokenHash]
	if !known {
		return RotationResult{}, ErrSessionNotFound
	}
	family := store.families[familyID]
	if family == nil {
		return RotationResult{}, ErrSessionNotFound
	}
	if _, reused := family.UsedTokens[tokenHash]; reused || family.CurrentToken != tokenHash {
		store.revokeFamilyLocked(family, FamilyReuseDetected, now)
		return RotationResult{}, ErrRefreshReuse
	}
	if family.State != FamilyActive {
		return RotationResult{}, stateError(family.State)
	}
	if family.ClientID != input.ClientID {
		store.revokeFamilyLocked(family, FamilyRevoked, now)
		return RotationResult{}, ErrInvalidClient
	}
	if !hmac.Equal(family.FingerprintHash[:], fingerprint[:]) {
		store.revokeFamilyLocked(family, FamilyRevoked, now)
		return RotationResult{}, ErrFingerprintMismatch
	}
	if !now.Before(family.AbsoluteExpiresAt) || !now.Before(family.IdleExpiresAt) {
		store.revokeFamilyLocked(family, FamilyExpired, now)
		return RotationResult{}, ErrRefreshExpired
	}
	authorized, err := store.authorizer.Authorize(ctx, family.SubjectID, family.AuthRevision)
	if err != nil || !authorized {
		store.revokeFamilyLocked(family, FamilyRevoked, now)
		return RotationResult{}, ErrAuthRevisionStale
	}
	refreshToken, nextHash, err := newRefreshToken()
	if err != nil {
		return RotationResult{}, err
	}
	family.UsedTokens[tokenHash] = now
	family.CurrentToken = nextHash
	family.Generation++
	family.LastUsedAt = now
	family.IdleExpiresAt = minTime(now.Add(store.idleTTL), family.AbsoluteExpiresAt)
	store.tokenFamilies[nextHash] = family.FamilyID
	session := store.sessions[family.SessionID]
	if session == nil || session.State != SessionActive {
		store.revokeFamilyLocked(family, FamilyRevoked, now)
		return RotationResult{}, ErrSessionRevoked
	}
	session.LastUsedAt = now
	session.IdleExpiresAt = family.IdleExpiresAt
	return RotationResult{SessionID: family.SessionID, FamilyID: family.FamilyID, RefreshToken: refreshToken, AuthRevision: family.AuthRevision, Generation: family.Generation, IdleExpiresAt: family.IdleExpiresAt, AbsoluteExpiresAt: family.AbsoluteExpiresAt}, nil
}

func (store *Store) RevokeFamily(_ context.Context, subjectID, familyID string) error {
	now := store.clock()
	store.mu.Lock()
	defer store.mu.Unlock()
	family := store.families[familyID]
	if family == nil || family.SubjectID != subjectID {
		return ErrSessionNotFound
	}
	store.revokeFamilyLocked(family, FamilyRevoked, now)
	return nil
}

func (store *Store) RevokeAll(_ context.Context, subjectID string) int {
	now := store.clock()
	store.mu.Lock()
	defer store.mu.Unlock()
	revoked := 0
	for _, family := range store.families {
		if family.SubjectID == subjectID && family.State == FamilyActive {
			store.revokeFamilyLocked(family, FamilyRevoked, now)
			revoked++
		}
	}
	return revoked
}

func (store *Store) RevokeAllSessions(ctx context.Context, subjectID string) error {
	store.RevokeAll(ctx, subjectID)
	return nil
}

func (store *Store) Cleanup(now time.Time) int {
	store.mu.Lock()
	defer store.mu.Unlock()
	expired := 0
	for _, family := range store.families {
		if family.State == FamilyActive && (!now.Before(family.AbsoluteExpiresAt) || !now.Before(family.IdleExpiresAt)) {
			store.revokeFamilyLocked(family, FamilyExpired, now)
			expired++
		}
	}
	return expired
}

type Snapshot struct {
	SessionID         string
	FamilyID          string
	SubjectID         string
	ClientID          string
	AuthRevision      uint64
	SessionState      SessionState
	FamilyState       FamilyState
	Generation        uint64
	IssuedAt          time.Time
	LastUsedAt        time.Time
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
	FingerprintHash   [32]byte
}

func (store *Store) Snapshot(familyID string) (Snapshot, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	family := store.families[familyID]
	if family == nil {
		return Snapshot{}, ErrSessionNotFound
	}
	session := store.sessions[family.SessionID]
	if session == nil {
		return Snapshot{}, ErrSessionNotFound
	}
	return Snapshot{SessionID: session.SessionID, FamilyID: family.FamilyID, SubjectID: family.SubjectID, ClientID: family.ClientID, AuthRevision: family.AuthRevision, SessionState: session.State, FamilyState: family.State, Generation: family.Generation, IssuedAt: family.IssuedAt, LastUsedAt: family.LastUsedAt, IdleExpiresAt: family.IdleExpiresAt, AbsoluteExpiresAt: family.AbsoluteExpiresAt, FingerprintHash: family.FingerprintHash}, nil
}

func DeriveFingerprint(key []byte, input FingerprintInput) [32]byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("as360-session-fingerprint-v1\x00"))
	_, _ = mac.Write([]byte(strings.TrimSpace(input.UserAgent)))
	_, _ = mac.Write([]byte("\x00"))
	_, _ = mac.Write([]byte(strings.TrimSpace(input.ClientIP)))
	var result [32]byte
	copy(result[:], mac.Sum(nil))
	return result
}

func HashRefreshToken(token string) [32]byte {
	return sha256.Sum256([]byte("as360-refresh-token-v1\x00" + token))
}

func newRefreshToken() (string, [32]byte, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", [32]byte{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(randomBytes)
	return token, HashRefreshToken(token), nil
}

func newPrefixedID(prefix string) (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

func (store *Store) revokeFamilyLocked(family *refreshFamily, state FamilyState, now time.Time) {
	family.State = state
	session := store.sessions[family.SessionID]
	if session != nil {
		session.State = SessionRevoked
		session.RevokedAt = now
	}
}

func (store *Store) revokeOldestIfBoundedLocked(subjectID string, now time.Time) {
	active := make([]*refreshFamily, 0, store.maxFamiliesPerSubject+1)
	for _, family := range store.families {
		if family.SubjectID == subjectID && family.State == FamilyActive {
			active = append(active, family)
		}
	}
	for len(active) >= store.maxFamiliesPerSubject {
		oldest := active[0]
		for _, candidate := range active[1:] {
			if candidate.IssuedAt.Before(oldest.IssuedAt) {
				oldest = candidate
			}
		}
		store.revokeFamilyLocked(oldest, FamilyRevoked, now)
		filtered := active[:0]
		for _, candidate := range active {
			if candidate != oldest {
				filtered = append(filtered, candidate)
			}
		}
		active = filtered
	}
}

func stateError(state FamilyState) error {
	if state == FamilyExpired {
		return ErrRefreshExpired
	}
	return ErrSessionRevoked
}

func minTime(left, right time.Time) time.Time {
	if left.Before(right) {
		return left
	}
	return right
}
