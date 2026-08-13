# AGA Browser QA Remediation ExecPlan

## Objective

Remediate findings F-001 through F-024 in
docs/demo-evidence/AGA_BROWSER_QA_REVIEW_2026-08-06.md so the local-preprod
AGA candidate supports a truthful, repeatable Manager -> Inspector -> Lead
Inspector/CAA Reviewer -> Auditee lifecycle without privacy, authority,
audit-integrity, transaction, or routing regressions.

The result remains candidate-only and release pending. This plan cannot
establish production-ready status.

## Status

- Plan status: `paused` — the 2026-08-07 canonical AGA preprod successor is the
  sole implementation direction.
- Preserve the current dirty remediation work and historical evidence. The
  successor Gate 0 must map every open finding and classify each change before
  editing; do not discard or continue synthetic-only lifecycle work here.

## User-visible outcome

A freshly prepared local AGA instance lands each authenticated role on a
supported AGA route, exposes only exact authorized operations, presents one
server-bound synthetic inspection, supports classification through closure
with durable reasons and exact record identity, allows reliable local account
switching, and renders truthful unavailable/error states instead of generic
404s, stale projections, or fake controls.

## Scope

- AGA OpenAPI and generated transport contracts.
- Workspace authorization, generation/reset, scope resolution, pagination,
  idempotency, transaction, projection, and lifecycle logic.
- Local OIDC logout and role-switch behavior.
- AGA React routing, controls, cache, stale-state handling, and role
  projections.
- Disposable local-preprod preparation and connected qualification.
- Focused unit, contract, integration, browser, privacy, concurrency,
  failure-injection, and artifact-boundary verification.
- Plan, index, evidence, and technical-debt synchronization.

## Explicit exclusions

- Production deployment, hosted environments, release approval, and any
  production-ready claim.
- External/production IdP certification, tenant configuration, secrets, or
  interoperability claims beyond the then-current connected local OIDC run.
- Real regulatory ingestion, operator data, enforcement decisions, or legal
  advice.
- Changes to the root legacy HTML/CSS/JavaScript oracle.
- Canonical normal-runtime data or grants except shared local OIDC/session code
  required by the visible logout contract.
- Browser persistence for question bodies, idempotency receipts, or private
  role projections.
- Compatibility adapters for obsolete candidate contracts.
- Branch, commit, push, deployment, or external-system changes.

## Dependencies and authorization gates

- The current user message explicitly authorizes local candidate code changes for
  this plan; it does not authorize commits, pushes, production/deployment
  actions, or external-system changes.
- The accepted predecessor AGA contracts remain authoritative unless this plan
  names a corrective successor change. Governed AGA Task 9, source/technical
  validation, publication, release, and production readiness do not advance.
- Connected qualification is a separately invoked disposable fixture-owner
  operation. Manual start/status/stop must not silently perform qualification
  or make readiness decisions.
- Stakeholder/user sign-off remains required before moving this plan or either
  predecessor to completed.

## Assumptions and ownership boundaries

- api/openapi/source/paths/platform.json and
  api/openapi/source/schemas/platform.json are the transport authoring sources;
  YAML and generated artifacts are outputs.
- PostgreSQL/server code owns generation, authority, eligibility, previews,
  revisions, digests, idempotency, and lifecycle state.
- Client code treats question keys, preview IDs, record IDs, capabilities, and
  scope facts as opaque server-owned values.
- Preparation may drive existing authenticated candidate commands but must not
  auto-select, auto-ready, create Findings, accept CAPs, verify Evidence, or
  close Findings.
- Auditee output is a structurally separate projection, not a redacted CAA
  projection.
- CAP acceptance remains separate from Finding closure.
- Reopen and authorized-close reasons are append-only audit facts.
- Organization aliases normalize only through one declared server helper;
  department, unit, membership, subject, and object identity remain exact.
- For each operation, resolve all current bindings matching the exact
  generation, organization alias, department, unit, and provider scope/version.
  Select exactly one binding eligible for that operation and object; never union
  roles from different binding rows, and neutral-deny zero or multiple matches.
- Inspector, Lead, and Auditee operations use their assigned subject, binding,
  and revision pins. Manager/Reviewer operations use a current exact
  operation-eligible binding. Admin exceptions are explicit read/reset-only
  authority unless a contract says otherwise.
- Unknown or unproven authorization fails closed without existence signals.

## Repository orientation and affected interfaces

Contract/generated surfaces:

- api/openapi/source/paths/platform.json
- api/openapi/source/schemas/platform.json
- api/openapi/aviasurveil360.yaml (generated output)
- api/openapi/tests/aga-demo-workspace-contract.test.mjs
- apps/web/src/generated/transport/api-types.ts
- apps/web/src/backend/aga-demo-workspace.ts
- apps/web/src/backend/http-backend.ts

Go domain/persistence:

- apps/api/internal/agademoworkspace/{types.go,service.go,authorization.go,
  postgres_binding_resolver.go,postgres_fact_resolvers.go,batch.go,
  lifecycle.go,lifecycle_types.go,lifecycle_projection.go}
- apps/api/internal/preproddata/agademoworkspace/{postgres_store.go,
  postgres_lifecycle_store.go,postgres_provision.go}
- apps/api/internal/agademoworkspace/contract.go
- apps/api/internal/httpapi/aga_demo_workspace_api.go

Auth/runtime:

- apps/api/internal/httpapi/auth.go
- apps/api/internal/identity/oidc.go
- apps/api/internal/identity/oidc_remote.go
- apps/web/src/auth/{http-auth-gate.tsx,session-client.ts,session-provider.tsx}
- apps/web/src/app/{public-http-config.ts,router.tsx}

React surfaces:

- apps/web/src/app/aga-demo-workspace-routes.tsx
- apps/web/src/features/checklists/aga-classification-workspace-page.tsx
- apps/web/src/features/inspections/{aga-demo-inspection-package-page.tsx,
  aga-demo-inspection-page.tsx}
- apps/web/src/features/findings/aga-demo-potential-finding-page.tsx
- apps/web/src/features/caps/aga-demo-cap-evidence-page.tsx

Preparation/evidence:

- scripts/test-aga-manager-multi-role-demo-connected.sh
- scripts/test-aga-hybrid-demo-workspace-connected.sh
- a new remediation evidence file preserving the original QA report.

## Interface decisions

1. Replace broad capability booleans with exact query/command operation
   allowlists. Admin has read/reset authority, not Manager mutations.
2. Return one canonical opaque server question key, including its base/overlay
   discriminator, and use it unchanged in UI, persistence, cache, and batch.
3. Return server-owned row eligibility and specific ineligibility reasons.
4. Declare a runtime surface discriminator so AGA demo routes do not probe
   unsupported normal APIs.
5. Use discriminated CAA and Auditee lifecycle response shapes. Auditee has
   only issued/public facts and excludes questions, Potential Findings, CAA
   workflow history, internal notes, role pins, workload, and private scoring.
6. Resolve an operation-eligible exact binding tuple as described above, pin
   the selected binding and authorization-scope digest into idempotency, and
   compare object scope for every read/write.
7. Persist a bounded user explanation plus a closed reason code for required
   lifecycle operations in append-only events and event digests.
8. Bound page values and validate before multiplication or slicing.
9. Commit domain mutation and idempotency receipt atomically with CAS checks.
10. Make preview identity server-owned and expiry-aware; exact replay is stable
    and a new post-expiry intent gets a new identity.
11. Use crypto.randomUUID per user intent, retaining it only for in-flight
    retry; never persist operation counters in browser storage.
12. Make authority bindings generation-scoped and resolve only the one current
    ACTIVE generation.

## Ordered phases

### Phase 0 — Plan, tracker, and authorization gate

- Create this plan and exactly one active-index row.
- Keep the older AGA plan state truthful while remediation is open.
- Obtain an independent GPT-5.6-sol ultra read-only review of this plan.
- Incorporate or explicitly reject every critical/important review finding.
- Add F-001 through F-024 to the technical-debt tracker and mark the two
  predecessor AGA rows as corrective-remediation required while retaining their
  historical evidence.
- Record the ultra review resolution below. The current user instruction is
  the explicit local implementation authorization; commits, pushes, production,
  deployment, and external systems remain unauthorized.

### Phase 1 — Transport, projection, authorization, and bounds

Addresses F-007, F-015, F-016, F-018, F-021, F-022, and F-023.

- Update OpenAPI schemas and regenerate transport artifacts.
- Add exact operation allowlists, canonical keys, eligibility, bounded
  pagination, reason fields, and CAA/Auditee response discriminators.
- Resolve all current generation bindings for the exact organization alias,
  department, unit, provider scope/version, membership, subject, operation, and
  object. Select one eligible binding; neutral-deny zero or multiple matches;
  never union roles from different rows. Recheck this selection in each write
  transaction and bind it into the idempotency scope digest.
- Apply object-aware authorization before projecting or mutating.
- Normalize organization aliases through the shared helper, then compare exact
  department/unit/subject/object identity.
- Validate all Finding/CAP/Evidence selectors, including Auditee early-return
  paths, and neutrally deny unrelated IDs.
- Reject the documented huge page value and all values above the closed maximum
  with controlled client errors.
- Add negative tests for wrong unit, mixed bindings, stale generation,
  arbitrary IDs, relationship mismatch, and Auditee CAA-only access.

### Phase 2 — Atomic mutation, reset, preview, and reasons

Addresses F-010, F-011, F-019, and F-024.

- Include generation in authority keys and uniqueness constraints.
- Reconstruct a fresh generation-scoped scope/target/binding set from immutable
  sealed taxonomy/run/fixture inputs, with a deterministic revision-1 Draft,
  new digests, and a new seal. Do not clone edited Draft content, workspace
  additions, rewords, lifecycle state, or an earlier seal.
- Publish the new ACTIVE generation last in one reset transaction with exact
  count/digest validation, terminal-state/CAS checks, uniqueness, and rollback
  on ambiguity.
- Filter all sealed authority views and resolvers to the current ACTIVE
  generation.
- Commit Draft, batch, readiness, recommendation, inspection, lifecycle, and
  idempotency response atomically.
- Recheck CAS, scope, generation, and idempotency inside the transaction.
- Persist a bounded user-entered explanation plus closed reopen/authorized-close
  reason code; include both in command/idempotency digests and authorized CAA
  audit projection, never in the Auditee projection.
- Bind preview identity to expiry and add exact-replay/post-expiry tests.
- Add fault injection and concurrent same-key/different-key/stale-CAS tests.

### Phase 3 — Classification and package state

Addresses F-002, F-003, F-005, F-008, F-009, F-012, F-013, F-014, and F-020.

- Use canonical server keys unchanged for every base and overlay action.
- Disable ineligible Include with the server reason.
- Start the package filter empty; select the first available form only when
  the user has not edited the field.
- Replace component counters with per-intent UUIDs.
- Clear a preview when filter, action, or reason changes; display frozen
  values and confirm only while controls match.
- Disable confirmation for zero matches, over-500 matches, or ineligible
  Include.
- Clear stale recommendation/inspection artifacts on refresh errors.
- Treat cached metadata as a hint and refetch sealed text before loaded state.
- Add Page 1 -> Page 2 -> Page 1 regression coverage.

### Phase 4 — Logout, landing, routes, and role surfaces

Addresses F-004, F-006, F-017, and the UI part of F-018.

- Use OIDC end_session_endpoint when available, with allowlisted post-logout
  return; truthfully report local-only logout when unavailable.
- Make AGA demo authenticated root land on a supported role route without
  probing normal Inspector assignments.
- Replace implicit root logout with an explicit landing component.
- Use a closed AGA suffix matcher; unknown suffixes render stable not-found
  without API calls, redirects, logout, or navigation mutation.
- Give Auditee only public CAP/Evidence context; no Potential Finding or CAA
  review/closure surface.
- Back every visible mutation control with an exact capability or specific
  disabled reason.

### Phase 5 — Separately authorized repeatable connected lifecycle

Addresses F-001 and validates combined behavior.

- Keep prepare non-mutating and return exit 2 while Manager authority is
  pending. Add a separately invoked fixture-owner-approved qualify mode that
  drives real authenticated Manager commands against disposable data.
- Qualify explicitly dispositions all 1,310 leaves in server-previewed batches
  of at most 500, performs MARK_READY_FOR_DEMO_SIMULATION, creates the current
  recommendation and inspection, and emits a metadata-only receipt. React
  never auto-selects or auto-readies.
- Create one current recommendation and inspection with a metadata-only
  preparation receipt containing generation, Draft, counts, and digests.
- Make qualify fail unless exactly one discoverable current inspection remains.
  Keep manual start/status/stop separate from qualify and retain the current
  inspection until separately authorized teardown.
- Update and verify scripts/start-aga-demo.sh, status-aga-demo.sh,
  stop-aga-demo.sh, Make targets, and the runbook with explicit prepare,
  qualify, recovery/fault, cleanup, and serve modes and expected exit codes.
- Verify Manager, Inspector, Lead/CAA Reviewer, and Auditee distinct
  projections over the same inspection.
- Complete one happy lifecycle and the negative authority/privacy matrix.
- Prove zero canonical delta, unchanged grants, and disposable cleanup.

### Phase 6 — Full verification and evidence synchronization

- Run focused gates, then full local gates and connected/browser evidence.
- Use an isolated browser profile and clean task-owned processes.
- Run demo/HTTP artifact-boundary scanners, normal-runtime capability-absence
  checks, connected privacy/media-off E2E, browser storage/cache scans, and
  canonical/overlay/grant/forbidden-artifact delta checks.
- Verify visible logout -> second account without cookie clearing, role landing,
  unknown-route no-API/no-session-mutation, reload/reset races, and
  make aga-demo-up/status/teardown.
- Have GPT-5.6-sol xhigh verify every acceptance mapping after implementation.
- Create fresh remediation evidence while preserving the original report.
- Synchronize this plan, the active index, docs index, build summary, and
  technical-debt tracker.
- Move to completed only with fresh evidence for every required acceptance;
  otherwise keep active/paused/blocked literal.

## Verification commands

Focused Go and tagged runtime:

    go -C apps/api test -count=1 ./internal/agademoworkspace ./internal/preproddata/agademoworkspace ./internal/httpapi ./internal/identity
    go -C apps/api test -count=1 -tags=preproddemo ./internal/agacandidatedemo ./internal/agademoworkspace ./internal/preproddata/agademoworkspace ./internal/httpapi ./cmd/api

Contracts and web:

    ./scripts/generate-contracts.sh
    npm --prefix apps/web run contracts:check
    npm --prefix apps/web run typecheck
    npm --prefix apps/web test
    npm --prefix apps/web run build:demo
    npm --prefix apps/web run build:http

Artifact, regression, and docs:

    node --test api/openapi/tests/aga-demo-workspace-contract.test.mjs tests/openapi-workspace-operation-contract.test.mjs
    node --test tests/aga-hybrid-demo-workspace-boundary.test.mjs tests/demo-boundary-smoke.test.js
    node --test tests/*.test.js
    node tests/harness-docs-smoke.test.js
    git diff --check

Connected/browser:

    npm --prefix apps/web run test:e2e:aga-preprod -- --list
    npm --prefix apps/web run test:e2e:aga-manager -- --list
    make aga-demo-up
    make aga-demo-status
    bash scripts/test-aga-manager-multi-role-demo-connected.sh --mode qualify
    make aga-demo-status
    make aga-demo-down

Each command must record exit status, discovery/pass counts, and a literal
evidence label.

## Acceptance criteria

1. F-001: prepare is non-mutating/pending without authority; separately
   authorized qualify leaves one discoverable current server-bound inspection
   and completes the synthetic lifecycle without 404s.
2. F-002/F-009/F-012: eligible row Include mutates the intended Draft leaf;
   ineligible Include is disabled with a reason; base/overlay mappings persist.
3. F-003/F-008/F-013: initial filter has candidates; control changes invalidate
   previews; invalid previews cannot confirm.
4. F-004/F-006/F-017: visible logout supports local role switching; Inspector
   lands on AGA; unknown routes do not call APIs or log out.
5. F-005/F-010/F-011: operation keys are intent-scoped, retries replay
   exactly, expiry creates fresh previews, and domain/receipt commits are
   atomic under failure/concurrency.
6. F-007/F-015/F-016/F-021/F-022/F-024: operation-eligible exact bindings,
   object IDs, role capabilities, aliases, fresh generation reconstruction,
   and stale bindings fail closed.
7. F-014/F-020: refresh errors clear stale state and page navigation refetches
   sealed question text.
8. F-018: Auditee JSON and DOM structurally omit Potential Findings, CAA
   panels/history, question text, internal notes, private identity/workload,
   and private scoring.
9. F-019: bounded user explanations and closed reopen/authorized-close reason
   codes round-trip through persistence, digest, reload, and authorized audit
   projection without becoming Auditee data.
10. F-023: huge/over-limit pages return controlled 400 responses without panic.
11. The 1,310 inventory, 25-body page limit, immutable question digest,
    no-browser-persistence, no-canonical-delta, unchanged-grants, and cleanup
    boundaries remain green.
12. Evidence is labelled verified locally, candidate-only, release pending,
    production-ready: not established; external/production IdP remains not run.
13. A one-to-one evidence matrix covers F-001 through F-024, including
    overlay/grant/canonical deltas, no forbidden artifacts, page-body/cache
    bounds, and task-owned browser/process cleanup.

## Risks, idempotence, and recovery

- Separate CAA/Auditee DTOs and negative JSON/DOM tests prevent projection
  leakage.
- Reset validates exact counts before publication and leaves the old generation
  current on any failure.
- Canonical opaque keys prevent control-character namespace drift.
- Deterministic lock order and concurrency/fault tests prevent partial commits.
- Local OIDC-provider logout claims are scoped to the connected run only.
- Preparation uses receipts/counts/digests and disposable data; it never writes
  canonical records.
- Replaying the same intent key and payload returns the exact receipt; changed
  scope/payload fails closed.
- If a phase fails, record the blocker and keep controls disabled; do not add
  fallbacks or perform manual canonical repair.
- Preserve zero canonical/overlay/grant delta, no cross-schema objects, no
  original question bodies in workspace/log/build/media artifacts, active page
  body limit 25, at most four text-free metadata pages, and purge on
  session/role/BFCache transitions.

## Progress

- [x] F-001 through F-024 merged QA findings reviewed by GPT-5.6-sol xhigh.
- [x] Repository planning contract and affected boundaries inspected.
- [x] GPT-5.6-sol ultra plan review completed; critical/important corrections
  incorporated and implementation authorization recorded.
- [ ] Transport/authorization/projection remediation.
- [ ] Atomic mutation/reset/remediation.
- [ ] Classification/package state remediation.
- [ ] Auth/routing/role-surface remediation.
- [ ] Connected lifecycle preparation.
- [ ] Full verification and GPT-5.6-sol xhigh acceptance.
- [x] Evidence/index/tracker synchronization for plan/review setup; product
  remediation evidence remains pending.
- [x] Local UI-state remediation for F-003, F-005, F-008, F-013, F-014, and
  F-020: server-first form selection, UUID intent keys, frozen/matching
  preview confirmation, invalid-preview disabling, stale-artifact clearing,
  and sealed-text refetch are covered by focused React tests.
- [x] Local route and session remediation for F-004 (local application-session
  logout), F-006, and F-017: authenticated tagged-local root landing is AGA
  role-specific, no root route revokes a session implicitly, and unknown AGA
  suffixes render an API-free not-found view. The existing local contract does
  not expose an OIDC end-session endpoint or ID-token-hint handoff, so
  identity-provider SSO termination remains unimplemented and must not be
  claimed.
- [x] Local F-024 reset hardening: reset rebuilds the fresh immutable-fixture
  graph, validates exact binding/scope/target counts and generation IDs both
  before and inside the SQL function, locks exactly one ACTIVE generation, and
  publishes ACTIVE only after the new seal graph is complete.
- [x] Local HTTP OIDC cookie remediation: the disposable preprod API now uses
  explicit non-Secure `avia_session`/`avia_csrf` cookies only when
  `AVIA_COOKIE_SECURE=false` is supplied outside production; production keeps
  the `__Host-` secure defaults and rejects the insecure override. The client
  selects the protocol-appropriate CSRF cookie so stale secure cookies cannot
  poison a local POST query header, and the service-worker app-shell marker was
  advanced for the updated bundle.
- [ ] F-010 remains open: Draft, batch, recommendation, inspection, lifecycle,
  and idempotency receipt writes still do not share one universal transactional
  storage operation. Do not claim crash-safe replay, fault-injection coverage,
  or completion of the atomic-mutation phase from the partial reset work.

## Decisions

- Security, privacy, exact authority, and transaction integrity precede
  usability.
- Candidate contracts evolve directly; no obsolete compatibility layer.
- Auditee safety is structural, not conditional redaction.
- UI never auto-classifies or auto-readies.
- Current user authorization covers local implementation only; source control,
  production, deployment, and external systems remain out of scope.

## Discoveries

- Current authorization is broader than aggregate scope; role binding can
  compose scopes; writes and receipts have separate commit boundaries; reset
  omits generation-bound authority; and AGA routes can probe unsupported APIs.

## Ultra review resolutions

- Replaced the unsafe single-tuple resolver with operation-eligible exact
  binding selection and neutral ambiguity denial.
- Replaced reset cloning with immutable-fixture reconstruction and fresh
  revision-1 generation publication.
- Split non-mutating prepare from explicitly authorized disposable qualify,
  including MARK_READY_FOR_DEMO_SIMULATION and lifecycle scripts/Make targets.
- Made the Auditee contract a positive allowlist and kept all CAA workflow
  fields structurally absent.
- Added bounded user explanations alongside closed reason codes.
- Corrected OpenAPI authoring sources, predecessor/index/tracker truth, exact
  connected modes, and the one-to-one evidence matrix.

## Outcome notes

Populate after each material phase with fresh commands, browser artifacts,
remaining gaps, and the final candidate-only readiness decision. Do not infer
success from code presence.

### 2026-08-07 local implementation slice

`verified locally` for the focused unit/type gates only:

- `npm --prefix apps/web run typecheck` passed.
- Focused Vitest route, AGA route, package/classification, and visible-logout
  suites passed (29 tests across four files in the first UI slice; 21 tests
  across three route/logout files in the follow-up slice).
- `GOCACHE=/private/tmp/avia-aga-go-cache go -C apps/api test -count=1
  ./internal/preproddata/agademoworkspace ./internal/agademoworkspace
  ./internal/httpapi` passed. The identity package passed separately after the
  sandbox blocked its local `httptest` loopback bind; it was rerun locally with
  the required loopback permission and passed.
- `git diff --check` passed.

No connected qualification, database integration/fault injection, browser
run, provider end-session verification, full suite, artifact boundary, or
independent xhigh acceptance was run in this slice. The work remains
`candidate-only`, `release pending`, and `production-ready: not established`.

### 2026-08-07 independent xhigh acceptance

The independent GPT-5.6-sol xhigh review completed with `NOT ACCEPTED`. The
focused AGA Go, tagged Go, React, OpenAPI, boundary, docs, and whitespace gates
were `verified locally`, but the plan is not complete. It confirmed these
remaining material blockers: F-002 server row eligibility and Include gating,
F-007 aggregate object/subject scope, F-013 mixed Include confirmation,
F-015 reachable Admin read/reset UI, F-018 Auditee DOM privacy, F-021 selector
and relationship validation, F-010 universal mutation/receipt atomicity, and
F-004 provider end-session logout. F-001 connected qualification and fresh
browser/database fault evidence remain `not run`. Full Vitest was not green:
90 files/756 tests passed and the planning wizard file had two reproducible
failures outside the direct AGA files. The plan is now `paused`,
`candidate-only`, `release pending`, and `production-ready: not established`.

### 2026-08-07 residual source/UI remediation slice

The second Luna-equivalent execution slice added server-owned Include
eligibility and reasons, fail-closed direct Include commands, mixed Include
preview blocking, an Admin read/history/reset-only surface, exact lifecycle
selector and CAP/Evidence relationship validation before Auditee projection,
positive Auditee-only CAP/Evidence DOM, and subject-plus-scope lifecycle
binding checks. The OpenAPI bundle and generated Go/TypeScript transports were
regenerated. Focused Go, tagged Go, React, contract, boundary, docs, and
whitespace checks passed locally. This slice still has no connected
qualification, browser execution, provider end-session, or universal F-010
transaction proof; final independent xhigh acceptance is pending.

### 2026-08-07 local HTTP login-loop remediation

`verified locally` for the disposable local-preprod login/workspace path:

- The root cause was an HTTP client receiving secure `__Host-` cookies; Safari
  discarded the session and returned `/auth/session` as 401. After the first
  local-cookie fix, stale secure CSRF state could still be selected by the old
  app shell, causing the read-only POST workspace queries to fail closed as
  404. The API boundary now receives an explicit environment-bound cookie
  policy, and the client prefers `avia_csrf` on HTTP while retaining secure
  defaults on HTTPS.
- Fresh isolated Playwright verification passed for Manager login, the fixed
  `/department-manager/aga-demo-workspace` route, absence of an alert, the
  server-sealed `1310` inventory count and first table row, and CSRF-backed
  logout: `1 passed`.
- Focused Go packages (`internal/httpapi`, `internal/platform/config`,
  `internal/preproddata/agademoworkspace`) passed; focused Vitest passed
  `19/19`; `npm --prefix apps/web run build:http` and typecheck passed; and
  `git diff --check` passed.

This is still `candidate-only`, `release pending`, and
`production-ready: not established`; full remediation acceptance, provider
end-session verification, universal transaction proof, and stakeholder signoff
remain outstanding.

### 2026-08-07 browser 404 and unsupported-surface remediation

`verified locally` for the rebuilt disposable AGA HTTP artifact and its
user-facing route boundaries:

- Non-JSON HTTP failures now preserve their status as `BackendHttpError`
  instead of being misreported as a protocol failure. The Inspector
  assignment projection renders an explicit empty state when the tagged AGA
  profile has no canonical assignment endpoint.
- In the AGA-only local profile, unsupported canonical role routes are kept
  off the operational navigation and direct links are routed to the connected
  AGA workspace. The Admin checklist-builder route remains available because
  it is backed by the tagged candidate API.
- Focused React/backend tests passed: 7 files, 154 tests. Typecheck and the
  telemetry-disabled `build:http` artifact build passed.
- The isolated preprod browser/privacy matrix passed `17 passed (9.5s)`.
  An additional isolated smoke covered all eight disposable role accounts;
  no raw backend 404 alert was rendered. Lifecycle `404` responses for an
  absent server projection remain expected privacy-preserving responses and
  are rendered as a bounded empty state.
- A read-only traversal of all 85 role-bound route contracts (plus the root
  entry through the login smoke) rendered no raw backend-404 page. Inspector,
  Lead, and Auditee route traversals emitted only the expected absent-lifecycle
  404 responses; no other unexpected failed request was observed.
- `make aga-demo-status` reports the running candidate at
  `http://127.0.0.1:4174/department-manager/aga-demo-workspace` with 1,310
  API questions and the connected matching auditee account.

The connected manager lifecycle pass remains fresh from the prior disposable
target (`1 passed`); this UI-only slice does not mutate the current clean
target. The plan remains `candidate-only`, `release pending`, and
`production-ready: not established`; universal F-010 transaction proof,
provider end-session verification, and stakeholder signoff remain open.

### 2026-08-07 in-app browser follow-up

The connected in-app browser reached the authenticated Manager AGA workspace
and exposed one remaining raw 404 alert on the classification landing screen.
The UI now maps a 404 from the sealed classification/package inventory to a
bounded local-workspace explanation instead of exposing the transport error;
focused classification and package tests cover this behavior. The same
browser session expired before the remaining route traversal could complete,
so fresh authenticated in-app traversal after sign-in is `blocked` pending a
new local identity-provider sign-in. The implementation remains
`candidate-only`, `release pending`, and `production-ready: not established`.

### 2026-08-07 app-shell cache follow-up

The remaining visible raw 404 was traced to a stale Service Worker app shell,
not to a second UI transport path. The working tree had advanced the worker
marker without advancing the generated manifest, so the new worker rejected
its install and the browser kept the previous cached `index.html` and hashed
bundle. The app-shell manifest, worker marker, offline version vector, and
offline test server now share version `5`; worker registration also carries
the version query (`/sw.js?v=5`) so a rebuilt shell is discovered despite an
immutable static response. The HTTP server sends `no-store` for `index.html`,
`sw.js`, and `app-shell-assets.json`; Vite-hashed JS/CSS remain immutable. The
worker remains app-shell-only and does not cache API/auth paths. A waiting
worker is still deliberately not auto-activated: the existing offline safety
policy requires explicit coordinated activation rather than risking local
outbox or package state.
`check:app-shell`, the focused 10-file/203-test React gate, `build:http`, local
status (1,310 questions), header checks, and `git diff --check` passed. The
result remains `candidate-only`, `release pending`, and
`production-ready: not established`.

### 2026-08-07 AGA workspace UX redesign

`verified locally` for the Manager classification workspace presentation and
the connected synthetic sign-in path. The former wide hash-heavy table is now
a three-step Find → Compare → Decide flow: a short local-candidate guide,
server-backed queue cards with technical identifiers behind disclosure, and a
selected-question decision file with explicit scope, confidence, eligibility,
and Draft actions. The page no longer renders the obsolete classification
subtitle, and the focused AGA test locks the guide copy and heading hierarchy.
Because this changes the app-shell asset graph, the generated shell marker and
HTTP/demo manifests were advanced from version 4 to version 5. The workspace
also defines the shared `.sr-only` utility, tightens the Manager header
line-height/spacing, prevents the Dashboard breadcrumb from wrapping, and
replaces the top-bar emoji notification with the product SVG icon language.

Fresh in-app browser verification signed in as the synthetic Manager account,
rendered the `1,310`-question queue without a raw backend error, opened a
decision file, and exercised Page 1 → Page 2 → Page 1 pagination. This remains
candidate-only visual evidence; full remediation acceptance, responsive
matrix coverage, provider end-session verification, universal F-010
transaction proof, and stakeholder signoff remain outstanding. The plan is
`paused`, `release pending`, and `production-ready: not established`.

### 2026-08-07 lifecycle entry-point follow-up

`verified locally` for the no-inspection Manager state across Inspection,
Potential Findings, and CAP/Evidence routes. Those screens correctly fail closed
until the server has released an immutable synthetic inspection, but previously
left the user with disabled controls and no route to the required setup. The
Manager lifecycle navigation now exposes a `Package builder` entry, and the
empty lifecycle state explains the ordered Manager → Inspector → Lead handoff
with a direct `Open package builder` action. Focused lifecycle tests passed
`24/24`; typecheck, demo/HTTP builds, app-shell scans, and `git diff --check`
also passed. An isolated Manager browser check verified the new entry point,
absence of a raw UI 404 alert, and the expected privacy-preserving absent
lifecycle query response. The plan remains `candidate-only`, `release pending`,
and `production-ready: not established`.

## Execution Prompt

Do not resume this plan independently while it is `paused`. Execute
docs/exec-plans/completed/2026-08-07-canonical-aga-preprod-end-to-end-product-plan.md
from Gate 0 after explicit implementation authorization, using this plan as
dirty-diff, open-finding, and donor evidence.
Read it completely, then read AGENTS.md, ARCHITECTURE.md, docs/PLANS.md,
docs/demo-evidence/AGA_BROWSER_QA_REVIEW_2026-08-06.md, relevant product
security/Auditee/lifecycle specifications, and the affected active AGA plans.
Check the plan index and working tree before editing. If the ultra review is
not recorded and resolved, perform only that review and plan updates. Preserve
the root legacy oracle and unrelated user changes. Keep the work candidate-only;
do not touch production, deployment, branches, commits, pushes, or external
systems. Record fresh evidence after each phase, use an isolated browser
profile, clean task-owned processes, and finish with independent xhigh
verification and synchronized plan/index/evidence/tracker records.
