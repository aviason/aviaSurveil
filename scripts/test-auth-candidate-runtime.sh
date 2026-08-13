#!/bin/sh
set -eu

# This runner creates every credential and certificate in a task-owned,
# disposable directory. It intentionally starts only the isolated auth
# candidate profile; it neither reads nor alters any other serving topology.
repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
shared_auth_root=$(CDPATH= cd -- "$repository_root/../../shared/auth" && pwd)
compose_file="$repository_root/deploy/local/auth/compose.auth-candidate.yaml"
runtime_directory=$(mktemp -d /private/tmp/avia-auth-runtime.XXXXXX)
secret_directory="$runtime_directory/secrets"
project_name="avia-auth-runtime-$$"
auth_port=${AVIA_AUTH_CANDIDATE_PORT:-18081}
callback_port=${AVIA_AUTH_CANDIDATE_CALLBACK_PORT:-18082}
browser_seed_image="avia-auth-candidate-browser-seed:local"

cleanup() {
	status=$?
	trap - EXIT INT TERM
	if [ "$status" -ne 0 ]; then
		docker compose --project-name "$project_name" --file "$compose_file" --profile auth-candidate logs --no-color auth-candidate auth-postgres auth-mailpit >&2 || true
	fi
  docker compose --project-name "$project_name" --file "$compose_file" --profile auth-candidate down --volumes --remove-orphans >/dev/null 2>&1 || true
  docker image rm --force "$browser_seed_image" >/dev/null 2>&1 || true
  rm -rf "$runtime_directory"
	exit "$status"
}
trap cleanup EXIT INT TERM

mkdir -p "$secret_directory"
chmod 0700 "$runtime_directory" "$secret_directory"

database_password=$(openssl rand -hex 24)
smtp_password=$(openssl rand -hex 24)
printf '%s\n' "$database_password" >"$secret_directory/auth_database_password"
printf 'postgresql://auth_owner:%s@auth-postgres:5432/auth_candidate?sslmode=disable\n' "$database_password" >"$secret_directory/auth_database_url"
printf '%s\n' "$smtp_password" >"$secret_directory/auth_smtp_password"
printf 'aviasurveil360:%s\n' "$smtp_password" >"$secret_directory/auth_smtp_auth"
openssl rand 32 >"$secret_directory/auth_data_encryption_key"
openssl rand 32 >"$secret_directory/auth_mfa_key"
openssl rand -hex 24 >"$secret_directory/auth_oidc_client_secret"
openssl rand -hex 24 >"$secret_directory/auth_browser_password"
openssl rand -hex 24 >"$secret_directory/auth_browser_reset_password"
openssl rand -hex 24 >"$secret_directory/auth_load_password"
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out "$secret_directory/auth_signing_key" >/dev/null 2>&1
openssl req -x509 -newkey rsa:2048 -nodes -days 1 \
  -keyout "$secret_directory/auth_mailpit_key" \
  -out "$secret_directory/auth_mailpit_cert" \
  -subj '/CN=auth-mailpit' \
  -addext 'subjectAltName=DNS:auth-mailpit' >/dev/null 2>&1
cp "$secret_directory/auth_mailpit_cert" "$secret_directory/auth_mailpit_ca"
chmod 0400 "$secret_directory"/*
unset database_password smtp_password

export AVIA_AUTH_CANDIDATE_SECRETS_DIR="$secret_directory"
export AVIA_AUTH_CANDIDATE_PORT="$auth_port"
export AVIA_AUTH_CANDIDATE_CALLBACK_PORT="$callback_port"

docker compose --project-name "$project_name" --file "$compose_file" --profile auth-candidate up --build --detach --wait

network_name="${project_name}_auth-internal"
docker run --rm --network "$network_name" alpine:3.23 wget -qO- http://auth-candidate:8080/health/live | grep -F '"status":"alive"' >/dev/null
docker run --rm --network "$network_name" alpine:3.23 wget -qO- http://auth-candidate:8080/health/ready | grep -F '"status":"ready"' >/dev/null
docker run --rm --network "$network_name" alpine:3.23 wget -qO- http://auth-candidate:8080/.well-known/openid-configuration | grep -F '"issuer":"http://127.0.0.1:' >/dev/null

# Readiness must represent live recovery dependencies, not merely a successful
# startup. Stop only the task-owned Mailpit instance, verify fail-closed
# readiness, restore it, and verify readiness recovers before exercising the
# durable auth restart below.
docker compose --project-name "$project_name" --file "$compose_file" stop auth-mailpit >/dev/null
printf '%s\n' 'auth-candidate dependency-loss: probing Mailpit outage'
for attempt in 1 2 3 4 5 6 7 8 9 10 11 12; do
  if docker run --rm --network "$network_name" alpine:3.23 sh -ec 'wget -S -O /dev/null http://auth-candidate:8080/health/ready 2>&1 | grep -F "HTTP/1.1 503" >/dev/null'; then
    break
  fi
  sleep 1
done
if ! docker run --rm --network "$network_name" alpine:3.23 sh -ec 'wget -S -O /dev/null http://auth-candidate:8080/health/ready 2>&1 | grep -F "HTTP/1.1 503" >/dev/null'; then
  printf '%s\n' 'auth-candidate dependency-loss readiness did not return HTTP 503' >&2
  exit 1
fi
docker compose --project-name "$project_name" --file "$compose_file" start auth-mailpit >/dev/null
printf '%s\n' 'auth-candidate dependency-loss: awaiting Mailpit recovery'
for attempt in 1 2 3 4 5 6 7 8 9 10 11 12; do
  if docker run --rm --network "$network_name" alpine:3.23 wget -qO- http://auth-candidate:8080/health/ready | grep -F '"status":"ready"' >/dev/null; then
    break
  fi
  sleep 1
done
docker run --rm --network "$network_name" alpine:3.23 wget -qO- http://auth-candidate:8080/health/ready | grep -F '"status":"ready"' >/dev/null
docker compose --project-name "$project_name" --file "$compose_file" restart auth-candidate >/dev/null
printf '%s\n' 'auth-candidate restart: awaiting durable runtime readiness'
for attempt in 1 2 3 4 5 6 7 8 9 10 11 12; do
  if docker run --rm --network "$network_name" alpine:3.23 wget -qO- http://auth-candidate:8080/health/ready | grep -F '"status":"ready"' >/dev/null; then
    break
  fi
  sleep 1
done
docker run --rm --network "$network_name" alpine:3.23 wget -qO- http://auth-candidate:8080/health/ready | grep -F '"status":"ready"' >/dev/null

# Seed one synthetic, active local account only to exercise the mounted
# enumeration-safe recovery → encrypted outbox → authenticated STARTTLS
# Mailpit path. The recovery token is never printed or written to the repo.
docker compose --project-name "$project_name" --file "$compose_file" exec -T auth-postgres \
  psql -v ON_ERROR_STOP=1 -U auth_owner -d auth_candidate -c "INSERT INTO auth_identity.accounts(subject_id, state, password_hash, email_verified, auth_revision, created_at, updated_at) VALUES ('usr_AAAAAAAAAAAAAAAAAAAAAA', 'active', 'synthetic-disposable-password-hash', true, 1, now(), now()); INSERT INTO auth_identity.identifiers(subject_id, identifier_type, normalized_value, verified_at, created_at) VALUES ('usr_AAAAAAAAAAAAAAAAAAAAAA', 'email', 'recovery@example.invalid', now(), now());" >/dev/null
docker run --rm --network "$network_name" alpine:3.23 wget -qO /dev/null --post-data 'email=recovery%40example.invalid' http://auth-candidate:8080/recover/password
for attempt in 1 2 3 4 5 6 7 8 9 10; do
  if docker run --rm --network "$network_name" alpine:3.23 sh -ec 'wget -qO- http://auth-mailpit:8025/api/v1/messages | grep -q recovery@example.invalid'; then
    break
  fi
  sleep 1
done
docker run --rm --network "$network_name" alpine:3.23 sh -ec 'wget -qO- http://auth-mailpit:8025/api/v1/messages | grep -q recovery@example.invalid'

if [ "${AVIA_AUTH_CANDIDATE_BACKUP_RESTORE:-0}" = "1" ]; then
  dump_path="$runtime_directory/auth-candidate.dump"
  printf '%s\n' 'auth-candidate backup/restore: capturing task-owned PostgreSQL dump'
  docker compose --project-name "$project_name" --file "$compose_file" exec -T auth-postgres \
    pg_dump -U auth_owner -d auth_candidate --format=custom --no-owner --no-privileges >"$dump_path"
  test -s "$dump_path"
  printf '%s\n' 'auth-candidate backup/restore: removing and restoring auth schema'
  docker compose --project-name "$project_name" --file "$compose_file" stop auth-candidate >/dev/null
  docker compose --project-name "$project_name" --file "$compose_file" exec -T auth-postgres \
    psql -v ON_ERROR_STOP=1 -U auth_owner -d auth_candidate -c 'DROP SCHEMA auth_identity CASCADE;' >/dev/null
  docker compose --project-name "$project_name" --file "$compose_file" exec -T auth-postgres \
    pg_restore -U auth_owner -d auth_candidate --no-owner --no-privileges <"$dump_path"
  docker compose --project-name "$project_name" --file "$compose_file" start auth-candidate >/dev/null
  for attempt in 1 2 3 4 5 6 7 8 9 10 11 12; do
    if docker run --rm --network "$network_name" alpine:3.23 wget -qO- http://auth-candidate:8080/health/ready | grep -F '"status":"ready"' >/dev/null; then
      break
    fi
    sleep 1
  done
  docker run --rm --network "$network_name" alpine:3.23 wget -qO- http://auth-candidate:8080/health/ready | grep -F '"status":"ready"' >/dev/null
  docker compose --project-name "$project_name" --file "$compose_file" exec -T auth-postgres \
    psql -v ON_ERROR_STOP=1 -U auth_owner -d auth_candidate -tAc "SELECT 1 FROM auth_identity.accounts WHERE subject_id = 'usr_AAAAAAAAAAAAAAAAAAAAAA' AND state = 'active'" | grep -Fx '1' >/dev/null
  printf '%s\n' 'auth-candidate backup/restore: verified locally'
fi

if [ "${AVIA_AUTH_CANDIDATE_BROWSER_QUALIFICATION:-0}" = "1" ]; then
  browser_code_file="$runtime_directory/browser-mfa-code"

  password_reset_url_file="$runtime_directory/browser-password-reset-url"
  mfa_reset_url_file="$runtime_directory/browser-mfa-reset-url"
  : >"$browser_code_file"
  : >"$password_reset_url_file"
  : >"$mfa_reset_url_file"
  chmod 0600 "$browser_code_file" "$password_reset_url_file" "$mfa_reset_url_file"
  docker build --file "$shared_auth_root/Dockerfile" --target go-build --tag "$browser_seed_image" "$shared_auth_root" >/dev/null
  docker run --rm --user "$(id -u):$(id -g)" --network "$network_name" \
    --mount "type=bind,src=$secret_directory,dst=/run/auth,readonly" \
    --mount "type=bind,src=$runtime_directory,dst=/run/runtime" \
    -e AVIA_AUTH_RUNTIME_BROWSER_SEED=1 \
    -e AVIA_AUTH_TEST_DATABASE_URL_FILE=/run/auth/auth_database_url \
    -e AVIA_AUTH_RUNTIME_BROWSER_PASSWORD_FILE=/run/auth/auth_browser_password \
    -e AVIA_AUTH_RUNTIME_MFA_KEY_FILE=/run/auth/auth_mfa_key \
    -e AVIA_AUTH_RUNTIME_MFA_CODE_FILE=/run/runtime/browser-mfa-code \
    -e AVIA_AUTH_RUNTIME_PASSWORD_RESET_URL_FILE=/run/runtime/browser-password-reset-url \
    -e AVIA_AUTH_RUNTIME_MFA_RESET_URL_FILE=/run/runtime/browser-mfa-reset-url \
    -e AVIA_AUTH_RUNTIME_BROWSER_ISSUER="http://127.0.0.1:${auth_port}" \
    -e GOCACHE=/tmp/auth-browser-go-cache \
    "$browser_seed_image" go test ./internal/qualification -run '^TestRuntimeBrowserSeed$' -count=1
  AVIA_AUTH_CANDIDATE_BASE_URL="http://127.0.0.1:${auth_port}" \
  AVIA_AUTH_CANDIDATE_CALLBACK_PORT="$callback_port" \
  AVIA_AUTH_CANDIDATE_CLIENT_SECRET_FILE="$secret_directory/auth_oidc_client_secret" \
  AVIA_AUTH_CANDIDATE_BROWSER_PASSWORD_FILE="$secret_directory/auth_browser_password" \
  AVIA_AUTH_CANDIDATE_BROWSER_RESET_PASSWORD_FILE="$secret_directory/auth_browser_reset_password" \
  AVIA_AUTH_CANDIDATE_MFA_CODE_FILE="$browser_code_file" \
  AVIA_AUTH_CANDIDATE_PASSWORD_RESET_URL_FILE="$password_reset_url_file" \
  AVIA_AUTH_CANDIDATE_MFA_RESET_URL_FILE="$mfa_reset_url_file" \
  AVIA_AUTH_CANDIDATE_BROWSER_HEADED="${AVIA_AUTH_CANDIDATE_BROWSER_HEADED:-0}" \
  node "$repository_root/apps/web/scripts/test-auth-candidate-browser.mjs" >"$runtime_directory/browser-qualification.log" 2>&1 || {
    cat "$runtime_directory/browser-qualification.log" >&2
    exit 1
  }
  cat "$runtime_directory/browser-qualification.log"
fi

if [ "${AVIA_AUTH_CANDIDATE_LOAD_QUALIFICATION:-0}" = "1" ]; then
  load_log="$runtime_directory/load-qualification.log"
  case "$(uname -m)" in
    arm64|aarch64) ;;
    *) printf '%s\n' 'auth-candidate load qualification requires a native ARM64 host' >&2; exit 1 ;;
  esac
  if [ "$(docker image inspect --format '{{.Architecture}}' aviasurveil360/auth-candidate:local)" != "arm64" ]; then
    printf '%s\n' 'auth-candidate load qualification requires an ARM64 candidate image' >&2
    exit 1
  fi
  load_login_count=${AVIA_AUTH_CANDIDATE_LOAD_LOGIN_COUNT:-2}
  load_rejected_count=${AVIA_AUTH_CANDIDATE_LOAD_REJECTED_COUNT:-2}
  load_recovery_count=${AVIA_AUTH_CANDIDATE_LOAD_RECOVERY_COUNT:-4}
  case "$load_login_count" in
    1|2) ;;
    *) printf '%s\n' 'auth-candidate load login count must be 1 or 2 to preserve the Argon2id capacity bound' >&2; exit 1 ;;
  esac
  case "$load_rejected_count" in
    1|2) ;;
    *) printf '%s\n' 'auth-candidate load rejected count must be 1 or 2 to preserve the Argon2id capacity bound' >&2; exit 1 ;;
  esac
  case "$load_recovery_count" in
    1|2|3|4|5|6|7|8) ;;
    *) printf '%s\n' 'auth-candidate load recovery count must be from 1 to 8' >&2; exit 1 ;;
  esac
  docker build --file "$shared_auth_root/Dockerfile" --target go-build --tag "$browser_seed_image" "$shared_auth_root" >/dev/null
  docker run --rm --user "$(id -u):$(id -g)" --network "$network_name" \
    --mount "type=bind,src=$secret_directory,dst=/run/auth,readonly" \
    -e AVIA_AUTH_RUNTIME_LOAD_SEED=1 \
    -e AVIA_AUTH_TEST_DATABASE_URL_FILE=/run/auth/auth_database_url \
    -e AVIA_AUTH_RUNTIME_LOAD_PASSWORD_FILE=/run/auth/auth_load_password \
    -e GOCACHE=/tmp/auth-load-go-cache \
    "$browser_seed_image" go test ./internal/qualification -run '^TestRuntimeLoadSeed$' -count=1
  AVIA_AUTH_CANDIDATE_BASE_URL="http://127.0.0.1:${auth_port}" \
  AVIA_AUTH_CANDIDATE_CLIENT_SECRET_FILE="$secret_directory/auth_oidc_client_secret" \
  AVIA_AUTH_CANDIDATE_LOAD_PASSWORD_FILE="$secret_directory/auth_load_password" \
  AVIA_AUTH_CANDIDATE_LOAD_LOGIN_COUNT="$load_login_count" \
  AVIA_AUTH_CANDIDATE_LOAD_REJECTED_COUNT="$load_rejected_count" \
  AVIA_AUTH_CANDIDATE_LOAD_RECOVERY_COUNT="$load_recovery_count" \
  node "$repository_root/apps/web/scripts/test-auth-candidate-load.mjs" >"$load_log" 2>&1 || {
    cat "$load_log" >&2
    exit 1
  }
  for attempt in 1 2 3 4 5 6 7 8 9 10; do
    load_delivery_count=$(docker run --rm --network "$network_name" alpine:3.23 sh -ec "wget -qO- http://auth-mailpit:8025/api/v1/messages | grep -o 'load-candidate@example.invalid' | wc -l" || true)
    if [ "${load_delivery_count:-0}" -ge "$load_recovery_count" ]; then
      break
    fi
    sleep 1
  done
  if [ "${load_delivery_count:-0}" -lt "$load_recovery_count" ]; then
    printf '%s\n' 'auth-candidate mixed-load recovery delivery count was incomplete' >&2
    exit 1
  fi
  cat "$load_log"
fi

printf '%s\n' 'auth-candidate isolated runtime: verified locally'
