//go:build canonicaltest

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/httpapi"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/regulatory"
	"github.com/aviason/aviaSurveil/internal/testprofile"
	"github.com/aviason/aviaSurveil/migrations"
)

// Break caught: replay checked only before the candidate-root lock makes the
// second byte-identical overlapping caller observe stale mutable state and
// conflict instead of replaying the first caller's committed result.
func TestTask5FixRound3OverlappingIdenticalRootCommandsReplayOneEffect(t *testing.T) {
	cases := []struct {
		name               string
		expectedCandidates int
		run                func(context.Context, *regulatory.AdminService, identity.Principal, regulatory.CandidateView) (regulatory.CandidateView, error)
	}{
		{
			name:               "edit",
			expectedCandidates: 2,
			run: func(ctx context.Context, service *regulatory.AdminService, admin identity.Principal, candidate regulatory.CandidateView) (regulatory.CandidateView, error) {
				return service.CreateRevision(ctx, admin, task5Round2Edit(candidate, "TASK5-R3-IDENTICAL-EDIT"))
			},
		},
		{
			name:               "submit",
			expectedCandidates: 1,
			run: func(ctx context.Context, service *regulatory.AdminService, admin identity.Principal, candidate regulatory.CandidateView) (regulatory.CandidateView, error) {
				return service.Submit(ctx, admin, regulatory.SubmitCommand{
					OperationID: "TASK5-R3-IDENTICAL-SUBMIT", IdempotencyKey: "TASK5-R3-IDENTICAL-SUBMIT",
					CandidateID: candidate.CandidateID, ExpectedRevision: candidate.Revision,
					ExpectedContentDigest: candidate.ContentDigest, Reason: "Overlapping identical submission.",
				})
			},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			service, admin, run := task5Round2Service(t, "task5_r3_identical_"+testCase.name)
			candidate := *run.Candidate
			lockTx, err := service.Pool.Begin(context.Background())
			if err != nil {
				t.Fatalf("begin overlap barrier: %v", err)
			}
			defer lockTx.Rollback(context.Background())
			if _, err := lockTx.Exec(context.Background(), `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, candidate.CandidateRootID); err != nil {
				t.Fatalf("hold candidate-root lock: %v", err)
			}

			type result struct {
				candidate regulatory.CandidateView
				err       error
			}
			results := make(chan result, 2)
			start := make(chan struct{})
			var ready sync.WaitGroup
			ready.Add(2)
			for range 2 {
				go func() {
					ready.Done()
					<-start
					view, err := testCase.run(context.Background(), service, admin, candidate)
					results <- result{candidate: view, err: err}
				}()
			}
			ready.Wait()
			close(start)
			select {
			case got := <-results:
				t.Fatalf("identical command bypassed held root lock: %+v err=%v", got.candidate, got.err)
			case <-time.After(150 * time.Millisecond):
			}
			if err := lockTx.Commit(context.Background()); err != nil {
				t.Fatalf("release overlap barrier: %v", err)
			}
			first, second := <-results, <-results
			if first.err != nil || second.err != nil {
				t.Fatalf("identical overlap errors=%v/%v", first.err, second.err)
			}
			if first.candidate.CandidateID != second.candidate.CandidateID ||
				first.candidate.Revision != second.candidate.Revision ||
				first.candidate.ContentDigest != second.candidate.ContentDigest ||
				first.candidate.Status != second.candidate.Status {
				t.Fatalf("identical overlap results differ: first=%+v second=%+v", first.candidate, second.candidate)
			}

			operationID := "TASK5-R3-IDENTICAL-" + strings.ToUpper(testCase.name)
			for table, query := range map[string]string{
				"command": `SELECT COUNT(*) FROM governed_candidate_commands WHERE operation_id=$1`,
				"audit":   `SELECT COUNT(*) FROM audit_events WHERE operation_id=$1`,
			} {
				var count int
				if err := service.Pool.QueryRow(context.Background(), query, operationID).Scan(&count); err != nil || count != 1 {
					t.Fatalf("%s effect count=%d want=1 err=%v", table, count, err)
				}
			}
			var candidates int
			if err := service.Pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM template_draft_versions WHERE candidate_root_id=$1`, candidate.CandidateRootID).Scan(&candidates); err != nil || candidates != testCase.expectedCandidates {
				t.Fatalf("candidate count=%d want=%d err=%v", candidates, testCase.expectedCandidates, err)
			}

			changed := candidate
			if testCase.name == "edit" {
				command := task5Round2Edit(changed, operationID)
				command.ChangeReason = "Conflicting semantic retry."
				_, err = service.CreateRevision(context.Background(), admin, command)
			} else {
				_, err = service.Submit(context.Background(), admin, regulatory.SubmitCommand{
					OperationID: operationID, IdempotencyKey: operationID,
					CandidateID: changed.CandidateID, ExpectedRevision: changed.Revision,
					ExpectedContentDigest: changed.ContentDigest, Reason: "Conflicting semantic retry.",
				})
			}
			if !errors.Is(err, application.ErrConflict) {
				t.Fatalf("changed payload reused identical identity error=%v, want conflict", err)
			}
		})
	}
}

type task5ParityFixture struct {
	Import struct {
		OperationID    string `json:"operationId"`
		IdempotencyKey string `json:"idempotencyKey"`
	} `json:"import"`
	Edit struct {
		OperationID    string `json:"operationId"`
		IdempotencyKey string `json:"idempotencyKey"`
		ChangeReason   string `json:"changeReason"`
	} `json:"edit"`
	Successor struct {
		CandidateID   string `json:"candidateId"`
		Revision      int64  `json:"revision"`
		ContentDigest string `json:"contentDigest"`
		Status        string `json:"status"`
	} `json:"successor"`
	Submit struct {
		OperationID    string `json:"operationId"`
		IdempotencyKey string `json:"idempotencyKey"`
		Reason         string `json:"reason"`
	} `json:"submit"`
	Submitted struct {
		Status string `json:"status"`
	} `json:"submitted"`
	Run struct {
		GenerationRunID string `json:"generationRunId"`
		Status          string `json:"status"`
		InputDigest     string `json:"inputDigest"`
		OutputDigest    string `json:"outputDigest"`
		RequestID       string `json:"requestId"`
		ProviderID      string `json:"providerId"`
	} `json:"run"`
	Validation struct {
		OperationID       string                     `json:"operationId"`
		IdempotencyKey    string                     `json:"idempotencyKey"`
		ChangedSourceHash string                     `json:"changedSourceHash"`
		Issue             regulatory.ValidationIssue `json:"issue"`
	} `json:"validation"`
}

func loadTask5ParityFixture(t *testing.T) task5ParityFixture {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(apiModuleRoot(t), "..", "..", "docs", "regulatory-sources", "fixtures", "task5-admin-parity.v1.json"))
	if err != nil {
		t.Fatalf("read Task 5 parity fixture: %v", err)
	}
	var fixture task5ParityFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode Task 5 parity fixture: %v", err)
	}
	return fixture
}

// Break caught: a stubbed fetch can echo mock expectations without ever
// invoking the repository Go handler or observing persisted PostgreSQL state.
func TestTask5FixRound3RealHandlerMatchesSharedMockParityContract(t *testing.T) {
	ctx := context.Background()
	pool := createTestDatabase(t, "task5_r3_real_handler_parity")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migration 21: %v", err)
	}
	now := time.Date(2026, 7, 29, 14, 0, 0, 0, time.UTC)
	if err := testprofile.Reset(ctx, pool, now); err != nil {
		t.Fatalf("reset canonical profile: %v", err)
	}
	if err := testprofile.BootstrapSyntheticRegulatoryGenerationInputs(ctx, pool); err != nil {
		t.Fatalf("bootstrap synthetic inputs: %v", err)
	}
	fixture := loadTask5ParityFixture(t)
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{Pool: pool, Clock: func() time.Time { return now }})
	handler := httpapi.NewCanonicalTestBoundary("task5-r3-token").Protect(api.Handler())
	request := func(method, path string, body any) *httptest.ResponseRecorder {
		var encoded []byte
		if body != nil {
			encoded, _ = json.Marshal(body)
		}
		httpRequest := httptest.NewRequest(method, path, bytes.NewReader(encoded))
		httpRequest.Header.Set(httpapi.CanonicalTestTokenHeader, "task5-r3-token")
		httpRequest.Header.Set(httpapi.CanonicalTestSubjectHeader, "USR-ADMIN-ADA")
		if body != nil {
			httpRequest.Header.Set("Content-Type", "application/json")
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httpRequest)
		return response
	}

	bundle := regulatory.SyntheticCandidateBundle()
	importBody := map[string]any{"operationId": fixture.Import.OperationID, "idempotencyKey": fixture.Import.IdempotencyKey, "candidateBundle": bundle}
	importedResponse := request(http.MethodPost, "/v1/admin/governed-checklist/generation-runs", importBody)
	if importedResponse.Code != http.StatusCreated {
		t.Fatalf("real-handler import status=%d body=%s", importedResponse.Code, importedResponse.Body.String())
	}
	var imported regulatory.GenerationRunView
	if err := json.Unmarshal(importedResponse.Body.Bytes(), &imported); err != nil || imported.Candidate == nil {
		t.Fatalf("decode real-handler import: %+v err=%v", imported, err)
	}
	if !reflect.DeepEqual(imported.Candidate.Questions, bundle.InspectionChecklist.Questions) {
		t.Fatalf("real-handler lost required question governance transport fields: got=%+v want=%+v", imported.Candidate.Questions, bundle.InspectionChecklist.Questions)
	}
	mappings := cloneMappings(imported.Candidate.Mappings)
	mappings[0].Rationale = "Synthetic test-profile rationale reviewed by an Admin without changing the controlled source claim."
	editBody := map[string]any{
		"operationId": fixture.Edit.OperationID, "idempotencyKey": fixture.Edit.IdempotencyKey,
		"candidateId": imported.Candidate.CandidateID, "expectedRevision": imported.Candidate.Revision,
		"expectedContentDigest": imported.Candidate.ContentDigest, "changeReason": fixture.Edit.ChangeReason,
		"mappings": mappings, "questions": imported.Candidate.Questions, "requiredOwners": imported.Candidate.RequiredOwners,
	}
	invalidBody := cloneJSONMap(editBody)
	invalidMappings := cloneMappings(mappings)
	invalidMappings[0].Citations[0].SourceHash = fixture.Validation.ChangedSourceHash
	invalidBody["operationId"], invalidBody["idempotencyKey"], invalidBody["mappings"] = fixture.Validation.OperationID, fixture.Validation.IdempotencyKey, invalidMappings
	invalidResponse := request(http.MethodPost, "/v1/admin/governed-checklist/candidates/"+imported.Candidate.CandidateID+"/revisions", invalidBody)
	var validation struct {
		Issues []regulatory.ValidationIssue `json:"issues"`
	}
	if invalidResponse.Code != http.StatusUnprocessableEntity {
		t.Fatalf("real-handler validation status=%d body=%s", invalidResponse.Code, invalidResponse.Body.String())
	}
	if err := json.Unmarshal(invalidResponse.Body.Bytes(), &validation); err != nil || len(validation.Issues) != 1 || validation.Issues[0] != fixture.Validation.Issue {
		t.Fatalf("real-handler issue=%+v want=%+v err=%v", validation.Issues, fixture.Validation.Issue, err)
	}

	editedResponse := request(http.MethodPost, "/v1/admin/governed-checklist/candidates/"+imported.Candidate.CandidateID+"/revisions", editBody)
	var edited regulatory.CandidateView
	if editedResponse.Code != http.StatusCreated {
		t.Fatalf("real-handler edit status=%d body=%s", editedResponse.Code, editedResponse.Body.String())
	}
	if err := json.Unmarshal(editedResponse.Body.Bytes(), &edited); err != nil ||
		edited.CandidateID != fixture.Successor.CandidateID || edited.Revision != fixture.Successor.Revision ||
		edited.ContentDigest != fixture.Successor.ContentDigest || edited.Status != fixture.Successor.Status {
		t.Fatalf("real-handler successor=%+v want=%+v err=%v", edited, fixture.Successor, err)
	}
	if replay := request(http.MethodPost, "/v1/admin/governed-checklist/candidates/"+imported.Candidate.CandidateID+"/revisions", editBody); replay.Code != http.StatusCreated || replay.Body.String() != editedResponse.Body.String() {
		t.Fatalf("real-handler edit replay=%d/%s want=%s", replay.Code, replay.Body.String(), editedResponse.Body.String())
	}
	conflictingEdit := cloneJSONMap(editBody)
	conflictingEdit["changeReason"] = "Conflicting semantic retry."
	if conflict := request(http.MethodPost, "/v1/admin/governed-checklist/candidates/"+imported.Candidate.CandidateID+"/revisions", conflictingEdit); conflict.Code != http.StatusConflict {
		t.Fatalf("real-handler conflicting edit status=%d body=%s", conflict.Code, conflict.Body.String())
	}

	submitBody := map[string]any{
		"operationId": fixture.Submit.OperationID, "idempotencyKey": fixture.Submit.IdempotencyKey,
		"candidateId": edited.CandidateID, "expectedRevision": edited.Revision,
		"expectedContentDigest": edited.ContentDigest, "reason": fixture.Submit.Reason,
	}
	submittedResponse := request(http.MethodPost, "/v1/admin/governed-checklist/candidates/"+edited.CandidateID+"/submissions", submitBody)
	var submitted regulatory.CandidateView
	if submittedResponse.Code != http.StatusOK {
		t.Fatalf("real-handler submit status=%d body=%s", submittedResponse.Code, submittedResponse.Body.String())
	}
	if err := json.Unmarshal(submittedResponse.Body.Bytes(), &submitted); err != nil || submitted.Status != fixture.Submitted.Status {
		t.Fatalf("real-handler submitted=%+v err=%v", submitted, err)
	}
	if replay := request(http.MethodPost, "/v1/admin/governed-checklist/candidates/"+edited.CandidateID+"/submissions", submitBody); replay.Code != http.StatusOK || replay.Body.String() != submittedResponse.Body.String() {
		t.Fatalf("real-handler submit replay=%d/%s want=%s", replay.Code, replay.Body.String(), submittedResponse.Body.String())
	}
	conflictingSubmit := cloneJSONMap(submitBody)
	conflictingSubmit["reason"] = "Conflicting semantic retry."
	if conflict := request(http.MethodPost, "/v1/admin/governed-checklist/candidates/"+edited.CandidateID+"/submissions", conflictingSubmit); conflict.Code != http.StatusConflict {
		t.Fatalf("real-handler conflicting submit status=%d body=%s", conflict.Code, conflict.Body.String())
	}

	runResponse := request(http.MethodGet, "/v1/admin/governed-checklist/generation-runs/"+fixture.Run.GenerationRunID, nil)
	candidateResponse := request(http.MethodGet, "/v1/admin/governed-checklist/candidates/"+submitted.CandidateID, nil)
	var rereadRun regulatory.GenerationRunView
	var rereadCandidate regulatory.CandidateView
	if runResponse.Code != http.StatusOK || json.Unmarshal(runResponse.Body.Bytes(), &rereadRun) != nil || rereadRun.Candidate == nil {
		t.Fatalf("real-handler reread run=%d/%s", runResponse.Code, runResponse.Body.String())
	}
	if candidateResponse.Code != http.StatusOK || json.Unmarshal(candidateResponse.Body.Bytes(), &rereadCandidate) != nil {
		t.Fatalf("real-handler reread candidate=%d/%s", candidateResponse.Code, candidateResponse.Body.String())
	}
	if rereadRun.GenerationRunID != fixture.Run.GenerationRunID || rereadRun.Status != fixture.Run.Status ||
		rereadRun.InputDigest != fixture.Run.InputDigest || rereadRun.OutputDigest == nil || *rereadRun.OutputDigest != fixture.Run.OutputDigest ||
		rereadRun.RequestID != fixture.Run.RequestID || rereadRun.ProviderID != fixture.Run.ProviderID {
		t.Fatalf("real-handler immutable run=%+v want=%+v", rereadRun, fixture.Run)
	}
	if rereadRun.Candidate.CandidateID != rereadCandidate.CandidateID ||
		rereadRun.Candidate.Revision != rereadCandidate.Revision ||
		rereadRun.Candidate.ContentDigest != rereadCandidate.ContentDigest ||
		rereadRun.Candidate.Status != rereadCandidate.Status ||
		rereadCandidate.Status != fixture.Submitted.Status {
		t.Fatalf("real-handler run/candidate leaf mismatch: run=%+v candidate=%+v", rereadRun.Candidate, rereadCandidate)
	}
}
