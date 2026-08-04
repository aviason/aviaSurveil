# Working Scenario Audit Remediation Implementation Plan

> Use `browser:control-in-app-browser` for real-click verification when this plan's browser checks are run.

**Goal:** Resolve all 13 findings from the 20 July 2026 working-scenario audit and restore one truthful, reproducible, Audit-scoped workflow across all eight demo roles.

**Architecture:** Preserve the frontend-only HTML/CSS/Vanilla JavaScript demo and browser-local mock state. Treat `state.audits`, Audit-scoped inspection workspaces, `state.findings`, CAP/Evidence records, planning items, reports, notifications, and Audit Log entries as canonical. Add shared projections for session actor, authority, lifecycle presentation, owner/next action, closure basis, and deterministic demo time; use them in both renderers and mutation handlers so a correct-looking screen cannot conceal a wrong write.

**Tech Stack:** HTML, CSS, Vanilla JavaScript, browser-local mock state, Node `node:test`/`assert`, localhost HTTP serving, Codex in-app Browser.

**Status:** `completed` — all 13 audited findings, automated remediation contracts, silent-state integrity checks, and the original 70-check Browser matrix are `verified locally`. Release is `release pending`; production readiness is `not run`.

### Execution status — 2026-07-20

| Tasks | Status | Evidence / next action |
|---|---|---|
| 1–11 | completed | All 13 WSA contracts pass; there is no standalone WSA-012. |
| 12 | completed | Syntax 7/7, focused 45/45, demo boundary 1/1, full suite 103/103. |
| 13 | completed | In-app Browser real-click matrix completed at 70 PASS, 0 FAIL, 0 blocked; final fresh-tab console reported 0 warning/error entries and task-owned server/browser cleanup passed. |
| 14 | completed | Predecessor remains superseded; bilingual remediation evidence, BUILD_SUMMARY companions, completed-plan index, and tracker are reconciled. |

## Global Constraints

- Keep the frontend-only/demo-only boundary: no backend, database, API, real authentication, real storage, real messaging, production authorization, or framework migration.
- Work on the current branch. Do not create/switch branches, commit, push, open a PR, deploy, or post GitHub comments without separate authorization.
- Preserve unrelated working-tree changes. Inspect focused diffs before editing and handoff.
- Use real UI interactions for browser evidence; do not pass scenarios through direct `state` or `localStorage` mutation.
- Serve `http://127.0.0.1:4173/index.html`; never use `file://`.
- CAP acceptance never closes a Finding. Closure requires accepted Evidence plus verification, or a reason-required authorized path recorded in the Audit Log.
- Keep `Comment to Auditee` separate from `Internal CAA Note`; preserve Evidence versions and organization isolation.
- Keep routine coordination visible only for advance-notice Audits; ad hoc/unannounced advance notice stays withheld.
- Keep English canonical docs and Turkish companions for stakeholder evidence.
- Use literal evidence labels: `verified locally`, `not run`, `blocked`, `candidate-only`, `release pending`, `production-ready`. Never claim `production-ready`.
- Preserve the original audit reports as evidence; record remediation in new evidence files.

## Objective And Finding Map

| Finding | Severity | Required outcome |
|---|---|---|
| WSA-001 | High | Administration opens its authorized preview or is explicitly disabled; no silent no-op. |
| WSA-002 | High | Preparation never says approved/released before approval completion and GM release. |
| WSA-003 | Medium | Mobile summaries are hidden on desktop; one primary decision action per surface. |
| WSA-004 | High | Session identity, assigned Lead, mutation authority, and logged actor are one canonical user. |
| WSA-005 | Blocking | Lead Assigned Audits includes canonical/materialized Audits and routine coordination reaches Auditee. |
| WSA-006 | High | Inspector mutations enforce canonical user/team/question scope inside handlers. |
| WSA-007 | Medium | Result selection and `Create Potential Finding` are separate, truthful actions. |
| WSA-008 | Medium | Observation starts with CAP/Evidence off and no Due Date unless explicitly configured. |
| WSA-009 | High | Accepted CAP has one status/owner/next-action tuple across roles. |
| WSA-010 | High | Authorized closure never renders evidence-verified wording or fictional completed steps. |
| WSA-011 | Medium | Raw status keys are humanized and counters include returned records. |
| WSA-013 | Medium | Assignment search and visible results update together immediately. |
| WSA-014 | Medium | New lifecycle timestamps use the deterministic demo clock. |

There is no standalone WSA-012. The screenshot carrying that number is supporting evidence for Blocking WSA-005, exactly as recorded by the audit.

## Inputs And Source Of Truth

Before implementation, reread:

- `AGENTS.md`
- `docs/demo-evidence/WORKING_SCENARIO_AUDIT_2026-07-20.md`
- `docs/demo-evidence/WORKING_SCENARIO_AUDIT_2026-07-20.turkce.md`
- `docs/product-specs/workflows/MASTER_WORKFLOW.md`
- `docs/product-specs/workflows/SURVEILLANCE_PLANNING_WORKFLOW.md`
- `docs/product-specs/workflows/AUDIT_CHECKLIST_WORKFLOW.md`
- `docs/product-specs/workflows/FINDING_CAP_EVIDENCE_WORKFLOW.md`
- `docs/product-specs/workflows/REMINDERS_AND_ESCALATION_WORKFLOW.md`
- `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md`
- `docs/product-specs/data-and-rules/CONCEPTUAL_DATA_MODEL.md`
- `docs/product-specs/modules/AUDITEE_PORTAL.md`
- `docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.md`
- `docs/product-specs/screen-specs/DEPARTMENT_MANAGER_WORKSPACES.md`
- `docs/product-specs/ux-plan/NAVIGATION_AND_INFORMATION_ARCHITECTURE.md`
- `docs/demo-evidence/BUILD_SUMMARY.md`

## Scope

**In scope:** canonical actor/authority, eight role entries, planning/preparation truth, Lead Audit projection, routine coordination reachability, checklist/Potential Finding semantics, CAP/closure projections, responsive actions, human labels, reactive search, deterministic timestamps, focused/full tests, real-click evidence, and plan/index/tracker reconciliation.

**Out of scope:** production auth/security/storage/audit logs/notifications/signatures, new modules, broad redesign, backend/framework work, rewriting the original audit, Git publication, deployment, and release work.

## Assumptions

- `USR-AYLIN` is the canonical Inspector session and `USR-CANER` is the canonical Lead session; `Ahmed Ali` and `John Lead Inspector` are stale presentation identities.
- An unassigned question is editable only by an Inspector on that Audit team; a question assigned to another Inspector is read-only.
- Lead Assigned Audits is a projection of canonical `state.audits`; synthetic `AUD-2025-*` aliases are not operational records.
- `DEMO_TODAY` remains `2026-06-15`; new writes may use deterministic sequence time but never the host date.
- Existing passing privacy, report approval, reminder, Evidence-version, ad hoc, and unannounced behavior remains a regression gate.

## Ownership Boundaries

- Inspector: only canonical Audit/team/question scope; creates Potential Findings, not real Findings.
- Lead: only assigned plans/Audits; assigns questions, reviews Potential Findings, sends routine coordination.
- Department Manager/GM/Finance/Executive Director: only documented approval stage actions.
- Auditee: only matching organization records and CAA-visible comments; never internal notes.
- Administration: read-only preview unless an existing visible control is explicitly enabled by contract.

---

## Phase 1 — Lock The Audit Into Tests

### Task 1: Add a focused failing regression suite

**Files:** Create `tests/working-scenario-audit-remediation.test.js`; reuse `scenario-integrity-regression.test.js`, `browser-scenario-contract-smoke.test.js`, `inspection-coordination-smoke.test.js`, `lead-inspector-nav-smoke.test.js`, and `login-experience-smoke.test.js`.

- [x] Use the existing `vm` + stub DOM harness and load all runtime modules through `app.js`.
- [x] Add one named test per WSA ID. Assertions must cover Admin entry; joint approval/preparation label; desktop/mobile action counts; actor/assigned-user equality; canonical/materialized Lead rows; byte-for-byte no-mutation authority failures; explicit Potential Finding creation; rendered Observation defaults; cross-role `CAB-2026-012` tuple; authorized closure stepper/banner; human status/counters; reactive search; and 15 Jun timestamps.
- [x] For WSA-005, drive Finance → GM → ED → GM release → Department accept → Lead assign → Lead proposal → Department confirm/materialize from `freshState()`. Do not insert an Audit directly.
- [x] For WSA-006, snapshot the target workspace, Potential Findings, Findings, and Audit Log before unauthorized actions and assert exact equality afterward.
- [x] For WSA-010, require `Authorized closure (audit-logged)`, forbid `after evidence acceptance`, and forbid completed CAP/Evidence steps when records do not exist.

Run: `node --test tests/working-scenario-audit-remediation.test.js`

Expected before implementation: traceable failures for the audited WSA IDs. Preserve this red baseline; do not weaken assertions.

---

## Phase 2 — Identity, Entry, And Authority

### Task 2: Introduce one canonical session actor

**Files:** Modify `js/data.js`, `js/helpers.js`, `js/planning.js`, `js/app.js`, `js/views.js`, `tests/login-experience-smoke.test.js`, and the new focused test.

- [x] Add `userId` to `ROLES`: `USR-AYLIN`, `USR-CANER`, `USR-MANAGER-MEHMET`, `USR-OKAN`, `USR-DERYA`, `USR-UFUK`, `USR-FLY-NAMIBIA-QM`, `USR-ADMIN`.
- [x] Add and reuse these helpers:

```js
function currentSessionUser(target) {
  var role = ROLES[target.role];
  if (!role || !role.userId) return null;
  return (target.users || []).filter(function (user) {
    return user.id === role.userId && user.roleKey === target.role && user.status === 'Active';
  })[0] || null;
}

function currentSessionActor(target) {
  var user = currentSessionUser(target);
  if (!user) throw new Error('Active session user is required for ' + target.role + '.');
  return { userId: user.id, name: user.name, role: target.role, organizationId: target.role === 'auditee' ? ROLES.auditee.org : '' };
}
```

- [x] Replace hard-coded `ROLES[state.role].user`, `planningLeadSessionName`, and Inspector defaults in top bars, mutations, notifications, and logs with the canonical actor.
- [x] Store `leadInspectorUserId` plus display name on planning/Audit records. Migrate only unambiguous `Caner Yildiz` records to `USR-CANER`; log/discard ambiguous mappings.
- [x] Assert the visible/logged Lead is Caner and Inspector is Aylin.

Run: `node --test tests/login-experience-smoke.test.js tests/working-scenario-audit-remediation.test.js`

### Task 3: Make Administration entry explicit

**Files:** Modify `js/app.js`, `js/views.js`, `tests/login-experience-smoke.test.js`, and the focused test.

- [x] Add `ADMIN_ALLOWED_VIEWS` for `templates`, `template-preview`, `users`, `settings`, and `profile` and use it in `roleCanOpenView`.
- [x] Test `handleAction('role', data-role=admin)`, not direct state assignment; require role `admin`, view `templates`, and visible Administration/Admin Preview content.
- [x] Keep unavailable Admin mutations visibly disabled with explanatory copy.

Run: `node --test tests/login-experience-smoke.test.js tests/working-scenario-audit-remediation.test.js`

### Task 4: Enforce Inspector and Lead guards inside mutations

**Files:** Modify `js/inspection.js`, `js/planning.js`, `js/app.js`, `js/views.js`, `tests/inspection-execution-smoke.test.js`, `tests/lead-inspector-nav-smoke.test.js`, and the focused test.

- [x] Add pure `inspectionMutationAuthority(target, auditId, questionId, actor)` returning `{ allowed, reason }`; require exact role, Audit-team membership, question assignment, and submitted/reopen stage.
- [x] Apply it before status/comment/file, Save Draft, Complete Sections, Submit, Potential Finding, and Reopen writes. Unauthorized paths produce no partial state.
- [x] Render the same reason in disabled/read-only UI.
- [x] Require assigned `leadInspectorUserId` for Lead proposal, question assignment, release, and coordination actions.

Run: `node --test tests/inspection-execution-smoke.test.js tests/lead-inspector-nav-smoke.test.js tests/working-scenario-audit-remediation.test.js`

Expected: WSA-001, WSA-004, and WSA-006 pass.

---

## Phase 3 — Planning And Coordination Reachability

### Task 5: Derive preparation presentation from approval plus release

**Files:** Modify `js/planning.js`, `js/views.js`, `tests/planning-release-smoke.test.js`, `tests/planning-workspace-smoke.test.js`, and the focused test.

- [x] Add `planningPreparationPresentation(item)`. Before approval completion return `Awaiting approval` with the approval owner; after approval but before GM release return `Approved — Awaiting GM Release`; otherwise use the preparation status.
- [x] Use it in Planning, Finance, GM, ED, Department Manager, counters, and histories. Returned approvals must say `Returned for revision`/`Awaiting renewed approval`, never `Approved`.
- [x] Keep mutations blocked until approval and release preconditions truly pass.

Run: `node --test tests/planning-release-smoke.test.js tests/planning-workspace-smoke.test.js tests/working-scenario-audit-remediation.test.js`

### Task 6: Replace the synthetic Lead table with canonical Audits

**Files:** Modify `js/views.js`, `js/work-items.js`, `js/app.js`, `tests/lead-inspector-workspace-smoke.test.js`, `tests/lead-inspector-nav-smoke.test.js`, `tests/inspection-coordination-smoke.test.js`, and the focused test.

- [x] Rewrite `leadAssignedAuditRows()` from `state.audits` filtered by canonical Lead ID. Preserve exact Audit ID in row ID and route; remove operational `AUD-2025-*` aliases.
- [x] Derive organization, domain/type, dates, status, progress, checklist, notice policy, and coordination readiness from canonical state; compute KPIs instead of fixed `18/9/4/3/2`.
- [x] Make table/mobile actions navigate with the exact Audit ID.
- [x] For `noticePolicy === 'advance'`, expose one `Send Coordination Package` action after assignment release and create one Audit/org-scoped Auditee request. For `withheld`, expose no advance request.
- [x] Change coordination tests to enter through Lead Assigned Audits; direct `state.view = 'lead-assignment'` is not reachability evidence.

Run: `node --test tests/lead-inspector-workspace-smoke.test.js tests/lead-inspector-nav-smoke.test.js tests/inspection-coordination-smoke.test.js tests/working-scenario-audit-remediation.test.js`

Expected: WSA-002 and Blocking WSA-005 pass; `AUD-2026-001` and materialized `AUD-2026-009` are reachable and routine Auditee date response works.

---

## Phase 4 — Potential Finding, CAP, And Closure Truth

### Task 7: Separate result selection from Potential Finding creation

**Files:** Modify `js/app.js`, `js/views.js`, `js/inspection.js`, `tests/inspection-execution-smoke.test.js`, and the focused test.

- [x] Stop result/comment handlers from creating Potential Findings as a side effect.
- [x] Add explicit `Create Potential Finding`, enabled only for Non-Compliant/Observation plus required comment.
- [x] Route it through one Audit/question-scoped handler; repeated clicks open/reuse the existing record, never duplicate.
- [x] Keep Lead convert/return/dismiss severity and reason gates.

Run: `node --test tests/inspection-execution-smoke.test.js tests/working-scenario-audit-remediation.test.js`

### Task 8: Initialize Observation from canonical defaults

**Files:** Modify `js/views.js`, `js/app.js`, `js/inspection.js`, `tests/inspection-execution-smoke.test.js`, `tests/scenario-integrity-regression.test.js`, and the focused test.

- [x] Observation opens as severity `0`; Non-Compliant opens with severity blank.
- [x] Initialize CAP/Evidence/Due Date via `findingRequirementDefaults`; remove hard-coded checked fields and `2026-07-15`.
- [x] Update the three values atomically on severity change and revalidate at conversion.

Run: `node --test tests/inspection-execution-smoke.test.js tests/scenario-integrity-regression.test.js tests/working-scenario-audit-remediation.test.js`

### Task 9: Use one canonical Finding work-state projection

**Files:** Modify `js/work-items.js`, `js/views.js`, `js/manager-workspaces.js`, `tests/manager-cap-monitoring-smoke.test.js`, `tests/service-provider-portal-smoke.test.js`, `tests/browser-scenario-contract-smoke.test.js`, and the focused test.

- [x] Add `findingWorkState(finding)` returning `{ statusKey, statusLabel, ownerRole, ownerLabel, nextAction, dueDate, closureBasis }` from canonical Finding/CAP/Evidence state.
- [x] Use it in Inspector, Lead, manager, dashboard/work items, Auditee, reminders, and reports.
- [x] Require `CAB-2026-012` everywhere to be `EVIDENCE_REQUIRED`, `CAP Accepted — Evidence Required`, owner `Auditee`, next `Upload evidence`; Finding remains open.

Run: `node --test tests/manager-cap-monitoring-smoke.test.js tests/service-provider-portal-smoke.test.js tests/browser-scenario-contract-smoke.test.js tests/working-scenario-audit-remediation.test.js`

### Task 10: Project lifecycle steps from the actual closure basis

**Files:** Modify `js/work-items.js`, `js/views.js`, `js/reports.js`, `tests/department-manager-findings-smoke.test.js`, `tests/scenario-integrity-regression.test.js`, and the focused test.

- [x] Add `findingLifecycleProjection(finding)` using actual CAP submission/acceptance, Evidence versions/review, and closure type—not `status === 'CLOSED'` alone.
- [x] Evidence-verified closure completes all steps only with accepted Evidence and verification.
- [x] Authorized closure shows basis/reason/actor/date, completes Finding-issued and Closed, and marks unperformed CAP/Evidence as not required/incomplete.
- [x] Remove unconditional `after evidence acceptance` copy; keep blank authorized reason no-op and public/internal notes separate.

Run: `node --test tests/department-manager-findings-smoke.test.js tests/scenario-integrity-regression.test.js tests/working-scenario-audit-remediation.test.js`

Expected: WSA-007, WSA-008, WSA-009, and WSA-010 pass.

---

## Phase 5 — Labels, Responsive Actions, Search, And Demo Time

### Task 11: Fix human labels, action duplication, search, and timestamps

**Files:** Modify `js/approval.js`, `js/planning.js`, `js/reports.js`, `js/helpers.js`, `js/app.js`, `js/inspection.js`, `js/views.js`, `css/styles.css`; update `manager-reports-approval-smoke.test.js`, `lead-inspector-workspace-smoke.test.js`, `premium-ui-remediation-smoke.test.js`, `scenario-integrity-regression.test.js`, and the focused test.

- [x] Map `returned_to_lead`, `released_to_department`, `accepted_by_department`, `lead_inspector_assigned`, `team_schedule_proposed`, and `ready_for_execution` to human labels in tables, histories, counters, and search. Returned reports must be counted in an explicit actionable category.
- [x] Keep `.mobile-decision-summary--mobile-only` hidden at desktop despite later selectors. Remove accidental base-class use on desktop commands. At 1440×900 render one Finance `Approve Budget` and one Auditee `Respond`; at 390×844 render one reachable primary action with no horizontal overflow.
- [x] On `inspector-assignment-query` input, synchronize results immediately and preserve focus/caret (or update only the results region). Clear/blur/navigation must never show stale query/results.
- [x] Add `demoNowIso()`/deterministic sequence time backed by `DEMO_TODAY`. Replace lifecycle `new Date().toISOString()` writes and host-time `logTimestamp()`; preserve explicit historical seed dates.

Run:

```bash
node --test tests/manager-reports-approval-smoke.test.js tests/lead-inspector-workspace-smoke.test.js tests/premium-ui-remediation-smoke.test.js tests/scenario-integrity-regression.test.js tests/working-scenario-audit-remediation.test.js
```

Expected: WSA-003, WSA-011, WSA-013, and WSA-014 pass.

---

## Phase 6 — Full Verification And Evidence

### Task 12: Run automated gates

- [x] Run `node --check` on every modified JavaScript file.
- [x] Run the focused gate containing the new test plus every test named in Tasks 2–11.
- [x] Run `node --test tests/demo-boundary-smoke.test.js`.
- [x] Run `node --test tests/*.test.js`.
- [x] Record actual pass counts; never reuse the superseded `88/88` claim.

Expected: zero failures.

### Task 13: Re-run the complete 70-check audit with real clicks

**Files:** Create `docs/demo-evidence/WORKING_SCENARIO_REMEDIATION_2026-07-20.md` and `.turkce.md`; update `BUILD_SUMMARY.md` and its Turkish companion only after reproduced evidence. Store screenshots/console/cleanup under `/private/tmp/aviasurveil360-working-scenario-remediation-20260720`.

- [x] Start `python3 -m http.server 4173 --bind 127.0.0.1` from the project root and use the in-app Browser.
- [x] Reset between independent scenarios and preserve state only within deliberate lifecycle chains.
- [x] Rerun the original 70 checks: eight roles; complete planning/materialization; routine/ad hoc/unannounced; five checklist packages and Audit isolation; draft/submit/reopen; assignment authority; Potential Finding decisions; CAP/Evidence/closure; reminders/dashboard/work items; Preliminary/Final approvals; privacy/internal notes/cross-org isolation; all visible controls; desktop and mobile 390×844 navigation/responsiveness.
- [x] For silent-state checks, record exact Audit/Finding/org before and after and prove only the intended record changed.
- [x] Capture evidence per WSA ID plus recovered routine coordination and closure-basis distinction. Check console after critical chains and at end.
- [x] Stop the server and clean task-owned Browser/Chrome/helper/headless/webdriver processes without touching the everyday profile; save cleanup evidence.
- [x] Report each WSA as `verified locally`, `blocked`, or `not run` and include the full PASS/FAIL/blocked matrix. Update BUILD_SUMMARY only with reproduced claims; release stays `release pending`.

Expected: 70 PASS, 0 FAIL, 0 blocked. Otherwise keep the plan `active` and record the exact next unresolved finding.

### Task 14: Reconcile lifecycle records

**Files:** Update this plan, `docs/exec-plans/index.md`, and `docs/exec-plans/tech-debt-tracker.md`.

- [x] Set this plan `completed` only after automated and complete browser gates pass and the evidence/index/tracker records agree.
- [x] Keep the predecessor Browser Scenario Integrity plan `superseded`.
- [x] Close the new tracker note only when all 13 findings are `verified locally`; otherwise keep one concrete next todo.
- [x] Run:

```bash
rg -n "WORKING_SCENARIO_(AUDIT|REMEDIATION)_2026-07-20|working-scenario-audit-remediation" docs/exec-plans docs/demo-evidence
git diff --check
git status --short
```

## Verification Exit Criteria

1. All 13 findings have automated contracts and fresh `verified locally` browser evidence.
2. The complete original matrix reports 70 PASS, 0 FAIL, 0 blocked.
3. Canonical and materialized Audits are reachable through Lead and routine coordination reaches Auditee.
4. Unauthorized Inspector/Lead actions cause no partial writes; displayed and logged actor IDs match.
5. `CAB-2026-012` has one canonical work-state tuple across roles.
6. Authorized and evidence-verified closure remain factually distinct.
7. Desktop/mobile action count, human labels, search, and deterministic timestamps pass.
8. Full suite, console, screenshots, server/browser cleanup, plan/index/tracker, and bilingual evidence agree.
9. `production-ready` is not claimed.

## Risks And Controls

| Risk | Control |
|---|---|
| Identity cleanup breaks seeded labels/tests. | Add stable IDs first; migrate projections; update only canonical-identity assertions. |
| Canonical Lead rows change counts/screens broadly. | Derive KPIs from rows and rerun full desktop/mobile suite. |
| Strong guards block valid unassigned work. | Test mine/other/unassigned/non-team separately. |
| Search rerender loses caret. | Preserve selection or update only result region; test typing/clear/blur/nav. |
| Closure projection reintroduces CAP-as-closure. | Derive steps from actual records and assert accepted CAP remains open. |
| Demo clock rewrites history. | Apply only to new writes; preserve explicit seed timestamps. |
| Earlier evidence hides residual defects. | Keep predecessor `superseded`; accept only fresh counts/evidence. |

## Dependencies

Local Node, Python 3 or equivalent static server, Codex in-app Browser, and existing repository docs/evidence. No network, external service, or production credential is required.

## Execution Prompt

```text
/goal Implement the AviaSurveil360 Working Scenario Audit Remediation plan completely, fixing all 13 audited findings and verifying the entire 70-check workflow matrix locally.

Repository: /Users/marlonjd/Developer/web/aviaSurveil360
Canonical plan: docs/exec-plans/completed/2026-07-20-working-scenario-audit-remediation-plan.md
Canonical audit: docs/demo-evidence/WORKING_SCENARIO_AUDIT_2026-07-20.md
Turkish audit companion: docs/demo-evidence/WORKING_SCENARIO_AUDIT_2026-07-20.turkce.md

Execute the plan task-by-task. Use browser:control-in-app-browser for localhost real-click verification. Start by rereading repo-local AGENTS.md and every source-of-truth file named by the plan. Add the focused WSA regression contracts and implement in plan order. Do not weaken tests to preserve current defects.

Preserve the frontend-only HTML/CSS/Vanilla JavaScript and browser-local mock-state boundary. Do not add a backend, API, database, real auth, storage, messaging, framework migration, or production claim. Work on the current branch; do not create/switch branches, commit, push, open a PR, deploy, or post GitHub comments. Preserve unrelated working-tree changes.

Completion requires all 13 findings verified locally plus a fresh rerun of the original 70 checks with 70 PASS, 0 FAIL, and 0 blocked through real clicks at http://127.0.0.1:4173/index.html. Do not use file:// or direct state/localStorage mutation as browser evidence. Capture screenshots and console/cleanup evidence under /private/tmp/aviasurveil360-working-scenario-remediation-20260720. Create canonical English and Turkish remediation evidence, update BUILD_SUMMARY companions only with reproduced claims, and reconcile the active plan, docs/exec-plans/index.md, and docs/exec-plans/tech-debt-tracker.md. Use literal evidence labels verified locally, not run, blocked, candidate-only, release pending, and production-ready; never claim production-ready.

Do not stop at automated green tests. Explicitly verify silent-state integrity by proving each action changes only the intended Audit, Finding, CAP, Evidence, report, or organization record. If any check remains failing or blocked, keep the plan active, record exact evidence and the next concrete todo, and do not describe remediation as complete.
```
