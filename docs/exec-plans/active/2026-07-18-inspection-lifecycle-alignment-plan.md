# Inspection Lifecycle Alignment Implementation Plan


**Goal:** Align the frontend-only AviaSurveil360 demo with the accepted inspection lifecycle, including George's Service Provider coordination rule, the full Preliminary Report approval chain, explicit CAP verification outcomes, the existing GM release step, and honest demo-boundary language.

**Architecture:** Keep the current static HTML/CSS/Vanilla JavaScript architecture and browser-local state model. Extend the existing report state machine and role workspaces instead of creating a parallel workflow, add one explicit CAP verification decision primitive, and use tests plus bilingual product documentation as the lifecycle contract.

**Tech Stack:** HTML, CSS, Vanilla JavaScript, browser-local mock state, Node.js `node:test`/assert smoke tests, and Playwright-based local rendered QA.

**Status:** `ready-for-verification` as of 2026-07-18. Implementation and local
verification are complete; stakeholder review/sign-off is the only next action.

**Verification recorded:** focused lifecycle suite 18/18; full local suite
66/66; JavaScript syntax and harness documentation checks passed; isolated
rendered QA completed at `1440x900`, `1024x768`, and `390x844` with zero
browser console warnings/errors and zero page-level horizontal overflow.
Temporary screenshots are under
`/private/tmp/aviasurveil360-lifecycle-qa-20260718/`. Production behavior and
stakeholder acceptance are not verified.

## Global Constraints

- Use `AviaSurveil360` as the canonical product name.
- Keep the implementation frontend-only and demo-only.
- Do not add a backend, database, API, real authentication, real authorization enforcement, real file upload/storage, real notification delivery, real e-signature, production audit log, framework migration, or legal/enforcement automation.
- Preserve the completed George coordination rule: Routine inspections notify the Service Provider only after Lead Inspector assignment; Ad Hoc / Unannounced inspections withhold advance notice; notified providers may confirm or propose an alternative date; the shared package includes checklist files and relevant information.
- Preserve the Finding -> CAP -> Evidence -> CAA Review -> Closure workflow.
- CAP acceptance is not Finding closure.
- `Partially Close` and `Not Close` outcomes must leave the Finding open.
- Enforcement remains a recommendation/referral for separate authorized human review and must never apply a sanction automatically.
- Keep `Comment to Auditee` separate from `Internal CAA Note`.
- Preserve Evidence version history; do not overwrite earlier Evidence records.
- Keep existing browser-local Preliminary Reports that were already released as historical records; do not silently move them backward into a pending approval state.
- Update canonical English stakeholder/product documents and their existing `.turkce.md` companions together.
- Do not create a branch, commit, push, or publish unless the user explicitly authorizes that Git action during execution.
- Bump every CSS/JavaScript asset query token in `index.html` whenever visible CSS or JavaScript behavior changes.

---

## Objective

Make the accepted eight-stage inspection lifecycle truthful and consistently visible in the demo:

1. Annual Planning: Department Manager -> Finance -> General Manager -> Executive Director -> GM Release to Department -> plan preparation.
2. Inspection Assignment: Department Manager assigns Lead Inspector; Lead Inspector assigns the team and checklist; George's Routine versus Ad Hoc Service Provider coordination rule applies.
3. Inspection Execution: inspectors complete checklist work, Findings, observations, and mock Evidence filenames.
4. Preliminary Report: Lead Inspector -> Department Manager -> General Manager -> Executive Director -> issued to the Service Provider.
5. Corrective Action Plan: Service Provider submits root cause, corrective action, preventive action, responsible person, target date, and mock Evidence filenames.
6. CAP Verification: the inspection team records `Close`, `Partially Close`, or `Not Close`; only `Close` closes the Finding.
7. Final Report: Lead Inspector -> Department Manager -> General Manager -> Executive Director -> issued and locked with a demo-only approval mark.
8. Enforcement And Monitoring: Executive Director may record a recommendation/referral for separate authorized review; dashboards remain informational.

The lifecycle footer must describe actual demo behavior:

- approvals and timestamps are browser-local mock records;
- traceability is demo audit history, not a production audit trail;
- attachments are mock filenames/local browser state, not secure document storage.

## Scope

### In Scope

- Correct the Preliminary Report chain so CAP-required reports no longer bypass General Manager and Executive Director review.
- Extend the General Manager Report Approvals workspace to show both Preliminary and Final Reports.
- Add an Executive Director Preliminary Reports approval workspace and role navigation entry.
- Release a Preliminary Report to the Service Provider only after Executive Director approval.
- Keep CAP-required and no-CAP-required reports on the same approval chain; use the CAP flag only to determine the post-release Service Provider action.
- Add explicit CAP verification outcomes: `close`, `partially_close`, and `not_close`.
- Keep `partially_close` and `not_close` Findings open with a visible next action.
- Preserve the existing GM Release to Department step after Executive Director planning approval and show it in lifecycle documentation/copy.
- Add regression tests for authority, exact report identity, notifications, organization scoping, Finding closure, enforcement boundaries, and demo claims.
- Update English/Turkish product specs, build evidence, plan index, and durable tracker records.
- Run focused and full local automated checks plus rendered desktop/tablet/mobile workflow QA.

### Out Of Scope

- Redrawing or overwriting the stakeholder's original temporary PNG source.
- Removing the GM Release to Department step from the implemented planning workflow.
- Changing George's already-implemented Service Provider coordination behavior.
- Production identity, signing, records management, secure storage, or notification delivery.
- Real sanctions, certificate actions, administrative penalties, suspensions, or enforcement execution.
- Changing the legal meaning of `Close`, `Partially Close`, or `Not Close`; these remain configured demo review outcomes pending regulatory-owner validation.
- Broad Modern Aviation SaaS restyling; responsive support needed by the new controls is included, but the visual rollout remains in its separate active plan.

## Assumptions

- The lifecycle image and the user's accepted follow-up are the stakeholder direction for this plan.
- The existing GM Release to Department step is intentional and should be added to the canonical lifecycle rather than removed from code.
- A Preliminary Report is issued to the Service Provider after Executive Director approval whether or not CAP is required.
- `capRequired` controls the Service Provider's next action after release:
  - `true`: respond with CAP and Evidence;
  - `false`: acknowledge/view the issued Preliminary Report; no CAP submission is requested.
- `Partially Close` means implementation is only partly verified; the Finding remains open and the Service Provider must provide the remaining action/Evidence.
- `Not Close` means verification failed or is insufficient; the Finding remains open and the Service Provider must correct/resubmit.
- Historical browser-local records already in `released_to_service_provider` or `closed` remain readable and are not retroactively returned to approvers.
- The current JavaScript files are intentionally large; this plan extends their established patterns without an unrelated module migration.

## Dependencies

- George coordination implementation and `tests/inspection-coordination-smoke.test.js` must remain passing.
- Existing shared report helpers in `js/reports.js` and role projections in `js/manager-workspaces.js` remain the only report source of truth.
- Existing `FINDING_STATUS` states remain authoritative; CAP verification adds a decision record and reuses `CLOSED` or `EVIDENCE_MORE_INFO` rather than adding ambiguous partial-closure statuses.
- Existing General Manager and Executive Director role shells remain available.
- Local Playwright/browser dependencies used by the prior 85-route audit remain available for rendered QA.

## Ownership Boundaries

| Area | Owner During This Plan | Boundary |
|---|---|---|
| Lifecycle/product rule | Product owner / stakeholder | Confirms the accepted sequence and wording; does not define production legal authority. |
| Static demo state machine | Frontend implementation | Implements browser-local transitions, guards, notifications, and visible next actions only. |
| Report approval authority | Configured demo roles | Department Manager, General Manager, and Executive Director actions are mock governance records, not real signatures. |
| CAP verification | Inspector / Lead Inspector demo roles | Records configured review outcomes; only `Close` changes the Finding to `CLOSED`. |
| Enforcement | Authorized product/legal owner outside this plan | Demo may create a recommendation/referral only. |
| Security/storage/signing | Future production architecture owners | No production claim or implementation is added here. |
| Stakeholder lifecycle graphic | Product/design handoff | Canonical Markdown/Mermaid is updated; replacement of the source PNG requires a separately supplied editable asset or explicit image-generation request. |

## File Map

### Runtime

- `js/data.js` - default UI state, saved-state normalization, report seed/state compatibility, and CAP verification record compatibility.
- `js/reports.js` - shared report lookup/read models and the new CAP verification decision primitive.
- `js/manager-workspaces.js` - Department Manager, General Manager, and Executive Director report transitions/projections.
- `js/views.js` - Department Manager, General Manager, Executive Director, Finding/Evidence review, Service Provider, and lifecycle copy.
- `js/app.js` - role routes, action handlers, notifications, persistence calls, and modal decisions.
- `css/styles.css` - responsive layout and decision-control styling for the new approval/verification controls only.
- `index.html` - synchronized cache-busting query token.

### Tests

- Create `tests/inspection-lifecycle-alignment-smoke.test.js` - end-to-end state-machine contract for planning, coordination, Preliminary Report, CAP verification, Final Report, and enforcement boundaries.
- Modify `tests/department-preliminary-review-smoke.test.js` - Department Manager always forwards Preliminary Reports to General Manager.
- Modify `tests/general-manager-workspace-smoke.test.js` - General Manager accepts Preliminary and Final Reports, preserving exact type and identity.
- Modify `tests/executive-director-workspace-smoke.test.js` - Executive Director sees/decides Preliminary Reports separately from Final Reports.
- Modify `tests/report-approval-smoke.test.js` - complete Preliminary and Final chains are isolated and deterministic.
- Modify `tests/service-provider-portal-smoke.test.js` - Preliminary Reports are invisible before Executive Director release and organization scoped after release.
- Modify `tests/manager-cap-monitoring-smoke.test.js` - partial/not-close decisions do not close Findings or falsely complete CAP metrics.
- Modify `tests/demo-boundary-smoke.test.js` - no secure-storage, real-signature, automatic-sanction, or production-audit-trail claim appears.
- Modify `tests/harness-docs-smoke.test.js` only if the new lifecycle document links require registry assertions.

### Product Documentation And Evidence

- Modify `docs/product-specs/workflows/MASTER_WORKFLOW.md` and `.turkce.md`.
- Modify `docs/product-specs/workflows/SURVEILLANCE_PLANNING_WORKFLOW.md` and `.turkce.md`.
- Modify `docs/product-specs/workflows/FINDING_CAP_EVIDENCE_WORKFLOW.md` and `.turkce.md`.
- Modify `docs/product-specs/modules/AUDIT_PLANNING.md` and `.turkce.md`.
- Modify `docs/product-specs/modules/AUDITEE_PORTAL.md` and `.turkce.md`.
- Modify `docs/product-specs/ux-plan/NAVIGATION_AND_INFORMATION_ARCHITECTURE.md` and `.turkce.md`.
- Modify `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md` and `.turkce.md`.
- Modify `docs/demo-evidence/BUILD_SUMMARY.md` and `.turkce.md` after verification.
- Modify `MANIFEST.md`, `docs/exec-plans/index.md`, and `docs/exec-plans/tech-debt-tracker.md` to keep package and plan truth synchronized.

---

## Task 1: Freeze The Lifecycle Contract With Failing Tests

**Files:**

- Create: `tests/inspection-lifecycle-alignment-smoke.test.js`
- Modify: `tests/department-preliminary-review-smoke.test.js`
- Modify: `tests/general-manager-workspace-smoke.test.js`
- Modify: `tests/executive-director-workspace-smoke.test.js`
- Modify: `tests/report-approval-smoke.test.js`
- Modify: `tests/service-provider-portal-smoke.test.js`
- Modify: `tests/manager-cap-monitoring-smoke.test.js`

**Interfaces:**

- Consumes: `freshState()`, `applyManagerReportDecision()`, `applyGeneralManagerReportDecision()`, `applyExecutivePreliminaryReportDecision()`, `applyExecutiveFinalReportDecision()`, `applyCapVerificationDecision()`, `serviceProviderVisibleReports()`, and planning/coordination helpers.
- Produces: a failing executable contract for every accepted lifecycle transition before runtime behavior changes.

- [x] **Step 1: Add the Preliminary Report state-machine assertions**

  Add an exact-ID test shaped as follows:

  ```js
  const state = context.freshState();
  const report = context.reportArtifactById('PR-2026-018', state);

  let result = context.applyManagerReportDecision(
    state,
    report.id,
    'approve',
    'Department review complete.',
    { role: 'manager', name: context.ROLES.manager.user }
  );
  assert.equal(result.ok, true);
  assert.equal(report.status, 'submitted_to_gm');
  assert.equal(report.ownerRole, 'gm');
  assert.equal(context.serviceProviderVisibleReports(state, report.organizationId, 'Preliminary Report').length, 0);

  result = context.applyGeneralManagerReportDecision(
    state,
    report.id,
    'approve',
    'GM review complete.',
    { role: 'gm', name: context.ROLES.gm.user }
  );
  assert.equal(result.ok, true);
  assert.equal(report.status, 'submitted_to_executive');
  assert.equal(report.ownerRole, 'executiveDirector');

  result = context.applyExecutivePreliminaryReportDecision(state, report.id, {
    decision: 'approve',
    actor: { role: 'executiveDirector', name: context.ROLES.executiveDirector.user },
    rationale: 'Approved for release to the Service Provider.'
  });
  assert.equal(result.ok, true);
  assert.equal(report.status, 'released_to_service_provider');
  assert.equal(report.ownerRole, 'auditee');
  assert.equal(report.locked, true);
  assert.equal(context.serviceProviderVisibleReports(state, report.organizationId, 'Preliminary Report').length, 1);
  ```

- [x] **Step 2: Add CAP and no-CAP branch assertions**

  Clone one Preliminary Report under a new exact ID, set `capRequired = false`, run the same three approvals, and assert:

  ```js
  assert.equal(noCapReport.status, 'released_to_service_provider');
  assert.equal(context.serviceProviderRequiredAction(noCapReport, []), 'View Report');
  assert.equal(context.serviceProviderRequiredAction(capReport, linkedOpenFindings), 'Respond to CAP and Evidence requests');
  ```

  The test must prove that `capRequired` changes the recipient action, not the approval chain.

- [x] **Step 3: Add role and return-path guards**

  Assert that:

  ```js
  assert.equal(context.applyGeneralManagerReportDecision(state, report.id, 'approve', '', {
    role: 'manager', name: 'Wrong role'
  }).ok, false);

  assert.equal(context.applyExecutivePreliminaryReportDecision(state, report.id, {
    decision: 'approve',
    actor: { role: 'gm', name: 'Wrong role' }
  }).ok, false);
  ```

  Also assert that a GM return moves the report to `pending_manager`, an ED return moves it to `submitted_to_gm`, and neither action publishes it to the Service Provider.

- [x] **Step 4: Add explicit CAP verification assertions**

  Create three independent `EVIDENCE_SUBMITTED` Finding states and assert:

  ```js
  const closeResult = context.applyCapVerificationDecision(closeState, findingId, {
    result: 'close',
    actor: { role: 'inspector', name: context.ROLES.inspector.user },
    commentToAuditee: 'Implementation verified; Finding closed.',
    internalNote: 'Verified against the latest submitted Evidence.'
  });
  assert.equal(closeResult.ok, true);
  assert.equal(closeResult.finding.status, 'CLOSED');

  const partialResult = context.applyCapVerificationDecision(partialState, findingId, {
    result: 'partially_close',
    actor: { role: 'leadInspector', name: context.ROLES.leadInspector.user },
    commentToAuditee: 'Partially verified; provide the remaining implementation Evidence.',
    internalNote: 'One corrective action remains unverified.'
  });
  assert.equal(partialResult.finding.status, 'EVIDENCE_MORE_INFO');
  assert.equal(partialResult.finding.capVerification.findingClosed, false);

  const notCloseResult = context.applyCapVerificationDecision(notCloseState, findingId, {
    result: 'not_close',
    actor: { role: 'inspector', name: context.ROLES.inspector.user },
    commentToAuditee: 'Verification was not sufficient; corrective action remains open.',
    internalNote: 'Submitted material does not demonstrate implementation.'
  });
  assert.equal(notCloseResult.finding.status, 'EVIDENCE_MORE_INFO');
  assert.equal(notCloseResult.finding.capVerification.findingClosed, false);
  ```

- [x] **Step 5: Keep George coordination and enforcement assertions in the lifecycle test**

  Reuse the public helpers to assert that a Routine inspection waits for provider confirmation, an Ad Hoc / Unannounced inspection withholds notice, and an enforcement referral has `recommendationOnly === true` without changing Findings or audit closure.

- [x] **Step 6: Run the new/changed tests and confirm the expected red state**

  Run:

  ```bash
  node --test tests/inspection-lifecycle-alignment-smoke.test.js tests/department-preliminary-review-smoke.test.js tests/general-manager-workspace-smoke.test.js tests/executive-director-workspace-smoke.test.js tests/report-approval-smoke.test.js tests/service-provider-portal-smoke.test.js tests/manager-cap-monitoring-smoke.test.js
  ```

  Expected before implementation: failures show that Department Manager currently releases CAP-required Preliminary Reports directly, GM rejects Preliminary Reports, `applyExecutivePreliminaryReportDecision` is missing, and `applyCapVerificationDecision` is missing.

## Task 2: Implement The Canonical Preliminary Report Approval State Machine

**Files:**

- Modify: `js/manager-workspaces.js`
- Modify: `js/reports.js`
- Modify: `js/data.js`

**Interfaces:**

- Consumes: report artifacts from `reportArtifactById()`, existing report history/audit-log/notification collections, and the current role identifiers.
- Produces: `applyManagerReportDecision()` with one Preliminary Report forward path, generalized `applyGeneralManagerReportDecision()`, `executivePreliminaryReportProjection()`, and `applyExecutivePreliminaryReportDecision()`.

- [x] **Step 1: Replace the Preliminary Report Department Manager branch**

  Replace the two Preliminary transitions with one transition:

  ```js
  var MANAGER_REPORT_TRANSITIONS = {
    'Preliminary Report': {
      approve: {
        status: 'submitted_to_gm',
        ownerRole: 'gm',
        action: 'Department Manager approved and forwarded Preliminary Report to General Manager'
      }
    },
    'Final Report': {
      approve: {
        status: 'submitted_to_gm',
        ownerRole: 'gm',
        action: 'Department Manager approved and forwarded Final Report to General Manager'
      }
    }
  };
  ```

  In `applyManagerReportDecision()`, select `.approve` for both report types. Keep `capRequired` unchanged; do not publish or notify the Service Provider here.

- [x] **Step 2: Generalize the General Manager transition without weakening role checks**

  Update `applyGeneralManagerReportDecision()` so both `Preliminary Report` and `Final Report` are eligible only when `status === 'submitted_to_gm'`, `ownerRole === 'gm'`, and `locked !== true`.

  On approve:

  ```js
  report.status = 'submitted_to_executive';
  report.ownerRole = 'executiveDirector';
  report.locked = false;
  report.generalManagerDecision = 'approve';
  ```

  On return:

  ```js
  report.status = 'pending_manager';
  report.ownerRole = 'manager';
  report.locked = false;
  ```

  Use the actual `report.reportType` in messages and history so Preliminary and Final records cannot be confused.

- [x] **Step 3: Add the Executive Director Preliminary Report projection**

  Add:

  ```js
  function executivePreliminaryReportProjection(target, filters) {
    var s = target || state;
    var opts = filters || {};
    var query = String(opts.query || '').trim().toLowerCase();
    var reports = reportReadModels(s).filter(function (report) {
      return report.reportType === 'Preliminary Report' &&
        ['submitted_to_executive', 'released_to_service_provider', 'rejected'].indexOf(report.status) !== -1;
    });
    var rows = reports.filter(function (report) {
      return !query || [report.id, report.auditId, report.organization, report.status].join(' ').toLowerCase().indexOf(query) !== -1;
    });
    return {
      rows: rows,
      pending: rows.filter(function (report) { return report.status === 'submitted_to_executive'; }).length,
      issued: rows.filter(function (report) { return report.status === 'released_to_service_provider'; }).length
    };
  }
  ```

  Preserve organization scoping in the report artifact; do not create a cross-organization release shortcut.

- [x] **Step 4: Add the Executive Director Preliminary Report decision primitive**

  Implement:

  ```js
  function applyExecutivePreliminaryReportDecision(target, reportId, input) {
    // Validate actor === executiveDirector, exact report type/ID, unlocked
    // submitted_to_executive stage, and approve/return/reject decision.
    // Approve releases one controlled copy to the report's organization.
  }
  ```

  Required approve state:

  ```js
  report.status = 'released_to_service_provider';
  report.ownerRole = 'auditee';
  report.locked = true;
  report.releasedAt = at;
  report.sharedAt = at;
  report.authorizedBy = actorName;
  report.authorizedAt = at;
  report.sharedBy = actorName;
  report.mockApprovalSignature = {
    label: 'DEMO mock approval mark - not a real e-signature',
    signer: actorName,
    date: at
  };
  ```

  Required return state is `submitted_to_gm` with owner `gm`. Required reject state is `rejected` with no Service Provider notification. Only approve creates an `auditee` notification with `organizationId: report.organizationId`.

- [x] **Step 5: Normalize selection state without rewriting historical releases**

  Add `selectedPreliminaryReportId`, `preliminaryReportQuery`, and `preliminaryReportStatus` to `executiveDirectorUi`. In `mergeDemoState()`, select an eligible pending Preliminary Report only when the saved selection is invalid and no unsaved decision exists. Leave already released/closed historical artifacts unchanged.

- [x] **Step 6: Run the state-machine tests**

  Run:

  ```bash
  node --test tests/inspection-lifecycle-alignment-smoke.test.js tests/report-approval-smoke.test.js tests/department-preliminary-review-smoke.test.js tests/general-manager-workspace-smoke.test.js tests/service-provider-portal-smoke.test.js
  ```

  Expected: all state-machine and scoping assertions pass; no Preliminary Report is visible to the Service Provider before Executive Director approval.

## Task 3: Expose Preliminary Report Reviews In Every Required Role Workspace

**Files:**

- Modify: `js/app.js`
- Modify: `js/views.js`
- Modify: `css/styles.css`
- Modify: `index.html`

**Interfaces:**

- Consumes: the report projections and decisions from Task 2.
- Produces: one Department Manager forward action, a mixed-type General Manager approval queue, a dedicated Executive Director Preliminary Reports route, and visible report-type-safe decision copy.

- [x] **Step 1: Simplify the Department Manager action**

  Replace the two-path approval menu with one primary action:

  ```html
  <button class="btn btn--primary" data-act="department-prelim-approve">
    Approve &amp; Forward to General Manager
  </button>
  ```

  Update `handleDepartmentPreliminaryApprove()` to call `applyManagerReportDecision()` for the exact selected report. Remove direct `auditee` notification/release logic from this handler. Keep `Request Changes` and exact-ID isolation.

- [x] **Step 2: Generalize the General Manager queue and dossier**

  Add `reportType` to each `generalManagerProjection().approvalRows` item, add separate `pendingPreliminaryReports` and `pendingFinalReports` counts, keep `reportsAwaitingApproval` as their total, and change headings/copy from `Final Report` to `Report` where the queue can contain both types. The selected dossier must still show the exact type:

  ```html
  <span>Selected Preliminary Report</span>
  ```

  or:

  ```html
  <span>Selected Final Report</span>
  ```

  The action remains `Forward to Executive Director`; GM must not issue, lock, or share either report.

- [x] **Step 3: Add the Executive Director Preliminary Reports route**

  Add:

  ```js
  { view: 'executive-preliminary-reports', label: 'Preliminary Reports', icon: '□' }
  ```

  to the Executive Director navigation, add the title/allowlist/render route, and implement `viewExecutivePreliminaryReportsWorkspace()` with:

  - pending/issued/returned summary counts;
  - exact Report ID, audit, organization, owner, Due Date, status, and CAP-required flag;
  - `Approve & Issue to Service Provider`, `Return to General Manager`, and `Reject Report` decisions;
  - no enforcement-referral action on Preliminary Reports;
  - a visible note that approval is a browser-local mock mark, not a real e-signature.

- [x] **Step 4: Wire Executive Director actions**

  Add action names scoped to Preliminary Reports:

  ```text
  executive-preliminary-select
  executive-preliminary-decision-choice
  executive-preliminary-confirm
  executive-preliminary-preview
  ```

  The confirm handler calls `applyExecutivePreliminaryReportDecision()`, persists state, clears only the Preliminary decision form, and leaves Final Report selection/decisions untouched.

- [x] **Step 5: Make the Service Provider next action CAP-aware**

  Update `serviceProviderRequiredAction()` so a released Preliminary Report with `capRequired === false` returns `View Report`, even if unrelated historical Findings exist on the same audit. A CAP-required report with linked open Findings continues to return `Respond to CAP and Evidence requests`.

- [x] **Step 6: Add responsive support only for the new controls**

  Reuse existing panel, table, badge, and stacked-row classes. Add CSS only where the new Executive Preliminary queue or three-action decision panel needs phone/tablet adaptation. At `390x844`, the document must satisfy:

  ```js
  document.documentElement.scrollWidth === document.documentElement.clientWidth
  ```

  and every visible decision control must have a practical touch area.

- [x] **Step 7: Bump the asset token**

  Change every CSS/JavaScript query string in `index.html` to one shared token:

  ```text
  20260718-lifecycle-alignment-v1
  ```

- [x] **Step 8: Run role-workspace tests**

  Run:

  ```bash
  node --test tests/department-preliminary-review-smoke.test.js tests/general-manager-workspace-smoke.test.js tests/executive-director-workspace-smoke.test.js tests/service-provider-portal-smoke.test.js
  ```

  Expected: Department Manager, General Manager, Executive Director, and Service Provider each see only their correct stage and permitted actions.

## Task 4: Implement Close, Partially Close, And Not Close CAP Verification

**Files:**

- Modify: `js/reports.js`
- Modify: `js/app.js`
- Modify: `js/views.js`
- Modify: `js/data.js`
- Modify: `css/styles.css`

**Interfaces:**

- Consumes: an exact Finding at `EVIDENCE_SUBMITTED`, its latest Evidence version, current CAA actor, and separate auditee/internal comments.
- Produces: `applyCapVerificationDecision(target, findingId, input)` returning `{ ok, message, finding }` and an audit-friendly `finding.capVerification` record.

- [x] **Step 1: Add the verification-result metadata**

  Add:

  ```js
  var CAP_VERIFICATION_RESULTS = {
    close: { label: 'Close', evidenceStatus: 'Accepted', findingStatus: 'CLOSED', findingClosed: true },
    partially_close: { label: 'Partially Close', evidenceStatus: 'Partially Accepted', findingStatus: 'EVIDENCE_MORE_INFO', findingClosed: false },
    not_close: { label: 'Not Close', evidenceStatus: 'More Information Requested', findingStatus: 'EVIDENCE_MORE_INFO', findingClosed: false }
  };
  ```

- [x] **Step 2: Implement the decision primitive**

  `applyCapVerificationDecision()` must reject:

  - actors other than `inspector` or `leadInspector`;
  - unknown result values;
  - Findings not at `EVIDENCE_SUBMITTED`;
  - Findings without a latest Evidence version;
  - blank `commentToAuditee` or blank `internalNote`.

  It must append, never replace, these records:

  ```js
  finding.capVerification = {
    result: input.result,
    label: meta.label,
    findingClosed: meta.findingClosed,
    actorRole: input.actor.role,
    actorName: input.actor.name,
    verifiedAt: at,
    evidenceId: latest.id,
    evidenceVersion: latest.version
  };
  finding.commentsToAuditee.push({
    author: input.actor.name,
    date: DEMO_TODAY,
    text: input.commentToAuditee
  });
  finding.internalNotes.push({
    author: input.actor.name,
    date: DEMO_TODAY,
    text: input.internalNote
  });
  ```

  For `close`, also set `closedDate` and `closureType = 'evidence-verified'`. For the other outcomes, clear neither prior Evidence versions nor CAP history and do not set a closed date.

- [x] **Step 3: Replace the binary Evidence review controls**

  Update `modalReviewEvidence()` to show three explicit controls:

  ```html
  <button data-decision="not_close">Not Close</button>
  <button data-decision="partially_close">Partially Close</button>
  <button data-decision="close">Close Finding</button>
  ```

  Add concise helper copy:

  - Close: all required implementation and Evidence verified; Finding closes.
  - Partially Close: some implementation verified; Finding remains open.
  - Not Close: verification insufficient; Finding remains open.

  Keep separate `Comment to Auditee` and `Internal CAA Note` fields and make both required for this decision record.

- [x] **Step 4: Route the existing Evidence action through the primitive**

  Replace the direct mutation in `evDecision()` with `applyCapVerificationDecision()`. Map only the new explicit result values; do not preserve the old ambiguous `accept`/`moreinfo` mutation path behind hidden controls.

- [x] **Step 5: Surface the latest verification result**

  Show the result label, actor, date, Evidence version, and next action in:

  - Finding Detail;
  - Inspector/Lead CAP review detail;
  - Department Manager CAP Monitoring drawer;
  - Service Provider CAP detail, excluding `Internal CAA Note`.

  `Partially Close` and `Not Close` must display `Finding remains open` and must not increment closed metrics.

- [x] **Step 6: Preserve authorized closure as a separate path**

  Keep `doAuthorizedClosure()` separate. Its existing reason requirement and audit-log behavior remain; it must not create a fake CAP verification record.

- [x] **Step 7: Run CAP/Finding regression tests**

  Run:

  ```bash
  node --test tests/inspection-lifecycle-alignment-smoke.test.js tests/manager-cap-monitoring-smoke.test.js tests/service-provider-portal-smoke.test.js tests/demo-boundary-smoke.test.js
  ```

  Expected: only `Close` closes a Finding; `Partially Close` and `Not Close` remain open, keep version history, and expose no internal note to the Service Provider.

## Task 5: Synchronize Planning, Lifecycle, Enforcement, And Demo-Boundary Copy

**Files:**

- Modify: `js/app.js`
- Modify: `js/views.js`
- Modify: `tests/demo-boundary-smoke.test.js`
- Modify: `tests/planning-release-smoke.test.js`
- Modify: `tests/planning-workspace-smoke.test.js`
- Modify: `tests/inspection-coordination-smoke.test.js`

**Interfaces:**

- Consumes: existing planning preparation state and George coordination state.
- Produces: consistent user-visible lifecycle summaries without changing already-correct planning or coordination transitions.

- [x] **Step 1: Correct all lifecycle summary strings**

  Replace abbreviated or contradictory chains with the accepted sequences. The canonical short strings are:

  ```text
  Planning: Department Manager -> Finance -> General Manager -> Executive Director -> GM Release to Department -> Department preparation.
  Preliminary Report: Lead Inspector -> Department Manager -> General Manager -> Executive Director -> Service Provider issue.
  Final Report: Lead Inspector -> Department Manager -> General Manager -> Executive Director -> Service Provider issue.
  ```

- [x] **Step 2: Keep GM Release visible and unchanged**

  Assert that Executive Director planning approval still results in:

  ```js
  plan.preparation.status === 'not_released'
  planningWorkspaceNextAction(plan).label === 'GM Release to Department'
  ```

  Do not merge this release into Executive Director approval.

- [x] **Step 3: Keep George coordination visible and unchanged**

  Ensure lifecycle/help copy states:

  ```text
  Routine: notify the Service Provider after Lead Inspector assignment; share the proposed date, checklist, and relevant information; wait for confirmation or an accepted alternative date.
  Ad Hoc / Unannounced: withhold advance notification and coordination.
  ```

- [x] **Step 4: Replace production-strength footer claims**

  Use these exact boundaries wherever the lifecycle/footer is summarized:

  ```text
  Browser-local mock approvals with demo timestamps.
  Demo audit history for traceability; not a production audit trail.
  Mock filenames and local browser state; no secure document storage.
  ```

  Search for contradictory claims:

  ```bash
  rg -n "all approvals are electronic|securely stored|secure storage|production audit trail|automatic penalty|automatic suspension" index.html js css docs
  ```

  Expected after correction: no unqualified production claim remains.

- [x] **Step 5: Preserve enforcement as referral-only**

  Keep the Executive Director Final Report action label `Refer for Enforcement Review` and the stored contract:

  ```js
  report.enforcementReferral.recommendationOnly === true
  report.enforcementReferral.status === 'pending_authorized_review'
  ```

  No Warning, Corrective Directive, Administrative Penalty, Suspension, or Follow-up Inspection control may directly mutate a sanction/certificate state.

- [x] **Step 6: Run the planning, coordination, and boundary tests**

  Run:

  ```bash
  node --test tests/planning-workspace-smoke.test.js tests/planning-release-smoke.test.js tests/inspection-coordination-smoke.test.js tests/executive-director-workspace-smoke.test.js tests/demo-boundary-smoke.test.js
  ```

  Expected: the existing planning release and George coordination flows pass unchanged, and lifecycle copy no longer overclaims production behavior.

## Task 6: Update Canonical English And Turkish Lifecycle Documentation

**Files:**

- Modify: `docs/product-specs/workflows/MASTER_WORKFLOW.md`
- Modify: `docs/product-specs/workflows/MASTER_WORKFLOW.turkce.md`
- Modify: `docs/product-specs/workflows/SURVEILLANCE_PLANNING_WORKFLOW.md`
- Modify: `docs/product-specs/workflows/SURVEILLANCE_PLANNING_WORKFLOW.turkce.md`
- Modify: `docs/product-specs/workflows/FINDING_CAP_EVIDENCE_WORKFLOW.md`
- Modify: `docs/product-specs/workflows/FINDING_CAP_EVIDENCE_WORKFLOW.turkce.md`
- Modify: `docs/product-specs/modules/AUDIT_PLANNING.md`
- Modify: `docs/product-specs/modules/AUDIT_PLANNING.turkce.md`
- Modify: `docs/product-specs/modules/AUDITEE_PORTAL.md`
- Modify: `docs/product-specs/modules/AUDITEE_PORTAL.turkce.md`
- Modify: `docs/product-specs/ux-plan/NAVIGATION_AND_INFORMATION_ARCHITECTURE.md`
- Modify: `docs/product-specs/ux-plan/NAVIGATION_AND_INFORMATION_ARCHITECTURE.turkce.md`
- Modify: `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md`
- Modify: `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.turkce.md`

**Interfaces:**

- Consumes: the verified runtime transitions from Tasks 2-5.
- Produces: one bilingual lifecycle source of truth with a Mermaid diagram and explicit demo boundaries.

- [x] **Step 1: Add the canonical lifecycle diagram to `MASTER_WORKFLOW.md`**

  Use a Mermaid flowchart that includes these distinct nodes:

  ```mermaid
  flowchart TD
    A["Department Manager planning"] --> B["Finance review"]
    B --> C["General Manager approval"]
    C --> D["Executive Director approval"]
    D --> E["GM Release to Department"]
    E --> F["Lead Inspector and team assignment"]
    F --> G{"Inspection notice policy"}
    G -->|"Routine"| H["Service Provider date and checklist coordination"]
    G -->|"Ad Hoc / Unannounced"| I["Advance notice withheld"]
    H --> J["Inspection execution"]
    I --> J
    J --> K["Preliminary Report: Lead -> DM -> GM -> ED"]
    K --> L["Issue to Service Provider"]
    L --> M["CAP and Evidence response when required"]
    M --> N{"CAP verification result"}
    N -->|"Close"| O["Finding closed"]
    N -->|"Partially Close"| P["Finding remains open"]
    N -->|"Not Close"| P
    O --> Q["Final Report: Lead -> DM -> GM -> ED"]
    P --> M
    Q --> R["Monitoring and referral-only enforcement review"]
  ```

  Add the same semantic diagram/copy to the Turkish companion using stakeholder-ready Turkish labels.

- [x] **Step 2: Update workflow and module rules**

  State explicitly that:

  - Preliminary approval never branches around GM/ED based on CAP requirement;
  - Service Provider visibility starts only after ED release;
  - CAP result affects post-release work;
  - `Partially Close` and `Not Close` leave the Finding open;
  - authorized closure remains separate;
  - planning ED approval is followed by GM Release to Department.

- [x] **Step 3: Update permission and navigation documentation**

  Record:

  - GM may return/forward Preliminary and Final Reports but cannot issue or lock them;
  - ED may issue Preliminary and Final Reports with demo-only mock marks;
  - Service Provider sees only its organization's ED-released Preliminary Reports and issued Final Reports;
  - only Inspector/Lead Inspector records CAP verification outcomes in this demo;
  - internal notes never appear in Service Provider views.

- [x] **Step 4: Run documentation checks**

  Run:

  ```bash
  node tests/harness-docs-smoke.test.js
  rg -n "Preliminary Report|Partially Close|Not Close|GM Release to Department|Routine|Ad Hoc" docs/product-specs
  git diff --check
  ```

  Expected: the docs smoke test passes; every accepted lifecycle rule appears in canonical English and Turkish surfaces; no whitespace errors are reported.

## Task 7: Full Verification, Rendered QA, Evidence, And Plan Tracking

**Files:**

- Modify: `docs/demo-evidence/BUILD_SUMMARY.md`
- Modify: `docs/demo-evidence/BUILD_SUMMARY.turkce.md`
- Modify: `MANIFEST.md`
- Modify: `docs/exec-plans/index.md`
- Modify: `docs/exec-plans/tech-debt-tracker.md`
- Update: `docs/exec-plans/active/2026-07-18-inspection-lifecycle-alignment-plan.md`

**Interfaces:**

- Consumes: all implementation, tests, rendered screenshots, console results, and demo-boundary scans.
- Produces: literal local verification evidence and synchronized active-plan state.

- [x] **Step 1: Run JavaScript syntax checks**

  Run:

  ```bash
  node --check js/data.js
  node --check js/reports.js
  node --check js/manager-workspaces.js
  node --check js/views.js
  node --check js/app.js
  ```

  Expected: all commands exit `0` with no syntax output.

- [x] **Step 2: Run the focused lifecycle suite**

  Run:

  ```bash
  node --test tests/planning-workspace-smoke.test.js tests/planning-release-smoke.test.js tests/inspection-coordination-smoke.test.js tests/inspection-team-smoke.test.js tests/inspection-execution-smoke.test.js tests/department-preliminary-review-smoke.test.js tests/general-manager-workspace-smoke.test.js tests/executive-director-workspace-smoke.test.js tests/report-approval-smoke.test.js tests/service-provider-portal-smoke.test.js tests/manager-cap-monitoring-smoke.test.js tests/inspection-lifecycle-alignment-smoke.test.js tests/demo-boundary-smoke.test.js
  ```

  Expected: all focused tests pass with zero failures.

- [x] **Step 3: Run the full local test suite**

  Run:

  ```bash
  node --test tests/*.test.js
  ```

  Expected: every repository test passes; the final count is recorded literally in the build summary.

- [x] **Step 4: Run rendered lifecycle QA in an isolated browser profile**

  Serve locally:

  ```bash
  python3 -m http.server 4173 --bind 127.0.0.1
  ```

  Replay at `1440x900`, `1024x768`, and `390x844`:

  1. Department Manager approves Preliminary Report -> only GM queue changes.
  2. General Manager forwards -> only ED Preliminary queue changes.
  3. Service Provider cannot see the report yet.
  4. Executive Director approves -> Service Provider sees the exact report and CAP-aware next action.
  5. `Close`, `Partially Close`, and `Not Close` are each recorded in separate reset states.
  6. Only `Close` changes the Finding to closed.
  7. Planning ED approval still leads to GM Release to Department.
  8. Routine and Ad Hoc coordination still follow George's rule.
  9. Enforcement remains referral-only.

  Capture screenshots and assert:

  - no page-level horizontal overflow;
  - no clipped decision controls;
  - no console errors/warnings;
  - correct role and exact Report/Finding ID on every screen;
  - no internal CAA note in Service Provider HTML.

- [x] **Step 5: Clean browser automation processes**

  Check:

  ```bash
  ps -axo pid,ppid,stat,command | egrep "playwright|puppeteer|webdriver|HeadlessChrome|Google Chrome.*remote-debugging"
  ```

  Expected: no lifecycle-QA process remains after the server/browser session is stopped.

- [x] **Step 6: Update evidence and package truth**

  Record only literal evidence in both build summaries:

  - exact focused/full test counts;
  - viewport set and screenshot path;
  - console/overflow outcome;
  - George coordination status;
  - demo-only approval, history, storage, and enforcement boundaries;
  - production behavior not verified.

  Add the new test and plan to `MANIFEST.md`.

- [x] **Step 7: Synchronize the plan lifecycle**

  Update this plan's checkboxes and `docs/exec-plans/index.md` next action to match reality. If all implementation and verification steps pass, set the index status to `ready-for-verification` with stakeholder sign-off as the only next action. Keep the plan under `active/` until stakeholder sign-off; do not move it to `completed/` automatically.

  Close the matching lifecycle alignment tracker note only when the focused/full suites and rendered matrix are recorded. Keep production signing/enforcement and secure-storage notes open.

- [x] **Step 8: Final integrity checks**

  Run:

  ```bash
  git diff --check
  git status --short
  ```

  Expected: no whitespace errors; only intentional lifecycle/previous user changes are present. Do not stage, commit, push, or change branches without explicit user authorization.

---

## Verification Matrix

| Requirement | Automated Evidence | Rendered Evidence |
|---|---|---|
| Planning keeps GM Release after ED approval | `planning-release-smoke`, `executive-director-workspace-smoke` | ED Planning -> GM release next action |
| George Routine coordination | `inspection-coordination-smoke` | Lead Assignment and Service Provider Coordination |
| George Ad Hoc no-notice rule | `inspection-coordination-smoke` | Ad Hoc assignment state |
| Preliminary Lead -> DM -> GM -> ED | lifecycle, department, GM, ED, report tests | Four-role replay |
| No pre-ED Service Provider visibility | lifecycle and portal tests | Service Provider before/after replay |
| CAP-required flag affects next action only | lifecycle and portal tests | Released Preliminary dossier |
| Close/Partial/Not Close are explicit | lifecycle and CAP monitoring tests | Evidence review modal and status |
| Partial/Not Close stay open | lifecycle and CAP monitoring tests | Finding Detail and manager metrics |
| Final Report chain remains intact | report, GM, ED tests | Final Report role replay |
| Enforcement remains referral-only | ED and boundary tests | ED Final Report decision panel |
| No production security/signing claim | boundary and docs tests | Lifecycle/footer/portal copy |
| Service Provider privacy | portal and lifecycle tests | Organization-scoped portal HTML |

## Risks And Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Existing tests encode direct DM -> Service Provider release | False regressions or accidental bypass retention | Change tests first and require the new red state before runtime edits. |
| General Manager projection is Final-only | Preliminary Reports could enter state without a usable UI | Generalize the projection/queue and assert exact type in every row/dossier. |
| Executive Director Final Report UI includes enforcement actions | Reusing it blindly could expose enforcement on Preliminary Reports | Add a separate Preliminary route/decision panel with approve/return/reject only. |
| Historical browser state contains already-released Preliminary Reports | Automatic migration could corrupt stakeholder demo history | Preserve completed releases; apply the chain to new/pending decisions and recommend Reset Demo Data for a clean replay. |
| `Partially Close` is mistaken for actual Finding closure | Dashboards or stakeholders may treat the Finding as closed | Keep technical status `EVIDENCE_MORE_INFO`, store `findingClosed: false`, and display `Finding remains open`. |
| Evidence acceptance currently closes directly | New three-way decision could leave duplicate mutation paths | Route the UI through one `applyCapVerificationDecision()` primitive and remove hidden binary mutations. |
| Role notifications leak across organizations | Service Provider privacy breach in demo | Scope release notifications by `organizationId` and test a second organization. |
| Lifecycle footer repeats production-strength image claims | Misleading stakeholder statement | Replace with exact browser-local/demo wording and scan prohibited phrases. |
| New ED route increases visual audit surface | Mobile clipping or route coverage gap | Add the route to rendered QA and assert document width at all three viewports. |
| Large dirty working tree overlaps lifecycle files | Accidental loss of George or UI-audit changes | Inspect diffs before each task, patch narrowly, and never reset/revert unrelated work. |

## Completion Criteria

This plan is implementation-complete only when:

- George's Routine/Ad Hoc coordination tests remain passing.
- Department Manager cannot directly publish a Preliminary Report.
- General Manager can review both Preliminary and Final Reports.
- Executive Director has a separate Preliminary Report decision surface.
- Service Provider sees a Preliminary Report only after ED approval and only for its own organization.
- CAP-required and no-CAP-required Preliminary Reports follow the same approval chain.
- `Close`, `Partially Close`, and `Not Close` are explicit, audit-friendly demo decisions.
- Only `Close` sets Finding status to `CLOSED`.
- Planning still includes GM Release to Department after ED approval.
- Final Report and enforcement-referral behavior remains passing and non-automatic.
- Canonical English/Turkish lifecycle docs match runtime behavior.
- Focused and full tests pass, rendered QA is clean, and evidence/index/tracker records are synchronized.
- Stakeholder review remains the next action; production readiness is not claimed.

## Execution Prompt

```text
Execute docs/exec-plans/active/2026-07-18-inspection-lifecycle-alignment-plan.md in its defined order in the current AviaSurveil360 working tree. Preserve all existing user changes and George's completed Routine/Ad Hoc Service Provider coordination behavior. Begin with focused lifecycle tests, then implement the full Preliminary Report chain (Lead Inspector -> Department Manager -> General Manager -> Executive Director -> Service Provider), explicit Close / Partially Close / Not Close CAP verification, planning GM Release visibility, referral-only enforcement boundaries, bilingual docs, and literal local verification evidence. Keep the project frontend-only/demo-only. Use apply_patch for edits. Do not create or switch branches, commit, push, publish, or replace the stakeholder's source PNG unless the user explicitly authorizes that action. Stop and report if an implementation choice would change the accepted lifecycle or expand into production security, signing, storage, notifications, or enforcement execution.
```
