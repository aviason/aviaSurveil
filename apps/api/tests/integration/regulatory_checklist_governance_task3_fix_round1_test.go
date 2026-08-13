package integration_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/testprofile"
	"github.com/aviason/aviaSurveil/migrations"
)

// This catches a generated candidate that merely names a run while omitting
// the run's exact effective provider scope/source snapshot and output digest.
func TestTask3FixRound1RejectsUnlinkedOrMismatchedGeneratedCandidate(t *testing.T) {
	pool := createTestDatabase(t, "task3r1_candidate_linkage")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	for _, statement := range []string{
		`INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ('task3r1-generator', 'test', 'Generator')`,
		`INSERT INTO organizations (id, legal_name, organization_type, status) VALUES ('task3r1-org', 'Task 3 R1 operator', 'OPERATOR', 'ACTIVE')`,
		`INSERT INTO regulated_targets (id, target_kind, organization_id) VALUES ('task3r1-target', 'ORGANIZATION', 'task3r1-org')`,
		`INSERT INTO organization_service_provider_scopes (id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, primary_target_id) VALUES ('task3r1-scope', 'task3r1-org', 'AIR_OPERATOR', 'AOC-T3R1', 'ACTIVE', '2025-01-01', 'task3r1-target')`,
		`INSERT INTO regulatory_generation_runs (id, status, input_digest, output_digest, input_schema_version, generation_policy_version, provider_catalog_version, provider_adapter_version, inspection_type, target_id, input_artifact, output_artifact) VALUES ('task3r1-run', 'GENERATED', 'sha256:a996ea1286ef2f75b848b4edd1f8e5dea2eb045145c1adb2cb536c573ee6d249', 'sha256:1e6f52acbdfb4660273b1471e657afcc571bed19212dfce3c000156001cbef63', '1.0.0', 'policy-v1', '1.0.0', 'fixture-v1', 'RAMP_INSPECTION', 'task3r1-target', '{"request":"task3r1"}', '{"output":"task3r1"}')`,
		`INSERT INTO template_masters (id, title, owner_role) VALUES ('task3r1-template', 'Task 3 R1 template', 'Admin Preview')`,
	} {
		if _, err := pool.Exec(context.Background(), statement); err != nil {
			t.Fatalf("seed candidate linkage fixture: %v", err)
		}
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO template_draft_versions (
			id, template_id, version, status, owner_role, creator_subject_id,
			change_reason, question_version_ids, revision, generation_run_id,
			candidate_content_digest, candidate_schema_version
		) VALUES (
			'task3r1-empty-unlinked', 'task3r1-template', 1, 'GENERATED_DRAFT',
			'Admin Preview', 'task3r1-generator', 'Must reject unlinked run.', '{}', 1,
			'task3r1-run', 'sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc', '1.0.0'
		)
	`); err == nil {
		t.Fatal("generated candidate accepted an unlinked run, empty question identity content, and unrelated output digest")
	}
}

// This catches a decision guard that trusts a historical ACTIVE membership
// instead of resolving its effective successor and current unit status facts.
func TestTask3FixRound1DecisionActorRequiresEffectiveMembershipAndUnit(t *testing.T) {
	pool := createTestDatabase(t, "task3r1_decision_actor")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	for _, statement := range []string{
		`INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ('task3r1-manager', 'test', 'Manager'), ('task3r1-generator-2', 'test', 'Generator')`,
		`INSERT INTO question_versions (id, question_id, version, prompt, configured_reference, expected_evidence, created_by_subject_id) VALUES ('task3r1-question-v1', 'task3r1-question', 1, 'Question?', 'Reference', 'Evidence', 'task3r1-generator-2')`,
		`INSERT INTO caa_department_memberships (id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from) VALUES ('task3r1-member-active', 'task3r1-manager', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'DEPARTMENT_MANAGER', 'ACTIVE', '2025-01-01')`,
		`INSERT INTO organizations (id, legal_name, organization_type, status) VALUES ('task3r1-actor-org', 'Actor operator', 'OPERATOR', 'ACTIVE')`,
		`INSERT INTO regulated_targets (id, target_kind, organization_id) VALUES ('task3r1-actor-target', 'ORGANIZATION', 'task3r1-actor-org')`,
		`INSERT INTO organization_service_provider_scopes (id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, primary_target_id) VALUES ('task3r1-actor-scope', 'task3r1-actor-org', 'AIR_OPERATOR', 'AOC-ACTOR', 'ACTIVE', '2025-01-01', 'task3r1-actor-target')`,
		`INSERT INTO regulatory_source_versions (id, source_identity, version_identity, title, source_class, source_status, source_locator, source_hash, effective_from, source_metadata) VALUES ('task3r1-actor-source', 'ACTOR-SOURCE', '1', 'Actor source', 'PRIMARY_AUTHORITY', 'PUBLIC_REFERENCE', 'Actor locator', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', '2025-01-01', '{}')`,
		`INSERT INTO regulatory_normalized_clauses (id, regulatory_source_version_id, clause_identity, annex_identity, section_identity, clause_locator, source_hash, normalized_digest) VALUES ('task3r1-actor-clause', 'task3r1-actor-source', 'ACTOR-1', 'ACTOR', '1', 'Actor locator', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb')`,
		`INSERT INTO regulatory_generation_runs (id, status, input_digest, output_digest, input_schema_version, generation_policy_version, provider_catalog_version, provider_adapter_version, inspection_type, target_id, input_artifact, output_artifact) VALUES ('task3r1-actor-run', 'GENERATED', 'sha256:acd79a9b04d41e44ae222295f1210bd1d61835a6b439b650220794341905d7ff', 'sha256:6cb519e2449c9aed11c88cfef2755e0c5288209ead4d953a3956e8d37c8d79a4', '1.0.0', 'policy', '1.0.0', 'fixture', 'RAMP', 'task3r1-actor-target', '{"request":"actor"}', '{"output":"actor"}')`,
		`INSERT INTO regulatory_generation_run_scope_facts (generation_run_id, organization_service_provider_scope_id, scope_root_id, organization_id, service_provider_type_id, authorization_identifier, scope_status, effective_from, effective_to, regulated_target_id) SELECT 'task3r1-actor-run', id, root_id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, effective_to, primary_target_id FROM organization_service_provider_scopes WHERE id = 'task3r1-actor-scope'`,
		`INSERT INTO regulatory_generation_run_source_snapshots (generation_run_id, regulatory_source_version_id, regulatory_normalized_clause_id, source_hash, clause_locator) VALUES ('task3r1-actor-run', 'task3r1-actor-source', 'task3r1-actor-clause', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'Actor locator')`,
		`INSERT INTO template_masters (id, title, owner_role) VALUES ('task3r1-actor-template', 'Actor template', 'Admin Preview')`,
		`INSERT INTO template_draft_versions (id, template_id, version, status, owner_role, creator_subject_id, change_reason, question_version_ids, revision, generation_run_id, candidate_content_digest, candidate_schema_version, candidate_root_id) VALUES ('task3r1-actor-candidate', 'task3r1-actor-template', 1, 'GENERATED_DRAFT', 'Admin Preview', 'task3r1-generator-2', 'Actor test candidate.', ARRAY['task3r1-question-v1'], 1, 'task3r1-actor-run', 'sha256:6cb519e2449c9aed11c88cfef2755e0c5288209ead4d953a3956e8d37c8d79a4', '1.0.0', 'task3r1-actor-candidate')`,
		`INSERT INTO candidate_required_owner_assignments (id, candidate_draft_version_id, candidate_revision, candidate_content_digest, department_id, organizational_unit_id, approval_required) VALUES ('task3r1-actor-owner', 'task3r1-actor-candidate', 1, 'sha256:6cb519e2449c9aed11c88cfef2755e0c5288209ead4d953a3956e8d37c8d79a4', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', true)`,
		`INSERT INTO caa_department_memberships (id, root_id, supersedes_id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from) VALUES ('task3r1-member-revoked', 'task3r1-member-active', 'task3r1-member-active', 'task3r1-manager', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'DEPARTMENT_MANAGER', 'REVOKED', '2026-01-01')`,
	} {
		if _, err := pool.Exec(context.Background(), statement); err != nil {
			t.Fatalf("seed effective actor fixture: %v", err)
		}
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO department_review_decisions (id, candidate_draft_version_id, candidate_revision, candidate_content_digest, decision, actor_subject_id, actor_department_membership_id, actor_department_id, actor_organizational_unit_id, reason, decided_at, operation_id, idempotency_key, semantic_payload_digest) VALUES ('task3r1-stale-membership', 'task3r1-actor-candidate', 1, 'sha256:6cb519e2449c9aed11c88cfef2755e0c5288209ead4d953a3956e8d37c8d79a4', 'TECHNICALLY_APPROVED', 'task3r1-manager', 'task3r1-member-active', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'Must reject stale assignment.', '2026-07-29T10:00:00Z', 'task3r1-op-stale', 'task3r1-idem-stale', 'sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff')`); err == nil {
		t.Fatal("revoked membership predecessor authorized a technical decision")
	}
}

// This catches a rollback exception that recognizes only the seeded source ID
// while silently deleting adopted clauses or rows under that same source.
func TestTask3FixRound1RollbackRefusesAdoptedSeededSourceChildren(t *testing.T) {
	pool := createTestDatabase(t, "task3r1_seeded_child_rollback")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO regulatory_normalized_clauses (id, regulatory_source_version_id, clause_identity, annex_identity, section_identity, clause_locator, source_hash, normalized_digest)
		VALUES ('task3r1-adopted-seeded-clause', 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28', 'ADOPTED-1', 'ANNEX_6_PART_I', 'ADOPTED-1', 'Adopted locator', 'sha256:13fe82d1767320443f91ed61cf7d3b4bba0ea24f217fad45bbd9cae5fc682af2', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa');
		INSERT INTO state_compliance_crosswalk_rows (id, regulatory_source_version_id, normalized_clause_id, stable_row_identity, annex_identity, section_identity, row_digest)
		VALUES ('task3r1-adopted-seeded-row', 'NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28', 'task3r1-adopted-seeded-clause', 'CC:NAMB:ANNEX6:ADOPTED-1', 'ANNEX_6_PART_I', 'ADOPTED-1', 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa')
	`); err != nil {
		t.Fatalf("seed adopted CC children: %v", err)
	}
	down, err := os.ReadFile(filepath.Join(apiModuleRoot(t), "migrations", "000021_regulatory_checklist_governance.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(down)); err == nil {
		t.Fatal("guarded rollback deleted adopted clause/row under migration-owned CC source")
	}
	for _, id := range []string{"task3r1-adopted-seeded-clause", "task3r1-adopted-seeded-row"} {
		var exists bool
		table := "regulatory_normalized_clauses"
		if id == "task3r1-adopted-seeded-row" {
			table = "state_compliance_crosswalk_rows"
		}
		if err := pool.QueryRow(context.Background(), `SELECT EXISTS (SELECT 1 FROM `+table+` WHERE id = $1)`, id).Scan(&exists); err != nil || !exists {
			t.Fatalf("rollback refusal lost adopted history %s: exists=%t err=%v", id, exists, err)
		}
	}
}

// This catches historical-version coverage that checks the checklist template
// alone but not the real immutable Audit package pinned to its released
// canonical scope snapshot.
func TestTask3FixRound1CanonicalCabinAuditPackageBindingRemainsExact(t *testing.T) {
	pool := createTestDatabase(t, "task3r1_historical_binding")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := testprofile.Reset(context.Background(), pool, testprofile.CanonicalScenarioTime()); err != nil {
		t.Fatalf("seed canonical historical profile: %v", err)
	}
	var inspectionID, canonicalScopeSnapshotID, digest, snapshot string
	var packageVersion int
	if err := pool.QueryRow(context.Background(), `SELECT inspection_id, canonical_scope_snapshot_id, package_version, package_digest, snapshot::text FROM inspection_packages WHERE id = 'PKG-CAB-2026-001'`).Scan(&inspectionID, &canonicalScopeSnapshotID, &packageVersion, &digest, &snapshot); err != nil {
		t.Fatalf("read canonical Audit package binding: %v", err)
	}
	if inspectionID != "AUD-2026-001" || canonicalScopeSnapshotID != "scope-snapshot-package-001" || packageVersion != 1 || digest != "sha256:candidate-cabin-package-v1" || snapshot == "" {
		t.Fatalf("canonical Audit package binding changed: %q/%q/%d/%q/%q", inspectionID, canonicalScopeSnapshotID, packageVersion, digest, snapshot)
	}
}

// This catches same-named or partial schema objects that a relation-only
// inventory would incorrectly accept as governed Task 3 enforcement.
func TestTask3FixRound1SemanticGovernedCatalogInventory(t *testing.T) {
	pool := createTestDatabase(t, "task3r1_semantic_inventory")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	type triggerExpectation struct {
		relation, function string
		eventMask          int
	}
	for trigger, want := range map[string]triggerExpectation{
		"regulatory_generation_run_scope_facts_append_only":      {"regulatory_generation_run_scope_facts", "governed_append_only_guard", 24},
		"regulatory_generation_run_scope_facts_guard":            {"regulatory_generation_run_scope_facts", "validate_governed_generation_scope_fact", 4},
		"template_draft_versions_generated_lineage_guard":        {"template_draft_versions", "validate_governed_generated_candidate", 4},
		"department_review_decisions_actor_guard":                {"department_review_decisions", "validate_governed_decision_actor", 4},
		"checklist_publication_decisions_actor_guard":            {"checklist_publication_decisions", "validate_governed_decision_actor", 4},
		"checklist_publication_decisions_approval_guard":         {"checklist_publication_decisions", "validate_governed_publication_approval", 4},
		"checklist_template_versions_governed_publication_guard": {"checklist_template_versions", "validate_governed_published_template", 4},
	} {
		var relation, function string
		var eventMask int
		var before bool
		if err := pool.QueryRow(context.Background(), `SELECT relation.relname, function.proname, trigger.tgtype::int & 60, (trigger.tgtype::int & 2) = 2 FROM pg_trigger trigger JOIN pg_class relation ON relation.oid = trigger.tgrelid JOIN pg_proc function ON function.oid = trigger.tgfoid WHERE trigger.tgname = $1 AND NOT trigger.tgisinternal`, trigger).Scan(&relation, &function, &eventMask, &before); err != nil {
			t.Fatalf("inspect trigger %s: %v", trigger, err)
		}
		if relation != want.relation || function != want.function || eventMask != want.eventMask || !before {
			t.Fatalf("trigger %s = %q/%q/%d/before=%t", trigger, relation, function, eventMask, before)
		}
	}
	assertTask3CatalogSHA256(t, pool)
}

func assertTask3CatalogSHA256(t *testing.T, pool *database.Pool) {
	t.Helper()
	// The placeholders intentionally establish RED before the fresh catalog
	// digests are pinned below. Every complete definition is normalized in SQL
	// before Go hashes it, so whitespace-only PostgreSQL formatting is ignored.
	functions := map[string]string{
		"validate_governed_generation_scope_fact": "sha256:81dee2f59fec1e135bcfbe482af3266e138720054202747090be629d70704161",
		"validate_governed_generated_candidate":   "sha256:26c1ef1e1f3c3734885bbb916c210de7504963a1af5b498bd02c24639b92c5b2",
		"validate_governed_decision_actor":        "sha256:979a1f9c4101495d7e9ad977aac3f5908806a7c7bf5105debb8bd262f92ec759",
		"validate_governed_publication_approval":  "sha256:0359bdf9228fc14dbe3f606d563290762ec3a8a785a4cf38780e182277552293",
		"validate_governed_published_template":    "sha256:0e020db4f7b10ffe168ee649cc87c2b83bc71e85fa58670a4dcfd8c41bca7632",
	}
	constraints := map[string]string{
		"caa_departments":                             "sha256:717e0cbba58f5e4cae3e1c00011d4e2bdf1fcd0730d9445d8f9fd7a7658aaeac",
		"caa_organizational_units":                    "sha256:0b85c8a4be4ffd057818fceadccf7576d33eea7ce958ff49fcdb8b5711581fa9",
		"caa_department_memberships":                  "sha256:7f9865f9ce12a01b07e28dc34b69a84b3d4fef59db621152ff93d82d7526dcfd",
		"caa_department_status_facts":                 "sha256:b71a49c0de8d0375d6c6a1696d4ff41ed08f8c4b4c191d98a29e7a5a30cd9582",
		"caa_organizational_unit_status_facts":        "sha256:de4b34586e41fed9663882c88817e18a115b4d890594c085b54ef1426a0e031e",
		"service_provider_types":                      "sha256:80a6c7480b8fb56ce14376b46543ade7aa2a1d7e2087310e75d2a961d8d83ad0",
		"regulated_targets":                           "sha256:f2ce662924881e0bbe03840b96686cef5fdfdde79800c311d55a21313d5afcf5",
		"organization_service_provider_scopes":        "sha256:9259d64adbafa9cbe75e6c064ea1b893055708d202122a3ca4d7e51493756e6c",
		"organization_service_provider_scope_targets": "sha256:a1d30604e7730d5e6ab8d5aa4a16348325d2a4e5e9a0ba326ff123a662601759",
		"regulatory_source_versions":                  "sha256:95a607a6ba2c2df4d9cafbe0c1b52570822ea2ee063f707d8f83894041e885cd",
		"regulatory_normalized_clauses":               "sha256:6224628f8a089d0b4baef64c97ef6c5141a421ce270d4671c56a7b405992899e",
		"state_compliance_crosswalk_rows":             "sha256:dc65be3bddffbc1aa512bb07c917b03c8701d789be67ea06d225547420151ab2",
		"regulatory_evaluations":                      "sha256:3df44d8ede59e8bc96cc6dc950e793727aa201c3ef80ecc4e0753972cf8055bd",
		"regulatory_evaluation_partitions":            "sha256:11270726ba2b8edd5373b8edcbde76ccd42ea3de59d4fab0922470e6760e3957",
		"regulatory_evaluation_partition_rows":        "sha256:1114f7114c786f4fb48f12ba6fdd7ba901ef0b7f5c79dc5f958a0e9906fd3020",
		"regulatory_generation_runs":                  "sha256:e2f396e64569dec64fe7bfa326a55a575fb3da4bc6bcf1ef0ab8f9acf0ba26e4",
		"regulatory_generation_run_scope_facts":       "sha256:79127aebea9929c953e64aa29fc7e1fe6d48611a35fd7140b24159338e92919f",
		"regulatory_generation_run_source_snapshots":  "sha256:b36801e1951a5d6fd7cba29ac856259b6d5eea0c727a9cd1cb244772bda38e1d",
		"regulatory_generated_mapping_snapshots":      "sha256:afd6c63fe290f799f1906045ac1e8a598310d740b7ee9d89fff77f34547a3e8b",
		"template_draft_versions":                     "sha256:cb3205342d51c3c7c1f74aa3e8430fd4de3240eb565d43e4bb9f9abf15b59d55",
		"candidate_required_owner_assignments":        "sha256:8354714cacad796b5f19f93e35e26ea45a0ed348b318cf8c98333c87e166332b",
		"department_review_decisions":                 "sha256:6ff9fcd302cd4c320fcf0a69eb2641c969ee76e318167a2e7236f45e48647284",
		"checklist_publication_decisions":             "sha256:2efef6c07f104672e36eca104c78ffb646c86a72328c00f2a12d0eac49acf110",
		"checklist_template_versions":                 "sha256:1bfbabb1020120df65a4d465cb0d33fa556f86312c981e7b61f4a649fc6bcf49",
	}
	indexes := map[string]string{
		"caa_department_membership_root_identity_idx":           "sha256:1a2a69ba4f35c95bf6fb1579e29b8da2e8bd22e5c8e74b2432ec1d64f0c07e8e",
		"caa_department_memberships_effective_authority_idx":    "sha256:c2ddfafda5d4f3f7e6ce1ca3ecdb7dfeeeaf1c5c80f8de27e21ebdf8c02cd732",
		"caa_department_status_fact_root_identity_idx":          "sha256:9fc173bc244fa11e3b10f5ed9c62680e3af7d99f29d02db09983e60795a8a476",
		"caa_department_status_facts_effective_idx":             "sha256:04d49a14bce90c8dc4d4b1dd42bd48ab2d940096dcbe234a05e7b3aef75c9337",
		"caa_organizational_unit_status_fact_root_identity_idx": "sha256:8e896155f2648286384df7ee32c5121bd19519cffa394d3fecda29200356a6c8",
		"caa_organizational_unit_status_facts_effective_idx":    "sha256:f758b7f770da4eed42e6b685cec67075139cd13d3e7a086b3bbc025dc4a1b5d0",
		"organization_service_provider_scope_root_identity_idx": "sha256:dc75092873dae06bb416d12bd20c7e06f54fbd9ea6fd9a692e65603e303dbc18",
		"organization_service_provider_scope_applicability_idx": "sha256:aa9fabd6c22213a80d5447ce803a7a6e5b3219ec7b489081cbfb4df254628cce",
		"regulatory_generation_run_scope_facts_lookup_idx":      "sha256:9e19dce5f623b6a58020c866851698303269ec1d7ba7eefe3f1b0aac5e83ff1e",
	}

	actual := map[string]string{}
	mismatches := []string{}
	for name, expected := range functions {
		var definition string
		if err := pool.QueryRow(context.Background(), `SELECT regexp_replace(pg_get_functiondef($1::regprocedure), '\\s+', ' ', 'g')`, name+"()").Scan(&definition); err != nil {
			t.Fatalf("read complete function definition for %s: %v", name, err)
		}
		actual["function/"+name] = catalogSHA256(definition)
		if actual["function/"+name] != expected {
			mismatches = append(mismatches, "function/"+name)
		}
	}
	for relation, expected := range constraints {
		var definition string
		if err := pool.QueryRow(context.Background(), `SELECT regexp_replace(COALESCE(string_agg(pg_get_constraintdef(oid), E'\n' ORDER BY conname), ''), '\\s+', ' ', 'g') FROM pg_constraint WHERE conrelid = $1::regclass`, relation).Scan(&definition); err != nil {
			t.Fatalf("read complete sorted constraints for %s: %v", relation, err)
		}
		actual["constraint/"+relation] = catalogSHA256(definition)
		if actual["constraint/"+relation] != expected {
			mismatches = append(mismatches, "constraint/"+relation)
		}
	}
	for name, expected := range indexes {
		var definition string
		if err := pool.QueryRow(context.Background(), `SELECT regexp_replace(pg_get_indexdef($1::regclass), '\\s+', ' ', 'g')`, name).Scan(&definition); err != nil {
			t.Fatalf("read complete index definition for %s: %v", name, err)
		}
		actual["index/"+name] = catalogSHA256(definition)
		if actual["index/"+name] != expected {
			mismatches = append(mismatches, "index/"+name)
		}
	}

	if len(mismatches) > 0 {
		keys := make([]string, 0, len(actual))
		for key := range actual {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			t.Logf("%s=%s", key, actual[key])
		}
		sort.Strings(mismatches)
		t.Fatalf("complete Task 3 catalog definition hashes changed: %v", mismatches)
	}
}

func catalogSHA256(definition string) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(definition)))
}
