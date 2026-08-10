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
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/checklistgovernance"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/regulatory"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/testprofile"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

func task6FixRound2DecisionCounts(t *testing.T, service *checklistgovernance.Service, candidateID string) (int, int) {
	t.Helper()
	var decisions, audits int
	if err := service.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM department_review_decisions WHERE candidate_draft_version_id=$1`,
		candidateID,
	).Scan(&decisions); err != nil {
		t.Fatalf("count review decisions: %v", err)
	}
	if err := service.Pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_events
		 WHERE entity_type='GOVERNED_CANDIDATE' AND entity_id=$1
		   AND action='TECHNICAL_APPROVAL_RECORDED'`,
		candidateID,
	).Scan(&audits); err != nil {
		t.Fatalf("count lifecycle Audits: %v", err)
	}
	return decisions, audits
}

// Break caught: different required owners must serialize on the same root,
// retain both exact decisions/Audits, and only the second approval may advance
// the candidate to TECHNICALLY_APPROVED.
func TestTask6FixRound2DifferentOwnerApprovalsSerializeExactly(t *testing.T) {
	ctx := context.Background()
	service, foiManager, submitted := task6SubmittedCandidate(t, "task6_fix2_different_owner")
	if _, err := service.Pool.Exec(ctx, `
		INSERT INTO candidate_required_owner_assignments
			(id,candidate_draft_version_id,candidate_revision,candidate_content_digest,
			 department_id,organizational_unit_id,approval_required)
		VALUES ('OWNER-TASK6-FIX2-AIR',$1,$2,$3,
			'AIRWORTHINESS_INSPECTORATE','AIRWORTHINESS_INSPECTORATE',true)`,
		submitted.CandidateID, submitted.Revision, submitted.ContentDigest,
	); err != nil {
		t.Fatalf("add exact AIR owner: %v", err)
	}
	airManager := identity.Principal{
		SubjectID: "USR-TASK6-AIR-MANAGER",
		Roles:     []identity.Role{identity.RoleDepartmentManager},
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
	type result struct {
		candidate regulatory.CandidateView
		err       error
	}
	foiResult := make(chan result, 1)
	go func() {
		candidate, err := service.Approve(ctx, foiManager, checklistgovernance.ReviewCommand{
			OperationID: "TASK6-FIX2-FOI-APPROVE", IdempotencyKey: "TASK6-FIX2-FOI-APPROVE",
			CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
			ExpectedContentDigest: submitted.ContentDigest,
			Reason:                "FOI exact owner approval is queued first.",
		})
		foiResult <- result{candidate: candidate, err: err}
	}()
	waitForTask6AdvisoryWaiters(t, service, 1)
	airResult := make(chan result, 1)
	go func() {
		candidate, err := service.Approve(ctx, airManager, checklistgovernance.ReviewCommand{
			OperationID: "TASK6-FIX2-AIR-APPROVE", IdempotencyKey: "TASK6-FIX2-AIR-APPROVE",
			CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
			ExpectedContentDigest: submitted.ContentDigest,
			Reason:                "AIR exact owner approval is queued second.",
		})
		airResult <- result{candidate: candidate, err: err}
	}()
	waitForTask6AdvisoryWaiters(t, service, 2)
	if err := blocker.Commit(ctx); err != nil {
		t.Fatalf("release candidate root lock: %v", err)
	}
	first, second := <-foiResult, <-airResult
	if first.err != nil || first.candidate.Status != "DEPARTMENT_REVIEW" {
		t.Fatalf("first owner result=%+v err=%v", first.candidate, first.err)
	}
	if second.err != nil || second.candidate.Status != "TECHNICALLY_APPROVED" {
		t.Fatalf("second owner result=%+v err=%v", second.candidate, second.err)
	}
	decisions, audits := task6FixRound2DecisionCounts(t, service, submitted.CandidateID)
	if decisions != 2 || audits != 2 {
		t.Fatalf("joint approval decisions=%d Audits=%d, want 2/2", decisions, audits)
	}
}

// Break caught: return and reject waiting on one root must not both commit.
// The first waiter wins and the loser leaves no decision or Audit.
func TestTask6FixRound2ReturnWinsQueuedRejectWithExactEffects(t *testing.T) {
	ctx := context.Background()
	service, manager, submitted := task6SubmittedCandidate(t, "task6_fix2_return_reject")
	blocker, err := service.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(ctx)
	if _, err := blocker.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1,0))`,
		submitted.CandidateRootID,
	); err != nil {
		t.Fatal(err)
	}
	run := func(operation, decision string, output chan<- error) {
		command := checklistgovernance.ReviewCommand{
			OperationID: operation, IdempotencyKey: operation,
			CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
			ExpectedContentDigest: submitted.ContentDigest,
			Reason:                decision + " queued on the exact root.",
		}
		if decision == "RETURNED" {
			_, err := service.Return(ctx, manager, command)
			output <- err
			return
		}
		_, err := service.Reject(ctx, manager, command)
		output <- err
	}
	returned := make(chan error, 1)
	go run("TASK6-FIX2-RETURN-WINNER", "RETURNED", returned)
	waitForTask6AdvisoryWaiters(t, service, 1)
	rejected := make(chan error, 1)
	go run("TASK6-FIX2-REJECT-LOSER", "REJECTED", rejected)
	waitForTask6AdvisoryWaiters(t, service, 2)
	if err := blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-returned; err != nil {
		t.Fatalf("return winner: %v", err)
	}
	if err := <-rejected; !errors.Is(err, application.ErrConflict) {
		t.Fatalf("reject loser error=%v, want conflict", err)
	}
	detail, err := service.GetReviewItem(ctx, manager, submitted.CandidateID)
	if err != nil || detail.Candidate.Status != "RETURNED" ||
		len(detail.Decisions) != 1 || detail.Decisions[0].OperationID != "TASK6-FIX2-RETURN-WINNER" {
		t.Fatalf("terminal detail=%+v err=%v", detail, err)
	}
	assertTask6DecisionEffect(t, service, "TASK6-FIX2-RETURN-WINNER", "RETURNED", 1)
	assertTask6DecisionEffect(t, service, "TASK6-FIX2-REJECT-LOSER", "REJECTED", 0)
}

// Break caught: creating a same-root successor while an approval waits must
// make that approval stale before any decision or Audit write.
func TestTask6FixRound2SuccessorWinsQueuedStaleApproval(t *testing.T) {
	ctx := context.Background()
	service, manager, submitted := task6SubmittedCandidate(t, "task6_fix2_successor_approval")
	blocker, err := service.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(ctx)
	if _, err := blocker.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1,0))`,
		submitted.CandidateRootID,
	); err != nil {
		t.Fatal(err)
	}
	successorResult := make(chan error, 1)
	go func() {
		tx, err := service.Pool.Begin(ctx)
		if err != nil {
			successorResult <- err
			return
		}
		defer tx.Rollback(ctx)
		if _, err = tx.Exec(ctx,
			`SELECT pg_advisory_xact_lock(hashtextextended($1,0))`,
			submitted.CandidateRootID,
		); err == nil {
			_, err = tx.Exec(ctx, `
				INSERT INTO template_draft_versions
					(id,template_id,version,status,owner_role,creator_subject_id,change_reason,
					 question_version_ids,revision,generation_run_id,candidate_content_digest,
					 candidate_schema_version,candidate_root_id,supersedes_candidate_id)
				SELECT 'CAND-TASK6-FIX2-SUCCESSOR',template_id,version+1,'GENERATED_DRAFT',
					owner_role,creator_subject_id,'Concurrent exact-root successor.',
					question_version_ids,revision+1,generation_run_id,candidate_content_digest,
					candidate_schema_version,candidate_root_id,id
				FROM template_draft_versions WHERE id=$1`,
				submitted.CandidateID,
			)
		}
		if err == nil {
			err = tx.Commit(ctx)
		}
		successorResult <- err
	}()
	waitForTask6AdvisoryWaiters(t, service, 1)
	approvalResult := make(chan error, 1)
	go func() {
		_, err := service.Approve(ctx, manager, checklistgovernance.ReviewCommand{
			OperationID: "TASK6-FIX2-STALE-APPROVE", IdempotencyKey: "TASK6-FIX2-STALE-APPROVE",
			CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
			ExpectedContentDigest: submitted.ContentDigest,
			Reason:                "Approval must lose after a queued successor.",
		})
		approvalResult <- err
	}()
	waitForTask6AdvisoryWaiters(t, service, 2)
	if err := blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-successorResult; err != nil {
		t.Fatalf("create successor: %v", err)
	}
	if err := <-approvalResult; !errors.Is(err, application.ErrConflict) {
		t.Fatalf("stale approval error=%v, want conflict", err)
	}
	var leaf string
	if err := service.Pool.QueryRow(ctx, `
		SELECT candidate.id FROM template_draft_versions candidate
		WHERE candidate.candidate_root_id=$1
		  AND NOT EXISTS (
			SELECT 1 FROM template_draft_versions successor
			WHERE successor.supersedes_candidate_id=candidate.id
		  )`,
		submitted.CandidateRootID,
	).Scan(&leaf); err != nil || leaf != "CAND-TASK6-FIX2-SUCCESSOR" {
		t.Fatalf("current leaf=%s err=%v", leaf, err)
	}
	assertTask6DecisionEffect(t, service, "TASK6-FIX2-STALE-APPROVE", "TECHNICALLY_APPROVED", 0)
}

// Break caught: overlapping commands that share only one semantic identity
// must not both commit or replay as equal.
func TestTask6FixRound2ConflictingCommandIdentityInterleaving(t *testing.T) {
	ctx := context.Background()
	for _, shared := range []string{"operation", "idempotency"} {
		t.Run(shared, func(t *testing.T) {
			service, manager, submitted := task6SubmittedCandidate(t, "task6_fix2_conflict_"+shared)
			first := checklistgovernance.ReviewCommand{
				OperationID: "TASK6-FIX2-CONFLICT-OP-A", IdempotencyKey: "TASK6-FIX2-CONFLICT-IDEM-A",
				CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
				ExpectedContentDigest: submitted.ContentDigest, Reason: "First command wins.",
			}
			second := first
			second.OperationID = "TASK6-FIX2-CONFLICT-OP-B"
			second.IdempotencyKey = "TASK6-FIX2-CONFLICT-IDEM-B"
			second.Reason = "Changed command must conflict."
			if shared == "operation" {
				second.OperationID = first.OperationID
			} else {
				second.IdempotencyKey = first.IdempotencyKey
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
				t.Fatal(err)
			}
			firstResult := make(chan error, 1)
			go func() {
				_, err := service.Approve(ctx, manager, first)
				firstResult <- err
			}()
			waitForTask6AdvisoryWaiters(t, service, 1)
			secondResult := make(chan error, 1)
			go func() {
				_, err := service.Approve(ctx, manager, second)
				secondResult <- err
			}()
			waitForTask6AdvisoryWaiters(t, service, 2)
			if err := blocker.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			if err := <-firstResult; err != nil {
				t.Fatalf("first command: %v", err)
			}
			if err := <-secondResult; !errors.Is(err, application.ErrConflict) {
				t.Fatalf("second command error=%v, want conflict", err)
			}
			decisions, audits := task6FixRound2DecisionCounts(t, service, submitted.CandidateID)
			if decisions != 1 || audits != 1 {
				t.Fatalf("conflicting identity decisions=%d Audits=%d, want 1/1", decisions, audits)
			}
		})
	}
}

func task6FixRound2InvalidateAuthority(t *testing.T, service *checklistgovernance.Service, boundary string) {
	t.Helper()
	ctx := context.Background()
	switch boundary {
	case "inactive-membership":
		_, err := service.Pool.Exec(ctx, `
			INSERT INTO caa_department_memberships
				(id,root_id,supersedes_id,subject_id,department_id,organizational_unit_id,
				 membership_role,status,effective_from)
			VALUES ('MEM-TASK6-FIX2-REVOKED','MEM-TASK6-FOI','MEM-TASK6-FOI',
				'USR-TASK6-FOI-MANAGER','FLIGHT_OPERATIONS_INSPECTORATE',
				'FLIGHT_OPERATIONS_INSPECTORATE','DEPARTMENT_MANAGER','REVOKED','2026-01-01')`)
		if err != nil {
			t.Fatal(err)
		}
	case "expired-membership":
		_, err := service.Pool.Exec(ctx, `
			INSERT INTO caa_department_memberships
				(id,root_id,supersedes_id,subject_id,department_id,organizational_unit_id,
				 membership_role,status,effective_from,effective_to)
			VALUES ('MEM-TASK6-FIX2-EXPIRED','MEM-TASK6-FOI','MEM-TASK6-FOI',
				'USR-TASK6-FOI-MANAGER','FLIGHT_OPERATIONS_INSPECTORATE',
				'FLIGHT_OPERATIONS_INSPECTORATE','DEPARTMENT_MANAGER','ACTIVE',
				'2025-02-01','2026-01-01')`)
		if err != nil {
			t.Fatal(err)
		}
	case "inactive-department":
		_, err := service.Pool.Exec(ctx, `
			INSERT INTO caa_department_status_facts
				(id,root_id,supersedes_id,department_id,status,effective_from)
			VALUES ('STATUS-TASK6-FIX2-DEPT',
				'seed-department-status-FLIGHT_OPERATIONS_INSPECTORATE',
				'seed-department-status-FLIGHT_OPERATIONS_INSPECTORATE',
				'FLIGHT_OPERATIONS_INSPECTORATE','INACTIVE','2026-01-01')`)
		if err != nil {
			t.Fatal(err)
		}
	case "inactive-unit":
		_, err := service.Pool.Exec(ctx, `
			INSERT INTO caa_organizational_unit_status_facts
				(id,root_id,supersedes_id,organizational_unit_id,status,effective_from)
			VALUES ('STATUS-TASK6-FIX2-UNIT',
				'seed-unit-status-FLIGHT_OPERATIONS_INSPECTORATE',
				'seed-unit-status-FLIGHT_OPERATIONS_INSPECTORATE',
				'FLIGHT_OPERATIONS_INSPECTORATE','INACTIVE','2026-01-01')`)
		if err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unknown authority boundary %q", boundary)
	}
}

// Break caught: every current-authority invalidation must deny queue, detail,
// return, reject, approve, and publish without any decision or Audit effect.
func TestTask6FixRound2InactiveAndExpiredAuthorityDeniesEverySurface(t *testing.T) {
	ctx := context.Background()
	for _, boundary := range []string{"inactive-membership", "expired-membership", "inactive-department", "inactive-unit"} {
		t.Run(boundary, func(t *testing.T) {
			service, manager, candidate := task6SubmittedCandidate(
				t, "task6_fix2_authority_"+strings.ReplaceAll(boundary, "-", "_"),
			)
			task6FixRound2InvalidateAuthority(t, service, boundary)
			if queue, err := service.ListQueue(ctx, manager); !errors.Is(err, application.ErrForbidden) || len(queue) != 0 {
				t.Fatalf("queue=%+v err=%v, want forbidden empty", queue, err)
			}
			if _, err := service.GetReviewItem(ctx, manager, candidate.CandidateID); !errors.Is(err, application.ErrForbidden) {
				t.Fatalf("detail error=%v, want forbidden", err)
			}
			for _, operation := range []string{"return", "reject", "approve"} {
				command := checklistgovernance.ReviewCommand{
					OperationID:    fmt.Sprintf("TASK6-FIX2-%s-%s", boundary, operation),
					IdempotencyKey: fmt.Sprintf("TASK6-FIX2-%s-%s", boundary, operation),
					CandidateID:    candidate.CandidateID, ExpectedRevision: candidate.Revision,
					ExpectedContentDigest: candidate.ContentDigest,
					Reason:                "Invalid current authority must have zero effects.",
				}
				var err error
				switch operation {
				case "return":
					_, err = service.Return(ctx, manager, command)
				case "reject":
					_, err = service.Reject(ctx, manager, command)
				case "approve":
					_, err = service.Approve(ctx, manager, command)
				}
				if !errors.Is(err, application.ErrForbidden) {
					t.Fatalf("%s error=%v, want forbidden", operation, err)
				}
			}
			decisions, audits := task6FixRound2DecisionCounts(t, service, candidate.CandidateID)
			if decisions != 0 || audits != 0 {
				t.Fatalf("invalid authority decisions=%d Audits=%d, want 0/0", decisions, audits)
			}

			publishService, publishManager, publishCandidate := task6SubmittedCandidate(
				t, "task6_fix2_authority_publish_"+strings.ReplaceAll(boundary, "-", "_"),
			)
			approved, err := publishService.Approve(ctx, publishManager, checklistgovernance.ReviewCommand{
				OperationID:           "TASK6-FIX2-PREPUBLISH-" + boundary,
				IdempotencyKey:        "TASK6-FIX2-PREPUBLISH-" + boundary,
				CandidateID:           publishCandidate.CandidateID,
				ExpectedRevision:      publishCandidate.Revision,
				ExpectedContentDigest: publishCandidate.ContentDigest,
				Reason:                "Approve while exact authority is still current.",
			})
			if err != nil {
				t.Fatalf("approve before authority invalidation: %v", err)
			}
			task6FixRound2InvalidateAuthority(t, publishService, boundary)
			publishOperation := "TASK6-FIX2-" + boundary + "-publish"
			if _, err := publishService.Publish(ctx, publishManager, checklistgovernance.PublicationCommand{
				OperationID:           publishOperation,
				IdempotencyKey:        publishOperation,
				CandidateID:           approved.CandidateID,
				ExpectedRevision:      approved.Revision,
				ExpectedContentDigest: approved.ContentDigest,
				Reason:                "Invalid current authority must not publish.",
			}); !errors.Is(err, application.ErrForbidden) {
				t.Fatalf("publish error=%v, want forbidden", err)
			}
			var publicationDecisions, publicationAudits int
			if err := publishService.Pool.QueryRow(ctx,
				`SELECT COUNT(*) FROM checklist_publication_decisions WHERE operation_id=$1`,
				publishOperation,
			).Scan(&publicationDecisions); err != nil {
				t.Fatal(err)
			}
			if err := publishService.Pool.QueryRow(ctx,
				`SELECT COUNT(*) FROM audit_events WHERE operation_id=$1`,
				publishOperation,
			).Scan(&publicationAudits); err != nil {
				t.Fatal(err)
			}
			if publicationDecisions != 0 || publicationAudits != 0 {
				t.Fatalf("invalid publication authority decisions=%d Audits=%d, want 0/0", publicationDecisions, publicationAudits)
			}
		})
	}
}

func task6FixRound2ReverseMappings(t *testing.T) (*checklistgovernance.Service, identity.Principal, regulatory.CandidateView, []regulatory.ComplianceMapping) {
	t.Helper()
	ctx := context.Background()
	pool := createTestDatabase(t, "task6_fix2_reverse_mapping_order")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap synthetic inputs: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO identity_references(subject_id,issuer,display_name) VALUES
			('USR-TASK6-FIX2-ADMIN','task6-fix2','Reverse Admin'),
			('USR-TASK6-FIX2-MANAGER','task6-fix2','Reverse Manager');
		INSERT INTO caa_department_memberships
			(id,subject_id,department_id,organizational_unit_id,membership_role,status,effective_from)
		VALUES ('MEM-TASK6-FIX2-MANAGER','USR-TASK6-FIX2-MANAGER',
			'FLIGHT_OPERATIONS_INSPECTORATE','FLIGHT_OPERATIONS_INSPECTORATE',
			'DEPARTMENT_MANAGER','ACTIVE','2025-01-01')`); err != nil {
		t.Fatalf("seed reverse-order identities: %v", err)
	}
	bundle := regulatory.SyntheticCandidateBundle()
	first := bundle.ComplianceMappings[0]
	first.MappingID = "MAP-Z-FIRST"
	second := bundle.ComplianceMappings[0]
	second.MappingID = "MAP-A-SECOND"
	bundle.ComplianceMappings = []regulatory.ComplianceMapping{first, second}
	bundle.InspectionChecklist.Questions[0].MappingIDs = []string{first.MappingID, second.MappingID}
	digest, err := regulatory.CanonicalSHA256(map[string]any{
		"complianceMappings":  bundle.ComplianceMappings,
		"inspectionChecklist": bundle.InspectionChecklist,
	})
	if err != nil {
		t.Fatal(err)
	}
	bundle.OutputDigest = digest
	now := time.Date(2026, 7, 29, 19, 0, 0, 0, time.UTC)
	admin := identity.Principal{SubjectID: "USR-TASK6-FIX2-ADMIN", Roles: []identity.Role{identity.RoleAdmin}}
	adminService := regulatory.NewAdminService(pool, func() time.Time { return now })
	run, err := adminService.Import(ctx, admin, "TASK6-FIX2-REVERSE-IMPORT", "TASK6-FIX2-REVERSE-IMPORT", bundle)
	if err != nil || run.Candidate == nil {
		t.Fatalf("import reverse mappings: run=%+v err=%v", run, err)
	}
	submitted, err := adminService.Submit(ctx, admin, regulatory.SubmitCommand{
		OperationID: "TASK6-FIX2-REVERSE-SUBMIT", IdempotencyKey: "TASK6-FIX2-REVERSE-SUBMIT",
		CandidateID: run.Candidate.CandidateID, ExpectedRevision: run.Candidate.Revision,
		ExpectedContentDigest: run.Candidate.ContentDigest,
		Reason:                "Submit non-lexical mapping order.",
	})
	if err != nil {
		t.Fatalf("submit reverse mappings: %v", err)
	}
	manager := identity.Principal{SubjectID: "USR-TASK6-FIX2-MANAGER", Roles: []identity.Role{identity.RoleDepartmentManager}}
	return checklistgovernance.NewService(pool, func() time.Time { return now }), manager, submitted, bundle.ComplianceMappings
}

// Break caught: lexical mapping-id sorting must never replace the original
// candidate array order at import, projection, digest recomputation, or
// immutable publication.
func TestTask6FixRound2ReverseMappingOrderSurvivesPublicationBytes(t *testing.T) {
	ctx := context.Background()
	service, manager, submitted, expectedMappings := task6FixRound2ReverseMappings(t)
	if len(submitted.Mappings) != 2 ||
		submitted.Mappings[0].MappingID != "MAP-Z-FIRST" ||
		submitted.Mappings[1].MappingID != "MAP-A-SECOND" {
		t.Fatalf("candidate projection reordered mappings: %+v", submitted.Mappings)
	}
	approved, err := service.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "TASK6-FIX2-REVERSE-APPROVE", IdempotencyKey: "TASK6-FIX2-REVERSE-APPROVE",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest,
		Reason:                "Approve exact non-lexical mapping order.",
	})
	if err != nil {
		t.Fatalf("approve reverse mappings: %v", err)
	}
	published, err := service.Publish(ctx, manager, checklistgovernance.PublicationCommand{
		OperationID: "TASK6-FIX2-REVERSE-PUBLISH", IdempotencyKey: "TASK6-FIX2-REVERSE-PUBLISH",
		CandidateID: approved.CandidateID, ExpectedRevision: approved.Revision,
		ExpectedContentDigest: approved.ContentDigest,
		Reason:                "Publish exact non-lexical mapping order.",
	})
	if err != nil {
		t.Fatalf("publish reverse mappings: %v", err)
	}
	var raw []byte
	if err := service.Pool.QueryRow(ctx,
		`SELECT snapshot FROM checklist_template_versions WHERE id=$1`,
		published.TemplateVersionID,
	).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var snapshot struct {
		Mappings []regulatory.ComplianceMapping `json:"complianceMappings"`
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatal(err)
	}
	actual, _ := json.Marshal(snapshot.Mappings)
	expected, _ := json.Marshal(expectedMappings)
	if !bytes.Equal(actual, expected) {
		t.Fatalf("published mapping bytes reordered\nactual=%s\nwant=%s", actual, expected)
	}
	var ordinals []int
	if err := service.Pool.QueryRow(ctx, `
		SELECT array_agg(mapping_ordinal ORDER BY mapping_ordinal)
		FROM regulatory_generated_mapping_snapshots
		WHERE candidate_draft_version_id=$1`,
		submitted.CandidateID,
	).Scan(&ordinals); err != nil || len(ordinals) != 2 || ordinals[0] != 0 || ordinals[1] != 1 {
		t.Fatalf("persisted mapping ordinals=%v err=%v", ordinals, err)
	}
}

const task6FixRound4FrozenSuccessorMarker = "-- TASK 6 FIX ROUND 4 FROZEN EDITED SUCCESSOR DATA"
const task6FixRound4FrozenSuccessorEndMarker = "-- END TASK 6 FIX ROUND 4 FROZEN EDITED SUCCESSOR DATA"

func task6FixRound2ApplyFrozenV21(t *testing.T) *database.Pool {
	t.Helper()
	ctx := context.Background()
	pool := createTestDatabase(t, "task6_fix2_frozen_pretask6_v21")
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
		versionText := strings.TrimLeft(strings.SplitN(name, "_", 2)[0], "0")
		var version int
		if _, err := fmt.Sscanf(versionText, "%d", &version); err != nil {
			t.Fatalf("parse %s version: %v", name, err)
		}
		if version >= 21 {
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(raw)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations(version,name) VALUES ($1,$2)`,
			version, name,
		); err != nil {
			t.Fatalf("record %s: %v", name, err)
		}
	}
	fixturePath := filepath.Join(
		apiModuleRoot(t), "tests", "fixtures", "task6", "pre-task6-v21.sql",
	)
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read frozen pre-Task6 v21 fixture: %v", err)
	}
	fixtureParts := strings.SplitN(
		string(fixture), task6FixRound4FrozenSuccessorMarker, 2,
	)
	if len(fixtureParts) != 2 {
		t.Fatalf("frozen pre-Task6 v21 fixture is missing successor data marker")
	}
	fixtureTail := strings.SplitN(
		fixtureParts[1], task6FixRound4FrozenSuccessorEndMarker, 2,
	)
	if len(fixtureTail) != 2 {
		t.Fatalf("frozen pre-Task6 v21 fixture is missing successor end marker")
	}
	schemaFixture := fixtureParts[0] + fixtureTail[1]
	if _, err := pool.Exec(ctx, schemaFixture); err != nil {
		t.Fatalf("apply frozen pre-Task6 v21 fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO schema_migrations(version,name)
		VALUES (21,'000021_regulatory_checklist_governance.up.sql')`); err != nil {
		t.Fatalf("record frozen ledgered version 21: %v", err)
	}
	// The current synthetic fixture service records explicit source-currentness
	// facts introduced in migration 25. Apply only the independent 22-25
	// prerequisites here; the frozen v21 governance objects and history remain
	// untouched so migrations.Apply still exercises their forward repair.
	for _, file := range files {
		name := filepath.Base(file)
		versionText := strings.TrimLeft(strings.SplitN(name, "_", 2)[0], "0")
		var version int
		if _, err := fmt.Sscanf(versionText, "%d", &version); err != nil {
			t.Fatalf("parse %s version: %v", name, err)
		}
		if version < 22 || version > 25 {
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(raw)); err != nil {
			t.Fatalf("apply frozen-fixture prerequisite %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations(version,name) VALUES ($1,$2)`,
			version, name,
		); err != nil {
			t.Fatalf("record frozen-fixture prerequisite %s: %v", name, err)
		}
	}
	return pool
}

func task6FixRound2SeedFrozenHistory(t *testing.T, pool *database.Pool) (string, []regulatory.ComplianceMapping) {
	t.Helper()
	ctx := context.Background()
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap frozen history inputs: %v", err)
	}
	bundle := regulatory.SyntheticCandidateBundle()
	first := bundle.ComplianceMappings[0]
	first.MappingID = "MAP-Z-FROZEN-FIRST"
	second := bundle.ComplianceMappings[0]
	second.MappingID = "MAP-A-FROZEN-SECOND"
	bundle.ComplianceMappings = []regulatory.ComplianceMapping{first, second}
	bundle.InspectionChecklist.Questions[0].QuestionID = "Q-TASK6-FIX2-FROZEN"
	bundle.InspectionChecklist.Questions[0].MappingIDs = []string{first.MappingID, second.MappingID}
	bundle.CandidateBundleID = "CAND-TASK6-FIX2-FROZEN"
	bundle.GenerationRunID = "GENRUN-TASK6-FIX2-FROZEN"
	digest, err := regulatory.CanonicalSHA256(map[string]any{
		"complianceMappings":  bundle.ComplianceMappings,
		"inspectionChecklist": bundle.InspectionChecklist,
	})
	if err != nil {
		t.Fatal(err)
	}
	bundle.OutputDigest = digest
	requestBytes, _ := json.Marshal(bundle.GenerationRequest)
	var request map[string]any
	if err := json.Unmarshal(requestBytes, &request); err != nil {
		t.Fatal(err)
	}
	delete(request, "canonicalInputDigest")
	request["frozenPreTask6Fixture"] = true
	bundle.InputDigest, err = regulatory.CanonicalSHA256(request)
	if err != nil {
		t.Fatal(err)
	}
	requestBytes, _ = json.Marshal(request)
	outputBytes, _ := json.Marshal(map[string]any{
		"complianceMappings":  bundle.ComplianceMappings,
		"inspectionChecklist": bundle.InspectionChecklist,
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO identity_references(subject_id,issuer,display_name) VALUES
			('USR-TASK6-FIX2-FROZEN-ADMIN','task6-fix2','Frozen Admin'),
			('USR-TASK6-FIX2-FROZEN-MANAGER','task6-fix2','Frozen Manager');
		INSERT INTO caa_department_memberships
			(id,subject_id,department_id,organizational_unit_id,membership_role,status,effective_from)
		VALUES ('MEM-TASK6-FIX2-FROZEN','USR-TASK6-FIX2-FROZEN-MANAGER',
			'FLIGHT_OPERATIONS_INSPECTORATE','FLIGHT_OPERATIONS_INSPECTORATE',
			'DEPARTMENT_MANAGER','ACTIVE','2025-01-01')`); err != nil {
		t.Fatalf("seed frozen identities: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO regulatory_generation_runs
			(id,status,input_digest,output_digest,input_schema_version,generation_policy_version,
			 provider_catalog_version,provider_adapter_version,inspection_type,target_id,
			 input_artifact,output_artifact)
		VALUES ($1,'GENERATED',$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb,$11::jsonb)`,
		bundle.GenerationRunID, bundle.InputDigest, bundle.OutputDigest,
		bundle.GenerationRequest.SchemaVersion,
		bundle.GenerationRequest.GenerationPolicyVersion,
		bundle.GenerationRequest.ProviderCatalogVersion,
		bundle.GenerationRequest.ProviderVersion,
		bundle.GenerationRequest.InspectionType,
		bundle.GenerationRequest.Target.TargetID,
		string(requestBytes), string(outputBytes),
	); err != nil {
		t.Fatalf("seed frozen generation run: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO regulatory_generation_run_scope_facts
			(generation_run_id,organization_service_provider_scope_id,scope_root_id,
			 organization_id,service_provider_type_id,authorization_identifier,
			 scope_status,effective_from,effective_to,regulated_target_id)
		SELECT $1,id,root_id,organization_id,service_provider_type_id,
		       authorization_identifier,status,effective_from,effective_to,primary_target_id
		FROM organization_service_provider_scopes WHERE id=$2`,
		bundle.GenerationRunID,
		bundle.GenerationRequest.ServiceProviderScopeFactIDs[0],
	); err != nil {
		t.Fatalf("seed frozen scope fact: %v", err)
	}
	for _, source := range bundle.GenerationRequest.SourceSnapshots {
		for _, clauseID := range source.ClauseIDs {
			if _, err := pool.Exec(ctx, `
				INSERT INTO regulatory_generation_run_source_snapshots
					(generation_run_id,regulatory_source_version_id,
					 regulatory_normalized_clause_id,source_hash,clause_locator)
				SELECT $1,$2,id,$3,clause_locator
				FROM regulatory_normalized_clauses WHERE id=$4`,
				bundle.GenerationRunID, source.SourceSnapshotID, source.SourceHash, clauseID,
			); err != nil {
				t.Fatalf("seed frozen source snapshot: %v", err)
			}
		}
	}
	for _, stableRowID := range bundle.GenerationRequest.SecondaryCrosswalkPartition.StableRowIDs {
		if _, err := pool.Exec(ctx, `
			INSERT INTO regulatory_generation_run_crosswalk_partition_rows
				(generation_run_id,evaluation_partition_id,
				 state_compliance_crosswalk_row_id,stable_row_identity)
			SELECT $1,row.partition_id,row.state_compliance_crosswalk_row_id,row.stable_row_identity
			FROM regulatory_evaluation_partition_rows row
			WHERE row.partition_id=$2 AND row.stable_row_identity=$3`,
			bundle.GenerationRunID,
			bundle.GenerationRequest.SecondaryCrosswalkPartition.PartitionID,
			stableRowID,
		); err != nil {
			t.Fatalf("seed frozen partition row: %v", err)
		}
	}
	question := bundle.InspectionChecklist.Questions[0]
	questionVersionID := "QV-TASK6-FIX2-FROZEN"
	if _, err := pool.Exec(ctx, `
		INSERT INTO question_versions
			(id,question_id,version,prompt,configured_reference,expected_evidence,
			 created_by_subject_id)
		VALUES ($1,$2,1,$3,$4,$5,'SYNTHETIC-REGULATORY-GENERATOR')`,
		questionVersionID, question.QuestionID, question.Prompt,
		strings.Join(question.MappingIDs, ","),
		strings.Join(question.ExpectedEvidence, "; "),
	); err != nil {
		t.Fatalf("seed frozen question: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO template_masters(id,title,owner_role)
		VALUES ('TPL-TASK6-FIX2-FROZEN',$1,'Admin Preview')`,
		bundle.InspectionChecklist.ChecklistID,
	); err != nil {
		t.Fatalf("seed frozen template: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO template_draft_versions
			(id,template_id,version,status,owner_role,creator_subject_id,change_reason,
			 question_version_ids,revision,generation_run_id,candidate_content_digest,
			 candidate_schema_version,candidate_root_id)
		VALUES ($1,'TPL-TASK6-FIX2-FROZEN',1,'PUBLISHED','Admin Preview',
			'SYNTHETIC-REGULATORY-GENERATOR','Frozen pre-repair candidate.',
			ARRAY[$2],1,$3,$4,$5,$1)`,
		bundle.CandidateBundleID, questionVersionID, bundle.GenerationRunID,
		bundle.OutputDigest, bundle.SchemaVersion,
	); err != nil {
		t.Fatalf("seed frozen candidate: %v", err)
	}
	for _, mapping := range bundle.ComplianceMappings {
		raw, _ := json.Marshal(mapping)
		if _, err := pool.Exec(ctx, `
			INSERT INTO regulatory_generated_mapping_snapshots
				(candidate_draft_version_id,mapping_id,snapshot)
			VALUES ($1,$2,$3::jsonb)`,
			bundle.CandidateBundleID, mapping.MappingID, string(raw),
		); err != nil {
			t.Fatalf("seed frozen mapping: %v", err)
		}
	}
	questionRaw, _ := json.Marshal(question)
	if _, err := pool.Exec(ctx, `
		INSERT INTO regulatory_generated_question_snapshots
			(candidate_draft_version_id,question_id,snapshot)
		VALUES ($1,$2,$3::jsonb)`,
		bundle.CandidateBundleID, question.QuestionID, string(questionRaw),
	); err != nil {
		t.Fatalf("seed frozen question snapshot: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO candidate_required_owner_assignments
			(id,candidate_draft_version_id,candidate_revision,candidate_content_digest,
			 department_id,organizational_unit_id,approval_required)
		VALUES ('OWNER-TASK6-FIX2-FROZEN',$1,1,$2,
			'FLIGHT_OPERATIONS_INSPECTORATE','FLIGHT_OPERATIONS_INSPECTORATE',true)`,
		bundle.CandidateBundleID, bundle.OutputDigest,
	); err != nil {
		t.Fatalf("seed frozen owner: %v", err)
	}
	approvalSemantic := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	publicationSemantic := "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_events
			(event_id,occurred_at,actor_subject_id,actor_role,action,entity_type,
			 entity_id,entity_version,after_status,reason,operation_id,
			 correlation_id,request_id,details)
		VALUES
			('AE-TASK6-FIX2-FROZEN-COMMAND','2026-01-01T00:00:00Z',
			 'USR-TASK6-FIX2-FROZEN-ADMIN','admin','IMPORTED_GENERATION_RUN',
			 'GOVERNED_CANDIDATE',$1,1,'GENERATED_DRAFT','Frozen import.',
			 'TASK6-FIX2-FROZEN-COMMAND','TASK6-FIX2-FROZEN-COMMAND',
			 'TASK6-FIX2-FROZEN-COMMAND','{}'),
			('AE-TASK6-FIX2-FROZEN-APPROVE','2026-01-02T00:00:00Z',
			 'USR-TASK6-FIX2-FROZEN-MANAGER','manager','TECHNICAL_APPROVAL_RECORDED',
			 'GOVERNED_CANDIDATE',$1,1,'TECHNICALLY_APPROVED','Frozen approval.',
			 'TASK6-FIX2-FROZEN-APPROVE','TASK6-FIX2-FROZEN-APPROVE',
			 'TASK6-FIX2-FROZEN-APPROVE','{}'),
			('AE-TASK6-FIX2-FROZEN-PUBLISH','2026-01-03T00:00:00Z',
			 'USR-TASK6-FIX2-FROZEN-MANAGER','manager','CHECKLIST_PUBLISHED',
			 'GOVERNED_CANDIDATE',$1,1,'PUBLISHED','Frozen publication.',
			 'TASK6-FIX2-FROZEN-PUBLISH','TASK6-FIX2-FROZEN-PUBLISH',
			 'TASK6-FIX2-FROZEN-PUBLISH','{}')`,
		bundle.CandidateBundleID,
	); err != nil {
		t.Fatalf("seed frozen Audits: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO governed_candidate_commands
			(id,command_kind,operation_id,idempotency_key,semantic_payload_digest,
			 generation_run_id,candidate_draft_version_id,candidate_revision,
			 candidate_content_digest,actor_subject_id,reason,audit_event_id)
		VALUES ('CMD-TASK6-FIX2-FROZEN','IMPORTED_GENERATION_RUN',
			'TASK6-FIX2-FROZEN-COMMAND','TASK6-FIX2-FROZEN-COMMAND',
			'sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc',
			$1,$2,1,$3,'USR-TASK6-FIX2-FROZEN-ADMIN','Frozen import.',
			'AE-TASK6-FIX2-FROZEN-COMMAND')`,
		bundle.GenerationRunID, bundle.CandidateBundleID, bundle.OutputDigest,
	); err != nil {
		t.Fatalf("seed frozen command: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO department_review_decisions
			(id,candidate_draft_version_id,candidate_revision,candidate_content_digest,
			 decision,actor_subject_id,actor_department_membership_id,
			 actor_department_id,actor_organizational_unit_id,reason,decided_at,
			 operation_id,idempotency_key,semantic_payload_digest)
		VALUES ('DRD-TASK6-FIX2-FROZEN',$1,1,$2,'TECHNICALLY_APPROVED',
			'USR-TASK6-FIX2-FROZEN-MANAGER','MEM-TASK6-FIX2-FROZEN',
			'FLIGHT_OPERATIONS_INSPECTORATE','FLIGHT_OPERATIONS_INSPECTORATE',
			'Frozen approval.','2026-01-02T00:00:00Z',
			'TASK6-FIX2-FROZEN-APPROVE','TASK6-FIX2-FROZEN-APPROVE',$3)`,
		bundle.CandidateBundleID, bundle.OutputDigest, approvalSemantic,
	); err != nil {
		t.Fatalf("seed frozen review decision: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO checklist_publication_decisions
			(id,candidate_draft_version_id,candidate_revision,candidate_content_digest,
			 actor_subject_id,actor_department_membership_id,actor_department_id,
			 actor_organizational_unit_id,reason,decided_at,operation_id,
			 idempotency_key,semantic_payload_digest)
		VALUES ('PUBDEC-TASK6-FIX2-FROZEN',$1,1,$2,
			'USR-TASK6-FIX2-FROZEN-MANAGER','MEM-TASK6-FIX2-FROZEN',
			'FLIGHT_OPERATIONS_INSPECTORATE','FLIGHT_OPERATIONS_INSPECTORATE',
			'Frozen publication.','2026-01-03T00:00:00Z',
			'TASK6-FIX2-FROZEN-PUBLISH','TASK6-FIX2-FROZEN-PUBLISH',$3)`,
		bundle.CandidateBundleID, bundle.OutputDigest, publicationSemantic,
	); err != nil {
		t.Fatalf("seed frozen publication decision: %v", err)
	}
	snapshot, _ := json.Marshal(map[string]any{
		"candidateId":            bundle.CandidateBundleID,
		"candidateRevision":      1,
		"candidateContentDigest": bundle.OutputDigest,
		"complianceMappings":     bundle.ComplianceMappings,
		"questions":              bundle.InspectionChecklist.Questions,
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO checklist_template_versions
			(id,template_id,version,title,snapshot,published_at,
			 candidate_draft_version_id,candidate_revision,candidate_content_digest,
			 publication_decision_id)
		VALUES ('CTV-TASK6-FIX2-FROZEN','TPL-TASK6-FIX2-FROZEN',1,
			'Frozen published template',$1::jsonb,'2026-01-03T00:00:00Z',
			$2,1,$3,'PUBDEC-TASK6-FIX2-FROZEN')`,
		string(snapshot), bundle.CandidateBundleID, bundle.OutputDigest,
	); err != nil {
		t.Fatalf("seed frozen template version: %v", err)
	}
	return bundle.CandidateBundleID, bundle.ComplianceMappings
}

func task6FixRound2HistorySnapshot(t *testing.T, pool *database.Pool, candidateID string) map[string]string {
	t.Helper()
	queries := map[string]string{
		"source": `SELECT to_jsonb(row)::text FROM regulatory_source_versions row
			WHERE id='SOURCE-SYNTHETIC-OPS-AOC'`,
		"candidate": `SELECT (to_jsonb(row) - ARRAY[
			'entry_path','lineage_kind','owner_resolution_digest','blocker_digest',
			'existing_candidate_id','governed_source_binding_set_id',
			'legacy_authority_state','creation_basis'
		])::text FROM template_draft_versions row
			WHERE id='CAND-TASK6-FIX2-FROZEN'`,
		"review": `SELECT (to_jsonb(row)-'candidate_root_id')::text
			FROM department_review_decisions row WHERE id='DRD-TASK6-FIX2-FROZEN'`,
		"publication": `SELECT (to_jsonb(row)-'candidate_root_id')::text
			FROM checklist_publication_decisions row WHERE id='PUBDEC-TASK6-FIX2-FROZEN'`,
		"template": `SELECT to_jsonb(row)::text FROM checklist_template_versions row
			WHERE id='CTV-TASK6-FIX2-FROZEN'`,
		"audit": `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY event_id),'[]')::text
			FROM audit_events row WHERE entity_id='CAND-TASK6-FIX2-FROZEN'`,
		"command": `SELECT to_jsonb(row)::text FROM governed_candidate_commands row
			WHERE id='CMD-TASK6-FIX2-FROZEN'`,
		"mapping": `SELECT COALESCE(
				jsonb_agg(to_jsonb(row)-'mapping_ordinal' ORDER BY mapping_id),'[]'
			)::text FROM regulatory_generated_mapping_snapshots row
			WHERE candidate_draft_version_id='CAND-TASK6-FIX2-FROZEN'`,
	}
	output := map[string]string{}
	for label, query := range queries {
		var snapshot string
		if err := pool.QueryRow(context.Background(), query).Scan(&snapshot); err != nil {
			t.Fatalf("capture %s history for %s: %v", label, candidateID, err)
		}
		output[label] = snapshot
	}
	return output
}

// Break caught: an upgrade proof derived from the mutable current migration
// can silently inherit Task 6 definitions and cannot prove repair of an
// already-ledgered pre-Task6 v21 database with real history.
func TestTask6FixRound2FrozenPreTask6V21RepairsCompleteInventoryAndHistory(t *testing.T) {
	ctx := context.Background()
	pool := task6FixRound2ApplyFrozenV21(t)
	var rootColumns, ordinalColumns int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema='public' AND (
			(table_name IN ('department_review_decisions','checklist_publication_decisions')
			 AND column_name='candidate_root_id')
			OR (table_name='regulatory_generated_mapping_snapshots'
			 AND column_name='mapping_ordinal')
		)`).Scan(&rootColumns); err != nil || rootColumns != 0 {
		t.Fatalf("frozen fixture already contains Task 6 repair columns=%d err=%v", rootColumns, err)
	}
	candidateID, expectedMappings := task6FixRound2SeedFrozenHistory(t, pool)
	before := task6FixRound2HistorySnapshot(t, pool, candidateID)
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("repair frozen pre-Task6 v21: %v", err)
	}

	expectedColumns := []struct {
		table, column, dataType, nullable string
	}{
		{"department_review_decisions", "candidate_root_id", "text", "NO"},
		{"checklist_publication_decisions", "candidate_root_id", "text", "NO"},
		{"regulatory_generated_mapping_snapshots", "mapping_ordinal", "integer", "NO"},
		{"candidate_required_owner_assignments", "approval_required", "boolean", "NO"},
		{"checklist_template_versions", "publication_decision_id", "text", "YES"},
		{"template_draft_versions", "candidate_content_digest", "text", "YES"},
	}
	for _, expected := range expectedColumns {
		var dataType, nullable string
		if err := pool.QueryRow(ctx, `
			SELECT data_type,is_nullable FROM information_schema.columns
			WHERE table_schema='public' AND table_name=$1 AND column_name=$2`,
			expected.table, expected.column,
		).Scan(&dataType, &nullable); err != nil ||
			dataType != expected.dataType || nullable != expected.nullable {
			t.Fatalf("column %s.%s=%s/%s err=%v, want %s/%s",
				expected.table, expected.column, dataType, nullable, err,
				expected.dataType, expected.nullable,
			)
		}
	}
	expectedConstraints := map[string]string{
		"department_review_decisions_candidate_root_fkey":                 "FOREIGN KEY (candidate_root_id) REFERENCES template_draft_versions(id)",
		"checklist_publication_decisions_candidate_root_fkey":             "FOREIGN KEY (candidate_root_id) REFERENCES template_draft_versions(id)",
		"regulatory_generated_mapping_snapshots_mapping_ordinal_check":    "CHECK ((mapping_ordinal >= 0))",
		"regulatory_generated_mapping_snapshots_candidate_ordinal_unique": "UNIQUE (candidate_draft_version_id, mapping_ordinal)",
	}
	for name, expected := range expectedConstraints {
		var definition string
		if err := pool.QueryRow(ctx,
			`SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname=$1`,
			name,
		).Scan(&definition); err != nil || definition != expected {
			t.Fatalf("constraint %s=%q err=%v, want %q", name, definition, err, expected)
		}
	}
	expectedIndexes := []string{
		"candidate_required_owner_assignments_review_queue_idx",
		"department_review_decisions_candidate_idx",
		"department_review_decisions_exact_owner_approval_idx",
		"checklist_publication_decisions_candidate_unique_idx",
		"checklist_template_versions_governed_candidate_unique_idx",
		"template_draft_versions_governed_review_queue_idx",
		"regulatory_generated_mapping_snapshots_candidate_ordinal_unique",
	}
	for _, name := range expectedIndexes {
		var definition string
		if err := pool.QueryRow(ctx,
			`SELECT pg_get_indexdef(to_regclass('public.' || $1))`, name,
		).Scan(&definition); err != nil || !strings.Contains(definition, name) {
			t.Fatalf("index %s definition=%q err=%v", name, definition, err)
		}
	}
	expectedTriggers := map[string][]string{
		"department_review_decisions_actor_guard":                {"BEFORE", "INSERT", "validate_governed_decision_actor"},
		"checklist_publication_decisions_actor_guard":            {"BEFORE", "INSERT", "validate_governed_decision_actor"},
		"checklist_publication_decisions_approval_guard":         {"BEFORE", "INSERT", "validate_governed_publication_approval"},
		"checklist_template_versions_governed_publication_guard": {"BEFORE", "INSERT", "validate_governed_published_template"},
		"template_draft_versions_generated_immutable":            {"BEFORE", "UPDATE", "governed_generated_candidate_immutable_guard"},
		"department_review_decisions_append_only":                {"BEFORE", "UPDATE", "governed_append_only_guard"},
		"checklist_publication_decisions_append_only":            {"BEFORE", "UPDATE", "governed_append_only_guard"},
	}
	for name, expected := range expectedTriggers {
		var timing, events, function string
		if err := pool.QueryRow(ctx, `
			SELECT CASE WHEN (tgtype & 2)=2 THEN 'BEFORE' ELSE 'AFTER' END,
			       concat_ws(',',CASE WHEN (tgtype & 4)=4 THEN 'INSERT' END,
			                    CASE WHEN (tgtype & 16)=16 THEN 'UPDATE' END,
			                    CASE WHEN (tgtype & 8)=8 THEN 'DELETE' END),
			       p.proname
			FROM pg_trigger trigger
			JOIN pg_proc p ON p.oid=trigger.tgfoid
			WHERE trigger.tgname=$1 AND NOT trigger.tgisinternal`,
			name,
		).Scan(&timing, &events, &function); err != nil ||
			timing != expected[0] || !strings.Contains(events, expected[1]) ||
			function != expected[2] {
			t.Fatalf("trigger %s=%s/%s/%s err=%v want=%v",
				name, timing, events, function, err, expected,
			)
		}
	}
	expectedFunctions := map[string][]string{
		"validate_governed_decision_actor": {
			"effective_membership", "department_status", "unit_status",
			"candidate_root_id",
		},
		"validate_governed_publication_approval": {
			"candidate_required_owner_assignments", "TECHNICALLY_APPROVED",
		},
		"validate_governed_published_template": {
			"publication_decision_id", "candidate_content_digest",
		},
		"governed_generated_candidate_immutable_guard": {
			"DEPARTMENT_REVIEW", "TECHNICALLY_APPROVED", "PUBLISHED",
		},
		"validate_governed_generated_candidate": {
			"candidate_root_id", "supersedes_candidate_id", "output_digest",
		},
	}
	for name, phrases := range expectedFunctions {
		var definition string
		if err := pool.QueryRow(ctx,
			`SELECT pg_get_functiondef(($1 || '()')::regprocedure)`,
			name,
		).Scan(&definition); err != nil {
			t.Fatalf("read function %s: %v", name, err)
		}
		for _, phrase := range phrases {
			if !strings.Contains(definition, phrase) {
				t.Fatalf("function %s missing %q:\n%s", name, phrase, definition)
			}
		}
	}
	var reviewRoot, publicationRoot string
	if err := pool.QueryRow(ctx,
		`SELECT candidate_root_id FROM department_review_decisions
		 WHERE id='DRD-TASK6-FIX2-FROZEN'`,
	).Scan(&reviewRoot); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT candidate_root_id FROM checklist_publication_decisions
		 WHERE id='PUBDEC-TASK6-FIX2-FROZEN'`,
	).Scan(&publicationRoot); err != nil {
		t.Fatal(err)
	}
	if reviewRoot != candidateID || publicationRoot != candidateID {
		t.Fatalf("decision root backfill review=%s publication=%s want=%s",
			reviewRoot, publicationRoot, candidateID,
		)
	}
	var mappingIDs []string
	if err := pool.QueryRow(ctx, `
		SELECT array_agg(mapping_id ORDER BY mapping_ordinal)
		FROM regulatory_generated_mapping_snapshots
		WHERE candidate_draft_version_id=$1`,
		candidateID,
	).Scan(&mappingIDs); err != nil ||
		len(mappingIDs) != 2 ||
		mappingIDs[0] != expectedMappings[0].MappingID ||
		mappingIDs[1] != expectedMappings[1].MappingID {
		t.Fatalf("mapping backfill order=%v err=%v want=%s,%s",
			mappingIDs, err, expectedMappings[0].MappingID, expectedMappings[1].MappingID,
		)
	}
	after := task6FixRound2HistorySnapshot(t, pool, candidateID)
	for label, expected := range before {
		if after[label] != expected {
			t.Fatalf("first repair changed frozen %s history\nbefore=%s\nafter=%s",
				label, expected, after[label],
			)
		}
	}
	now := time.Date(2026, 7, 29, 20, 0, 0, 0, time.UTC)
	admin := identity.Principal{
		SubjectID: "USR-TASK6-FIX2-FROZEN-ADMIN",
		Roles:     []identity.Role{identity.RoleAdmin},
	}
	adminService := regulatory.NewAdminService(pool, func() time.Time { return now })
	run, err := adminService.Import(
		ctx, admin, "TASK6-FIX2-REPAIRED-IMPORT", "TASK6-FIX2-REPAIRED-IMPORT",
		regulatory.SyntheticCandidateBundle(),
	)
	if err != nil || run.Candidate == nil {
		t.Fatalf("post-repair import: run=%+v err=%v", run, err)
	}
	submitted, err := adminService.Submit(ctx, admin, regulatory.SubmitCommand{
		OperationID:           "TASK6-FIX2-REPAIRED-SUBMIT",
		IdempotencyKey:        "TASK6-FIX2-REPAIRED-SUBMIT",
		CandidateID:           run.Candidate.CandidateID,
		ExpectedRevision:      run.Candidate.Revision,
		ExpectedContentDigest: run.Candidate.ContentDigest,
		Reason:                "Exercise the full repaired lifecycle.",
	})
	if err != nil {
		t.Fatalf("post-repair submit: %v", err)
	}
	manager := identity.Principal{
		SubjectID: "USR-TASK6-FIX2-FROZEN-MANAGER",
		Roles:     []identity.Role{identity.RoleDepartmentManager},
	}
	lifecycle := checklistgovernance.NewService(pool, func() time.Time { return now })
	approved, err := lifecycle.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID:           "TASK6-FIX2-REPAIRED-APPROVE",
		IdempotencyKey:        "TASK6-FIX2-REPAIRED-APPROVE",
		CandidateID:           submitted.CandidateID,
		ExpectedRevision:      submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest,
		Reason:                "Approve the exact repaired candidate.",
	})
	if err != nil {
		t.Fatalf("post-repair approve: %v", err)
	}
	if _, err := lifecycle.Publish(ctx, manager, checklistgovernance.PublicationCommand{
		OperationID:           "TASK6-FIX2-REPAIRED-PUBLISH",
		IdempotencyKey:        "TASK6-FIX2-REPAIRED-PUBLISH",
		CandidateID:           approved.CandidateID,
		ExpectedRevision:      approved.Revision,
		ExpectedContentDigest: approved.ContentDigest,
		Reason:                "Publish the exact repaired candidate separately.",
	}); err != nil {
		t.Fatalf("post-repair publish: %v", err)
	}
	fullHistoryQueries := map[string]string{
		"source": `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]')::text
			FROM regulatory_source_versions row`,
		"candidate": `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]')::text
			FROM template_draft_versions row`,
		"review": `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]')::text
			FROM department_review_decisions row`,
		"publication": `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]')::text
			FROM checklist_publication_decisions row`,
		"template": `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]')::text
			FROM checklist_template_versions row`,
		"audit": `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]')::text
			FROM audit_events row`,
		"command": `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]')::text
			FROM governed_candidate_commands row`,
		"mapping": `SELECT COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text),'[]')::text
			FROM regulatory_generated_mapping_snapshots row`,
	}
	beforeSecondRepair := map[string]string{}
	for label, query := range fullHistoryQueries {
		var snapshot string
		if err := pool.QueryRow(ctx, query).Scan(&snapshot); err != nil {
			t.Fatalf("capture complete %s history: %v", label, err)
		}
		beforeSecondRepair[label] = snapshot
	}
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("second idempotent repair: %v", err)
	}
	afterSecond := task6FixRound2HistorySnapshot(t, pool, candidateID)
	for label, expected := range before {
		if afterSecond[label] != expected {
			t.Fatalf("second repair changed frozen %s history", label)
		}
	}
	for label, query := range fullHistoryQueries {
		var snapshot string
		if err := pool.QueryRow(ctx, query).Scan(&snapshot); err != nil ||
			snapshot != beforeSecondRepair[label] {
			t.Fatalf("second repair changed complete %s history err=%v", label, err)
		}
	}
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM schema_migrations WHERE version=21`,
	).Scan(&ordinalColumns); err != nil || ordinalColumns != 1 {
		t.Fatalf("ledger version 21 rows=%d err=%v", ordinalColumns, err)
	}
}

func TestTask6FixRound2CanonicalBlockedGenerationOperationUsesExactValidator(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "task6_fix2_blocked_operation")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := testprofile.BootstrapBlockedRealOPSAOCGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap blocked real OPS/AOC inputs: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO identity_references(subject_id,issuer,display_name)
		VALUES ('USR-MANAGER-NORA','task6-fix2','Blocked-real Manager')
		ON CONFLICT (subject_id) DO NOTHING;
		INSERT INTO caa_department_memberships
			(id,subject_id,department_id,organizational_unit_id,membership_role,status,effective_from)
		VALUES ('MEM-TASK6-FIX2-BLOCKED','USR-MANAGER-NORA',
			'FLIGHT_OPERATIONS_INSPECTORATE','FLIGHT_OPERATIONS_INSPECTORATE',
			'DEPARTMENT_MANAGER','ACTIVE','2025-01-01')`); err != nil {
		t.Fatalf("seed blocked-real manager: %v", err)
	}

	exact := regulatory.RealOPSAOCGenerationRequest().Request()
	if exact.CanonicalInputDigest != "sha256:e0231ec606fc59adfe5a425674bc4b3130b92e1df70df22ee666cc4ba7d91583" {
		t.Fatalf("exact blocked request digest=%s", exact.CanonicalInputDigest)
	}
	wantGapIDs := []string{
		"CONTROLLED_PROCEDURE",
		"PART_140_AUTHORITY",
		"PART_127_APPLICABILITY",
		"AMBIGUOUS_OWNERSHIP",
	}
	gotGapIDs := make([]string, 0, len(exact.UnresolvedSourceGaps))
	for _, gap := range exact.UnresolvedSourceGaps {
		gotGapIDs = append(gotGapIDs, gap.GapID)
	}
	if !reflect.DeepEqual(gotGapIDs, wantGapIDs) {
		t.Fatalf("exact blocked request gaps=%v, want %v", gotGapIDs, wantGapIDs)
	}

	now := time.Date(2026, 7, 29, 16, 30, 0, 0, time.UTC)
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
		Pool: pool, Clock: func() time.Time { return now },
	})
	handler := httpapi.NewCanonicalTestBoundary("task6-fix2-blocked-token").Protect(api.Handler())
	encoded, err := json.Marshal(map[string]any{
		"operationId":       "TASK6-FIX2-BLOCKED-VALIDATE",
		"idempotencyKey":    "TASK6-FIX2-BLOCKED-VALIDATE",
		"generationRequest": exact,
	})
	if err != nil {
		t.Fatalf("marshal exact blocked request: %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/department-manager/governed-checklist/blocked-generation-validations",
		bytes.NewReader(encoded),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(httpapi.CanonicalTestTokenHeader, "task6-fix2-blocked-token")
	request.Header.Set(httpapi.CanonicalTestSubjectHeader, "USR-MANAGER-NORA")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("blocked-generation status=%d body=%s", response.Code, response.Body.String())
	}
	var output struct {
		Status         string                           `json:"status"`
		RequestID      string                           `json:"requestId"`
		BlockingIssues []regulatory.UnresolvedSourceGap `json:"blockingIssues"`
		EffectCounts   map[string]int                   `json:"effectCounts"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &output); err != nil {
		t.Fatalf("decode blocked-generation result: %v", err)
	}
	if output.Status != "BLOCKED" || output.RequestID != exact.RequestID ||
		!reflect.DeepEqual(output.BlockingIssues, exact.UnresolvedSourceGaps) {
		t.Fatalf("blocked-generation output=%+v", output)
	}
	wantCounts := map[string]int{
		"generationRuns": 0, "candidates": 0, "reviewDecisions": 0,
		"publicationDecisions": 0, "checklistVersions": 0, "auditEvents": 0,
	}
	if !reflect.DeepEqual(output.EffectCounts, wantCounts) {
		t.Fatalf("blocked-generation effect counts=%v, want %v", output.EffectCounts, wantCounts)
	}
	for label, query := range map[string]string{
		"run":         `SELECT COUNT(*) FROM regulatory_generation_runs WHERE input_artifact->>'requestId'='GENREQ-OPS-AOC-0001'`,
		"candidate":   `SELECT COUNT(*) FROM template_draft_versions WHERE id='CAND-REAL-OPS-AOC-BLOCKED'`,
		"review":      `SELECT COUNT(*) FROM department_review_decisions WHERE operation_id LIKE 'TASK6-FIX2-BLOCKED-%'`,
		"publication": `SELECT COUNT(*) FROM checklist_publication_decisions WHERE operation_id LIKE 'TASK6-FIX2-BLOCKED-%'`,
		"version":     `SELECT COUNT(*) FROM checklist_template_versions WHERE candidate_draft_version_id='CAND-REAL-OPS-AOC-BLOCKED'`,
		"audit":       `SELECT COUNT(*) FROM audit_events WHERE operation_id LIKE 'TASK6-FIX2-BLOCKED-%'`,
	} {
		var count int
		if err := pool.QueryRow(ctx, query).Scan(&count); err != nil || count != 0 {
			t.Fatalf("blocked-real %s effects=%d err=%v", label, count, err)
		}
	}
}
