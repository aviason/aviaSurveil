package agademoworkspace

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLifecycleProjectionKeepsCAAHistoryOutOfThePublicShape(t *testing.T) {
	aggregate, _, lead, auditee, _, finding := findingFixture(t, true, false, false)
	submitCAP(t, &aggregate, auditee, finding, "projection root cause")
	reviewCAPForTest(t, &aggregate, lead, finding, "ACCEPT")
	public := ProjectLifecycle(aggregate, auditee)
	caa := ProjectCAALifecycle(aggregate, lead)
	publicBytes, err := json.Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	caaBytes, err := json.Marshal(caa)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(publicBytes), "internalCaaNote") || strings.Contains(string(publicBytes), "subjectId") {
		t.Fatalf("public projection contains CAA-only fields: %s", publicBytes)
	}
	if !strings.Contains(string(caaBytes), "internalCaaNote") || !strings.Contains(string(caaBytes), "recommendationId") || !strings.Contains(string(caaBytes), "roleHistory") {
		t.Fatalf("CAA projection omitted audit fields: %s", caaBytes)
	}
}
