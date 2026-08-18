# Department Manager Planning Intake UX Redesign

Date: 2026-08-17
Last updated: 2026-08-18
Status: ready-for-verification — implementation, connected qualification, public release/app-shell verification, Chrome + Computer Use browser-control checks, root `make check`, and explicit stakeholder acceptance are recorded; IAB-native zoom/reduced-motion/full-Tab controls are outside the revised acceptance boundary; production readiness not claimed

## Planning authority

This plan follows [`docs/PLANS.md`](../../PLANS.md), the repository-local
[`AGENTS.md`](../../../AGENTS.md), the
[`verification matrix`](../../agent-harness/verification-matrix.md), and the
[`output contract`](../../agent-harness/output-contract.md).

The user explicitly requested a senior-design-manager-quality redesign plan,
use of `ui-ux-pro-max` and `ui-self-critique`, local in-app Browser
verification during implementation, a durable ExecPlan, and a self-contained
execution prompt. Planning does not authorize commit, push, deployment,
traffic, or release actions.

## Objective and user-visible outcome

Redesign the Department Manager Planning queue entry point and the complete
New Inspection intake so a competent aviation oversight user can understand
what to do without knowing the server's draft, digest, batching, or aggregate
terminology.

The finished workflow must have one obvious next action at every step. Selecting
an authorized supplier tuple and pressing `Continue` must create the governed
server draft and advance in one action. The separate visible `Open audit setup
for this supplier` and `Save draft` controls must be removed. After draft
creation, the form must autosave and show a compact, truthful save state.

The user-visible outcome is:

1. Planning opens with a task-oriented queue and a clear `New inspection`
   action rather than technical slogans and raw aggregate identity.
2. New Inspection is one coherent five-step editor: Basics, Purpose, Schedule,
   Checklist & budget, Review.
3. A persistent, product-specific `Inspection brief` summarizes the exact
   supplier, authority scope, notice policy, schedule, checklist count, budget,
   and save state without exposing opaque identifiers as primary content.
4. Validation appears beside the field that needs attention, moves focus to the
   first invalid field, and explains recovery. The page does not jump to a
   detached top-level error banner for one field.
5. The 1,310-question catalog opens on the server's recommended subset and uses
   progressive disclosure. Advanced filters and immutable identities remain
   available without dominating the primary task.
6. The user reviews and confirms one intelligible selection change. The client
   may execute bounded digest-chained batches internally, but it does not make
   the user repeat `Preview next exact batch` and `Confirm selection` for every
   500-question protocol batch.
7. Review is written for a decision maker: readable sections, edit links,
   one `Submit to Finance` action, clear progress, and no primary-display
   draft/digest noise.
8. Desktop, tablet, and phone layouts remain usable, keyboard accessible, and
   visually coherent in the local in-app Browser.

## Product classification and design direction

### Concrete surface, role, and job

- Surface: Department Manager Planning queue and New Inspection intake.
- Role: Department Manager acting as the accountable planning authority.
- Single job: define a reviewable inspection brief and submit the resulting
  Planning item to Finance without accidentally creating an executable Audit.
- Product shape: a governance editor with a catalog-selection workspace, not a
  marketing page, dashboard, generic wizard, or developer console.

### Signature element

The signature element is the `Inspection brief`: a persistent summary of the
governed plan that grows as the user completes the workflow. On wide screens it
is a restrained right rail aligned with the editor. On narrow screens it is a
compact collapsible summary above the sticky action bar. It is specific to the
aviation planning workflow and must show:

- supplier and regulated target in human language;
- provider scope and application type;
- announced/unannounced notice consequence;
- planned date, mode, and location;
- selected question count and recommendation basis;
- requested budget and currency;
- `Not saved`, `Saving…`, `Saved`, or `Couldn't save — Retry` state;
- no raw draft ID, scope ID, digest, or question-version ID in the primary
  summary.

### Visual system

Use the existing Avia workbench semantic tokens and system font stack from
`apps/web/src/styles/tokens.css`. The `ui-ux-pro-max` research supports a
trust-and-authority direction: navy, white, slate, one action blue, restrained
success green, high contrast, and subtle motion. Do not add external fonts,
gradients, decorative badges, glass effects, illustrations, fake metrics, or
new package dependencies.

The visual hierarchy must come from:

- an intentional content max-width rather than the current near-full-window
  stretch;
- one primary heading and one compact five-step progress indicator;
- open layout and section dividers rather than nested cards around every
  group;
- a consistent 4/8px spacing rhythm;
- deliberate control typography and 44px minimum targets;
- one primary action per screen;
- semantic, token-driven error, success, focus, selected, disabled, and loading
  states;
- at most 150–250ms state transitions, disabled by `prefers-reduced-motion`.

## Baseline findings

The user-supplied 2026-08-17 screenshots and current source show these
reportable defects:

1. `Open audit setup for this supplier`, `Save draft`, and `Next` compete on
   the first screen. `Next` is disabled until the user discovers the separate
   draft-creation command. This exposes server implementation order as product
   navigation.
2. Direct loading of a later route without a draft can render the requested
   step heading while showing step-one setup controls. Route identity and
   visible content disagree.
3. Three macro stages and five substeps are shown simultaneously, with repeated
   copy such as `Stage 1 of 3 · substep 2 of 5` and `Set up — Step 2 of 5`.
   Orientation costs more attention than it saves.
4. The progress descriptions wrap into narrow vertical columns at large
   desktop widths. The page uses excessive whitespace above a wide generic
   panel and feels unfinished rather than calm.
5. Required fields are not visibly marked. Validation occurs after `Next` and
   produces a large detached banner above the form instead of an inline message
   beside `Purpose`, `Planned Date`, or `Location`.
6. The first screen repeats the exact supplier tuple in a large technical
   notice and exposes `SCOPE-NAMIBIA-DEMO-AGA-QUALIFICATION` as visible option
   copy. The user needs scope meaning, not storage identity.
7. Step four places search, seven filters, recommendation explanation, 25
   dense question cards, immutable identities, selected tray, five selection
   commands, pagination, catalog version, and budget in one uninterrupted
   surface.
8. The visible commands `Stage`, `Preview next exact batch`, `Confirm
   selection`, and repeated 500-question commits describe the transport
   protocol rather than the Department Manager's intent.
9. The default question presentation uses equal visual weight for prompt,
   technical identity, advisory states, risk, and dossier actions. Suggested
   questions are not the clear starting point.
10. Question dossier content is materially better structured than the main
    catalog, but keyboard return-focus, close semantics, long-identity
    treatment, and mobile sheet behavior need explicit verification.
11. The queue leads with technical messages and, in the public screenshot, raw
    `plan-intake-*` IDs and repetitive records. Human-readable reference work is
    already active in the separate Demo Record Presentation plan and must not
    be overwritten.
12. The current `NewAuditWizardPage` owns loading, five-step form state,
    autosave candidates, catalog filters, dossier, selection batching, review,
    and submission in one large component, making interaction refinement
    fragile.

## Scope

### Included

- Department Manager Planning page composition around the queue and New
  Inspection entry action.
- All five New Inspection routes under
  `/department-manager/new-audit/step-{1..5}`.
- Draft creation, autosave presentation, dirty/saving/saved/error states, and
  predictable Back/Cancel behavior.
- Inline validation, required markers, focus management, status feedback, and
  loading states.
- Supplier/provider scope/target/type presentation while preserving exact
  server-owned values.
- Question discovery, recommendation-first defaults, advanced filters,
  selection summary, review/confirmation, bounded batch progress, dossier, and
  pagination.
- Review-and-submit information architecture.
- Responsive desktop/tablet/phone composition and keyboard accessibility.
- Focused unit, mock E2E, connected qualification, and in-app Browser visual
  verification.
- Refactoring the current monolithic page into focused planning-intake
  components when this reduces state and test complexity.
- Coordination with the active Demo Record Presentation plan for short human
  references.

### Explicit exclusions

- No change to Planning approval order, Finance/GM/Executive authority, Audit
  materialization, Inspector start, or notice privacy.
- No synthetic production data, fake supplier choices, fabricated budgets, or
  queue-data cleanup. Repetitive public demo records are a separate data and
  environment concern.
- No change to canonical catalog contents, AI recommendation policy, regulatory
  meaning, or legal interpretation.
- No weakening of exact immutable question identity, selection digest,
  append-only receipts, expected revision, idempotency, or the 500-item batch
  boundary.
- No replacement of native controls with inaccessible custom widgets merely
  for visual style.
- No new UI library, icon package, font dependency, animation library, or broad
  design-system rewrite.
- No redesign of Finance, GM, Executive, Inspector, Auditee, or Admin surfaces.
- No public deployment, release lock, commit, push, branch operation, or
  infrastructure change under this plan alone.

## Assumptions and ownership boundaries

- `apps/surveil` owns the React UI and application behavior in this plan.
- AviaWorkspace owns public/cloud release and is not modified by the UI
  implementation unless a later separately authorized release task is opened.
- Existing backend operations are sufficient: list authorized scope options,
  create/get/save/submit draft, list catalog, preview selection, and commit
  selection. The first implementation attempt must remain frontend-only.
- If a necessary interaction cannot be represented without an API change, stop
  and record the exact contract gap before editing Go/OpenAPI.
- Direct routes remain supported. A later-step route without `draftId` keeps its
  exact URL but presents the safe Basics entry state; only a server-owned draft
  can render downstream Purpose, Schedule, Checklist, or Review state.
- A draft is created once, on the user's first valid `Continue` action. That
  action is explicit enough to satisfy the existing server-owned scope
  boundary; a second visible `Open audit setup` action is not required.
- After draft creation, autosave is the persistence model. `Continue` must
  flush any queued save before navigation. Concurrent saves are serialized and
  revision-aware.
- Changing the exact supplier tuple after later-step data exists is destructive
  to the current selection context. It requires a clear consequence dialog and
  an explicit confirmation before opening a replacement draft.
- Preserve all unrelated dirty-worktree changes. The active Demo Record
  Presentation plan owns shared short-reference helpers; reuse its result and
  do not recreate or revert it.

## Repository orientation and affected interfaces

Primary implementation surfaces:

- `apps/web/src/features/planning/new-audit-wizard.tsx`
- `apps/web/src/features/planning/new-audit-wizard.test.tsx`
- `apps/web/src/features/planning/planning-workspaces.tsx`
- `apps/web/src/styles/features/planning-intake.css`
- `apps/web/src/styles/tokens.css` only for missing semantic tokens that are
  reusable beyond this page
- `apps/web/src/styles/responsive.css` only when shell-level breakpoints are
  required
- `apps/web/tests/e2e/canonical-scenario.spec.ts` or a focused new
  `planning-intake-ux.spec.ts`
- `apps/web/tests/e2e/qualification-all-role.spec.ts` for the connected flow

Expected component split, if source inspection confirms it reduces coupling:

- `planning-intake-progress.tsx`
- `planning-intake-brief.tsx`
- `planning-intake-actions.tsx`
- `planning-intake-field.tsx` or an equivalent lightweight field/error helper
- `planning-intake-catalog.tsx`
- `planning-intake-selection-review.tsx`
- `planning-intake-review.tsx`
- a small autosave hook/state helper colocated under the planning feature

Do not split files solely to hit a line count. State ownership must remain
obvious and the route component must remain composition glue rather than a new
indirection layer.

## Required lifecycle invariants

The redesign is acceptable only if these facts remain true:

1. The exact supplier organization, provider scope, regulated target,
   application type, catalog version, and usage class come from the server.
2. The client cannot author or substitute a scope ID, catalog, selection digest,
   aggregate ID, revision, or immutable question identity.
3. `Continue` from step one creates at most one idempotent draft and navigates
   only after the authoritative draft is returned.
4. Back/forward, unmount, direct reload, and runtime restart restore the same
   authoritative draft and current values.
5. Routine/Announced and Ad Hoc/Unannounced notice behavior remains exact;
   withheld purpose/risk/location does not leak to Auditee projections.
6. Blank budget is invalid; literal zero remains valid and still enters Finance
   Review.
7. Question selection remains an explicit Department Manager decision. Offline
   recommendations are advisory; they never silently commit a selection.
8. Preview and commit commands stay digest-bound, idempotent, revision-safe,
   and limited to 500 unique identities per batch.
9. One user confirmation may orchestrate multiple bounded batches, but the UI
   must show progress, stop on the first error, retain completed receipts, and
   offer a safe retry/resume path. It must not imply atomic completion before
   every required batch succeeds.
10. Submit creates a Planning item in Finance Review and does not create an
    executable Audit or expose withheld notice.

## Ordered implementation

### Gate 0 — Preserve and prove the baseline

1. Read the current `AGENTS.md`, this plan, Demo Record Presentation plan,
   verification matrix, and output contract.
2. Run `make repos-status` from AviaWorkspace and `git status --short` in
   `apps/surveil`. Record all pre-existing edits that overlap the target files;
   do not reset or overwrite them.
3. Run the current focused wizard tests and typecheck before changing source.
4. Start the local demo on `127.0.0.1` and use Chrome + Computer Use to capture
   the current step-one, inline-error absence, step-four density, selection
   commands, dossier, and phone behavior. Do not use the everyday Chrome
   profile.
5. Record the baseline mismatch ledger in this plan: current evidence, intended
   repair, and any deliberate non-change.

Expected observation: the current flow reproduces the double action on step
one, detached errors, duplicate progress hierarchy, and protocol-heavy catalog
without unexpected runtime errors.

### Gate 0 evidence — 2026-08-17

- `make repos-status` is `verified locally`: `auth main clean`, `data main
  clean`, and `surveil main dirty (71 paths)`.
- Workspace root is on `main` with unrelated untracked `output/` and
  `presentation/` evidence. The Surveil checkout is on `main`; its existing
  dirty work includes offline/API work plus overlapping planning changes in
  `apps/web/src/features/planning/new-audit-wizard.tsx`,
  `apps/web/src/features/planning/new-audit-wizard.test.tsx`,
  `apps/web/src/features/planning/planning-workspaces.tsx`,
  `docs/exec-plans/index.md`, and the two active 2026-08-17 planning plans.
  These changes were preserved; no reset, clean, checkout, or branch operation
  was run.
- `npm --prefix apps/web test -- --run
  src/features/planning/new-audit-wizard.test.tsx` is `verified locally`:
  `1` test file and `18/18` tests passed.
- `npm --prefix apps/web run typecheck` is `verified locally` with exit `0`.
- Current source baseline is `verified locally` by inspection: the wizard is a
  946-line monolithic route component; the focused test is 397 lines; the
  planning workspace is 489 lines; and `planning-intake.css` is 615 lines.
  Existing user changes already default catalog browsing to `SUGGESTED_NOW`
  and reuse the shared `record-presentation` helper in queue presentation. The
  UX implementation must preserve those edits and the unrelated dirty tree.
- Local demo `npm --prefix apps/web run dev:demo -- --host 127.0.0.1` is running
  at `http://127.0.0.1:5173/` as a task-owned Vite process.
- IAB step-one identity is `verified locally`: URL
  `http://127.0.0.1:5173/department-manager/new-audit/step-1`, title
  `AviaSurveil360 — Civil Aviation Oversight`, meaningful Department Manager
  content, no framework overlay, and no error/warn console logs. The visible
  baseline has `Open audit setup for this supplier`, `Save draft`, disabled
  `Next`, three macro stages plus five substeps, raw provider scope identity,
  and no Inspection brief.
- IAB direct later-route mismatch is `verified locally`: loading step four
  after a local draft reload produced `Planning intake draft ... was not
  found`, while still rendering the step-four heading alongside step-one
  supplier/setup controls and the `Open audit setup for this supplier` action.
- IAB schedule validation is `verified locally` as a baseline defect: the
  current UI renders `Planned date is required` in a detached `role=alert`
  above the form and does not move focus to the invalid field.
- IAB phone baseline at `390x844` is `verified locally`: the shell collapses
  its navigation, the three-stage progress copy becomes a tall vertical stack,
  the detached alert remains above the form, and the action bar is not a
  persistent safe-area action surface. Document width was 375 CSS pixels with
  no horizontal overflow in this state; console error/warn logs were empty.
- Full rendered step-four catalog/dossier interaction is `not run`: the
  disposable demo draft was lost on the full route reload, the seeded draft
  no longer matched the current authorized catalog/scope tuple, and the IAB
  native date control did not accept automated segment entry. Source inspection
  confirms the baseline protocol controls are `Stage suggested questions`,
  `Stage all matching eligible questions`, `Preview next exact batch`,
  `Confirm selection`, and `Undo staged changes`, plus a raw immutable-ID tray
  and dossier. This is a baseline gap, not a passing visual result.

Baseline mismatch ledger:

| Evidence | Intended repair | Deliberate non-change |
|---|---|---|
| Separate draft-opening command, manual Save draft, disabled Next | First valid Continue creates the server draft; autosave owns later persistence | Server-owned creation, revision, and idempotency remain unchanged |
| Three macro stages plus five substeps wrap and repeat | One compact five-step model | Existing shell navigation remains |
| Direct later route can show step-four heading with step-one controls | Canonicalize a route without a valid draft to step one | Direct links remain supported |
| Detached validation alert and no first-invalid focus | Inline field errors with accessible focus and recovery | Server validation semantics remain authoritative |
| Raw scope/question IDs and protocol batch controls dominate the catalog | Human-readable rows, collapsed technical details, one selection-review decision | Exact IDs, digests, revisions, receipts, and 500-item batches remain in transport |
| Phone progress is a tall vertical stack and actions are not sticky | Collapsed brief, compact progress, safe-area sticky actions, no horizontal overflow | Existing mobile shell behavior remains |

Gate 0 status: the focused source/test/browser baseline is `verified locally`;
the full baseline catalog/dossier path is `not run` for the explicit reasons
above. The result remains `candidate-only`, `release pending`, and production
readiness not claimed.

### Gate 1 — Create and approve the complete design concept

Use the frontend-app-builder concept workflow before coding. Generate a
coherent concept suite, not a header-only mockup:

1. Desktop 1440×900 — step one before draft creation.
2. Desktop 1440×900 — step four with recommended questions and a populated
   Inspection brief.
3. Desktop 1440×900 — review/submit state.
4. Desktop error state showing inline validation and focused recovery.
5. Phone 390×844 — one setup step with collapsed brief and sticky action bar.
6. Phone 390×844 — checklist selection with filters collapsed and selected
   summary reachable.

The concept must preserve the existing shell and brand palette while replacing
the content architecture. All controls and text are code-native. Use no
decorative hero, eyebrow, marketing badges, gradient CTA, fake metrics, or
dashboard card grid.

Run `ui-self-critique` before asking for approval:

- identify anything that resembles a generic SaaS onboarding wizard;
- remove at least one redundant visual element or copy line;
- verify the Inspection brief is workflow-specific rather than a generic
  sidebar;
- ensure only one primary CTA exists in each state;
- verify step-one, step-four, review, error, and mobile concepts are complete.

Stop for user approval of the concept suite. After approval, treat it as the
visual specification and record the accepted concept paths in this plan.

Accepted concept suite — user approval recorded 2026-08-17:

1. Desktop 1440×900 — Basics before draft creation:
   `/Users/marlonjd/.codex/generated_images/01a01085-ef87-7381-a0df-521b8159068d/exec-14577181-7494-46c3-92f8-976d36adaa15.png`
2. Desktop 1440×900 — Checklist & budget, recommendation-first:
   `/Users/marlonjd/.codex/generated_images/01a01085-ef87-7381-a0df-521b8159068d/exec-04094d99-cf54-4d33-81d1-0cd93e8b63ef.png`
3. Desktop 1440×900 — Review and submit:
   `/Users/marlonjd/.codex/generated_images/01a01085-ef87-7381-a0df-521b8159068d/exec-f0f1c41c-76e6-4f85-99fa-a83fc3fd365d.png`
4. Desktop 1440×900 — Inline validation and focused recovery:
   `/Users/marlonjd/.codex/generated_images/01a01085-ef87-7381-a0df-521b8159068d/exec-d32aad7a-cbd8-47bf-822d-a445d8f8c97d.png`
5. Phone 390×844 — Purpose setup with collapsed brief and sticky actions:
   `/Users/marlonjd/.codex/generated_images/01a01085-ef87-7381-a0df-521b8159068d/exec-2935036a-1c3f-47b5-b4d5-567f278a1f05.png`
6. Phone 390×844 — Checklist selection with collapsed filters and reachable selection review:
   `/Users/marlonjd/.codex/generated_images/01a01085-ef87-7381-a0df-521b8159068d/exec-7da7e572-6c0c-4808-9017-42dad7c0d0dc.png`

Supplementary feature-detail concepts added after review feedback:

- Desktop 1440×900 — expanded Advanced filters, bulk actions, pagination,
  selection summary, and details actions:
  `/Users/marlonjd/.codex/generated_images/01a01085-ef87-7381-a0df-521b8159068d/exec-d435fd4e-f263-4f44-bafb-6bf0962beabf.png`
- Phone 390×844 — expanded Advanced filters sheet with all eight filters:
  `/Users/marlonjd/.codex/generated_images/01a01085-ef87-7381-a0df-521b8159068d/exec-c8ca87c9-b433-4673-b6e6-9fecaa1d33ec.png`

The pre-approval `ui-self-critique` is `verified locally` as a design review:
the generated research direction's purple/gradient/external-font suggestions
were rejected in favor of the existing Avia tokens; the right rail was kept
strictly inspection-plan-specific; the redundant mobile supplier/provider
summary was removed from concept 5; and the equally weighted checklist
`Continue` action was made secondary so `Review selection` is the only primary
decision in concepts 2 and 6. The candidate suite contains no decorative hero,
generic dashboard card grid, fake KPI, repeated badge, raw identifier, or
protocol control in primary presentation.

### Task 1 — Simplify routing, progress, and first-draft creation

1. Replace the three-stage/five-substep combination with one five-step model:
   `Basics`, `Purpose`, `Schedule`, `Checklist & budget`, `Review`.
2. Make the progress indicator compact and non-interactive unless completed
   step navigation is deliberately supported and state-safe.
3. If a later-step URL has no valid `draftId`, replace-navigate to step one so
   the URL, heading, progress, fields, and action bar agree.
4. Remove visible `Open audit setup for this supplier` and `Save draft`
   controls.
5. Use one primary `Continue` action on step one. When the exact tuple is valid,
   it creates the authoritative draft with a loading label such as `Creating
   draft…`, stores the returned revision, and navigates to step two.
6. Keep `Cancel` secondary. If no draft or edits exist, Cancel returns to
   Planning directly. If unsaved local edits or an autosave failure exist,
   confirm before leaving.
7. Replace raw scope identifiers in visible labels with provider type,
   authorization identifier when meaningful, and target label. Preserve exact
   IDs in values, API calls, data attributes, and optional technical details.
8. When revisiting step one with later-step content, changing the tuple must
   explain that checklist context will be replaced and require confirmation.

Expected observation: a first-time user selects or accepts the valid tuple and
presses one `Continue` button; no hidden prerequisite action remains.

### Task 2 — Add serialized autosave and field-level validation

1. Introduce an explicit autosave state machine: `clean`, `dirty`, `saving`,
   `saved`, `error`.
2. Debounce eligible field changes after draft creation. Serialize writes so
   only one expected-revision save is in flight; coalesce later changes and
   save them against the returned revision.
3. Make `Continue` validate the current step, flush the latest autosave, then
   navigate. It must not race a background request or restore stale values.
4. Show `Saving…`, `Saved`, and `Couldn't save — Retry` in the Inspection brief
   and action bar. Never use a toast as the only persistence evidence.
5. Add visible required markers and concise helper text where a field's meaning
   is not obvious.
6. Validate on blur and on Continue. Render errors beneath the owning field,
   set `aria-invalid`/`aria-describedby`, announce the error, and focus the
   first invalid control.
7. Use a top error summary only when multiple fields are invalid or a page-level
   server failure has no field owner. The summary must link/focus the affected
   fields and include a recovery action.
8. Preserve raw budget input semantics: blank is invalid; zero is valid.

Expected observation: the user never has to decide whether to press Save. A
failed save is visible and recoverable without losing typed values.

### Task 3 — Recompose steps one through three as an inspection brief editor

1. Use one main editor column and the Inspection brief rail at desktop widths.
2. Group step-one controls by dependency: Supplier → Provider scope → Regulated
   target → Inspection type. Disable dependent controls with explanatory helper
   text only while upstream data is unavailable.
3. Remove the large repeated `Assign the supplier before opening the Audit`
   callout. Replace it with a short, calm statement near the action: `Continuing
   creates a Planning draft; it does not create an Audit.`
4. Step two must explain the operational consequence of Routine/Announced vs
   Ad Hoc/Unannounced next to the category control. Notice policy is a derived
   read-only status in the brief, not a large generic card.
5. Purpose is the primary field on step two. Trigger and configured risk are
   secondary/read-only when the backend contract does not permit meaningful
   user choice.
6. Step three must use clear date, mode, and location grouping, real labels,
   localized display, and inline required state. Avoid a giant empty textarea
   or input solely to fill horizontal space.
7. On tablet/phone, collapse the brief to a summary disclosure and keep the
   action bar reachable without covering content.

Expected observation: each screen communicates one decision, one consequence,
and one next action.

### Task 4 — Redesign checklist and budget selection around user intent

1. Rename the step consistently to `Checklist & budget` everywhere.
2. Default to the server's `SUGGESTED_NOW` result and state how many suggested
   questions are shown. Never label the default surface as all 1,310 when a
   recommendation filter is active.
3. Keep Search visible. Move Form, Domain, Topic, Risk, Checklist focus, Source
   gap, Recommendation, and Selected state into an `Advanced filters`
   disclosure with active-filter count and a clear reset action.
4. Present questions as a readable review list, not equal-weight technical
   cards. Lead with prompt and form/item reference; show risk and recommendation
   reason selectively; place immutable identity inside dossier/technical
   details.
5. Make selection direct and obvious. Each row needs a native checkbox with a
   44px target, selected state, and one secondary `View details` action.
6. Provide one clear bulk action, `Use suggested questions`, and subordinate
   `Add all matching eligible questions` behind an advanced/overflow action
   with exact count and consequence.
7. Keep a sticky or adjacent selected summary that shows count, additions,
   removals, estimated resource requirement when server-derived, and a `Review
   selection` action. Do not render 100 raw immutable identities as the primary
   tray.
8. `Review selection` opens an accessible summary dialog/sheet. One user
   `Confirm selection` action then runs the existing preview/commit chain in
   bounded batches. Show truthful progress (`500 of 1,310 confirmed`), stop on
   error, and permit retry/resume.
9. Preserve an explicit `Undo changes` before confirmation and keep already
   committed receipts immutable.
10. Keep pagination for server results. If rendered result density still
    exceeds 50 rows in a future mode, retain bounded page rendering rather than
    loading all bodies into the DOM.
11. Move catalog version to a compact technical-details disclosure. Put Budget
    and Currency in a clear `Resources` section after the selection summary.
12. Dossier opens with focus trapped, Escape/Close support, background inert,
    and focus returned to the triggering question. On phone it becomes a
    full-height sheet with readable long identifiers.

Expected observation: the default checklist screen answers `What is suggested,
why, what have I selected, and what happens next?` without exposing batching
protocol as the workflow.

### Task 5 — Make review and submission decision-ready

1. Replace the undifferentiated definition grid with readable sections:
   `Inspection`, `Schedule`, `Checklist`, `Resources`, `Notice & governance`.
2. Add `Edit` links that return to the owning step without losing state.
3. Lead with human labels and short references. Put draft ID, catalog version,
   exact digest, and revision in collapsed technical details.
4. Show the governance route once, in plain language, and clarify that submit
   creates a Planning item for Finance rather than an Audit.
5. Remove the separate Preview toggle. The review page itself is the preview.
6. Use one primary `Submit to Finance` action with loading, success, and error
   states. Keep Back secondary.
7. After success, navigate to the Planning queue, select the new human-readable
   record, and show a concise confirmation with current owner and next action.

Expected observation: the Department Manager can review the complete brief,
edit any section, and submit without reading internal terminology.

### Task 6 — Align the Planning queue entry and handoff

1. Replace technical boundary chips above the queue with one concise governance
   sentence or remove them when the page already communicates the rule.
2. Use a clear `New inspection` primary action in the page header.
3. Reuse the Demo Record Presentation plan's business title and short stable
   reference. Do not expose raw `plan-intake-*` IDs in normal rows or button
   labels.
4. Add status/search controls only when they operate on real data. Do not add
   decorative filters or fake summary metrics.
5. Preserve the table-first queue on desktop. On phone, use the repository's
   existing responsive row pattern rather than a generic card dashboard.
6. Keep data quality literal. Do not fabricate varied dates, budgets,
   organizations, or statuses to make the redesign look better.

Expected observation: Planning is a work queue with a clear creation action,
not a marketing header above a database dump.

### Task 7 — Self-critique, fidelity repair, and local verification

1. Run the post-implementation `ui-self-critique` against the accepted concept
   and rendered application.
2. Remove one visual element, card, color use, or copy line that does not serve
   the single job. Record what was removed and why in this plan.
3. Build a fidelity ledger with at least these comparisons: first-viewport
   hierarchy, progress, first-step action model, Inspection brief, inline
   errors, recommendation-first checklist, selected summary, review screen,
   action bar, dossier, and phone composition.
4. Use `view_image` on the accepted concept screenshots and the latest Browser
   screenshots in the same QA pass. Do not claim fidelity from DOM inspection
   alone.
5. Use Chrome + Computer Use for the complete target flow:

   `Planning → New inspection → valid scope → Continue creates draft → Purpose
   → Schedule → recommended checklist → Review selection → Confirm → Budget →
   Review → Submit to Finance → selected queue record`.

6. Verify 1440×900, 1024×768, 768×1024, and 390×844. Capture first viewport,
   error state, checklist state, review state, dossier, and phone state.
7. Verify page identity, nonblank render, no framework overlay, console health,
   no horizontal overflow, no clipped content, no nested scroll trap, and no
   primary control hidden behind the sticky action bar.
8. Run native Chrome keyboard-only traversal with Computer Use. Verify logical order, visible focus, Enter/Space
   activation, Escape dismissal, returned modal focus, and focus on the first
   invalid field.
9. Verify reduced-motion behavior through Chrome DevTools emulation and real
   Chrome 200% zoom without loss of content or
   action access.
10. Clean up task-owned Browser tabs, Vite, Playwright, and browser helper
    processes before closeout.

The revised acceptance boundary uses Chrome + Computer Use for browser-level
zoom, reduced-motion, and native keyboard checks. IAB remains useful
supplementary evidence for the existing viewport flow but is not a required
control surface for this plan. Standalone Playwright remains supplementary for
these browser-level checks.

## Implementation and verification evidence — 2026-08-17

Tasks 1–7 are implemented in the current Surveil checkout. The change remains
frontend-only: source inspection found no hard contract gap, so no backend or
OpenAPI source was changed. The monolithic wizard was kept in one route file;
the extracted ownership boundaries are the progress, Inspection brief,
selection-review dialog, catalog/dossier, and review/action sections. The
existing Demo Record Presentation helper remains the source of human-readable
Planning labels and short references.

Implemented behavior is `verified locally` in focused source tests, IAB viewport
evidence, and Chrome + Computer Use browser-control checks:

- One five-step model is used everywhere: Basics, Purpose, Schedule, Checklist
  & budget, Review. Draftless later routes preserve the requested URL while
  presenting the safe Basics entry state.
- Step one has one Continue action; the first valid Continue creates the
  server-owned draft once. `Open audit setup for this supplier` and manual
  `Save draft` are absent. Autosave is serialized against returned revisions,
  exposes Saved/Saving/Not saved/Couldn't save + Retry, and preserves retry
  intent. The focused wizard suite covers the failure/retry path.
- Validation is inline, accessible, and focuses the first invalid field. Blank
  budget is rejected; zero budget reaches Review and Finance submission.
- The Inspection brief carries supplier, provider, target, type, approach,
  notice, schedule, checklist count, budget, and autosave state. It collapses
  on phone widths.
- The catalog starts from the server `SUGGESTED_NOW` recommendation, keeps
  Advanced filters collapsed initially, exposes Form, Domain, Topic, Risk,
  Checklist focus, Source gap, Recommendation, and Selected state when opened,
  and presents question prompt first with a two-line clamp plus a third
  metadata line. Immutable IDs remain out of primary question presentation and
  are available only in the question dossier/server-owned evidence boundary.
- One Review selection decision runs the existing preview/commit chain in
  bounded 500-item batches with digest-bound receipts, progress, and retry.
  The focused test observed `500/500/310` batches.
- Review has section Edit actions, no Preview toggle, one Submit to Finance
  action, a single governance path, and explicit Audit non-creation and
  Inspector-start separation. The post-submit queue selects a human-readable
  record such as `Routine / Announced — Fly Namibia · Plan #39C8D`.
- The final post-implementation self-critique removed the nonessential
  `Technical details` disclosure from Checklist & budget and Review, after the
  earlier removal of `Summary of the governed plan.`; the brief facts and
  autosave state remain.

Automated evidence:

| Check | Evidence |
|---|---|
| Focused wizard | `verified locally` — `21/21` tests passed after the stale Question dossier close guard |
| Full React regression | `verified locally` — `722/722` tests passed; `0` failed/pending |
| Typecheck | `verified locally` — exit `0` |
| Style ownership | `verified locally` — `8/8` tests passed |
| Demo/HTTP builds | `verified locally` — both build; existing non-blocking dynamic-import and chunk-size warnings remain |
| Focused mock E2E | `verified locally` — `tests/e2e/planning-intake-ux.spec.ts` passed |
| Contracts, artifact, OpenAPI/parity | `verified locally` — contracts `16/16`, artifact scan `79 files / 184 inputs`, OpenAPI/parity `20/20` |
| Root Node smoke suite | `verified locally` — `103/103` tests passed |
| Demo boundary, diff, harness docs | `verified locally` — all passed |
| Connected qualification | `verified locally` — public `namibia/demo` full profile; `qualification-all-role.spec.ts` `1 passed` in 2.4m, including the 1,310-question server selection audit, digest-bound selection, Finance/GM/Executive approval, preparation, materialization, Inspector checklist, Auditee privacy isolation, responsive surfaces, boundary matrix, and credential cleanup; result `/.state/qualification/namibia/demo/results/namibia-demo-20260818t10221787037748z-06cc627fffe34586a800e889055f3612.jsonl` |
| Visible-actions/accessibility suites | `verified locally` — full visible-actions suite `4 passed` (85 surfaces at desktop/tablet/mobile plus command execution); accessibility suite `5 passed` (85 responsive routes at all three viewports plus two focus traps) |
| Root `make check` | `verified locally` — `73/73` Python tests, harness-check, validate, and diff-check passed after the preprod/prod Compose lock refresh and v2 attestation fixture correction |
| Public release and app shell | `verified locally` — exact `namibia/demo` apply `2 added, 5 changed, 2 destroyed`; health `200`; tunnel connected; lock `sha256:ed425897684898da319043d4b70c68ddbd1bee9e00310147f6193814bdb1c1fc`; public app-shell release fingerprint, manifest SHA, and worker SHA match the lock |

In-app Browser evidence is `verified locally` for the primary interaction gate:

- `1440×900`: Basics, Checklist & budget, Review, and submitted Planning queue
  were captured. The queue selected the newly submitted human-readable record;
  console error/warn output was empty.
- `1024×768`: Advanced filters expanded and inline Schedule errors were
  captured. The filter set is discoverable, action bar is in the viewport, and
  `clientWidth === scrollWidth` in both states.
- `768×1024`: tablet shell/layout and Inspection brief were captured with no
  horizontal overflow.
- `390×844`: Purpose setup and checklist selection were captured; the mobile
  brief is collapsed, filters start collapsed, question prompt/info hierarchy
  is readable, and `clientWidth === scrollWidth === 375`. DOM rect/hit checks
  confirmed both sticky actions are visible and clickable (`Continue` was hit
  at viewport y=800/820); the IAB image file itself is shorter than the CSS
  viewport because the browser chrome is excluded from its bitmap.
- Selection dialog Escape closed the modal and returned focus to `Review
  selection` after the inert background was released. Inline error submission
  focused `planning-intake-plannedDate`. IAB console errors/warnings were
  empty, and no framework overlay appeared.
- IAB full flow completed: Planning → New inspection → scope → draft → Purpose
  → Schedule → suggested checklist → Review selection → Confirm → zero Budget
  → Review → Submit to Finance → selected queue record.

Chrome + Computer Use browser-control evidence is `verified locally` for the
revised acceptance boundary:

- Chrome's real browser zoom menu reached `200%`; the Chrome toolbar reported
  `Yakınlaştır: %200` while the public AviaSurveil shell remained rendered.
- Computer Use native `Tab` moved focus from the page to `Skip to sign in`, then
  to the organization sign-in button. Native `Return` on `Skip to sign in`
  moved focus to the sign-in region.
- Chrome DevTools Rendering selected
  `prefers-reduced-motion: reduce` and exposed the active emulation in the
  Rendering panel.
- The public unauthenticated shell's Chrome console also recorded the expected
  unauthenticated `/auth/session` `401` and the current public telemetry
  `/otel/v1/{traces,metrics}` `405` responses; Chrome control evidence does not
  claim a clean public-console run.

The IAB-native zoom, reduced-motion, and full-Tab controls are no longer
required by this revised acceptance boundary. The existing IAB viewport flow
and DOM/overflow evidence remain supplementary `verified locally` evidence.

Final IAB screenshot evidence used in the same `view_image` fidelity pass:

- `/tmp/avia-surveil-final-basics-1440.png`
- `/tmp/avia-surveil-final-checklist-1440.png`
- `/tmp/avia-surveil-final-filters-1024.png`
- `/tmp/avia-surveil-final-tablet-768.png`
- `/tmp/avia-surveil-final-setup-390.png`
- `/tmp/avia-surveil-final-checklist-390.png`
- `/tmp/avia-surveil-final-error-1024.png`
- `/tmp/avia-surveil-final-review-1440.png`
- `/tmp/avia-surveil-final-queue-1440.png`

Final mismatch ledger:

| Area | Final observation | Status |
|---|---|---|
| First viewport and progress | Existing Avia shell; one compact five-step model; one first-step Continue | `verified locally` |
| Inspection brief | Product-specific scope, notice, schedule, checklist, budget, and save state; desktop rail/mobile disclosure | `verified locally` |
| Checklist hierarchy | Prompt leads; two-line clamp and third metadata line; technical identity is not primary | `verified locally` |
| Filters and selection | Eight advanced filters are present and collapsed initially; one Review selection decision retains bounded batches | `verified locally` |
| Inline errors and review | Errors sit below fields/focus first invalid; review has Edit and no Preview | `verified locally` |
| Responsive action access | Sticky actions have DOM rects/hit targets in all tested sizes; no document overflow | `verified locally` |
| Console/overlay/privacy | No IAB error/warn logs or framework overlay; no Audit creation or Inspector start after Finance submission | `verified locally` |
| IAB viewport DOM/overflow/console | 1440×900, 1024×768, 768×1024, and 390×844 each had one main, correct Department Planning identity, no horizontal overflow, and no IAB error/warn logs | `verified locally` |
| Chrome + Computer Use browser controls | Real Chrome `200%` zoom, native Tab/Return focus and activation, and DevTools `prefers-reduced-motion: reduce` emulation were exercised; public console caveat recorded above | `verified locally` |

The plan remains `active`; Chrome + Computer Use browser-control evidence,
public demo smoke/no-op evidence, public all-role qualification, and root
`make check` are `verified locally`. The qualification timeout was resolved by
keeping browser pagination bounded, using one server-side selection projection
for the complete 1,310-question audit, and staging selected rows through the
real search/review flow. A stale Question dossier response was also prevented
from reopening a closed modal. IAB-native controls remain outside the revised
acceptance boundary; production readiness is not claimed. Stakeholder
acceptance remains before a lifecycle transition to `ready-for-verification`.

## Verification matrix

| Surface | Command or method | Expected observation |
|---|---|---|
| Type safety | `npm --prefix apps/web run typecheck` | Exit 0 with no new type error. |
| Focused wizard behavior | `npm --prefix apps/web test -- --run src/features/planning/new-audit-wizard.test.tsx` | Draft creation, autosave, validation, recommendation default, batching, restart, privacy, zero budget, and responsive tests pass. |
| Full React regression | `npm --prefix apps/web test` | Full Vitest suite passes without weakening unrelated assertions. |
| Demo build | `npm --prefix apps/web run build:demo` | Demo artifact builds with no missing style/component asset. |
| HTTP build | `npm --prefix apps/web run build:http` | Connected artifact builds and contains no mock leakage. |
| Mock E2E | `npm --prefix apps/web run test:e2e:mock -- tests/e2e/planning-intake-ux.spec.ts` | The complete user-intent flow passes with exact state changes. |
| Visible actions/accessibility | `npm --prefix apps/web run test:e2e:visible-actions` and `npm --prefix apps/web run test:e2e:accessibility` | Changed route has no inert visible control or reported critical accessibility violation. |
| Connected qualification | Run the repository-supported local qualification profile and focused `qualification-all-role.spec.ts` path | Real HTTP scope, draft, catalog, save, selection, submit, and privacy boundaries pass. Label unavailable infrastructure `blocked`, never passing. |
| Chrome + Computer Use visual QA | Chrome + Computer Use at the agreed desktop/tablet/phone sizes, with real browser zoom, DevTools reduced-motion emulation, and native Tab/Return | Accepted concept fidelity, full target flow, console, focus, modal, overflow, loading, error, sticky-action, zoom, reduced-motion, and keyboard behavior pass. |
| App boundary | `node tests/demo-boundary-smoke.test.js` when demo boundary copy or artifact behavior changes | No production or regulatory claim drift. |
| Repo hygiene | `git diff --check` | Exit 0; no whitespace damage. |
| Harness docs | `make harness-check` when plan/evidence changes are finalized | Plan and index resolve with one matching active row. |

If Chrome Computer Use is unavailable, record `blocked` for the browser-control
acceptance gate. IAB remains supplementary viewport evidence. Automated
Playwright remains useful regression evidence, but it does not replace native
Chrome Computer Use for the browser-level zoom and keyboard checks.

## Acceptance criteria

The plan may move to `ready-for-verification` only when all of the following are
observed:

1. Step one has exactly one primary action and no visible `Open audit setup`
   or `Save draft` button.
2. One valid `Continue` action creates the authoritative draft once and reaches
   step two with a visible saved state.
3. Later-step direct routes without a draft preserve their exact URL while
   presenting the safe Basics entry state; route, heading, progress, form, and
   actions never disagree.
4. One five-step progress model is used consistently. No macro-stage/substep
   duplication or narrow wrapped progress prose remains.
5. Autosave is serialized, revision-safe, visible, retryable, and does not lose
   values across Back/Next/reload/runtime restart.
6. Required fields are marked; errors render inline, are announced, and focus
   the first invalid control.
7. The Inspection brief remains accurate through scope, purpose, notice,
   schedule, checklist, budget, and save-state changes.
8. The default checklist result is the recommended subset for the selected
   scope/type; the UI does not claim all 1,310 are the default result.
9. Advanced filters are discoverable but collapsed initially. Technical IDs are
   not primary row content.
10. The user confirms one understandable selection review; bounded internal
    batches preserve exact digest/receipt semantics and show truthful progress.
11. Blank budget is rejected inline; zero budget is accepted and submitted to
    Finance Review.
12. Review has section-level Edit routes, no redundant Preview toggle, one
    submit action, and a clear post-submit queue handoff.
13. Routine and Ad Hoc notice behavior, Auditee privacy, approval authority,
    Audit non-creation, and Inspector-start separation remain intact.
14. No fake data, package dependency, external font, gradient decoration,
    generic metric cards, or new design-system framework is added.
15. Focused/full tests and builds pass; connected qualification passes or is
    explicitly `blocked` with no false claim.
16. In-app Browser desktop/tablet/phone, keyboard, dossier, error, loading,
    zoom, reduced-motion, console, and overflow checks pass.
17. The accepted concept and final Browser screenshots have a completed
    fidelity ledger with no fixable agency-review comment remaining.
18. The final self-critique records at least one removed nonessential element.

## Risks, dependencies, idempotence, and recovery

### Risks

- Hiding protocol steps could accidentally weaken explicit selection consent.
  Mitigation: retain one clear review/confirm decision and exact server receipts
  while orchestrating batches behind truthful progress.
- Background autosave can race navigation or scope replacement. Mitigation:
  serialize writes, coalesce dirty state, flush before navigation, and bind
  every save to the returned revision.
- A polished rail can become generic dashboard furniture. Mitigation: every
  brief field must be directly derived from the inspection plan and removable
  if it does not help the current decision.
- Step refactoring can break direct links or browser restoration. Mitigation:
  add explicit direct-route, reload, back/forward, and persistent-runtime tests
  before visual changes.
- Shared queue labels overlap with Demo Record Presentation work. Mitigation:
  classify the dirty tree at Gate 0 and reuse that plan's helper rather than
  replacing it.
- Public demo data may continue to look repetitive. Mitigation: keep data
  literal and track seed/data diversity separately; do not solve it with UI
  fabrication.

### Dependencies

- Current authorized scope/catalog endpoints and planning-intake commands.
- The active Demo Record Presentation changes for human-readable queue labels.
- In-app Browser availability for final acceptance.
- Image generation for the complete concept suite and explicit user approval
  before source implementation.
- A connected local qualification runtime for the HTTP boundary gate.

### Idempotence

- Concept generation and screenshot comparison are read-only and repeatable.
- Step-one draft creation uses one generated operation/idempotency key per user
  intent and disables repeat activation while pending.
- Autosave retries reuse the same semantic payload key where required and never
  invent a revision.
- Selection retries resume from the latest committed receipt/digest and never
  replay completed batches as new intent.
- Tests use disposable mock/local state and must not mutate public demo data.

### Recovery

- Before implementation, capture the exact diff of every overlapping file.
- Keep changes scoped to planning intake components/styles/tests. Do not use
  `git reset`, broad checkout, recursive clean, or branch operations.
- If a task fails, revert only that task's known edits through a reviewed patch
  or continue from the last passing component boundary.
- Autosave failures retain local values and expose Retry; they do not navigate
  or clear the draft.
- Selection-chain failures retain completed receipts and pending intent; they
  do not claim full confirmation or discard the user's selection.
- No deployment rollback is part of this plan because deployment is excluded.

## Decisions and discoveries

### 2026-08-17 — Expose user intent, not server command order

Decision: remove the visible draft-opening and manual-save decisions. The
existing server still owns creation and persistence; `Continue` and autosave
are the human interaction model.

### 2026-08-17 — Use one five-step model

Decision: remove the simultaneous three-stage/five-substep hierarchy. Five
steps already match the route and field ownership and are easier to scan,
validate, deep-link, and test.

### 2026-08-17 — Use a governance editor, not a dashboard refresh

Decision: the creative classification is `governance editor`. The Inspection
brief is the single bold/product-specific element. The rest is restrained,
open, and token-driven.

### 2026-08-17 — Reject generic redesign shortcuts

The initial direction included a conventional premium card layout and a generic
right sidebar. `ui-self-critique` found that this could belong to any SaaS
onboarding product. The plan was revised to:

- remove nested decorative cards;
- tie the right rail exclusively to inspection-plan facts and save state;
- retain a table-first queue;
- use no fake KPIs, gradient CTA, illustration, or marketing hero;
- make selection-review progress the only complex state treatment.

### 2026-08-17 — Reuse the existing type and token system

`ui-ux-pro-max` recommended a trust-and-authority palette and Lexend/Source
Sans. The palette is already close to Avia's semantic tokens, but new external
fonts would create CSP, artifact, dependency, and cross-screen consistency
risk. The plan therefore keeps the existing workbench font stack and improves
hierarchy through existing tokens, scale, measure, weight, and spacing.

### 2026-08-17 — In-app Browser discovery race diagnosed and recovered

The first exact `get("iab")` call reported that the in-app Browser was
unavailable. Following the Browser skill's bootstrap troubleshooting path
without resetting the persistent runtime showed one current `iab` backend bound
to this exact Codex session. Retrying the same exact selector then succeeded.
The evidence indicates a transient backend-registration/discovery race rather
than a missing plugin or broken local application.

The recovered IAB opened
`http://127.0.0.1:5173/department-manager/new-audit/step-1`; URL and title were
correct, the DOM contained `New Inspection` and `Supplier / organization`, no
framework overlay was present, and console error/warn logs were empty. IAB
connectivity and the basic page identity are `verified locally`. The complete
multi-step baseline screenshot/interaction matrix remains `not run` and stays
part of Gate 0.

### 2026-08-18 — Remove implementation status from Manager chrome and fix mobile stepper

The Manager shell must not expose `manager`, `Release-bound authenticated
session`, or environment-label plumbing as page content. Those values belong
to diagnostics, not the user task. The five-step mobile indicator now switches
to readable stacked rows below 760px; it does not squeeze all labels into five
equal-width columns.

## Current progress

- [x] Read repository, planning, verification, and output authorities.
- [x] Read the user-named `ui-ux-pro-max` and `ui-self-critique` skills.
- [x] Run `ui-ux-pro-max` design-system, form, feedback, accessibility, and
  style searches.
- [x] Inspect the eight supplied screenshots and current React/CSS/test/API
  surfaces.
- [x] Classify the surface as a governance editor and define the Inspection
  brief signature element.
- [x] Complete the pre-implementation self-critique and revise generic choices.
- [x] Create this self-contained implementation and verification plan.
- [x] Diagnose the initial IAB discovery failure and verify the recovered IAB
  against local step one with clean console output.
- [x] Gate 0 dirty-tree baseline and current focused test evidence (`verified locally`; full catalog/dossier baseline `not run` with the recorded local-demo/IAB limitation).
- [x] Gate 1 complete concept suite and user approval.
- [x] Tasks 1–7 implementation and primary local verification (`verified locally`).
- [x] Final self-critique and fidelity ledger recorded; nonessential brief and technical-detail copy removed.
- [x] Complete Chrome + Computer Use native Tab/Return, real 200% zoom, and reduced-motion emulation subchecks (`verified locally`).
- [x] Remove Manager runtime/environment text and correct the 390px stepper layout; focused UI tests 39/39, desktop 1440x900 and mobile 390x844 IAB screenshots, DOM, interaction, and console checks are `verified locally`.
- [x] Public all-role qualification and root `make check` blockers resolved with literal local evidence.
- [x] Stakeholder acceptance and plan lifecycle transition to `ready-for-verification` recorded 2026-08-18.

### 2026-08-18 — Stakeholder acceptance

The user explicitly approved the completed implementation and evidence in this
conversation. The plan therefore transitions to `ready-for-verification`. This
acceptance covers the implemented candidate/public-demo scope and does not claim
production readiness.

## Current outcome

Planning and Gate 1 are complete and accepted. Tasks 1–7 are implemented with
frontend-only planning-intake changes, focused/full React evidence, builds,
contract and mock-flow evidence, public connected qualification, visible-
actions/accessibility evidence, primary IAB viewport evidence, and Chrome +
Computer Use browser-control evidence all `verified locally`. Public release
lock/app-shell digest matching, exact release-check, health, tunnel, apply,
qualification, root `make check`, and direct no-op evidence are
`verified locally`. IAB-native control coverage is outside the revised
acceptance boundary; production readiness is not claimed.

The plan is now ready for independent verification. The exhaustive catalog
audit, root release-lock/attestation fixture blockers, public release, connected
all-role flow, and stakeholder acceptance are resolved with the evidence above.
Production readiness is not claimed.

## Execution Prompt

Use the following prompt to execute this plan in a new Codex task:

```text
Implement the active AviaSurveil ExecPlan:

apps/surveil/docs/exec-plans/active/2026-08-17-department-manager-planning-intake-ux-redesign-plan.md

Work from /Users/marlonjd/Developer/monorepos/avia on the current branches.
Read the root and apps/surveil AGENTS.md files, apps/surveil/docs/PLANS.md, the
plan, verification matrix, output contract, and the active Demo Record
Presentation plan completely before changing source. Preserve every unrelated
dirty-worktree change; do not reset, clean, switch/create branches, commit,
push, deploy, or modify public infrastructure unless I explicitly authorize
that exact action later.

Use the ui-ux-pro-max, ui-self-critique, frontend-app-builder, and
frontend-testing-debugging skills faithfully. Start at Gate 0. Capture the
current diff and focused test baseline. Then generate the complete Gate 1
concept suite for desktop step one, desktop checklist selection, desktop
review, desktop inline-error state, phone setup, and phone checklist selection.
Keep the existing Avia shell and semantic token palette. Treat the surface as a
Department Manager governance editor, not a dashboard or marketing page. The
signature element is the product-specific Inspection brief. Use no decorative
hero, generic SaaS card grid, gradient CTA, fake KPI, external font, new UI
library, or invented data. Run ui-self-critique on the concept, remove at least
one nonessential element, show me the complete concept suite, and pause for my
approval before coding.

After approval, implement Tasks 1 through 7 in order. The visible interaction
must remove `Open audit setup for this supplier`, remove manual `Save draft`,
use one five-step progress model, create the server-owned draft through the
first valid Continue action, autosave with serialized revision-safe state,
show inline accessible validation, use a responsive Inspection brief, make the
question catalog recommendation-first with advanced filters collapsed, and
replace repeated protocol batch buttons with one understandable selection
review/confirmation while preserving every bounded digest-chained receipt.
Keep zero budget valid, withheld-notice privacy intact, Audit creation absent at
Planning submission, and all authority boundaries exact.

Refactor the monolithic wizard only where component/state ownership becomes
clearer. Reuse the Demo Record Presentation helper for human labels. Do not
fabricate queue data or alter backend/OpenAPI unless source inspection proves a
hard contract gap; stop and report that gap before any backend edit.

Verify continuously with focused tests, then run the exact plan verification
matrix. Use Chrome + Computer Use as the mandatory browser-level control gate
at 1440x900, 1024x768, 768x1024, and 390x844. IAB may provide supplementary
viewport evidence. Exercise the complete Planning → New inspection → scope →
draft → purpose → schedule → suggested checklist → review/confirm selection →
budget → review → Finance submission → selected queue flow. Check console,
overlays, focus, native keyboard, Escape/return focus, inline errors, loading,
autosave failure/retry, overflow, real 200% browser zoom, reduced-motion
emulation, and sticky actions. Use view_image on accepted concepts and final
Chrome screenshots in the same fidelity pass; maintain a mismatch ledger and
keep fixing until no agency-review comment remains.

After implementation, run the final ui-self-critique, remove one remaining
nonessential visual/copy element, update the plan's progress, decisions,
evidence, current outcome, and exactly one matching index row. Use literal
evidence labels: verified locally, not run, blocked, candidate-only, release
pending, and production readiness not claimed. Clean up task-owned browser and
dev-server processes before handoff.
```
