# Governed Service Provider Checklist — Local Evidence

Status: `candidate-only`; `release pending`; `production-ready: not established`.

## Scope boundary

The positive approval/publication path uses only the explicit internal
`SYNTHETIC-OPS-AOC` test profile. It proves local workflow mechanics, not an
official checklist library, legal conclusion, source-owner confirmation, or
real OPS/AOC approval. The exact real OPS/AOC request is `blocked` because the
controlled procedure, Part 140 authority/supersession, Part 127 applicability,
and ownership decisions are unresolved. Its blocked validation has zero
generation-run, candidate, decision, publication, checklist-version, and Audit
effects.

## Task 9 evidence

- `node scripts/verify-governed-checklist-test-inventory.mjs` passed with all
  24 required artifacts present and nonzero named suite counts.
- OpenAPI/generated transport checks passed: `./scripts/check-contracts.sh`
  reported 17/17 tests; the focused OpenAPI command reported 17/17.
- The Task 9 canonical PostgreSQL proof publishes only the synthetic candidate,
  denies Auditee queue/published-version/source-lineage access, then verifies
  the blocked real request leaves the persisted lifecycle counts unchanged.
- Mock Playwright passed at 1440×900 and 390×844. It checks the blocked-real
  boundary first, then synthetic Admin import/submission and the separate
  manager technical-approval/publication commands, with console and document
  overflow failures treated as test failures.
- The disposable local HTTP profile repeats the Task 9 canonical proof and the
  same browser flow against PostgreSQL, the then-current local OIDC provider, and MinIO; current task-owned
  containers, Vite, Playwright, Chrome, and runtime directory were removed.
- Migration recovery evidence remains in the version-20-to-21 integration
  suite: pre-data rollback is allowed, post-data rollback is refused, and
  forward repair preserves governed source/candidate/review/publication/Audit
  history.

## Remaining dependency

Source-owner and responsible Department Manager confirmation of the controlled
NCAA Operations procedure, current Part 140 authority/supersession, and exact
Part 127 applicability is `blocked`. No local check changes that dependency.
