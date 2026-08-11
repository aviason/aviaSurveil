package migrations

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIdentityMigrationContainsFailClosedBoundaries(t *testing.T) {
	for _, required := range []string{
		"CREATE SCHEMA IF NOT EXISTS auth_identity",
		"UNIQUE (normalized_value)",
		"state IN (",
		"'deletion-pending'",
		"subject_id ~ '^usr_[A-Za-z0-9_-]{22}$'",
		"state <> 'active' OR email_verified = true",
		"resend_count >= 0 AND resend_count <= 3",
		"sensitive operations must not proceed",
	} {
		if !strings.Contains(identitySchema, required) {
			t.Errorf("identity schema missing %q", required)
		}
	}
}

func TestSessionMigrationContainsRotationBoundaries(t *testing.T) {
	for _, required := range []string{
		"provider_sessions",
		"refresh_families",
		"refresh_token_history",
		"current_token_hash bytea NOT NULL UNIQUE",
		"state IN ('active', 'revoked', 'reuse-detected', 'expired')",
		"octet_length(fingerprint_hash) = 32",
		"concurrent rotations must lock the family row",
	} {
		if !strings.Contains(sessionsSchema, required) {
			t.Errorf("session schema missing %q", required)
		}
	}
}

func TestProviderAdminAuditMigrationContainsAppendOnlyBoundaries(t *testing.T) {
	for _, required := range []string{
		"provider_clients",
		"provider_signing_keys",
		"private_key_ciphertext",
		"state IN ('active', 'overlap', 'retired')",
		"audit_events",
		"fields jsonb NOT NULL",
		"Append-only redacted security events",
	} {
		if !strings.Contains(providerAdminAuditSchema, required) {
			t.Errorf("provider admin/audit schema missing %q", required)
		}
	}
}

func TestMFAMigrationContainsDurableProtectionBoundaries(t *testing.T) {
	for _, required := range []string{
		"mfa_factors",
		"secret_ciphertext bytea NOT NULL",
		"last_used_counter bigint NOT NULL DEFAULT -1",
		"mfa_recovery_codes",
		"code_hash bytea NOT NULL",
		"raw recovery codes are never persisted",
	} {
		if !strings.Contains(mfaSchema, required) {
			t.Errorf("MFA schema missing %q", required)
		}
	}
}

func TestChallengeMigrationContainsDurableProtectionBoundaries(t *testing.T) {
	for _, required := range []string{
		"identity_challenges",
		"token_hash bytea PRIMARY KEY",
		"'email-verification', 'password-reset', 'mfa-recovery', 'admin-recovery'",
		"attempt_count integer NOT NULL DEFAULT 0",
		"Consume and rejection paths lock rows",
	} {
		if !strings.Contains(challengesSchema, required) {
			t.Errorf("challenge schema missing %q", required)
		}
	}
}

func TestOIDCRuntimeMigrationContainsDurableProtectionBoundaries(t *testing.T) {
	for _, required := range []string{
		"oidc_auth_requests",
		"oidc_authorization_codes",
		"code_hash bytea PRIMARY KEY",
		"oidc_access_tokens",
		"oidc_refresh_tokens",
		"raw refresh credentials are never persisted",
	} {
		if !strings.Contains(oidcRuntimeSchema, required) {
			t.Errorf("OIDC runtime schema missing %q", required)
		}
	}
}

func TestPostgreSQLIdentityMigration(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("AVIA_AUTH_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("not run: AVIA_AUTH_TEST_DATABASE_URL is not configured")
	}
	// The connected database must be a disposable auth-only database. The
	// integration package is intentionally opt-in and never targets the normal
	// application database.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open disposable PostgreSQL: %v", err)
	}
	defer pool.Close()
	if err := Apply(ctx, pool); err != nil {
		t.Fatalf("apply identity migration: %v", err)
	}
	if err := Apply(ctx, pool); err != nil {
		t.Fatalf("reapply identity migration: %v", err)
	}
	var version int64
	if err := pool.QueryRow(ctx, `SELECT MAX(version) FROM auth_identity.schema_migrations`).Scan(&version); err != nil {
		t.Fatalf("read identity migration version: %v", err)
	}
	if version != LatestVersion() {
		t.Fatalf("migration version = %d, want %d", version, LatestVersion())
	}

	transaction, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	subjectID := "usr_" + strings.Repeat("A", 22)
	now := time.Now().UTC()
	if _, err := transaction.Exec(ctx, `INSERT INTO auth_identity.accounts(subject_id, state, password_hash, email_verified, auth_revision, created_at, updated_at) VALUES ($1, 'active', 'argon2id-test', true, 1, $2, $2)`, subjectID, now); err != nil {
		t.Fatalf("insert synthetic identity: %v", err)
	}
	if _, err := transaction.Exec(ctx, `INSERT INTO auth_identity.identifiers(subject_id, identifier_type, normalized_value, verified_at, created_at) VALUES ($1, 'email', 'synthetic-a@example.invalid', $2, $2)`, subjectID, now); err != nil {
		t.Fatalf("insert synthetic identifier: %v", err)
	}
	if _, err := transaction.Exec(ctx, `SAVEPOINT duplicate_identifier`); err != nil {
		t.Fatal(err)
	}
	if _, err := transaction.Exec(ctx, `INSERT INTO auth_identity.identifiers(subject_id, identifier_type, normalized_value, created_at) VALUES ($1, 'username', 'synthetic-a@example.invalid', $2)`, subjectID, now); err == nil {
		t.Fatal("cross-field duplicate identifier was accepted")
	}
	if _, err := transaction.Exec(ctx, `ROLLBACK TO SAVEPOINT duplicate_identifier`); err != nil {
		t.Fatal(err)
	}
}
