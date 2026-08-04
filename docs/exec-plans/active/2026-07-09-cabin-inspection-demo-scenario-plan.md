# Cabin Inspection Demo Scenario Implementation Plan


**Goal:** Replace the current Flight Operations hero scenario with a Cabin Inspection scenario based on the reviewed master checklist workbook while preserving the AviaSurveil360 frontend-only demo boundary and the Finding -> CAP -> Evidence -> CAA Review -> Closure workflow.

**Architecture:** Keep the existing static HTML, CSS, and Vanilla JavaScript architecture. Transform the workbook content into curated mock seed data in `js/data.js`, update the existing route renderers/controllers in `js/views.js` and `js/app.js`, then update scenario docs, bilingual evidence summaries, and focused smoke tests. The workbook is used as a demo template source, not as a live import, backend data source, or binding regulatory source.

**Tech Stack:** HTML, CSS, Vanilla JavaScript, static mock data, browser-local demo state, direct Node smoke tests, focused rendered browser QA when visible workflows change.

## Global Constraints

- Keep AviaSurveil360 as the canonical product name.
- Keep the executable artifact frontend-only and demo-only.
- Do not add backend, database, API, real authentication, real authorization enforcement, real file upload/storage, real AI service, real regulatory ingestion, real notification service, production audit log, mobile/offline production app, framework migration, e-signature, or deployment.
- Preserve the Finding -> CAP -> Evidence -> CAA Review -> Closure workflow.
- CAP acceptance is not finding closure.
- A finding closes only after evidence acceptance, verification completion, or an authorized closure path that is explicitly audit-logged.
- Separate `Comment to Auditee` from `Internal CAA Note`.
- Keep auditee views limited to their own organization and CAA-visible information.
- Use careful regulatory language: `reference`, `configured check`, `expected evidence`, `finding basis`, and `review result`.
- Treat workbook references such as `ICAO Annex 6` as demo/configured references, not legal advice or verified clause-level obligations.
- Keep source code identifiers, comments, canonical docs, file names where practical, and implementation plans in English.
- Keep Turkish companion docs in sync when stakeholder-facing English docs with an existing `.turkce.md` companion are updated.
- Bump `index.html` CSS/JS asset query tokens whenever CSS or JS behavior changes.

---

## Objective

Create a new primary demo story:

`CAA Manager sees a Cabin Inspection plan -> CAA Inspector opens FlyNamibia Cabin Inspection -> inspector runs a Cabin Inspection checklist -> inspector marks a critical emergency equipment item Non-Compliant -> system creates Finding CAB-2026-001 -> FlyNamibia submits CAP and mock evidence -> inspector reviews evidence -> finding closes only after evidence acceptance -> manager dashboard updates.`

The new scenario should make the product feel more aviation-specific and operational by using visible cabin inspection sections, risk categories, severity, expected evidence, and photo/evidence review patterns derived from the reviewed workbook.

## Source Workbook Profile

Reviewed attachment:

- `AviaSurveil360_Full_Master_Checklist_Risk_Updated.xlsx`
- Workbook profile: one worksheet, range `A1:S127`, 126 checklist rows, no formulas, no dropdown validations, no conditional formatting.
- Columns: `Module`, `Section`, `Sub Section`, `No.`, `Checklist Question`, `Requirement`, `Reference`, `Risk Category`, `Severity`, `Finding Type`, `Evidence Required`, `Photo Required`, `Attachment Required`, `Compliance`, `Inspector Comments`, `Root Cause`, `Corrective Action Required`, `Responsible Person`, `Status`.
- Module: `Cabin Inspection`.
- Sections: `GALLEY` 20, `LAV` 28, `PAX SEAT` 24, `EM EQ` 17, `VID+CREW SEAT` 17, `COCKPIT+CAB GEN COND+EXITS` 20.
- Risk categories: `Cabin Safety` 91, `Emergency Preparedness` 15, `Flight Operations` 20.
- Severity distribution: `Level 2` 121, `Level 1` 5.
- Finding types: `Compliance` 71, `Equipment` 35, `Operational` 20.
- All rows mark `Evidence Required` as `Yes`, `Photo Required` as `Optional`, `Attachment Required` as `No`, `Corrective Action Required` as `Yes`, and `Status` as `Open`.

Preferred hero row:

- Section: `EM EQ`
- Sub Section: `Emergency Equipments`
- Workbook row: 78
- No.: `5`
- Source question: `Is the pbe installed, serviceable and in compliance with applicable requirements?`
- Curated demo question: `Is the PBE installed, serviceable, accessible, and in compliance with configured cabin emergency equipment requirements?`
- Requirement: `Company Cabin Inspection Manual`
- Reference: `ICAO Annex 6`
- Risk category: `Emergency Preparedness`
- Severity: `Level 1 Critical`
- Finding type: `Equipment`
- Expected evidence: `PBE replacement/serviceability record, cabin defect rectification reference, and inspector photo filename`

Backup candidate rows:

- Workbook row 79: first aid oxygen masks, `Level 1`, `Emergency Preparedness`, `Equipment`.
- Workbook row 60: oxygen mask compartment, `Level 1`, `Emergency Preparedness`, `Equipment`.
- Workbook row 98: video-system oxygen compartment, `Level 1`, `Emergency Preparedness`, `Equipment`.
- Workbook row 33: lavatory oxygen compartment, `Level 1`, `Emergency Preparedness`, `Equipment`.

## Scope

In scope:

- Make the Cabin Inspection scenario the primary first-run demo path and login/role-select copy.
- Replace the current hero checklist template with a curated `Cabin Inspection` template sourced from the workbook.
- Keep a small, runnable checklist subset for demo speed, while preserving the full 126-row workbook profile in docs and admin/template metadata.
- Add admin preview visibility for the Cabin Inspection template structure and workbook-derived sections.
- Update finding IDs, labels, lifecycle text, dashboards, reports, regulatory trace mock records, AI assistant mock text, CAP effectiveness mock text, and NASP/SSP references that currently hard-code `OPS-2026-001` or crew training.
- Update tests that assert `q2`, `OPS-2026-001`, `Crew training records incomplete`, or `Flight Operations Audit`.
- Update `docs/product-specs/scenarios/DEMO_SCENARIO_OPERATOR_AUDIT.md` and `.turkce.md`.
- Update `docs/demo-evidence/BUILD_SUMMARY.md` and `.turkce.md` after implementation verification.
- Preserve the original Flight Operations template as a secondary/admin-library sample only if it remains useful and does not confuse the primary demo.

Out of scope:

- Real Excel import in the browser.
- Backend storage, database schema, API ingestion, or admin upload workflow.
- Real regulatory validation of the workbook's references.
- Clause-level ICAO mapping unless a verified regulatory source is separately provided.
- Real file upload or evidence storage; mock evidence continues to display selected file names only.
- New framework, build system, package manager setup, or deployment.
- Rewriting all V2 screens unrelated to the hero scenario.

## Assumptions

- The current active app already uses `SEED_CHECKLIST`, `SEED_QUESTION_BANK`, `SEED_TEMPLATE_LIBRARY`, seeded audits, findings, regulatory traces, and direct route renderers.
- The primary live finding is currently created at runtime and should remain created at runtime, not pre-seeded.
- The demo can use a curated subset of workbook questions for the runner, because a 126-row live checklist would slow the sales/demo path.
- The admin template preview can communicate the 126-row source coverage without rendering every row in the main execution path.
- The auditee is `FlyNamibia` per stakeholder direction; internal IDs may still reuse existing `ORG-XYZ` demo identifiers where lower-risk.
- Demo date context remains deterministic; only scenario labels and due dates change.

## Ownership Boundaries

This plan owns:

- `js/data.js`
- `js/views.js`
- `js/app.js`
- `js/helpers.js` only if shared labels/status helpers need small changes.
- `index.html` only for asset token updates.
- Focused tests under `tests/`.
- `docs/product-specs/scenarios/DEMO_SCENARIO_OPERATOR_AUDIT.md`
- `docs/product-specs/scenarios/DEMO_SCENARIO_OPERATOR_AUDIT.turkce.md`
- `docs/demo-evidence/BUILD_SUMMARY.md`
- `docs/demo-evidence/BUILD_SUMMARY.turkce.md`
- `docs/exec-plans/index.md`
- This plan file.

This plan does not own:

- Production architecture docs beyond demo wording updates.
- Original source/reference materials.
- Branches, commits, pushes, PRs, or GitHub comments.

## File Map

Primary implementation:

- `js/data.js` - new Cabin Inspection checklist seed, question bank rows, template library row, audit seed, finding/report/regulatory trace mocks, AI/CAP/SSP references.
- `js/views.js` - visible copy for role select, template preview, checklist runner, finding modal defaults, report previews, dashboard labels, and any hard-coded Flight Operations/Crew Training strings.
- `js/app.js` - controller defaults for potential finding conversion, live finding creation, hero finding ID, and route copy that currently references `OPS-2026-001`.
- `index.html` - asset query token bump after JS/CSS behavior changes.

Focused tests:

- `tests/inspection-execution-smoke.test.js` - switch hero question from `q2`/`OPS-2026-001` to the Cabin Inspection PBE scenario.
- `tests/checklist-comment-render-smoke.test.js` - switch the comment/render assertion to the new hero question ID.
- `tests/inspector-nav-smoke.test.js` - update expected visible strings where needed.
- `tests/table-first-workbench-smoke.test.js` and `tests/demo-boundary-smoke.test.js` - update only if hero IDs/copy are asserted.
- Add `tests/cabin-inspection-scenario-smoke.test.js` if existing tests cannot clearly cover the full new path.

Documentation:

- `docs/product-specs/scenarios/DEMO_SCENARIO_OPERATOR_AUDIT.md` - replace the old scenario content with the Cabin Inspection scenario.
- `docs/product-specs/scenarios/DEMO_SCENARIO_OPERATOR_AUDIT.turkce.md` - Turkish companion update.
- `docs/demo-evidence/BUILD_SUMMARY.md` and `.turkce.md` - update after implementation and verification only.

## Scenario Data Contract

Use these canonical demo values:

| Field | Value |
|---|---|
| Audit | `2026 Cabin Inspection - FlyNamibia` |
| Audit ID | reuse existing primary audit ID if lower-risk, otherwise `AUD-2026-CAB-001` |
| Checklist | `Cabin Inspection` |
| Template ID | `TPL-CABIN-2026` |
| Hero question ID | `cab-em-eq-pbe` |
| Hero question | `Is the PBE installed, serviceable, accessible, and in compliance with configured cabin emergency equipment requirements?` |
| Finding ID | `CAB-2026-001` |
| Finding title | `PBE not serviceable or not accessible in cabin emergency equipment check` |
| Organization | `FlyNamibia` |
| Severity | `Level 1 Critical` |
| Risk category | `Emergency Preparedness` |
| Finding type | `Equipment` |
| Finding basis | `Checklist item answered Non-Compliant` |
| Requirement | `Company Cabin Inspection Manual` |
| Regulatory reference | `Configured reference: ICAO Annex 6 / Cabin emergency equipment (demo reference)` |
| CAP root cause sample | `Pre-flight cabin equipment serviceability checks did not reconcile the PBE position with the deferred defect list.` |
| CAP corrective action sample | `Replace or service the affected PBE, update the cabin defect record, and confirm serviceability before release.` |
| CAP preventive action sample | `Add a supervisor review of emergency equipment checks and monthly sampling of PBE serviceability records.` |
| Target completion date | `2026-07-15` |
| Mock evidence filename | `FlyNamibia_PBE_Serviceability_Record_CAB-2026-001.pdf` |
| Optional mock photo filename | `PBE_Cabin_Position_Photo.jpg` |

## Phases

### Phase 1 - Baseline And Search Map

- [x] **Step 1: Confirm current working tree**

  Run:

  ```bash
  git status --short
  ```

  Expected: note any unrelated local changes and avoid reverting them.

- [x] **Step 2: Map current hero-scenario hard-codes**

  Run:

  ```bash
  rg -n "OPS-2026-001|Flight Operations Audit|Crew training|crew training|q2|TPL-FOPS-2026|Finding OPS|Training_Record|REG-NAMCARS-OPS|TRACE-OPS" js tests docs/product-specs docs/demo-evidence
  ```

  Expected: every hard-coded old hero string is classified as `replace`, `keep as secondary sample`, or `historical evidence only`.

- [x] **Step 3: Decide the low-risk audit ID strategy**

  Prefer reusing the current primary audit ID if many route handlers assume it. Use the new display labels to show `2026 Cabin Inspection - FlyNamibia`. Create a new audit ID only if route/test references are easy to update in one pass.

  Expected: one audit ID strategy recorded in implementation notes or the final summary.

### Phase 2 - Mock Data Conversion

- [x] **Step 1: Update `SEED_CHECKLIST` in `js/data.js`**

  Replace the primary checklist with `TPL-CABIN-2026`, `Cabin Inspection`, domain `Cabin Safety`, version `v1.0 (2026 demo)`, owner `Cabin Safety Section`.

  Required runnable items:

  | ID | Text | Ref | Evidence |
  |---|---|---|---|
  | `cab-galley-oven` | `Is the oven installed, serviceable, and in compliance with configured cabin inspection requirements?` | `Configured reference: ICAO Annex 6 / Company Cabin Inspection Manual (demo)` | `Galley equipment serviceability record or cabin defect rectification note` |
  | `cab-lav-oxygen-compartment` | `Is the lavatory oxygen compartment installed, serviceable, and in compliance with configured cabin emergency equipment requirements?` | `Configured reference: ICAO Annex 6 / Company Cabin Inspection Manual (demo)` | `Lavatory oxygen compartment serviceability record and inspection note` |
  | `cab-seat-oxygen-mask` | `Is the passenger oxygen mask compartment installed, serviceable, and in compliance with configured cabin emergency equipment requirements?` | `Configured reference: ICAO Annex 6 / Company Cabin Inspection Manual (demo)` | `Passenger seat oxygen mask compartment check record` |
  | `cab-em-eq-pbe` | `Is the PBE installed, serviceable, accessible, and in compliance with configured cabin emergency equipment requirements?` | `Configured reference: ICAO Annex 6 / Cabin emergency equipment (demo reference)` | `PBE replacement/serviceability record, cabin defect rectification reference, and inspector photo filename` |
  | `cab-em-eq-first-aid-oxygen` | `Are first aid oxygen masks installed, serviceable, accessible, and in compliance with configured cabin emergency equipment requirements?` | `Configured reference: ICAO Annex 6 / Cabin emergency equipment (demo reference)` | `First aid oxygen serviceability record and inspection sign-off` |
  | `cab-exit-safety-strap` | `Is the exit safety strap installed, serviceable, and in compliance with configured exit equipment requirements?` | `Configured reference: ICAO Annex 6 / Cabin exits (demo reference)` | `Exit equipment inspection record and rectification note if applicable` |

- [x] **Step 2: Update `SEED_QUESTION_BANK`**

  Add or replace question bank rows so the Cabin Inspection items are visible in admin/checklist management. At minimum, include the six runnable IDs above with:

  - `department: 'Cabin Safety'`
  - `category` mapped to `Galley`, `Emergency Preparedness`, or `Exits`
  - `evidenceRequired: true`
  - `allowPotentialFinding: true`
  - `commentRequired: true` for `cab-em-eq-pbe`
  - `notes` stating `Workbook-derived demo question; wording curated for stakeholder demo.`

- [x] **Step 3: Update template library and managed checklist approvals**

  Make `TPL-CABIN-2026` the published primary template. Keep `TPL-FOPS-2026` only as secondary if needed.

  Expected visible library rows:

  - `Cabin Inspection`, domain `Cabin Safety`, version `v1.0 (2026 demo)`, items `126 source rows / 6 runnable demo rows`, status `Published`.
  - Existing templates may remain as non-primary examples.

- [x] **Step 4: Update audits and organization risk focus**

  Ensure the primary FlyNamibia audit displays:

  - `ref: '2026 Cabin Inspection - FlyNamibia'`
  - `type: 'Cabin Inspection'`
  - `domain: 'Cabin Safety'`
  - `templateId: 'TPL-CABIN-2026'`
  - `location: 'FlyNamibia aircraft cabin / on-site inspection'`
  - manager/dashboard risk focus includes `Emergency equipment serviceability`, `PBE serviceability`, and `Cabin inspection CAP follow-up`.

### Phase 3 - Live Finding And Controller Defaults

- [x] **Step 1: Update finding sequence defaults in `js/app.js`**

  Replace hard-coded `OPS-2026-001` creation defaults with `CAB-2026-001` for the new hero path.

  Required modal defaults:

  - `Finding ID`: `CAB-2026-001`
  - `Finding title`: `PBE not serviceable or not accessible in cabin emergency equipment check`
  - `Severity`: `Level 1 Critical`
  - `Due date`: 30-day target unless existing status logic expects a fixed demo date.
  - `CAP required`: `Yes`

- [x] **Step 2: Update potential finding conversion**

  Any lead-inspector conversion path that currently defaults to `Crew training records incomplete` must default to the PBE finding title and the `cab-em-eq-pbe` checklist item.

- [x] **Step 3: Preserve lifecycle behavior**

  Verify the new `CAB-2026-001` path still transitions:

  ```text
  WAITING_CAP -> CAP_SUBMITTED -> EVIDENCE_REQUIRED -> EVIDENCE_SUBMITTED -> CLOSED
  ```

  CAP acceptance must leave the status at `EVIDENCE_REQUIRED`.

### Phase 4 - View Copy And Admin Preview

- [x] **Step 1: Update role-select and demo intro copy**

  Replace old intro text with:

  ```text
  Demo scenario: a CAA Inspector raises Finding CAB-2026-001 for FlyNamibia from a Cabin Inspection emergency equipment checklist. CAP acceptance does not close the finding; accepted evidence is required.
  ```

- [x] **Step 2: Update checklist runner labels**

  The runner should show section-level context from the workbook:

  - `GALLEY`
  - `LAV`
  - `PAX SEAT`
  - `EM EQ`
  - `VID+CREW SEAT`
  - `COCKPIT+CAB GEN COND+EXITS`

  The live path should visibly guide the user to `EM EQ / PBE`.

- [x] **Step 3: Update template preview**

  Change the admin callout from "only Flight Operations Audit template is openable" to Cabin Inspection.

  Required preview summary:

  ```text
  Source workbook profile: 126 Cabin Inspection rows across 6 sections. This demo runs a curated 6-question subset; the source workbook remains a mock/configured checklist reference, not a live import or legal source.
  ```

- [x] **Step 4: Update reports, AI assistant, regulatory trace, and management copy**

  Replace primary old scenario terms with the new cabin inspection terms:

  - `Crew training records` -> `PBE serviceability / emergency equipment`
  - `Flight Operations Audit` -> `Cabin Inspection`
  - `OPS-2026-001` -> `CAB-2026-001`
  - `REG-NAMCARS-OPS-2026` only where it is directly tied to the hero scenario -> a mock cabin safety reference ID such as `REG-CABIN-EQ-2026`
  - `TRACE-OPS-TRG-4.2` -> `TRACE-CAB-PBE-EMEQ`

  Keep older historical seed findings only when clearly not part of the hero story.

### Phase 5 - Scenario Docs And Evidence Docs

- [x] **Step 1: Update English scenario doc**

  Rewrite `docs/product-specs/scenarios/DEMO_SCENARIO_OPERATOR_AUDIT.md` around:

  - Audit: `2026 Cabin Inspection - FlyNamibia`
  - Checklist: `Cabin Inspection`
  - Question: `Is the PBE installed, serviceable, accessible, and in compliance with configured cabin emergency equipment requirements?`
  - Finding: `CAB-2026-001`
  - Severity: `Level 1 Critical`
  - Due date: 30 days
  - CAP/evidence flow with the sample root cause/actions/evidence from this plan.

- [x] **Step 2: Update Turkish companion scenario doc**

  Update `docs/product-specs/scenarios/DEMO_SCENARIO_OPERATOR_AUDIT.turkce.md` with the same story in Turkish, preserving careful demo/regulatory wording.

- [x] **Step 3: Update build summaries after verification**

  After implementation and tests pass, update both:

  - `docs/demo-evidence/BUILD_SUMMARY.md`
  - `docs/demo-evidence/BUILD_SUMMARY.turkce.md`

  Required evidence language:

  - `verified locally` only for checks actually run.
  - `demo-only`, `mock workbook-derived checklist`, and `not a legal/regulatory source` wording.
  - New state proof: CAP acceptance leaves `CAB-2026-001` at `EVIDENCE_REQUIRED`; evidence acceptance closes `CAB-2026-001`.

### Phase 6 - Tests And Verification

- [x] **Step 1: Update focused smoke tests**

  Change old hero assertions:

  - `q2` -> `cab-em-eq-pbe`
  - `OPS-2026-001` -> `CAB-2026-001`
  - `Crew training records incomplete` -> `PBE not serviceable or not accessible in cabin emergency equipment check`
  - `Flight Operations Audit` -> `Cabin Inspection` where the assertion is for the primary demo path.

- [x] **Step 2: Run syntax checks**

  Run:

  ```bash
  node --check js/data.js
  node --check js/helpers.js
  node --check js/approval.js
  node --check js/planning.js
  node --check js/checklists.js
  node --check js/inspection.js
  node --check js/reports.js
  node --check js/work-items.js
  node --check js/views.js
  node --check js/app.js
  ```

  Expected: all pass.

- [x] **Step 3: Run smoke tests**

  Run all available direct Node smoke tests:

  ```bash
  for test_file in tests/*.test.js; do node "$test_file"; done
  ```

  Expected: all pass. If shell loop behavior is not desired in the execution environment, run each file directly.

- [x] **Step 4: Run targeted browser smoke**

  Open `index.html` directly or serve the folder locally if browser behavior requires it.

  Verify:

  - Role select intro names `CAB-2026-001` and Cabin Inspection.
  - Inspector opens the FlyNamibia Cabin Inspection.
  - Checklist runner shows Cabin Inspection and the `EM EQ / PBE` question.
  - Marking the PBE question `Non-Compliant` opens the finding form with the correct ID/title/severity.
  - Issuing the finding creates `CAB-2026-001`.
  - Auditee sees only FlyNamibia finding data.
  - Auditee submits CAP and mock evidence filenames.
  - Inspector accepts CAP; status remains `EVIDENCE_REQUIRED`.
  - Inspector accepts evidence; status becomes `CLOSED`.
  - Manager dashboard updates.
  - No visible `Internal CAA Note`, inspector workload, other-organization data, or internal risk scoring appears to the auditee.

- [x] **Step 5: Demo boundary check**

  Confirm no new backend, database, API, real authentication, real upload/storage, real regulatory ingestion, real AI service, production audit log, framework migration, or deployment was added.

## Risks

- **Workbook looks official but is not verified clause-level regulatory material.** Mitigation: label it as a mock/configured checklist source and use careful reference language.
- **A 126-row checklist could slow the demo.** Mitigation: run a curated 6-row subset and show the 126-row source profile in admin/template context.
- **Hard-coded old hero strings are spread across JS, tests, and docs.** Mitigation: start with an `rg` map, update tests in the same implementation pass, and run the full smoke set.
- **Changing the primary scenario could break existing smoke tests.** Mitigation: update tests intentionally to assert the new scenario rather than preserving obsolete text.
- **Level 1 Critical wording can imply enforcement automation.** Mitigation: keep enforcement separate and human-authorized; do not add automatic legal/enforcement outcomes.
- **Auditee privacy regressions.** Mitigation: keep existing auditee visibility guards and rerun boundary smoke tests.

## Dependencies

- Reviewed workbook profile embedded in this plan.
- Existing static demo files and smoke tests.
- Existing product rules for Finding/CAP/Evidence lifecycle, auditee visibility, regulatory wording, and demo-only boundaries.
- No external service dependency.

## Verification

Minimum completion evidence:

- `node --check` passes for all changed JS files.
- All direct Node smoke tests pass.
- Targeted browser smoke verifies the new Cabin Inspection end-to-end path.
- `git diff --check` passes.
- Scenario docs and Turkish companion are updated.
- Build summaries are updated with exact verification labels.
- Active execution-plan index reflects the final status and next todo.

## Execution Prompt

```text
Implement the Cabin Inspection demo scenario in the AviaSurveil360 frontend-only static demo. Read AGENTS.md first and follow the demo-only boundaries.

Use the reviewed workbook as a mock/configured checklist source, not as a live Excel import or legal/regulatory source. Replace the current primary Flight Operations / crew training hero scenario with this Cabin Inspection story:

CAA Manager sees a Cabin Inspection plan. CAA Inspector opens the 2026 Cabin Inspection - FlyNamibia audit, runs a Cabin Inspection checklist, and marks the EM EQ / PBE question Non-Compliant: "Is the PBE installed, serviceable, accessible, and in compliance with configured cabin emergency equipment requirements?" The system creates Finding CAB-2026-001 with severity Level 1 Critical, risk category Emergency Preparedness, finding type Equipment, and expected evidence "PBE replacement/serviceability record, cabin defect rectification reference, and inspector photo filename." FlyNamibia submits root cause, corrective action, preventive action, target completion date, and mock evidence filenames. Inspector accepts CAP, which must leave the finding at EVIDENCE_REQUIRED. Inspector then accepts evidence, which closes the finding. Manager dashboard updates.

Update js/data.js, js/views.js, js/app.js, index.html asset tokens, focused tests, docs/product-specs/scenarios/DEMO_SCENARIO_OPERATOR_AUDIT.md, docs/product-specs/scenarios/DEMO_SCENARIO_OPERATOR_AUDIT.turkce.md, docs/demo-evidence/BUILD_SUMMARY.md, docs/demo-evidence/BUILD_SUMMARY.turkce.md, and docs/exec-plans/index.md as needed. Preserve auditee privacy, separate Comment to Auditee from Internal CAA Note, and keep CAP acceptance distinct from finding closure.

Use a curated 6-question runnable Cabin Inspection subset from the workbook and show the full source profile as 126 rows across GALLEY, LAV, PAX SEAT, EM EQ, VID+CREW SEAT, and COCKPIT+CAB GEN COND+EXITS in admin/template context. Keep the original Flight Operations template only as a secondary sample if it does not confuse the primary demo.

Run syntax checks for changed JS files, run all direct Node smoke tests, perform targeted browser smoke of the full Cabin Inspection scenario, check auditee visibility boundaries, run git diff --check, and update evidence docs with literal verification labels. Do not add backend, database, API, real auth, real upload/storage, real AI, real regulatory ingestion, production audit log, framework migration, deployment, branch operations, commits, pushes, PRs, or GitHub comments.
```
