package config

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/netpolicy"
)

type LookupEnv func(string) (string, bool)

type runtimeRequirements struct {
	objectStore bool
	scanner     bool
	renderer    bool
	oidc        bool
}

var (
	allRuntimeRequirements = runtimeRequirements{
		objectStore: true,
		scanner:     true,
		renderer:    true,
		oidc:        true,
	}
	apiRuntimeRequirements = runtimeRequirements{
		objectStore: true,
		scanner:     true,
		oidc:        true,
	}
)

type Settings struct {
	Environment                 string
	DatabaseURL                 string
	HTTPAddress                 string
	WorkerInterval              time.Duration
	TestPrincipal               string
	TestSession                 string
	DevSessionSecret            string
	OIDCIssuerURL               string
	OIDCDiscoveryURL            string
	OIDCDiscoveryPrivateNetwork bool
	OIDCClientID                string
	OIDCClientSecret            string
	OIDCRedirectURL             string
	KeycloakAdminURL            string
	KeycloakRealm               string
	KeycloakServiceClientID     string
	KeycloakServiceClientSecret string
	SessionEncryptionKey        []byte
	SessionIdleDuration         time.Duration
	SessionAbsoluteDuration     time.Duration
	CookieSecure                bool
	CanonicalSeed               bool
	CanonicalTestProfile        bool
	CanonicalTestToken          string
	ObjectStoreEndpoint         string
	ObjectStorePublicEndpoint   string
	ObjectStoreAccessKey        string
	ObjectStoreSecretKey        string
	ObjectStoreTLS              bool
	ObjectStorePublicTLS        bool
	ObjectStorePrivateNetwork   bool
	ObjectStoreRegion           string
	ObjectStoreCORSOrigins      []string
	QuarantineBucket            string
	CanonicalBucket             string
	AttachmentBucket            string
	DocumentBucket              string
	AllowServerManagedCORS      bool
	ScannerMode                 string
	ClamAVAddress               string
	ClamAVMaximumSignatureAge   time.Duration
	GotenbergURL                string
	GotenbergTimeout            time.Duration
	GotenbergRendererHash       string
	SMTPAddress                 string
	SMTPFrom                    string
	SMTPUsername                string
	SMTPPassword                string
	SMTPTimeout                 time.Duration
	SMTPPrivateNetwork          bool
	IdentityHealthURL           string
	GotenbergHealthURL          string
	SMTPHealthAddress           string
	RuntimeHealthTimeout        time.Duration
	OTLPHTTPEndpoint            string
}

func Load(lookup LookupEnv) (Settings, error) {
	return load(lookup, allRuntimeRequirements)
}

func LoadAPI(lookup LookupEnv) (Settings, error) {
	return load(lookup, apiRuntimeRequirements)
}

func LoadScheduler(lookup LookupEnv) (Settings, error) {
	return load(lookup, runtimeRequirements{})
}

func load(lookup LookupEnv, requirements runtimeRequirements) (Settings, error) {
	environment := valueOrDefault(lookup, "AVIA_ENVIRONMENT", "development")
	settings := Settings{
		Environment:             environment,
		DatabaseURL:             value(lookup, "AVIA_DATABASE_URL"),
		HTTPAddress:             valueOrDefault(lookup, "AVIA_HTTP_ADDRESS", ":8080"),
		TestPrincipal:           value(lookup, "AVIA_TEST_PRINCIPAL"),
		TestSession:             value(lookup, "AVIA_TEST_SESSION"),
		DevSessionSecret:        value(lookup, "AVIA_DEV_SESSION_SECRET"),
		OIDCIssuerURL:           value(lookup, "AVIA_OIDC_ISSUER_URL"),
		OIDCDiscoveryURL:        value(lookup, "AVIA_OIDC_DISCOVERY_URL"),
		OIDCClientID:            value(lookup, "AVIA_OIDC_CLIENT_ID"),
		OIDCClientSecret:        value(lookup, "AVIA_OIDC_CLIENT_SECRET"),
		OIDCRedirectURL:         value(lookup, "AVIA_OIDC_REDIRECT_URL"),
		KeycloakAdminURL:        value(lookup, "AVIA_KEYCLOAK_ADMIN_URL"),
		KeycloakRealm:           value(lookup, "AVIA_KEYCLOAK_REALM"),
		KeycloakServiceClientID: value(lookup, "AVIA_KEYCLOAK_SERVICE_CLIENT_ID"),
		KeycloakServiceClientSecret: value(
			lookup,
			"AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET",
		),
		SessionIdleDuration:       30 * time.Minute,
		SessionAbsoluteDuration:   8 * time.Hour,
		CookieSecure:              true,
		CanonicalTestToken:        value(lookup, "AVIA_CANONICAL_TEST_TOKEN"),
		ObjectStoreEndpoint:       value(lookup, "AVIA_OBJECT_STORE_ENDPOINT"),
		ObjectStorePublicEndpoint: value(lookup, "AVIA_OBJECT_STORE_PUBLIC_ENDPOINT"),
		ObjectStoreAccessKey:      value(lookup, "AVIA_OBJECT_STORE_ACCESS_KEY"),
		ObjectStoreSecretKey:      value(lookup, "AVIA_OBJECT_STORE_SECRET_KEY"),
		ObjectStoreRegion:         value(lookup, "AVIA_OBJECT_STORE_REGION"),
		ObjectStoreCORSOrigins:    commaValues(value(lookup, "AVIA_OBJECT_STORE_CORS_ORIGINS")),
		QuarantineBucket:          valueOrDefault(lookup, "AVIA_OBJECT_STORE_QUARANTINE_BUCKET", "evidence-quarantine"),
		CanonicalBucket:           valueOrDefault(lookup, "AVIA_OBJECT_STORE_CANONICAL_BUCKET", "evidence-clean"),
		AttachmentBucket:          valueOrDefault(lookup, "AVIA_OBJECT_STORE_ATTACHMENT_BUCKET", "inspection-attachments"),
		DocumentBucket:            valueOrDefault(lookup, "AVIA_OBJECT_STORE_DOCUMENT_BUCKET", "generated-documents"),
		ScannerMode:               value(lookup, "AVIA_SCANNER_MODE"),
		ClamAVAddress:             value(lookup, "AVIA_CLAMAV_ADDRESS"),
		GotenbergURL:              value(lookup, "AVIA_GOTENBERG_URL"),
		GotenbergRendererHash:     value(lookup, "AVIA_GOTENBERG_RENDERER_HASH"),
		SMTPAddress:               value(lookup, "AVIA_SMTP_ADDRESS"),
		SMTPFrom:                  value(lookup, "AVIA_SMTP_FROM"),
		SMTPUsername:              value(lookup, "AVIA_SMTP_USERNAME"),
		SMTPPassword:              value(lookup, "AVIA_SMTP_PASSWORD"),
		IdentityHealthURL:         value(lookup, "AVIA_IDENTITY_HEALTH_URL"),
		GotenbergHealthURL:        value(lookup, "AVIA_GOTENBERG_HEALTH_URL"),
		SMTPHealthAddress:         value(lookup, "AVIA_SMTP_HEALTH_ADDRESS"),
		OTLPHTTPEndpoint:          value(lookup, "AVIA_OTEL_EXPORTER_OTLP_ENDPOINT"),
	}
	if settings.OIDCDiscoveryURL == "" {
		settings.OIDCDiscoveryURL = settings.OIDCIssuerURL
	}
	if settings.ObjectStorePublicEndpoint == "" && settings.Environment != "production" {
		settings.ObjectStorePublicEndpoint = settings.ObjectStoreEndpoint
	}
	canonicalProfile, err := parseBoolean(lookup, "AVIA_ENABLE_CANONICAL_TEST_PROFILE", false)
	if err != nil {
		return Settings{}, err
	}
	canonicalSeed, err := parseBoolean(lookup, "AVIA_ENABLE_CANONICAL_SEED", false)
	if err != nil {
		return Settings{}, err
	}
	settings.CanonicalSeed = canonicalSeed
	settings.CanonicalTestProfile = canonicalProfile
	objectStoreTLS, err := parseBoolean(lookup, "AVIA_OBJECT_STORE_TLS", false)
	if err != nil {
		return Settings{}, err
	}
	settings.ObjectStoreTLS = objectStoreTLS
	objectStorePublicTLS, err := parseBoolean(
		lookup,
		"AVIA_OBJECT_STORE_PUBLIC_TLS",
		objectStoreTLS,
	)
	if err != nil {
		return Settings{}, err
	}
	settings.ObjectStorePublicTLS = objectStorePublicTLS
	objectStorePrivateNetwork, err := parseBoolean(
		lookup,
		"AVIA_OBJECT_STORE_PRIVATE_NETWORK",
		false,
	)
	if err != nil {
		return Settings{}, err
	}
	settings.ObjectStorePrivateNetwork = objectStorePrivateNetwork
	clamAVMaximumSignatureAge, err := time.ParseDuration(
		valueOrDefault(lookup, "AVIA_CLAMAV_MAX_SIGNATURE_AGE", "48h"),
	)
	if err != nil || clamAVMaximumSignatureAge <= 0 {
		return Settings{}, fmt.Errorf("AVIA_CLAMAV_MAX_SIGNATURE_AGE must be a positive duration")
	}
	settings.ClamAVMaximumSignatureAge = clamAVMaximumSignatureAge
	gotenbergTimeout, err := time.ParseDuration(
		valueOrDefault(lookup, "AVIA_GOTENBERG_TIMEOUT", "30s"),
	)
	if err != nil || gotenbergTimeout <= 0 || gotenbergTimeout > 2*time.Minute {
		return Settings{}, fmt.Errorf(
			"AVIA_GOTENBERG_TIMEOUT must be positive and no greater than two minutes",
		)
	}
	settings.GotenbergTimeout = gotenbergTimeout
	smtpTimeout, err := time.ParseDuration(
		valueOrDefault(lookup, "AVIA_SMTP_TIMEOUT", "10s"),
	)
	if err != nil || smtpTimeout <= 0 || smtpTimeout > time.Minute {
		return Settings{}, fmt.Errorf(
			"AVIA_SMTP_TIMEOUT must be positive and no greater than one minute",
		)
	}
	settings.SMTPTimeout = smtpTimeout
	smtpPrivateNetwork, err := parseBoolean(
		lookup,
		"AVIA_SMTP_PRIVATE_NETWORK",
		false,
	)
	if err != nil {
		return Settings{}, err
	}
	settings.SMTPPrivateNetwork = smtpPrivateNetwork
	oidcDiscoveryPrivateNetwork, err := parseBoolean(
		lookup,
		"AVIA_OIDC_DISCOVERY_PRIVATE_NETWORK",
		false,
	)
	if err != nil {
		return Settings{}, err
	}
	settings.OIDCDiscoveryPrivateNetwork = oidcDiscoveryPrivateNetwork
	runtimeHealthTimeout, err := time.ParseDuration(
		valueOrDefault(lookup, "AVIA_RUNTIME_HEALTH_TIMEOUT", "1s"),
	)
	if err != nil || runtimeHealthTimeout <= 0 ||
		runtimeHealthTimeout > 5*time.Second {
		return Settings{}, fmt.Errorf(
			"AVIA_RUNTIME_HEALTH_TIMEOUT must be positive and no greater than five seconds",
		)
	}
	settings.RuntimeHealthTimeout = runtimeHealthTimeout
	serverManagedCORS, err := parseBoolean(
		lookup,
		"AVIA_OBJECT_STORE_SERVER_MANAGED_CORS",
		false,
	)
	if err != nil {
		return Settings{}, err
	}
	settings.AllowServerManagedCORS = settings.Environment == "test" &&
		(settings.CanonicalSeed || serverManagedCORS)

	if settings.Environment == "production" {
		for _, key := range []string{"AVIA_TEST_PRINCIPAL", "AVIA_TEST_SESSION", "AVIA_DEV_SESSION_SECRET", "AVIA_ENABLE_CANONICAL_SEED", "AVIA_ENABLE_CANONICAL_TEST_PROFILE", "AVIA_CANONICAL_TEST_TOKEN", "AVIA_OBJECT_STORE_SERVER_MANAGED_CORS"} {
			if value(lookup, key) != "" {
				return Settings{}, fmt.Errorf("%s is forbidden in production", key)
			}
		}
	}
	if serverManagedCORS && settings.Environment != "test" {
		return Settings{}, fmt.Errorf(
			"AVIA_OBJECT_STORE_SERVER_MANAGED_CORS requires AVIA_ENVIRONMENT=test",
		)
	}
	if settings.CanonicalSeed && settings.Environment != "test" {
		return Settings{}, fmt.Errorf("AVIA_ENABLE_CANONICAL_SEED requires AVIA_ENVIRONMENT=test")
	}
	if settings.CanonicalTestProfile && settings.Environment != "test" {
		return Settings{}, fmt.Errorf("AVIA_ENABLE_CANONICAL_TEST_PROFILE requires AVIA_ENVIRONMENT=test")
	}
	if settings.CanonicalTestProfile && !settings.CanonicalSeed {
		return Settings{}, fmt.Errorf("AVIA_ENABLE_CANONICAL_TEST_PROFILE requires AVIA_ENABLE_CANONICAL_SEED=true")
	}
	if settings.CanonicalTestToken != "" && !settings.CanonicalTestProfile {
		return Settings{}, fmt.Errorf("AVIA_CANONICAL_TEST_TOKEN requires AVIA_ENABLE_CANONICAL_TEST_PROFILE=true")
	}
	if settings.CanonicalTestProfile && len(settings.CanonicalTestToken) < 16 {
		return Settings{}, fmt.Errorf("AVIA_CANONICAL_TEST_TOKEN is required and must contain at least 16 characters")
	}
	if settings.CanonicalSeed && settings.ScannerMode != "deterministic-test" {
		return Settings{}, fmt.Errorf("AVIA_SCANNER_MODE=deterministic-test is required by the canonical seed")
	}
	if settings.Environment == "production" && settings.ScannerMode == "deterministic-test" {
		return Settings{}, fmt.Errorf("AVIA_SCANNER_MODE=deterministic-test is forbidden in production")
	}
	if settings.Environment != "test" && (settings.TestPrincipal != "" || settings.TestSession != "") {
		return Settings{}, fmt.Errorf("AVIA_TEST_PRINCIPAL and AVIA_TEST_SESSION require AVIA_ENVIRONMENT=test")
	}
	if settings.Environment != "development" && settings.DevSessionSecret != "" {
		return Settings{}, fmt.Errorf("AVIA_DEV_SESSION_SECRET requires AVIA_ENVIRONMENT=development")
	}
	if settings.Environment == "test" && (settings.TestPrincipal == "") != (settings.TestSession == "") {
		return Settings{}, fmt.Errorf("AVIA_TEST_PRINCIPAL and AVIA_TEST_SESSION must be configured together")
	}
	if settings.DatabaseURL == "" {
		return Settings{}, fmt.Errorf("AVIA_DATABASE_URL is required")
	}
	if !contains([]string{"development", "test", "production"}, settings.Environment) {
		return Settings{}, fmt.Errorf("AVIA_ENVIRONMENT must be development, test, or production")
	}

	objectStoreConfigured := settings.ObjectStoreEndpoint != "" ||
		settings.ObjectStorePublicEndpoint != "" ||
		settings.ObjectStoreAccessKey != "" ||
		settings.ObjectStoreSecretKey != "" ||
		len(settings.ObjectStoreCORSOrigins) > 0
	if (settings.Environment == "production" && requirements.objectStore) ||
		settings.CanonicalSeed ||
		objectStoreConfigured {
		for _, entry := range []struct {
			name  string
			value any
		}{
			{name: "AVIA_OBJECT_STORE_ENDPOINT", value: settings.ObjectStoreEndpoint},
			{name: "AVIA_OBJECT_STORE_PUBLIC_ENDPOINT", value: settings.ObjectStorePublicEndpoint},
			{name: "AVIA_OBJECT_STORE_ACCESS_KEY", value: settings.ObjectStoreAccessKey},
			{name: "AVIA_OBJECT_STORE_SECRET_KEY", value: settings.ObjectStoreSecretKey},
			{name: "AVIA_OBJECT_STORE_CORS_ORIGINS", value: settings.ObjectStoreCORSOrigins},
		} {
			missing := entry.value == ""
			if values, ok := entry.value.([]string); ok {
				missing = len(values) == 0
			}
			if missing {
				return Settings{}, fmt.Errorf("%s is required when object storage is enabled", entry.name)
			}
		}
		buckets := []string{
			settings.QuarantineBucket,
			settings.CanonicalBucket,
			settings.AttachmentBucket,
			settings.DocumentBucket,
		}
		seenBuckets := make(map[string]struct{}, len(buckets))
		for _, bucket := range buckets {
			if _, exists := seenBuckets[bucket]; exists {
				return Settings{}, fmt.Errorf("all object-store buckets must be distinct")
			}
			seenBuckets[bucket] = struct{}{}
		}
		if settings.Environment == "production" &&
			!settings.ObjectStoreTLS &&
			!settings.ObjectStorePrivateNetwork {
			return Settings{}, fmt.Errorf(
				"plaintext object-store transport requires AVIA_OBJECT_STORE_PRIVATE_NETWORK=true",
			)
		}
		if settings.Environment == "production" && !settings.ObjectStorePublicTLS {
			return Settings{}, fmt.Errorf("AVIA_OBJECT_STORE_PUBLIC_TLS=true is required in production")
		}
	}

	if settings.Environment == "production" && requirements.scanner {
		if settings.ScannerMode != "clamav" {
			return Settings{}, fmt.Errorf("AVIA_SCANNER_MODE=clamav is required in production")
		}
		if settings.ClamAVAddress == "" {
			return Settings{}, fmt.Errorf("AVIA_CLAMAV_ADDRESS is required in production")
		}
	}

	gotenbergConfigured := settings.GotenbergURL != "" ||
		settings.GotenbergRendererHash != ""
	if (settings.Environment == "production" && requirements.renderer) ||
		gotenbergConfigured {
		if settings.GotenbergURL == "" {
			return Settings{}, fmt.Errorf(
				"AVIA_GOTENBERG_URL is required when document rendering is enabled",
			)
		}
		if !isSHA256(settings.GotenbergRendererHash) {
			return Settings{}, fmt.Errorf(
				"AVIA_GOTENBERG_RENDERER_HASH must be a sha256 digest",
			)
		}
		rendererURL, err := url.Parse(settings.GotenbergURL)
		if err != nil || rendererURL.Host == "" ||
			(rendererURL.Scheme != "http" && rendererURL.Scheme != "https") ||
			rendererURL.User != nil || rendererURL.RawQuery != "" ||
			rendererURL.Fragment != "" {
			return Settings{}, fmt.Errorf(
				"AVIA_GOTENBERG_URL must be an absolute HTTP(S) URL without credentials, query, or fragment",
			)
		}
	}

	smtpKeys := []struct {
		name  string
		value string
	}{
		{name: "AVIA_SMTP_ADDRESS", value: settings.SMTPAddress},
		{name: "AVIA_SMTP_FROM", value: settings.SMTPFrom},
		{name: "AVIA_SMTP_USERNAME", value: settings.SMTPUsername},
		{name: "AVIA_SMTP_PASSWORD", value: settings.SMTPPassword},
	}
	smtpConfigured := false
	for _, entry := range smtpKeys {
		if entry.value != "" {
			smtpConfigured = true
			break
		}
	}
	if smtpConfigured {
		for _, entry := range smtpKeys {
			if entry.value == "" {
				return Settings{}, fmt.Errorf(
					"%s is required when SMTP delivery is enabled",
					entry.name,
				)
			}
		}
		if _, _, err := net.SplitHostPort(settings.SMTPAddress); err != nil {
			return Settings{}, fmt.Errorf(
				"AVIA_SMTP_ADDRESS must contain host and port",
			)
		}
		from, err := mail.ParseAddress(settings.SMTPFrom)
		if err != nil || from.Address == "" {
			return Settings{}, fmt.Errorf(
				"AVIA_SMTP_FROM must be a valid email address",
			)
		}
		if !settings.SMTPPrivateNetwork {
			return Settings{}, fmt.Errorf(
				"plaintext SMTP transport requires AVIA_SMTP_PRIVATE_NETWORK=true",
			)
		}
	}

	oidcKeys := []struct {
		name  string
		value string
	}{
		{name: "AVIA_OIDC_ISSUER_URL", value: settings.OIDCIssuerURL},
		{name: "AVIA_OIDC_CLIENT_ID", value: settings.OIDCClientID},
		{name: "AVIA_OIDC_CLIENT_SECRET", value: settings.OIDCClientSecret},
		{name: "AVIA_OIDC_REDIRECT_URL", value: settings.OIDCRedirectURL},
		{name: "AVIA_SESSION_ENCRYPTION_KEY", value: value(lookup, "AVIA_SESSION_ENCRYPTION_KEY")},
	}
	oidcConfigured := false
	for _, entry := range oidcKeys {
		if entry.value != "" {
			oidcConfigured = true
			break
		}
	}
	if (settings.Environment == "production" && requirements.oidc) ||
		oidcConfigured {
		for _, entry := range oidcKeys {
			if entry.value == "" {
				return Settings{}, fmt.Errorf("%s is required when OIDC authentication is enabled", entry.name)
			}
		}
		key, err := base64.StdEncoding.DecodeString(value(lookup, "AVIA_SESSION_ENCRYPTION_KEY"))
		if err != nil || len(key) != 32 {
			return Settings{}, fmt.Errorf("AVIA_SESSION_ENCRYPTION_KEY must be base64 for exactly 32 bytes")
		}
		settings.SessionEncryptionKey = key
		issuerURL, err := url.Parse(settings.OIDCIssuerURL)
		if err != nil || issuerURL.Scheme == "" || issuerURL.Host == "" {
			return Settings{}, fmt.Errorf("AVIA_OIDC_ISSUER_URL must be an absolute URL")
		}
		discoveryURL, err := url.Parse(settings.OIDCDiscoveryURL)
		if err != nil || discoveryURL.Host == "" ||
			(discoveryURL.Scheme != "http" && discoveryURL.Scheme != "https") ||
			discoveryURL.User != nil || discoveryURL.RawQuery != "" ||
			discoveryURL.Fragment != "" {
			return Settings{}, fmt.Errorf(
				"AVIA_OIDC_DISCOVERY_URL must be an absolute HTTP(S) URL without credentials, query, or fragment",
			)
		}
		redirectURL, err := url.Parse(settings.OIDCRedirectURL)
		if err != nil || redirectURL.Scheme == "" || redirectURL.Host == "" {
			return Settings{}, fmt.Errorf("AVIA_OIDC_REDIRECT_URL must be an absolute URL")
		}
		if settings.Environment == "production" && (issuerURL.Scheme != "https" || redirectURL.Scheme != "https") {
			return Settings{}, fmt.Errorf("production OIDC issuer and redirect URLs must use HTTPS")
		}
		if settings.Environment == "production" &&
			discoveryURL.Scheme == "http" &&
			!settings.OIDCDiscoveryPrivateNetwork {
			return Settings{}, fmt.Errorf(
				"plaintext OIDC discovery requires AVIA_OIDC_DISCOVERY_PRIVATE_NETWORK=true",
			)
		}
	}

	keycloakAdminKeys := []struct {
		name  string
		value string
	}{
		{name: "AVIA_KEYCLOAK_ADMIN_URL", value: settings.KeycloakAdminURL},
		{name: "AVIA_KEYCLOAK_REALM", value: settings.KeycloakRealm},
		{name: "AVIA_KEYCLOAK_SERVICE_CLIENT_ID", value: settings.KeycloakServiceClientID},
		{name: "AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET", value: settings.KeycloakServiceClientSecret},
	}
	keycloakAdminConfigured := false
	for _, entry := range keycloakAdminKeys {
		if entry.value != "" {
			keycloakAdminConfigured = true
			break
		}
	}
	if keycloakAdminConfigured {
		for _, entry := range keycloakAdminKeys {
			if entry.value == "" {
				return Settings{}, fmt.Errorf(
					"%s is required when Keycloak administration is enabled",
					entry.name,
				)
			}
		}
		adminURL, err := url.Parse(settings.KeycloakAdminURL)
		if err != nil ||
			adminURL.Host == "" ||
			(adminURL.Scheme != "http" && adminURL.Scheme != "https") {
			return Settings{}, fmt.Errorf(
				"AVIA_KEYCLOAK_ADMIN_URL must be an absolute HTTP(S) URL",
			)
		}
	}

	for _, entry := range []struct {
		name  string
		value string
	}{
		{name: "AVIA_IDENTITY_HEALTH_URL", value: settings.IdentityHealthURL},
		{name: "AVIA_GOTENBERG_HEALTH_URL", value: settings.GotenbergHealthURL},
	} {
		if entry.value == "" {
			continue
		}
		healthURL, err := url.Parse(entry.value)
		if err != nil || healthURL.Host == "" ||
			(healthURL.Scheme != "http" && healthURL.Scheme != "https") ||
			healthURL.User != nil || healthURL.RawQuery != "" ||
			healthURL.Fragment != "" {
			return Settings{}, fmt.Errorf(
				"%s must be an absolute HTTP(S) URL without credentials, query, or fragment",
				entry.name,
			)
		}
	}
	if settings.SMTPHealthAddress != "" {
		if _, _, err := net.SplitHostPort(settings.SMTPHealthAddress); err != nil {
			return Settings{}, fmt.Errorf(
				"AVIA_SMTP_HEALTH_ADDRESS must contain host and port",
			)
		}
	}
	if settings.OTLPHTTPEndpoint != "" {
		endpoint, err := url.Parse(settings.OTLPHTTPEndpoint)
		if err != nil || endpoint.Scheme != "http" || endpoint.Host == "" ||
			endpoint.User != nil || endpoint.RawQuery != "" ||
			endpoint.Fragment != "" ||
			!netpolicy.IsPrivateHost(endpoint.Hostname()) {
			return Settings{}, fmt.Errorf(
				"AVIA_OTEL_EXPORTER_OTLP_ENDPOINT must be an absolute private HTTP URL without credentials, query, or fragment",
			)
		}
	}

	workerMilliseconds := valueOrDefault(lookup, "AVIA_WORKER_INTERVAL_MS", "1000")
	milliseconds, err := strconv.Atoi(workerMilliseconds)
	if err != nil || milliseconds < 50 {
		return Settings{}, fmt.Errorf("AVIA_WORKER_INTERVAL_MS must be an integer of at least 50")
	}
	settings.WorkerInterval = time.Duration(milliseconds) * time.Millisecond
	return settings, nil
}

func value(lookup LookupEnv, key string) string {
	if raw, ok := lookup(key); ok {
		return strings.TrimSpace(raw)
	}
	return ""
}

func valueOrDefault(lookup LookupEnv, key, fallback string) string {
	if resolved := value(lookup, key); resolved != "" {
		return resolved
	}
	return fallback
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func parseBoolean(lookup LookupEnv, key string, fallback bool) (bool, error) {
	raw := value(lookup, key)
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", key)
	}
	return parsed, nil
}

func commaValues(raw string) []string {
	if raw == "" {
		return nil
	}
	values := []string{}
	for _, candidate := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func isSHA256(value string) bool {
	if !strings.HasPrefix(value, "sha256:") ||
		len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
