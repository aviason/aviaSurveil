package agademoworkspace

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

func TestLifecycleProjectionPreservesEmptyCollections(t *testing.T) {
	aggregate := LifecycleAggregate{}
	projection := ProjectLifecycle(aggregate, identity.Principal{})
	caa := ProjectCAALifecycle(aggregate, identity.Principal{})

	collections := []struct {
		name  string
		value any
	}{
		{"questions", projection.Questions},
		{"responses", projection.Responses},
		{"potentialFindings", projection.PotentialFindings},
		{"findings", projection.Findings},
		{"capRevisions", projection.CAPRevisions},
		{"evidenceVersions", projection.EvidenceVersions},
		{"verificationDecisions", projection.VerificationDecisions},
		{"caa.roleHistory", caa.RoleHistory},
	}
	for _, collection := range collections {
		if reflect.ValueOf(collection.value).IsNil() {
			t.Fatalf("%s must be an empty, non-nil collection", collection.name)
		}
	}

	encoded, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"questions":[]`, `"responses":[]`, `"potentialFindings":[]`, `"findings":[]`, `"capRevisions":[]`, `"evidenceVersions":[]`, `"verificationDecisions":[]`} {
		if !strings.Contains(string(encoded), expected) {
			t.Fatalf("projection JSON omitted empty array %s: %s", expected, encoded)
		}
	}
}

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
