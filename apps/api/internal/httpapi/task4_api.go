package httpapi

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/assignments"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/datafeed"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi/generated"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/inspections"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/planning"
	"github.com/go-chi/chi/v5"
)

func decodedCanonicalPathParam(request *http.Request, name string) string {
	raw := chi.URLParam(request, name)
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return raw
	}
	return decoded
}

func (api *CanonicalAPI) getPlanningIntakeDraft(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	draft, err := api.planning.GetIntakeDraft(
		request.Context(), actor, decodedCanonicalPathParam(request, "draftId"),
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(draft.Revision))
	api.respond(writer, planningIntakeDraftView(draft), nil)
}

func (api *CanonicalAPI) savePlanningIntakeDraft(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.SavePlanningIntakeDraftInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	draftID := decodedCanonicalPathParam(request, "draftId")
	if draftID != input.DraftId || !validRevisionCommandHeaders(
		request, input.IdempotencyKey, input.ExpectedRevision,
	) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	if err := api.requireCatalogRuntimeProfile(request.Context(), optionalString(input.Values.CatalogVersion)); err != nil {
		api.respond(writer, nil, err)
		return
	}
	draft, err := api.planning.SaveIntakeDraft(
		request.Context(), actor, planning.SaveIntakeDraftCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			DraftID: input.DraftId, ExpectedRevision: *input.ExpectedRevision,
			Values: planning.IntakeDraftValues{
				OrganizationID:     input.Values.OrganizationId,
				OrganizationName:   input.Values.OrganizationName,
				ApplicationType:    input.Values.ApplicationType,
				Domain:             input.Values.Domain,
				InspectionCategory: planning.InspectionCategory(input.Values.InspectionCategory),
				NoticePolicy:       planning.NoticePolicy(input.Values.NoticePolicy),
				Purpose:            input.Values.Purpose, TriggerType: input.Values.TriggerType,
				RiskCategory: input.Values.RiskCategory, PlannedDate: input.Values.PlannedDate,
				Mode: input.Values.Mode, Location: input.Values.Location,
				TemplateVersionID: optionalString(input.Values.TemplateVersionId), Scope: optionalString(input.Values.Scope),
				CatalogVersion: optionalString(input.Values.CatalogVersion), ScopeDraftID: optionalString(input.Values.ScopeDraftId),
				SelectionDigest:              optionalString(input.Values.SelectionDigest),
				SelectedQuestionVersionIDs:   append([]string(nil), input.Values.SelectedQuestionVersionIds...),
				EstimatedResourceRequirement: optionalFloat(input.Values.EstimatedResourceRequirement),
				FormDistribution:             input.Values.FormDistribution,
				DomainDistribution:           input.Values.DomainDistribution,
				ProviderScopeID:              optionalString(input.Values.ProviderScopeId), RegulatedTargetID: optionalString(input.Values.RegulatedTargetId),
				RequestedBudget: input.Values.RequestedBudget, Currency: input.Values.Currency,
			},
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(draft.Revision))
	api.respond(writer, planningIntakeDraftView(draft), nil)
}

func (api *CanonicalAPI) submitPlanningIntake(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.SubmitPlanningIntakeInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	draftID := decodedCanonicalPathParam(request, "draftId")
	if draftID != input.DraftId || !validRevisionCommandHeaders(
		request, input.IdempotencyKey, input.ExpectedRevision,
	) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	if err := api.requireDraftRuntimeProfile(request.Context(), draftID); err != nil {
		api.respond(writer, nil, err)
		return
	}
	result, err := api.planning.SubmitIntake(
		request.Context(), actor, planning.SubmitIntakeCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			DraftID: input.DraftId, PlanningItemID: optionalString(input.PlanningItemId),
			ExpectedRevision: *input.ExpectedRevision,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(result.Draft.Revision))
	api.respond(writer, generated.SubmitPlanningIntakeOutput{
		Draft:        planningIntakeDraftView(result.Draft),
		PlanningItem: planningView(result.PlanningItem),
	}, nil)
}

func (api *CanonicalAPI) getInspectionPackageDraft(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if api.preprodExerciseProfile {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	draft, err := api.packageDrafts.Get(
		request.Context(), actor, decodedCanonicalPathParam(request, "packageDraftId"),
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(draft.Revision))
	api.respond(writer, inspectionPackageDraftView(draft), nil)
}

func (api *CanonicalAPI) saveInspectionPackageDraft(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if api.preprodExerciseProfile {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.SaveInspectionPackageDraftInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	draftID := decodedCanonicalPathParam(request, "packageDraftId")
	if draftID != input.PackageDraftId || !validRevisionCommandHeaders(
		request, input.IdempotencyKey, input.ExpectedRevision,
	) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	draft, err := api.packageDrafts.Save(
		request.Context(), actor, inspections.SavePackageDraftCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			PackageDraftID: input.PackageDraftId, ExpectedRevision: *input.ExpectedRevision,
			RiskFocus: input.RiskFocus,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(draft.Revision))
	api.respond(writer, inspectionPackageDraftView(draft), nil)
}

func (api *CanonicalAPI) listTeamMembers(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var role *identity.Role
	if value := strings.TrimSpace(request.URL.Query().Get("role")); value != "" {
		candidate := identity.Role(value)
		role = &candidate
	}
	members, err := api.assignments.ListTeamMembers(
		request.Context(), actor, role,
		boundedPageLimit(optionalIntQuery(request, "limit")),
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	items := make([]generated.TeamMemberView, 0, len(members))
	for _, member := range members {
		items = append(items, teamMemberView(member))
	}
	api.respond(writer, generated.ListTeamMembersOutput{Items: items}, nil)
}

func (api *CanonicalAPI) getTeamMember(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	member, err := api.assignments.GetTeamMember(
		request.Context(), actor, decodedCanonicalPathParam(request, "subjectId"),
	)
	api.respond(writer, teamMemberView(member), err)
}

func (api *CanonicalAPI) listAuditTeams(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	items, err := api.assignments.ListAuditTeams(
		request.Context(), actor,
		boundedPageLimit(optionalIntQuery(request, "limit")),
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	views := make([]generated.InspectionTeamAuditView, 0, len(items))
	for _, item := range items {
		views = append(views, inspectionTeamAuditView(item))
	}
	api.respond(writer, generated.ListInspectionTeamAuditsOutput{Items: views}, nil)
}

func (api *CanonicalAPI) getAuditTeam(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	item, err := api.assignments.GetAuditTeam(
		request.Context(), actor, decodedCanonicalPathParam(request, "auditId"),
	)
	api.respond(writer, inspectionTeamAuditView(item), err)
}

func (api *CanonicalAPI) prepareAudit(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.PrepareAuditInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	planningItemID := decodedCanonicalPathParam(request, "planningItemId")
	if !validRevisionCommandHeaders(request, input.IdempotencyKey, &input.ExpectedPlanningRevision) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	item, err := api.assignments.Prepare(request.Context(), actor, assignments.PrepareCommand{
		OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
		PlanningItemID:           planningItemID,
		ExpectedPlanningRevision: input.ExpectedPlanningRevision,
	})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(item.Revision))
	api.respond(writer, generated.PreparationView{
		AssignmentId: item.AssignmentID, PlanningItemId: item.PlanningItemID,
		OrganizationId: item.OrganizationID, Status: string(item.Status), Revision: item.Revision,
	}, nil)
}

func (api *CanonicalAPI) getCanonicalAuditPreparation(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	item, err := api.assignments.GetPreparationForLead(request.Context(), actor,
		strings.TrimSpace(request.URL.Query().Get("assignmentId")),
		strings.TrimSpace(request.URL.Query().Get("planningItemId")))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(item.Revision))
	api.respond(writer, canonicalAssignmentView(item), nil)
}

func (api *CanonicalAPI) assignAuditLead(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.AssignLeadInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	assignmentID := decodedCanonicalPathParam(request, "assignmentId")
	if !validRevisionCommandHeaders(request, input.IdempotencyKey, &input.ExpectedInspectionRevision) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	item, err := api.assignments.AssignLead(request.Context(), actor, assignments.AssignLeadCommand{
		OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
		AssignmentID:               assignmentID,
		ExpectedInspectionRevision: input.ExpectedInspectionRevision, LeadSubjectID: input.LeadSubjectId,
	})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(item.Revision))
	api.respond(writer, canonicalAssignmentView(item), nil)
}

func (api *CanonicalAPI) assignAuditTeam(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.AssignTeamInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	assignmentID := decodedCanonicalPathParam(request, "assignmentId")
	if !validRevisionCommandHeaders(request, input.IdempotencyKey, &input.ExpectedRevision) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	item, err := api.assignments.AssignTeam(request.Context(), actor, assignments.AssignTeamCommand{
		OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
		AssignmentID: assignmentID, ExpectedRevision: input.ExpectedRevision,
		PreviewID: input.PreviewId, PreviewDigest: input.PreviewDigest,
		MemberSubjectIDs: append([]string(nil), input.MemberSubjectIds...),
	})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(item.Revision))
	api.respond(writer, canonicalAssignmentView(item), nil)
}

func (api *CanonicalAPI) previewAuditTeam(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.PreviewAssignTeamInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	assignmentID := decodedCanonicalPathParam(request, "assignmentId")
	if !validRevisionCommandHeaders(request, input.IdempotencyKey, &input.ExpectedRevision) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	item, err := api.assignments.PreviewTeam(request.Context(), actor, assignments.PreviewTeamCommand{
		OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
		AssignmentID: assignmentID, ExpectedRevision: input.ExpectedRevision,
		MemberSubjectIDs: append([]string(nil), input.MemberSubjectIds...),
	})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(item.AssignmentRevision))
	api.respond(writer, preparationEditPreviewView(item), nil)
}

func (api *CanonicalAPI) assignAuditQuestionCoverage(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.AssignQuestionsInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	assignmentID := decodedCanonicalPathParam(request, "assignmentId")
	if !validRevisionCommandHeaders(request, input.IdempotencyKey, &input.ExpectedRevision) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	questionAssignments := make([]assignments.QuestionAssignment, 0, len(input.QuestionAssignments))
	for _, assignment := range input.QuestionAssignments {
		questionAssignments = append(questionAssignments, assignments.QuestionAssignment{
			QuestionID: assignment.QuestionId, SubjectID: assignment.SubjectId,
		})
	}
	item, err := api.assignments.AssignQuestions(request.Context(), actor, assignments.AssignQuestionsCommand{
		OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
		AssignmentID: assignmentID, ExpectedRevision: input.ExpectedRevision,
		PreviewID: input.PreviewId, PreviewDigest: input.PreviewDigest,
		OperationKind:       assignments.QuestionCoverageOperationKind(input.OperationKind),
		QuestionAssignments: questionAssignments,
	})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(item.Revision))
	api.respond(writer, canonicalAssignmentView(item), nil)
}

func (api *CanonicalAPI) previewAuditQuestionCoverage(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.PreviewAssignQuestionsInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	assignmentID := decodedCanonicalPathParam(request, "assignmentId")
	if !validRevisionCommandHeaders(request, input.IdempotencyKey, &input.ExpectedRevision) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	questionAssignments := make([]assignments.QuestionAssignment, 0, len(input.QuestionAssignments))
	for _, assignment := range input.QuestionAssignments {
		questionAssignments = append(questionAssignments, assignments.QuestionAssignment{QuestionID: assignment.QuestionId, SubjectID: assignment.SubjectId})
	}
	item, err := api.assignments.PreviewQuestions(request.Context(), actor, assignments.PreviewQuestionsCommand{
		OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
		AssignmentID: assignmentID, ExpectedRevision: input.ExpectedRevision,
		OperationKind:       assignments.QuestionCoverageOperationKind(input.OperationKind),
		QuestionAssignments: questionAssignments,
	})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(item.AssignmentRevision))
	api.respond(writer, preparationEditPreviewView(item), nil)
}

func (api *CanonicalAPI) confirmAuditPreparation(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.ConfirmAuditPreparationInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	assignmentID := decodedCanonicalPathParam(request, "assignmentId")
	if !validRevisionCommandHeaders(request, input.IdempotencyKey, &input.ExpectedAssignmentRevision) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	item, err := api.assignments.ConfirmPreparation(
		request.Context(), actor, assignments.ConfirmPreparationCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			AssignmentID: assignmentID, ExpectedAssignmentRevision: input.ExpectedAssignmentRevision,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(item.Revision))
	api.respond(writer, preparationConfirmationView(item), nil)
}

func (api *CanonicalAPI) materializeCanonicalAudit(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.MaterializeCanonicalAuditInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	assignmentID := decodedCanonicalPathParam(request, "assignmentId")
	if !validRevisionCommandHeaders(request, input.IdempotencyKey, &input.ExpectedAssignmentRevision) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	correlationID, err := datafeed.NewEventID()
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	item, err := api.application.MaterializeInspection(
		request.Context(), actor, application.MaterializeInspectionCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			CorrelationID: correlationID, AssignmentID: assignmentID,
			ExpectedAssignmentRevision: input.ExpectedAssignmentRevision,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(item.AssignmentRevision))
	api.respond(writer, canonicalMaterializedAuditView(item), nil)
}

func (api *CanonicalAPI) listAuditeeCoordination(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	items, err := api.assignments.ListAuditeeCoordination(request.Context(), actor)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	views := make([]generated.AuditeeCoordinationView, 0, len(items))
	for _, item := range items {
		views = append(views, auditeeCoordinationView(item))
	}
	api.respond(writer, generated.AuditeeCoordinationPage{Items: views}, nil)
}

func (api *CanonicalAPI) respondAuditeeCoordination(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.RespondToAuditeeCoordinationInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	auditID := decodedCanonicalPathParam(request, "auditId")
	if auditID != input.AuditId || !validRevisionCommandHeaders(
		request, input.IdempotencyKey, input.ExpectedRevision,
	) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	item, err := api.assignments.RespondAuditeeCoordination(
		request.Context(), actor, assignments.RespondCoordinationCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			InspectionID: input.AuditId, OrganizationID: input.OrganizationId,
			ExpectedRevision: *input.ExpectedRevision,
			Decision:         assignments.CoordinationDecision(input.Decision),
			AlternativeDate:  input.AlternativeDate,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(item.Revision))
	api.respond(writer, auditeeCoordinationView(item), nil)
}

func (api *CanonicalAPI) reviewAuditeeCoordination(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.ReviewAuditeeCoordinationInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	auditID := decodedCanonicalPathParam(request, "auditId")
	if auditID != input.AuditId || !validRevisionCommandHeaders(request, input.IdempotencyKey, &input.ExpectedRevision) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	item, err := api.assignments.ReviewAuditeeCoordination(
		request.Context(), actor, assignments.ReviewCoordinationCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			InspectionID: input.AuditId, OrganizationID: input.OrganizationId,
			ExpectedRevision: input.ExpectedRevision,
			Decision:         assignments.ReviewCoordinationDecision(input.Decision), Reason: input.Reason,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(item.Revision))
	api.respond(writer, auditeeCoordinationView(item), nil)
}

func validRevisionCommandHeaders(
	request *http.Request,
	idempotencyKey string,
	expectedRevision *int64,
) bool {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	return expectedRevision != nil &&
		idempotencyKey != "" &&
		request.Header.Get("Idempotency-Key") == idempotencyKey &&
		request.Header.Get("If-Match") == strongRevisionETag(*expectedRevision)
}

func planningIntakeDraftView(draft planning.IntakeDraft) generated.PlanningIntakeDraftView {
	return generated.PlanningIntakeDraftView{
		Id: draft.ID, OrganizationId: draft.OrganizationID,
		OrganizationName: draft.OrganizationName, ApplicationType: draft.ApplicationType,
		Domain: draft.Domain, InspectionCategory: string(draft.InspectionCategory),
		NoticePolicy: string(draft.NoticePolicy), Purpose: draft.Purpose,
		TriggerType: draft.TriggerType, RiskCategory: draft.RiskCategory,
		PlannedDate: draft.PlannedDate, Mode: draft.Mode, Location: draft.Location,
		TemplateVersionId: optionalStringPointer(draft.TemplateVersionID), Scope: optionalStringPointer(draft.Scope),
		CatalogVersion: optionalStringPointer(draft.CatalogVersion), ScopeDraftId: optionalStringPointer(draft.ScopeDraftID),
		SelectionDigest:              optionalStringPointer(draft.SelectionDigest),
		SelectedQuestionVersionIds:   append([]string(nil), draft.SelectedQuestionVersionIDs...),
		EstimatedResourceRequirement: optionalFloatPointer(draft.EstimatedResourceRequirement),
		FormDistribution:             draft.FormDistribution,
		DomainDistribution:           draft.DomainDistribution,
		ProviderScopeId:              optionalStringPointer(draft.ProviderScopeID), RegulatedTargetId: optionalStringPointer(draft.RegulatedTargetID),
		RequestedBudget: draft.RequestedBudget, Currency: draft.Currency,
		Revision: draft.Revision, SubmittedPlanningItemId: draft.SubmittedPlanningItemID,
		UpdatedAt: draft.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	}
}

func inspectionPackageDraftView(
	draft inspections.PackageDraft,
) generated.InspectionPackageDraftView {
	questions := make([]generated.InspectionPackageDraftQuestionView, 0, len(draft.Questions))
	for _, question := range draft.Questions {
		questions = append(questions, generated.InspectionPackageDraftQuestionView{
			Id: question.ID, Prompt: question.Prompt, WhyIncluded: question.WhyIncluded,
			ExpectedEvidence:    append([]string(nil), question.ExpectedEvidence...),
			ConfiguredReference: question.ConfiguredReference,
		})
	}
	return generated.InspectionPackageDraftView{
		Id: draft.ID, SourceAuditId: draft.SourceAuditID,
		OrganizationId: draft.OrganizationID, OrganizationName: draft.OrganizationName,
		ApplicationType: draft.ApplicationType, Domain: draft.Domain, Status: draft.Status,
		PackageVersion: draft.PackageVersion, Revision: draft.Revision,
		RiskFocus: append([]string(nil), draft.RiskFocus...), Questions: questions,
		UpdatedAt: draft.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	}
}

func teamMemberView(member assignments.TeamMember) generated.TeamMemberView {
	return generated.TeamMemberView{
		SubjectId: member.SubjectID, DisplayName: member.DisplayName,
		Role: generated.Role(member.Role), OrganizationId: member.OrganizationID,
		Revision: member.Revision,
	}
}

func inspectionTeamAuditView(item assignments.TeamAudit) generated.InspectionTeamAuditView {
	members := make([]generated.TeamMemberView, 0, len(item.Members))
	for _, member := range item.Members {
		members = append(members, teamMemberView(member))
	}
	questionAssignments := make([]map[string]any, 0, len(item.Assignments))
	for _, assignment := range item.Assignments {
		questionAssignments = append(questionAssignments, map[string]any{
			"questionId": assignment.QuestionID,
			"assignedMemberSubjectIds": append(
				[]string(nil), assignment.AssignedMemberSubjectIDs...,
			),
		})
	}
	history := make([]map[string]any, 0, len(item.History))
	for _, event := range item.History {
		history = append(history, map[string]any{
			"eventId":        event.EventID,
			"occurredAt":     event.OccurredAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
			"actorSubjectId": event.ActorSubjectID,
			"action":         event.Action,
			"detail":         event.Detail,
		})
	}
	return generated.InspectionTeamAuditView{
		AuditId: item.AuditID, OrganizationId: item.OrganizationID,
		OrganizationName: item.OrganizationName, Title: item.Title, Status: item.Status,
		ScheduledStartDate: item.ScheduledStartDate,
		ScheduledEndDate:   item.ScheduledEndDate,
		LeadInspector:      teamMemberView(item.LeadInspector), Members: members,
		Assignments: questionAssignments, Documents: []generated.DocumentMetadataView{},
		History: history, Revision: item.Revision,
	}
}

func auditeeCoordinationView(
	item assignments.AuditeeCoordination,
) generated.AuditeeCoordinationView {
	return generated.AuditeeCoordinationView{
		AuditId: item.InspectionID, OrganizationId: item.OrganizationID,
		OrganizationName: item.OrganizationName, Title: item.Title,
		InspectionCategory: item.InspectionCategory,
		ScheduledStartDate: item.ScheduledStartDate, Status: string(item.Status),
		AlternativeDate: item.AlternativeDate, NextAction: item.NextAction,
		Revision: item.Revision,
	}
}

func preparationConfirmationView(item assignments.Preparation) generated.PreparationConfirmationView {
	return generated.PreparationConfirmationView{
		PlanningItemId: item.PlanningItemID, InspectionId: item.InspectionID,
		OrganizationId: item.OrganizationID, Status: string(item.Status),
		Revision: item.Revision, PreparationId: item.PreparationID,
		PreparationDigest:           item.PreparationDigest,
		SelectedQuestionCount:       int64(item.SelectedQuestionCount),
		ConfirmedAt:                 item.ConfirmedAt.UTC().Format(time.RFC3339Nano),
		ConfirmedAssignmentRevision: item.ConfirmedAssignmentRevision,
	}
}

func preparationEditPreviewView(item assignments.PreparationEditPreview) generated.PreparationEditPreviewView {
	questionAssignments := make([]generated.QuestionCoverageInput, 0, len(item.QuestionAssignments))
	for _, assignment := range item.QuestionAssignments {
		questionAssignments = append(questionAssignments, generated.QuestionCoverageInput{QuestionId: assignment.QuestionID, SubjectId: assignment.SubjectID})
	}
	return generated.PreparationEditPreviewView{
		PreviewId: item.PreviewID, AssignmentId: item.AssignmentID,
		AssignmentRevision: item.AssignmentRevision, EditKind: item.EditKind,
		Digest: item.Digest, ExpiresAt: item.ExpiresAt.UTC().Format(time.RFC3339Nano),
		MemberSubjectIds: append([]string(nil), item.MemberSubjectIDs...), QuestionAssignments: questionAssignments,
	}
}

func canonicalAssignmentView(item assignments.Assignment) generated.CanonicalAssignmentView {
	questionAssignments := make([]generated.QuestionCoverageInput, 0, len(item.QuestionAssignments))
	for _, assignment := range item.QuestionAssignments {
		questionAssignments = append(questionAssignments, generated.QuestionCoverageInput{
			QuestionId: assignment.QuestionID, SubjectId: assignment.SubjectID,
		})
	}
	view := generated.CanonicalAssignmentView{
		Id: item.ID, InspectionId: item.InspectionID, OrganizationId: item.OrganizationID,
		LeadSubjectId: item.LeadSubjectID, MemberSubjectIds: append([]string(nil), item.MemberSubjectIDs...),
		QuestionAssignments: questionAssignments, Status: string(item.Status),
		ScheduledStartDate: item.ScheduledStartDate, ScheduledEndDate: item.ScheduledEndDate,
		Revision: item.Revision, SelectedQuestionVersionIds: append([]string(nil), item.SelectedQuestionVersionIDs...),
	}
	if item.PreparationID != "" {
		view.PreparationId = &item.PreparationID
		view.PreparationDigest = &item.PreparationDigest
		if item.PreparationConfirmedAt != nil {
			confirmedAt := item.PreparationConfirmedAt.UTC().Format(time.RFC3339Nano)
			view.PreparationConfirmedAt = &confirmedAt
		}
		revision := item.PreparationConfirmedAssignmentRevision
		view.PreparationConfirmedAssignmentRevision = &revision
	}
	return view
}

func canonicalMaterializedAuditView(item assignments.MaterializedInspection) generated.CanonicalMaterializedAuditView {
	return generated.CanonicalMaterializedAuditView{
		InspectionId: item.InspectionID, AssignmentId: item.AssignmentID,
		PackageId: item.PackageID, PackageVersion: item.PackageVersion,
		PackageDigest: item.PackageDigest, Status: string(item.Status),
		NoticeWithheld: item.NoticeWithheld, AssignmentRevision: item.AssignmentRevision,
		ExpiresAt: item.ExpiresAt.UTC().Format(time.RFC3339Nano),
	}
}
