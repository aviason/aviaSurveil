#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_FILE="${REPOSITORY_ROOT}/deploy/local/compose.test.yaml"
COMPOSE_PROJECT="aviasurveil360-task11-oidc"
TASK_POSTGRES_PORT="${AVIA_TASK11_OIDC_POSTGRES_PORT:-55443}"
TASK_KEYCLOAK_PORT="${AVIA_TASK11_OIDC_KEYCLOAK_PORT:-58092}"
TASK_KEYCLOAK_MANAGEMENT_PORT="${AVIA_TASK11_OIDC_KEYCLOAK_MANAGEMENT_PORT:-59013}"
TASK_OBJECT_STORE_PORT="${AVIA_TASK11_OIDC_OBJECT_STORE_PORT:-59014}"
TASK_OBJECT_STORE_CONSOLE_PORT="${AVIA_TASK11_OIDC_OBJECT_STORE_CONSOLE_PORT:-59015}"
TASK_API_PORT="${AVIA_TASK11_OIDC_API_PORT:-58093}"
RUNTIME_DIRECTORY="$(mktemp -d /private/tmp/aviasurveil360-task11-oidc.XXXXXX)"
TEST_RUNTIME_HELPER="${REPOSITORY_ROOT}/scripts/lib/init-local-test-runtime.sh"
SHARED_GO_CACHE="$(go env GOCACHE)"
TASK_GO_CACHE="${RUNTIME_DIRECTORY}/go-cache"
TASK_GO_TMP="${RUNTIME_DIRECTORY}/go-tmp"
API_PID=""
WORKER_PID=""
RUNTIME_SCAN_COMPLETE="false"
export COMPOSE_PROGRESS="plain"
export AVIA_TEST_POSTGRES_PORT="${TASK_POSTGRES_PORT}"
export AVIA_TEST_KEYCLOAK_PORT="${TASK_KEYCLOAK_PORT}"
export AVIA_TEST_KEYCLOAK_MANAGEMENT_PORT="${TASK_KEYCLOAK_MANAGEMENT_PORT}"
export AVIA_TEST_OBJECT_STORE_PORT="${TASK_OBJECT_STORE_PORT}"
export AVIA_TEST_OBJECT_STORE_CONSOLE_PORT="${TASK_OBJECT_STORE_CONSOLE_PORT}"
export AVIA_TEST_RUNTIME_DIR="${RUNTIME_DIRECTORY}"
mkdir -p "${TASK_GO_TMP}"
export GOTMPDIR="${TASK_GO_TMP}"

. "${TEST_RUNTIME_HELPER}"
initialize_local_test_runtime \
  "${RUNTIME_DIRECTORY}" \
  "http://127.0.0.1:4174" \
  "${REPOSITORY_ROOT}"

read_runtime_secret() {
  tr -d '\r\n' <"${RUNTIME_DIRECTORY}/secrets/$1"
}

APP_DATABASE_PASSWORD="$(read_runtime_secret app_database_password)"
KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD="$(read_runtime_secret keycloak_bootstrap_admin_password)"
KEYCLOAK_SERVICE_CLIENT_SECRET="$(read_runtime_secret keycloak_service_client_secret)"
MINIO_ROOT_PASSWORD="$(read_runtime_secret minio_root_password)"
MINIO_ROOT_USER="$(read_runtime_secret minio_root_user)"
OIDC_CLIENT_SECRET="$(read_runtime_secret oidc_client_secret)"
SESSION_ENCRYPTION_KEY="$(read_runtime_secret session_encryption_key)"
APPLICATION_ADMIN_USERNAME="local.admin.$(openssl rand -hex 6)@example.test"
APPLICATION_ADMIN_PASSWORD="$(openssl rand -hex 20)Aa1!"

seed_task_go_cache() {
  mkdir -p "${TASK_GO_CACHE}"
  if [[ -d "${SHARED_GO_CACHE}" && "${SHARED_GO_CACHE}" != "${TASK_GO_CACHE}" ]]; then
    cp -al "${SHARED_GO_CACHE}/." "${TASK_GO_CACHE}/"
  fi
}

scan_runtime_artifacts_for_secret_leaks() {
  local docker_log="${RUNTIME_DIRECTORY}/docker-runtime.log"
  local secret_path secret_name secret_value
  local -a artifact_targets=(
    "${RUNTIME_DIRECTORY}/api.log"
    "${RUNTIME_DIRECTORY}/worker.log"
    "${docker_log}"
  )

  docker compose --project-name "${COMPOSE_PROJECT}" --file "${COMPOSE_FILE}" \
    logs --no-color >"${docker_log}"
  if [[ -d "${RUNTIME_DIRECTORY}/playwright-results" ]]; then
    artifact_targets+=("${RUNTIME_DIRECTORY}/playwright-results")
  fi

  for secret_path in "${RUNTIME_DIRECTORY}/secrets/"*; do
    secret_name="${secret_path##*/}"
    secret_value="$(tr -d '\r\n' <"${secret_path}")"
    if [[ -n "${secret_value}" ]] &&
      rg --fixed-strings --quiet -- "${secret_value}" "${artifact_targets[@]}"; then
      echo "generated secret found in OIDC runtime artifacts: ${secret_name}" >&2
      return 1
    fi
  done
  if rg --fixed-strings --quiet -- "${APPLICATION_ADMIN_PASSWORD}" \
    "${artifact_targets[@]}"; then
    echo "generated application administrator password found in OIDC runtime artifacts" >&2
    return 1
  fi
  RUNTIME_SCAN_COMPLETE="true"
  echo "OIDC runtime secret/log scan: zero generated-secret matches"
}

cleanup() {
  local status=$?
  local secret_scan_status=0
  trap - EXIT
  set +e
  if [[ -n "${WORKER_PID}" ]]; then
    kill "${WORKER_PID}" 2>/dev/null
    wait "${WORKER_PID}" 2>/dev/null
  fi
  if [[ -n "${API_PID}" ]]; then
    kill "${API_PID}" 2>/dev/null
    wait "${API_PID}" 2>/dev/null
  fi
  if [[ "${RUNTIME_SCAN_COMPLETE}" != "true" ]]; then
    scan_runtime_artifacts_for_secret_leaks
    secret_scan_status=$?
    if [[ ${secret_scan_status} -ne 0 ]]; then
      status=1
    fi
  fi
  if [[ ${status} -ne 0 ]]; then
    if [[ ${secret_scan_status} -eq 0 ]]; then
      for log_file in "${RUNTIME_DIRECTORY}"/*.log; do
        if [[ -f "${log_file}" ]]; then
          echo "--- ${log_file} ---" >&2
          tail -n 200 "${log_file}" >&2
        fi
      done
    else
      echo "runtime logs suppressed because the generated-secret scan failed" >&2
    fi
  fi
  env GOCACHE="${TASK_GO_CACHE}" go clean -cache
  docker compose --project-name "${COMPOSE_PROJECT}" --file "${COMPOSE_FILE}" down --volumes --remove-orphans
  rm -rf "${RUNTIME_DIRECTORY}"
  exit "${status}"
}
trap cleanup EXIT

docker compose --project-name "${COMPOSE_PROJECT}" --file "${COMPOSE_FILE}" down --volumes --remove-orphans
docker compose --project-name "${COMPOSE_PROJECT}" --file "${COMPOSE_FILE}" up --detach --wait postgres keycloak-postgres keycloak object-store

KEYCLOAK_PUBLIC_URL="http://127.0.0.1:${TASK_KEYCLOAK_PORT}/identity"
KEYCLOAK_ADMIN_TOKEN="$(
  curl --fail --silent --show-error \
    --request POST \
    "${KEYCLOAK_PUBLIC_URL}/realms/master/protocol/openid-connect/token" \
    --header "Content-Type: application/x-www-form-urlencoded" \
    --data-urlencode "client_id=admin-cli" \
    --data-urlencode "grant_type=password" \
    --data-urlencode "username=local-bootstrap-admin" \
    --data-urlencode "password=${KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD}" |
    node -e 'let body="";process.stdin.on("data",chunk=>body+=chunk);process.stdin.on("end",()=>{const value=JSON.parse(body);if(!value.access_token)process.exit(1);process.stdout.write(value.access_token);});'
)"
node -e 'process.stdout.write(JSON.stringify({username:process.argv[1],email:process.argv[1],firstName:"Local",lastName:"Administrator",enabled:true,emailVerified:true,attributes:{organization_id:["CAA"]},requiredActions:["CONFIGURE_TOTP"]}))' \
  "${APPLICATION_ADMIN_USERNAME}" |
  curl --fail --silent --show-error \
    --request POST \
    "${KEYCLOAK_PUBLIC_URL}/admin/realms/aviasurveil360/users" \
    --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
    --header "Content-Type: application/json" \
    --data-binary @- \
    --output /dev/null
APPLICATION_ADMIN_SUBJECT_ID="$(
  curl --fail --silent --show-error \
    --get \
    "${KEYCLOAK_PUBLIC_URL}/admin/realms/aviasurveil360/users" \
    --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
    --data-urlencode "email=${APPLICATION_ADMIN_USERNAME}" \
    --data-urlencode "exact=true" |
    node -e 'let body="";process.stdin.on("data",chunk=>body+=chunk);process.stdin.on("end",()=>{const users=JSON.parse(body);if(users.length!==1||!users[0].id)process.exit(1);process.stdout.write(users[0].id);});'
)"
if [[ -z "${APPLICATION_ADMIN_SUBJECT_ID}" ]]; then
  echo "Keycloak did not return the application administrator subject" >&2
  exit 1
fi
curl --fail --silent --show-error \
  "${KEYCLOAK_PUBLIC_URL}/admin/realms/aviasurveil360/users/${APPLICATION_ADMIN_SUBJECT_ID}" \
  --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
  --output "${RUNTIME_DIRECTORY}/application-admin.json"
node -e '
  const user = JSON.parse(require("node:fs").readFileSync(process.argv[1], "utf8"));
  const organizationIDs = user.attributes?.organization_id;
  if (!Array.isArray(organizationIDs) || organizationIDs.length !== 1 || organizationIDs[0] !== "CAA") {
    process.stderr.write("Keycloak application administrator is missing the exact CAA organization attribute\n");
    process.exit(1);
  }
' "${RUNTIME_DIRECTORY}/application-admin.json"
KEYCLOAK_WEB_CLIENT_ID="$(
  curl --fail --silent --show-error \
    --get \
    "${KEYCLOAK_PUBLIC_URL}/admin/realms/aviasurveil360/clients" \
    --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
    --data-urlencode "clientId=aviasurveil360-web" |
    node -e 'let body="";process.stdin.on("data",chunk=>body+=chunk);process.stdin.on("end",()=>{const clients=JSON.parse(body);if(clients.length!==1||!clients[0].id)process.exit(1);process.stdout.write(clients[0].id);});'
)"
if [[ -z "${KEYCLOAK_WEB_CLIENT_ID}" ]]; then
  echo "Keycloak did not return the reviewed web client" >&2
  exit 1
fi
curl --fail --silent --show-error \
  "${KEYCLOAK_PUBLIC_URL}/admin/realms/aviasurveil360/clients/${KEYCLOAK_WEB_CLIENT_ID}/protocol-mappers/models" \
  --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
  --output "${RUNTIME_DIRECTORY}/protocol-mappers.json"
node -e '
  const mappers = JSON.parse(require("node:fs").readFileSync(process.argv[1], "utf8"));
  const mapper = mappers.find((candidate) => candidate.name === "AviaSurveil360 organization");
  if (
    mapper?.protocolMapper !== "oidc-usermodel-attribute-mapper" ||
    mapper.config?.["user.attribute"] !== "organization_id" ||
    mapper.config?.["claim.name"] !== "organization_id" ||
    mapper.config?.["id.token.claim"] !== "true"
  ) {
    process.stderr.write("Keycloak web client is missing the reviewed organization ID-token mapper\n");
    process.exit(1);
  }
' "${RUNTIME_DIRECTORY}/protocol-mappers.json"
APPLICATION_ADMIN_ROLE="$(
  curl --fail --silent --show-error \
    "${KEYCLOAK_PUBLIC_URL}/admin/realms/aviasurveil360/roles/admin" \
    --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}"
)"
node -e 'process.stdout.write(JSON.stringify([JSON.parse(process.argv[1])]))' \
  "${APPLICATION_ADMIN_ROLE}" |
  curl --fail --silent --show-error \
    --request POST \
    "${KEYCLOAK_PUBLIC_URL}/admin/realms/aviasurveil360/users/${APPLICATION_ADMIN_SUBJECT_ID}/role-mappings/realm" \
    --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
    --header "Content-Type: application/json" \
    --data-binary @- \
    --output /dev/null
node -e 'process.stdout.write(JSON.stringify({type:"password",temporary:false,value:process.argv[1]}))' \
  "${APPLICATION_ADMIN_PASSWORD}" |
  curl --fail --silent --show-error \
    --request PUT \
    "${KEYCLOAK_PUBLIC_URL}/admin/realms/aviasurveil360/users/${APPLICATION_ADMIN_SUBJECT_ID}/reset-password" \
    --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
    --header "Content-Type: application/json" \
    --data-binary @- \
    --output /dev/null
unset KEYCLOAK_ADMIN_TOKEN APPLICATION_ADMIN_ROLE

export AVIA_TEST_DATABASE_URL="postgres://aviasurveil:${APP_DATABASE_PASSWORD}@127.0.0.1:${TASK_POSTGRES_PORT}/aviasurveil?sslmode=disable"
export AVIA_TEST_OIDC_ISSUER_URL="${KEYCLOAK_PUBLIC_URL}/realms/aviasurveil360"
export AVIA_TEST_OIDC_CLIENT_ID="aviasurveil360-web"
export AVIA_TEST_OIDC_CLIENT_SECRET="${OIDC_CLIENT_SECRET}"
export AVIA_TEST_OIDC_REDIRECT_URL="http://127.0.0.1:4174/auth/callback"
export AVIA_TEST_OBJECT_STORE_ENDPOINT="127.0.0.1:${TASK_OBJECT_STORE_PORT}"
export AVIA_TEST_OBJECT_STORE_ACCESS_KEY="${MINIO_ROOT_USER}"
export AVIA_TEST_OBJECT_STORE_SECRET_KEY="${MINIO_ROOT_PASSWORD}"
export AVIA_ENVIRONMENT="test"
export AVIA_DATABASE_URL="${AVIA_TEST_DATABASE_URL}"
export AVIA_HTTP_ADDRESS="127.0.0.1:${TASK_API_PORT}"
export AVIA_OIDC_ISSUER_URL="${AVIA_TEST_OIDC_ISSUER_URL}"
export AVIA_OIDC_CLIENT_ID="${AVIA_TEST_OIDC_CLIENT_ID}"
export AVIA_OIDC_CLIENT_SECRET="${AVIA_TEST_OIDC_CLIENT_SECRET}"
export AVIA_OIDC_REDIRECT_URL="${AVIA_TEST_OIDC_REDIRECT_URL}"
export AVIA_SESSION_ENCRYPTION_KEY="${SESSION_ENCRYPTION_KEY}"
export AVIA_ENABLE_CANONICAL_SEED="false"
export AVIA_ENABLE_CANONICAL_TEST_PROFILE="false"
export AVIA_OBJECT_STORE_ENDPOINT="${AVIA_TEST_OBJECT_STORE_ENDPOINT}"
export AVIA_OBJECT_STORE_ACCESS_KEY="${MINIO_ROOT_USER}"
export AVIA_OBJECT_STORE_SECRET_KEY="${MINIO_ROOT_PASSWORD}"
export AVIA_OBJECT_STORE_TLS="false"
export AVIA_OBJECT_STORE_CORS_ORIGINS="http://127.0.0.1:4174"
export AVIA_OBJECT_STORE_SERVER_MANAGED_CORS="true"
export AVIA_OBJECT_STORE_QUARANTINE_BUCKET="avia-quarantine"
export AVIA_OBJECT_STORE_CANONICAL_BUCKET="avia-canonical"
export AVIA_SCANNER_MODE="clamav"
export AVIA_WORKER_INTERVAL_MS="50"
export AVIA_KEYCLOAK_ADMIN_URL="${KEYCLOAK_PUBLIC_URL}"
export AVIA_KEYCLOAK_REALM="aviasurveil360"
export AVIA_KEYCLOAK_SERVICE_CLIENT_ID="aviasurveil360-lifecycle"
export AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET="${KEYCLOAK_SERVICE_CLIENT_SECRET}"
export AVIA_HTTP_API_URL="http://127.0.0.1:${TASK_API_PORT}"
export AVIA_HTTP_API_TARGET="${AVIA_HTTP_API_URL}"
export AVIA_HTTP_TEST_PROFILE=""
export AVIA_OIDC_TEST_ADMIN_USERNAME="${APPLICATION_ADMIN_USERNAME}"
export AVIA_OIDC_TEST_ADMIN_PASSWORD="${APPLICATION_ADMIN_PASSWORD}"
export AVIA_OIDC_TEST_KEYCLOAK_BASE_URL="${KEYCLOAK_PUBLIC_URL}"
export AVIA_OIDC_TEST_KEYCLOAK_ADMIN_USERNAME="local-bootstrap-admin"
export AVIA_OIDC_TEST_KEYCLOAK_ADMIN_PASSWORD="${KEYCLOAK_BOOTSTRAP_ADMIN_PASSWORD}"
export AVIA_OIDC_TEST_COMPOSE_FILE="${COMPOSE_FILE}"
export AVIA_OIDC_TEST_COMPOSE_PROJECT="${COMPOSE_PROJECT}"
export AVIA_PLAYWRIGHT_OUTPUT_DIR="${RUNTIME_DIRECTORY}/playwright-results"
export GOCACHE="${TASK_GO_CACHE}"
seed_task_go_cache

go -C "${REPOSITORY_ROOT}/apps/api" build -o "${RUNTIME_DIRECTORY}/api" ./cmd/api
go -C "${REPOSITORY_ROOT}/apps/api" build -o "${RUNTIME_DIRECTORY}/worker" ./cmd/worker

(
  cd "${REPOSITORY_ROOT}/apps/api"
  exec "${RUNTIME_DIRECTORY}/api"
) >"${RUNTIME_DIRECTORY}/api.log" 2>&1 &
API_PID=$!

(
  cd "${REPOSITORY_ROOT}/apps/api"
  exec "${RUNTIME_DIRECTORY}/worker"
) >"${RUNTIME_DIRECTORY}/worker.log" 2>&1 &
WORKER_PID=$!

for _ in {1..120}; do
  if curl --fail --silent "${AVIA_HTTP_API_URL}/health/ready" >/dev/null; then
    break
  fi
  if ! kill -0 "${API_PID}" 2>/dev/null; then
    echo "API exited before readiness" >&2
    exit 1
  fi
  sleep 0.25
done
curl --fail --silent "${AVIA_HTTP_API_URL}/health/ready" >/dev/null
kill -0 "${WORKER_PID}"

npm --prefix "${REPOSITORY_ROOT}/apps/web" run typecheck
npm --prefix "${REPOSITORY_ROOT}/apps/web" run test:e2e:oidc
scan_runtime_artifacts_for_secret_leaks
