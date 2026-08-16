// Command runtime-database-bootstrap reconciles the small, fixed PostgreSQL
// boundary used by the cloud runtime. It is deliberately a source-controlled
// bootstrap command: the host release lifecycle never executes psql or edits
// database state outside this image.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

const (
	surveilDatabase = "surveil"
	authDatabase    = "auth"
)

type roleSpec struct {
	name     string
	password string
}

func main() {
	var err error
	if len(os.Args) > 1 && os.Args[1] == "permissions" {
		err = runPermissions(context.Background(), os.Getenv, os.ReadFile, os.Stdout)
	} else if len(os.Args) > 1 {
		err = fmt.Errorf("unsupported runtime database bootstrap command")
	} else {
		err = run(context.Background(), os.Getenv, os.ReadFile, os.Stdout)
	}
	if err != nil {
		// Do not include a database URL, password, or provider response in this
		// message. The systemd unit records only this bounded diagnostic.
		fmt.Fprintln(os.Stderr, "runtime database bootstrap:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, lookup func(string) string, readFile func(string) ([]byte, error), output io.Writer) error {
	masterURL, err := requiredSecretFile(lookup("AVIA_DATABASE_MASTER_URL_FILE"), readFile)
	if err != nil {
		return err
	}
	passwords := map[string]string{}
	for name, environment := range map[string]string{
		"surveil_runtime":   "AVIA_DATABASE_RUNTIME_PASSWORD_FILE",
		"surveil_migration": "AVIA_DATABASE_MIGRATION_PASSWORD_FILE",
		"surveil_bootstrap": "AVIA_DATABASE_BOOTSTRAP_PASSWORD_FILE",
		"auth_runtime":      "AVIA_AUTH_DATABASE_PASSWORD_FILE",
	} {
		passwords[name], err = requiredSecretFile(lookup(environment), readFile)
		if err != nil {
			return fmt.Errorf("read database credential %s: %w", name, err)
		}
	}

	master, err := database.Open(ctx, masterURL)
	if err != nil {
		return fmt.Errorf("open managed PostgreSQL bootstrap connection: %w", err)
	}
	defer master.Close()
	if err := waitForDatabase(ctx, master); err != nil {
		return err
	}
	for _, role := range []roleSpec{
		{name: "surveil_runtime", password: passwords["surveil_runtime"]},
		{name: "surveil_migration", password: passwords["surveil_migration"]},
		{name: "surveil_bootstrap", password: passwords["surveil_bootstrap"]},
		{name: "auth_runtime", password: passwords["auth_runtime"]},
	} {
		if err := ensureLoginRole(ctx, master, role); err != nil {
			return err
		}
	}
	for _, name := range []string{surveilDatabase, authDatabase} {
		if err := ensureDatabase(ctx, master, name); err != nil {
			return err
		}
	}
	if err := configureSurveil(ctx, masterURL, master, passwords["surveil_migration"]); err != nil {
		return err
	}
	if err := configureAuth(ctx, masterURL, master); err != nil {
		return err
	}
	if err := verifyCredentials(ctx, masterURL, passwords); err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, "runtime database boundary reconciled: databases=2 roles=4")
	return err
}

func runPermissions(ctx context.Context, lookup func(string) string, readFile func(string) ([]byte, error), output io.Writer) error {
	masterURL, err := requiredSecretFile(lookup("AVIA_DATABASE_MASTER_URL_FILE"), readFile)
	if err != nil {
		return err
	}
	migrationPassword, err := requiredSecretFile(lookup("AVIA_DATABASE_MIGRATION_PASSWORD_FILE"), readFile)
	if err != nil {
		return fmt.Errorf("read database credential surveil_migration: %w", err)
	}
	master, err := database.Open(ctx, masterURL)
	if err != nil {
		return fmt.Errorf("open managed PostgreSQL permission connection: %w", err)
	}
	defer master.Close()
	if err := waitForDatabase(ctx, master); err != nil {
		return err
	}
	if err := configureSurveilWithTableGrants(ctx, masterURL, master, migrationPassword, true); err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, "runtime database table permissions reconciled")
	return err
}

func waitForDatabase(ctx context.Context, pool *database.Pool) error {
	deadline := time.Now().Add(10 * time.Minute)
	for {
		if err := pool.Ping(ctx); err == nil {
			return nil
		} else if time.Now().After(deadline) {
			return fmt.Errorf("managed PostgreSQL did not become ready")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

func ensureLoginRole(ctx context.Context, pool *database.Pool, role roleSpec) error {
	var exists, canLogin, superuser, createDatabase, createRole, inherit, replication, bypassRLS bool
	err := pool.QueryRow(ctx, `
		SELECT true, rolcanlogin, rolsuper, rolcreatedb, rolcreaterole,
		       rolinherit, rolreplication, rolbypassrls
		  FROM pg_roles WHERE rolname=$1`, role.name).Scan(
		&exists, &canLogin, &superuser, &createDatabase, &createRole, &inherit, &replication, &bypassRLS)
	if errors.Is(err, pgx.ErrNoRows) {
		quoted, quoteErr := quoteLiteral(ctx, pool, role.password)
		if quoteErr != nil {
			return fmt.Errorf("quote credential for %s: %w", role.name, quoteErr)
		}
		if _, err := pool.Exec(ctx, "CREATE ROLE "+role.name+" LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS PASSWORD "+quoted); err != nil {
			return fmt.Errorf("create fixed database role %s: %w", role.name, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect fixed database role %s: %w", role.name, err)
	}
	if !exists || !canLogin || superuser || createDatabase || createRole || inherit || replication || bypassRLS {
		return fmt.Errorf("fixed database role %s privilege drifted", role.name)
	}
	// Existing passwords are never rotated by a release. They are checked by
	// verifyCredentials below, so a stale secret fails closed.
	return nil
}

func ensureDatabase(ctx context.Context, pool *database.Pool, name string) error {
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname=$1)`, name).Scan(&exists); err != nil {
		return fmt.Errorf("inspect database %s: %w", name, err)
	}
	if exists {
		return nil
	}
	if _, err := pool.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		return fmt.Errorf("create database %s: %w", name, err)
	}
	return nil
}

func configureSurveil(ctx context.Context, masterURL string, master *database.Pool, migrationPassword string) error {
	return configureSurveilWithTableGrants(ctx, masterURL, master, migrationPassword, false)
}

func configureSurveilWithTableGrants(ctx context.Context, masterURL string, master *database.Pool, migrationPassword string, grantLoaderTables bool) error {
	pool, err := openDatabaseURL(ctx, masterURL, surveilDatabase)
	if err != nil {
		return err
	}
	defer pool.Close()
	if _, err := master.Exec(ctx, `REVOKE ALL ON DATABASE surveil FROM PUBLIC`); err != nil {
		return fmt.Errorf("revoke public access to surveil: %w", err)
	}
	if _, err := master.Exec(ctx, `GRANT CONNECT ON DATABASE surveil TO surveil_runtime, surveil_migration, surveil_bootstrap`); err != nil {
		return fmt.Errorf("grant surveil database access: %w", err)
	}
	if _, err := master.Exec(ctx, `GRANT CREATE, TEMPORARY ON DATABASE surveil TO surveil_migration`); err != nil {
		return fmt.Errorf("grant migration database access: %w", err)
	}
	if _, err := pool.Exec(ctx, `REVOKE CREATE ON SCHEMA public FROM PUBLIC`); err != nil {
		return fmt.Errorf("revoke public schema creation: %w", err)
	}
	if _, err := pool.Exec(ctx, `GRANT USAGE ON SCHEMA public TO surveil_runtime, surveil_bootstrap`); err != nil {
		return fmt.Errorf("grant runtime schema usage: %w", err)
	}
	if _, err := pool.Exec(ctx, `GRANT USAGE, CREATE ON SCHEMA public TO surveil_migration`); err != nil {
		return fmt.Errorf("grant migration schema usage: %w", err)
	}
	for _, statement := range []string{
		`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO surveil_runtime`,
		`GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO surveil_runtime`,
		`GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO surveil_runtime`,
		`REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM surveil_bootstrap`,
		`REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM surveil_bootstrap`,
		`REVOKE ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public FROM surveil_bootstrap`,
		`REVOKE ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC`,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			return fmt.Errorf("configure surveil database privileges: %w", err)
		}
	}

	// RDS master users can manage the fixed database boundary but are not
	// PostgreSQL superusers. ALTER DEFAULT PRIVILEGES FOR ROLE ... must run as
	// the owning role itself, so use the migration role for only that narrow
	// operation set.
	migrationPool, err := openMigrationDatabaseURL(ctx, masterURL, migrationPassword)
	if err != nil {
		return err
	}
	defer migrationPool.Close()
	for _, statement := range []string{
		`ALTER DEFAULT PRIVILEGES FOR ROLE surveil_migration IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO surveil_runtime`,
		`ALTER DEFAULT PRIVILEGES FOR ROLE surveil_migration IN SCHEMA public GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO surveil_runtime`,
		`ALTER DEFAULT PRIVILEGES FOR ROLE surveil_migration IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO surveil_runtime`,
		`ALTER DEFAULT PRIVILEGES FOR ROLE surveil_migration IN SCHEMA public REVOKE ALL ON TABLES FROM surveil_bootstrap`,
		`ALTER DEFAULT PRIVILEGES FOR ROLE surveil_migration IN SCHEMA public REVOKE ALL ON SEQUENCES FROM surveil_bootstrap`,
		`ALTER DEFAULT PRIVILEGES FOR ROLE surveil_migration IN SCHEMA public REVOKE ALL ON FUNCTIONS FROM surveil_bootstrap`,
		`ALTER DEFAULT PRIVILEGES FOR ROLE surveil_migration IN SCHEMA public REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC`,
	} {
		if _, err := migrationPool.Exec(ctx, statement); err != nil {
			return fmt.Errorf("configure surveil default privileges: %w", err)
		}
	}
	if !grantLoaderTables {
		return nil
	}
	bootstrapTables := []string{
		"identity_references", "organizations", "regulated_targets",
		"organization_service_provider_scopes", "organization_service_provider_scope_targets",
		"service_provider_types", "user_profiles", "user_settings", "user_lifecycle_requests",
		"desired_membership_versions", "desired_membership_sync", "caa_department_memberships",
		"canonical_question_catalogs", "canonical_question_catalog_forms", "question_versions",
		"canonical_question_version_provenance", "canonical_question_catalog_memberships",
		"canonical_question_catalog_membership_events", "canonical_question_catalog_applicabilities",
		"canonical_question_catalog_import_runs", "canonical_question_catalog_ai_enrichments",
	}
	for _, table := range bootstrapTables {
		if _, err := pool.Exec(ctx, "GRANT SELECT, INSERT ON TABLE public."+table+" TO surveil_bootstrap"); err != nil {
			return fmt.Errorf("grant bounded bootstrap table privilege %s: %w", table, err)
		}
	}
	if _, err := pool.Exec(ctx, `GRANT EXECUTE ON FUNCTION public.governed_sha256(text) TO surveil_bootstrap`); err != nil {
		return fmt.Errorf("grant bounded bootstrap digest validator privilege: %w", err)
	}
	return nil
}

func openMigrationDatabaseURL(ctx context.Context, masterURL, password string) (*database.Pool, error) {
	databaseURL := migrationDatabaseURL(masterURL, password)
	if databaseURL == "" {
		return nil, fmt.Errorf("managed PostgreSQL migration URL is malformed")
	}
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open managed PostgreSQL migration role: %w", err)
	}
	return pool, nil
}

func migrationDatabaseURL(masterURL, password string) string {
	return roleURL(masterURL, "surveil_migration", password, surveilDatabase)
}

func configureAuth(ctx context.Context, masterURL string, master *database.Pool) error {
	if _, err := master.Exec(ctx, `REVOKE ALL ON DATABASE auth FROM PUBLIC`); err != nil {
		return fmt.Errorf("revoke public access to auth: %w", err)
	}
	if _, err := master.Exec(ctx, `GRANT CONNECT, CREATE, TEMPORARY ON DATABASE auth TO auth_runtime`); err != nil {
		return fmt.Errorf("grant auth database access: %w", err)
	}
	pool, err := openDatabaseURL(ctx, masterURL, authDatabase)
	if err != nil {
		return err
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `REVOKE CREATE ON SCHEMA public FROM PUBLIC`); err != nil {
		return fmt.Errorf("revoke public auth schema creation: %w", err)
	}
	return nil
}

func verifyCredentials(ctx context.Context, masterURL string, passwords map[string]string) error {
	for role, databaseName := range map[string]string{
		"surveil_runtime":   surveilDatabase,
		"surveil_migration": surveilDatabase,
		"surveil_bootstrap": surveilDatabase,
		"auth_runtime":      authDatabase,
	} {
		pool, err := database.Open(ctx, roleURL(masterURL, role, passwords[role], databaseName))
		if err != nil {
			return fmt.Errorf("open fixed role %s for credential verification: %w", role, err)
		}
		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			return fmt.Errorf("fixed role %s credential verification failed", role)
		}
		pool.Close()
	}
	return nil
}

func configureDatabaseURL(raw, databaseName string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parsed.Path = "/" + databaseName
	return parsed.String()
}

func openDatabaseURL(ctx context.Context, masterURL, name string) (*database.Pool, error) {
	databaseURL := configureDatabaseURL(masterURL, name)
	if databaseURL == "" {
		return nil, fmt.Errorf("managed PostgreSQL URL is malformed")
	}
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open managed PostgreSQL database %s: %w", name, err)
	}
	return pool, nil
}

func roleURL(masterURL, role, password, databaseName string) string {
	parsed, err := url.Parse(masterURL)
	if err != nil {
		return ""
	}
	parsed.User = url.UserPassword(role, password)
	parsed.Path = "/" + databaseName
	return parsed.String()
}

func quoteLiteral(ctx context.Context, pool *database.Pool, value string) (string, error) {
	var quoted string
	if err := pool.QueryRow(ctx, `SELECT quote_literal($1)`, value).Scan(&quoted); err != nil {
		return "", err
	}
	return quoted, nil
}

func requiredSecretFile(path string, readFile func(string) ([]byte, error)) (string, error) {
	if strings.TrimSpace(path) == "" || !strings.HasPrefix(path, "/") || strings.Contains(path, "\x00") {
		return "", fmt.Errorf("required secret file path is missing")
	}
	data, err := readFile(path)
	if err != nil {
		return "", fmt.Errorf("read required secret file: %w", err)
	}
	value := strings.TrimSpace(string(data))
	if value == "" || strings.ContainsAny(value, "\r\n") || len(value) > 16*1024 {
		return "", fmt.Errorf("required secret file is empty or malformed")
	}
	return value, nil
}
