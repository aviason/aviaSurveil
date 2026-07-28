# Identity And Realistic Data Foundation Plan

**Status:** `completed — Tasks 1–9 verified locally; Task 9 publication pending`

**Reviewed task count:** 9

**Objective:** Complete the production-intended Keycloak-backed account
lifecycle and create a deterministic, privacy-safe, high-volume preprod data
loader that exercises AviaSurveil360 through authoritative application
boundaries.

**User-visible outcome:** Administrators can see and operate the real account
directory, every supported role can complete its intended first-login and
account-lifecycle path, and the local preprod profile can be populated with
repeatable connected data covering every important workflow state.

**Execution dependency:** On 2026-07-27 the user authorized Task 1 discovery,
contract, and owner-decision packaging. Plan 1 and the combined Plans 2–4
stakeholder disposition were completed on 2026-07-28. The current user
authorization permits Tasks 2–9 in sequence, with one freshly verified,
published task commit before the next task starts. Tasks 1–6 are complete and
published; Tasks 2–7 are `26da3c0`, `8cf2b57`, `bedff45`, `ac5f786`,
`26c7022`, and `a2d4744`.
This plan is the predecessor of the AviaCore/ML readiness and local preprod
release-candidate plans. Task 6 is complete, `verified locally`, published,
and remotely confirmed. Task 7 is complete, `verified locally`, published as
`a2d474431d8bcc33ebc0557630cd7038cbcd8ec0`, and remotely confirmed. Task 8
is complete and `verified locally` under owner-approved qualification profile
revision `1.1.0`, published as
`9a1b5cadff82a047bc0400acdb39f3fab2fbecab`, and remotely confirmed. Task 9
is complete and `verified locally`; its focused publication remains pending.
Owner directive `OWNER-DIRECTIVE-2026-07-28-P5T9-01` accepts only the exact
fresh stress run's 19-second duration overrun without changing profile
`stress@1.1.0`, its fail-closed envelope, its volume, or any functional gate.

## Scope

- Keycloak-backed user directory, provisioning, invitation/required-action
  state, TOTP status, desired application membership, role and organization
  authority, activation, suspension, deactivation, reactivation, and session
  revocation.
- Complete HTTP and Admin UI support for the account lifecycle.
- A separate out-of-process preprod data loader with deterministic profiles.
- A complete disposable profile namespace: application PostgreSQL database,
  isolated Keycloak realm/database and service client, synthetic provider
  accounts, Mailpit mailbox, object prefix, and loader-owned queues/jobs. No
  profile writes identities into the shared normal realm.
- Synthetic organizations, users, audits, checklists, Findings, CAPs,
  Evidence references and versions, reports, communications, notifications,
  audit history, and edge states.
- Exact profile manifests, expected counts, relationship checks, privacy
  checks, reset/replay behavior, and evidence.

## Explicit Exclusions

- Public self-registration.
- Real customer, employee, passenger, licence-holder, or operational data.
- Passwords, TOTP secrets, provider tokens, private keys, or recovery codes in
  fixtures, logs, manifests, or evidence.
- A reset or seed endpoint in the normal API, OIDC profile, or HTTP artifact.
- Direct SQL fixture writes that bypass authoritative domain transitions,
  audit events, revisions, privacy, or later AviaCore feed emission.
- AWS deployment, production traffic, external email delivery, or production
  identity federation.
- AviaCore ingestion and ML data products, which belong to the successor plan.

## Durable Constraints

- Preserve the root demo and accepted legacy oracle.
- Keep demo, canonical test, local preprod, and normal HTTP data strictly
  separated.
- Keycloak remains the credential and authentication authority. AviaSurveil
  owns a revisioned, auditable desired-membership aggregate and stores only the
  minimum application identity projection. Provider account, desired
  membership, application profile, invitation delivery, and session state must
  never be collapsed into one ambiguous status.
- CAA roles require the exact `CAA` organization. `auditee` requires one
  non-CAA organization. A client cannot assert roles or organization.
- Internal CAA Notes, private risk values, enforcement deliberation, and
  provider credentials never enter Auditee projections or synthetic evidence.
- The loader must fail closed unless the target declares the exact
  `local-preprod` profile and presents one short-lived, single-use operation
  authorization for exactly one named action. Load or resume authority never
  implies drop/recreate authority.
- The application runtime must use a least-privilege Keycloak service account.
  The bootstrap administrator credential is one-shot bootstrap/recovery
  material and must not be mounted into API, worker, scheduler, or loader
  runtime.
- Normal runtime binaries, route tables, images, startup, migrations, and
  databases contain no demo/canonical seed data and no reachable seed/reset
  implementation. Test-profile code belongs only to a separately selected
  test artifact.
- Every generated record must have a stable scenario identity, source seed,
  owner organization, effective time, revision, and expected relationship.
- Loader intent/result/checkpoint/token-consumption/cleanup control records live
  in a retained append-only control/evidence store outside the disposable
  application/identity/data namespace.
- Commits, pushes, deployment, and external writes require separate current
  authorization.

## Ownership And Required Decisions

| Surface | Accountable owner |
|---|---|
| Membership states, role combinations, organization transfer, activation/deactivation semantics | Product / CAA Operations |
| Keycloak service identity, invitations, MFA, recovery, session and break-glass enforcement | Identity + Security |
| Desired-membership aggregate, lifecycle transactions, directory and retry semantics | Backend |
| PII purpose/minimization, email/audit retention, identifier reuse, legal hold and deletion | Privacy + Records/Legal |
| Disposable loader target, token issuance, resource envelopes and destructive cleanup | Platform/DBA + Security |
| Profile distributions and workflow truth | Product/domain + QA |
| Admin UI and accessibility | Frontend + Accessibility/QA |
| Milestone and successor authorization | Release authority / user |

Task 1 cannot pass with an unnamed owner or unresolved decision. The 11 Task 1
decisions were explicitly approved and versioned on 2026-07-28. Any future
change requires a new contract version; implementation may continue only
around, never through, a newly opened blocked boundary.

## Repository Orientation

Identity authority currently lives in:

- `apps/api/internal/identity/`
- `apps/api/internal/platform/session/`
- `apps/api/internal/administration/users.go`
- `apps/api/internal/administration/user_lifecycle_worker.go`
- `apps/api/internal/administration/projections.go`
- `apps/api/internal/httpapi/auth.go`
- `apps/web/src/features/admin/users-roles-page.tsx`

The canonical test profile is
`apps/api/internal/testprofile/canonical.go`. It remains a bounded browser/test
fixture and must not become the preprod loader. The local production-like
services live under `deploy/local/` and `scripts/`.

## Workload Profiles

The generator must publish an exact manifest before writing data. The initial
engineering profiles remain frozen as contract `1.0.0` targets:

| Profile | Organizations | Users | Audits | Checklist responses | Findings | CAP revisions | Evidence refs/versions | Report versions | Communications and notifications | Audit events |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `smoke` | 3 | 9 | 2 | 24 | 8 | 12 | 16 | 6 | 40 | 250 |
| `acceptance` | 25 | 250 | 1,000 | 10,000 | 3,000 | 4,500 | 6,000 | 2,000 | 20,000 | 100,000 |
| `realistic` | 100 | 2,000 | 20,000 | 250,000 | 60,000 | 100,000 | 200,000 | 75,000 | 1,000,000 | 5,000,000 |
| `stress` | 200 | 4,000 | 40,000 | 500,000 | 120,000 | 200,000 | 400,000 | 150,000 | 2,000,000 | 10,000,000 |

These are synthetic engineering targets, not forecasts or regulatory targets.
On 2026-07-28 the owner approved `OWNER-DIRECTIVE-2026-07-28-P5T8-01`:
full-volume `realistic@1.0.0` and `stress@1.0.0` are retained unchanged as
deferred release-readiness endurance evidence with status `not run`. The
required local qualification profiles are versioned as `realistic@1.1.0`
(2× `acceptance@1.0.0`, maximum 900 seconds total) and `stress@1.1.0`
(4× `acceptance@1.0.0`, maximum 1,800 seconds total and exact 512 MiB object
payload). Both retain all 40 data families, eight roles, 86 routes, 306 visible
actions, relationship reconciliation, privacy, resume, resource, and
whole-namespace cleanup gates. Earlier manifests and create-only evidence must
remain retained.

All four profiles carry the same named identity and end-to-end workflow
scenario families, all eight roles, and an explicit 86-route
data/empty/denied disposition. Increasing scale may add instances but may not
remove a scenario. Task 1 must also freeze per-profile wall-clock, CPU, memory,
disk, object-byte, and cleanup budgets; the numeric row counts alone are not a
feasibility decision.

## Progress

- [x] (2026-07-27) Scope decomposed from the approved local-first preprod
  readiness design.
- [x] (2026-07-27) Existing Keycloak, account lifecycle, canonical profile,
  and full-profile evidence inspected.
- [x] (2026-07-27) Independent cross-repository plan review completed; the
  review added explicit membership authority, least-privilege Keycloak
  administration, deactivation/transfer, normal-artifact seed exclusion,
  append-only-safe reset, complete four-profile coverage, and owner-decision
  gates. Runtime verification remains `not run`.
- [x] (2026-07-27) User authorized Task 1 discovery, contract, and
  owner-decision packaging without waiting for Plan 1 visual stakeholder
  closure. Tasks 2-9 were unauthorized at that checkpoint.
- [x] (2026-07-27) Created the English product contract and machine-readable
  mutation test. The final literal contract run passed 26/26 and the literal
  harness-docs smoke command reported `harness-docs-smoke: ok`.
- [x] (2026-07-28) Obtained all 11 named owner decisions, recorded separate
  `OWNER-DIRECTIVE-2026-07-28-P5T1-*` approval references and exact effective
  values in contract/profile version `1.0.0`, and aligned the optional-TOTP and
  laptop-bounded profile decisions. Runtime implementation remains `not run`.
- [x] (2026-07-28) Committed the complete Task 1 contract, profile, product
  index, and 26/26 mutation gate as `d0f5b29` (`docs(plan5): freeze identity
  and data foundation contract`). Tasks 2-9 were unauthorized and `not run` at
  that checkpoint.
- [x] (2026-07-28) Recorded the combined Plans 2–4 stakeholder disposition and
  moved those three local `candidate-only` milestones to `completed/`. This
  satisfied the predecessor-disposition gate only; Task 2 remains
  unauthorized, not started, and awaits separate explicit authorization.
- [x] (2026-07-28) Task 2 recorded the first strict RED gate,
  `./scripts/test-normal-artifact-boundary.sh`, failed with exit 1 and
  `normal-artifact-boundary: ./cmd/api transitively links
  internal/testprofile`; no Task 2 production implementation preceded this
  result.
- [x] (2026-07-28) Task 2 Keycloak-directory RED was recorded before its
  production implementation. `go test ./internal/identity -run
  'TestKeycloakAdminClient' -count=1` failed with exit 1 because the
  least-privilege `ClientID`/`ClientSecret` configuration and bounded
  `ListDirectory` provider projection did not exist.
- [x] (2026-07-28) Task 2 least-privilege runtime-wiring RED was recorded
  before implementation. The focused platform-config test failed to compile
  because service-client settings did not exist; the Keycloak realm and local
  Compose contract suite failed 5 tests because the lifecycle client, generated
  secret, secret-file wiring, and exact four-role service-account mapping did
  not exist.
- [x] (2026-07-28) Final Task 2 staged review found that membership history was
  described as append-only without a database immutability trigger. The new
  focused integration assertion failed with exit 1 and `desired membership
  history accepted an in-place rewrite`; migration 18 now rejects both update
  and delete, and the focused rerun passed with exit 0.
- [x] (2026-07-28) The same staged review found that the directory derived an
  invitation label from lifecycle/required-action state without joining the
  existing email-delivery fact. After correcting an invalid multi-statement
  test setup, the focused projection test failed with exit 1 because it
  returned `required-actions-complete` instead of `delivered`; the SQLC
  projection now joins the latest lifecycle-bound email delivery and the
  focused rerun passed with exit 0.
- [x] (2026-07-28) Task 2 is complete and `verified locally`. Normal artifacts
  exclude canonical test/reset code; the explicitly tagged canonical-test API
  remains available; membership history is append-only with an immutable
  membership ID, monotonic revision, and current desired/observed record; API
  and worker use only the generated least-privilege Keycloak service-client
  secret; and the Admin directory is provider-account-based with bounded
  pagination, live consistency token, exact filters, drift, required-action,
  MFA, lifecycle, profile, and last-session projections. Provider failure is
  explicit HTTP 503 and pre-login accounts require no session row.
- [x] (2026-07-28) Published the complete Task 2 scope to `origin/main` as
  `26da3c0fb9a3b81bc3a7d6704913cf9fb53ab7b1`
  (`feat(identity): add provider-backed access directory`) and confirmed the
  exact remote ref before starting Task 3.
- [x] (2026-07-28) Task 3 began only after Task 2 was published and its exact
  remote revision was confirmed. No Task 3 production implementation preceded
  its focused RED result.
- [x] (2026-07-28) Task 3 focused RED was recorded before production
  implementation. The Administration/identity test command failed with exit 1
  because the six approved action constants, mandatory `reason` and
  `expectedMembershipRevision`, and Keycloak execute-actions/MFA-reset/logout
  methods did not exist.
- [x] (2026-07-28) Task 3 activation RED was recorded before the activation
  reconciliation implementation. The focused canonical integration build
  failed with exit 1 because `ReconcileActivatedMembership` was undefined.
- [x] (2026-07-28) Final acceptance review recorded an additional strict RED:
  `go -C apps/api test -tags canonicaltest -count=1 ./tests/integration -run
  '^TestTask3DuplicateProviderSubjectRequiresManualReviewBeforeDelivery$'`
  failed with exit 1 because `identity.ErrKeycloakDuplicateSubject` was
  undefined. The final implementation now stops before invitation delivery,
  records `DUPLICATE_SUBJECT` as manual review, and raises one lifecycle alert.
- [x] (2026-07-28) Admin UI completion review added a focused lifecycle-action
  test before the UI expansion. The Web test failed 1/2 with exit 1 because no
  `Lifecycle action` control existed. The HTTP Admin surface now submits every
  approved non-provision lifecycle action with exact revision, reason, role,
  organization, and future-effective transfer fields; the focused rerun passed
  24/24.
- [x] (2026-07-28) Task 3 is complete and `verified locally`. The final
  race-enabled canonical harness passed the complete Administration, identity,
  notification, and 101-test integration packages; the real Keycloak/Mailpit
  all-eight-role proof passed; SQLC, OpenAPI, realm/Compose, Web typecheck,
  focused Web tests, documentation smoke, and cleanup gates passed.
- [x] (2026-07-28) Published the complete Task 3 scope to `origin/main` as
  `8cf2b57fd487e4ba1b2439717425344bb06ea7e3`
  (`feat(identity): complete lifecycle recovery flows`) and confirmed the
  exact remote ref before starting Task 4.
- [x] (2026-07-28) Task 4 is complete and `verified locally`. Exact desired
  membership, provider observation, OIDC claims, session revision, role, and
  organization must agree; lifecycle mutations revoke old authority and force
  a fresh provider observation/login. The frozen 30-second heartbeat,
  60-second maximum observation age, and 120-second provider-loss denial
  deadline pass under the race harness. Live Keycloak/Mailpit/Playwright
  verifies invitation consumption, first login, optional TOTP enrollment,
  wrong-TOTP denial, non-TOTP login, provider restart, deactivation, and exact
  session invalidation. Expiry, lockout, disabled/required-action accounts,
  revoked sessions, JWKS rotation, and clock skew have focused coverage.
  Bootstrap, application Admin, and break-glass remain separate authorities;
  no standing break-glass account is imported, provider login/admin events are
  audited, and actual break-glass use remains blocked without the separately
  authorized external alarm/incident gate.
- [x] (2026-07-28) Published the complete Task 4 scope to `origin/main` as
  `bedff4575704511d107f148c289424b46485d0b2`
  (`feat(identity): enforce live session authority`) and confirmed the exact
  remote ref before starting Task 5.
- [x] (2026-07-28) Task 5 is complete and `verified locally`. The strict RED
  sequence failed at the missing provider-account fact, five missing
  Admin-experience contracts, overlapping lifecycle refresh regression, and
  erased provider identity facts after role replacement. GREEN renders the
  exact provider/application/membership/session record; implements
  reason-confirmed provision and all nine non-provision lifecycle actions;
  exposes exact disabled reasons and stable loading, empty, unavailable,
  pending, succeeded, failed, stale, and retry states; preserves the complete
  Keycloak user representation during authority updates; and passes keyboard,
  focus, error-summary, tablet, mobile, and no-overlap coverage. Fresh
  verification passed Web typecheck, the focused 6/6 Web tests, the isolated
  canonical HTTP Playwright flow 1/1 in 5.4 seconds, the race-enabled identity
  package, the full 650/650 Web suite, and 26/26 identity/data contract tests.
  The task commit was published to `origin/main` as
  `ac5f78619d033ab7d9ed5c63e3de10fdc337ea38` and the exact remote ref was
  confirmed before Task 6.
- [x] (2026-07-28) Published the complete Task 5 scope to `origin/main` as
  `ac5f78619d033ab7d9ed5c63e3de10fdc337ea38`
  (`feat(identity): complete admin lifecycle experience`) and confirmed the
  exact remote ref before starting Task 6.
- [x] (2026-07-28) Task 6 is complete and `verified locally`. Task 5 and its
  publication record are remotely confirmed. The strict Node boundary RED
  failed 3/3 because the loader script and source did not exist and the
  normal-artifact guard did not positively identify the loader command. The
  strict Go RED then failed to
  build because both new packages had no implementation files and
  `loadRunConfiguration` was undefined. The initial sandboxed Go attempt was
  discarded because the Go cache was outside the writable sandbox; the
  task-owned-cache rerun is the behavioral RED. Acceptance review then added
  the complete disposable-namespace contract; its focused Node rerun failed
  1/4 because the isolated object-store initializer and the required
  PostgreSQL, migration, Keycloak, Mailpit, and object-store services did not
  exist. This RED preceded that infrastructure implementation. The
  exact-completed-run replay test then failed because the runner attempted to
  consume the already-used authorization instead of returning the immutable
  successful result without touching the target. That RED preceded the result
  binding/replay implementation. The reconciliation RED then failed because a
  run reporting one organization against the intent's expected three was
  incorrectly accepted as `SUCCEEDED`; that RED preceded exact count
  reconciliation. The frozen-profile mutation RED then failed because the
  generator accepted a changed `smoke@1.0.0` count/distribution without a new
  profile version; that RED preceded catalog equality enforcement. The cleanup
  linkage RED then failed because a consumed `LOAD_EMPTY_TARGET` authorization
  could incorrectly attest namespace cleanup; that RED preceded exact
  successful-result, target, and `DROP_RECREATE_TARGET` authorization
  validation. The direct result-store RED then failed because an incomplete
  successful result could bypass the runner and bind to the run; that RED
  preceded store-level intent reconciliation. The bounded-streaming RED then
  failed because `RunInput` retained every authoritative command in a slice
  and exposed no `CommandStream.Next` boundary; that RED preceded the
  constant-memory iterator contract required by the larger frozen profiles.
  The checkpoint-bound RED then failed with 2,050 retained checkpoint names
  for 2,049 streamed commands; that RED preceded the intent-sized, maximum
  1,024 command-checkpoint schedule. The namespace overwrite RED then failed
  because the initializer exposed `--rotate` and force-move paths outside the
  operation-authorization boundary; that RED preceded create-only
  initialization. The boundary-redaction RED then failed because a sentinel
  returned by a server-owned command boundary leaked into both the persisted
  failure result and returned error; that RED preceded public error
  classification. The command-payload RED then failed all four unsafe cases:
  malformed JSON, nested `client_secret`, a hyphenated access-token key, and a
  payload above 1 MiB; that RED preceded bounded structural payload
  validation. The resume RED then failed with an append-only checkpoint
  conflict because `RESUME_RUN` restarted checkpoint sequence 1 and replayed
  from the beginning; that RED preceded durable checkpoint discovery and
  resumable-stream positioning. The isolated SMTP RED then failed because the
  generated realm retained normal username `aviasurveil360` while the
  dedicated Mailpit auth file required `aviasurveil360-preprod`; that RED
  preceded exact SMTP-user parameterization. The control-store-root RED then
  failed because a group/world-readable directory was accepted for retained
  manifests and authorization hashes; that RED preceded explicit broad-root
  rejection and private-directory validation. Final diff review added an
  intent-reader RED that failed because a second JSON object after a valid
  canonical intent was accepted; that RED preceded strict trailing-content
  rejection. No implementation of a corrected behavior preceded its
  corresponding failure. The final loader keeps
  every normal runtime free of loader/reset imports, uses the exact
  server-owned command boundary, retains immutable digest-linked control
  records outside the disposable target, and permits cleanup attestation only
  after a separate consumed `DROP_RECREATE_TARGET` authorization. At Task 6
  closure, Tasks 7–9 were still `not run`.
- [x] (2026-07-28) The first complete Task 6 verification attempt passed the
  4/4 Node boundary suite, HTTP artifact scan, and normal-artifact boundary.
  The required race run failed six loader tests because their task-owned
  temporary control-store roots did not explicitly apply the new private
  directory mode. This was a test-fixture failure, not passing evidence. After
  correcting those fixtures, the fresh required rerun passed the Node boundary
  4/4, both race-enabled Go packages, the HTTP artifact scan across 146 files
  and 177 inputs, and the normal-artifact boundary. Compose rendered exactly
  the 15 reviewed normal services for `full` and the nine isolated preprod
  services for `local-preprod-loader`; the three new shell entrypoints are
  executable and syntax-clean; Go vet passed; and the related identity,
  Keycloak, and Compose policy regression passed 50/50.
- [x] (2026-07-28) Published the complete Task 6 scope to `origin/main` as
  `26c7022ebed02c99c7183850f7138ea63580bf0c`
  (`feat(data): add isolated preprod loader`) and confirmed the exact remote
  ref before starting Task 7.
- [x] (2026-07-28) Task 7 is complete and `verified locally`. The strict
  connected-scenario RED and every later acceptance-review RED are recorded
  below. The final fresh aggregate exited 0 with 40/40 reconciled families,
  86 routes, 306 visible actions, all eight roles, 45/45 retained
  cross-organization privacy canaries, a separately authorized networkless
  cleanup attestation, and zero task-owned residue. Task 7 was published as
  `a2d474431d8bcc33ebc0557630cd7038cbcd8ec0` and the exact remote ref was
  confirmed; Tasks 8–9 remain `not run`.
- [x] (2026-07-28) Task 8 recorded the strict scale-contract RED
  `node --test tests/preprod-data-scale-contract.test.mjs` exited 1 with
  `Could not find 'tests/preprod-data-scale-contract.test.mjs'`. The strict
  profile-runner RED `./scripts/test-preprod-data-profile.sh smoke` exited 127
  with `no such file or directory`. Neither required Task 8 artifact existed
  before these failures.
- [x] (2026-07-28) Task 8 `smoke@1.0.0` and `acceptance@1.0.0`
  qualification passed on the current implementation with exact seven-file
  evidence, complete catalogs, interruption/resume, positive resource
  samples, zero privacy findings, bounded cleanup, and zero residue.
- [x] (2026-07-28) At the owner's request, the running full-volume
  `realistic@1.0.0` attempt
  `run-task8-realistic-20260728-121351-92512` was interrupted safely with exit
  130. Its cleanup trap removed all task-owned containers, volumes, and
  networks; protected PID 13055 remained alive. The create-only failed run is
  retained.
- [x] (2026-07-28) Owner-approved qualification revision
  `OWNER-DIRECTIVE-2026-07-28-P5T8-01` is complete. The new strict RED
  package passed 27/30 Node tests but rejected absent `1.1.0` qualification
  manifests, total-duration enforcement, and owner metadata. The focused Go
  RED failed to compile because `QualificationSeconds` was undefined and also
  rejected unknown `stress@1.1.0`.
- [x] (2026-07-28) Final `realistic@1.1.0` completed in 326/900 seconds
  total with 111 positive resource samples, 228 seconds generation, 25.64 MiB
  peak loader memory, 0.528 peak CPU cores, zero privacy findings, complete
  reconciliation, cleanup in four seconds, and zero residue.
- [x] (2026-07-28) The first `stress@1.1.0` live attempt exposed
  `TARGET_RECONCILIATION_FAILED`. The focused RED then failed with `first
  local stress object size = 0, expected 14914`, proving the exact payload
  helper was limited to version `1.0.0`. The versioned fix retained the 8 GiB
  endurance payload and produced the required local exact 512 MiB payload.
- [x] (2026-07-28) Final `stress@1.1.0` completed in 649/1,800 seconds
  total with exactly 536,870,912 object bytes across 36,000 versions, 258
  positive resource samples, 528 seconds generation, 24.12 MiB peak loader
  memory, 0.912 peak CPU cores, zero privacy findings, cleanup in five
  seconds, and zero residue. Final focused Node, race-enabled Go, vet, shell,
  normal-artifact, and diff gates passed.
- [x] (2026-07-28) Task 9 completed its fresh full matrix. The final exact Go
  race rerun passed every package after all Task 9 changes, with scenarios at
  147.408 seconds and integration at 42.686 seconds. The fresh realistic run
  passed at 872/900 seconds. The fresh stress authoritative run passed every
  exact data, relationship, privacy, resume, resource, and cleanup gate and
  cleaned to residue 0; it recorded 1,819/1,800 seconds, and owner directive
  `OWNER-DIRECTIVE-2026-07-28-P5T9-01` accepted only that exact 19-second Task
  9 overrun. A separate final main-agent code/specification review found no
  Critical or Important finding. Task 9 publication remains pending.

## Tasks

### Task 1: Freeze Identity And Data-Profile Acceptance Contracts

**Task status:** `complete`; the versioned contract, all named owner decisions,
and machine-checkable acceptance gates are closed. API, Keycloak, session,
directory, loader, and profile runtime implementation remains `not run`, and
Tasks 2–9 were not part of the Task 1 commit. The current authorization now
permits them sequentially.

**Files**

- Create `docs/product-specs/data-and-rules/PREPROD_IDENTITY_AND_DATA_PROFILE.md`.
- Create `tests/preprod-identity-data-contract.test.mjs`.
- Modify `docs/product-specs/index.md`.
- Modify this plan, `docs/exec-plans/index.md`, and
  `docs/exec-plans/tech-debt-tracker.md`.

**Work**

- [x] Enumerate all eight roles, permitted organization combinations, account
  states, first-login required actions, session outcomes, and forbidden role
  combinations.
- [x] Freeze separate state machines and field authorities for provider
  account, desired application membership, application profile, invitation/
  recovery delivery, MFA enrollment, and sessions. The membership aggregate
  must have a stable business key, monotonic revision, requested/effective
  time, actor/reason, organization, role set, and drift/reconciliation state.
- [x] Define distinct `requested`, `invited`, `active`, `suspended`,
  `deactivated`, and `reactivation-pending` membership semantics. Suspension is
  temporary; deactivation removes future authority without deleting identity
  or rewriting historical ownership.
- [x] Obtain named Product, Security, Identity, Records, and Operations
  decisions for invitation channel/expiry/resend, account activation,
  Auditee MFA, recovery/MFA reset, deactivation and
  identifier-reuse retention, organization transfer, permissible multi-role
  sets, bootstrap Admin, break-glass, and Keycloak service-account privileges.
  Missing decisions block the affected implementation.
- [x] Freeze the invitation baseline as Keycloak execute-actions delivery
  through the approved local channel, never an application-returned/stored
  temporary password. `UPDATE_PASSWORD` is mandatory for every newly
  provisioned passwordless account; owners decide whether `VERIFY_EMAIL` and
  `CONFIGURE_TOTP` additionally apply by role. A different credential bootstrap
  requires an explicit plan/contract revision before Task 3. The approved
  channel is authenticated local SMTP/Mailpit with 24-hour expiry, prior-action
  invalidation, and at most three resends per 24 hours. `VERIFY_EMAIL` applies
  to every role; TOTP is optional for every role and `CONFIGURE_TOTP` is not
  required.
- [x] Freeze a numeric maximum provider-observation age, reconciliation
  heartbeat/deadline, bounded provider-call/cache policy, and outage behavior.
  Existing authority must fail closed no later than that deadline after
  provider disablement, role/organization drift, stale observations, or loss
  of provider availability.
- [x] Freeze the four workload profiles above as versioned manifests with
  exact counts, deterministic seed input, clock origin, identity namespace,
  state distribution, maximum object-byte budget, and numeric CPU, memory,
  disk, duration, and cleanup envelopes.
- [x] Derive exact proposed per-profile counts/distributions for every
  generated family,
  including plans/approvals, assignments, templates/questions/packages,
  Potential Findings, review decisions, sessions/offline grants, calendar,
  outbox/delivery/scanner/render jobs, objects/versions, and sync/change-feed
  state. Audit/outbox totals derive from the frozen command mix rather than an
  independent invented number.
- [x] Require exact counts and relationship digests for every generated
  authoritative entity/version, provider account/state, membership, invitation
  delivery/mail item, session/offline grant, object/version, audit/outbox,
  notification/delivery, scanner/render job, calendar/sync record, and
  intentional empty/denied disposition. Any generated type absent from the
  manifest fails the contract.
- [x] Define exact lifecycle distributions for planned, active, overdue,
  returned, rejected, corrected, superseded, reopened, partially closed, not
  closed, authorized-closed, and normally verified-closed records.
- [x] Define a closed synthetic-data dictionary. Every text generator must use
  allowlisted vocabulary and must never derive from repository history, logs,
  customer exports, or local address books.
- [x] Add mutation tests that reject public registration, normal-mode seed
  routes, client-authored roles, real-looking PII, missing expected counts, and
  unversioned workload changes.
- [x] Require every profile to include the same named lifecycle scenario
  catalog, eight-role coverage, and all 86 route dispositions; fail if a
  smaller profile silently drops a workflow.
- [x] Define a normal-artifact contract that rejects a seed/reset route,
  loader command, canonical-header authentication, test-profile import graph,
  seed startup hook, non-empty clean schema, or demo data in API/worker/
  scheduler/migration images.

**Verification**

Run:

    node --test tests/preprod-identity-data-contract.test.mjs
    node tests/harness-docs-smoke.test.js
    git diff --check

Expected: the contract test fails before the specification and manifest schema
exist, then passes with one exact active contract and no normal-mode seed
surface.

**Acceptance**

- The role/account/profile matrix is machine-checkable.
- The profiles have exact deterministic counts and privacy bounds.
- No implementation starts with ambiguous membership or data-volume semantics.
- No unresolved owner decision is converted into an implementation default.

Current acceptance result: Task 1 is complete. Contract `1.0.0` records all 11
owner approvals and exact effective values; the mutation test rejects an
unresolved decision used as an implementation assumption. This is
documentation/contract acceptance only. Runtime implementation and feasibility
are `not run`.

**Fresh literal verification — 2026-07-28**

- `node --test tests/preprod-identity-data-contract.test.mjs` — passed 26/26
  with exit 0.
- `node tests/harness-docs-smoke.test.js` — passed with exit 0 and
  `harness-docs-smoke: ok`.
- `git diff --check` — passed with exit 0 and no output.

**Task 2 gate:** satisfied. The later explicit authorization started Task 2
after the combined Plans 2–4 stakeholder disposition and published baseline.

### Task 2: Replace Session-Derived Directory Placeholders With Keycloak State

**Task status:** `complete`; implementation and fresh verification passed on
2026-07-28 and the exact published revision is
`26da3c0fb9a3b81bc3a7d6704913cf9fb53ab7b1`.

**Recorded RED — normal-artifact split:** On 2026-07-28,
`./scripts/test-normal-artifact-boundary.sh` failed with exit 1 because
`./cmd/api` transitively linked `internal/testprofile`. This is the expected
pre-implementation failure.

**Recorded RED — append-only enforcement:** Final staged review added an
integration mutation proof before the missing enforcement. The focused
lifecycle-worker test failed with exit 1 and `desired membership history
accepted an in-place rewrite`. Migration 18 now applies the shared immutable
row trigger, and the focused rerun passed with exit 0.

**Recorded RED — invitation delivery authority:** After correcting an invalid
multi-statement test setup, the focused Administration projection test failed
with exit 1 because a seeded delivered invitation was returned as
`required-actions-complete`. The SQLC directory projection now joins the
latest lifecycle-bound email-delivery fact and returns its normalized delivery
state; the focused rerun passed with exit 0.

**Files**

- Modify `apps/api/internal/identity/keycloak_admin.go`.
- Modify `apps/api/internal/administration/projections.go`.
- Modify `apps/api/cmd/api/main.go`, canonical test-handler composition,
  `apps/api/Dockerfile`, and build scripts to create separate normal and
  canonical-test API artifacts.
- Create `scripts/test-normal-artifact-boundary.sh`.
- Modify `deploy/local/keycloak/realm-source.json`, local Compose/config, and
  secret initialization to introduce the least-privilege service account and
  remove bootstrap-admin material from normal runtime.
- Add the next forward-only desired-membership migration and Administration
  SQLC queries before building the directory projection.
- Modify `apps/api/internal/httpapi/management_admin_assistant_api.go`.
- Modify `api/openapi/source/paths/platform.json`.
- Modify `api/openapi/source/schemas/platform.json`.
- Regenerate `api/openapi/aviasurveil360.yaml`,
  `apps/api/internal/httpapi/generated/api.gen.go`, and
  `apps/web/src/generated/transport/api-types.ts`.
- Modify `apps/web/src/backend/backend.ts`,
  `apps/web/src/backend/http-backend.ts`, and transport tests.
- Test `apps/api/tests/integration/identity_organization_scope_test.go`.

**Work**

- [x] First split and verify normal versus canonical-test API composition.
  Tasks 2-5 cannot pass while any normal API/worker/scheduler/migration binary
  or image transitively links `internal/testprofile`, canonical reset, or
  loader code. Preserve a separately named positive canonical-test artifact.
- [x] Persist append-only desired-membership versions and a current
  desired/observed synchronization record before exposing the directory.
  Expose the revision for Task 4 session enforcement.
- [x] Extend the Keycloak admin adapter with paginated, bounded, timeout-aware
  directory reads and a minimal account-state projection.
- [x] Join provider identity state to AviaSurveil profile, organization,
  revisioned desired membership, lifecycle-request, invitation-delivery, and
  active-session summaries without copying credentials or required-action
  secrets.
- [x] Return real email, provider enabled state, desired membership state,
  derived invitation/required-action state, MFA enrollment boolean, roles,
  organization, membership revision/drift, and last successful application
  session time. Do not present invitation as a native Keycloak state when it is
  derived from application lifecycle and delivery records.
- [x] Replace bootstrap-admin password grant use with a confidential
  least-privilege service account using client credentials and exact realm
  management permissions. Prove the bootstrap credential is unavailable to
  normal runtime and that excess permissions fail policy tests.
- [x] Distinguish `provider-unavailable` from an empty directory. Never replace
  an unavailable provider with stale success or `Not configured in demo`.
- [x] Enforce exact Admin authorization, bounded pagination, search, role,
  organization, and account-status filters.
- [x] Return exactly one account row per subject with `roles[]`, deterministic
  sort/tie-breaker, stable opaque cursor, snapshot/consistency semantics,
  maximum page size/provider calls, and no unbounded N+1 Keycloak queries.
- [x] Add raw-wire tests proving Auditee and non-Admin denial and proving that
  credentials, TOTP material, provider tokens, and Internal CAA data cannot
  appear.

**Verification**

Run:

    ./scripts/check-contracts.sh
    ./scripts/test-normal-artifact-boundary.sh
    go -C apps/api test -tags canonicaltest -race -p 1 -count=1 ./internal/identity ./internal/administration ./internal/httpapi ./tests/integration
    npm --prefix apps/web test -- --run src/backend/http-backend.test.ts

Expected: real provider states appear for provisioned users, pre-login users
appear without requiring a session row, and provider failure is explicit.

**Fresh literal verification — 2026-07-28**

- `./scripts/check-contracts.sh` — passed with exit 0 and
  `contracts-check: ok`.
- `./scripts/test-normal-artifact-boundary.sh` — passed with exit 0 and
  `normal-artifact-boundary: ok`.
- `go -C apps/api test -tags canonicaltest -race -p 1 -count=1
  ./internal/identity ./internal/administration ./internal/httpapi
  ./tests/integration` — passed with exit 0.
- `npm --prefix apps/web test -- --run src/backend/http-backend.test.ts` —
  passed 22/22 with exit 0.
- `./scripts/check-sqlc.sh` — passed with exit 0 and `sqlc-check: ok`.
- `node --test deploy/local/keycloak/realm-contract.test.mjs
  tests/local-compose-policy.test.mjs` — passed 24/24 with exit 0.
- `npm --prefix apps/web run typecheck` — passed with exit 0.

The durable Task 1–2 record is
[Plan 5 Identity And Data Foundation Evidence](../../demo-evidence/PLAN5_IDENTITY_DATA_FOUNDATION_2026-07-28.md).

**Acceptance**

- The directory is account-based rather than active-session-based.
- Every exposed field has one named authority and a fail-closed unavailable
  state.
- Provider drift is visible and cannot silently overwrite desired membership.

### Task 3: Complete Provisioning, Invitation, And Recovery

**Task status:** `complete`; strict RED evidence, implementation, full focused
verification, cleanup evidence, and publication are recorded. The exact
published revision is `8cf2b57fd487e4ba1b2439717425344bb06ea7e3`.

**Recorded RED:** `go -C apps/api test -count=1
./internal/administration ./internal/identity -run
'Task3|Passwordless|ExecuteActions'` failed with exit 1 before Task 3
production implementation. The compiler reported the missing six lifecycle
actions, `reason`, `expectedMembershipRevision`,
`IssueExecuteActionsEmail`, `ResetUserMFA`, and `ForceUserLogout`.

**Additional recorded REDs:** The activation-focused canonical integration
build failed with exit 1 because `ReconcileActivatedMembership` was undefined.
Final acceptance review then added the duplicate-provider-subject test; it
failed with exit 1 because `identity.ErrKeycloakDuplicateSubject` was
undefined. Admin UI review added the missing lifecycle-action control test,
which failed 1/2 because no `Lifecycle action` label existed. All three
failures preceded their respective production implementations.

**Files**

- Modify `apps/api/internal/administration/users.go`.
- Modify `apps/api/internal/administration/user_lifecycle_worker.go`.
- Modify `apps/api/internal/identity/keycloak_admin.go`.
- Modify `apps/api/internal/notifications/`.
- Add the next forward-only lifecycle/invitation/delivery/retry migration under
  `apps/api/migrations/`.
- Modify Administration SQLC queries and regenerate their checked artifacts.
- Create `scripts/test-preprod-identity-lifecycle.sh` and its all-eight-role
  command/evidence/cleanup contract.
- Modify the lifecycle OpenAPI schemas and generated transports.
- Test `apps/api/tests/integration/identity_organization_scope_test.go`.

**Work**

- [x] Preserve `PROVISION`, `UPDATE_ROLES`, `SUSPEND`, and `REACTIVATE`; add
  the Task 1-approved `DEACTIVATE`, `TRANSFER_ORGANIZATION`, invitation resend,
  required-password reset, MFA reset, and forced-logout actions with
  reason-required audit records and exact OpenAPI enums.
- [x] Require `expectedMembershipRevision` on every account/membership
  authority mutation; provisioning requires no existing revision. A stale
  revision returns a typed conflict and performs no provider, delivery,
  session, audit-success, or outbox-success mutation.
- [x] Make provisioning produce only the approved one-time, expiring Keycloak
  execute-actions invitation path without generating, returning, or storing an
  application temporary password.
- [x] Implement only the exact Task 1-approved local invitation contract:
  bounded Keycloak execute-actions delivery through authenticated local mail;
  require `UPDATE_PASSWORD` and add the role-policy-selected `VERIFY_EMAIL`
  and/or `CONFIGURE_TOTP`; freeze TTL, resend, already-used, expiry, lockout,
  recovery, MFA reset/re-enrollment, and all-eight-role behavior.
- [x] Record invitation issue, delivery, expiry, resend, activation,
  cancellation, and recovery as separate application facts. Never store a
  credential or action token; provider and delivery acknowledgements remain
  reconcilable after timeout.
- [x] Apply the approved optional-all-roles TOTP policy: never require
  `CONFIGURE_TOTP`, and store only provider-derived enrollment status.
- [x] Make duplicate email, duplicate subject, role drift, unknown
  organization, expired invitation, and stale lifecycle request deterministic
  typed outcomes.
- [x] Preserve idempotency through provider timeout and lost acknowledgement.
  Reconcile a provider-created user before retrying creation.
- [x] Classify retryable, permanent, and manual-review provider outcomes; use
  approved exponential backoff and attempt/time caps, terminal
  `FAILED_PERMANENT`/`MANUAL_REVIEW` states, operator reconciliation, metrics,
  and alerts. Never retry every provider failure indefinitely.
- [x] Audit request, provider result, subject binding, role/organization
  binding, session invalidation, invitation delivery, and failure without
  recording secret material.
- [x] Expose all approved non-provision actions through the HTTP Admin UI with
  reason and exact membership revision, plus role selection for updates and
  destination/effective-time inputs for organization transfer.
- [x] Make deactivation distinct from suspension and deletion: revoke
  sessions, disable future authority, retain the minimum identity/membership
  tombstone under the approved retention policy, and require a new audited
  reactivation decision.

**Verification**

Run the focused Keycloak and lifecycle integration tests under the canonical
service harness, then:

    ./scripts/check-sqlc.sh
    ./scripts/check-contracts.sh
    go -C apps/api test -tags canonicaltest -race -p 1 -count=1 ./internal/administration ./internal/identity ./internal/notifications ./tests/integration
    ./scripts/test-preprod-identity-lifecycle.sh invitation

Expected: every action is idempotent, reason-audited, provider-reconciled, and
secret-free.

**Fresh literal verification — 2026-07-28**

- `./scripts/check-sqlc.sh` — exit 0; `sqlc-check: ok`.
- `./scripts/check-contracts.sh` — exit 0; `contracts-check: ok`; OpenAPI
  contract tests 16/16.
- `./scripts/test-preprod-identity-lifecycle.sh invitation` — exit 0; the
  race-enabled Administration, identity, notification, and full canonical
  integration packages passed, with integration completing in 35.130 seconds;
  the harness printed `Plan 5 Task 3 invitation and lifecycle harness:
  verified locally` and removed all task-owned containers, volumes, and
  network.
- `./scripts/test-preprod-identity-lifecycle.sh all-eight-roles` — exit 0; the
  real Keycloak/Mailpit all-eight-role integration test passed and the harness
  printed `Plan 5 Task 3 all-eight-role identity lifecycle: verified locally`
  before complete cleanup.
- `node --test deploy/local/keycloak/realm-contract.test.mjs
  tests/local-compose-policy.test.mjs` — exit 0; 24/24 passed.
- `npm --prefix apps/web run typecheck` — exit 0.
- `npm --prefix apps/web test -- --run src/backend/http-backend.test.ts
  src/features/admin/users-roles-page.test.tsx` — exit 0; 24/24 passed.
- `node --test tests/harness-docs-smoke.test.js` — exit 0;
  `harness-docs-smoke: ok`.

The first expanded full-suite harness run exposed an existing projection
fixture that passed an interval where migration 19 requires an absolute
timestamp. The next run exposed missing task-harness MinIO credentials. The
fixture and isolated harness were corrected; neither failing run is presented
as completion evidence, and the final complete rerun above passed.

**Acceptance**

- A new user can move from requested through invited, activated/first login,
  MFA, active, suspended, deactivated, and explicitly reactivated states.
- Failed delivery/provider operations are visible and deterministically
  classified as bounded-retryable, permanent, or manual-review; permanent
  failures are never automatically retried.

### Task 4: Enforce Role, Organization, MFA, And Session Lifecycle

**Task status:** `complete`; Task 3 was published and remotely confirmed before
start, all strict RED results are recorded, the complete Task 4 gate is
`verified locally`, and the focused task is published as
`bedff4575704511d107f148c289424b46485d0b2`.

**Recorded RED:** `go -C apps/api test -tags canonicaltest -count=1
./internal/identity ./tests/integration -run 'Task4'` failed with exit 1
because `identity.ValidateApplicationAuthority`,
`identity.AuthorityObservation`, `identity.AuthorityObserver`, and the
session-manager `AuthorityObserver` dependency were undefined. The subsequent
provider-observation RED, `go -C apps/api test -count=1 ./internal/identity
-run 'Task4'`, failed with exit 1 because
`KeycloakAdminClient.ObserveUserAuthority` did not exist. The frozen-deadline
RED then ran under the isolated canonical harness and failed because the
31-second authentication performed only the initial provider observation
instead of the required heartbeat. The first-login activation RED then failed
to compile because `session.ActivationReconciler` and the matching manager
dependency did not exist. The callback fail-closed RED,
`go -C apps/api test -count=1 ./internal/httpapi -run
'^TestTask4OIDCCallbackRejectsStaleAuthorityAndExpiresCookies$'`, failed with
exit 1 because stale authority returned HTTP 500 `SESSION_CREATE_FAILED`
without expiring browser cookies instead of HTTP 401 `STALE_AUTHORITY`. The
mutation-freshness RED, `go -C apps/api test -tags canonicaltest -run
'^TestTask4AuthorityMutationForcesFreshProviderObservation$'
./tests/integration`, then failed to compile because
`session.RequireFreshAuthorityObservation` did not exist.
The first updated `./scripts/test-http-oidc-profile.sh` acceptance run then
failed before API/browser startup because the isolated one-shot lifecycle
actor lacked the retained identity reference required by the
`user_lifecycle_requests` foreign key. The harness removed its containers,
volumes, network, runtime directory, and generated secrets after the failure.
The second OIDC acceptance run passed the authoritative Admin setup but failed
readiness before Playwright because the stale harness started API and worker
bucket initialization concurrently and selected unavailable ClamAV for this
isolated auth lane. Cleanup again removed all task-owned runtime state.
The third run reached Playwright with healthy API and worker processes, then
failed before opening a browser because the repository-pinned Chromium 1228
binary was absent. The exact pinned Chromium and headless-shell artifacts were
subsequently installed; no product implementation was changed to bypass the
required browser gate.
The fourth run reached real Keycloak TOTP enrollment and returned to the app,
then failed because the first navigation-time session poll raised a transient
browser `Failed to fetch` exception instead of retrying. The helper remained
strict for persistent HTTP status failures; task-owned browser and container
processes were cleaned up.
The fifth run confirmed that in-page `fetch` remained unavailable after the
callback even though the page stayed on the app origin. The assertion was
therefore moved to Playwright's browser-context request client, which shares
the same cookies and reports the same `/auth/session` HTTP status without
depending on page JavaScript execution.
The sixth run then reported the exact HTTP outcome: the callback returned to
the app but `/auth/session` remained HTTP 401 for the full polling window. A
secret-free in-harness diagnostic was added to compare desired membership,
sync, retained identity/profile, and live provider authority before cleanup.
That diagnostic reported ACTIVE membership revision 2, exact/current sync,
matching retained issuer and profile, and live enabled/unlocked Admin/CAA
authority with TOTP enrolled and no required actions, but zero application
sessions. The remaining failure is therefore after activation in callback
session creation; the browser test now captures the callback status/problem
directly.
The eighth run captured HTTP 401 `STALE_AUTHORITY` directly. Combined with the
exact diagnostic, this isolated the defect to the pre-activation session clock:
the activation revision became effective milliseconds after `Manager.Create`
captured `now`, so its post-activation transaction falsely treated the new
revision as future-effective. GREEN refreshes the clock after successful
activation without relaxing any authority comparison.
The next live run proved that the exact session was persisted and authoritative
but that a plain `127.0.0.1` origin rejected the production `Secure`
`__Host-` cookies. The isolated browser origin was corrected to
`http://localhost:4174`; cookie security attributes were not weakened. The
following run reached the lifecycle API and exposed stale test payloads that
omitted the mandatory reason. The final provider RED found that the reviewed
realm had not registered `UPDATE_PASSWORD` and `VERIFY_EMAIL` execute-action
providers; the focused realm contract failed 1/3 before those required actions
were added.
Final diff review added a concurrent-revision regression. The isolated race
harness failed with exit 1 because session creation returned a real session ID
with no error after the desired-membership synchronization row changed away
from the revision that creation had validated. GREEN now requires exactly one
current-revision synchronization row to be refreshed before inserting the
session; a concurrent revision change returns `ErrUnauthenticated` and creates
no stale session.
The same review added an exact-organization variant. The focused identity test
failed because padded `" CAA "` was normalized and accepted. GREEN now rejects
non-canonical leading/trailing whitespace rather than silently rewriting an
organization identity.

**Fresh GREEN:** `./scripts/check-sqlc.sh` exited 0 with `sqlc-check: ok`.
`./scripts/test-preprod-identity-lifecycle.sh session-authority` exited 0
after running the exact tagged race command across session, identity,
Administration, HTTP, and the full integration package; integration completed
in 37.017 seconds and the harness printed `Plan 5 Task 4 session authority:
verified locally`. `./scripts/test-http-oidc-profile.sh` exited 0 with
Playwright 1/1 in 46.7 seconds and `OIDC runtime secret/log scan: zero
generated-secret matches`. Both harnesses removed their task-owned containers,
volumes, networks, runtime state, and browser processes. The focused remote
OIDC rotation/clock test and the combined realm/Task 1 contract suite also
passed; the latter reported 29/29. A direct tagged race invocation without its
fixtures failed only with connection refusals at the documented PostgreSQL and
MinIO ports; the final isolated harness provisioned those dependencies and ran
that same command successfully.

**Files**

- Modify `apps/api/internal/administration/authorization.go`.
- Modify `apps/api/internal/platform/session/manager.go`.
- Modify `apps/api/internal/identity/oidc_remote.go`.
- Modify `apps/api/internal/identity/keycloak_admin.go`.
- Modify `apps/api/internal/httpapi/auth.go`.
- Add the next forward-only session-membership-revision migration, session
  SQLC queries, and regenerated checked artifacts.
- Test identity, session, authorization, and organization integration packages.

**Work**

- [x] Enforce allowed role sets and prohibit CAA/Auditee organization drift at
  request, provider, callback, session, and projection boundaries.
- [x] Revoke all active sessions after role change, organization transfer,
  suspend, deactivate, MFA reset, or forced logout.
- [x] Require a fresh OIDC login after authority changes and reject stale role
  or organization claims.
- [x] Persist the desired-membership revision in every application session and
  fail closed unless current desired membership, observed provider authority,
  token claims, and the session revision agree.
- [x] Fail both new and existing sessions on provider/desired-membership drift,
  partial multi-call role replacement, old membership revision, or provider
  unavailability. Enforce the Task 1 maximum observation age and deny an
  already-active session within the frozen deadline after provider disablement,
  authority drift, stale observation, or provider loss. Reconciliation may
  restore authority only after fresh exact desired and observed state agree.
- [x] Add controlled organization transfer only as a separately reasoned
  `TRANSFER_ORGANIZATION` lifecycle action with expected membership revision;
  never mutate historical record ownership.
- [x] Define and test bootstrap Admin, application Admin, and break-glass
  identities as separate authorities. Break-glass use must be alarmed and
  audited.
- [x] Exercise expiry, lockout, TOTP failure, disabled user, revoked session,
  provider restart, signing-key rotation, and clock-skew boundaries.

**Verification**

Run:

    ./scripts/check-sqlc.sh
    go -C apps/api test -tags canonicaltest -race -p 1 -count=1 ./internal/platform/session ./internal/identity ./internal/administration ./internal/httpapi ./tests/integration
    ./scripts/test-http-oidc-profile.sh
    ./scripts/test-preprod-identity-lifecycle.sh session-authority

Expected: all stale and invalid authority paths fail closed and every valid
first-login path reaches only its authorized workspace.

**Acceptance**

- No account can retain old authority after a lifecycle change.
- MFA and account state are enforced by the provider, not simulated by the UI.

### Task 5: Finish The Users And Roles Admin Experience

**Task status:** `complete — verified locally and published`; Task 5 is
published as `ac5f78619d033ab7d9ed5c63e3de10fdc337ea38`, and its exact remote
revision is confirmed. Task 6 remains `not run`.

**Files**

- Modify `apps/web/src/features/admin/users-roles-page.tsx`.
- Modify `apps/web/src/features/admin/users-roles-page.test.tsx`.
- Modify shared Admin styles only where existing patterns require it.
- Modify `apps/web/src/backend/backend.ts` and HTTP transport mappings as
  required by Tasks 2-4.
- Add focused Playwright coverage under `apps/web/tests/e2e/`.

**Work**

- [x] Render provider, application profile, role, organization, invitation,
  MFA, account, and session state without placeholder values.
- [x] Add working provision, resend invitation, update role, transfer
  organization, suspend, deactivate, reactivate, MFA reset, and force logout
  flows.
- [x] Require confirmation and reason entry for authority-changing or
  destructive actions.
- [x] Disable actions with an exact reason when provider state or actor
  authority makes them unavailable.
- [x] Provide stable loading, empty, unavailable, pending, succeeded, failed,
  stale, and retry states.
- [x] Verify keyboard, focus, error summary, mobile layout, and no-overlap
  behavior at desktop, tablet, and mobile.

**Verification**

Run:

    npm --prefix apps/web run typecheck
    npm --prefix apps/web test -- --run src/features/admin/users-roles-page.test.tsx
    npm --prefix apps/web run test:e2e:http -- --grep "user lifecycle"

Expected: every visible control performs the exact backend action or is
disabled with a specific reason.

**Acceptance**

- The Admin UI covers the complete server-supported lifecycle.
- No toast-only, fake, or demo-only account success remains in the HTTP profile.

**Implementation and fresh verification**

- The HTTP Admin page renders exact provider, application-profile, desired
  membership, role, organization, invitation, MFA, account, synchronization,
  and last-session facts. Provisioning and every server-supported
  non-provision action use a reasoned confirmation dialog and current revision.
- Directory refresh is monotonic, so an older pending read cannot overwrite a
  later terminal result. Provider authority updates first read and preserve the
  complete current Keycloak user representation before changing the
  server-owned organization attribute and exact role set.
- `npm --prefix apps/web run typecheck` passed with exit 0.
- `npm --prefix apps/web test -- --run
  src/features/admin/users-roles-page.test.tsx` passed 6/6 with exit 0.
- `npm --prefix apps/web run test:e2e:http -- --grep "user lifecycle"` passed
  1/1 in 5.4 seconds inside the isolated canonical HTTP profile; the focused
  outbox drain passed and all task-owned services, storage, Vite, and browser
  processes were cleaned.
- `go -C apps/api test -race -count=1 ./internal/identity` passed with exit 0
  in 1.536 seconds, including complete provider-user preservation. A preceding
  restricted-sandbox attempt could not bind the Go test server's loopback
  listener and exited before assertions; the identical unrestricted rerun is
  the completion evidence.
- The complete Web regression passed 650/650 tests across 64 files in 80.13
  seconds. The identity/data contract passed 26/26.

### Task 6: Build The Out-Of-Process Preprod Data Loader

**Task status:** `complete`; Task 5 is published and remotely confirmed. The
strict RED sequence and the corrected fresh completion gate are recorded.
Task 6 is `verified locally`, published as
`26c7022ebed02c99c7183850f7138ea63580bf0c`, and remotely confirmed.

**Files**

- Create `apps/api/internal/preproddata/`.
- Create `apps/api/cmd/preprod-data-loader/main.go`.
- Create `apps/api/internal/preproddata/profiles/`.
- Create `scripts/load-preprod-data.sh`.
- Create `tests/preprod-data-boundary.test.mjs`.
- Modify `apps/api/Dockerfile` only to add a separately named loader target.
- Modify local Compose only through an explicit one-shot profile.
- Modify isolated Keycloak realm/database initialization, Mailpit namespace,
  object namespace, and retained loader-control-store configuration.

**Work**

- [x] Implement deterministic ID, clock, text, relationship, object-metadata,
  and lifecycle-state generation from a required seed and profile version.
- [x] Generate and persist a canonical content-addressed intent manifest before
  data writes. Include profile, seed hash, expected counts, distribution,
  code/contract digests, exact disposable target identity, and run ID. Call it
  signed only if a separately approved signing key, custody, verifier, and
  detached-signature contract exist.
- [x] Publish a separate append-only run-result manifest with actual counts,
  relationship digests, checkpoints, and failures; publish a third cleanup
  attestation only after a later cleanup. All three live outside the disposable
  target, link by digest, and never rewrite intent into a passing result.
- [x] Require exact `local-preprod` environment identity, an allowlisted
  dedicated disposable database name, and one short-lived, single-use
  operation authorization whose stored value is hashed and whose scope binds
  target,
  run ID, intent digest, exactly one of `LOAD_EMPTY_TARGET`, `RESUME_RUN`, or
  `DROP_RECREATE_TARGET`, issuer, expiry, and nonce. One token cannot authorize
  multiple actions.
- [x] Bind the target fingerprint to environment marker, database name,
  owner, PostgreSQL system identifier, host/port, Compose project, isolated
  Keycloak realm/database identifier and lifecycle service client, Mailpit
  namespace, object bucket/prefix, profile/version, run ID, and intent digest.
  Transport authorization only through an ephemeral `0600` file/secret mount;
  never expose it in CLI arguments, environment dumps, logs, or evidence.
- [x] Keep the loader binary, image target, and Compose service absent from
  normal API/worker/runtime artifacts and normal startup.
- [x] Re-run Task 2's normal-artifact boundary and prove the new loader did not
  reintroduce `internal/testprofile`, `internal/preproddata`, `__test/reset`, or
  loader commands into normal artifacts.
- [x] Use application services or their exact server-owned command boundary so
  revisions, audit events, outbox records, privacy, and authorization invariants
  are preserved.
- [x] Make rerun with the same run ID an exact replay and reject a different
  manifest under the same run ID.
- [x] Produce no credential, secret, real PII, or Evidence bytes. Fixed,
  unmistakably synthetic Internal CAA Note, private-risk, and enforcement
  canaries may exist only in private fields to prove non-leakage; they must
  never enter manifests, Auditee output, generated documents, mail, logs, or
  evidence.
- [x] Never delete append-only rows or individual users by run ID.
  Repeatability uses a clean, loader-exclusive application DB, isolated
  Keycloak realm/database/accounts/client, Mailpit mailbox, object namespace,
  and queues/jobs; whole-namespace drop/recreate requires exact current
  authorization, ownership preflight, and backup/retention decision.
- [x] Retain intent/result/checkpoints, token hashes/consumption, and cleanup
  attestations in the external append-only control store after disposal.

**Verification**

Run:

    node --test tests/preprod-data-boundary.test.mjs
    go -C apps/api test -race -p 1 -count=1 ./internal/preproddata ./cmd/preprod-data-loader
    node apps/web/scripts/assert-http-artifact.mjs apps/web/dist/http
    ./scripts/test-normal-artifact-boundary.sh

Expected: loader surfaces exist only in the explicit one-shot profile and are
absent from the normal HTTP artifact and API route table.

Fresh result:

- `node --test tests/preprod-data-boundary.test.mjs` — exit 0; 4/4 passed.
- `env GOCACHE=/private/tmp/aviasurveil360-task6-final2-go-cache go -C
  apps/api test -race -p 1 -count=1 ./internal/preproddata
  ./cmd/preprod-data-loader` — exit 0; both packages passed
  (`internal/preproddata` 1.576s; command 1.341s).
- `node apps/web/scripts/assert-http-artifact.mjs apps/web/dist/http` — exit
  0; `http-artifact-scan: ok (146 files, 177 inputs)`.
- `./scripts/test-normal-artifact-boundary.sh` — exit 0; both focused
  `internal/httpapi` runs passed and `normal-artifact-boundary: ok`.

**Acceptance**

- Preprod data generation cannot be triggered through the application API.
- Every normal API, worker, scheduler, migration binary/image and its route/
  command table remains free of `internal/testprofile`, loader, and reset code.
- One immutable intent plus its digest-linked run result and later cleanup
  attestation deterministically describe one generated dataset lifecycle.
- Cleanup cannot reinterpret or erase previously accepted history.

### Task 7: Generate Complete Connected Lifecycle Scenarios

**Task status:** `complete — verified locally and published`; Task 6 is
published and remotely confirmed. The strict pre-implementation RED and later
focused RED results are recorded below. The final real-service aggregate and
focused regressions passed with exact cleanup. Task 7 is published as
`a2d474431d8bcc33ebc0557630cd7038cbcd8ec0`; the exact `origin/main` ref is
confirmed.

**Strict RED**

- `env GOCACHE=/private/tmp/aviasurveil360-task7-red-go-cache go -C apps/api
  test -count=1 ./internal/preproddata/scenarios` — exit 1; expected failure:
  `no non-test Go files in .../internal/preproddata/scenarios`, so the
  connected-scenario implementation was absent.
- `env GOCACHE=/private/tmp/aviasurveil360-task7-store-red-go-cache go -C
  apps/api test -tags integration -count=1
  ./internal/preproddata/scenarios -run
  '^TestPostgresStoreMaterializesTheCompleteSmokeDomainAndReconciles$'` — exit
  1; expected compile failure: `undefined: scenarios.NewPostgresStore`, so no
  server-owned PostgreSQL materialization/reconciliation boundary existed.
- `env GOCACHE=/private/tmp/aviasurveil360-task7-connected-boundary-red-go-cache
  go -C apps/api test -count=1 ./internal/preproddata/scenarios -run
  '^TestConnectedBoundaryMaterializesIdentityAndInvitationStateOutsidePostgres$'`
  — exit 1; expected compile failure included
  `undefined: scenarios.ProviderAccount`,
  `undefined: scenarios.InvitationDelivery`, and
  `undefined: scenarios.ObjectVersion`, proving that no connected external
  identity, invitation-delivery, or object boundary existed.
- `env GOCACHE=/private/tmp/aviasurveil360-task7-object-boundary-red-go-cache
  go -C apps/api test -count=1 ./internal/preproddata/scenarios -run
  '^TestConnectedBoundaryWritesSafeDeterministicObjectVersions$'` — exit 1;
  expected assertion failure showed an empty external `ObjectVersion` instead
  of the exact synthetic JSON object, bucket, run prefix, digest, and byte
  count, proving that object-version commands were not connected to an object
  boundary.
- `env GOCACHE=/private/tmp/aviasurveil360-task7-object-stream-red-go-cache go
  -C apps/api test -count=1 ./internal/preproddata/scenarios -run
  '^TestStreamObjectVersionMetadataMatchesTheSafeSyntheticJSON$'` — exit 1;
  expected assertion failure showed generated digest
  `sha256:983ed0ba247f4c296c70e1f3cc6e0acea85e1f3f5727952b13d1d592c54a5d37`
  instead of the safe JSON digest
  `sha256:e987b65b88ba32599eaa230a005e31d952ebd54bac24361ffd22995fb5d6dd2b`,
  proving stream object metadata did not yet represent the actual safe
  synthetic object bytes.
- `env GOCACHE=/private/tmp/aviasurveil360-task7-reconcile-drift-red-go-cache
  go -C apps/api test -count=1 ./internal/preproddata/scenarios -run
  '^TestConnectedBoundaryRejectsExternalIdentityReconciliationDrift$'` — exit
  1; expected assertion failure `external identity drift was accepted`,
  proving reconciliation still returned PostgreSQL evidence without rejecting
  a mismatched external provider account.
- `env
  'AVIA_TEST_DATABASE_URL=postgres://aviasurveil:aviasurveil@127.0.0.1:55432/aviasurveil?sslmode=disable'
  GOCACHE=/private/tmp/aviasurveil360-task7-store-records-red-go-cache go -C
  apps/api test -tags integration -count=1
  ./internal/preproddata/scenarios -run
  '^TestPostgresStoreMaterializesTheCompleteSmokeDomainAndReconciles$'` — exit
  1; expected compile failure `store.Records undefined`, proving the durable
  PostgreSQL scenario store could not yet supply run-scoped records to the
  connected reconciliation boundary.
- `env GOCACHE=/private/tmp/aviasurveil360-task7-keycloak-endpoint-red-go-cache
  go -C apps/api test -count=1 ./internal/preproddata/scenarios -run
  '^TestKeycloakEndpointCreatesExactProviderAccountAndRejectsDrift$'` — exit 1;
  expected compile failures `undefined: scenarios.NewKeycloakEndpoint` and
  `undefined: scenarios.KeycloakEndpointConfig`, proving there was no
  deterministic-subject Keycloak adapter for connected scenario accounts or
  provider-drift reconciliation.
- `env GOCACHE=/private/tmp/aviasurveil360-task7-mailpit-endpoint-red-go-cache
  go -C apps/api test -count=1 ./internal/preproddata/scenarios -run
  '^TestMailpitInvitationEndpointDeliversOnceAndReconcilesRecipient$'` — exit
  1; expected compile failures
  `undefined: scenarios.NewMailpitInvitationEndpoint` and
  `undefined: scenarios.MailpitInvitationEndpointConfig`, proving there was no
  idempotent Keycloak execute-actions/Mailpit recipient-delivery adapter.
- `env GOCACHE=/private/tmp/aviasurveil360-task7-object-endpoint-red-go-cache go
  -C apps/api test -count=1 ./internal/preproddata/scenarios -run
  '^TestObjectEndpointWritesExactlyOnceAndRejectsContentDrift$'` — exit 1;
  expected compile failures for `NewConnectedObjectEndpoint`,
  `ConnectedObjectEndpointConfig`, `ObjectBlob`, and `ObjectBackend`, proving
  there was no create-only, replay-safe, drift-reconciling object endpoint.
- `env GOCACHE=/private/tmp/aviasurveil360-task7-minio-backend-red-go-cache go
  -C apps/api test -tags integration -count=1
  ./internal/preproddata/scenarios -run
  '^TestMinIOObjectBackendPersistsAndReconcilesExactScenarioJSON$'` — exit 1;
  expected compile failures `undefined: scenarios.NewMinIOObjectBackend` and
  `undefined: scenarios.MinIOObjectBackendConfig`, proving the connected object
  contract had no real MinIO/S3 implementation.
- `env GOCACHE=/private/tmp/aviasurveil360-task7-cli-red-go-cache go -C
  apps/api test -count=1 ./cmd/preprod-data-loader -run
  '^TestRunConnectedInvokesTheAuthorizedConnectedRunnerAndPrintsResult$'` —
  exit 1; expected compile failures for `commandDependencies`,
  `runWithDependencies`, `RouteCatalogFile`, and `BehaviorLedgerFile`, proving
  the one-shot loader had no connected-run command or canonical catalog
  binding.
- `env
  GOCACHE=/private/tmp/aviasurveil360-task7-connected-inputs-red-go-cache go -C
  apps/api test -count=1 ./cmd/preprod-data-loader -run
  '^TestLoadConnectedInputsBindsIntentSeedAuthorizationAndCanonicalCatalogs$'`
  — exit 1; expected compile failure `undefined: loadConnectedInputs`, proving
  no pre-I/O binding jointly validated immutable intent, 0600 seed, one-time
  authorization, frozen profile, and canonical route/action catalogs.
- `./scripts/test-preprod-connected-scenarios.sh smoke` — exit 127; expected
  shell failure `no such file or directory:
  ./scripts/test-preprod-connected-scenarios.sh`, proving the required
  whole-namespace load, reconciliation, privacy, and cleanup aggregate did not
  exist before implementation.
- `env GOCACHE=/private/tmp/aviasurveil360-task7-privacy-canary-red-go-cache go
  -C apps/api test -count=1 ./internal/preproddata/scenarios -run
  '^TestPrivacyCanariesReferenceExistingCrossOrganizationRecords$'` — exit 1;
  expected assertion failure reported that the first
  `list/auditee-a-from-b` canary
  `synthetic-privacy-list-0950e5c315fb3356de171255` did not exist, proving the
  privacy matrix initially named decorative canaries rather than retained
  cross-organization records.
- `env GOCACHE=/private/tmp/aviasurveil360-task7-cleanup-command-red-go-cache
  go -C apps/api test -count=1 ./cmd/preprod-data-loader -run
  '^(TestRunRecordCleanupInvokesRecorderAndPrintsExactEvidence|TestRecordCleanupRejectsLoadAuthorizationWithoutConsumingIt|TestRecordCleanupConsumesDropAuthorizationAndAppendsAttestation)$'`
  — exit 1; expected compile failures reported unknown dependency field
  `recordCleanup` and undefined `recordCleanupData`, proving the loader had no
  offline cleanup-attestation command and could not yet enforce a separately
  consumed `DROP_RECREATE_TARGET` authorization after namespace cleanup.
- `node --test --test-name-pattern='cleanup recorder runs offline'
  tests/preprod-data-boundary.test.mjs` — exit 1; expected assertion failure
  captured wrapper exit 1 and usage limited to
  `prepare|verify-authorization|run-connected`, proving post-cleanup evidence
  could not be recorded and that no `--no-deps` path protected the already
  cleaned disposable namespace from accidental dependency restart.
- The first implemented `./scripts/test-preprod-connected-scenarios.sh smoke`
  run exited 1 during the real namespace health gate. Two isolated
  reproductions captured MinIO's exact failure:
  `unsupported condition keys '[s3:prefix]' used for action
  's3:GetBucketLocation'`. Cleanup removed every task-owned container, volume,
  and network after each run; this failed aggregate is not completion
  evidence.
- `node --test --test-name-pattern='preprod object policy scopes only
  ListBucket' tests/preprod-data-boundary.test.mjs` — exit 1; expected
  assertion failure showed one statement combining
  `s3:GetBucketLocation` and `s3:ListBucket`, proving the invalid prefix
  condition was still applied to both actions before the policy fix.
- The next full aggregate rerun passed MinIO, PostgreSQL, Mailpit, and Keycloak
  health, then exited 1 because `preprod-migration` logged
  `AVIA_ENVIRONMENT must be development, test, or production`. Cleanup again
  removed the exact task-owned namespace; this is not completion evidence.
- `node --test --test-name-pattern='preprod migration uses the supported
  database-only config mode' tests/preprod-data-boundary.test.mjs` — exit 1;
  expected assertion failure showed
  `AVIA_ENVIRONMENT: local-preprod` instead of the generic migration
  binary's supported `development` config mode, while its exact disposable
  database target remained separately configured.
- The third full aggregate passed service health, migration, immutable-intent
  preparation, and load-authorization verification, then exited 1 with
  `AUTHORITATIVE_COMMAND_FAILED family=providerAccounts`; exact cleanup
  removed all task-owned resources afterward.
- `env GOCACHE=/private/tmp/aviasurveil360-task7-provider-authority-red-go-cache
  go -C apps/api test -count=1 ./internal/preproddata/scenarios -run
  '^TestEveryProviderAccountHasExactRoleOrganizationAuthority$'` — exit 1;
  expected assertion failure identified smoke account
  `synthetic-provideraccounts-12d03e474037858c42b1d40e` with role `auditee`
  and organization `CAA`, proving the role distribution and organization
  authority used different source indices.
- The fourth full aggregate rerun retained the provider-organization fix but
  still exited 1 with
  `AUTHORITATIVE_COMMAND_FAILED family=providerAccounts`; exact cleanup again
  left zero task-owned resources. An isolated real-Keycloak diagnostic then
  proved the remaining root cause without exposing credentials: user creation
  returned `201 Created`, while a read by the client-supplied synthetic `id`
  returned `404 Not Found`. Keycloak assigns its own provider subject and does
  not accept the scenario record ID as that subject.
- `env GOCACHE=/private/tmp/aviasurveil360-task7-provider-subject-red2-go-cache
  go -C apps/api test -count=1 ./internal/preproddata/scenarios -run
  '^(TestConnectedBoundaryMaterializesIdentityAndInvitationStateOutsidePostgres|TestKeycloakEndpointCreatesExactProviderAccountAndRejectsDrift|TestMailpitInvitationEndpointDeliversOnceAndReconcilesRecipient)$'`
  — exit 1; expected compile failures reported the missing `ScenarioID`,
  the old error-only `EnsureProviderAccount` result, and the incompatible
  identity endpoint interface. This proves the connected boundary could not
  capture a provider-assigned subject, bind it to the deterministic scenario
  account and membership before PostgreSQL materialization, or reuse that
  retained subject for invitation delivery and reconciliation.
- The first focused implementation rerun then compiled and reached invitation
  delivery but failed with `invalid synthetic invitation delivery`. This
  exposed a second provider-subject assumption: Mailpit delivery validation
  still required the provider subject itself to use the deterministic
  `synthetic-*` record-ID shape rather than accepting the bounded subject
  assigned and returned by Keycloak.
- Final acceptance review added a stricter offline-cleanup RED. `node --test
  --test-name-pattern='cleanup recorder runs offline'
  tests/preprod-data-boundary.test.mjs` exited 1 because the captured first
  Docker argument was `compose` instead of `run`. Although `--no-deps`
  prevented service restart, Compose still created empty declared networks and
  volumes around the attestation command. This proves cleanup evidence was not
  yet recorded in a networkless standalone container that leaves the already
  cleaned namespace untouched.

**Files**

- Create scenario builders under `apps/api/internal/preproddata/scenarios/`.
- Create scenario contract tests under
  `apps/api/internal/preproddata/scenarios/`.
- Create `scripts/test-preprod-connected-scenarios.sh` and its whole-namespace
  reconciliation/privacy/cleanup contract.
- Modify the profile manifests from Task 1 only through a new version when
  counts change.

**Work**

- [x] Generate every planning, Finance, GM, Executive, assignment, checklist,
  Potential Finding, Finding, CAP, Evidence, report, communication,
  notification, calendar, closure, reopen, correction, and supersession state.
- [x] Generate cross-organization negative fixtures whose records exist but are
  never visible to the wrong Auditee.
- [x] Generate append-only version histories and preserve exact predecessor,
  effective-time, known-time, actor, organization, and decision-reason links.
- [x] Generate clean, rejected, expired, delayed, retrying, and unavailable
  object-processing metadata without committing unsafe binary fixtures.
- [x] Include offline checkout, causal sync, stale revision, duplicate replay,
  and recovery re-entry scenarios.
- [x] Guarantee that every one of the 86 HTTP routes has meaningful authorized
  data or an intentionally asserted empty/denied state.
- [x] Include requested, invited, activated, suspended, deactivated,
  reactivated, transferred, role-changed, MFA-reset, forced-logout,
  expired-invitation, provider-unavailable, and provider-drift identity cases
  in every profile.
- [x] Require every profile—not only `acceptance`—to satisfy the same scenario,
  eight-role, visible-action, and 86-route coverage manifest.
- [x] Prove the privacy matrix across list, direct ID, search, filter, count,
  pagination, cache, report/PDF, download, notification, calendar, offline
  sync, raw wire, logs, and evidence for at least two Auditee organizations
  plus CAA-private canaries.

**Verification**

Run:

    ./scripts/test-preprod-connected-scenarios.sh smoke

The aggregate loads a clean smoke namespace through the Task 6 operation
boundary, then reconciles Keycloak accounts/required actions/roles/
organizations, desired memberships, sessions, invitation Mailpit deliveries,
PostgreSQL entities/versions/audit/outbox, objects, jobs, route dispositions,
and private/Auditee canaries before scoped whole-namespace cleanup. Expected:
every required state appears, exact manifest counts/digests match, all
foreign-key/lifecycle invariants hold, and no privacy canary crosses its
allowed surface.

Fresh literal result: `./scripts/test-preprod-connected-scenarios.sh smoke`
exited 0 and printed:

    Plan 5 Task 7 connected smoke scenarios: verified locally
    profile=smoke; families=40; routes=86; actions=306; roles=8
    privacy=45/45; target cleanup=verified locally; residue=0

The aggregate also passed the complete scenario and loader command packages,
the 7/7 loader-boundary suite, real PostgreSQL materialization, real Keycloak
provider-assigned subject binding, exact Keycloak roles/organization/required
actions, Mailpit recipient reconciliation, create-only MinIO objects, exact
domain row counts, immutable intent/result/checkpoint evidence, and a separate
consumed `DROP_RECREATE_TARGET` cleanup authorization. The cleanup recorder
ran in a read-only, capability-free, `--network none` standalone container and
did not recreate the cleaned Compose namespace.

**Acceptance**

- The dataset supports full role-based UAT rather than decorative list volume.
- Empty and denied states remain intentional and separately testable.

### Task 8: Qualify Smoke, Acceptance, Realistic, And Stress Profiles

**Task status:** `complete — verified locally and published`; Tasks 1–7 are
complete, published, and remotely confirmed. Task 8 is published as
`9a1b5cadff82a047bc0400acdb39f3fab2fbecab`, and the exact `origin/main`
revision was confirmed. Task 9 remains `not run`.

**Strict RED**

- `node --test tests/preprod-data-scale-contract.test.mjs` exited 1:
  `Could not find 'tests/preprod-data-scale-contract.test.mjs'`.
- `./scripts/test-preprod-data-profile.sh smoke` exited 127:
  `no such file or directory`.
- Later strict REDs proved, in order: the interruption hook was undefined;
  non-empty resume was rejected; streaming relationship and object
  reconciliation were absent; the stress payload helper was absent or wrong;
  the normal API probe Compose service was missing; its entrypoint lacked
  secrets and then failed with `AVIA_OBJECT_STORE_ENDPOINT is required when
  object storage is enabled`; an uppercase run ID was invalid; explicit
  qualification environment propagation was missing; and the first
  functionally successful smoke run reported non-evidentiary zero memory and
  CPU peaks.
- Real connected acceptance then failed with
  `AUTHORITATIVE_COMMAND_FAILED family=assignments` and later
  `AUTHORITATIVE_COMMAND_FAILED family=checklistResponses`. Focused scale
  tests identified duplicate relationship keys at indices 1,000, 20,000, and
  40,000 for both families before the relationship cycles were corrected.
- The owner revision RED package passed 27/30 Node tests but rejected absent
  `1.1.0` metadata and duration enforcement. Focused Go RED failed because
  `QualificationSeconds` was undefined and `stress@1.1.0` was unknown.
- The first local stress revision attempt failed live with
  `TARGET_RECONCILIATION_FAILED`; its focused RED reported `first local stress
  object size = 0, expected 14914`.

No matching implementation preceded its RED. Every failed run directory is
retained as create-only evidence.

**Files**

- Create `scripts/test-preprod-data-profile.sh`.
- Create `tests/preprod-data-scale-contract.test.mjs`.
- Add profile evidence under a new create-only
  `docs/demo-evidence/preprod-data/` run directory during execution.

**Work**

- [x] Load each profile into a clean isolated database and compare actual
  counts, relationship hashes, lifecycle distribution, and audit/event totals
  to its manifest.
- [x] Reconcile the complete namespace: Keycloak accounts, enabled/required-
  action/role/organization state, desired memberships/revisions, invitation
  delivery and Mailpit records, sessions, application rows, objects, audit/
  outbox, scanner/render/notification jobs, and queued/delivered outcomes.
- [x] Measure generation time, database size, object metadata size, query
  plans, top queries, API latency, worker lag, and cleanup time.
- [x] Prove bounded memory and streaming behavior for local qualification
  profiles `realistic@1.1.0` and `stress@1.1.0`.
- [x] Interrupt generation at defined boundaries, resume safely, and prove no
  duplicate authoritative records or conflicting revisions.
- [x] Reset only the exact loader-owned local-preprod dataset and prove no
  cross-database, shared realm, or non-preprod target can be selected. Reset
  means an explicitly authorized drop/recreate of the complete disposable
  application/Keycloak/Mailpit/object/job namespace, not selective deletion
  from append-only tables or a shared identity realm.
- [x] Scan generated text and evidence for forbidden data patterns and secret
  material.
- [x] Fail closed when any profile exceeds its frozen total qualification,
  generation, memory, CPU, disk, object-byte, or cleanup envelope. The
  owner-approved versioned `stress@1.1.0` local qualification remains mandatory;
  resource pressure is not permission to skip it. Full-volume
  `realistic@1.0.0` and `stress@1.0.0` endurance are separate deferred
  release-readiness evidence and remain literally `not run`.

**Verification**

Run:

    node --test tests/preprod-data-scale-contract.test.mjs
    ./scripts/test-preprod-data-profile.sh smoke
    ./scripts/test-preprod-data-profile.sh acceptance
    ./scripts/test-preprod-data-profile.sh realistic
    ./scripts/test-preprod-data-profile.sh stress

Expected: exact manifest reconciliation, zero privacy findings, bounded
resource use, deterministic replay, retained control records, and safe
complete-namespace cleanup for `smoke@1.0.0`, `acceptance@1.0.0`,
`realistic@1.1.0`, and `stress@1.1.0`. The latter two complete within 900 and
1,800 seconds total respectively.

Fresh literal results:

| Profile run | Generation / total seconds | Resource peaks | Exact payload and result |
|---|---:|---|---|
| `run-task8-smoke-20260728-123820-3753` (`smoke@1.0.0`) | 17 / 125 | 1.637 MiB; 0.132 cores; 5 samples | 6,576 object bytes; privacy 0; envelope clean; cleanup 3 s; residue 0 |
| `run-task8-acceptance-20260728-124053-5944` (`acceptance@1.0.0`) | 124 / 219 | 25.59 MiB; 0.542 cores; 58 samples | 2,466,000 object bytes; privacy 0; envelope clean; cleanup 3 s; residue 0 |
| `run-task8-realistic-20260728-124500-8671` (`realistic@1.1.0`) | 228 / 326 of 900 | 25.64 MiB; 0.528 cores; 111 samples | 4,932,000 object bytes; privacy 0; envelope clean; cleanup 4 s; residue 0 |
| `run-task8-stress-20260728-130443-17486` (`stress@1.1.0`) | 528 / 649 of 1,800 | 24.12 MiB; 0.912 cores; 258 samples | exact 536,870,912 object bytes; privacy 0; envelope clean; cleanup 5 s; residue 0 |

Every success directory contains exactly seven bounded files. Each control
record retains the expected failed `COMMAND_STREAM_FAILED` result followed by
one `SUCCEEDED` result and separately consumed load, resume, and cleanup
authorizations.

Final focused verification:

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

**Acceptance**

- Every active local qualification profile is reproducible from source and
  carries literal performance evidence.
- A failed profile is not called ready and cannot become the successor plan's
  accepted input.
- Full-volume `realistic@1.0.0` and `stress@1.0.0` are not required to close
  this local Task 8 gate; they remain retained, deferred, release-readiness
  endurance evidence with status `not run`.

### Task 9: Run The Full Identity And Data Foundation Gate

**Task status:** `complete — verified locally; publication pending`; strict
final-evidence RED and every literal matrix result are recorded. Tasks 1–8 are
complete, published, and remotely confirmed.

**Strict RED**

- `test -f
  docs/demo-evidence/PREPROD_IDENTITY_AND_DATA_FOUNDATION_2026-07-27.md`
  exited 1 before Task 9 execution. The required final evidence artifact did
  not exist, so no full-gate completion claim was possible.
- The first exact `go -C apps/api test -race -p 1 -count=1 ./...` attempt
  compiled and passed all packages through `preproddata/scenarios` but failed
  to build `tests/integration`: eight integration files referenced the
  correctly `canonicaltest`-tagged HTTP boundary from the untagged graph.
  Focused Node RED was 7/8 with the same undefined symbols.
- Adding the matching test-file build tags exposed three shared pure test
  helpers that tags-off tests also consume. The helpers were moved without
  behavior change to an untagged integration helper file. The first tagged
  compile then rejected one newly unused `fmt` import. After removing it,
  `go -C apps/api test -tags canonicaltest -run '^$'
  ./tests/integration` passed in 0.666 seconds and the focused boundary suite
  passed 8/8.
- The second exact full-race attempt compiled the integration package and
  passed `preproddata/scenarios` in 203.363 seconds, then correctly failed its
  live integration tests because the required task-owned PostgreSQL and MinIO
  endpoints were not running (`127.0.0.1:55432` and `127.0.0.1:59001`
  connection refused). An environment-backed exact rerun remains required.
- The environment-backed exact rerun passed every package. The two longest
  packages were `internal/preproddata/scenarios` at 199.981 seconds and
  `tests/integration` at 55.441 seconds. The ad hoc outer shell then returned
  exit 1 after the successful Go process because its cleanup function assigned
  to zsh's read-only `status` parameter. This was not a test failure; the exact
  task-owned Docker project and generated runtime directory were explicitly
  removed immediately afterward, independent Docker queries returned zero
  container/volume/network residue, and protected PID 13055 remained alive.
- The first `./scripts/test-local-full-profile.sh` attempt accepted the
  previously recorded digest/SBOM/scan manifest, created the exact disposable
  project
  `aviasurveil360-task-plan3-full-20260728141438-53080`, and then failed closed
  because its worker was unhealthy. The cleanup trap removed the task-owned
  containers, volumes, networks, and state. A separate task-owned diagnostic
  reproduction preserved the missing logs long enough to prove that the
  accepted `api` and `worker` images were stale: they identified source
  revision `3acedc80c7c4545e715fa694ea1e2fbc6163447d`, while the current
  Compose/entrypoint/config contract uses the Plan 5 least-privilege Keycloak
  service client. The stale API rejected missing
  `AVIA_KEYCLOAK_ADMIN_USERNAME`; the stale worker expected the removed
  bootstrap-admin password mount. The diagnostic project was then fully
  removed. Current-source image build, digest-bound SBOM, HIGH/CRITICAL scan,
  and a clean full-profile rerun are required; the failed runs are not accepted
  evidence.
- The current-source rebuild completed 9/9 images, 9/9 CycloneDX SBOMs, and
  9/9 fail-closed HIGH/CRITICAL scans. The next local-full run proved the
  rebuilt API and worker healthy, then failed before browser evidence because
  the runtime checker still expected Mailpit on `platform` only. The maintained
  Compose policy intentionally requires Mailpit on both `identity` for
  Keycloak delivery and `platform` for worker delivery. A focused 6/7 runtime
  contract RED captured the mismatch; the exact expectation was synchronized
  to `identity platform`, after which the runtime plus Compose-policy suites
  passed 28/28. The failed full-profile project and all scoped state were
  removed.
- The following clean rerun reached Playwright and failed because the bootstrap
  Admin had no authoritative desired membership while the historical browser
  fixture attempted to give one provider account seven CAA roles. The strict
  local-profile contract recorded 11/13 passed before implementation. The
  harness was changed to create the Admin membership through the
  authoritative one-shot identity setup and to exercise eight distinct
  exact-role sessions. The first live implementation attempt then failed
  because `docker compose cp` could not write the setup binary into the
  read-only API root filesystem. The next attempt used an auto-removed
  `compose run --rm --no-deps` container with a read-only bind-mounted binary,
  but failed when both the bootstrap script and the lifecycle attempted to
  create the same Keycloak email. Each failed project completed exact cleanup.
  The final ordering lets the lifecycle create the Admin, then verifies the
  exact CAA/Admin provider authority and configures only the test credential
  and `CONFIGURE_TOTP`.
- The clean final `./scripts/test-local-full-profile.sh` run exited 0. It
  completed one Playwright test in 1.1 minutes, opened eight distinct
  single-role sessions across all 86 routes and ten scenario families,
  reconciled nine required Mailpit messages, survived the worker restart, and
  passed the complete runtime check. Its JSON summary reported
  `expectedDirectLoads=86`, `expectedScenarioFamilies=10`, `tests=1`,
  `skipped=0`, and `status=verified locally`. Cleanup removed project
  `aviasurveil360-task-plan3-full-20260728150244-74596`, all task-owned
  containers, volumes, networks, and scoped state. A post-run Docker query
  returned zero task-owned residue, and protected PID 13055 remained alive.
- The first Task 9 OIDC-profile run passed one-shot identity setup and proved
  the Admin's active exact CAA/Admin provider and application authority, then
  timed out after 360 seconds because its historical E2E selector still
  expected the removed `Submit provisioning` button. The reviewed lifecycle UI
  now requires `Review provisioning` followed by the explicit
  `Confirm Provision` dialog. The RED run's generated-secret scan found zero
  matches and exact Docker cleanup left zero residue. The E2E was synchronized
  to the maintained confirmation boundary; Web typecheck, the focused
  Users/Roles suite (6/6), stale-selector scan, and `git diff --check` passed
  before the clean OIDC rerun. The clean rerun exited 0: one real-browser
  OIDC/MFA/provisioning/session-revocation test passed in 50.4 seconds, the
  generated-secret/log scan found zero matches, and exact task-owned cleanup
  completed.
- The first Task 9 `realistic@1.1.0` rerun materialized and reconciled every
  exact count, relationship, privacy canary, and resume state, but failed
  closed on the owner duration gate: generation took 858 seconds and total
  qualification took 1,085/900 seconds. Cleanup completed in 11 seconds with
  zero residue. Run
  `run-task8-realistic-20260728-154050-89149` remains retained failed evidence.
  Comparison with the accepted Task 8 generation time of 228 seconds exposed
  redundant Keycloak round trips: every synthetic account fetched a new
  service token and the same realm-role representation. The focused cache RED
  reported `Keycloak service-token requests = 3, expected 1`. The endpoint now
  reuses only an unexpired `expires_in`-bounded service token and validated
  realm-role metadata while retaining per-account user and role
  reconciliation. The focused test passed, Node boundary/scale tests passed
  12/12, Go vet passed, `git diff --check` passed, and the complete changed
  scenario package passed under race in 217.252 seconds before the clean
  realistic rerun. That rerun reduced generation to 716 seconds but still
  failed closed at 966/900 qualification seconds; cleanup completed in 17
  seconds with zero residue. Create-only run
  `run-task8-realistic-20260728-160743-98228` is retained. The next focused RED
  reported `Keycloak user reads = 4, expected 2`: newly created users were read
  both before and after role mapping even though only the post-mapping read can
  prove the final state. The implementation now performs one post-mapping exact
  user/role check for a new account and preserves the complete final global
  provider reconciliation. The focused test passed, boundary/scale tests
  passed 12/12, Go vet and `git diff --check` passed, and the scenario package
  passed under race in 198.129 seconds before the next clean rerun. That rerun
  again reconciled every exact count, relationship, privacy canary, and resume
  state, but generation took 740 seconds and total qualification took 999/900
  seconds. Cleanup completed in 16 seconds with zero residue. Create-only run
  `run-task8-realistic-20260728-163103-6332` is retained as failed evidence.
  The final focused RED reported
  `Keycloak user reads after listed reconciliation = 4, expected 2`: the
  complete `briefRepresentation=false` namespace listing was discarded and
  every account was fetched again. Reconciliation now validates the already
  listed complete representation directly while retaining the exact
  per-account realm-role read. The focused test passed, boundary/scale tests
  passed 12/12, Go vet and `git diff --check` passed, and the complete scenario
  package passed under race in 199.467 seconds. The single bounded rerun then
  passed as `run-task8-realistic-20260728-165658-16700`: generation completed
  in 672 seconds and total qualification in 872/900 seconds; seven bounded
  evidence files, 326 resource samples, 26.17 MiB peak loader memory, 1.064
  peak CPU cores, privacy findings 0, every resource envelope, exact
  reconciliation, controlled interruption/resume, and 10-second
  whole-namespace cleanup all passed. Independent Docker inspection found zero
  task-owned residue and PID 13055 remained alive.
- The fresh Task 9 `stress@1.1.0` run
  `run-task8-stress-20260728-171259-25353` completed its authoritative data run
  with outcome `SUCCEEDED`: exact reconciliation, every data family and
  relationship digest, privacy findings 0, controlled interruption/resume,
  resource measurement, and whole-namespace cleanup all passed. It
  nevertheless failed closed on the owner duration gate because generation
  took 1,624 seconds and total qualification took 1,819/1,800 seconds. Cleanup
  completed in 12 seconds with residue 0. The create-only run retains nine
  bounded files, including its failure record; 785 resource samples measured
  28.62 MiB peak loader memory and 0.648 peak CPU cores. Independent Docker
  inspection found zero task-owned residue and PID 13055 remained alive. No
  automatic rerun was started. On 2026-07-28 the owner explicitly approved
  `OWNER-DIRECTIVE-2026-07-28-P5T9-01`, accepting only this exact retained
  run's 19-second qualification overrun for the Task 9 matrix. The exception
  does not change `stress@1.1.0`, its 1,800-second fail-closed envelope, its
  exact volume, or any data, relationship, privacy, resume, resource, or
  cleanup gate. The run continues to record exit 1 and
  `qualification-duration-exceeded` literally; the owner decision makes a
  repeat stress run unnecessary for Task 9.
- After the final Keycloak reconciliation changes, the exact
  `go -C apps/api test -race -p 1 -count=1 ./...` command was rerun in a clean,
  task-owned PostgreSQL/MinIO environment and exited 0. Every package passed;
  `internal/preproddata/scenarios` completed in 147.408 seconds and
  `tests/integration` in 42.686 seconds. The Bash cleanup wrapper exited 0,
  removed its containers, volumes, network, and temporary runtime directory,
  and independent Docker queries found zero task-owned residue. PID 13055
  remained alive.

**Files**

- Create `docs/demo-evidence/PREPROD_IDENTITY_AND_DATA_FOUNDATION_2026-07-27.md`
  during execution.
- Modify `docs/demo-evidence/BUILD_SUMMARY.md`,
  `docs/exec-plans/index.md`, and `docs/exec-plans/tech-debt-tracker.md`.
- Update this plan with literal results.

**Work**

- [x] Run contract generation, SQLC, Go race, React, root/oracle, demo/HTTP
  builds, OIDC, user lifecycle, profile generation, privacy, and residue gates.
- [x] Exercise all eight roles from provider creation through invitation
  delivery, mandatory/role-policy required actions, MFA, first login,
  authorized workspace, suspension, deactivation, reactivation, role/
  organization change, forced logout, and stale-session denial.
- [x] Prove account changes revoke old authority and remain correct after
  Keycloak, API, and worker restart.
- [x] Prove normal/full mode has no test-profile or data-loader route and starts
  clean unless the separate loader is explicitly invoked. Inspect normal
  command dependency graphs and binary symbols, image contents/SBOMs/commands,
  route tables, Compose services/environment, startup logs, and exact
  post-migration per-table baseline counts; separately prove the canonical test
  artifact still works.
- [x] Prove the application uses the least-privilege Keycloak service account,
  rotates it without authority loss, and has no bootstrap-admin credential in
  normal API, worker, scheduler, or loader containers.
- [x] Obtain an independent code and specification review; fix every Critical
  and Important finding before handoff.
- [x] Record exact commands, versions, counts, hashes, timings, skips, blockers,
  and cleanup.

**Verification**

The final matrix includes:

    ./scripts/check-contracts.sh
    ./scripts/check-sqlc.sh
    node --test tests/preprod-identity-data-contract.test.mjs
    node --test tests/preprod-data-boundary.test.mjs
    node --test tests/preprod-data-scale-contract.test.mjs
    go -C apps/api test -race -p 1 -count=1 ./...
    npm --prefix apps/web run typecheck
    npm --prefix apps/web test
    ./scripts/test-preprod-identity-lifecycle.sh all-eight-roles
    node --test tests/*.test.js tests/parity/react-legacy-parity.test.mjs
    npm --prefix apps/web run build:demo
    npm --prefix apps/web run build:http
    ./scripts/test-normal-artifact-boundary.sh
    ./scripts/test-local-full-profile.sh
    ./scripts/test-http-oidc-profile.sh
    ./scripts/test-preprod-data-profile.sh smoke
    ./scripts/test-preprod-data-profile.sh acceptance
    ./scripts/test-preprod-data-profile.sh realistic
    ./scripts/test-preprod-data-profile.sh stress
    git diff --check

Expected: every applicable gate passes or carries an exact owner-approved
exception, task-owned services and output are cleaned, and literal non-claims
remain. The profile commands select the active local qualification versions;
full-volume endurance remains `not run`.

**Acceptance**

- Identity and synthetic data are `verified locally`.
- The result remains `candidate-only` and `release pending`.
- Success satisfies the AviaCore/ML plan's identity/data entry dependency; that
  plan still requires separate execution and per-phase AviaCore authority. It
  does not authorize AWS, production deployment, or production data.

## Risks And Mitigations

| Risk | Mitigation |
|---|---|
| Directory shows only users with sessions | Keycloak-backed paginated directory plus pre-login coverage |
| Admin action leaks provider authority | Server-side role/organization validation and raw-wire denial tests |
| Seed logic becomes a production backdoor | Separate binary/image/profile, environment allowlist, one-run token, no API route |
| Direct fixture writes bypass business truth | Generate through authoritative application commands and verify audit/outbox envelopes |
| Synthetic text resembles real PII | Closed vocabulary, deterministic domains, secret/PII scans, no local data sources |
| High-volume generation exhausts the laptop | Streaming generation, bounded batches, profile resource measurements, safe interruption |
| Profile counts drift silently | Versioned manifest, exact reconciliation, mutation tests |
| Historical records change after role transfer | Preserve record organization/actor identity; transfer only future authority |
| Desired membership and provider roles drift | Revisioned desired state, provider observations, reconciliation, alarms, and fail-closed sessions |
| Bootstrap administrator becomes an application super-credential | One-shot bootstrap only; least-privilege service account and excess-permission tests |
| Cleanup deletes append-only or unrelated data | Dedicated disposable target, exact authorization, preflight, drop/recreate, and retained manifests |

## Idempotence And Recovery

- Contract and discovery checks are read-only and repeatable.
- The loader must reserve one run ID and manifest digest before mutation.
- Same run ID and same digest resumes or reports complete; same run ID and a
  different digest fails.
- A failed run remains inspectable. Cleanup never targets application rows; it
  may drop/recreate only the exact disposable loader-owned application DB,
  isolated Keycloak realm/database/accounts/client, Mailpit mailbox, object
  namespace, and queues/jobs after the authorization and ownership preflight
  passes. Retained control/evidence records survive cleanup.
- Provider lifecycle operations reconcile remote state before retry and use
  one operation/idempotency identity.

## Decisions

- Use four versioned profiles instead of one unbounded dataset.
- Keep account creation Admin-controlled; public registration remains disabled.
- Use real local Keycloak and server-side sessions for acceptance evidence.
- Keep the preprod loader separate from normal runtime and test-profile reset.
- Treat Keycloak provider account, desired application membership, application
  profile, invitation delivery, and session state as separate authorities.
- Use a least-privilege Keycloak service account; never reuse the bootstrap
  administrator in normal runtime.
- Defer AviaCore event delivery to the successor plan so identity/data truth is
  stable before feed qualification.
- Allow Plan 5 Task 1 contract work to proceed in parallel with Plan 1 visual
  stakeholder closure. This sequencing decision applies only to Task 1.
- Record the 11 approved Task 1 owner values only through their distinct
  2026-07-28 approval references and effective contract `1.0.0`; any later
  change requires a new version and cannot become a silent runtime default.
- Record the combined Plans 2–4 stakeholder disposition as completed on
  2026-07-28. A later explicit authorization started Task 2 and the remaining
  Plan 5 sequence.
- Accept `OWNER-DIRECTIVE-2026-07-28-P5T9-01` only for the retained Task 9
  stress run `run-task8-stress-20260728-171259-25353`. It accepts that run's
  exact 19-second qualification overrun and does not revise
  `stress@1.1.0`, its 1,800-second fail-closed envelope, any expected count,
  or any relationship, privacy, resume, resource, or cleanup gate.

## Discoveries

- The current canonical profile has nine principals covering eight roles, but
  it is intentionally test-only.
- Before Task 2, the access directory derived roles from active session rows
  and hard-coded email, MFA, invitation, and account status as not configured.
- Backend lifecycle services already support provision, role update, suspend,
  reactivate, Keycloak reconciliation, session invalidation, and outbox/audit
  recording. The UI exposes only a subset.
- The normal Keycloak realm disables self-registration and imports no
  application users.
- Task 2 proved artifact-level separation: normal API, worker, scheduler, and
  migration dependency and string scans exclude canonical test/reset code,
  while a tagged canonical-test API remains available.
- Task 3 added deactivation, organization transfer, recovery, invitation, and
  exact-revision lifecycle behavior. Task 2 replaced bootstrap-admin password
  use in API/worker with the exact least-privilege confidential service client.
- Task 4 binds every application session to the current desired-membership
  revision and fresh exact provider authority. A post-activation clock refresh
  is required because the activation revision can become effective after
  session creation begins.
- Production `Secure` `__Host-` cookies work in the isolated HTTP browser gate
  only when the loopback origin is `localhost`; Chromium correctly rejects
  those cookies on a plain `127.0.0.1` origin.
- The core OIDC verifier refreshes cached JWKS on a new key ID, rejects expired
  tokens, accepts `nbf` inside its five-minute skew tolerance, and rejects
  tokens beyond that boundary.
- Early command attempts observed transient `node` unavailability. The final
  fresh literal commands ran successfully: 26/26 contract tests and the
  harness-docs smoke check passed.

## Outcome Notes

Task 1 produced product contract `1.0.0`, exact four-profile manifests, 11
approved owner decisions, and the machine-readable mutation test. Tasks 2–5
are `verified locally`: they establish normal/canonical-test artifact
separation, append-only desired membership, least-privilege Keycloak client
credentials, a provider-backed Admin directory, exact-revision lifecycle
actions, live invitation/MFA behavior, fail-closed session authority, and the
complete reason-confirmed Users and Roles Admin experience. Task 6 is also
`verified locally`: its isolated out-of-process loader boundary produces
deterministic intent data and append-only digest-linked lifecycle records
without entering normal artifacts. Task 7 is `verified locally`: its
server-owned connected command boundary materializes all 40 smoke families
through PostgreSQL, provider-assigned Keycloak subjects, Mailpit, and MinIO;
reconciles all 86 routes, 306 actions, eight roles, and 45 privacy canaries;
and proves separately authorized zero-residue cleanup. Task 8 is `verified
locally`: all four active profiles passed complete relationship, privacy,
interruption/resume, resource, duration, and cleanup gates. Owner-approved
`realistic@1.1.0` completed in 326/900 seconds and `stress@1.1.0` in
649/1,800 seconds. Their full-volume `1.0.0` endurance counterparts remain
retained deferred release-readiness evidence with status `not run`. Task 9 is
`verified locally`: the full contract, SQLC, Go race, React/root, demo/HTTP,
normal-artifact, all-eight-role lifecycle, local-full, OIDC, and four-profile
matrix completed with zero task-owned residue. The fresh realistic run passed
at 872/900 seconds. The fresh stress run passed exact data, relationship,
privacy, resume, resource, and cleanup gates but recorded 1,819/1,800 seconds;
the owner accepted only that exact 19-second Task 9 overrun through
`OWNER-DIRECTIVE-2026-07-28-P5T9-01`. Task 9 publication is pending.
Deployment and production readiness remain `not run`; the artifact remains
`candidate-only`, and release remains `release pending`. Task 8 was published
as `9a1b5cadff82a047bc0400acdb39f3fab2fbecab`, and the exact `origin/main`
revision was confirmed.

## Execution Prompt

```text
Plan 5 Tasks 1-9 are complete and verified locally. Tasks 1-8 are published and remotely confirmed; Task 9 publication metadata remains pending. Preserve every retained create-only run, the literal 1,819/1,800-second stress result, exact-run owner directive OWNER-DIRECTIVE-2026-07-28-P5T9-01, and the unchanged stress@1.1.0 fail-closed contract. Full-volume realistic@1.0.0 and stress@1.0.0 endurance remains not run. Do not restart Plan 5 work or claim deployment, release, or production readiness.

Before Plan 6 executable work, publish and remotely confirm the focused Plan 5 Task 9 commit, record its exact hash in this completed plan, the final evidence, completed index, active index, tracker, manifest, and build summary, and push that metadata commit. Then read the complete Plan 6 authority gates and begin only its first authorized task. Preserve the root demo, normal HTTP no-seed boundary, Keycloak authority, organization isolation, append-only histories, training_allowed: false, production_ml_readiness: NOT_READY, and unrelated worktree changes.
```
