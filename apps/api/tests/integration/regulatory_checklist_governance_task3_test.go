package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aviason/aviaSurveil/migrations"
)

const (
	task3SourceDigest    = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	task3ClauseDigest    = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	task3InputDigest     = "sha256:ca5ce54a7970d13ad57e658a37a48e6da9bff2012b09c8ee560dc53efd6a1a54"
	task3OutputDigest    = "sha256:6a8b4873df532ec3e67d03f4e6c9ef32a80f5666f7e0def78c8ee974e554a9f5"
	task3CandidateDigest = "sha256:49b0285b8dcbd477d155c0ed54051bcdbbfb075dbc045e3ce6de27b4317f2daa"
	task3SemanticDigest  = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	task3DifferentDigest = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
)

// This catches a migration that stores CC row labels only, rather than a
// versioned crosswalk source and a database-enforced split by stable identity.
func TestTask3CrosswalkPartitionsAreVersionedAndDisjoint(t *testing.T) {
	pool := createTestDatabase(t, "task3_crosswalk_partitions")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	for _, statement := range []string{
		`INSERT INTO regulatory_source_versions (id, source_identity, version_identity, title, source_class, source_status, source_locator, source_hash, effective_from, source_metadata) VALUES ('task3-cc-source', 'NCAA-CC-ANNEX6-PARTI-A610', 'SUPPLIED-1', 'Annex 6 Part I supplied compliance crosswalk', 'STATE_COMPLIANCE_CROSSWALK', 'SUPPLIED_WORKING_COPY', 'CC.zip/Annex_NAMB_A610.docx', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', '2026-07-29', '{"supplied":true}')`,
		`INSERT INTO regulatory_normalized_clauses (id, regulatory_source_version_id, clause_identity, annex_identity, section_identity, clause_locator, source_hash, normalized_digest) VALUES ('task3-cc-clause', 'task3-cc-source', 'ANNEX6-4.2.2.2', 'ANNEX_6_PART_I', '4.2.2.2', 'Annex 6 Part I 4.2.2.2', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb')`,
		`INSERT INTO state_compliance_crosswalk_rows (id, regulatory_source_version_id, normalized_clause_id, stable_row_identity, annex_identity, section_identity, row_digest) VALUES ('task3-cc-row', 'task3-cc-source', 'task3-cc-clause', 'CC:NAMB:ANNEX6:4.2.2.2', 'ANNEX_6_PART_I', '4.2.2.2', 'sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb')`,
		`INSERT INTO regulatory_evaluations (id, evaluation_identity, purpose) VALUES ('task3-evaluation', 'OPS-AOC-2026-07-29', 'generation quality evaluation')`,
		`INSERT INTO regulatory_evaluation_partitions (id, evaluation_id, partition_kind) VALUES ('task3-training', 'task3-evaluation', 'GENERATION_INPUT'), ('task3-holdout', 'task3-evaluation', 'BLIND_HOLDOUT')`,
		`INSERT INTO regulatory_evaluation_partition_rows (evaluation_id, partition_id, state_compliance_crosswalk_row_id, stable_row_identity) VALUES ('task3-evaluation', 'task3-training', 'task3-cc-row', 'CC:NAMB:ANNEX6:4.2.2.2')`,
	} {
		if _, err := pool.Exec(context.Background(), statement); err != nil {
			t.Fatalf("persist supplied CC source and training identity: %v", err)
		}
	}

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO regulatory_evaluation_partition_rows (
			evaluation_id, partition_id, state_compliance_crosswalk_row_id, stable_row_identity
		) VALUES (
			'task3-evaluation', 'task3-holdout', 'task3-cc-row', 'CC:NAMB:ANNEX6:4.2.2.2'
		)
	`); err == nil {
		t.Fatal("one stable CC identity was accepted in both training and blind holdout partitions")
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO regulatory_source_versions (
			id, source_identity, version_identity, title, source_class, source_status,
			source_locator, source_hash, effective_from, source_metadata
		) VALUES (
			'task3-invalid-hash', 'INVALID', '1', 'Invalid digest source', 'PRIMARY_AUTHORITY',
			'PUBLIC_REFERENCE', 'loc', 'sha256:not-a-valid-digest', '2026-07-29', '{}'
		)
	`); err == nil {
		t.Fatal("invalid source hash was accepted")
	}
}

// This catches lineage that can be relabeled after generation, or that accepts
// an unknown scope/source/target while leaving only a JSON-shaped trace.
func TestTask3GenerationLineageRejectsUnknownOrConflictingFacts(t *testing.T) {
	pool := createTestDatabase(t, "task3_generation_lineage")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	for _, statement := range []string{
		`INSERT INTO organizations (id, legal_name, organization_type, status) VALUES ('task3-org', 'Task 3 Operator', 'OPERATOR', 'ACTIVE')`,
		`INSERT INTO regulated_targets (id, target_kind, organization_id) VALUES ('task3-target', 'ORGANIZATION', 'task3-org')`,
		`INSERT INTO regulatory_source_versions (id, source_identity, version_identity, title, source_class, source_status, source_locator, source_hash, effective_from, source_metadata) VALUES ('task3-primary-source', 'ICAO-OPS-PQ', '2024-R1.1', 'OPS protocol questions', 'PRIMARY_AUTHORITY', 'PUBLIC_REFERENCE', 'PQ 4.450', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', '2024-09-01', '{}')`,
		`INSERT INTO regulatory_normalized_clauses (id, regulatory_source_version_id, clause_identity, annex_identity, section_identity, clause_locator, source_hash, normalized_digest) VALUES ('task3-primary-clause', 'task3-primary-source', 'PQ-4.450', 'ICAO_PQ_OPS', '4.450', 'OPS PQ 4.450', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb')`,
		`INSERT INTO organization_service_provider_scopes (id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, primary_target_id) VALUES ('task3-run-scope', 'task3-org', 'AIR_OPERATOR', 'AOC-RUN', 'ACTIVE', '2025-01-01', 'task3-target')`,
		`INSERT INTO regulatory_generation_runs (id, status, input_digest, output_digest, input_schema_version, generation_policy_version, provider_catalog_version, provider_adapter_version, inspection_type, target_id, input_artifact, output_artifact) VALUES ('task3-run', 'GENERATED', 'sha256:ca5ce54a7970d13ad57e658a37a48e6da9bff2012b09c8ee560dc53efd6a1a54', 'sha256:6a8b4873df532ec3e67d03f4e6c9ef32a80f5666f7e0def78c8ee974e554a9f5', '1.0.0', 'regulatory-checklist-v1', '1.0.0', 'fixture-provider-1', 'RAMP_INSPECTION', 'task3-target', '{"requestId":"GENREQ-1"}', '{"candidateId":"CAND-1"}')`,
		`INSERT INTO regulatory_generation_run_scope_facts (generation_run_id, organization_service_provider_scope_id, scope_root_id, organization_id, service_provider_type_id, authorization_identifier, scope_status, effective_from, effective_to, regulated_target_id) SELECT 'task3-run', id, root_id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, effective_to, primary_target_id FROM organization_service_provider_scopes WHERE id = 'task3-run-scope'`,
		`INSERT INTO regulatory_generation_run_source_snapshots (generation_run_id, regulatory_source_version_id, regulatory_normalized_clause_id, source_hash, clause_locator) VALUES ('task3-run', 'task3-primary-source', 'task3-primary-clause', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'OPS PQ 4.450')`,
	} {
		if _, err := pool.Exec(context.Background(), statement); err != nil {
			t.Fatalf("persist generation lineage: %v", err)
		}
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO regulatory_generation_run_scope_facts (generation_run_id, organization_service_provider_scope_id, scope_root_id, organization_id, service_provider_type_id, authorization_identifier, scope_status, effective_from, regulated_target_id) VALUES ('task3-run', 'missing-scope', 'missing-root', 'task3-org', 'AIR_OPERATOR', 'AOC-MISSING', 'ACTIVE', '2025-01-01', 'task3-target')`); err == nil {
		t.Fatal("generation lineage accepted a missing exact provider scope")
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO regulatory_generation_run_source_snapshots (generation_run_id, regulatory_source_version_id, regulatory_normalized_clause_id, source_hash, clause_locator) VALUES ('task3-run', 'task3-primary-source', 'task3-primary-clause', 'sha256:1111111111111111111111111111111111111111111111111111111111111111', 'OPS PQ 4.450')`); err == nil {
		t.Fatal("generation lineage accepted a source snapshot hash that differs from the versioned source")
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO regulatory_generation_runs (id, status, input_digest, output_digest, input_schema_version, generation_policy_version, provider_catalog_version, provider_adapter_version, inspection_type, target_id, input_artifact, output_artifact) VALUES ('task3-run-conflict', 'GENERATED', 'sha256:ca5ce54a7970d13ad57e658a37a48e6da9bff2012b09c8ee560dc53efd6a1a54', 'sha256:8ef8a964a99530e425523ec8d5f91beede076fb3a540841f0a90eb5f3ff0d06c', '1.0.0', 'regulatory-checklist-v1', '1.0.0', 'fixture-provider-1', 'RAMP_INSPECTION', 'task3-target', '{"requestId":"GENREQ-1"}', '{"candidateId":"CAND-2"}')`); err == nil {
		t.Fatal("one bounded input digest acquired a conflicting output digest")
	}
}

// This catches a publication record that is not pinned to the exact reviewed
// candidate revision, actor assignment, content digest, and command payload.
func TestTask3ReviewAndPublicationRecordsAreAppendOnlyAndApprovalBound(t *testing.T) {
	pool := createTestDatabase(t, "task3_review_publication")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	for _, statement := range []string{
		`INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ('task3-manager', 'test', 'Task 3 Manager'), ('task3-generator', 'test', 'Task 3 Generator')`,
		`INSERT INTO question_versions (id, question_id, version, prompt, configured_reference, expected_evidence, created_by_subject_id) VALUES ('task3-question-v1', 'task3-question', 1, 'Question?', 'Task 3 source', 'Task 3 evidence', 'task3-generator')`,
		`INSERT INTO caa_department_memberships (id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from) VALUES ('task3-manager-membership', 'task3-manager', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'DEPARTMENT_MANAGER', 'ACTIVE', '2025-01-01')`,
		`INSERT INTO organizations (id, legal_name, organization_type, status) VALUES ('task3-review-org', 'Review Operator', 'OPERATOR', 'ACTIVE')`,
		`INSERT INTO regulated_targets (id, target_kind, organization_id) VALUES ('task3-review-target', 'ORGANIZATION', 'task3-review-org')`,
		`INSERT INTO organization_service_provider_scopes (id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, primary_target_id) VALUES ('task3-review-scope', 'task3-review-org', 'AIR_OPERATOR', 'AOC-REVIEW', 'ACTIVE', '2025-01-01', 'task3-review-target')`,
		`INSERT INTO regulatory_source_versions (id, source_identity, version_identity, title, source_class, source_status, source_locator, source_hash, effective_from, source_metadata) VALUES ('task3-review-source', 'TASK3-REVIEW', '1', 'Review source', 'PRIMARY_AUTHORITY', 'PUBLIC_REFERENCE', 'Review locator', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', '2025-01-01', '{}')`,
		`INSERT INTO regulatory_normalized_clauses (id, regulatory_source_version_id, clause_identity, annex_identity, section_identity, clause_locator, source_hash, normalized_digest) VALUES ('task3-review-clause', 'task3-review-source', 'TASK3-REVIEW-1', 'TASK3', '1', 'Review locator', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb')`,
		`INSERT INTO regulatory_generation_runs (id, status, input_digest, output_digest, input_schema_version, generation_policy_version, provider_catalog_version, provider_adapter_version, inspection_type, target_id, input_artifact, output_artifact) VALUES ('task3-review-run', 'GENERATED', 'sha256:52f42b3e05747bf24622088b910cb301bd2e1b898cb7433a40723fb909605df4', 'sha256:49b0285b8dcbd477d155c0ed54051bcdbbfb075dbc045e3ce6de27b4317f2daa', '1.0.0', 'regulatory-checklist-v1', '1.0.0', 'fixture-provider-1', 'RAMP_INSPECTION', 'task3-review-target', '{"requestId":"GENREQ-review"}', '{"candidateId":"CAND-review"}')`,
		`INSERT INTO regulatory_generation_run_scope_facts (generation_run_id, organization_service_provider_scope_id, scope_root_id, organization_id, service_provider_type_id, authorization_identifier, scope_status, effective_from, effective_to, regulated_target_id) SELECT 'task3-review-run', id, root_id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, effective_to, primary_target_id FROM organization_service_provider_scopes WHERE id = 'task3-review-scope'`,
		`INSERT INTO regulatory_generation_run_source_snapshots (generation_run_id, regulatory_source_version_id, regulatory_normalized_clause_id, source_hash, clause_locator) VALUES ('task3-review-run', 'task3-review-source', 'task3-review-clause', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'Review locator')`,
		`INSERT INTO template_masters (id, title, owner_role) VALUES ('task3-template', 'Task 3 generated template', 'Admin Preview')`,
		`INSERT INTO template_draft_versions (id, template_id, version, status, owner_role, creator_subject_id, change_reason, question_version_ids, revision, generation_run_id, candidate_content_digest, candidate_schema_version, candidate_root_id) VALUES ('task3-candidate', 'task3-template', 1, 'GENERATED_DRAFT', 'Admin Preview', 'task3-generator', 'Generated from frozen lineage.', ARRAY['task3-question-v1'], 1, 'task3-review-run', 'sha256:49b0285b8dcbd477d155c0ed54051bcdbbfb075dbc045e3ce6de27b4317f2daa', '1.0.0', 'task3-candidate')`,
		`INSERT INTO candidate_required_owner_assignments (id, candidate_draft_version_id, candidate_revision, candidate_content_digest, department_id, organizational_unit_id, approval_required) VALUES ('task3-required-owner', 'task3-candidate', 1, 'sha256:49b0285b8dcbd477d155c0ed54051bcdbbfb075dbc045e3ce6de27b4317f2daa', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', true)`,
	} {
		if _, err := pool.Exec(context.Background(), statement); err != nil {
			t.Fatalf("persist generated candidate review basis: %v", err)
		}
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO checklist_publication_decisions (id, candidate_draft_version_id, candidate_revision, candidate_content_digest, actor_subject_id, actor_department_membership_id, actor_department_id, actor_organizational_unit_id, reason, decided_at, operation_id, idempotency_key, semantic_payload_digest) VALUES ('task3-unapproved-publication', 'task3-candidate', 1, 'sha256:49b0285b8dcbd477d155c0ed54051bcdbbfb075dbc045e3ce6de27b4317f2daa', 'task3-manager', 'task3-manager-membership', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'Must not publish before technical approval.', now(), 'task3-op-publish-denied', 'task3-idem-publish-denied', 'sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff')`); err == nil {
		t.Fatal("publication decision was accepted without the required technical approval")
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO department_review_decisions (id, candidate_draft_version_id, candidate_revision, candidate_content_digest, decision, actor_subject_id, actor_department_membership_id, actor_department_id, actor_organizational_unit_id, reason, decided_at, operation_id, idempotency_key, semantic_payload_digest) VALUES ('task3-technical-approval', 'task3-candidate', 1, 'sha256:49b0285b8dcbd477d155c0ed54051bcdbbfb075dbc045e3ce6de27b4317f2daa', 'TECHNICALLY_APPROVED', 'task3-manager', 'task3-manager-membership', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'Technically reviewed.', now(), 'task3-op-approve', 'task3-idem-approve', 'sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff')`); err != nil {
		t.Fatalf("persist technical approval: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO department_review_decisions (id, candidate_draft_version_id, candidate_revision, candidate_content_digest, decision, actor_subject_id, actor_department_membership_id, actor_department_id, actor_organizational_unit_id, reason, decided_at, operation_id, idempotency_key, semantic_payload_digest) VALUES ('task3-idempotency-conflict', 'task3-candidate', 1, 'sha256:49b0285b8dcbd477d155c0ed54051bcdbbfb075dbc045e3ce6de27b4317f2daa', 'TECHNICALLY_APPROVED', 'task3-manager', 'task3-manager-membership', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'Different command payload.', now(), 'task3-op-approve-conflict', 'task3-idem-approve', 'sha256:1111111111111111111111111111111111111111111111111111111111111111')`); err == nil {
		t.Fatal("idempotency identity accepted a changed semantic payload")
	}
	if _, err := pool.Exec(context.Background(), `UPDATE template_draft_versions SET change_reason = 'rewritten after generation' WHERE id = 'task3-candidate'`); err == nil {
		t.Fatal("generated candidate revision was mutable")
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO checklist_publication_decisions (id, candidate_draft_version_id, candidate_revision, candidate_content_digest, actor_subject_id, actor_department_membership_id, actor_department_id, actor_organizational_unit_id, reason, decided_at, operation_id, idempotency_key, semantic_payload_digest) VALUES ('task3-publication', 'task3-candidate', 1, 'sha256:49b0285b8dcbd477d155c0ed54051bcdbbfb075dbc045e3ce6de27b4317f2daa', 'task3-manager', 'task3-manager-membership', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'Separate publication decision.', '2026-07-29T12:00:00Z', 'task3-op-publish', 'task3-idem-publish', 'sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff')`); err != nil {
		t.Fatalf("persist separate publication decision: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO checklist_template_versions (id, template_id, version, title, snapshot, published_at, candidate_draft_version_id, candidate_revision, candidate_content_digest, publication_decision_id) VALUES ('task3-published-v1', 'task3-template', 1, 'Task 3 published template', '{"digest":"sha256:49b0285b8dcbd477d155c0ed54051bcdbbfb075dbc045e3ce6de27b4317f2daa"}', '2026-07-29T12:01:00Z', 'task3-candidate', 1, 'sha256:49b0285b8dcbd477d155c0ed54051bcdbbfb075dbc045e3ce6de27b4317f2daa', 'task3-publication')`); err != nil {
		t.Fatalf("bind governed published template: %v", err)
	}
	var technicalActor, technicalMembership, technicalReason, technicalOperation, technicalIdempotency, technicalPayload string
	var publicationActor, publicationMembership, publicationReason, publicationOperation, publicationIdempotency, publicationPayload string
	var candidateID, publishedDigest, publicationID string
	var technicalRevision, publicationRevision, publishedRevision int64
	if err := pool.QueryRow(context.Background(), `SELECT actor_subject_id, actor_department_membership_id, candidate_revision, reason, operation_id, idempotency_key, semantic_payload_digest FROM department_review_decisions WHERE id = 'task3-technical-approval'`).Scan(&technicalActor, &technicalMembership, &technicalRevision, &technicalReason, &technicalOperation, &technicalIdempotency, &technicalPayload); err != nil {
		t.Fatalf("read technical approval record: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT actor_subject_id, actor_department_membership_id, candidate_revision, reason, operation_id, idempotency_key, semantic_payload_digest FROM checklist_publication_decisions WHERE id = 'task3-publication'`).Scan(&publicationActor, &publicationMembership, &publicationRevision, &publicationReason, &publicationOperation, &publicationIdempotency, &publicationPayload); err != nil {
		t.Fatalf("read publication decision record: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT candidate_draft_version_id, candidate_revision, candidate_content_digest, publication_decision_id FROM checklist_template_versions WHERE id = 'task3-published-v1'`).Scan(&candidateID, &publishedRevision, &publishedDigest, &publicationID); err != nil {
		t.Fatalf("read governed published binding: %v", err)
	}
	if technicalActor != "task3-manager" || technicalMembership != "task3-manager-membership" || technicalRevision != 1 || technicalReason != "Technically reviewed." || technicalOperation != "task3-op-approve" || technicalIdempotency != "task3-idem-approve" || technicalPayload != task3SemanticDigest || publicationActor != "task3-manager" || publicationMembership != "task3-manager-membership" || publicationRevision != 1 || publicationReason != "Separate publication decision." || publicationOperation != "task3-op-publish" || publicationIdempotency != "task3-idem-publish" || publicationPayload != task3SemanticDigest || candidateID != "task3-candidate" || publishedRevision != 1 || publishedDigest != task3CandidateDigest || publicationID != "task3-publication" {
		t.Fatalf("attributed decision/publication binding did not persist exactly")
	}
	for _, statement := range []string{
		`INSERT INTO regulatory_evaluations (id, evaluation_identity, purpose) VALUES ('task3-repair-evaluation', 'task3-repair', 'repair preservation')`,
		`INSERT INTO regulatory_evaluation_partitions (id, evaluation_id, partition_kind) VALUES ('task3-repair-training', 'task3-repair-evaluation', 'GENERATION_INPUT')`,
		`INSERT INTO regulatory_evaluation_partition_rows (evaluation_id, partition_id, state_compliance_crosswalk_row_id, stable_row_identity) VALUES ('task3-repair-evaluation', 'task3-repair-training', 'NCAA-CC-A610-ROW-4.2.2.2', 'CC:NAMB:ANNEX6:4.2.2.2')`,
		`INSERT INTO inspections (id, organization_id, assigned_inspector_subject_id, title, inspection_type, status, revision) VALUES ('task3-repair-audit', 'task3-review-org', 'task3-generator', 'Repair binding audit', 'RAMP', 'IN_PROGRESS', 1)`,
		`INSERT INTO inspection_packages (id, inspection_id, checklist_template_version_id, package_version, snapshot, package_digest) VALUES ('task3-repair-package', 'task3-repair-audit', 'task3-published-v1', 1, '{"digest":"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}', 'sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee')`,
	} {
		if _, err := pool.Exec(context.Background(), statement); err != nil {
			t.Fatalf("seed repair preservation history: %v", err)
		}
	}
	repairRows := map[string]string{
		"source":      `SELECT to_jsonb(regulatory_source_versions)::text FROM regulatory_source_versions WHERE id = 'task3-review-source'`,
		"clause":      `SELECT to_jsonb(regulatory_normalized_clauses)::text FROM regulatory_normalized_clauses WHERE id = 'task3-review-clause'`,
		"partition":   `SELECT to_jsonb(regulatory_evaluation_partition_rows)::text FROM regulatory_evaluation_partition_rows WHERE partition_id = 'task3-repair-training'`,
		"scope":       `SELECT to_jsonb(regulatory_generation_run_scope_facts)::text FROM regulatory_generation_run_scope_facts WHERE generation_run_id = 'task3-review-run'`,
		"run":         `SELECT to_jsonb(regulatory_generation_runs)::text FROM regulatory_generation_runs WHERE id = 'task3-review-run'`,
		"snapshot":    `SELECT to_jsonb(regulatory_generation_run_source_snapshots)::text FROM regulatory_generation_run_source_snapshots WHERE generation_run_id = 'task3-review-run'`,
		"candidate":   `SELECT to_jsonb(template_draft_versions)::text FROM template_draft_versions WHERE id = 'task3-candidate'`,
		"owner":       `SELECT to_jsonb(candidate_required_owner_assignments)::text FROM candidate_required_owner_assignments WHERE id = 'task3-required-owner'`,
		"technical":   `SELECT to_jsonb(department_review_decisions)::text FROM department_review_decisions WHERE id = 'task3-technical-approval'`,
		"publication": `SELECT to_jsonb(checklist_publication_decisions)::text FROM checklist_publication_decisions WHERE id = 'task3-publication'`,
		"published":   `SELECT to_jsonb(checklist_template_versions)::text FROM checklist_template_versions WHERE id = 'task3-published-v1'`,
		"package":     `SELECT to_jsonb(inspection_packages)::text FROM inspection_packages WHERE id = 'task3-repair-package'`,
	}
	beforeRepair := map[string]string{}
	for name, query := range repairRows {
		var row string
		if err := pool.QueryRow(context.Background(), query).Scan(&row); err != nil {
			t.Fatalf("capture %s before repair: %v", name, err)
		}
		beforeRepair[name] = row
	}
	if err := migrations.RepairRegulatoryChecklistGovernance(context.Background(), pool); err != nil {
		t.Fatalf("first additive repair: %v", err)
	}
	if err := migrations.RepairRegulatoryChecklistGovernance(context.Background(), pool); err != nil {
		t.Fatalf("idempotent additive repair: %v", err)
	}
	for name, query := range repairRows {
		var after string
		if err := pool.QueryRow(context.Background(), query).Scan(&after); err != nil || after != beforeRepair[name] {
			t.Fatalf("repair changed %s history: before=%q after=%q err=%v", name, beforeRepair[name], after, err)
		}
	}
}

// This catches a schema-only delivery that never imports the supplied CC rows
// into the versioned secondary crosswalk boundary.
func TestTask3ImportsSuppliedAnnex6CrosswalkMetadataWithoutSourceText(t *testing.T) {
	pool := createTestDatabase(t, "task3_supplied_crosswalk_import")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	var sourceClass, sourceStatus, sourceHash, metadata string
	var rowCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT source_class, source_status, source_hash, source_metadata::text
		FROM regulatory_source_versions
		WHERE source_identity = 'NCAA-CC-ANNEX6-PARTI-A610'
	`).Scan(&sourceClass, &sourceStatus, &sourceHash, &metadata); err != nil {
		t.Fatalf("read imported supplied CC source: %v", err)
	}
	if sourceClass != "STATE_COMPLIANCE_CROSSWALK" || sourceStatus != "SUPPLIED_WORKING_COPY" || sourceHash != "sha256:13fe82d1767320443f91ed61cf7d3b4bba0ea24f217fad45bbd9cae5fc682af2" {
		t.Fatalf("supplied CC source identity = %q/%q/%q", sourceClass, sourceStatus, sourceHash)
	}
	if metadata == "" || len(metadata) > 600 || metadata == `{"fullText":true}` {
		t.Fatalf("supplied CC import stored unbounded or absent metadata: %q", metadata)
	}
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM state_compliance_crosswalk_rows WHERE regulatory_source_version_id = 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28'`).Scan(&rowCount); err != nil {
		t.Fatalf("count imported supplied CC rows: %v", err)
	}
	if rowCount != 5 {
		t.Fatalf("imported supplied CC row count = %d, want 5 stable Annex/section identities", rowCount)
	}
}

// This catches a partial migration that leaves a Task 3 table or its
// append-only database guard absent while schema-shaped fixtures still pass.
func TestTask3SchemaInventoryAndRecoveryPreserveGovernedHistory(t *testing.T) {
	pool := createTestDatabase(t, "task3_schema_recovery")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	for _, table := range []string{
		"regulatory_source_versions",
		"regulatory_normalized_clauses",
		"state_compliance_crosswalk_rows",
		"regulatory_evaluations",
		"regulatory_evaluation_partitions",
		"regulatory_evaluation_partition_rows",
		"regulatory_generation_runs",
		"regulatory_generation_run_scope_facts",
		"regulatory_generation_run_source_snapshots",
		"candidate_required_owner_assignments",
		"department_review_decisions",
		"checklist_publication_decisions",
	} {
		var relation *string
		if err := pool.QueryRow(context.Background(), "SELECT to_regclass($1)::text", "public."+table).Scan(&relation); err != nil || relation == nil {
			t.Fatalf("Task 3 table %s absent: relation=%v err=%v", table, relation, err)
		}
		var triggerCount int
		if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM pg_trigger WHERE tgrelid = $1::regclass AND NOT tgisinternal AND tgname = $2`, table, table+"_append_only").Scan(&triggerCount); err != nil {
			t.Fatalf("inspect append-only trigger for %s: %v", table, err)
		}
		if triggerCount != 1 {
			t.Fatalf("Task 3 table %s has %d append-only triggers, want 1", table, triggerCount)
		}
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO regulatory_source_versions (id, source_identity, version_identity, title, source_class, source_status, source_locator, source_hash, effective_from, source_metadata) VALUES ('task3-recovery-source', 'TASK3-RECOVERY', '1', 'Recovery source', 'PRIMARY_AUTHORITY', 'PUBLIC_REFERENCE', 'source locator', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', '2026-07-29', '{}')`); err != nil {
		t.Fatalf("seed non-baseline source history: %v", err)
	}
	var before string
	if err := pool.QueryRow(context.Background(), `SELECT to_jsonb(regulatory_source_versions)::text FROM regulatory_source_versions WHERE id = 'task3-recovery-source'`).Scan(&before); err != nil {
		t.Fatalf("capture source history before repair: %v", err)
	}
	if err := migrations.RepairRegulatoryChecklistGovernance(context.Background(), pool); err != nil {
		t.Fatalf("additive forward repair: %v", err)
	}
	var after string
	if err := pool.QueryRow(context.Background(), `SELECT to_jsonb(regulatory_source_versions)::text FROM regulatory_source_versions WHERE id = 'task3-recovery-source'`).Scan(&after); err != nil {
		t.Fatalf("read source history after repair: %v", err)
	}
	if before != after {
		t.Fatalf("forward repair rewrote Task 3 source history: before=%q after=%q", before, after)
	}
	down, err := os.ReadFile(filepath.Join(apiModuleRoot(t), "migrations", "000021_regulatory_checklist_governance.down.sql"))
	if err != nil {
		t.Fatalf("read guarded down migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(down)); err == nil {
		t.Fatal("guarded down migration accepted non-baseline Task 3 source history")
	}
}
