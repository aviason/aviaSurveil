package httpapi

import (
	"net/http"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi/generated"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/reports"
)

func (api *CanonicalAPI) createAdminOrganization(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.CreateAdminOrganizationInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if !validOptionalRevisionCommandHeaders(
		request,
		input.IdempotencyKey,
		nil,
	) || input.ExpectedRevision != nil {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	record, err := api.application.CreateAdminOrganization(
		request.Context(),
		actor,
		application.CreateAdminOrganizationCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			OrganizationID: input.OrganizationId, LegalName: input.LegalName,
			OrganizationType: input.OrganizationType,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writeJSON(writer, http.StatusCreated, generated.AdminOrganizationView{
		Id: record.ID, LegalName: record.LegalName,
		OrganizationType: record.OrganizationType, Status: record.Status,
		Scope: "CAA oversight", DetailAvailable: true, DisabledReason: nil,
	})
}

func (api *CanonicalAPI) createPlanningIntakeDraft(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.CreatePlanningIntakeDraftInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if !validOptionalRevisionCommandHeaders(
		request,
		input.IdempotencyKey,
		nil,
	) || input.ExpectedRevision != nil {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	values := map[string]any{
		"organizationId":     input.Values.OrganizationId,
		"organizationName":   input.Values.OrganizationName,
		"applicationType":    input.Values.ApplicationType,
		"domain":             input.Values.Domain,
		"inspectionCategory": input.Values.InspectionCategory,
		"noticePolicy":       input.Values.NoticePolicy,
		"purpose":            input.Values.Purpose,
		"triggerType":        input.Values.TriggerType,
		"riskCategory":       input.Values.RiskCategory,
		"plannedDate":        input.Values.PlannedDate,
		"mode":               input.Values.Mode,
		"location":           input.Values.Location,
		"templateVersionId":  input.Values.TemplateVersionId,
		"scope":              input.Values.Scope,
		"requestedBudget":    input.Values.RequestedBudget,
		"currency":           input.Values.Currency,
	}
	record, err := api.application.CreatePlanningIntakeDraft(
		request.Context(),
		actor,
		application.CreatePlanningIntakeDraftCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			DraftID: input.DraftId, OrganizationID: input.Values.OrganizationId,
			Values: values,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writeJSON(writer, http.StatusCreated, generated.PlanningIntakeDraftView{
		Id: record.ID, OrganizationId: record.OrganizationID,
		OrganizationName: input.Values.OrganizationName,
		ApplicationType:  input.Values.ApplicationType, Domain: input.Values.Domain,
		InspectionCategory: input.Values.InspectionCategory,
		NoticePolicy:       input.Values.NoticePolicy, Purpose: input.Values.Purpose,
		TriggerType: input.Values.TriggerType, RiskCategory: input.Values.RiskCategory,
		PlannedDate: input.Values.PlannedDate, Mode: input.Values.Mode,
		Location: input.Values.Location, TemplateVersionId: input.Values.TemplateVersionId,
		Scope: input.Values.Scope, RequestedBudget: input.Values.RequestedBudget,
		Currency: input.Values.Currency, Revision: record.Revision,
		SubmittedPlanningItemId: record.SubmittedPlanningItemID, UpdatedAt: record.UpdatedAt,
	})
}

func (api *CanonicalAPI) createReminderRule(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.CreateReminderRuleInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if !validOptionalRevisionCommandHeaders(request, input.IdempotencyKey, nil) ||
		input.ExpectedRevision != nil {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	record, err := api.application.CreateReminderRule(
		request.Context(),
		actor,
		application.CreateReminderRuleCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			RuleID: input.RuleId, Label: input.Label, OffsetDays: input.OffsetDays,
			Channel: input.Channel,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(record.Revision))
	writeJSON(writer, http.StatusCreated, generated.ReminderRuleView{
		Id: record.ID, Label: record.Label, OffsetDays: record.OffsetDays,
		Channel: record.Channel, Status: record.Status, Revision: record.Revision,
	})
}

func (api *CanonicalAPI) createAuditWorkspace(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.CreateAuditWorkspaceInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if !validOptionalRevisionCommandHeaders(request, input.IdempotencyKey, nil) ||
		request.Header.Get("If-Match") != "" {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	expiresAt, err := time.Parse(time.RFC3339, input.ExpiresAt)
	if err != nil {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	questions := make([]application.AuditWorkspaceQuestion, 0, len(input.Questions))
	for _, question := range input.Questions {
		questions = append(questions, application.AuditWorkspaceQuestion{
			QuestionID: question.QuestionId,
			AssignedInspectorSubjectIDs: append(
				[]string(nil),
				question.AssignedInspectorSubjectIds...,
			),
		})
	}
	record, err := api.application.CreateAuditWorkspace(
		request.Context(),
		actor,
		application.CreateAuditWorkspaceCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			PlanningItemID:           input.PlanningItemId,
			ExpectedPlanningRevision: input.ExpectedPlanningRevision,
			AuditID:                  input.AuditId, AssignmentID: input.AssignmentId,
			PackageID: input.PackageId, PackageDraftID: input.PackageDraftId,
			TemplateID: input.TemplateId, TemplateVersionID: input.TemplateVersionId,
			LeadInspectorSubjectID: input.LeadInspectorSubjectId,
			MemberSubjectIDs:       append([]string(nil), input.MemberSubjectIds...),
			ScheduledStartDate:     input.ScheduledStartDate,
			ScheduledEndDate:       input.ScheduledEndDate,
			ExpiresAt:              expiresAt, Questions: questions,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(record.Revision))
	writeJSON(writer, http.StatusCreated, generated.AuditWorkspaceView{
		AuditId: record.AuditID, AssignmentId: record.AssignmentID,
		PackageId: record.PackageID, PackageDraftId: record.PackageDraftID,
		TemplateVersionId: record.TemplateVersionID,
		PackageVersion:    record.PackageVersion, Revision: record.Revision,
	})
}

func (api *CanonicalAPI) createReportVersion(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.CreateReportVersionInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if !validOptionalRevisionCommandHeaders(request, input.IdempotencyKey, nil) ||
		input.ExpectedRevision != nil {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	record, err := api.application.CreateReportVersion(
		request.Context(),
		actor,
		application.CreateReportVersionCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			ReportVersionID: input.ReportVersionId, ReportID: input.ReportId,
			AuditID: input.AuditId, Kind: reports.Kind(input.Kind),
			Version: input.Version, Status: input.Status,
			FindingIDs: append([]string(nil), input.FindingIds...),
			Content:    input.Content,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(record.Revision))
	writeJSON(writer, http.StatusCreated, generated.ReportVersionView{
		ReportVersionId: record.ReportVersionID, ReportId: record.ReportID,
		OrganizationId: record.OrganizationID, AuditId: record.AuditID,
		FindingIds:  append([]string(nil), record.FindingIDs...),
		ContentHash: record.ContentHash, Version: record.Version,
		Status:   generated.ReportApprovalStatus(record.Status),
		Revision: record.Revision, IssuedAt: record.IssuedAt,
	})
}
