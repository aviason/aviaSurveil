package fieldsync

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestOfflineLeasePolicyUsesTheGateZeroMaximum(t *testing.T) {
	if grantDuration != 7*24*time.Hour {
		t.Fatalf("grant duration = %s, want seven server-time days", grantDuration)
	}
}

func TestProfilePublicJWKMustMatchTheStoredThumbprintAndRemainPublic(t *testing.T) {
	pad := func(value []byte) []byte {
		padded := make([]byte, 32)
		copy(padded[32-len(value):], value)
		return padded
	}
	x, y := elliptic.P256().ScalarBaseMult(big.NewInt(1).Bytes())
	publicJWKBytes, err := json.Marshal(map[string]string{
		"crv": "P-256", "kty": "EC",
		"x": base64.RawURLEncoding.EncodeToString(pad(x.Bytes())),
		"y": base64.RawURLEncoding.EncodeToString(pad(y.Bytes())),
	})
	if err != nil {
		t.Fatalf("encode public JWK: %v", err)
	}
	publicJWK := json.RawMessage(publicJWKBytes)
	profileKeyID, err := profilePublicJWKThumbprint(publicJWK)
	if err != nil {
		t.Fatalf("thumbprint public JWK: %v", err)
	}
	if err := validateProfilePublicJWK(profileKeyID, publicJWK); err != nil {
		t.Fatalf("validate matching public JWK: %v", err)
	}
	if err := validateProfilePublicJWK("sha256:"+strings.Repeat("0", 64), publicJWK); err == nil {
		t.Fatal("a mismatched profile key id must be rejected")
	}
	privateJWK := json.RawMessage(`{"crv":"P-256","d":"private","kty":"EC","x":"AaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaAA","y":"AaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaAA"}`)
	if _, err := profilePublicJWKThumbprint(privateJWK); err == nil {
		t.Fatal("a private JWK must not be registered as a profile public key")
	}
}

func TestProfileAuthorityProofVerifiesTheStableRequestHash(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate profile authority: %v", err)
	}
	pad := func(value []byte) []byte {
		padded := make([]byte, 32)
		copy(padded[32-len(value):], value)
		return padded
	}
	publicJWKBytes, err := json.Marshal(map[string]string{
		"crv": "P-256", "kty": "EC",
		"x": base64.RawURLEncoding.EncodeToString(pad(key.PublicKey.X.Bytes())),
		"y": base64.RawURLEncoding.EncodeToString(pad(key.PublicKey.Y.Bytes())),
	})
	if err != nil {
		t.Fatalf("encode profile authority: %v", err)
	}
	publicJWK := json.RawMessage(publicJWKBytes)
	profileKeyID, err := profilePublicJWKThumbprint(publicJWK)
	if err != nil {
		t.Fatalf("thumbprint profile authority: %v", err)
	}
	requestHash := "sha256:" + strings.Repeat("a", 64)
	digest := sha256.Sum256([]byte(requestHash))
	r, s, err := ecdsa.Sign(rand.Reader, key, digest[:])
	if err != nil {
		t.Fatalf("sign request hash: %v", err)
	}
	signature := append(pad(r.Bytes()), pad(s.Bytes())...)
	proof := base64.RawURLEncoding.EncodeToString(signature)
	if err := verifyProfileAuthorityProof(publicJWK, profileKeyID, requestHash, proof); err != nil {
		t.Fatalf("verify profile authority proof: %v", err)
	}
	if err := verifyProfileAuthorityProof(publicJWK, profileKeyID, requestHash+"-tampered", proof); err == nil {
		t.Fatal("a proof must not verify a different request hash")
	}
}
