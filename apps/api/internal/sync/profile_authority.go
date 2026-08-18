package fieldsync

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/aviason/aviaSurveil/internal/platform/idempotency"
)

func profilePublicJWKThumbprint(raw json.RawMessage) (string, error) {
	var jwk map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&jwk); err != nil {
		return "", fmt.Errorf("decode profile public JWK: %w", err)
	}
	if jwk == nil {
		return "", errors.New("profile public JWK must be an object")
	}
	if _, private := jwk["d"]; private {
		return "", errors.New("profile public JWK must not contain a private key")
	}
	if jwk["kty"] != "EC" || jwk["crv"] != "P-256" {
		return "", errors.New("profile public JWK must be an EC P-256 key")
	}
	x, ok := jwk["x"].(string)
	if !ok || x == "" {
		return "", errors.New("profile public JWK x coordinate is required")
	}
	y, ok := jwk["y"].(string)
	if !ok || y == "" {
		return "", errors.New("profile public JWK y coordinate is required")
	}
	xBytes, err := base64.RawURLEncoding.DecodeString(x)
	if err != nil || len(xBytes) != 32 {
		return "", errors.New("profile public JWK x coordinate is invalid")
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(y)
	if err != nil || len(yBytes) != 32 {
		return "", errors.New("profile public JWK y coordinate is invalid")
	}
	curve := elliptic.P256()
	if !curve.IsOnCurve(new(big.Int).SetBytes(xBytes), new(big.Int).SetBytes(yBytes)) {
		return "", errors.New("profile public JWK point is not on P-256")
	}
	hash, err := idempotency.SemanticHash(jwk)
	if err != nil {
		return "", fmt.Errorf("hash profile public JWK: %w", err)
	}
	return "sha256:" + hash, nil
}

func validateProfilePublicJWK(profileKeyID string, raw json.RawMessage) error {
	if profileKeyID == "" {
		return errors.New("profile key id is required")
	}
	thumbprint, err := profilePublicJWKThumbprint(raw)
	if err != nil {
		return err
	}
	if thumbprint != profileKeyID {
		return errors.New("profile key id does not match the public JWK thumbprint")
	}
	return nil
}

func verifyProfileAuthorityProof(raw json.RawMessage, profileKeyID, requestHash, proof string) error {
	if err := validateProfilePublicJWK(profileKeyID, raw); err != nil {
		return err
	}
	var jwk struct {
		X string `json:"x"`
		Y string `json:"y"`
	}
	if err := json.Unmarshal(raw, &jwk); err != nil {
		return err
	}
	x, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return errors.New("profile proof x coordinate is invalid")
	}
	y, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return errors.New("profile proof y coordinate is invalid")
	}
	publicKey := ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(x),
		Y:     new(big.Int).SetBytes(y),
	}
	signature, err := base64.RawURLEncoding.DecodeString(proof)
	if err != nil || len(signature) != 64 {
		return errors.New("profile authority proof is not a P-256 IEEE signature")
	}
	hash := sha256.Sum256([]byte(requestHash))
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])
	if !ecdsa.Verify(&publicKey, hash[:], r, s) {
		return errors.New("profile authority proof does not verify")
	}
	return nil
}
