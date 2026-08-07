# Canonical AGA Preprod End-To-End Product ExecPlan

This ExecPlan is a living document. Keep `Progress`, `Decision Log`,
`Discoveries`, and `Outcome` synchronized with actual work. Follow
[`docs/PLANS.md`](../../PLANS.md), the repository
[`AGENTS.md`](../../../AGENTS.md), and the literal evidence vocabulary in the
[`output contract`](../../agent-harness/output-contract.md).

## Status

- Plan status: `active` — implementation authorized; Gate 0 is in progress.
- Planning authority: the user requested this successor plan on 2026-08-07.
- Independent plan review: Sol Ultra final closure `ACCEPTED` on 2026-08-07;
  Critical `0`, Important `0`.
- Implementation authority: granted by the user's 2026-08-07 execution request
  for Gate 0, Tasks 1–9, and Task 11 only. Task 10 remains separately
  unauthorized and `not run`.
- Local full-system qualification: `not run`.
- External preprod deployment: `not run` and separately authorized.
- Release: `release pending`.
- Production readiness: `production-ready: not established`.
- Immediate next gate: complete the recorded Gate 0 RED contract and
  specification synchronization without changing branches, committing,
  pushing, deploying, or mutating an external environment.

## Objective

Replace the current stakeholder-facing AGA synthetic workspace with one clear,
canonical preprod product journey. A Department Manager creates a new Audit
draft, selects an exact subset from the 1,310-question AGA catalog, and submits
the scope and resource requirement through the accepted approval chain. After
GM Release, the Department Manager assigns the Lead Inspector; the Lead
Inspector assigns the team and exact question coverage; and the Department
Manager confirms preparation before an immutable non-executable canonical Audit
is materialized. Inspector, Lead Inspector, Auditee, CAA review, CAP, Evidence,
report, closure, and dashboard work then continue in the existing canonical
system instead of a second AGA-only lifecycle.

The 1,310 imported questions remain non-authoritative preprod exercise data
until their real source, applicability, technical-review, and publication gates
are satisfied. Their use in a preprod exercise must never be described as real
regulatory approval or operational readiness.

## User-Visible Outcome

The primary presentation path is no longer a screen catalog or a separate AGA
classification product. It is one task-based workflow:

1. The Department Manager lands on a compact work queue and selects
   `New Audit`.
2. `Set up` captures the organization, provider scope, regulated target, Audit
   type, announced/unannounced policy, purpose, date, mode, and location.
3. `Choose questions` searches and filters the complete 1,310-question catalog,
   lets the Manager add or remove exact questions, supports server-previewed
   bounded batch selection, and keeps a visible selected-question tray and
   count.
4. `Review and submit` shows the exact selected count, form/domain distribution,
   catalog version, selection digest, estimated resource requirement, budget,
   notice policy, and approval path before submission.
5. Finance, General Manager, Executive Director, and General Manager Release
   use their existing role queues. Every decision remains bound to the same
   immutable Audit-scope digest.
6. After GM Release, the Department Manager assigns the Lead Inspector. The
   Lead Inspector assigns the inspection team and gives every selected question
   explicit team-member coverage. A scope or approved-resource change returns
   through Planning; personnel preparation alone never mutates the released
   question snapshot.
7. The Department Manager confirms preparation and creates the canonical Audit
   workspace. The Audit and checklist are `SCHEDULED`/`NOT_STARTED` or awaiting
   announced-inspection coordination; they are not silently executable or
   marked `IN_PROGRESS`.
8. An authorized Inspector starts and completes the selected checklist. A non-compliant
   answer may create a Potential Finding, and only Lead conversion creates the
   canonical Finding.
9. After field execution, Preliminary Report approval and issue precede any
   required Auditee CAP/Evidence response. The Auditee then submits CAP and
   Evidence through its organization-scoped portal.
   CAP acceptance leaves the Finding open; accepted, verified Evidence closes
   the normal happy-path Finding.
10. Final Report approval/issue follows closure; manager queues, dashboards,
    and audit history update from the same canonical records.

The Manager should be able to explain the product from this one path without
opening technical hashes, classification internals, synthetic lifecycle setup,
or duplicate AGA-only routes.

The current question-review experience shown in the accepted screenshots is a
separate retained product surface, not disposable demo chrome. Department
Manager `Checklist Management -> Question Review` keeps the full
`Find -> Compare -> Decide` composition: summary cards, server-side search and
filters, 25-row bounded queue, question/metadata comparison, right-side
Decision file, collapsed technical identifiers, governance signals,
controlled reason, `Retain`/`Include`/`Exclude`/`Defer`, domain
reclassification, and topic actions. The same visual shell is backed by two
explicit canonical modes. Governed candidate review records Draft review/
classification decisions and can hand an eligible successor to the separately
authorized technical-approval and publication stages. Exercise review records
only exercise disposition and classification facts; technical approval/
publication controls are disabled with the source/lineage reason and no
exercise record can enter the governed publication chain. Neither mode may
imply that opening or classifying a question approves it.

New Audit reuses the same visual language and queue/dossier interaction, but
with a smaller purpose-specific action set: find questions, compare them, and
add/remove/defer them for one Audit scope. Audit selection never replaces the
full Question Review workspace and never performs classification, technical
approval, or publication.

## Success Definition

The product succeeds when a stakeholder can use an isolated preprod instance
from sign-in to closure and understand the system without implementation help.
Specifically:

- all 1,310 questions are imported with exact identity and digest reconciliation;
- the New Audit flow is dynamic rather than bound to fixed IDs, one organization,
  or one six-question template;
- the Department Manager can find, select, and review questions without loading
  all question bodies at once, then post-release Lead/team preparation gives
  every selected question explicit assignment coverage;
- approval and release preserve the exact selection snapshot;
- one canonical Audit drives checklist, Potential Finding, Finding, CAP,
  Evidence, reports, closure, and dashboards;
- the normal HTTP application has no fallback to mock state;
- the old AGA-only lifecycle is removed from the stakeholder route after
  verified cutover; and
- local and any later external-preprod claims remain literal and do not imply
  production readiness.

## Scope

### Included

- A versioned preprod AGA catalog containing the exact 52 form identities and
  1,310 question versions already proven by the sealed AGA intake package.
- Exact import lineage: package version, package JSON digest, form code,
  proposal ID, ordinal, question digest, text, source-gap state, proposed
  classification, and import-run identity.
- A durable preprod-exercise usage boundary that cannot be mistaken for a
  governed operational checklist.
- Canonical question-catalog list, search, filter, detail, batch-preview, and
  selection APIs.
- A retained Department Manager Question Review workspace using the current
  `Find -> Compare -> Decide` queue/dossier interface. Governed candidates use
  it for Draft disposition, classification, technical-review readiness, and
  the separately controlled approval/publication handoff; exercise questions
  use it for review/classification only and cannot enter that handoff.
- A rebuilt Department Manager New Audit workflow with server-generated draft
  identity, question selection, saved restart, exact review, and approval
  submission, followed after GM Release by canonical Lead/team preparation and
  per-question assignment coverage.
- Immutable scope snapshots carried through Finance, General Manager,
  Executive Director, General Manager Release, and Department materialization.
- Canonical Audit/assignment/package/checklist creation from the selected
  subset rather than every question in a template.
- Existing canonical Potential Finding, Finding, CAP, Evidence, report,
  notification, audit-history, dashboard, and Auditee projections.
- Task-first role navigation and simple empty/error/loading states.
- A versioned deterministic preprod dataset with all required role accounts,
  organizations, provider scopes, targets, team members, and one repeatable
  hero scenario.
- Local connected qualification across PostgreSQL, Keycloak, MinIO, Mailpit,
  Go API/worker, and the React HTTP artifact.
- A separately gated handoff into the existing local-preprod and external AWS
  preprod plans after the canonical product flow passes locally.
- Removal of the obsolete AGA-only stakeholder runtime after the canonical
  replacement is accepted.

### Explicitly Excluded

- Treating the 1,310 extracted questions as official, legally authoritative,
  source-owner attested, technically approved, or production published.
- Weakening `SOURCE_MAPPING_REQUIRED`, source-currentness, real Department
  Manager review, real publication, or operational package-eligibility rules.
- Mutating a question/catalog usage class or promoting an exercise record into
  an operational record. Governed use requires a new source-bound governed
  candidate and immutable question version through the existing authority path.
- Reusing the synthetic AGA lifecycle stores as the canonical product database.
- Loading all 1,310 question bodies into one browser response or storing them
  in URLs, browser history, Web Storage, service-worker caches, telemetry, or
  retained browser media.
- Automatic question approval, automatic question selection, automatic team
  assignment, automatic Finding creation, automatic CAP acceptance, automatic
  Evidence verification, or automatic closure.
- A compatibility bridge that keeps both stakeholder lifecycles active after
  cutover.
- Changes to the intact root HTML/CSS/Vanilla JavaScript oracle.
- Production data, real PII, real operator records, external source-owner
  decisions, legal advice, or enforcement execution.
- Loading `PREPROD_EXERCISE` content into a shared or long-lived preprod
  database, tenant, or schema whose whole exercise namespace cannot be removed.
- Branch changes, commits, pushes, deployment, external uploads, or remote
  infrastructure mutations without separate exact authorization.

## Product Truth And Data Boundary

The question catalog supports two distinct usage classes:

| Usage class | Meaning | Executable where | Required language |
|---|---|---|---|
| `GOVERNED_OPERATIONAL` | Source-authoritative, technically approved, separately published and package-eligible content | Environments that permit operational data | Governed published checklist |
| `PREPROD_EXERCISE` | Exact imported AGA question text used to exercise the product before real source/approval completion | Explicit preprod exercise profile only | Preprod exercise question; not operationally approved |

The new flow must not turn `PREPROD_EXERCISE` into a weaker route to
`GOVERNED_OPERATIONAL`. A catalog version, Audit-scope draft, plan snapshot,
Audit package, and every derived exercise record remain bound to one usage
class. Usage class is immutable: there is no exercise-to-operational promotion
or mutation command. Operational use begins with a separately created,
source/lineage-bound governed candidate and new immutable question version.
Cross-class selection, publication, or materialization fails closed.

The retained Question Review visual shell does not erase this boundary. Its
governed mode uses the existing candidate, required-owner, technical approval,
and publication aggregates. Its exercise mode uses a separate exercise-review
Draft/event aggregate and can record disposition, domain, and topic review only;
it exposes no publication command and cannot write a governed approval or
publication table.

The preprod HTTP artifact contains no embedded seed or mock data. A separately
built, one-shot loader populates a disposable environment. The running normal
API reads the resulting canonical records under the explicit preprod exercise
configuration; production/default configuration must reject that usage class.

## Core Invariants

1. Preserve exact record, role, organization, provider scope, target, route,
   version, revision, question, selection, and Audit identity.
2. Every selected question references one immutable catalog question version.
3. Every saved selection revision has a deterministic ordered digest.
4. Finance and executive decisions bind the exact submitted scope, budget, and
   resource-requirement digest. A changed question set, budget, notice policy,
   or approved resource requirement requires a new Manager submission. Named
   Lead/team preparation occurs only after GM Release and cannot mutate that
   released scope.
5. Audit materialization uses the released snapshot and does not reread a
   mutable catalog or client selection.
6. Audit creation produces `SCHEDULED` or coordination-pending inspection state
   and a `NOT_STARTED` checklist. Checklist answer, Potential Finding, package
   execution, offline execution grant/sync, and execution events fail closed
   until an authorized Inspector starts the Audit. Start advances inspection,
   assignment execution readiness, and checklist in one transaction and emits
   `audit.started` only there.
7. Routine/announced coordination and Ad Hoc/unannounced withholding remain
   distinct.
8. Potential Finding and canonical Finding remain distinct.
9. CAP acceptance is not Finding closure.
10. Evidence versions, checklist versions, selection snapshots, report
    versions, and audit history are append-only.
11. `Comment to Auditee` and `Internal CAA Note` remain structurally separate.
12. Auditee projections exclude Internal CAA Notes, other organizations, CAA
    workload, private risk scoring, question-bank governance internals, and
    enforcement deliberations.
13. Every visible control navigates exactly, performs a real canonical command,
    creates a local artifact, or is disabled with a specific reason.
14. The root legacy oracle remains intact.

## Reuse, Adaptation, And Removal Strategy

The existing work is a donor implementation, not a second permanent product.

### Reuse substantially as-is

- The sealed 52-form/1,310-question package identity, digest reconciliation,
  archive safety, and one-shot loader authorization patterns.
- Bounded 25-row pagination, server-side text search, exact question identity,
  transient question-text handling, and digest verification from the AGA
  workspace.
- Server-issued, digest-bound batch preview and the 500-item maximum.
- Local Keycloak role accounts, disposable namespace ownership, status/cleanup,
  privacy, residue, and connected-browser harness patterns.
- Canonical Planning decisions and role authority.
- Canonical Audit, assignment, checklist, Potential Finding, Finding, CAP,
  Evidence, reports, notification, audit-event, and Auditee projection modules.
- Existing React workbench, queue, dossier, selected-question, status, and
  responsive patterns when they make the New Audit flow clearer.

### Adapt into canonical product features

- Adapt the AGA question queue into a generic canonical question picker owned by
  `features/planning/` or `features/checklists/`, without AGA workspace route or
  lifecycle dependencies.
- Preserve the full current queue/dossier review composition as Department
  Manager `Checklist Management -> Question Review`. Move its implementation
  into canonical checklist-governance ownership and retain the visible
  `Find -> Compare -> Decide` hierarchy, selected-question Decision file,
  controlled reasons, Draft disposition, reclassification, and topic controls.
- Adapt Include/Exclude batch logic into Add/Remove Audit-scope selection with
  server-owned preview, exact affected-set digest, expiry, CAS, and atomic
  execution.
- Adapt the AGA package summary into the New Audit review step.
- Extend the canonical Planning intake contract with provider scope, target,
  catalog version, scope-draft identity, selected-question digest, estimated
  resource requirement, and post-release preparation linkage.
- Replace the fixed package-draft surface with a planning-owned Audit scope
  draft. Do not preserve the old fixed-ID package draft contract beside it.
- Change canonical Audit workspace creation to consume the immutable released
  subset and to separate creation from start.
- Reuse lifecycle UI concepts, but route them to canonical backend capabilities
  and canonical IDs.

### Do not copy forward

- Fixed AGA routes such as `/department-manager/aga-demo-workspace/*` as the
  stakeholder journey.
- Fixed draft, plan, organization, template, inspection, or Finding IDs.
- Technical hash-heavy copy as primary task content.
- A second AGA-only Finding/CAP/Evidence lifecycle store.
- Client-invented authority, scope, preview, readiness, recommendation,
  assignment, or object identity.
- Synthetic-only recommendation/readiness stages that duplicate normal
  Planning approval and release.

### Remove after verified connected cutover

- AGA-only Manager/Inspector/Lead/CAA Reviewer/Auditee lifecycle routes and
  route registrations.
- The duplicate `agademoworkspace` lifecycle service and its disposable
  workspace schema only after all reusable generic security fixes are
  incorporated, donor routes are disabled, and the complete connected
  canonical journey passes with the donor unavailable.
- The old AGA demo runtime composition and `make aga-demo-*` entry points,
  replacing them with one canonical preprod product start/status/stop path.
- Duplicate frontend types, tests, styles, and docs that exist only to keep the
  obsolete stakeholder lifecycle alive.

Retain the controlled AGA import source and provenance tooling only to the
extent needed to build and verify the canonical catalog. Rename or relocate it
to make its import-only ownership explicit; do not leave a callable shadow
lifecycle.

## Current Repository Orientation

### Canonical product surfaces

- `apps/web/src/features/planning/new-audit-wizard.tsx` contains the current
  five-step Manager intake. It is fixed to one draft, one submitted plan, one
  organization, and one six-question template, and it exposes only a disabled
  AGA recommendation notice.
- `apps/api/internal/planning/` and the Planning HTTP routes own persisted intake
  drafts and the Department Manager -> Finance -> General Manager -> Executive
  Director -> General Manager Release chain.
- `apps/api/internal/application/clean_state_creation.go` creates canonical
  inspections, assignments, question assignments, packages, checklists, and
  data-feed events from a released plan.
- `apps/api/internal/inspections/draft_service.go` and
  `inspection-package-builder-page.tsx` currently edit only risk focus for one
  fixed package draft; they are not a usable question-selection workflow.
- Canonical checklist, Potential Finding, Finding, CAP, Evidence, report,
  audit-event, notification, and Auditee modules already exist under
  `apps/api/internal/` and `apps/web/src/features/`.
- `checklist_template_versions.snapshot` and `inspection_packages.snapshot`
  already provide immutable version/package storage, but current Audit creation
  expects assignment coverage for every question in the selected template.
- Canonical `question_versions` already owns immutable question bodies. Existing
  regulatory generation, Draft, required-owner, technical-approval,
  publication, source-binding, and package-eligibility tables own the governed
  path; the successor must extend these authorities rather than create a
  parallel operational question bank.

### AGA donor surfaces

- `apps/api/internal/agacandidatedemo/` and
  `apps/api/internal/preproddata/agacandidatedemo/` own the sealed candidate
  package and exact 1,310-question read boundary.
- `apps/api/internal/agademoworkspace/` and
  `apps/api/internal/preproddata/agademoworkspace/` own the duplicate synthetic
  classification, selection, recommendation, inspection, Finding, CAP, and
  Evidence lifecycle.
- `apps/web/src/features/checklists/aga-classification-workspace-page.tsx`
  already solves bounded question discovery and focused decision review.
- `aga-demo-inspection-package-page.tsx` already proves server-previewed batch
  selection, exact package summary, and role handoff.
- `scripts/start-aga-demo.sh`, `status-aga-demo.sh`, `stop-aga-demo.sh`, and the
  Make targets own the current disposable AGA runtime.

### In-progress worktree boundary

The 2026-08-07 browser-remediation work is currently modified in the working
tree. Gate 0 must inspect and classify those changes before any successor edit.
Generic auth/session, CSRF, route, service-worker, accessibility, and privacy
fixes may be retained. Synthetic-only lifecycle work must not be completed
merely to preserve an obsolete runtime. No user change may be reset, silently
overwritten, or discarded.

### Gate 0 freeze record — 2026-08-07

The user authorized local implementation of this successor. The current dirty
worktree was inspected without reset or overwrite. Its disposition is:

| Disposition | Files and rationale |
|---|---|
| Reusable canonical foundation | `apps/web/src/offline/storage-readiness.ts`, `apps/web/src/sw.ts`, `apps/web/vite.config.ts`, and `apps/web/tests/e2e/support/offline-static-server.ts` app-shell marker/version synchronization; `apps/web/src/styles/utilities.css` shared `.sr-only`; `apps/web/src/styles/shell.css` SVG sizing; `apps/web/src/styles/responsive.css` generic SVG/breadcrumb layout; and `apps/web/src/ui/application-topbar.tsx` accessible notification SVG. Carry these forward only where they remain route- and product-agnostic. |
| Import/provenance donor | No modified dirty file is an import donor. The clean `apps/api/internal/agacandidatedemo/` and `apps/api/internal/preproddata/agacandidatedemo/` package reader, digest, lineage, and reconciliation code is the donor for Task 1. |
| Obsolete synthetic-only runtime | `apps/web/src/app/aga-demo-workspace-routes.tsx`, `apps/web/src/features/caps/aga-demo-cap-evidence-page.tsx`, `apps/web/src/features/findings/aga-demo-potential-finding-page.tsx`, `apps/web/src/features/inspections/aga-demo-inspection-page.tsx`, its test and `aga-demo-lifecycle.css`, and the AGA-specific classification CSS/link changes. Preserve mechanics and historical evidence, but do not carry fixed AGA routes, package-builder CTAs, or synthetic lifecycle state into the canonical runtime. |
| Planning/history foundation | The paused predecessor-plan edits and the new successor row in `docs/exec-plans/index.md` are retained. They preserve historical receipts and make this successor the sole implementation direction. |

The frozen ownership map is:

| Aggregate/table boundary | Authority and FK/event rule |
|---|---|
| `question_versions` | The sole immutable question body/version authority. A preprod import may insert a new version through the existing immutable creation path, but catalog tables never copy or update its body. |
| Catalog/import lineage | New catalog version, membership, import-run, and lineage records reference `question_versions.id`; membership changes are append-only status events. They carry `usage_class` and cannot reference governed approval/publication records when `PREPROD_EXERCISE`. |
| Governed review | Existing candidate/source/required-owner/technical-approval/publication aggregates remain authoritative for `GOVERNED_OPERATIONAL`; new review commands extend those aggregates only through their existing command boundaries. |
| Exercise review | A separate append-only Draft/event aggregate references `question_versions.id` and records disposition/domain/topic facts only. It has no technical-approval or publication FK and no promotion command. |
| Audit scope | Planning-owned scope draft, selection preview/receipt, immutable submitted/released snapshots, and selection digest reference catalog membership and question-version IDs. No client question body is accepted. |
| Preparation/materialization | Post-GM-Release Lead/team/per-question coverage and confirmation snapshots FK to the released scope digest. Materialization creates the canonical inspection/package/checklist and `audit.planned` in one transaction. |
| Execution/history | Inspector start is a separate atomic transition that emits `audit.started`; checklist, Potential Finding, Finding, CAP, Evidence, report, audit-history, idempotency, and outbox records remain append-only and exact-Audit scoped. |

The deliberate RED contract was written before runtime implementation:

- `GOCACHE=/private/tmp/avia-go-cache go -C apps/api test ./internal/questioncatalog`
  failed with undefined successor catalog/import/selection symbols, naming the
  missing exact-import, digest, bounded-selection, and CAS capabilities.
- `GOCACHE=/private/tmp/avia-go-cache go -C apps/api test ./internal/checklistgovernance -run 'Test(ExerciseQuestionReviewCannotPublish|GovernedQuestionReviewUsesSeparateTechnicalAndPublicationCommands)'`
  failed with undefined successor Question Review mode/command symbols.
- `node --test api/openapi/tests/canonical-aga-successor-contract.test.mjs`
  failed because the canonical catalog/scope/review paths and usage-class
  schemas are not yet present.

These are intentional RED failures, not fixture or syntax failures. Gate 0
also synchronized the Audit Planning, Surveillance Planning, New Inspection
form, Checklist Builder/Runner, Audit Checklist Workflow, and preprod identity
specifications with the governed-published versus dedicated-disposable
exercise boundary, catalog/scope selection, and pre-start fail-closed rules.

Open Browser QA findings now have explicit successor ownership: F-002 and
F-013 → Tasks 2–4 selection/review preview guards; F-004 → Task 8 connected
OIDC/session verification (provider end-session remains `not run` unless
locally proven); F-007 and F-021 → Tasks 5–6 exact scope/identity binding and
Task 8 negative authorization; F-010 → Tasks 2, 4–6 transaction/idempotency
and Task 8 fault/concurrency; F-015 → Tasks 3, 7, and 8 capability/mutation
separation; F-018 → Tasks 6 and 8 Auditee projection/privacy. The historical
QA report and predecessor plans remain intact; the durable tracker records the
same mapping.

## Planned Data And Interface Model

Gate 0 may refine names, but not the ownership boundaries below.

### Catalog

Use existing canonical `question_versions` as the single source of truth for
immutable question bodies and identities. Add only the catalog-version,
membership, import-lineage, and usage-boundary records needed to organize and
query those versions. Do not create a second body/version table. Gate 0 freezes
the exact existing-to-successor table/FK mapping before migration work.

The catalog projection contains:

- catalog ID, catalog version, usage class, status, source package identity,
  package digest, question count, and publication/import time;
- immutable canonical `question_versions.id`, form code, proposal ID, ordinal,
  prompt digest, source locator, source-gap state, and proposed domain/topic/
  risk; catalog membership availability/deprecation is represented by
  append-only membership-status events, never by updating a question version;
- a deterministic catalog root digest over all ordered question versions; and
- indexes for form, prompt search, domain, topic, source state, and question
  identity.

The exact 1,310 bodies may be copied into the disposable canonical preprod
catalog because the user-visible product must execute them. The loader must
verify every copied body against the sealed source digest and must not copy raw
PDF/archive bytes into canonical business tables.

The import creates new `PREPROD_EXERCISE` question-version identities and
immutable membership/lineage records in a whole-namespace-disposable target.
It never attaches exercise versions to governed candidate, technical approval,
publication, or operational checklist-template records. A governed question
with equal wording or digest is still a separately authored version with its
own source bindings and authority history.

### Question Review

Move the accepted queue/dossier presentation into canonical checklist-
governance feature ownership. Define the successor API/aggregate explicitly:

- summary, bounded queue/search/filter, selected-question detail, governance
  signals, and technical-identifier disclosure queries;
- an exercise-review Draft with revision, immutable question-version reference,
  `Retain`/`Include`/`Exclude`/`Defer`, controlled reason, reviewed domain/topic,
  actor, timestamp, and append-only successor history;
- the existing governed candidate/Draft authority extended with exact-digest
  disposition, domain, and topic successor events, while required-owner,
  technical approval, and publication remain the existing separate commands
  for `GOVERNED_OPERATIONAL` mode; and
- server-derived capability flags that disable technical approval/publication
  for `PREPROD_EXERCISE` with the exact source/lineage blocker.

Both modes reuse one visual/component contract, but their command handlers and
aggregate types remain explicit. No UI flag may route an exercise Draft into a
governed publication command.

### Audit scope draft

Replace the fixed package-draft model with a planning-owned scope aggregate:

- scope-draft ID and owning Planning intake draft ID;
- organization, provider scope, regulated target, Audit type, catalog version,
  and usage class pins;
- revision, status, ordered selected-question versions, selected count, and
  selection digest;
- server-issued Add/Remove batch previews with exact filter, affected set,
  digest, expiry, and consumption; and
- append-only submitted/released snapshots.

The browser never submits question text as authority. It submits only opaque
server-returned question-version identities and current CAS pins.

### Planning snapshot

Submitting for Finance creates one immutable Planning scope snapshot containing
all setup, selected-question, estimated resource requirement, budget, notice,
catalog, usage, and digest facts. It contains no named Lead, team, or question
assignment. Every later decision records that snapshot digest. A returned plan
creates a new draft revision and requires resubmission through Finance.

### Post-release preparation

After GM Release, the Department Manager records the Lead Inspector against the
exact released scope. The Lead Inspector then records the team and ordered
per-question assignment coverage. These append-only preparation revisions bind
the released scope digest and may not change selected questions, budget, notice,
or approved resource requirement. A change to an approved fact returns to a new
Planning revision; personnel-only changes remain post-release preparation.

The Department Manager preparation confirmation freezes one preparation
snapshot and digest after every selected question has explicit coverage and the
notice-policy inputs are complete. It does not claim Auditee date confirmation
or execution readiness; coordination follows non-executable materialization.

### Audit materialization

Department materialization consumes the released Planning snapshot and matching
confirmed preparation snapshot and atomically creates:

- canonical `inspections` record in `AWAITING_AUDITEE_CONFIRMATION` for
  announced work or `SCHEDULED`/ready-for-start with advance notice withheld
  for unannounced work;
- exact team and question assignments;
- immutable `inspection_packages` snapshot containing only the selected
  ordered questions and assignments;
- `inspection_checklists` in `NOT_STARTED` execution state;
- planning/audit history, outbox, idempotency, and synchronization records; and
- the `audit.planned` event.

For announced work, successful Auditee/CAA date coordination advances
`AWAITING_AUDITEE_CONFIRMATION` to `SCHEDULED`/ready-for-start. For unannounced
work, no advance portal request or shared checklist package is created. The
authorized Inspector start command separately validates the applicable
readiness state, then atomically transitions the inspection, assignment
execution readiness, and checklist to executable/`IN_PROGRESS` state and emits
`audit.started`. Before that transaction, checklist response, Potential Finding,
execution-package access, offline grants, and sync commands fail closed.

## Assumptions And Ownership

### Assumptions

- The sealed accepted package continues to reconcile to 52 forms, 31 forms with
  questions, 21 zero-boundary forms, and exactly 1,310 unique questions.
- The canonical core lifecycle remains the only long-term product lifecycle.
- A preprod exercise can use non-authoritative data if the environment, catalog,
  Audit, UI, and evidence label it truthfully and production/default
  configuration cannot enable it.
- The presentation subset is chosen by a human Manager in the UI; test fixtures
  may use a deterministic subset only for repeatability.
- The current root demo remains the behavior/UI oracle where a canonical React
  surface needs a missing interaction pattern.

### Owners

- Product/CAA Operations: accept the task flow, field order, question selection
  semantics, and hero scenario.
- Department Manager: owns Audit-scope selection/submission, post-release Lead
  assignment, preparation confirmation, and materialization within exact scope.
- Lead Inspector: owns post-release team selection and per-question assignment
  coverage within the released scope and approved resource requirement.
- Finance, General Manager, and Executive Director: own their existing Planning
  decisions only.
- Admin/preprod operator: owns the one-shot catalog/data load and environment
  status/cleanup, not question approval.
- Source owner and responsible real Department Manager: remain required for
  future `GOVERNED_OPERATIONAL` use.
- Engineering/QA: own canonical integration, privacy, idempotence, artifact,
  browser, and cleanup evidence.
- Infrastructure/Operations: own any later remote preprod release and rollback.

## Stop Conditions

Stop the affected work package and record `blocked` if:

- the source package no longer reconciles to the accepted 1,310 unique
  identities and digests;
- catalog import changes question text or loses package/form/proposal lineage;
- production/default configuration can enable `PREPROD_EXERCISE` content;
- an exercise review can write or invoke governed technical-approval,
  publication, or operational-template records;
- a Manager can select outside the exact provider scope/target/department;
- a selected question is silently omitted during submit or materialization;
- approval continues after a scope-digest change without resubmission;
- Audit creation rereads mutable catalog state instead of the released snapshot;
- Audit creation emits `audit.started` before Inspector start;
- a scheduled/not-started Audit accepts a checklist answer, Potential Finding,
  execution package, offline execution grant/sync, or execution event;
- an Auditee receives internal CAA data or another organization's record;
- CAP acceptance closes a Finding;
- an original body enters URLs, browser persistence, telemetry, logs, build
  artifacts, or retained connected-test media;
- the implementation requires both canonical and synthetic stakeholder
  lifecycles to remain active; or
- local/remote cleanup cannot target only the exact task-owned environment.

## Ordered Work Packages

### Gate 0 — Freeze The Successor And Reprioritize Existing AGA Work

Objective: prevent more work from being sunk into the duplicate lifecycle and
freeze the canonical contract before runtime changes.

Work:

1. Inspect the complete current working-tree diff and classify each modified
   AGA browser-remediation change as reusable canonical foundation,
   import-specific donor work, or synthetic-only work.
2. Freeze the pre-approval `Set up -> Choose questions -> Review and submit`
   journey, the post-release Department Manager/Lead preparation journey, route
   map, role map, usage class, scope digest, preparation digest, report order,
   and materialization semantics in product/contract tests.
3. Synchronize the product specifications before any migration/runtime change.
   Record normal/operational selection as governed published question versions
   only; allow `PREPROD_EXERCISE` only in a dedicated disposable preprod
   profile; forbid exercise publication/promotion; and replace New Audit's
   pre-approval `Checklist Template` input with catalog/scope selection while
   retaining post-release preparation authority.
4. Freeze an exact table/aggregate map before migration: existing canonical
   `question_versions` is the only body/version authority; existing governed
   candidate/source/owner/technical-approval/publication tables remain the
   operational governance authority; new records are limited to catalog
   membership/import lineage, exercise-review Draft/events, Planning scope, and
   post-release preparation. Define every FK and append-only event boundary.
5. Replace the fixed package-draft contract directly; do not add a compatibility
   API or parallel question bank.
6. Write RED tests for exact 1,310 import, exercise/governed publication
   separation, both Question Review modes, dynamic New Audit drafts, question
   selection, approval snapshot binding, post-release assignment coverage,
   subset materialization, scheduled-before-start denial, accepted report order,
   real Evidence artifact flow, Auditee privacy, and CAP/closure separation.
7. Map every open Browser QA item (`F-002`, `F-004`, `F-007`, `F-010`, `F-013`,
   `F-015`, `F-018`, `F-021`) to a successor task or the technical-debt tracker
   before touching its existing dirty change. Preserve historical evidence.
8. Keep the synthetic lifecycle plans paused while this successor is the only
   implementation direction. Do not call them completed and do not discard
   their reusable/security work.
9. Record the Gate 0 decision and do not begin database/runtime changes if any
   accepted invariant is ambiguous.

Acceptance:

- RED failures name deliberately missing successor capabilities, not syntax or
  fixture errors.
- Every in-progress user change has a documented disposition.
- The exact schema/FK/event map proves there is one immutable question-version
  authority and no exercise-to-operational promotion path.
- Product specifications are the source of truth for operational-published vs
  dedicated-preprod-exercise selection and catalog/scope-based New Audit.
- The plan index exposes one unambiguous next product implementation step.

### Task 1 — Build The Canonical 1,310-Question Preprod Catalog

Objective: make the accepted AGA question inventory available to normal
canonical preprod APIs without keeping the AGA lifecycle overlay alive.

Work:

1. Add forward-only catalog-membership and import-lineage tables that reference
   existing immutable `question_versions`; never add membership state to,
   update, or otherwise mutate body/version rows, and do not create another
   prompt/body table.
2. Add a versioned `aga-preprod@1.0.0` data profile or extension contract bound
   to the exact accepted package identities and `PREPROD_EXERCISE` usage.
3. Adapt the existing controlled package reader and loader into an import-only
   command that validates the package and writes the catalog in one dedicated,
   wholly destroyable preprod exercise database/tenant/schema. Shared or stable
   preprod storage is rejected.
4. Copy only question/form/catalog business content, never raw PDF/archive
   bytes, credentials, or test media.
5. Recompute every prompt digest and the aggregate catalog digest inside the
   target transaction. Publish the catalog version only after exact count,
   uniqueness, lineage, and digest reconciliation.
6. Seed exact synthetic role accounts, department membership, operator,
   provider scope, regulated target, team, reminder rules, and an empty
   canonical Planning workspace through the existing preprod loader authority.
7. Prove idempotent replay, retention/rollback procedure, and whole-namespace
   cleanup; partial imports remain unreadable and no cleanup selectively deletes
   append-only business history.

Acceptance:

- exactly 52 forms and 1,310 question versions are readable in the active
  catalog, with 21 zero-boundary forms preserved as form records;
- every text hash equals the sealed package digest for its full identity;
- no duplicate full identity or prompt-version ID exists;
- catalog retirement/deprecation is an append-only membership event and never
  mutates an immutable question version;
- normal production/default startup cannot enable the exercise catalog; and
- the import tool is absent from normal API/worker/scheduler artifacts.

### Task 2 — Add Canonical Question Discovery And Audit-Scope Selection

Objective: let the Manager select exact question versions through simple,
bounded, durable canonical commands.

Work:

1. Add catalog summary, list/search/filter, detail, selection preview, selection
   execute, and selected-set queries to OpenAPI sources and generate Go/TypeScript
   artifacts.
2. Page text-bearing results at 25. Support server-side search and filters for
   form, domain, topic, proposed risk, source-gap state, and selected state.
3. Create the planning-owned scope aggregate with server-generated ID,
   revision, ordered selection, digest, and audit-event history.
4. Adapt the existing AGA batch-preview mechanics to `ADD` and `REMOVE`, keep
   the 500-item cap, and reject zero, expired, stale, over-limit, or changed-set
   confirmation without partial writes.
5. Add exact per-question and batch actions. Selection never implies source,
   technical, or publication approval.
6. Expose selected count, estimated resource requirement, form/domain
   distribution, and complete selection digest. Named assignment coverage is
   deliberately absent before GM Release.
7. Enforce Manager department/provider scope before returning catalog
   eligibility or mutating the draft.

Acceptance:

- the Manager can reach first, middle, and last questions and find a known body
  fragment without loading more than 25 bodies;
- row and batch actions persist after reload and exact replay;
- a stale preview or scope revision creates no partial successor;
- exercise and operational question versions cannot be mixed; and
- unauthorized roles receive no private catalog or selection signal.

### Task 3 — Build Canonical Question Review

Objective: preserve the accepted `Find -> Compare -> Decide` governance
experience on canonical routes and aggregates before any AGA workspace service
is removed.

Work:

1. Add canonical summary, bounded queue/search/filter, detail, governance-
   signal, technical-identifier, Draft history, and capability contracts under
   checklist-governance ownership; generate Go and TypeScript transports.
2. Add the exercise-review Draft/event aggregate described above with exact
   question-version identity, revision/CAS, controlled reason, disposition,
   domain/topic successor facts, actor, timestamp, and idempotency. It cannot
   write governed approval, publication, or checklist-template tables.
3. Add exact-digest governed Draft successor commands for disposition, domain,
   and topic review, then route eligible candidates through the existing
   required-owner, technical-approval, publication, source-binding, and
   package-eligibility aggregates. Do not duplicate those authorities in the
   review workspace.
4. Move the screenshot-backed page into canonical `Checklist Management ->
   Question Review` ownership and remove all `agaDemoWorkspace` command, route,
   fixed-generation, and synthetic lifecycle dependencies.
5. Preserve summary cards, server search/filters, 25-row queue, left queue/right
   Decision file, selected state, collapsed technical identifiers, governance
   signals, controlled reason, `Retain`/`Include`/`Exclude`/`Defer`, domain
   reclassification, topic actions, pagination, keyboard/focus behavior, and
   responsive stacking.
6. For exercise mode, show the exact exercise/source-gap label and disable
   technical approval/publication with a specific reason. For governed mode,
   expose those commands only from server-derived eligibility/capability facts.
7. Add separate interaction/DOM and visual tests for canonical Question Review
   and New Audit selection. Use invented privacy-safe fixtures, never the real
   1,310 bodies, at 1440x900, 1024x768, and 390x844; assert queue/dossier/action
   separation, pagination/filtering, focus order, accessible names, keyboard
   operation, and horizontal/nested overflow.

Acceptance:

- the accepted screenshot composition is recognizable and fully functional on
  a canonical route with no AGA workspace dependency;
- exercise review persists only exercise Draft/event facts and has no technical
  approval/publication operation in its capability or transport surface;
- governed review still uses exact existing authority and append-only
  publication boundaries;
- no enabled control is inert or toast-only; and
- dedicated visual/DOM/accessibility baselines pass at all three viewports.

### Task 4 — Rebuild New Audit As The Primary Manager Workflow

Objective: replace the fixed wizard and separate package builder with one clear
task flow.

Work:

1. Remove fixed `DRAFT_ID`, `SUBMITTED_PLAN_ID`, organization options, and
   template options. Create/list/resume drafts through server-generated opaque
   IDs.
2. Implement the visible stages `Set up`, `Choose questions`, and `Review and
   submit`, using step URLs only where direct resume remains safe. Named Lead,
   team, and question assignment do not occur in this pre-approval wizard.
3. Select organization, provider scope, and regulated target from authorized
   server results; do not infer scope from organization type.
4. Embed the adapted queue/dossier question picker, selected tray, selected
   counter, clear filters, batch preview, and undo/remove behavior.
5. Reuse the approved review workspace's typography, cards, queue density,
   selected-row state, right-side dossier, progressive technical disclosure,
   filters, pagination, focus behavior, and responsive stacking. Do not copy
   its governance actions into Audit selection.
6. Display source-gap and preprod-exercise truth compactly without making
   technical governance data the primary task.
7. Save every stage with CAS/idempotency, restore after refresh/login, and show
   specific conflict recovery.
8. Review shows exact catalog version, selected count/digest, estimated
   resource requirement, budget, notice, dates, and approval chain. Submit
   atomically creates the immutable Planning scope snapshot without named team
   or assignment facts.
9. Adapt useful legacy/AGA HTML, CSS, and components into canonical feature
   ownership. Do not import root runtime code or retain AGA route dependencies.

Acceptance:

- a new Manager account can create more than one independent draft;
- no product task requires a fixed repository fixture ID;
- the primary path is understandable at 1440, 1024, and 390 CSS pixels;
- no enabled control is inert or toast-only; and
- the review digest matches the server snapshot returned after submission.

### Task 5 — Bind Approval, Release, Preparation, And Audit Materialization

Objective: carry the exact Manager-authored scope through governance and create
the executable subset only after release.

Work:

1. Extend Planning views/decisions with immutable scope-snapshot ID and digest.
2. Keep Finance, General Manager, Executive Director, and General Manager
   Release transitions unchanged in authority and order.
3. Reject any decision whose current Planning revision or scope digest differs
   from the submitted snapshot.
4. A return sends the item to the Manager, creates a new editable draft
   revision, and requires Finance review again after any resubmission.
5. After GM Release, let the Department Manager assign the Lead Inspector to
   the exact released scope; let only that Lead assign the inspection team and
   per-question coverage; require Department Manager preparation confirmation.
6. Add bounded bulk assignment preview/confirm with CAS and idempotency; never
   silently assign all questions. A selected-scope, budget, notice, or approved-
   resource change creates a new Planning revision and returns through Finance.
7. Replace `CreateAuditWorkspace` client-authored IDs and full-template
   assignment input with one server-owned materialization command consuming the
   released Planning snapshot, confirmed preparation snapshot, and current
   actor authority.
8. Materialize only exact selected questions and assignments, with immutable
   text/reference/version snapshot and catalog lineage.
9. Create announced work in `AWAITING_AUDITEE_CONFIRMATION`; create unannounced
   work in `SCHEDULED`/ready-for-start with advance notice withheld and no
   portal coordination request. In both cases create the checklist in
   `NOT_STARTED`; do not create `IN_PROGRESS` or emit `audit.started` here.
10. Add or reuse the authorized Inspector start transition. In one transaction,
    require announced date coordination to have advanced to `SCHEDULED` or the
    unannounced withheld branch to be ready, validate assignment readiness,
    make the inspection/assignment/checklist executable, and emit
    `audit.started`.
11. Deny before start: checklist responses, Potential Findings, execution-
    package access, offline execution grants/sync, and execution events.
12. Preserve announced coordination and unannounced privacy.

Acceptance:

- the released snapshot selected count, package question count, and assignment
  coverage are identical;
- no unselected question enters the Audit package;
- a later catalog or draft change cannot alter the Audit;
- duplicate materialization replays exactly and conflicting re-creation fails;
- Audit creation and Inspector start are observably distinct; and
- announced materialization requires coordination before start, while
  unannounced materialization creates no advance Auditee notification/package;
- every pre-start execution command fails without partial state or event; and
- Planning, Audit, package, assignment, audit-event, outbox, sync, and
  idempotency facts commit atomically.

### Task 6 — Complete The Canonical Multi-Role Lifecycle

Objective: demonstrate the whole product from the selected checklist to
closure using normal role routes.

Work:

1. Route assigned Inspector and Lead users to the canonical Audit/Checklist,
   not fixed AGA workspace routes.
2. Render only the immutable selected package questions and exact assignments.
3. Complete checklist response, comment, real attachment upload/scan/version/
   download, Potential Finding, Lead return/dismiss/convert, and checklist
   submission through the canonical object-storage boundary.
4. Prepare the Preliminary Report after field execution, complete Lead -> DM ->
   GM -> ED approval, and issue it to the exact matching Auditee organization.
5. Issue the canonical Finding and, when CAP is required, complete Auditee CAP
   revision, CAA CAP review, Auditee Evidence upload/scan/version/download,
   Inspector/Lead Evidence verification, and closure.
6. Assert CAP acceptance leaves the Finding open and `Close` after accepted
   Evidence records `EVIDENCE_VERIFIED` closure.
7. Only after the closure decision, prepare the Final Report and complete Lead
   -> DM -> GM -> ED approval/issue using the exact Audit/Finding IDs.
8. Update Department Manager and executive dashboards from the same records.
9. Prove the announced hero scenario and an unannounced privacy negative path.

Acceptance:

- separate authenticated sessions complete the same canonical Audit across all
  required roles without hidden shared browser state;
- every route resolves from server-owned current work, not fixture constants;
- Auditee JSON/DOM contains only allowed organization-scoped fields;
- report history proves `Execution -> Preliminary Report approval/issue -> CAP
  -> Evidence -> verification/closure -> Final Report approval/issue`;
- report approval does not close open Findings; and
- the complete lifecycle history preserves exact actor, reason, version, and
  timestamp facts.

### Task 7 — Cut Over Navigation And Disable The Duplicate AGA Product

Objective: make the canonical workflow the only stakeholder product path while
retaining donor source/schema temporarily for a connected no-donor fallback
qualification.

Work:

1. Make role homes task-first: Manager `New Audit`/approval/attention, Inspector
   assigned Audits, Lead review, Auditee Findings/CAP/Evidence, and executive
   decisions.
2. Keep only role-relevant operational navigation in the HTTP preprod profile.
3. Move raw import diagnostics, package ingestion receipts, and parser/lineage
   operations to an Admin-only configuration area with progressive disclosure.
   Keep the full Question Review and responsible Department Manager governance
   actions in `Checklist Management`; Admin must not absorb technical approval
   or publication authority.
4. Unregister/disable AGA-only stakeholder routes, fixed handoff links, client
   capabilities, API operation wiring, and runtime composition. Keep donor
   source, schema, scripts, and tests physically present but unavailable until
   Task 8 connected qualification passes.
5. Keep the import-only package reader/loader and immutable lineage under
   clearly named ownership.
6. Add route/capability/artifact assertions proving there is no runtime fallback
   to the disabled donor when canonical state is absent or errors.
7. Replace operator navigation/runbook defaults with the canonical preprod
   entry point, while preserving removal instructions for Task 9.
8. Preserve the root demo and accepted baselines.

Acceptance:

- a role user never needs an `aga-demo-workspace` route;
- route, capability, OpenAPI, and artifact scans find no callable shadow
  lifecycle or fallback, even though removal has not yet occurred;
- every remaining AGA-named import surface is import/provenance-only and absent
  from normal runtime artifacts; and
- the primary role navigation has no dead or irrelevant controls.

### Task 8 — Qualify The Full Canonical Product Locally With Donor Disabled

Objective: prove the complete system in one repeatable disposable local-preprod
environment.

Work:

1. Add one canonical `make preprod-up`, `make preprod-status`, and
   `make preprod-down` operator path, or extend the existing local stack with
   those exact semantics. Do not retain alias targets for the obsolete demo.
2. Start PostgreSQL, Keycloak, MinIO, ClamAV, Mailpit, Gotenberg, API, worker,
   scheduler, and the HTTP React artifact with the AGA preprod data profile.
3. Reconcile the exact catalog, users, roles, organization, provider scope,
   target, and empty/new product starting state.
4. Assert the donor stakeholder routes/APIs/runtime are unavailable, then run
   the connected hero journey through Manager scope selection, Finance, GM, ED,
   GM Release, Department Manager Lead assignment, Lead team/question
   assignment, Department preparation, Auditee coordination, Inspector, Lead,
   Preliminary Report issue, Auditee CAP/Evidence, CAA verification/closure,
   Final Report, and dashboards.
5. Run negative authority, cross-organization, stale digest, stale preview,
   duplicate operation, scheduled-before-start, transaction fault/concurrency,
   restart/reload, and unannounced privacy scenarios.
6. Prove N-1 migration upgrade plus backup/restore, catalog/scope/preparation/
   materialization atomicity, real object upload/scan/version/download, report
   rendering, and notification delivery in the connected services.
7. Verify browser storage/cache, logs, telemetry, artifacts, grants, canonical
   lineage, and task-owned process/container cleanup.
8. Use isolated browser profiles and retain only privacy-safe evidence.

Acceptance:

- the exact 1,310 catalog is present and the hero Audit contains exactly the
  human-selected subset;
- the complete canonical lifecycle passes across distinct real local OIDC
  sessions;
- the normal HTTP artifact contains no mock/seed/testprofile input;
- there is no enabled donor route/API/runtime or fallback during the run;
- restart/reload recovers the same draft, decisions, Audit, and lifecycle;
- final task-owned residue is zero; and
- the result is labelled `canonical full-system local preprod candidate;
  verified locally`, `candidate-only`, and `release pending`.

### Task 9 — Remove The Disabled Duplicate AGA Product And Requalify

Objective: delete the obsolete stakeholder lifecycle only after connected
canonical qualification, then prove deletion did not remove retained Question
Review or import provenance.

Work:

1. Require Task 8 connected qualification evidence as the entry gate. If it is
   absent or red, do not delete donor source/schema/scripts.
2. Remove AGA-only stakeholder routes, fixed handoff links, route guards,
   transport operations, runtime wiring, lifecycle services/stores/schema,
   scripts, Make targets, synthetic lifecycle tests, and duplicate docs.
3. Retain or rename only the package reader/loader and immutable import lineage
   that the canonical preprod profile actually uses; assert it is absent from
   normal runtime artifacts.
4. Re-run focused canonical Question Review/New Audit/lifecycle tests, full Go,
   React, root, build/artifact, migration, connected hero/negative, visual/
   accessibility, privacy, and cleanup gates after deletion.
5. Update README, manifest, architecture, runbook, product specs, plan index,
   historical plan links, and technical debt without rewriting old evidence.

Acceptance:

- source, route, OpenAPI, capability, artifact, migration, script, Make-target,
  and documentation scans find no callable duplicate stakeholder lifecycle;
- canonical Question Review still preserves the accepted interface and both
  explicit governance modes;
- the donor-disabled connected qualification passes again after deletion; and
- only clearly named import/provenance code remains outside normal artifacts.

### Task 10 — External Preprod Release Gate And Handoff

Objective: make the locally qualified product available in an actual preprod
environment without conflating local proof with deployment authorization.

Work:

1. Complete the applicable
   [Local Preprod Release Candidate](2026-07-27-local-preprod-release-candidate-plan.md)
   prerequisites using the new canonical flow.
2. Freeze artifact digests, migration set, data-profile digest, rollback input,
   OIDC/TLS configuration, backup/restore evidence, and operator runbook.
3. Obtain separate explicit authorization for read-only cloud discovery and
   each cost-bearing or mutating action.
4. Execute the applicable
   [AWS Preprod Validation](2026-07-27-aws-preprod-validation-plan.md) or another
   explicitly selected environment plan; this ExecPlan does not choose or
   authorize a remote provider by itself.
5. Permit the exercise loader only in a dedicated, disposable environment or
   tenant/database/schema whose complete exercise namespace can be destroyed
   without selectively deleting append-only history. Reject shared or stable
   preprod targets before upload.
6. Freeze retention, backup, restore, rollback, and whole-namespace cleanup
   procedures before loading the approved synthetic dataset. Run the remote
   role/browser matrix, then record exact result and residue.
7. Do not promote to production or use real users/data from this plan.

Acceptance:

- remote actions occur only under their exact authorization slices;
- HTTPS/OIDC/role/privacy and the full hero lifecycle pass in the selected
  preprod environment;
- the target is proven dedicated/disposable and shared-preprod loading fails
  closed before any write;
- rollback and cleanup are proven against exact environment identity; and
- the strongest possible claim is `preprod verified` if the external plan's
  own gates pass. `Production-ready` remains unestablished.

### Task 11 — Aggregate Verification, Evidence, And Plan Closeout

Objective: synchronize implementation truth, evidence, and plan lifecycle.

Work:

1. Run focused tests first, then the complete required matrix once against a
   fresh disposable target.
2. Record exact discovered/passed counts, immutable catalog/scope/package
   digests, selected count, lifecycle IDs, privacy checks, and cleanup results.
3. Update product specs, architecture, README, manifest, runbook, build summary,
   plan index, predecessor plan statuses, and technical debt.
4. Preserve older synthetic evidence as historical; do not rewrite it as
   canonical evidence.
5. Request stakeholder/user verification of the task flow before moving this
   plan to `completed`. Tasks 1–9 plus local qualification and post-deletion
   requalification define implementation completion for this ExecPlan. If Task
   10 is not separately authorized, record it as `not run` and hand external
   release to the selected environment plan; do not block truthful local
   closeout or imply remote preprod evidence.

Acceptance:

- all requested behavior has fresh evidence;
- plan, index, evidence, tracker, product docs, and runtime agree;
- the old stakeholder AGA lifecycle is absent;
- any external preprod step not authorized or not run is labelled literally;
  and
- no production-readiness claim is made.

## Verification Commands And Expected Observations

Gate 0 must replace or refine placeholder package names after the contract is
frozen. The final matrix must include at least the following.

### Catalog and canonical Go packages

```bash
go -C apps/api test -count=1 ./internal/questioncatalog ./internal/checklistgovernance ./internal/planning ./internal/assignments ./internal/inspections ./internal/checklists ./internal/potentialfindings ./internal/findings ./internal/caps ./internal/evidence ./internal/reports ./internal/httpapi
```

Expected: nonzero catalog/import, exercise/governed review separation, scope
draft, approval binding, post-release preparation, materialization, pre-start
denial/start-state, lifecycle, report-order, authority, privacy, object-artifact,
and idempotency tests pass.

```bash
go -C apps/api test -count=1 ./internal/preproddata/...
```

Expected: the versioned AGA preprod profile reconciles exact catalog and role
facts; the loader remains one-shot, target-bound, resumable where declared, and
whole-namespace cleanable.

```bash
go -C apps/api test -count=1 ./...
```

Expected: the full Go suite passes both before cutover and after Task 9 donor
deletion, with nonzero discovery and no package relying on the removed runtime.

### Contracts and frontend

```bash
./scripts/generate-contracts.sh
npm --prefix apps/web run contracts:check
npm --prefix apps/web run typecheck
npm --prefix apps/web test -- src/features/planning src/features/checklists src/features/inspections src/features/findings src/features/caps src/features/evidence src/backend
```

Expected: generated contracts are clean; canonical Question Review, dynamic New
Audit, bounded question picker, post-release assignment/preparation, role pages,
and negative privacy cases pass with nonzero discovery.

```bash
npm --prefix apps/web test
node scripts/run-root-tests.mjs
```

`scripts/run-root-tests.mjs` is a successor discovery guard: it recursively
discovers every root `*.test.js` and `*.test.mjs` under `tests/` (including
`tests/parity/`), prints the discovered files/count, fails on an unexpected
zero/omission, and then invokes Node's test runner. Expected: the full React and
root/parity regressions pass. The root oracle is unchanged.

### Build and artifact boundaries

```bash
npm --prefix apps/web run build:demo
npm --prefix apps/web run build:http
npm --prefix apps/web run check:app-shell
node apps/web/scripts/assert-http-artifact.mjs apps/web/dist/http
node apps/web/scripts/assert-parity-boundary.mjs
```

Expected: demo and HTTP artifacts build; HTTP contains no mock/seed/import tool,
question body, credential, obsolete AGA lifecycle route, or testprofile input.

Add a focused successor scanner that asserts normal runtime absence of the
removed AGA workspace contract and import-only code.

### Browser discovery and connected scenario

```bash
npm --prefix apps/web run test:e2e:preprod -- --list
make preprod-up
make preprod-status
npm --prefix apps/web run test:e2e:preprod -- canonical-aga-full-system.spec.ts
npm --prefix apps/web run test:e2e:preprod -- canonical-question-surfaces.visual.spec.ts
make preprod-status
make preprod-down
```

Expected: the dedicated project discovers a nonzero full-system and negative
scenario set; status reports exact service health and 1,310 catalog questions;
the donor runtime is unavailable; the hero scenario completes from Manager
draft through Preliminary Report, Evidence-verified closure, and Final Report;
Question Review and New Audit have separate invented-fixture visual/DOM/
accessibility baselines at 1440x900, 1024x768, and 390x844; final cleanup reports
zero task-owned residue.

### Migration, recovery, transaction, and connected artifact gates

Gate 0 must bind these observations to repository-owned commands before Task 1:

- fresh install and N-1 forward migration reach the same schema invariants;
- backup/restore reproduces catalog, scope, approval, preparation, Audit,
  report, Finding, CAP, Evidence, and append-only event digests;
- injected failures and concurrency at selection, approval, preparation,
  materialization, Inspector start, upload/scan, and closure yield one complete
  successor or no successor, never a partial aggregate;
- MinIO/ClamAV prove real attachment and Evidence object upload, scan, immutable
  version, authorized download, denial, and cleanup;
- Gotenberg renders Preliminary and Final report artifacts from exact versions;
  Mailpit receives only expected organization-scoped notifications; and
- rerunning the connected matrix after Task 9 deletion proves there is no donor
  fallback.

### Documentation and hygiene

```bash
node tests/harness-docs-smoke.test.js
rg -n "docs/agent-harness|agent-harness/index|output-contract|verification-matrix|entropy-cleanup" AGENTS.md MANIFEST.md docs
git diff --check
```

Expected: documentation authority references and whitespace pass, and plan/
index/status language is synchronized.

### External preprod

External commands are intentionally absent until an environment and exact
authorization slices are selected. Record them as `not run` rather than
inventing a deployment command.

## Acceptance Criteria

The plan is ready for stakeholder verification only when fresh evidence proves:

1. The canonical catalog contains exactly 1,310 unique question versions from
   the accepted package, with exact text/digest and lineage reconciliation.
2. Production/default configuration cannot select `PREPROD_EXERCISE` content.
3. The Manager can create multiple independent drafts and dynamically select
   organization, provider scope, target, type, date, and exact question scope.
4. The Manager can reach all 1,310 questions through bounded pagination,
   search, and filters without loading all bodies.
5. Row and server-previewed batch selection persist with CAS, expiry,
   idempotency, and exact selection digest.
6. Named Lead/team assignment is absent before approval. After GM Release, the
   Department Manager assigns the Lead Inspector and that Lead gives every
   selected question explicit team-member coverage before preparation confirms.
7. Finance, GM, ED, and GM Release bind the same immutable scope snapshot.
8. A returned/revised plan must be resubmitted through Finance.
9. Materialization creates only the selected ordered subset and never rereads
   mutable catalog state.
10. Audit creation leaves announced work `AWAITING_AUDITEE_CONFIRMATION` and
    unannounced work `SCHEDULED` with advance notice withheld; both create a
    `NOT_STARTED` checklist. Authorized Inspector start is a separate atomic
    transition after applicable readiness, and every execution command fails
    closed beforehand.
11. Routine/announced coordination and unannounced withholding behave exactly.
12. Inspector and Lead execute only their assigned selected questions.
13. Potential Finding, canonical Finding, CAP, Evidence, and closure remain
    distinct state transitions.
14. CAP acceptance leaves the Finding open; accepted/verified Evidence closes
    the normal happy path with `EVIDENCE_VERIFIED` basis.
15. Auditee projections structurally exclude internal and cross-organization
    data.
16. Report history follows `Execution -> Preliminary Report approval/issue ->
    CAP -> Evidence -> verification/closure -> Final Report approval/issue`,
    uses exact canonical Audit/Finding identity, and report approval does not
    close open work.
17. Dashboards and queues reflect the same canonical records.
18. The primary role journey contains no fixed AGA route or fixture identity.
19. The duplicate AGA stakeholder lifecycle is first disabled and qualified
    unavailable, then removed only after connected canonical cutover passes.
20. The HTTP artifact contains no mock/seed/import/runtime shadow capability.
21. The full local connected run survives reload/restart, N-1 migration, backup/
    restore, transaction faults/concurrency, and donor deletion; proves real
    object upload/scan/version/download, report rendering, and notifications;
    leaves zero residue; and uses an isolated browser profile.
22. Any external preprod deployment remains separately authorized and has its
    own literal result.
23. Department Manager Question Review retains the accepted
    `Find -> Compare -> Decide` visual and interaction contract, including the
    bounded queue, Decision file, controlled Draft actions, reclassification,
    topic controls, progressive technical details, and responsive behavior.
24. New Audit visibly reuses that interaction language while keeping Audit
    scope selection distinct from classification, technical approval, and
    publication.
25. Exercise Question Review can persist disposition/domain/topic successor
    facts but cannot expose or invoke governed technical approval/publication;
    governed mode continues to use the existing authority chain.
26. Canonical `question_versions` is the sole immutable body/version authority;
    availability/deprecation is append-only catalog membership history and
    usage class cannot mutate or promote.
27. Separate invented-fixture visual/DOM/accessibility tests cover Question
    Review and New Audit at 1440x900, 1024x768, and 390x844 without retaining
    any real AGA question body.
28. External exercise loading rejects shared/stable preprod and is permitted
    only for a dedicated whole-namespace-disposable target.

## Risks And Mitigations

| Risk | Mitigation |
|---|---|
| Imported questions are mistaken for approved regulatory content | Persist and display `PREPROD_EXERCISE`; preserve source gaps; production/default rejects the usage class. |
| Exercise review becomes a publication shortcut | Use a separate exercise Draft/event aggregate; omit publication capability/transport; require a new source-bound governed version for operational use. |
| A parallel catalog creates two question authorities | Keep `question_versions` as the only immutable body/version source; add only membership/import-lineage records and freeze FK mapping in Gate 0. |
| Existing AGA work is discarded or overwritten | Gate 0 inventories the dirty diff and carries reusable generic fixes forward before removing synthetic-only code. |
| The primary Audit flow becomes a technical classification tool | Put purpose-specific selection inside New Audit while retaining the full Department Manager Question Review workspace; hide only raw import/parser internals behind Admin disclosure. |
| A selection changes during approval | Freeze an immutable scope snapshot and require every decision to bind its digest. |
| Whole-template behavior silently adds unselected questions | Materialize from exact released selected refs and assert equal counts/digests. |
| Browser overloads on 1,310 bodies | Page at 25, server-side search/filter, selected refs only, bounded redacted cache. |
| Batch selection hides unseen impact | Server-issued exact preview, 500-item cap, explicit confirmation, CAS, expiry. |
| Client invents scope or authority | Server-returned opaque scope/target/question/team identities and transaction-time revalidation. |
| Audit is started during creation | Create a scheduled inspection and `NOT_STARTED` checklist; deny every execution path until one atomic Inspector-start transaction. |
| Duplicate lifecycle removal hides a missing dependency | Disable it first, qualify the connected canonical journey with no fallback, delete second, then rerun the full matrix. |
| Preliminary/Final reports occur at the wrong lifecycle point | Lock the accepted report/CAP/Evidence order in state, history, and connected tests. |
| Auditee sees question-bank or internal CAA data | Positive allowlist DTOs plus raw JSON/DOM cross-scope scans. |
| CAP acceptance is displayed as closure | State-machine tests and visible lifecycle stepper require Evidence verification. |
| Preprod loader leaks into normal artifacts | One-shot out-of-process loader and explicit artifact/link scans. |
| Exercise data pollutes shared preprod append-only history | Reject shared/stable targets and load only into a dedicated whole-namespace-disposable exercise environment. |
| Remote deployment is inferred from local success | Separate Task 10 authority gate and literal `not run`/`preprod verified` labels. |

## Dependencies

- Accepted sealed AGA package and immutable 52-form/1,310-question identity.
- Current in-progress browser-remediation diff, preserved until Gate 0 classifies
  every change.
- Canonical Planning, assignment, Audit, checklist, Potential Finding, Finding,
  CAP, Evidence, report, notification, and audit-event modules.
- Existing preprod identity/data loader and exact eight-role authority contract.
- Local PostgreSQL, Keycloak, MinIO, ClamAV, Mailpit, Gotenberg, API/worker, and
  React HTTP stack.
- Explicit implementation authorization before Gate 0 changes runtime code.
- Explicit environment/action authorization before external preprod work.

Real source-owner attestation and real Department Manager technical/publication
decisions are not dependencies for the truthfully labelled preprod exercise.
They remain mandatory for any future `GOVERNED_OPERATIONAL` use.

## Idempotence And Recovery

- Catalog import is target-bound and publishes an active version only after
  exact reconciliation. A failed import leaves no readable partial catalog.
- Draft creation uses server-generated identity; exact create replay returns the
  same draft.
- Scope row/batch changes are CAS-protected and idempotent. Stale previews are
  discarded and regenerated.
- Submission creates one immutable scope snapshot. A returned plan creates a
  new editable revision without rewriting earlier snapshots or decisions.
- Every approval command uses exact Planning revision and scope digest.
- Post-release Lead/team preparation is append-only, CAS-protected, and bound to
  the released scope; exact replay returns the same preparation revision.
- Materialization is one server-owned atomic command over the released scope and
  confirmed preparation snapshots. Exact replay returns the same Audit; changed
  payload/snapshot or second non-replay creation conflicts.
- Audit start is separately idempotent and cannot replay against another Audit
  or assignment.
- CAP, Evidence, report, and audit histories remain append-only.
- Local connected runs use a complete disposable namespace. Cleanup targets
  only exact task-owned resources and never selectively deletes append-only
  history.
- External preprod recovery follows its environment plan's exact backup,
  rollback, and residue contract; no broad destructive command is authorized by
  this document.

## Progress

- [x] 2026-08-07: User rejected the current complex demo direction and selected
  a full-system preprod product flow with Department Manager question selection.
- [x] 2026-08-07: User authorized planning and explicitly allowed reuse,
  copying, or adaptation of suitable AGA demo parts.
- [x] 2026-08-07: Planning contract, architecture, manifest, verification
  matrix, output contract, product workflows, Manager specifications, completed
  React migration authority, current AGA Manager plan, current browser
  remediation plan, canonical Planning/Audit materialization code, package
  draft code, and AGA donor surfaces inspected.
- [x] 2026-08-07: Canonical successor scope, reuse/removal strategy, ordered
  work packages, verification, and external-preprod authorization boundary
  recorded in this plan.
- [x] 2026-08-07: User confirmed the current screenshot-backed question-review
  interface is valuable. The plan now preserves it as Department Manager
  `Checklist Management -> Question Review` and distinguishes it from the
  simplified New Audit question selector.
- [x] 2026-08-07: Independent Sol Ultra review returned `NOT ACCEPTED` with four
  critical and seven important findings. The plan now adds a canonical Question
  Review work package, restores post-GM-Release team authority and report order,
  separates exercise/governed review aggregates, freezes one question-version
  authority, adds scheduled execution denial, qualifies before deletion, pauses
  synthetic implementation directions, hardens visual/recovery/artifact tests,
  and restricts exercise data to a dedicated disposable target.
- [x] 2026-08-07: Sol Ultra closure pass two found no Critical issue and three
  Important/two Minor residuals. The plan now separates preparation confirmation
  from announced Auditee coordination/readiness, moves the operational-vs-
  exercise catalog/scope rule into Gate 0 product-spec synchronization, removes
  executable legacy slices from the paused Aug-03 prompt, requires a full Go
  suite, and makes catalog/import tables reference rather than modify immutable
  `question_versions`.
- [x] 2026-08-07: Sol Ultra final closure returned `ACCEPTED` with no Critical
  or Important finding after exact announced/unannounced non-executable
  materialization language was synchronized across Objective, acceptance,
  Decision Log, Execution Prompt, and plan index.
- [x] 2026-08-07: Gate 0 froze the canonical table/FK/aggregate/event
  boundaries, classified every dirty AGA change, mapped F-002/F-004/F-007/
  F-010/F-013/F-015/F-018/F-021, synchronized the product specifications,
  and recorded deliberate RED contract failures. The paused predecessor plans
  remain historical and this successor is the sole implementation direction.
- [x] Gate 0: successor contract, dirty-diff disposition, RED tests, and
  predecessor reprioritization.
- [ ] Task 1: canonical 1,310-question catalog and preprod data profile
  (in progress).
- [ ] Task 2: question discovery and scope-selection backend.
- [ ] Task 3: canonical Question Review workspace and governance modes.
- [ ] Task 4: primary New Audit workflow.
- [ ] Task 5: approval binding, post-release preparation, and materialization.
- [ ] Task 6: accepted canonical multi-role/report lifecycle.
- [ ] Task 7: navigation cutover and donor disablement.
- [ ] Task 8: local connected qualification with donor disabled.
- [ ] Task 9: duplicate lifecycle removal and requalification.
- [ ] Task 10: separately authorized external preprod release gate.
- [ ] Task 11: aggregate evidence and stakeholder handoff.

## Decision Log

### 2026-08-07 — Use one canonical lifecycle

The existing AGA workspace proves useful question-selection and role-handoff
mechanics, but it intentionally avoids canonical Audit, Finding, CAP, and
Evidence stores. The successor integrates those mechanics into the canonical
product and removes the duplicate stakeholder lifecycle after cutover.

### 2026-08-07 — Put question selection inside New Audit

The Department Manager chooses the scope while creating the Audit draft. The
selection is frozen before Finance submission and remains visible through
approval and release. This matches the expected mental model and eliminates the
separate classification/package maze from the main task.

Named personnel are deliberately not selected here. After GM Release, the
Department Manager assigns the Lead Inspector, the Lead assigns the team and
question coverage, and the Department Manager confirms preparation. This keeps
the successor aligned with the accepted surveillance-planning authority model.

### 2026-08-07 — Preserve the accepted question-review interface

The screenshot-backed `Find -> Compare -> Decide` queue and Decision file are a
strong Department Manager governance workspace and remain in the product under
Checklist Management. New Audit borrows its visual/interaction language but
uses only Audit-scope selection actions. Full Draft disposition,
reclassification, technical-review readiness, and publication handoff remain
separate so the simpler creation flow does not erase or misstate governance.

The shell has explicit governed and exercise modes. Governed mode uses the
existing candidate/source/owner/technical-approval/publication authorities.
Exercise mode writes only review/classification Draft events and cannot publish.

### 2026-08-07 — Keep governance truth while enabling preprod exercise

The 1,310 questions are usable as explicitly labelled preprod exercise content,
not as operationally approved content. The usage class is environment- and
record-bound and cannot satisfy the real governed publication path.

Canonical `question_versions` remains the only immutable body/version authority.
Exercise-to-operational mutation/promotion is forbidden; operational use starts
with a separately source-bound governed candidate and version.

### 2026-08-07 — Adapt donor code; do not preserve a compatibility product

Pagination, search, batch preview, digest, loader, and UI patterns may be moved
or rewritten in canonical modules. A permanent adapter between AGA workspace
and canonical lifecycle is forbidden. The old runtime is disabled first,
connected qualification runs without it, then its source/schema/scripts are
removed and the complete matrix runs again.

### 2026-08-07 — Separate Audit creation from Audit start

Canonical materialization currently creates an `IN_PROGRESS` Audit and emits
both planned and started events. The successor creates announced work as
`AWAITING_AUDITEE_CONFIRMATION` or unannounced work as `SCHEDULED` with notice
withheld, plus a `NOT_STARTED` checklist in both cases. All execution paths fail
before applicable readiness and one atomic Inspector-start transition emits
`audit.started`.

### 2026-08-07 — Preserve accepted report order

The canonical hero path is `Execution -> Preliminary Report approval/issue ->
CAP -> Evidence -> verification/closure -> Final Report approval/issue`.
Preliminary and Final Reports are not both deferred until after closure.

### 2026-08-07 — Qualify locally before any external preprod action

The requested product target includes an actual preprod environment, but this
planning request does not authorize deployment. The complete canonical flow is
qualified locally first, then handed to the existing external-preprod plan
under separate action-by-action authority. Exercise data may be loaded remotely
only into a dedicated whole-namespace-disposable target, never shared/stable
preprod.

### 2026-08-07 — Gate 0 freeze and dirty-worktree disposition

Implementation is authorized only for local Gate 0, Tasks 1–9, and Task 11.
The app-shell/accessibility/responsive changes are reusable canonical
foundation; no dirty runtime file is an import donor; fixed AGA route,
package-builder, and synthetic lifecycle changes are obsolete after their
mechanics are adapted. Catalog/import tables reference immutable
`question_versions` and usage class is immutable. The first RED tests now name
the missing catalog, CAS selection, review-command, and OpenAPI capabilities;
their initial failures are preserved as Gate 0 evidence. Task 1 is the next
implementation step. Task 10 remains `not run`.

## Discoveries

### 2026-08-07 — Most of the real lifecycle already exists

The canonical repository already has Planning approvals, Audit workspace
creation, assignments, checklist execution, Potential Findings, Findings, CAP,
Evidence, reports, audit history, OIDC, and preprod loader infrastructure. The
largest product gap is coherent integration, not a new lifecycle engine.

### 2026-08-07 — The current New Audit wizard is a fixed fixture

The React wizard uses constant draft and plan IDs, one organization, and one
template. It saves a Planning item but offers no real question selection or
team assignment. The AGA section is only a disabled recommendation notice.

### 2026-08-07 — The canonical package draft is not a package builder

The current package-draft API/UI loads one fixed record and edits only a
comma-separated risk-focus field. Questions are read-only snapshots. It should
be replaced, not extended alongside a new selector.

### 2026-08-07 — Canonical Audit creation assumes the whole template

`CreateAuditWorkspace` requires assignment coverage for every question in the
template snapshot and rejects any extra/missing question. It must consume the
released selected subset instead.

### 2026-08-07 — The AGA demo already solves the difficult catalog mechanics

The AGA donor path already proves 53 bounded pages, question body/digest
composition, server-side search, 500-item server previews, selection state,
exact scope/target pins, distinct role sessions, and connected cleanup. These
mechanics can materially reduce implementation risk when moved into canonical
ownership.

### 2026-08-07 — Current AGA browser remediation is in progress

The working tree contains uncommitted AGA auth, route, service-worker,
responsive, and lifecycle remediation. Successor execution must start by
classifying and preserving reusable work; planning does not authorize deleting
or reverting it.

### 2026-08-07 — Current scheduled creation is still executable

The current materializer creates inspection/checklist execution state too early,
and checklist commands do not consistently gate on inspection start. The
successor requires a real `NOT_STARTED` checklist and negative guards for
responses, Potential Findings, execution packages, offline grants/sync, and
events until atomic Inspector start.

## Outcome

Planning outcome as of 2026-08-07:

- A self-contained canonical successor plan now replaces the idea of improving
  the separate complex AGA demo indefinitely.
- The target product path starts with Department Manager New Audit and exact
  question selection, then uses the accepted approval chain, post-release
  Department Manager/Lead preparation, and canonical multi-role lifecycle
  through Preliminary Report, Evidence-verified closure, and Final Report.
- Existing AGA work is explicitly treated as reusable donor code and import
  provenance. Synthetic implementation plans are paused; the duplicate
  stakeholder lifecycle is disabled, qualified unavailable, then removed and
  requalified.
- The accepted `Find -> Compare -> Decide` Question Review interface remains a
  first-class Department Manager governance surface; only its AGA-only route
  and duplicate lifecycle ownership are replaced.
- The exact 1,310 questions remain truthful preprod exercise content; real
  regulatory authority and production readiness are not claimed.
- Sol Ultra's initial `NOT ACCEPTED` findings and follow-up residuals are
  incorporated. Final independent closure is `ACCEPTED` with Critical `0` and
  Important `0`; this accepts the plan, not runtime implementation.
- No runtime implementation, branch action, commit, push, deployment, or
  external mutation was performed by this planning task.

## Execution Prompt

```text
Execute docs/exec-plans/active/2026-08-07-canonical-aga-preprod-end-to-end-product-plan.md in order after the user explicitly authorizes implementation.

Begin at Gate 0. Read this plan completely, then inspect the current working-tree diff and the complete paused AGA browser-remediation plan before editing. Preserve every unrelated or user-owned change. Classify the in-progress AGA changes as reusable canonical foundation, import-only donor work, or synthetic-only work; do not reset or discard them.

Build one canonical product journey: Department Manager creates a dynamic Audit draft, selects an exact subset from the 1,310-question preprod exercise catalog, and submits the immutable scope/resource snapshot through Finance -> GM -> ED -> GM Release. After release, the Department Manager assigns the Lead Inspector; the Lead assigns the team and exact question coverage; and the Department Manager confirms preparation. Materialize announced work as AWAITING_AUDITEE_CONFIRMATION or unannounced work as SCHEDULED with notice withheld; create a NOT_STARTED checklist in both cases. Inspector start must be a separate atomic gate after applicable readiness. Complete Execution -> Preliminary Report approval/issue -> CAP -> Evidence -> verification/closure -> Final Report approval/issue across distinct role sessions.

Reuse or adapt the existing AGA pagination, search, digest, batch-preview, loader, and UI patterns where they reduce risk, but do not preserve the AGA workspace as a second stakeholder lifecycle. Implement canonical Question Review before deleting its donor, with explicit governed and exercise aggregates behind the retained Find -> Compare -> Decide shell. Keep question_versions as the single immutable body/version authority. PREPROD_EXERCISE cannot mutate/promote or invoke governed publication and fails closed in production/default configuration. Disable the donor, qualify the complete connected journey without fallback, delete the donor, then rerun the matrix. Preserve Auditee privacy, append-only versions/history, Comment to Auditee/Internal CAA Note separation, CAP/closure separation, exact identity, and root-oracle integrity.

Keep this plan and docs/exec-plans/index.md synchronized after every material gate. Use test-first contract changes, generated artifacts from source, isolated browser profiles, and exact task-owned cleanup. Do not change branches, commit, push, deploy, upload, or mutate remote infrastructure without separate explicit authorization. External preprod work begins only after local canonical qualification and its own action-by-action authorization.
```
