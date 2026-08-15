package config

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/netpolicy"
)

type LookupEnv func(string) (string, bool)

type runtimeRequirements struct {
	objectStore bool
	oidc        bool
	smtp        bool
}

var (
	allRuntimeRequirements = runtimeRequirements{
		objectStore: true,
		oidc:        true,
		smtp:        true,
	}
	apiRuntimeRequirements = runtimeRequirements{
		objectStore: true,
		oidc:        true,
	}
)

type Settings struct {
	Environment                 string
	DataEnabled                 bool
	DataMode                    string
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
	FirstPartyAdminURL          string
	FirstPartyAdminSecretFile   string
	SessionEncryptionKey        []byte
	SessionIdleDuration         time.Duration
	SessionAbsoluteDuration     time.Duration
	CookieSecure                bool
	CanonicalSeed               bool
	CanonicalTestProfile        bool
	CanonicalTestToken          string
	ObjectStoreEndpoint         string
	ObjectStoreMode             string
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
	SMTPAddress                 string
	SMTPFrom                    string
	SMTPUsername                string
	SMTPPassword                string
	SMTPTimeout                 time.Duration
	SMTPPrivateNetwork          bool
	SMTPTransport               string
	SMTPTLSServerName           string
	IdentityHealthURL           string
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

func LoadDatabaseRuntime(lookup LookupEnv) (Settings, error) {
	return load(lookup, runtimeRequirements{})
}

func load(lookup LookupEnv, requirements runtimeRequirements) (Settings, error) {
	environment := valueOrDefault(lookup, "AVIA_ENVIRONMENT", "development")
	cloudEnvironment := environment == "demo" || environment == "production"
	dataEnabled, err := parseDataMode(lookup)
	if err != nil {
		return Settings{}, err
	}
	dataMode := value(lookup, "AVIA_DATA_MODE")
	if dataEnabled {
		localCandidate := environment == "development" && dataMode == "local-candidate"
		directMTLS := environment == "production" && dataMode == "direct-mtls"
		if !localCandidate && !directMTLS {
			return Settings{}, fmt.Errorf("AVIA_DATA=1 requires AVIA_DATA_MODE=local-candidate in development or direct-mtls in production")
		}
	} else if dataMode != "" {
		return Settings{}, fmt.Errorf("AVIA_DATA_MODE requires AVIA_DATA=1")
	}
	if !dataEnabled {
		if err := rejectDisabledDataFeedConfiguration(lookup); err != nil {
			return Settings{}, err
		}
	}
	databaseURL, err := valueOrFile(lookup, "AVIA_DATABASE_URL")
	if err != nil {
		return Settings{}, err
	}
	oidcClientSecret, err := valueOrFile(lookup, "AVIA_OIDC_CLIENT_SECRET")
	if err != nil {
		return Settings{}, err
	}
	sessionEncryptionKey, err := valueOrFile(lookup, "AVIA_SESSION_ENCRYPTION_KEY")
	if err != nil {
		return Settings{}, err
	}
	objectStoreAccessKey, err := valueOrFile(lookup, "AVIA_OBJECT_STORE_ACCESS_KEY")
	if err != nil {
		return Settings{}, err
	}
	objectStoreSecretKey, err := valueOrFile(lookup, "AVIA_OBJECT_STORE_SECRET_KEY")
	if err != nil {
		return Settings{}, err
	}
	smtpPassword, err := valueOrFile(lookup, "AVIA_SMTP_PASSWORD")
	if err != nil {
		return Settings{}, err
	}
	settings := Settings{
		Environment:               environment,
		DataEnabled:               dataEnabled,
		DataMode:                  dataMode,
		DatabaseURL:               databaseURL,
		HTTPAddress:               valueOrDefault(lookup, "AVIA_HTTP_ADDRESS", ":8080"),
		TestPrincipal:             value(lookup, "AVIA_TEST_PRINCIPAL"),
		TestSession:               value(lookup, "AVIA_TEST_SESSION"),
		DevSessionSecret:          value(lookup, "AVIA_DEV_SESSION_SECRET"),
		OIDCIssuerURL:             value(lookup, "AVIA_OIDC_ISSUER_URL"),
		OIDCDiscoveryURL:          value(lookup, "AVIA_OIDC_DISCOVERY_URL"),
		OIDCClientID:              value(lookup, "AVIA_OIDC_CLIENT_ID"),
		OIDCClientSecret:          oidcClientSecret,
		OIDCRedirectURL:           value(lookup, "AVIA_OIDC_REDIRECT_URL"),
		FirstPartyAdminURL:        value(lookup, "AVIA_AUTH_ADMIN_URL"),
		FirstPartyAdminSecretFile: value(lookup, "AVIA_AUTH_ADMIN_SECRET_FILE"),
		SessionIdleDuration:       30 * time.Minute,
		SessionAbsoluteDuration:   8 * time.Hour,
		CookieSecure:              true,
		CanonicalTestToken:        value(lookup, "AVIA_CANONICAL_TEST_TOKEN"),
		ObjectStoreEndpoint:       value(lookup, "AVIA_OBJECT_STORE_ENDPOINT"),
		ObjectStoreMode:           value(lookup, "AVIA_OBJECT_STORE_MODE"),
		ObjectStorePublicEndpoint: value(lookup, "AVIA_OBJECT_STORE_PUBLIC_ENDPOINT"),
		ObjectStoreAccessKey:      objectStoreAccessKey,
		ObjectStoreSecretKey:      objectStoreSecretKey,
		ObjectStoreRegion:         value(lookup, "AVIA_OBJECT_STORE_REGION"),
		ObjectStoreCORSOrigins:    commaValues(value(lookup, "AVIA_OBJECT_STORE_CORS_ORIGINS")),
		QuarantineBucket:          valueOrDefault(lookup, "AVIA_OBJECT_STORE_QUARANTINE_BUCKET", "evidence-quarantine"),
		CanonicalBucket:           valueOrDefault(lookup, "AVIA_OBJECT_STORE_CANONICAL_BUCKET", "evidence-clean"),
		AttachmentBucket:          valueOrDefault(lookup, "AVIA_OBJECT_STORE_ATTACHMENT_BUCKET", "inspection-attachments"),
		DocumentBucket:            valueOrDefault(lookup, "AVIA_OBJECT_STORE_DOCUMENT_BUCKET", "generated-documents"),
		ScannerMode:               valueOrDefault(lookup, "AVIA_SCANNER_MODE", "disabled"),
		SMTPAddress:               value(lookup, "AVIA_SMTP_ADDRESS"),
		SMTPFrom:                  value(lookup, "AVIA_SMTP_FROM"),
		SMTPUsername:              value(lookup, "AVIA_SMTP_USERNAME"),
		SMTPPassword:              smtpPassword,
		SMTPTransport:             value(lookup, "AVIA_SMTP_TRANSPORT"),
		SMTPTLSServerName:         value(lookup, "AVIA_SMTP_TLS_SERVER_NAME"),
		IdentityHealthURL:         value(lookup, "AVIA_IDENTITY_HEALTH_URL"),
		SMTPHealthAddress:         value(lookup, "AVIA_SMTP_HEALTH_ADDRESS"),
		OTLPHTTPEndpoint:          value(lookup, "AVIA_OTEL_EXPORTER_OTLP_ENDPOINT"),
	}
	cookieSecure, err := parseBoolean(lookup, "AVIA_COOKIE_SECURE", true)
	if err != nil {
		return Settings{}, err
	}
	settings.CookieSecure = cookieSecure
	if settings.OIDCDiscoveryURL == "" {
		settings.OIDCDiscoveryURL = settings.OIDCIssuerURL
	}
	if settings.ObjectStorePublicEndpoint == "" && !cloudEnvironment {
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
	if settings.SMTPTransport == "" {
		settings.SMTPTransport = "private-plaintext"
	}
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
	settings.AllowServerManagedCORS = (settings.Environment == "test" &&
		(settings.CanonicalSeed || serverManagedCORS)) ||
		(settings.Environment == "development" && serverManagedCORS)

	if cloudEnvironment {
		if !settings.CookieSecure {
			return Settings{}, fmt.Errorf("AVIA_COOKIE_SECURE=false is forbidden in production")
		}
		for _, key := range []string{"AVIA_TEST_PRINCIPAL", "AVIA_TEST_SESSION", "AVIA_DEV_SESSION_SECRET", "AVIA_ENABLE_CANONICAL_SEED", "AVIA_ENABLE_CANONICAL_TEST_PROFILE", "AVIA_CANONICAL_TEST_TOKEN", "AVIA_OBJECT_STORE_SERVER_MANAGED_CORS"} {
			if value(lookup, key) != "" {
				return Settings{}, fmt.Errorf("%s is forbidden in production", key)
			}
		}
	}
	if serverManagedCORS && settings.Environment != "test" && settings.Environment != "development" {
		return Settings{}, fmt.Errorf(
			"AVIA_OBJECT_STORE_SERVER_MANAGED_CORS requires AVIA_ENVIRONMENT=test or development",
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
	if cloudEnvironment && settings.ScannerMode == "deterministic-test" {
		return Settings{}, fmt.Errorf("AVIA_SCANNER_MODE=deterministic-test is forbidden in cloud environments")
	}
	if !contains([]string{"disabled", "deterministic-test", "guardduty-s3"}, settings.ScannerMode) {
		return Settings{}, fmt.Errorf("unsupported AVIA_SCANNER_MODE %q", settings.ScannerMode)
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
	if !contains([]string{"demo", "development", "test", "production"}, settings.Environment) {
		return Settings{}, fmt.Errorf("AVIA_ENVIRONMENT must be demo, development, test, or production")
	}
	firstPartyConfigured := settings.FirstPartyAdminURL != "" || settings.FirstPartyAdminSecretFile != ""
	if firstPartyConfigured {
		if settings.FirstPartyAdminURL == "" {
			return Settings{}, fmt.Errorf("AVIA_AUTH_ADMIN_URL is required for AviaAuth administration")
		}
		if settings.FirstPartyAdminSecretFile == "" {
			return Settings{}, fmt.Errorf("AVIA_AUTH_ADMIN_SECRET_FILE is required for AviaAuth administration")
		}
		adminURL, err := url.Parse(settings.FirstPartyAdminURL)
		if err != nil || adminURL.Host == "" || (adminURL.Scheme != "http" && adminURL.Scheme != "https") || adminURL.User != nil || adminURL.RawQuery != "" || adminURL.Fragment != "" {
			return Settings{}, fmt.Errorf("AVIA_AUTH_ADMIN_URL must be an absolute HTTP(S) URL without credentials, query, or fragment")
		}
		if !strings.HasPrefix(settings.FirstPartyAdminSecretFile, "/") {
			return Settings{}, fmt.Errorf("AVIA_AUTH_ADMIN_SECRET_FILE must be an absolute secret-file path")
		}
	}

	objectStoreConfigured := settings.ObjectStoreMode != "" ||
		settings.ObjectStoreEndpoint != "" ||
		settings.ObjectStorePublicEndpoint != "" ||
		settings.ObjectStoreAccessKey != "" ||
		settings.ObjectStoreSecretKey != "" ||
		len(settings.ObjectStoreCORSOrigins) > 0
	if (cloudEnvironment && requirements.objectStore) ||
		settings.CanonicalSeed ||
		objectStoreConfigured {
		type requiredValue struct {
			name  string
			value any
		}
		requiredValues := []requiredValue{{name: "AVIA_OBJECT_STORE_CORS_ORIGINS", value: settings.ObjectStoreCORSOrigins}}
		switch settings.ObjectStoreMode {
		case "aws-s3":
			requiredValues = append(requiredValues, requiredValue{name: "AVIA_OBJECT_STORE_REGION", value: settings.ObjectStoreRegion})
			if settings.ObjectStoreAccessKey != "" || settings.ObjectStoreSecretKey != "" {
				return Settings{}, fmt.Errorf("static object-store credentials are forbidden in aws-s3 mode; use the AWS IAM provider chain")
			}
		case "", "minio":
			requiredValues = append(requiredValues,
				requiredValue{name: "AVIA_OBJECT_STORE_ENDPOINT", value: settings.ObjectStoreEndpoint},
				requiredValue{name: "AVIA_OBJECT_STORE_PUBLIC_ENDPOINT", value: settings.ObjectStorePublicEndpoint},
				requiredValue{name: "AVIA_OBJECT_STORE_ACCESS_KEY", value: settings.ObjectStoreAccessKey},
				requiredValue{name: "AVIA_OBJECT_STORE_SECRET_KEY", value: settings.ObjectStoreSecretKey},
			)
		default:
			return Settings{}, fmt.Errorf("unsupported AVIA_OBJECT_STORE_MODE %q", settings.ObjectStoreMode)
		}
		for _, entry := range requiredValues {
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
		if cloudEnvironment && settings.ObjectStoreMode != "aws-s3" &&
			!settings.ObjectStoreTLS &&
			!settings.ObjectStorePrivateNetwork {
			return Settings{}, fmt.Errorf(
				"plaintext object-store transport requires AVIA_OBJECT_STORE_PRIVATE_NETWORK=true",
			)
		}
		if cloudEnvironment && settings.ObjectStoreMode != "aws-s3" && !settings.ObjectStorePublicTLS {
			return Settings{}, fmt.Errorf("AVIA_OBJECT_STORE_PUBLIC_TLS=true is required in production")
		}
	}
	if settings.ObjectStoreMode == "" && objectStoreConfigured {
		settings.ObjectStoreMode = "minio"
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
	if settings.Environment == "production" && requirements.smtp && !smtpConfigured {
		return Settings{}, fmt.Errorf("encrypted SMTP delivery is required by the production worker")
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
		switch settings.SMTPTransport {
		case "private-plaintext":
			if !settings.SMTPPrivateNetwork {
				return Settings{}, fmt.Errorf("plaintext SMTP transport requires AVIA_SMTP_PRIVATE_NETWORK=true")
			}
		case "starttls", "implicit-tls":
			if settings.SMTPTLSServerName == "" {
				return Settings{}, fmt.Errorf("AVIA_SMTP_TLS_SERVER_NAME is required for encrypted SMTP")
			}
		case "":
			return Settings{}, fmt.Errorf("AVIA_SMTP_TRANSPORT is required when SMTP delivery is enabled")
		default:
			return Settings{}, fmt.Errorf("AVIA_SMTP_TRANSPORT must be private-plaintext, starttls, or implicit-tls")
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
		{name: "AVIA_SESSION_ENCRYPTION_KEY", value: sessionEncryptionKey},
	}
	oidcConfigured := false
	for _, entry := range oidcKeys {
		if entry.value != "" {
			oidcConfigured = true
			break
		}
	}
	if (cloudEnvironment && requirements.oidc) ||
		oidcConfigured {
		if !firstPartyConfigured {
			return Settings{}, fmt.Errorf("AviaAuth administration is required when OIDC authentication is enabled")
		}
		for _, entry := range oidcKeys {
			if entry.value == "" {
				return Settings{}, fmt.Errorf("%s is required when OIDC authentication is enabled", entry.name)
			}
		}
		key, err := base64.StdEncoding.DecodeString(sessionEncryptionKey)
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
		if cloudEnvironment && (issuerURL.Scheme != "https" || redirectURL.Scheme != "https") {
			return Settings{}, fmt.Errorf("cloud OIDC issuer and redirect URLs must use HTTPS")
		}
		if cloudEnvironment &&
			discoveryURL.Scheme == "http" &&
			!settings.OIDCDiscoveryPrivateNetwork {
			return Settings{}, fmt.Errorf(
				"plaintext OIDC discovery requires AVIA_OIDC_DISCOVERY_PRIVATE_NETWORK=true",
			)
		}
	}

	for _, entry := range []struct {
		name  string
		value string
	}{
		{name: "AVIA_IDENTITY_HEALTH_URL", value: settings.IdentityHealthURL},
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

func valueOrFile(lookup LookupEnv, name string) (string, error) {
	inline := value(lookup, name)
	fileName := name + "_FILE"
	filePath := value(lookup, fileName)
	if inline != "" && filePath != "" {
		return "", fmt.Errorf("%s and %s are mutually exclusive", name, fileName)
	}
	if filePath == "" {
		return inline, nil
	}
	contents, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("%s could not be read", fileName)
	}
	resolved := strings.TrimSpace(string(contents))
	if resolved == "" {
		return "", fmt.Errorf("%s must contain a non-empty value", fileName)
	}
	return resolved, nil
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

var dataFeedEnvironmentKeys = []string{
	"AVIA_DATA_FEED_TENANT_ID",
	"AVIA_DATA_FEED_OWNING_ORGANIZATION_ID",
	"AVIA_DATA_FEED_REPLAY_ID",
	"AVIA_DATA_FEED_REPLAY_RUN_ID",
	"AVIA_DATA_FEED_PAYLOAD_KEY_FILE",
	"AVIA_DATA_FEED_PAYLOAD_KEY_REF",
	"AVIA_DATA_FEED_ENDPOINT",
	"AVIA_DATA_FEED_CA_BUNDLE_FILE",
	"AVIA_DATA_FEED_CA_BUNDLE_SHA256",
	"AVIA_DATA_FEED_REVOCATION_LIST_FILE",
	"AVIA_DATA_FEED_CLIENT_CERTIFICATE_FILE",
	"AVIA_DATA_FEED_CLIENT_PRIVATE_KEY_FILE",
	"AVIA_DATA_FEED_EXPECTED_CLIENT_SAN",
}

func parseDataMode(lookup LookupEnv) (bool, error) {
	switch valueOrDefault(lookup, "AVIA_DATA", "0") {
	case "0":
		return false, nil
	case "1":
		return true, nil
	default:
		return false, fmt.Errorf("AVIA_DATA must be 0 or 1")
	}
}

func rejectDisabledDataFeedConfiguration(lookup LookupEnv) error {
	for _, key := range dataFeedEnvironmentKeys {
		if value(lookup, key) != "" {
			return fmt.Errorf("%s requires AVIA_DATA=1", key)
		}
	}
	return nil
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
