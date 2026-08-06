package agademoworkspace

import (
	"context"
	"strings"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

func (service *Service) lifecycleQuery(ctx context.Context, principal identity.Principal, request QueryRequest, decision AuthorizationDecision) (QueryResponse, error) {
	if request.OperationID == OperationGetCurrentInspection || request.OperationID == OperationGetInspectionQuestionPage {
		if strings.TrimSpace(request.InspectionID) == "" {
			current, currentErr := service.currentInspectionID(ctx)
			if currentErr != nil {
				return QueryResponse{}, ErrNeutralDenied
			}
			request.InspectionID = current
		}
	} else if strings.TrimSpace(request.InspectionID) == "" {
		return QueryResponse{}, ErrNeutralDenied
	}
	if request.OperationID == OperationGetFinding && strings.TrimSpace(request.FindingID) == "" {
		return QueryResponse{}, ErrNeutralDenied
	}
	if request.OperationID == OperationGetCAPEvidence && strings.TrimSpace(request.EvidenceID) == "" && strings.TrimSpace(request.CapID) == "" {
		return QueryResponse{}, ErrNeutralDenied
	}
	if service == nil {
		return QueryResponse{}, ErrWorkspaceStore
	}
	store, ok := service.reader.(preprod.LifecycleStore)
	if !ok {
		store, ok = service.command.(preprod.LifecycleStore)
	}
	if !ok {
		return QueryResponse{}, ErrCapabilityUnavailable
	}
	snapshotStore := service.reader
	if snapshotStore == nil {
		snapshotStore = service.command
	}
	if snapshotStore == nil {
		return QueryResponse{}, ErrWorkspaceStore
	}
	workspace, err := snapshotStore.Snapshot(ctx)
	if err != nil {
		return QueryResponse{}, ErrNeutralDenied
	}
	aggregate, _, err := loadLifecycleFromStore(ctx, store, workspace.Generation.GenerationID, request.InspectionID)
	if err != nil || !lifecycleVisibleToPrincipal(aggregate, principal) || (!decision.IsAdmin && !bindingMatchesLifecycleObject(decision.Binding, aggregate, principal)) {
		return QueryResponse{}, ErrNeutralDenied
	}
	// Selector values are object references, not presentation hints. Validate
	// them after the aggregate is loaded and access-scoped, but before any role
	// projection can return the aggregate's surrounding lifecycle context.
	if err := validateLifecycleQuerySelectors(aggregate, request); err != nil {
		return QueryResponse{}, err
	}
	response := QueryResponse{Operation: request.OperationID, Generation: workspace.Generation, LifecycleAvailable: true}
	if principal.HasRole(identity.RoleAuditee) {
		projection, projectionErr := ProjectAuditeeLifecycle(aggregate, principal)
		if projectionErr != nil {
			return QueryResponse{}, projectionErr
		}
		response.LifecycleAuditee = &projection
		return response, nil
	}
	projection := ProjectLifecycle(aggregate, principal)
	response.Lifecycle = &projection
	if request.OperationID == OperationGetRoleHistory {
		caa := ProjectCAALifecycle(aggregate, principal)
		response.LifecycleCAA = &caa
	}
	if request.OperationID == OperationGetCurrentInspection {
		response.CurrentInspection = response.Lifecycle
	}
	if request.OperationID == OperationGetInspectionQuestionPage {
		if !service.canReceiveLifecycleQuestionText(ctx, principal, aggregate) {
			return QueryResponse{}, ErrNeutralDenied
		}
		page, pageErr := service.lifecycleQuestionPage(ctx, aggregate, request.Page, request.PageSize)
		if pageErr != nil {
			return QueryResponse{}, pageErr
		}
		response.QuestionPage = &page
	}
	return response, nil
}

func validateLifecycleQuerySelectors(aggregate LifecycleAggregate, request QueryRequest) error {
	findingID := strings.TrimSpace(request.FindingID)
	capID := strings.TrimSpace(request.CapID)
	evidenceID := strings.TrimSpace(request.EvidenceID)

	switch request.OperationID {
	case OperationGetFinding:
		if findingID == "" || capID != "" || evidenceID != "" {
			return ErrNeutralDenied
		}
		if _, found := aggregate.latestFinding(findingID); !found {
			return ErrNeutralDenied
		}
	case OperationGetCAPEvidence:
		if capID == "" && evidenceID == "" {
			return ErrNeutralDenied
		}
		selectedFindingID := ""
		if capID != "" {
			value, found := lifecycleCAPFindingID(aggregate, capID)
			if !found {
				return ErrNeutralDenied
			}
			selectedFindingID = value
		}
		if evidenceID != "" {
			value, found := lifecycleEvidenceFindingID(aggregate, evidenceID)
			if !found || selectedFindingID != "" && selectedFindingID != value {
				return ErrNeutralDenied
			}
			selectedFindingID = value
		}
		if _, found := aggregate.latestFinding(selectedFindingID); !found {
			return ErrNeutralDenied
		}
		if findingID != "" {
			if _, found := aggregate.latestFinding(findingID); !found || findingID != selectedFindingID {
				return ErrNeutralDenied
			}
		}
	default:
		if findingID != "" || capID != "" || evidenceID != "" {
			return ErrNeutralDenied
		}
	}
	return nil
}

func lifecycleCAPFindingID(aggregate LifecycleAggregate, capID string) (string, bool) {
	findingID := ""
	for _, revision := range aggregate.CAPRevisions {
		if revision.CAPID != capID {
			continue
		}
		if findingID != "" && findingID != revision.FindingID {
			return "", false
		}
		findingID = revision.FindingID
	}
	return findingID, findingID != ""
}

func lifecycleEvidenceFindingID(aggregate LifecycleAggregate, evidenceID string) (string, bool) {
	findingID := ""
	for _, version := range aggregate.EvidenceVersions {
		if version.EvidenceID != evidenceID {
			continue
		}
		if findingID != "" && findingID != version.FindingID {
			return "", false
		}
		findingID = version.FindingID
	}
	return findingID, findingID != ""
}

func (service *Service) currentInspectionID(ctx context.Context) (string, error) {
	store, ok := service.reader.(CurrentLifecycleStore)
	if !ok {
		store, ok = service.command.(CurrentLifecycleStore)
	}
	if !ok {
		return "", ErrLifecycleNotFound
	}
	snapshotStore := service.reader
	if snapshotStore == nil {
		snapshotStore = service.command
	}
	if snapshotStore == nil {
		return "", ErrWorkspaceStore
	}
	workspace, err := snapshotStore.Snapshot(ctx)
	if err != nil {
		return "", err
	}
	streams, err := store.ListLifecycleStreams(ctx, workspace.Generation.GenerationID)
	if err != nil || len(streams) != 1 {
		if len(streams) > 1 {
			return "", ErrCurrentObjectAmbiguous
		}
		return "", ErrLifecycleNotFound
	}
	return streams[0].LifecycleID, nil
}

func (service *Service) canReceiveLifecycleQuestionText(ctx context.Context, principal identity.Principal, aggregate LifecycleAggregate) bool {
	if principal.HasRole(identity.RoleInspector) && principal.SubjectID == aggregate.Inspector.SubjectID {
		return true
	}
	if principal.HasRole(identity.RoleLeadInspector) && principal.SubjectID == aggregate.Lead.SubjectID {
		return true
	}
	if principal.HasRole(identity.RoleDepartmentManager) {
		binding, found, err := service.ResolveBindingForOperation(ctx, principal, OperationGetInspectionQuestionPage)
		return err == nil && found && bindingMatchesLifecycleObject(binding, aggregate, principal) && bindingHasWorkspaceRole(binding, principal, "MANAGER")
	}
	return false
}

func (service *Service) lifecycleQuestionPage(ctx context.Context, aggregate LifecycleAggregate, page, pageSize int) (QuestionTextPage, error) {
	if page < 0 {
		return QuestionTextPage{}, ErrMalformedCommand
	}
	if pageSize == 0 {
		pageSize = MaxQuestionTextPage
	}
	if pageSize > MaxQuestionTextPage {
		return QuestionTextPage{}, ErrMalformedCommand
	}
	if page > MaxWorkspacePage {
		return QuestionTextPage{}, ErrMalformedCommand
	}
	start, end := boundedPageWindow(page, pageSize, len(aggregate.Questions))
	questions := aggregate.Questions[start:end]
	byKey, err := service.resolveLifecycleQuestionText(ctx, aggregate.GenerationID, questions)
	if err != nil {
		return QuestionTextPage{}, err
	}
	items := make([]LifecycleQuestionPageItem, 0, len(questions))
	for _, question := range questions {
		body, found := byKey[question.QuestionRef.Key()]
		if !found {
			return QuestionTextPage{}, ErrQuestionBodyIncomplete
		}
		items = append(items, LifecycleQuestionPageItem{QuestionKey: question.QuestionKey, QuestionRef: question.QuestionRef, RootSequence: question.RootSequence, Projection: question.Projection, QuestionText: body.Text, QuestionTextDigest: body.TextDigest})
	}
	result := QuestionTextPage{Items: items, Page: page, PageSize: pageSize}
	if end < len(aggregate.Questions) {
		next := page + 1
		result.NextPage = &next
	}
	return result, nil
}

type transientQuestionText struct {
	Text       string
	TextDigest string
}

func (service *Service) resolveLifecycleQuestionText(ctx context.Context, generationID string, questions []LifecycleQuestionSnapshot) (map[string]transientQuestionText, error) {
	baseIdentities := make([]aga.BaseIdentity, 0, len(questions))
	for _, question := range questions {
		if question.QuestionRef.Base != nil {
			baseIdentities = append(baseIdentities, *question.QuestionRef.Base)
		}
	}
	var bodies []QuestionBody
	var err error
	if len(baseIdentities) > 0 {
		bodies, err = composeReviewPage(ctx, service.questionBodies, baseIdentities, "")
		if err != nil {
			return nil, err
		}
	}
	byIdentity := bodyMap(bodies)
	var workspaceVersions []preprod.WorkspaceQuestionVersion
	workspaceVersionsLoaded := false
	byKey := make(map[string]transientQuestionText, len(questions))
	for _, question := range questions {
		if question.QuestionRef.Base != nil {
			body, found := byIdentity[question.QuestionRef.Base.Key()]
			if !found || body.TextDigest != question.QuestionRef.Base.TextDigest {
				return nil, ErrQuestionBodyIncomplete
			}
			byKey[question.QuestionRef.Key()] = transientQuestionText{Text: body.Text, TextDigest: body.TextDigest}
			continue
		}
		if question.QuestionRef.Workspace == nil || question.QuestionRef.Workspace.GenerationID != generationID {
			return nil, ErrQuestionBodyIncomplete
		}
		if !workspaceVersionsLoaded {
			workspaceVersions, err = service.listCurrentWorkspaceQuestionVersions(ctx, generationID)
			if err != nil {
				return nil, ErrQuestionBodyIncomplete
			}
			workspaceVersionsLoaded = true
		}
		found := false
		for _, version := range workspaceVersions {
			if version.Reference().Key() != question.QuestionRef.Key() || version.BodyDigest != question.QuestionRef.Workspace.BodyDigest || version.BodyDigest != aga.ComputeWorkspaceBodyDigest(version.Body) {
				continue
			}
			byKey[question.QuestionRef.Key()] = transientQuestionText{Text: version.Body, TextDigest: version.BodyDigest}
			found = true
			break
		}
		if !found {
			return nil, ErrQuestionBodyIncomplete
		}
	}
	return byKey, nil
}

func ProjectLifecycle(aggregate LifecycleAggregate, principal identity.Principal) LifecycleProjection {
	projection := LifecycleProjection{
		InspectionID: aggregate.InspectionID, GenerationID: aggregate.GenerationID, OrganizationID: aggregate.OrganizationID,
		ProviderScopeID: aggregate.ProviderScopeID, State: aggregate.State, Revision: aggregate.Revision,
		Questions: append([]LifecycleQuestionSnapshot{}, aggregate.Questions...), Responses: append([]LifecycleResponse{}, aggregate.Responses...),
		PotentialFindings: append([]LifecyclePotentialFinding{}, aggregate.PotentialFindings...), Findings: append([]LifecycleFinding{}, aggregate.Findings...),
		CAPRevisions: append([]LifecycleCAPRevision{}, aggregate.CAPRevisions...), EvidenceVersions: append([]LifecycleEvidenceVersion{}, aggregate.EvidenceVersions...),
		VerificationDecisions: append([]LifecycleVerificationDecision{}, aggregate.VerificationDecisions...), UpdatedAt: aggregate.UpdatedAt, Digest: aggregate.Digest,
	}
	redactLifecycleProjection(&projection)
	projection.CurrentOwnerRole, projection.NextAction = lifecycleOwnerAndAction(aggregate)
	_ = principal
	return projection
}

func ProjectCAALifecycle(aggregate LifecycleAggregate, principal identity.Principal) LifecycleCAAProjection {
	projection := LifecycleCAAProjection{LifecycleProjection: LifecycleProjection{
		InspectionID: aggregate.InspectionID, GenerationID: aggregate.GenerationID, OrganizationID: aggregate.OrganizationID,
		ProviderScopeID: aggregate.ProviderScopeID, State: aggregate.State, Revision: aggregate.Revision,
		Questions: append([]LifecycleQuestionSnapshot{}, aggregate.Questions...), Responses: append([]LifecycleResponse{}, aggregate.Responses...),
		PotentialFindings: append([]LifecyclePotentialFinding{}, aggregate.PotentialFindings...), Findings: append([]LifecycleFinding{}, aggregate.Findings...),
		CAPRevisions: append([]LifecycleCAPRevision{}, aggregate.CAPRevisions...), EvidenceVersions: append([]LifecycleEvidenceVersion{}, aggregate.EvidenceVersions...),
		VerificationDecisions: append([]LifecycleVerificationDecision{}, aggregate.VerificationDecisions...), UpdatedAt: aggregate.UpdatedAt, Digest: aggregate.Digest,
	}, RecommendationID: aggregate.RecommendationID, RecommendationDigest: aggregate.RecommendationDigest, Inspector: aggregate.Inspector, Lead: aggregate.Lead, Auditee: aggregate.Auditee, ReasonHistory: append([]LifecycleReasonRecord{}, aggregate.ReasonHistory...)}
	projection.CurrentOwnerRole, projection.NextAction = lifecycleOwnerAndAction(aggregate)
	projection.RoleHistory = []LifecycleRoleEvent{{Role: "INSPECTOR", Action: "PINNED", OccurredAt: aggregate.CreatedAt}, {Role: "LEAD", Action: "PINNED", OccurredAt: aggregate.CreatedAt}, {Role: "AUDITEE", Action: "PINNED", OccurredAt: aggregate.CreatedAt}}
	_ = principal
	return projection
}

func ProjectAuditeeLifecycle(aggregate LifecycleAggregate, principal identity.Principal) (LifecycleAuditeeProjection, error) {
	if !lifecycleBindingPinMatchesPrincipal(aggregate.Auditee, principal) || principal.SubjectID != aggregate.Auditee.SubjectID || !principal.HasRole(identity.RoleAuditee) {
		return LifecycleAuditeeProjection{}, ErrNeutralDenied
	}
	projection := LifecycleAuditeeProjection{
		InspectionID: aggregate.InspectionID, GenerationID: aggregate.GenerationID, OrganizationID: aggregate.OrganizationID,
		State: aggregate.State, Revision: aggregate.Revision, NextAction: publicOwnerLabel(aggregate),
		PublicOwnerLabel: publicOwnerLabel(aggregate), UpdatedAt: aggregate.UpdatedAt, Digest: aggregate.Digest,
		Findings:              make([]LifecycleAuditeeFinding, 0, len(aggregate.Findings)),
		CAPRevisions:          make([]LifecycleAuditeeCAPRevision, 0, len(aggregate.CAPRevisions)),
		EvidenceVersions:      make([]LifecycleAuditeeEvidenceVersion, 0, len(aggregate.EvidenceVersions)),
		VerificationDecisions: make([]LifecycleAuditeeVerificationDecision, 0, len(aggregate.VerificationDecisions)),
	}
	for _, finding := range aggregate.Findings {
		projection.Findings = append(projection.Findings, LifecycleAuditeeFinding{
			FindingID: finding.FindingID, Severity: finding.Severity, State: finding.State, NextAction: finding.NextAction,
			CAPRequired: finding.CAPRequired, EvidenceRequired: finding.EvidenceRequired, DueDateRequired: finding.DueDateRequired,
			DueDate: finding.DueDate, Revision: finding.Revision, CreatedAt: finding.CreatedAt,
		})
	}
	for _, revision := range aggregate.CAPRevisions {
		projection.CAPRevisions = append(projection.CAPRevisions, LifecycleAuditeeCAPRevision{
			CAPID: revision.CAPID, FindingID: revision.FindingID, Revision: revision.Revision, State: revision.State,
			RootCause: revision.RootCause, CorrectiveAction: revision.CorrectiveAction, PreventiveAction: revision.PreventiveAction,
			ResponsiblePerson: revision.ResponsiblePerson, TargetDate: revision.TargetDate, CommentToAuditee: revision.CommentToAuditee,
			CreatedAt: revision.CreatedAt,
		})
	}
	for _, evidence := range aggregate.EvidenceVersions {
		projection.EvidenceVersions = append(projection.EvidenceVersions, LifecycleAuditeeEvidenceVersion{
			EvidenceID: evidence.EvidenceID, FindingID: evidence.FindingID, Version: evidence.Version, FileName: evidence.FileName,
			ReviewState: evidence.ReviewState, CommentToAuditee: evidence.CommentToAuditee, CreatedAt: evidence.CreatedAt,
		})
	}
	for _, decision := range aggregate.VerificationDecisions {
		projection.VerificationDecisions = append(projection.VerificationDecisions, LifecycleAuditeeVerificationDecision{
			VerificationID: decision.VerificationID, FindingID: decision.FindingID, EvidenceID: decision.EvidenceID,
			EvidenceVersion: decision.EvidenceVersion, Outcome: decision.Outcome, CommentToAuditee: decision.CommentToAuditee,
			CreatedAt: decision.CreatedAt,
		})
	}
	return projection, nil
}

func lifecycleVisibleToPrincipal(aggregate LifecycleAggregate, principal identity.Principal) bool {
	if principal.OrganizationID == "" || principal.SubjectID == "" {
		return false
	}
	if principal.HasRole(identity.RoleAdmin) && isCAAOrganization(principal.OrganizationID) {
		return true
	}
	if principal.HasRole(identity.RoleDepartmentManager) && workspaceOrganizationMatchesPrincipal(principal.OrganizationID, aggregate.OrganizationID) {
		return true
	}
	return lifecycleBindingPinMatchesPrincipal(aggregate.Inspector, principal) || lifecycleBindingPinMatchesPrincipal(aggregate.Lead, principal) || lifecycleBindingPinMatchesPrincipal(aggregate.Auditee, principal)
}

func lifecycleBindingPinMatchesPrincipal(pin LifecycleBindingPin, principal identity.Principal) bool {
	if pin.SubjectID == "" || principal.SubjectID == "" || pin.SubjectID != principal.SubjectID || principal.OrganizationID == "" {
		return false
	}
	sourceOrganizationID := pin.SourceOrganizationID
	if sourceOrganizationID == "" {
		sourceOrganizationID = pin.OrganizationID
	}
	return sourceOrganizationID == principal.OrganizationID || workspaceOrganizationMatchesPrincipal(principal.OrganizationID, pin.OrganizationID)
}

func redactLifecycleProjection(projection *LifecycleProjection) {
	if projection == nil {
		return
	}
	for index := range projection.Responses {
		projection.Responses[index].ActorSubjectID = ""
	}
	for index := range projection.PotentialFindings {
		projection.PotentialFindings[index].ActorSubjectID = ""
	}
	for index := range projection.CAPRevisions {
		projection.CAPRevisions[index].ActorSubjectID = ""
		projection.CAPRevisions[index].InternalCAANote = ""
	}
	for index := range projection.EvidenceVersions {
		projection.EvidenceVersions[index].ActorSubjectID = ""
		projection.EvidenceVersions[index].InternalCAANote = ""
	}
	for index := range projection.VerificationDecisions {
		projection.VerificationDecisions[index].ActorSubjectID = ""
		projection.VerificationDecisions[index].InternalCAANote = ""
	}
}

func lifecycleOwnerAndAction(aggregate LifecycleAggregate) (string, string) {
	owner, action := "Department Manager", "Review synthetic lifecycle"
	switch aggregate.State {
	case InspectionReady:
		owner, action = "Assigned Inspector", "Start inspection"
	case InspectionInProgress:
		owner, action = "Assigned Inspector", "Record checklist response"
	case InspectionSubmitted:
		owner, action = "Lead Inspector", "Review Potential Findings"
	case InspectionCompleted:
		owner, action = "Department Manager", "Review completed inspection"
	}
	for _, finding := range aggregate.Findings {
		if finding.State != FindingClosed {
			owner, action = "Service Provider", finding.NextAction
		}
	}
	return owner, action
}

func publicOwnerLabel(aggregate LifecycleAggregate) string {
	_, action := lifecycleOwnerAndAction(aggregate)
	if strings.TrimSpace(action) == "" {
		return "CAA workspace"
	}
	return action
}
