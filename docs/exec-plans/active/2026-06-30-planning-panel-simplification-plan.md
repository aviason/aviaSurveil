# Planning Panel Simplification Implementation Plan


**Goal:** Make Audit Planning a single, simple, role-based Planning panel while preserving the `Department Manager -> GM -> Finance Review -> Executive Director` approval chain.

**Architecture:** Keep the static frontend-only demo architecture. Reuse the existing approval primitive in `js/approval.js` and planning preparation helpers in `js/planning.js`; consolidate the current `calendar`, `planning-board`, and `planning-approvals` user-facing planning surfaces into one canonical Planning workspace route, while keeping compatibility aliases where existing tests or links still need them.

**Tech Stack:** HTML, CSS, Vanilla JavaScript, mock data, client-side state, Node smoke tests.

---

## Context

The KILLCRITIC review found that the approval chain is technically sound but the product surface is split too widely:

- `planning-approvals` handles governance decisions.
- `planning-board` handles release and audit preparation.
- `calendar` / `Audit Work Queue` handles audit execution queueing and is also used as a planning destination in several role-home rows.

The target product direction is:

- Planning should be a distinct module/panel.
- The `Department Manager -> GM -> Finance Review -> Executive Director` chain must remain intact.
- The system should stay simple and not feel like many separate modules.

## Objective

Create one canonical **Planning** workspace in the demo:

1. One nav entry named `Planning` for Department Manager, GM, Finance Review, Executive Director, and Lead Inspector when planning preparation is relevant.
2. One page that shows the selected planning item, current owner, next action, due/target timing, approval status, preparation status, and history.
3. A compact segmented control inside the page for `Overview`, `Approval`, and `Preparation`, not separate top-level modules.
4. ED approval must lead visibly to the next operational step: `GM Release to Department`, then Department Manager acceptance, lead assignment, team/date/resource proposal, and Department Manager confirmation.

## Scope

### In Scope

- Add a canonical `planning` view route.
- Change role navigation so planning appears as one primary item instead of separate `Planning Board` and `Planning Approvals` entries.
- Keep `Audit Work Queue` as the inspector/audit execution queue, not the governance planning panel.
- Preserve old `planning-board` and `planning-approvals` route handling as compatibility aliases that render the new Planning workspace with the matching initial tab.
- Refactor visible planning copy away from implementation-phase wording such as `Phase 0B · thin planning approval slice`.
- Update planning smoke tests to prove the simple panel, chain order, role-aware actions, and post-ED next action.
- Update demo summary docs after the UI behavior changes.

### Out Of Scope

- No backend, database, API, real authentication, real authorization service, real file upload, real document generation, real finance/accounting integration, email/SMS, e-signature, or framework migration.
- No new legal, enforcement, certificate, or closure automation.
- No drag-and-drop board behavior.
- No advanced risk-based planning engine.
- No broad redesign of findings, CAP, evidence, reports, USOAP, SSP/NASP, AI assistant, or offline simulation.
- No branch, commit, or push unless the user explicitly asks.

## Assumptions

- `SEED_PLANNING_ITEMS` stays as the thin demo source for the main planning scenario.
- `applyApprovalDecision()` remains the source of truth for approval movement.
- `releasePlanningItem()`, `acceptReleasedPlanningItem()`, `assignLeadInspectorToPlanningItem()`, `proposePlanningTeamAndSchedule()`, and `confirmPlanningPreparation()` remain the source of truth for post-approval planning preparation.
- Existing `planning-board` and `planning-approvals` tests should not be deleted until replacement coverage is passing.
- The demo date context remains `DEMO_TODAY = 2026-06-15`.

## Files To Modify

- `js/app.js`
  - Add canonical `planning` route title.
  - Collapse role navigation to one `Planning` item for planning roles.
  - Keep route compatibility for `planning-board` and `planning-approvals`.
  - Route planning-related role-home rows to `planning`, not `calendar`, when the item is a planning governance item.

- `js/views.js`
  - Add `viewPlanningWorkspace()`.
  - Keep `viewPlanningApprovals()` and `viewPlanningBoard()` as thin wrappers or aliases if needed for compatibility.
  - Add helper functions for planning tabs, planning overview metrics, and the post-approval next action.
  - Remove implementation-phase visible copy from the stakeholder demo surface.

- `js/data.js`
  - Add default planning UI state if needed, such as `selectedFilters.planningTab`.
  - Do not change the approval chain semantics unless a failing test proves a gap.

- `css/styles.css`
  - Add compact styling for the Planning segmented control and unified planning panel.
  - Reuse existing card, list, badge, stepper, decision panel, and planning-board styling where practical.

- `tests/planning-workspace-smoke.test.js`
  - Create this test to cover the new canonical workspace.

- `tests/planning-render-smoke.test.js`
  - Update route/render expectations so legacy route names delegate to the new Planning workspace.

- `tests/governance-render-smoke.test.js`
  - Update role render expectations to use `Planning`, while still proving GM, Finance, and ED can render the approval controls.

- `tests/planning-release-smoke.test.js`
  - Keep the existing semantic flow test. Add a post-approval UI-facing assertion only if the shared helper is moved into testable logic.

- `docs/demo-evidence/BUILD_SUMMARY.md`
  - Update the planning/navigation summary after implementation.

- `docs/demo-evidence/BUILD_SUMMARY.turkce.md`
  - Update the Turkish companion summary with the same demo boundary and visible planning change.

- `MANIFEST.md`
  - Update only if new/renamed tests or files change the package inventory.

## Phases

### Phase 0: Baseline Guard

Verify the current planning behavior before changing it.

- [ ] **Step 1: Run existing planning and governance tests.**

  Run:

  ```bash
  node tests/approval-smoke.test.js
  node tests/planning-release-smoke.test.js
  node tests/planning-render-smoke.test.js
  node tests/governance-render-smoke.test.js
  ```

  Expected:

  ```text
  approval-smoke: ok
  planning-release-smoke: ok
  planning-render-smoke: ok
  governance-render-smoke: ok
  ```

- [ ] **Step 2: Inspect current route labels.**

  Run:

  ```bash
  rg -n "Planning Board|Planning Approvals|Audit Work Queue|planning-board|planning-approvals|calendar" js/app.js js/views.js tests
  ```

  Expected: output shows the current split surfaces that this plan intentionally consolidates.

### Phase 1: Write Canonical Planning Workspace Tests

Define the target behavior before implementation.

- [ ] **Step 1: Create `tests/planning-workspace-smoke.test.js`.**

  Use this complete initial test file:

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
    'js/views.js'
  ].forEach((file) => {
    vm.runInContext(fs.readFileSync(path.join(root, file), 'utf8'), context, { filename: file });
  });

  context.state = context.freshState();

  assert.equal(typeof context.viewPlanningWorkspace, 'function', 'canonical Planning workspace renderer exists');

  context.state.role = 'gm';
  context.state.view = 'planning';
  let html = context.viewPlanningWorkspace();
  assert.match(html, /Planning/);
  assert.match(html, /Department Manager[\s\S]*GM Review[\s\S]*Finance Review[\s\S]*Executive Director Approval/);
  assert.match(html, /Send to Finance Review/);
  assert.doesNotMatch(html, /Phase 0B|thin planning approval slice/i);

  context.applyApprovalDecision(context.state.planningItems[0], {
    decision: 'forward',
    actor: { role: 'gm', name: context.ROLES.gm.user },
    comment: 'Scope accepted for finance review.'
  });

  context.state.role = 'finance';
  html = context.viewPlanningWorkspace();
  assert.match(html, /Approve Budget/);
  assert.match(html, /Finance Review/);

  context.applyApprovalDecision(context.state.planningItems[0], {
    decision: 'approve',
    actor: { role: 'finance', name: context.ROLES.finance.user },
    comment: 'Budget accepted.'
  });

  context.state.role = 'executiveDirector';
  html = context.viewPlanningWorkspace();
  assert.match(html, /Approve Plan/);
  assert.match(html, /Executive Director Approval/);

  context.applyApprovalDecision(context.state.planningItems[0], {
    decision: 'approve',
    actor: { role: 'executiveDirector', name: context.ROLES.executiveDirector.user },
    comment: 'Approved for release.'
  });

  context.state.role = 'gm';
  html = context.viewPlanningWorkspace();
  assert.match(html, /GM Release to Department|Release to Department/);
  assert.match(html, /Approved/);

  console.log('planning-workspace-smoke: ok');
  ```

- [ ] **Step 2: Run the new test and confirm it fails for the right reason.**

  Run:

  ```bash
  node tests/planning-workspace-smoke.test.js
  ```

  Expected before implementation:

  ```text
  AssertionError: canonical Planning workspace renderer exists
  ```

### Phase 2: Consolidate Route And Navigation

Create one top-level Planning entry while preserving compatibility.

- [ ] **Step 1: Modify `js/app.js` navigation.**

  Target rules:

  - Department Manager nav has one `Planning` item instead of both `Planning Board` and `Planning Approvals`.
  - GM nav has one `Planning` item, plus report items.
  - Finance nav has one `Planning` item.
  - Executive Director nav has one `Planning` item, plus report items.
  - Lead Inspector may use `Planning` for assigned audit preparation.
  - Inspector keeps `Audit Work Queue`; this is execution work, not governance planning.

  Concrete target nav entries:

  ```js
  { view: 'planning', label: 'Planning', icon: '▤' }
  ```

- [ ] **Step 2: Add `planning` to `VIEW_TITLES`.**

  Target value:

  ```js
  planning: 'Planning'
  ```

- [ ] **Step 3: Update role-home rows that currently point planning governance items to `calendar`.**

  Planning records such as `PLAN-2026-Q3` and finance/ED planning rows should point to:

  ```js
  'planning'
  ```

  Audit execution rows such as `AUD-2026-001` may continue to point to:

  ```js
  'calendar'
  ```

### Phase 3: Build The Unified Planning Workspace

Add the canonical renderer and make legacy planning screens delegate to it.

- [ ] **Step 1: Add `planningWorkspaceTab()` in `js/views.js`.**

  Required behavior:

  - Default tab is `overview`.
  - `state.params.tab` can select `overview`, `approval`, or `preparation`.
  - Legacy `planning-approvals` route maps to `approval`.
  - Legacy `planning-board` route maps to `preparation`.

- [ ] **Step 2: Add `viewPlanningWorkspace()` in `js/views.js`.**

  Required visible sections:

  - Page title: `Planning`.
  - Description: `Single planning panel for approval, release, and audit preparation.`
  - Guardrails: `Frontend-only demo`, `Mock approval history`, `No real authorization service`.
  - Top metrics: current owner, next action, approval status, preparation status.
  - Planning item detail: organization, department, risk category, trigger type, requested budget, target month, proposed inspectors.
  - Approval progress using existing `approvalProgressHtml(item)`.
  - Role-aware decision panel using existing `approvalDecisionPanelHtml(item)`.
  - Preparation action panel using existing `planningPrepActionPanel(item)`.
  - Approval history and preparation history as secondary panels.

- [ ] **Step 3: Preserve legacy function names.**

  Keep these functions callable:

  ```js
  function viewPlanningApprovals() {
    return viewPlanningWorkspace('approval');
  }

  function viewPlanningBoard() {
    return viewPlanningWorkspace('preparation');
  }
  ```

  If the implementation chooses a different parameter shape, keep the external behavior equivalent: old tests and route handlers must still render the unified workspace.

- [ ] **Step 4: Remove visible implementation-phase language.**

  The Planning page must not show:

  ```text
  Phase 0B
  thin planning approval slice
  ```

### Phase 4: Make Post-Approval Continuity Obvious

ED approval should not look like the planning story is over if preparation still needs GM/Department action.

- [ ] **Step 1: Add a Planning next-action helper if current inline copy is not enough.**

  Desired outputs:

  - Before approval complete: use `approvalSummary(item).nextAction`.
  - After ED approval and before release: `GM Release to Department`.
  - After release: `Department Manager accept released audit`.
  - After acceptance: `Department Manager assign Lead Inspector`.
  - After lead assigned: `Lead Inspector propose team, dates, and resources`.
  - After proposal: `Department Manager confirm and generate mock assignment package`.
  - After confirmation: `Ready for execution`.

- [ ] **Step 2: Display that next action in the Planning hero and next action panel.**

  The top of the Planning page must answer:

  ```text
  Who owns this now?
  What must happen next?
  Is approval complete?
  Is preparation complete?
  ```

### Phase 5: Update Tests

Keep semantic tests and update render tests.

- [ ] **Step 1: Run the new canonical test.**

  Run:

  ```bash
  node tests/planning-workspace-smoke.test.js
  ```

  Expected:

  ```text
  planning-workspace-smoke: ok
  ```

- [ ] **Step 2: Update `tests/planning-render-smoke.test.js`.**

  Required assertions:

  - `viewPlanningApprovals()` still renders a Planning page.
  - The page includes `Send to Finance Review` for GM at the first stage.
  - The page includes `Finance Not Approved` during Finance Review.
  - A finance return still displays `Returned to GM Action`.

- [ ] **Step 3: Update `tests/governance-render-smoke.test.js`.**

  Required assertions:

  - GM Planning route renders.
  - Finance Planning route renders.
  - Executive Director Planning route renders.
  - Checklist and report routes still render.

- [ ] **Step 4: Run all planning/governance tests.**

  Run:

  ```bash
  node tests/approval-smoke.test.js
  node tests/planning-release-smoke.test.js
  node tests/planning-workspace-smoke.test.js
  node tests/planning-render-smoke.test.js
  node tests/governance-render-smoke.test.js
  ```

  Expected: all commands print `ok`.

### Phase 6: Update Demo Documentation

Reflect the visible product simplification.

- [ ] **Step 1: Update `docs/demo-evidence/BUILD_SUMMARY.md`.**

  Required content:

  - The demo now has a single Planning panel for Department Manager, GM, Finance Review, Executive Director, and relevant Lead Inspector preparation work.
  - The `Department Manager -> GM -> Finance Review -> Executive Director` chain is preserved.
  - `Audit Work Queue` remains an execution queue, not a separate planning governance module.
  - Behavior is still frontend-only and mock/demo only.

- [ ] **Step 2: Update `docs/demo-evidence/BUILD_SUMMARY.turkce.md`.**

  Required content in Turkish:

  - Planlama artik tek bir Planning panelinde toparlandi.
  - Department Manager -> GM -> Finance Review -> Executive Director zinciri korundu.
  - Audit Work Queue saha/denetim is kuyrugu olarak kaldi.
  - Backend, gercek yetkilendirme, gercek finans entegrasyonu veya gercek dokuman uretimi eklenmedi.

- [ ] **Step 3: Update `MANIFEST.md` only if the new test file or changed package inventory needs to be listed.**

### Phase 7: Final Verification

- [ ] **Step 1: Run syntax checks.**

  Run:

  ```bash
  node --check js/data.js
  node --check js/helpers.js
  node --check js/approval.js
  node --check js/planning.js
  node --check js/views.js
  node --check js/app.js
  ```

  Expected: no output and exit code `0` for each command.

- [ ] **Step 2: Run focused smoke checks.**

  Run:

  ```bash
  node tests/approval-smoke.test.js
  node tests/planning-release-smoke.test.js
  node tests/planning-workspace-smoke.test.js
  node tests/planning-render-smoke.test.js
  node tests/governance-render-smoke.test.js
  node tests/demo-boundary-smoke.test.js
  ```

  Expected: all commands print `ok`.

- [ ] **Step 3: Search for forbidden planning fragmentation in role navigation.**

  Run:

  ```bash
  rg -n "Planning Board|Planning Approvals|Phase 0B|thin planning approval slice" js/app.js js/views.js
  ```

  Expected:

  - No visible-nav occurrences of `Planning Board` or `Planning Approvals`.
  - No occurrences of `Phase 0B` or `thin planning approval slice`.
  - Compatibility function names may remain only if they are not exposed as top-level user-facing nav labels.

- [ ] **Step 4: Browser smoke check.**

  Open `index.html` directly or serve locally only if needed. Verify:

  - GM sees one `Planning` item and can send the plan to Finance.
  - Finance sees one Planning review and can approve or return.
  - Executive Director can approve the plan.
  - After ED approval, GM sees `Release to Department`.
  - Department Manager can accept and continue preparation.
  - Inspector `Audit Work Queue` still opens and is not renamed into governance Planning.

## Verification

Required local verification before claiming `ready-for-verification`:

```bash
node --check js/data.js
node --check js/helpers.js
node --check js/approval.js
node --check js/planning.js
node --check js/views.js
node --check js/app.js
node tests/approval-smoke.test.js
node tests/planning-release-smoke.test.js
node tests/planning-workspace-smoke.test.js
node tests/planning-render-smoke.test.js
node tests/governance-render-smoke.test.js
node tests/demo-boundary-smoke.test.js
```

Manual/browser verification is required for visible navigation clarity. If browser automation is unavailable, report the limitation as `not run` and keep the status below `ready-for-verification`.

## Risks

- Hiding old planning routes from nav while keeping compatibility wrappers can leave stale labels in tests or docs. Use targeted `rg` checks.
- Over-consolidating `Audit Work Queue` into Planning could confuse execution work with governance approval work. Keep execution queue language for Inspector.
- A single panel can become dense. Keep the first viewport focused on owner, next action, approval status, preparation status, and one primary action.
- Updating docs without Turkish companion text would break the repository bilingual stakeholder pattern for demo summaries.

## Dependencies

- Existing approval primitive in `js/approval.js`.
- Existing preparation helpers in `js/planning.js`.
- Existing seeded planning item in `js/data.js`.
- Existing card, badge, decision panel, approval history, and stepper UI patterns in `js/views.js` and `css/styles.css`.
- Existing demo-only constraints in `AGENTS.md`.

## Ownership Boundaries

- Product ownership: simplify Planning IA without changing the legally careful demo boundary.
- Frontend ownership: static HTML/CSS/Vanilla JS only.
- Test ownership: add and update deterministic Node smoke checks before claiming local verification.
- Documentation ownership: update English canonical demo summary and Turkish companion summary after visible UI copy changes.
- Change-control ownership: do not create branches, commits, pushes, backend services, or production claims without explicit user instruction.

## Self-Review

- Spec coverage: the plan covers the separate Planning panel, preserved DM -> GM -> Finance -> ED chain, simplified module count, post-approval release continuity, tests, docs, and demo-only guardrails.
- Placeholder scan: no `TBD`, `TODO`, or unspecified implementation placeholders are used.
- Type consistency: route names use `planning`, legacy wrappers keep `planning-approvals` and `planning-board`, and existing role keys remain `manager`, `gm`, `finance`, and `executiveDirector`.

## Execution Prompt

Use this exact prompt to execute the plan:

```text
Implement docs/exec-plans/active/2026-06-30-planning-panel-simplification-plan.md in /Users/marlonjd/Developer/web/aviaSurveil360.

Hard constraints:
- Read AGENTS.md first and obey the demo-first/static frontend rules.
- Do not create branches, commit, push, add backend/API/database/auth/upload/email/e-signature/document generation, or migrate frameworks.
- Preserve the Department Manager -> GM -> Finance Review -> Executive Director approval chain.
- Make Planning a single canonical role-based panel; hide Planning Board / Planning Approvals as top-level user-facing nav, but keep compatibility wrappers/routes if tests or links need them.
- Keep Audit Work Queue as the inspector/audit execution queue, not a separate planning governance module.
- Preserve unrelated dirty worktree changes.

Implementation target:
1. Add a canonical planning view in js/app.js and js/views.js.
2. Collapse planning nav for Department Manager, GM, Finance Review, Executive Director, and relevant Lead Inspector work into one Planning entry.
3. Add viewPlanningWorkspace() with Overview / Approval / Preparation sections or tabs.
4. Preserve applyApprovalDecision() and planning helper semantics.
5. Make ED approval visibly continue to GM Release to Department and preparation next actions.
6. Remove stakeholder-visible implementation-phase copy such as Phase 0B / thin planning approval slice.
7. Add tests/planning-workspace-smoke.test.js and update planning/governance render tests.
8. Update docs/demo-evidence/BUILD_SUMMARY.md, docs/demo-evidence/BUILD_SUMMARY.turkce.md, and MANIFEST.md only if package inventory changes.

Required verification:
node --check js/data.js
node --check js/helpers.js
node --check js/approval.js
node --check js/planning.js
node --check js/views.js
node --check js/app.js
node tests/approval-smoke.test.js
node tests/planning-release-smoke.test.js
node tests/planning-workspace-smoke.test.js
node tests/planning-render-smoke.test.js
node tests/governance-render-smoke.test.js
node tests/demo-boundary-smoke.test.js

Final report:
- List changed files.
- Report verification as verified locally / not run / blocked.
- Explicitly state that no backend, auth service, real finance integration, real upload/storage, or production authorization was added.
- Call out any remaining mock/demo behavior.
```
