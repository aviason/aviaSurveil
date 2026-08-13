package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/checklists"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/potentialfindings"
)

func TestChecklistMutationSubmitReadOnlyAndAuthorizedReopenUseExactRevisions(t *testing.T) {
	pool := canonicalDatabase(t, "checklist_transitions")
	service := testService(pool)
	inspector := principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector)
	// The shared canonical fixture includes an issued Preliminary Report for
	// lifecycle tests. This focused checklist transition test exercises the
	// pre-report reopen boundary, so remove that unrelated history explicitly.
	if _, err := pool.Exec(context.Background(), `
		ALTER TABLE report_versions DISABLE TRIGGER report_versions_immutable;
		DELETE FROM report_approval_states
		WHERE report_version_id IN (SELECT id FROM report_versions WHERE inspection_id = 'audit-cabin-001');
		DELETE FROM report_versions WHERE inspection_id = 'audit-cabin-001';
		ALTER TABLE report_versions ENABLE TRIGGER report_versions_immutable;
	`); err != nil {
		t.Fatalf("remove report history for checklist transition test: %v", err)
	}

	edited, err := service.UpsertChecklistResponse(context.Background(), inspector, application.UpsertChecklistResponseCommand{
		OperationID: "op-response-edit-001", CorrelationID: "corr-checklist", ResponseID: "response-cabin-001",
		InspectionID: "audit-cabin-001", PackageID: "package-cabin-001", QuestionID: "q-cabin-crew-training",
		ExpectedResponseRevision: int64Pointer(1), Answer: "OBSERVATION",
		CommentToAuditee: "Training roster requires clarification.", InternalCAANote: "Internal CAA follow-up.",
	})
	if err != nil || edited.Revision != 2 || edited.Answer != "OBSERVATION" {
		t.Fatalf("edit response = %+v, err = %v", edited, err)
	}
	otherInspector := principal("inspector-other", "caa", "session-inspector", identity.RoleInspector)
	if _, err := service.UpsertChecklistResponse(context.Background(), otherInspector, application.UpsertChecklistResponseCommand{
		OperationID: "op-response-forbidden", CorrelationID: "corr-checklist", ResponseID: "response-cabin-001",
		InspectionID: "audit-cabin-001", PackageID: "package-cabin-001", QuestionID: "q-cabin-crew-training",
		ExpectedResponseRevision: int64Pointer(2), Answer: "COMPLIANT", CommentToAuditee: "not authorized",
	}); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("unassigned response edit error = %v", err)
	}

	submitted, err := service.SubmitChecklist(context.Background(), inspector, application.SubmitChecklistCommand{
		OperationID: "op-checklist-submit", CorrelationID: "corr-checklist", InspectionID: "audit-cabin-001", ExpectedChecklistRevision: 1,
	})
	if err != nil || submitted.Status != checklists.StatusSubmitted || submitted.Revision != 2 {
		t.Fatalf("submit checklist = %+v, err = %v", submitted, err)
	}
	if _, err := service.UpsertChecklistResponse(context.Background(), inspector, application.UpsertChecklistResponseCommand{
		OperationID: "op-response-after-submit", CorrelationID: "corr-checklist", ResponseID: "response-cabin-001",
		InspectionID: "audit-cabin-001", PackageID: "package-cabin-001", QuestionID: "q-cabin-crew-training",
		ExpectedResponseRevision: int64Pointer(2), Answer: "COMPLIANT", CommentToAuditee: "should be read-only",
	}); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("submitted checklist edit error = %v", err)
	}

	reopened, err := service.ReopenChecklist(context.Background(), principal("lead-001", "caa", "session-lead", identity.RoleLeadInspector), application.ReopenChecklistCommand{
		OperationID: "op-checklist-reopen", CorrelationID: "corr-checklist", InspectionID: "audit-cabin-001",
		ExpectedChecklistRevision: 2, Reason: "Documented response correction required.",
	})
	if err != nil || reopened.Status != checklists.StatusInProgress || reopened.Revision != 3 {
		t.Fatalf("reopen checklist = %+v, err = %v", reopened, err)
	}
}

func TestInspectorCreatesOnlyQuestionScopedPotentialFindingWithoutFindingAuthority(t *testing.T) {
	pool := canonicalDatabase(t, "create_potential_finding")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO inspection_question_assignments (inspection_id, question_id, subject_id, assignment_revision)
		VALUES ('audit-cabin-001', 'q-cabin-door-check', 'inspector-cabin-001', 1)
	`); err != nil {
		t.Fatalf("seed question assignment: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO checklist_responses (
			id, inspection_id, package_id, question_id, assigned_inspector_subject_id,
			response_value, comment_to_auditee, internal_caa_note, revision
		) VALUES (
			'response-door-001', 'audit-cabin-001', 'package-cabin-001', 'q-cabin-door-check',
			'inspector-cabin-001', 'NON_COMPLIANT', 'Door check record gap.', 'Internal pattern review.', 1
		)
	`); err != nil {
		t.Fatalf("seed question response: %v", err)
	}
	service := testService(pool)
	inspector := principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector)
	command := application.CreatePotentialFindingCommand{
		OperationID: "op-potential-create", CorrelationID: "corr-potential",
		InspectionID: "audit-cabin-001", QuestionID: "q-cabin-door-check", ChecklistResponseID: "response-door-001",
		ExpectedChecklistResponseRevision: 1, Title: "Cabin door record gap",
		Description: "Required cabin door check record was not available.", CommentToAuditee: "Provide the completed record.",
		InternalCAANote: "Internal CAA Note: compare the prior inspection.", ExpectedEvidence: "Completed cabin door check record.",
	}
	created, err := service.CreatePotentialFinding(context.Background(), inspector, command)
	if err != nil {
		t.Fatalf("create Potential Finding: %v", err)
	}
	if created.ID == "" || created.Status != potentialfindings.StatusPendingLeadReview || created.Revision != 1 || created.FindingID != "" {
		t.Fatalf("Potential Finding result = %+v", created)
	}
	raw, err := json.Marshal(created)
	if err != nil || strings.Contains(strings.ToLower(string(raw)), "severity") {
		t.Fatalf("Inspector response exposed severity authority: %s, err = %v", raw, err)
	}
	replayed, err := service.CreatePotentialFinding(context.Background(), inspector, command)
	if err != nil || replayed != created {
		t.Fatalf("Potential Finding replay = %+v, err = %v", replayed, err)
	}
	var potentialCount, findingCount int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM potential_findings WHERE checklist_response_id = 'response-door-001'").Scan(&potentialCount); err != nil || potentialCount != 1 {
		t.Fatalf("Potential Finding count = %d, err = %v", potentialCount, err)
	}
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM findings WHERE potential_finding_id = $1", created.ID).Scan(&findingCount); err != nil || findingCount != 0 {
		t.Fatalf("Inspector-created Finding count = %d, err = %v", findingCount, err)
	}
}

func TestLeadReturnOrDismissPotentialFindingIsSeparateReasonRequiredDecision(t *testing.T) {
	pool := canonicalDatabase(t, "decide_potential_finding")
	service := testService(pool)
	lead := principal("lead-001", "caa", "session-lead", identity.RoleLeadInspector)
	if _, err := service.DecidePotentialFinding(context.Background(), lead, application.DecidePotentialFindingCommand{
		OperationID: "op-potential-no-reason", CorrelationID: "corr-potential", PotentialFindingID: "potential-cabin-001",
		ExpectedRevision: 1, Decision: potentialfindings.DecisionReturn,
	}); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("reasonless return error = %v", err)
	}
	result, err := service.DecidePotentialFinding(context.Background(), lead, application.DecidePotentialFindingCommand{
		OperationID: "op-potential-return", CorrelationID: "corr-potential", PotentialFindingID: "potential-cabin-001",
		ExpectedRevision: 1, Decision: potentialfindings.DecisionReturn, Reason: "Clarify the expected Evidence basis.",
	})
	if err != nil || result.Status != potentialfindings.StatusReturned || result.Revision != 2 {
		t.Fatalf("Lead return = %+v, err = %v", result, err)
	}
	if _, err := service.DecidePotentialFinding(context.Background(), principal("inspector-cabin-001", "caa", "session-inspector", identity.RoleInspector), application.DecidePotentialFindingCommand{
		OperationID: "op-potential-inspector", CorrelationID: "corr-potential", PotentialFindingID: "potential-cabin-001",
		ExpectedRevision: 2, Decision: potentialfindings.DecisionDismiss, Reason: "not authorized",
	}); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("Inspector Lead-decision error = %v", err)
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}
