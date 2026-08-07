package migrations

import (
	"strings"
	"testing"
)

func TestCanonicalAGAMigrationKeepsQuestionVersionAuthorityAndAppendOnlyBoundaries(t *testing.T) {
	available, err := load()
	if err != nil {
		t.Fatal(err)
	}
	var sql string
	for _, migration := range available {
		if migration.version == 29 {
			sql = migration.sql
			break
		}
	}
	if sql == "" {
		t.Fatal("canonical AGA migration 29 is not embedded")
	}
	for _, required := range []string{
		"REFERENCES question_versions(id)",
		"REFERENCES canonical_question_catalog_memberships(catalog_id, question_version_id)",
		"REFERENCES canonical_audit_scope_snapshot_questions(snapshot_id, question_version_id)",
		"canonical_question_catalog_memberships",
		"canonical_exercise_question_review_events",
		"canonical_audit_scope_snapshots",
		"canonical_audit_scope_draft_questions",
		"canonical_audit_preparation_snapshots",
		"reject_immutable_row_change()",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 29 missing %q", required)
		}
	}
}
