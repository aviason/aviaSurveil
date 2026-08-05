package agademoworkspace

import (
	"context"
	"strings"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

func (service *Service) lifecycleQuery(ctx context.Context, principal identity.Principal, request QueryRequest) (QueryResponse, error) {
	if strings.TrimSpace(request.InspectionID) == "" {
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
	if err != nil || !lifecycleVisibleToPrincipal(aggregate, principal) {
		return QueryResponse{}, ErrNeutralDenied
	}
	response := QueryResponse{Operation: request.OperationID, Generation: workspace.Generation, LifecycleAvailable: true}
	if principal.HasRole(identity.RoleAuditee) {
		projection, projectionErr := ProjectAuditeeLifecycle(aggregate, principal)
		if projectionErr != nil {
			return QueryResponse{}, projectionErr
		}
		response.LifecycleAuditee = &projection
		response.Lifecycle = &projection.LifecycleProjection
		return response, nil
	}
	projection := ProjectLifecycle(aggregate, principal)
	response.Lifecycle = &projection
	if request.OperationID == OperationGetRoleHistory {
		caa := ProjectCAALifecycle(aggregate, principal)
		response.LifecycleCAA = &caa
	}
	if request.OperationID == OperationGetFinding && request.FindingID != "" {
		if _, found := aggregate.latestFinding(request.FindingID); !found {
			return QueryResponse{}, ErrNeutralDenied
		}
	}
	if request.OperationID == OperationGetCAPEvidence && request.EvidenceID != "" {
		found := false
		for _, evidence := range aggregate.EvidenceVersions {
			if evidence.EvidenceID == request.EvidenceID {
				found = true
				break
			}
		}
		if !found {
			return QueryResponse{}, ErrNeutralDenied
		}
	}
	return response, nil
}

func ProjectLifecycle(aggregate LifecycleAggregate, principal identity.Principal) LifecycleProjection {
	projection := LifecycleProjection{
		InspectionID: aggregate.InspectionID, GenerationID: aggregate.GenerationID, OrganizationID: aggregate.OrganizationID,
		ProviderScopeID: aggregate.ProviderScopeID, State: aggregate.State, Revision: aggregate.Revision,
		Questions: append([]LifecycleQuestionSnapshot(nil), aggregate.Questions...), Responses: append([]LifecycleResponse(nil), aggregate.Responses...),
		PotentialFindings: append([]LifecyclePotentialFinding(nil), aggregate.PotentialFindings...), Findings: append([]LifecycleFinding(nil), aggregate.Findings...),
		CAPRevisions: append([]LifecycleCAPRevision(nil), aggregate.CAPRevisions...), EvidenceVersions: append([]LifecycleEvidenceVersion(nil), aggregate.EvidenceVersions...),
		VerificationDecisions: append([]LifecycleVerificationDecision(nil), aggregate.VerificationDecisions...), UpdatedAt: aggregate.UpdatedAt, Digest: aggregate.Digest,
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
		Questions: append([]LifecycleQuestionSnapshot(nil), aggregate.Questions...), Responses: append([]LifecycleResponse(nil), aggregate.Responses...),
		PotentialFindings: append([]LifecyclePotentialFinding(nil), aggregate.PotentialFindings...), Findings: append([]LifecycleFinding(nil), aggregate.Findings...),
		CAPRevisions: append([]LifecycleCAPRevision(nil), aggregate.CAPRevisions...), EvidenceVersions: append([]LifecycleEvidenceVersion(nil), aggregate.EvidenceVersions...),
		VerificationDecisions: append([]LifecycleVerificationDecision(nil), aggregate.VerificationDecisions...), UpdatedAt: aggregate.UpdatedAt, Digest: aggregate.Digest,
	}, RecommendationID: aggregate.RecommendationID, RecommendationDigest: aggregate.RecommendationDigest, Inspector: aggregate.Inspector, Lead: aggregate.Lead, Auditee: aggregate.Auditee}
	projection.CurrentOwnerRole, projection.NextAction = lifecycleOwnerAndAction(aggregate)
	projection.RoleHistory = []LifecycleRoleEvent{{Role: "INSPECTOR", Action: "PINNED", OccurredAt: aggregate.CreatedAt}, {Role: "LEAD", Action: "PINNED", OccurredAt: aggregate.CreatedAt}, {Role: "AUDITEE", Action: "PINNED", OccurredAt: aggregate.CreatedAt}}
	_ = principal
	return projection
}

func ProjectAuditeeLifecycle(aggregate LifecycleAggregate, principal identity.Principal) (LifecycleAuditeeProjection, error) {
	if !lifecycleBindingPinMatchesPrincipal(aggregate.Auditee, principal) || principal.SubjectID != aggregate.Auditee.SubjectID || !principal.HasRole(identity.RoleAuditee) {
		return LifecycleAuditeeProjection{}, ErrNeutralDenied
	}
	return LifecycleAuditeeProjection{LifecycleProjection: ProjectLifecycle(aggregate, principal), PublicOwnerLabel: publicOwnerLabel(aggregate)}, nil
}

func lifecycleVisibleToPrincipal(aggregate LifecycleAggregate, principal identity.Principal) bool {
	if principal.OrganizationID == "" || principal.SubjectID == "" {
		return false
	}
	if workspaceOrganizationMatchesPrincipal(principal.OrganizationID, aggregate.OrganizationID) {
		return true
	}
	return lifecycleBindingPinMatchesPrincipal(aggregate.Inspector, principal) || lifecycleBindingPinMatchesPrincipal(aggregate.Lead, principal) || lifecycleBindingPinMatchesPrincipal(aggregate.Auditee, principal)
}

func lifecycleBindingPinMatchesPrincipal(pin LifecycleBindingPin, principal identity.Principal) bool {
	if pin.SubjectID == "" || principal.SubjectID == "" || principal.OrganizationID == "" {
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
