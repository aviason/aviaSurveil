package challenge

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidChallenge = errors.New("challenge is invalid")
	ErrChallengeExpired = errors.New("challenge is expired")
	ErrChallengeUsed    = errors.New("challenge was already used")
	ErrChallengeLocked  = errors.New("challenge attempts are exhausted")
)

type Clock func() time.Time

type Purpose string

const (
	PurposeEmailVerification Purpose = "email-verification"
	PurposePasswordReset     Purpose = "password-reset"
	PurposeMFARecovery       Purpose = "mfa-recovery"
	PurposeAdminRecovery     Purpose = "admin-recovery"
)

type Config struct {
	Clock Clock
}

type Store struct {
	mu         sync.Mutex
	clock      Clock
	challenges map[[32]byte]*record
}

type record struct {
	subject  string
	purpose  Purpose
	expires  time.Time
	attempts int
	max      int
	used     bool
}

type Issued struct {
	Subject string
	Purpose Purpose
	Token   string
	Expires time.Time
}

func NewStore(configuration Config) *Store {
	if configuration.Clock == nil {
		configuration.Clock = time.Now
	}
	return &Store{clock: configuration.Clock, challenges: make(map[[32]byte]*record)}
}

func (store *Store) Issue(subject string, purpose Purpose, ttl time.Duration, maxAttempts int) (Issued, error) {
	if strings.TrimSpace(subject) == "" || purpose == "" || ttl <= 0 || ttl > 72*time.Hour || maxAttempts < 1 || maxAttempts > 10 {
		return Issued{}, ErrInvalidChallenge
	}
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return Issued{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	hash := hashToken(token)
	expires := store.clock().Add(ttl)
	store.mu.Lock()
	store.challenges[hash] = &record{subject: subject, purpose: purpose, expires: expires, max: maxAttempts}
	store.mu.Unlock()
	return Issued{Subject: subject, Purpose: purpose, Token: token, Expires: expires}, nil
}

func (store *Store) Consume(subject string, purpose Purpose, token string) error {
	hash := hashToken(token)
	store.mu.Lock()
	defer store.mu.Unlock()
	challenge := store.challenges[hash]
	if challenge == nil || challenge.subject != subject || challenge.purpose != purpose {
		return ErrInvalidChallenge
	}
	if challenge.used {
		return ErrChallengeUsed
	}
	if !store.clock().Before(challenge.expires) {
		return ErrChallengeExpired
	}
	if challenge.attempts >= challenge.max {
		return ErrChallengeLocked
	}
	challenge.used = true
	return nil
}

func (store *Store) RejectAttempt(subject string, purpose Purpose, token string) error {
	hash := hashToken(token)
	store.mu.Lock()
	defer store.mu.Unlock()
	challenge := store.challenges[hash]
	if challenge == nil || challenge.subject != subject || challenge.purpose != purpose {
		return ErrInvalidChallenge
	}
	if challenge.used {
		return ErrChallengeUsed
	}
	if !store.clock().Before(challenge.expires) {
		return ErrChallengeExpired
	}
	if challenge.attempts >= challenge.max {
		return ErrChallengeLocked
	}
	challenge.attempts++
	if challenge.attempts >= challenge.max {
		return ErrChallengeLocked
	}
	return ErrInvalidChallenge
}

func (store *Store) Invalidate(subject string, purpose Purpose) int {
	store.mu.Lock()
	defer store.mu.Unlock()
	count := 0
	for _, challenge := range store.challenges {
		if challenge.subject == subject && challenge.purpose == purpose && !challenge.used {
			challenge.used = true
			count++
		}
	}
	return count
}

func (store *Store) Cleanup(at time.Time, limit int) int {
	if limit < 1 {
		return 0
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	keys := make([][32]byte, 0, len(store.challenges))
	for hash, challenge := range store.challenges {
		if !at.Before(challenge.expires) || challenge.used {
			keys = append(keys, hash)
		}
	}
	sort.Slice(keys, func(left, right int) bool { return string(keys[left][:]) < string(keys[right][:]) })
	if len(keys) > limit {
		keys = keys[:limit]
	}
	for _, hash := range keys {
		delete(store.challenges, hash)
	}
	removed := len(keys)
	return removed
}

func hashToken(token string) [32]byte {
	return sha256.Sum256([]byte("as360-challenge-v1\x00" + strings.TrimSpace(token)))
}
