package jwks_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kindred_server/internal/auth"
	"kindred_server/internal/jwks"
	"kindred_server/internal/platform/router"
	"kindred_server/pkg/security"
)

func TestHandlerReturnsJWKSDocument(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	kid := security.JWKThumbprintKID(&priv.PublicKey)
	doc := auth.JWKS{Keys: []auth.JWK{auth.PublicJWK(&priv.PublicKey, kid)}}

	r := router.New()
	jwks.RegisterRoutes(r, jwks.NewHandler(doc, "https://test.example"))
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/.well-known/jwks.json")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Cache-Control"), "max-age=300") {
		t.Errorf("missing Cache-Control header: %q", resp.Header.Get("Cache-Control"))
	}

	var out auth.JWKS
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out.Keys) != 1 {
		t.Fatalf("keys len = %d", len(out.Keys))
	}
	k := out.Keys[0]
	if k.Kty != "RSA" || k.Alg != "RS256" || k.Use != "sig" {
		t.Errorf("wrong static fields: %+v", k)
	}
	if k.Kid != kid {
		t.Errorf("kid mismatch: %s vs %s", k.Kid, kid)
	}
	if k.N == "" || k.E == "" {
		t.Errorf("n/e missing: %+v", k)
	}
}
