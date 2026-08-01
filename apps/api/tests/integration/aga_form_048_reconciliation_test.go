//go:build canonicaltest

package integration

import (
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/regulatory"
)

func TestAGAForm048Reconciliation(t *testing.T) {
	result, err := regulatory.CreateHybridReconciliation(regulatory.HybridReconciliationInput{PredecessorID: "candidate-draft", PredecessorDigest: "sha256:previous", Questions: []regulatory.CandidateDraftQuestion{{QuestionID: "q1", Wording: "Current wording"}}}, []regulatory.CandidateDraftQuestion{{QuestionID: "q1", Wording: "Supplied wording"}})
	if err != nil || len(result.Diffs) != 1 || result.Diffs[0].Outcome != "CHANGED" || !result.BindingRequired {
		t.Fatalf("reconciliation=%+v err=%v", result, err)
	}
}
