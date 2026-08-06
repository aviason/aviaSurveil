package agademoworkspace

import (
	"context"
	"strings"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

// hydrateSetupCommand is the only path by which the browser-facing package
// builder obtains readiness, recommendation, and lifecycle authority facts.
// The input command remains an opaque selection/digest envelope; all mutable
// authority fields below are resolved again from the current workspace before
// the command reaches a write boundary.
func (service *Service) hydrateSetupCommand(ctx context.Context, principal identity.Principal, workspace preprod.LoadedWorkspace, command CommandEnvelope) (CommandEnvelope, error) {
	if strings.TrimSpace(command.SetupDigest) == "" {
		return command, nil
	}
	setup, err := service.simulationSetup(ctx, principal, workspace)
	if err != nil || setup.SimulationSetupDigest != command.SetupDigest {
		return CommandEnvelope{}, ErrRecommendationFactsUnavailable
	}

	switch command.OperationID {
	case OperationMarkReady:
		if setup.ReadinessEventDigest != "" || setup.ReadinessState != aga.DraftWorking {
			return CommandEnvelope{}, ErrCommandConflict
		}
		command.ProviderScopeProfileDigest = setup.ProviderScopeDigest
		command.ReadinessEventID = readinessEventIDFor(workspace.Generation.GenerationID, command.IdempotencyKey, principal.SubjectID)
	case OperationCreateRecommendation:
		if setup.ReadinessState != aga.DraftReadyForDemoSimulation || setup.ReadinessEventDigest == "" {
			return CommandEnvelope{}, ErrCommandConflict
		}
		readiness, found := currentReadinessEvent(workspace.Draft.Draft)
		if !found || readiness.ReadinessEventDigest != setup.ReadinessEventDigest {
			return CommandEnvelope{}, ErrCommandConflict
		}
		effectiveAt, parseErr := time.Parse(time.RFC3339Nano, setup.EffectiveAt)
		if parseErr != nil {
			return CommandEnvelope{}, ErrRecommendationFactsUnavailable
		}
		command.OrganizationID = setup.OrganizationLabel
		command.ProviderScopeRootID = setup.ProviderScopeRootID
		command.ProviderScopeID = setup.ProviderScopeID
		command.ProviderScopeVersion = setup.ProviderScopeVersion
		command.ProviderTypeID = setup.ProviderTypeID
		command.DepartmentID = setup.DepartmentID
		command.OrganizationalUnitID = setup.OrganizationalUnitID
		command.TargetID = setup.TargetID
		command.CanonicalTargetKind = setup.CanonicalTargetKind
		command.TargetProfileCode = setup.TargetProfileCode
		command.InspectionProfileCode = setup.InspectionProfileCode
		command.InspectionTypeCode = setup.InspectionTypeCode
		command.OperationQualifiers = append([]aga.Qualifier(nil), setup.OperationQualifiers...)
		command.ActivityQualifiers = append([]aga.Qualifier(nil), setup.ActivityQualifiers...)
		command.EffectiveAt = effectiveAt
		command.TaxonomyVersion = setup.TaxonomyVersion
		command.TaxonomyDigest = setup.TaxonomyDigest
		command.ClassificationRunID = setup.ClassificationRunID
		command.ClassificationRunDigest = setup.ClassificationRunDigest
		command.DraftID = setup.DraftID
		command.DraftRevision = setup.DraftRevision
		command.DraftContentDigest = setup.DraftContentDigest
		command.ExpectedDraftRevision = setup.DraftRevision
		command.ReadinessEventID = readiness.ReadinessEventID
		command.ReadinessEventDigest = readiness.ReadinessEventDigest
		command.ProviderScopeProfileDigest = setup.ProviderScopeDigest
	case OperationCreateInspection:
		if strings.TrimSpace(command.InspectorSelectionPin) == "" || strings.TrimSpace(command.LeadSelectionPin) == "" {
			return CommandEnvelope{}, ErrLifecycleBindingMismatch
		}
		snapshot, currentErr := service.currentRecommendation(ctx, workspace.Generation.GenerationID)
		if currentErr != nil {
			return CommandEnvelope{}, currentErr
		}
		if setup.RecommendationID == "" || setup.RecommendationID != snapshot.Recommendation.RecommendationID || setup.RecommendationDigest != snapshot.Recommendation.Digest || setup.RecommendationRevision != snapshot.Recommendation.Revision {
			return CommandEnvelope{}, ErrCommandConflict
		}
		if service.lifecycleBindings == nil {
			return CommandEnvelope{}, ErrLifecycleBindingMismatch
		}
		inspectorFacts, inspectorErr := service.lifecycleBindings(ctx, workspace, snapshot.Recommendation, "INSPECTOR")
		if inspectorErr != nil {
			return CommandEnvelope{}, ErrLifecycleBindingMismatch
		}
		leadFacts, leadErr := service.lifecycleBindings(ctx, workspace, snapshot.Recommendation, "LEAD")
		if leadErr != nil {
			return CommandEnvelope{}, ErrLifecycleBindingMismatch
		}
		inspector, selectErr := selectBindingByPin(inspectorFacts, command.InspectorSelectionPin)
		if selectErr != nil {
			return CommandEnvelope{}, selectErr
		}
		lead, selectErr := selectBindingByPin(leadFacts, command.LeadSelectionPin)
		if selectErr != nil {
			return CommandEnvelope{}, selectErr
		}
		command.RecommendationID = snapshot.Recommendation.RecommendationID
		command.RecommendationDigest = snapshot.Recommendation.Digest
		command.ExpectedRecommendationRevision = snapshot.Recommendation.Revision
		command.InspectorBindingID = inspector.BindingID
		command.InspectorBindingRevision = inspector.BindingRevision
		command.LeadBindingID = lead.BindingID
		command.LeadBindingRevision = lead.BindingRevision
	default:
		return command, nil
	}
	return command, nil
}

func currentReadinessEvent(draft aga.Draft) (aga.ReadinessEvent, bool) {
	if strings.TrimSpace(draft.CurrentReadinessEventID) == "" {
		return aga.ReadinessEvent{}, false
	}
	var result aga.ReadinessEvent
	found := false
	for _, event := range draft.ReadinessEvents {
		if event.ReadinessEventID != draft.CurrentReadinessEventID {
			continue
		}
		if found {
			return aga.ReadinessEvent{}, false
		}
		result = event
		found = true
	}
	return result, found
}
