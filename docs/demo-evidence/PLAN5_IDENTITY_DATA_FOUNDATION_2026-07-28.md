# Plan 5 Identity And Data Foundation Evidence

**Evidence date:** 2026-07-28

**Scope status:** Tasks 1–8 complete, `verified locally`, published, and
remotely confirmed; Task 9 `not run`

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

Task 5 is `verified locally` and published on `origin/main` as
`ac5f78619d033ab7d9ed5c63e3de10fdc337ea38`. The exact remote ref was
confirmed before Task 6.

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

## Task 6 Out-Of-Process Preprod Data Loader

Task 6 is complete and `verified locally`. Task 5 and its exact publication
record are remotely confirmed. The complete Task 6 scope is published on
`origin/main` as `26c7022ebed02c99c7183850f7138ea63580bf0c`
(`feat(data): add isolated preprod loader`), and `git ls-remote` returned that
exact revision for `refs/heads/main`.

### Strict RED evidence

Before Task 6 production implementation, `node --test
tests/preprod-data-boundary.test.mjs` failed 3/3 with exit 1. The first two
tests could not find `scripts/load-preprod-data.sh` or
`apps/api/internal/preproddata/loader.go`; the third proved the
normal-artifact guard did not yet positively identify the separate
`preprod-data-loader` command.

The task-owned-cache Go RED, `env
GOCACHE=/private/tmp/aviasurveil360-task6-red-go-cache go -C apps/api test
-count=1 ./internal/preproddata ./cmd/preprod-data-loader`, failed with exit 1.
Both packages had no non-test implementation files and the command test could
not resolve `loadRunConfiguration`. A preceding sandboxed attempt failed only
because the default Go cache was outside the writable sandbox and is not
behavioral evidence. No Task 6 production implementation preceded the
behavioral failures.

Acceptance review added the missing complete disposable-namespace contract.
The next `node --test tests/preprod-data-boundary.test.mjs` run failed 1/4
with exit 1 because `deploy/local/minio/preprod-init.sh` and the isolated
PostgreSQL, migration, Keycloak, Mailpit, and object-store service topology
did not exist. This RED preceded implementation of those surfaces.

The focused exact-replay RED, `env
GOCACHE=/private/tmp/aviasurveil360-task6-replay-red-go-cache go -C apps/api
test -count=1 ./internal/preproddata -run
'^TestRunnerPersistsIntentBeforeAuthoritativeCommands$'`, failed with exit 1
because a completed-run replay attempted to consume the already-used
authorization. This RED preceded the immutable successful-result binding and
no-target-touch replay implementation.

The focused reconciliation RED, `env
GOCACHE=/private/tmp/aviasurveil360-task6-reconcile-red-go-cache go -C
apps/api test -count=1 ./internal/preproddata -run
'^TestRunnerRejectsReconciliationCountsOutsideIntent$'`, failed with exit 1
because the runner accepted one actual organization against the immutable
intent's expected three and published `SUCCEEDED`. This RED preceded exact
count reconciliation.

The focused profile-mutation RED, `env
GOCACHE=/private/tmp/aviasurveil360-task6-profile-red-go-cache go -C apps/api
test -count=1 ./internal/preproddata -run
'^TestGeneratorRejectsUnversionedMutationOfFrozenProfile$'`, failed with exit
1 because the generator accepted a changed `smoke@1.0.0`
count/distribution without a new version. This RED preceded exact frozen
catalog equality enforcement.

The focused cleanup-linkage RED, `env
GOCACHE=/private/tmp/aviasurveil360-task6-cleanup-red-go-cache go -C apps/api
test -count=1 ./internal/preproddata -run
'^TestIntentAuthorizationAndControlRecordsAreImmutableAndBound$'`, failed with
exit 1 because a consumed `LOAD_EMPTY_TARGET` authorization was incorrectly
accepted as cleanup authority. This RED preceded exact successful-result,
target-fingerprint, and `DROP_RECREATE_TARGET` authorization validation.

The focused result-store RED used the same test and failed with exit 1 because
an incomplete `SUCCEEDED` result could be appended directly and bound to the
run without matching every intent count and relationship-digest family. This
RED preceded store-level reconciliation.

The bounded-streaming boundary RED, `node --test
tests/preprod-data-boundary.test.mjs`, then failed 1/4 with exit 1 because
`RunInput` retained every authoritative command in a slice and no
`CommandStream.Next` iterator existed. This RED preceded the constant-memory
streaming command contract required by the larger frozen profiles.

The focused checkpoint-bound RED, `env
GOCACHE=/private/tmp/aviasurveil360-task6-checkpoint-red-go-cache go -C
apps/api test -count=1 ./internal/preproddata -run
'^TestRunnerBoundsCheckpointMemoryForStreamingProfiles$'`, failed with exit 1
because 2,049 streamed commands retained 2,050 checkpoint names. This RED
preceded the intent-sized schedule capped at 1,024 command checkpoints.

The namespace overwrite RED, `node --test
tests/preprod-data-boundary.test.mjs`, then failed 1/4 with exit 1 because the
initializer exposed `--rotate` and force-move paths outside the
operation-authorization boundary. This RED preceded create-only namespace
initialization; later whole-namespace disposal/recreation remains subject to
the exact authorized cleanup path.

The focused boundary-redaction RED, `env
GOCACHE=/private/tmp/aviasurveil360-task6-redaction-red-go-cache go -C
apps/api test -count=1 ./internal/preproddata -run
'^TestRunnerRedactsBoundaryFailureFromResultAndError$'`, failed with exit 1
because a sentinel returned by the server-owned command boundary leaked into
the persisted failure result and returned error. This RED preceded safe public
failure classification.

The focused command-payload RED, `env
GOCACHE=/private/tmp/aviasurveil360-task6-command-red-go-cache go -C apps/api
test -count=1 ./internal/preproddata -run
'^TestAuthoritativeCommandValidationRejectsUnsafePayloads$'`, failed all four
unsafe cases: malformed JSON, nested `client_secret`, a hyphenated
access-token key, and a payload above 1 MiB. This RED preceded bounded
structural payload validation.

The focused resume RED, `env
GOCACHE=/private/tmp/aviasurveil360-task6-resume-red-go-cache go -C apps/api
test -count=1 ./internal/preproddata -run
'^TestRunnerResumesAfterLastDurableCheckpoint$'`, failed with exit 1 and
`append-only control record conflicts` because `RESUME_RUN` restarted
checkpoint sequence 1 and replayed from the beginning. This RED preceded
durable checkpoint discovery and resumable-stream positioning.

The isolated SMTP RED, `node --test
tests/preprod-data-boundary.test.mjs`, then failed 1/4 with exit 1 because the
generated realm retained normal username `aviasurveil360` while the dedicated
Mailpit auth file required `aviasurveil360-preprod`. This RED preceded exact
SMTP-user parameterization in the realm builder.

The focused control-store-root RED, `env
GOCACHE=/private/tmp/aviasurveil360-task6-store-root-red-go-cache go -C
apps/api test -count=1 ./internal/preproddata -run
'^TestControlStoreRejectsBroadOrNonPrivateRoots$'`, failed with exit 1 because
a group/world-readable directory was accepted for retained manifests and
authorization hashes. This RED preceded broad-root rejection and private-mode
validation.

Final diff review added a strict intent-reader RED. `env
GOCACHE=/private/tmp/aviasurveil360-task6-trailing-red-go-cache go -C apps/api
test -count=1 ./cmd/preprod-data-loader -run
'^TestReadIntentRejectsTrailingContent$'` failed with exit 1 because a second
JSON object after a valid canonical intent was accepted. This RED preceded
strict trailing-content rejection. The focused GREEN passed with exit 0.

### Verification

The first complete Task 6 verification attempt passed the 4/4 Node boundary
suite, HTTP artifact scan, and normal-artifact boundary. The required race run
failed six loader tests because their task-owned temporary control-store roots
did not explicitly apply the new private directory mode. This test-fixture
failure is retained as incomplete evidence and is not represented as passing.

After the fixture correction, the complete required gate was rerun from the
final Task 6 source:

| Command | Literal result |
| --- | --- |
| `node --test tests/preprod-data-boundary.test.mjs` | exit 0; 4/4 passed; 0 failed, skipped, or cancelled |
| `env GOCACHE=/private/tmp/aviasurveil360-task6-final2-go-cache go -C apps/api test -race -p 1 -count=1 ./internal/preproddata ./cmd/preprod-data-loader` | exit 0; `internal/preproddata` passed in 1.576s; `cmd/preprod-data-loader` passed in 1.341s |
| `node apps/web/scripts/assert-http-artifact.mjs apps/web/dist/http` | exit 0; `http-artifact-scan: ok (146 files, 177 inputs)` |
| `./scripts/test-normal-artifact-boundary.sh` | exit 0; focused normal and canonical-test `internal/httpapi` runs passed in 0.553s and 0.479s; `normal-artifact-boundary: ok` |
| `env GOCACHE=/private/tmp/aviasurveil360-task6-final2-vet-go-cache go -C apps/api vet ./internal/preproddata ./cmd/preprod-data-loader` | exit 0 |
| `node --test tests/preprod-identity-data-contract.test.mjs deploy/local/keycloak/realm-contract.test.mjs tests/local-compose-policy.test.mjs` | exit 0; 50/50 passed |
| `docker compose -f deploy/local/compose.yaml --profile full config --services` | exit 0; exactly the 15 reviewed normal services; no preprod service |
| `docker compose -f deploy/local/compose.yaml --profile local-preprod-loader config --services` | exit 0; exactly nine isolated preprod services from PostgreSQL/migration through the one-shot loader |
| `sh -n deploy/local/minio/preprod-init.sh scripts/init-local-preprod-namespace.sh scripts/load-preprod-data.sh scripts/test-normal-artifact-boundary.sh` | exit 0 |

The three new shell entrypoints have mode `0755`. The protected Plan 1 visual
review process remained PID 13055 with its original command. Docker inspection
reported no running containers and no active Compose projects. No deployment,
AWS operation, production change, real PII, or task-owned runtime service was
used.

Task 7 is now complete and `verified locally`; Tasks 8–9 remain `not run`.

## Task 7 Connected Lifecycle Scenarios

Task 7 is complete, `verified locally`, and published as
`a2d474431d8bcc33ebc0557630cd7038cbcd8ec0`; the exact `origin/main` ref is
confirmed. The strict pre-implementation RED was recorded with `env
GOCACHE=/private/tmp/aviasurveil360-task7-red-go-cache go -C apps/api test
-count=1 ./internal/preproddata/scenarios`: exit 1, `no non-test Go files in
.../internal/preproddata/scenarios`. A second strict RED, `env
GOCACHE=/private/tmp/aviasurveil360-task7-store-red-go-cache go -C apps/api
test -tags integration -count=1 ./internal/preproddata/scenarios -run
'^TestPostgresStoreMaterializesTheCompleteSmokeDomainAndReconciles$'`, exited 1
with `undefined: scenarios.NewPostgresStore`, proving the real PostgreSQL
materialization/reconciliation boundary was absent before implementation. A
third strict RED, `env
GOCACHE=/private/tmp/aviasurveil360-task7-connected-boundary-red-go-cache go -C
apps/api test -count=1 ./internal/preproddata/scenarios -run
'^TestConnectedBoundaryMaterializesIdentityAndInvitationStateOutsidePostgres$'`,
exited 1 with the expected missing production types
`scenarios.ProviderAccount`, `scenarios.InvitationDelivery`, and
`scenarios.ObjectVersion`. This proves that the connected external identity,
invitation-delivery, and object boundary did not exist before its Task 7
implementation. The focused object RED, `env
GOCACHE=/private/tmp/aviasurveil360-task7-object-boundary-red-go-cache go -C
apps/api test -count=1 ./internal/preproddata/scenarios -run
'^TestConnectedBoundaryWritesSafeDeterministicObjectVersions$'`, then exited 1
with an empty external `ObjectVersion` instead of the expected synthetic JSON
content, exact bucket/run prefix, SHA-256 digest, and byte count. The stream
metadata RED, `env
GOCACHE=/private/tmp/aviasurveil360-task7-object-stream-red-go-cache go -C
apps/api test -count=1 ./internal/preproddata/scenarios -run
'^TestStreamObjectVersionMetadataMatchesTheSafeSyntheticJSON$'`, exited 1
because the generated digest `sha256:983ed0ba247f4c296c70e1f3cc6e0acea85e1f3f5727952b13d1d592c54a5d37`
did not equal the actual safe JSON digest
`sha256:e987b65b88ba32599eaa230a005e31d952ebd54bac24361ffd22995fb5d6dd2b`.
The external reconciliation RED, `env
GOCACHE=/private/tmp/aviasurveil360-task7-reconcile-drift-red-go-cache go -C
apps/api test -count=1 ./internal/preproddata/scenarios -run
'^TestConnectedBoundaryRejectsExternalIdentityReconciliationDrift$'`, exited 1
with `external identity drift was accepted`, proving that PostgreSQL evidence
alone did not yet reject external provider drift. The durable-record RED, `env
'AVIA_TEST_DATABASE_URL=postgres://aviasurveil:aviasurveil@127.0.0.1:55432/aviasurveil?sslmode=disable'
GOCACHE=/private/tmp/aviasurveil360-task7-store-records-red-go-cache go -C
apps/api test -tags integration -count=1
./internal/preproddata/scenarios -run
'^TestPostgresStoreMaterializesTheCompleteSmokeDomainAndReconciles$'`, exited 1
with `store.Records undefined`, proving the real PostgreSQL store could not yet
supply run-scoped durable records to external reconciliation. The Keycloak
adapter RED, `env
GOCACHE=/private/tmp/aviasurveil360-task7-keycloak-endpoint-red-go-cache go -C
apps/api test -count=1 ./internal/preproddata/scenarios -run
'^TestKeycloakEndpointCreatesExactProviderAccountAndRejectsDrift$'`, exited 1
with missing `scenarios.NewKeycloakEndpoint` and
`scenarios.KeycloakEndpointConfig`, proving there was no deterministic-subject
connected Keycloak adapter or provider-drift reconciliation. The Mailpit RED,
`env GOCACHE=/private/tmp/aviasurveil360-task7-mailpit-endpoint-red-go-cache go
-C apps/api test -count=1 ./internal/preproddata/scenarios -run
'^TestMailpitInvitationEndpointDeliversOnceAndReconcilesRecipient$'`, exited 1
with missing `scenarios.NewMailpitInvitationEndpoint` and
`scenarios.MailpitInvitationEndpointConfig`, proving there was no idempotent
Keycloak execute-actions/Mailpit recipient adapter. The object endpoint RED,
`env GOCACHE=/private/tmp/aviasurveil360-task7-object-endpoint-red-go-cache go
-C apps/api test -count=1 ./internal/preproddata/scenarios -run
'^TestObjectEndpointWritesExactlyOnceAndRejectsContentDrift$'`, exited 1 with
the expected missing connected object endpoint/backend/blob symbols, proving
there was no create-only, replay-safe, drift-reconciling object boundary.
The real MinIO backend RED, `env
GOCACHE=/private/tmp/aviasurveil360-task7-minio-backend-red-go-cache go -C
apps/api test -tags integration -count=1
./internal/preproddata/scenarios -run
'^TestMinIOObjectBackendPersistsAndReconcilesExactScenarioJSON$'`, exited 1
with missing `scenarios.NewMinIOObjectBackend` and
`scenarios.MinIOObjectBackendConfig`, proving the contract had no real
MinIO/S3 adapter. The CLI RED, `env
GOCACHE=/private/tmp/aviasurveil360-task7-cli-red-go-cache go -C apps/api test
-count=1 ./cmd/preprod-data-loader -run
'^TestRunConnectedInvokesTheAuthorizedConnectedRunnerAndPrintsResult$'`, exited
1 with the expected missing dependency-aware runner, `runWithDependencies`,
and canonical route/behavior catalog path fields, proving the one-shot loader
had no connected-run command. The connected-input RED, `env
GOCACHE=/private/tmp/aviasurveil360-task7-connected-inputs-red-go-cache go -C
apps/api test -count=1 ./cmd/preprod-data-loader -run
'^TestLoadConnectedInputsBindsIntentSeedAuthorizationAndCanonicalCatalogs$'`,
exited 1 with `undefined: loadConnectedInputs`, proving no pre-I/O binding
jointly validated the immutable intent, 0600 seed, one-time authorization,
frozen profile, and canonical route/action catalogs. Tasks 8–9 remain
`not run`. The plan-literal aggregate RED,
`./scripts/test-preprod-connected-scenarios.sh smoke`, exited 127 with `no such
file or directory`, proving the whole-namespace load, reconciliation, privacy,
and cleanup harness did not exist before implementation. The privacy-canary
RED, `env
GOCACHE=/private/tmp/aviasurveil360-task7-privacy-canary-red-go-cache go -C
apps/api test -count=1 ./internal/preproddata/scenarios -run
'^TestPrivacyCanariesReferenceExistingCrossOrganizationRecords$'`, exited 1
because the first `list/auditee-a-from-b` canary
`synthetic-privacy-list-0950e5c315fb3356de171255` did not exist, proving the
initial matrix used decorative IDs instead of retained cross-organization
records. The cleanup-command RED, `env
GOCACHE=/private/tmp/aviasurveil360-task7-cleanup-command-red-go-cache go -C
apps/api test -count=1 ./cmd/preprod-data-loader -run
'^(TestRunRecordCleanupInvokesRecorderAndPrintsExactEvidence|TestRecordCleanupRejectsLoadAuthorizationWithoutConsumingIt|TestRecordCleanupConsumesDropAuthorizationAndAppendsAttestation)$'`,
exited 1 with unknown dependency field `recordCleanup` and undefined
`recordCleanupData`. This proves the loader had no offline
cleanup-attestation command and could not yet enforce the separate consumed
`DROP_RECREATE_TARGET` authorization required after whole-namespace cleanup.
The wrapper RED, `node --test --test-name-pattern='cleanup recorder runs
offline' tests/preprod-data-boundary.test.mjs`, exited 1 because
`scripts/load-preprod-data.sh record-cleanup` returned its old usage limited
to `prepare|verify-authorization|run-connected`. This also proved there was no
`--no-deps` execution path preventing accidental restart of the already
cleaned disposable namespace. The first implemented
`./scripts/test-preprod-connected-scenarios.sh smoke` run exited 1 at the real
namespace health gate. Two isolated reproductions captured the root cause
from the MinIO policy compiler: `unsupported condition keys '[s3:prefix]'
used for action 's3:GetBucketLocation'`. Every failed/diagnostic namespace
cleaned all task-owned containers, volumes, and networks; none is completion
evidence. The focused policy RED, `node --test
--test-name-pattern='preprod object policy scopes only ListBucket'
tests/preprod-data-boundary.test.mjs`, exited 1 because the actual action list
was `['s3:GetBucketLocation', 's3:ListBucket']` instead of a separate
unconditional location statement and prefix-scoped list statement. The next
full aggregate passed MinIO, PostgreSQL, Mailpit, and Keycloak health but
exited 1 when `preprod-migration` logged `AVIA_ENVIRONMENT must be
development, test, or production`; exact namespace cleanup passed afterward.
The focused migration-config RED, `node --test
--test-name-pattern='preprod migration uses the supported database-only config
mode' tests/preprod-data-boundary.test.mjs`, exited 1 because the service
carried `AVIA_ENVIRONMENT: local-preprod` rather than the generic migration
binary's supported `development` mode. Its exact disposable database
name/user/host binding remained a separate required assertion. The third full
aggregate passed all service health gates, migration, immutable-intent
preparation, and load-authorization verification, then exited 1 with
`AUTHORITATIVE_COMMAND_FAILED family=providerAccounts`; exact cleanup again
left zero task-owned residue. The provider-authority RED, `env
GOCACHE=/private/tmp/aviasurveil360-task7-provider-authority-red-go-cache go -C
apps/api test -count=1 ./internal/preproddata/scenarios -run
'^TestEveryProviderAccountHasExactRoleOrganizationAuthority$'`, exited 1 and
identified smoke account
`synthetic-provideraccounts-12d03e474037858c42b1d40e` with role `auditee`
and organization `CAA`. This proved profile-distributed role and organization
authority were derived from different source indices.
The fourth aggregate rerun retained that fix but still exited 1 at
`providerAccounts`; exact cleanup again left zero task-owned resources. A
credential-safe isolated real-Keycloak diagnostic returned `201 Created` for
the user create and `404 Not Found` when the client-supplied synthetic `id`
was read back. This proved Keycloak assigns the provider subject and does not
accept the scenario record ID as that subject. The focused provider-subject
RED, `env
GOCACHE=/private/tmp/aviasurveil360-task7-provider-subject-red2-go-cache go -C
apps/api test -count=1 ./internal/preproddata/scenarios -run
'^(TestConnectedBoundaryMaterializesIdentityAndInvitationStateOutsidePostgres|TestKeycloakEndpointCreatesExactProviderAccountAndRejectsDrift|TestMailpitInvitationEndpointDeliversOnceAndReconcilesRecipient)$'`,
exited 1 with missing `ScenarioID`, the old error-only
`EnsureProviderAccount` result, and incompatible identity endpoint signatures.
This proves no boundary captured and durably bound the provider-assigned
subject before PostgreSQL materialization or reused it for invitation and
provider reconciliation.
The first focused implementation rerun then reached invitation delivery and
failed with `invalid synthetic invitation delivery`, proving Mailpit validation
still assumed a provider subject must use the deterministic `synthetic-*`
record-ID shape instead of the bounded subject assigned by Keycloak.
Final acceptance review added a stricter offline-cleanup RED. `node --test
--test-name-pattern='cleanup recorder runs offline'
tests/preprod-data-boundary.test.mjs` exited 1 because the captured first
Docker argument was `compose` rather than `run`. `--no-deps` prevented
dependency restart but Compose still created empty declared networks and
volumes around attestation, so the recorder was not yet a networkless
standalone container that leaves the cleaned namespace untouched.

### Fresh GREEN verification

| Command | Literal result |
|---|---|
| `go -C apps/api test -count=1 ./internal/preproddata/scenarios` | exit 0; `ok .../scenarios 0.630s` in the focused pre-aggregate regression |
| `go -C apps/api test -count=1 ./cmd/preprod-data-loader` | exit 0; `ok .../cmd/preprod-data-loader 0.492s` |
| PostgreSQL integration `TestPostgresStoreMaterializesTheCompleteSmokeDomainAndReconciles` | exit 0; `ok .../scenarios 1.819s` against the task-owned PostgreSQL endpoint |
| MinIO integration `TestMinIOObjectBackendPersistsAndReconcilesExactScenarioJSON` | exit 0; `ok .../scenarios 0.459s` against the task-owned MinIO endpoint |
| `node --test tests/preprod-data-boundary.test.mjs` | exit 0; 7/7 passed, including exact MinIO policy, supported migration config, normal-artifact exclusion, and networkless cleanup recorder |
| focused provider-assigned subject and invitation regression | exit 0; `ok .../scenarios 0.604s` |
| `./scripts/test-preprod-connected-scenarios.sh smoke` | exit 0; `Plan 5 Task 7 connected smoke scenarios: verified locally`; `profile=smoke; families=40; routes=86; actions=306; roles=8`; `privacy=45/45; target cleanup=verified locally; residue=0` |
| `go -C apps/api test -race -count=1 ./internal/preproddata/... ./cmd/preprod-data-loader` | exit 0; four packages passed: core `2.167s`, profiles `1.531s`, scenarios `4.980s`, loader command `2.021s` |
| `go -C apps/api vet ./internal/preproddata/... ./cmd/preprod-data-loader` | exit 0 |
| `./scripts/test-normal-artifact-boundary.sh` | exit 0; `normal-artifact-boundary: ok` |
| `node tests/harness-docs-smoke.test.js` and `node tests/demo-boundary-smoke.test.js` | exit 0; both printed `ok` |
| `bash -n scripts/load-preprod-data.sh scripts/test-preprod-connected-scenarios.sh` | exit 0 |
| `git diff --check` | exit 0 |

The final aggregate rebuilt the isolated one-shot artifacts, started only the
task-owned disposable PostgreSQL, Keycloak/PostgreSQL, Mailpit, and MinIO
namespace, applied migrations, consumed the exact `LOAD_EMPTY_TARGET`
authorization, and published a `SUCCEEDED` result with all 40 expected family
counts and relationship digests. It reconciled the provider-assigned Keycloak
subjects back to deterministic scenario accounts and immutable memberships,
all eight exact realm roles and organizations, required actions and Mailpit
recipients, create-only synthetic MinIO objects, domain row counts, retained
cross-organization records, and 45/45 privacy canaries across 15 surfaces and
three canary classes.

Cleanup then removed the whole disposable namespace, consumed a separately
issued `DROP_RECREATE_TARGET` authorization, and appended the exact
intent/result/target/authorization-bound cleanup attestation from a read-only,
capability-free, `--network none` standalone loader container. Final Compose
container, volume, and network residue was zero. The pre-existing Plan 1
visual-review process remained alive. No AWS, deployment, production, real
PII, or normal HTTP seed surface was used.

Task 7 is therefore `verified locally` and remains `candidate-only`. Task 8
all-profile qualification, release, deployment, AWS, and production readiness
remain `not run`.

Task 7 was committed as `a2d474431d8bcc33ebc0557630cd7038cbcd8ec0`
(`feat(data): add connected preprod scenarios`), pushed to `origin/main`, and
confirmed by both the local upstream ref and `git ls-remote`.

## Task 8 All-Profile Qualification

Task 8 began with a strict pre-implementation scale-contract command,
`node --test tests/preprod-data-scale-contract.test.mjs`, exited 1 with
`Could not find 'tests/preprod-data-scale-contract.test.mjs'`. Its strict
profile-runner command, `./scripts/test-preprod-data-profile.sh smoke`, exited
127 with `no such file or directory`. Neither required Task 8 artifact existed
before these failures.

`smoke@1.0.0` and `acceptance@1.0.0` subsequently passed on the current Task 8
implementation. Their create-only evidence records exact 40-family,
86-route, 306-action, eight-role, relationship, privacy, controlled
interruption/resume, positive resource, API, cleanup, and zero-residue results.
The earlier success and failure directories remain preserved. The final
accepted current-code runs are recorded below.

### Owner-approved qualification revision

During full-volume `realistic@1.0.0` run
`preprod-data/run-task8-realistic-20260728-121351-92512/`, the owner directed
a safe stop and approved `OWNER-DIRECTIVE-2026-07-28-P5T8-01`. The command
exited 130. Its cleanup trap removed every task-owned container, volume, and
network; subsequent Docker residue checks returned no entries, and protected
Plan 1 PID 13055 remained alive. The failed create-only directory and every
earlier run remain retained.

The revision makes `realistic@1.1.0` the required 2×-acceptance local
qualification with a maximum total duration of 900 seconds. It makes
`stress@1.1.0` the required 4×-acceptance local qualification with a maximum
total duration of 1,800 seconds and an exact 512 MiB synthetic object payload.
Both retain all 40 data families, all eight roles, all 86 route dispositions,
all 306 action dispositions, exact relationship and distribution
reconciliation, 45/45 privacy canaries, controlled interruption/resume,
positive resource measurement, API/query evidence, and whole-namespace
cleanup.

The unchanged full-volume `realistic@1.0.0` and `stress@1.0.0` manifests,
including the latter's exact 8 GiB object payload, are separate deferred
release-readiness endurance evidence. Their status is literally `not run`;
they are not local Task 8 completion claims.

The strict revision RED command
`node --test tests/preprod-identity-data-contract.test.mjs
tests/preprod-data-scale-contract.test.mjs` exited 1 with 27/30 passing and
reported missing `1.1.0` qualification metadata, total-duration enforcement,
and owner-decision synchronization. The focused Go RED command failed to
compile because `profiles.ResourceEnvelope.QualificationSeconds` did not
exist and separately rejected `stress@1.1.0` as unknown. No revision
implementation preceded these results. Revised local realistic/stress GREEN
qualification was then executed.

The first `stress@1.1.0` attempt
`preprod-data/run-task8-stress-20260728-125100-11538/` failed with
`TARGET_RECONCILIATION_FAILED`. Cleanup removed the complete disposable
namespace and retained the create-only failed evidence. The focused RED
`TestLocalStressQualificationRetainsAnExactBoundedObjectPayload` then failed
with `first local stress object size = 0, expected 14914`: the exact payload
helper selected only `stress@1.0.0`. The versioned fix retained the exact
8 GiB endurance calculation and enabled the exact 512 MiB local calculation.

### Final accepted profile evidence

| Run | Counts sampled | Generation / total | Resource and API evidence | Payload, privacy, cleanup |
|---|---|---:|---|---|
| `preprod-data/run-task8-smoke-20260728-123820-3753/` (`smoke@1.0.0`) | 3 organizations; 9 accounts; 2 audits; 250 audit events; 24 object versions | 17 / 125 s | 5 samples; 1.637 MiB; 0.132 cores; API p95 107.909 ms | 6,576 bytes; privacy 0; envelope clean; cleanup 3 s; residue 0 |
| `preprod-data/run-task8-acceptance-20260728-124053-5944/` (`acceptance@1.0.0`) | 25 organizations; 250 accounts; 1,000 audits; 100,000 audit events; 9,000 object versions | 124 / 219 s | 58 samples; 25.59 MiB; 0.542 cores; API p95 106.57 ms | 2,466,000 bytes; privacy 0; envelope clean; cleanup 3 s; residue 0 |
| `preprod-data/run-task8-realistic-20260728-124500-8671/` (`realistic@1.1.0`) | 50 organizations; 500 accounts; 2,000 audits; 200,000 audit events; 18,000 object versions | 228 / 326 of 900 s | 111 samples; 25.64 MiB; 0.528 cores; API p95 97.372 ms | 4,932,000 bytes; privacy 0; envelope clean; cleanup 4 s; residue 0 |
| `preprod-data/run-task8-stress-20260728-130443-17486/` (`stress@1.1.0`) | 100 organizations; 1,000 accounts; 4,000 audits; 400,000 audit events; 36,000 object versions | 528 / 649 of 1,800 s | 258 samples; 24.12 MiB; 0.912 cores; API p95 118.797 ms | exact 536,870,912 bytes; privacy 0; envelope clean; cleanup 5 s; residue 0 |

Every final run contains exactly seven evidence files and reconciles all 40
data families, all eight roles, 86 route dispositions, 306 visible-action
dispositions, exact lifecycle distributions, and relationship digests. Each
retains one expected `COMMAND_STREAM_FAILED` interruption result followed by
one `SUCCEEDED` result, exact load/resume/drop authorizations, positive
resource samples, query/API evidence, 45/45 privacy canaries, and
whole-namespace cleanup with zero container, volume, or network residue.

The full-volume `realistic@1.0.0` and `stress@1.0.0` endurance runs remain
literal `not run` release-readiness evidence. No AWS, deployment, production,
real PII, or normal HTTP seed action occurred.

### Final focused verification

The first final race rerun inside the restricted sandbox exited 1 because
`httptest` could not bind `[::1]:0` (`operation not permitted`), not because an
assertion failed. The exact command was rerun with local loopback permission
and passed:

| Command | Literal result |
|---|---|
| `node --test tests/preprod-identity-data-contract.test.mjs tests/preprod-data-scale-contract.test.mjs tests/preprod-data-boundary.test.mjs` | exit 0; 37/37 passed |
| `go -C apps/api test -race -p 1 -count=1 ./internal/preproddata/... ./cmd/preprod-data-loader` | exit 0; core 1.541 s, profiles 1.234 s, scenarios 183.776 s, loader 1.514 s |
| `go -C apps/api vet ./internal/preproddata/... ./cmd/preprod-data-loader` | exit 0 |
| `bash -n scripts/load-preprod-data.sh scripts/test-preprod-connected-scenarios.sh scripts/test-preprod-data-profile.sh` | exit 0 |
| `./scripts/test-normal-artifact-boundary.sh` | exit 0; `normal-artifact-boundary: ok` |
| `git diff --check` | exit 0 |

Task 8 is therefore `verified locally`, remains `candidate-only`, and is
`release pending`. It was published as
`9a1b5cadff82a047bc0400acdb39f3fab2fbecab`, and the exact `origin/main`
revision was confirmed. Task 9 remains `not run`.
