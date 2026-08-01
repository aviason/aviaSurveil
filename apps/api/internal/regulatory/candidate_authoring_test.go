package regulatory

import "testing"

func TestCreateDraftFromExistingCandidateKeepsCandidateOriginAndBlocksMissingMapping(t *testing.T) {
	draft, err := CreateDraftFromExistingCandidate(CandidateDraftInput{CandidateID: "candidate", CandidateRevision: 1, CandidateDigest: "sha256:candidate", ProviderScopeID: "scope", TargetID: "target", InspectionType: "aerodrome", Questions: []CandidateDraftQuestion{{QuestionID: "q1", Wording: "Inspect"}}})
	if err != nil || draft.Origin != string(ExistingChecklistCandidateOrigin) || draft.GenerationRunID != "" || len(draft.Blockers) == 0 {
		t.Fatalf("draft=%+v err=%v", draft, err)
	}
}

func TestCreateHybridReconciliationComputesDigestBoundDiffs(t *testing.T) {
	reconciled, err := CreateHybridReconciliation(HybridReconciliationInput{PredecessorID: "draft-1", PredecessorDigest: "sha256:old", Questions: []CandidateDraftQuestion{{QuestionID: "q1", Wording: "Updated wording"}}}, []CandidateDraftQuestion{{QuestionID: "q1", Wording: "Old wording"}})
	if err != nil || reconciled.Origin != string(HybridReconciledOrigin) || len(reconciled.Diffs) != 1 || reconciled.Diffs[0].Outcome != "CHANGED" {
		t.Fatalf("reconciled=%+v err=%v", reconciled, err)
	}
}
