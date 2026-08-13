#!/bin/sh
set -eu

umask 077

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
local_state_dir=${AVIASURVEIL_LOCAL_STATE_DIR:-"$repository_root/.local/aviasurveil360"}
secret_directory="$local_state_dir/secrets"
recovery_directory="$local_state_dir/recovery"
recovery_tls_directory="$recovery_directory/tls"
recovery_public_cert="$recovery_tls_directory/public.crt"
recovery_private_key="$recovery_tls_directory/private.key"
rotate=false

if [ "${1:-}" = "--rotate" ]; then
  rotate=true
  shift
fi

if [ "$#" -ne 0 ]; then
  echo "usage: $0 [--rotate]" >&2
  exit 64
fi

secret_files="
app_database_password
backup_minio_root_password
backup_minio_root_user
backup_object_access_key
backup_object_secret_key
backup_pgbackrest_access_key
backup_pgbackrest_secret_key
backup_repository_cipher_passphrase
grafana_admin_password
minio_api_access_key
minio_api_secret_key
minio_root_password
minio_root_user
minio_worker_access_key
minio_worker_secret_key
session_encryption_key
smtp_password
smtp_auth_file
"

if [ "$rotate" = false ]; then
  for filename in $secret_files; do
    if [ -e "$secret_directory/$filename" ]; then
      echo "local credentials already exist; refusing to overwrite without --rotate" >&2
      exit 1
    fi
  done
  if [ -e "$recovery_public_cert" ] || [ -e "$recovery_private_key" ]; then
    echo "local recovery TLS material already exists; refusing to overwrite without --rotate" >&2
    exit 1
  fi
fi

mkdir -p \
  "$secret_directory" \
  "$recovery_tls_directory"
chmod 0700 \
  "$local_state_dir" \
  "$secret_directory" \
  "$recovery_directory" \
  "$recovery_tls_directory"

temporary_directory="$local_state_dir/.secrets-next-$$"
cleanup() {
  rm -rf -- "$temporary_directory"
}
trap cleanup EXIT HUP INT TERM
mkdir "$temporary_directory"
chmod 0700 "$temporary_directory"

for filename in $secret_files; do
  if [ "$filename" = "smtp_auth_file" ]; then
    smtp_password_value=$(tr -d '\r\n' <"$temporary_directory/smtp_password")
    printf 'aviasurveil360:%s\n' "$smtp_password_value" >"$temporary_directory/$filename"
    unset smtp_password_value
  elif [ "$filename" = "minio_root_user" ] ||
    [ "$filename" = "backup_minio_root_user" ] ||
    [ "$filename" = "backup_object_access_key" ] ||
    [ "$filename" = "backup_pgbackrest_access_key" ] ||
    [ "$filename" = "minio_api_access_key" ] ||
    [ "$filename" = "minio_worker_access_key" ]; then
    openssl rand -hex 10 >"$temporary_directory/$filename"
  elif [ "$filename" = "session_encryption_key" ]; then
    openssl rand -base64 32 >"$temporary_directory/$filename"
  else
    openssl rand -hex 32 >"$temporary_directory/$filename"
  fi
  chmod 0600 "$temporary_directory/$filename"
done

openssl req \
  -x509 \
  -newkey rsa:2048 \
  -nodes \
  -sha256 \
  -days 30 \
  -subj "/CN=backup-minio" \
  -addext "subjectAltName=DNS:backup-minio" \
  -keyout "$temporary_directory/private.key" \
  -out "$temporary_directory/public.crt" \
  >/dev/null 2>&1
chmod 0600 "$temporary_directory/private.key"
chmod 0644 "$temporary_directory/public.crt"

for filename in $secret_files; do
  mv -f -- "$temporary_directory/$filename" "$secret_directory/$filename"
done
mv -f -- "$temporary_directory/private.key" "$recovery_private_key"
mv -f -- "$temporary_directory/public.crt" "$recovery_public_cert"
chmod 0600 "$recovery_private_key"
chmod 0644 "$recovery_public_cert"

echo "Local credential files initialized under $secret_directory"
