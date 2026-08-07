package config_test

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/config"
)

func TestProductionRejectsTestAndDevelopmentBypasses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
	}{
		{name: "test identity", key: "AVIA_TEST_PRINCIPAL"},
		{name: "test session", key: "AVIA_TEST_SESSION"},
		{name: "development session secret", key: "AVIA_DEV_SESSION_SECRET"},
		{name: "canonical seed", key: "AVIA_ENABLE_CANONICAL_SEED"},
		{name: "canonical test profile", key: "AVIA_ENABLE_CANONICAL_TEST_PROFILE"},
		{name: "canonical test token", key: "AVIA_CANONICAL_TEST_TOKEN"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			values := map[string]string{
				"AVIA_ENVIRONMENT":  "production",
				"AVIA_DATABASE_URL": "postgres://example.invalid/avia",
				test.key:            "enabled",
			}

			_, err := config.Load(mapLookup(values))
			if err == nil {
				t.Fatalf("Load() accepted production bypass %s", test.key)
			}
			if !strings.Contains(err.Error(), test.key) {
				t.Fatalf("Load() error %q does not identify %s", err, test.key)
			}
		})
	}
}

func TestProductionAPIRuntimeDoesNotRequireWorkerOnlyAdapters(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"AVIA_ENVIRONMENT":                    "production",
		"AVIA_DATABASE_URL":                   "postgres://example.invalid/avia",
		"AVIA_OIDC_ISSUER_URL":                "https://identity.example/realms/avia",
		"AVIA_OIDC_CLIENT_ID":                 "aviasurveil360",
		"AVIA_OIDC_CLIENT_SECRET":             "provider-secret",
		"AVIA_OIDC_REDIRECT_URL":              "https://avia.example/auth/callback",
		"AVIA_OIDC_DISCOVERY_URL":             "http://keycloak:8080/identity/realms/avia",
		"AVIA_OIDC_DISCOVERY_PRIVATE_NETWORK": "true",
		"AVIA_SESSION_ENCRYPTION_KEY":         base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")),
		"AVIA_OBJECT_STORE_ENDPOINT":          "minio:9000",
		"AVIA_OBJECT_STORE_PUBLIC_ENDPOINT":   "localhost:8443",
		"AVIA_OBJECT_STORE_ACCESS_KEY":        "production-access",
		"AVIA_OBJECT_STORE_SECRET_KEY":        "production-secret",
		"AVIA_OBJECT_STORE_CORS_ORIGINS":      "https://localhost:8443",
		"AVIA_OBJECT_STORE_PUBLIC_TLS":        "true",
		"AVIA_OBJECT_STORE_PRIVATE_NETWORK":   "true",
		"AVIA_SCANNER_MODE":                   "clamav",
		"AVIA_CLAMAV_ADDRESS":                 "clamav:3310",
	}

	settings, err := config.LoadAPI(mapLookup(values))
	if err != nil {
		t.Fatalf("LoadAPI() required worker-only adapters: %v", err)
	}
	if settings.OIDCDiscoveryURL != values["AVIA_OIDC_DISCOVERY_URL"] ||
		!settings.OIDCDiscoveryPrivateNetwork {
		t.Fatalf("LoadAPI() private discovery settings = %+v", settings)
	}

	values["AVIA_OIDC_DISCOVERY_PRIVATE_NETWORK"] = "false"
	if _, err := config.LoadAPI(mapLookup(values)); err == nil ||
		!strings.Contains(err.Error(), "AVIA_OIDC_DISCOVERY_PRIVATE_NETWORK") {
		t.Fatalf("LoadAPI() insecure discovery error = %v", err)
	}
}

func TestProductionSchedulerRuntimeRequiresOnlyItsDatabaseCapability(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"AVIA_ENVIRONMENT":  "production",
		"AVIA_DATABASE_URL": "postgres://example.invalid/avia",
	}
	if _, err := config.LoadScheduler(mapLookup(values)); err != nil {
		t.Fatalf("LoadScheduler() required unrelated adapters: %v", err)
	}

	values["AVIA_ENABLE_CANONICAL_SEED"] = "true"
	if _, err := config.LoadScheduler(mapLookup(values)); err == nil ||
		!strings.Contains(err.Error(), "AVIA_ENABLE_CANONICAL_SEED") {
		t.Fatalf("LoadScheduler() production bypass error = %v", err)
	}
}

func TestLocalPreprodEnvironmentIsAcceptedForTheDedicatedDisposableProfile(t *testing.T) {
	t.Parallel()

	settings, err := config.LoadAPI(mapLookup(map[string]string{
		"AVIA_ENVIRONMENT":  "local-preprod",
		"AVIA_DATABASE_URL": "postgres://preprod.example/avia",
	}))
	if err != nil {
		t.Fatalf("LoadAPI() rejected the dedicated local-preprod environment: %v", err)
	}
	if settings.Environment != "local-preprod" {
		t.Fatalf("environment = %q, want local-preprod", settings.Environment)
	}
}

func TestTelemetryEndpointIsOptionalAndMustBePrivateHTTP(t *testing.T) {
	t.Parallel()

	settings, err := config.LoadScheduler(mapLookup(map[string]string{
		"AVIA_ENVIRONMENT":                 "development",
		"AVIA_DATABASE_URL":                "postgres://127.0.0.1/avia",
		"AVIA_OTEL_EXPORTER_OTLP_ENDPOINT": "http://otel-collector:4318",
	}))
	if err != nil {
		t.Fatalf("LoadScheduler() telemetry config error = %v", err)
	}
	if settings.OTLPHTTPEndpoint != "http://otel-collector:4318" {
		t.Fatalf("OTLP endpoint = %q", settings.OTLPHTTPEndpoint)
	}

	for _, endpoint := range []string{
		"https://telemetry.example.invalid",
		"http://telemetry.example.invalid:4318",
		"http://8.8.8.8:4318",
		"http://user:secret@otel-collector:4318",
		"file:///private/telemetry",
	} {
		_, err := config.LoadScheduler(mapLookup(map[string]string{
			"AVIA_ENVIRONMENT":                 "development",
			"AVIA_DATABASE_URL":                "postgres://127.0.0.1/avia",
			"AVIA_OTEL_EXPORTER_OTLP_ENDPOINT": endpoint,
		}))
		if err == nil || !strings.Contains(err.Error(), "AVIA_OTEL_EXPORTER_OTLP_ENDPOINT") {
			t.Fatalf("unsafe telemetry endpoint %q error = %v", endpoint, err)
		}
	}
}

func TestCanonicalHTTPProfileRequiresExplicitTestOnlyObjectStoreConfiguration(t *testing.T) {
	t.Parallel()
	values := map[string]string{
		"AVIA_ENVIRONMENT":                   "test",
		"AVIA_DATABASE_URL":                  "postgres://127.0.0.1/avia",
		"AVIA_ENABLE_CANONICAL_SEED":         "true",
		"AVIA_ENABLE_CANONICAL_TEST_PROFILE": "true",
		"AVIA_CANONICAL_TEST_TOKEN":          "candidate-test-token-1234",
		"AVIA_OBJECT_STORE_ENDPOINT":         "127.0.0.1:59001",
		"AVIA_OBJECT_STORE_ACCESS_KEY":       "local-access",
		"AVIA_OBJECT_STORE_SECRET_KEY":       "local-secret",
		"AVIA_OBJECT_STORE_CORS_ORIGINS":     "http://127.0.0.1:4173",
		"AVIA_SCANNER_MODE":                  "deterministic-test",
	}
	settings, err := config.Load(mapLookup(values))
	if err != nil {
		t.Fatalf("Load() canonical HTTP profile: %v", err)
	}
	if !settings.CanonicalTestProfile || settings.CanonicalTestToken != values["AVIA_CANONICAL_TEST_TOKEN"] {
		t.Fatalf("canonical profile = %+v", settings)
	}
	if settings.ObjectStoreEndpoint != "127.0.0.1:59001" || len(settings.ObjectStoreCORSOrigins) != 1 || !settings.AllowServerManagedCORS {
		t.Fatalf("object-store profile = %+v", settings)
	}

	for _, missing := range []string{
		"AVIA_CANONICAL_TEST_TOKEN", "AVIA_OBJECT_STORE_ENDPOINT", "AVIA_OBJECT_STORE_ACCESS_KEY",
		"AVIA_OBJECT_STORE_SECRET_KEY", "AVIA_OBJECT_STORE_CORS_ORIGINS",
	} {
		t.Run("missing "+missing, func(t *testing.T) {
			candidate := cloneValues(values)
			delete(candidate, missing)
			if _, err := config.Load(mapLookup(candidate)); err == nil || !strings.Contains(err.Error(), missing) {
				t.Fatalf("missing %s error = %v", missing, err)
			}
		})
	}
}

func TestCanonicalSeedAndHeaderProfileAreSeparated(t *testing.T) {
	t.Parallel()
	values := map[string]string{
		"AVIA_ENVIRONMENT":               "test",
		"AVIA_DATABASE_URL":              "postgres://127.0.0.1/avia",
		"AVIA_ENABLE_CANONICAL_SEED":     "true",
		"AVIA_OBJECT_STORE_ENDPOINT":     "127.0.0.1:59001",
		"AVIA_OBJECT_STORE_ACCESS_KEY":   "local-access",
		"AVIA_OBJECT_STORE_SECRET_KEY":   "local-secret",
		"AVIA_OBJECT_STORE_CORS_ORIGINS": "http://127.0.0.1:4174",
		"AVIA_SCANNER_MODE":              "deterministic-test",
	}
	settings, err := config.Load(mapLookup(values))
	if err != nil {
		t.Fatalf("Load() seed-only OIDC profile: %v", err)
	}
	if !settings.CanonicalSeed || settings.CanonicalTestProfile || settings.CanonicalTestToken != "" {
		t.Fatalf("seed/profile settings = %+v", settings)
	}
	if !settings.AllowServerManagedCORS {
		t.Fatalf("seed-only local lane must allow deterministic object-store CORS: %+v", settings)
	}

	canonicalHeaders := cloneValues(values)
	canonicalHeaders["AVIA_ENABLE_CANONICAL_TEST_PROFILE"] = "true"
	canonicalHeaders["AVIA_CANONICAL_TEST_TOKEN"] = "candidate-test-token-1234"
	headerSettings, err := config.Load(mapLookup(canonicalHeaders))
	if err != nil {
		t.Fatalf("Load() canonical-header profile: %v", err)
	}
	if !headerSettings.CanonicalSeed || !headerSettings.CanonicalTestProfile {
		t.Fatalf("header profile must require the seed explicitly: %+v", headerSettings)
	}

	production := cloneValues(values)
	production["AVIA_ENVIRONMENT"] = "production"
	if _, err := config.Load(mapLookup(production)); err == nil || !strings.Contains(err.Error(), "AVIA_ENABLE_CANONICAL_SEED") {
		t.Fatalf("production seed flag error = %v", err)
	}
}

func TestOIDCProfileExplicitlyAllowsTestOnlyServerManagedObjectStoreCORS(t *testing.T) {
	t.Parallel()
	values := map[string]string{
		"AVIA_ENVIRONMENT":                      "test",
		"AVIA_DATABASE_URL":                     "postgres://127.0.0.1/avia",
		"AVIA_OBJECT_STORE_ENDPOINT":            "127.0.0.1:59001",
		"AVIA_OBJECT_STORE_ACCESS_KEY":          "local-access",
		"AVIA_OBJECT_STORE_SECRET_KEY":          "local-secret",
		"AVIA_OBJECT_STORE_CORS_ORIGINS":        "http://127.0.0.1:4174",
		"AVIA_OBJECT_STORE_SERVER_MANAGED_CORS": "true",
	}
	settings, err := config.Load(mapLookup(values))
	if err != nil {
		t.Fatalf("Load() OIDC object-store profile: %v", err)
	}
	if settings.CanonicalSeed || !settings.AllowServerManagedCORS {
		t.Fatalf("OIDC object-store CORS settings = %+v", settings)
	}

	values["AVIA_ENVIRONMENT"] = "production"
	if _, err := config.Load(mapLookup(values)); err == nil ||
		!strings.Contains(err.Error(), "AVIA_OBJECT_STORE_SERVER_MANAGED_CORS") {
		t.Fatalf("production server-managed CORS error = %v", err)
	}
}

func TestExplicitTestProfileLoadsDeterministicPrincipal(t *testing.T) {
	t.Parallel()

	settings, err := config.Load(mapLookup(map[string]string{
		"AVIA_ENVIRONMENT":    "test",
		"AVIA_DATABASE_URL":   "postgres://127.0.0.1/avia",
		"AVIA_TEST_PRINCIPAL": "inspector-cabin-001",
		"AVIA_TEST_SESSION":   "session-cabin-001",
	}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if settings.TestPrincipal != "inspector-cabin-001" {
		t.Fatalf("TestPrincipal = %q", settings.TestPrincipal)
	}
	if settings.TestSession != "session-cabin-001" {
		t.Fatalf("TestSession = %q", settings.TestSession)
	}
}

func TestProductionRequiresCompleteHTTPSOIDCAndSessionConfiguration(t *testing.T) {
	t.Parallel()

	base := map[string]string{
		"AVIA_ENVIRONMENT":                  "production",
		"AVIA_DATABASE_URL":                 "postgres://example.invalid/avia",
		"AVIA_OIDC_ISSUER_URL":              "https://identity.example/realms/avia",
		"AVIA_OIDC_CLIENT_ID":               "aviasurveil360",
		"AVIA_OIDC_CLIENT_SECRET":           "provider-secret",
		"AVIA_OIDC_REDIRECT_URL":            "https://avia.example/auth/callback",
		"AVIA_SESSION_ENCRYPTION_KEY":       base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")),
		"AVIA_OBJECT_STORE_ENDPOINT":        "objects.example:443",
		"AVIA_OBJECT_STORE_PUBLIC_ENDPOINT": "objects.example:443",
		"AVIA_OBJECT_STORE_ACCESS_KEY":      "production-access",
		"AVIA_OBJECT_STORE_SECRET_KEY":      "production-secret",
		"AVIA_OBJECT_STORE_CORS_ORIGINS":    "https://avia.example",
		"AVIA_OBJECT_STORE_TLS":             "true",
		"AVIA_OBJECT_STORE_PUBLIC_TLS":      "true",
		"AVIA_SCANNER_MODE":                 "clamav",
		"AVIA_CLAMAV_ADDRESS":               "scanner.example:3310",
		"AVIA_CLAMAV_MAX_SIGNATURE_AGE":     "48h",
		"AVIA_GOTENBERG_URL":                "http://gotenberg:3000",
		"AVIA_GOTENBERG_TIMEOUT":            "30s",
		"AVIA_GOTENBERG_RENDERER_HASH":      "sha256:56c47f7b913f3b978554115a0191c4a9dcc2558f9090f27f3f13f28a7c2f8329",
	}
	settings, err := config.Load(mapLookup(base))
	if err != nil {
		t.Fatalf("Load() complete production config: %v", err)
	}
	if settings.OIDCIssuerURL != base["AVIA_OIDC_ISSUER_URL"] || settings.OIDCClientID != "aviasurveil360" {
		t.Fatalf("OIDC settings = %+v", settings)
	}
	if settings.SessionIdleDuration != 30*time.Minute || settings.SessionAbsoluteDuration != 8*time.Hour {
		t.Fatalf("session policy = idle %s absolute %s", settings.SessionIdleDuration, settings.SessionAbsoluteDuration)
	}
	if len(settings.SessionEncryptionKey) != 32 || !settings.CookieSecure {
		t.Fatalf("session security config = key bytes %d, secure cookie %t", len(settings.SessionEncryptionKey), settings.CookieSecure)
	}

	for _, missing := range []string{
		"AVIA_OIDC_ISSUER_URL", "AVIA_OIDC_CLIENT_ID", "AVIA_OIDC_CLIENT_SECRET",
		"AVIA_OIDC_REDIRECT_URL", "AVIA_SESSION_ENCRYPTION_KEY",
	} {
		t.Run("missing "+missing, func(t *testing.T) {
			values := cloneValues(base)
			delete(values, missing)
			if _, err := config.Load(mapLookup(values)); err == nil || !strings.Contains(err.Error(), missing) {
				t.Fatalf("Load() missing %s error = %v", missing, err)
			}
		})
	}
}

func TestCookieSecureOverrideIsAllowedOnlyOutsideProduction(t *testing.T) {
	t.Parallel()

	local, err := config.LoadScheduler(mapLookup(map[string]string{
		"AVIA_ENVIRONMENT":   "development",
		"AVIA_DATABASE_URL":  "postgres://127.0.0.1/avia",
		"AVIA_COOKIE_SECURE": "false",
	}))
	if err != nil {
		t.Fatalf("local cookie configuration: %v", err)
	}
	if local.CookieSecure {
		t.Fatal("local HTTP cookie override was ignored")
	}

	_, err = config.LoadScheduler(mapLookup(map[string]string{
		"AVIA_ENVIRONMENT":   "production",
		"AVIA_DATABASE_URL":  "postgres://127.0.0.1/avia",
		"AVIA_COOKIE_SECURE": "false",
	}))
	if err == nil || !strings.Contains(err.Error(), "AVIA_COOKIE_SECURE") {
		t.Fatalf("production accepted insecure cookies: %v", err)
	}
}

func TestProductionRejectsInsecureOIDCEndpointsAndInvalidEncryptionKey(t *testing.T) {
	t.Parallel()
	base := map[string]string{
		"AVIA_ENVIRONMENT":                  "production",
		"AVIA_DATABASE_URL":                 "postgres://example.invalid/avia",
		"AVIA_OIDC_ISSUER_URL":              "https://identity.example/realms/avia",
		"AVIA_OIDC_CLIENT_ID":               "aviasurveil360",
		"AVIA_OIDC_CLIENT_SECRET":           "provider-secret",
		"AVIA_OIDC_REDIRECT_URL":            "https://avia.example/auth/callback",
		"AVIA_SESSION_ENCRYPTION_KEY":       base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")),
		"AVIA_OBJECT_STORE_ENDPOINT":        "objects.example:443",
		"AVIA_OBJECT_STORE_PUBLIC_ENDPOINT": "objects.example:443",
		"AVIA_OBJECT_STORE_ACCESS_KEY":      "production-access",
		"AVIA_OBJECT_STORE_SECRET_KEY":      "production-secret",
		"AVIA_OBJECT_STORE_CORS_ORIGINS":    "https://avia.example",
		"AVIA_OBJECT_STORE_TLS":             "true",
		"AVIA_OBJECT_STORE_PUBLIC_TLS":      "true",
		"AVIA_SCANNER_MODE":                 "clamav",
		"AVIA_CLAMAV_ADDRESS":               "scanner.example:3310",
		"AVIA_CLAMAV_MAX_SIGNATURE_AGE":     "48h",
		"AVIA_GOTENBERG_URL":                "http://gotenberg:3000",
		"AVIA_GOTENBERG_TIMEOUT":            "30s",
		"AVIA_GOTENBERG_RENDERER_HASH":      "sha256:56c47f7b913f3b978554115a0191c4a9dcc2558f9090f27f3f13f28a7c2f8329",
	}
	for name, mutation := range map[string]func(map[string]string){
		"HTTP issuer":   func(values map[string]string) { values["AVIA_OIDC_ISSUER_URL"] = "http://identity.example/realms/avia" },
		"HTTP redirect": func(values map[string]string) { values["AVIA_OIDC_REDIRECT_URL"] = "http://avia.example/auth/callback" },
		"short session key": func(values map[string]string) {
			values["AVIA_SESSION_ENCRYPTION_KEY"] = base64.StdEncoding.EncodeToString([]byte("short"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			values := cloneValues(base)
			mutation(values)
			if _, err := config.Load(mapLookup(values)); err == nil {
				t.Fatalf("Load() accepted %s", name)
			}
		})
	}
}

func TestProductionObjectStorageUsesPrivateTransportPublicHTTPSSigningAndRealClamAV(t *testing.T) {
	t.Parallel()
	values := map[string]string{
		"AVIA_ENVIRONMENT":                  "production",
		"AVIA_DATABASE_URL":                 "postgres://example.invalid/avia",
		"AVIA_OIDC_ISSUER_URL":              "https://identity.example/realms/avia",
		"AVIA_OIDC_CLIENT_ID":               "aviasurveil360",
		"AVIA_OIDC_CLIENT_SECRET":           "provider-secret",
		"AVIA_OIDC_REDIRECT_URL":            "https://avia.example/auth/callback",
		"AVIA_SESSION_ENCRYPTION_KEY":       base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")),
		"AVIA_OBJECT_STORE_ENDPOINT":        "minio:9000",
		"AVIA_OBJECT_STORE_PUBLIC_ENDPOINT": "localhost:8443",
		"AVIA_OBJECT_STORE_ACCESS_KEY":      "production-access",
		"AVIA_OBJECT_STORE_SECRET_KEY":      "production-secret",
		"AVIA_OBJECT_STORE_CORS_ORIGINS":    "https://localhost:8443",
		"AVIA_OBJECT_STORE_TLS":             "false",
		"AVIA_OBJECT_STORE_PUBLIC_TLS":      "true",
		"AVIA_OBJECT_STORE_PRIVATE_NETWORK": "true",
		"AVIA_SCANNER_MODE":                 "clamav",
		"AVIA_CLAMAV_ADDRESS":               "clamav:3310",
		"AVIA_CLAMAV_MAX_SIGNATURE_AGE":     "48h",
		"AVIA_GOTENBERG_URL":                "http://gotenberg:3000",
		"AVIA_GOTENBERG_TIMEOUT":            "30s",
		"AVIA_GOTENBERG_RENDERER_HASH":      "sha256:56c47f7b913f3b978554115a0191c4a9dcc2558f9090f27f3f13f28a7c2f8329",
	}
	settings, err := config.Load(mapLookup(values))
	if err != nil {
		t.Fatalf("Load() production object/scanner config: %v", err)
	}
	if settings.ObjectStorePublicEndpoint != "localhost:8443" ||
		!settings.ObjectStorePublicTLS ||
		!settings.ObjectStorePrivateNetwork ||
		settings.QuarantineBucket != "evidence-quarantine" ||
		settings.CanonicalBucket != "evidence-clean" ||
		settings.AttachmentBucket != "inspection-attachments" ||
		settings.DocumentBucket != "generated-documents" ||
		settings.ClamAVAddress != "clamav:3310" ||
		settings.ClamAVMaximumSignatureAge != 48*time.Hour ||
		settings.GotenbergURL != "http://gotenberg:3000" ||
		settings.GotenbergTimeout != 30*time.Second ||
		settings.GotenbergRendererHash != values["AVIA_GOTENBERG_RENDERER_HASH"] {
		t.Fatalf("production object/scanner settings = %+v", settings)
	}

	for name, mutation := range map[string]func(map[string]string){
		"missing public signer": func(candidate map[string]string) {
			delete(candidate, "AVIA_OBJECT_STORE_PUBLIC_ENDPOINT")
		},
		"insecure public signer": func(candidate map[string]string) {
			candidate["AVIA_OBJECT_STORE_PUBLIC_TLS"] = "false"
		},
		"plaintext non-private internal transport": func(candidate map[string]string) {
			candidate["AVIA_OBJECT_STORE_PRIVATE_NETWORK"] = "false"
		},
		"deterministic scanner": func(candidate map[string]string) {
			candidate["AVIA_SCANNER_MODE"] = "deterministic-test"
		},
		"missing ClamAV endpoint": func(candidate map[string]string) {
			delete(candidate, "AVIA_CLAMAV_ADDRESS")
		},
		"missing Gotenberg endpoint": func(candidate map[string]string) {
			delete(candidate, "AVIA_GOTENBERG_URL")
		},
		"invalid Gotenberg renderer hash": func(candidate map[string]string) {
			candidate["AVIA_GOTENBERG_RENDERER_HASH"] = "sha256:not-a-digest"
		},
		"unbounded Gotenberg timeout": func(candidate map[string]string) {
			candidate["AVIA_GOTENBERG_TIMEOUT"] = "3m"
		},
		"duplicate clean and attachment buckets": func(candidate map[string]string) {
			candidate["AVIA_OBJECT_STORE_ATTACHMENT_BUCKET"] = "evidence-clean"
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := cloneValues(values)
			mutation(candidate)
			if _, err := config.Load(mapLookup(candidate)); err == nil {
				t.Fatalf("Load() accepted %s", name)
			}
		})
	}
}

func TestKeycloakAdminConfigurationRequiresACompleteInternalEndpointAndServiceCredentials(t *testing.T) {
	t.Parallel()
	base := map[string]string{
		"AVIA_ENVIRONMENT":                    "development",
		"AVIA_DATABASE_URL":                   "postgres://127.0.0.1/avia",
		"AVIA_KEYCLOAK_ADMIN_URL":             "http://keycloak:8080/identity",
		"AVIA_KEYCLOAK_REALM":                 "aviasurveil360",
		"AVIA_KEYCLOAK_SERVICE_CLIENT_ID":     "aviasurveil360-lifecycle",
		"AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET": "lifecycle-client-secret",
	}
	settings, err := config.Load(mapLookup(base))
	if err != nil {
		t.Fatalf("Load() Keycloak admin config: %v", err)
	}
	if settings.KeycloakAdminURL != base["AVIA_KEYCLOAK_ADMIN_URL"] ||
		settings.KeycloakRealm != "aviasurveil360" ||
		settings.KeycloakServiceClientID != "aviasurveil360-lifecycle" ||
		settings.KeycloakServiceClientSecret != "lifecycle-client-secret" {
		t.Fatalf("Keycloak admin settings = %+v", settings)
	}

	for _, missing := range []string{
		"AVIA_KEYCLOAK_ADMIN_URL",
		"AVIA_KEYCLOAK_REALM",
		"AVIA_KEYCLOAK_SERVICE_CLIENT_ID",
		"AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET",
	} {
		t.Run("missing "+missing, func(t *testing.T) {
			values := cloneValues(base)
			delete(values, missing)
			if _, err := config.Load(mapLookup(values)); err == nil ||
				!strings.Contains(err.Error(), missing) {
				t.Fatalf("missing %s error = %v", missing, err)
			}
		})
	}

	insecure := cloneValues(base)
	insecure["AVIA_KEYCLOAK_ADMIN_URL"] = "file:///run/keycloak"
	if _, err := config.Load(mapLookup(insecure)); err == nil ||
		!strings.Contains(err.Error(), "AVIA_KEYCLOAK_ADMIN_URL") {
		t.Fatalf("invalid Keycloak admin URL error = %v", err)
	}
}

func TestSMTPConfigurationRequiresCompleteBoundedPrivateTransport(t *testing.T) {
	t.Parallel()
	base := map[string]string{
		"AVIA_ENVIRONMENT":          "development",
		"AVIA_DATABASE_URL":         "postgres://example.invalid/avia",
		"AVIA_SMTP_ADDRESS":         "mailpit:1025",
		"AVIA_SMTP_FROM":            "no-reply@aviasurveil360.local",
		"AVIA_SMTP_USERNAME":        "aviasurveil360",
		"AVIA_SMTP_PASSWORD":        "smtp-secret",
		"AVIA_SMTP_TIMEOUT":         "10s",
		"AVIA_SMTP_PRIVATE_NETWORK": "true",
	}
	settings, err := config.Load(mapLookup(base))
	if err != nil {
		t.Fatalf("Load() SMTP config: %v", err)
	}
	if settings.SMTPAddress != "mailpit:1025" ||
		settings.SMTPFrom != "no-reply@aviasurveil360.local" ||
		settings.SMTPUsername != "aviasurveil360" ||
		settings.SMTPPassword != "smtp-secret" ||
		settings.SMTPTimeout != 10*time.Second ||
		!settings.SMTPPrivateNetwork {
		t.Fatalf("SMTP settings = %+v", settings)
	}

	for _, missing := range []string{
		"AVIA_SMTP_ADDRESS",
		"AVIA_SMTP_FROM",
		"AVIA_SMTP_USERNAME",
		"AVIA_SMTP_PASSWORD",
	} {
		t.Run("missing "+missing, func(t *testing.T) {
			values := cloneValues(base)
			delete(values, missing)
			if _, err := config.Load(mapLookup(values)); err == nil ||
				!strings.Contains(err.Error(), missing) {
				t.Fatalf("missing %s error = %v", missing, err)
			}
		})
	}
	for name, mutate := range map[string]func(map[string]string){
		"non-private plaintext transport": func(values map[string]string) {
			values["AVIA_SMTP_PRIVATE_NETWORK"] = "false"
		},
		"invalid address": func(values map[string]string) {
			values["AVIA_SMTP_ADDRESS"] = "mailpit"
		},
		"unbounded timeout": func(values map[string]string) {
			values["AVIA_SMTP_TIMEOUT"] = "2m"
		},
	} {
		t.Run(name, func(t *testing.T) {
			values := cloneValues(base)
			mutate(values)
			if _, err := config.Load(mapLookup(values)); err == nil {
				t.Fatalf("Load() accepted %s", name)
			}
		})
	}
}

func TestRuntimeHealthEndpointsAreBoundedAndContainNoCredentials(t *testing.T) {
	t.Parallel()

	settings, err := config.Load(mapLookup(map[string]string{
		"AVIA_DATABASE_URL":           "postgres://localhost/avia",
		"AVIA_IDENTITY_HEALTH_URL":    "http://keycloak:8080/identity/realms/aviasurveil360/.well-known/openid-configuration",
		"AVIA_GOTENBERG_HEALTH_URL":   "http://gotenberg:3000/health",
		"AVIA_SMTP_HEALTH_ADDRESS":    "mailpit:1025",
		"AVIA_RUNTIME_HEALTH_TIMEOUT": "750ms",
	}))
	if err != nil {
		t.Fatalf("Load() runtime health config: %v", err)
	}
	if settings.IdentityHealthURL == "" ||
		settings.GotenbergHealthURL == "" ||
		settings.SMTPHealthAddress != "mailpit:1025" ||
		settings.RuntimeHealthTimeout != 750*time.Millisecond {
		t.Fatalf("runtime health settings = %+v", settings)
	}

	for key, value := range map[string]string{
		"AVIA_IDENTITY_HEALTH_URL":  "http://user:secret@keycloak/health",
		"AVIA_GOTENBERG_HEALTH_URL": "file:///private/renderer",
		"AVIA_SMTP_HEALTH_ADDRESS":  "mailpit",
	} {
		_, err := config.Load(mapLookup(map[string]string{
			"AVIA_DATABASE_URL": "postgres://localhost/avia",
			key:                 value,
		}))
		if err == nil {
			t.Fatalf("unsafe %s=%q was accepted", key, value)
		}
	}
}

func cloneValues(source map[string]string) map[string]string {
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func mapLookup(values map[string]string) config.LookupEnv {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
