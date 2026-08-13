package httpapi

import (
	"net/http"
	"strings"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/httpapi/generated"
	"github.com/aviason/aviaSurveil/internal/reports"
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
	if err := api.requireCatalogRuntimeProfile(request.Context(), optionalString(input.Values.CatalogVersion)); err != nil {
		api.respond(writer, nil, err)
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
		"requestedBudget":    input.Values.RequestedBudget,
		"currency":           input.Values.Currency,
	}
	optionalDraftValue(values, "templateVersionId", input.Values.TemplateVersionId)
	optionalDraftValue(values, "scope", input.Values.Scope)
	optionalDraftValue(values, "catalogVersion", input.Values.CatalogVersion)
	optionalDraftValue(values, "scopeDraftId", input.Values.ScopeDraftId)
	optionalDraftValue(values, "selectionDigest", input.Values.SelectionDigest)
	optionalDraftValue(values, "providerScopeId", input.Values.ProviderScopeId)
	optionalDraftValue(values, "regulatedTargetId", input.Values.RegulatedTargetId)
	if len(input.Values.SelectedQuestionVersionIds) > 0 {
		values["selectedQuestionVersionIds"] = append([]string(nil), input.Values.SelectedQuestionVersionIds...)
	}
	if input.Values.EstimatedResourceRequirement != nil {
		values["estimatedResourceRequirement"] = *input.Values.EstimatedResourceRequirement
	}
	if len(input.Values.FormDistribution) > 0 {
		values["formDistribution"] = input.Values.FormDistribution
	}
	if len(input.Values.DomainDistribution) > 0 {
		values["domainDistribution"] = input.Values.DomainDistribution
	}
	record, err := api.application.CreatePlanningIntakeDraft(
		request.Context(),
		actor,
		application.CreatePlanningIntakeDraftCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			DraftID: optionalString(input.DraftId), OrganizationID: input.Values.OrganizationId,
			Values: values,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	organizationName := input.Values.OrganizationName
	if value, ok := record.Values["organizationName"].(string); ok && strings.TrimSpace(value) != "" {
		organizationName = value
	}
	writeJSON(writer, http.StatusCreated, generated.PlanningIntakeDraftView{
		Id: record.ID, OrganizationId: record.OrganizationID,
		OrganizationName: organizationName,
		ApplicationType:  input.Values.ApplicationType, Domain: input.Values.Domain,
		InspectionCategory: input.Values.InspectionCategory,
		NoticePolicy:       input.Values.NoticePolicy, Purpose: input.Values.Purpose,
		TriggerType: input.Values.TriggerType, RiskCategory: input.Values.RiskCategory,
		PlannedDate: input.Values.PlannedDate, Mode: input.Values.Mode,
		Location: input.Values.Location, TemplateVersionId: input.Values.TemplateVersionId,
		Scope: input.Values.Scope, CatalogVersion: draftStringValue(record.Values, "catalogVersion", input.Values.CatalogVersion),
		ScopeDraftId:                 draftStringValue(record.Values, "scopeDraftId", input.Values.ScopeDraftId),
		SelectionDigest:              draftStringValue(record.Values, "selectionDigest", input.Values.SelectionDigest),
		SelectedQuestionVersionIds:   stringSliceValue(record.Values, "selectedQuestionVersionIds", input.Values.SelectedQuestionVersionIds),
		EstimatedResourceRequirement: input.Values.EstimatedResourceRequirement,
		FormDistribution:             mapValue(record.Values, "formDistribution", input.Values.FormDistribution),
		DomainDistribution:           mapValue(record.Values, "domainDistribution", input.Values.DomainDistribution),
		ProviderScopeId:              draftStringValue(record.Values, "providerScopeId", input.Values.ProviderScopeId), RegulatedTargetId: draftStringValue(record.Values, "regulatedTargetId", input.Values.RegulatedTargetId),
		RequestedBudget: input.Values.RequestedBudget,
		Currency:        input.Values.Currency, Revision: record.Revision,
		SubmittedPlanningItemId: record.SubmittedPlanningItemID, UpdatedAt: record.UpdatedAt,
	})
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func optionalStringPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	copy := strings.TrimSpace(value)
	return &copy
}

func draftStringValue(values map[string]any, key string, fallback *string) *string {
	if value, ok := values[key].(string); ok {
		return optionalStringPointer(value)
	}
	return fallback
}

func stringSliceValue(values map[string]any, key string, fallback []string) []string {
	raw, ok := values[key]
	if !ok {
		return append([]string(nil), fallback...)
	}
	switch typed := raw.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		output := make([]string, 0, len(typed))
		for _, item := range typed {
			if value, ok := item.(string); ok {
				output = append(output, value)
			}
		}
		return output
	default:
		return append([]string(nil), fallback...)
	}
}

func mapValue(values map[string]any, key string, fallback map[string]any) map[string]any {
	raw, ok := values[key]
	if !ok {
		return fallback
	}
	if typed, ok := raw.(map[string]any); ok {
		return typed
	}
	return fallback
}

func optionalFloat(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func optionalFloatPointer(value float64) *float64 {
	if value == 0 {
		return nil
	}
	copy := value
	return &copy
}

func optionalDraftValue(values map[string]any, key string, value *string) {
	if normalized := optionalString(value); normalized != "" {
		values[key] = normalized
	}
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
		FindingIds:  append([]string{}, record.FindingIDs...),
		ContentHash: record.ContentHash, Version: record.Version,
		Status:   generated.ReportApprovalStatus(record.Status),
		Revision: record.Revision, IssuedAt: record.IssuedAt,
	})
}
