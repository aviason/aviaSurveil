# Full Backend Scenario Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> Execute this plan within its defined scope and acceptance criteria.
> Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement every capability required by all 86 React screens in the
Go/PostgreSQL backend and prove that complete multi-role scenarios produce the
same authorized outcomes in deterministic mock and real HTTP profiles.

**Architecture:** Extend the existing Go modular monolith and one versioned
OpenAPI contract. Preserve module-owned tables, application-service
coordination, same-origin session authority, transactional idempotency/audit/
change/outbox writes, and typed `HttpBackend` mapping. New platform-facing work
is queued through outbox adapters; Plan 3 supplies the production-like local
the then-current external OIDC provider, ClamAV, Mailpit, Gotenberg, and gateway services.

**Tech Stack:** Go 1.26, `chi`, PostgreSQL 17, `pgx`, `sqlc`, OpenAPI 3.1,
generated Go/TypeScript transport, React `HttpBackend`, external OIDC boundary,
MinIO-compatible private storage, Vitest contracts, Go race/integration tests,
and Playwright mock/HTTP transcript parity.

**Status:** `completed` — Tasks 1–12 remain `verified locally`, and on 28 July
2026 the user accepted Plan 2 as a completed local `candidate-only` milestone.
The historical independent Plan 2 review remains `not run`; that truthful
checkpoint boundary was not rewritten by stakeholder acceptance. Release is
`release pending`, and deployment and production readiness are `not run`.

## Stakeholder Closeout — 28 July 2026

The user explicitly authorized the combined Plans 2–4 stakeholder closure and
accepted this plan only as a completed local `candidate-only` milestone. The
canonical implementation and verification basis remains
[Full Backend Scenario Parity Evidence](../../demo-evidence/FULL_BACKEND_SCENARIO_PARITY_2026-07-22.md);
the combined decision and shared exclusions are recorded in the
[Plans 2–4 Stakeholder Disposition](../../demo-evidence/stakeholder/PLANS2_4_STAKEHOLDER_DISPOSITION_2026-07-28.md).

This closeout does not turn the historical independent-review result into a
pass and does not authorize a release, deployment, provider integration, or
production change. Production retention, legal hold, deletion, records
operations, identity federation, external email/storage/scanning/model
providers, and operating decisions remain deferred to their owners. The
artifact remains `candidate-only`; release remains `release pending`;
deployment and production readiness remain `not run`; no `production-ready`
claim is made.

## Objective

Replace the remaining demo-only capability boundary with real server authority.
At completion all 86 React routes are available in both demo and HTTP profiles,
every visible command reaches an authorized Go application service or explicit
field-local boundary, and all critical scenarios replay against live PostgreSQL
without test skips or mock fallbacks.

## Scope

- Expand the OpenAPI, generated transports, `Backend`, and `HttpBackend` for all
  capabilities frozen by the 86-screen React plan.
- Add authoritative persistence and services for communications, calendars,
  profiles/settings, teams/assignments, documents, notifications/reminders,
  risk/analytics projections, administration, and advisory drafts.
- Complete existing organization, planning, inspection, checklist,
  Potential Finding, Finding, CAP, Evidence, report, configuration, audit, and
  sync modules for every screen state and action.
- Activate the 69 routes currently marked demo-only in HTTP profile only after
  their contract, authorization, persistence, and live test pass.
- Run identical screen and scenario contracts through `MockBackend` and
  `HttpBackend` and compare normalized transcripts.
- Preserve the existing local OIDC and canonical-header lanes as separate tests.
- Produce synchronized evidence and a complete HTTP capability ledger.

## Assumptions

- Plan 1 has completed Tasks 1–12, reached `ready-for-verification`, and its 86
  route IDs, source-role metadata, capability interfaces, deterministic demo
  scenarios, and visible-action ledger have been independently accepted.
- Plan 3 will replace deterministic worker adapters with local real services;
  this plan must still persist and expose real job state, retry state, and
  immutable output metadata.
- Risk, safety intelligence, SSP/NASP, USOAP, and Oversight Health projections
  are configured management indicators, not automatic legal decisions.
- Inspector Assistant produces a server-side advisory draft using a
  deterministic provider; no external model or autonomous action is introduced.
- Production retention/legal-hold rules remain owner decisions, so this plan
  performs no destructive automatic deletion.

## Global Constraints

- Follow `AGENTS.md`, `MODULE_ARCHITECTURE.md`, canonical workflow docs, and the
  frozen Plan 1 capability contract.
- Server object and field authorization is authoritative for list, direct ID,
  download, decision, audit, notification, and sync paths.
- Auditee response schemas structurally omit internal CAA fields.
- CAP acceptance never closes a Finding; scan-clean exact Evidence and a
  distinct authorized verification/closure decision remain required.
- Every submitted CAP, Evidence, report, checklist template, question set, and
  generated document version is immutable.
- Every state transition writes a required audit event. Important transitions
  also write authorized change feed and outbox records in the same transaction.
- Mutations require idempotency keys and expected revisions where replay or
  concurrency is possible. Conflicts preserve the user's draft.
- Do not add microservices, Kafka, RabbitMQ, destructive retention, external AI,
  production deployment, or HTTP-to-mock fallback.
- Work on the current branch and preserve unrelated untracked paths.

## Ownership Boundaries

| Owner | Responsibility |
|---|---|
| Product/CAA Operations | Scenario truth, role authority, decision vocabulary, and indicator meaning |
| Backend | OpenAPI, domain services, authorization, PostgreSQL, outbox, workers, and live projections |
| Frontend | Frozen capability interface, transport mapping, route activation, conflict/error presentation |
| Identity/Security | Session/OIDC/CSRF policy and field-level privacy review |
| QA | Mock/HTTP invariant transcripts, raw-wire scans, concurrency, migration, recovery, and browser gates |
| Records/Legal | Later production retention/legal-hold decision; no destructive policy is inferred here |

## Module And Persistence Structure

Existing packages remain authoritative and are expanded in place:

- `internal/organizations`, `planning`, `inspections`, `checklists`,
  `potentialfindings`, `findings`, `caps`, `evidence`, `reports`,
  `configuration`, `auditlog`, `identity`, and `sync`.

New bounded packages:

- `internal/assignments`
- `internal/communications`
- `internal/notifications`
- `internal/documents`
- `internal/risk`
- `internal/administration`
- `internal/assistant`

Each persistent module owns `store/postgres/queries.sql` and generated sqlc
files. Cross-module workflows are coordinated by `internal/application` and do
not update another module's tables directly.

### Incremental HTTP mapping rule

Tasks 4–9 each own the generated transport mapping, `HttpBackend` capability
slice, and dual-backend contract for the domain implemented in that task. Their
mock/HTTP transcript checks must not depend on future Task 10 work. Route
metadata remains demo-only until Task 10 runs the aggregate live gate and flips
reviewed routes to dual-profile availability.

## Required Scenario Set

1. Routine/announced Cabin Inspection from Planning through closed Finding.
2. Ad Hoc/unannounced intake through Finance, GM release, Lead preparation,
   executable Audit materialization, withheld notice, and field assignment.
3. Checklist submission/reopen, Potential Finding return/dismiss/convert, and
   Observation defaults.
4. CAP submit/revise/accept/return, Evidence version upload/scan/review, Close,
   Partially Close, Not Close, and separate authorized closure.
5. Preliminary and Final Report DM → GM → Executive Director chains and
   organization-scoped Auditee issue/preview.
6. Checklist template/question/version/package authoring and immutable Audit
   snapshot behavior.
7. Organization master data, teams/workload, calendars, communications,
   documents, reminders, settings, users/roles projection, and audit log.
8. Management risk/SSP/USOAP/CAP-effectiveness projections with no automatic
   enforcement or legal decision.
9. Offline Inspector checkout, causal sync, conflict/re-entry, attachment
   delivery, expiry/revoke/logout/user-switch boundaries.
10. Advisory Inspector draft request with no canonical mutation.

## Phases

1. **Contract and persistence — Tasks 1–2:** freeze the full OpenAPI bundle,
   migrations, SQLC stores, and transaction ownership.
2. **Domain authority — Tasks 3–9:** complete identity, planning, checklist,
   lifecycle, report/document, communication, notification, risk/admin, and
   advisory-draft capabilities.
3. **HTTP activation and proof — Tasks 10–12:** map `HttpBackend`, activate all
   86 HTTP routes, enforce identical scenario transcripts, and record evidence.

---

### Task 1: Modularize And Freeze The Full OpenAPI Contract

**Progress:** `verified locally` on 2026-07-23. The generated OpenAPI 3.1 bundle
contains 75 paths, 81 operations, 79 non-health capability operations covering
all 80 frozen Backend methods (the role-safe `getFinding` read is shared), 160
schemas, and 30 guarded mutations. All six non-empty source fragments assemble
deterministically. TypeScript and Go transports are generated from the bundle.

RED recorded on 2026-07-23: `./scripts/check-contracts.sh` reached the existing
lint and Go generator successfully, then failed the new contract tests with 44
named full-platform capability-to-operation omissions, all six missing
deterministic source fragments, the missing full-platform example directory,
and the missing OIDC command-guard metadata. This is the intended feature RED;
it is not a tooling failure.

GREEN and checkpoint review: `./scripts/check-contracts.sh` passed 15/15 tests
with lint, source/bundle equality, examples, role metadata, closed-response
schemas, two deterministic generations, and exact TypeScript/Go drift checks.
`npm --prefix apps/web run typecheck`, the generated Go package test with an
isolated writable Go cache, and `git diff --check` passed. The main-agent
spec-compliance review and separate code-quality review found no remaining
Critical or Important findings. Review-found exact transport-shape,
role-matrix, response-closure, and second-generation drift gaps were fixed
through focused RED → GREEN contract cycles. This review is not independent.

**Files**

- Create `api/openapi/source/openapi.json`
- Create `api/openapi/source/paths/core.json`
- Create `api/openapi/source/paths/workflows.json`
- Create `api/openapi/source/paths/platform.json`
- Create `api/openapi/source/schemas/domain.json`
- Create `api/openapi/source/schemas/platform.json`
- Modify `api/openapi/aviasurveil360.yaml` as the generated bundled artifact
- Modify `scripts/generate-contracts.sh`
- Modify `scripts/check-contracts.sh`
- Modify `api/openapi/tests/contract-examples.test.mjs`
- Add closed-schema examples under `api/openapi/examples/full-platform/`

**Interfaces**

The bundled contract remains OpenAPI 3.1 JSON content at the existing path.
Source fragments are assembled deterministically; generated TypeScript and Go
types must fail drift checks. Every mutation declares `Idempotency-Key`, CSRF,
expected revision, typed problem responses, and role/organization security.

- [x] Write failing bundling/drift/example tests that require every Plan 1
  capability method to map to an operation ID and every response to use closed
  schemas with Auditee-safe variants.
- [x] Run `./scripts/check-contracts.sh`; confirm failure names missing full
  platform operation IDs, not invalid tooling.
- [x] Implement deterministic fragment assembly, schemas, operations, examples,
  security metadata, pagination, ETags/revisions, and generated transports.
- [x] Run contract lint, bundle reproducibility, examples, TypeScript generation,
  Go generation, and `git diff --exit-code` against a second generation.
- [ ] Commit exactly `feat(api): freeze full platform contract`. `not run` —
  Git staging and commits are not authorized in this task.

### Task 2: Extend Migrations, SQLC, And Transaction Ownership

**Progress:** `verified locally` on 2026-07-23. Schema version 10 is composed of
four new forward-only migrations over the retained v6 boundary. It adds 25
full-platform relations, six module-owned SQLC stores with 46 checked queries,
append-only version triggers, retention-safe tombstones, scoped deduplication,
named list/detail indexes, and an exact command-transaction helper linking
idempotency, audit, authorized change, and outbox rows in one transaction.

RED recorded on 2026-07-23: the focused inventory test compiled and named all
four absent migrations plus the six absent module-owned SQLC stores;
`./scripts/check-sqlc.sh` failed on the first missing generated assignments
store. With only the PostgreSQL test service running, the sandbox-exempt live
test then reported schema version 6 instead of 10, missing workflow/platform
relations and scoped indexes, absent `operation_id` linkage columns, and absent
module queries. A separate application-transition RED reached the live schema
and found zero linked idempotency/audit/change/outbox envelopes. These were
feature REDs. The first sandboxed localhost attempt was discarded as an
environment failure because network access stopped before PostgreSQL.

GREEN and checkpoint review: fresh v10 and retained v6 N-1 upgrade, immutable
update/delete rejection, scoped uniqueness, six named list indexes, six detail
PK indexes, module write ownership, two-scope operation linkage, and the
existing lost-acknowledgement replay passed against PostgreSQL 17.6. The final
focused live command passed under `-race -p 1 -count=1`; all
`./internal/... ./migrations` packages passed under `-race`;
`./scripts/check-sqlc.sh` and `git diff --check` passed. The main-agent
spec-compliance review and separate code-quality review found no remaining
Critical or Important findings. Review-found globally scoped operation-link
identity and notification full-list index defects were fixed through focused
RED → GREEN cycles; SQLC legacy/full-schema ownership was separated to avoid
unrelated generated-type drift. This review is not independent.

**Files**

- Create `apps/api/migrations/000007_full_workflow_projection.up.sql`
- Create `apps/api/migrations/000008_communications_documents.up.sql`
- Create `apps/api/migrations/000009_notifications_risk_admin.up.sql`
- Create `apps/api/migrations/000010_identity_settings.up.sql`
- Modify `apps/api/migrations/migrations.go`
- Add module-owned `store/postgres/queries.sql` and generated files for every
  new persistent package
- Modify `scripts/check-sqlc.sh`
- Create `apps/api/tests/integration/full_schema_test.go`

**Interfaces**

Migrations are forward-only and N-1 compatible for the supported local upgrade
window. Foreign keys may reference stable IDs but no module query writes another
module's tables. Append-only/version tables reject updates and deletes through
database constraints where practical.

- [x] Write failing live-PostgreSQL tests for fresh migration, N-1 upgrade,
  immutable versions, uniqueness/idempotency, organization scope indexes,
  outbox/change/audit linkage, and forbidden cross-module writes.
- [x] Run the focused integration test and `./scripts/check-sqlc.sh`; confirm red
  for absent migrations/queries.
- [x] Implement migrations, queries, sqlc configuration, constraints, indexes,
  retention-safe tombstone fields, and exact transaction helpers.
- [x] Run fresh/N-1 migrations, SQLC drift, race tests, and PostgreSQL query plan
  assertions for list/detail paths.
- [ ] Commit exactly `feat(api): add full platform persistence`. `not run` —
  Git staging and commits are not authorized in this task.

### Task 3: Complete Identity, Profile, Role, And Organization Authority

**Progress:** `verified locally` on 2026-07-23. RED recorded on 2026-07-23: the focused Task 3
integration test stopped at compile time on the absent
`internal/administration` application package, before any database connection.
The test also names the missing revisioned profile/settings service, guarded
organization reader, profile HTTP operation, user-lifecycle request/outbox
envelope, and target-session invalidation behavior. This is the intended
feature RED; it is not a tooling or PostgreSQL failure.

GREEN and checkpoint review: profile and settings updates now preserve
server-projected role/organization authority, require expected revisions and
idempotency, write one linked audit/change/outbox/idempotency envelope, expose
strong `"rev-N"` ETags, and reject closed-schema role/organization injection.
The organization registry uses guard-before-store authorization for all seven
CAA roles; Auditee list access is denied and own-organization direct reads
return closed projections while cross-organization IDs return indistinguishable
not-found results before storage. User lifecycle requests persist `PENDING`
provisioning jobs and invalidate target sessions for suspension/role changes;
logout revocation is audited. Migration 10 backfills retained v9 subjects,
tombstoned profiles cannot create new sessions, and the then-current pinned provider
Authorization Code + PKCE test passed 1/1 under `-race`.

The final focused PostgreSQL/session/HTTP command passed all eight selected
tests under `-race -p 1 -count=1`; all `./internal/... ./migrations` packages
passed under `-race`; contract checks passed 15/15; SQLC drift, `go vet`, and
`git diff --check` passed. The main-agent spec-compliance review and separate
code-quality review found no remaining Critical or Important findings.
Review-found idempotency-key reuse, tombstoned-login, session-audit,
retained-subject backfill, and organization-list ETag defects were fixed
through focused RED → GREEN cycles. This review is not independent.

**Files**

- Extend `apps/api/internal/identity/`
- Extend `apps/api/internal/organizations/`
- Create `apps/api/internal/administration/users.go`
- Create `apps/api/internal/administration/authorization.go`
- Create `apps/api/internal/administration/store/postgres/queries.sql`
- Modify `apps/api/internal/httpapi/auth.go`
- Modify `apps/api/internal/httpapi/api_projections.go`
- Create `apps/api/tests/integration/identity_organization_scope_test.go`

**Interfaces**

Profiles and settings are application records keyed by stable session subject.
Role membership is projected from the authenticated session; clients cannot
assert roles. User provisioning commands persist requested lifecycle state and
an outbox job; Plan 3 performs the real provider-admin API call.

- [x] Write failing tests for role/list/direct-ID authorization, user lifecycle
  request, profile/settings revisions, logout/revocation, Auditee own-org-only
  raw wire data, and no pre-guard data fetch.
- [x] Run focused Go and HTTP tests; confirm missing service/operation failures.
- [x] Implement application services, projections, authorization, session
  invalidation, provisioning job records, and audit events.
- [x] Run race tests, live OIDC session tests, raw JSON forbidden-field scans,
  and direct-ID/list isolation.
- [ ] Commit exactly `feat(api): complete identity and organization authority`.
  `not run` — Git staging and commits are not authorized in this task.

### Task 4: Complete Planning, Assignment, Team, And Inspection Packages

**Progress:** `verified locally` on 2026-07-24. Routine and Ad Hoc intake,
zero-budget Finance, return/re-entry, Department preparation, Lead/team/question
assignment, workload, exact-template materialization, duplicate replay,
withheld-notice privacy, package drafts, team projections, and Auditee
coordination now share the live application and HTTP capability slice.

RED recorded: the first focused scenario failed to compile because the
assignment module was absent. Later focused review REDs proved that the legacy
planning-decision path produced no `command_transaction_links` row, canonical
HTTP reset returned `404` for the Planning draft, and materialization returned
success when the Inspection was no longer in `PREPARATION`. Each was fixed
through a focused GREEN cycle without weakening authority, privacy, or
immutable snapshot rules.

GREEN and checkpoint review: all three focused Task 4 integration scenarios
passed together under `-race`; the complete integration package and complete
internal/migration race suite passed. Frontend transport tests passed 15/15,
the complete web unit suite passed 604/604 across 58 files, the mock Backend
contract passed 16/16, and the focused live `HttpBackend` normalized transcript
passed 1/1 against the same shared implementation. Typecheck, `go vet`, the
15/15 generated-contract gate, SQLC drift, and `git diff --check` passed. The
main-agent specification-compliance review and separate main-agent code-quality
review found no remaining Critical or Important findings after the transaction
link and materialization row-effect corrections. These reviews are not
independent.

**Files**

- Extend `apps/api/internal/planning/`
- Create `apps/api/internal/assignments/state.go`
- Create `apps/api/internal/assignments/service.go`
- Create `apps/api/internal/assignments/authorization.go`
- Create `apps/api/internal/assignments/store/postgres/queries.sql`
- Extend `apps/api/internal/inspections/`
- Modify `apps/api/internal/application/service.go`
- Create `apps/api/tests/integration/planning_assignment_scenario_test.go`
- Modify `apps/web/src/backend/http-backend.ts`
- Modify `apps/web/src/backend/transport-mappers.ts`
- Modify `apps/web/src/backend/http-backend.test.ts`
- Expand `apps/web/tests/contract/` for planning, assignment, team, and package capabilities

**Interfaces**

Planning item creation, Finance/GM/Executive decisions, Department preparation,
Lead/team assignment, question assignment, package snapshot, and Audit
materialization are separate commands. Unannounced notice remains withheld and
no executable Audit exists before preparation confirmation.

- [x] Write failing routine and Ad Hoc scenario tests with revision conflict,
  denial, zero-budget Finance, return/re-entry, workload, exact template version,
  duplicate materialization, and organization-notice assertions.
- [x] Run focused Go/integration tests and confirm missing states/services.
- [x] Implement state machines, transactional services, projections, assignment
  package snapshots, audit events, change feed, outbox records, generated
  transport mappers, and the matching `HttpBackend` capability slice.
- [x] Run race/integration and normalized mock/HTTP planning transcripts.
- [ ] Commit exactly `feat(api): complete planning and assignment workflow`.
  `not run` — Git staging and commits are not authorized in this task.

### Task 5: Complete Checklist, Template, Question, And Package Configuration

**Progress:** `verified locally` — the frozen question/template draft editing,
reorder, immutable published-version history, immutable package assembly, and
shared mock/HTTP checklist-configuration transcript pass. The Plan 1 handoff
exposes question create/list and draft create/add/reorder commands, but no
standalone question update/delete or template publish command. This task did
not invent those commands: the seeded published version and exact immutable
Audit/package snapshots are tested, while publication remains outside the
transport contract.

**Files**

- Extend `apps/api/internal/checklists/`
- Extend `apps/api/internal/configuration/`
- Create `apps/api/internal/application/template_workflow.go`
- Modify `apps/api/internal/httpapi/route_families_api.go`
- Create `apps/api/tests/integration/template_execution_snapshot_test.go`
- Modify `apps/web/src/backend/http-backend.ts`
- Modify `apps/web/src/backend/transport-mappers.ts`
- Modify `apps/web/src/backend/http-backend.test.ts`
- Expand `apps/web/tests/contract/` for checklist and configuration capabilities

**Interfaces**

Draft questions/templates/packages are editable only at valid draft revisions.
Publish creates immutable versions. Existing Audits retain exact template,
question, reference, evidence-expectation, and package snapshots.

- [x] Write failing tests for the frozen question create/list and draft
  add/reorder actions, multiline text, immutable published-version history,
  package assembly, Audit snapshot immutability, checklist
  save/submit/reopen, and role denial.
- [x] Run focused tests and confirm absent operations/state red.
- [x] Implement state, services, authorization, projections, version snapshots,
  transaction/audit/outbox behavior, generated transport mappers, and the
  matching `HttpBackend` capability slice.
- [x] Run race/integration, generated-contract, and mock/HTTP checklist/template
  transcript parity.
- [ ] Commit exactly `feat(api): complete checklist configuration workflow`.
  `not run` — Git staging and commits are not authorized in this task.

The intended RED failed to compile because `configuration.NewWorkspaceService`,
`configuration.ErrWorkspaceForbidden`, and the application template workflow
commands did not exist. The focused direct and exact-HTTP scenarios then passed
`2/2` under `-race`; the complete Go integration package and the complete
internal/migration race suites passed. Contract drift passed `15/15`, SQLC drift
passed, frontend transport/mapper tests passed `16/16`, typecheck passed, the
complete web suite passed `606/606` across `58/58` files, the MockBackend
contract passed `17/17`, the focused live Task 5 transcript passed `1/1`, and
the complete live shared HttpBackend contract passed `16/16`.

The complete live transcript exposed an older organization-registry mismatch:
the Auditee profile was denied instead of receiving its own organization-scoped
projection. A new RED → GREEN correction added authoritative organization
scope; its focused internal, direct-integration, and HTTP tests passed under
`-race`. Main-agent specification-compliance and separate main-agent
code-quality reviews found no remaining Critical or Important findings.
Independent review is `not run`; these self-reviews are not independent.

### Task 6: Complete Finding, CAP, Evidence, And Closure Authority

**Progress:** `verified locally` — exact Observation defaults and explicit
CAP/Evidence configuration, Potential Finding attachment binding, immutable
latest-only CAP/Evidence review, separate CAP and Evidence authority,
scan-clean Evidence gating, explicit authorized closure, closed-safe
organization projections, transactional upload envelopes, concurrency, replay,
and exact MockBackend/HttpBackend lifecycle transcripts pass.

**Files**

- Extend `apps/api/internal/potentialfindings/`
- Extend `apps/api/internal/findings/`
- Extend `apps/api/internal/caps/`
- Extend `apps/api/internal/evidence/`
- Extend `apps/api/internal/application/canonical_transitions.go`
- Extend `apps/api/internal/application/auditee_projections.go`
- Create `apps/api/tests/integration/full_finding_lifecycle_test.go`
- Modify `apps/web/src/backend/http-backend.ts`
- Modify `apps/web/src/backend/transport-mappers.ts`
- Modify `apps/web/src/backend/http-backend.test.ts`
- Expand `apps/web/tests/contract/` for Finding, CAP, Evidence, and closure capabilities

**Interfaces**

The exact chain is Checklist Response → Potential Finding → Lead decision →
Finding → immutable CAP revision/review → immutable Evidence version/scan/review
→ verification or explicit authorized closure. Partial/not-close remains open.

- [x] Write failing tests for every valid/invalid transition, Observation
  defaults, return/dismiss reasons, concurrency, exact version scan gate,
  CAP-not-closure, partial/not-close, authorized closure, organization privacy,
  audit/outbox atomicity, and idempotent replay.
- [x] Run focused Go/live HTTP tests and confirm state/service red failures.
- [x] Implement missing reads/commands/projections, transaction coordination,
  generated transport mappers, and the matching `HttpBackend` capability slice
  without overwriting submitted versions.
- [x] Run race/integration, upload/worker contract, raw-wire privacy, and full
  mock/HTTP canonical transcript comparison.
- [ ] Commit exactly `feat(api): complete finding lifecycle authority`.
  `not run` — Git staging and commits are not authorized in this task.

The intended RED reached three authoritative failures: an Observation
conversion produced `WAITING_FOR_CAP` with CAP and Evidence enabled, an earlier
scan-clean Evidence version could close despite a later immutable version, and
Evidence review accepted empty public/Internal comments. Subsequent focused RED
cycles exposed ignored Inspection Attachment IDs, incorrect CAP-only and
Evidence-only state selection, Observation UI defaults, cross-organization
Evidence readiness disclosure, CAP direct-ID status/privacy drift, accepted
path/body identity drift, missing Evidence upload transaction links, review of
an earlier CAP after resubmission, and an old CAP projection that remained
`MORE_INFORMATION_REQUESTED` instead of `SUPERSEDED`.

GREEN and checkpoint review: the focused Potential Finding/Finding/CAP/Evidence
domain plus full lifecycle selection passed under `-race`; the complete
internal/migration/integration race matrix passed with only the separately
fixture-owned pinned-provider test explicitly excluded from that direct
command. Upload/worker regressions passed, and the live deterministic worker
processed two scan batches with exact scan outbox counts `pending=0` and
`terminal=0`. Contract drift passed `15/15`, SQLC drift passed, focused
frontend transport/UI/mock tests passed `36/36`, typecheck passed, the complete
web suite passed `610/610` across `58/58` files, the MockBackend contract passed
`18/18` including `17/17` shared scenarios plus one mock-only boundary, and the
complete live shared `HttpBackend` contract passed `17/17`. `go vet`, the
harness-document smoke test (`1/1`), and `git diff --check` passed.

Main-agent specification-compliance review found two Important issues:
reviewing an earlier CAP revision after resubmission and post-resubmission
mock/HTTP CAP status drift. Both were fixed through new RED → GREEN cycles.
Separate main-agent code-quality review found no remaining Critical or
Important findings. Independent review is `not run`; these self-reviews are not
independent.

### Task 7: Complete Reports And Document Job Authority

**Progress:** `verified locally` on 2026-07-24. Intended RED records the missing report
preparation/readiness and exact immutable-family/version rules, the absence of
an idempotent document render job and rendered immutable output/private object
metadata, unavailable download authorization and organization-scoped Auditee
report projections, accepted report path/body identity drift, and the missing
`documents` and `auditeeReports` `HttpBackend` capability slices.

Focused RED confirmed on 2026-07-24: using a task-local writable Go cache, the
report package failed to compile because the preparation API and typed kinds
were absent, and the integration package failed because the root document
service package was absent. The preceding default-cache sandbox denial is not
counted as the feature RED.

GREEN and checkpoint review: Preliminary and Final reports are distinct typed
immutable families; exact ready versions, Audit/organization/Finding identity,
stage, role, and path/body identity are enforced. Issue creates one
transaction-linked, idempotent render request without closing a Finding. The
deterministic Plan 2 renderer writes a private object, immutable document
version, hash/size metadata, audit/change/completion outbox records, and
supports crash-safe retry. Download authorization rechecks the caller,
organization, released report source snapshot, exact Finding identities, and
`CLEAN`/`CANONICAL` object state before returning a short-lived instruction.
Auditee report and document projections are organization-scoped and omit
unsafe legacy snapshots.

Focused report/document/worker integration selection passed under `-race`;
the complete internal/migration/integration race matrix passed with only the
separately fixture-owned pinned-provider test explicitly excluded from that
direct command. The final focused worker reset-race regression passed, and the
complete shared live `HttpBackend` contract passed `17/17`. The focused live
report transcript passed `1/1`; final live worker state was literally
`SUCCEEDED|1|1|1|0|1` for job status, attempts, document versions, private
metadata rows, pending render-request outbox rows, and completion outbox rows.
Contract drift passed `15/15`; SQLC drift, `go vet`, typecheck,
`build:http`, harness-document smoke, and `git diff --check` passed. The
complete web suite passed `611/611` across `58/58` files. Focused frontend
transport and backend contracts passed `32/32` across `2/2` files, comprising
`14/14` HTTP mapper tests and `18/18` MockBackend tests (`17/17` shared plus
one mock-only boundary).

Main-agent specification-compliance review found two Important issues:
immutable report family kind drift and missing exact Audit/organization
validation for report Finding IDs. The separate main-agent code-quality/privacy
review found one Important issue: legacy unsafe report source snapshots could
authorize a same-organization download. All three were fixed through new
RED → GREEN cycles. No Critical or Important finding remains. Independent
review is `not run`; these self-reviews are not independent.

**Files**

- Extend `apps/api/internal/reports/`
- Create `apps/api/internal/documents/state.go`
- Create `apps/api/internal/documents/service.go`
- Create `apps/api/internal/documents/store/postgres/queries.sql`
- Create `apps/api/internal/documents/renderer.go`
- Extend `apps/api/internal/application/service.go`
- Create `apps/api/tests/integration/report_document_workflow_test.go`
- Modify `apps/web/src/backend/http-backend.ts`
- Modify `apps/web/src/backend/transport-mappers.ts`
- Modify `apps/web/src/backend/http-backend.test.ts`
- Expand `apps/web/tests/contract/` for report and document capabilities

**Interfaces**

Preliminary and Final Reports are separate immutable version families. DM, GM,
and Executive decisions bind exact versions. Organization issue creates an
Auditee-safe projection. Document rendering is an idempotent outbox job whose
real Gotenberg adapter arrives in Plan 3.

- [x] Write failing tests for preparation/readiness, return, DM/GM/Executive
  authority, exact version, duplicate decision, Auditee issue, document job,
  output hash/version, download authorization, and no Finding auto-closure.
- [x] Run focused tests; confirm missing document/report operations.
- [x] Implement report/document states, services, projections, job records,
  private object metadata, audit/change/outbox transaction behavior, generated
  transport mappers, and the matching `HttpBackend` capability slice.
- [x] Run race/integration and normalized mock/HTTP report transcripts.
- [ ] Commit exactly `feat(api): complete report and document workflow`.
  `not run` — Git staging and commits are not authorized in this task.

### Task 8: Add Communications, Notifications, Reminders, And Calendars

**Progress:** `verified locally`. The implementation covers exact
organization/object visibility, complete exclusion of `INTERNAL_CAA` notes
from Auditee list/direct access, immutable attachment metadata,
recipient-only unread/read authority, exact-revision/idempotent read commands,
one in-app record and one email job per recipient event, 30/15/7/due/overdue
reminder scheduling with duplicate suppression, safe notification content,
provider-agnostic delivery retry state, and calendar projections derived only
from already-authorized Audit work.

Focused RED confirmed on 2026-07-24: the Task 8 integration selection stopped
at setup/compile time because the root `internal/communications` domain package
did not exist. PostgreSQL and the test body were not reached. This is the
intended feature RED, not a tooling or sandbox failure.

After the minimum domain/application GREEN passed all three direct live
scenarios, the exact HTTP RED reached the server and returned `404` for
`POST /v1/communications`, confirming the canonical routes were still absent.
After the HTTP GREEN, the focused frontend RED failed because
`backend.communications` was undefined before any request was made.
After the frontend GREEN, the focused shared Mock transcript RED proved that a
saved Auditee-visible message did not yet create its recipient-specific
notification. The first correction run also exposed the frozen calendar
projection's exact work-date semantics (`2026-06-18`, not the assignment start
date).

Subsequent specification-review REDs proved that an Auditee could attach to an
incoming CAA message, expired but non-revoked recipient sessions were omitted,
unsupported roles could reach Communications, mutation headers were not exact,
a roleless principal could list Notifications, an unreleased preparation Audit
leaked into the Auditee calendar, and the Manager Calendar capability was
incorrectly denied despite the frozen capability matrix. Code-quality REDs
proved that the delivery adapter ran while the job row remained locked and that
storage accepted the same immutable object metadata for two Communication
attachment rows. The first complete live transcript also returned
`forbidden` for the older sync scenario because the canonical clock was not
injected into the grant and operation services. Each defect was corrected
through a focused GREEN without weakening identity, privacy, semantic truth,
or the root oracle.

The final five-scenario Task 8 integration selection passed together under
`-race`; the complete internal/migration/integration race matrix passed with
only the separately script-owned pinned-provider test explicitly excluded from
that direct command. Contract drift passed `15/15`; SQLC drift, `go vet`, and
typecheck passed. Focused frontend transport and Mock contracts passed `34/34`
across `2/2` files. The complete web suite passed `613/613` across `58/58`
files; the MockBackend contract passed `19/19`. The focused live Task 8 shared
transcript passed `1/1`, and after the canonical clock correction the complete
live shared `HttpBackend` contract passed `18/18`. `build:http`, the HTTP
artifact scan (`144` files, `152` inputs), the app-shell scan, and the exact
`86`-route/two-profile parity boundary passed.

One manual live run logged a recoverable document-worker deadlock while the
test-only canonical reset concurrently removed fixture render state. The
shared contract remained `18/18`, the worker transaction rolled back, and
read-only inspection found no persistent render job or pending render outbox
row. This transient is recorded separately and does not establish a normal
OIDC/full-mode defect; the script-owned worker/reset profile gates remain
required in Tasks 11–12.

The main-agent specification-compliance review and separate main-agent
code-quality review found no remaining Critical or Important findings after
the correction cycles above. These self-reviews are not independent;
independent review is `not run`.

**Files**

- Create `apps/api/internal/communications/`
- Create `apps/api/internal/notifications/`
- Extend existing reminder/configuration projections
- Create `apps/api/internal/application/communications_workflow.go`
- Create `apps/api/tests/integration/communications_notification_test.go`
- Modify `apps/web/src/backend/http-backend.ts`
- Modify `apps/web/src/backend/transport-mappers.ts`
- Modify `apps/web/src/backend/http-backend.test.ts`
- Expand `apps/web/tests/contract/` for communications, notification, reminder, and calendar capabilities

**Interfaces**

Threads are object- and organization-scoped. `AUDITEE_VISIBLE` messages and
`INTERNAL_CAA` notes are distinct types and storage paths. Notification events
produce recipient-specific in-app records and idempotent email jobs; Plan 3
delivers them through SMTP. Calendars are projections over authorized work, not
an independent source of truth.

- [x] Write failing tests for message visibility, unread counts, internal-note
  exclusion, attachment metadata, reminder rules, Due Soon/Overdue scheduling,
  duplicate suppression, calendar scope, and email job audit trail.
- [x] Run focused tests and confirm absent modules.
- [x] Implement services, queries, projections, scheduler command, recipient
  policy, outbox jobs, retries, audit events, generated transport mappers, and
  the matching `HttpBackend` capability slice.
- [x] Run race/integration and mock/HTTP communications/notification transcripts.
- [ ] Commit exactly `feat(api): add communications and notification authority`.
  `not run` — Git staging and commits are not authorized in this task.

### Task 9: Add Risk, Intelligence, Analytics, Admin, And Advisory Draft Projections

**Progress:** completed and `verified locally`. The intended RED integration
contract required the absent governed risk/management projection service and
assistant provider/service, plus exact role denial, source/provenance,
non-decision labels, prompt/data minimization, no canonical mutation,
administration/audit filters, generated transport mappers, `HttpBackend`
slices, and normalized Mock/HTTP transcripts. The focused race run failed at
compile time exactly because `internal/assistant` did not exist:
`no required module provides package .../internal/assistant`. No production
code was changed to manufacture that RED.

The GREEN implementation adds immutable versioned advisory risk projections,
explicit CAP-effectiveness reasons, a minimized deterministic assistant
provider boundary, transactionally linked assistant idempotency/audit/change/
outbox writes, closed administration projections and filters, the embedded
86-screen/108-action frozen registry, generated transport mapping, and the
matching `HttpBackend` slices. Focused frontend mapping plus Mock contracts
passed `36/36`; the complete web suite passed `615/615` across `58/58` files.
The final scoped Go race run passed, including `internal/httpapi` in `1.538s`
and the complete integration package in `44.107s`. The final live shared
`HttpBackend` contract passed `19/19`, including exact Task 9
MockBackend/HttpBackend normalized transcript parity. Contract drift passed
`15/15`, canonical OpenAPI examples passed `14/14`, and SQLC drift, typecheck,
`go vet`, `build:http`, `git diff --check`, the 144-file/152-input HTTP artifact
scan, and the 86-route parity-boundary scan all passed.

One live attempt without the required deterministic worker stopped at
`EVIDENCE_SUBMITTED`; it was an incomplete harness configuration and is not an
accepted product run. One later worker/reset overlap returned PostgreSQL
`40P01 deadlock`; it is recorded as transient and not counted. The accepted
fresh live run was fully green.

The main-agent specification-compliance review found one Important Auditee
privacy issue: an Internal CAA-only message could change the Auditee message
screen from `empty` to `ready`. A new RED reproduced the disclosure, and the
minimum GREEN scopes the state query to `AUDITEE_VISIBLE` messages for the
actor's exact organization. The focused race test and all final regressions
passed. The separate main-agent code-quality review found no remaining
Critical or Important findings. These self-reviews are not independent;
independent review is `not run`.

**Files**

- Create `apps/api/internal/risk/`
- Extend `apps/api/internal/administration/`
- Create `apps/api/internal/assistant/provider.go`
- Create `apps/api/internal/assistant/service.go`
- Create `apps/api/internal/assistant/deterministic_provider.go`
- Create `apps/api/tests/integration/management_admin_assistant_test.go`
- Modify `apps/web/src/backend/http-backend.ts`
- Modify `apps/web/src/backend/transport-mappers.ts`
- Modify `apps/web/src/backend/http-backend.test.ts`
- Expand `apps/web/tests/contract/` for risk, analytics, administration, and advisory-draft capabilities

**Interfaces**

Risk/SSP/USOAP/CAP-effectiveness/Oversight Health values are versioned
projections with source, calculation time, reason, and non-decision label.
Assistant drafts accept authorized Finding/checklist context, redact forbidden
fields, return a draft with provenance, and perform no canonical mutation.

- [x] Write failing tests for projection source/refresh, role denial, no automatic
  enforcement/closure, administration lists/audit filters, deterministic draft,
  prompt/data minimization, no mutation, and complete audit events.
- [x] Run focused tests and confirm missing modules.
- [x] Implement read models, configured calculations, admin projections,
  deterministic advisory provider/service, generated transport mappers, and the
  matching `HttpBackend` capability slice.
- [x] Run race/integration, raw-wire privacy, and mock/HTTP projection parity.
- [ ] Commit exactly `feat(api): add governed management projections`.
  `not run` — Git staging and commits are not authorized in this task.

### Task 10: Finalize The HttpBackend Registry And Activate 86 HTTP Routes

**Progress:** completed and `verified locally`. The intended focused RED passed
40/44 and failed exactly four assertions: only 17 routes were dual-profile, a
formerly blocked HTTP direct load redirected, the aggregate registry was
undefined, and the risk mapper leaked unexpected private fields. Timeout,
cancellation, conflict/problem, authentication-loss, and cursor behavior were
already green.

The minimum GREEN requires all 28 Backend capability slices, makes all 86
routes exactly `["demo", "http"]`, adds the missing profile transport, and
rebuilds the risk projection through an explicit allowlist. Focused registry,
route, mapper, router, and shared-error tests passed 44/44. Typecheck, the full
web suite (620/620 across 58/58 files), MockBackend contract (20/20), live
`HttpBackend` contract (19/19), `build:http`, the 144-file/152-input HTTP
artifact scan, and the 86-route/two-profile parity boundary passed.

The full canonical HTTP browser matrix passed 10/10. It includes the exact
canonical lifecycle, three first-route viewports, two real offline-sync
scenarios, the release boundary, and 86 routes at desktop/tablet/mobile:
258/258 visible-action inventories. The worker-backed attachment path exposed
two real clock defects and the Preliminary Report fixture exposed an invalid
not-ready Department Review state; each was corrected through separate RED →
GREEN cycles without adding a test-only normal-mode route or mock fallback.
The complete Go `-race -p 1 -count=1 ./...` matrix passed, as did contract drift
(15/15), canonical examples (14/14), SQLC drift, and `git diff --check`.

The main-agent specification-compliance review found one Critical accepted-
baseline hash replacement; the manifest was restored to the accepted `92a8…`
hash while preserving the user-edited audit, so standalone baseline integrity
remains truthfully `not verified`. The separate main-agent code-quality review
found one Important missing canonical Planning clock and fixed it through RED
→ GREEN. A proposed worker-clock change then failed the live contract 18/19 by
creating duplicate fixed-time object IDs; systematic debugging removed that
incorrect injection and the accepted fresh live run passed 19/19. No Critical
or Important findings remain. These self-reviews are not independent;
independent review is `not run`.

**Files**

- Modify `apps/web/src/backend/backend.ts`
- Modify `apps/web/src/backend/http-backend.ts`
- Modify `apps/web/src/backend/http-backend.test.ts`
- Modify `apps/web/src/backend/transport-mappers.ts`
- Modify `apps/web/src/backend/transport-mappers.test.ts`
- Modify `apps/web/src/app/route-contracts.ts`
- Modify `apps/web/src/app/route-contracts.test.ts`
- Expand `apps/web/tests/contract/`
- Modify `apps/web/scripts/assert-http-artifact.mjs`

**Interfaces**

Tasks 4–9 have already mapped their domain slices from generated transport to
UI-domain projections. Task 10 fails on any unmapped capability, consolidates
shared error/cursor/session behavior, and changes each route to
`availableProfiles: ["demo", "http"]` only after its exact live contract passes.
No handwritten transport DTO or hidden retry is allowed.

- [x] Write failing adapter contracts for every capability, HTTP problem,
  cancellation, timeout, conflict, unauthorized session, cursor, raw forbidden
  field, and route-profile activation.
- [x] Run focused Vitest/HTTP contracts and confirm any remaining registry,
  shared-error, or activation gaps rather than domain mappings deferred from
  Tasks 4–9.
- [x] Complete the aggregate registry and shared adapters, activate each route
  in reviewed domain batches, and keep local draft/conflict presentation explicit.
- [x] Run typecheck, all backend contracts against mock and live HTTP, artifact
  exclusions, and direct loads for all 86 routes.
- [ ] Commit exactly `feat(web): activate full http capability parity`.
  `not run` — Git staging and commits are not authorized in this task.

### Task 11: Enforce Complete Multi-Role Mock And HTTP Transcript Parity

**Progress:** completed and `verified locally` on 2026-07-24. The shared
scenario implementation covers all 10 required scenario families and records
45 required proofs. Its normalized transcript fixes exact entity IDs,
revisions, statuses, owners, roles, organizations, versions, audit-event
types, notification/document jobs, denials, and dashboard projections. The
same transcript passed under `MockBackend` and live `HttpBackend`.

The intended mutation REDs reject a skipped HTTP project, normalized mismatch,
forbidden private field, missing denial, missing audit event, missing scenario
proof, and UI-only state. The live RED → GREEN sequence additionally exposed
and corrected a seeded OfflineGrant that bypassed normal checkout, non-contract
grant IDs, attachment upload without normal registration, an incomplete Final
Report authority chain, a stale test fixture that depended on the removed
grant, a canonical browser fixture at the wrong report stage, a hard-coded
browser device ID, and an HTTP visible-action report fixture that produced two
403 console errors. Each correction was made at the authoritative fixture or
normal command boundary; no HTTP-to-mock fallback or normal-mode reset/seed
route was added.

The accepted fresh `./scripts/test-http-profile.sh` run passed the complete Go
race matrix, contract drift 15/15, SQLC drift, web typecheck, 620/620 web tests
across 58/58 files, demo and HTTP builds, the 144-file/152-input HTTP artifact
scan, live `HttpBackend` 19/19, Mock Playwright 35/35, HTTP Playwright 17/17,
the 7/7 full-platform scenario/mutation project in each profile, exactly
258/258 visible-action inventories, and worker/outbox observability. The fresh
normal OIDC profile passed 1/1. Focused HTTP offline checks passed 3/3,
including causal recovery and stale-revision conflict/re-entry.

Expiry, revoke, logout, and user-switch grant boundaries remain proved by the
required Go/offline regression suites; the shared frozen Backend transcript
proves checkout, causal replay, conflict/re-entry, attachment delivery, and
device/session-scope denial. The test-profile reset is guarded by explicit
test-only configuration, and the normal OIDC/full application returns 404 for
`/__test/*`.

The main-agent specification-compliance review and the separate main-agent
code-quality review found no remaining Critical or Important findings.
Independent review is `not run`; these two reviews are not independent.

**Files**

- Create `apps/web/tests/e2e/full-platform-scenarios.spec.ts`
- Create `apps/web/tests/contract/full-platform-backend.contract.ts`
- Modify `apps/web/playwright.config.ts`
- Modify `scripts/test-http-profile.sh`
- Modify `tests/parity/behavior-ledger.json`
- Modify `tests/parity/react-legacy-parity.test.mjs`
- Create `apps/api/tests/integration/full_platform_denials_test.go`
- Create `apps/api/cmd/test-profile-reset/`
- Create `apps/api/internal/httpapi/test_profile_boundary_test.go`
- Create `scripts/reset-test-profile.sh`

**Interfaces**

The same scenario implementation runs under mock and HTTP projects. A normalized
transcript records exact entity IDs, revision/status/owner, role, organization,
version, audit-event types, notification/document jobs, denials, and dashboard
projections.

Deterministic reset is an explicit test-profile-only, out-of-process one-shot
command. It is never registered as a route in the normal OIDC/full API. The full
profile starts from fresh scoped volumes and uses normal authorized
provisioning/application commands; `/__test/*` must return 404 there.

- [x] Write failing scenario tests for the 10 required scenario families and
  mutation fixtures for a skipped HTTP project, normalized mismatch, forbidden
  field, missing denial, missing audit event, or UI-only state change.
- [x] Run mock and HTTP scenario commands and confirm the HTTP gaps fail.
- [x] Complete the test-profile-only reset command, deterministic clocks/IDs,
  transcript normalizer, live service orchestration, and fail-closed project
  checks. Add mutation tests proving reset/seed routes cannot be registered in
  normal OIDC/full configuration.
- [x] Run Go race/integration, HTTP contracts, 86 HTTP routes × three viewports,
  OIDC, offline sync, and exact mock/HTTP transcript equality.
- [ ] Commit exactly `test(api): enforce full scenario parity`.
  `not run` — Git staging and commits are not authorized in this task.

### Task 12: Run The Full Backend Matrix And Prepare Handoff

**Progress:** `verified locally` on 2026-07-24. The exact required matrix passed
from fresh runs: contract 15/15, canonical examples 14/14, SQLC drift, complete
Go race, typecheck, web 621/621 across 58/58 files, root/oracle 108/108, both
builds, both 144-file/76-asset app-shell scans, the 144-file/152-input HTTP
artifact scan, 86 routes/two profiles, Mock browser 35/35, HTTP browser 17/17,
exactly 258/258 visible-action inventories, live Backend 19/19, OIDC 1/1,
offline 7/7, and worker/outbox observability. Task-owned services and generated
test residue were cleaned.

Two final quality corrections followed strict RED → GREEN: the test-profile
reset now retries only PostgreSQL `40P01` with a bounded context-aware policy,
and the OIDC logout route issues one command under React `StrictMode` while
truthfully presenting caught failure. Final OIDC output contains no unhandled
logout rejection. The final spec review also corrected stale Task 2 inventory
evidence to the current 25 full-platform relations and 46 queries in the six new
stores, and restored the literal Plan 1 standalone-baseline `not verified`
label. No accepted baseline or user-edited audit was changed.

The main-agent specification-compliance review and separate main-agent
code-quality review found no remaining Critical or Important findings. These
reviews are not independent. Independent Plan 2 review is `not run`.

**Files**

- Create `docs/demo-evidence/FULL_BACKEND_SCENARIO_PARITY_2026-07-22.md`
- Modify `docs/demo-evidence/BUILD_SUMMARY.md`
- Modify `docs/index.md`
- Modify `MANIFEST.md`
- Modify `docs/exec-plans/index.md`
- Modify `docs/exec-plans/tech-debt-tracker.md`
- Modify this plan

- [x] Run clean generation, migrations, SQLC, Go race/unit/integration, React,
  artifacts, root oracle, mock contracts/browser, canonical HTTP, OIDC, offline,
  all 86 HTTP routes, all 10 scenario families, audits, and cleanup.
- [x] Record endpoint/operation/module/table/role/screen/action coverage and
  prove 86 dual-profile routes with zero mock imports in HTTP and zero
  test/reset routes in the normal full API.
- [x] Record transient failures separately and count only fresh full green runs.
- [x] Write synchronized evidence and set the plan to
  `ready-for-verification` only if every required local gate passes.
- [x] Record the completed plan in the explicitly authorized final local
  Conventional Commit. Push remains `not run` because it was not authorized.

## Required Verification Matrix

```bash
npm --prefix apps/web ci
./scripts/check-contracts.sh
./scripts/check-sqlc.sh
node api/openapi/tests/contract-examples.test.mjs
GOCACHE=/private/tmp/aviasurveil360-full-go-cache go -C apps/api test -race -p 1 -count=1 ./...
npm --prefix apps/web run typecheck
npm --prefix apps/web test
node --test tests/*.test.js tests/parity/react-legacy-parity.test.mjs
npm --prefix apps/web run build:demo
npm --prefix apps/web run build:http
npm --prefix apps/web run check:app-shell
node apps/web/scripts/assert-http-artifact.mjs apps/web/dist/http
node apps/web/scripts/assert-parity-boundary.mjs
npm --prefix apps/web run test:e2e:mock
./scripts/test-http-profile.sh
./scripts/test-http-oidc-profile.sh
npm --prefix apps/web run test:e2e:offline
```

Expected final scope: 86 dual-profile routes, complete live PostgreSQL
projections, 10 mock/HTTP-identical scenario families, zero HTTP mock/seed input,
zero skipped projects, and no task-owned residue.

## Risks And Controls

| Risk | Control |
|---|---|
| OpenAPI becomes an unreviewable monolith | Deterministic source fragments and generated bundled canonical artifact |
| Shared database becomes cross-module coupling | Module-owned SQLC stores, transaction coordinator, forbidden-write tests |
| UI and API drift despite matching names | Same backend contract and scenario implementation against mock and HTTP |
| Auditee or direct-ID privacy leak | Raw-wire schema variants, list/direct-ID/download denial tests |
| Idempotency hides partial transactions | Mutation/audit/change/outbox exact transaction assertions and lost-ack replay |
| Indicator or assistant becomes authority | Non-decision labels, no-mutation tests, explicit role/field authorization |
| External services block backend progress | Persist real job state now; activate adapters in Plan 3 |
| Route activation exposes an incomplete screen | Per-route HTTP activation only after focused live contract and direct-load pass |

## Dependencies

- Plan 1 route/capability/action contracts and deterministic scenarios.
- Plan 1 Tasks 1–12 must be complete and its contract handoff independently
  accepted; partial overlap with Plan 1 is not allowed.
- Existing Go authority, upload, OIDC, offline sync, and local profile foundations.
- Plan 3 consumes the provisioning, scan, email, and document job adapters.
- Plan 4 consumes health, metrics, job, migration, backup, and failure semantics.

## Out Of Scope

- Real provider-admin API provisioning, TOTP enrollment UI, ClamAV signatures,
  SMTP delivery, Gotenberg rendering, Caddy HTTPS, and consolidated Compose.
- Production identity federation, external model provider, legal retention,
  enforcement case automation, deployment, Terraform, Terragrunt, or AWS.
- Traffic cutover or root-demo removal.

## Execution Prompt

```text
Execute docs/exec-plans/completed/2026-07-22-full-backend-scenario-parity-plan.md only after the Full React 86-Screen Migration completes Tasks 1-12, reaches ready-for-verification, and its route/capability/action handoff is independently accepted. Do not overlap this plan with unfinished React-plan work. Do not dispatch subagents unless explicitly authorized. Work on the current branch and preserve unrelated docs/demo-evidence/stakeholder/ and outputs/ content.

Keep one Go modular monolith with separate API/worker processes, one generated OpenAPI contract, PostgreSQL module-owned stores, transactional idempotency/audit/change/outbox writes, and strict server authorization. Tasks 4-9 must each implement their own generated transport mapping and HttpBackend slice before running mock/HTTP transcript parity; Task 10 is the aggregate registry and route-activation gate. Implement every capability used by all 86 routes and activate a route in HTTP only after its focused live contract passes. Never fall back to mock in the HTTP artifact, and never expose test reset/seed routes in normal OIDC/full mode.

Preserve Potential Finding authority, immutable checklist/CAP/Evidence/report/document versions, Auditee organization isolation, Internal CAA Note separation, scan-clean Evidence gating, report decision authority, explicit closure basis, offline recovery, and advisory-only risk/assistant behavior. Run the same scenario implementations against MockBackend and HttpBackend and require exact normalized transcript parity.

Use the plan's defined task order and exact per-task commit messages only when Git actions are separately authorized. Inspect upstream, allowlist, cached names, full cached diff, and diff check before every commit. Do not start external platform-service integration, deploy, remove the root demo, or claim production readiness. Finish with synchronized evidence and stakeholder verification as the next todo.
```
