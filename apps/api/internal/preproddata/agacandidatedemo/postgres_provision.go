package agacandidatedemo

import (
	"context"
	"fmt"
	"strings"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

// RolePasswords holds the three distinct local-preprod database credentials.
// It is consumed only by the disposable bootstrap owner, never by the normal
// API, tagged reader, or one-shot writer.
type RolePasswords struct {
	NormalAPI string
	Reader    string
	Writer    string
}

const roleBaselineDDL = `
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'preprod_aga_demo_owner') THEN
    CREATE ROLE preprod_aga_demo_owner NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'preprod_aga_demo_writer') THEN
    CREATE ROLE preprod_aga_demo_writer LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'preprod_aga_demo_reader') THEN
    CREATE ROLE preprod_aga_demo_reader LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'preprod_normal_api') THEN
    CREATE ROLE preprod_normal_api LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
  END IF;
END $$;
REVOKE ALL ON DATABASE aviasurveil360_local_preprod FROM PUBLIC;
GRANT CONNECT ON DATABASE aviasurveil360_local_preprod TO preprod_normal_api, preprod_aga_demo_reader, preprod_aga_demo_writer;
REVOKE ALL ON SCHEMA public FROM preprod_aga_demo_reader, preprod_aga_demo_writer, preprod_normal_api;
GRANT USAGE ON SCHEMA public TO preprod_normal_api;
GRANT SELECT ON schema_migrations, identity_references, user_profiles, desired_membership_versions, desired_membership_sync, session_references, oidc_login_states TO preprod_normal_api;
GRANT SELECT ON caa_department_memberships, caa_department_status_facts, caa_organizational_unit_status_facts, caa_organizational_units TO preprod_normal_api;
GRANT INSERT, DELETE ON oidc_login_states TO preprod_normal_api;
GRANT INSERT, UPDATE ON session_references TO preprod_normal_api;
GRANT UPDATE (display_name, email) ON identity_references TO preprod_normal_api;
GRANT UPDATE (observed_provider_enabled, observed_organization_id, observed_roles, observed_at, drift_state) ON desired_membership_sync TO preprod_normal_api;
GRANT INSERT ON audit_events TO preprod_normal_api;
GRANT USAGE, SELECT ON SEQUENCE audit_events_sequence_id_seq TO preprod_normal_api;
ALTER ROLE preprod_aga_demo_reader SET default_transaction_read_only = on;
`

// ProvisionOverlaySchema creates the empty immutable schema under the
// disposable bootstrap owner and installs distinct login credentials before a
// writer can connect. No governed-domain or provider table is named here.
func ProvisionOverlaySchema(ctx context.Context, pool *database.Pool, passwords RolePasswords) error {
	if pool == nil {
		return fmt.Errorf("AGA demo bootstrap pool is required")
	}
	if strings.TrimSpace(passwords.NormalAPI) == "" || strings.TrimSpace(passwords.Reader) == "" || strings.TrimSpace(passwords.Writer) == "" {
		return fmt.Errorf("AGA demo bootstrap passwords are required")
	}
	return database.WithinTransaction(ctx, pool, func(ctx context.Context, transaction pgx.Tx) error {
		if _, err := transaction.Exec(ctx, roleBaselineDDL); err != nil {
			return fmt.Errorf("create AGA demo roles: %w", err)
		}
		for _, credential := range []struct {
			role     string
			password string
		}{
			{NormalAPIRole, passwords.NormalAPI},
			{OverlayReaderRole, passwords.Reader},
			{OverlayWriterRole, passwords.Writer},
		} {
			var quoted string
			if err := transaction.QueryRow(ctx, "SELECT quote_literal($1)", credential.password).Scan(&quoted); err != nil {
				return fmt.Errorf("quote AGA demo role password: %w", err)
			}
			if _, err := transaction.Exec(ctx, "ALTER ROLE "+credential.role+" PASSWORD "+quoted); err != nil {
				return fmt.Errorf("set AGA demo role password: %w", err)
			}
		}
		if _, err := transaction.Exec(ctx, OverlaySchemaDDL); err != nil {
			return fmt.Errorf("create empty immutable AGA demo schema: %w", err)
		}
		return nil
	})
}
