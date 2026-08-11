package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment   string
	ServiceName   string
	AWSRegion     string
	UsersTable    string
	AppTable      string
	UploadsBucket string
	S3Endpoint    string
	S3AccessKeyID string
	S3SecretKey   string

	UploadURLTTL            time.Duration
	DownloadURLTTL          time.Duration
	TokenTTL                time.Duration
	RefreshTokenIdleTTL     time.Duration
	RefreshTokenAbsoluteTTL time.Duration
	LocalHTTPAddr           string

	// JWT signing/verification configuration. The private key may be
	// provided in three ways, in priority order:
	//   1. JWTPrivateKeyPEM env var (raw PEM, used by tests / local dev)
	//   2. JWTPrivateKeySSMPath (SSM SecureString, fetched at startup)
	//   3. JWTPrivateKeyPath (local file, used by `make dev` autogen)
	JWTPrivateKeyPEM              string
	JWTPrivateKeySSMPath          string
	JWTPrivateKeyPath             string
	JWTIssuer                     string
	JWTAudience                   string
	TransparencySigningKeyBase64  string
	TransparencySigningKeySSMPath string

	// DynamoDBEndpoint lets the DynamoDB client target a non-AWS endpoint,
	// such as DynamoDB Local running at http://localhost:8000 in the dev
	// environment. Leave empty for preprod/prod to hit real AWS.
	DynamoDBEndpoint string

	VerificationTokenTTL        time.Duration
	PasswordResetTokenTTL       time.Duration
	MaxFailedLoginAttempts      int
	LockoutDuration             time.Duration
	RequireVerifiedEmail        bool
	MobileDeepLinkScheme        string
	PhoneVerificationTTL        time.Duration
	ReturnPhoneVerificationCode bool

	ThreadBasePrice                int
	ThreadDemandStep               int
	ThreadDemandMultiplierStepPct  int
	ThreadMinCost                  int
	ThreadMaxCost                  int
	DeliveryBaseReward             int
	DeliveryRatingBonusPerStar     int
	DeliveryMinRewardRating        int
	RepeatRecipientCooldownDays    int
	NewAccountActiveThreadLimit    int
	ActiveThreadLimit              int
	NewAccountAgeDays              int
	MessageThreadMaxServerMessages int
	MessageClosedRetentionDays     int

	StoreKitBundleID                string
	StoreKitPremiumMonthlyProductID string
	StoreKitEnvironment             string
	StoreKitEnableOnlineChecks      bool

	RateLimitMax    int
	RateLimitWindow time.Duration
	// RateLimitTable, when set, enables the distributed DynamoDB-backed
	// rate limiter. When empty, rate limiting falls back to in-memory.
	// Intentionally points at the same table as USERS_TABLE in deployed
	// envs — both are pk-keyed and rate-limit rows use an `rl:` prefix +
	// the table's TTL attribute, so they don't collide with user records.
	RateLimitTable string

	// SeedUserEmail, when set, creates a pre-verified user on startup so
	// developers don't have to register/verify manually in the dev env.
	SeedUserEmail       string
	SeedUserPassword    string
	SeedUserDisplayName string
	SeedUserCity        string

	// SeedDemo, when true, populates the demo data set (10 users, ~22 items,
	// sample requests + S3 placeholders). Idempotent — existing rows are
	// skipped. Off by default so prod / preprod don't accidentally seed.
	SeedDemo bool

	// SMTP Configuration
	MailFrom     string
	MailPassword string
	MailSMTPHost string
	MailSMTPPort int

	// SMS Configuration
	SMSWebhookURL   string
	SMSWebhookToken string

	AnalyticsEnabled        bool
	AnalyticsFirehoseStream string
	AnalyticsSchemaVersion  int
}

func Load() Config {
	appEnv := env("APP_ENV", "dev")
	return Config{
		Environment:   appEnv,
		ServiceName:   env("SERVICE_NAME", "kindred-server"),
		AWSRegion:     env("AWS_REGION", "eu-central-1"),
		UsersTable:    env("USERS_TABLE", "kindred-server-dev-users"),
		AppTable:      env("APP_TABLE", "kindred-server-dev-app"),
		UploadsBucket: env("UPLOADS_BUCKET", ""),
		S3Endpoint:    env("S3_ENDPOINT", ""),
		S3AccessKeyID: env("S3_ACCESS_KEY_ID", ""),
		S3SecretKey:   env("S3_SECRET_ACCESS_KEY", ""),

		UploadURLTTL:            time.Duration(envInt("UPLOAD_URL_TTL_SECONDS", 900)) * time.Second,
		DownloadURLTTL:          time.Duration(envInt("DOWNLOAD_URL_TTL_SECONDS", 3600)) * time.Second,
		TokenTTL:                time.Duration(envInt("TOKEN_TTL_SECONDS", 1800)) * time.Second,
		RefreshTokenIdleTTL:     time.Duration(envInt("REFRESH_TOKEN_IDLE_TTL_SECONDS", 2592000)) * time.Second,
		RefreshTokenAbsoluteTTL: time.Duration(envInt("REFRESH_TOKEN_ABSOLUTE_TTL_SECONDS", 7776000)) * time.Second,
		LocalHTTPAddr:           env("LOCAL_HTTP_ADDR", ""),

		JWTPrivateKeyPEM:              env("JWT_PRIVATE_KEY_PEM", ""),
		JWTPrivateKeySSMPath:          env("JWT_PRIVATE_KEY_SSM_PATH", ""),
		JWTPrivateKeyPath:             env("JWT_PRIVATE_KEY_PATH", ""),
		JWTIssuer:                     env("JWT_ISSUER", ""),
		JWTAudience:                   env("JWT_AUDIENCE", "kindred-mobile"),
		TransparencySigningKeyBase64:  env("TRANSPARENCY_SIGNING_KEY_B64", ""),
		TransparencySigningKeySSMPath: env("TRANSPARENCY_SIGNING_KEY_SSM_PATH", ""),

		DynamoDBEndpoint: env("DYNAMODB_ENDPOINT", ""),

		VerificationTokenTTL:        time.Duration(envInt("EMAIL_VERIFICATION_TTL_SECONDS", 86400)) * time.Second,
		PasswordResetTokenTTL:       time.Duration(envInt("PASSWORD_RESET_TTL_SECONDS", 3600)) * time.Second,
		MaxFailedLoginAttempts:      envInt("MAX_FAILED_LOGIN_ATTEMPTS", 5),
		LockoutDuration:             time.Duration(envInt("LOCKOUT_DURATION_SECONDS", 900)) * time.Second,
		RequireVerifiedEmail:        envBool("REQUIRE_VERIFIED_EMAIL", false),
		MobileDeepLinkScheme:        env("MOBILE_DEEP_LINK_SCHEME", "kindred"),
		PhoneVerificationTTL:        time.Duration(envInt("PHONE_VERIFICATION_TTL_SECONDS", 600)) * time.Second,
		ReturnPhoneVerificationCode: envBool("RETURN_PHONE_VERIFICATION_CODE", appEnv == "dev"),

		ThreadBasePrice:                envInt("THREAD_BASE_PRICE", 10),
		ThreadDemandStep:               envInt("THREAD_DEMAND_STEP", 10),
		ThreadDemandMultiplierStepPct:  envInt("THREAD_DEMAND_MULTIPLIER_STEP_PCT", 10),
		ThreadMinCost:                  envInt("THREAD_MIN_COST", 8),
		ThreadMaxCost:                  envInt("THREAD_MAX_COST", 15),
		DeliveryBaseReward:             envInt("DELIVERY_BASE_REWARD", 50),
		DeliveryRatingBonusPerStar:     envInt("DELIVERY_RATING_BONUS_PER_STAR", 5),
		DeliveryMinRewardRating:        envInt("DELIVERY_MIN_REWARD_RATING", 3),
		RepeatRecipientCooldownDays:    envInt("REPEAT_RECIPIENT_REWARD_COOLDOWN_DAYS", 30),
		NewAccountActiveThreadLimit:    envInt("NEW_ACCOUNT_ACTIVE_THREAD_LIMIT", 1),
		ActiveThreadLimit:              envInt("ACTIVE_THREAD_LIMIT", 5),
		NewAccountAgeDays:              envInt("NEW_ACCOUNT_AGE_DAYS", 7),
		MessageThreadMaxServerMessages: envInt("MESSAGE_THREAD_MAX_SERVER_MESSAGES", 1000),
		MessageClosedRetentionDays:     envInt("MESSAGE_CLOSED_RETENTION_DAYS", 30),
		StoreKitBundleID:               env("STOREKIT_BUNDLE_ID", ""),
		StoreKitPremiumMonthlyProductID: env(
			"STOREKIT_PREMIUM_MONTHLY_PRODUCT_ID",
			"com.radlof.kindred.premium.monthly_sub",
		),
		StoreKitEnvironment:        env("STOREKIT_ENVIRONMENT", defaultStoreKitEnvironment(appEnv)),
		StoreKitEnableOnlineChecks: envBool("STOREKIT_ENABLE_ONLINE_CHECKS", defaultStoreKitOnlineChecks(appEnv)),

		RateLimitMax:    envInt("AUTH_RATE_LIMIT_MAX", 20),
		RateLimitWindow: time.Duration(envInt("AUTH_RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second,
		RateLimitTable:  env("RATE_LIMIT_TABLE", ""),

		SeedUserEmail:       env("SEED_USER_EMAIL", ""),
		SeedUserPassword:    env("SEED_USER_PASSWORD", ""),
		SeedUserDisplayName: env("SEED_USER_DISPLAY_NAME", ""),
		SeedUserCity:        env("SEED_USER_CITY", ""),

		SeedDemo: envBool("SEED_DEMO", false),

		MailFrom:     env("MAIL_FROM", ""),
		MailPassword: env("MAIL_PASSWORD", ""),
		MailSMTPHost: env("MAIL_SMTP_HOST", "smtp.gmail.com"),
		MailSMTPPort: envInt("MAIL_SMTP_PORT", 587),

		SMSWebhookURL:   env("SMS_WEBHOOK_URL", ""),
		SMSWebhookToken: env("SMS_WEBHOOK_TOKEN", ""),

		AnalyticsEnabled:        envBool("ANALYTICS_ENABLED", false),
		AnalyticsFirehoseStream: env("ANALYTICS_FIREHOSE_STREAM", ""),
		AnalyticsSchemaVersion:  envInt("ANALYTICS_SCHEMA_VERSION", 1),
	}
}

func defaultStoreKitEnvironment(appEnv string) string {
	if appEnv == "prod" {
		return "Production"
	}
	return "Sandbox"
}

func defaultStoreKitOnlineChecks(appEnv string) bool {
	return appEnv == "prod" || appEnv == "preprod"
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
