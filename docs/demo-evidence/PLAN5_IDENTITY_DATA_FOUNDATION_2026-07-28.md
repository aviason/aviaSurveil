# Plan 5 Identity And Data Foundation Evidence

**Evidence date:** 2026-07-28

**Scope status:** Tasks 1–5 complete; Tasks 6–9 `not run`

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

Task 3 is `verified locally` and published on `origin/main` as
`8cf2b57fd487e4ba1b2439717425344bb06ea7e3`. Task 4 is complete and `verified
locally`; its complete focused scope is published on `origin/main` as
`bedff4575704511d107f148c289424b46485d0b2`.

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

## Task 4 Role, Organization, MFA, And Session Lifecycle

Task 4 is complete and `verified locally`. The Task 3 publication gate was
closed at `8cf2b57fd487e4ba1b2439717425344bb06ea7e3` before Task 4 started.

### Strict RED evidence

Before Task 4 production implementation, `go -C apps/api test -tags
canonicaltest -count=1 ./internal/identity ./tests/integration -run 'Task4'`
failed with exit 1. The compiler reported that
`identity.ValidateApplicationAuthority`, `identity.AuthorityObservation`,
`identity.AuthorityObserver`, and the session-manager `AuthorityObserver`
dependency were undefined.

The focused provider-observation RED, `go -C apps/api test -count=1
./internal/identity -run 'Task4'`, then failed with exit 1 because
`KeycloakAdminClient.ObserveUserAuthority` did not exist.

The frozen 30/60/120-second boundary test then ran through the isolated
canonical harness. It failed because authentication at 31 seconds had made
only the new-login provider call rather than the required heartbeat. The
harness removed every task-owned container, volume, and network after the
failure.

The first-login activation test then failed to compile because
`session.ActivationReconciler` and the matching session-manager dependency did
not exist. No callback/session activation wiring preceded this result.

The callback fail-closed RED, `go -C apps/api test -count=1
./internal/httpapi -run
'^TestTask4OIDCCallbackRejectsStaleAuthorityAndExpiresCookies$'`, failed with
exit 1. The stale-authority session rejection surfaced as HTTP 500
`SESSION_CREATE_FAILED` and did not expire browser cookies; the required
boundary is HTTP 401 `STALE_AUTHORITY` with both cookies expired.

The mutation-freshness RED, `go -C apps/api test -tags canonicaltest -run
'^TestTask4AuthorityMutationForcesFreshProviderObservation$'
./tests/integration`, failed to compile because
`session.RequireFreshAuthorityObservation` did not exist. This proved the
normal read heartbeat had no separate mechanism for the frozen
fresh-observation-before-authority-mutation boundary.

The first updated `./scripts/test-http-oidc-profile.sh` acceptance run failed
before API/browser startup. The authoritative lifecycle request rejected the
isolated one-shot actor because its retained identity reference did not yet
exist, proving the harness could not silently invent a requester that violated
the lifecycle foreign key. Cleanup removed the task-owned containers,
volumes, network, runtime directory, and generated secrets.

The second OIDC acceptance run passed the authoritative Admin setup and then
failed readiness before Playwright. The stale lane raced API and worker bucket
initialization and selected unavailable ClamAV for the isolated auth test.
Cleanup again removed all task-owned runtime state.

The third OIDC run reached Playwright with healthy API and worker processes,
then failed before browser launch because the repository-pinned Chromium 1228
binary was not installed. The exact pinned Chromium and headless-shell
artifacts were subsequently installed; the browser gate was not skipped or
weakened.

The fourth OIDC run reached real Keycloak TOTP enrollment and returned to the
app, then failed because the first navigation-time session poll raised a
transient browser `Failed to fetch` exception instead of retrying. Persistent
HTTP failures remain fail-closed; cleanup removed the browser and isolated
stack.

The fifth OIDC run confirmed that in-page `fetch` remained unavailable after
the callback even though the page stayed on the app origin. The assertion was
moved to Playwright's browser-context request client, which shares the same
cookies and reports the exact `/auth/session` status without depending on page
JavaScript execution.

The sixth OIDC run then reported the exact HTTP outcome: the callback returned
to the app but `/auth/session` remained HTTP 401 for the full polling window.
A secret-free in-harness diagnostic was added to compare desired membership,
sync, retained identity/profile, and live provider authority before cleanup.

That diagnostic reported ACTIVE membership revision 2, exact/current sync,
matching retained issuer and profile, and live enabled/unlocked Admin/CAA
authority with TOTP enrolled and no required actions, but zero application
sessions. The remaining failure is after activation in callback session
creation; the browser test now captures the callback status/problem directly.

The eighth OIDC run captured HTTP 401 `STALE_AUTHORITY` directly. Combined
with the exact diagnostic, this isolated the defect to the pre-activation
session clock: activation became effective milliseconds after
`Manager.Create` captured `now`, so its post-activation transaction falsely
treated the new revision as future-effective. GREEN refreshes the clock after
successful activation without relaxing any authority comparison.

The next live run proved that the session was persisted with exact active
authority but that Chromium rejected production `Secure` `__Host-` cookies on
plain `127.0.0.1`. The isolated application origin was corrected to
`http://localhost:4174`; production cookie attributes were not weakened.

Later live runs exposed two stale harness inputs: lifecycle probes omitted the
mandatory reason, and the reviewed realm did not register the approved
`UPDATE_PASSWORD` and `VERIFY_EMAIL` execute-action providers. The focused
realm contract failed 1/3 before the two providers were added. The final
browser path consumes the real Mailpit invitation link, completes required
actions, verifies provider email/required-action state, enrolls TOTP for the
Admin, rejects a wrong OTP, accepts the correct OTP, proves a new Inspector can
log in without TOTP, restarts Keycloak, deactivates the exact Admin through the
revisioned lifecycle API, and observes both provider and application session
invalidation.

Final diff review added a concurrent-revision regression. The isolated race
harness failed with exit 1 because creation returned a real session ID with no
error after the synchronization row changed away from the membership revision
already validated by the request. The final implementation requires exactly
one matching current-revision synchronization row to be refreshed before
session insertion; otherwise it returns `ErrUnauthenticated` and writes no
stale session.

The same review recorded a focused exact-organization RED: padded `" CAA "`
was normalized and accepted. The authority validator now rejects leading or
trailing whitespace instead of silently rewriting organization identity; the
focused rerun passed.

### Implemented authority boundary

- Every session stores the exact desired-membership revision, current provider
  observation reference/state, token role/organization claims, and provider
  session identity.
- New and existing sessions fail closed unless desired membership, live
  provider authority, token claims, role, organization, and stored revision
  agree exactly.
- Provider observation follows the approved 30-second heartbeat, 60-second
  maximum age, and 120-second provider-loss denial deadline. Drift,
  disablement, lockout, required actions, or exact authority mismatch deny
  immediately; reconciliation restores authority only after fresh exact
  agreement.
- Role change, organization transfer, suspend, deactivate, MFA reset, and
  forced logout revoke application sessions. Authority mutations require a
  fresh provider observation, and old sessions never revive.
- First login reconciles an invited membership to active before creating the
  current-revision session. A callback stale-authority failure is HTTP 401
  `STALE_AUTHORITY` and expires both browser cookies.
- Bootstrap and break-glass provider identities have no application
  membership and cannot create an application session. The realm imports no
  standing bootstrap/break-glass administrator and keeps login events,
  detailed Admin events, and the `jboss-logging` listener enabled. Actual
  break-glass use remains `not run` and blocked unless its external two-person,
  15-minute, alarm, incident, audit, rotation, and session-closure gate is
  separately satisfied.
- Remote OIDC coverage proves expired-token rejection, cached JWKS refresh on
  signing-key rotation, acceptance inside the verifier's five-minute `nbf`
  tolerance, and rejection beyond it.

### Fresh GREEN verification

| Command | Literal result |
|---|---|
| `./scripts/check-sqlc.sh` | exit 0; `sqlc-check: ok` |
| `go -C apps/api test -tags canonicaltest -race -p 1 -count=1 ./internal/platform/session ./internal/identity ./internal/administration ./internal/httpapi ./tests/integration` | exit 0 inside the isolated session-authority harness; all five packages passed; integration completed in 37.017 seconds |
| `./scripts/test-http-oidc-profile.sh` | exit 0; Playwright 1/1 in 46.7 seconds; `OIDC runtime secret/log scan: zero generated-secret matches`; cleanup complete |
| `./scripts/test-preprod-identity-lifecycle.sh session-authority` | exit 0; `Plan 5 Task 4 session authority: verified locally`; cleanup complete |
| `go -C apps/api test -count=1 ./internal/identity -run 'TestTask4RemoteOIDCProviderRefreshesRotatedSigningKeysAndEnforcesClockBoundaries'` | exit 0 |
| `node --test deploy/local/keycloak/realm-contract.test.mjs tests/preprod-identity-data-contract.test.mjs` | exit 0; 29/29 passed |

A standalone tagged race invocation after the verification-command correction
compiled every package but failed because no PostgreSQL or MinIO fixtures were
listening at the documented ports. The authoritative final
`session-authority` harness provisioned those exact dependencies and ran the
same race command successfully. Both final isolated stacks removed containers,
volumes, networks, runtime directories, generated secrets, API/worker/Vite
processes, and browser processes.

### Non-claims and next boundary

Task 4 did not finish the Users and Roles Admin experience. Task 5 now closes
that boundary. The preprod loader, every workload profile, deployment,
production identity, and production readiness remain `not run` in Tasks 6–9.
The artifact is still `candidate-only` and `release pending`.

## Task 5 Users And Roles Admin Experience

### Strict RED evidence

Before Task 5 production changes, `npm --prefix apps/web test -- --run
src/features/admin/users-roles-page.test.tsx` failed with exit 1: 1 of 3 tests
failed. The complete backend-shaped retained account rendered, but the first
independently derived required fact, `Provider account`, was absent. This
proved the page did not yet expose the complete provider, application,
membership, authority-alignment, and session record or the exact per-action
availability reasons. No Task 5 production implementation preceded this RED.

The expanded pre-implementation focused run then failed 5 of 6 tests. Its
independent first failures were the missing `Review provisioning` control, the
same complete-fact contract, the missing reasoned confirmation dialog, the
missing loading status, and the missing confirmation dialog on the retryable
lifecycle path. The sole passing test was the retained CAA role/organization
client-side rejection. These failures also establish RED coverage for every
server-supported lifecycle action, focus return, loading/empty/unavailable/
stale/retry directory states, lifecycle retry reconciliation, and focused
command-error presentation before production implementation.

The first isolated canonical HTTP Playwright run provisioned the real
Keycloak account and completed the first lifecycle request, then failed 0/1
after the role-update worker completed. Repeated refresh clicks had issued
overlapping reads, so an older `PENDING` response overwrote a later
`SUCCEEDED` response. The browser assertion received the exact candidate-002
status regression. API and worker logs contained no secrets, both lifecycle
work batches completed, and the harness removed every task-owned container,
volume, network, runtime directory, Vite process, and browser process.

The rerun passed monotonic lifecycle reconciliation and then failed because
the post-role-change directory entry no longer contained the exact provider
email/display name. The captured UI still proved `manager`, `CAA`, `invited`,
revision 2, and `in-sync`, isolating the regression to erased provider
identity facts. The focused Keycloak client RED then failed with exit 1: the
authority update sent only `organization_id`, omitted username, email, first
name, last name, email-verification state, and another retained attribute, and
never read the current user representation. This reproduces the browser
evidence without changing the accepted provider contract.

After the preservation fix, the next HTTP run completed provisioning and role
replacement with the provider identity intact, then failed only in the new
responsive helper: it measured an off-screen control before scrolling and
received the correct negative Y coordinate. The helper now follows the
established repository geometry contract and scrolls each target into view
before asserting viewport bounds.

The first complete Web regression run then passed 649 of 650 tests and failed
only the stylesheet ownership contract because the mobile media block repeated
the exact `.admin-lifecycle-dialog` selector. The mobile overrides are now
scoped through the Administration page root so each selector has one owner;
no threshold, mask, or accepted visual baseline changed.

### Implemented Admin experience

- The HTTP Admin directory renders exact provider, application profile,
  desired membership, role, organization, invitation, MFA, account,
  synchronization, and last-session facts without placeholders.
- Provisioning and all nine non-provision lifecycle actions are visible and
  use a reasoned confirmation dialog. Role replacement and organization
  transfer collect their exact additional inputs; every existing-account
  request uses the current membership revision.
- Each action is either operable or disabled with a specific provider,
  authority, state, or input reason. Loading, empty, unavailable, pending,
  succeeded, failed, stale, and retry states are stable and test-covered.
- Directory refresh is monotonic so an older response cannot replace a newer
  terminal result. A Keycloak authority update reads and preserves the current
  username, email, display name, verification state, and retained attributes
  before changing only server-owned organization and role authority.
- Keyboard dialog operation, Escape, focus placement and restoration, error
  summary focus, desktop/tablet/mobile geometry, and no-overlap behavior are
  verified in focused component and real HTTP browser coverage.

### Fresh GREEN verification

| Command | Literal result |
|---|---|
| `npm --prefix apps/web run typecheck` | exit 0 |
| `npm --prefix apps/web test -- --run src/features/admin/users-roles-page.test.tsx` | exit 0; 6/6 passed |
| `npm --prefix apps/web run test:e2e:http -- --grep "user lifecycle"` | exit 0; isolated canonical HTTP Playwright 1/1 in 5.4 seconds; focused outbox drain passed; task-owned services, storage, Vite, and browser processes cleaned |
| `go -C apps/api test -race -count=1 ./internal/identity` | exit 0; unrestricted package rerun passed in 1.536 seconds; the preceding restricted-sandbox attempt could not bind a loopback test listener and exited before assertions |
| `go -C apps/api test -count=1 ./internal/identity -run '^TestTask5UpdateUserAuthorityPreservesProviderIdentityAndAttributes$'` | exit 0 |
| `npm --prefix apps/web test -- --run` | exit 0; 650/650 passed across 64 files in 80.13 seconds |
| `node --test tests/preprod-identity-data-contract.test.mjs` | exit 0; 26/26 passed |

### Non-claims and next boundary

Task 5 completes only the local candidate Users and Roles Admin experience.
The loader, profile generation, generated datasets, workload qualification,
deployment, external email, production identity federation, release, and
production readiness remain `not run` in Tasks 6–9. The artifact remains
`candidate-only` and `release pending`.
