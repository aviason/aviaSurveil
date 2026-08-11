package provider

import "testing"

func TestSelectedProviderBoundary(t *testing.T) {
	if SelectedLibrary != "github.com/zitadel/oidc/v3" {
		t.Fatalf("selected library = %q", SelectedLibrary)
	}
	if SelectedVersion != "v3.47.5" {
		t.Fatalf("selected version = %q", SelectedVersion)
	}
	boundary := NewProtocolBoundary()
	for name, value := range map[string]string{
		"authorization": boundary.AuthorizationEndpoint,
		"token":         boundary.TokenEndpoint,
		"jwks":          boundary.JWKSURI,
		"logout":        boundary.EndSessionEndpoint,
	} {
		if value == "" {
			t.Errorf("%s endpoint is empty", name)
		}
	}
}
