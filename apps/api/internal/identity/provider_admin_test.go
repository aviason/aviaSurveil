package identity

import "testing"

func TestProviderAdminBoundaryIsProviderNeutral(t *testing.T) {
	var client ProviderAdmin = (*KeycloakAdminClient)(nil)
	if client == nil {
		t.Fatal("Keycloak baseline adapter must satisfy the provider-neutral boundary")
	}
	var _ ProviderAdmin = (*KeycloakAdminClient)(nil)
}
