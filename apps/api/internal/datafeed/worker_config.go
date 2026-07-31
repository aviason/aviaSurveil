package datafeed

import (
	"crypto/aes"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// WorkerConfig contains references to mounted secrets only. The worker never
// accepts key material, certificates, or payload values in environment text.
type WorkerConfig struct {
	TenantID             string
	OwningOrganizationID string
	ReplayID             string
	PayloadKeyFile       string
	BatchLimit           int
	MTLS                 MTLSClientConfig
}

// ReplayWorkerConfig binds a one-shot replay worker to the already persisted
// immutable replay run. It intentionally carries no event selector or source
// cut input; those are authorized and recorded before this process starts.
type ReplayWorkerConfig struct {
	WorkerConfig
	ReplayRunID string
}

// LoadPayloadKeyFile reads raw AES-256 material from a mounted secret file.
// It intentionally does not trim, base64-decode, or log secret bytes.
func LoadPayloadKeyFile(path string) ([]byte, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mounted datafeed payload key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("mounted datafeed payload key must contain exactly 32 bytes")
	}
	if _, err := aes.NewCipher(key); err != nil {
		return nil, fmt.Errorf("mounted datafeed payload key: %w", err)
	}
	return key, nil
}

// LoadWorkerConfig is deliberately separate from the API runtime settings:
// the feed worker has no fallback endpoint, scope, or development bypass.
func LoadWorkerConfig(lookup func(string) (string, bool)) (WorkerConfig, error) {
	value := func(key string) string {
		if raw, ok := lookup(key); ok {
			return strings.TrimSpace(raw)
		}
		return ""
	}
	config := WorkerConfig{
		TenantID:             value("AVIA_DATA_FEED_TENANT_ID"),
		OwningOrganizationID: value("AVIA_DATA_FEED_OWNING_ORGANIZATION_ID"),
		ReplayID:             value("AVIA_DATA_FEED_REPLAY_ID"),
		PayloadKeyFile:       value("AVIA_DATA_FEED_PAYLOAD_KEY_FILE"),
		BatchLimit:           maxBatchItems,
		MTLS: MTLSClientConfig{
			Endpoint:               value("AVIA_DATA_FEED_ENDPOINT"),
			CABundleFile:           value("AVIA_DATA_FEED_CA_BUNDLE_FILE"),
			ApprovedCABundleSHA256: value("AVIA_DATA_FEED_CA_BUNDLE_SHA256"),
			RevocationListFile:     value("AVIA_DATA_FEED_REVOCATION_LIST_FILE"),
			ClientCertificateFile:  value("AVIA_DATA_FEED_CLIENT_CERTIFICATE_FILE"),
			ClientPrivateKeyFile:   value("AVIA_DATA_FEED_CLIENT_PRIVATE_KEY_FILE"),
			ExpectedClientSAN:      value("AVIA_DATA_FEED_EXPECTED_CLIENT_SAN"),
		},
	}
	for _, required := range []struct{ name, value string }{
		{"AVIA_DATA_FEED_TENANT_ID", config.TenantID},
		{"AVIA_DATA_FEED_OWNING_ORGANIZATION_ID", config.OwningOrganizationID},
		{"AVIA_DATA_FEED_REPLAY_ID", config.ReplayID},
		{"AVIA_DATA_FEED_PAYLOAD_KEY_FILE", config.PayloadKeyFile},
		{"AVIA_DATA_FEED_ENDPOINT", config.MTLS.Endpoint},
		{"AVIA_DATA_FEED_CA_BUNDLE_FILE", config.MTLS.CABundleFile},
		{"AVIA_DATA_FEED_CA_BUNDLE_SHA256", config.MTLS.ApprovedCABundleSHA256},
		{"AVIA_DATA_FEED_REVOCATION_LIST_FILE", config.MTLS.RevocationListFile},
		{"AVIA_DATA_FEED_CLIENT_CERTIFICATE_FILE", config.MTLS.ClientCertificateFile},
		{"AVIA_DATA_FEED_CLIENT_PRIVATE_KEY_FILE", config.MTLS.ClientPrivateKeyFile},
		{"AVIA_DATA_FEED_EXPECTED_CLIENT_SAN", config.MTLS.ExpectedClientSAN},
	} {
		if required.value == "" {
			return WorkerConfig{}, fmt.Errorf("%s is required", required.name)
		}
	}
	endpoint, err := url.Parse(config.MTLS.Endpoint)
	if err != nil || endpoint.Scheme != "https" || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || endpoint.Path != "/v3/aviasurveil/event-batches" {
		return WorkerConfig{}, fmt.Errorf("datafeed publisher requires an approved https endpoint")
	}
	if !isSHA256(config.MTLS.ApprovedCABundleSHA256) {
		return WorkerConfig{}, fmt.Errorf("AVIA_DATA_FEED_CA_BUNDLE_SHA256 must be a SHA-256 digest")
	}
	approvedSAN := "urn:aviacore:source:aviasurveil360:tenant:" + config.TenantID
	if config.MTLS.ExpectedClientSAN != approvedSAN {
		return WorkerConfig{}, fmt.Errorf("AVIA_DATA_FEED_EXPECTED_CLIENT_SAN must bind the configured tenant/source SAN")
	}
	return config, nil
}

func LoadReplayWorkerConfig(lookup func(string) (string, bool)) (ReplayWorkerConfig, error) {
	config, err := LoadWorkerConfig(lookup)
	if err != nil {
		return ReplayWorkerConfig{}, err
	}
	runID := ""
	if raw, ok := lookup("AVIA_DATA_FEED_REPLAY_RUN_ID"); ok {
		runID = strings.TrimSpace(raw)
	}
	if !validReplayUUID(runID) {
		return ReplayWorkerConfig{}, fmt.Errorf("AVIA_DATA_FEED_REPLAY_RUN_ID must be a UUID")
	}
	if config.ReplayID != runID {
		return ReplayWorkerConfig{}, fmt.Errorf("AVIA_DATA_FEED_REPLAY_ID must match AVIA_DATA_FEED_REPLAY_RUN_ID")
	}
	return ReplayWorkerConfig{WorkerConfig: config, ReplayRunID: runID}, nil
}
