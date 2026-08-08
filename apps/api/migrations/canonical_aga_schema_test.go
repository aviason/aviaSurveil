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
		"read-time derivation",
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

func TestQuestionVersionProvenanceAllowsSameUsageCatalogReuse(t *testing.T) {
	available, err := load()
	if err != nil {
		t.Fatal(err)
	}
	var sql string
	for _, migration := range available {
		if migration.version == 37 {
			sql = migration.sql
			break
		}
	}
	if sql == "" {
		t.Fatal("question-version provenance migration 37 is not embedded")
	}
	for _, required := range []string{
		"ADD PRIMARY KEY (question_version_id, usage_class, catalog_id)",
		"usage_class <> NEW.usage_class",
		"catalog_id = NEW.catalog_id",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 37 missing %q", required)
		}
	}
}

func TestPreparationConfirmationPinsAssignmentRevisionAndSupportsMultiInspectorCoverage(t *testing.T) {
	available, err := load()
	if err != nil {
		t.Fatal(err)
	}
	var sql string
	for _, migration := range available {
		if migration.version == 38 {
			sql = migration.sql
			break
		}
	}
	if sql == "" {
		t.Fatal("preparation confirmation migration 38 is not embedded")
	}
	for _, required := range []string{
		"confirmed_assignment_revision",
		"canonical_preparation_confirmation_bindings",
		"legacy-binding",
		"cannot reconcile confirmed preparation without an exact audit assignment",
		"canonical_audit_preparation_questions_pkey",
		"PRIMARY KEY (preparation_id, question_version_id, subject_id)",
		"canonical_audit_preparation_questions_position_subject_key",
		"canonical_scope_snapshot_question_position_key",
		"UNIQUE (snapshot_id, question_version_id, position)",
		"canonical_preparation_question_scope_position_fkey",
		"FOREIGN KEY (released_scope_snapshot_id, question_version_id, position)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 38 missing %q", required)
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

func TestExerciseReviewStateIsScopedToTheAuthorizedAudit(t *testing.T) {
	available, err := load()
	if err != nil {
		t.Fatal(err)
	}
	var sql string
	for _, migration := range available {
		if migration.version == 39 {
			sql = migration.sql
			break
		}
	}
	if sql == "" {
		t.Fatal("exercise review scope migration 39 is not embedded")
	}
	for _, required := range []string{
		"scope_draft_id text REFERENCES canonical_audit_scope_drafts(id)",
		"canonical_exercise_question_review_drafts_scope_revision_key",
		"UNIQUE (scope_draft_id, catalog_id, question_version_id, revision)",
		"canonical_exercise_question_review_drafts_scope_required",
		"canonical_exercise_question_review_events_scope_required",
		"cannot infer canonical exercise review scope",
		"VALIDATE CONSTRAINT canonical_exercise_question_review_drafts_scope_required",
		"VALIDATE CONSTRAINT canonical_exercise_question_review_events_scope_required",
		"canonical_exercise_question_review_drafts_scope_question_idx",
		"canonical_exercise_question_review_events_scope_question_idx",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 39 missing %q", required)
		}
	}
}

func TestPreparationCoverageRequiresServerIssuedSingleUsePreview(t *testing.T) {
	available, err := load()
	if err != nil {
		t.Fatal(err)
	}
	var sql string
	for _, migration := range available {
		if migration.version == 41 {
			sql = migration.sql
			break
		}
	}
	if sql == "" {
		t.Fatal("preparation preview migration 41 is not embedded")
	}
	for _, required := range []string{
		"canonical_audit_preparation_edit_previews",
		"UNIQUE (assignment_id, edit_kind, operation_id)",
		"UNIQUE (assignment_id, edit_kind, idempotency_key)",
		"expires_at timestamptz NOT NULL",
		"canonical_audit_preparation_edit_preview_consumptions",
		"canonical_preparation_edit_preview_consumptions_append_only",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 41 missing %q", required)
		}
	}
}

func TestInspectionAttachmentCompletionPinsImmutableVersionAndTerminalEventBoundary(t *testing.T) {
	available, err := load()
	if err != nil {
		t.Fatal(err)
	}
	var sql string
	for _, migration := range available {
		if migration.version == 40 {
			sql = migration.sql
			break
		}
	}
	if sql == "" {
		t.Fatal("inspection attachment version migration 40 is not embedded")
	}
	for _, required := range []string{
		"CREATE TABLE inspection_attachment_versions",
		"UNIQUE (inspection_attachment_id, version)",
		"UNIQUE (id, source_object_metadata_id)",
		"inspection_attachment_versions_append_only",
		"inspection_attachment_versions_attachment_organization_fkey",
		"current_version_id text",
		"inspection_attachments_current_version_fkey",
		"inspection_attachments_current_version_source_fkey",
		"inspection_attachments_current_version_object_pair_check",
		"inspection-attachment-version:legacy:",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 40 missing %q", required)
		}
	}
}
