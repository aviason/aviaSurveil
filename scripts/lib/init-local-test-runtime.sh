#!/bin/sh
set -eu

initialize_local_test_runtime() {
  runtime_directory=$1
  public_origin=$2
  repository_root=$3
  secret_directory="$runtime_directory/secrets"

  umask 077
  mkdir -p "$secret_directory"
  chmod 0700 "$runtime_directory" "$secret_directory"

  for filename in \
    app_database_password \
    minio_api_access_key \
    minio_api_secret_key \
    minio_root_password \
    minio_worker_access_key \
    minio_worker_secret_key \
    smtp_password
  do
    openssl rand -hex 32 >"$secret_directory/$filename"
    chmod 0600 "$secret_directory/$filename"
  done
  openssl rand -hex 10 >"$secret_directory/minio_root_user"
  openssl rand -hex 10 >"$secret_directory/minio_api_access_key"
  openssl rand -hex 10 >"$secret_directory/minio_worker_access_key"
  openssl rand -base64 32 >"$secret_directory/session_encryption_key"
  chmod 0600 \
    "$secret_directory/minio_root_user" \
    "$secret_directory/minio_api_access_key" \
    "$secret_directory/minio_worker_access_key" \
    "$secret_directory/session_encryption_key"
  smtp_password_value=$(tr -d '\r\n' <"$secret_directory/smtp_password")
  printf 'aviasurveil360:%s\n' "$smtp_password_value" \
    >"$secret_directory/smtp_auth_file"
  chmod 0600 "$secret_directory/smtp_auth_file"
  unset smtp_password_value

  : "$public_origin" "$repository_root"
}
