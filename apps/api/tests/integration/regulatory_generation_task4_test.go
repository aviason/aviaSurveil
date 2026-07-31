package integration_test

import (
	"context"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/regulatory"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/testprofile"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

func TestTask4SyntheticImportPersistsCompleteLineageAndReplaysExactly(t *testing.T) {
	pool := createTestDatabase(t, "task4_import")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(context.Background(), pool); err != nil {
		t.Fatalf("bootstrap synthetic lineage: %v", err)
	}
	store := regulatory.ImportStore{Pool: pool}
	candidate := regulatory.SyntheticCandidateBundle()
	first, err := store.Import(context.Background(), candidate)
	if err != nil {
		t.Fatalf("first transactional import: %v", err)
	}
	if first.Replayed {
		t.Fatal("first import was marked replayed")
	}
	second, err := store.Import(context.Background(), candidate)
	if err != nil {
		t.Fatalf("identical replay: %v", err)
	}
	if !second.Replayed || second.GenerationRunID != first.GenerationRunID || second.InputDigest != first.InputDigest || second.OutputDigest != first.OutputDigest {
		t.Fatalf("replay identity changed: first=%+v second=%+v", first, second)
	}
	for table, want := range map[string]int{"regulatory_generation_runs": 1, "regulatory_generation_run_scope_facts": 1, "regulatory_generation_run_source_snapshots": 1, "regulatory_generation_run_crosswalk_partition_rows": 1, "question_versions": 1, "template_draft_versions": 1, "regulatory_generated_mapping_snapshots": 1, "regulatory_generated_question_snapshots": 1, "candidate_required_owner_assignments": 1, "department_review_decisions": 0, "checklist_publication_decisions": 0, "inspection_packages": 0} {
		var count int
		if err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil || count != want {
			t.Fatalf("%s count=%d err=%v want=%d", table, count, err, want)
		}
	}
	var mappingSnapshot, questionSnapshot string
	if err := pool.QueryRow(context.Background(), `SELECT snapshot::text FROM regulatory_generated_mapping_snapshots WHERE candidate_draft_version_id=$1`, candidate.CandidateBundleID).Scan(&mappingSnapshot); err != nil || !strings.Contains(mappingSnapshot, `"relationship": "ADDRESSES"`) || !strings.Contains(mappingSnapshot, `"locator": "Synthetic OPS/AOC 1"`) {
		t.Fatalf("mapping snapshot lost semantics: %q err=%v", mappingSnapshot, err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT snapshot::text FROM regulatory_generated_question_snapshots WHERE candidate_draft_version_id=$1`, candidate.CandidateBundleID).Scan(&questionSnapshot); err != nil || !strings.Contains(questionSnapshot, `"mandatoryCore": true`) || !strings.Contains(questionSnapshot, `"expectedEvidence"`) {
		t.Fatalf("question snapshot lost semantics: %q err=%v", questionSnapshot, err)
	}
}

func TestTask4ConflictingOutputRollsBackWithoutGovernanceSideEffects(t *testing.T) {
	pool := createTestDatabase(t, "task4_conflict")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(context.Background(), pool); err != nil {
		t.Fatalf("bootstrap synthetic lineage: %v", err)
	}
	store := regulatory.ImportStore{Pool: pool}
	candidate := regulatory.SyntheticCandidateBundle()
	if _, err := store.Import(context.Background(), candidate); err != nil {
		t.Fatalf("first import: %v", err)
	}
	conflict := regulatory.SyntheticCandidateBundle()
	conflict.InspectionChecklist.Questions[0].Prompt = "Different bytes for one bounded request."
	var err error
	conflict.OutputDigest, err = regulatory.CanonicalSHA256(map[string]any{"complianceMappings": conflict.ComplianceMappings, "inspectionChecklist": conflict.InspectionChecklist})
	if err != nil {
		t.Fatalf("digest conflicting output: %v", err)
	}
	if _, err := store.Import(context.Background(), conflict); err == nil {
		t.Fatal("conflicting output for one input digest was accepted")
	}
	for table, want := range map[string]int{"regulatory_generation_runs": 1, "template_draft_versions": 1, "department_review_decisions": 0, "checklist_publication_decisions": 0, "inspection_packages": 0} {
		var count int
		if err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil || count != want {
			t.Fatalf("%s count=%d err=%v want=%d", table, count, err, want)
		}
	}
}

func TestTask4ImportFailsClosedWhenSyntheticPrerequisitesAreAbsent(t *testing.T) {
	pool := createTestDatabase(t, "task4_missing")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if _, err := (regulatory.ImportStore{Pool: pool}).Import(context.Background(), regulatory.SyntheticCandidateBundle()); err == nil {
		t.Fatal("import accepted missing target/scope/source prerequisites")
	}
	var runs int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM regulatory_generation_runs`).Scan(&runs); err != nil || runs != 0 {
		t.Fatalf("failed import left generation runs=%d err=%v", runs, err)
	}
}

func TestTask4ReplayFailsClosedWhenAnImmutableGraphEdgeIsMissing(t *testing.T) {
	pool := createTestDatabase(t, "task4_replay_graph")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(context.Background(), pool); err != nil {
		t.Fatalf("bootstrap synthetic lineage: %v", err)
	}
	store := regulatory.ImportStore{Pool: pool}
	candidate := regulatory.SyntheticCandidateBundle()
	if _, err := store.Import(context.Background(), candidate); err != nil {
		t.Fatalf("first import: %v", err)
	}
	// This disposable-database test bypasses append-only triggers solely to prove
	// replay readback refuses an otherwise matching digest with a missing edge.
	if _, err := pool.Exec(context.Background(), `ALTER TABLE regulatory_generated_mapping_snapshots DISABLE TRIGGER ALL`); err != nil {
		t.Fatalf("disable test-only snapshot trigger: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `DELETE FROM regulatory_generated_mapping_snapshots WHERE candidate_draft_version_id=$1`, candidate.CandidateBundleID); err != nil {
		t.Fatalf("delete test-only snapshot: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `ALTER TABLE regulatory_generated_mapping_snapshots ENABLE TRIGGER ALL`); err != nil {
		t.Fatalf("restore test-only snapshot trigger: %v", err)
	}
	if _, err := store.Import(context.Background(), candidate); err == nil || !strings.Contains(err.Error(), "replay graph") {
		t.Fatalf("replay accepted missing immutable mapping snapshot: %v", err)
	}
}

func TestTask4GenerationLineageRejectsBlindHoldoutRows(t *testing.T) {
	pool := createTestDatabase(t, "task4_holdout")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(context.Background(), pool); err != nil {
		t.Fatalf("bootstrap synthetic lineage: %v", err)
	}
	candidate := regulatory.SyntheticCandidateBundle()
	if _, err := (regulatory.ImportStore{Pool: pool}).Import(context.Background(), candidate); err != nil {
		t.Fatalf("persist valid disposable run for holdout rejection: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO regulatory_generation_run_crosswalk_partition_rows (generation_run_id, evaluation_partition_id, state_compliance_crosswalk_row_id, stable_row_identity) VALUES ($1, 'PARTITION-SYNTHETIC-HOLDOUT', 'CCROW-SYNTHETIC-OPS-AOC-HOLDOUT-1', 'CC:SYNTHETIC:OPS:AOC:HOLDOUT:1')`, candidate.GenerationRunID); err == nil {
		t.Fatal("generation lineage accepted a blind-holdout row")
	}
}

func TestTask4BlockedRealRequestResolvesPersistedFactsAndHasNoSideEffects(t *testing.T) {
	pool := createTestDatabase(t, "task4_blocked_real")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := testprofile.BootstrapBlockedRealOPSAOCGenerationInputs(context.Background(), pool); err != nil {
		t.Fatalf("bootstrap blocked real OPS/AOC facts: %v", err)
	}
	request := regulatory.RealOPSAOCGenerationRequest()
	if err := (regulatory.ImportStore{Pool: pool}).ValidateBlockedRealOPSAOCRequest(context.Background(), request); err != regulatory.ErrBlockedAuthority {
		t.Fatalf("real request did not resolve then block: %v", err)
	}
	if _, err := regulatory.NewImportedResultProvider(regulatory.SyntheticCandidateBundle()).Generate(context.Background(), request); err != regulatory.ErrBlockedAuthority {
		t.Fatalf("provider did not block resolved real request: %v", err)
	}
	for table, want := range map[string]int{"regulatory_generation_runs": 0, "template_draft_versions": 0, "department_review_decisions": 0, "checklist_publication_decisions": 0, "checklist_template_versions": 0, "inspection_packages": 0} {
		var count int
		if err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil || count != want {
			t.Fatalf("%s count=%d err=%v want=%d", table, count, err, want)
		}
	}
}

func TestTask4ReplayRejectsSameCountWrongLineageValues(t *testing.T) {
	cases := []struct {
		name, table, update string
	}{
		{"scope", "regulatory_generation_run_scope_facts", `UPDATE regulatory_generation_run_scope_facts SET authorization_identifier='WRONG-AUTHORIZATION' WHERE generation_run_id=$1`},
		{"source locator", "regulatory_generation_run_source_snapshots", `UPDATE regulatory_generation_run_source_snapshots SET clause_locator='Wrong source locator' WHERE generation_run_id=$1`},
		{"partition row", "regulatory_generation_run_crosswalk_partition_rows", `UPDATE regulatory_generation_run_crosswalk_partition_rows SET stable_row_identity='CC:WRONG:ROW' WHERE generation_run_id=$1`},
		{"required owner", "candidate_required_owner_assignments", `UPDATE candidate_required_owner_assignments SET department_id='AIRWORTHINESS_INSPECTORATE', organizational_unit_id='AIRWORTHINESS_INSPECTORATE' WHERE candidate_draft_version_id=$1`},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			pool := createTestDatabase(t, "task4_replay_"+strings.ReplaceAll(scenario.name, " ", "_"))
			if err := migrations.Apply(context.Background(), pool); err != nil {
				t.Fatalf("apply migrations: %v", err)
			}
			if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(context.Background(), pool); err != nil {
				t.Fatalf("bootstrap synthetic lineage: %v", err)
			}
			store := regulatory.ImportStore{Pool: pool}
			candidate := regulatory.SyntheticCandidateBundle()
			if _, err := store.Import(context.Background(), candidate); err != nil {
				t.Fatalf("first import: %v", err)
			}
			if _, err := pool.Exec(context.Background(), `ALTER TABLE `+scenario.table+` DISABLE TRIGGER ALL`); err != nil {
				t.Fatalf("disable test-only lineage trigger: %v", err)
			}
			identity := candidate.GenerationRunID
			if scenario.table == "candidate_required_owner_assignments" {
				identity = candidate.CandidateBundleID
			}
			if _, err := pool.Exec(context.Background(), scenario.update, identity); err != nil {
				t.Fatalf("replace same-count lineage value: %v", err)
			}
			if _, err := pool.Exec(context.Background(), `ALTER TABLE `+scenario.table+` ENABLE TRIGGER ALL`); err != nil {
				t.Fatalf("restore test-only lineage trigger: %v", err)
			}
			if _, err := store.Import(context.Background(), candidate); err == nil || !strings.Contains(err.Error(), "replay graph") {
				t.Fatalf("replay accepted same-count wrong %s value: %v", scenario.name, err)
			}
		})
	}
}

func TestTask4NodeCLIImportUsesTaskCreatedLoopbackDatabaseAndReplays(t *testing.T) {
	pool := createTestDatabase(t, "task4_node_cli")
	var databaseName string
	if err := pool.QueryRow(context.Background(), `SELECT current_database()`).Scan(&databaseName); err != nil {
		t.Fatalf("read task-created database name: %v", err)
	}
	baseURL := os.Getenv("AVIA_TEST_DATABASE_URL")
	if baseURL == "" {
		baseURL = "postgres://aviasurveil:aviasurveil@127.0.0.1:55432/aviasurveil?sslmode=disable"
	}
	databaseURL, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse task database URL: %v", err)
	}
	databaseURL.Path = "/" + databaseName
	repositoryRoot := filepath.Clean(filepath.Join(apiModuleRoot(t), "..", ".."))
	fixture := filepath.Join(repositoryRoot, "docs", "regulatory-sources", "fixtures", "synthetic-ops-aoc-generation-candidate.v1.json")
	runImport := func() string {
		command := exec.Command("node", "scripts/regulatory/import-checklist-candidate.mjs", fixture)
		command.Dir = repositoryRoot
		command.Env = append(os.Environ(), "AVIA_REGULATORY_TEST_MODE=1", "AVIA_REGULATORY_DATABASE_URL="+databaseURL.String(), "GOCACHE=/private/tmp/avia-task4-fix2-gocache")
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("run actual Node import command: %v output=%s", err, output)
		}
		return string(output)
	}
	if output := runImport(); !strings.Contains(output, "replayed=false") {
		t.Fatalf("first Node import output = %q", output)
	}
	if output := runImport(); !strings.Contains(output, "replayed=true") {
		t.Fatalf("second Node import output = %q", output)
	}
	candidate := regulatory.SyntheticCandidateBundle()
	for table, want := range map[string]int{"regulatory_generation_runs": 1, "regulatory_generation_run_scope_facts": 1, "regulatory_generation_run_source_snapshots": 1, "regulatory_generation_run_crosswalk_partition_rows": 1, "regulatory_generated_mapping_snapshots": 1, "regulatory_generated_question_snapshots": 1, "candidate_required_owner_assignments": 1, "department_review_decisions": 0, "checklist_publication_decisions": 0, "checklist_template_versions": 0, "inspection_packages": 0} {
		var count int
		if err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil || count != want {
			t.Fatalf("Node CLI %s count=%d err=%v want=%d", table, count, err, want)
		}
	}
	var runID, ownerDepartment, ownerUnit string
	if err := pool.QueryRow(context.Background(), `SELECT generation_run_id FROM template_draft_versions WHERE id=$1`, candidate.CandidateBundleID).Scan(&runID); err != nil || runID != candidate.GenerationRunID {
		t.Fatalf("Node CLI candidate/run identity mismatch: %q err=%v", runID, err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT department_id, organizational_unit_id FROM candidate_required_owner_assignments WHERE candidate_draft_version_id=$1`, candidate.CandidateBundleID).Scan(&ownerDepartment, &ownerUnit); err != nil || ownerDepartment != "FLIGHT_OPERATIONS_INSPECTORATE" || ownerUnit != "FLIGHT_OPERATIONS_INSPECTORATE" {
		t.Fatalf("Node CLI owner mismatch: %q/%q err=%v", ownerDepartment, ownerUnit, err)
	}
}
