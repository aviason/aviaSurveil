# Department Manager AGA Inspection Package And Multi-Role Demo ExecPlan

This ExecPlan is a living document. Keep `Progress`, `Decision Log`,
`Discoveries`, and `Outcome` synchronized with actual work. Follow
[`docs/PLANS.md`](../../PLANS.md), the repository
[`AGENTS.md`](../../../AGENTS.md), and the literal evidence vocabulary in the
[`output contract`](../../agent-harness/output-contract.md).

## Global Constraints

1. This plan extends, but does not rewrite or weaken, the accepted boundaries
   of the
   [AGA Hybrid Question Classification And Synthetic Demo Lifecycle plan](2026-08-03-aga-hybrid-classification-demo-lifecycle-plan.md).
   That predecessor remains the authority for classification semantics,
   immutable identity, Draft disposition, deterministic recommendation,
   lifecycle state, authorization, and connected qualification.
2. The source package contains 1,310 candidate AGA question bodies. They remain
   `candidate-only`, `release pending`, and
   `production-ready: not established`.
3. The 1,310 bodies are not technically approved, published, legally
   authoritative, or source-owner attested. Every original question continues
   to carry the predecessor's governance truth, including
   `SOURCE_MAPPING_REQUIRED` and `NOT_ATTESTED` where applicable.
4. A Department Manager may review, filter, and select the candidate inventory
   without manually opening every row. A reviewed batch preview may disposition
   a bounded result set. The product must never silently interpret this as
   question-level approval or publication.
5. The selected subset may enter only a synthetic, local-preprod,
   `simulation-only` inspection package after an explicit Department Manager
   readiness confirmation. It must not enter the canonical Audit Plan,
   Checklist, Assignment, Finding, CAP, Evidence, Organization, or publication
   stores.
6. The full question bodies remain owned by the immutable
   `preprod_aga_demo` sealed overlay. The sibling
   `preprod_aga_demo_workspace` continues to store identities, digests,
   classifications, decisions, and genuinely new or reworded synthetic text
   only. Original bodies must not be copied into workspace tables.
7. The tagged API may compose an authorized response from the separate sealed
   overlay reader pool and workspace reader pool. It must not grant the
   workspace database roles overlay access, create a cross-schema join, or
   route either store through canonical application credentials.
8. Original question bodies must not enter URLs, browser history, Web Storage,
   IndexedDB, service-worker caches, telemetry, logs, retained screenshots,
   Playwright traces/videos, command evidence, or generated build artifacts.
   The browser may hold text for only the active bounded page in memory. It may
   retain redacted metadata for at most four pages, but cached metadata must not
   contain `questionText` or any equivalent body field.
9. Exact record, role, organization, route, version, identity, and digest
   boundaries remain fail-closed. An unauthorized or ambiguously bound actor
   receives a neutral denial with no question count, text, identifier, or
   existence signal.
10. CAP acceptance is not Finding closure. A normal synthetic closure still
    requires accepted CAP, accepted and verified Evidence, and the recorded CAA
    verification transition. The separately authorized closure path remains
    explicit and audit-recorded.
11. No visible control may be fake or toast-only. It must navigate exactly,
    perform a real candidate command, create a local synthetic artifact, or be
    disabled with a specific reason.
12. This plan authorizes no implementation, external upload, deployment,
    production mutation, commit, push, or branch operation. Begin execution
    only after the user explicitly authorizes implementation.

## Status

- Plan status: `paused` — the 2026-08-07 canonical AGA preprod successor is the
  sole implementation direction; preserve verified donor mechanics/evidence and
  do not continue the duplicate synthetic stakeholder lifecycle.
- Design request and implementation authorization: supplied by the user on
  2026-08-05.
- Gate 0 and Tasks 1–9: `verified locally`.
- Connected happy path and four-case fault matrix: `verified locally`.
- Release: `release pending`.
- Production readiness: `production-ready: not established`.
- Immediate next gate: no independent implementation step. Gate 0 of the
  canonical successor must classify and carry forward reusable donor/security
  work; no release or production action is authorized.

## Objective And User-Visible Outcome

Deliver an interactive local-preprod demo in which an exactly bound Department
Manager can:

1. open an AGA candidate inventory that truthfully reports all 1,310 sealed
   questions;
2. read the sealed candidate text through bounded server pagination;
3. search and filter by form, domain, topic, confidence, blocker, source gap,
   external involvement, and current disposition;
4. include, exclude, defer, reclassify, or batch-disposition an exact subset
   after seeing a count and representative preview;
5. inspect a package summary that shows included count, excluded count,
   deferred/blocking count, source gaps, simulation labels, target, provider
   scope, Inspector, and Lead Inspector bindings;
6. explicitly mark the complete Draft `READY_FOR_DEMO_SIMULATION` without
   approving or publishing any question;
7. create one deterministic immutable recommendation and release it into one
   synthetic inspection snapshot; and
8. hand the snapshot through the existing Inspector, Lead Inspector, Auditee,
   CAP, Evidence, CAA review, and closure screens.

The default presentation path should use a small, deliberately selected,
provider/target/profile-eligible subset so the lifecycle is easy to demonstrate.
The product must nevertheless prove that the Manager can reach and disposition
all 1,310 candidates through pagination. Only explicitly included questions
that are eligible for the exact synthetic provider, target, inspection profile,
inspection type, and qualifiers may enter one inspection. An included but
ineligible question must fail readiness/recommendation with no write; the
recommendation engine must never silently drop it. No question is automatically
included merely because it exists in the inventory.

## Product Truth And Language

The UI and evidence must distinguish these concepts:

| Concept | Required language | Forbidden inference |
|---|---|---|
| Source inventory | `1,310 candidate AGA questions` | 1,310 approved questions |
| Text projection | `sealed candidate text` | published checklist text |
| Manager action | `simulation disposition` | technical approval |
| Complete Draft | `ready for demo simulation` | ready for operations |
| Recommendation | `synthetic recommendation` | approved Audit Plan |
| Inspection | `synthetic inspection` | canonical/production inspection |
| Source gap | `source mapping required` | source verified |
| Release state | `candidate-only`, `release pending` | production-ready |

The Manager does not have to inspect each of the 1,310 rows one by one. Batch
actions are allowed only when the server first returns an exact, digest-bound
preview of the filter result and the Manager confirms that preview. This is a
simulation readiness decision over the Draft, not a legal, source, technical,
or publication decision over the original questions.

## Scope

### Included

- Authorized, paginated composition of sealed question text with the existing
  workspace classification projection.
- Department Manager inventory and package-builder UI.
- A completed server-side batch pipeline with exact filters,
  `INCLUDE`/`EXCLUDE`/`DEFER` selection actions, server-issued previews, and CAS
  protected atomic execution.
- Server-derived simulation setup facts and opaque binding pins for the one
  synthetic Aerodrome Operator fixture.
- Browser access to the already implemented readiness, recommendation, and
  inspection commands without asking the browser to invent authority facts or
  identifiers.
- Immutable inspection question snapshots and explicit handoff links to the
  existing multi-role lifecycle routes.
- A presentation-sized fixture selection and a repeatable local demo-session
  operator runbook.
- Focused unit, contract, authorization, artifact, browser, connected,
  isolation, idempotency, recovery, and cleanup verification.
- A durable evidence summary using only literal evidence labels.

### Explicitly Excluded

- Real NCAA/CAA source-owner attestation, legal interpretation, applicability
  confirmation, risk acceptance, technical approval, or publication.
- Canonical Audit Plan, Checklist, Assignment, Finding, CAP, Evidence, User,
  Organization, Provider, or Regulatory Library writes.
- Reusing the canonical New Audit Wizard or making the synthetic recommendation
  appear in normal operational queues.
- Production deployment, AWS validation, public hosting, real-user invitations,
  WhatsApp/Drive delivery, or any other external action.
- A permanent compatibility layer beside the current AGA workspace contract.
  Evolve the candidate-only contract directly and update all callers/tests.
- Client-side loading or persistence of all 1,310 bodies at once.
- AI reclassification or new model execution. This plan consumes the accepted
  deterministic classification artifact from the predecessor.
- Automatic selection, automatic readiness, automatic assignment, automatic
  finding creation, or automatic closure.
- Weakening media, log, telemetry, authorization, artifact, or database-role
  controls to make the demo easier.

## Assumptions, Owners, And Stop Conditions

### Assumptions

- The accepted sealed overlay contains exactly 52 forms and 1,310 candidate
  questions, and each workspace identity contains the matching package version,
  package JSON digest, form code, proposal ID, ordinal, and text digest.
- The existing local-preprod fixture supplies exactly one scoped Department
  Manager, Inspector, Lead Inspector, CAA Reviewer, Auditee, synthetic provider
  scope, and synthetic target for this demo generation.
- The existing classification Draft, deterministic recommendation domain, and
  lifecycle domain remain valid and are extended only where the browser lacks
  server-derived setup facts or an authorized text projection.
- The default demo subset is a presentation fixture, not a product policy.

### Owners

- Product/demo owner: chooses the presentation subset and approves the operator
  script/runbook wording.
- Department Manager fixture actor: performs simulation dispositions,
  readiness, recommendation, and synthetic inspection release.
- Inspector and Lead Inspector fixture actors: execute and review the synthetic
  checklist/finding path.
- Auditee fixture actor: submits CAP revisions and Evidence versions.
- CAA reviewer fixture actor: reviews CAP, verifies Evidence, and closes only
  through an authorized lifecycle transition.
- Source owner and real Department Manager: remain external and `blocked` for
  any real question approval/publication.

### Stop Conditions

Stop and record `blocked` for the affected work package if any of the following
is observed:

- the overlay receipt is not the accepted immutable 1,310-question package;
- any identity-to-text digest mismatch or duplicate full identity appears;
- an unauthorized role receives a count, text, identifier, or existence signal;
- response composition requires a workspace-role overlay grant, cross-schema
  join, or canonical credential;
- original text appears in workspace persistence, logs, browser persistence,
  build artifacts, or retained test media;
- provider scope, target, or lifecycle binding is missing, stale, ambiguous, or
  not exactly tied to the authenticated fixture actor;
- the Draft contains an undispositioned or blocking item at readiness time;
- an explicitly included question is ineligible for the selected provider,
  target, inspection profile/type, or qualifiers, or the recommendation would
  silently reduce the Manager's eligible selected set;
- a later Draft mutation changes an already created recommendation or inspection
  snapshot;
- the synthetic path writes to or becomes reachable from canonical workflow
  surfaces;
- the connected demo cannot prove exact cleanup of its disposable namespace and
  task-owned processes.

## Repository Orientation And Affected Interfaces

### Existing authorities to preserve

- `apps/api/internal/agacandidatedemo/` owns read-only access to the sealed
  `preprod_aga_demo` projection. Its `Question` already contains `Text` and
  `TextDigest`.
- `apps/api/internal/agademoworkspace/` owns authorization, classification,
  readiness, recommendation, and lifecycle service behavior.
- `apps/api/internal/preproddata/agademoworkspace/` owns the sibling workspace
  store and its least-privilege role matrix. Its reader and command roles must
  keep `OverlayAccess: false`.
- `apps/api/cmd/api/profile_preproddemo.go` is the only tagged local-preprod
  runtime composition root. Its AGA candidate and workspace services are
  currently separate factories: the candidate factory owns the overlay reader
  pool, while the workspace factory owns only workspace reader and command
  pools. No overlay pool is currently injectable into the workspace service.
- `api/openapi/source/schemas/platform.json` and
  `api/openapi/source/paths/platform.json` are contract sources. Generated
  OpenAPI and TypeScript artifacts must be regenerated, never hand-edited.
- `apps/web/src/backend/aga-demo-workspace.ts` and the HTTP backend expose the
  candidate-only query/command families with telemetry suppression.
- `apps/web/src/features/checklists/aga-classification-workspace-page.tsx`
  already provides bounded Manager pagination, filters, classification, Draft
  dispositions, and batch actions, but does not display original text.
- `apps/web/src/features/inspections/aga-demo-inspection-page.tsx` already
  renders the synthetic lifecycle and intentionally disables Manager release
  controls because the browser lacks exact server-derived setup pins.
- `apps/web/src/app/aga-demo-workspace-routes.tsx` owns the fixed role routes.
- `scripts/test-aga-hybrid-demo-workspace-connected.sh` is the accepted
  disposable connected qualification authority and must be extended or wrapped,
  not bypassed.

### Planned backend interfaces

1. Add a narrow `QuestionBodyResolver` interface to
   `apps/api/internal/agademoworkspace/`. It accepts a bounded list of complete
   sealed-base identities, never free-form client IDs or workspace-authored
   identities, and returns one exact body plus digest for every identity or
   fails the complete response.
2. Add a separate `QuestionTextSearchResolver` operation for non-empty normalized
   body search. It returns complete sealed-base identities only. The application
   layer intersects those identities with workspace metadata filters before
   deterministic pagination; there is no database cross-schema join. Search
   values remain transient and must not enter preview persistence, logs, or
   telemetry.
3. Give the workspace tagged runtime its own least-privilege overlay-reader
   pool, or refactor both tagged factories into one explicitly owned runtime
   bundle with a single close chain. In either design, test partial
   initialization and close-on-error. Do not treat the candidate service's
   current private pool as reusable, add overlay methods to the workspace store,
   or grant workspace roles overlay access.
4. Create a transport-only `ClassificationReviewItem` projection in
   `apps/api/internal/agademoworkspace/`. Do not add body fields to storage-owned
   `preprod.ClassificationItem` or any persisted/digested workspace type.
5. Evolve `SEARCH_ITEMS` directly so the authorized CAA Admin and exactly bound
   Department Manager transport projections include `questionText` and matching
   `questionTextDigest` only for the active page. The service, not a client flag,
   decides whether text is returned. Other roles must not receive either field.
6. Enforce a 25-row maximum for text-bearing queries. The UI may retain redacted
   metadata for at most four pages but text for only the active page. Search
   remains POST-body based and `Cache-Control: no-store` with telemetry
   suppression.
7. Add Manager-only `GET_SIMULATION_SETUP` as a read-only query. It returns the
   current provider/target labels, exact generation/Draft/taxonomy/run facts,
   eligible Inspector and Lead choices as opaque server-issued selection pins,
   current readiness/recommendation state, and a deterministic
   `simulationSetupDigest`. It produces no token, event, or persisted state and
   exposes no Auditee binding, secret, or canonical ID.
8. Require the Manager to explicitly choose one eligible Inspector and one
   eligible Lead from those server-returned choices. Readiness/recommendation/
   inspection commands consume current setup facts and the setup digest; the
   server generates `readinessEventId` at commit and re-resolves all authority
   facts. The browser invents no authority ID or binding revision.
9. Add mandatory `GET_CURRENT_RECOMMENDATION` and current-inspection resolution
   operations. The simplest candidate-only rule is one recommendation and one
   inspection per active generation/scope: exact idempotent replay returns the
   existing object, while a second non-replay creation conflicts. Zero or
   multiple current matches receive neutral denial.
10. Add a text-bearing lifecycle query such as
    `GET_INSPECTION_QUESTION_PAGE`. It resolves the immutable snapshot refs and
    transiently composes at most 25 bodies for the scoped Manager, assigned
    Inspector, and assigned Lead only. Auditee and an unassigned CAA Reviewer
    receive no question body. Command responses must never include composed
    bodies because command responses are persisted for idempotent replay.

The implementation may refine type names during Gate 0, but it must preserve
these authority, persistence, current-object, and transient-projection splits
and may not introduce a second compatibility contract.

### Planned frontend interfaces

- Evolve the existing classification workspace into an inventory-first table
  with a readable candidate-text column/detail disclosure and persistent
  simulation labels.
- Add the fixed Manager route:
  `/department-manager/aga-demo-workspace/inspection-package`.
- Add
  `apps/web/src/features/inspections/aga-demo-inspection-package-page.tsx` as a
  three-stage surface:
  `Inventory and disposition -> Package preview -> Simulation release`.
- Keep the existing Manager inspection route for the immutable released
  snapshot and the existing Inspector, Lead Inspector, and Auditee routes for
  lifecycle execution.
- Do not place question identities, text, search terms, or object IDs in query
  strings or route parameters.

## Detailed Interaction Contract

### Inventory and review

- The summary distinguishes the fixed `1,310 sealed candidate AGA questions`
  from the current Draft leaf count when workspace-added/reworded candidates
  exist. It also shows form count, disposition totals, source-gap totals, and
  simulation labels.
- The table initially loads 25 rows. Next/previous pagination and exact search
  allow the Manager to reach the complete inventory without preloading it.
- Each row shows sealed candidate text, form/ordinal, classification, confidence,
  governance/source-gap state, Draft disposition, and available actions.
- Body search resolves sealed-base identities in the overlay, intersects them
  with workspace filters before pagination, and also matches workspace-owned
  bodies through the workspace store. Search terms remain transient.
- New or reworded workspace candidates show the workspace-owned body and a
  distinct label. Original sealed bodies are composed transiently.
- A text/digest mismatch fails the page neutrally; it is never shown as a
  partial warning beside potentially incorrect text.
- Changing page, filter, role, subject/session, or BFCache state clears every
  body and selected-row text value before the next response is rendered. Only
  redacted metadata may remain in the four-page cache.

### Selection and batch disposition

- Row actions remain `Include`, `Exclude`, and `Defer` and preserve the current
  reclassification and proposal-resolution controls.
- The current batch engine is not a complete selection path: it supports only a
  main-domain filter, four non-selection actions, a 500-item cap, and no
  end-to-end workspace preview persistence/response. This plan must add closed
  server-side filter and action schemas covering all visible filters plus
  `INCLUDE`, `EXCLUDE`, and `DEFER`; append/read/consume preview storage;
  server-issued preview ID/digest; idempotency/CAS; and atomic execution.
- Batch actions operate on the exact server-side filter, not the browser's
  current 25 rows. The preview returns total match count, current-disposition
  counts, proposed changes, eligibility/blocker/source-gap counts, preview ID,
  preview digest, and expiry.
- Preserve the 500-item maximum. A result above 500 is not silently chunked and
  cannot execute; the UI requires the Manager to narrow the filter and may show
  a deterministic, repeatable sequence of smaller filters. Dispositioning all
  1,310 therefore requires multiple explicitly previewed/confirmed batches, not
  one `Include all` shortcut.
- The confirmation text explicitly states that the action is a simulation
  disposition and not technical approval/publication.
- Execution supplies the preview ID/digest and current Draft CAS pins. A stale
  filter, generation, Draft, or preview is rejected without partial mutation.
- Remove the current client-generated preview-ID path. The browser must never
  invent a preview ID or execute without the exact server preview digest. There
  is no client-generated preview ID in the successor contract.

### Package preview and readiness

- The package preview is server-derived from the current immutable identity and
  Draft state. It lists `eligible and included`, `included but ineligible`,
  excluded, deferred, unresolved blocker, and source-gap counts plus
  form/domain/topic distribution, provider/target, Manager-selectable eligible
  Inspector/Lead choices, and all governance disclaimers.
- The `Mark ready for demo simulation` control remains disabled until every
  question has an accepted disposition, all predecessor readiness invariants
  pass, the setup facts are exact/current, at least one included question is
  eligible, and `included but ineligible` is zero.
- Readiness creates an append-only event pinned to generation, Draft revision
  and digest, taxonomy, classification run, provider scope, target, and current
  actor binding. It does not change any question governance field.

### Recommendation and inspection release

- `Create synthetic recommendation` is enabled only after readiness. Its output
  is deterministic for the same pinned inputs and contains exactly the ordered
  eligible-and-included references. Any included ineligible reference fails the
  whole command; it is never silently filtered out.
- `Release synthetic inspection` is enabled only when the recommendation and
  role bindings remain current and the Manager has explicitly selected one
  server-returned eligible Inspector and Lead. The server validates and pins
  those choices and creates one immutable question snapshot.
- A later Draft/recommendation mutation never changes the released inspection.
- The UI presents role handoff links and a concise demo checklist, never a claim
  of canonical assignment or production release.

### Multi-role lifecycle

The presentation path is:

1. Department Manager selects the demo subset, confirms readiness, creates the
   recommendation, and releases the synthetic inspection.
2. Inspector starts the inspection, records responses, creates a potential
   finding from one non-compliant answer, and submits the checklist.
3. Lead Inspector returns or converts the potential finding; the happy path
   converts it into a Finding.
4. Auditee sees only its organization-scoped projection, submits CAP revision
   1, receives a CAA response, and submits the accepted revision if required.
5. The exact CAA Reviewer fixture, whose authenticated application role is
   `leadInspector` but whose workspace binding is `CAA_REVIEWER`, accepts CAP.
   The Finding remains open.
6. Auditee submits Evidence version 1.
7. The assigned Inspector or assigned Lead Inspector verifies Evidence with
   outcome `CLOSE`; that verification atomically closes the Finding with
   `EVIDENCE_VERIFIED` basis. A separate Manager `AUTHORIZED_CLOSE` is used only
   for the no-Evidence `PENDING_CLOSURE` branch, not this happy path. The history
   records every role and state transition.

Internal CAA Notes remain absent from the Auditee projection. Other
organizations, CAA workload, private risk scoring, and enforcement deliberation
remain excluded.

## Ordered Work Packages

### Gate 0 — Freeze The Successor Contract And RED Tests

Objective: convert this plan into an exact implementation contract before any
runtime behavior changes.

Work:

1. Inventory the predecessor's accepted query/command operations, role matrix,
   response projections, generation/Draft/recommendation pins, fixture bindings,
   and connected test phases.
2. Freeze the transport-only review item, body/search resolvers, text-bearing
   classification/lifecycle queries, active-page text cache, and no-persistence
   schemas.
3. Freeze the missing batch filter/action/preview/store/execute contract,
   including the 500-item limit and removal of client-generated preview IDs.
4. Freeze read-only `GET_SIMULATION_SETUP`, mandatory
   `GET_CURRENT_RECOMMENDATION`, current-inspection resolution,
   generation/scope uniqueness, server-generated readiness IDs, eligible role
   choices, and the fixed Manager package route.
5. Write failing tests for eligibility parity/no silent drop, Manager/Admin text
   access, assigned Inspector/Lead snapshot text, unauthorized omission,
   identity/digest mismatch, setup ambiguity, batch preview/consume, reload,
   second non-replay creation, immutable snapshot, and no canonical/artifact
   leak.
6. Record a Gate 0 decision entry and stop if any design would violate the
   predecessor's store or authority boundaries.

Acceptance:

- Contract tests fail only for the deliberately missing successor fields and
  operations.
- No implementation code or database grant is changed before the RED contract
  is recorded.
- The plan and index still describe the actual status as `not run` or the exact
  newly reached gate.

### Task 1 — Add Authorized Server-Side Question Text Composition

Objective: let the exact Manager read all 1,310 sealed bodies without copying
them into workspace persistence or broadening database privileges.

Work:

1. Add the transport-only review item plus body and body-search resolver
   interfaces with complete-identity request/response types.
2. Implement the tagged Postgres resolvers against only a dedicated
   least-privilege overlay reader pool and sealed projection.
3. Inject the resolver through an explicitly owned pool/close chain in
   `profile_preproddemo.go`; cover partial initialization and close-on-error.
   Normal/canonical builds continue to have no resolver or route capability.
4. Intersect body-search identities with workspace filters before pagination,
   without persisting the search term or creating a cross-schema join.
5. Compose each bounded workspace result with one exact sealed body or the
   workspace-owned new/reworded body. Verify the returned digest equals both the
   body hash and immutable identity `textDigest`.
6. Fail the complete response on missing, duplicate, ambiguous, or mismatched
   identities.
7. Enforce role projection, 25-row maximum, active-page-only text retention,
   `no-store`, telemetry suppression,
   and request/response logging redaction.
8. Prove all 1,310 identities are reachable exactly once across deterministic
   pagination and that no body is persisted in workspace tables.

Acceptance:

- CAA Admin and exact Department Manager receive correct bounded text.
- Inspector, Lead Inspector, Auditee, unrelated CAA roles, wrong organization,
  stale session, and ambiguous binding receive neutral denial or a text-free
  projection as defined by Gate 0.
- The role/grant matrix is byte-for-byte unchanged except for intentionally
  updated test metadata; `OverlayAccess` remains false for workspace roles.
- Partial initialization closes every already-opened pool exactly once and
  exposes no partially capable workspace service.

### Task 2 — Complete The Server-Side Batch Selection Pipeline

Objective: replace the current incomplete/client-invented batch path with a
closed, durable, atomic Manager selection workflow.

Work:

1. Extend the closed batch action enum with `INCLUDE`, `EXCLUDE`, and `DEFER` and
   retain the existing controlled reason requirements, including the
   simulation-only reason for source-gap inclusion.
2. Extend the batch filter schema to the exact visible Manager filters and
   canonicalize it before digesting. Page/cursor/client row state is never
   authority.
3. Wire `PreviewDraftBatch` and `ExecuteDraftBatch` through the workspace
   service and add append/read/consume preview store operations plus a closed
   preview response projection.
4. Generate preview ID/digest on the server; bind generation, run, Draft CAS,
   canonical filter digest, sorted affected-identity digest, action, reason,
   count, and expiry. Remove the UI's client-generated preview ID.
5. Recompute the exact set in the execution transaction and either append one
   complete successor Draft or write nothing. Preserve idempotent replay and
   consume semantics.
6. Keep the 500-item cap. Reject larger previews with a specific narrowing
   instruction; prove that sequential explicitly confirmed batches can
   disposition the fixed 1,310 inventory repeatably.

Acceptance:

- No batch execution is possible without an unexpired server-issued preview ID
  and matching digest/current Draft CAS.
- Selection batch actions preserve per-row governance and do not imply technical
  approval or publication.
- Stale, replay-conflicting, expired, over-500, changed-filter, or changed-set
  execution creates no partial Draft successor.

### Task 3 — Expose Server-Owned Simulation Setup And Release Pins

Objective: make the existing readiness, recommendation, and inspection commands
safely callable by the browser.

Work:

1. Implement Manager-only read-only `GET_SIMULATION_SETUP` with exact actor,
   provider scope, target, Draft, taxonomy, run, readiness, recommendation,
   eligible Inspector/Lead choices, and deterministic setup digest. It creates
   no token or state.
2. Update readiness so the server generates the readiness event ID and
   revalidates every setup fact at commit time.
3. Require explicit Manager selection of one server-returned Inspector and Lead
   choice for inspection creation; revalidate both binding pins at commit.
4. Implement mandatory `GET_CURRENT_RECOMMENDATION` and current-inspection
   resolution for reload without browser-persisted object IDs.
5. Enforce one recommendation and one inspection per active generation/scope.
   Exact idempotent replay returns the recorded result; a second non-replay
   creation conflicts.
6. Preserve idempotent replay and CAS conflict semantics; reject stale setup or
   bindings before any write.
7. Update OpenAPI sources, regenerate all artifacts, and update backend clients
   and contract tests.

Acceptance:

- The browser supplies no invented organization, provider, target, readiness,
  Inspector, or Lead authority fact, and Manager selection remains explicit.
- Exact replay returns the recorded response; changed inputs with the same
  idempotency key fail.
- Ambiguous or stale setup produces no readiness, recommendation, or inspection.
- Reload resolves the same current recommendation/inspection; zero, multiple,
  or historical-only matches fail neutrally.

### Task 4 — Make The 1,310-Question Manager Inventory Usable

Objective: present the full candidate inventory clearly and safely at desktop,
tablet, and mobile sizes.

Work:

1. Render question text and governance labels in the existing bounded table or
   an accessible row disclosure at 1440, 1024, and 390 CSS-pixel widths.
2. Preserve the current filters and add disposition filtering if Gate 0 confirms
   it is missing.
3. Show exact result and Draft counts, loading/error/empty states, pagination,
   an active-page-only text cache, and at most four redacted metadata pages.
4. Keep filter state in component memory; do not mirror sensitive search or text
   into URLs or browser storage.
5. Add accessible names, focus behavior, table/disclosure semantics, keyboard
   operation, visible status text, and overflow protection.
6. Update focused Vitest and route tests with realistic long question text.

Acceptance:

- The Manager can reach the first, middle, and last sealed questions and search
  a known body fragment.
- Network activity stays bounded and no route loads 1,310 bodies at once.
- There is no horizontal page overflow or console error at required viewports.

### Task 5 — Build The Inspection Package Preview And Release UI

Objective: turn an explicitly dispositioned subset into the existing immutable
synthetic inspection path.

Work:

1. Add the fixed package-builder route and three-stage UI.
2. Use the completed server-side batch pipeline for filtered disposition; show
   exact impact, cap results at 500, and require explicit confirmation.
3. Render eligible, included-but-ineligible, excluded, deferred, blocker, and
   source-gap counts. Block readiness on every failed invariant with a specific
   reason; never allow recommendation to silently reduce Manager selection.
4. Enable and wire `Mark ready for demo simulation`,
   `Create synthetic recommendation`, and `Release synthetic inspection` using
   only server-issued pins.
5. Handle conflict/reload/retry without duplicate events or snapshots.
6. After release, show immutable counts/digests, assigned role labels, and exact
   navigation into the Manager snapshot and role handoff screens.

Acceptance:

- The default demo subset can be created entirely through visible controls.
- No enabled action is fake, and every disabled action names the unmet invariant.
- A stale Draft or setup refreshes safely and does not partially release.

### Task 6 — Close The Interactive Multi-Role Handoff

Objective: ensure each fixture actor can open its role route and continue the
same synthetic inspection without shared hidden browser state.

Work:

1. Resolve the actor's only authorized current synthetic inspection server-side
   on fixed routes.
2. Add the authorized, transient, at-most-25-row inspection-question text query
   and render it for the scoped Manager, assigned Inspector, and assigned Lead.
   Keep bodies out of the lifecycle aggregate and command response.
3. Confirm Inspector and Lead Inspector projections contain the immutable
   eligible selected references, digest-matched transient text, and
   role-appropriate controls.
4. Confirm the Auditee sees no question body and only organization-scoped
   Finding, CAP, Evidence,
   and `Comment to Auditee` data.
5. Preserve complete lifecycle history and immutable version chains.
6. Add a compact in-product demo guide that labels the next role and action but
   does not reveal credentials or sensitive IDs.
7. Update focused role-route and lifecycle tests, including body pagination,
   digest equality, wrong-role/cross-scope denial, negative privacy, and
   authority cases.

Acceptance:

- One browser session per role can complete the happy path using visible routes
  and controls.
- CAP acceptance leaves the Finding open; accepted verified Evidence permits
  the recorded closure transition.
- Cross-role and cross-organization projections remain neutral and private.

### Task 7 — Prove Security, Privacy, Artifact, And Store Boundaries

Objective: demonstrate that the richer UI did not turn candidate text into a
new persistence or leakage channel.

Work:

1. Extend OpenAPI/Node boundary tests and demo/HTTP artifact scans.
2. Verify normal API, normal demo, HTTP, service worker, source map, manifest,
   log, telemetry, and error outputs contain no original body or fixture secret.
3. Inspect database role grants and confirm zero workspace-to-overlay access.
4. Compare canonical forbidden-system snapshots before and after the connected
   run and require zero delta.
5. Disable screenshots, traces, and videos for all connected text-bearing AGA
   tests; use DOM assertions and digest/count receipts instead.
6. Verify browser storage and Cache Storage contain no question body, search
   term, setup digest, or lifecycle secret after navigation and logout.

Acceptance:

- Privacy and artifact scans pass with exact forbidden needles from a controlled
  test fixture.
- Canonical delta is zero and original workspace-body persistence is zero.
- Unauthorized response bodies, logs, and test evidence contain no existence
  signal or candidate text.

### Task 8 — Qualify A Repeatable Demo Session And Connected Scenario

Objective: support both disposable automated qualification and a bounded manual
stakeholder presentation without weakening target-bound authority.

Work:

1. Extend the accepted connected harness or add a narrow wrapper with explicit
   `prepare`, `start`, `status`, and `stop` operations. Reuse its exact
   authorization, disposable namespace, isolated browser profile, and cleanup
   protocol.
2. `start` leaves only the explicitly named local demo session running; `status`
   reports generation, readiness, recommendation, inspection, and process state
   without question bodies or secrets; `stop` removes only task-owned resources
   and proves residue zero.
3. Add a connected Playwright scenario that verifies 1,310 inventory
   reachability and then uses a small deterministic eligible subset to complete
   the full role flow.
4. Cover stale Draft/setup, unauthorized text request, ambiguous binding, and
   mid-command replay fault cases.
5. Re-run the predecessor's zero-delta, grant, sealed-overlay, replay, cleanup,
   and residue checks.
6. Create `docs/demo-handoff/AGA_MANAGER_MULTI_ROLE_DEMO_RUNBOOK.md` with start,
   login-role order, presentation script, expected labels, recovery, and stop
   steps. Do not include credentials or external links.

Acceptance:

- Automated connected happy path and fault matrix pass with media retention off.
- A manual session can be started, presented across Manager, Inspector, Lead,
  CAA Reviewer, and Auditee fixture actors, inspected with `status`, and stopped
  without rebuilding product state by hand.
- Final cleanup proves no task-owned Compose, browser, Playwright, Vite, API,
  secret, authorization, namespace, or database residue.

### Task 9 — Aggregate Verification, Evidence, And Handoff

Objective: run the smallest complete verification gate and publish a truthful
local evidence record.

Work:

1. Run focused Go, contract, frontend, build, artifact, boundary, and E2E
   discovery checks.
2. Run the connected happy path and fault matrix once against a fresh disposable
   namespace.
3. Run final whitespace, documentation-reference, git status, process, and
   residue inspections.
4. Create a dated evidence summary under `docs/demo-evidence/` containing only
   commands, literal outcomes, immutable receipt digests, known limitations,
   and the demo operator entry point.
5. Update this plan's progress/decisions/discoveries/outcome and the plan index.

Acceptance:

- Every required gate has fresh evidence and an exact outcome.
- The final claim is no stronger than
  `interactive local-preprod multi-role AGA demo; verified locally`.
- The result remains `candidate-only`, `release pending`, and
  `production-ready: not established`.

## Verification Commands And Expected Observations

Run focused commands first and stop at the first unexplained failure. Exact file
lists may be refined in Gate 0, but the final gate must cover every affected
surface.

```bash
go -C apps/api test -count=1 ./internal/agademoworkspace ./internal/preproddata/agademoworkspace ./internal/httpapi
```

Expected: focused workspace, authorization, composition, recommendation, and
lifecycle tests pass; test discovery is nonzero.

```bash
go -C apps/api test -count=1 -tags=preproddemo ./internal/agacandidatedemo ./internal/agademoworkspace ./internal/httpapi ./cmd/api
```

Expected: tagged resolver wiring and local-preprod route tests pass while normal
runtime capability remains absent.

```bash
./scripts/generate-contracts.sh
npm --prefix apps/web run contracts:check
node --test api/openapi/tests/aga-demo-workspace-contract.test.mjs
```

Expected: generated OpenAPI and TypeScript artifacts match sources; new
operations/fields are closed-schema validated and no hand-edited drift remains.

```bash
npm --prefix apps/web run typecheck
npm --prefix apps/web test -- src/backend/aga-demo-workspace.test.ts src/backend/http-backend.test.ts src/auth/aga-demo-workspace-guard.test.tsx src/app/aga-demo-workspace-routes.test.tsx src/features/checklists/aga-classification-workspace-page.test.tsx src/features/inspections/aga-demo-inspection-package-page.test.tsx src/features/inspections/aga-demo-inspection-page.test.tsx
```

Expected: typecheck and focused UI/backend tests pass with nonzero discovery,
including long text, batch preview, readiness/release, role handoff, and negative
privacy cases.

```bash
npm --prefix apps/web run build:demo
node apps/web/scripts/assert-aga-workspace-artifact-boundary.mjs --profile demo --artifact apps/web/dist
npm --prefix apps/web run build:http
node apps/web/scripts/assert-aga-workspace-artifact-boundary.mjs --profile http --artifact apps/web/dist
node apps/web/scripts/assert-http-artifact.mjs apps/web/dist
```

Expected: demo and HTTP builds succeed. Neither artifact contains AGA candidate
bodies, fixture secrets, workspace endpoints in an unauthorized build, or mock
state in the HTTP artifact.

```bash
node --test tests/aga-hybrid-demo-workspace-boundary.test.mjs tests/demo-boundary-smoke.test.js
npm --prefix apps/web run test:e2e:aga-preprod -- --list
```

Expected: boundary tests pass and the dedicated Playwright project discovers a
nonzero Manager package and multi-role scenario set with screenshot, trace, and
video retention disabled.

```bash
node --test apps/web/scripts/assert-aga-workspace-artifact-boundary.test.mjs
npm --prefix apps/web test
node --test tests/*.test.js
```

Expected: the artifact scanner's own tests, full frontend regression, and full
root Node regression pass with nonzero discovery after the focused checks.

```bash
bash scripts/test-aga-manager-multi-role-demo-connected.sh
```

Expected: a fresh disposable connected run proves the exact 1,310 inventory,
small deterministic eligible demo subset, no silent recommendation drop,
immutable recommendation/inspection, complete happy lifecycle, negative fault
matrix, zero canonical delta, unchanged grants, and zero final residue. If Gate
0 elects to extend the existing connected script instead of adding this wrapper,
record and use that exact command here.

```bash
node tests/harness-docs-smoke.test.js
rg -n "docs/agent-harness|agent-harness/index|output-contract|verification-matrix|entropy-cleanup" AGENTS.md MANIFEST.md docs
git diff --check
```

Expected: documentation references, harness authority reference scan, and
whitespace pass. Run the harness smoke whenever this plan changes harness
documentation or registry entries.

Browser verification must use an isolated profile and must end with inspection
and cleanup of task-owned Chrome/Chrome Helper, Playwright, webdriver, Vite, API,
and Compose processes.

## Acceptance Criteria

The plan is complete only when all statements below are backed by fresh local
evidence:

1. The exact Department Manager can reach all 1,310 candidate questions through
   bounded pagination and read the correct sealed text for each full identity.
2. Every returned original body hash equals the response digest and immutable
   workspace identity text digest.
3. No request loads more than 25 original bodies; the UI retains body text for
   only the active page and no more than four redacted metadata pages.
4. Unauthorized roles and organizations receive no body, count, identifier, or
   existence signal.
5. The Manager can select a subset using row or server-issued digest-bound batch
   actions. Every batch is at most 500 items, and no candidate is implicitly
   approved or automatically selected.
6. Readiness is possible only for a complete valid Draft and records
   `READY_FOR_DEMO_SIMULATION`, never technical approval/publication. Any
   included-but-ineligible question fails the operation with no write.
7. Recommendation and inspection creation consume server-derived current scope,
   target, role binding, generation, Draft, taxonomy, and run pins.
8. The released inspection contains an immutable ordered snapshot of exactly the
   explicitly included and eligible set. The recommendation engine silently
   drops nothing, and later Draft changes do not alter the snapshot.
9. The assigned Inspector and Lead read digest-matched question text through a
   transient at-most-25-row lifecycle query; an Auditee or unassigned reviewer
   receives no question body.
10. Manager, Inspector, Lead Inspector, CAA Reviewer, and Auditee complete the
    synthetic lifecycle using role-appropriate projections and visible controls.
11. CAP acceptance does not close the Finding; assigned Inspector/Lead Evidence
    verification with `CLOSE` outcome closes the normal happy-path Finding.
12. Original bodies never enter workspace persistence, browser persistence,
    telemetry, logs, artifacts, or retained test media.
13. Workspace roles retain zero overlay access and the normal/canonical runtime
    exposes no candidate capability.
14. Tagged pool initialization/cleanup is exact, including partial-failure
    close-on-error, and canonical forbidden systems have zero connected-run
    delta.
15. Reload returns the same current recommendation/inspection, while a second
    non-replay creation or ambiguous current lookup fails with no write.
16. The UI is usable without page overflow or console errors at 1440, 1024, and
    390 widths.
17. The demo runbook starts, explains, recovers, and stops an isolated manual
    session without credentials or production claims.
18. Final residue inspection is zero and the evidence label remains
    `verified locally`, not `production-ready`.

## Risks And Mitigations

| Risk | Mitigation |
|---|---|
| Question text is copied into the writable workspace | Compose transiently through a narrow resolver; scan workspace tables and artifacts for controlled text needles. |
| Identity and body drift produce the wrong visible question | Resolve by complete immutable identity and require body hash = overlay digest = workspace identity digest. |
| Manager batch-selects unseen rows accidentally | Require exact count/digest preview, explicit confirmation, CAS, and clear simulation-only language. |
| Current batch support is mistaken for a complete selection pipeline | Implement closed filters/actions, preview storage/response, server IDs/digests, atomic execution, and remove client-generated IDs before enabling batch selection. |
| A selected question is silently removed as provider/target/profile-ineligible | Show eligible/ineligible counts and fail readiness/recommendation on any included ineligible reference. |
| 1,310 bodies overload or leak through the browser | Server page at 25; keep text only for the active page and at most four redacted metadata pages; no browser persistence or media retention. |
| Body search bypasses workspace filters or persists sensitive terms | Resolve overlay identities separately, intersect before pagination, and prohibit search terms in previews, persistence, logs, or telemetry. |
| Candidate classification is mistaken for governance approval | Keep source, authority, risk, decision, and release labels visible; never mutate them during readiness. |
| Client invents authority facts to enable disabled controls | Return a read-only setup digest and eligible choices, generate readiness IDs server-side, and revalidate every fact at commit. |
| Stale Draft or bindings create a misleading inspection | Pin every command to current revisions/digests and fail before writes. |
| Inspector/Lead execute keys without readable question text | Add assigned-role transient lifecycle text pagination while keeping bodies out of aggregates and command responses. |
| Tagged workspace composition leaks or double-closes pools | Give the resolver an explicit least-privilege pool ownership/close chain and test partial initialization. |
| Reload loses the current synthetic object or selects an ambiguous one | Require current recommendation/inspection queries and one-per-generation/scope uniqueness. |
| Synthetic records appear in canonical queues | Keep tagged routes/stores and fixed AGA workspace navigation; assert zero canonical delta and artifact separation. |
| Auditee receives internal CAA material | Maintain separate projections and negative field-level tests. |
| CAP acceptance is displayed as closure | Assert lifecycle states and require accepted verified Evidence for normal closure. |
| Manual demo leaves data/process residue | Provide explicit status/stop operations and exact task-owned cleanup verification. |
| Presentation subset is confused with product policy | Label it as a deterministic demo fixture and permit other explicit subsets under the same invariants. |
| Existing dirty worktree is overwritten | Inspect status/diff before edits; preserve unrelated user files and generated evidence. |

## Dependencies

- Accepted immutable 1,310-question sealed overlay and its reader role.
- Accepted AGA workspace generation, taxonomy, classification run, Draft,
  recommendation, lifecycle, and fixture bindings from the predecessor.
- Separate local-preprod overlay reader, workspace reader, and workspace command
  credentials, with an explicit additional overlay-reader pool or unified tagged
  ownership bundle for workspace response composition.
- Existing tagged OIDC/session fixture and isolated Playwright project.
- Existing connected target-bound authorization and cleanup protocol.
- Explicit user authorization before implementation begins.

External source-owner and real Department Manager approvals are deliberately not
dependencies for this synthetic plan. Their absence prevents real use,
publication, or production claims but does not prevent a truthfully labeled
local simulation.

## Idempotence And Recovery

- Text queries are read-only, bounded, `no-store`, and safe to repeat against the
  same immutable generation. Any composition mismatch fails the whole response.
- Batch previews are side-effect free and digest-bound. Batch execution is
  idempotent, capped at 500, and CAS-protected; a stale preview is discarded and
  regenerated. Dispositioning more than 500 rows requires separately confirmed
  deterministic batches.
- Readiness, recommendation, inspection, and lifecycle commands use server-owned
  IDs plus idempotency keys. Exact replay returns the stored result; key reuse
  with changed inputs fails.
- `GET_SIMULATION_SETUP` is read-only and creates no token or state. Its digest
  is a comparison pin; commit commands re-resolve facts and generate event IDs.
- A created recommendation and inspection are immutable snapshots. Recovery
  reloads the one current object through mandatory authorized query operations
  rather than reconstructing it from client state. A second non-replay object
  for the same active generation/scope conflicts.
- Failure before a transaction commit creates no event. Failure after commit is
  recovered by idempotent replay and the stored command receipt.
- A nonterminal lifecycle is not silently reset. Reset remains the explicit
  Admin-only synthetic operation governed by predecessor constraints.
- Automated connected qualification always cleans its disposable namespace.
  Manual demo `start` may retain only its named session until explicit `stop`;
  `status` must make that retained state visible and `stop` must prove zero
  residue.
- Cleanup targets only exact task-owned resources. Never use broad destructive
  paths, unresolved globs, or canonical database cleanup.

## Progress

- [x] 2026-08-05: User requested a concrete implementation plan for Department
  Manager access to the 1,310 candidates and an interactive multi-role demo.
- [x] 2026-08-05: Repository planning contract, architecture, verification
  matrix, output contract, demo scenario, current AGA plan, route/UI, OpenAPI,
  tagged API composition, and sealed reader boundaries inspected.
- [x] 2026-08-05: Successor scope and non-negotiable simulation/governance
  distinction frozen in this plan.
- [x] 2026-08-05: Independent read-only Sol Ultra review completed; three
  Critical and six Important findings were validated against current code and
  incorporated into the plan. No runtime implementation was performed.
- [x] 2026-08-05: Sol Ultra follow-up closure review confirmed all nine findings
  are materially resolved and found no residual or newly introduced Critical or
  Important issue.
- [x] Gate 0: contract and RED tests — frozen successor contract and deliberate
  RED tests recorded; implementation was then completed against that boundary.
- [x] Task 1: authorized text composition — bounded, digest-matched,
  fail-closed transient composition passed focused Go, HTTP, and frontend
  checks.
- [x] Task 2: complete server-side batch selection pipeline — server-issued,
  digest-bound previews and atomic 500-item-capped execution passed focused
  unit, API, and connected Manager checks.
- [x] Task 3: server-owned setup/release pins and current-object recovery —
  read-only setup, server-issued readiness, current recommendation/inspection
  reload, replay, CAS, and uniqueness protections passed.
- [x] Task 4: Manager inventory UI — all 1,310 candidates were reached through
  53 bounded pages with exact identity/body digest checks.
- [x] Task 5: package-builder/release UI — deterministic eligible subset,
  readiness, recommendation, and synthetic inspection release passed with
  visible controls and no silent drop.
- [x] Task 6: transient lifecycle question text and multi-role handoff — the
  Manager → Inspector → Lead Inspector → CAA Reviewer → Auditee → CAP →
  Evidence → closure flow passed with CAP/closure separation.
- [x] Task 7: privacy/store/artifact qualification — contract, role, pool,
  persistence, artifact, media-retention, and canonical-delta boundaries passed.
- [x] Task 8: connected/manual demo session qualification — fresh connected
  happy-path and four-case fault-matrix receipts passed with zero residue.
- [x] Task 9: aggregate verification and handoff — focused, tagged, contract,
  build, artifact, browser discovery, full Vitest, full Node, documentation,
  evidence, and cleanup gates passed.

## Decision Log

### 2026-08-05 — Freeze the successor transport and RED boundary before runtime work

Gate 0 is accepted as a contract-only step. The new RED tests require bounded
sealed-base text composition, transport-only review items, server-issued
digest-bound batch previews, read-only setup/current-object queries, server-owned
readiness IDs, and transient assigned-role lifecycle text. The focused Node
contract test fails because the successor OpenAPI operations and schemas are not
implemented; the focused Go test fails because the successor interfaces and
projections do not yet exist. No runtime code, database grant, generated
contract, or canonical surface was changed before recording this RED result.

### 2026-08-05 — Allow reviewed batch disposition, not unreviewed real use

The Manager does not need to open all 1,310 rows individually. Exact filtered
batch previews are a valid simulation review mechanism. They do not establish
source authority, technical approval, publication, or real operational use.

### 2026-08-05 — Compose bodies at the tagged API boundary

Original bodies stay in the sealed overlay. The tagged API composes bounded,
authorized text responses through explicitly owned separate reader pools.
Workspace roles receive no overlay grant and workspace persistence receives no
original body copy.

### 2026-08-05 — Use Draft disposition as the package selection source

The existing append-only Draft `Include`/`Exclude`/`Defer` contract remains the
single source of selection truth. A separate client-only selection model would
create drift and unsafe reload behavior.

### 2026-08-05 — Keep server ownership of release facts

The currently disabled Manager controls must be enabled by server-derived setup,
readiness, provider, target, and binding pins. The browser must not invent these
facts merely to make the demo clickable.

### 2026-08-05 — Demonstrate a small subset while proving full inventory access

The presentation uses a concise deterministic subset so viewers can understand
the complete lifecycle. Separate pagination/digest evidence proves access to all
1,310 candidates. The fixture subset is not a hard product limit.

### 2026-08-05 — Preserve fixed role routes without leaking object IDs

Each authenticated fixture actor resolves its only authorized synthetic record
server-side. Inspection or Finding identifiers do not enter URLs, browser
history, or cross-role handoff text.

### 2026-08-05 — Fail on included ineligible questions instead of silently filtering

The existing recommendation engine applies provider/target/profile eligibility
after `INCLUDE`. For this Manager-authored package, a mismatch between explicit
eligible selection and recommendation output would be misleading. Preview must
show the mismatch and readiness/recommendation must fail atomically until it is
resolved.

### 2026-08-05 — Treat batch selection as new backend work

The current batch domain does not support selection actions end to end, and the
UI invents a preview ID. This plan now requires a complete server-issued,
persisted, digest-bound preview/consume pipeline with a retained 500-item cap.

### 2026-08-05 — Use a read-only setup digest, not a query-issued token

`GET_SIMULATION_SETUP` remains side-effect free. The Manager selects from
server-returned eligible Inspector/Lead choices; the command re-resolves current
facts and generates the readiness event ID at commit.

### 2026-08-05 — Compose lifecycle question text outside persisted aggregates

The assigned Inspector and Lead need readable question bodies to execute the
checklist. Their authorized bounded query composes text transiently from
immutable snapshot refs. Aggregates and idempotency-persisted command responses
continue to contain refs/digests only.

### 2026-08-05 — Accept the independent Sol Ultra plan review

The user requested an independent `gpt-5.6-sol` review at `ultra` reasoning.
Three Critical and six Important findings were checked against the repository:
eligibility parity, incomplete batch support, lifecycle text, query-token
semantics, body search, current-object reload, transport/cache separation,
tagged pool ownership, and verification commands. All nine were incorporated;
the follow-up closure review confirmed no residual or newly introduced Critical
or Important finding.

### 2026-08-05 — Implement the frozen successor contract

The user-authorized implementation kept original bodies in the sealed overlay,
added explicit tagged reader-pool ownership, and composed only bounded
digest-matched text for authorized Manager/Admin and assigned Inspector/Lead
queries. Workspace persistence continues to store refs, digests, and metadata,
not original bodies. Unauthorized and Auditee projections remain neutral and
fail closed.

### 2026-08-05 — Make selection, setup, and current objects server-owned

The Manager now uses server-issued digest-bound batch previews with an atomic
500-item cap, explicit selection actions, and CAS protection. Setup is
read-only, readiness IDs are generated at commit, and recommendation/inspection
queries are mandatory for reload and generation/scope uniqueness. Included but
ineligible questions fail without a write; the recommendation path never
silently drops them.

### 2026-08-05 — Record connected happy and fault evidence

Fresh connected evidence passed the 14-phase happy path with 17 browser tests
and the four-case receipt-gap fault matrix. The finalizer recorded 10 lifecycle
commands and `EVIDENCE_VERIFIED` closure; final residue and canonical delta
were zero. The immutable ledger digests and privacy-safe summary are recorded
in [dated evidence](../../demo-evidence/AGA_MANAGER_MULTI_ROLE_DEMO_2026-08-05.md).

### 2026-08-05 — Keep plan lifecycle separate from local proof

All implementation and verification gates required by this historical execution
are `verified locally`. The plan is now `paused` in favor of the canonical
successor; the retained local result does not imply release or production
readiness and must not be extended as a duplicate stakeholder lifecycle.

## Discoveries

### 2026-08-05 — The lifecycle backend already implements the core commands

The predecessor already implements
`MARK_READY_FOR_DEMO_SIMULATION`, `CREATE_RECOMMENDATION`,
`CREATE_INSPECTION`, and the complete Finding/CAP/Evidence lifecycle. The main
successor gaps are safe classification/lifecycle body projection, a real batch
selection pipeline, eligibility parity, and browser-consumable server setup
pins—not a second lifecycle engine.

### 2026-08-05 — The current Manager table intentionally omits original text

The classification page already paginates at 25 rows and maintains a bounded
cache, but shows identities and classifications rather than sealed question
bodies. This plan preserves that bounded behavior while adding transient text.

### 2026-08-05 — Tagged service factories do not currently share the overlay pool

The tagged composition root has separate candidate-service and workspace-service
factories. The candidate factory privately owns the overlay pool; the workspace
factory cannot reuse it. Implementation needs an explicitly owned additional
least-privilege overlay pool or one unified tagged runtime bundle, including
partial-failure cleanup tests.

### 2026-08-05 — Manager release controls are disabled for a valid reason

The current UI lacks exact server-derived scope, readiness, recommendation, and
binding pins. Enabling the buttons without closing that gap would manufacture a
demo action and violate the visible-control rule.

### 2026-08-05 — Existing batch and recommendation behavior is not package-safe

The current batch domain accepts only four non-selection actions, only a
main-domain filter, and at most 500 items; the workspace service/UI do not carry
a server-issued preview end to end. Separately, the recommendation engine may
filter included rows for provider/target/profile eligibility. Both behaviors
must become explicit, fail-closed package contracts before the Manager demo is
truthful.

### 2026-08-05 — Existing lifecycle projections expose refs/keys, not bodies

The current inspection snapshot correctly stores immutable refs/digests, but the
Inspector UI shows a question key rather than readable sealed text. Execution
therefore needs a separate assigned-role transient text query, not body storage
inside the lifecycle aggregate or command response.

### 2026-08-05 — Empty collections must serialize as arrays

Connected HTTP projections exposed empty Go slices as JSON `null`, which caused
the browser to fail while rendering the first lifecycle page. Runtime
projections now preserve empty arrays explicitly; a focused regression test
locks that transport boundary.

### 2026-08-05 — CAP revisions can share a revision number

The append-only CAP flow records multiple states under the same revision. The
client now selects the latest append order rather than sorting only by revision,
so `PENDING_CAA_REVIEW` remains actionable and CAP acceptance cannot be mistaken
for closure.

### 2026-08-05 — The deterministic demo subset replaces initial fixture includes

The sealed source Draft contains initial `INCLUDE` rows. The connected Manager
scenario explicitly excludes both `UNSET` and initial `INCLUDE` rows before
including the small eligible presentation subset, proving all 1,310 candidates
are dispositioned without automatic selection.

## Outcome

Execution outcome as of 2026-08-05:

- Gate 0 and Tasks 1–9 are implemented and `verified locally` against the
  frozen successor contract. The Manager can reach all 1,310 sealed candidate
  questions through 53 bounded pages, inspect digest-matched transient text,
  and disposition an explicit deterministic subset without treating it as
  approval or publication.
- The package-builder and tagged API keep provider/target/profile eligibility
  fail closed, use server-owned setup/readiness/release facts, and preserve
  immutable recommendation and inspection snapshots. Assigned Inspector and
  Lead text is transient and bounded; Auditee and unauthorized projections do
  not expose bodies, counts, or identity signals.
- The connected happy path completed the Manager → Inspector → Lead Inspector →
  CAA Reviewer → Auditee → CAP → Evidence → closure flow. CAP acceptance stayed
  separate from Finding closure, and the normal happy path closed only after
  assigned-role Evidence verification with `CLOSE`.
- The four-case fault matrix recovered receipt gaps with no duplicate effects;
  canonical delta, overlay delta after seal, browser privacy leaks, and final
  task-owned residue were all zero. The runbook and dated machine-readable/text
  evidence are written as workspace deliverables without source bodies or
  credentials.
- Aggregate verification passed: focused and tagged Go, generated OpenAPI and
  closed-schema contract tests, focused frontend/typecheck, demo/HTTP builds and
  artifact scans, boundary/discovery, artifact scanner, full Vitest (90/753),
  full root Node (103/103), harness documentation, whitespace, and cleanup.
- The result is exactly `interactive local-preprod multi-role AGA demo;
  verified locally`; it remains `candidate-only`, `release pending`, and
  `production-ready: not established`. Plan lifecycle is `paused`; donor
  classification and future implementation continue only in the canonical
  successor.

## Execution Prompt

Do not execute this plan independently while it is `paused`. After the user
explicitly authorizes canonical successor implementation, execute
`2026-08-07-canonical-aga-preprod-end-to-end-product-plan.md` from Gate 0 on the
current branch and use this document as donor/historical evidence. Read the
predecessor plan and every
authority named in `Repository Orientation` before changing runtime code. Keep
this plan and `docs/exec-plans/index.md` synchronized after every accepted gate.
Use test-first contract changes, preserve unrelated worktree changes, regenerate
derived contracts from source, and stop on any Global Constraint or Stop
Condition. Do not commit, push, deploy, upload, or change branches without a
separate explicit user instruction. Finish only after focused and connected
verification, zero-residue cleanup, and a literal evidence handoff.
