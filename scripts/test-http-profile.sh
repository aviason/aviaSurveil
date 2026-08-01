#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_FILE="${REPOSITORY_ROOT}/deploy/local/compose.test.yaml"
COMPOSE_PROJECT="aviasurveil360-task11"
TASK_POSTGRES_PORT="${AVIA_TASK11_POSTGRES_PORT:-55442}"
TASK_KEYCLOAK_PORT="${AVIA_TASK11_KEYCLOAK_PORT:-58090}"
TASK_KEYCLOAK_MANAGEMENT_PORT="${AVIA_TASK11_KEYCLOAK_MANAGEMENT_PORT:-59010}"
TASK_MAILPIT_HTTP_PORT="${AVIA_TASK11_MAILPIT_HTTP_PORT:-58095}"
TASK_OBJECT_STORE_PORT="${AVIA_TASK11_OBJECT_STORE_PORT:-59011}"
TASK_OBJECT_STORE_CONSOLE_PORT="${AVIA_TASK11_OBJECT_STORE_CONSOLE_PORT:-59012}"
TASK_API_PORT="${AVIA_TASK11_API_PORT:-58091}"
RUNTIME_DIRECTORY="$(mktemp -d /private/tmp/aviasurveil360-task11.XXXXXX)"
TEST_RUNTIME_HELPER="${REPOSITORY_ROOT}/scripts/lib/init-local-test-runtime.sh"
SHARED_GO_CACHE="$(go env GOCACHE)"
TASK_GO_CACHE="${RUNTIME_DIRECTORY}/go-cache"
TASK_GO_TMP="${RUNTIME_DIRECTORY}/go-tmp"
API_PID=""
WORKER_PID=""
FOCUSED_E2E="${AVIA_HTTP_PROFILE_FOCUSED_E2E:-}"
case "${FOCUSED_E2E}" in
  "" | user-lifecycle | visible-actions | governed-checklist | governed-checklist-intake | regulatory-source-refresh) ;;
  *)
    echo "AVIA_HTTP_PROFILE_FOCUSED_E2E must be empty, user-lifecycle, visible-actions, governed-checklist, governed-checklist-intake, or regulatory-source-refresh" >&2
    exit 64
    ;;
esac
export COMPOSE_PROGRESS="plain"
export AVIA_TEST_POSTGRES_PORT="${TASK_POSTGRES_PORT}"
export AVIA_TEST_KEYCLOAK_PORT="${TASK_KEYCLOAK_PORT}"
export AVIA_TEST_KEYCLOAK_MANAGEMENT_PORT="${TASK_KEYCLOAK_MANAGEMENT_PORT}"
export AVIA_TEST_MAILPIT_HTTP_PORT="${TASK_MAILPIT_HTTP_PORT}"
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

seed_task_go_cache() {
  mkdir -p "${TASK_GO_CACHE}"
  if [[ -d "${SHARED_GO_CACHE}" && "${SHARED_GO_CACHE}" != "${TASK_GO_CACHE}" ]]; then
    cp -al "${SHARED_GO_CACHE}/." "${TASK_GO_CACHE}/"
  fi
}

cleanup() {
  local status=$?
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
  if [[ ${status} -ne 0 ]]; then
    for log_file in "${RUNTIME_DIRECTORY}"/*.log; do
      if [[ -f "${log_file}" ]]; then
        echo "--- ${log_file} ---" >&2
        tail -n 200 "${log_file}" >&2
      fi
    done
  fi
  env GOCACHE="${TASK_GO_CACHE}" go clean -cache
  docker compose --project-name "${COMPOSE_PROJECT}" --file "${COMPOSE_FILE}" down --volumes --remove-orphans
  rm -rf "${RUNTIME_DIRECTORY}"
  exit "${status}"
}
trap cleanup EXIT

node "${REPOSITORY_ROOT}/scripts/verify-governed-checklist-test-inventory.mjs"

docker compose --project-name "${COMPOSE_PROJECT}" --file "${COMPOSE_FILE}" down --volumes --remove-orphans
docker compose --project-name "${COMPOSE_PROJECT}" --file "${COMPOSE_FILE}" up --detach --wait postgres keycloak-postgres mailpit keycloak object-store

export AVIA_TEST_DATABASE_URL="postgres://aviasurveil:${APP_DATABASE_PASSWORD}@127.0.0.1:${TASK_POSTGRES_PORT}/aviasurveil?sslmode=disable"
export AVIA_TEST_OIDC_ISSUER_URL="http://127.0.0.1:${TASK_KEYCLOAK_PORT}/identity/realms/aviasurveil360"
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
export AVIA_ENABLE_CANONICAL_SEED="true"
export AVIA_ENABLE_CANONICAL_TEST_PROFILE="true"
export AVIA_CANONICAL_TEST_TOKEN="${AVIA_CANONICAL_TEST_TOKEN:-$(openssl rand -hex 32)}"
export AVIA_OBJECT_STORE_ENDPOINT="${AVIA_TEST_OBJECT_STORE_ENDPOINT}"
export AVIA_OBJECT_STORE_ACCESS_KEY="${MINIO_ROOT_USER}"
export AVIA_OBJECT_STORE_SECRET_KEY="${MINIO_ROOT_PASSWORD}"
export AVIA_OBJECT_STORE_TLS="false"
export AVIA_OBJECT_STORE_CORS_ORIGINS="http://127.0.0.1:4174"
export AVIA_OBJECT_STORE_QUARANTINE_BUCKET="avia-quarantine"
export AVIA_OBJECT_STORE_CANONICAL_BUCKET="avia-canonical"
export AVIA_SCANNER_MODE="deterministic-test"
export AVIA_WORKER_INTERVAL_MS="50"
export AVIA_KEYCLOAK_ADMIN_URL="http://127.0.0.1:${TASK_KEYCLOAK_PORT}/identity"
export AVIA_KEYCLOAK_REALM="aviasurveil360"
export AVIA_KEYCLOAK_SERVICE_CLIENT_ID="aviasurveil360-lifecycle"
export AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET="${KEYCLOAK_SERVICE_CLIENT_SECRET}"
export AVIA_HTTP_API_URL="http://127.0.0.1:${TASK_API_PORT}"
export AVIA_HTTP_API_TARGET="${AVIA_HTTP_API_URL}"
export AVIA_HTTP_TEST_PROFILE="canonical"
export GOCACHE="${TASK_GO_CACHE}"
seed_task_go_cache

"${SCRIPT_DIR}/reset-test-profile.sh"

go -C "${REPOSITORY_ROOT}/apps/api" build -tags canonicaltest -o "${RUNTIME_DIRECTORY}/api" ./cmd/api
go -C "${REPOSITORY_ROOT}/apps/api" build -o "${RUNTIME_DIRECTORY}/worker" ./cmd/worker
if [[ -z "${FOCUSED_E2E}" ]]; then
  go -C "${REPOSITORY_ROOT}/apps/api" test -race -p 1 -count=1 ./...
  "${SCRIPT_DIR}/check-contracts.sh"
  "${SCRIPT_DIR}/check-sqlc.sh"
fi

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
if [[ "${FOCUSED_E2E}" == "visible-actions" ]]; then
  npm --prefix "${REPOSITORY_ROOT}/apps/web" run test:e2e:http -- \
    tests/e2e/visible-action-contract.spec.ts
elif [[ "${FOCUSED_E2E}" == "user-lifecycle" ]]; then
  npm --prefix "${REPOSITORY_ROOT}/apps/web" run test:e2e:http -- \
    --grep "user lifecycle"
elif [[ "${FOCUSED_E2E}" == "governed-checklist" ]]; then
  npm --prefix "${REPOSITORY_ROOT}/apps/web" run test:contract:http
  go -C "${REPOSITORY_ROOT}/apps/api" test -tags canonicaltest ./tests/integration \
    -run '^TestTask9SyntheticPublicationAndBlockedRealPilotHaveSeparatePersistedEffects$' \
    -count=1
  npm --prefix "${REPOSITORY_ROOT}/apps/web" run test:e2e:http -- \
    tests/e2e/regulatory-checklist-governance.http.spec.ts
elif [[ "${FOCUSED_E2E}" == "governed-checklist-intake" ]]; then
  npm --prefix "${REPOSITORY_ROOT}/apps/web" run test:contract:http -- --run src/backend/governed-checklist-intake-parity.test.ts
  go -C "${REPOSITORY_ROOT}/apps/api" test -tags canonicaltest ./internal/checklistintake ./internal/regulatory ./tests/integration \
    -run '^(Test(AGAZipPDFV1LimitsAreFrozen|InventoryArchive|ParseBoundedPDF|IdentityResolution|AGAForm048CandidateIntake|GovernedChecklistDualAuthoring|AGAForm048Reconciliation|GovernedChecklistIntakeLifecycle))$' \
    -count=1
  npm --prefix "${REPOSITORY_ROOT}/apps/web" run test:e2e:http -- \
    tests/e2e/governed-checklist-intake.http.spec.ts
elif [[ "${FOCUSED_E2E}" == "regulatory-source-refresh" ]]; then
  go -C "${REPOSITORY_ROOT}/apps/api" test -tags canonicaltest ./tests/integration \
    -run '^(TestRegulatorySourceRefreshTask6.*|TestTask6DepartmentFilteredQueueAndCurrentAssignmentAuthority|TestTask6PublicationIsSeparateDigestVerifiedAndImmutable|TestTask6CanonicalHTTPManagerLifecycleUsesRealPostgreSQLAuthority|TestTask6TechnicalApprovalFailsClosedOnPersistedSourceGap|TestTask6SubmissionAndPublicationFailClosedOnPersistedSourceGap|TestTask9SyntheticPublicationAndBlockedRealPilotHaveSeparatePersistedEffects)$' \
    -count=1
  npm --prefix "${REPOSITORY_ROOT}/apps/web" run test:e2e:http -- \
    tests/e2e/regulatory-source-refresh.http.spec.ts
else
  npm --prefix "${REPOSITORY_ROOT}/apps/web" test
  npm --prefix "${REPOSITORY_ROOT}/apps/web" run build:demo
  npm --prefix "${REPOSITORY_ROOT}/apps/web" run build:http
  node "${REPOSITORY_ROOT}/apps/web/scripts/assert-http-artifact.mjs" "${REPOSITORY_ROOT}/apps/web/dist/http"
  npm --prefix "${REPOSITORY_ROOT}/apps/web" run test:contract:http
  npm --prefix "${REPOSITORY_ROOT}/apps/web" run test:e2e:mock
  npm --prefix "${REPOSITORY_ROOT}/apps/web" run test:e2e:http
fi

for _ in {1..120}; do
  PENDING_SCAN_COUNT="$(docker compose --project-name "${COMPOSE_PROJECT}" --file "${COMPOSE_FILE}" exec --no-TTY postgres psql --username aviasurveil --dbname aviasurveil --tuples-only --no-align --command "SELECT count(*) FROM outbox_messages WHERE topic IN ('evidence.scan_requested', 'inspection_attachment.scan_requested') AND delivered_at IS NULL AND terminal_state IS NULL")"
  if [[ "${PENDING_SCAN_COUNT}" == "0" ]]; then
    break
  fi
  sleep 0.25
done
TERMINAL_SCAN_COUNT="$(docker compose --project-name "${COMPOSE_PROJECT}" --file "${COMPOSE_FILE}" exec --no-TTY postgres psql --username aviasurveil --dbname aviasurveil --tuples-only --no-align --command "SELECT count(*) FROM outbox_messages WHERE topic IN ('evidence.scan_requested', 'inspection_attachment.scan_requested') AND terminal_state IS NOT NULL")"
if [[ "${PENDING_SCAN_COUNT}" != "0" || "${TERMINAL_SCAN_COUNT}" != "0" ]]; then
  echo "Worker outbox is not drained: pending=${PENDING_SCAN_COUNT} terminal=${TERMINAL_SCAN_COUNT}" >&2
  exit 1
fi
if [[ -z "${FOCUSED_E2E}" ]] &&
  ! grep -q "scan work batch completed" "${RUNTIME_DIRECTORY}/worker.log"; then
  echo "Worker log has no completed-batch observability event" >&2
  exit 1
fi
if [[ -z "${FOCUSED_E2E}" ]]; then
  echo "Worker/outbox observability: verified locally"
else
  echo "Focused HTTP outbox drain: verified locally"
fi
