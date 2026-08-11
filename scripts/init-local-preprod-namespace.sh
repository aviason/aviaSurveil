#!/bin/sh
set -eu

umask 077

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
state_directory=${AVIA_PREPROD_STATE_DIR:-"$repository_root/.local/aviasurveil360-preprod"}
secret_directory="$state_directory/secrets"
keycloak_directory="$state_directory/keycloak"
control_store_directory="$state_directory/control-store"
realm_builder="$repository_root/deploy/local/keycloak/build-realm.mjs"
realm_source="$repository_root/deploy/local/keycloak/realm-source.json"
runtime_realm="$keycloak_directory/realm.json"
namespace_record="$state_directory/namespace.json"

if [ "$#" -ne 0 ]; then
  echo "usage: $0" >&2
  exit 64
fi

secret_files="
preprod_app_database_password
preprod_normal_api_database_password
preprod_canonical_demo_oidc_qualification_password
preprod_keycloak_bootstrap_admin_password
preprod_keycloak_database_password
preprod_keycloak_service_client_secret
preprod_session_encryption_key
preprod_data_feed_payload_key
preprod_minio_api_access_key
preprod_minio_api_secret_key
preprod_minio_loader_access_key
preprod_minio_loader_secret_key
preprod_minio_root_password
preprod_minio_root_user
preprod_oidc_client_secret
preprod_smtp_password
preprod_smtp_auth_file
preprod_loader_seed
"

for filename in $secret_files; do
  if [ -e "$secret_directory/$filename" ]; then
    echo "local-preprod material exists; initializer is create-only" >&2
    exit 1
  fi
done
if [ -e "$runtime_realm" ] || [ -e "$namespace_record" ]; then
  echo "local-preprod namespace exists; initializer is create-only" >&2
  exit 1
fi

mkdir -p \
  "$secret_directory" \
  "$keycloak_directory" \
  "$control_store_directory/intents" \
  "$control_store_directory/results" \
  "$control_store_directory/checkpoints" \
  "$control_store_directory/authorizations" \
  "$control_store_directory/cleanup"
chmod 0700 \
  "$state_directory" \
  "$secret_directory" \
  "$keycloak_directory" \
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
    preprod_smtp_auth_file)
      smtp_password_value=$(tr -d '\r\n' <"$temporary_directory/preprod_smtp_password")
      printf 'aviasurveil360-preprod:%s\n' "$smtp_password_value" >"$temporary_directory/$filename"
      unset smtp_password_value
      ;;
    preprod_minio_root_user | preprod_minio_api_access_key | preprod_minio_loader_access_key)
      openssl rand -hex 10 >"$temporary_directory/$filename"
      ;;
    preprod_loader_seed)
      openssl rand -hex 32 >"$temporary_directory/$filename"
      ;;
    preprod_data_feed_payload_key)
      openssl rand -hex 16 | tr -d '\r\n' >"$temporary_directory/$filename"
      ;;
    preprod_session_encryption_key)
      openssl rand -base64 32 >"$temporary_directory/$filename"
      ;;
    *)
      openssl rand -hex 32 >"$temporary_directory/$filename"
      ;;
  esac
  chmod 0600 "$temporary_directory/$filename"
done

node "$realm_builder" \
  --source "$realm_source" \
  --output "$temporary_directory/realm.json" \
  --client-secret-file "$temporary_directory/preprod_oidc_client_secret" \
  --service-client-secret-file "$temporary_directory/preprod_keycloak_service_client_secret" \
  --smtp-password-file "$temporary_directory/preprod_smtp_password" \
  --realm-name aviasurveil360-local-preprod \
  --web-client-id aviasurveil360-local-preprod-web \
  --service-client-id aviasurveil360-local-preprod-lifecycle \
  --public-origin "${AVIA_PREPROD_WEB_ORIGIN:-https://localhost:8445}" \
  --smtp-host preprod-mailpit \
  --smtp-user aviasurveil360-preprod

printf '%s\n' \
  '{' \
  '  "schemaVersion": "preprod-namespace/v1",' \
  '  "environment": "local-preprod",' \
  '  "databaseName": "aviasurveil360_local_preprod",' \
  '  "databaseOwner": "aviasurveil360_preprod_loader",' \
  '  "composeProject": "aviasurveil360-local-preprod",' \
  '  "keycloakRealm": "aviasurveil360-local-preprod",' \
  '  "keycloakDatabase": "keycloak_local_preprod",' \
  '  "keycloakServiceClientId": "aviasurveil360-local-preprod-lifecycle",' \
  '  "mailpitNamespace": "aviasurveil360-local-preprod",' \
  '  "objectBucket": "aviasurveil360-local-preprod",' \
  '  "objectPrefixPolicy": "runs/{runId}/",' \
  '  "loaderQueueNamespace": "aviasurveil360-local-preprod",' \
  '  "controlStore": "outside-disposable-target",' \
  '  "operationAuthorization": "separate-ephemeral-0600-file-required"' \
  '}' >"$temporary_directory/namespace.json"
chmod 0600 "$temporary_directory/realm.json" "$temporary_directory/namespace.json"

for filename in $secret_files; do
  mv -- "$temporary_directory/$filename" "$secret_directory/$filename"
done
mv -- "$temporary_directory/realm.json" "$runtime_realm"
mv -- "$temporary_directory/namespace.json" "$namespace_record"
chmod 0600 "$runtime_realm" "$namespace_record"

echo "Local-preprod namespace initialized under $state_directory"
