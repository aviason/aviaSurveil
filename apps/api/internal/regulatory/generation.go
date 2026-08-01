// Package regulatory owns the bounded, non-authoritative generation seam.
// It deliberately has no network, credential, prompt, HTTP, or publication API.
package regulatory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const GeneratedDraft = "GENERATED_DRAFT"

var (
	ErrInvalidRequest   = errors.New("invalid bounded regulatory generation request")
	ErrInvalidCandidate = errors.New("invalid governed regulatory candidate bundle")
	ErrBlockedAuthority = errors.New("source-bound OPS/AOC pilot authority remains unresolved")
)

type SourceSnapshot struct {
	SourceSnapshotID string   `json:"sourceSnapshotId"`
	SourceHash       string   `json:"sourceHash"`
	ClauseIDs        []string `json:"clauseIds"`
	ClauseLocators   []string `json:"clauseLocators"`
}
type Target struct {
	TargetID string `json:"targetId"`
	Kind     string `json:"kind"`
}
type CrosswalkPartition struct {
	PartitionID  string   `json:"partitionId"`
	StableRowIDs []string `json:"stableRowIds"`
}
type UnresolvedSourceGap struct {
	GapID  string `json:"gapId"`
	Reason string `json:"reason"`
}
type GenerationRequest struct {
	SchemaVersion               string                `json:"schemaVersion"`
	RequestID                   string                `json:"requestId"`
	OrganizationID              string                `json:"organizationId"`
	ServiceProviderScopeFactIDs []string              `json:"serviceProviderScopeFactIds"`
	ServiceProviderTypes        []string              `json:"serviceProviderTypes"`
	ProviderCatalogVersion      string                `json:"providerCatalogVersion"`
	InspectionType              string                `json:"inspectionType"`
	Target                      Target                `json:"target"`
	SourceSnapshots             []SourceSnapshot      `json:"sourceSnapshots"`
	SecondaryCrosswalkPartition CrosswalkPartition    `json:"secondaryCrosswalkPartition"`
	UnresolvedSourceGaps        []UnresolvedSourceGap `json:"unresolvedSourceGaps"`
	GenerationPolicyVersion     string                `json:"generationPolicyVersion"`
	ProviderID                  string                `json:"providerId"`
	ProviderVersion             string                `json:"providerVersion"`
	RequestedOutputs            []string              `json:"requestedOutputs"`
	CanonicalInputDigest        string                `json:"canonicalInputDigest"`
}
type ValidatedGenerationRequest struct {
	request                 GenerationRequest
	sourceAuthorityResolved bool
}

func (request ValidatedGenerationRequest) Request() GenerationRequest { return request.request }

type Citation struct {
	SourceSnapshotID string `json:"sourceSnapshotId"`
	SourceHash       string `json:"sourceHash"`
	ClauseID         string `json:"clauseId"`
	Locator          string `json:"locator"`
}
type SourceGap struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type QuestionOrigin string

const (
	RegulatoryTraceOrigin            QuestionOrigin = "REGULATORY_TRACE"
	ExistingChecklistCandidateOrigin QuestionOrigin = "EXISTING_CHECKLIST_CANDIDATE"
	HybridReconciledOrigin           QuestionOrigin = "HYBRID_RECONCILED"

	SourceMappingRequired = "SOURCE_MAPPING_REQUIRED"
)

type ScopeGuardrails struct {
	MandatoryControl           bool `json:"mandatoryControl"`
	SafetyCritical             bool `json:"safetyCritical"`
	UnknownHistory             bool `json:"unknownHistory"`
	SourceChanged              bool `json:"sourceChanged"`
	OverdueControl             bool `json:"overdueControl"`
	AutomaticDeferralPermitted bool `json:"automaticDeferralPermitted"`
}

type ScopeRecommendation struct {
	Classification          string          `json:"classification"`
	InputSignals            []string        `json:"inputSignals"`
	OperationalHistoryBasis string          `json:"operationalHistoryBasis"`
	Rationale               string          `json:"rationale"`
	Guardrails              ScopeGuardrails `json:"guardrails"`
	ApprovalReviewState     string          `json:"approvalReviewState"`
	AutomaticDeferral       bool            `json:"automaticDeferral"`
}

type RegulatoryTrace struct {
	State                         string   `json:"state"`
	SourceIdentity                string   `json:"sourceIdentity,omitempty"`
	SourceTitle                   string   `json:"sourceTitle,omitempty"`
	ImmutableVersion              string   `json:"immutableVersion,omitempty"`
	SHA256                        string   `json:"sha256,omitempty"`
	Locator                       string   `json:"locator,omitempty"`
	Page                          string   `json:"page,omitempty"`
	Section                       string   `json:"section,omitempty"`
	Clause                        string   `json:"clause,omitempty"`
	SourceType                    string   `json:"sourceType,omitempty"`
	Applicability                 string   `json:"applicability,omitempty"`
	NationalReference             string   `json:"nationalReference,omitempty"`
	ControlledCAAProcedureMapping string   `json:"controlledCaaProcedureMapping,omitempty"`
	VerificationObjective         string   `json:"verificationObjective,omitempty"`
	ExpectedEvidence              []string `json:"expectedEvidence,omitempty"`
	CurrentnessState              string   `json:"currentnessState,omitempty"`
	TechnicalReviewState          string   `json:"technicalReviewState,omitempty"`
}

// QuestionReconciliation keeps the historical checklist material visibly
// separate from the approved source chain. The legacy values are candidate
// input only; the current values must match the governed question views that
// are eligible for Department Manager review and a separate publication
// decision.
type QuestionReconciliation struct {
	LegacyQuestionID           string   `json:"legacyQuestionId"`
	LegacyWording              string   `json:"legacyWording"`
	LegacyOperationalIntent    string   `json:"legacyOperationalIntent"`
	LegacyResultHistory        string   `json:"legacyResultHistory"`
	LegacyExpectedEvidence     []string `json:"legacyExpectedEvidence"`
	LegacyApplicability        string   `json:"legacyApplicability"`
	LegacyScopeClassification  string   `json:"legacyScopeClassification"`
	CurrentWording             string   `json:"currentWording"`
	CurrentExpectedEvidence    []string `json:"currentExpectedEvidence"`
	CurrentApplicability       string   `json:"currentApplicability"`
	CurrentScopeClassification string   `json:"currentScopeClassification"`
	WordingChanged             bool     `json:"wordingChanged"`
	EvidenceChanged            bool     `json:"evidenceChanged"`
	ApplicabilityChanged       bool     `json:"applicabilityChanged"`
	ScopeChanged               bool     `json:"scopeChanged"`
}

type ComplianceMapping struct {
	MappingID     string     `json:"mappingId"`
	Requirement   string     `json:"requirement"`
	Relationship  string     `json:"relationship"`
	Applicability string     `json:"applicability"`
	Citations     []Citation `json:"citations"`
	SourceGap     *SourceGap `json:"sourceGap"`
	Rationale     string     `json:"rationale"`
}
type ChecklistQuestion struct {
	QuestionID          string                  `json:"questionId"`
	MappingIDs          []string                `json:"mappingIds"`
	Prompt              string                  `json:"prompt"`
	Citations           []Citation              `json:"citations"`
	VerificationMethod  string                  `json:"verificationMethod"`
	ExpectedEvidence    []string                `json:"expectedEvidence"`
	AllowedAnswers      []string                `json:"allowedAnswers"`
	MandatoryCore       bool                    `json:"mandatoryCore"`
	SafetyCritical      bool                    `json:"safetyCritical"`
	Origin              QuestionOrigin          `json:"origin"`
	ScopeRecommendation ScopeRecommendation     `json:"scopeRecommendation"`
	RegulatoryTrace     RegulatoryTrace         `json:"regulatoryTrace"`
	Reconciliation      *QuestionReconciliation `json:"reconciliation"`
}
type InspectionChecklist struct {
	ChecklistID string              `json:"checklistId"`
	Questions   []ChecklistQuestion `json:"questions"`
}

// SourceCurrentnessBinding is immutable candidate input that proves the
// source snapshot was explicitly activated in the currentness ledger. A
// predecessor is present only for a source change; an initial baseline has no
// predecessor. Source rows alone are intentionally not authority/currentness
// declarations.
type SourceCurrentnessBinding struct {
	CurrentSourceSnapshotID  string `json:"currentSourceSnapshotId"`
	CurrentSourceHash        string `json:"currentSourceHash"`
	PreviousSourceSnapshotID string `json:"previousSourceSnapshotId,omitempty"`
	PreviousSourceHash       string `json:"previousSourceHash,omitempty"`
}

type CandidateBundle struct {
	SchemaVersion       string                    `json:"schemaVersion"`
	CandidateBundleID   string                    `json:"candidateBundleId"`
	GenerationRunID     string                    `json:"generationRunId"`
	Status              string                    `json:"status"`
	GenerationRequest   GenerationRequest         `json:"generationRequest"`
	InputDigest         string                    `json:"inputDigest"`
	OutputDigest        string                    `json:"outputDigest"`
	ComplianceMappings  []ComplianceMapping       `json:"complianceMappings"`
	InspectionChecklist InspectionChecklist       `json:"inspectionChecklist"`
	SourceCurrentness   *SourceCurrentnessBinding `json:"sourceCurrentness,omitempty"`
}

// RegulatoryGenerationProvider is intentionally narrow: callers cannot receive
// arbitrary model text, and providers cannot return an unvalidated authority state.
type RegulatoryGenerationProvider interface {
	Generate(context.Context, ValidatedGenerationRequest) (CandidateBundle, error)
}
type DeterministicFixtureProvider struct{}

func NewDeterministicFixtureProvider() DeterministicFixtureProvider {
	return DeterministicFixtureProvider{}
}
func (DeterministicFixtureProvider) Generate(_ context.Context, request ValidatedGenerationRequest) (CandidateBundle, error) {
	if !request.sourceAuthorityResolved {
		return CandidateBundle{}, ErrBlockedAuthority
	}
	bundle := SyntheticCandidateBundle()
	if err := ValidateCandidateBundle(bundle, request); err != nil {
		return CandidateBundle{}, err
	}
	return bundle, nil
}

type ImportedResultProvider struct{ candidate CandidateBundle }

func NewImportedResultProvider(candidate CandidateBundle) ImportedResultProvider {
	return ImportedResultProvider{candidate: candidate}
}
func (provider ImportedResultProvider) Generate(_ context.Context, request ValidatedGenerationRequest) (CandidateBundle, error) {
	if !request.sourceAuthorityResolved {
		return CandidateBundle{}, ErrBlockedAuthority
	}
	if err := ValidateCandidateBundle(provider.candidate, request); err != nil {
		return CandidateBundle{}, err
	}
	return provider.candidate, nil
}

func CanonicalSHA256(value any) (string, error) {
	encoded, err := canonicalJSON(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(encoded))
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}
func canonicalJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return "", err
	}
	return canonicalValue(decoded)
}
func canonicalValue(value any) (string, error) {
	switch typed := value.(type) {
	case nil, bool, float64, string:
		encoded, err := json.Marshal(typed)
		return string(encoded), err
	case []any:
		parts := make([]string, len(typed))
		for index, item := range typed {
			rendered, err := canonicalValue(item)
			if err != nil {
				return "", err
			}
			parts[index] = rendered
		}
		return "[" + join(parts, ", ") + "]", nil
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(left, right int) bool {
			return len(keys[left]) < len(keys[right]) || (len(keys[left]) == len(keys[right]) && keys[left] < keys[right])
		})
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			rendered, err := canonicalValue(typed[key])
			if err != nil {
				return "", err
			}
			quoted, _ := json.Marshal(key)
			parts = append(parts, string(quoted)+": "+rendered)
		}
		return "{" + join(parts, ", ") + "}", nil
	default:
		return "", fmt.Errorf("unsupported canonical JSON type %T", value)
	}
}
func join(values []string, separator string) string {
	result := ""
	for index, value := range values {
		if index > 0 {
			result += separator
		}
		result += value
	}
	return result
}
func requestDigest(request GenerationRequest) (string, error) {
	unsigned := request
	unsigned.CanonicalInputDigest = ""
	encoded, err := canonicalJSON(unsigned)
	if err != nil {
		return "", err
	}
	var decoded map[string]any
	_ = json.Unmarshal([]byte(encoded), &decoded)
	delete(decoded, "canonicalInputDigest")
	return CanonicalSHA256(decoded)
}
func candidateContentProjection(bundle CandidateBundle) map[string]any {
	checklist := bundle.InspectionChecklist
	checklist.Questions = append([]ChecklistQuestion(nil), checklist.Questions...)
	for index, question := range checklist.Questions {
		if question.RegulatoryTrace.State != SourceMappingRequired {
			continue
		}
		// Source-gap review states are server-derived projections. They may be
		// carried by transport views (for example NOT_AVAILABLE), but they do
		// not alter the immutable candidate content digest.
		question.RegulatoryTrace.TechnicalReviewState = ""
		checklist.Questions[index] = question
	}
	return map[string]any{"complianceMappings": bundle.ComplianceMappings, "inspectionChecklist": checklist}
}

func candidateOutputDigest(bundle CandidateBundle) (string, error) {
	return CanonicalSHA256(candidateContentProjection(bundle))
}

func candidateOutputArtifact(bundle CandidateBundle) ([]byte, error) {
	return json.Marshal(candidateContentProjection(bundle))
}

func ValidateRequest(request GenerationRequest, sourceAuthorityResolved bool) (ValidatedGenerationRequest, error) {
	if !isPinnedSyntheticRequest(request) {
		return ValidatedGenerationRequest{}, ErrInvalidRequest
	}
	digest, err := requestDigest(request)
	if err != nil || request.CanonicalInputDigest != digest {
		return ValidatedGenerationRequest{}, ErrInvalidRequest
	}
	return ValidatedGenerationRequest{request: request, sourceAuthorityResolved: sourceAuthorityResolved}, nil
}
func ValidateCandidateBundle(bundle CandidateBundle, request ValidatedGenerationRequest) error {
	if bundle.SchemaVersion != "1.0.0" || bundle.CandidateBundleID == "" || bundle.GenerationRunID == "" || bundle.Status != GeneratedDraft || !sameRequest(bundle.GenerationRequest, request.request) || bundle.InputDigest != request.request.CanonicalInputDigest || len(bundle.ComplianceMappings) == 0 || len(bundle.InspectionChecklist.Questions) == 0 {
		return ErrInvalidCandidate
	}
	digest, err := candidateOutputDigest(bundle)
	if err != nil || bundle.OutputDigest != digest {
		return ErrInvalidCandidate
	}
	mappings := map[string]bool{}
	for _, mapping := range bundle.ComplianceMappings {
		if mapping.MappingID == "" || mappings[mapping.MappingID] || mapping.Requirement == "" || mapping.Rationale == "" || !oneOf(mapping.Relationship, "ADDRESSES", "PARTIALLY_ADDRESSES", "NOT_ADDRESSED", "CONTEXT_ONLY") || !oneOf(mapping.Applicability, "DIRECT", "CONDITIONAL", "CONTEXTUAL") || (mapping.SourceGap != nil && (mapping.SourceGap.Status != "UNRESOLVED" || strings.TrimSpace(mapping.SourceGap.Reason) == "")) || !supportedSyntheticMapping(mapping, request.request) || !validCitations(mapping.Citations, request.request) {
			return ErrInvalidCandidate
		}
		mappings[mapping.MappingID] = true
	}
	questions := map[string]bool{}
	for _, question := range bundle.InspectionChecklist.Questions {
		requiresCitation := question.RegulatoryTrace.State != SourceMappingRequired
		if strings.TrimSpace(question.QuestionID) == "" || questions[question.QuestionID] || strings.TrimSpace(question.Prompt) == "" || strings.TrimSpace(question.VerificationMethod) == "" || len(question.MappingIDs) == 0 || len(question.ExpectedEvidence) == 0 || hasBlank(question.ExpectedEvidence) || !uniqueStrings(question.MappingIDs) || !sameStrings(question.AllowedAnswers, allowedAnswers) || !supportedSyntheticQuestion(question, request.request) || (requiresCitation && !validCitations(question.Citations, request.request)) || !validQuestionGovernance(question) {
			return ErrInvalidCandidate
		}
		for _, mappingID := range question.MappingIDs {
			if !mappings[mappingID] {
				return ErrInvalidCandidate
			}
		}
		questions[question.QuestionID] = true
	}
	if !validSourceCurrentnessBinding(bundle) {
		return ErrInvalidCandidate
	}
	return nil
}

func validSourceCurrentnessBinding(bundle CandidateBundle) bool {
	resolvedTrace := false
	sourceChanged := false
	for _, question := range bundle.InspectionChecklist.Questions {
		if question.RegulatoryTrace.State == "RESOLVED" {
			resolvedTrace = true
			sourceChanged = sourceChanged || question.ScopeRecommendation.Guardrails.SourceChanged
		}
	}
	if !resolvedTrace {
		return bundle.SourceCurrentness == nil
	}
	if bundle.SourceCurrentness == nil || len(bundle.GenerationRequest.SourceSnapshots) != 1 {
		return false
	}
	binding := bundle.SourceCurrentness
	current := bundle.GenerationRequest.SourceSnapshots[0]
	if strings.TrimSpace(binding.CurrentSourceSnapshotID) == "" ||
		!strings.HasPrefix(binding.CurrentSourceHash, "sha256:") ||
		binding.CurrentSourceSnapshotID != current.SourceSnapshotID ||
		binding.CurrentSourceHash != current.SourceHash {
		return false
	}
	hasPreviousID := strings.TrimSpace(binding.PreviousSourceSnapshotID) != ""
	hasPreviousHash := strings.TrimSpace(binding.PreviousSourceHash) != ""
	if hasPreviousID != hasPreviousHash || (hasPreviousHash && !strings.HasPrefix(binding.PreviousSourceHash, "sha256:")) {
		return false
	}
	if hasPreviousID && binding.PreviousSourceSnapshotID == binding.CurrentSourceSnapshotID {
		return false
	}
	return sourceChanged == hasPreviousID
}

func validQuestionGovernance(question ChecklistQuestion) bool {
	if !oneOf(string(question.Origin), string(RegulatoryTraceOrigin), string(ExistingChecklistCandidateOrigin), string(HybridReconciledOrigin)) {
		return false
	}
	scope := question.ScopeRecommendation
	if !oneOf(scope.Classification, "MANDATORY_CORE", "FOCUSED_FULL", "ROTATIONAL_SAMPLE", "DEFER_ELIGIBLE") ||
		len(scope.InputSignals) == 0 || hasBlank(scope.InputSignals) ||
		strings.TrimSpace(scope.OperationalHistoryBasis) == "" ||
		strings.TrimSpace(scope.Rationale) == "" ||
		scope.ApprovalReviewState != "TECHNICAL_REVIEW_REQUIRED" {
		return false
	}
	guardrails := scope.Guardrails
	if question.MandatoryCore != guardrails.MandatoryControl || question.SafetyCritical != guardrails.SafetyCritical {
		return false
	}
	if scope.AutomaticDeferral && (guardrails.MandatoryControl || guardrails.SafetyCritical || guardrails.UnknownHistory || guardrails.SourceChanged || guardrails.OverdueControl || !guardrails.AutomaticDeferralPermitted) {
		return false
	}
	if question.Origin == HybridReconciledOrigin {
		if !validQuestionReconciliation(question) {
			return false
		}
	} else if question.Reconciliation != nil {
		return false
	}
	trace := question.RegulatoryTrace
	if question.Origin == ExistingChecklistCandidateOrigin && trace.State != SourceMappingRequired {
		return false
	}
	if question.Origin == RegulatoryTraceOrigin && trace.State != "RESOLVED" {
		return false
	}
	if trace.State == SourceMappingRequired {
		// A literal gap is a candidate-only repair state, not a partial trace.
		// It may retain historical wording only under the explicit legacy origin;
		// any citation would look authoritative while omitting the required
		// immutable version/currentness chain.
		return question.Origin == ExistingChecklistCandidateOrigin &&
			!scope.AutomaticDeferral && question.Citations != nil && len(question.Citations) == 0 &&
			isLiteralSourceMappingRequired(trace)
	}
	if trace.State != "RESOLVED" ||
		strings.TrimSpace(trace.SourceIdentity) == "" || strings.TrimSpace(trace.SourceTitle) == "" ||
		strings.TrimSpace(trace.ImmutableVersion) == "" || !strings.HasPrefix(trace.SHA256, "sha256:") ||
		strings.TrimSpace(trace.Locator) == "" || strings.TrimSpace(trace.Page) == "" ||
		strings.TrimSpace(trace.Section) == "" || strings.TrimSpace(trace.Clause) == "" ||
		strings.TrimSpace(trace.SourceType) == "" || strings.TrimSpace(trace.Applicability) == "" ||
		strings.TrimSpace(trace.NationalReference) == "" || strings.TrimSpace(trace.ControlledCAAProcedureMapping) == "" ||
		strings.TrimSpace(trace.VerificationObjective) == "" || len(trace.ExpectedEvidence) == 0 || hasBlank(trace.ExpectedEvidence) ||
		!sameStrings(trace.ExpectedEvidence, question.ExpectedEvidence) ||
		!oneOf(trace.CurrentnessState, "CURRENT", "STALE") ||
		trace.TechnicalReviewState != "TECHNICAL_REVIEW_REQUIRED" {
		return false
	}
	if len(question.Citations) != 1 || question.Citations[0].SourceHash != trace.SHA256 ||
		question.Citations[0].ClauseID != trace.Clause || question.Citations[0].Locator != trace.Locator {
		return false
	}
	return true
}

func isLiteralSourceMappingRequired(trace RegulatoryTrace) bool {
	technicalReviewState := strings.TrimSpace(trace.TechnicalReviewState)
	return trace.State == SourceMappingRequired &&
		strings.TrimSpace(trace.SourceIdentity) == "" &&
		strings.TrimSpace(trace.SourceTitle) == "" &&
		strings.TrimSpace(trace.ImmutableVersion) == "" &&
		strings.TrimSpace(trace.SHA256) == "" &&
		strings.TrimSpace(trace.Locator) == "" &&
		strings.TrimSpace(trace.Page) == "" &&
		strings.TrimSpace(trace.Section) == "" &&
		strings.TrimSpace(trace.Clause) == "" &&
		strings.TrimSpace(trace.SourceType) == "" &&
		strings.TrimSpace(trace.Applicability) == "" &&
		strings.TrimSpace(trace.NationalReference) == "" &&
		strings.TrimSpace(trace.ControlledCAAProcedureMapping) == "" &&
		strings.TrimSpace(trace.VerificationObjective) == "" &&
		len(trace.ExpectedEvidence) == 0 &&
		strings.TrimSpace(trace.CurrentnessState) == "" &&
		(technicalReviewState == "" || technicalReviewState == "NOT_AVAILABLE")
}

func validQuestionReconciliation(question ChecklistQuestion) bool {
	comparison := question.Reconciliation
	if comparison == nil || strings.TrimSpace(comparison.LegacyQuestionID) == "" ||
		strings.TrimSpace(comparison.LegacyWording) == "" ||
		strings.TrimSpace(comparison.LegacyOperationalIntent) == "" ||
		strings.TrimSpace(comparison.LegacyResultHistory) == "" ||
		len(comparison.LegacyExpectedEvidence) == 0 || hasBlank(comparison.LegacyExpectedEvidence) ||
		strings.TrimSpace(comparison.LegacyApplicability) == "" ||
		strings.TrimSpace(comparison.LegacyScopeClassification) == "" ||
		comparison.CurrentWording != question.Prompt ||
		!sameStrings(comparison.CurrentExpectedEvidence, question.ExpectedEvidence) ||
		comparison.CurrentApplicability != question.RegulatoryTrace.Applicability ||
		comparison.CurrentScopeClassification != question.ScopeRecommendation.Classification {
		return false
	}
	return comparison.WordingChanged || comparison.EvidenceChanged ||
		comparison.ApplicabilityChanged || comparison.ScopeChanged
}

func validCitations(citations []Citation, request GenerationRequest) bool {
	if len(citations) == 0 {
		return false
	}
	for _, citation := range citations {
		if citation.SourceSnapshotID != request.SourceSnapshots[0].SourceSnapshotID || citation.SourceHash != request.SourceSnapshots[0].SourceHash || citation.ClauseID != request.SourceSnapshots[0].ClauseIDs[0] || citation.Locator != request.SourceSnapshots[0].ClauseLocators[0] {
			return false
		}
	}
	return true
}
func sameRequest(left, right GenerationRequest) bool {
	a, _ := canonicalJSON(left)
	b, _ := canonicalJSON(right)
	return a == b
}
func uniqueStrings(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}
func hasBlank(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return true
		}
	}
	return false
}
func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
func supportedSyntheticMapping(mapping ComplianceMapping, request GenerationRequest) bool {
	return isPinnedSyntheticRequest(request) && mapping.SourceGap == nil && mapping.Requirement == syntheticSupportedRequirement && mapping.Rationale == syntheticSupportedRationale
}
func supportedSyntheticQuestion(question ChecklistQuestion, request GenerationRequest) bool {
	if !isPinnedSyntheticRequest(request) {
		return false
	}
	switch request.RequestID {
	case "GENREQ-SYNTHETIC-LEGACY-CHECKLIST-0003":
		return question.Prompt == syntheticLegacyCandidateQuestion &&
			question.VerificationMethod == syntheticLegacyCandidateVerificationMethod &&
			sameStrings(question.ExpectedEvidence, syntheticLegacyCandidateExpectedEvidence)
	default:
		return question.Prompt == syntheticSupportedQuestion &&
			question.VerificationMethod == syntheticSupportedVerificationMethod &&
			sameStrings(question.ExpectedEvidence, syntheticSupportedExpectedEvidence)
	}
}
func sameStrings(left, right []string) bool {
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

const syntheticSourceHash = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
const syntheticImpactSourceHash = "sha256:4444444444444444444444444444444444444444444444444444444444444444"

const syntheticSupportedRequirement = "Synthetic controlled training requirement for verifying ramp safety evidence."
const syntheticSupportedRationale = "This synthetic source is test-profile-only and proves the bounded import workflow without asserting real authority."
const syntheticEditedRationale = "Synthetic test-profile rationale reviewed by an Admin without changing the controlled source claim."
const syntheticSupportedQuestion = "Can the inspector reconcile the synthetic ramp-safety evidence with the controlled synthetic requirement?"
const syntheticSupportedVerificationMethod = "Physical observation and controlled-record reconciliation"
const syntheticLegacyCandidateQuestion = "Historical checklist candidate: is the cabin/ramp safety control recorded for later source mapping?"
const syntheticLegacyCandidateVerificationMethod = "Historical candidate review only; no validation or execution claim"

var syntheticSupportedExpectedEvidence = []string{"Synthetic inspection observation", "Synthetic controlled record"}
var syntheticLegacyCandidateExpectedEvidence = []string{"Synthetic historical checklist wording", "Synthetic historical result record"}

var allowedAnswers = []string{"COMPLIANT", "NON_COMPLIANT", "OBSERVATION", "NOT_APPLICABLE", "NOT_CHECKED"}

func syntheticRequest() GenerationRequest {
	request := GenerationRequest{SchemaVersion: "1.0.0", RequestID: "GENREQ-SYNTHETIC-OPS-AOC-0001", OrganizationID: "ORG-SYNTHETIC-AOC", ServiceProviderScopeFactIDs: []string{"SCOPE-SYNTHETIC-AOC"}, ServiceProviderTypes: []string{"AIR_OPERATOR"}, ProviderCatalogVersion: "1.0.0", InspectionType: "RAMP_INSPECTION", Target: Target{TargetID: "TARGET-SYNTHETIC-AOC", Kind: "ORGANIZATION"}, SourceSnapshots: []SourceSnapshot{{SourceSnapshotID: "SOURCE-SYNTHETIC-OPS-AOC", SourceHash: syntheticSourceHash, ClauseIDs: []string{"CLAUSE-SYNTHETIC-OPS-AOC-1"}, ClauseLocators: []string{"Synthetic OPS/AOC 1"}}}, SecondaryCrosswalkPartition: CrosswalkPartition{PartitionID: "PARTITION-SYNTHETIC-INPUT", StableRowIDs: []string{"CC:SYNTHETIC:OPS:AOC:1"}}, UnresolvedSourceGaps: []UnresolvedSourceGap{}, GenerationPolicyVersion: "regulatory-checklist-v1", ProviderID: "deterministic-regulatory-fixture", ProviderVersion: "1.0.0", RequestedOutputs: []string{"COMPLIANCE_MAPPING", "INSPECTION_CHECKLIST"}}
	request.CanonicalInputDigest, _ = requestDigest(request)
	return request
}

func syntheticImpactRequest() GenerationRequest {
	request := syntheticRequest()
	request.RequestID = "GENREQ-SYNTHETIC-OPS-AOC-IMPACT-0002"
	request.SourceSnapshots = []SourceSnapshot{{SourceSnapshotID: "SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2", SourceHash: syntheticImpactSourceHash, ClauseIDs: []string{"CLAUSE-SYNTHETIC-OPS-AOC-IMPACT-2"}, ClauseLocators: []string{"Synthetic OPS/AOC impact 2"}}}
	request.SecondaryCrosswalkPartition = CrosswalkPartition{PartitionID: "PARTITION-SYNTHETIC-IMPACT-INPUT", StableRowIDs: []string{"CC:SYNTHETIC:OPS:AOC:IMPACT:2"}}
	request.CanonicalInputDigest, _ = requestDigest(request)
	return request
}

// syntheticLegacyCandidateRequest is an internal fixture for an old checklist
// supplied only as a candidate input. Its current source context exists so a
// reviewer can repair the question, but the question itself deliberately has
// no regulatory mapping and therefore cannot leave Draft.
func syntheticLegacyCandidateRequest() GenerationRequest {
	request := syntheticRequest()
	request.RequestID = "GENREQ-SYNTHETIC-LEGACY-CHECKLIST-0003"
	// Historical wording is candidate input, not a second evaluation universe.
	// It reuses the frozen, current generation-input partition and remains
	// non-authoritative because its question is explicitly SOURCE_MAPPING_REQUIRED.
	request.SecondaryCrosswalkPartition = CrosswalkPartition{
		PartitionID:  "PARTITION-SYNTHETIC-INPUT",
		StableRowIDs: []string{"CC:SYNTHETIC:OPS:AOC:1"},
	}
	request.CanonicalInputDigest, _ = requestDigest(request)
	return request
}

// syntheticHybridReconciledRequest binds a new candidate to the current V2
// controlled source chain. It is intentionally a new request/root rather than
// a mutation of the legacy candidate or any historical published checklist.
func syntheticHybridReconciledRequest() GenerationRequest {
	request := syntheticImpactRequest()
	request.RequestID = "GENREQ-SYNTHETIC-HYBRID-RECONCILED-0004"
	request.CanonicalInputDigest, _ = requestDigest(request)
	return request
}

// isPinnedSyntheticRequest permits only the explicit internal test
// profiles. It does not accept a real source, a free-form update, or an
// unconfirmed OPS/AOC authority claim.
func isPinnedSyntheticRequest(request GenerationRequest) bool {
	if request.SchemaVersion != "1.0.0" || request.RequestID == "" || request.OrganizationID != "ORG-SYNTHETIC-AOC" || !sameStrings(request.ServiceProviderScopeFactIDs, []string{"SCOPE-SYNTHETIC-AOC"}) || !sameStrings(request.ServiceProviderTypes, []string{"AIR_OPERATOR"}) || request.ProviderCatalogVersion != "1.0.0" || request.InspectionType != "RAMP_INSPECTION" || request.Target != (Target{TargetID: "TARGET-SYNTHETIC-AOC", Kind: "ORGANIZATION"}) || len(request.SourceSnapshots) != 1 || len(request.UnresolvedSourceGaps) != 0 || request.GenerationPolicyVersion != "regulatory-checklist-v1" || request.ProviderID != "deterministic-regulatory-fixture" || request.ProviderVersion != "1.0.0" || !sameStrings(request.RequestedOutputs, []string{"COMPLIANCE_MAPPING", "INSPECTION_CHECKLIST"}) {
		return false
	}
	source := request.SourceSnapshots[0]
	return (request.RequestID == "GENREQ-SYNTHETIC-OPS-AOC-0001" && source.SourceSnapshotID == "SOURCE-SYNTHETIC-OPS-AOC" && source.SourceHash == syntheticSourceHash && sameStrings(source.ClauseIDs, []string{"CLAUSE-SYNTHETIC-OPS-AOC-1"}) && sameStrings(source.ClauseLocators, []string{"Synthetic OPS/AOC 1"}) && request.SecondaryCrosswalkPartition.PartitionID == "PARTITION-SYNTHETIC-INPUT" && sameStrings(request.SecondaryCrosswalkPartition.StableRowIDs, []string{"CC:SYNTHETIC:OPS:AOC:1"})) ||
		(request.RequestID == "GENREQ-SYNTHETIC-OPS-AOC-IMPACT-0002" && source.SourceSnapshotID == "SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2" && source.SourceHash == syntheticImpactSourceHash && sameStrings(source.ClauseIDs, []string{"CLAUSE-SYNTHETIC-OPS-AOC-IMPACT-2"}) && sameStrings(source.ClauseLocators, []string{"Synthetic OPS/AOC impact 2"}) && request.SecondaryCrosswalkPartition.PartitionID == "PARTITION-SYNTHETIC-IMPACT-INPUT" && sameStrings(request.SecondaryCrosswalkPartition.StableRowIDs, []string{"CC:SYNTHETIC:OPS:AOC:IMPACT:2"})) ||
		(request.RequestID == "GENREQ-SYNTHETIC-LEGACY-CHECKLIST-0003" && source.SourceSnapshotID == "SOURCE-SYNTHETIC-OPS-AOC" && source.SourceHash == syntheticSourceHash && sameStrings(source.ClauseIDs, []string{"CLAUSE-SYNTHETIC-OPS-AOC-1"}) && sameStrings(source.ClauseLocators, []string{"Synthetic OPS/AOC 1"}) && request.SecondaryCrosswalkPartition.PartitionID == "PARTITION-SYNTHETIC-INPUT" && sameStrings(request.SecondaryCrosswalkPartition.StableRowIDs, []string{"CC:SYNTHETIC:OPS:AOC:1"})) ||
		(request.RequestID == "GENREQ-SYNTHETIC-HYBRID-RECONCILED-0004" && source.SourceSnapshotID == "SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2" && source.SourceHash == syntheticImpactSourceHash && sameStrings(source.ClauseIDs, []string{"CLAUSE-SYNTHETIC-OPS-AOC-IMPACT-2"}) && sameStrings(source.ClauseLocators, []string{"Synthetic OPS/AOC impact 2"}) && request.SecondaryCrosswalkPartition.PartitionID == "PARTITION-SYNTHETIC-IMPACT-INPUT" && sameStrings(request.SecondaryCrosswalkPartition.StableRowIDs, []string{"CC:SYNTHETIC:OPS:AOC:IMPACT:2"}))
}
func SyntheticGenerationRequest() ValidatedGenerationRequest {
	request, err := ValidateRequest(syntheticRequest(), true)
	if err != nil {
		panic(err)
	}
	return request
}

func SyntheticLegacyCandidateGenerationRequest() ValidatedGenerationRequest {
	request, err := ValidateRequest(syntheticLegacyCandidateRequest(), true)
	if err != nil {
		panic(err)
	}
	return request
}

func SyntheticHybridReconciledGenerationRequest() ValidatedGenerationRequest {
	request, err := ValidateRequest(syntheticHybridReconciledRequest(), true)
	if err != nil {
		panic(err)
	}
	return request
}
func RealOPSAOCGenerationRequest() ValidatedGenerationRequest {
	request := syntheticRequest()
	request.RequestID = "GENREQ-OPS-AOC-0001"
	request.OrganizationID = "ORG-FLY-NAMIBIA"
	request.ServiceProviderScopeFactIDs = []string{"SCOPE-OPS-AOC-SOURCE-BOUND"}
	request.Target = Target{TargetID: "TARGET-OPS-AOC-SOURCE-BOUND", Kind: "ORGANIZATION"}
	request.SourceSnapshots = []SourceSnapshot{{SourceSnapshotID: "NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28", SourceHash: "sha256:13fe82d1767320443f91ed61cf7d3b4bba0ea24f217fad45bbd9cae5fc682af2", ClauseIDs: []string{"NCAA-CC-A610-4.2.2.2"}, ClauseLocators: []string{"Annex 6 Part I 4.2.2.2"}}}
	request.SecondaryCrosswalkPartition = CrosswalkPartition{PartitionID: "CC-OPS-TRAIN-1", StableRowIDs: []string{"CC:NAMB:ANNEX6:4.2.2.2"}}
	request.UnresolvedSourceGaps = []UnresolvedSourceGap{{GapID: "CONTROLLED_PROCEDURE", Reason: "The controlled NCAA Operations surveillance/ramp-inspection procedure has not been supplied."}, {GapID: "PART_140_AUTHORITY", Reason: "Current Part 140 authority and supersession require source-owner confirmation."}, {GapID: "PART_127_APPLICABILITY", Reason: "Exact Part 127 operation/configuration applicability requires Department Manager confirmation."}}
	request.UnresolvedSourceGaps = append(request.UnresolvedSourceGaps, UnresolvedSourceGap{
		GapID:  "AMBIGUOUS_OWNERSHIP",
		Reason: "Exact source ownership and controlled-procedure stewardship remain unresolved.",
	})
	request.ProviderID = "imported-result-only"
	request.CanonicalInputDigest, _ = requestDigest(request)
	validated, err := validateRealOPSAOCRequest(request)
	if err != nil {
		panic(err)
	}
	return validated
}

// validateRealOPSAOCRequest records the exact supplied source-bound request.
// It is intentionally separate from the synthetic test profile: validation
// proves the binding, while the false authority flag still blocks generation.
func validateRealOPSAOCRequest(request GenerationRequest) (ValidatedGenerationRequest, error) {
	wantGaps := []UnresolvedSourceGap{
		{GapID: "CONTROLLED_PROCEDURE", Reason: "The controlled NCAA Operations surveillance/ramp-inspection procedure has not been supplied."},
		{GapID: "PART_140_AUTHORITY", Reason: "Current Part 140 authority and supersession require source-owner confirmation."},
		{GapID: "PART_127_APPLICABILITY", Reason: "Exact Part 127 operation/configuration applicability requires Department Manager confirmation."},
		{GapID: "AMBIGUOUS_OWNERSHIP", Reason: "Exact source ownership and controlled-procedure stewardship remain unresolved."},
	}
	if request.SchemaVersion != "1.0.0" || request.RequestID != "GENREQ-OPS-AOC-0001" || request.OrganizationID != "ORG-FLY-NAMIBIA" || !sameStrings(request.ServiceProviderScopeFactIDs, []string{"SCOPE-OPS-AOC-SOURCE-BOUND"}) || !sameStrings(request.ServiceProviderTypes, []string{"AIR_OPERATOR"}) || request.ProviderCatalogVersion != "1.0.0" || request.InspectionType != "RAMP_INSPECTION" || request.Target != (Target{TargetID: "TARGET-OPS-AOC-SOURCE-BOUND", Kind: "ORGANIZATION"}) || len(request.SourceSnapshots) != 1 || request.SourceSnapshots[0].SourceSnapshotID != "NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28" || request.SourceSnapshots[0].SourceHash != "sha256:13fe82d1767320443f91ed61cf7d3b4bba0ea24f217fad45bbd9cae5fc682af2" || !sameStrings(request.SourceSnapshots[0].ClauseIDs, []string{"NCAA-CC-A610-4.2.2.2"}) || !sameStrings(request.SourceSnapshots[0].ClauseLocators, []string{"Annex 6 Part I 4.2.2.2"}) || request.SecondaryCrosswalkPartition.PartitionID != "CC-OPS-TRAIN-1" || !sameStrings(request.SecondaryCrosswalkPartition.StableRowIDs, []string{"CC:NAMB:ANNEX6:4.2.2.2"}) || !sameSourceGaps(request.UnresolvedSourceGaps, wantGaps) || request.GenerationPolicyVersion != "regulatory-checklist-v1" || request.ProviderID != "imported-result-only" || request.ProviderVersion != "1.0.0" || !sameStrings(request.RequestedOutputs, []string{"COMPLIANCE_MAPPING", "INSPECTION_CHECKLIST"}) {
		return ValidatedGenerationRequest{}, ErrInvalidRequest
	}
	digest, err := requestDigest(request)
	if err != nil || request.CanonicalInputDigest != digest {
		return ValidatedGenerationRequest{}, ErrInvalidRequest
	}
	return ValidatedGenerationRequest{request: request, sourceAuthorityResolved: false}, nil
}
func sameSourceGaps(left, right []UnresolvedSourceGap) bool {
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
func SyntheticCandidateBundle() CandidateBundle {
	request := syntheticRequest()
	citation := Citation{SourceSnapshotID: "SOURCE-SYNTHETIC-OPS-AOC", SourceHash: syntheticSourceHash, ClauseID: "CLAUSE-SYNTHETIC-OPS-AOC-1", Locator: "Synthetic OPS/AOC 1"}
	scope, trace := syntheticQuestionGovernance(citation, "SYNTHETIC-OPS-AOC")
	bundle := CandidateBundle{SchemaVersion: "1.0.0", CandidateBundleID: "CAND-SYNTHETIC-OPS-AOC-0001", GenerationRunID: "GENRUN-SYNTHETIC-OPS-AOC-0001", Status: GeneratedDraft, GenerationRequest: request, InputDigest: request.CanonicalInputDigest, ComplianceMappings: []ComplianceMapping{{MappingID: "MAP-SYNTHETIC-OPS-AOC-001", Requirement: syntheticSupportedRequirement, Relationship: "ADDRESSES", Applicability: "DIRECT", Citations: []Citation{citation}, Rationale: syntheticSupportedRationale}}, InspectionChecklist: InspectionChecklist{ChecklistID: "CHK-SYNTHETIC-OPS-AOC-001", Questions: []ChecklistQuestion{{QuestionID: "Q-SYNTHETIC-OPS-AOC-001", MappingIDs: []string{"MAP-SYNTHETIC-OPS-AOC-001"}, Prompt: syntheticSupportedQuestion, Citations: []Citation{citation}, VerificationMethod: syntheticSupportedVerificationMethod, ExpectedEvidence: syntheticSupportedExpectedEvidence, AllowedAnswers: allowedAnswers, MandatoryCore: true, SafetyCritical: true, Origin: RegulatoryTraceOrigin, ScopeRecommendation: scope, RegulatoryTrace: trace}}}, SourceCurrentness: &SourceCurrentnessBinding{CurrentSourceSnapshotID: citation.SourceSnapshotID, CurrentSourceHash: citation.SourceHash}}
	bundle.OutputDigest, _ = candidateOutputDigest(bundle)
	return bundle
}

// SyntheticImpactCandidateBundle is the sole source-change test profile. It
// is a new root candidate, never a revision of an earlier published history.
func SyntheticImpactCandidateBundle() CandidateBundle {
	request := syntheticImpactRequest()
	citation := Citation{SourceSnapshotID: "SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2", SourceHash: syntheticImpactSourceHash, ClauseID: "CLAUSE-SYNTHETIC-OPS-AOC-IMPACT-2", Locator: "Synthetic OPS/AOC impact 2"}
	scope, trace := syntheticQuestionGovernance(citation, "SYNTHETIC-OPS-AOC")
	scope.Guardrails.SourceChanged = true
	bundle := CandidateBundle{SchemaVersion: "1.0.0", CandidateBundleID: "CAND-SYNTHETIC-OPS-AOC-IMPACT-0002", GenerationRunID: "GENRUN-SYNTHETIC-OPS-AOC-IMPACT-0002", Status: GeneratedDraft, GenerationRequest: request, InputDigest: request.CanonicalInputDigest, ComplianceMappings: []ComplianceMapping{{MappingID: "MAP-SYNTHETIC-OPS-AOC-IMPACT-002", Requirement: syntheticSupportedRequirement, Relationship: "ADDRESSES", Applicability: "DIRECT", Citations: []Citation{citation}, Rationale: syntheticSupportedRationale}}, InspectionChecklist: InspectionChecklist{ChecklistID: "CHK-SYNTHETIC-OPS-AOC-IMPACT-002", Questions: []ChecklistQuestion{{QuestionID: "Q-SYNTHETIC-OPS-AOC-IMPACT-002", MappingIDs: []string{"MAP-SYNTHETIC-OPS-AOC-IMPACT-002"}, Prompt: syntheticSupportedQuestion, Citations: []Citation{citation}, VerificationMethod: syntheticSupportedVerificationMethod, ExpectedEvidence: syntheticSupportedExpectedEvidence, AllowedAnswers: allowedAnswers, MandatoryCore: true, SafetyCritical: true, Origin: RegulatoryTraceOrigin, ScopeRecommendation: scope, RegulatoryTrace: trace}}}, SourceCurrentness: &SourceCurrentnessBinding{CurrentSourceSnapshotID: citation.SourceSnapshotID, CurrentSourceHash: citation.SourceHash, PreviousSourceSnapshotID: "SOURCE-SYNTHETIC-OPS-AOC", PreviousSourceHash: syntheticSourceHash}}
	bundle.OutputDigest, _ = candidateOutputDigest(bundle)
	return bundle
}

// SyntheticLegacyChecklistCandidateBundle is the explicit non-authoritative
// import fixture. Its wording, operational intent, and result history are
// candidate inputs only; SOURCE_MAPPING_REQUIRED keeps it repairable in Draft
// and fail-closed everywhere else.
func SyntheticLegacyChecklistCandidateBundle() CandidateBundle {
	request := syntheticLegacyCandidateRequest()
	citation := Citation{SourceSnapshotID: "SOURCE-SYNTHETIC-OPS-AOC", SourceHash: syntheticSourceHash, ClauseID: "CLAUSE-SYNTHETIC-OPS-AOC-1", Locator: "Synthetic OPS/AOC 1"}
	scope := ScopeRecommendation{
		Classification: "FOCUSED_FULL",
		InputSignals: []string{
			"Historical checklist wording is candidate input only.",
			"Unknown result-history quality prevents automatic deferral.",
		},
		OperationalHistoryBasis: "SYNTHETIC_LEGACY_HISTORY_UNKNOWN",
		Rationale:               "Keep the legacy candidate visible for Department Manager repair; it cannot establish regulatory applicability.",
		Guardrails: ScopeGuardrails{
			UnknownHistory:             true,
			AutomaticDeferralPermitted: false,
		},
		ApprovalReviewState: "TECHNICAL_REVIEW_REQUIRED",
		AutomaticDeferral:   false,
	}
	bundle := CandidateBundle{
		SchemaVersion: "1.0.0", CandidateBundleID: "CAND-SYNTHETIC-LEGACY-CHECKLIST-0003", GenerationRunID: "GENRUN-SYNTHETIC-LEGACY-CHECKLIST-0003", Status: GeneratedDraft,
		GenerationRequest: request, InputDigest: request.CanonicalInputDigest,
		ComplianceMappings: []ComplianceMapping{{
			MappingID: "MAP-SYNTHETIC-LEGACY-CANDIDATE-003", Requirement: syntheticSupportedRequirement,
			Relationship: "ADDRESSES", Applicability: "DIRECT", Citations: []Citation{citation}, Rationale: syntheticSupportedRationale,
		}},
		InspectionChecklist: InspectionChecklist{ChecklistID: "CHK-SYNTHETIC-LEGACY-CHECKLIST-003", Questions: []ChecklistQuestion{{
			QuestionID: "Q-SYNTHETIC-LEGACY-CANDIDATE-003", MappingIDs: []string{"MAP-SYNTHETIC-LEGACY-CANDIDATE-003"},
			Prompt: syntheticLegacyCandidateQuestion, VerificationMethod: syntheticLegacyCandidateVerificationMethod,
			Citations: []Citation{}, ExpectedEvidence: syntheticLegacyCandidateExpectedEvidence, AllowedAnswers: allowedAnswers,
			Origin: ExistingChecklistCandidateOrigin, ScopeRecommendation: scope,
			RegulatoryTrace: RegulatoryTrace{State: SourceMappingRequired},
		}}},
	}
	bundle.OutputDigest, _ = candidateOutputDigest(bundle)
	return bundle
}

// SyntheticHybridReconciledCandidateBundle models the only supported migration
// path from the legacy candidate fixture: a fresh immutable candidate uses the
// current V2 controlled source chain while preserving a visible comparison of
// the old candidate input. It never rewrites historical published versions or
// inspection-package snapshots.
func SyntheticHybridReconciledCandidateBundle() CandidateBundle {
	request := syntheticHybridReconciledRequest()
	citation := Citation{SourceSnapshotID: "SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2", SourceHash: syntheticImpactSourceHash, ClauseID: "CLAUSE-SYNTHETIC-OPS-AOC-IMPACT-2", Locator: "Synthetic OPS/AOC impact 2"}
	_, trace := syntheticQuestionGovernance(citation, "SYNTHETIC-OPS-AOC")
	scope := ScopeRecommendation{
		Classification: "ROTATIONAL_SAMPLE",
		InputSignals: []string{
			"Legacy checklist candidate was reconciled against the current synthetic controlled source chain.",
			"The controlled source changed, so the sampled control cannot be automatically deferred.",
		},
		OperationalHistoryBasis: "SYNTHETIC_LEGACY_CANDIDATE_RECONCILED",
		Rationale:               "Use a rotational sample only with Department Manager technical approval; the current trace, not historical wording, is authoritative.",
		Guardrails: ScopeGuardrails{
			SourceChanged:              true,
			AutomaticDeferralPermitted: false,
		},
		ApprovalReviewState: "TECHNICAL_REVIEW_REQUIRED",
		AutomaticDeferral:   false,
	}
	comparison := &QuestionReconciliation{
		LegacyQuestionID:           "Q-SYNTHETIC-LEGACY-CANDIDATE-003",
		LegacyWording:              "Historical checklist candidate: is the cabin/ramp safety control recorded for later source mapping?",
		LegacyOperationalIntent:    "Preserve a cabin/ramp safety observation as a candidate input for a future controlled mapping.",
		LegacyResultHistory:        "Synthetic historical outcome record is unverified candidate input and does not establish a clean-history window.",
		LegacyExpectedEvidence:     []string{"Synthetic historical checklist wording", "Synthetic historical result record"},
		LegacyApplicability:        "UNKNOWN_CANDIDATE_INPUT",
		LegacyScopeClassification:  "UNKNOWN_CANDIDATE_INPUT",
		CurrentWording:             syntheticSupportedQuestion,
		CurrentExpectedEvidence:    append([]string(nil), syntheticSupportedExpectedEvidence...),
		CurrentApplicability:       "DIRECT",
		CurrentScopeClassification: "ROTATIONAL_SAMPLE",
		WordingChanged:             true, EvidenceChanged: true, ApplicabilityChanged: true, ScopeChanged: true,
	}
	bundle := CandidateBundle{
		SchemaVersion: "1.0.0", CandidateBundleID: "CAND-SYNTHETIC-HYBRID-RECONCILED-0004", GenerationRunID: "GENRUN-SYNTHETIC-HYBRID-RECONCILED-0004", Status: GeneratedDraft,
		GenerationRequest: request, InputDigest: request.CanonicalInputDigest,
		ComplianceMappings: []ComplianceMapping{{
			MappingID: "MAP-SYNTHETIC-HYBRID-RECONCILED-004", Requirement: syntheticSupportedRequirement,
			Relationship: "ADDRESSES", Applicability: "DIRECT", Citations: []Citation{citation}, Rationale: syntheticSupportedRationale,
		}},
		InspectionChecklist: InspectionChecklist{ChecklistID: "CHK-SYNTHETIC-HYBRID-RECONCILED-004", Questions: []ChecklistQuestion{{
			QuestionID: "Q-SYNTHETIC-HYBRID-RECONCILED-004", MappingIDs: []string{"MAP-SYNTHETIC-HYBRID-RECONCILED-004"},
			Prompt: syntheticSupportedQuestion, Citations: []Citation{citation}, VerificationMethod: syntheticSupportedVerificationMethod,
			ExpectedEvidence: syntheticSupportedExpectedEvidence, AllowedAnswers: allowedAnswers,
			Origin: HybridReconciledOrigin, ScopeRecommendation: scope, RegulatoryTrace: trace, Reconciliation: comparison,
		}}},
		SourceCurrentness: &SourceCurrentnessBinding{CurrentSourceSnapshotID: citation.SourceSnapshotID, CurrentSourceHash: citation.SourceHash, PreviousSourceSnapshotID: "SOURCE-SYNTHETIC-OPS-AOC", PreviousSourceHash: syntheticSourceHash},
	}
	bundle.OutputDigest, _ = candidateOutputDigest(bundle)
	return bundle
}

func syntheticQuestionGovernance(citation Citation, sourceIdentity string) (ScopeRecommendation, RegulatoryTrace) {
	title, version, page, section, nationalReference, procedure := "Synthetic test-profile source", "1", "1", "Synthetic OPS/AOC", "Synthetic NAMCAR OPS/AOC reference", "Synthetic CAA ramp-inspection procedure mapping"
	if citation.SourceSnapshotID == "SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2" {
		title, version, page, section = "Synthetic test-profile impact source", "2", "2", "Synthetic OPS/AOC impact"
		nationalReference = "Synthetic NAMCAR OPS/AOC impact reference"
		procedure = "Synthetic CAA ramp-inspection procedure impact mapping"
	}
	return ScopeRecommendation{
			Classification: "MANDATORY_CORE",
			InputSignals: []string{
				"Mandatory synthetic safety-control configuration.",
				"No validated operational history is available for automatic deferral.",
			},
			OperationalHistoryBasis: "SYNTHETIC_UNKNOWN_HISTORY",
			Rationale:               "The synthetic mandatory safety control remains included pending Department Manager review.",
			Guardrails: ScopeGuardrails{
				MandatoryControl: true, SafetyCritical: true, UnknownHistory: true,
				AutomaticDeferralPermitted: false,
			},
			ApprovalReviewState: "TECHNICAL_REVIEW_REQUIRED",
			AutomaticDeferral:   false,
		}, RegulatoryTrace{
			State: "RESOLVED", SourceIdentity: sourceIdentity,
			SourceTitle: title, ImmutableVersion: version,
			SHA256: citation.SourceHash, Locator: citation.Locator, Page: page,
			Section: section, Clause: citation.ClauseID,
			SourceType: "SYNTHETIC_INTERNAL_TEST_PROFILE", Applicability: "DIRECT",
			NationalReference:             nationalReference,
			ControlledCAAProcedureMapping: procedure,
			VerificationObjective:         "Verify the controlled synthetic ramp-safety requirement through observation and record reconciliation.",
			ExpectedEvidence:              append([]string(nil), syntheticSupportedExpectedEvidence...),
			CurrentnessState:              "CURRENT",
			TechnicalReviewState:          "TECHNICAL_REVIEW_REQUIRED",
		}
}
