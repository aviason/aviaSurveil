package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/administration"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/assistant"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi/generated"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/go-chi/chi/v5"
)

func (api *CanonicalAPI) getRiskOverview(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	organizationID := strings.TrimSpace(request.URL.Query().Get("organizationId"))
	record, err := api.risk.GetOverview(request.Context(), actor, organizationID)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	var projectedOrganizationID *string
	if record.OrganizationID != "" {
		projectedOrganizationID = &record.OrganizationID
	}
	api.respond(writer, generated.RiskOverviewView{
		OrganizationId:      projectedOrganizationID,
		OverdueFindingCount: int64(record.OverdueFindingCount),
		OpenFindingCount:    int64(record.OpenFindingCount),
		RepeatFindingCount:  int64(record.RepeatFindingCount),
		Revision:            int64(record.Revision),
	}, nil)
}

func (api *CanonicalAPI) getRiskManagementProjection(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	record, err := api.risk.GetManagementProjection(request.Context(), actor)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	body, err := json.Marshal(record)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	var output generated.RiskManagementProjectionView
	if err := json.Unmarshal(body, &output); err != nil {
		api.respond(writer, nil, err)
		return
	}
	api.respond(writer, output, nil)
}

func (api *CanonicalAPI) listAdministrationScreenProjections(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	records, err := api.administration.ListScreenProjections(
		request.Context(), actor,
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	items := make([]generated.AdministrationScreenProjection, 0, len(records))
	for _, record := range records {
		item, err := administrationScreenProjection(record)
		if err != nil {
			api.respond(writer, nil, err)
			return
		}
		items = append(items, item)
	}
	api.respond(writer, generated.AdministrationScreenProjectionList(items), nil)
}

func (api *CanonicalAPI) getAdministrationScreenProjection(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	record, err := api.administration.GetScreenProjection(
		request.Context(), actor, chi.URLParam(request, "screenId"),
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	output, err := administrationScreenProjection(record)
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) invokeAdministrationVisibleAction(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.InvokeAdministrationVisibleActionInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	screenID := chi.URLParam(request, "screenId")
	actionID := chi.URLParam(request, "actionId")
	if input.ScreenId != screenID || input.ActionId != actionID ||
		strings.TrimSpace(input.OperationId) == "" ||
		!validOptionalRevisionCommandHeaders(
			request, input.IdempotencyKey, input.ExpectedRevision,
		) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	record, err := api.administration.InvokeVisibleAction(
		request.Context(), actor, screenID, actionID,
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	effect, err := json.Marshal(record.Effect)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	api.respond(writer, generated.VisibleActionResult{
		ScreenId: record.ScreenID,
		ActionId: record.ActionID,
		Effect:   effect,
	}, nil)
}

func (api *CanonicalAPI) listAdminReportDefinitions(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	records, err := api.administration.ListReportDefinitions(
		request.Context(), actor, request.URL.Query().Get("search"),
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	items := make([]generated.AdminReportDefinitionView, 0, len(records))
	for _, record := range records {
		items = append(items, generated.AdminReportDefinitionView{
			Id: record.ID, Title: record.Title, Description: record.Description,
			PackageFields: append([]string(nil), record.PackageFields...),
			ActionReason:  record.ActionReason,
		})
	}
	api.respond(writer, generated.AdminReportDefinitionPage{
		Items: items, NextCursor: nil,
	}, nil)
}

func (api *CanonicalAPI) listAdminAccessDirectory(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	role := identity.Role(strings.TrimSpace(request.URL.Query().Get("role")))
	limit := 0
	if rawLimit := strings.TrimSpace(request.URL.Query().Get("limit")); rawLimit != "" {
		parsed := optionalIntQuery(request, "limit")
		if parsed == nil || *parsed < 1 || *parsed > 25 {
			api.respond(writer, nil, administration.ErrInvalid)
			return
		}
		limit = int(*parsed)
	}
	page, err := api.administration.ListAccessDirectory(
		request.Context(),
		actor,
		administration.AccessDirectoryFilters{
			Search:        request.URL.Query().Get("search"),
			Role:          role,
			Organization:  request.URL.Query().Get("organizationId"),
			AccountStatus: request.URL.Query().Get("accountStatus"),
			Cursor:        request.URL.Query().Get("cursor"),
			Limit:         limit,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	items := make([]generated.AdminAccessDirectoryEntryView, 0, len(page.Items))
	for _, record := range page.Items {
		var organizationID *string
		if record.OrganizationID != "" {
			value := record.OrganizationID
			organizationID = &value
		}
		var membershipID *string
		if record.MembershipID != "" {
			value := record.MembershipID
			membershipID = &value
		}
		var lastSuccessfulSessionAt *string
		if record.LastSuccessfulSession != nil {
			value := record.LastSuccessfulSession.UTC().Format(time.RFC3339Nano)
			lastSuccessfulSessionAt = &value
		}
		roles := make([]generated.Role, len(record.Roles))
		for index, recordRole := range record.Roles {
			roles[index] = generated.Role(recordRole)
		}
		items = append(items, generated.AdminAccessDirectoryEntryView{
			SubjectId: record.SubjectID, DisplayName: record.DisplayName,
			Roles: roles, OrganizationId: organizationID, Email: record.Email,
			MfaEnrolled: record.MFAEnrolled, MfaState: record.MFAState,
			RequiredActions:         append([]string(nil), record.RequiredActions...),
			InvitationState:         record.InvitationState,
			AccountStatus:           record.AccountStatus,
			ApplicationProfileState: record.ApplicationProfile,
			MembershipId:            membershipID, MembershipState: record.MembershipState,
			MembershipRevision:      record.MembershipRevision,
			MembershipDrift:         record.MembershipDrift,
			LastSuccessfulSessionAt: lastSuccessfulSessionAt,
			ProviderObservedAt:      record.ProviderObservedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	var nextCursor *string
	if page.NextCursor != "" {
		nextCursor = &page.NextCursor
	}
	api.respond(writer, generated.AdminAccessDirectoryPage{
		Items: items, NextCursor: nextCursor,
		ConsistencyToken: page.ConsistencyToken,
		ProviderCalls:    int64(page.ProviderCalls),
	}, nil)
}

func (api *CanonicalAPI) requestUserLifecycle(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.RequestUserLifecycleInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if api.users == nil ||
		strings.TrimSpace(request.Header.Get("Idempotency-Key")) == "" ||
		request.Header.Get("Idempotency-Key") != input.IdempotencyKey {
		api.respond(writer, nil, administration.ErrInvalid)
		return
	}
	roles := make([]identity.Role, len(input.Roles))
	for index, role := range input.Roles {
		roles[index] = identity.Role(role)
	}
	command := administration.RequestUserLifecycleCommand{
		OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
		Action: administration.UserLifecycleAction(input.Action),
		Roles:  roles, OrganizationID: input.OrganizationId,
		Reason:                     input.Reason,
		ExpectedMembershipRevision: input.ExpectedMembershipRevision,
	}
	if input.SubjectId != nil {
		command.SubjectID = *input.SubjectId
	}
	if input.Email != nil {
		command.Email = *input.Email
	}
	if input.DisplayName != nil {
		command.DisplayName = *input.DisplayName
	}
	if input.EffectiveAt != nil {
		effectiveAt, err := time.Parse(time.RFC3339Nano, *input.EffectiveAt)
		if err != nil {
			api.respond(writer, nil, administration.ErrInvalid)
			return
		}
		command.EffectiveAt = &effectiveAt
	}
	record, err := api.users.RequestLifecycle(
		request.Context(),
		actor,
		command,
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writeJSON(writer, http.StatusAccepted, userLifecycleRequestView(record))
}

func (api *CanonicalAPI) getUserLifecycleRequest(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	if api.users == nil {
		api.respond(writer, nil, administration.ErrInvalid)
		return
	}
	record, err := api.users.GetLifecycle(
		request.Context(),
		actor,
		chi.URLParam(request, "requestId"),
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	api.respond(writer, userLifecycleRequestView(record), nil)
}

func userLifecycleRequestView(
	record administration.UserLifecycleRequest,
) generated.UserLifecycleRequestView {
	var subjectID, email, displayName, failureReason *string
	var membershipID, effectiveAt, providerFailureClass, providerAcknowledgedAt *string
	if record.SubjectID != "" {
		subjectID = &record.SubjectID
	}
	if record.Email != "" {
		email = &record.Email
	}
	if record.DisplayName != "" {
		displayName = &record.DisplayName
	}
	if record.FailureReason != "" {
		failureReason = &record.FailureReason
	}
	if record.MembershipID != "" {
		membershipID = &record.MembershipID
	}
	if record.EffectiveAt != nil {
		value := record.EffectiveAt.UTC().Format(time.RFC3339Nano)
		effectiveAt = &value
	}
	if record.ProviderFailureClass != "" {
		providerFailureClass = &record.ProviderFailureClass
	}
	if record.ProviderAcknowledgedAt != nil {
		value := record.ProviderAcknowledgedAt.UTC().Format(time.RFC3339Nano)
		providerAcknowledgedAt = &value
	}
	roles := make([]generated.Role, len(record.Roles))
	for index, role := range record.Roles {
		roles[index] = generated.Role(role)
	}
	return generated.UserLifecycleRequestView{
		Id: record.ID, SubjectId: subjectID, Action: string(record.Action),
		Roles: roles, OrganizationId: record.OrganizationID,
		Email: email, DisplayName: displayName, Status: string(record.Status),
		IdempotencyKey:              record.IdempotencyKey,
		ExpectedMembershipRevision:  record.ExpectedMembershipRevision,
		ResultingMembershipRevision: record.ResultingMembershipRevision,
		MembershipId:                membershipID,
		Reason:                      record.Reason,
		EffectiveAt:                 effectiveAt,
		ProviderFailureClass:        providerFailureClass,
		ProviderAcknowledgedAt:      providerAcknowledgedAt,
		AttemptCount:                int64(record.AttemptCount),
		RequestedBySubjectId:        record.RequestedBy,
		OutboxMessageId:             record.OutboxMessageID,
		FailureReason:               failureReason,
		CreatedAt:                   record.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:                   record.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (api *CanonicalAPI) listAdminOrganizations(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	records, err := api.administration.ListOrganizations(
		request.Context(), actor, administration.OrganizationFilters{
			Search:           request.URL.Query().Get("search"),
			OrganizationType: request.URL.Query().Get("organizationType"),
			Status:           request.URL.Query().Get("status"),
			Scope:            request.URL.Query().Get("scope"),
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	items := make([]generated.AdminOrganizationView, 0, len(records))
	for _, record := range records {
		items = append(items, adminOrganizationView(record))
	}
	api.respond(writer, generated.AdminOrganizationPage{
		Items: items, NextCursor: nil,
	}, nil)
}

func (api *CanonicalAPI) getAdminOrganization(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	record, err := api.administration.GetOrganization(
		request.Context(), actor, chi.URLParam(request, "organizationId"),
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	api.respond(writer, adminOrganizationView(record), nil)
}

func (api *CanonicalAPI) listAdminAuditEvents(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	records, err := api.administration.ListAuditEvents(
		request.Context(), actor, administration.AuditEventFilters{
			Actor:    request.URL.Query().Get("actor"),
			Action:   request.URL.Query().Get("action"),
			Entity:   request.URL.Query().Get("entity"),
			System:   request.URL.Query().Get("system"),
			DateText: request.URL.Query().Get("dateText"),
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	items := make([]generated.AuditEventView, 0, len(records))
	for _, record := range records {
		var actorRole *string
		if record.ActorRole != "" {
			value := string(record.ActorRole)
			actorRole = &value
		}
		var beforeStatus, afterStatus, reason *string
		if record.BeforeStatus != "" {
			value := record.BeforeStatus
			beforeStatus = &value
		}
		if record.AfterStatus != "" {
			value := record.AfterStatus
			afterStatus = &value
		}
		if record.Reason != "" {
			value := record.Reason
			reason = &value
		}
		items = append(items, generated.AuditEventView{
			EventId:    record.EventID,
			OccurredAt: record.OccurredAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
			ActorRole:  actorRole,
			Action:     record.Action, EntityType: record.EntityType,
			EntityId: record.EntityID, BeforeStatus: beforeStatus,
			AfterStatus: afterStatus, Reason: reason,
		})
	}
	api.respond(writer, generated.ListAuditEventsOutput{
		Items: items, NextCursor: nil,
	}, nil)
}

func (api *CanonicalAPI) getAssistantGuidance(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	record, err := api.assistant.GetGuidance(actor)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	api.respond(writer, generated.AssistantGuidance{
		AdvisoryOnly: record.AdvisoryOnly,
		ProhibitedActions: append(
			[]string(nil), record.ProhibitedActions...,
		),
	}, nil)
}

func (api *CanonicalAPI) createAssistantDraft(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.CreateAssistantDraftInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if !validOptionalRevisionCommandHeaders(
		request, input.IdempotencyKey, input.ExpectedRevision,
	) {
		api.respond(writer, nil, assistant.ErrInvalid)
		return
	}
	record, err := api.assistant.CreateDraft(
		request.Context(),
		actor,
		assistant.CreateDraftCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			ExpectedRevision: input.ExpectedRevision,
			FindingID:        input.FindingId, Prompt: input.Prompt,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	api.respondCreated(writer, generated.AssistantDraftView{
		Id: record.ID, FindingId: record.FindingID, Prompt: record.Prompt,
		Draft: record.Draft, AdvisoryOnly: record.AdvisoryOnly,
		CanCreateFinding: record.CanCreateFinding,
		CanSetSeverity:   record.CanSetSeverity,
		CanCloseFinding:  record.CanCloseFinding,
	}, nil)
}

func administrationScreenProjection(
	record administration.ScreenProjection,
) (generated.AdministrationScreenProjection, error) {
	var organizationID, directRecordID *string
	if record.OrganizationID != "" {
		value := record.OrganizationID
		organizationID = &value
	}
	if record.DirectRecordID != "" {
		value := record.DirectRecordID
		directRecordID = &value
	}
	actions := make([]generated.VisibleScreenAction, 0, len(record.VisibleActions))
	for _, action := range record.VisibleActions {
		effect, err := json.Marshal(action.Effect)
		if err != nil {
			return generated.AdministrationScreenProjection{}, fmt.Errorf(
				"encode visible action %s: %w", action.ID, err,
			)
		}
		actions = append(actions, generated.VisibleScreenAction{
			Id: action.ID, Label: action.Label, Kind: string(action.Kind), Effect: effect,
		})
	}
	return generated.AdministrationScreenProjection{
		ScreenId: record.ScreenID, OrganizationId: organizationID,
		DirectRecordId: directRecordID, State: string(record.State),
		Overdue: record.Overdue, VersionHistory: record.VersionHistory,
		VisibleActions: actions,
	}, nil
}

func adminOrganizationView(
	record administration.OrganizationProjection,
) generated.AdminOrganizationView {
	var disabledReason *string
	if record.DisabledReason != "" {
		value := record.DisabledReason
		disabledReason = &value
	}
	return generated.AdminOrganizationView{
		Id: record.ID, LegalName: record.LegalName,
		OrganizationType: record.OrganizationType, Status: record.Status,
		Scope: record.Scope, DetailAvailable: record.DetailAvailable,
		DisabledReason: disabledReason,
	}
}
