# Stakeholder Feedback Remediation Implementation Plan


**Status:** ready-for-verification

**Goal:** Resolve the nine 2026-07-15 stakeholder feedback items by simplifying Inspector and report screens, seeding decision-ready GM/ED/Finance records, and changing the planning approval order to `Department Manager -> Finance Review -> GM Review -> Executive Director` with Finance returns routed to Department Manager.

**Architecture:** Keep the existing frontend-only HTML, CSS, and Vanilla JavaScript application. Make focused changes to the shared seed/migration state, approval primitive, role renderers, event handlers, targeted Node smoke tests, responsive CSS, and synchronized product/evidence documentation. Preserve exact record identity and all Finding/CAP/Evidence, Service Provider privacy, report-authority, and post-ED GM-release boundaries.

**Tech Stack:** Static HTML, CSS, Vanilla JavaScript, browser-local demo state, Node.js `assert`/`node:test` smoke tests, in-app Browser rendered QA.

## Global Constraints

- Work from `/Users/marlonjd/Developer/web/aviaSurveil360` on the current branch.
- Preserve all unrelated dirty-worktree changes; do not overwrite or revert user work.
- Do not create or switch branches, commit, stage, push, deploy, open a PR, or post GitHub comments unless the user separately authorizes that exact action.
- Keep the demo frontend-only: no backend, database, API, real authentication, real authorization enforcement, real upload/storage, real notification service, real e-signature, production reporting engine, immutable audit log, or framework migration.
- Use `AviaSurveil360` as the product name and keep English source code/comments/docs canonical.
- Update a matching `.turkce.md` companion whenever a canonical stakeholder-facing English document changes.
- CAP acceptance is not Finding closure. Report approval must not close Findings or bypass required CAP, Evidence, verification, or authorized closure work.
- GM remains an intermediate Final Report reviewer. Executive Director alone may issue, add a demo mock approval mark, and lock an eligible Final Report.
- Enforcement referral remains recommendation-only and must not execute a sanction, affect a certificate, or close an Audit, Finding, CAP, or Evidence requirement.
- Service Provider screens remain scoped to their own organization and must not expose Internal CAA Notes, other organizations, workload, internal risk, or enforcement deliberations.
- Keep evidence language literal: `demo-only`, `verified locally`, `not run`, and `production-readiness not claimed`.

---

## Objective

Implement the approved design in
`docs/product-specs/workflows/2026-07-15-stakeholder-feedback-remediation-design.md`
and provide one coherent stakeholder-ready demo reset state.

The implemented result must cover all nine source notes:

1. Remove the redundant `Next inspection` dossier from My Assignments.
2. Keep the Lead Inspector Preliminary Report content inside its frame.
3. Remove CAP/lifecycle status from Preliminary Report Finding review content.
4. Stop attachment filename/description text from overlapping.
5. Make Final Report summary and key-finding metrics compact.
6. Populate General Manager Report Approvals with a decision-ready sample.
7. Populate Executive Director Final Reports with a visible approval screen and choices.
8. Change Finance planning order to Department Manager, Finance, GM, Executive Director.
9. Populate Finance Review with sample plan/budget data on initial load.

## Scope

### In scope

- Inspector My Assignments render cleanup.
- Lead Inspector Preliminary Report inspection/finding semantics and responsive layout.
- Lead Inspector attachment table containment.
- Lead Inspector Final Report overview metric density.
- Canonical GM-pending and ED-pending Final Report sample artifacts/projections.
- GM and ED default selection and decision panels.
- Planning approval order, ownership, return targets, messages, notifications, and labels.
- Browser-state migration to the new seed/order contract.
- Focused/full automated verification and four-viewport rendered QA.
- Canonical English/Turkish product/evidence docs and active plan tracking.

### Explicitly out of scope

- Any production service or integration.
- A general approval-engine rewrite.
- A broad application redesign or framework migration.
- New legal, regulatory, enforcement, certificate, retention, signature, or audit-log policy.
- Changes to Service Provider access beyond regression protection.
- Branch/commit/push/PR/deployment activity.

## Assumptions

- The supplied screenshots represent stakeholder acceptance feedback, not a request to reproduce image styling pixel-for-pixel.
- Removing Preliminary Finding status means removing Finding CAP/lifecycle status from the Lead Inspector Preliminary workflow; report-level Draft/Review status remains visible.
- `Return for Revision` from Finance goes to Department Manager, as confirmed by Furkan Ozdemir on 2026-07-15.
- GM plan return also goes to Department Manager. A corrected plan must pass Finance again before GM can forward it to Executive Director.
- Executive Director report decisions remain the existing four choices: Approve, Return, Reject, and recommendation-only Enforcement Review referral.
- The existing post-ED `GM Release to Department` preparation boundary remains unchanged.
- Representative GM and ED samples may use distinct existing internal audits/organizations, provided Service Provider selectors do not expose those records.

## File Responsibility Map

| File | Responsibility in this plan |
|---|---|
| `js/data.js` | Seed GM/ED report examples, reorder the planning chain, bump/migrate demo state, and set decision-ready defaults. |
| `js/approval.js` | Planning-stage next-action, status, forward, and return behavior for the new Finance-before-GM order. |
| `js/planning.js` | Finance decision result messages and Finance-to-GM / Finance-to-Department ownership contract. |
| `js/manager-workspaces.js` | Existing GM/ED exact-artifact projections and decisions; modify only if sample eligibility exposes a contract gap. |
| `js/views.js` | Remove the assignment dossier, simplify Preliminary findings, contain attachments, compact Final metrics, and update planning/Finance copy. |
| `js/app.js` | Update action/log/notification copy only where the new owner sequence requires it. |
| `css/styles.css` | Remove dead dossier styles and add responsive containment/compact metric rules. |
| `tests/stakeholder-readiness-regressions.test.js` | Add direct regression coverage for all nine feedback items and migration/order invariants. |
| `tests/premium-ui-remediation-smoke.test.js` | Replace the obsolete assertion that requires the removed assignment dossier. |
| `tests/lead-inspector-nav-smoke.test.js` | Update Preliminary Findings/attachment expectations. |
| `tests/general-manager-workspace-smoke.test.js` | Prove default GM sample visibility and exact decisions. |
| `tests/executive-director-workspace-smoke.test.js` | Prove default ED pending selection and visible choices. |
| `tests/finance-review-workspace-smoke.test.js` | Prove the default Finance sample and new advance/return owners. |
| `tests/planning-workspace-smoke.test.js` | Prove the new rail/order and complete approval traversal. |
| `tests/planning-render-smoke.test.js` | Update role-aware planning render expectations and Finance return copy. |
| `tests/planning-release-smoke.test.js` | Traverse Finance -> GM -> ED before the unchanged GM release/preparation flow. |
| `tests/governance-render-smoke.test.js` | Update the shared governance traversal to the new planning order. |
| `docs/product-specs/ux-plan/NAVIGATION_AND_INFORMATION_ARCHITECTURE.md` + `.turkce.md` | State Finance placement, decisions, and owners. |
| `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md` + `.turkce.md` | State the planning authority/return sequence without changing report authority. |
| `docs/demo-evidence/BUILD_SUMMARY.md` + `.turkce.md` | Record only freshly verified implementation and rendered evidence. |
| `MANIFEST.md` | Update only if the tracked file inventory changes; no update is needed when only existing tests are extended. |
| `docs/exec-plans/index.md` | Track this plan's status and one next concrete todo. |
| `docs/exec-plans/tech-debt-tracker.md` | Add an entry only if verification leaves a durable blocker, accepted risk, or missing evidence. |

## Phases

| Phase | Deliverable |
|---|---|
| 1 | Inspector and Lead Preliminary UI regressions fixed with tests first. |
| 2 | Attachment and Final Report density regressions fixed with rendered containment hooks. |
| 3 | GM and ED decision-ready report samples integrated with exact canonical IDs. |
| 4 | Finance-before-GM planning contract implemented and migrated. |
| 5 | Canonical English/Turkish truth synchronized. |
| 6 | Full automated and four-viewport rendered verification completed. |

---

### Task 1: Remove The Redundant My Assignments Dossier

**Files:**

- Modify: `tests/stakeholder-readiness-regressions.test.js`
- Modify: `tests/premium-ui-remediation-smoke.test.js`
- Modify: `js/views.js` (`inspectorNextAssignmentDossier`, `viewInspectorAssignments`)
- Modify: `css/styles.css` (`.inspector-next-dossier*` rules)

**Interfaces:**

- Consumes: `viewInspectorAssignments()`, `inspectorAssignmentKpis()`, `inspectorAssignmentFilterBar()`, `inspectorAssignmentTable()`.
- Produces: My Assignments page order `header -> KPIs -> filters -> table -> optional In Progress detail` with no `inspector-next-dossier` markup.

- [x] **Step 1: Add the failing render regression**

Append this test to `tests/stakeholder-readiness-regressions.test.js`:

```js
test('My Assignments starts with operational KPIs and no Next inspection dossier', () => {
  const context = buildContext();
  context.state = context.freshState();
  context.state.role = 'inspector';
  context.state.view = 'inspector-assignments';
  context.state.params = {};

  const html = context.viewInspectorAssignments();
  assert.match(html, /inspector-assignment-kpis/);
  assert.match(html, /inspector-assignment-table/);
  assert.doesNotMatch(html, /inspector-next-dossier|Next inspection|Open audit dossier/);
  assert.ok(html.indexOf('inspector-assignment-kpis') < html.indexOf('inspector-assignment-table'));
});
```

In `tests/premium-ui-remediation-smoke.test.js`, replace the obsolete dossier assertions with:

```js
assert.doesNotMatch(html, /inspector-next-dossier|Next inspection/);
assert.match(html, /inspector-assignment-kpis/);
assert.match(html, /inspector-assignment-table/);
```

- [x] **Step 2: Run the focused tests and confirm RED**

Run:

```bash
node --test tests/stakeholder-readiness-regressions.test.js
node tests/premium-ui-remediation-smoke.test.js
```

Expected before implementation: the new regression fails because
`viewInspectorAssignments()` still renders `inspectorNextAssignmentDossier()`.
The existing premium test may also fail until its obsolete expectation is
replaced.

- [x] **Step 3: Remove only the duplicate dossier surface**

Change `viewInspectorAssignments()` to:

```js
function viewInspectorAssignments() {
  var ui = inspectorAssignmentsUiState();
  if (state.params && state.params.filter === 'checklists' && ui.status === 'all') ui.status = 'in-progress';
  var rows = inspectorAssignmentFilteredRows(ui);
  return '<div class="inspector-assignment-page">' +
    inspectorAssignmentHeader(ui) +
    inspectorAssignmentKpis(ui) +
    inspectorAssignmentFilterBar(ui) +
    inspectorAssignmentTable(rows, ui) +
    (ui.status === 'in-progress' ? inspectorAssignmentDetailPanel(rows, ui) : '') +
  '</div>';
}
```

Delete `inspectorNextAssignmentDossier()` and delete CSS selectors used only by
`.inspector-next-dossier`, while preserving shared assignment donut/detail
styles used by the In Progress detail view.

- [x] **Step 4: Verify GREEN and neighboring Inspector behavior**

Run:

```bash
node --test tests/stakeholder-readiness-regressions.test.js
node tests/premium-ui-remediation-smoke.test.js
node tests/inspector-nav-smoke.test.js
```

Expected: all commands pass; assignment table actions and the In Progress
detail remain covered.

- [x] **Step 5: Review the focused diff without staging or committing**

Run:

```bash
git diff -- js/views.js css/styles.css tests/stakeholder-readiness-regressions.test.js tests/premium-ui-remediation-smoke.test.js
git diff --check
```

Expected: only the redundant dossier and its exclusive styles/expectations are
removed; no branch, stage, or commit action occurs.

---

### Task 2: Simplify And Contain Preliminary Report Findings

**Files:**

- Modify: `tests/stakeholder-readiness-regressions.test.js`
- Modify: `tests/lead-inspector-nav-smoke.test.js`
- Modify: `js/views.js` (`leadPreliminaryFindingsSidePanel`, `leadPreliminaryInspectionStep`, `leadPreliminaryReportBody`)
- Modify: `css/styles.css` (`.prelim-workflow-grid`, `.prelim-finding-item`, `.prelim-area-table`)

**Interfaces:**

- Consumes: `leadPreliminaryWorkflowFindings(row)`, selected Finding IDs, Finding severity/area/Due Date.
- Produces: Preliminary `Inspection & Findings` UI with no CAP/lifecycle-status column or status badges, contained at desktop/mobile widths.

- [x] **Step 1: Add the failing semantic regression**

Append:

```js
test('Preliminary Inspection and Findings omits CAP lifecycle status', () => {
  const context = buildContext();
  context.state = context.freshState();
  context.state.role = 'leadInspector';
  context.state.leadPreliminaryReportsUi.mode = 'workflow';
  context.state.leadPreliminaryReportsUi.selectedReportId = 'PR-2026-018';
  context.state.preliminaryReportDrafts['PR-2026-018'].step = 'inspection';

  const html = context.viewLeadPreliminaryWorkflow();
  assert.match(html, /Inspection &amp; Findings|Inspection & Findings/);
  assert.match(html, /Findings Review/);
  assert.match(html, /<th>Finding<\/th><th>Level<\/th><th>Due Date<\/th><th>Action<\/th>/);
  assert.doesNotMatch(html, /<th>Status<\/th>|CAP Submitted|CAP Accepted|Waiting for CAP/);
  assert.equal((html.match(/<h2>Inspection Overview<\/h2>/g) || []).length, 1);
});
```

Update the Preliminary assertions in `tests/lead-inspector-nav-smoke.test.js`
so the inspection step requires `Finding`, `Level`, `Due Date`, and `Review`,
and explicitly rejects `Status` plus the CAP lifecycle labels.

- [x] **Step 2: Run focused tests and confirm RED**

Run:

```bash
node --test tests/stakeholder-readiness-regressions.test.js
node tests/lead-inspector-nav-smoke.test.js
```

Expected before implementation: failure on the existing `Status` header,
side-panel status `<i>`, and duplicate `Inspection Overview` heading.

- [x] **Step 3: Remove CAP/lifecycle status from Preliminary rendering**

In `leadPreliminaryInspectionStep()` build rows as:

```js
var findingRows = findings.map(function (finding) {
  return '<tr><td>' + esc(finding.id) + '</td><td>' + esc(finding.levelLabel) +
    '</td><td>' + esc(fmtDate(finding.dueDate)) +
    '</td><td><button class="btn btn--sm" data-act="preliminary-report-view-finding" data-id="' +
    esc(finding.id) + '">Review</button></td></tr>';
}).join('');
```

Render this header exactly:

```html
<thead><tr><th>Finding</th><th>Level</th><th>Due Date</th><th>Action</th></tr></thead>
```

Delete the side-panel `<i>` status cell, remove the duplicate
`<h2>Inspection Overview</h2>`, and remove `finding.status` from the generated
Preliminary report body line. Keep the report header's Draft/Review status.

- [x] **Step 4: Rebalance the findings panel grid**

Use a contained grid contract such as:

```css
.prelim-workflow-grid {
  grid-template-columns: minmax(0, 1.45fr) minmax(330px, .85fr);
}

.prelim-workflow-card,
.prelim-findings-panel {
  min-width: 0;
}

.prelim-finding-item {
  grid-template-columns: 28px 44px minmax(0, 1fr) minmax(92px, auto) 58px;
}

.prelim-finding-copy,
.prelim-finding-item > em {
  min-width: 0;
  overflow-wrap: anywhere;
}
```

Keep the existing mobile breakpoint behavior, but verify the panel stacks and
does not require page-level horizontal scrolling.

- [x] **Step 5: Verify GREEN**

Run:

```bash
node --test tests/stakeholder-readiness-regressions.test.js
node tests/lead-inspector-nav-smoke.test.js
node tests/lead-inspector-workspace-smoke.test.js
node tests/report-approval-smoke.test.js
```

Expected: all pass; report-level Draft/Department Review state remains visible,
while Finding CAP/lifecycle state is absent from the Preliminary workflow.

---

### Task 3: Fix Attachment Overlap And Compact Final Report Metrics

**Files:**

- Modify: `tests/stakeholder-readiness-regressions.test.js`
- Modify: `tests/lead-inspector-nav-smoke.test.js`
- Modify: `js/views.js` (`leadPreliminaryAttachmentsStep`, `viewLeadFinalReportReady` overview markup)
- Modify: `css/styles.css` (`.prelim-attachment-table*`, `.final-overview-org-summary*`, `.final-overview-finding-*`)

**Interfaces:**

- Consumes: mock filename-only attachment rows and existing Final Report counts/actions.
- Produces: non-overlapping attachment table and compact Final Report overview summaries with unchanged data/actions.

- [x] **Step 1: Add failing markup/style contract tests**

Append:

```js
test('Preliminary attachments and Final Report metrics expose compact containment hooks', () => {
  const context = buildContext();
  context.state = context.freshState();
  context.state.role = 'leadInspector';
  context.state.leadPreliminaryReportsUi.mode = 'workflow';
  context.state.leadPreliminaryReportsUi.selectedReportId = 'PR-2026-018';
  context.state.preliminaryReportDrafts['PR-2026-018'].step = 'attachments';
  const attachmentHtml = context.viewLeadPreliminaryWorkflow();
  assert.match(attachmentHtml, /prelim-attachment-table-wrap/);

  context.state.params = { filter: 'final', reportId: 'FR-2026-018', finalReportId: 'FR-2026-018' };
  const finalHtml = context.viewLeadFinalReportReady();
  assert.match(finalHtml, /final-overview-org-summary is-compact/);
  assert.match(finalHtml, /final-overview-finding-grid is-compact/);

  const css = fs.readFileSync(path.join(root, 'css/styles.css'), 'utf8');
  assert.match(css, /\.prelim-attachment-table-wrap\s*\{[^}]*overflow-x:\s*auto/s);
  assert.match(css, /\.prelim-attachment-table\s*\{[^}]*table-layout:\s*fixed/s);
  assert.match(css, /\.final-overview-org-summary\.is-compact/s);
  assert.match(css, /\.final-overview-finding-grid\.is-compact/s);
});
```

- [x] **Step 2: Run and confirm RED**

Run:

```bash
node --test tests/stakeholder-readiness-regressions.test.js
node tests/lead-inspector-nav-smoke.test.js
```

Expected before implementation: missing attachment wrapper and compact
modifier classes/styles.

- [x] **Step 3: Add the attachment containment wrapper**

Wrap the table without changing its actions:

```js
'<div class="prelim-attachment-table-wrap"><table class="prelim-attachment-table">' +
  '<thead>...</thead><tbody>' + attachmentRows + '</tbody>' +
'</table></div>'
```

Implement explicit table behavior:

```css
.prelim-attachment-table-wrap {
  max-width: 100%;
  overflow-x: auto;
}

.prelim-attachment-table {
  width: 100%;
  table-layout: fixed;
}

.prelim-attachment-table th:nth-child(1) { width: 25%; }
.prelim-attachment-table th:nth-child(2) { width: 22%; }
.prelim-attachment-table th:nth-child(3) { width: 16%; }
.prelim-attachment-table th:nth-child(4) { width: 18%; }
.prelim-attachment-table th:nth-child(5) { width: 12%; }
.prelim-attachment-table th:nth-child(6) { width: 7%; }

.prelim-attachment-table td {
  min-width: 0;
  overflow-wrap: anywhere;
  word-break: break-word;
  vertical-align: top;
}
```

- [x] **Step 4: Add compact Final Report modifiers and styling**

Change the two wrappers to include `is-compact`, then implement a restrained
desktop density contract:

```css
.final-overview-org-summary.is-compact {
  gap: 12px;
  padding: 12px 16px;
}

.final-overview-org-summary.is-compact > div {
  min-height: 64px;
  padding: 8px 12px;
}

.final-overview-finding-grid.is-compact {
  gap: 0;
}

.final-overview-finding-grid.is-compact .final-overview-finding-card {
  min-height: 150px;
  padding: 16px 14px;
}

.final-overview-finding-grid.is-compact .final-overview-finding-card strong {
  font-size: 24px;
  line-height: 1.1;
}
```

Adjust values after rendered comparison if the existing selector specificity
requires it; keep text readable and actions at least 40px high.

- [x] **Step 5: Verify GREEN and relevant responsive tests**

Run:

```bash
node --test tests/stakeholder-readiness-regressions.test.js
node tests/lead-inspector-nav-smoke.test.js
node tests/premium-ui-remediation-smoke.test.js
node tests/manager-workspace-responsive-smoke.test.js
git diff --check
```

Expected: all pass; no data/action is removed from Final Report summaries.

---

### Task 4: Seed Decision-Ready GM And ED Final Reports

**Files:**

- Modify: `tests/stakeholder-readiness-regressions.test.js`
- Modify: `tests/general-manager-workspace-smoke.test.js`
- Modify: `tests/executive-director-workspace-smoke.test.js`
- Modify: `js/data.js` (`SEED_AUDIT_REPORTS`, `SEED_MANAGER_REPORTS`, `freshState`, `mergeDemoState`, canonical report normalization)
- Modify: `js/views.js` (`viewGeneralManagerReportApprovals`, `viewExecutiveFinalReportsWorkspace` only if selection fallback requires it)

**Interfaces:**

- Consumes: `reportArtifactById`, `reportReadModels`, `generalManagerProjection`, `executiveFinalReportProjection`, existing GM/ED decision helpers.
- Produces: distinct exact artifacts `FR-2026-021` at GM review and `FR-2026-022` at ED review, with decision-ready default selections.

- [x] **Step 1: Add failing seed/projection regressions**

Append:

```js
test('fresh demo state contains distinct decision-ready GM and ED Final Reports', () => {
  const context = buildContext();
  const state = context.freshState();
  const gmReport = context.reportArtifactById('FR-2026-021', state);
  const edReport = context.reportArtifactById('FR-2026-022', state);

  assert.ok(gmReport);
  assert.equal(gmReport.reportType, 'Final Report');
  assert.equal(gmReport.status, 'submitted_to_gm');
  assert.equal(gmReport.ownerRole, 'gm');
  assert.equal(gmReport.locked, false);

  assert.ok(edReport);
  assert.equal(edReport.reportType, 'Final Report');
  assert.equal(edReport.status, 'submitted_to_executive');
  assert.equal(edReport.ownerRole, 'executiveDirector');
  assert.equal(edReport.locked, false);
  assert.notEqual(gmReport.id, edReport.id);

  assert.ok(context.generalManagerProjection(state).approvalRows.some((row) => row.id === gmReport.id));
  assert.ok(context.executiveFinalReportProjection(state, { status: 'pending' }).rows.some((row) => row.id === edReport.id));
});
```

Append a render regression:

```js
test('GM and ED default workspaces show working report decisions', () => {
  const context = buildContext();
  context.state = context.freshState();
  context.state.role = 'gm';
  let html = context.viewGeneralManagerReportApprovals();
  assert.match(html, /FR-2026-021/);
  assert.match(html, /Open Report/);
  assert.match(html, /Return Report/);
  assert.match(html, /Forward to Executive Director/);

  context.state.role = 'executiveDirector';
  html = context.viewExecutiveFinalReportsWorkspace();
  assert.match(html, /FR-2026-022/);
  assert.match(html, /Approve Report/);
  assert.match(html, /Return for Revision/);
  assert.match(html, /Reject Report/);
  assert.match(html, /Refer for Enforcement Review/);
});
```

- [x] **Step 2: Run and confirm RED**

Run:

```bash
node --test tests/stakeholder-readiness-regressions.test.js
node tests/general-manager-workspace-smoke.test.js
node tests/executive-director-workspace-smoke.test.js
```

Expected before implementation: missing `FR-2026-021`/`FR-2026-022`; GM queue
empty; ED defaults to an issued or manager-stage report.

- [x] **Step 3: Add exact canonical seed artifacts and projections**

Add two complete objects to `SEED_AUDIT_REPORTS` and matching read projections
to `SEED_MANAGER_REPORTS`. Use these required fields and internally coherent
existing audits:

```js
{
  id: 'FR-2026-021',
  approvalPackageId: 'FR-2026-021',
  auditId: 'AUD-2026-002',
  organizationId: 'ORG-SKY',
  organization: 'SkyCargo Air',
  title: 'Q1 Ramp Inspection Final Report - Amendment Review',
  reportType: 'Final Report',
  version: '1.1',
  leadInspector: 'Caner Yildiz',
  submittedAt: '2026-07-14 09:30',
  status: 'submitted_to_gm',
  ownerRole: 'gm',
  dueDate: '2026-07-21',
  issued: false,
  locked: false,
  finalLocked: false,
  mockApprovalSignature: null,
  enforcementReferral: null,
  capRequired: true,
  attachments: ['Ramp_Final_Report_Amendment_Summary.pdf'],
  summary: 'Amended Final Report awaiting General Manager intermediate review.',
  history: [{ at: '2026-07-14 09:30', actor: 'Mehmet Kaya', action: 'Department Manager approved and forwarded to General Manager' }]
}
```

```js
{
  id: 'FR-2026-022',
  approvalPackageId: 'FR-2026-022',
  auditId: 'AUD-2026-003',
  organizationId: 'ORG-BLU',
  organization: 'BlueWing Aviation',
  title: 'Airworthiness Surveillance Final Report - Amendment Review',
  reportType: 'Final Report',
  version: '1.1',
  leadInspector: 'Aylin Sezer',
  submittedAt: '2026-07-14 11:15',
  status: 'submitted_to_executive',
  ownerRole: 'executiveDirector',
  dueDate: '2026-07-22',
  issued: false,
  locked: false,
  finalLocked: false,
  mockApprovalSignature: null,
  enforcementReferral: null,
  capRequired: false,
  attachments: ['Airworthiness_Final_Report_Amendment.pdf'],
  summary: 'Finance-independent Final Report awaiting Executive Director decision after GM review.',
  history: [{ at: '2026-07-14 11:15', actor: 'Okan Demir', action: 'General Manager reviewed and forwarded Final Report to Executive Director' }]
}
```

The `SEED_AUDIT_REPORTS` objects must also carry the same report authority
fields and appropriate `approval` chains. Do not create one mutable artifact
that appears in both queues.

- [x] **Step 4: Generalize canonical seed merge and defaults**

Replace the hard-coded interactive canonical ID list with seed-derived IDs:

```js
var canonicalIds = SEED_MANAGER_REPORTS.filter(function (report) {
  return !!report.approvalPackageId;
}).map(function (report) {
  return report.id;
});
```

Set fresh defaults:

```js
generalManagerUi: { selectedReportId: 'FR-2026-021', validationMessage: '' },
```

and:

```js
selectedReportId: 'FR-2026-022'
```

inside `executiveDirectorUi`. Ensure older saved state receives the new seed
IDs without overwriting existing report histories/attachments. When migrating
from a pre-plan demo state, select the new pending ED record only if the saved
selection is absent or not an eligible pending decision and no unsaved ED
decision form exists.

- [x] **Step 5: Verify exact decision behavior**

Extend the GM/ED tests to apply a decision to each new ID and assert that the
other sample is unchanged:

```js
const edBefore = JSON.stringify(context.reportArtifactById('FR-2026-022', state));
const gmResult = context.applyGeneralManagerReportDecision(
  state,
  'FR-2026-021',
  'approve',
  'Reviewed and forwarded.',
  { role: 'gm', name: context.ROLES.gm.user }
);
assert.equal(gmResult.ok, true);
assert.equal(gmResult.report.ownerRole, 'executiveDirector');
assert.equal(JSON.stringify(context.reportArtifactById('FR-2026-022', state)), edBefore);
```

Run:

```bash
node --test tests/stakeholder-readiness-regressions.test.js
node tests/general-manager-workspace-smoke.test.js
node tests/executive-director-workspace-smoke.test.js
node tests/manager-reports-approval-smoke.test.js
node tests/report-approval-smoke.test.js
```

Expected: all pass; GM and ED see separate exact artifacts and their existing
authority boundaries remain intact.

---

### Task 5: Reorder Planning To Department Manager, Finance, GM, ED

**Files:**

- Modify: `tests/stakeholder-readiness-regressions.test.js`
- Modify: `tests/finance-review-workspace-smoke.test.js`
- Modify: `tests/planning-workspace-smoke.test.js`
- Modify: `tests/planning-render-smoke.test.js`
- Modify: `tests/planning-release-smoke.test.js`
- Modify: `tests/governance-render-smoke.test.js`
- Modify: `tests/executive-director-workspace-smoke.test.js`
- Modify: `js/data.js` (`SEED_PLANNING_ITEMS`, notification defaults, migration)
- Modify: `js/approval.js` (`approvalNextAction`, `setApprovalStatusFromChain`, `applyApprovalDecision`)
- Modify: `js/planning.js` (`applyFinancePlanningDecision`)
- Modify: `js/views.js` (shared Planning and Finance copy/buttons/rails)
- Modify: `js/app.js` (decision log/notification copy if required)

**Interfaces:**

- Consumes: shared approval record `{ chain, currentIndex, outcome, history }`, `applyApprovalDecision`, `applyFinancePlanningDecision`, `applyExecutivePlanningDecision`.
- Produces: `manager -> finance -> gm -> executiveDirector`, Finance return to manager, GM return to manager, and unchanged post-ED GM release.

- [x] **Step 1: Add the failing planning contract test**

Append:

```js
test('planning follows Department Manager, Finance, GM, ED and Finance returns to Department Manager', () => {
  const context = buildContext();
  const state = context.freshState();
  const plan = state.planningItems[0];

  assert.deepEqual(
    JSON.parse(JSON.stringify(plan.approval.chain.map((stage) => stage.role))),
    ['manager', 'finance', 'gm', 'executiveDirector']
  );
  assert.equal(context.approvalSummary(plan).ownerRole, 'finance');

  let result = context.applyFinancePlanningDecision(plan, {
    decision: 'return',
    actor: { role: 'finance', name: context.ROLES.finance.user },
    comment: 'Reconcile travel and accommodation.'
  });
  assert.equal(result.ok, true);
  assert.equal(context.approvalSummary(plan).ownerRole, 'manager');

  context.applyApprovalDecision(plan, {
    decision: 'forward',
    actor: { role: 'manager', name: context.ROLES.manager.user },
    comment: 'Budget revised and resubmitted.'
  });
  result = context.applyFinancePlanningDecision(plan, {
    decision: 'approve',
    actor: { role: 'finance', name: context.ROLES.finance.user },
    comment: 'Budget approved.'
  });
  assert.equal(result.ok, true);
  assert.equal(context.approvalSummary(plan).ownerRole, 'gm');

  context.applyApprovalDecision(plan, {
    decision: 'forward',
    actor: { role: 'gm', name: context.ROLES.gm.user },
    comment: 'Forwarded to Executive Director.'
  });
  assert.equal(context.approvalSummary(plan).ownerRole, 'executiveDirector');
});
```

- [x] **Step 2: Update focused test expectations before production code**

Change all planning traversals to this order:

```js
context.applyFinancePlanningDecision(item, {
  decision: 'approve',
  actor: { role: 'finance', name: context.ROLES.finance.user },
  comment: 'Budget accepted.'
});
context.applyApprovalDecision(item, {
  decision: 'forward',
  actor: { role: 'gm', name: context.ROLES.gm.user },
  comment: 'Finance-reviewed plan forwarded to Executive Director.'
});
context.applyExecutivePlanningDecision(item, {
  decision: 'approve_and_sign',
  actor: { role: 'executiveDirector', name: context.ROLES.executiveDirector.user },
  comment: 'Approved for GM release.'
});
```

Replace old expectations:

- `Send to Finance Review` on GM -> `Forward to Executive Director`.
- Finance approval owner `executiveDirector` -> `gm`.
- Finance return owner `gm` -> `manager`.
- Rail order `Department Manager, GM, Finance, Executive Director` ->
  `Department Manager, Finance Review, GM Review, Executive Director Approval`.

- [x] **Step 3: Run the planning suite and confirm RED**

Run:

```bash
node --test tests/stakeholder-readiness-regressions.test.js
node tests/finance-review-workspace-smoke.test.js
node tests/planning-workspace-smoke.test.js
node tests/planning-render-smoke.test.js
node tests/planning-release-smoke.test.js
node tests/governance-render-smoke.test.js
node tests/executive-director-workspace-smoke.test.js
```

Expected before implementation: failures identify the old chain, empty default
Finance queue, Finance-to-ED advance, and Finance-to-GM return behavior.

- [x] **Step 4: Change the seed planning chain and sample owner**

Use this exact chain in `SEED_PLANNING_ITEMS`:

```js
chain: [
  { role: 'manager', label: 'Department Manager', returnToRole: null },
  { role: 'finance', label: 'Finance Review', returnToRole: 'manager', notApprovedReturnToRole: 'manager' },
  { role: 'gm', label: 'GM Review', returnToRole: 'manager' },
  { role: 'executiveDirector', label: 'Executive Director Approval', returnToRole: 'gm' }
],
currentIndex: 1,
```

Set the seed status/history/notification to Finance review:

```js
status: 'sent_to_finance',
history: [{
  actor: 'Selin Demir',
  role: 'manager',
  action: 'submitted',
  date: '2026-06-15 10:30',
  comment: 'Submitted budget-required Q3 Cabin Inspection surveillance item for Finance Review.'
}]
```

The initial Finance Review filter remains `pending`, so the seeded plan must
appear without pre-mutating state in tests or browser QA.

- [x] **Step 5: Change shared approval behavior**

Update planning next actions:

```js
if (stage.role === 'manager') return 'Submit to Finance Review';
if (stage.role === 'finance') return 'Review budget: approve or return to Department Manager';
if (stage.role === 'gm') return 'Forward to Executive Director or return to Department Manager';
if (stage.role === 'executiveDirector') return 'Final approval or return to GM';
```

For `decision === 'forward'`, use `forwarded` for GM; do not label it
`sent_to_finance`. `finance_not_approved` must use the configured manager return
index. The planning status projection must map owner roles as follows:

```js
if (summary.ownerRole === 'manager') record.status = 'returned_to_department_manager';
else if (summary.ownerRole === 'finance') record.status = 'sent_to_finance';
else if (summary.ownerRole === 'gm') record.status = 'under_gm_review';
else if (summary.ownerRole === 'executiveDirector') record.status = 'pending_ed_approval';
```

Change `applyFinancePlanningDecision()` success text to:

```js
input.decision === 'approve'
  ? 'Budget approved and advanced to General Manager review.'
  : 'Budget returned to Department Manager for revision.'
```

- [x] **Step 6: Update role-aware decision buttons and copy**

Use these planning actions:

```js
// Department Manager
approvalDecisionButton(record, 'forward', 'Submit to Finance Review', null, true)

// General Manager planning item
approvalDecisionButton(record, 'forward', 'Forward to Executive Director', null, true) +
approvalDecisionButton(record, 'return', 'Return to Department Manager', 'danger')
```

Update Finance page purpose, approval strip, and rail copy to the confirmed
order. Do not change report-approval chains, which remain Department Manager ->
GM -> ED.

- [x] **Step 7: Migrate older saved planning records without skipping Finance**

Bump `DEMO_STATE_VERSION` from `8` to `9`. During merge, detect a saved
planning chain whose roles equal `manager,gm,finance,executiveDirector` and
replace only its chain metadata with the new seed chain while preserving:

- plan ID, budget, filters, comments, and preparation state;
- terminal approved/rejected outcome;
- existing history;
- completed Finance review data when present.

For a non-terminal old record:

- old GM stage with no Finance decision -> new Finance stage;
- old Finance stage -> new Finance stage;
- old ED stage with an approved Finance decision -> new ED stage;
- `finance_not_approved` -> Department Manager stage.

Add migration assertions to the stakeholder regression test; never silently
advance an unreviewed budget to GM or ED.

- [x] **Step 8: Verify GREEN and unchanged release boundary**

Run:

```bash
node --test tests/stakeholder-readiness-regressions.test.js
node tests/finance-review-workspace-smoke.test.js
node tests/planning-workspace-smoke.test.js
node tests/planning-render-smoke.test.js
node tests/planning-release-smoke.test.js
node tests/governance-render-smoke.test.js
node tests/executive-director-workspace-smoke.test.js
node tests/approval-smoke.test.js
```

Expected: all pass; after ED approval `preparation.status` remains
`not_released` and only GM can release to Department Manager.

---

### Task 6: Synchronize Product Truth And Plan Tracking

**Files:**

- Modify: `docs/product-specs/ux-plan/NAVIGATION_AND_INFORMATION_ARCHITECTURE.md`
- Modify: `docs/product-specs/ux-plan/NAVIGATION_AND_INFORMATION_ARCHITECTURE.turkce.md`
- Modify: `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md`
- Modify: `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.turkce.md`
- Modify after fresh verification: `docs/demo-evidence/BUILD_SUMMARY.md`
- Modify after fresh verification: `docs/demo-evidence/BUILD_SUMMARY.turkce.md`
- Modify: `docs/exec-plans/active/2026-07-15-stakeholder-feedback-remediation-plan.md`
- Modify: `docs/exec-plans/index.md`
- Modify only if needed: `docs/exec-plans/tech-debt-tracker.md`

**Interfaces:**

- Consumes: verified implementation state and literal test/browser evidence.
- Produces: synchronized English/Turkish source truth and accurate active-plan status.

- [x] **Step 1: Update canonical planning authority text**

Add the same contract to both English product docs:

```text
Planning approval order: Department Manager -> Finance Review -> GM Review -> Executive Director. Finance approval advances to GM Review. Finance Return for Revision goes to Department Manager. GM may forward a Finance-reviewed plan to Executive Director or return it to Department Manager; a corrected submission must pass Finance again. Executive Director approval does not release the plan directly; GM Release to Department remains a separate next action.
```

Add the equivalent human-ready Turkish text to both `.turkce.md` companions.
Do not change the separate Final Report authority order.

- [x] **Step 2: Update build evidence only after commands run**

Add one dated section to each build summary containing:

- the nine implemented feedback outcomes;
- the exact new planning order and Finance return owner;
- GM/ED/Finance default sample behavior;
- focused/full automated counts from fresh output;
- four viewport sizes and rendered findings from fresh Browser evidence;
- `demo-only`, `verified locally`, `not run`, and
  `production-readiness not claimed` labels.

Do not reuse old pass counts or screenshot paths as fresh evidence.

- [x] **Step 3: Keep tracking state literal**

During implementation keep this plan `active` and set its index next todo to
the next unchecked task. After all automated and rendered verification passes,
change it to `ready-for-verification` with next todo:

```text
Stakeholder feedback items 1-9 are implemented and verified locally. Next: Furkan/Burak stakeholder review and sign-off before moving the plan to completed.
```

Do not move the file to `completed/` without stakeholder sign-off. Add a
`note-open` tracker entry only for a real durable gap that remains after Task 7.

- [x] **Step 4: Run focused documentation checks**

Run:

```bash
rg -n 'Department Manager.*GM.*Finance|Finance approval advances to ED|Finance.*return.*GM' docs/product-specs docs/demo-evidence
rg -n 'Department Manager.*Finance.*GM.*Executive Director' docs/product-specs docs/demo-evidence
node tests/harness-docs-smoke.test.js
node tests/demo-boundary-smoke.test.js
git diff --check
```

Expected: no current product/evidence text claims the old planning order or old
Finance owners; historical execution plans may retain their original text but
the new plan and index identify this corrective follow-up.

---

### Task 7: Run Full Automated And Rendered Verification

**Files:**

- Modify only for real failures: in-scope implementation/test/docs files above.
- Do not write screenshots, traces, temporary scripts, or browser profiles into the repository.

**Interfaces:**

- Consumes: completed Tasks 1-6.
- Produces: fresh evidence for `ready-for-verification` or a literal blocked/gap report.

- [x] **Step 1: Run every JavaScript syntax check**

Run:

```bash
for file in js/*.js; do node --check "$file" || exit 1; done
```

Expected: exit code 0 and no syntax errors.

- [x] **Step 2: Run focused regression commands individually**

Run:

```bash
node --test tests/stakeholder-readiness-regressions.test.js
node tests/premium-ui-remediation-smoke.test.js
node tests/lead-inspector-nav-smoke.test.js
node tests/lead-inspector-workspace-smoke.test.js
node tests/general-manager-workspace-smoke.test.js
node tests/executive-director-workspace-smoke.test.js
node tests/finance-review-workspace-smoke.test.js
node tests/planning-workspace-smoke.test.js
node tests/planning-render-smoke.test.js
node tests/planning-release-smoke.test.js
node tests/governance-render-smoke.test.js
node tests/report-approval-smoke.test.js
node tests/manager-reports-approval-smoke.test.js
node tests/manager-workspace-responsive-smoke.test.js
node tests/demo-boundary-smoke.test.js
```

Expected: every command exits 0 and prints its normal pass marker/output.

- [x] **Step 3: Run the full suite and whitespace gate**

Run:

```bash
node --test tests/*.test.js
git diff --check
```

Expected: zero failed, cancelled, skipped, or todo tests; `git diff --check`
exits 0.

- [x] **Step 4: Start an isolated local static preview**

Use an available local port, preferring:

```bash
python3 -m http.server 4173 --bind 127.0.0.1
```

The flow under test is: role selection -> affected workspace -> stakeholder
reported control/state -> corrected rendered result.

- [x] **Step 5: Use the in-app Browser first**

Read and follow the Browser skill before browser actions. Name one session,
reuse the same tab, and verify page URL/title, meaningful DOM, no framework
overlay, console warning/error health, screenshot evidence, and interaction
state after every decision.

Required viewport matrix:

- `1536 x 864`
- `1366 x 768`
- `1024 x 768`
- `390 x 844`

Required routes/states:

1. Inspector -> My Assignments: no Next Inspection dossier; KPI/filter/table first.
2. Lead Inspector -> Preliminary Reports -> `PR-2026-018` -> Inspection & Findings: no CAP-status column/badges and no clipped right panel.
3. Lead Inspector -> Preliminary attachments: filenames/descriptions do not overlap.
4. Lead Inspector -> Final Reports -> overview: compact organization/CAP and key-finding metrics.
5. General Manager -> Report Approvals: `FR-2026-021` selected; open, return validation, and forward work.
6. Executive Director -> Final Reports: `FR-2026-022` selected; all four choices visible; exercise one non-terminal validation and one approval on a reset copy.
7. Finance Review: sample plan visible; Return for Revision moves to Department Manager.
8. Reset, Finance Review: approve moves to GM; GM forwards to ED; ED approves; GM release remains the next action.

For each viewport record:

- document `scrollWidth === clientWidth`;
- no relevant nested table/card overflow;
- no clipped primary button, status, filename, description, or decision panel;
- no overlapping text;
- exact selected plan/report IDs remain stable;
- console warnings/errors are empty or explicitly explained.

- [x] **Step 6: Verify boundaries after interactions**

Confirm in DOM/state/tests:

- GM forwarding does not issue/sign/lock a Final Report.
- ED approval does not close Findings.
- Finance does not sign or release a plan.
- ED plan approval still requires GM Release to Department.
- Service Provider routes do not expose `SkyCargo Air`, `BlueWing Aviation`,
  `FR-2026-021`, `FR-2026-022`, Internal CAA Notes, workload, or internal risk.
- Attachment interactions remain filename-only.

- [x] **Step 7: Clean up browser/test processes**

Inspect processes using the approved process commands and stop only the local
server/test browser processes started for this verification. Confirm there is
no leftover Playwright, Puppeteer, webdriver, headless Chrome, or temporary
HTTP server owned by this task. Do not touch the user's everyday Chrome
profile.

- [x] **Step 8: Reconcile the plan and evidence state**

If all required evidence passes:

- check Tasks 1-7;
- set this plan to `ready-for-verification`;
- update the index next todo to stakeholder sign-off;
- write the fresh English/Turkish build-summary evidence;
- leave production/release/external evidence labelled `not run`.

If any required check fails, leave the plan `active`, set the index next todo
to the first concrete failing check, and record a durable tracker note only if
the gap cannot be fixed within scope.

## Implementation Verification — 2026-07-15

Status: **ready-for-verification**. All nine stakeholder feedback items are
implemented and **verified locally** in the frontend-only demo. All 11
`js/*.js` syntax checks passed; all 15 focused Task 7 commands passed; the
stakeholder regression command passed 21/21; the full Node suite passed 55/55
with zero failures, cancellations, skips, or todo tests; documentation checks
and `git diff --check` passed.

Fresh isolated in-app Browser verification passed at `1536x864`, `1366x768`,
`1024x768`, and `390x844` across the Inspector, Lead Preliminary/Final, GM, ED,
Finance, and Service Provider flows. Final results had no page or relevant
nested horizontal overflow, no clipped/overlapping relevant controls or text,
no framework overlay, and zero relevant console warnings/errors. Exact report
and plan IDs, decision state changes, Finding/CAP/Evidence closure boundaries,
GM-intermediate and ED-final report authority, recommendation-only enforcement
referral, post-ED GM Release, and Fly Namibia Service Provider privacy remained
intact. The isolated browser tabs/server were closed and no task-owned test
process residue remained.

External production, release, deployment, real-device, real identity/signature,
real authorization, backend/database, upload/storage, notification,
production reporting/audit-log, enforcement-execution, legal/regulatory, and
stakeholder sign-off evidence is **not run**. This result is **demo-only**;
**production-readiness not claimed**. The plan remains under `active/` until
Furkan/Burak stakeholder review and sign-off.

---

## Verification Matrix

| Requirement | Automated evidence | Rendered evidence |
|---|---|---|
| Assignment dossier removed | stakeholder + premium tests | Inspector My Assignments at four viewports |
| Preliminary CAP status removed | stakeholder + Lead nav tests | Inspection & Findings table/side panel |
| Preliminary frame containment | responsive/static assertions | no page/nested overflow at four viewports |
| Attachment overlap fixed | compact/containment assertions | filename/description visual check |
| Final metrics compact | modifier/style assertions | Final overview screenshot comparison |
| GM sample/actions | GM projection/decision tests | GM queue, detail, return/forward |
| ED sample/options | ED projection/decision tests | ED decision panel and validation |
| Finance order/sample | planning/finance tests | Finance -> GM -> ED interaction |
| Finance return owner | exact owner assertion | Finance return -> Department Manager |
| Post-ED release boundary | planning release tests | ED approval -> GM Release next action |
| Privacy/closure boundaries | full suite + demo boundary | Service Provider and open Finding checks |

## Risks And Mitigations

- **Risk: Old browser state keeps empty GM/ED/Finance screens.** Mitigation:
  bump the state version, merge missing seed IDs, migrate the old chain, and
  test a pre-v9 saved fixture.
- **Risk: New report seeds drift between `auditReports` and `managerReports`.**
  Mitigation: create matching IDs in both collections and assert exact-artifact
  decisions do not mutate the other sample.
- **Risk: Reordering planning changes report approval accidentally.**
  Mitigation: limit the new sequence to planning records and rerun GM/ED report
  authority tests.
- **Risk: Removing status hides report review state.** Mitigation: remove only
  Finding CAP/lifecycle status; preserve report Draft/Department Review status.
- **Risk: CSS compaction reduces readability or touch targets.** Mitigation:
  verify all four viewports and preserve at least 40px actionable controls.
- **Risk: Existing dirty changes overlap the same large files.** Mitigation:
  inspect focused diffs after each task and never revert unrelated hunks.
- **Risk: Historical docs still contain the old order.** Mitigation: update
  current canonical product/evidence docs and identify this plan as the
  corrective follow-up; do not rewrite historical plans as if their original
  evidence had used the new order.

## Dependencies

- Approved design:
  `docs/product-specs/workflows/2026-07-15-stakeholder-feedback-remediation-design.md`.
- Existing canonical report lifecycle and role authority from
  `docs/product-specs/screen-specs/DEPARTMENT_MANAGER_WORKSPACES.md`.
- Existing privacy/permission boundary from
  `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md` and
  `docs/product-specs/modules/AUDITEE_PORTAL.md`.
- Existing frontend-only evidence/limitations from
  `docs/demo-evidence/BUILD_SUMMARY.md`.
- Existing shared approval implementation in `js/approval.js`, planning wrapper
  in `js/planning.js`, report helpers in `js/reports.js`, and GM/ED projections
  in `js/manager-workspaces.js`.
- Existing stakeholder readiness remediation remains the broader authority and
  privacy baseline; this plan is a corrective follow-up for newly supplied
  stakeholder feedback.

## Ownership Boundaries

- Product/stakeholder owner: accepts the new planning order, simplified report
  screens, and seeded demo examples.
- Frontend implementer: owns browser-local state, renderers, actions, CSS,
  tests, and synchronized demo docs only.
- Product/legal/security owners: retain ownership of any future production
  identity, signature validity, authorization, enforcement, retention,
  immutable audit logging, and regulatory policy.
- External verifier/stakeholder: performs sign-off before the plan can move
  from `ready-for-verification` to `completed`.

## Completion Gate

This plan may move to `ready-for-verification` only when:

- all Tasks 1-7 are checked;
- all nine feedback items have automated and rendered evidence;
- every `js/*.js` syntax check passes;
- the full Node suite reports zero failures;
- `git diff --check` passes;
- all four browser viewports pass console, clipping, overlap, overflow, and
  interaction checks;
- exact GM/ED report IDs and Finance/GM/ED plan ownership are verified;
- CAP/Evidence closure, GM/ED authority, Service Provider privacy, and
  post-ED GM release boundaries remain intact;
- English/Turkish product/evidence docs and the plan index match the result;
- external production/release evidence remains explicitly `not run` and
  production readiness is not claimed.

## Plan Self-Review

- **Spec coverage:** Tasks 1-5 map directly to feedback items 1-9; Tasks 6-7
  cover source truth, tracking, and required verification.
- **Placeholder scan:** the plan contains no TBD/TODO/implement-later steps;
  every code-changing task names files, assertions, commands, and expected
  results.
- **Type consistency:** report IDs use `FR-2026-021`/`FR-2026-022`; owner roles
  use `manager`, `finance`, `gm`, and `executiveDirector`; planning records keep
  the existing `approval.chain/currentIndex/outcome/history` shape.
- **Scope check:** the work stays within the static frontend demo and does not
  add a second lifecycle, backend service, or broad redesign.
- **Authority check:** planning order changes; Final Report authority does not.
  Finance returns to Department Manager, GM forwards to ED, ED approves, and GM
  release remains separate.
- **Line-count check:** this plan and its Execution Prompt are both far below
  the user's 4,000-line ceiling.

## Execution Prompt

```text
Implement docs/exec-plans/active/2026-07-15-stakeholder-feedback-remediation-plan.md in /Users/marlonjd/Developer/web/aviaSurveil360.

Read AGENTS.md, the approved design linked by the plan, the active plan, and its listed canonical product/evidence sources before editing. Work on the current branch and preserve all unrelated dirty-worktree changes. Do not create or switch branches, stage, commit, push, deploy, open a PR, or post GitHub comments unless I separately authorize that exact action.

Execute Tasks 1-7 in order. Do not dispatch subagents unless I explicitly request delegation. Add focused regression coverage for each scoped correction, then rerun the focused and neighboring tests.

Keep AviaSurveil360 frontend-only with HTML, CSS, Vanilla JavaScript, mock data, and browser-local state. Do not add backend, database, API, real authentication/authorization, real upload/storage, real notifications, real e-signature, production reporting/audit-log services, framework migration, or production-readiness claims.

Implement all nine stakeholder items: remove the My Assignments Next Inspection dossier; remove CAP/lifecycle status from Lead Preliminary Finding content while retaining report status; fix Preliminary frame and attachment overlap; compact Final Report metrics; seed distinct decision-ready GM and ED Final Reports; expose working GM/ED decisions; change planning to Department Manager -> Finance Review -> GM Review -> Executive Director; route Finance and GM revision returns to Department Manager; show the Finance sample on reset; preserve post-ED GM Release to Department.

Preserve exact report/plan identity, GM-intermediate and ED-final report authority, CAP/Evidence closure rules, recommendation-only enforcement referral, and Service Provider organization privacy. Migrate pre-v9 browser state without skipping Finance or overwriting unrelated saved records.

After implementation, run every js/*.js syntax check, every focused command in Task 7, node --test tests/*.test.js, and git diff --check. Then use the in-app Browser first against an isolated local static server at 1536x864, 1366x768, 1024x768, and 390x844. Verify meaningful content, zero relevant console errors/warnings, no framework overlay, no page/relevant nested overflow, no clipped/overlapping controls, stable exact IDs, and working state changes for Inspector, Lead Preliminary/Final, GM, ED, and Finance flows. Clean up only the test server/browser processes started by this task.

Update the English/Turkish canonical product and build-evidence docs only from fresh evidence. Keep the active plan, index next todo, and any durable tracker note synchronized. Mark the plan ready-for-verification only when every required local gate passes; do not mark completed without stakeholder sign-off. Report remaining production/external evidence as not run and state that the result is demo-only and production-readiness is not claimed.
```
