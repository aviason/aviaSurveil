package identity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProviderAdminBoundaryIsProviderNeutral(t *testing.T) {
	var client ProviderAdmin = (*FirstPartyAdminClient)(nil)
	if client == nil {
		t.Fatal("first-party adapter must satisfy the provider-neutral boundary")
	}
	var _ ProviderAdmin = (*FirstPartyAdminClient)(nil)
}

func TestFirstPartyAdminBootstrapClientUsesOnlyBootstrapSecretFile(t *testing.T) {
	secretPath := filepath.Join(t.TempDir(), "bootstrap-secret")
	if err := os.WriteFile(secretPath, []byte("0123456789abcdef0123456789abcdef"), 0o400); err != nil {
		t.Fatalf("write bootstrap secret: %v", err)
	}
	client, err := NewFirstPartyAdminClient(FirstPartyAdminConfig{
		BaseURL:             "http://auth:8081",
		BootstrapSecretFile: secretPath,
		Target:              "namibia/demo",
	})
	if err != nil || client == nil {
		t.Fatalf("bootstrap client = %v, err = %v", client, err)
	}
	if _, err := NewFirstPartyAdminClient(FirstPartyAdminConfig{
		BaseURL:             "http://auth:8081",
		SecretFile:          secretPath,
		BootstrapSecretFile: secretPath,
	}); err == nil {
		t.Fatal("runtime and bootstrap secret files must not be combined")
	}
}
