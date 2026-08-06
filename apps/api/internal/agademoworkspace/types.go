package agademoworkspace

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

// Operation families are deliberately explicit. A route may accept only the
// operations assigned to its family; the service never infers a family from a
// client-supplied object identifier.
type OperationFamily string

const (
	FamilyClassificationQuery   OperationFamily = "classification-query"
	FamilyClassificationCommand OperationFamily = "classification-command"
	FamilyRecommendationQuery   OperationFamily = "recommendation-query"
	FamilyRecommendationCommand OperationFamily = "recommendation-command"
	FamilyLifecycleQuery        OperationFamily = "lifecycle-query"
	FamilyLifecycleCommand      OperationFamily = "lifecycle-command"
	FamilyAdminCommand          OperationFamily = "admin-command"
)

const (
	OperationGetSummary                = "GET_SUMMARY"
	OperationGetTaxonomy               = "GET_TAXONOMY"
	OperationGetProviderConfiguration  = "GET_PROVIDER_CONFIGURATION"
	OperationSearchItems               = "SEARCH_ITEMS"
	OperationGetDraft                  = "GET_DRAFT"
	OperationGetHistory                = "GET_HISTORY"
	OperationGetSimulationSetup        = "GET_SIMULATION_SETUP"
	OperationGetCurrentRecommendation  = "GET_CURRENT_RECOMMENDATION"
	OperationGetCurrentInspection      = "GET_CURRENT_INSPECTION"
	OperationGetInspectionQuestionPage = "GET_INSPECTION_QUESTION_PAGE"

	OperationPreviewBatch = "PREVIEW_BATCH"
	OperationExecuteBatch = "EXECUTE_BATCH"
	OperationRetain       = "RETAIN"
	OperationReclassify   = "RECLASSIFY_MAIN_DOMAIN"
	OperationAddTopic     = "ADD_TOPIC"
	OperationRemoveTopic  = "REMOVE_TOPIC"
	OperationResolve      = "RESOLVE_CLASSIFICATION_PROPOSALS"
	OperationInclude      = "INCLUDE"
	OperationExclude      = "EXCLUDE"
	OperationDefer        = "DEFER"
	OperationAddCandidate = "ADD_CANDIDATE"
	OperationReword       = "REWORD_CANDIDATE"
	OperationMarkReady    = "MARK_READY_FOR_DEMO_SIMULATION"

	OperationCreateRecommendation = "CREATE_RECOMMENDATION"
	OperationCreateInspection     = "CREATE_INSPECTION"

	OperationGetInspection   = "GET_INSPECTION"
	OperationGetFinding      = "GET_FINDING"
	OperationGetCAPEvidence  = "GET_CAP_EVIDENCE"
	OperationGetRoleHistory  = "GET_ROLE_HISTORY"
	OperationStartInspection = "START_INSPECTION"
	OperationRecordResponse  = "RECORD_RESPONSE"
	OperationCreateFinding   = "CREATE_POTENTIAL_FINDING"
	OperationSubmitChecklist = "SUBMIT_CHECKLIST"
	OperationReopenChecklist = "REOPEN_CHECKLIST"
	OperationReturnFinding   = "RETURN_POTENTIAL_FINDING"
	OperationDismissFinding  = "DISMISS_POTENTIAL_FINDING"
	OperationConvertFinding  = "CONVERT_POTENTIAL_FINDING"
	OperationSubmitCAP       = "SUBMIT_CAP_REVISION"
	OperationReviewCAP       = "REVIEW_CAP"
	OperationSubmitEvidence  = "SUBMIT_EVIDENCE_VERSION"
	OperationVerifyEvidence  = "VERIFY_EVIDENCE"
	OperationAuthorizedClose = "AUTHORIZED_CLOSE"
	OperationResetGeneration = "RESET_GENERATION"

	// MaxWorkspacePage is deliberately finite even though individual payloads
	// are capped at 25 rows.  Keeping page arithmetic in a small closed range
	// prevents an untrusted page value from overflowing before a slice bound is
	// checked.
	MaxWorkspacePage = 10000
)

var (
	ErrNeutralDenied                  = errors.New("AGA demo workspace neutral denial")
	ErrMalformedCommand               = errors.New("AGA demo workspace command is malformed")
	ErrCommandConflict                = errors.New("AGA demo workspace command conflicts with current state")
	ErrCapabilityUnavailable          = errors.New("AGA demo workspace capability is unavailable")
	ErrWorkspaceStore                 = errors.New("AGA demo workspace store is unavailable")
	ErrRecommendationFactsUnavailable = errors.New("AGA demo recommendation facts are unavailable")
	ErrRecommendationAmbiguous        = errors.New("AGA demo recommendation facts are ambiguous")
)

// BindingResolver is the only authority input accepted by the workspace
// service. Implementations resolve a current workspace-only binding from the
// authenticated subject and membership; they must not consult canonical
// business-domain assignment authority.
type BindingResolver interface {
	Resolve(context.Context, identity.Principal) (preprod.AuthorityBinding, bool, error)
}

// OperationBindingResolver is the narrow authority seam used by operations
// that need one exact sealed membership row.  A resolver must never combine
// roles or scope values from separate rows to manufacture a broader binding.
type OperationBindingResolver interface {
	ResolveForOperation(context.Context, identity.Principal, string) (preprod.AuthorityBinding, bool, error)
}

type BindingResolverFunc func(context.Context, identity.Principal) (preprod.AuthorityBinding, bool, error)

func (resolver BindingResolverFunc) Resolve(ctx context.Context, principal identity.Principal) (preprod.AuthorityBinding, bool, error) {
	return resolver(ctx, principal)
}

// RecommendationScopeResolver is the server-side fact boundary. The
// browser supplies only exact identifiers; this resolver returns the one
// immutable synthetic scope fact that owns the provider type, target, interval
// and qualifier maps.
type RecommendationScopeResolver func(context.Context, preprod.LoadedWorkspace, aga.RecommendationRequest) ([]aga.ProviderScopeFact, error)

// SimulationSetupResolver is the server-owned fact boundary for the browser
// package builder. It returns a comparison projection only; it does not mint
// readiness state or mutate a workspace.
type SimulationSetupResolver func(context.Context, preprod.LoadedWorkspace, identity.Principal) (SimulationSetupProjection, error)

// StaticBindingResolver is useful for service doubles and the local tagged
// runtime. It returns only the binding keyed by the authenticated subject.
type StaticBindingResolver struct {
	Bindings map[string]preprod.AuthorityBinding
}

func (resolver StaticBindingResolver) Resolve(_ context.Context, principal identity.Principal) (preprod.AuthorityBinding, bool, error) {
	binding, ok := resolver.Bindings[principal.SubjectID]
	if !ok {
		return preprod.AuthorityBinding{}, false, nil
	}
	return binding, true, nil
}

type ServiceConfig struct {
	// Store is retained as the command fallback for small local doubles. The
	// tagged runtime supplies separate reader and command stores.
	Store                preprod.Store
	ReaderStore          preprod.Store
	CommandStore         preprod.Store
	Resolver             BindingResolver
	QuestionBodies       QuestionBodyResolver
	QuestionTextSearch   QuestionTextSearchResolver
	RecommendationScopes RecommendationScopeResolver
	SimulationSetup      SimulationSetupResolver
	LifecycleBindings    LifecycleBindingResolver
	Clock                func() time.Time
}

type Service struct {
	store                   preprod.Store
	reader                  preprod.Store
	command                 preprod.Store
	resolver                BindingResolver
	questionBodies          QuestionBodyResolver
	questionTextSearch      QuestionTextSearchResolver
	recommendationScopes    RecommendationScopeResolver
	simulationSetupResolver SimulationSetupResolver
	lifecycleBindings       LifecycleBindingResolver
	clock                   func() time.Time
}

type Capability struct {
	Available             bool   `json:"available"`
	Projection            string `json:"projection"`
	ClassificationEnabled bool   `json:"classificationEnabled"`
	RecommendationEnabled bool   `json:"recommendationEnabled"`
	LifecycleEnabled      bool   `json:"lifecycleEnabled"`
	ResetEnabled          bool   `json:"resetEnabled"`
}

type QueryRequest struct {
	OperationID         string `json:"operationId"`
	Search              string `json:"search,omitempty"`
	Page                int    `json:"page,omitempty"`
	PageSize            int    `json:"pageSize,omitempty"`
	DomainCode          string `json:"domainCode,omitempty"`
	TopicCode           string `json:"topicCode,omitempty"`
	Confidence          string `json:"confidence,omitempty"`
	Blocker             string `json:"blocker,omitempty"`
	SourceGap           string `json:"sourceGap,omitempty"`
	ExternalInvolvement string `json:"externalInvolvement,omitempty"`
	FormCode            string `json:"formCode,omitempty"`
	Disposition         string `json:"disposition,omitempty"`
	InspectionID        string `json:"inspectionId,omitempty"`
	FindingID           string `json:"findingId,omitempty"`
	CapID               string `json:"capId,omitempty"`
	EvidenceID          string `json:"evidenceId,omitempty"`
}

type CommandEnvelope struct {
	OperationID                    string                  `json:"operationId"`
	IdempotencyKey                 string                  `json:"idempotencyKey"`
	ExpectedGenerationID           string                  `json:"expectedGenerationId"`
	ExpectedDraftRevision          int                     `json:"expectedDraftRevision,omitempty"`
	ExpectedDraftContentDigest     string                  `json:"expectedDraftContentDigest,omitempty"`
	ExpectedRecommendationRevision int                     `json:"expectedRecommendationRevision,omitempty"`
	ExpectedLifecycleRevision      int                     `json:"expectedLifecycleRevision,omitempty"`
	ExpectedLifecycleDigest        string                  `json:"expectedLifecycleDigest,omitempty"`
	ExpectedGenerationRevision     int                     `json:"expectedGenerationRevision,omitempty"`
	ExpectedGenerationSealDigest   string                  `json:"expectedGenerationSealDigest,omitempty"`
	OrganizationID                 string                  `json:"organizationId,omitempty"`
	ProviderScopeRootID            string                  `json:"providerScopeRootId,omitempty"`
	ProviderScopeID                string                  `json:"providerScopeId,omitempty"`
	ProviderScopeVersion           int                     `json:"providerScopeVersion,omitempty"`
	ProviderTypeID                 string                  `json:"providerTypeId,omitempty"`
	DepartmentID                   string                  `json:"departmentId,omitempty"`
	OrganizationalUnitID           string                  `json:"organizationalUnitId,omitempty"`
	TargetID                       string                  `json:"targetId,omitempty"`
	CanonicalTargetKind            string                  `json:"canonicalTargetKind,omitempty"`
	TargetProfileCode              string                  `json:"targetProfileCode,omitempty"`
	InspectionProfileCode          string                  `json:"inspectionProfileCode,omitempty"`
	InspectionTypeCode             string                  `json:"inspectionTypeCode,omitempty"`
	OperationQualifiers            []aga.Qualifier         `json:"operationQualifiers,omitempty"`
	ActivityQualifiers             []aga.Qualifier         `json:"activityQualifiers,omitempty"`
	EffectiveAt                    time.Time               `json:"effectiveAt,omitempty"`
	TaxonomyVersion                string                  `json:"taxonomyVersion,omitempty"`
	TaxonomyDigest                 string                  `json:"taxonomyDigest,omitempty"`
	ClassificationRunID            string                  `json:"classificationRunId,omitempty"`
	ClassificationRunDigest        string                  `json:"classificationRunDigest,omitempty"`
	DraftID                        string                  `json:"draftId,omitempty"`
	DraftRevision                  int                     `json:"draftRevision,omitempty"`
	DraftContentDigest             string                  `json:"draftContentDigest,omitempty"`
	ReadinessEventID               string                  `json:"readinessEventId,omitempty"`
	ReadinessEventDigest           string                  `json:"readinessEventDigest,omitempty"`
	ProviderScopeProfileDigest     string                  `json:"providerScopeProfileDigest,omitempty"`
	RecommendationID               string                  `json:"recommendationId,omitempty"`
	RecommendationDigest           string                  `json:"recommendationDigest,omitempty"`
	InspectionID                   string                  `json:"inspectionId,omitempty"`
	InspectorBindingID             string                  `json:"inspectorBindingId,omitempty"`
	InspectorBindingRevision       int                     `json:"inspectorBindingRevision,omitempty"`
	LeadBindingID                  string                  `json:"leadBindingId,omitempty"`
	LeadBindingRevision            int                     `json:"leadBindingRevision,omitempty"`
	PotentialFindingID             string                  `json:"potentialFindingId,omitempty"`
	PotentialFindingRootID         string                  `json:"potentialFindingRootId,omitempty"`
	FindingID                      string                  `json:"findingId,omitempty"`
	CapID                          string                  `json:"capId,omitempty"`
	EvidenceID                     string                  `json:"evidenceId,omitempty"`
	Answer                         string                  `json:"answer,omitempty"`
	CommentToAuditee               string                  `json:"commentToAuditee,omitempty"`
	InternalCAANote                string                  `json:"internalCaaNote,omitempty"`
	EvidenceFileName               string                  `json:"evidenceFileName,omitempty"`
	Severity                       string                  `json:"severity,omitempty"`
	CapRequired                    bool                    `json:"capRequired,omitempty"`
	EvidenceRequired               bool                    `json:"evidenceRequired,omitempty"`
	DueDateRequired                bool                    `json:"dueDateRequired,omitempty"`
	DueDate                        *time.Time              `json:"dueDate,omitempty"`
	RootCause                      string                  `json:"rootCause,omitempty"`
	CorrectiveAction               string                  `json:"correctiveAction,omitempty"`
	PreventiveAction               string                  `json:"preventiveAction,omitempty"`
	ResponsiblePerson              string                  `json:"responsiblePerson,omitempty"`
	Outcome                        string                  `json:"outcome,omitempty"`
	Action                         aga.DraftAction         `json:"action,omitempty"`
	TargetQuestionKey              string                  `json:"targetQuestionKey,omitempty"`
	ReasonCode                     string                  `json:"reasonCode,omitempty"`
	ReasonExplanation              string                  `json:"reasonExplanation,omitempty"`
	MainDomainCode                 string                  `json:"mainDomainCode,omitempty"`
	TopicCode                      string                  `json:"topicCode,omitempty"`
	ResolutionMode                 aga.ResolutionMode      `json:"resolutionMode,omitempty"`
	ExactProjection                *aga.ProposalProjection `json:"exactProjection,omitempty"`
	WorkspaceBody                  string                  `json:"workspaceBody,omitempty"`
	WorkspaceBodyDigest            string                  `json:"workspaceBodyDigest,omitempty"`
	PreviewID                      string                  `json:"previewId,omitempty"`
	PreviewDigest                  string                  `json:"previewDigest,omitempty"`
	BatchFilter                    *BatchFilter            `json:"batchFilter,omitempty"`
	BatchAction                    BatchAction             `json:"batchAction,omitempty"`
	SetupDigest                    string                  `json:"simulationSetupDigest,omitempty"`
	InspectorSelectionPin          string                  `json:"inspectorSelectionPin,omitempty"`
	LeadSelectionPin               string                  `json:"leadSelectionPin,omitempty"`
	LifecyclePayload               json.RawMessage         `json:"lifecyclePayload,omitempty"`
}

type QueryResponse struct {
	Operation              string                               `json:"operation"`
	Generation             preprod.Generation                   `json:"generation"`
	Draft                  *aga.Draft                           `json:"draft,omitempty"`
	Items                  []ClassificationReviewItem           `json:"items,omitempty"`
	ReviewItems            []ClassificationReviewItem           `json:"reviewItems,omitempty"`
	ItemCount              int                                  `json:"itemCount,omitempty"`
	Page                   int                                  `json:"page,omitempty"`
	PageSize               int                                  `json:"pageSize,omitempty"`
	NextPage               *int                                 `json:"nextPage,omitempty"`
	BaseQuestionCount      int                                  `json:"baseQuestionCount,omitempty"`
	DraftIncludedCount     int                                  `json:"draftIncludedCount,omitempty"`
	DraftExcludedCount     int                                  `json:"draftExcludedCount,omitempty"`
	DraftDeferredCount     int                                  `json:"draftDeferredCount,omitempty"`
	Taxonomy               *preprod.TaxonomyVersion             `json:"taxonomy,omitempty"`
	ProviderConfiguration  []preprod.ProviderConfigurationEntry `json:"providerConfiguration,omitempty"`
	History                []preprod.Generation                 `json:"history,omitempty"`
	LifecycleAvailable     bool                                 `json:"lifecycleAvailable"`
	Lifecycle              *LifecycleProjection                 `json:"lifecycle,omitempty"`
	LifecycleCAA           *LifecycleCAAProjection              `json:"lifecycleCaa,omitempty"`
	LifecycleAuditee       *LifecycleAuditeeProjection          `json:"lifecycleAuditee,omitempty"`
	QuestionPage           *QuestionTextPage                    `json:"questionPage,omitempty"`
	SimulationSetup        *SimulationSetupProjection           `json:"simulationSetup,omitempty"`
	RecommendationSnapshot *preprod.RecommendationSnapshot      `json:"recommendationSnapshot,omitempty"`
	CurrentInspection      *LifecycleProjection                 `json:"currentInspection,omitempty"`
	BatchPreview           *BatchPreviewProjection              `json:"batchPreview,omitempty"`
}

type CommandResponse struct {
	OperationID        string                          `json:"operationId"`
	Replayed           bool                            `json:"replayed"`
	Generation         *preprod.Generation             `json:"generation,omitempty"`
	Draft              *aga.Draft                      `json:"draft,omitempty"`
	ResetTombstone     *preprod.ResetTombstone         `json:"resetTombstone,omitempty"`
	Recommendation     *preprod.RecommendationSnapshot `json:"recommendation,omitempty"`
	BatchPreview       *BatchPreviewProjection         `json:"batchPreview,omitempty"`
	Lifecycle          *LifecycleProjection            `json:"lifecycle,omitempty"`
	LifecycleAuditee   *LifecycleAuditeeProjection     `json:"lifecycleAuditee,omitempty"`
	LifecycleAvailable bool                            `json:"lifecycleAvailable"`
	Reason             string                          `json:"reason,omitempty"`
}

type AuthorizationDecision struct {
	Allowed      bool
	IsAdmin      bool
	Binding      preprod.AuthorityBinding
	ScopeDigest  string
	Operation    string
	Organization string
}

func (request QueryRequest) Validate() error {
	if request.OperationID == "" || request.Page < 0 || request.Page > MaxWorkspacePage || request.PageSize < 0 || request.PageSize > 1000 {
		return ErrMalformedCommand
	}
	if request.PageSize > MaxQuestionTextPage && (request.OperationID == OperationSearchItems || request.OperationID == OperationGetInspectionQuestionPage) {
		return ErrMalformedCommand
	}
	return nil
}

func (command CommandEnvelope) Validate(family OperationFamily) error {
	if command.OperationID == "" || command.IdempotencyKey == "" || command.ExpectedGenerationID == "" {
		return ErrMalformedCommand
	}
	if family == FamilyLifecycleCommand && (command.ExpectedLifecycleRevision < 1 || command.ExpectedLifecycleDigest == "") {
		return ErrMalformedCommand
	}
	if family == FamilyAdminCommand && (command.ExpectedGenerationRevision < 1 || command.ExpectedGenerationSealDigest == "") {
		return ErrMalformedCommand
	}
	if family == FamilyClassificationCommand && command.ExpectedDraftRevision < 1 {
		return ErrMalformedCommand
	}
	if command.OperationID == OperationMarkReady && (command.ReadinessEventID != "" || command.SetupDigest == "") {
		return ErrMalformedCommand
	}
	if family == FamilyRecommendationCommand {
		switch command.OperationID {
		case OperationCreateRecommendation:
			if command.DraftRevision < 1 || command.ExpectedDraftRevision < 1 || command.DraftRevision != command.ExpectedDraftRevision || command.DraftID == "" || command.DraftContentDigest == "" {
				return ErrMalformedCommand
			}
		case OperationCreateInspection:
			if command.SetupDigest != "" {
				if command.InspectorSelectionPin == "" || command.LeadSelectionPin == "" {
					return ErrMalformedCommand
				}
				break
			}
			if command.ExpectedRecommendationRevision < 1 || command.RecommendationID == "" || command.RecommendationDigest == "" {
				return ErrMalformedCommand
			}
		default:
			return ErrMalformedCommand
		}
	}
	return nil
}
