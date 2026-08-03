package agaapplicability

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
)

func TestClassifySealedBaseContract(t *testing.T) {
	_, result := exactClassificationFixture(t)
	if len(result.Items) != FrozenBaseQuestionCount {
		t.Fatalf("sealed item count = %d", len(result.Items))
	}
	seenTextDigests := make(map[string]string)
	foundRepeatedDigest := false
	for _, item := range result.Items {
		if previousKey, exists := seenTextDigests[item.Identity.TextDigest]; exists && previousKey != item.Identity.Key() {
			foundRepeatedDigest = true
			break
		}
		seenTextDigests[item.Identity.TextDigest] = item.Identity.Key()
	}
	if !foundRepeatedDigest {
		t.Fatal("accepted repeated text digest on distinct complete identities was not preserved")
	}
	for _, item := range result.Items {
		if item.Projection.MainDomainCode != "QUALITY_MANAGEMENT" || item.AgreementConfidence != ConfidenceHigh {
			t.Fatal("sealed classification invariant failed")
		}
		expectedRecommendation := RecommendationAutoProposed
		if item.QuestionSourceProposalGap {
			expectedRecommendation = RecommendationBlockedSourceGap
		} else if item.ExternalApplicabilityUnresolved {
			expectedRecommendation = RecommendationManagerReview
		}
		if item.RecommendationState != expectedRecommendation || item.InputDigest != ComputeRunInputDigest(FrozenFixedInputDigests()) {
			t.Fatal("sealed outcome/input digest invariant failed")
		}
		if item.SourceMappingState != SourceMappingRequired || item.SourceAuthorityState != SourceAuthorityNotAttested || item.RiskClassificationState != RiskExpertReviewRequired || item.DecisionState != DecisionNotSupplied {
			t.Fatal("governance invariant failed")
		}
		if item.PassOneResultDigest == "" || item.PassTwoResultDigest == "" || item.ItemSemanticDigest == "" {
			t.Fatal("digest graph incomplete")
		}
	}
}

func TestPassInputsAndSealsBindRoleRunModelAndAcceptedInventory(t *testing.T) {
	input, _ := exactClassificationFixture(t)
	if input.CandidateRecords[0].InputDigest == input.ChallengeRecords[0].InputDigest {
		t.Fatal("candidate and challenge pass inputs must differ")
	}
	if _, err := ReconcileClassification(input); err != nil {
		t.Fatalf("sealed pass input fixture error = %v", err)
	}
	bad := input
	bad.PassOneSealDigest = digestHex("format-only-seal")
	if _, err := ReconcileClassification(bad); !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("format-only pass seal error = %v", err)
	}
	bad = input
	bad.OrderedBaseIdentities = slices.Clone(input.OrderedBaseIdentities)
	bad.OrderedBaseIdentities[0], bad.OrderedBaseIdentities[1] = bad.OrderedBaseIdentities[1], bad.OrderedBaseIdentities[0]
	if _, err := ReconcileClassification(bad); !errors.Is(err, ErrPassBijection) {
		t.Fatalf("accepted inventory order error = %v", err)
	}
}

func TestPassInputAndReceiptUseFrozenClosedPreimages(t *testing.T) {
	input, _ := exactClassificationFixture(t)
	passInputs := input.PassInputsByRole[PassCandidate]
	if len(passInputs) != 25 {
		t.Fatalf("candidate pass input batch count = %d", len(passInputs))
	}
	first := passInputs[0]
	if first.SchemaVersion != "aga-hybrid-classification-pass-input/v1" || first.Purpose != "ROW_CLASSIFICATION_PRIVATE_INPUT" || len(first.Items) == 0 {
		t.Fatal("pass input frozen envelope invariant failed")
	}
	if got := ComputeClassificationPassInputDigest(first); got != input.CandidateRecords[0].InputDigest {
		t.Fatal("pass input digest was not reconstructed")
	}
	mutated := cloneJSON(first)
	mutated.Items[0].QuestionBody = "changed"
	if ComputeClassificationPassInputDigest(mutated) == input.CandidateRecords[0].InputDigest {
		t.Fatal("pass input body did not affect digest")
	}
	if encoded, err := json.Marshal(input.PassOneSealReceipt); err != nil || strings.Contains(string(encoded), "ClassificationRunID") {
		t.Fatal("pass seal receipt is not lower-camel JSON")
	}
}

func TestFrozen25BatchClassificationManifestRuntimePin(t *testing.T) {
	if FrozenBatchManifestDigest != "sha256:dee3a0101dcfdeaef9dbb8c3f53d7e4a99de9499eaa7d82a039eb6cac077c96b" {
		t.Fatal("classification runtime does not pin the accepted 25-batch manifest")
	}
	if !reflect.DeepEqual(frozenBatchItemCounts, []int{50, 49, 55, 54, 54, 53, 52, 54, 53, 52, 52, 57, 58, 58, 58, 61, 59, 53, 51, 52, 56, 52, 56, 57, 4}) {
		t.Fatal("classification runtime batch shape is not the accepted 25-batch shape")
	}
	input, _ := exactClassificationFixture(t)
	input.BatchManifestDigest = "sha256:000bb8c32076a74c9468c19e9d8b35901c83279e0635a0e84dc109801977873c"
	if _, err := ReconcileClassification(input); !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("discovery manifest runtime rejection = %v", err)
	}
}

func TestCandidateAndChallengeRequireRoleNeutralPrivateSnapshots(t *testing.T) {
	input, _ := exactClassificationFixture(t)
	candidate := input.PassInputsByRole[PassCandidate][0]
	challenge := cloneJSON(input.PassInputsByRole[PassChallenge][0])
	if ComputeRoleNeutralPassInputDigest(candidate) != ComputeRoleNeutralPassInputDigest(challenge) {
		t.Fatal("equivalent candidate/challenge private snapshots did not reconcile")
	}
	challenge.Items[0].QuestionBody = "private-snapshot-divergence"
	if ComputeRoleNeutralPassInputDigest(candidate) == ComputeRoleNeutralPassInputDigest(challenge) {
		t.Fatal("role-neutral private snapshot divergence was not observable")
	}
	input.PassInputsByRole = cloneJSON(input.PassInputsByRole)
	input.PassInputsByRole[PassChallenge][0] = challenge
	if _, err := ReconcileClassification(input); !errors.Is(err, ErrPrivateInputMismatch) {
		t.Fatalf("role-neutral private snapshot reconciliation error = %v", err)
	}
}

func TestClassificationDerivesEvidenceFactsFromPrivateInput(t *testing.T) {
	input, _ := exactClassificationFixture(t)
	input.EvidenceFactsByIdentity = cloneJSON(input.EvidenceFactsByIdentity)
	for key := range input.EvidenceFactsByIdentity {
		input.EvidenceFactsByIdentity[key] = EvidenceFacts{"QUESTION_BODY_DIGEST": {{Digest: digestHex("caller-controlled-evidence")}}}
		break
	}
	if _, err := ReconcileClassification(input); err != nil {
		t.Fatalf("caller-supplied evidence facts were trusted instead of derived = %v", err)
	}
}

func TestClassificationResultJSONRoundTripIsTextFreeAndDraftable(t *testing.T) {
	_, result := exactClassificationFixture(t)
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal classification result = %v", err)
	}
	if strings.Contains(string(encoded), "questionBody") || strings.Contains(string(encoded), "private question") {
		t.Fatal("classification result retained private body material")
	}
	var roundTrip ClassificationResult
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatalf("unmarshal classification result = %v", err)
	}
	if _, err := NewDraftFromClassification(roundTrip, "aga-ws-generation-roundtrip"); err != nil {
		t.Fatalf("text-free classification round trip is not draftable = %v", err)
	}
}

func TestSealedClassificationItemRejectsSemanticDigestTampering(t *testing.T) {
	_, result := exactClassificationFixture(t)
	item := result.Items[0]
	item.ItemSemanticDigest = digestHex("semantic-tamper")
	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal tampered sealed item = %v", err)
	}
	var decoded SealedClassificationItem
	if err := json.Unmarshal(encoded, &decoded); !errors.Is(err, ErrQuestionReferenceUnion) {
		t.Fatalf("semantic digest tamper error = %v", err)
	}
}

func TestClassificationDiagnosticsNeverEchoUntrustedPassRole(t *testing.T) {
	input, _ := exactClassificationFixture(t)
	record := input.CandidateRecords[0]
	record.PassRole = "private-untrusted-pass-role"
	err := validatePassRecord(FrozenTaxonomy(), record, PassCandidate, record.InputDigest, input, input.EvidenceFactsByIdentity[record.Identity.Key()])
	if err == nil || strings.Contains(err.Error(), string(record.PassRole)) {
		t.Fatal("classification diagnostic echoed an untrusted pass role")
	}
}

func TestSealedClassificationItemJSONAndSafeDigestBoundary(t *testing.T) {
	_, result := exactClassificationFixture(t)
	encoded, err := json.Marshal(result.Items[0])
	if err != nil {
		t.Fatalf("marshal sealed item error = %v", err)
	}
	var roundTrip SealedClassificationItem
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatalf("unmarshal sealed item error = %v", err)
	}
	if !reflect.DeepEqual(sealedClassificationItemObject(roundTrip), sealedClassificationItemObject(result.Items[0])) {
		t.Fatal("sealed item JSON round trip changed serialized semantic fields")
	}
	if err := json.Unmarshal(append(encoded[:len(encoded)-1], []byte(`,"extra":true}`)...), &roundTrip); !errors.Is(err, ErrQuestionReferenceUnion) {
		t.Fatalf("open sealed item JSON error = %v", err)
	}
	if _, err := DigestValue("AGA-TEST-V1", map[string]string{"x": string([]byte{0xff})}); err == nil {
		t.Fatal("invalid UTF-8 digest input accepted")
	}
	var ref QuestionRef
	if err := json.Unmarshal([]byte(`{"questionOrigin":"SEALED_BASE","packageVersion":"AGA_ALL_FORMS_SOURCE_RISK_DRAFT_V1","packageJsonSha256":"sha256:5ebcce2d70ee22fef4165b490cb6e4b276ad776f40dbaf12e5cea85c9da91b15","formCode":"FSS-AGA-FORM-002","proposalId":"\ud800","ordinal":1,"textDigest":"sha256:60e3f5c234faaa7ba1ba40636cbbbd116965c8df9066b186f272c6b0c4c61a4a"}`), &ref); !errors.Is(err, ErrQuestionReferenceUnion) {
		t.Fatalf("lone surrogate error = %v", err)
	}
	if err := json.Unmarshal([]byte(`{"questionOrigin":"SEALED_BASE","packageVersion":"AGA_ALL_FORMS_SOURCE_RISK_DRAFT_V1","packageJsonSha256":"sha256:5ebcce2d70ee22fef4165b490cb6e4b276ad776f40dbaf12e5cea85c9da91b15","formCode":"FSS-AGA-FORM-002","proposalId":"literal-\\ud800","ordinal":1,"textDigest":"sha256:60e3f5c234faaa7ba1ba40636cbbbd116965c8df9066b186f272c6b0c4c61a4a"}`), &ref); err != nil {
		t.Fatalf("escaped backslash text error = %v", err)
	}
}

func TestClassificationFatalErrorsAbortRun(t *testing.T) {
	if PassCandidate != "CANDIDATE" || PassChallenge != "CHALLENGE" {
		t.Fatalf("pass roles are not frozen values: %q/%q", PassCandidate, PassChallenge)
	}
	if validDigest("sha256:" + strings.Repeat("A", 64)) {
		t.Fatal("uppercase SHA-256 hex was accepted")
	}
	taxonomy := FrozenTaxonomy()
	base, _ := exactClassificationFixture(t)
	id := base.OrderedBaseIdentities[0]
	projection := completeProjection()
	record := base.CandidateRecords[0]

	short := base
	short.OrderedBaseIdentities = slices.Clone(base.OrderedBaseIdentities[:FrozenBaseQuestionCount-1])
	if _, err := ReconcileClassification(short); !errors.Is(err, ErrPassBijection) {
		t.Fatalf("short ordered identity set error = %v", err)
	}

	swapped := base
	swapped.CandidateRecords = slices.Clone(base.CandidateRecords)
	swapped.CandidateRecords[0], swapped.CandidateRecords[1] = swapped.CandidateRecords[1], swapped.CandidateRecords[0]
	if _, err := ReconcileClassification(swapped); !errors.Is(err, ErrPassBijection) {
		t.Fatalf("candidate record order mismatch error = %v", err)
	}

	wrongInputPin := base
	wrongInputPin.PassInputsByRole = cloneJSON(base.PassInputsByRole)
	wrongInputPin.PassInputsByRole[PassCandidate][0].Items[0].QuestionBody = "changed"
	if _, err := ReconcileClassification(wrongInputPin); !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("per-identity pass input pin error = %v", err)
	}

	duplicate := base
	duplicate.CandidateRecords = slices.Clone(base.CandidateRecords)
	duplicate.CandidateRecords[len(duplicate.CandidateRecords)-1] = record
	if _, err := ReconcileClassification(duplicate); !errors.Is(err, ErrDuplicateIdentity) {
		t.Fatalf("duplicate error = %v", err)
	} else {
		diagnostic := err.Error()
		for _, forbidden := range []string{id.PackageVersion, id.PackageJSONSHA256, id.FormCode, id.ProposalID, fmt.Sprint(id.Ordinal), id.TextDigest} {
			if strings.Contains(diagnostic, forbidden) {
				t.Fatalf("duplicate diagnostic leaked complete identity component")
			}
		}
	}

	unknown := projection
	unknown.MainDomainCode = "MODEL_INVENTED_DOMAIN"
	if _, err := NewPassProposalRecord(taxonomy, PassProposalInput{
		Identity: id, ClassificationRunID: base.ClassificationRunID, PassRole: PassCandidate,
		PassRunID: "aga-classification-pass-candidate-test", PromptDigest: base.PromptDigest,
		ModelDescriptorDigest: digestHex("model"), InputDigest: digestHex("batch"),
		Projection: unknown,
	}); !errors.Is(err, ErrUnknownCode) {
		t.Fatalf("unknown-code error = %v", err)
	}

	mismatch := projection
	mismatch.CanonicalTargetKind = "DEVICE"
	mismatch.TargetProfileCode = "MOVEMENT_AREA"
	if _, err := NewPassProposalRecord(taxonomy, PassProposalInput{
		Identity: id, ClassificationRunID: base.ClassificationRunID, PassRole: PassCandidate,
		PassRunID: "aga-classification-pass-candidate-test", PromptDigest: base.PromptDigest,
		ModelDescriptorDigest: digestHex("model"), InputDigest: digestHex("batch"),
		Projection: mismatch,
	}); !errors.Is(err, ErrTargetProfileMismatch) {
		t.Fatalf("target/profile error = %v", err)
	}
	if _, err := NewPassProposalRecord(taxonomy, PassProposalInput{
		Identity: id, ClassificationRunID: base.ClassificationRunID, PassRole: PassCandidate,
		PassRunID: "aga-classification-pass-candidate-test", PromptDigest: digestHex("wrong-prompt"),
		ModelDescriptorDigest: ComputeModelDescriptorDigest(testModelDescriptor()), InputDigest: digestHex("batch"),
		Projection: projection,
	}); !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("wrong prompt pin error = %v", err)
	}
	if _, err := NewPassProposalRecord(taxonomy, PassProposalInput{
		Identity: id, ClassificationRunID: base.ClassificationRunID, PassRole: PassCandidate,
		PassRunID: "aga-classification-pass-challenge-test", PromptDigest: base.PromptDigest,
		ModelDescriptorDigest: ComputeModelDescriptorDigest(testModelDescriptor()), InputDigest: digestHex("batch"),
		Projection: projection,
	}); !errors.Is(err, ErrPassBijection) {
		t.Fatalf("pass role/run mismatch error = %v", err)
	}
	multiProfileMismatch := completeProjection()
	multiProfileMismatch.InspectionProfileCodes = []string{"AERODROME_CERTIFICATION", "AERODROME_MANAGEMENT_SYSTEM"}
	multiProfileMismatch.InspectionTypeCodes = []string{"FOLLOW_UP"}
	if _, err := NewPassProposalRecord(taxonomy, PassProposalInput{
		Identity: id, ClassificationRunID: base.ClassificationRunID, PassRole: PassCandidate,
		PassRunID: "aga-classification-pass-candidate-test", PromptDigest: base.PromptDigest,
		ModelDescriptorDigest: ComputeModelDescriptorDigest(testModelDescriptor()), InputDigest: digestHex("batch"),
		Projection: multiProfileMismatch,
	}); !errors.Is(err, ErrTargetProfileMismatch) {
		t.Fatalf("inspection type not allowed by every selected profile error = %v", err)
	}

	badSelector := cloneJSON(record)
	badSelector.ConfidenceEvidence[0].InputFactSelector = "MODEL_PRIVATE_REASONING"
	badSelector.PassResultDigest = ComputePassResultDigest(badSelector)
	invalid := base
	invalid.CandidateRecords = slices.Clone(base.CandidateRecords)
	invalid.CandidateRecords[0] = badSelector
	if _, err := ReconcileClassification(invalid); !errors.Is(err, ErrUnknownInputFactSelector) {
		t.Fatalf("unknown selector error = %v", err)
	}

	badFact := cloneJSON(record)
	badFact.ConfidenceEvidence[0].InputFactValueDigest = digestHex("untrusted")
	badFact.PassResultDigest = ComputePassResultDigest(badFact)
	invalid.CandidateRecords = slices.Clone(base.CandidateRecords)
	invalid.CandidateRecords[0] = badFact
	if _, err := ReconcileClassification(invalid); !errors.Is(err, ErrEvidenceFactMismatch) {
		t.Fatalf("fact mismatch error = %v", err)
	}
	badPassRun := cloneJSON(record)
	badPassRun.PassRunID = "aga-classification-pass-challenge-test"
	badPassRun.PassResultDigest = ComputePassResultDigest(badPassRun)
	invalid.CandidateRecords = slices.Clone(base.CandidateRecords)
	invalid.CandidateRecords[0] = badPassRun
	if _, err := ReconcileClassification(invalid); !errors.Is(err, ErrPassBijection) {
		t.Fatalf("sealed pass role/run mismatch error = %v", err)
	}
	badGovernance := base
	badGovernance.GovernanceByIdentity = make(map[string]GovernanceState, len(base.GovernanceByIdentity))
	for key, governance := range base.GovernanceByIdentity {
		badGovernance.GovernanceByIdentity[key] = governance
	}
	value := badGovernance.GovernanceByIdentity[id.Key()]
	value.SourceMappingState = "APPROVED"
	badGovernance.GovernanceByIdentity[id.Key()] = value
	if _, err := ReconcileClassification(badGovernance); !errors.Is(err, ErrUnknownCode) {
		t.Fatalf("unknown governance state error = %v", err)
	}

	wrongAggregateTotals := base
	wrongAggregateTotals.GovernanceByIdentity = make(map[string]GovernanceState, len(base.GovernanceByIdentity))
	for key, governance := range base.GovernanceByIdentity {
		wrongAggregateTotals.GovernanceByIdentity[key] = cloneJSON(governance)
	}
	governance := wrongAggregateTotals.GovernanceByIdentity[id.Key()]
	governance.QuestionSourceProposalGap = false
	wrongAggregateTotals.GovernanceByIdentity[id.Key()] = governance
	if _, err := ReconcileClassification(wrongAggregateTotals); !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("frozen aggregate exception totals error = %v", err)
	}
}

func TestConfidenceRecommendationPrecedence(t *testing.T) {
	taxonomy := FrozenTaxonomy()
	projection := completeProjection()
	facts := trustedFacts()
	all := completeEvidence(t, taxonomy, projection, facts)
	coreKeys := CoreProposalBindingKeys(taxonomy, projection)
	coreOnly := make([]ConfidenceEvidence, 0, len(coreKeys))
	for _, evidence := range all {
		if coreKeys[evidence.ProposalField+"\x00"+evidence.ProposalValueDigest] {
			coreOnly = append(coreOnly, evidence)
		}
	}
	if got := DeriveOutcome(taxonomy, projection, projection, all, all, false, false); got != (ClassificationOutcome{AgreementConfidence: ConfidenceHigh, RecommendationState: RecommendationAutoProposed}) {
		t.Fatal("HIGH outcome invariant failed")
	}
	if got := DeriveOutcome(taxonomy, projection, projection, coreOnly, coreOnly, false, false); got != (ClassificationOutcome{AgreementConfidence: ConfidenceMedium, RecommendationState: RecommendationManagerReview}) {
		t.Fatal("MEDIUM outcome invariant failed")
	}
	if got := DeriveOutcome(taxonomy, projection, projection, nil, nil, false, false); got != (ClassificationOutcome{AgreementConfidence: ConfidenceLow, RecommendationState: RecommendationManagerReview}) {
		t.Fatal("LOW outcome invariant failed")
	}
	disagreement := projection
	disagreement.MainDomainCode = "SAFETY_MANAGEMENT_RISK_ASSESSMENT"
	if got := DeriveOutcome(taxonomy, projection, disagreement, all, all, false, false); got.AgreementConfidence != ConfidenceLow || got.RecommendationState != RecommendationManagerReview {
		t.Fatal("disagreement outcome invariant failed")
	}
	if got := DeriveOutcome(taxonomy, projection, projection, all, all, true, false); got.RecommendationState != RecommendationBlockedSourceGap || got.AgreementConfidence != ConfidenceHigh {
		t.Fatal("source-gap outcome invariant failed")
	}
	if got := DeriveOutcome(taxonomy, projection, projection, all, all, false, true); got.RecommendationState != RecommendationManagerReview || got.AgreementConfidence != ConfidenceHigh {
		t.Fatal("external unresolved outcome invariant failed")
	}

	withExternal := completeProjection()
	edge := externalEdge("ANSP", "COORDINATION", "ANSP_COORDINATION_REQUIRED")
	binding := ExternalInvolvementBinding(taxonomy, edge)
	edge.ConfidenceEvidence = []ConfidenceEvidence{{
		ProposalField: "externalInvolvements", ProposalValueDigest: binding.ValueDigest,
		RationaleCode: "EXTERNAL_INTERFACE_CUE", InputFactSelector: "VALIDATOR_SIGNAL_RULE_MATCH_DIGEST",
		InputFactValueDigest: digestHex("external-signal"), SignalRuleID: "EXPLICIT_EXTERNAL_ACTOR_V1",
	}}
	withExternal.ExternalInvolvements = []ExternalInvolvement{edge}
	withExternalEvidence := completeEvidence(t, taxonomy, withExternal, facts)
	if got := DeriveOutcome(taxonomy, withExternal, withExternal, withExternalEvidence, withExternalEvidence, false, false); got.AgreementConfidence != ConfidenceMedium {
		t.Fatal("external signal confidence invariant failed")
	}
	direct := edge.ConfidenceEvidence[0]
	direct.InputFactSelector = "QUESTION_BODY_DIGEST"
	direct.InputFactValueDigest = facts["QUESTION_BODY_DIGEST"][0].Digest
	direct.SignalRuleID = ""
	withExternal.ExternalInvolvements[0].ConfidenceEvidence = append(withExternal.ExternalInvolvements[0].ConfidenceEvidence, direct)
	if got := DeriveOutcome(taxonomy, withExternal, withExternal, withExternalEvidence, withExternalEvidence, false, false); got.AgreementConfidence != ConfidenceHigh {
		t.Fatal("external corroboration confidence invariant failed")
	}
}

func TestConfidenceEvidenceBindsEveryProposal(t *testing.T) {
	taxonomy := FrozenTaxonomy()
	projection := completeProjection()
	facts := trustedFacts()
	evidence := completeEvidence(t, taxonomy, projection, facts)
	bindings := ProposalValueBindings(taxonomy, projection)
	if len(bindings) < 10 {
		t.Fatalf("proposal bindings = %d", len(bindings))
	}
	if err := ValidateConfidenceEvidence(taxonomy, projection, evidence, facts); err != nil {
		t.Fatalf("complete evidence error = %v", err)
	}
	mutated := slices.Clone(evidence)
	mutated[0].ProposalValueDigest = digestHex("wrong-value")
	if err := ValidateConfidenceEvidence(taxonomy, projection, mutated, facts); !errors.Is(err, ErrEvidenceBinding) {
		t.Fatalf("wrong proposal binding error = %v", err)
	}
	unknownRule := slices.Clone(evidence)
	unknownRule[0].SignalRuleID = "MODEL_RULE"
	if err := ValidateConfidenceEvidence(taxonomy, projection, unknownRule, facts); !errors.Is(err, ErrUnknownSignalRule) {
		t.Fatalf("unknown signal rule error = %v", err)
	}
	signalMismatch := slices.Clone(evidence)
	signalMismatch[0].InputFactSelector = "VALIDATOR_SIGNAL_RULE_MATCH_DIGEST"
	signalMismatch[0].InputFactValueDigest = facts["VALIDATOR_SIGNAL_RULE_MATCH_DIGEST"][0].Digest
	signalMismatch[0].SignalRuleID = "SOURCE_PROPOSAL_GAP_V1"
	signalMismatch[0].RationaleCode = "SOURCE_GAP_CUE"
	if err := ValidateConfidenceEvidence(taxonomy, projection, signalMismatch, facts); !errors.Is(err, ErrEvidenceBinding) {
		t.Fatalf("signal rule field/value mismatch error = %v", err)
	}
	sourceBinding := ProposalValueBinding{}
	for _, binding := range ProposalValueBindings(taxonomy, projection) {
		if binding.ProposalField == "evidenceExpectationCodes" && binding.SemanticValue == "SAFETY_MANAGEMENT_RECORD" {
			sourceBinding = binding
		}
	}
	validSignalFacts := EvidenceFacts{"VALIDATOR_SIGNAL_RULE_MATCH_DIGEST": {{Digest: digestHex("source-gap-signal"), SignalRuleID: "SOURCE_PROPOSAL_GAP_V1"}}}
	wrongValueSignal := []ConfidenceEvidence{{
		ProposalField: sourceBinding.ProposalField, ProposalValueDigest: sourceBinding.ValueDigest,
		RationaleCode: "SOURCE_GAP_CUE", InputFactSelector: "VALIDATOR_SIGNAL_RULE_MATCH_DIGEST",
		InputFactValueDigest: digestHex("source-gap-signal"), SignalRuleID: "SOURCE_PROPOSAL_GAP_V1",
	}}
	if err := ValidateConfidenceEvidence(taxonomy, projection, wrongValueSignal, validSignalFacts); !errors.Is(err, ErrEvidenceBinding) {
		t.Fatalf("signal allowed field but wrong value error = %v", err)
	}
}

func TestExternalInvolvementEdgesAreIndependentAndOptional(t *testing.T) {
	taxonomy := FrozenTaxonomy()
	without := completeProjection()
	without.ExternalInvolvements = nil
	if err := ValidateProjection(taxonomy, without); err != nil {
		t.Fatalf("zero edge projection error = %v", err)
	}
	with := completeProjection()
	with.ExternalInvolvements = []ExternalInvolvement{
		externalEdge("ANSP", "COORDINATION", "ANSP_COORDINATION_REQUIRED"),
		externalEdge("CNS_PROVIDER", "EVIDENCE_CONTRIBUTION", "EVIDENCE_CONTRIBUTION_REQUIRED"),
	}
	facts := trustedFacts()
	for i := range with.ExternalInvolvements {
		binding := ExternalInvolvementBinding(taxonomy, with.ExternalInvolvements[i])
		with.ExternalInvolvements[i].ConfidenceEvidence = []ConfidenceEvidence{{
			ProposalField: "externalInvolvements", ProposalValueDigest: binding.ValueDigest,
			RationaleCode: "EXTERNAL_INTERFACE_CUE", InputFactSelector: "QUESTION_BODY_DIGEST",
			InputFactValueDigest: facts["QUESTION_BODY_DIGEST"][0].Digest,
		}}
	}
	if err := ValidateProjectionEvidence(taxonomy, with, trustedFacts()); err != nil {
		t.Fatalf("two independent edges error = %v", err)
	}
	duplicate := with
	duplicate.ExternalInvolvements = append(slices.Clone(with.ExternalInvolvements), with.ExternalInvolvements[0])
	if err := ValidateProjection(taxonomy, duplicate); !errors.Is(err, ErrDuplicateProposalValue) {
		t.Fatalf("duplicate edge error = %v", err)
	}
	forbiddenSelector := with
	forbiddenSelector.ExternalInvolvements = cloneJSON(with.ExternalInvolvements)
	forbiddenSelector.ExternalInvolvements[0].ConfidenceEvidence[0].InputFactSelector = "FORM_METADATA_DIGEST"
	forbiddenSelector.ExternalInvolvements[0].ConfidenceEvidence[0].InputFactValueDigest = trustedFacts()["FORM_METADATA_DIGEST"][0].Digest
	if err := ValidateProjectionEvidence(taxonomy, forbiddenSelector, trustedFacts()); !errors.Is(err, ErrEvidenceBinding) {
		t.Fatalf("external-edge forbidden selector error = %v", err)
	}
	operator := with
	operator.ExternalInvolvements = []ExternalInvolvement{externalEdge("AERODROME_OPERATOR", "COORDINATION", "ANSP_COORDINATION_REQUIRED")}
	if err := ValidateProjection(taxonomy, operator); !errors.Is(err, ErrUnknownCode) {
		t.Fatalf("provider hierarchy/self edge error = %v", err)
	}
	tooManyEdgeSources := cloneJSON(with)
	tooManyEdgeSources.ExternalInvolvements[0].SourceRefs = make([]SourceReference, 17)
	for index := range tooManyEdgeSources.ExternalInvolvements[0].SourceRefs {
		tooManyEdgeSources.ExternalInvolvements[0].SourceRefs[index] = SourceReference{Kind: "PACKAGE_SOURCE_PROPOSAL", ReferenceDigest: digestHex(fmt.Sprintf("edge-source-%d", index))}
	}
	if err := ValidateProjection(taxonomy, tooManyEdgeSources); !errors.Is(err, ErrInvalidResolution) {
		t.Fatalf("external edge sourceRefs maximum error = %v", err)
	}
	wrongEdgeRationale := cloneJSON(with)
	wrongEdgeRationale.ExternalInvolvements[0].RationaleCodes = []string{"GOVERNANCE_CUE"}
	if err := ValidateProjection(taxonomy, wrongEdgeRationale); !errors.Is(err, ErrUnknownCode) {
		t.Fatalf("external edge rationale subset error = %v", err)
	}
	wrongEdgeSignal := cloneJSON(with)
	wrongEdgeSignal.ExternalInvolvements = wrongEdgeSignal.ExternalInvolvements[:1]
	wrongEdgeBinding := ExternalInvolvementBinding(taxonomy, wrongEdgeSignal.ExternalInvolvements[0])
	wrongEdgeSignal.ExternalInvolvements[0].ConfidenceEvidence = []ConfidenceEvidence{{
		ProposalField: "externalInvolvements", ProposalValueDigest: wrongEdgeBinding.ValueDigest,
		RationaleCode: "SOURCE_GAP_CUE", InputFactSelector: "VALIDATOR_SIGNAL_RULE_MATCH_DIGEST",
		InputFactValueDigest: digestHex("mismatched-edge-rule"), SignalRuleID: "SOURCE_PROPOSAL_GAP_V1",
	}}
	if _, err := NewPassProposalRecord(taxonomy, PassProposalInput{
		Identity: baseIdentity(1, digestHex("edge-rule")), ClassificationRunID: "aga-classification-run-test",
		PassRole: PassCandidate, PassRunID: "aga-classification-pass-candidate-test", PromptDigest: FrozenPromptDigest,
		ModelDescriptorDigest: ComputeModelDescriptorDigest(testModelDescriptor()), InputDigest: digestHex("edge-rule-input"),
		Projection: wrongEdgeSignal,
	}); !errors.Is(err, ErrEvidenceBinding) {
		t.Fatalf("external edge signal rule mismatch error = %v", err)
	}
}

func TestPassProposalRecordsAreCompleteAndResolvable(t *testing.T) {
	_, result := exactClassificationFixture(t)
	if len(result.CandidateRecords) != FrozenBaseQuestionCount || len(result.ChallengeRecords) != FrozenBaseQuestionCount || len(result.Items) != FrozenBaseQuestionCount {
		t.Fatalf("counts = candidate %d challenge %d items %d", len(result.CandidateRecords), len(result.ChallengeRecords), len(result.Items))
	}
	if result.Aggregate.ItemCount != FrozenBaseQuestionCount || result.Aggregate.PassProposalRecordCount != FrozenPassProposalRecordCount || result.Aggregate.AggregateDigest != result.AggregateDigest || result.RunReceipt.ClassificationRunDigest != result.ClassificationRunDigest {
		t.Fatal("aggregate/run graph invariant failed")
	}
	if result.Aggregate.Exceptions.BlockedSourceGap.Count != FrozenSourceGapCount || result.Aggregate.Exceptions.ExternalApplicabilityUnresolved.Count != FrozenExternalUnresolvedCount || result.Aggregate.Exceptions.SourceGapExternalUnresolvedOverlap.Count != FrozenSourceExternalOverlap || codeCountValue(result.Aggregate.Distributions.ExtractionStateCounts, ExtractionCandidate) != FrozenExtractedCandidateCount || codeCountValue(result.Aggregate.Distributions.ExtractionStateCounts, ExtractionExactSourceBacked) != FrozenExactSourceBackedCount {
		t.Fatal("frozen aggregate totals invariant failed")
	}
	passDigests := make(map[string]struct{}, 2620)
	for _, record := range append(slices.Clone(result.CandidateRecords), result.ChallengeRecords...) {
		passDigests[record.PassResultDigest] = struct{}{}
	}
	for _, item := range result.Items {
		if _, ok := passDigests[item.PassOneResultDigest]; !ok {
			t.Fatal("item candidate record does not resolve")
		}
		if _, ok := passDigests[item.PassTwoResultDigest]; !ok {
			t.Fatal("item challenge record does not resolve")
		}
	}
}

func TestClassificationDigestGraphIsNonCircular(t *testing.T) {
	if _, err := canonicalJSON(map[string]string{"invalid": string([]byte{0xff})}); err == nil {
		t.Fatal("canonical JSON accepted invalid UTF-8 before hashing")
	}
	taxonomy := FrozenTaxonomy()
	id := baseIdentity(1, digestHex("one"))
	projection := completeProjection()
	facts := trustedFacts()
	record := sealedRecord(t, taxonomy, id, PassCandidate, projection, completeEvidence(t, taxonomy, projection, facts))
	originalPass := record.PassResultDigest
	record.PassResultDigest = digestHex("ignored-self")
	if got := ComputePassResultDigest(record); got != originalPass {
		t.Fatalf("pass digest included itself: got %s want %s", got, originalPass)
	}
	item := SealedClassificationItem{
		Identity: id, Projection: projection, AgreementConfidence: ConfidenceHigh,
		RecommendationState: RecommendationAutoProposed, GovernanceState: governanceState(),
		PassOneResultDigest: digestHex("p1"), PassTwoResultDigest: digestHex("p2"),
		PassOneRunID: "aga-classification-pass-candidate-one", PassTwoRunID: "aga-classification-pass-challenge-one",
		PromptDigest: FrozenPromptDigest, ModelDescriptorDigests: []string{ComputeModelDescriptorDigest(testModelDescriptor())},
		TaxonomyDigest: taxonomy.Digest, InputDigest: ComputeRunInputDigest(FrozenFixedInputDigests()),
		ClassificationRunDigest: digestHex("run"), AggregateDigest: digestHex("aggregate"),
	}
	item.ItemSemanticDigest = ComputeItemSemanticDigest(item)
	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal sealed item: %v", err)
	}
	if strings.Contains(string(encoded), `"identity"`) || strings.Contains(string(encoded), `"projection"`) || !strings.Contains(string(encoded), `"packageVersion"`) || !strings.Contains(string(encoded), `"mainDomainCode"`) {
		t.Fatal("sealed item flat schema invariant failed")
	}
	originalItem := item.ItemSemanticDigest
	item.PassOneResultDigest = digestHex("changed-pass")
	item.PassTwoResultDigest = digestHex("changed-pass-two")
	item.ClassificationRunDigest = digestHex("changed-run")
	item.AggregateDigest = digestHex("changed-aggregate")
	item.ItemSemanticDigest = digestHex("changed-self")
	if got := ComputeItemSemanticDigest(item); got != originalItem {
		t.Fatalf("semantic digest included excluded back-reference: got %s want %s", got, originalItem)
	}
	item.Projection.MainDomainCode = "SAFETY_MANAGEMENT_RISK_ASSESSMENT"
	if got := ComputeItemSemanticDigest(item); got == originalItem {
		t.Fatal("semantic digest ignored semantic projection change")
	}
}

func TestProjectionNormalizationIsDeterministicAndOptionalSetsStayArrays(t *testing.T) {
	taxonomy := FrozenTaxonomy()
	projection := completeProjection()
	reordered := completeProjection()
	slices.Reverse(reordered.TopicCodes)
	slices.Reverse(reordered.InspectionTypeCodes)
	first := sealedRecord(t, taxonomy, baseIdentity(1, digestHex("normalization")), PassCandidate, projection, nil)
	second := sealedRecord(t, taxonomy, baseIdentity(1, digestHex("normalization")), PassCandidate, reordered, nil)
	if first.PassResultDigest != second.PassResultDigest {
		t.Fatalf("reorder-equivalent pass digests differ: %s/%s", first.PassResultDigest, second.PassResultDigest)
	}
	optional := completeProjection()
	optional.TopicCodes = nil
	optional.EvidenceExpectationCodes = nil
	record := sealedRecord(t, taxonomy, baseIdentity(2, digestHex("optional")), PassCandidate, optional, nil)
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"topicCodes":null`) || strings.Contains(string(encoded), `"evidenceExpectationCodes":null`) || strings.Contains(string(encoded), `"externalInvolvements":null`) {
		t.Fatal("schema-required array invariant failed")
	}
}

func sealedRecord(t *testing.T, taxonomy Taxonomy, identity BaseIdentity, role PassRole, projection ProposalProjection, evidence []ConfidenceEvidence) PassProposalRecord {
	t.Helper()
	record, err := NewPassProposalRecord(taxonomy, PassProposalInput{
		Identity: identity, ClassificationRunID: "aga-classification-run-test", PassRole: role,
		PassRunID: "aga-classification-pass-" + strings.ToLower(string(role)) + "-test", PromptDigest: FrozenPromptDigest,
		ModelDescriptorDigest: ComputeModelDescriptorDigest(testModelDescriptor()), InputDigest: digestHex("batch-" + string(role)),
		Projection: projection, RationaleCodes: []string{"SOURCE_EVIDENCE_PRESENT"},
		ConfidenceEvidence: evidence, SourceRefs: []SourceReference{{Kind: "PACKAGE_SOURCE_PROPOSAL", ReferenceDigest: digestHex("source")}},
	})
	if err != nil {
		t.Fatalf("NewPassProposalRecord() error = %v", err)
	}
	return record
}

func completeProjection() ProposalProjection {
	return ProposalProjection{
		MainDomainCode:         "QUALITY_MANAGEMENT",
		TopicCodes:             []string{"QUALITY_MANAGEMENT_SYSTEM", "STAFFING_AND_COMPETENCE"},
		InspectionProfileCodes: []string{"AERODROME_MANAGEMENT_SYSTEM"},
		InspectionTypeCodes:    []string{"DOCUMENT_AND_RECORD_REVIEW", "PERIODIC_SURVEILLANCE"},
		CanonicalTargetKind:    "SYSTEM", TargetProfileCode: "AERODROME_MANAGEMENT_SYSTEM",
		OperationQualifiers:      []Qualifier{{Key: "OPERATION_STATUS", Value: "ACTIVE"}},
		ActivityQualifiers:       []Qualifier{{Key: "ACTIVITY_TYPE", Value: "MAINTENANCE"}},
		ApplicabilityDisposition: "APPLICABLE",
		EvidenceExpectationCodes: []string{"AUDIT_OR_INSPECTION_RECORD", "SAFETY_MANAGEMENT_RECORD"},
	}
}

func trustedFacts() EvidenceFacts {
	return EvidenceFacts{
		"QUESTION_BODY_DIGEST":               {{Digest: digestHex("body")}},
		"FORM_METADATA_DIGEST":               {{Digest: digestHex("form")}},
		"SOURCE_PROPOSAL_DIGEST":             {{Digest: digestHex("proposal")}},
		"SOURCE_REFERENCE_DIGEST":            {{Digest: digestHex("reference")}},
		"RESEARCH_ROW_DIGEST":                {{Digest: digestHex("research")}},
		"VALIDATOR_SIGNAL_RULE_MATCH_DIGEST": {{Digest: digestHex("signal"), SignalRuleID: "EXPLICIT_EXTERNAL_ACTOR_V1"}},
	}
}

func completeEvidence(t *testing.T, taxonomy Taxonomy, projection ProposalProjection, facts EvidenceFacts) []ConfidenceEvidence {
	t.Helper()
	bindings := ProposalValueBindings(taxonomy, projection)
	evidence := make([]ConfidenceEvidence, 0, len(bindings))
	for _, binding := range bindings {
		if binding.ProposalField == "externalInvolvements" {
			continue
		}
		evidence = append(evidence, ConfidenceEvidence{
			ProposalField: binding.ProposalField, ProposalValueDigest: binding.ValueDigest,
			RationaleCode: "SOURCE_EVIDENCE_PRESENT", InputFactSelector: "QUESTION_BODY_DIGEST",
			InputFactValueDigest: facts["QUESTION_BODY_DIGEST"][0].Digest,
		})
	}
	return evidence
}

func baseIdentity(ordinal int, textDigest string) BaseIdentity {
	return BaseIdentity{
		PackageVersion:    FrozenPackageVersion,
		PackageJSONSHA256: FrozenPackageJSONSHA256, FormCode: "FSS-AGA-FORM-002",
		ProposalID: fmt.Sprintf("all-forms-preview-002-%04d", ordinal), Ordinal: ordinal,
		TextDigest: textDigest,
	}
}

func packageIdentity(index int) BaseIdentity {
	form := (index / 50) + 1
	ordinal := (index % 50) + 1
	return BaseIdentity{
		PackageVersion: FrozenPackageVersion, PackageJSONSHA256: FrozenPackageJSONSHA256,
		FormCode:   fmt.Sprintf("FSS-AGA-FORM-%03d", form),
		ProposalID: fmt.Sprintf("all-forms-preview-%03d-%04d", form, ordinal), Ordinal: ordinal,
		TextDigest: digestHex(fmt.Sprintf("text-%04d", index/2)),
	}
}

func testModelDescriptor() ModelDescriptor {
	return ModelDescriptor{
		ModelID: "test-model", ModelIDSource: "accepted-collaboration-spawn-agent-model-override",
		RuntimeReportedFamily: "test-family", Service: "Codex", Interface: "API",
		RequestedReasoningEffort: "xhigh", ForkTurns: "none", SnapshotBuildLabel: nil,
		UnavailableFields: []string{"exactModelVersion", "serviceTier", "snapshotBuildLabel"},
	}
}

var (
	exactFixtureOnce   sync.Once
	exactFixtureInput  ClassificationInput
	exactFixtureResult ClassificationResult
	exactFixtureError  error
	exactFixtureBodies map[string]string
)

type testBatchManifest struct {
	Batches []struct {
		Identities []BaseIdentity `json:"identities"`
	} `json:"batches"`
}

func loadExactFixtureBodies() (map[string]string, error) {
	bytes, err := os.ReadFile("../../../../deliverables/aga-all-forms-source-risk-draft-2026-08-01/AGA_ALL_FORMS_SOURCE_RISK_DRAFT.json")
	if err != nil {
		return nil, err
	}
	var document struct {
		Forms []struct {
			FormCode  string `json:"formCode"`
			Questions []struct {
				ProposalID   string `json:"proposalId"`
				Ordinal      int    `json:"ordinal"`
				OriginalText string `json:"originalText"`
				TextDigest   string `json:"textDigest"`
			} `json:"questions"`
		} `json:"forms"`
	}
	if err := json.Unmarshal(bytes, &document); err != nil {
		return nil, err
	}
	bodies := make(map[string]string, FrozenBaseQuestionCount)
	for _, form := range document.Forms {
		for _, question := range form.Questions {
			identity := BaseIdentity{PackageVersion: FrozenPackageVersion, PackageJSONSHA256: FrozenPackageJSONSHA256, FormCode: form.FormCode, ProposalID: question.ProposalID, Ordinal: question.Ordinal, TextDigest: question.TextDigest}
			if rawTextDigest(question.OriginalText) != question.TextDigest {
				return nil, ErrDigestMismatch
			}
			bodies[identity.Key()] = question.OriginalText
		}
	}
	if len(bodies) != FrozenBaseQuestionCount {
		return nil, ErrPassBijection
	}
	return bodies, nil
}

func exactClassificationFixture(t *testing.T) (ClassificationInput, ClassificationResult) {
	t.Helper()
	exactFixtureOnce.Do(func() {
		exactFixtureInput, exactFixtureResult, exactFixtureError = buildExactClassificationFixture()
	})
	if exactFixtureError != nil {
		t.Fatalf("build exact classification fixture: %v", exactFixtureError)
	}
	return exactFixtureInput, exactFixtureResult
}

func buildExactClassificationFixture() (ClassificationInput, ClassificationResult, error) {
	manifestBytes, err := os.ReadFile("../../../../deliverables/aga-question-classification-contract-v1/classification-batch-manifest.json")
	if err != nil {
		return ClassificationInput{}, ClassificationResult{}, err
	}
	var manifest testBatchManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return ClassificationInput{}, ClassificationResult{}, err
	}
	exactFixtureBodies, err = loadExactFixtureBodies()
	if err != nil {
		return ClassificationInput{}, ClassificationResult{}, err
	}
	taxonomy := FrozenTaxonomy()
	projection := completeProjection()
	input := testClassificationInput("aga-classification-run-test")
	input.OrderedBaseIdentities = make([]BaseIdentity, 0, FrozenBaseQuestionCount)
	input.PassInputsByRole = map[PassRole][]ClassificationPassInput{PassCandidate: {}, PassChallenge: {}}
	input.CandidateRecords = make([]PassProposalRecord, 0, FrozenBaseQuestionCount)
	input.ChallengeRecords = make([]PassProposalRecord, 0, FrozenBaseQuestionCount)
	input.GovernanceByIdentity = make(map[string]GovernanceState, FrozenBaseQuestionCount)
	input.EvidenceFactsByIdentity = make(map[string]EvidenceFacts, FrozenBaseQuestionCount)
	index := 0
	for batchIndex, batch := range manifest.Batches {
		candidatePassInput := testClassificationPassInput(input, PassCandidate, "aga-classification-pass-candidate-test", ComputeModelDescriptorDigest(testModelDescriptor()), batchIndex+1, batch.Identities)
		challengePassInput := testClassificationPassInput(input, PassChallenge, "aga-classification-pass-challenge-test", ComputeModelDescriptorDigest(testModelDescriptor()), batchIndex+1, batch.Identities)
		candidateInput := ComputeClassificationPassInputDigest(candidatePassInput)
		challengeInput := ComputeClassificationPassInputDigest(challengePassInput)
		input.PassInputsByRole[PassCandidate] = append(input.PassInputsByRole[PassCandidate], candidatePassInput)
		input.PassInputsByRole[PassChallenge] = append(input.PassInputsByRole[PassChallenge], challengePassInput)
		for _, identity := range batch.Identities {
			if index >= FrozenBaseQuestionCount {
				return ClassificationInput{}, ClassificationResult{}, ErrPassBijection
			}
			facts := trustedFacts()
			facts["QUESTION_BODY_DIGEST"] = []EvidenceFact{{Digest: identity.TextDigest}}
			evidence := make([]ConfidenceEvidence, 0)
			for _, binding := range ProposalValueBindings(taxonomy, projection) {
				if binding.ProposalField != "externalInvolvements" {
					evidence = append(evidence, ConfidenceEvidence{ProposalField: binding.ProposalField, ProposalValueDigest: binding.ValueDigest, RationaleCode: "SOURCE_EVIDENCE_PRESENT", InputFactSelector: "QUESTION_BODY_DIGEST", InputFactValueDigest: identity.TextDigest})
				}
			}
			candidate, err := NewPassProposalRecord(taxonomy, PassProposalInput{
				Identity: identity, ClassificationRunID: input.ClassificationRunID, PassRole: PassCandidate,
				PassRunID: "aga-classification-pass-candidate-test", PromptDigest: FrozenPromptDigest,
				ModelDescriptorDigest: ComputeModelDescriptorDigest(testModelDescriptor()), InputDigest: candidateInput,
				Projection: projection, RationaleCodes: []string{"SOURCE_EVIDENCE_PRESENT"}, ConfidenceEvidence: evidence,
				SourceRefs: []SourceReference{{Kind: "PACKAGE_SOURCE_PROPOSAL", ReferenceDigest: digestHex("source")}},
			})
			if err != nil {
				return ClassificationInput{}, ClassificationResult{}, err
			}
			challenge, err := NewPassProposalRecord(taxonomy, PassProposalInput{
				Identity: identity, ClassificationRunID: input.ClassificationRunID, PassRole: PassChallenge,
				PassRunID: "aga-classification-pass-challenge-test", PromptDigest: FrozenPromptDigest,
				ModelDescriptorDigest: ComputeModelDescriptorDigest(testModelDescriptor()), InputDigest: challengeInput,
				Projection: projection, RationaleCodes: []string{"SOURCE_EVIDENCE_PRESENT"}, ConfidenceEvidence: evidence,
				SourceRefs: []SourceReference{{Kind: "PACKAGE_SOURCE_PROPOSAL", ReferenceDigest: digestHex("source")}},
			})
			if err != nil {
				return ClassificationInput{}, ClassificationResult{}, err
			}
			governance := governanceState()
			if index < FrozenSourceGapCount {
				governance.QuestionSourceProposalGap = true
				governance.ExternalApplicabilityUnresolved = true
				governance.BlockerCodes = append(governance.BlockerCodes, "PROVIDER_APPLICABILITY_UNRESOLVED")
			} else if index < FrozenExternalUnresolvedCount {
				governance.ExternalApplicabilityUnresolved = true
				governance.BlockerCodes = append(governance.BlockerCodes, "PROVIDER_APPLICABILITY_UNRESOLVED")
			}
			if index >= FrozenExtractedCandidateCount {
				governance.ExtractionState = ExtractionExactSourceBacked
			}
			sort.Strings(governance.BlockerCodes)
			key := identity.Key()
			input.OrderedBaseIdentities = append(input.OrderedBaseIdentities, identity)
			input.CandidateRecords = append(input.CandidateRecords, candidate)
			input.ChallengeRecords = append(input.ChallengeRecords, challenge)
			input.GovernanceByIdentity[key] = governance
			input.EvidenceFactsByIdentity[key] = facts
			index++
		}
	}
	if index != FrozenBaseQuestionCount {
		return ClassificationInput{}, ClassificationResult{}, ErrPassBijection
	}
	input.PassOneSealReceipt = testPassSeal(input, PassCandidate, input.CandidateRecords)
	input.PassTwoSealReceipt = testPassSeal(input, PassChallenge, input.ChallengeRecords)
	input.PassOneSealDigest = input.PassOneSealReceipt.PassSealDigest
	input.PassTwoSealDigest = input.PassTwoSealReceipt.PassSealDigest
	result, err := ReconcileClassification(input)
	return input, result, err
}

func testClassificationPassInput(input ClassificationInput, role PassRole, passRunID, modelDigest string, batchOrdinal int, identities []BaseIdentity) ClassificationPassInput {
	items := make([]ClassificationPassInputItem, len(identities))
	globalOffset := 0
	for index := 0; index < batchOrdinal-1; index++ {
		globalOffset += frozenBatchItemCounts[index]
	}
	for index, identity := range identities {
		extraction := ExtractionCandidate
		if globalOffset+index >= FrozenExtractedCandidateCount {
			extraction = ExtractionExactSourceBacked
		}
		proposals := []string{digestHex("source-proposal")}
		if globalOffset+index < FrozenSourceGapCount {
			proposals = []string{}
		}
		unresolved := "false"
		if globalOffset+index < FrozenExternalUnresolvedCount {
			unresolved = "true"
		}
		items[index] = ClassificationPassInputItem{Identity: identity, QuestionBody: exactFixtureBodies[identity.Key()], PackageFacts: ClassificationPackageFacts{FormKind: "CHECKLIST", FormRiskBand: "PROPOSED_REVIEW_REQUIRED", QuestionRiskBand: "PROPOSED_REVIEW_REQUIRED", QuestionRiskDomain: "UNCLASSIFIED", SourceMappingState: SourceMappingRequired, SourceAuthorityState: SourceAuthorityNotAttested, ExtractionState: extraction, RiskClassificationState: RiskExpertReviewRequired, DecisionState: DecisionNotSupplied, SourceProposalDigests: proposals, SourceReferenceDigests: []string{}}, ResearchCandidateFacts: ClassificationResearchCandidateFacts{FormCode: identity.FormCode, ProposalID: identity.ProposalID, Ordinal: fmt.Sprint(identity.Ordinal), TextDigest: identity.TextDigest, TargetKind: "AERODROME_OR_HELIPORT", OperationActivityQualifier: "AERODROME_OR_HELIPORT", PrimarySubjectProposal: "AERODROME_OPERATOR", OperationalInterfaceCandidates: "", EvidenceContributorCandidates: "", ProviderApplicabilityUnresolved: unresolved, UnresolvedReasons: "", SourceRefs: ""}}
	}
	return ClassificationPassInput{SchemaVersion: "aga-hybrid-classification-pass-input/v1", Purpose: "ROW_CLASSIFICATION_PRIVATE_INPUT", ClassificationRunID: input.ClassificationRunID, PassRole: role, PassRunID: passRunID, BatchOrdinal: batchOrdinal, TaxonomyVersion: FrozenTaxonomy().Version, TaxonomyDigest: FrozenTaxonomy().Digest, PromptDigest: FrozenPromptDigest, ModelDescriptorDigest: modelDigest, BatchManifestDigest: FrozenBatchManifestDigest, FixedInputDigests: input.FixedInputDigests, Items: items}
}

func testPassSeal(input ClassificationInput, role PassRole, records []PassProposalRecord) PassSealReceipt {
	passRunID := "aga-classification-pass-candidate-test"
	if role == PassChallenge {
		passRunID = "aga-classification-pass-challenge-test"
	}
	modelDigest := ComputeModelDescriptorDigest(testModelDescriptor())
	receipt := PassSealReceipt{ClassificationRunID: input.ClassificationRunID, PassRole: role, PassRunID: passRunID, PromptDigest: FrozenPromptDigest, ModelDescriptorDigest: modelDigest, BatchManifestDigest: FrozenBatchManifestDigest, BatchCount: len(frozenBatchItemCounts), ItemCount: FrozenBaseQuestionCount}
	offset := 0
	for batchIndex, count := range frozenBatchItemCounts {
		passInput := ComputeClassificationPassInputDigest(input.PassInputsByRole[role][batchIndex])
		receipt.OrderedInputDigests = append(receipt.OrderedInputDigests, passInput)
		batchRecords := records[offset : offset+count]
		output := PassBatchOutput{SchemaVersion: "aga-hybrid-classification-pass-batch/v1", ClassificationRunID: input.ClassificationRunID, PassRole: role, PassRunID: passRunID, BatchOrdinal: batchIndex + 1, PromptDigest: FrozenPromptDigest, ModelDescriptorDigest: modelDigest, InputDigest: passInput, Records: batchRecords}
		output.BatchOutputDigest = ComputePassBatchOutputDigest(output)
		receipt.OrderedBatchOutputDigests = append(receipt.OrderedBatchOutputDigests, output.BatchOutputDigest)
		for _, record := range batchRecords {
			receipt.OrderedPassResultDigests = append(receipt.OrderedPassResultDigests, record.PassResultDigest)
		}
		offset += count
	}
	receipt.PassInputSetDigest = computePassInputSetDigest(receipt)
	receipt.PassSealDigest = ComputePassSeal(receipt)
	return receipt
}

func testClassificationInput(runID string) ClassificationInput {
	fixed := FrozenFixedInputDigests()
	return ClassificationInput{
		ClassificationRunID: runID, RunInputDigest: ComputeRunInputDigest(fixed),
		PromptDigest: FrozenPromptDigest, TaxonomyDigest: FrozenTaxonomy().Digest,
		FixedInputDigests: fixed, ModelDescriptors: []ModelDescriptor{testModelDescriptor()},
		BatchManifestDigest: FrozenBatchManifestDigest,
		PassOneSealDigest:   digestHex("pass-one-seal"), PassTwoSealDigest: digestHex("pass-two-seal"),
	}
}

func governanceState() GovernanceState {
	return GovernanceState{
		SourceMappingState:      SourceMappingRequired,
		SourceAuthorityState:    SourceAuthorityNotAttested,
		RiskClassificationState: RiskExpertReviewRequired,
		DecisionState:           DecisionNotSupplied,
		ExtractionState:         ExtractionCandidate,
		BlockerCodes:            []string{"SOURCE_MAPPING_REQUIRED"},
	}
}

func externalEdge(provider, role, condition string) ExternalInvolvement {
	return ExternalInvolvement{
		ProviderTypeCode: provider, InvolvementRoleCode: role, ConditionCode: condition,
		ApplicabilityDisposition: "CONDITIONAL_ON_SERVICE_ARRANGEMENT", RationaleCodes: []string{"EXTERNAL_INTERFACE_CUE"},
		SourceRefs: []SourceReference{{Kind: "PACKAGE_SOURCE_PROPOSAL", ReferenceDigest: digestHex("external-source")}},
	}
}

func digestHex(seed string) string {
	value := fmt.Sprintf("%x", []byte(seed))
	for len(value) < 64 {
		value += "0"
	}
	return "sha256:" + value[:64]
}
