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

if [ -z "${AVIA_DATABASE_URL:-}" ] && [ -z "${AVIA_DATABASE_URL_FILE:-}" ]; then
  database_password=$(read_secret "${AVIA_DATABASE_PASSWORD_FILE:-/run/secrets/app_database_password}")
  database_url_file=$(mktemp /tmp/avia-database-url.XXXXXX)
  chmod 600 "$database_url_file"
  printf 'postgres://%s:%s@%s:%s/%s?sslmode=disable' \
    "${AVIA_DATABASE_USER:-aviasurveil360}" \
    "$database_password" \
    "${AVIA_DATABASE_HOST:-postgres}" \
    "${AVIA_DATABASE_PORT:-5432}" \
    "${AVIA_DATABASE_NAME:-aviasurveil360}" >"$database_url_file"
  export AVIA_DATABASE_URL_FILE="$database_url_file"
  trap 'rm -f "$database_url_file"' EXIT
  unset database_password
fi

exec /app/worker
