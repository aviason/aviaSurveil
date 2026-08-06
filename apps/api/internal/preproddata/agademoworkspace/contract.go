// Package agademoworkspace owns the disposable, synthetic AGA demo data plane.
//
// The package is deliberately separate from the canonical application data
// stores.  It contains the contracts used by the one-shot loader and by the
// later service layer, but it does not expose a canonical-domain writer.
package agademoworkspace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
)

const (
	WorkspaceSchemaName   = "preprod_aga_demo_workspace"
	WorkspaceOwnerRole    = "preprod_aga_demo_workspace_owner"
	WorkspaceExporterRole = "preprod_aga_demo_workspace_fixture_exporter"
	WorkspaceLoaderRole   = "preprod_aga_demo_workspace_loader"
	WorkspaceReaderRole   = "preprod_aga_demo_workspace_reader"
	WorkspaceCommandRole  = "preprod_aga_demo_workspace_command"

	WorkspaceSchemaVersion = "aga-demo-workspace/v1"
	FixtureSchemaVersion   = "aga-demo-workspace-authority-fixture/v1"
	WorkspaceSealState     = "SEALED"
	GenerationActive       = "ACTIVE"
	GenerationReset        = "RESET"

	LoadOperation = "LOAD_AGA_DEMO_WORKSPACE"
	SealOperation = "SEAL_AGA_DEMO_WORKSPACE"

	ProviderScopeStatusActive = "ACTIVE"
	ProviderScopeStatusEnded  = "ENDED"

	QuestionVersionAdd    = "ADD_CANDIDATE"
	QuestionVersionReword = "REWORD_CANDIDATE"
)

var (
	ErrWorkspaceContract      = errors.New("AGA demo workspace contract is invalid")
	ErrWorkspaceSealed        = errors.New("AGA demo workspace is sealed")
	ErrWorkspaceNotSealed     = errors.New("AGA demo workspace is not sealed")
	ErrWorkspaceGeneration    = errors.New("AGA demo workspace generation is invalid")
	ErrWorkspaceAppendOnly    = errors.New("AGA demo workspace append-only invariant failed")
	ErrWorkspaceIdempotency   = errors.New("AGA demo workspace idempotency conflict")
	ErrWorkspaceCAS           = errors.New("AGA demo workspace compare-and-swap conflict")
	ErrWorkspaceFixture       = errors.New("AGA demo workspace authority fixture is invalid")
	ErrWorkspaceInput         = errors.New("AGA demo workspace loader input is invalid")
	ErrWorkspaceLoaderRevoked = errors.New("AGA demo workspace loader credential is revoked")
)

var (
	workspaceIDPattern = regexp.MustCompile(`^aga-ws-[a-z0-9][a-z0-9-]{7,63}$`)
	digestPattern      = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
	rolePattern        = regexp.MustCompile(`^preprod_aga_demo_workspace_[a-z_]+$`)
)

// RoleGrant is the closed role matrix used by provisioning and static tests.
// Runtime credentials are never accepted from a normal application settings
// object; they are mounted only into the service named by the role.
type RoleGrant struct {
	Role              string   `json:"role"`
	Login             bool     `json:"login"`
	Inherit           bool     `json:"inherit"`
	SchemaUsage       bool     `json:"schemaUsage"`
	SelectProjections []string `json:"selectProjections"`
	ExecuteFunctions  []string `json:"executeFunctions"`
	DirectTableDML    bool     `json:"directTableDml"`
	CanonicalAccess   bool     `json:"canonicalAccess"`
	OverlayAccess     bool     `json:"overlayAccess"`
	RuntimeCredential bool     `json:"runtimeCredential"`
}

func WorkspaceRoleMatrix() []RoleGrant {
	return []RoleGrant{
		{Role: WorkspaceOwnerRole, Login: false, Inherit: false, SchemaUsage: true, RuntimeCredential: false},
		{Role: WorkspaceExporterRole, Login: true, Inherit: false, SelectProjections: []string{"predecessor_fixture_identity_columns"}, RuntimeCredential: false, CanonicalAccess: false, OverlayAccess: false},
		{Role: WorkspaceLoaderRole, Login: true, Inherit: false, SchemaUsage: true, SelectProjections: []string{"loader_projections"}, ExecuteFunctions: []string{"workspace_load"}, RuntimeCredential: false, CanonicalAccess: false, OverlayAccess: false},
		{Role: WorkspaceReaderRole, Login: true, Inherit: false, SchemaUsage: true, SelectProjections: []string{"sealed_workspace_projections"}, RuntimeCredential: true, CanonicalAccess: false, OverlayAccess: false},
		{Role: WorkspaceCommandRole, Login: true, Inherit: false, SchemaUsage: true, ExecuteFunctions: []string{"workspace_query", "workspace_command", "workspace_reset"}, RuntimeCredential: true, DirectTableDML: false, CanonicalAccess: false, OverlayAccess: false},
	}
}

func ValidateWorkspaceRoleMatrix(matrix []RoleGrant) error {
	if len(matrix) != 5 {
		return fmt.Errorf("%w: role count", ErrWorkspaceContract)
	}
	seen := make(map[string]bool, len(matrix))
	for _, grant := range matrix {
		if !rolePattern.MatchString(grant.Role) || seen[grant.Role] {
			return fmt.Errorf("%w: role identity", ErrWorkspaceContract)
		}
		seen[grant.Role] = true
		if grant.Inherit || grant.CanonicalAccess || grant.OverlayAccess || grant.DirectTableDML {
			return fmt.Errorf("%w: broad grant", ErrWorkspaceContract)
		}
		if grant.Role == WorkspaceOwnerRole && grant.Login {
			return fmt.Errorf("%w: owner login", ErrWorkspaceContract)
		}
		if grant.Role == WorkspaceCommandRole && len(grant.ExecuteFunctions) == 0 {
			return fmt.Errorf("%w: command functions", ErrWorkspaceContract)
		}
		if grant.Role == WorkspaceReaderRole && len(grant.SelectProjections) == 0 {
			return fmt.Errorf("%w: reader projections", ErrWorkspaceContract)
		}
	}
	for _, role := range []string{WorkspaceOwnerRole, WorkspaceExporterRole, WorkspaceLoaderRole, WorkspaceReaderRole, WorkspaceCommandRole} {
		if !seen[role] {
			return fmt.Errorf("%w: missing role", ErrWorkspaceContract)
		}
	}
	return nil
}

type Generation struct {
	GenerationID            string    `json:"generationId"`
	State                   string    `json:"state"`
	ClassificationRunID     string    `json:"classificationRunId"`
	ClassificationRunDigest string    `json:"classificationRunDigest"`
	TaxonomyVersion         string    `json:"taxonomyVersion"`
	TaxonomyDigest          string    `json:"taxonomyDigest"`
	FixtureDigest           string    `json:"fixtureDigest"`
	Revision                int       `json:"revision"`
	SealDigest              string    `json:"sealDigest"`
	CreatedAt               time.Time `json:"createdAt"`
	ResetFromGenerationID   string    `json:"resetFromGenerationId,omitempty"`
}

func (generation Generation) Validate() error {
	if !workspaceIDPattern.MatchString(generation.GenerationID) || generation.State != GenerationActive && generation.State != GenerationReset || generation.Revision < 1 || generation.CreatedAt.IsZero() {
		return fmt.Errorf("%w: generation", ErrWorkspaceGeneration)
	}
	if !validDigest(generation.ClassificationRunDigest) || !validDigest(generation.TaxonomyDigest) || !validDigest(generation.FixtureDigest) || !validDigest(generation.SealDigest) {
		return fmt.Errorf("%w: generation digests", ErrWorkspaceGeneration)
	}
	return nil
}

type ResetTombstone struct {
	TombstoneID          string    `json:"tombstoneId"`
	FromGenerationID     string    `json:"fromGenerationId"`
	ToGenerationID       string    `json:"toGenerationId"`
	ExpectedGenerationID string    `json:"expectedGenerationId"`
	ExpectedRevision     int       `json:"expectedGenerationRevision"`
	ExpectedSealDigest   string    `json:"expectedGenerationSealDigest"`
	ReasonCode           string    `json:"reasonCode"`
	ActorSubjectID       string    `json:"actorSubjectId"`
	CreatedAt            time.Time `json:"createdAt"`
	TombstoneDigest      string    `json:"tombstoneDigest"`
}

type TaxonomyVersion struct {
	Version       string    `json:"taxonomyVersion"`
	Digest        string    `json:"taxonomyDigest"`
	PackageDigest string    `json:"packageDigest"`
	PublishedAt   time.Time `json:"publishedAt"`
	Sealed        bool      `json:"sealed"`
}

type ClassificationRun struct {
	RunID                   string                   `json:"classificationRunId"`
	State                   string                   `json:"state"`
	TaxonomyVersion         string                   `json:"taxonomyVersion"`
	TaxonomyDigest          string                   `json:"taxonomyDigest"`
	InputDigest             string                   `json:"inputDigest"`
	AggregateDigest         string                   `json:"aggregateDigest"`
	ClassificationRunDigest string                   `json:"classificationRunDigest"`
	Result                  aga.ClassificationResult `json:"result"`
	CandidateRecordCount    int                      `json:"candidateRecordCount"`
	ChallengeRecordCount    int                      `json:"challengeRecordCount"`
	ItemCount               int                      `json:"itemCount"`
	CreatedAt               time.Time                `json:"createdAt"`
}

type ClassificationPassRecord struct {
	Identity              aga.BaseIdentity         `json:"identity"`
	RunID                 string                   `json:"classificationRunId"`
	PassRole              aga.PassRole             `json:"passRole"`
	PassRunID             string                   `json:"passRunId"`
	PassResultDigest      string                   `json:"passResultDigest"`
	PromptDigest          string                   `json:"promptDigest"`
	ModelDescriptorDigest string                   `json:"modelDescriptorDigest"`
	InputDigest           string                   `json:"inputDigest"`
	ProposalProjection    aga.ProposalProjection   `json:"proposalProjection"`
	RationaleCodes        []string                 `json:"rationaleCodes"`
	ConfidenceEvidence    []aga.ConfidenceEvidence `json:"confidenceEvidence"`
	SourceRefs            []aga.SourceReference    `json:"sourceRefs"`
}

type ClassificationItem struct {
	Identity                 aga.BaseIdentity       `json:"identity"`
	QuestionKey              string                 `json:"questionKey"`
	Projection               aga.ProposalProjection `json:"projection"`
	AgreementConfidence      aga.Confidence         `json:"agreementConfidence"`
	RecommendationState      string                 `json:"recommendationState"`
	Governance               aga.GovernanceState    `json:"governance"`
	ItemSemanticDigest       string                 `json:"itemSemanticDigest"`
	CandidateDigest          string                 `json:"candidatePassResultDigest"`
	ChallengeDigest          string                 `json:"challengePassResultDigest"`
	DraftAgreementConfidence *aga.Confidence        `json:"draftAgreementConfidence"`
	DraftRecommendationState string                 `json:"draftRecommendationState"`
	DraftReviewState         string                 `json:"draftReviewState"`
	DraftDisposition         *string                `json:"draftDisposition"`
}

type DraftRecord struct {
	Draft     aga.Draft `json:"draft"`
	CreatedAt time.Time `json:"createdAt"`
}

type WorkspaceQuestionVersion struct {
	GenerationID      string                 `json:"generationId"`
	RootID            string                 `json:"questionRootId"`
	VersionID         string                 `json:"questionVersionId"`
	ProposalID        string                 `json:"proposalId"`
	RootSequence      int                    `json:"rootSequence"`
	BodyDigest        string                 `json:"bodyDigest"`
	Body              string                 `json:"body,omitempty"`
	ParentQuestionKey *aga.ParentQuestionKey `json:"parentQuestionKey"`
	ActorSubjectID    string                 `json:"createdBySubjectId"`
	CreatedAt         time.Time              `json:"createdAt"`
	ReasonCode        string                 `json:"reasonCode"`
	CurrentLeaf       bool                   `json:"currentLeaf"`
}

func (version WorkspaceQuestionVersion) Reference() aga.QuestionRef {
	return aga.WorkspaceQuestionReference(aga.WorkspaceQuestionRef{
		GenerationID: version.GenerationID, RootID: version.RootID, VersionID: version.VersionID,
		ProposalID: version.ProposalID, RootSequence: version.RootSequence, BodyDigest: version.BodyDigest,
		ParentQuestionKey: version.ParentQuestionKey, ActorSubjectID: version.ActorSubjectID,
		CreatedAt: version.CreatedAt.UTC(), ReasonCode: version.ReasonCode,
	})
}

type ManagerDecision struct {
	DecisionID     string    `json:"decisionId"`
	GenerationID   string    `json:"generationId"`
	DraftID        string    `json:"draftId"`
	DraftRevision  int       `json:"draftRevision"`
	QuestionKey    string    `json:"questionKey"`
	Action         string    `json:"action"`
	Disposition    string    `json:"disposition,omitempty"`
	ReasonCode     string    `json:"reasonCode"`
	ActorSubjectID string    `json:"actorSubjectId"`
	CreatedAt      time.Time `json:"createdAt"`
}

type BatchPreview struct {
	Preview aga.DraftBatchPreview `json:"preview"`
	Action  aga.DraftBatchAction  `json:"action"`
	Filter  aga.DraftBatchFilter  `json:"filter"`
}

// SelectionBatchPreviewRecord is the append-only storage representation for
// the successor manager batch flow. It contains immutable question
// identities and digests only; the original question body is never copied
// into the workspace batch artifact.
type SelectionBatchPreviewItem struct {
	QuestionKey    string `json:"questionKey"`
	IdentityDigest string `json:"identityDigest"`
}

type SelectionBatchPreviewRecord struct {
	PreviewID               string                      `json:"previewId"`
	GenerationID            string                      `json:"generationId"`
	DraftID                 string                      `json:"draftId"`
	DraftRevision           int                         `json:"draftRevision"`
	DraftContentDigest      string                      `json:"draftContentDigest"`
	ClassificationRunDigest string                      `json:"classificationRunDigest"`
	FilterJSON              string                      `json:"filterJson"`
	FilterDigest            string                      `json:"filterDigest"`
	AffectedIdentityDigest  string                      `json:"affectedIdentityDigest"`
	Action                  string                      `json:"action"`
	ReasonCode              string                      `json:"reasonCode"`
	Items                   []SelectionBatchPreviewItem `json:"items"`
	PreviewDigest           string                      `json:"previewDigest"`
	ExpiresAt               time.Time                   `json:"expiresAt"`
	ConsumedAt              *time.Time                  `json:"consumedAt,omitempty"`
}

type AuthorityBinding struct {
	BindingID            string   `json:"bindingId"`
	SubjectSlot          string   `json:"subjectSlot"`
	MembershipSlot       string   `json:"membershipSlot"`
	OrganizationID       string   `json:"organizationId"`
	DepartmentID         string   `json:"departmentId"`
	OrganizationalUnitID string   `json:"organizationalUnitId"`
	OperationRoles       []string `json:"operationRoles"`
	BindingDigest        string   `json:"bindingDigest"`
	Active               bool     `json:"active"`
}

type ProviderTarget struct {
	TargetID      string `json:"targetId"`
	CanonicalKind string `json:"canonicalTargetKind"`
	ProfileCode   string `json:"targetProfileCode"`
}

type ProviderScope struct {
	GenerationID         string           `json:"generationId"`
	OrganizationID       string           `json:"organizationId"`
	ProviderScopeRootID  string           `json:"providerScopeRootId"`
	ProviderScopeID      string           `json:"providerScopeId"`
	ProviderScopeVersion int              `json:"providerScopeVersion"`
	ProviderTypeID       string           `json:"providerTypeId"`
	ProviderTypeCode     string           `json:"providerTypeCode"`
	Status               string           `json:"status"`
	EffectiveFrom        time.Time        `json:"effectiveFrom"`
	EffectiveTo          *time.Time       `json:"effectiveTo,omitempty"`
	DepartmentID         string           `json:"departmentId"`
	OrganizationalUnitID string           `json:"organizationalUnitId"`
	OperationQualifiers  []aga.Qualifier  `json:"operationQualifiers"`
	ActivityQualifiers   []aga.Qualifier  `json:"activityQualifiers"`
	Targets              []ProviderTarget `json:"targets"`
	ProfileDigest        string           `json:"profileDigest"`
}

type RecommendationSnapshot struct {
	Recommendation aga.Recommendation `json:"recommendation"`
	CreatedAt      time.Time          `json:"createdAt"`
	SnapshotDigest string             `json:"snapshotDigest"`
}

type ReadinessSnapshot struct {
	Event     aga.ReadinessEvent `json:"event"`
	CreatedAt time.Time          `json:"createdAt"`
}

type LifecycleStream struct {
	LifecycleID      string    `json:"lifecycleId"`
	GenerationID     string    `json:"generationId"`
	RecommendationID string    `json:"recommendationId"`
	Revision         int       `json:"revision"`
	Digest           string    `json:"digest"`
	State            string    `json:"state"`
	CreatedAt        time.Time `json:"createdAt"`
}

type LifecycleEvent struct {
	EventID        string          `json:"eventId"`
	LifecycleID    string          `json:"lifecycleId"`
	Sequence       int             `json:"sequence"`
	OperationID    string          `json:"operationId"`
	CommandKey     string          `json:"commandKey"`
	EventType      string          `json:"eventType"`
	Payload        json.RawMessage `json:"payload"`
	ActorSubjectID string          `json:"actorSubjectId"`
	CreatedAt      time.Time       `json:"createdAt"`
	PreviousDigest string          `json:"previousDigest"`
	EventDigest    string          `json:"eventDigest"`
}

type IdempotencyResponse struct {
	GenerationID             string          `json:"generationId"`
	ActorSubjectID           string          `json:"actorSubjectId"`
	OperationID              string          `json:"operationId"`
	IdempotencyKey           string          `json:"idempotencyKey"`
	CommandHash              string          `json:"commandHash"`
	AuthorizationScopeDigest string          `json:"authorizationScopeDigest"`
	StatusCode               int             `json:"statusCode"`
	Response                 json.RawMessage `json:"response"`
	CreatedAt                time.Time       `json:"createdAt"`
}

type WorkspaceSealReceipt struct {
	GenerationID             string    `json:"generationId"`
	ClassificationRunDigest  string    `json:"classificationRunDigest"`
	FixtureDigest            string    `json:"fixtureDigest"`
	WorkspaceAggregateDigest string    `json:"workspaceAggregateDigest"`
	SealDigest               string    `json:"sealDigest"`
	SealedAt                 time.Time `json:"sealedAt"`
	LoaderRevoked            bool      `json:"loaderRevoked"`
}

type LoadedWorkspace struct {
	Generation       Generation
	Taxonomy         TaxonomyVersion
	Run              ClassificationRun
	Items            []ClassificationItem
	CandidateRecords []ClassificationPassRecord
	ChallengeRecords []ClassificationPassRecord
	Draft            DraftRecord
	Fixture          FixtureManifest
	Seal             WorkspaceSealReceipt
}

type LoadInput struct {
	GenerationID          string
	Classification        aga.ClassificationResult
	Fixture               FixtureManifest
	TaxonomyVersion       TaxonomyVersion
	ExpectedPackageDigest string
	Now                   time.Time
}

type AppendQuestionVersionInput struct {
	GenerationID      string
	RootID            string
	VersionID         string
	ProposalID        string
	RootSequence      int
	Body              string
	BodyDigest        string
	ParentQuestionKey *aga.ParentQuestionKey
	ActorSubjectID    string
	ReasonCode        string
	Now               time.Time
	Action            string
}

type ResetInput struct {
	ExpectedGenerationID         string
	ExpectedGenerationRevision   int
	ExpectedGenerationSealDigest string
	ReasonCode                   string
	ActorSubjectID               string
	Now                          time.Time
}

type StoredResponseKey struct {
	GenerationID   string
	ActorSubjectID string
	OperationID    string
	IdempotencyKey string
}

func (key StoredResponseKey) Validate() error {
	if key.GenerationID == "" || key.ActorSubjectID == "" || key.OperationID == "" || key.IdempotencyKey == "" {
		return fmt.Errorf("%w: idempotency key", ErrWorkspaceContract)
	}
	return nil
}

func validDigest(value string) bool { return digestPattern.MatchString(value) }

func digestValue(domain string, value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(append([]byte(domain+"\n"), data...))
	return "sha256:" + hex.EncodeToString(hash[:])
}

func cloneJSON[T any](value T) T {
	data, _ := json.Marshal(value)
	var copy T
	_ = json.Unmarshal(data, &copy)
	return copy
}

func normalizeCodes(values []string) []string {
	copy := append([]string(nil), values...)
	sort.Strings(copy)
	return copy
}

func exactStringSet(left, right []string) bool {
	left = normalizeCodes(left)
	right = normalizeCodes(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func trimReason(value string) string { return strings.TrimSpace(value) }
