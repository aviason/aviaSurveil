package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi/generated"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/regulatory"
	"github.com/go-chi/chi/v5"
)

func (api *CanonicalAPI) listAdminGovernedSources(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	items, err := api.governedCandidates.ListSources(request.Context(), actor)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	output := make([]generated.GovernedSourceSnapshotView, 0, len(items))
	for _, item := range items {
		partitions := make([]generated.GovernedSourcePartitionFactView, 0, len(item.Partitions))
		for _, fact := range item.Partitions {
			partitions = append(partitions, generated.GovernedSourcePartitionFactView{
				EvaluationId: fact.EvaluationID, PartitionId: fact.PartitionID, Role: fact.Role,
				CrosswalkRowId: fact.CrosswalkRowID, StableRowIdentity: fact.StableRowIdentity,
			})
		}
		applicabilityFacts := make([]generated.GovernedSourceApplicabilityFactView, 0, len(item.ApplicabilityFacts))
		for _, fact := range item.ApplicabilityFacts {
			sourceGap := json.RawMessage("null")
			if fact.SourceGap != nil {
				sourceGap, _ = json.Marshal(fact.SourceGap)
			}
			applicabilityFacts = append(applicabilityFacts, generated.GovernedSourceApplicabilityFactView{
				CandidateId: fact.CandidateID, MappingId: fact.MappingID, Relationship: fact.Relationship,
				Applicability: fact.Applicability, SourceGap: sourceGap,
			})
		}
		gaps := make([]generated.GovernedUnresolvedSourceGapView, 0, len(item.UnresolvedGaps))
		for _, gap := range item.UnresolvedGaps {
			gaps = append(gaps, generated.GovernedUnresolvedSourceGapView{GapId: gap.GapID, Reason: gap.Reason})
		}
		output = append(output, generated.GovernedSourceSnapshotView{
			SourceId: item.SourceID, SourceIdentity: item.SourceIdentity, VersionIdentity: item.VersionIdentity,
			Title: item.Title, SourceHash: item.SourceHash, Locator: item.Locator, ClauseId: item.ClauseID,
			ClauseLocator: item.ClauseLocator, Partitions: partitions, ApplicabilityFacts: applicabilityFacts,
			UnresolvedGaps: gaps, GenerationRunIds: item.GenerationRunIDs, CandidateIds: item.CandidateIDs,
		})
	}
	api.respond(writer, generated.GovernedSourceSnapshotPage{Items: output}, nil)
}

func (api *CanonicalAPI) activateAdminGovernedSourceCurrentness(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var raw generated.GovernedSourceCurrentnessActivationInput
	if !decodeJSON(writer, request, &raw) {
		return
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	requiredFields := []string{
		"operationId", "idempotencyKey", "currentSourceSnapshotId", "currentSourceHash",
		"previousSourceSnapshotId", "previousSourceHash", "reason",
	}
	// This generated input is deliberately a RawMessage so the handler can
	// distinguish omitted predecessor fields from explicit JSON null. Preserve
	// the OpenAPI additionalProperties:false boundary explicitly before decoding
	// the typed command below.
	if len(fields) != len(requiredFields) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	for _, field := range requiredFields {
		if _, ok := fields[field]; !ok {
			api.respond(writer, nil, application.ErrInvalid)
			return
		}
	}
	var input struct {
		OperationID              string  `json:"operationId"`
		IdempotencyKey           string  `json:"idempotencyKey"`
		CurrentSourceSnapshotID  string  `json:"currentSourceSnapshotId"`
		CurrentSourceHash        string  `json:"currentSourceHash"`
		PreviousSourceSnapshotID *string `json:"previousSourceSnapshotId"`
		PreviousSourceHash       *string `json:"previousSourceHash"`
		Reason                   string  `json:"reason"`
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	previousSourceSnapshotID, previousSourceHash := "", ""
	if input.PreviousSourceSnapshotID != nil {
		previousSourceSnapshotID = *input.PreviousSourceSnapshotID
	}
	if input.PreviousSourceHash != nil {
		previousSourceHash = *input.PreviousSourceHash
	}
	view, err := api.governedCandidates.ActivateSourceCurrentness(request.Context(), actor, regulatory.SourceCurrentnessActivationCommand{
		OperationID:              input.OperationID,
		IdempotencyKey:           input.IdempotencyKey,
		CurrentSourceSnapshotID:  input.CurrentSourceSnapshotID,
		CurrentSourceHash:        input.CurrentSourceHash,
		PreviousSourceSnapshotID: previousSourceSnapshotID,
		PreviousSourceHash:       previousSourceHash,
		Reason:                   input.Reason,
	})
	api.respondCreated(writer, generated.GovernedSourceCurrentnessActivationView{
		EventId:                  view.EventID,
		ImpactReviewDraftId:      view.ImpactReviewDraftID,
		SourceIdentity:           view.SourceIdentity,
		PreviousSourceSnapshotId: view.PreviousSourceSnapshotID,
		PreviousSourceHash:       view.PreviousSourceHash,
		CurrentSourceSnapshotId:  view.CurrentSourceSnapshotID,
		CurrentSourceHash:        view.CurrentSourceHash,
		Status:                   view.Status,
		ActivatedAt:              view.ActivatedAt.Format(time.RFC3339Nano),
	}, err)
}

func (api *CanonicalAPI) importAdminGovernedGenerationRun(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.ImportAdminGovernedGenerationRunInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	raw, err := json.Marshal(input.CandidateBundle)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	var bundle regulatory.CandidateBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	view, err := api.governedCandidates.Import(request.Context(), actor, input.OperationId, input.IdempotencyKey, bundle)
	api.respondCreated(writer, governedGenerationRunView(view), err)
}
func (api *CanonicalAPI) getAdminGovernedGenerationRun(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.governedCandidates.GetRun(request.Context(), actor, chi.URLParam(request, "generationRunId"))
	api.respond(writer, governedGenerationRunView(output), err)
}
func (api *CanonicalAPI) getAdminGovernedCandidate(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.governedCandidates.GetCandidate(request.Context(), actor, chi.URLParam(request, "candidateId"))
	api.respond(writer, governedCandidateView(output), err)
}

func (api *CanonicalAPI) createAdminGovernedCandidateRevision(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.CreateAdminGovernedCandidateRevisionInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if input.CandidateId != chi.URLParam(request, "candidateId") {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	mappings, questions, owners, err := governedEditPayload(input)
	if err != nil {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	output, err := api.governedCandidates.CreateRevision(request.Context(), actor, regulatory.EditCommand{OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey, CandidateID: input.CandidateId, ExpectedRevision: input.ExpectedRevision, ExpectedContentDigest: input.ExpectedContentDigest, ChangeReason: input.ChangeReason, Mappings: mappings, Questions: questions, RequiredOwners: owners})
	api.respondCreated(writer, governedCandidateView(output), err)
}
func (api *CanonicalAPI) submitAdminGovernedCandidateReview(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.SubmitAdminGovernedCandidateReviewInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if input.CandidateId != chi.URLParam(request, "candidateId") {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	output, err := api.governedCandidates.Submit(request.Context(), actor, regulatory.SubmitCommand{OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey, CandidateID: input.CandidateId, ExpectedRevision: input.ExpectedRevision, ExpectedContentDigest: input.ExpectedContentDigest, Reason: input.Reason})
	api.respond(writer, governedCandidateView(output), err)
}

func governedEditPayload(input generated.CreateAdminGovernedCandidateRevisionInput) ([]regulatory.ComplianceMapping, []regulatory.ChecklistQuestion, []regulatory.RequiredOwner, error) {
	raw, err := json.Marshal(input.Mappings)
	if err != nil {
		return nil, nil, nil, err
	}
	var mappings []regulatory.ComplianceMapping
	if err = json.Unmarshal(raw, &mappings); err != nil {
		return nil, nil, nil, err
	}
	raw, err = json.Marshal(input.Questions)
	if err != nil {
		return nil, nil, nil, err
	}
	var questions []regulatory.ChecklistQuestion
	if err = json.Unmarshal(raw, &questions); err != nil {
		return nil, nil, nil, err
	}
	raw, err = json.Marshal(input.RequiredOwners)
	if err != nil {
		return nil, nil, nil, err
	}
	var owners []regulatory.RequiredOwner
	err = json.Unmarshal(raw, &owners)
	return mappings, questions, owners, err
}
func governedCandidateView(value regulatory.CandidateView) generated.GovernedCandidateView {
	output := generated.GovernedCandidateView{}
	raw, _ := json.Marshal(value)
	_ = json.Unmarshal(raw, &output)
	if value.GenerationRunID != "" {
		output.Lineage = generated.GovernedCandidateLineage{
			GovernedGenerationRunLineage: &generated.GovernedGenerationRunLineage{
				LineageType:           "GENERATION_RUN",
				EntryPath:             "GENERATION_RUN",
				LineageKind:           "GENERATION_RUN",
				CandidateRootId:       value.CandidateRootID,
				SupersedesCandidateId: value.SupersedesCandidateID,
				GenerationRunId:       value.GenerationRunID,
			},
		}
	} else {
		output.Lineage = generated.GovernedCandidateLineage{
			GovernedExistingCandidateLineage: &generated.GovernedExistingCandidateLineage{
				LineageType:           "EXISTING_CANDIDATE",
				EntryPath:             "EXISTING_CANDIDATE",
				LineageKind:           "EXISTING_CANDIDATE",
				CandidateRootId:       value.CandidateRootID,
				SupersedesCandidateId: value.SupersedesCandidateID,
				ExistingCandidateId:   value.CandidateID,
			},
		}
	}
	return output
}
func governedGenerationRunView(value regulatory.GenerationRunView) generated.GovernedGenerationRunView {
	output := generated.GovernedGenerationRunView{
		GenerationRunId: value.GenerationRunID, Status: value.Status, InputDigest: value.InputDigest,
		OutputDigest: value.OutputDigest, InputSchemaVersion: value.InputSchemaVersion,
		GenerationPolicyVersion: value.GenerationPolicyVersion, ProviderCatalogVersion: value.ProviderCatalogVersion,
		ProviderId: value.ProviderID, ProviderAdapterVersion: value.ProviderAdapterVersion, InspectionType: value.InspectionType,
		TargetId: value.TargetID, RequestId: value.RequestID, Failure: json.RawMessage("null"),
	}
	if value.Failure != nil {
		output.Failure, _ = json.Marshal(generated.GovernedGenerationFailureView{
			Code: value.Failure.Code, Reason: value.Failure.Reason, RequestId: value.Failure.RequestID,
			OperationId: value.Failure.OperationID, IdempotencyKey: value.Failure.IdempotencyKey,
		})
	}
	if value.Candidate != nil {
		raw, _ := json.Marshal(governedCandidateView(*value.Candidate))
		output.Candidate = raw
	} else {
		output.Candidate = json.RawMessage("null")
	}
	return output
}
