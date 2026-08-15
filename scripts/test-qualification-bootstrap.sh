#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
EVIDENCE_ROOT="$(mktemp -d /private/tmp/aviasurveil360-qualification-bootstrap.XXXXXX)"
COMPOSE_PROJECT="aviasurveil360-qualification-$$"
export AVIA_TEST_RUNTIME_DIR="${EVIDENCE_ROOT}"
export AVIA_TEST_POSTGRES_PORT="$((55432 + ($$ % 1000)))"

cleanup() {
	local status=$?
	trap - EXIT
	set +e
	docker compose --project-name "${COMPOSE_PROJECT}" --file "${REPOSITORY_ROOT}/deploy/local/compose.test.yaml" down --volumes --remove-orphans >/dev/null 2>&1
	if [[ "${status}" -ne 0 && -f "${EVIDENCE_ROOT}/integration.log" ]]; then
		tail -n 100 "${EVIDENCE_ROOT}/integration.log" >&2
	fi
	rm -rf "${EVIDENCE_ROOT}"
	exit "${status}"
}
trap cleanup EXIT

umask 077
. "${REPOSITORY_ROOT}/scripts/lib/init-local-test-runtime.sh"
initialize_local_test_runtime "${EVIDENCE_ROOT}" "http://127.0.0.1:4174" "${REPOSITORY_ROOT}"

docker compose --project-name "${COMPOSE_PROJECT}" --file "${REPOSITORY_ROOT}/deploy/local/compose.test.yaml" down --volumes --remove-orphans >/dev/null 2>&1 || true
docker compose --project-name "${COMPOSE_PROJECT}" --file "${REPOSITORY_ROOT}/deploy/local/compose.test.yaml" up --detach --wait postgres >"${EVIDENCE_ROOT}/postgres.log"

for _ in {1..40}; do
	if (echo >"/dev/tcp/127.0.0.1/${AVIA_TEST_POSTGRES_PORT}") 2>/dev/null; then
		break
	fi
	sleep 0.25
done
if ! (echo >"/dev/tcp/127.0.0.1/${AVIA_TEST_POSTGRES_PORT}") 2>/dev/null; then
	echo "qualification bootstrap PostgreSQL host port did not become reachable" >&2
	exit 1
fi

database_password="$(LC_ALL=C tr -d '\r\n' <"${EVIDENCE_ROOT}/secrets/app_database_password")"
AVIA_TEST_DATABASE_URL="postgres://aviasurveil:${database_password}@127.0.0.1:${AVIA_TEST_POSTGRES_PORT}/aviasurveil?sslmode=disable" \
	GOTOOLCHAIN=local GOCACHE="${GOCACHE:-/private/tmp/avia-surveil-go-cache}" GOMODCACHE="${GOMODCACHE:-/private/tmp/avia-surveil-go-modcache}" \
	go -C "${REPOSITORY_ROOT}/apps/api" test ./tests/integration -run '^TestQualificationBootstrapReplayDriftAndPermissionBoundary$' -count=1 -v >"${EVIDENCE_ROOT}/integration.log"

printf '%s\n' "qualification-bootstrap: verified locally"
