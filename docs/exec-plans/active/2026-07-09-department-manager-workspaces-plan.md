# Department And General Manager Workspaces Implementation Plan


**Goal:** Complete the state-backed Department Manager and General Manager demo experience: restricted role navigation, dashboards, Audits, Findings Review, Inspection Team, Reports Approval with valid PDFs, CAP Monitoring, Checklist Management, and Risk Dashboards, using the single user-facing organization name `Fly Namibia`.

**Architecture:** Preserve the existing static HTML, CSS, and Vanilla JavaScript application. Keep shared persisted records in `js/data.js`; use `js/manager-workspaces.js` for pure manager selectors/mutations and PDF helpers; render role-specific workspaces in `js/views.js`; and dispatch interactions through `js/app.js`. Project dashboards, CAP monitoring, risks, and GM summaries from shared audits/findings/CAPs/reports/users instead of adding conflicting screen-only datasets. Store Preliminary and Final Reports as separate artifacts and preserve published checklist versions.

**Tech Stack:** HTML, CSS, Vanilla JavaScript, browser-local demo state, direct Node `assert`/`vm` smoke tests, browser-side Blob downloads, local static-server browser QA.

## Global Constraints

- Use `AviaSurveil360` as the canonical product name.
- Display the operator/service provider as `Fly Namibia` everywhere in changed user-facing UI, report content, current stakeholder docs, and generated filenames.
- Display the organization type as `Operator / Service Provider` when a type is needed.
- Keep the implementation frontend-only and demo-only.
- Do not add a backend, database, API, real authentication, real authorization enforcement, real upload/storage, real notification delivery, production reporting engine, e-signature, framework migration, or package-manager setup.
- CAP acceptance is not finding closure.
- Final Report issuance/locking requires the configured final authorized approval; Department Manager approval alone only forwards it.
- Keep `Comment to Auditee` separate from `Internal CAA Note`.
- Preserve evidence version history conceptually.
- Use careful wording: `reference`, `configured rule`, `finding basis`, `expected evidence`, and `review result`.
- Do not create, switch, rename, or delete a branch.
- After each independently verified task, create one focused Conventional Commit and push the current branch before starting the next task.
- Do not stage unrelated files. If a task overlaps pre-existing uncommitted work, review the staged diff and include only the files/hunks owned by that task.
- Do not create a PR unless the user explicitly requests it later.
- Preserve the unrelated untracked file `docs/exec-plans/active/2026-07-08-modern-aviation-saas-rollout-plan.md`.
- Bump the `index.html` asset query token after JavaScript or CSS behavior changes.

---

## Objective

Deliver the approved design in
`docs/product-specs/screen-specs/DEPARTMENT_MANAGER_WORKSPACES.md` so a
Department Manager can:

1. use only the approved Department Manager sidebar entries and open its compact
   operational Dashboard;
2. open Findings Review and immediately inspect a Fly Namibia audit and its
   findings;
3. open Inspection Team, use a row ellipsis action menu, and manage the selected
   manager-scoped team through meaningful demo interactions;
4. open Reports Approval, review separate Preliminary and Final Report
   artifacts, and approve, request revision, or return each report;
5. download valid Preliminary, Final, Executive Summary, and Team Assignment
   demo PDFs;
6. monitor CAPs and open the selected CAP detail drawer from a row ellipsis;
7. create, edit, remove, duplicate, archive, and version Department Manager
   checklist packages, sections, and questions in browser-local state;
8. review Department Manager risks and navigate from the manager dashboard;
9. use only the approved General Manager sidebar entries, review departments
   and cross-department risk, and make the final authorized report decision;
10. complete the paths at desktop and mobile sizes without console errors,
   clipping, overlap, or page-level horizontal overflow.

## Scope

In scope:

- canonical user-facing `Fly Namibia` normalization on current application and
  stakeholder surfaces touched by the active demo;
- Department Manager sidebar allowlist: Dashboard, Audits, Reports Approval,
  Risk Dashboard, Inspection Team, Findings Review, CAP Monitoring, and
  Checklist Management;
- General Manager sidebar allowlist: Dashboard, Report Approvals, Departments,
  Risk Dashboard, and Settings;
- seeded manager-scoped audit findings that do not pre-seed the live hero
  finding `CAB-2026-001`;
- manager reporting lines and inspection-team metadata;
- working filters, tabs, row selection, ellipsis menu, team mutations, notes,
  mock filename attachments, messages, and history;
- separate manager Preliminary and Final Report records and decisions;
- reusable dependency-free PDF generation/download helpers;
- Department Manager Dashboard, CAP Monitoring with right-side detail drawer,
  Checklist Management, and Risk Dashboard;
- General Manager Dashboard, department overview, risk summary, and Final Report
  final-authorized approval/return flow;
- responsive styling and automated/browser verification;
- bilingual build evidence and plan lifecycle updates after verification.

Out of scope is listed separately below.

## Assumptions

- The current branch remains unchanged; no branch operation is authorized.
- `AUD-2026-001` remains the primary Cabin Inspection audit and `ORG-XYZ`
  remains its internal organization ID.
- The live checklist still creates `CAB-2026-001`; the manager review page uses
  other seeded finding IDs so the live scenario remains demonstrable.
- The supplied screenshots define the interaction anatomy and density, not
  exact counts such as 28 findings or 24 inspections.
- Existing direct Node smoke tests remain the canonical automated harness; no
  `package.json` is introduced.
- Existing saved browser state may be version 4 and must receive version 5
  defaults without throwing or losing unrelated state.

## Dependencies

- `js/data.js` state and seed conventions.
- `js/helpers.js` escaping, status, role-visibility, persistence, notification,
  and audit-log helpers.
- `js/approval.js` existing governance labels and shared approval concepts.
- `js/views.js` page shell, table, badge, modal, and icon helpers.
- `js/app.js` route state, delegated `data-act` dispatcher, persistence, modal,
  and toast helpers.
- Local Browser/in-app browser tooling for rendered QA.
- `/usr/bin/file` and bundled `pdfinfo` for downloaded PDF inspection.

## Ownership Boundaries

This plan owns:

- `index.html`
- `css/styles.css`
- `js/data.js`
- `js/manager-workspaces.js` (new)
- `js/views.js`
- `js/app.js`
- `MANIFEST.md`
- focused new tests under `tests/`
- active canonical scenario/build-summary name normalization where needed
- `docs/demo-evidence/BUILD_SUMMARY.md`
- `docs/demo-evidence/BUILD_SUMMARY.turkce.md`
- `docs/exec-plans/index.md`
- this plan file

This plan owns its focused Conventional Commits and pushes to the already
configured current branch after each verified task. It does not own unrelated
active plans, original reference materials, branch operations, pull requests,
GitHub comments, production architecture, or deployment.

## File Map

- `js/manager-workspaces.js` — pure manager selectors, validation/mutation
  functions, report decisions, CAP/checklist/risk/GM projections, and generic
  PDF builder/download helpers.
- `js/data.js` — canonical organization constant, enriched users, inspection
  teams, pre-existing manager-review findings, separate manager report
  artifacts, fresh-state defaults, and version-5 merge repair.
- `js/views.js` — manager/GM dashboards, Findings Review, Inspection Team,
  Reports Approval, CAP Monitoring, Checklist Management, Risk Dashboards, and
  their responsive content, menus, tabs, forms, and empty/error states.
- `js/app.js` — manager navigation/routes and delegated handlers that call the
  manager helper module, persist, render, navigate, open modals, or download.
- `css/styles.css` — manager workbench visual system and responsive rules.
- `index.html` — new script include plus asset query token bump.
- `tests/department-manager-state-smoke.test.js` — naming, seed, merge, selector,
  and route-state foundation.
- `tests/department-manager-findings-smoke.test.js` — Findings Review render and
  interaction contract.
- `tests/inspection-team-smoke.test.js` — manager scope, menu, mutations,
  message/history, and assignment PDF contract.
- `tests/manager-reports-approval-smoke.test.js` — separate artifacts and all
  manager decision transitions.
- `tests/manager-report-pdf-smoke.test.js` — PDF header, MIME/filename inputs,
  and browser download trigger contract.
- `tests/manager-navigation-dashboard-smoke.test.js` — exact Department Manager
  allowlist, dashboard projections, and card navigation.
- `tests/manager-cap-monitoring-smoke.test.js` — filters, counters, ellipsis,
  drawer tabs, updates, documents, history, and closure boundary.
- `tests/manager-checklist-management-smoke.test.js` — package/version/section/
  question CRUD and published-version preservation.
- `tests/manager-risk-dashboard-smoke.test.js` — risk aggregates, filters,
  exposure matrix, recent findings, and export.
- `tests/general-manager-workspace-smoke.test.js` — exact GM allowlist,
  department/risk summaries, and final authorized report decisions.

## Phases And Tasks

### Task 1: Canonical Naming And Manager State Foundation

**Files:**

- Create: `tests/department-manager-state-smoke.test.js`
- Create: `js/manager-workspaces.js`
- Modify: `js/data.js:7-12, 40-60, 102-110, 245-420, 900-960, 922-990, 1115-1382`
- Modify: `index.html` script list and asset tokens
- Modify: `MANIFEST.md` static prototype and smoke-test lists

**Interfaces:**

- Produces: `CANONICAL_SERVICE_PROVIDER_NAME === 'Fly Namibia'`.
- Produces: `state.users`, `state.inspectionTeams`, `state.managerReports`,
  `state.managerFindingsUi`, `state.inspectionTeamUi`, and
  `state.managerReportsUi`.
- Produces: `ensureManagerWorkspaceState(targetState) -> object`.
- Produces: `managerFindingsForAudit(targetState, auditId) -> Finding[]`.
- Produces: `managerReportById(targetState, reportId) -> ManagerReport|null`.
- Consumes: existing `deepClone`, audits, findings, and state merge boundary.

- [x] **Step 1: Write the failing state smoke test**

Create a VM test that loads `js/data.js` and the not-yet-created helper module,
then asserts the exact foundation:

```js
assert.equal(context.CANONICAL_SERVICE_PROVIDER_NAME, 'Fly Namibia');
assert.equal(context.ROLES.auditee.orgName, 'Fly Namibia');
assert.equal(context.SEED_ORGS[0].name, 'Fly Namibia');

const fresh = context.freshState();
assert.equal(fresh.managerFindingsUi.selectedAuditId, 'AUD-2026-001');
assert.equal(fresh.inspectionTeamUi.selectedAuditId, 'AUD-2026-001');
assert.equal(fresh.managerReportsUi.selectedReportId, 'PR-2026-018');
assert.ok(fresh.inspectionTeams.some((team) => team.auditId === 'AUD-2026-001'));
assert.ok(fresh.managerReports.some((report) => report.id === 'PR-2026-018'));
assert.ok(fresh.managerReports.some((report) => report.id === 'FR-2026-018'));
assert.equal(context.managerFindingsForAudit(fresh, 'AUD-2026-001').length, 4);

const migrated = context.mergeDemoState({ demoStateVersion: 4, findings: [] });
assert.equal(migrated.demoStateVersion, 5);
assert.equal(migrated.managerFindingsUi.selectedAuditId, 'AUD-2026-001');
assert.ok(migrated.managerReports.length >= 2);
```

- [x] **Step 2: Run the test and verify RED**

Run:

```bash
node tests/department-manager-state-smoke.test.js
```

Expected: FAIL because `js/manager-workspaces.js`, the canonical constant, and
the new state collections do not exist.

- [x] **Step 3: Add canonical seeds and state defaults**

Add these exact public shapes in `js/data.js`:

```js
var CANONICAL_SERVICE_PROVIDER_NAME = 'Fly Namibia';
var DEMO_STATE_VERSION = 5;

var SEED_INSPECTION_TEAMS = [
  {
    id: 'TEAM-AUD-2026-001',
    auditId: 'AUD-2026-001',
    department: 'Cabin Safety',
    status: 'In Progress',
    startDate: '2026-06-15',
    endDate: '2026-06-20',
    leadUserId: 'USR-CANER',
    memberIds: ['USR-CANER', 'USR-AYLIN', 'USR-MEHMET', 'USR-SELIN'],
    notes: 'Preliminary Report review target: 21 Jun 2026. Final Report submission target: 27 Jun 2026.',
    attachments: [],
    messages: [],
    history: [{ at: '2026-06-10 09:00', actor: 'Mehmet Kaya', action: 'Inspection team confirmed' }]
  }
];

var SEED_MANAGER_REPORTS = [
  {
    id: 'PR-2026-018', auditId: 'AUD-2026-001', organization: CANONICAL_SERVICE_PROVIDER_NAME,
    reportType: 'Preliminary Report', version: '1.0', leadInspector: 'Caner Yildiz',
    submittedAt: '2026-07-09 10:30', status: 'pending_manager', ownerRole: 'manager',
    capRequired: true, managerComment: '', attachments: ['Cabin_Checklist_Response_Summary.pdf'],
    summary: 'Preliminary Cabin Inspection report for authorized review.',
    history: [{ at: '2026-07-09 10:30', actor: 'Caner Yildiz', action: 'Submitted to Department Manager' }]
  },
  {
    id: 'FR-2026-018', auditId: 'AUD-2026-001', organization: CANONICAL_SERVICE_PROVIDER_NAME,
    reportType: 'Final Report', version: '2.0', leadInspector: 'Caner Yildiz',
    submittedAt: '2026-07-10 14:20', status: 'pending_manager', ownerRole: 'manager',
    capRequired: true, managerComment: '', attachments: ['CAP_Evidence_Summary.pdf'],
    summary: 'Final Cabin Inspection report prepared after the configured CAP/evidence stage.',
    history: [{ at: '2026-07-10 14:20', actor: 'Caner Yildiz', action: 'Final Report submitted to Department Manager' }]
  }
];
```

Enrich internal users with stable `id`, `roleKey`, `department`, `email`, and
`reportsToRole`; add `USR-SELIN` as a Cabin Safety inspector. Add four
non-hero findings under `AUD-2026-001`, one each for severities 1, 2, 3, and 0,
with real lifecycle fields and configured/demo references. Do not seed
`CAB-2026-001`.

Add these exact state defaults:

```js
users: deepClone(SEED_USERS),
inspectionTeams: deepClone(SEED_INSPECTION_TEAMS),
managerReports: deepClone(SEED_MANAGER_REPORTS),
managerFindingsUi: { query: '', status: 'all', dateRange: 'all', selectedAuditId: 'AUD-2026-001', tab: 'overview' },
inspectionTeamUi: { query: '', department: 'all', status: 'all', dateRange: 'all', selectedAuditId: 'AUD-2026-001', tab: 'overview', openMenuAuditId: '' },
managerReportsUi: { query: '', reportType: 'all', status: 'all', selectedReportId: 'PR-2026-018', tab: 'summary', validationMessage: '' },
```

In `mergeDemoState`, merge each UI object with its fresh default and repair
missing arrays from the new seeds. Merge the four required manager findings by
ID so an older saved `findings` array still exposes the first-load manager
review scenario without deleting user-created findings.

- [x] **Step 4: Add the helper module boundary and script registration**

Create `js/manager-workspaces.js` with defensive state normalization and pure
lookup selectors:

```js
function ensureManagerWorkspaceState(target) {
  var s = target || state;
  if (!Array.isArray(s.users)) s.users = deepClone(SEED_USERS);
  if (!Array.isArray(s.inspectionTeams)) s.inspectionTeams = deepClone(SEED_INSPECTION_TEAMS);
  if (!Array.isArray(s.managerReports)) s.managerReports = deepClone(SEED_MANAGER_REPORTS);
  return s;
}

function managerFindingsForAudit(target, auditId) {
  var s = ensureManagerWorkspaceState(target);
  return s.findings.filter(function (finding) { return finding.auditId === auditId; });
}

function managerReportById(target, reportId) {
  var s = ensureManagerWorkspaceState(target);
  return s.managerReports.filter(function (report) { return report.id === reportId; })[0] || null;
}
```

Load the new script after `js/reports.js` and before `js/work-items.js` in
`index.html`. Add it to `MANIFEST.md`.

- [x] **Step 5: Normalize active user-facing Fly Namibia strings**

Replace visible `FlyNamibia` and `FlyNamibia (Pty) Ltd` forms with
`Fly Namibia` in active JS data/view copy, tests, the canonical current scenario,
and bilingual build summaries. Use underscores in generated filenames, for
example `Fly_Namibia_Preliminary_Report_PR-2026-018.pdf`.

Do not rewrite historical completed-plan evidence. Negative examples in the
approved design's naming contract may remain because they explain forbidden
variants.

- [x] **Step 6: Run GREEN checks**

Run:

```bash
node --check js/data.js
node --check js/manager-workspaces.js
node tests/department-manager-state-smoke.test.js
```

Expected: syntax exits 0 and the state smoke prints its `ok` marker.

- [x] **Step 7: Review checkpoint without committing**

Run:

```bash
git diff --check
git status --short
```

Expected: no whitespace errors; the unrelated untracked rollout plan remains
untouched. Do not commit.

- [x] **Step 8: Commit and push the verified foundation**

Stage only Task 1 files/hunks, inspect `git diff --cached`, commit with
`feat: add department manager workspace foundation`, and push the current
branch. Confirm the push succeeds before treating Task 1 as fully closed.

### Task 2: Department Manager Findings Review

**Files:**

- Create: `tests/department-manager-findings-smoke.test.js`
- Modify: `js/manager-workspaces.js`
- Modify: `js/views.js` near `viewFindings()` and shared view helpers
- Modify: `js/app.js:6-35, 115-130, 446-495, 657-900`
- Modify: `css/styles.css`

**Interfaces:**

- Consumes: state and selectors from Task 1.
- Produces: `managerInspectionRows(targetState, filters) -> ManagerInspectionRow[]`.
- Produces: `managerFindingCounts(findings) -> { total, critical, major, minor, observations, open, inReview, closed }`.
- Produces: `viewManagerFindingsReview() -> string`.
- Produces actions: `manager-findings-select`, `manager-findings-filter`,
  `manager-findings-tab`, `manager-findings-open-finding`,
  `manager-findings-open-report`, and `manager-findings-export`.

- [x] **Step 1: Write the failing Findings Review smoke test**

Load the full script order and assert:

```js
context.state = context.freshState();
context.state.role = 'manager';
context.state.view = 'findings-review';
context.render();
let html = elements.get('app-root').innerHTML;
assert.match(html, /Findings Review/);
assert.match(html, /AUD-2026-001/);
assert.match(html, /Fly Namibia/);
assert.match(html, /Findings Overview/);
assert.match(html, /Findings List/);
assert.match(html, /By Department/);
assert.match(html, /By Level/);
assert.match(html, /Current Owner/);
assert.match(html, /Next Action/);
assert.match(html, /Due Date/);

context.handleAction('manager-findings-tab', dataEl({ 'data-tab': 'list' }));
html = elements.get('app-root').innerHTML;
assert.match(html, /CAB-2026-011/);
assert.match(html, /Level 1 Critical/);
```

Also assert that selecting a different audit changes
`state.managerFindingsUi.selectedAuditId` and that a no-result query renders
`No inspections match these filters.`.

- [x] **Step 2: Run the test and verify RED**

Run `node tests/department-manager-findings-smoke.test.js`.

Expected: FAIL because the route, renderer, and actions do not exist.

- [x] **Step 3: Implement pure inspection aggregation**

Add selectors that derive rows from `state.audits` and `state.findings`:

```js
function managerFindingCounts(findings) {
  return findings.reduce(function (counts, finding) {
    counts.total += 1;
    if (finding.severity === 1) counts.critical += 1;
    else if (finding.severity === 2) counts.major += 1;
    else if (finding.severity === 3) counts.minor += 1;
    else counts.observations += 1;
    if (finding.status === 'CLOSED') counts.closed += 1;
    else if (finding.status === 'CAP_SUBMITTED' || finding.status === 'EVIDENCE_SUBMITTED') counts.inReview += 1;
    else counts.open += 1;
    return counts;
  }, { total: 0, critical: 0, major: 0, minor: 0, observations: 0, open: 0, inReview: 0, closed: 0 });
}
```

`managerInspectionRows` returns audit/organization/team-lead fields plus the
count object and filters by query/status/date range without mutating state.

- [x] **Step 4: Add manager navigation and route dispatch**

In `NAV.manager`, expose `Findings Review` with view `findings-review`. Add
`findings-review` to `VIEW_TITLES` and `renderContent()`:

```js
case 'findings-review': return viewManagerFindingsReview();
```

Keep the generic `findings` route available for finding-detail navigation but
remove the redundant manager `Open Findings` navigation label.

- [x] **Step 5: Render the master-detail workspace**

Implement `viewManagerFindingsReview()` with:

- search/status/date controls;
- five compact inspection KPIs;
- inspection rows with a selected-state rail;
- right-side header and the four required tabs;
- compact finding-level and finding-status summaries;
- department breakdown table;
- full finding list with Owner, Next Action, Due Date, Status, Severity, Audit,
  and Organization;
- buttons that switch to the list tab or navigate to `reports-approval` with the
  related report selected.

Use semantic buttons and existing escape/status helpers. If the current
selection is filtered out, select the first visible row without mutating audit
data. If no rows remain, render the exact empty state.

- [x] **Step 6: Wire delegated actions and export**

Handlers update only manager UI state, persist when state changes, and render.
`manager-findings-open-finding` navigates to the existing `finding` route.
`manager-findings-export` creates a UTF-8 CSV Blob containing the visible list
and downloads `Fly_Namibia_Findings_Review.csv`; it must not be toast-only.

- [x] **Step 7: Add focused styling and verify GREEN**

Add namespaced `.manager-findings-*` / shared `.manager-workbench-*` styles.
Run:

```bash
node --check js/manager-workspaces.js
node --check js/views.js
node --check js/app.js
node tests/department-manager-findings-smoke.test.js
node tests/table-first-workbench-smoke.test.js
```

Expected: all exit 0.

- [x] **Step 8: Review checkpoint without committing**

Run `git diff --check`; inspect the diff for duplicated hard-coded audit/finding
datasets. Do not commit.

- [x] **Step 9: Commit and push Findings Review**

Stage only Task 2 files/hunks, inspect `git diff --cached`, commit with
`feat: add department manager findings review`, and push the current branch.

### Task 3: Inspection Team Workspace And Actions

**Files:**

- Create: `tests/inspection-team-smoke.test.js`
- Modify: `js/manager-workspaces.js`
- Modify: `js/views.js`
- Modify: `js/app.js`
- Modify: `css/styles.css`

**Interfaces:**

- Consumes: `state.users`, `state.inspectionTeams`, audit records, PDF helpers
  introduced in this task, persistence/log/notification helpers.
- Produces: `managerTeamRows(targetState, filters) -> ManagerTeamRow[]`.
- Produces: `managerTeamByAuditId(targetState, auditId) -> InspectionTeam|null`.
- Produces: `addManagerTeamMember`, `removeManagerTeamMember`,
  `changeManagerTeamLead`, `updateManagerTeamSchedule`, and
  `recordManagerTeamMessage`, each returning `{ ok, message, team }`.
- Produces: `viewInspectionTeam() -> string`.
- Produces actions prefixed `manager-team-`.

- [x] **Step 1: Write the failing Inspection Team smoke test**

Assert manager scoping, the menu, detail, and mutations:

```js
context.state = context.freshState();
context.state.role = 'manager';
context.state.view = 'inspection-team';
context.render();
let html = elements.get('app-root').innerHTML;
assert.match(html, /Inspection Team/);
assert.match(html, /Fly Namibia/);
assert.match(html, /View Team Details/);
assert.doesNotMatch(html, /Other Manager Private Inspector/);

context.handleAction('manager-team-menu', dataEl({ 'data-id': 'AUD-2026-001' }));
assert.equal(context.state.inspectionTeamUi.openMenuAuditId, 'AUD-2026-001');
context.handleAction('manager-team-select', dataEl({ 'data-id': 'AUD-2026-001' }));
assert.equal(context.state.inspectionTeamUi.selectedAuditId, 'AUD-2026-001');

const duplicate = context.addManagerTeamMember(context.state, 'AUD-2026-001', 'USR-AYLIN');
assert.equal(duplicate.ok, false);
const removeLead = context.removeManagerTeamMember(context.state, 'AUD-2026-001', 'USR-CANER');
assert.equal(removeLead.ok, false);
const changed = context.changeManagerTeamLead(context.state, 'AUD-2026-001', 'USR-AYLIN');
assert.equal(changed.ok, true);
```

Also assert schedule changes append history and messages append a team message.

- [x] **Step 2: Run the test and verify RED**

Run `node tests/inspection-team-smoke.test.js`.

Expected: FAIL because the route and team functions do not exist.

- [x] **Step 3: Implement scoped selectors and mutation validation**

Only include users with `reportsToRole === 'manager'` in selectable members.
Use one result shape everywhere:

```js
function teamMutationResult(ok, message, team) {
  return { ok: ok, message: message, team: team || null };
}
```

Reject duplicate members. Reject removal of `leadUserId`. Reject a lead change
to a user who is not already a team member. Successful mutations append a
history entry with `logTimestamp()` when available and do not change other
teams.

- [x] **Step 4: Add route, navigation, and master-detail renderer**

Add `Inspection Team` to manager nav, `VIEW_TITLES`, and render dispatch. Render
summary metrics, filters, team rows, and the selected detail tabs:

- Overview
- Team Members
- Assignments
- Documents
- History

The overview shows member role, name, department, email, and status. Notes use
a real textarea. Attachments display selected filenames only.

- [x] **Step 5: Implement the ellipsis menu and focused forms**

Use `aria-expanded` and an anchored menu for the selected row. Implement these
outcomes:

- View Team Details — select the row and show Overview.
- Edit Team / Add Inspector / Remove Inspector / Change Lead Inspector — open
  focused modals and call the validated mutation functions.
- Update Schedule — modal with `startDate` and `endDate`; reject an end before
  the start.
- View Assignment Package — modal preview.
- View Audit Details — existing `audit-detail` route.
- Send Message to Team — required message body; append to `team.messages` and
  history.
- Download Team Assignment — valid PDF download.
- View Activity Log — select the History tab.

Member-row ellipses may expose only supported Add/Remove/Make Lead actions.

- [x] **Step 6: Add generic PDF builder and assignment download**

Generalize the existing dependency-free PDF logic behind these exact public
functions in `js/manager-workspaces.js`:

```js
function pdfSafeText(text) {
  return String(text || '')
    .replace(/[–—]/g, '-')
    .replace(/[“”]/g, '"')
    .replace(/[‘’]/g, "'")
    .replace(/[^\x20-\x7E]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

function pdfEscape(text) {
  return pdfSafeText(text).replace(/\\/g, '\\\\').replace(/\(/g, '\\(').replace(/\)/g, '\\)');
}

function pdfWrap(text, maxChars) {
  var clean = pdfSafeText(text);
  if (!clean) return [''];
  return clean.split(' ').reduce(function (lines, word) {
    var current = lines[lines.length - 1] || '';
    var candidate = current ? current + ' ' + word : word;
    if (candidate.length > maxChars && current) lines.push(word);
    else lines[lines.length - 1] = candidate;
    return lines;
  }, ['']);
}

function buildAviaPdfDocument(lines) {
  var printable = [];
  (lines && lines.length ? lines : ['AviaSurveil360 Demo Report']).forEach(function (line) {
    pdfWrap(line, 96).forEach(function (wrapped) { printable.push(wrapped); });
  });
  var pages = [];
  while (printable.length) pages.push(printable.splice(0, 43));
  var objects = ['<< /Type /Catalog /Pages 2 0 R >>', ''];
  var fontObject = objects.push('<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>');
  var pageReferences = [];
  pages.forEach(function (pageLines, pageIndex) {
    var y = 792;
    var content = pageLines.map(function (line, lineIndex) {
      if (!line) { y -= 8; return ''; }
      var size = pageIndex === 0 && lineIndex === 0 ? 18 : 10;
      var command = 'BT /F1 ' + size + ' Tf 54 ' + y + ' Td (' + pdfEscape(line) + ') Tj ET\n';
      y -= pageIndex === 0 && lineIndex === 0 ? 24 : 14;
      return command;
    }).join('');
    var contentObject = objects.push('<< /Length ' + content.length + ' >>\nstream\n' + content + 'endstream');
    var pageObject = objects.push('<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 842] /Resources << /Font << /F1 ' + fontObject + ' 0 R >> >> /Contents ' + contentObject + ' 0 R >>');
    pageReferences.push(pageObject + ' 0 R');
  });
  objects[1] = '<< /Type /Pages /Kids [' + pageReferences.join(' ') + '] /Count ' + pageReferences.length + ' >>';
  var pdf = '%PDF-1.4\n';
  var offsets = [0];
  objects.forEach(function (body, index) {
    offsets[index + 1] = pdf.length;
    pdf += (index + 1) + ' 0 obj\n' + body + '\nendobj\n';
  });
  var xref = pdf.length;
  pdf += 'xref\n0 ' + (objects.length + 1) + '\n0000000000 65535 f \n';
  for (var i = 1; i <= objects.length; i += 1) pdf += String(offsets[i]).padStart(10, '0') + ' 00000 n \n';
  return pdf + 'trailer\n<< /Size ' + (objects.length + 1) + ' /Root 1 0 R >>\nstartxref\n' + xref + '\n%%EOF';
}

function downloadAviaPdf(filename, lines, env) {
  var runtime = env || { Blob: Blob, URL: URL, document: document };
  var pdf = buildAviaPdfDocument(lines);
  var blob = new runtime.Blob([pdf], { type: 'application/pdf' });
  var url = runtime.URL.createObjectURL(blob);
  var link = runtime.document.createElement('a');
  link.href = url;
  link.download = filename;
  runtime.document.body.appendChild(link);
  link.click();
  runtime.document.body.removeChild(link);
  runtime.URL.revokeObjectURL(url);
  return { ok: true, filename: filename, mime: 'application/pdf' };
}
```

The builder must escape parentheses/backslashes, wrap long lines, build valid
xref offsets, and return a `%PDF-1.4` document. The team assignment filename is
`Fly_Namibia_AUD-2026-001_Team_Assignment.pdf`.

- [x] **Step 7: Verify GREEN and regressions**

Run:

```bash
node --check js/manager-workspaces.js
node --check js/views.js
node --check js/app.js
node tests/inspection-team-smoke.test.js
node tests/planning-workspace-smoke.test.js
node tests/planning-release-smoke.test.js
```

Expected: all exit 0.

- [x] **Step 8: Review checkpoint without committing**

Run `git diff --check`; confirm every enabled menu item changes visible state,
opens content, navigates, or downloads. Do not commit.

- [x] **Step 9: Commit and push Inspection Team**

After Step 8 passes, stage only Task 3 files/hunks, inspect
`git diff --cached`, commit with `feat: add department manager inspection team`,
and push the current branch.

### Task 4: Reports Approval, Preliminary/Final Decisions, And PDFs

**Files:**

- Create: `tests/manager-reports-approval-smoke.test.js`
- Create: `tests/manager-report-pdf-smoke.test.js`
- Modify: `js/manager-workspaces.js`
- Modify: `js/views.js`
- Modify: `js/app.js`
- Modify: `css/styles.css`

**Interfaces:**

- Consumes: `state.managerReports`, `managerReportById`, generic PDF helpers,
  role/notification/log helpers.
- Produces: `managerReportRows(targetState, filters) -> ManagerReport[]`.
- Produces: `applyManagerReportDecision(targetState, reportId, decision, comment, actor) -> { ok, message, report }`.
- Produces: `managerReportPdfLines(report, variant) -> string[]` where `variant`
  is `report` or `executive`.
- Produces: `viewManagerReportsApproval() -> string`.
- Produces actions prefixed `manager-report-`.

- [x] **Step 1: Write the failing report decision test**

Assert separate artifacts and exact transitions:

```js
const preliminaryState = context.freshState();
let result = context.applyManagerReportDecision(preliminaryState, 'PR-2026-018', 'approve', 'Approved for CAP response.', { role: 'manager', name: 'Mehmet Kaya' });
assert.equal(result.ok, true);
assert.equal(result.report.status, 'released_to_service_provider');
assert.equal(result.report.ownerRole, 'auditee');
assert.equal(context.managerReportById(preliminaryState, 'FR-2026-018').status, 'pending_manager');

const finalState = context.freshState();
result = context.applyManagerReportDecision(finalState, 'FR-2026-018', 'approve', 'Approved for final governance review.', { role: 'manager', name: 'Mehmet Kaya' });
assert.equal(result.report.status, 'submitted_to_executive');
assert.equal(result.report.ownerRole, 'executiveDirector');
assert.notEqual(result.report.status, 'issued');

const revisionState = context.freshState();
result = context.applyManagerReportDecision(revisionState, 'PR-2026-018', 'revision', '', { role: 'manager', name: 'Mehmet Kaya' });
assert.equal(result.ok, false);
assert.match(result.message, /comment/i);
```

Also test `return` with a comment, terminal-stage decision blocking, appended
history, and the unchanged sibling report.

- [x] **Step 2: Run decision test and verify RED**

Run `node tests/manager-reports-approval-smoke.test.js`.

Expected: FAIL because decision functions and the route do not exist.

- [x] **Step 3: Implement report selectors and decision state machine**

Use these manager-stage transitions:

```js
var MANAGER_REPORT_TRANSITIONS = {
  'Preliminary Report': {
    approveWithCap: { status: 'released_to_service_provider', ownerRole: 'auditee', action: 'Approved and released to Service Provider' },
    approveWithoutCap: { status: 'submitted_to_gm', ownerRole: 'gm', action: 'Approved and forwarded to General Manager' }
  },
  'Final Report': {
    approve: { status: 'submitted_to_executive', ownerRole: 'executiveDirector', action: 'Department Manager approved and forwarded for final authorized approval' }
  }
};
```

`revision` sets `revision_requested`; `return` sets `returned_to_lead`; both set
`ownerRole: 'leadInspector'` and require a non-empty comment. Only
`pending_manager` records accept a manager decision. Every success records
`managerDecision`, `managerDecisionAt`, `managerComment`, and a history entry.

- [x] **Step 4: Write the failing PDF/download smoke test**

Inject fake Blob/URL/document objects and assert:

```js
const preliminary = context.managerReportById(context.freshState(), 'PR-2026-018');
const reportPdf = context.buildAviaPdfDocument(context.managerReportPdfLines(preliminary, 'report'));
assert.match(reportPdf, /^%PDF-1\.4/);
assert.match(reportPdf, /Fly Namibia/);

const download = context.downloadAviaPdf('Fly_Namibia_Preliminary_Report_PR-2026-018.pdf', ['Fly Namibia'], fakeEnv);
assert.equal(download.mime, 'application/pdf');
assert.equal(clickedDownload, 'Fly_Namibia_Preliminary_Report_PR-2026-018.pdf');
assert.equal(createdBlobOptions.type, 'application/pdf');
```

Repeat for Final Report and Executive Summary filenames.

- [x] **Step 5: Run PDF test and verify RED**

Run `node tests/manager-report-pdf-smoke.test.js`.

Expected: FAIL because report-specific line/filename behavior is missing.

- [x] **Step 6: Render Reports Approval and wire interactions**

Add the route/nav label `Reports Approval`. Render:

- All Reports, Preliminary Reports, Final Reports, Pending My Approval,
  Revision Requested, and Approved counters;
- search/type/status filters;
- report queue with selected row;
- right dossier with Summary, Findings, Attachments, Comments, and History tabs;
- required Manager Comments textarea;
- Request Revision, Return Report, and Approve Report controls;
- Download PDF menu containing the report-appropriate PDF and Executive Summary;
- `Review Full Report` preview.

`Approve Report` calls `applyManagerReportDecision`. Revision/return display
inline `validationMessage` when the comment is empty. A success persists,
records a mock in-app notification for the next owner, records the audit log,
and re-renders the selected dossier.

- [x] **Step 7: Implement report PDF lines and downloads**

`managerReportPdfLines` includes product name, report type/version, Audit ID,
Organization, Lead Inspector, submitted date, status, executive summary,
finding counts, CAP/evidence note, and a `Demo-only` footer. Do not include
Internal CAA Notes in a Service Provider-facing variant.

Use exact filenames:

- `Fly_Namibia_Preliminary_Report_PR-2026-018.pdf`
- `Fly_Namibia_Final_Report_FR-2026-018.pdf`
- `Fly_Namibia_Executive_Summary_PR-2026-018.pdf` or the selected Final ID

- [x] **Step 8: Verify GREEN and existing governance regressions**

Run:

```bash
node --check js/manager-workspaces.js
node --check js/views.js
node --check js/app.js
node tests/manager-reports-approval-smoke.test.js
node tests/manager-report-pdf-smoke.test.js
node tests/department-preliminary-review-smoke.test.js
node tests/report-approval-smoke.test.js
node tests/governance-render-smoke.test.js
```

Expected: all exit 0.

- [x] **Step 9: Review checkpoint without committing**

Run `git diff --check`; verify manager approval does not issue/lock a Final
Report. Do not commit.

- [x] **Step 10: Commit and push Reports Approval and PDFs**

Stage only Task 4 files/hunks, inspect `git diff --cached`, commit with
`feat: add manager report approvals and pdf downloads`, and push the current
branch.

### Task 5: Department Manager Navigation And Dashboard

**Files:**

- Create: `tests/manager-navigation-dashboard-smoke.test.js`
- Modify: `js/manager-workspaces.js`
- Modify: `js/views.js`
- Modify: `js/app.js`
- Modify: `css/styles.css`
- Modify: `index.html`

**Interfaces:**

- Consumes: shared audits, findings, reports, inspection teams, and CAP state.
- Produces: `managerDashboardProjection(targetState) -> ManagerDashboardProjection`.
- Produces: the exact `NAV.manager` allowlist and working dashboard-card routes.

- [x] **Step 1: Write the failing manager navigation/dashboard smoke test**

Assert the exact visible navigation contract and shared-state projections:

```js
assert.deepEqual(context.NAV.manager.map(function (item) { return item.label; }), [
  'Dashboard', 'Audits', 'Reports Approval', 'Risk Dashboard',
  'Inspection Team', 'Findings Review', 'CAP Monitoring', 'Checklist Management'
]);
const dashboard = context.managerDashboardProjection(context.freshState());
assert.equal(dashboard.organization, 'Fly Namibia');
assert.ok(dashboard.totalAudits >= dashboard.inProgressAudits);
assert.ok(Array.isArray(dashboard.recentHighRiskFindings));
assert.match(context.viewManagerDashboard(), /Reports Awaiting Approval/);
assert.doesNotMatch(context.viewManagerDashboard(), /Calendar|Documents|Corrective Actions/);
```

Also assert every dashboard task card has a `data-act="go"` route present in
the manager allowlist.

- [x] **Step 2: Run the focused test and verify RED**

Run `node tests/manager-navigation-dashboard-smoke.test.js`.

Expected: FAIL because the current manager sidebar contains legacy entries and
the dashboard does not expose the approved projection contract.

- [x] **Step 3: Implement the projection and exact sidebar allowlist**

Implement `managerDashboardProjection` by deriving counts from shared state:
audits, pending manager reports, open/high-risk findings, active/overdue CAPs,
manager-scoped team members, upcoming audits, and recent high-risk findings.
Replace `NAV.manager` with the exact ordered allowlist from Step 1. Keep legacy
routes internal only when existing flows still need them.

- [x] **Step 4: Render and wire the compact Department Manager Dashboard**

Render six KPI cards, seven task cards, upcoming audits, and recent high-risk
findings using the existing card/table visual system. Each card navigates to the
named route and retains `Fly Namibia` copy. Add responsive rules without
changing unrelated role dashboards.

- [x] **Step 5: Verify GREEN and regressions**

Run:

```bash
node --check js/manager-workspaces.js
node --check js/views.js
node --check js/app.js
node tests/manager-navigation-dashboard-smoke.test.js
node tests/department-manager-findings-smoke.test.js
node tests/inspection-team-smoke.test.js
```

Expected: all exit 0.

- [x] **Step 6: Commit and push manager navigation/dashboard**

Run `git diff --check`, stage only Task 5 files/hunks, inspect
`git diff --cached`, commit with `feat: add department manager dashboard`, and
push the current branch.

### Task 6: CAP Monitoring And Detail Drawer

**Files:**

- Create: `tests/manager-cap-monitoring-smoke.test.js`
- Modify: `js/data.js`
- Modify: `js/manager-workspaces.js`
- Modify: `js/views.js`
- Modify: `js/app.js`
- Modify: `css/styles.css`
- Modify: `index.html`

**Interfaces:**

- Consumes: `state.findings`, each finding's CAP/evidence/history, audits, and
  users; CAP records remain linked to their finding instead of creating a
  conflicting closure model.
- Produces: `managerCapRows(targetState, filters) -> ManagerCapRow[]`.
- Produces: `managerCapById(targetState, capId) -> ManagerCapRow|null`.
- Produces: `managerCapMetrics(rows) -> ManagerCapMetrics`.
- Produces: `addManagerCapUpdate(targetState, capId, text, actor) -> Result`.
- Produces: `viewManagerCapMonitoring() -> string` and `manager-cap-*` actions.

- [x] **Step 1: Write the failing CAP Monitoring smoke test**

```js
const state = context.freshState();
const rows = context.managerCapRows(state, { status: 'all', department: 'all', inspection: 'all', due: 'all' });
assert.ok(rows.length >= 1);
assert.equal(rows[0].organization, 'Fly Namibia');
assert.ok(rows[0].findingId);
assert.ok(rows[0].dueDate);
const before = rows[0].history.length;
const result = context.addManagerCapUpdate(state, rows[0].id, 'Evidence review scheduled.', 'Department Manager');
assert.equal(result.ok, true);
assert.equal(context.managerCapById(state, rows[0].id).history.length, before + 1);
assert.notEqual(context.managerCapById(state, rows[0].id).findingStatus, 'CLOSED');
```

Assert the rendered route contains the five drawer tabs, row ellipsis,
`Add Update`, filters, progress, Due Date, days left/overdue, and an empty state.

- [x] **Step 2: Run the test and verify RED**

Run `node tests/manager-cap-monitoring-smoke.test.js`.

Expected: FAIL because the Department Manager CAP projection/route is absent.

- [x] **Step 3: Add defensive CAP seed enrichment and selectors**

Enrich representative seeded manager findings with CAP fields required by the
reference: ID, status, action owner, assignee, Due Date, target closure date,
priority, progress, root cause, action plan, updates, attachment filenames,
mock notifications, and history. Repair missing arrays/fields when older saved
state loads. Derive filters, counters, overdue/upcoming summaries, and progress
from these shared finding CAP records.

- [x] **Step 4: Implement update validation without closing findings**

`addManagerCapUpdate` rejects blank text, appends a timestamped update and CAP
history entry, updates `lastUpdate`, and leaves finding status/closure evidence
unchanged. No manager CAP action may equate CAP acceptance with finding closure.

- [x] **Step 5: Render CAP Monitoring and the ellipsis detail drawer**

Render reference-style filters, six compact counters, CAP table, status
overview, overdue list, and upcoming list. The row ellipsis selects the exact
CAP and opens a right drawer with `Overview`, `Action Plan`, `Updates`,
`Documents`, and `History`. Wire drawer close, tab changes, Add Update, and
attachment filename display through delegated actions.

- [x] **Step 6: Verify GREEN and lifecycle regressions**

Run:

```bash
node --check js/data.js
node --check js/manager-workspaces.js
node --check js/views.js
node --check js/app.js
node tests/manager-cap-monitoring-smoke.test.js
node tests/demo-boundary-smoke.test.js
node tests/inspector-nav-smoke.test.js
```

Expected: all exit 0 and the closure-boundary smoke remains unchanged.

- [x] **Step 7: Commit and push CAP Monitoring**

Run `git diff --check`, stage only Task 6 files/hunks, inspect
`git diff --cached`, commit with `feat: add department manager cap monitoring`,
and push the current branch.

### Task 7: Checklist Management

**Files:**

- Create: `tests/manager-checklist-management-smoke.test.js`
- Modify: `js/data.js`
- Modify: `js/manager-workspaces.js`
- Modify: `js/views.js`
- Modify: `js/app.js`
- Modify: `css/styles.css`
- Modify: `index.html`

**Interfaces:**

- Consumes: `state.managedChecklists`, `state.questionBank`, shared audit-log
  conventions, and browser-local persistence.
- Produces: `managerChecklistPackages(targetState) -> ChecklistPackage[]`.
- Produces: `createManagerChecklistPackage`, `duplicateManagerChecklistPackage`,
  `archiveManagerChecklistPackage`, and `publishManagerChecklistVersion`.
- Produces: `addManagerChecklistSection`, `removeManagerChecklistSection`,
  `moveManagerChecklistSection`, `saveManagerChecklistQuestion`,
  `duplicateManagerChecklistQuestion`, and `removeManagerChecklistQuestion`.
- Produces: `viewManagerChecklistManagement() -> string` and
  `manager-checklist-*` actions.

- [x] **Step 1: Write the failing checklist management smoke test**

```js
const state = context.freshState();
const original = context.managerChecklistPackages(state)[0];
const duplicate = context.duplicateManagerChecklistPackage(state, original.id, 'Department Manager');
assert.equal(duplicate.ok, true);
assert.equal(duplicate.package.status, 'Draft');
const section = context.addManagerChecklistSection(state, duplicate.package.id, 'Dispatch');
assert.equal(section.ok, true);
const question = context.saveManagerChecklistQuestion(state, duplicate.package.id, section.section.id, {
  text: 'Are dispatch records complete?', reference: 'Configured procedure OPS-1',
  evidenceMethods: ['Document Review'], likelihood: 'Possible', impact: 'Major',
  findingTypes: ['Compliance'], mandatory: true, critical: false, status: 'Active'
});
assert.equal(question.ok, true);
assert.equal(context.managerChecklistPackages(state).filter(function (item) { return item.id === original.id; })[0].version, original.version);
```

Also assert blank names/questions are rejected, the last section cannot be
removed, archive hides a package from the Active filter, and publishing creates
a preserved version/history entry.

- [x] **Step 2: Run the test and verify RED**

Run `node tests/manager-checklist-management-smoke.test.js`.

Expected: FAIL because the Department Manager package/section/question API and
route are absent.

- [x] **Step 3: Normalize managed checklist packages for the reference layout**

Add defensive package fields for department, effective date, owner, status,
version, attachments, sections, questions, and history while preserving current
published checklist records. Mutations operate on drafts; publishing snapshots
the prior version and increments the new version without overwriting history.

- [x] **Step 4: Implement validated package, section, and question mutations**

Use stable IDs; reject duplicate package/section names in the same scope; reject
blank question text/reference; calculate the demo risk score from likelihood and
impact; and record actor/action/timestamp history for every create, duplicate,
archive, reorder, save, remove, or publish operation.

- [x] **Step 5: Render and wire Checklist Management**

Match the supplied three-column package/section/question anatomy. Implement
working create, duplicate, archive, publish, add/remove/reorder section, and
add/edit/duplicate/activate/deactivate/remove question controls. The edit panel
contains the exact fields in the approved design and persists browser-local
state. Attachment controls store selected filenames only.

- [x] **Step 6: Verify GREEN and existing checklist governance**

Run:

```bash
node --check js/data.js
node --check js/manager-workspaces.js
node --check js/views.js
node --check js/app.js
node tests/manager-checklist-management-smoke.test.js
node tests/checklist-management-smoke.test.js
node tests/checklist-approval-smoke.test.js
node tests/governance-render-smoke.test.js
```

Expected: all exit 0; current published/runnable checklist behavior remains
available.

- [x] **Step 7: Commit and push Checklist Management**

Run `git diff --check`, stage only Task 7 files/hunks, inspect
`git diff --cached`, commit with `feat: add department checklist management`,
and push the current branch.

### Task 8: Department Manager Risk Dashboard

**Files:**

- Create: `tests/manager-risk-dashboard-smoke.test.js`
- Modify: `js/manager-workspaces.js`
- Modify: `js/views.js`
- Modify: `js/app.js`
- Modify: `css/styles.css`
- Modify: `index.html`

**Interfaces:**

- Consumes: shared findings, CAPs, audits, departments, risk levels, Due Dates,
  and closure status.
- Produces: `managerRiskProjection(targetState, filters) -> ManagerRiskProjection`.
- Produces: `managerRiskCsv(projection) -> string`.
- Produces: `viewManagerRiskDashboard() -> string` and `manager-risk-*` actions.

- [x] **Step 1: Write the failing manager risk smoke test**

```js
const projection = context.managerRiskProjection(context.freshState(), {
  dateRange: 'all', department: 'all', inspection: 'all', risk: 'all'
});
assert.equal(projection.totalFindings,
  projection.high + projection.medium + projection.low + projection.veryLow);
assert.equal(projection.matrix.length, 25);
assert.ok(Array.isArray(projection.recentHighRiskFindings));
assert.match(context.managerRiskCsv(projection), /Fly Namibia/);
assert.match(context.viewManagerRiskDashboard(), /Risk Exposure Matrix/);
```

Assert a no-match filter returns zero metrics and a visible empty state without
throwing.

- [x] **Step 2: Run the test and verify RED**

Run `node tests/manager-risk-dashboard-smoke.test.js`.

Expected: FAIL because the role-specific projection and route do not exist.

- [x] **Step 3: Implement shared-state risk aggregation**

Normalize finding risk into High, Medium, Low, and Very Low management buckets;
calculate the 5x5 likelihood/impact matrix; aggregate by department and week;
count overdue CAPs by risk; and select recent high-risk findings. Keep the
Oversight Health/risk indicator disclaimer and never mutate enforcement or
closure state.

- [x] **Step 4: Render filters, compact charts, tables, and export**

Render four filters, six KPI cards, findings-by-risk ring, trend, 5x5 matrix,
top risky areas, department stacked distribution, overdue CAP summary, and
recent findings table. Use HTML/CSS and existing chart primitives; do not add a
chart library or handcrafted SVG. Export a browser-side UTF-8 CSV rather than a
toast-only action.

- [x] **Step 5: Verify GREEN and risk boundary regressions**

Run:

```bash
node --check js/manager-workspaces.js
node --check js/views.js
node --check js/app.js
node tests/manager-risk-dashboard-smoke.test.js
node tests/table-first-workbench-smoke.test.js
node tests/demo-boundary-smoke.test.js
```

Expected: all exit 0.

- [x] **Step 6: Commit and push the Department Manager Risk Dashboard**

Run `git diff --check`, stage only Task 8 files/hunks, inspect
`git diff --cached`, commit with `feat: add department manager risk dashboard`,
and push the current branch.

### Task 9: General Manager Dashboard, Navigation, And Final Approval

**Files:**

- Create: `tests/general-manager-workspace-smoke.test.js`
- Modify: `js/manager-workspaces.js`
- Modify: `js/views.js`
- Modify: `js/app.js`
- Modify: `css/styles.css`
- Modify: `index.html`

**Interfaces:**

- Consumes: Department Manager-approved Final Reports from Task 4, shared
  departments, findings, CAPs, audits, and risk projections.
- Produces: the exact `NAV.gm` allowlist.
- Produces: `generalManagerProjection(targetState) -> GeneralManagerProjection`.
- Produces: `applyGeneralManagerReportDecision(targetState, reportId, decision, comment, actor) -> Result`.
- Produces: GM Dashboard, Report Approvals, Departments, and Risk Dashboard
  views/actions without Department Manager editing controls.

- [x] **Step 1: Write the failing General Manager workspace smoke test**

```js
assert.deepEqual(context.NAV.gm.map(function (item) { return item.label; }), [
  'Dashboard', 'Report Approvals', 'Departments', 'Risk Dashboard', 'Settings'
]);
const state = context.freshState();
const finalReport = state.managerReports.filter(function (report) {
  return report.type === 'Final Report';
})[0];
finalReport.status = 'submitted_to_executive';
finalReport.ownerRole = 'executiveDirector';
let result = context.applyGeneralManagerReportDecision(state, finalReport.id, 'return', '', 'General Manager');
assert.equal(result.ok, false);
result = context.applyGeneralManagerReportDecision(state, finalReport.id, 'approve', 'Final authorized approval.', 'General Manager');
assert.equal(result.ok, true);
assert.equal(result.report.status, 'issued');
assert.equal(result.report.locked, true);
```

Assert a Preliminary Report and a Final Report not at the GM stage cannot be
issued, and the GM view contains no team/checklist editing controls.

- [x] **Step 2: Run the test and verify RED**

Run `node tests/general-manager-workspace-smoke.test.js`.

Expected: FAIL because the exact GM allowlist, projection, and final-authorized
decision function do not exist.

- [x] **Step 3: Implement the GM projection and decision guard**

Project Pending Final Reports, High Risk Findings, reports awaiting GM approval,
Overdue CAPs, department rows, risk heat map/distribution, and approval rows from
shared state. `approve` is allowed only for an unlocked Final Report with
`status: submitted_to_executive`; it sets `issued`, `locked: true`, decision
metadata, history, audit log, and mock notification. `return` requires a comment
and sends the report to the configured Department Manager/Lead Inspector stage.

- [x] **Step 4: Render the exact GM navigation and workspaces**

Replace `NAV.gm` with the allowlist from Step 1. Render the supplied GM Dashboard
anatomy, Report Approvals queue, Departments overview, cross-department Risk
Dashboard, and existing Settings route. Row open/approve/return controls update
visible state and history; no enabled control is toast-only.

- [x] **Step 5: Verify GREEN and report governance regressions**

Run:

```bash
node --check js/manager-workspaces.js
node --check js/views.js
node --check js/app.js
node tests/general-manager-workspace-smoke.test.js
node tests/manager-reports-approval-smoke.test.js
node tests/report-approval-smoke.test.js
node tests/governance-render-smoke.test.js
```

Expected: all exit 0 and Department Manager approval alone still does not issue
or lock the Final Report.

- [x] **Step 6: Commit and push General Manager workspaces**

Run `git diff --check`, stage only Task 9 files/hunks, inspect
`git diff --cached`, commit with `feat: add general manager oversight dashboard`,
and push the current branch.

### Task 10: Responsive Fidelity, Interaction Polish, And Boundary QA

**Files:**

- Modify: `css/styles.css`
- Modify: `index.html` asset tokens
- Modify: any focused tests above only when a verified render contract requires
  a correction
- Create/Update: `design-qa.md` in the project root as the screenshot comparison
  ledger required by the selected image-to-code workflow

**Interfaces:**

- Consumes: all routes and interactions from Tasks 2-9.
- Produces: desktop/mobile render evidence and `design-qa.md` with
  `final result: passed` or a literal blocker.

- [x] **Step 1: Run all focused tests before browser work**

Run:

```bash
node tests/department-manager-state-smoke.test.js
node tests/department-manager-findings-smoke.test.js
node tests/inspection-team-smoke.test.js
node tests/manager-reports-approval-smoke.test.js
node tests/manager-report-pdf-smoke.test.js
node tests/manager-navigation-dashboard-smoke.test.js
node tests/manager-cap-monitoring-smoke.test.js
node tests/manager-checklist-management-smoke.test.js
node tests/manager-risk-dashboard-smoke.test.js
node tests/general-manager-workspace-smoke.test.js
```

Expected: all pass.

- [x] **Step 2: Start an isolated local preview**

Run the static server on an available localhost port and use an isolated browser
profile. Preferred command:

```bash
python3 -m http.server 4360 --bind 127.0.0.1
```

Open the demo through the in-app Browser. Reset demo state before each approval
branch.

- [x] **Step 3: Verify the desktop interaction path at 1536x864**

Verify visible behavior:

1. Department Manager navigation shows exactly the approved eight entries and
   every Dashboard task card opens its named route.
2. Findings Review initially selects Fly Namibia; all four tabs work; a finding
   detail and related report open.
3. Inspection Team ellipsis opens on the correct row; View Team Details,
   member/schedule change, message, History, and Team Assignment PDF work.
4. Reports Approval switches Preliminary/Final filters; revision with no comment
   validates; revision/return/approve update status/history; both PDF types and
   Executive Summary download.
5. CAP Monitoring filters work; the row ellipsis opens the correct five-tab CAP
   drawer; Add Update changes visible update/history state without closure.
6. Checklist Management package, section, question, archive, duplicate, and
   publish-version interactions persist browser-local state.
7. Department Manager Risk Dashboard filters/export work and display the risk
   disclaimer.
8. General Manager navigation shows exactly the approved five entries; the
   dashboard, departments, risk, and Final Report approve/return paths work.
9. Browser console has zero warnings/errors for changed paths.

- [x] **Step 4: Verify mobile at 390x844**

Confirm no page-level horizontal overflow, clipped primary decisions, overlap,
or inaccessible detail content. Master-detail panes stack with list first and
selected detail second. Ellipsis menus remain within the viewport.

- [x] **Step 5: Capture and compare visual evidence**

Capture the Department Manager Dashboard, Findings Review, Inspection Team,
Reports Approval, CAP Monitoring drawer, Checklist Management editor, Department
Manager Risk Dashboard, and General Manager Dashboard/approval state.
Use `view_image` on the supplied references and the new captures in the same QA
pass. Record at least these comparison points in `design-qa.md`:

- sidebar/header anatomy;
- table density and selected-row treatment;
- split-pane proportions;
- tab and status treatment;
- action-menu placement;
- typography scale;
- spacing/container model;
- responsive collapse.

Fix all P0/P1/P2 mismatches, recapture, and set `final result: passed` only when
the comparison passes. Remaining P3 polish may be documented.

- [x] **Step 6: Inspect downloaded PDFs**

For each downloaded file, run `/usr/bin/file` and bundled `pdfinfo`. Expected:
PDF document identification, at least one page, and no parser error.

- [x] **Step 7: Run boundary and full regression suite**

Run:

```bash
find js -maxdepth 1 -name '*.js' -exec node --check '{}' ';'
node --test tests/*.test.js
node tests/demo-boundary-smoke.test.js
git diff --check
```

Expected: all JavaScript parses, all smoke tests pass, demo boundary passes,
and no whitespace errors appear.

- [x] **Step 8: Clean up browser/test processes**

Inspect for leftover local server, test Chrome/Chrome Helper, webdriver,
Playwright, Puppeteer, or headless browser processes. Stop only processes
started by this task. Preserve the user's everyday browser profile and data.

- [x] **Step 9: Commit and push integrated QA fixes/evidence**

Stage only Task 10 fixes and `design-qa.md`, inspect `git diff --cached`, commit
with `test: verify manager workspace flows`, and push the current branch.

### Task 11: Evidence, Documentation, And Plan Lifecycle

**Files:**

- Modify: `docs/demo-evidence/BUILD_SUMMARY.md`
- Modify: `docs/demo-evidence/BUILD_SUMMARY.turkce.md`
- Modify: `docs/product-specs/scenarios/DEMO_SCENARIO_OPERATOR_AUDIT.md`
- Modify: `docs/product-specs/scenarios/DEMO_SCENARIO_OPERATOR_AUDIT.turkce.md`
- Modify: `docs/exec-plans/index.md`
- Modify: this plan file
- Modify: `docs/exec-plans/tech-debt-tracker.md` only if a durable gap remains

**Interfaces:**

- Consumes: fresh verification output and browser/PDF evidence from Task 10.
- Produces: truthful bilingual demo evidence and synchronized plan status.

- [x] **Step 1: Update current canonical naming and build evidence**

Use `Fly Namibia` in the active scenario and build summaries. Add a concise
section covering:

- Department Manager Findings Review;
- manager-scoped Inspection Team actions;
- separate Preliminary and Final Report manager approvals;
- real browser-generated demo PDFs;
- exact Department Manager and General Manager role navigation;
- Department Manager Dashboard, CAP Monitoring, Checklist Management, and Risk
  Dashboard;
- General Manager Dashboard, Departments, Risk Dashboard, and final authorized
  Final Report approval;
- exact local test/browser/PDF verification actually performed;
- explicit demo-only and production-readiness-not-claimed boundaries.

- [x] **Step 2: Synchronize this plan and active index**

If all required checks pass, mark task checkboxes complete and set the index
row to `ready-for-verification` with one next todo: stakeholder review/sign-off.
Do not move the plan to `completed/` without explicit stakeholder/user
sign-off. If a required check is unavailable, leave the plan `active` or
`blocked` as appropriate and record the exact gap.

- [x] **Step 3: Run docs/content checks**

Run:

```bash
git diff --check
rg -n "Department Manager Workspaces|department-manager-workspaces-plan|Findings Review|Inspection Team|Reports Approval" docs/exec-plans docs/product-specs docs/demo-evidence
```

Expected: no broken relative plan/spec references and status/next todo match the
actual result.

- [x] **Step 4: Final evidence review**

Re-read the approved design acceptance criteria, this plan's objective,
`docs/exec-plans/index.md`, `design-qa.md`, and the bilingual build summaries.
Report only literal `verified locally`, `not run`, `blocked`, `demo-only`, and
`production-readiness not claimed` evidence labels supported by this turn.

- [x] **Step 5: Commit and push evidence/plan closure**

Stage only Task 11 documentation and plan-tracking files, inspect
`git diff --cached`, commit with `docs: record manager workspace verification`,
and push the current branch. Confirm `docs/exec-plans/index.md` still names the
single next todo supported by the final verification state.

## Verification Matrix

| Surface | Automated proof | Rendered proof |
|---|---|---|
| Canonical naming/state | `department-manager-state-smoke` | Visible `Fly Namibia` on all manager/GM routes |
| Manager navigation/dashboard | `manager-navigation-dashboard-smoke` | Exact eight-entry sidebar and working dashboard cards |
| Findings Review | `department-manager-findings-smoke` | Filters, tabs, selected audit, finding detail/report navigation |
| Inspection Team | `inspection-team-smoke` | Ellipsis, details, mutation forms, message, history, PDF |
| Reports Approval | `manager-reports-approval-smoke` | Preliminary/Final filters and all manager decisions |
| PDF | `manager-report-pdf-smoke` | Browser downloads plus `file`/`pdfinfo` |
| CAP Monitoring | `manager-cap-monitoring-smoke` | Filters, ellipsis drawer, five tabs, update/history |
| Checklist Management | `manager-checklist-management-smoke` | Package, version, section, and question actions |
| Manager Risk Dashboard | `manager-risk-dashboard-smoke` | Filters, decision charts/tables, CSV export |
| General Manager | `general-manager-workspace-smoke` | Exact sidebar, departments/risk, final authorization |
| Regression/boundary | full `node --test`, `demo-boundary-smoke` | Console, desktop/mobile overflow, auditee privacy spot check |

## Risks And Mitigations

- **Parallel hard-coded datasets:** mitigate by deriving screen rows from core
  state and keeping new records in `data.js`, not inside render functions.
- **Saved-state drift:** mitigate with version-5 merge defaults and ID-based seed
  repair tests.
- **Approval-chain conflict:** keep manager report artifacts separate from the
  existing governance records, then preserve existing report tests unchanged.
- **Hero scenario contamination:** use non-`CAB-2026-001` seeded findings for
  first-load review.
- **Auditee privacy regression:** keep organization filters and Internal CAA
  Notes separation under existing boundary tests and browser spot checks.
- **Inert reference controls:** show only actions with a state change, content
  view, navigation, modal, or real demo download.
- **PDF parser failure:** unit-test the PDF header/Blob contract and inspect
  actual downloads with `file` and `pdfinfo`.
- **Large-file fragility:** place pure new logic in `js/manager-workspaces.js`
  and keep app/view edits as thin dispatch/render integration.
- **Visual density on mobile:** use a stacked master-detail layout and verify
  the exact 390x844 viewport.
- **Broad scope and shared-file conflicts:** execute one task at a time, stage
  only owned hunks, review each staged diff, commit, and push before continuing.

## Explicit Out Of Scope

- Production backend, database, API, authentication, authorization, audit-log,
  evidence repository, document management, or notification infrastructure.
- Real file upload/storage; attachment interactions retain filenames only.
- Legal/regulatory validation, automated enforcement, certificate action, or
  automatic finding closure.
- Production PDF/report templates, e-signature, records retention, or digital
  signing.
- Framework migration, new package ecosystem, hosted CI, deployment, or new
  remote infrastructure. Git push to the already configured current branch is
  required after each verified task.
- Exact replication of reference-screen record counts or personal names.
- Broad redesign of unrelated roles and routes.

## Execution Prompt

```text
Execute docs/exec-plans/active/2026-07-09-department-manager-workspaces-plan.md according to the approved design at docs/product-specs/screen-specs/DEPARTMENT_MANAGER_WORKSPACES.md. Work on the current branch without branch operations, PRs, or GitHub comments. Preserve the unrelated untracked modern-aviation-saas rollout plan. Run the focused and full Node, browser, responsive, PDF, and design-QA evidence gates relevant to the changed behavior. Keep the frontend-only demo boundary, normalize changed user-facing organization copy to Fly Namibia, and keep docs/exec-plans/index.md and the plan checkboxes synchronized with the actual final state.
```
