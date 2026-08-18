#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
API_ROOT="${REPOSITORY_ROOT}/apps/api"

cd "${API_ROOT}"
go test ./internal/application ./internal/httpapi/... \
  -run 'TestScopeRecommendation_(MultiHistoryGolden|SingleHistoryGolden|ComparableAuditKeyIsolation|ValidatedCleanTruthTable|PrecedenceAndMandatoryFloor|FixedClockBoundaries|AuditeeProjectionExcludesInternalFields)$' \
  -count=1
go test ./tests/integration \
  -run 'TestPriorAuditRecommendation_(GoldenParity|MandatoryFloorDirectAPI|SnapshotFreeze|ReplayImmutability|AuditeePrivacy)$' \
  -count=1

echo "prior-audit-recommendations: deterministic fixtures and policy checks passed"
