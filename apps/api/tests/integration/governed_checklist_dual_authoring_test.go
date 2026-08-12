//go:build canonicaltest

package integration

import (
	"testing"

	"github.com/aviason/aviaSurveil/internal/regulatory"
)

func TestGovernedChecklistDualAuthoring(t *testing.T) {
	resolver := regulatory.SourceAuthorityDecisionSet{
		"attestation-authority": {DecisionID: "attestation-authority", Outcome: "ACCEPT", SourceID: "regulatory", SourceVersionID: "v1", SourceHash: "sha256:authority", ChainRole: "REGULATORY_AUTHORITY"},
		"attestation-procedure": {DecisionID: "attestation-procedure", Outcome: "ACCEPT", SourceID: "procedure", SourceVersionID: "v2", SourceHash: "sha256:procedure", ChainRole: "CONTROLLED_CAA_PROCEDURE"},
	}
	draft, err := regulatory.CreateOfficialSourceDraft(regulatory.OfficialSourceDraftRequest{CandidateID: "official-synthetic", ProviderScopeID: "scope", TargetID: "target", InspectionType: "aerodrome", Questions: []regulatory.OfficialSourceQuestion{{QuestionID: "q1", Wording: "Inspect", Clauses: []regulatory.OfficialSourceClauseRef{{ClauseID: "authority", SourceID: "regulatory", Role: "REGULATORY_AUTHORITY", SourceVersionID: "v1", SourceHash: "sha256:authority", Current: true, SourceAuthorityAttestationID: "attestation-authority"}, {ClauseID: "procedure", SourceID: "procedure", Role: "CONTROLLED_CAA_PROCEDURE", SourceVersionID: "v2", SourceHash: "sha256:procedure", Current: true, SourceAuthorityAttestationID: "attestation-procedure"}}}}}, resolver)
	if err != nil || draft.GenerationRunID != "" || draft.Origin != string(regulatory.RegulatoryTraceOrigin) {
		t.Fatalf("official source draft=%+v err=%v", draft, err)
	}
}
