# Browser Scenario Integrity Remediation Implementation Plan


**Goal:** Make the AviaSurveil360 demo use one canonical, audit-scoped lifecycle from Planning through checklist execution, Finding, CAP, Evidence, authorized review, closure, reporting, reminders, and Auditee visibility, with every visible action verified in a real localhost browser.

**Architecture:** Preserve the frontend-only HTML/CSS/Vanilla JavaScript architecture and browser-local mock state. Replace parallel synthetic workflow state with canonical `Audit`, audit-scoped checklist answer, `Potential Finding`, `Finding`, `CAP`, `Evidence`, report, notification, and audit-log records; put role and transition checks in shared domain helpers used by both UI actions and tests. Keep the controlled Cabin scenario as the primary story while making every visible non-Cabin start/continue control either operate on a real mock checklist package or present an explicitly disabled, non-interactive boundary.

**Tech Stack:** HTML, CSS, Vanilla JavaScript, browser-local mock state, Node `node:test`/assert smoke tests, localhost HTTP serving, Codex in-app Browser real-click verification.

**Status:** `superseded` — retained as historical `candidate-only` evidence.
A fresh 70-check real-click audit on 2026-07-20 found 13 residual defects
(1 Blocking, 6 High, 6 Medium), so its earlier completeness claims are not the
current remediation authority. The successor is now completed:
[`2026-07-20-working-scenario-audit-remediation-plan.md`](../completed/2026-07-20-working-scenario-audit-remediation-plan.md).
Production readiness is not claimed.

## Global Constraints

- Use `AviaSurveil360` as the canonical product name.
- Remain frontend-only and demo-only: no backend, database, API, real authentication, production authorization service, real file storage, real email/SMS/WhatsApp delivery, or framework migration.
- Preserve unrelated working-tree changes. The repository is already dirty and changed during the read-only audit; inspect before every edit and never overwrite another actor's work.
- Work on the current branch. Do not create, switch, rename, or delete branches; do not commit or push unless the user separately asks for that exact Git action.
- Use `Finding -> CAP -> Evidence -> CAA Review -> Closure` as the canonical lifecycle.
- CAP acceptance must never close a Finding by itself.
- Separate `Comment to Auditee` from `Internal CAA Note` in state, UI, notifications, reports, and tests.
- Preserve Evidence versions; never overwrite an earlier Evidence record.
- Keep regulatory wording as configured references and demo finding bases, not legal advice or automatic enforcement.
- `Oversight Health Index` remains an indicator only and cannot make closure, enforcement, certificate, or legal decisions.
- Use `Due Date`, `Target`, `Due Soon`, and `Overdue` language.
- Keep English implementation and canonical documentation; update matching `.turkce.md` companions whenever a stakeholder-facing canonical document changes.
- Run the browser only through `http://127.0.0.1:<port>`; never use `file://`.
- Clean up the local HTTP server and task-owned Browser/Chrome automation processes before completion.
- Do not mark the plan completed until the active index, verification evidence, and technical-debt tracker match the actual final state.

---

## Objective

Remediate the integrity failures proven by the 2026-07-20 browser audit:

1. Audit-scope checklist answers and Potential Findings so identical question IDs cannot collide across Audits.
2. Remove synthetic CAP/Finding state and make Inspector, Lead Inspector, Department Manager, Auditee, Evidence, report, and dashboard screens read the same canonical records.
3. Enforce view and action authority for Inspector, Lead Inspector, Department Manager, GM, Executive Director, Finance, Auditee, and Administration roles.
4. Stop Inspector CAP acceptance from auto-filling later approvals, preparing Final Reports, or changing role.
5. Make Department Manager closure decisions mutate canonical lifecycle state only when the required evidence/authorized-closure contract allows it.
6. Make the seed Cabin Planning item complete and provide a scoped post-release Lead preparation route through executable Audit materialization.
7. Make every visible checklist start/continue action truthful and audit-specific across Cabin, Security, Ramp, Airworthiness, and Flight Operations.
8. Make Observation CAP/Evidence requirements configurable and non-mandatory by default.
9. Require authorized role, lifecycle stage, and reason when reopening a submitted checklist.
10. Distinguish evidence-verified closure from authorized closure in reports and Auditee timelines.
11. Provide deterministic browser-local 30/15/7/due-today/overdue/manager-escalation events without claiming real delivery.
12. Align the main demo scenario, test contracts, and browser evidence with the implemented lifecycle.

## Scope

### In Scope

- Canonical state helpers and persistence normalization for checklist answers, Potential Findings, Findings, CAPs, Evidence, closure decisions, reminders, and planning materialization.
- Central view authorization plus action-level authority and transition guards.
- Removal or canonical derivation of `F-014-*` and `F-2026-*` UI-only records.
- Deterministic non-Cabin mock checklist packages for every template offered as runnable.
- Seed Planning normalization for inspection category, notice policy, application type, template, scope, location, mode, and planned date.
- A scoped Lead Inspector preparation route or task surface without granting unrelated Department Manager Planning authority.
- Routine/Announced coordination date propagation and Ad Hoc/Unannounced notice withholding.
- Real browser verification for all roles and the scenario matrix in this plan.
- Focused and full automated regression tests.
- Matching English/Turkish stakeholder documentation and demo evidence updates.
- Plan index and technical-debt tracker synchronization.

### Out Of Scope

- Production backend, database, API, authentication, authorization server, tenant service, file storage, messaging provider, or reporting engine.
- Real regulatory ingestion, automatic legal conclusions, automatic enforcement, certificate action, or signed decisions.
- Production-grade immutable audit log, evidence retention, malware scanning, encryption, recovery, or records-management policy.
- Mobile offline application, advanced BI, AI decision-making, or framework migration.
- Broad visual redesign unrelated to fixing misleading, inaccessible, or broken controls.
- Rewriting historical source/reference materials.
- Branch, commit, push, pull-request, GitHub comment, issue, or release actions.

## Assumptions

- `state.findings` remains the single Finding source of truth.
- `state.audits`, `state.inspectionWorkspaces`, `state.inspectionTeams`, `state.inspectionCoordinations`, `state.reportArtifacts`, `state.notifications`, and `state.auditLog` remain canonical collections.
- Audit-scoped checklist answers live under `state.inspectionWorkspaces[auditId].answersByQuestionId[questionId]`; a global question-only map is not canonical.
- Existing dirty changes belong to the user or another active actor; implementation must re-read touched hunks immediately before patching.
- The accepted Planning chain remains `Department Manager -> Finance Review -> GM Review -> Executive Director -> GM Release to Department -> Department acceptance -> Lead preparation -> Department confirmation -> execution`.
- A Lead Inspector may receive a scoped preparation task for one released plan without receiving broad Department Manager Planning controls.
- Observation defaults to `capRequired: false` and `evidenceRequired: false`; the Lead Inspector may explicitly enable either when converting a Potential Finding.
- Evidence verification result `close` produces canonical closure type `evidence-verified`; only the separate reason-required CAA action produces `authorized` closure.
- Browser-local reminder events are deterministic derived records, not evidence of real delivery.
- The final automated test count may exceed 72 because this plan adds focused regression coverage; success means zero failures, not preserving a historic count.

## Dependencies

- Repo-local `AGENTS.md` and the source-of-truth documents listed there.
- Current helpers in `js/inspection.js`, `js/planning.js`, `js/approval.js`, `js/reports.js`, `js/work-items.js`, and `js/manager-workspaces.js`.
- Current render/action routing in `js/app.js` and `js/views.js`.
- Existing smoke-test VM harness pattern under `tests/`.
- A local Python or equivalent static HTTP server and Codex in-app Browser.
- No external service or network dependency.

## Ownership Boundaries

- Department Manager: Planning intake, released-plan acceptance, Lead assignment, preparation confirmation, department report decisions, and authorized management review within configured demo rules.
- Finance Review: budget decision only.
- General Manager: intermediate plan/report review and approved-plan release to Department.
- Executive Director: final plan/report decision only; approval does not silently execute or release.
- Lead Inspector: scoped plan preparation, team/date/resource proposal, Potential Finding decision, report preparation, and assigned-audit coordination.
- CAA Inspector: assigned checklist execution, CAP review, Evidence review, and permitted finding/evidence actions; no management approval authority.
- Auditee: own-organization CAP, Evidence, coordination, messages, CAA-visible comments, and closure status only.
- Administration: templates and configured rules only; no operational closure authority.

## File Map

### Runtime

- Modify `js/data.js`: canonical seed fields, persistence normalization, mock checklist packages, closure/reminder constants, and removal/migration of obsolete UI-only state.
- Modify `js/inspection.js`: audit-scoped answer access, Potential Finding isolation, Observation flags, submit/reopen guards, and transition helpers.
- Modify `js/planning.js`: seed normalization, scoped Lead preparation, materialization invariants, coordination date propagation, and idempotency.
- Modify `js/approval.js`: reuse existing role-aware decision primitives; add no UI-specific state.
- Modify `js/reports.js`: canonical report lifecycle inputs and exact approval authority where required.
- Modify `js/work-items.js`: derived reminder/escalation work items and canonical row composition.
- Modify `js/manager-workspaces.js`: canonical CAP/closure/report rows only.
- Modify `js/app.js`: central route/action guards, canonical handlers, reminder derivation, no silent role changes, and no UI-only closure mutation.
- Modify `js/views.js`: truthful controls, canonical CAP/Finding screens, dynamic closure labels/timeline, reason-required reopen, scoped Lead preparation, and supported-template behavior.
- Modify `css/styles.css`: only styles needed for disabled template states, guarded action feedback, reopen dialog, and reminder timeline readability.
- Modify `index.html`: asset version only if browser cache validation proves it is required.

### Tests

- Create `tests/scenario-integrity-regression.test.js`: P0 canonical state, role, collision, closure, and false-contract regressions.
- Create `tests/browser-scenario-contract-smoke.test.js`: rendered action/route contract for all roles without browser automation.
- Modify `tests/inspection-execution-smoke.test.js`.
- Modify `tests/audit-work-queue-smoke.test.js`.
- Modify `tests/inspector-nav-smoke.test.js`.
- Modify `tests/lead-inspector-nav-smoke.test.js`.
- Modify `tests/lead-inspector-workspace-smoke.test.js`.
- Modify `tests/planning-release-smoke.test.js`.
- Modify `tests/planning-workspace-smoke.test.js`.
- Modify `tests/unannounced-inspection-intake-smoke.test.js`.
- Modify `tests/inspection-coordination-smoke.test.js`.
- Modify `tests/service-provider-portal-smoke.test.js`.
- Modify `tests/department-manager-findings-smoke.test.js`.
- Modify `tests/manager-cap-monitoring-smoke.test.js`.
- Modify `tests/manager-reports-approval-smoke.test.js`.
- Modify `tests/demo-boundary-smoke.test.js`.

### Documentation And Tracking

- Modify `docs/product-specs/scenarios/DEMO_SCENARIO_OPERATOR_AUDIT.md` and `.turkce.md`.
- Modify `docs/product-specs/workflows/AUDIT_CHECKLIST_WORKFLOW.md` and `.turkce.md`.
- Modify `docs/product-specs/workflows/FINDING_CAP_EVIDENCE_WORKFLOW.md` and `.turkce.md`.
- Modify `docs/product-specs/workflows/REMINDERS_AND_ESCALATION_WORKFLOW.md` and `.turkce.md`.
- Modify `docs/product-specs/modules/NOTIFICATIONS_AND_REMINDERS.md` and `.turkce.md`.
- Modify `docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.md` and `.turkce.md`.
- Modify `docs/demo-evidence/BUILD_SUMMARY.md` and `.turkce.md`.
- Create `docs/demo-evidence/BROWSER_SCENARIO_INTEGRITY_2026-07-20.md` and `.turkce.md` only after fresh verification succeeds.
- Modify `MANIFEST.md` if the evidence inventory changes.
- Modify this plan, `docs/exec-plans/index.md`, and `docs/exec-plans/tech-debt-tracker.md` as execution status changes.

---

## Phase 1 — Freeze The Baseline And Repair P0 Data Integrity

### Task 1: Stabilize The Dirty Worktree And Add Failing Regression Contracts

**Files:**

- Create: `tests/scenario-integrity-regression.test.js`
- Create: `tests/browser-scenario-contract-smoke.test.js`
- Modify: no runtime file in this task

**Interfaces:**

- Consumes: current `freshState()`, `recordChecklistResult()`, `createPotentialFinding()`, route normalization, CAP decision handlers, and report renderers.
- Produces: deterministic failing tests for every P0 and P1 state invariant before implementation starts.

- [x] **Step 1: Capture and preserve the working-tree baseline.**

  Run:

  ```bash
  git status --short --branch
  git diff -- js/data.js js/inspection.js js/planning.js js/app.js js/views.js tests
  ```

  Expected: inspect existing user changes; do not reset, restore, stage, commit, or overwrite them.

- [x] **Step 2: Create the VM test harness with the exact runtime load order.**

  Add this loader to `tests/scenario-integrity-regression.test.js`:

  ```js
  const assert = require('node:assert/strict');
  const fs = require('node:fs');
  const path = require('node:path');
  const vm = require('node:vm');

  const root = path.resolve(__dirname, '..');
  const context = { console, window: undefined, document: undefined, setTimeout, clearTimeout };
  vm.createContext(context);
  [
    'js/data.js',
    'js/helpers.js',
    'js/approval.js',
    'js/planning.js',
    'js/checklists.js',
    'js/inspection.js',
    'js/reports.js',
    'js/manager-workspaces.js',
    'js/work-items.js'
  ].forEach((file) => vm.runInContext(fs.readFileSync(path.join(root, file), 'utf8'), context, { filename: file }));
  ```

- [x] **Step 3: Add failing P0 assertions.**

  The focused test must assert:

  ```js
  const state = context.freshState();
  const auditA = 'AUD-2026-001';
  const auditB = 'AUD-2026-006';
  state.audits.push(Object.assign({}, state.audits[0], { id: auditB }));

  context.state = state;
  context.recordChecklistResult(auditA, 'CAB-EME-01', 'noncompliant', 'Audit A exception', []);
  context.recordChecklistResult(auditB, 'CAB-EME-01', 'observation', 'Audit B observation', []);

  assert.equal(context.checklistAnswerForAudit(state, auditA, 'CAB-EME-01').comment, 'Audit A exception');
  assert.equal(context.checklistAnswerForAudit(state, auditB, 'CAB-EME-01').comment, 'Audit B observation');
  assert.notEqual(
    context.createPotentialFinding(auditA, 'CAB-EME-01').id,
    context.createPotentialFinding(auditB, 'CAB-EME-01').id
  );
  ```

  Add separate assertions that Inspector CAP acceptance does not change role, does not populate later approval timestamps, and does not make a Final Report ready; Department Manager approval does not mutate only `capTrackingUi`; and Inspector cannot render or execute `unit-manager-review`.

- [x] **Step 4: Add failing rendered-contract assertions.**

  `tests/browser-scenario-contract-smoke.test.js` must load `js/views.js` and `js/app.js` with the existing DOM stub pattern, then assert:

  ```js
  assert.doesNotMatch(inspectorHtml, /Approve Closure Decision/);
  assert.doesNotMatch(inspectorHtml, /Open Department Manager Approval/);
  assert.doesNotMatch(inspectorAcceptHtml, /Final Report is ready/);
  assert.match(submittedChecklistHtml, /Reopen for Editing/);
  assert.match(submittedChecklistHtml, /Reason for reopening/);
  assert.match(evidenceClosedReportHtml, /Evidence accepted and verified/);
  assert.doesNotMatch(evidenceClosedReportHtml, /Authorized closure \(audit-logged\)/);
  ```

- [x] **Step 5: Run focused tests and record the expected failures.**

  Run:

  ```bash
  node --test tests/scenario-integrity-regression.test.js tests/browser-scenario-contract-smoke.test.js
  ```

  Expected: FAIL on missing audit-scoped answer helper, missing role/action guard, silent approval chain, reopen reason, and closure-label assertions.

### Task 2: Make Checklist Answers And Potential Findings Audit-Scoped

**Files:**

- Modify: `js/inspection.js`
- Modify: `js/data.js`
- Modify: `js/app.js`
- Modify: `js/views.js`
- Modify: `tests/scenario-integrity-regression.test.js`
- Modify: `tests/inspection-execution-smoke.test.js`

**Interfaces:**

- Produces: `checklistAnswerForAudit(target, auditId, questionId)`, audit-scoped `recordChecklistResult()`, and audit-scoped `createPotentialFinding()`.
- Consumes: `inspectionWorkspaceForAudit(target, auditId)` and `workspace.answersByQuestionId`.

- [x] **Step 1: Add the canonical answer accessor.**

  Implement in `js/inspection.js`:

  ```js
  function checklistAnswerForAudit(target, auditId, questionId) {
    var workspace = inspectionWorkspaceForAudit(target, auditId);
    return workspace.answersByQuestionId[questionId] || null;
  }
  ```

- [x] **Step 2: Write through the audit workspace, not the global question map.**

  Replace question-only storage with:

  ```js
  var workspace = inspectionWorkspaceForAudit(state, auditId);
  var previous = workspace.answersByQuestionId[questionId] || {};
  workspace.answersByQuestionId[questionId] = Object.assign({}, previous, {
    auditId: auditId,
    questionId: questionId,
    answer: answer,
    comment: note,
    evidenceFiles: normalizeMockFiles(files)
  });
  return workspace.answersByQuestionId[questionId];
  ```

  Do not keep a mutable global `state.checklistAnswers[questionId]` compatibility write.

- [x] **Step 3: Make Potential Finding creation read and write the same audit-scoped answer.**

  Use:

  ```js
  var answer = checklistAnswerForAudit(state, auditId, questionId);
  if (!answer) throw new Error('Checklist answer required before creating a Potential Finding.');
  if (answer.potentialFindingId) return potentialFindingById(answer.potentialFindingId);
  ```

  Persist `potentialFindingId` only on that audit workspace answer.

- [x] **Step 4: Normalize legacy browser state without cross-audit guessing.**

  During persistence normalization, copy a legacy answer only when it has a non-empty `auditId`; otherwise leave it unimported and add a demo audit-log migration note. Never assign one legacy answer to multiple Audits.

- [x] **Step 5: Run collision and existing inspection tests.**

  Run:

  ```bash
  node --test tests/scenario-integrity-regression.test.js tests/inspection-execution-smoke.test.js tests/checklist-comment-render-smoke.test.js
  ```

  Expected: two Audits retain distinct answers and distinct Potential Findings; existing Cabin execution tests pass.

### Task 3: Centralize View And Action Authority

**Files:**

- Modify: `js/app.js`
- Modify: `js/inspection.js`
- Modify: `js/reports.js`
- Modify: `tests/browser-scenario-contract-smoke.test.js`
- Modify: `tests/inspector-nav-smoke.test.js`
- Modify: `tests/lead-inspector-nav-smoke.test.js`
- Modify: `tests/department-manager-findings-smoke.test.js`

**Interfaces:**

- Produces: `roleCanOpenView(role, view)`, `requireActionRole(allowedRoles, actionLabel)`, and guarded `go()`/`renderContent()` paths.
- Consumes: role-specific allowed/restricted view maps and existing transition helpers.

- [x] **Step 1: Replace negative-only route checks with one role authorization function.**

  Implement:

  ```js
  function roleCanOpenView(role, view) {
    if (view === 'unit-manager-review') return role === 'manager';
    if (view === 'lead-planning-preparation') return role === 'leadInspector';
    if (role === 'inspector') return !INSPECTOR_RESTRICTED_VIEWS[view];
    if (role === 'leadInspector') return !LEAD_INSPECTOR_RESTRICTED_VIEWS[view];
    if (role === 'gm') return !!GENERAL_MANAGER_ALLOWED_VIEWS[view];
    if (role === 'auditee') return !!AUDITEE_ALLOWED_VIEWS[view];
    return true;
  }
  ```

  Apply it both before navigation mutation and before view rendering. A denied view must redirect to the role home and add no domain mutation.

- [x] **Step 2: Guard sensitive action handlers independently of visibility.**

  Add:

  ```js
  function requireActionRole(allowedRoles, actionLabel) {
    if (allowedRoles.indexOf(state.role) === -1) {
      throw new Error(actionLabel + ' is not authorized for ' + state.role + '.');
    }
  }
  ```

  Use it for checklist reopen, Lead Potential Finding decisions, Inspector CAP/Evidence decisions, Department Manager closure approval, GM decisions, ED decisions, and authorized closure.

- [x] **Step 3: Remove the test-only direct Inspector management route contract.**

  Replace the block in `tests/inspector-nav-smoke.test.js` that directly assigns `state.view = 'unit-manager-review'` with denial assertions:

  ```js
  context.state.role = 'inspector';
  context.go('unit-manager-review', { findingId: canonicalFinding.id });
  assert.equal(context.state.view, context.homeView('inspector'));
  assert.doesNotMatch(elements.get('app-root').innerHTML, /Approve Closure Decision/);
  ```

- [x] **Step 4: Run authority tests.**

  Run:

  ```bash
  node --test tests/browser-scenario-contract-smoke.test.js tests/inspector-nav-smoke.test.js tests/lead-inspector-nav-smoke.test.js tests/department-manager-findings-smoke.test.js
  ```

  Expected: every unauthorized route/action is mutation-free and every authorized role retains its intended workspace.

### Task 4: Replace Synthetic CAP State With Canonical Finding Lifecycle

**Files:**

- Modify: `js/data.js`
- Modify: `js/inspection.js`
- Modify: `js/manager-workspaces.js`
- Modify: `js/app.js`
- Modify: `js/views.js`
- Modify: `tests/scenario-integrity-regression.test.js`
- Modify: `tests/inspector-nav-smoke.test.js`
- Modify: `tests/manager-cap-monitoring-smoke.test.js`
- Modify: `tests/department-manager-findings-smoke.test.js`
- Modify: `tests/service-provider-portal-smoke.test.js`

**Interfaces:**

- Produces: canonical `findingById()`-backed CAP review rows and canonical decision mutations.
- Removes: `F-014-*` and `F-2026-*` arrays as independent operational state.

- [x] **Step 1: Derive every CAP row from canonical Findings.**

  Replace hard-coded arrays with a mapper whose output keeps the existing view shape:

  ```js
  function capReviewRowFromFinding(finding) {
    return {
      id: finding.id,
      detailId: finding.id,
      title: finding.title,
      organization: orgName(finding.orgId),
      statusKey: String(finding.status || '').toLowerCase(),
      dueDateText: finding.dueDate ? fmtDate(finding.dueDate) : 'No Due Date',
      owner: roleName(derivedStatus(finding).ownerRole)
    };
  }
  ```

- [x] **Step 2: Make Inspector CAP acceptance canonical and bounded.**

  On accept, mutate only the selected Finding and latest CAP revision:

  ```js
  finding.cap.status = 'Accepted';
  finding.status = finding.evidenceRequired ? 'EVIDENCE_REQUIRED' : 'READY_FOR_VERIFICATION';
  finding.capAcceptedAt = logTimestamp();
  ```

  Do not populate Lead/Department Manager/GM timestamps, do not set Final Report ready, and do not change `state.role`.

- [x] **Step 3: Make return/revision decisions canonical.**

  A return must require a `Comment to Auditee`, set `CAP_MORE_INFO`, append a CAP revision/history entry, create an organization-scoped Auditee notification, and leave internal notes private.

- [x] **Step 4: Make Department Manager approval update canonical approval state without bypassing closure rules.**

  Record:

  ```js
  finding.managementReview = {
    decision: decision,
    decidedBy: currentActorLabel(),
    decidedAt: logTimestamp(),
    note: decisionNote
  };
  ```

  Close only through the canonical evidence-verification or authorized-closure helper. If required Evidence is not accepted, retain an open status and show the precise next action.

- [x] **Step 5: Delete false-contract assertions.**

  Remove assertions that expect silent `leadInspector` role switching or auto-filled later approvals. Add assertions that all role screens resolve the same Finding ID and status.

- [x] **Step 6: Run canonical CAP tests.**

  Run:

  ```bash
  node --test tests/scenario-integrity-regression.test.js tests/inspector-nav-smoke.test.js tests/manager-cap-monitoring-smoke.test.js tests/department-manager-findings-smoke.test.js tests/service-provider-portal-smoke.test.js
  ```

  Expected: one Finding ID and one lifecycle status are visible across all roles; no UI-only closure state remains.

---

## Phase 2 — Repair Planning, Coordination, And Checklist Execution

### Task 5: Complete Seed Planning And Scoped Lead Preparation

**Files:**

- Modify: `js/data.js`
- Modify: `js/planning.js`
- Modify: `js/app.js`
- Modify: `js/views.js`
- Modify: `tests/audit-work-queue-smoke.test.js`
- Modify: `tests/lead-inspector-nav-smoke.test.js`
- Modify: `tests/lead-inspector-workspace-smoke.test.js`
- Modify: `tests/planning-release-smoke.test.js`
- Modify: `tests/planning-workspace-smoke.test.js`
- Modify: `tests/unannounced-inspection-intake-smoke.test.js`

**Interfaces:**

- Produces: complete seed Planning records, `lead-planning-preparation` scoped route, and idempotent Audit materialization.
- Consumes: existing approval/release/preparation helpers and `materializeReadyPlanningInspection()`.

- [x] **Step 1: Add the complete execution contract to every seed plan.**

  For `PLAN-2026-Q3-CABIN`, persist:

  ```js
  applicationType: 'Continued Surveillance',
  inspectionCategory: 'Routine / Announced',
  noticePolicy: 'advance',
  templateId: 'TPL-CABIN-2026',
  plannedDate: '2026-09-10',
  mode: 'On-site',
  location: 'Fly Namibia HQ',
  scope: 'Cabin emergency equipment serviceability oversight.',
  auditId: ''
  ```

  Apply equivalent explicit fields to Flight Operations and Airworthiness seeds.

- [x] **Step 2: Normalize existing browser-saved plans.**

  Fill only missing fields from the matching seed item. Preserve user-entered values and approval/preparation history.

- [x] **Step 3: Expose one scoped Lead preparation task.**

  Add a Lead notification/task action that navigates to:

  ```js
  go('lead-planning-preparation', { planningItemId: item.id });
  ```

  Render only the selected plan's approved scope, Lead/team/date/resource proposal, and history. Do not expose Department Manager approval, budget, release, or unrelated plan controls.

- [x] **Step 4: Resolve contradictory navigation tests.**

  Keep broad `Planning` absent from Lead navigation and test the scoped action instead:

  ```js
  assert.doesNotMatch(leadNavLabels, /Planning/);
  assert.match(leadHomeHtml, /Continue plan preparation/);
  assert.match(scopedPreparationHtml, /Propose Team \/ Dates \/ Resources/);
  assert.doesNotMatch(scopedPreparationHtml, /Release to Department|Approve Budget|Approve & Sign/);
  ```

- [x] **Step 5: Confirm and materialize the seed hero plan.**

  After Department Manager confirmation, assert one Audit, one inspection team, one inspection workspace, and one coordination record. Repeating confirmation must not duplicate any record.

- [x] **Step 6: Run the full Planning-focused suite.**

  Run:

  ```bash
  node --test tests/audit-work-queue-smoke.test.js tests/lead-inspector-nav-smoke.test.js tests/lead-inspector-workspace-smoke.test.js tests/planning-release-smoke.test.js tests/planning-workspace-smoke.test.js tests/unannounced-inspection-intake-smoke.test.js
  ```

  Expected: zero contradictory Lead navigation assertions; both seed Cabin and newly created Ad Hoc plans reach deterministic materialization.

### Task 6: Propagate Coordination Decisions And Preserve Notice Privacy

**Files:**

- Modify: `js/planning.js`
- Modify: `js/app.js`
- Modify: `js/views.js`
- Modify: `tests/inspection-coordination-smoke.test.js`
- Modify: `tests/service-provider-portal-smoke.test.js`

**Interfaces:**

- Produces: canonical accepted inspection date and organization-scoped coordination visibility.

- [x] **Step 1: Update the canonical Audit when an alternative date is accepted.**

  Apply:

  ```js
  coordination.confirmedDate = coordination.alternativeDate;
  audit.date = coordination.confirmedDate;
  audit.endDate = coordination.confirmedDate;
  ```

  Add a coordination-history entry and preserve the original proposed date in history.

- [x] **Step 2: Keep Ad Hoc notice withheld.**

  Assert `noticePolicy === 'withheld'`, `status === 'notice_withheld'`, empty `notifiedAt`, no Auditee notification, and no Service Provider coordination row.

- [x] **Step 3: Verify Routine alternate-date round trip.**

  Assert the Lead Audit header, Service Provider coordination page, Inspector assignment, and report preparation all display the accepted date.

- [x] **Step 4: Run coordination/privacy tests.**

  Run:

  ```bash
  node --test tests/inspection-coordination-smoke.test.js tests/service-provider-portal-smoke.test.js tests/unannounced-inspection-intake-smoke.test.js
  ```

  Expected: accepted Routine date is consistent; Ad Hoc records remain private.

### Task 7: Make Every Visible Checklist Action Truthful And Audit-Specific

**Files:**

- Modify: `js/data.js`
- Modify: `js/checklists.js`
- Modify: `js/inspection.js`
- Modify: `js/app.js`
- Modify: `js/views.js`
- Modify: `tests/inspection-execution-smoke.test.js`
- Modify: `tests/audit-work-queue-smoke.test.js`
- Modify: `tests/checklist-management-smoke.test.js`

**Interfaces:**

- Produces: published demo checklist packages for `TPL-CABIN-2026`, `TPL-FOPS-2026`, `TPL-AIRW-2026`, `TPL-RAMP-2026`, and `TPL-SEC-2026`, or a visibly disabled action when a package is intentionally unavailable.

- [x] **Step 1: Add deterministic minimal packages for advertised runnable templates.**

  Each non-Cabin package must contain at least one section and three configured-reference questions, unique question IDs, allowed outcomes, and demo expected-evidence text. Example Security IDs:

  ```js
  {
    templateId: 'TPL-SEC-2026',
    sections: [{
      id: 'SEC-ACCESS',
      title: 'Access Control',
      questions: [
        { id: 'SEC-ACCESS-01', text: 'Are configured restricted-area access controls applied?', ref: 'Configured security reference SEC-ACCESS-01' },
        { id: 'SEC-ACCESS-02', text: 'Are sampled visitor records complete?', ref: 'Configured security reference SEC-ACCESS-02' },
        { id: 'SEC-ACCESS-03', text: 'Are access exceptions reviewed and recorded?', ref: 'Configured security reference SEC-ACCESS-03' }
      ]
    }]
  }
  ```

  Use equally careful configured-reference wording for Ramp, Airworthiness, and Flight Operations.

- [x] **Step 2: Route row actions by exact Audit ID.**

  Every Start/Continue button must carry its canonical `audit.id`; remove fallback-to-Cabin behavior when an ID is absent or unknown.

- [x] **Step 3: Disable intentionally unsupported controls.**

  If any template remains unavailable after this task, render a disabled button with `aria-disabled="true"` and explicit `Template preview only`; do not render a working-looking Start/Continue control.

- [x] **Step 4: Add per-template execution tests.**

  For each template, open the exact Audit, record one Compliant and one exception answer, and assert the workspace, progress, Potential Finding, and Audit ID remain isolated.

- [x] **Step 5: Run checklist tests.**

  Run:

  ```bash
  node --test tests/inspection-execution-smoke.test.js tests/audit-work-queue-smoke.test.js tests/checklist-management-smoke.test.js tests/scenario-integrity-regression.test.js
  ```

  Expected: Cabin, Security, Ramp, Airworthiness, and Flight Operations open their own packages with no fallback or collision.

---

## Phase 3 — Repair Observation, Reopen, Closure, Reports, And Reminders

### Task 8: Make Observation Requirements Configurable

**Files:**

- Modify: `js/inspection.js`
- Modify: `js/app.js`
- Modify: `js/views.js`
- Modify: `tests/scenario-integrity-regression.test.js`
- Modify: `tests/inspection-execution-smoke.test.js`

**Interfaces:**

- Produces: conversion options `{ severity, capRequired, evidenceRequired, dueDate }` with Observation-safe defaults.

- [x] **Step 1: Derive defaults from severity.**

  Implement:

  ```js
  function findingRequirementDefaults(severity) {
    if (severity === 4) return { capRequired: false, evidenceRequired: false, dueDate: null };
    return { capRequired: true, evidenceRequired: true, dueDate: '2026-07-15' };
  }
  ```

- [x] **Step 2: Expose explicit Lead choices.**

  On Potential Finding conversion, show checked/unchecked `CAP Required` and `Evidence Required` controls initialized from the defaults. A Lead choice must persist on the canonical Finding.

- [x] **Step 3: Derive the correct initial status.**

  Use:

  ```js
  status: capRequired ? 'WAITING_CAP' : (evidenceRequired ? 'EVIDENCE_REQUIRED' : 'OPEN_OBSERVATION')
  ```

- [x] **Step 4: Test both Observation paths.**

  Assert default Observation has no forced CAP/due date, while an explicitly configured Observation can require CAP and Evidence.

### Task 9: Require Authority, Stage, And Reason For Checklist Reopen

**Files:**

- Modify: `js/inspection.js`
- Modify: `js/app.js`
- Modify: `js/views.js`
- Modify: `tests/scenario-integrity-regression.test.js`
- Modify: `tests/inspection-execution-smoke.test.js`

**Interfaces:**

- Produces: `reopenInspectionChecklistForEditing(target, auditId, metadata)` with required `{ role, userId, reason, at }`.

- [x] **Step 1: Enforce the reopen contract in the domain helper.**

  Validate:

  ```js
  if (['inspector', 'leadInspector'].indexOf(metadata.role) === -1) throw new Error('Inspector or Lead Inspector authority required.');
  if (!workspace.submittedAt) throw new Error('Only a submitted checklist can be reopened.');
  if (!normalizeApprovalText(metadata.reason)) throw new Error('Reason for reopening is required.');
  if (audit.reportIssuedAt || audit.status === 'Closed') throw new Error('Issued or closed inspections cannot be reopened.');
  ```

- [x] **Step 2: Add a reason-required confirmation dialog.**

  Keep the submitted checklist read-only until the user supplies a reason and confirms. Separate the reason from checklist comments.

- [x] **Step 3: Preserve submission history.**

  Append a history/audit-log record with previous submission time, actor, reopen time, reason, and exact Audit ID.

- [x] **Step 4: Test denial and success.**

  Assert missing reason, wrong role, non-submitted state, and issued report are mutation-free; an authorized reopen changes status and preserves previous submission metadata.

### Task 10: Normalize Closure Types And Auditee Timeline Labels

**Files:**

- Modify: `js/data.js`
- Modify: `js/inspection.js`
- Modify: `js/app.js`
- Modify: `js/views.js`
- Modify: `tests/scenario-integrity-regression.test.js`
- Modify: `tests/service-provider-portal-smoke.test.js`
- Modify: `tests/manager-report-pdf-smoke.test.js`

**Interfaces:**

- Produces: exact closure types `evidence-verified` and `authorized`, plus `closureBasisLabel(finding)`.

- [x] **Step 1: Use explicit canonical closure constants.**

  Evidence `close` must write:

  ```js
  finding.status = 'CLOSED';
  finding.closedDate = DEMO_TODAY;
  finding.closureType = 'evidence-verified';
  ```

  Authorized closure must write `closureType = 'authorized'` and retain its required reason/actor/history.

- [x] **Step 2: Map closure labels centrally.**

  Implement:

  ```js
  function closureBasisLabel(finding) {
    if (finding.closureType === 'evidence-verified' || finding.closureType === 'evidence-accepted') {
      return 'Evidence accepted and verified';
    }
    if (finding.closureType === 'authorized' || finding.closureType === 'authorized-no-cap') {
      return 'Authorized closure (audit-logged)';
    }
    return 'Closure basis not recorded';
  }
  ```

- [x] **Step 3: Render the Auditee timeline dynamically.**

  The final step must use `closureBasisLabel(finding)` and never hard-code `Authorized closure`.

- [x] **Step 4: Test partial/not-close/close and both closure paths.**

  Assert partial/not-close leave the Finding open, evidence close labels the report/timeline correctly, and authorized closure retains its distinct label and reason.

### Task 11: Add Deterministic Reminder And Escalation Events

**Files:**

- Modify: `js/data.js`
- Modify: `js/work-items.js`
- Modify: `js/app.js`
- Modify: `js/views.js`
- Modify: `tests/scenario-integrity-regression.test.js`
- Modify: `tests/department-manager-findings-smoke.test.js`
- Modify: `tests/service-provider-portal-smoke.test.js`
- Modify: `tests/demo-boundary-smoke.test.js`

**Interfaces:**

- Produces: `deriveReminderStage(finding, today)` and idempotent browser-local reminder event records.

- [x] **Step 1: Derive exact stages from Due Date.**

  Implement a pure helper returning one of:

  ```js
  '30_days', '15_days', '7_days', 'due_today', 'overdue', 'none'
  ```

  Use calendar-day difference against `DEMO_TODAY`; closed Findings return `none`.

- [x] **Step 2: Create idempotent demo events.**

  Use event IDs of the form:

  ```js
  'REM-' + finding.id + '-' + stage
  ```

  Do not duplicate an existing stage event. Store recipient role, organization scope, created date, demo channel `in_app`, and delivery status `demo_recorded`.

- [x] **Step 3: Add manager escalation rules.**

  Level 1 open Findings produce immediate manager attention; overdue Findings produce a manager escalation event. Neither event triggers enforcement automatically.

- [x] **Step 4: Keep manual reminder available and traceable.**

  Manual reminder remains a distinct audit-log event and must not impersonate an automatic stage event.

- [x] **Step 5: Render history and boundary copy.**

  Show stage, recipient, date, status, and `Demo in-app event; no real delivery` on relevant Finding/Auditee/Manager screens.

- [x] **Step 6: Run reminder and privacy tests.**

  Run:

  ```bash
  node --test tests/scenario-integrity-regression.test.js tests/department-manager-findings-smoke.test.js tests/service-provider-portal-smoke.test.js tests/demo-boundary-smoke.test.js
  ```

  Expected: all five timing stages, critical attention, overdue escalation, idempotency, organization scope, and no enforcement side effect pass.

---

## Phase 4 — Align Documentation And Prove The Rendered Product

### Task 12: Align Scenario, Screen, Workflow, And Build Documentation

**Files:**

- Modify: `docs/product-specs/scenarios/DEMO_SCENARIO_OPERATOR_AUDIT.md`
- Modify: `docs/product-specs/scenarios/DEMO_SCENARIO_OPERATOR_AUDIT.turkce.md`
- Modify: matching checklist/Finding/reminder workflow English/Turkish files that exist
- Modify: `docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.md`
- Modify: `docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.turkce.md`
- Modify: `docs/demo-evidence/BUILD_SUMMARY.md`
- Modify: `docs/demo-evidence/BUILD_SUMMARY.turkce.md`
- Modify: `MANIFEST.md`

**Interfaces:**

- Consumes: verified runtime behavior from Tasks 1-11.
- Produces: implementation-accurate stakeholder and package documentation.

- [x] **Step 1: Align the hero Finding creation flow.**

  Replace direct Inspector issue language with:

  ```text
  Inspector records a Non-Compliant checklist result with a required comment.
  AviaSurveil360 creates a Potential Finding for Lead Inspector review.
  The Lead Inspector returns, dismisses, or converts it to canonical Finding CAB-2026-001.
  ```

- [x] **Step 2: Document Observation and closure boundaries exactly.**

  State that Observation CAP/Evidence is configurable; CAP acceptance is not closure; Evidence `Close` and reason-required authorized closure are distinct.

- [x] **Step 3: Document reminder behavior literally.**

  Describe deterministic browser-local in-app events, not real delivery or production scheduling.

- [x] **Step 4: Update build evidence only with verified claims.**

  Use `verified locally` for passing automated/browser evidence and retain `demo-only`/`production-readiness not claimed` boundaries.

- [x] **Step 5: Check changed Markdown paths and bilingual parity.**

  Run:

  ```bash
  rg -n "Potential Finding|CAP acceptance|Evidence accepted and verified|Authorized closure|30-day|15-day|7-day|demo-only" docs/product-specs docs/demo-evidence MANIFEST.md
  git diff --check
  ```

  Expected: English/Turkish lifecycle statements agree; no stale direct-issue or every-closure-is-authorized wording remains.

### Task 13: Run The Full Automated And Real-Browser Scenario Matrix

**Files:**

- Create after successful verification: `docs/demo-evidence/BROWSER_SCENARIO_INTEGRITY_2026-07-20.md`
- Create after successful verification: `docs/demo-evidence/BROWSER_SCENARIO_INTEGRITY_2026-07-20.turkce.md`
- Modify: this plan
- Modify: `docs/exec-plans/index.md`
- Modify: `docs/exec-plans/tech-debt-tracker.md`
- Modify: `MANIFEST.md` if new evidence files are created

**Interfaces:**

- Consumes: completed runtime, tests, and docs.
- Produces: final PASS/PARTIAL/FAIL/BLOCKED matrix, screenshots, console result, cleanup evidence, and synchronized plan status.

- [x] **Step 1: Run syntax and focused tests.**

  Run:

  ```bash
  for f in js/*.js; do node --check "$f" || exit 1; done
  node --test tests/scenario-integrity-regression.test.js tests/browser-scenario-contract-smoke.test.js
  ```

  Expected: all syntax checks and focused tests pass.

- [x] **Step 2: Run the full test suite.**

  Run:

  ```bash
  node --test tests/*.test.js
  ```

  Expected: zero failures. Record the exact discovered/pass/fail counts; do not preserve or claim a historic count.

- [x] **Step 3: Start an isolated local HTTP server.**

  Run from the project root using an available localhost port. Record the PID/session and use only `http://127.0.0.1:<port>/index.html`.

- [x] **Step 4: Execute the real-click browser matrix.**

  Verify, with no direct state mutation:

  1. All eight role home/navigation boundaries.
  2. Full Planning approval/release/scoped Lead preparation/materialization handoff.
  3. Routine coordination, alternative date, canonical date propagation.
  4. Ad Hoc/Unannounced intake, withheld notice, Inspector visibility, Auditee absence.
  5. Cabin, Security, Ramp, Airworthiness, and Flight Operations checklist start/continue.
  6. Same question ID in two Audits with distinct answers and Potential Findings.
  7. Non-Compliant and Observation through Lead return, dismiss, and convert.
  8. Checklist submit and guarded, reason-required reopen.
  9. Auditee CAP submit/revision; Inspector accept/return without role switching.
  10. Evidence v1/v2, Not Close, Partially Close, Close, canonical report basis.
  11. Department Manager review with canonical Finding state consistency.
  12. Reminder stages and overdue manager escalation.
  13. Preliminary and Final Report DM/GM/ED decisions.
  14. Auditee organization privacy and Internal CAA Note separation.
  15. Closed report and Auditee closure-basis labels.

- [x] **Step 5: Save evidence for every critical repaired flow.**

  Save screenshots outside source directories while testing, then reference the accepted evidence in the bilingual evidence documents. Include at least checklist isolation, scoped Lead preparation, Inspector denial of management route, CAP acceptance without auto-chain, reason-required reopen, and evidence-verified closure report.

- [x] **Step 6: Check browser console and clean up processes.**

  Record unexpected console errors as failures. Close Browser tabs, stop the local server, and verify no task-owned HTTP server, Playwright/Puppeteer/webdriver/headless Chrome, or remote-debugging Chrome process remains.

- [x] **Step 7: Synchronize plan tracking.**

  If every required gate passes, mark this plan `ready-for-verification`, set the active index next todo to stakeholder review/sign-off, and close the related technical-debt note. If any gate remains incomplete, keep the plan `active` or mark it `blocked` only when the repo's blocker threshold is met; record the exact next action.

---

## Verification

Completion requires all of the following:

- JavaScript syntax checks pass.
- Focused P0/P1 regression tests pass.
- Full `tests/*.test.js` suite has zero failures.
- No test expects an unauthorized role switch, auto-filled approval, UI-only closure, or broad Lead Planning access.
- Two Audits using the same question ID retain distinct answers and Potential Findings.
- Inspector cannot open or execute Department Manager approval.
- CAP acceptance leaves later approval/report stages untouched.
- Department Manager review and all role screens reference the same canonical Finding.
- Seed Cabin and new Ad Hoc plans reach idempotent Audit materialization.
- Routine accepted alternative date is consistent across Audit, coordination, Inspector, and report screens.
- Ad Hoc notice remains withheld from the Auditee.
- Every visible checklist Start/Continue action is functional or explicitly disabled.
- Observation defaults do not force CAP/Evidence.
- Checklist reopen requires authorized role, valid stage, and reason, and preserves history.
- Evidence-verified and authorized closure labels are distinct in CAA and Auditee screens.
- Reminder stages are deterministic, idempotent, scoped, and explicitly demo-only.
- Auditee privacy and Internal CAA Note separation pass automated and browser checks.
- English/Turkish docs and build evidence match the implementation.
- Local server and Browser/Chrome automation processes are cleaned up.
- Active plan index and technical-debt tracker match the actual state.

## Risks And Mitigations

- **Concurrent working-tree edits:** Re-read every touched hunk immediately before patching and use narrow `apply_patch` edits; stop and report overlapping incompatible changes.
- **Persistence migration loses browser-local demo data:** Normalize only known fields, retain legacy values when unambiguous, and test a pre-remediation state fixture.
- **Removing synthetic records breaks polished screens:** Preserve view-model shapes while deriving them from canonical Findings; test exact IDs and actions before deleting arrays.
- **Role visibility mistaken for security:** Guard both rendering/navigation and handler execution; tests must call handlers with wrong roles and assert no mutation.
- **Non-Cabin packages invent regulatory obligations:** Use careful `Configured reference` demo wording and minimal mock questions; do not claim legal compliance.
- **Reminder UI implies real delivery:** Persist `in_app`/`demo_recorded` boundary fields and show explicit no-real-delivery copy.
- **Browser cache serves stale assets:** Use a fresh localhost origin or an updated asset version, verify fetched script content, and record the tested URL.
- **Large cross-cutting change becomes unreviewable:** Complete phases in order and run the focused gate after every task; do not combine unrelated visual redesign.

## Execution Prompt

```text
Implement the active AviaSurveil360 plan at docs/exec-plans/active/2026-07-20-browser-scenario-integrity-remediation-plan.md from the project root.

First read repo-local AGENTS.md and every source-of-truth document it requires. Preserve all existing unrelated dirty-worktree changes and re-read each touched hunk before editing because the working tree changed during the preceding browser audit. Do not create or switch branches, commit, push, open a PR, or post GitHub comments.

Execute the plan task-by-task in order. Prioritize the P0 canonical-data and authority tasks before Planning, checklist breadth, lifecycle labels, reminders, documentation, and browser evidence. Use the focused regression contract, implement the smallest coherent fix, and rerun the focused gate after each task. Do not retain false-contract tests that expect silent role switching, auto-filled approvals, UI-only closure, or broad Lead Planning access.

The implementation must remain frontend-only and demo-only. It must use canonical audit-scoped checklist answers and canonical state.findings records across Inspector, Lead Inspector, Department Manager, Auditee, Evidence, report, reminder, and dashboard screens. CAP acceptance must not close a Finding. Evidence-verified closure and reason-required authorized closure must remain distinct. Observation must not require CAP/Evidence by default. Submitted checklist reopen must require authorized role, valid stage, and reason. All visible checklist controls must work for their exact Audit or be explicitly disabled.

Run syntax checks, focused tests, and the complete tests/*.test.js suite with zero failures. Then serve the app over localhost and use the Browser skill for real clicks through the complete scenario matrix in Task 13; do not use file:// or direct state mutation as browser evidence. Save critical screenshots, check console errors, and clean up the server and Browser/Chrome automation processes.

Update the canonical English/Turkish docs, BUILD_SUMMARY pair, MANIFEST when needed, this plan's checkboxes/status, docs/exec-plans/index.md, and docs/exec-plans/tech-debt-tracker.md so tracking matches the verified final state. Use literal evidence labels: verified locally, not run, blocked, candidate-only, release pending, and production-ready must not be blended. Do not claim production readiness.
```
