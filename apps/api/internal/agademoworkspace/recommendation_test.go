package agademoworkspace

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

func recommendationCommandForTest(generationID string) CommandEnvelope {
	return CommandEnvelope{
		OperationID: OperationCreateRecommendation, IdempotencyKey: "recommendation-test-idempotency", ExpectedGenerationID: generationID,
		ExpectedDraftRevision: 1, DraftID: "aga-ws-draft-0001", DraftRevision: 1, DraftContentDigest: digestForTest(),
		OrganizationID: "AGA-DEMO-CAA", ProviderScopeRootID: "aga-ws-scope-root-matching", ProviderScopeID: "aga-ws-scope-matching", ProviderScopeVersion: 1,
		ProviderTypeID: "aga-ws-provider-type-aerodrome-operator", DepartmentID: "AGA-DEMO-DEPARTMENT", OrganizationalUnitID: "AGA-DEMO-UNIT",
		TargetID: "aga-ws-target-matching", CanonicalTargetKind: "SYSTEM", TargetProfileCode: "AERODROME_MANAGEMENT_SYSTEM",
		InspectionProfileCode: "AERODROME_MANAGEMENT_SYSTEM", InspectionTypeCode: "PERIODIC_SURVEILLANCE",
		OperationQualifiers: []aga.Qualifier{{Key: "OPERATION_STATUS", Value: "ACTIVE"}}, ActivityQualifiers: []aga.Qualifier{{Key: "ACTIVITY_TYPE", Value: "MAINTENANCE"}},
		EffectiveAt: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC), TaxonomyVersion: "AGA_QUESTION_CLASSIFICATION_V1", TaxonomyDigest: digestForTest(),
		ClassificationRunID: "aga-classification-run-test", ClassificationRunDigest: digestForTest(), ReadinessEventID: "aga-ws-readiness-event-0001", ReadinessEventDigest: digestForTest(),
	}
}

func TestRecommendationRequestPreservesEveryServerPin(t *testing.T) {
	command := recommendationCommandForTest("aga-ws-generation-0001")
	request := recommendationRequest(command)
	if request.OperationID != command.OperationID || request.IdempotencyKey != command.IdempotencyKey || request.ExpectedGenerationID != command.ExpectedGenerationID || request.DraftRevision != command.DraftRevision || request.ExpectedDraftRevision != command.ExpectedDraftRevision || request.ReadinessEventID != command.ReadinessEventID || request.ReadinessEventDigest != command.ReadinessEventDigest {
		t.Fatal("recommendation request omitted a generation, Draft, or readiness pin")
	}
	if !reflect.DeepEqual(request.OperationQualifiers, command.OperationQualifiers) || !reflect.DeepEqual(request.ActivityQualifiers, command.ActivityQualifiers) || request.EffectiveAt != command.EffectiveAt {
		t.Fatal("recommendation request changed qualifier or effective-time facts")
	}
}

func TestRecommendationFactsUnavailableWritesNothing(t *testing.T) {
	store := &serviceTestStore{workspace: testWorkspace(), responses: map[preprod.StoredResponseKey]preprod.IdempotencyResponse{}}
	service := NewService(ServiceConfig{
		Store:    store,
		Resolver: StaticBindingResolver{Bindings: map[string]preprod.AuthorityBinding{"manager": bindingFor("manager", "AGA-DEMO-CAA", "manager")}},
	})
	command := recommendationCommandForTest(store.workspace.Generation.GenerationID)
	_, err := service.Command(context.Background(), principalFor("manager", "AGA-DEMO-CAA", identity.RoleDepartmentManager), FamilyRecommendationCommand, command)
	if !errors.Is(err, ErrRecommendationFactsUnavailable) {
		t.Fatalf("missing server facts error = %v, want %v", err, ErrRecommendationFactsUnavailable)
	}
	if len(store.responses) != 0 {
		t.Fatal("neutral recommendation failure wrote an idempotency response")
	}
}

func TestRecommendationSnapshotStoreRejectsTamperedSnapshot(t *testing.T) {
	store := preprod.NewMemoryStore()
	_, _, err := store.PutRecommendationSnapshot(context.Background(), preprod.RecommendationSnapshot{})
	if !errors.Is(err, preprod.ErrWorkspaceNotSealed) {
		t.Fatalf("unsealed recommendation store error = %v", err)
	}
}

func validRecommendationSnapshotForTest(t *testing.T) preprod.RecommendationSnapshot {
	t.Helper()
	taxonomy := aga.FrozenTaxonomy()
	question := aga.QuestionRef{
		Origin:       aga.QuestionOriginBase,
		Base:         &aga.BaseIdentity{PackageVersion: aga.FrozenPackageVersion, PackageJSONSHA256: aga.FrozenPackageJSONSHA256, FormCode: "FSS-AGA-FORM-001", ProposalID: "proposal-1", Ordinal: 1, TextDigest: digestForTest()},
		RootSequence: 1,
	}
	projection := aga.ProposalProjection{
		MainDomainCode: "GOVERNANCE_ORGANIZATION_PERSONNEL", TopicCodes: []string{"AERODROME_ORGANIZATIONAL_CHANGE"},
		InspectionProfileCodes: []string{"AERODROME_MANAGEMENT_SYSTEM"}, InspectionTypeCodes: []string{"PERIODIC_SURVEILLANCE"},
		CanonicalTargetKind: "SYSTEM", TargetProfileCode: "AERODROME_MANAGEMENT_SYSTEM",
		OperationQualifiers: []aga.Qualifier{{Key: "OPERATION_STATUS", Value: "ACTIVE"}}, ActivityQualifiers: []aga.Qualifier{{Key: "ACTIVITY_TYPE", Value: "MAINTENANCE"}},
		ApplicabilityDisposition: "APPLICABLE", EvidenceExpectationCodes: []string{"SOURCE_REFERENCE"},
	}
	recommendation := aga.Recommendation{
		RecommendationID: "aga-ws-recommendation-00000001", Revision: 1, OperationID: OperationCreateRecommendation, IdempotencyKey: "recommendation-test-idempotency",
		GenerationID: "aga-ws-generation-0001", DraftID: "aga-ws-draft-0001", DraftRevision: 1, DraftContentDigest: digestForTest(),
		TaxonomyVersion: taxonomy.Version, TaxonomyDigest: taxonomy.Digest, ClassificationRunID: "aga-classification-run-test", ClassificationRunDigest: digestForTest(), AggregateDigest: digestForTest(),
		OrganizationID: "AGA-DEMO-CAA", ProviderScopeRootID: "aga-ws-scope-root-0001", ProviderScopeID: "aga-ws-scope-0001", ProviderScopeVersion: 1, ProviderScopeProfileDigest: digestForTest(),
		ProviderTypeID: "provider-type", ProviderTypeCode: "AERODROME_OPERATOR", DepartmentID: "department", OrganizationalUnitID: "unit", TargetID: "target", CanonicalTargetKind: "SYSTEM", TargetProfileCode: "AERODROME_MANAGEMENT_SYSTEM",
		InspectionProfileCode: "AERODROME_MANAGEMENT_SYSTEM", InspectionTypeCode: "PERIODIC_SURVEILLANCE", OperationQualifiers: projection.OperationQualifiers, ActivityQualifiers: projection.ActivityQualifiers,
		EffectiveAt: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC), ReadinessEventID: "aga-ws-readiness-event-0001", ReadinessEventDigest: digestForTest(),
		Items: []aga.RecommendationItem{{QuestionRef: question, RootSequence: 1, Current: true, DraftDisposition: aga.DispositionInclude, Projection: projection}},
	}
	var err error
	recommendation.Digest, err = aga.DigestExcludingJSONFields("AGA-DETERMINISTIC-RECOMMENDATION-V1", recommendation, "digest")
	if err != nil {
		t.Fatalf("recommendation digest error = %v", err)
	}
	snapshot := preprod.RecommendationSnapshot{Recommendation: recommendation, CreatedAt: time.Date(2026, 8, 4, 12, 1, 0, 0, time.UTC)}
	snapshot.SnapshotDigest, err = aga.DigestExcludingJSONFields("AGA-DEMO-RECOMMENDATION-SNAPSHOT-V1", snapshot, "snapshotDigest")
	if err != nil {
		t.Fatalf("snapshot digest error = %v", err)
	}
	return snapshot
}

func TestRecommendationSnapshotIsImmutableAndFullyReferenced(t *testing.T) {
	snapshot := validRecommendationSnapshotForTest(t)
	if err := preprod.ValidateRecommendationSnapshot(snapshot); err != nil {
		t.Fatalf("valid recommendation snapshot error = %v", err)
	}
	snapshot.Recommendation.Items[0].Current = false
	if err := preprod.ValidateRecommendationSnapshot(snapshot); !errors.Is(err, preprod.ErrWorkspaceAppendOnly) {
		t.Fatalf("tampered recommendation snapshot error = %v", err)
	}
}

func TestRecommendationSnapshotRoundTripsThroughJSONBShape(t *testing.T) {
	snapshot := validRecommendationSnapshotForTest(t)
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal recommendation snapshot: %v", err)
	}
	var decoded preprod.RecommendationSnapshot
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal recommendation snapshot: %v", err)
	}
	if err := preprod.ValidateRecommendationSnapshot(decoded); err != nil {
		recommendationDigest, _ := aga.DigestExcludingJSONFields("AGA-DETERMINISTIC-RECOMMENDATION-V1", decoded.Recommendation, "digest")
		snapshotDigest, _ := aga.DigestExcludingJSONFields("AGA-DEMO-RECOMMENDATION-SNAPSHOT-V1", decoded, "snapshotDigest")
		item := decoded.Recommendation.Items[0]
		t.Fatalf("JSONB-shaped recommendation snapshot validation error = %v deepEqual=%t recommendationDigestMatch=%t snapshotDigestMatch=%t questionRefError=%v projectionError=%v current=%t rootSequence=%d questionRootSequence=%d disposition=%s originalRecommendationDigest=%s decodedRecommendationDigest=%s originalSnapshotDigest=%s decodedSnapshotDigest=%s", err, reflect.DeepEqual(snapshot, decoded), decoded.Recommendation.Digest == recommendationDigest, decoded.SnapshotDigest == snapshotDigest, aga.ValidateQuestionRef(item.QuestionRef), aga.ValidateProjection(aga.FrozenTaxonomy(), item.Projection), item.Current, item.RootSequence, item.QuestionRef.RootSequence, item.DraftDisposition, snapshot.Recommendation.Digest, recommendationDigest, snapshot.SnapshotDigest, snapshotDigest)
	}
}
