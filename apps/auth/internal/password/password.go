package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

var (
	ErrPasswordTooLong   = errors.New("password exceeds the configured byte bound")
	ErrPasswordTooShort  = errors.New("password is shorter than the configured policy")
	ErrInvalidHash       = errors.New("password hash is invalid")
	ErrHashCapacity      = errors.New("password hashing capacity is unavailable")
	ErrPasswordReused    = errors.New("password was used previously")
	ErrCompromised       = errors.New("password is not allowed by the compromised-password policy")
	ErrInvalidParameters = errors.New("Argon2id parameters are outside the safe bounds")
)

const (
	defaultMemoryKiB = 64 * 1024
	defaultTime      = 3
	defaultThreads   = 2
	defaultKeyLength = 32
	defaultSaltLen   = 16
	maxMemoryKiB     = 256 * 1024
	maxTime          = 10
	maxThreads       = 8
	maxKeyLength     = 64
	maxSaltLength    = 64
)

type Params struct {
	MemoryKiB uint32
	Time      uint32
	Threads   uint8
	KeyLength uint32
	SaltLen   uint32
	MaxBytes  int
	Capacity  int
}

func DefaultParams() Params {
	return Params{
		MemoryKiB: defaultMemoryKiB,
		Time:      defaultTime,
		Threads:   defaultThreads,
		KeyLength: defaultKeyLength,
		SaltLen:   defaultSaltLen,
		MaxBytes:  1024,
		Capacity:  2,
	}
}

type Hasher struct {
	params Params
	gate   chan struct{}
	dummy  string
	mu     sync.RWMutex
}

func New(params Params) (*Hasher, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	hasher := &Hasher{params: params, gate: make(chan struct{}, params.Capacity)}
	salt := make([]byte, params.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate dummy password salt: %w", err)
	}
	digest, err := hasher.derive([]byte("AS360-DUMMY-PASSWORD-V1"), salt, params)
	if err != nil {
		return nil, fmt.Errorf("derive dummy password hash: %w", err)
	}
	hasher.dummy = encode(params, salt, digest)
	return hasher, nil
}

func NewDefault() (*Hasher, error) {
	return New(DefaultParams())
}

func (hasher *Hasher) Params() Params {
	return hasher.params
}

func (hasher *Hasher) DummyHash() string {
	hasher.mu.RLock()
	defer hasher.mu.RUnlock()
	return hasher.dummy
}

func (hasher *Hasher) Hash(value []byte) (string, error) {
	if err := hasher.validatePasswordBytes(value); err != nil {
		return "", err
	}
	release, err := hasher.acquire()
	if err != nil {
		return "", err
	}
	defer release()
	salt := make([]byte, hasher.params.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	digest, err := hasher.derive(value, salt, hasher.params)
	if err != nil {
		return "", fmt.Errorf("derive password hash: %w", err)
	}
	return encode(hasher.params, salt, digest), nil
}

// Verify always performs a bounded Argon2id operation, including for malformed
// or unknown-account hashes, so an unknown identifier cannot take a cheaper
// branch than a known account.
func (hasher *Hasher) Verify(encodedHash string, value []byte) (bool, error) {
	if err := hasher.validatePasswordBytes(value); err != nil {
		return false, err
	}
	params, salt, expected, err := parse(encodedHash)
	if err != nil || !withinBounds(params, hasher.params) {
		params, salt, expected, err = parse(hasher.DummyHash())
		if err != nil {
			return false, ErrInvalidHash
		}
	}
	release, err := hasher.acquire()
	if err != nil {
		return false, err
	}
	defer release()
	actual, err := hasher.derive(value, salt, params)
	if err != nil {
		return false, fmt.Errorf("derive password verification hash: %w", err)
	}
	return subtle.ConstantTimeCompare(actual, expected) == 1 && err == nil, nil
}

type Policy struct {
	MinBytes    int
	MaxBytes    int
	Compromised func([]byte) bool
}

func DefaultPolicy() Policy {
	return Policy{MinBytes: 12, MaxBytes: 1024}
}

func (policy Policy) Validate(value []byte, hasher *Hasher, current string, history []string) error {
	if len(value) < policy.MinBytes {
		return ErrPasswordTooShort
	}
	if len(value) > policy.MaxBytes {
		return ErrPasswordTooLong
	}
	if policy.Compromised != nil && policy.Compromised(value) {
		return ErrCompromised
	}
	if hasher == nil {
		return ErrInvalidHash
	}
	if current != "" {
		matches, err := hasher.Verify(current, value)
		if err != nil {
			return err
		}
		if matches {
			return ErrPasswordReused
		}
	}
	for _, previous := range history {
		matches, err := hasher.Verify(previous, value)
		if err != nil {
			return err
		}
		if matches {
			return ErrPasswordReused
		}
	}
	return nil
}

func (hasher *Hasher) validatePasswordBytes(value []byte) error {
	if len(value) > hasher.params.MaxBytes {
		return ErrPasswordTooLong
	}
	return nil
}

func (hasher *Hasher) acquire() (func(), error) {
	select {
	case hasher.gate <- struct{}{}:
		return func() { <-hasher.gate }, nil
	default:
		return nil, ErrHashCapacity
	}
}

func (hasher *Hasher) derive(value, salt []byte, params Params) ([]byte, error) {
	if len(value) > params.MaxBytes {
		return nil, ErrPasswordTooLong
	}
	return argon2.IDKey(value, salt, params.Time, params.MemoryKiB, params.Threads, params.KeyLength), nil
}

func validateParams(params Params) error {
	if params.MemoryKiB < 16*1024 || params.MemoryKiB > maxMemoryKiB ||
		params.Time == 0 || params.Time > maxTime ||
		params.Threads == 0 || params.Threads > maxThreads ||
		params.KeyLength < 16 || params.KeyLength > maxKeyLength ||
		params.SaltLen < 16 || params.SaltLen > maxSaltLength ||
		params.MaxBytes < 64 || params.MaxBytes > 4096 ||
		params.Capacity < 1 || params.Capacity > 32 {
		return ErrInvalidParameters
	}
	return nil
}

func withinBounds(candidate, configured Params) bool {
	return candidate.MemoryKiB >= 16*1024 && candidate.MemoryKiB <= configured.MemoryKiB &&
		candidate.Time >= 1 && candidate.Time <= configured.Time &&
		candidate.Threads >= 1 && candidate.Threads <= configured.Threads &&
		candidate.KeyLength >= 16 && candidate.KeyLength <= configured.KeyLength &&
		candidate.SaltLen >= 16 && candidate.SaltLen <= maxSaltLength
}

func encode(params Params, salt, digest []byte) string {
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		params.MemoryKiB,
		params.Time,
		params.Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(digest),
	)
}

func parse(encoded string) (Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return Params{}, nil, nil, ErrInvalidHash
	}
	var memory, rounds, threads uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &rounds, &threads); err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 16 || len(salt) > maxSaltLength {
		return Params{}, nil, nil, ErrInvalidHash
	}
	digest, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(digest) < 16 || len(digest) > maxKeyLength {
		return Params{}, nil, nil, ErrInvalidHash
	}
	return Params{MemoryKiB: memory, Time: rounds, Threads: uint8(threads), KeyLength: uint32(len(digest)), SaltLen: uint32(len(salt)), MaxBytes: 4096, Capacity: 1}, salt, digest, nil
}
