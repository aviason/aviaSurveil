package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"kindred_server/internal/auth"
	"kindred_server/pkg/security"
)

func newTestSignerVerifier(t *testing.T) (*rsa.PrivateKey, *auth.Signer, *auth.Verifier) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	const (
		iss = "https://kindred-test.local"
		aud = "kindred-mobile"
	)
	kid := security.JWKThumbprintKID(&priv.PublicKey)
	return priv,
		auth.NewSigner(priv, iss, aud, kid),
		auth.NewVerifier(&priv.PublicKey, iss, aud)
}

func TestSignVerifyRoundtrip(t *testing.T) {
	_, signer, verifier := newTestSignerVerifier(t)
	tok, exp, err := signer.Sign("user-1", "u@example.com", "device-1", 7, time.Hour)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if !exp.After(time.Now()) {
		t.Fatalf("exp not in future: %v", exp)
	}
	if strings.Count(tok, ".") != 2 {
		t.Fatalf("token doesn't look like a JWT: %q", tok)
	}
	claims, err := verifier.Parse(tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.UserID != "user-1" || claims.Email != "u@example.com" ||
		claims.TokenVersion != 7 || claims.DeviceID != "device-1" {
		t.Errorf("claims roundtrip mismatch: %+v", claims)
	}
}

func TestSignSerializesSingleAudienceAsString(t *testing.T) {
	_, signer, _ := newTestSignerVerifier(t)
	tok, _, err := signer.Sign("user-1", "u@example.com", "device-1", 7, time.Hour)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("token parts = %d, want 3", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if aud, ok := claims["aud"].(string); !ok || aud != "kindred-mobile" {
		t.Fatalf("aud = %#v, want string kindred-mobile", claims["aud"])
	}
}

func TestRejectsAlgConfusionHS256(t *testing.T) {
	priv, _, verifier := newTestSignerVerifier(t)
	// Forge an HS256 token using the public-key bytes as the HMAC secret.
	pubPEM := priv.PublicKey.N.Bytes()
	mal := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "https://kindred-test.local",
		"aud": "kindred-mobile",
		"sub": "attacker",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := mal.SignedString(pubPEM)
	if err != nil {
		t.Fatalf("sign hs256: %v", err)
	}
	if _, err := verifier.Parse(signed); err == nil {
		t.Fatal("expected HS256 token to be rejected")
	}
}

func TestRejectsExpired(t *testing.T) {
	_, signer, verifier := newTestSignerVerifier(t)
	tok, _, err := signer.Sign("user-1", "u@example.com", "", 0, -time.Minute)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := verifier.Parse(tok); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestRejectsBadSignature(t *testing.T) {
	_, signer, _ := newTestSignerVerifier(t)
	_, _, otherVerifier := newTestSignerVerifier(t) // independent keypair
	tok, _, err := signer.Sign("user-1", "u@example.com", "", 0, time.Hour)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := otherVerifier.Parse(tok); err == nil {
		t.Fatal("expected token signed with other key to be rejected")
	}
}
