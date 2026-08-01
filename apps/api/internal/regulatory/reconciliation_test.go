package regulatory

import "testing"

func TestComputeReconciliationKeepsGapsAndDigestBound(t *testing.T) {
	result, err := ComputeReconciliation("candidate-root", "sha256:old", []CandidateDraftQuestion{{QuestionID: "Q1", Wording: "old"}, {QuestionID: "Q2", Wording: "removed"}}, []OfficialSourceQuestion{{QuestionID: "Q1", Wording: "new", Clauses: []OfficialSourceClauseRef{{ClauseID: "C1"}}}, {QuestionID: "Q3", Wording: "added"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceMapping != SourceMappingRequired || result.Digest == "" || len(result.Fields) != 3 {
		t.Fatalf("unexpected reconciliation result: %+v", result)
	}
	seen := map[string]string{}
	for _, field := range result.Fields {
		seen[field.QuestionID] = field.Outcome
	}
	if seen["Q1"] != "CHANGED" || seen["Q2"] != "REMOVED" || seen["Q3"] != "ADDED" {
		t.Fatalf("unexpected reconciliation outcomes: %#v", seen)
	}
}
