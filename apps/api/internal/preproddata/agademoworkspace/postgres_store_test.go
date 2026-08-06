package agademoworkspace

import (
	"context"
	"strings"
	"testing"
	"time"

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

func TestResetReconstructionRequiresCompleteFreshGenerationGraph(t *testing.T) {
	fixture := resetFixtureForTest(t)
	generationID := "aga-ws-generation-reset-test"
	authorityBindings := workspaceAuthorityBindingRows(generationID, fixture)
	providerScopes, providerTargets := workspaceProviderRows(generationID, time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC))
	if err := validateResetReconstruction(generationID, fixture, authorityBindings, providerScopes, providerTargets); err != nil {
		t.Fatalf("valid reset reconstruction rejected: %v", err)
	}
	providerTargets[0].(map[string]any)["generation_id"] = "aga-ws-generation-stale"
	if err := validateResetReconstruction(generationID, fixture, authorityBindings, providerScopes, providerTargets); err == nil {
		t.Fatal("stale generation-scoped target accepted during reset reconstruction")
	}
}

func resetFixtureForTest(t *testing.T) FixtureManifest {
	t.Helper()
	template := DefaultFixtureTemplate()
	accounts := make([]FixtureAccount, 0, len(template.RoleSlots))
	for _, slot := range template.RoleSlots {
		organizationID := "AGA-DEMO-CAA"
		if slot.Slot == "AUDITEE_OTHER_ORGANIZATION" {
			organizationID = "AGA-DEMO-OTHER-ORG"
		}
		accounts = append(accounts, FixtureAccount{
			Slot: slot.Slot, SubjectID: "subject-" + slot.Slot, MembershipID: "membership-" + slot.Slot,
			MembershipVersion: 1, OrganizationID: organizationID,
			MembershipDigest: digestValue("AGA-DEMO-TEST-MEMBERSHIP-V1", slot.Slot),
		})
	}
	fixture, err := ExportFixture(
		context.Background(), template,
		FixtureSourceFunc(func(context.Context, []string) ([]FixtureAccount, error) { return accounts, nil }),
		"aga-ws-fixture-test", digestValue("AGA-DEMO-TEST-TARGET-V1", "target"), "base-run-test",
		digestValue("AGA-DEMO-TEST-PROVIDER-V1", "provider"), time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("export test fixture: %v", err)
	}
	return fixture
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
	base := LoadedWorkspace{Items: []ClassificationItem{{QuestionKey: aga.BaseQuestionReference(identity).Key(), Identity: identity}}}

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
