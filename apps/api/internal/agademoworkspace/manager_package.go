package agademoworkspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

const (
	MaxQuestionTextPage = 25
	MaxBatchPreviewSize = 500
)

type BatchAction string

const (
	BatchInclude BatchAction = "INCLUDE"
	BatchExclude BatchAction = "EXCLUDE"
	BatchDefer   BatchAction = "DEFER"
)

// BatchFilter is the complete visible inventory filter. It intentionally has
// no page, cursor, or client-row fields: the server canonicalizes this value
// and computes the affected set from the sealed workspace snapshot.
type BatchFilter struct {
	Search              string `json:"search,omitempty"`
	DomainCode          string `json:"domainCode,omitempty"`
	TopicCode           string `json:"topicCode,omitempty"`
	Confidence          string `json:"confidence,omitempty"`
	Blocker             string `json:"blocker,omitempty"`
	SourceGap           string `json:"sourceGap,omitempty"`
	ExternalInvolvement string `json:"externalInvolvement,omitempty"`
	FormCode            string `json:"formCode,omitempty"`
	Disposition         string `json:"disposition,omitempty"`
}

type BatchDispositionCounts struct {
	Include int `json:"include"`
	Exclude int `json:"exclude"`
	Defer   int `json:"defer"`
	Unset   int `json:"unset"`
}

type BatchPreviewProjection struct {
	PreviewID               string                 `json:"previewId"`
	GenerationID            string                 `json:"generationId"`
	DraftID                 string                 `json:"draftId"`
	DraftRevision           int                    `json:"draftRevision"`
	DraftContentDigest      string                 `json:"draftContentDigest"`
	ClassificationRunDigest string                 `json:"classificationRunDigest"`
	Filter                  BatchFilter            `json:"filter"`
	FilterDigest            string                 `json:"filterDigest"`
	AffectedIdentityDigest  string                 `json:"affectedIdentityDigest"`
	Action                  BatchAction            `json:"action"`
	ReasonCode              string                 `json:"reasonCode"`
	Count                   int                    `json:"count"`
	CurrentDisposition      BatchDispositionCounts `json:"currentDisposition"`
	EligibleCount           int                    `json:"eligibleCount"`
	IneligibleCount         int                    `json:"ineligibleCount"`
	BlockerCount            int                    `json:"blockerCount"`
	SourceGapCount          int                    `json:"sourceGapCount"`
	ExpiresAt               string                 `json:"expiresAt"`
	PreviewDigest           string                 `json:"previewDigest"`
	ConsumedAt              *string                `json:"consumedAt,omitempty"`
}

type BatchPreviewConsume struct {
	PreviewID     string `json:"previewId"`
	PreviewDigest string `json:"previewDigest"`
}

type SimulationRoleChoice struct {
	SelectionPin string `json:"selectionPin"`
	Label        string `json:"label"`
	Role         string `json:"role"`
}

type SimulationSetupProjection struct {
	// ReadinessEventID is intentionally never serialized. A setup read is
	// side-effect free and cannot issue a commit identifier.
	ReadinessEventID        string                 `json:"-"`
	GenerationID            string                 `json:"generationId"`
	GenerationRevision      int                    `json:"generationRevision"`
	GenerationSealDigest    string                 `json:"generationSealDigest"`
	DraftID                 string                 `json:"draftId"`
	DraftRevision           int                    `json:"draftRevision"`
	DraftContentDigest      string                 `json:"draftContentDigest"`
	TaxonomyVersion         string                 `json:"taxonomyVersion"`
	TaxonomyDigest          string                 `json:"taxonomyDigest"`
	ClassificationRunID     string                 `json:"classificationRunId"`
	ClassificationRunDigest string                 `json:"classificationRunDigest"`
	OrganizationLabel       string                 `json:"organizationLabel"`
	ProviderLabel           string                 `json:"providerLabel"`
	TargetLabel             string                 `json:"targetLabel"`
	ProviderScopeRootID     string                 `json:"providerScopeRootId"`
	ProviderScopeID         string                 `json:"providerScopeId"`
	ProviderScopeVersion    int                    `json:"providerScopeVersion"`
	ProviderTypeID          string                 `json:"providerTypeId"`
	ProviderTypeCode        string                 `json:"providerTypeCode"`
	DepartmentID            string                 `json:"departmentId"`
	OrganizationalUnitID    string                 `json:"organizationalUnitId"`
	TargetID                string                 `json:"targetId"`
	CanonicalTargetKind     string                 `json:"canonicalTargetKind"`
	TargetProfileCode       string                 `json:"targetProfileCode"`
	InspectionProfileCode   string                 `json:"inspectionProfileCode"`
	InspectionTypeCode      string                 `json:"inspectionTypeCode"`
	OperationQualifiers     []aga.Qualifier        `json:"operationQualifiers"`
	ActivityQualifiers      []aga.Qualifier        `json:"activityQualifiers"`
	EffectiveAt             string                 `json:"effectiveAt"`
	ProviderScopeDigest     string                 `json:"providerScopeDigest"`
	ReadinessState          string                 `json:"readinessState"`
	ReadinessEventDigest    string                 `json:"readinessEventDigest,omitempty"`
	RecommendationID        string                 `json:"recommendationId,omitempty"`
	RecommendationRevision  int                    `json:"recommendationRevision,omitempty"`
	RecommendationDigest    string                 `json:"recommendationDigest,omitempty"`
	CurrentLeafCount        int                    `json:"currentLeafCount"`
	IncludedCount           int                    `json:"includedCount"`
	ExcludedCount           int                    `json:"excludedCount"`
	DeferredCount           int                    `json:"deferredCount"`
	UnsetCount              int                    `json:"unsetCount"`
	IncludedEligibleCount   int                    `json:"includedEligibleCount"`
	IncludedIneligibleCount int                    `json:"includedIneligibleCount"`
	IncludedBlockerCount    int                    `json:"includedBlockerCount"`
	IncludedSourceGapCount  int                    `json:"includedSourceGapCount"`
	FormDistribution        map[string]int         `json:"formDistribution"`
	DomainDistribution      map[string]int         `json:"domainDistribution"`
	TopicDistribution       map[string]int         `json:"topicDistribution"`
	InspectorChoices        []SimulationRoleChoice `json:"inspectorChoices"`
	LeadChoices             []SimulationRoleChoice `json:"leadChoices"`
	SimulationSetupDigest   string                 `json:"simulationSetupDigest"`
}

var (
	ErrQuestionBodyResolverUnavailable = errors.New("AGA sealed question body resolver is unavailable")
	ErrQuestionBodyDigestMismatch      = errors.New("AGA sealed question body digest mismatch")
	ErrQuestionBodyIdentityMismatch    = errors.New("AGA sealed question body identity mismatch")
	ErrQuestionBodyIncomplete          = errors.New("AGA sealed question body page is incomplete")
	ErrBatchPreviewNotFound            = errors.New("AGA batch preview is unavailable")
	ErrBatchPreviewExpired             = errors.New("AGA batch preview is expired")
	ErrBatchPreviewTooLarge            = errors.New("AGA batch preview exceeds the 500-item limit")
	ErrBatchPreviewConflict            = errors.New("AGA batch preview conflicts with current Draft")
	ErrIncludedQuestionIneligible      = errors.New("AGA included question is ineligible for the selected simulation scope")
	ErrCurrentObjectAmbiguous          = errors.New("AGA current synthetic object is ambiguous")
)

// QuestionBody is a transient sealed-overlay result. It is deliberately not a
// workspace storage type: callers must resolve complete immutable identities
// and the service validates the body digest before it is projected.
type QuestionBody struct {
	Identity   aga.BaseIdentity `json:"identity"`
	Text       string           `json:"text"`
	TextDigest string           `json:"textDigest"`
}

type QuestionBodyResolver interface {
	Resolve(context.Context, []aga.BaseIdentity) ([]QuestionBody, error)
}

type QuestionBodyResolverFunc func(context.Context, []aga.BaseIdentity) ([]QuestionBody, error)

func (resolver QuestionBodyResolverFunc) Resolve(ctx context.Context, identities []aga.BaseIdentity) ([]QuestionBody, error) {
	return resolver(ctx, identities)
}

// QuestionTextSearchResolver is separate from workspace metadata filtering.
// The application layer intersects the returned immutable base identities with
// workspace rows before it paginates; no cross-schema search join is allowed.
type QuestionTextSearchResolver interface {
	Search(context.Context, string) ([]aga.BaseIdentity, error)
}

type QuestionTextSearchResolverFunc func(context.Context, string) ([]aga.BaseIdentity, error)

func (resolver QuestionTextSearchResolverFunc) Search(ctx context.Context, value string) ([]aga.BaseIdentity, error) {
	return resolver(ctx, value)
}

// ClassificationReviewItem is a transport-only projection. The anonymous
// storage item remains the metadata source, while text fields are transient
// pointers so unauthorized projections omit both body and body digest.
type ClassificationReviewItem struct {
	preprod.ClassificationItem
	QuestionRef              aga.QuestionRef `json:"questionRef"`
	QuestionOrigin           string          `json:"questionOrigin"`
	IncludeEligible          bool            `json:"includeEligible"`
	IncludeEligibilityReason string          `json:"includeEligibilityReason"`
	QuestionText             *string         `json:"questionText,omitempty"`
	QuestionTextDigest       *string         `json:"questionTextDigest,omitempty"`
	TextOrigin               string          `json:"textOrigin,omitempty"`
}

func (ClassificationReviewItem) textProjection() {}

type LifecycleQuestionPageItem struct {
	QuestionKey        string                 `json:"questionKey"`
	QuestionRef        aga.QuestionRef        `json:"questionRef"`
	RootSequence       int                    `json:"rootSequence"`
	Projection         aga.ProposalProjection `json:"projection"`
	QuestionText       string                 `json:"questionText"`
	QuestionTextDigest string                 `json:"questionTextDigest"`
}

type QuestionTextPage struct {
	Items    []LifecycleQuestionPageItem `json:"items"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"pageSize"`
	NextPage *int                        `json:"nextPage,omitempty"`
}

func bodyDigest(text string) string {
	hash := sha256.Sum256([]byte(text))
	return "sha256:" + hex.EncodeToString(hash[:])
}

func composeReviewPage(ctx context.Context, resolver QuestionBodyResolver, identities []aga.BaseIdentity, _ string) ([]QuestionBody, error) {
	if resolver == nil {
		return nil, ErrQuestionBodyResolverUnavailable
	}
	if len(identities) > MaxQuestionTextPage {
		return nil, fmt.Errorf("%w: %d rows", ErrQuestionBodyIncomplete, len(identities))
	}
	for _, identity := range identities {
		if err := identity.Validate(); err != nil {
			return nil, fmt.Errorf("%w: invalid identity", ErrQuestionBodyIdentityMismatch)
		}
	}
	resolved, err := resolver.Resolve(ctx, identities)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQuestionBodyIncomplete, err)
	}
	if len(resolved) != len(identities) {
		return nil, ErrQuestionBodyIncomplete
	}
	wanted := make(map[string]aga.BaseIdentity, len(identities))
	for _, identity := range identities {
		wanted[identity.Key()] = identity
	}
	seen := make(map[string]struct{}, len(resolved))
	for _, body := range resolved {
		identity, ok := wanted[body.Identity.Key()]
		if !ok {
			return nil, ErrQuestionBodyIdentityMismatch
		}
		if _, duplicate := seen[body.Identity.Key()]; duplicate {
			return nil, ErrQuestionBodyIdentityMismatch
		}
		seen[body.Identity.Key()] = struct{}{}
		if identity != body.Identity || body.TextDigest != identity.TextDigest || bodyDigest(body.Text) != body.TextDigest {
			return nil, ErrQuestionBodyDigestMismatch
		}
	}
	if len(seen) != len(wanted) {
		return nil, ErrQuestionBodyIncomplete
	}
	sort.SliceStable(resolved, func(left, right int) bool { return resolved[left].Identity.Key() < resolved[right].Identity.Key() })
	return resolved, nil
}

func canReceiveQuestionText(principal identity.Principal, binding preprod.AuthorityBinding) bool {
	if principal.HasRole(identity.RoleAdmin) && isCAAOrganization(principal.OrganizationID) {
		return true
	}
	return principal.HasRole(identity.RoleDepartmentManager) && bindingHasWorkspaceRole(binding, principal, "MANAGER")
}

func metadataItem(item preprod.ClassificationItem) ClassificationReviewItem {
	return ClassificationReviewItem{ClassificationItem: item, IncludeEligibilityReason: "SIMULATION_SCOPE_UNAVAILABLE"}
}

func bodyIdentities(items []preprod.ClassificationItem) []aga.BaseIdentity {
	identities := make([]aga.BaseIdentity, 0, len(items))
	for _, item := range items {
		if item.Identity.FormCode == "" {
			continue
		}
		identities = append(identities, item.Identity)
	}
	return identities
}

func bodyMap(values []QuestionBody) map[string]QuestionBody {
	result := make(map[string]QuestionBody, len(values))
	for _, value := range values {
		result[value.Identity.Key()] = value
	}
	return result
}

func normalizeBodySearch(value string) (string, error) {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(value) > 256 {
		return "", ErrMalformedCommand
	}
	return value, nil
}

func validateBatchPreviewConsume(value BatchPreviewConsume) error {
	if !strings.HasPrefix(value.PreviewID, "aga-ws-preview-") || len(value.PreviewDigest) != len("sha256:")+64 || !strings.HasPrefix(value.PreviewDigest, "sha256:") {
		return ErrBatchPreviewNotFound
	}
	return nil
}
