package httpapi

import (
	"encoding/json"
	"testing"

	"github.com/aviason/aviaSurveil/internal/httpapi/generated"
)

func TestGovernedChecklistTransportMappingKeepsStrictTraceVariants(t *testing.T) {
	input := generated.GovernedRegulatoryTraceView{State: "SOURCE_MAPPING_REQUIRED", Content: &generated.GovernedRegulatoryTraceContent{GovernedRegulatoryTraceSourceMappingRequiredContent: &generated.GovernedRegulatoryTraceSourceMappingRequiredContent{State: "SOURCE_MAPPING_REQUIRED", GapReason: "owner decision pending", MissingFields: []string{"sourceAuthority"}}}}
	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var decoded generated.GovernedRegulatoryTraceView
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Content == nil || decoded.Content.GovernedRegulatoryTraceSourceMappingRequiredContent == nil || decoded.Content.GovernedRegulatoryTraceResolvedContent != nil {
		t.Fatalf("trace cross-variant mapping: %+v", decoded)
	}
	var invalid generated.GovernedRegulatoryTraceView
	if err := json.Unmarshal([]byte(`{"state":"SOURCE_MAPPING_REQUIRED","content":{"state":"RESOLVED","sourceChain":[]}}`), &invalid); err != nil {
		// The outer shape may decode, but the inner strict discriminator must not
		// silently turn a resolved payload into a source-gap variant.
		return
	}
	if invalid.Content == nil || invalid.Content.GovernedRegulatoryTraceResolvedContent == nil {
		t.Fatal("strict union lost resolved discriminator")
	}
}
