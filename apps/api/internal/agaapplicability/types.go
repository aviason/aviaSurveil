package agaapplicability

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrDuplicateIdentity          = errors.New("duplicate complete question identity")
	ErrPassBijection              = errors.New("candidate/challenge pass identity bijection mismatch")
	ErrUnknownCode                = errors.New("unknown controlled taxonomy code")
	ErrTargetProfileMismatch      = errors.New("canonical target kind and target profile mismatch")
	ErrDuplicateProposalValue     = errors.New("duplicate set-like proposal value")
	ErrEvidenceBinding            = errors.New("confidence evidence does not bind an emitted proposal value")
	ErrEvidenceFactMismatch       = errors.New("confidence evidence fact digest is not trusted input")
	ErrUnknownInputFactSelector   = errors.New("unknown input fact selector")
	ErrUnknownSignalRule          = errors.New("unknown validator signal rule")
	ErrDigestMismatch             = errors.New("digest mismatch")
	ErrQuestionReferenceUnion     = errors.New("question reference union is not exact")
	ErrParentQuestionKey          = errors.New("parent question key is invalid")
	ErrWorkspaceIdentityAlias     = errors.New("workspace root, version, or proposal identity alias")
	ErrCrossGenerationParent      = errors.New("workspace question crosses generation")
	ErrCrossRootParent            = errors.New("workspace successor crosses root")
	ErrCyclicParent               = errors.New("workspace parent cycle")
	ErrMissingParent              = errors.New("workspace parent is missing")
	ErrNonCurrentQuestion         = errors.New("question reference is not the current leaf")
	ErrByteIdenticalReword        = errors.New("reword is byte-identical")
	ErrDraftConflict              = errors.New("draft revision or content digest conflict")
	ErrDraftNotReady              = errors.New("draft is not ready for demo simulation")
	ErrReasonRequired             = errors.New("controlled reason is required")
	ErrInvalidReason              = errors.New("reason is not allowed for action")
	ErrInvalidResolution          = errors.New("proposal resolution is incomplete or invalid")
	ErrBatchLimit                 = errors.New("draft batch exceeds 500 items")
	ErrPreviewExpired             = errors.New("draft batch preview expired")
	ErrPreviewMismatch            = errors.New("draft batch preview no longer matches")
	ErrReadinessPinMismatch       = errors.New("readiness pin mismatch")
	ErrProviderScopeMismatch      = errors.New("provider scope identity mismatch")
	ErrProviderScopeNotApplicable = errors.New("provider scope is not active at effective time")
	ErrTargetMismatch             = errors.New("typed target mismatch")
	ErrQualifierMismatch          = errors.New("operation or activity qualifiers are not exact")
	ErrNoEligibleRecommendation   = errors.New("no eligible current included question")
	ErrCommandEnvelope            = errors.New("generic command envelope is incomplete")
	ErrPrivateInputMismatch       = errors.New("private classification input mismatch")
	ErrInputBounds                = errors.New("classification input bounds invalid")
)

type PassRole string

const (
	PassCandidate PassRole = "CANDIDATE"
	PassChallenge PassRole = "CHALLENGE"
)

type Confidence string

const (
	ConfidenceHigh   Confidence = "HIGH"
	ConfidenceMedium Confidence = "MEDIUM"
	ConfidenceLow    Confidence = "LOW"
)

const (
	FrozenPackageVersion           = "AGA_ALL_FORMS_SOURCE_RISK_DRAFT_V1"
	FrozenPackageJSONSHA256        = "sha256:5ebcce2d70ee22fef4165b490cb6e4b276ad776f40dbaf12e5cea85c9da91b15"
	FrozenPromptDigest             = "sha256:2ff6f7fc5c5e337c592bc67540f3f925c852de2d74529ece9ddc46ca39d1cb84"
	FrozenBatchManifestDigest      = "sha256:dee3a0101dcfdeaef9dbb8c3f53d7e4a99de9499eaa7d82a039eb6cac077c96b"
	FrozenOrderedIdentityDigest    = "sha256:4d11d492e87619ca8e39db0dce74d85b93f7d652589067395481b5e1067aedcc"
	RecommendationAutoProposed     = "AUTO_PROPOSED_HIGH_CONFIDENCE"
	RecommendationManagerReview    = "MANAGER_REVIEW_REQUIRED"
	RecommendationBlockedSourceGap = "BLOCKED_SOURCE_GAP"
	ClassificationRunSealed        = "SEALED"
	SourceMappingRequired          = "SOURCE_MAPPING_REQUIRED"
	SourceAuthorityNotAttested     = "NOT_ATTESTED"
	RiskExpertReviewRequired       = "CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW"
	DecisionNotSupplied            = "DECISION_NOT_SUPPLIED"
	ExtractionCandidate            = "EXTRACTED_CANDIDATE"
	ExtractionExactSourceBacked    = "EXACT_SOURCE_BACKED"
	GenerationActive               = "ACTIVE"
	FrozenBaseQuestionCount        = 1310
	FrozenPassProposalRecordCount  = 2620
	FrozenSourceGapCount           = 49
	FrozenExternalUnresolvedCount  = 51
	FrozenSourceExternalOverlap    = 49
	FrozenExtractedCandidateCount  = 1282
	FrozenExactSourceBackedCount   = 28
)

var (
	classificationRunIDPattern = regexp.MustCompile(`^aga-classification-run-[a-z0-9][a-z0-9-]{0,63}$`)
	passRunIDPattern           = regexp.MustCompile(`^aga-classification-pass-(?:candidate|challenge)-[a-z0-9][a-z0-9-]{0,63}$`)
	serverIDPattern            = regexp.MustCompile(`^aga-ws-[a-z0-9][a-z0-9-]{7,63}$`)
	formCodePattern            = regexp.MustCompile(`^FSS-AGA-FORM-(?:0(?:0[1-9]|[12][0-9]|3[0-4])|035A|0(?:3[6-9]|4[0-8]|5[0-3]))$`)
	subjectIDPattern           = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
	digestPattern              = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
)

type BaseIdentity struct {
	PackageVersion    string `json:"packageVersion"`
	PackageJSONSHA256 string `json:"packageJsonSha256"`
	FormCode          string `json:"formCode"`
	ProposalID        string `json:"proposalId"`
	Ordinal           int    `json:"ordinal"`
	TextDigest        string `json:"textDigest"`
}

func (identity BaseIdentity) Key() string {
	return strings.Join([]string{
		identity.PackageVersion,
		identity.PackageJSONSHA256,
		identity.FormCode,
		identity.ProposalID,
		strconv.Itoa(identity.Ordinal),
		identity.TextDigest,
	}, "\x1f")
}

func (identity BaseIdentity) Validate() error {
	if !utf8.ValidString(identity.PackageVersion) || !utf8.ValidString(identity.PackageJSONSHA256) || !utf8.ValidString(identity.FormCode) || !utf8.ValidString(identity.ProposalID) || !utf8.ValidString(identity.TextDigest) || identity.PackageVersion != FrozenPackageVersion || identity.PackageJSONSHA256 != FrozenPackageJSONSHA256 || !formCodePattern.MatchString(identity.FormCode) || identity.ProposalID == "" || len(identity.ProposalID) > 128 || identity.Ordinal < 1 || identity.Ordinal > 256 || !validDigest(identity.TextDigest) {
		return fmt.Errorf("%w: incomplete base identity", ErrQuestionReferenceUnion)
	}
	return nil
}

type Qualifier struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SourceReference struct {
	Kind            string `json:"kind"`
	ReferenceDigest string `json:"referenceDigest"`
}

type ConfidenceEvidence struct {
	ProposalField        string `json:"proposalField"`
	ProposalValueDigest  string `json:"proposalValueDigest"`
	RationaleCode        string `json:"rationaleCode"`
	InputFactSelector    string `json:"inputFactSelector"`
	InputFactValueDigest string `json:"inputFactValueDigest"`
	SignalRuleID         string `json:"signalRuleId,omitempty"`
}

type ExternalInvolvement struct {
	ProviderTypeCode         string               `json:"providerTypeCode"`
	InvolvementRoleCode      string               `json:"involvementRoleCode"`
	ConditionCode            string               `json:"conditionCode"`
	ApplicabilityDisposition string               `json:"applicabilityDisposition"`
	RationaleCodes           []string             `json:"rationaleCodes"`
	ConfidenceEvidence       []ConfidenceEvidence `json:"confidenceEvidence"`
	SourceRefs               []SourceReference    `json:"sourceRefs"`
	BlockerCodes             []string             `json:"blockerCodes"`
}

type ProposalProjection struct {
	MainDomainCode           string                `json:"mainDomainCode"`
	TopicCodes               []string              `json:"topicCodes"`
	InspectionProfileCodes   []string              `json:"inspectionProfileCodes"`
	InspectionTypeCodes      []string              `json:"inspectionTypeCodes"`
	CanonicalTargetKind      string                `json:"canonicalTargetKind"`
	TargetProfileCode        string                `json:"targetProfileCode"`
	OperationQualifiers      []Qualifier           `json:"operationQualifiers"`
	ActivityQualifiers       []Qualifier           `json:"activityQualifiers"`
	ApplicabilityDisposition string                `json:"applicabilityDisposition"`
	EvidenceExpectationCodes []string              `json:"evidenceExpectationCodes"`
	ExternalInvolvements     []ExternalInvolvement `json:"externalInvolvements"`
}

type ProposalValueBinding struct {
	ProposalField string
	ValueDigest   string
	Core          bool
	ValueShape    string
	SemanticValue string
}

type EvidenceFact struct {
	Digest       string
	SignalRuleID string
}

type EvidenceFacts map[string][]EvidenceFact

type FixedInputDigests struct {
	PackageJSONSHA256                  string `json:"packageJsonSha256"`
	SealedOverlayLoaderZIPSHA256       string `json:"sealedOverlayLoaderZipSha256"`
	ProviderCatalogSHA256              string `json:"providerCatalogSha256"`
	ResearchZIPSHA256                  string `json:"researchZipSha256"`
	ResearchQuestionCSVSHA256          string `json:"researchQuestionCsvSha256"`
	ProviderClassificationCSVSHA256    string `json:"providerClassificationCsvSha256"`
	AmbiguityCSVSHA256                 string `json:"ambiguityCsvSha256"`
	WorkbookSHA256                     string `json:"workbookSha256"`
	AuditChecklistWorkflowSHA256       string `json:"auditChecklistWorkflowSha256"`
	FindingCAPEvidenceWorkflowSHA256   string `json:"findingCapEvidenceWorkflowSha256"`
	ProductionContractVocabularySHA256 string `json:"productionContractVocabularySha256"`
}

type ModelDescriptor struct {
	ModelID                  *string  `json:"modelId"`
	ModelIDSource            string   `json:"modelIdSource"`
	DisplayedModelLabel      *string  `json:"displayedModelLabel"`
	Service                  *string  `json:"service"`
	Interface                *string  `json:"interface"`
	RequestedReasoningEffort *string  `json:"requestedReasoningEffort"`
	ForkTurns                *string  `json:"forkTurns"`
	SnapshotBuildLabel       *string  `json:"snapshotBuildLabel"`
	UnavailableFields        []string `json:"unavailableFields"`
}

type PassProposalInput struct {
	Identity              BaseIdentity
	ClassificationRunID   string
	PassRole              PassRole
	PassRunID             string
	PromptDigest          string
	ModelDescriptorDigest string
	InputDigest           string
	Projection            ProposalProjection
	RationaleCodes        []string
	ConfidenceEvidence    []ConfidenceEvidence
	SourceRefs            []SourceReference
}

type PassProposalRecord struct {
	Identity              BaseIdentity         `json:"identity"`
	ClassificationRunID   string               `json:"classificationRunId"`
	PassRole              PassRole             `json:"passRole"`
	PassRunID             string               `json:"passRunId"`
	PromptDigest          string               `json:"promptDigest"`
	ModelDescriptorDigest string               `json:"modelDescriptorDigest"`
	InputDigest           string               `json:"inputDigest"`
	ProposalProjection    ProposalProjection   `json:"proposalProjection"`
	RationaleCodes        []string             `json:"rationaleCodes"`
	ConfidenceEvidence    []ConfidenceEvidence `json:"confidenceEvidence"`
	SourceRefs            []SourceReference    `json:"sourceRefs"`
	PassResultDigest      string               `json:"passResultDigest"`
}

func (record *PassProposalRecord) UnmarshalJSON(data []byte) error {
	if !utf8.Valid(data) || hasJSONLoneSurrogate(data) {
		return ErrQuestionReferenceUnion
	}
	object, err := closedJSONObject(data)
	if err != nil || !hasExactJSONKeys(object, "identity", "classificationRunId", "passRole", "passRunId", "promptDigest", "modelDescriptorDigest", "inputDigest", "proposalProjection", "rationaleCodes", "confidenceEvidence", "sourceRefs", "passResultDigest") {
		return ErrQuestionReferenceUnion
	}
	type passProposalRecordAlias PassProposalRecord
	var decoded passProposalRecordAlias
	if err := strictJSONDecode(data, &decoded); err != nil {
		return ErrQuestionReferenceUnion
	}
	recordValue := PassProposalRecord(decoded)
	canonical, err := json.Marshal(decoded)
	if err != nil || !semanticJSONEqual(data, canonical) {
		return ErrQuestionReferenceUnion
	}
	expected, err := NewPassProposalRecordForSuppliedProvenance(FrozenTaxonomy(), PassProposalInput{
		Identity:              recordValue.Identity,
		ClassificationRunID:   recordValue.ClassificationRunID,
		PassRole:              recordValue.PassRole,
		PassRunID:             recordValue.PassRunID,
		PromptDigest:          recordValue.PromptDigest,
		ModelDescriptorDigest: recordValue.ModelDescriptorDigest,
		InputDigest:           recordValue.InputDigest,
		Projection:            recordValue.ProposalProjection,
		RationaleCodes:        recordValue.RationaleCodes,
		ConfidenceEvidence:    recordValue.ConfidenceEvidence,
		SourceRefs:            recordValue.SourceRefs,
	})
	if err != nil || !reflect.DeepEqual(expected, recordValue) {
		return ErrQuestionReferenceUnion
	}
	*record = recordValue
	return nil
}

type GovernanceState struct {
	SourceMappingState              string   `json:"sourceMappingState"`
	SourceAuthorityState            string   `json:"sourceAuthorityState"`
	RiskClassificationState         string   `json:"riskClassificationState"`
	DecisionState                   string   `json:"decisionState"`
	ExtractionState                 string   `json:"extractionState"`
	QuestionSourceProposalGap       bool     `json:"questionSourceProposalGap"`
	ExternalApplicabilityUnresolved bool     `json:"externalApplicabilityUnresolved"`
	BlockerCodes                    []string `json:"blockerCodes"`
}

type SealedClassificationItem struct {
	Identity            BaseIdentity         `json:"identity"`
	Projection          ProposalProjection   `json:"projection"`
	AgreementConfidence Confidence           `json:"agreementConfidence"`
	RecommendationState string               `json:"recommendationState"`
	RationaleCodes      []string             `json:"rationaleCodes"`
	ConfidenceEvidence  []ConfidenceEvidence `json:"confidenceEvidence"`
	SourceRefs          []SourceReference    `json:"sourceRefs"`
	GovernanceState
	PassDisagreementCodes   []string `json:"passDisagreementCodes"`
	PassOneResultDigest     string   `json:"passOneResultDigest"`
	PassTwoResultDigest     string   `json:"passTwoResultDigest"`
	PassOneRunID            string   `json:"passOneRunId"`
	PassTwoRunID            string   `json:"passTwoRunId"`
	PromptDigest            string   `json:"promptDigest"`
	ModelDescriptorDigests  []string `json:"modelDescriptorDigests"`
	TaxonomyDigest          string   `json:"taxonomyDigest"`
	InputDigest             string   `json:"inputDigest"`
	ItemSemanticDigest      string   `json:"itemSemanticDigest"`
	ClassificationRunDigest string   `json:"classificationRunDigest"`
	AggregateDigest         string   `json:"aggregateDigest"`
}

func sealedClassificationItemObject(item SealedClassificationItem) map[string]any {
	return map[string]any{
		"packageVersion":                  item.Identity.PackageVersion,
		"packageJsonSha256":               item.Identity.PackageJSONSHA256,
		"formCode":                        item.Identity.FormCode,
		"proposalId":                      item.Identity.ProposalID,
		"ordinal":                         item.Identity.Ordinal,
		"textDigest":                      item.Identity.TextDigest,
		"taxonomyVersion":                 FrozenTaxonomy().Version,
		"mainDomainCode":                  item.Projection.MainDomainCode,
		"topicCodes":                      nonNilSlice(item.Projection.TopicCodes),
		"inspectionProfileCodes":          nonNilSlice(item.Projection.InspectionProfileCodes),
		"inspectionTypeCodes":             nonNilSlice(item.Projection.InspectionTypeCodes),
		"canonicalTargetKind":             item.Projection.CanonicalTargetKind,
		"targetProfileCode":               item.Projection.TargetProfileCode,
		"operationQualifiers":             nonNilSlice(item.Projection.OperationQualifiers),
		"activityQualifiers":              nonNilSlice(item.Projection.ActivityQualifiers),
		"applicabilityDisposition":        item.Projection.ApplicabilityDisposition,
		"evidenceExpectationCodes":        nonNilSlice(item.Projection.EvidenceExpectationCodes),
		"externalInvolvements":            nonNilSlice(item.Projection.ExternalInvolvements),
		"agreementConfidence":             item.AgreementConfidence,
		"recommendationState":             item.RecommendationState,
		"rationaleCodes":                  nonNilSlice(item.RationaleCodes),
		"confidenceEvidence":              nonNilSlice(item.ConfidenceEvidence),
		"sourceRefs":                      nonNilSlice(item.SourceRefs),
		"sourceMappingState":              item.SourceMappingState,
		"sourceAuthorityState":            item.SourceAuthorityState,
		"riskClassificationState":         item.RiskClassificationState,
		"decisionState":                   item.DecisionState,
		"extractionState":                 item.ExtractionState,
		"questionSourceProposalGap":       item.QuestionSourceProposalGap,
		"externalApplicabilityUnresolved": item.ExternalApplicabilityUnresolved,
		"passDisagreementCodes":           nonNilSlice(item.PassDisagreementCodes),
		"passOneResultDigest":             item.PassOneResultDigest,
		"passTwoResultDigest":             item.PassTwoResultDigest,
		"passOneRunId":                    item.PassOneRunID,
		"passTwoRunId":                    item.PassTwoRunID,
		"promptDigest":                    item.PromptDigest,
		"modelDescriptorDigests":          nonNilSlice(item.ModelDescriptorDigests),
		"taxonomyDigest":                  item.TaxonomyDigest,
		"inputDigest":                     item.InputDigest,
		"itemSemanticDigest":              item.ItemSemanticDigest,
		"classificationRunDigest":         item.ClassificationRunDigest,
		"aggregateDigest":                 item.AggregateDigest,
	}
}

func (item SealedClassificationItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(sealedClassificationItemObject(item))
}

func (item *SealedClassificationItem) UnmarshalJSON(data []byte) error {
	if !utf8.Valid(data) || hasJSONLoneSurrogate(data) {
		return ErrQuestionReferenceUnion
	}
	object, err := closedJSONObject(data)
	if err != nil {
		return ErrQuestionReferenceUnion
	}
	keys := []string{"packageVersion", "packageJsonSha256", "formCode", "proposalId", "ordinal", "textDigest", "taxonomyVersion", "mainDomainCode", "topicCodes", "inspectionProfileCodes", "inspectionTypeCodes", "canonicalTargetKind", "targetProfileCode", "operationQualifiers", "activityQualifiers", "applicabilityDisposition", "evidenceExpectationCodes", "externalInvolvements", "agreementConfidence", "recommendationState", "rationaleCodes", "confidenceEvidence", "sourceRefs", "sourceMappingState", "sourceAuthorityState", "riskClassificationState", "decisionState", "extractionState", "questionSourceProposalGap", "externalApplicabilityUnresolved", "passDisagreementCodes", "passOneResultDigest", "passTwoResultDigest", "passOneRunId", "passTwoRunId", "promptDigest", "modelDescriptorDigests", "taxonomyDigest", "inputDigest", "itemSemanticDigest", "classificationRunDigest", "aggregateDigest"}
	if !hasExactJSONKeys(object, keys...) {
		return ErrQuestionReferenceUnion
	}
	var flat struct {
		PackageVersion    string `json:"packageVersion"`
		PackageJSONSHA256 string `json:"packageJsonSha256"`
		FormCode          string `json:"formCode"`
		ProposalID        string `json:"proposalId"`
		Ordinal           int    `json:"ordinal"`
		TextDigest        string `json:"textDigest"`
		TaxonomyVersion   string `json:"taxonomyVersion"`
		ProposalProjection
		AgreementConfidence Confidence           `json:"agreementConfidence"`
		RecommendationState string               `json:"recommendationState"`
		RationaleCodes      []string             `json:"rationaleCodes"`
		ConfidenceEvidence  []ConfidenceEvidence `json:"confidenceEvidence"`
		SourceRefs          []SourceReference    `json:"sourceRefs"`
		GovernanceState
		PassDisagreementCodes   []string `json:"passDisagreementCodes"`
		PassOneResultDigest     string   `json:"passOneResultDigest"`
		PassTwoResultDigest     string   `json:"passTwoResultDigest"`
		PassOneRunID            string   `json:"passOneRunId"`
		PassTwoRunID            string   `json:"passTwoRunId"`
		PromptDigest            string   `json:"promptDigest"`
		ModelDescriptorDigests  []string `json:"modelDescriptorDigests"`
		TaxonomyDigest          string   `json:"taxonomyDigest"`
		InputDigest             string   `json:"inputDigest"`
		ItemSemanticDigest      string   `json:"itemSemanticDigest"`
		ClassificationRunDigest string   `json:"classificationRunDigest"`
		AggregateDigest         string   `json:"aggregateDigest"`
	}
	if err := strictJSONDecode(data, &flat); err != nil {
		return ErrQuestionReferenceUnion
	}
	if flat.TaxonomyVersion != FrozenTaxonomy().Version {
		return ErrQuestionReferenceUnion
	}
	identity := BaseIdentity{PackageVersion: flat.PackageVersion, PackageJSONSHA256: flat.PackageJSONSHA256, FormCode: flat.FormCode, ProposalID: flat.ProposalID, Ordinal: flat.Ordinal, TextDigest: flat.TextDigest}
	if err := identity.Validate(); err != nil {
		return ErrQuestionReferenceUnion
	}
	decoded := SealedClassificationItem{Identity: identity, Projection: flat.ProposalProjection, AgreementConfidence: flat.AgreementConfidence, RecommendationState: flat.RecommendationState, RationaleCodes: flat.RationaleCodes, ConfidenceEvidence: flat.ConfidenceEvidence, SourceRefs: flat.SourceRefs, GovernanceState: flat.GovernanceState, PassDisagreementCodes: flat.PassDisagreementCodes, PassOneResultDigest: flat.PassOneResultDigest, PassTwoResultDigest: flat.PassTwoResultDigest, PassOneRunID: flat.PassOneRunID, PassTwoRunID: flat.PassTwoRunID, PromptDigest: flat.PromptDigest, ModelDescriptorDigests: flat.ModelDescriptorDigests, TaxonomyDigest: flat.TaxonomyDigest, InputDigest: flat.InputDigest, ItemSemanticDigest: flat.ItemSemanticDigest, ClassificationRunDigest: flat.ClassificationRunDigest, AggregateDigest: flat.AggregateDigest}
	if err := ValidateProjection(FrozenTaxonomy(), decoded.Projection); err != nil || !contains([]string{string(ConfidenceHigh), string(ConfidenceMedium), string(ConfidenceLow)}, string(decoded.AgreementConfidence)) || !contains([]string{RecommendationAutoProposed, RecommendationManagerReview, RecommendationBlockedSourceGap}, decoded.RecommendationState) || decoded.SourceMappingState != SourceMappingRequired || decoded.SourceAuthorityState != SourceAuthorityNotAttested || decoded.RiskClassificationState != RiskExpertReviewRequired || decoded.DecisionState != DecisionNotSupplied || !contains([]string{ExtractionCandidate, ExtractionExactSourceBacked}, decoded.ExtractionState) || !validDigest(decoded.PromptDigest) || decoded.TaxonomyDigest != FrozenTaxonomy().Digest || decoded.InputDigest != ComputeRunInputDigestForPrompt(FrozenFixedInputDigests(), decoded.PromptDigest) || !validDigest(decoded.PassOneResultDigest) || !validDigest(decoded.PassTwoResultDigest) || !validDigest(decoded.ItemSemanticDigest) || !validDigest(decoded.ClassificationRunDigest) || !validDigest(decoded.AggregateDigest) || len(decoded.ModelDescriptorDigests) < 1 || len(decoded.ModelDescriptorDigests) > 2 || !reflect.DeepEqual(uniqueSorted(decoded.ModelDescriptorDigests), decoded.ModelDescriptorDigests) {
		return ErrQuestionReferenceUnion
	}
	rationales, rationaleErr := normalizeStrings(decoded.RationaleCodes, FrozenTaxonomy().RationaleCodes, "rationaleCodes", false)
	evidence, evidenceErr := normalizeEvidence(FrozenTaxonomy(), decoded.ConfidenceEvidence)
	sources, sourceErr := normalizeSourceRefs(FrozenTaxonomy(), decoded.SourceRefs)
	disagreements, disagreementErr := normalizeStrings(decoded.PassDisagreementCodes, FrozenTaxonomy().DisagreementCodes, "passDisagreementCodes", false)
	if rationaleErr != nil || evidenceErr != nil || sourceErr != nil || disagreementErr != nil || !reflect.DeepEqual(rationales, decoded.RationaleCodes) || !reflect.DeepEqual(evidence, decoded.ConfidenceEvidence) || !reflect.DeepEqual(sources, decoded.SourceRefs) || !reflect.DeepEqual(disagreements, decoded.PassDisagreementCodes) || validatePassRunID(PassCandidate, decoded.PassOneRunID) != nil || validatePassRunID(PassChallenge, decoded.PassTwoRunID) != nil {
		return ErrQuestionReferenceUnion
	}
	for _, digest := range decoded.ModelDescriptorDigests {
		if !validDigest(digest) {
			return ErrQuestionReferenceUnion
		}
	}
	canonical, err := json.Marshal(decoded)
	if err != nil || !semanticJSONEqual(data, canonical) || decoded.ItemSemanticDigest != ComputeItemSemanticDigest(decoded) {
		return ErrQuestionReferenceUnion
	}
	*item = decoded
	return nil
}

type ClassificationInput struct {
	ClassificationRunID            string
	CandidateClassificationRunID   string
	ChallengeClassificationRunID   string
	RunInputDigest                 string
	PromptDigest                   string
	TaxonomyDigest                 string
	FixedInputDigests              FixedInputDigests
	ModelDescriptors               []ModelDescriptor
	SuppliedModelDescriptorDigests []string
	AcceptSuppliedPassInputDigests bool
	BatchManifestDigest            string
	PassOneSealDigest              string
	PassTwoSealDigest              string
	PassOneSealReceipt             PassSealReceipt
	PassTwoSealReceipt             PassSealReceipt
	OrderedBaseIdentities          []BaseIdentity
	PassInputsByRole               map[PassRole][]ClassificationPassInput
	CandidateRecords               []PassProposalRecord
	ChallengeRecords               []PassProposalRecord
	GovernanceByIdentity           map[string]GovernanceState
	EvidenceFactsByIdentity        map[string]EvidenceFacts
}

type ClassificationPackageFacts struct {
	FormKind                string   `json:"formKind"`
	FormRiskBand            string   `json:"formRiskBand"`
	QuestionRiskBand        string   `json:"questionRiskBand"`
	QuestionRiskDomain      string   `json:"questionRiskDomain"`
	SourceMappingState      string   `json:"sourceMappingState"`
	SourceAuthorityState    string   `json:"sourceAuthorityState"`
	ExtractionState         string   `json:"extractionState"`
	RiskClassificationState string   `json:"riskClassificationState"`
	DecisionState           string   `json:"decisionState"`
	SourceProposalDigests   []string `json:"sourceProposalDigests"`
	SourceReferenceDigests  []string `json:"sourceReferenceDigests"`
}

type ClassificationResearchCandidateFacts struct {
	FormCode                        string `json:"form_code"`
	ProposalID                      string `json:"proposal_id"`
	Ordinal                         string `json:"ordinal"`
	TextDigest                      string `json:"text_digest"`
	TargetKind                      string `json:"target_kind"`
	OperationActivityQualifier      string `json:"operation_activity_qualifier"`
	PrimarySubjectProposal          string `json:"primary_subject_proposal"`
	OperationalInterfaceCandidates  string `json:"operational_interface_candidates"`
	EvidenceContributorCandidates   string `json:"evidence_contributor_candidates"`
	ProviderApplicabilityUnresolved string `json:"provider_applicability_unresolved"`
	UnresolvedReasons               string `json:"unresolved_reasons"`
	SourceRefs                      string `json:"source_refs"`
}

type ClassificationPassInputItem struct {
	Identity               BaseIdentity                         `json:"identity"`
	QuestionBody           string                               `json:"questionBody"`
	PackageFacts           ClassificationPackageFacts           `json:"packageFacts"`
	ResearchCandidateFacts ClassificationResearchCandidateFacts `json:"researchCandidateFacts"`
}

type ClassificationPassInput struct {
	SchemaVersion         string                        `json:"schemaVersion"`
	Purpose               string                        `json:"purpose"`
	ClassificationRunID   string                        `json:"classificationRunId"`
	PassRole              PassRole                      `json:"passRole"`
	PassRunID             string                        `json:"passRunId"`
	BatchOrdinal          int                           `json:"batchOrdinal"`
	TaxonomyVersion       string                        `json:"taxonomyVersion"`
	TaxonomyDigest        string                        `json:"taxonomyDigest"`
	PromptDigest          string                        `json:"promptDigest"`
	ModelDescriptorDigest string                        `json:"modelDescriptorDigest"`
	BatchManifestDigest   string                        `json:"batchManifestDigest"`
	FixedInputDigests     FixedInputDigests             `json:"fixedInputDigests"`
	Items                 []ClassificationPassInputItem `json:"items"`
}

type PassBatchOutput struct {
	SchemaVersion         string               `json:"schemaVersion"`
	ClassificationRunID   string               `json:"classificationRunId"`
	PassRole              PassRole             `json:"passRole"`
	PassRunID             string               `json:"passRunId"`
	BatchOrdinal          int                  `json:"batchOrdinal"`
	PromptDigest          string               `json:"promptDigest"`
	ModelDescriptorDigest string               `json:"modelDescriptorDigest"`
	InputDigest           string               `json:"inputDigest"`
	Records               []PassProposalRecord `json:"records"`
	BatchOutputDigest     string               `json:"batchOutputDigest"`
}

type PassSealReceipt struct {
	ClassificationRunID       string   `json:"classificationRunId"`
	PassRole                  PassRole `json:"passRole"`
	PassRunID                 string   `json:"passRunId"`
	PromptDigest              string   `json:"promptDigest"`
	ModelDescriptorDigest     string   `json:"modelDescriptorDigest"`
	BatchManifestDigest       string   `json:"batchManifestDigest"`
	BatchCount                int      `json:"batchCount"`
	ItemCount                 int      `json:"itemCount"`
	OrderedInputDigests       []string `json:"orderedInputDigests"`
	PassInputSetDigest        string   `json:"passInputSetDigest"`
	OrderedBatchOutputDigests []string `json:"orderedBatchOutputDigests"`
	OrderedPassResultDigests  []string `json:"orderedPassResultDigests"`
	PassSealDigest            string   `json:"passSealDigest"`
}

type CodeCount struct {
	Code  string `json:"code"`
	Count int    `json:"count"`
}

type ClassificationDistributions struct {
	AgreementConfidenceCounts      []CodeCount `json:"agreementConfidenceCounts"`
	ApplicabilityDispositionCounts []CodeCount `json:"applicabilityDispositionCounts"`
	CanonicalTargetKindCounts      []CodeCount `json:"canonicalTargetKindCounts"`
	DisagreementCodeCounts         []CodeCount `json:"disagreementCodeCounts"`
	EvidenceExpectationCodeCounts  []CodeCount `json:"evidenceExpectationCodeCounts"`
	ExternalProviderTypeCodeCounts []CodeCount `json:"externalProviderTypeCodeCounts"`
	ExtractionStateCounts          []CodeCount `json:"extractionStateCounts"`
	InspectionProfileCodeCounts    []CodeCount `json:"inspectionProfileCodeCounts"`
	InspectionTypeCodeCounts       []CodeCount `json:"inspectionTypeCodeCounts"`
	MainDomainCodeCounts           []CodeCount `json:"mainDomainCodeCounts"`
	RecommendationStateCounts      []CodeCount `json:"recommendationStateCounts"`
	TargetProfileCodeCounts        []CodeCount `json:"targetProfileCodeCounts"`
	TopicCodeCounts                []CodeCount `json:"topicCodeCounts"`
}

type ExceptionInventory struct {
	Count                  int      `json:"count"`
	OrderedIdentityDigests []string `json:"orderedIdentityDigests"`
	OrderedIdentityDigest  string   `json:"orderedIdentityDigest"`
}

type ClassificationExceptions struct {
	BlockedSourceGap                   ExceptionInventory `json:"blockedSourceGap"`
	ExternalApplicabilityUnresolved    ExceptionInventory `json:"externalApplicabilityUnresolved"`
	ManagerReviewRequired              ExceptionInventory `json:"managerReviewRequired"`
	PassDisagreement                   ExceptionInventory `json:"passDisagreement"`
	SourceGapExternalUnresolvedOverlap ExceptionInventory `json:"sourceGapExternalUnresolvedOverlap"`
}

type ClassificationAggregate struct {
	ItemCount                  int                         `json:"itemCount"`
	PassProposalRecordCount    int                         `json:"passProposalRecordCount"`
	OrderedItemSemanticDigests []string                    `json:"orderedItemSemanticDigests"`
	Distributions              ClassificationDistributions `json:"distributions"`
	Exceptions                 ClassificationExceptions    `json:"exceptions"`
	DistributionDigest         string                      `json:"distributionDigest"`
	AggregateDigest            string                      `json:"aggregateDigest"`
}

type ClassificationRunReceipt struct {
	ClassificationRunID     string            `json:"classificationRunId"`
	State                   string            `json:"state"`
	TaxonomyVersion         string            `json:"taxonomyVersion"`
	TaxonomyDigest          string            `json:"taxonomyDigest"`
	FixedInputDigests       FixedInputDigests `json:"fixedInputDigests"`
	InputDigest             string            `json:"inputDigest"`
	PromptDigest            string            `json:"promptDigest"`
	ModelDescriptors        []ModelDescriptor `json:"modelDescriptors"`
	ModelDescriptorDigests  []string          `json:"modelDescriptorDigests"`
	BatchManifestDigest     string            `json:"batchManifestDigest"`
	PassOneSealDigest       string            `json:"passOneSealDigest"`
	PassTwoSealDigest       string            `json:"passTwoSealDigest"`
	AggregateDigest         string            `json:"aggregateDigest"`
	ClassificationRunDigest string            `json:"classificationRunDigest"`
}

type ClassificationResult struct {
	ClassificationRunID     string                     `json:"classificationRunId"`
	State                   string                     `json:"state"`
	TaxonomyVersion         string                     `json:"taxonomyVersion"`
	TaxonomyDigest          string                     `json:"taxonomyDigest"`
	InputDigest             string                     `json:"inputDigest"`
	AggregateDigest         string                     `json:"aggregateDigest"`
	ClassificationRunDigest string                     `json:"classificationRunDigest"`
	Aggregate               ClassificationAggregate    `json:"aggregate"`
	RunReceipt              ClassificationRunReceipt   `json:"runReceipt"`
	CandidateRecords        []PassProposalRecord       `json:"candidateRecords"`
	ChallengeRecords        []PassProposalRecord       `json:"challengeRecords"`
	Items                   []SealedClassificationItem `json:"items"`
	PassOneSealReceipt      PassSealReceipt            `json:"passOneSealReceipt"`
	PassTwoSealReceipt      PassSealReceipt            `json:"passTwoSealReceipt"`
}

func (result *ClassificationResult) UnmarshalJSON(data []byte) error {
	if !utf8.Valid(data) || hasJSONLoneSurrogate(data) {
		return ErrQuestionReferenceUnion
	}
	object, err := closedJSONObject(data)
	if err != nil || !hasExactJSONKeys(object, "classificationRunId", "state", "taxonomyVersion", "taxonomyDigest", "inputDigest", "aggregateDigest", "classificationRunDigest", "aggregate", "runReceipt", "candidateRecords", "challengeRecords", "items", "passOneSealReceipt", "passTwoSealReceipt") {
		return ErrQuestionReferenceUnion
	}
	type classificationResultAlias ClassificationResult
	var decoded classificationResultAlias
	if err := strictJSONDecode(data, &decoded); err != nil {
		return ErrQuestionReferenceUnion
	}
	canonical, err := json.Marshal(decoded)
	if err != nil || !semanticJSONEqual(data, canonical) {
		return ErrQuestionReferenceUnion
	}
	*result = ClassificationResult(decoded)
	return nil
}

// PostgreSQL jsonb preserves JSON values but not object-member order. The
// closed-field checks above still reject unknown fields; this comparison keeps
// the same semantic/canonical values while allowing a persistence round trip
// through jsonb to reorder object members.
func semanticJSONEqual(left, right []byte) bool {
	var leftValue, rightValue any
	if json.Unmarshal(left, &leftValue) != nil || json.Unmarshal(right, &rightValue) != nil {
		return false
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

type ClassificationOutcome struct {
	AgreementConfidence Confidence
	RecommendationState string
}

type QuestionOrigin string

const (
	QuestionOriginBase      QuestionOrigin = "SEALED_BASE"
	QuestionOriginWorkspace QuestionOrigin = "WORKSPACE"
)

type ParentQuestionKey struct {
	Base                  *BaseIdentity `json:"base,omitempty"`
	WorkspaceGenerationID string        `json:"workspaceGenerationId,omitempty"`
	WorkspaceRootID       string        `json:"workspaceRootId,omitempty"`
	WorkspaceVersionID    string        `json:"workspaceVersionId,omitempty"`
	WorkspaceProposalID   string        `json:"workspaceProposalId,omitempty"`
	WorkspaceRootSequence int           `json:"workspaceRootSequence,omitempty"`
	WorkspaceBodyDigest   string        `json:"workspaceBodyDigest,omitempty"`
}

type WorkspaceQuestionRef struct {
	GenerationID      string             `json:"generationId"`
	RootID            string             `json:"rootId"`
	VersionID         string             `json:"versionId"`
	ProposalID        string             `json:"proposalId"`
	RootSequence      int                `json:"rootSequence"`
	BodyDigest        string             `json:"bodyDigest"`
	ParentQuestionKey *ParentQuestionKey `json:"parentQuestionKey"`
	ActorSubjectID    string             `json:"actorSubjectId"`
	CreatedAt         time.Time          `json:"createdAt"`
	ReasonCode        string             `json:"reasonCode"`
}

type QuestionRef struct {
	Origin       QuestionOrigin        `json:"questionOrigin"`
	Base         *BaseIdentity         `json:"base,omitempty"`
	Workspace    *WorkspaceQuestionRef `json:"workspace,omitempty"`
	RootSequence int                   `json:"rootSequence"`
}

func parentQuestionKeyObject(parent *ParentQuestionKey) any {
	if parent == nil {
		return nil
	}
	if parent.Base != nil {
		return map[string]any{
			"questionOrigin":    QuestionOriginBase,
			"packageVersion":    parent.Base.PackageVersion,
			"packageJsonSha256": parent.Base.PackageJSONSHA256,
			"formCode":          parent.Base.FormCode,
			"proposalId":        parent.Base.ProposalID,
			"ordinal":           parent.Base.Ordinal,
			"textDigest":        parent.Base.TextDigest,
		}
	}
	return map[string]any{
		"questionOrigin":    QuestionOriginWorkspace,
		"generationId":      parent.WorkspaceGenerationID,
		"questionRootId":    parent.WorkspaceRootID,
		"questionVersionId": parent.WorkspaceVersionID,
		"proposalId":        parent.WorkspaceProposalID,
		"rootSequence":      parent.WorkspaceRootSequence,
		"bodyDigest":        parent.WorkspaceBodyDigest,
	}
}

func questionRefObject(reference QuestionRef) map[string]any {
	if reference.Origin == QuestionOriginBase && reference.Base != nil {
		return map[string]any{
			"questionOrigin":    QuestionOriginBase,
			"packageVersion":    reference.Base.PackageVersion,
			"packageJsonSha256": reference.Base.PackageJSONSHA256,
			"formCode":          reference.Base.FormCode,
			"proposalId":        reference.Base.ProposalID,
			"ordinal":           reference.Base.Ordinal,
			"textDigest":        reference.Base.TextDigest,
		}
	}
	if reference.Origin == QuestionOriginWorkspace && reference.Workspace != nil {
		return map[string]any{
			"questionOrigin":     QuestionOriginWorkspace,
			"generationId":       reference.Workspace.GenerationID,
			"questionRootId":     reference.Workspace.RootID,
			"questionVersionId":  reference.Workspace.VersionID,
			"proposalId":         reference.Workspace.ProposalID,
			"rootSequence":       reference.Workspace.RootSequence,
			"bodyDigest":         reference.Workspace.BodyDigest,
			"parentQuestionKey":  parentQuestionKeyObject(reference.Workspace.ParentQuestionKey),
			"createdBySubjectId": reference.Workspace.ActorSubjectID,
			"createdAt":          reference.Workspace.CreatedAt.UTC().Format(time.RFC3339Nano),
			"reasonCode":         reference.Workspace.ReasonCode,
		}
	}
	return map[string]any{"questionOrigin": reference.Origin}
}

func (reference QuestionRef) MarshalJSON() ([]byte, error) {
	return json.Marshal(questionRefObject(reference))
}

func (reference *QuestionRef) UnmarshalJSON(data []byte) error {
	if !utf8.Valid(data) || hasJSONLoneSurrogate(data) {
		return ErrQuestionReferenceUnion
	}
	object, err := closedJSONObject(data)
	if err != nil {
		return ErrQuestionReferenceUnion
	}
	var origin QuestionOrigin
	if raw, ok := object["questionOrigin"]; !ok || json.Unmarshal(raw, &origin) != nil {
		return ErrQuestionReferenceUnion
	}
	var decoded QuestionRef
	switch origin {
	case QuestionOriginBase:
		if !hasExactJSONKeys(object, "questionOrigin", "packageVersion", "packageJsonSha256", "formCode", "proposalId", "ordinal", "textDigest") {
			return ErrQuestionReferenceUnion
		}
		var flat struct {
			PackageVersion    string `json:"packageVersion"`
			PackageJSONSHA256 string `json:"packageJsonSha256"`
			FormCode          string `json:"formCode"`
			ProposalID        string `json:"proposalId"`
			Ordinal           int    `json:"ordinal"`
			TextDigest        string `json:"textDigest"`
		}
		if err := json.Unmarshal(data, &flat); err != nil {
			return ErrQuestionReferenceUnion
		}
		identity := BaseIdentity{
			PackageVersion: flat.PackageVersion, PackageJSONSHA256: flat.PackageJSONSHA256,
			FormCode: flat.FormCode, ProposalID: flat.ProposalID, Ordinal: flat.Ordinal, TextDigest: flat.TextDigest,
		}
		decoded = BaseQuestionReference(identity)
	case QuestionOriginWorkspace:
		if !hasExactJSONKeys(object, "questionOrigin", "generationId", "questionRootId", "questionVersionId", "proposalId", "rootSequence", "bodyDigest", "parentQuestionKey", "createdBySubjectId", "createdAt", "reasonCode") {
			return ErrQuestionReferenceUnion
		}
		var flat struct {
			GenerationID      string          `json:"generationId"`
			RootID            string          `json:"questionRootId"`
			VersionID         string          `json:"questionVersionId"`
			ProposalID        string          `json:"proposalId"`
			RootSequence      int             `json:"rootSequence"`
			BodyDigest        string          `json:"bodyDigest"`
			ParentQuestionKey json.RawMessage `json:"parentQuestionKey"`
			ActorSubjectID    string          `json:"createdBySubjectId"`
			CreatedAt         string          `json:"createdAt"`
			ReasonCode        string          `json:"reasonCode"`
		}
		if err := json.Unmarshal(data, &flat); err != nil {
			return ErrQuestionReferenceUnion
		}
		createdAt, err := parseCanonicalUTCTimestamp(flat.CreatedAt)
		if err != nil {
			return ErrQuestionReferenceUnion
		}
		parent, err := decodeParentQuestionKey(flat.ParentQuestionKey)
		if err != nil {
			return err
		}
		decoded = WorkspaceQuestionReference(WorkspaceQuestionRef{
			GenerationID: flat.GenerationID, RootID: flat.RootID, VersionID: flat.VersionID,
			ProposalID: flat.ProposalID, RootSequence: flat.RootSequence, BodyDigest: flat.BodyDigest,
			ParentQuestionKey: parent, ActorSubjectID: flat.ActorSubjectID, CreatedAt: createdAt, ReasonCode: flat.ReasonCode,
		})
	default:
		return ErrQuestionReferenceUnion
	}
	if err := ValidateQuestionRef(decoded); err != nil {
		return err
	}
	*reference = decoded
	return nil
}

func closedJSONObject(data []byte) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil || object == nil {
		return nil, ErrQuestionReferenceUnion
	}
	return object, nil
}

func strictJSONDecode(data []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func hasExactJSONKeys(object map[string]json.RawMessage, keys ...string) bool {
	if len(object) != len(keys) {
		return false
	}
	for _, key := range keys {
		if _, ok := object[key]; !ok {
			return false
		}
	}
	return true
}

func parseCanonicalUTCTimestamp(value string) (time.Time, error) {
	if !regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]{1,9})?Z$`).MatchString(value) {
		return time.Time{}, ErrQuestionReferenceUnion
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || !strings.HasSuffix(value, "Z") {
		return time.Time{}, ErrQuestionReferenceUnion
	}
	return parsed.UTC(), nil
}

func decodeParentQuestionKey(data json.RawMessage) (*ParentQuestionKey, error) {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil, nil
	}
	object, err := closedJSONObject(data)
	if err != nil {
		return nil, ErrParentQuestionKey
	}
	var origin QuestionOrigin
	if raw, ok := object["questionOrigin"]; !ok || json.Unmarshal(raw, &origin) != nil {
		return nil, ErrParentQuestionKey
	}
	switch origin {
	case QuestionOriginBase:
		if !hasExactJSONKeys(object, "questionOrigin", "packageVersion", "packageJsonSha256", "formCode", "proposalId", "ordinal", "textDigest") {
			return nil, ErrParentQuestionKey
		}
		var flat struct {
			PackageVersion    string `json:"packageVersion"`
			PackageJSONSHA256 string `json:"packageJsonSha256"`
			FormCode          string `json:"formCode"`
			ProposalID        string `json:"proposalId"`
			Ordinal           int    `json:"ordinal"`
			TextDigest        string `json:"textDigest"`
		}
		if err := json.Unmarshal(data, &flat); err != nil {
			return nil, ErrParentQuestionKey
		}
		base := BaseIdentity{
			PackageVersion: flat.PackageVersion, PackageJSONSHA256: flat.PackageJSONSHA256,
			FormCode: flat.FormCode, ProposalID: flat.ProposalID, Ordinal: flat.Ordinal, TextDigest: flat.TextDigest,
		}
		if err := base.Validate(); err != nil {
			return nil, ErrParentQuestionKey
		}
		return &ParentQuestionKey{Base: &base}, nil
	case QuestionOriginWorkspace:
		if !hasExactJSONKeys(object, "questionOrigin", "generationId", "questionRootId", "questionVersionId", "proposalId", "rootSequence", "bodyDigest") {
			return nil, ErrParentQuestionKey
		}
		var flat struct {
			GenerationID string `json:"generationId"`
			RootID       string `json:"questionRootId"`
			VersionID    string `json:"questionVersionId"`
			ProposalID   string `json:"proposalId"`
			RootSequence int    `json:"rootSequence"`
			BodyDigest   string `json:"bodyDigest"`
		}
		if err := json.Unmarshal(data, &flat); err != nil {
			return nil, ErrParentQuestionKey
		}
		return &ParentQuestionKey{
			WorkspaceGenerationID: flat.GenerationID, WorkspaceRootID: flat.RootID,
			WorkspaceVersionID: flat.VersionID, WorkspaceProposalID: flat.ProposalID,
			WorkspaceRootSequence: flat.RootSequence, WorkspaceBodyDigest: flat.BodyDigest,
		}, nil
	default:
		return nil, ErrParentQuestionKey
	}
}

func BaseQuestionReference(identity BaseIdentity) QuestionRef {
	copy := identity
	return QuestionRef{Origin: QuestionOriginBase, Base: &copy}
}

func WorkspaceQuestionReference(workspace WorkspaceQuestionRef) QuestionRef {
	copy := workspace
	return QuestionRef{Origin: QuestionOriginWorkspace, Workspace: &copy, RootSequence: workspace.RootSequence}
}

func (reference QuestionRef) Key() string {
	if reference.Origin == QuestionOriginBase && reference.Base != nil {
		return "base\x1f" + reference.Base.Key()
	}
	if reference.Origin == QuestionOriginWorkspace && reference.Workspace != nil {
		return "workspace\x1f" + reference.Workspace.GenerationID + "\x1f" + reference.Workspace.RootID + "\x1f" + reference.Workspace.VersionID + "\x1f" + reference.Workspace.ProposalID
	}
	return ""
}

type DraftAction string

const (
	DraftRetain                         DraftAction = "RETAIN"
	DraftInclude                        DraftAction = "INCLUDE"
	DraftExclude                        DraftAction = "EXCLUDE"
	DraftDefer                          DraftAction = "DEFER"
	DraftReclassifyMainDomain           DraftAction = "RECLASSIFY_MAIN_DOMAIN"
	DraftAddTopic                       DraftAction = "ADD_TOPIC"
	DraftRemoveTopic                    DraftAction = "REMOVE_TOPIC"
	DraftResolveClassificationProposals DraftAction = "RESOLVE_CLASSIFICATION_PROPOSALS"
	DraftAddCandidate                   DraftAction = "ADD_CANDIDATE"
	DraftRewordCandidate                DraftAction = "REWORD_CANDIDATE"
	DraftMarkReady                      DraftAction = "MARK_READY_FOR_DEMO_SIMULATION"
)

const (
	DraftWorking                = "WORKING"
	DraftReadyForDemoSimulation = "READY_FOR_DEMO_SIMULATION"
	ReviewAutoPreselected       = "AUTO_PRESELECTED"
	ReviewPendingManager        = "PENDING_MANAGER_REVIEW"
	ReviewManagerDisposed       = "MANAGER_DISPOSED"
	DispositionInclude          = "INCLUDE"
	DispositionExclude          = "EXCLUDE"
	DispositionDefer            = "DEFER"
	DraftItemOriginSealedBase   = "SEALED_BASE"
	DraftItemOriginAuthored     = "MANAGER_AUTHORED"
	DraftItemOriginReworded     = "MANAGER_REWORDED"
)

type ResolutionMode string

const (
	ResolutionCandidate ResolutionMode = "ACCEPT_CANDIDATE_PASS"
	ResolutionChallenge ResolutionMode = "ACCEPT_CHALLENGE_PASS"
	ResolutionSetExact  ResolutionMode = "SET_EXACT"
)

type ProposalResolution struct {
	Mode               ResolutionMode      `json:"mode"`
	ProposalProjection *ProposalProjection `json:"proposalProjection,omitempty"`
}

type DraftItem struct {
	QuestionRef               QuestionRef         `json:"questionRef"`
	Origin                    string              `json:"draftItemOrigin"`
	CurrentProjection         ProposalProjection  `json:"proposalProjection"`
	DraftAgreementConfidence  *Confidence         `json:"draftAgreementConfidence"`
	RecommendationState       string              `json:"draftRecommendationState"`
	ReviewState               string              `json:"draftReviewState"`
	Disposition               *string             `json:"draftDisposition"`
	QuestionSourceProposalGap bool                `json:"questionSourceProposalGap"`
	ProposalResolution        *ProposalResolution `json:"proposalResolution"`
	Current                   bool                `json:"currentLeaf"`

	SealedAgreementConfidence       *Confidence `json:"-"`
	SourceMappingState              string      `json:"-"`
	SourceAuthorityState            string      `json:"-"`
	RiskClassificationState         string      `json:"-"`
	ExternalApplicabilityUnresolved bool        `json:"-"`
	DecisionState                   string      `json:"-"`
	ExtractionState                 string      `json:"-"`
	sealedGovernance                GovernanceState
	sealedRecommendationState       string
	candidatePassRecord             *PassProposalRecord
	challengePassRecord             *PassProposalRecord
	candidatePassResultDigest       string
	challengePassResultDigest       string
	sealedBaseRootSequence          int
}

type ReadinessEvent struct {
	ReadinessEventID           string    `json:"readinessEventId"`
	GenerationID               string    `json:"generationId"`
	ClassificationRunID        string    `json:"classificationRunId"`
	ClassificationRunDigest    string    `json:"classificationRunDigest"`
	TaxonomyVersion            string    `json:"taxonomyVersion"`
	TaxonomyDigest             string    `json:"taxonomyDigest"`
	DraftID                    string    `json:"draftId"`
	DraftRevision              int       `json:"draftRevision"`
	DraftContentDigest         string    `json:"draftContentDigest"`
	ProviderScopeProfileDigest string    `json:"providerScopeProfileDigest"`
	ActorSubjectID             string    `json:"actorSubjectId"`
	ReasonCode                 string    `json:"reasonCode"`
	CreatedAt                  time.Time `json:"createdAt"`
	ReadinessEventDigest       string    `json:"readinessEventDigest"`
}

type Draft struct {
	DraftID                   string            `json:"draftId"`
	GenerationID              string            `json:"generationId"`
	GenerationState           string            `json:"generationState"`
	Revision                  int               `json:"revision"`
	ContentDigest             string            `json:"contentDigest"`
	State                     string            `json:"state"`
	ClassificationRunID       string            `json:"classificationRunId"`
	ClassificationRunState    string            `json:"classificationRunState"`
	ClassificationRunDigest   string            `json:"classificationRunDigest"`
	AggregateDigest           string            `json:"aggregateDigest"`
	TaxonomyDigest            string            `json:"taxonomyDigest"`
	TaxonomyVersion           string            `json:"taxonomyVersion"`
	PackageVersion            string            `json:"packageVersion"`
	PackageJSONSHA256         string            `json:"packageJsonSha256"`
	ClassificationInputDigest string            `json:"classificationInputDigest"`
	FixedInputDigests         FixedInputDigests `json:"fixedInputDigests"`
	BaseQuestionCount         int               `json:"baseQuestionCount"`
	ClassificationItemCount   int               `json:"classificationItemCount"`
	ReadinessEvents           []ReadinessEvent  `json:"readinessEvents"`
	CurrentReadinessEventID   string            `json:"currentReadinessEventId"`
	Items                     []DraftItem       `json:"items"`
	sealedCandidateRecords    map[string]PassProposalRecord
	sealedChallengeRecords    map[string]PassProposalRecord
	sealedPassGraphValidated  bool
}

type DraftCommand struct {
	OperationID                string
	IdempotencyKey             string
	ExpectedGenerationID       string
	Action                     DraftAction
	TargetQuestionKey          string
	ExpectedRevision           int
	ExpectedContentDigest      string
	ReasonCode                 string
	ActorSubjectID             string
	CreatedAt                  time.Time
	MainDomainCode             string
	TopicCode                  string
	ResolutionMode             ResolutionMode
	ExactProjection            *ProposalProjection
	WorkspaceBody              string
	WorkspaceBodyDigest        string
	ReadinessEventID           string
	ProviderScopeProfileDigest string
}

type IDAllocator interface {
	NextRootID() string
	NextVersionID() string
	NextProposalID() string
	NextPreviewID() string
}

type SequentialIDAllocator struct {
	prefix   string
	root     int
	version  int
	proposal int
	preview  int
}

func NewSequentialIDAllocator(prefix string) *SequentialIDAllocator {
	return &SequentialIDAllocator{prefix: prefix}
}

func (allocator *SequentialIDAllocator) NextRootID() string {
	allocator.root++
	return fmt.Sprintf("aga-ws-root-%s-%d", allocator.prefix, allocator.root)
}
func (allocator *SequentialIDAllocator) NextVersionID() string {
	allocator.version++
	return fmt.Sprintf("aga-ws-version-%s-%d", allocator.prefix, allocator.version)
}
func (allocator *SequentialIDAllocator) NextProposalID() string {
	allocator.proposal++
	return fmt.Sprintf("aga-ws-proposal-%s-%d", allocator.prefix, allocator.proposal)
}
func (allocator *SequentialIDAllocator) NextPreviewID() string {
	allocator.preview++
	return fmt.Sprintf("aga-ws-preview-%s-%d", allocator.prefix, allocator.preview)
}

type DraftBatchFilter struct {
	MainDomainCodes []string `json:"mainDomainCodes"`
}

type DraftBatchAction struct {
	Action         DraftAction `json:"action"`
	MainDomainCode string      `json:"mainDomainCode,omitempty"`
	TopicCode      string      `json:"topicCode,omitempty"`
	ReasonCode     string      `json:"reasonCode,omitempty"`
}

type DraftBatchPreview struct {
	PreviewID              string    `json:"previewId"`
	GenerationID           string    `json:"generationId"`
	ClassificationRunID    string    `json:"classificationRunId"`
	ActionDigest           string    `json:"actionDigest"`
	DraftRevision          int       `json:"draftRevision"`
	DraftContentDigest     string    `json:"draftContentDigest"`
	FilterDigest           string    `json:"filterDigest"`
	OrderedIdentityDigests []string  `json:"orderedIdentityDigests"`
	OrderedIdentityDigest  string    `json:"orderedIdentityDigest"`
	Count                  int       `json:"count"`
	ExpiresAt              time.Time `json:"expiresAt"`
	PreviewDigest          string    `json:"previewDigest"`
}

type DraftBatchExecution struct {
	OperationID                string `json:"operationId"`
	IdempotencyKey             string `json:"idempotencyKey"`
	ExpectedGenerationID       string `json:"expectedGenerationId"`
	ExpectedDraftRevision      int    `json:"expectedDraftRevision"`
	ExpectedDraftContentDigest string `json:"expectedDraftContentDigest"`
	PreviewID                  string `json:"previewId"`
	PreviewDigest              string `json:"previewDigest"`
}

type TypedTarget struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	ProfileCode string `json:"profileCode"`
}

type ProviderScopeFact struct {
	GenerationID         string
	ProfileDigest        string
	OrganizationID       string
	ProviderScopeRootID  string
	ProviderScopeID      string
	ProviderScopeVersion int
	ProviderTypeID       string
	ProviderTypeCode     string
	Status               string
	EffectiveFrom        time.Time
	EffectiveTo          *time.Time
	DepartmentID         string
	OrganizationalUnitID string
	Targets              []TypedTarget
	OperationQualifiers  []Qualifier
	ActivityQualifiers   []Qualifier
}

func providerScopeProfilePreimage(scope ProviderScopeFact) map[string]any {
	targets := append([]TypedTarget{}, scope.Targets...)
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].ID == targets[j].ID {
			return targets[i].Kind+"\x00"+targets[i].ProfileCode < targets[j].Kind+"\x00"+targets[j].ProfileCode
		}
		return targets[i].ID < targets[j].ID
	})
	operation, _ := normalizeQualifiers(scope.OperationQualifiers, FrozenTaxonomy().OperationQualifierValues, "operationQualifiers")
	activity, _ := normalizeQualifiers(scope.ActivityQualifiers, FrozenTaxonomy().ActivityQualifierValues, "activityQualifiers")
	var effectiveTo any
	if scope.EffectiveTo != nil {
		effectiveTo = scope.EffectiveTo.UTC().Format(time.RFC3339Nano)
	}
	return map[string]any{
		"generationId": scope.GenerationID, "organizationId": scope.OrganizationID,
		"providerScopeRootId": scope.ProviderScopeRootID, "providerScopeId": scope.ProviderScopeID,
		"providerScopeVersion": scope.ProviderScopeVersion, "providerTypeId": scope.ProviderTypeID,
		"providerTypeCode": scope.ProviderTypeCode, "departmentId": scope.DepartmentID,
		"organizationalUnitId": scope.OrganizationalUnitID, "targets": targets,
		"operationQualifiers": operation, "activityQualifiers": activity, "status": scope.Status,
		"effectiveFrom": scope.EffectiveFrom.UTC().Format(time.RFC3339Nano), "effectiveTo": effectiveTo,
	}
}

func ComputeProviderScopeProfileDigest(scope ProviderScopeFact) string {
	digest, err := DigestValue("AGA-DEMO-PROVIDER-SCOPE-PROFILE-V1", providerScopeProfilePreimage(scope))
	if err != nil {
		return ""
	}
	return digest
}

type RecommendationRequest struct {
	OperationID             string
	IdempotencyKey          string
	ExpectedGenerationID    string
	OrganizationID          string
	ProviderScopeRootID     string
	ProviderScopeID         string
	ProviderScopeVersion    int
	ProviderTypeID          string
	DepartmentID            string
	OrganizationalUnitID    string
	TargetID                string
	CanonicalTargetKind     string
	TargetProfileCode       string
	InspectionProfileCode   string
	InspectionTypeCode      string
	OperationQualifiers     []Qualifier
	ActivityQualifiers      []Qualifier
	EffectiveAt             time.Time
	TaxonomyVersion         string
	TaxonomyDigest          string
	ClassificationRunID     string
	ClassificationRunDigest string
	DraftID                 string
	DraftRevision           int
	DraftContentDigest      string
	ExpectedDraftRevision   int
	ReadinessEventID        string
	ReadinessEventDigest    string
}

type RecommendationItem struct {
	QuestionRef      QuestionRef        `json:"questionRef"`
	RootSequence     int                `json:"rootSequence"`
	Current          bool               `json:"current"`
	DraftDisposition string             `json:"draftDisposition"`
	Projection       ProposalProjection `json:"projection"`
}

// RecommendationItem carries the accepted package order beside the compact
// base QuestionRef union. Base references intentionally omit rootSequence from
// their flat public shape, so restore that immutable ordering field when a
// persisted recommendation snapshot is decoded through JSON/JSONB.
func (item *RecommendationItem) UnmarshalJSON(data []byte) error {
	type recommendationItemAlias RecommendationItem
	var decoded recommendationItemAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if decoded.QuestionRef.Base != nil {
		decoded.QuestionRef.RootSequence = decoded.RootSequence
	}
	*item = RecommendationItem(decoded)
	return nil
}

type Recommendation struct {
	RecommendationID           string               `json:"recommendationId"`
	Revision                   int                  `json:"revision"`
	OperationID                string               `json:"operationId"`
	IdempotencyKey             string               `json:"idempotencyKey"`
	GenerationID               string               `json:"generationId"`
	DraftID                    string               `json:"draftId"`
	DraftRevision              int                  `json:"draftRevision"`
	DraftContentDigest         string               `json:"draftContentDigest"`
	TaxonomyVersion            string               `json:"taxonomyVersion"`
	TaxonomyDigest             string               `json:"taxonomyDigest"`
	ClassificationRunID        string               `json:"classificationRunId"`
	ClassificationRunDigest    string               `json:"classificationRunDigest"`
	AggregateDigest            string               `json:"aggregateDigest"`
	OrganizationID             string               `json:"organizationId"`
	ProviderScopeRootID        string               `json:"providerScopeRootId"`
	ProviderScopeID            string               `json:"providerScopeId"`
	ProviderScopeVersion       int                  `json:"providerScopeVersion"`
	ProviderScopeProfileDigest string               `json:"providerScopeProfileDigest"`
	ProviderTypeID             string               `json:"providerTypeId"`
	ProviderTypeCode           string               `json:"providerTypeCode"`
	DepartmentID               string               `json:"departmentId"`
	OrganizationalUnitID       string               `json:"organizationalUnitId"`
	TargetID                   string               `json:"targetId"`
	CanonicalTargetKind        string               `json:"canonicalTargetKind"`
	TargetProfileCode          string               `json:"targetProfileCode"`
	InspectionProfileCode      string               `json:"inspectionProfileCode"`
	InspectionTypeCode         string               `json:"inspectionTypeCode"`
	OperationQualifiers        []Qualifier          `json:"operationQualifiers"`
	ActivityQualifiers         []Qualifier          `json:"activityQualifiers"`
	EffectiveAt                time.Time            `json:"effectiveAt"`
	ReadinessEventID           string               `json:"readinessEventId"`
	ReadinessEventDigest       string               `json:"readinessEventDigest"`
	Items                      []RecommendationItem `json:"items"`
	Digest                     string               `json:"digest"`
}

func validDigest(value string) bool {
	return digestPattern.MatchString(value)
}

func digestValue(domain string, value any) string {
	digest, err := DigestValue(domain, value)
	if err != nil {
		return ""
	}
	return digest
}

func DigestValue(domain string, value any) (string, error) {
	canonical, err := canonicalJSON(value)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(append([]byte(domain), canonical...))
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}

func hasJSONLoneSurrogate(data []byte) bool {
	for index := 0; index+5 < len(data); index++ {
		if data[index] != '\\' {
			continue
		}
		if data[index+1] == '\\' {
			index++
			continue
		}
		if data[index+1] != 'u' {
			continue
		}
		value, err := strconv.ParseUint(string(data[index+2:index+6]), 16, 16)
		if err != nil || value < 0xd800 || value > 0xdfff {
			continue
		}
		if value <= 0xdbff && index+11 < len(data) && data[index+6] == '\\' && data[index+7] == 'u' {
			next, nextErr := strconv.ParseUint(string(data[index+8:index+12]), 16, 16)
			if nextErr == nil && next >= 0xdc00 && next <= 0xdfff {
				index += 11
				continue
			}
		}
		return true
	}
	return false
}

func canonicalJSON(value any) ([]byte, error) {
	if err := validateUTF8Value(reflect.ValueOf(value), make(map[utf8Visit]struct{})); err != nil {
		return nil, err
	}
	marshaled, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(marshaled))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := writeCanonicalJSON(&output, decoded); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

type utf8Visit struct {
	typeOf  reflect.Type
	pointer uintptr
}

func validateUTF8Value(value reflect.Value, seen map[utf8Visit]struct{}) error {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil
		}
		return validateUTF8Value(value.Elem(), seen)
	}
	switch value.Kind() {
	case reflect.String:
		if !utf8.ValidString(value.String()) {
			return errors.New("invalid UTF-8 string")
		}
	case reflect.Pointer:
		if value.IsNil() {
			return nil
		}
		visit := utf8Visit{typeOf: value.Type(), pointer: value.Pointer()}
		if _, ok := seen[visit]; ok {
			return nil
		}
		seen[visit] = struct{}{}
		return validateUTF8Value(value.Elem(), seen)
	case reflect.Map:
		if value.IsNil() {
			return nil
		}
		visit := utf8Visit{typeOf: value.Type(), pointer: value.Pointer()}
		if _, ok := seen[visit]; ok {
			return nil
		}
		seen[visit] = struct{}{}
		iterator := value.MapRange()
		for iterator.Next() {
			if err := validateUTF8Value(iterator.Key(), seen); err != nil {
				return err
			}
			if err := validateUTF8Value(iterator.Value(), seen); err != nil {
				return err
			}
		}
	case reflect.Slice:
		if value.IsNil() {
			return nil
		}
		visit := utf8Visit{typeOf: value.Type(), pointer: value.Pointer()}
		if _, ok := seen[visit]; ok {
			return nil
		}
		seen[visit] = struct{}{}
		fallthrough
	case reflect.Array:
		for index := 0; index < value.Len(); index++ {
			if err := validateUTF8Value(value.Index(index), seen); err != nil {
				return err
			}
		}
	case reflect.Struct:
		typeOf := value.Type()
		for index := 0; index < value.NumField(); index++ {
			if typeOf.Field(index).PkgPath != "" {
				continue
			}
			if err := validateUTF8Value(value.Field(index), seen); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeCanonicalJSON(output *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		output.WriteString("null")
	case bool:
		if typed {
			output.WriteString("true")
		} else {
			output.WriteString("false")
		}
	case string:
		return writeJSONString(output, typed)
	case json.Number:
		lexical := typed.String()
		if lexical == "-0" || strings.ContainsAny(lexical, ".eE") {
			return fmt.Errorf("canonical JSON permits finite integers only: %s", lexical)
		}
		integer, err := strconv.ParseInt(lexical, 10, 64)
		if err != nil {
			return err
		}
		output.WriteString(strconv.FormatInt(integer, 10))
	case []any:
		output.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				output.WriteByte(',')
			}
			if err := writeCanonicalJSON(output, item); err != nil {
				return err
			}
		}
		output.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool { return bytes.Compare([]byte(keys[i]), []byte(keys[j])) < 0 })
		output.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				output.WriteByte(',')
			}
			if err := writeJSONString(output, key); err != nil {
				return err
			}
			output.WriteByte(':')
			if err := writeCanonicalJSON(output, typed[key]); err != nil {
				return err
			}
		}
		output.WriteByte('}')
	default:
		return fmt.Errorf("unsupported canonical JSON type %T", value)
	}
	return nil
}

func writeJSONString(output *bytes.Buffer, value string) error {
	if !utf8.ValidString(value) {
		return errors.New("invalid UTF-8 string")
	}
	output.WriteByte('"')
	for _, character := range value {
		switch character {
		case '"', '\\':
			output.WriteByte('\\')
			output.WriteRune(character)
		case '\b':
			output.WriteString("\\b")
		case '\t':
			output.WriteString("\\t")
		case '\n':
			output.WriteString("\\n")
		case '\f':
			output.WriteString("\\f")
		case '\r':
			output.WriteString("\\r")
		default:
			if character < 0x20 {
				fmt.Fprintf(output, "\\u%04x", character)
			} else {
				output.WriteRune(character)
			}
		}
	}
	output.WriteByte('"')
	return nil
}

func cloneJSON[T any](value T) T {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var cloned T
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		panic(err)
	}
	return cloned
}

func nonNilSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}
