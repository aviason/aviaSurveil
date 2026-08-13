package mfa

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrFactorNotFound   = errors.New("MFA factor not found")
	ErrFactorDisabled   = errors.New("MFA factor is not enabled")
	ErrInvalidCode      = errors.New("MFA code is invalid")
	ErrCodeReplayed     = errors.New("MFA code was already used")
	ErrRecoveryInvalid  = errors.New("recovery code is invalid")
	ErrRecoveryLocked   = errors.New("recovery code attempts are temporarily locked")
	ErrRecoveryConsumed = errors.New("recovery code was already consumed")
	ErrMFAUnavailable   = errors.New("MFA secret protection unavailable")
	ErrRevisionConflict = errors.New("MFA auth revision conflict")
)

type Clock func() time.Time

// SessionRevoker is the provider credential boundary used when a privileged
// MFA mutation invalidates existing authentication material.
type SessionRevoker interface {
	RevokeAllSessions(context.Context, string) error
}

type Config struct {
	EncryptionKey         []byte
	SessionRevoker        SessionRevoker
	Clock                 Clock
	Period                time.Duration
	Window                int
	Digits                int
	MaxRecoveryFailures   int
	RecoveryFailureWindow time.Duration
	RecoveryLockDuration  time.Duration
}

type Store struct {
	mu                    sync.Mutex
	key                   []byte
	clock                 Clock
	period                time.Duration
	window                int
	digits                int
	maxRecoveryFailures   int
	recoveryFailureWindow time.Duration
	recoveryLockDuration  time.Duration
	factors               map[string]*factor
}

type factor struct {
	secretCiphertext      []byte
	enabled               bool
	lastCounter           int64
	recoveryHashes        map[[32]byte]struct{}
	recoveryFailures      int
	recoveryWindowStarted time.Time
	recoveryLockedUntil   time.Time
}

type Enrollment struct {
	SubjectID  string
	Secret     string
	OTPAuthURI string
}

type Snapshot struct {
	SubjectID       string
	Enabled         bool
	RecoveryCount   int
	LastUsedCounter int64
}

func NewStore(configuration Config) (*Store, error) {
	if len(configuration.EncryptionKey) != 32 {
		return nil, errors.New("MFA store requires a 32-byte encryption key")
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
	if configuration.RecoveryFailureWindow == 0 {
		configuration.RecoveryFailureWindow = 15 * time.Minute
	}
	if configuration.RecoveryLockDuration == 0 {
		configuration.RecoveryLockDuration = 15 * time.Minute
	}
	if configuration.Period < 15*time.Second || configuration.Period > 5*time.Minute || configuration.Window < 0 || configuration.Window > 2 || (configuration.Digits != 0 && configuration.Digits != 6 && configuration.Digits != 8) || configuration.MaxRecoveryFailures < 1 || configuration.MaxRecoveryFailures > 20 || configuration.RecoveryFailureWindow < time.Minute || configuration.RecoveryFailureWindow > 24*time.Hour || configuration.RecoveryLockDuration < time.Minute || configuration.RecoveryLockDuration > 24*time.Hour {
		return nil, errors.New("MFA timing or recovery policy is invalid")
	}
	if configuration.Digits == 0 {
		configuration.Digits = 6
	}
	return &Store{
		key: append([]byte(nil), configuration.EncryptionKey...), clock: configuration.Clock,
		period: configuration.Period, window: configuration.Window, digits: configuration.Digits,
		maxRecoveryFailures: configuration.MaxRecoveryFailures, recoveryFailureWindow: configuration.RecoveryFailureWindow, recoveryLockDuration: configuration.RecoveryLockDuration, factors: make(map[string]*factor),
	}, nil
}

func (store *Store) Enroll(subjectID, issuer, accountLabel string) (Enrollment, error) {
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
	store.mu.Lock()
	defer store.mu.Unlock()
	if existing := store.factors[subjectID]; existing != nil && existing.enabled {
		return Enrollment{}, errors.New("MFA factor already enabled")
	}
	store.factors[subjectID] = &factor{secretCiphertext: ciphertext, lastCounter: -1, recoveryHashes: make(map[[32]byte]struct{})}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)
	return Enrollment{SubjectID: subjectID, Secret: secret, OTPAuthURI: "otpauth://totp/" + urlEscape(accountLabel) + "?secret=" + secret + "&issuer=" + urlEscape(issuer) + "&algorithm=SHA1&digits=" + strconv.Itoa(store.digits) + "&period=" + strconv.FormatInt(int64(store.period/time.Second), 10)}, nil
}

func (store *Store) ConfirmEnrollment(subjectID, code string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	factorRecord := store.factors[subjectID]
	if factorRecord == nil {
		return ErrFactorNotFound
	}
	secret, err := store.decrypt(subjectID, factorRecord.secretCiphertext)
	if err != nil {
		return ErrMFAUnavailable
	}
	counter, ok := store.matchCounter(secret, code, store.counter(store.clock()))
	if !ok {
		return ErrInvalidCode
	}
	factorRecord.enabled = true
	factorRecord.lastCounter = counter
	return nil
}

func (store *Store) Verify(subjectID, code string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	factorRecord := store.factors[subjectID]
	if factorRecord == nil {
		return ErrFactorNotFound
	}
	if !factorRecord.enabled {
		return ErrFactorDisabled
	}
	secret, err := store.decrypt(subjectID, factorRecord.secretCiphertext)
	if err != nil {
		return ErrMFAUnavailable
	}
	counter, ok := store.matchCounter(secret, code, store.counter(store.clock()))
	if !ok {
		return ErrInvalidCode
	}
	if counter <= factorRecord.lastCounter {
		return ErrCodeReplayed
	}
	factorRecord.lastCounter = counter
	return nil
}

func (store *Store) GenerateRecoveryCodes(subjectID string, count int) ([]string, error) {
	if count < 1 || count > 20 {
		return nil, errors.New("recovery code count is invalid")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	factorRecord := store.factors[subjectID]
	if factorRecord == nil || !factorRecord.enabled {
		return nil, ErrFactorDisabled
	}
	codes := make([]string, 0, count)
	for len(codes) < count {
		raw := make([]byte, 10)
		if _, err := rand.Read(raw); err != nil {
			return nil, err
		}
		code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
		hash := hashRecoveryCode(code)
		if _, exists := factorRecord.recoveryHashes[hash]; exists {
			continue
		}
		factorRecord.recoveryHashes[hash] = struct{}{}
		codes = append(codes, code)
	}
	factorRecord.recoveryFailures = 0
	factorRecord.recoveryWindowStarted = time.Time{}
	factorRecord.recoveryLockedUntil = time.Time{}
	return codes, nil
}

func (store *Store) ConsumeRecoveryCode(subjectID, code string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	factorRecord := store.factors[subjectID]
	if factorRecord == nil || !factorRecord.enabled {
		return ErrFactorDisabled
	}
	now := store.clock().UTC()
	if !factorRecord.recoveryLockedUntil.IsZero() && now.Before(factorRecord.recoveryLockedUntil) {
		return ErrRecoveryLocked
	}
	if (!factorRecord.recoveryLockedUntil.IsZero() && !now.Before(factorRecord.recoveryLockedUntil)) || (!factorRecord.recoveryWindowStarted.IsZero() && !now.Before(factorRecord.recoveryWindowStarted.Add(store.recoveryFailureWindow))) {
		factorRecord.recoveryFailures = 0
		factorRecord.recoveryWindowStarted = time.Time{}
		factorRecord.recoveryLockedUntil = time.Time{}
	}
	hash := hashRecoveryCode(code)
	if _, exists := factorRecord.recoveryHashes[hash]; !exists {
		if factorRecord.recoveryWindowStarted.IsZero() {
			factorRecord.recoveryWindowStarted = now
		}
		factorRecord.recoveryFailures++
		if factorRecord.recoveryFailures >= store.maxRecoveryFailures {
			factorRecord.recoveryLockedUntil = now.Add(store.recoveryLockDuration)
			return ErrRecoveryLocked
		}
		return ErrRecoveryInvalid
	}
	delete(factorRecord.recoveryHashes, hash)
	factorRecord.recoveryFailures = 0
	factorRecord.recoveryWindowStarted = time.Time{}
	factorRecord.recoveryLockedUntil = time.Time{}
	return nil
}

func (store *Store) Reset(subjectID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, exists := store.factors[subjectID]; !exists {
		return ErrFactorNotFound
	}
	delete(store.factors, subjectID)
	return nil
}

func (store *Store) Snapshot(subjectID string) (Snapshot, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	factorRecord := store.factors[subjectID]
	if factorRecord == nil {
		return Snapshot{}, ErrFactorNotFound
	}
	return Snapshot{SubjectID: subjectID, Enabled: factorRecord.enabled, RecoveryCount: len(factorRecord.recoveryHashes), LastUsedCounter: factorRecord.lastCounter}, nil
}

func (store *Store) CurrentCodeForTesting(subjectID string, at time.Time) (string, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	factorRecord := store.factors[subjectID]
	if factorRecord == nil {
		return "", ErrFactorNotFound
	}
	secret, err := store.decrypt(subjectID, factorRecord.secretCiphertext)
	if err != nil {
		return "", ErrMFAUnavailable
	}
	return generateCode(secret, store.counter(at), store.digits), nil
}

func (store *Store) counter(at time.Time) int64 {
	return at.Unix() / int64(store.period/time.Second)
}

func (store *Store) matchCounter(secret []byte, code string, current int64) (int64, bool) {
	code = strings.TrimSpace(code)
	for offset := -store.window; offset <= store.window; offset++ {
		counter := current + int64(offset)
		if counter >= 0 && constantTimeString(generateCode(secret, counter, store.digits), code) {
			return counter, true
		}
	}
	return 0, false
}

func generateCode(secret []byte, counter int64, digits int) string {
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], uint64(counter))
	mac := hmac.New(sha1.New, secret)
	_, _ = mac.Write(message[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	number := (uint32(sum[offset])&0x7f)<<24 | uint32(sum[offset+1])<<16 | uint32(sum[offset+2])<<8 | uint32(sum[offset+3])
	modulus := uint32(1000000)
	if digits == 8 {
		modulus = 100000000
	}
	return fmt.Sprintf("%0*d", digits, number%modulus)
}

func (store *Store) encrypt(subjectID string, plaintext []byte) ([]byte, error) {
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

func (store *Store) decrypt(subjectID string, ciphertext []byte) ([]byte, error) {
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

func hashRecoveryCode(code string) [32]byte {
	return sha256.Sum256([]byte("as360-recovery-v1\x00" + strings.ToUpper(strings.TrimSpace(code))))
}

func constantTimeString(expected, actual string) bool {
	if len(expected) != len(actual) {
		return false
	}
	return hmac.Equal([]byte(expected), []byte(actual))
}

func urlEscape(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "%20")
	return value
}
