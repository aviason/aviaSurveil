package agademoworkspace

import (
	"fmt"
	"strings"
)

// ValidateWorkspaceDatabaseUser is called before a runtime pool is opened.
// The reader and command credentials are intentionally not interchangeable.
func ValidateWorkspaceDatabaseUser(user string, command bool) error {
	want := WorkspaceReaderRole
	if command {
		want = WorkspaceCommandRole
	}
	if user != want {
		return fmt.Errorf("AGA demo workspace requires the dedicated %s role", want)
	}
	return nil
}

func WorkspaceRoleDDL() string {
	return strings.TrimSpace(fmt.Sprintf(`
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '%s') THEN
    CREATE ROLE %s NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '%s') THEN
    CREATE ROLE %s LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '%s') THEN
    CREATE ROLE %s LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '%s') THEN
    CREATE ROLE %s LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '%s') THEN
    CREATE ROLE %s LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
  END IF;
END $$;
ALTER ROLE %s SET default_transaction_read_only = on;
ALTER ROLE %s SET default_transaction_read_only = off;
GRANT CONNECT ON DATABASE aviasurveil360_local_preprod TO %s, %s, %s, %s;
REVOKE CREATE, TEMP ON DATABASE aviasurveil360_local_preprod FROM %s, %s, %s, %s;
`, WorkspaceOwnerRole, WorkspaceOwnerRole,
		WorkspaceExporterRole, WorkspaceExporterRole,
		WorkspaceLoaderRole, WorkspaceLoaderRole,
		WorkspaceReaderRole, WorkspaceReaderRole,
		WorkspaceCommandRole, WorkspaceCommandRole,
		WorkspaceExporterRole, WorkspaceLoaderRole,
		WorkspaceExporterRole, WorkspaceLoaderRole, WorkspaceReaderRole, WorkspaceCommandRole,
		WorkspaceExporterRole, WorkspaceLoaderRole, WorkspaceReaderRole, WorkspaceCommandRole))
}

// WorkspaceRuntimeGrantDDL is intentionally explicit: the command role gets
// EXECUTE on named functions only, and the reader gets SELECT on projection
// views only. No wildcard grant is used because it would silently widen the
// later Task 9 privilege inventory.
func WorkspaceRuntimeGrantDDL() string {
	return strings.TrimSpace(fmt.Sprintf(`
GRANT USAGE ON SCHEMA %s TO %s, %s, %s;
REVOKE ALL ON ALL TABLES IN SCHEMA %s FROM PUBLIC, %s, %s, %s;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA %s FROM PUBLIC, %s, %s, %s;
GRANT SELECT ON %s.sealed_generations, %s.sealed_classification_items,
  %s.sealed_drafts, %s.sealed_authority_bindings, %s.sealed_provider_scopes,
  %s.sealed_recommendations, %s.sealed_lifecycle_projection
  TO %s;
GRANT EXECUTE ON FUNCTION %s.workspace_query(jsonb),
  %s.workspace_command(jsonb), %s.workspace_reset(jsonb)
  TO %s;
GRANT EXECUTE ON FUNCTION %s.workspace_query(jsonb) TO %s, %s;
GRANT EXECUTE ON FUNCTION %s.workspace_load(jsonb) TO %s;
`, WorkspaceSchemaName,
		WorkspaceReaderRole, WorkspaceCommandRole, WorkspaceLoaderRole,
		WorkspaceSchemaName, WorkspaceExporterRole, WorkspaceLoaderRole, WorkspaceReaderRole,
		WorkspaceSchemaName, WorkspaceExporterRole, WorkspaceLoaderRole, WorkspaceReaderRole,
		WorkspaceSchemaName, WorkspaceSchemaName, WorkspaceSchemaName,
		WorkspaceSchemaName, WorkspaceSchemaName, WorkspaceSchemaName,
		WorkspaceSchemaName, WorkspaceReaderRole,
		WorkspaceSchemaName, WorkspaceSchemaName, WorkspaceSchemaName, WorkspaceCommandRole,
		WorkspaceSchemaName, WorkspaceReaderRole, WorkspaceCommandRole,
		WorkspaceSchemaName, WorkspaceLoaderRole,
	))
}
