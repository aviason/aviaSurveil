package integration_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

var requiredFoundationTables = []string{
	"identity_references",
	"session_references",
	"oidc_login_states",
	"organizations",
	"inspections",
	"checklist_template_versions",
	"inspection_packages",
	"checklist_responses",
	"inspection_question_assignments",
	"inspection_checklists",
	"potential_findings",
	"findings",
	"cap_revisions",
	"evidence_versions",
	"review_decisions",
	"report_versions",
	"report_decisions",
	"report_approval_states",
	"offline_grants",
	"idempotency_responses",
	"authorized_sync_changes",
	"sync_cursors",
	"sync_cursor_tokens",
	"object_metadata",
	"upload_sessions",
	"evidence_version_states",
	"inspection_attachments",
	"audit_events",
	"outbox_messages",
	"surveillance_plan_items",
	"reminder_rules",
	"template_draft_versions",
}

func TestMigrationsApplyFromAnEmptyDatabase(t *testing.T) {
	pool := createTestDatabase(t, "empty")

	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	assertFoundationSchema(t, pool)
	if version, err := migrations.CurrentVersion(context.Background(), pool); err != nil || version != migrations.LatestVersion {
		t.Fatalf("migration version = %d, err = %v", version, err)
	}
	var organizationID, legalName, organizationType, status string
	if err := pool.QueryRow(context.Background(), `
		SELECT id, legal_name, organization_type, status
		FROM organizations
	`).Scan(
		&organizationID,
		&legalName,
		&organizationType,
		&status,
	); err != nil {
		t.Fatalf("read built-in authority organization: %v", err)
	}
	if organizationID != "CAA" ||
		legalName != "Civil Aviation Authority" ||
		organizationType != "AUTHORITY" ||
		status != "ACTIVE" {
		t.Fatalf(
			"built-in authority organization = %q %q %q %q",
			organizationID,
			legalName,
			organizationType,
			status,
		)
	}
}

func TestEveryRetainedNMinusOneFixtureUpgrades(t *testing.T) {
	fixtures, err := filepath.Glob(filepath.Join(apiModuleRoot(t), "tests", "fixtures", "n-1", "*.sql"))
	if err != nil {
		t.Fatalf("find N-1 fixtures: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no retained N-1 migration fixture")
	}

	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(strings.TrimSuffix(filepath.Base(fixture), ".sql"), func(t *testing.T) {
			pool := createTestDatabase(t, "upgrade")
			contents, err := loadFixture(apiModuleRoot(t), fixture)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			if _, err := pool.Exec(context.Background(), string(contents)); err != nil {
				t.Fatalf("apply fixture: %v", err)
			}
			if err := migrations.Apply(context.Background(), pool); err != nil {
				t.Fatalf("upgrade fixture: %v", err)
			}
			assertFoundationSchema(t, pool)
		})
	}
}

func loadFixture(moduleRoot, fixture string) ([]byte, error) {
	contents, err := os.ReadFile(fixture)
	if err != nil {
		return nil, err
	}
	var expanded strings.Builder
	for _, line := range strings.Split(string(contents), "\n") {
		const includePrefix = "-- avia-include: "
		if !strings.HasPrefix(line, includePrefix) {
			expanded.WriteString(line)
			expanded.WriteByte('\n')
			continue
		}
		included, err := os.ReadFile(filepath.Join(moduleRoot, strings.TrimPrefix(line, includePrefix)))
		if err != nil {
			return nil, err
		}
		expanded.Write(included)
		expanded.WriteByte('\n')
	}
	return []byte(expanded.String()), nil
}

func TestMigrationsAreForwardOnlyExceptTheGuardedGovernedRecoveryMigration(t *testing.T) {
	migrationFiles, err := filepath.Glob(filepath.Join(apiModuleRoot(t), "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("find migrations: %v", err)
	}
	if len(migrationFiles) == 0 {
		t.Fatal("no migrations")
	}
	for _, migrationFile := range migrationFiles {
		if strings.HasSuffix(migrationFile, ".down.sql") {
			if filepath.Base(migrationFile) != "000021_regulatory_checklist_governance.down.sql" {
				t.Errorf("unapproved down migration is not allowed: %s", migrationFile)
			}
			continue
		}
		if !strings.HasSuffix(migrationFile, ".up.sql") {
			t.Errorf("migration is not forward-only: %s", migrationFile)
		}
	}
}

func createTestDatabase(t *testing.T, label string) *database.Pool {
	t.Helper()
	ctx := context.Background()
	baseURL := os.Getenv("AVIA_TEST_DATABASE_URL")
	if baseURL == "" {
		baseURL = "postgres://aviasurveil:aviasurveil@127.0.0.1:55432/aviasurveil?sslmode=disable"
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}
	adminURL := *parsed
	adminURL.Path = "/postgres"
	admin, err := database.Open(ctx, adminURL.String())
	if err != nil {
		t.Fatalf("open PostgreSQL admin connection: %v", err)
	}
	databaseName := fmt.Sprintf("avia_%s_%d", label, time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+databaseName); err != nil {
		admin.Close()
		t.Fatalf("create test database: %v", err)
	}
	admin.Close()

	databaseURL := *parsed
	databaseURL.Path = "/" + databaseName
	pool, err := database.Open(ctx, databaseURL.String())
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		admin, openErr := database.Open(context.Background(), adminURL.String())
		if openErr != nil {
			t.Errorf("reopen PostgreSQL admin connection: %v", openErr)
			return
		}
		defer admin.Close()
		if _, dropErr := admin.Exec(context.Background(), "DROP DATABASE "+databaseName+" WITH (FORCE)"); dropErr != nil {
			t.Errorf("drop test database: %v", dropErr)
		}
	})
	return pool
}

func assertFoundationSchema(t *testing.T, pool *database.Pool) {
	t.Helper()
	for _, table := range requiredFoundationTables {
		var relation *string
		if err := pool.QueryRow(context.Background(), "SELECT to_regclass($1)::text", "public."+table).Scan(&relation); err != nil {
			t.Fatalf("look up table %s: %v", table, err)
		}
		if relation == nil {
			t.Errorf("required table %s does not exist", table)
		}
	}
	var retiredPackageDraftTable *string
	if err := pool.QueryRow(
		context.Background(),
		"SELECT to_regclass('public.inspection_package_drafts')::text",
	).Scan(&retiredPackageDraftTable); err != nil {
		t.Fatalf("look up retired inspection_package_drafts table: %v", err)
	}
	if retiredPackageDraftTable != nil {
		t.Errorf("retired inspection_package_drafts table still exists: %s", *retiredPackageDraftTable)
	}
}
