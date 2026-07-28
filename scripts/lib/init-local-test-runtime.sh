#!/bin/sh
set -eu

initialize_local_test_runtime() {
  runtime_directory=$1
  public_origin=$2
  repository_root=$3
  secret_directory="$runtime_directory/secrets"
  keycloak_directory="$runtime_directory/keycloak"

  umask 077
  mkdir -p "$secret_directory" "$keycloak_directory"
  chmod 0700 "$runtime_directory" "$secret_directory" "$keycloak_directory"

  for filename in \
    app_database_password \
    keycloak_bootstrap_admin_password \
    keycloak_database_password \
    keycloak_service_client_secret \
    minio_root_password \
    oidc_client_secret \
    smtp_password
  do
    openssl rand -hex 32 >"$secret_directory/$filename"
    chmod 0600 "$secret_directory/$filename"
  done
  openssl rand -hex 10 >"$secret_directory/minio_root_user"
  openssl rand -base64 32 >"$secret_directory/session_encryption_key"
  chmod 0600 \
    "$secret_directory/minio_root_user" \
    "$secret_directory/session_encryption_key"
  smtp_password_value=$(tr -d '\r\n' <"$secret_directory/smtp_password")
  printf 'aviasurveil360:%s\n' "$smtp_password_value" \
    >"$secret_directory/smtp_auth_file"
  chmod 0600 "$secret_directory/smtp_auth_file"
  unset smtp_password_value

  node "$repository_root/deploy/local/keycloak/build-realm.mjs" \
    --source "$repository_root/deploy/local/keycloak/realm-source.json" \
    --output "$keycloak_directory/realm.json" \
    --client-secret-file "$secret_directory/oidc_client_secret" \
    --service-client-secret-file "$secret_directory/keycloak_service_client_secret" \
    --smtp-password-file "$secret_directory/smtp_password" \
    --public-origin "$public_origin"
  chmod 0600 "$keycloak_directory/realm.json"
}
