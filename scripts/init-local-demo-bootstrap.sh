#!/bin/sh
set -eu

umask 077

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
local_target=${AVIA_LOCAL_TARGET:-namibia/demo}
case "$local_target" in
  namibia/dev | namibia/demo) ;;
  *) echo "AVIA_LOCAL_TARGET must be one exact local target: namibia/dev or namibia/demo" >&2; exit 64 ;;
esac
local_environment=${local_target##*/}
state_directory=${AVIASURVEIL_LOCAL_STATE_DIR:-"$repository_root/.local/aviasurveil360"}
secret_directory="$state_directory/secrets"
roster_directory="$state_directory/roster-credentials"
bootstrap_record="$state_directory/bootstrap.json"

if [ "$#" -ne 0 ]; then
  echo "usage: $0" >&2
  exit 64
fi

if [ -e "$bootstrap_record" ] || [ -e "$secret_directory" ] || [ -e "$roster_directory" ]; then
  echo "local bootstrap material already exists; refusing to overwrite" >&2
  exit 1
fi

mkdir -p "$secret_directory" "$roster_directory"
chmod 0700 "$state_directory" "$secret_directory" "$roster_directory"

temporary_directory="$state_directory/.bootstrap-next-$$"
cleanup() { rm -rf -- "$temporary_directory"; }
trap cleanup EXIT HUP INT TERM
mkdir -p "$temporary_directory/secrets" "$temporary_directory/roster-credentials"
chmod 0700 "$temporary_directory" "$temporary_directory/secrets" "$temporary_directory/roster-credentials"

write_hex() { openssl rand -hex 32 >"$1"; chmod 0400 "$1"; }
write_b64() { openssl rand -base64 32 >"$1"; chmod 0400 "$1"; }

write_hex "$temporary_directory/secrets/rds_master_database_password"
write_hex "$temporary_directory/secrets/surveil_runtime_database_password"
write_hex "$temporary_directory/secrets/surveil_migration_database_password"
write_hex "$temporary_directory/secrets/surveil_bootstrap_database_password"
write_hex "$temporary_directory/secrets/auth_runtime_database_password"
write_hex "$temporary_directory/secrets/oidc_client_secret"
write_hex "$temporary_directory/secrets/auth_admin_secret"
write_hex "$temporary_directory/secrets/auth_bootstrap_secret"
write_hex "$temporary_directory/secrets/app_bootstrap_password"
write_b64 "$temporary_directory/secrets/session_encryption_key"
write_hex "$temporary_directory/secrets/minio_root_password"
write_hex "$temporary_directory/secrets/minio_api_secret_key"
write_hex "$temporary_directory/secrets/minio_worker_secret_key"
openssl rand -hex 10 >"$temporary_directory/secrets/minio_root_user"
openssl rand -hex 10 >"$temporary_directory/secrets/minio_api_access_key"
openssl rand -hex 10 >"$temporary_directory/secrets/minio_worker_access_key"
chmod 0400 "$temporary_directory/secrets/minio_root_user" "$temporary_directory/secrets/minio_api_access_key" "$temporary_directory/secrets/minio_worker_access_key"
write_hex "$temporary_directory/secrets/smtp_password"

openssl genrsa 2048 >"$temporary_directory/secrets/auth_signing_key" 2>/dev/null
openssl rand 32 >"$temporary_directory/secrets/auth_data_encryption_key"
openssl rand 32 >"$temporary_directory/secrets/auth_mfa_key"
chmod 0400 "$temporary_directory/secrets/auth_signing_key" "$temporary_directory/secrets/auth_data_encryption_key" "$temporary_directory/secrets/auth_mfa_key"

master_password=$(tr -d '\r\n' <"$temporary_directory/secrets/rds_master_database_password")
surveil_runtime_password=$(tr -d '\r\n' <"$temporary_directory/secrets/surveil_runtime_database_password")
surveil_migration_password=$(tr -d '\r\n' <"$temporary_directory/secrets/surveil_migration_database_password")
surveil_bootstrap_password=$(tr -d '\r\n' <"$temporary_directory/secrets/surveil_bootstrap_database_password")
auth_runtime_password=$(tr -d '\r\n' <"$temporary_directory/secrets/auth_runtime_database_password")
printf 'postgres://avia_master:%s@postgres:5432/postgres?sslmode=disable\n' "$master_password" >"$temporary_directory/secrets/rds_master_database_url"
printf 'postgres://surveil_runtime:%s@postgres:5432/surveil?sslmode=disable\n' "$surveil_runtime_password" >"$temporary_directory/secrets/surveil_runtime_database_url"
printf 'postgres://surveil_migration:%s@postgres:5432/surveil?sslmode=disable\n' "$surveil_migration_password" >"$temporary_directory/secrets/surveil_migration_database_url"
printf 'postgres://surveil_bootstrap:%s@postgres:5432/surveil?sslmode=disable\n' "$surveil_bootstrap_password" >"$temporary_directory/secrets/surveil_bootstrap_database_url"
printf 'postgres://auth_runtime:%s@postgres:5432/auth?sslmode=disable\n' "$auth_runtime_password" >"$temporary_directory/secrets/auth_database_url"
chmod 0400 "$temporary_directory/secrets/rds_master_database_url" "$temporary_directory/secrets/surveil_runtime_database_url" "$temporary_directory/secrets/surveil_migration_database_url" "$temporary_directory/secrets/surveil_bootstrap_database_url" "$temporary_directory/secrets/auth_database_url"
unset master_password surveil_runtime_password surveil_migration_password surveil_bootstrap_password auth_runtime_password

printf 'aviasurveil360:%s\n' "$(tr -d '\r\n' <"$temporary_directory/secrets/smtp_password")" >"$temporary_directory/secrets/smtp_auth_file"
chmod 0400 "$temporary_directory/secrets/smtp_auth_file"

for purpose in PLATFORM-ADMIN AGA-MANAGER FINANCE-REVIEWER GENERAL-MANAGER EXECUTIVE-DIRECTOR LEAD-INSPECTOR INSPECTOR TARGET-AUDITEE CONTROL-AUDITEE; do
  write_hex "$temporary_directory/roster-credentials/$purpose"
done

printf '%s\n' \
  '{' \
  '  "schemaVersion": 1,' \
  "  \"target\": \"$local_target\"," \
  "  \"environment\": \"$local_environment\"," \
  '  "identityProvider": "first-party",' \
  '  "credentialMode": "target-secret-version",' \
  '  "controlStore": "task-owned-local-state"' \
  '}' >"$temporary_directory/bootstrap.json"
chmod 0400 "$temporary_directory/bootstrap.json"

for file in "$temporary_directory/secrets"/*; do mv -- "$file" "$secret_directory/"; done
for file in "$temporary_directory/roster-credentials"/*; do mv -- "$file" "$roster_directory/"; done
mv -- "$temporary_directory/bootstrap.json" "$bootstrap_record"
chmod 0400 "$bootstrap_record"

echo "Local $local_target bootstrap material initialized under $state_directory"
