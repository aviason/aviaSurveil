package agaapplicability

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

var draftReasonCodes = []string{
	"CLASSIFICATION_EXPERT_REVIEW",
	"MANAGER_EXACT_RESOLUTION",
	"MANAGER_SCOPE_DECISION",
	"SIMULATION_SOURCE_GAP_OVERRIDE",
	"SYNTHETIC_CANDIDATE_ADDED",
	"SYNTHETIC_CANDIDATE_REWORDED",
	"SYNTHETIC_LIFECYCLE_CORRECTION",
	"SYNTHETIC_LIFECYCLE_REVIEW",
	"SYNTHETIC_RESET",
}

// passBijectionError keeps the public sentinel stable while retaining a
// private, aggregate-safe validation category for connected runtime
// diagnostics. It never includes a question identity or emitted proposal
// content.
func passBijectionError(category string) error {
	return fmt.Errorf("%w: %s", ErrPassBijection, category)
}

func NewDraftFromClassification(classification ClassificationResult, generationID string) (Draft, error) {
	if err := validateFrozenTaxonomyAuthority(); err != nil {
		return Draft{}, err
	}
	if !serverIDPattern.MatchString(generationID) || classification.State != ClassificationRunSealed || !classificationRunIDPattern.MatchString(classification.ClassificationRunID) {
		return Draft{}, fmt.Errorf("%w: classification is not sealed", ErrDraftNotReady)
	}
	for name, value := range map[string]string{
		"taxonomyDigest":          classification.TaxonomyDigest,
		"aggregateDigest":         classification.AggregateDigest,
		"classificationRunDigest": classification.ClassificationRunDigest,
	} {
		if !validDigest(value) {
			return Draft{}, fmt.Errorf("%w: %s", ErrDigestMismatch, name)
		}
	}
	if len(classification.Items) != FrozenBaseQuestionCount || len(classification.CandidateRecords) != FrozenBaseQuestionCount || len(classification.ChallengeRecords) != FrozenBaseQuestionCount {
		return Draft{}, passBijectionError("record counts")
	}
	taxonomy := FrozenTaxonomy()
	promptDigest := classification.RunReceipt.PromptDigest
	if !validDigest(promptDigest) || classification.TaxonomyVersion != taxonomy.Version || classification.TaxonomyDigest != taxonomy.Digest || classification.InputDigest != ComputeRunInputDigestForPrompt(FrozenFixedInputDigests(), promptDigest) {
		return Draft{}, fmt.Errorf("%w: taxonomy", ErrDigestMismatch)
	}
	for _, item := range classification.Items {
		if !contains([]string{RecommendationAutoProposed, RecommendationManagerReview, RecommendationBlockedSourceGap}, item.RecommendationState) {
			return Draft{}, ErrUnknownCode
		}
	}
	expectedAggregate := buildClassificationAggregate(taxonomy, classification.Items, len(classification.CandidateRecords)+len(classification.ChallengeRecords))
	if !reflect.DeepEqual(expectedAggregate, classification.Aggregate) || classification.AggregateDigest != expectedAggregate.AggregateDigest {
		return Draft{}, fmt.Errorf("%w: classification aggregate", ErrDigestMismatch)
	}
	if classification.Aggregate.ItemCount != FrozenBaseQuestionCount || classification.Aggregate.PassProposalRecordCount != FrozenPassProposalRecordCount || classification.Aggregate.Exceptions.BlockedSourceGap.Count != FrozenSourceGapCount || classification.Aggregate.Exceptions.ExternalApplicabilityUnresolved.Count != FrozenExternalUnresolvedCount || classification.Aggregate.Exceptions.SourceGapExternalUnresolvedOverlap.Count != FrozenSourceExternalOverlap || codeCountValue(classification.Aggregate.Distributions.ExtractionStateCounts, ExtractionCandidate) != FrozenExtractedCandidateCount || codeCountValue(classification.Aggregate.Distributions.ExtractionStateCounts, ExtractionExactSourceBacked) != FrozenExactSourceBackedCount {
		return Draft{}, fmt.Errorf("%w: frozen aggregate counts", ErrDigestMismatch)
	}
	sealedIdentityOrder := make([]BaseIdentity, len(classification.Items))
	for index := range classification.Items {
		sealedIdentityOrder[index] = classification.Items[index].Identity
	}
	if digestValue("AGA-CLASSIFICATION-ORDERED-IDENTITIES-V1", sealedIdentityOrder) != FrozenOrderedIdentityDigest || classification.PassOneSealReceipt.PassSealDigest != classification.RunReceipt.PassOneSealDigest || classification.PassTwoSealReceipt.PassSealDigest != classification.RunReceipt.PassTwoSealDigest || classification.PassOneSealReceipt.PromptDigest != promptDigest || classification.PassTwoSealReceipt.PromptDigest != promptDigest || !validSealedPassReceipt(classification.PassOneSealReceipt, PassCandidate, classification.CandidateRecords) || !validSealedPassReceipt(classification.PassTwoSealReceipt, PassChallenge, classification.ChallengeRecords) {
		return Draft{}, fmt.Errorf("%w: pass seal graph", ErrDigestMismatch)
	}
	receipt := classification.RunReceipt
	candidateRunID := classification.PassOneSealReceipt.ClassificationRunID
	challengeRunID := classification.PassTwoSealReceipt.ClassificationRunID
	if !classificationRunIDPattern.MatchString(candidateRunID) || !classificationRunIDPattern.MatchString(challengeRunID) {
		return Draft{}, fmt.Errorf("%w: pass run identity", ErrPassBijection)
	}
	if receipt.ClassificationRunID != classification.ClassificationRunID || receipt.State != ClassificationRunSealed || receipt.TaxonomyVersion != taxonomy.Version || receipt.TaxonomyDigest != taxonomy.Digest || receipt.InputDigest != classification.InputDigest || receipt.AggregateDigest != classification.AggregateDigest || receipt.PromptDigest != promptDigest || receipt.BatchManifestDigest != FrozenBatchManifestDigest || receipt.ClassificationRunDigest != classification.ClassificationRunDigest || receipt.ClassificationRunDigest != digestExcludingJSONFields("AGA-CLASSIFICATION-RUN-V1", receipt, "classificationRunDigest") || !reflect.DeepEqual(receipt.FixedInputDigests, FrozenFixedInputDigests()) {
		return Draft{}, fmt.Errorf("%w: classification run receipt", ErrDigestMismatch)
	}
	if len(receipt.ModelDescriptorDigests) < 1 || !validDigestSet(receipt.ModelDescriptorDigests, 2) || !reflect.DeepEqual(uniqueSorted(receipt.ModelDescriptorDigests), receipt.ModelDescriptorDigests) {
		return Draft{}, fmt.Errorf("%w: model descriptor digest set", ErrDigestMismatch)
	}
	for _, descriptor := range receipt.ModelDescriptors {
		if err := validateModelDescriptor(descriptor); err != nil {
			return Draft{}, ErrDigestMismatch
		}
	}
	candidate := make(map[string]PassProposalRecord, len(classification.CandidateRecords))
	challenge := make(map[string]PassProposalRecord, len(classification.ChallengeRecords))
	for _, record := range classification.CandidateRecords {
		if record.PassRole != PassCandidate || record.ClassificationRunID != candidateRunID || record.PromptDigest != promptDigest || !contains(receipt.ModelDescriptorDigests, record.ModelDescriptorDigest) || record.PassResultDigest != ComputePassResultDigest(record) {
			return Draft{}, passBijectionError("candidate record provenance")
		}
		if _, duplicate := candidate[record.Identity.Key()]; duplicate {
			return Draft{}, ErrDuplicateIdentity
		}
		candidate[record.Identity.Key()] = record
	}
	for _, record := range classification.ChallengeRecords {
		if record.PassRole != PassChallenge || record.ClassificationRunID != challengeRunID || record.PromptDigest != promptDigest || !contains(receipt.ModelDescriptorDigests, record.ModelDescriptorDigest) || record.PassResultDigest != ComputePassResultDigest(record) {
			return Draft{}, passBijectionError("challenge record provenance")
		}
		if _, duplicate := challenge[record.Identity.Key()]; duplicate {
			return Draft{}, ErrDuplicateIdentity
		}
		challenge[record.Identity.Key()] = record
	}
	draftID := "aga-ws-draft-" + strings.TrimPrefix(generationID, "aga-ws-")
	if !serverIDPattern.MatchString(draftID) {
		return Draft{}, ErrWorkspaceIdentityAlias
	}
	draft := Draft{
		DraftID:                   draftID,
		GenerationID:              generationID,
		GenerationState:           GenerationActive,
		Revision:                  1,
		State:                     DraftWorking,
		ClassificationRunID:       classification.ClassificationRunID,
		ClassificationRunState:    classification.State,
		ClassificationRunDigest:   classification.ClassificationRunDigest,
		AggregateDigest:           classification.AggregateDigest,
		TaxonomyDigest:            classification.TaxonomyDigest,
		TaxonomyVersion:           classification.TaxonomyVersion,
		PackageVersion:            FrozenPackageVersion,
		PackageJSONSHA256:         FrozenPackageJSONSHA256,
		ClassificationInputDigest: classification.InputDigest,
		FixedInputDigests:         classification.RunReceipt.FixedInputDigests,
		BaseQuestionCount:         len(classification.Items),
		ClassificationItemCount:   len(classification.Items),
		ReadinessEvents:           []ReadinessEvent{},
		Items:                     make([]DraftItem, 0, len(classification.Items)),
		sealedCandidateRecords:    make(map[string]PassProposalRecord, len(classification.CandidateRecords)),
		sealedChallengeRecords:    make(map[string]PassProposalRecord, len(classification.ChallengeRecords)),
	}
	for key, record := range candidate {
		draft.sealedCandidateRecords[key] = clonePassProposalRecord(record)
	}
	for key, record := range challenge {
		draft.sealedChallengeRecords[key] = clonePassProposalRecord(record)
	}
	identities := make(map[string]struct{}, len(classification.Items))
	for index, sealed := range classification.Items {
		if err := sealed.Identity.Validate(); err != nil {
			return Draft{}, err
		}
		key := sealed.Identity.Key()
		if _, exists := identities[key]; exists {
			return Draft{}, ErrDuplicateIdentity
		}
		identities[key] = struct{}{}
		if sealed.ItemSemanticDigest != ComputeItemSemanticDigest(sealed) || sealed.ClassificationRunDigest != classification.ClassificationRunDigest || sealed.AggregateDigest != classification.AggregateDigest || sealed.InputDigest != classification.InputDigest || sealed.TaxonomyDigest != taxonomy.Digest || sealed.PromptDigest != promptDigest || !reflect.DeepEqual(sealed.ModelDescriptorDigests, receipt.ModelDescriptorDigests) {
			return Draft{}, fmt.Errorf("%w: sealed item graph", ErrDigestMismatch)
		}
		candidateRecord, candidateExists := candidate[key]
		challengeRecord, challengeExists := challenge[key]
		if !candidateExists || !challengeExists || sealed.PassOneResultDigest != candidateRecord.PassResultDigest || sealed.PassTwoResultDigest != challengeRecord.PassResultDigest || sealed.PassOneRunID != candidateRecord.PassRunID || sealed.PassTwoRunID != challengeRecord.PassRunID {
			return Draft{}, passBijectionError("sealed item pass references")
		}
		candidateProjection := candidateRecord.ProposalProjection
		challengeProjection := challengeRecord.ProposalProjection
		candidateProjection, err := normalizeProjection(taxonomy, candidateProjection)
		if err != nil {
			return Draft{}, err
		}
		challengeProjection, err = normalizeProjection(taxonomy, challengeProjection)
		if err != nil {
			return Draft{}, err
		}
		currentProjection, err := normalizeProjection(taxonomy, sealed.Projection)
		if err != nil {
			return Draft{}, err
		}
		if !reflect.DeepEqual(currentProjection, candidateProjection) {
			return Draft{}, passBijectionError("candidate projection")
		}
		outcome := DeriveOutcome(taxonomy, candidateProjection, challengeProjection, candidateRecord.ConfidenceEvidence, challengeRecord.ConfidenceEvidence, sealed.QuestionSourceProposalGap, sealed.ExternalApplicabilityUnresolved)
		if sealed.AgreementConfidence != outcome.AgreementConfidence || sealed.RecommendationState != outcome.RecommendationState || !reflect.DeepEqual(sealed.RationaleCodes, candidateRecord.RationaleCodes) || !reflect.DeepEqual(sealed.ConfidenceEvidence, candidateRecord.ConfidenceEvidence) || !reflect.DeepEqual(sealed.SourceRefs, candidateRecord.SourceRefs) {
			return Draft{}, passBijectionError("sealed item outcome")
		}
		confidence := sealed.AgreementConfidence
		candidateCopy := clonePassProposalRecord(candidateRecord)
		challengeCopy := clonePassProposalRecord(challengeRecord)
		reference := BaseQuestionReference(sealed.Identity)
		reference.RootSequence = index + 1
		item := DraftItem{
			QuestionRef:                     reference,
			Origin:                          DraftItemOriginSealedBase,
			CurrentProjection:               currentProjection,
			SealedAgreementConfidence:       &confidence,
			DraftAgreementConfidence:        &confidence,
			RecommendationState:             sealed.RecommendationState,
			QuestionSourceProposalGap:       sealed.QuestionSourceProposalGap,
			SourceMappingState:              sealed.SourceMappingState,
			SourceAuthorityState:            sealed.SourceAuthorityState,
			RiskClassificationState:         sealed.RiskClassificationState,
			ExternalApplicabilityUnresolved: sealed.ExternalApplicabilityUnresolved,
			DecisionState:                   sealed.DecisionState,
			ExtractionState:                 sealed.ExtractionState,
			sealedGovernance:                cloneJSON(sealed.GovernanceState),
			sealedRecommendationState:       sealed.RecommendationState,
			Current:                         true,
			candidatePassRecord:             &candidateCopy,
			challengePassRecord:             &challengeCopy,
			candidatePassResultDigest:       sealed.PassOneResultDigest,
			challengePassResultDigest:       sealed.PassTwoResultDigest,
			sealedBaseRootSequence:          index + 1,
		}
		switch sealed.RecommendationState {
		case RecommendationAutoProposed:
			include := DispositionInclude
			item.ReviewState = ReviewAutoPreselected
			item.Disposition = &include
		case RecommendationManagerReview, RecommendationBlockedSourceGap:
			item.ReviewState = ReviewPendingManager
		default:
			return Draft{}, fmt.Errorf("%w: recommendationState=%q", ErrUnknownCode, sealed.RecommendationState)
		}
		draft.Items = append(draft.Items, item)
	}
	draft.sealedPassGraphValidated = true
	draft.ContentDigest = ComputeDraftContentDigest(draft)
	return draft, nil
}

// HydrateDraftForRuntime restores the sealed pass and governance context that
// is intentionally omitted from the public draft JSON projection. A draft
// revision is persisted as an append-only public payload, while command
// validation still needs the immutable sealed classification inputs that were
// stored in the sibling classification run projection.
func HydrateDraftForRuntime(draft Draft, classification ClassificationResult) (Draft, error) {
	base, err := NewDraftFromClassification(classification, draft.GenerationID)
	if err != nil {
		return Draft{}, err
	}
	return HydrateDraftForRuntimeFromSealedDraft(draft, base)
}

// HydrateDraftForRuntimeFromSealedDraft restores the immutable hidden
// classification context from a previously validated sealed base draft. The
// connected PostgreSQL command path uses this form after validating the
// classification run once; the public behavior remains identical to
// HydrateDraftForRuntime while avoiding repeated 1,310-item pass validation on
// every append-only Draft command.
func HydrateDraftForRuntimeFromSealedDraft(draft, base Draft) (Draft, error) {
	if draft.GenerationID != base.GenerationID {
		return Draft{}, fmt.Errorf("%w: sealed base generation context", ErrDraftNotReady)
	}
	if draft.ClassificationRunID != base.ClassificationRunID {
		return Draft{}, fmt.Errorf("%w: sealed base classification context", ErrDraftNotReady)
	}
	if base.State != DraftWorking {
		return Draft{}, fmt.Errorf("%w: sealed base state context", ErrDraftNotReady)
	}
	if len(base.Items) != FrozenBaseQuestionCount {
		return Draft{}, fmt.Errorf("%w: sealed base item context", ErrDraftNotReady)
	}
	hydrated := cloneDraft(draft)
	hydrated.sealedPassGraphValidated = base.sealedPassGraphValidated
	hydrated.sealedCandidateRecords = base.sealedCandidateRecords
	hydrated.sealedChallengeRecords = base.sealedChallengeRecords
	sealedByKey := make(map[string]DraftItem, len(base.Items))
	for _, item := range base.Items {
		sealedByKey[item.QuestionRef.Key()] = item
	}
	for index := range hydrated.Items {
		item := &hydrated.Items[index]
		sealed, found := sealedByKey[item.QuestionRef.Key()]
		if !found {
			if item.QuestionRef.Workspace != nil {
				item.SourceMappingState = SourceMappingRequired
				item.SourceAuthorityState = SourceAuthorityNotAttested
				item.RiskClassificationState = RiskExpertReviewRequired
			}
			continue
		}
		item.SealedAgreementConfidence = cloneConfidence(sealed.SealedAgreementConfidence)
		if item.QuestionRef.Base != nil && sealed.QuestionRef.Base != nil {
			// Base references intentionally omit rootSequence from their public
			// union shape. Restore the sealed, append-only ordering metadata before
			// command validation and content-digest recomputation.
			item.QuestionRef.RootSequence = sealed.QuestionRef.RootSequence
		}
		item.SourceMappingState = sealed.SourceMappingState
		item.SourceAuthorityState = sealed.SourceAuthorityState
		item.RiskClassificationState = sealed.RiskClassificationState
		item.ExternalApplicabilityUnresolved = sealed.ExternalApplicabilityUnresolved
		item.DecisionState = sealed.DecisionState
		item.ExtractionState = sealed.ExtractionState
		item.sealedGovernance = cloneJSON(sealed.sealedGovernance)
		item.sealedRecommendationState = sealed.sealedRecommendationState
		item.candidatePassRecord = sealed.candidatePassRecord
		item.challengePassRecord = sealed.challengePassRecord
		item.candidatePassResultDigest = sealed.candidatePassResultDigest
		item.challengePassResultDigest = sealed.challengePassResultDigest
		item.sealedBaseRootSequence = sealed.sealedBaseRootSequence
	}
	if ComputeDraftContentDigest(hydrated) != hydrated.ContentDigest {
		return Draft{}, ErrDraftConflict
	}
	if err := validateDraftQuestionGraphWithTaxonomy(hydrated, FrozenTaxonomy()); err != nil {
		return Draft{}, err
	}
	return hydrated, nil
}

func cloneConfidence(value *Confidence) *Confidence {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func clonePassPointer(value *PassProposalRecord) *PassProposalRecord {
	if value == nil {
		return nil
	}
	copy := clonePassProposalRecord(*value)
	return &copy
}

func validSealedPassReceipt(receipt PassSealReceipt, role PassRole, records []PassProposalRecord) bool {
	if receipt.PassRole != role || receipt.BatchCount != len(frozenBatchItemCounts) || receipt.ItemCount != FrozenBaseQuestionCount || len(receipt.OrderedInputDigests) != len(frozenBatchItemCounts) || len(receipt.OrderedBatchOutputDigests) != len(frozenBatchItemCounts) || len(receipt.OrderedPassResultDigests) != FrozenBaseQuestionCount || receipt.PassSealDigest != ComputePassSeal(receipt) || len(records) != FrozenBaseQuestionCount {
		return false
	}
	for index, record := range records {
		if receipt.OrderedPassResultDigests[index] != record.PassResultDigest || !validDigest(record.PassResultDigest) {
			return false
		}
	}
	return true
}

func ComputeDraftContentDigest(draft Draft) string {
	encoded, err := json.Marshal(draft)
	if err != nil {
		panic(err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		panic(err)
	}
	delete(object, "contentDigest")
	delete(object, "readinessEvents")
	delete(object, "currentReadinessEventId")
	baseOrder := make([]map[string]any, 0)
	for _, item := range draft.Items {
		if item.QuestionRef.Base != nil {
			baseOrder = append(baseOrder, map[string]any{"identity": item.QuestionRef.Base, "rootSequence": item.QuestionRef.RootSequence})
		}
	}
	object["sealedBaseOrder"] = baseOrder
	return digestValue("AGA-CLASSIFICATION-DRAFT-CONTENT-V1", object)
}

func ComputeWorkspaceBodyDigest(body string) string {
	if !utf8.ValidString(body) {
		return ""
	}
	hash := sha256.Sum256([]byte(body))
	return "sha256:" + hex.EncodeToString(hash[:])
}

func cloneQuestionRef(reference QuestionRef) QuestionRef {
	result := reference
	if reference.Base != nil {
		value := *reference.Base
		result.Base = &value
	}
	if reference.Workspace != nil {
		value := *reference.Workspace
		if value.ParentQuestionKey != nil {
			parent := *value.ParentQuestionKey
			if parent.Base != nil {
				base := *parent.Base
				parent.Base = &base
			}
			value.ParentQuestionKey = &parent
		}
		result.Workspace = &value
	}
	return result
}

func clonePassProposalRecord(record PassProposalRecord) PassProposalRecord {
	result := record
	result.ProposalProjection = cloneJSON(record.ProposalProjection)
	result.RationaleCodes = cloneJSON(record.RationaleCodes)
	result.ConfidenceEvidence = cloneJSON(record.ConfidenceEvidence)
	result.SourceRefs = cloneJSON(record.SourceRefs)
	return result
}

func cloneDraftItem(item DraftItem) DraftItem {
	result := item
	result.QuestionRef = cloneQuestionRef(item.QuestionRef)
	result.CurrentProjection = cloneJSON(item.CurrentProjection)
	if item.ProposalResolution != nil {
		resolution := *item.ProposalResolution
		if resolution.ProposalProjection != nil {
			projection := cloneJSON(*resolution.ProposalProjection)
			resolution.ProposalProjection = &projection
		}
		result.ProposalResolution = &resolution
	}
	if item.SealedAgreementConfidence != nil {
		value := *item.SealedAgreementConfidence
		result.SealedAgreementConfidence = &value
	}
	if item.candidatePassRecord != nil {
		value := clonePassProposalRecord(*item.candidatePassRecord)
		result.candidatePassRecord = &value
	}
	if item.challengePassRecord != nil {
		value := clonePassProposalRecord(*item.challengePassRecord)
		result.challengePassRecord = &value
	}
	if item.DraftAgreementConfidence != nil {
		value := *item.DraftAgreementConfidence
		result.DraftAgreementConfidence = &value
	}
	result.sealedBaseRootSequence = item.sealedBaseRootSequence
	result.sealedGovernance = cloneJSON(item.sealedGovernance)
	result.sealedRecommendationState = item.sealedRecommendationState
	if item.Disposition != nil {
		value := *item.Disposition
		result.Disposition = &value
	}
	return result
}

// cloneDraftItemForRuntime copies the mutable public/decision fields while
// retaining pointers to the immutable, already-validated sealed pass records.
// Those records are never modified by a Draft command; sharing them keeps the
// connected append-only command path bounded by the mutable Draft projection.
func cloneDraftItemForRuntime(item DraftItem) DraftItem {
	result := item
	result.QuestionRef = cloneQuestionRef(item.QuestionRef)
	result.CurrentProjection = cloneJSON(item.CurrentProjection)
	if item.ProposalResolution != nil {
		resolution := *item.ProposalResolution
		if resolution.ProposalProjection != nil {
			projection := cloneJSON(*resolution.ProposalProjection)
			resolution.ProposalProjection = &projection
		}
		result.ProposalResolution = &resolution
	}
	if item.SealedAgreementConfidence != nil {
		value := *item.SealedAgreementConfidence
		result.SealedAgreementConfidence = &value
	}
	if item.DraftAgreementConfidence != nil {
		value := *item.DraftAgreementConfidence
		result.DraftAgreementConfidence = &value
	}
	if item.Disposition != nil {
		value := *item.Disposition
		result.Disposition = &value
	}
	return result
}

func cloneDraft(draft Draft) Draft {
	if draft.sealedPassGraphValidated {
		result := draft
		result.Items = make([]DraftItem, len(draft.Items))
		for index, item := range draft.Items {
			result.Items[index] = cloneDraftItemForRuntime(item)
		}
		result.ReadinessEvents = append([]ReadinessEvent{}, draft.ReadinessEvents...)
		result.sealedCandidateRecords = draft.sealedCandidateRecords
		result.sealedChallengeRecords = draft.sealedChallengeRecords
		return result
	}
	result := draft
	result.Items = make([]DraftItem, len(draft.Items))
	for index, item := range draft.Items {
		result.Items[index] = cloneDraftItem(item)
	}
	result.ReadinessEvents = append([]ReadinessEvent{}, draft.ReadinessEvents...)
	result.sealedCandidateRecords = make(map[string]PassProposalRecord, len(draft.sealedCandidateRecords))
	for key, record := range draft.sealedCandidateRecords {
		result.sealedCandidateRecords[key] = clonePassProposalRecord(record)
	}
	result.sealedChallengeRecords = make(map[string]PassProposalRecord, len(draft.sealedChallengeRecords))
	for key, record := range draft.sealedChallengeRecords {
		result.sealedChallengeRecords[key] = clonePassProposalRecord(record)
	}
	return result
}

func cloneDraftForRuntimeCommand(draft Draft) Draft {
	result := draft
	result.Items = append([]DraftItem(nil), draft.Items...)
	result.ReadinessEvents = append([]ReadinessEvent{}, draft.ReadinessEvents...)
	return result
}

func draftEqual(left, right Draft) bool { return reflect.DeepEqual(left, right) }

func currentDraftItems(draft Draft) []DraftItem {
	result := make([]DraftItem, 0)
	for _, item := range draft.Items {
		if item.Current {
			if draft.sealedPassGraphValidated {
				result = append(result, cloneDraftItemForRuntime(item))
			} else {
				result = append(result, cloneDraftItem(item))
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].QuestionRef.RootSequence == result[j].QuestionRef.RootSequence {
			return strings.Compare(result[i].QuestionRef.Key(), result[j].QuestionRef.Key()) < 0
		}
		return result[i].QuestionRef.RootSequence < result[j].QuestionRef.RootSequence
	})
	return result
}

func replaceCurrentDraftItem(draft *Draft, replacement DraftItem) bool {
	for index := range draft.Items {
		if draft.Items[index].Current && draft.Items[index].QuestionRef.Key() == replacement.QuestionRef.Key() {
			if draft.sealedPassGraphValidated {
				draft.Items[index] = cloneDraftItemForRuntime(replacement)
			} else {
				draft.Items[index] = cloneDraftItem(replacement)
			}
			return true
		}
	}
	return false
}

func ValidateQuestionRef(reference QuestionRef) error {
	switch reference.Origin {
	case QuestionOriginBase:
		if reference.Base == nil || reference.Workspace != nil || reference.RootSequence < 0 {
			return ErrQuestionReferenceUnion
		}
		return reference.Base.Validate()
	case QuestionOriginWorkspace:
		if reference.Base != nil || reference.Workspace == nil || reference.RootSequence < 1 || reference.RootSequence != reference.Workspace.RootSequence {
			return ErrQuestionReferenceUnion
		}
		workspace := reference.Workspace
		_, offset := workspace.CreatedAt.Zone()
		if !serverIDPattern.MatchString(workspace.GenerationID) || !serverIDPattern.MatchString(workspace.RootID) || !serverIDPattern.MatchString(workspace.VersionID) || !serverIDPattern.MatchString(workspace.ProposalID) || !validDigest(workspace.BodyDigest) || !subjectIDPattern.MatchString(workspace.ActorSubjectID) || workspace.CreatedAt.IsZero() || offset != 0 || !contains([]string{"SYNTHETIC_CANDIDATE_ADDED", "SYNTHETIC_CANDIDATE_REWORDED"}, workspace.ReasonCode) {
			return ErrQuestionReferenceUnion
		}
		if workspace.ParentQuestionKey == nil && workspace.ReasonCode != "SYNTHETIC_CANDIDATE_ADDED" {
			return ErrParentQuestionKey
		}
		if workspace.ParentQuestionKey != nil && workspace.ReasonCode == "SYNTHETIC_CANDIDATE_ADDED" {
			return ErrParentQuestionKey
		}
		if workspace.ParentQuestionKey != nil {
			parent := workspace.ParentQuestionKey
			hasBase := parent.Base != nil
			hasWorkspace := parent.WorkspaceGenerationID != "" || parent.WorkspaceRootID != "" || parent.WorkspaceVersionID != "" || parent.WorkspaceProposalID != "" || parent.WorkspaceRootSequence != 0 || parent.WorkspaceBodyDigest != ""
			if hasBase == hasWorkspace {
				return ErrParentQuestionKey
			}
			if hasBase {
				if err := parent.Base.Validate(); err != nil {
					return ErrParentQuestionKey
				}
			} else if !serverIDPattern.MatchString(parent.WorkspaceGenerationID) || !serverIDPattern.MatchString(parent.WorkspaceRootID) || !serverIDPattern.MatchString(parent.WorkspaceVersionID) || !serverIDPattern.MatchString(parent.WorkspaceProposalID) || parent.WorkspaceRootSequence < 1 || !validDigest(parent.WorkspaceBodyDigest) {
				return ErrParentQuestionKey
			}
		}
		return nil
	default:
		return ErrQuestionReferenceUnion
	}
}

func ValidateWorkspaceQuestionRefs(refs []WorkspaceQuestionRef, generationID string) error {
	if !serverIDPattern.MatchString(generationID) {
		return ErrCrossGenerationParent
	}
	byVersion := make(map[string]WorkspaceQuestionRef, len(refs))
	proposalIDs := make(map[string]struct{}, len(refs))
	rootSequences := make(map[int]string)
	rootInitialCounts := make(map[string]int)
	baseInitialRoots := make(map[string]string)
	for _, reference := range refs {
		if reference.GenerationID != generationID {
			return ErrCrossGenerationParent
		}
		if !serverIDPattern.MatchString(reference.RootID) || !serverIDPattern.MatchString(reference.VersionID) || !serverIDPattern.MatchString(reference.ProposalID) || reference.RootSequence < 1 {
			return ErrQuestionReferenceUnion
		}
		if _, exists := byVersion[reference.VersionID]; exists {
			return ErrWorkspaceIdentityAlias
		}
		if _, exists := proposalIDs[reference.ProposalID]; exists {
			return ErrWorkspaceIdentityAlias
		}
		byVersion[reference.VersionID] = reference
		proposalIDs[reference.ProposalID] = struct{}{}
		if existingRoot, exists := rootSequences[reference.RootSequence]; exists && existingRoot != reference.RootID {
			return ErrWorkspaceIdentityAlias
		}
		rootSequences[reference.RootSequence] = reference.RootID
		if reference.ParentQuestionKey == nil || reference.ParentQuestionKey.Base != nil {
			rootInitialCounts[reference.RootID]++
			if reference.ParentQuestionKey != nil && reference.ParentQuestionKey.Base != nil {
				baseKey := reference.ParentQuestionKey.Base.Key()
				if root, exists := baseInitialRoots[baseKey]; exists && root != reference.RootID {
					return ErrWorkspaceIdentityAlias
				}
				baseInitialRoots[baseKey] = reference.RootID
			}
		}
	}
	for _, count := range rootInitialCounts {
		if count != 1 {
			return ErrWorkspaceIdentityAlias
		}
	}
	childrenByVersion := make(map[string]int)
	for _, reference := range refs {
		parent := reference.ParentQuestionKey
		if parent == nil || parent.Base != nil {
			continue
		}
		if parent.WorkspaceGenerationID != generationID {
			return ErrCrossGenerationParent
		}
		if parent.WorkspaceVersionID == reference.VersionID {
			return ErrCyclicParent
		}
		parentReference, exists := byVersion[parent.WorkspaceVersionID]
		if !exists {
			return ErrMissingParent
		}
		if parent.WorkspaceRootID != reference.RootID || parentReference.RootID != reference.RootID || parent.WorkspaceRootSequence != reference.RootSequence || parentReference.RootSequence != reference.RootSequence {
			return ErrCrossRootParent
		}
		if parent.WorkspaceProposalID != parentReference.ProposalID || parent.WorkspaceBodyDigest != parentReference.BodyDigest {
			return ErrParentQuestionKey
		}
		childrenByVersion[parent.WorkspaceVersionID]++
		if childrenByVersion[parent.WorkspaceVersionID] > 1 {
			return ErrWorkspaceIdentityAlias
		}
	}
	for _, reference := range refs {
		seen := make(map[string]struct{})
		current := reference
		for current.ParentQuestionKey != nil && current.ParentQuestionKey.Base == nil {
			if _, exists := seen[current.VersionID]; exists {
				return ErrCyclicParent
			}
			seen[current.VersionID] = struct{}{}
			parent, exists := byVersion[current.ParentQuestionKey.WorkspaceVersionID]
			if !exists {
				break
			}
			current = parent
		}
	}
	return nil
}

func ApplyDraftCommand(draft Draft, command DraftCommand, allocator IDAllocator) (Draft, error) {
	if command.OperationID == "" || command.IdempotencyKey == "" || command.ExpectedGenerationID != draft.GenerationID {
		return Draft{}, ErrCommandEnvelope
	}
	if command.ExpectedRevision != draft.Revision || command.ExpectedContentDigest != draft.ContentDigest || ComputeDraftContentDigest(draft) != draft.ContentDigest {
		return Draft{}, ErrDraftConflict
	}
	if draft.GenerationState != GenerationActive || draft.ClassificationRunState != ClassificationRunSealed || (draft.State != DraftWorking && draft.State != DraftReadyForDemoSimulation) {
		return Draft{}, ErrDraftConflict
	}
	if err := validateDraftReason(command.Action, command.ReasonCode); err != nil {
		return Draft{}, err
	}
	if err := validateDraftQuestionGraph(draft); err != nil {
		return Draft{}, err
	}
	next := cloneDraft(draft)
	taxonomy := FrozenTaxonomy()
	var target DraftItem
	var targetFound bool
	if command.Action != DraftAddCandidate && command.Action != DraftMarkReady {
		for _, item := range currentDraftItems(next) {
			if item.QuestionRef.Key() == command.TargetQuestionKey {
				target, targetFound = item, true
				break
			}
		}
		if !targetFound {
			return Draft{}, ErrNonCurrentQuestion
		}
	}
	if command.Action == DraftInclude && target.QuestionSourceProposalGap && command.ReasonCode == "" {
		return Draft{}, ErrReasonRequired
	}
	if command.Action == DraftInclude && target.QuestionSourceProposalGap && command.ReasonCode != "SIMULATION_SOURCE_GAP_OVERRIDE" {
		return Draft{}, ErrInvalidReason
	}
	if draft.State == DraftReadyForDemoSimulation && command.Action != DraftMarkReady {
		next.State = DraftWorking
		next.CurrentReadinessEventID = ""
	}
	markingReady := false

	switch command.Action {
	case DraftRetain:
		// A retain command records a manager decision without changing semantics.
	case DraftInclude, DraftExclude, DraftDefer:
		if target.QuestionRef.Workspace != nil && target.ProposalResolution == nil {
			return Draft{}, ErrInvalidResolution
		}
		disposition := map[DraftAction]string{
			DraftInclude: DispositionInclude,
			DraftExclude: DispositionExclude,
			DraftDefer:   DispositionDefer,
		}[command.Action]
		target.Disposition = &disposition
		target.ReviewState = ReviewManagerDisposed
		replaceCurrentDraftItem(&next, target)
	case DraftReclassifyMainDomain:
		if err := validateCode(taxonomy.MainDomainCodes, command.MainDomainCode, "mainDomainCode"); err != nil {
			return Draft{}, err
		}
		target.CurrentProjection.MainDomainCode = command.MainDomainCode
		if err := ValidateProjection(taxonomy, target.CurrentProjection); err != nil {
			return Draft{}, err
		}
		setExactProposalResolution(&target)
		demoteSemanticEdit(&target)
		replaceCurrentDraftItem(&next, target)
	case DraftAddTopic:
		if err := validateCode(taxonomy.TopicCodes, command.TopicCode, "topicCode"); err != nil {
			return Draft{}, err
		}
		if contains(target.CurrentProjection.TopicCodes, command.TopicCode) {
			return Draft{}, fmt.Errorf("%w: topic %s", ErrDuplicateProposalValue, command.TopicCode)
		}
		target.CurrentProjection.TopicCodes = append(target.CurrentProjection.TopicCodes, command.TopicCode)
		target.CurrentProjection, _ = normalizeProjection(taxonomy, target.CurrentProjection)
		setExactProposalResolution(&target)
		demoteSemanticEdit(&target)
		replaceCurrentDraftItem(&next, target)
	case DraftRemoveTopic:
		if err := validateCode(taxonomy.TopicCodes, command.TopicCode, "topicCode"); err != nil {
			return Draft{}, err
		}
		updated := removeString(target.CurrentProjection.TopicCodes, command.TopicCode)
		if len(updated) == len(target.CurrentProjection.TopicCodes) {
			return Draft{}, fmt.Errorf("%w: topic removal", ErrInvalidResolution)
		}
		target.CurrentProjection.TopicCodes = updated
		target.CurrentProjection, _ = normalizeProjection(taxonomy, target.CurrentProjection)
		setExactProposalResolution(&target)
		demoteSemanticEdit(&target)
		replaceCurrentDraftItem(&next, target)
	case DraftResolveClassificationProposals:
		resolved, resolution, err := resolveProjection(taxonomy, draft, target, command)
		if err != nil {
			return Draft{}, err
		}
		target.CurrentProjection = resolved
		target.ProposalResolution = resolution
		demoteSemanticEdit(&target)
		replaceCurrentDraftItem(&next, target)
	case DraftAddCandidate:
		if allocator == nil || command.ExactProjection == nil || !subjectIDPattern.MatchString(command.ActorSubjectID) || command.WorkspaceBody == "" || command.CreatedAt.IsZero() {
			return Draft{}, fmt.Errorf("%w: add candidate fields", ErrInvalidResolution)
		}
		if command.WorkspaceBodyDigest != ComputeWorkspaceBodyDigest(command.WorkspaceBody) {
			return Draft{}, ErrDigestMismatch
		}
		projection, err := normalizeProjection(taxonomy, *command.ExactProjection)
		if err != nil {
			return Draft{}, err
		}
		sequence := 1
		for _, item := range currentDraftItems(next) {
			if item.QuestionRef.RootSequence >= sequence {
				sequence = item.QuestionRef.RootSequence + 1
			}
		}
		workspace := WorkspaceQuestionRef{
			GenerationID: next.GenerationID,
			RootID:       allocator.NextRootID(), VersionID: allocator.NextVersionID(), ProposalID: allocator.NextProposalID(),
			RootSequence: sequence, BodyDigest: command.WorkspaceBodyDigest, ActorSubjectID: command.ActorSubjectID,
			CreatedAt: normalizedCreatedAt(command.CreatedAt), ReasonCode: command.ReasonCode,
		}
		if err := ValidateQuestionRef(WorkspaceQuestionReference(workspace)); err != nil {
			return Draft{}, err
		}
		next.Items = append(next.Items, newWorkspaceDraftItem(workspace, projection))
	case DraftRewordCandidate:
		if allocator == nil || !subjectIDPattern.MatchString(command.ActorSubjectID) || command.WorkspaceBody == "" || command.CreatedAt.IsZero() {
			return Draft{}, fmt.Errorf("%w: reword fields", ErrInvalidResolution)
		}
		if command.WorkspaceBodyDigest != ComputeWorkspaceBodyDigest(command.WorkspaceBody) {
			return Draft{}, ErrDigestMismatch
		}
		if (target.QuestionRef.Workspace != nil && target.QuestionRef.Workspace.BodyDigest == command.WorkspaceBodyDigest) || (target.QuestionRef.Base != nil && target.QuestionRef.Base.TextDigest == command.WorkspaceBodyDigest) {
			return Draft{}, ErrByteIdenticalReword
		}
		parent := parentKeyForQuestion(target.QuestionRef)
		rootID := ""
		if target.QuestionRef.Workspace != nil {
			rootID = target.QuestionRef.Workspace.RootID
		} else {
			rootID = allocator.NextRootID()
		}
		workspace := WorkspaceQuestionRef{
			GenerationID: next.GenerationID,
			RootID:       rootID, VersionID: allocator.NextVersionID(), ProposalID: allocator.NextProposalID(),
			RootSequence: target.QuestionRef.RootSequence, BodyDigest: command.WorkspaceBodyDigest,
			ParentQuestionKey: parent, ActorSubjectID: command.ActorSubjectID,
			CreatedAt: normalizedCreatedAt(command.CreatedAt), ReasonCode: command.ReasonCode,
		}
		if err := ValidateQuestionRef(WorkspaceQuestionReference(workspace)); err != nil {
			return Draft{}, err
		}
		for index := range next.Items {
			if next.Items[index].Current && next.Items[index].QuestionRef.Key() == target.QuestionRef.Key() {
				next.Items[index].Current = false
			}
		}
		successor := target
		successor.QuestionRef = WorkspaceQuestionReference(workspace)
		successor.Origin = DraftItemOriginReworded
		successor.ProposalResolution = nil
		successor.SealedAgreementConfidence = nil
		successor.candidatePassRecord = nil
		successor.challengePassRecord = nil
		successor.candidatePassResultDigest = ""
		successor.challengePassResultDigest = ""
		successor.Current = true
		markSourceGap(&successor)
		next.Items = append(next.Items, successor)
	case DraftMarkReady:
		if draft.State != DraftWorking || readinessEventIDExists(draft, command.ReadinessEventID) || !serverIDPattern.MatchString(command.ReadinessEventID) || !subjectIDPattern.MatchString(command.ActorSubjectID) || command.CreatedAt.IsZero() || !validDigest(command.ProviderScopeProfileDigest) {
			return Draft{}, ErrDraftNotReady
		}
		if err := validateReadinessDraft(next, command.ReasonCode); err != nil {
			return Draft{}, err
		}
		next.State = DraftReadyForDemoSimulation
		markingReady = true
	default:
		return Draft{}, ErrUnknownCode
	}

	if err := validateDraftQuestionGraph(next); err != nil {
		return Draft{}, err
	}
	next.Revision++
	next.ContentDigest = ComputeDraftContentDigest(next)
	if markingReady {
		event := ReadinessEvent{
			ReadinessEventID: command.ReadinessEventID, GenerationID: next.GenerationID,
			ClassificationRunID: next.ClassificationRunID, ClassificationRunDigest: next.ClassificationRunDigest,
			TaxonomyVersion: next.TaxonomyVersion, TaxonomyDigest: next.TaxonomyDigest,
			DraftID: next.DraftID, DraftRevision: next.Revision, DraftContentDigest: next.ContentDigest,
			ProviderScopeProfileDigest: command.ProviderScopeProfileDigest, ActorSubjectID: command.ActorSubjectID,
			ReasonCode: command.ReasonCode, CreatedAt: command.CreatedAt.UTC(),
		}
		event.ReadinessEventDigest = digestExcludingJSONFields("AGA-DEMO-READINESS-EVENT-V1", event, "readinessEventDigest")
		next.ReadinessEvents = append(next.ReadinessEvents, event)
		next.CurrentReadinessEventID = event.ReadinessEventID
	}
	return next, nil
}

// ApplyDraftCommandFromValidatedRuntime applies the high-volume manager
// disposition command against a draft whose public payload and sealed graph
// were fully validated during runtime hydration. The command changes only the
// target disposition/review state, so the immutable graph and every untouched
// item remain covered by that earlier validation. All other command types use
// the complete command path above.
func ApplyDraftCommandFromValidatedRuntime(draft Draft, command DraftCommand, allocator IDAllocator) (Draft, error) {
	if !draft.sealedPassGraphValidated || command.Action != DraftInclude {
		return ApplyDraftCommand(draft, command, allocator)
	}
	if command.OperationID == "" || command.IdempotencyKey == "" || command.ExpectedGenerationID != draft.GenerationID {
		return Draft{}, ErrCommandEnvelope
	}
	if command.ExpectedRevision != draft.Revision || command.ExpectedContentDigest != draft.ContentDigest {
		return Draft{}, ErrDraftConflict
	}
	if !draftAcceptsCommands(draft) {
		return Draft{}, ErrDraftConflict
	}
	if err := validateDraftReason(command.Action, command.ReasonCode); err != nil {
		return Draft{}, err
	}

	next := cloneDraftForRuntimeCommand(draft)
	taxonomy := FrozenTaxonomy()
	targetIndex := -1
	for index := range next.Items {
		if next.Items[index].Current && next.Items[index].QuestionRef.Key() == command.TargetQuestionKey {
			targetIndex = index
			break
		}
	}
	if targetIndex < 0 {
		return Draft{}, ErrNonCurrentQuestion
	}
	target := cloneDraftItemForRuntime(next.Items[targetIndex])
	if target.QuestionSourceProposalGap && command.ReasonCode == "" {
		return Draft{}, ErrReasonRequired
	}
	if target.QuestionSourceProposalGap && command.ReasonCode != "SIMULATION_SOURCE_GAP_OVERRIDE" {
		return Draft{}, ErrInvalidReason
	}
	if target.QuestionRef.Workspace != nil && target.ProposalResolution == nil {
		return Draft{}, ErrInvalidResolution
	}
	disposition := DispositionInclude
	target.Disposition = &disposition
	target.ReviewState = ReviewManagerDisposed
	next.Items[targetIndex] = target
	if next.State == DraftReadyForDemoSimulation {
		next.State = DraftWorking
		next.CurrentReadinessEventID = ""
	}
	if err := validateDraftItemStateWithTaxonomy(next, target, taxonomy); err != nil {
		return Draft{}, err
	}
	next.Revision++
	next.ContentDigest = ComputeDraftContentDigest(next)
	return next, nil
}

// ApplyDraftDispositionBatchFromValidatedRuntime applies a server-created
// selection batch in one validated runtime pass. The batch is still revision
// counted once per affected item, but the immutable sealed graph is cloned and
// the content digest is recomputed only once for the atomic successor.
func ApplyDraftDispositionBatchFromValidatedRuntime(draft Draft, commands []DraftCommand) (Draft, error) {
	if len(commands) == 0 || !draft.sealedPassGraphValidated {
		return Draft{}, ErrCommandEnvelope
	}
	if !draftAcceptsCommands(draft) {
		return Draft{}, ErrDraftConflict
	}
	first := commands[0]
	if first.OperationID == "" || first.IdempotencyKey == "" || first.ExpectedGenerationID != draft.GenerationID || first.ExpectedRevision != draft.Revision || first.ExpectedContentDigest != draft.ContentDigest {
		return Draft{}, ErrDraftConflict
	}
	// A runtime disposition batch cannot change question references, parent
	// links, current-leaf state, or sealed pass records. The Draft arrived here
	// with sealedPassGraphValidated=true from the cold hydrate path, and every
	// predecessor batch preserves that invariant. Re-running the full 1,310-row
	// pass graph for each server-issued page would make the bounded batch API
	// unusable while adding no new validation surface.

	next := cloneDraftForRuntimeCommand(draft)
	indexByKey := make(map[string]int, len(next.Items))
	for index := range next.Items {
		if !next.Items[index].Current {
			continue
		}
		key := next.Items[index].QuestionRef.Key()
		if key == "" {
			return Draft{}, ErrNonCurrentQuestion
		}
		indexByKey[key] = index
	}
	for index, command := range commands {
		if command.OperationID == "" || command.IdempotencyKey == "" || command.ExpectedGenerationID != draft.GenerationID || command.ExpectedContentDigest == "" || (command.ExpectedRevision != draft.Revision && command.ExpectedRevision != draft.Revision+index) {
			return Draft{}, ErrCommandEnvelope
		}
		if command.Action != DraftInclude && command.Action != DraftExclude && command.Action != DraftDefer {
			return Draft{}, fmt.Errorf("%w: disposition batch action", ErrUnknownCode)
		}
		if err := validateDraftReason(command.Action, command.ReasonCode); err != nil {
			return Draft{}, err
		}
		targetIndex, ok := indexByKey[command.TargetQuestionKey]
		if !ok || !next.Items[targetIndex].Current {
			return Draft{}, ErrNonCurrentQuestion
		}
		target := cloneDraftItemForRuntime(next.Items[targetIndex])
		if command.Action == DraftInclude && target.QuestionSourceProposalGap && command.ReasonCode == "" {
			return Draft{}, ErrReasonRequired
		}
		if command.Action == DraftInclude && target.QuestionSourceProposalGap && command.ReasonCode != "SIMULATION_SOURCE_GAP_OVERRIDE" {
			return Draft{}, ErrInvalidReason
		}
		if target.QuestionRef.Workspace != nil && target.ProposalResolution == nil {
			return Draft{}, ErrInvalidResolution
		}
		disposition := map[DraftAction]string{DraftInclude: DispositionInclude, DraftExclude: DispositionExclude, DraftDefer: DispositionDefer}[command.Action]
		target.Disposition = &disposition
		target.ReviewState = ReviewManagerDisposed
		next.Items[targetIndex] = target
	}
	if next.State == DraftReadyForDemoSimulation {
		next.State = DraftWorking
		next.CurrentReadinessEventID = ""
	}
	next.Revision += len(commands)
	next.ContentDigest = ComputeDraftContentDigest(next)
	return next, nil
}

func normalizedCreatedAt(value time.Time) time.Time {
	return value.UTC()
}

func validateDraftReason(action DraftAction, reason string) error {
	required := action != DraftRetain
	if action == DraftInclude {
		required = false
	}
	if required && reason == "" {
		return ErrReasonRequired
	}
	if reason != "" && !contains(draftReasonCodes, reason) {
		return ErrInvalidReason
	}
	allowed := map[DraftAction][]string{
		DraftInclude:                        {"MANAGER_SCOPE_DECISION", "SIMULATION_SOURCE_GAP_OVERRIDE"},
		DraftExclude:                        {"MANAGER_SCOPE_DECISION"},
		DraftDefer:                          {"MANAGER_SCOPE_DECISION"},
		DraftReclassifyMainDomain:           {"CLASSIFICATION_EXPERT_REVIEW"},
		DraftAddTopic:                       {"CLASSIFICATION_EXPERT_REVIEW"},
		DraftRemoveTopic:                    {"CLASSIFICATION_EXPERT_REVIEW"},
		DraftResolveClassificationProposals: {"MANAGER_EXACT_RESOLUTION"},
		DraftAddCandidate:                   {"SYNTHETIC_CANDIDATE_ADDED"},
		DraftRewordCandidate:                {"SYNTHETIC_CANDIDATE_REWORDED"},
		DraftMarkReady:                      {"MANAGER_SCOPE_DECISION", "SIMULATION_SOURCE_GAP_OVERRIDE"},
	}
	if choices, exists := allowed[action]; exists && reason != "" && !contains(choices, reason) {
		return ErrInvalidReason
	}
	return nil
}

func validateReadinessDraft(draft Draft, reason string) error {
	promptDigest := ""
	for _, item := range draft.Items {
		if item.candidatePassRecord == nil {
			continue
		}
		if promptDigest == "" {
			promptDigest = item.candidatePassRecord.PromptDigest
		} else if promptDigest != item.candidatePassRecord.PromptDigest {
			return ErrDraftNotReady
		}
	}
	if draft.GenerationState != GenerationActive || draft.ClassificationRunState != ClassificationRunSealed || draft.BaseQuestionCount != FrozenBaseQuestionCount || draft.ClassificationItemCount != FrozenBaseQuestionCount || draft.PackageVersion != FrozenPackageVersion || draft.PackageJSONSHA256 != FrozenPackageJSONSHA256 || draft.TaxonomyVersion != FrozenTaxonomy().Version || draft.TaxonomyDigest != FrozenTaxonomy().Digest || !classificationRunIDPattern.MatchString(draft.ClassificationRunID) || !validDigest(draft.ClassificationRunDigest) || !validDigest(draft.AggregateDigest) || !validDigest(promptDigest) || !reflect.DeepEqual(draft.FixedInputDigests, FrozenFixedInputDigests()) || draft.ClassificationInputDigest != ComputeRunInputDigestForPrompt(draft.FixedInputDigests, promptDigest) {
		return ErrDraftNotReady
	}
	current := currentDraftItems(draft)
	if len(current) < FrozenBaseQuestionCount {
		return ErrDraftNotReady
	}
	for _, item := range current {
		if item.ReviewState == ReviewPendingManager || item.Disposition == nil {
			return ErrDraftNotReady
		}
		if err := ValidateProjection(FrozenTaxonomy(), item.CurrentProjection); err != nil {
			return ErrDraftNotReady
		}
		if item.SourceMappingState != SourceMappingRequired || item.SourceAuthorityState != SourceAuthorityNotAttested || item.RiskClassificationState != RiskExpertReviewRequired {
			return ErrDraftNotReady
		}
		if item.QuestionSourceProposalGap {
			if item.RecommendationState != RecommendationBlockedSourceGap || ((item.Origin != DraftItemOriginSealedBase || item.ProposalResolution != nil) && item.DraftAgreementConfidence != nil) {
				return ErrDraftNotReady
			}
			if *item.Disposition == DispositionInclude && reason != "SIMULATION_SOURCE_GAP_OVERRIDE" {
				return ErrDraftNotReady
			}
		}
		if item.QuestionRef.Workspace != nil && (item.ReviewState != ReviewManagerDisposed || *item.Disposition != DispositionInclude) {
			return ErrDraftNotReady
		}
		if item.ReviewState == ReviewAutoPreselected {
			if item.QuestionRef.Base == nil || item.QuestionSourceProposalGap || item.RecommendationState != RecommendationAutoProposed || item.DraftAgreementConfidence == nil || *item.DraftAgreementConfidence != ConfidenceHigh || *item.Disposition != DispositionInclude {
				return ErrDraftNotReady
			}
		}
	}
	return nil
}

func demoteSemanticEdit(item *DraftItem) {
	item.DraftAgreementConfidence = nil
	item.ReviewState = ReviewPendingManager
	item.Disposition = nil
	if item.QuestionSourceProposalGap {
		item.RecommendationState = RecommendationBlockedSourceGap
	} else {
		item.RecommendationState = RecommendationManagerReview
	}
}

func setExactProposalResolution(item *DraftItem) {
	projection := cloneJSON(item.CurrentProjection)
	item.ProposalResolution = &ProposalResolution{Mode: ResolutionSetExact, ProposalProjection: &projection}
}

func readinessEventIDExists(draft Draft, eventID string) bool {
	for _, event := range draft.ReadinessEvents {
		if event.ReadinessEventID == eventID {
			return true
		}
	}
	return false
}

func markSourceGap(item *DraftItem) {
	item.QuestionSourceProposalGap = true
	item.SourceMappingState = SourceMappingRequired
	item.SourceAuthorityState = SourceAuthorityNotAttested
	item.RiskClassificationState = RiskExpertReviewRequired
	demoteSemanticEdit(item)
}

func newWorkspaceDraftItem(reference WorkspaceQuestionRef, projection ProposalProjection) DraftItem {
	projectionCopy := cloneJSON(projection)
	item := DraftItem{
		QuestionRef: WorkspaceQuestionReference(reference), Origin: DraftItemOriginAuthored,
		CurrentProjection: cloneJSON(projection), ProposalResolution: &ProposalResolution{Mode: ResolutionSetExact, ProposalProjection: &projectionCopy},
		RecommendationState: RecommendationBlockedSourceGap, ReviewState: ReviewPendingManager, Current: true,
	}
	markSourceGap(&item)
	return item
}

func parentKeyForQuestion(reference QuestionRef) *ParentQuestionKey {
	if reference.Base != nil {
		base := *reference.Base
		return &ParentQuestionKey{Base: &base}
	}
	workspace := reference.Workspace
	return &ParentQuestionKey{
		WorkspaceGenerationID: workspace.GenerationID,
		WorkspaceRootID:       workspace.RootID,
		WorkspaceVersionID:    workspace.VersionID,
		WorkspaceProposalID:   workspace.ProposalID,
		WorkspaceRootSequence: workspace.RootSequence,
		WorkspaceBodyDigest:   workspace.BodyDigest,
	}
}

func resolveProjection(taxonomy Taxonomy, draft Draft, item DraftItem, command DraftCommand) (ProposalProjection, *ProposalResolution, error) {
	if command.MainDomainCode != "" || command.TopicCode != "" || command.WorkspaceBody != "" || command.WorkspaceBodyDigest != "" || command.ReadinessEventID != "" || command.ProviderScopeProfileDigest != "" {
		return ProposalProjection{}, nil, ErrInvalidResolution
	}
	var source ProposalProjection
	resolution := &ProposalResolution{Mode: command.ResolutionMode}
	switch command.ResolutionMode {
	case ResolutionCandidate:
		record, ok := sealedPassForDraft(draft, item, PassCandidate)
		if item.Origin != DraftItemOriginSealedBase || command.ExactProjection != nil || !ok {
			return ProposalProjection{}, nil, ErrInvalidResolution
		}
		source = record.ProposalProjection
	case ResolutionChallenge:
		record, ok := sealedPassForDraft(draft, item, PassChallenge)
		if item.Origin != DraftItemOriginSealedBase || command.ExactProjection != nil || !ok {
			return ProposalProjection{}, nil, ErrInvalidResolution
		}
		source = record.ProposalProjection
	case ResolutionSetExact:
		if command.ExactProjection == nil {
			return ProposalProjection{}, nil, ErrInvalidResolution
		}
		source = *command.ExactProjection
	default:
		return ProposalProjection{}, nil, ErrInvalidResolution
	}
	normalized, err := normalizeProjection(taxonomy, source)
	if err != nil {
		return ProposalProjection{}, nil, err
	}
	if command.ResolutionMode == ResolutionSetExact {
		copy := cloneJSON(normalized)
		resolution.ProposalProjection = &copy
	}
	return normalized, resolution, nil
}

func validStoredPass(record *PassProposalRecord, expectedDigest string, reference QuestionRef, role PassRole) bool {
	return validStoredPassShape(record, expectedDigest, reference, role) && record.PassResultDigest == ComputePassResultDigest(*record)
}

func validStoredPassShape(record *PassProposalRecord, expectedDigest string, reference QuestionRef, role PassRole) bool {
	return record != nil && reference.Base != nil && record.PassRole == role && reflect.DeepEqual(record.Identity, *reference.Base) && record.PassResultDigest == expectedDigest
}

func sealedPassForDraft(draft Draft, item DraftItem, role PassRole) (PassProposalRecord, bool) {
	if item.QuestionRef.Base == nil {
		return PassProposalRecord{}, false
	}
	key := item.QuestionRef.Base.Key()
	var sealed PassProposalRecord
	var copied *PassProposalRecord
	var expectedDigest string
	if role == PassCandidate {
		sealed, _ = draft.sealedCandidateRecords[key]
		copied = item.candidatePassRecord
		expectedDigest = item.candidatePassResultDigest
	} else if role == PassChallenge {
		sealed, _ = draft.sealedChallengeRecords[key]
		copied = item.challengePassRecord
		expectedDigest = item.challengePassResultDigest
	} else {
		return PassProposalRecord{}, false
	}
	valid := validStoredPass
	if draft.sealedPassGraphValidated {
		valid = validStoredPassShape
	}
	if !valid(&sealed, sealed.PassResultDigest, item.QuestionRef, role) || !valid(copied, expectedDigest, item.QuestionRef, role) || expectedDigest != sealed.PassResultDigest || !reflect.DeepEqual(*copied, sealed) {
		return PassProposalRecord{}, false
	}
	return clonePassProposalRecord(sealed), true
}

func sealedPassFailureCategory(draft Draft, item DraftItem, role PassRole) string {
	if item.QuestionRef.Base == nil {
		return "reference-missing"
	}
	key := item.QuestionRef.Base.Key()
	var sealed PassProposalRecord
	var copied *PassProposalRecord
	var expectedDigest string
	if role == PassCandidate {
		var found bool
		sealed, found = draft.sealedCandidateRecords[key]
		if !found {
			return "sealed-record-missing"
		}
		copied = item.candidatePassRecord
		expectedDigest = item.candidatePassResultDigest
	} else if role == PassChallenge {
		var found bool
		sealed, found = draft.sealedChallengeRecords[key]
		if !found {
			return "sealed-record-missing"
		}
		copied = item.challengePassRecord
		expectedDigest = item.challengePassResultDigest
	} else {
		return "role-invalid"
	}
	if copied == nil {
		return "copied-pass-missing"
	}
	if sealed.PassRole != role {
		return "sealed-role-mismatch"
	}
	if !reflect.DeepEqual(sealed.Identity, *item.QuestionRef.Base) {
		return "sealed-identity-mismatch"
	}
	if sealed.PassResultDigest != ComputePassResultDigest(sealed) {
		return "sealed-digest-mismatch"
	}
	if copied.PassRole != role {
		return "copied-role-mismatch"
	}
	if copied.ClassificationRunID != sealed.ClassificationRunID {
		return "copied-run-mismatch"
	}
	if !reflect.DeepEqual(copied.Identity, *item.QuestionRef.Base) {
		return "copied-identity-mismatch"
	}
	if copied.PassResultDigest != ComputePassResultDigest(*copied) {
		return "copied-digest-mismatch"
	}
	if expectedDigest != sealed.PassResultDigest {
		return "pass-digest-mismatch"
	}
	if !reflect.DeepEqual(*copied, sealed) {
		return "copied-pass-diff"
	}
	return ""
}

func removeString(values []string, target string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != target {
			result = append(result, value)
		}
	}
	return result
}

func validateDraftQuestionGraph(draft Draft) error {
	return validateDraftQuestionGraphWithTaxonomy(draft, FrozenTaxonomy())
}

func validateDraftQuestionGraphWithTaxonomy(draft Draft, taxonomy Taxonomy) error {
	refs := make([]WorkspaceQuestionRef, 0)
	currentBySequence := make(map[int]struct{})
	questionKeys := make(map[string]struct{})
	baseItems := make(map[string]DraftItem)
	children := make(map[string]int)
	baseChildren := make(map[string]int)
	addedRoots := 0
	for _, item := range draft.Items {
		if err := ValidateQuestionRef(item.QuestionRef); err != nil {
			return err
		}
		if _, duplicate := questionKeys[item.QuestionRef.Key()]; duplicate {
			return ErrWorkspaceIdentityAlias
		}
		questionKeys[item.QuestionRef.Key()] = struct{}{}
		if err := ValidateProjection(taxonomy, item.CurrentProjection); err != nil {
			return err
		}
		if err := validateDraftItemStateWithTaxonomy(draft, item, taxonomy); err != nil {
			return err
		}
		if item.Current {
			if _, exists := currentBySequence[item.QuestionRef.RootSequence]; exists {
				return ErrNonCurrentQuestion
			}
			currentBySequence[item.QuestionRef.RootSequence] = struct{}{}
		}
		if item.QuestionRef.Workspace != nil {
			expectedOrigin := DraftItemOriginReworded
			if item.QuestionRef.Workspace.ParentQuestionKey == nil {
				expectedOrigin = DraftItemOriginAuthored
			}
			if item.Origin != expectedOrigin || item.SealedAgreementConfidence != nil || item.candidatePassRecord != nil || item.challengePassRecord != nil || item.candidatePassResultDigest != "" || item.challengePassResultDigest != "" {
				return ErrPassBijection
			}
			refs = append(refs, *item.QuestionRef.Workspace)
			parent := item.QuestionRef.Workspace.ParentQuestionKey
			if parent == nil {
				addedRoots++
			}
			if parent != nil && parent.Base != nil {
				baseChildren[parent.Base.Key()]++
			}
			if parent != nil && parent.Base == nil {
				children[parent.WorkspaceVersionID]++
			}
		} else if item.QuestionRef.Base != nil {
			_, candidateValid := sealedPassForDraft(draft, item, PassCandidate)
			_, challengeValid := sealedPassForDraft(draft, item, PassChallenge)
			if item.Origin != DraftItemOriginSealedBase || item.SealedAgreementConfidence == nil || !candidateValid || !challengeValid {
				return passBijectionError("draft base pass graph candidate=" + sealedPassFailureCategory(draft, item, PassCandidate) + " challenge=" + sealedPassFailureCategory(draft, item, PassChallenge))
			}
			baseItems[item.QuestionRef.Base.Key()] = item
		}
	}
	if len(baseItems) != draft.BaseQuestionCount {
		return passBijectionError("draft base item count")
	}
	if err := ValidateWorkspaceQuestionRefs(refs, draft.GenerationID); err != nil {
		return err
	}
	for key, baseItem := range baseItems {
		if baseChildren[key] > 1 {
			return ErrWorkspaceIdentityAlias
		}
		shouldBeCurrent := baseChildren[key] == 0
		if baseItem.Current != shouldBeCurrent {
			return ErrNonCurrentQuestion
		}
	}
	for _, item := range draft.Items {
		if item.QuestionRef.Workspace == nil {
			continue
		}
		if parent := item.QuestionRef.Workspace.ParentQuestionKey; parent != nil && parent.Base != nil {
			baseItem, exists := baseItems[parent.Base.Key()]
			if !exists || item.QuestionRef.RootSequence != baseItem.QuestionRef.RootSequence {
				return ErrWorkspaceIdentityAlias
			}
		}
		shouldBeCurrent := children[item.QuestionRef.Workspace.VersionID] == 0
		if item.Current != shouldBeCurrent {
			return ErrNonCurrentQuestion
		}
	}
	if len(currentBySequence) != draft.BaseQuestionCount+addedRoots {
		return ErrNonCurrentQuestion
	}
	return nil
}

func validateDraftItemState(draft Draft, item DraftItem) error {
	return validateDraftItemStateWithTaxonomy(draft, item, FrozenTaxonomy())
}

func validateDraftItemStateWithTaxonomy(draft Draft, item DraftItem, taxonomy Taxonomy) error {
	if !contains([]string{RecommendationAutoProposed, RecommendationManagerReview, RecommendationBlockedSourceGap}, item.RecommendationState) {
		return ErrInvalidResolution
	}
	if item.QuestionSourceProposalGap != (item.RecommendationState == RecommendationBlockedSourceGap) {
		return ErrInvalidResolution
	}
	switch item.ReviewState {
	case ReviewPendingManager:
		if item.Disposition != nil {
			return ErrInvalidResolution
		}
	case ReviewAutoPreselected:
		if item.Origin != DraftItemOriginSealedBase || item.QuestionSourceProposalGap || item.ProposalResolution != nil || item.DraftAgreementConfidence == nil || *item.DraftAgreementConfidence != ConfidenceHigh || item.RecommendationState != RecommendationAutoProposed || item.Disposition == nil || *item.Disposition != DispositionInclude {
			return ErrInvalidResolution
		}
	case ReviewManagerDisposed:
		if item.Disposition == nil || !contains([]string{DispositionInclude, DispositionExclude, DispositionDefer}, *item.Disposition) {
			return ErrInvalidResolution
		}
	default:
		return ErrInvalidResolution
	}
	if item.Origin == DraftItemOriginAuthored || item.Origin == DraftItemOriginReworded {
		if item.DraftAgreementConfidence != nil || !item.QuestionSourceProposalGap {
			return ErrInvalidResolution
		}
	}
	if item.Origin == DraftItemOriginSealedBase && (item.QuestionRef.Base == nil || item.sealedBaseRootSequence < 1 || item.QuestionRef.RootSequence != item.sealedBaseRootSequence) {
		return ErrInvalidResolution
	}
	if item.Origin == DraftItemOriginSealedBase && (item.SourceMappingState != item.sealedGovernance.SourceMappingState || item.SourceAuthorityState != item.sealedGovernance.SourceAuthorityState || item.RiskClassificationState != item.sealedGovernance.RiskClassificationState || item.DecisionState != item.sealedGovernance.DecisionState || item.ExtractionState != item.sealedGovernance.ExtractionState || item.QuestionSourceProposalGap != item.sealedGovernance.QuestionSourceProposalGap || item.ExternalApplicabilityUnresolved != item.sealedGovernance.ExternalApplicabilityUnresolved) {
		return ErrInvalidResolution
	}
	if item.ProposalResolution == nil {
		if item.Origin == DraftItemOriginAuthored {
			return ErrInvalidResolution
		}
		if item.Origin == DraftItemOriginSealedBase {
			candidate, candidateOK := sealedPassForDraft(draft, item, PassCandidate)
			challenge, challengeOK := sealedPassForDraft(draft, item, PassChallenge)
			if !candidateOK || !challengeOK {
				return passBijectionError("draft unresolved base pass graph candidate=" + sealedPassFailureCategory(draft, item, PassCandidate) + " challenge=" + sealedPassFailureCategory(draft, item, PassChallenge))
			}
			if item.SealedAgreementConfidence == nil || item.DraftAgreementConfidence == nil || *item.DraftAgreementConfidence != *item.SealedAgreementConfidence || !projectionFieldSetsEqual(taxonomy, item.CurrentProjection, candidate.ProposalProjection) || item.QuestionRef.RootSequence != item.sealedBaseRootSequence {
				return ErrInvalidResolution
			}
			outcome := DeriveOutcome(taxonomy, candidate.ProposalProjection, challenge.ProposalProjection, candidate.ConfidenceEvidence, challenge.ConfidenceEvidence, item.QuestionSourceProposalGap, item.ExternalApplicabilityUnresolved)
			if *item.SealedAgreementConfidence != outcome.AgreementConfidence || item.RecommendationState != outcome.RecommendationState || item.RecommendationState != item.sealedRecommendationState {
				return ErrInvalidResolution
			}
		}
		return nil
	}
	resolution := item.ProposalResolution
	switch resolution.Mode {
	case ResolutionCandidate:
		record, ok := sealedPassForDraft(draft, item, PassCandidate)
		if resolution.ProposalProjection != nil || !ok || !projectionFieldSetsEqual(taxonomy, item.CurrentProjection, record.ProposalProjection) {
			return ErrInvalidResolution
		}
	case ResolutionChallenge:
		record, ok := sealedPassForDraft(draft, item, PassChallenge)
		if resolution.ProposalProjection != nil || !ok || !projectionFieldSetsEqual(taxonomy, item.CurrentProjection, record.ProposalProjection) {
			return ErrInvalidResolution
		}
	case ResolutionSetExact:
		if resolution.ProposalProjection == nil || !projectionFieldSetsEqual(taxonomy, item.CurrentProjection, *resolution.ProposalProjection) {
			return ErrInvalidResolution
		}
	default:
		return ErrInvalidResolution
	}
	if item.DraftAgreementConfidence != nil {
		return ErrInvalidResolution
	}
	return nil
}

func ProjectionFieldSetsEqual(left, right ProposalProjection) bool {
	return projectionFieldSetsEqual(FrozenTaxonomy(), left, right)
}

func projectionFieldSetsEqual(taxonomy Taxonomy, left, right ProposalProjection) bool {
	for _, field := range taxonomy.ProposalFields {
		if !projectionFieldEqual(taxonomy, left, right, field) {
			return false
		}
	}
	return true
}

func QuestionSnapshot(draft Draft) ([]QuestionRef, error) {
	if err := validateDraftQuestionGraph(draft); err != nil {
		return nil, err
	}
	items := currentDraftItems(draft)
	result := make([]QuestionRef, len(items))
	for index, item := range items {
		result[index] = cloneQuestionRef(item.QuestionRef)
	}
	return result, nil
}

func PreviewDraftBatch(draft Draft, filter DraftBatchFilter, action DraftBatchAction, now time.Time, allocator IDAllocator) (DraftBatchPreview, error) {
	if allocator == nil {
		return DraftBatchPreview{}, ErrPreviewMismatch
	}
	if !draftAcceptsCommands(draft) {
		return DraftBatchPreview{}, ErrDraftNotReady
	}
	if !contains([]string{string(DraftRetain), string(DraftReclassifyMainDomain), string(DraftAddTopic), string(DraftRemoveTopic)}, string(action.Action)) {
		return DraftBatchPreview{}, fmt.Errorf("%w: batch action", ErrUnknownCode)
	}
	if ComputeDraftContentDigest(draft) != draft.ContentDigest {
		return DraftBatchPreview{}, ErrDraftConflict
	}
	if err := validateDraftQuestionGraph(draft); err != nil {
		return DraftBatchPreview{}, err
	}
	if err := validateBatchAction(FrozenTaxonomy(), action); err != nil {
		return DraftBatchPreview{}, err
	}
	normalizedDomains, err := normalizeStrings(filter.MainDomainCodes, FrozenTaxonomy().MainDomainCodes, "mainDomainCodes", false)
	if err != nil {
		return DraftBatchPreview{}, err
	}
	filter.MainDomainCodes = normalizedDomains
	items := batchFilteredItems(draft, filter)
	if len(items) > 500 {
		return DraftBatchPreview{}, ErrBatchLimit
	}
	identities := make([]string, len(items))
	for index, item := range items {
		identities[index] = digestValue("AGA-DRAFT-QUESTION-IDENTITY-V1", item.QuestionRef)
	}
	sort.Strings(identities)
	preview := DraftBatchPreview{
		PreviewID:    allocator.NextPreviewID(),
		GenerationID: draft.GenerationID, ClassificationRunID: draft.ClassificationRunID,
		DraftRevision: draft.Revision, DraftContentDigest: draft.ContentDigest,
		ActionDigest:           digestValue("AGA-DRAFT-BATCH-ACTION-V1", action),
		FilterDigest:           digestValue("AGA-DRAFT-BATCH-FILTER-V1", filter),
		OrderedIdentityDigests: identities,
		OrderedIdentityDigest:  digestValue("AGA-DRAFT-BATCH-IDENTITY-SET-V1", identities),
		Count:                  len(items), ExpiresAt: now.UTC().Add(5 * time.Minute),
	}
	if !serverIDPattern.MatchString(preview.PreviewID) {
		return DraftBatchPreview{}, ErrPreviewMismatch
	}
	preview.PreviewDigest = digestExcludingJSONFields("AGA-DRAFT-BATCH-PREVIEW-V1", preview, "previewDigest")
	return preview, nil
}

func batchFilteredItems(draft Draft, filter DraftBatchFilter) []DraftItem {
	items := currentDraftItems(draft)
	if len(filter.MainDomainCodes) == 0 {
		return items
	}
	result := make([]DraftItem, 0, len(items))
	for _, item := range items {
		if contains(filter.MainDomainCodes, item.CurrentProjection.MainDomainCode) {
			result = append(result, item)
		}
	}
	return result
}

func ExecuteDraftBatch(draft Draft, preview DraftBatchPreview, execution DraftBatchExecution, filter DraftBatchFilter, action DraftBatchAction, now time.Time) (Draft, error) {
	if !draftAcceptsCommands(draft) {
		return Draft{}, ErrDraftNotReady
	}
	if execution.OperationID == "" || execution.IdempotencyKey == "" || execution.ExpectedGenerationID != draft.GenerationID || execution.PreviewID != preview.PreviewID || execution.PreviewDigest != preview.PreviewDigest || !serverIDPattern.MatchString(preview.PreviewID) || !validDigest(preview.PreviewDigest) {
		return Draft{}, ErrPreviewMismatch
	}
	if execution.ExpectedDraftRevision != draft.Revision || execution.ExpectedDraftContentDigest != draft.ContentDigest || preview.DraftRevision != draft.Revision || preview.DraftContentDigest != draft.ContentDigest || ComputeDraftContentDigest(draft) != draft.ContentDigest {
		return Draft{}, ErrDraftConflict
	}
	if !now.Before(preview.ExpiresAt) {
		return Draft{}, ErrPreviewExpired
	}
	if preview.GenerationID != draft.GenerationID || preview.ClassificationRunID != draft.ClassificationRunID || preview.ActionDigest != digestValue("AGA-DRAFT-BATCH-ACTION-V1", action) {
		return Draft{}, ErrPreviewMismatch
	}
	if err := validateBatchAction(FrozenTaxonomy(), action); err != nil {
		return Draft{}, err
	}
	if preview.PreviewDigest != digestExcludingJSONFields("AGA-DRAFT-BATCH-PREVIEW-V1", preview, "previewDigest") {
		return Draft{}, ErrPreviewMismatch
	}
	normalizedDomains, err := normalizeStrings(filter.MainDomainCodes, FrozenTaxonomy().MainDomainCodes, "mainDomainCodes", false)
	if err != nil {
		return Draft{}, err
	}
	filter.MainDomainCodes = normalizedDomains
	if preview.FilterDigest != digestValue("AGA-DRAFT-BATCH-FILTER-V1", filter) {
		return Draft{}, ErrPreviewMismatch
	}
	items := batchFilteredItems(draft, filter)
	if len(items) != preview.Count || len(items) > 500 {
		return Draft{}, ErrPreviewMismatch
	}
	identities := make([]string, len(items))
	for index, item := range items {
		identities[index] = digestValue("AGA-DRAFT-QUESTION-IDENTITY-V1", item.QuestionRef)
	}
	sort.Strings(identities)
	if !reflect.DeepEqual(identities, preview.OrderedIdentityDigests) || digestValue("AGA-DRAFT-BATCH-IDENTITY-SET-V1", identities) != preview.OrderedIdentityDigest {
		return Draft{}, ErrPreviewMismatch
	}
	next := cloneDraft(draft)
	if next.State == DraftReadyForDemoSimulation {
		next.State = DraftWorking
		next.CurrentReadinessEventID = ""
	}
	for _, item := range items {
		target := item
		switch action.Action {
		case DraftRetain:
		case DraftReclassifyMainDomain:
			if err := validateCode(FrozenTaxonomy().MainDomainCodes, action.MainDomainCode, "mainDomainCode"); err != nil {
				return Draft{}, err
			}
			target.CurrentProjection.MainDomainCode = action.MainDomainCode
			setExactProposalResolution(&target)
			demoteSemanticEdit(&target)
		case DraftAddTopic:
			if err := validateCode(FrozenTaxonomy().TopicCodes, action.TopicCode, "topicCode"); err != nil {
				return Draft{}, err
			}
			if contains(target.CurrentProjection.TopicCodes, action.TopicCode) {
				return Draft{}, ErrDuplicateProposalValue
			}
			target.CurrentProjection.TopicCodes = append(target.CurrentProjection.TopicCodes, action.TopicCode)
			sort.Strings(target.CurrentProjection.TopicCodes)
			setExactProposalResolution(&target)
			demoteSemanticEdit(&target)
		case DraftRemoveTopic:
			if !contains(target.CurrentProjection.TopicCodes, action.TopicCode) {
				return Draft{}, ErrInvalidResolution
			}
			target.CurrentProjection.TopicCodes = removeString(target.CurrentProjection.TopicCodes, action.TopicCode)
			target.CurrentProjection, _ = normalizeProjection(FrozenTaxonomy(), target.CurrentProjection)
			setExactProposalResolution(&target)
			demoteSemanticEdit(&target)
		default:
			return Draft{}, fmt.Errorf("%w: batch action", ErrUnknownCode)
		}
		replaceCurrentDraftItem(&next, target)
	}
	if err := validateDraftQuestionGraph(next); err != nil {
		return Draft{}, err
	}
	next.Revision++
	next.ContentDigest = ComputeDraftContentDigest(next)
	return next, nil
}

func draftAcceptsCommands(draft Draft) bool {
	return draft.GenerationState == GenerationActive && draft.ClassificationRunState == ClassificationRunSealed && (draft.State == DraftWorking || draft.State == DraftReadyForDemoSimulation)
}

func validateBatchAction(taxonomy Taxonomy, action DraftBatchAction) error {
	if err := validateDraftReason(action.Action, action.ReasonCode); err != nil {
		return err
	}
	switch action.Action {
	case DraftRetain:
		if action.MainDomainCode != "" || action.TopicCode != "" {
			return ErrInvalidResolution
		}
	case DraftReclassifyMainDomain:
		if action.TopicCode != "" {
			return ErrInvalidResolution
		}
		return validateCode(taxonomy.MainDomainCodes, action.MainDomainCode, "mainDomainCode")
	case DraftAddTopic, DraftRemoveTopic:
		if action.MainDomainCode != "" {
			return ErrInvalidResolution
		}
		return validateCode(taxonomy.TopicCodes, action.TopicCode, "topicCode")
	default:
		return ErrInvalidResolution
	}
	return nil
}
