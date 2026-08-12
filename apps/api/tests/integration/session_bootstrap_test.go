package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/config"
	"github.com/aviason/aviaSurveil/internal/platform/session"
	"github.com/aviason/aviaSurveil/migrations"
)

func TestExplicitTestSessionBootstrapIsDeterministicAndIdempotent(t *testing.T) {
	pool := createTestDatabase(t, "session_bootstrap")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	settings := config.Settings{
		Environment:   "test",
		TestPrincipal: "inspector-cabin-001",
		TestSession:   "session-cabin-001",
	}
	now := time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)

	if err := session.BootstrapTestProfile(context.Background(), pool, settings, now); err != nil {
		t.Fatalf("first bootstrap: %v", err)
	}
	if err := session.BootstrapTestProfile(context.Background(), pool, settings, now.Add(time.Hour)); err != nil {
		t.Fatalf("idempotent bootstrap: %v", err)
	}

	var subjectID string
	var expiresAt time.Time
	var createdAt time.Time
	if err := pool.QueryRow(
		context.Background(),
		"SELECT subject_id, expires_at, created_at FROM session_references WHERE id = $1",
		settings.TestSession,
	).Scan(&subjectID, &expiresAt, &createdAt); err != nil {
		t.Fatalf("read test session: %v", err)
	}
	if subjectID != settings.TestPrincipal {
		t.Fatalf("session subject = %q", subjectID)
	}
	if !expiresAt.Equal(now.Add(8 * time.Hour)) {
		t.Fatalf("session expiry = %s", expiresAt)
	}
	if !createdAt.Equal(now) {
		t.Fatalf("session creation changed across bootstrap: %s", createdAt)
	}

	var identityCount int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM identity_references WHERE subject_id = $1", settings.TestPrincipal).Scan(&identityCount); err != nil {
		t.Fatalf("count test identities: %v", err)
	}
	if identityCount != 1 {
		t.Fatalf("identity count = %d", identityCount)
	}
}
