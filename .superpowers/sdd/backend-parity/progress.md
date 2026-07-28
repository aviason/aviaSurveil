# Full Backend Scenario Parity Progress

Plan:
`docs/exec-plans/completed/2026-07-22-full-backend-scenario-parity-plan.md`

## Current State

- Stakeholder closeout: on 2026-07-28 the user accepted Plan 2 as a completed
  local `candidate-only` milestone through the combined Plans 2–4 disposition.
  The historical independent Plan 2 review remains `not run`; release remains
  `release pending`; deployment and production readiness remain `not run`.
- Prerequisite review: completed after an evidence-based Plan 1 preflight and
  the user's explicit acceptance decision. Plan 1 is
  `ready-for-verification`; its one-shot visual result remains literally
  71/259 passed and 188/259 failed, complete comparison-by-comparison manual
  review is `not run`, and standalone baseline integrity is
  `not verified` because the accepted manifest hash differs from the
  user-edited canonical audit hash. The accepted manifest was not replaced.
- Backend Task 1: completed and `verified locally`.
- Backend Task 2: completed and `verified locally`.
- Backend Task 3: completed and `verified locally`.
- Backend Task 4: completed and `verified locally`.
- Backend Task 5: completed and `verified locally`.
- Backend Task 6: completed and `verified locally`.
- Backend Task 7: completed and `verified locally`.
- Backend Task 8: completed and `verified locally`.
- Backend Task 9: completed and `verified locally`.
- Backend Task 10: completed and `verified locally`.
- Backend Task 11: completed and `verified locally`.
- Backend Task 12: completed and `verified locally`.

## Task 3 Evidence

- Focused live identity, settings-migration, user-lifecycle,
  organization-authorization, organization-live, profile-HTTP,
  browser-session, and OIDC-login-state matrix: passed under `-race`
  (`8/8` selected tests).
- Real authorization-code plus PKCE flow against the pinned local Keycloak:
  passed under `-race` (`1/1`).
- Full internal and migration race suite: passed.
- Contract drift gate: passed (`15/15`).
- SQLC drift gate: passed.
- `go vet` for internal, migration, and integration packages: passed.
- `git diff --check`: passed.
- Main-agent specification-compliance review: no remaining Critical or
  Important findings.
- Separate main-agent code-quality review: no remaining Critical or Important
  findings.
- Independent review: `not run`; the two reviews above are not independent.

## Task 4 Evidence

- Focused routine, Ad Hoc, and canonical HTTP planning/assignment scenarios:
  passed together under `-race` (`3/3` selected tests).
- Complete Go integration package: passed under `-race` with PostgreSQL,
  pinned local Keycloak, and private MinIO test services.
- Complete internal and migration race suite: passed.
- Frontend HTTP transport/mapper tests: passed (`15/15`).
- Complete web unit suite: passed (`604/604` across `58/58` files).
- Shared mock Backend contract: passed (`16/16`).
- Focused shared live `HttpBackend` Task 4 normalized transcript: passed
  (`1/1`).
- Contract drift gate: passed (`15/15`).
- SQLC drift gate, typecheck, `go vet`, and `git diff --check`: passed.
- Main-agent specification-compliance review: no remaining Critical or
  Important findings after fixing missing Planning decision transaction links
  and fail-closed materialization row-effect validation through RED → GREEN.
- Separate main-agent code-quality review: no remaining Critical or Important
  findings.
- Independent review: `not run`; the two reviews above are not independent.

## Task 5 Evidence

- Intended RED: the focused integration scenario failed to compile because
  `configuration.NewWorkspaceService`,
  `configuration.ErrWorkspaceForbidden`, and the template workflow commands
  did not exist.
- Frozen contract boundary: question create/list and draft
  create/add/reorder are live; the Plan 1 handoff exposes no standalone
  question update/delete or template publish command. No command was invented.
  The seeded published version, version history, and exact immutable
  Audit/package snapshots are tested.
- Focused direct and exact-HTTP template/checklist scenarios: passed together
  under `-race` (`2/2` selected tests).
- Complete Go integration package: passed under `-race`.
- Complete internal and migration race suite: passed.
- Contract drift gate: passed (`15/15`).
- SQLC drift gate: passed.
- Frontend HTTP transport/mapper tests: passed (`16/16`).
- Typecheck: passed.
- Complete web unit suite: passed (`606/606` across `58/58` files).
- Shared mock Backend contract: passed (`17/17`).
- Focused shared live `HttpBackend` Task 5 transcript: passed (`1/1`).
- Complete shared live `HttpBackend` contract: passed (`16/16`) with the
  deterministic worker active.
- The complete live transcript exposed an older Task 3 Auditee organization
  registry mismatch. A strict RED → GREEN correction now returns only the
  caller's own organization; focused internal, direct-integration, and HTTP
  tests passed under `-race`.
- Main-agent specification-compliance review: no remaining Critical or
  Important findings.
- Separate main-agent code-quality review: no remaining Critical or Important
  findings.
- Independent review: `not run`; the two reviews above are not independent.

## Task 6 Evidence

- Intended RED: Observation conversion incorrectly produced
  `WAITING_FOR_CAP` with CAP/Evidence required, an earlier scan-clean Evidence
  version could close despite a later immutable version, and Evidence review
  accepted empty `Comment to Auditee` and `Internal CAA Note` values.
- Focused correction REDs additionally proved ignored exact Inspection
  Attachment IDs, incorrect CAP-only/Evidence-only states, incorrect
  Observation UI defaults, cross-organization Evidence readiness disclosure,
  CAP direct-ID projection drift, accepted HTTP path/body identity drift,
  missing upload transaction links, earlier-CAP review after resubmission, and
  post-resubmission CAP supersession drift.
- Focused Potential Finding/Finding/CAP/Evidence domain plus full lifecycle
  selection: passed under `-race`.
- Complete internal, migration, and integration race matrix: passed with the
  separately fixture-owned pinned-Keycloak flow explicitly excluded from the
  direct command. The unconfigured attempt failed before that test body because
  `AVIA_TEST_OIDC_ISSUER_URL` was absent.
- Evidence upload/worker regressions: passed. The live deterministic worker
  processed two scan batches; exact scan outbox counts were `pending=0` and
  `terminal=0`.
- Contract drift gate: passed (`15/15`).
- SQLC drift gate: passed.
- Focused frontend HTTP transport, Lead UI, and mock contract tests: passed
  (`36/36`).
- Typecheck: passed.
- Complete web unit suite: passed (`610/610` across `58/58` files).
- Mock Backend contract: passed (`18/18`), comprising `17/17` shared scenarios
  and one mock-only 86-screen capability-boundary test.
- Complete shared live `HttpBackend` contract: passed (`17/17`) with the
  deterministic worker active.
- `go vet`, harness-document smoke (`1/1`), and `git diff --check`: passed.
- Main-agent specification-compliance review found two Important issues:
  earlier CAP review after resubmission and post-resubmission mock/HTTP CAP
  status drift. Both were fixed through new RED → GREEN cycles; no Critical or
  Important findings remain.
- Separate main-agent code-quality review: no remaining Critical or Important
  findings.
- Independent review: `not run`; the two reviews above are not independent.

## Task 7 Evidence

- Intended RED: prove that report preparation/readiness and exact immutable
  family/version rules are absent, Executive issue does not enqueue one
  idempotent document render job, no rendered immutable document version or
  private object metadata is produced, and document download authorization plus
  organization-scoped Auditee report projections are unavailable. The focused
  HTTP RED also covers report path/body identity drift and the missing
  `documents` and `auditeeReports` `HttpBackend` capability slices.
- Focused RED confirmed on 2026-07-24: with a task-local writable Go cache, the
  report package failed to compile because `reports.Prepare`,
  `reports.PrepareInput`, and typed report kinds did not exist, while the
  integration package failed because the root `internal/documents` service
  package did not exist. The preceding default-cache attempt was an
  environment-only sandbox denial and is not the feature RED.
- GREEN and checkpoint review: typed Preliminary/Final immutable families,
  exact ready version plus Audit/organization/Finding identity, decision
  authority, issue locking, one idempotent render request, private immutable
  output metadata, crash-safe retry, and no Finding auto-closure are enforced.
  Download authorization rechecks organization, released source snapshot,
  exact Finding identities, and `CLEAN`/`CANONICAL` object state. Auditee
  report/document projections are organization-scoped and unsafe legacy
  snapshots are excluded.
- Focused report/document/worker integration selection: passed under `-race`.
  The complete internal/migration/integration race matrix passed with only the
  separately fixture-owned pinned-Keycloak test explicitly excluded from that
  direct command.
- Contract drift gate: passed (`15/15`).
- SQLC drift gate: passed.
- Frontend focused HTTP transport plus backend contracts: passed (`32/32`
  across `2/2` files), comprising `14/14` HTTP mapper tests and `18/18`
  MockBackend tests (`17/17` shared plus one mock-only boundary).
- Complete web suite: passed (`611/611` across `58/58` files).
- Typecheck and `build:http`: passed; the HTTP artifact scan covered 144 files
  and 152 inputs.
- Complete shared live `HttpBackend` contract: passed (`17/17`).
- Focused live report transcript: passed (`1/1`).
- Final live document worker state:
  `SUCCEEDED|1|1|1|0|1` for status, attempts, document versions, private
  metadata rows, pending render-request outbox rows, and completion outbox
  rows. The reset-race regression passed with no worker error.
- `go vet`, harness-document smoke (`1/1`), and `git diff --check`: passed.
- Main-agent specification-compliance review found two Important issues:
  immutable report family kind drift and missing exact Audit/organization
  validation of report Finding IDs. Both were fixed through new RED → GREEN
  cycles.
- Separate main-agent code-quality/privacy review found one Important issue:
  same-organization download of a legacy unsafe report source snapshot. It was
  fixed through a new RED → GREEN cycle. No Critical or Important finding
  remains.
- Independent review: `not run`; the two reviews above are not independent.

## Task 8 Evidence

- Intended RED: require the absent communications and notifications
  domain/application services, canonical HTTP routes, generated transport
  mappers, and `HttpBackend` slices. The scenario covers organization/object
  scope, immutable attachment metadata, Auditee exclusion of `INTERNAL_CAA`
  notes, recipient-only unread/read behavior, exact revision and idempotency,
  recipient-specific in-app records plus email jobs, 30/15/7/due/overdue
  reminder scheduling and duplicate suppression, safe notification content,
  and authorized-work-only calendar list/direct reads.
- Focused RED confirmed on 2026-07-24: the Task 8 integration selection failed
  at setup/compile time because the root `internal/communications` domain
  package was absent. PostgreSQL and the test body were not reached; this is
  the intended feature RED.
- Minimum domain/application GREEN: all three direct live PostgreSQL scenarios
  passed. The next exact HTTP RED reached the server and returned `404` for
  `POST /v1/communications`, confirming the canonical routes were absent.
- HTTP GREEN passed. The focused frontend RED then failed before its first
  request because `backend.communications` was undefined.
- Frontend GREEN passed. The focused shared Mock transcript RED then proved
  that an Auditee-visible message did not create its recipient-specific
  notification. The first correction run also exposed the frozen calendar
  projection's exact work-date semantics (`2026-06-18`, not the assignment
  start date).
- GREEN and checkpoint review: completed. The final complete Go
  race/integration matrix, generated contract/SQLC drift gates, frontend
  mapping tests, full web suite, and Mock/HTTP communications transcript all
  passed. The active plan contains the literal Task 8 results.

## Task 9 Evidence

- Intended RED: require the absent governed risk/management and assistant
  services, exact administration projections, generated transport mapping,
  `HttpBackend` slices, and normalized Mock/HTTP parity. The focused race
  selection failed at compile time because `internal/assistant` did not exist.
- GREEN: immutable versioned advisory risk projections, CAP-effectiveness
  reasons, minimized deterministic assistant input, transactional assistant
  idempotency/audit/change/outbox linkage, closed Admin filters, and the frozen
  86-screen/108-action registry are implemented.
- Focused frontend HTTP mapping plus Mock contracts passed `36/36`; the full
  web suite passed `615/615` across `58/58` files.
- Final scoped Go race verification passed, including `internal/httpapi` in
  `1.538s` and the complete integration package in `44.107s`.
- Final complete live `HttpBackend` contract passed `19/19`; the focused Task 9
  shared transcript passed and covered exactly 86 screens plus 108 actions.
- Contract drift passed `15/15`; canonical OpenAPI examples passed `14/14`.
  SQLC drift, typecheck, `go vet`, `build:http`, `git diff --check`, the
  144-file/152-input HTTP artifact scan, and 86-route parity-boundary scan
  passed.
- A worker/reset PostgreSQL `40P01` deadlock was transient and is not counted.
  The accepted fresh full live run was green.
- Main-agent specification-compliance review found one Important Auditee
  privacy issue: Internal CAA-only message activity affected the Auditee
  message-screen state. A new RED reproduced it and the minimum GREEN now
  scopes the projection to `AUDITEE_VISIBLE` messages for the exact actor
  organization.
- Separate main-agent code-quality review: no remaining Critical or Important
  findings.
- Independent review: `not run`; the two reviews above are not independent.

## Task 10 Evidence

- Intended focused RED: `40/44` passed and exactly four failed for the
  17-versus-86 dual-route count, a redirected formerly blocked direct load, an
  undefined aggregate registry, and leaked private risk fields.
- Aggregate GREEN: all 28 Backend capability slices are required and
  registered; all 86 route contracts are exactly `["demo", "http"]`; profile
  transport and strict allowlisted risk mapping are complete. Focused tests
  passed `44/44`.
- Complete web unit suite: passed `620/620` across `58/58` files. Typecheck and
  `build:http` passed.
- Shared MockBackend contract: passed `20/20`.
- Complete shared live `HttpBackend` contract: passed `19/19` on the accepted
  fresh run with the deterministic scanner/document worker active.
- HTTP artifact scan: passed (`144 files`, `152 inputs`); parity boundary:
  passed (`86 routes`, `2 build profiles`).
- Full canonical HTTP browser matrix: passed `10/10`, including `2/2` real
  offline-sync scenarios and exactly `258/258` visible-action inventories for
  86 routes at desktop, tablet, and mobile.
- Complete Go race matrix: passed with
  `GOCACHE=/private/tmp/avia-task10-go-cache go test -race -p 1 -count=1 ./...`.
  Contract drift passed `15/15`, canonical examples passed `14/14`, and SQLC
  drift plus `git diff --check` passed.
- Strict correction cycles covered the missing upload/object-store logical
  clocks, the invalid not-ready Preliminary Report Department Review fixture,
  nullable unreleased Auditee report projections, exact V0/V1 report versions,
  the HTTP offline lazy-route race, and explicit live-control ownership.
- Main-agent specification-compliance review found one Critical accepted-
  baseline hash replacement. It was corrected by restoring the accepted
  `92a8…` manifest hash while preserving the user-edited audit; standalone
  baseline integrity remains `not verified`.
- Separate main-agent code-quality review found one Important missing
  canonical Planning clock and fixed it through RED → GREEN. A proposed
  worker-clock injection then failed live `18/19` because fixed-time fallback
  IDs collided; systematic debugging removed that incorrect change and the
  accepted fresh live run passed `19/19`.
- No remaining Critical or Important findings. Independent review: `not run`;
  the two reviews above are not independent.

## Task 11 Evidence

- The shared full-platform implementation covers exactly 10 scenario families
  and records 45 required proofs. Exact normalized Mock/HTTP transcript parity
  includes IDs, revisions, statuses, owners, roles, organization/version
  scope, audit events, notification/document jobs, denials, and dashboard
  projections.
- Intended mutation REDs reject a skipped HTTP project, normalized mismatch,
  forbidden private field, missing denial, missing audit event, missing
  required proof, and UI-only state.
- Live RED → GREEN corrections covered the normal OfflineGrant checkout
  boundary, exact grant identity, normal Inspection Attachment registration,
  the complete Final Report DM → GM → Executive Director chain, fixture-owned
  grant creation, canonical browser report-stage preparation, dynamic browser
  device scope, and the HTTP visible-action report authority chain.
- Accepted fresh `./scripts/test-http-profile.sh`: passed the complete Go
  `-race -p 1 -count=1 ./...` matrix; contract drift `15/15`; SQLC drift;
  typecheck; `620/620` web tests across `58/58` files; demo and HTTP builds;
  the `144 files`/`152 inputs` HTTP artifact scan; live `HttpBackend` `19/19`;
  Mock Playwright `35/35`; HTTP Playwright `17/17`; full-platform `7/7` in
  each profile; exactly `258/258` visible-action inventories; and
  worker/outbox observability.
- Fresh normal OIDC profile: passed `1/1`. Focused HTTP offline browser checks:
  passed `3/3`, including causal recovery and stale-revision
  conflict/re-entry.
- Expiry, revoke, logout, and user-switch grant boundaries pass in the
  required Go/offline regression suites. The shared frozen Backend transcript
  covers checkout, causal replay, conflict/re-entry, attachment delivery, and
  device/session-scope denial.
- Main-agent specification-compliance review: no remaining Critical or
  Important findings.
- Separate main-agent code-quality review: no remaining Critical or Important
  findings.
- Independent review: `not run`; the two reviews above are not independent.

## Task 12 Evidence

- Exact final matrix: contract `15/15`, canonical examples `14/14`, SQLC drift,
  complete Go race, typecheck, web `621/621` across `58/58` files,
  root/oracle `108/108`, demo and HTTP builds, app-shell `144 files` and
  `76 assets` for both profiles, HTTP artifact `144 files`/`152 inputs`, and
  parity boundary `86 routes`/`2 build profiles`.
- Browser/profile matrix: independent Mock `35/35`; full HTTP profile live
  Backend `19/19`, Mock `35/35`, HTTP `17/17`, and exactly `258/258`
  visible-action inventories; OIDC `1/1`; offline `7/7`; worker/outbox
  observability passed.
- Final scope: 75 OpenAPI paths, 81 operations, 79 non-health operations,
  80 frozen Backend methods, 160 schemas, 30 guarded mutations, six source
  fragments, schema version 11, 25 full-platform relations, 19 SQLC stores,
  87 named queries, 28 Backend slices, 86 dual-profile routes, eight roles,
  108 unique screen/action pairs, 10 scenario families, and 45 proofs.
- A real worker/reset `40P01` exposed by the first Task 12 full profile was
  fixed through a focused RED → GREEN cycle with test-profile-only, bounded,
  context-aware deadlock retry. The accepted fresh full profile passed and
  cleaned its scoped services.
- An OIDC pass exposed an unhandled second logout under React `StrictMode`.
  A focused RED observed two commands; the minimum GREEN adds a one-shot guard
  and truthful caught failure state. The final OIDC run passed `1/1` without
  the rejection.
- Final spec review found stale Task 2 inventory evidence and an incorrect
  Plan 1 standalone-baseline label. Focused documentation REDs reproduced
  both; synchronized evidence now records 25 relations/46 new-store queries
  and preserves standalone baseline integrity as `not verified`.
- Main-agent specification-compliance review: no remaining Critical or
  Important findings.
- Separate main-agent code-quality review: no remaining Critical or Important
  findings.
- Independent review: `not run`; the two reviews above are not independent.
- Evidence:
  `docs/demo-evidence/FULL_BACKEND_SCENARIO_PARITY_2026-07-22.md`.

## Repository Actions

- Staging: completed selectively for Plan 2; unrelated user-owned changes were
  excluded.
- Commit: explicitly authorized after verification; this ledger is included in
  the same final local commit.
- Push: `not run`.
- Deployment: `not run`.
- Release status: `release pending`.
- Artifact status: `candidate-only`.
- Production status: not `production-ready`.
