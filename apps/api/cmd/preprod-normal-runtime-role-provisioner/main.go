// Command preprod-normal-runtime-role-provisioner creates the disposable
// local-preprod login used by the long-lived API and worker. It
// deliberately runs with the bootstrap owner credential once, after the
// schema migrations complete; the normal runtime containers never receive
// that credential.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

const (
	bootstrapOwnerRole = "aviasurveil360_preprod_loader"
	normalRuntimeRole  = "preprod_normal_api"
	databaseName       = "aviasurveil360_local_preprod"
)

// normalRuntimeRoleDDL removes implicit public privileges and gives the
// non-owner runtime account only the DML/function privileges required by the
// normal application. It is intentionally not a migration: both the role
// password and the local-preprod lifecycle are disposable runtime concerns.
const normalRuntimeRoleDDL = `
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'preprod_normal_api') THEN
    CREATE ROLE preprod_normal_api LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
  END IF;
END $$;
ALTER ROLE preprod_normal_api LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
REVOKE aviasurveil360_preprod_loader FROM preprod_normal_api;
REVOKE ALL PRIVILEGES ON DATABASE aviasurveil360_local_preprod FROM PUBLIC;
REVOKE ALL PRIVILEGES ON DATABASE aviasurveil360_local_preprod FROM preprod_normal_api;
GRANT CONNECT ON DATABASE aviasurveil360_local_preprod TO preprod_normal_api;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM preprod_normal_api;
GRANT USAGE ON SCHEMA public TO preprod_normal_api;
REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM PUBLIC;
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM PUBLIC;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO preprod_normal_api;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO preprod_normal_api;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO preprod_normal_api;
ALTER DEFAULT PRIVILEGES FOR ROLE aviasurveil360_preprod_loader IN SCHEMA public REVOKE ALL PRIVILEGES ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE aviasurveil360_preprod_loader IN SCHEMA public REVOKE ALL PRIVILEGES ON SEQUENCES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE aviasurveil360_preprod_loader IN SCHEMA public REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE aviasurveil360_preprod_loader IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO preprod_normal_api;
ALTER DEFAULT PRIVILEGES FOR ROLE aviasurveil360_preprod_loader IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO preprod_normal_api;
ALTER DEFAULT PRIVILEGES FOR ROLE aviasurveil360_preprod_loader IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO preprod_normal_api;
`

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		slog.Error("local-preprod normal runtime role provisioner failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, arguments []string) error {
	command, err := parseCommand(arguments)
	if err != nil {
		return err
	}
	ownerPassword, err := readSecret("/run/secrets/preprod_app_database_password")
	if err != nil {
		return err
	}
	normalPassword, err := readSecret("/run/secrets/preprod_normal_api_database_password")
	if err != nil {
		return err
	}
	ownerPool, err := database.Open(ctx, databaseURL(bootstrapOwnerRole, ownerPassword))
	if err != nil {
		return fmt.Errorf("open local-preprod bootstrap PostgreSQL connection: %w", err)
	}
	defer ownerPool.Close()

	if command == "provision" {
		if err := provision(ctx, ownerPool, normalPassword); err != nil {
			return err
		}
		fmt.Println("local-preprod normal runtime role provisioned")
		return nil
	}
	return verifyLeastPrivilege(ctx, normalPassword)
}

func parseCommand(arguments []string) (string, error) {
	if len(arguments) == 0 {
		return "provision", nil
	}
	if len(arguments) == 1 && arguments[0] == "verify-least-privilege" {
		return arguments[0], nil
	}
	return "", fmt.Errorf("usage: preprod-normal-runtime-role-provisioner [verify-least-privilege]")
}

func provision(ctx context.Context, pool *database.Pool, normalPassword string) error {
	if pool == nil {
		return fmt.Errorf("local-preprod bootstrap pool is required")
	}
	if strings.TrimSpace(normalPassword) == "" {
		return fmt.Errorf("local-preprod normal runtime password is required")
	}
	return database.WithinTransaction(ctx, pool, func(ctx context.Context, transaction pgx.Tx) error {
		if _, err := transaction.Exec(ctx, normalRuntimeRoleDDL); err != nil {
			return fmt.Errorf("apply local-preprod normal runtime role contract: %w", err)
		}
		var quotedPassword string
		if err := transaction.QueryRow(ctx, "SELECT quote_literal($1)", normalPassword).Scan(&quotedPassword); err != nil {
			return fmt.Errorf("quote local-preprod normal runtime password: %w", err)
		}
		if _, err := transaction.Exec(ctx, "ALTER ROLE "+normalRuntimeRole+" PASSWORD "+quotedPassword); err != nil {
			return fmt.Errorf("set local-preprod normal runtime password: %w", err)
		}
		return nil
	})
}

func verifyLeastPrivilege(ctx context.Context, normalPassword string) error {
	normalPool, err := database.Open(ctx, databaseURL(normalRuntimeRole, normalPassword))
	if err != nil {
		return fmt.Errorf("open local-preprod normal runtime privilege probe: %w", err)
	}
	defer normalPool.Close()
	var currentRole string
	if err := normalPool.QueryRow(ctx, "SELECT current_user").Scan(&currentRole); err != nil {
		return fmt.Errorf("read local-preprod normal runtime current role: %w", err)
	}
	if currentRole != normalRuntimeRole {
		return fmt.Errorf("local-preprod normal runtime connected as %q, want %q", currentRole, normalRuntimeRole)
	}
	var migrationCount int
	if err := normalPool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&migrationCount); err != nil {
		return fmt.Errorf("normal runtime schema read privilege is missing: %w", err)
	}
	if migrationCount == 0 {
		return fmt.Errorf("normal runtime privilege probe requires completed migrations")
	}
	for _, statement := range []string{
		"CREATE TABLE public.preprod_normal_runtime_privilege_probe (id integer)",
		"SET ROLE aviasurveil360_preprod_loader",
	} {
		if err := mustFail(ctx, normalPool, statement); err != nil {
			return fmt.Errorf("local-preprod normal runtime least privilege: %w", err)
		}
	}
	fmt.Printf("local-preprod normal runtime least privilege verified: migrations=%d\n", migrationCount)
	return nil
}

func mustFail(ctx context.Context, pool *database.Pool, statement string) error {
	if _, err := pool.Exec(ctx, statement); err == nil {
		return fmt.Errorf("unexpectedly permitted operation")
	}
	return nil
}

func databaseURL(role, password string) string {
	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(role, password),
		Host:     net.JoinHostPort("preprod-postgres", "5432"),
		Path:     databaseName,
		RawQuery: "sslmode=disable",
	}).String()
}

func readSecret(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read local-preprod role bootstrap secret: %w", err)
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return "", fmt.Errorf("local-preprod role bootstrap secret is empty")
	}
	return value, nil
}
