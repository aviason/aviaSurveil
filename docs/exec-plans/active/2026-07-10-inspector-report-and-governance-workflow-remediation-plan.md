# Inspector, Report, Service Provider, and Governance Workflow Remediation Implementation Plan


**Goal:** Correct the Inspector and Lead Inspector assignment/report paths, replace the stale SMS execution surface with the Cabin Inspection scenario, rebuild the Service Provider's CAP/Preliminary/Final Report views, and add focused Finance Review and Executive Director workspaces for planning and Final Report decisions.

**Architecture:** Keep the current static HTML/CSS/Vanilla JavaScript application and browser-local mock state. Add role-specific projections and renderers over the existing planning, audit, team, Finding, and report records; do not create parallel lifecycle engines. Normalize interactive state by `auditId`, `reportId`, `planId`, and `questionId` so every action reopens the same artifact and survives rerendering.

**Tech Stack:** HTML, CSS, Vanilla JavaScript, browser-local mock state, direct Node smoke tests, and local rendered browser QA.

## Global Constraints

- Keep AviaSurveil360 frontend-only and demo-only: no backend, database, API, real authentication, real authorization enforcement, real upload/storage, real notification delivery, real regulatory ingestion, real AI service, production audit log, framework migration, or deployment.
- Preserve the canonical `Finding -> CAP -> Evidence -> CAA Review -> Closure` rule. CAP acceptance is not closure, and report approval must not close open Findings.
- Keep all signatures visibly labelled as mock demo approval marks. Do not claim or simulate a legally valid e-signature.
- Enforcement choices are configured demo referrals/recommendations only. They must not automatically impose a fee, suspension, revocation, certificate action, audit closure, or Finding closure.
- Preserve the planning chain `Department Manager -> GM -> Finance Review -> Executive Director`; ED approval continues to explicit `GM Release to Department`, not directly to execution.
- Use the canonical Cabin Inspection/Fly Namibia data already in the repository. Screenshot organizations and IDs are layout references, not new source facts.
- Keep visible controls functional. A dropdown must open as a dropdown; a row action must open the selected record; no toast-only substitute for a promised screen transition.
- Preserve unrelated dirty-worktree changes. Do not create/switch branches, commit, push, deploy, open a PR, or post GitHub comments unless the user separately requests that exact action.
- Repository implementation docs remain English; update existing Turkish companion docs whenever the matching canonical stakeholder/product doc changes.

---

**Date:** 2026-07-10
**Status:** ready-for-verification
**Owner:** Codex implementation owner; stakeholder/product owner approves the final screen direction and policy wording.
**Verification boundary:** The independent review blockers are remediated and the corrective [Stakeholder Readiness Final Remediation](2026-07-10-stakeholder-readiness-final-remediation-plan.md) returned a local demo GO after focused/full automated checks and fresh four-viewport rendered QA. Stakeholder acceptance remains the next gate.

**2026-07-10 final remediation checkpoint:** The follow-up plan's Tasks 1-9 are implemented and **verified locally** for the frontend-only, **demo-only** surface. Canonical report identity, authority, organization privacy, exact IDs, assignment/execution integration, visible controls, responsive containment, selected PDF content, and keyboard/modal behavior passed focused/full checks and fresh rendered QA at `1536x864`, `1366x768`, `1024x768`, and `390x844`. Production signing/enforcement authority and all production/external surfaces remain **not run**; **production-readiness not claimed**.

## Objective

Deliver one coherent demo path:

1. Lead Inspector selects real audit-team members, adds another Inspector, and assigns separate question batches to different people.
2. A Cabin Safety Inspector opens the Cabin Inspection, sees the six workbook-derived Cabin sections, submits once, and remains in the Inspector workspace with a useful success/status modal.
3. Lead Inspector opens a Preliminary Report by its existing Report ID and lands on the report-linked `Inspection & Findings` workflow, including visible Findings Review.
4. Service Provider sees organization-scoped Corrective Actions (CAP), Preliminary Reports, and Final Reports screens with clear Level, Status, Due Date, required action, and working response/view flows.
5. Finance Review sees one restrained budget-review queue and can approve the budget or return it for revision without replacing GM/ED authority.
6. Executive Director sees a compact Dashboard, Planning, and Final Reports experience; can review a plan, `Approve & Sign (Demo)` or reject it; and can review, preview, approve, return, reject, or refer a Final Report for enforcement review.
7. Full-report preview and browser-generated demo output use the selected report's real state rather than hard-coded SkyCargo/JetFast content.

## Stakeholder Feedback Traceability

| Feedback | Planned product contract | Acceptance evidence |
|---|---|---|
| 1. Other Inspectors must be selectable; Add Inspector must work. | Persist team membership and question assignment per audit/question/Inspector, prevent duplicates, and compute workload/counts from state. | Assign one batch to Aylin and another to Mehmet; both mappings survive rerender/reset-from-saved-state. Add a third Inspector and assign a question to that member. |
| 2. Inspector submit must not route to Lead Inspector. | First click opens an Inspector-context success modal; role and audit route stay unchanged. | Modal appears immediately with stable timestamp, status, timeline, `View Submitted Checklist`, and `Return to My Assignments`. |
| 4. Checklist sections must match Cabin Safety. | Replace the global SMS sections for `AUD-2026-001` with the six workbook-derived Cabin sections. | The six sections render in the documented order and `EM EQ / PBE` is reachable. No SMS title or SMS section label appears for the Cabin audit. |
| 5. Screens look too basic. | Use the existing premium dossier/command/approval patterns with stronger selected states, hierarchy, and responsive decision panels. | Desktop/mobile visual review shows a clear selected record, primary action, approval path, and no generic card wall or clipped action. |
| 6. Preliminary Reports must not start from an empty report. | Remove free-form `New Preliminary Report`; replace it with `Open Report Package` / `Continue Existing Report`, resolved by Report ID and Audit ID. | Every interactive report opens an existing linked package; no orphan shell is created. |
| 8. Findings Review is missing in Preliminary Reports. | Make `Inspection & Findings` the default report step and keep the Findings panel fully visible. | Report ID, row action, and package chooser all open the same workflow with severity/status, selection, and View actions. |
| 9. ED experience should be simple. | ED sidebar has Dashboard, Planning, Final Reports, then utility entries Notifications and Settings. Remove duplicate Audit Reports/Reports modules. | Navigation and route allowlist match the contract; utility entries open working demo pages. |
| 10. Final Reports is an ED approval screen. | Add a Final-Report-only queue with status cards, filters, report dossier, and Review action. | Only Final Reports owned by or previously decided by ED appear in the operational queue. |
| 12. Preview/Review should open the detailed decision screen. | Review opens a selected-report dossier with summary/findings/documents/history tabs and a decision panel. | Selected Report ID remains stable from list to review and back. |
| 13. Enforcement actions need a dropdown. | Reveal a real configured-action dropdown only after `Refer for Enforcement Review` is selected; require rationale. | Dropdown opens normally, selection persists, and no sanction or closure side effect occurs. |
| 14. ED needs a Dashboard. | Add six decision-oriented KPIs, pending plan/report queues, department/risk context, and overdue actions below the first decision viewport. | Metrics derive from state and link to filtered Planning or Final Reports views. |
| 15. `Preview Full Report` needs a full preview page. | Add a selected-report viewer with contents navigation, zoom, print/download demo controls, and return-to-review. | Viewer is state-backed, responsive, and does not claim ungenerated pages. |
| 16. Use the supplied Final Report template direction. | Render an A4-style first-page template from the selected report/audit/findings/team/signature state. | Fly Namibia, correct IDs, counts, conclusion, next steps, and mock signature match the selected record. |
| 17. ED Planning should use the supplied queue/detail direction. | Add status cards, filters, plan queue, selected dossier, approval flow, and a single row-level action trigger. | `Review / Take Action` opens the exact plan; no separate inert eye/kebab controls remain. |
| Added: one Planning command should expose approval/signing. | The single action trigger offers Preview and `Approve & Sign (Demo)` only when the record is at the ED stage; decision opens the detail panel before confirmation. | No one-click irreversible state change from the menu; confirmation writes actor, time, history, and mock signature. |
| Added: ED reviews planned audits and approves or rejects. | Detail panel shows Overview, Plan Information, Departments & Scope, Budget & Resources, Approval History, and Documents & Notes; Reject requires a reason. | Approve produces `approved` plus mock signature and keeps GM release next; Reject prevents release/execution. |
| Added: Finance Review should only approve/reject budgets; no Dashboard is needed. | Finance has one `Finance Review` workspace with a budget queue, selected budget dossier, approval history, and only `Approve Budget` / `Return for Revision`. | Finance lands directly on the review workspace. Return requires a reason and returns to GM action; approval advances to ED and cannot sign/issue the plan. |
| Added: Service Provider needs a CAP workspace with Level, Status, and Due Date. | Replace duplicate request-center/CAP navigation with a state-backed CAP queue, canonical status tabs/filters, progress, and a selected Finding dossier/timeline. | Only Fly Namibia records render; Level, current Status, configured Due Date, progress, next action, and Respond are visible and functional. |
| Added: Service Provider needs separate Preliminary Reports and Final Reports screens. | Add organization-scoped report lists/details. Preliminary Reports show Total/Pending Your Response/Under Review/Closed and the configured response Due Date; Final Reports show only ED-issued packages. | Each row opens its own Report ID; internal CAA notes/enforcement deliberations never render. |
| Added: Service Provider must be able to view the Final Report. | `View Report` opens the shared state-backed report viewer with auditee-safe content and mock download actions. | The selected Fly Namibia report ID, metadata, findings, and allowed attachments remain consistent through list/detail/preview. |
| Added: Service Provider Final Reports should use the latest simplified table. | The table uses only Report ID, Audit/Inspection, Date Released, Findings, and Action. CAP requirement is handled by the KPI/filter and separate CAP workspace. | Neither `Status` nor `Required Action` table headers/cells render. |

Feedback numbers 3, 7, and 11 were not supplied; this plan does not infer hidden requirements for them.

## Chosen Approach

### Recommended: role-specific composition over shared state

Build dedicated Service Provider, Finance, and ED renderers, but consume the existing `planningItems`, `inspectionTeams`, `users`, `audits`, `findings`, `auditReports`, and `managerReports` boundaries. Add exact ID links where the current read models are duplicated. This gives the requested screen direction without creating a second CAP, planning, or report lifecycle.

### Rejected: patch each current screen in place

This is faster initially but leaves the existing root causes: global assignee state, hard-coded SMS content, Report ID/kebab route divergence, and GM acting on ED-owned reports.

### Deferred: replace all report models with a new report subsystem

A full `auditReports`/`managerReports` consolidation would be cleaner, but it is larger than the stakeholder delta and risks unrelated manager/report regressions. This plan instead establishes one canonical interactive package per report and explicit read-model links; a broader data-model rewrite can be planned later if still needed.

## Current Root Causes To Remove

- `state.leadAssignmentUi.assignee` is global. It does not preserve different Inspectors for different question batches.
- `addedInspectors` is UI-only and is not the canonical `inspectionTeams.memberIds` roster.
- Lead assignment metrics (`42`, `84`, and workload values) are hard-coded.
- `viewInspectorAuditExecution()` renders a hard-coded SMS audit and eight SMS sections even when `AUD-2026-001` is the Cabin Inspection.
- Inspector submission shows only a toast on first click; the second-click modal links to the Lead-only assignment route.
- `handlePreliminaryReportOpen()` routes a Report ID to the generic approval page, while the row action opens the rich report workflow. This is why Findings Review disappears.
- `leadPreliminaryReportRows()` and `leadPreliminaryWorkflowFindings()` use global/hard-coded report content rather than report-linked state.
- The current Auditee navigation duplicates Received Reports/My CAPs/CAP Management, while the CAP request center and Final Report detail are separate hard-coded projections.
- `serviceProviderFinalReportMeta()` and `serviceProviderCapRequirements()` are fixed to one AVSEC/FR record instead of selecting organization-scoped state.
- Current Service Provider due rules are hard-coded by severity. Due Date must come from the configured Finding/report record, not an invented legal deadline.
- ED has no home dashboard, no dedicated plan/report queue, and duplicate generic report navigation.
- `applyGeneralManagerReportDecision()` currently issues and locks a report whose owner is `executiveDirector`; GM final authority must be removed.
- Finance uses the shared dense planning panel and exposes more decision variants than the requested simple budget review.
- Report preview/export helpers are tied to hard-coded inspection/operator values instead of the selected report.

## Scope

### In scope

- Multi-Inspector selection, Add Inspector, duplicate prevention, question-level assignment, workload calculation, release notifications, and persisted browser-local state.
- Audit-aware Cabin Inspection execution content, progress, finding linkage, download copy, submission status, and read-only submitted view.
- Existing-report-only Preliminary Report entry, canonical workflow routing, per-report draft UI state, and visible Findings Review.
- Service Provider CAP queue/detail/respond flow plus separate Preliminary Reports and Final Reports list/detail/preview routes, all restricted to the current organization.
- One Finance Review/Planning workspace with a budget queue/dossier, approval history, and simple decisions.
- ED Dashboard, Planning queue/detail/decisions, Final Reports queue/review/decisions, notification utility, and settings utility.
- Mock plan/report approval signatures; state-backed report preview and browser-generated demo PDF/text artifact.
- Enforcement referral dropdown with configured demo categories and rationale.
- Premium/responsive/accessibility pass on changed surfaces.
- Focused tests, browser QA, bilingual product/evidence updates, plan-index synchronization, and durable boundary tracking.

### Explicitly out of scope

- Real identity verification, sign-in flow, digital certificate, e-signature, signing provider, or signature validity claim.
- Real budgeting, finance integration, accounting ledger, currency conversion, procurement, or budget authorization.
- Real enforcement case management or execution of a penalty, suspension, revocation, conditional approval, certificate action, or legal notice.
- A real 56-page report engine. The viewer may show the actual generated demo page count and a contents outline; it must not claim pages that do not exist.
- Free-form report creation, backend document storage, real uploads, real email/notification delivery, or immutable audit evidence.
- Rewriting unrelated manager, auditee, admin, CAP, evidence, AI, offline, regulatory, or SSP/NASP modules.
- Broad Modern Aviation SaaS rollout outside the screens owned by this plan.

## Assumptions And Product Decisions

- The phrase `approve and sign in` is interpreted as `Approve & Sign (Demo)`, not authentication.
- Reference screenshots define hierarchy and interaction intent. Canonical demo content remains Cabin Inspection / Fly Namibia.
- Service Provider operational navigation is `Corrective Actions (CAP)`, `Preliminary Reports`, and `Final Reports`; existing Messages, Documents, and Settings remain utilities. Duplicate `Received Reports`, `My CAPs`, and `CAP Management` entries are removed. No new Service Provider Dashboard is added in this plan.
- Service Provider CAP rows show the canonical configured Finding status and Due Date. Severity never auto-calculates a legal deadline.
- Service Provider Final Reports intentionally omit both `Status` and `Required Action` table columns. CAP work belongs in the separate Corrective Actions screen; `Reports Requiring CAP` may remain a derived KPI/filter.
- Preliminary Reports become visible only after the configured release-to-Service-Provider stage. Final Reports become visible only after ED issue/lock.
- ED has three operational modules (Dashboard, Planning, Final Reports) and two working utility entries (Notifications, Settings).
- Finance has one operational workspace: `Finance Review`. It lands directly on the dedicated `finance-review` route and does not need a Dashboard or additional modules. The legacy Finance `planning` link redirects to this route. Finance does not create plans, edit audit scope, sign plans, release audits, issue reports, or close records.
- Canonical planning order remains Department Manager, GM, Finance Review, Executive Director even where a reference image visually orders stages differently.
- `Finance Not Approved` is presented as `Return for Revision` and returns the item to GM action, matching the existing governance rule.
- `Approve with Adjustment` remains supported by the underlying primitive for compatibility but is not a primary Finance UI decision in this simplified surface.
- ED is the only final report issue/sign authority. GM remains an intermediate reviewer when configured and may not issue, sign, or lock a Final Report.
- ED plan approval does not release the audit. The next owner/action becomes `GM / Release to Department`.
- Final Report approval does not close Findings. If open CAP/evidence work exists, the audit becomes `Report Issued` or `Follow-up Open`; it becomes `Closed` only when the configured audit/finding closure rules are satisfied.

## State And Interface Contracts

### Audit-scoped assignment state

Use canonical user IDs and audit IDs rather than display names:

```js
state.leadAssignmentsByAudit[auditId] = {
  activeInspectorUserId: 'USR-AYLIN',
  selectedQuestionIds: { 'CAB-Q001': true },
  assignmentsByQuestionId: {
    'CAB-Q001': {
      inspectorUserId: 'USR-AYLIN',
      dueDate: '2026-06-15',
      priority: 'Normal',
      instructions: '',
      assignedAt: '2026-07-10T11:24:00.000Z'
    }
  },
  draftSavedAt: '',
  releasedAt: ''
};
```

Team membership comes from `state.inspectionTeams[*].memberIds`. `Add Inspector` appends a valid active internal Inspector user ID only when it is not already present.

### Audit-scoped execution/submission state

```js
state.inspectionWorkspaces[auditId] = {
  selectedSectionKey: 'em-eq',
  answersByQuestionId: {},
  downloadedAt: '',
  draftSavedAt: '',
  allSectionsCompletedAt: '',
  submittedAt: '',
  submittedByUserId: 'USR-AYLIN'
};
```

The execution package resolver consumes the opened audit and the published checklist/template. It returns title, organization, dates, six section definitions, question rows, progress, and finding context.

### Report identity and draft state

- Every interactive queue row must carry `reportId`, `auditId`, and `reportType`.
- Every portal-visible report package must also carry `organizationId` plus the lifecycle timestamps it actually uses: `sharedAt`, `sharedBy`, and `responseDueDate` for Preliminary Reports; `releasedAt`/`issuedAt`, `issued`, `locked`, `finalAuthorizedBy`, and `finalAuthorizedAt` for Final Reports.
- The primary report read model must link to its canonical approval package with `approvalPackageId`; do not add a third mutable report collection.
- Treat `managerReports` as a list/read projection only. Each row must link by exact ID to the richer canonical `auditReports` package; neither collection may become a second independently mutable report lifecycle.
- Add exact helpers rather than first-match-by-audit lookups:

```js
function reportForAuditAndType(auditId, reportType) {}
function reportArtifactById(reportId) {}
function openLeadPreliminaryReport(reportId) {}
```

- Store editable Lead report state per Report ID:

```js
state.preliminaryReportDrafts[reportId] = {
  step: 'inspection',
  content: '',
  includedFindingIds: {},
  findingLevel: 'all',
  findingQuery: '',
  mockAttachmentNames: [],
  declarations: { accurate: true, evidenceBased: true, readyForReview: true },
  draftSavedAt: '',
  submittedAt: ''
};
```

### Service Provider portal state and visibility

Do not create hard-coded CAP/report arrays for the portal. Project the current organization's visible records from canonical Findings and report packages:

```js
state.serviceProviderUi = {
  cap: {
    group: 'all', auditId: 'all', level: 'all', status: 'all', query: '',
    selectedFindingId: 'CAB-2026-001'
  },
  preliminaryReports: {
    auditId: 'all', status: 'all', query: '', selectedReportId: 'PR-2026-018'
  },
  finalReports: {
    auditId: 'all', year: 'all', capRequirement: 'all', query: '', selectedReportId: 'FR-2026-018'
  },
  reportPreview: { reportId: '', zoom: 100 }
};
```

Required pure selectors:

```js
function serviceProviderVisibleFindings(state, organizationId) {}
function serviceProviderCapRows(state, organizationId, filters) {}
function serviceProviderVisibleReports(state, organizationId, reportType) {}
function serviceProviderRequiredAction(report, linkedFindings) {}
```

Visibility rules:

- Findings must match the logged-in Service Provider organization.
- Every selected Finding/report resolver must re-check `organizationId` before returning detail content; knowing or changing a client-side ID must never reveal another organization's record.
- Preliminary Reports require `released_to_service_provider` (or the equivalent explicit release field) and matching organization.
- Preliminary Report status groups are `pending_response`, `under_authority_review`, and `closed`; response Due Date comes from the released report record and may render `Not configured`.
- Final Reports require `issued === true`/`status === 'issued'`, locked final authorization, and matching organization.
- Report preview excludes Internal CAA Notes, inspector workload, internal risk scoring, private dashboard data, enforcement deliberations/referrals, and other organizations.
- CAP progress is derived from the real lifecycle: Waiting for CAP; CAP Submitted; CAP Accepted/Evidence Required; Evidence Submitted/CAA Review; Closed. It is never a decorative random percentage.
- Due Date comes from the configured Finding/record. Missing Due Dates render `Not configured`, not a severity-derived date.

### Planning budget and role UI state

Extend the existing planning item rather than creating Finance-only plans:

```js
item.budget = {
  currency: 'USD',
  requested: 12500,
  availableForPlan: 21000,
  remainingAnnualBudget: 420000,
  lines: [
    { category: 'Travel', amount: 5000 },
    { category: 'Accommodation', amount: 3000 },
    { category: 'Daily Allowance', amount: 1500 },
    { category: 'Vehicle', amount: 1000 },
    { category: 'Equipment / Tools', amount: 1500 },
    { category: 'Miscellaneous', amount: 500 }
  ]
};
```

The line total must equal `requested`; tests must reject inconsistent seed or edited demo data.

```js
state.financeUi = {
  query: '', status: 'pending', selectedPlanId: 'PLAN-2026-Q3-CABIN',
  decision: '', comment: '', openActionPlanId: ''
};

state.executiveDirectorUi = {
  dashboardRange: 'current',
  planningQuery: '', planningDepartment: 'all', planningRisk: 'all', planningStatus: 'all',
  selectedPlanId: 'PLAN-2026-Q3-CABIN', openPlanActionId: '', planDecision: '', planComment: '',
  reportQuery: '', reportOrganization: 'all', reportType: 'Final Report', reportStatus: 'all',
  selectedReportId: 'FR-2026-018', reportTab: 'summary', reportDecision: '',
  enforcementCategory: '', reportComment: '', previewZoom: 100
};
```

### Decision helpers

```js
function applyFinancePlanningDecision(item, input) {}
// input.decision: 'approve' | 'return'

function applyExecutivePlanningDecision(item, input) {}
// input.decision: 'approve_and_sign' | 'reject'

function executiveFinalReportProjection(state, filters) {}

function applyExecutiveFinalReportDecision(state, reportId, input) {}
// input.decision: 'approve' | 'enforcement_referral' | 'reject' | 'return'
```

Rules:

- Finance `return` requires a comment and invokes the configured Finance-to-GM return path.
- ED plan approval is allowed only at `pending_ed_approval`, records `mockApprovalSignature`, and preserves GM release as the next action.
- ED plan reject requires a comment and blocks release/execution.
- ED Final Report approval is allowed only for an unlocked Final Report owned by ED after configured prior reviews.
- GM approval advances to ED but cannot issue, sign, or lock.
- `enforcement_referral`, `reject`, and `return` require rationale.
- Enforcement referral stores `recommendationOnly: true` and `status: 'pending_authorized_review'`; it never mutates Findings, CAPs, certificates, or enforcement outcomes.
- Final approval stores ED actor/time/mock signature and locks the report artifact. It changes audit status according to remaining Finding/CAP/Evidence work, never by unconditional closure.

## Navigation And Route Contract

| Role | Sidebar | Home | Dedicated routes |
|---|---|---|---|
| Inspector | Existing Inspector IA | `inspector-assignments` | Submitted checklist stays on `audit-detail` or a read-only Inspector checklist route. |
| Lead Inspector | Existing Lead IA | `lead-review` | `lead-assignment`, `lead-assignment-questions`, and canonical Preliminary Report workflow. |
| Service Provider | Corrective Actions (CAP), Preliminary Reports, Final Reports, Messages, Documents, Settings | `service-provider-cap` | `service-provider-cap`, `service-provider-preliminary-reports`, `service-provider-final-reports`, `service-provider-report-preview`, plus role-safe utilities. |
| Finance Review | Finance Review | `finance-review` | One dedicated `finance-review` route; legacy `planning` redirects here and any other direct navigation returns here. |
| Executive Director | Dashboard, Planning, Final Reports, Notifications, Settings | `executive-dashboard` | `executive-dashboard`, role-specific `planning`, `executive-final-reports`, `executive-final-report-review`, `executive-final-report-preview`, `notifications`, `settings`. |

Add strict Finance and ED allowlists in `normalizeViewForRole()`. Direct navigation to Lead/Manager/Admin-only routes must return the user to that role's home without changing role state.

## File Map

- `js/data.js` — state version bump; audit-scoped assignment/execution state; report ID links; per-report drafts; Service Provider/Finance/ED UI state; planning budget detail; mock signature/enforcement fields; saved-state migration.
- `js/planning.js` — Finance and ED planning projections/decision wrappers while preserving the shared approval primitive and GM release boundary.
- `js/reports.js` — exact report lookup, selected-report finalization, mock signature helper, and non-closing report issuance rules.
- `js/inspection.js` — audit-aware Cabin checklist/Potential Finding integration where the simplified execution screen currently creates hard-coded SMS findings.
- `js/manager-workspaces.js` — Finance/ED queue projections; change GM final review from issue/lock to forward-to-ED; add ED final decision validation and mutation.
- `js/views.js` — state-driven assignment, Cabin execution, success modal content helper, Lead Preliminary workflow, Service Provider CAP/Preliminary/Final Report surfaces, Finance surfaces, ED dashboard/planning/report surfaces, and shared report document template.
- `js/app.js` — role navigation/allowlists, route dispatch, field/action handlers, ID-preserving navigation, notifications, mock downloads, and decision orchestration.
- `css/styles.css` — selected Inspector states, success timeline, Lead Preliminary Findings layout, Service Provider queue/dossier/report layouts, Finance budget dossier, ED dashboard/planning/report dossier, viewer/template, responsive and focus styles.
- `index.html` — load/cache token update after runtime/CSS changes.
- `tests/lead-inspector-nav-smoke.test.js` — per-question/Inspector persistence and Add Inspector behavior.
- `tests/lead-inspector-workspace-smoke.test.js` — canonical Preliminary workflow and Findings Review instead of the legacy generic report page.
- `tests/inspection-team-smoke.test.js` — canonical team membership/duplicate prevention if shared roster mutations change.
- `tests/inspection-execution-smoke.test.js` — audit-aware Cabin sections/questions/finding linkage.
- `tests/inspector-nav-smoke.test.js` — first-click success modal, idempotency, stable timestamp, and no Lead route.
- `tests/audit-work-queue-smoke.test.js` — Cabin audit opens Cabin execution content.
- `tests/service-provider-portal-smoke.test.js` — new focused portal navigation, organization isolation, CAP Level/Status/Due Date/progress/respond, Preliminary visibility, and Final Report table/detail behavior.
- `tests/service-provider-final-report-smoke.test.js` — selected Final Report preview/download, auditee-safe tabs/content, and state-backed CAP linkage.
- `tests/planning-workspace-smoke.test.js`, `tests/planning-render-smoke.test.js`, `tests/governance-render-smoke.test.js` — role-specific Finance/ED planning views and preserved chain.
- `tests/finance-review-workspace-smoke.test.js` — new focused Finance navigation, budget math, queue, decision, validation, and ownership test.
- `tests/executive-director-workspace-smoke.test.js` — new focused ED navigation, dashboard, plan/report decision, preview, signature, enforcement, and authority test.
- `tests/report-approval-smoke.test.js`, `tests/general-manager-workspace-smoke.test.js`, `tests/manager-reports-approval-smoke.test.js` — GM forwards; ED issues/locks; Findings remain unchanged.
- `tests/manager-report-pdf-smoke.test.js` — selected report ID/operator/template and PDF header/filename.
- `tests/demo-boundary-smoke.test.js` — mock signature/enforcement and no-production-boundary assertions.
- `tests/premium-ui-remediation-smoke.test.js`, `tests/manager-workspace-responsive-smoke.test.js` — changed markup and responsive guards.
- `docs/product-specs/modules/AUDITEE_PORTAL.md` and `.turkce.md` — CAP plus separate Preliminary/Final Report portal contract.
- `docs/product-specs/ux-plan/NAVIGATION_AND_INFORMATION_ARCHITECTURE.md` and `.turkce.md` — Service Provider/Finance/ED role IA.
- `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md` and `.turkce.md` — portal visibility, Finance/ED permissions, and mock signature/enforcement boundaries.
- `docs/product-specs/screen-specs/DEPARTMENT_MANAGER_WORKSPACES.md` and `.turkce.md` — correct final authority from GM issuer to GM reviewer plus ED issuer.
- `docs/demo-evidence/BUILD_SUMMARY.md` and `.turkce.md` — implemented demo behavior and fresh verification evidence.
- `MANIFEST.md` — register the three new smoke-test files.
- `docs/exec-plans/index.md`, this plan, and `docs/exec-plans/tech-debt-tracker.md` — lifecycle, next todo, and durable production-boundary note.

## Task 1: Lock The Desired Contracts With Failing Tests

**Files:**

- Create: `tests/finance-review-workspace-smoke.test.js`
- Create: `tests/executive-director-workspace-smoke.test.js`
- Modify the focused existing tests listed in the File Map.

**Interfaces produced:** executable role, route, state, authority, and report-linkage contracts used by every later task.

- [x] Add a Lead assignment test that assigns `CAB-Q001`/`CAB-Q002` to one Inspector and `CAB-Q003`/`CAB-Q004` to another, rerenders, and asserts both user IDs remain mapped.
- [x] Add Add Inspector tests for active internal Inspector filtering, duplicate prevention, canonical `inspectionTeams.memberIds` persistence, and computed roster/workload counts.
- [x] Replace the old Inspector second-click/Lead-workspace assertions with first-click modal, stable stored timestamp, `state.role === 'inspector'`, unchanged audit context, read-only submitted view, return-to-assignments, and idempotent resubmission assertions.
- [x] Assert the exact six Cabin section labels/order and the absence of SMS copy for `AUD-2026-001`.
- [x] Assert Report ID, row action, and report chooser all call the same report-specific workflow and show `Inspection & Findings` plus Findings Review by default.
- [x] Add Service Provider tests for exact navigation, organization isolation, CAP Level/Status/Due Date/progress/respond, Preliminary release visibility, Final issue visibility, exact Final Report columns, selected dossier, and auditee-safe preview/download.
- [x] Add Finance tests for a single operational navigation entry/home, budget total validation, two decisions, required return reason, GM return ownership, and ED advance ownership.
- [x] Add ED tests for exact navigation/allowlist, Dashboard home, single Planning action trigger, stage-gated `Approve & Sign (Demo)`, rejection reason, mock signature, and post-approval GM release ownership.
- [x] Add report-authority tests proving GM cannot issue/lock and ED can issue/lock only an eligible Final Report.
- [x] Assert enforcement referral requires category/rationale, remains recommendation-only, and leaves audit/Finding/CAP/Evidence states unchanged.
- [x] Assert selected-report Review -> Preview -> Return keeps the same Report ID and that the rendered/downloaded template contains Fly Namibia and its actual counts.
- [x] Run the targeted tests before implementation. Expected result: failures identify the current global assignment state, SMS content, wrong Preliminary route, hard-coded/duplicated Service Provider projections, missing Finance/ED views, and GM final-authority behavior.

## Task 2: Normalize State, IDs, And Saved-State Migration

**Files:** `js/data.js`, `js/reports.js`, `js/planning.js`, `js/manager-workspaces.js`, focused tests.

**Interfaces consumed:** Task 1 test contracts.
**Interfaces produced:** audit-scoped assignment/execution state, exact report lookup, Service Provider/Finance/ED UI state, budget structure, and role-correct decision helpers.

- [x] Bump `DEMO_STATE_VERSION` and add ID-based migration/defaulting for every new state object; do not require users to clear `localStorage` to see new seed records.
- [x] Replace the global Lead assignment working state with `leadAssignmentsByAudit[auditId]`. Migrate the existing selected questions, assignee, due date, priority, note, and timestamps into the active audit record.
- [x] Replace global Inspector execution/submission fields with `inspectionWorkspaces[auditId]`, preserving compatible saved answers and submission timestamps.
- [x] Add `approvalPackageId`/exact report identity links to interactive Preliminary/Final read models. Remove first-match ambiguity from `reportForAudit()` consumers by using `reportForAuditAndType()` or `reportArtifactById()`.
- [x] Move report draft working state under `preliminaryReportDrafts[reportId]` so editing one report cannot change another.
- [x] Add `serviceProviderUi` defaults and organization-scoped Finding/report selectors. Remove interactive dependence on the hard-coded `serviceProviderFinalReportMeta()` and `serviceProviderCapRequirements()` arrays.
- [x] Add Finance/ED UI state defaults and normalize selected IDs against available records.
- [x] Extend the canonical planning item with numeric currency/budget lines and validate the line total against the requested total.
- [x] Seed enough historical read-only planning/final-report rows to demonstrate filtering and status counts, but keep the primary interactive plan/report linked to the canonical Cabin/Fly Namibia records.
- [x] Change the current GM final-report mutation so GM approval advances ownership to ED without issuing, signing, locking, closing the audit, or closing Findings.
- [x] Add the Finance and ED planning wrappers and ED report decision helper with the validation/side effects defined above.
- [x] Run Task 1 data/logic tests. Expected result: all pure state and ownership assertions pass before UI work.

## Task 3: Make Lead Assignment Truly Multi-Inspector

**Files:** `js/views.js`, `js/app.js`, `css/styles.css`, `tests/lead-inspector-nav-smoke.test.js`, `tests/inspection-team-smoke.test.js`.

**Interfaces consumed:** `leadAssignmentsByAudit`, `inspectionTeams.memberIds`, canonical `users`.
**Interfaces produced:** per-question assignment UI and computed metrics.

- [x] Refactor `leadAssignmentInspectors()` to resolve active CAA Inspectors from the selected audit team and `state.users`; stop maintaining a separate roster of unrelated display names.
- [x] Refactor `leadAssignmentAvailableInspectors()` to return active internal Inspector users not already in the audit team.
- [x] Make every Inspector card a semantic selectable button with `aria-pressed`, a strong active state, and synchronized `Assign To` selection.
- [x] Make Add Inspector append the selected user ID to the selected audit team's `memberIds`, reject duplicates, select the new member, persist, and rerender.
- [x] Update `handleLeadAssignmentAssign()` to write one assignment object per selected question without overwriting earlier assignments owned by other Inspectors.
- [x] Derive assigned/unassigned totals, per-Inspector workload, team count, summary, progress bars, and filters from the assignment map. If the UI represents only the curated question subset, label/count that subset rather than displaying unsupported `126 / 42 / 84` values.
- [x] Make Reassign and Remove Assignment operate only on selected question IDs; remove the hard-coded Maria Silva path.
- [x] Release notifications to the assigned Inspector role/users, not back to Lead Inspector, while keeping them in-UI/demo-only.
- [x] Verify keyboard selection, Add Inspector modal focus/close behavior, rerender persistence, and responsive layout.
- [x] Run the two focused tests. Expected result: multi-person mappings, duplicate prevention, and computed metrics pass.

## Task 4: Replace The SMS Execution Surface And Fix Inspector Submission

**Files:** `js/data.js`, `js/inspection.js`, `js/views.js`, `js/app.js`, `css/styles.css`, Inspector execution/navigation tests.

**Interfaces consumed:** `inspectionWorkspaces[auditId]`, canonical Cabin checklist, selected audit.
**Interfaces produced:** audit-aware execution package, read-only submitted view, and success status modal.

- [x] Add an execution-package resolver that returns the opened audit's title, organization, dates, inspection ID, published template, sections, questions, answers, and progress.
- [x] For the canonical Cabin audit, render these sections in order: Galley; Lavatories; Passenger Seats; Emergency Equipment; Video + Crew Seat; Cockpit, Cabin General Condition + Exits.
- [x] Replace the hard-coded SMS question IDs/content/download text/finding target with the selected audit/template/question data. Reuse the canonical Potential Finding path in `js/inspection.js` rather than creating `SMS-*` Findings against `AUD-2026-005`.
- [x] Keep status values `Compliant`, `Non-Compliant`, `Observation`, and `Not Applicable`; enforce required comment behavior where the existing checklist rules require it.
- [x] On first submit, persist the exact timestamp once, notify Lead Inspector, rerender the submitted state, and immediately open `Checklist Submitted Successfully` without changing role or route.
- [x] Show actual Inspection ID, timestamp, `Waiting for Lead Inspector Review`, next step, and a five-stage timeline: Checklist Completed; Submitted; Lead Inspector Review (current); Final Report Preparation; Department Approval.
- [x] `View Submitted Checklist` opens the same audit in Inspector-owned read-only state. `Return to My Assignments` navigates to `inspector-assignments`.
- [x] Reopening the submitted button shows the same status/timestamp. Remove every Inspector CTA that navigates to `lead-assignment-questions`.
- [x] Add an Inspector route guard for Lead-only assignment/review routes.
- [x] Run the focused Inspector tests. Expected result: Cabin content, stable submission state, idempotency, and role isolation pass.

## Task 5: Make Preliminary Reports Existing-Artifact And Findings-First

**Files:** `js/data.js`, `js/reports.js`, `js/views.js`, `js/app.js`, `css/styles.css`, Lead report tests.

**Interfaces consumed:** exact report IDs, `preliminaryReportDrafts[reportId]`, linked audit/findings.
**Interfaces produced:** one canonical Preliminary Report opening path.

- [x] Replace `+ New Preliminary Report` with `Open Report Package` or `Continue Existing Report`. If multiple eligible packages exist, open a real selector listing Report ID, Audit ID, organization, status, owner, and next action.
- [x] Implement `openLeadPreliminaryReport(reportId)` to set selected Report ID, workflow mode, and default `inspection` step; do not navigate by generic `auditId` alone.
- [x] Route the Report ID, row action, and package selector through that single helper. Remove the current generic `viewAuditReportsApproval()` detour for these Lead interactions.
- [x] Resolve report metadata and Findings from the selected report's linked audit. Keep historical read-only rows non-editable if they do not have a complete package.
- [x] Make `Inspection & Findings` the default step and show total/severity/status, inclusion state, Review/View action, and selected count without clipping.
- [x] Keep Report Content, Attachments, and Review & Submit state isolated by Report ID. Mock uploads remain filenames only.
- [x] Ensure every visible report action performs a screen/state change; do not use a toast as the only result.
- [x] Run Lead report tests at desktop-width DOM assumptions. Expected result: all three entry points open the same Findings-first workflow.

## Task 6: Rebuild The Service Provider CAP, Preliminary Reports, And Final Reports Workspaces

**Files:** `js/data.js`, `js/reports.js`, `js/views.js`, `js/app.js`, `css/styles.css`, Service Provider tests.

**Interfaces consumed:** organization-scoped Finding/report selectors, `serviceProviderUi`, shared report viewer.
**Interfaces produced:** three distinct external workspaces with one canonical record identity and strict privacy.

- [x] Replace duplicate Auditee navigation with operational entries `Corrective Actions (CAP)`, `Preliminary Reports`, and `Final Reports`, followed by existing Messages, Documents, and Settings utilities. Set the Service Provider home to `service-provider-cap` and add a strict allowlist.
- [x] Build Corrective Actions (CAP) from `visibleFindings()`/canonical organization-scoped Findings, not `serviceProviderCapRequirements()`. Derive Total, Open, In Progress, Awaiting Review, and Closed groups from lifecycle state.
- [x] Guard both list and selected-detail resolution by the authenticated demo organization's `organizationId`; an arbitrary `viewFinding(id)` or report ID must resolve to `not found/forbidden` when it is outside that organization.
- [x] Render CAP filters/tabs plus columns Finding ID, Audit/Inspection, Finding Title, Level, Status, Due Date, Progress, and Action. Use the configured Due Date and real lifecycle-derived progress; render `Not configured` when absent.
- [x] Build the selected CAP dossier with Finding ID, Level, canonical Status, Audit/Inspection, Due Date, title/description, CAP timeline, current owner/next action, and a `Respond` button that opens the existing CAP/evidence form appropriate to the current lifecycle step.
- [x] Preserve CAP acceptance != closure, evidence version history, auditee-visible comments, and the absence of Internal CAA Notes.
- [x] Build Preliminary Reports KPI/filter state for Total, Pending Your Response, Under Review by Authority, and Closed. Show only explicitly shared/released Preliminary Reports for Fly Namibia.
- [x] Render Preliminary columns Report ID, Audit/Inspection, Date Shared, Findings, Due Date, and Action. Due Date is the configured response target; no severity-derived deadline is allowed.
- [x] Build the selected Preliminary dossier with status, report summary, Date Shared, Shared By, Total Findings, Response Due Date, external visibility/confidentiality label, auditee-safe description, allowed attachments, `View Report`, and `Send Message to Inspector` using the existing in-UI message flow.
- [x] Build Final Reports KPI/filter state for Total Final Reports, Reports Requiring CAP, Closed Reports, and This Year. Show only ED-issued/locked Fly Namibia Final Reports. Derive `Closed Reports` from reports whose linked Findings are all closed; do not reintroduce a report Status column.
- [x] Render the exact simplified Final Report columns: Report ID, Audit/Inspection, Date Released, Findings, and Action. Do not render Status or Required Action columns.
- [x] Build the selected Final Report dossier with Report ID, audit/date/period/Lead Inspector/version/findings/classification metadata, auditee-safe objective/scope, allowed attachments, `View Report`, individual mock downloads, and `Download All` state change.
- [x] Route Preliminary/Final `View Report` through the shared state-backed viewer with an auditee-safe projection. Never expose internal notes, enforcement deliberations, internal risk, workload, other organizations, or unreleased drafts.
- [x] Ensure Contact/Send Message opens a real in-UI composer/thread and every mock document action opens preview or generates a browser-local demo download rather than only showing a toast.
- [x] Run Service Provider tests at desktop/mobile assumptions. Expected result: navigation, isolation, CAP lifecycle, Preliminary visibility, exact Final columns, selected IDs, preview, and mock downloads pass.

## Task 7: Build The Simplified Finance Review Workspace

**Files:** `js/app.js`, `js/planning.js`, `js/views.js`, `css/styles.css`, Finance/planning tests.

**Interfaces consumed:** planning approval chain, numeric budget structure, `financeUi`.
**Interfaces produced:** one Finance Review queue/dossier and two-decision UI.

- [x] Replace the current generic Finance navigation with one operational entry, `Finance Review`; add the dedicated `finance-review` route, make it the Finance home/strict allowlist destination, and redirect the legacy Finance `planning` route to it.
- [x] Build one focused Finance Review page with a compact pending-count/total-budget summary strip, review queue, selected plan budget dossier, and approval-history rail. Do not add a separate Finance Dashboard or unrelated modules.
- [x] Show tabs Budget Summary, Budget Breakdown, Supporting Documents, and Comments & History. All displayed documents remain mock filenames.
- [x] Show only `Approve Budget` and `Return for Revision` as primary decisions. Return requires a reason; approval note is optional.
- [x] Finance approval advances to ED. Return goes to GM action. Neither action signs/releases the plan, edits scope, or bypasses ED.
- [x] Keep the canonical approval rail order Department Manager -> GM -> Finance Review -> Executive Director.
- [x] Run Finance and shared planning tests. Expected result: simple UI and ownership transitions pass without changing the release chain.

## Task 8: Add The Executive Director Shell And Dashboard

**Files:** `js/app.js`, `js/views.js`, `css/styles.css`, ED tests.

**Interfaces consumed:** `executiveDirectorUi`, planning/report projections, notifications.
**Interfaces produced:** ED home/dashboard and role boundary.

- [x] Replace duplicate ED navigation with Dashboard, Planning, Final Reports, Notifications, Settings and add a strict ED allowlist.
- [x] Set `homeView('executiveDirector')` to `executive-dashboard` and dispatch dedicated ED renderers.
- [x] Build six derived KPI cards: Total Audits; Audits in Progress; Pending Approval; Final Reports; Overdue Actions; Closed This Period.
- [x] Put pending Planning/Final Report queues in the first decision viewport. Add department overview, one restrained risk context, and recent overdue actions below; avoid a first-screen chart wall.
- [x] Make every KPI/queue row navigate to a filtered Planning or Final Reports view using the selected record ID.
- [x] Keep risk indicators informational only and include the non-automatic decision guardrail where risk context is shown.
- [x] Verify utility navigation, direct-route normalization, keyboard focus, and responsive stacking.
- [x] Run ED navigation/dashboard tests. Expected result: ED lands on a focused working dashboard and cannot enter role-inappropriate routes.

## Task 9: Build ED Planning List, Detail, Approve/Sign, And Reject

**Files:** `js/planning.js`, `js/views.js`, `js/app.js`, `css/styles.css`, ED/planning tests.

**Interfaces consumed:** `planningItems`, `applyExecutivePlanningDecision`, ED UI state.
**Interfaces produced:** state-backed Planning queue/detail and mock approval signature.

- [x] Render stage cards for Draft, Department Review, GM Review, Finance Review, ED Final Approval, and Rejected/Returned, with counts derived from state.
- [x] Add search, Department, Risk Category, date, and status filters that update the visible queue.
- [x] Use one row-level `Review / Take Action` trigger. Its menu may offer Review Plan, Preview Full Plan, and `Approve & Sign (Demo)` when eligible; quick choices must open the selected detail/decision state and never commit a decision immediately.
- [x] Build the selected plan side summary and approval flow, then a full detail panel with Overview, Plan Information, Departments & Scope, Budget & Resources, Approval History, and Documents & Notes.
- [x] Put `Approve & Sign (Demo)` and Reject in the decision panel. Require Reject rationale; show a clear demo-signature notice before approval confirmation.
- [x] On approval, call the shared approval primitive, record actor/time/history/mock approval mark, and show GM Release to Department as the next action. Do not release or schedule automatically.
- [x] On Reject, record the required reason, prevent release/execution, update queue counters, and preserve history.
- [x] Preview/Download Plan must use the selected `planId`; downloads are browser-generated demo artifacts with no real document service.
- [x] Run ED planning and shared governance tests. Expected result: selected IDs, decision validation, signature boundary, and next-owner continuity pass.

## Task 10: Build ED Final Reports Queue And Decision Dossier

**Files:** `js/manager-workspaces.js`, `js/reports.js`, `js/views.js`, `js/app.js`, `css/styles.css`, report authority tests.

**Interfaces consumed:** `executiveFinalReportProjection`, selected report, role-correct final decision helper.
**Interfaces produced:** ED Final Reports list/review and enforcement referral.

- [x] Render Final Report summary cards for Total, Pending Approval, Approved, and Returned/Rejected; derive values from actual report records.
- [x] Add search, organization, report type (fixed to Final Report or hidden), and status filters.
- [x] Render Report ID, organization, audit type, submitted by/on, status, Due Date, and one Review action. Only Final Reports appear in this operational module.
- [x] Build the selected Review page with report metadata and tabs Executive Summary, Findings Summary, Documents, and History.
- [x] Add decision choices: Approve Report; Refer for Enforcement Review; Reject Report; Return for Revision. Require rationale for every non-approval choice.
- [x] When Enforcement Review is selected, reveal a real dropdown with configured demo categories such as Administrative Fee, Partial Suspension, Full Suspension, Certificate/License Revocation, Conditional Approval, and Other. Label the block as referral/recommendation-only.
- [x] Confirming enforcement referral records category/rationale/recommendation-only status and does not apply a sanction or close any audit/Finding.
- [x] GM may only advance a reviewed report to ED. ED approval records mock signature, issues/locks the selected report, and computes audit status from remaining follow-up work.
- [x] Disable terminal actions after decision and preserve append-only demo history.
- [x] Run report authority and boundary tests. Expected result: ED-only final issue/lock, validation, and non-automatic enforcement pass.

## Task 11: Add State-Backed Full Report Preview, Template, And Download

**Files:** `js/views.js`, `js/app.js`, `js/reports.js`, `js/manager-workspaces.js`, `css/styles.css`, PDF/report tests.

**Interfaces consumed:** selected report, linked audit/findings/team/signature.
**Interfaces produced:** reusable report document renderer and selected-report download.

- [x] Extract the useful visual structure from `viewLeadFinalReportDocument()` into `finalReportDocumentHtml(report, audit, findings, team)`; remove hard-coded SkyCargo/INS values.
- [x] Render the A4-style first page with AviaSurveil360/report header, Report ID/version/date/confidentiality label, organization/audit metadata, audit team, submitted by/on, current status, executive summary, findings overview, detailed findings summary, conclusion, next steps, and mock signature area.
- [x] Make `Preview Full Report` open a dedicated selected-report route with return-to-review, contents outline, actual demo page count, zoom, print, and download controls.
- [x] Do not display `1 / 56` unless 56 pages are actually generated. Use the actual count or `Sample page 1`.
- [x] Parameterize browser-generated PDF lines/document/filename by selected report. Keep a valid PDF header and visibly include `Demo-only browser-generated document`.
- [x] Ensure report preview counts and download counts use the same linked Findings selector.
- [x] Verify preview -> review round-trip, selected ID stability, print/download behavior, and mock signature appearance only after approval.
- [x] Run report/PDF tests. Expected result: selected Fly Namibia report metadata and valid demo PDF evidence pass.

## Task 12: Apply Premium Visual, Responsive, And Accessibility QA

**Files:** `css/styles.css`, `js/views.js`, `index.html`, visual/responsive smoke tests.

**Interfaces consumed:** all final markup from Tasks 3-11.
**Interfaces produced:** stakeholder-ready visual hierarchy without behavior changes.

- [x] Reuse existing command headers, approval packages, selected dossiers, status pills, decision bars, and report paper styles before adding new primitives.
- [x] Give assignment Inspector cards a strong but restrained active state and keep question/assignee/action relationships visually obvious.
- [x] Give the submit modal a clear success state and readable five-step timeline without redirecting attention to Lead-only work.
- [x] Keep Preliminary Findings Review visible at `1536x864`, `1366x768`, and `1024x768`; prevent clipping of status/View controls.
- [x] Keep Service Provider CAP/Preliminary/Final screens queue-first with a clearly selected right-side dossier on desktop; preserve exact columns and prevent external records/actions from collapsing into a generic card wall.
- [x] Keep Finance visually simple: one queue, one selected budget dossier, one history rail, and two decision actions; no Dashboard/card wall.
- [x] Keep ED decision-first: selected plan/report larger than supporting queue on desktop; decision panel visible without hidden primary controls.
- [x] On mobile, stack queue -> selected summary -> detail -> decision in task order; use full-screen drawers for plan/report detail where needed.
- [x] Add visible focus styles, semantic button labels, `aria-expanded`/`aria-pressed`, modal focus return, escape-to-close, and non-color status labels.
- [x] Bump all CSS/JS asset query tokens in `index.html` once the visual/runtime pass is complete.
- [x] Run responsive markup/style tests. Expected result: no known fixed-value/hard-coded visual contract remains and all changed controls are reachable.

## Task 13: Update Product Truth, Evidence, And Plan State

**Files:** the English/Turkish product specs, build summaries, `MANIFEST.md`, this plan, index, and tracker.

**Interfaces consumed:** verified implementation and evidence.
**Interfaces produced:** synchronized product rules, package inventory, and handoff.

- [x] Update Auditee Portal plus Service Provider/Finance/ED IA and permissions in the canonical English docs and matching `.turkce.md` companions.
- [x] Correct the Department/GM screen spec so GM is an intermediate reviewer and ED is the final issue/sign authority.
- [x] Register all three new smoke tests in `MANIFEST.md`.
- [x] Update both build summaries with literal `demo-only`, `verified locally`, `not run`, and `production-readiness not claimed` language based on actual evidence.
- [x] Record executed task checkboxes and the exact final verification evidence in this plan.
- [x] Keep the plan index at `active` during implementation, `ready-for-verification` only after all required local checks pass, and `completed` only after stakeholder sign-off and archive requirements are met.
- [x] Keep the durable mock signing/enforcement boundary note open until product/legal/security owners define production identity, signature, authority, audit-log, retention, and enforcement policy.

## Verification

### Focused logic and render checks

Run after the owning task, not only at the end:

```bash
node tests/lead-inspector-nav-smoke.test.js
node tests/lead-inspector-workspace-smoke.test.js
node tests/inspection-team-smoke.test.js
node tests/inspection-execution-smoke.test.js
node tests/inspector-nav-smoke.test.js
node tests/audit-work-queue-smoke.test.js
node tests/service-provider-portal-smoke.test.js
node tests/service-provider-final-report-smoke.test.js
node tests/finance-review-workspace-smoke.test.js
node tests/executive-director-workspace-smoke.test.js
node tests/planning-workspace-smoke.test.js
node tests/planning-render-smoke.test.js
node tests/governance-render-smoke.test.js
node tests/report-approval-smoke.test.js
node tests/general-manager-workspace-smoke.test.js
node tests/manager-reports-approval-smoke.test.js
node tests/manager-report-pdf-smoke.test.js
node tests/demo-boundary-smoke.test.js
```

Expected: every command prints its `...: ok` marker or passes under its current Node test format.

### Full local suite

```bash
for file in js/*.js; do node --check "$file" || exit 1; done
node --test tests/*.test.js
git diff --check
```

Expected: all JavaScript parses, all tests pass, and the diff has no whitespace errors.

### Rendered browser matrix

Serve locally only when browser behavior requires it:

```bash
python3 -m http.server 4173 --bind 127.0.0.1
```

Use an isolated browser profile and verify at:

- `1536 x 864`
- `1366 x 768`
- `1024 x 768`
- `390 x 844`

Required routes/interactions:

- Lead Inspector: Assignment Questions -> select Inspector -> Add Inspector -> assign separate batches -> release.
- Inspector: My Assignments -> Cabin Inspection -> six sections -> submit -> success modal -> submitted checklist -> return.
- Lead Inspector: Preliminary Reports -> Report ID -> Findings Review -> content/attachments/review -> back to list.
- Service Provider: Corrective Actions -> Level/Status/Due Date filters -> selected dossier -> Respond; Preliminary Reports -> selected shared report -> View/Message; Final Reports -> exact simplified columns -> selected report -> View/Download.
- Finance: Finance Review queue -> selected budget -> approve; reset -> return with reason.
- ED: Dashboard -> Planning -> single action -> detail -> preview -> approve/sign; reset -> reject with reason.
- ED: Final Reports -> Review -> enforcement dropdown validation -> full preview -> return -> approve/sign on eligible record.

For each changed route verify:

- no console warning/error
- no page-level or nested horizontal overflow
- no clipped action/status text
- primary decision visible and keyboard reachable
- selected ID remains correct through navigation
- no stale hard-coded organization/audit/report values
- auditee privacy and Finding/CAP/Evidence closure rules remain unchanged

After browser/GUI automation, inspect and clean up leftover Chrome, Chrome Helper, headless Chrome, webdriver, Playwright, Puppeteer, and test-server processes before reporting completion.

### Boundary checks

- No real sign-in/e-signature was added.
- No real enforcement action was executed or implied.
- No report approval closed a Finding or bypassed CAP/Evidence requirements.
- No Service Provider route exposed another organization, unreleased report, Internal CAA Note, enforcement deliberation, workload, or internal risk score.
- No ED plan approval bypassed GM release.
- No Finance decision bypassed ED.
- No mock file interaction read or uploaded file contents.
- No backend, API, database, framework, real notification delivery, or production audit log was added.

## Implementation Evidence — 2026-07-10

> **Independent final-review override — 2026-07-10:** The implementation-era
> evidence below is not sufficient for final acceptance. A fresh independent
> review reproduced Blocking report lifecycle, GM/ED authority, Service Provider
> privacy, stale Lead Final Report, responsive nested-overflow, visible-control,
> and product-truth defects. This plan remains `blocked` until
> [Stakeholder Readiness Final Remediation](2026-07-10-stakeholder-readiness-final-remediation-plan.md)
> completes and receives an independent GO verdict.

**Scope status:** Tasks 1-13 implemented in order. Each implementation task was
committed and pushed separately on the existing `main` branch as explicitly
requested by the user. No branch was created or switched, and no PR or deploy
was performed.

**verified locally**:

- Multi-Inspector assignment and Add Inspector persist canonical user IDs per
  Audit ID and Question ID, prevent duplicates, derive roster/workload counts,
  and survive rerendering.
- The Inspector opens the Fly Namibia Cabin Inspection with the exact six
  Cabin sections, submits once, receives the success/status modal immediately,
  remains in the Inspector role, and can reopen a read-only submitted checklist
  or return to My Assignments without a Lead Inspector redirect.
- Preliminary Report ID, row action, and package selection open the same
  Findings-first workflow with visible Findings Review and stable selected IDs.
- Service Provider CAP, Preliminary Reports, and simplified Final Reports are
  organization-scoped and use canonical Finding/report state. Out-of-scope IDs,
  internal notes, enforcement deliberations, workload, internal risk, other
  organizations, and unreleased drafts do not render.
- Finance uses one Finance Review workspace and only Approve Budget / Return for
  Revision decisions. Approval advances to ED; return goes to GM action.
- Executive Director has the exact Dashboard, Planning, Final Reports,
  Notifications, and Settings allowlist. ED plan approval records a mock mark
  and retains `GM / Release to Department`; ED alone can issue/mock-sign/lock
  an eligible Final Report.
- The selected Final Report drives dossier, preview, template, counts, team,
  mock signature, and browser-generated PDF filename/content. Enforcement is a
  recommendation-only referral with no sanction or closure side effect.
- CAP acceptance remains distinct from Finding closure, and report decisions do
  not bypass required CAP, Evidence, or verification work.
- JavaScript syntax: every `js/*.js` file passed `node --check`.
- Full suite: `node --test tests/*.test.js` passed 34 tests, 0 failed.
- Whitespace: `git diff --check` passed.
- Rendered browser QA passed at `1536x864`, `1366x768`, `1024x768`, and
  `390x844` with no console warning/error, no measured page-level horizontal
  overflow, no clipped primary control, stable record IDs, working navigation/
  state/modal/preview/download actions, and correct mobile task ordering.
- Browser/test cleanup found no temporary HTTP server or separately launched
  test Chrome process left running.

**demo-only**: role selection, persistence, approval marks, downloads,
notifications, uploads, report generation, and enforcement referrals remain
browser-local simulations.

**not run**: backend/database/API integration, real authentication or
authorization enforcement, real signature identity/legal validity, immutable
production audit logging, real storage/notification/reporting/enforcement,
release, deployment, penetration testing, accessibility certification, and
stakeholder acceptance.

**production-readiness not claimed**. Stakeholder review/sign-off is the next
todo before this plan can move to `completed/`. The production signing and
enforcement authority contract remains `note-open` in the durable tracker.

## Risks And Mitigations

- **Two report read models can drift.** Add exact package links and keep lifecycle mutation in one canonical interactive package; test list/review/preview identity end-to-end.
- **Service Provider hard-coded projections can diverge from canonical Findings/reports.** Replace them with organization-scoped selectors and verify CAP/list/detail/preview IDs together.
- **Severity-derived deadlines can look like binding regulation.** Render only configured Due Dates/response targets and use `Not configured` when absent.
- **Saved browser state can hide new seeds/state shapes.** Bump state version and merge new IDs/defaults without requiring a manual reset.
- **Assignment totals can misrepresent the 126-row source workbook.** Compute from represented rows and clearly distinguish `126 source rows` from the curated executable subset.
- **Reference screenshots can overwrite canonical content.** Reuse layout only; keep Cabin Inspection/Fly Namibia and careful regulatory wording.
- **Mock signing can look legally valid.** Use `Approve & Sign (Demo)` and `not a real e-signature` adjacent to every signature control/output.
- **Enforcement dropdown can imply automatic sanctions.** Use referral/recommendation language, required rationale, separate stored status, and no operational side effect.
- **Changing final authority can regress GM tests.** Update GM and ED authority tests together before changing renderers.
- **A richer ED dashboard can become a chart wall.** Put pending decisions first and move secondary trends/risk context below.
- **Broad CSS overlap with Modern Aviation SaaS rollout.** Execute this plan's functional surfaces first and do not run overlapping broad styling work concurrently in `css/styles.css`, `js/views.js`, or `js/app.js`.

## Dependencies And Sequencing

- [Planning Panel Simplification](2026-06-30-planning-panel-simplification-plan.md) supplies the canonical shared Planning panel, approval primitive, and explicit post-ED GM release step.
- [CAA Governance Workflow And Multi-Role Expansion](2026-06-28-caa-governance-workflow-and-roles-plan.md) supplies the intended `Lead -> Department Manager -> GM -> ED` report chain and demo signature/enforcement boundaries.
- [Cabin Inspection Demo Scenario](2026-07-09-cabin-inspection-demo-scenario-plan.md) supplies the six Cabin sections, canonical PBE question, Fly Namibia story, and workbook/source boundary.
- `docs/product-specs/modules/AUDITEE_PORTAL.md` and `docs/product-specs/workflows/FINDING_CAP_EVIDENCE_WORKFLOW.md` supply organization isolation, auditee actions, evidence versioning, and closure rules.
- [Department And General Manager Oversight Workspaces](2026-07-09-department-manager-workspaces-plan.md) is the current manager/GM baseline. This follow-up changes final issue authority from GM to ED; it must update the related tests/docs without inferring completion for the baseline plan.
- [Premium UI Remediation](2026-07-08-premium-ui-remediation-plan.md) is the verified visual baseline.
- [Modern Aviation SaaS Rollout](2026-07-08-modern-aviation-saas-rollout-plan.md) remains the broader visual program. This plan owns only the stakeholder-specific Inspector/Lead/Service Provider/Finance/ED deltas and should run before overlapping phases 5/6/7 or be explicitly folded into them.

## Ownership Boundaries

- Product/stakeholder owner approves whether the proposed Service Provider, Finance, ED, and report-template simplifications are acceptable.
- Product/legal/security owners must define any future production signature identity, legal effect, enforcement authority, allowed action taxonomy, retention, and audit evidence.
- Frontend implementation may create only browser-local demo state, mock signatures, mock downloads, and in-UI notifications.
- Regulatory references remain configured references, not legal advice or automatic findings/enforcement rules.
- Implementation agents may modify only in-scope static demo files, focused tests, and synchronized docs; unrelated local changes remain untouched.

## Completion Gate

The plan may move to `ready-for-verification` only when:

- all Tasks 1-13 are checked with evidence
- all syntax and Node tests pass
- the rendered route/viewport matrix passes with no console errors or actionable clipping/overflow
- multi-Inspector assignment, Cabin submission, linked Lead Preliminary Findings Review, Service Provider CAP/Preliminary/Final Report flows, Finance decisions, ED plan decisions, ED report decisions, and full preview are verified end-to-end
- mock signature/enforcement labels and non-automatic side effects are verified
- English/Turkish docs, `MANIFEST.md`, plan index, plan status, and durable tracker are synchronized
- any unrun external/production evidence is labelled `not run`; production readiness is not claimed

## Self-Review

- Spec coverage: every supplied feedback item and follow-up screenshot maps to a task and acceptance case.
- Placeholder scan: all state shapes, routes, decisions, validation rules, files, commands, and expected outcomes are explicit.
- Type consistency: role keys remain `inspector`, `leadInspector`, `auditee`, `finance`, and `executiveDirector`; canonical IDs use `organizationId`, `auditId`, `planId`, `reportId`, `findingId`, `questionId`, and `inspectorUserId` consistently.
- Scope check: the plan corrects targeted workflow/authority/UI deltas without adding production services or parallel CAP/report lifecycles.
- Lifecycle check: Finance advances to ED; ED plan approval advances to GM release; GM report approval advances to ED; ED report approval cannot close open Findings.

## Execution Prompt

Use this exact prompt to execute the plan:

```text
Implement docs/exec-plans/active/2026-07-10-inspector-report-and-governance-workflow-remediation-plan.md in /Users/marlonjd/Developer/web/aviaSurveil360.

Read AGENTS.md and the plan's linked active-plan/product sources first. Work on the current branch and preserve unrelated dirty-worktree changes. Do not create or switch branches, commit, push, deploy, open a PR, or post GitHub comments unless I separately request that exact action.

Keep the application frontend-only with HTML, CSS, Vanilla JavaScript, mock data, and browser-local state. Do not add backend, database, API, real auth, real e-signature, real upload/storage, real notification delivery, real enforcement execution, production audit log, framework migration, or production-readiness claims.

Execute Tasks 1-13 in order with tests first. Correct the root causes rather than adding parallel UI-only state: persist question assignments per audit/question/Inspector; derive the Cabin Inspection execution screen from the selected audit/template; show the Inspector submit success modal immediately without entering a Lead route; open every Lead Preliminary Report by its existing Report ID/Audit ID into the same Findings-first workflow; rebuild Service Provider CAP/Preliminary/Final Report screens from organization-scoped canonical records with the exact requested columns; build the single Finance approval surface; make ED the final plan/report decision authority while preserving GM release and GM intermediate report review; add state-backed Final Report review/preview/template; keep signatures and enforcement referral clearly demo-only and non-automatic.

Run the focused tests after each task, then all js/*.js syntax checks, node --test tests/*.test.js, git diff --check, and the local rendered browser matrix at 1536x864, 1366x768, 1024x768, and 390x844 with an isolated browser profile. Verify no console errors, clipping/overflow, stale IDs/content, privacy regression, CAP/Evidence closure bypass, automatic sanction, real signature claim, or leftover browser/test processes. Update the English/Turkish product/evidence docs, MANIFEST, this plan, the active plan index, and the durable tracker using literal evidence labels.
```
