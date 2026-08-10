#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_FILE="${REPOSITORY_ROOT}/deploy/local/compose.test.yaml"
COMPOSE_PROJECT="aviasurveil360-task13-recovery"
RUNTIME_DIRECTORY="$(mktemp -d /private/tmp/aviasurveil360-task13-recovery.XXXXXX)"
API_PID=""
export COMPOSE_PROGRESS="plain"
export AVIA_TEST_RUNTIME_DIR="${RUNTIME_DIRECTORY}"

. "${REPOSITORY_ROOT}/scripts/lib/init-local-test-runtime.sh"
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
  if [[ -n "${API_PID}" ]]; then
    kill "${API_PID}" 2>/dev/null
    wait "${API_PID}" 2>/dev/null
  fi
  docker compose --project-name "${COMPOSE_PROJECT}" --file "${COMPOSE_FILE}" down --volumes --remove-orphans
  rm -rf "${RUNTIME_DIRECTORY}"
  exit "${status}"
}
trap cleanup EXIT

docker compose --project-name "${COMPOSE_PROJECT}" --file "${COMPOSE_FILE}" down --volumes --remove-orphans
docker compose --project-name "${COMPOSE_PROJECT}" --file "${COMPOSE_FILE}" up --detach --wait postgres object-store

export AVIA_ENVIRONMENT="test"
export AVIA_DATABASE_URL="postgres://aviasurveil:${APP_DATABASE_PASSWORD}@127.0.0.1:55432/aviasurveil?sslmode=disable"
export AVIA_HTTP_ADDRESS="127.0.0.1:58083"
export AVIA_ENABLE_CANONICAL_SEED="true"
export AVIA_ENABLE_CANONICAL_TEST_PROFILE="true"
export AVIA_CANONICAL_TEST_TOKEN="$(openssl rand -hex 32)"
export AVIA_OBJECT_STORE_ENDPOINT="127.0.0.1:59001"
export AVIA_OBJECT_STORE_ACCESS_KEY="${MINIO_ROOT_USER}"
export AVIA_OBJECT_STORE_SECRET_KEY="${MINIO_ROOT_PASSWORD}"
export AVIA_OBJECT_STORE_TLS="false"
export AVIA_OBJECT_STORE_CORS_ORIGINS="http://127.0.0.1:4174"
export AVIA_OBJECT_STORE_QUARANTINE_BUCKET="avia-quarantine"
export AVIA_OBJECT_STORE_CANONICAL_BUCKET="avia-canonical"
export AVIA_SCANNER_MODE="deterministic-test"
export AVIA_WORKER_INTERVAL_MS="1000"
export AVIA_ENABLE_LOCAL_RECOVERY_DRILL="true"
export AVIA_RECOVERY_DRILL_DIRECTORY="${RUNTIME_DIRECTORY}"
export GOCACHE="${RUNTIME_DIRECTORY}/go-cache"
export GOTMPDIR="${RUNTIME_DIRECTORY}/go-tmp"
mkdir -p "${GOCACHE}" "${GOTMPDIR}"

go -C "${REPOSITORY_ROOT}/apps/api" build -tags canonicaltest -o "${RUNTIME_DIRECTORY}/api" ./cmd/api
(
  cd "${REPOSITORY_ROOT}/apps/api"
  exec "${RUNTIME_DIRECTORY}/api"
) >"${RUNTIME_DIRECTORY}/api.log" 2>&1 &
API_PID=$!

for _ in {1..120}; do
  if curl --fail --silent "http://127.0.0.1:58083/health/ready" >/dev/null; then
    break
  fi
  if ! kill -0 "${API_PID}" 2>/dev/null; then
    tail -n 200 "${RUNTIME_DIRECTORY}/api.log" >&2
    exit 1
  fi
  sleep 0.25
done
curl --fail --silent "http://127.0.0.1:58083/health/ready" >/dev/null
kill "${API_PID}"
wait "${API_PID}" || true
API_PID=""

POSTGRES_CONTAINER="$(docker compose --project-name "${COMPOSE_PROJECT}" --file "${COMPOSE_FILE}" ps --quiet postgres)"
if [[ -z "${POSTGRES_CONTAINER}" ]]; then
  echo "PostgreSQL recovery container was not resolved" >&2
  exit 1
fi

docker exec "${POSTGRES_CONTAINER}" pg_dump --username aviasurveil --dbname aviasurveil --format custom --no-owner --no-privileges >"${RUNTIME_DIRECTORY}/postgres.dump"
if [[ ! -s "${RUNTIME_DIRECTORY}/postgres.dump" ]]; then
  echo "PostgreSQL backup artifact is empty" >&2
  exit 1
fi

FINGERPRINT_SQL="SELECT jsonb_build_object('migration',(SELECT max(version) FROM schema_migrations),'organizations',(SELECT jsonb_agg(id ORDER BY id) FROM organizations),'inspections',(SELECT jsonb_agg(id ORDER BY id) FROM inspections),'templates',(SELECT jsonb_agg(id ORDER BY id) FROM checklist_template_versions),'plans',(SELECT jsonb_agg(id ORDER BY id) FROM surveillance_plan_items),'reminders',(SELECT jsonb_agg(id ORDER BY id) FROM reminder_rules))::text"
SOURCE_FINGERPRINT="$(docker exec "${POSTGRES_CONTAINER}" psql --username aviasurveil --dbname aviasurveil --tuples-only --no-align --command "${FINGERPRINT_SQL}")"

docker exec "${POSTGRES_CONTAINER}" dropdb --username aviasurveil --if-exists aviasurveil_restore
docker exec "${POSTGRES_CONTAINER}" createdb --username aviasurveil aviasurveil_restore
docker exec --interactive "${POSTGRES_CONTAINER}" pg_restore --username aviasurveil --dbname aviasurveil_restore --no-owner --no-privileges <"${RUNTIME_DIRECTORY}/postgres.dump"
RESTORED_FINGERPRINT="$(docker exec "${POSTGRES_CONTAINER}" psql --username aviasurveil --dbname aviasurveil_restore --tuples-only --no-align --command "${FINGERPRINT_SQL}")"
if [[ -z "${SOURCE_FINGERPRINT}" || "${SOURCE_FINGERPRINT}" != "${RESTORED_FINGERPRINT}" ]]; then
  echo "PostgreSQL restored fingerprint differs from source" >&2
  exit 1
fi

go -C "${REPOSITORY_ROOT}/apps/api" run ./cmd/local-recovery-drill
POSTGRES_SHA256="$(shasum -a 256 "${RUNTIME_DIRECTORY}/postgres.dump" | awk '{print $1}')"
docker exec "${POSTGRES_CONTAINER}" dropdb --username aviasurveil --if-exists aviasurveil_restore

echo "PostgreSQL backup/restore: verified locally"
echo "PostgreSQL backup SHA-256: ${POSTGRES_SHA256}"
echo "Object-store exact-byte backup/restore: verified locally"
echo "Artifact status: candidate-only"
