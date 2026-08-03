package main

import (
	"strings"
	"testing"
)

func TestNormalAPIAuthenticationProbesCoverOnlyTheExistingAuthSurface(t *testing.T) {
	t.Parallel()

	probes := normalAPIAuthenticationProbes()
	joined := ""
	for _, probe := range probes {
		if strings.TrimSpace(probe.label) == "" || strings.TrimSpace(probe.statement) == "" {
			t.Fatal("normal API probe label and statement are required")
		}
		joined += "\n" + probe.statement
	}
	for _, required := range []string{
		"set_config('avia.traceparent'",
		"session_references",
		"desired_membership_versions",
		"desired_membership_sync",
		"identity_references",
		"user_profiles",
		"caa_department_memberships",
		"caa_department_status_facts",
		"caa_organizational_unit_status_facts",
		"caa_organizational_units",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("normal API auth-surface probe is missing %q", required)
		}
	}
	if strings.Contains(joined, "preprod_aga_demo") ||
		strings.Contains(strings.ToUpper(joined), " INSERT ") ||
		strings.Contains(strings.ToUpper(joined), " DELETE ") {
		t.Fatal("normal API positive probes must not access the overlay or create/delete rows")
	}
}
