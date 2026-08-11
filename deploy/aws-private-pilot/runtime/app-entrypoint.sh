#!/bin/sh
set -eu

read_secret() {
  secret_path=$1
  if [ ! -f "$secret_path" ]; then
    echo "required mounted runtime credential is unavailable" >&2
    exit 1
  fi
  secret_value=$(tr -d '\r\n' <"$secret_path")
  if [ -z "$secret_value" ]; then
    echo "required mounted runtime credential is empty" >&2
    exit 1
  fi
  printf '%s' "$secret_value"
}

role=${1:-}
case "$role" in
  api|worker|migrate) ;;
  *) echo "unsupported private-pilot runtime role" >&2; exit 64 ;;
esac

database_user=${AVIA_DATABASE_USER:-aviasurveil360_runtime}
database_secret=/run/secrets/app_database_password
if [ "$role" = migrate ]; then
  database_user=aviasurveil360_migrator
  database_secret=/run/secrets/app_migration_password
fi
database_password=$(read_secret "$database_secret")
case "$database_password" in
  *[!A-Za-z0-9_-]*)
    echo "database credential must be 32-128 URL-safe characters" >&2
    exit 1
    ;;
esac
if [ "${#database_password}" -lt 32 ] || [ "${#database_password}" -gt 128 ]; then
  echo "database credential must be 32-128 URL-safe characters" >&2
  exit 1
fi
database_sslrootcert=${AVIA_DATABASE_SSLROOTCERT:?required reviewed RDS CA bundle path}
if [ ! -r "$database_sslrootcert" ]; then
  echo "reviewed RDS CA bundle is unavailable" >&2
  exit 1
fi
database_max_connections=${AVIA_DATABASE_MAX_CONNECTIONS:-4}
case "$database_max_connections" in
  ''|*[!0-9]*) echo "database pool limit must be an integer from 1 through 8" >&2; exit 1 ;;
esac
if [ "$database_max_connections" -lt 1 ] || [ "$database_max_connections" -gt 8 ]; then
  echo "database pool limit must be an integer from 1 through 8" >&2
  exit 1
fi
export AVIA_DATABASE_URL="postgres://${database_user}:${database_password}@${AVIA_DATABASE_HOST:?}:${AVIA_DATABASE_PORT:-5432}/${AVIA_DATABASE_NAME:-aviasurveil360}?sslmode=${AVIA_DATABASE_SSLMODE:-verify-full}&sslrootcert=${database_sslrootcert}&pool_max_conns=${database_max_connections}"
unset database_password

case "$role" in
  api|worker)
    AVIA_OIDC_CLIENT_SECRET=$(read_secret /run/secrets/oidc_client_secret)
    AVIA_SESSION_ENCRYPTION_KEY=$(read_secret /run/secrets/session_encryption_key)
    AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET=$(read_secret /run/secrets/keycloak_service_client_secret)
    export AVIA_OIDC_CLIENT_SECRET AVIA_SESSION_ENCRYPTION_KEY AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET
    ;;
esac

unset database_max_connections

if [ "$role" = worker ]; then
  AVIA_SMTP_PASSWORD=$(read_secret /run/secrets/app_smtp_password)
  export AVIA_SMTP_PASSWORD
fi

case "$role" in
  api) exec /app/api ;;
  worker) exec /app/worker ;;
  migrate) exec /app/migrate ;;
esac
