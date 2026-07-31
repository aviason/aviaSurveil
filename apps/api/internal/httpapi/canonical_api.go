package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/administration"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/assignments"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/assistant"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/caps"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/checklistgovernance"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/configuration"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/documents"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/evidence"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/findings"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi/generated"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/inspections"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/inspections/attachments"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/organizations"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/planning"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/idempotency"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/potentialfindings"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/regulatory"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/reports"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/risk"
	fieldsync "github.com/MarlonJD/aviaSurveil360/apps/api/internal/sync"
	"github.com/go-chi/chi/v5"
)

type CanonicalAPIDependencies struct {
	Pool               *database.Pool
	Application        *application.Service
	GrantService       *fieldsync.GrantService
	SyncOperations     *fieldsync.OperationService
	EvidenceUploads    *evidence.UploadService
	AttachmentUploads  *attachments.UploadService
	Planning           *planning.Service
	Profiles           *identity.ProfileService
	Assignments        *assignments.Service
	PackageDrafts      *inspections.PackageDraftService
	AdminWorkspace     *configuration.WorkspaceService
	Risk               *risk.Service
	Administration     *administration.ProjectionService
	DirectoryProvider  administration.AccessDirectoryProvider
	Users              *administration.UserService
	Assistant          *assistant.Service
	Communications     *application.CommunicationsWorkflow
	Documents          *documents.Service
	GovernedCandidates *regulatory.AdminService
	GovernedLifecycle  *checklistgovernance.Service
	Clock              func() time.Time
}

type CanonicalAPI struct {
	pool               *database.Pool
	application        *application.Service
	grants             *fieldsync.GrantService
	syncOperations     *fieldsync.OperationService
	evidenceUploads    *evidence.UploadService
	attachmentUploads  *attachments.UploadService
	planning           *planning.Service
	profiles           *identity.ProfileService
	assignments        *assignments.Service
	packageDrafts      *inspections.PackageDraftService
	adminWorkspace     *configuration.WorkspaceService
	risk               *risk.Service
	administration     *administration.ProjectionService
	users              *administration.UserService
	assistant          *assistant.Service
	communications     *application.CommunicationsWorkflow
	documents          *documents.Service
	governedCandidates *regulatory.AdminService
	governedLifecycle  *checklistgovernance.Service
	clock              func() time.Time
}

func NewCanonicalAPI(dependencies CanonicalAPIDependencies) *CanonicalAPI {
	clock := dependencies.Clock
	if clock == nil {
		clock = time.Now
	}
	syncOperations := dependencies.SyncOperations
	if syncOperations == nil {
		syncOperations = fieldsync.NewOperationService(dependencies.Pool, fieldsync.OperationDependencies{Clock: clock})
	}
	planningService := dependencies.Planning
	if planningService == nil {
		planningService = planning.NewService(dependencies.Pool, planning.Dependencies{Clock: clock})
	}
	profileService := dependencies.Profiles
	if profileService == nil && dependencies.Pool != nil {
		profileService = identity.NewProfileService(dependencies.Pool, identity.ProfileServiceDependencies{Clock: clock})
	}
	assignmentService := dependencies.Assignments
	if assignmentService == nil && dependencies.Pool != nil {
		assignmentService = assignments.NewService(dependencies.Pool, assignments.Dependencies{Clock: clock})
	}
	packageDraftService := dependencies.PackageDrafts
	if packageDraftService == nil && dependencies.Pool != nil {
		packageDraftService = inspections.NewPackageDraftService(
			dependencies.Pool,
			inspections.PackageDraftDependencies{Clock: clock},
		)
	}
	adminWorkspaceService := dependencies.AdminWorkspace
	if adminWorkspaceService == nil && dependencies.Pool != nil {
		adminWorkspaceService = configuration.NewWorkspaceService(dependencies.Pool)
	}
	riskService := dependencies.Risk
	if riskService == nil && dependencies.Pool != nil {
		riskService = risk.NewService(
			dependencies.Pool,
			risk.Dependencies{Clock: clock},
		)
	}
	administrationService := dependencies.Administration
	if administrationService == nil && dependencies.Pool != nil {
		administrationService = administration.NewProjectionService(
			dependencies.Pool,
			administration.ProjectionDependencies{
				Clock: clock, DirectoryProvider: dependencies.DirectoryProvider,
			},
		)
	}
	userService := dependencies.Users
	if userService == nil && dependencies.Pool != nil {
		userService = administration.NewUserService(
			dependencies.Pool,
			administration.UserServiceDependencies{Clock: clock},
		)
	}
	assistantService := dependencies.Assistant
	if assistantService == nil && dependencies.Pool != nil {
		assistantService = assistant.NewService(
			dependencies.Pool,
			assistant.Dependencies{
				Clock: clock, Provider: assistant.NewDeterministicProvider(),
			},
		)
	}
	communicationsWorkflow := dependencies.Communications
	if communicationsWorkflow == nil && dependencies.Pool != nil {
		communicationsWorkflow = application.NewCommunicationsWorkflow(
			dependencies.Pool,
			application.CommunicationsWorkflowDependencies{Clock: clock},
		)
	}
	governedCandidates := dependencies.GovernedCandidates
	if governedCandidates == nil && dependencies.Pool != nil {
		governedCandidates = regulatory.NewAdminService(dependencies.Pool, clock)
	}
	governedLifecycle := dependencies.GovernedLifecycle
	if governedLifecycle == nil && dependencies.Pool != nil {
		governedLifecycle = checklistgovernance.NewService(dependencies.Pool, clock)
	}
	return &CanonicalAPI{
		pool: dependencies.Pool, application: dependencies.Application, grants: dependencies.GrantService,
		syncOperations:  syncOperations,
		evidenceUploads: dependencies.EvidenceUploads, attachmentUploads: dependencies.AttachmentUploads,
		planning:           planningService,
		profiles:           profileService,
		assignments:        assignmentService,
		packageDrafts:      packageDraftService,
		adminWorkspace:     adminWorkspaceService,
		risk:               riskService,
		administration:     administrationService,
		users:              userService,
		assistant:          assistantService,
		communications:     communicationsWorkflow,
		documents:          dependencies.Documents,
		governedCandidates: governedCandidates,
		governedLifecycle:  governedLifecycle,
		clock:              clock,
	}
}

func (api *CanonicalAPI) Handler() http.Handler {
	router := chi.NewRouter()
	router.Get("/v1/assignments", api.listAssignments)
	router.Get("/v1/inspection-packages/{id}", api.getInspectionPackage)
	router.Post("/v1/inspection-packages/{id}/checkout", api.checkoutInspectionPackage)
	router.Put("/v1/checklist-responses/{responseId}", api.upsertChecklistResponse)
	router.Post("/v1/checklists/{auditId}/submit", api.submitChecklist)
	router.Post("/v1/checklists/{auditId}/reopen", api.reopenChecklist)
	router.Get("/v1/potential-findings", api.listPotentialFindings)
	router.Post("/v1/potential-findings", api.createPotentialFinding)
	router.Get("/v1/potential-findings/{potentialFindingId}", api.getPotentialFinding)
	router.Post("/v1/potential-findings/{id}/decisions", api.decidePotentialFinding)
	router.Get("/v1/findings", api.listFindings)
	router.Get("/v1/findings/{id}", api.getFinding)
	router.Get("/v1/findings/{id}/evidence", api.listEvidenceVersions)
	router.Get("/v1/findings/{findingId}/cap-revisions", api.listCapRevisions)
	router.Post("/v1/findings/{id}/authorized-closure", api.authorizedCloseFinding)
	router.Post("/v1/caps", api.submitCAP)
	router.Get("/v1/cap-revisions/{capRevisionId}", api.getCapRevision)
	router.Post("/v1/caps/{capRevisionId}/reviews", api.reviewCAP)
	router.Post("/v1/inspection-attachments/{id}/uploads", api.beginInspectionAttachmentUpload)
	router.Post("/v1/inspection-attachments/uploads/{uploadId}/complete", api.completeInspectionAttachmentUpload)
	router.Post("/v1/evidence/uploads", api.beginEvidenceUpload)
	router.Post("/v1/evidence/uploads/{uploadId}/complete", api.completeEvidenceUpload)
	router.Post("/v1/evidence/{evidenceVersionId}/reviews", api.reviewEvidence)
	router.Get("/v1/report-versions/{id}", api.getReportVersion)
	router.Post("/v1/report-versions", api.createReportVersion)
	router.Post("/v1/report-versions/{id}/decisions", api.decideReport)
	router.Get("/v1/documents", api.listDocuments)
	router.Get("/v1/documents/{documentId}", api.getDocument)
	router.Get("/v1/auditee/report-versions", api.listAuditeeReleasedReports)
	router.Get("/v1/auditee/report-versions/{reportVersionId}", api.getAuditeeReleasedReport)
	router.Get("/v1/dashboards/manager", api.getManagerDashboard)
	router.Get("/v1/organizations", api.listOrganizations)
	router.Get("/v1/profile", api.getMyProfile)
	router.Put("/v1/profile", api.updateMyProfile)
	router.Get("/v1/planning/items", api.listPlanningItems)
	router.Post("/v1/planning/items/{id}/decisions", api.decidePlanningItem)
	router.Post("/v1/planning/intake-drafts", api.createPlanningIntakeDraft)
	router.Post("/v1/audit-workspaces", api.createAuditWorkspace)
	router.Get("/v1/planning/intake-drafts/{draftId}", api.getPlanningIntakeDraft)
	router.Put("/v1/planning/intake-drafts/{draftId}", api.savePlanningIntakeDraft)
	router.Post("/v1/planning/intake-drafts/{draftId}/submissions", api.submitPlanningIntake)
	router.Get("/v1/inspection-package-drafts/{packageDraftId}", api.getInspectionPackageDraft)
	router.Put("/v1/inspection-package-drafts/{packageDraftId}", api.saveInspectionPackageDraft)
	router.Get("/v1/team-members", api.listTeamMembers)
	router.Get("/v1/team-members/{subjectId}", api.getTeamMember)
	router.Get("/v1/audit-teams", api.listAuditTeams)
	router.Get("/v1/audit-teams/{auditId}", api.getAuditTeam)
	router.Get("/v1/auditee/coordination", api.listAuditeeCoordination)
	router.Post("/v1/auditee/coordination/{auditId}/responses", api.respondAuditeeCoordination)
	router.Get("/v1/configuration/checklist-template-versions", api.listChecklistTemplateVersions)
	router.Get("/v1/configuration/checklist-template-versions/{templateVersionId}", api.getChecklistTemplateVersion)
	router.Get("/v1/configuration/reminder-rules", api.listReminderRules)
	router.Get("/v1/communications", api.listCommunications)
	router.Post("/v1/communications", api.sendCommunication)
	router.Get("/v1/calendar-items", api.listCalendarItems)
	router.Get("/v1/calendar-items/{calendarItemId}", api.getCalendarItem)
	router.Get("/v1/notifications", api.listNotifications)
	router.Post("/v1/notifications/{notificationId}/read", api.markNotificationRead)
	router.Get("/v1/audit-events", api.listAuditEvents)
	router.Get("/v1/risk/overview", api.getRiskOverview)
	router.Get("/v1/risk/management", api.getRiskManagementProjection)
	router.Get("/v1/administration/screens", api.listAdministrationScreenProjections)
	router.Get("/v1/administration/screens/{screenId}", api.getAdministrationScreenProjection)
	router.Post(
		"/v1/administration/screens/{screenId}/actions/{actionId}",
		api.invokeAdministrationVisibleAction,
	)
	router.Get("/v1/admin/regulatory-references", api.listAdminRegulatoryReferences)
	router.Get("/v1/admin/governed-checklist/sources", api.listAdminGovernedSources)
	router.Post("/v1/admin/governed-checklist/source-currentness-activations", api.activateAdminGovernedSourceCurrentness)
	router.Post("/v1/admin/governed-checklist/generation-runs", api.importAdminGovernedGenerationRun)
	router.Get("/v1/admin/governed-checklist/generation-runs/{generationRunId}", api.getAdminGovernedGenerationRun)
	router.Get("/v1/admin/governed-checklist/candidates/{candidateId}", api.getAdminGovernedCandidate)
	router.Post("/v1/admin/governed-checklist/candidates/{candidateId}/revisions", api.createAdminGovernedCandidateRevision)
	router.Post("/v1/admin/governed-checklist/candidates/{candidateId}/submissions", api.submitAdminGovernedCandidateReview)
	router.Post("/v1/department-manager/governed-checklist/blocked-generation-validations", api.validateDepartmentManagerBlockedGeneration)
	router.Get("/v1/department-manager/governed-checklist/review-queue", api.listDepartmentManagerGovernedReviewQueue)
	router.Get("/v1/department-manager/governed-checklist/candidates/{candidateId}", api.getDepartmentManagerGovernedCandidate)
	router.Post("/v1/department-manager/governed-checklist/candidates/{candidateId}/returns", api.returnDepartmentManagerGovernedCandidate)
	router.Post("/v1/department-manager/governed-checklist/candidates/{candidateId}/rejections", api.rejectDepartmentManagerGovernedCandidate)
	router.Post("/v1/department-manager/governed-checklist/candidates/{candidateId}/technical-approvals", api.approveDepartmentManagerGovernedCandidate)
	router.Post("/v1/department-manager/governed-checklist/candidates/{candidateId}/publications", api.publishDepartmentManagerGovernedCandidate)
	router.Get("/v1/department-manager/governed-checklist/published-versions/{templateVersionId}", api.getDepartmentManagerGovernedPublishedVersion)
	router.Get("/v1/admin/templates", api.listAdminTemplateMasters)
	router.Get("/v1/admin/questions", api.listAdminQuestions)
	router.Post("/v1/admin/questions", api.createAdminQuestion)
	router.Get("/v1/admin/templates/{templateId}", api.getAdminTemplate)
	router.Post("/v1/admin/templates/{templateId}/drafts", api.createAdminTemplateDraft)
	router.Post(
		"/v1/admin/templates/{templateId}/drafts/{draftVersionId}/questions",
		api.addAdminTemplateDraftQuestion,
	)
	router.Post(
		"/v1/admin/templates/{templateId}/drafts/{draftVersionId}/questions/{questionId}/moves",
		api.moveAdminTemplateDraftQuestion,
	)
	router.Get("/v1/admin/inspection-packages/{packageId}", api.getAdminInspectionPackage)
	router.Get("/v1/admin/report-definitions", api.listAdminReportDefinitions)
	router.Get("/v1/admin/access-directory", api.listAdminAccessDirectory)
	router.Post(
		"/v1/admin/user-lifecycle-requests",
		api.requestUserLifecycle,
	)
	router.Get(
		"/v1/admin/user-lifecycle-requests/{requestId}",
		api.getUserLifecycleRequest,
	)
	router.Get("/v1/admin/organizations", api.listAdminOrganizations)
	router.Post("/v1/admin/organizations", api.createAdminOrganization)
	router.Get("/v1/admin/organizations/{organizationId}", api.getAdminOrganization)
	router.Post("/v1/admin/reminder-rules", api.createReminderRule)
	router.Get("/v1/admin/audit-events", api.listAdminAuditEvents)
	router.Get("/v1/assistant/guidance", api.getAssistantGuidance)
	router.Post("/v1/assistant/drafts", api.createAssistantDraft)
	router.Post("/v1/sync/operations", api.pushFieldOperation)
	router.Get("/v1/sync/changes", api.pullSyncChanges)
	return router
}

func (api *CanonicalAPI) listAssignments(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.assignmentProjection(request.Context(), actor, optionalQuery(request, "status"), optionalIntQuery(request, "limit"))
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) getInspectionPackage(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.inspectionPackageProjection(request.Context(), actor, chi.URLParam(request, "id"))
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) checkoutInspectionPackage(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.CheckoutInspectionPackageInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if input.PackageId != chi.URLParam(request, "id") {
		api.respond(writer, nil, fmt.Errorf("%w: package path and body must match", application.ErrInvalid))
		return
	}
	grant, err := api.grants.Issue(request.Context(), actor, fieldsync.CheckoutInput{
		OperationID: input.OperationId, CorrelationID: input.OperationId, PackageID: input.PackageId,
		ExpectedPackageVersion: input.ExpectedPackageVersion, DeviceInstanceID: input.DeviceInstanceId,
	})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	packageView, err := api.inspectionPackageProjection(request.Context(), actor, input.PackageId)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	allowed := make([]generated.FieldCommandType, 0, len(grant.AllowedCommandTypes))
	for _, command := range grant.AllowedCommandTypes {
		allowed = append(allowed, generated.FieldCommandType(command))
	}
	api.respond(writer, generated.CheckoutInspectionPackageOutput{
		InspectionPackage: packageView,
		OfflineGrant: generated.OfflineGrant{
			GrantId: grant.ID, SubjectId: grant.SubjectID, OrganizationId: grant.OrganizationID,
			PackageId: grant.PackageID, PackageVersion: grant.PackageVersion, PackageDigest: grant.PackageDigest,
			AllowedCommandTypes: allowed, AssignmentScope: map[string]any{"questionIds": grant.QuestionIDs},
			DeviceInstanceId: grant.DeviceInstanceID, IssuedAt: grant.IssuedAt.UTC().Format(time.RFC3339Nano),
			ExpiresAt: grant.ExpiresAt.UTC().Format(time.RFC3339Nano), ProtocolVersion: int64(grant.ProtocolVersion),
		},
	}, nil)
}

func (api *CanonicalAPI) upsertChecklistResponse(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.UpsertChecklistResponseInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if input.ResponseId != chi.URLParam(request, "responseId") {
		api.respond(writer, nil, fmt.Errorf("%w: response path and body must match", application.ErrInvalid))
		return
	}
	var packageID string
	if err := api.pool.QueryRow(request.Context(), `SELECT id FROM inspection_packages WHERE inspection_id = $1 ORDER BY package_version DESC LIMIT 1`, input.AuditId).Scan(&packageID); err != nil {
		api.respond(writer, nil, application.ErrNotFound)
		return
	}
	_, err := api.application.UpsertChecklistResponse(request.Context(), actor, application.UpsertChecklistResponseCommand{
		OperationID: input.OperationId, CorrelationID: input.OperationId, ResponseID: input.ResponseId,
		InspectionID: input.AuditId, PackageID: packageID, QuestionID: input.QuestionId,
		ExpectedResponseRevision: input.ExpectedResponseRevision, Answer: string(input.Answer),
		CommentToAuditee: input.Comment,
	})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	output, err := api.checklistResponseProjection(request.Context(), input.ResponseId)
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) submitChecklist(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.SubmitChecklistInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if input.AuditId != chi.URLParam(request, "auditId") {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	result, err := api.application.SubmitChecklist(request.Context(), actor, application.SubmitChecklistCommand{
		OperationID: input.OperationId, CorrelationID: input.OperationId, InspectionID: input.AuditId,
		ExpectedChecklistRevision: input.ExpectedChecklistRevision,
	})
	api.respond(writer, generated.SubmitChecklistOutput{AuditId: result.InspectionID, ChecklistStatus: string(result.Status), ChecklistRevision: result.Revision}, err)
}

func (api *CanonicalAPI) reopenChecklist(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.ReopenChecklistInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	result, err := api.application.ReopenChecklist(request.Context(), actor, application.ReopenChecklistCommand{
		OperationID: input.OperationId, CorrelationID: input.OperationId, InspectionID: input.AuditId,
		ExpectedChecklistRevision: input.ExpectedChecklistRevision, Reason: input.Reason,
	})
	api.respond(writer, generated.SubmitChecklistOutput{AuditId: result.InspectionID, ChecklistStatus: string(result.Status), ChecklistRevision: result.Revision}, err)
}

func (api *CanonicalAPI) listPotentialFindings(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.potentialFindingsProjection(request.Context(), actor, optionalQuery(request, "status"), optionalIntQuery(request, "limit"))
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) getPotentialFinding(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.authorizedPotentialFindingProjection(request.Context(), actor, chi.URLParam(request, "potentialFindingId"))
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) createPotentialFinding(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.CreatePotentialFindingInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	result, err := api.application.CreatePotentialFinding(request.Context(), actor, application.CreatePotentialFindingCommand{
		OperationID: input.OperationId, CorrelationID: input.OperationId, InspectionID: input.AuditId,
		QuestionID: input.QuestionId, ChecklistResponseID: input.ChecklistResponseId,
		ExpectedChecklistResponseRevision: input.ExpectedChecklistResponseRevision, Title: input.Title,
		Description: input.Description, CommentToAuditee: input.RequiredComment,
		ExpectedEvidence:        "PBE serviceability record and cabin position confirmation",
		InspectionAttachmentIDs: input.InspectionAttachmentIds,
	})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	output, err := api.potentialFindingProjection(request.Context(), result.ID)
	api.respondCreated(writer, output, err)
}

func (api *CanonicalAPI) decidePotentialFinding(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var raw json.RawMessage
	if !decodeJSON(writer, request, &raw) {
		return
	}
	var discriminator struct {
		OperationID        string `json:"operationId"`
		PotentialFindingID string `json:"potentialFindingId"`
		Decision           string `json:"decision"`
	}
	if err := json.Unmarshal(raw, &discriminator); err != nil {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	if discriminator.PotentialFindingID != chi.URLParam(request, "id") {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	if discriminator.Decision == "CONVERT" {
		var input generated.ConvertPotentialFindingInput
		if err := json.Unmarshal(raw, &input); err != nil {
			api.respond(writer, nil, application.ErrInvalid)
			return
		}
		var dueDate *time.Time
		if input.DueDate != nil {
			parsed, err := time.Parse("2006-01-02", *input.DueDate)
			if err != nil {
				api.respond(writer, nil, fmt.Errorf("%w: Due Date must use YYYY-MM-DD", application.ErrInvalid))
				return
			}
			dueDate = &parsed
		}
		result, err := api.application.ConvertPotentialFinding(request.Context(), actor, application.ConvertPotentialFindingCommand{
			OperationID: input.OperationId, CorrelationID: input.OperationId, PotentialFindingID: input.PotentialFindingId,
			ExpectedRevision: input.ExpectedPotentialFindingRevision, Severity: potentialfindings.Severity(input.Severity),
			CAPRequired: input.CapRequired, EvidenceRequired: input.EvidenceRequired, DueDate: dueDate,
			RequirementsSpecified: true,
		})
		if err != nil {
			api.respond(writer, nil, err)
			return
		}
		potential, err := api.potentialFindingProjection(request.Context(), result.PotentialFindingID)
		if err != nil {
			api.respond(writer, nil, err)
			return
		}
		finding, err := api.findingProjection(request.Context(), actor, result.FindingID)
		api.respond(writer, generated.PotentialFindingDecisionOutput{PotentialFinding: potential, Finding: &finding}, err)
		return
	}
	var input generated.ReturnOrDismissPotentialFindingInput
	if err := json.Unmarshal(raw, &input); err != nil {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	result, err := api.application.DecidePotentialFinding(request.Context(), actor, application.DecidePotentialFindingCommand{
		OperationID: input.OperationId, CorrelationID: input.OperationId, PotentialFindingID: input.PotentialFindingId,
		ExpectedRevision: input.ExpectedPotentialFindingRevision, Decision: potentialfindings.Decision(input.Decision), Reason: input.Reason,
	})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	potential, err := api.potentialFindingProjection(request.Context(), result.ID)
	api.respond(writer, generated.PotentialFindingDecisionOutput{PotentialFinding: potential}, err)
}

func (api *CanonicalAPI) listFindings(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.findingsProjection(request.Context(), actor, optionalQuery(request, "status"), optionalIntQuery(request, "limit"))
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) getFinding(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.findingProjection(request.Context(), actor, chi.URLParam(request, "id"))
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) authorizedCloseFinding(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.AuthorizedCloseInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if input.FindingId != chi.URLParam(request, "id") {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	_, err := api.application.AuthorizedCloseFinding(request.Context(), actor, application.AuthorizedCloseFindingCommand{
		OperationID: input.OperationId, CorrelationID: input.OperationId, FindingID: input.FindingId,
		ExpectedFindingRevision: input.ExpectedFindingRevision, Reason: input.Reason,
	})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	output, err := api.findingProjection(request.Context(), actor, input.FindingId)
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) listCapRevisions(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.capRevisionsProjection(request.Context(), actor, chi.URLParam(request, "findingId"))
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) getCapRevision(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.capRevisionByIDProjection(request.Context(), actor, chi.URLParam(request, "capRevisionId"))
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) submitCAP(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.SubmitCapInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	target, err := time.Parse("2006-01-02", input.TargetCompletionDate)
	if err != nil {
		api.respond(writer, nil, fmt.Errorf("%w: Target completion date must use YYYY-MM-DD", application.ErrInvalid))
		return
	}
	result, err := api.application.SubmitCAP(request.Context(), actor, application.SubmitCAPCommand{
		OperationID: input.OperationId, CorrelationID: input.OperationId, FindingID: input.FindingId,
		ExpectedFindingRevision: input.ExpectedFindingRevision, RootCause: input.RootCause,
		CorrectiveAction: input.CorrectiveAction, PreventiveAction: input.PreventiveAction,
		ResponsiblePerson: input.ResponsiblePerson, TargetCompletionDate: target, CommentToCAA: input.CommentToCaa,
	})
	api.respondCreated(writer, generated.SubmitCapOutput{
		CapRevisionId: result.CAPRevisionID, CapRevision: result.CAPRevision, CapStatus: string(result.CAPStatus),
		FindingStatus: generated.FindingStatus(result.FindingStatus), FindingRevision: result.FindingRevision,
	}, err)
}

func (api *CanonicalAPI) reviewCAP(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.ReviewCapInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if input.CapRevisionId != chi.URLParam(request, "capRevisionId") {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	result, err := api.application.ReviewCAP(request.Context(), actor, application.ReviewCAPCommand{
		OperationID: input.OperationId, CorrelationID: input.OperationId, CAPRevisionID: input.CapRevisionId,
		ExpectedCAPRevision: input.ExpectedCapRevision, FindingID: input.FindingId,
		ExpectedFindingRevision: input.ExpectedFindingRevision, Decision: caps.ReviewDecision(input.Decision),
		CommentToAuditee: input.CommentToAuditee, InternalCAANote: input.InternalCaaNote,
	})
	api.respond(writer, generated.ReviewCapOutput{
		CapRevisionId: result.CAPRevisionID, CapRevision: input.ExpectedCapRevision,
		CapStatus: generated.CapStatus(result.CAPStatus), FindingStatus: generated.FindingStatus(result.FindingStatus),
		FindingRevision: result.FindingRevision,
	}, err)
}

func (api *CanonicalAPI) beginInspectionAttachmentUpload(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.BeginInspectionAttachmentUploadInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	result, err := api.attachmentUploads.Begin(request.Context(), actor, attachments.BeginUploadInput{
		OperationID: input.OperationId, CorrelationID: input.OperationId, InspectionAttachmentID: input.InspectionAttachmentId,
		PackageID: input.PackageId, ByteSize: input.ByteSize, SHA256: input.Sha256,
		FileName: input.FileName, DeclaredMediaType: input.DeclaredMediaType,
	})
	api.respondCreated(writer, generated.BeginInspectionAttachmentUploadOutput{
		UploadId: result.UploadID, StagingObjectKey: result.StagingObjectKey, UploadUrl: result.UploadURL,
		RequiredHeaders: generated.UploadRequiredHeaders{
			ContentType:    result.RequiredHeaders.ContentType,
			XAmzMetaSha256: result.RequiredHeaders.SHA256,
			IfNoneMatch:    result.RequiredHeaders.IfNoneMatch,
		},
		ExpiresAt: result.ExpiresAt.UTC().Format(time.RFC3339Nano), MaximumByteSize: result.MaximumByteSize,
	}, err)
}

func (api *CanonicalAPI) completeInspectionAttachmentUpload(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.CompleteInspectionAttachmentUploadInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	result, err := api.attachmentUploads.Complete(request.Context(), actor, attachments.CompleteUploadInput{
		OperationID: input.OperationId, CorrelationID: input.OperationId, UploadID: input.UploadId,
		SHA256: input.Sha256, ByteSize: input.ByteSize,
	})
	api.respond(writer, generated.CompleteInspectionAttachmentUploadOutput{
		InspectionAttachmentId: result.InspectionAttachmentID, UploadState: result.UploadState, ScanState: result.ScanState,
	}, err)
}

func (api *CanonicalAPI) beginEvidenceUpload(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.BeginEvidenceUploadInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	result, err := api.evidenceUploads.Begin(request.Context(), actor, evidence.BeginUploadInput{
		OperationID: input.OperationId, CorrelationID: input.OperationId, FindingID: input.FindingId,
		ExpectedFindingRevision: input.ExpectedFindingRevision, FileName: input.FileName,
		DeclaredMediaType: input.DeclaredMediaType, ByteSize: input.ByteSize, SHA256: input.Sha256,
	})
	api.respondCreated(writer, generated.BeginEvidenceUploadOutput{
		UploadId: result.UploadID, StagingObjectKey: result.StagingObjectKey, UploadUrl: result.UploadURL,
		RequiredHeaders: generated.UploadRequiredHeaders{
			ContentType:    result.RequiredHeaders.ContentType,
			XAmzMetaSha256: result.RequiredHeaders.SHA256,
			IfNoneMatch:    result.RequiredHeaders.IfNoneMatch,
		},
		ExpiresAt: result.ExpiresAt.UTC().Format(time.RFC3339Nano), MaximumByteSize: result.MaximumByteSize,
	}, err)
}

func (api *CanonicalAPI) completeEvidenceUpload(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.CompleteEvidenceUploadInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	result, err := api.evidenceUploads.Complete(request.Context(), actor, evidence.CompleteUploadInput{
		OperationID: input.OperationId, CorrelationID: input.OperationId, UploadID: input.UploadId,
		SHA256: input.Sha256, ByteSize: input.ByteSize,
	})
	api.respond(writer, generated.CompleteEvidenceUploadOutput{
		EvidenceVersionId: result.EvidenceVersionID, Version: result.Version, UploadState: result.UploadState,
		ScanState: result.ScanState, ReviewState: generated.EvidenceReviewState(result.ReviewState),
	}, err)
}

func (api *CanonicalAPI) listEvidenceVersions(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	versions, err := api.evidenceUploads.ListVersions(request.Context(), actor, chi.URLParam(request, "id"))
	items := make([]generated.EvidenceVersionView, 0, len(versions))
	for _, version := range versions {
		items = append(items, generated.EvidenceVersionView{
			Id: version.ID, FindingId: version.FindingID, OrganizationId: version.OrganizationID,
			Version: version.Version, FileName: version.FileName, SubmittedAt: version.SubmittedAt.UTC().Format(time.RFC3339Nano),
			UploadState: generated.EvidenceUploadState(version.UploadState), ScanState: generated.EvidenceScanState(version.ScanState),
			ReviewState: generated.EvidenceReviewState(version.ReviewState), Revision: version.Revision,
		})
	}
	api.respond(writer, generated.ListEvidenceVersionsOutput{Items: items}, err)
}

func (api *CanonicalAPI) reviewEvidence(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.ReviewEvidenceInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if input.EvidenceVersionId != chi.URLParam(request, "evidenceVersionId") {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	result, err := api.application.ReviewEvidence(request.Context(), actor, application.ReviewEvidenceCommand{
		OperationID: input.OperationId, CorrelationID: input.OperationId, EvidenceVersionID: input.EvidenceVersionId,
		ExpectedEvidenceVersionRevision: input.ExpectedEvidenceVersionRevision, FindingID: input.FindingId,
		ExpectedFindingRevision: input.ExpectedFindingRevision, Decision: evidence.Decision(input.Decision),
		CommentToAuditee: input.CommentToAuditee, InternalCAANote: input.InternalCaaNote,
	})
	api.respond(writer, generated.ReviewEvidenceOutput{
		ReviewDecisionId: result.ReviewDecisionID, EvidenceVersionId: result.EvidenceVersionID,
		EvidenceVersionRevision: result.EvidenceRevision, FindingStatus: generated.FindingStatus(result.FindingStatus),
		FindingRevision: result.FindingRevision,
	}, err)
}

func (api *CanonicalAPI) getReportVersion(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.reportProjection(request.Context(), actor, chi.URLParam(request, "id"))
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) decideReport(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	var input generated.DecideReportInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	if input.ReportVersionId != chi.URLParam(request, "id") {
		api.respond(writer, nil, application.ErrInvalid)
		return
	}
	decision := reports.Decision(input.Decision)
	if input.Decision == "ISSUE_AND_LOCK" {
		decision = reports.DecisionIssue
	}
	_, err := api.application.DecideReport(request.Context(), actor, application.DecideReportCommand{
		OperationID: input.OperationId, CorrelationID: input.OperationId, ReportVersionID: input.ReportVersionId,
		ExpectedRevision: input.ExpectedReportVersionRevision, Decision: decision, Reason: input.Reason,
	})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	output, err := api.reportProjection(request.Context(), actor, input.ReportVersionId)
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) listDocuments(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.documentsProjection(
		request.Context(), actor, strings.TrimSpace(request.URL.Query().Get("organizationId")),
	)
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) getDocument(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.documentProjection(
		request.Context(), actor, chi.URLParam(request, "documentId"), true,
	)
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) listAuditeeReleasedReports(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.auditeeReleasedReportsProjection(
		request.Context(), actor, strings.TrimSpace(request.URL.Query().Get("kind")),
	)
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) getAuditeeReleasedReport(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.auditeeReleasedReportProjection(
		request.Context(), actor, chi.URLParam(request, "reportVersionId"),
	)
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) getManagerDashboard(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	output, err := api.managerProjection(request.Context(), actor)
	api.respond(writer, output, err)
}

func (api *CanonicalAPI) pushFieldOperation(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	if !actor.HasRole(identity.RoleInspector) {
		api.respond(writer, nil, fieldsync.ErrGrantScope)
		return
	}
	var envelope generated.PushFieldOperationRequest
	if !decodeJSON(writer, request, &envelope) {
		return
	}
	result, err := api.syncOperations.Push(request.Context(), actor, envelope.Operation)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	var output generated.PushFieldOperationResult
	if err := json.Unmarshal(encoded, &output); err != nil {
		api.respond(writer, nil, err)
		return
	}
	api.respond(writer, output, nil)
}

func (api *CanonicalAPI) pullSyncChanges(writer http.ResponseWriter, request *http.Request) {
	actor, ok := requirePrincipal(writer, request)
	if !ok {
		return
	}
	cursor := optionalQuery(request, "cursor")
	packageID := optionalQuery(request, "packageId")
	grantID := optionalQuery(request, "offlineGrantId")
	limit := optionalIntQuery(request, "limit")
	if packageID == nil || grantID == nil {
		api.respond(writer, nil, fmt.Errorf("%w: packageId and offlineGrantId are required", application.ErrInvalid))
		return
	}
	resolvedLimit := 0
	if limit != nil {
		resolvedLimit = int(*limit)
	}
	result, err := api.syncOperations.Pull(request.Context(), actor, fieldsync.PullInput{
		PackageID: *packageID, OfflineGrantID: *grantID, Cursor: cursor, Limit: resolvedLimit,
	})
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	var output generated.SyncPullResponse
	if err := json.Unmarshal(encoded, &output); err != nil {
		api.respond(writer, nil, err)
		return
	}
	api.respond(writer, output, nil)
}

func (api *CanonicalAPI) respond(writer http.ResponseWriter, output any, err error) {
	if err == nil {
		writeJSON(writer, http.StatusOK, output)
		return
	}
	var governedValidation *regulatory.ValidationError
	if errors.As(err, &governedValidation) {
		issues := make([]generated.GovernedValidationIssue, 0, len(governedValidation.Issues))
		for _, issue := range governedValidation.Issues {
			sourceIdentity, sourceHash, clauseID, locator := issue.SourceIdentity, issue.SourceHash, issue.ClauseID, issue.Locator
			issues = append(issues, generated.GovernedValidationIssue{
				FieldPath: issue.FieldPath, Code: issue.Code, Message: issue.Message,
				SourceIdentity: &sourceIdentity, SourceHash: &sourceHash, ClauseId: &clauseID, Locator: &locator,
			})
		}
		detail, code := governedValidation.Error(), "INVALID_GOVERNED_CANDIDATE"
		writer.Header().Set("Content-Type", "application/problem+json")
		writer.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(writer).Encode(generated.GovernedValidationProblem{
			Type: "about:blank", Title: "Governed candidate validation failed",
			Status: http.StatusUnprocessableEntity, Detail: &detail, Code: &code, Issues: issues,
		})
		return
	}
	status := http.StatusInternalServerError
	code := "INTERNAL_ERROR"
	switch {
	case errors.Is(err, identity.ErrKeycloakUnavailable):
		status, code = http.StatusServiceUnavailable, "PROVIDER_UNAVAILABLE"
	case errors.Is(err, application.ErrForbidden), errors.Is(err, evidence.ErrEvidenceForbidden),
		errors.Is(err, organizations.ErrForbidden),
		errors.Is(err, risk.ErrForbidden), errors.Is(err, administration.ErrForbidden),
		errors.Is(err, assistant.ErrForbidden),
		errors.Is(err, configuration.ErrWorkspaceForbidden),
		errors.Is(err, assignments.ErrForbidden), errors.Is(err, inspections.ErrPackageDraftForbidden),
		errors.Is(err, attachments.ErrAttachmentForbidden), errors.Is(err, fieldsync.ErrGrantScope),
		errors.Is(err, fieldsync.ErrGrantExpired), errors.Is(err, fieldsync.ErrGrantRevoked),
		errors.Is(err, fieldsync.ErrAssignmentChanged), errors.Is(err, fieldsync.ErrPackageRevoked),
		errors.Is(err, fieldsync.ErrSessionRevoked), errors.Is(err, fieldsync.ErrCursorScope):
		status, code = http.StatusForbidden, "FORBIDDEN"
	case errors.Is(err, application.ErrNotFound), errors.Is(err, identity.ErrProfileNotFound),
		errors.Is(err, organizations.ErrNotFound), errors.Is(err, assignments.ErrNotFound),
		errors.Is(err, risk.ErrNotFound), errors.Is(err, administration.ErrNotFound),
		errors.Is(err, assistant.ErrNotFound),
		errors.Is(err, configuration.ErrWorkspaceNotFound),
		errors.Is(err, inspections.ErrPackageDraftNotFound):
		status, code = http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, administration.ErrMembershipRevisionConflict):
		status, code = http.StatusConflict, "MEMBERSHIP_REVISION_CONFLICT"
	case errors.Is(err, application.ErrConflict), errors.Is(err, identity.ErrConflict),
		errors.Is(err, assignments.ErrConflict), errors.Is(err, inspections.ErrPackageDraftConflict),
		errors.Is(err, administration.ErrConflict),
		errors.Is(err, idempotency.ErrOperationIDReuse):
		status, code = http.StatusConflict, "CONFLICT"
	case errors.Is(err, identity.ErrPrecondition):
		status, code = http.StatusPreconditionFailed, "PRECONDITION_FAILED"
	case errors.Is(err, application.ErrInvalid), errors.Is(err, evidence.ErrInvalidUpload),
		errors.Is(err, identity.ErrInvalidProfile),
		errors.Is(err, risk.ErrInvalid), errors.Is(err, assistant.ErrInvalid),
		errors.Is(err, administration.ErrInvalid),
		errors.Is(err, configuration.ErrWorkspaceInvalid),
		errors.Is(err, assignments.ErrInvalid), errors.Is(err, inspections.ErrPackageDraftInvalid),
		errors.Is(err, attachments.ErrInvalidUpload), errors.Is(err, evidence.ErrObjectMismatch),
		errors.Is(err, attachments.ErrObjectMismatch):
		status, code = http.StatusUnprocessableEntity, "INVALID_COMMAND"
	}
	if status == http.StatusInternalServerError {
		writeProblem(writer, status, "Internal server error", "the request could not be completed", code)
		return
	}
	title := publicErrorTitle(err)
	writeProblem(writer, status, title, title, code)
}

func (api *CanonicalAPI) respondCreated(writer http.ResponseWriter, output any, err error) {
	if err != nil {
		api.respond(writer, nil, err)
		return
	}
	writeJSON(writer, http.StatusCreated, output)
}

func publicErrorTitle(err error) string {
	message := strings.TrimSpace(err.Error())
	for {
		original := message
		for _, prefix := range []string{
			"forbidden:", "not found:", "conflict:", "invalid:",
			"invalid evidence upload:", "invalid inspection attachment upload:",
			"invalid upload:", "object mismatch:",
		} {
			if strings.HasPrefix(strings.ToLower(message), prefix) {
				message = strings.TrimSpace(message[len(prefix):])
				break
			}
		}
		if message == original {
			break
		}
	}
	if message == "" {
		return "Request could not be completed."
	}
	runes := []rune(message)
	runes[0] = unicode.ToUpper(runes[0])
	message = string(runes)
	if !strings.ContainsAny(message[len(message)-1:], ".?!") {
		message += "."
	}
	return message
}

func requirePrincipal(writer http.ResponseWriter, request *http.Request) (identity.Principal, bool) {
	principal, ok := PrincipalFromContext(request.Context())
	if !ok {
		writeProblem(writer, http.StatusUnauthorized, "Authentication required", "no server principal is attached", "UNAUTHENTICATED")
	}
	return principal, ok
}

func decodeJSON(writer http.ResponseWriter, request *http.Request, destination any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeProblem(writer, http.StatusBadRequest, "Invalid JSON request", err.Error(), "INVALID_JSON")
		return false
	}
	return true
}

func optionalQuery(request *http.Request, name string) *string {
	value := strings.TrimSpace(request.URL.Query().Get(name))
	if value == "" {
		return nil
	}
	return &value
}

func optionalIntQuery(request *http.Request, name string) *int64 {
	value := strings.TrimSpace(request.URL.Query().Get(name))
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

var _ = findings.StatusClosed
