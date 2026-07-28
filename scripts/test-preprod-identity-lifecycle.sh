#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_FILE="${REPOSITORY_ROOT}/deploy/local/compose.test.yaml"
COMPOSE_PROJECT="aviasurveil360-plan5-task3"
TASK_POSTGRES_PORT="${AVIA_PLAN5_TASK3_POSTGRES_PORT:-55445}"
TASK_KEYCLOAK_PORT="${AVIA_PLAN5_TASK3_KEYCLOAK_PORT:-58094}"
TASK_KEYCLOAK_MANAGEMENT_PORT="${AVIA_PLAN5_TASK3_KEYCLOAK_MANAGEMENT_PORT:-59016}"
TASK_MAILPIT_HTTP_PORT="${AVIA_PLAN5_TASK3_MAILPIT_HTTP_PORT:-58095}"
TASK_OBJECT_STORE_PORT="${AVIA_PLAN5_TASK3_OBJECT_STORE_PORT:-59001}"
TASK_OBJECT_STORE_CONSOLE_PORT="${AVIA_PLAN5_TASK3_OBJECT_STORE_CONSOLE_PORT:-59002}"
RUNTIME_DIRECTORY="$(mktemp -d /private/tmp/aviasurveil360-plan5-task3.XXXXXX)"
TEST_RUNTIME_HELPER="${REPOSITORY_ROOT}/scripts/lib/init-local-test-runtime.sh"
MODE="${1:-}"

case "${MODE}" in
  invitation | all-eight-roles | session-authority) ;;
  *)
    echo "usage: $0 invitation|all-eight-roles|session-authority" >&2
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

. "${TEST_RUNTIME_HELPER}"
initialize_local_test_runtime \
  "${RUNTIME_DIRECTORY}" \
  "http://127.0.0.1:4174" \
  "${REPOSITORY_ROOT}"

read_runtime_secret() {
  tr -d '\r\n' <"${RUNTIME_DIRECTORY}/secrets/$1"
}

APP_DATABASE_PASSWORD="$(read_runtime_secret app_database_password)"
KEYCLOAK_SERVICE_CLIENT_SECRET="$(
  read_runtime_secret keycloak_service_client_secret
)"
MINIO_ROOT_PASSWORD="$(read_runtime_secret minio_root_password)"
MINIO_ROOT_USER="$(read_runtime_secret minio_root_user)"

cleanup() {
  local status=$?
  trap - EXIT
  set +e
  docker compose \
    --project-name "${COMPOSE_PROJECT}" \
    --file "${COMPOSE_FILE}" \
    down --volumes --remove-orphans
  rm -rf "${RUNTIME_DIRECTORY}"
  exit "${status}"
}
trap cleanup EXIT

docker compose \
  --project-name "${COMPOSE_PROJECT}" \
  --file "${COMPOSE_FILE}" \
  down --volumes --remove-orphans
docker compose \
  --project-name "${COMPOSE_PROJECT}" \
  --file "${COMPOSE_FILE}" \
  up --detach --wait postgres keycloak-postgres mailpit keycloak object-store

export AVIA_TEST_DATABASE_URL="postgres://aviasurveil:${APP_DATABASE_PASSWORD}@127.0.0.1:${TASK_POSTGRES_PORT}/aviasurveil?sslmode=disable"
export AVIA_TEST_OBJECT_STORE_ENDPOINT="127.0.0.1:${TASK_OBJECT_STORE_PORT}"
export AVIA_TEST_OBJECT_STORE_ACCESS_KEY="${MINIO_ROOT_USER}"
export AVIA_TEST_OBJECT_STORE_SECRET_KEY="${MINIO_ROOT_PASSWORD}"
export AVIA_TEST_KEYCLOAK_ADMIN_URL="http://127.0.0.1:${TASK_KEYCLOAK_PORT}/identity"
export AVIA_TEST_KEYCLOAK_REALM="aviasurveil360"
export AVIA_TEST_KEYCLOAK_SERVICE_CLIENT_ID="aviasurveil360-lifecycle"
export AVIA_TEST_KEYCLOAK_SERVICE_CLIENT_SECRET="${KEYCLOAK_SERVICE_CLIENT_SECRET}"
export AVIA_TEST_MAILPIT_HTTP_URL="http://127.0.0.1:${TASK_MAILPIT_HTTP_PORT}"

if [[ "${MODE}" == "all-eight-roles" ]]; then
  go -C "${REPOSITORY_ROOT}/apps/api" test \
    -tags canonicaltest -race -p 1 -count=1 \
    ./tests/integration \
    -run '^TestTask3KeycloakInvitationAllEightRoles$'
  echo "Plan 5 Task 3 all-eight-role identity lifecycle: verified locally"
  exit 0
fi

if [[ "${MODE}" == "session-authority" ]]; then
  go -C "${REPOSITORY_ROOT}/apps/api" test \
    -tags canonicaltest -race -p 1 -count=1 \
    ./internal/platform/session ./internal/identity \
    ./internal/administration ./internal/httpapi ./tests/integration
  echo "Plan 5 Task 4 session authority: verified locally"
  exit 0
fi

go -C "${REPOSITORY_ROOT}/apps/api" test \
  -tags canonicaltest -race -p 1 -count=1 \
  ./internal/administration ./internal/identity ./internal/notifications \
  ./tests/integration
echo "Plan 5 Task 3 invitation and lifecycle harness: verified locally"
