# AGA Hybrid Classification Task 6 Evidence — 2026-08-04

Status: `verified locally` for the Task 6 implementation slice. The product
remains `candidate-only`; PostgreSQL persistence, connected role verification,
and browser execution remain `not run`.

## Implemented

- Added exact server-side recommendation request pin propagation for generation,
  provider scope, typed target, taxonomy/run, Draft, qualifiers, effective
  time, and readiness.
- Added fail-closed recommendation errors for missing and ambiguous server
  facts, and disabled the former placeholder recommendation command path.
- Added deterministic recommendation and snapshot identifiers, recomputed
  recommendation/snapshot digests, full discriminated question-reference
  validation, stable ordering, and append-only memory/PostgreSQL snapshot
  seams.
- Added HTTP neutral no-store handling for missing/ambiguous recommendation
  facts.
- Added the authorized HTTP Planning-only AGA status surface. It exposes no
  recommendation identifiers or raw classification data and keeps ordinary
  demo Planning behavior unchanged.
- Regenerated the OpenAPI bundle and Go/TypeScript transport contracts for the
  recommendation snapshot response.

## Verification

The following Task 6 checks passed locally. Go used the task-owned cache
`/tmp/avia-aga-go-cache-task6` because the default sandbox cache location is
not writable in this environment.

- `GOCACHE=/tmp/avia-aga-go-cache-task6 go -C apps/api test -count=1 ./internal/agaapplicability ./internal/agademoworkspace ./internal/preproddata/agademoworkspace ./internal/httpapi`
- `npm --prefix apps/web test -- src/features/planning/new-audit-wizard.test.tsx` — 15/15
- `npm --prefix apps/web run typecheck`
- `npm --prefix apps/web run build:demo`
- `node apps/web/scripts/assert-aga-workspace-artifact-boundary.mjs --profile demo --artifact apps/web/dist`
- `npm --prefix apps/web run build:http`
- `node apps/web/scripts/assert-aga-workspace-artifact-boundary.mjs --profile http --artifact apps/web/dist`
- `node apps/web/scripts/assert-http-artifact.mjs apps/web/dist`

The focused Go tests include server-derived-fact rejection, kind/profile
mismatch, ambiguous question-leaf rejection, readiness pinning, immutable
snapshot validation, no-write neutral failure, and HTTP no-store behavior.
