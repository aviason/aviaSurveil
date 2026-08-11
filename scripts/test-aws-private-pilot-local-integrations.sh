#!/usr/bin/env bash
set -euo pipefail

script_directory="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repository_root="$(cd "${script_directory}/.." && pwd)"
compose_file="${repository_root}/deploy/local/compose.test.yaml"
network_overlay="${repository_root}/deploy/local/compose.test.external-network.yaml"
runtime_directory="$(mktemp -d /private/tmp/avia-aws-pilot-integration.XXXXXX)"
project="aviasurveil360-task-aws-private-pilot-$(date -u +%Y%m%d%H%M%S)-$$"
network="${project}-network"
test_subnet="${AVIA_AWS_PILOT_TEST_SUBNET:-172.31.254.0/28}"
network_created=false

case "${runtime_directory}" in
  /private/tmp/avia-aws-pilot-integration.*) ;;
  *)
    echo "unexpected task runtime directory" >&2
    exit 1
    ;;
esac

cleanup() {
  local exit_code=$?
  trap - EXIT INT TERM
  set +e
  if [[ "${exit_code}" -ne 0 ]]; then
    docker compose \
      --project-name "${project}" \
      --file "${compose_file}" \
      --file "${network_overlay}" \
      logs --no-color --tail 120 2>/dev/null || true
  fi
  docker compose \
    --project-name "${project}" \
    --file "${compose_file}" \
    --file "${network_overlay}" \
    down --volumes --remove-orphans --timeout 15 >/dev/null 2>&1 || true
  if [[ "${network_created}" == true ]]; then
    docker network rm "${network}" >/dev/null 2>&1 || true
  fi
  rm -rf -- "${runtime_directory}"
  if docker ps --all --quiet \
      --filter "label=com.docker.compose.project=${project}" | grep -q . ||
    docker volume ls --quiet \
      --filter "label=com.docker.compose.project=${project}" | grep -q . ||
    docker network ls --quiet \
      --filter "label=com.docker.compose.project=${project}" | grep -q .; then
    echo "task-owned Docker residue remains for ${project}" >&2
    exit 1
  fi
  if [[ "${exit_code}" -eq 0 ]]; then
    echo "verified locally: task-owned PostgreSQL/MinIO/ClamAV/Mailpit integration and zero residue"
  fi
  exit "${exit_code}"
}
trap cleanup EXIT INT TERM

choose_port_block() {
  local attempt base_port offset free
  for attempt in {1..200}; do
    base_port=$((55000 + RANDOM % 8000))
    free=true
    for offset in {0..5}; do
      if nc -z 127.0.0.1 "$((base_port + offset))" >/dev/null 2>&1; then
        free=false
        break
      fi
    done
    if [[ "${free}" == true ]]; then
      printf '%s\n' "${base_port}"
      return 0
    fi
  done
  return 1
}

port_base="$(choose_port_block)"
export AVIA_TEST_POSTGRES_PORT="${port_base}"
export AVIA_TEST_OBJECT_STORE_PORT="$((port_base + 1))"
export AVIA_TEST_OBJECT_STORE_CONSOLE_PORT="$((port_base + 2))"
export AVIA_TEST_CLAMAV_PORT="$((port_base + 3))"
export AVIA_TEST_MAILPIT_SMTP_PORT="$((port_base + 4))"
export AVIA_TEST_MAILPIT_HTTP_PORT="$((port_base + 5))"
export AVIA_TEST_RUNTIME_DIR="${runtime_directory}"
export AVIA_TEST_EXTERNAL_NETWORK="${network}"
export COMPOSE_PROGRESS=plain

docker network create \
  --driver bridge \
  --subnet "${test_subnet}" \
  --label "com.docker.compose.project=${project}" \
  "${network}" >/dev/null
network_created=true

# shellcheck source=scripts/lib/init-local-test-runtime.sh
. "${repository_root}/scripts/lib/init-local-test-runtime.sh"
initialize_local_test_runtime \
  "${runtime_directory}" \
  "http://127.0.0.1:4174" \
  "${repository_root}"

docker compose \
  --project-name "${project}" \
  --file "${compose_file}" \
  --file "${network_overlay}" \
  up --detach --wait postgres object-store clamav mailpit

read_secret() {
  tr -d '\r\n' <"${runtime_directory}/secrets/$1"
}

app_password="$(read_secret app_database_password)"
root_user="$(read_secret minio_root_user)"
root_password="$(read_secret minio_root_password)"
api_access="$(read_secret minio_api_access_key)"
api_secret="$(read_secret minio_api_secret_key)"
worker_access="$(read_secret minio_worker_access_key)"
worker_secret="$(read_secret minio_worker_secret_key)"

export AVIA_TEST_DATABASE_URL="postgres://aviasurveil:${app_password}@127.0.0.1:${AVIA_TEST_POSTGRES_PORT}/aviasurveil?sslmode=disable"
export AVIA_TEST_OBJECT_STORE_ENDPOINT="127.0.0.1:${AVIA_TEST_OBJECT_STORE_PORT}"
export AVIA_TEST_OBJECT_STORE_ACCESS_KEY="${root_user}"
export AVIA_TEST_OBJECT_STORE_SECRET_KEY="${root_password}"
export AVIA_TEST_OBJECT_STORE_API_ACCESS_KEY="${api_access}"
export AVIA_TEST_OBJECT_STORE_API_SECRET_KEY="${api_secret}"
export AVIA_TEST_OBJECT_STORE_WORKER_ACCESS_KEY="${worker_access}"
export AVIA_TEST_OBJECT_STORE_WORKER_SECRET_KEY="${worker_secret}"
export AVIA_TEST_CLAMAV_ADDRESS="127.0.0.1:${AVIA_TEST_CLAMAV_PORT}"
export GOCACHE="${GOCACHE:-/private/tmp/avia-aws-pilot-go-cache}"
export GOTMPDIR="${runtime_directory}/go-tmp"
mkdir -p "${GOTMPDIR}"

go -C "${repository_root}/apps/api" test -v -count=1 ./tests/integration \
  -run '^(TestMigrationsApplyFromAnEmptyDatabase|TestExactObjectVersionIdentityRequiresACompleteNonEmptyPair|TestManagedEvidenceScanPersistsExactSourceAndCanonicalVersions|TestManagedEvidenceScanCrashRecoveryDoesNotDuplicateCanonicalVersion|TestMinIOObjectStoreKeepsObjectsPrivateAndHonorsSignedBoundaries|TestMinIOObjectStoreNeverOverwritesUnderConcurrentWritesAndCopies|TestLiveClamAVAdapterScansCleanAndEICARStreams|TestLiveClamAVMinIOEvidencePipeline)$'

docker compose \
  --project-name "${project}" \
  --file "${compose_file}" \
  --file "${network_overlay}" \
  stop clamav
export AVIA_TEST_CLAMAV_FAILURE_MODE=unavailable
go -C "${repository_root}/apps/api" test -v -count=1 ./tests/integration \
  -run '^TestLiveClamAVMinIOPipelineFailsClosedDuringInjectedScannerLoss$'
unset AVIA_TEST_CLAMAV_FAILURE_MODE

export AVIA_TEST_MAILPIT_SMTP_ADDRESS="127.0.0.1:${AVIA_TEST_MAILPIT_SMTP_PORT}"
export AVIA_TEST_MAILPIT_API_URL="http://127.0.0.1:${AVIA_TEST_MAILPIT_HTTP_PORT}"
export AVIA_TEST_SMTP_PASSWORD_FILE="${runtime_directory}/secrets/smtp_password"
export AVIA_TEST_COMPOSE_FILES="${compose_file} ${network_overlay}"
export AVIA_TEST_COMPOSE_PROJECT="${project}"
export AVIA_TEST_MAILPIT_SERVICE=mailpit
go -C "${repository_root}/apps/api" test -v -count=1 ./tests/integration \
  -run '^TestRealMailpitDeliveryFailureRestartAndExactMetadata$'
