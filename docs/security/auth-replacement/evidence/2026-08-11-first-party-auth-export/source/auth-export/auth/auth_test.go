package auth

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuthenticateDevHeader(t *testing.T) {
	req := httptest.NewRequest("POST", "/v2/posts", nil)
	req.Header.Set("X-User-ID", "6878")

	user, err := Authenticate(req, Config{AllowDevHeader: true})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != 6878 {
		t.Fatalf("user.ID = %d", user.ID)
	}
}

func TestAuthenticateRejectsDevHeaderWhenDisabled(t *testing.T) {
	req := httptest.NewRequest("POST", "/v2/posts", nil)
	req.Header.Set("X-User-ID", "6878")

	if _, err := Authenticate(req, Config{}); err == nil {
		t.Fatal("expected unauthorized")
	}
}

func TestAuthenticateJWT(t *testing.T) {
	service := testTokenService(t)
	issued := issueTestAccessToken(t, service, time.Now())
	req := httptest.NewRequest("POST", "/v2/posts", nil)
	req.Header.Set("Authorization", "Bearer "+issued.Token)

	user, err := Authenticate(req, Config{TokenVerifier: service})
	if err != nil {
		t.Fatal(err)
	}
	if user.Subject != "usr_test" || user.SessionID != "session-1" || user.ClientID != "emsi.ios" || user.AuthVersion != 1 {
		t.Fatalf("user = %#v", user)
	}
}

func TestAuthenticateRejectsExpiredJWT(t *testing.T) {
	service := testTokenService(t)
	issued := issueTestAccessToken(t, service, time.Now().Add(-2*time.Hour))
	req := httptest.NewRequest("POST", "/v2/posts", nil)
	req.Header.Set("Authorization", "Bearer "+issued.Token)

	if _, err := Authenticate(req, Config{TokenVerifier: service}); err == nil {
		t.Fatal("expected unauthorized")
	}
}

func TestAuthenticateRejectsWrongAlg(t *testing.T) {
	service := testTokenService(t)
	issued := issueTestAccessToken(t, service, time.Now())
	parts := strings.Split(issued.Token, ".")
	if len(parts) != 3 {
		t.Fatalf("token parts = %d", len(parts))
	}
	req := httptest.NewRequest("POST", "/v2/posts", nil)
	req.Header.Set("Authorization", "Bearer "+"eyJhbGciOiJIUzI1NiIsImtpZCI6InRlc3Qta2V5IiwidHlwIjoiSldUIn0."+parts[1]+"."+parts[2])

	if _, err := Authenticate(req, Config{TokenVerifier: service}); err == nil {
		t.Fatal("expected unauthorized")
	}
}

func TestJWKSExposesPublicKeyOnly(t *testing.T) {
	service := testTokenService(t)
	jwks := service.JWKS()
	keys, ok := jwks["keys"].([]map[string]any)
	if !ok || len(keys) != 1 {
		t.Fatalf("jwks = %#v", jwks)
	}
	key := keys[0]
	for _, field := range []string{"kty", "kid", "n", "e", "alg"} {
		if key[field] == "" {
			t.Fatalf("missing %s in jwk %#v", field, key)
		}
	}
	if _, ok := key["d"]; ok {
		t.Fatalf("jwks exposed private exponent: %#v", key)
	}
}

func testTokenService(t *testing.T) *TokenService {
	t.Helper()
	service, err := NewTokenService(TokenServiceConfig{
		AppEnv:              "development",
		IssuerURL:           "https://auth.example.test",
		KeyID:               "test-key",
		AccessTokenAudience: []string{"emsi-api"},
		AllowedClientIDs:    []string{"emsi.ios"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func issueTestAccessToken(t *testing.T, service *TokenService, now time.Time) AccessToken {
	t.Helper()
	issued, err := service.IssueAccessToken(AccessTokenInput{
		Subject:     "usr_test",
		SessionID:   "session-1",
		ClientID:    "emsi.ios",
		Role:        "member",
		AuthTime:    now,
		AuthVersion: 1,
	}, time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	return issued
}
