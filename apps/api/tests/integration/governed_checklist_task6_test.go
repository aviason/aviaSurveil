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
	"github.com/aviason/aviaSurveil/internal/checklistgovernance"
	"github.com/aviason/aviaSurveil/internal/httpapi"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/regulatory"
	"github.com/aviason/aviaSurveil/internal/testprofile"
	"github.com/aviason/aviaSurveil/migrations"
)

func task6SubmittedCandidate(t *testing.T, label string) (*checklistgovernance.Service, identity.Principal, regulatory.CandidateView) {
	t.Helper()
	ctx := context.Background()
	pool := createTestDatabase(t, label)
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migration 21: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap synthetic inputs: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO identity_references (subject_id,issuer,display_name) VALUES
			('USR-TASK6-ADMIN','task6-test','Task 6 Admin'),
			('USR-TASK6-FOI-MANAGER','task6-test','Task 6 FOI Manager'),
			('USR-TASK6-AIR-MANAGER','task6-test','Task 6 AIR Manager'),
			('USR-TASK6-GENERIC-MANAGER','task6-test','Task 6 Generic Manager');
		INSERT INTO caa_department_memberships
			(id,subject_id,department_id,organizational_unit_id,membership_role,status,effective_from)
		VALUES
			('MEM-TASK6-FOI','USR-TASK6-FOI-MANAGER','FLIGHT_OPERATIONS_INSPECTORATE','FLIGHT_OPERATIONS_INSPECTORATE','DEPARTMENT_MANAGER','ACTIVE','2025-01-01'),
			('MEM-TASK6-AIR','USR-TASK6-AIR-MANAGER','AIRWORTHINESS_INSPECTORATE','AIRWORTHINESS_INSPECTORATE','DEPARTMENT_MANAGER','ACTIVE','2025-01-01')`); err != nil {
		t.Fatalf("seed Task 6 identities and assignments: %v", err)
	}
	now := time.Date(2026, 7, 29, 16, 0, 0, 0, time.UTC)
	adminService := regulatory.NewAdminService(pool, func() time.Time { return now })
	admin := identity.Principal{SubjectID: "USR-TASK6-ADMIN", Roles: []identity.Role{identity.RoleAdmin}}
	run, err := adminService.Import(ctx, admin, "TASK6-IMPORT-"+label, "TASK6-IMPORT-"+label, regulatory.SyntheticCandidateBundle())
	if err != nil || run.Candidate == nil {
		t.Fatalf("import Task 6 candidate: %+v err=%v", run, err)
	}
	submitted, err := adminService.Submit(ctx, admin, regulatory.SubmitCommand{
		OperationID: "TASK6-SUBMIT-" + label, IdempotencyKey: "TASK6-SUBMIT-" + label,
		CandidateID: run.Candidate.CandidateID, ExpectedRevision: run.Candidate.Revision,
		ExpectedContentDigest: run.Candidate.ContentDigest, Reason: "Submit exact synthetic candidate for Task 6 review.",
	})
	if err != nil {
		t.Fatalf("submit Task 6 candidate: %v", err)
	}
	manager := identity.Principal{SubjectID: "USR-TASK6-FOI-MANAGER", Roles: []identity.Role{identity.RoleDepartmentManager}}
	return checklistgovernance.NewService(pool, func() time.Time { return now }), manager, submitted
}

// Break caught: a manager role claim alone must never reveal or mutate a
// governed candidate; the queue and commands must resolve the current exact
// department/unit assignment from PostgreSQL.
func TestTask6DepartmentFilteredQueueAndCurrentAssignmentAuthority(t *testing.T) {
	ctx := context.Background()
	service, foiManager, submitted := task6SubmittedCandidate(t, "task6_queue_authority")
	queue, err := service.ListQueue(ctx, foiManager)
	if err != nil || len(queue) != 1 || queue[0].Candidate.CandidateID != submitted.CandidateID {
		t.Fatalf("FOI queue=%+v err=%v", queue, err)
	}
	if len(queue[0].RequiredOwners) != 1 ||
		queue[0].RequiredOwners[0].DepartmentID != "FLIGHT_OPERATIONS_INSPECTORATE" ||
		queue[0].RequiredOwners[0].OrganizationalUnitID != "FLIGHT_OPERATIONS_INSPECTORATE" ||
		!queue[0].RequiredOwners[0].ApprovalRequired {
		t.Fatalf("queue lost exact owner authority: %+v", queue[0])
	}
	crossDepartment := identity.Principal{SubjectID: "USR-TASK6-AIR-MANAGER", Roles: []identity.Role{identity.RoleDepartmentManager}}
	if got, err := service.ListQueue(ctx, crossDepartment); err != nil || len(got) != 0 {
		t.Fatalf("cross-department queue leaked candidate: %+v err=%v", got, err)
	}
	generic := identity.Principal{SubjectID: "USR-TASK6-GENERIC-MANAGER", Roles: []identity.Role{identity.RoleDepartmentManager}}
	if _, err := service.ListQueue(ctx, generic); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("generic manager role obtained queue authority: %v", err)
	}
	command := checklistgovernance.ReviewCommand{
		OperationID: "TASK6-CROSS-RETURN", IdempotencyKey: "TASK6-CROSS-RETURN",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest, Reason: "Cross-department return must fail.",
	}
	if _, err := service.Return(ctx, crossDepartment, command); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("cross-department return was accepted: %v", err)
	}
}

// Break caught: return/reject are real, separately attributed lifecycle
// decisions. Identical retries replay; changed semantics conflict; and each
// command writes one decision and one Audit event.
func TestTask6ReturnAndRejectAreExactTerminalOrRevisionRequiredTransitions(t *testing.T) {
	ctx := context.Background()
	t.Run("return", func(t *testing.T) {
		service, manager, submitted := task6SubmittedCandidate(t, "task6_return")
		command := checklistgovernance.ReviewCommand{
			OperationID: "TASK6-RETURN", IdempotencyKey: "TASK6-RETURN",
			CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
			ExpectedContentDigest: submitted.ContentDigest, Reason: "Return for one controlled revision.",
		}
		returned, err := service.Return(ctx, manager, command)
		if err != nil || returned.Status != "RETURNED" {
			t.Fatalf("return result=%+v err=%v", returned, err)
		}
		replay, err := service.Return(ctx, manager, command)
		if err != nil || replay.CandidateID != returned.CandidateID || replay.Status != "RETURNED" {
			t.Fatalf("return replay=%+v err=%v", replay, err)
		}
		changed := command
		changed.Reason = "Conflicting reason."
		if _, err := service.Return(ctx, manager, changed); !errors.Is(err, application.ErrConflict) {
			t.Fatalf("conflicting return identity was accepted: %v", err)
		}
		assertTask6DecisionEffect(t, service, command.OperationID, "RETURNED", 1)
	})
	t.Run("reject", func(t *testing.T) {
		service, manager, submitted := task6SubmittedCandidate(t, "task6_reject")
		command := checklistgovernance.ReviewCommand{
			OperationID: "TASK6-REJECT", IdempotencyKey: "TASK6-REJECT",
			CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
			ExpectedContentDigest: submitted.ContentDigest, Reason: "Reject terminally with exact attribution.",
		}
		rejected, err := service.Reject(ctx, manager, command)
		if err != nil || rejected.Status != "REJECTED" {
			t.Fatalf("reject result=%+v err=%v", rejected, err)
		}
		if _, err := service.Return(ctx, manager, checklistgovernance.ReviewCommand{
			OperationID: "TASK6-ILLEGAL-RETURN", IdempotencyKey: "TASK6-ILLEGAL-RETURN",
			CandidateID: rejected.CandidateID, ExpectedRevision: rejected.Revision,
			ExpectedContentDigest: rejected.ContentDigest, Reason: "Rejected is terminal.",
		}); !errors.Is(err, application.ErrConflict) {
			t.Fatalf("terminal rejected candidate accepted another transition: %v", err)
		}
		assertTask6DecisionEffect(t, service, command.OperationID, "REJECTED", 1)
	})
}

// Break caught: RETURNED content must never be edited in place. Admin creates
// a new immutable Generated Draft leaf, and only that exact leaf may be
// resubmitted for a fresh review.
func TestTask6ReturnedCandidateCreatesImmutableRevisionAndResubmitsExactLeaf(t *testing.T) {
	ctx := context.Background()
	service, manager, submitted := task6SubmittedCandidate(t, "task6_revise_resubmit")
	returned, err := service.Return(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "TASK6-RETURN-FOR-REVISION", IdempotencyKey: "TASK6-RETURN-FOR-REVISION",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest, Reason: "Return for a controlled rationale revision.",
	})
	if err != nil {
		t.Fatalf("return for revision: %v", err)
	}
	adminService := regulatory.NewAdminService(service.Pool, service.Clock)
	mappings := append([]regulatory.ComplianceMapping(nil), returned.Mappings...)
	mappings[0].Rationale = "Synthetic test-profile rationale reviewed by an Admin without changing the controlled source claim."
	revised, err := adminService.CreateRevision(ctx, identity.Principal{
		SubjectID: "USR-TASK6-ADMIN", Roles: []identity.Role{identity.RoleAdmin},
	}, regulatory.EditCommand{
		OperationID: "TASK6-REVISE-RETURNED", IdempotencyKey: "TASK6-REVISE-RETURNED",
		CandidateID: returned.CandidateID, ExpectedRevision: returned.Revision,
		ExpectedContentDigest: returned.ContentDigest, ChangeReason: "Apply the returned controlled correction.",
		Mappings: mappings, Questions: returned.Questions, RequiredOwners: returned.RequiredOwners,
	})
	if err != nil {
		t.Fatalf("create returned successor: %v", err)
	}
	if revised.CandidateID == returned.CandidateID || revised.Revision != returned.Revision+1 ||
		revised.Status != regulatory.GeneratedDraft || revised.SupersedesCandidateID == nil ||
		*revised.SupersedesCandidateID != returned.CandidateID {
		t.Fatalf("invalid returned successor: %+v", revised)
	}
	// A resolved Draft revision must reuse the exact already-activated source
	// currentness proof. It cannot silently drop the binding (which would make
	// the revised trace appear current) or create a new activation while the
	// Admin is only editing candidate wording/rationale.
	var currentnessBindings int
	if err := service.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM regulatory_generation_run_source_currentness_bindings binding
		JOIN regulatory_source_currentness_events event ON event.event_id=binding.currentness_event_id
		WHERE binding.generation_run_id=$1
		  AND binding.current_source_version_id='SOURCE-SYNTHETIC-OPS-AOC'
		  AND binding.current_source_hash='sha256:1111111111111111111111111111111111111111111111111111111111111111'
		  AND event.current_source_version_id=binding.current_source_version_id
		  AND event.current_source_hash=binding.current_source_hash`, revised.GenerationRunID).Scan(&currentnessBindings); err != nil || currentnessBindings != 1 {
		t.Fatalf("resolved revision did not retain exact immutable source-currentness binding: count=%d err=%v", currentnessBindings, err)
	}
	var originalStatus string
	if err := service.Pool.QueryRow(ctx, `SELECT status FROM template_draft_versions WHERE id=$1`, returned.CandidateID).Scan(&originalStatus); err != nil || originalStatus != "RETURNED" {
		t.Fatalf("returned ancestor was mutated: status=%s err=%v", originalStatus, err)
	}
	resubmitted, err := adminService.Submit(ctx, identity.Principal{
		SubjectID: "USR-TASK6-ADMIN", Roles: []identity.Role{identity.RoleAdmin},
	}, regulatory.SubmitCommand{
		OperationID: "TASK6-RESUBMIT-REVISION", IdempotencyKey: "TASK6-RESUBMIT-REVISION",
		CandidateID: revised.CandidateID, ExpectedRevision: revised.Revision,
		ExpectedContentDigest: revised.ContentDigest, Reason: "Resubmit exact corrected leaf.",
	})
	if err != nil || resubmitted.Status != "DEPARTMENT_REVIEW" {
		t.Fatalf("resubmit result=%+v err=%v", resubmitted, err)
	}
}

// Break caught: one owner must never approve joint material for all owners.
// Each exact current owner decision is append-only; only the final required
// owner advances the candidate, and old-revision decisions never transfer.
func TestTask6JointPartialCompleteApprovalAndEditedReviewInvalidation(t *testing.T) {
	ctx := context.Background()
	service, foiManager, submitted := task6SubmittedCandidate(t, "task6_joint")
	if _, err := service.Pool.Exec(ctx, `
		INSERT INTO candidate_required_owner_assignments
			(id,candidate_draft_version_id,candidate_revision,candidate_content_digest,
			 department_id,organizational_unit_id,approval_required)
		VALUES ('OWNER-TASK6-JOINT-AIR',$1,$2,$3,
			'AIRWORTHINESS_INSPECTORATE','AIRWORTHINESS_INSPECTORATE',true)`,
		submitted.CandidateID, submitted.Revision, submitted.ContentDigest); err != nil {
		t.Fatalf("add exact joint owner: %v", err)
	}
	airManager := identity.Principal{SubjectID: "USR-TASK6-AIR-MANAGER", Roles: []identity.Role{identity.RoleDepartmentManager}}
	partial, err := service.Approve(ctx, foiManager, checklistgovernance.ReviewCommand{
		OperationID: "TASK6-APPROVE-FOI", IdempotencyKey: "TASK6-APPROVE-FOI",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest, Reason: "FOI approves only its owned scope.",
	})
	if err != nil || partial.Status != "DEPARTMENT_REVIEW" {
		t.Fatalf("partial approval=%+v err=%v", partial, err)
	}
	if replay, err := service.Approve(ctx, foiManager, checklistgovernance.ReviewCommand{
		OperationID: "TASK6-APPROVE-FOI", IdempotencyKey: "TASK6-APPROVE-FOI",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest, Reason: "FOI approves only its owned scope.",
	}); err != nil || replay.Status != "DEPARTMENT_REVIEW" {
		t.Fatalf("partial approval replay=%+v err=%v", replay, err)
	}
	approved, err := service.Approve(ctx, airManager, checklistgovernance.ReviewCommand{
		OperationID: "TASK6-APPROVE-AIR", IdempotencyKey: "TASK6-APPROVE-AIR",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest, Reason: "AIR approves only its owned scope.",
	})
	if err != nil || approved.Status != "TECHNICALLY_APPROVED" {
		t.Fatalf("complete joint approval=%+v err=%v", approved, err)
	}
	var approvals int
	if err := service.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM department_review_decisions
		WHERE candidate_draft_version_id=$1 AND candidate_revision=$2
		  AND candidate_content_digest=$3 AND decision='TECHNICALLY_APPROVED'`,
		submitted.CandidateID, submitted.Revision, submitted.ContentDigest).Scan(&approvals); err != nil || approvals != 2 {
		t.Fatalf("joint approvals=%d want=2 err=%v", approvals, err)
	}

	// A separate returned joint review proves fail-closed invalidation.
	invalidationService, manager, invalidationCandidate := task6SubmittedCandidate(t, "task6_invalidation")
	if _, err := invalidationService.Pool.Exec(ctx, `
		INSERT INTO candidate_required_owner_assignments
			(id,candidate_draft_version_id,candidate_revision,candidate_content_digest,
			 department_id,organizational_unit_id,approval_required)
		VALUES ('OWNER-TASK6-INVALIDATION-AIR',$1,$2,$3,
			'AIRWORTHINESS_INSPECTORATE','AIRWORTHINESS_INSPECTORATE',true)`,
		invalidationCandidate.CandidateID, invalidationCandidate.Revision, invalidationCandidate.ContentDigest); err != nil {
		t.Fatalf("add invalidation joint owner: %v", err)
	}
	if _, err := invalidationService.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "TASK6-OLD-APPROVAL", IdempotencyKey: "TASK6-OLD-APPROVAL",
		CandidateID: invalidationCandidate.CandidateID, ExpectedRevision: invalidationCandidate.Revision,
		ExpectedContentDigest: invalidationCandidate.ContentDigest, Reason: "Old revision approval.",
	}); err != nil {
		t.Fatalf("approve old review: %v", err)
	}
	returned, err := invalidationService.Return(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "TASK6-RETURN-PARTIAL", IdempotencyKey: "TASK6-RETURN-PARTIAL",
		CandidateID: invalidationCandidate.CandidateID, ExpectedRevision: invalidationCandidate.Revision,
		ExpectedContentDigest: invalidationCandidate.ContentDigest, Reason: "Return after only one joint owner approved.",
	})
	if err != nil || returned.Status != "RETURNED" {
		t.Fatalf("return partial joint review=%+v err=%v", returned, err)
	}
	mappings := append([]regulatory.ComplianceMapping(nil), returned.Mappings...)
	mappings[0].Rationale = "Synthetic test-profile rationale reviewed by an Admin without changing the controlled source claim."
	revised, err := regulatory.NewAdminService(invalidationService.Pool, invalidationService.Clock).CreateRevision(ctx, identity.Principal{
		SubjectID: "USR-TASK6-ADMIN", Roles: []identity.Role{identity.RoleAdmin},
	}, regulatory.EditCommand{
		OperationID: "TASK6-INVALIDATION-REVISION", IdempotencyKey: "TASK6-INVALIDATION-REVISION",
		CandidateID: returned.CandidateID, ExpectedRevision: returned.Revision,
		ExpectedContentDigest: returned.ContentDigest, ChangeReason: "Revise after partial joint review.",
		Mappings: mappings, Questions: returned.Questions, RequiredOwners: returned.RequiredOwners,
	})
	if err != nil {
		t.Fatalf("revise partial joint review: %v", err)
	}
	var transferred int
	if err := invalidationService.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM department_review_decisions
		WHERE candidate_draft_version_id=$1 AND candidate_revision=$2
		  AND candidate_content_digest=$3 AND decision='TECHNICALLY_APPROVED'`,
		revised.CandidateID, revised.Revision, revised.ContentDigest).Scan(&transferred); err != nil || transferred != 0 {
		t.Fatalf("old decision transferred to revised content: count=%d err=%v", transferred, err)
	}
}

// Break caught: technical approval and publication must remain separate. Only
// a separately approved exact leaf can publish; publication recomputes
// persisted bytes, creates one immutable version with ordered questions, and
// identical retry returns that same version.
func TestTask6PublicationIsSeparateDigestVerifiedAndImmutable(t *testing.T) {
	ctx := context.Background()
	service, manager, submitted := task6SubmittedCandidate(t, "task6_publish")
	approved, err := service.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "TASK6-TECHNICAL-APPROVAL", IdempotencyKey: "TASK6-TECHNICAL-APPROVAL",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest, Reason: "Technically approve exact synthetic candidate.",
	})
	if err != nil || approved.Status != "TECHNICALLY_APPROVED" {
		t.Fatalf("technical approval=%+v err=%v", approved, err)
	}
	for table, want := range map[string]int{"checklist_publication_decisions": 0, "checklist_template_versions": 0} {
		var got int
		if err := service.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(&got); err != nil || got != want {
			t.Fatalf("technical approval created %s=%d err=%v", table, got, err)
		}
	}
	command := checklistgovernance.PublicationCommand{
		OperationID: "TASK6-PUBLISH", IdempotencyKey: "TASK6-PUBLISH",
		CandidateID: approved.CandidateID, ExpectedRevision: approved.Revision,
		ExpectedContentDigest: approved.ContentDigest, Reason: "Publish separately after exact technical approval.",
	}
	published, err := service.Publish(ctx, manager, command)
	if err != nil {
		t.Fatalf("publish exact approved candidate: %v", err)
	}
	if published.CandidateID != approved.CandidateID || published.CandidateRevision != approved.Revision ||
		published.CandidateContentDigest != approved.ContentDigest ||
		published.TemplateVersionID == "" || published.PublicationDecisionID == "" {
		t.Fatalf("invalid publication projection: %+v", published)
	}
	replayed, err := service.Publish(ctx, manager, command)
	if err != nil || replayed.TemplateVersionID != published.TemplateVersionID ||
		replayed.PublicationDecisionID != published.PublicationDecisionID ||
		replayed.CandidateID != published.CandidateID ||
		replayed.CandidateRevision != published.CandidateRevision ||
		replayed.CandidateContentDigest != published.CandidateContentDigest ||
		!replayed.PublishedAt.Equal(published.PublishedAt) {
		t.Fatalf("publication replay=%+v want=%+v err=%v", replayed, published, err)
	}
	conflict := command
	conflict.Reason = "Changed publication semantics."
	if _, err := service.Publish(ctx, manager, conflict); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("conflicting publication identity was accepted: %v", err)
	}
	var decisions, versions, publicationAudits, approvalAudits, orderedQuestions int
	if err := service.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM checklist_publication_decisions WHERE operation_id=$1`, command.OperationID).Scan(&decisions); err != nil {
		t.Fatal(err)
	}
	if err := service.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM checklist_template_versions WHERE id=$1 AND candidate_draft_version_id=$2 AND candidate_revision=$3 AND candidate_content_digest=$4`, published.TemplateVersionID, approved.CandidateID, approved.Revision, approved.ContentDigest).Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if err := service.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE operation_id=$1`, command.OperationID).Scan(&publicationAudits); err != nil {
		t.Fatal(err)
	}
	if err := service.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE operation_id='TASK6-TECHNICAL-APPROVAL'`).Scan(&approvalAudits); err != nil {
		t.Fatal(err)
	}
	if err := service.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM template_version_questions WHERE template_version_id=$1 AND position=0`, published.TemplateVersionID).Scan(&orderedQuestions); err != nil {
		t.Fatal(err)
	}
	if decisions != 1 || versions != 1 || publicationAudits != 1 || approvalAudits != 1 || orderedQuestions != 1 {
		t.Fatalf("publication effects decisions=%d versions=%d publicationAudits=%d approvalAudits=%d orderedQuestions=%d", decisions, versions, publicationAudits, approvalAudits, orderedQuestions)
	}
}

// Break caught: trusting the candidate digest column without recomputing the
// persisted mapping/question snapshots would publish changed bytes.
func TestTask6PublicationDigestMismatchRollsBackEveryEffect(t *testing.T) {
	ctx := context.Background()
	service, manager, submitted := task6SubmittedCandidate(t, "task6_digest_rollback")
	approved, err := service.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "TASK6-ROLLBACK-APPROVE", IdempotencyKey: "TASK6-ROLLBACK-APPROVE",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest, Reason: "Approve before tamper simulation.",
	})
	if err != nil {
		t.Fatalf("approve before tamper: %v", err)
	}
	if _, err := service.Pool.Exec(ctx, `ALTER TABLE regulatory_generated_mapping_snapshots DISABLE TRIGGER regulatory_generated_mapping_snapshots_append_only`); err != nil {
		t.Fatalf("disable snapshot guard for tamper simulation: %v", err)
	}
	if _, err := service.Pool.Exec(ctx, `
		UPDATE regulatory_generated_mapping_snapshots
		SET snapshot=jsonb_set(snapshot,'{rationale}','"tampered persisted bytes"'::jsonb)
		WHERE candidate_draft_version_id=$1`,
		approved.CandidateID); err != nil {
		t.Fatalf("simulate persisted-byte mismatch: %v", err)
	}
	if _, err := service.Pool.Exec(ctx, `ALTER TABLE regulatory_generated_mapping_snapshots ENABLE TRIGGER regulatory_generated_mapping_snapshots_append_only`); err != nil {
		t.Fatalf("restore snapshot guard after tamper simulation: %v", err)
	}
	if _, err := service.Publish(ctx, manager, checklistgovernance.PublicationCommand{
		OperationID: "TASK6-PUBLISH-TAMPERED", IdempotencyKey: "TASK6-PUBLISH-TAMPERED",
		CandidateID: approved.CandidateID, ExpectedRevision: approved.Revision,
		ExpectedContentDigest: approved.ContentDigest, Reason: "Must roll back mismatched bytes.",
	}); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("tampered persisted bytes publication error=%v, want conflict", err)
	}
	for table, want := range map[string]int{"checklist_publication_decisions": 0, "checklist_template_versions": 0} {
		var got int
		if err := service.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(&got); err != nil || got != want {
			t.Fatalf("digest mismatch left %s=%d err=%v", table, got, err)
		}
	}
	var audit int
	if err := service.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE operation_id='TASK6-PUBLISH-TAMPERED'`).Scan(&audit); err != nil || audit != 0 {
		t.Fatalf("digest mismatch left Audit=%d err=%v", audit, err)
	}
}

// Break caught: a canonical manager route must use the same persisted
// lifecycle service as direct calls, preserve the approval/publication split,
// and reject an Admin principal that lacks a current department assignment.
func TestTask6CanonicalHTTPManagerLifecycleUsesRealPostgreSQLAuthority(t *testing.T) {
	ctx := context.Background()
	service, _, submitted := task6SubmittedCandidate(t, "task6_http")
	if _, err := service.Pool.Exec(ctx, `
		INSERT INTO identity_references (subject_id,issuer,display_name)
		VALUES ('USR-MANAGER-NORA','task6-test','Nora Department Manager')
		ON CONFLICT (subject_id) DO NOTHING;
		INSERT INTO caa_department_memberships
			(id,subject_id,department_id,organizational_unit_id,membership_role,status,effective_from)
		VALUES ('MEM-TASK6-NORA','USR-MANAGER-NORA',
			'FLIGHT_OPERATIONS_INSPECTORATE','FLIGHT_OPERATIONS_INSPECTORATE',
			'DEPARTMENT_MANAGER','ACTIVE','2025-01-01')`); err != nil {
		t.Fatalf("seed canonical manager assignment: %v", err)
	}
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
		Pool: service.Pool, Clock: service.Clock,
	})
	handler := httpapi.NewCanonicalTestBoundary("task6-http-token").Protect(api.Handler())
	request := func(method, path, subject string, body any) *httptest.ResponseRecorder {
		var encoded []byte
		if body != nil {
			encoded, _ = json.Marshal(body)
		}
		httpRequest := httptest.NewRequest(method, path, bytes.NewReader(encoded))
		httpRequest.Header.Set(httpapi.CanonicalTestTokenHeader, "task6-http-token")
		httpRequest.Header.Set(httpapi.CanonicalTestSubjectHeader, subject)
		if body != nil {
			httpRequest.Header.Set("Content-Type", "application/json")
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httpRequest)
		return response
	}
	queue := request(http.MethodGet, "/v1/department-manager/governed-checklist/review-queue", "USR-MANAGER-NORA", nil)
	if queue.Code != http.StatusOK || !bytes.Contains(queue.Body.Bytes(), []byte(submitted.CandidateID)) ||
		!bytes.Contains(queue.Body.Bytes(), []byte(`"status":"DEPARTMENT_REVIEW"`)) {
		t.Fatalf("canonical manager queue status=%d body=%s", queue.Code, queue.Body.String())
	}
	adminDenied := request(http.MethodGet, "/v1/department-manager/governed-checklist/review-queue", "USR-ADMIN-ADA", nil)
	if adminDenied.Code != http.StatusForbidden {
		t.Fatalf("Admin manager-queue status=%d body=%s", adminDenied.Code, adminDenied.Body.String())
	}
	command := map[string]any{
		"operationId": "TASK6-HTTP-APPROVE", "idempotencyKey": "TASK6-HTTP-APPROVE",
		"candidateId": submitted.CandidateID, "expectedRevision": submitted.Revision,
		"expectedContentDigest": submitted.ContentDigest, "reason": "Canonical manager technical approval.",
	}
	approve := request(http.MethodPost,
		"/v1/department-manager/governed-checklist/candidates/"+submitted.CandidateID+"/technical-approvals",
		"USR-MANAGER-NORA", command)
	if approve.Code != http.StatusOK || !bytes.Contains(approve.Body.Bytes(), []byte(`"status":"TECHNICALLY_APPROVED"`)) {
		t.Fatalf("canonical approve status=%d body=%s", approve.Code, approve.Body.String())
	}
	var versionsBefore int
	if err := service.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM checklist_template_versions`).Scan(&versionsBefore); err != nil || versionsBefore != 0 {
		t.Fatalf("approval leaked publication versions=%d err=%v", versionsBefore, err)
	}
	command["operationId"] = "TASK6-HTTP-PUBLISH"
	command["idempotencyKey"] = "TASK6-HTTP-PUBLISH"
	command["reason"] = "Canonical manager publication after technical approval."
	publish := request(http.MethodPost,
		"/v1/department-manager/governed-checklist/candidates/"+submitted.CandidateID+"/publications",
		"USR-MANAGER-NORA", command)
	if publish.Code != http.StatusCreated ||
		!bytes.Contains(publish.Body.Bytes(), []byte(`"candidateId":"`+submitted.CandidateID+`"`)) ||
		!bytes.Contains(publish.Body.Bytes(), []byte(`"templateVersionId":"CTV-GOV-`)) {
		t.Fatalf("canonical publish status=%d body=%s", publish.Code, publish.Body.String())
	}
	var versionsAfter, decisionsAfter int
	if err := service.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM checklist_template_versions`).Scan(&versionsAfter); err != nil {
		t.Fatal(err)
	}
	if err := service.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM checklist_publication_decisions`).Scan(&decisionsAfter); err != nil {
		t.Fatal(err)
	}
	if versionsAfter != 1 || decisionsAfter != 1 {
		t.Fatalf("canonical publication versions=%d decisions=%d", versionsAfter, decisionsAfter)
	}
}

// Break caught: a candidate linked to an unresolved persisted source gap may
// remain visible for review, but it must not acquire an approval or Audit
// effect until the blocker is resolved through governed source facts.
func TestTask6TechnicalApprovalFailsClosedOnPersistedSourceGap(t *testing.T) {
	ctx := context.Background()
	service, manager, submitted := task6SubmittedCandidate(t, "task6_source_gap")
	if _, err := service.Pool.Exec(ctx, `
		INSERT INTO regulatory_source_gap_facts
			(id,regulatory_source_version_id,gap_id,reason,ordinal)
		VALUES ('GAP-TASK6-SYNTHETIC','SOURCE-SYNTHETIC-OPS-AOC',
			'TASK6_UNRESOLVED','A controlled source-owner confirmation is unresolved.',0)`); err != nil {
		t.Fatalf("insert persisted review blocker: %v", err)
	}
	item, err := service.GetReviewItem(ctx, manager, submitted.CandidateID)
	if err != nil || len(item.BlockingIssues) != 1 ||
		item.BlockingIssues[0].Code != "UNRESOLVED_SOURCE_GAP" {
		t.Fatalf("review blocker projection=%+v err=%v", item.BlockingIssues, err)
	}
	_, err = service.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "TASK6-BLOCKED-APPROVE", IdempotencyKey: "TASK6-BLOCKED-APPROVE",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest, Reason: "Must fail closed on the persisted source gap.",
	})
	var validation *regulatory.ValidationError
	if !errors.As(err, &validation) || len(validation.Issues) != 1 ||
		validation.Issues[0].Code != "UNRESOLVED_SOURCE_GAP" {
		t.Fatalf("blocked approval error=%T %+v", err, err)
	}
	assertTask6DecisionEffect(t, service, "TASK6-BLOCKED-APPROVE", "TECHNICALLY_APPROVED", 0)
}

func TestTask6SubmissionAndPublicationFailClosedOnPersistedSourceGap(t *testing.T) {
	ctx := context.Background()
	t.Run("submission", func(t *testing.T) {
		pool := createTestDatabase(t, "task6_blocked_submission")
		if err := migrations.Apply(ctx, pool); err != nil {
			t.Fatal(err)
		}
		if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO identity_references (subject_id,issuer,display_name)
			VALUES ('USR-TASK6-BLOCKED-ADMIN','task6-test','Blocked Admin')`); err != nil {
			t.Fatal(err)
		}
		admin := identity.Principal{SubjectID: "USR-TASK6-BLOCKED-ADMIN", Roles: []identity.Role{identity.RoleAdmin}}
		adminService := regulatory.NewAdminService(pool, nil)
		run, err := adminService.Import(ctx, admin, "TASK6-BLOCKED-SUBMIT-IMPORT", "TASK6-BLOCKED-SUBMIT-IMPORT", regulatory.SyntheticCandidateBundle())
		if err != nil || run.Candidate == nil {
			t.Fatalf("import before blocked submission: %+v err=%v", run, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO regulatory_source_gap_facts
				(id,regulatory_source_version_id,gap_id,reason,ordinal)
			VALUES ('GAP-TASK6-SUBMIT','SOURCE-SYNTHETIC-OPS-AOC',
				'TASK6_SUBMIT','Submission blocker.',0)`); err != nil {
			t.Fatal(err)
		}
		_, err = adminService.Submit(ctx, admin, regulatory.SubmitCommand{
			OperationID: "TASK6-BLOCKED-SUBMIT", IdempotencyKey: "TASK6-BLOCKED-SUBMIT",
			CandidateID: run.Candidate.CandidateID, ExpectedRevision: run.Candidate.Revision,
			ExpectedContentDigest: run.Candidate.ContentDigest, Reason: "Must fail closed before review.",
		})
		var validation *regulatory.ValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("blocked submission error=%T %v", err, err)
		}
		var status string
		if err := pool.QueryRow(ctx, `SELECT status FROM template_draft_versions WHERE id=$1`, run.Candidate.CandidateID).Scan(&status); err != nil || status != "GENERATED_DRAFT" {
			t.Fatalf("blocked submission status=%s err=%v", status, err)
		}
	})
	t.Run("publication", func(t *testing.T) {
		service, manager, submitted := task6SubmittedCandidate(t, "task6_blocked_publication")
		approved, err := service.Approve(ctx, manager, checklistgovernance.ReviewCommand{
			OperationID: "TASK6-BLOCKED-PUBLISH-APPROVE", IdempotencyKey: "TASK6-BLOCKED-PUBLISH-APPROVE",
			CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
			ExpectedContentDigest: submitted.ContentDigest, Reason: "Approve before a later source blocker.",
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.Pool.Exec(ctx, `
			INSERT INTO regulatory_source_gap_facts
				(id,regulatory_source_version_id,gap_id,reason,ordinal)
			VALUES ('GAP-TASK6-PUBLISH','SOURCE-SYNTHETIC-OPS-AOC',
				'TASK6_PUBLISH','Publication blocker.',0)`); err != nil {
			t.Fatal(err)
		}
		_, err = service.Publish(ctx, manager, checklistgovernance.PublicationCommand{
			OperationID: "TASK6-BLOCKED-PUBLISH", IdempotencyKey: "TASK6-BLOCKED-PUBLISH",
			CandidateID: approved.CandidateID, ExpectedRevision: approved.Revision,
			ExpectedContentDigest: approved.ContentDigest, Reason: "Must fail closed before publication.",
		})
		var validation *regulatory.ValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("blocked publication error=%T %v", err, err)
		}
		for _, table := range []string{"checklist_publication_decisions", "checklist_template_versions"} {
			var count int
			if err := service.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil || count != 0 {
				t.Fatalf("blocked publication left %s=%d err=%v", table, count, err)
			}
		}
	})
}

func TestTask6ConcurrentIdenticalApprovalReplaysOneExactEffect(t *testing.T) {
	ctx := context.Background()
	service, manager, submitted := task6SubmittedCandidate(t, "task6_concurrent_approval")
	command := checklistgovernance.ReviewCommand{
		OperationID: "TASK6-CONCURRENT-APPROVE", IdempotencyKey: "TASK6-CONCURRENT-APPROVE",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest, Reason: "One exact concurrent technical approval.",
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			_, err := service.Approve(ctx, manager, command)
			results <- err
		}()
	}
	close(start)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatalf("identical concurrent approval did not replay: %v", err)
		}
	}
	assertTask6DecisionEffect(t, service, command.OperationID, "TECHNICALLY_APPROVED", 1)
}

// Break caught: version 21 is already ledgered in existing databases, so
// lifecycle queue/concurrency indexes and status/guard definitions must be
// restored by the forward repair without rewriting governed history.
func TestTask6Migration21FreshInventoryAndForwardRepairPreserveHistory(t *testing.T) {
	ctx := context.Background()
	service, _, submitted := task6SubmittedCandidate(t, "task6_migration_repair")
	expectedIndexes := []string{
		"candidate_required_owner_assignments_review_queue_idx",
		"department_review_decisions_candidate_idx",
		"department_review_decisions_exact_owner_approval_idx",
		"checklist_publication_decisions_candidate_unique_idx",
		"checklist_template_versions_governed_candidate_unique_idx",
		"template_draft_versions_governed_review_queue_idx",
	}
	assertInventory := func(stage string) {
		t.Helper()
		for _, indexName := range expectedIndexes {
			var exists bool
			if err := service.Pool.QueryRow(ctx, `SELECT to_regclass('public.' || $1) IS NOT NULL`, indexName).Scan(&exists); err != nil || !exists {
				t.Fatalf("%s missing index %s exists=%v err=%v", stage, indexName, exists, err)
			}
		}
		var statusDefinition, guardDefinition string
		if err := service.Pool.QueryRow(ctx, `
			SELECT pg_get_constraintdef(oid)
			FROM pg_constraint
			WHERE conrelid='template_draft_versions'::regclass
			  AND conname='template_draft_versions_status_check'`).Scan(&statusDefinition); err != nil {
			t.Fatalf("%s read status constraint: %v", stage, err)
		}
		if err := service.Pool.QueryRow(ctx, `SELECT pg_get_functiondef('governed_generated_candidate_immutable_guard()'::regprocedure)`).Scan(&guardDefinition); err != nil {
			t.Fatalf("%s read immutable guard: %v", stage, err)
		}
		if !strings.Contains(statusDefinition, "PUBLISHED") ||
			!strings.Contains(guardDefinition, "TECHNICALLY_APPROVED") ||
			!strings.Contains(guardDefinition, "PUBLISHED") {
			t.Fatalf("%s definitions status=%s guard=%s", stage, statusDefinition, guardDefinition)
		}
	}
	assertInventory("fresh")

	var candidateCount, migrationCount int
	if err := service.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM template_draft_versions WHERE candidate_root_id=$1`, submitted.CandidateRootID).Scan(&candidateCount); err != nil {
		t.Fatal(err)
	}
	if err := service.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=21`).Scan(&migrationCount); err != nil {
		t.Fatal(err)
	}
	for _, indexName := range expectedIndexes {
		if _, err := service.Pool.Exec(ctx, "DROP INDEX "+indexName); err != nil {
			t.Fatalf("drop %s for repair simulation: %v", indexName, err)
		}
	}
	if _, err := service.Pool.Exec(ctx, `
		ALTER TABLE template_draft_versions DROP CONSTRAINT template_draft_versions_status_check;
		ALTER TABLE template_draft_versions ADD CONSTRAINT template_draft_versions_status_check
			CHECK (status IN ('DRAFT','GENERATED_DRAFT','DEPARTMENT_REVIEW','RETURNED','REJECTED','TECHNICALLY_APPROVED'));
		CREATE OR REPLACE FUNCTION governed_generated_candidate_immutable_guard()
		RETURNS trigger LANGUAGE plpgsql AS $broken$
		BEGIN RAISE EXCEPTION 'broken Task 6 guard'; END;
		$broken$`); err != nil {
		t.Fatalf("simulate ledgered migration drift: %v", err)
	}
	if err := migrations.Apply(ctx, service.Pool); err != nil {
		t.Fatalf("apply migration 21 forward repair: %v", err)
	}
	assertInventory("repaired")
	var candidateCountAfter, migrationCountAfter int
	if err := service.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM template_draft_versions WHERE candidate_root_id=$1`, submitted.CandidateRootID).Scan(&candidateCountAfter); err != nil {
		t.Fatal(err)
	}
	if err := service.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=21`).Scan(&migrationCountAfter); err != nil {
		t.Fatal(err)
	}
	if candidateCountAfter != candidateCount || migrationCountAfter != migrationCount || migrationCountAfter != 1 {
		t.Fatalf("repair rewrote history candidate=%d/%d migration=%d/%d", candidateCount, candidateCountAfter, migrationCount, migrationCountAfter)
	}
}

func assertTask6DecisionEffect(t *testing.T, service *checklistgovernance.Service, operationID, decision string, want int) {
	t.Helper()
	var decisions, audits int
	if err := service.Pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM department_review_decisions review
		JOIN template_draft_versions candidate ON candidate.id=review.candidate_draft_version_id
		WHERE review.operation_id=$1 AND review.decision=$2
		  AND review.candidate_root_id=candidate.candidate_root_id`,
		operationID, decision).Scan(&decisions); err != nil {
		t.Fatalf("count Task 6 decision: %v", err)
	}
	if err := service.Pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM audit_events WHERE operation_id=$1`, operationID).Scan(&audits); err != nil {
		t.Fatalf("count Task 6 Audit: %v", err)
	}
	if decisions != want || audits != want {
		t.Fatalf("operation %s decision=%d audit=%d want=%d", operationID, decisions, audits, want)
	}
}
