# Ad Hoc / Unannounced Inspection Intake Alignment Implementation Plan


**Goal:** Let a Department Manager create an Ad Hoc / Unannounced inspection from the canonical Planning workspace, preserve the full approval and release lifecycle, withhold advance Service Provider notice, and enter the Audit Work Queue only when preparation is complete.

**Architecture:** Keep the existing frontend-only HTML/CSS/Vanilla JavaScript demo. Convert the legacy `New Audit Wizard` into a Planning intake that creates a governed `planningItem`; reuse the existing approval and preparation helpers; materialize the executable `Audit`, inspection team, and coordination record only after Department Manager confirmation of the released plan. Keep the old `wizard` route/action names only as internal compatibility aliases while the visible product language becomes `New Inspection`.

**Tech Stack:** HTML, CSS, Vanilla JavaScript, browser-local mock state, Node `node:test`/assert smoke tests, local browser verification.

## Global Constraints

- Use `AviaSurveil360` as the canonical product name.
- Remain frontend-only and demo-only: no backend, database, API, real authentication, real authorization service, real notification delivery, real file storage, or framework migration.
- Preserve the accepted lifecycle: `Department Manager -> Finance Review -> General Manager -> Executive Director -> GM Release to Department -> Department preparation -> execution`.
- `Ad Hoc / Unannounced` means advance Service Provider notification and coordination are withheld; the inspection must not appear in the Service Provider portal before execution.
- `Special Inspection` is an application/audit type, not a substitute for the separate `Inspection Category` and `Advance-notice Policy` fields.
- Lead Inspector and team assignment occur after approval, GM release, and Department Manager acceptance; they must not be collected by the intake wizard.
- Keep canonical source identifiers, code comments, implementation notes, and plan tracking in English.
- Update matching `.turkce.md` companions when stakeholder-facing canonical docs change.
- Preserve unrelated working-tree changes. Do not create or switch branches, commit, push, or initialize a repository.
- Do not claim production readiness. The final evidence label remains `demo-only; verified locally; production-readiness not claimed`.

---

## Objective

Replace the misleading Department Manager flow:

```text
Audits -> + New Audit -> Create & schedule -> Scheduled -> CAA Inspector -> Start checklist
```

with the governed flow:

```text
Planning -> + New Inspection -> Ad Hoc / Unannounced
-> Finance Review -> GM Review -> Executive Director Approval
-> GM Release to Department -> Department acceptance
-> Lead Inspector/team/date preparation -> Department confirmation
-> Scheduled Audit + No Advance Notice -> execution
```

The submission result must remain in the Planning Command Center with the newly created plan selected. It must not open the generic Audit Detail screen or create an executable audit before approval and preparation are complete.

## Scope

### In Scope

- Add a Department Manager `+ New Inspection` action to the canonical Planning workspace.
- Remove the Department Manager `+ New Audit` action from the Audit Work Queue.
- Rework the existing wizard into a Planning intake with explicit `Routine / Announced` and `Ad Hoc / Unannounced` categories.
- Derive and persist `noticePolicy` as `advance` or `withheld`.
- Create a governed planning item and send it to Finance Review on submission.
- Keep Lead Inspector/team selection out of intake and in the existing post-release preparation flow.
- Materialize an Audit, inspection team, inspection workspace, and coordination record only after the planning item reaches `ready_for_execution`.
- For Ad Hoc / Unannounced, create `notice_withheld` coordination state with no Service Provider notification.
- For Routine / Announced, create `ready_to_notify` coordination state for the existing Lead Inspector coordination flow.
- Add deterministic smoke coverage for intake, approval/release, materialization, privacy, persistence, route copy, and compatibility aliases.
- Update screen inventory, bilingual build evidence, package manifest, plan checkboxes, and active-plan index.

### Out Of Scope

- Emergency approval bypasses or a new fast-track authority chain.
- Automatic legal, enforcement, certificate, suspension, or closure decisions.
- Production notifications, email, SMS, WhatsApp, secure evidence storage, or immutable audit logs.
- A new backend, database, API, authentication system, authorization service, or frontend framework.
- A broad redesign of dashboards, Findings, CAP, Evidence, reports, or unrelated role workspaces.
- Drag-and-drop planning, advanced BI, automatic risk scoring, or optimization engines.
- Branch creation, commits, pushes, pull requests, or GitHub comments.

## Assumptions

- All inspections, including Ad Hoc / Unannounced, use the currently accepted approval chain; an emergency bypass requires a separate stakeholder decision and is not inferred here.
- Finance remains in the chain even when the requested demo budget is `0`.
- `state.auditSeq` remains the source for executable Audit IDs.
- Dynamic Planning IDs use `PLAN-2026-INS-###`, derived from existing planning items, so no new persisted counter or state migration version is required.
- Existing `applyApprovalDecision()`, `releasePlanningItem()`, `acceptReleasedPlanningItem()`, `assignLeadInspectorToPlanningItem()`, `proposePlanningTeamAndSchedule()`, and `confirmPlanningPreparation()` remain the sources of truth for governance transitions.
- The first executable Audit is created only after `confirmPlanningPreparation()` succeeds.
- Existing checklist/template lookup remains sufficient for the demo; unsupported templates may render a non-runnable checklist but must still preserve the planning and notice-policy record.

## Ownership Boundaries

- Department Manager owns inspection intake, submission, released-plan acceptance, Lead Inspector assignment, and final preparation confirmation.
- Finance Review owns budget approval or return for revision.
- General Manager owns forwarding a Finance-reviewed plan to the Executive Director and releasing an approved plan to the Department.
- Executive Director owns final plan approval or rejection; Executive Director approval alone does not release the plan.
- Lead Inspector owns the team/date/resource proposal and, for Routine / Announced inspections, the later Service Provider coordination action.
- Service Provider sees only advance-notice-required coordination records released to its organization; Ad Hoc / Unannounced records remain hidden.

## Files

### Runtime

- Modify `js/planning.js`
  - Add intake validation, category-to-policy mapping, governed planning-item creation, dynamic ID generation, and ready-plan materialization helpers.
- Modify `js/app.js`
  - Route the new Planning action, capture/validate intake fields, submit to Finance Review, and materialize the Audit after preparation confirmation.
- Modify `js/views.js`
  - Add the Planning header action, remove the Audit Work Queue creation action, and render the revised intake steps and post-materialization Audit link.
- Modify `js/data.js`
  - Update wizard defaults and persistence normalization only where the new fields require it; do not bump `DEMO_STATE_VERSION` unless a failing persistence test proves a migration is required.
- Modify `css/styles.css`
  - Add only the small intake-policy callout and responsive form styles required by the revised flow.

### Tests

- Create `tests/unannounced-inspection-intake-smoke.test.js`.
- Modify `tests/planning-workspace-smoke.test.js`.
- Modify `tests/planning-release-smoke.test.js`.
- Modify `tests/inspection-coordination-smoke.test.js`.
- Modify `tests/table-first-workbench-smoke.test.js`.
- Modify `tests/audit-work-queue-smoke.test.js` only if Department Manager queue assertions belong there after the new test is in place.
- Modify `tests/demo-boundary-smoke.test.js` only if new visible boundary copy requires an assertion.

### Documentation And Tracking

- Modify `docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.md`.
- Modify `docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.turkce.md`.
- Modify `docs/demo-evidence/BUILD_SUMMARY.md`.
- Modify `docs/demo-evidence/BUILD_SUMMARY.turkce.md`.
- Modify `MANIFEST.md`.
- Modify `docs/exec-plans/index.md`.
- Modify this plan as tasks complete.
- Modify `docs/exec-plans/tech-debt-tracker.md` only if implementation leaves a durable blocker, accepted risk, or missing-evidence item.

---

## Task 1: Freeze The Intake And Notice-Policy Contract With Failing Tests

**Files:**

- Create: `tests/unannounced-inspection-intake-smoke.test.js`
- Modify: `tests/planning-workspace-smoke.test.js`
- Modify: `tests/table-first-workbench-smoke.test.js`

**Interfaces:**

- Consumes: `freshState()`, `viewPlanningWorkspace()`, `viewCalendar()`, `viewAuditWizard()`, and the existing planning approval/preparation helpers.
- Produces: executable expectations for `inspectionCategoryPolicy()`, `createPlanningInspection()`, and `materializeReadyPlanningInspection()`.

- [x] **Step 1: Create the new focused smoke-test harness.**

  Create `tests/unannounced-inspection-intake-smoke.test.js` with the same VM-loading pattern as `tests/inspection-coordination-smoke.test.js` and load these files in order:

  ```js
  [
    'js/data.js',
    'js/helpers.js',
    'js/approval.js',
    'js/planning.js',
    'js/checklists.js',
    'js/inspection.js',
    'js/reports.js',
    'js/manager-workspaces.js',
    'js/work-items.js',
    'js/views.js'
  ].forEach((file) => {
    vm.runInContext(fs.readFileSync(path.join(root, file), 'utf8'), context, { filename: file });
  });
  ```

- [x] **Step 2: Add the Ad Hoc / Unannounced intake assertions.**

  Use this exact input shape and assertions:

  ```js
  const state = context.freshState();
  const input = {
    organizationId: 'ORG-SKY',
    applicationType: 'Special Inspection',
    domain: 'Security',
    inspectionCategory: 'Ad Hoc / Unannounced',
    purpose: 'Unannounced access-control compliance inspection.',
    triggerType: 'Risk based / targeted inspection',
    riskCategory: 'Cargo gate access control',
    plannedDate: '2026-07-24',
    mode: 'On-site',
    location: 'SkyCargo Terminal',
    templateId: 'TPL-SEC-2026',
    scope: 'Restricted cargo-area access controls and records.',
    currency: 'USD',
    requestedBudget: 0
  };

  const item = context.createPlanningInspection(state, input, {
    role: 'manager',
    name: context.ROLES.manager.user
  });

  assert.match(item.id, /^PLAN-2026-INS-\d{3}$/);
  assert.equal(item.inspectionCategory, 'Ad Hoc / Unannounced');
  assert.equal(item.noticePolicy, 'withheld');
  assert.equal(item.applicationType, 'Special Inspection');
  assert.equal(item.approval.currentIndex, 1);
  assert.equal(context.approvalSummary(item).ownerRole, 'finance');
  assert.equal(item.preparation.status, 'not_released');
  assert.equal(item.auditId, '');
  assert.equal(state.audits.some((audit) => audit.ref === item.title), false);
  assert.equal(state.inspectionCoordinations.some((record) => record.planningItemId === item.id), false);
  ```

- [x] **Step 3: Add validation and category-policy assertions.**

  Assert the exact mapping and rejection behavior:

  ```js
  assert.deepEqual(
    JSON.parse(JSON.stringify(context.inspectionCategoryPolicy('Routine / Announced'))),
    { inspectionCategory: 'Routine / Announced', noticePolicy: 'advance', coordinationStatus: 'ready_to_notify' }
  );
  assert.deepEqual(
    JSON.parse(JSON.stringify(context.inspectionCategoryPolicy('Ad Hoc / Unannounced'))),
    { inspectionCategory: 'Ad Hoc / Unannounced', noticePolicy: 'withheld', coordinationStatus: 'notice_withheld' }
  );
  assert.throws(
    () => context.createPlanningInspection(state, Object.assign({}, input, { inspectionCategory: '' }), {
      role: 'manager', name: context.ROLES.manager.user
    }),
    /Inspection Category is required/i
  );
  assert.throws(
    () => context.createPlanningInspection(state, input, { role: 'inspector', name: 'Wrong role' }),
    /Department Manager/i
  );
  ```

- [x] **Step 4: Add the complete approval, release, preparation, and materialization assertions.**

  Advance the created item through the existing helpers, then assert:

  ```js
  context.applyApprovalDecision(item, {
    decision: 'approve',
    actor: { role: 'finance', name: context.ROLES.finance.user },
    comment: 'No additional budget required.'
  });
  context.applyApprovalDecision(item, {
    decision: 'forward',
    actor: { role: 'gm', name: context.ROLES.gm.user },
    comment: 'Forwarded for final approval.'
  });
  context.applyApprovalDecision(item, {
    decision: 'approve',
    actor: { role: 'executiveDirector', name: context.ROLES.executiveDirector.user },
    comment: 'Approved for controlled release.'
  });
  context.releasePlanningItem(item, { actorRole: 'gm', actorName: context.ROLES.gm.user });
  context.acceptReleasedPlanningItem(item, { actorRole: 'manager', actorName: context.ROLES.manager.user });
  context.assignLeadInspectorToPlanningItem(item, {
    actorRole: 'manager', actorName: context.ROLES.manager.user, leadInspector: 'Caner Yildiz'
  });
  context.proposePlanningTeamAndSchedule(item, {
    actorRole: 'leadInspector',
    actorName: 'Caner Yildiz',
    team: ['Caner Yildiz', 'Aylin Sezer'],
    startDate: '2026-07-24',
    endDate: '2026-07-24',
    resources: 'Two inspectors; internal checklist package.'
  });
  context.confirmPlanningPreparation(item, { actorRole: 'manager', actorName: context.ROLES.manager.user });

  const result = context.materializeReadyPlanningInspection(state, item);
  assert.equal(result.created, true);
  assert.equal(result.audit.status, 'Scheduled');
  assert.equal(result.audit.inspectionCategory, 'Ad Hoc / Unannounced');
  assert.equal(result.audit.noticePolicy, 'withheld');
  assert.equal(result.coordination.status, 'notice_withheld');
  assert.equal(result.coordination.noticePolicy, 'withheld');
  assert.equal(result.coordination.notifiedAt, '');
  assert.equal(context.serviceProviderInspectionCoordinationRows(state, 'ORG-SKY').length, 0);
  assert.equal(state.notifications.some((notification) => notification.role === 'auditee' && notification.organizationId === 'ORG-SKY'), false);

  const replay = context.materializeReadyPlanningInspection(state, item);
  assert.equal(replay.created, false);
  assert.equal(replay.audit.id, result.audit.id);
  assert.equal(state.audits.filter((audit) => audit.id === result.audit.id).length, 1);
  assert.equal(state.inspectionCoordinations.filter((record) => record.auditId === result.audit.id).length, 1);
  ```

- [x] **Step 5: Add a Routine / Announced mirror assertion.**

  Create and advance a second item with `inspectionCategory: 'Routine / Announced'`, then assert its materialized coordination record has:

  ```js
  assert.equal(routineResult.audit.noticePolicy, 'advance');
  assert.equal(routineResult.coordination.status, 'ready_to_notify');
  assert.equal(routineResult.coordination.noticePolicy, 'advance');
  assert.equal(routineResult.coordination.notifiedAt, '');
  ```

  Do not send a Service Provider notification during materialization; the existing Lead Inspector action remains responsible for that transition.

- [x] **Step 6: Add rendered-surface assertions.**

  Update `tests/planning-workspace-smoke.test.js` and `tests/table-first-workbench-smoke.test.js` to prove:

  ```js
  context.state.role = 'manager';
  context.state.view = 'planning';
  let html = context.viewPlanningWorkspace();
  assert.match(html, /\+ New Inspection/);

  context.state.view = 'calendar';
  html = context.viewCalendar();
  assert.doesNotMatch(html, /\+ New Audit/);

  context.state.wizard = {
    step: 2,
    orgId: 'ORG-SKY',
    type: 'Special Inspection',
    domain: 'Security',
    inspectionCategory: 'Ad Hoc / Unannounced',
    noticePolicy: 'withheld'
  };
  html = context.viewAuditWizard();
  assert.match(html, /Ad Hoc \/ Unannounced/);
  assert.match(html, /Service Provider will not be informed in advance/);
  assert.doesNotMatch(html, /Lead inspector|Team members/);
  assert.doesNotMatch(html, /Create &amp; schedule audit/);
  ```

- [x] **Step 7: Run the focused tests and confirm the expected red state.**

  Run:

  ```bash
  node --test tests/unannounced-inspection-intake-smoke.test.js tests/planning-workspace-smoke.test.js tests/table-first-workbench-smoke.test.js
  ```

  Expected before implementation: failures report missing intake/materialization helpers, missing `+ New Inspection`, and legacy `+ New Audit` / wizard copy.

---

## Task 2: Implement Governed Planning Intake Helpers

**Files:**

- Modify: `js/planning.js`

**Interfaces:**

- Consumes: `target.planningItems`, `target.orgs`, `normalizeApprovalText()`, `DEMO_TODAY`, and the canonical role keys.
- Produces:
  - `inspectionCategoryPolicy(category): { inspectionCategory, noticePolicy, coordinationStatus }`
  - `nextPlanningInspectionId(target): string`
  - `createPlanningInspection(target, input, actor): PlanningItem`

- [x] **Step 1: Add the category-policy mapping.**

  Add this mapping and helper near the top of `js/planning.js`:

  ```js
  var INSPECTION_CATEGORY_POLICIES = {
    'Routine / Announced': {
      inspectionCategory: 'Routine / Announced',
      noticePolicy: 'advance',
      coordinationStatus: 'ready_to_notify'
    },
    'Ad Hoc / Unannounced': {
      inspectionCategory: 'Ad Hoc / Unannounced',
      noticePolicy: 'withheld',
      coordinationStatus: 'notice_withheld'
    }
  };

  function inspectionCategoryPolicy(category) {
    var policy = INSPECTION_CATEGORY_POLICIES[category];
    if (!policy) throw new Error('Inspection Category is required.');
    return Object.assign({}, policy);
  }
  ```

- [x] **Step 2: Add deterministic dynamic Planning IDs.**

  ```js
  function nextPlanningInspectionId(target) {
    var max = (target.planningItems || []).reduce(function (current, item) {
      var match = String(item.id || '').match(/^PLAN-2026-INS-(\d{3})$/);
      return match ? Math.max(current, Number(match[1])) : current;
    }, 0);
    return 'PLAN-2026-INS-' + String(max + 1).padStart(3, '0');
  }
  ```

- [x] **Step 3: Add input validation and organization lookup.**

  Validate the Department Manager role and every field required to make the Planning dossier understandable:

  ```js
  function planningOrganizationForIntake(target, organizationId) {
    return (target.orgs || []).filter(function (organization) {
      return organization.id === organizationId;
    })[0] || null;
  }

  function validatePlanningInspectionInput(target, input, actor) {
    if (!actor || actor.role !== 'manager') throw new Error('Department Manager is required to create an inspection plan.');
    if (!planningOrganizationForIntake(target, input.organizationId)) throw new Error('Organization is required.');
    if (!normalizeApprovalText(input.applicationType)) throw new Error('Application type is required.');
    if (!normalizeApprovalText(input.domain)) throw new Error('Domain is required.');
    inspectionCategoryPolicy(input.inspectionCategory);
    if (!normalizeApprovalText(input.purpose)) throw new Error('Inspection purpose is required.');
    if (!normalizeApprovalText(input.plannedDate)) throw new Error('Planned date is required.');
    if (!normalizeApprovalText(input.location)) throw new Error('Location is required.');
    if (!normalizeApprovalText(input.templateId)) throw new Error('Checklist template is required.');
    var requestedBudget = Number(input.requestedBudget || 0);
    if (!Number.isFinite(requestedBudget) || requestedBudget < 0) throw new Error('Requested budget must be zero or a positive number.');
    return requestedBudget;
  }
  ```

- [x] **Step 4: Create a Planning item, not an Audit.**

  Implement `createPlanningInspection()` so the returned item includes the exact canonical fields and chain:

  ```js
  function createPlanningInspection(target, input, actor) {
    input = input || {};
    var requestedBudget = validatePlanningInspectionInput(target, input, actor);
    var organization = planningOrganizationForIntake(target, input.organizationId);
    var policy = inspectionCategoryPolicy(input.inspectionCategory);
    var currency = normalizeApprovalText(input.currency) || 'USD';
    var id = nextPlanningInspectionId(target);
    var item = {
      id: id,
      title: input.applicationType + ' - ' + organization.name,
      department: input.domain,
      organization: organization.name,
      organizationId: organization.id,
      applicationType: input.applicationType,
      inspectionCategory: policy.inspectionCategory,
      noticePolicy: policy.noticePolicy,
      purpose: normalizeApprovalText(input.purpose),
      riskCategory: normalizeApprovalText(input.riskCategory) || 'Configured inspection risk',
      triggerType: normalizeApprovalText(input.triggerType) || 'Department Manager initiated',
      plannedDate: input.plannedDate,
      targetMonth: String(input.plannedDate).slice(0, 7),
      mode: input.mode || 'On-site',
      location: normalizeApprovalText(input.location),
      templateId: input.templateId,
      scope: normalizeApprovalText(input.scope),
      budgetRequired: requestedBudget > 0,
      requestedBudget: currency + ' ' + requestedBudget.toLocaleString('en-US'),
      budget: {
        currency: currency,
        requested: requestedBudget,
        availableForPlan: requestedBudget,
        remainingAnnualBudget: 0,
        lines: [{ category: 'Inspection resources', amount: requestedBudget }]
      },
      proposedInspectors: [],
      status: 'sent_to_finance',
      financeReview: null,
      auditId: '',
      preparation: {
        status: 'not_released',
        releasedBy: null,
        releasedDate: null,
        acceptedBy: null,
        acceptedDate: null,
        leadInspector: null,
        proposedTeam: [],
        proposedStartDate: null,
        proposedEndDate: null,
        resources: null,
        assignmentPackage: null,
        history: []
      },
      approval: {
        chain: [
          { role: 'manager', label: 'Department Manager', returnToRole: null },
          { role: 'finance', label: 'Finance Review', returnToRole: 'manager', notApprovedReturnToRole: 'manager' },
          { role: 'gm', label: 'GM Review', returnToRole: 'manager' },
          { role: 'executiveDirector', label: 'Executive Director Approval', returnToRole: 'gm' }
        ],
        currentIndex: 1,
        outcome: null,
        returnPolicy: 'configured_role',
        history: [{
          actor: actor.name,
          role: actor.role,
          action: 'submitted',
          date: approvalDecisionDate(),
          comment: 'Submitted ' + policy.inspectionCategory + ' inspection plan for Finance Review.'
        }]
      }
    };
    target.planningItems.unshift(item);
    return item;
  }
  ```

- [x] **Step 5: Run the helper-focused test and confirm the intake half passes.**

  Run:

  ```bash
  node --test tests/unannounced-inspection-intake-smoke.test.js
  ```

  Expected: intake, category, validation, and approval-owner assertions pass; materialization assertions still fail because `materializeReadyPlanningInspection()` is not implemented.

---

## Task 3: Materialize The Executable Audit Only After Preparation

**Files:**

- Modify: `js/planning.js`
- Modify: `tests/planning-release-smoke.test.js`
- Modify: `tests/inspection-coordination-smoke.test.js`

**Interfaces:**

- Consumes: a `planningItem` whose `preparation.status` is `ready_for_execution`, `target.auditSeq`, `target.audits`, `target.inspectionTeams`, `target.inspectionCoordinations`, `target.inspectionWorkspaces`, and `target.users`.
- Produces: `materializeReadyPlanningInspection(target, item): { audit, coordination, created }`.

- [x] **Step 1: Add ready-state and idempotency guards.**

  ```js
  function materializedPlanningAudit(target, item) {
    return item.auditId
      ? (target.audits || []).filter(function (audit) { return audit.id === item.auditId; })[0] || null
      : null;
  }

  function materializeReadyPlanningInspection(target, item) {
    if (!item || !item.preparation || item.preparation.status !== 'ready_for_execution') {
      throw new Error('Planning preparation must be ready for execution before creating the Audit.');
    }
    var existing = materializedPlanningAudit(target, item);
    if (existing) {
      return {
        audit: existing,
        coordination: inspectionCoordinationByAuditId(target, existing.id),
        created: false
      };
    }
  }
  ```

  Continue the same function in the following steps; do not leave an early undefined return after Task 3 is complete.

- [x] **Step 2: Create the executable Audit with inherited policy.**

  Add this body after the idempotency branch:

  ```js
  var policy = inspectionCategoryPolicy(item.inspectionCategory);
  var auditId = 'AUD-2026-' + String(target.auditSeq).padStart(3, '0');
  var team = item.preparation.proposedTeam.slice();
  if (team.indexOf(item.preparation.leadInspector) === -1) team.unshift(item.preparation.leadInspector);
  var audit = {
    id: auditId,
    ref: item.title,
    orgId: item.organizationId,
    type: item.applicationType,
    domain: item.department,
    templateId: item.templateId,
    date: item.preparation.proposedStartDate || item.plannedDate,
    endDate: item.preparation.proposedEndDate || item.plannedDate,
    mode: item.mode,
    location: item.location,
    inspectionCategory: policy.inspectionCategory,
    noticePolicy: policy.noticePolicy,
    lead: item.preparation.leadInspector,
    team: team,
    status: 'Scheduled',
    checklistStarted: false,
    planningItemId: item.id
  };
  target.audits.push(audit);
  target.auditSeq += 1;
  item.auditId = audit.id;
  item.preparation.assignmentPackage.auditId = audit.id;
  ```

- [x] **Step 3: Create the inspection team and workspace.**

  ```js
  var userIdByName = {};
  (target.users || []).forEach(function (user) { userIdByName[user.name] = user.id; });
  target.inspectionTeams.push({
    id: 'TEAM-' + audit.id,
    auditId: audit.id,
    department: audit.domain,
    status: 'Scheduled',
    startDate: audit.date,
    endDate: audit.endDate,
    leadUserId: userIdByName[audit.lead] || '',
    memberIds: audit.team.map(function (name) { return userIdByName[name] || ''; }).filter(Boolean),
    notes: item.preparation.resources,
    attachments: [],
    messages: [],
    history: [{ at: approvalDecisionDate(), actor: ROLES.manager.user, action: 'Inspection team confirmed from Planning' }]
  });
  target.inspectionWorkspaces[audit.id] = {
    selectedSectionKey: 'galley',
    answersByQuestionId: {},
    downloadedAt: '',
    downloadedAttachmentIds: {},
    draftSavedAt: '',
    allSectionsCompletedAt: '',
    submittedAt: '',
    submittedByUserId: '',
    lastSubmittedAt: '',
    lastSubmittedByUserId: '',
    reopenedAt: '',
    reopenedByUserId: ''
  };
  ```

- [x] **Step 4: Create exactly one coordination record.**

  ```js
  var template = (target.templateLibrary || []).filter(function (candidate) {
    return candidate.id === audit.templateId;
  })[0] || null;
  var coordination = {
    auditId: audit.id,
    planningItemId: item.id,
    organizationId: audit.orgId,
    inspectionCategory: policy.inspectionCategory,
    noticePolicy: policy.noticePolicy,
    status: policy.coordinationStatus,
    proposedDate: audit.date,
    alternativeDate: '',
    confirmedDate: policy.noticePolicy === 'withheld' ? audit.date : '',
    checklistName: template ? template.name : 'Inspection Checklist',
    checklistFiles: [],
    sharedInformation: policy.noticePolicy === 'advance'
      ? ['Inspection scope', 'On-site location', 'Lead Inspector contact']
      : [],
    notifiedAt: '',
    respondedAt: '',
    caaConfirmedAt: '',
    providerComment: '',
    history: policy.noticePolicy === 'withheld'
      ? [{ at: approvalDecisionDate(), actor: 'System', action: 'Advance notice withheld by configured inspection policy' }]
      : []
  };
  target.inspectionCoordinations.push(coordination);
  return { audit: audit, coordination: coordination, created: true };
  ```

- [x] **Step 5: Extend release and coordination tests.**

  In `tests/planning-release-smoke.test.js`, materialize the approved/prepared item and assert `item.auditId`, Audit status, and one inspection team. In `tests/inspection-coordination-smoke.test.js`, keep the seed assertions and add one dynamically materialized Ad Hoc record that remains absent from `serviceProviderInspectionCoordinationRows()`.

- [x] **Step 6: Run the lifecycle-focused tests.**

  Run:

  ```bash
  node --test tests/unannounced-inspection-intake-smoke.test.js tests/planning-release-smoke.test.js tests/inspection-coordination-smoke.test.js tests/inspection-lifecycle-alignment-smoke.test.js
  ```

  Expected: all tests pass, including idempotency and Service Provider privacy.

---

## Task 4: Replace The Legacy Department Manager Creation Surface

**Files:**

- Modify: `js/app.js`
- Modify: `js/views.js`
- Modify: `js/data.js`
- Modify: `css/styles.css`
- Modify: `tests/planning-workspace-smoke.test.js`
- Modify: `tests/table-first-workbench-smoke.test.js`

**Interfaces:**

- Consumes: `createPlanningInspection()`, `inspectionCategoryPolicy()`, and `materializeReadyPlanningInspection()`.
- Produces: the visible Department Manager flow `Planning -> + New Inspection -> Submit for Finance Review -> selected Planning Command Center`.

- [x] **Step 1: Move the primary creation action to Planning.**

  In `viewPlanningWorkspace()`, pass this action to `pageHead()` only for the Department Manager:

  ```js
  var planningActions = state.role === 'manager'
    ? '<button class="btn btn--primary btn--sm" data-act="new-planning-inspection">+ New Inspection</button>'
    : '';
  ```

  Use:

  ```js
  pageHead(workspaceTitle, workspacePurpose, planningActions)
  ```

  In `viewCalendar()`, remove the role-specific block that appends `+ New Audit`.

- [x] **Step 2: Initialize intake state without premature assignment.**

  Replace the visible semantics of `startWizard()` while keeping the function as a compatibility alias:

  ```js
  function startPlanningInspectionIntake() {
    state.wizard = {
      step: 1,
      orgId: state.orgs[0].id,
      type: AUDIT_TYPES[0],
      domain: AUDIT_DOMAINS[0],
      inspectionCategory: 'Routine / Announced',
      noticePolicy: 'advance',
      purpose: '',
      triggerType: 'Department Manager initiated',
      riskCategory: '',
      date: '2026-12-10',
      mode: 'On-site',
      location: '',
      templateId: 'TPL-CABIN-2026',
      scope: '',
      currency: 'USD',
      requestedBudget: '0'
    };
    go('wizard');
  }

  function startWizard() {
    startPlanningInspectionIntake();
  }
  ```

  Add `case 'new-planning-inspection': startPlanningInspectionIntake(); break;` and keep `new-audit` mapped to the same function only as an internal compatibility alias.

- [x] **Step 3: Capture the new fields and remove assignment capture.**

  Extend `wizardCapture()` with:

  ```js
  if (document.getElementById('wz-inspection-category')) w.inspectionCategory = val('wz-inspection-category');
  if (document.getElementById('wz-purpose')) w.purpose = val('wz-purpose');
  if (document.getElementById('wz-trigger')) w.triggerType = val('wz-trigger');
  if (document.getElementById('wz-risk')) w.riskCategory = val('wz-risk');
  if (document.getElementById('wz-budget')) w.requestedBudget = val('wz-budget');
  if (document.getElementById('wz-currency')) w.currency = val('wz-currency');
  w.noticePolicy = inspectionCategoryPolicy(w.inspectionCategory).noticePolicy;
  ```

  Remove `wz-lead`, `.wz-team`, `lead`, and `team` intake handling.

  Give the category select `data-field="wizard-inspection-category"` and add this branch to the existing document `change` listener so the policy explanation updates immediately:

  ```js
  if (field === 'wizard-inspection-category') {
    state.wizard.inspectionCategory = e.target.value;
    state.wizard.noticePolicy = inspectionCategoryPolicy(e.target.value).noticePolicy;
    render();
  }
  ```

- [x] **Step 4: Render five clear intake steps.**

  Keep five steps but replace their labels with:

  ```js
  var steps = [
    'Inspection basics',
    'Category and purpose',
    'When and where',
    'Checklist, scope and budget',
    'Review and submit'
  ];
  ```

  Step 2 must render the canonical categories and the selected policy explanation:

  ```html
  <select id="wz-inspection-category">
    <option>Routine / Announced</option>
    <option>Ad Hoc / Unannounced</option>
  </select>
  ```

  For Ad Hoc / Unannounced, show:

  ```text
  No Advance Notice — the Service Provider will not be informed in advance and the coordination branch will be skipped.
  ```

  For Routine / Announced, show:

  ```text
  Advance Notice Required — coordination starts only after approval, release, and Lead Inspector assignment.
  ```

  Step 5 must show `Inspection Category`, `Notification Policy`, and `Approval Path`; it must not show Lead Inspector or Team.

  Change `VIEW_TITLES.wizard` from `New Audit Wizard` to `New Inspection`, and make the Step 1 Cancel button navigate to `planning` instead of `calendar`.

- [x] **Step 5: Validate each transition and submit to Finance Review.**

  In `wizardNext()`, require purpose on Step 2 and location on Step 3. Replace `wizardCreate()` with:

  ```js
  function wizardCreate() {
    wizardCapture();
    var w = state.wizard;
    try {
      var item = createPlanningInspection(state, {
        organizationId: w.orgId,
        applicationType: w.type,
        domain: w.domain,
        inspectionCategory: w.inspectionCategory,
        purpose: w.purpose,
        triggerType: w.triggerType,
        riskCategory: w.riskCategory,
        plannedDate: w.date,
        mode: w.mode,
        location: w.location,
        templateId: w.templateId,
        scope: w.scope,
        currency: w.currency,
        requestedBudget: Number(w.requestedBudget || 0)
      }, {
        role: state.role,
        name: ROLES[state.role].user
      });
      state.wizard = null;
      addLog('Inspection plan submitted to Finance Review', item.id);
      pushNotification('finance', '🧾', item.id + ' is waiting for Finance Review.');
      toast('Inspection submitted', item.id + ' sent to Finance Review.', 'ok');
      persistAfterAction();
      go('planning', { planningId: item.id, tab: 'overview' });
    } catch (error) {
      toast('Inspection not submitted', error.message, 'warn');
    }
  }
  ```

  Change the final visible button to `Submit for Finance Review` and the page title to `New Inspection`.

  Extend `go()` so the selected dynamic Planning item is preserved:

  ```js
  if (opts.planningId !== undefined && opts.planningId !== null && opts.planningId !== '') {
    state.params.planningId = opts.planningId;
  }
  ```

- [x] **Step 6: Materialize after preparation confirmation.**

  In `handlePlanningConfirmPrep()`, call `materializeReadyPlanningInspection(state, item)` immediately after `confirmPlanningPreparation()` succeeds. Log the created Audit ID, keep the Planning item selected, and show a success toast that distinguishes the notice policy:

  ```js
  var result = materializeReadyPlanningInspection(state, item);
  addLog('Audit created from approved Planning item', result.audit.id);
  toast(
    'Ready for Execution',
    result.audit.id + (result.audit.noticePolicy === 'withheld'
      ? ' created with No Advance Notice.'
      : ' created; Service Provider coordination is ready.'),
    'ok'
  );
  ```

- [x] **Step 7: Add the Audit link to the preparation package.**

  When `item.auditId` exists, `planningAssignmentPackageHtml(item)` must render:

  ```html
  <button class="btn btn--primary" data-act="nav" data-view="audit-detail" data-id="AUD-ID">Open Audit Work Item</button>
  ```

  Use the actual escaped `item.auditId` in the implementation.

- [x] **Step 8: Add focused policy styling.**

  Add a small `.inspection-policy-callout` block that uses existing `--line`, `--warn-bg`, `--info-bg`, and text tokens. At widths below the current mobile breakpoint, ensure intake columns collapse to one column without horizontal overflow. Do not introduce a new design system or broad page restyling.

- [x] **Step 9: Run the UI/render smoke tests.**

  Run:

  ```bash
  node --test tests/unannounced-inspection-intake-smoke.test.js tests/planning-workspace-smoke.test.js tests/table-first-workbench-smoke.test.js tests/audit-work-queue-smoke.test.js
  ```

  Expected: all tests pass and no visible legacy creation copy remains in the Department Manager flow.

---

## Task 5: Prove Persistence, Privacy, And Regression Safety

**Files:**

- Modify: `tests/unannounced-inspection-intake-smoke.test.js`
- Modify: `tests/inspection-coordination-smoke.test.js`
- Modify: `js/data.js` only if the failing round-trip test requires normalization.

**Interfaces:**

- Consumes: `mergeDemoState()`, dynamically created Planning/Audit/coordination/team/workspace records.
- Produces: proof that browser-local reload preserves the new lifecycle without disclosing unannounced records.

- [x] **Step 1: Add a JSON round-trip assertion.**

  After materialization in the focused test, run:

  ```js
  const restored = context.mergeDemoState(JSON.parse(JSON.stringify(state)));
  const restoredItem = restored.planningItems.filter((candidate) => candidate.id === item.id)[0];
  const restoredAudit = restored.audits.filter((candidate) => candidate.id === item.auditId)[0];
  const restoredCoordination = context.inspectionCoordinationByAuditId(restored, item.auditId);

  assert.equal(restoredItem.noticePolicy, 'withheld');
  assert.equal(restoredItem.preparation.status, 'ready_for_execution');
  assert.equal(restoredAudit.noticePolicy, 'withheld');
  assert.equal(restoredCoordination.status, 'notice_withheld');
  assert.equal(context.serviceProviderInspectionCoordinationRows(restored, restoredAudit.orgId).some((row) => row.auditId === restoredAudit.id), false);
  ```

- [x] **Step 2: Change normalization only if the test fails.**

  If `mergeDemoState()` drops a dynamic field, add explicit default-preserving assignments for `applicationType`, `inspectionCategory`, `noticePolicy`, `plannedDate`, `templateId`, `scope`, and `auditId`. Do not replace dynamic values with the first seed plan's values.

- [x] **Step 3: Run syntax and targeted tests.**

  Run:

  ```bash
  node --check js/data.js
  node --check js/planning.js
  node --check js/views.js
  node --check js/app.js
  node --test tests/unannounced-inspection-intake-smoke.test.js tests/planning-release-smoke.test.js tests/planning-workspace-smoke.test.js tests/inspection-coordination-smoke.test.js tests/inspection-lifecycle-alignment-smoke.test.js tests/table-first-workbench-smoke.test.js tests/audit-work-queue-smoke.test.js
  ```

  Expected: syntax checks exit `0`; all targeted tests pass.

- [x] **Step 4: Run the full local smoke suite.**

  Run:

  ```bash
  node --test tests/*.test.js
  ```

  Expected: all tests pass with zero failures. Record the exact test count in the plan and build summary; do not reuse an older count.

---

## Task 6: Verify The Rendered Department Manager Flow

**Files:**

- Evidence-only unless a defect is found in the changed runtime files.

**Interfaces:**

- Consumes: the implemented static demo and local browser.
- Produces: desktop/mobile screenshots and an interaction log for the exact changed path.

- [x] **Step 1: Start the local static server.**

  Run from the project root:

  ```bash
  python3 -m http.server 4360
  ```

- [x] **Step 2: Verify the desktop path at approximately `1440x900`.**

  Execute exactly:

  ```text
  Role Select -> Department Manager -> Planning -> + New Inspection
  -> Special Inspection -> Ad Hoc / Unannounced
  -> enter purpose, date, location, checklist, scope, and zero budget
  -> Submit for Finance Review
  ```

  Verify:

  - The resulting page is `Department Planning`, not `Audit Detail`.
  - The new `PLAN-2026-INS-###` item is selected.
  - Current Owner is `Finance Review`.
  - Next Action is Finance budget review.
  - The dossier shows `Ad Hoc / Unannounced` and `No Advance Notice`/`withheld` meaning.
  - No executable Audit row has been created yet.
  - No Service Provider notification is visible.

- [x] **Step 3: Verify the lifecycle completion path.**

  Use role switching to complete Finance Review, GM Review, Executive Director approval, GM Release, Department acceptance, Lead assignment, Lead team/date/resource proposal, and Department confirmation. Verify the final Planning package links to one scheduled Audit and the coordination state says `No Advance Notice`.

- [x] **Step 4: Verify Service Provider privacy.**

  Switch to the matching Service Provider portal and confirm the newly created Ad Hoc / Unannounced Audit is absent from `Inspection Coordination`. Confirm unrelated Routine / Announced coordination remains functional.

- [x] **Step 5: Verify mobile layout at approximately `390x844`.**

  Repeat intake Step 2 and the submission result. Confirm no horizontal overflow, clipped category options, hidden primary button, overlapping sidebar, or unreadable policy callout.

- [x] **Step 6: Review console and clean up.**

  Record zero unexpected console errors, close automation tabs, stop the local server, and check for leftover Chrome, Chrome Helper, webdriver, Playwright, Puppeteer, or local server processes.

---

## Task 7: Synchronize Documentation, Evidence, And Plan Tracking

**Files:**

- Modify: `docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.md`
- Modify: `docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.turkce.md`
- Modify: `docs/demo-evidence/BUILD_SUMMARY.md`
- Modify: `docs/demo-evidence/BUILD_SUMMARY.turkce.md`
- Modify: `MANIFEST.md`
- Modify: `docs/exec-plans/index.md`
- Modify: this plan.

**Interfaces:**

- Consumes: fresh command output and browser evidence from Tasks 5-6.
- Produces: synchronized stakeholder-facing workflow truth and an accurate active-plan next todo.

- [x] **Step 1: Update the screen inventory.**

  Replace the legacy New Audit Wizard description with the new Planning intake fields and result:

  ```text
  Organization, Application Type, Domain, Inspection Category, Advance-notice Policy, Purpose/Trigger, Planned Date, Mode, Location, Checklist Template, Scope, Requested Budget, Approval Path.
  ```

  State that Lead Inspector/team selection occurs after approval and GM release, and submission returns to the selected Planning Command Center item.

- [x] **Step 2: Update the Turkish companion with equivalent meaning.**

  Use `Ad Hoc / Unannounced`, `Routine / Announced`, `Advance notification withheld`, `Finance Review`, and `GM Release to Department` consistently; do not translate canonical status values into incompatible identifiers.

- [x] **Step 3: Update bilingual build evidence.**

  Add the exact focused/full test counts, desktop/mobile browser path, console result, Service Provider privacy result, and process cleanup result. Keep the explicit frontend-only and production-readiness boundary.

- [x] **Step 4: Update package inventory.**

  Add `tests/unannounced-inspection-intake-smoke.test.js` and this plan file to `MANIFEST.md` if they are not already listed.

- [x] **Step 5: Close plan checkboxes and update the active-plan row.**

  If implementation and required verification pass, set this plan's index status to `ready-for-verification` with the one next todo:

  ```text
  Stakeholder review/sign-off of the Department Manager Ad Hoc / Unannounced intake and withheld-notice behavior.
  ```

  Keep the plan under `active/` until stakeholder sign-off. Do not move it to `completed/` in the implementation task.

- [x] **Step 6: Run docs and diff checks.**

  Run:

  ```bash
  git diff --check
  node tests/harness-docs-smoke.test.js
  node tests/demo-boundary-smoke.test.js
  rg -n "Ad Hoc / Unannounced|Advance notification withheld|New Inspection|Finance Review|GM Release to Department" js tests docs MANIFEST.md
  ```

  Expected: all commands pass; search output shows aligned runtime, tests, canonical docs, Turkish companion, and evidence.

---

## Verification Matrix

| Risk | Required Evidence |
|---|---|
| Planning intake semantics | New focused smoke test proves a Planning item is created before an Audit |
| Approval authority | Existing approval and planning release tests remain green |
| Premature assignment | Render assertions prove intake omits Lead Inspector/team |
| Unannounced privacy | Dynamic coordination test and Service Provider browser check prove the record is hidden |
| Routine regression | Existing coordination test plus dynamic Routine mirror remain green |
| Premature execution | New item has no Audit until `ready_for_execution` |
| Duplicate records | Idempotent materialization assertions prove one Audit and one coordination record |
| Browser persistence | JSON round-trip through `mergeDemoState()` preserves plan/audit/policy |
| UI regression | Desktop/mobile click-through, console review, and overflow inspection |
| Demo boundary | `demo-boundary-smoke` and explicit build-summary language |
| Package truth | `MANIFEST.md`, plan index, plan checkboxes, and bilingual evidence agree |

## Risks And Mitigations

- **Risk: dynamic planning items inherit seed-only values during merge.** Mitigation: round-trip test dynamic IDs and change normalization only when the failing test proves a field is lost.
- **Risk: `Special Inspection` is treated as automatically unannounced.** Mitigation: keep application type and inspection category as separate required fields and test both.
- **Risk: an Audit is created before authority approval.** Mitigation: create only a planning item on submit and guard materialization on `ready_for_execution`.
- **Risk: Ad Hoc records leak to the Service Provider portal.** Mitigation: create `notice_withheld`, send no auditee notification, and assert organization-scoped coordination rows exclude the record before and after persistence.
- **Risk: Routine coordination breaks.** Mitigation: keep `ready_to_notify` and the existing Lead Inspector action; test a dynamic Routine item alongside seed records.
- **Risk: duplicate Audit creation after repeated clicks or reload.** Mitigation: store `item.auditId`, return the existing record on replay, and assert single-record counts.
- **Risk: existing dirty working-tree changes overlap `js/app.js`, `js/views.js`, CSS, evidence, or plan index.** Mitigation: inspect diffs before each edit, preserve unrelated hunks, and stop for user direction if safe separation is impossible.
- **Risk: the selected checklist cannot execute.** Mitigation: keep template identity and planning/notice behavior correct; do not claim every template is runnable unless the execution package test proves it.

## Dependencies

- Existing approval primitive in `js/approval.js`.
- Existing planning preparation helpers in `js/planning.js`.
- Existing role routing/actions in `js/app.js`.
- Existing Planning and intake renderers in `js/views.js`.
- Existing coordination privacy selectors and Service Provider portal behavior.
- Existing browser-local persistence through `freshState()`, `saveDemoState()`, and `mergeDemoState()`.
- Local Node runtime and browser automation available to this workspace.

## Completion Criteria

This plan reaches `ready-for-verification` only when:

1. Department Manager creates the inspection from Planning.
2. `Ad Hoc / Unannounced` and `noticePolicy: withheld` are explicit and persisted.
3. Submission creates a Planning item owned by Finance Review and stays in Planning.
4. Lead/team assignment is absent from intake and remains post-release.
5. An Audit is created only after preparation confirmation.
6. The resulting Ad Hoc coordination record is `notice_withheld` and absent from the Service Provider portal.
7. Dynamic Routine coordination remains `ready_to_notify` and usable.
8. Focused and full smoke suites pass with recorded fresh counts.
9. Desktop/mobile browser checks, console review, and process cleanup pass.
10. Bilingual docs, build evidence, manifest, plan, and active index are synchronized.

## Verification Record — 2026-07-20

Evidence label: `demo-only; verified locally; production-readiness not claimed`.

- Syntax: `node --check` passed for `js/data.js`, `js/planning.js`,
  `js/views.js`, and `js/app.js`.
- Targeted automation: the final seven-file command passed 16/16 with zero
  failures, cancellations, skips, or todos.
- Full automation: `node --test tests/*.test.js` passed 72/72 with zero
  failures, cancellations, skips, or todos.
- Desktop Browser (`1440x900`): the exact intake path stayed in selected
  Planning, entered Finance Review at zero budget, completed the approval /
  release / preparation chain, and produced exactly one `Scheduled` Audit
  after Department confirmation.
- Service Provider privacy: `AUD-2026-009` and the unannounced title were absent
  from the matching Fly Namibia portal; Routine `AUD-2026-001` coordination and
  its confirm/alternative actions remained visible.
- Mobile Browser (`390x844`): Step 2 and the selected Finance Review result had
  no document overflow; category, policy callout, closed sidebar, and primary
  action bounds were valid.
- Browser console: zero unexpected errors.
- Cleanup: Browser tabs and the local server were closed; process inspection
  found no task-owned server or automation residue.
- Evidence: screenshots and `interaction-log.txt` are under
  `/private/tmp/aviasurveil360-unannounced-intake-qa-20260720/`.
- Rendered QA found an overly broad Lead Planning shortcut. A regression test
  captured the role-boundary conflict; the final implementation exposes only a
  scoped post-release preparation task and the final 72/72 suite covers both
  the restriction and the task route.

## Execution Prompt

```text
Implement docs/exec-plans/active/2026-07-20-unannounced-inspection-intake-alignment-plan.md in /Users/marlonjd/Developer/monorepos/avia/apps/surveil according to its defined scope and acceptance criteria. Follow the repo AGENTS.md and all linked source-of-truth documents. Work in the current local checkout; do not create or switch branches, commit, push, or overwrite unrelated dirty working-tree changes. Preserve the frontend-only demo boundary. The required outcome is: Department Manager creates an Ad Hoc / Unannounced inspection from Planning, submission stays in the selected Planning Command Center item and enters Finance Review, Lead/team assignment remains post-release, the executable Audit is materialized only after preparation confirmation, and the Service Provider never receives advance notice or portal visibility for the unannounced inspection. Run every focused/full test and desktop/mobile browser verification in the plan, keep plan checkboxes and docs/exec-plans/index.md synchronized with actual progress, update bilingual build evidence, and finish with the harness Done/Remaining/Blocked/Verification/Next readout.
```
