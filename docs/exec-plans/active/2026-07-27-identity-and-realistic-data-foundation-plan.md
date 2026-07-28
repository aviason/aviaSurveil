# Identity And Realistic Data Foundation Plan

**Status:** `active — Task 1 authorized`

**Reviewed task count:** 9

**Objective:** Complete the production-intended Keycloak-backed account
lifecycle and create a deterministic, privacy-safe, high-volume preprod data
loader that exercises AviaSurveil360 through authoritative application
boundaries.

**User-visible outcome:** Administrators can see and operate the real account
directory, every supported role can complete its intended first-login and
account-lifecycle path, and the local preprod profile can be populated with
repeatable connected data covering every important workflow state.

**Execution dependency:** On 2026-07-27 the user authorized only Task 1
discovery, contract, and owner-decision packaging without waiting for the Plan
1 visual stakeholder closure. Plans 1 and 5 Task 1 may proceed in parallel.
Tasks 2-9 remain unauthorized and retain the combined Plans 1-4 stakeholder
disposition, completed Task 1 acceptance, and separate current task
authorization gates. This plan is the predecessor of the AviaCore/ML readiness
and local preprod release-candidate plans.

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
engineering profiles are approved as contract `1.0.0` targets:

| Profile | Organizations | Users | Audits | Checklist responses | Findings | CAP revisions | Evidence refs/versions | Report versions | Communications and notifications | Audit events |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `smoke` | 3 | 9 | 2 | 24 | 8 | 12 | 16 | 6 | 40 | 250 |
| `acceptance` | 25 | 250 | 1,000 | 10,000 | 3,000 | 4,500 | 6,000 | 2,000 | 20,000 | 100,000 |
| `realistic` | 100 | 2,000 | 20,000 | 250,000 | 60,000 | 100,000 | 200,000 | 75,000 | 1,000,000 | 5,000,000 |
| `stress` | 200 | 4,000 | 40,000 | 500,000 | 120,000 | 200,000 | 400,000 | 150,000 | 2,000,000 | 10,000,000 |

These are synthetic engineering targets, not forecasts, runtime feasibility
claims, or regulatory targets. The approved disk ceilings are 2 GiB, 20 GiB,
50 GiB, and 64 GiB respectively. Stress is bounded to 12 GiB memory, an
8-hour duration, and an 8 GiB object payload. Any later owner-approved workload
revision must version the entire profile and retain earlier manifests.

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
  closure. Tasks 2-9 remain unauthorized.
- [x] (2026-07-27) Created the English product contract and machine-readable
  mutation test. The final literal contract run passed 26/26 and the literal
  harness-docs smoke command reported `harness-docs-smoke: ok`.
- [x] (2026-07-28) Obtained all 11 named owner decisions, recorded separate
  `OWNER-DIRECTIVE-2026-07-28-P5T1-*` approval references and exact effective
  values in contract/profile version `1.0.0`, and aligned the optional-TOTP and
  laptop-bounded profile decisions. Runtime implementation remains `not run`.
- [x] (2026-07-28) Committed the complete Task 1 contract, profile, product
  index, and 26/26 mutation gate as `d0f5b29` (`docs(plan5): freeze identity
  and data foundation contract`). Tasks 2-9 remain unauthorized and `not run`.

## Tasks

### Task 1: Freeze Identity And Data-Profile Acceptance Contracts

**Task status:** `complete`; the versioned contract, all named owner decisions,
and machine-checkable acceptance gates are closed. API, Keycloak, session,
directory, loader, and profile runtime implementation remains `not run`, and
Tasks 2-9 remain unauthorized.

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

**Task 2 gate:** blocked. Task 2 requires the deferred combined Plans 1-4
stakeholder disposition and separate explicit Task 2 authorization. Plan 1
stakeholder sign-off remains open; the Task 1-only sequencing waiver does not
waive that gate.

### Task 2: Replace Session-Derived Directory Placeholders With Keycloak State

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

- [ ] First split and verify normal versus canonical-test API composition.
  Tasks 2-5 cannot pass while any normal API/worker/scheduler/migration binary
  or image transitively links `internal/testprofile`, canonical reset, or
  loader code. Preserve a separately named positive canonical-test artifact.
- [ ] Persist append-only desired-membership versions and a current
  desired/observed synchronization record before exposing the directory.
  Expose the revision for Task 4 session enforcement.
- [ ] Extend the Keycloak admin adapter with paginated, bounded, timeout-aware
  directory reads and a minimal account-state projection.
- [ ] Join provider identity state to AviaSurveil profile, organization,
  revisioned desired membership, lifecycle-request, invitation-delivery, and
  active-session summaries without copying credentials or required-action
  secrets.
- [ ] Return real email, provider enabled state, desired membership state,
  derived invitation/required-action state, MFA enrollment boolean, roles,
  organization, membership revision/drift, and last successful application
  session time. Do not present invitation as a native Keycloak state when it is
  derived from application lifecycle and delivery records.
- [ ] Replace bootstrap-admin password grant use with a confidential
  least-privilege service account using client credentials and exact realm
  management permissions. Prove the bootstrap credential is unavailable to
  normal runtime and that excess permissions fail policy tests.
- [ ] Distinguish `provider-unavailable` from an empty directory. Never replace
  an unavailable provider with stale success or `Not configured in demo`.
- [ ] Enforce exact Admin authorization, bounded pagination, search, role,
  organization, and account-status filters.
- [ ] Return exactly one account row per subject with `roles[]`, deterministic
  sort/tie-breaker, stable opaque cursor, snapshot/consistency semantics,
  maximum page size/provider calls, and no unbounded N+1 Keycloak queries.
- [ ] Add raw-wire tests proving Auditee and non-Admin denial and proving that
  credentials, TOTP material, provider tokens, and Internal CAA data cannot
  appear.

**Verification**

Run:

    ./scripts/check-contracts.sh
    ./scripts/test-normal-artifact-boundary.sh
    go -C apps/api test -race -p 1 -count=1 ./internal/identity ./internal/administration ./internal/httpapi ./tests/integration
    npm --prefix apps/web test -- --run src/backend/http-backend.test.ts

Expected: real provider states appear for provisioned users, pre-login users
appear without requiring a session row, and provider failure is explicit.

**Acceptance**

- The directory is account-based rather than active-session-based.
- Every exposed field has one named authority and a fail-closed unavailable
  state.
- Provider drift is visible and cannot silently overwrite desired membership.

### Task 3: Complete Provisioning, Invitation, And Recovery

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

- [ ] Preserve `PROVISION`, `UPDATE_ROLES`, `SUSPEND`, and `REACTIVATE`; add
  the Task 1-approved `DEACTIVATE`, `TRANSFER_ORGANIZATION`, invitation resend,
  required-password reset, MFA reset, and forced-logout actions with
  reason-required audit records and exact OpenAPI enums.
- [ ] Require `expectedMembershipRevision` on every account/membership
  authority mutation; provisioning requires no existing revision. A stale
  revision returns a typed conflict and performs no provider, delivery,
  session, audit-success, or outbox-success mutation.
- [ ] Make provisioning produce only the approved one-time, expiring Keycloak
  execute-actions invitation path without generating, returning, or storing an
  application temporary password.
- [ ] Implement only the exact Task 1-approved local invitation contract:
  bounded Keycloak execute-actions delivery through authenticated local mail;
  require `UPDATE_PASSWORD` and add the role-policy-selected `VERIFY_EMAIL`
  and/or `CONFIGURE_TOTP`; freeze TTL, resend, already-used, expiry, lockout,
  recovery, MFA reset/re-enrollment, and all-eight-role behavior.
- [ ] Record invitation issue, delivery, expiry, resend, activation,
  cancellation, and recovery as separate application facts. Never store a
  credential or action token; provider and delivery acknowledgements remain
  reconcilable after timeout.
- [ ] Require `CONFIGURE_TOTP` for internal roles and for Auditee accounts when
  the approved policy says so; store only provider-derived enrollment status.
- [ ] Make duplicate email, duplicate subject, role drift, unknown
  organization, expired invitation, and stale lifecycle request deterministic
  typed outcomes.
- [ ] Preserve idempotency through provider timeout and lost acknowledgement.
  Reconcile a provider-created user before retrying creation.
- [ ] Classify retryable, permanent, and manual-review provider outcomes; use
  approved exponential backoff and attempt/time caps, terminal
  `FAILED_PERMANENT`/`MANUAL_REVIEW` states, operator reconciliation, metrics,
  and alerts. Never retry every provider failure indefinitely.
- [ ] Audit request, provider result, subject binding, role/organization
  binding, session invalidation, invitation delivery, and failure without
  recording secret material.
- [ ] Make deactivation distinct from suspension and deletion: revoke
  sessions, disable future authority, retain the minimum identity/membership
  tombstone under the approved retention policy, and require a new audited
  reactivation decision.

**Verification**

Run the focused Keycloak and lifecycle integration tests under the canonical
service harness, then:

    ./scripts/check-sqlc.sh
    ./scripts/check-contracts.sh
    go -C apps/api test -race -p 1 -count=1 ./internal/administration ./internal/identity ./tests/integration
    ./scripts/test-preprod-identity-lifecycle.sh invitation

Expected: every action is idempotent, reason-audited, provider-reconciled, and
secret-free.

**Acceptance**

- A new user can move from requested through invited, activated/first login,
  MFA, active, suspended, deactivated, and explicitly reactivated states.
- Failed delivery/provider operations are visible and deterministically
  classified as bounded-retryable, permanent, or manual-review; permanent
  failures are never automatically retried.

### Task 4: Enforce Role, Organization, MFA, And Session Lifecycle

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

- [ ] Enforce allowed role sets and prohibit CAA/Auditee organization drift at
  request, provider, callback, session, and projection boundaries.
- [ ] Revoke all active sessions after role change, organization transfer,
  suspend, deactivate, MFA reset, or forced logout.
- [ ] Require a fresh OIDC login after authority changes and reject stale role
  or organization claims.
- [ ] Persist the desired-membership revision in every application session and
  fail closed unless current desired membership, observed provider authority,
  token claims, and the session revision agree.
- [ ] Fail both new and existing sessions on provider/desired-membership drift,
  partial multi-call role replacement, old membership revision, or provider
  unavailability. Enforce the Task 1 maximum observation age and deny an
  already-active session within the frozen deadline after provider disablement,
  authority drift, stale observation, or provider loss. Reconciliation may
  restore authority only after fresh exact desired and observed state agree.
- [ ] Add controlled organization transfer only as a separately reasoned
  `TRANSFER_ORGANIZATION` lifecycle action with expected membership revision;
  never mutate historical record ownership.
- [ ] Define and test bootstrap Admin, application Admin, and break-glass
  identities as separate authorities. Break-glass use must be alarmed and
  audited.
- [ ] Exercise expiry, lockout, TOTP failure, disabled user, revoked session,
  provider restart, signing-key rotation, and clock-skew boundaries.

**Verification**

Run:

    ./scripts/check-sqlc.sh
    go -C apps/api test -race -p 1 -count=1 ./internal/platform/session ./internal/identity ./internal/administration ./internal/httpapi ./tests/integration
    ./scripts/test-http-oidc-profile.sh
    ./scripts/test-preprod-identity-lifecycle.sh session-authority

Expected: all stale and invalid authority paths fail closed and every valid
first-login path reaches only its authorized workspace.

**Acceptance**

- No account can retain old authority after a lifecycle change.
- MFA and account state are enforced by the provider, not simulated by the UI.

### Task 5: Finish The Users And Roles Admin Experience

**Files**

- Modify `apps/web/src/features/admin/users-roles-page.tsx`.
- Modify `apps/web/src/features/admin/users-roles-page.test.tsx`.
- Modify shared Admin styles only where existing patterns require it.
- Modify `apps/web/src/backend/backend.ts` and HTTP transport mappings as
  required by Tasks 2-4.
- Add focused Playwright coverage under `apps/web/tests/e2e/`.

**Work**

- [ ] Render provider, application profile, role, organization, invitation,
  MFA, account, and session state without placeholder values.
- [ ] Add working provision, resend invitation, update role, transfer
  organization, suspend, deactivate, reactivate, MFA reset, and force logout
  flows.
- [ ] Require confirmation and reason entry for authority-changing or
  destructive actions.
- [ ] Disable actions with an exact reason when provider state or actor
  authority makes them unavailable.
- [ ] Provide stable loading, empty, unavailable, pending, succeeded, failed,
  stale, and retry states.
- [ ] Verify keyboard, focus, error summary, mobile layout, and no-overlap
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

### Task 6: Build The Out-Of-Process Preprod Data Loader

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

- [ ] Implement deterministic ID, clock, text, relationship, object-metadata,
  and lifecycle-state generation from a required seed and profile version.
- [ ] Generate and persist a canonical content-addressed intent manifest before
  data writes. Include profile, seed hash, expected counts, distribution,
  code/contract digests, exact disposable target identity, and run ID. Call it
  signed only if a separately approved signing key, custody, verifier, and
  detached-signature contract exist.
- [ ] Publish a separate append-only run-result manifest with actual counts,
  relationship digests, checkpoints, and failures; publish a third cleanup
  attestation only after a later cleanup. All three live outside the disposable
  target, link by digest, and never rewrite intent into a passing result.
- [ ] Require exact `local-preprod` environment identity, an allowlisted
  dedicated disposable database name, and one short-lived, single-use
  operation authorization whose stored value is hashed and whose scope binds
  target,
  run ID, intent digest, exactly one of `LOAD_EMPTY_TARGET`, `RESUME_RUN`, or
  `DROP_RECREATE_TARGET`, issuer, expiry, and nonce. One token cannot authorize
  multiple actions.
- [ ] Bind the target fingerprint to environment marker, database name,
  owner, PostgreSQL system identifier, host/port, Compose project, isolated
  Keycloak realm/database identifier and lifecycle service client, Mailpit
  namespace, object bucket/prefix, profile/version, run ID, and intent digest.
  Transport authorization only through an ephemeral `0600` file/secret mount;
  never expose it in CLI arguments, environment dumps, logs, or evidence.
- [ ] Keep the loader binary, image target, and Compose service absent from
  normal API/worker/runtime artifacts and normal startup.
- [ ] Re-run Task 2's normal-artifact boundary and prove the new loader did not
  reintroduce `internal/testprofile`, `internal/preproddata`, `__test/reset`, or
  loader commands into normal artifacts.
- [ ] Use application services or their exact server-owned command boundary so
  revisions, audit events, outbox records, privacy, and authorization invariants
  are preserved.
- [ ] Make rerun with the same run ID an exact replay and reject a different
  manifest under the same run ID.
- [ ] Produce no credential, secret, real PII, or Evidence bytes. Fixed,
  unmistakably synthetic Internal CAA Note, private-risk, and enforcement
  canaries may exist only in private fields to prove non-leakage; they must
  never enter manifests, Auditee output, generated documents, mail, logs, or
  evidence.
- [ ] Never delete append-only rows or individual users by run ID.
  Repeatability uses a clean, loader-exclusive application DB, isolated
  Keycloak realm/database/accounts/client, Mailpit mailbox, object namespace,
  and queues/jobs; whole-namespace drop/recreate requires exact current
  authorization, ownership preflight, and backup/retention decision.
- [ ] Retain intent/result/checkpoints, token hashes/consumption, and cleanup
  attestations in the external append-only control store after disposal.

**Verification**

Run:

    node --test tests/preprod-data-boundary.test.mjs
    go -C apps/api test -race -p 1 -count=1 ./internal/preproddata ./cmd/preprod-data-loader
    node apps/web/scripts/assert-http-artifact.mjs apps/web/dist/http
    ./scripts/test-normal-artifact-boundary.sh

Expected: loader surfaces exist only in the explicit one-shot profile and are
absent from the normal HTTP artifact and API route table.

**Acceptance**

- Preprod data generation cannot be triggered through the application API.
- Every normal API, worker, scheduler, migration binary/image and its route/
  command table remains free of `internal/testprofile`, loader, and reset code.
- One immutable intent plus its digest-linked run result and later cleanup
  attestation deterministically describe one generated dataset lifecycle.
- Cleanup cannot reinterpret or erase previously accepted history.

### Task 7: Generate Complete Connected Lifecycle Scenarios

**Files**

- Create scenario builders under `apps/api/internal/preproddata/scenarios/`.
- Create scenario contract tests under
  `apps/api/internal/preproddata/scenarios/`.
- Create `scripts/test-preprod-connected-scenarios.sh` and its whole-namespace
  reconciliation/privacy/cleanup contract.
- Modify the profile manifests from Task 1 only through a new version when
  counts change.

**Work**

- [ ] Generate every planning, Finance, GM, Executive, assignment, checklist,
  Potential Finding, Finding, CAP, Evidence, report, communication,
  notification, calendar, closure, reopen, correction, and supersession state.
- [ ] Generate cross-organization negative fixtures whose records exist but are
  never visible to the wrong Auditee.
- [ ] Generate append-only version histories and preserve exact predecessor,
  effective-time, known-time, actor, organization, and decision-reason links.
- [ ] Generate clean, rejected, expired, delayed, retrying, and unavailable
  object-processing metadata without committing unsafe binary fixtures.
- [ ] Include offline checkout, causal sync, stale revision, duplicate replay,
  and recovery re-entry scenarios.
- [ ] Guarantee that every one of the 86 HTTP routes has meaningful authorized
  data or an intentionally asserted empty/denied state.
- [ ] Include requested, invited, activated, suspended, deactivated,
  reactivated, transferred, role-changed, MFA-reset, forced-logout,
  expired-invitation, provider-unavailable, and provider-drift identity cases
  in every profile.
- [ ] Require every profile—not only `acceptance`—to satisfy the same scenario,
  eight-role, visible-action, and 86-route coverage manifest.
- [ ] Prove the privacy matrix across list, direct ID, search, filter, count,
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

**Acceptance**

- The dataset supports full role-based UAT rather than decorative list volume.
- Empty and denied states remain intentional and separately testable.

### Task 8: Qualify Smoke, Acceptance, Realistic, And Stress Profiles

**Files**

- Create `scripts/test-preprod-data-profile.sh`.
- Create `tests/preprod-data-scale-contract.test.mjs`.
- Add profile evidence under a new create-only
  `docs/demo-evidence/preprod-data/` run directory during execution.

**Work**

- [ ] Load each profile into a clean isolated database and compare actual
  counts, relationship hashes, lifecycle distribution, and audit/event totals
  to its manifest.
- [ ] Reconcile the complete namespace: Keycloak accounts, enabled/required-
  action/role/organization state, desired memberships/revisions, invitation
  delivery and Mailpit records, sessions, application rows, objects, audit/
  outbox, scanner/render/notification jobs, and queued/delivered outcomes.
- [ ] Measure generation time, database size, object metadata size, query
  plans, top queries, API latency, worker lag, and cleanup time.
- [ ] Prove bounded memory and streaming behavior for `realistic` and `stress`.
- [ ] Interrupt generation at defined boundaries, resume safely, and prove no
  duplicate authoritative records or conflicting revisions.
- [ ] Reset only the exact loader-owned local-preprod dataset and prove no
  cross-database, shared realm, or non-preprod target can be selected. Reset
  means an explicitly authorized drop/recreate of the complete disposable
  application/Keycloak/Mailpit/object/job namespace, not selective deletion
  from append-only tables or a shared identity realm.
- [ ] Scan generated text and evidence for forbidden data patterns and secret
  material.
- [ ] Fail closed when any profile exceeds its frozen duration, memory, CPU,
  disk, object-byte, or cleanup envelope. An owner-approved versioned profile
  revision is required; resource pressure is not permission to skip `stress`.

**Verification**

Run:

    node --test tests/preprod-data-scale-contract.test.mjs
    ./scripts/test-preprod-data-profile.sh smoke
    ./scripts/test-preprod-data-profile.sh acceptance
    ./scripts/test-preprod-data-profile.sh realistic
    ./scripts/test-preprod-data-profile.sh stress

Expected: exact manifest reconciliation, zero privacy findings, bounded
resource use, deterministic replay, retained control records, and safe
complete-namespace cleanup for all four profiles.

**Acceptance**

- Every profile is reproducible from source and carries literal performance
  evidence.
- A failed profile is not called ready and cannot become the successor plan's
  accepted input.

### Task 9: Run The Full Identity And Data Foundation Gate

**Files**

- Create `docs/demo-evidence/PREPROD_IDENTITY_AND_DATA_FOUNDATION_2026-07-27.md`
  during execution.
- Modify `docs/demo-evidence/BUILD_SUMMARY.md`,
  `docs/exec-plans/index.md`, and `docs/exec-plans/tech-debt-tracker.md`.
- Update this plan with literal results.

**Work**

- [ ] Run contract generation, SQLC, Go race, React, root/oracle, demo/HTTP
  builds, OIDC, user lifecycle, profile generation, privacy, and residue gates.
- [ ] Exercise all eight roles from provider creation through invitation
  delivery, mandatory/role-policy required actions, MFA, first login,
  authorized workspace, suspension, deactivation, reactivation, role/
  organization change, forced logout, and stale-session denial.
- [ ] Prove account changes revoke old authority and remain correct after
  Keycloak, API, and worker restart.
- [ ] Prove normal/full mode has no test-profile or data-loader route and starts
  clean unless the separate loader is explicitly invoked. Inspect normal
  command dependency graphs and binary symbols, image contents/SBOMs/commands,
  route tables, Compose services/environment, startup logs, and exact
  post-migration per-table baseline counts; separately prove the canonical test
  artifact still works.
- [ ] Prove the application uses the least-privilege Keycloak service account,
  rotates it without authority loss, and has no bootstrap-admin credential in
  normal API, worker, scheduler, or loader containers.
- [ ] Obtain an independent code and specification review; fix every Critical
  and Important finding before handoff.
- [ ] Record exact commands, versions, counts, hashes, timings, skips, blockers,
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

Expected: every applicable gate passes, task-owned services and output are
cleaned, and literal non-claims remain.

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

## Discoveries

- The current canonical profile has nine principals covering eight roles, but
  it is intentionally test-only.
- The current access directory derives roles from active session rows and
  hard-codes email, MFA, invitation, and account status as not configured.
- Backend lifecycle services already support provision, role update, suspend,
  reactivate, Keycloak reconciliation, session invalidation, and outbox/audit
  recording. The UI exposes only a subset.
- The normal Keycloak realm disables self-registration and imports no
  application users.
- The current normal API source still links the canonical test-profile/reset
  package even though the route is profile-gated; artifact-level separation is
  therefore an implementation prerequisite, not established evidence.
- Existing lifecycle enums omit deactivation and organization transfer, and
  the current Keycloak client uses bootstrap-admin password credentials.
- Early command attempts observed transient `node` unavailability. The final
  fresh literal commands ran successfully: 26/26 contract tests and the
  harness-docs smoke check passed.

## Outcome Notes

Task 1 produced product contract `1.0.0`, exact four-profile manifests, 11
approved owner decisions, and the machine-readable mutation test. API,
Keycloak, migration, SQLC, frontend, loader, and runtime implementation;
runtime verification; external identity writes; Git publication; and
deployment are `not run`.

## Execution Prompt

```text
Task 1 in docs/exec-plans/active/2026-07-27-identity-and-realistic-data-foundation-plan.md is complete at contract/profile version 1.0.0. Do not start Task 2 until the deferred combined Plans 1-4 stakeholder disposition is recorded and Task 2 receives separate current authorization. Plan 1 visual stakeholder sign-off remains open. Read AGENTS.md, docs/PLANS.md, the plan index, the complete plan, and the current identity/full-profile evidence first. Preserve the root demo, normal HTTP no-seed boundary, Keycloak authority, organization isolation, append-only histories, and unrelated worktree changes.

Use strict RED -> GREEN -> focused review for each task. Establish the desired-membership revision and least-privilege Keycloak service identity before directory/lifecycle acceptance. The normal API/worker/scheduler/migration images must not link testprofile or loader/reset code. Repeatable loader reset uses only an exactly authorized disposable target and never selectively deletes append-only rows. Qualify smoke, acceptance, realistic, and stress with the same scenario/role/route catalog. Do not add public registration, a normal API reset/seed route, real PII, plaintext credentials, client-authored roles, direct fixture writes that bypass authoritative domain behavior, AWS actions, commits, or pushes without separate authorization. Keep the plan, index, tracker, and evidence synchronized with literal results. Stop after each task's acceptance gate and fix all Critical or Important review findings before continuing.
```
