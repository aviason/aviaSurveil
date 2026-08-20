# Demo MVP New Audit Planning Redesign

Date: 2026-08-20
Last updated: 2026-08-20
Status: active — frontend proposal flow, mock/HTTP contract, additive route, and migration boundary implemented; full lifecycle verification in progress; production readiness not claimed

## Planning authority

This plan follows [`docs/PLANS.md`](../../PLANS.md), the repository-local
[`AGENTS.md`](../../../AGENTS.md), the
[`verification matrix`](../../agent-harness/verification-matrix.md), and the
[`output contract`](../../agent-harness/output-contract.md).

The product decisions are fixed by the
[`Demo MVP Department Manager Decisions`](../../product-specs/workflows/2026-08-20-demo-mvp-department-manager-decisions.md).
The completed
[`Department Manager Planning Intake UX Redesign`](2026-08-17-department-manager-planning-intake-ux-redesign-plan.md)
is the current implementation baseline. This plan is its deliberate successor
for New Audit behavior and UX. It does not reopen or weaken the earlier plan's
verified accessibility, persistence, privacy, or responsive behavior.

This plan does not authorize a branch operation, commit, push, release,
deployment, cloud write, or external-system mutation.

## Objective and user-visible outcome

Redesign New Audit around one Department Manager job:

> Create a clear, finance-reviewable Audit plan without selecting the final
> checklist or assigning named inspectors too early.

The completed New Audit flow must let the user:

1. identify the inspected organization, operation/site, and inspection type;
2. state the purpose, schedule, mode, and conditional location or meeting link;
3. plan inspector capacity, estimated checklist-item volume, and budget;
4. review the exact plan Finance will receive; and
5. submit it with one primary action.

New Audit must not show Trigger type, editable Risk Category, historical Audit
recommendations, recommended checklist content, or exact checklist-item
selection. Those belong after the required approval and release boundary.

The target interaction keeps the current five-step editor rhythm while
changing the decisions owned by each step:

1. `Scope`
2. `Purpose`
3. `Schedule`
4. `Resources & budget`
5. `Review`

The implementation must remain end-to-end coherent. The current checklist
selection capability is moved to a post-release Department Manager preparation
route without visually redesigning that selector in this plan. A later
separately approved plan will redesign recommendation and checklist selection.

## User, context, and success definition

### Primary user

The Department Manager is accountable for proposing a feasible inspection and
obtaining financial/governance approval. The user understands aviation
oversight but should not need to understand database aggregates, selection
digests, catalog batches, target fallback labels, or server command order.

### Primary question per screen

- Scope: `Who or what will be inspected, and under which inspection type?`
- Purpose: `Why is this Audit being planned?`
- Schedule: `When and how will it take place?`
- Resources & budget: `What capacity and budget should Finance approve?`
- Review: `Is this the plan I want Finance and the later approvers to review?`

### Success

- No meaningless or system-owned field asks the user to make a choice.
- The normal path contains one primary action per screen.
- A user can complete the plan without opening checklist-item details.
- System-resolved values remain visible enough to build trust without being
  rendered as disabled or single-option dropdowns.
- Finance receives the exact scope, schedule, capacity, checklist estimate,
  and budget the Department Manager reviewed.
- Final checklist selection remains an explicit Department Manager decision,
  but only after the Planning item reaches the required released state.

## Current rendered baseline — 20 August 2026

The design baseline is the current React application rendered directly from
the present checkout at
`http://127.0.0.1:5173/department-manager/new-audit/step-1`. Historical
React/legacy parity screenshots and the 10 August manual-review screenshots are
not design references for this plan.

The current 1440×900 application was exercised through Scope/Basics, Purpose,
Schedule, Checklist & budget, selection confirmation, and Review. The current
390×844 Scope/Basics state was also rendered. Current evidence showed:

- the present navy Avia shell and restrained open workbench composition;
- one horizontal five-step progress rail on desktop;
- a main editor plus sticky `Inspection brief` rail;
- a sticky bottom action bar with one normal primary next action;
- serialized save-state presentation and inline field treatment;
- a visually coherent current system with no horizontal overflow or captured
  console error in the exercised flow; and
- a tall five-row progress list on phone that consumes substantial first-
  viewport space.

The current visual system is a strength to preserve. The redesign target is
primarily information architecture and lifecycle timing, not a wholesale
restyle. The reportable current problems are:

1. The title still says `New Inspection`, not `New Audit`.
2. Scope always renders four dropdowns, even when provider scope or target has
   no real choice.
3. Purpose still contains Inspection approach, Trigger type, and Risk Category.
4. Schedule uses an unrestricted Location field and an emoji calendar glyph.
5. `Checklist & budget` performs historical recommendation and exact selection
   before Finance.
6. The right summary lists many future `To be set` rows and repeats obsolete
   approach/question-selection facts.
7. The mobile stepper occupies too much vertical space before the current
   task.

## UI/UX design direction

The user-named `ui-ux-pro-max` workflow classifies this as an internal
government/compliance workflow. Apply `Accessible & Ethical`, `Trust &
Authority`, restrained flat design, and progressive disclosure.

The skill's landing-page and `Exaggerated Minimalism` suggestions are not
appropriate for this product and are explicitly rejected. New Audit is a
professional governance editor, not a marketing page or decorative onboarding
wizard.

### Visual system

- Keep the existing Avia semantic tokens and workbench system-font stack.
- Use the existing navy shell, white/slate surfaces, action blue, restrained
  success green, warning amber, and error red semantically.
- Do not add a page-specific font, gradient, glass effect, illustration, fake
  KPI, decorative hero, or new design-system dependency.
- Use open sections and dividers rather than nesting every group in a card.
- Use the existing 4/8px rhythm, 48px form controls, and 44px minimum action
  targets.
- Use inline/shared vector icons only where an icon clarifies meaning. Remove
  the current emoji calendar glyph.
- Use 150–250ms state transitions for disclosure and feedback only, and respect
  `prefers-reduced-motion`.

### Layout decision

Preserve the current main-editor + sticky-summary composition on desktop. It is
already visually coherent and helps a five-step workflow retain context.

Rename `Inspection brief` to `Audit plan summary` and make it progressive:

- show resolved Scope facts immediately;
- add Purpose, Schedule, Resources, and Budget facts only after they have real
  values instead of filling the rail with repeated `To be set` rows;
- remove Inspection approach, Notice policy as a primary fact, exact Questions
  selected, and other obsolete Planning content;
- keep autosave state in the summary header;
- keep Finance/governance metadata for Review rather than repeating it on every
  step; and
- keep the complete authoritative summary on Review.

On phone, replace the current always-expanded five-row progress list with one
compact line such as `Step 2 of 5 · Purpose` plus an optional `View all steps`
disclosure. Keep the Audit plan summary collapsed by default and place it after
the current step heading but before the sticky action region only when opened.

Preserve the current sticky action bar, open section styling, readable content
width, and shell integration. Do not replace the latest UI with the older card
wizard or legacy visual baseline.

## Detailed interaction specification

### Step 1 — Scope

Purpose: establish the exact server-authorized inspection context.

Controls and behavior:

1. `Inspected Organization` is a visible required select using the
   human-readable organization name.
2. After organization selection, render one `Operation / site` group:
   - if provider scope and regulated target each have one valid value, select
     them automatically and show a read-only summary with an
     `Automatically selected` status;
   - if provider scope has several values, show its human-readable selection;
   - after provider scope resolution, show a regulated-target selection only
     when several valid targets remain;
   - never show a disabled single-option dropdown;
   - never show a `... regulated target` fallback or a raw ID.
3. `Inspection type` remains a required user selection even when the scope is
   otherwise resolved automatically.
4. Remove Domain from the user form. If required, derive and display it as
   server-owned context in Review.
5. Reserve layout space and show a skeleton/status if authorized scope loading
   exceeds 300ms. A load failure includes `Try again`; an empty authorized
   scope explains that no Audit can be planned for the organization.
6. `Continue` is the only primary action. The first valid Continue creates one
   authoritative draft, reports `Creating draft…`, and advances only after the
   server returns the draft and revision.
7. `Cancel` is secondary and returns to Planning. If unsaved input exists,
   confirm before leaving.

When a saved draft returns to Scope, changing organization/provider/target/type
requires one consequence dialog because it invalidates derived notice,
location, and workload estimates. Purpose, date, manual budget, and other safe
values are retained; dependent server-derived values are recomputed.

### Step 2 — Purpose

Purpose: state why the Audit is being planned without asking for unrelated
system metadata.

- Keep Purpose as the visual and semantic focus of the screen.
- Show an optional `Start from a purpose` preset selector above the textarea.
- Selecting a preset loads editable text into a required `Purpose` textarea.
- If the textarea already contains edited text, require confirmation before a
  preset replaces it.
- `Custom purpose` leaves the textarea fully user-authored.
- Persist the resolved purpose text; a preset identifier may be retained only
  as non-authoritative provenance.
- Remove Inspection approach/category, Trigger type, and Risk Category from
  this step.
- Show the current Scope only in the Audit plan summary; do not add another
  inline duplicate block above the form.

`Continue` remains the only primary action. Purpose validates on blur and on
continue, and the latest autosave revision is flushed before navigation.

### Step 3 — Schedule

Purpose: choose when and how the Audit will take place.

#### Date and mode

- `Planned date` is required and uses a semantic/native date control with a
  localized readable value in summaries.
- Replace the current emoji calendar glyph with a consistent vector/calendar
  control or the native date affordance.
- `Mode` uses two visible radio choices: `On-site` and `Remote` so the dependent
  location behavior is apparent without opening a select.
- Hybrid is not added until separately approved.

#### On-site location

- Location is required for On-site only.
- When the target supplies a location, render its canonical label read-only
  with an adjacent `Edit` button.
- `Edit` reveals previously used canonical locations and a final
  `Enter another location` choice.
- A selected canonical location persists both `locationId` and visible label.
- Manual text is resolved server-side against canonical labels and aliases.
  Exact/likely duplicates offer the canonical option before a new location is
  accepted.
- If no automatic location exists, show a compact `Add location` action rather
  than an empty disabled field.

#### Remote meeting details

- Hide Location completely for Remote.
- Offer an optional `Add online meeting link` disclosure.
- Validate an entered link as HTTP(S) and show its error beside the field.
- Switching from On-site to Remote preserves the in-session on-site choice if
  the user switches back, but it does not submit or snapshot that location
  while Remote is selected.

`Continue` validates the currently visible schedule fields, focuses the first
invalid field, flushes autosave, and advances only after the latest revision is
saved.

### Step 4 — Resources & budget

Purpose: estimate the capacity and budget Finance is being asked to approve.

- Replace the current historical-recommendation/checklist-selection page;
  those capabilities do not remain hidden on this route.
- `Required inspectors` is a required positive integer. Show the server's
  eligible-roster count and evaluation time as advisory context; a higher
  request produces a non-blocking capacity warning for Finance rather than a
  hard form error.
- `Estimated checklist items` is a required positive integer.
- The server returns a suggested count, safe minimum/maximum, applicable item
  count, basis label, and estimate version/digest for the chosen scope/type.
- Start the field at the suggested count. Allow manual change.
- A value outside the safe range produces a non-blocking warning explaining
  that Finance will review the entered value; it is not silently corrected.
- `Browse checklist items` is secondary progressive disclosure. It opens a
  read-only drawer/sheet with search, relevant filters, matching count, and a
  concise preview. It contains no selection checkboxes and makes no immutable
  checklist decision. `Use this count` may copy the current filtered count to
  the estimate.
- Keep server pagination and render at most 50 item rows at once; never render
  all 1,310 item bodies.
- `Requested budget` is required; blank is invalid and numeric zero is valid.
- Currency remains an explicit controlled selection until a single configured
  organization currency can be proven.

Layout uses two restrained form groups, `Resources` and `Budget`, within the
current open editor style. It must not become a dashboard/KPI card grid.

`Continue to review` is the only primary action. It validates on blur and on
continue, focuses the first invalid field, flushes autosave, and advances only
after the latest revision is saved.

### Step 5 — Review

Purpose: let the Department Manager verify exactly what is sent for approval.

Use five open sections that match the owning steps:

1. `Scope`
   - Inspected Organization
   - provider scope
   - regulated target
   - inspection type
   - any server-derived domain label when meaningful
2. `Purpose`
   - resolved purpose text
3. `Schedule`
   - planned date
   - mode
   - location for On-site or meeting link for Remote
4. `Resources and budget`
   - required inspector count
   - estimated checklist-item count
   - system suggestion/range and manual-override indication
   - requested budget and currency
5. `Approval context`
   - `Initiated by Department Manager` as secondary system metadata
   - server-derived notice policy when applicable
   - current Finance/GM/Executive/GM-release path once, in plain language
   - explicit statement that submission creates a Planning item, not an
     executable Audit or final checklist

Each editable section has a labelled `Edit` action returning to its current
step route. Review is the preview; do not add another Preview dialog.

Use one primary `Submit to Finance` action with pending, success, and
recoverable error states. On success, return to Planning with the new record
selected and show its human-readable reference, current owner, and next
action.

## Form, feedback, and accessibility contract

- Use persistent labels, not placeholder-only inputs.
- Mark required fields visually and semantically.
- Validate on blur and on Continue/Submit, not on every keystroke.
- Put the error below its field, bind `aria-invalid` and
  `aria-describedby`, and announce it through `role=alert`/an appropriate live
  region.
- Use a page error summary only for multiple field errors or a server error
  without one field owner. Summary actions focus the relevant control.
- Preserve logical DOM/tab order and visible 2–4px focus treatment.
- Completed step labels may be buttons only when navigation is state-safe;
  future steps are not interactive.
- After Continue, Back, or a Review Edit route change and draft hydration,
  focus the new `tabIndex={-1}` step heading and announce
  `Step n of 5 · <label>`. Autosave rerenders must not steal focus.
- Drawers/dialogs trap focus, close on Escape and a labelled Close button,
  make the background inert, and return focus to the trigger.
- Do not communicate auto-selection, warning, success, or error by color alone.
- Keep body text at least 16px on phone controls, controls at least 44px high,
  and touch targets at least 44×44px with 8px separation.
- At 200% browser zoom and 390×844, no field, error, step label, or primary
  action may be clipped or require horizontal page scrolling.
- Sticky actions reserve content/safe-area space and must not create a nested
  scroll trap.
- Autosave is serialized and revision-bound. Show `Saving…`, `Saved`, or
  `Couldn't save — Retry` without using a toast as the only evidence.

## Architectural correction

### Verified current contract gap

The current implementation cannot satisfy these decisions through a
frontend-only change:

- `PlanningIntakeDraftValues` contains Trigger, Risk Category, exact selected
  question IDs, selection digest, and selection-derived resource data.
- complete draft validation requires Trigger, Risk Category, Location, and at
  least one exact selected question;
- Finance/GM/Executive decisions bind a submitted canonical scope snapshot
  containing exact question identities; and
- GM Release copies that exact selection into the released Audit-package
  snapshot.

Therefore the implementation must separate approval of the Planning proposal
from preparation of the executable Audit package.

### Target aggregate separation

#### Planning proposal

Owns and freezes:

- organization/provider scope/regulated target/inspection type;
- purpose;
- planned date, mode, conditional location or meeting link;
- required inspector count;
- estimated checklist-item count plus the estimate basis/version;
- requested budget and currency;
- derived initiator and notice metadata; and
- approval history.

Planning approvals bind an immutable `planning_submission_snapshot` ID and
digest. They do not bind exact checklist-item identities or a selection digest.

Finance, General Manager, and Executive Director read one role-authorized
`PlanningProposalDetailView` sourced from that immutable snapshot. It contains
the exact labels and values shown on Department Manager Review: purpose, mode,
conditional location/meeting details, inspector count, checklist-item estimate
and basis, budget/currency, snapshot ID/digest, and approval metadata. Decision
commands pin that same Planning snapshot ID and digest. No approval projection,
audit event, or outbox payload contains a checklist selection digest.

#### Audit-package scope

Begins only after the Planning item reaches the required released state. It
owns:

- the current governed catalog identity;
- post-approval historical recommendation evaluation;
- exact checklist-item identities and selection digest;
- selected-count variance from the approved plan estimate;
- Lead Inspector and inspector assignment; and
- later per-item coverage and preparation confirmation.

Use an explicit state sequence:

1. `PlanningProposal: DRAFT`
2. `PlanningProposal: SUBMITTED`
3. `PlanningProposal: RELEASED`
4. `AuditPackageScope: DRAFT`
5. `AuditPackageScope: SELECTION_CONFIRMED`
6. `AuditPackageScope: FINALIZED`
7. `AuditPreparation: IN_PROGRESS`
8. `AuditPreparation: CONFIRMED`
9. Audit materialization

An idempotent Department Manager package-finalization command pins the released
Planning snapshot, catalog root, recommendation evaluation, exact selection
digest/count, and selected identities into the immutable FINALIZED package
snapshot. Assignment, per-item coverage, preparation confirmation, and
materialization bind that FINALIZED package snapshot ID. `Planning RELEASED`
and `AuditPackageScope FINALIZED` are distinct decisions and must never share a
status label or digest field.

The Planning snapshot freezes an `approvedChecklistItemCeiling`. For this MVP,
the ceiling equals the Department Manager's submitted estimated checklist-item
count reviewed by Finance. Post-release selection may finalize only when
`selectedCount <= approvedChecklistItemCeiling`; `ceiling + 1` fails closed
with `PLANNING_AMENDMENT_REQUIRED`. A later checklist-selection plan may propose
a tolerance/amendment UX, but this plan has exact enforceable semantics.

The executable Audit is still created/materialized only after the immutable
FINALIZED Audit-package scope and required preparation confirmation exist.

## API and data-contract target

### Planning input and view

Replace the current client-authored `PlanningIntakeDraftValues` shape with a
write model that contains only user decisions:

- organization ID;
- provider scope ID;
- regulated target ID;
- inspection type;
- purpose text and optional preset provenance;
- planned date;
- mode;
- a discriminated On-site location input of either
  `{kind: CANONICAL, locationId}` or
  `{kind: NEW, proposedLabel, acceptedResolutionToken}`, or an optional Remote
  meeting link as allowed by mode;
- required inspector count;
- estimated checklist-item count;
- workload-estimate ID/digest;
- requested budget; and
- currency.

The server view adds human labels, initiated-by metadata, notice policy,
workload suggestion/range/basis, revision, and timestamps.

The client never writes a canonical location label. The server derives labels
from canonical IDs. A location-resolution token binds the organization, target,
normalized proposed label, candidate matches, resolver revision, and expiry so
a retry cannot accept a different resolution.

Purpose presets are server-managed read data with stable ID, version, active
state, display order, and optional scope/inspection-type applicability. The
Planning snapshot always stores resolved purpose text; preset ID/version is
provenance only. An empty preset list leaves the required free-text field fully
usable.

Required inspector count is a positive user-entered estimate, not hard-blocked
by current roster size. The server may return an advisory `eligibleRosterCount`
with scope and evaluation time. A requested count above it shows a non-blocking
capacity warning for Finance; it is not described as scheduling availability.

Workload-estimate provenance is entirely server-owned and includes catalog ID,
version/root digest, usage class, policy version, evaluation time, applicable
count, suggested count, safe range, and estimate digest. Catalog-root drift at
post-release setup recomputes the estimate and yields an explicit
amendment/readiness result; it never silently changes the approved ceiling.

Remove obsolete active-write fields rather than adding compatibility fallbacks:

- client-authored organization name;
- Domain;
- Inspection approach/category;
- client-authored notice policy;
- Trigger type;
- Risk Category;
- template/scope compatibility fields;
- catalog version and scope-draft ID from pre-Finance Planning;
- exact selected question IDs and selection digest; and
- selection-derived form/domain distributions.

Historical immutable snapshots remain readable as historical records. Active
records follow the explicit transition matrix under Persistence; there is no
runtime dual-read fallback.

### New read models/commands

Provide typed, Department-Manager-authorized operations for:

- listing applicable versioned purpose presets;
- listing recent/canonical locations for the selected organization/target;
- resolving a manual location against canonical labels and aliases;
- obtaining the planning workload estimate for the exact scope/type; and
- idempotently ensuring and reading a post-release Audit-package setup for a
  released Planning item, including scope-draft ID, catalog/root, selection
  digest/count, revision, variance state, and next action.

All commands retain operation ID, idempotency, expected revision, exact role,
and organization/scope authorization. Labels are server-derived.

Target route families:

- `GET /v1/planning/purpose-presets`
- `GET /v1/planning/locations`
- `POST /v1/planning/location-resolutions`
- `POST /v1/planning/workload-estimates`
- `GET /v1/planning/items/{planningItemId}/proposal` for authorized approvers
- `POST /v1/planning/items/{planningItemId}/audit-package-setup` as idempotent
  ensure/create
- `GET /v1/planning/items/{planningItemId}/audit-package-setup` for reload and
  resume
- `POST /v1/planning/items/{planningItemId}/audit-package-finalizations`

Exact method names may follow existing generated conventions, but the read/
write separation and response facts above are required.

### Persistence

Use the next forward-only migration to:

- create immutable submitted/released Planning snapshot storage distinct from
  question-selection snapshots;
- persist required inspector count, estimated checklist-item count, estimate
  identity/bounds, currency, mode, and conditional location/meeting facts;
- introduce canonical inspection locations and normalized aliases, scoped so
  one spelling does not create a new historical universe;
- connect the post-release canonical Audit-package scope draft to the released
  Planning snapshot; and
- remove the database constraints that require a question selection before a
  Planning item can be submitted or released.

The repository migration runner embeds only `*.up.sql`. Use
`000057_<name>.up.sql`, advance `LatestVersion` to 57, and verify fresh 0→57,
existing 56→57, repeated Apply at 57, transaction-failure rollback, and schema/
data invariants. Operational database restore is the rollback path; do not add
an unused down migration or claim unsupported up/down/up verification.

Perform one deterministic data cutover with no runtime compatibility reader:

| Existing record | Cutover disposition |
|---|---|
| Unsubmitted DRAFT | Rewrite the mutable draft to the new safe-field shape, drop obsolete selection/Trigger/Risk fields, retain its ID/revision and user-authored safe values, then recompute derived location/workload facts. |
| RETURNED | Preserve Planning item ID, revision, approval history, and prior immutable submitted snapshot; rewrite the mutable successor draft to the new shape and create a new Planning submission snapshot on resubmit. |
| FINANCE_REVIEW / GM_REVIEW / EXECUTIVE_DIRECTOR_REVIEW / GM_RELEASE | Backfill one immutable Planning snapshot from the planning-only projection of the current submitted scope snapshot; preserve item ID/revision/history and switch subsequent decisions to the Planning snapshot pin. |
| RELEASED with exact scope snapshot | Backfill the Planning snapshot and retain the existing exact released scope snapshot as the already-FINALIZED legacy Audit-package scope; do not force re-selection. |

Add fixture tests for every row. The application reads only the new Planning
snapshot contract after migration.

### Historical matching

Move historical recommendation evaluation to post-release Audit-package setup.
Retain the existing exact safety dimensions: organization, provider-scope root
and version, regulated target, inspection type, catalog/root lineage, and usage
class. Change only location from an eligibility key into a score/explanation:

- same location: strong supporting match;
- different location: keep the history and label the difference;
- unknown location: keep the history and label it unknown.

Update server, mock, golden fixtures, and tests so location aliases resolve to
one canonical location identity and location mismatch never discards otherwise
comparable history. Cross-catalog history remains out of scope until a separate
question-successor/source-digest policy exists.

## Route and component target

### New Audit routes

Keep the current five-route rhythm and replace only their user-facing purpose:

- `/department-manager/new-audit/step-1` — Scope
- `/department-manager/new-audit/step-2?draftId=...` — Purpose
- `/department-manager/new-audit/step-3?draftId=...` — Schedule
- `/department-manager/new-audit/step-4?draftId=...` — Resources & budget
- `/department-manager/new-audit/step-5?draftId=...` — Review

Update current route labels and visible-action contracts in place. No route
alias or compatibility layer is required.

### Post-release checklist route

Move the current selector behavior to:

`/department-manager/planning/:planningItemId/setup/checklist`

This plan may extract and relocate the existing catalog/recommendation/
selection components and their tests, but it must not visually redesign that
surface. Its first-page title and context must truthfully state that it is
post-release Audit preparation. The later checklist-selection plan owns its
full UX/UI redesign.

Register it as a new additive route identity under parent `audit-plan` for both
demo and HTTP profiles. Assign a new `ui-audit-*` identity after inspecting the
current registry; do not reuse the historical `ui-audit-043` or rewrite the
legacy source oracle. Route contracts, screen registry, visible actions,
accessibility inventory, and parity disposition must agree.

### Suggested React ownership

- `new-audit-page.tsx` — route composition and authoritative draft lifecycle
- `new-audit-scope-step.tsx`
- `new-audit-purpose-step.tsx`
- `new-audit-schedule-step.tsx`
- `new-audit-resources-step.tsx`
- `new-audit-review-step.tsx`
- `audit-plan-summary.tsx`
- `planning-workload-preview.tsx`
- `planning-location-field.tsx`
- `post-release-checklist-selection-page.tsx` — extracted current selector
- a small serialized autosave hook colocated with New Audit

Split by state ownership, not arbitrary line count. Delete obsolete catalog
selection state and CSS from New Audit after the selector is extracted.

## Scope

### Included

- New Audit product/API/data separation described above.
- The current five-step New Audit composition, relabelled and simplified as
  Scope, Purpose, Schedule, Resources & budget, Review.
- Purpose preset + editable-text behavior.
- Conditional canonical location and optional remote meeting link.
- Required inspector count and editable checklist-item workload estimate.
- Read-only checklist-item pool preview for estimation.
- Finance-ready Review and submit behavior.
- Planning proposal snapshots that do not contain exact checklist selection.
- Preservation and relocation of current checklist-selection UI after release.
- Post-release historical matching semantics needed by that relocated surface.
- Mock/HTTP parity, generated OpenAPI/types, deterministic fixtures, tests,
  accessibility, and browser verification.
- Minimal Planning parent-page integration for New Audit entry, resume, return,
  and selected post-submit record.

### Explicit exclusions

- No visual redesign of the post-release checklist selector.
- No full Planning-home dashboard/table redesign; its filters and quick views
  require a separate plan.
- No Department Manager/Lead Inspector team-assignment ownership transfer or
  team UI redesign beyond the required post-release boundary. The decision
  record still targets Department Manager ownership of Lead Inspector and
  inspector assignment; a mandatory successor plan must implement that before
  the full Department Manager preparation experience is called complete.
- No change to the Finance → General Manager → Executive Director → General
  Manager Release chain in this plan.
- No change to checklist authoring/publication, Inspector execution, Findings,
  CAP, reports, or other role panels.
- No AI-generated legal, regulatory, compliance, enforcement, or approval
  decision.
- No new UI framework, icon package, external font, animation package, or
  speculative dark-mode redesign.
- No deployment, release, public data mutation, infrastructure write, commit,
  or push without separate explicit authorization.

## Ownership and repository orientation

Primary product/documentation surfaces:

- `docs/product-specs/workflows/2026-08-20-demo-mvp-department-manager-decisions.md`
- `docs/product-specs/modules/AUDIT_PLANNING.md`
- `docs/product-specs/workflows/SURVEILLANCE_PLANNING_WORKFLOW.md`
- `docs/product-specs/workflows/AUDIT_CHECKLIST_WORKFLOW.md`
- `docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.md`
- `docs/product-specs/data-and-rules/CONCEPTUAL_DATA_MODEL.md`
- `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md`

Primary API/data surfaces:

- `api/openapi/aviasurveil360.yaml`
- `apps/api/migrations/000057_*.{up,down}.sql`
- `apps/api/internal/planning/intake.go`
- `apps/api/internal/planning/service.go`
- `apps/api/internal/application/clean_state_creation.go`
- `apps/api/internal/application/canonical_scope.go`
- `apps/api/internal/httpapi/clean_state_creation_api.go`
- `apps/api/internal/assignments/`
- `apps/api/internal/httpapi/canonical_aga_api.go`
- `apps/api/internal/httpapi/task4_api.go`
- generated Go transport and focused planning/assignment tests

Primary React/mock surfaces:

- `apps/web/src/backend/backend.ts`
- `apps/web/src/backend/http-backend.ts`
- `apps/web/src/backend/transport-mappers.ts`
- `apps/web/src/features/planning/new-audit-wizard.tsx` or its replacement
- `apps/web/src/features/planning/new-audit-wizard.test.tsx`
- `apps/web/src/features/planning/planning-workspaces.tsx`
- `apps/web/src/styles/features/planning-intake.css`
- `apps/web/src/mock/mock-engine.ts`
- `apps/web/src/mock/seed-data.ts`
- `apps/web/src/mock/prior-audit-recommendations.ts`
- `apps/web/src/app/route-contracts.ts`
- `apps/web/tests/e2e/planning-intake-ux.spec.ts`
- route, parity, visible-action, accessibility, and qualification tests

AviaWorkspace owns release composition. It is not modified or executed under
this implementation plan without separate authorization.

## Ordered implementation

Tasks 2–6 form one coordinated cutover milestone. Add schema and the dormant
new proposal/package APIs first; extract and prove the post-release selector;
add approver projection; then switch Go, mock, React, and routes together and
delete obsolete pre-Finance selection paths. The currently working lifecycle
remains the runnable default until the replacement passes the complete
Planning → approval/release → package selection → preparation → materialization
focused test. Do not expose an intermediate build that can release a plan but
cannot finalize its package.

### Gate 0 — Baseline, dirty-tree, and contract proof

1. Read all authorities and this plan completely.
2. Run `make repos-status` at the workspace root and `git status --short` in
   AviaSurveil; preserve the existing untracked test catalog and every
   unrelated change.
3. Run the current focused React wizard tests, planning Go tests, contract
   checks, and typecheck.
4. Record current route count, current required intake fields, current
   pre-Finance selection dependency, and current approval snapshot dependency.
5. Capture the current latest-code Scope/Basics, Purpose, Schedule, Checklist &
   budget, selector/dialog, Review, inline-error, and 390px states using an
   isolated browser profile. Do not use historical parity/manual screenshots.

Expected: current behavior passes its existing tests and visibly contradicts
the new decision record in the named ways. Those are baseline facts, not
regressions to preserve.

### Gate 0.5 — Close provisional product decisions

Before source edits, verify checklist/checklist-item/question terminology
against the current data model and confirm these recommended plan decisions in
the Department Manager decision record:

- use `Checklist items` only if one executable row is one question/control;
- persist an optional Remote meeting link in the Planning snapshot so Review
  and authorized approvers see the same logistics;
- preserve the current Finance → General Manager → Executive Director →
  General Manager Release chain for this slice; and
- use server-managed, versioned purpose presets with free-text fallback.

If any decision is not accepted, remove that capability from implementation
scope rather than hard-coding an unresolved behavior.

### Task 1 — Align canonical product specifications and failing contracts

1. Update the planning, checklist, screen, data-model, and permission specs to
   distinguish Planning proposal from post-release Audit-package preparation.
2. Remove pre-Finance exact-selection language from current specs.
3. Add failing OpenAPI/Go/React contract tests for the new write model,
   planning snapshots, workload estimate, location resolution, and
   post-release scope-draft creation.
4. Add negative tests proving exact selected question IDs are rejected in New
   Audit writes and unavailable in Finance planning projections.

Expected: product documents agree, while new tests fail for the absent
contract rather than for fixture or runner errors.

### Task 2 — Separate Planning persistence from Audit-package persistence

1. Add the forward-only migration, transactional failure behavior, cutover
   matrix, and operational restore notes.
2. Implement immutable submitted/released Planning snapshots and digest pins.
3. Consolidate first-draft creation so the first Continue creates only the
   Planning draft; update `application/clean_state_creation.go`,
   `application/canonical_scope.go`, and the creation handler so no canonical
   Audit-package scope row is created pre-approval.
4. Add role-authorized `PlanningProposalDetailView` read projection/endpoint
   and switch Finance/GM/Executive UI to the exact immutable submitted values.
5. Replace approval command pins from submitted question-scope snapshots to
   submitted Planning snapshots.
6. Make GM Release release the Planning snapshot without requiring selected
   questions.
7. Create the post-release canonical scope draft only through the authorized
   Department Manager ensure/read setup command.
8. Add the explicit package state machine and idempotent finalization command;
   gate assignment/preparation/materialization on FINALIZED.
9. Ensure returned Planning items create a new submitted Planning snapshot
   revision and never overwrite earlier approval evidence.

Add DB-level negative tests proving first Continue and Submit to Finance create
no canonical Audit-package scope row, selection operation, question snapshot,
assignment, or executable Audit.

Expected: a complete Planning proposal reaches RELEASED with zero exact
checklist items, while no executable Audit/package exists.

### Task 3 — Add canonical location and workload-estimate services

1. Add versioned purpose-preset read data with applicability and free-text
   fallback.
2. Add canonical location/alias persistence and target-derived location lookup.
3. Add recent-location listing scoped by organization/target and manual
   resolution with exact/likely/new outcomes.
4. Add a deterministic planning workload estimate for scope + inspection type,
   returning server-owned catalog/root/policy provenance, suggested, safe
   range, applicable count, basis, and digest.
5. Add the eligible-roster-count advisory with scope and evaluation time; do
   not treat it as schedule availability or a hard maximum.
6. Store the chosen estimate, approved checklist-item ceiling, and source facts
   in the Planning snapshot.
7. Move full historical recommendation evaluation behind post-release setup
   and make location a weighted/explained signal rather than a filter key.
8. Add no-history, same-location, alias-location, different-location, and
   unknown-location golden tests.

Expected: location spelling cannot split history, and Planning receives only a
numeric workload recommendation rather than question recommendations.

### Task 4 — Regenerate transport and establish mock/HTTP parity

1. Update OpenAPI schemas and endpoints.
2. Regenerate checked Go/TypeScript transport artifacts with the repository's
   existing commands.
3. Replace frontend backend types/mappers with the new write/view models.
4. Update mock state and commands to mirror HTTP authority, derived labels,
   location resolution, workload estimates, Planning snapshots, and
   post-release setup.
5. Remove obsolete route actions, fields, fixture values, and selection
   dependencies rather than keeping dual behavior.

Expected: contract checks and focused mock/HTTP parity tests pass with one
behavior model.

### Task 5 — Recompose the current five-step New Audit experience

1. Preserve the current desktop five-step rhythm and rename/recompose it as
   Scope, Purpose, Schedule, Resources & budget, Review.
2. Implement auto-resolved Operation/site summaries and conditional controls.
3. Remove Domain, approach/category, Trigger, Risk Category, recommendation,
   and final selection UI from New Audit.
4. Implement Purpose presets + editable text, Schedule date/mode/conditional
   location or meeting link, and Resources/workload/budget on their owning
   current steps.
5. Implement the read-only workload preview drawer without selection controls.
6. Preserve and simplify the current Audit plan summary rail; hide unset and
   obsolete rows, and keep its collapsed mobile form.
7. Replace the mobile five-row progress block with the compact current-step +
   optional all-steps disclosure while preserving desktop progress.
8. Preserve serialized autosave, expected-revision recovery, inline validation,
   focus recovery, Back/Cancel, reload, and runtime-restart behavior.
9. Implement the decision-ready Review and Submit-to-Finance handoff.
10. Delete dead New Audit catalog state, batching code, dossier code, and CSS
   after post-release extraction.

Expected: the normal user never sees or selects a checklist item in New Audit,
and every displayed control changes the Planning result.

### Task 6 — Relocate current checklist selection after release

1. At Gate 0 capture the latest-code selector subtree baseline: recommendation
   summary, search, advanced filters, catalog list, selected summary,
   selection confirmation/progress, and question dossier. Explicitly exclude
   the old wizard stepper, Audit plan summary, Budget fields, and wizard footer.
2. Extract that bounded selector subtree into the post-release Department
   Manager route.
3. Preserve the bounded subtree's rendered hierarchy and interactions in this
   plan; change only route context, data ownership, and lifecycle timing.
4. Bind it to the idempotent ensure/read setup projection and current governed
   catalog.
5. Freeze the exact selection only after the explicit Department Manager
   selection confirmation.
6. Attach the FINALIZED Audit-package scope snapshot to later assignment and
   materialization preparation.
7. Add a visible fail-closed state for `selectedCount >
   approvedChecklistItemCeiling` requiring a Planning amendment; detailed
   resolution UX remains deferred.
8. Test deep-link reload, browser/runtime restart, idempotent retry, two-tab
   revision conflict, one-draft uniqueness, and pre-release denial.

Expected: exact question IDs and historical recommendations first appear only
after the released Planning handoff, and the later execution package remains
immutable and complete.

### Task 7 — Route, parent handoff, and regression repair

1. Keep the current five canonical step routes and update their labels/actions
   to the new ownership model.
2. Update Planning entry/resume/return links and post-submit selected record.
3. Route released records requiring setup to the post-release preparation
   surface.
4. Update screen manifests, parity inventories, route tests, visible actions,
   and accessibility inventories without weakening coverage.
5. Implement user-facing recovery for each migration disposition: rewritten
   DRAFT resumes at its first incomplete step; RETURNED resumes its successor
   draft with return reason/history; pending approvals open the backfilled
   immutable dossier; RELEASED legacy records continue directly to their
   existing finalized package/preparation.
6. Confirm Auditee/Inspector projections contain no private Planning purpose,
   budget, workload, location, meeting link, or recommendation data before the
   configured release/notice boundary.

Expected: every route has one truthful page identity and every visible action
works or is explicitly disabled with a reason.

### Task 8 — UX critique and visual/accessibility verification

1. Run the `ui-ux-pro-max` pre-delivery UX validation search and its critical
   accessibility/touch/performance checklist.
2. Review the rendered flow against this plan and remove at least one remaining
   nonessential copy or visual element.
3. Run the complete New Audit flow at 1440×900, 1024×768, 768×1024, and
   390×844.
4. Verify auto-selected and multi-option scope states, purpose preset/custom,
   on-site auto/edit/manual location, duplicate alias, Remote meeting link,
   inside/outside workload range, blank/zero budget, Review edits, autosave
   failure/retry, return/resubmit, and submit success.
5. Verify keyboard-only order, visible focus, Escape/focus return, first-invalid
   focus, 200% zoom, reduced motion, screen-reader names/states, touch targets,
   sticky action clearance, console, loading, errors, and zero horizontal
   overflow.
6. Verify that the bounded latest-code selector subtree is visually unchanged
   except for truthful post-release context and remains inaccessible before
   release; the old wizard/budget chrome is intentionally absent.
7. Clean task-owned Browser, Chrome helper, Vite, Playwright, and test processes.

Expected: no fixable UX, accessibility, or lifecycle issue remains in the
planned slice.

## Verification matrix

| Surface | Command/method | Expected observation |
|---|---|---|
| Documentation | `node tests/harness-docs-smoke.test.js` and `git diff --check` | Links resolve and plan/spec/index agree. |
| Contracts/codegen | `npm --prefix apps/web run contracts:check` | OpenAPI examples and checked generated outputs are current. |
| React types | `npm --prefix apps/web run typecheck` | No type error. |
| Focused React | `npm --prefix apps/web test -- --run src/features/planning/new-audit-wizard.test.tsx` plus relocated-selector tests | Five-step, persistence, location, workload, validation, and post-release boundary pass. |
| Go planning | `go -C apps/api test -count=1 ./internal/planning ./internal/assignments ./internal/httpapi` | Planning snapshots, approval pins, release, setup, selection, and authorization pass. |
| Migration | `go -C apps/api test -count=1 ./migrations` and `go -C apps/api test -count=1 ./tests/integration -run 'TestNewAuditPlanningMigration|TestNewAuditPlanningCutover'` | Fresh 0→57, existing 56→57, repeated Apply, failure rollback, and all status-cutover fixtures pass. |
| Full React | `npm --prefix apps/web test` | No unrelated React regression. |
| Builds | `npm --prefix apps/web run build:demo` and `npm --prefix apps/web run build:http` | Demo and connected artifacts build with correct boundaries. |
| Mock E2E | `npm --prefix apps/web run test:e2e:mock -- tests/e2e/planning-intake-ux.spec.ts` | Scope → Purpose → Schedule → Resources & budget → Review → Finance works without checklist selection. |
| Connected qualification | focused all-role qualification lifecycle | Planning approval/release, post-release selection, preparation, materialization, Inspector execution, and privacy complete or are labelled `blocked`. |
| Visible actions/a11y | `npm --prefix apps/web run test:e2e:visible-actions` and `npm --prefix apps/web run test:e2e:accessibility` | No inert enabled action or critical a11y violation. |
| Browser QA | isolated browser at four viewports | UX/focus/zoom/motion/loading/error/overflow checks pass. |
| Repository | `make qualification-bootstrap-check && make check` from AviaSurveil, then root `make check` when cross-repository contracts change | Required local gates pass; no release claim is inferred. |

## Acceptance criteria

1. The page title is `New Audit`; the organization label is `Inspected
   Organization`.
2. New Audit keeps exactly five focused steps: Scope, Purpose, Schedule,
   Resources & budget, Review.
3. Provider scope and target remain server-owned and are not rendered as
   dropdowns when only one valid option exists.
4. Auto-resolved scope is shown as human-readable read-only context; no raw or
   generated fallback label is visible.
5. Inspection type remains a real Department Manager selection.
6. Domain, Inspection approach/category, Trigger type, and Risk Category are
   absent from the user form.
7. Purpose supports presets and editable/free text without silently replacing
   user edits.
8. On-site location is auto-resolved when possible, read-only until Edit, and
   canonicalized against recent locations/aliases. Remote hides location and
   permits an optional meeting link.
9. Location never acts as the strict key that discards otherwise relevant
   historical Audits.
10. Required inspector count, estimated checklist-item count, recommendation
    range, budget, and currency are visible in Review and Finance Review.
11. Finance, GM, and Executive read the immutable
    `PlanningProposalDetailView`; every Review field equals the submitted
    dossier value and all approval decisions pin its Planning snapshot ID and
    digest without a selection digest.
12. The checklist-item estimate is editable; an out-of-range value produces a
    non-blocking explanation and is never silently changed.
13. Browse checklist items is opt-in, read-only, filterable, paginated, and
    contains no final-selection control.
14. New Audit contains no historical recommendation card, recommended
    checklist list, exact checklist selection, selection digest, or batching
    protocol.
15. Finance/GM/Executive/GM release binds a Planning snapshot without exact
    checklist IDs.
16. Final checklist selection and historical recommendation are available only
    in post-release Department Manager preparation.
17. The existing checklist-selection UI is not visually redesigned under this
    plan.
18. No executable Audit or Inspector assignment is created by Submit to
    Finance.
19. Blank budget fails inline; zero budget remains valid.
20. Draft creation, serialized autosave, retry, reload, Back/Cancel, returned
    revision, and submit idempotency remain exact.
21. One primary action, inline recovery, 44px targets, keyboard support,
    reduced motion, 200% zoom, and responsive no-overflow behavior pass.
22. Successful Continue, Back, and Review Edit navigation announces and focuses
    the new step heading without autosave focus theft.
23. Desktop retains the current horizontal five-step rail and Audit plan
    summary; phone uses a compact current-step indicator and collapsed summary
    without losing navigation context.
24. Mock and HTTP behavior agree; no private Planning/checklist data leaks to
    Auditee projections.
25. Full relevant verification is `verified locally` or each unavailable gate
    is labelled literally `blocked`/`not run` with its reason.

## Risks, dependencies, idempotence, and recovery

### Risks and mitigations

- **Approval meaning drift:** moving exact selection after approval could make
  Finance approve an unbounded workload. Store and approve the estimate plus
  its range/basis; fail closed on material variance until the later amendment
  UX is approved.
- **Snapshot confusion:** reusing one snapshot for plan and package recreates
  the current coupling. Use separate immutable aggregate identities and
  digests.
- **Location proliferation:** creating a canonical location on each draft save
  creates duplicates. Resolve aliases during entry and create a new canonical
  location only through an explicit accepted resolution at submit/setup.
- **Scope-change staleness:** scope changes can leave stale location/workload
  values. Invalidate only dependent derived values, recompute them, and retain
  safe user-authored fields.
- **False simplification:** hiding controls without changing backend
  requirements would produce invisible defaults. Remove obsolete write fields
  and validation from the contract.
- **Later-selector regression:** extraction can unintentionally redesign or
  weaken selection consent. Preserve its existing UI tests and batch/digest
  behavior; only change lifecycle context.
- **Large change surface:** API, migration, mock, React, and lifecycle are
  coupled. Complete each task with focused passing evidence before starting the
  next; do not leave dual behavior.

### Dependencies

- Current organization/provider-scope/target authorization.
- Current governed catalog and recommendation engine.
- Existing planning approval chain and assignment/materialization services.
- Disposable PostgreSQL qualification and generated OpenAPI toolchain.
- A later approved checklist-selection UX plan for the deferred redesign and
  workload-amendment interaction.

### Idempotence

- Draft creation, autosave, submit, approval, release, location resolution,
  post-release setup, selection commits, and materialization retain scoped
  operation IDs, semantic hashes, expected revisions, and retry-safe results.
- Workload estimates are deterministic for their recorded scope/type/catalog
  inputs and expose an identity/digest.
- Location resolution does not create a canonical location until the user has
  accepted the resolution and the owning command commits.
- Browser/tests use disposable local state and do not mutate public demo data.

### Recovery

- Capture the dirty-tree baseline and exact overlapping hunks before source
  edits.
- Apply the migration only to disposable local profiles until fresh 0→57,
  existing 56→57, repeated Apply, transactional-failure rollback, and cutover
  invariants pass.
- If an implementation task fails, repair or revert only that task's explicit
  patch; never use reset, broad checkout, recursive clean, or branch changes.
- Preserve immutable submitted/released Planning and Audit-package snapshots;
  a returned plan creates a successor snapshot rather than overwriting history.
- Keep the current released demo untouched until separate release authority is
  granted.

## Decisions and discoveries

### 2026-08-20 — New Audit is planning, not package assembly

Decision: New Audit ends at a finance-reviewable plan. Historical
recommendation and exact checklist selection move to post-release Department
Manager preparation.

### 2026-08-20 — Preserve five single-purpose steps from the current UI

Decision: use Scope, Purpose, Schedule, Resources & budget, Review. Direct
rendering of the latest React code showed that its five-step desktop rhythm is
clear and visually mature. Replacing checklist selection with resource
planning gives step four a valid single purpose without overloading another
screen.

### 2026-08-20 — Auto-resolved values remain visible, not interactive

Decision: show one compact Operation/site summary immediately. Invisible
system choices reduce trust; disabled single-option dropdowns create fake work.

### 2026-08-20 — Preserve and progressively simplify the summary rail

Decision: the latest `Inspection brief` rail is useful in the current
five-step desktop flow. Rename it `Audit plan summary`, keep only resolved
facts, remove obsolete/future placeholder rows, and preserve its collapsed
mobile behavior.

### 2026-08-20 — Current render, not historical screenshots, is the baseline

Decision: old legacy-parity and 10 August manual-review screenshots are not
used as design authority. The current checkout rendered at localhost at
1440×900 and 390×844 is the baseline for visual and interaction planning.

### 2026-08-20 — Preserve the current selector UI only as a transition

Decision: relocate the current checklist selector after release to maintain a
working end-to-end product. Its visual redesign is explicitly deferred.

### 2026-08-20 — Backend separation is mandatory

Discovery: the current Go/OpenAPI contract requires exact selected questions
before submission and pins them through approval/release. A frontend-only hide
would be false and unsafe; the plan/package snapshot split is required.

### 2026-08-20 — Reuse Avia's visual system

Decision: apply the skill's accessible trust-and-authority principles through
the existing tokens and font stack. Reject its landing-page/exaggerated
minimalism output and add no new visual dependency.

### 2026-08-20 — Parallel Sol max and Sol ultra reviews repaired the plan

Both requested independent reviews completed read-only against the current
React, Go, OpenAPI, migration, mock, and assignment code. Their overlapping
Critical/Important findings were accepted and repaired:

- add the immutable complete approver dossier and Review→Finance field parity;
- split Planning RELEASED from Audit-package FINALIZED with an explicit state
  machine and exact approved checklist-item ceiling;
- include the actual clean-state creation application/handler path;
- use one atomic cutover so no runnable checkpoint loses end-to-end flow;
- replace unsupported down/up migration claims with the forward-only runner's
  real qualification matrix;
- add per-status data cutover, post-release ensure/read/reload semantics, and a
  new additive route identity;
- make purpose presets, location resolution, workload provenance, and roster
  context server-owned and typed;
- retain all historical compatibility dimensions except location, which alone
  becomes a weighted/explained signal;
- bound “unchanged selector UI” to the selector subtree rather than its old
  wizard/budget chrome; and
- add successful-route focus management and exact runnable verification
  commands.

The Department Manager team-assignment ownership transfer remains explicitly
deferred to a mandatory successor plan; this plan no longer implies that the
full post-approval Department Manager workspace is completed.

## Current progress

- [x] Read the user-named `ui-ux-pro-max` skill completely.
- [x] Run its required design-system search and focused UX/product searches.
- [x] Inspect the current New Audit React component, tests, CSS, routes,
  planning intake types, Go validation, migrations, approval snapshot binding,
  mock engine, and post-release assignment boundary.
- [x] Render the latest current checkout through steps 1–5 at 1440×900 and
  Scope at 390×844; record the current UI strengths/defects and exclude older
  parity/manual-review screenshots from design authority.
- [x] Read the matching completed 2026-08-17 Planning Intake UX plan completely.
- [x] Record the new Department Manager decisions in product documentation.
- [x] Draft this self-contained successor ExecPlan.
- [x] Complete parallel Sol max and Sol ultra reviews.
- [x] Apply validated review corrections and record review outcomes.
- [x] Execute Gate 0 on the current branch: root and Surveil dirty-tree status,
  focused wizard tests, typecheck, current contract coupling, and latest-code
  Browser baseline are recorded; unrelated dirty docs/test-catalog changes
  remain untouched.
- [x] Run the `ui-ux-pro-max` design-system search for an accessible,
  trust-and-authority governance editor. Existing Avia tokens, system fonts,
  no gradients, no external fonts, and no new icon/UI packages remain the
  implementation authority.
- [x] Recompose the React surface as `New Audit` with Scope → Purpose →
  Schedule → Resources & budget → Review, a progressive Audit plan summary,
  inline validation, autosave presentation, conditional location/meeting
  details, read-only workload preview, and mobile compact progress disclosure.
- [x] Add the proposal-only web/mock contract, persistent mock state, HTTP
  adapter paths, OpenAPI source fragments, generated transport artifacts, and
  additive `ui-audit-087` post-release preparation route boundary.
- [x] Add forward-only migration `000057_planning_proposal_boundary.up.sql`
  with separate mutable proposal and immutable proposal snapshot tables; the
  migration source/tests compile locally and the disposable bootstrap gate is
  `verified locally`, while full cutover fixtures remain pending.
- [x] Verify the focused React suite (`14/14`), focused mock E2E, contract
  generation/check, Go planning/HTTP tests, and Playwright rendered flow.
- [x] Complete the first rendered QA pass at 1440×900, 1024×768, 768×1024,
  and 390×844 with screenshots in `/tmp`, clean console output, and zero
  horizontal overflow. The in-app Browser screenshot endpoint and native date
  segment entry remain `blocked`; isolated Playwright screenshot/date fallback
  is `verified locally`.
- [x] Run `test:e2e:visible-actions`: `4 passed` across desktop, tablet,
  mobile, and command execution. Run `test:e2e:accessibility`: `5 passed`
  across 85 legacy visual routes and focus traps. The additive post-release
  route is covered by its own route/component contract rather than historical
  parity fixtures.
- [x] Extract the bounded post-release selector and wire idempotent
  proposal-linked setup/read/finalization with the approved checklist-item
  ceiling; verify the mock release → selection receipt → finalization flow.
- [ ] Finish full migration cutover fixtures, connected HTTP qualification,
  and the complete repository verification matrix.

## Current outcome

The current checkout contains a candidate New Audit Planning-proposal flow and
the supporting proposal/package contract. The normal demo path is `verified
locally` through Scope → Purpose → Schedule → Resources & budget → Review →
Finance submission, with zero budget accepted and no Audit/Inspector assignment
created by proposal submission. The mock post-release path is also `verified
locally` through governed selection receipt and ceiling-checked package
finalization. The result is `candidate-only` and `release pending`; production
readiness is not claimed. Full data cutover fixtures and connected HTTP
qualification remain `not run` or `blocked` pending their local runtime
evidence.

## Implementation evidence — 2026-08-20

- React focused New Audit suite: `verified locally` — `14/14` tests passed.
- Full React suite: `verified locally` — `95` files and `731` tests passed.
- React typecheck: `verified locally` — exit `0`.
- Demo and HTTP builds: `verified locally` — both build; existing Vite
  ineffective-dynamic-import and chunk-size warnings remain non-blocking.
- OpenAPI/codegen: `verified locally` — `contracts:check` passed with `16/16`
  contract tests and generated TypeScript/Go outputs current.
- Go focused gates: `verified locally` — planning, HTTP, assignments, and
  migration packages pass; `go -C apps/api test ./...` compiles all packages but the
  integration package is `blocked` when the task-owned PostgreSQL runtime is
  not running (`127.0.0.1:55432` refused).
- Qualification bootstrap: `verified locally` — the final `make
  qualification-bootstrap-check` created the task-owned PostgreSQL/catalog
  fixture, applied migration head `57` including the proposal-linked
  post-release scope columns, and completed the foundation/roster/catalog/
  permission checks; the runtime was cleaned up.
- Mock E2E: `verified locally` — New Audit proposal flow passed at 390×844.
- Visible actions/accessibility: `verified locally` — visible-actions `4
  passed`; accessibility `5 passed`.
- Post-release selector: `verified locally` — focused mock lifecycle test
  passed (`1/1`) for release → governed catalog → explicit selection receipt →
  approved-ceiling package finalization. Connected HTTP selector execution is
  `blocked` with the task-owned runtime unavailable.
- Browser/visual QA: `verified locally` — isolated Playwright screenshots and
  interactions passed at 1440×900, 1024×768, 768×1024, and 390×844 with clean
  console output and no horizontal overflow. In-app Browser page identity,
  DOM, focus, inline-error, and console checks passed; its screenshot endpoint
  and native date segment entry are `blocked`, so Playwright is the recorded
  screenshot/date fallback.
- Repository gates: `verified locally` — root `make check` passed (`73/73`
  Python tests plus harness/validate/diff checks); Surveil harness smoke and
  `git diff --check` passed.

## Execution Prompt

```text
Implement docs/exec-plans/active/2026-08-20-demo-mvp-new-audit-planning-redesign-plan.md
from /Users/marlonjd/Developer/monorepos/avia/apps/surveil on the current branch.

Read AGENTS.md, docs/PLANS.md, the plan, verification matrix, output contract,
the 2026-08-20 Department Manager decision record, and the completed 2026-08-17
Planning Intake UX plan completely. Preserve unrelated dirty-tree changes. Do
not create/switch branches, commit, push, deploy, modify public infrastructure,
or mutate external systems without explicit current authorization.

Start at Gate 0. Prove the current frontend/API coupling before editing. Then
execute Tasks 1–8 in order, keeping each task locally passing before continuing.
Complete Gate 0.5 product decisions before source changes. Treat Tasks 2–6 as
one atomic cutover: build dormant additive schema/API/selector/projection first,
switch all layers together only after the full replacement lifecycle passes,
then delete obsolete paths.

Do not hide checklist selection while leaving it required by the backend.
Separate immutable Planning proposal snapshots from post-release Audit-package
scope snapshots. Add the exact approver dossier and pin it through Finance/GM/
Executive decisions. Use Planning DRAFT/SUBMITTED/RELEASED and package DRAFT/
SELECTION_CONFIRMED/FINALIZED as distinct states. Enforce selected count <=
approved checklist-item ceiling. Remove obsolete active-write fields; do not
add dual-read UI fallbacks. Apply the forward-only status cutover matrix and
preserve immutable history. Preserve the current Finance → GM → Executive
Director → GM Release chain, privacy boundaries, expected revisions,
idempotency, and audit evidence.

Preserve the latest current UI's five-step desktop composition and build New
Audit as Scope → Purpose → Schedule → Resources & budget → Review. Use `New
Audit` and `Inspected Organization`. Keep the current open editor, sticky
action bar, horizontal desktop progress, and progressively simplified `Audit
plan summary` rail. On phone, replace the tall five-row progress block with a
compact current-step disclosure. Do not use historical parity/manual-review
screenshots as design authority.

Auto-select single provider/target values without dropdowns and show them as
compact human-readable context. Remove Domain, approach/category, Trigger,
Risk Category, historical recommendation, and exact checklist selection from
New Audit. Add purpose presets + editable text, conditional canonical
location/remote meeting link, required inspector count, editable checklist-item
estimate with server range, read-only opt-in workload preview, budget, and
decision-ready Review.

Relocate the existing checklist selector after release without visually
redesigning its bounded recommendation/search/filter/list/selection/dialog/
dossier subtree. Do not carry over the old wizard stepper, summary rail, Budget,
or footer. Add idempotent ensure/read, reload/restart/two-tab protection, exact
selection consent, bounded digest chain, receipts, and package finalization.
Its detailed UX and Planning-amendment flow are future plans.

Use the existing Avia tokens and system font. Apply Accessible & Ethical,
Trust & Authority, progressive disclosure, one primary action, inline recovery,
44px targets, semantic controls, vector icons, restrained motion, and
responsive no-overflow behavior. Do not add a UI framework, external font,
icon package, gradient, decorative hero, fake KPI, or generic card dashboard.

Run the plan's focused tests continuously, then the full verification matrix.
Use an isolated browser profile at 1440×900, 1024×768, 768×1024, and 390×844.
Verify keyboard, focus return, first-invalid focus, 200% zoom, reduced motion,
autosave failure/retry, all location modes, workload-range behavior, blank/zero
budget, return/resubmit, console, privacy, and no overflow. Clean task-owned
browser/dev/test processes. Update the plan and index with literal evidence
labels; do not claim deployment or production readiness.
```
