package httpapi

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/aviason/aviaSurveil/internal/httpapi/generated"
	"github.com/aviason/aviaSurveil/internal/regulatory"
)

// Break caught: generated HTTP transport could alter the literal legacy
// source-gap candidate between JSON decoding and the regulatory application
// boundary, while the direct Go fixture still passed.
func TestGeneratedImportTransportPreservesLegacySourceGapCandidate(t *testing.T) {
	want := regulatory.SyntheticLegacyChecklistCandidateBundle()
	payload, err := json.Marshal(map[string]any{
		"operationId":     "HTTP-LEGACY-SOURCE-GAP-IMPORT",
		"idempotencyKey":  "HTTP-LEGACY-SOURCE-GAP-IMPORT",
		"candidateBundle": want,
	})
	if err != nil {
		t.Fatal(err)
	}
	var input generated.ImportAdminGovernedGenerationRunInput
	if err := json.Unmarshal(payload, &input); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(input.CandidateBundle)
	if err != nil {
		t.Fatal(err)
	}
	var got regulatory.CandidateBundle
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("generated HTTP transport changed the legacy source-gap candidate\n got=%s\nwant=%s", encoded, mustMarshalGovernanceTransport(t, want))
	}
	if err := regulatory.ValidateCandidateBundle(got, regulatory.SyntheticLegacyCandidateGenerationRequest()); err != nil {
		t.Fatalf("transport-preserved legacy source-gap candidate rejected: %v", err)
	}
}

func mustMarshalGovernanceTransport(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}
