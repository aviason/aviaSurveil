package admin

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"
)

func TestClientRegistryUsesExactSecureRedirectsAndHashedSecret(t *testing.T) {
	registry := NewClientRegistry(nil)
	client, err := registry.Register("web", "client-secret", []string{"https://app.example.invalid/callback"}, []string{"https://app.example.invalid/"}, []string{"openid", "profile"})
	if err != nil {
		t.Fatal(err)
	}
	if client.SecretHash == [32]byte{} {
		t.Fatal("client secret was not hashed")
	}
	if err := registry.Authenticate("web", "client-secret"); err != nil {
		t.Fatal(err)
	}
	if err := registry.Authenticate("web", "wrong"); !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("wrong client secret = %v", err)
	}
	for _, redirect := range []string{"https://app.example.invalid/callback*", "http://app.example.invalid/callback", "https://app.example.invalid/callback#fragment"} {
		if _, err := registry.Register("client-"+redirect[:1], "secret", []string{redirect}, nil, nil); err == nil {
			t.Fatalf("unsafe redirect accepted: %s", redirect)
		}
	}
	if err := registry.Revoke("web"); err != nil {
		t.Fatal(err)
	}
	if err := registry.Authenticate("web", "client-secret"); !errors.Is(err, ErrClientInactive) {
		t.Fatalf("revoked client authenticate = %v", err)
	}
}

func TestKeyRingProvidesRS256OverlapAndRetirement(t *testing.T) {
	initial, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	ring, err := NewKeyRing("key-2026-a", initial, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if err := ring.Ready(); err != nil {
		t.Fatal(err)
	}
	if _, err := ring.Rotate("key-2026-b", 2048, time.Hour); err != nil {
		t.Fatal(err)
	}
	if len(ring.VerificationKeys()) != 2 {
		t.Fatalf("verification key count = %d, want 2", len(ring.VerificationKeys()))
	}
	if err := ring.Retire("key-2026-a"); err != nil {
		t.Fatal(err)
	}
	if len(ring.VerificationKeys()) != 1 {
		t.Fatalf("retired verification key count = %d, want 1", len(ring.VerificationKeys()))
	}
	if err := ring.Retire("key-2026-b"); err == nil {
		t.Fatal("active signing key retired")
	}
	now = now.Add(2 * time.Hour)
	if err := ring.Ready(); err != nil {
		t.Fatal(err)
	}
}
