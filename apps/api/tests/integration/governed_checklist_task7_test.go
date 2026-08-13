//go:build canonicaltest

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/checklistgovernance"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/regulatory"
)

func TestTask7PublishedSelectionRequiresCurrentScopeTypedTargetInspectionAndDepartment(t *testing.T) {
	ctx := context.Background()
	service, manager, submitted := task6SubmittedCandidate(t, "task7_applicability")
	prePublicationRequest := checklistgovernance.PublishedChecklistSelectionRequest{
		OrganizationID: "ORG-SYNTHETIC-AOC", InspectionType: "RAMP_INSPECTION",
		TargetID: "TARGET-SYNTHETIC-AOC", TargetKind: "ORGANIZATION",
		DepartmentID: "FLIGHT_OPERATIONS_INSPECTORATE", At: time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC),
	}
	if unpublished, err := service.ListApplicablePublishedVersions(ctx, prePublicationRequest); err != nil || len(unpublished) != 0 {
		t.Fatalf("unpublished candidate was selectable: %+v err=%v", unpublished, err)
	}
	approved, err := service.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "TASK7-APPROVE", IdempotencyKey: "TASK7-APPROVE",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest, Reason: "Approve synthetic Task 7 fixture.",
	})
	if err != nil {
		t.Fatalf("approve synthetic Task 7 fixture: %v", err)
	}
	publication, err := service.Publish(ctx, manager, checklistgovernance.PublicationCommand{
		OperationID: "TASK7-PUBLISH", IdempotencyKey: "TASK7-PUBLISH",
		CandidateID: approved.CandidateID, ExpectedRevision: approved.Revision,
		ExpectedContentDigest: approved.ContentDigest, Reason: "Publish synthetic Task 7 fixture.",
	})
	if err != nil {
		t.Fatalf("publish synthetic Task 7 fixture: %v", err)
	}
	request := checklistgovernance.PublishedChecklistSelectionRequest{
		OrganizationID: "ORG-SYNTHETIC-AOC", InspectionType: "RAMP_INSPECTION",
		TargetID: "TARGET-SYNTHETIC-AOC", TargetKind: "ORGANIZATION",
		DepartmentID: "FLIGHT_OPERATIONS_INSPECTORATE",
		At:           time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC),
	}
	versions, err := service.ListApplicablePublishedVersions(ctx, request)
	if err != nil {
		t.Fatalf("list exact applicable version: %v", err)
	}
	if len(versions) != 1 || versions[0].TemplateVersionID != publication.TemplateVersionID ||
		versions[0].CandidateContentDigest != submitted.ContentDigest ||
		versions[0].ProviderScopeID != "SCOPE-SYNTHETIC-AOC" ||
		len(versions[0].Questions) == 0 {
		t.Fatalf("exact applicable published version=%+v", versions)
	}

	for _, mutation := range []struct {
		name   string
		mutate func(*checklistgovernance.PublishedChecklistSelectionRequest)
	}{
		{"wrong department", func(value *checklistgovernance.PublishedChecklistSelectionRequest) {
			value.DepartmentID = "AIRWORTHINESS_INSPECTORATE"
		}},
		{"wrong inspection type", func(value *checklistgovernance.PublishedChecklistSelectionRequest) {
			value.InspectionType = "BASE_INSPECTION"
		}},
		{"wrong target identity", func(value *checklistgovernance.PublishedChecklistSelectionRequest) {
			value.TargetID = "TARGET-NOT-SYNTHETIC"
		}},
		{"wrong target kind", func(value *checklistgovernance.PublishedChecklistSelectionRequest) { value.TargetKind = "DEVICE" }},
		{"before effective date", func(value *checklistgovernance.PublishedChecklistSelectionRequest) {
			value.At = time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
		}},
	} {
		t.Run(mutation.name, func(t *testing.T) {
			wrong := request
			mutation.mutate(&wrong)
			selected, err := service.ListApplicablePublishedVersions(ctx, wrong)
			if err != nil {
				t.Fatalf("selection %s: %v", mutation.name, err)
			}
			if len(selected) != 0 {
				t.Fatalf("selection %s leaked version=%+v", mutation.name, selected)
			}
			if _, err := checklistgovernance.ComposeApplicablePublishedPackage(wrong, selected); !errors.Is(err, checklistgovernance.ErrPublishedChecklistNotApplicable) {
				t.Fatalf("composition %s error=%v, want not applicable", mutation.name, err)
			}
		})
	}
	if _, err := service.Pool.Exec(ctx, `
		INSERT INTO organization_service_provider_scopes (
			id,root_id,supersedes_id,organization_id,service_provider_type_id,authorization_identifier,status,effective_from
		) VALUES (
			'SCOPE-SYNTHETIC-AOC-TARGET-REMOVED','SCOPE-SYNTHETIC-AOC','SCOPE-SYNTHETIC-AOC',
			'ORG-SYNTHETIC-AOC','AIR_OPERATOR','AOC-SYNTHETIC','ACTIVE','2026-07-01'
		)
	`); err != nil {
		t.Fatalf("append target-removing active scope successor: %v", err)
	}
	if targetRemoved, err := service.ListApplicablePublishedVersions(ctx, request); err != nil || len(targetRemoved) != 0 {
		t.Fatalf("target-removed current scope selected published version=%+v err=%v", targetRemoved, err)
	}
	if _, err := service.Pool.Exec(ctx, `
		INSERT INTO organization_service_provider_scopes (
			id,root_id,supersedes_id,organization_id,service_provider_type_id,authorization_identifier,status,effective_from,primary_target_id
		) VALUES (
			'SCOPE-SYNTHETIC-AOC-SUSPENDED','SCOPE-SYNTHETIC-AOC','SCOPE-SYNTHETIC-AOC-TARGET-REMOVED',
			'ORG-SYNTHETIC-AOC','AIR_OPERATOR','AOC-SYNTHETIC','SUSPENDED','2026-07-15','TARGET-SYNTHETIC-AOC'
		)
	`); err != nil {
		t.Fatalf("append inactive scope successor: %v", err)
	}
	stale, err := service.ListApplicablePublishedVersions(ctx, request)
	if err != nil || len(stale) != 0 {
		t.Fatalf("suspended current scope selected published version=%+v err=%v", stale, err)
	}
}

func TestTask7MaterializedPublishedPackagePinsFullApplicabilityAndSurvivesLaterPublication(t *testing.T) {
	ctx := context.Background()
	service, manager, submitted := task6SubmittedCandidate(t, "task7_materialize")
	approved, err := service.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "TASK7-MATERIALIZE-APPROVE", IdempotencyKey: "TASK7-MATERIALIZE-APPROVE",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest, Reason: "Approve exact synthetic execution fixture.",
	})
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	publication, err := service.Publish(ctx, manager, checklistgovernance.PublicationCommand{
		OperationID: "TASK7-MATERIALIZE-PUBLISH", IdempotencyKey: "TASK7-MATERIALIZE-PUBLISH",
		CandidateID: approved.CandidateID, ExpectedRevision: approved.Revision,
		ExpectedContentDigest: approved.ContentDigest, Reason: "Publish exact synthetic execution fixture.",
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if _, err := service.Pool.Exec(ctx, `
		INSERT INTO identity_references (subject_id,issuer,display_name)
		VALUES ('USR-TASK7-INSPECTOR','task7-test','Task 7 Inspector'),
		       ('USR-TASK7-OTHER-INSPECTOR','task7-test','Task 7 Other Inspector');
		INSERT INTO inspections (id,organization_id,assigned_inspector_subject_id,title,inspection_type,status)
		VALUES ('INSP-TASK7','ORG-SYNTHETIC-AOC','USR-TASK7-INSPECTOR','Synthetic Task 7 Audit','RAMP_INSPECTION','PREPARATION')
	`); err != nil {
		t.Fatalf("seed inspection: %v", err)
	}
	request := checklistgovernance.PublishedChecklistSelectionRequest{
		OrganizationID: "ORG-SYNTHETIC-AOC", InspectionType: "RAMP_INSPECTION",
		TargetID: "TARGET-SYNTHETIC-AOC", TargetKind: "ORGANIZATION",
		DepartmentID: "FLIGHT_OPERATIONS_INSPECTORATE", At: time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC),
	}
	materializeCommand := checklistgovernance.MaterializeApplicablePublishedPackageCommand{
		OperationID: "TASK7-MATERIALIZE", IdempotencyKey: "TASK7-MATERIALIZE", CorrelationID: "TASK7-MATERIALIZE", InspectionID: "INSP-TASK7",
		PackageID: "PKG-TASK7", PackageVersion: 1, ExpiresAt: time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC),
		Selection: request, AssignedInspectorSubjectIDs: map[string][]string{"Q-SYNTHETIC-OPS-AOC-001": {"USR-TASK7-INSPECTOR"}},
	}
	result, err := service.MaterializeApplicablePublishedPackage(ctx, manager, materializeCommand)
	if err != nil {
		t.Fatalf("materialize governed package: %v", err)
	}
	if result.TemplateVersionID != publication.TemplateVersionID || result.PackageDigest == "" {
		t.Fatalf("materialized result=%+v", result)
	}
	if replay, err := service.MaterializeApplicablePublishedPackage(ctx, manager, materializeCommand); err != nil || replay != result {
		t.Fatalf("materialization replay=%+v err=%v, want exact original=%+v", replay, err, result)
	}
	if _, err := service.Pool.Exec(ctx, `
		INSERT INTO identity_references (subject_id,issuer,display_name)
		VALUES ('USR-TASK7-LEAD','task7-test','Task 7 Lead Inspector');
		INSERT INTO audit_assignments (
			id, inspection_id, organization_id, lead_subject_id, status,
			scheduled_start_date, scheduled_end_date, revision
		) VALUES (
			'ASSIGN-TASK7', 'INSP-TASK7', 'ORG-SYNTHETIC-AOC',
			'USR-TASK7-LEAD', 'IN_PROGRESS', '2026-07-29', '2026-08-29', 1
		);
		INSERT INTO audit_team_members (assignment_id, subject_id, member_role, revision)
		VALUES
			('ASSIGN-TASK7', 'USR-TASK7-LEAD', 'LEAD_INSPECTOR', 1),
			('ASSIGN-TASK7', 'USR-TASK7-INSPECTOR', 'INSPECTOR', 1);
		INSERT INTO audit_question_assignments (assignment_id, question_id, subject_id, revision)
		VALUES ('ASSIGN-TASK7', 'Q-SYNTHETIC-OPS-AOC-001', 'USR-TASK7-INSPECTOR', 1)
	`); err != nil {
		t.Fatalf("seed exact canonical Task 7 execution assignment: %v", err)
	}
	changedReplay := materializeCommand
	changedReplay.ExpiresAt = changedReplay.ExpiresAt.AddDate(0, 0, 1)
	if _, err := service.MaterializeApplicablePublishedPackage(ctx, manager, changedReplay); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("changed materialization replay accepted: %v", err)
	}
	var snapshotBefore []byte
	var checklistStatus, inspectionStatus string
	if err := service.Pool.QueryRow(ctx, `SELECT snapshot FROM inspection_packages WHERE id='PKG-TASK7'`).Scan(&snapshotBefore); err != nil {
		t.Fatal(err)
	}
	var pinned struct {
		PublishedVersions  []checklistgovernance.PublishedChecklistVersionPin     `json:"publishedVersions"`
		Applicability      checklistgovernance.PublishedChecklistApplicabilityPin `json:"applicability"`
		PublishedQuestions []regulatory.ChecklistQuestion                         `json:"publishedQuestions"`
	}
	if err := json.Unmarshal(snapshotBefore, &pinned); err != nil {
		t.Fatalf("decode governed package snapshot: %v", err)
	}
	if len(pinned.PublishedVersions) != 1 || pinned.PublishedVersions[0].TemplateVersionID != publication.TemplateVersionID || pinned.PublishedVersions[0].CandidateContentDigest != submitted.ContentDigest ||
		pinned.Applicability.OrganizationID != request.OrganizationID || pinned.Applicability.TargetID != request.TargetID || pinned.Applicability.TargetKind != request.TargetKind || pinned.Applicability.DepartmentID != request.DepartmentID ||
		len(pinned.PublishedQuestions) != 1 || pinned.PublishedQuestions[0].QuestionID != "Q-SYNTHETIC-OPS-AOC-001" || len(pinned.PublishedQuestions[0].MappingIDs) != 1 || len(pinned.PublishedQuestions[0].Citations) != 1 || len(pinned.PublishedQuestions[0].ExpectedEvidence) != 2 {
		t.Fatalf("package did not pin exact governed version/scope/question bytes: %+v", pinned)
	}
	if err := service.Pool.QueryRow(ctx, `SELECT status FROM inspection_checklists WHERE inspection_id='INSP-TASK7'`).Scan(&checklistStatus); err != nil || checklistStatus != "IN_PROGRESS" {
		t.Fatalf("checklist status=%q err=%v", checklistStatus, err)
	}
	if err := service.Pool.QueryRow(ctx, `SELECT status FROM inspections WHERE id='INSP-TASK7'`).Scan(&inspectionStatus); err != nil || inspectionStatus != "READY_TO_EXECUTE" {
		t.Fatalf("inspection status=%q err=%v", inspectionStatus, err)
	}
	var assignments int
	if err := service.Pool.QueryRow(ctx, `SELECT count(*) FROM inspection_question_assignments WHERE inspection_id='INSP-TASK7'`).Scan(&assignments); err != nil || assignments != 1 {
		t.Fatalf("assignments=%d err=%v", assignments, err)
	}
	if _, err := service.MaterializeApplicablePublishedPackage(ctx, manager, checklistgovernance.MaterializeApplicablePublishedPackageCommand{
		OperationID: "TASK7-MATERIALIZE-WRONG-DEPARTMENT", CorrelationID: "TASK7-MATERIALIZE-WRONG-DEPARTMENT", InspectionID: "INSP-TASK7",
		PackageID: "PKG-TASK7-DENIED", PackageVersion: 2, ExpiresAt: time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC),
		Selection:                   checklistgovernance.PublishedChecklistSelectionRequest{OrganizationID: request.OrganizationID, InspectionType: request.InspectionType, TargetID: request.TargetID, TargetKind: request.TargetKind, DepartmentID: "AIRWORTHINESS_INSPECTORATE", At: request.At},
		AssignedInspectorSubjectIDs: map[string][]string{"Q-SYNTHETIC-OPS-AOC-001": {"USR-TASK7-INSPECTOR"}},
	}); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("cross-department materialization accepted: %v", err)
	}
	runner := application.NewService(service.Pool, application.Dependencies{Clock: func() time.Time { return time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC) }})
	inspector := identity.Principal{SubjectID: "USR-TASK7-INSPECTOR", Roles: []identity.Role{identity.RoleInspector}}
	otherInspector := identity.Principal{SubjectID: "USR-TASK7-OTHER-INSPECTOR", Roles: []identity.Role{identity.RoleInspector}}
	if _, err := runner.UpsertChecklistResponse(ctx, otherInspector, application.UpsertChecklistResponseCommand{
		OperationID: "TASK7-OTHER-ANSWER", CorrelationID: "TASK7-OTHER-ANSWER", ResponseID: "RESP-TASK7-OTHER", InspectionID: "INSP-TASK7", PackageID: "PKG-TASK7", QuestionID: "Q-SYNTHETIC-OPS-AOC-001", Answer: "COMPLIANT",
	}); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("unassigned Inspector answer accepted: %v", err)
	}
	if response, err := runner.UpsertChecklistResponse(ctx, inspector, application.UpsertChecklistResponseCommand{
		OperationID: "TASK7-ANSWER", CorrelationID: "TASK7-ANSWER", ResponseID: "RESP-TASK7", InspectionID: "INSP-TASK7", PackageID: "PKG-TASK7", QuestionID: "Q-SYNTHETIC-OPS-AOC-001", Answer: "NON_COMPLIANT", CommentToAuditee: "Synthetic discrepancy documented for the auditee.",
	}); !errors.Is(err, application.ErrConflict) || response.Revision != 0 {
		t.Fatalf("non-canonical published package entered execution: response=%+v err=%v", response, err)
	}
	var executionResponses, executionFindings int
	if err := service.Pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM checklist_responses WHERE inspection_id='INSP-TASK7'),
			(SELECT count(*) FROM potential_findings WHERE inspection_id='INSP-TASK7')
	`).Scan(&executionResponses, &executionFindings); err != nil {
		t.Fatalf("read denied non-canonical execution effects: %v", err)
	}
	if executionResponses != 0 || executionFindings != 0 {
		t.Fatalf("non-canonical execution effects = responses %d Potential Findings %d", executionResponses, executionFindings)
	}
	if _, err := service.Pool.Exec(ctx, `UPDATE inspection_packages SET snapshot='{}'::jsonb WHERE id='PKG-TASK7'`); err == nil {
		t.Fatal("Inspector/package snapshot mutation unexpectedly succeeded")
	}
	if _, err := service.Pool.Exec(ctx, `
		INSERT INTO organization_service_provider_scopes (
			id,root_id,supersedes_id,organization_id,service_provider_type_id,authorization_identifier,status,effective_from,primary_target_id
		) VALUES (
			'SCOPE-SYNTHETIC-AOC-SUSPENDED-AFTER-PACKAGE','SCOPE-SYNTHETIC-AOC','SCOPE-SYNTHETIC-AOC',
			'ORG-SYNTHETIC-AOC','AIR_OPERATOR','AOC-SYNTHETIC','SUSPENDED','2026-07-30','TARGET-SYNTHETIC-AOC'
		)
	`); err != nil {
		t.Fatalf("append post-package scope change: %v", err)
	}
	if _, err := service.Pool.Exec(ctx, `UPDATE checklist_template_versions SET title='forbidden mutation' WHERE id=$1`, publication.TemplateVersionID); err == nil {
		t.Fatal("published version mutation unexpectedly succeeded")
	}
	var snapshotAfter []byte
	if err := service.Pool.QueryRow(ctx, `SELECT snapshot FROM inspection_packages WHERE id='PKG-TASK7'`).Scan(&snapshotAfter); err != nil {
		t.Fatal(err)
	}
	if string(snapshotBefore) != string(snapshotAfter) {
		t.Fatal("in-progress Audit package changed after later publication/source mutation attempt")
	}
}
