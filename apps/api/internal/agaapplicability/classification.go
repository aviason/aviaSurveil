package agaapplicability

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

func NewPassProposalRecord(taxonomy Taxonomy, input PassProposalInput) (PassProposalRecord, error) {
	if err := input.Identity.Validate(); err != nil {
		return PassProposalRecord{}, err
	}
	if !classificationRunIDPattern.MatchString(input.ClassificationRunID) {
		return PassProposalRecord{}, fmt.Errorf("%w: missing run identity", ErrDigestMismatch)
	}
	if input.PassRole != PassCandidate && input.PassRole != PassChallenge {
		return PassProposalRecord{}, ErrUnknownCode
	}
	if err := validatePassRunID(input.PassRole, input.PassRunID); err != nil {
		return PassProposalRecord{}, err
	}
	if input.PromptDigest != FrozenPromptDigest {
		return PassProposalRecord{}, fmt.Errorf("%w: promptDigest", ErrDigestMismatch)
	}
	for name, value := range map[string]string{
		"modelDescriptorDigest": input.ModelDescriptorDigest,
		"inputDigest":           input.InputDigest,
	} {
		if !validDigest(value) {
			return PassProposalRecord{}, fmt.Errorf("%w: %s", ErrDigestMismatch, name)
		}
	}
	projection, err := normalizeProjection(taxonomy, input.Projection)
	if err != nil {
		return PassProposalRecord{}, err
	}
	rationales, err := normalizeStrings(input.RationaleCodes, taxonomy.RationaleCodes, "rationaleCodes", false)
	if err != nil {
		return PassProposalRecord{}, err
	}
	evidence, err := normalizeEvidence(taxonomy, input.ConfidenceEvidence)
	if err != nil {
		return PassProposalRecord{}, err
	}
	if err := validateEvidenceBindings(taxonomy, projection, evidence, false); err != nil {
		return PassProposalRecord{}, err
	}
	for index := range projection.ExternalInvolvements {
		edgeEvidence, err := normalizeEvidence(taxonomy, projection.ExternalInvolvements[index].ConfidenceEvidence)
		if err != nil {
			return PassProposalRecord{}, err
		}
		projection.ExternalInvolvements[index].ConfidenceEvidence = edgeEvidence
		if err := validateExternalEvidenceBinding(taxonomy, projection.ExternalInvolvements[index]); err != nil {
			return PassProposalRecord{}, err
		}
		binding := ExternalInvolvementBinding(taxonomy, projection.ExternalInvolvements[index])
		for _, tuple := range edgeEvidence {
			if err := validateSignalRuleBinding(taxonomy, binding, tuple); err != nil {
				return PassProposalRecord{}, err
			}
		}
	}
	sourceRefs, err := normalizeSourceRefs(taxonomy, input.SourceRefs)
	if err != nil {
		return PassProposalRecord{}, err
	}
	if len(rationales) > 16 || len(evidence) > 68 || len(sourceRefs) > 32 {
		return PassProposalRecord{}, fmt.Errorf("%w: pass proposal cardinality", ErrInvalidResolution)
	}
	record := PassProposalRecord{
		Identity:              input.Identity,
		ClassificationRunID:   input.ClassificationRunID,
		PassRole:              input.PassRole,
		PassRunID:             input.PassRunID,
		PromptDigest:          input.PromptDigest,
		ModelDescriptorDigest: input.ModelDescriptorDigest,
		InputDigest:           input.InputDigest,
		ProposalProjection:    projection,
		RationaleCodes:        nonNilSlice(rationales),
		ConfidenceEvidence:    nonNilSlice(evidence),
		SourceRefs:            nonNilSlice(sourceRefs),
	}
	record.PassResultDigest = ComputePassResultDigest(record)
	return record, nil
}

func validateEvidenceBindings(taxonomy Taxonomy, projection ProposalProjection, evidence []ConfidenceEvidence, external bool) error {
	bindings := make(map[string]struct{})
	for _, binding := range ProposalValueBindings(taxonomy, projection) {
		if external == (binding.ProposalField == "externalInvolvements") {
			bindings[binding.ProposalField+"\x00"+binding.ValueDigest] = struct{}{}
		}
	}
	for _, tuple := range evidence {
		bindingKey := tuple.ProposalField + "\x00" + tuple.ProposalValueDigest
		if _, ok := bindings[bindingKey]; !ok {
			return ErrEvidenceBinding
		}
		for _, binding := range ProposalValueBindings(taxonomy, projection) {
			if binding.ProposalField+"\x00"+binding.ValueDigest == bindingKey {
				if err := validateSignalRuleBinding(taxonomy, binding, tuple); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

func validateExternalEvidenceBinding(taxonomy Taxonomy, edge ExternalInvolvement) error {
	binding := ExternalInvolvementBinding(taxonomy, edge)
	for _, tuple := range edge.ConfidenceEvidence {
		if tuple.ProposalField != binding.ProposalField || tuple.ProposalValueDigest != binding.ValueDigest {
			return fmt.Errorf("%w: external involvement", ErrEvidenceBinding)
		}
	}
	return nil
}

func ComputePassResultDigest(record PassProposalRecord) string {
	return digestExcludingJSONFields("AGA-CLASSIFICATION-PASS-PROPOSAL-V1", record, "passResultDigest")
}

func ComputeItemSemanticDigest(item SealedClassificationItem) string {
	return digestExcludingJSONFields(
		"AGA-CLASSIFICATION-ITEM-V1",
		item,
		"itemSemanticDigest",
		"passOneResultDigest",
		"passTwoResultDigest",
		"classificationRunDigest",
		"aggregateDigest",
	)
}

func digestExcludingJSONFields(domain string, value any, fields ...string) string {
	digest, _ := DigestExcludingJSONFields(domain, value, fields...)
	return digest
}

func DigestExcludingJSONFields(domain string, value any, fields ...string) (string, error) {
	if err := validateUTF8Value(reflect.ValueOf(value), make(map[utf8Visit]struct{})); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	decoder := json.NewDecoder(strings.NewReader(string(encoded)))
	decoder.UseNumber()
	var object map[string]any
	if err := decoder.Decode(&object); err != nil {
		return "", err
	}
	for _, field := range fields {
		delete(object, field)
	}
	return DigestValue(domain, object)
}

func DeriveOutcome(
	taxonomy Taxonomy,
	candidate ProposalProjection,
	challenge ProposalProjection,
	candidateEvidence []ConfidenceEvidence,
	challengeEvidence []ConfidenceEvidence,
	sourceGap bool,
	externalUnresolved bool,
) ClassificationOutcome {
	candidateBindings := ProposalValueBindings(taxonomy, candidate)
	challengeBindings := ProposalValueBindings(taxonomy, challenge)
	candidateCovered := coveredEvidenceKeys(taxonomy, candidate, candidateEvidence)
	challengeCovered := coveredEvidenceKeys(taxonomy, challenge, challengeEvidence)

	coreAgree := bindingSetsEqual(candidateBindings, challengeBindings, true)
	coreEvidenced := bindingsCovered(candidateBindings, candidateCovered, true) && bindingsCovered(challengeBindings, challengeCovered, true)
	confidence := ConfidenceHigh
	if !coreAgree || !coreEvidenced {
		confidence = ConfidenceLow
	} else {
		allAgree := bindingSetsEqual(candidateBindings, challengeBindings, false)
		allEvidenced := bindingsCovered(candidateBindings, candidateCovered, false) && bindingsCovered(challengeBindings, challengeCovered, false)
		if !allAgree || !allEvidenced {
			confidence = ConfidenceMedium
		}
	}
	recommendation := RecommendationManagerReview
	if sourceGap {
		recommendation = RecommendationBlockedSourceGap
	} else if confidence == ConfidenceHigh && !externalUnresolved {
		recommendation = RecommendationAutoProposed
	}
	return ClassificationOutcome{AgreementConfidence: confidence, RecommendationState: recommendation}
}

func coveredEvidenceKeys(taxonomy Taxonomy, projection ProposalProjection, top []ConfidenceEvidence) map[string]bool {
	covered := make(map[string]bool)
	bindings := make(map[string]ProposalValueBinding)
	for _, binding := range ProposalValueBindings(taxonomy, projection) {
		bindings[binding.ProposalField+"\x00"+binding.ValueDigest] = binding
	}
	markCovered := func(tuple ConfidenceEvidence) {
		key := tuple.ProposalField + "\x00" + tuple.ProposalValueDigest
		binding, exists := bindings[key]
		if exists && evidenceTupleCountsForCoverage(taxonomy, binding, tuple) {
			covered[key] = true
		}
	}
	for _, tuple := range top {
		markCovered(tuple)
	}
	for _, edge := range projection.ExternalInvolvements {
		for _, tuple := range edge.ConfidenceEvidence {
			markCovered(tuple)
		}
	}
	return covered
}

func evidenceTupleCountsForCoverage(taxonomy Taxonomy, binding ProposalValueBinding, tuple ConfidenceEvidence) bool {
	if tuple.SignalRuleID == "" {
		return true
	}
	for _, rule := range taxonomy.SignalRuleFieldRules[tuple.SignalRuleID] {
		valueAllowed := contains(rule.AllowedValues, binding.SemanticValue) || (binding.ValueShape == "EXTERNAL_EDGE_TUPLE" && contains(rule.AllowedValues, "ANY_TAXONOMY_VALID_EXTERNAL_EDGE"))
		if rule.ProposalField == binding.ProposalField && rule.ValueShape == binding.ValueShape && valueAllowed && contains(rule.AllowedRationaleCodes, tuple.RationaleCode) {
			return rule.SignalAloneSatisfiesEvidence
		}
	}
	return false
}

func bindingSetsEqual(left, right []ProposalValueBinding, coreOnly bool) bool {
	leftSet := make(map[string]struct{})
	rightSet := make(map[string]struct{})
	for _, binding := range left {
		if !coreOnly || binding.Core {
			leftSet[binding.ProposalField+"\x00"+binding.ValueDigest] = struct{}{}
		}
	}
	for _, binding := range right {
		if !coreOnly || binding.Core {
			rightSet[binding.ProposalField+"\x00"+binding.ValueDigest] = struct{}{}
		}
	}
	return reflect.DeepEqual(leftSet, rightSet)
}

func bindingsCovered(bindings []ProposalValueBinding, covered map[string]bool, coreOnly bool) bool {
	for _, binding := range bindings {
		if coreOnly && !binding.Core {
			continue
		}
		if !covered[binding.ProposalField+"\x00"+binding.ValueDigest] {
			return false
		}
	}
	return true
}

func validatePassRecord(taxonomy Taxonomy, record PassProposalRecord, expectedRole PassRole, expectedInputDigest string, input ClassificationInput, facts EvidenceFacts) error {
	if err := record.Identity.Validate(); err != nil {
		return err
	}
	if record.PassRole != expectedRole {
		return ErrPassBijection
	}
	if record.ClassificationRunID != input.ClassificationRunID || record.PromptDigest != input.PromptDigest {
		return fmt.Errorf("%w: run/prompt pin", ErrPassBijection)
	}
	if err := validatePassRunID(record.PassRole, record.PassRunID); err != nil {
		return err
	}
	if !validDigest(record.ModelDescriptorDigest) || record.InputDigest != expectedInputDigest || !validDigest(record.InputDigest) {
		return fmt.Errorf("%w: pass pins", ErrDigestMismatch)
	}
	normalizedProjection, err := normalizeProjection(taxonomy, record.ProposalProjection)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(normalizedProjection, record.ProposalProjection) {
		return fmt.Errorf("%w: pass proposal is not normalized", ErrDigestMismatch)
	}
	if err := ValidateConfidenceEvidence(taxonomy, record.ProposalProjection, record.ConfidenceEvidence, facts); err != nil {
		return err
	}
	for _, edge := range record.ProposalProjection.ExternalInvolvements {
		if err := validateExternalEvidenceBinding(taxonomy, edge); err != nil {
			return err
		}
		binding := ExternalInvolvementBinding(taxonomy, edge)
		for _, tuple := range edge.ConfidenceEvidence {
			if err := validateSignalRuleBinding(taxonomy, binding, tuple); err != nil {
				return err
			}
			if !trustedEvidenceFact(facts, tuple) {
				return fmt.Errorf("%w: external involvement", ErrEvidenceFactMismatch)
			}
		}
	}
	normalizedRationales, err := normalizeStrings(record.RationaleCodes, taxonomy.RationaleCodes, "rationaleCodes", false)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(normalizedRationales, record.RationaleCodes) {
		return fmt.Errorf("%w: rationales not normalized", ErrDigestMismatch)
	}
	normalizedSources, err := normalizeSourceRefs(taxonomy, record.SourceRefs)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(normalizedSources, record.SourceRefs) {
		return fmt.Errorf("%w: source refs not normalized", ErrDigestMismatch)
	}
	normalizedEvidence, err := normalizeEvidence(taxonomy, record.ConfidenceEvidence)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(normalizedEvidence, record.ConfidenceEvidence) {
		return fmt.Errorf("%w: evidence not normalized", ErrDigestMismatch)
	}
	if !validDigest(record.PassResultDigest) || record.PassResultDigest != ComputePassResultDigest(record) {
		return fmt.Errorf("%w: passResultDigest", ErrDigestMismatch)
	}
	return nil
}

func validatePassRunID(role PassRole, passRunID string) error {
	if !passRunIDPattern.MatchString(passRunID) {
		return fmt.Errorf("%w: passRunId", ErrDigestMismatch)
	}
	expectedPrefix := "aga-classification-pass-candidate-"
	if role == PassChallenge {
		expectedPrefix = "aga-classification-pass-challenge-"
	}
	if !strings.HasPrefix(passRunID, expectedPrefix) {
		return fmt.Errorf("%w: pass role/run identity", ErrPassBijection)
	}
	return nil
}

func validateGovernance(taxonomy Taxonomy, governance GovernanceState) error {
	if governance.SourceMappingState != SourceMappingRequired || governance.SourceAuthorityState != SourceAuthorityNotAttested || governance.RiskClassificationState != RiskExpertReviewRequired || governance.DecisionState != DecisionNotSupplied || !contains([]string{ExtractionCandidate, ExtractionExactSourceBacked}, governance.ExtractionState) {
		return fmt.Errorf("%w: incomplete governance state", ErrUnknownCode)
	}
	normalized, err := normalizeStrings(governance.BlockerCodes, taxonomy.BlockerCodes, "blockerCodes", false)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(normalized, governance.BlockerCodes) {
		return fmt.Errorf("%w: blockers not normalized", ErrDigestMismatch)
	}
	if governance.QuestionSourceProposalGap && !contains(governance.BlockerCodes, SourceMappingRequired) {
		return fmt.Errorf("%w: source-gap blocker missing", ErrEvidenceBinding)
	}
	if governance.ExternalApplicabilityUnresolved && !contains(governance.BlockerCodes, "PROVIDER_APPLICABILITY_UNRESOLVED") {
		return fmt.Errorf("%w: provider blocker missing", ErrEvidenceBinding)
	}
	return nil
}

func FrozenFixedInputDigests() FixedInputDigests {
	return FixedInputDigests{
		PackageJSONSHA256:                  FrozenPackageJSONSHA256,
		SealedOverlayLoaderZIPSHA256:       "sha256:30700a88aeb5b26514bf7eb76bef050deb08b96294db94117d185de5c9f163b2",
		ProviderCatalogSHA256:              "sha256:42079b4046542e392c393fe6de1052d84f96938ea163cf63deed5ae9c4b6a789",
		ResearchZIPSHA256:                  "sha256:137592c739bc22f6be026f5bad94c5b200bb983132017d026b7e39634ab392c7",
		ResearchQuestionCSVSHA256:          "sha256:e39685d467c9c66220b20e998deab366a138148f4d532db7fac07e58e64e7a7c",
		ProviderClassificationCSVSHA256:    "sha256:d52a98739db61828c16aa734154be18b11e6ebb358eeeb7f84c3d92a4a5430de",
		AmbiguityCSVSHA256:                 "sha256:6e97a193f5e12dbe81f87d44d4b22c36ce446a40be7ef0f9fc939e8fbf1e654d",
		WorkbookSHA256:                     "sha256:e4d054f741b11ca9d848842a891d6f811f2e644aba29a7ffda970bfe6abb931e",
		AuditChecklistWorkflowSHA256:       "sha256:7dee737c7c5e47e996857e956514a8d46d1a4444234b021cac77cd6cff6b30a2",
		FindingCAPEvidenceWorkflowSHA256:   "sha256:896f9fa7d498fdc20c582134a15ed6acdc11b78926e655854c43e49fbb24815c",
		ProductionContractVocabularySHA256: "sha256:3ef3349d738feb9789aaab6e92246f55948053604a8304706fc1bbd0cd786769",
	}
}

func ComputeModelDescriptorDigest(descriptor ModelDescriptor) string {
	descriptor.UnavailableFields = uniqueSorted(nonNilSlice(descriptor.UnavailableFields))
	return digestValue("AGA-MODEL-DESCRIPTOR-V1", descriptor)
}

func validateModelDescriptor(descriptor ModelDescriptor) error {
	for _, value := range []string{descriptor.ModelID, descriptor.ModelIDSource, descriptor.RuntimeReportedFamily, descriptor.Service, descriptor.Interface, descriptor.RequestedReasoningEffort, descriptor.ForkTurns} {
		if !utf8.ValidString(value) || len(value) == 0 || len(value) > 128 {
			return ErrInputBounds
		}
	}
	if descriptor.SnapshotBuildLabel != nil && (!utf8.ValidString(*descriptor.SnapshotBuildLabel) || len(*descriptor.SnapshotBuildLabel) > 128) {
		return ErrInputBounds
	}
	if descriptor.UnavailableFields == nil {
		return ErrInputBounds
	}
	if descriptor.ModelID == "" || descriptor.ModelIDSource != "accepted-collaboration-spawn-agent-model-override" || descriptor.RuntimeReportedFamily == "" || descriptor.Service != "Codex" || descriptor.Interface != "API" || descriptor.RequestedReasoningEffort != "xhigh" || descriptor.ForkTurns != "none" {
		return fmt.Errorf("%w: model descriptor", ErrDigestMismatch)
	}
	allowedUnavailable := []string{"exactModelVersion", "serviceTier", "snapshotBuildLabel"}
	normalized, err := normalizeStrings(descriptor.UnavailableFields, allowedUnavailable, "unavailableFields", false)
	if err != nil || !reflect.DeepEqual(normalized, descriptor.UnavailableFields) {
		return fmt.Errorf("%w: model availability", ErrDigestMismatch)
	}
	hasSnapshotUnavailable := contains(descriptor.UnavailableFields, "snapshotBuildLabel")
	if (descriptor.SnapshotBuildLabel == nil) != hasSnapshotUnavailable {
		return fmt.Errorf("%w: snapshot availability", ErrDigestMismatch)
	}
	return nil
}

func ComputeRunInputDigest(fixed FixedInputDigests) string {
	taxonomy := FrozenTaxonomy()
	return digestValue("AGA-CLASSIFICATION-RUN-INPUT-V1", map[string]any{
		"taxonomyVersion":     taxonomy.Version,
		"taxonomyDigest":      taxonomy.Digest,
		"fixedInputDigests":   fixed,
		"promptDigest":        FrozenPromptDigest,
		"batchManifestDigest": FrozenBatchManifestDigest,
	})
}

var frozenBatchItemCounts = []int{50, 49, 55, 54, 54, 53, 52, 54, 53, 52, 52, 57, 58, 58, 58, 61, 59, 53, 51, 52, 56, 52, 56, 57, 4}

func batchOrdinalForIndex(index int) int {
	offset := 0
	for ordinal, count := range frozenBatchItemCounts {
		if index < offset+count {
			return ordinal
		}
		offset += count
	}
	return -1
}

func passInputItemForIndex(inputs []ClassificationPassInput, index int) (ClassificationPassInputItem, bool) {
	offset := 0
	for _, input := range inputs {
		if index < offset+len(input.Items) {
			return input.Items[index-offset], true
		}
		offset += len(input.Items)
	}
	return ClassificationPassInputItem{}, false
}

func ComputeClassificationPassInputDigest(input ClassificationPassInput) string {
	return digestValue("AGA-CLASSIFICATION-PASS-INPUT-V1", input)
}

func ComputeRoleNeutralPassInputDigest(input ClassificationPassInput) string {
	return digestValue("AGA-CLASSIFICATION-ROLE-NEUTRAL-PASS-INPUT-V1", map[string]any{
		"schemaVersion": input.SchemaVersion, "purpose": input.Purpose,
		"classificationRunId": input.ClassificationRunID, "batchOrdinal": input.BatchOrdinal,
		"taxonomyVersion": input.TaxonomyVersion, "taxonomyDigest": input.TaxonomyDigest,
		"promptDigest": input.PromptDigest, "batchManifestDigest": input.BatchManifestDigest,
		"fixedInputDigests": input.FixedInputDigests, "items": input.Items,
	})
}

func ComputePassSeal(receipt PassSealReceipt) string {
	return digestExcludingJSONFields("AGA-CLASSIFICATION-PASS-SEAL-V1", receipt, "passSealDigest")
}

func computePassInputSetDigest(receipt PassSealReceipt) string {
	return digestValue("AGA-CLASSIFICATION-PASS-INPUT-SET-V1", map[string]any{"classificationRunId": receipt.ClassificationRunID, "passRole": receipt.PassRole, "passRunId": receipt.PassRunID, "batchManifestDigest": receipt.BatchManifestDigest, "orderedInputDigests": receipt.OrderedInputDigests})
}

func ComputePassBatchOutputDigest(output PassBatchOutput) string {
	return digestExcludingJSONFields("AGA-CLASSIFICATION-PASS-BATCH-V1", output, "batchOutputDigest")
}

func validatePassInput(input ClassificationPassInput, role PassRole, expectedRunID, expectedPassRunID, expectedModelDigest string, expectedOrdinal, expectedCount int, expectedIdentities []BaseIdentity) error {
	if input.Items == nil || expectedCount < 1 || expectedCount > 64 {
		return ErrInputBounds
	}
	encoded, err := canonicalJSON(input)
	if err != nil || len(encoded) > 98304 {
		return ErrInputBounds
	}
	if input.SchemaVersion != "aga-hybrid-classification-pass-input/v1" || input.Purpose != "ROW_CLASSIFICATION_PRIVATE_INPUT" || input.ClassificationRunID != expectedRunID || input.PassRole != role || input.PassRunID != expectedPassRunID || input.BatchOrdinal != expectedOrdinal || input.TaxonomyVersion != FrozenTaxonomy().Version || input.TaxonomyDigest != FrozenTaxonomy().Digest || input.PromptDigest != FrozenPromptDigest || input.ModelDescriptorDigest != expectedModelDigest || input.BatchManifestDigest != FrozenBatchManifestDigest || !reflect.DeepEqual(input.FixedInputDigests, FrozenFixedInputDigests()) || len(input.Items) != expectedCount {
		return ErrDigestMismatch
	}
	if err := validatePassRunID(role, input.PassRunID); err != nil {
		return err
	}
	for index, item := range input.Items {
		if !reflect.DeepEqual(item.Identity, expectedIdentities[index]) || !utf8.ValidString(item.QuestionBody) || len(item.QuestionBody) == 0 || len(item.QuestionBody) > 2048 || item.Identity.TextDigest != rawTextDigest(item.QuestionBody) || !validPackageFacts(item.PackageFacts) || !validResearchFacts(item.ResearchCandidateFacts, item.Identity) {
			return ErrDigestMismatch
		}
	}
	return nil
}

func rawTextDigest(body string) string {
	sum := sha256.Sum256([]byte(body))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func deriveEvidenceFacts(item ClassificationPassInputItem) EvidenceFacts {
	return EvidenceFacts{
		"QUESTION_BODY_DIGEST":    {{Digest: rawTextDigest(item.QuestionBody)}},
		"FORM_METADATA_DIGEST":    {{Digest: digestValue("AGA-FORM-METADATA-FACT-V1", map[string]any{"formCode": item.Identity.FormCode, "formKind": item.PackageFacts.FormKind, "formRiskBand": item.PackageFacts.FormRiskBand})}},
		"RESEARCH_ROW_DIGEST":     {{Digest: digestValue("AGA-RESEARCH-ROW-V1", item.ResearchCandidateFacts)}},
		"SOURCE_PROPOSAL_DIGEST":  evidenceFactsFromDigests(item.PackageFacts.SourceProposalDigests),
		"SOURCE_REFERENCE_DIGEST": evidenceFactsFromDigests(item.PackageFacts.SourceReferenceDigests),
	}
}

func evidenceFactsFromDigests(values []string) []EvidenceFact {
	result := make([]EvidenceFact, len(values))
	for index, value := range values {
		result[index] = EvidenceFact{Digest: value}
	}
	return result
}

func validPackageFacts(facts ClassificationPackageFacts) bool {
	return contains([]string{"ADMINISTRATIVE_FORM", "CHECKLIST", "FORM"}, facts.FormKind) && contains([]string{"PROPOSED_CONTROL_ASSURANCE", "PROPOSED_HIGH_OPERATIONAL", "PROPOSED_REVIEW_REQUIRED", "PROPOSED_SAFETY_CRITICAL"}, facts.FormRiskBand) && contains([]string{"PROPOSED_CONTROL_ASSURANCE", "PROPOSED_HIGH_OPERATIONAL", "PROPOSED_REVIEW_REQUIRED", "PROPOSED_SAFETY_CRITICAL"}, facts.QuestionRiskBand) && contains([]string{"AERODROME_OPERATIONAL_SAFETY", "GOVERNANCE_AND_RECORDS", "OPERATIONAL_ASSURANCE", "UNCLASSIFIED"}, facts.QuestionRiskDomain) && facts.SourceMappingState == SourceMappingRequired && facts.SourceAuthorityState == SourceAuthorityNotAttested && contains([]string{ExtractionCandidate, ExtractionExactSourceBacked}, facts.ExtractionState) && facts.RiskClassificationState == RiskExpertReviewRequired && facts.DecisionState == DecisionNotSupplied && validDigestSet(facts.SourceProposalDigests, 8) && validDigestSet(facts.SourceReferenceDigests, 8)
}

func validResearchFacts(facts ClassificationResearchCandidateFacts, identity BaseIdentity) bool {
	return facts.FormCode == identity.FormCode && facts.ProposalID == identity.ProposalID && facts.Ordinal == strconv.Itoa(identity.Ordinal) && facts.TextDigest == identity.TextDigest && contains([]string{"AERODROME_AND_SURROUNDING_AIRSPACE", "AERODROME_APRON_FUNCTION", "AERODROME_DATA", "AERODROME_EMERGENCY_PLAN", "AERODROME_OR_HELIPORT", "AERODROME_RFFS_FUNCTION", "AERODROME_RUNWAY_SAFETY_PROGRAMME", "AERODROME_SMS", "AERODROME_SYSTEM_OR_ASSET", "AERODROME_WILDLIFE_PROGRAMME", "MILITARY_AERODROME_CIVIL_USE"}, facts.TargetKind) && contains([]string{"AERODROME_AND_SURROUNDING_AIRSPACE", "AERODROME_APRON_FUNCTION", "AERODROME_DATA", "AERODROME_EMERGENCY_PLAN", "AERODROME_OR_HELIPORT", "AERODROME_RFFS_FUNCTION", "AERODROME_RUNWAY_SAFETY_PROGRAMME", "AERODROME_SMS", "AERODROME_SYSTEM_OR_ASSET", "AERODROME_WILDLIFE_PROGRAMME", "MILITARY_CIVIL_USE_APPLICATION"}, facts.OperationActivityQualifier) && facts.PrimarySubjectProposal == "AERODROME_OPERATOR" && contains([]string{"", "AIR_OPERATOR", "AIS_AIM_PROVIDER", "ANSP", "CNS_PROVIDER"}, facts.OperationalInterfaceCandidates) && contains([]string{"", "AIR_OPERATOR", "AIS_AIM_PROVIDER", "ANSP", "CNS_PROVIDER"}, facts.EvidenceContributorCandidates) && contains([]string{"false", "true"}, facts.ProviderApplicabilityUnresolved) && contains([]string{"", "missing_source_reference", "possible_external_actor_as_grammatical_subject"}, facts.UnresolvedReasons) && utf8.ValidString(facts.SourceRefs) && len(facts.SourceRefs) <= 256
}

func validDigestSet(values []string, maximum int) bool {
	return len(values) <= maximum && len(uniqueSorted(values)) == len(values) && func() bool {
		for _, value := range values {
			if !validDigest(value) {
				return false
			}
		}
		return true
	}()
}

func validatePassSeal(receipt PassSealReceipt, role PassRole, records []PassProposalRecord, batchInputs []ClassificationPassInput, modelDigest string, orderedIdentities []BaseIdentity, verifyOutputs bool) error {
	if receipt.ClassificationRunID == "" || receipt.PassRole != role || !validDigest(receipt.ModelDescriptorDigest) || receipt.ModelDescriptorDigest != modelDigest || receipt.PromptDigest != FrozenPromptDigest || receipt.BatchManifestDigest != FrozenBatchManifestDigest || receipt.BatchCount != len(frozenBatchItemCounts) || receipt.ItemCount != FrozenBaseQuestionCount || len(batchInputs) != len(frozenBatchItemCounts) || len(receipt.OrderedInputDigests) != len(frozenBatchItemCounts) || len(receipt.OrderedBatchOutputDigests) != len(frozenBatchItemCounts) || len(receipt.OrderedPassResultDigests) != FrozenBaseQuestionCount || !validDigest(receipt.PassSealDigest) || len(records) != FrozenBaseQuestionCount || len(orderedIdentities) != FrozenBaseQuestionCount {
		return ErrDigestMismatch
	}
	if err := validatePassRunID(role, receipt.PassRunID); err != nil {
		return err
	}
	inputDigests := make([]string, 0, len(frozenBatchItemCounts))
	batchOutputs := make([]string, 0, len(frozenBatchItemCounts))
	results := make([]string, 0, len(records))
	offset := 0
	for batchIndex, count := range frozenBatchItemCounts {
		identities := orderedIdentities[offset : offset+count]
		if err := validatePassInput(batchInputs[batchIndex], role, receipt.ClassificationRunID, receipt.PassRunID, modelDigest, batchIndex+1, count, identities); err != nil {
			return err
		}
		expectedInput := ComputeClassificationPassInputDigest(batchInputs[batchIndex])
		inputDigests = append(inputDigests, expectedInput)
		batchRecords := records[offset : offset+count]
		for _, record := range batchRecords {
			if verifyOutputs && (record.InputDigest != expectedInput || record.PassRunID != receipt.PassRunID || record.ModelDescriptorDigest != modelDigest || record.PassRole != role || record.ClassificationRunID != receipt.ClassificationRunID) {
				return ErrDigestMismatch
			}
			results = append(results, record.PassResultDigest)
		}
		output := PassBatchOutput{SchemaVersion: "aga-hybrid-classification-pass-batch/v1", ClassificationRunID: receipt.ClassificationRunID, PassRole: role, PassRunID: receipt.PassRunID, BatchOrdinal: batchIndex + 1, PromptDigest: FrozenPromptDigest, ModelDescriptorDigest: modelDigest, InputDigest: expectedInput, Records: batchRecords}
		output.BatchOutputDigest = ComputePassBatchOutputDigest(output)
		batchOutputs = append(batchOutputs, output.BatchOutputDigest)
		offset += count
	}
	if !reflect.DeepEqual(receipt.OrderedInputDigests, inputDigests) || receipt.PassInputSetDigest != computePassInputSetDigest(receipt) || receipt.PassSealDigest != ComputePassSeal(receipt) || (verifyOutputs && (!reflect.DeepEqual(receipt.OrderedBatchOutputDigests, batchOutputs) || !reflect.DeepEqual(receipt.OrderedPassResultDigests, results))) {
		return ErrDigestMismatch
	}
	return nil
}

func ReconcileClassification(input ClassificationInput) (ClassificationResult, error) {
	taxonomy := FrozenTaxonomy()
	if !classificationRunIDPattern.MatchString(input.ClassificationRunID) || input.PromptDigest != FrozenPromptDigest || input.TaxonomyDigest != taxonomy.Digest || input.BatchManifestDigest != FrozenBatchManifestDigest || !reflect.DeepEqual(input.FixedInputDigests, FrozenFixedInputDigests()) || input.RunInputDigest != ComputeRunInputDigest(input.FixedInputDigests) {
		return ClassificationResult{}, fmt.Errorf("%w: classification pins", ErrDigestMismatch)
	}
	modelDigests := make([]string, 0, len(input.ModelDescriptors))
	modelDigestSet := make(map[string]struct{}, len(input.ModelDescriptors))
	for _, descriptor := range input.ModelDescriptors {
		if err := validateModelDescriptor(descriptor); err != nil {
			return ClassificationResult{}, err
		}
		digest := ComputeModelDescriptorDigest(descriptor)
		if _, duplicate := modelDigestSet[digest]; duplicate {
			return ClassificationResult{}, fmt.Errorf("%w: model descriptor", ErrDuplicateProposalValue)
		}
		modelDigestSet[digest] = struct{}{}
		modelDigests = append(modelDigests, digest)
	}
	if len(modelDigests) < 1 || len(modelDigests) > 2 {
		return ClassificationResult{}, fmt.Errorf("%w: model descriptor count", ErrDigestMismatch)
	}
	sort.Strings(modelDigests)
	if len(input.OrderedBaseIdentities) != FrozenBaseQuestionCount || len(input.PassInputsByRole) != 2 || len(input.CandidateRecords) != FrozenBaseQuestionCount || len(input.ChallengeRecords) != FrozenBaseQuestionCount || len(input.GovernanceByIdentity) != FrozenBaseQuestionCount || len(input.EvidenceFactsByIdentity) != FrozenBaseQuestionCount {
		return ClassificationResult{}, ErrPassBijection
	}
	if input.PassInputsByRole == nil || input.PassInputsByRole[PassCandidate] == nil || input.PassInputsByRole[PassChallenge] == nil {
		return ClassificationResult{}, ErrInputBounds
	}
	for index := range input.PassInputsByRole[PassCandidate] {
		if index >= len(input.PassInputsByRole[PassChallenge]) || ComputeRoleNeutralPassInputDigest(input.PassInputsByRole[PassCandidate][index]) != ComputeRoleNeutralPassInputDigest(input.PassInputsByRole[PassChallenge][index]) {
			return ClassificationResult{}, errors.Join(ErrPrivateInputMismatch, ErrDigestMismatch)
		}
	}
	if digestValue("AGA-HYBRID-ORDERED-IDENTITIES-V1", input.OrderedBaseIdentities) != FrozenOrderedIdentityDigest {
		return ClassificationResult{}, ErrPassBijection
	}
	candidatePrecheck := make(map[string]struct{}, FrozenBaseQuestionCount)
	challengePrecheck := make(map[string]struct{}, FrozenBaseQuestionCount)
	for index := range input.OrderedBaseIdentities {
		candidateKey := input.CandidateRecords[index].Identity.Key()
		challengeKey := input.ChallengeRecords[index].Identity.Key()
		if _, exists := candidatePrecheck[candidateKey]; exists {
			return ClassificationResult{}, ErrDuplicateIdentity
		}
		if _, exists := challengePrecheck[challengeKey]; exists {
			return ClassificationResult{}, ErrDuplicateIdentity
		}
		candidatePrecheck[candidateKey] = struct{}{}
		challengePrecheck[challengeKey] = struct{}{}
	}
	for index, identity := range input.OrderedBaseIdentities {
		if !reflect.DeepEqual(input.CandidateRecords[index].Identity, identity) || !reflect.DeepEqual(input.ChallengeRecords[index].Identity, identity) {
			return ClassificationResult{}, ErrPassBijection
		}
	}
	if input.PassOneSealDigest != input.PassOneSealReceipt.PassSealDigest || input.PassTwoSealDigest != input.PassTwoSealReceipt.PassSealDigest || input.PassOneSealReceipt.ClassificationRunID != input.ClassificationRunID || input.PassTwoSealReceipt.ClassificationRunID != input.ClassificationRunID {
		return ClassificationResult{}, ErrDigestMismatch
	}
	if err := validatePassSeal(input.PassOneSealReceipt, PassCandidate, input.CandidateRecords, input.PassInputsByRole[PassCandidate], input.PassOneSealReceipt.ModelDescriptorDigest, input.OrderedBaseIdentities, false); err != nil {
		return ClassificationResult{}, err
	}
	if err := validatePassSeal(input.PassTwoSealReceipt, PassChallenge, input.ChallengeRecords, input.PassInputsByRole[PassChallenge], input.PassTwoSealReceipt.ModelDescriptorDigest, input.OrderedBaseIdentities, false); err != nil {
		return ClassificationResult{}, err
	}
	candidateIdentitySet := make(map[string]struct{}, FrozenBaseQuestionCount)
	challengeIdentitySet := make(map[string]struct{}, FrozenBaseQuestionCount)
	usedModelDigests := make(map[string]struct{}, len(modelDigestSet))
	for index := 0; index < FrozenBaseQuestionCount; index++ {
		candidateKey := input.CandidateRecords[index].Identity.Key()
		challengeKey := input.ChallengeRecords[index].Identity.Key()
		if _, duplicate := candidateIdentitySet[candidateKey]; duplicate {
			return ClassificationResult{}, ErrDuplicateIdentity
		}
		if _, duplicate := challengeIdentitySet[challengeKey]; duplicate {
			return ClassificationResult{}, ErrDuplicateIdentity
		}
		candidateIdentitySet[candidateKey] = struct{}{}
		challengeIdentitySet[challengeKey] = struct{}{}
		usedModelDigests[input.CandidateRecords[index].ModelDescriptorDigest] = struct{}{}
		usedModelDigests[input.ChallengeRecords[index].ModelDescriptorDigest] = struct{}{}
	}
	if !reflect.DeepEqual(usedModelDigests, modelDigestSet) {
		return ClassificationResult{}, ErrDigestMismatch
	}

	result := ClassificationResult{
		ClassificationRunID: input.ClassificationRunID,
		State:               ClassificationRunSealed,
		TaxonomyVersion:     taxonomy.Version,
		TaxonomyDigest:      taxonomy.Digest,
		InputDigest:         input.RunInputDigest,
		CandidateRecords:    append([]PassProposalRecord(nil), input.CandidateRecords...),
		ChallengeRecords:    append([]PassProposalRecord(nil), input.ChallengeRecords...),
		Items:               make([]SealedClassificationItem, 0, FrozenBaseQuestionCount),
	}
	seenIdentities := make(map[string]struct{}, FrozenBaseQuestionCount)
	for index, expectedIdentity := range input.OrderedBaseIdentities {
		if err := expectedIdentity.Validate(); err != nil {
			return ClassificationResult{}, err
		}
		key := expectedIdentity.Key()
		if _, duplicate := seenIdentities[key]; duplicate {
			return ClassificationResult{}, ErrDuplicateIdentity
		}
		seenIdentities[key] = struct{}{}
		expectedInputDigest := input.PassOneSealReceipt.OrderedInputDigests[batchOrdinalForIndex(index)]
		if !validDigest(expectedInputDigest) {
			return ClassificationResult{}, ErrPassBijection
		}
		candidate := input.CandidateRecords[index]
		challenge := input.ChallengeRecords[index]
		if !reflect.DeepEqual(candidate.Identity, expectedIdentity) || !reflect.DeepEqual(challenge.Identity, expectedIdentity) {
			return ClassificationResult{}, ErrPassBijection
		}
		_, factsExist := input.EvidenceFactsByIdentity[key]
		governance, governanceExists := input.GovernanceByIdentity[key]
		if !factsExist || !governanceExists {
			return ClassificationResult{}, ErrPassBijection
		}
		passItem, passItemExists := passInputItemForIndex(input.PassInputsByRole[PassCandidate], index)
		challengePassItem, challengePassItemExists := passInputItemForIndex(input.PassInputsByRole[PassChallenge], index)
		facts := deriveEvidenceFacts(passItem)
		if err := validateGovernance(taxonomy, governance); err != nil {
			return ClassificationResult{}, err
		}
		if !passItemExists || !challengePassItemExists || !reflect.DeepEqual(passItem, challengePassItem) || !reflect.DeepEqual(facts, deriveEvidenceFacts(challengePassItem)) {
			return ClassificationResult{}, ErrPassBijection
		}
		if err := validatePassRecord(taxonomy, candidate, PassCandidate, expectedInputDigest, input, facts); err != nil {
			return ClassificationResult{}, err
		}
		expectedChallengeInput := input.PassTwoSealReceipt.OrderedInputDigests[batchOrdinalForIndex(index)]
		if !validDigest(expectedChallengeInput) || expectedChallengeInput == expectedInputDigest {
			return ClassificationResult{}, ErrDigestMismatch
		}
		if err := validatePassRecord(taxonomy, challenge, PassChallenge, expectedChallengeInput, input, facts); err != nil {
			return ClassificationResult{}, err
		}
		if _, trusted := modelDigestSet[candidate.ModelDescriptorDigest]; !trusted {
			return ClassificationResult{}, fmt.Errorf("%w: candidate model descriptor", ErrDigestMismatch)
		}
		if _, trusted := modelDigestSet[challenge.ModelDescriptorDigest]; !trusted {
			return ClassificationResult{}, fmt.Errorf("%w: challenge model descriptor", ErrDigestMismatch)
		}
		if err := validateGovernance(taxonomy, governance); err != nil {
			return ClassificationResult{}, err
		}
		outcome := DeriveOutcome(
			taxonomy,
			candidate.ProposalProjection,
			challenge.ProposalProjection,
			candidate.ConfidenceEvidence,
			challenge.ConfidenceEvidence,
			governance.QuestionSourceProposalGap,
			governance.ExternalApplicabilityUnresolved,
		)
		item := SealedClassificationItem{
			Identity:               candidate.Identity,
			Projection:             cloneJSON(candidate.ProposalProjection),
			AgreementConfidence:    outcome.AgreementConfidence,
			RecommendationState:    outcome.RecommendationState,
			RationaleCodes:         append([]string{}, candidate.RationaleCodes...),
			ConfidenceEvidence:     append([]ConfidenceEvidence{}, candidate.ConfidenceEvidence...),
			SourceRefs:             append([]SourceReference{}, candidate.SourceRefs...),
			GovernanceState:        cloneJSON(governance),
			PassDisagreementCodes:  disagreementCodes(taxonomy, candidate.ProposalProjection, challenge.ProposalProjection),
			PassOneResultDigest:    candidate.PassResultDigest,
			PassTwoResultDigest:    challenge.PassResultDigest,
			PassOneRunID:           candidate.PassRunID,
			PassTwoRunID:           challenge.PassRunID,
			PromptDigest:           input.PromptDigest,
			ModelDescriptorDigests: uniqueSorted([]string{candidate.ModelDescriptorDigest, challenge.ModelDescriptorDigest}),
			TaxonomyDigest:         taxonomy.Digest,
			InputDigest:            input.RunInputDigest,
		}
		item.ItemSemanticDigest = ComputeItemSemanticDigest(item)
		result.Items = append(result.Items, item)
	}
	if err := validatePassSeal(input.PassOneSealReceipt, PassCandidate, input.CandidateRecords, input.PassInputsByRole[PassCandidate], input.PassOneSealReceipt.ModelDescriptorDigest, input.OrderedBaseIdentities, true); err != nil {
		return ClassificationResult{}, err
	}
	if err := validatePassSeal(input.PassTwoSealReceipt, PassChallenge, input.ChallengeRecords, input.PassInputsByRole[PassChallenge], input.PassTwoSealReceipt.ModelDescriptorDigest, input.OrderedBaseIdentities, true); err != nil {
		return ClassificationResult{}, err
	}
	result.Aggregate = buildClassificationAggregate(taxonomy, result.Items, len(result.CandidateRecords)+len(result.ChallengeRecords))
	if result.Aggregate.ItemCount != FrozenBaseQuestionCount || result.Aggregate.PassProposalRecordCount != FrozenPassProposalRecordCount ||
		result.Aggregate.Exceptions.BlockedSourceGap.Count != FrozenSourceGapCount ||
		result.Aggregate.Exceptions.ExternalApplicabilityUnresolved.Count != FrozenExternalUnresolvedCount ||
		result.Aggregate.Exceptions.SourceGapExternalUnresolvedOverlap.Count != FrozenSourceExternalOverlap ||
		codeCountValue(result.Aggregate.Distributions.ExtractionStateCounts, ExtractionCandidate) != FrozenExtractedCandidateCount ||
		codeCountValue(result.Aggregate.Distributions.ExtractionStateCounts, ExtractionExactSourceBacked) != FrozenExactSourceBackedCount {
		return ClassificationResult{}, fmt.Errorf("%w: frozen aggregate totals", ErrDigestMismatch)
	}
	result.AggregateDigest = result.Aggregate.AggregateDigest
	orderedDescriptors := append([]ModelDescriptor{}, input.ModelDescriptors...)
	sort.Slice(orderedDescriptors, func(i, j int) bool {
		return ComputeModelDescriptorDigest(orderedDescriptors[i]) < ComputeModelDescriptorDigest(orderedDescriptors[j])
	})
	result.RunReceipt = ClassificationRunReceipt{
		ClassificationRunID: input.ClassificationRunID, State: ClassificationRunSealed,
		TaxonomyVersion: taxonomy.Version, TaxonomyDigest: taxonomy.Digest,
		FixedInputDigests: input.FixedInputDigests, InputDigest: input.RunInputDigest,
		PromptDigest: input.PromptDigest, ModelDescriptors: orderedDescriptors,
		ModelDescriptorDigests: modelDigests, BatchManifestDigest: input.BatchManifestDigest,
		PassOneSealDigest: input.PassOneSealDigest, PassTwoSealDigest: input.PassTwoSealDigest,
		AggregateDigest: result.AggregateDigest,
	}
	result.RunReceipt.ClassificationRunDigest = digestExcludingJSONFields("AGA-CLASSIFICATION-RUN-V1", result.RunReceipt, "classificationRunDigest")
	result.ClassificationRunDigest = result.RunReceipt.ClassificationRunDigest
	result.PassOneSealReceipt = input.PassOneSealReceipt
	result.PassTwoSealReceipt = input.PassTwoSealReceipt
	for index := range result.Items {
		result.Items[index].AggregateDigest = result.AggregateDigest
		result.Items[index].ClassificationRunDigest = result.ClassificationRunDigest
	}
	return result, nil
}

func buildClassificationAggregate(taxonomy Taxonomy, items []SealedClassificationItem, passRecordCount int) ClassificationAggregate {
	distributions := ClassificationDistributions{
		AgreementConfidenceCounts:      codeCounts([]string{string(ConfidenceHigh), string(ConfidenceMedium), string(ConfidenceLow)}),
		ApplicabilityDispositionCounts: codeCounts(taxonomy.ApplicabilityDispositions),
		CanonicalTargetKindCounts:      codeCounts(taxonomy.CanonicalTargetKinds),
		DisagreementCodeCounts:         codeCounts(taxonomy.DisagreementCodes),
		EvidenceExpectationCodeCounts:  codeCounts(taxonomy.EvidenceExpectationCodes),
		ExternalProviderTypeCodeCounts: codeCounts(taxonomy.ExternalProviderTypes),
		ExtractionStateCounts:          codeCounts([]string{ExtractionCandidate, ExtractionExactSourceBacked}),
		InspectionProfileCodeCounts:    codeCounts(taxonomy.InspectionProfileCodes),
		InspectionTypeCodeCounts:       codeCounts(taxonomy.InspectionTypeCodes),
		MainDomainCodeCounts:           codeCounts(taxonomy.MainDomainCodes),
		RecommendationStateCounts:      codeCounts([]string{RecommendationAutoProposed, RecommendationManagerReview, RecommendationBlockedSourceGap}),
		TargetProfileCodeCounts:        codeCounts(taxonomy.TargetProfileCodes),
		TopicCodeCounts:                codeCounts(taxonomy.TopicCodes),
	}
	semanticDigests := make([]string, 0, len(items))
	blocked := make([]string, 0)
	externalUnresolved := make([]string, 0)
	managerReview := make([]string, 0)
	disagreement := make([]string, 0)
	overlap := make([]string, 0)
	for _, item := range items {
		semanticDigests = append(semanticDigests, item.ItemSemanticDigest)
		incrementCodeCount(distributions.AgreementConfidenceCounts, string(item.AgreementConfidence))
		incrementCodeCount(distributions.ApplicabilityDispositionCounts, item.Projection.ApplicabilityDisposition)
		incrementCodeCount(distributions.CanonicalTargetKindCounts, item.Projection.CanonicalTargetKind)
		incrementCodeCount(distributions.ExtractionStateCounts, item.ExtractionState)
		incrementCodeCount(distributions.MainDomainCodeCounts, item.Projection.MainDomainCode)
		incrementCodeCount(distributions.RecommendationStateCounts, item.RecommendationState)
		incrementCodeCount(distributions.TargetProfileCodeCounts, item.Projection.TargetProfileCode)
		for _, code := range item.PassDisagreementCodes {
			incrementCodeCount(distributions.DisagreementCodeCounts, code)
		}
		for _, code := range item.Projection.EvidenceExpectationCodes {
			incrementCodeCount(distributions.EvidenceExpectationCodeCounts, code)
		}
		for _, edge := range item.Projection.ExternalInvolvements {
			incrementCodeCount(distributions.ExternalProviderTypeCodeCounts, edge.ProviderTypeCode)
		}
		for _, code := range item.Projection.InspectionProfileCodes {
			incrementCodeCount(distributions.InspectionProfileCodeCounts, code)
		}
		for _, code := range item.Projection.InspectionTypeCodes {
			incrementCodeCount(distributions.InspectionTypeCodeCounts, code)
		}
		for _, code := range item.Projection.TopicCodes {
			incrementCodeCount(distributions.TopicCodeCounts, code)
		}
		identityDigest := digestValue("AGA-CLASSIFICATION-BASE-IDENTITY-V1", item.Identity)
		if item.RecommendationState == RecommendationBlockedSourceGap {
			blocked = append(blocked, identityDigest)
		}
		if item.ExternalApplicabilityUnresolved {
			externalUnresolved = append(externalUnresolved, identityDigest)
		}
		if item.RecommendationState == RecommendationManagerReview {
			managerReview = append(managerReview, identityDigest)
		}
		if len(item.PassDisagreementCodes) > 0 {
			disagreement = append(disagreement, identityDigest)
		}
		if item.QuestionSourceProposalGap && item.ExternalApplicabilityUnresolved {
			overlap = append(overlap, identityDigest)
		}
	}
	exceptions := ClassificationExceptions{
		BlockedSourceGap:                   exceptionInventory(blocked),
		ExternalApplicabilityUnresolved:    exceptionInventory(externalUnresolved),
		ManagerReviewRequired:              exceptionInventory(managerReview),
		PassDisagreement:                   exceptionInventory(disagreement),
		SourceGapExternalUnresolvedOverlap: exceptionInventory(overlap),
	}
	aggregate := ClassificationAggregate{
		ItemCount: len(items), PassProposalRecordCount: passRecordCount,
		OrderedItemSemanticDigests: nonNilSlice(semanticDigests), Distributions: distributions, Exceptions: exceptions,
	}
	aggregate.DistributionDigest = digestValue("AGA-CLASSIFICATION-DISTRIBUTIONS-V1", map[string]any{"distributions": distributions})
	aggregate.AggregateDigest = digestExcludingJSONFields("AGA-CLASSIFICATION-AGGREGATE-V1", aggregate, "aggregateDigest")
	return aggregate
}

func codeCounts(codes []string) []CodeCount {
	result := make([]CodeCount, len(codes))
	for index, code := range codes {
		result[index] = CodeCount{Code: code}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return result
}

func incrementCodeCount(counts []CodeCount, code string) {
	for index := range counts {
		if counts[index].Code == code {
			counts[index].Count++
			return
		}
	}
}

func codeCountValue(counts []CodeCount, code string) int {
	for _, count := range counts {
		if count.Code == code {
			return count.Count
		}
	}
	return -1
}

func exceptionInventory(identities []string) ExceptionInventory {
	identities = nonNilSlice(identities)
	return ExceptionInventory{
		Count: len(identities), OrderedIdentityDigests: identities,
		OrderedIdentityDigest: digestValue("AGA-CLASSIFICATION-EXCEPTION-IDENTITY-INVENTORY-V1", map[string]any{"count": len(identities), "orderedIdentityDigests": identities}),
	}
}

func disagreementCodes(taxonomy Taxonomy, candidate, challenge ProposalProjection) []string {
	mapping := map[string]string{
		"activityQualifiers":       "ACTIVITY_QUALIFIER_DISAGREEMENT",
		"applicabilityDisposition": "APPLICABILITY_DISAGREEMENT",
		"canonicalTargetKind":      "CANONICAL_TARGET_KIND_DISAGREEMENT",
		"evidenceExpectationCodes": "EVIDENCE_EXPECTATION_DISAGREEMENT",
		"externalInvolvements":     "EXTERNAL_INVOLVEMENT_DISAGREEMENT",
		"inspectionProfileCodes":   "INSPECTION_PROFILE_DISAGREEMENT",
		"inspectionTypeCodes":      "INSPECTION_TYPE_DISAGREEMENT",
		"mainDomainCode":           "MAIN_DOMAIN_DISAGREEMENT",
		"operationQualifiers":      "OPERATION_QUALIFIER_DISAGREEMENT",
		"targetProfileCode":        "TARGET_PROFILE_DISAGREEMENT",
		"topicCodes":               "TOPIC_SET_DISAGREEMENT",
	}
	result := make([]string, 0)
	for _, field := range taxonomy.ProposalFields {
		if !ProjectionFieldEqual(candidate, challenge, field) {
			result = append(result, mapping[field])
		}
	}
	sort.Strings(result)
	return result
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; !exists {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
