package application

import "testing"

func TestCanonicalExecutionTypeNormalizesAliases(t *testing.T) {
	tests := map[string]string{
		"RAMP":                  "RAMP_INSPECTION",
		"RAMP_INSPECTION":       "RAMP_INSPECTION",
		"CABIN":                 "CABIN_INSPECTION",
		"CABIN_INSPECTION":      "CABIN_INSPECTION",
		"FOLLOW_UP":             "FOLLOW_UP",
		"PERIODIC_SURVEILLANCE": "PERIODIC_SURVEILLANCE",
	}
	for input, want := range tests {
		got, err := CanonicalExecutionType(input)
		if err != nil || got != want {
			t.Fatalf("CanonicalExecutionType(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	if _, err := CanonicalExecutionType("ATO"); err == nil {
		t.Fatal("unsupported execution type was accepted")
	}
}

func TestCanonicalAuditTypeProviderPolicyFailsClosed(t *testing.T) {
	tests := []struct {
		provider string
		typeCode string
		allowed  bool
	}{
		{provider: "AERODROME_OPERATOR", typeCode: "RAMP_INSPECTION", allowed: true},
		{provider: "AERODROME_OPERATOR", typeCode: "CABIN_INSPECTION", allowed: false},
		{provider: "AIR_OPERATOR", typeCode: "RAMP", allowed: true},
		{provider: "AIR_OPERATOR", typeCode: "CABIN", allowed: true},
		{provider: "AIR_OPERATOR", typeCode: "FOLLOW_UP", allowed: true},
		{provider: "UNKNOWN_PROVIDER", typeCode: "RAMP_INSPECTION", allowed: false},
		{provider: "ATO", typeCode: "RAMP_INSPECTION", allowed: false},
		{provider: "AERODROME_OPERATOR", typeCode: "FOLLOW_UP", allowed: true},
		{provider: "FUEL_PROVIDER", typeCode: "PERIODIC_SURVEILLANCE", allowed: true},
	}
	for _, testCase := range tests {
		if got := canonicalAuditTypeAllowedForProvider(testCase.provider, testCase.typeCode); got != testCase.allowed {
			t.Fatalf("provider %q type %q allowed = %t; want %t", testCase.provider, testCase.typeCode, got, testCase.allowed)
		}
	}
}
