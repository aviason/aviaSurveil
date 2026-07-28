//go:build integration

package scenarios_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/profiles"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/scenarios"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

func TestPostgresStoreMaterializesTheCompleteSmokeDomainAndReconciles(
	t *testing.T,
) {
	ctx := context.Background()
	pool := createScenarioDatabase(t)
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	profile, err := profiles.Lookup("smoke", "1.0.0")
	if err != nil {
		t.Fatalf("lookup smoke: %v", err)
	}
	stream, err := scenarios.NewStream(
		profile,
		[]byte("task-7-connected-scenarios"),
		loadCanonicalCatalog(t),
	)
	if err != nil {
		t.Fatalf("new stream: %v", err)
	}
	store, err := scenarios.NewPostgresStore(
		pool,
		profile,
		"run-task7-connected-smoke",
	)
	if err != nil {
		t.Fatalf("new PostgreSQL store: %v", err)
	}
	if err := store.Initialize(ctx); err != nil {
		t.Fatalf("initialize PostgreSQL store: %v", err)
	}
	for {
		command, nextErr := stream.Next(ctx)
		if nextErr != nil {
			if nextErr.Error() == "EOF" {
				break
			}
			t.Fatalf("next scenario command: %v", nextErr)
		}
		if err := store.Apply(ctx, command); err != nil {
			t.Fatalf("apply %s: %v", command.OperationID, err)
		}
	}

	providerRecords, err := store.Records(ctx, "providerAccounts")
	if err != nil {
		t.Fatalf("read durable provider-account records: %v", err)
	}
	if len(providerRecords) != 9 ||
		providerRecords[0].Family != "providerAccounts" ||
		providerRecords[0].RecordID == "" {
		t.Fatalf(
			"durable provider-account records = %#v",
			providerRecords,
		)
	}

	reconciliation, err := store.Reconcile(ctx)
	if err != nil {
		t.Fatalf("reconcile PostgreSQL store: %v", err)
	}
	if !reflect.DeepEqual(
		reconciliation.ActualCounts,
		profile.ExpectedCounts,
	) {
		t.Fatalf(
			"reconciled counts differ:\nactual=%#v\nexpected=%#v",
			reconciliation.ActualCounts,
			profile.ExpectedCounts,
		)
	}
	for family, digest := range reconciliation.RelationshipDigests {
		if !strings.HasPrefix(digest, "sha256:") ||
			len(digest) != len("sha256:")+64 {
			t.Fatalf("%s relationship digest = %q", family, digest)
		}
	}

	assertSQLCount(t, pool, "organizations", 3)
	assertSQLCount(t, pool, "identity_references", 9)
	assertSQLCount(t, pool, "desired_membership_versions", 18)
	assertSQLCount(t, pool, "user_profiles", 9)
	assertSQLCount(t, pool, "session_references", 18)
	assertSQLCount(t, pool, "surveillance_plan_items", 4)
	assertSQLCount(t, pool, "inspections", 2)
	assertSQLCount(t, pool, "audit_question_assignments", 3)
	assertSQLCount(t, pool, "template_masters", 4)
	assertSQLCount(t, pool, "checklist_template_versions", 6)
	assertSQLCount(t, pool, "question_versions", 24)
	assertSQLCount(t, pool, "inspection_packages", 2)
	assertSQLCount(t, pool, "checklist_responses", 24)
	assertSQLCount(t, pool, "potential_findings", 12)
	assertSQLCount(t, pool, "findings", 8)
	assertSQLCount(t, pool, "cap_revisions", 12)
	assertSQLCount(t, pool, "evidence_versions", 16)
	assertSQLCount(t, pool, "review_decisions", 16)
	assertSQLCount(t, pool, "report_versions", 6)
	assertSQLCount(t, pool, "communication_messages", 16)
	assertSQLCount(t, pool, "notification_records", 24)
	assertSQLCount(t, pool, "audit_events", 250)
	assertSQLCount(t, pool, "outbox_messages", 80)
	assertSQLCount(t, pool, "notification_delivery_jobs", 48)
	assertSQLCount(t, pool, "object_metadata", 24)
	assertSQLCount(t, pool, "document_render_jobs", 6)
	assertSQLCount(t, pool, "reminder_dispatches", 20)
	assertSQLCount(t, pool, "offline_grants", 4)
	assertSQLCount(t, pool, "authorized_sync_changes", 120)
}

func assertSQLCount(
	t *testing.T,
	pool *database.Pool,
	table string,
	expected int64,
) {
	t.Helper()
	var actual int64
	if err := pool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM "+table,
	).Scan(&actual); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if actual != expected {
		t.Fatalf("%s count = %d, expected %d", table, actual, expected)
	}
}

func createScenarioDatabase(t *testing.T) *database.Pool {
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
	databaseName := fmt.Sprintf(
		"avia_task7_scenarios_%d",
		time.Now().UnixNano(),
	)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+databaseName); err != nil {
		admin.Close()
		t.Fatalf("create scenario database: %v", err)
	}
	admin.Close()

	databaseURL := *parsed
	databaseURL.Path = "/" + databaseName
	pool, err := database.Open(ctx, databaseURL.String())
	if err != nil {
		t.Fatalf("open scenario database: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		admin, openErr := database.Open(context.Background(), adminURL.String())
		if openErr != nil {
			t.Errorf("reopen PostgreSQL admin connection: %v", openErr)
			return
		}
		defer admin.Close()
		if _, dropErr := admin.Exec(
			context.Background(),
			"DROP DATABASE "+databaseName+" WITH (FORCE)",
		); dropErr != nil {
			t.Errorf("drop scenario database: %v", dropErr)
		}
	})
	return pool
}
