# AGA Hybrid Classification Task 7 Evidence — 2026-08-04

Status: `verified locally` for the Task 7 synthetic lifecycle backend slice.
The product remains `candidate-only`, `release pending`, and
`production-ready: not established`. Connected PostgreSQL/grant/zero-delta,
role, and browser execution remain `not run` and are reserved for Task 9.

## Implemented scope

- Added the append-only synthetic inspection aggregate and complete
  Inspection → response → Potential Finding → Finding → CAP → Evidence →
  verification → authorized-closure state machine.
- Enforced exact current recommendation, generation, readiness, organization,
  provider-scope, Inspector/Lead/Auditee binding, compare-and-swap revision and
  digest, idempotency, required-comment, answer, CAP/Evidence/due-date, and
  role-operation boundaries.
- Kept CAP acceptance separate from Finding closure and kept Evidence review,
  Evidence verification, and authorized closure as separate immutable events.
- Added structurally distinct public, CAA, and organization-scoped Auditee
  projections with redacted internal notes, actor identities, and role history.
- Added the append-only PostgreSQL command-store seam and closed lifecycle
  OpenAPI/Go/TypeScript contracts. No real database was accessed or modified.

## Verification

The exact Task 7 focused lifecycle command passed:

```text
GOCACHE=/tmp/avia-aga-go-cache-task7 go -C apps/api test -count=1 ./internal/agademoworkspace -run 'Test(LifecycleRequiresPotentialFindingConversion|InspectionRequiresExactCurrentRecommendation|FindingInitialStateCoversEveryCAPEvidenceChoice|DueDateChoiceIsIndependentAndExact|ReopenAndInspectionCompletionTransitionsAreTotal|CAPAndEvidenceResubmissionTransitionsAreTotal|CAPAcceptanceLeavesFindingOpen|EvidenceReviewOutcomeMappingIsAtomic|EvidenceVerificationAndAuthorizedClosureAreSeparate|AuditeeProjectionIsOrganizationScoped)$'
ok github.com/MarlonJD/aviaSurveil360/apps/api/internal/agademoworkspace
```

The specified package and contract checks also passed:

```text
GOCACHE=/tmp/avia-aga-go-cache-task7 go -C apps/api test -count=1 ./internal/agademoworkspace ./internal/preproddata/agademoworkspace ./internal/httpapi
ok /internal/agademoworkspace
ok /internal/preproddata/agademoworkspace
ok /internal/httpapi

./scripts/generate-contracts.sh
openapi-bundle: wrote api/openapi/aviasurveil360.yaml
generated apps/api/internal/httpapi/generated/api.gen.go

npm --prefix apps/web run contracts:check
contracts-check: ok

node --test api/openapi/tests/aga-demo-workspace-contract.test.mjs
3 tests passed, 0 failed
```

The contract check also ran the repository's 16-test OpenAPI regression suite;
all 16 passed. The generated artifacts are the checked-in bundled OpenAPI,
Go transport, and TypeScript transport outputs.

## Boundary

This evidence is service-double/unit/static evidence only. Canonical
Audit/Finding/CAP/Evidence zero-delta and connected browser behavior are
`not run` until Task 9. No commit, push, deployment, branch change, external
system access, or real database write occurred.
