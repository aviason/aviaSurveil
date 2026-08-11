package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

var ErrInvalidPasswordHash = errors.New("invalid password hash")

type Argon2idParams struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

var DefaultArgon2idParams = Argon2idParams{
	Memory:      64 * 1024,
	Iterations:  3,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

func HashPassword(password string, params Argon2idParams) (string, error) {
	if params.Memory == 0 {
		params = DefaultArgon2idParams
	}
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		params.Memory,
		params.Iterations,
		params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func VerifyPassword(password, encodedHash string) (bool, error) {
	params, salt, key, err := decodePasswordHash(encodedHash)
	if err != nil {
		return false, err
	}
	derived := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, uint32(len(key)))
	return subtle.ConstantTimeCompare(derived, key) == 1, nil
}

func decodePasswordHash(encodedHash string) (Argon2idParams, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return Argon2idParams{}, nil, nil, ErrInvalidPasswordHash
	}
	paramParts := strings.Split(parts[3], ",")
	if len(paramParts) != 3 {
		return Argon2idParams{}, nil, nil, ErrInvalidPasswordHash
	}
	parsed := map[string]int{}
	for _, part := range paramParts {
		keyValue := strings.SplitN(part, "=", 2)
		if len(keyValue) != 2 {
			return Argon2idParams{}, nil, nil, ErrInvalidPasswordHash
		}
		value, err := strconv.Atoi(keyValue[1])
		if err != nil || value <= 0 {
			return Argon2idParams{}, nil, nil, ErrInvalidPasswordHash
		}
		parsed[keyValue[0]] = value
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Argon2idParams{}, nil, nil, ErrInvalidPasswordHash
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Argon2idParams{}, nil, nil, ErrInvalidPasswordHash
	}
	return Argon2idParams{
		Memory:      uint32(parsed["m"]),
		Iterations:  uint32(parsed["t"]),
		Parallelism: uint8(parsed["p"]),
		SaltLength:  uint32(len(salt)),
		KeyLength:   uint32(len(key)),
	}, salt, key, nil
}
