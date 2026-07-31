package integration_test

import (
	"context"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

// A generation run may identify its artifacts only by their canonical JSONB
// content addresses; a shape-valid arbitrary digest is not provenance.
func TestTask3FixRound2RejectsNonCanonicalGenerationInputDigest(t *testing.T) {
	pool := createTestDatabase(t, "task3r2_input_digest")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	for _, statement := range []string{
		`INSERT INTO organizations (id, legal_name, organization_type, status) VALUES ('task3r2-digest-org', 'Digest operator', 'OPERATOR', 'ACTIVE')`,
		`INSERT INTO regulated_targets (id, target_kind, organization_id) VALUES ('task3r2-digest-target', 'ORGANIZATION', 'task3r2-digest-org')`,
	} {
		if _, err := pool.Exec(context.Background(), statement); err != nil {
			t.Fatalf("seed generation digest fixture: %v", err)
		}
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO regulatory_generation_runs (
			id, status, input_digest, output_digest, input_schema_version,
			generation_policy_version, provider_catalog_version,
			provider_adapter_version, inspection_type, target_id,
			input_artifact, output_artifact
		) VALUES (
			'task3r2-noncanonical-input', 'GENERATED',
			'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
			'sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
			'1.0.0', 'policy-v1', '1.0.0', 'fixture-v1', 'RAMP',
			'task3r2-digest-target', '{"requestId":"GENREQ-r2"}', '{"candidateId":"CAND-r2"}'
		)
	`); err == nil {
		t.Fatal("generation run accepted a shape-valid input digest that is not the canonical JSONB SHA-256 content address")
	}
}

func seedTask3FixRound2Candidate(t *testing.T, label string) *database.Pool {
	t.Helper()
	pool := createTestDatabase(t, label)
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	for _, statement := range []string{
		`INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ('task3r2-manager', 'test', 'Manager'), ('task3r2-generator', 'test', 'Generator')`,
		`INSERT INTO question_versions (id, question_id, version, prompt, configured_reference, expected_evidence, created_by_subject_id) VALUES ('task3r2-question', 'task3r2-question', 1, 'Question?', 'Reference', 'Evidence', 'task3r2-generator')`,
		`INSERT INTO caa_department_memberships (id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from) VALUES ('task3r2-membership', 'task3r2-manager', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'DEPARTMENT_MANAGER', 'ACTIVE', '2025-01-01')`,
		`INSERT INTO organizations (id, legal_name, organization_type, status) VALUES ('task3r2-org', 'Task 3 R2 operator', 'OPERATOR', 'ACTIVE')`,
		`INSERT INTO regulated_targets (id, target_kind, organization_id) VALUES ('task3r2-target', 'ORGANIZATION', 'task3r2-org')`,
		`INSERT INTO organization_service_provider_scopes (id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, primary_target_id) VALUES ('task3r2-scope', 'task3r2-org', 'AIR_OPERATOR', 'AOC-T3R2', 'ACTIVE', '2025-01-01', 'task3r2-target')`,
		`INSERT INTO regulatory_source_versions (id, source_identity, version_identity, title, source_class, source_status, source_locator, source_hash, effective_from, source_metadata) VALUES ('task3r2-source', 'T3R2', '1', 'Task 3 fix round 2 source', 'PRIMARY_AUTHORITY', 'PUBLIC_REFERENCE', 'T3R2 locator', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', '2025-01-01', '{}')`,
		`INSERT INTO regulatory_normalized_clauses (id, regulatory_source_version_id, clause_identity, annex_identity, section_identity, clause_locator, source_hash, normalized_digest) VALUES ('task3r2-clause', 'task3r2-source', 'T3R2-1', 'T3R2', '1', 'T3R2 locator', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb')`,
		`INSERT INTO regulatory_generation_runs (id, status, input_digest, output_digest, input_schema_version, generation_policy_version, provider_catalog_version, provider_adapter_version, inspection_type, target_id, input_artifact, output_artifact) VALUES ('task3r2-run', 'GENERATED', 'sha256:8479460bfbfefa7c07cf02769ef0791a95e9cfb5d9ab57758677eb3896e0e6c0', 'sha256:b8bd92b62743f472c25f3101a938f3886221fb8cd63bf8b7250e7409416ff319', '1.0.0', 'policy-v1', '1.0.0', 'fixture-v1', 'RAMP', 'task3r2-target', '{"requestId":"GENREQ-r2-decision"}', '{"candidateId":"CAND-r2-decision"}')`,
		`INSERT INTO regulatory_generation_run_scope_facts (generation_run_id, organization_service_provider_scope_id, scope_root_id, organization_id, service_provider_type_id, authorization_identifier, scope_status, effective_from, effective_to, regulated_target_id) SELECT 'task3r2-run', id, root_id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, effective_to, primary_target_id FROM organization_service_provider_scopes WHERE id = 'task3r2-scope'`,
		`INSERT INTO regulatory_generation_run_source_snapshots (generation_run_id, regulatory_source_version_id, regulatory_normalized_clause_id, source_hash, clause_locator) VALUES ('task3r2-run', 'task3r2-source', 'task3r2-clause', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'T3R2 locator')`,
		`INSERT INTO template_masters (id, title, owner_role) VALUES ('task3r2-template', 'Task 3 R2 template', 'Admin Preview')`,
		`INSERT INTO template_draft_versions (id, template_id, version, status, owner_role, creator_subject_id, change_reason, question_version_ids, revision, generation_run_id, candidate_content_digest, candidate_schema_version, candidate_root_id) VALUES ('task3r2-candidate', 'task3r2-template', 1, 'GENERATED_DRAFT', 'Admin Preview', 'task3r2-generator', 'Generated from frozen lineage.', ARRAY['task3r2-question'], 1, 'task3r2-run', 'sha256:b8bd92b62743f472c25f3101a938f3886221fb8cd63bf8b7250e7409416ff319', '1.0.0', 'task3r2-candidate')`,
		`INSERT INTO candidate_required_owner_assignments (id, candidate_draft_version_id, candidate_revision, candidate_content_digest, department_id, organizational_unit_id, approval_required) VALUES ('task3r2-owner', 'task3r2-candidate', 1, 'sha256:b8bd92b62743f472c25f3101a938f3886221fb8cd63bf8b7250e7409416ff319', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', true)`,
	} {
		if _, err := pool.Exec(context.Background(), statement); err != nil {
			t.Fatalf("seed Task 3 fix round 2 candidate: %v", err)
		}
	}
	return pool
}

func insertTask3FixRound2Decision(pool *database.Pool, id, membershipID, departmentID, unitID string) error {
	_, err := pool.Exec(context.Background(), `
		INSERT INTO department_review_decisions (
			id, candidate_draft_version_id, candidate_revision, candidate_content_digest,
			decision, actor_subject_id, actor_department_membership_id,
			actor_department_id, actor_organizational_unit_id, reason, decided_at,
			operation_id, idempotency_key, semantic_payload_digest
		) VALUES ($1, 'task3r2-candidate', 1,
			'sha256:b8bd92b62743f472c25f3101a938f3886221fb8cd63bf8b7250e7409416ff319',
			'TECHNICALLY_APPROVED', 'task3r2-manager', $2, $3, $4,
			'Regression decision.', '2026-07-29T12:00:00Z', $1 || '-op', $1 || '-idem',
			'sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff'
		)`, id, membershipID, departmentID, unitID)
	return err
}

func TestTask3FixRound2DecisionActorFailsClosedOnMissingStatusFacts(t *testing.T) {
	for _, scenario := range []struct {
		name, setup, departmentID, unitID string
	}{
		{
			name:         "missing department status",
			setup:        `INSERT INTO caa_departments (id, name, status) VALUES ('task3r2-missing-department', 'Missing department', 'ACTIVE'); INSERT INTO caa_organizational_units (id, department_id, name, status) VALUES ('task3r2-missing-unit', 'task3r2-missing-department', 'Missing unit', 'ACTIVE'); INSERT INTO caa_department_memberships (id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from) VALUES ('task3r2-missing-department-membership', 'task3r2-manager', 'task3r2-missing-department', 'task3r2-missing-unit', 'DEPARTMENT_MANAGER', 'ACTIVE', '2025-01-01')`,
			departmentID: "task3r2-missing-department", unitID: "task3r2-missing-unit",
		},
		{
			name:         "missing organizational unit status",
			setup:        `INSERT INTO caa_departments (id, name, status) VALUES ('task3r2-unit-department', 'Unit department', 'ACTIVE'); INSERT INTO caa_organizational_units (id, department_id, name, status) VALUES ('task3r2-unit-missing', 'task3r2-unit-department', 'Unit missing', 'ACTIVE'); INSERT INTO caa_department_status_facts (id, root_id, department_id, status, effective_from) VALUES ('task3r2-unit-department-status', 'task3r2-unit-department-status', 'task3r2-unit-department', 'ACTIVE', '2025-01-01'); INSERT INTO caa_department_memberships (id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from) VALUES ('task3r2-missing-unit-membership', 'task3r2-manager', 'task3r2-unit-department', 'task3r2-unit-missing', 'DEPARTMENT_MANAGER', 'ACTIVE', '2025-01-01')`,
			departmentID: "task3r2-unit-department", unitID: "task3r2-unit-missing",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			pool := seedTask3FixRound2Candidate(t, "task3r2_"+scenario.name[:7])
			if _, err := pool.Exec(context.Background(), scenario.setup); err != nil {
				t.Fatalf("seed %s fixture: %v", scenario.name, err)
			}
			membershipID := "task3r2-missing-department-membership"
			if scenario.name == "missing organizational unit status" {
				membershipID = "task3r2-missing-unit-membership"
			}
			if err := insertTask3FixRound2Decision(pool, "task3r2-"+membershipID, membershipID, scenario.departmentID, scenario.unitID); err == nil {
				t.Fatalf("decision actor was accepted with %s", scenario.name)
			}
		})
	}
}

func TestTask3FixRound2DecisionActorRejectsInactiveStatusFacts(t *testing.T) {
	for _, scenario := range []struct {
		name, setup, departmentID, unitID string
	}{
		{
			name:         "inactive department",
			setup:        `INSERT INTO caa_departments (id, name, status) VALUES ('task3r2-inactive-department', 'Inactive department', 'ACTIVE'); INSERT INTO caa_organizational_units (id, department_id, name, status) VALUES ('task3r2-inactive-department-unit', 'task3r2-inactive-department', 'Inactive department unit', 'ACTIVE'); INSERT INTO caa_department_status_facts (id, root_id, department_id, status, effective_from) VALUES ('task3r2-inactive-department-status', 'task3r2-inactive-department-status', 'task3r2-inactive-department', 'INACTIVE', '2025-01-01'); INSERT INTO caa_organizational_unit_status_facts (id, root_id, organizational_unit_id, status, effective_from) VALUES ('task3r2-inactive-department-unit-status', 'task3r2-inactive-department-unit-status', 'task3r2-inactive-department-unit', 'ACTIVE', '2025-01-01'); INSERT INTO caa_department_memberships (id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from) VALUES ('task3r2-inactive-department-membership', 'task3r2-manager', 'task3r2-inactive-department', 'task3r2-inactive-department-unit', 'DEPARTMENT_MANAGER', 'ACTIVE', '2025-01-01')`,
			departmentID: "task3r2-inactive-department", unitID: "task3r2-inactive-department-unit",
		},
		{
			name:         "inactive organizational unit",
			setup:        `INSERT INTO caa_departments (id, name, status) VALUES ('task3r2-inactive-unit-department', 'Inactive unit department', 'ACTIVE'); INSERT INTO caa_organizational_units (id, department_id, name, status) VALUES ('task3r2-inactive-unit', 'task3r2-inactive-unit-department', 'Inactive unit', 'ACTIVE'); INSERT INTO caa_department_status_facts (id, root_id, department_id, status, effective_from) VALUES ('task3r2-inactive-unit-department-status', 'task3r2-inactive-unit-department-status', 'task3r2-inactive-unit-department', 'ACTIVE', '2025-01-01'); INSERT INTO caa_organizational_unit_status_facts (id, root_id, organizational_unit_id, status, effective_from) VALUES ('task3r2-inactive-unit-status', 'task3r2-inactive-unit-status', 'task3r2-inactive-unit', 'INACTIVE', '2025-01-01'); INSERT INTO caa_department_memberships (id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from) VALUES ('task3r2-inactive-unit-membership', 'task3r2-manager', 'task3r2-inactive-unit-department', 'task3r2-inactive-unit', 'DEPARTMENT_MANAGER', 'ACTIVE', '2025-01-01')`,
			departmentID: "task3r2-inactive-unit-department", unitID: "task3r2-inactive-unit",
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			pool := seedTask3FixRound2Candidate(t, "task3r2_"+scenario.name[:8])
			if _, err := pool.Exec(context.Background(), scenario.setup); err != nil {
				t.Fatalf("seed %s fixture: %v", scenario.name, err)
			}
			membershipID := "task3r2-inactive-department-membership"
			if scenario.name == "inactive organizational unit" {
				membershipID = "task3r2-inactive-unit-membership"
			}
			if err := insertTask3FixRound2Decision(pool, "task3r2-"+membershipID, membershipID, scenario.departmentID, scenario.unitID); err == nil {
				t.Fatalf("decision actor was accepted with %s", scenario.name)
			}
		})
	}
}

// Each negative case is deliberately isolated so one rejected malformed
// candidate cannot hide another missing generation-lineage invariant.
func TestTask3FixRound2GeneratedCandidateNegativeMatrix(t *testing.T) {
	const digest = "sha256:b8bd92b62743f472c25f3101a938f3886221fb8cd63bf8b7250e7409416ff319"
	cases := []struct {
		name, setup, candidate string
	}{
		{"missing exact scope fact", `INSERT INTO regulatory_generation_runs (id, status, input_digest, output_digest, input_schema_version, generation_policy_version, provider_catalog_version, provider_adapter_version, inspection_type, target_id, input_artifact, output_artifact) VALUES ('task3r2-no-scope-run', 'GENERATED', governed_jsonb_sha256('{"request":"no-scope"}'::jsonb), governed_jsonb_sha256('{"candidate":"no-scope"}'::jsonb), '1', 'policy', '1', 'fixture', 'RAMP', 'task3r2-target', '{"request":"no-scope"}', '{"candidate":"no-scope"}')`, `INSERT INTO template_draft_versions (id, template_id, version, status, owner_role, creator_subject_id, change_reason, question_version_ids, revision, generation_run_id, candidate_content_digest, candidate_schema_version, candidate_root_id) VALUES ('task3r2-no-scope-candidate', 'task3r2-template', 2, 'GENERATED_DRAFT', 'Admin Preview', 'task3r2-generator', 'No scope.', ARRAY['task3r2-question'], 2, 'task3r2-no-scope-run', governed_jsonb_sha256('{"candidate":"no-scope"}'::jsonb), '1', 'task3r2-no-scope-candidate')`},
		{"missing source snapshot", `INSERT INTO regulatory_generation_runs (id, status, input_digest, output_digest, input_schema_version, generation_policy_version, provider_catalog_version, provider_adapter_version, inspection_type, target_id, input_artifact, output_artifact) VALUES ('task3r2-no-snapshot-run', 'GENERATED', governed_jsonb_sha256('{"request":"no-snapshot"}'::jsonb), governed_jsonb_sha256('{"candidate":"no-snapshot"}'::jsonb), '1', 'policy', '1', 'fixture', 'RAMP', 'task3r2-target', '{"request":"no-snapshot"}', '{"candidate":"no-snapshot"}'); INSERT INTO regulatory_generation_run_scope_facts (generation_run_id, organization_service_provider_scope_id, scope_root_id, organization_id, service_provider_type_id, authorization_identifier, scope_status, effective_from, effective_to, regulated_target_id) SELECT 'task3r2-no-snapshot-run', id, root_id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, effective_to, primary_target_id FROM organization_service_provider_scopes WHERE id = 'task3r2-scope'`, `INSERT INTO template_draft_versions (id, template_id, version, status, owner_role, creator_subject_id, change_reason, question_version_ids, revision, generation_run_id, candidate_content_digest, candidate_schema_version, candidate_root_id) VALUES ('task3r2-no-snapshot-candidate', 'task3r2-template', 2, 'GENERATED_DRAFT', 'Admin Preview', 'task3r2-generator', 'No snapshot.', ARRAY['task3r2-question'], 2, 'task3r2-no-snapshot-run', governed_jsonb_sha256('{"candidate":"no-snapshot"}'::jsonb), '1', 'task3r2-no-snapshot-candidate')`},
		{"empty question identities", "", `INSERT INTO template_draft_versions (id, template_id, version, status, owner_role, creator_subject_id, change_reason, question_version_ids, revision, generation_run_id, candidate_content_digest, candidate_schema_version, candidate_root_id) VALUES ('task3r2-empty-questions', 'task3r2-template', 2, 'GENERATED_DRAFT', 'Admin Preview', 'task3r2-generator', 'Empty questions.', '{}', 2, 'task3r2-run', '` + digest + `', '1', 'task3r2-empty-questions')`},
		{"unknown question identity", "", `INSERT INTO template_draft_versions (id, template_id, version, status, owner_role, creator_subject_id, change_reason, question_version_ids, revision, generation_run_id, candidate_content_digest, candidate_schema_version, candidate_root_id) VALUES ('task3r2-unknown-question', 'task3r2-template', 2, 'GENERATED_DRAFT', 'Admin Preview', 'task3r2-generator', 'Unknown question.', ARRAY['missing-question'], 2, 'task3r2-run', '` + digest + `', '1', 'task3r2-unknown-question')`},
		{"unrelated candidate digest", "", `INSERT INTO template_draft_versions (id, template_id, version, status, owner_role, creator_subject_id, change_reason, question_version_ids, revision, generation_run_id, candidate_content_digest, candidate_schema_version, candidate_root_id) VALUES ('task3r2-unrelated-digest', 'task3r2-template', 2, 'GENERATED_DRAFT', 'Admin Preview', 'task3r2-generator', 'Wrong digest.', ARRAY['task3r2-question'], 2, 'task3r2-run', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', '1', 'task3r2-unrelated-digest')`},
		{"invalid root", "", `INSERT INTO template_draft_versions (id, template_id, version, status, owner_role, creator_subject_id, change_reason, question_version_ids, revision, generation_run_id, candidate_content_digest, candidate_schema_version, candidate_root_id) VALUES ('task3r2-invalid-root', 'task3r2-template', 2, 'GENERATED_DRAFT', 'Admin Preview', 'task3r2-generator', 'Wrong root.', ARRAY['task3r2-question'], 2, 'task3r2-run', '` + digest + `', '1', 'task3r2-candidate')`},
		{"invalid successor", `INSERT INTO template_masters (id, title, owner_role) VALUES ('task3r2-other-template', 'Other template', 'Admin Preview'); INSERT INTO template_draft_versions (id, template_id, version, status, owner_role, creator_subject_id, change_reason, question_version_ids, revision) VALUES ('task3r2-unrelated-parent', 'task3r2-other-template', 1, 'DRAFT', 'Admin Preview', 'task3r2-generator', 'Unrelated parent.', ARRAY['task3r2-question'], 1)`, `INSERT INTO template_draft_versions (id, template_id, version, status, owner_role, creator_subject_id, change_reason, question_version_ids, revision, generation_run_id, candidate_content_digest, candidate_schema_version, candidate_root_id, supersedes_candidate_id) VALUES ('task3r2-invalid-successor', 'task3r2-template', 2, 'GENERATED_DRAFT', 'Admin Preview', 'task3r2-generator', 'Wrong parent.', ARRAY['task3r2-question'], 2, 'task3r2-run', '` + digest + `', '1', 'task3r2-candidate', 'task3r2-unrelated-parent')`},
		{"non-increasing revision", "", `INSERT INTO template_draft_versions (id, template_id, version, status, owner_role, creator_subject_id, change_reason, question_version_ids, revision, generation_run_id, candidate_content_digest, candidate_schema_version, candidate_root_id, supersedes_candidate_id) VALUES ('task3r2-nonincreasing', 'task3r2-template', 1, 'GENERATED_DRAFT', 'Admin Preview', 'task3r2-generator', 'Non-increasing.', ARRAY['task3r2-question'], 1, 'task3r2-run', '` + digest + `', '1', 'task3r2-candidate', 'task3r2-candidate')`},
		{"conflicting successor", `INSERT INTO template_draft_versions (id, template_id, version, status, owner_role, creator_subject_id, change_reason, question_version_ids, revision, generation_run_id, candidate_content_digest, candidate_schema_version, candidate_root_id, supersedes_candidate_id) VALUES ('task3r2-first-successor', 'task3r2-template', 2, 'GENERATED_DRAFT', 'Admin Preview', 'task3r2-generator', 'First successor.', ARRAY['task3r2-question'], 2, 'task3r2-run', '` + digest + `', '1', 'task3r2-candidate', 'task3r2-candidate')`, `INSERT INTO template_draft_versions (id, template_id, version, status, owner_role, creator_subject_id, change_reason, question_version_ids, revision, generation_run_id, candidate_content_digest, candidate_schema_version, candidate_root_id, supersedes_candidate_id) VALUES ('task3r2-conflicting-successor', 'task3r2-template', 3, 'GENERATED_DRAFT', 'Admin Preview', 'task3r2-generator', 'Conflicting successor.', ARRAY['task3r2-question'], 3, 'task3r2-run', '` + digest + `', '1', 'task3r2-candidate', 'task3r2-candidate')`},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			pool := seedTask3FixRound2Candidate(t, "task3r2_candidate_matrix")
			if scenario.setup != "" {
				if _, err := pool.Exec(context.Background(), scenario.setup); err != nil {
					t.Fatalf("seed %s fixture: %v", scenario.name, err)
				}
			}
			if _, err := pool.Exec(context.Background(), scenario.candidate); err == nil {
				t.Fatalf("generated candidate accepted %s", scenario.name)
			}
		})
	}
}
