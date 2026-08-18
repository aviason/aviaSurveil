package qualificationbootstrap

import "testing"

func TestPriorAuditHistoryTargetPolicy(t *testing.T) {
	tests := map[string]bool{
		"namibia/dev":     true,
		"namibia/demo":    true,
		"namibia/preprod": false,
		"namibia/prod":    false,
		"other/dev":       false,
	}
	for target, want := range tests {
		if got := isPriorAuditHistoryTarget(target); got != want {
			t.Errorf("isPriorAuditHistoryTarget(%q) = %t, want %t", target, got, want)
		}
	}
}

func TestPriorAuditOrganizationName(t *testing.T) {
	if got := priorAuditOrganizationName("namibia/dev"); got != "Namibia Dev AGA Qualification Operator" {
		t.Fatalf("dev organization name = %q", got)
	}
	if got := priorAuditOrganizationName("namibia/demo"); got != "Namibia AGA Qualification Operator" {
		t.Fatalf("demo organization name = %q", got)
	}
}
