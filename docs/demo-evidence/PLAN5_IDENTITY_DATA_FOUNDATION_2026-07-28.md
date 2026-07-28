# Plan 5 Identity And Data Foundation Evidence

**Evidence date:** 2026-07-28

**Scope status:** Tasks 1–3 complete; Tasks 4–9 `not run`

**Artifact status:** `candidate-only`

**Release status:** `release pending`

## Task 1 Contract

Contract/profile version `1.0.0`, all 11 owner directives, and the 26/26
machine-readable contract gate were published in `d0f5b29`. The Plans 2–4
closure and Task 1 synchronization baseline was published in `b12126d`.

## Task 2 Identity Directory Foundation

Task 2 is `verified locally`.

The complete Task 2 scope is published on `origin/main` as
`26da3c0fb9a3b81bc3a7d6704913cf9fb53ab7b1`.

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

## Task 3 Provisioning, Invitation, And Recovery

Task 3 is `verified locally`. It is awaiting its focused task commit and push;
Task 4 is `not run`.

### Strict RED evidence

Before production implementation,
`go -C apps/api test -count=1 ./internal/administration
./internal/identity -run 'Task3|Passwordless|ExecuteActions'` failed with exit
1 because the approved action constants, mandatory reason/revision fields, and
Keycloak execute-actions, MFA-reset, and forced-logout methods did not exist.

The activation-focused canonical integration build then failed with exit 1
because `ReconcileActivatedMembership` was undefined. No activation
reconciliation implementation preceded that result.

Final acceptance review added
`TestTask3DuplicateProviderSubjectRequiresManualReviewBeforeDelivery`.
`go -C apps/api test -tags canonicaltest -count=1 ./tests/integration -run
'^TestTask3DuplicateProviderSubjectRequiresManualReviewBeforeDelivery$'`
failed with exit 1 because `identity.ErrKeycloakDuplicateSubject` was
undefined. The implementation now classifies that conflict as
`DUPLICATE_SUBJECT`/`MANUAL_REVIEW`, raises one lifecycle alert, and performs
no invitation delivery.

Admin UI completion review then added a focused action-selection assertion.
`npm --prefix apps/web test -- --run
src/features/admin/users-roles-page.test.tsx` failed 1/2 with exit 1 because
the page had no `Lifecycle action` control. The implemented HTTP Admin control
now submits every approved non-provision action with reason and exact
membership revision, and exposes the additional role/transfer fields only
when needed.

### Implemented lifecycle contract

- The exact lifecycle action enum is `PROVISION`, `UPDATE_ROLES`, `SUSPEND`,
  `REACTIVATE`, `DEACTIVATE`, `TRANSFER_ORGANIZATION`,
  `RESEND_INVITATION`, `RESET_PASSWORD`, `RESET_MFA`, and `FORCE_LOGOUT`.
  Every request requires a reason. Provisioning requires revision zero; every
  existing-account action requires the exact current membership revision.
- A stale request terminates as
  `FAILED_PERMANENT`/`STALE_MEMBERSHIP_REVISION` before provider, delivery,
  session, success-audit, or outbox-success effects.
- Passwordless provisioning uses only Keycloak execute-actions email with
  `UPDATE_PASSWORD` and `VERIFY_EMAIL`, a 24-hour lifetime, authenticated
  local Mailpit SMTP, prior-invitation cancellation, and at most three resends
  per 24 hours. The fourth resend is a deterministic permanent failure and
  reaches no provider delivery.
- The approved TOTP policy is optional for all eight roles.
  `CONFIGURE_TOTP` is never a required action; the application stores only
  provider-derived enrollment status.
- Recovery is Admin-assisted, reason-audited, session-invalidating, and issues
  only `UPDATE_PASSWORD` for 15 minutes. MFA reset and forced logout invoke
  the provider and invalidate application session authority without storing
  credentials, action tokens, TOTP secrets, or provider tokens.
- Invitation issue, delivery acceptance, retry failure, terminal failure,
  expiry, resend cancellation, activation consumption, recovery, MFA reset,
  and lifecycle cancellation are append-only facts with terminal-transition
  checks.
- Provisioning reconciles a provider-created user after duplicate-email/lost
  acknowledgement. Duplicate subjects, duplicate email, role/organization
  drift, unknown organizations, expiry, and stale revisions have deterministic
  typed outcomes.
- Provider failures use exponential backoff capped at five attempts and 30
  minutes. Retry exhaustion and provider authority failures enter
  `MANUAL_REVIEW`; permanent failures are never retried; terminal outcomes
  create lifecycle alerts. A fresh operator command after provider correction
  is verified as the reconciliation path. Persisted-job telemetry records the
  worker outcome and ready age.
- Suspension, deactivation, and reactivation are distinct. Deactivation
  revokes sessions, disables provider authority, retains the minimum identity
  and append-only membership tombstone, and requires a new audited
  reactivation decision. Organization transfer is future-effective, updates
  the application profile and provider authority, appends a membership
  revision, and forces logout.
- Request, provider result, subject, exact role/organization binding,
  membership revision, session invalidation, invitation/recovery delivery,
  retry, and terminal failures are audited without secret material.
- The HTTP Admin directory exposes all nine non-provision lifecycle actions.
  Role update selects the new exact role; transfer requires a destination and
  future effective time; every action submits the current membership revision
  and an operator-entered reason.

### Fresh GREEN verification

| Command | Literal result |
|---|---|
| `./scripts/check-sqlc.sh` | exit 0; `sqlc-check: ok` |
| `./scripts/check-contracts.sh` | exit 0; `contracts-check: ok`; 16/16 OpenAPI tests |
| `./scripts/test-preprod-identity-lifecycle.sh invitation` | exit 0; race-enabled Administration, identity, notification, and full canonical integration packages passed; integration 35.130s; task-owned services and storage cleaned |
| `./scripts/test-preprod-identity-lifecycle.sh all-eight-roles` | exit 0; real Keycloak/Mailpit all-eight-role integration passed in 2.834s; task-owned services and storage cleaned |
| `node --test deploy/local/keycloak/realm-contract.test.mjs tests/local-compose-policy.test.mjs` | exit 0; 24/24 passed |
| `npm --prefix apps/web run typecheck` | exit 0 |
| `npm --prefix apps/web test -- --run src/backend/http-backend.test.ts src/features/admin/users-roles-page.test.tsx` | exit 0; 24/24 passed |
| `node --test tests/harness-docs-smoke.test.js` | exit 0; `harness-docs-smoke: ok` |

The initial expanded full-suite run failed on a projection fixture that passed
an interval into migration 19's absolute timestamp field. After that fixture
was corrected, a second run failed because the expanded harness had not wired
its generated MinIO credentials. The harness was corrected to start and
authenticate to task-owned MinIO. Neither failed run is completion evidence;
the final full rerun and the separate all-eight-role rerun above passed and
cleaned up.

### Boundaries

Task 3 does not establish membership/session freshness enforcement, the
preprod loader, any workload-profile feasibility, deployment, external email,
production identity federation, release readiness, or production readiness.
Those remain `not run`.
