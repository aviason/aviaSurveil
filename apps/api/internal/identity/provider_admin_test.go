package identity

import "testing"

func TestProviderAdminBoundaryIsProviderNeutral(t *testing.T) {
	var client ProviderAdmin = (*FirstPartyAdminClient)(nil)
	if client == nil {
		t.Fatal("first-party adapter must satisfy the provider-neutral boundary")
	}
	var _ ProviderAdmin = (*FirstPartyAdminClient)(nil)
}
