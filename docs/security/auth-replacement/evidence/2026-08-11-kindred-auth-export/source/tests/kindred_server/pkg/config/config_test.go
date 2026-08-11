package config_test

import (
	"testing"
	"time"

	"kindred_server/pkg/config"
)

// clearAll unsets every env var Load() reads so a test can observe defaults.
func clearAll(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"APP_ENV", "SERVICE_NAME", "AWS_REGION",
		"USERS_TABLE", "APP_TABLE", "UPLOADS_BUCKET", "S3_ENDPOINT", "S3_ACCESS_KEY_ID", "S3_SECRET_ACCESS_KEY",
		"UPLOAD_URL_TTL_SECONDS", "DOWNLOAD_URL_TTL_SECONDS",
		"TOKEN_SECRET", "TOKEN_TTL_SECONDS", "LOCAL_HTTP_ADDR",
		"TRANSPARENCY_SIGNING_KEY_B64", "TRANSPARENCY_SIGNING_KEY_SSM_PATH",
		"DYNAMODB_ENDPOINT",
		"EMAIL_VERIFICATION_TTL_SECONDS", "PASSWORD_RESET_TTL_SECONDS",
		"MAX_FAILED_LOGIN_ATTEMPTS", "LOCKOUT_DURATION_SECONDS",
		"REQUIRE_VERIFIED_EMAIL", "MOBILE_DEEP_LINK_SCHEME",
		"MESSAGE_THREAD_MAX_SERVER_MESSAGES", "MESSAGE_CLOSED_RETENTION_DAYS",
		"AUTH_RATE_LIMIT_MAX", "AUTH_RATE_LIMIT_WINDOW_SECONDS", "RATE_LIMIT_TABLE",
		"SEED_USER_EMAIL", "SEED_USER_PASSWORD", "SEED_USER_DISPLAY_NAME", "SEED_USER_CITY",
		"SMS_WEBHOOK_URL", "SMS_WEBHOOK_TOKEN",
		"STOREKIT_BUNDLE_ID", "STOREKIT_PREMIUM_MONTHLY_PRODUCT_ID", "STOREKIT_ENVIRONMENT", "STOREKIT_ENABLE_ONLINE_CHECKS",
		"ANALYTICS_ENABLED", "ANALYTICS_FIREHOSE_STREAM", "ANALYTICS_SCHEMA_VERSION",
	} {
		t.Setenv(k, "")
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearAll(t)
	c := config.Load()
	if c.Environment != "dev" {
		t.Errorf("Environment = %q", c.Environment)
	}
	if c.ServiceName != "kindred-server" {
		t.Errorf("ServiceName = %q", c.ServiceName)
	}
	if c.AWSRegion != "eu-central-1" {
		t.Errorf("AWSRegion = %q", c.AWSRegion)
	}
	if c.TokenTTL != 1800*time.Second {
		t.Errorf("TokenTTL = %v", c.TokenTTL)
	}
	if c.UploadURLTTL != 900*time.Second {
		t.Errorf("UploadURLTTL = %v", c.UploadURLTTL)
	}
	if c.MaxFailedLoginAttempts != 5 {
		t.Errorf("MaxFailedLoginAttempts = %d", c.MaxFailedLoginAttempts)
	}
	if c.RequireVerifiedEmail {
		t.Error("RequireVerifiedEmail default should be false")
	}
	if c.MobileDeepLinkScheme != "kindred" {
		t.Errorf("MobileDeepLinkScheme = %q, want kindred", c.MobileDeepLinkScheme)
	}
	if c.RateLimitMax != 20 {
		t.Errorf("RateLimitMax = %d", c.RateLimitMax)
	}
	if c.MessageThreadMaxServerMessages != 1000 {
		t.Errorf("MessageThreadMaxServerMessages = %d", c.MessageThreadMaxServerMessages)
	}
	if c.MessageClosedRetentionDays != 30 {
		t.Errorf("MessageClosedRetentionDays = %d", c.MessageClosedRetentionDays)
	}
	if c.UploadsBucket != "" || c.RateLimitTable != "" || c.SeedUserEmail != "" {
		t.Error("optional fields should default to empty")
	}
	if c.AnalyticsEnabled || c.AnalyticsFirehoseStream != "" || c.AnalyticsSchemaVersion != 1 {
		t.Errorf("analytics defaults not applied: %+v", c)
	}
	if c.StoreKitEnvironment != "Sandbox" {
		t.Errorf("StoreKitEnvironment = %q, want Sandbox", c.StoreKitEnvironment)
	}
	if c.StoreKitEnableOnlineChecks {
		t.Error("StoreKitEnableOnlineChecks default should be false in dev")
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	clearAll(t)
	t.Setenv("APP_ENV", "prod")
	t.Setenv("SERVICE_NAME", "svc")
	t.Setenv("UPLOADS_BUCKET", "buck")
	t.Setenv("S3_ENDPOINT", "http://localhost:9000")
	t.Setenv("SMS_WEBHOOK_URL", "https://sms.example/send")
	t.Setenv("SMS_WEBHOOK_TOKEN", "fixture-secret")
	t.Setenv("TRANSPARENCY_SIGNING_KEY_SSM_PATH", "/kindred/prod/transparency_signing_key")
	t.Setenv("MOBILE_DEEP_LINK_SCHEME", "kindred-beta")
	t.Setenv("ANALYTICS_ENABLED", "true")
	t.Setenv("ANALYTICS_FIREHOSE_STREAM", "analytics-stream")
	t.Setenv("ANALYTICS_SCHEMA_VERSION", "2")
	c := config.Load()
	if c.Environment != "prod" || c.ServiceName != "svc" || c.UploadsBucket != "buck" || c.S3Endpoint != "http://localhost:9000" {
		t.Errorf("overrides not applied: %+v", c)
	}
	if c.SMSWebhookURL != "https://sms.example/send" || c.SMSWebhookToken != "fixture-secret" {
		t.Errorf("sms overrides not applied: %+v", c)
	}
	if c.TransparencySigningKeySSMPath != "/kindred/prod/transparency_signing_key" {
		t.Errorf("transparency signing key SSM path not applied: %+v", c)
	}
	if c.MobileDeepLinkScheme != "kindred-beta" {
		t.Errorf("mobile deep link scheme override not applied: %+v", c)
	}
	if !c.AnalyticsEnabled || c.AnalyticsFirehoseStream != "analytics-stream" || c.AnalyticsSchemaVersion != 2 {
		t.Errorf("analytics overrides not applied: %+v", c)
	}
	if c.StoreKitEnvironment != "Production" {
		t.Errorf("StoreKitEnvironment = %q, want Production", c.StoreKitEnvironment)
	}
	if !c.StoreKitEnableOnlineChecks {
		t.Error("StoreKitEnableOnlineChecks default should be true in prod")
	}
}

func TestLoad_StoreKitOnlineChecksPreprodAndOverride(t *testing.T) {
	clearAll(t)
	t.Setenv("APP_ENV", "preprod")
	c := config.Load()
	if !c.StoreKitEnableOnlineChecks {
		t.Error("StoreKitEnableOnlineChecks default should be true in preprod")
	}

	t.Setenv("STOREKIT_ENABLE_ONLINE_CHECKS", "false")
	c = config.Load()
	if c.StoreKitEnableOnlineChecks {
		t.Error("StoreKitEnableOnlineChecks env override should disable online checks")
	}
}

func TestLoad_DurationParsing(t *testing.T) {
	clearAll(t)
	t.Setenv("TOKEN_TTL_SECONDS", "120")
	t.Setenv("UPLOAD_URL_TTL_SECONDS", "30")
	t.Setenv("LOCKOUT_DURATION_SECONDS", "45")
	c := config.Load()
	if c.TokenTTL != 120*time.Second {
		t.Errorf("TokenTTL = %v", c.TokenTTL)
	}
	if c.UploadURLTTL != 30*time.Second {
		t.Errorf("UploadURLTTL = %v", c.UploadURLTTL)
	}
	if c.LockoutDuration != 45*time.Second {
		t.Errorf("LockoutDuration = %v", c.LockoutDuration)
	}
}

func TestLoad_DurationParseFailureFallsBackToDefault(t *testing.T) {
	clearAll(t)
	t.Setenv("TOKEN_TTL_SECONDS", "not-a-number")
	c := config.Load()
	if c.TokenTTL != 1800*time.Second {
		t.Errorf("malformed int should fall back to default; got %v", c.TokenTTL)
	}
}

func TestLoad_BoolParsing(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{"true", true}, {"false", false}, {"1", true}, {"0", false},
		{"TRUE", true}, {"garbage", false}, // garbage falls back to default (false)
	}
	for _, c := range cases {
		clearAll(t)
		t.Setenv("REQUIRE_VERIFIED_EMAIL", c.val)
		got := config.Load().RequireVerifiedEmail
		if got != c.want {
			t.Errorf("REQUIRE_VERIFIED_EMAIL=%q -> %v, want %v", c.val, got, c.want)
		}
	}
}

func TestLoad_IntOverrides(t *testing.T) {
	clearAll(t)
	t.Setenv("MAX_FAILED_LOGIN_ATTEMPTS", "10")
	t.Setenv("AUTH_RATE_LIMIT_MAX", "100")
	t.Setenv("MESSAGE_THREAD_MAX_SERVER_MESSAGES", "500")
	t.Setenv("MESSAGE_CLOSED_RETENTION_DAYS", "14")
	c := config.Load()
	if c.MaxFailedLoginAttempts != 10 {
		t.Errorf("MaxFailedLoginAttempts = %d", c.MaxFailedLoginAttempts)
	}
	if c.RateLimitMax != 100 {
		t.Errorf("RateLimitMax = %d", c.RateLimitMax)
	}
	if c.MessageThreadMaxServerMessages != 500 {
		t.Errorf("MessageThreadMaxServerMessages = %d", c.MessageThreadMaxServerMessages)
	}
	if c.MessageClosedRetentionDays != 14 {
		t.Errorf("MessageClosedRetentionDays = %d", c.MessageClosedRetentionDays)
	}
}
