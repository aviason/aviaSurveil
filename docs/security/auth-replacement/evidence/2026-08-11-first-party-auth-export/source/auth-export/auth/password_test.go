package auth

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	params := Argon2idParams{
		Memory:      8 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}
	hash, err := HashPassword("correct horse battery staple", params)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword("correct horse battery staple", hash)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("password did not verify")
	}
	ok, err = VerifyPassword("wrong", hash)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("wrong password verified")
	}
}

func TestIssueAccessTokenCarriesSessionClaim(t *testing.T) {
	service := testTokenService(t)
	issued := issueTestAccessToken(t, service, time.Now())
	req := httptest.NewRequest("GET", "/User/get-user-info", nil)
	req.Header.Set("Authorization", "Bearer "+issued.Token)
	user, err := Authenticate(req, Config{TokenVerifier: service})
	if err != nil {
		t.Fatal(err)
	}
	if user.Subject != "usr_test" || user.Role != "member" || user.SessionID != "session-1" {
		t.Fatalf("user = %#v", user)
	}
}
