package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/httpapi/generated"
	"github.com/aviason/aviaSurveil/internal/planning"
	"github.com/go-chi/chi/v5"
)

func proposalValuesFromGenerated(value generated.PlanningProposalDraftValues) (planning.ProposalDraftValues, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return planning.ProposalDraftValues{}, err
	}
	var output planning.ProposalDraftValues
	if err := json.Unmarshal(raw, &output); err != nil {
		return planning.ProposalDraftValues{}, err
	}
	return output, nil
}

func (api *CanonicalAPI) listPlanningPurposePresets(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	presets, err := api.planning.ListPurposePresets(request.Context(), actor)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	output := make([]generated.PlanningPurposePreset, 0, len(presets))
	for _, preset := range presets {
		output = append(output, generated.PlanningPurposePreset{Id: preset.ID, Version: preset.Version, Label: preset.Label, Purpose: preset.Purpose, Active: preset.Active, DisplayOrder: preset.DisplayOrder})
	}
	writeJSON(writer, http.StatusOK, output)
}

func (api *CanonicalAPI) listPlanningLocations(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	locations, err := api.planning.ListProposalLocations(request.Context(), actor, request.URL.Query().Get("organizationId"), request.URL.Query().Get("regulatedTargetId"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	output := make([]generated.PlanningLocationOption, 0, len(locations))
	for _, location := range locations {
		output = append(output, generated.PlanningLocationOption{Id: location.ID, Label: location.Label, Aliases: location.Aliases, Source: location.Source})
	}
	writeJSON(writer, http.StatusOK, output)
}

func (api *CanonicalAPI) resolvePlanningLocation(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input struct {
		OperationID       string `json:"operationId"`
		IdempotencyKey    string `json:"idempotencyKey"`
		OrganizationID    string `json:"organizationId"`
		RegulatedTargetID string `json:"regulatedTargetId"`
		ProposedLabel     string `json:"proposedLabel"`
	}
	if !decodeJSON(writer, request, &input) || strings.TrimSpace(input.OperationID) == "" || strings.TrimSpace(input.IdempotencyKey) == "" {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	resolution, err := api.planning.ResolveProposalLocation(request.Context(), actor, input.OrganizationID, input.RegulatedTargetID, input.ProposedLabel)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	var location json.RawMessage
	if resolution.Location != nil {
		location, _ = json.Marshal(generated.PlanningLocationOption{Id: resolution.Location.ID, Label: resolution.Location.Label, Aliases: resolution.Location.Aliases, Source: resolution.Location.Source})
	} else {
		location = json.RawMessage("null")
	}
	writeJSON(writer, http.StatusOK, map[string]any{"outcome": resolution.Outcome, "location": json.RawMessage(location), "acceptedResolutionToken": resolution.AcceptedResolutionToken, "message": resolution.Message})
}

func (api *CanonicalAPI) getPlanningWorkloadEstimate(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input planning.ProposalDraftValues
	var metadata struct {
		OperationID       string `json:"operationId"`
		IdempotencyKey    string `json:"idempotencyKey"`
		OrganizationID    string `json:"organizationId"`
		ProviderScopeID   string `json:"providerScopeId"`
		RegulatedTargetID string `json:"regulatedTargetId"`
		InspectionType    string `json:"inspectionType"`
	}
	if !decodeJSON(writer, request, &metadata) || strings.TrimSpace(metadata.OperationID) == "" || strings.TrimSpace(metadata.IdempotencyKey) == "" {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	input.OrganizationID, input.ProviderScopeID, input.RegulatedTargetID, input.InspectionType = metadata.OrganizationID, metadata.ProviderScopeID, metadata.RegulatedTargetID, metadata.InspectionType
	estimate, err := api.planning.GetProposalWorkloadEstimate(request.Context(), actor, input)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writeJSON(writer, http.StatusOK, estimate)
}

func (api *CanonicalAPI) createPlanningProposalDraft(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.CreatePlanningProposalDraftInput
	if !decodeJSON(writer, request, &input) || strings.TrimSpace(input.OperationId) == "" || strings.TrimSpace(input.IdempotencyKey) == "" || input.ExpectedRevision != nil {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	values, err := proposalValuesFromGenerated(input.Values)
	if err != nil {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	draftID := ""
	if input.DraftId != nil {
		draftID = *input.DraftId
	}
	record, err := api.planning.CreateProposalDraft(request.Context(), actor, planning.CreateProposalDraftCommand{OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey, DraftID: draftID, Values: values})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writeJSON(writer, http.StatusCreated, record)
}

func (api *CanonicalAPI) getPlanningProposalDraft(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	record, err := api.planning.GetProposalDraft(request.Context(), actor, chi.URLParam(request, "draftId"))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writeJSON(writer, http.StatusOK, record)
}

func (api *CanonicalAPI) savePlanningProposalDraft(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.SavePlanningProposalDraftInput
	if !decodeJSON(writer, request, &input) || strings.TrimSpace(input.OperationId) == "" || strings.TrimSpace(input.IdempotencyKey) == "" || input.ExpectedRevision <= 0 {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	values, err := proposalValuesFromGenerated(input.Values)
	if err != nil {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	record, err := api.planning.SaveProposalDraft(request.Context(), actor, planning.SaveProposalDraftCommand{OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey, DraftID: chi.URLParam(request, "draftId"), ExpectedRevision: input.ExpectedRevision, Values: values})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writeJSON(writer, http.StatusOK, record)
}

func (api *CanonicalAPI) submitPlanningProposal(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.SubmitPlanningIntakeInput
	if !decodeJSON(writer, request, &input) || strings.TrimSpace(input.OperationId) == "" || strings.TrimSpace(input.IdempotencyKey) == "" || input.ExpectedRevision == nil || *input.ExpectedRevision <= 0 {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	planningItemID := ""
	if input.PlanningItemId != nil {
		planningItemID = *input.PlanningItemId
	}
	result, err := api.planning.SubmitProposal(request.Context(), actor, planning.SubmitProposalCommand{OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey, DraftID: chi.URLParam(request, "draftId"), PlanningItemID: planningItemID, ExpectedRevision: *input.ExpectedRevision})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func planningAuditPackageSetupView(value planning.AuditPackageSetup) generated.PlanningAuditPackageSetupView {
	return generated.PlanningAuditPackageSetupView{
		PlanningItemId:               value.PlanningItemID,
		PlanningSnapshotId:           value.PlanningSnapshotID,
		PlanningSnapshotDigest:       value.PlanningSnapshotDigest,
		ScopeDraftId:                 value.ScopeDraftID,
		Status:                       string(value.Status),
		Revision:                     value.Revision,
		CatalogVersion:               value.CatalogVersion,
		CatalogRootDigest:            value.CatalogRootDigest,
		SelectedCount:                value.SelectedCount,
		SelectionDigest:              value.SelectionDigest,
		ApprovedChecklistItemCeiling: value.ApprovedChecklistItemCeiling,
		NextAction:                   value.NextAction,
	}
}

func (api *CanonicalAPI) ensurePlanningAuditPackageSetup(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.EnsurePlanningAuditPackageSetupInput
	if !decodeJSON(writer, request, &input) || strings.TrimSpace(input.OperationId) == "" || strings.TrimSpace(input.IdempotencyKey) == "" || input.ExpectedPlanningRevision <= 0 {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	setup, err := api.planning.EnsureAuditPackageSetup(request.Context(), actor, planning.EnsureAuditPackageSetupCommand{
		OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
		PlanningItemID: chi.URLParam(request, "planningItemId"), ExpectedPlanningRevision: input.ExpectedPlanningRevision,
	})
	api.respond(writer, planningAuditPackageSetupView(setup), err)
}

func (api *CanonicalAPI) getPlanningAuditPackageSetup(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	setup, err := api.planning.GetAuditPackageSetup(request.Context(), actor, chi.URLParam(request, "planningItemId"))
	api.respond(writer, planningAuditPackageSetupView(setup), err)
}

func (api *CanonicalAPI) finalizePlanningAuditPackage(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.FinalizePlanningAuditPackageInput
	if !decodeJSON(writer, request, &input) || strings.TrimSpace(input.OperationId) == "" || strings.TrimSpace(input.IdempotencyKey) == "" || input.ExpectedPlanningRevision <= 0 || input.ExpectedSetupRevision <= 0 {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	setup, err := api.planning.FinalizeAuditPackage(request.Context(), actor, planning.FinalizeAuditPackageCommand{
		OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
		PlanningItemID: chi.URLParam(request, "planningItemId"), ExpectedPlanningRevision: input.ExpectedPlanningRevision,
		ExpectedSetupRevision: input.ExpectedSetupRevision, ExpectedSelectionDigest: input.ExpectedSelectionDigest,
	})
	api.respond(writer, planningAuditPackageSetupView(setup), err)
}
