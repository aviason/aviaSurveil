package middleware_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kindred_server/internal/auth"
	"kindred_server/internal/platform/middleware"
	"kindred_server/internal/testutil"
)

// makeToken mints an auth token using the same scheme as auth.Service.issue,
// so tests can create tokens with arbitrary claims (e.g. already-expired).
func makeToken(t *testing.T, secret string, claims auth.Claims) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." +
		base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// helper: build a real auth.Service against a FakeUserRepo, register a user,
// and return a valid bearer token for that user.
func issueToken(t *testing.T) (svc *auth.Service, token, userID string) {
	t.Helper()
	ts := testutil.NewTestServer(t)
	id, tok := ts.RegisterAndLogin(t, "mw@example.com", "fixture-secret", "MW")
	return ts.AuthService, tok, id
}

func runAuthMW(svc *auth.Service, header string) (*httptest.ResponseRecorder, string) {
	var seen string
	h := middleware.Authenticator(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = middleware.UserID(r)
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	h.ServeHTTP(rec, req)
	return rec, seen
}

func TestAuthenticator_MissingHeader(t *testing.T) {
	svc, _, _ := issueToken(t)
	rec, seen := runAuthMW(svc, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if seen != "" {
		t.Errorf("handler should not have run; got userID=%q", seen)
	}
}

func TestAuthenticator_MalformedBearer(t *testing.T) {
	svc, tok, _ := issueToken(t)
	cases := []string{
		"Token " + tok,        // wrong scheme
		tok,                   // no scheme
		"Bearer ",             // empty token
		"Bearer    ",          // whitespace token
	}
	for _, h := range cases {
		rec, _ := runAuthMW(svc, h)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("header %q: status = %d, want 401", h, rec.Code)
		}
	}
}

func TestAuthenticator_InvalidToken(t *testing.T) {
	svc, _, _ := issueToken(t)
	cases := []string{
		"Bearer not-a-token",
		"Bearer aaaa.bbbb",
		"Bearer " + strings.Repeat("a", 32) + "." + strings.Repeat("b", 32),
	}
	for _, h := range cases {
		rec, _ := runAuthMW(svc, h)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("header %q: status = %d, want 401", h, rec.Code)
		}
	}
}

func TestAuthenticator_ValidTokenSetsUserID(t *testing.T) {
	svc, tok, id := issueToken(t)
	rec, seen := runAuthMW(svc, "Bearer "+tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if seen != id {
		t.Errorf("UserID = %q, want %q", seen, id)
	}
}

func TestAuthenticator_ExpiredToken(t *testing.T) {
	svc, _, _ := issueToken(t)
	expired := makeToken(t, "test-secret", auth.Claims{
		UserID:    "u1",
		Email:     "x@x.com",
		ExpiresAt: time.Now().Add(-time.Hour),
	})
	rec, _ := runAuthMW(svc, "Bearer "+expired)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expired token: status = %d, want 401", rec.Code)
	}
}

func TestAuthenticator_WrongSignature(t *testing.T) {
	svc, _, _ := issueToken(t)
	// Token signed with a different secret should fail signature check.
	bad := makeToken(t, "wrong-secret", auth.Claims{
		UserID:    "u1",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	rec, _ := runAuthMW(svc, "Bearer "+bad)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong-signature token: status = %d, want 401", rec.Code)
	}
}
