//go:build canonicaltest

package integration_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/checklistgovernance"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/regulatory"
)

func TestTask8SourceImpactProjectionPinsAffectedImmutableLineage(t *testing.T) {
	ctx := context.Background()
	governance, _, submitted := task6SubmittedCandidate(t, "task8_source_impact")
	admin := regulatory.NewAdminService(governance.Pool, governance.Clock)
	impact, err := admin.GetSourceImpact(ctx, identity.Principal{SubjectID: "USR-TASK6-ADMIN", Roles: []identity.Role{identity.RoleAdmin}}, "SOURCE-SYNTHETIC-OPS-AOC")
	if err != nil || impact.SourceHash != submitted.SourceSnapshots[0].SourceHash || len(impact.ClauseIDs) != 2 || impact.ClauseIDs[0] != "CLAUSE-SYNTHETIC-OPS-AOC-1" || impact.ClauseIDs[1] != "CLAUSE-SYNTHETIC-OPS-AOC-HOLDOUT-1" || len(impact.CandidateIDs) != 1 || impact.CandidateIDs[0] != submitted.CandidateID || len(impact.MappingIDs) != 1 || len(impact.QuestionIDs) != 1 || len(impact.ScopeFactIDs) != 1 {
		t.Fatalf("source impact=%+v err=%v", impact, err)
	}
}

func TestTask8RunHoldoutEvaluationUsesOnlyReservedPartitionAndRejectsInputIdentity(t *testing.T) {
	ctx := context.Background()
	governance, _, submitted := task6SubmittedCandidate(t, "task8_holdout")
	admin := regulatory.NewAdminService(governance.Pool, governance.Clock)
	actor := identity.Principal{SubjectID: "USR-TASK6-ADMIN", Roles: []identity.Role{identity.RoleAdmin}}
	result, err := admin.EvaluateGenerationRunHoldout(ctx, actor, submitted.GenerationRunID, []regulatory.HoldoutReview{{StableRowIdentity: "CC:SYNTHETIC:OPS:AOC:HOLDOUT:1", Outcome: regulatory.HoldoutSupported}})
	if err != nil || result.ReviewedCount != 1 || result.SupportedCount != 1 {
		t.Fatalf("holdout result=%+v err=%v", result, err)
	}
	var holdoutAudit []byte
	if err := governance.Pool.QueryRow(ctx, `SELECT details::text FROM audit_events WHERE entity_type='REGULATORY_GENERATION_RUN' AND entity_id=$1 AND action='regulatory.blind_holdout_evaluated'`, submitted.GenerationRunID).Scan(&holdoutAudit); err != nil || !containsAll(string(holdoutAudit), "CC:SYNTHETIC:OPS:AOC:HOLDOUT:1", "SupportedCount", "ReviewedCount") {
		t.Fatalf("persisted holdout Audit=%s err=%v", holdoutAudit, err)
	}
	if replay, err := admin.EvaluateGenerationRunHoldout(ctx, actor, submitted.GenerationRunID, []regulatory.HoldoutReview{{StableRowIdentity: "CC:SYNTHETIC:OPS:AOC:HOLDOUT:1", Outcome: regulatory.HoldoutSupported}}); err != nil || replay != result {
		t.Fatalf("holdout replay=%+v err=%v", replay, err)
	}
	if _, err := admin.EvaluateGenerationRunHoldout(ctx, actor, submitted.GenerationRunID, []regulatory.HoldoutReview{{StableRowIdentity: "CC:SYNTHETIC:OPS:AOC:1", Outcome: regulatory.HoldoutSupported}}); !errors.Is(err, regulatory.ErrHoldoutOverlap) {
		t.Fatalf("generation-input identity evaluated as holdout: %v", err)
	}
}

// A changed controlled source may create a new, review-required candidate,
// but it must never rewrite the pre-existing source-bound candidate lineage.
func TestTask8SyntheticSourceChangeCreatesNewImpactCandidateWithoutMutatingPriorLineage(t *testing.T) {
	ctx := context.Background()
	governance, manager, prior := task6SubmittedCandidate(t, "task8_source_change")
	admin := regulatory.NewAdminService(governance.Pool, governance.Clock)
	actor := identity.Principal{SubjectID: "USR-TASK6-ADMIN", Roles: []identity.Role{identity.RoleAdmin}}
	approved, err := governance.Approve(ctx, manager, checklistgovernance.ReviewCommand{OperationID: "TASK8-APPROVE", IdempotencyKey: "TASK8-APPROVE", CandidateID: prior.CandidateID, ExpectedRevision: prior.Revision, ExpectedContentDigest: prior.ContentDigest, Reason: "Approve the immutable Task 8 prior candidate."})
	if err != nil {
		t.Fatal(err)
	}
	publication, err := governance.Publish(ctx, manager, checklistgovernance.PublicationCommand{OperationID: "TASK8-PUBLISH", IdempotencyKey: "TASK8-PUBLISH", CandidateID: approved.CandidateID, ExpectedRevision: approved.Revision, ExpectedContentDigest: approved.ContentDigest, Reason: "Publish the immutable Task 8 prior candidate."})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := governance.Pool.Exec(ctx, `INSERT INTO identity_references (subject_id,issuer,display_name) VALUES ('USR-TASK8-INSPECTOR','task8-test','Task 8 Inspector'); INSERT INTO inspections (id,organization_id,assigned_inspector_subject_id,title,inspection_type,status) VALUES ('INSP-TASK8','ORG-SYNTHETIC-AOC','USR-TASK8-INSPECTOR','Task 8 immutable package','RAMP_INSPECTION','PREPARATION')`); err != nil {
		t.Fatal(err)
	}
	_, err = governance.MaterializeApplicablePublishedPackage(ctx, manager, checklistgovernance.MaterializeApplicablePublishedPackageCommand{OperationID: "TASK8-MATERIALIZE", IdempotencyKey: "TASK8-MATERIALIZE", CorrelationID: "TASK8-MATERIALIZE", InspectionID: "INSP-TASK8", PackageID: "PKG-TASK8", PackageVersion: 1, ExpiresAt: time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC), Selection: checklistgovernance.PublishedChecklistSelectionRequest{OrganizationID: "ORG-SYNTHETIC-AOC", InspectionType: "RAMP_INSPECTION", TargetID: "TARGET-SYNTHETIC-AOC", TargetKind: "ORGANIZATION", DepartmentID: "FLIGHT_OPERATIONS_INSPECTORATE", At: time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)}, AssignedInspectorSubjectIDs: map[string][]string{"Q-SYNTHETIC-OPS-AOC-001": {"USR-TASK8-INSPECTOR"}}})
	if err != nil {
		t.Fatal(err)
	}
	before, err := admin.GetSourceImpact(ctx, actor, "SOURCE-SYNTHETIC-OPS-AOC")
	if err != nil {
		t.Fatal(err)
	}
	var priorVersion, priorPackage, priorEvents, priorReviews, priorPublications []byte
	if err := governance.Pool.QueryRow(ctx, `SELECT snapshot FROM checklist_template_versions WHERE id=$1`, publication.TemplateVersionID).Scan(&priorVersion); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT snapshot FROM inspection_packages WHERE id='PKG-TASK8'`).Scan(&priorPackage); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('eventId',event_id,'action',action,'details',details) ORDER BY sequence_id),'[]'::jsonb)::text FROM audit_events WHERE entity_id=$1`, prior.CandidateID).Scan(&priorEvents); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(decision) ORDER BY id),'[]'::jsonb)::text FROM department_review_decisions decision WHERE candidate_draft_version_id=$1`, prior.CandidateID).Scan(&priorReviews); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(decision) ORDER BY id),'[]'::jsonb)::text FROM checklist_publication_decisions decision WHERE candidate_draft_version_id=$1`, prior.CandidateID).Scan(&priorPublications); err != nil {
		t.Fatal(err)
	}
	activation := activateSyntheticImpactCurrentness(t, ctx, admin, "TASK8-SOURCE-CURRENTNESS")
	run, err := admin.Import(ctx, actor, "TASK8-IMPACT-IMPORT", "TASK8-IMPACT-IMPORT", regulatory.SyntheticImpactCandidateBundle())
	if err != nil || run.Candidate == nil || run.Candidate.CandidateID == prior.CandidateID || run.Candidate.Status != regulatory.GeneratedDraft {
		t.Fatalf("impact import=%+v err=%v", run, err)
	}
	after, err := admin.GetSourceImpact(ctx, actor, "SOURCE-SYNTHETIC-OPS-AOC")
	if err != nil || len(after.CandidateIDs) != 1 || after.CandidateIDs[0] != prior.CandidateID || after.SourceHash != before.SourceHash {
		t.Fatalf("prior source lineage mutated before=%+v after=%+v err=%v", before, after, err)
	}
	changed, err := admin.GetSourceImpact(ctx, actor, "SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2")
	if err != nil || len(changed.CandidateIDs) != 1 || changed.CandidateIDs[0] != run.Candidate.CandidateID || changed.SourceHash == before.SourceHash {
		t.Fatalf("changed source impact=%+v err=%v", changed, err)
	}
	var afterVersion, afterPackage, afterEvents, afterReviews, afterPublications, impactDetails []byte
	if err := governance.Pool.QueryRow(ctx, `SELECT snapshot FROM checklist_template_versions WHERE id=$1`, publication.TemplateVersionID).Scan(&afterVersion); err != nil || string(afterVersion) != string(priorVersion) {
		t.Fatalf("published version mutated err=%v", err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT snapshot FROM inspection_packages WHERE id='PKG-TASK8'`).Scan(&afterPackage); err != nil || string(afterPackage) != string(priorPackage) {
		t.Fatalf("in-progress package mutated err=%v", err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('eventId',event_id,'action',action,'details',details) ORDER BY sequence_id),'[]'::jsonb)::text FROM audit_events WHERE entity_id=$1`, prior.CandidateID).Scan(&afterEvents); err != nil || string(afterEvents) != string(priorEvents) {
		t.Fatalf("prior candidate Audit history mutated err=%v", err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(decision) ORDER BY id),'[]'::jsonb)::text FROM department_review_decisions decision WHERE candidate_draft_version_id=$1`, prior.CandidateID).Scan(&afterReviews); err != nil || string(afterReviews) != string(priorReviews) {
		t.Fatalf("prior technical-review decisions mutated err=%v", err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(decision) ORDER BY id),'[]'::jsonb)::text FROM checklist_publication_decisions decision WHERE candidate_draft_version_id=$1`, prior.CandidateID).Scan(&afterPublications); err != nil || string(afterPublications) != string(priorPublications) {
		t.Fatalf("prior publication decisions mutated err=%v", err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT details::text FROM audit_events WHERE entity_type='REGULATORY_SOURCE_IMPACT' AND entity_id=$1`, run.Candidate.CandidateID).Scan(&impactDetails); err != nil || string(impactDetails) == "" || activation.ImpactReviewDraftID == nil || !containsAll(string(impactDetails), "currentnessEventId", activation.EventID, "impactReviewDraftId", *activation.ImpactReviewDraftID, "SOURCE-SYNTHETIC-OPS-AOC", "SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2") {
		t.Fatalf("source impact Audit=%s err=%v", impactDetails, err)
	}
	if _, err := governance.Publish(ctx, manager, checklistgovernance.PublicationCommand{OperationID: "TASK8-REPUBLISH-SUPERSEDED", IdempotencyKey: "TASK8-REPUBLISH-SUPERSEDED", CandidateID: approved.CandidateID, ExpectedRevision: approved.Revision, ExpectedContentDigest: approved.ContentDigest, Reason: "Attempt to reuse superseded published candidate."}); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("published prior candidate was reusable: %v", err)
	}
}

func containsAll(value string, want ...string) bool {
	for _, item := range want {
		if !strings.Contains(value, item) {
			return false
		}
	}
	return true
}
