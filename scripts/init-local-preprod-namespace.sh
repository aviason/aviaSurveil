#!/bin/sh
set -eu

umask 077

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
state_directory=${AVIA_PREPROD_STATE_DIR:-"$repository_root/.local/aviasurveil360-preprod"}
secret_directory="$state_directory/secrets"
control_store_directory="$state_directory/control-store"
namespace_record="$state_directory/namespace.json"

if [ "$#" -ne 0 ]; then
  echo "usage: $0" >&2
  exit 64
fi

secret_files="
preprod_app_database_password
preprod_normal_api_database_password
preprod_canonical_demo_oidc_qualification_password
preprod_auth_database_password
preprod_auth_database_url
preprod_auth_signing_key
preprod_auth_data_encryption_key
preprod_auth_mfa_key
preprod_auth_admin_secret
preprod_session_encryption_key
preprod_minio_api_access_key
preprod_minio_api_secret_key
preprod_minio_loader_access_key
preprod_minio_loader_secret_key
preprod_minio_root_password
preprod_minio_root_user
preprod_oidc_client_secret
preprod_loader_seed
"

for filename in $secret_files; do
  if [ -e "$secret_directory/$filename" ]; then
    echo "local-preprod material exists; initializer is create-only" >&2
    exit 1
  fi
done
if [ -e "$namespace_record" ]; then
  echo "local-preprod namespace exists; initializer is create-only" >&2
  exit 1
fi

mkdir -p \
  "$secret_directory" \
  "$control_store_directory/intents" \
  "$control_store_directory/results" \
  "$control_store_directory/checkpoints" \
  "$control_store_directory/authorizations" \
  "$control_store_directory/cleanup"
chmod 0700 \
  "$state_directory" \
  "$secret_directory" \
  "$control_store_directory" \
  "$control_store_directory/intents" \
  "$control_store_directory/results" \
  "$control_store_directory/checkpoints" \
  "$control_store_directory/authorizations" \
  "$control_store_directory/cleanup"

temporary_directory="$state_directory/.namespace-next-$$"
cleanup() {
  rm -rf -- "$temporary_directory"
}
trap cleanup EXIT HUP INT TERM
mkdir "$temporary_directory"
chmod 0700 "$temporary_directory"

for filename in $secret_files; do
  case "$filename" in
    preprod_auth_database_url)
      auth_database_password=$(tr -d '\r\n' <"$temporary_directory/preprod_auth_database_password")
      printf 'postgres://auth_preprod:%s@preprod-auth-postgres:5432/auth_local_preprod?sslmode=disable\n' "$auth_database_password" >"$temporary_directory/$filename"
      unset auth_database_password
      ;;
    preprod_auth_database_password | preprod_auth_admin_secret | preprod_oidc_client_secret)
      openssl rand -hex 32 >"$temporary_directory/$filename"
      ;;
    preprod_auth_data_encryption_key | preprod_auth_mfa_key)
      openssl rand 32 >"$temporary_directory/$filename"
      ;;
    preprod_auth_signing_key)
      openssl genrsa 2048 >"$temporary_directory/$filename" 2>/dev/null
      ;;
    preprod_minio_root_user | preprod_minio_api_access_key | preprod_minio_loader_access_key)
      openssl rand -hex 10 >"$temporary_directory/$filename"
      ;;
    preprod_loader_seed)
      openssl rand -hex 32 >"$temporary_directory/$filename"
      ;;
    preprod_session_encryption_key)
      openssl rand -base64 32 >"$temporary_directory/$filename"
      ;;
    *)
      openssl rand -hex 32 >"$temporary_directory/$filename"
      ;;
  esac
  chmod 0400 "$temporary_directory/$filename"
done

printf '%s\n' \
  '{' \
  '  "schemaVersion": "preprod-namespace/v2",' \
  '  "environment": "local-preprod",' \
  '  "identityProvider": "first-party",' \
  '  "databaseName": "aviasurveil360_local_preprod",' \
  '  "databaseOwner": "aviasurveil360_preprod_loader",' \
  '  "authDatabaseName": "auth_local_preprod",' \
  '  "authDatabaseRole": "auth_preprod",' \
  '  "authAdminAddress": "http://preprod-auth:8081",' \
  '  "publicIssuerPath": "/identity",' \
  '  "composeProject": "aviasurveil360-local-preprod",' \
  '  "objectBucket": "aviasurveil360-local-preprod",' \
  '  "objectPrefixPolicy": "runs/{runId}/",' \
  '  "loaderQueueNamespace": "aviasurveil360-local-preprod",' \
  '  "controlStore": "outside-disposable-target",' \
  '  "operationAuthorization": "separate-ephemeral-0400-file-required"' \
  '}' >"$temporary_directory/namespace.json"
chmod 0400 "$temporary_directory/namespace.json"

for filename in $secret_files; do
  mv -- "$temporary_directory/$filename" "$secret_directory/$filename"
done
mv -- "$temporary_directory/namespace.json" "$namespace_record"
chmod 0400 "$namespace_record"

echo "Local-preprod first-party identity namespace initialized under $state_directory"
