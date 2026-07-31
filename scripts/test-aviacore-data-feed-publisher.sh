#!/usr/bin/env bash
set -euo pipefail

if [[ "${1:-}" != "acceptance" || $# -ne 1 ]]; then
  echo "usage: $0 acceptance" >&2
  exit 64
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
if [[ -n "${AVIA_DATA_FEED_EVIDENCE_ROOT:-}" ]]; then
  echo "AVIA_DATA_FEED_EVIDENCE_ROOT must be unset; acceptance creates a fresh root" >&2
  exit 64
fi
EVIDENCE_ROOT="$(mktemp -d /private/tmp/aviasurveil360-data-feed-task5.XXXXXX)"
export AVIA_DATA_FEED_EVIDENCE_ROOT="${EVIDENCE_ROOT}"
export AVIA_TEST_RUNTIME_DIR="${EVIDENCE_ROOT}"
COMPOSE_PROJECT="aviasurveil360-task5-publisher"
export AVIA_TEST_POSTGRES_PORT="$((56000 + RANDOM % 1000))"

cleanup() {
  local status=$?
  trap - EXIT
  set +e
  if [[ "${status}" -ne 0 ]]; then
    for log in "${EVIDENCE_ROOT}"/{go-race,postgres-integration,sqlc,command-contract}.log; do
      if [[ -f "${log}" ]]; then
        echo "Task 5 acceptance failure tail: ${log##*/}" >&2
        tail -n 80 "${log}" >&2
      fi
    done
  fi
  docker compose --project-name "${COMPOSE_PROJECT}" --file "${REPOSITORY_ROOT}/deploy/local/compose.test.yaml" down --volumes --remove-orphans >/dev/null 2>&1 || true
  rm -rf "${EVIDENCE_ROOT}"
  exit "${status}"
}
trap cleanup EXIT

printf '%s\n' "task5_data_feed_acceptance" >"${EVIDENCE_ROOT}/run-kind"
. "${REPOSITORY_ROOT}/scripts/lib/init-local-test-runtime.sh"
initialize_local_test_runtime "${EVIDENCE_ROOT}" "http://127.0.0.1:4174" "${REPOSITORY_ROOT}"
DATABASE_PASSWORD="$(tr -d '\r\n' <"${EVIDENCE_ROOT}/secrets/app_database_password")"
docker compose --project-name "${COMPOSE_PROJECT}" --file "${REPOSITORY_ROOT}/deploy/local/compose.test.yaml" down --volumes --remove-orphans >"${EVIDENCE_ROOT}/preflight-cleanup.log" 2>&1 || true
docker compose --project-name "${COMPOSE_PROJECT}" --file "${REPOSITORY_ROOT}/deploy/local/compose.test.yaml" up --detach --wait postgres >"${EVIDENCE_ROOT}/postgres.log"
for _ in {1..40}; do
  if (echo > "/dev/tcp/127.0.0.1/${AVIA_TEST_POSTGRES_PORT}") 2>/dev/null; then break; fi
  sleep 0.25
done
if ! (echo > "/dev/tcp/127.0.0.1/${AVIA_TEST_POSTGRES_PORT}") 2>/dev/null; then
  echo "Task 5 PostgreSQL host port did not become reachable" >&2
  exit 1
fi
go -C "${REPOSITORY_ROOT}/apps/api" test -race -count=1 ./internal/datafeed ./cmd/data-feed-worker >"${EVIDENCE_ROOT}/go-race.log"
AVIA_TEST_DATABASE_URL="postgres://aviasurveil:${DATABASE_PASSWORD}@127.0.0.1:${AVIA_TEST_POSTGRES_PORT}/aviasurveil?sslmode=disable" go -C "${REPOSITORY_ROOT}/apps/api" test ./tests/integration -run '^(TestDataFeedPublisherMigrationFencesRetryAndQuarantineTransitions|TestDataFeedPublisherConcreteLeaseAndReceiptPersistence)$' -count=1 >"${EVIDENCE_ROOT}/postgres-integration.log"
"${REPOSITORY_ROOT}/scripts/check-sqlc.sh" >"${EVIDENCE_ROOT}/sqlc.log"
node --test "${REPOSITORY_ROOT}/tests/aviacore-data-feed-publisher-command-contract.test.mjs" >"${EVIDENCE_ROOT}/command-contract.log"

if rg -n --glob '*.go' '(slog\.(Info|Warn|Error|Debug).*payload|payload.*slog\.(Info|Warn|Error|Debug)|ClientPrivateKeyFile.*slog|PayloadKeyFile.*slog)' "${REPOSITORY_ROOT}/apps/api/internal/datafeed" >"${EVIDENCE_ROOT}/forbidden-log-scan.log"; then
  echo "data-feed source contains a value or secret log candidate" >&2
  exit 1
fi

printf '%s\n' "Task 5 direct-mTLS data-feed publisher: verified locally"
printf '%s\n' "TLS receiver/receipt/retry/quarantine test package: verified locally"
printf '%s\n' "Task-owned PostgreSQL migration/lease/receipt integration: verified locally"
printf '%s\n' "Evidence root created and cleaned: ${EVIDENCE_ROOT}"
printf '%s\n' "Artifact status: candidate-only"
printf '%s\n' "production-ready: not established"
