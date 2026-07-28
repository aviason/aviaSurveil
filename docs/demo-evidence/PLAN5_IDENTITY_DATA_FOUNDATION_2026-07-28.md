# Plan 5 Identity And Data Foundation Evidence

**Evidence date:** 2026-07-28

**Scope status:** Tasks 1–2 complete; Task 3 and later tasks `not run`

**Artifact status:** `candidate-only`

**Release status:** `release pending`

## Task 1 Contract

Contract/profile version `1.0.0`, all 11 owner directives, and the 26/26
machine-readable contract gate were published in `d0f5b29`. The Plans 2–4
closure and Task 1 synchronization baseline was published in `b12126d`.

## Task 2 Identity Directory Foundation

Task 2 is `verified locally`.

- Normal API, worker, scheduler, and migration dependency/string scans exclude
  `internal/testprofile`, canonical loader/reset code, and canonical-header
  authority. The canonical-test API remains an explicitly tagged artifact.
- Desired membership uses an immutable application membership ID,
  append-only versions, monotonic revisions, requested/effective facts, exact
  role and organization state, provider observations, and one current
  desired/observed synchronization row.
- API and worker use a generated confidential Keycloak service-client secret.
  The imported service account has exactly `query-users`, `view-users`,
  `manage-users`, and `view-realm`; bootstrap-admin material is not mounted in
  either normal application runtime.
- The Admin directory reads bounded Keycloak account pages and returns one row
  per provider subject with real provider email/enabled/MFA/required-action
  state, `roles[]`, organization, application profile, desired membership,
  drift, derived invitation state, and last successful application session.
- Pagination is capped at 25 accounts and 26 provider calls, uses an opaque
  filter-bound cursor and traversal consistency token, and supports search,
  role, organization, and account-status filters.
- A provider account without an application session is visible. Keycloak
  unavailability returns HTTP 503 `PROVIDER_UNAVAILABLE`; it is not converted
  into an empty or stale-success page.
- Raw-wire checks deny Auditee access and exclude password, TOTP-secret,
  provider-token, and Internal CAA Note fields.

## Strict RED Evidence

1. `./scripts/test-normal-artifact-boundary.sh` failed with exit 1 because the
   normal API transitively linked `internal/testprofile`.
2. The focused Keycloak test failed with exit 1 because service-client
   configuration and `ListDirectory` did not exist.
3. Platform configuration failed to compile and the realm/Compose suite failed
   5 tests because the lifecycle client, generated secret, exact roles, and
   secret-file wiring did not exist.
4. Final staged review added an update/delete mutation assertion and the
   focused lifecycle-worker integration test failed with exit 1 because
   desired-membership history accepted an in-place rewrite. Migration 18 now
   applies the immutable-row trigger; the focused rerun passed with exit 0.
5. After correcting an invalid multi-statement test setup, the focused
   Administration projection test failed with exit 1 because a seeded
   delivered invitation returned `required-actions-complete`. The checked
   SQLC projection now joins the latest lifecycle-bound email-delivery fact;
   the focused rerun passed with exit 0 and returned `delivered`.

No corresponding Task 2 production implementation preceded those RED results.

## Fresh GREEN Verification

| Command | Literal result |
|---|---|
| `./scripts/check-contracts.sh` | exit 0; `contracts-check: ok`; OpenAPI tests 16/16 |
| `./scripts/test-normal-artifact-boundary.sh` | exit 0; `normal-artifact-boundary: ok` |
| `go -C apps/api test -tags canonicaltest -race -p 1 -count=1 ./internal/identity ./internal/administration ./internal/httpapi ./tests/integration` | exit 0; all four packages passed |
| `npm --prefix apps/web test -- --run src/backend/http-backend.test.ts` | exit 0; 22/22 passed |
| `./scripts/check-sqlc.sh` | exit 0; `sqlc-check: ok` |
| `node --test deploy/local/keycloak/realm-contract.test.mjs tests/local-compose-policy.test.mjs` | exit 0; 24/24 passed |
| `npm --prefix apps/web run typecheck` | exit 0 |

The final race run used task-owned local PostgreSQL and MinIO containers. An
earlier attempt exposed the stale untagged build-graph assertion and absent
MinIO dependency; the assertion was corrected for the explicit
`canonicaltest` artifact and the final complete run passed.

## Boundaries

Task 2 does not complete invitation/recovery delivery, deactivation, transfer,
MFA reset, forced logout, expected-membership-revision command guards,
authority/session freshness enforcement, the preprod loader, any workload
profile, deployment, or production readiness. Those remain `not run`.
