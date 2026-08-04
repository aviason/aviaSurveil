# UI Screenshot Audit Remediation Implementation Plan


**Goal:** Convert every Issue in the 19 July 2026 86-screen screenshot audit into a verified Pass while preserving all passing workflows and applying shared mobile-readability, next-action, and touch-target improvements that prevent recurrence.

**Architecture:** Keep the static HTML/CSS/Vanilla JavaScript architecture and browser-local state. Extend the accepted Inspector Assignments and Executive Planning table-to-card pattern in `js/views.js` and `css/styles.css`, reuse existing decision/workbench primitives, and add direct-Node contract tests before implementation. Finish with a fresh 86-route × 3-viewport Playwright matrix and synchronized English/Turkish evidence.

**Tech Stack:** HTML, CSS, Vanilla JavaScript, browser-local mock state, direct Node.js smoke tests, local static server, in-app Browser/Playwright rendered QA, Markdown evidence.

## Global Constraints

- Use `AviaSurveil360` as the canonical product name.
- Keep the implementation frontend-only, demo-only, and browser-local.
- Do not add a backend, database, API, real authentication, real authorization enforcement, real upload/storage, real AI service, real regulatory ingestion, real notification delivery, production audit log, mobile/offline production app, e-signature, framework migration, package manager, or external runtime dependency.
- Preserve the Finding -> CAP -> Evidence -> CAA Review -> Closure lifecycle; CAP acceptance is not finding closure.
- Preserve the separation between `Comment to Auditee` and `Internal CAA Note`.
- Preserve role visibility and organization scope; Auditee users see only their own organization and CAA-visible content.
- Keep regulatory wording careful: `reference`, `configured check`, `expected evidence`, `finding basis`, and `review result`.
- Preserve visible working behavior. Do not replace real controls with inert placeholders or toast-only substitutes.
- Dropdown-looking controls remain real dropdowns. Long values must be fully reviewable without clipped single-line fields.
- Reuse the existing palette, spacing, typography, line icons, cards, and workbench patterns; do not introduce a second design system.
- Use `Due Date`, `Target`, `Due Soon`, and `Overdue` consistently.
- At mobile width, primary and icon-only action targets touched by this plan must be at least `44px` in one dimension.
- Status remains understandable without color alone through visible text.
- Bump the shared asset query token in `index.html` whenever CSS or JavaScript behavior changes.
- Do not create, switch, rename, or delete branches. Do not stage, commit, or push unless the user explicitly authorizes that exact Git action in the execution thread.

---

## Objective

Close the 10 Issue routes recorded in:

- `docs/demo-evidence/UI_SCREEN_AUDIT_2026-07-19.md`
- `docs/demo-evidence/UI_SCREEN_AUDIT_2026-07-19.turkce.md`

The Issue routes are:

1. Inspector AI Inspector Assistant
2. Lead Inspector Assigned Audits
3. Lead Inspector Preliminary Reports
4. Lead Inspector Assign Checklist Questions
5. Department Manager Inspection Team
6. Department Manager Findings Review
7. Department Manager Checklist Management
8. Executive Director Dashboard at tablet width
9. Admin Question Bank
10. Admin Checklist Builder

The plan also applies the highest-value shared recommendations to long dossier screens so the fixes are systemic rather than one-off overrides.

## Acceptance Criteria

- All 10 Issue routes pass at `1440x1000`, `1024x900`, and `390x844`.
- The route inventory remains 86 screens and produces 258 accepted screenshots.
- Every accepted screenshot is non-blank, stable, and shows the intended role and route heading.
- `document.documentElement.scrollWidth === window.innerWidth` on all 258 checks.
- Critical nested workspaces report `scrollWidth <= clientWidth + 1` unless a deliberate scroller is visible and retains the primary action.
- Lead Assigned Audits, Lead Preliminary Reports, Lead Assignment Questions, Manager Inspection Team, Manager Findings Review, and Admin Checklist Builder render readable mobile record cards instead of clipped desktop rows.
- Manager Checklist Management and Admin Question Bank expose the complete question/reference value in multiline controls.
- Executive Dashboard decision queues stack cleanly at tablet width; Report ID, status, and CTA never collide.
- AI Inspector Assistant is reachable from Inspector finding/checklist context and returns to its source.
- Selected long dossier pages surface owner, next action, Due Date/Target, status, and primary decision near the first viewport.
- All visible controls retain meaningful browser-local behavior.
- Focused tests, the full direct-Node suite, JavaScript syntax checks, demo-boundary checks, and `git diff --check` pass.
- English/Turkish audit and build evidence match; no production-readiness claim is introduced.
- No task-owned server or browser-automation process remains after QA.

## Scope

### In scope

- Shared responsive record-card, decision-summary, touch-target, and overflow primitives.
- Markup changes for all 10 Issue routes.
- Contextual AI Assistant entry and return behavior.
- High-value next-action summaries for long operational dossier pages.
- Focused render/CSS contract tests.
- Full 86 × 3 Playwright recapture and audit evidence refresh.
- Plan index, technical-debt tracker, and bilingual evidence synchronization.

### Out of scope

- New modules, roles, or broad information-architecture changes.
- Rewriting all 76 passing screens solely for visual consistency.
- Changing lifecycle authority, approval authority, CAP closure, enforcement, or Auditee visibility.
- Seeding misleading legal/regulatory data to make an empty state look busy.
- Real accessibility certification; screen-reader and contrast evidence stay explicitly scoped unless measured.
- Production architecture, deployment, branch work, commits, pushes, PRs, or GitHub comments.

## Assumptions

- The accepted baseline is 76 Pass / 10 Issue.
- The current branch begins at or after `624d788 fix(ui): improve responsive workbench layouts`.
- Existing Inspector Assignments and Executive Planning card implementations are the responsive reference.
- There is no `package.json`; tests run directly with `node`.
- In-app Browser/Playwright remains available. A temporary harness may live in `/private/tmp`; do not add Playwright as a repository dependency.
- Executive Preliminary Reports may legitimately be empty; clarify the state instead of fabricating approval data.

## Ownership Boundaries

| Owner | Responsibility |
|---|---|
| Execution agent | Implement tasks in order, run focused checks, maintain plan/index state, capture evidence, and report literal verification status. |
| Product/stakeholder owner | Confirm card information priority and contextual AI entry match demo expectations. |
| Future production owners | Define real accessibility certification, identity, authorization, evidence storage, AI governance, regulatory governance, and deployment. |

## Dependencies

- `docs/demo-evidence/UI_SCREEN_AUDIT_2026-07-19.md`
- `docs/product-specs/ux-plan/UX_PRINCIPLES.md`
- `docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.md`
- `docs/product-specs/screen-specs/DEPARTMENT_MANAGER_WORKSPACES.md`
- `docs/exec-plans/active/2026-07-08-modern-aviation-saas-rollout-plan.md`
- `docs/agent-harness/verification-matrix.md`
- Current render/state functions in `js/views.js`, routing/actions in `js/app.js`, and responsive rules in `css/styles.css`.

## File Map

### Primary implementation

- Modify `index.html` — update the shared CSS/JS cache token once implementation is complete.
- Modify `css/styles.css` — shared responsive cards, split-pane containment, multiline fields, touch/type tokens, decision summaries, and route-specific fixes.
- Modify `js/views.js` — render mobile record companions, multiline fields, decision summaries, Executive queue structure, and contextual AI entry.
- Modify `js/app.js` — preserve active parent navigation for AI context, pass source parameters, and implement return navigation if generic navigation is insufficient.
- Modify `js/data.js` only if a browser-local UI default is required; do not add a competing dataset.

### Tests

- Create `tests/ui-screenshot-audit-remediation-smoke.test.js`.
- Modify `tests/lead-inspector-workspace-smoke.test.js`.
- Modify `tests/inspection-team-smoke.test.js`.
- Modify `tests/department-manager-findings-smoke.test.js`.
- Modify `tests/manager-checklist-management-smoke.test.js`.
- Modify `tests/executive-director-workspace-smoke.test.js`.
- Modify `tests/checklist-management-smoke.test.js`.
- Modify `tests/inspector-nav-smoke.test.js`.
- Modify `tests/manager-workspace-responsive-smoke.test.js`.
- Modify `tests/premium-ui-remediation-smoke.test.js`.

### Evidence and tracking

- Modify both `docs/demo-evidence/UI_SCREEN_AUDIT_2026-07-19*` files.
- Modify both `docs/demo-evidence/BUILD_SUMMARY*` files.
- Modify `docs/exec-plans/index.md`.
- Modify `docs/exec-plans/tech-debt-tracker.md`.
- Modify this plan's checkboxes and execution notes.

---

## Task 1: Freeze The Audit Contract With A Failing Consolidated Test

**Files:**

- Create: `tests/ui-screenshot-audit-remediation-smoke.test.js`

**Interfaces:**

- Consumes `freshState()`, current view functions, `css/styles.css`, and `js/app.js` source.
- Produces one gate naming every Issue route and the planned markup/CSS contract.

- [x] **Step 1: Confirm the working-tree baseline**

  Run:

  ```bash
  git status --short
  git log -3 --oneline
  ```

  Expected: no branch change; task-owned plan/audit docs may be modified; unrelated user files remain untouched.

- [x] **Step 2: Create the consolidated test harness**

  Start `tests/ui-screenshot-audit-remediation-smoke.test.js` with:

  ```js
  const assert = require('node:assert/strict');
  const fs = require('node:fs');
  const path = require('node:path');
  const vm = require('node:vm');

  const root = path.resolve(__dirname, '..');
  const context = { console, window: undefined, document: undefined, setTimeout, clearTimeout };
  vm.createContext(context);

  [
    'js/data.js', 'js/helpers.js', 'js/approval.js', 'js/planning.js',
    'js/checklists.js', 'js/inspection.js', 'js/reports.js',
    'js/manager-workspaces.js', 'js/work-items.js', 'js/views.js'
  ].forEach((file) => {
    vm.runInContext(fs.readFileSync(path.join(root, file), 'utf8'), context, { filename: file });
  });

  const css = fs.readFileSync(path.join(root, 'css/styles.css'), 'utf8');
  const app = fs.readFileSync(path.join(root, 'js/app.js'), 'utf8');

  function reset(role, view, params = {}) {
    context.state = context.freshState();
    context.state.role = role;
    context.state.view = view;
    context.state.params = params;
  }
  ```

- [x] **Step 3: Add exact route assertions**

  Append:

  ```js
  reset('leadInspector', 'lead-review');
  assert.match(context.viewLeadAssignedAudits(), /lead-assigned-mobile-list/);
  assert.match(context.viewLeadPreliminaryReports(), /lead-preliminary-mobile-list/);

  reset('leadInspector', 'lead-assignment-questions', { auditId: 'AUD-2026-001' });
  assert.match(context.viewLeadAssignmentQuestions(), /lead-assignment-question-mobile-list/);

  reset('manager', 'inspection-team');
  assert.match(context.viewInspectionTeam(), /manager-team-mobile-list/);

  reset('manager', 'findings-review');
  assert.match(context.viewManagerFindingsReview(), /manager-findings-mobile-list/);

  reset('manager', 'manager-checklists');
  assert.match(context.viewManagerChecklistManagement(), /manager-checklist-reference-textarea/);

  reset('executiveDirector', 'executive-dashboard');
  assert.match(context.viewExecutiveDirectorDashboard(), /executive-decision-queue/);

  reset('admin', 'question-bank');
  assert.match(context.viewQuestionBank(), /<textarea id="qb-text"/);

  reset('admin', 'checklist-builder');
  assert.match(context.viewChecklistBuilder(), /admin-checklist-mobile-list/);

  reset('inspector', 'finding', { findingId: 'F-2026-002' });
  assert.match(context.viewFindingDetail(), /data-view="ai-assistant"/);
  assert.match(app, /state\.view === 'ai-assistant'/);

  assert.match(css, /\.responsive-record-list/);
  assert.match(css, /\.mobile-decision-summary/);
  assert.match(css, /@media \(max-width:\s*640px\)[\s\S]*?min-height:\s*44px/);
  assert.match(css, /@media \(max-width:\s*1180px\)[\s\S]*?\.executive-dashboard-grid[\s\S]*?grid-template-columns:\s*1fr/);

  console.log('ui-screenshot-audit-remediation-smoke: ok');
  ```

- [x] **Step 4: Verify the red state**

  Run:

  ```bash
  node tests/ui-screenshot-audit-remediation-smoke.test.js
  ```

  Expected: FAIL on the first missing remediation marker, initially `lead-assigned-mobile-list`.

- [x] **Step 5: Record the exact first failure**

  Add a dated Task 1 execution note with the command, assertion, and literal `candidate-only` status. Do not claim verification.

### Task 1 Execution Note — 2026-07-19

- Status: `candidate-only`.
- Baseline: `624d788 fix(ui): improve responsive workbench layouts`; no branch action or Git mutation was performed.
- Red command: `node tests/ui-screenshot-audit-remediation-smoke.test.js`.
- Result: exit code `1` at `tests/ui-screenshot-audit-remediation-smoke.test.js:29` because `viewLeadAssignedAudits()` did not match `/lead-assigned-mobile-list/`.
- Scope boundary: no production CSS or JavaScript changed in Task 1.

---

## Task 2: Add Shared Responsive Record And Decision Primitives

**Files:**

- Modify: `css/styles.css`
- Modify: `tests/premium-ui-remediation-smoke.test.js`

**Interfaces:**

- Produces `.responsive-record-list`, `.responsive-record-card`, `.responsive-record-card__identity`, `.responsive-record-card__facts`, `.responsive-record-card__actions`, and `.mobile-decision-summary`.
- Desktop tables remain semantic and visible above each route's chosen card breakpoint.

- [x] **Step 1: Add failing shared-style assertions**

  Append:

  ```js
  assert.match(styles, /\.responsive-record-list\s*\{[^}]*display:\s*none/s);
  assert.match(styles, /\.responsive-record-card\s*\{[^}]*min-width:\s*0/s);
  assert.match(styles, /\.responsive-record-card__facts\s*\{[^}]*grid-template-columns:\s*repeat\(2,\s*minmax\(0,\s*1fr\)\)/s);
  assert.match(styles, /@media \(max-width:\s*640px\)[\s\S]*?\.responsive-record-card__actions[\s\S]*?min-height:\s*44px/s);
  assert.match(styles, /\.mobile-decision-summary\s*\{[^}]*min-width:\s*0/s);
  ```

- [x] **Step 2: Run the focused test and verify failure**

  ```bash
  node tests/premium-ui-remediation-smoke.test.js
  ```

  Expected: FAIL because `.responsive-record-list` is absent.

- [x] **Step 3: Add the shared CSS block**

  ```css
  .responsive-record-list { display: none; min-width: 0; }
  .responsive-record-card {
    min-width: 0;
    padding: 14px;
    background: var(--surface);
    border: 1px solid var(--line);
    border-radius: var(--radius-sm);
    box-shadow: var(--shadow-sm);
  }
  .responsive-record-card__identity,
  .responsive-record-card__facts,
  .responsive-record-card__actions,
  .mobile-decision-summary { min-width: 0; }
  .responsive-record-card__identity b,
  .responsive-record-card__identity small,
  .responsive-record-card__facts b,
  .responsive-record-card__facts small { display: block; overflow-wrap: anywhere; }
  .responsive-record-card__facts {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
    margin-top: 12px;
  }
  .responsive-record-card__facts span {
    min-width: 0;
    padding: 9px 10px;
    background: var(--surface-alt);
    border: 1px solid var(--line-soft);
    border-radius: 8px;
  }
  .responsive-record-card__facts small { color: var(--muted); font-size: 11px; }
  .responsive-record-card__actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: 12px;
  }
  .mobile-decision-summary {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 8px;
    min-width: 0;
    padding: 12px;
    background: var(--blue-50);
    border: 1px solid #d4e2f7;
    border-left: 4px solid var(--blue-500);
    border-radius: var(--radius-sm);
  }
  @media (max-width: 640px) {
    .responsive-record-card__facts,
    .mobile-decision-summary { grid-template-columns: 1fr; }
    .responsive-record-card__actions .btn,
    .responsive-record-card__actions button,
    .mobile-decision-summary .btn { min-height: 44px; }
  }
  ```

- [x] **Step 4: Run focused tests**

  ```bash
  node tests/premium-ui-remediation-smoke.test.js
  node tests/ui-screenshot-audit-remediation-smoke.test.js
  ```

  Expected: premium test PASS; consolidated audit test advances to route-specific markers.

### Task 2 Execution Note — 2026-07-19

- Red: `node tests/premium-ui-remediation-smoke.test.js` exited `1` because `.responsive-record-list { display: none; }` was absent.
- Green: `node tests/premium-ui-remediation-smoke.test.js` exited `0` with `premium-ui-remediation-smoke: ok` after the shared primitive block was added.
- Progress gate: `node tests/ui-screenshot-audit-remediation-smoke.test.js` still exits `1` at the expected first route-specific marker, `/lead-assigned-mobile-list/`.

---

## Task 3: Remediate Lead Inspector Dense Workbenches

**Files:**

- Modify: `js/views.js` around `viewLeadAssignedAudits`, `viewLeadPreliminaryReports`, and `viewLeadAssignmentQuestions`
- Modify: `css/styles.css` around `.lead-assigned-*`, Preliminary Report, and `.lead-assignment-question-*` rules
- Modify: `tests/lead-inspector-workspace-smoke.test.js`

**Interfaces:**

- Produces `leadAssignedMobileCardsHtml(rows)`, `leadPreliminaryMobileCardsHtml(rows)`, and `leadAssignmentQuestionMobileCardsHtml(rows, ui)`.
- Cards preserve identity, organization, status/progress, owner/inspector, Due Date/Target when present, and existing actions/data attributes.
- Desktop tables, filters, and pagination remain unchanged.

- [x] **Step 1: Add failing Lead assertions**

  Add after existing desktop assertions:

  ```js
  assert.match(html, /class="responsive-record-list lead-assigned-mobile-list"/);
  assert.match(html, /data-mobile-record="AUD-2025-045"/);
  assert.match(html, /Organization[\s\S]*Fly Namibia/);
  assert.match(html, /Progress[\s\S]*75%/);
  assert.match(html, /data-view="lead-assignment"/);

  context.state.view = 'lead-assignment-questions';
  context.state.params = { auditId: 'AUD-2026-001' };
  html = context.viewLeadAssignmentQuestions();
  assert.match(html, /lead-assignment-question-mobile-list/);
  assert.match(html, /Risk Level/);
  assert.match(html, /Assigned Inspector/);

  context.state.view = 'audit-reports';
  context.state.params = { filter: 'preliminary' };
  html = context.viewLeadPreliminaryReports();
  assert.match(html, /lead-preliminary-mobile-list/);
  assert.match(html, /Report ID/);
  assert.match(html, /Organization/);
  ```

- [x] **Step 2: Verify the red state**

  ```bash
  node tests/lead-inspector-workspace-smoke.test.js
  ```

  Expected: FAIL on `lead-assigned-mobile-list`.

- [x] **Step 3: Implement Assigned Audits cards from existing row objects**

  Add `leadAssignedMobileCardsHtml(rows)` beside `leadAssignedTableHtml`. Use the actual row property names returned by `leadAssignedAuditFilteredRows`; do not duplicate mock records. Required card shape:

  ```html
  <article class="responsive-record-card" data-mobile-record="AUD-2025-045">
    <div class="responsive-record-card__identity">Audit ID and type</div>
    <div class="responsive-record-card__facts">Organization, Department, Risk, Progress</div>
    <div class="responsive-record-card__actions">Existing Open action</div>
  </article>
  ```

- [x] **Step 4: Implement Preliminary Report cards**

  Reuse each report's actual ID, Audit ID, organization, inspection type, status, owner/Lead Inspector, submitted date, and existing open-package action. Preserve identical `data-act`, `data-id`, and route params between table and card.

- [x] **Step 5: Implement Assignment Question cards**

  Each card contains the existing checkbox/question identifier, complete question text, section, risk as visible text, assigned/unassigned Inspector state, and existing selection behavior. Ensure each actionable control appears exactly once at the active breakpoint.

- [x] **Step 6: Add route-specific visibility rules**

  ```css
  @media (max-width: 1100px) {
    .lead-assigned-table-panel,
    .lead-preliminary-table-wrap,
    .lead-assignment-question-table-panel { display: none; }
    .lead-assigned-mobile-list,
    .lead-preliminary-mobile-list,
    .lead-assignment-question-mobile-list {
      display: grid;
      grid-template-columns: 1fr;
      gap: 12px;
    }
  }
  @media (min-width: 800px) and (max-width: 1100px) {
    .lead-assigned-mobile-list,
    .lead-preliminary-mobile-list,
    .lead-assignment-question-mobile-list {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }
  ```

- [x] **Step 7: Run focused Lead checks**

  ```bash
  node --check js/views.js
  node tests/lead-inspector-workspace-smoke.test.js
  node tests/lead-inspector-nav-smoke.test.js
  node tests/ui-screenshot-audit-remediation-smoke.test.js
  ```

  Expected: Lead tests PASS; consolidated audit test advances to Manager markers.

### Task 3 Execution Note — 2026-07-19

- Red: `node tests/lead-inspector-workspace-smoke.test.js` exited `1` on the missing `responsive-record-list lead-assigned-mobile-list` contract.
- Green: `node --check js/views.js`, `node tests/lead-inspector-workspace-smoke.test.js`, and `node tests/lead-inspector-nav-smoke.test.js` all exited `0`.
- Progress gate: `node tests/ui-screenshot-audit-remediation-smoke.test.js` advanced to the expected Manager failure, `/manager-team-mobile-list/`.

---

## Task 4: Remediate Manager Inspection Team And Findings Review

**Files:**

- Modify: `js/views.js` around `managerTeamFilters`, `managerTeamTable`, `managerTeamMemberTable`, `viewInspectionTeam`, `managerFindingsFilters`, `managerFindingsInspectionTable`, and `viewManagerFindingsReview`
- Modify: `css/styles.css` around `.manager-team-*` and `.manager-findings-*`
- Modify: `tests/inspection-team-smoke.test.js`
- Modify: `tests/department-manager-findings-smoke.test.js`
- Modify: `tests/manager-workspace-responsive-smoke.test.js`

**Interfaces:**

- Produces `managerTeamMobileCards(rows, selectedAuditId, openMenuAuditId)`, `managerTeamMemberMobileCards(row, allowActions)`, and `managerFindingsMobileCards(rows, selectedAuditId)`.
- Reuses current `data-act`, `data-id`, `data-user`, and `data-tab` values so browser-local mutations remain authoritative.
- Filter grids fit the actual split-pane width, not only the document viewport.

- [x] **Step 1: Add failing Inspection Team assertions**

  Render `viewInspectionTeam()` in `tests/inspection-team-smoke.test.js` and add:

  ```js
  assert.match(html, /manager-team-mobile-list/);
  assert.match(html, /data-mobile-record="AUD-2026-001"/);
  assert.match(html, /Current Status/);
  assert.match(html, /data-act="manager-team-select"/);
  assert.match(html, /data-act="manager-team-menu"/);
  assert.match(html, /manager-team-member-mobile-list/);
  ```

- [x] **Step 2: Add failing Findings Review assertions**

  Render `viewManagerFindingsReview()` in `tests/department-manager-findings-smoke.test.js` and add:

  ```js
  assert.match(html, /manager-findings-mobile-list/);
  assert.match(html, /data-mobile-record="AUD-2026-001"/);
  assert.match(html, /Organization/);
  assert.match(html, /Audit Date/);
  assert.match(html, /Team Leader/);
  assert.match(html, /Status/);
  assert.match(html, /data-act="manager-findings-select"/);
  ```

- [x] **Step 3: Add failing split-pane CSS assertions**

  Append to `tests/manager-workspace-responsive-smoke.test.js`:

  ```js
  assert.match(styles, /\.manager-team-filters\s*\{[^}]*grid-template-columns:\s*repeat\(2,\s*minmax\(0,\s*1fr\)\)/s);
  assert.match(styles, /\.manager-team-search\s*\{[^}]*grid-column:\s*1\s*\/\s*-1/s);
  assert.match(styles, /\.manager-findings-filters\s*\{[^}]*grid-template-columns:\s*repeat\(2,\s*minmax\(0,\s*1fr\)\)/s);
  assert.match(styles, /@media \(max-width:\s*1100px\)[\s\S]*?\.manager-team-mobile-list[\s\S]*?display:\s*grid/s);
  assert.match(styles, /@media \(max-width:\s*1100px\)[\s\S]*?\.manager-findings-mobile-list[\s\S]*?display:\s*grid/s);
  ```

- [x] **Step 4: Verify the red state**

  ```bash
  node tests/inspection-team-smoke.test.js
  node tests/department-manager-findings-smoke.test.js
  node tests/manager-workspace-responsive-smoke.test.js
  ```

  Expected: FAIL on missing card/filter contracts.

- [x] **Step 5: Reflow split-pane filters at wide desktop**

  Use two contained columns because each list pane is approximately `520-560px` wide in the `1440px` split layout:

  ```css
  .manager-team-filters,
  .manager-findings-filters { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .manager-team-search,
  .manager-findings-search { grid-column: 1 / -1; }
  .manager-team-reset,
  .manager-findings-reset { justify-self: stretch; min-height: 36px; }
  ```

  Check cascade order so later media blocks do not restore the clipping grid.

- [x] **Step 6: Render Inspection Team cards**

  Each audit card shows Audit ID/type, organization/department, schedule, Lead Inspector, team count, status, select action, and existing action-menu trigger. Member cards show role, name, department, full email, status, and Make Lead/Remove actions when allowed.

- [x] **Step 7: Render Findings Review cards**

  Each card shows Audit ID/type, organization, Audit Date, Team Leader, status, total findings/severity summary, and select/open action. Keep the selected dossier after the list in mobile document order.

- [x] **Step 8: Add responsive visibility and tab safeguards**

  At `max-width: 1100px`, hide only the clipped register tables and show two card columns through `800px`; at `max-width: 640px`, use one column and full-width actions. Keep `overflow-x: auto` on tab lists, add `scrollbar-gutter: stable`, preserve full labels, and retain visible focus.

- [x] **Step 9: Run focused Manager checks**

  ```bash
  node --check js/views.js
  node tests/inspection-team-smoke.test.js
  node tests/department-manager-findings-smoke.test.js
  node tests/manager-workspace-responsive-smoke.test.js
  node tests/department-manager-state-smoke.test.js
  node tests/ui-screenshot-audit-remediation-smoke.test.js
  ```

  Expected: Manager tests PASS; consolidated audit test advances to Checklist/Executive/Admin markers.

### Task 4 Execution Note — 2026-07-19

- Red: the Inspection Team and Findings Review tests exited `1` on their missing mobile-list markers; the responsive test exited `1` on the missing `max-width: 1100px` card visibility contract.
- Green: `node --check js/views.js`, both route smoke tests, `manager-workspace-responsive-smoke`, and `department-manager-state-smoke` exited `0`.
- Progress gate: the consolidated audit test advanced to the Manager Checklist multiline marker, `/manager-checklist-reference-textarea/`.

---

## Task 5: Expose The Complete Manager Checklist Reference

**Files:**

- Modify: `js/views.js` in `viewManagerChecklistManagement`
- Modify: `css/styles.css` near Manager Checklist Management rules
- Modify: `tests/manager-checklist-management-smoke.test.js`

**Interfaces:**

- Replaces the clipped `#manager-checklist-question-reference` single-line control with a multiline field.
- Preserves the existing field key, delegated handler, browser-local value, and reference-only wording.

- [x] **Step 1: Add a failing multiline assertion**

  ```js
  assert.match(html, /<textarea[^>]*id="manager-checklist-question-reference"[^>]*class="manager-checklist-reference-textarea"/);
  assert.match(html, /Configured Requirement \/ Reference/);
  assert.match(html, /ICAO Annex 6/);
  assert.doesNotMatch(html, /<input[^>]*id="manager-checklist-question-reference"/);
  ```

- [x] **Step 2: Verify the red state**

  ```bash
  node tests/manager-checklist-management-smoke.test.js
  ```

  Expected: FAIL because the control is currently an input.

- [x] **Step 3: Replace the input without changing state wiring**

  Use escaped content and the existing `data-field`:

  ```html
  <textarea id="manager-checklist-question-reference"
            class="manager-checklist-reference-textarea"
            data-field="manager-checklist-question-reference"
            rows="3">Configured reference value</textarea>
  ```

- [x] **Step 4: Add readable field styling**

  ```css
  .manager-checklist-reference-textarea {
    width: 100%;
    min-height: 78px;
    resize: vertical;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    line-height: 1.45;
  }
  ```

- [x] **Step 5: Run focused checks**

  ```bash
  node --check js/views.js
  node tests/manager-checklist-management-smoke.test.js
  node tests/checklist-comment-render-smoke.test.js
  node tests/ui-screenshot-audit-remediation-smoke.test.js
  ```

  Expected: tests PASS and the complete reference remains present in render output.

### Task 5 Execution Note — 2026-07-19

- Red: `node tests/manager-checklist-management-smoke.test.js` exited `1` because `#manager-checklist-question-reference` was still a single-line input.
- Green: `node --check js/views.js`, `manager-checklist-management-smoke`, and `checklist-comment-render-smoke` exited `0`; the same ID remains consumed by the existing save handler.
- Progress gate: the consolidated audit test now reaches the Executive decision-queue marker.

---

## Task 6: Remediate Admin Question Bank And Checklist Builder

**Files:**

- Modify: `js/views.js` in `viewQuestionBank` and `viewChecklistBuilder`
- Modify: `css/styles.css` around `.configuration-*`, `.ops-table-wrap`, and Admin mobile rules
- Modify: `tests/checklist-management-smoke.test.js`
- Modify: `tests/premium-ui-remediation-smoke.test.js`

**Interfaces:**

- Question Bank exposes `#qb-text` as a multiline textarea while retaining `data-act="qb-create"`.
- Checklist Builder produces `adminChecklistVersionMobileCardsHtml(working, isEditable)` and `adminQuestionBankMobileCardsHtml(working, isEditable)` from the desktop table objects.
- Existing Add, Up, Down, and read-only/version-state behavior remains intact.

- [x] **Step 1: Add failing Admin assertions**

  ```js
  html = context.viewQuestionBank();
  assert.match(html, /<textarea id="qb-text"[^>]*rows="3"/);
  assert.match(html, /data-act="qb-create"/);

  html = context.viewChecklistBuilder();
  assert.match(html, /admin-checklist-mobile-list/);
  assert.match(html, /admin-question-bank-mobile-list/);
  assert.match(html, /Configured reference/);
  assert.match(html, /Expected evidence/);
  assert.match(html, /data-act="checklist-move-question"/);
  ```

- [x] **Step 2: Verify the red state**

  ```bash
  node tests/checklist-management-smoke.test.js
  ```

  Expected: FAIL because `#qb-text` is an input and mobile lists are absent.

- [x] **Step 3: Convert Question Bank question text to a textarea**

  ```html
  <div class="form-row form-row--full">
    <label for="qb-text">Question text</label>
    <textarea id="qb-text" rows="3">Does the training matrix reconcile to sampled crew certificate evidence?</textarea>
    <small class="field-help">Use a complete, testable inspection question.</small>
  </div>
  ```

  Preserve `#qb-title`, the create action, escaping, and browser-local save behavior.

- [x] **Step 4: Render current-version mobile cards**

  Each card shows order, full question, Question ID, version status, and Up/Down controls when editable. Edge actions remain visibly disabled rather than disappearing.

- [x] **Step 5: Render reusable Question Bank mobile cards**

  Each card shows title, full question text, full configured reference, expected evidence, and Add/In version/Read-only state. Apply `overflow-wrap: anywhere` to long content.

- [x] **Step 6: Add responsive visibility rules**

  Hide only the two Admin tables at `max-width: 760px`; show one-column `.admin-checklist-mobile-list` and `.admin-question-bank-mobile-list`. Keep readable desktop/tablet tables intact.

- [x] **Step 7: Run focused Admin checks**

  ```bash
  node --check js/views.js
  node tests/checklist-management-smoke.test.js
  node tests/checklist-approval-smoke.test.js
  node tests/premium-ui-remediation-smoke.test.js
  node tests/ui-screenshot-audit-remediation-smoke.test.js
  ```

  Expected: Admin tests PASS; consolidated test advances to Executive/AI markers.

### Task 6 Execution Note — 2026-07-19

- Red: `checklist-management-smoke` exited `1` because `#qb-text` was a single-line input; the new premium CSS assertions then exited `1` because Admin mobile visibility rules were absent.
- Green: `node --check js/views.js`, `checklist-management-smoke`, `checklist-approval-smoke`, and `premium-ui-remediation-smoke` exited `0`. Editable versions retain Add/Up/Down actions, edge ordering actions render disabled, and locked versions remain explicit.
- Progress gate: the consolidated audit test now stops only at the Executive decision-queue marker, which is owned by Task 7.

---

## Task 7: Reflow Executive Decision Queues At Tablet Width

**Files:**

- Modify: `js/views.js` around `executivePlanningQueueHtml`, `executiveReportQueueHtml`, `viewExecutiveDirectorDashboard`, and `viewExecutivePreliminaryReportsWorkspace`
- Modify: `css/styles.css` around `.executive-dashboard-*`, `.executive-decision-list`, and `.executive-empty`
- Modify: `tests/executive-director-workspace-smoke.test.js`
- Modify: `tests/manager-workspace-responsive-smoke.test.js`

**Interfaces:**

- Adds `.executive-decision-queue` to both dashboard decision panels.
- At `max-width: 1180px`, queues stack and each item places identity, status, and CTA vertically without collision.
- Preliminary empty state explains source/next action without fabricated reports.

- [x] **Step 1: Add failing render assertions**

  ```js
  html = context.viewExecutiveDirectorDashboard();
  assert.equal((html.match(/executive-decision-queue/g) || []).length, 2);
  assert.match(html, /Final Report approvals/);
  assert.match(html, /Review report/);

  html = context.viewExecutivePreliminaryReportsWorkspace();
  assert.match(html, /No eligible Preliminary Reports/);
  assert.match(html, /Reports reach this queue after General Manager approval/);
  assert.match(html, /data-view="executive-notifications"/);
  ```

- [x] **Step 2: Add failing tablet CSS assertions**

  ```js
  assert.match(styles, /@media \(max-width:\s*1180px\)[\s\S]*?\.executive-dashboard-grid\s*\{[^}]*grid-template-columns:\s*1fr/s);
  assert.match(styles, /@media \(max-width:\s*1180px\)[\s\S]*?\.executive-decision-list article\s*\{[^}]*grid-template-columns:\s*1fr/s);
  assert.match(styles, /\.executive-decision-list article[^}]*min-width:\s*0/s);
  ```

- [x] **Step 3: Verify the red state**

  ```bash
  node tests/executive-director-workspace-smoke.test.js
  node tests/manager-workspace-responsive-smoke.test.js
  ```

  Expected: FAIL on missing queue class or tablet rule.

- [x] **Step 4: Add queue class and tablet reflow**

  ```css
  .executive-decision-list article,
  .executive-decision-list article > div { min-width: 0; }
  @media (max-width: 1180px) {
    .executive-dashboard-grid { grid-template-columns: 1fr; }
    .executive-decision-list article {
      grid-template-columns: 1fr;
      align-items: stretch;
    }
    .executive-decision-list article > div:last-child { justify-content: flex-start; }
  }
  ```

- [x] **Step 5: Improve the valid Preliminary empty state**

  Explain that reports enter after General Manager approval, preserve approved/returned items elsewhere, and add one working navigation action to Notifications. Do not seed fake approval data solely for visual density.

- [x] **Step 6: Run focused Executive checks**

  ```bash
  node --check js/views.js
  node tests/executive-director-workspace-smoke.test.js
  node tests/manager-workspace-responsive-smoke.test.js
  node tests/ui-screenshot-audit-remediation-smoke.test.js
  ```

  Expected: Executive tests PASS; consolidated test advances to AI or completes.

### Task 7 Execution Note — 2026-07-19

- Red: the Executive workspace smoke exited `1` with zero `.executive-decision-queue` panels, and the responsive smoke exited `1` because the `1180px` reflow contract was absent.
- Green: `node --check js/views.js`, `executive-director-workspace-smoke`, and `manager-workspace-responsive-smoke` exited `0`. Both decision panels now stack safely at tablet width, and the valid Preliminary empty state links to Notifications without fabricating records.
- Progress gate: the consolidated audit test reaches its AI assertion, where it exposes the plan's stale `viewFindingDetail()` test name; the repository render function is `viewFinding()`, so Task 8 will correct the contract before driving the contextual entry red state.

---

## Task 8: Add Contextual AI Inspector Assistant Entry And Return

**Files:**

- Modify: `js/views.js` around Inspector Finding Detail, Checklist Runner, and `viewAiAssistant`
- Modify: `js/app.js` around `isNavActive` and generic navigation dispatch
- Modify: `tests/inspector-nav-smoke.test.js`
- Modify: `tests/premium-ui-remediation-smoke.test.js`

**Interfaces:**

- Entry uses `data-act="nav"`, `data-view="ai-assistant"`, `data-source-view`, and relevant record ID.
- AI view consumes `state.params.sourceView`, `findingId` or `auditId`, and renders a working Back action.
- Findings stays active while AI is opened from a finding.

- [x] **Step 1: Add failing contextual navigation tests**

  ```js
  context.state.role = 'inspector';
  context.state.view = 'finding';
  context.state.params = { findingId: 'F-2026-002' };
  let findingHtml = context.viewFindingDetail();
  assert.match(findingHtml, /data-view="ai-assistant"/);
  assert.match(findingHtml, /data-source-view="finding"/);
  assert.match(findingHtml, /data-id="F-2026-002"/);

  context.state.view = 'ai-assistant';
  context.state.params = { sourceView: 'finding', findingId: 'F-2026-002' };
  const aiHtml = context.viewAiAssistant();
  assert.match(aiHtml, /Back to Finding/);
  assert.match(aiHtml, /data-view="finding"/);
  assert.match(aiHtml, /F-2026-002/);
  assert.match(appSource, /view === 'findings' && state\.view === 'ai-assistant'/);
  ```

- [x] **Step 2: Verify the red state**

  ```bash
  node tests/inspector-nav-smoke.test.js
  ```

  Expected: FAIL because no contextual AI entry exists.

- [x] **Step 3: Add the Finding entry**

  Place a secondary `AI draft assistance` button near finding description/reference tools, never as the primary lifecycle decision. Pass Finding ID and source view.

- [x] **Step 4: Add the Checklist entry only when a question is selected**

  Pass audit/question context. Keep `AI-generated draft - requires authorized review` and demo-only copy visible.

- [x] **Step 5: Add source context and Back action**

  Derive context only from existing mock records. With absent params, show a demo preview plus a working Back to My Assignments action; do not leave an orphaned route.

- [x] **Step 6: Preserve active parent navigation**

  Extend `isNavActive` so Findings remains active for AI opened from a finding and My Assignments/Calendar remains active for AI opened from a checklist. Do not add AI as broad primary navigation.

- [x] **Step 7: Run focused navigation and boundary checks**

  ```bash
  node --check js/views.js
  node --check js/app.js
  node tests/inspector-nav-smoke.test.js
  node tests/demo-boundary-smoke.test.js
  node tests/premium-ui-remediation-smoke.test.js
  node tests/ui-screenshot-audit-remediation-smoke.test.js
  ```

  Expected: all tests PASS; AI remains explicitly mock/draft assistance.

### Task 8 Execution Note — 2026-07-19

- Contract correction: the plan's sample used non-existent `viewFindingDetail()` and a non-canonical `F-2026-002` Finding fixture. Tests now exercise the actual `viewFinding()` surface with seeded `SEC-2026-002`.
- Red: `inspector-nav-smoke` exited `1` because the rendered Finding had no contextual AI entry.
- Green: both Finding and selected checklist-question routes pass source context into the assistant; the assistant renders a working source-aware Back action, an unscoped preview falls back to My Assignments, and parent navigation remains active. `node --check` for both JavaScript files plus Inspector nav, demo-boundary, premium UI, and consolidated audit smokes all exited `0`.

---

## Task 9: Apply Shared Next-Action, Touch, Type, And Long-Page Recommendations

**Files:**

- Modify: `js/views.js`
- Modify: `css/styles.css`
- Modify: `tests/premium-ui-remediation-smoke.test.js`
- Modify: closest existing route test when its markup changes

**Interfaces:**

- Reuses existing `decision-bar`, `commandMetric`, `workbench-command`, and action markup.
- Produces a compact `.mobile-decision-summary` on selected long screens.
- Does not change lifecycle status, owner, or decision authority.

- [x] **Step 1: Lock the exact screen set**

  Apply summaries only to:

  1. Inspector Audit Detail
  2. Inspector Finding Detail
  3. Lead CAP Review Detail
  4. Manager Organization Risk Profile
  5. Manager Organization Detail
  6. Finance Review
  7. Auditee Corrective Actions
  8. Executive Report Preview

  Each summary uses available values for current owner, next action, Due Date/Target, status, and one primary action. Use existing domain-safe fallbacks when a value is absent; do not invent records.

- [x] **Step 2: Add failing render assertions**

  In each closest route test, assert exactly one `.mobile-decision-summary` and literal `Current owner`, `Next action`, and `Due Date` or `Target` labels.

- [x] **Step 3: Render summaries from existing selected records**

  If a screen already has `workbench-command` or `decision-bar`, add the mobile class to that element instead of creating duplicate mutable UI. Keep the primary action's existing data attributes.

- [x] **Step 4: Audit shared mobile targets**

  At `max-width: 640px`, set `min-height: 44px` for touched route tabs, pagination, icon-only row actions, and primary/secondary form actions. Do not globally inflate inline text links.

- [x] **Step 5: Raise only sub-12px operational labels**

  Introduce `--text-label-mobile: 12px` for actionable labels and card metadata at mobile width. Demo/legal boundary fine print may remain smaller when non-interactive, but never below `11px`.

- [x] **Step 6: Preserve non-color status meaning**

  Every touched status component must retain visible status text. Add visible text where a dot alone currently carries meaning.

- [x] **Step 7: Run focused and cross-route checks**

  Run each closest route test after its change, then:

  ```bash
  node tests/premium-ui-remediation-smoke.test.js
  node tests/stakeholder-readiness-regressions.test.js
  node tests/demo-boundary-smoke.test.js
  node tests/ui-screenshot-audit-remediation-smoke.test.js
  ```

  Expected: PASS with no lifecycle or boundary regression.

### Task 9 Execution Note — 2026-07-19

- Red: all five closest route tests exited `1` because their covered screens rendered zero `.mobile-decision-summary` sections.
- Green: the exact eight-screen set now renders one source-backed summary each with Current owner, Next action, Due Date/Target, visible Status text, and a working existing-style action. Mobile touch targets are at least `44px`, actionable labels use `--text-label-mobile: 12px`, and scoped boundary fine print remains at least `11px`.
- Verification: Inspector nav, Lead/premium UI, Finance, Service Provider, and Executive route tests passed; stakeholder readiness reported `22/22`; demo-boundary and consolidated audit smokes also exited `0`.

---

## Task 10: Refresh Asset Tokens And Run The Complete Automated Gate

**Files:**

- Modify: `index.html`
- Verify: all `js/*.js`
- Verify: all `tests/*.test.js`

**Interfaces:**

- Produces one shared dated asset token for the stylesheet and eleven frontend scripts.
- Produces a clean direct-Node and syntax baseline before rendered QA.

- [x] **Step 1: Set the new shared asset token**

  Replace `20260719-mobile-workbench-v1` with:

  ```text
  20260719-ui-audit-remediation-v1
  ```

- [x] **Step 2: Verify token consistency**

  ```bash
  rg -n "20260719-ui-audit-remediation-v1|20260719-mobile-workbench-v1" index.html
  ```

  Expected: the new token appears on every asset reference; the old token appears zero times.

- [x] **Step 3: Run every JavaScript syntax check**

  ```bash
  for file in js/*.js; do node --check "$file" || exit 1; done
  ```

  Expected: exit code 0 with no syntax errors.

- [x] **Step 4: Run the focused remediation suite**

  ```bash
  node tests/ui-screenshot-audit-remediation-smoke.test.js
  node tests/lead-inspector-workspace-smoke.test.js
  node tests/inspection-team-smoke.test.js
  node tests/department-manager-findings-smoke.test.js
  node tests/manager-checklist-management-smoke.test.js
  node tests/checklist-management-smoke.test.js
  node tests/executive-director-workspace-smoke.test.js
  node tests/inspector-nav-smoke.test.js
  node tests/manager-workspace-responsive-smoke.test.js
  node tests/premium-ui-remediation-smoke.test.js
  node tests/demo-boundary-smoke.test.js
  ```

  Expected: every command exits 0.

- [x] **Step 5: Run the complete direct-Node suite**

  ```bash
  for test_file in tests/*.test.js; do node "$test_file" || exit 1; done
  ```

  Expected: every test exits 0; record the exact test-file/pass count rather than assuming the historical count.

- [x] **Step 6: Run whitespace and boundary checks**

  ```bash
  git diff --check
  node tests/demo-boundary-smoke.test.js
  ```

  Expected: both pass.

### Task 10 Execution Note — 2026-07-19

- Asset contract: all 12 stylesheet/script references use `20260719-ui-audit-remediation-v1`; the prior token appears zero times in `index.html`.
- Automated gate: all `js/*.js` syntax checks, the 11-test focused remediation suite, and all `39/39` discovered direct-Node test files passed.
- Boundary/whitespace gate: `git diff --check` and `demo-boundary-smoke` exited `0`.

---

## Task 11: Run Fresh 86-Route Playwright QA And Inspect Every Screenshot

**Files:**

- Create temporary evidence: `/private/tmp/aviasurveil360-ui-audit-remediation-2026-07-19/`
- Modify after verification: both `docs/demo-evidence/UI_SCREEN_AUDIT_2026-07-19*` files

**Interfaces:**

- Consumes the 86-screen inventory and current role/view routing.
- Produces 258 accepted screenshots, `capture-results.json`, viewport metrics, console results, and contact sheets.

- [x] **Step 1: Start an isolated local server**

  ```bash
  python3 -m http.server 4173 --bind 127.0.0.1
  ```

  Expected: `http://127.0.0.1:4173/index.html` loads current assets.

- [x] **Step 2: Use the Browser skill and an isolated Browser/Playwright session**

  Do not use the user's everyday Chrome profile. Capture:

  - desktop `1440x1000`
  - tablet `1024x900`
  - mobile `390x844`

  Name files `<viewport>__<route-slug>.png`.

- [x] **Step 3: Capture the 10 former Issue routes first**

  Record this metric shape for every route/viewport:

  ```js
  {
    role,
    route,
    viewport,
    heading,
    bodyWidth: document.body.scrollWidth,
    documentWidth: document.documentElement.scrollWidth,
    innerWidth: window.innerWidth,
    nestedOverflow: [...document.querySelectorAll(
      '[data-qa-overflow], .ops-table-wrap, .manager-team-list, .manager-findings-inspections, .executive-decision-queue'
    )].filter((node) => node.scrollWidth > node.clientWidth + 1)
      .map((node) => ({
        className: node.className,
        clientWidth: node.clientWidth,
        scrollWidth: node.scrollWidth
      })),
    consoleIssues
  }
  ```

  Expected: correct heading, zero document overflow, zero unintended nested overflow, zero console warnings/errors, and readable cards/fields/actions.

- [x] **Step 4: Inspect all first 30 screenshots individually**

  Reject and recapture blank, loading, cropped, stale, or wrong-state files. Dimensions and byte size alone are insufficient acceptance evidence.

- [x] **Step 5: Capture the remaining 76 routes at all three viewports**

  Preserve or reset browser-local state intentionally. Record the role-family reset policy in `capture-results.json`.

- [x] **Step 6: Inspect all 258 screenshots**

  Use contact sheets to navigate the set, then individually open every file flagged by overflow, tiny text, clipped content, unusual height, empty state, or route mismatch. Contact sheets do not replace individual flagged-image inspection.

- [x] **Step 7: Run interaction checks for changed controls**

  Verify:

  - Lead audit/report/question card actions open the correct existing detail.
  - Manager Team select/menu/member actions update visible state or open their existing focused UI.
  - Manager Findings card selection updates the dossier.
  - Manager Checklist and Question Bank multiline values edit/save through existing handlers.
  - Admin Checklist Builder Add/Up/Down remains functional where allowed.
  - Executive decision CTA opens the correct report.
  - AI opens from a Finding and returns to the same Finding.
  - Keyboard Tab reaches every new action and focus remains visible.

- [x] **Step 8: Record accessibility evidence boundaries**

  Report only locally checked visible focus, keyboard reachability, text wrapping, status text, and touch dimensions. Keep screen-reader and contrast certification as `not run` unless separately measured.

- [x] **Step 9: Stop and clean up automation**

  Finalize the browser, stop the server, then run:

  ```bash
  ps -axo pid,ppid,stat,command | egrep "python3 -m http.server 4173|playwright|puppeteer|webdriver|HeadlessChrome|Google Chrome.*remote-debugging" | egrep -v "egrep|rg"
  ```

  Expected: no task-owned process remains. Do not stop unrelated user processes.

### Task 11 Execution Note — 2026-07-19

- Isolated QA: served the current tree on `127.0.0.1:4173` and used the
  isolated in-app Browser session at `1440x1000`, `1024x900`, and `390x844`.
  A temporary QA-only wrapper selected deterministic seeded deep routes; it did
  not change production navigation and was removed after capture.
- TDD from rendered findings: Browser inspection exposed the Checklist Builder
  navigation target, AI tablet action clipping/document containment, Lead
  assignment tablet fact layout, Manager Checklist fact wrapping, and seven
  mobile changed-action families below the 44px target. The direct remediation
  contract was extended first; `node
  tests/ui-screenshot-audit-remediation-smoke.test.js` failed on the new mobile
  touch assertions before the scoped CSS change and passed afterward. The final
  mobile Browser audit measured 41 visible changed-action targets at a minimum
  44 × 44 CSS pixels.
- Former Issues first: all 30 route/viewport screenshots for the 10 baseline
  Issue routes were individually inspected and accepted after recapture where
  needed.
- Complete matrix: 86 routes × 3 viewports produced 258/258 accepted
  screenshots with 0 capture errors, 0 console warnings/errors, 0 route/role
  mismatches, 0 missing headings, 0 document overflow failures, and 0
  unintended nested overflow failures. All 27 role-family/viewport contact
  sheets were inspected; every metric-flagged or state-sensitive image was
  opened individually.
- Interaction/focus: 14/14 changed-control scenarios passed. Browser Tab and
  visible-focus checks passed for 56 changed-action targets, including AI entry
  from and return to the same Finding.
- Evidence: synchronized machine-readable results, 258 screenshots, 27 contact
  sheets, and a compact summary are under
  `/private/tmp/aviasurveil360-ui-audit-remediation-2026-07-19/`.
- Accessibility boundary: visible focus, browser keyboard reachability, text
  wrapping, status text, and mobile touch dimensions are `verified locally`.
  Screen-reader testing and automated contrast auditing are `not run`; no full
  accessibility-compliance claim is made.
- Cleanup: finalized the isolated Browser, stopped the static server, removed
  temporary QA files, and ran the required process search. It returned no
  task-owned server or browser-automation residue.

---

## Task 12: Synchronize Evidence, Plan State, And Durable Follow-Up

**Files:**

- Modify: both `docs/demo-evidence/UI_SCREEN_AUDIT_2026-07-19*` files
- Modify: both `docs/demo-evidence/BUILD_SUMMARY*` files
- Modify: `docs/exec-plans/index.md`
- Modify: `docs/exec-plans/tech-debt-tracker.md`
- Modify: this plan

**Interfaces:**

- Produces synchronized English/Turkish local evidence.
- Moves this plan to `ready-for-verification` only after all required checks pass.
- Closes the tracker note only after the full 258-view matrix is accepted.

- [x] **Step 1: Update both audit reports**

  Change the 76 Pass / 10 Issue summary only when the fresh matrix proves every former Issue passes. Preserve original findings in a `Remediation result` section. Record evidence folder, route/view count, screenshot count, capture errors, console issues, route mismatches, document/nested overflow, keyboard/focus checks, and accessibility checks not run.

- [x] **Step 2: Update both build summaries**

  Add a dated checkpoint using only literal `verified locally`, `not run`, and `demo-only` labels. Do not claim production readiness or full accessibility compliance.

- [x] **Step 3: Update this plan accurately**

  Mark only completed steps and add exact commands/results. If a route remains unresolved, leave its task unchecked and keep the plan active with one concrete next todo.

- [x] **Step 4: Synchronize the active index**

  If all gates pass, use status `ready-for-verification` and next todo:

  ```text
  Stakeholder review/sign-off of the 86-screen remediation evidence; after sign-off, move the plan to completed and return Modern Aviation SaaS Rollout to its next visual-system slice.
  ```

  Otherwise keep `active` and name exactly one next action.

- [x] **Step 5: Synchronize the tracker**

  Set the screenshot-audit remediation note to `note-closed` only when all 10 former Issues pass. Otherwise keep `note-open` and name the remaining route/viewport defect.

- [x] **Step 6: Run final documentation checks**

  ```bash
  git diff --check
  node tests/harness-docs-smoke.test.js
  node tests/demo-boundary-smoke.test.js
  rg -n "UI_SCREEN_AUDIT_2026-07-19|ui-screenshot-audit-remediation" docs/exec-plans docs/demo-evidence
  ```

  Expected: all commands pass and referenced files exist.

- [x] **Step 7: Report Git state without mutating it**

  ```bash
  git status --short
  git diff --stat
  ```

  Expected: only task-owned implementation/test/docs plus untouched unrelated user files. Do not stage, commit, or push without explicit authorization.

### Task 12 Execution Note — 2026-07-19

- Evidence synchronization: both audit reports now record the current 86 Pass /
  0 Issue result while preserving the original 76 Pass / 10 Issue table as the
  pre-remediation baseline. Both build summaries contain the same dated local,
  demo-only checkpoint and the same `not run` accessibility boundaries.
- Tracking: this active plan remains in place with every task complete. The
  active index is `ready-for-verification` with the required stakeholder
  sign-off next todo, and the durable screenshot-audit note is `note-closed`.
- Automated verification: syntax checks passed for all 11 `js/*.js` files; all
  39/39 direct-Node test files passed. The focused consolidated remediation
  smoke passed, and `stakeholder-readiness-regressions` passed 22/22.
- Documentation verification: `git diff --check`,
  `harness-docs-smoke`, and `demo-boundary-smoke` passed. The required `rg`
  reference scan returned 39 matching references and all referenced files
  exist.
- Evidence integrity: `capture-results.json` validated 258/258 accepted
  screenshots, 258 unique route/viewport keys, and 14/14 passing interactions;
  the evidence folders contain exactly 258 PNG screenshots and 27 JPG contact
  sheets.
- Git state: reported with `git status --short` and `git diff --stat` without
  staging or mutation. Task-owned implementation/test/docs changes are present;
  the pre-existing unrelated user files remain untouched. No
  branch, stage, commit, push, deployment, or GitHub write occurred.

---

## Verification Ladder

### Per-task gate

1. Write or extend the failing direct-Node test.
2. Run it and record the expected failure.
3. Make the smallest implementation change.
4. Run the focused test and `node --check` for touched JavaScript.
5. Run `tests/ui-screenshot-audit-remediation-smoke.test.js`.

### Pre-browser gate

```bash
for file in js/*.js; do node --check "$file" || exit 1; done
for test_file in tests/*.test.js; do node "$test_file" || exit 1; done
git diff --check
```

### Rendered gate

- 86 routes × 3 viewports = 258 accepted screenshots.
- 0 capture errors.
- 0 console warnings/errors.
- 0 role/route heading mismatches.
- 0 document horizontal overflow.
- 0 unintended critical nested overflow.
- Every interaction in Task 11 passes.

### Evidence gate

- English/Turkish audit reports agree.
- English/Turkish build summaries agree.
- Active index, plan checkbox state, and tracker agree.
- Claims remain local/demo-scoped.

## Risks And Mitigations

| Risk | Mitigation |
|---|---|
| Duplicate desktop/mobile controls fire twice or confuse navigation | Show exactly one representation at each breakpoint; keep equivalent names/data actions and keyboard-test the active representation. |
| Mobile cards omit fields hidden by the old table | Test identity, organization, status, owner/inspector, Due Date/Target, and action contracts. |
| Mobile passes but desktop split filters still clip | Base filters on pane width and inspect at both 1440px and 1024px. |
| Long text wrapping produces unusably tall cards | Structure facts and actions; never clip question/reference content. |
| AI entry implies a production AI service | Keep draft-review and demo-only copy visible; run boundary tests. |
| Executive empty state becomes misleading seeded work | Explain valid eligibility/source; use controlled QA state if a populated card needs inspection. |
| Shared 44px rules inflate desktop density | Scope them to `max-width: 640px` and touched controls. |
| Metrics pass while visuals are wrong | Inspect accepted screenshots and reject stale, wrong-state, blank, loading, or cropped files. |
| Broad CSS regresses 76 passing routes | Use route-scoped selectors, full automated suite, and full 86-route recapture. |
| Docs and plan state drift | Complete Task 12 in one final synchronized pass. |

## Completion Definition

The plan is complete only when:

- all 12 tasks are checked;
- all 10 former Issue routes pass at all three viewports;
- 258 screenshots are accepted and inspected;
- focused and full automated gates pass;
- changed controls pass interaction and visible-focus checks;
- bilingual evidence, active index, and tracker are synchronized;
- no task-owned browser/server process remains;
- no production or full-accessibility claim is made;
- remaining work, if any, is captured as one explicit next todo.

## Execution Prompt

```text
Implement `docs/exec-plans/active/2026-07-19-ui-screenshot-audit-remediation-plan.md` according to its defined scope and order in the current AviaSurveil360 working tree. Start by reading the repo-local `AGENTS.md`, the plan, `docs/demo-evidence/UI_SCREEN_AUDIT_2026-07-19.md`, and the source documents listed under Dependencies. Preserve the static HTML/CSS/Vanilla JavaScript architecture, browser-local demo state, working controls, role visibility, Finding/CAP/Evidence lifecycle, careful regulatory wording, and demo-only boundaries. Fix all 10 Issue routes, apply the shared next-action/touch/type recommendations, bump the asset token, run every JavaScript syntax check and every `tests/*.test.js`, then capture and inspect all 86 routes at 1440x1000, 1024x900, and 390x844 with Playwright. Update the bilingual audit/build evidence, plan checkbox state, active plan index, and technical-debt tracker with literal results. Clean up browser and local-server processes. Do not create or change branches, and do not stage, commit, push, deploy, or write to GitHub unless the user explicitly authorizes that exact action in the new thread.
```
