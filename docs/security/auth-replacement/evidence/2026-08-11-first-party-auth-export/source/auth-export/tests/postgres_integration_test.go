//go:build integration

package tests

import (
	"database/sql"
	"os"
	"strings"
	"testing"
)

// TestPostgresIntegration documents the safety guard for an integration
// harness. AviaSurveil360 must supply a task-owned PostgreSQL database driver
// and DSN; this export intentionally does not embed credentials or provision a
// database. Add a blank import for the chosen pgx database/sql driver in the
// integration application before enabling this build tag.
func TestPostgresIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("AVIASURVEIL360_AUTH_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("AVIASURVEIL360_AUTH_TEST_DATABASE_URL is not configured")
	}
	if !strings.Contains(strings.ToLower(dsn), "aviasurveil360_auth_test") {
		t.Fatal("refusing a database whose DSN does not contain aviasurveil360_auth_test")
	}
	var db *sql.DB
	_ = db
	t.Fatal("integration adapter and explicit pgx driver registration are required")
}
