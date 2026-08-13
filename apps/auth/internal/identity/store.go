package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/password"
	"github.com/MarlonJD/aviaSurveil360/apps/auth/internal/throttle"
)

type Clock func() time.Time

// SessionRevoker is implemented by the provider-session adapter. Identity
// security-state changes must invalidate every prior provider family.
type SessionRevoker interface {
	RevokeAllSessions(context.Context, string) error
}

type Store struct {
	mu             sync.RWMutex
	accounts       map[string]*account
	identifiers    map[string]string
	invitations    map[string]*invitation
	hasher         *password.Hasher
	passwordPolicy password.Policy
	limiter        throttle.Limiter
	sessionRevoker SessionRevoker
	authPolicy     AuthenticationPolicy
	clock          Clock
}

type account struct {
	subjectID       string
	email           string
	username        string
	state           AccountState
	emailVerified   bool
	passwordHash    string
	passwordHistory []string
	authRevision    uint64
	failedAttempts  int
	lockedUntil     time.Time
	createdAt       time.Time
	updatedAt       time.Time
}

type invitation struct {
	subjectID     string
	tokenHash     [32]byte
	issuedAt      time.Time
	expiresAt     time.Time
	resendCount   int
	invalidatedAt time.Time
	consumedAt    time.Time
}

type Config struct {
	Hasher               *password.Hasher
	PasswordPolicy       password.Policy
	Limiter              throttle.Limiter
	SessionRevoker       SessionRevoker
	AuthenticationPolicy AuthenticationPolicy
	Clock                Clock
}

type AuthenticationPolicy struct {
	Window          time.Duration
	GlobalLimit     int
	IdentifierLimit int
	BrowserLimit    int
	ClientLimit     int
	SubjectLimit    int
}

func DefaultAuthenticationPolicy() AuthenticationPolicy {
	return AuthenticationPolicy{
		Window:          time.Minute,
		GlobalLimit:     600,
		IdentifierLimit: 30,
		BrowserLimit:    60,
		ClientLimit:     120,
		SubjectLimit:    20,
	}
}

func NewStore(configuration Config) (*Store, error) {
	if configuration.Hasher == nil || configuration.Limiter == nil {
		return nil, errors.New("identity store requires a password hasher and limiter")
	}
	if configuration.PasswordPolicy.MinBytes < 1 || configuration.PasswordPolicy.MaxBytes < configuration.PasswordPolicy.MinBytes {
		return nil, errors.New("identity store password policy is invalid")
	}
	if configuration.Clock == nil {
		configuration.Clock = time.Now
	}
	if configuration.AuthenticationPolicy.Window <= 0 {
		configuration.AuthenticationPolicy = DefaultAuthenticationPolicy()
	}
	if configuration.AuthenticationPolicy.GlobalLimit < 1 || configuration.AuthenticationPolicy.IdentifierLimit < 1 || configuration.AuthenticationPolicy.BrowserLimit < 1 || configuration.AuthenticationPolicy.ClientLimit < 1 || configuration.AuthenticationPolicy.SubjectLimit < 1 {
		return nil, errors.New("identity authentication admission policy is invalid")
	}
	return &Store{
		accounts:       make(map[string]*account),
		identifiers:    make(map[string]string),
		invitations:    make(map[string]*invitation),
		hasher:         configuration.Hasher,
		passwordPolicy: configuration.PasswordPolicy,
		limiter:        configuration.Limiter,
		sessionRevoker: configuration.SessionRevoker,
		authPolicy:     configuration.AuthenticationPolicy,
		clock:          configuration.Clock,
	}, nil
}

type InvitationInput struct {
	Email    string
	Username string
}

func (store *Store) ProvisionInvitation(_ context.Context, input InvitationInput) (AccountSnapshot, InvitationSnapshot, error) {
	email, err := NormalizeIdentifier(IdentifierEmail, input.Email)
	if err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, err
	}
	username := ""
	if strings.TrimSpace(input.Username) != "" {
		normalizedUsername, usernameErr := NormalizeIdentifier(IdentifierUsername, input.Username)
		if usernameErr != nil {
			return AccountSnapshot{}, InvitationSnapshot{}, usernameErr
		}
		username = normalizedUsername.Normalized
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, exists := store.identifiers[email.Normalized]; exists || (username != "" && store.identifierExistsLocked(username)) {
		return AccountSnapshot{}, InvitationSnapshot{}, ErrDuplicateIdentifier
	}
	subjectID, err := newUniqueSubject(store.accounts)
	if err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, err
	}
	now := store.clock()
	record := &account{
		subjectID:    subjectID,
		email:        email.Normalized,
		username:     username,
		state:        AccountInvited,
		authRevision: 1,
		createdAt:    now,
		updatedAt:    now,
	}
	rawToken, tokenHash, err := newRandomTokenHash()
	if err != nil {
		return AccountSnapshot{}, InvitationSnapshot{}, err
	}
	invite := &invitation{subjectID: subjectID, tokenHash: tokenHash, issuedAt: now, expiresAt: now.Add(24 * time.Hour)}
	store.accounts[subjectID] = record
	store.identifiers[email.Normalized] = subjectID
	if username != "" {
		store.identifiers[username] = subjectID
	}
	store.invitations[subjectID] = invite
	snapshot := invite.snapshot()
	snapshot.Token = rawToken
	return record.snapshot(), snapshot, nil
}

func (store *Store) RegisterPublic(context.Context, InvitationInput) error {
	return ErrPublicRegistrationDisabled
}

func (store *Store) SetEmailVerified(ctx context.Context, subjectID string, expectedRevision uint64) (AccountSnapshot, error) {
	store.mu.Lock()
	record, ok := store.accounts[subjectID]
	if !ok {
		store.mu.Unlock()
		return AccountSnapshot{}, ErrAccountNotFound
	}
	if record.authRevision != expectedRevision {
		store.mu.Unlock()
		return AccountSnapshot{}, ErrRevisionConflict
	}
	if record.state == AccountDeleted || record.state == AccountDeletionPending {
		store.mu.Unlock()
		return AccountSnapshot{}, ErrInvalidTransition
	}
	record.emailVerified = true
	record.authRevision++
	record.updatedAt = store.clock()
	snapshot := record.snapshot()
	store.mu.Unlock()
	if err := store.revokeSessions(ctx, subjectID); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

func (store *Store) Activate(ctx context.Context, subjectID string, expectedRevision uint64, newPassword []byte) (AccountSnapshot, error) {
	store.mu.Lock()
	record, ok := store.accounts[subjectID]
	if !ok {
		store.mu.Unlock()
		return AccountSnapshot{}, ErrAccountNotFound
	}
	if record.authRevision != expectedRevision {
		store.mu.Unlock()
		return AccountSnapshot{}, ErrRevisionConflict
	}
	if record.state != AccountInvited || !record.emailVerified {
		store.mu.Unlock()
		return AccountSnapshot{}, ErrEmailNotVerified
	}
	history := append([]string(nil), record.passwordHistory...)
	current := record.passwordHash
	store.mu.Unlock()
	if err := store.passwordPolicy.Validate(newPassword, store.hasher, current, history); err != nil {
		return AccountSnapshot{}, err
	}
	hash, err := store.hasher.Hash(newPassword)
	if err != nil {
		return AccountSnapshot{}, err
	}
	store.mu.Lock()
	record, ok = store.accounts[subjectID]
	if !ok {
		return AccountSnapshot{}, ErrAccountNotFound
	}
	if record.authRevision != expectedRevision || record.state != AccountInvited {
		return AccountSnapshot{}, ErrRevisionConflict
	}
	record.passwordHash = hash
	record.state = AccountActive
	record.authRevision++
	record.updatedAt = store.clock()
	snapshot := record.snapshot()
	store.mu.Unlock()
	if err := store.revokeSessions(ctx, subjectID); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

type AuthenticationRequest struct {
	Identifier     string
	Password       []byte
	BrowserBinding string
	DeviceKey      string
}

type AuthenticationResult struct {
	Account AccountSnapshot
}

func (store *Store) Authenticate(ctx context.Context, request AuthenticationRequest) (AuthenticationResult, error) {
	identifier, identifierErr := DetectIdentifier(request.Identifier)
	store.mu.RLock()
	var record *account
	if identifierErr == nil {
		if subjectID := store.identifiers[identifier.Normalized]; subjectID != "" {
			record = store.accounts[subjectID]
		}
	}
	var subjectID, encodedHash string
	var revision uint64
	if record != nil {
		subjectID, encodedHash, revision = record.subjectID, record.passwordHash, record.authRevision
	}
	store.mu.RUnlock()

	identifierValue := request.Identifier
	if identifierErr == nil {
		identifierValue = identifier.Normalized
	}
	browserBinding := strings.TrimSpace(request.BrowserBinding)
	if browserBinding == "" {
		browserBinding = "missing"
	}
	clientKey := strings.TrimSpace(request.DeviceKey)
	if clientKey == "" {
		clientKey = "missing"
	}
	rules := []throttle.Rule{
		{Key: throttle.Key("auth:global", "password"), Window: store.authPolicy.Window, Limit: store.authPolicy.GlobalLimit, Global: true},
		{Key: throttle.Key("auth:identifier", identifierValue), Window: store.authPolicy.Window, Limit: store.authPolicy.IdentifierLimit},
		{Key: throttle.Key("auth:browser", browserBinding), Window: store.authPolicy.Window, Limit: store.authPolicy.BrowserLimit},
		{Key: throttle.Key("auth:client", clientKey), Window: store.authPolicy.Window, Limit: store.authPolicy.ClientLimit},
	}
	if subjectID != "" {
		rules = append(rules, throttle.Rule{Key: throttle.Key("auth:subject", subjectID), Window: store.authPolicy.Window, Limit: store.authPolicy.SubjectLimit})
	}
	if err := store.limiter.Allow(ctx, rules...); err != nil {
		if errors.Is(err, throttle.ErrRateLimited) {
			return AuthenticationResult{}, ErrAuthenticationRateLimited
		}
		return AuthenticationResult{}, ErrAuthenticationUnavailable
	}

	if encodedHash == "" {
		encodedHash = store.hasher.DummyHash()
	}
	verified, verifyErr := store.hasher.Verify(encodedHash, request.Password)
	if verifyErr != nil {
		return AuthenticationResult{}, ErrAuthenticationFailed
	}

	store.mu.Lock()
	if subjectID == "" {
		store.mu.Unlock()
		return AuthenticationResult{}, ErrAuthenticationFailed
	}
	record = store.accounts[subjectID]
	if record == nil || record.authRevision != revision || record.passwordHash != encodedHash || !verified || !store.loginAllowedLocked(record) {
		if record != nil && record.authRevision == revision && record.passwordHash == encodedHash && !verified {
			store.recordFailureLocked(record)
		}
		store.mu.Unlock()
		return AuthenticationResult{}, ErrAuthenticationFailed
	}
	if record.state == AccountLocked {
		record.state = AccountActive
	}
	record.failedAttempts = 0
	record.lockedUntil = time.Time{}
	record.updatedAt = store.clock()
	result := AuthenticationResult{Account: record.snapshot()}
	store.mu.Unlock()
	return result, nil
}

func (store *Store) ChangePassword(ctx context.Context, subjectID string, expectedRevision uint64, currentPassword, newPassword []byte) (AccountSnapshot, error) {
	store.mu.RLock()
	record, ok := store.accounts[subjectID]
	if !ok {
		store.mu.RUnlock()
		return AccountSnapshot{}, ErrAccountNotFound
	}
	if record.authRevision != expectedRevision || record.state != AccountActive {
		store.mu.RUnlock()
		return AccountSnapshot{}, ErrRevisionConflict
	}
	currentHash := record.passwordHash
	history := append([]string(nil), record.passwordHistory...)
	store.mu.RUnlock()
	validCurrent, err := store.hasher.Verify(currentHash, currentPassword)
	if err != nil || !validCurrent {
		return AccountSnapshot{}, ErrAuthenticationFailed
	}
	if err := store.passwordPolicy.Validate(newPassword, store.hasher, currentHash, history); err != nil {
		return AccountSnapshot{}, err
	}
	newHash, err := store.hasher.Hash(newPassword)
	if err != nil {
		return AccountSnapshot{}, err
	}
	store.mu.Lock()
	record, ok = store.accounts[subjectID]
	if !ok {
		store.mu.Unlock()
		return AccountSnapshot{}, ErrAccountNotFound
	}
	if record.authRevision != expectedRevision || record.passwordHash != currentHash {
		store.mu.Unlock()
		return AccountSnapshot{}, ErrRevisionConflict
	}
	record.passwordHistory = prependHistory(record.passwordHash, record.passwordHistory, 5)
	record.passwordHash = newHash
	record.authRevision++
	record.failedAttempts = 0
	record.lockedUntil = time.Time{}
	record.updatedAt = store.clock()
	snapshot := record.snapshot()
	store.mu.Unlock()
	if err := store.revokeSessions(ctx, subjectID); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

func (store *Store) Transition(ctx context.Context, subjectID string, expectedRevision uint64, target AccountState) (AccountSnapshot, error) {
	store.mu.Lock()
	record, ok := store.accounts[subjectID]
	if !ok {
		store.mu.Unlock()
		return AccountSnapshot{}, ErrAccountNotFound
	}
	if record.authRevision != expectedRevision {
		store.mu.Unlock()
		return AccountSnapshot{}, ErrRevisionConflict
	}
	if !allowedTransition(record.state, target) {
		store.mu.Unlock()
		return AccountSnapshot{}, ErrInvalidTransition
	}
	record.state = target
	record.authRevision++
	record.updatedAt = store.clock()
	snapshot := record.snapshot()
	store.mu.Unlock()
	if err := store.revokeSessions(ctx, subjectID); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

func (store *Store) revokeSessions(ctx context.Context, subjectID string) error {
	if store.sessionRevoker == nil {
		return nil
	}
	if err := store.sessionRevoker.RevokeAllSessions(ctx, subjectID); err != nil {
		return fmt.Errorf("%w: %v", ErrSessionRevocationUnavailable, err)
	}
	return nil
}

func (store *Store) Snapshot(subjectID string) (AccountSnapshot, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	record, ok := store.accounts[subjectID]
	if !ok {
		return AccountSnapshot{}, ErrAccountNotFound
	}
	return record.snapshot(), nil
}

func (store *Store) loginAllowedLocked(record *account) bool {
	if record == nil || !record.emailVerified {
		return false
	}
	if record.state != AccountActive && record.state != AccountLocked {
		return false
	}
	if !record.lockedUntil.IsZero() && record.lockedUntil.After(store.clock()) {
		return false
	}
	return true
}

func (store *Store) recordFailureLocked(record *account) {
	if record == nil || !record.emailVerified || record.state != AccountActive {
		return
	}
	now := store.clock()
	if !record.lockedUntil.IsZero() {
		if now.Before(record.lockedUntil) {
			return
		}
		record.failedAttempts = 0
		record.lockedUntil = time.Time{}
	}
	record.failedAttempts++
	if record.failedAttempts >= 5 {
		record.lockedUntil = now.Add(15 * time.Minute)
	}
	record.updatedAt = now
}

func allowedTransition(current, target AccountState) bool {
	if current == target {
		return false
	}
	switch current {
	case AccountInvited:
		return target == AccountDisabled || target == AccountDeletionPending
	case AccountActive:
		return target == AccountDisabled || target == AccountSuspended || target == AccountDeletionPending
	case AccountDisabled, AccountSuspended, AccountLocked:
		return target == AccountActive || target == AccountDeletionPending
	case AccountDeletionPending:
		return target == AccountDeleted || target == AccountActive
	case AccountDeleted:
		return false
	default:
		return false
	}
}

func (record *account) snapshot() AccountSnapshot {
	return AccountSnapshot{
		SubjectID:      record.subjectID,
		Email:          record.email,
		Username:       record.username,
		State:          record.state,
		EmailVerified:  record.emailVerified,
		AuthRevision:   record.authRevision,
		FailedAttempts: record.failedAttempts,
		LockedUntil:    record.lockedUntil,
		CreatedAt:      record.createdAt,
		UpdatedAt:      record.updatedAt,
	}
}

func (invite *invitation) snapshot() InvitationSnapshot {
	return InvitationSnapshot{SubjectID: invite.subjectID, IssuedAt: invite.issuedAt, ExpiresAt: invite.expiresAt, ResendCount: invite.resendCount, InvalidatedAt: invite.invalidatedAt, ConsumedAt: invite.consumedAt}
}

func (store *Store) identifierExistsLocked(value string) bool {
	_, exists := store.identifiers[value]
	return exists
}

func newUniqueSubject(accounts map[string]*account) (string, error) {
	for attempt := 0; attempt < 8; attempt++ {
		subjectID, err := NewSubjectID()
		if err != nil {
			return "", err
		}
		if _, exists := accounts[subjectID]; !exists {
			return subjectID, nil
		}
	}
	return "", errors.New("unable to allocate unique subject")
}

func newRandomTokenHash() (string, [32]byte, error) {
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", [32]byte{}, err
	}
	raw := base64.RawURLEncoding.EncodeToString(token[:])
	return raw, sha256.Sum256([]byte(raw)), nil
}

func prependHistory(current string, history []string, maximum int) []string {
	result := make([]string, 0, maximum)
	if current != "" {
		result = append(result, current)
	}
	result = append(result, history...)
	if len(result) > maximum {
		result = result[:maximum]
	}
	return result
}
