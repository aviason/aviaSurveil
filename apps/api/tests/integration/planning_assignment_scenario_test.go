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

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/assignments"
	"github.com/aviason/aviaSurveil/internal/httpapi"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/planning"
	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/questioncatalog"
	"github.com/aviason/aviaSurveil/internal/testprofile"
	"github.com/aviason/aviaSurveil/migrations"
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
	var usageClasses struct {
		Catalog   string
		Scope     string
		Submitted string
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT catalog.usage_class, scope.usage_class, submitted.usage_class
		FROM planning_intake_drafts draft
		JOIN canonical_audit_scope_drafts scope ON scope.planning_intake_draft_id = draft.id
		JOIN canonical_question_catalogs catalog ON catalog.id = scope.catalog_id
		JOIN canonical_audit_scope_snapshots submitted
		  ON submitted.scope_draft_id = scope.id AND submitted.stage = 'SUBMITTED'
		WHERE draft.submitted_planning_item_id = 'plan-routine'
		ORDER BY submitted.revision DESC
		LIMIT 1
	`).Scan(&usageClasses.Catalog, &usageClasses.Scope, &usageClasses.Submitted); err != nil {
		t.Fatalf("inspect routine canonical usage classes: %v", err)
	}
	if usageClasses.Catalog != "GOVERNED_OPERATIONAL" ||
		usageClasses.Scope != "GOVERNED_OPERATIONAL" ||
		usageClasses.Submitted != "GOVERNED_OPERATIONAL" {
		t.Fatalf("routine canonical usage classes = %+v", usageClasses)
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
	if _, err := pool.Exec(context.Background(), `
		UPDATE session_references
		SET revoked_at = $1
		WHERE id = 'session-lead'
	`, canonicalNow); err != nil {
		t.Fatalf("revoke Lead browser session before assignment: %v", err)
	}
	leadRole := identity.RoleLeadInspector
	leadMembers, err := assignmentService.ListTeamMembers(context.Background(), manager, &leadRole, 100)
	if err != nil {
		t.Fatalf("list prepared Lead roster without an active browser session: %v", err)
	}
	if len(leadMembers) != 1 || leadMembers[0].SubjectID != "lead-001" || leadMembers[0].Role != identity.RoleLeadInspector {
		t.Fatalf("prepared Lead roster = %+v", leadMembers)
	}
	inspectorRole := identity.RoleInspector
	inspectorMembers, err := assignmentService.ListTeamMembers(context.Background(), manager, &inspectorRole, 100)
	if err != nil {
		t.Fatalf("list prepared Inspector roster: %v", err)
	}
	if len(inspectorMembers) != 2 {
		t.Fatalf("prepared Inspector roster = %+v", inspectorMembers)
	}

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
	teamPreview, err := assignmentService.PreviewTeam(context.Background(), lead, assignments.PreviewTeamCommand{
		OperationID: "op-routine-team-preview", IdempotencyKey: "idem-routine-team-preview",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		MemberSubjectIDs: []string{"inspector-cabin-001", "inspector-other"},
	})
	if err != nil {
		t.Fatalf("preview routine team: %v", err)
	}
	assignment, err = assignmentService.AssignTeam(context.Background(), lead, assignments.AssignTeamCommand{
		OperationID: "op-routine-team", IdempotencyKey: "idem-routine-team",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		PreviewID: teamPreview.PreviewID, PreviewDigest: teamPreview.Digest,
		MemberSubjectIDs: []string{"inspector-cabin-001", "inspector-other"},
	})
	if err != nil {
		t.Fatalf("assign routine team: %v", err)
	}
	if len(assignment.SelectedQuestionVersionIDs) != 1 ||
		assignment.SelectedQuestionVersionIDs[0] != "q-cabin-crew-training" {
		t.Fatalf("team assignment dropped released question identities: %+v", assignment.SelectedQuestionVersionIDs)
	}
	questionAssignments := []assignments.QuestionAssignment{
		{QuestionID: "q-cabin-crew-training", SubjectID: "inspector-cabin-001"},
		{QuestionID: "q-cabin-crew-training", SubjectID: "inspector-other"},
	}
	coveragePreview, err := assignmentService.PreviewQuestions(context.Background(), lead, assignments.PreviewQuestionsCommand{
		OperationID: "op-routine-questions-preview", IdempotencyKey: "idem-routine-questions-preview",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		OperationKind: assignments.QuestionCoverageReplace, QuestionAssignments: questionAssignments,
	})
	if err != nil {
		t.Fatalf("preview routine questions: %v", err)
	}
	assignment, err = assignmentService.AssignQuestions(context.Background(), lead, assignments.AssignQuestionsCommand{
		OperationID: "op-routine-questions", IdempotencyKey: "idem-routine-questions",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		PreviewID: coveragePreview.PreviewID, PreviewDigest: coveragePreview.Digest,
		OperationKind:       assignments.QuestionCoverageReplace,
		QuestionAssignments: questionAssignments,
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
	teamCorrectionPreview, err := assignmentService.PreviewTeam(context.Background(), lead, assignments.PreviewTeamCommand{
		OperationID: "op-routine-team-correction-preview", IdempotencyKey: "idem-routine-team-correction-preview",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		MemberSubjectIDs: []string{"inspector-cabin-001"},
	})
	if err != nil {
		t.Fatalf("preview routine team correction: %v", err)
	}
	assignment, err = assignmentService.AssignTeam(context.Background(), lead, assignments.AssignTeamCommand{
		OperationID: "op-routine-team-correction", IdempotencyKey: "idem-routine-team-correction",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		PreviewID: teamCorrectionPreview.PreviewID, PreviewDigest: teamCorrectionPreview.Digest,
		MemberSubjectIDs: []string{"inspector-cabin-001"},
	})
	if err != nil {
		t.Fatalf("remove routine team member: %v", err)
	}
	activeCoverage := []assignments.QuestionAssignment{{QuestionID: "q-cabin-crew-training", SubjectID: "inspector-cabin-001"}}
	if len(assignment.QuestionAssignments) != 1 || assignment.QuestionAssignments[0] != activeCoverage[0] {
		t.Fatalf("team correction response coverage = %+v, want %+v", assignment.QuestionAssignments, activeCoverage)
	}
	var removedCoverageCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM audit_question_assignments
		WHERE assignment_id=$1 AND subject_id='inspector-other'
	`, assignment.ID).Scan(&removedCoverageCount); err != nil {
		t.Fatalf("count removed Inspector coverage: %v", err)
	}
	if removedCoverageCount != 0 {
		t.Fatalf("removed Inspector retained %d question coverage rows", removedCoverageCount)
	}
	activeCoveragePreview, err := assignmentService.PreviewQuestions(context.Background(), lead, assignments.PreviewQuestionsCommand{
		OperationID: "op-routine-active-coverage-preview", IdempotencyKey: "idem-routine-active-coverage-preview",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		OperationKind: assignments.QuestionCoverageAdd, QuestionAssignments: activeCoverage,
	})
	if err != nil {
		t.Fatalf("preview active routine coverage: %v", err)
	}
	assignment, err = assignmentService.AssignQuestions(context.Background(), lead, assignments.AssignQuestionsCommand{
		OperationID: "op-routine-active-coverage", IdempotencyKey: "idem-routine-active-coverage",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		PreviewID: activeCoveragePreview.PreviewID, PreviewDigest: activeCoveragePreview.Digest,
		OperationKind: assignments.QuestionCoverageAdd, QuestionAssignments: activeCoverage,
	})
	if err != nil {
		t.Fatalf("restore routine questions-assigned state: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO audit_question_assignments (assignment_id, question_id, subject_id, revision, created_at)
		VALUES ($1, 'q-cabin-crew-training', 'inspector-other', 1, $2)
	`, assignment.ID, canonicalNow); err != nil {
		t.Fatalf("seed stale removed-Inspector coverage: %v", err)
	}
	if _, err := assignmentService.ConfirmPreparation(context.Background(), manager, assignments.ConfirmPreparationCommand{
		OperationID: "op-routine-confirm-stale-member", IdempotencyKey: "idem-routine-confirm-stale-member",
		AssignmentID: assignment.ID, ExpectedAssignmentRevision: assignment.Revision,
	}); !errors.Is(err, assignments.ErrConflict) {
		t.Fatalf("stale removed-Inspector confirmation error = %v", err)
	}
	staleCoverage := []assignments.QuestionAssignment{{QuestionID: "q-cabin-crew-training", SubjectID: "inspector-other"}}
	staleRemovalPreview, err := assignmentService.PreviewQuestions(context.Background(), lead, assignments.PreviewQuestionsCommand{
		OperationID: "op-routine-stale-removal-preview", IdempotencyKey: "idem-routine-stale-removal-preview",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		OperationKind: assignments.QuestionCoverageRemove, QuestionAssignments: staleCoverage,
	})
	if err != nil {
		t.Fatalf("preview stale removed-Inspector coverage cleanup: %v", err)
	}
	assignment, err = assignmentService.AssignQuestions(context.Background(), lead, assignments.AssignQuestionsCommand{
		OperationID: "op-routine-stale-removal", IdempotencyKey: "idem-routine-stale-removal",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		PreviewID: staleRemovalPreview.PreviewID, PreviewDigest: staleRemovalPreview.Digest,
		OperationKind: assignments.QuestionCoverageRemove, QuestionAssignments: staleCoverage,
	})
	if err != nil {
		t.Fatalf("remove stale removed-Inspector coverage: %v", err)
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
	seedLegacyPreparationPredecessor(t, pool, confirmed.PreparationID)

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
	confirmedCoordination, err := assignmentService.RespondAuditeeCoordination(
		context.Background(),
		auditee,
		assignments.RespondCoordinationCommand{
			OperationID: "op-routine-coordination-confirm", IdempotencyKey: "idem-routine-coordination-confirm",
			InspectionID: materialized.InspectionID, OrganizationID: "airline-xyz",
			ExpectedRevision: materialized.AssignmentRevision, Decision: assignments.CoordinationConfirm,
		},
	)
	if err != nil || confirmedCoordination.Status != assignments.StatusConfirmed ||
		confirmedCoordination.Revision != materialized.AssignmentRevision+1 {
		t.Fatalf("confirm routine Auditee coordination = %+v, err = %v", confirmedCoordination, err)
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
	adhocSnapshot := `{"planningItemId":"plan-ad-hoc","catalogVersion":"aga-fixture@1.0.0","usageClass":"GOVERNED_OPERATIONAL","noticePolicy":"WITHHELD","selectedQuestionVersionIds":["q-cabin-crew-training"]}`
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
			'GOVERNED_OPERATIONAL', $2, governed_jsonb_sha256($1::jsonb), 1, $1::jsonb, 'manager-001'
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
	teamPreview, err := service.PreviewTeam(context.Background(), lead, assignments.PreviewTeamCommand{
		OperationID: "op-ad-hoc-team-preview", IdempotencyKey: "idem-ad-hoc-team-preview",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		MemberSubjectIDs: []string{"inspector-cabin-001"},
	})
	if err != nil {
		t.Fatalf("preview Ad Hoc team: %v", err)
	}
	assignment, err = service.AssignTeam(context.Background(), lead, assignments.AssignTeamCommand{
		OperationID: "op-ad-hoc-team", IdempotencyKey: "idem-ad-hoc-team",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		PreviewID: teamPreview.PreviewID, PreviewDigest: teamPreview.Digest,
		MemberSubjectIDs: []string{"inspector-cabin-001"},
	})
	if err != nil {
		t.Fatalf("assign Ad Hoc team: %v", err)
	}
	questionAssignments := []assignments.QuestionAssignment{{QuestionID: "q-cabin-crew-training", SubjectID: "inspector-cabin-001"}}
	coveragePreview, err := service.PreviewQuestions(context.Background(), lead, assignments.PreviewQuestionsCommand{
		OperationID: "op-ad-hoc-questions-preview", IdempotencyKey: "idem-ad-hoc-questions-preview",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		OperationKind: assignments.QuestionCoverageReplace, QuestionAssignments: questionAssignments,
	})
	if err != nil {
		t.Fatalf("preview Ad Hoc questions: %v", err)
	}
	assignment, err = service.AssignQuestions(context.Background(), lead, assignments.AssignQuestionsCommand{
		OperationID: "op-ad-hoc-questions", IdempotencyKey: "idem-ad-hoc-questions",
		AssignmentID: assignment.ID, ExpectedRevision: assignment.Revision,
		PreviewID: coveragePreview.PreviewID, PreviewDigest: coveragePreview.Digest,
		OperationKind:       assignments.QuestionCoverageReplace,
		QuestionAssignments: questionAssignments,
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

func TestCanonicalPreparationCoveragePinsQuestionPositionToReleasedSnapshot(t *testing.T) {
	pool := canonicalDatabase(t, "preparation_position")
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `
		INSERT INTO question_versions (
			id, question_id, version, prompt, configured_reference, expected_evidence, created_by_subject_id
		) VALUES (
			'q-preparation-position-other', 'q-preparation-position-other', 1,
			'Synthetic position-bound question.', 'Fixture reference', 'Fixture evidence', 'manager-001'
		)
	`); err != nil {
		t.Fatalf("seed second position-bound question: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO canonical_question_version_provenance (question_version_id, usage_class, catalog_id)
		VALUES ('q-preparation-position-other', 'GOVERNED_OPERATIONAL', 'catalog-cabin-fixture')
	`); err != nil {
		t.Fatalf("seed second question provenance: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO canonical_question_catalog_memberships (
			catalog_id, question_version_id, usage_class, form_code, proposal_id, ordinal,
			question_digest, source_gap_state
		) VALUES (
			'catalog-cabin-fixture', 'q-preparation-position-other', 'GOVERNED_OPERATIONAL',
			'CABIN', 'proposal-preparation-position-other', 2, 'sha256:position-other', 'NONE'
		)
	`); err != nil {
		t.Fatalf("seed second catalog membership: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO planning_intake_drafts (id, organization_id, values, created_by_subject_id)
		VALUES ('draft-preparation-position', 'airline-xyz', '{"fixture":"position"}'::jsonb, 'manager-001')
	`); err != nil {
		t.Fatalf("seed position planning draft: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO canonical_audit_scope_drafts (
			id, planning_intake_draft_id, organization_id, provider_scope_id, regulated_target_id,
			audit_type, catalog_id, usage_class, revision, status, selected_question_count,
			selection_digest, notice_policy, created_by_subject_id
		) VALUES (
			'scope-draft-preparation-position', 'draft-preparation-position', 'airline-xyz',
			'scope-airline-xyz-air-operator', 'target-airline-xyz', 'CABIN',
			'catalog-cabin-fixture', 'GOVERNED_OPERATIONAL', 1, 'RELEASED', 2,
			'sha256:preparation-position-selection', 'ADVANCE', 'manager-001'
		)
	`); err != nil {
		t.Fatalf("seed position released scope: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO canonical_audit_scope_snapshots (
			id, scope_draft_id, revision, stage, catalog_id, usage_class, selection_digest,
			planning_snapshot_digest, selected_question_count, snapshot, created_by_subject_id
		) VALUES (
			'scope-snapshot-preparation-position', 'scope-draft-preparation-position', 1,
			'RELEASED', 'catalog-cabin-fixture', 'GOVERNED_OPERATIONAL',
			'sha256:preparation-position-selection', '', 2, '{}'::jsonb, 'manager-001'
		)
	`); err != nil {
		t.Fatalf("seed position released snapshot: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO canonical_audit_scope_snapshot_questions (
			snapshot_id, catalog_id, question_version_id, position
		) VALUES
			('scope-snapshot-preparation-position', 'catalog-cabin-fixture', 'q-cabin-crew-training', 0),
			('scope-snapshot-preparation-position', 'catalog-cabin-fixture', 'q-preparation-position-other', 1)
	`); err != nil {
		t.Fatalf("seed position released questions: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO canonical_audit_preparation_snapshots (
			id, assignment_id, released_scope_snapshot_id, lead_subject_id, revision, status,
			preparation_digest, snapshot
		) VALUES (
			'preparation-position', 'assignment-cabin-001', 'scope-snapshot-preparation-position',
			'lead-001', 1, 'DRAFT', 'sha256:preparation-position', '{}'::jsonb
		)
	`); err != nil {
		t.Fatalf("seed position preparation: %v", err)
	}

	for _, subjectID := range []string{"inspector-cabin-001", "inspector-other"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO canonical_audit_preparation_questions (
				preparation_id, released_scope_snapshot_id, question_version_id, subject_id, position
			) VALUES (
				'preparation-position', 'scope-snapshot-preparation-position',
				'q-cabin-crew-training', $1, 0
			)
		`, subjectID); err != nil {
			t.Fatalf("assign multiple Inspectors to one released question: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO canonical_audit_preparation_questions (
			preparation_id, released_scope_snapshot_id, question_version_id, subject_id, position
		) VALUES (
			'preparation-position', 'scope-snapshot-preparation-position',
			'q-preparation-position-other', 'lead-001', 0
		)
	`); err == nil {
		t.Fatal("coverage with another question at the released position unexpectedly succeeded")
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
	seedCanonicalPlanningHTTPDraft(t, pool)
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
			"applicationType":"CABIN",
			"domain":"Cabin Safety",
			"inspectionCategory":"Ad Hoc / Unannounced",
			"noticePolicy":"ADVANCE",
			"purpose":"Immediate risk-triggered surveillance",
			"triggerType":"Risk Trigger",
			"riskCategory":"Cabin Safety",
			"plannedDate":"2026-07-18",
			"mode":"On-site",
			"location":"Windhoek",
			"templateVersionId":"",
			"scope":"",
			"catalogVersion":"task4-planning@1.0.0",
			"scopeDraftId":"SCOPE-DRAFT-TASK4-PLAN",
			"selectionDigest":"8bf3518c051416c444a9b441fe44a67f9e17fd1c54723a2ef5cf91e1a67833e0",
			"selectedQuestionVersionIds":["QV-CAB-EMEQ-PBE-001-V1"],
			"providerScopeId":"SCOPE-OPS-AOC-SOURCE-BOUND",
			"regulatedTargetId":"TARGET-OPS-AOC-SOURCE-BOUND",
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
		{"/v1/team-members?role=leadInspector", "USR-MANAGER-NORA", `"subjectId":"USR-LEAD-CANER"`},
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

	for _, legacy := range []struct {
		path       string
		statusCode int
		required   string
	}{
		{"/v1/inspection-package-drafts/PKG-AUD-2026-001-CABIN", http.StatusNotFound, ""},
		{"/v1/audit-teams?limit=20", http.StatusOK, `"items":[]`},
		{"/v1/audit-teams/AUD-2026-001", http.StatusNotFound, `"code":"NOT_FOUND"`},
	} {
		request := task4Request(http.MethodGet, legacy.path, "", "USR-MANAGER-NORA")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != legacy.statusCode ||
			(legacy.required != "" && !strings.Contains(response.Body.String(), legacy.required)) {
			t.Fatalf("disabled legacy Planning path %s status=%d body=%s",
				legacy.path, response.Code, response.Body.String())
		}
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
	if respondCoordinationResponse.Code != http.StatusConflict ||
		!strings.Contains(respondCoordinationResponse.Body.String(), `"code":"CONFLICT"`) {
		t.Fatalf("POST Auditee coordination response status=%d body=%s",
			respondCoordinationResponse.Code, respondCoordinationResponse.Body.String())
	}
}

func seedCanonicalPlanningHTTPDraft(t *testing.T, pool *database.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO canonical_question_catalogs (
			id, catalog_version, usage_class, profile_name, profile_version,
			status, source_package_version, source_package_json_sha256,
			source_package_zip_sha256, root_digest, catalog_root_digest, question_count, form_count,
			created_by_subject_id
		) VALUES (
			'CAT-TASK4-PLAN', 'task4-planning@1.0.0', 'GOVERNED_OPERATIONAL',
			'aga-preprod', '1.0.0', 'SEALED', '1.0.0',
			'sha256:task4-planning-json', 'sha256:task4-planning-zip',
			'sha256:task4-planning-root', 'sha256:task4-planning-root', 1, 1, 'USR-MANAGER-NORA'
		);

		INSERT INTO canonical_question_catalog_forms (
			catalog_id, form_code, form_digest, archive_digest,
			question_count, source_gap_state
		) VALUES (
			'CAT-TASK4-PLAN', 'CABIN', 'sha256:task4-form',
			'sha256:task4-archive', 1, 'SOURCE_MAPPING_REQUIRED'
		);

		INSERT INTO canonical_question_version_provenance (
			question_version_id, usage_class, catalog_id
		) VALUES (
			'QV-CAB-EMEQ-PBE-001-V1', 'GOVERNED_OPERATIONAL', 'CAT-TASK4-PLAN'
		);

		INSERT INTO canonical_question_catalog_memberships (
			catalog_id, question_version_id, usage_class, form_code,
			proposal_id, ordinal, question_digest, source_locator,
			source_gap_state, proposed_domain, proposed_topic, proposed_risk_band
		) VALUES (
			'CAT-TASK4-PLAN', 'QV-CAB-EMEQ-PBE-001-V1', 'GOVERNED_OPERATIONAL',
			'CABIN', 'PBE-001', 1, 'sha256:task4-pbe-question',
			'fixture://task4/pbe', 'SOURCE_MAPPING_REQUIRED',
			'Cabin Safety', 'PBE', 'MEDIUM'
		);

		INSERT INTO canonical_question_catalog_applicabilities (
			catalog_id, question_version_id, provider_scope_id,
			regulated_target_id, status, reason, actor_subject_id
		) VALUES (
			'CAT-TASK4-PLAN', 'QV-CAB-EMEQ-PBE-001-V1',
			'SCOPE-OPS-AOC-SOURCE-BOUND', 'TARGET-OPS-AOC-SOURCE-BOUND', 'ELIGIBLE',
			'Exact task-owned Planning fixture eligibility.', 'USR-MANAGER-NORA'
		);

		UPDATE planning_intake_drafts
		SET values = values || '{
			"applicationType":"CABIN",
			"templateVersionId":"",
			"scope":"",
			"catalogVersion":"task4-planning@1.0.0",
			"scopeDraftId":"SCOPE-DRAFT-TASK4-PLAN",
			"selectionDigest":"8bf3518c051416c444a9b441fe44a67f9e17fd1c54723a2ef5cf91e1a67833e0",
			"selectedQuestionVersionIds":["QV-CAB-EMEQ-PBE-001-V1"],
			"providerScopeId":"SCOPE-OPS-AOC-SOURCE-BOUND",
			"regulatedTargetId":"TARGET-OPS-AOC-SOURCE-BOUND"
		}'::jsonb
		WHERE id = 'PLAN-DRAFT-2026-001';

		INSERT INTO canonical_audit_scope_drafts (
			id, planning_intake_draft_id, organization_id, provider_scope_id,
			regulated_target_id, audit_type, catalog_id, usage_class, revision,
			status, selected_question_count, selection_digest, requested_budget,
			notice_policy, created_by_subject_id
		) VALUES (
			'SCOPE-DRAFT-TASK4-PLAN', 'PLAN-DRAFT-2026-001',
			'ORG-FLY-NAMIBIA', 'SCOPE-OPS-AOC-SOURCE-BOUND',
			'TARGET-OPS-AOC-SOURCE-BOUND', 'CABIN',
			'CAT-TASK4-PLAN', 'GOVERNED_OPERATIONAL', 1, 'DRAFT', 1,
			'8bf3518c051416c444a9b441fe44a67f9e17fd1c54723a2ef5cf91e1a67833e0',
			0, 'ADVANCE', 'USR-MANAGER-NORA'
		);

		INSERT INTO canonical_audit_scope_selection_operations (
			id, scope_draft_id, operation_id, idempotency_key, operation_kind,
			expected_digest, result_digest, affected_question_version_ids,
			filter_payload, actor_subject_id
		) VALUES (
			'SELECTION-TASK4-PLAN', 'SCOPE-DRAFT-TASK4-PLAN',
			'OP-SELECTION-TASK4-PLAN', 'IDEM-SELECTION-TASK4-PLAN', 'ADD', '',
			'8bf3518c051416c444a9b441fe44a67f9e17fd1c54723a2ef5cf91e1a67833e0',
			'["QV-CAB-EMEQ-PBE-001-V1"]', '{}', 'USR-MANAGER-NORA'
		);

		INSERT INTO canonical_audit_scope_selection_questions (
			operation_id, catalog_id, question_version_id, position, selection_digest
		) VALUES (
			'SELECTION-TASK4-PLAN', 'CAT-TASK4-PLAN',
			'QV-CAB-EMEQ-PBE-001-V1', 0,
			'8bf3518c051416c444a9b441fe44a67f9e17fd1c54723a2ef5cf91e1a67833e0'
		);

		INSERT INTO canonical_audit_scope_draft_questions (
			scope_draft_id, revision, catalog_id, question_version_id,
			position, selection_digest
		) VALUES (
			'SCOPE-DRAFT-TASK4-PLAN', 1, 'CAT-TASK4-PLAN',
			'QV-CAB-EMEQ-PBE-001-V1', 0,
			'8bf3518c051416c444a9b441fe44a67f9e17fd1c54723a2ef5cf91e1a67833e0'
		)
	`); err != nil {
		t.Fatalf("seed canonical Planning HTTP draft: %v", err)
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
		ApplicationType: "CABIN", Domain: "Cabin Safety",
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
			audit_type, catalog_id, usage_class, catalog_root_digest, revision, status, selected_question_count,
			selection_digest, requested_budget, notice_policy, created_by_subject_id
		) VALUES ($1, $2, $3, 'scope-airline-xyz-air-operator', 'target-airline-xyz', 'CABIN',
			'catalog-cabin-fixture', 'GOVERNED_OPERATIONAL', 'sha256:fixture-catalog-root', 1, 'DRAFT', 1, $4, 5000, 'ADVANCE', 'manager-001')
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

// seedLegacyPreparationPredecessor models the immutable pre-38 CONFIRMED
// receipt that migration 38 retains alongside its revision-bound successor.
// The current schema correctly rejects new unbound confirmations, so this
// isolated test database removes that validation only while installing the
// historical predecessor; the materializer must ignore it rather than count
// both rows as current confirmation candidates.
func seedLegacyPreparationPredecessor(t *testing.T, pool *database.Pool, successorID string) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		ALTER TABLE canonical_audit_preparation_snapshots
		DROP CONSTRAINT canonical_audit_preparation_confirmed_revision_shape
	`); err != nil {
		t.Fatalf("allow retained legacy confirmation fixture: %v", err)
	}
	legacyID := successorID + ":legacy-predecessor"
	result, err := pool.Exec(ctx, `
		INSERT INTO canonical_audit_preparation_snapshots (
			id, assignment_id, released_scope_snapshot_id, lead_subject_id, revision, status,
			preparation_digest, confirmed_by_subject_id, confirmed_at,
			confirmed_assignment_revision, snapshot, created_at
		)
		SELECT $1, assignment_id, released_scope_snapshot_id, lead_subject_id, revision + 2,
		       'CONFIRMED', preparation_digest, confirmed_by_subject_id, confirmed_at,
		       NULL, snapshot, $2
		FROM canonical_audit_preparation_snapshots
		WHERE id = $3
	`, legacyID, canonicalNow, successorID)
	if err != nil {
		t.Fatalf("seed retained legacy confirmation predecessor: %v", err)
	}
	if result.RowsAffected() != 1 {
		t.Fatalf("seed retained legacy confirmation rows=%d", result.RowsAffected())
	}
	var confirmationCount, currentBoundCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE confirmed_assignment_revision IS NOT NULL)
		FROM canonical_audit_preparation_snapshots
		WHERE assignment_id = (
			SELECT assignment_id FROM canonical_audit_preparation_snapshots WHERE id = $1
		)
		  AND released_scope_snapshot_id = (
			SELECT released_scope_snapshot_id FROM canonical_audit_preparation_snapshots WHERE id = $1
		)
		  AND status = 'CONFIRMED'
	`, successorID).Scan(&confirmationCount, &currentBoundCount); err != nil {
		t.Fatalf("read legacy/current confirmation pair: %v", err)
	}
	if confirmationCount != 2 || currentBoundCount != 1 {
		t.Fatalf("legacy/current confirmation pair count=%d currentBound=%d", confirmationCount, currentBoundCount)
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
