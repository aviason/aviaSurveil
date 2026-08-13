package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/auditlog"
	auditstore "github.com/aviason/aviaSurveil/internal/auditlog/store/postgres"
	"github.com/aviason/aviaSurveil/internal/configuration"
	configurationstore "github.com/aviason/aviaSurveil/internal/configuration/store/postgres"
	"github.com/aviason/aviaSurveil/internal/httpapi/generated"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/organizations"
	"github.com/aviason/aviaSurveil/internal/planning"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func (api *CanonicalAPI) listOrganizations(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	records, err := organizations.NewPostgresService(api.pool).ListRegistry(
		request.Context(), actor, boundedPageLimit(optionalIntQuery(request, "limit")),
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	etag, err := strongProjectionETag(records)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	items := make([]generated.OrganizationSummary, 0, len(records))
	for _, record := range records {
		item := generated.OrganizationSummary{
			Id: record.ID, LegalName: record.LegalName, OrganizationType: record.OrganizationType,
			Status: record.Status, OpenFindingCount: record.OpenFindingCount, Revision: record.Revision,
		}
		if record.LastAuditDate != "" {
			value := record.LastAuditDate
			item.LastAuditDate = &value
		}
		if record.NextAuditDate != "" {
			value := record.NextAuditDate
			item.NextAuditDate = &value
		}
		items = append(items, item)
	}
	writer.Header().Set("ETag", etag)
	api.respond(writer, generated.ListOrganizationsOutput{Items: items}, nil)
}

func strongProjectionETag(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode projection ETag source: %w", err)
	}
	hash := fnv.New64a()
	_, _ = hash.Write(encoded)
	return fmt.Sprintf(`"rev-%d"`, hash.Sum64()), nil
}

func (api *CanonicalAPI) getMyProfile(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	if api.profiles == nil {
		api.respond(writer, nil, fmt.Errorf("profile service is unavailable"))
		return
	}
	profile, err := api.profiles.GetProfile(request.Context(), actor)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(profile.Revision))
	api.respond(writer, profileView(profile), nil)
}

func (api *CanonicalAPI) updateMyProfile(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.UpdateMyProfileInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if api.profiles == nil {
		api.respond(writer, nil, fmt.Errorf("profile service is unavailable"))
		return
	}
	idempotencyKey := strings.TrimSpace(request.Header.Get("Idempotency-Key"))
	if input.ExpectedRevision == nil || idempotencyKey == "" || idempotencyKey != input.IdempotencyKey {
		api.respond(writer, nil, identity.ErrInvalidProfile)
		return
	}
	if request.Header.Get("If-Match") != strongRevisionETag(*input.ExpectedRevision) {
		api.respond(writer, nil, identity.ErrPrecondition)
		return
	}
	profile, err := api.profiles.UpdateProfile(request.Context(), actor, identity.UpdateProfileCommand{
		OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
		ExpectedRevision: *input.ExpectedRevision, DisplayName: input.DisplayName,
	})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(profile.Revision))
	api.respond(writer, profileView(profile), nil)
}

func strongRevisionETag(revision int64) string {
	return fmt.Sprintf(`"rev-%d"`, revision)
}

func profileView(profile identity.Profile) generated.ProfileView {
	var organizationID *string
	if profile.OrganizationID != "" {
		value := profile.OrganizationID
		organizationID = &value
	}
	return generated.ProfileView{
		SubjectId: profile.SubjectID, Role: generated.Role(profile.Role),
		OrganizationId: organizationID, DisplayName: profile.DisplayName, Revision: profile.Revision,
	}
}

func (api *CanonicalAPI) listPlanningItems(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	items, err := api.planning.List(request.Context(), actor, boundedPageLimit(optionalIntQuery(request, "limit")))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	views := make([]generated.PlanningItemView, 0, len(items))
	for _, item := range items {
		views = append(views, planningView(item))
	}
	api.respond(writer, generated.ListPlanningItemsOutput{Items: views}, nil)
}

func (api *CanonicalAPI) decidePlanningItem(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.PlanningDecisionInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if input.PlanningItemId != chi.URLParam(request, "id") {
		api.respond(writer, nil, fmt.Errorf("%w: planning path and body must match", application.ErrInvalid))
		return
	}
	item, err := api.planning.Decide(request.Context(), actor, planning.DecideCommand{
		OperationID: input.OperationId, PlanningItemID: input.PlanningItemId,
		ExpectedRevision: input.ExpectedPlanningRevision, Decision: planning.Decision(input.Decision),
		Reason: input.Reason, ExpectedSubmittedScopeSnapshotID: input.ExpectedSubmittedScopeSnapshotId,
		ExpectedPlanningSnapshotDigest: input.ExpectedPlanningSnapshotDigest,
	})
	api.respond(writer, planningView(item), err)
}

func (api *CanonicalAPI) listChecklistTemplateVersions(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	if !configuration.CanReadChecklistTemplateVersionDetail(actor) {
		api.respond(writer, nil, fmt.Errorf("%w: Admin configuration authority is required", application.ErrForbidden))
		return
	}
	records, err := configurationstore.New(api.pool).ListChecklistTemplateVersions(request.Context(), boundedPageLimit(optionalIntQuery(request, "limit")))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	items := make([]generated.ChecklistTemplateVersionView, 0, len(records))
	for _, record := range records {
		items = append(items, generated.ChecklistTemplateVersionView{
			Id: record.ID, TemplateId: record.TemplateID, Title: record.Title,
			Version: int64(record.Version), Status: "PUBLISHED",
			PublishedAt:   record.PublishedAt.Time.UTC().Format(time.RFC3339Nano),
			QuestionCount: record.QuestionCount,
		})
	}
	api.respond(writer, generated.ListChecklistTemplateVersionsOutput{Items: items}, nil)
}

func (api *CanonicalAPI) getChecklistTemplateVersion(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	if !configuration.CanReadChecklistTemplateVersionDetail(actor) {
		api.respond(writer, nil, fmt.Errorf("%w: Admin configuration authority is required", application.ErrForbidden))
		return
	}
	record, err := configurationstore.New(api.pool).GetChecklistTemplateVersion(request.Context(), chi.URLParam(request, "templateVersionId"))
	if err != nil {
		api.respond(writer, nil, checklistTemplateVersionDetailStoreError(err))
		return
	}
	view, err := checklistTemplateVersionDetailView(checklistTemplateVersionRecord{
		ID: record.ID, TemplateID: record.TemplateID, Title: record.Title,
		Version: record.Version, PublishedAt: record.PublishedAt.Time.UTC(),
		QuestionCount: record.QuestionCount, Snapshot: record.Snapshot,
	})
	api.respond(writer, view, err)
}

type checklistTemplateVersionRecord struct {
	ID            string
	TemplateID    string
	Title         string
	Version       int32
	PublishedAt   time.Time
	QuestionCount int64
	Snapshot      []byte
}

type checklistTemplateVersionSnapshot struct {
	Questions []checklistTemplateQuestionSnapshot `json:"questions"`
}

type checklistTemplateQuestionSnapshot struct {
	ID                  string                      `json:"id"`
	SectionID           string                      `json:"sectionId"`
	Prompt              string                      `json:"prompt"`
	RegulatoryReference string                      `json:"regulatoryReference"`
	ExpectedEvidence    string                      `json:"expectedEvidence"`
	AllowedAnswers      []generated.ChecklistAnswer `json:"allowedAnswers"`
	CommentRequiredFor  []generated.ChecklistAnswer `json:"commentRequiredFor"`
}

func checklistTemplateVersionDetailStoreError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return application.ErrNotFound
	}
	return err
}

func checklistTemplateVersionDetailView(record checklistTemplateVersionRecord) (generated.ChecklistTemplateVersionDetailView, error) {
	if strings.TrimSpace(record.ID) == "" || strings.TrimSpace(record.TemplateID) == "" || strings.TrimSpace(record.Title) == "" {
		return generated.ChecklistTemplateVersionDetailView{}, fmt.Errorf("%w: checklist template version identity is required", application.ErrInvalid)
	}
	var snapshot checklistTemplateVersionSnapshot
	if err := json.Unmarshal(record.Snapshot, &snapshot); err != nil {
		return generated.ChecklistTemplateVersionDetailView{}, fmt.Errorf("decode checklist template snapshot: %w", err)
	}
	if len(snapshot.Questions) == 0 {
		return generated.ChecklistTemplateVersionDetailView{}, fmt.Errorf("%w: checklist template snapshot must include questions", application.ErrInvalid)
	}
	questionCount := record.QuestionCount
	if questionCount == 0 {
		questionCount = int64(len(snapshot.Questions))
	}
	if questionCount != int64(len(snapshot.Questions)) {
		return generated.ChecklistTemplateVersionDetailView{}, fmt.Errorf("%w: checklist template snapshot question count mismatch", application.ErrInvalid)
	}
	view := generated.ChecklistTemplateVersionDetailView{
		Id: record.ID, TemplateId: record.TemplateID, Title: record.Title,
		Version: int64(record.Version), Status: "PUBLISHED",
		PublishedAt:   record.PublishedAt.UTC().Format(time.RFC3339Nano),
		QuestionCount: questionCount,
		Questions:     make([]generated.ChecklistTemplateQuestionView, 0, len(snapshot.Questions)),
	}
	for index, question := range snapshot.Questions {
		if strings.TrimSpace(question.ID) == "" || strings.TrimSpace(question.SectionID) == "" || strings.TrimSpace(question.Prompt) == "" ||
			len(question.AllowedAnswers) == 0 || len(question.CommentRequiredFor) == 0 {
			return generated.ChecklistTemplateVersionDetailView{}, fmt.Errorf("%w: checklist template snapshot question %d is incomplete", application.ErrInvalid, index+1)
		}
		regulatoryReference := optionalTemplateText(question.RegulatoryReference)
		expectedEvidence := optionalTemplateText(question.ExpectedEvidence)
		view.Questions = append(view.Questions, generated.ChecklistTemplateQuestionView{
			Id: question.ID, SectionId: question.SectionID, Prompt: question.Prompt,
			RegulatoryReference: regulatoryReference, ExpectedEvidence: expectedEvidence,
			AllowedAnswers:     append([]generated.ChecklistAnswer(nil), question.AllowedAnswers...),
			CommentRequiredFor: append([]generated.ChecklistAnswer(nil), question.CommentRequiredFor...),
		})
	}
	return view, nil
}

func optionalTemplateText(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func (api *CanonicalAPI) listReminderRules(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	if !configuration.CanPreview(actor) {
		api.respond(writer, nil, fmt.Errorf("%w: Admin configuration authority is required", application.ErrForbidden))
		return
	}
	records, err := configurationstore.New(api.pool).ListReminderRules(request.Context(), boundedPageLimit(optionalIntQuery(request, "limit")))
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	items := make([]generated.ReminderRuleView, 0, len(records))
	for _, record := range records {
		items = append(items, generated.ReminderRuleView{
			Id: record.ID, Label: record.Label, OffsetDays: int64(record.OffsetDays),
			Channel: record.Channel, Status: record.Status, Revision: record.Revision,
		})
	}
	api.respond(writer, generated.ListReminderRulesOutput{Items: items}, nil)
}

func (api *CanonicalAPI) listAuditEvents(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	if !auditlog.CanReadInternal(actor) {
		api.respond(writer, nil, fmt.Errorf("%w: Internal CAA audit-trail authority is required", application.ErrForbidden))
		return
	}
	records, err := auditstore.New(api.pool).ListAuditEvents(request.Context(), auditstore.ListAuditEventsParams{
		EntityTypeFilter: valueOr(optionalQuery(request, "entityType"), ""),
		EntityIDFilter:   valueOr(optionalQuery(request, "entityId"), ""),
		ResultLimit:      boundedPageLimit(optionalIntQuery(request, "limit")),
	})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	items := make([]generated.AuditEventView, 0, len(records))
	for _, record := range records {
		items = append(items, generated.AuditEventView{
			EventId: record.EventID, OccurredAt: record.OccurredAt.Time.UTC().Format(time.RFC3339Nano),
			ActorRole: record.ActorRole, Action: record.Action, EntityType: record.EntityType,
			EntityId: record.EntityID, BeforeStatus: record.BeforeStatus,
			AfterStatus: record.AfterStatus, Reason: record.Reason,
		})
	}
	api.respond(writer, generated.ListAuditEventsOutput{Items: items}, nil)
}

func (api *CanonicalAPI) listAdminRegulatoryReferences(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	records, err := api.adminWorkspace.ListRegulatoryReferences(
		request.Context(), actor,
		valueOr(optionalQuery(request, "search"), ""),
		valueOr(optionalQuery(request, "status"), ""),
		int(boundedPageLimit(optionalIntQuery(request, "limit"))),
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	items := make([]generated.AdminRegulatoryReferenceView, 0, len(records))
	for _, record := range records {
		items = append(items, generated.AdminRegulatoryReferenceView{
			Id: record.ID, Title: record.Title, Version: record.Version,
			Status: record.Status, EffectiveDate: record.EffectiveDate,
			ConfiguredRules: append([]string(nil), record.ConfiguredRules...),
			ChangeHistory:   append([]string(nil), record.ChangeHistory...),
			Mappings:        adminRegulatoryMappingViews(record.Mappings),
		})
	}
	etag, err := strongProjectionETag(items)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", etag)
	api.respond(writer, generated.AdminRegulatoryReferencePage{Items: items}, nil)
}

func (api *CanonicalAPI) listAdminTemplateMasters(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	records, err := api.adminWorkspace.ListTemplateMasters(
		request.Context(), actor,
		int(boundedPageLimit(optionalIntQuery(request, "limit"))),
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	items := make([]generated.AdminTemplateMasterView, 0, len(records))
	for _, record := range records {
		items = append(items, generated.AdminTemplateMasterView{
			Id: record.ID, Title: record.Title, PublishedVersionId: record.PublishedVersionID,
			Status: record.Status, Owner: record.Owner, ItemCount: record.ItemCount,
			PreviewPath: record.PreviewPath, DisabledReason: record.DisabledReason,
			Revision: record.Revision,
		})
	}
	etag, err := strongProjectionETag(items)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", etag)
	api.respond(writer, generated.AdminTemplateMasterPage{Items: items}, nil)
}

func (api *CanonicalAPI) listAdminQuestions(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	records, err := api.adminWorkspace.ListQuestions(
		request.Context(), actor, valueOr(optionalQuery(request, "search"), ""),
		int(boundedPageLimit(optionalIntQuery(request, "limit"))),
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	items := make([]generated.AdminQuestionView, 0, len(records))
	for _, record := range records {
		items = append(items, adminQuestionView(record))
	}
	etag, err := strongProjectionETag(items)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", etag)
	api.respond(writer, generated.AdminQuestionPage{Items: items}, nil)
}

func (api *CanonicalAPI) createAdminQuestion(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.CreateAdminQuestionInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if !validOptionalRevisionCommandHeaders(
		request, input.IdempotencyKey, input.ExpectedRevision,
	) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	question, err := api.application.CreateAdminQuestion(
		request.Context(), actor, application.CreateAdminQuestionCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			ExpectedRevision: input.ExpectedRevision, Prompt: input.Prompt,
			ConfiguredReference: input.ConfiguredReference,
			ExpectedEvidence:    input.ExpectedEvidence,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(question.Revision))
	writeJSON(writer, http.StatusCreated, adminQuestionView(question))
}

func (api *CanonicalAPI) getAdminTemplate(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	template, err := api.adminWorkspace.GetTemplate(
		request.Context(), actor, chi.URLParam(request, "templateId"),
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(template.Revision))
	api.respond(writer, adminTemplateView(template), nil)
}

func (api *CanonicalAPI) createAdminTemplateDraft(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.CreateAdminTemplateDraftInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	templateID := chi.URLParam(request, "templateId")
	if templateID != input.TemplateId ||
		!validRevisionCommandHeaders(request, input.IdempotencyKey, input.ExpectedRevision) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	draft, err := api.application.CreateAdminTemplateDraft(
		request.Context(), actor, application.CreateAdminTemplateDraftCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			TemplateID: input.TemplateId, ExpectedRevision: *input.ExpectedRevision,
			ChangeReason: input.ChangeReason,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(draft.Revision))
	writeJSON(writer, http.StatusCreated, adminTemplateVersionView(draft))
}

func (api *CanonicalAPI) addAdminTemplateDraftQuestion(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.AddAdminTemplateDraftQuestionInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	templateID := chi.URLParam(request, "templateId")
	draftID := chi.URLParam(request, "draftVersionId")
	if templateID != input.TemplateId || draftID != input.DraftVersionId ||
		!validRevisionCommandHeaders(request, input.IdempotencyKey, input.ExpectedRevision) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	draft, err := api.application.AddAdminTemplateDraftQuestion(
		request.Context(), actor, application.AddAdminTemplateDraftQuestionCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			TemplateID: input.TemplateId, DraftVersionID: input.DraftVersionId,
			QuestionID: input.QuestionId, ExpectedRevision: *input.ExpectedRevision,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(draft.Revision))
	api.respond(writer, adminTemplateVersionView(draft), nil)
}

func (api *CanonicalAPI) moveAdminTemplateDraftQuestion(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.MoveAdminTemplateDraftQuestionInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	templateID := chi.URLParam(request, "templateId")
	draftID := chi.URLParam(request, "draftVersionId")
	questionID := chi.URLParam(request, "questionId")
	if templateID != input.TemplateId || draftID != input.DraftVersionId ||
		questionID != input.QuestionId ||
		!validRevisionCommandHeaders(request, input.IdempotencyKey, input.ExpectedRevision) {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	draft, err := api.application.MoveAdminTemplateDraftQuestion(
		request.Context(), actor, application.MoveAdminTemplateDraftQuestionCommand{
			OperationID: input.OperationId, IdempotencyKey: input.IdempotencyKey,
			TemplateID: input.TemplateId, DraftVersionID: input.DraftVersionId,
			QuestionID: input.QuestionId, Direction: input.Direction,
			ExpectedRevision: *input.ExpectedRevision,
		},
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", strongRevisionETag(draft.Revision))
	api.respond(writer, adminTemplateVersionView(draft), nil)
}

func (api *CanonicalAPI) getAdminInspectionPackage(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	record, err := api.adminWorkspace.GetInspectionPackage(
		request.Context(), actor, chi.URLParam(request, "packageId"),
	)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	etag, err := strongProjectionETag(record)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writer.Header().Set("ETag", etag)
	api.respond(writer, generated.AdminInspectionPackageView{
		Id: record.ID, AuditId: record.AuditID,
		OrganizationId: record.OrganizationID, OrganizationName: record.OrganizationName,
		QuestionIds:          append([]string{}, record.QuestionIDs...),
		ConfiguredReferences: append([]string{}, record.ConfiguredReferences...),
		ExpectedEvidence:     append([]string{}, record.ExpectedEvidence...),
		RiskFocus:            append([]string{}, record.RiskFocus...),
	}, nil)
}

func validOptionalRevisionCommandHeaders(
	request *http.Request,
	idempotencyKey string,
	expectedRevision *int64,
) bool {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" || request.Header.Get("Idempotency-Key") != idempotencyKey {
		return false
	}
	if expectedRevision == nil {
		return strings.TrimSpace(request.Header.Get("If-Match")) == ""
	}
	return request.Header.Get("If-Match") == strongRevisionETag(*expectedRevision)
}

func adminQuestionView(record configuration.AdminQuestion) generated.AdminQuestionView {
	return generated.AdminQuestionView{
		Id: record.ID, Prompt: record.Prompt,
		ConfiguredReference: record.ConfiguredReference,
		ExpectedEvidence:    record.ExpectedEvidence, Revision: record.Revision,
	}
}

func adminRegulatoryMappingViews(
	records []configuration.AdminRegulatoryMapping,
) []generated.AdminRegulatoryMappingView {
	items := make([]generated.AdminRegulatoryMappingView, 0, len(records))
	for _, record := range records {
		sources := make([]generated.AdminRegulatorySourceView, 0, len(record.Sources))
		for _, source := range record.Sources {
			sources = append(sources, generated.AdminRegulatorySourceView{
				Id: source.ID, Title: source.Title, SourceType: source.SourceType,
				Version: source.Version, Status: source.Status,
				Locator: source.Locator, Url: source.URL,
			})
		}
		questions := make(
			[]generated.AdminProposedInspectionQuestionView,
			0,
			len(record.ProposedQuestions),
		)
		for _, question := range record.ProposedQuestions {
			questions = append(questions, generated.AdminProposedInspectionQuestionView{
				Id: question.ID, Prompt: question.Prompt,
				VerificationMethod: question.VerificationMethod,
				EvidenceExamples:   append([]string(nil), question.EvidenceExamples...),
				WhyIncluded:        question.WhyIncluded,
			})
		}
		scopeQuestions := make(
			[]generated.AdminChecklistQuestionScopeRecommendationView,
			0,
			len(record.ScopeRecommendation.QuestionRecommendations),
		)
		for _, recommendation := range record.ScopeRecommendation.QuestionRecommendations {
			scopeQuestions = append(
				scopeQuestions,
				generated.AdminChecklistQuestionScopeRecommendationView{
					QuestionId:              recommendation.QuestionID,
					Classification:          recommendation.Classification,
					Rationale:               recommendation.Rationale,
					HistoryBasis:            recommendation.HistoryBasis,
					RequiresManagerApproval: recommendation.RequiresManagerApproval,
				},
			)
		}
		items = append(items, generated.AdminRegulatoryMappingView{
			Id: record.ID, AuditArea: record.AuditArea,
			ServiceProviderTypes:       append([]string(nil), record.ServiceProviderTypes...),
			ApplicableRegulations:      append([]string(nil), record.ApplicableRegulations...),
			CriticalElement:            record.CriticalElement,
			ProtocolQuestionId:         record.ProtocolQuestionID,
			ProtocolQuestion:           record.ProtocolQuestion,
			AnnexReferences:            append([]string(nil), record.AnnexReferences...),
			NationalReferences:         append([]string(nil), record.NationalReferences...),
			CaaImplementationReference: record.CAAImplementationReference,
			Requirement:                record.Requirement,
			VerificationObjective:      record.VerificationObjective,
			ExpectedEvidence:           append([]string(nil), record.ExpectedEvidence...),
			WhyIncluded:                record.WhyIncluded,
			ReviewStatus:               record.ReviewStatus,
			SourceGap:                  record.SourceGap,
			RefreshPolicy: generated.AdminRegulatoryRefreshPolicyView{
				SourceCollectionId:             record.RefreshPolicy.SourceCollectionID,
				LastCheckedAt:                  record.RefreshPolicy.LastCheckedAt,
				NextReconciliationDate:         record.RefreshPolicy.NextReconciliationDate,
				NextExpertValidationDate:       record.RefreshPolicy.NextExpertValidationDate,
				EventDrivenReview:              record.RefreshPolicy.EventDrivenReview,
				ReconciliationIntervalMonths:   record.RefreshPolicy.ReconciliationIntervalMonths,
				ExpertValidationIntervalMonths: record.RefreshPolicy.ExpertValidationIntervalMonths,
				SourceChangeState:              record.RefreshPolicy.SourceChangeState,
				UpdateMode:                     record.RefreshPolicy.UpdateMode,
				DocumentCount:                  record.RefreshPolicy.DocumentCount,
				ManifestPath:                   record.RefreshPolicy.ManifestPath,
				Guardrails:                     append([]string(nil), record.RefreshPolicy.Guardrails...),
			},
			ScopeRecommendation: generated.AdminChecklistScopeRecommendationView{
				Id:                      record.ScopeRecommendation.ID,
				Status:                  record.ScopeRecommendation.Status,
				HistoryState:            record.ScopeRecommendation.HistoryState,
				GeneratedAt:             record.ScopeRecommendation.GeneratedAt,
				Signals:                 append([]string(nil), record.ScopeRecommendation.Signals...),
				Guardrails:              append([]string(nil), record.ScopeRecommendation.Guardrails...),
				QuestionRecommendations: scopeQuestions,
			},
			Sources:           sources,
			ProposedQuestions: questions,
		})
	}
	return items
}

func adminTemplateVersionView(
	record configuration.AdminTemplateVersion,
) generated.AdminTemplateVersionView {
	return generated.AdminTemplateVersionView{
		Id: record.ID, TemplateId: record.TemplateID, Version: record.Version,
		Status: record.Status, Owner: record.Owner, CreatorSubjectId: record.CreatorSubjectID,
		ChangeReason: record.ChangeReason, QuestionIds: append([]string(nil), record.QuestionIDs...),
		Revision: record.Revision, CreatedAt: record.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func adminTemplateView(record configuration.AdminTemplate) generated.AdminTemplateView {
	versions := make([]generated.AdminTemplateVersionView, 0, len(record.Versions))
	for _, version := range record.Versions {
		versions = append(versions, adminTemplateVersionView(version))
	}
	return generated.AdminTemplateView{
		Id: record.ID, PublishedVersionId: record.PublishedVersionID,
		Versions: versions, Revision: record.Revision,
	}
}

func planningView(item planning.Item) generated.PlanningItemView {
	view := generated.PlanningItemView{
		Id: item.ID, Title: item.Title, PlanYear: int64(item.PlanYear),
		OrganizationId: item.OrganizationID, OrganizationName: item.OrganizationName,
		InspectionType: item.InspectionType, ScheduledDate: item.ScheduledDate,
		EstimatedBudget: item.EstimatedBudget, Status: generated.PlanningStatus(item.Status),
		CurrentOwnerRole: generated.Role(item.CurrentOwnerRole), NextAction: item.NextAction,
		Revision: item.Revision,
	}
	if item.SubmittedScopeSnapshotID != "" {
		view.SubmittedScopeSnapshotId = &item.SubmittedScopeSnapshotID
	}
	if item.PlanningSnapshotDigest != "" {
		view.PlanningSnapshotDigest = &item.PlanningSnapshotDigest
	}
	return view
}

func boundedPageLimit(limit *int64) int32 {
	if limit == nil || *limit <= 0 || *limit > 100 {
		return 100
	}
	return int32(*limit)
}
