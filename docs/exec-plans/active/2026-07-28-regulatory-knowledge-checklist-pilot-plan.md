# Regulatory Knowledge And Traceable Checklist Pilot

## Objective

Turn the supplied 2024 ICAO Protocol Question material, the NCAA Annex-to-NAMCAR
compliance crosswalk, and the supplied audit-area workbook into a
`candidate-only` regulatory-knowledge pilot that keeps practical checklist
questions traceable through:

ICAO PQ -> Critical Element -> Annex/SARP -> national reference -> NCAA
implementation source -> requirement -> verification objective -> inspection
question -> expected Evidence.

The user-visible outcome is an Administration Regulatory Library and Checklist
Builder that expose the source chain for a first OPS / Air Operator (AOC) cabin
and ramp-inspection pilot without presenting an AI draft as an approved legal or
regulatory conclusion.

## Scope

- Inventory the supplied nine 2024 ICAO PQ DOCX files, the supplied regulatory
  mapping workbook, and `CC.zip`.
- Use the OPS CE-5/CE-6/CE-7 material to establish the model; implement the
  first practical slice around OPS PQ 4.450 / CE-7 and Air Operator (AOC)
  cabin/ramp inspection.
- Add typed mapping and source-provenance fields to the Admin regulatory
  reference projection in the React mock and HTTP profiles.
- Show the mapping chain, review state, source gap, proposed verification
  questions, and evidence expectations in the Regulatory Library.
- Resolve the exact mapping identity from Question Bank / Checklist Builder
  records and show why each practical question is included.
- Preserve immutable published checklist version identity and the existing
  Admin Preview versus Department Manager publication boundary.
- Record the model and pilot boundary in canonical English product
  documentation.

## Explicit Exclusions

- No access to ICAO or NCAA self-assessment portals.
- No automated publication, official compliance finding, official EI
  calculation, enforcement decision, certificate action, or legal advice.
- No production ingestion pipeline, vector database, model-provider
  integration, or external AI call.
- No claim that the supplied workbook is a clause-level authoritative mapping;
  it is an audit-area taxonomy and seed input.
- No claim that the NCAA controlled surveillance/ramp-inspection procedure has
  been mapped until that source is supplied and reviewed.
- No changes to the root legacy demo oracle.
- No commit, push, deployment, branch operation, or external system mutation.

## Assumptions And Ownership Boundaries

- The supplied ICAO PQ DOCX files are authorized working source material for
  this repository task.
- `CC.zip` is treated as an NCAA/State compliance crosswalk source, not as a
  substitute for current promulgated NAMCAR/NAMCATS text or an NCAA controlled
  procedure.
- Public NCAA download pages may identify available/current document families,
  but publication and effective-date validation remain a technical-expert and
  regulatory-owner responsibility.
- AI output remains a draft. A technical expert validates the source linkage,
  requirement interpretation, question decomposition, evidence expectation,
  and publication decision.
- Department Manager remains the owner of checklist publication. Admin Preview
  can prepare working drafts but cannot publish them.

## Repository Orientation And Affected Interfaces

- Product model and workflow:
  - `docs/product-specs/data-and-rules/CONCEPTUAL_DATA_MODEL.md`
  - `docs/product-specs/workflows/AUDIT_CHECKLIST_WORKFLOW.md`
- React candidate:
  - `apps/web/src/backend/backend.ts`
  - `apps/web/src/backend/transport-mappers.ts`
  - `apps/web/src/mock/seed-data.ts`
  - `apps/web/src/features/admin/regulatory-library-page.tsx`
  - `apps/web/src/features/admin/checklist-builder-page.tsx`
  - focused Admin tests and shared styling
- HTTP candidate:
  - `api/openapi/source/schemas/platform.json`
  - generated OpenAPI artifact and generated client/server types
  - `apps/api/internal/configuration/workspace.go`
  - `apps/api/internal/httpapi/route_families_api.go`
  - `apps/api/internal/testprofile/canonical.go`
- Source attachments are read-only inputs and are not copied into production
  runtime artifacts.

## Ordered Tasks

### Task 1 — Inventory And Source Assessment

- [x] Confirm the workbook has one `AI Regulatory Mapping` sheet with 12 audit
  areas and eight taxonomy columns.
- [x] Confirm the nine PQ files cover LEG, ORG, PEL, OPS, AIR, AIG, ANS, AGA,
  and SSP and identify 851 total 2024 PQs.
- [x] Confirm the OPS file contains 136 PQs and explicit CE-5, CE-6, and CE-7
  rows.
- [x] Confirm `CC.zip` contains 30 Annex-to-national compliance crosswalk DOCX
  files, including Annex 6 Part I (`Annex_NAMB_A610.docx`).
- [x] Identify the first evidence-backed pilot chain:
  - OPS PQ 4.450 / CE-7;
  - Annex 6 Part I 4.2.2.2 and related cabin/equipment SARPs;
  - NAMCAR 121.07.6-121.07.8 and 135.07.6-135.07.8 plus the applicable
    cabin/equipment clauses in the supplied crosswalk;
  - a declared gap for the NCAA controlled surveillance/ramp-inspection
    procedure.

### Task 2 — Model And Contract

- [x] Define source identity, mapping status, PQ/CE, Annex/SARP, national
  reference, NCAA implementation source, requirement, verification objective,
  proposed question, and expected Evidence fields.
- [x] Add the typed projection to OpenAPI, React backend types, mock data, Go
  configuration projection, and HTTP mapping.
- [x] Regenerate checked transport artifacts and keep demo/HTTP projections
  aligned.

### Task 3 — User Experience

- [x] Add an exact OPS / Air Operator mapping filter and trace presentation to
  the Regulatory Library.
- [x] Show review status, source gaps, why-included text, proposed questions,
  and Evidence expectations without overstating authority.
- [x] Resolve mapping IDs from Question Bank / Checklist Builder records and
  show the trace for each exact question.
- [x] Keep all publication controls and immutable version behavior unchanged.

### Task 4 — Documentation And Verification

- [x] Add the regulatory knowledge entities and human-validation boundary to
  the product model and checklist workflow.
- [x] Add focused component/domain tests for trace projection, filters,
  source-gap language, and draft/published boundaries.
- [x] Run the smallest applicable contract, typecheck, test, build, and
  repository checks.
- [x] Record literal results and remaining source gaps here and in the plan
  index.

## Commands And Expected Observations

```bash
npm --prefix apps/web run typecheck
```

Expected: React types accept the generated mapping projection with no errors.

```bash
npm --prefix apps/web test -- src/features/admin/admin-secondary-pages.test.tsx src/backend/transport-mappers.test.ts
```

Expected: the Admin regulatory trace and immutable checklist Draft boundaries
pass in the mock profile.

```bash
npm --prefix apps/web run build:demo
```

Expected: the demo artifact builds and remains `candidate-only`.

```bash
./scripts/check-contracts.sh
```

Expected: OpenAPI generation, generated Go/TypeScript types, examples, and drift
checks pass.

```bash
go test ./internal/configuration ./internal/httpapi
```

Run from `apps/api`. Expected: HTTP projection, authority, and mapping decoding
tests pass.

```bash
node --test tests/checklist-management-smoke.test.js tests/manager-checklist-management-smoke.test.js
git diff --check
```

Expected: legacy checklist semantics remain intact and the diff has no
whitespace errors.

## Verification And Acceptance Criteria

- The pilot mapping has exact, stable source and mapping identities.
- The UI shows the complete declared chain and does not hide the missing NCAA
  procedure/manual source.
- One regulatory requirement may expose multiple practical checklist
  questions; no one-question-per-clause assumption is embedded.
- Every proposed question includes a verification purpose, expected Evidence,
  and why-included explanation.
- AI/mapping output is visibly `EXPERT_REVIEW_REQUIRED` and cannot publish
  itself.
- The immutable published version and Department Manager publication ownership
  remain unchanged.
- Demo and HTTP backend projections use the same typed contract.
- Relevant focused checks are fresh and literal.

## Risks, Dependencies, Idempotence, And Recovery

- Risk: the workbook may be mistaken for an authoritative clause map.
  Mitigation: label it as taxonomy/seed input and keep source-level review
  states.
- Risk: a CC crosswalk may not match the latest promulgated national text.
  Mitigation: retain source version/date, public NCAA source URL, and expert
  review requirement.
- Risk: mapping a State-level PQ directly to an operator checklist may create
  invalid questions. Mitigation: model requirement and verification objective
  separately, retain why-included text, and require technical validation.
- Risk: long trace content may degrade mobile readability. Mitigation: use
  progressive disclosure and existing responsive Admin patterns.
- Changes are additive and idempotent. Existing immutable published versions
  are not rewritten by a runtime command.
- Recovery is a normal file-level revert of this plan's additive contract,
  projection, seed, UI, test, and documentation changes. No database migration
  or external state is introduced by this pilot.

## Progress

- 2026-07-28: Plan created. Source inventory and first OPS / AOC pilot chain are
  complete.
- 2026-07-28: Implementation is in progress. No verification result is claimed
  yet.
- 2026-07-28: The typed demo/HTTP mapping projection, OPS / AOC Regulatory
  Library presentation, exact-question Checklist Builder trace, and product
  model/workflow documentation are implemented. Fresh verification is pending.
- 2026-07-28: Focused contract, TypeScript, component, Go, demo-build,
  legacy-boundary, responsive-browser, console, process-cleanup, and diff gates
  passed. The pilot is `verified locally`, `candidate-only`, and ready for NCAA
  technical-expert validation.

## Decisions

- Start with OPS and Air Operator (AOC), using CE-7 PQ 4.450 as the practical
  ramp/cabin pilot.
- Represent mappings inside the versioned regulatory-reference snapshot for
  this pilot. Do not add a new mutable runtime table before the model and
  expert-review workflow are accepted.
- Keep question records linked by exact configured mapping identity. Do not
  copy the full source graph into each checklist response.
- Treat the controlled NCAA procedure/manual as a required, visible source gap.

## Discoveries

- The supplied workbook is an eight-column, 12-area mapping taxonomy rather
  than a clause- or PQ-level mapping.
- The 2024 OPS PQ document explicitly describes CE-7 verification through the
  surveillance plan/programme, inspection reports, and deficiency tracking.
- PQ 4.450 calls for risk-based ramp inspection coverage including cabin/safety.
- The supplied Annex 6 Part I crosswalk maps 4.2.2.2 to NAMCAR
  121.07.6-121.07.8 and 135.07.6-135.07.8 and supplies more granular cabin and
  equipment references for practical question decomposition.
- The NCAA public download library is live and versioned; source currency must
  be reviewed rather than inferred from a filename.

## Outcome Notes

Implemented a versioned OPS / Air Operator (AOC) regulatory mapping with exact
source identities, PQ 4.450 / CE-7, Annex 6 and national references,
requirement/verification interpretation, a visible controlled-source gap, and
six practical cabin/ramp questions linked to the immutable
`CTV-CABIN-1` checklist. The same projection is available through the demo and
HTTP candidate contracts. AI output remains `EXPERT_REVIEW_REQUIRED`; no
publication, legal conclusion, self-assessment portal access, or external
system mutation occurred.

Fresh local results on 2026-07-28:

- `npm --prefix apps/web run typecheck`: passed.
- focused Vitest: 2 files, 21 tests passed.
- `./scripts/check-contracts.sh`: passed; 16 contract tests passed and generated
  drift was clean.
- `go test ./internal/configuration ./internal/httpapi`: the first invocation
  was blocked by sandbox access to the default user Go cache; the exact tests
  passed after setting `GOCACHE=/private/tmp/avia-regulatory-go-cache`.
- `npm --prefix apps/web run build:demo`: passed; Vite emitted its non-failing
  chunk-size advisory.
- focused root smoke tests: 3 passed
  (`checklist-management`, `manager-checklist-management`, and
  `demo-boundary`).
- In-app browser QA at 1440x900 and 390x844: Regulatory Library and Checklist
  Builder had zero document horizontal overflow and zero console
  warning/error entries; six candidate questions and six exact published
  traces were visible, the provider filter worked, progressive disclosure
  opened, and immutable edit remained disabled.
- Task-owned browser/Vite/automation process check: no matching process
  remained.
- `git diff --check`: passed.

Remaining validation is deliberately external to this local implementation:
the controlled NCAA Operations surveillance/ramp-inspection procedure or
manual, Part 127 applicability, Part 140 linkage, current promulgated national
text, requirement interpretation, six question decompositions, and Evidence
expectations require NCAA technical-expert review. This is recorded in the
technical-debt tracker.

## Execution Prompt

Continue the active Regulatory Knowledge And Traceable Checklist Pilot. Read
this plan completely, preserve the root demo oracle and all unrelated user
changes, and work on the current branch. Implement Tasks 2-4 using the exact
OPS / Air Operator pilot chain already recorded. Keep the missing NCAA
controlled procedure/manual visible, retain `EXPERT_REVIEW_REQUIRED`, preserve
immutable published checklist identity and Department Manager publication
authority, run the listed focused checks, and update this plan plus
`docs/exec-plans/index.md` with literal results. Do not commit, push, deploy, or
perform branch operations.
