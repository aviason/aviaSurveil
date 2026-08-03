package agacandidatedemo_test

import (
	"encoding/json"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/agacandidatedemo"
)

func TestProjectionJSONMatchesClosedHTTPContract(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"capability", agacandidatedemo.Capability{Available: true, Labels: []string{"candidate-only"}}, `{"available":true,"labels":["candidate-only"]}`},
		{"summary", agacandidatedemo.Summary{PackageDigest: "sha256:sealed", FormCount: 52, QuestionCount: 1310, SourceRequirements: []string{"EXACT_SOURCE_BYTES"}, Labels: []string{"release pending"}}, `{"packageDigest":"sha256:sealed","formCount":52,"questionCount":1310,"sourceRequirements":["EXACT_SOURCE_BYTES"],"labels":["release pending"]}`},
		{"form", agacandidatedemo.Form{Code: "FORM-1", Title: "Synthetic", QuestionCount: 1, QuestionExtractionState: "EXTRACTED_CANDIDATE_BOUNDARIES"}, `{"code":"FORM-1","title":"Synthetic","questionCount":1,"questionExtractionState":"EXTRACTED_CANDIDATE_BOUNDARIES"}`},
		{"question", agacandidatedemo.Question{ProposalID: "P-1", FormCode: "FORM-1", Ordinal: 1, Text: "Synthetic", TextDigest: "sha256:text", SourceGapCategory: "PROPOSAL_PRESENT_REVIEW_REQUIRED", RiskBand: "PROPOSED_REVIEW_REQUIRED"}, `{"proposalId":"P-1","formCode":"FORM-1","ordinal":1,"text":"Synthetic","textDigest":"sha256:text","sourceGapCategory":"PROPOSAL_PRESENT_REVIEW_REQUIRED","riskBand":"PROPOSED_REVIEW_REQUIRED"}`},
		{"page", agacandidatedemo.Page[agacandidatedemo.Form]{Items: []agacandidatedemo.Form{}, NextCursor: nil}, `{"items":[],"nextCursor":null}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload, err := json.Marshal(test.value)
			if err != nil {
				t.Fatal(err)
			}
			if string(payload) != test.want {
				t.Fatalf("projection JSON = %s, want %s", payload, test.want)
			}
		})
	}
}
