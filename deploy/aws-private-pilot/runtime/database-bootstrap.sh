#!/bin/sh
set -eu

read_secret() {
  value=$(tr -d '\r\n' <"$1")
  if [ -z "$value" ]; then
    echo "required database bootstrap credential is unavailable" >&2
    exit 1
  fi
  printf '%s' "$value"
}

PGPASSWORD=$(read_secret /run/secrets/database_bootstrap_password)
export PGPASSWORD
if [ ! -r "${PGSSLROOTCERT:?required reviewed RDS CA bundle path}" ]; then
  echo "reviewed RDS CA bundle is unavailable" >&2
  exit 1
fi

mode=${AVIA_DATABASE_BOOTSTRAP_MODE:-prepare}
case "$mode" in
  prepare)
    AVIA_BOOTSTRAP_APP_PASSWORD=$(read_secret /run/secrets/app_database_password)
    AVIA_BOOTSTRAP_KEYCLOAK_PASSWORD=$(read_secret /run/secrets/keycloak_database_password)
    export AVIA_BOOTSTRAP_APP_PASSWORD AVIA_BOOTSTRAP_KEYCLOAK_PASSWORD

    timeout 120 psql --no-psqlrc --set=ON_ERROR_STOP=1 <<'SQL'
\getenv app_password AVIA_BOOTSTRAP_APP_PASSWORD
\getenv keycloak_password AVIA_BOOTSTRAP_KEYCLOAK_PASSWORD
SELECT 'CREATE ROLE aviasurveil360_owner NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT'
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'aviasurveil360_owner') \gexec
SELECT 'CREATE ROLE keycloak_owner NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT'
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'keycloak_owner') \gexec

SELECT format('CREATE ROLE aviasurveil360_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT PASSWORD %L', :'app_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'aviasurveil360_runtime') \gexec
SELECT format('ALTER ROLE aviasurveil360_runtime LOGIN PASSWORD %L', :'app_password') \gexec

SELECT 'CREATE ROLE aviasurveil360_migrator NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT'
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'aviasurveil360_migrator') \gexec
ALTER ROLE aviasurveil360_migrator NOLOGIN PASSWORD NULL;

SELECT format('CREATE ROLE keycloak_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT PASSWORD %L', :'keycloak_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'keycloak_runtime') \gexec
SELECT format('ALTER ROLE keycloak_runtime LOGIN PASSWORD %L', :'keycloak_password') \gexec

SELECT 'CREATE DATABASE aviasurveil360 OWNER aviasurveil360_owner'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'aviasurveil360') \gexec
SELECT 'CREATE DATABASE keycloak OWNER keycloak_owner'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'keycloak') \gexec

ALTER DATABASE aviasurveil360 OWNER TO aviasurveil360_owner;
ALTER DATABASE keycloak OWNER TO keycloak_owner;
REVOKE CONNECT ON DATABASE aviasurveil360 FROM PUBLIC, keycloak_runtime;
REVOKE CONNECT ON DATABASE keycloak FROM PUBLIC, aviasurveil360_runtime, aviasurveil360_migrator;
GRANT CONNECT ON DATABASE aviasurveil360 TO aviasurveil360_runtime, aviasurveil360_migrator;
GRANT CONNECT ON DATABASE keycloak TO keycloak_runtime;
SQL

    timeout 120 psql --no-psqlrc --set=ON_ERROR_STOP=1 --dbname=aviasurveil360 <<'SQL'
REVOKE ALL ON SCHEMA public FROM PUBLIC;
ALTER SCHEMA public OWNER TO aviasurveil360_migrator;
GRANT USAGE ON SCHEMA public TO aviasurveil360_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE aviasurveil360_migrator IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO aviasurveil360_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE aviasurveil360_migrator IN SCHEMA public
  GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO aviasurveil360_runtime;
SQL

    timeout 120 psql --no-psqlrc --set=ON_ERROR_STOP=1 --dbname=keycloak <<'SQL'
REVOKE ALL ON SCHEMA public FROM PUBLIC;
ALTER SCHEMA public OWNER TO keycloak_runtime;
GRANT USAGE, CREATE ON SCHEMA public TO keycloak_runtime;
SQL
    ;;
  migration-enable)
    AVIA_BOOTSTRAP_MIGRATION_PASSWORD=$(read_secret /run/secrets/app_migration_password)
    export AVIA_BOOTSTRAP_MIGRATION_PASSWORD
    timeout 120 psql --no-psqlrc --set=ON_ERROR_STOP=1 <<'SQL'
\getenv migration_password AVIA_BOOTSTRAP_MIGRATION_PASSWORD
SELECT format('ALTER ROLE aviasurveil360_migrator LOGIN PASSWORD %L', :'migration_password') \gexec
SQL
    ;;
  lockdown)
    timeout 120 psql --no-psqlrc --set=ON_ERROR_STOP=1 <<'SQL'
ALTER ROLE aviasurveil360_migrator NOLOGIN PASSWORD NULL;
SQL
    ;;
  *)
    echo "unsupported database bootstrap mode" >&2
    exit 64
    ;;
esac

unset PGPASSWORD AVIA_BOOTSTRAP_APP_PASSWORD AVIA_BOOTSTRAP_MIGRATION_PASSWORD AVIA_BOOTSTRAP_KEYCLOAK_PASSWORD value mode
