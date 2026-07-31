package datafeed

import (
	"os"
	"strings"
	"testing"
)

func TestLoadWorkerConfigRequiresMountedSecretsAndExplicitScope(t *testing.T) {
	values := map[string]string{
		"AVIA_DATA_FEED_TENANT_ID":               "tenant-ncaa",
		"AVIA_DATA_FEED_OWNING_ORGANIZATION_ID":  "org-ncaa",
		"AVIA_DATA_FEED_REPLAY_ID":               "task5-local-replay",
		"AVIA_DATA_FEED_ENDPOINT":                "https://receiver.example/v3/aviasurveil/event-batches",
		"AVIA_DATA_FEED_CA_BUNDLE_FILE":          "/run/secrets/ca.pem",
		"AVIA_DATA_FEED_CA_BUNDLE_SHA256":        strings.Repeat("a", 64),
		"AVIA_DATA_FEED_REVOCATION_LIST_FILE":    "/run/secrets/ca.crl",
		"AVIA_DATA_FEED_CLIENT_CERTIFICATE_FILE": "/run/secrets/client.pem",
		"AVIA_DATA_FEED_CLIENT_PRIVATE_KEY_FILE": "/run/secrets/client.key",
		"AVIA_DATA_FEED_EXPECTED_CLIENT_SAN":     "urn:aviacore:source:aviasurveil360:tenant:tenant-ncaa",
		"AVIA_DATA_FEED_PAYLOAD_KEY_FILE":        "/run/secrets/payload-key",
	}
	config, err := LoadWorkerConfig(mapLookup(values))
	if err != nil {
		t.Fatalf("load worker config: %v", err)
	}
	if config.TenantID != "tenant-ncaa" || config.OwningOrganizationID != "org-ncaa" || config.BatchLimit != 100 {
		t.Fatalf("worker config = %+v", config)
	}
	delete(values, "AVIA_DATA_FEED_PAYLOAD_KEY_FILE")
	if _, err := LoadWorkerConfig(mapLookup(values)); err == nil || !strings.Contains(err.Error(), "AVIA_DATA_FEED_PAYLOAD_KEY_FILE") {
		t.Fatalf("missing payload key file error = %v", err)
	}
}

func TestLoadPayloadKeyFileAcceptsOnlyExactAESKeyMaterial(t *testing.T) {
	file := t.TempDir() + "/payload-key"
	if err := os.WriteFile(file, testEncryptionKey, 0o600); err != nil {
		t.Fatal(err)
	}
	key, err := LoadPayloadKeyFile(file)
	if err != nil || string(key) != string(testEncryptionKey) {
		t.Fatalf("key=%q err=%v", key, err)
	}
	if err := os.WriteFile(file, []byte("short"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPayloadKeyFile(file); err == nil {
		t.Fatal("short payload key was accepted")
	}
}

func TestLoadWorkerConfigRejectsUnsafePublisherSettings(t *testing.T) {
	values := map[string]string{
		"AVIA_DATA_FEED_TENANT_ID":               "tenant-ncaa",
		"AVIA_DATA_FEED_OWNING_ORGANIZATION_ID":  "org-ncaa",
		"AVIA_DATA_FEED_REPLAY_ID":               "task5-local-replay",
		"AVIA_DATA_FEED_ENDPOINT":                "http://receiver.example/v3/aviasurveil/event-batches",
		"AVIA_DATA_FEED_CA_BUNDLE_FILE":          "/run/secrets/ca.pem",
		"AVIA_DATA_FEED_CA_BUNDLE_SHA256":        strings.Repeat("a", 64),
		"AVIA_DATA_FEED_REVOCATION_LIST_FILE":    "/run/secrets/ca.crl",
		"AVIA_DATA_FEED_CLIENT_CERTIFICATE_FILE": "/run/secrets/client.pem",
		"AVIA_DATA_FEED_CLIENT_PRIVATE_KEY_FILE": "/run/secrets/client.key",
		"AVIA_DATA_FEED_EXPECTED_CLIENT_SAN":     "urn:aviacore:source:aviasurveil360:tenant:tenant-ncaa",
		"AVIA_DATA_FEED_PAYLOAD_KEY_FILE":        "/run/secrets/payload-key",
	}
	if _, err := LoadWorkerConfig(mapLookup(values)); err == nil || !strings.Contains(err.Error(), "approved https endpoint") {
		t.Fatalf("unsafe endpoint error = %v", err)
	}
	values["AVIA_DATA_FEED_ENDPOINT"] = "https://receiver.example/v3/aviasurveil/event-batches"
	values["AVIA_DATA_FEED_EXPECTED_CLIENT_SAN"] = "urn:aviacore:source:aviasurveil360:tenant:other-tenant"
	if _, err := LoadWorkerConfig(mapLookup(values)); err == nil || !strings.Contains(err.Error(), "SAN") {
		t.Fatalf("mismatched SAN error = %v", err)
	}
}

func TestLoadReplayWorkerConfigRequiresTheImmutableRunIdentityToMatchTransportReplayID(t *testing.T) {
	values := map[string]string{
		"AVIA_DATA_FEED_TENANT_ID":               "tenant-ncaa",
		"AVIA_DATA_FEED_OWNING_ORGANIZATION_ID":  "org-ncaa",
		"AVIA_DATA_FEED_REPLAY_ID":               "10000000-0000-4000-8000-000000000091",
		"AVIA_DATA_FEED_REPLAY_RUN_ID":           "10000000-0000-4000-8000-000000000091",
		"AVIA_DATA_FEED_ENDPOINT":                "https://receiver.example/v3/aviasurveil/event-batches",
		"AVIA_DATA_FEED_CA_BUNDLE_FILE":          "/run/secrets/ca.pem",
		"AVIA_DATA_FEED_CA_BUNDLE_SHA256":        strings.Repeat("a", 64),
		"AVIA_DATA_FEED_REVOCATION_LIST_FILE":    "/run/secrets/ca.crl",
		"AVIA_DATA_FEED_CLIENT_CERTIFICATE_FILE": "/run/secrets/client.pem",
		"AVIA_DATA_FEED_CLIENT_PRIVATE_KEY_FILE": "/run/secrets/client.key",
		"AVIA_DATA_FEED_EXPECTED_CLIENT_SAN":     "urn:aviacore:source:aviasurveil360:tenant:tenant-ncaa",
		"AVIA_DATA_FEED_PAYLOAD_KEY_FILE":        "/run/secrets/payload-key",
	}
	config, err := LoadReplayWorkerConfig(mapLookup(values))
	if err != nil || config.ReplayRunID != values["AVIA_DATA_FEED_REPLAY_RUN_ID"] || config.ReplayID != config.ReplayRunID {
		t.Fatalf("replay worker config=%+v err=%v", config, err)
	}
	values["AVIA_DATA_FEED_REPLAY_RUN_ID"] = "10000000-0000-4000-8000-000000000092"
	if _, err := LoadReplayWorkerConfig(mapLookup(values)); err == nil || !strings.Contains(err.Error(), "match") {
		t.Fatalf("mismatched replay identities error=%v", err)
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) { value, ok := values[key]; return value, ok }
}
