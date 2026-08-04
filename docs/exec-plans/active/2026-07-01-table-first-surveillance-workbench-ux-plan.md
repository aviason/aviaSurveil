# Table-First Surveillance Workbench UX Implementation Plan


**Goal:** Redesign the frontend-only AviaSurveil360 demo around a table-first, action-first Surveillance Workbench so users see the work item, owner, next action, due date, status, and risk before secondary details.

**Architecture:** Preserve the static HTML, CSS, and Vanilla JavaScript demo architecture. Add a small shared work-item projection layer so audits, findings, CAPs, evidence, planning approvals, reports, templates, organizations, users, and audit logs can render through consistent table/list patterns without changing mock workflow semantics.

**Tech Stack:** HTML, CSS, Vanilla JavaScript, mock data, client-side state, existing Node smoke tests, screenshot evidence under `qa/screenshots/`.

**Implementation note (2026-07-01):** Implemented locally and marked
`ready-for-verification` in `docs/exec-plans/index.md`. The frontend-only
architecture and demo boundaries are preserved. Local verification covered
syntax checks, deterministic Node smoke tests, in-app Browser row-click,
auditee-privacy, CAP/evidence lifecycle, OHI guardrail, desktop/mobile viewport
checks, and representative changed-route screenshots under
`qa/screenshots/table-first-2026-07-01/`.

**Implementation note (2026-07-02, deeper pass):** A second, deeper
simplification pass was executed against this plan's objective using the
2026-07-02 Playwright QA set as evidence. It removed the Inspector Workbench
hero card, the Audit Work Queue attention strip (counts moved into filter
chips), the Checklist Runner progress card (now a one-line progress band with
the active question panel first on mobile), the redundant shared-table
`Lifecycle` column, a dead Internal CAA Notes render block in `viewFinding()`,
and the boxed OHI callout (now a one-line guardrail note with unchanged
wording). The Organization Risk Profile was restructured into a single risk
header band with full-width Findings/Audit History tables, fixing the only
desktop 1920px overflow. Shared work-queue tables now stack into
priority-first rows below 640px while keeping row-click routes. Verified
locally: all `node --check` passes, all 17 Node smoke tests, full CAP/evidence
lifecycle click-through (CAP accepted != closure; closure only on evidence
acceptance), auditee privacy render checks, and a fresh 140-capture
Playwright set under `qa/screenshots/playwright-2026-07-02/` with 0 errors,
0 console issues, and 0 horizontal overflow on both viewports. Build summary
docs updated in English and Turkish.

---

## Context

The current demo already carries the right product logic: role-based navigation, audit queueing, checklist execution, Potential Finding review, Finding -> CAP -> Evidence -> closure, auditee visibility boundaries, planning approvals, reports, risk screens, and admin previews.

The main UX issue found from the 78 full-page desktop screenshots is not missing data. It is that the same operational information is spread across many card stacks, KPI blocks, separate panels, and long vertical detail pages. Users repeatedly have to re-derive:

- What is the work item?
- Who owns it now?
- What is the next action?
- What is due, overdue, or blocked?
- What happens if I click?

The target direction is a restrained, civil-aviation-authority-appropriate **Surveillance Workbench**:

- top-level work surfaces are primarily tables or table-like rows;
- rows are clickable and open an existing detail page or modal;
- nested relationships are visible through indentation, compact lifecycle chips, and optional expandable row detail;
- cards are reserved for detail pages, modal bodies, and genuinely framed evidence or report previews;
- dashboards become attention queues, not metric walls.

## Evidence Source

Screenshot index: `qa/screenshots/README.md`

Captured evidence summary:

- 78 full-page desktop screenshots.
- Main workflow captured: Potential Finding -> Lead conversion -> Finding -> CAP -> Evidence -> Closure.
- Desktop viewport: 1280x720 full-page captures.
- Demo boundary: frontend-only, client-side mock state.

Key screenshot observations:

- `01-inspector-todays-workbench.jpg`: useful role question and next-action cards, but the user still scans four card zones instead of one prioritized work queue.
- `02-inspector-audit-work-queue-active.jpg` and `32-manager-audit-work-queue.jpg`: closest to the desired direction; rows already show next action, owner, due date, and status.
- `05` through `08` checklist screenshots: long stacked checklist cards make the current task hard to keep in view; the active question should behave like an inspector checklist table with one active detail row.
- `09`, `10`, `55` lead inspector screens: too many cards before the decision; the important work is "review pending finding/report decision".
- `11`, `13`, `15`, `17`, `19`, `20`, `23`, `25`, `27` finding detail states: strong lifecycle logic, but CAP, evidence, comments, internal notes, and audit trail should read as a dossier opened from a table row.
- `28`, `43`, `45`, `47`, `48` manager dashboards: useful signals, but should be normalized into management attention queues.
- `29`, `30`, `31`, `57`, `61`, `62` planning screens: role-specific approval is correct; surface should become one approval work table plus a selected planning dossier.
- `38`, `39`, `40`, `41`, `68`, `69`, `70`, `71`, `73`, `75`, `76`, `77`, `78` admin/config screens: table-first already fits, but preview/builder pages still use stacked cards where editable/preview rows would be clearer.

## UX Direction

### Product Signature Element

Use a shared **Work Item Row** as the signature pattern.

Each row should answer:

| Field | Purpose |
|---|---|
| Priority | Critical, overdue, waiting review, due soon, normal. |
| Item | Audit, Finding, CAP, Evidence, Report, Planning Item, Template, User, Organization, Audit Log entry. |
| Organization | Visible where permitted. Hidden from auditee when outside its own organization. |
| Lifecycle | Compact status path, such as `Finding -> CAP -> Evidence -> Closure`. |
| Owner | Current owner, role-aware. |
| Next Action | The primary action the user can or should take. |
| Due Date / Target | Due Date, Target, overdue/due soon label. |
| Status | Plain lifecycle status. |
| Open | Row click opens detail page; action button performs the next action. |

The row is not just a data table row. It is a compact command surface.

### Table Patterns

| Pattern | Use |
|---|---|
| `ops-table` | Dense work queues: audits, findings, reports, planning approvals, admin lists. |
| `ops-row--parent` | Parent item such as Audit or Finding. |
| `ops-row--child` | Nested CAP, Evidence, Report, or checklist item below a parent. |
| `ops-row--attention` | Critical, overdue, or waiting-review rows. |
| `dossier-page` | Existing detail route opened from a row; includes next action at the top and secondary tabs/sections below. |
| `attention-strip` | Compact counts above a table, replacing multiple KPI cards where possible. |
| `active-row-panel` | Checklist runner or builder: table on the left/top, active item details below or beside it. |

### `ui-self-critique` Rules To Apply During Execution

Before implementing each major screen group:

1. Name the role, surface, and single job.
2. Classify the surface as `queue`, `dossier`, `governance`, `editor`, `reconciliation`, `communication`, or `security`.
3. Define one product-specific detail that makes the surface AviaSurveil360-specific.
4. Remove at least one generic dashboard/card element when it does not serve the screen's job.

After implementing each major screen group:

1. Inspect the rendered screen from the screenshot path or browser.
2. Remove one visual element or copy line that does not serve the next action.
3. Record what was verified locally.

Creative Production is not the primary path for this execution plan. Use it only if stakeholders later ask for visual territories or brand/asset exploration. This plan is an operational UX simplification plan, not a moodboard or campaign asset task.

## Scope

### In Scope

- Convert primary work surfaces from card-first to table-first where it improves task clarity.
- Keep row click navigation to existing detail pages.
- Add optional nested rows where parent-child relationships matter: Audit -> Finding -> CAP -> Evidence, Planning Item -> Approval Step -> Preparation Step, Report -> Approval Step.
- Preserve all mock workflow state transitions.
- Preserve demo-only labels and production boundary warnings.
- Preserve auditee visibility rules.
- Reduce KPI cards to compact attention strips when a table is more useful.
- Keep existing static frontend architecture.
- Add focused smoke tests for the shared work-item projection and table-first render expectations.

### Out Of Scope

- No backend, database, API, real authentication, real authorization enforcement, real file upload, real email/SMS/WhatsApp notification, real document storage, real AI service, real regulatory ingestion, production audit-log system, mobile/offline production app, framework migration, or advanced BI/report builder.
- No automatic enforcement, certificate, legal, or closure decision.
- No change to CAP acceptance semantics: CAP accepted is not finding closure.
- No branch, commit, push, or PR unless explicitly requested.
- No broad rebrand, logo exploration, generated hero imagery, or marketing creative.

## Assumptions

- `qa/screenshots/README.md` is the current screenshot evidence set for this plan.
- `DEMO_TODAY` remains `2026-06-15` for deterministic due/overdue behavior.
- `js/helpers.js` remains the source for status, owner, due-date, primary-action, and visibility helpers.
- `js/views.js` can be improved incrementally without a framework migration.
- New shared helpers may be added as a plain browser script if that keeps `js/views.js` readable.
- Current dirty working-tree changes in `.gitignore`, `css/styles.css`, `js/views.js`, and `tests/lead-inspector-workspace-smoke.test.js` may be user work; implementation must inspect and preserve them before editing.

## Dependencies

- Existing mock data in `js/data.js`.
- Existing status and action helpers in `js/helpers.js`.
- Existing role routing and navigation in `js/app.js`.
- Existing render functions in `js/views.js`.
- Existing styles in `css/styles.css`.
- Existing static script order in `index.html`.
- Existing smoke-test style under `tests/`.
- Screenshot evidence under `qa/screenshots/`.

## Ownership Boundaries

- Product/UX owner decides whether every dashboard card is removed or whether a small attention strip remains.
- Engineering owner preserves workflow semantics, mock state, tests, and demo boundary.
- Regulatory/domain owner reviews terminology if visible copy changes regulatory meaning.
- Stakeholders validate whether the table-first workbench is easier to explain in a demo.

## Files To Modify During Execution

Expected files:

- `index.html`
  - Add `js/work-items.js` to the script order if a shared projection file is created.

- `css/styles.css`
  - Add table-first components: `attention-strip`, `ops-table`, `ops-row`, nested row states, row action affordances, active-row panel, and dossier tabs/section styling.
  - Reduce card-heavy dashboard layouts where table-first patterns replace them.

- `js/work-items.js` (new)
  - Provide shared work-item projection functions.
  - Keep this file data-shaping only; do not own state transitions.

- `js/helpers.js`
  - Add small helper wrappers only if the projection layer needs stable labels not already present.
  - Preserve current visibility and action helpers.

- `js/views.js`
  - Replace repeated `list__row` card-like lists with reusable table rendering.
  - Convert priority work surfaces to table-first layouts.
  - Keep existing routes and modals.

- `js/app.js`
  - Adjust navigation labels only if needed to align with the workbench IA.
  - Preserve role-home route behavior and row-click navigation.

- `tests/table-first-workbench-smoke.test.js` (new)
  - Verify the shared projection layer and key render outputs.

- Existing tests under `tests/`
  - Update only when render text or route structure intentionally changes.

- `docs/demo-evidence/BUILD_SUMMARY.md`
  - Update after execution to describe the table-first UX change and verification.

- `docs/demo-evidence/BUILD_SUMMARY.turkce.md`
  - Turkish companion update after execution if the English build summary changes.

- `MANIFEST.md`
  - Update if `js/work-items.js` or a new test is added.

## Work-Item Projection Interface

If `js/work-items.js` is created, use this concrete shape:

```js
function workItemFromFinding(finding, options) {
  return {
    id: finding.id,
    type: 'Finding',
    title: finding.title,
    organization: orgName(finding.orgId),
    priority: workItemPriorityForFinding(finding),
    lifecycle: statusMeta(finding).label,
    owner: ownerLabel(finding),
    nextAction: nextActionLabel(finding),
    dueText: workItemDueTextForFinding(finding),
    statusHtml: statusBadge(finding),
    primaryAction: primaryActionFor(finding),
    route: { view: 'finding', id: finding.id },
    children: workItemChildrenForFinding(finding, options || {})
  };
}
```

Use equivalent functions for audits, planning items, reports, templates, organizations, users, and audit log entries. Do not duplicate workflow transition logic in this file.

## Screen-By-Screen Target Treatment

| # | Screenshot | Current Surface | Target Treatment |
|---:|---|---|---|
| 1 | `00-login-role-select.jpg` | Role cards | Keep cards; this is not an operations table. Simplify role copy if needed. |
| 2 | `01-inspector-todays-workbench.jpg` | Four workbench card zones | Convert to one prioritized `My Work Today` table with an attention strip above it. |
| 3 | `02-inspector-audit-work-queue-active.jpg` | Audit queue rows | Upgrade to `ops-table`; preserve active/completed filter and row click. |
| 4 | `03-inspector-audit-work-queue-completed.jpg` | Completed audit queue | Same `ops-table`; completed rows use closed status and report action. |
| 5 | `04-inspector-audit-detail-aud-2026-001.jpg` | Audit dossier cards | Keep as dossier, but make related findings/checklist a compact table below the next-action band. |
| 6 | `05-inspector-checklist-runner-empty.jpg` | Long checklist cards | Convert to checklist table with one active row detail panel. |
| 7 | `06-inspector-checklist-non-compliant-comment.jpg` | Flagged checklist card | Active row panel shows required comment, mock evidence filename, and Potential Finding action. |
| 8 | `07-inspector-checklist-mock-evidence-selected.jpg` | Flagged checklist card with file | Same active row panel; file chip stays secondary. |
| 9 | `08-inspector-potential-finding-created.jpg` | Checklist after Potential Finding | Active row shows Potential Finding child row and link to lead review. |
| 10 | `09-lead-inspector-review-pending-potential.jpg` | Lead workspace cards and long sections | Convert top to `Lead Review Queue` table; pending potential finding is the first actionable row. |
| 11 | `10-lead-inspector-potential-finding-decision-ready.jpg` | Lead decision area | Keep decision form, but open it from selected queue row/dossier. |
| 12 | `11-lead-inspector-finding-detail-created.jpg` | Finding detail dossier | Convert secondary cards into dossier sections/tabs; top next action remains dominant. |
| 13 | `12-auditee-my-findings-with-new-finding.jpg` | KPI cards plus list | Replace KPI cards with compact attention strip; list becomes `My CAA Requests` table. |
| 14 | `13-auditee-finding-detail-waiting-cap.jpg` | Auditee finding dossier | Keep dossier; top should say what the CAA needs and by when. CAP section should be the active panel. |
| 15 | `14-auditee-cap-submission-modal-complete.jpg` | CAP modal | Keep modal; improve as step-like form only if needed. No table requirement. |
| 16 | `15-auditee-finding-detail-cap-submitted.jpg` | Auditee finding dossier | Show row state `Waiting CAA Review`; CAP details secondary. |
| 17 | `16-inspector-cap-review-queue.jpg` | Findings filtered list | Upgrade to `CAP Review Queue` table with CAP child summary and review action. |
| 18 | `17-inspector-finding-detail-cap-submitted.jpg` | Inspector finding dossier | Keep top action `Review CAP`; collapse non-active sections behind tabs/details. |
| 19 | `18-inspector-cap-review-modal.jpg` | CAP review modal | Keep modal; show auditee CAP fields in a compact review table/definition grid. |
| 20 | `19-inspector-finding-detail-cap-accepted-evidence-required.jpg` | Finding dossier | Row/lifecycle state should make `Evidence Required` visible; CAP accepted is not closure. |
| 21 | `20-auditee-finding-detail-evidence-required.jpg` | Auditee finding dossier | Evidence upload becomes active panel; CAP details secondary. |
| 22 | `21-auditee-evidence-upload-modal-empty.jpg` | Evidence modal | Keep modal; make accepted evidence expectation visible before file control. |
| 23 | `22-auditee-evidence-upload-modal-selected.jpg` | Evidence modal with file | Keep modal; selected filename remains mock-only. |
| 24 | `23-auditee-finding-detail-evidence-submitted.jpg` | Auditee finding dossier | Show `Waiting CAA Review` status in top band and table row. |
| 25 | `24-inspector-evidence-review-queue.jpg` | Findings filtered list | Upgrade to `Evidence Review Queue` table with latest evidence child row. |
| 26 | `25-inspector-finding-detail-evidence-submitted.jpg` | Inspector finding dossier | Keep top action `Review evidence`; evidence becomes active section. |
| 27 | `26-inspector-evidence-review-modal.jpg` | Evidence review modal | Keep modal; decision, auditee comment, internal note separation must remain clear. |
| 28 | `27-inspector-finding-detail-closed.jpg` | Closed finding dossier | Show closed state and report action; history/evidence secondary. |
| 29 | `28-manager-dashboard-updated.jpg` | Manager dashboard cards | Convert to `Management Attention` table plus compact OHI/summary strip. |
| 30 | `29-manager-planning-overview.jpg` | Planning dossier cards | Convert to `Planning Workbench` table plus selected planning item dossier. |
| 31 | `30-manager-planning-approval-tab.jpg` | Planning approval tab | Same planning table; selected row shows approval decision panel. |
| 32 | `31-manager-planning-preparation-tab.jpg` | Planning preparation tab | Same planning table; selected row shows preparation next step. |
| 33 | `32-manager-audit-work-queue.jpg` | Audit queue rows | Upgrade to shared `ops-table`; manager-specific owner/action columns. |
| 34 | `33-manager-new-audit-wizard-step-1.jpg` | Wizard | Keep wizard; not table-first. Make route launched from work table. |
| 35 | `34-manager-new-audit-wizard-step-2.jpg` | Wizard | Keep wizard. |
| 36 | `35-manager-new-audit-wizard-step-3.jpg` | Wizard | Keep wizard. |
| 37 | `36-manager-new-audit-wizard-step-4.jpg` | Wizard | Keep wizard. |
| 38 | `37-manager-new-audit-wizard-step-5.jpg` | Wizard review | Keep wizard; review summary can use compact definition table. |
| 39 | `38-manager-checklist-approvals.jpg` | Approval dossier cards | Convert to checklist approval queue table plus selected approval dossier. |
| 40 | `39-manager-question-bank.jpg` | Form plus list | Keep add form compact; render questions as table rows with references and expected evidence. |
| 41 | `40-manager-checklist-builder.jpg` | Builder cards | Convert current template questions and question bank into two table panes or table + active detail. |
| 42 | `41-manager-checklist-versions.jpg` | Version list | Render as version table; row opens approval/history detail. |
| 43 | `42-manager-audit-reports.jpg` | Report approval dossier | Convert to report approval queue plus selected report dossier. |
| 44 | `43-manager-safety-intelligence.jpg` | Risk dashboard cards | Convert to management signal table; recommended action becomes selected-row side panel. |
| 45 | `44-manager-org-risk-profile.jpg` | Organization risk dossier | Keep dossier; findings/audit history become tables, context becomes right-side facts. |
| 46 | `45-manager-ssp-nasp.jpg` | Objective cards | Convert SPI/NASP actions into tables under objective rows. |
| 47 | `46-manager-open-findings.jpg` | Findings list | Upgrade to `ops-table` with Finding -> CAP/Evidence nested rows where relevant. |
| 48 | `47-manager-cap-effectiveness.jpg` | CAP effectiveness cards | Convert to effectiveness review table by finding/org, with recurrence and verification columns. |
| 49 | `48-manager-usoap-readiness.jpg` | Readiness cards | Convert PQ readiness into evidence-gap table. |
| 50 | `49-manager-organizations.jpg` | Organization list | Render as organizations table with risk, open findings, next audit, and open dossier action. |
| 51 | `50-manager-organization-detail-org-xyz.jpg` | Organization dossier | Keep dossier; audits/findings/risk signals become tables. |
| 52 | `51-inspector-inspection-evidence.jpg` | Offline/evidence screen | Render outbox/evidence as table; active item shows mock sync status. |
| 53 | `52-inspector-reports.jpg` | Reports list | Render report table with finding, org, closed date, status, action. |
| 54 | `53-inspector-report-preview.jpg` | Report preview | Keep dossier/preview; source findings and approvals become tables. |
| 55 | `54-inspector-ai-assistant.jpg` | AI assistant cards | Keep assistant as review surface, but draft suggestions should be rows requiring authorized review. |
| 56 | `55-lead-inspector-review-queue.jpg` | Lead review cards/sections | Convert to lead work table; selected row opens report/finding decision dossier. |
| 57 | `56-lead-inspector-audit-reports.jpg` | Report approval dossier | Same report approval table pattern as manager, scoped to Lead Inspector. |
| 58 | `57-gm-planning.jpg` | GM planning approval | Same planning table pattern, role-specific action column. |
| 59 | `58-gm-checklist-approvals.jpg` | GM checklist approval | Same checklist approval table pattern. |
| 60 | `59-gm-audit-reports.jpg` | GM report approval | Same report approval table pattern. |
| 61 | `60-gm-reports.jpg` | Reports list | Same report table, GM scope. |
| 62 | `61-finance-planning.jpg` | Finance planning review | Same planning table, finance budget/action columns. |
| 63 | `62-executive-director-planning.jpg` | ED planning approval | Same planning table, ED final approval action. |
| 64 | `63-executive-director-audit-reports.jpg` | ED report approval | Same report approval table, ED final approval action. |
| 65 | `64-executive-director-reports.jpg` | Reports list | Same report table. |
| 66 | `65-auditee-my-findings-after-closure.jpg` | Auditee finding list | Same `My CAA Requests` table; closed rows route to report. |
| 67 | `66-auditee-messages.jpg` | Message list | Render communication table with message, related finding, visibility, date. |
| 68 | `67-auditee-reports.jpg` | Shared documents list | Render shared documents table; only auditee-visible records. |
| 69 | `68-admin-templates.jpg` | Template table | Already table-first; align with `ops-table`. |
| 70 | `69-admin-template-preview.jpg` | Long template preview cards | Convert preview questions to table with expandable regulatory trace/details. |
| 71 | `70-admin-regulatory-library.jpg` | Regulatory library preview | Render references as regulatory table; trace details open in dossier/row detail. |
| 72 | `71-admin-question-bank.jpg` | Question bank list | Same question table pattern as manager. |
| 73 | `73-admin-checklist-versions.jpg` | Version history list | Version table with status, owner, reason, action. |
| 74 | `74-admin-reports.jpg` | Reports list | Report table. |
| 75 | `75-admin-users.jpg` | Users table | Already table-first; align styling and row affordance. |
| 76 | `76-admin-settings.jpg` | Settings cards/forms | Configuration categories can be rows; individual settings remain forms. |
| 77 | `77-admin-organizations.jpg` | Organizations table/list | Align with organizations `ops-table`. |
| 78 | `78-admin-audit-log.jpg` | Audit log table/list | Keep table; add actor, action, target, timestamp, system/manual columns. |

## Phases

### Phase 0: Baseline And Visual Audit

- [ ] **Step 1: Reconfirm dirty worktree ownership.**

  Run:

  ```bash
  git status --short
  ```

  Expected: existing modified files are inspected before editing and unrelated user changes are preserved.

- [ ] **Step 2: Re-read screenshot index and evidence.**

  Run:

  ```bash
  sed -n '1,220p' qa/screenshots/README.md
  ```

  Expected: 78 screenshots are listed and the main Finding -> CAP -> Evidence -> Closure workflow is documented.

- [ ] **Step 3: Run baseline smoke tests before visual changes.**

  Run:

  ```bash
  node tests/demo-boundary-smoke.test.js
  node tests/audit-work-queue-smoke.test.js
  node tests/inspection-execution-smoke.test.js
  node tests/checklist-management-smoke.test.js
  node tests/checklist-approval-smoke.test.js
  node tests/report-approval-smoke.test.js
  node tests/planning-workspace-smoke.test.js
  node tests/planning-render-smoke.test.js
  node tests/planning-release-smoke.test.js
  node tests/governance-render-smoke.test.js
  ```

  Expected: existing passing baseline is known before changing UI. If a test is already failing, document it and decide whether it is in scope.

### Phase 1: Add Shared Table-First Primitives

- [ ] **Step 1: Add or refactor CSS primitives.**

  Modify `css/styles.css` to add:

  ```css
  .attention-strip { display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 14px; }
  .attention-pill { display: inline-flex; align-items: center; gap: 6px; border: 1px solid var(--line); background: var(--surface); border-radius: 999px; padding: 7px 11px; font-size: 12.5px; font-weight: 650; }
  .ops-table { width: 100%; border-collapse: collapse; table-layout: fixed; }
  .ops-table th, .ops-table td { text-align: left; padding: 10px 12px; border-bottom: 1px solid var(--line-soft); vertical-align: top; font-size: 13px; }
  .ops-table th { color: var(--muted); background: var(--surface-alt); font-size: 11.5px; text-transform: uppercase; letter-spacing: .04em; }
  .ops-row { cursor: pointer; }
  .ops-row:hover td { background: var(--surface-alt); }
  .ops-row--child td:first-child { padding-left: 34px; }
  .ops-row--attention td { box-shadow: inset 3px 0 0 var(--warn); }
  .ops-row--danger td { box-shadow: inset 3px 0 0 var(--danger); }
  .ops-cell-title { font-weight: 650; }
  .ops-cell-sub { color: var(--muted); font-size: 12px; margin-top: 3px; }
  .ops-actions { display: flex; justify-content: flex-end; gap: 8px; flex-wrap: wrap; }
  .dossier-sections { display: grid; grid-template-columns: minmax(0, 1fr) 320px; gap: 16px; align-items: start; }
  .active-row-panel { border: 1px solid var(--line); border-radius: var(--radius); background: var(--surface); box-shadow: var(--shadow-sm); }
  ```

  Expected: existing `.table`, `.list`, `.card`, `.nextbar`, and `.stepper` styles still work.

- [ ] **Step 2: Add a shared renderer helper inside `js/views.js` or a new `js/work-items.js`.**

  Preferred: create `js/work-items.js`, then add it to `index.html` before `js/views.js`.

  Expected script order:

  ```html
  <script src="js/data.js"></script>
  <script src="js/helpers.js"></script>
  <script src="js/approval.js"></script>
  <script src="js/planning.js"></script>
  <script src="js/checklists.js"></script>
  <script src="js/inspection.js"></script>
  <script src="js/reports.js"></script>
  <script src="js/work-items.js"></script>
  <script src="js/views.js"></script>
  <script src="js/app.js"></script>
  ```

### Phase 2: Convert Core Operational Queues

- [ ] **Step 1: Convert `viewCalendar()` audit queue to `ops-table`.**

  Preserve active/completed filters, manager `+ New Audit`, row click to `audit-detail`, and action buttons.

- [ ] **Step 2: Convert `viewFindings()` and `viewAuditeeMyFindings()` to table-first.**

  Requirements:

  - Internal roles see organization and internal-facing action labels.
  - Auditee sees only its own organization and no internal CAA notes.
  - CAP/evidence review queues show the appropriate child summary.
  - `primaryActionFor(finding)` remains the action source.

- [ ] **Step 3: Convert `viewInspectorDashboard()` into one prioritized table.**

  Replace the four card zones with:

  - attention strip counts: today's inspections, CAP reviews, evidence reviews, high-risk signals;
  - a unified `My Work Today` table sorted by priority then due date;
  - quick actions as a compact toolbar, not a separate card stack.

- [ ] **Step 4: Convert `viewManagerDashboard()` and `viewSafetyIntelligenceDashboard()` into management attention tables.**

  Preserve OHI as a management indicator only. Do not make it an automatic legal, enforcement, certificate, suspension, or closure decision.

### Phase 3: Convert Dossier Pages Without Breaking Workflow

- [ ] **Step 1: Update `viewAuditDetail()`.**

  Keep the next-action band. Convert audit metadata and related findings into compact table sections.

- [ ] **Step 2: Update `viewFinding()`.**

  Keep:

  - `nextActionBar(f)`;
  - lifecycle stepper;
  - CAP accepted is not closure language;
  - auditee/internal note separation;
  - evidence version history;
  - audit log.

  Reduce stacked cards by using dossier sections and tables for CAP, evidence, comments, and audit trail.

- [ ] **Step 3: Update `viewOrgDetail()` and `viewOrganizationRiskProfile()`.**

  Keep the risk dossier, but render findings and audit history as tables.

### Phase 4: Convert Checklist And Template Workflows

- [ ] **Step 1: Refactor `viewChecklistRunner()`.**

  Target:

  - table of checklist questions with answer state, evidence expectation, and finding status;
  - selected/active question panel for answer buttons, comment, mock evidence, Potential Finding creation;
  - progress remains visible above the table.

- [ ] **Step 2: Refactor `viewTemplatePreview()`.**

  Convert the long question cards into a table with expandable trace/details.

- [ ] **Step 3: Refactor `viewQuestionBank()` and `viewChecklistBuilder()`.**

  Use a table or two-pane table layout. The add/draft form remains compact and secondary.

- [ ] **Step 4: Refactor `viewChecklistApprovals()` and `viewChecklistVersions()`.**

  Use approval/version tables with selected-row dossier detail.

### Phase 5: Convert Governance, Report, And Admin Surfaces

- [ ] **Step 1: Refactor planning role screens.**

  `viewPlanningWorkspace()`, `viewPlanningApprovals()`, and compatibility routes must show one planning work table plus selected detail/actions for manager, GM, Finance, and Executive Director.

- [ ] **Step 2: Refactor report approval and report list screens.**

  `viewLeadReviewQueue()`, `viewAuditReportsApproval()`, `viewReports()`, and `viewReport()` should use report/finding tables for source items and keep preview/dossier sections for report content.

- [ ] **Step 3: Refactor admin list screens.**

  `viewTemplates()`, `viewRegulatoryLibrary()`, `viewAuditLog()`, `viewOrganizations()`, `viewUsers()`, and `viewSettings()` should use table-first list surfaces where they are list/configuration surfaces.

- [ ] **Step 4: Refactor auditee communication and document screens.**

  `viewMessages()` and `viewReports()` under auditee should show communication/document tables with related finding/report context and no internal CAA data.

### Phase 6: Testing And Verification

- [ ] **Step 1: Add `tests/table-first-workbench-smoke.test.js`.**

  Verify:

  - `workItemFromFinding()` returns item, owner, next action, due text, route, and action.
  - Inspector dashboard render includes `My Work Today` and not the old `A. Attention Needed` / `B. My Upcoming Work` zone labels.
  - Auditee findings render includes `My CAA Requests` and does not include `Internal CAA Note`.
  - Manager dashboard render includes `Management Attention` and preserves OHI guardrail language.
  - Audit queue render still routes rows to `audit-detail`.

- [ ] **Step 2: Run syntax checks.**

  Run:

  ```bash
  node --check js/data.js
  node --check js/helpers.js
  node --check js/approval.js
  node --check js/planning.js
  node --check js/checklists.js
  node --check js/inspection.js
  node --check js/reports.js
  node --check js/views.js
  node --check js/app.js
  ```

  If `js/work-items.js` exists, include:

  ```bash
  node --check js/work-items.js
  ```

- [ ] **Step 3: Run targeted smoke tests.**

  Run the same baseline tests from Phase 0 plus the new table-first smoke test.

- [ ] **Step 4: Run browser visual QA.**

  Recreate desktop screenshots for the same 78 routes or at least the changed route set. Confirm:

  - no horizontal overflow at 1280px;
  - key row text is visible;
  - row click opens the expected page;
  - action buttons still perform the existing mock workflow;
  - auditee still cannot see internal CAA notes or other organizations.

- [ ] **Step 5: Apply the `ui-self-critique` loop.**

  For each changed screen group, classify the surface and remove at least one generic card/dashboard element that does not serve the main job.

### Phase 7: Documentation And Plan State

- [ ] **Step 1: Update build summary docs.**

  Update:

  - `docs/demo-evidence/BUILD_SUMMARY.md`
  - `docs/demo-evidence/BUILD_SUMMARY.turkce.md`

  Include:

  - table-first Surveillance Workbench change;
  - unchanged frontend-only demo boundary;
  - verification results;
  - any remaining mock behavior.

- [ ] **Step 2: Update `MANIFEST.md` if new files are added.**

  Add `js/work-items.js` and `tests/table-first-workbench-smoke.test.js` if they exist.

- [ ] **Step 3: Update `docs/exec-plans/index.md`.**

  Mark this plan `ready-for-verification` only after implementation and local checks pass.

## Verification

Minimum completion evidence:

- `verified locally` syntax checks for all changed JS files.
- `verified locally` targeted Node smoke tests.
- `verified locally` browser smoke through the main scenario:
  `Audit Work Queue -> Audit Detail -> Checklist -> Potential Finding -> Lead conversion -> Finding -> Auditee CAP -> Inspector CAP review -> Auditee evidence -> Inspector evidence review -> Closed -> Manager dashboard`.
- `verified locally` screenshot review for changed screens.
- `verified locally` no backend/database/API/auth/upload/email/framework migration added.
- `verified locally` auditee visibility boundary remains intact.

## Risks

| Risk | Mitigation |
|---|---|
| Table-first UI becomes too dense | Use attention strip, sticky/visible next-action column, and selected-row detail. |
| Important lifecycle context disappears | Keep lifecycle chips/stepper in rows and full lifecycle in dossier. |
| Auditee sees internal data | Reuse `visibleFindings()` and keep internal CAA notes excluded in auditee rendering. |
| Row action and row navigation conflict | Stop event propagation for action buttons if needed; keep row click for open/detail only. |
| Refactor touches too many screens at once | Execute by phases and verify after each screen group. |
| Existing user changes are overwritten | Inspect dirty files before editing and patch around unrelated changes. |
| Demo starts to imply production behavior | Preserve guardrails: demo data, mock regulatory library, no real upload, no production sync, no real AI service. |

## Execution Prompt

Use this exact prompt to execute the plan:

```text
Implement docs/exec-plans/active/2026-07-01-table-first-surveillance-workbench-ux-plan.md task by task. Preserve the frontend-only HTML/CSS/Vanilla JS architecture and all AviaSurveil360 product rules. Do not add a backend, database, API, real authentication, real file upload, real notification service, real AI service, real regulatory ingestion, framework migration, branch, commit, or push. Use ui-self-critique as the design quality gate for each changed screen group. Keep row-click navigation to existing detail pages, prioritize table-first work queues, preserve CAP accepted is not closure, preserve evidence version history, and ensure auditee users never see internal CAA notes, other organizations, inspector workload, or internal risk scoring. Run the verification steps in the plan and update docs/demo-evidence/BUILD_SUMMARY.md, docs/demo-evidence/BUILD_SUMMARY.turkce.md, MANIFEST.md if new files are added, and docs/exec-plans/index.md when complete.
```
