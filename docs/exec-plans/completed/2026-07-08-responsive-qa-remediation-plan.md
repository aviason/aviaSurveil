# Responsive QA Remediation Implementation Plan


**Goal:** Fix the visual, responsive, navigation, and workflow regressions found during the 2026-07-08 AviaSurveil360 full-site QA pass.

**Architecture:** Keep the current frontend-only static demo architecture. Consolidate inspector Finding and CAP work into one route, introduce shared responsive layout primitives for dense workbench screens, and verify every primary role surface across desktop, tablet, and mobile breakpoints before the next stakeholder review.

**Tech Stack:** HTML, CSS, Vanilla JavaScript, mock client-side state, existing Node smoke tests, local static server, Browser/visual QA.

**Execution Status 2026-07-08:** completed. The local static demo and deployed Vercel site both passed the full rendered route/viewport matrix with 258 checks, 0 failures, and 0 console warnings/errors. The production fresh-load check confirmed `20260708-responsive-qa` CSS/JS assets on `https://aviasurveil360.vercel.app`.

## Global Constraints

- Keep the demo frontend-only: no backend, database, API, authentication, real uploads, email, or framework migration.
- Preserve AviaSurveil360 as the canonical product name.
- Use English for code, implementation plans, verification notes, and committed docs.
- CAP acceptance is not finding closure; closure remains tied to accepted evidence, completed verification, or an authorized closure path.
- Auditee/service-provider views must only expose that organization's own records.
- Dropdown-looking controls must behave as real dropdowns or be removed/disabled.
- Wide tables may scroll inside their own wrappers, but the document/page must not create uncontrolled horizontal overflow.

---

## QA Scope Covered On 2026-07-08

The requested QA target is the whole clickable demo across the following viewport classes:

- `1920 x 1080` xlarge desktop
- `1536 x 864` large desktop
- `1366 x 768` medium desktop
- `1024 x 768` tablet landscape / narrow desktop
- `768 x 1024` tablet portrait
- `390 x 844` mobile

Primary surfaces to verify:

- Inspector: Dashboard, My Assignments, Findings, Calendar, audit execution, checklist runner, Reports, Messages.
- Lead Inspector: Assigned Audits, assignment workflow, Preliminary Reports, Final Reports, Calendar, Messages, Analytics, Settings.
- Department Manager: Dashboard, Planning, Checklist Approvals, Question Bank, Checklist Builder, Version History, Audit Reports, Audit Work Queue, Organizations, Findings, CAP Reviews, dashboards.
- Service Provider: Received Reports, My CAPs, Communications, Documents, Settings.
- Admin: Regulatory Library, Checklist Builder/Templates, Question Bank, Version History, Reports, Users, Settings, Organizations, Audit Log.
- GM, Finance, Executive Director: Planning and approval/report surfaces.

Browser automation note: the in-app Browser `domSnapshot()` API failed in this session with an internal snapshot runtime error, and two full-matrix locator loops timed out. The issue inventory below is therefore based on the supplied production screenshot, prior local browser evidence, focused local render inspection, route/source inspection, and existing smoke coverage. The remediation work must include a fresh, completed viewport matrix before sign-off.

## QA Findings Inventory

### QA-01: Deployed/cached inspector screen still shows legacy CAP Verification

**Evidence:** User screenshot from `aviasurveil360.vercel.app` shows Inspector sidebar with separate `Findings` and `CAP Verification`, while the intended workflow is one `Findings` workspace that includes CAP status and verification.

**User impact:** Inspectors see two overlapping work queues for the same operational object. It conflicts with the latest product decision that CAPs are findings with CAP lifecycle states.

**Likely files:** `js/app.js`, `js/views.js`, `index.html` asset version strings, Vercel deployment/cache behavior.

### QA-02: Legacy CAP Verification table has badge/text collisions

**Evidence:** In the supplied screenshot, `Level 3 (Observation)` badges collide with owner names such as `Sarah K.` and `David L.`. Status chips on the far right are partially constrained by the right rail.

**User impact:** Finding level, owner, and verification status are hard to read. This is especially risky because level and owner are key action fields.

**Likely files:** `css/styles.css`, legacy CAP review table markup in `js/views.js`.

### QA-03: Dense table plus right summary rail exceeds practical content width

**Evidence:** The CAP table, right `Quick Links`, and `Notes` rail create a cramped center table even on a large browser window. On narrower desktop/tablet, this pattern is likely to overflow or hide important columns.

**User impact:** Reviewers must scroll horizontally or lose context between finding, CAP owner, due date, and status.

**Likely files:** `css/styles.css`, `viewInspectorCapReviews()`, legacy `viewLeadCapTracking()`/CAP detail layouts.

### QA-04: Filter rows are not consistently responsive

**Evidence:** Prior screenshots and the current feedback show filter controls drifting outside their frame or taking too much horizontal space. This affects My Assignments, Findings/CAP, and assignment/filter workbenches.

**User impact:** At medium/tablet widths, users cannot quickly scan or use filters without layout noise.

**Likely files:** `css/styles.css`, filter-row markup in `js/views.js`.

### QA-05: Checklist runner and attached-file cells are fragile at narrow widths

**Evidence:** Earlier inspector screenshots showed attached-file/action cells pushed to the edge and comments/file columns crowding the checklist table.

**User impact:** Inspectors cannot confidently open evidence or confirm that a file is attached.

**Likely files:** `css/styles.css`, `viewInspectorAuditExecution()`.

### QA-06: Lead assignment question table can collapse text vertically

**Evidence:** Earlier QA screenshot showed checklist question text collapsing into letter-by-letter vertical layout when the assignment table is squeezed.

**User impact:** The assignment workflow becomes unusable on tablet/narrow desktop.

**Likely files:** `css/styles.css`, `viewLeadAssignmentQuestions()`.

### QA-07: Preliminary/final report step cards still have proportional imbalance risks

**Evidence:** Prior report screenshots showed next-step cards and action buttons stretched or misaligned in report flows.

**User impact:** Lead Inspectors and Department Managers cannot quickly understand the report approval path.

**Likely files:** `css/styles.css`, `viewLeadPreliminaryWorkflow()`, `viewLeadFinalReportPrepare()`, `viewLeadFinalReportReady()`.

### QA-08: Asset cache/versioning is not a reliable deployment signal

**Evidence:** Local server logs during QA initially served older `20260707-inspector-assignments` assets before the latest `20260708-findings-workspace` asset version appeared. The production screenshot also indicates a stale workflow may still be visible after push/deploy.

**User impact:** Stakeholders may review an old UI after a fix is pushed, creating false regressions.

**Likely files:** `index.html`, deployment handoff notes, release QA checklist.

### QA-09: Current automated smoke tests do not cover rendered responsive quality

**Evidence:** Existing Node smoke tests verify content and basic flows, but the Browser matrix could not complete in this session and there is no committed responsive QA harness that asserts no page-level overflow across roles/viewports.

**User impact:** Visual regressions can reappear even when Node tests pass.

**Likely files:** `tests/`, `docs/demo-evidence/`, optional QA script outside production app.

## File Structure

- Modify `index.html`
  - Bump cache version after UI fixes so deployed browsers fetch the new CSS/JS.
- Modify `css/styles.css`
  - Add shared responsive workbench, filter, table, rail, badge, and mobile stacking primitives.
  - Repair legacy CAP/table styles that can still be reached.
- Modify `js/app.js`
  - Enforce Inspector `Findings` as the single visible Findings/CAP entry.
  - Add compatibility redirects for old CAP Verification actions/routes if needed.
- Modify `js/views.js`
  - Consolidate Inspector Findings/CAP views.
  - Refactor dense table/right-rail layouts into responsive main/detail sections.
  - Ensure checklist, assignment, report, and manager review screens use shared wrappers.
- Modify `tests/inspector-nav-smoke.test.js`
  - Assert Inspector has `Findings` and does not expose `CAP Verification`.
  - Assert Non-Compliant checklist items appear in unified Findings.
- Modify `tests/service-provider-final-report-smoke.test.js`
  - Keep Service Provider CAP flow aligned with unified Findings language.
- Create or update `docs/demo-evidence/RESPONSIVE_QA_2026-07-08.md`
  - Record the completed viewport matrix, screenshots or screenshot paths, issues fixed, and residual risks.
- Update `docs/exec-plans/index.md`
  - Track this plan as active until fixes and verification are complete.
- Update `docs/exec-plans/tech-debt-tracker.md`
  - Track unresolved responsive QA risk until the matrix is completed.

## Task 1: Establish A Reproducible Responsive QA Baseline

**Files:**
- Create: `docs/demo-evidence/RESPONSIVE_QA_2026-07-08.md`
- Modify: `docs/exec-plans/active/2026-07-08-responsive-qa-remediation-plan.md`

**Interfaces:**
- Consumes: current static demo at `index.html`.
- Produces: a route/viewport matrix that later tasks must turn from failing to passing.

- [x] **Step 1: Start the static demo**

Run:

```bash
python3 -m http.server 4173
```

Expected: `http://127.0.0.1:4173/index.html` serves the demo.

- [x] **Step 2: Record matrix pages**

Create `docs/demo-evidence/RESPONSIVE_QA_2026-07-08.md` with this route list:

```markdown
# Responsive QA Evidence - 2026-07-08

## Viewports

- 1920 x 1080
- 1536 x 864
- 1366 x 768
- 1024 x 768
- 768 x 1024
- 390 x 844

## Routes

### Inspector
- Dashboard
- My Assignments
- Findings
- Calendar / Audit Work Queue
- SMS Oversight Audit execution
- Checklist Runner
- Reports
- Messages

### Lead Inspector
- Assigned Audits
- Assign Checklist Questions
- Preliminary Reports
- Final Reports
- Prepare Final Report
- Calendar
- Messages

### Department Manager
- Dashboard
- Planning
- Checklist Approvals
- Question Bank
- Checklist Builder
- Version History
- Audit Reports
- Audit Work Queue
- Organizations
- Findings
- CAP Reviews

### Service Provider
- Received Reports
- My CAPs
- Communications
- Documents
- Settings

### Admin / Governance
- Regulatory Library
- Templates
- Question Bank
- Version History
- Reports
- Users
- Settings
- Organizations
- Audit Log
- GM Planning
- Finance Planning
- Executive Director Planning
```

- [x] **Step 3: Capture each failing state**

For every failing viewport/route, add:

```markdown
## Finding QA-XX

- Viewport:
- Role / route:
- What the user sees:
- Evidence:
- Expected behavior:
- Candidate owner files:
```

- [x] **Step 4: Mark pass/fail per route**

Add a compact table:

```markdown
| Viewport | Role | Route | Result | Notes |
|---|---|---|---|---|
| 1920 x 1080 | Inspector | Findings | fail | Legacy CAP Verification or table collision visible. |
```

## Task 2: Remove Legacy Inspector CAP Verification From The Visible IA

**Files:**
- Modify: `js/app.js`
- Modify: `js/views.js`
- Test: `tests/inspector-nav-smoke.test.js`

**Interfaces:**
- Consumes: `NAV.inspector`, `go()`, `viewFindings()`, `viewInspectorCapReviews()`.
- Produces: Inspector sees one `Findings` nav item for findings and CAP lifecycle.

- [x] **Step 1: Write the nav assertion**

Update `tests/inspector-nav-smoke.test.js` to assert:

```js
assert(output.includes('Findings'), 'Inspector nav should show Findings.');
assert(!output.includes('CAP Verification'), 'Inspector nav should not show separate CAP Verification.');
assert(output.includes('CAP & Verification'), 'Unified Findings detail should include CAP & Verification.');
```

- [x] **Step 2: Run the targeted test**

Run:

```bash
node tests/inspector-nav-smoke.test.js
```

Expected before fix if stale code is present: failure on `CAP Verification`.

- [x] **Step 3: Route old CAP entry points to Findings**

In `js/app.js`, ensure any inspector attempt to open old CAP verification routes lands on:

```js
go('findings', { filter: 'open' });
```

Keep Lead Inspector and Department Manager CAP-specific routes separate only where they represent approval/closure ownership.

- [x] **Step 4: Verify**

Run:

```bash
node tests/inspector-nav-smoke.test.js
```

Expected: test passes and local Inspector sidebar has no separate `CAP Verification`.

## Task 3: Repair Dense Table And Right-Rail Layouts

**Files:**
- Modify: `css/styles.css`
- Modify: `js/views.js`

**Interfaces:**
- Consumes: Findings/CAP tables, legacy CAP review tables, right summary rails.
- Produces: No page-level horizontal overflow; tables scroll inside wrappers only.

- [x] **Step 1: Add shared layout wrappers**

In `css/styles.css`, add or normalize:

```css
.responsive-workbench {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 16px;
  min-width: 0;
}

@media (min-width: 1200px) {
  .responsive-workbench--with-rail {
    grid-template-columns: minmax(0, 1fr) minmax(280px, 340px);
    align-items: start;
  }
}

.responsive-table-shell {
  max-width: 100%;
  min-width: 0;
  overflow-x: auto;
}

.responsive-table-shell table {
  width: 100%;
  min-width: 860px;
}

.responsive-rail {
  min-width: 0;
}

@media (max-width: 1199px) {
  .responsive-rail {
    order: 2;
  }
}
```

- [x] **Step 2: Apply wrappers to CAP/Finding tables**

In `js/views.js`, wrap dense CAP/Finding tables with:

```html
<div class="responsive-table-shell">...</div>
```

Use `responsive-workbench responsive-workbench--with-rail` around main table plus summary/notes rail.

- [x] **Step 3: Prevent badge collisions**

Add:

```css
.badge,
.status-pill,
.finding-level-pill,
.cap-level-badge {
  max-width: 100%;
  white-space: normal;
  line-height: 1.25;
}

td .badge,
td .status-pill,
td .finding-level-pill,
td .cap-level-badge {
  display: inline-flex;
  width: max-content;
  max-width: 100%;
}
```

- [x] **Step 4: Verify target regression**

At `1920 x 1080`, Inspector Findings/CAP route must not show:

- Level badge colliding with owner.
- Status chip clipped by right rail.
- Document-level horizontal scroll.

## Task 4: Normalize Filter Rows Across Workbenches

**Files:**
- Modify: `css/styles.css`
- Modify: `js/views.js`

**Interfaces:**
- Consumes: My Assignments, Findings, CAP review, lead assignment filters.
- Produces: Filters wrap cleanly or collapse to a compact secondary row without leaving the card.

- [x] **Step 1: Add shared filter grid**

Add:

```css
.responsive-filter-row {
  display: grid;
  grid-template-columns: minmax(220px, 1.5fr) repeat(auto-fit, minmax(160px, 1fr));
  gap: 12px;
  align-items: end;
  min-width: 0;
}

@media (max-width: 760px) {
  .responsive-filter-row {
    grid-template-columns: 1fr;
  }
}
```

- [x] **Step 2: Replace one-off filter rows**

Replace custom filter wrappers on high-risk workbenches with `responsive-filter-row`.

- [x] **Step 3: Verify**

At `1366 x 768`, `1024 x 768`, and `390 x 844`, filter controls must stay inside the card and remain tappable.

## Task 5: Repair Checklist And Assignment Tables For Tablet/Mobile

**Files:**
- Modify: `css/styles.css`
- Modify: `js/views.js`

**Interfaces:**
- Consumes: `viewInspectorAuditExecution()`, `viewLeadAssignmentQuestions()`.
- Produces: No vertical letter collapse; evidence/file controls remain visible.

- [x] **Step 1: Set minimum table widths inside scroll shells**

Use:

```css
.inspection-table,
.assignment-question-table {
  min-width: 920px;
}
```

Only the table wrapper should scroll, not the page.

- [x] **Step 2: Prevent vertical word collapse**

Add:

```css
.assignment-question-table td,
.inspection-table td {
  word-break: normal;
  overflow-wrap: anywhere;
}

.assignment-question-table .question-title,
.inspection-table .checklist-item-title {
  min-width: 220px;
}
```

- [x] **Step 3: Verify**

At `768 x 1024` and `390 x 844`, assigned question text must not render as one character per line.

## Task 6: Rebalance Preliminary And Final Report Screens

**Files:**
- Modify: `css/styles.css`
- Modify: `js/views.js`

**Interfaces:**
- Consumes: Lead Inspector preliminary report workflow, final report list, final report preparation, department review.
- Produces: Report stages are proportional, readable, and action buttons are visible at medium/tablet widths.

- [x] **Step 1: Collapse report step cards below desktop**

Use a responsive grid:

```css
.report-step-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 14px;
}
```

- [x] **Step 2: Keep action bars wrapping**

Add:

```css
.report-action-bar,
.final-report-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.report-action-bar .btn,
.final-report-actions .btn {
  white-space: normal;
}
```

- [x] **Step 3: Verify**

At `1024 x 768` and `768 x 1024`, preliminary/final report screens must not show tall empty columns, clipped submit buttons, or overlapped next-step copy.

## Task 7: Add A Rendered QA Gate Before Future Pushes

**Files:**
- Create: `docs/demo-evidence/RESPONSIVE_QA_2026-07-08.md`
- Modify: `tests/inspector-nav-smoke.test.js`
- Modify: `tests/service-provider-final-report-smoke.test.js`

**Interfaces:**
- Consumes: existing static demo and smoke tests.
- Produces: repeatable evidence that Node tests and browser QA both ran.

- [x] **Step 1: Run syntax checks**

Run:

```bash
node --check js/app.js
node --check js/views.js
```

Expected: no output and exit code `0`.

- [x] **Step 2: Run all Node smoke tests**

Run:

```bash
node -e "const fs=require('fs'); const path=require('path'); const {spawnSync}=require('child_process'); const tests=fs.readdirSync('tests').filter(f=>f.endsWith('.test.js')).sort(); for (const f of tests) { const file=path.join('tests', f); const r=spawnSync(process.execPath, [file], {stdio:'inherit'}); if (r.status !== 0) process.exit(r.status || 1); }"
```

Expected: every smoke test prints `ok`.

- [x] **Step 3: Run browser QA matrix**

For each viewport listed above, verify:

```js
document.documentElement.scrollWidth <= document.documentElement.clientWidth + 4
```

Exceptions are allowed only for intentional inner table wrappers such as `.responsive-table-shell`.

- [x] **Step 4: Record evidence**

Update `docs/demo-evidence/RESPONSIVE_QA_2026-07-08.md` with:

- URL tested.
- Viewports tested.
- Routes tested.
- Any remaining findings.
- Screenshot paths when screenshots are captured.
- Console errors/warnings.

## Task 8: Refresh Asset Version And Verify Deployment

**Files:**
- Modify: `index.html`
- Modify: `docs/demo-evidence/RESPONSIVE_QA_2026-07-08.md`

**Interfaces:**
- Consumes: completed responsive fixes.
- Produces: Vercel/browser loads the updated static assets.

- [x] **Step 1: Bump asset query version**

In `index.html`, replace the current asset query string with a new token:

```html
?v=20260708-responsive-qa
```

Use the same token for CSS and all JS script tags.

- [x] **Step 2: Verify local fresh load**

Open:

```text
http://127.0.0.1:4173/index.html
```

Expected:

- Inspector sidebar shows `Findings`, not `CAP Verification`.
- Findings screen shows CAP lifecycle states inside the same workspace.
- No stale 20260707 assets are requested.

- [x] **Step 3: Verify deployed fresh load after push**

Open:

```text
https://aviasurveil360.vercel.app
```

Expected:

- Same Inspector IA as local.
- No old CAP Verification screen in Inspector.
- Browser dev asset URLs use the new version token.

## Verification

Minimum completion gate:

```bash
git diff --check
node --check js/app.js
node --check js/views.js
node -e "const fs=require('fs'); const path=require('path'); const {spawnSync}=require('child_process'); const tests=fs.readdirSync('tests').filter(f=>f.endsWith('.test.js')).sort(); for (const f of tests) { const file=path.join('tests', f); const r=spawnSync(process.execPath, [file], {stdio:'inherit'}); if (r.status !== 0) process.exit(r.status || 1); }"
```

Rendered QA gate:

- Local static app loads with no console errors.
- Every route in the QA scope is checked at `1920 x 1080`, `1536 x 864`, `1366 x 768`, `1024 x 768`, `768 x 1024`, and `390 x 844`.
- No page-level horizontal overflow, except intentional inner table scrolling.
- No vertical letter collapse.
- No badge/owner/status text overlap.
- Inspector IA shows one `Findings` workspace for findings and CAP lifecycle.
- Deployed Vercel URL is checked after push.

## Risks

- The current repo does not have a committed Playwright dependency, so full rendered QA depends on the Codex Browser runtime or manual browser inspection.
- Some old CAP/lead routes are intentionally retained for role-specific approval and closure flows; redirects must not remove Department Manager or Lead Inspector ownership screens.
- Existing stakeholder screenshots may reference cached production assets; deployment verification must distinguish stale browser cache from real code regression.

## Dependencies

- Local static server availability.
- Browser runtime or equivalent manual browser access for viewport QA.
- Vercel deployment completing after push.

## Ownership Boundaries

- This plan fixes frontend demo rendering, navigation, and responsive behavior only.
- It does not add backend workflow, real authentication, real file upload, real notification delivery, real reporting engine, or production-grade audit storage.
- It does not change regulatory meaning or introduce automatic enforcement.

## Explicit Out Of Scope

- Framework migration.
- Real Playwright package installation unless separately approved.
- Production deployment architecture.
- Real database/API/auth/upload/email services.
- Legal/regulatory claims beyond the configured demo language.

## Execution Prompt

```text
Implement docs/exec-plans/active/2026-07-08-responsive-qa-remediation-plan.md task by task. Keep the AviaSurveil360 demo frontend-only. First reproduce and record the responsive QA baseline, then remove stale Inspector CAP Verification IA, repair dense table/right-rail/filter/checklist/assignment/report layouts, update smoke tests, bump asset versions, verify locally across 1920x1080, 1536x864, 1366x768, 1024x768, 768x1024, and 390x844, then verify the deployed Vercel URL after push. Run git diff --check, node --check js/app.js, node --check js/views.js, and all tests/*.test.js before final handoff.
```
