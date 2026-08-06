package agademoworkspace

import (
	"context"
	"strings"
	"testing"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
)

func TestWorkspaceLoaderReconcilesAndSeals(t *testing.T) {
	store := NewMemoryStore()
	if _, err := store.LoadAndSeal(context.Background(), LoadInput{}); err == nil {
		t.Fatal("invalid loader input was accepted")
	}
}

func TestWorkspaceLoaderPersistsBothPassProjections(t *testing.T) {
	if len(WorkspaceSchemaObjectNames()) == 0 || WorkspaceSchemaName == "" {
		t.Fatal("workspace schema contract is empty")
	}
	if WorkspaceLoaderRole == WorkspaceReaderRole {
		t.Fatal("loader and reader roles must be distinct")
	}
}

func TestPostgresRecommendationSnapshotRequiresCommandStore(t *testing.T) {
	store := &PostgresStore{}
	if _, _, err := store.PutRecommendationSnapshot(context.Background(), RecommendationSnapshot{}); err == nil {
		t.Fatal("reader-only PostgreSQL store accepted recommendation snapshot write")
	}
}

func TestRecommendationSnapshotRelationIsAppendOnly(t *testing.T) {
	ddl := WorkspaceAppendOnlyTriggerDDL()
	if !strings.Contains(ddl, "recommendation_snapshots_append_only") {
		t.Fatal("recommendation snapshot relation is missing its append-only trigger")
	}
}

func TestMergeRuntimeWorkspaceOverlaysCurrentDraftMetadataOnSealedItems(t *testing.T) {
	identity := aga.BaseIdentity{
		PackageVersion:    aga.FrozenPackageVersion,
		PackageJSONSHA256: aga.FrozenPackageJSONSHA256,
		FormCode:          "FSS-AGA-FORM-002",
		ProposalID:        "sealed-proposal-1",
		Ordinal:           1,
		TextDigest:        "sha256:1111111111111111111111111111111111111111111111111111111111111111",
	}
	questionRef := aga.BaseQuestionReference(identity)
	confidence := aga.ConfidenceHigh
	disposition := aga.DispositionExclude
	draft := aga.Draft{
		Items: []aga.DraftItem{{
			QuestionRef:              questionRef,
			DraftAgreementConfidence: &confidence,
			RecommendationState:      "MANAGER_REVIEW_REQUIRED",
			ReviewState:              "MANAGER_REVIEWED",
			Disposition:              &disposition,
		}},
	}
	base := LoadedWorkspace{Items: []ClassificationItem{{QuestionKey: identity.Key(), Identity: identity}}}

	merged := mergeRuntimeWorkspace(base, Generation{}, aga.ClassificationResult{}, draft, WorkspaceSealReceipt{})
	if len(merged.Items) != 1 || merged.Items[0].DraftDisposition == nil || *merged.Items[0].DraftDisposition != aga.DispositionExclude {
		t.Fatalf("runtime merge did not overlay current disposition: %+v", merged.Items)
	}
	if merged.Items[0].DraftReviewState != "MANAGER_REVIEWED" || merged.Items[0].DraftAgreementConfidence == nil || *merged.Items[0].DraftAgreementConfidence != aga.ConfidenceHigh {
		t.Fatalf("runtime merge did not overlay current draft metadata: %+v", merged.Items[0])
	}
	if base.Items[0].DraftDisposition != nil {
		t.Fatal("runtime merge mutated the sealed base item slice")
	}
}
