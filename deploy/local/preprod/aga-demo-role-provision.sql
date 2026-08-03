-- Disposable local-preprod bootstrap only. This is not a production migration.
-- The bootstrap principal runs this file before either API or AGA loader starts.

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

CREATE SCHEMA IF NOT EXISTS preprod_aga_demo AUTHORIZATION preprod_aga_demo_owner;
REVOKE ALL ON SCHEMA preprod_aga_demo FROM PUBLIC, preprod_normal_api, preprod_aga_demo_reader, preprod_aga_demo_writer;
GRANT USAGE ON SCHEMA preprod_aga_demo TO preprod_aga_demo_reader;
GRANT USAGE, CREATE ON SCHEMA preprod_aga_demo TO preprod_aga_demo_writer;
ALTER ROLE preprod_aga_demo_reader SET default_transaction_read_only = on;
