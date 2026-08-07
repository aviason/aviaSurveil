//go:build canonicaltest

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/assignments"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/planning"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/questioncatalog"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/testprofile"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

func TestRoutinePlanningReturnReentryAndAssignmentMaterialization(t *testing.T) {
	pool := canonicalDatabase(t, "planning_assignment_routine")
	seedPlanningActors(t, pool)
	seedPlanningDraft(t, pool, "draft-routine", "airline-xyz")

	planningService := planning.NewService(pool, planning.Dependencies{
		Clock:       func() time.Time { return canonicalNow },
		IDGenerator: scenarioIDGenerator(),
	})
	manager := principal("manager-001", "caa", "session-manager", identity.RoleDepartmentManager)
	finance := principal("finance-001", "caa", "session-finance", identity.RoleFinance)
	gm := principal("gm-001", "caa", "session-gm", identity.RoleGeneralManager)
	executive := principal("executive-001", "caa", "session-executive", identity.RoleExecutiveDirector)

	draft, err := planningService.GetIntakeDraft(context.Background(), manager, "draft-routine")
	if err != nil {
		t.Fatalf("get routine draft: %v", err)
	}
	draft, err = planningService.SaveIntakeDraft(context.Background(), manager, planning.SaveIntakeDraftCommand{
		OperationID: "op-routine-save-1", IdempotencyKey: "idem-routine-save-1",
		DraftID: draft.ID, ExpectedRevision: draft.Revision,
		Values: routineIntakeValues(0),
	})
	if err != nil {
		t.Fatalf("save zero-budget routine draft: %v", err)
	}
	submitted, err := planningService.SubmitIntake(context.Background(), manager, planning.SubmitIntakeCommand{
		OperationID: "op-routine-submit-1", IdempotencyKey: "idem-routine-submit-1",
		DraftID: draft.ID, PlanningItemID: "plan-routine", ExpectedRevision: draft.Revision,
	})
	if err != nil {
		t.Fatalf("submit zero-budget routine draft: %v", err)
	}
	if submitted.PlanningItem.EstimatedBudget != 0 ||
		submitted.PlanningItem.Status != planning.StatusFinanceReview ||
		submitted.PlanningItem.CurrentOwnerRole != identity.RoleFinance {
		t.Fatalf("zero-budget planning item = %+v", submitted.PlanningItem)
	}

	returned := decidePlanning(t, planningService, finance, submitted.PlanningItem,
		"op-routine-finance-return", planning.DecisionReturnForRevision)
	if returned.Status != planning.StatusReturned || returned.CurrentOwnerRole != identity.RoleDepartmentManager {
		t.Fatalf("returned planning item = %+v", returned)
	}
	assertCommandTransactionLink(t, pool, "op-routine-finance-return", "finance-001:planning_decision")
	correctedValues := routineIntakeValues(12500)
	correctedValues.Purpose = "Cabin safety and emergency equipment, with corrected resource allocation."
	corrected, err := planningService.SaveIntakeDraft(context.Background(), manager, planning.SaveIntakeDraftCommand{
		OperationID: "op-routine-save-2", IdempotencyKey: "idem-routine-save-2",
		DraftID: submitted.Draft.ID, ExpectedRevision: submitted.Draft.Revision,
		Values: correctedValues,
	})
	if err != nil {
		t.Fatalf("save returned routine draft: %v", err)
	}
	resubmitted, err := planningService.SubmitIntake(context.Background(), manager, planning.SubmitIntakeCommand{
		OperationID: "op-routine-submit-2", IdempotencyKey: "idem-routine-submit-2",
		DraftID: corrected.ID, PlanningItemID: "plan-routine", ExpectedRevision: corrected.Revision,
	})
	if err != nil {
		t.Fatalf("resubmit returned routine draft: %v", err)
	}
	if resubmitted.PlanningItem.Status != planning.StatusFinanceReview ||
		resubmitted.PlanningItem.Revision != returned.Revision+1 ||
		resubmitted.PlanningItem.EstimatedBudget != 12500 {
		t.Fatalf("resubmitted planning item = %+v", resubmitted.PlanningItem)
	}

	approvedBudget := decidePlanning(t, planningService, finance, resubmitted.PlanningItem,
		"op-routine-finance-approve", planning.DecisionApproveBudget)
	forwarded := decidePlanning(t, planningService, gm, approvedBudget,
		"op-routine-gm-forward", planning.DecisionForwardForFinalApproval)
	approved := decidePlanning(t, planningService, executive, forwarded,
		"op-routine-executive-approve", planning.DecisionApprovePlan)
	released := decidePlanning(t, planningService, gm, approved,
		"op-routine-gm-release", planning.DecisionReleasePlan)

	assignmentService := assignments.NewService(pool, assignments.Dependencies{
		Clock:       func() time.Time { return canonicalNow },
		IDGenerator: scenarioIDGenerator(),
	})
	preparation, err := assignmentService.Prepare(context.Background(), manager, assignments.PrepareCommand{
		OperationID: "op-routine-prepare", IdempotencyKey: "idem-routine-prepare",
		PlanningItemID: released.ID, InspectionID: "audit-routine",
		ExpectedPlanningRevision: released.Revision,
	})
	if err != nil {
		t.Fatalf("prepare routine Audit: %v", err)
	}
	if preparation.Status != assignments.StatusPreparation || preparation.Revision != 1 || preparation.InspectionID != "" {
		t.Fatalf("routine preparation = %+v", preparation)
	}
	assertNoExecutableAudit(t, pool, "")

	assignment, err := assignmentService.AssignLead(context.Background(), manager, assignments.AssignLeadCommand{
		OperationID: "op-routine-lead", IdempotencyKey: "idem-routine-lead",
		AssignmentID: preparation.AssignmentID, InspectionID: "audit-routine",
		ExpectedInspectionRevision: preparation.Revision, LeadSubjectID: "lead-001",
		ScheduledStartDate: "2026-08-12", ScheduledEndDate: "2026-08-13",
	})
	if err != nil {
		t.Fatalf("assign routine Lead Inspector: %v", err)
	}
	lead := principal("lead-001", "caa", "session-lead", identity.RoleLeadInspector)
	if _, err := assignmentService.AssignTeam(context.Background(), manager, assignments.AssignTeamCommand{
		OperationID: "op-routine-team-denied", IdempotencyKey: "idem-routine-team-denied",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		MemberSubjectIDs: []string{"inspector-cabin-001"},
	}); !errors.Is(err, assignments.ErrForbidden) {
		t.Fatalf("Department Manager team-assignment denial error = %v", err)
	}
	if _, err := assignmentService.AssignTeam(context.Background(), lead, assignments.AssignTeamCommand{
		OperationID: "op-routine-team-stale", IdempotencyKey: "idem-routine-team-stale",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision + 1,
		MemberSubjectIDs: []string{"inspector-cabin-001"},
	}); !errors.Is(err, assignments.ErrConflict) {
		t.Fatalf("stale team-assignment revision error = %v", err)
	}
	assignment, err = assignmentService.AssignTeam(context.Background(), lead, assignments.AssignTeamCommand{
		OperationID: "op-routine-team", IdempotencyKey: "idem-routine-team",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		MemberSubjectIDs: []string{"inspector-cabin-001", "inspector-other"},
	})
	if err != nil {
		t.Fatalf("assign routine team: %v", err)
	}
	assignment, err = assignmentService.AssignQuestions(context.Background(), lead, assignments.AssignQuestionsCommand{
		OperationID: "op-routine-questions", IdempotencyKey: "idem-routine-questions",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		QuestionAssignments: []assignments.QuestionAssignment{
			{QuestionID: "q-cabin-crew-training", SubjectID: "inspector-cabin-001"},
			{QuestionID: "q-cabin-crew-training", SubjectID: "inspector-other"},
		},
	})
	if err != nil {
		t.Fatalf("assign routine questions: %v", err)
	}
	workload, err := assignmentService.ListWorkload(context.Background(), manager)
	if err != nil {
		t.Fatalf("list assignment workload: %v", err)
	}
	if workload["inspector-cabin-001"] != 1 || workload["inspector-other"] != 1 {
		t.Fatalf("routine workload = %#v", workload)
	}
	confirmed, err := assignmentService.ConfirmPreparation(context.Background(), manager, assignments.ConfirmPreparationCommand{
		OperationID: "op-routine-confirm", IdempotencyKey: "idem-routine-confirm",
		AssignmentID: assignment.ID, ExpectedAssignmentRevision: assignment.Revision,
	})
	if err != nil {
		t.Fatalf("confirm routine preparation: %v", err)
	}
	if confirmed.Revision != assignment.Revision+1 {
		t.Fatalf("confirmed routine preparation = %+v", confirmed)
	}

	applicationService := testService(pool)
	if _, err := applicationService.MaterializeInspection(
		context.Background(),
		principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector),
		application.MaterializeInspectionCommand{
			OperationID: "op-routine-materialize-denied", CorrelationID: "00000000-0000-4000-8000-000000000001",
			AssignmentID: assignment.ID, ExpectedAssignmentRevision: confirmed.Revision,
			ExpiresAt: canonicalNow.Add(72 * time.Hour),
		},
	); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("Inspector materialization denial error = %v", err)
	}
	materialized, err := applicationService.MaterializeInspection(context.Background(), manager, application.MaterializeInspectionCommand{
		OperationID: "op-routine-materialize", CorrelationID: "00000000-0000-4000-8000-000000000002",
		AssignmentID: assignment.ID, ExpectedAssignmentRevision: confirmed.Revision,
		ExpiresAt: canonicalNow.Add(72 * time.Hour),
	})
	if err != nil {
		t.Fatalf("materialize routine Audit: %v", err)
	}
	replayed, err := applicationService.MaterializeInspection(context.Background(), manager, application.MaterializeInspectionCommand{
		OperationID: "op-routine-materialize", CorrelationID: "00000000-0000-4000-8000-000000000002",
		AssignmentID: assignment.ID, ExpectedAssignmentRevision: confirmed.Revision,
		ExpiresAt: canonicalNow.Add(72 * time.Hour),
	})
	if err != nil || replayed != materialized {
		t.Fatalf("materialization replay = %+v, err = %v", replayed, err)
	}
	if materialized.TemplateVersionID != "" ||
		materialized.Status != assignments.StatusAwaitingAuditeeConfirmation {
		t.Fatalf("materialized routine Audit = %+v", materialized)
	}
	if _, err := applicationService.MaterializeInspection(context.Background(), manager, application.MaterializeInspectionCommand{
		OperationID: "op-routine-materialize-duplicate", CorrelationID: "00000000-0000-4000-8000-000000000003",
		AssignmentID: assignment.ID, ExpectedAssignmentRevision: materialized.AssignmentRevision,
		ExpiresAt: canonicalNow.Add(96 * time.Hour),
	}); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("duplicate materialization error = %v", err)
	}
	assertCanonicalMaterializedSnapshot(t, pool, materialized.PackageID)

	auditee := principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee)
	coordination, err := assignmentService.ListAuditeeCoordination(context.Background(), auditee)
	if err != nil {
		t.Fatalf("list routine Auditee coordination: %v", err)
	}
	if len(coordination) != 1 || coordination[0].InspectionID != materialized.InspectionID ||
		coordination[0].Status != assignments.StatusAwaitingAuditeeConfirmation {
		t.Fatalf("routine Auditee coordination = %+v", coordination)
	}
	if _, err := assignmentService.ListAuditeeCoordination(context.Background(),
		principal("auditee-other", "airline-other", "session-auditee-other", identity.RoleAuditee),
	); err != nil {
		t.Fatalf("list other-organization coordination: %v", err)
	}
}

func TestAdHocPlanningWithholdsAuditeeNoticeAfterMaterialization(t *testing.T) {
	pool := canonicalDatabase(t, "planning_assignment_ad_hoc")
	seedPlanningActors(t, pool)
	seedPlanningDraft(t, pool, "draft-ad-hoc", "airline-xyz")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO surveillance_plan_items (
			id, title, plan_year, organization_id, inspection_type, scheduled_date,
			estimated_budget, status, current_owner_role, next_action, revision
		) VALUES (
			'plan-ad-hoc', 'Ad Hoc / Unannounced — Airline XYZ', 2026, 'airline-xyz',
			'Air Operator Certificate · Cabin Safety', '2026-08-20', 5000,
			'RELEASED', 'manager', 'Department Manager to prepare the scheduled Audit', 5
		)
	`); err != nil {
		t.Fatalf("seed released Ad Hoc planning item: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE planning_intake_drafts
		SET values = $2::jsonb, submitted_planning_item_id = 'plan-ad-hoc', revision = 2
		WHERE id = $1
	`, "draft-ad-hoc", mustJSON(t, adHocIntakeValues())); err != nil {
		t.Fatalf("seed submitted Ad Hoc draft: %v", err)
	}
	adhocSnapshot := `{"planningItemId":"plan-ad-hoc","catalogVersion":"aga-fixture@1.0.0","usageClass":"PREPROD_EXERCISE","noticePolicy":"WITHHELD","selectedQuestionVersionIds":["q-cabin-crew-training"]}`
	adhocDigest := questioncatalog.SelectionDigest([]string{"q-cabin-crew-training"})
	if _, err := pool.Exec(context.Background(), `
		UPDATE canonical_audit_scope_drafts
		SET status = 'RELEASED', notice_policy = 'WITHHELD', selection_digest = $1, selected_question_count = 1
		WHERE id = 'scope-draft-draft-ad-hoc'
	`, adhocDigest); err != nil {
		t.Fatalf("seed released Ad Hoc scope draft: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO canonical_audit_scope_snapshots (
			id, scope_draft_id, revision, stage, catalog_id, usage_class, selection_digest,
			planning_snapshot_digest, selected_question_count, snapshot, created_by_subject_id
		) VALUES (
			'scope-snapshot-ad-hoc-released', 'scope-draft-draft-ad-hoc', 1, 'RELEASED', 'catalog-cabin-fixture',
			'PREPROD_EXERCISE', $2, governed_jsonb_sha256($1::jsonb), 1, $1::jsonb, 'manager-001'
		)
	`, adhocSnapshot, adhocDigest); err != nil {
		t.Fatalf("seed released Ad Hoc snapshot: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO canonical_audit_scope_snapshot_questions (snapshot_id, catalog_id, question_version_id, position)
		VALUES ('scope-snapshot-ad-hoc-released', 'catalog-cabin-fixture', 'q-cabin-crew-training', 0)
	`); err != nil {
		t.Fatalf("seed released Ad Hoc scope questions: %v", err)
	}
	service := assignments.NewService(pool, assignments.Dependencies{
		Clock:       func() time.Time { return canonicalNow },
		IDGenerator: scenarioIDGenerator(),
	})
	manager := principal("manager-001", "caa", "session-manager", identity.RoleDepartmentManager)
	preparation, err := service.Prepare(context.Background(), manager, assignments.PrepareCommand{
		OperationID: "op-ad-hoc-prepare", IdempotencyKey: "idem-ad-hoc-prepare",
		PlanningItemID: "plan-ad-hoc", InspectionID: "audit-ad-hoc", ExpectedPlanningRevision: 5,
	})
	if err != nil {
		t.Fatalf("prepare Ad Hoc Audit: %v", err)
	}
	assignment, err := service.AssignLead(context.Background(), manager, assignments.AssignLeadCommand{
		OperationID: "op-ad-hoc-lead", IdempotencyKey: "idem-ad-hoc-lead",
		AssignmentID: preparation.AssignmentID, InspectionID: preparation.InspectionID,
		ExpectedInspectionRevision: preparation.Revision, LeadSubjectID: "lead-001",
		ScheduledStartDate: "2026-08-20", ScheduledEndDate: "2026-08-20",
	})
	if err != nil {
		t.Fatalf("assign Ad Hoc Lead Inspector: %v", err)
	}
	lead := principal("lead-001", "caa", "session-lead", identity.RoleLeadInspector)
	assignment, err = service.AssignTeam(context.Background(), lead, assignments.AssignTeamCommand{
		OperationID: "op-ad-hoc-team", IdempotencyKey: "idem-ad-hoc-team",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		MemberSubjectIDs: []string{"inspector-cabin-001"},
	})
	if err != nil {
		t.Fatalf("assign Ad Hoc team: %v", err)
	}
	assignment, err = service.AssignQuestions(context.Background(), lead, assignments.AssignQuestionsCommand{
		OperationID: "op-ad-hoc-questions", IdempotencyKey: "idem-ad-hoc-questions",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		QuestionAssignments: []assignments.QuestionAssignment{
			{QuestionID: "q-cabin-crew-training", SubjectID: "inspector-cabin-001"},
		},
	})
	if err != nil {
		t.Fatalf("assign Ad Hoc questions: %v", err)
	}
	confirmed, err := service.ConfirmPreparation(context.Background(), manager, assignments.ConfirmPreparationCommand{
		OperationID: "op-ad-hoc-confirm", IdempotencyKey: "idem-ad-hoc-confirm",
		AssignmentID: assignment.ID, ExpectedAssignmentRevision: assignment.Revision,
	})
	if err != nil {
		t.Fatalf("confirm Ad Hoc preparation: %v", err)
	}
	materialized, err := testService(pool).MaterializeInspection(context.Background(), manager, application.MaterializeInspectionCommand{
		OperationID: "op-ad-hoc-materialize", CorrelationID: "00000000-0000-4000-8000-000000000004",
		AssignmentID: assignment.ID, ExpectedAssignmentRevision: confirmed.Revision,
		ExpiresAt: canonicalNow.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("materialize Ad Hoc Audit: %v", err)
	}
	if materialized.Status != assignments.StatusScheduled || !materialized.NoticeWithheld {
		t.Fatalf("materialized Ad Hoc Audit = %+v", materialized)
	}
	auditee := principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee)
	coordination, err := service.ListAuditeeCoordination(context.Background(), auditee)
	if err != nil {
		t.Fatalf("list Ad Hoc coordination: %v", err)
	}
	if len(coordination) != 0 {
		t.Fatalf("Ad Hoc coordination leaked to Auditee: %+v", coordination)
	}
}

func TestPlanningAssignmentHTTPContractsAndNoticePrivacy(t *testing.T) {
	pool := createTestDatabase(t, "planning_assignment_http")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := testprofile.Reset(context.Background(), pool, canonicalNow); err != nil {
		t.Fatalf("reset canonical HTTP profile: %v", err)
	}
	assignmentService := assignments.NewService(pool, assignments.Dependencies{
		Clock: func() time.Time { return canonicalNow },
	})
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
		Pool: pool, Application: testService(pool), Assignments: assignmentService,
		Clock: func() time.Time { return canonicalNow },
	})
	handler := httpapi.NewCanonicalTestBoundary("task-4-token").Protect(api.Handler())

	getDraft := task4Request(http.MethodGet,
		"/v1/planning/intake-drafts/PLAN-DRAFT-2026-001", "",
		"USR-MANAGER-NORA",
	)
	getDraftResponse := httptest.NewRecorder()
	handler.ServeHTTP(getDraftResponse, getDraft)
	if getDraftResponse.Code != http.StatusOK {
		t.Fatalf("GET planning draft status=%d body=%s",
			getDraftResponse.Code, getDraftResponse.Body.String())
	}
	if body := getDraftResponse.Body.String(); !strings.Contains(body, `"noticePolicy":"ADVANCE"`) ||
		!strings.Contains(body, `"requestedBudget":0`) {
		t.Fatalf("GET planning draft body=%s", body)
	}

	saveDraftBody := `{
		"operationId":"OP-HTTP-PLAN-SAVE",
		"expectedRevision":1,
		"idempotencyKey":"IDEM-HTTP-PLAN-SAVE",
		"draftId":"PLAN-DRAFT-2026-001",
		"values":{
			"organizationId":"ORG-FLY-NAMIBIA",
			"organizationName":"Client cannot change this",
			"applicationType":"Air Operator Certificate",
			"domain":"Cabin Safety",
			"inspectionCategory":"Ad Hoc / Unannounced",
			"noticePolicy":"ADVANCE",
			"purpose":"Immediate risk-triggered surveillance",
			"triggerType":"Risk Trigger",
			"riskCategory":"Cabin Safety",
			"plannedDate":"2026-07-18",
			"mode":"On-site",
			"location":"Windhoek",
			"templateVersionId":"CTV-CABIN-1",
			"scope":"Cabin safety",
			"requestedBudget":0,
			"currency":"NAD"
		}
	}`
	saveDraft := task4Request(http.MethodPut,
		"/v1/planning/intake-drafts/PLAN-DRAFT-2026-001", saveDraftBody,
		"USR-MANAGER-NORA",
	)
	saveDraft.Header.Set("Idempotency-Key", "IDEM-HTTP-PLAN-SAVE")
	saveDraft.Header.Set("If-Match", `"rev-1"`)
	saveDraftResponse := httptest.NewRecorder()
	handler.ServeHTTP(saveDraftResponse, saveDraft)
	if saveDraftResponse.Code != http.StatusOK {
		t.Fatalf("PUT planning draft status=%d body=%s",
			saveDraftResponse.Code, saveDraftResponse.Body.String())
	}
	if body := saveDraftResponse.Body.String(); !strings.Contains(body, `"noticePolicy":"WITHHELD"`) ||
		!strings.Contains(body, `"organizationName":"Fly Namibia"`) {
		t.Fatalf("PUT planning draft did not enforce server truth: %s", body)
	}

	for _, route := range []struct {
		path      string
		subjectID string
		required  string
	}{
		{"/v1/inspection-package-drafts/PKG-AUD-2026-001-CABIN", "USR-MANAGER-NORA", `"packageVersion":1`},
		{"/v1/team-members?role=leadInspector", "USR-MANAGER-NORA", `"subjectId":"USR-LEAD-CANER"`},
		{"/v1/audit-teams?limit=20", "USR-MANAGER-NORA", `"questionId":"CAB-EMEQ-PBE-001"`},
		{"/v1/audit-teams/AUD-2026-001", "USR-MANAGER-NORA", `"leadInspector"`},
		{"/v1/auditee/coordination", "USR-AUDITEE-FLY", `"auditId":"AUD-2026-001"`},
	} {
		request := task4Request(http.MethodGet, route.path, "", route.subjectID)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Errorf("GET %s status=%d body=%s", route.path, response.Code, response.Body.String())
			continue
		}
		if body := response.Body.String(); !strings.Contains(body, route.required) {
			t.Errorf("GET %s body missing %s: %s", route.path, route.required, body)
		}
		if route.path == "/v1/auditee/coordination" {
			for _, forbidden := range []string{
				"ORG-SKYCARGO", "Internal CAA Note", "inspectorWorkload",
				"Ad Hoc / Unannounced", "WITHHELD",
			} {
				if strings.Contains(response.Body.String(), forbidden) {
					t.Errorf("Auditee coordination contains %q: %s",
						forbidden, response.Body.String())
				}
			}
		}
	}

	savePackage := task4Request(
		http.MethodPut,
		"/v1/inspection-package-drafts/PKG-AUD-2026-001-CABIN",
		`{
			"operationId":"OP-HTTP-PACKAGE-SAVE",
			"idempotencyKey":"IDEM-HTTP-PACKAGE-SAVE",
			"expectedRevision":1,
			"packageDraftId":"PKG-AUD-2026-001-CABIN",
			"riskFocus":["PBE serviceability","Cabin inspection CAP follow-up"]
		}`,
		"USR-MANAGER-NORA",
	)
	savePackage.Header.Set("Idempotency-Key", "IDEM-HTTP-PACKAGE-SAVE")
	savePackage.Header.Set("If-Match", `"rev-1"`)
	savePackageResponse := httptest.NewRecorder()
	handler.ServeHTTP(savePackageResponse, savePackage)
	if savePackageResponse.Code != http.StatusOK ||
		!strings.Contains(savePackageResponse.Body.String(), `"revision":2`) {
		t.Fatalf("PUT Inspection Package draft status=%d body=%s",
			savePackageResponse.Code, savePackageResponse.Body.String())
	}

	respondCoordination := task4Request(
		http.MethodPost,
		"/v1/auditee/coordination/AUD-2026-001/responses",
		`{
			"operationId":"OP-HTTP-COORDINATION-CONFIRM",
			"idempotencyKey":"IDEM-HTTP-COORDINATION-CONFIRM",
			"expectedRevision":1,
			"auditId":"AUD-2026-001",
			"organizationId":"ORG-FLY-NAMIBIA",
			"decision":"CONFIRM",
			"alternativeDate":null
		}`,
		"USR-AUDITEE-FLY",
	)
	respondCoordination.Header.Set("Idempotency-Key", "IDEM-HTTP-COORDINATION-CONFIRM")
	respondCoordination.Header.Set("If-Match", `"rev-1"`)
	respondCoordinationResponse := httptest.NewRecorder()
	handler.ServeHTTP(respondCoordinationResponse, respondCoordination)
	if respondCoordinationResponse.Code != http.StatusOK ||
		!strings.Contains(respondCoordinationResponse.Body.String(), `"status":"CONFIRMED"`) ||
		!strings.Contains(respondCoordinationResponse.Body.String(), `"revision":2`) {
		t.Fatalf("POST Auditee coordination response status=%d body=%s",
			respondCoordinationResponse.Code, respondCoordinationResponse.Body.String())
	}
}

func task4Request(method, path, body, subjectID string) *http.Request {
	var payload *bytes.Reader
	if body == "" {
		payload = bytes.NewReader(nil)
	} else {
		payload = bytes.NewReader([]byte(body))
	}
	request := httptest.NewRequest(method, path, payload)
	request.Header.Set(httpapi.CanonicalTestTokenHeader, "task-4-token")
	request.Header.Set(httpapi.CanonicalTestSubjectHeader, subjectID)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func seedPlanningActors(t *testing.T, pool *database.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (subject_id, issuer, display_name)
		VALUES ('finance-001', 'test', 'Finance Reviewer')
	`); err != nil {
		t.Fatalf("seed Finance identity: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO user_profiles (
			subject_id, display_name, organization_id, revision, created_at, updated_at
		) VALUES ('finance-001', 'Finance Reviewer', 'caa', 1, $1, $1)
	`, canonicalNow); err != nil {
		t.Fatalf("seed Finance profile: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO user_settings (
			subject_id, notification_preferences, locale, timezone, revision, updated_at
		) VALUES ('finance-001', '{}'::jsonb, 'en', 'UTC', 1, $1)
	`, canonicalNow); err != nil {
		t.Fatalf("seed Finance settings: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO session_references (
			id, subject_id, organization_id, expires_at, last_seen_at,
			absolute_expires_at, roles
		) VALUES
			('session-finance', 'finance-001', 'caa', $1, $2, $1, ARRAY['finance']),
			('session-inspector-other', 'inspector-other', 'caa', $1, $2, $1, ARRAY['inspector'])
	`, canonicalNow.Add(24*time.Hour), canonicalNow); err != nil {
		t.Fatalf("seed planning actor sessions: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE session_references
		SET expires_at = $1, absolute_expires_at = $1
		WHERE id IN ('session-lead', 'session-inspector', 'session-inspector-other', 'session-manager', 'session-finance', 'session-gm', 'session-executive')
	`, canonicalNow.Add(365*24*time.Hour)); err != nil {
		t.Fatalf("extend Lead Inspector fixture session: %v", err)
	}
	templateSnapshot := map[string]any{
		"schemaVersion":   1,
		"protocolVersion": 1,
		"questions": []map[string]any{
			{
				"id": "q-cabin-crew-training", "sectionId": "CREW",
				"prompt":              "Are cabin crew training records current?",
				"regulatoryReference": "Configured Cabin Crew Training reference",
				"expectedEvidence":    "Current training records",
			},
			{
				"id": "q-cabin-emergency-equipment", "sectionId": "EMERGENCY EQUIPMENT",
				"prompt":              "Is emergency equipment serviceable?",
				"regulatoryReference": "Configured Emergency Equipment reference",
				"expectedEvidence":    "Serviceability records",
			},
		},
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO checklist_template_versions (
			id, template_id, version, title, snapshot, published_at
		) VALUES (
			'template-assignment-v7', 'template-assignment', 7,
			'Assignment Scenario Template', $1, $2
		)
	`, mustJSON(t, templateSnapshot), canonicalNow); err != nil {
		t.Fatalf("seed assignment template: %v", err)
	}
}

func routineIntakeValues(budget float64) planning.IntakeDraftValues {
	return planning.IntakeDraftValues{
		OrganizationID: "airline-xyz", OrganizationName: "Airline XYZ",
		ApplicationType: "Air Operator Certificate", Domain: "Cabin Safety",
		InspectionCategory: planning.InspectionCategoryRoutine,
		NoticePolicy:       planning.NoticePolicyAdvance,
		Purpose:            "Annual routine oversight", TriggerType: "Annual Plan",
		RiskCategory: "Cabin Safety", PlannedDate: "2026-08-12", Mode: "On-site",
		Location: "Windhoek", CatalogVersion: "aga-fixture@1.0.0",
		ScopeDraftID: "", SelectionDigest: "", SelectedQuestionVersionIDs: []string{"q-cabin-crew-training"},
		ProviderScopeID: "scope-airline-xyz-air-operator", RegulatedTargetID: "target-airline-xyz",
		RequestedBudget: budget, Currency: "NAD",
	}
}

func adHocIntakeValues() planning.IntakeDraftValues {
	values := routineIntakeValues(5000)
	values.InspectionCategory = planning.InspectionCategoryAdHoc
	values.NoticePolicy = planning.NoticePolicyWithheld
	values.Purpose = "Immediate risk-triggered surveillance"
	values.TriggerType = "Risk Trigger"
	values.PlannedDate = "2026-08-20"
	return values
}

func decidePlanning(
	t *testing.T,
	service *planning.Service,
	actor identity.Principal,
	item planning.Item,
	operationID string,
	decision planning.Decision,
) planning.Item {
	t.Helper()
	pins, err := service.List(context.Background(), actor, 100)
	if err != nil {
		t.Fatalf("list planning pins before decision %s: %v", decision, err)
	}
	for _, candidate := range pins {
		if candidate.ID == item.ID {
			item.SubmittedScopeSnapshotID = candidate.SubmittedScopeSnapshotID
			item.PlanningSnapshotDigest = candidate.PlanningSnapshotDigest
			break
		}
	}
	output, err := service.Decide(context.Background(), actor, planning.DecideCommand{
		OperationID: operationID, PlanningItemID: item.ID, ExpectedRevision: item.Revision,
		Decision: decision, Reason: "Scenario authority decision.",
		ExpectedSubmittedScopeSnapshotID: item.SubmittedScopeSnapshotID,
		ExpectedPlanningSnapshotDigest:   item.PlanningSnapshotDigest,
	})
	if err != nil {
		t.Fatalf("planning decision %s: %v", decision, err)
	}
	return output
}

func seedPlanningDraft(t *testing.T, pool *database.Pool, draftID, organizationID string) {
	t.Helper()
	values := routineIntakeValues(5000)
	values.ScopeDraftID = "scope-draft-" + draftID
	values.SelectionDigest = questioncatalog.SelectionDigest(values.SelectedQuestionVersionIDs)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO planning_intake_drafts (
			id, organization_id, values, revision, created_by_subject_id,
			created_at, updated_at
		) VALUES ($1, $2, $3::jsonb, 1, 'manager-001', $4, $4)
	`, draftID, organizationID, mustJSON(t, values), canonicalNow); err != nil {
		t.Fatalf("seed Planning intake draft: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO canonical_audit_scope_drafts (
			id, planning_intake_draft_id, organization_id, provider_scope_id, regulated_target_id,
			audit_type, catalog_id, usage_class, revision, status, selected_question_count,
			selection_digest, requested_budget, notice_policy, created_by_subject_id
		) VALUES ($1, $2, $3, 'scope-airline-xyz-air-operator', 'target-airline-xyz', 'CABIN',
			'catalog-cabin-fixture', 'PREPROD_EXERCISE', 1, 'DRAFT', 1, $4, 5000, 'ADVANCE', 'manager-001')
	`, values.ScopeDraftID, draftID, organizationID, values.SelectionDigest); err != nil {
		t.Fatalf("seed canonical planning scope: %v", err)
	}
	selectionOperationID := "selection-" + draftID
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO canonical_audit_scope_selection_operations (
			id, scope_draft_id, operation_id, idempotency_key, operation_kind,
			expected_digest, result_digest, affected_question_version_ids, filter_payload, actor_subject_id
		) VALUES ($1, $2, $1, $1, 'REPLACE', '', $3, '["q-cabin-crew-training"]'::jsonb, '{}'::jsonb, 'manager-001')
	`, selectionOperationID, values.ScopeDraftID, values.SelectionDigest); err != nil {
		t.Fatalf("seed canonical question selection: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO canonical_audit_scope_selection_questions (
			operation_id, catalog_id, question_version_id, position, selection_digest
		) VALUES ($1, 'catalog-cabin-fixture', 'q-cabin-crew-training', 0, $2)
	`, selectionOperationID, values.SelectionDigest); err != nil {
		t.Fatalf("seed canonical selected question: %v", err)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode test JSON: %v", err)
	}
	return encoded
}

func assertNoExecutableAudit(t *testing.T, pool *database.Pool, inspectionID string) {
	t.Helper()
	if inspectionID == "" {
		var count int
		if err := pool.QueryRow(context.Background(), `
			SELECT COUNT(*) FROM audit_assignments
			WHERE planning_item_id = 'plan-routine' AND inspection_id IS NULL AND status = 'PREPARATION'
		`).Scan(&count); err != nil {
			t.Fatalf("read pre-materialization assignment: %v", err)
		}
		if count != 1 {
			t.Fatalf("pre-materialization assignment count=%d", count)
		}
		return
	}
	var status string
	var packageCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT inspection.status, COUNT(package.id)
		FROM inspections inspection
		LEFT JOIN inspection_packages package ON package.inspection_id = inspection.id
		WHERE inspection.id = $1
		GROUP BY inspection.status
	`, inspectionID).Scan(&status, &packageCount); err != nil {
		t.Fatalf("read pre-materialization Audit: %v", err)
	}
	if status != string(assignments.StatusPreparation) || packageCount != 0 {
		t.Fatalf("pre-materialization Audit status=%q packages=%d", status, packageCount)
	}
}

func assertCanonicalMaterializedSnapshot(t *testing.T, pool *database.Pool, packageID string) {
	t.Helper()
	var templateID *string
	var snapshot []byte
	var scopeID *string
	if err := pool.QueryRow(context.Background(), `
		SELECT checklist_template_version_id, canonical_scope_snapshot_id, snapshot
		FROM inspection_packages WHERE id = $1
	`, packageID).Scan(&templateID, &scopeID, &snapshot); err != nil {
		t.Fatalf("read canonical materialized package: %v", err)
	}
	if templateID != nil || scopeID == nil || *scopeID == "" || !json.Valid(snapshot) {
		t.Fatalf("canonical package template=%v scope=%v snapshot=%s", templateID, scopeID, snapshot)
	}
}

func assertMaterializedSnapshot(t *testing.T, pool *database.Pool, packageID, templateVersionID string) {
	t.Helper()
	var storedTemplateVersionID string
	var snapshot []byte
	var digest string
	if err := pool.QueryRow(context.Background(), `
		SELECT checklist_template_version_id, snapshot, package_digest
		FROM inspection_packages
		WHERE id = $1
	`, packageID).Scan(&storedTemplateVersionID, &snapshot, &digest); err != nil {
		t.Fatalf("read materialized package: %v", err)
	}
	if storedTemplateVersionID != templateVersionID ||
		!json.Valid(snapshot) || len(digest) < len("sha256:")+1 {
		t.Fatalf("materialized package template=%q snapshot=%s digest=%q",
			storedTemplateVersionID, snapshot, digest)
	}
}

func assertCommandTransactionLink(
	t *testing.T,
	pool *database.Pool,
	operationID string,
	idempotencyScope string,
) {
	t.Helper()
	var auditAction, changeOperationID, outboxOperationID string
	if err := pool.QueryRow(context.Background(), `
		SELECT audit.action, change.operation_id, outbox.operation_id
		FROM command_transaction_links link
		JOIN audit_events audit ON audit.event_id = link.audit_event_id
		JOIN authorized_sync_changes change ON change.sequence_id = link.change_sequence_id
		JOIN outbox_messages outbox ON outbox.id = link.outbox_message_id
		WHERE link.operation_id = $1 AND link.idempotency_scope = $2
	`, operationID, idempotencyScope).Scan(
		&auditAction, &changeOperationID, &outboxOperationID,
	); err != nil {
		t.Fatalf("command transaction link %s: %v", operationID, err)
	}
	if auditAction != "PLANNING_RETURNED_FOR_REVISION" ||
		changeOperationID != operationID || outboxOperationID != operationID {
		t.Fatalf(
			"command transaction link %s action=%q changeOperation=%q outboxOperation=%q",
			operationID, auditAction, changeOperationID, outboxOperationID,
		)
	}
}
