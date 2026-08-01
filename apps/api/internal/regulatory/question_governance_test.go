package regulatory

import "testing"

// Break caught: a generated question could previously be persisted with only
// loose citations and flags, leaving publication and reviewers without the
// required scope and regulatory-trace views.
func TestGeneratedQuestionCarriesCompleteScopeRecommendationAndRegulatoryTrace(t *testing.T) {
	bundle := SyntheticCandidateBundle()
	question := bundle.InspectionChecklist.Questions[0]

	if question.Origin != RegulatoryTraceOrigin {
		t.Fatalf("question origin = %q, want %q", question.Origin, RegulatoryTraceOrigin)
	}
	if question.ScopeRecommendation.Classification != "MANDATORY_CORE" ||
		len(question.ScopeRecommendation.InputSignals) == 0 ||
		question.ScopeRecommendation.OperationalHistoryBasis == "" ||
		question.ScopeRecommendation.Rationale == "" ||
		!question.ScopeRecommendation.Guardrails.MandatoryControl ||
		!question.ScopeRecommendation.Guardrails.SafetyCritical ||
		!question.ScopeRecommendation.Guardrails.UnknownHistory ||
		question.ScopeRecommendation.ApprovalReviewState != "TECHNICAL_REVIEW_REQUIRED" {
		t.Fatalf("incomplete scope recommendation: %+v", question.ScopeRecommendation)
	}
	if question.RegulatoryTrace.State != "RESOLVED" ||
		question.RegulatoryTrace.SourceIdentity != "SYNTHETIC-OPS-AOC" ||
		question.RegulatoryTrace.SourceTitle == "" ||
		question.RegulatoryTrace.ImmutableVersion == "" ||
		question.RegulatoryTrace.SHA256 != syntheticSourceHash ||
		question.RegulatoryTrace.Locator == "" ||
		question.RegulatoryTrace.Page == "" ||
		question.RegulatoryTrace.Section == "" ||
		question.RegulatoryTrace.Clause == "" ||
		question.RegulatoryTrace.SourceType == "" ||
		question.RegulatoryTrace.Applicability == "" ||
		question.RegulatoryTrace.NationalReference == "" ||
		question.RegulatoryTrace.ControlledCAAProcedureMapping == "" ||
		question.RegulatoryTrace.VerificationObjective == "" ||
		len(question.RegulatoryTrace.ExpectedEvidence) == 0 ||
		question.RegulatoryTrace.CurrentnessState != "CURRENT" ||
		question.RegulatoryTrace.TechnicalReviewState != "TECHNICAL_REVIEW_REQUIRED" {
		t.Fatalf("incomplete regulatory trace: %+v", question.RegulatoryTrace)
	}
	if err := ValidateCandidateBundle(bundle, SyntheticGenerationRequest()); err != nil {
		t.Fatalf("complete traced candidate rejected: %v", err)
	}
}

// Break caught: a reviewer could not receive a repairable Draft when a legacy
// question has an explicit source-mapping gap, which encourages empty or
// invented citations instead of the required literal state.
func TestSourceMappingRequiredQuestionRemainsARepairableDraft(t *testing.T) {
	bundle := SyntheticCandidateBundle()
	question := &bundle.InspectionChecklist.Questions[0]
	question.Origin = ExistingChecklistCandidateOrigin
	question.Citations = []Citation{}
	question.RegulatoryTrace = RegulatoryTrace{State: SourceMappingRequired}
	bundle.SourceCurrentness = nil
	bundle.OutputDigest, _ = candidateOutputDigest(bundle)

	if err := ValidateCandidateBundle(bundle, SyntheticGenerationRequest()); err != nil {
		t.Fatalf("explicit source-mapping Draft rejected: %v", err)
	}
	if question.RegulatoryTrace.State != SourceMappingRequired {
		t.Fatalf("source-gap state = %q, want literal %q", question.RegulatoryTrace.State, SourceMappingRequired)
	}
	partialTrace := SyntheticLegacyChecklistCandidateBundle()
	partialTrace.InspectionChecklist.Questions[0].Citations = []Citation{{
		SourceSnapshotID: "SOURCE-SYNTHETIC-OPS-AOC",
		SourceHash:       syntheticSourceHash,
		ClauseID:         "CLAUSE-SYNTHETIC-OPS-AOC-1",
		Locator:          "Synthetic OPS/AOC 1",
	}}
	partialTrace.OutputDigest, _ = candidateOutputDigest(partialTrace)
	if err := ValidateCandidateBundle(partialTrace, SyntheticLegacyCandidateGenerationRequest()); err == nil {
		t.Fatal("source-gap Draft with a partial citation was accepted")
	}
	partialTrace = SyntheticLegacyChecklistCandidateBundle()
	partialTrace.InspectionChecklist.Questions[0].RegulatoryTrace.SourceTitle = "Unverified partial trace"
	partialTrace.OutputDigest, _ = candidateOutputDigest(partialTrace)
	if err := ValidateCandidateBundle(partialTrace, SyntheticLegacyCandidateGenerationRequest()); err == nil {
		t.Fatal("source-gap Draft with a partial trace was accepted")
	}
}

func TestSourceMappingRequiredTransportMarkerRemainsLiteral(t *testing.T) {
	bundle := SyntheticLegacyChecklistCandidateBundle()
	bundle.InspectionChecklist.Questions[0].RegulatoryTrace.TechnicalReviewState = "NOT_AVAILABLE"
	bundle.OutputDigest, _ = candidateOutputDigest(bundle)

	if err := ValidateCandidateBundle(bundle, SyntheticLegacyCandidateGenerationRequest()); err != nil {
		t.Fatalf("transport-compatible source-gap Draft rejected: %v", err)
	}
}

func TestSourceMappingRequiredTechnicalProjectionDoesNotChangeContentDigest(t *testing.T) {
	bundle := SyntheticLegacyChecklistCandidateBundle()
	baseline := bundle.OutputDigest
	bundle.InspectionChecklist.Questions[0].RegulatoryTrace.TechnicalReviewState = "NOT_AVAILABLE"
	projected, err := candidateOutputDigest(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if projected != baseline {
		t.Fatalf("source-gap technical projection changed content digest: baseline=%s projected=%s", baseline, projected)
	}
}

// Break caught: a historical question could be marked as a fully resolved
// regulatory question without the mandatory hybrid reconciliation record. An
// existing checklist stays a candidate-only source-gap Draft until its current
// controlled trace is reconciled under HYBRID_RECONCILED.
func TestExistingChecklistCandidateCannotBeElevatedToResolvedAuthority(t *testing.T) {
	question := SyntheticCandidateBundle().InspectionChecklist.Questions[0]
	question.Origin = ExistingChecklistCandidateOrigin

	if validQuestionGovernance(question) {
		t.Fatal("EXISTING_CHECKLIST_CANDIDATE with a resolved trace was accepted without hybrid reconciliation")
	}
}

// Break caught: the legacy candidate fixture previously tried to create a
// second GENERATION_INPUT partition in the same immutable evaluation, which
// PostgreSQL correctly rejects. Historical candidate input reuses the frozen
// current input partition while its question remains source-gap blocked.
func TestLegacyCandidateReusesTheFrozenCurrentInputPartition(t *testing.T) {
	request := syntheticLegacyCandidateRequest()
	if request.SecondaryCrosswalkPartition.PartitionID != "PARTITION-SYNTHETIC-INPUT" {
		t.Fatalf("legacy partition = %q, want frozen current input", request.SecondaryCrosswalkPartition.PartitionID)
	}
	const wantDigest = "sha256:5991bdb521a2e3385f94f282027670e914a5935e5a40e0a9b1daf87172573f3c"
	if request.CanonicalInputDigest != wantDigest {
		t.Fatalf("legacy input digest = %q, want %q", request.CanonicalInputDigest, wantDigest)
	}
}

// Break caught: automatic deferral could silently omit a mandatory,
// safety-critical, changed, overdue, or unknown-history control.
func TestQuestionGovernanceRejectsAutomaticDeferralForGuardedControls(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		mandatory bool
		safety    bool
		unknown   bool
		changed   bool
		overdue   bool
	}{
		{name: "mandatory", mandatory: true},
		{name: "safety-critical", safety: true},
		{name: "unknown-history", unknown: true},
		{name: "source-changed", changed: true},
		{name: "overdue", overdue: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			question := SyntheticCandidateBundle().InspectionChecklist.Questions[0]
			question.MandatoryCore = testCase.mandatory
			question.SafetyCritical = testCase.safety
			question.ScopeRecommendation.Guardrails = ScopeGuardrails{
				MandatoryControl:           testCase.mandatory,
				SafetyCritical:             testCase.safety,
				UnknownHistory:             testCase.unknown,
				SourceChanged:              testCase.changed,
				OverdueControl:             testCase.overdue,
				AutomaticDeferralPermitted: true,
			}
			question.ScopeRecommendation.AutomaticDeferral = true
			if validQuestionGovernance(question) {
				t.Fatalf("automatic deferral was accepted for guarded %s control", testCase.name)
			}
		})
	}
}

// Break caught: permissive validation could manufacture a generated Draft
// without the classification/rationale or immutable trace a reviewer needs,
// leaving an empty citation-like shape that only fails much later.
func TestQuestionGovernanceRejectsMissingRequiredScopeAndTraceFields(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		mutate func(*ChecklistQuestion)
	}{
		{
			name: "missing origin",
			mutate: func(question *ChecklistQuestion) {
				question.Origin = ""
			},
		},
		{
			name: "missing trace",
			mutate: func(question *ChecklistQuestion) {
				question.RegulatoryTrace = RegulatoryTrace{}
			},
		},
		{
			name: "missing classification",
			mutate: func(question *ChecklistQuestion) {
				question.ScopeRecommendation.Classification = ""
			},
		},
		{
			name: "missing rationale",
			mutate: func(question *ChecklistQuestion) {
				question.ScopeRecommendation.Rationale = ""
			},
		},
		{
			name: "missing applicability",
			mutate: func(question *ChecklistQuestion) {
				question.RegulatoryTrace.Applicability = ""
			},
		},
		{
			name: "missing currentness",
			mutate: func(question *ChecklistQuestion) {
				question.RegulatoryTrace.CurrentnessState = ""
			},
		},
		{
			name: "missing technical review",
			mutate: func(question *ChecklistQuestion) {
				question.RegulatoryTrace.TechnicalReviewState = ""
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			question := SyntheticCandidateBundle().InspectionChecklist.Questions[0]
			testCase.mutate(&question)
			if validQuestionGovernance(question) {
				t.Fatalf("incomplete %s question passed governance validation: %+v", testCase.name, question)
			}
		})
	}
}

// Break caught: an imported Draft could claim TECHNICALLY_APPROVED in its
// question payload before an attributed Department Manager decision existed.
// Approval is a derived read-model state, not a client-controlled Draft fact.
func TestQuestionGovernanceRejectsPreclaimedTechnicalApproval(t *testing.T) {
	bundle := SyntheticCandidateBundle()
	question := &bundle.InspectionChecklist.Questions[0]
	question.ScopeRecommendation.ApprovalReviewState = "TECHNICALLY_APPROVED"
	question.RegulatoryTrace.TechnicalReviewState = "TECHNICALLY_APPROVED"
	bundle.OutputDigest, _ = candidateOutputDigest(bundle)

	if err := ValidateCandidateBundle(bundle, SyntheticGenerationRequest()); err == nil {
		t.Fatal("Draft with a preclaimed technical approval was accepted")
	}
}

// Break caught: scope classification behavior was only exercised by the
// mandatory synthetic fixture, so focused and rotational-sample controls
// could regress despite using a complete current regulatory trace.
func TestQuestionGovernanceAllowsFocusedAndRotationalSampleControls(t *testing.T) {
	for _, classification := range []string{"FOCUSED_FULL", "ROTATIONAL_SAMPLE"} {
		t.Run(classification, func(t *testing.T) {
			question := SyntheticCandidateBundle().InspectionChecklist.Questions[0]
			question.MandatoryCore = false
			question.SafetyCritical = false
			question.ScopeRecommendation.Classification = classification
			question.ScopeRecommendation.InputSignals = []string{"Current source applicability", "Reviewed operational history"}
			question.ScopeRecommendation.OperationalHistoryBasis = "SYNTHETIC_REVIEWED_HISTORY"
			question.ScopeRecommendation.Rationale = "A reviewed current source trace supports the selected scope classification."
			question.ScopeRecommendation.Guardrails = ScopeGuardrails{AutomaticDeferralPermitted: true}
			question.ScopeRecommendation.AutomaticDeferral = false
			if !validQuestionGovernance(question) {
				t.Fatalf("complete %s control was rejected: %+v", classification, question)
			}
		})
	}
}

// Break caught: a legacy question could be relabelled as reconciled without
// retaining the candidate-only comparison that lets a reviewer see what
// changed. A reconciled question must bind the legacy input to the current
// approved trace without elevating the old wording or history to authority.
func TestHybridReconciledQuestionRetainsCandidateOnlyComparison(t *testing.T) {
	bundle := SyntheticHybridReconciledCandidateBundle()
	question := bundle.InspectionChecklist.Questions[0]

	if question.Origin != HybridReconciledOrigin {
		t.Fatalf("origin = %q, want %q", question.Origin, HybridReconciledOrigin)
	}
	if question.Reconciliation == nil {
		t.Fatal("hybrid question is missing its legacy-to-current comparison")
	}
	comparison := question.Reconciliation
	if comparison.LegacyQuestionID == "" || comparison.LegacyWording == "" ||
		comparison.LegacyOperationalIntent == "" || comparison.LegacyResultHistory == "" ||
		len(comparison.LegacyExpectedEvidence) == 0 || comparison.LegacyApplicability == "" ||
		comparison.LegacyScopeClassification == "" || comparison.CurrentWording != question.Prompt ||
		!sameStrings(comparison.CurrentExpectedEvidence, question.ExpectedEvidence) ||
		comparison.CurrentApplicability != question.RegulatoryTrace.Applicability ||
		comparison.CurrentScopeClassification != question.ScopeRecommendation.Classification ||
		!comparison.WordingChanged || !comparison.EvidenceChanged ||
		!comparison.ApplicabilityChanged || !comparison.ScopeChanged {
		t.Fatalf("incomplete hybrid comparison: %+v", comparison)
	}
	if err := ValidateCandidateBundle(bundle, SyntheticHybridReconciledGenerationRequest()); err != nil {
		t.Fatalf("hybrid reconciled candidate rejected: %v", err)
	}
}

// Break caught: an immutable question snapshot can correctly record that a
// technical review is required, while the live reviewer projection continued
// to display that review as pending after the exact Department Manager
// approval had been recorded.
func TestQuestionGovernanceProjectionShowsRecordedTechnicalApproval(t *testing.T) {
	question := SyntheticCandidateBundle().InspectionChecklist.Questions[0]
	projectQuestionTechnicalReviewState("TECHNICALLY_APPROVED", &question)
	if question.ScopeRecommendation.ApprovalReviewState != "TECHNICALLY_APPROVED" ||
		question.RegulatoryTrace.TechnicalReviewState != "TECHNICALLY_APPROVED" {
		t.Fatalf("approved projection remained pending: scope=%q trace=%q",
			question.ScopeRecommendation.ApprovalReviewState,
			question.RegulatoryTrace.TechnicalReviewState)
	}

	gap := SyntheticLegacyChecklistCandidateBundle().InspectionChecklist.Questions[0]
	projectQuestionTechnicalReviewState("TECHNICALLY_APPROVED", &gap)
	if gap.RegulatoryTrace.CurrentnessState != SourceMappingRequired ||
		gap.RegulatoryTrace.TechnicalReviewState != "NOT_AVAILABLE" {
		t.Fatalf("source-gap projection lost its literal blocked state: %+v", gap.RegulatoryTrace)
	}
}
