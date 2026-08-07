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

func TestCanonicalAuditPackageMigrationPinsReleasedScopeWithoutTemplateFallback(t *testing.T) {
	available, err := load()
	if err != nil {
		t.Fatal(err)
	}
	var sql string
	for _, migration := range available {
		if migration.version == 30 {
			sql = migration.sql
			break
		}
	}
	if sql == "" {
		t.Fatal("canonical audit package migration 30 is not embedded")
	}
	for _, required := range []string{
		"ALTER COLUMN checklist_template_version_id DROP NOT NULL",
		"canonical_scope_snapshot_id text",
		"REFERENCES canonical_audit_scope_snapshots(id)",
		"(checklist_template_version_id IS NOT NULL) <> (canonical_scope_snapshot_id IS NOT NULL)",
		"NEW.canonical_scope_snapshot_id IS DISTINCT FROM OLD.canonical_scope_snapshot_id",
		"ADD PRIMARY KEY (preparation_id, question_version_id, subject_id)",
		"canonical_question_version_provenance",
		"PREPROD_EXERCISE question versions cannot be published",
		"planning_snapshot_digest text",
		"governed_jsonb_sha256(snapshot)",
		"canonical_audit_scope_snapshot_planning_digest",
		"assignment_id text REFERENCES audit_assignments(id)",
		"canonical_question_catalog_publication_shape",
		"exact published candidate template",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 30 missing %q", required)
		}
	}
}

func TestCanonicalPreparationAssignmentBoundaryAllowsServerOwnedInspectionUntilMaterialization(t *testing.T) {
	available, err := load()
	if err != nil {
		t.Fatal(err)
	}
	var sql string
	for _, migration := range available {
		if migration.version == 31 {
			sql = migration.sql
			break
		}
	}
	if sql == "" {
		t.Fatal("canonical preparation boundary migration 31 is not embedded")
	}
	for _, required := range []string{
		"ALTER COLUMN inspection_id DROP NOT NULL",
		"ALTER COLUMN lead_subject_id DROP NOT NULL",
		"status IN ('LEAD_ASSIGNED', 'TEAM_ASSIGNED', 'QUESTIONS_ASSIGNED')",
		"inspection_id IS NULL AND lead_subject_id IS NOT NULL",
		"status NOT IN ('PREPARATION', 'LEAD_ASSIGNED', 'TEAM_ASSIGNED', 'QUESTIONS_ASSIGNED')",
		"audit_assignments_canonical_preparation_shape",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 31 missing %q", required)
		}
	}
}

func TestReturnedPotentialFindingHasAppendOnlySuccessorBoundary(t *testing.T) {
	available, err := load()
	if err != nil {
		t.Fatal(err)
	}
	var sql string
	for _, migration := range available {
		if migration.version == 32 {
			sql = migration.sql
			break
		}
	}
	if sql == "" {
		t.Fatal("Potential Finding successor migration 32 is not embedded")
	}
	for _, required := range []string{
		"supersedes_potential_finding_id",
		"DROP INDEX IF EXISTS potential_findings_checklist_response_idx",
		"WHERE status <> 'RETURNED'",
		"potential_findings_successor_idx",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 32 missing %q", required)
		}
	}
}

func TestCanonicalPreparationEditsAreAppendOnlyReceipts(t *testing.T) {
	available, err := load()
	if err != nil {
		t.Fatal(err)
	}
	var sql string
	for _, migration := range available {
		if migration.version == 33 {
			sql = migration.sql
			break
		}
	}
	if sql == "" {
		t.Fatal("canonical preparation edit history migration 33 is not embedded")
	}
	for _, required := range []string{
		"canonical_audit_preparation_edit_events",
		"UNIQUE (assignment_id, assignment_revision, edit_kind)",
		"reject_canonical_preparation_edit_change",
		"BEFORE UPDATE OR DELETE",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 33 missing %q", required)
		}
	}
}

func TestExerciseReviewEventsPinOperationAndIdempotencyKeys(t *testing.T) {
	available, err := load()
	if err != nil {
		t.Fatal(err)
	}
	var sql string
	for _, migration := range available {
		if migration.version == 34 {
			sql = migration.sql
			break
		}
	}
	if sql == "" {
		t.Fatal("exercise review idempotency migration 34 is not embedded")
	}
	for _, required := range []string{
		"operation_id text",
		"idempotency_key text",
		"canonical_exercise_question_review_events_operation_idx",
		"canonical_exercise_question_review_events_idempotency_idx",
		"DISABLE TRIGGER canonical_exercise_question_review_events_append_only",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 34 missing %q", required)
		}
	}
}
