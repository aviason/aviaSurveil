package agademoworkspace

import (
	"testing"

	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

func TestSimulationSetupScopesUseOnlyTheExplicitCAAOrganizationAlias(t *testing.T) {
	scopes := []preprod.ProviderScope{
		{OrganizationID: "AGA-DEMO-CAA", ProviderScopeID: "matching"},
		{OrganizationID: "AGA-DEMO-OTHER-ORG", ProviderScopeID: "other"},
	}

	for _, organization := range []string{"CAA", "AGA-DEMO-CAA"} {
		filtered := simulationSetupScopesForPrincipal(scopes, organization)
		if len(filtered) != 1 || filtered[0].ProviderScopeID != "matching" {
			t.Fatalf("organization %q selected %+v, want only the matching CAA scope", organization, filtered)
		}
	}

	if filtered := simulationSetupScopesForPrincipal(scopes, "UNRELATED-ORG"); len(filtered) != 0 {
		t.Fatalf("unrelated organization selected sealed scopes: %+v", filtered)
	}

	ambiguous := append(scopes, preprod.ProviderScope{OrganizationID: "CAA", ProviderScopeID: "second-caa-alias"})
	if filtered := simulationSetupScopesForPrincipal(ambiguous, "CAA"); len(filtered) != 2 {
		t.Fatalf("multiple CAA scopes were silently collapsed: %+v", filtered)
	}
}
