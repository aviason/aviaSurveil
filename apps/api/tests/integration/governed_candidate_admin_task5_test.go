package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/regulatory"
	"github.com/aviason/aviaSurveil/internal/testprofile"
	"github.com/aviason/aviaSurveil/migrations"
)

func TestTask5AdminEditsImmutableCandidateAndSubmitsExactRevisionWithoutDownstreamEffects(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "task5_admin_candidate")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migration 21: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ('USR-TASK5-ADMIN', 'task5-test', 'Task 5 Admin')`); err != nil {
		t.Fatalf("create Admin identity: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap synthetic inputs: %v", err)
	}
	admin := identity.Principal{SubjectID: "USR-TASK5-ADMIN", Roles: []identity.Role{identity.RoleAdmin}}
	service := regulatory.NewAdminService(pool, nil)
	bundle := regulatory.SyntheticCandidateBundle()
	run, err := service.Import(ctx, admin, "TASK5-IMPORT-1", "TASK5-IMPORT-1", bundle)
	if err != nil {
		t.Fatalf("import through Admin seam: %v", err)
	}
	if run.Candidate == nil || run.Candidate.CandidateID != bundle.CandidateBundleID || run.Candidate.Status != regulatory.GeneratedDraft || len(run.Candidate.SourceSnapshots) != 1 || len(run.Candidate.ScopeFactIDs) != 1 || len(run.Candidate.CrosswalkPartitionIDs) != 1 {
		t.Fatalf("complete persisted candidate/run projection missing: %+v", run)
	}
	if replay, err := service.Import(ctx, admin, "TASK5-IMPORT-1", "TASK5-IMPORT-1", bundle); err != nil || replay.GenerationRunID != run.GenerationRunID {
		t.Fatalf("identical import replay changed identity: %+v err=%v", replay, err)
	}
	if _, err := service.Import(ctx, admin, "TASK5-IMPORT-1", "TASK5-IMPORT-DIFFERENT-KEY", bundle); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("same import operation with a different idempotency key was accepted: %v", err)
	}
	if _, err := service.Import(ctx, admin, "TASK5-IMPORT-DIFFERENT-OPERATION", "TASK5-IMPORT-1", bundle); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("same import idempotency key with a different operation was accepted: %v", err)
	}
	sources, err := service.ListSources(ctx, admin)
	if err != nil {
		t.Fatalf("list persisted source projection: %v", err)
	}
	var inputSeen, holdoutSeen bool
	for _, source := range sources {
		if source.ClauseID == "CLAUSE-SYNTHETIC-OPS-AOC-1" && len(source.Partitions) == 1 && source.Partitions[0].Role == "GENERATION_INPUT" {
			inputSeen = true
		}
		if source.ClauseID == "CLAUSE-SYNTHETIC-OPS-AOC-HOLDOUT-1" && len(source.Partitions) == 1 && source.Partitions[0].Role == "BLIND_HOLDOUT" {
			holdoutSeen = true
		}
	}
	if !inputSeen || !holdoutSeen {
		t.Fatalf("source projection did not preserve input/holdout partition roles: %+v", sources)
	}
	conflictingBundle := regulatory.SyntheticCandidateBundle()
	conflictingBundle.GenerationRunID = "GENRUN-CONFLICT"
	if _, err := service.Import(ctx, admin, "TASK5-IMPORT-1", "TASK5-IMPORT-1", conflictingBundle); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("conflicting import command was accepted: %v", err)
	}
	failedBundle := regulatory.SyntheticCandidateBundle()
	failedBundle.ComplianceMappings[0].Rationale = "Unsupported import prose."
	if _, err := service.Import(ctx, admin, "TASK5-FAIL-1", "TASK5-FAIL-1", failedBundle); !errors.Is(err, application.ErrInvalid) {
		t.Fatalf("invalid import was accepted: %v", err)
	}
	failedRun, err := service.GetRun(ctx, admin, "GENRUN-FAILED-FAIL-1")
	if err != nil || failedRun.Status != "FAILED" || failedRun.Failure == nil || failedRun.Candidate != nil {
		t.Fatalf("failed run projection missing exact failure identity: %+v err=%v", failedRun, err)
	}

	mappings := append([]regulatory.ComplianceMapping(nil), bundle.ComplianceMappings...)
	mappings[0].Rationale = "Synthetic test-profile rationale reviewed by an Admin without changing the controlled source claim."
	questions := append([]regulatory.ChecklistQuestion(nil), bundle.InspectionChecklist.Questions...)
	owners := []regulatory.RequiredOwner{{DepartmentID: "FLIGHT_OPERATIONS_INSPECTORATE", OrganizationalUnitID: "FLIGHT_OPERATIONS_INSPECTORATE", ApprovalRequired: true}}
	edit := regulatory.EditCommand{OperationID: "TASK5-EDIT-1", IdempotencyKey: "TASK5-EDIT-1", CandidateID: bundle.CandidateBundleID, ExpectedRevision: 1, ExpectedContentDigest: run.Candidate.ContentDigest, ChangeReason: "Correct synthetic candidate rationale without changing source lineage.", Mappings: mappings, Questions: questions, RequiredOwners: owners}
	invalid := edit
	invalid.OperationID, invalid.IdempotencyKey = "TASK5-EDIT-INVALID", "TASK5-EDIT-INVALID"
	invalid.Mappings = append([]regulatory.ComplianceMapping(nil), mappings...)
	invalid.Mappings[0].Rationale = "Invent an unsupported legal compliance conclusion."
	if _, err := service.CreateRevision(ctx, admin, invalid); !errors.Is(err, application.ErrInvalid) {
		t.Fatalf("unsupported candidate prose was accepted: %v", err)
	}
	edited, err := service.CreateRevision(ctx, admin, edit)
	if err != nil {
		t.Fatalf("create immutable successor: %v", err)
	}
	if edited.CandidateID == bundle.CandidateBundleID || edited.CandidateRootID != bundle.CandidateBundleID || edited.SupersedesCandidateID == nil || *edited.SupersedesCandidateID != bundle.CandidateBundleID || edited.Revision != 2 || edited.ContentDigest == bundle.OutputDigest || edited.Status != regulatory.GeneratedDraft {
		t.Fatalf("invalid immutable successor: %+v", edited)
	}
	replayed, err := service.CreateRevision(ctx, admin, edit)
	if err != nil || replayed.CandidateID != edited.CandidateID {
		t.Fatalf("identical edit replay must return exact successor: %+v err=%v", replayed, err)
	}
	editIdentityConflict := edit
	editIdentityConflict.IdempotencyKey = "TASK5-EDIT-DIFFERENT-KEY"
	if _, err := service.CreateRevision(ctx, admin, editIdentityConflict); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("same edit operation with a different idempotency key was accepted: %v", err)
	}
	conflict := edit
	conflict.ChangeReason = "Conflicting semantic retry."
	if _, err := service.CreateRevision(ctx, admin, conflict); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("conflicting idempotency was accepted: %v", err)
	}
	stale := edit
	stale.OperationID, stale.IdempotencyKey, stale.ChangeReason = "TASK5-EDIT-STALE", "TASK5-EDIT-STALE", "Stale parent revision."
	if _, err := service.CreateRevision(ctx, admin, stale); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("stale edit was accepted: %v", err)
	}
	if _, err := service.Submit(ctx, admin, regulatory.SubmitCommand{OperationID: "TASK5-SUBMIT-ANCESTOR", IdempotencyKey: "TASK5-SUBMIT-ANCESTOR", CandidateID: bundle.CandidateBundleID, ExpectedRevision: 1, ExpectedContentDigest: run.Candidate.ContentDigest, Reason: "Ancestor must not be submitted after successor."}); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("ancestor submission was accepted after successor: %v", err)
	}

	submit := regulatory.SubmitCommand{OperationID: "TASK5-SUBMIT-1", IdempotencyKey: "TASK5-SUBMIT-1", CandidateID: edited.CandidateID, ExpectedRevision: edited.Revision, ExpectedContentDigest: edited.ContentDigest, Reason: "Submit exact immutable successor for department review."}
	submitted, err := service.Submit(ctx, admin, submit)
	if err != nil {
		t.Fatalf("submit exact successor: %v", err)
	}
	if submitted.CandidateID != edited.CandidateID || submitted.Revision != edited.Revision || submitted.ContentDigest != edited.ContentDigest || submitted.Status != "DEPARTMENT_REVIEW" {
		t.Fatalf("submission did not transition exact revision: %+v", submitted)
	}
	if _, err := service.Submit(ctx, admin, submit); err != nil {
		t.Fatalf("identical submission replay: %v", err)
	}
	badSubmit := submit
	badSubmit.Reason = "Conflicting submission retry."
	if _, err := service.Submit(ctx, admin, badSubmit); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("conflicting submission idempotency was accepted: %v", err)
	}
	for table, want := range map[string]int{"template_draft_versions": 2, "governed_candidate_commands": 4, "audit_events": 5, "department_review_decisions": 0, "checklist_publication_decisions": 0, "checklist_template_versions": 0, "inspection_packages": 0} {
		var got int
		if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(&got); err != nil || got != want {
			t.Fatalf("%s got=%d want=%d err=%v", table, got, want, err)
		}
	}
	var rootStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM template_draft_versions WHERE id=$1`, bundle.CandidateBundleID).Scan(&rootStatus); err != nil || rootStatus != regulatory.GeneratedDraft {
		t.Fatalf("imported root was overwritten: status=%s err=%v", rootStatus, err)
	}
}

func TestTask5AdminBoundaryRejectsNonAdminAndBlockedRealPathCreatesNoCandidate(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "task5_denial")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migration 21: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ('USR-TASK5-INSPECTOR', 'task5-test', 'Task 5 Inspector')`); err != nil {
		t.Fatal(err)
	}
	service := regulatory.NewAdminService(pool, nil)
	if _, err := service.ListSources(ctx, identity.Principal{SubjectID: "USR-TASK5-INSPECTOR", Roles: []identity.Role{identity.RoleInspector}}); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("non-Admin source access was accepted: %v", err)
	}
	if err := testprofile.BootstrapBlockedRealOPSAOCGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap blocked real facts: %v", err)
	}
	if err := (regulatory.ImportStore{Pool: pool}).ValidateBlockedRealOPSAOCRequest(ctx, regulatory.RealOPSAOCGenerationRequest()); !errors.Is(err, regulatory.ErrBlockedAuthority) {
		t.Fatalf("real source-bound path did not remain blocked: %v", err)
	}
	for _, table := range []string{"regulatory_generation_runs", "template_draft_versions", "governed_candidate_commands"} {
		var got int
		if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(&got); err != nil || got != 0 {
			t.Fatalf("blocked real path left %s=%d err=%v", table, got, err)
		}
	}
}

func TestTask5ForwardRepairRestoresCommandLedgerWithoutChangingVersionOrHistory(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "task5_repair")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migration 21: %v", err)
	}
	if _, err := pool.Exec(ctx, `DROP TABLE governed_candidate_commands`); err != nil {
		t.Fatalf("remove pre-Task5 ledger fixture: %v", err)
	}
	if err := migrations.RepairRegulatoryChecklistGovernance(ctx, pool); err != nil {
		t.Fatalf("repair missing Task5 ledger: %v", err)
	}
	if err := migrations.RepairRegulatoryChecklistGovernance(ctx, pool); err != nil {
		t.Fatalf("idempotent Task5 ledger repair: %v", err)
	}
	var version int64
	var table string
	if err := pool.QueryRow(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil || version != migrations.LatestVersion {
		t.Fatalf("repair changed migration version=%d err=%v", version, err)
	}
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.governed_candidate_commands')::text`).Scan(&table); err != nil || table != "governed_candidate_commands" {
		t.Fatalf("Task5 ledger missing after repair table=%q err=%v", table, err)
	}
}
