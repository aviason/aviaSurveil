#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOCACHE="${GOCACHE:-/private/tmp/avia-aga-intake-go-cache}"
export GOCACHE
if [[ "${1:-}" == "--security-only" ]]; then
  node "${ROOT}/scripts/verify-governed-checklist-test-inventory.mjs" --phase task3
  node --test "${ROOT}/tests/governed-checklist-intake-security.test.mjs" "${ROOT}/tests/local-compose-policy.test.mjs"
  node "${ROOT}/scripts/check-governed-checklist-intake-cleanup.mjs"
  exit 0
fi
node "${ROOT}/scripts/verify-governed-checklist-test-inventory.mjs" --phase task8
go -C "${ROOT}/apps/api" test ./internal/checklistintake ./internal/regulatory ./internal/identity -count=1
npm --prefix "${ROOT}/apps/web" run test -- --run src/backend/governed-checklist-intake-parity.test.ts src/features/admin/checklist-governed-intake-components.test.tsx
