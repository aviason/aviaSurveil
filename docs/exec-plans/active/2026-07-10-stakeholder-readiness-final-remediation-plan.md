# Stakeholder Readiness Final Remediation Implementation Plan


**Goal:** Resolve every Blocking finding and every material Suggestion from the independent review of `b1782c2^..961d5c2`, then re-establish truthful stakeholder-review readiness for the AviaSurveil360 frontend-only demo.

**Architecture:** Repair shared report identity, authority, notification scope, and closure contracts before role-specific rendering. Keep `auditReports` as the single mutable approval lifecycle, make role workspaces exact-ID projections over that state, then remediate responsive/control behavior and synchronize the English/Turkish product truth before an independent rendered GO/NO-GO gate.

**Tech Stack:** HTML, CSS, Vanilla JavaScript, browser-local mock state, Node.js built-in `assert`/`vm` smoke tests, isolated in-app browser QA.

**Status:** ready-for-verification
**Date:** 2026-07-10
**Owner:** implementation agent; independent verifier owns the final GO/NO-GO decision; stakeholder/product owner owns wording and policy acceptance.
**Predecessor:** [Inspector, Report, Service Provider, and Governance Workflow Remediation](2026-07-10-inspector-report-and-governance-workflow-remediation-plan.md), returned to `ready-for-verification` after this corrective local demo GO.
**Approved design:** [Stakeholder Readiness Remediation Design](../../product-specs/workflows/2026-07-10-stakeholder-readiness-remediation-design.md).

## Global Constraints

- Keep AviaSurveil360 frontend-only and demo-only. Do not add a backend, API, database, real authentication, real authorization service, real file storage, real notification delivery, production audit log, real e-signature, real enforcement execution, framework migration, or deployment.
- Preserve `Finding -> CAP -> Evidence -> CAA Review -> Closure`. CAP acceptance is not Finding closure, and report approval must not close an open Finding or bypass Evidence/verification work.
- GM is an intermediate Final Report reviewer only. ED is the only demo issue/mock-sign/lock authority.
- Service Provider users may see only their own organization's CAA-visible records. They may not see internal CAA notes, other organizations, Inspector workload, internal risk scoring, private dashboard data, or enforcement deliberations.
- Keep all signatures labelled `DEMO mock approval mark - not a real e-signature`; enforcement remains a recommendation-only referral.
- Preserve evidence versions conceptually and keep mock upload behavior filename-only.
- Use the canonical Fly Namibia Cabin Inspection data. Do not revive stale SMS, SkyCargo, JetFast, or orphan `INS-2026-014` content in the remediated paths.
- Preserve unrelated dirty/untracked files, including `docs/exec-plans/active/2026-07-08-modern-aviation-saas-rollout-plan.md`.
- Do not create or switch branches, commit, push, deploy, or create a PR unless the user separately authorizes that exact Git action in the execution task.
- Use English for source, tests, plan notes, and canonical docs; update matching `.turkce.md` companions whenever an existing bilingual stakeholder-facing document changes.

---

## Objective

Convert the independent review's NO-GO into a defensible GO by removing split report truth, closing GM/ED authority ambiguity, enforcing organization-scoped Service Provider utilities, replacing hard-coded Lead Final Report behavior, connecting assignment IDs to execution, fixing exact Preliminary IDs, eliminating nested responsive overflow and inert controls, and reconciling all package/documentation claims with fresh evidence.

## Scope

### In scope

- Canonical Preliminary/Final report identity, projection, persistence migration, decisions, history, ownership, issue/lock, preview, and PDF state.
- GM, ED, Department Manager, Lead Inspector, Inspector, Finance, and Service Provider workflow boundaries touched by the review findings.
- Auditee Messages and Settings privacy plus organization/user-scoped notifications.
- Multi-Inspector question IDs and assignment consumption by Inspector execution.
- Lead Final Report queue/detail/prepare/preview/attachment/record/submit controls.
- Exact-ID Department Preliminary decisions and logs.
- Four-viewport responsive behavior, keyboard reachability, modal focus return, console, clipping, nested overflow, stale content, and visible-control behavior.
- Focused/full Node tests, JavaScript syntax, whitespace checks, bilingual docs, MANIFEST, plan/index/tracker truth.

### Out of scope

- Production architecture, authentication, authorization enforcement, identity assurance, signature validity, regulatory/legal policy, retention, hosted audit logs, enforcement operations, deployment, CI changes, real devices, framework migration, and broad visual redesign.
- New product modules or generalized enterprise workflow abstractions.
- Closure of the durable production signing/enforcement note.

## Assumptions

- `AUD-2026-001`, `ORG-XYZ`, `PR-2026-018`, `FR-2026-018`, and `CAB-2026-001` remain the canonical active demo scenario identifiers.
- Browser-local state migration must preserve existing user demo state when safe and reset only invalid split-report structures deterministically.
- The static demo can enforce authority and privacy as executable demo contracts even though it is not production authorization.
- Stakeholder sign-off remains a separate human gate after local GO verification.

## Source And Dependency Map

Read these before implementation:

1. `AGENTS.md`
2. `docs/product-specs/workflows/2026-07-10-stakeholder-readiness-remediation-design.md`
3. `docs/exec-plans/active/2026-07-10-inspector-report-and-governance-workflow-remediation-plan.md`
4. `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md`
5. `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.turkce.md`
6. `docs/product-specs/screen-specs/DEPARTMENT_MANAGER_WORKSPACES.md`
7. `docs/product-specs/screen-specs/DEPARTMENT_MANAGER_WORKSPACES.turkce.md`
8. `docs/product-specs/modules/AUDITEE_PORTAL.md`
9. `docs/product-specs/modules/AUDITEE_PORTAL.turkce.md`
10. `docs/product-specs/ux-plan/NAVIGATION_AND_INFORMATION_ARCHITECTURE.md`
11. `docs/product-specs/ux-plan/NAVIGATION_AND_INFORMATION_ARCHITECTURE.turkce.md`
12. `docs/demo-evidence/BUILD_SUMMARY.md`
13. `docs/demo-evidence/BUILD_SUMMARY.turkce.md`
14. `MANIFEST.md`
15. `docs/exec-plans/index.md`
16. `docs/exec-plans/tech-debt-tracker.md`

## File Responsibility Map

- `js/data.js` — seed distinct canonical report artifacts and migrate persisted demo state.
- `js/reports.js` — exact report/artifact/projection selectors, portal-safe report selectors, preview/PDF state derivation.
- `js/manager-workspaces.js` — Department Manager, GM, and ED decision contracts and read projections.
- `js/helpers.js` — organization/user-aware demo notification creation and visibility.
- `js/inspection.js` — canonical Cabin section/question identity consumed by assignment and execution.
- `js/views.js` — role-safe rendering, exact-ID controls, Lead Final Report state, auditee utility screens, responsive markup.
- `js/app.js` — role routing, action handlers, selected-ID propagation, modal/control behavior, notification calls.
- `css/styles.css` — responsive screen preview, task-order stacking, clipping/overflow, focus and disabled states.
- `index.html` — one final synchronized asset query token bump.
- `tests/*.test.js` — executable regression contracts and full discovery surface.
- English/Turkish product/evidence docs plus plan tracking — synchronized product truth and literal evidence labels.

## Phase Order

1. Lock every reproduced failure with focused negative tests.
2. Repair canonical report state and persistence.
3. Repair authority and closure boundaries.
4. Repair Service Provider privacy and notification scope.
5. Replace stale Lead/Preliminary/assignment paths with exact state.
6. Repair responsive and visible-control behavior.
7. Synchronize docs and tracking.
8. Run independent final verification.

---

### Task 1: Lock The Independent Review Findings With Failing Tests

**Files:**
- Create: `tests/stakeholder-readiness-regressions.test.js`
- Modify: `tests/general-manager-workspace-smoke.test.js`
- Modify: `tests/manager-reports-approval-smoke.test.js`
- Modify: `tests/service-provider-portal-smoke.test.js`
- Modify: `tests/lead-inspector-nav-smoke.test.js`
- Modify: `tests/inspection-execution-smoke.test.js`
- Modify: `tests/department-preliminary-review-smoke.test.js`
- Modify: `tests/manager-workspace-responsive-smoke.test.js`

**Interfaces:**
- Consumes: current `freshState()`, report selectors/decision helpers, render functions, DOM stubs, and action handlers.
- Produces: failing executable contracts for Tasks 2-6 and a single final regression entrypoint.

**Acceptance:** Every Blocking finding fails for the reason observed in the independent review. Existing positive assertions that bless incorrect GM final-authority copy are replaced with intermediate-review expectations.

- [x] **Step 1: Add a failing canonical report identity and mutation-isolation test**

Add assertions equivalent to:

```js
const state = context.freshState();
const preliminary = context.reportArtifactById('PR-2026-018', state);
const finalReport = context.reportArtifactById('FR-2026-018', state);

assert.ok(preliminary);
assert.ok(finalReport);
assert.notEqual(finalReport.id, preliminary.id);
assert.equal(finalReport.reportType, 'Final Report');

finalReport.status = 'submitted_to_executive';
finalReport.ownerRole = 'executiveDirector';
const result = context.applyExecutiveFinalReportDecision(state, finalReport.id, {
  decision: 'approve',
  actor: { role: 'executiveDirector', name: 'Ufuk Aslan' }
});
assert.equal(result.ok, true);
assert.equal(context.reportArtifactById(finalReport.id, state).status, 'issued');
assert.equal(context.reportArtifactById(preliminary.id, state).status, 'draft');
```

- [x] **Step 2: Add failing authority and copy tests**

```js
const before = JSON.stringify(finalReport);
const denied = context.applyExecutiveFinalReportDecision(state, finalReport.id, {
  decision: 'approve',
  actor: { role: 'gm', name: 'Okan Demir' }
});
assert.equal(denied.ok, false);
assert.equal(JSON.stringify(finalReport), before);

assert.match(context.ROLE_DESC.gm, /intermediate|review|forward/i);
assert.doesNotMatch(context.ROLE_DESC.gm, /final report authorization/i);
assert.doesNotMatch(gmHtml, /Approve & Issue|final authorization|issues and locks/i);
assert.match(gmHtml, /Forward to Executive Director/i);
```

- [x] **Step 3: Add failing auditee privacy tests**

```js
state.notifications.unshift({
  id: 'N-PRIVATE-SKY', role: 'auditee', organizationId: 'ORG-SKY',
  text: 'SkyCargo confidential CAP message', time: 'Just now', unread: true
});
state.role = 'auditee';
state.view = 'messages';
assert.doesNotMatch(context.renderContent(), /SkyCargo confidential CAP message/);

state.view = 'settings';
const settingsHtml = context.renderContent();
assert.doesNotMatch(settingsHtml, /Inspector Workload Balance|Oversight Health Index weights|internal risk/i);
```

- [x] **Step 4: Add failing Lead Final Report, assignment, exact-ID, and visible-control tests**

```js
assert.match(leadFinalHtml, /FR-2026-018/);
assert.doesNotMatch(leadFinalHtml, /FR-2026-014|INS-2026-014|SkyCargo/);
assert.match(leadFinalHtml, /data-act="lead-final-report-edit"[^>]*data-id="FR-2026-018"/);

const executionPackage = context.inspectionExecutionPackageForAudit(state, 'AUD-2026-001');
const executionIds = executionPackage.questions.map((question) => question.id);
const assignedIds = Object.keys(state.leadAssignmentsByAudit['AUD-2026-001'].assignmentsByQuestionId);
assert.ok(assignedIds.every((id) => executionIds.includes(id)));

context.state.departmentPreliminaryReviewUi.selectedReportId = 'PR-ISOLATION-CHECK';
context.handleDepartmentPreliminaryApprove('service_provider');
assert.match(context.state.notifications[0].text, /PR-ISOLATION-CHECK/);
```

- [x] **Step 5: Run focused tests and record the expected failures**

Run:

```bash
node tests/stakeholder-readiness-regressions.test.js
node tests/general-manager-workspace-smoke.test.js
node tests/service-provider-portal-smoke.test.js
node tests/lead-inspector-nav-smoke.test.js
node tests/department-preliminary-review-smoke.test.js
node tests/manager-workspace-responsive-smoke.test.js
```

Expected: FAIL on distinct Final artifact, GM actor rejection/copy, auditee organization filtering/settings, canonical Lead Final ID, shared assignment IDs, exact Preliminary ID, and fixed-width preview guards. Record these failures in this plan's execution log; do not weaken the assertions.

**Task 1 RED evidence — 2026-07-10:** `node tests/stakeholder-readiness-regressions.test.js` discovered 10 focused subtests and failed 10 for the intended missing contracts: both IDs resolved to `RPT-AUD-2026-001`; GM could invoke ED authority; non-GM actors could invoke the GM primitive; notification visibility was absent; Service Provider Settings rendered internal OHI/workload material; Lead Final rows were stale/hard-coded; assignment keys were `CAB-Q*` instead of execution `cab-*` IDs; Department Preliminary output used `PR-2026-018` instead of the selected fixture; and the mobile preview containment guards were absent. The focused GM, Service Provider, Lead, Preliminary, responsive, report identity, and execution tests independently reproduced the same failures. One initial execution-test harness TypeError was corrected before accepting RED, after which the test failed on the expected exact-ID mismatch. Production implementation remained unchanged during RED.

---

### Task 2: Establish One Canonical Preliminary And Final Report Lifecycle

**Files:**
- Modify: `js/data.js`
- Modify: `js/reports.js`
- Modify: `js/manager-workspaces.js`
- Modify: `js/views.js`
- Modify: `js/app.js`
- Test: `tests/stakeholder-readiness-regressions.test.js`
- Test: `tests/manager-reports-approval-smoke.test.js`
- Test: `tests/report-approval-smoke.test.js`
- Test: `tests/manager-report-pdf-smoke.test.js`
- Test: `tests/service-provider-final-report-smoke.test.js`

**Interfaces:**
- Consumes: `freshState()`, persisted browser-local state, `auditReports`, `managerReports`, exact Report IDs.
- Produces:
  - `reportArtifactById(reportId, targetState)` returning the exact canonical artifact.
  - `reportProjectionById(reportId, targetState)` returning read-only queue metadata.
  - `reportForAuditAndType(auditId, reportType, targetState)` with no first-match ambiguity.
  - distinct canonical artifacts for `PR-2026-018` and `FR-2026-018`.

**Acceptance:** A decision through any role mutates one canonical artifact; every other role immediately observes the same status/owner/history. Preliminary and Final Report decisions cannot mutate each other.

- [x] **Step 1: Seed distinct canonical artifacts**

Model the Final artifact explicitly instead of sharing the Preliminary package:

```js
{
  id: 'FR-2026-018',
  auditId: 'AUD-2026-001',
  organizationId: 'ORG-XYZ',
  reportType: 'Final Report',
  status: 'pending_manager',
  ownerRole: 'manager',
  issued: false,
  locked: false,
  finalAuthorizedBy: '',
  finalAuthorizedAt: '',
  mockApprovalSignature: null,
  history: []
}
```

Keep Preliminary state on its own `PR-2026-018` canonical artifact. Projection rows may contain display-only values but must reference the same exact canonical ID.

- [x] **Step 2: Normalize persisted split-report state**

Update `freshState()`/migration logic so old state that maps both projections to `RPT-AUD-2026-001` is deterministically expanded into distinct Preliminary and Final artifacts. Preserve safe comments, attachments, timestamps, and history; do not copy issued/locked state from Preliminary to Final or vice versa.

- [x] **Step 3: Make selectors exact and remove interactive first-match fallbacks**

Implement exact behavior:

```js
function reportArtifactById(reportId, targetState) {
  return (targetState.auditReports || []).find((report) => report.id === reportId) || null;
}

function reportProjectionById(reportId, targetState) {
  return (targetState.managerReports || []).find((report) => report.id === reportId) || null;
}
```

`reportForAuditAndType` must return exactly one artifact of the requested type or `null`; interactive handlers must pass Report ID instead of silently selecting another artifact.

- [x] **Step 4: Route all report decisions, preview, template, and PDF reads through canonical state**

Update Department Manager, GM, ED, Lead Inspector, and Service Provider code so status, owner, lock, issue timestamps, counts, team, findings, signature, enforcement referral, preview filename, and PDF content are derived from the exact canonical artifact. Keep `managerReports` mutation-free after initialization/migration.

- [x] **Step 5: Run focused report tests**

Run:

```bash
node tests/stakeholder-readiness-regressions.test.js
node tests/manager-reports-approval-smoke.test.js
node tests/report-approval-smoke.test.js
node tests/manager-report-pdf-smoke.test.js
node tests/service-provider-final-report-smoke.test.js
```

Expected: PASS. Add an assertion that serializing the Preliminary artifact before a Final decision yields identical content afterward.

**Task 2 GREEN evidence — 2026-07-10:** the two canonical subtests in `tests/stakeholder-readiness-regressions.test.js` pass; `manager-reports-approval-smoke`, `report-approval-smoke`, `manager-report-pdf-smoke`, and `service-provider-final-report-smoke` pass. A v6 shared-artifact migration fixture failed on stale `pending_manager` state when migration was intentionally disabled, then `department-manager-state-smoke` passed after restoring the canonical split migration. Remaining stakeholder-regression failures belong to Tasks 3-7 and remain unchanged at this checkpoint.

---

### Task 3: Enforce GM Intermediate Review, ED Final Authority, And Closure Boundaries

**Files:**
- Modify: `js/manager-workspaces.js`
- Modify: `js/app.js`
- Modify: `js/views.js`
- Modify: `js/data.js`
- Modify: `js/approval.js` only where shared stale copy remains
- Test: `tests/general-manager-workspace-smoke.test.js`
- Test: `tests/executive-director-workspace-smoke.test.js`
- Test: `tests/manager-reports-approval-smoke.test.js`
- Test: `tests/report-approval-smoke.test.js`
- Test: `tests/demo-boundary-smoke.test.js`

**Interfaces:**
- Consumes: canonical artifacts from Task 2.
- Produces: role-validated `applyGeneralManagerReportDecision(...)` and `applyExecutiveFinalReportDecision(...)` transitions with non-mutating denial results.

**Acceptance:** GM copy and behavior only return/forward. ED helper rejects all non-ED actors. Approval issues/locks the Final artifact while Findings remain unchanged and the audit becomes `Follow-up Open` whenever an open Finding exists.

- [x] **Step 1: Validate actor roles before any mutation**

Add checks before timestamps/history are created:

```js
if (!actor || actor.role !== 'gm') {
  return generalManagerDecisionResult(false, 'Only the General Manager may record this intermediate review.', report);
}
```

```js
if (!input.actor || input.actor.role !== 'executiveDirector') {
  return executiveReportDecisionResult(false, 'Only the Executive Director may issue, mock-sign, or lock a Final Report.', report);
}
```

Tests must compare full serialized state before/after denied decisions.

- [x] **Step 2: Replace GM final-authority copy and action labels**

Use `Forward to Executive Director`, `Forward Final Report`, `Awaiting GM Review`, and intermediate-review explanations. Remove `Approve & Issue Final Report`, `final authorization`, `issues and locks`, and any toast claiming GM issued the report.

- [x] **Step 3: Keep ED issue/lock and Finding closure separate**

On ED approval:

```js
report.status = 'issued';
report.ownerRole = 'auditee';
report.issued = true;
report.locked = true;
audit.status = openFindings.length ? 'Follow-up Open' : 'Closed';
```

Do not mutate any Finding status. Preserve the explicit CAP acceptance transition to `EVIDENCE_REQUIRED`.

- [x] **Step 4: Remove stale combined `Executive Director / GM` language**

Update role descriptions, data questions, approval rails, dashboard cards, modals, history, notification, toast, and CAP/report distribution copy. GM and ED must never be presented as a shared final authority.

- [x] **Step 5: Run authority and boundary tests**

Run:

```bash
node tests/general-manager-workspace-smoke.test.js
node tests/executive-director-workspace-smoke.test.js
node tests/manager-reports-approval-smoke.test.js
node tests/report-approval-smoke.test.js
node tests/demo-boundary-smoke.test.js
```

Expected: PASS with explicit negative GM-as-ED and report-approval-closure-bypass assertions.

**Task 3 GREEN evidence — 2026-07-10:** the GM wrong-actor and GM-as-ED regression subtests pass. `general-manager-workspace-smoke`, `executive-director-workspace-smoke`, `manager-reports-approval-smoke`, `report-approval-smoke`, and `demo-boundary-smoke` all pass. Exact stale combined-authority searches in `js/` and `tests/` return no match. ED approval preserves open Finding state and sets the audit to `Follow-up Open`.

---

### Task 4: Enforce Service Provider Organization Privacy Across Utility Routes

**Files:**
- Modify: `js/helpers.js`
- Modify: `js/data.js`
- Modify: `js/reports.js`
- Modify: `js/views.js`
- Modify: `js/app.js`
- Test: `tests/service-provider-portal-smoke.test.js`
- Test: `tests/service-provider-final-report-smoke.test.js`
- Test: `tests/stakeholder-readiness-regressions.test.js`

**Interfaces:**
- Consumes: active role, active auditee organization ID, optional Inspector user ID.
- Produces:
  - `pushNotification(role, icon, text, options)` where `options` may contain `organizationId` and `userId`.
  - `notificationVisibleToSession(notification, targetState)` used by topbar counts, panels, Messages, and role workspaces.
  - auditee-safe Settings rendering or a deliberate redirect/removal.

**Acceptance:** Injected other-organization messages never render, never affect counts, and are never relabelled as Fly Namibia. Service Provider utility screens expose no workload/internal-risk/OHI internals.

- [x] **Step 1: Add organization/user identity to notification creation**

Use an explicit options object:

```js
function pushNotification(role, icon, text, options) {
  options = options || {};
  state.notifications.unshift({
    id: 'N' + state.notifSeq++,
    role: role,
    organizationId: options.organizationId || '',
    userId: options.userId || '',
    icon: icon,
    text: text,
    time: 'Just now',
    unread: true
  });
}
```

Update all auditee and Inspector assignment call sites with the canonical organization/user ID.

- [x] **Step 2: Centralize notification visibility**

`notificationVisibleToSession` must require role match, auditee organization match, and Inspector user match when those scopes exist. Use it in unread counts, topbar panel, Messages, and any notification-derived dashboard card.

- [x] **Step 3: Make Service Provider Settings auditee-safe**

Either render a small Service Provider preferences/demo-boundary surface or remove/redirect the route. Do not call the shared internal `computeOHI()` settings renderer for auditee sessions.

- [x] **Step 4: Add privacy-negative render cases**

Test other-org notification text, counts, empty-state metadata, Settings, report attachments, CAP rows, Internal CAA Notes, enforcement deliberations, workload, and internal risk terms.

- [x] **Step 5: Run Service Provider tests**

Run:

```bash
node tests/service-provider-portal-smoke.test.js
node tests/service-provider-final-report-smoke.test.js
node tests/stakeholder-readiness-regressions.test.js
```

Expected: PASS. Confirm Fly Namibia receives only `ORG-XYZ` records and messages.

**Task 4 GREEN evidence — 2026-07-10:** the two privacy regression subtests pass; `service-provider-portal-smoke` and `service-provider-final-report-smoke` pass. Other-organization messages are absent from content and unread counts, Settings contains only Service Provider-safe preferences/boundaries, and direct `reports` navigation resolves to the organization-scoped Documents projection. The direct-route assertion was observed RED with that normalization disabled, then GREEN after restoration.

---

### Task 5: Replace Stale Lead Final Reports And Preliminary First-Match Paths

**Files:**
- Modify: `js/reports.js`
- Modify: `js/views.js`
- Modify: `js/app.js`
- Modify: `js/data.js` only for required UI state defaults/migration
- Test: `tests/lead-inspector-nav-smoke.test.js`
- Test: `tests/lead-inspector-workspace-smoke.test.js`
- Test: `tests/department-preliminary-review-smoke.test.js`
- Test: `tests/manager-report-pdf-smoke.test.js`
- Test: `tests/stakeholder-readiness-regressions.test.js`

**Interfaces:**
- Consumes: exact canonical artifacts and projections from Task 2.
- Produces:
  - `leadFinalReportRows(targetState)` derived from canonical Final Reports.
  - exact `data-id` propagation for every Lead Final action.
  - `handleFinalReportReadyAction(action, reportId)` and exact Preliminary decision handlers.

**Acceptance:** `FR-2026-018`/Fly Namibia stays selected through list, prepare, edit, preview, attachments, inspection record, review, submit, and mock PDF. No remediated Lead/Department screen falls back to `FR-2026-014`, `INS-2026-014`, `PR-2026-018`, or SkyCargo literals when another ID is selected.

- [x] **Step 1: Derive the Lead Final queue from canonical state**

Replace the hard-coded array with mapped canonical artifacts/projections. Each row carries `reportId`, `auditId`, `organizationId`, `organization`, `status`, `ownerRole`, Findings/CAP counts, dates, and exact action eligibility.

- [x] **Step 2: Propagate `reportId` through every visible action**

Render:

```html
<button data-act="final-report-ready-action"
        data-id="FR-2026-018"
        data-final-action="prepare">Prepare Report</button>
```

Every handler reads `data-id`; no action may read a fixed report or inspection ID.

- [x] **Step 3: Make every visible Lead control real or explicitly disabled**

`Edit Report Information`, `View Inspection Record`, preview, attachment management/download, continue, save, review, and submit must update visible state, navigate, open a modal, or generate the intended mock file. Staged features must use `disabled` and explanatory copy instead of inert buttons or toast-only substitutes.

- [x] **Step 4: Use selected Preliminary IDs for decisions, notifications, and audit logs**

Replace `reportForAudit(row.auditId)` and fixed `PR-2026-018` notification/log strings in interactive review paths with the selected exact artifact ID. Add an isolation fixture containing a second Preliminary Report for the same audit.

- [x] **Step 5: Run Lead/Preliminary/PDF tests**

Run:

```bash
node tests/lead-inspector-nav-smoke.test.js
node tests/lead-inspector-workspace-smoke.test.js
node tests/department-preliminary-review-smoke.test.js
node tests/manager-report-pdf-smoke.test.js
node tests/stakeholder-readiness-regressions.test.js
```

Expected: PASS with no stale organization/report/inspection IDs in the selected flows.

**Task 5 GREEN evidence — 2026-07-10:** exact Lead Final and Department Preliminary subtests pass; `department-preliminary-review-smoke`, `lead-inspector-workspace-smoke`, and `manager-report-pdf-smoke` pass. Lead Final rows now derive from canonical Final artifacts, every interactive action carries the exact Report ID, attachment management opens a report-scoped modal with real mock-record downloads, inspection-record navigation opens the linked audit, and the isolation fixture mutates/logs/notifies only `PR-ISOLATION-CHECK`. `lead-inspector-nav-smoke` progressed through these final-report assertions and then stopped at the intentionally failing Task 6 legacy `CAB-Q00N` identity assertion.

---

### Task 6: Connect Multi-Inspector Assignment To Cabin Execution And User Scope

**Files:**
- Modify: `js/inspection.js`
- Modify: `js/views.js`
- Modify: `js/app.js`
- Modify: `js/data.js` for assignment-state migration only if required
- Test: `tests/lead-inspector-nav-smoke.test.js`
- Test: `tests/inspection-team-smoke.test.js`
- Test: `tests/inspection-execution-smoke.test.js`
- Test: `tests/inspector-nav-smoke.test.js`
- Test: `tests/stakeholder-readiness-regressions.test.js`

**Interfaces:**
- Consumes: canonical Cabin package from `inspectionExecutionPackageForAudit(state, auditId)`, inspection team membership, current Inspector user ID.
- Produces: assignments keyed by the exact execution question IDs and an execution projection filtered for the current Inspector.

**Acceptance:** Add Inspector and duplicate prevention still work; assigned questions exist in the six-section Cabin execution package; assigned Inspector sees the intended questions/notification while another Inspector does not inherit user-specific items.

- [x] **Step 1: Build assignment rows from the canonical Cabin package**

Remove the separate `CAB-Q001...` catalog. Map the six canonical sections/questions from `inspection.js` into the Lead assignment UI while preserving display number, text, section, risk, and exact question ID.

- [x] **Step 2: Migrate old browser-local assignment keys**

Map known legacy `CAB-Q00N` keys to canonical execution IDs only when the audit/package match is unambiguous. Drop invalid keys rather than assigning them to another question.

- [x] **Step 3: Filter Inspector execution and notifications by current user**

Use `notificationVisibleToSession` from Task 4. The execution view must show the full audit context but clearly mark/filter assigned questions for the current Inspector according to the chosen demo behavior; it may not silently show another Inspector's personal assignment as its own.

- [x] **Step 4: Repair visible assignment controls and copy**

Use canonical people (`Caner Yildiz`, `Mehmet Kaya`, configured Inspectors). Assignment tabs must navigate/select or be disabled. `Download Assignment Plan` must generate a browser-local mock file or be explicitly disabled; a timestamp-plus-toast alone is not sufficient.

- [x] **Step 5: Run assignment and execution tests**

Run:

```bash
node tests/lead-inspector-nav-smoke.test.js
node tests/inspection-team-smoke.test.js
node tests/inspection-execution-smoke.test.js
node tests/inspector-nav-smoke.test.js
node tests/stakeholder-readiness-regressions.test.js
```

Expected: PASS with exact shared question IDs, per-user visibility, six sections, one-time submit modal, stable timestamp, and Inspector role isolation.

**Task 6 GREEN evidence — 2026-07-10:** `lead-inspector-nav-smoke`, `inspection-team-smoke`, `inspection-execution-smoke`, `inspector-nav-smoke`, and the exact-ID stakeholder subtest pass. Lead assignment rows now come from the six-question Cabin execution package; only the unambiguous legacy PBE key migrates and invalid keys are dropped; per-Inspector scope marks another Inspector's question read-only; released notifications carry `userId`; and Download Assignment Plan generates a browser-local mock text file.

---

### Task 7: Remove Responsive Overflow, Clipping, And Inert Interaction Debt

**Files:**
- Modify: `css/styles.css`
- Modify: `js/views.js`
- Modify: `js/app.js`
- Modify: `index.html`
- Test: `tests/manager-workspace-responsive-smoke.test.js`
- Test: `tests/premium-ui-remediation-smoke.test.js`
- Test: `tests/stakeholder-readiness-regressions.test.js`

**Interfaces:**
- Consumes: corrected markup and selected-report state from Tasks 2-6.
- Produces: screen-responsive report preview with print/PDF page styling preserved and a complete visible-control action inventory.

**Acceptance:** No page-level or nested horizontal overflow, clipped primary/status copy, off-screen decision control, inert visible control, or console error at `1536x864`, `1366x768`, `1024x768`, or `390x844`.

- [x] **Step 1: Add static guards against the reproduced fixed-width mobile rule**

Update responsive tests so the screen media rules reject `.state-final-report-doc { width: 700px; }` and require width/min-width containment for the canvas and zoom stage.

- [x] **Step 2: Reflow or scale the screen preview while preserving print output**

Use screen-specific containment equivalent to:

```css
@media screen and (max-width: 980px) {
  .executive-report-canvas { min-width: 0; overflow-x: hidden; }
  .executive-report-zoom-stage { width: 100%; min-width: 0; }
  .state-final-report-doc {
    width: 100%;
    max-width: 794px;
    min-width: 0;
    box-sizing: border-box;
  }
}
```

Retain controlled print/PDF dimensions under `@media print`; do not solve screen overflow by clipping required content.

- [x] **Step 3: Audit every changed visible control**

Create a checklist mapping label -> `data-act` -> handler -> observable result for GM, ED, Lead Final, assignment, Service Provider Messages/Settings, and report preview. Remove, disable, or implement every orphan control.

- [x] **Step 4: Verify keyboard/modal semantics**

Check visible focus, semantic labels, `aria-pressed`/`aria-expanded`, Escape close, focus return, full-screen mobile modal/drawer behavior, and primary-action reachability.

- [x] **Step 5: Bump all static asset query tokens once**

After CSS/JS behavior is stable, change the stylesheet and all eleven script query tokens in `index.html` to one new exact version string. Update the responsive smoke assertion to the same count/value.

- [x] **Step 6: Run responsive/static tests**

Run:

```bash
node tests/manager-workspace-responsive-smoke.test.js
node tests/premium-ui-remediation-smoke.test.js
node tests/stakeholder-readiness-regressions.test.js
```

Expected: PASS. These static tests are necessary but do not replace Task 9 rendered measurements.

**Task 7 static GREEN evidence — 2026-07-10:** `manager-workspace-responsive-smoke`, `premium-ui-remediation-smoke`, and the stakeholder regression file pass. Screen-only report CSS removes the reproduced 700px mobile width, contains the canvas/zoom/document, and leaves print rules separate. Changed orphan controls are implemented or disabled; modal Escape/focus return already existed and Tab focus containment is now explicit. All twelve static asset references use `20260710-stakeholder-remediation-v14`. Rendered viewport evidence is recorded under Task 9.

---

### Task 8: Synchronize Bilingual Product Truth, Package Inventory, And Plan State

**Files:**
- Modify: `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md`
- Modify: `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.turkce.md`
- Modify: `docs/product-specs/screen-specs/DEPARTMENT_MANAGER_WORKSPACES.md`
- Modify: `docs/product-specs/screen-specs/DEPARTMENT_MANAGER_WORKSPACES.turkce.md`
- Modify: `docs/product-specs/modules/AUDITEE_PORTAL.md`
- Modify: `docs/product-specs/modules/AUDITEE_PORTAL.turkce.md`
- Modify: `docs/product-specs/ux-plan/NAVIGATION_AND_INFORMATION_ARCHITECTURE.md`
- Modify: `docs/product-specs/ux-plan/NAVIGATION_AND_INFORMATION_ARCHITECTURE.turkce.md`
- Modify: `docs/demo-evidence/BUILD_SUMMARY.md`
- Modify: `docs/demo-evidence/BUILD_SUMMARY.turkce.md`
- Modify: `MANIFEST.md`
- Modify: `docs/exec-plans/active/2026-06-28-caa-governance-workflow-and-roles-plan.md`
- Modify: `docs/exec-plans/active/2026-07-10-inspector-report-and-governance-workflow-remediation-plan.md`
- Modify: `docs/exec-plans/active/2026-07-10-stakeholder-readiness-final-remediation-plan.md`
- Modify: `docs/exec-plans/index.md`
- Modify: `docs/exec-plans/tech-debt-tracker.md`
- Test: `tests/harness-docs-smoke.test.js`

**Interfaces:**
- Consumes: verified implementation evidence from Tasks 2-7.
- Produces: one consistent English/Turkish product contract, complete package inventory, and truthful plan lifecycle.

**Acceptance:** No doc claims stale SMS/SkyCargo Inspector behavior, GM final authority, automatic audit closure, completed privacy/responsive evidence without proof, or production readiness. All changed English canonical docs have synchronized Turkish companions.

- [x] **Step 1: Update canonical role and permission truth**

Add Finance, GM, and ED columns/actions to both permission matrices. State Finance approval-only boundaries, GM intermediate review, ED final issue/mock-sign/lock, auditee organization/privacy restrictions, and non-automatic closure/enforcement.

- [x] **Step 2: Correct bilingual workflow and build evidence**

Replace stale SMS/SkyCargo Inspector copy with the Fly Namibia Cabin Inspection flow. Use literal evidence labels only after the corresponding verification has run: `verified locally`, `not run`, `demo-only`, and `production-readiness not claimed`.

- [x] **Step 3: Reconcile MANIFEST with discovery**

Run:

```bash
rg --files tests | sort
```

List every discovered `tests/*.test.js` file exactly once, including `tests/stakeholder-readiness-regressions.test.js`.

- [x] **Step 4: Resolve the local-only Modern Aviation SaaS plan link**

Preserve `docs/exec-plans/active/2026-07-08-modern-aviation-saas-rollout-plan.md` content. Before any remote-completeness claim, ensure either the file is included in a separately authorized Git publication or its canonical index/dependency links are removed and the local-only decision is recorded. Do not leave an `origin/main` plan index link pointing to a path absent from that revision.

- [x] **Step 5: Resolve plan truth conflicts**

Add an erratum to the 2026-06-28 governance plan: ED report approval does not automatically close an audit with open Findings/CAP/Evidence work. Keep the 2026-07-10 predecessor `blocked` until final verification. Keep this follow-up `active` or `ready-for-verification` according to actual evidence only.

- [x] **Step 6: Update the tracker without closing production boundaries**

Keep `Production signing and enforcement authority contract` as `note-open`. Keep the final-remediation blocker note open until Task 9 returns GO; close only that remediation note when evidence and plan/index state are synchronized.

- [x] **Step 7: Run documentation checks**

Run:

```bash
node tests/harness-docs-smoke.test.js
rg -n 'SMS Oversight Audit|SkyCargo|Approve & Issue Final Report|final General Manager authorization|Executive Director / GM' docs js tests
git diff --check
```

Expected: docs smoke PASS; stale terms absent from remediated active paths except clearly labelled historical/negative test fixtures; whitespace PASS.

**Task 8 checkpoint — 2026-07-10:** English/Turkish permission, Department/GM, Auditee Portal, IA, and build-evidence documents now state the canonical identity, authority, privacy, assignment, and closure contracts with literal evidence labels. `MANIFEST.md` lists all 35 discovered test files exactly once. The governance erratum is recorded; predecessor remains `blocked`; this follow-up is `ready-for-verification`; the Modern Aviation SaaS plan is explicitly local-only; production signing/enforcement remains `note-open`. `harness-docs-smoke` and `git diff --check` pass. Stale-term search now returns historical evidence/plans, explicit negative/privacy fixtures, and legacy non-canonical compatibility data; current Fly Namibia report/assignment/portal paths are covered by negative assertions. Fresh rendered evidence remains Task 9.

---

### Task 9: Execute The Independent Final GO/NO-GO Gate

**Files:**
- Modify: `docs/exec-plans/active/2026-07-10-stakeholder-readiness-final-remediation-plan.md` with exact evidence only
- Modify: `docs/exec-plans/active/2026-07-10-inspector-report-and-governance-workflow-remediation-plan.md` only if status changes are supported
- Modify: `docs/exec-plans/index.md`
- Modify: `docs/exec-plans/tech-debt-tracker.md`
- Modify: `docs/demo-evidence/BUILD_SUMMARY.md` and `.turkce.md` only if final evidence changes their claims

**Interfaces:**
- Consumes: all Tasks 1-8 and a clean isolated browser demo state.
- Produces: findings-first independent readout, exact local evidence, final plan/index/tracker state, and stakeholder GO or NO-GO.

**Acceptance:** All automated and rendered checks pass; no Blocking finding remains; push/deploy claims are limited to actions actually authorized and performed. Tests passing alone is not sufficient.

- [x] **Step 1: Run JavaScript syntax checks**

Run:

```bash
for file in js/*.js; do node --check "$file" || exit 1; done
```

Expected: exit 0 for every JavaScript file.

- [x] **Step 2: Run every focused test named by Tasks 1-8**

Run each command individually. Expected: each prints its `: ok` marker or Node test PASS; no test is skipped because the full suite passes.

- [x] **Step 3: Run full discovery and whitespace checks**

Run:

```bash
node --test tests/*.test.js
git diff --check
```

Expected: 0 failed, 0 cancelled, 0 skipped, 0 todo; whitespace exit 0. Record the exact discovered/pass count rather than reusing an old count.

- [x] **Step 4: Run an isolated four-viewport browser matrix**

At `1536x864`, `1366x768`, `1024x768`, and `390x844`, measure both page-level and nested scroll containers. Exercise:

1. Department Manager approves `FR-2026-018` to GM.
2. GM returns with required reason; reset; GM forwards to ED and never issues/locks.
3. A GM actor is rejected by the ED decision primitive without mutation.
4. ED previews the selected Fly Namibia report, validates return/reject/referral, then approves/issues with the demo mark.
5. Open Findings remain open and audit is `Follow-up Open`.
6. Service Provider CAP, Preliminary, Final, Messages, Documents, and Settings show only `ORG-XYZ` auditee-safe data.
7. An injected other-organization message remains absent from content and unread counts.
8. Lead Final list selects `FR-2026-018`; every visible action has an observable result and selected ID remains stable.
9. Multi-Inspector Add Inspector, exact question assignment, release, per-user notification, Inspector execution, six Cabin sections, submit modal, and role isolation work.
10. Department Preliminary second-same-audit fixture keeps the selected Report ID through decision, notification, log, preview, and PDF.

For every route assert:

- zero console warnings/errors
- zero page-level and nested horizontal overflow
- zero clipped action/status text
- primary decisions visible and keyboard reachable
- modal Escape/focus return works
- no stale IDs/organizations
- no inert visible controls
- no privacy, authority, closure, automatic enforcement, or real-signature regression

- [x] **Step 5: Inspect generated mock files and cleanup processes**

Verify PDF headers, filenames, selected Report IDs, Fly Namibia content, template/state linkage, and demo labels. Confirm no local test server, Playwright/Puppeteer/webdriver, headless Chrome, remote-debugging Chrome, or separate browser automation process remains.

- [x] **Step 6: Reconcile final plan lifecycle**

If and only if the independent readout is GO:

- mark this plan `ready-for-verification`
- change the predecessor from `blocked` back to `ready-for-verification` or mark it `superseded` with this plan linked, according to the verified ownership boundary
- update each index row's single next todo
- close the final-remediation tracker note with the exact verification pointer
- keep production signing/enforcement `note-open`

If any Blocking finding remains, keep this plan `active` or `blocked` according to the actual condition and record one concrete next todo.

**Task 9 independent readout — 2026-07-10: GO for stakeholder review of the frontend-only demo.** No Blocking finding remains on the locally exercised surface. The final QA pass found two additional desired-contract regressions before this verdict: the Lead Final list exposed the inspection ID as the primary link instead of canonical `FR-2026-018`, and the `1024x768` report preview had page/nested horizontal overflow. Each was first locked with a failing assertion that failed for the reproduced reason, then fixed with the smallest selected-ID markup and screen-only responsive containment changes. A cache-token assertion also failed before the twelve asset references were moved from v13 to v14. None of the desired-contract assertions was weakened.

Exact local evidence:

- every top-level `js/*.js` file passed `node --check`
- every focused command named by Tasks 1-8 passed individually
- `node --test tests/*.test.js` passed with 44 tests, 0 failed, 0 cancelled, 0 skipped, and 0 todo
- `git diff --check` passed
- fresh isolated-browser QA passed at `1536x864`, `1366x768`, `1024x768`, and `390x844`; document/body width equalled each viewport, remediated report containers had no measured nested horizontal overflow, primary actions were not clipped, and browser console warnings/errors were zero
- GM rendered only intermediate review and no issue/mock-sign/lock controls; ED had final authority but no eligible issue action before the required forward; direct negative actor/mutation assertions passed in Node
- Service Provider Messages, Documents, Final Reports, and Settings stayed Fly-Namibia-scoped with no SkyCargo, `INS-2026-014`, Internal CAA Note, workload, risk, or cross-organization count leakage
- Lead list/readiness/preview retained `FR-2026-018`; the browser PDF action reported `Fly_Namibia_Final_Report_FR-2026-018.pdf`, while focused PDF tests verified the selected report ID, Fly Namibia content, demo label, linked state, and filename
- attachment modal focus wrapped in both directions; Escape closed the modal and returned focus to `Manage Attachments`
- isolated browser tabs, viewport override, and local HTTP server were closed; process cleanup found no separately launched QA browser/server residue

The predecessor is `ready-for-verification`, this plan remains `ready-for-verification` pending stakeholder sign-off, and the final-remediation tracker item is `note-closed`. Production signing/enforcement remains `note-open`. Production, deployed, release, real-identity, legal-signature, immutable audit-log, real enforcement, real-device, penetration, accessibility-certification, and external stakeholder-acceptance surfaces are **not run**. Scope is **demo-only**; **production-readiness not claimed**.

## Verification Matrix

| Contract | Automated evidence | Rendered evidence | Pass condition |
|---|---|---|---|
| Canonical report lifecycle | report regression + approval/PDF tests | DM -> GM -> ED selected record | One artifact state across roles; PR/FR isolated |
| GM/ED authority | negative actor/copy tests | GM return/forward; ED issue | GM cannot issue/sign/lock; ED-only final action |
| CAP/Evidence closure | boundary/approval tests | issued report with open Finding | Finding unchanged; audit `Follow-up Open` |
| Service Provider privacy | other-org negative tests | all auditee routes | No cross-org/private/internal content or counts |
| Lead Final state | Lead/PDF tests | list -> prepare -> preview -> submit | Exact ID/state; no stale content; all controls work |
| Assignment integration | team/execution tests | Lead assign -> Inspector execute | Shared question IDs and per-user scope |
| Preliminary exact ID | second-report isolation test | select -> decision -> preview/PDF | No first-match or hard-coded ID |
| Responsive/accessibility | CSS/static guards | four target viewports | No overflow/clipping/inert controls; keyboard reachable |
| Docs/tracking | docs smoke + searches | content review | English/Turkish and plan state match evidence |

## Risks And Mitigations

- **Persisted browser state drift:** Migration may preserve invalid split state. Mitigate with old-state fixtures and deterministic identity/type validation.
- **Projection mutation regression:** Existing helpers may still mutate `managerReports`. Mitigate by snapshot assertions around every role decision.
- **Privacy through counts/empty states:** Text filtering alone may miss metadata leakage. Mitigate by testing content, unread counts, KPIs, attachments, and selected IDs.
- **Responsive false positive:** Page-level width can pass while a nested canvas overflows. Mitigate by measuring every scroll container at each viewport.
- **Stale positive tests:** Existing tests may assert incorrect GM or orphan-report copy. Replace only after a failing desired-contract assertion proves the intended correction.
- **Documentation overclaim:** Do not write `verified locally` until Task 9 has fresh evidence; use `not run` for production/deployed/real-device surfaces.
- **Scope expansion:** Do not introduce backend, framework, generic workflow engine, or broad visual redesign while remediating these contracts.

## Dependencies

- Tasks 3-6 depend on Task 2 canonical report state.
- Task 4 notification visibility is consumed by Task 6 Inspector user scope.
- Task 7 depends on stable markup/actions from Tasks 3-6.
- Task 8 depends on verified behavior from Tasks 2-7.
- Task 9 depends on every prior task and an independent reviewer who does not rely on the implementer's summary.

## Ownership Boundaries

- Implementation agent: code, tests, docs, local browser evidence, plan/index/tracker updates.
- Independent verifier: fresh source/test/Git/browser inspection and final GO/NO-GO.
- Stakeholder/product owner: final screen/copy acceptance.
- Product/legal/security: production identity, signature, authority, enforcement, audit-log, retention, and security policy.
- User: any authorization to branch, commit, push, deploy, or create a PR.

## Completion Gate

This plan may move to `ready-for-verification` only when:

- all Tasks 1-9 are checked with literal evidence
- every reproduced Blocking finding has a regression test and verified fix
- all material Suggestions are fixed or explicitly accepted with owner/next action
- syntax, focused tests, full discovery, and whitespace pass
- four-viewport rendered QA passes page and nested overflow, console, clipping, navigation, stable ID, privacy, authority, closure, PDF, and visible-control checks
- bilingual docs, MANIFEST, predecessor status, active index, and tracker are synchronized
- external/production/deployed/real-device evidence is labelled `not run`
- no production-readiness claim is made
- the independent findings-first readout returns GO

## Execution Prompt

```text
Implement docs/exec-plans/active/2026-07-10-stakeholder-readiness-final-remediation-plan.md in /Users/marlonjd/Developer/monorepos/avia/apps/surveil.

Start by reading AGENTS.md, docs/product-specs/workflows/2026-07-10-stakeholder-readiness-remediation-design.md, the blocked predecessor plan, and every source document in the plan's Source And Dependency Map. Do not rely on prior task summaries; inspect current code, tests, browser behavior, and Git state yourself.

Execute Tasks 1-9 in order. Add focused regression assertions for canonical report identity, GM/ED role authority, Service Provider organization privacy, state-backed Lead Final Reports, exact Preliminary IDs, assignment/execution IDs, responsive nested overflow, and visible-control behavior before implementing the smallest scoped fix. Do not weaken a desired-contract assertion to make it pass.

Keep the demo frontend-only and Vanilla JavaScript. Preserve Finding -> CAP -> Evidence -> CAA Review -> Closure; CAP acceptance is not closure and report approval must not close open Findings. GM remains intermediate; ED alone may issue/mock-sign/lock. Enforce auditee organization privacy across Messages, Settings, counts, documents, reports, CAP, evidence, and notifications. Remove stale SMS/SkyCargo/INS-2026-014 content from remediated paths. Every visible control must have a real observable result or be clearly disabled.

After implementation, run every focused command in the plan, syntax-check all js/*.js files, run node --test tests/*.test.js, and run git diff --check. Then perform fresh isolated-browser QA at 1536x864, 1366x768, 1024x768, and 390x844. Measure page-level and nested overflow, inspect console output, exercise negative authority/privacy cases, verify selected IDs and PDF content, test keyboard/modal behavior, and clean up all test browser/server processes. Passing Node tests alone is not sufficient.

Update the English/Turkish product specs and build summaries, MANIFEST, predecessor plan status, this plan, docs/exec-plans/index.md, and docs/exec-plans/tech-debt-tracker.md so they match the actual evidence. Use the literal labels verified locally, not run, demo-only, and production-readiness not claimed. Keep the production signing/enforcement note open.

Preserve unrelated dirty/untracked files. Do not create or switch branches, commit, push, deploy, or create a PR unless I separately authorize that exact Git action in the current execution task.

Finish with a findings-first independent GO/NO-GO readout covering Tasks 1-9, remaining risks, exact test/browser evidence, plan/index/tracker consistency, repository status, and any unverified production or external surfaces.
```
