package session

import (
	"context"
	"fmt"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/config"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

const testSessionAbsoluteDuration = 8 * time.Hour

func BootstrapTestProfile(ctx context.Context, pool *database.Pool, settings config.Settings, now time.Time) error {
	if settings.TestPrincipal == "" && settings.TestSession == "" {
		return nil
	}
	if settings.Environment != "test" {
		return fmt.Errorf("test session bootstrap requires AVIA_ENVIRONMENT=test")
	}
	if settings.TestPrincipal == "" || settings.TestSession == "" {
		return fmt.Errorf("test principal and session must both be configured")
	}
	if now.IsZero() {
		return fmt.Errorf("test session bootstrap time is required")
	}
	now = now.UTC()
	return database.WithinTransaction(ctx, pool, func(ctx context.Context, transaction pgx.Tx) error {
		if _, err := transaction.Exec(ctx, `
			INSERT INTO identity_references (subject_id, issuer, display_name, created_at)
			VALUES ($1, 'urn:aviasurveil360:test', $1, $2)
			ON CONFLICT (subject_id) DO NOTHING
		`, settings.TestPrincipal, now); err != nil {
			return fmt.Errorf("bootstrap test identity: %w", err)
		}
		if _, err := transaction.Exec(ctx, `
			INSERT INTO session_references (id, subject_id, expires_at, created_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO NOTHING
		`, settings.TestSession, settings.TestPrincipal, now.Add(testSessionAbsoluteDuration), now); err != nil {
			return fmt.Errorf("bootstrap test session: %w", err)
		}
		var existingSubject string
		if err := transaction.QueryRow(ctx, "SELECT subject_id FROM session_references WHERE id = $1", settings.TestSession).Scan(&existingSubject); err != nil {
			return fmt.Errorf("read bootstrapped test session: %w", err)
		}
		if existingSubject != settings.TestPrincipal {
			return fmt.Errorf("test session %s already belongs to another principal", settings.TestSession)
		}
		return nil
	})
}
