# CAA Governance Workflow And Multi-Role Expansion Plan

- **Date:** 2026-06-28
- **Status:** ready-for-verification
- **Build mode:** Frontend-only clickable demo (HTML + CSS + Vanilla JS, mock
  data, client-side state). No backend/DB/API/auth, per `AGENTS.md`.
- **Related plans:**
  - `2026-06-14-aviasurveil-demo-only-prototype-plan.md` (base demo)
  - `2026-06-23-ncaa-platform-v2-and-mvp-plan.md` (regulatory intelligence / AI
    / offline / production track — a **different axis**; this plan does not
    replace it)
- **Decisions locked with stakeholder (2026-06-28):**
  1. Build mode = expand the frontend-only demo (not production MVP yet).
  2. Primary module focus = **Checklist Management**, but a small Planning
     Approval vertical slice must come first to prove the stakeholder's
     `Department Manager → GM → Finance → ED` chain.
  3. Role model = **all seven roles as distinct switchable actors**.

**Current implementation note (2026-06-29):** Phases 0-4 are implemented in the
frontend-only demo and are `verified locally` with syntax checks, deterministic
Node smoke checks, desktop browser QA, and mobile Planning Approval content
visual QA. The former mobile Planning Approval blocker is closed in
`docs/exec-plans/completed/2026-06-29-governance-browser-qa-mobile-blocker.md`.
Production-readiness is not claimed. Next step: stakeholder review/sign-off
before moving this plan to `completed/`.

---

## Objective

Add the **internal CAA governance and approval layer** described in the six
stakeholder documents to the existing AviaSurveil360 demo. Today the demo
implements only the middle of the lifecycle (`Finding → CAP → Evidence → CAA
Review → Closure`). These documents wrap a full end-to-end audit lifecycle
around it:

```
Planning Item (DM)
  → GM Review → Finance Review → ED Approval → Approved
  → GM Releases to Department → DM Accepts → DM Assigns Lead Inspector
  → Lead Inspector proposes team/dates/resources → DM Confirms
  → Audit Assignment Package → Ready for Execution
  → Inspectors run checklists → Lead Inspector reviews
  → Potential Finding → Convert to Finding
      → [ Finding → CAP → Evidence → CAA Review → Closure ]   (existing demo)
  → Generate Preliminary Report
  → DM Review → GM Review → ED Approval → Final Report → Audit Closed
```

Deliver this as a phased expansion of the static prototype. Build the reusable
approval chain against one thin Planning Approval slice first, then continue
with Checklist Management.

## Source Inputs

Six stakeholder `.docx` documents (Turkish narrative + English UI labels),
provided 2026-06-28:

1. **Checklist Management Role** — Question Bank, Checklist Builder, Version
   History, Approval Queue, checklist lifecycle. Owned by Department Manager.
2. **Planlama modülü** — Planning module: Planning Board, Planning Item,
   Planning Approvals, Released Audits / Audit Preparation; `DM → GM → Finance →
   ED` approval chain; budget/finance logic; assignment package.
3. **Department Manager ekranı** — DM dashboard (KPI cards, planning items,
   schedule, notifications), Trello-style planning board.
4. **Inspector and lead inspector ekranı** — full execution workflow; Inspector
   vs Lead Inspector role split; per-role dashboards and screens.
5. **checklist sayfası** — Inspector checklist runner (4 results, mandatory
   comment, evidence, Potential Finding) and Lead Inspector review (Approve /
   Return / Convert to Finding).
6. **prelimenary and final report ekranı ve süreci** — Report module:
   Preliminary → Final; `Lead → DM → GM → ED` approval chain; progress bar;
   return-for-revision; enforcement recommendation.

Original `.docx` files are stakeholder inputs and are **not** modified by this
plan.

---

## Scope

### In Scope

- Expand the demo role model from 4 to **7 distinct switchable roles**:
  Inspector, Lead Inspector, Department Manager, GM, Finance, Executive
  Director, Auditee (Admin Preview retained as a configuration role).
- Build a **reusable approval-chain primitive** (stages, current position,
  Approve / Return / Reject, mandatory return reason, approval history, progress
  bar) and reuse it in Checklist Approval, Planning Approval, and Report
  Approval.
- **Phase 0B:** a thin **Planning Approval vertical slice** proving the
  `Department Manager → GM → Finance → Executive Director` flow before the
  larger Checklist module starts.
- **Phase 1A:** Checklist approval shell (Department Manager → GM queue over the
  same primitive).
- **Phase 1B:** Checklist Management authoring screens (Question Bank +
  Checklist Builder).
- **Phase 1C:** Checklist versioning and publish/archive behavior.
- **Phase 2:** Inspection execution with Inspector / Lead Inspector split,
  4-result checklist runner, Potential Finding → Convert to Finding.
- **Phase 3:** Planning module and `DM → GM → Finance → ED` approval chain,
  budget/finance loop, Released Audits / Audit Preparation, assignment package.
- **Phase 4:** Report module (Preliminary → Final), `Lead → DM → GM → ED`
  approval chain, enforcement **recommendation**, mock final-report generation.
- Preserve all existing demo behavior: Finding/CAP/Evidence/Closure lifecycle,
  auditee isolation, regulatory trace, risk, AI assistant, USOAP, SSP/NASP,
  offline outbox.

### Out Of Scope (explicit)

- No backend, database, API, server, real authentication, real authorization
  enforcement, real file upload/storage, real e-signature, real email/SMS, or
  framework migration. All approvals, signatures, notifications, budgets, and
  documents are **mock/demo only**.
- No automatic legal, enforcement, certificate suspension, or closure decision.
  Enforcement is a **recommendation** field only and is always human-authorized.
- No real finance/accounting integration or real budget figures. Budget amounts
  are illustrative demo data.
- No production-grade audit trail, security model, or data retention.
- This plan does not execute the production MVP track from the `2026-06-23`
  plan; it only extends the clickable demo.

## Assumptions

- The static architecture stays: `index.html` + `css/styles.css` +
  `js/{data,helpers,views,app}.js`, role-driven `NAV` + `routeView()` switch,
  `localStorage` demo persistence behind the existing boundary in `js/data.js`.
- `DEMO_TODAY = 2026-06-15` remains the deterministic "today".
- Regulatory references use existing careful language (`regulatory reference`,
  `configured rule`, `finding basis`, `expected evidence`). Document examples
  using `NAMCARS` clause numbers map onto the existing mock regulatory library.
- The existing AI-draft governance pattern (Accept / Edit / Reject, "AI-generated
  draft — requires authorized review") is reused for the report Executive
  Summary; no new AI capability is implied.
- Persona mapping (demo only):
  | Role | Demo persona |
  |---|---|
  | Inspector | Aylin Sezer, Mehmet Aydin |
  | Lead Inspector | Caner Yildiz (already labelled "Lead Inspector" in `SEED_USERS`) |
  | Department Manager | Selin Demir (today's "CAA Manager") |
  | GM | new persona (e.g. "Okan Demir, GM") |
  | Finance | new persona (e.g. "Derya Acar, Finance Review") |
  | Executive Director | new persona (e.g. "Ufuk Aslan, Executive Director") |
  | Auditee | James Carter (Airline XYZ) |

---

## Architecture: the reusable approval-chain primitive

Three modules need the same governed approval pattern (Checklist Approval Queue,
Planning Approval, Report Approval). Build it **once** in Phase 0 and reuse it.

**Data shape (per approvable record):**

```js
approval: {
  chain: [
    { role: 'department_manager', label: 'Department Manager', returnToRole: null },
    { role: 'gm', label: 'GM Review', returnToRole: 'department_manager' },
    { role: 'finance', label: 'Finance Review', returnToRole: 'gm',
      notApprovedReturnToRole: 'gm' },
    { role: 'executive_director', label: 'ED Approval', returnToRole: 'gm' }
  ],
  currentIndex: 1,            // whose desk it is on now
  outcome: null,              // null | 'approved' | 'rejected'
  returnPolicy: 'configured_role', // configured_role | previous_stage | origin
  history: [                  // immutable, append-only
    { actor: 'Selin Demir', role: 'department_manager',
      action: 'submitted', date: '2026-06-15 10:30', comment: '' },
    { actor: 'Okan Demir', role: 'gm',
      action: 'returned', date: '2026-06-16 09:00',
      comment: 'Please clarify Finding 2.' }   // reason mandatory on return
  ]
}
```

**Behaviors:**

- `advance(record, actor, comment?)` — moves `currentIndex` forward, logs
  `approved`/`submitted`; at the last stage sets `outcome='approved'`.
- `returnForRevision(record, actor, reason, options?)` — **reason mandatory**;
  resolves the destination from `options.returnToRole`, the current stage's
  configured `returnToRole`, or the chain `returnPolicy`. It must not blindly
  move one step back unless the record explicitly uses `previous_stage`.
- `notApprove(record, actor, reason)` — planning-specific Finance action;
  **reason mandatory**; returns to GM action without killing the item.
- `reject(record, actor, reason)` — sets `outcome='rejected'`; logs `rejected`.
- Helpers derive **current owner + next action + status label** from the chain
  position (mirrors how `FINDING_STATUS` already drives owner/next-action).

**Shared UI:**

- **Approval Progress Bar** — `Lead ✔ → DM ✔ → GM 🟡 → ED ⚪` stepper.
- **Approval History panel** — actor, action, timestamp, comment.
- **Decision panel** — role-aware buttons (Approve / Return for Revision /
  Reject / Forward), with a required-reason textarea on Return/Reject.

This primitive is the single highest-leverage piece; Phases 1/3/4 consume it.
The first implementation target is Phase 0B Planning Approval, because it uses
the most complex return policy and proves the primitive before Checklist
Management depends on it.

---

## Phase 0 — Foundation (partly implemented; finish before Phase 0B)

**Goal:** make the app multi-role and approval-aware before the first module.

- Expand `ROLES` in `js/data.js` to the seven roles + retain `admin`. Add
  `EXPERIENCE_LABEL`, sidebar `NAV` entries, and `homeView()` for each new role.
- Add the approval-chain primitive (`js/helpers.js` for logic, `js/views.js`
  for the shared progress-bar / history / decision components).
- Extend the demo status vocabulary (`V2_STATUS` and new status maps) for
  checklist lifecycle, planning lifecycle, and report lifecycle.
- Update the login / role-switch screen to list all seven experiences with each
  role's guiding question.
- Add new demo personas to `SEED_USERS`.

**Current implementation note (2026-06-29):** the role registry foundation is
implemented locally: Inspector, Lead Inspector, Department Manager, GM, Finance,
Executive Director, Auditee, and Admin Preview are switchable, and new internal
roles have safe role-home placeholders. The approval-chain primitive is still
pending.

**Verification:** every role is switchable; each lands on a sensible (even if
placeholder) home; no console errors; existing four roles unchanged. Then
approval-chain helpers render progress/history/decision controls without module
specific logic.

## Phase 0B — Planning Approval vertical slice (FIRST BUILD TARGET)

**Goal:** prove the stakeholder's headline flow before building the larger
Checklist Management module:

```
Department Manager submits planning item
  → GM Review
  → Finance Review
  → Executive Director Approval
  → Approved
```

This is intentionally a **thin vertical slice**, not the full Planning module.
It exists to harden the shared approval primitive against the most complex
return rules.

**Screens / view surface:**

1. **Department Manager Planning Approvals** — one seeded Planning Item card and
   detail panel showing purpose, risk justification, budget-required flag,
   requested amount, target month, proposed inspectors, current owner, next
   action, and approval progress.
2. **GM Review** — GM can Approve/Forward, Return to Department Manager with a
   mandatory reason, Reject with a mandatory reason, or Send to Finance Review.
   If `budgetRequired=true`, Finance Review is mandatory before ED.
3. **Finance Review** — Finance can Approve, Approve with Adjustment, or
   Not Approve. `Not Approved` must return the item to **GM action** with a
   mandatory reason; it must not delete or final-reject the item.
4. **Executive Director Approval** — ED can Approve, Approve with Note, Reject,
   or Return to GM with a mandatory reason.

**Data model additions (`js/data.js`):**

```js
SEED_PLANNING_ITEMS = [{
  id: 'PLAN-2026-Q3-OPS',
  title: 'Q3 Flight Operations Surveillance Plan',
  department: 'Flight Operations',
  organization: 'Airline XYZ',
  riskCategory: 'Repeat training-record finding',
  budgetRequired: true,
  requestedBudget: 'USD 12,500',
  targetMonth: '2026-09',
  status: 'submitted_to_gm',
  approval: { chain:[...], currentIndex:1, outcome:null, returnPolicy:'configured_role',
              history:[...] }
}];
```

**Implementation boundary:**

- Use existing static stack and mock data.
- Use the same approval components that Checklist and Report will later reuse.
- Do **not** build the full Trello board, released-audit package, team proposal,
  or execution handoff yet; those remain in Phase 3.
- If `views.js` grows further, prefer plain script splitting such as
  `js/approval.js`, `js/views-planning.js`, and `js/views-checklists.js` loaded
  from `index.html`, still with no framework migration.

**Verification:** DM submits/opens the seeded item → GM sends to Finance →
Finance Not Approved returns to GM with reason → GM resends/forwards → Finance
Approves with Adjustment → ED Approves → item becomes Approved; progress and
history are append-only and accurate for each step.

## Phase 1 — Checklist Management module

Owner role: **Department Manager** (build/edit), with **GM** approval.

Build in three smaller execution slices:

- **Phase 1A — Checklist approval shell:** a seeded checklist version uses the
  same shared primitive with `Department Manager → GM`; GM can Approve / Return
  / Reject with mandatory reasons.
- **Phase 1B — Authoring screens:** Question Bank + Checklist Builder.
- **Phase 1C — Versioning and publishing:** Version History, mandatory Reason
  for Change, `Published (Active)` / archive previous active version.

**Screens (new views in `routeView()`):**

1. **Question Bank** — list (`ID`, `Question`, `Regulation`, `Status`) +
   `+ New Question`. Question Detail form: Basic Information (Title, Text,
   Regulation Reference, Department, Category); Inspector Requirements (`Comment
   Required`, `Evidence Required`, `Allow Potential Finding`); Guidance
   (Inspector Guidance, Example Evidence, Notes). Save / Cancel.
2. **Checklist Builder** — Question Bank panel (left) + Checklist panel (right);
   `+ Add From Question Bank` / `+ Create New Question`; reorderable question
   list (demo: up/down controls, drag optional); Checklist Properties (Name,
   Department, Inspection Type: Routine / Risk Based / Unannounced). Actions:
   Save Draft / Preview / Submit for Approval.
3. **Version History** — checklists are **never edited in place**. Editing an
   `Active` version prompts `Create Version X.Y?`; list shows
   `Version / Status / Created By`. A mandatory **Reason for Change** is captured
   and shown in history.
4. **Approval Queue** — uses the Phase 0 primitive with chain
   `Department Manager → GM`. GM sees Checklist / Changes / Reason / Impact and
   decides Approve / Return / Reject. Approved → **Published (Active)**.

**Checklist lifecycle status:** `Draft → Under Review → Approved →
Published (Active) → Archived` (only one Published version per checklist;
publishing archives the prior active version).

**Data model additions (`js/data.js`):**

```js
SEED_DEPARTMENTS = ['Flight Operations', 'Airworthiness', 'Licensing',
                    'SMS', 'Security', 'Aerodrome', 'ANS/ATM'];

SEED_QUESTION_BANK = [{ id:'QB-001', title, text, regulationRef, department,
  category, commentRequired, evidenceRequired, allowPotentialFinding,
  inspectorGuidance, exampleEvidence, notes, status:'Active' }, ...];

SEED_MANAGED_CHECKLISTS = [{
  id:'CL-FOPS', name:'Flight Operations Surveillance', department, inspectionType,
  versions:[{ version:'2.4', status:'draft', createdBy, questionIds:[...ordered],
              changeReason, approval:{...} }, ...],
  publishedVersion:'2.3'
}];
```

The runnable `SEED_CHECKLIST` / `SEED_TEMPLATE_LIBRARY` stay for back-compat;
Phase 2 consumes the **published** managed checklist.

**Guardrails:** Inspector never edits checklists (read-only active version only);
regulation fields use `regulatory reference` wording; version history is
append-only (no overwrite — aligns with the existing evidence-version principle).

**Verification:** create question → build checklist → submit → GM approves →
publishes as Active and archives prior; editing Active creates a new Draft
version with a mandatory reason; Inspector cannot reach edit controls.

## Phase 2 — Inspection execution + Inspector / Lead Inspector split

**Inspector** (simple operational role):

- Dashboard: My Audits, Assigned / Completed / Pending checklist counts.
- Audit screen: only the inspector's own assigned checklists.
- **Checklist runner (refined):** per question — Result =
  `Compliant / Non-Compliant / Observation / Not Applicable`; **comment
  mandatory** when Non-Compliant or Observation; evidence (mock filenames:
  Photo / Document / Voice Note); **Potential Finding** checkbox + summary only
  (no severity / risk / category — Lead decides). Previous / Save / Next;
  Submit Section.

**Lead Inspector** (review + orchestration role):

- Dashboard: Audit Progress %, Pending Reviews, Reports Waiting.
- Audit Workspace tabs: Overview / Assignments / Review / Reports /
  Communication.
- **Assignments:** assign each checklist to an inspector.
- **Review screen:** open a submitted checklist item → Approve / Return to
  Inspector / **Convert to Finding**. Convert-to-Finding form (Lead only): Title,
  Regulation, **Severity** (Minor/Major/Critical) → creates a Finding that
  enters the **existing** `WAITING_CAP` lifecycle unchanged.
- When all checklists complete: **Generate Preliminary Report** (hands off to
  Phase 4).

**Data model additions:** per-audit `assignments` (checklist→inspector);
checklist responses keyed by audit + item with the 4-result model; a
`potentialFindings` collection (`pending / converted / dismissed`). The existing
`OPS-2026-001` hero finding becomes the output of Convert-to-Finding.

**Guardrails:** Inspector sets no severity; auditee sees nothing here; Internal
CAA Note vs Comment to Auditee separation preserved on the resulting finding.

**Verification:** Inspector answers + raises Potential Finding → Lead reviews →
Convert sets severity → finding appears in the existing finding lifecycle and
dashboards; Return sends only that checklist back.

## Phase 3 — Full Planning module + release / assignment package

Four screens (per the Planning doc):

1. **Planning Board** — list/Trello-style columns: `Draft / Pending GM / Finance
   Review / ED Approval / Approved / Released`. Cards are **not** drag-movable
   (workflow-driven), but stage is visible at a glance. Filters + actions
   (Create / Edit / Send to GM / Withdraw / Duplicate / Export — mock).
2. **Planning Item Detail** — DM input form: Basic Information; Oversight
   Justification (Reason, Risk Category, Trigger Type, Narrative); Budget &
   Resource Estimate (Budget Required?, Amount, Currency, Notes, Travel, HOTAC,
   Inspectors, Duration); Preliminary Scope; Attachments (mock). Actions: Save
   Draft / Submit to GM / Cancel.
3. **Planning Approvals** — expanded role-aware view over the Phase 0 primitive
   and Phase 0B vertical slice:
   - **GM:** Approve as proposed / Modify and continue / Reject / **Send to
     Finance Review** / Send to ED without finance. Rule: **if budget required,
     Finance Review is mandatory.**
   - **Finance:** Approved / Approved with Adjustment / **Not Approved**. On Not
     Approved the item returns to **GM action** (not killed): GM may revise &
     still send to ED, defer, or reduce scope and resend to Finance.
   - **ED:** Approve / Approve with note / Reject / Return to GM.
4. **Released Audits / Audit Preparation** — `GM releases → DM accepts → DM
   appoints Lead Inspector → Lead proposes team/dates/resources → DM confirms →
   System generates Audit Assignment Package → Ready for Execution`. The package
   feeds Phase 2 execution.

**Planning status model:** `Draft → Submitted to GM → Under GM Review → Sent to
Finance Review → Finance Reviewed → Pending ED Approval → Approved / Approved
with Budget Limitation / Rejected → Released to Department → Accepted by
Department → Lead Inspector Assigned → Team & Schedule Confirmed → Ready for
Execution`.

**Roles used here:** Department Manager, GM, Finance, Executive Director, and
Lead Inspector. These roles must already exist from Phase 0.

**Guardrails:** budgets and the assignment package are mock; finance "rejection"
never deletes an item; ED approval does not auto-trigger execution (explicit GM
release required).

**Verification:** DM creates item with budget → routes through GM → Finance
(test the Not-Approved → GM loop) → ED → Approved → GM release → DM accept → Lead
proposal → DM confirm → Ready for Execution; Department Manager dashboard KPI
cards update.

## Phase 4 — Report module + `Lead → DM → GM → ED` approval chain

- **Report screen:** header (Audit, ID, Lead Inspector, Status); left section
  tabs: Overview (system-filled) / Executive Summary (**AI-generated draft —
  requires authorized review**, with Regenerate + Accept/Edit) / Findings (from
  Phase 2) / Observations / Recommendations / Attachments (auto-listed) /
  Approval History. Actions: Save Draft / Preview PDF (mock) / Submit.
- **Approval Progress Bar** at top (Phase 0 primitive).
- **Approval chain** `Lead Inspector → Department Manager → GM → Executive
  Director`. Each level: Approve / **Return for Revision** (mandatory reason,
  returns to the record-specific configured destination). DM decision panel
  includes **Enforcement Recommendation** (Administrative / Warning / Suspension
  / Other + reason) — recommendation only, never automatic.
- **Final Report:** on ED approval the system generates Report Number, Approval
  Date, Approved By, **mock Digital Signature** (clearly labelled demo), Download
  PDF/Word (mock). Final report is **locked / non-editable**; audit → Closed.

**Report status model:** `Draft → Submitted to DM → Returned → Resubmitted →
Submitted to GM → Returned → Resubmitted → Submitted to ED → Approved → Final
Report Generated → Closed`.

**Guardrails:** "Digital Signature" is a mock placeholder, not a real
e-signature; enforcement wording is careful and recommendation-only; final
report immutability is demo-simulated.

**Verification:** generate preliminary → submit up the chain → test a Return at
GM (back to DM with reason) → ED approves → final report generated, locked, audit
closed, manager dashboards update.

---

## End-to-end demo narrative (target after all phases)

1. **DM** raises a thin Planning Approval item (budget required) → **GM** →
   **Finance** → **ED** approve. This proves the shared approval primitive
   early. *(Phase 0B)*
2. **DM** builds a Flight Operations checklist in Checklist Management; **GM**
   approves; it publishes as Active. *(Phase 1A-1C)*
3. **DM** raises the full Planning Item (budget required) → **GM** → **Finance** →
   **ED** approve → **GM** releases to department → **DM** accepts → assigns
   **Lead Inspector** → Lead proposes team/dates → DM confirms → assignment
   package → Ready for Execution. *(Phase 3)*
4. **Inspector** runs the published checklist, marks `Crew training records` as
   Non-Compliant, raises a Potential Finding. *(Phase 2)*
5. **Lead Inspector** reviews and Converts to Finding (sets Major) →
   `OPS-2026-001` enters the existing Finding → CAP → Evidence → Closure flow.
   *(Phase 2 + existing demo)*
6. **Auditee** submits CAP + evidence; Inspector reviews; finding closes only on
   evidence acceptance. *(existing demo, unchanged)*
7. **Lead Inspector** generates the Preliminary Report → **DM** → **GM** →
   **ED** approve → Final Report generated and locked → audit Closed. *(Phase 4)*

---

## Verification Plan

- **Per phase:** open `index.html` directly (or `python3 -m http.server 4360`);
  click through each new screen; **no console errors**; confirm the phase's
  workflow end-to-end as listed above.
- **Role isolation:** Auditee sees none of the new internal governance screens,
  approval chains, budgets, or internal notes. Inspector cannot edit checklists
  or set severity.
- **Approval primitive:** Return-for-revision requires a reason at every level
  and moves to the configured destination (`returnToRole` / `returnPolicy`),
  not always the previous stage; Finance `Not Approved` returns to GM action;
  approval history is append-only; progress bar reflects true position.
- **Immutability:** checklist versions and the final report cannot be edited
  once Published / Final.
- **Regression:** existing Finding/CAP/Evidence/Closure flow, dashboards,
  regulatory trace, AI, USOAP, SSP, and offline outbox still work.
- **Copy check:** terminology stays consistent (`Finding`, `CAP`, `Evidence`,
  `Due Date`, `Overdue`, `Auditee`, `Internal CAA Note`, `Comment to Auditee`);
  enforcement and signature wording stays careful/mock.
- **Demo truth:** ribbon still states frontend-only / mock; no backend, DB, API,
  auth, real upload, real signature, or framework added.

## Risks

- **Scope size.** Four modules + 3 new roles is large for one pass. Mitigation:
  strict phasing; Phase 0B ships and is verified before Checklist work; Phase
  1A/1B/1C ship before Phase 2 starts.
- **State model growth.** `freshState()` / `mergeDemoState()` must stay
  backward-compatible. Mitigation: bump `DEMO_STATE_VERSION`; add defensive
  defaults in `mergeDemoState` for every new collection.
- **Regulatory/enforcement overreach.** Enforcement recommendation and digital
  signature could read as binding. Mitigation: careful wording + visible mock
  labels per `AGENTS.md`.
- **`views.js` size.** Already ~1,400 lines; four modules add substantial UI.
  Mitigation: keep view functions small and sectioned; prefer plain script
  splits (`js/approval.js`, `js/views-planning.js`, `js/views-checklists.js`)
  once a module becomes non-trivial. No framework migration.
- **Demo coherence.** Long approval chains can bore a sales audience. Mitigation:
  seed records mid-chain so each role has something waiting; keep the hero
  scenario fast.

## Dependencies

- Phase 0 role foundation is partly implemented; the approval primitive remains
  the immediate dependency.
- Phase 0B Planning Approval validates the primitive before Checklist
  Management depends on it.
- Phase 1A/1B/1C produce the published checklist consumed by Phase 2.
- Phase 2 consumes Phase 1's published checklist and feeds Phase 4's report.
- Phase 3 produces the audit/assignment package that Phase 2 executes (demo can
  also run Phase 2 on seeded audits before Phase 3 lands).
- No external/library dependencies; static stack only.

## Ownership Boundaries

- This plan owns the **governance/approval workflow + multi-role** expansion of
  the **demo**.
- It does **not** own the production MVP architecture, regulatory intelligence,
  AI, or offline tracks — those remain in `2026-06-23-ncaa-platform-v2-and-mvp-plan.md`.
- Original `.docx` stakeholder inputs are read-only references.

---

## Erratum — 2026-07-10

Executive Director Final Report approval does not automatically close an audit when any Finding, CAP, Evidence, verification, or authorized closure work remains open. Issuing the report may move the audit to `Follow-up Open`; CAP acceptance is not Finding closure. GM review is intermediate only, and Executive Director alone owns demo issue/mock-sign/lock authority. This erratum supersedes any earlier wording that implied ED/GM approval itself completed audit closure.

Status: **demo-only** contract correction **verified locally** by focused report/authority regressions. Production authority and legal effect are **not run**; **production-readiness not claimed**.

## Execution Prompt

```text
You are working in /Users/marlonjd/Developer/monorepos/avia/apps/surveil.

Task: perform stakeholder review/sign-off for the CAA Governance Workflow And Multi-Role Expansion demo lane.

Read first:
- AGENTS.md
- docs/exec-plans/index.md
- docs/exec-plans/active/2026-06-28-caa-governance-workflow-and-roles-plan.md
- docs/demo-evidence/BUILD_SUMMARY.md
- docs/exec-plans/completed/2026-06-29-governance-browser-qa-mobile-blocker.md
- docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md

Do:
1. Review the verified-local evidence for syntax, Node smoke checks, desktop browser QA, and mobile Planning Approval visual QA.
2. Prepare a compact stakeholder sign-off readout covering the implemented governance flows: Planning Approval, Checklist Approval, Report Approval, Inspector Work Queue/Offline Field, Auditee isolation, and Admin Question Bank.
3. Keep the demo-only boundary explicit: no backend, database, API, real auth, real upload, real AI, real regulatory ingestion, real notification service, production audit-log readiness, or production readiness.
4. If stakeholder/user sign-off is explicitly given, move this plan to `docs/exec-plans/completed/` and update the completed index per AGENTS.md.
5. If new gaps are found, record them in `docs/exec-plans/` and keep the plan `ready-for-verification` or mark it `blocked` as appropriate.

Verification:
- Active plan index and tech-debt tracker match the final status.
- `docs/demo-evidence/BUILD_SUMMARY.md` remains the canonical evidence file.
- Any new blocker is recorded durably instead of only in chat.

Final response:
- State Done / Remaining / Blocked / Verification / Next.
```
