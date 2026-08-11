package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters for balanced security and performance (~150ms on modern hardware)
const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // 64MB
	argon2Threads = 4
	argon2SaltLen = 16
	argon2HashLen = 32
)

// HashPassword creates a quantum-safe password hash using Argon2id
// Returns format: $argon2id$v=19$m=65536,t=3,p=4$<base64 salt>$<base64 hash>
func HashPassword(password string) string {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		panic("failed to generate salt: " + err.Error())
	}

	// Generate hash using Argon2id (quantum-safe, memory-hard)
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2HashLen,
	)

	// Format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2Memory,
		argon2Time,
		argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded
}

// VerifyPassword verifies a password against its Argon2id hash
func VerifyPassword(password, encoded string) bool {
	parsed, err := parseArgon2idHash(encoded)
	if err != nil {
		return false
	}

	// Hash the password with the extracted parameters
	hash := argon2.IDKey(
		[]byte(password),
		parsed.salt,
		parsed.time,
		parsed.memory,
		parsed.threads,
		argon2HashLen,
	)

	// Compare hashes in constant time to prevent timing attacks
	return len(hash) == len(parsed.hash) && subtleCompare(hash, parsed.hash)
}

type argon2idHash struct {
	time    uint32
	memory  uint32
	threads uint8
	salt    []byte
	hash    []byte
}

func parseArgon2idHash(encoded string) (*argon2idHash, error) {
	// Expected format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	if len(encoded) == 0 || encoded[0] != '$' {
		return nil, fmt.Errorf("invalid hash format")
	}

	// Split by $ delimiter
	parts := make([]string, 0)
	var current string
	for i := 1; i < len(encoded); i++ {
		if encoded[i] == '$' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(encoded[i])
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid hash format: expected at least 5 parts, got %d", len(parts))
	}

	if parts[0] != "argon2id" {
		return nil, fmt.Errorf("not an argon2id hash")
	}

	// Parse parameters: m=65536,t=3,p=4
	var time, memory, threads uint32
	_, err := fmt.Sscanf(parts[2], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hash parameters: %w", err)
	}

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, fmt.Errorf("failed to decode hash: %w", err)
	}

	return &argon2idHash{
		time:    uint32(time),
		memory:  memory,
		threads: uint8(threads),
		salt:    salt,
		hash:    hash,
	}, nil
}

// subtleCompare performs constant-time comparison to prevent timing attacks
func subtleCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := range a {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
