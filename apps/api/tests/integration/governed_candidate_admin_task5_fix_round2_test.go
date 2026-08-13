package integration_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/regulatory"
	"github.com/aviason/aviaSurveil/internal/testprofile"
	"github.com/aviason/aviaSurveil/migrations"
)

// Break caught: an actual version-21 database can predate Task 5 while its
// migration ledger already says 21. Apply must repair the pre-Task-5 lineage
// functions/triggers as well as create the later Task 5 objects.
func TestTask5FixRound2RepairsGenuineAlreadyLedgeredPreTask5Version21State(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "task5_r2_genuine_v21")
	if _, err := pool.Exec(ctx, `CREATE TABLE schema_migrations (version bigint PRIMARY KEY,name text NOT NULL,applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		t.Fatalf("create migration ledger: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(apiModuleRoot(t), "migrations", "*.up.sql"))
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	for _, file := range files {
		name := filepath.Base(file)
		versionText := strings.TrimLeft(strings.SplitN(name, "_", 2)[0], "0")
		var version int
		if _, err := fmt.Sscanf(versionText, "%d", &version); err != nil {
			t.Fatalf("parse %s version: %v", name, err)
		}
		if version >= 21 {
			continue
		}
		contents, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(contents)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations(version,name) VALUES ($1,$2)`, version, name); err != nil {
			t.Fatalf("record %s: %v", name, err)
		}
	}
	v21Name := "000021_regulatory_checklist_governance.up.sql"
	v21Contents, err := os.ReadFile(filepath.Join(apiModuleRoot(t), "migrations", v21Name))
	if err != nil {
		t.Fatalf("read migration 21: %v", err)
	}
	preTask5, _, found := strings.Cut(string(v21Contents), "-- Task 5 Admin boundary:")
	if !found {
		t.Fatal("migration 21 has no Task 5 boundary marker")
	}
	if _, err := pool.Exec(ctx, preTask5); err != nil {
		t.Fatalf("apply genuine pre-Task-5 migration 21: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations(version,name) VALUES (21,$1)`, v21Name); err != nil {
		t.Fatalf("record already-ledgered version 21: %v", err)
	}
	for _, file := range files {
		name := filepath.Base(file)
		versionText := strings.TrimLeft(strings.SplitN(name, "_", 2)[0], "0")
		var version int
		if _, err := fmt.Sscanf(versionText, "%d", &version); err != nil {
			t.Fatalf("parse %s version: %v", name, err)
		}
		if version <= 21 {
			continue
		}
		contents, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(contents)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations(version,name) VALUES ($1,$2)`, version, name); err != nil {
			t.Fatalf("record %s: %v", name, err)
		}
	}
	var task5Table *string
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.governed_candidate_commands')::text`).Scan(&task5Table); err != nil || task5Table != nil {
		t.Fatalf("fixture is not genuine pre-Task-5 state: table=%v err=%v", task5Table, err)
	}
	if _, err := pool.Exec(ctx, `
		DROP TRIGGER template_draft_versions_generated_lineage_guard ON template_draft_versions;
		DROP TRIGGER regulatory_generation_run_crosswalk_partition_rows_guard ON regulatory_generation_run_crosswalk_partition_rows;
		CREATE OR REPLACE FUNCTION validate_governed_generated_candidate() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RETURN NEW; END $$;
		CREATE OR REPLACE FUNCTION validate_governed_generation_crosswalk_partition() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RETURN NEW; END $$;
	`); err != nil {
		t.Fatalf("corrupt pre-Task-5 lineage guards: %v", err)
	}

	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("forward-repair already-ledgered pre-Task-5 version 21: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap repaired synthetic inputs: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO identity_references(subject_id,issuer,display_name) VALUES ('USR-TASK5-R2-GENUINE-REPAIR','task5-r2','Genuine Repair Admin')`); err != nil {
		t.Fatalf("seed repaired Admin identity: %v", err)
	}
	admin := identity.Principal{SubjectID: "USR-TASK5-R2-GENUINE-REPAIR", Roles: []identity.Role{identity.RoleAdmin}}
	service := regulatory.NewAdminService(pool, func() time.Time {
		return time.Date(2026, 7, 29, 13, 0, 0, 0, time.UTC)
	})
	run, err := service.Import(ctx, admin, "TASK5-R2-GENUINE-IMPORT", "TASK5-R2-GENUINE-IMPORT", regulatory.SyntheticCandidateBundle())
	if err != nil || run.Candidate == nil {
		t.Fatalf("import after genuine version-21 repair: run=%+v err=%v", run, err)
	}
	if _, err := service.CreateRevision(ctx, admin, task5Round2Edit(*run.Candidate, "TASK5-R2-GENUINE-EDIT")); err != nil {
		t.Fatalf("edit successor after genuine version-21 repair: %v", err)
	}
	historyQueries := map[string]string{
		"source":      `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]')::text FROM regulatory_source_versions row`,
		"candidate":   `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]')::text FROM template_draft_versions row`,
		"review":      `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]')::text FROM department_review_decisions row`,
		"publication": `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]')::text FROM checklist_publication_decisions row`,
		"template":    `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]')::text FROM checklist_template_versions row`,
		"audit":       `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]')::text FROM audit_events row`,
		"command":     `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]')::text FROM governed_candidate_commands row`,
	}
	before := map[string]string{}
	for name, query := range historyQueries {
		var snapshot string
		if err := pool.QueryRow(ctx, query).Scan(&snapshot); err != nil {
			t.Fatalf("capture repaired %s history: %v", name, err)
		}
		before[name] = snapshot
	}
	if _, err := pool.Exec(ctx, `
		DROP TRIGGER template_draft_versions_generated_lineage_guard ON template_draft_versions;
		DROP TRIGGER regulatory_generation_run_crosswalk_partition_rows_guard ON regulatory_generation_run_crosswalk_partition_rows;
		CREATE OR REPLACE FUNCTION validate_governed_generated_candidate() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RETURN NEW; END $$;
		CREATE OR REPLACE FUNCTION validate_governed_generation_crosswalk_partition() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RETURN NEW; END $$;
	`); err != nil {
		t.Fatalf("re-corrupt repaired lineage guards: %v", err)
	}
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("idempotent genuine version-21 repair: %v", err)
	}
	for name, query := range historyQueries {
		var after string
		if err := pool.QueryRow(ctx, query).Scan(&after); err != nil || after != before[name] {
			t.Fatalf("repair changed %s history: before=%q after=%q err=%v", name, before[name], after, err)
		}
	}
	for function, phrase := range map[string]string{
		"validate_governed_generated_candidate()":            "complete exact generation lineage",
		"validate_governed_generation_crosswalk_partition()": "exact generation-input partition row",
	} {
		var definition string
		if err := pool.QueryRow(ctx, `SELECT pg_get_functiondef($1::regprocedure)`, function).Scan(&definition); err != nil || !strings.Contains(definition, phrase) {
			t.Fatalf("function %s was not repaired: err=%v definition=%q", function, err, definition)
		}
	}
	for _, trigger := range []string{
		"template_draft_versions_generated_lineage_guard",
		"regulatory_generation_run_crosswalk_partition_rows_guard",
	} {
		var enabled string
		if err := pool.QueryRow(ctx, `SELECT tgenabled::text FROM pg_trigger WHERE tgname=$1 AND NOT tgisinternal`, trigger).Scan(&enabled); err != nil || enabled != "O" {
			t.Fatalf("lineage trigger %s was not repaired: enabled=%q err=%v", trigger, enabled, err)
		}
	}
	var version int64
	if err := pool.QueryRow(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil || version != migrations.LatestVersion {
		t.Fatalf("repair changed migration ledger version=%d err=%v", version, err)
	}
}

func task5Round2Service(t *testing.T, label string) (*regulatory.AdminService, identity.Principal, regulatory.GenerationRunView) {
	t.Helper()
	ctx := context.Background()
	pool := createTestDatabase(t, label)
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migration 21: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap synthetic inputs: %v", err)
	}
	subjectID := "USR-" + label
	if _, err := pool.Exec(ctx, `INSERT INTO identity_references (subject_id,issuer,display_name) VALUES ($1,'task5-r2','Task 5 Round 2 Admin')`, subjectID); err != nil {
		t.Fatalf("seed Admin identity: %v", err)
	}
	admin := identity.Principal{SubjectID: subjectID, Roles: []identity.Role{identity.RoleAdmin}}
	service := regulatory.NewAdminService(pool, func() time.Time {
		return time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	})
	run, err := service.Import(ctx, admin, "TASK5-R2-BASE-"+label, "TASK5-R2-BASE-"+label, regulatory.SyntheticCandidateBundle())
	if err != nil {
		t.Fatalf("import baseline candidate: %v", err)
	}
	return service, admin, run
}

func task5Round2Edit(candidate regulatory.CandidateView, operationID string) regulatory.EditCommand {
	mappings := append([]regulatory.ComplianceMapping(nil), candidate.Mappings...)
	mappings[0].Rationale = "Synthetic test-profile rationale reviewed by an Admin without changing the controlled source claim."
	return regulatory.EditCommand{
		OperationID: operationID, IdempotencyKey: operationID,
		CandidateID: candidate.CandidateID, ExpectedRevision: candidate.Revision,
		ExpectedContentDigest: candidate.ContentDigest,
		ChangeReason:          "Controlled concurrent edit " + operationID,
		Mappings:              mappings,
		Questions:             append([]regulatory.ChecklistQuestion(nil), candidate.Questions...),
		RequiredOwners:        append([]regulatory.RequiredOwner(nil), candidate.RequiredOwners...),
	}
}

// Break caught: operations against one candidate root must wait on one
// transaction-scoped root lock. Without it, edit/submit can commit against
// different rows and create two effects from the same current leaf.
func TestTask5FixRound2SerializesConcurrentRootCommands(t *testing.T) {
	cases := []struct {
		name string
		run  func(context.Context, *regulatory.AdminService, identity.Principal, regulatory.CandidateView, string) error
	}{
		{
			name: "edit versus edit",
			run: func(ctx context.Context, service *regulatory.AdminService, admin identity.Principal, candidate regulatory.CandidateView, suffix string) error {
				_, err := service.CreateRevision(ctx, admin, task5Round2Edit(candidate, "TASK5-R2-EDIT-EDIT-"+suffix))
				return err
			},
		},
		{
			name: "edit versus submit",
			run: func(ctx context.Context, service *regulatory.AdminService, admin identity.Principal, candidate regulatory.CandidateView, suffix string) error {
				if suffix == "A" {
					_, err := service.CreateRevision(ctx, admin, task5Round2Edit(candidate, "TASK5-R2-EDIT-SUBMIT-"+suffix))
					return err
				}
				_, err := service.Submit(ctx, admin, regulatory.SubmitCommand{
					OperationID: "TASK5-R2-EDIT-SUBMIT-" + suffix, IdempotencyKey: "TASK5-R2-EDIT-SUBMIT-" + suffix,
					CandidateID: candidate.CandidateID, ExpectedRevision: candidate.Revision,
					ExpectedContentDigest: candidate.ContentDigest, Reason: "Concurrent exact-leaf submission.",
				})
				return err
			},
		},
		{
			name: "submit versus submit",
			run: func(ctx context.Context, service *regulatory.AdminService, admin identity.Principal, candidate regulatory.CandidateView, suffix string) error {
				_, err := service.Submit(ctx, admin, regulatory.SubmitCommand{
					OperationID: "TASK5-R2-SUBMIT-SUBMIT-" + suffix, IdempotencyKey: "TASK5-R2-SUBMIT-SUBMIT-" + suffix,
					CandidateID: candidate.CandidateID, ExpectedRevision: candidate.Revision,
					ExpectedContentDigest: candidate.ContentDigest, Reason: "Concurrent exact-leaf submission " + suffix,
				})
				return err
			},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			label := "task5_r2_" + strings.ReplaceAll(testCase.name, " ", "_")
			service, admin, run := task5Round2Service(t, label)
			pool := service.Pool
			candidate := *run.Candidate
			lockTx, err := pool.Begin(context.Background())
			if err != nil {
				t.Fatalf("begin root-lock barrier: %v", err)
			}
			defer lockTx.Rollback(context.Background())
			if _, err := lockTx.Exec(context.Background(), `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, candidate.CandidateRootID); err != nil {
				t.Fatalf("hold candidate-root lock: %v", err)
			}

			results := make(chan error, 2)
			start := make(chan struct{})
			var ready sync.WaitGroup
			ready.Add(2)
			for _, suffix := range []string{"A", "B"} {
				suffix := suffix
				go func() {
					ready.Done()
					<-start
					results <- testCase.run(context.Background(), service, admin, candidate, suffix)
				}()
			}
			ready.Wait()
			close(start)
			select {
			case err := <-results:
				t.Fatalf("root command bypassed the held transaction lock: %v", err)
			case <-time.After(150 * time.Millisecond):
			}
			if err := lockTx.Commit(context.Background()); err != nil {
				t.Fatalf("release candidate-root lock: %v", err)
			}
			first, second := <-results, <-results
			successes, conflicts := 0, 0
			for _, result := range []error{first, second} {
				switch {
				case result == nil:
					successes++
				case errors.Is(result, application.ErrConflict):
					conflicts++
				default:
					t.Fatalf("concurrent result=%v, want success or conflict", result)
				}
			}
			if successes != 1 || conflicts != 1 {
				t.Fatalf("success/conflict=%d/%d, want 1/1", successes, conflicts)
			}
			for table, want := range map[string]int{
				"governed_candidate_commands": 2,
				// Import now has a separately attributed source-impact Audit in
				// addition to the import and successful root command Audits.
				"audit_events": 3,
			} {
				var got int
				if err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&got); err != nil || got != want {
					t.Fatalf("%s count=%d want=%d err=%v", table, got, want, err)
				}
			}
			var candidates int
			if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM template_draft_versions WHERE candidate_root_id=$1`, candidate.CandidateRootID).Scan(&candidates); err != nil {
				t.Fatalf("count candidate revisions: %v", err)
			}
			if candidates < 1 || candidates > 2 {
				t.Fatalf("candidate revisions=%d, want one root and at most one successor", candidates)
			}
		})
	}
}

// Break caught: the generation graph and its import ledger/audit evidence
// cannot commit in separate transactions.
func TestTask5FixRound2RollsBackSuccessfulImportGraphWhenCommandPersistenceFails(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "task5_r2_atomic_success")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migration 21: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap synthetic inputs: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO identity_references (subject_id,issuer,display_name) VALUES ('USR-TASK5-R2-ATOMIC','task5-r2','Atomic Admin')`); err != nil {
		t.Fatalf("seed Admin identity: %v", err)
	}
	if _, err := pool.Exec(ctx, `CREATE FUNCTION task5_r2_fail_import_command() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.command_kind='IMPORTED_GENERATION_RUN' THEN RAISE EXCEPTION 'forced command interruption'; END IF; RETURN NEW; END $$; CREATE TRIGGER task5_r2_fail_import_command BEFORE INSERT ON governed_candidate_commands FOR EACH ROW EXECUTE FUNCTION task5_r2_fail_import_command()`); err != nil {
		t.Fatalf("install deterministic command interruption: %v", err)
	}
	service := regulatory.NewAdminService(pool, nil)
	admin := identity.Principal{SubjectID: "USR-TASK5-R2-ATOMIC", Roles: []identity.Role{identity.RoleAdmin}}
	bundle := regulatory.SyntheticCandidateBundle()
	if _, err := service.Import(ctx, admin, "TASK5-R2-ATOMIC-IMPORT", "TASK5-R2-ATOMIC-IMPORT", bundle); err == nil {
		t.Fatal("forced command interruption unexpectedly succeeded")
	}
	for table, want := range map[string]int{
		"regulatory_generation_runs":  0,
		"template_draft_versions":     0,
		"governed_candidate_commands": 0,
		// The synthetic baseline-currentness receipt is an explicit, durable
		// Audit event created by the test-profile bootstrap before the failed
		// import attempt.
		"audit_events": 1,
	} {
		var got int
		if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(&got); err != nil || got != want {
			t.Fatalf("%s count=%d want=%d err=%v", table, got, want, err)
		}
	}
}

// Break caught: malformed imports with missing request/digest fields must
// still have one inspectable FAILED run and one atomic ledger/audit identity.
func TestTask5FixRound2PersistsAndReplaysMissingFieldFailureAtomically(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "task5_r2_atomic_failure")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migration 21: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap synthetic inputs: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO identity_references (subject_id,issuer,display_name) VALUES ('USR-TASK5-R2-FAIL','task5-r2','Failure Admin')`); err != nil {
		t.Fatalf("seed Admin identity: %v", err)
	}
	service := regulatory.NewAdminService(pool, func() time.Time {
		return time.Date(2026, 7, 29, 12, 30, 0, 0, time.UTC)
	})
	admin := identity.Principal{SubjectID: "USR-TASK5-R2-FAIL", Roles: []identity.Role{identity.RoleAdmin}}
	malformed := regulatory.CandidateBundle{}
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := service.Import(ctx, admin, "TASK5-R2-MISSING", "TASK5-R2-MISSING-KEY", malformed); !errors.Is(err, application.ErrInvalid) {
			t.Fatalf("attempt %d error=%v, want invalid", attempt+1, err)
		}
	}
	failed, err := service.GetRun(ctx, admin, "GENRUN-FAILED-R2-MISSING")
	if err != nil {
		t.Fatalf("inspect missing-field failed run: %v", err)
	}
	if failed.Status != "FAILED" || failed.Candidate != nil || failed.Failure == nil ||
		failed.Failure.OperationID != "TASK5-R2-MISSING" ||
		failed.Failure.IdempotencyKey != "TASK5-R2-MISSING-KEY" ||
		failed.Failure.RequestID != "" {
		t.Fatalf("failed run projection is not exact: %+v", failed)
	}
	for table, want := range map[string]int{
		"regulatory_generation_runs":  1,
		"template_draft_versions":     0,
		"governed_candidate_commands": 1,
		// One event is the baseline-currentness receipt; the second is the
		// durable FAILED_IMPORT receipt for this malformed request.
		"audit_events": 2,
	} {
		var got int
		if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(&got); err != nil || got != want {
			t.Fatalf("%s count=%d want=%d err=%v", table, got, want, err)
		}
	}
	var kind, operationID, key, runID string
	var candidateID *string
	if err := pool.QueryRow(ctx, `SELECT command_kind,operation_id,idempotency_key,generation_run_id,candidate_draft_version_id FROM governed_candidate_commands`).Scan(&kind, &operationID, &key, &runID, &candidateID); err != nil {
		t.Fatalf("read failed-import command: %v", err)
	}
	if kind != "FAILED_IMPORT" || operationID != "TASK5-R2-MISSING" || key != "TASK5-R2-MISSING-KEY" || runID != failed.GenerationRunID || candidateID != nil {
		t.Fatalf("failed-import command=%s/%s/%s/%s/%v", kind, operationID, key, runID, candidateID)
	}
	var auditOperationID, correlationID, requestID, entityType, action string
	if err := pool.QueryRow(ctx, `SELECT operation_id,correlation_id,request_id,entity_type,action FROM audit_events WHERE operation_id=$1`, "TASK5-R2-MISSING").Scan(&auditOperationID, &correlationID, &requestID, &entityType, &action); err != nil {
		t.Fatalf("read failed-import audit identity: %v", err)
	}
	if auditOperationID != "TASK5-R2-MISSING" || correlationID != "TASK5-R2-MISSING" ||
		requestID != "" || entityType != "GOVERNED_GENERATION_RUN" || action != "FAILED_IMPORT" {
		t.Fatalf("failed-import audit identity=%s/%s/%q/%s/%s", auditOperationID, correlationID, requestID, entityType, action)
	}

	changed := malformed
	changed.SchemaVersion = "changed"
	if _, err := service.Import(ctx, admin, "TASK5-R2-MISSING", "TASK5-R2-MISSING-KEY", changed); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("changed failed payload reused identity without conflict: %v", err)
	}
	if _, err := service.CreateRevision(ctx, admin, regulatory.EditCommand{
		OperationID: "TASK5-R2-MISSING", IdempotencyKey: "TASK5-R2-MISSING-KEY",
		CandidateID: "CAND-DOES-NOT-EXIST", ExpectedRevision: 1,
		ExpectedContentDigest: "sha256:does-not-exist", ChangeReason: "Cross-command identity collision.",
		Mappings:       []regulatory.ComplianceMapping{{MappingID: "M"}},
		Questions:      []regulatory.ChecklistQuestion{{QuestionID: "Q"}},
		RequiredOwners: []regulatory.RequiredOwner{{DepartmentID: "D", OrganizationalUnitID: "O", ApprovalRequired: true}},
	}); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("failed-import identity reused by edit error=%v, want conflict", err)
	}
}
