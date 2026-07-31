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
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/checklistgovernance"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/regulatory"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/testprofile"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
	"github.com/jackc/pgx/v5/pgconn"
)

func regulatoryRefreshAdmin() identity.Principal {
	return identity.Principal{SubjectID: "USR-TASK6-ADMIN", Roles: []identity.Role{identity.RoleAdmin}}
}

func requireRegulatoryRefreshIssue(t *testing.T, err error, code string) {
	t.Helper()
	var validation *regulatory.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error=%T %v, want governed validation issue %s", err, err, code)
	}
	for _, issue := range validation.Issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("validation issues=%+v, want %s", validation.Issues, code)
}

func syntheticImpactCurrentnessCommand(operationID string) regulatory.SourceCurrentnessActivationCommand {
	bundle := regulatory.SyntheticImpactCandidateBundle()
	if bundle.SourceCurrentness == nil || bundle.SourceCurrentness.PreviousSourceSnapshotID == "" || bundle.SourceCurrentness.PreviousSourceHash == "" {
		panic("synthetic impact fixture must declare one exact predecessor/current currentness binding")
	}
	return regulatory.SourceCurrentnessActivationCommand{
		OperationID:              operationID,
		IdempotencyKey:           operationID,
		CurrentSourceSnapshotID:  bundle.SourceCurrentness.CurrentSourceSnapshotID,
		CurrentSourceHash:        bundle.SourceCurrentness.CurrentSourceHash,
		PreviousSourceSnapshotID: bundle.SourceCurrentness.PreviousSourceSnapshotID,
		PreviousSourceHash:       bundle.SourceCurrentness.PreviousSourceHash,
		Reason:                   "Activate the exact current synthetic source before a separate impact-review Draft may be imported.",
	}
}

func activateSyntheticImpactCurrentness(t *testing.T, ctx context.Context, admin *regulatory.AdminService, operationID string) regulatory.SourceCurrentnessActivationView {
	t.Helper()
	activation, err := admin.ActivateSourceCurrentness(ctx, regulatoryRefreshAdmin(), syntheticImpactCurrentnessCommand(operationID))
	if err != nil || activation.Status != "IMPACT_REVIEW_DRAFT" || activation.ImpactReviewDraftID == nil || activation.EventID == "" {
		t.Fatalf("activate synthetic impact source=%+v err=%v", activation, err)
	}
	return activation
}

func waitForSourceCurrentnessLockWaiter(t *testing.T, ctx context.Context, pool *database.Pool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var waiting int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM pg_locks WHERE locktype='advisory' AND NOT granted`).Scan(&waiting); err != nil {
			t.Fatalf("inspect source-currentness advisory lock waiter: %v", err)
		}
		if waiting > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("source-currentness activation did not wait on the shared advisory lock")
}

// Break caught: the source-gap fixture could pass direct validation only after
// an unrelated base candidate had been imported. A real Admin import must work
// from a clean source fixture and still create only the repairable Draft.
func TestRegulatorySourceRefreshTask6LegacyImportWorksFromCleanSyntheticInputs(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "regulatory_refresh_legacy_clean_import")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO identity_references (subject_id,issuer,display_name)
		VALUES ('USR-TASK6-ADMIN','regulatory-refresh','Task 6 Admin')`); err != nil {
		t.Fatal(err)
	}
	admin := regulatory.NewAdminService(pool, func() time.Time {
		return time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	})
	run, err := admin.Import(
		ctx, regulatoryRefreshAdmin(), "REFRESH-T6-CLEAN-LEGACY-IMPORT",
		"REFRESH-T6-CLEAN-LEGACY-IMPORT", regulatory.SyntheticLegacyChecklistCandidateBundle(),
	)
	if err != nil || run.Candidate == nil || len(run.Candidate.Questions) != 1 {
		t.Fatalf("clean legacy import=%+v err=%v", run, err)
	}
	question := run.Candidate.Questions[0]
	if question.Origin != regulatory.ExistingChecklistCandidateOrigin ||
		question.RegulatoryTrace.State != regulatory.SourceMappingRequired ||
		len(question.Citations) != 0 {
		t.Fatalf("clean legacy import lost literal candidate-only repair state: %+v", question)
	}
}

// Break caught: direct Go import passed while the canonical HTTP boundary
// returned a generic 500 for the same literal source-gap Draft. The HTTP
// route must preserve the repairable candidate-only state without silently
// changing it into a server error.
func TestRegulatorySourceRefreshTask6LegacySourceGapImportOverCanonicalHTTP(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "regulatory_refresh_legacy_http")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	now := testprofile.CanonicalScenarioTime()
	if err := testprofile.Reset(ctx, pool, now); err != nil {
		t.Fatalf("reset canonical profile: %v", err)
	}
	// The real canonical HTTP artifact seeds on startup and then serves a
	// browser-triggered /__test/reset before the Admin action. Preserve that
	// exact reset boundary here because a candidate import must remain valid
	// across both immutable profile resets.
	if err := testprofile.Reset(ctx, pool, now); err != nil {
		t.Fatalf("repeat canonical browser reset: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap synthetic inputs: %v", err)
	}
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
		Pool: pool, Clock: func() time.Time { return now },
	})
	handler := httpapi.NewCanonicalTestBoundary("regulatory-refresh-token").Protect(api.Handler())
	const operationID = "ADMIN-SYNTHETIC-LEGACY-CANDIDATE-IMPORT"
	payload, err := json.Marshal(map[string]any{
		"operationId":     operationID,
		"idempotencyKey":  operationID,
		"candidateBundle": regulatory.SyntheticLegacyChecklistCandidateBundle(),
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/governed-checklist/generation-runs", bytes.NewReader(payload))
	request.Header.Set(httpapi.CanonicalTestTokenHeader, "regulatory-refresh-token")
	request.Header.Set(httpapi.CanonicalTestSubjectHeader, "USR-ADMIN-ADA")
	request.Header.Set("Idempotency-Key", operationID)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		actor, ok := testprofile.Principal("USR-ADMIN-ADA")
		if !ok {
			t.Fatal("canonical Admin principal is unavailable")
		}
		_, directErr := regulatory.NewAdminService(pool, func() time.Time { return now }).Import(
			ctx, actor, operationID, operationID, regulatory.SyntheticLegacyChecklistCandidateBundle(),
		)
		t.Fatalf("legacy source-gap HTTP import status=%d body=%s directImportError=%v", response.Code, response.Body.String(), directErr)
	}
	var imported regulatory.GenerationRunView
	if err := json.Unmarshal(response.Body.Bytes(), &imported); err != nil || imported.Candidate == nil {
		t.Fatalf("decode legacy source-gap HTTP import=%+v err=%v body=%s", imported, err, response.Body.String())
	}
	question := imported.Candidate.Questions[0]
	if question.Origin != regulatory.ExistingChecklistCandidateOrigin || question.RegulatoryTrace.State != regulatory.SourceMappingRequired || len(question.Citations) != 0 {
		t.Fatalf("HTTP import did not preserve literal candidate-only repair state: %+v", question)
	}
}

// A supplied V2 row is deliberately inert at the transport boundary too.
// The explicit activation receipt is a separate Admin command, creates the
// immutable impact-review Draft before import, and is idempotent without
// becoming a technical approval or publication decision.
func TestRegulatorySourceRefreshTask6CanonicalHTTPRequiresExplicitSourceActivationBeforeV2Import(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "regulatory_refresh_activation_http")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	now := testprofile.CanonicalScenarioTime()
	if err := testprofile.Reset(ctx, pool, now); err != nil {
		t.Fatalf("reset canonical profile: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap synthetic inputs: %v", err)
	}
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{Pool: pool, Clock: func() time.Time { return now }})
	handler := httpapi.NewCanonicalTestBoundary("regulatory-refresh-token").Protect(api.Handler())
	requestJSON := func(t *testing.T, route string, payload any) *httptest.ResponseRecorder {
		t.Helper()
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, route, bytes.NewReader(body))
		request.Header.Set(httpapi.CanonicalTestTokenHeader, "regulatory-refresh-token")
		request.Header.Set(httpapi.CanonicalTestSubjectHeader, "USR-ADMIN-ADA")
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", "REGULATORY-REFRESH-HTTP")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}

	const rawImportOperation = "REFRESH-T6-HTTP-RAW-V2-IMPORT"
	rawImport := requestJSON(t, "/v1/admin/governed-checklist/generation-runs", map[string]any{
		"operationId": rawImportOperation, "idempotencyKey": rawImportOperation,
		"candidateBundle": regulatory.SyntheticImpactCandidateBundle(),
	})
	if rawImport.Code != http.StatusUnprocessableEntity {
		t.Fatalf("raw V2 HTTP import status=%d body=%s", rawImport.Code, rawImport.Body.String())
	}
	missingPredecessor := requestJSON(t, "/v1/admin/governed-checklist/source-currentness-activations", map[string]any{
		"operationId": "REFRESH-T6-HTTP-MISSING-PREDECESSOR", "idempotencyKey": "REFRESH-T6-HTTP-MISSING-PREDECESSOR",
		"currentSourceSnapshotId": "SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2",
		"currentSourceHash":       "sha256:4444444444444444444444444444444444444444444444444444444444444444",
		"reason":                  "The transport must require explicit predecessor fields, including explicit JSON null for a baseline.",
	})
	if missingPredecessor.Code != http.StatusUnprocessableEntity {
		t.Fatalf("source-currentness request without explicit predecessor fields status=%d body=%s", missingPredecessor.Code, missingPredecessor.Body.String())
	}
	unknownActivationField := requestJSON(t, "/v1/admin/governed-checklist/source-currentness-activations", map[string]any{
		"operationId":              "TESTPROFILE-SOURCE-CURRENTNESS-BASELINE-V1",
		"idempotencyKey":           "TESTPROFILE-SOURCE-CURRENTNESS-BASELINE-V1",
		"currentSourceSnapshotId":  "SOURCE-SYNTHETIC-OPS-AOC",
		"currentSourceHash":        "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		"previousSourceSnapshotId": nil,
		"previousSourceHash":       nil,
		"reason":                   "Synthetic internal test-profile baseline currentness declaration.",
		"unexpected":               true,
	})
	if unknownActivationField.Code != http.StatusUnprocessableEntity {
		t.Fatalf("source-currentness request with an unknown field status=%d body=%s", unknownActivationField.Code, unknownActivationField.Body.String())
	}

	activationCommand := syntheticImpactCurrentnessCommand("REFRESH-T6-HTTP-SOURCE-CURRENTNESS")
	activationResponse := requestJSON(t, "/v1/admin/governed-checklist/source-currentness-activations", activationCommand)
	if activationResponse.Code != http.StatusCreated {
		t.Fatalf("source-currentness activation status=%d body=%s", activationResponse.Code, activationResponse.Body.String())
	}
	var activation regulatory.SourceCurrentnessActivationView
	if err := json.Unmarshal(activationResponse.Body.Bytes(), &activation); err != nil || activation.Status != "IMPACT_REVIEW_DRAFT" || activation.ImpactReviewDraftID == nil {
		t.Fatalf("decode source-currentness activation=%+v err=%v body=%s", activation, err, activationResponse.Body.String())
	}
	replayResponse := requestJSON(t, "/v1/admin/governed-checklist/source-currentness-activations", activationCommand)
	var replay regulatory.SourceCurrentnessActivationView
	if replayResponse.Code != http.StatusCreated || json.Unmarshal(replayResponse.Body.Bytes(), &replay) != nil || replay.EventID != activation.EventID || replay.ImpactReviewDraftID == nil || *replay.ImpactReviewDraftID != *activation.ImpactReviewDraftID {
		t.Fatalf("source-currentness replay status=%d receipt=%+v expected=%+v body=%s", replayResponse.Code, replay, activation, replayResponse.Body.String())
	}

	const activatedImportOperation = "REFRESH-T6-HTTP-ACTIVATED-V2-IMPORT"
	activatedImport := requestJSON(t, "/v1/admin/governed-checklist/generation-runs", map[string]any{
		"operationId": activatedImportOperation, "idempotencyKey": activatedImportOperation,
		"candidateBundle": regulatory.SyntheticImpactCandidateBundle(),
	})
	if activatedImport.Code != http.StatusCreated {
		t.Fatalf("activated V2 HTTP import status=%d body=%s", activatedImport.Code, activatedImport.Body.String())
	}
	var imported regulatory.GenerationRunView
	if err := json.Unmarshal(activatedImport.Body.Bytes(), &imported); err != nil || imported.Candidate == nil {
		t.Fatalf("decode activated V2 import=%+v err=%v body=%s", imported, err, activatedImport.Body.String())
	}
	var eventRows, draftRows, bindingRows, candidateLinks int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_source_currentness_events WHERE event_id=$1`, activation.EventID).Scan(&eventRows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_source_impact_review_drafts WHERE id=$1 AND currentness_event_id=$2`, *activation.ImpactReviewDraftID, activation.EventID).Scan(&draftRows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_generation_run_source_currentness_bindings WHERE generation_run_id=$1 AND currentness_event_id=$2`, imported.GenerationRunID, activation.EventID).Scan(&bindingRows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_source_impact_candidate_links WHERE impact_review_draft_id=$1 AND candidate_draft_version_id=$2 AND generation_run_id=$3`, *activation.ImpactReviewDraftID, imported.Candidate.CandidateID, imported.GenerationRunID).Scan(&candidateLinks); err != nil {
		t.Fatal(err)
	}
	if eventRows != 1 || draftRows != 1 || bindingRows != 1 || candidateLinks != 1 {
		t.Fatalf("explicit HTTP activation graph event=%d draft=%d binding=%d candidateLink=%d", eventRows, draftRows, bindingRows, candidateLinks)
	}
}

// Pre-currentness Task 6 data cannot gain a silent legacy exception. A
// resolved question whose old generation run has no immutable activation
// binding must project stale and deny review, rather than treating the
// persisted citation as enough to technically approve it.
func TestRegulatorySourceRefreshTask6UnboundHistoricalCandidateFailsClosed(t *testing.T) {
	ctx := context.Background()
	governance, manager, submitted := task6SubmittedCandidate(t, "regulatory_refresh_unbound_historical_candidate")
	const bindingTrigger = "regulatory_generation_run_source_currentness_bindings_append_only"
	if _, err := governance.Pool.Exec(ctx, "ALTER TABLE regulatory_generation_run_source_currentness_bindings DISABLE TRIGGER "+bindingTrigger); err != nil {
		t.Fatalf("temporarily disable append-only binding trigger for historical-data simulation: %v", err)
	}
	triggerEnabled := false
	t.Cleanup(func() {
		if !triggerEnabled {
			if _, err := governance.Pool.Exec(context.Background(), "ALTER TABLE regulatory_generation_run_source_currentness_bindings ENABLE TRIGGER "+bindingTrigger); err != nil {
				t.Errorf("restore append-only binding trigger: %v", err)
			}
		}
	})
	if _, err := governance.Pool.Exec(ctx, `
		DELETE FROM regulatory_generation_run_source_currentness_bindings
		WHERE generation_run_id=$1`, submitted.GenerationRunID); err != nil {
		t.Fatalf("remove only the synthetic historical currentness binding: %v", err)
	}
	if _, err := governance.Pool.Exec(ctx, "ALTER TABLE regulatory_generation_run_source_currentness_bindings ENABLE TRIGGER "+bindingTrigger); err != nil {
		t.Fatalf("restore append-only binding trigger: %v", err)
	}
	triggerEnabled = true

	admin := regulatory.NewAdminService(governance.Pool, governance.Clock)
	projected, err := admin.GetCandidate(ctx, regulatoryRefreshAdmin(), submitted.CandidateID)
	if err != nil || len(projected.Questions) != 1 || projected.Questions[0].RegulatoryTrace.CurrentnessState != "STALE" {
		t.Fatalf("unbound historical candidate must project stale=%+v err=%v", projected, err)
	}
	_, err = governance.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "REFRESH-T6-UNBOUND-HISTORICAL-APPROVE", IdempotencyKey: "REFRESH-T6-UNBOUND-HISTORICAL-APPROVE",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest, Reason: "A historical candidate without immutable source-currentness proof cannot be technically approved.",
	})
	requireRegulatoryRefreshIssue(t, err, "SOURCE_CURRENTNESS_REQUIRED")
	var approvals int
	if err := governance.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM department_review_decisions
		WHERE candidate_draft_version_id=$1 AND decision='TECHNICALLY_APPROVED'`, submitted.CandidateID).Scan(&approvals); err != nil || approvals != 0 {
		t.Fatalf("unbound historical candidate technical-approval effects=%d err=%v", approvals, err)
	}
}

// Break caught: an aborted persistence transaction was being converted into a
// secondary FAILED_IMPORT write, which obscured the original database failure.
// A fail-closed import must expose the original persistence failure and leave
// no misleading candidate failure record behind.
func TestRegulatorySourceRefreshTask6DoesNotMaskPersistenceFailure(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "regulatory_refresh_persistence_failure")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO identity_references (subject_id,issuer,display_name)
		VALUES ('USR-TASK6-ADMIN','regulatory-refresh','Task 6 Admin')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE regulated_targets RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}
	admin := regulatory.NewAdminService(pool, func() time.Time {
		return time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	})
	_, err := admin.Import(
		ctx, regulatoryRefreshAdmin(), "REFRESH-T6-PERSISTENCE-FAILURE",
		"REFRESH-T6-PERSISTENCE-FAILURE", regulatory.SyntheticCandidateBundle(),
	)
	var databaseError *pgconn.PgError
	if !errors.As(err, &databaseError) || databaseError.Code != "23503" {
		t.Fatalf("persistence error=%T %v, want original foreign-key failure", err, err)
	}
	var failedImports int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM governed_candidate_commands WHERE operation_id='REFRESH-T6-PERSISTENCE-FAILURE'`).Scan(&failedImports); err != nil || failedImports != 0 {
		t.Fatalf("masked persistence failure recorded failed command count=%d err=%v", failedImports, err)
	}
}

// Break caught: an existing checklist candidate could be submitted with an
// empty trace, turning an unresolved source gap into a review or execution
// claim. The literal gap remains a Draft-only repair state.
func TestRegulatorySourceRefreshTask6SourceGapQuestionIsPersistedButFailsClosed(t *testing.T) {
	ctx := context.Background()
	governance, manager, _ := task6SubmittedCandidate(t, "regulatory_refresh_source_gap")
	admin := regulatory.NewAdminService(governance.Pool, governance.Clock)
	run, err := admin.Import(ctx, regulatoryRefreshAdmin(), "REFRESH-T6-LEGACY-IMPORT", "REFRESH-T6-LEGACY-IMPORT", regulatory.SyntheticLegacyChecklistCandidateBundle())
	if err != nil || run.Candidate == nil {
		t.Fatalf("import legacy candidate=%+v err=%v", run, err)
	}
	candidate := run.Candidate
	if candidate.Status != regulatory.GeneratedDraft || len(candidate.Questions) != 1 ||
		candidate.Questions[0].Origin != regulatory.ExistingChecklistCandidateOrigin ||
		candidate.Questions[0].RegulatoryTrace.State != regulatory.SourceMappingRequired ||
		len(candidate.Questions[0].Citations) != 0 {
		t.Fatalf("legacy candidate did not retain its explicit source-gap Draft state: %+v", candidate)
	}
	var snapshot []byte
	if err := governance.Pool.QueryRow(ctx, `SELECT snapshot::text FROM regulatory_generated_question_snapshots WHERE candidate_draft_version_id=$1 AND question_id=$2`, candidate.CandidateID, candidate.Questions[0].QuestionID).Scan(&snapshot); err != nil ||
		!strings.Contains(string(snapshot), `"SOURCE_MAPPING_REQUIRED"`) || strings.Contains(string(snapshot), `"citations":[{`) {
		t.Fatalf("persisted gap question=%s err=%v", snapshot, err)
	}

	_, err = admin.Submit(ctx, regulatoryRefreshAdmin(), regulatory.SubmitCommand{
		OperationID: "REFRESH-T6-LEGACY-SUBMIT", IdempotencyKey: "REFRESH-T6-LEGACY-SUBMIT",
		CandidateID: candidate.CandidateID, ExpectedRevision: candidate.Revision,
		ExpectedContentDigest: candidate.ContentDigest, Reason: "Source mapping must be repaired before review.",
	})
	requireRegulatoryRefreshIssue(t, err, "SOURCE_MAPPING_REQUIRED")
	if _, err := governance.Publish(ctx, manager, checklistgovernance.PublicationCommand{
		OperationID: "REFRESH-T6-LEGACY-PUBLISH", IdempotencyKey: "REFRESH-T6-LEGACY-PUBLISH",
		CandidateID: candidate.CandidateID, ExpectedRevision: candidate.Revision,
		ExpectedContentDigest: candidate.ContentDigest, Reason: "A source-gap Draft must not publish.",
	}); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("source-gap Draft publication error=%v, want conflict", err)
	}
	var status string
	if err := governance.Pool.QueryRow(ctx, `SELECT status FROM template_draft_versions WHERE id=$1`, candidate.CandidateID).Scan(&status); err != nil || status != regulatory.GeneratedDraft {
		t.Fatalf("source-gap candidate status=%q err=%v", status, err)
	}
	for _, table := range []string{"department_review_decisions", "checklist_publication_decisions", "checklist_template_versions"} {
		var count int
		if err := governance.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+table+" WHERE candidate_draft_version_id=$1", candidate.CandidateID).Scan(&count); err != nil || count != 0 {
			t.Fatalf("source-gap Draft created %s=%d err=%v", table, count, err)
		}
	}
}

// Break caught: a generic Admin can import and submit a candidate, but must
// never inherit Department Manager review or publication authority merely from
// that role. Both denied commands must leave their respective immutable
// decision and Audit ledgers untouched.
func TestRegulatorySourceRefreshTask6GenericAdminCannotTechnicallyApproveOrPublish(t *testing.T) {
	ctx := context.Background()
	governance, manager, submitted := task6SubmittedCandidate(t, "regulatory_refresh_admin_authority")
	admin := regulatoryRefreshAdmin()

	_, err := governance.Approve(ctx, admin, checklistgovernance.ReviewCommand{
		OperationID: "REFRESH-T6-ADMIN-APPROVE", IdempotencyKey: "REFRESH-T6-ADMIN-APPROVE",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest, Reason: "Generic Admin has no technical-approval authority.",
	})
	if !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("generic Admin technical approval error=%v, want forbidden", err)
	}
	assertTask6DecisionEffect(t, governance, "REFRESH-T6-ADMIN-APPROVE", "TECHNICALLY_APPROVED", 0)

	approved, err := governance.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "REFRESH-T6-MANAGER-APPROVE", IdempotencyKey: "REFRESH-T6-MANAGER-APPROVE",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest, Reason: "Current Department Manager technical approval.",
	})
	if err != nil || approved.Status != "TECHNICALLY_APPROVED" {
		t.Fatalf("manager technical approval=%+v err=%v", approved, err)
	}
	_, err = governance.Publish(ctx, admin, checklistgovernance.PublicationCommand{
		OperationID: "REFRESH-T6-ADMIN-PUBLISH", IdempotencyKey: "REFRESH-T6-ADMIN-PUBLISH",
		CandidateID: approved.CandidateID, ExpectedRevision: approved.Revision,
		ExpectedContentDigest: approved.ContentDigest, Reason: "Generic Admin has no publication authority.",
	})
	if !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("generic Admin publication error=%v, want forbidden", err)
	}
	var publications, audits int
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM checklist_publication_decisions WHERE operation_id='REFRESH-T6-ADMIN-PUBLISH'`).Scan(&publications); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE operation_id='REFRESH-T6-ADMIN-PUBLISH'`).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if publications != 0 || audits != 0 {
		t.Fatalf("generic Admin publication effects decisions=%d audits=%d, want 0/0", publications, audits)
	}
}

// Break caught: a transport client could forge both question review states as
// TECHNICALLY_APPROVED during Admin import. The canonical HTTP boundary must
// reject that Draft and persist neither a candidate nor its question snapshot;
// only an attributed Department Manager decision may project approval later.
func TestRegulatorySourceRefreshTask6CanonicalHTTPRejectsPreclaimedTechnicalApproval(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "regulatory_refresh_preclaimed_approval_http")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	now := testprofile.CanonicalScenarioTime()
	if err := testprofile.Reset(ctx, pool, now); err != nil {
		t.Fatal(err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatal(err)
	}
	bundle := regulatory.SyntheticCandidateBundle()
	question := &bundle.InspectionChecklist.Questions[0]
	question.ScopeRecommendation.ApprovalReviewState = "TECHNICALLY_APPROVED"
	question.RegulatoryTrace.TechnicalReviewState = "TECHNICALLY_APPROVED"
	digest, err := regulatory.CanonicalSHA256(map[string]any{
		"complianceMappings":  bundle.ComplianceMappings,
		"inspectionChecklist": bundle.InspectionChecklist,
	})
	if err != nil {
		t.Fatal(err)
	}
	bundle.OutputDigest = digest
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
		Pool: pool, Clock: func() time.Time { return now },
	})
	handler := httpapi.NewCanonicalTestBoundary("regulatory-refresh-token").Protect(api.Handler())
	const operationID = "REFRESH-T6-PRECLAIMED-APPROVAL"
	payload, err := json.Marshal(map[string]any{
		"operationId": operationID, "idempotencyKey": operationID, "candidateBundle": bundle,
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/governed-checklist/generation-runs", bytes.NewReader(payload))
	request.Header.Set(httpapi.CanonicalTestTokenHeader, "regulatory-refresh-token")
	request.Header.Set(httpapi.CanonicalTestSubjectHeader, "USR-ADMIN-ADA")
	request.Header.Set("Idempotency-Key", operationID)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("preclaimed approval import status=%d body=%s", response.Code, response.Body.String())
	}
	var candidates, snapshots int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM template_draft_versions WHERE id=$1`, bundle.CandidateBundleID).Scan(&candidates); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_generated_question_snapshots WHERE candidate_draft_version_id=$1`, bundle.CandidateBundleID).Scan(&snapshots); err != nil {
		t.Fatal(err)
	}
	if candidates != 0 || snapshots != 0 {
		t.Fatalf("preclaimed approval persisted candidate=%d questionSnapshots=%d", candidates, snapshots)
	}
}

// Break caught: a source-version/hash change could leave a previously
// approved candidate eligible for publication. The change must project the old
// trace as stale, deny publication, and leave an immutable impact Draft for
// the current source chain without rewriting the old bytes.
func TestRegulatorySourceRefreshTask6SourceChangeMarksTraceStaleAndBlocksPublication(t *testing.T) {
	ctx := context.Background()
	governance, manager, prior := task6SubmittedCandidate(t, "regulatory_refresh_stale")
	approved, err := governance.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "REFRESH-T6-STALE-APPROVE", IdempotencyKey: "REFRESH-T6-STALE-APPROVE",
		CandidateID: prior.CandidateID, ExpectedRevision: prior.Revision,
		ExpectedContentDigest: prior.ContentDigest, Reason: "Approve before the separate source-change event.",
	})
	if err != nil || approved.Status != "TECHNICALLY_APPROVED" {
		t.Fatalf("approve prior candidate=%+v err=%v", approved, err)
	}
	var priorSnapshot []byte
	if err := governance.Pool.QueryRow(ctx, `SELECT snapshot::text FROM regulatory_generated_question_snapshots WHERE candidate_draft_version_id=$1 AND question_id=$2`, prior.CandidateID, prior.Questions[0].QuestionID).Scan(&priorSnapshot); err != nil {
		t.Fatal(err)
	}
	admin := regulatory.NewAdminService(governance.Pool, governance.Clock)
	// V2 is only a supplied synthetic source row at this point. Importing it
	// cannot make it current or manufacture the impact-review Draft.
	_, err = admin.Import(ctx, regulatoryRefreshAdmin(), "REFRESH-T6-STALE-RAW-V2", "REFRESH-T6-STALE-RAW-V2", regulatory.SyntheticImpactCandidateBundle())
	if !errors.Is(err, application.ErrInvalid) {
		t.Fatalf("raw V2 import error=%v, want explicit source-currentness denial", err)
	}
	var rawRunRows, rawScopeRows, rawSourceRows, rawPartitionRows, rawCandidateRows, rawBindingRows, rawImpactDrafts int
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_generation_runs WHERE id='GENRUN-SYNTHETIC-OPS-AOC-IMPACT-0002'`).Scan(&rawRunRows); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_generation_run_scope_facts WHERE generation_run_id='GENRUN-SYNTHETIC-OPS-AOC-IMPACT-0002'`).Scan(&rawScopeRows); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_generation_run_source_snapshots WHERE generation_run_id='GENRUN-SYNTHETIC-OPS-AOC-IMPACT-0002'`).Scan(&rawSourceRows); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_generation_run_crosswalk_partition_rows WHERE generation_run_id='GENRUN-SYNTHETIC-OPS-AOC-IMPACT-0002'`).Scan(&rawPartitionRows); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM template_draft_versions WHERE id='CAND-SYNTHETIC-OPS-AOC-IMPACT-0002'`).Scan(&rawCandidateRows); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_generation_run_source_currentness_bindings WHERE generation_run_id='GENRUN-SYNTHETIC-OPS-AOC-IMPACT-0002'`).Scan(&rawBindingRows); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_source_impact_review_drafts WHERE current_source_version_id='SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2'`).Scan(&rawImpactDrafts); err != nil {
		t.Fatal(err)
	}
	if rawRunRows != 0 || rawScopeRows != 0 || rawSourceRows != 0 || rawPartitionRows != 0 || rawCandidateRows != 0 || rawBindingRows != 0 || rawImpactDrafts != 0 {
		t.Fatalf("raw V2 import left partial graph run=%d scope=%d source=%d partition=%d candidate=%d binding=%d impactDraft=%d", rawRunRows, rawScopeRows, rawSourceRows, rawPartitionRows, rawCandidateRows, rawBindingRows, rawImpactDrafts)
	}
	activation := activateSyntheticImpactCurrentness(t, ctx, admin, "REFRESH-T6-STALE-SOURCE-CURRENTNESS")
	var activatedDraftID string
	if err := governance.Pool.QueryRow(ctx, `
		SELECT id FROM regulatory_source_impact_review_drafts
		WHERE id=$1 AND currentness_event_id=$2
		  AND previous_source_version_id='SOURCE-SYNTHETIC-OPS-AOC'
		  AND current_source_version_id='SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2'`,
		*activation.ImpactReviewDraftID, activation.EventID,
	).Scan(&activatedDraftID); err != nil || activatedDraftID != *activation.ImpactReviewDraftID {
		t.Fatalf("activated immutable impact-review Draft=%q/%q err=%v", activatedDraftID, *activation.ImpactReviewDraftID, err)
	}
	impact, err := admin.Import(ctx, regulatoryRefreshAdmin(), "REFRESH-T6-STALE-IMPACT", "REFRESH-T6-STALE-IMPACT", regulatory.SyntheticImpactCandidateBundle())
	if err != nil || impact.Candidate == nil || impact.Candidate.Status != regulatory.GeneratedDraft || impact.Candidate.CandidateID == prior.CandidateID {
		t.Fatalf("impact Draft=%+v err=%v", impact, err)
	}
	impactReplay, err := admin.Import(ctx, regulatoryRefreshAdmin(), "REFRESH-T6-STALE-IMPACT-REPLAY", "REFRESH-T6-STALE-IMPACT-REPLAY", regulatory.SyntheticImpactCandidateBundle())
	if err != nil || impactReplay.Candidate == nil || impactReplay.Candidate.CandidateID != impact.Candidate.CandidateID {
		t.Fatalf("impact Draft replay=%+v err=%v", impactReplay, err)
	}
	var impactEvents, impactLinks int
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE event_id=$1`, "AE-SOURCE-IMPACT-"+impact.Candidate.CandidateID).Scan(&impactEvents); err != nil || impactEvents != 1 {
		t.Fatalf("impact audit effects=%d err=%v", impactEvents, err)
	}
	if err := governance.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM regulatory_source_impact_candidate_links
		WHERE impact_review_draft_id=$1 AND candidate_draft_version_id=$2 AND generation_run_id=$3`,
		activatedDraftID, impact.Candidate.CandidateID, impact.Candidate.GenerationRunID,
	).Scan(&impactLinks); err != nil || impactLinks != 1 {
		t.Fatalf("impact candidate link=%d err=%v", impactLinks, err)
	}
	projected, err := admin.GetCandidate(ctx, regulatoryRefreshAdmin(), prior.CandidateID)
	if err != nil || len(projected.Questions) != 1 || projected.Questions[0].RegulatoryTrace.CurrentnessState != "STALE" || !projected.Questions[0].ScopeRecommendation.Guardrails.SourceChanged {
		t.Fatalf("stale prior projection=%+v err=%v", projected, err)
	}
	_, err = governance.Publish(ctx, manager, checklistgovernance.PublicationCommand{
		OperationID: "REFRESH-T6-STALE-PUBLISH", IdempotencyKey: "REFRESH-T6-STALE-PUBLISH",
		CandidateID: approved.CandidateID, ExpectedRevision: approved.Revision,
		ExpectedContentDigest: approved.ContentDigest, Reason: "A stale trace cannot be published.",
	})
	requireRegulatoryRefreshIssue(t, err, "STALE_SOURCE_TRACE")
	var afterSnapshot []byte
	if err := governance.Pool.QueryRow(ctx, `SELECT snapshot::text FROM regulatory_generated_question_snapshots WHERE candidate_draft_version_id=$1 AND question_id=$2`, prior.CandidateID, prior.Questions[0].QuestionID).Scan(&afterSnapshot); err != nil || string(afterSnapshot) != string(priorSnapshot) {
		t.Fatalf("stale projection mutated immutable prior snapshot err=%v", err)
	}
	var publications int
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM checklist_publication_decisions WHERE candidate_draft_version_id=$1`, prior.CandidateID).Scan(&publications); err != nil || publications != 0 {
		t.Fatalf("stale candidate publication effects=%d err=%v", publications, err)
	}
}

// Source-currentness is a linear immutable ledger, not a sort over effective
// dates or all observed source rows. An older snapshot cannot be revived after
// a later activation because its original generation-run identity is already
// frozen; a controlled restoration must arrive as a new source-version ID.
func TestRegulatorySourceRefreshTask6SourceCurrentnessUsesExactHeadAndRejectsHistoricalReactivation(t *testing.T) {
	ctx := context.Background()
	governance, _, _ := task6SubmittedCandidate(t, "regulatory_refresh_currentness_head")
	admin := regulatory.NewAdminService(governance.Pool, governance.Clock)

	baselineConflict := syntheticImpactCurrentnessCommand("REFRESH-T6-CURRENTNESS-DUPLICATE-BASELINE")
	baselineConflict.PreviousSourceSnapshotID = ""
	baselineConflict.PreviousSourceHash = ""
	if _, err := admin.ActivateSourceCurrentness(ctx, regulatoryRefreshAdmin(), baselineConflict); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("second baseline activation error=%v, want conflict", err)
	}
	base := regulatory.SyntheticCandidateBundle().SourceCurrentness
	if base == nil {
		t.Fatal("synthetic baseline source-currentness binding is absent")
	}
	if _, err := admin.ActivateSourceCurrentness(ctx, regulatoryRefreshAdmin(), regulatory.SourceCurrentnessActivationCommand{
		OperationID: "REFRESH-T6-CURRENTNESS-SELF", IdempotencyKey: "REFRESH-T6-CURRENTNESS-SELF",
		CurrentSourceSnapshotID: base.CurrentSourceSnapshotID, CurrentSourceHash: base.CurrentSourceHash,
		PreviousSourceSnapshotID: base.CurrentSourceSnapshotID, PreviousSourceHash: base.CurrentSourceHash,
		Reason: "A self transition has no immutable predecessor/current meaning.",
	}); !errors.Is(err, application.ErrInvalid) {
		t.Fatalf("self activation error=%v, want invalid", err)
	}

	const restoredSourceID = "SOURCE-SYNTHETIC-OPS-AOC-RESTORED-V3"
	const restoredSourceHash = "sha256:7777777777777777777777777777777777777777777777777777777777777777"
	if _, err := governance.Pool.Exec(ctx, `
		INSERT INTO regulatory_source_versions
			(id,source_identity,version_identity,title,source_class,source_status,source_locator,source_hash,effective_from,source_metadata)
		VALUES
			($1,'SYNTHETIC-OPS-AOC','3','Synthetic same-day currentness fixture','STATE_COMPLIANCE_CROSSWALK','SUPPLIED_WORKING_COPY','Synthetic same-day source',$2,'2025-01-01','{"testProfileOnly":true,"sameDay":true}')`,
		restoredSourceID, restoredSourceHash,
	); err != nil {
		t.Fatal(err)
	}
	// V2 is present but unactivated. V3 has the same effective date as the
	// baseline; the exact V1 predecessor remains the only valid head.
	activation, err := admin.ActivateSourceCurrentness(ctx, regulatoryRefreshAdmin(), regulatory.SourceCurrentnessActivationCommand{
		OperationID: "REFRESH-T6-CURRENTNESS-SAME-DAY-V3", IdempotencyKey: "REFRESH-T6-CURRENTNESS-SAME-DAY-V3",
		CurrentSourceSnapshotID: restoredSourceID, CurrentSourceHash: restoredSourceHash,
		PreviousSourceSnapshotID: base.CurrentSourceSnapshotID, PreviousSourceHash: base.CurrentSourceHash,
		Reason: "Prove that exact immutable predecessor identity, not effective date, controls source currentness.",
	})
	if err != nil || activation.Status != "IMPACT_REVIEW_DRAFT" || activation.ImpactReviewDraftID == nil {
		t.Fatalf("same-day V3 activation=%+v err=%v", activation, err)
	}
	var sequence int
	if err := governance.Pool.QueryRow(ctx, `
		SELECT sequence_id FROM regulatory_source_currentness_events
		WHERE event_id=$1 AND previous_source_version_id=$2 AND current_source_version_id=$3`,
		activation.EventID, base.CurrentSourceSnapshotID, restoredSourceID,
	).Scan(&sequence); err != nil || sequence != 2 {
		t.Fatalf("exact same-day source-currentness ledger sequence=%d err=%v", sequence, err)
	}
	if _, err := admin.ActivateSourceCurrentness(ctx, regulatoryRefreshAdmin(), syntheticImpactCurrentnessCommand("REFRESH-T6-CURRENTNESS-WRONG-PREDECESSOR")); !errors.Is(err, application.ErrInvalid) {
		t.Fatalf("activation with V1 rather than exact V3 head error=%v, want invalid", err)
	}
	if _, err := admin.ActivateSourceCurrentness(ctx, regulatoryRefreshAdmin(), regulatory.SourceCurrentnessActivationCommand{
		OperationID: "REFRESH-T6-CURRENTNESS-HISTORICAL-REACTIVATION", IdempotencyKey: "REFRESH-T6-CURRENTNESS-HISTORICAL-REACTIVATION",
		CurrentSourceSnapshotID: base.CurrentSourceSnapshotID, CurrentSourceHash: base.CurrentSourceHash,
		PreviousSourceSnapshotID: restoredSourceID, PreviousSourceHash: restoredSourceHash,
		Reason: "An old snapshot must not become current again under a new event.",
	}); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("historical snapshot reactivation error=%v, want conflict", err)
	}
	var events, drafts int
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_source_currentness_events WHERE source_identity='SYNTHETIC-OPS-AOC'`).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_source_impact_review_drafts WHERE source_identity='SYNTHETIC-OPS-AOC'`).Scan(&drafts); err != nil {
		t.Fatal(err)
	}
	if events != 2 || drafts != 1 {
		t.Fatalf("linear immutable currentness graph events=%d drafts=%d, want 2/1", events, drafts)
	}
}

// An identical activation that races before the first transaction commits is
// a read of one immutable receipt, not a competing transition or duplicate
// impact-review Draft.
func TestRegulatorySourceRefreshTask6ConcurrentIdenticalSourceActivationReplaysOneReceipt(t *testing.T) {
	ctx := context.Background()
	governance, _, _ := task6SubmittedCandidate(t, "regulatory_refresh_currentness_idempotency")
	admin := regulatory.NewAdminService(governance.Pool, governance.Clock)
	command := syntheticImpactCurrentnessCommand("REFRESH-T6-CURRENTNESS-CONCURRENT-REPLAY")
	type result struct {
		view regulatory.SourceCurrentnessActivationView
		err  error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	for range 2 {
		go func() {
			<-start
			view, err := admin.ActivateSourceCurrentness(ctx, regulatoryRefreshAdmin(), command)
			results <- result{view: view, err: err}
		}()
	}
	close(start)
	first, second := <-results, <-results
	if first.err != nil || second.err != nil || first.view.EventID == "" || first.view.EventID != second.view.EventID || first.view.ImpactReviewDraftID == nil || second.view.ImpactReviewDraftID == nil || *first.view.ImpactReviewDraftID != *second.view.ImpactReviewDraftID {
		t.Fatalf("concurrent activation results first=%+v/%v second=%+v/%v", first.view, first.err, second.view, second.err)
	}
	var events, drafts, audits int
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_source_currentness_events WHERE operation_id=$1`, command.OperationID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM regulatory_source_impact_review_drafts WHERE current_source_version_id=$1`, command.CurrentSourceSnapshotID).Scan(&drafts); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE operation_id=$1`, command.OperationID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if events != 1 || drafts != 1 || audits != 1 {
		t.Fatalf("concurrent activation effects events=%d drafts=%d audits=%d, want 1/1/1", events, drafts, audits)
	}
}

// A source impact is shared evidence about an immutable predecessor/current
// transition. More than one independent candidate root may need review
// against it; each candidate/run remains one-to-one with its own link.
func TestRegulatorySourceRefreshTask6ImpactDraftCanLinkMultipleCandidateRoots(t *testing.T) {
	ctx := context.Background()
	governance, _, _ := task6SubmittedCandidate(t, "regulatory_refresh_multi_impact_links")
	admin := regulatory.NewAdminService(governance.Pool, governance.Clock)
	activation := activateSyntheticImpactCurrentness(t, ctx, admin, "REFRESH-T6-MULTI-IMPACT-SOURCE-CURRENTNESS")
	first, err := admin.Import(ctx, regulatoryRefreshAdmin(), "REFRESH-T6-MULTI-IMPACT-FIRST", "REFRESH-T6-MULTI-IMPACT-FIRST", regulatory.SyntheticImpactCandidateBundle())
	if err != nil || first.Candidate == nil || activation.ImpactReviewDraftID == nil {
		t.Fatalf("first impact candidate=%+v activation=%+v err=%v", first, activation, err)
	}
	second, err := admin.Import(ctx, regulatoryRefreshAdmin(), "REFRESH-T6-MULTI-IMPACT-SECOND", "REFRESH-T6-MULTI-IMPACT-SECOND", regulatory.SyntheticHybridReconciledCandidateBundle())
	if err != nil || second.Candidate == nil {
		t.Fatalf("second independently imported impact candidate=%+v err=%v", second, err)
	}
	if first.Candidate.CandidateID == second.Candidate.CandidateID || first.GenerationRunID == second.GenerationRunID {
		t.Fatalf("impact candidates must remain distinct immutable roots first=%+v second=%+v", first, second)
	}
	var links int
	if err := governance.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM regulatory_source_impact_candidate_links
		WHERE impact_review_draft_id=$1`, *activation.ImpactReviewDraftID,
	).Scan(&links); err != nil || links != 2 {
		t.Fatalf("impact Draft link cardinality=%d err=%v, want two independent candidate roots", links, err)
	}
	var distinctCandidates, distinctRuns int
	if err := governance.Pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT candidate_draft_version_id),COUNT(DISTINCT generation_run_id)
		FROM regulatory_source_impact_candidate_links
		WHERE impact_review_draft_id=$1`, *activation.ImpactReviewDraftID,
	).Scan(&distinctCandidates, &distinctRuns); err != nil || distinctCandidates != 2 || distinctRuns != 2 {
		t.Fatalf("impact Draft links candidates=%d runs=%d err=%v", distinctCandidates, distinctRuns, err)
	}
}

// Break caught: source staleness blocked a candidate's publication but still
// allowed a newly executable Audit package to snapshot the old published
// version. An existing package remains immutable, but a fresh package must
// fail closed once an impact-review Draft records the newer source chain.
func TestRegulatorySourceRefreshTask6StalePublishedVersionCannotMaterializeNewAuditPackage(t *testing.T) {
	ctx := context.Background()
	governance, manager, historical := task6SubmittedCandidate(t, "regulatory_refresh_stale_package")
	approved, err := governance.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "REFRESH-T6-STALE-PACKAGE-APPROVE", IdempotencyKey: "REFRESH-T6-STALE-PACKAGE-APPROVE",
		CandidateID: historical.CandidateID, ExpectedRevision: historical.Revision,
		ExpectedContentDigest: historical.ContentDigest, Reason: "Approve the historical synthetic version before the source change.",
	})
	if err != nil || approved.Status != "TECHNICALLY_APPROVED" {
		t.Fatalf("approve historical candidate=%+v err=%v", approved, err)
	}
	if _, err := governance.Publish(ctx, manager, checklistgovernance.PublicationCommand{
		OperationID: "REFRESH-T6-STALE-PACKAGE-PUBLISH", IdempotencyKey: "REFRESH-T6-STALE-PACKAGE-PUBLISH",
		CandidateID: approved.CandidateID, ExpectedRevision: approved.Revision,
		ExpectedContentDigest: approved.ContentDigest, Reason: "Publish the historical synthetic version before the source change.",
	}); err != nil {
		t.Fatalf("publish historical candidate=%v", err)
	}
	admin := regulatory.NewAdminService(governance.Pool, governance.Clock)
	activateSyntheticImpactCurrentness(t, ctx, admin, "REFRESH-T6-STALE-PACKAGE-SOURCE-CURRENTNESS")
	if _, err := admin.Import(ctx, regulatoryRefreshAdmin(), "REFRESH-T6-STALE-PACKAGE-IMPACT", "REFRESH-T6-STALE-PACKAGE-IMPACT", regulatory.SyntheticImpactCandidateBundle()); err != nil {
		t.Fatalf("import current-source impact Draft=%v", err)
	}
	if _, err := governance.Pool.Exec(ctx, `
		INSERT INTO identity_references (subject_id,issuer,display_name)
		VALUES ('USR-REFRESH-T6-STALE-INSPECTOR','refresh-task6','Refresh Task 6 Stale Inspector');
		INSERT INTO inspections (id,organization_id,assigned_inspector_subject_id,title,inspection_type,status)
		VALUES ('INSP-REFRESH-T6-STALE','ORG-SYNTHETIC-AOC','USR-REFRESH-T6-STALE-INSPECTOR','Refresh Task 6 stale-source package','RAMP_INSPECTION','PREPARATION')`); err != nil {
		t.Fatal(err)
	}
	_, err = governance.MaterializeApplicablePublishedPackage(ctx, manager, checklistgovernance.MaterializeApplicablePublishedPackageCommand{
		OperationID: "REFRESH-T6-STALE-PACKAGE-MATERIALIZE", IdempotencyKey: "REFRESH-T6-STALE-PACKAGE-MATERIALIZE", CorrelationID: "REFRESH-T6-STALE-PACKAGE-MATERIALIZE",
		InspectionID: "INSP-REFRESH-T6-STALE", PackageID: "PKG-REFRESH-T6-STALE", PackageVersion: 1,
		ExpiresAt: time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC),
		Selection: checklistgovernance.PublishedChecklistSelectionRequest{
			OrganizationID: "ORG-SYNTHETIC-AOC", InspectionType: "RAMP_INSPECTION", TargetID: "TARGET-SYNTHETIC-AOC", TargetKind: "ORGANIZATION", DepartmentID: "FLIGHT_OPERATIONS_INSPECTORATE", At: time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC),
		},
		AssignedInspectorSubjectIDs: map[string][]string{"Q-SYNTHETIC-OPS-AOC-001": {"USR-REFRESH-T6-STALE-INSPECTOR"}},
	})
	requireRegulatoryRefreshIssue(t, err, "STALE_SOURCE_TRACE")
	var packages, checklists, audits int
	var inspectionStatus string
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM inspection_packages WHERE inspection_id='INSP-REFRESH-T6-STALE'`).Scan(&packages); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM inspection_checklists WHERE inspection_id='INSP-REFRESH-T6-STALE'`).Scan(&checklists); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE operation_id='REFRESH-T6-STALE-PACKAGE-MATERIALIZE'`).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT status FROM inspections WHERE id='INSP-REFRESH-T6-STALE'`).Scan(&inspectionStatus); err != nil {
		t.Fatal(err)
	}
	if packages != 0 || checklists != 0 || audits != 0 || inspectionStatus != "PREPARATION" {
		t.Fatalf("stale package materialization effects packages=%d checklists=%d audits=%d inspectionStatus=%q", packages, checklists, audits, inspectionStatus)
	}
}

// Break caught: publication could validate source currentness immediately
// before a source-impact import committed. The source-change and publication
// paths must serialize on the same source identity: the publication waits for
// the impact event, then observes the stale trace and leaves no publication
// records behind.
func TestRegulatorySourceRefreshTask6SourceImpactSerializesWithPublication(t *testing.T) {
	ctx := context.Background()
	governance, manager, candidate := task6SubmittedCandidate(t, "regulatory_refresh_publish_race")
	approved, err := governance.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "REFRESH-T6-RACE-APPROVE", IdempotencyKey: "REFRESH-T6-RACE-APPROVE",
		CandidateID: candidate.CandidateID, ExpectedRevision: candidate.Revision,
		ExpectedContentDigest: candidate.ContentDigest, Reason: "Approve before the controlled source-impact transaction.",
	})
	if err != nil || approved.Status != "TECHNICALLY_APPROVED" {
		t.Fatalf("approve candidate=%+v err=%v", approved, err)
	}

	impactTx, err := governance.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer impactTx.Rollback(ctx)
	if _, err := impactTx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", regulatory.SourceCurrentnessLockKey("SYNTHETIC-OPS-AOC")); err != nil {
		t.Fatal(err)
	}
	admin := regulatory.NewAdminService(governance.Pool, governance.Clock)
	type activationResult struct {
		activation regulatory.SourceCurrentnessActivationView
		err        error
	}
	activationResultCh := make(chan activationResult, 1)
	go func() {
		activation, activationErr := admin.ActivateSourceCurrentness(ctx, regulatoryRefreshAdmin(), syntheticImpactCurrentnessCommand("REFRESH-T6-RACE-SOURCE-CURRENTNESS"))
		activationResultCh <- activationResult{activation: activation, err: activationErr}
	}()
	waitForSourceCurrentnessLockWaiter(t, ctx, governance.Pool)

	publishResult := make(chan error, 1)
	go func() {
		_, publishErr := governance.Publish(ctx, manager, checklistgovernance.PublicationCommand{
			OperationID: "REFRESH-T6-RACE-PUBLISH", IdempotencyKey: "REFRESH-T6-RACE-PUBLISH",
			CandidateID: approved.CandidateID, ExpectedRevision: approved.Revision,
			ExpectedContentDigest: approved.ContentDigest, Reason: "Publication must wait for the source-impact decision.",
		})
		publishResult <- publishErr
	}()

	select {
	case publishErr := <-publishResult:
		t.Fatalf("publication completed while the source-impact lock was held: %v", publishErr)
	case <-time.After(250 * time.Millisecond):
	}

	if err := impactTx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	impactTx = nil
	activation := <-activationResultCh
	if activation.err != nil || activation.activation.Status != "IMPACT_REVIEW_DRAFT" || activation.activation.ImpactReviewDraftID == nil {
		t.Fatalf("source-currentness activation after shared lock=%+v err=%v", activation.activation, activation.err)
	}

	requireRegulatoryRefreshIssue(t, <-publishResult, "STALE_SOURCE_TRACE")
	var publicationDecisions, templateVersions, publishAudits int
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM checklist_publication_decisions WHERE candidate_draft_version_id=$1`, approved.CandidateID).Scan(&publicationDecisions); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM checklist_template_versions WHERE candidate_draft_version_id=$1`, approved.CandidateID).Scan(&templateVersions); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE operation_id='REFRESH-T6-RACE-PUBLISH'`).Scan(&publishAudits); err != nil {
		t.Fatal(err)
	}
	if publicationDecisions != 0 || templateVersions != 0 || publishAudits != 0 {
		t.Fatalf("serialized stale publication persisted decisions=%d templates=%d audits=%d", publicationDecisions, templateVersions, publishAudits)
	}
}

// Break caught: a hybrid reconciliation could overwrite an old published
// checklist or its in-progress Audit package. A reconciliation instead makes a
// separate current-source Draft with visible delta data, then requires the
// normal technical-approval and publication decisions for future use.
func TestRegulatorySourceRefreshTask6HybridReconciliationPreservesHistoricalArtifacts(t *testing.T) {
	ctx := context.Background()
	governance, manager, historical := task6SubmittedCandidate(t, "regulatory_refresh_hybrid")
	approved, err := governance.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "REFRESH-T6-HISTORICAL-APPROVE", IdempotencyKey: "REFRESH-T6-HISTORICAL-APPROVE",
		CandidateID: historical.CandidateID, ExpectedRevision: historical.Revision,
		ExpectedContentDigest: historical.ContentDigest, Reason: "Publish the historical synthetic version before reconciliation.",
	})
	if err != nil {
		t.Fatal(err)
	}
	historicalPublication, err := governance.Publish(ctx, manager, checklistgovernance.PublicationCommand{
		OperationID: "REFRESH-T6-HISTORICAL-PUBLISH", IdempotencyKey: "REFRESH-T6-HISTORICAL-PUBLISH",
		CandidateID: approved.CandidateID, ExpectedRevision: approved.Revision,
		ExpectedContentDigest: approved.ContentDigest, Reason: "Separate historical publication decision.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := governance.Pool.Exec(ctx, `
		INSERT INTO identity_references (subject_id,issuer,display_name) VALUES ('USR-REFRESH-T6-INSPECTOR','refresh-task6','Refresh Task 6 Inspector');
		INSERT INTO inspections (id,organization_id,assigned_inspector_subject_id,title,inspection_type,status) VALUES ('INSP-REFRESH-T6','ORG-SYNTHETIC-AOC','USR-REFRESH-T6-INSPECTOR','Refresh Task 6 historical Audit','RAMP_INSPECTION','PREPARATION')`); err != nil {
		t.Fatal(err)
	}
	_, err = governance.MaterializeApplicablePublishedPackage(ctx, manager, checklistgovernance.MaterializeApplicablePublishedPackageCommand{
		OperationID: "REFRESH-T6-HISTORICAL-MATERIALIZE", IdempotencyKey: "REFRESH-T6-HISTORICAL-MATERIALIZE", CorrelationID: "REFRESH-T6-HISTORICAL-MATERIALIZE",
		InspectionID: "INSP-REFRESH-T6", PackageID: "PKG-REFRESH-T6", PackageVersion: 1,
		ExpiresAt:                   time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC),
		Selection:                   checklistgovernance.PublishedChecklistSelectionRequest{OrganizationID: "ORG-SYNTHETIC-AOC", InspectionType: "RAMP_INSPECTION", TargetID: "TARGET-SYNTHETIC-AOC", TargetKind: "ORGANIZATION", DepartmentID: "FLIGHT_OPERATIONS_INSPECTORATE", At: time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)},
		AssignedInspectorSubjectIDs: map[string][]string{"Q-SYNTHETIC-OPS-AOC-001": {"USR-REFRESH-T6-INSPECTOR"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var historicalVersion, historicalPackage []byte
	if err := governance.Pool.QueryRow(ctx, `SELECT snapshot::text FROM checklist_template_versions WHERE id=$1`, historicalPublication.TemplateVersionID).Scan(&historicalVersion); err != nil {
		t.Fatal(err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT snapshot::text FROM inspection_packages WHERE id='PKG-REFRESH-T6'`).Scan(&historicalPackage); err != nil {
		t.Fatal(err)
	}

	admin := regulatory.NewAdminService(governance.Pool, governance.Clock)
	legacy, err := admin.Import(ctx, regulatoryRefreshAdmin(), "REFRESH-T6-HYBRID-LEGACY", "REFRESH-T6-HYBRID-LEGACY", regulatory.SyntheticLegacyChecklistCandidateBundle())
	if err != nil || legacy.Candidate == nil || legacy.Candidate.Questions[0].Origin != regulatory.ExistingChecklistCandidateOrigin || legacy.Candidate.Questions[0].RegulatoryTrace.State != regulatory.SourceMappingRequired {
		t.Fatalf("legacy candidate=%+v err=%v", legacy, err)
	}
	activateSyntheticImpactCurrentness(t, ctx, admin, "REFRESH-T6-HYBRID-SOURCE-CURRENTNESS")
	hybridRun, err := admin.Import(ctx, regulatoryRefreshAdmin(), "REFRESH-T6-HYBRID-IMPORT", "REFRESH-T6-HYBRID-IMPORT", regulatory.SyntheticHybridReconciledCandidateBundle())
	if err != nil || hybridRun.Candidate == nil || hybridRun.Candidate.CandidateID == historical.CandidateID || hybridRun.Candidate.CandidateRootID != hybridRun.Candidate.CandidateID || hybridRun.Candidate.Status != regulatory.GeneratedDraft {
		t.Fatalf("hybrid Draft=%+v err=%v", hybridRun, err)
	}
	hybrid := hybridRun.Candidate
	if len(hybrid.Questions) != 1 || hybrid.Questions[0].Origin != regulatory.HybridReconciledOrigin || hybrid.Questions[0].Reconciliation == nil || hybrid.Questions[0].RegulatoryTrace.SourceIdentity != "SYNTHETIC-OPS-AOC" || hybrid.Questions[0].RegulatoryTrace.SHA256 == historical.Questions[0].RegulatoryTrace.SHA256 {
		t.Fatalf("hybrid did not preserve source-authority boundary: %+v", hybrid)
	}
	submitted, err := admin.Submit(ctx, regulatoryRefreshAdmin(), regulatory.SubmitCommand{
		OperationID: "REFRESH-T6-HYBRID-SUBMIT", IdempotencyKey: "REFRESH-T6-HYBRID-SUBMIT",
		CandidateID: hybrid.CandidateID, ExpectedRevision: hybrid.Revision,
		ExpectedContentDigest: hybrid.ContentDigest, Reason: "Submit the separate reconciled Draft for technical review.",
	})
	if err != nil || submitted.Status != "DEPARTMENT_REVIEW" {
		t.Fatalf("submit hybrid=%+v err=%v", submitted, err)
	}
	hybridApproved, err := governance.Approve(ctx, manager, checklistgovernance.ReviewCommand{
		OperationID: "REFRESH-T6-HYBRID-APPROVE", IdempotencyKey: "REFRESH-T6-HYBRID-APPROVE",
		CandidateID: submitted.CandidateID, ExpectedRevision: submitted.Revision,
		ExpectedContentDigest: submitted.ContentDigest, Reason: "Department Manager technical approval for the new reconciled candidate.",
	})
	if err != nil || hybridApproved.Status != "TECHNICALLY_APPROVED" {
		t.Fatalf("approve hybrid=%+v err=%v", hybridApproved, err)
	}
	if _, err := governance.Publish(ctx, manager, checklistgovernance.PublicationCommand{
		OperationID: "REFRESH-T6-HYBRID-PUBLISH", IdempotencyKey: "REFRESH-T6-HYBRID-PUBLISH",
		CandidateID: hybridApproved.CandidateID, ExpectedRevision: hybridApproved.Revision,
		ExpectedContentDigest: hybridApproved.ContentDigest, Reason: "Separate publication decision for the reconciled candidate.",
	}); err != nil {
		t.Fatalf("publish hybrid=%v", err)
	}
	var afterVersion, afterPackage []byte
	if err := governance.Pool.QueryRow(ctx, `SELECT snapshot::text FROM checklist_template_versions WHERE id=$1`, historicalPublication.TemplateVersionID).Scan(&afterVersion); err != nil || string(afterVersion) != string(historicalVersion) {
		t.Fatalf("historical published version mutated err=%v", err)
	}
	if err := governance.Pool.QueryRow(ctx, `SELECT snapshot::text FROM inspection_packages WHERE id='PKG-REFRESH-T6'`).Scan(&afterPackage); err != nil || string(afterPackage) != string(historicalPackage) {
		t.Fatalf("historical in-progress Audit bytes mutated err=%v", err)
	}
}
