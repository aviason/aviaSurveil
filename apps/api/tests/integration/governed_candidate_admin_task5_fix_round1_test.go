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
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/regulatory"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/testprofile"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

// Break caught: ListSources must read persisted partition roles/row identities,
// applicability facts, and source gaps instead of fabricating empty strings or
// conflating partitions with generation runs.
func TestTask5FixRound1ProjectsExactPersistedSourceFacts(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "task5_source_projection")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migration 21: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap synthetic inputs: %v", err)
	}
	if err := testprofile.BootstrapBlockedRealOPSAOCGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap blocked real inputs: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ('USR-TASK5-PROJECTION', 'task5-test', 'Task 5 Projection Admin')`); err != nil {
		t.Fatalf("seed Admin identity: %v", err)
	}
	admin := identity.Principal{SubjectID: "USR-TASK5-PROJECTION", Roles: []identity.Role{identity.RoleAdmin}}
	service := regulatory.NewAdminService(pool, nil)
	bundle := regulatory.SyntheticCandidateBundle()
	if _, err := service.Import(ctx, admin, "TASK5-PROJECTION-IMPORT", "TASK5-PROJECTION-IMPORT", bundle); err != nil {
		t.Fatalf("import synthetic candidate: %v", err)
	}

	sources, err := service.ListSources(ctx, admin)
	if err != nil {
		t.Fatalf("list persisted source facts: %v", err)
	}
	var input, holdout, blocked *regulatory.SourceSnapshotView
	for index := range sources {
		source := &sources[index]
		switch source.ClauseID {
		case "CLAUSE-SYNTHETIC-OPS-AOC-1":
			input = source
		case "CLAUSE-SYNTHETIC-OPS-AOC-HOLDOUT-1":
			holdout = source
		case "NCAA-CC-A610-4.2.2.2":
			blocked = source
		}
	}
	if input == nil || len(input.Partitions) != 1 || input.Partitions[0] != (regulatory.SourcePartitionFactView{
		EvaluationID: "EVAL-SYNTHETIC-OPS-AOC", PartitionID: "PARTITION-SYNTHETIC-INPUT",
		Role: "GENERATION_INPUT", CrosswalkRowID: "CCROW-SYNTHETIC-OPS-AOC-1",
		StableRowIdentity: "CC:SYNTHETIC:OPS:AOC:1",
	}) {
		t.Fatalf("generation-input source projection is not exact: %+v", input)
	}
	if holdout == nil || len(holdout.Partitions) != 1 || holdout.Partitions[0].Role != "BLIND_HOLDOUT" || holdout.Partitions[0].PartitionID != "PARTITION-SYNTHETIC-HOLDOUT" || holdout.Partitions[0].StableRowIdentity != "CC:SYNTHETIC:OPS:AOC:HOLDOUT:1" {
		t.Fatalf("blind-holdout source projection is not exact: %+v", holdout)
	}
	if len(input.ApplicabilityFacts) != 1 || input.ApplicabilityFacts[0].CandidateID != bundle.CandidateBundleID || input.ApplicabilityFacts[0].MappingID != "MAP-SYNTHETIC-OPS-AOC-001" || input.ApplicabilityFacts[0].Relationship != "ADDRESSES" || input.ApplicabilityFacts[0].Applicability != "DIRECT" || input.ApplicabilityFacts[0].SourceGap != nil {
		t.Fatalf("stored mapping applicability projection is not exact: %+v", input.ApplicabilityFacts)
	}
	if blocked == nil || len(blocked.UnresolvedGaps) != 4 {
		t.Fatalf("stored blocked-source gaps are missing: %+v", blocked)
	}
	for index, want := range []regulatory.UnresolvedSourceGap{
		{GapID: "CONTROLLED_PROCEDURE", Reason: "The controlled NCAA Operations surveillance/ramp-inspection procedure has not been supplied."},
		{GapID: "PART_140_AUTHORITY", Reason: "Current Part 140 authority and supersession require source-owner confirmation."},
		{GapID: "PART_127_APPLICABILITY", Reason: "Exact Part 127 operation/configuration applicability requires Department Manager confirmation."},
		{GapID: "AMBIGUOUS_OWNERSHIP", Reason: "Exact source ownership and controlled-procedure stewardship remain unresolved."},
	} {
		if blocked.UnresolvedGaps[index] != want {
			t.Fatalf("blocked-source gap[%d] = %+v want %+v", index, blocked.UnresolvedGaps[index], want)
		}
	}
}

// Break caught: expected Task 5 absence, denial, validation, and conflict
// branches must be stable HTTP semantics backed by the real PostgreSQL state.
func TestTask5FixRound1RealPostgreSQLHTTPBehaviorAndStructuredValidation(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "task5_http_behavior")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migration 21: %v", err)
	}
	now := time.Date(2026, 7, 29, 9, 30, 0, 0, time.UTC)
	if err := testprofile.Reset(ctx, pool, now); err != nil {
		t.Fatalf("reset canonical profile: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap synthetic inputs: %v", err)
	}
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{Pool: pool, Clock: func() time.Time { return now }})
	handler := httpapi.NewCanonicalTestBoundary("task5-fix-token").Protect(api.Handler())
	request := func(method, path, subject string, body any) *httptest.ResponseRecorder {
		var encoded []byte
		if body != nil {
			encoded, _ = json.Marshal(body)
		}
		httpRequest := httptest.NewRequest(method, path, bytes.NewReader(encoded))
		httpRequest.Header.Set(httpapi.CanonicalTestTokenHeader, "task5-fix-token")
		httpRequest.Header.Set(httpapi.CanonicalTestSubjectHeader, subject)
		if body != nil {
			httpRequest.Header.Set("Content-Type", "application/json")
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httpRequest)
		return response
	}
	unauthenticated := httptest.NewRequest(http.MethodGet, "/v1/admin/governed-checklist/sources", nil)
	unauthenticatedResponse := httptest.NewRecorder()
	handler.ServeHTTP(unauthenticatedResponse, unauthenticated)
	if unauthenticatedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d body=%s", unauthenticatedResponse.Code, unauthenticatedResponse.Body.String())
	}
	if response := request(http.MethodGet, "/v1/admin/governed-checklist/sources", "USR-INSPECTOR-DAVID", nil); response.Code != http.StatusForbidden {
		t.Fatalf("non-Admin source read status=%d body=%s", response.Code, response.Body.String())
	}
	if response := request(http.MethodGet, "/v1/admin/governed-checklist/generation-runs/ABSENT", "USR-ADMIN-ADA", nil); response.Code != http.StatusNotFound {
		t.Fatalf("absent run status=%d body=%s", response.Code, response.Body.String())
	}
	if response := request(http.MethodGet, "/v1/admin/governed-checklist/candidates/ABSENT", "USR-ADMIN-ADA", nil); response.Code != http.StatusNotFound {
		t.Fatalf("absent candidate status=%d body=%s", response.Code, response.Body.String())
	}

	bundle := regulatory.SyntheticCandidateBundle()
	importBody := map[string]any{"operationId": "TASK5-HTTP-IMPORT", "idempotencyKey": "TASK5-HTTP-IMPORT", "candidateBundle": bundle}
	importedResponse := request(http.MethodPost, "/v1/admin/governed-checklist/generation-runs", "USR-ADMIN-ADA", importBody)
	if importedResponse.Code != http.StatusCreated {
		t.Fatalf("import status=%d body=%s", importedResponse.Code, importedResponse.Body.String())
	}
	var imported regulatory.GenerationRunView
	if err := json.Unmarshal(importedResponse.Body.Bytes(), &imported); err != nil || imported.Candidate == nil {
		t.Fatalf("decode imported run: %+v err=%v body=%s", imported, err, importedResponse.Body.String())
	}
	replayedResponse := request(http.MethodPost, "/v1/admin/governed-checklist/generation-runs", "USR-ADMIN-ADA", importBody)
	if replayedResponse.Code != http.StatusCreated || replayedResponse.Body.String() != importedResponse.Body.String() {
		t.Fatalf("import replay changed response: first=%s replay=%d/%s", importedResponse.Body.String(), replayedResponse.Code, replayedResponse.Body.String())
	}
	conflictingBundle := bundle
	conflictingBundle.GenerationRunID = "GENRUN-TASK5-HTTP-CONFLICT"
	conflictBody := map[string]any{"operationId": "TASK5-HTTP-IMPORT", "idempotencyKey": "TASK5-HTTP-IMPORT", "candidateBundle": conflictingBundle}
	if response := request(http.MethodPost, "/v1/admin/governed-checklist/generation-runs", "USR-ADMIN-ADA", conflictBody); response.Code != http.StatusConflict {
		t.Fatalf("conflicting import status=%d body=%s", response.Code, response.Body.String())
	}

	candidate := *imported.Candidate
	mappings := cloneMappings(candidate.Mappings)
	mappings[0].Rationale = "Synthetic test-profile rationale reviewed by an Admin without changing the controlled source claim."
	editBody := map[string]any{
		"operationId": "TASK5-HTTP-EDIT", "idempotencyKey": "TASK5-HTTP-EDIT",
		"candidateId": candidate.CandidateID, "expectedRevision": candidate.Revision,
		"expectedContentDigest": candidate.ContentDigest, "changeReason": "Apply the controlled synthetic edit.",
		"mappings": mappings, "questions": candidate.Questions, "requiredOwners": candidate.RequiredOwners,
	}
	pathMismatch := cloneJSONMap(editBody)
	pathMismatch["candidateId"] = "OTHER-CANDIDATE"
	if response := request(http.MethodPost, "/v1/admin/governed-checklist/candidates/"+candidate.CandidateID+"/revisions", "USR-ADMIN-ADA", pathMismatch); response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("path/body mismatch status=%d body=%s", response.Code, response.Body.String())
	}
	invalidBody := cloneJSONMap(editBody)
	invalidMappings := cloneMappings(mappings)
	invalidMappings[0].Citations[0].SourceHash = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	invalidBody["operationId"], invalidBody["idempotencyKey"], invalidBody["mappings"] = "TASK5-HTTP-INVALID", "TASK5-HTTP-INVALID", invalidMappings
	invalidResponse := request(http.MethodPost, "/v1/admin/governed-checklist/candidates/"+candidate.CandidateID+"/revisions", "USR-ADMIN-ADA", invalidBody)
	if invalidResponse.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid edit status=%d body=%s", invalidResponse.Code, invalidResponse.Body.String())
	}
	var validation struct {
		Issues []regulatory.ValidationIssue `json:"issues"`
	}
	if err := json.Unmarshal(invalidResponse.Body.Bytes(), &validation); err != nil || len(validation.Issues) != 1 || validation.Issues[0].FieldPath != "mappings[0].citations[0].sourceHash" {
		t.Fatalf("structured HTTP validation issue missing: %+v err=%v body=%s", validation, err, invalidResponse.Body.String())
	}
	editedResponse := request(http.MethodPost, "/v1/admin/governed-checklist/candidates/"+candidate.CandidateID+"/revisions", "USR-ADMIN-ADA", editBody)
	if editedResponse.Code != http.StatusCreated {
		t.Fatalf("valid edit status=%d body=%s", editedResponse.Code, editedResponse.Body.String())
	}
	var edited regulatory.CandidateView
	if err := json.Unmarshal(editedResponse.Body.Bytes(), &edited); err != nil || edited.SupersedesCandidateID == nil || *edited.SupersedesCandidateID != candidate.CandidateID || edited.Revision != 2 || edited.ContentDigest == candidate.ContentDigest {
		t.Fatalf("decode exact successor: %+v err=%v", edited, err)
	}
	staleBody := cloneJSONMap(editBody)
	staleBody["operationId"], staleBody["idempotencyKey"] = "TASK5-HTTP-STALE", "TASK5-HTTP-STALE"
	if response := request(http.MethodPost, "/v1/admin/governed-checklist/candidates/"+candidate.CandidateID+"/revisions", "USR-ADMIN-ADA", staleBody); response.Code != http.StatusConflict {
		t.Fatalf("stale edit status=%d body=%s", response.Code, response.Body.String())
	}
	ancestorSubmit := map[string]any{"operationId": "TASK5-HTTP-SUBMIT-ANCESTOR", "idempotencyKey": "TASK5-HTTP-SUBMIT-ANCESTOR", "candidateId": candidate.CandidateID, "expectedRevision": candidate.Revision, "expectedContentDigest": candidate.ContentDigest, "reason": "Ancestor submission must fail."}
	if response := request(http.MethodPost, "/v1/admin/governed-checklist/candidates/"+candidate.CandidateID+"/submissions", "USR-ADMIN-ADA", ancestorSubmit); response.Code != http.StatusConflict {
		t.Fatalf("ancestor submission status=%d body=%s", response.Code, response.Body.String())
	}
	submitBody := map[string]any{"operationId": "TASK5-HTTP-SUBMIT", "idempotencyKey": "TASK5-HTTP-SUBMIT", "candidateId": edited.CandidateID, "expectedRevision": edited.Revision, "expectedContentDigest": edited.ContentDigest, "reason": "Submit exact current leaf."}
	submittedResponse := request(http.MethodPost, "/v1/admin/governed-checklist/candidates/"+edited.CandidateID+"/submissions", "USR-ADMIN-ADA", submitBody)
	if submittedResponse.Code != http.StatusOK {
		t.Fatalf("leaf submission status=%d body=%s", submittedResponse.Code, submittedResponse.Body.String())
	}
	if response := request(http.MethodPost, "/v1/admin/governed-checklist/candidates/"+edited.CandidateID+"/submissions", "USR-ADMIN-ADA", submitBody); response.Code != http.StatusOK || response.Body.String() != submittedResponse.Body.String() {
		t.Fatalf("submission replay status=%d body=%s", response.Code, response.Body.String())
	}
	conflictingSubmit := cloneJSONMap(submitBody)
	conflictingSubmit["reason"] = "Conflicting retry."
	if response := request(http.MethodPost, "/v1/admin/governed-checklist/candidates/"+edited.CandidateID+"/submissions", "USR-ADMIN-ADA", conflictingSubmit); response.Code != http.StatusConflict {
		t.Fatalf("conflicting submission status=%d body=%s", response.Code, response.Body.String())
	}
}

func cloneJSONMap(input map[string]any) map[string]any {
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

// Break caught: command rows and linked audit rows must retain every exact
// identity/status field, and failed imports must remain inspectable without
// creating a candidate or any downstream workflow artifact.
func TestTask5FixRound1PersistsExactCommandAuditAndFailedRunEvidence(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "task5_command_audit")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migration 21: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap synthetic inputs: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ('USR-TASK5-EVIDENCE', 'task5-test', 'Task 5 Evidence Admin')`); err != nil {
		t.Fatalf("seed Admin identity: %v", err)
	}
	now := time.Date(2026, 7, 29, 10, 15, 0, 0, time.UTC)
	admin := identity.Principal{SubjectID: "USR-TASK5-EVIDENCE", OrganizationID: "CAA", Roles: []identity.Role{identity.RoleAdmin}}
	service := regulatory.NewAdminService(pool, func() time.Time { return now })
	bundle := regulatory.SyntheticCandidateBundle()
	run, err := service.Import(ctx, admin, "TASK5-EVIDENCE-IMPORT", "TASK5-EVIDENCE-IMPORT-KEY", bundle)
	if err != nil || run.Candidate == nil {
		t.Fatalf("import candidate: %+v err=%v", run, err)
	}
	mappings := cloneMappings(run.Candidate.Mappings)
	mappings[0].Rationale = "Synthetic test-profile rationale reviewed by an Admin without changing the controlled source claim."
	edit := regulatory.EditCommand{
		OperationID: "TASK5-EVIDENCE-EDIT", IdempotencyKey: "TASK5-EVIDENCE-EDIT-KEY",
		CandidateID: run.Candidate.CandidateID, ExpectedRevision: run.Candidate.Revision,
		ExpectedContentDigest: run.Candidate.ContentDigest, ChangeReason: "Apply the controlled synthetic edit.",
		Mappings: mappings, Questions: cloneQuestions(run.Candidate.Questions),
		RequiredOwners: append([]regulatory.RequiredOwner(nil), run.Candidate.RequiredOwners...),
	}
	edited, err := service.CreateRevision(ctx, admin, edit)
	if err != nil {
		t.Fatalf("create successor: %v", err)
	}
	submit := regulatory.SubmitCommand{
		OperationID: "TASK5-EVIDENCE-SUBMIT", IdempotencyKey: "TASK5-EVIDENCE-SUBMIT-KEY",
		CandidateID: edited.CandidateID, ExpectedRevision: edited.Revision,
		ExpectedContentDigest: edited.ContentDigest, Reason: "Submit exact current leaf.",
	}
	if _, err := service.Submit(ctx, admin, submit); err != nil {
		t.Fatalf("submit successor: %v", err)
	}

	wantCommands := []struct {
		kind, operationID, key, candidateID, digest, before, after string
		revision                                                   int64
	}{
		{"IMPORTED_GENERATION_RUN", "TASK5-EVIDENCE-IMPORT", "TASK5-EVIDENCE-IMPORT-KEY", bundle.CandidateBundleID, bundle.OutputDigest, "", "GENERATED_DRAFT", 1},
		{"REVISION_CREATED", "TASK5-EVIDENCE-EDIT", "TASK5-EVIDENCE-EDIT-KEY", edited.CandidateID, edited.ContentDigest, "GENERATED_DRAFT", "GENERATED_DRAFT", 2},
		{"DEPARTMENT_REVIEW_SUBMITTED", "TASK5-EVIDENCE-SUBMIT", "TASK5-EVIDENCE-SUBMIT-KEY", edited.CandidateID, edited.ContentDigest, "GENERATED_DRAFT", "DEPARTMENT_REVIEW", 2},
	}
	for _, want := range wantCommands {
		var kind, key, semantic, generationRunID, candidateID, digest, actor, reason, auditID string
		var revision int64
		var createdAt time.Time
		if err := pool.QueryRow(ctx, `SELECT command_kind,idempotency_key,semantic_payload_digest,generation_run_id,candidate_draft_version_id,candidate_revision,candidate_content_digest,actor_subject_id,reason,audit_event_id,created_at FROM governed_candidate_commands WHERE operation_id=$1`, want.operationID).Scan(&kind, &key, &semantic, &generationRunID, &candidateID, &revision, &digest, &actor, &reason, &auditID, &createdAt); err != nil {
			t.Fatalf("read command %s: %v", want.operationID, err)
		}
		if kind != want.kind || key != want.key || generationRunID != bundle.GenerationRunID || candidateID != want.candidateID || revision != want.revision || digest != want.digest || actor != admin.SubjectID || reason == "" || auditID != "AE-"+want.operationID || !createdAt.Equal(now) || len(semantic) != 71 {
			t.Fatalf("command %s is incomplete: kind=%s key=%s semantic=%s run=%s candidate=%s revision=%d digest=%s actor=%s reason=%q audit=%s at=%s", want.operationID, kind, key, semantic, generationRunID, candidateID, revision, digest, actor, reason, auditID, createdAt)
		}
		var action, entityType, entityID, before, after, auditReason, auditOperation string
		var entityVersion int64
		var occurredAt time.Time
		if err := pool.QueryRow(ctx, `SELECT action,entity_type,entity_id,entity_version,COALESCE(before_status,''),COALESCE(after_status,''),reason,operation_id,occurred_at FROM audit_events WHERE event_id=$1`, auditID).Scan(&action, &entityType, &entityID, &entityVersion, &before, &after, &auditReason, &auditOperation, &occurredAt); err != nil {
			t.Fatalf("read linked audit %s: %v", auditID, err)
		}
		if action != want.kind || entityType != "GOVERNED_CANDIDATE" || entityID != want.candidateID || entityVersion != want.revision || before != want.before || after != want.after || auditReason != reason || auditOperation != want.operationID || !occurredAt.Equal(now) {
			t.Fatalf("audit %s is incomplete: action=%s entity=%s/%s/%d before=%q after=%q reason=%q operation=%s at=%s", auditID, action, entityType, entityID, entityVersion, before, after, auditReason, auditOperation, occurredAt)
		}
	}

	beforeCommands, beforeAudits := exactCounts(t, pool, "governed_candidate_commands", "audit_events")
	if _, err := service.Import(ctx, admin, "TASK5-EVIDENCE-IMPORT", "TASK5-EVIDENCE-IMPORT-KEY", bundle); err != nil {
		t.Fatalf("identical import replay: %v", err)
	}
	if _, err := service.CreateRevision(ctx, admin, edit); err != nil {
		t.Fatalf("identical edit replay: %v", err)
	}
	if _, err := service.Submit(ctx, admin, submit); err != nil {
		t.Fatalf("identical submit replay: %v", err)
	}
	afterCommands, afterAudits := exactCounts(t, pool, "governed_candidate_commands", "audit_events")
	if beforeCommands != afterCommands || beforeAudits != afterAudits {
		t.Fatalf("identical replay added command/audit rows: before=%d/%d after=%d/%d", beforeCommands, beforeAudits, afterCommands, afterAudits)
	}

	failed := regulatory.SyntheticCandidateBundle()
	failed.ComplianceMappings[0].Rationale = "Unsupported failed-run content."
	if _, err := service.Import(ctx, admin, "TASK5-EVIDENCE-FAIL", "TASK5-EVIDENCE-FAIL-KEY", failed); !errors.Is(err, application.ErrInvalid) {
		t.Fatalf("invalid import error=%v", err)
	}
	failedRun, err := service.GetRun(ctx, admin, "GENRUN-FAILED-EVIDENCE-FAIL")
	if err != nil || failedRun.Status != "FAILED" || failedRun.RequestID != bundle.GenerationRequest.RequestID || failedRun.Failure == nil || failedRun.Failure.Code != "CANDIDATE_VALIDATION_FAILED" || failedRun.Failure.OperationID != "TASK5-EVIDENCE-FAIL" || failedRun.Failure.IdempotencyKey != "TASK5-EVIDENCE-FAIL-KEY" || failedRun.Candidate != nil {
		t.Fatalf("failed-run inspection is incomplete: %+v err=%v", failedRun, err)
	}
	for table, want := range map[string]int{"department_review_decisions": 0, "checklist_publication_decisions": 0, "checklist_template_versions": 0, "inspection_packages": 0} {
		var got int
		if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(&got); err != nil || got != want {
			t.Fatalf("failed import downstream %s=%d want=%d err=%v", table, got, want, err)
		}
	}
}

func exactCounts(t *testing.T, pool *database.Pool, tables ...string) (int, int) {
	t.Helper()
	counts := make([]int, len(tables))
	for index, table := range tables {
		if err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&counts[index]); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
	}
	return counts[0], counts[1]
}

// Break caught: a version-21 database may already contain governed history
// while derived Task 5 guards/indexes are missing or wrong. Forward repair must
// correct the complete Task 5 inventory without rewriting that history.
func TestTask5FixRound1Version21ForwardRepairCorrectsPartialObjectsAndPreservesHistory(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "task5_complete_repair")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply retained migration set: %v", err)
	}
	var migrationVersionBeforeRepair int64
	if err := pool.QueryRow(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&migrationVersionBeforeRepair); err != nil {
		t.Fatalf("read migration version before repair: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap synthetic inputs: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ('USR-TASK5-REPAIR', 'task5-test', 'Task 5 Repair Admin')`); err != nil {
		t.Fatalf("seed repair Admin identity: %v", err)
	}
	admin := identity.Principal{SubjectID: "USR-TASK5-REPAIR", OrganizationID: "CAA", Roles: []identity.Role{identity.RoleAdmin}}
	if _, err := regulatory.NewAdminService(pool, nil).Import(ctx, admin, "TASK5-REPAIR-IMPORT", "TASK5-REPAIR-IMPORT", regulatory.SyntheticCandidateBundle()); err != nil {
		t.Fatalf("seed governed command history: %v", err)
	}
	historyQueries := map[string]string{
		"source":    `SELECT jsonb_agg(to_jsonb(row) ORDER BY id)::text FROM regulatory_source_versions row`,
		"candidate": `SELECT jsonb_agg(to_jsonb(row) ORDER BY id)::text FROM template_draft_versions row`,
		"command":   `SELECT jsonb_agg(to_jsonb(row) ORDER BY id)::text FROM governed_candidate_commands row`,
		"audit":     `SELECT jsonb_agg(to_jsonb(row) ORDER BY event_id)::text FROM audit_events row`,
	}
	before := map[string]string{}
	for name, query := range historyQueries {
		var snapshot string
		if err := pool.QueryRow(ctx, query).Scan(&snapshot); err != nil {
			t.Fatalf("capture %s history: %v", name, err)
		}
		before[name] = snapshot
	}
	if _, err := pool.Exec(ctx, `
		DROP TRIGGER governed_candidate_commands_append_only ON governed_candidate_commands;
		DROP TRIGGER regulatory_source_gap_facts_append_only ON regulatory_source_gap_facts;
		DROP TRIGGER template_draft_versions_generated_immutable ON template_draft_versions;
		DROP INDEX regulatory_source_gap_facts_source_idx;
		CREATE INDEX regulatory_source_gap_facts_source_idx ON regulatory_source_gap_facts (gap_id);
		DROP INDEX governed_candidate_commands_run_candidate_idx;
		CREATE INDEX governed_candidate_commands_run_candidate_idx ON governed_candidate_commands (candidate_draft_version_id);
		ALTER TABLE governed_candidate_commands DROP CONSTRAINT governed_candidate_commands_command_kind_check;
		ALTER TABLE governed_candidate_commands ADD CONSTRAINT governed_candidate_commands_command_kind_check CHECK (command_kind = 'IMPORTED_GENERATION_RUN');
		CREATE OR REPLACE FUNCTION governed_generated_candidate_immutable_guard() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RETURN NEW; END;
		$$;
	`); err != nil {
		t.Fatalf("create partial/wrong Task 5 objects: %v", err)
	}
	if err := migrations.RepairRegulatoryChecklistGovernance(ctx, pool); err != nil {
		t.Fatalf("repair partial/wrong Task 5 objects: %v", err)
	}
	if err := migrations.RepairRegulatoryChecklistGovernance(ctx, pool); err != nil {
		t.Fatalf("idempotent complete Task 5 repair: %v", err)
	}
	for name, query := range historyQueries {
		var after string
		if err := pool.QueryRow(ctx, query).Scan(&after); err != nil || after != before[name] {
			t.Fatalf("repair changed %s history: before=%q after=%q err=%v", name, before[name], after, err)
		}
	}
	wantIndexes := map[string]string{
		"regulatory_source_gap_facts_source_idx":        "CREATE INDEX regulatory_source_gap_facts_source_idx ON public.regulatory_source_gap_facts USING btree (regulatory_source_version_id, ordinal, gap_id)",
		"governed_candidate_commands_run_candidate_idx": "CREATE INDEX governed_candidate_commands_run_candidate_idx ON public.governed_candidate_commands USING btree (generation_run_id, candidate_draft_version_id, candidate_revision)",
	}
	for name, want := range wantIndexes {
		var got string
		if err := pool.QueryRow(ctx, `SELECT pg_get_indexdef(to_regclass($1))`, "public."+name).Scan(&got); err != nil || got != want {
			t.Fatalf("repaired index %s=%q want=%q err=%v", name, got, want, err)
		}
	}
	for _, trigger := range []string{"governed_candidate_commands_append_only", "regulatory_source_gap_facts_append_only", "template_draft_versions_generated_immutable"} {
		var enabled string
		if err := pool.QueryRow(ctx, `SELECT tgenabled::text FROM pg_trigger WHERE tgname=$1 AND NOT tgisinternal`, trigger).Scan(&enabled); err != nil || enabled != "O" {
			t.Fatalf("repaired trigger %s enabled=%q err=%v", trigger, enabled, err)
		}
	}
	var functionDefinition, commandGuard string
	if err := pool.QueryRow(ctx, `SELECT pg_get_functiondef('governed_generated_candidate_immutable_guard()'::regprocedure)`).Scan(&functionDefinition); err != nil ||
		!strings.Contains(functionDefinition, "generated candidate revisions are immutable except governed status transitions") ||
		!strings.Contains(functionDefinition, "('GENERATED_DRAFT', 'DEPARTMENT_REVIEW')") {
		t.Fatalf("immutable guard was not repaired: %v definition=%q", err, functionDefinition)
	}
	if err := pool.QueryRow(ctx, `SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid='governed_candidate_commands'::regclass AND conname='governed_candidate_commands_command_kind_check'`).Scan(&commandGuard); err != nil || !strings.Contains(commandGuard, "REVISION_CREATED") || !strings.Contains(commandGuard, "DEPARTMENT_REVIEW_SUBMITTED") {
		t.Fatalf("command guard was not repaired: %v definition=%q", err, commandGuard)
	}
	var version int64
	if err := pool.QueryRow(ctx, `SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil || version != migrationVersionBeforeRepair {
		t.Fatalf("repair changed migration version=%d want=%d err=%v", version, migrationVersionBeforeRepair, err)
	}
}

// Break caught: every rejected edit must identify the exact field and frozen
// source lineage instead of returning one unstructured validation sentence.
func TestTask5FixRound1ReturnsStructuredFieldAndSourceValidationIssues(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "task5_validation_issues")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migration 21: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap synthetic inputs: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ('USR-TASK5-VALIDATION', 'task5-test', 'Task 5 Validation Admin')`); err != nil {
		t.Fatalf("seed Admin identity: %v", err)
	}
	admin := identity.Principal{SubjectID: "USR-TASK5-VALIDATION", Roles: []identity.Role{identity.RoleAdmin}}
	service := regulatory.NewAdminService(pool, nil)
	bundle := regulatory.SyntheticCandidateBundle()
	run, err := service.Import(ctx, admin, "TASK5-VALIDATION-IMPORT", "TASK5-VALIDATION-IMPORT", bundle)
	if err != nil || run.Candidate == nil {
		t.Fatalf("import synthetic candidate: %+v err=%v", run, err)
	}
	candidate := *run.Candidate
	base := regulatory.EditCommand{
		OperationID: "TASK5-VALIDATION-BASE", IdempotencyKey: "TASK5-VALIDATION-BASE",
		CandidateID: candidate.CandidateID, ExpectedRevision: candidate.Revision,
		ExpectedContentDigest: candidate.ContentDigest,
		ChangeReason:          "Apply the single controlled synthetic rationale alternative.",
		Mappings:              append([]regulatory.ComplianceMapping(nil), candidate.Mappings...),
		Questions:             append([]regulatory.ChecklistQuestion(nil), candidate.Questions...),
		RequiredOwners:        append([]regulatory.RequiredOwner(nil), candidate.RequiredOwners...),
	}
	base.Mappings[0].Rationale = "Synthetic test-profile rationale reviewed by an Admin without changing the controlled source claim."

	tests := []struct {
		name      string
		fieldPath string
		mutate    func(*regulatory.EditCommand)
	}{
		{name: "changed source hash", fieldPath: "mappings[0].citations[0].sourceHash", mutate: func(command *regulatory.EditCommand) {
			command.Mappings[0].Citations[0].SourceHash = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
		}},
		{name: "changed relationship", fieldPath: "mappings[0].relationship", mutate: func(command *regulatory.EditCommand) { command.Mappings[0].Relationship = "CONTEXT_ONLY" }},
		{name: "changed applicability", fieldPath: "mappings[0].applicability", mutate: func(command *regulatory.EditCommand) { command.Mappings[0].Applicability = "CONDITIONAL" }},
		{name: "duplicate mapping id", fieldPath: "mappings[1].mappingId", mutate: func(command *regulatory.EditCommand) {
			command.Mappings = append(command.Mappings, command.Mappings[0])
		}},
		{name: "question mapping mismatch", fieldPath: "questions[0].mappingIds[0]", mutate: func(command *regulatory.EditCommand) { command.Questions[0].MappingIDs[0] = "UNKNOWN-MAPPING" }},
		{name: "invalid allowed answers", fieldPath: "questions[0].allowedAnswers", mutate: func(command *regulatory.EditCommand) { command.Questions[0].AllowedAnswers = []string{"YES", "NO"} }},
		{name: "blank evidence", fieldPath: "questions[0].expectedEvidence[0]", mutate: func(command *regulatory.EditCommand) { command.Questions[0].ExpectedEvidence[0] = " " }},
		{name: "unsupported free text", fieldPath: "questions[0].prompt", mutate: func(command *regulatory.EditCommand) {
			command.Questions[0].Prompt = "Invent arbitrary regulatory prose."
		}},
		{name: "mandatory flag", fieldPath: "questions[0].mandatoryCore", mutate: func(command *regulatory.EditCommand) { command.Questions[0].MandatoryCore = false }},
		{name: "safety flag", fieldPath: "questions[0].safetyCritical", mutate: func(command *regulatory.EditCommand) { command.Questions[0].SafetyCritical = false }},
		{name: "source gap mismatch", fieldPath: "mappings[0].sourceGap", mutate: func(command *regulatory.EditCommand) {
			command.Mappings[0].SourceGap = &regulatory.SourceGap{Status: "UNRESOLVED", Reason: "Fabricated gap."}
		}},
		{name: "unknown owner", fieldPath: "requiredOwners[0].organizationalUnitId", mutate: func(command *regulatory.EditCommand) { command.RequiredOwners[0].OrganizationalUnitID = "UNKNOWN-UNIT" }},
		{name: "digest conflict", fieldPath: "expectedContentDigest", mutate: func(command *regulatory.EditCommand) {
			command.ExpectedContentDigest = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
		}},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := base
			command.OperationID = "TASK5-VALIDATION-" + test.name
			command.IdempotencyKey = command.OperationID
			command.Mappings = cloneMappings(base.Mappings)
			command.Questions = cloneQuestions(base.Questions)
			command.RequiredOwners = append([]regulatory.RequiredOwner(nil), base.RequiredOwners...)
			test.mutate(&command)
			_, err := service.CreateRevision(ctx, admin, command)
			var validation *regulatory.ValidationError
			if !errors.As(err, &validation) || len(validation.Issues) == 0 {
				t.Fatalf("case %d returned no structured issues: %v", index, err)
			}
			issue := validation.Issues[0]
			if issue.FieldPath != test.fieldPath || issue.SourceIdentity != "SYNTHETIC-OPS-AOC" || issue.SourceHash != "sha256:1111111111111111111111111111111111111111111111111111111111111111" || issue.ClauseID != "CLAUSE-SYNTHETIC-OPS-AOC-1" || issue.Locator != "Synthetic OPS/AOC 1" {
				t.Fatalf("case %d issue = %+v, want field=%s and exact source identity", index, issue, test.fieldPath)
			}
		})
	}
}

func cloneMappings(values []regulatory.ComplianceMapping) []regulatory.ComplianceMapping {
	cloned := make([]regulatory.ComplianceMapping, len(values))
	for index, value := range values {
		cloned[index] = value
		cloned[index].Citations = append([]regulatory.Citation(nil), value.Citations...)
	}
	return cloned
}

func cloneQuestions(values []regulatory.ChecklistQuestion) []regulatory.ChecklistQuestion {
	cloned := make([]regulatory.ChecklistQuestion, len(values))
	for index, value := range values {
		cloned[index] = value
		cloned[index].MappingIDs = append([]string(nil), value.MappingIDs...)
		cloned[index].Citations = append([]regulatory.Citation(nil), value.Citations...)
		cloned[index].ExpectedEvidence = append([]string(nil), value.ExpectedEvidence...)
		cloned[index].AllowedAnswers = append([]string(nil), value.AllowedAnswers...)
	}
	return cloned
}
