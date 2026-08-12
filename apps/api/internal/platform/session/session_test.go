package session_test

import (
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/session"
)

func TestReferenceIsActiveOnlyBeforeExpiryAndWithoutRevocation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)
	reference := session.Reference{ExpiresAt: now.Add(time.Minute)}
	if !reference.ActiveAt(now) {
		t.Fatal("unexpired session should be active")
	}
	if reference.ActiveAt(now.Add(time.Minute)) {
		t.Fatal("session at its expiry boundary should be inactive")
	}
	revokedAt := now.Add(-time.Second)
	reference.RevokedAt = &revokedAt
	if reference.ActiveAt(now) {
		t.Fatal("revoked session should be inactive")
	}
}
