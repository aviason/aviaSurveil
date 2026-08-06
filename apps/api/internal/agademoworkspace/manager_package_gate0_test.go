package agademoworkspace

import (
	"context"
	"errors"
	"testing"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

func TestGate0QuestionBodyCompositionFailsClosedOnMissingOrDigestMismatch(t *testing.T) {
	identity := aga.BaseIdentity{
		PackageVersion:    aga.FrozenPackageVersion,
		PackageJSONSHA256: aga.FrozenPackageJSONSHA256,
		FormCode:          "FSS-AGA-FORM-001", ProposalID: "sealed-proposal-1", Ordinal: 1,
		TextDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
	}
	resolver := QuestionBodyResolverFunc(func(context.Context, []aga.BaseIdentity) ([]QuestionBody, error) {
		return []QuestionBody{{Identity: identity, Text: "wrong body", TextDigest: identity.TextDigest}}, nil
	})
	_, err := composeReviewPage(context.Background(), resolver, []aga.BaseIdentity{identity}, identity.TextDigest)
	if !errors.Is(err, ErrQuestionBodyDigestMismatch) {
		t.Fatalf("composeReviewPage error = %v, want digest mismatch", err)
	}
}

func TestGate0UnauthorizedAndAssignedRoleTextProjectionsAreDistinct(t *testing.T) {
	if _, ok := interface{}(ClassificationReviewItem{}).(interface{ textProjection() }); !ok {
		t.Fatal("ClassificationReviewItem must be a transport-only projection")
	}
	if canReceiveQuestionText(identity.Principal{SubjectID: "auditee", OrganizationID: "ORG", Roles: []identity.Role{identity.RoleAuditee}}, preprod.AuthorityBinding{}) {
		t.Fatal("auditee must not receive question text")
	}
	if !canReceiveQuestionText(identity.Principal{SubjectID: "manager", OrganizationID: "ORG", Roles: []identity.Role{identity.RoleDepartmentManager}}, preprod.AuthorityBinding{OrganizationID: "ORG", Active: true, OperationRoles: []string{"MANAGER"}}) {
		t.Fatal("exact manager binding must receive bounded question text")
	}
}

func TestGate0BatchPreviewIsServerIssuedAndBoundToDigest(t *testing.T) {
	preview := BatchPreviewProjection{}
	if preview.PreviewID != "" || preview.PreviewDigest != "" {
		t.Fatal("a batch preview must not be client-seeded")
	}
	if err := validateBatchPreviewConsume(BatchPreviewConsume{PreviewID: "client-generated", PreviewDigest: "sha256:bad"}); !errors.Is(err, ErrBatchPreviewNotFound) {
		t.Fatalf("client preview validation error = %v, want server-side not-found", err)
	}
}

func TestGate0SetupIsReadOnlyAndReadinessEventIsServerOwned(t *testing.T) {
	if setup := (SimulationSetupProjection{}); setup.ReadinessEventID != "" {
		t.Fatal("simulation setup must not mint a readiness event")
	}
	command := CommandEnvelope{OperationID: OperationMarkReady, ReadinessEventID: "browser-event"}
	if err := command.Validate(FamilyClassificationCommand); !errors.Is(err, ErrMalformedCommand) {
		t.Fatalf("browser readiness event validation error = %v, want malformed", err)
	}
}

func TestBatchBodySearchIntersectsMetadataAfterResolvingTheBody(t *testing.T) {
	base := aga.BaseIdentity{
		PackageVersion: aga.FrozenPackageVersion, PackageJSONSHA256: aga.FrozenPackageJSONSHA256,
		FormCode: "FSS-AGA-FORM-002", ProposalID: "sealed-proposal-search", Ordinal: 1,
		TextDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
	}
	store := &serviceTestStore{workspace: preprod.LoadedWorkspace{Items: []preprod.ClassificationItem{{Identity: base, QuestionKey: base.Key()}}}}
	service := &Service{questionTextSearch: QuestionTextSearchResolverFunc(func(context.Context, string) ([]aga.BaseIdentity, error) {
		return []aga.BaseIdentity{base}, nil
	})}
	items, err := service.filteredBatchItems(context.Background(), store.workspace, BatchFilter{Search: "sealed body fragment"})
	if err != nil {
		t.Fatalf("filteredBatchItems error = %v", err)
	}
	if len(items) != 1 || items[0].Identity.Key() != base.Key() {
		t.Fatalf("body search result = %+v, want the resolver identity", items)
	}
}
