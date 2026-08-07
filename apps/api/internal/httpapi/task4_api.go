package httpapi

import (
	"net/http"
	"strings"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/assignments"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi/generated"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/inspections"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/planning"
	"github.com/go-chi/chi/v5"
)

func (api *CanonicalAPI) getPlanningIntakeDraft(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	draft, err := api.planning.GetIntakeDraft(
		request.Context(), actor, chi.URLParam(request, "draftId"),
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
	draftID := chi.URLParam(request, "draftId")
	if draftID != input.DraftId || !validRevisionCommandHeaders(
		request, input.IdempotencyKey, input.ExpectedRevision,
	) {
		api.respond(writer, nil, application.ErrInvalid)
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
	draftID := chi.URLParam(request, "draftId")
	if draftID != input.DraftId || !validRevisionCommandHeaders(
		request, input.IdempotencyKey, input.ExpectedRevision,
	) {
		api.respond(writer, nil, application.ErrInvalid)
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
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	draft, err := api.packageDrafts.Get(
		request.Context(), actor, chi.URLParam(request, "packageDraftId"),
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
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.SaveInspectionPackageDraftInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	draftID := chi.URLParam(request, "packageDraftId")
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
		request.Context(), actor, chi.URLParam(request, "subjectId"),
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
		request.Context(), actor, chi.URLParam(request, "auditId"),
	)
	api.respond(writer, inspectionTeamAuditView(item), err)
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
	auditID := chi.URLParam(request, "auditId")
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
