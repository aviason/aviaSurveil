#!/bin/sh
set -eu

read_secret() {
  secret_path=$1
  if [ ! -f "$secret_path" ]; then
    echo "required mounted file is unavailable" >&2
    exit 1
  fi
  secret_value=$(tr -d '\r\n' <"$secret_path")
  if [ -z "$secret_value" ]; then
    echo "required mounted file is empty" >&2
    exit 1
  fi
  printf '%s' "$secret_value"
}

database_password=$(read_secret "${AVIA_DATABASE_PASSWORD_FILE:-/run/secrets/app_database_password}")
oidc_client_secret=$(read_secret "${AVIA_OIDC_CLIENT_SECRET_FILE:-/run/secrets/oidc_client_secret}")
session_encryption_key=$(read_secret "${AVIA_SESSION_ENCRYPTION_KEY_FILE:-/run/secrets/session_encryption_key}")
object_store_access_key=$(read_secret "${AVIA_OBJECT_STORE_ACCESS_KEY_FILE:-/run/secrets/minio_api_access_key}")
object_store_secret_key=$(read_secret "${AVIA_OBJECT_STORE_SECRET_KEY_FILE:-/run/secrets/minio_api_secret_key}")

export AVIA_DATABASE_URL="postgres://${AVIA_DATABASE_USER:-aviasurveil360}:${database_password}@${AVIA_DATABASE_HOST:-postgres}:${AVIA_DATABASE_PORT:-5432}/${AVIA_DATABASE_NAME:-aviasurveil360}?sslmode=disable"
export AVIA_OIDC_CLIENT_SECRET="$oidc_client_secret"
export AVIA_SESSION_ENCRYPTION_KEY="$session_encryption_key"
export AVIA_OBJECT_STORE_ACCESS_KEY="$object_store_access_key"
export AVIA_OBJECT_STORE_SECRET_KEY="$object_store_secret_key"
unset database_password oidc_client_secret session_encryption_key
unset object_store_access_key object_store_secret_key

exec /app/api
