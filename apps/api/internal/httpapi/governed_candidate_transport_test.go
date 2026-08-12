package httpapi

import (
	"encoding/json"
	"testing"

	"github.com/aviason/aviaSurveil/internal/regulatory"
)

func TestGovernedCandidateViewEmitsGenerationRunLineage(t *testing.T) {
	view := governedCandidateView(regulatory.CandidateView{
		CandidateID:     "CAND-TRANSPORT-001",
		CandidateRootID: "CAND-TRANSPORT-001",
		GenerationRunID: "GENRUN-TRANSPORT-001",
		Status:          regulatory.GeneratedDraft,
	})
	if view.Lineage.GovernedGenerationRunLineage == nil {
		t.Fatal("generated candidate view lost its generation-run lineage discriminator")
	}
	if view.Lineage.GovernedGenerationRunLineage.GenerationRunId != "GENRUN-TRANSPORT-001" {
		t.Fatalf("generation-run lineage id = %q", view.Lineage.GovernedGenerationRunLineage.GenerationRunId)
	}
	if _, err := json.Marshal(view); err != nil {
		t.Fatalf("generated candidate view is not serializable: %v", err)
	}
}
