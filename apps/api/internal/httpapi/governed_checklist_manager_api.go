package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/checklistgovernance"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi/generated"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/regulatory"
	"github.com/go-chi/chi/v5"
)

func (api *CanonicalAPI) validateDepartmentManagerBlockedGeneration(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	if _, err := api.governedLifecycle.ListQueue(request.Context(), actor); err != nil {
		api.respond(writer, nil, err)
		return
	}
	var command generated.ValidateDepartmentManagerBlockedGenerationInput
	if !decodeJSON(writer, request, &command) {
		return
	}
	if strings.TrimSpace(command.OperationId) == "" || strings.TrimSpace(command.IdempotencyKey) == "" {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	input := command.GenerationRequest
	sourceSnapshots := make([]regulatory.SourceSnapshot, 0, len(input.SourceSnapshots))
	for _, source := range input.SourceSnapshots {
		sourceSnapshots = append(sourceSnapshots, regulatory.SourceSnapshot{
			SourceSnapshotID: source.SourceSnapshotId, SourceHash: source.SourceHash,
			ClauseIDs: source.ClauseIds, ClauseLocators: source.ClauseLocators,
		})
	}
	unresolved := make([]regulatory.UnresolvedSourceGap, 0, len(input.UnresolvedSourceGaps))
	for _, issue := range input.UnresolvedSourceGaps {
		unresolved = append(unresolved, regulatory.UnresolvedSourceGap{
			GapID: issue.GapId, Reason: issue.Reason,
		})
	}
	exact := regulatory.GenerationRequest{
		SchemaVersion: input.SchemaVersion, RequestID: input.RequestId,
		OrganizationID:              input.OrganizationId,
		ServiceProviderScopeFactIDs: input.ServiceProviderScopeFactIds,
		ServiceProviderTypes:        input.ServiceProviderTypes,
		ProviderCatalogVersion:      input.ProviderCatalogVersion,
		InspectionType:              input.InspectionType,
		Target: regulatory.Target{
			TargetID: input.Target.TargetId, Kind: input.Target.Kind,
		},
		SourceSnapshots: sourceSnapshots,
		SecondaryCrosswalkPartition: regulatory.CrosswalkPartition{
			PartitionID:  input.SecondaryCrosswalkPartition.PartitionId,
			StableRowIDs: input.SecondaryCrosswalkPartition.StableRowIds,
		},
		UnresolvedSourceGaps:    unresolved,
		GenerationPolicyVersion: input.GenerationPolicyVersion,
		ProviderID:              input.ProviderId,
		ProviderVersion:         input.ProviderVersion,
		RequestedOutputs:        input.RequestedOutputs,
		CanonicalInputDigest:    input.CanonicalInputDigest,
	}
	blockers, err := (regulatory.ImportStore{Pool: api.pool}).
		ValidateBlockedRealOPSAOCInput(request.Context(), exact)
	if !errors.Is(err, regulatory.ErrBlockedAuthority) {
		api.respond(writer, nil, err)
		return
	}
	issues := make([]generated.GovernedUnresolvedSourceGapView, 0, len(blockers))
	for _, blocker := range blockers {
		issues = append(issues, generated.GovernedUnresolvedSourceGapView{
			GapId: blocker.GapID, Reason: blocker.Reason,
		})
	}
	api.respond(writer, generated.GovernedBlockedGenerationResult{
		Status: "BLOCKED", RequestId: exact.RequestID, BlockingIssues: issues,
		EffectCounts: generated.GovernedBlockedGenerationEffectCounts{},
	}, nil)
}

func (api *CanonicalAPI) listDepartmentManagerGovernedReviewQueue(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	items, err := api.governedLifecycle.ListQueue(request.Context(), actor)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	output := make([]generated.DepartmentManagerGovernedReviewItem, 0, len(items))
	for _, item := range items {
		output = append(output, governedReviewItem(item))
	}
	api.respond(writer, generated.DepartmentManagerGovernedReviewQueue{Items: output}, nil)
}

func (api *CanonicalAPI) getDepartmentManagerGovernedCandidate(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	item, err := api.governedLifecycle.GetReviewItem(request.Context(), actor, chi.URLParam(request, "candidateId"))
	api.respond(writer, governedReviewItem(item), err)
}

func (api *CanonicalAPI) returnDepartmentManagerGovernedCandidate(writer http.ResponseWriter, request *http.Request) {
	api.handleGovernedReviewCommand(writer, request, api.governedLifecycle.Return)
}

func (api *CanonicalAPI) rejectDepartmentManagerGovernedCandidate(writer http.ResponseWriter, request *http.Request) {
	api.handleGovernedReviewCommand(writer, request, api.governedLifecycle.Reject)
}

func (api *CanonicalAPI) approveDepartmentManagerGovernedCandidate(writer http.ResponseWriter, request *http.Request) {
	api.handleGovernedReviewCommand(writer, request, api.governedLifecycle.Approve)
}

func (api *CanonicalAPI) publishDepartmentManagerGovernedCandidate(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.DepartmentManagerGovernedReviewCommandInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if input.CandidateId != chi.URLParam(request, "candidateId") {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	output, err := api.governedLifecycle.Publish(request.Context(), actor, checklistgovernance.PublicationCommand(governedReviewCommand(input)))
	api.respondCreated(writer, governedPublicationView(output), err)
}

func (api *CanonicalAPI) getDepartmentManagerGovernedPublishedVersion(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.governedLifecycle.GetPublishedVersion(
		request.Context(), actor, chi.URLParam(request, "templateVersionId"),
	)
	api.respond(writer, governedPublishedVersionView(output), err)
}

func (api *CanonicalAPI) handleGovernedReviewCommand(
	writer http.ResponseWriter,
	request *http.Request,
	command func(context.Context, identity.Principal, checklistgovernance.ReviewCommand) (regulatory.CandidateView, error),
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.DepartmentManagerGovernedReviewCommandInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if input.CandidateId != chi.URLParam(request, "candidateId") {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	output, err := command(request.Context(), actor, governedReviewCommand(input))
	api.respond(writer, governedCandidateView(output), err)
}

func governedReviewCommand(input generated.DepartmentManagerGovernedReviewCommandInput) checklistgovernance.ReviewCommand {
	return checklistgovernance.ReviewCommand{
		OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
		CandidateID: input.CandidateId, ExpectedRevision: input.ExpectedRevision,
		ExpectedContentDigest: input.ExpectedContentDigest, Reason: input.Reason,
	}
}

func governedReviewItem(item checklistgovernance.ReviewItem) generated.DepartmentManagerGovernedReviewItem {
	owners := make([]generated.GovernedRequiredOwnerView, 0, len(item.RequiredOwners))
	for _, owner := range item.RequiredOwners {
		owners = append(owners, generated.GovernedRequiredOwnerView{
			DepartmentId: owner.DepartmentID, OrganizationalUnitId: owner.OrganizationalUnitID,
			ApprovalRequired: owner.ApprovalRequired,
		})
	}
	decisions := make([]generated.GovernedReviewDecisionView, 0, len(item.Decisions))
	for _, decision := range item.Decisions {
		decisions = append(decisions, generated.GovernedReviewDecisionView{
			DecisionId: decision.DecisionID, Decision: decision.Decision,
			CandidateRootId: decision.CandidateRootID, CandidateId: decision.CandidateID,
			CandidateRevision:           decision.CandidateRevision,
			CandidateContentDigest:      decision.CandidateContentDigest,
			ActorSubjectId:              decision.ActorSubjectID,
			ActorDepartmentMembershipId: decision.ActorDepartmentMembershipID,
			ActorDepartmentId:           decision.ActorDepartmentID,
			ActorOrganizationalUnitId:   decision.ActorOrganizationalUnitID,
			Reason:                      decision.Reason, DecidedAt: decision.DecidedAt.UTC().Format(time.RFC3339),
			OperationId: decision.OperationID, IdempotencyKey: decision.IdempotencyKey,
			SemanticPayloadDigest: decision.SemanticPayloadDigest,
			AuditEventId:          decision.AuditEventID,
		})
	}
	issues := make([]generated.GovernedValidationIssue, 0, len(item.BlockingIssues))
	for _, issue := range item.BlockingIssues {
		sourceIdentity, sourceHash, clauseID, locator := issue.SourceIdentity, issue.SourceHash, issue.ClauseID, issue.Locator
		issues = append(issues, generated.GovernedValidationIssue{
			FieldPath: issue.FieldPath, Code: issue.Code, Message: issue.Message,
			SourceIdentity: &sourceIdentity, SourceHash: &sourceHash,
			ClauseId: &clauseID, Locator: &locator,
		})
	}
	return generated.DepartmentManagerGovernedReviewItem{
		Candidate: governedCandidateView(item.Candidate), RequiredOwners: owners,
		Decisions: decisions, BlockingIssues: issues,
	}
}

func governedPublicationView(
	value checklistgovernance.PublicationView,
) generated.GovernedPublicationView {
	return generated.GovernedPublicationView{
		TemplateVersionId:     value.TemplateVersionID,
		PublicationDecisionId: value.PublicationDecisionID,
		CandidateRootId:       value.CandidateRootID, CandidateId: value.CandidateID,
		CandidateRevision:           value.CandidateRevision,
		CandidateContentDigest:      value.CandidateContentDigest,
		ActorSubjectId:              value.ActorSubjectID,
		ActorDepartmentMembershipId: value.ActorMembershipID,
		ActorDepartmentId:           value.ActorDepartmentID,
		ActorOrganizationalUnitId:   value.ActorUnitID,
		Reason:                      value.Reason, DecidedAt: value.DecidedAt.UTC().Format(time.RFC3339),
		PublishedAt: value.PublishedAt.UTC().Format(time.RFC3339),
		OperationId: value.OperationID, IdempotencyKey: value.IdempotencyKey,
		SemanticPayloadDigest: value.SemanticPayloadDigest,
		AuditEventId:          value.AuditEventID,
	}
}

func governedPublishedVersionView(
	value checklistgovernance.PublishedVersionView,
) generated.GovernedPublishedVersionView {
	mappings := make([]generated.GovernedMappingView, 0, len(value.Mappings))
	for _, mapping := range value.Mappings {
		raw, _ := json.Marshal(mapping)
		var output generated.GovernedMappingView
		_ = json.Unmarshal(raw, &output)
		mappings = append(mappings, output)
	}
	questions := make([]generated.GovernedQuestionView, 0, len(value.Questions))
	for _, question := range value.Questions {
		raw, _ := json.Marshal(question)
		var output generated.GovernedQuestionView
		_ = json.Unmarshal(raw, &output)
		questions = append(questions, output)
	}
	return generated.GovernedPublishedVersionView{
		Publication: governedPublicationView(value.Publication),
		Mappings:    mappings, Questions: questions,
	}
}
