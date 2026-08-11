#!/bin/sh
set -eu

read_secret() {
  secret_path=$1
  if [ ! -f "$secret_path" ]; then
    echo "required mounted Keycloak credential is unavailable" >&2
    exit 1
  fi
  secret_value=$(tr -d '\r\n' <"$secret_path")
  if [ -z "$secret_value" ]; then
    echo "required mounted Keycloak credential is empty" >&2
    exit 1
  fi
  printf '%s' "$secret_value"
}

KC_DB_PASSWORD=$(read_secret "${KC_DB_PASSWORD_FILE:-/run/secrets/keycloak_database_password}")
KC_SMTP_PASSWORD=$(read_secret "${KC_SMTP_PASSWORD_FILE:-/run/secrets/keycloak_smtp_password}")
if [ ! -r "${AVIA_RDS_CA_BUNDLE_FILE:-/etc/ssl/certs/aws-rds-global-bundle.pem}" ]; then
  echo "reviewed RDS CA bundle is unavailable" >&2
  exit 1
fi
export KC_DB_PASSWORD KC_SMTP_PASSWORD

unset secret_value secret_path
exec /opt/keycloak/bin/kc.sh "$@"
