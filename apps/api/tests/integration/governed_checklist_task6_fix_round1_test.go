//go:build canonicaltest

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/checklistgovernance"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/regulatory"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/testprofile"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

func waitForTask6AdvisoryWaiters(t *testing.T, service *checklistgovernance.Service, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var got int
		if err := service.Pool.QueryRow(context.Background(), `
			SELECT COUNT(*) FROM pg_locks
			WHERE locktype='advisory' AND NOT granted`).Scan(&got); err != nil {
			t.Fatalf("count advisory waiters: %v", err)
		}
		if got >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("did not observe %d advisory lock waiters", want)
}

// Break caught: filtering membership facts by subject before selecting the
// latest fact per root lets a transferred predecessor retain confidential
// queue and decision authority.
func TestTask6FixRound1TransferredMembershipRevokesPredecessorAuthorityGlobally(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []string{"return", "reject", "approve", "publish"} {
		t.Run(operation, func(t *testing.T) {
			service, predecessor, submitted := task6SubmittedCandidate(t, "task6_fix1_transfer_"+operation)
			if operation == "publish" {
				approved, err := service.Approve(ctx, predecessor, checklistgovernance.ReviewCommand{
					OperationID: "TASK6-FIX1-PRETRANSFER-APPROVE", IdempotencyKey: "TASK6-FIX1-PRETRANSFER-APPROVE",
					CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
					ExpectedContentDigest: submitted.ContentDigest, Reason: "Approve before the authority transfer.",
				})
				if err != nil {
					t.Fatalf("approve before transfer: %v", err)
				}
				submitted = approved
			}
			// Task 2 deliberately forbids cross-subject successors in normal
			// writes. Disable only that trigger to model a legacy/corrupt fact
			// and prove the read boundary still resolves globally and fails
			// closed before filtering the predecessor subject.
			if _, err := service.Pool.Exec(ctx, `
				ALTER TABLE caa_department_memberships DISABLE TRIGGER department_membership_successor_guard;
				INSERT INTO identity_references (subject_id,issuer,display_name)
				VALUES ('USR-TASK6-SUCCESSOR','task6-test','Task 6 Successor');
				INSERT INTO caa_department_memberships
					(id,root_id,supersedes_id,subject_id,department_id,organizational_unit_id,
					 membership_role,status,effective_from)
				VALUES ('MEM-TASK6-FOI-SUCCESSOR','MEM-TASK6-FOI','MEM-TASK6-FOI',
					'USR-TASK6-SUCCESSOR','FLIGHT_OPERATIONS_INSPECTORATE',
					'FLIGHT_OPERATIONS_INSPECTORATE','DEPARTMENT_MANAGER','ACTIVE','2026-01-01');
				ALTER TABLE caa_department_memberships ENABLE TRIGGER department_membership_successor_guard`); err != nil {
				t.Fatalf("transfer membership root to successor subject: %v", err)
			}

			if queue, err := service.ListQueue(ctx, predecessor); !errors.Is(err, application.ErrForbidden) || len(queue) != 0 {
				t.Fatalf("predecessor queue=%+v err=%v, want forbidden empty", queue, err)
			}
			if _, err := service.GetReviewItem(ctx, predecessor, submitted.CandidateID); !errors.Is(err, application.ErrForbidden) {
				t.Fatalf("predecessor detail was visible: %v", err)
			}
			if assignments, err := identity.ResolveEffectiveDepartmentAssignments(
				ctx, service.Pool, predecessor.SubjectID, service.Clock(),
			); err != nil || len(assignments) != 0 {
				t.Fatalf("identity resolver retained predecessor assignments=%+v err=%v", assignments, err)
			}

			command := checklistgovernance.ReviewCommand{
				OperationID:    "TASK6-FIX1-TRANSFER-" + operation,
				IdempotencyKey: "TASK6-FIX1-TRANSFER-" + operation,
				CandidateID:    submitted.CandidateID, ExpectedRevision: submitted.Revision,
				ExpectedContentDigest: submitted.ContentDigest,
				Reason:                "Transferred predecessor must have no lifecycle authority.",
			}
			var err error
			switch operation {
			case "return":
				_, err = service.Return(ctx, predecessor, command)
			case "reject":
				_, err = service.Reject(ctx, predecessor, command)
			case "approve":
				_, err = service.Approve(ctx, predecessor, command)
			case "publish":
				_, err = service.Publish(ctx, predecessor, checklistgovernance.PublicationCommand(command))
			}
			if !errors.Is(err, application.ErrForbidden) {
				t.Fatalf("%s by transferred predecessor error=%v, want forbidden", operation, err)
			}
			assertTask6DecisionEffect(t, service, command.OperationID, map[string]string{
				"return": "RETURNED", "reject": "REJECTED", "approve": "TECHNICALLY_APPROVED",
			}[operation], 0)
			var publicationDecisions int
			if err := service.Pool.QueryRow(ctx,
				`SELECT COUNT(*) FROM checklist_publication_decisions WHERE operation_id=$1`,
				command.OperationID,
			).Scan(&publicationDecisions); err != nil || publicationDecisions != 0 {
				t.Fatalf("%s left publication decisions=%d err=%v", operation, publicationDecisions, err)
			}
		})
	}
}

// Break caught: resolving status before the candidate-root lock lets an
// approval waiting behind a terminal decision commit a partial approval
// against a candidate that has already been returned or rejected.
func TestTask6FixRound1TerminalDecisionWinsBeforeWaitingPartialApproval(t *testing.T) {
	ctx := context.Background()
	for _, terminal := range []string{"return", "reject"} {
		t.Run(terminal, func(t *testing.T) {
			service, manager, submitted := task6SubmittedCandidate(t, "task6_fix1_lock_"+terminal)
			if _, err := service.Pool.Exec(ctx, `
				INSERT INTO candidate_required_owner_assignments
					(id,candidate_draft_version_id,candidate_revision,candidate_content_digest,
					 department_id,organizational_unit_id,approval_required)
				VALUES ('OWNER-TASK6-FIX1-AIR',$1,$2,$3,
					'AIRWORTHINESS_INSPECTORATE','AIRWORTHINESS_INSPECTORATE',true)`,
				submitted.CandidateID, submitted.Revision, submitted.ContentDigest,
			); err != nil {
				t.Fatalf("add second owner: %v", err)
			}
			blocker, err := service.Pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer blocker.Rollback(ctx)
			if _, err := blocker.Exec(ctx,
				`SELECT pg_advisory_xact_lock(hashtextextended($1,0))`,
				submitted.CandidateRootID,
			); err != nil {
				t.Fatalf("hold candidate root lock: %v", err)
			}

			terminalResult := make(chan error, 1)
			go func() {
				command := checklistgovernance.ReviewCommand{
					OperationID:    "TASK6-FIX1-LOCK-" + terminal,
					IdempotencyKey: "TASK6-FIX1-LOCK-" + terminal,
					CandidateID:    submitted.CandidateID, ExpectedRevision: submitted.Revision,
					ExpectedContentDigest: submitted.ContentDigest,
					Reason:                "Terminal decision queued before a partial approval.",
				}
				if terminal == "return" {
					_, err := service.Return(ctx, manager, command)
					terminalResult <- err
					return
				}
				_, err := service.Reject(ctx, manager, command)
				terminalResult <- err
			}()
			waitForTask6AdvisoryWaiters(t, service, 1)

			approvalResult := make(chan error, 1)
			go func() {
				_, err := service.Approve(ctx, manager, checklistgovernance.ReviewCommand{
					OperationID:    "TASK6-FIX1-WAITING-APPROVE-" + terminal,
					IdempotencyKey: "TASK6-FIX1-WAITING-APPROVE-" + terminal,
					CandidateID:    submitted.CandidateID, ExpectedRevision: submitted.Revision,
					ExpectedContentDigest: submitted.ContentDigest,
					Reason:                "This waiting partial approval must lose.",
				})
				approvalResult <- err
			}()
			waitForTask6AdvisoryWaiters(t, service, 2)
			if err := blocker.Commit(ctx); err != nil {
				t.Fatalf("release candidate root lock: %v", err)
			}
			if err := <-terminalResult; err != nil {
				t.Fatalf("%s result: %v", terminal, err)
			}
			if err := <-approvalResult; !errors.Is(err, application.ErrConflict) {
				t.Fatalf("waiting approval error=%v, want conflict", err)
			}
			assertTask6DecisionEffect(
				t, service, "TASK6-FIX1-WAITING-APPROVE-"+terminal,
				"TECHNICALLY_APPROVED", 0,
			)
		})
	}
}

// Break caught: an already-ledgered version 21 that predates Task 6 contains
// the decision/publication tables and triggers but retains the weak actor
// function. Forward repair must replace that definition without applying a
// migration 22 or first installing the current v21 and damaging it.
func TestTask6FixRound1RepairsGenuinePreTask6DecisionActorDefinition(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "task6_fix1_genuine_pretask6_v21")
	if _, err := pool.Exec(ctx, `
		CREATE TABLE schema_migrations (
			version bigint PRIMARY KEY,
			name text NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		t.Fatalf("create migration ledger: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(apiModuleRoot(t), "migrations", "*.up.sql"))
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	for _, file := range files {
		name := filepath.Base(file)
		if strings.HasPrefix(name, "000021_") {
			continue
		}
		contents, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(contents)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
		versionText := strings.TrimLeft(strings.SplitN(name, "_", 2)[0], "0")
		var version int
		if _, err := fmt.Sscanf(versionText, "%d", &version); err != nil {
			t.Fatalf("parse %s version: %v", name, err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations(version,name) VALUES ($1,$2)`,
			version, name,
		); err != nil {
			t.Fatalf("record %s: %v", name, err)
		}
	}
	v21Name := "000021_regulatory_checklist_governance.up.sql"
	v21Bytes, err := os.ReadFile(filepath.Join(apiModuleRoot(t), "migrations", v21Name))
	if err != nil {
		t.Fatalf("read migration 21: %v", err)
	}
	v21 := string(v21Bytes)
	strongMarker := "DECLARE membership record; effective_membership record; department_status text; unit_status text;"
	strongAt := strings.Index(v21, strongMarker)
	if strongAt < 0 {
		t.Fatal("migration 21 has no pre-Task-6 actor-definition boundary")
	}
	functionAt := strings.LastIndex(v21[:strongAt], "CREATE OR REPLACE FUNCTION validate_governed_decision_actor()")
	functionEndRelative := strings.Index(v21[strongAt:], "$$;\n")
	if functionAt < 0 || functionEndRelative < 0 {
		t.Fatal("could not isolate the Task 6 actor-function replacement")
	}
	functionEnd := strongAt + functionEndRelative + len("$$;\n")
	preTask6V21 := v21[:functionAt] + v21[functionEnd:]
	if _, err := pool.Exec(ctx, preTask6V21); err != nil {
		t.Fatalf("apply genuine pre-Task-6 migration 21: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO schema_migrations(version,name) VALUES (21,$1)`,
		v21Name,
	); err != nil {
		t.Fatalf("record already-ledgered version 21: %v", err)
	}
	var before string
	if err := pool.QueryRow(ctx,
		`SELECT pg_get_functiondef('validate_governed_decision_actor()'::regprocedure)`,
	).Scan(&before); err != nil {
		t.Fatalf("read pre-Task-6 actor definition: %v", err)
	}
	if strings.Contains(before, "effective_membership") {
		t.Fatalf("fixture unexpectedly contains Task 6 actor validation: %s", before)
	}

	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("forward repair genuine pre-Task-6 v21: %v", err)
	}
	var after string
	if err := pool.QueryRow(ctx,
		`SELECT pg_get_functiondef('validate_governed_decision_actor()'::regprocedure)`,
	).Scan(&after); err != nil {
		t.Fatalf("read repaired actor definition: %v", err)
	}
	for _, required := range []string{
		"effective_membership",
		"department_status",
		"unit_status",
		"effective_membership.id IS DISTINCT FROM membership.id",
	} {
		if !strings.Contains(after, required) {
			t.Fatalf("forward repair retained weak actor definition; missing %q:\n%s", required, after)
		}
	}
	for _, table := range []string{
		"candidate_required_owner_assignments",
		"department_review_decisions",
		"checklist_publication_decisions",
		"checklist_template_versions",
		"template_version_questions",
	} {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT to_regclass('public.' || $1) IS NOT NULL`, table,
		).Scan(&exists); err != nil || !exists {
			t.Fatalf("repaired inventory missing table %s exists=%v err=%v", table, exists, err)
		}
	}
	for _, indexName := range []string{
		"candidate_required_owner_assignments_review_queue_idx",
		"department_review_decisions_candidate_idx",
		"department_review_decisions_exact_owner_approval_idx",
		"checklist_publication_decisions_candidate_unique_idx",
		"checklist_template_versions_governed_candidate_unique_idx",
		"template_draft_versions_governed_review_queue_idx",
	} {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT to_regclass('public.' || $1) IS NOT NULL`, indexName,
		).Scan(&exists); err != nil || !exists {
			t.Fatalf("repaired inventory missing index %s exists=%v err=%v", indexName, exists, err)
		}
	}
	var pinnedRootColumns int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema='public'
		  AND table_name IN ('department_review_decisions','checklist_publication_decisions')
		  AND column_name='candidate_root_id' AND is_nullable='NO'`,
	).Scan(&pinnedRootColumns); err != nil || pinnedRootColumns != 2 {
		t.Fatalf("root identity columns=%d want=2 err=%v", pinnedRootColumns, err)
	}
	for _, trigger := range []string{
		"department_review_decisions_actor_guard",
		"checklist_publication_decisions_actor_guard",
		"checklist_publication_decisions_approval_guard",
		"checklist_template_versions_governed_publication_guard",
		"template_draft_versions_generated_immutable",
		"department_review_decisions_append_only",
		"checklist_publication_decisions_append_only",
	} {
		var enabled string
		if err := pool.QueryRow(ctx, `
			SELECT tgenabled::text FROM pg_trigger
			WHERE tgname=$1 AND NOT tgisinternal`,
			trigger,
		).Scan(&enabled); err != nil || enabled != "O" {
			t.Fatalf("repaired trigger %s enabled=%q err=%v", trigger, enabled, err)
		}
	}
	var version int64
	if err := pool.QueryRow(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil || version != 21 {
		t.Fatalf("repair changed migration ledger version=%d err=%v", version, err)
	}

	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap repaired lifecycle inputs: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO identity_references(subject_id,issuer,display_name) VALUES
			('USR-TASK6-FIX1-REPAIR-ADMIN','task6-fix1','Repair Admin'),
			('USR-TASK6-FIX1-REPAIR-MANAGER','task6-fix1','Repair Manager');
		INSERT INTO caa_department_memberships
			(id,subject_id,department_id,organizational_unit_id,membership_role,status,effective_from)
		VALUES ('MEM-TASK6-FIX1-REPAIR','USR-TASK6-FIX1-REPAIR-MANAGER',
			'FLIGHT_OPERATIONS_INSPECTORATE','FLIGHT_OPERATIONS_INSPECTORATE',
			'DEPARTMENT_MANAGER','ACTIVE','2025-01-01')`); err != nil {
		t.Fatalf("seed repaired lifecycle identities: %v", err)
	}
	now := time.Date(2026, 7, 29, 18, 0, 0, 0, time.UTC)
	admin := identity.Principal{
		SubjectID: "USR-TASK6-FIX1-REPAIR-ADMIN",
		Roles:     []identity.Role{identity.RoleAdmin},
	}
	adminService := regulatory.NewAdminService(pool, func() time.Time { return now })
	run, err := adminService.Import(
		ctx, admin, "TASK6-FIX1-REPAIR-IMPORT", "TASK6-FIX1-REPAIR-IMPORT",
		regulatory.SyntheticCandidateBundle(),
	)
	if err != nil || run.Candidate == nil {
		t.Fatalf("post-repair import: run=%+v err=%v", run, err)
	}
	submitted, err := adminService.Submit(ctx, admin, regulatory.SubmitCommand{
		OperationID: "TASK6-FIX1-REPAIR-SUBMIT", IdempotencyKey: "TASK6-FIX1-REPAIR-SUBMIT",
		CandidateID: run.Candidate.CandidateID, ExpectedRevision: run.Candidate.Revision,
		ExpectedContentDigest: run.Candidate.ContentDigest, Reason: "Post-repair lifecycle submission.",
	})
	if err != nil {
		t.Fatalf("post-repair submit: %v", err)
	}
	manager := identity.Principal{
		SubjectID: "USR-TASK6-FIX1-REPAIR-MANAGER",
		Roles:     []identity.Role{identity.RoleDepartmentManager},
	}
	lifecycle := checklistgovernance.NewService(pool, func() time.Time { return now })
	approved, err := lifecycle.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "TASK6-FIX1-REPAIR-APPROVE", IdempotencyKey: "TASK6-FIX1-REPAIR-APPROVE",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest, Reason: "Post-repair exact approval.",
	})
	if err != nil {
		t.Fatalf("post-repair approve: %v", err)
	}
	if _, err := lifecycle.Publish(ctx, manager, checklistgovernance.PublicationCommand{
		OperationID: "TASK6-FIX1-REPAIR-PUBLISH", IdempotencyKey: "TASK6-FIX1-REPAIR-PUBLISH",
		CandidateID: approved.CandidateID, ExpectedRevision: approved.Revision,
		ExpectedContentDigest: approved.ContentDigest, Reason: "Post-repair separate publication.",
	}); err != nil {
		t.Fatalf("post-repair publish: %v", err)
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
	beforeRepairAgain := map[string]string{}
	for label, query := range historyQueries {
		var snapshot string
		if err := pool.QueryRow(ctx, query).Scan(&snapshot); err != nil {
			t.Fatalf("capture %s history: %v", label, err)
		}
		beforeRepairAgain[label] = snapshot
	}
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("second idempotent Task 6 repair: %v", err)
	}
	for label, query := range historyQueries {
		var afterRepairAgain string
		if err := pool.QueryRow(ctx, query).Scan(&afterRepairAgain); err != nil ||
			afterRepairAgain != beforeRepairAgain[label] {
			t.Fatalf("second repair changed %s history err=%v", label, err)
		}
	}
}

// Break caught: Task 6 evidence must exercise the exact controlled-procedure
// OPS/AOC request itself. A manager command against an invented candidate ID
// must not turn unresolved Part 140/Part 127/procedure/ownership facts into a
// candidate or any downstream lifecycle effect.
func TestTask6FixRound1BlockedRealOPSAOCServiceAndHTTPHaveZeroLifecycleEffects(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "task6_fix1_blocked_real")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := testprofile.BootstrapBlockedRealOPSAOCGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap blocked real OPS/AOC inputs: %v", err)
	}
	request := regulatory.RealOPSAOCGenerationRequest()
	if err := (regulatory.ImportStore{Pool: pool}).ValidateBlockedRealOPSAOCRequest(ctx, request); !errors.Is(err, regulatory.ErrBlockedAuthority) {
		t.Fatalf("exact real OPS/AOC request error=%v, want blocked authority", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO identity_references(subject_id,issuer,display_name)
		VALUES ('USR-MANAGER-NORA','task6-fix1','Blocked-real Manager')
		ON CONFLICT (subject_id) DO NOTHING;
		INSERT INTO caa_department_memberships
			(id,subject_id,department_id,organizational_unit_id,membership_role,status,effective_from)
		VALUES ('MEM-TASK6-FIX1-REAL-MANAGER','USR-MANAGER-NORA',
			'FLIGHT_OPERATIONS_INSPECTORATE','FLIGHT_OPERATIONS_INSPECTORATE',
			'DEPARTMENT_MANAGER','ACTIVE','2025-01-01')`); err != nil {
		t.Fatalf("seed blocked-real manager: %v", err)
	}
	now := time.Date(2026, 7, 29, 16, 0, 0, 0, time.UTC)
	service := checklistgovernance.NewService(pool, func() time.Time { return now })
	manager := identity.Principal{
		SubjectID: "USR-MANAGER-NORA",
		Roles:     []identity.Role{identity.RoleDepartmentManager},
	}
	const candidateID = "CAND-REAL-OPS-AOC-BLOCKED"
	command := checklistgovernance.ReviewCommand{
		OperationID:    "TASK6-FIX1-REAL-APPROVE",
		IdempotencyKey: "TASK6-FIX1-REAL-APPROVE",
		CandidateID:    candidateID, ExpectedRevision: 1,
		ExpectedContentDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Reason:                "Must remain blocked before candidate creation.",
	}
	if _, err := service.Approve(ctx, manager, command); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("service approval for blocked-real candidate error=%v, want not found", err)
	}
	publicationCommand := checklistgovernance.PublicationCommand(command)
	publicationCommand.OperationID = "TASK6-FIX1-REAL-PUBLISH"
	publicationCommand.IdempotencyKey = "TASK6-FIX1-REAL-PUBLISH"
	if _, err := service.Publish(ctx, manager, publicationCommand); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("service publication for blocked-real candidate error=%v, want not found", err)
	}

	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
		Pool: pool, Clock: service.Clock,
	})
	handler := httpapi.NewCanonicalTestBoundary("task6-fix1-real-token").Protect(api.Handler())
	post := func(path string, body any) *httptest.ResponseRecorder {
		encoded, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encoded))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(httpapi.CanonicalTestTokenHeader, "task6-fix1-real-token")
		req.Header.Set(httpapi.CanonicalTestSubjectHeader, manager.SubjectID)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res
	}
	input := map[string]any{
		"operationId": command.OperationID, "idempotencyKey": command.IdempotencyKey,
		"candidateId": candidateID, "expectedRevision": command.ExpectedRevision,
		"expectedContentDigest": command.ExpectedContentDigest, "reason": command.Reason,
	}
	if response := post(
		"/v1/department-manager/governed-checklist/candidates/"+candidateID+"/technical-approvals",
		input,
	); response.Code != http.StatusNotFound {
		t.Fatalf("blocked-real HTTP approval status=%d body=%s", response.Code, response.Body.String())
	}
	input["operationId"] = publicationCommand.OperationID
	input["idempotencyKey"] = publicationCommand.IdempotencyKey
	if response := post(
		"/v1/department-manager/governed-checklist/candidates/"+candidateID+"/publications",
		input,
	); response.Code != http.StatusNotFound {
		t.Fatalf("blocked-real HTTP publication status=%d body=%s", response.Code, response.Body.String())
	}

	for label, query := range map[string]string{
		"run":         `SELECT COUNT(*) FROM regulatory_generation_runs WHERE input_artifact->>'requestId'='GENREQ-OPS-AOC-0001'`,
		"candidate":   `SELECT COUNT(*) FROM template_draft_versions WHERE id='CAND-REAL-OPS-AOC-BLOCKED'`,
		"review":      `SELECT COUNT(*) FROM department_review_decisions WHERE operation_id LIKE 'TASK6-FIX1-REAL-%'`,
		"publication": `SELECT COUNT(*) FROM checklist_publication_decisions WHERE operation_id LIKE 'TASK6-FIX1-REAL-%'`,
		"version":     `SELECT COUNT(*) FROM checklist_template_versions WHERE candidate_draft_version_id='CAND-REAL-OPS-AOC-BLOCKED'`,
		"audit":       `SELECT COUNT(*) FROM audit_events WHERE operation_id LIKE 'TASK6-FIX1-REAL-%'`,
	} {
		var count int
		if err := pool.QueryRow(ctx, query).Scan(&count); err != nil || count != 0 {
			t.Fatalf("blocked-real %s effects=%d err=%v", label, count, err)
		}
	}
}

// Break caught: terminal detail cannot be reconstructed from an active-only
// queue. The authorized rejected leaf must expose its persisted decision to
// canonical HTTP-backed clients after the transition.
func TestTask6FixRound1RejectedCandidateRetainsAuthorizedTerminalDetail(t *testing.T) {
	ctx := context.Background()
	service, manager, submitted := task6SubmittedCandidate(t, "task6_fix1_terminal_detail")
	rejected, err := service.Reject(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID:    "TASK6-FIX1-TERMINAL-REJECT",
		IdempotencyKey: "TASK6-FIX1-TERMINAL-REJECT",
		CandidateID:    submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest,
		Reason:                "Persist an exact terminal rejection.",
	})
	if err != nil || rejected.Status != "REJECTED" {
		t.Fatalf("reject result=%+v err=%v", rejected, err)
	}
	detail, err := service.GetReviewItem(ctx, manager, rejected.CandidateID)
	if err != nil {
		t.Fatalf("load authorized terminal detail: %v", err)
	}
	if detail.Candidate.Status != "REJECTED" || len(detail.Decisions) != 1 ||
		detail.Decisions[0].Decision != "REJECTED" ||
		detail.Decisions[0].OperationID != "TASK6-FIX1-TERMINAL-REJECT" {
		t.Fatalf("terminal detail lost persisted decision: %+v", detail)
	}
	if queue, err := service.ListQueue(ctx, manager); err != nil || len(queue) != 0 {
		t.Fatalf("terminal candidate leaked into active queue: %+v err=%v", queue, err)
	}
}

func task6FixRound1MultiSubmittedCandidate(t *testing.T, label string) (*checklistgovernance.Service, identity.Principal, regulatory.CandidateView) {
	t.Helper()
	ctx := context.Background()
	pool := createTestDatabase(t, label)
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap synthetic inputs: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO identity_references(subject_id,issuer,display_name) VALUES
			('USR-TASK6-FIX1-MULTI-ADMIN','task6-fix1','Multi Admin'),
			('USR-TASK6-FIX1-MULTI-MANAGER','task6-fix1','Multi Manager');
		INSERT INTO caa_department_memberships
			(id,subject_id,department_id,organizational_unit_id,membership_role,status,effective_from)
		VALUES ('MEM-TASK6-FIX1-MULTI','USR-TASK6-FIX1-MULTI-MANAGER',
			'FLIGHT_OPERATIONS_INSPECTORATE','FLIGHT_OPERATIONS_INSPECTORATE',
			'DEPARTMENT_MANAGER','ACTIVE','2025-01-01')`); err != nil {
		t.Fatalf("seed multi publication identities: %v", err)
	}
	bundle := regulatory.SyntheticCandidateBundle()
	secondMapping := bundle.ComplianceMappings[0]
	secondMapping.MappingID = "MAP-SYNTHETIC-OPS-AOC-002"
	bundle.ComplianceMappings = append(bundle.ComplianceMappings, secondMapping)
	secondQuestion := bundle.InspectionChecklist.Questions[0]
	secondQuestion.QuestionID = "Q-SYNTHETIC-OPS-AOC-002"
	secondQuestion.MappingIDs = []string{secondMapping.MappingID}
	bundle.InspectionChecklist.Questions = append(bundle.InspectionChecklist.Questions, secondQuestion)
	digest, err := regulatory.CanonicalSHA256(map[string]any{
		"complianceMappings":  bundle.ComplianceMappings,
		"inspectionChecklist": bundle.InspectionChecklist,
	})
	if err != nil {
		t.Fatalf("compute multi synthetic digest: %v", err)
	}
	bundle.OutputDigest = digest
	now := time.Date(2026, 7, 29, 17, 0, 0, 0, time.UTC)
	admin := identity.Principal{
		SubjectID: "USR-TASK6-FIX1-MULTI-ADMIN",
		Roles:     []identity.Role{identity.RoleAdmin},
	}
	adminService := regulatory.NewAdminService(pool, func() time.Time { return now })
	run, err := adminService.Import(
		ctx, admin, "TASK6-FIX1-MULTI-IMPORT-"+label,
		"TASK6-FIX1-MULTI-IMPORT-"+label, bundle,
	)
	if err != nil || run.Candidate == nil {
		t.Fatalf("import multi synthetic bundle: run=%+v err=%v", run, err)
	}
	submitted, err := adminService.Submit(ctx, admin, regulatory.SubmitCommand{
		OperationID:    "TASK6-FIX1-MULTI-SUBMIT-" + label,
		IdempotencyKey: "TASK6-FIX1-MULTI-SUBMIT-" + label,
		CandidateID:    run.Candidate.CandidateID, ExpectedRevision: run.Candidate.Revision,
		ExpectedContentDigest: run.Candidate.ContentDigest,
		Reason:                "Submit exact multi-item synthetic candidate.",
	})
	if err != nil {
		t.Fatalf("submit multi synthetic candidate: %v", err)
	}
	manager := identity.Principal{
		SubjectID: "USR-TASK6-FIX1-MULTI-MANAGER",
		Roles:     []identity.Role{identity.RoleDepartmentManager},
	}
	return checklistgovernance.NewService(pool, func() time.Time { return now }), manager, submitted
}

// Break caught: publication must preserve every mapping and ordered question
// byte, not merely the digest and first question link.
func TestTask6FixRound1PublicationPreservesExactMultiItemImmutableSnapshot(t *testing.T) {
	ctx := context.Background()
	service, manager, submitted := task6FixRound1MultiSubmittedCandidate(
		t, "task6_fix1_multi_snapshot",
	)
	approved, err := service.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID:    "TASK6-FIX1-MULTI-APPROVE",
		IdempotencyKey: "TASK6-FIX1-MULTI-APPROVE",
		CandidateID:    submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest,
		Reason:                "Approve the exact multi-item synthetic candidate.",
	})
	if err != nil {
		t.Fatalf("approve multi candidate: %v", err)
	}
	for label, query := range map[string]string{
		"publication decision": `SELECT COUNT(*) FROM checklist_publication_decisions`,
		"template version":     `SELECT COUNT(*) FROM checklist_template_versions`,
		"ordered link":         `SELECT COUNT(*) FROM template_version_questions`,
	} {
		var count int
		if err := service.Pool.QueryRow(ctx, query).Scan(&count); err != nil || count != 0 {
			t.Fatalf("technical approval created %s count=%d err=%v", label, count, err)
		}
	}
	published, err := service.Publish(ctx, manager, checklistgovernance.PublicationCommand{
		OperationID:    "TASK6-FIX1-MULTI-PUBLISH",
		IdempotencyKey: "TASK6-FIX1-MULTI-PUBLISH",
		CandidateID:    approved.CandidateID, ExpectedRevision: approved.Revision,
		ExpectedContentDigest: approved.ContentDigest,
		Reason:                "Publish exact multi-item bytes separately.",
	})
	if err != nil {
		t.Fatalf("publish multi candidate: %v", err)
	}
	var rawSnapshot []byte
	if err := service.Pool.QueryRow(ctx,
		`SELECT snapshot FROM checklist_template_versions WHERE id=$1`,
		published.TemplateVersionID,
	).Scan(&rawSnapshot); err != nil {
		t.Fatalf("read published snapshot: %v", err)
	}
	var actual any
	if err := json.Unmarshal(rawSnapshot, &actual); err != nil {
		t.Fatalf("decode published snapshot: %v", err)
	}
	expected := map[string]any{
		"candidateId":            submitted.CandidateID,
		"candidateRevision":      submitted.Revision,
		"candidateContentDigest": submitted.ContentDigest,
		"complianceMappings":     submitted.Mappings,
		"questions":              submitted.Questions,
	}
	expectedBytes, _ := json.Marshal(expected)
	var canonicalExpected any
	if err := json.Unmarshal(expectedBytes, &canonicalExpected); err != nil {
		t.Fatalf("canonicalize expected publication snapshot: %v", err)
	}
	expectedBytes, _ = json.Marshal(canonicalExpected)
	actualBytes, _ := json.Marshal(actual)
	if !bytes.Equal(actualBytes, expectedBytes) {
		t.Fatalf("published canonical snapshot mismatch\nactual=%s\nwant=%s", actualBytes, expectedBytes)
	}
	var linkedIDs, candidateIDs []string
	if err := service.Pool.QueryRow(ctx, `
		SELECT array_agg(question_version_id ORDER BY position)
		FROM template_version_questions WHERE template_version_id=$1`,
		published.TemplateVersionID,
	).Scan(&linkedIDs); err != nil {
		t.Fatalf("read ordered published question IDs: %v", err)
	}
	if err := service.Pool.QueryRow(ctx,
		`SELECT question_version_ids FROM template_draft_versions WHERE id=$1`,
		submitted.CandidateID,
	).Scan(&candidateIDs); err != nil {
		t.Fatalf("read candidate question IDs: %v", err)
	}
	if len(linkedIDs) != 2 || !equalStrings(linkedIDs, candidateIDs) {
		t.Fatalf("ordered published IDs=%v candidate IDs=%v", linkedIDs, candidateIDs)
	}
	var decisionLink, auditSubject string
	if err := service.Pool.QueryRow(ctx, `
		SELECT version.publication_decision_id,audit.actor_subject_id
		FROM checklist_template_versions version
		JOIN audit_events audit ON audit.operation_id='TASK6-FIX1-MULTI-PUBLISH'
		WHERE version.id=$1`,
		published.TemplateVersionID,
	).Scan(&decisionLink, &auditSubject); err != nil ||
		decisionLink != published.PublicationDecisionID ||
		auditSubject != manager.SubjectID {
		t.Fatalf("publication linkage decision=%s auditActor=%s err=%v", decisionLink, auditSubject, err)
	}
	for label, statement := range map[string]string{
		"version update":           `UPDATE checklist_template_versions SET title='tampered' WHERE id='` + published.TemplateVersionID + `'`,
		"ordered link delete":      `DELETE FROM template_version_questions WHERE template_version_id='` + published.TemplateVersionID + `'`,
		"question snapshot update": `UPDATE question_versions SET prompt='tampered' WHERE id='` + linkedIDs[0] + `'`,
	} {
		if _, err := service.Pool.Exec(ctx, statement); err == nil {
			t.Fatalf("%s unexpectedly changed immutable publication history", label)
		}
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// Break caught: changing any persisted content/ordering/digest edge after
// approval must abort the entire separate publication transaction.
func TestTask6FixRound1EachPublicationTamperEdgeRollsBackCompletely(t *testing.T) {
	ctx := context.Background()
	for _, edge := range []string{"mapping", "question", "ordering", "expected-digest"} {
		t.Run(edge, func(t *testing.T) {
			service, manager, submitted := task6FixRound1MultiSubmittedCandidate(
				t, "task6_fix1_tamper_"+strings.ReplaceAll(edge, "-", "_"),
			)
			approved, err := service.Approve(ctx, manager, checklistgovernance.ReviewCommand{
				OperationID:    "TASK6-FIX1-TAMPER-APPROVE-" + edge,
				IdempotencyKey: "TASK6-FIX1-TAMPER-APPROVE-" + edge,
				CandidateID:    submitted.CandidateID, ExpectedRevision: submitted.Revision,
				ExpectedContentDigest: submitted.ContentDigest,
				Reason:                "Approve before one isolated tamper edge.",
			})
			if err != nil {
				t.Fatalf("approve before %s tamper: %v", edge, err)
			}
			switch edge {
			case "mapping":
				if _, err := service.Pool.Exec(ctx, `ALTER TABLE regulatory_generated_mapping_snapshots DISABLE TRIGGER regulatory_generated_mapping_snapshots_append_only`); err != nil {
					t.Fatalf("disable mapping guard: %v", err)
				}
				if _, err := service.Pool.Exec(ctx, `
					UPDATE regulatory_generated_mapping_snapshots
					SET snapshot=jsonb_set(snapshot,'{rationale}','"tampered mapping"'::jsonb)
					WHERE candidate_draft_version_id=$1 AND mapping_id='MAP-SYNTHETIC-OPS-AOC-001'`,
					approved.CandidateID); err != nil {
					t.Fatalf("simulate mapping tamper: %v", err)
				}
				if _, err := service.Pool.Exec(ctx, `ALTER TABLE regulatory_generated_mapping_snapshots ENABLE TRIGGER regulatory_generated_mapping_snapshots_append_only`); err != nil {
					t.Fatalf("restore mapping guard: %v", err)
				}
			case "question":
				if _, err := service.Pool.Exec(ctx, `ALTER TABLE regulatory_generated_question_snapshots DISABLE TRIGGER regulatory_generated_question_snapshots_append_only`); err != nil {
					t.Fatalf("disable question guard: %v", err)
				}
				if _, err := service.Pool.Exec(ctx, `
					UPDATE regulatory_generated_question_snapshots
					SET snapshot=jsonb_set(snapshot,'{prompt}','"tampered question"'::jsonb)
					WHERE candidate_draft_version_id=$1 AND question_id='Q-SYNTHETIC-OPS-AOC-001'`,
					approved.CandidateID); err != nil {
					t.Fatalf("simulate question tamper: %v", err)
				}
				if _, err := service.Pool.Exec(ctx, `ALTER TABLE regulatory_generated_question_snapshots ENABLE TRIGGER regulatory_generated_question_snapshots_append_only`); err != nil {
					t.Fatalf("restore question guard: %v", err)
				}
			case "ordering":
				if _, err := service.Pool.Exec(ctx, `ALTER TABLE template_draft_versions DISABLE TRIGGER template_draft_versions_generated_immutable`); err != nil {
					t.Fatalf("disable candidate guard: %v", err)
				}
				if _, err := service.Pool.Exec(ctx, `
					UPDATE template_draft_versions
					SET question_version_ids=ARRAY[
						question_version_ids[2],question_version_ids[1]
					] WHERE id=$1`,
					approved.CandidateID); err != nil {
					t.Fatalf("simulate ordering tamper: %v", err)
				}
				if _, err := service.Pool.Exec(ctx, `ALTER TABLE template_draft_versions ENABLE TRIGGER template_draft_versions_generated_immutable`); err != nil {
					t.Fatalf("restore candidate guard: %v", err)
				}
			}
			expectedDigest := approved.ContentDigest
			if edge == "expected-digest" {
				expectedDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
			}
			operationID := "TASK6-FIX1-TAMPER-PUBLISH-" + edge
			if _, err := service.Publish(ctx, manager, checklistgovernance.PublicationCommand{
				OperationID: operationID, IdempotencyKey: operationID,
				CandidateID: approved.CandidateID, ExpectedRevision: approved.Revision,
				ExpectedContentDigest: expectedDigest,
				Reason:                "Every isolated tamper edge must roll back publication.",
			}); !errors.Is(err, application.ErrConflict) {
				t.Fatalf("%s tamper publication error=%v, want conflict", edge, err)
			}
			for label, query := range map[string]string{
				"publication": `SELECT COUNT(*) FROM checklist_publication_decisions WHERE operation_id=$1`,
				"version":     `SELECT COUNT(*) FROM checklist_template_versions WHERE candidate_draft_version_id=$1`,
				"ordered link": `SELECT COUNT(*) FROM template_version_questions WHERE template_version_id IN (
					SELECT id FROM checklist_template_versions WHERE candidate_draft_version_id=$1)`,
				"audit": `SELECT COUNT(*) FROM audit_events WHERE operation_id=$1`,
			} {
				argument := any(approved.CandidateID)
				if label == "publication" || label == "audit" {
					argument = operationID
				}
				var count int
				if err := service.Pool.QueryRow(ctx, query, argument).Scan(&count); err != nil || count != 0 {
					t.Fatalf("%s tamper left %s=%d err=%v", edge, label, count, err)
				}
			}
			var status string
			if err := service.Pool.QueryRow(ctx,
				`SELECT status FROM template_draft_versions WHERE id=$1`,
				approved.CandidateID,
			).Scan(&status); err != nil || status != "TECHNICALLY_APPROVED" {
				t.Fatalf("%s tamper changed candidate status=%s err=%v", edge, status, err)
			}
		})
	}
}
