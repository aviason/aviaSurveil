package regulatory

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTask4DeterministicProviderReturnsOnlyAValidatedDraftForTheSyntheticRequest(t *testing.T) {
	t.Parallel()
	request := SyntheticGenerationRequest()
	provider := NewDeterministicFixtureProvider()

	bundle, err := provider.Generate(context.Background(), request)
	if err != nil {
		t.Fatalf("generate synthetic candidate: %v", err)
	}
	if err := ValidateCandidateBundle(bundle, request); err != nil {
		t.Fatalf("provider returned an invalid candidate: %v", err)
	}
	if bundle.Status != GeneratedDraft {
		t.Fatalf("candidate status = %q, want %q", bundle.Status, GeneratedDraft)
	}
	if len(bundle.ComplianceMappings) == 0 || len(bundle.InspectionChecklist.Questions) == 0 {
		t.Fatal("provider returned an empty candidate shell")
	}
}

func TestTask4CanonicalDigestMatchesTrackedSyntheticCandidateFixture(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "..", "..", "docs", "regulatory-sources", "fixtures", "synthetic-ops-aoc-generation-candidate.v1.json")
	bytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read tracked synthetic fixture: %v", err)
	}
	var fixture CandidateBundle
	if err := json.Unmarshal(bytes, &fixture); err != nil {
		t.Fatalf("decode tracked synthetic fixture: %v", err)
	}
	requestDigest, err := requestDigest(fixture.GenerationRequest)
	if err != nil || requestDigest != fixture.InputDigest {
		t.Fatalf("request digest = %q err=%v, want %q", requestDigest, err, fixture.InputDigest)
	}
	outputDigest, err := candidateOutputDigest(fixture)
	if err != nil || outputDigest != fixture.OutputDigest {
		t.Fatalf("output digest = %q err=%v, want %q", outputDigest, err, fixture.OutputDigest)
	}
}

func TestTask4ProviderRejectsUnresolvedRealPilotAndNeverReturnsAuthoritativeState(t *testing.T) {
	t.Parallel()
	provider := NewImportedResultProvider(SyntheticCandidateBundle())
	if _, err := provider.Generate(context.Background(), RealOPSAOCGenerationRequest()); err == nil {
		t.Fatal("real OPS/AOC request with unresolved authority was accepted")
	}

	candidate := SyntheticCandidateBundle()
	candidate.Status = "PUBLISHED"
	if err := ValidateCandidateBundle(candidate, SyntheticGenerationRequest()); err == nil {
		t.Fatal("candidate claiming publication was accepted")
	}
}

func TestTask8ImpactProfileIsAnExactSecondSyntheticProfile(t *testing.T) {
	t.Parallel()
	bundle := SyntheticImpactCandidateBundle()
	if _, err := ValidateRequest(bundle.GenerationRequest, true); err != nil {
		t.Fatalf("exact synthetic impact profile rejected: %v", err)
	}
	bundle.GenerationRequest.SourceSnapshots[0].SourceHash = "sha256:7777777777777777777777777777777777777777777777777777777777777777"
	bundle.GenerationRequest.CanonicalInputDigest, _ = requestDigest(bundle.GenerationRequest)
	if _, err := ValidateRequest(bundle.GenerationRequest, true); err == nil {
		t.Fatal("invented synthetic impact source hash accepted")
	}
}

func TestTask4ProviderRejectsEmbeddedRequestReplacementAndUnsupportedCandidateSemantics(t *testing.T) {
	t.Parallel()
	expected := SyntheticGenerationRequest()
	replaced := SyntheticCandidateBundle()
	replaced.GenerationRequest.SourceSnapshots[0].ClauseLocators = []string{"Invented synthetic locator"}
	replaced.GenerationRequest.CanonicalInputDigest, _ = requestDigest(replaced.GenerationRequest)
	replaced.InputDigest = replaced.GenerationRequest.CanonicalInputDigest
	replaced.OutputDigest, _ = candidateOutputDigest(replaced)
	if _, err := NewImportedResultProvider(replaced).Generate(context.Background(), expected); err == nil {
		t.Fatal("provider accepted a candidate whose embedded request replaced the validated source binding")
	}

	invalid := SyntheticCandidateBundle()
	invalid.ComplianceMappings[0].Relationship = "APPROVES"
	invalid.OutputDigest, _ = candidateOutputDigest(invalid)
	if err := ValidateCandidateBundle(invalid, expected); err == nil {
		t.Fatal("candidate accepted unsupported mapping relationship")
	}
	invalid = SyntheticCandidateBundle()
	invalid.ComplianceMappings[0].Citations[0].Locator = "Invented synthetic locator"
	invalid.OutputDigest, _ = candidateOutputDigest(invalid)
	if err := ValidateCandidateBundle(invalid, expected); err == nil {
		t.Fatal("candidate accepted fabricated clause locator")
	}
	invalid = SyntheticCandidateBundle()
	invalid.InspectionChecklist.Questions[0].ExpectedEvidence = []string{"   "}
	invalid.OutputDigest, _ = candidateOutputDigest(invalid)
	if err := ValidateCandidateBundle(invalid, expected); err == nil {
		t.Fatal("candidate accepted blank expected evidence")
	}
}

func TestTask4SupportedClaimRegistryAndPinnedSyntheticProviderIdentity(t *testing.T) {
	t.Parallel()
	request := SyntheticGenerationRequest()
	for name, mutate := range map[string]func(*CandidateBundle){
		"requirement": func(candidate *CandidateBundle) {
			candidate.ComplianceMappings[0].Requirement = "The operator is automatically legally compliant."
		},
		"rationale": func(candidate *CandidateBundle) {
			candidate.ComplianceMappings[0].Rationale = "This establishes legal authority for the operator."
		},
		"prompt": func(candidate *CandidateBundle) {
			candidate.InspectionChecklist.Questions[0].Prompt = "Does this automatically conclude regulatory compliance?"
		},
		"verification method": func(candidate *CandidateBundle) {
			candidate.InspectionChecklist.Questions[0].VerificationMethod = "Automatically certify legal compliance without inspector review."
		},
		"expected evidence": func(candidate *CandidateBundle) {
			candidate.InspectionChecklist.Questions[0].ExpectedEvidence = []string{"Automatic enforcement conclusion"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := SyntheticCandidateBundle()
			mutate(&candidate)
			candidate.OutputDigest, _ = candidateOutputDigest(candidate)
			if err := ValidateCandidateBundle(candidate, request); err == nil {
				t.Fatalf("paraphrased %s claim crossed the synthetic boundary", name)
			}
		})
	}

	alteredProvider := SyntheticCandidateBundle()
	alteredProvider.GenerationRequest.ProviderID = "another-fixture-provider"
	alteredProvider.GenerationRequest.CanonicalInputDigest, _ = requestDigest(alteredProvider.GenerationRequest)
	alteredProvider.InputDigest = alteredProvider.GenerationRequest.CanonicalInputDigest
	alteredProvider.OutputDigest, _ = candidateOutputDigest(alteredProvider)
	if _, err := ValidateRequest(alteredProvider.GenerationRequest, true); err == nil {
		t.Fatal("synthetic request accepted an unpinned provider identity")
	}
	for name, gap := range map[string]*SourceGap{
		"blank reason":    {Status: "UNRESOLVED", Reason: " "},
		"resolved status": {Status: "RESOLVED", Reason: "Synthetic source gap"},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := SyntheticCandidateBundle()
			candidate.ComplianceMappings[0].SourceGap = gap
			candidate.OutputDigest, _ = candidateOutputDigest(candidate)
			if err := ValidateCandidateBundle(candidate, request); err == nil {
				t.Fatalf("candidate accepted source gap with %s", name)
			}
		})
	}
}

// Break caught: Task 5's single controlled edit alternative belongs only to
// the Admin edit validator and must never broaden Task 4 import semantics.
func TestTask4ImportRejectsTask5EditedRationale(t *testing.T) {
	t.Parallel()
	candidate := SyntheticCandidateBundle()
	candidate.ComplianceMappings[0].Rationale = syntheticEditedRationale
	candidate.OutputDigest, _ = candidateOutputDigest(candidate)
	if err := ValidateCandidateBundle(candidate, SyntheticGenerationRequest()); err == nil {
		t.Fatal("Task 5 edited rationale crossed the Task 4 import registry")
	}
}

func TestTask5CanonicalDigestGoldenVectors(t *testing.T) {
	bundle := SyntheticCandidateBundle()
	requestDigestValue, err := requestDigest(bundle.GenerationRequest)
	if err != nil {
		t.Fatal(err)
	}
	if requestDigestValue != "sha256:16263c98748063fc7e29dbb7744189cf9a81fd635bed641579772bafa70a6d64" {
		t.Fatalf("request digest = %s", requestDigestValue)
	}
	outputDigestValue, err := candidateOutputDigest(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if outputDigestValue != "sha256:377598cb1bee5388b19c9d7d4de34f1ff9f6b16b7ac1d2ff6cc5d96af798ad19" {
		t.Fatalf("output digest = %s", outputDigestValue)
	}

	editedMappings := append([]ComplianceMapping(nil), bundle.ComplianceMappings...)
	editedMappings[0].Rationale = syntheticEditedRationale
	editedDigest, err := CanonicalSHA256(map[string]any{
		"complianceMappings": editedMappings,
		"inspectionChecklist": map[string]any{
			"checklistId": "TPL-" + bundle.InspectionChecklist.ChecklistID,
			"questions":   bundle.InspectionChecklist.Questions,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if editedDigest != "sha256:31554d0293d724c6ececb947d3479d306e6281e1a010fd380c5fe2bf626561de" {
		t.Fatalf("edited candidate digest = %s", editedDigest)
	}

	importSemantic, err := importSemanticDigest("TASK5-GOLDEN-IMPORT", bundle)
	if err != nil {
		t.Fatal(err)
	}
	if importSemantic != "sha256:c956df2ea2b43049e976aab22f79ab1d1526112b55a4b4bcea5e288b279b4ce1" {
		t.Fatalf("import semantic digest = %s", importSemantic)
	}

	owners := []RequiredOwner{{
		DepartmentID:         "FLIGHT_OPERATIONS_INSPECTORATE",
		OrganizationalUnitID: "FLIGHT_OPERATIONS_INSPECTORATE",
		ApprovalRequired:     true,
	}}
	editSemantic, err := editSemanticDigest(EditCommand{
		CandidateID:           bundle.CandidateBundleID,
		ExpectedRevision:      1,
		ExpectedContentDigest: bundle.OutputDigest,
		ChangeReason:          "Apply the single controlled synthetic alternative.",
		Mappings:              editedMappings,
		Questions:             bundle.InspectionChecklist.Questions,
		RequiredOwners:        owners,
	})
	if err != nil {
		t.Fatal(err)
	}
	if editSemantic != "sha256:e512051bbf68b320c910183b467ddd6ac255419b2c112e95fb21492470ca7515" {
		t.Fatalf("edit semantic digest = %s", editSemantic)
	}

	submitSemantic, err := submitSemanticDigest(SubmitCommand{
		OperationID:           "TASK5-GOLDEN-SUBMIT",
		IdempotencyKey:        "TASK5-GOLDEN-SUBMIT-KEY",
		CandidateID:           "CAND-EDIT-GOLDEN",
		ExpectedContentDigest: editedDigest,
		Reason:                "Submit exact leaf.",
		ExpectedRevision:      2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if submitSemantic != "sha256:c000a5149e35d1aac3985e779d9b51c231c74d1aa1c9cb1321cacdbfea92f697" {
		t.Fatalf("submit semantic digest = %s", submitSemantic)
	}
}

func TestTask4Round3RejectsArbitraryVerificationAndEvidenceClaims(t *testing.T) {
	t.Parallel()
	request := SyntheticGenerationRequest()
	for name, mutate := range map[string]func(*CandidateBundle){
		"verification method": func(candidate *CandidateBundle) {
			candidate.InspectionChecklist.Questions[0].VerificationMethod = "Automatically certify legal compliance without inspector review."
		},
		"expected evidence": func(candidate *CandidateBundle) {
			candidate.InspectionChecklist.Questions[0].ExpectedEvidence = []string{"Automatic enforcement conclusion"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := SyntheticCandidateBundle()
			mutate(&candidate)
			candidate.OutputDigest, _ = candidateOutputDigest(candidate)
			if err := ValidateCandidateBundle(candidate, request); err == nil {
				t.Fatalf("arbitrary %s claim crossed the bounded registry", name)
			}
		})
	}
}
