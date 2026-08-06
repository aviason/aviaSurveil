package agademoworkspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

// BuildRecommendationSnapshot is the immutable success boundary. It accepts
// one server-derived scope fact and a loaded, sealed workspace; callers cannot
// provide a question list or reconstruct a question reference from a digest.
func BuildRecommendationSnapshot(workspace preprod.LoadedWorkspace, scope aga.ProviderScopeFact, request aga.RecommendationRequest, now time.Time) (preprod.RecommendationSnapshot, error) {
	if workspace.Generation.State != preprod.GenerationActive || workspace.Generation.GenerationID == "" || workspace.Draft.Draft.GenerationID != workspace.Generation.GenerationID {
		return preprod.RecommendationSnapshot{}, ErrRecommendationFactsUnavailable
	}
	recommendation, err := aga.BuildRecommendation(workspace.Draft.Draft, scope, request)
	if err != nil {
		return preprod.RecommendationSnapshot{}, err
	}
	// BuildRecommendation intentionally projects only applicable items. The
	// workspace command boundary must also prove that it did not silently drop
	// an included item whose provider/target/profile facts are incompatible.
	recommended := make(map[string]struct{}, len(recommendation.Items))
	for _, item := range recommendation.Items {
		recommended[item.QuestionRef.Key()] = struct{}{}
	}
	for _, item := range workspace.Draft.Draft.Items {
		if !item.Current || item.Disposition == nil || *item.Disposition != aga.DispositionInclude {
			continue
		}
		if _, ok := recommended[item.QuestionRef.Key()]; !ok {
			return preprod.RecommendationSnapshot{}, ErrIncludedQuestionIneligible
		}
	}
	if recommendation.RecommendationID == "" {
		recommendation.RecommendationID = recommendationIDFor(request.ExpectedGenerationID, request.IdempotencyKey)
	}
	if recommendation.Revision < 1 {
		recommendation.Revision = 1
	}
	recommendation.Digest, err = aga.DigestExcludingJSONFields("AGA-DETERMINISTIC-RECOMMENDATION-V1", recommendation, "digest")
	if err != nil || recommendation.Digest == "" {
		return preprod.RecommendationSnapshot{}, ErrRecommendationFactsUnavailable
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	snapshot := preprod.RecommendationSnapshot{Recommendation: recommendation, CreatedAt: now}
	snapshot.SnapshotDigest, err = aga.DigestExcludingJSONFields("AGA-DEMO-RECOMMENDATION-SNAPSHOT-V1", snapshot, "snapshotDigest")
	if err != nil || snapshot.SnapshotDigest == "" {
		return preprod.RecommendationSnapshot{}, ErrRecommendationFactsUnavailable
	}
	return snapshot, nil
}

func recommendationIDFor(generationID, idempotencyKey string) string {
	hash := sha256.Sum256([]byte("AGA-DEMO-RECOMMENDATION-ID-V1\n" + generationID + "\x00" + idempotencyKey))
	return "aga-ws-recommendation-" + hex.EncodeToString(hash[:8])
}

func recommendationRequest(command CommandEnvelope) aga.RecommendationRequest {
	return aga.RecommendationRequest{
		OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey,
		ExpectedGenerationID: command.ExpectedGenerationID, OrganizationID: command.OrganizationID,
		ProviderScopeRootID: command.ProviderScopeRootID, ProviderScopeID: command.ProviderScopeID,
		ProviderScopeVersion: command.ProviderScopeVersion, ProviderTypeID: command.ProviderTypeID,
		DepartmentID: command.DepartmentID, OrganizationalUnitID: command.OrganizationalUnitID,
		TargetID: command.TargetID, CanonicalTargetKind: command.CanonicalTargetKind,
		TargetProfileCode: command.TargetProfileCode, InspectionProfileCode: command.InspectionProfileCode,
		InspectionTypeCode: command.InspectionTypeCode, OperationQualifiers: append([]aga.Qualifier(nil), command.OperationQualifiers...),
		ActivityQualifiers: append([]aga.Qualifier(nil), command.ActivityQualifiers...), EffectiveAt: command.EffectiveAt.UTC(),
		TaxonomyVersion: command.TaxonomyVersion, TaxonomyDigest: command.TaxonomyDigest,
		ClassificationRunID: command.ClassificationRunID, ClassificationRunDigest: command.ClassificationRunDigest,
		DraftID: command.DraftID, DraftRevision: command.DraftRevision, DraftContentDigest: command.DraftContentDigest,
		ExpectedDraftRevision: command.ExpectedDraftRevision, ReadinessEventID: command.ReadinessEventID,
		ReadinessEventDigest: command.ReadinessEventDigest,
	}
}

func (service *Service) createRecommendation(ctx context.Context, principal identity.Principal, workspace preprod.LoadedWorkspace, command CommandEnvelope) (preprod.RecommendationSnapshot, bool, error) {
	if service == nil || service.recommendationScopes == nil {
		return preprod.RecommendationSnapshot{}, false, ErrRecommendationFactsUnavailable
	}
	store, ok := service.command.(preprod.RecommendationSnapshotStore)
	if !ok {
		return preprod.RecommendationSnapshot{}, false, ErrRecommendationFactsUnavailable
	}
	if currentStore, currentOK := service.command.(CurrentRecommendationStore); currentOK {
		current, currentErr := currentStore.ListRecommendationSnapshots(ctx, workspace.Generation.GenerationID)
		if currentErr != nil {
			return preprod.RecommendationSnapshot{}, false, ErrRecommendationFactsUnavailable
		}
		if len(current) > 0 {
			if len(current) > 1 {
				return preprod.RecommendationSnapshot{}, false, ErrCurrentObjectAmbiguous
			}
			return preprod.RecommendationSnapshot{}, false, ErrCommandConflict
		}
	}
	request := recommendationRequest(command)
	scopes, err := service.recommendationScopes(ctx, workspace, request)
	if err != nil {
		return preprod.RecommendationSnapshot{}, false, ErrRecommendationFactsUnavailable
	}
	if len(scopes) == 0 {
		return preprod.RecommendationSnapshot{}, false, ErrRecommendationFactsUnavailable
	}
	if len(scopes) != 1 {
		return preprod.RecommendationSnapshot{}, false, ErrRecommendationAmbiguous
	}
	snapshot, err := BuildRecommendationSnapshot(workspace, scopes[0], request, service.clock())
	if err != nil {
		return preprod.RecommendationSnapshot{}, false, err
	}
	stored, replayed, err := store.PutRecommendationSnapshot(ctx, snapshot)
	if err != nil {
		return preprod.RecommendationSnapshot{}, false, fmt.Errorf("store recommendation snapshot: %w", err)
	}
	_ = principal
	return stored, replayed, nil
}

func validateRecommendationCommand(command CommandEnvelope) error {
	switch command.OperationID {
	case OperationCreateRecommendation:
		if strings.TrimSpace(command.SetupDigest) != "" {
			if command.DraftRevision < 1 || command.ExpectedDraftRevision < 1 || command.DraftRevision != command.ExpectedDraftRevision || strings.TrimSpace(command.DraftID) == "" || strings.TrimSpace(command.DraftContentDigest) == "" {
				return ErrMalformedCommand
			}
			return nil
		}
		if strings.TrimSpace(command.OrganizationID) == "" || strings.TrimSpace(command.ProviderScopeRootID) == "" || strings.TrimSpace(command.ProviderScopeID) == "" || command.ProviderScopeVersion < 1 || strings.TrimSpace(command.ProviderTypeID) == "" || strings.TrimSpace(command.DepartmentID) == "" || strings.TrimSpace(command.OrganizationalUnitID) == "" || strings.TrimSpace(command.TargetID) == "" || strings.TrimSpace(command.CanonicalTargetKind) == "" || strings.TrimSpace(command.TargetProfileCode) == "" || strings.TrimSpace(command.InspectionProfileCode) == "" || strings.TrimSpace(command.InspectionTypeCode) == "" || command.EffectiveAt.IsZero() || strings.TrimSpace(command.TaxonomyVersion) == "" || strings.TrimSpace(command.TaxonomyDigest) == "" || strings.TrimSpace(command.ClassificationRunID) == "" || strings.TrimSpace(command.ClassificationRunDigest) == "" || strings.TrimSpace(command.ReadinessEventID) == "" || strings.TrimSpace(command.ReadinessEventDigest) == "" {
			return ErrMalformedCommand
		}
	case OperationCreateInspection:
		if strings.TrimSpace(command.SetupDigest) != "" {
			if strings.TrimSpace(command.InspectorSelectionPin) == "" || strings.TrimSpace(command.LeadSelectionPin) == "" {
				return ErrMalformedCommand
			}
			return nil
		}
		if strings.TrimSpace(command.RecommendationID) == "" || strings.TrimSpace(command.RecommendationDigest) == "" || strings.TrimSpace(command.InspectorBindingID) == "" || command.InspectorBindingRevision < 1 || strings.TrimSpace(command.LeadBindingID) == "" || command.LeadBindingRevision < 1 {
			return ErrMalformedCommand
		}
	default:
		return ErrMalformedCommand
	}
	return nil
}
