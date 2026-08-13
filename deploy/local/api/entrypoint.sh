#!/bin/sh
set -eu

if [ -z "${AVIA_DATABASE_URL:-}" ] && [ -z "${AVIA_DATABASE_URL_FILE:-}" ]; then
  secret_path=${AVIA_DATABASE_PASSWORD_FILE:-/run/secrets/app_database_password}
  if [ ! -f "$secret_path" ]; then
    echo "required database input is unavailable" >&2
    exit 1
  fi
  database_password=$(tr -d '\r\n' <"$secret_path")
  if [ -z "$database_password" ]; then
    echo "required database input is empty" >&2
    exit 1
  fi
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

exec /app/api
