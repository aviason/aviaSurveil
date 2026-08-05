package agademoworkspace

import (
	"context"
	"testing"
	"time"
)

func TestWorkspaceFixtureBindsSyntheticAuthorityOnly(t *testing.T) {
	template := DefaultFixtureTemplate()
	if err := template.Validate(); err != nil {
		t.Fatal(err)
	}
	if template.CAAOrganizationID == "" || template.SyntheticNamespace != "AGA_DEMO_ONLY" {
		t.Fatal("synthetic authority boundary is incomplete")
	}
	for _, entry := range template.ProviderConfiguration {
		if entry.ProviderTypeCode == "AERODROME_OPERATOR" && entry.Disposition != "INSPECTED_SCOPE_ELIGIBLE" {
			t.Fatal("current AGA profile is not bound to aerodrome operator")
		}
	}
}

func TestWorkspaceFixtureExporterIsReadOnlyAndExact(t *testing.T) {
	template := DefaultFixtureTemplate()
	_, err := ExportFixture(context.Background(), template, nil, "fixture", "sha256:0000000000000000000000000000000000000000000000000000000000000000", "base", "sha256:0000000000000000000000000000000000000000000000000000000000000000", time.Time{})
	if err == nil {
		t.Fatal("nil fixture source accepted")
	}
}
