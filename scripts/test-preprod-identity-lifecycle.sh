#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_FILE="${REPOSITORY_ROOT}/deploy/local/compose.test.yaml"
COMPOSE_PROJECT="aviasurveil360-plan5-task3"
TASK_POSTGRES_PORT="${AVIA_PLAN5_TASK3_POSTGRES_PORT:-55445}"
TASK_MAILPIT_HTTP_PORT="${AVIA_PLAN5_TASK3_MAILPIT_HTTP_PORT:-58095}"
TASK_OBJECT_STORE_PORT="${AVIA_PLAN5_TASK3_OBJECT_STORE_PORT:-59001}"
TASK_OBJECT_STORE_CONSOLE_PORT="${AVIA_PLAN5_TASK3_OBJECT_STORE_CONSOLE_PORT:-59002}"
RUNTIME_DIRECTORY="$(mktemp -d /private/tmp/aviasurveil360-plan5-task3.XXXXXX)"
TEST_RUNTIME_HELPER="${REPOSITORY_ROOT}/scripts/lib/init-local-test-runtime.sh"
MODE="${1:-}"

case "${MODE}" in
  session-authority) ;;
  *)
    echo "usage: $0 session-authority" >&2
    exit 64
    ;;
esac

export COMPOSE_PROGRESS="plain"
export AVIA_TEST_POSTGRES_PORT="${TASK_POSTGRES_PORT}"
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
  up --detach --wait postgres mailpit object-store

export AVIA_TEST_DATABASE_URL="postgres://aviasurveil:${APP_DATABASE_PASSWORD}@127.0.0.1:${TASK_POSTGRES_PORT}/aviasurveil?sslmode=disable"
export AVIA_TEST_OBJECT_STORE_ENDPOINT="127.0.0.1:${TASK_OBJECT_STORE_PORT}"
export AVIA_TEST_OBJECT_STORE_ACCESS_KEY="${MINIO_ROOT_USER}"
export AVIA_TEST_OBJECT_STORE_SECRET_KEY="${MINIO_ROOT_PASSWORD}"
export AVIA_TEST_MAILPIT_HTTP_URL="http://127.0.0.1:${TASK_MAILPIT_HTTP_PORT}"

go -C "${REPOSITORY_ROOT}/apps/api" test \
  -tags canonicaltest -race -p 1 -count=1 -timeout=20m \
  ./internal/platform/session ./internal/identity \
  ./internal/administration ./internal/httpapi ./tests/integration
echo "API session authority and integration harness: verified locally"
