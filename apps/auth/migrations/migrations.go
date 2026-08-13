package migrations

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed 000001_identity.up.sql
var identitySchema string

//go:embed 000002_sessions.up.sql
var sessionsSchema string

//go:embed 000003_provider_admin_audit.up.sql
var providerAdminAuditSchema string

//go:embed 000004_mail_outbox.up.sql
var mailOutboxSchema string

//go:embed 000005_mfa.up.sql
var mfaSchema string

//go:embed 000006_challenges.up.sql
var challengesSchema string

//go:embed 000007_oidc_runtime.up.sql
var oidcRuntimeSchema string

//go:embed 000008_local_preprod_authority_admin.up.sql
var localPreprodAuthorityAdminSchema string

//go:embed 000009_first_party_auth_security_hardening.up.sql
var firstPartyAuthSecurityHardeningSchema string

const latestVersion int64 = 9

func LatestVersion() int64 {
	return latestVersion
}

func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("identity migration pool is nil")
	}
	transaction, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin identity migration: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if _, err := transaction.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS auth_identity`); err != nil {
		return fmt.Errorf("create identity schema: %w", err)
	}
	if _, err := transaction.Exec(ctx, `CREATE TABLE IF NOT EXISTS auth_identity.schema_migrations (version bigint PRIMARY KEY, applied_at timestamptz NOT NULL)`); err != nil {
		return fmt.Errorf("create identity migration table: %w", err)
	}
	migrations := []struct {
		version int64
		schema  string
		name    string
	}{
		{version: 1, schema: identitySchema, name: "identity"},
		{version: 2, schema: sessionsSchema, name: "sessions"},
		{version: 3, schema: providerAdminAuditSchema, name: "provider-admin-audit"},
		{version: 4, schema: mailOutboxSchema, name: "mail-outbox"},
		{version: 5, schema: mfaSchema, name: "mfa"},
		{version: 6, schema: challengesSchema, name: "challenges"},
		{version: 7, schema: oidcRuntimeSchema, name: "oidc-runtime"},
		{version: 8, schema: localPreprodAuthorityAdminSchema, name: "local-preprod-authority-admin"},
		{version: 9, schema: firstPartyAuthSecurityHardeningSchema, name: "first-party-auth-security-hardening"},
	}
	for _, migration := range migrations {
		var applied bool
		if err := transaction.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM auth_identity.schema_migrations WHERE version = $1)`, migration.version).Scan(&applied); err != nil {
			return fmt.Errorf("read %s migration state: %w", migration.name, err)
		}
		if applied {
			continue
		}
		if _, err := transaction.Exec(ctx, migration.schema); err != nil {
			return fmt.Errorf("apply %s schema: %w", migration.name, err)
		}
		if _, err := transaction.Exec(ctx, `INSERT INTO auth_identity.schema_migrations(version, applied_at) VALUES ($1, now())`, migration.version); err != nil {
			return fmt.Errorf("record %s migration: %w", migration.name, err)
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit identity migration: %w", err)
	}
	return nil
}
