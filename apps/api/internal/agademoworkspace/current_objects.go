package agademoworkspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

type CurrentRecommendationStore interface {
	ListRecommendationSnapshots(context.Context, string) ([]preprod.RecommendationSnapshot, error)
}

type CurrentLifecycleStore interface {
	ListLifecycleStreams(context.Context, string) ([]preprod.LifecycleStream, error)
}

func (service *Service) simulationSetup(ctx context.Context, principal identity.Principal, workspace preprod.LoadedWorkspace) (SimulationSetupProjection, error) {
	if service == nil || service.simulationSetupResolver == nil {
		return SimulationSetupProjection{}, ErrRecommendationFactsUnavailable
	}
	setup, err := service.simulationSetupResolver(ctx, workspace, principal)
	if err != nil {
		return SimulationSetupProjection{}, err
	}
	if setup.GenerationID != workspace.Generation.GenerationID || setup.DraftID != workspace.Draft.Draft.DraftID || setup.DraftRevision != workspace.Draft.Draft.Revision || setup.DraftContentDigest != workspace.Draft.Draft.ContentDigest || setup.TaxonomyVersion != workspace.Generation.TaxonomyVersion || setup.TaxonomyDigest != workspace.Generation.TaxonomyDigest || setup.ClassificationRunID != workspace.Run.Result.ClassificationRunID || setup.ClassificationRunDigest != workspace.Run.ClassificationRunDigest || setup.SimulationSetupDigest == "" {
		return SimulationSetupProjection{}, ErrRecommendationFactsUnavailable
	}
	if setupDigest(setup) != setup.SimulationSetupDigest {
		return SimulationSetupProjection{}, ErrRecommendationFactsUnavailable
	}
	if err := service.enrichSimulationSetup(ctx, &setup, workspace); err != nil {
		return SimulationSetupProjection{}, err
	}
	setup.SimulationSetupDigest = setupDigest(setup)
	if snapshots, currentErr := service.listCurrentRecommendations(ctx, workspace.Generation.GenerationID); currentErr == nil {
		if len(snapshots) > 1 {
			return SimulationSetupProjection{}, ErrCurrentObjectAmbiguous
		}
		if len(snapshots) == 1 {
			setup.RecommendationID = snapshots[0].Recommendation.RecommendationID
			setup.RecommendationRevision = snapshots[0].Recommendation.Revision
			setup.RecommendationDigest = snapshots[0].Recommendation.Digest
			setup.SimulationSetupDigest = setupDigest(setup)
		}
	} else if !errors.Is(currentErr, ErrCapabilityUnavailable) && !errors.Is(currentErr, ErrWorkspaceStore) {
		return SimulationSetupProjection{}, currentErr
	}
	if setup.ReadinessEventID != "" {
		return SimulationSetupProjection{}, ErrRecommendationFactsUnavailable
	}
	return setup, nil
}

func (service *Service) enrichSimulationSetup(ctx context.Context, setup *SimulationSetupProjection, workspace preprod.LoadedWorkspace) error {
	if setup == nil {
		return ErrRecommendationFactsUnavailable
	}
	setup.FormDistribution = map[string]int{}
	setup.DomainDistribution = map[string]int{}
	setup.TopicDistribution = map[string]int{}
	setup.CurrentLeafCount = 0
	setup.IncludedCount = 0
	setup.ExcludedCount = 0
	setup.DeferredCount = 0
	setup.UnsetCount = 0
	setup.IncludedEligibleCount = 0
	setup.IncludedIneligibleCount = 0
	setup.IncludedBlockerCount = 0
	setup.IncludedSourceGapCount = 0
	byKey := make(map[string]preprod.ClassificationItem, len(workspace.Items))
	for _, item := range workspace.Items {
		byKey[item.Identity.Key()] = item
		// Draft question references are typed union keys.  The sealed
		// classification projection historically stores the base identity key,
		// so index both forms before resolving an included Draft item.
		byKey[aga.BaseQuestionReference(item.Identity).Key()] = item
	}
	for _, item := range workspace.Draft.Draft.Items {
		if !item.Current {
			continue
		}
		setup.CurrentLeafCount++
		if item.Disposition == nil {
			setup.UnsetCount++
			continue
		}
		switch *item.Disposition {
		case aga.DispositionInclude:
			setup.IncludedCount++
			metadata, found := byKey[item.QuestionRef.Key()]
			if !found && item.QuestionRef.Workspace != nil {
				metadata, found = service.workspaceDraftMetadata(ctx, workspace.Generation.GenerationID, item)
			}
			if !found {
				return ErrQuestionBodyIdentityMismatch
			}
			if eligibleForSimulation(metadata, *setup) {
				setup.IncludedEligibleCount++
			} else {
				setup.IncludedIneligibleCount++
			}
			if len(metadata.Governance.BlockerCodes) > 0 {
				setup.IncludedBlockerCount++
			}
			if metadata.Governance.QuestionSourceProposalGap {
				setup.IncludedSourceGapCount++
			}
			setup.FormDistribution[metadata.Identity.FormCode]++
			setup.DomainDistribution[metadata.Projection.MainDomainCode]++
			for _, topic := range metadata.Projection.TopicCodes {
				setup.TopicDistribution[topic]++
			}
		case aga.DispositionExclude:
			setup.ExcludedCount++
		case aga.DispositionDefer:
			setup.DeferredCount++
		default:
			setup.UnsetCount++
		}
	}
	return nil
}

func (service *Service) workspaceDraftMetadata(ctx context.Context, generationID string, item aga.DraftItem) (preprod.ClassificationItem, bool) {
	if service == nil || item.QuestionRef.Workspace == nil || item.QuestionRef.Workspace.GenerationID != generationID {
		return preprod.ClassificationItem{}, false
	}
	versions, err := service.listCurrentWorkspaceQuestionVersions(ctx, generationID)
	if err != nil {
		return preprod.ClassificationItem{}, false
	}
	for _, version := range versions {
		if version.VersionID != item.QuestionRef.Workspace.VersionID || version.ProposalID != item.QuestionRef.Workspace.ProposalID || version.BodyDigest != item.QuestionRef.Workspace.BodyDigest || version.BodyDigest != aga.ComputeWorkspaceBodyDigest(version.Body) {
			continue
		}
		return preprod.ClassificationItem{
			QuestionKey: item.QuestionRef.Key(), Projection: item.CurrentProjection,
			RecommendationState: item.RecommendationState,
			Governance: aga.GovernanceState{
				SourceMappingState: aga.SourceMappingRequired, SourceAuthorityState: aga.SourceAuthorityNotAttested,
				RiskClassificationState: aga.RiskExpertReviewRequired, DecisionState: aga.DecisionNotSupplied,
				ExtractionState: aga.ExtractionCandidate, QuestionSourceProposalGap: true,
				BlockerCodes: []string{aga.SourceMappingRequired},
			},
			DraftAgreementConfidence: item.DraftAgreementConfidence, DraftRecommendationState: item.RecommendationState,
			DraftReviewState: item.ReviewState, DraftDisposition: item.Disposition,
		}, true
	}
	return preprod.ClassificationItem{}, false
}

func (service *Service) listCurrentWorkspaceQuestionVersions(ctx context.Context, generationID string) ([]preprod.WorkspaceQuestionVersion, error) {
	if service == nil {
		return nil, ErrWorkspaceStore
	}
	store, ok := service.reader.(preprod.CurrentWorkspaceQuestionVersionStore)
	if !ok {
		store, ok = service.command.(preprod.CurrentWorkspaceQuestionVersionStore)
	}
	if !ok {
		return nil, ErrCapabilityUnavailable
	}
	return store.ListCurrentWorkspaceQuestionVersions(ctx, generationID)
}

func (service *Service) listCurrentRecommendations(ctx context.Context, generationID string) ([]preprod.RecommendationSnapshot, error) {
	if service == nil {
		return nil, ErrWorkspaceStore
	}
	store, ok := service.reader.(CurrentRecommendationStore)
	if !ok {
		store, ok = service.command.(CurrentRecommendationStore)
	}
	if !ok {
		return nil, ErrCapabilityUnavailable
	}
	return store.ListRecommendationSnapshots(ctx, generationID)
}

func (service *Service) currentRecommendation(ctx context.Context, generationID string) (preprod.RecommendationSnapshot, error) {
	snapshots, err := service.listCurrentRecommendations(ctx, generationID)
	if err != nil {
		return preprod.RecommendationSnapshot{}, err
	}
	if len(snapshots) == 0 {
		return preprod.RecommendationSnapshot{}, ErrNeutralDenied
	}
	if len(snapshots) != 1 {
		return preprod.RecommendationSnapshot{}, ErrCurrentObjectAmbiguous
	}
	return snapshots[0], nil
}

func (service *Service) recommendationQuery(ctx context.Context, _ identity.Principal, request QueryRequest) (QueryResponse, error) {
	if service == nil || service.reader == nil {
		return QueryResponse{}, ErrWorkspaceStore
	}
	workspace, err := service.reader.Snapshot(ctx)
	if err != nil {
		return QueryResponse{}, ErrNeutralDenied
	}
	snapshot, currentErr := service.currentRecommendation(ctx, workspace.Generation.GenerationID)
	if currentErr != nil {
		return QueryResponse{}, ErrNeutralDenied
	}
	return QueryResponse{Operation: request.OperationID, Generation: workspace.Generation, RecommendationSnapshot: &snapshot, LifecycleAvailable: false}, nil
}

func readinessEventIDFor(generationID, idempotencyKey, actor string) string {
	digest := sha256.Sum256([]byte("AGA-DEMO-READINESS-EVENT-ID-V1\n" + generationID + "\x00" + idempotencyKey + "\x00" + actor))
	return "aga-ws-readiness-" + hex.EncodeToString(digest[:8])
}

func selectionPinFor(fact LifecycleBindingFact) string {
	digest := sha256.Sum256([]byte("AGA-DEMO-ROLE-SELECTION-PIN-V1\n" + fact.BindingID + "\x00" + fmt.Sprint(fact.BindingRevision) + "\x00" + fact.SubjectID))
	return "aga-ws-role-pin-" + hex.EncodeToString(digest[:8])
}

func selectBindingByPin(facts []LifecycleBindingFact, pin string) (LifecycleBindingFact, error) {
	if strings.TrimSpace(pin) == "" {
		return LifecycleBindingFact{}, ErrLifecycleBindingMismatch
	}
	var selected LifecycleBindingFact
	for _, fact := range facts {
		if selectionPinFor(fact) != pin {
			continue
		}
		if selected.BindingID != "" {
			return LifecycleBindingFact{}, ErrLifecycleBindingMismatch
		}
		selected = fact
	}
	if selected.BindingID == "" {
		return LifecycleBindingFact{}, ErrLifecycleBindingMismatch
	}
	return selected, nil
}

func setupDigest(setup SimulationSetupProjection) string {
	digest, _ := aga.DigestExcludingJSONFields("AGA-DEMO-SIMULATION-SETUP-V1", setup, "simulationSetupDigest", "readinessEventId")
	return digest
}
