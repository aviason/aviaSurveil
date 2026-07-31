#!/usr/bin/env bash
set -euo pipefail

if [[ "${1:-}" != "acceptance" || $# -ne 1 ]]; then
  echo "usage: $0 acceptance" >&2
  exit 64
fi
if [[ -n "${AVIA_DATA_FEED_RECOVERY_EVIDENCE_ROOT:-}" ]]; then
  echo "AVIA_DATA_FEED_RECOVERY_EVIDENCE_ROOT must be unset; acceptance creates a fresh root" >&2
  exit 64
fi
if [[ -z "${AVIA_RECOVERY_PRODUCER_MANIFEST:-}" || ! -r "${AVIA_RECOVERY_PRODUCER_MANIFEST}" ]]; then
  echo "AVIA_RECOVERY_PRODUCER_MANIFEST must name a readable manifest" >&2
  exit 64
fi
if [[ -z "${AVIA_RECOVERY_AVIACORE_MANIFEST:-}" || ! -r "${AVIA_RECOVERY_AVIACORE_MANIFEST}" ]]; then
  echo "AVIA_RECOVERY_AVIACORE_MANIFEST must name a readable manifest" >&2
  exit 64
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
PRODUCER_MANIFEST="$(cd "$(dirname "${AVIA_RECOVERY_PRODUCER_MANIFEST}")" && pwd)/$(basename "${AVIA_RECOVERY_PRODUCER_MANIFEST}")"
AVIACORE_MANIFEST="$(cd "$(dirname "${AVIA_RECOVERY_AVIACORE_MANIFEST}")" && pwd)/$(basename "${AVIA_RECOVERY_AVIACORE_MANIFEST}")"
EVIDENCE_ROOT="$(mktemp -d /private/tmp/aviasurveil360-datafeed-task6.XXXXXX)"
export AVIA_DATA_FEED_RECOVERY_EVIDENCE_ROOT="${EVIDENCE_ROOT}"

cleanup() {
  exit_status=$?
  rm -rf "${EVIDENCE_ROOT}"
  exit "${exit_status}"
}
trap cleanup EXIT

go -C "${REPOSITORY_ROOT}/apps/api" test -race -count=1 ./internal/datafeed ./cmd/data-feed-replay ./cmd/data-feed-backfill ./cmd/data-feed-reconcile >"${EVIDENCE_ROOT}/datafeed_replay.log"
go -C "${REPOSITORY_ROOT}/apps/api" run ./cmd/data-feed-reconcile "${PRODUCER_MANIFEST}" "${AVIACORE_MANIFEST}" >"${EVIDENCE_ROOT}/reconciliation.json"
git -C "${REPOSITORY_ROOT}" diff --check >"${EVIDENCE_ROOT}/diff-check.log"

printf '%s\n' "Task 6 replay/backfill/reconciliation local contract: verified locally"
printf '%s\n' "Manifest frontier reconciliation: verified locally"
printf '%s\n' "Evidence root created and cleaned: ${EVIDENCE_ROOT}"
printf '%s\n' "Artifact status: candidate-only"
printf '%s\n' "production-ready: not established"
