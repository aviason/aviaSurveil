package regulatory

import "testing"

func acceptedAuthorityFixtures() SourceAuthorityDecisionSet {
	return SourceAuthorityDecisionSet{
		"attestation-r": {DecisionID: "attestation-r", Outcome: "ACCEPT", SourceID: "regulatory", SourceVersionID: "v1", SourceHash: "sha256:r", ChainRole: "REGULATORY_AUTHORITY"},
		"attestation-p": {DecisionID: "attestation-p", Outcome: "ACCEPT", SourceID: "procedure", SourceVersionID: "v2", SourceHash: "sha256:p", ChainRole: "CONTROLLED_CAA_PROCEDURE"},
	}
}

func TestCreateOfficialSourceDraftRequiresAcceptedCurrentCompleteChain(t *testing.T) {
	request := OfficialSourceDraftRequest{CandidateID: "draft", ProviderScopeID: "scope", TargetID: "target", InspectionType: "aerodrome", Questions: []OfficialSourceQuestion{{QuestionID: "q1", Wording: "Inspect", Clauses: []OfficialSourceClauseRef{{ClauseID: "r1", SourceID: "regulatory", Role: "REGULATORY_AUTHORITY", SourceVersionID: "v1", SourceHash: "sha256:r", Current: true, SourceAuthorityAttestationID: "attestation-r"}, {ClauseID: "p1", SourceID: "procedure", Role: "CONTROLLED_CAA_PROCEDURE", SourceVersionID: "v2", SourceHash: "sha256:p", Current: true, SourceAuthorityAttestationID: "attestation-p"}}}}}
	draft, err := CreateOfficialSourceDraft(request, acceptedAuthorityFixtures())
	if err != nil || draft.Origin != string(RegulatoryTraceOrigin) || draft.GenerationRunID != "" || draft.BindingDigest == "" {
		t.Fatalf("draft=%+v err=%v", draft, err)
	}
	request.Questions[0].Clauses[0].SourceAuthorityAttestationID = ""
	if _, err := CreateOfficialSourceDraft(request, acceptedAuthorityFixtures()); err == nil {
		t.Fatal("source clause without an append-only attestation identity was accepted")
	}
}

func TestCreateOfficialSourceDraftRejectsSourceGapsAndMissingOwnerInputs(t *testing.T) {
	request := OfficialSourceDraftRequest{CandidateID: "draft", ProviderScopeID: "scope", TargetID: "target", InspectionType: "aerodrome", Questions: []OfficialSourceQuestion{{QuestionID: "q1", Wording: "Inspect", Clauses: []OfficialSourceClauseRef{{ClauseID: "gap", SourceID: "regulatory", Role: "REGULATORY_AUTHORITY", SourceVersionID: "v1", SourceHash: "sha256:r", Current: true, SourceAuthorityAttestationID: "attestation-r"}}}}}
	if _, err := CreateOfficialSourceDraft(request, acceptedAuthorityFixtures()); err == nil {
		t.Fatal("source chain without controlled procedure was accepted")
	}
}

func TestCreateOfficialSourceDraftRejectsCallerBooleanAuthority(t *testing.T) {
	request := OfficialSourceDraftRequest{CandidateID: "draft", ProviderScopeID: "scope", TargetID: "target", InspectionType: "aerodrome", Questions: []OfficialSourceQuestion{{QuestionID: "q1", Wording: "Inspect", Clauses: []OfficialSourceClauseRef{{ClauseID: "r1", SourceID: "regulatory", Role: "REGULATORY_AUTHORITY", SourceVersionID: "v1", SourceHash: "sha256:r", Current: true, SourceAuthorityAttestationID: "attestation-r"}, {ClauseID: "p1", SourceID: "procedure", Role: "CONTROLLED_CAA_PROCEDURE", SourceVersionID: "v2", SourceHash: "sha256:p", Current: true, SourceAuthorityAttestationID: "attestation-p"}}}}}
	if _, err := CreateOfficialSourceDraft(request, nil); err == nil {
		t.Fatal("nil server-side attestation resolver was accepted")
	}
	request.Questions[0].Clauses[0].SourceAuthorityAttestationID = "caller-forged"
	if _, err := CreateOfficialSourceDraft(request, acceptedAuthorityFixtures()); err == nil {
		t.Fatal("caller-forged attestation identity was accepted")
	}
}
