# Full React 86-Screen Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:executing-plans` to execute this plan task by task. Do not
> dispatch subagents unless the user explicitly authorizes subagent work.
> Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate every accepted root-demo screen into one source-faithful
React application so all 86 routes work completely in deterministic demo mode,
without creating inert controls or weakening the existing 17-route result.

**Architecture:** Keep the root Vanilla application intact as the visual and
behavioral oracle. Expand the typed route/screen registry from 17 to 86, add
domain-shaped mock capabilities to the existing `Backend`, and render every
screen through the same React shell and feature-owned CSS. New surfaces remain
demo-enabled and HTTP-disabled with an explicit reason until the next plan
implements and verifies their real Go capabilities.

**Tech Stack:** React 19, TypeScript 5.9, Vite 8, React Router 7, TanStack Query,
React Hook Form, Zod, deterministic `MemoryMockStore`, Vitest, React Testing
Library, Playwright 1.61, decoded-RGBA visual comparison, and the accepted root
HTML/CSS/JavaScript oracle.

**Status:** `active` — Tasks 1–12 have been executed. Task 11 is
`verified locally`; Task 12's non-visual matrix passed, but the one-shot visual
matrix remains literally 71/259 and standalone baseline integrity is
`not verified`. The plan therefore does not meet the
`ready-for-verification` gate. Independent review for Tasks 11–12 is `not run`,
and Plan 2 remains `blocked`. Historical task evidence follows. Task 4 passed
its functional, contract, type, build,
route-boundary, root-diff, hygiene, and independent spec/code-quality gates.
Its final exact Inspector visual command returned 19 passed / 15 nominal
failures after truthful identity, assignment-scope, lifecycle, and accessible
navigation corrections. The user inspected the visible difference, accepted
the deviations, and explicitly directed that no further pixel-matching loop be
run; accepted baselines, masks, thresholds, semantic truth, and the root oracle
were not weakened. Task 5 passed its functional, authority, identity,
responsive-hierarchy, build, boundary, hygiene, and independent review gates;
its bounded Lead visual result remains literally 11/43 under the same accepted
no-pixel-loop disposition. Task 6 passed its lifecycle/authority/identity,
responsive-browser, build, boundary, hygiene, and independent review gates;
its bounded Manager visual result remains literally 10/40. Task 7 passed its
planning/governance/privacy, risk/CAP truth, responsive-browser, build,
boundary, hygiene, and independent review gates; its bounded route-pair result
remains literally 0/36 and post-correction visuals are `not reverified`. Task 8
passed its exact governance/authority/identity, stage-eligible queue, profile and
notification persistence, responsive-browser, build, boundary, hygiene, and
independent review gates; its bounded visual result remains literally 15/42
route pairs with the primitive gallery green, and post-correction visuals are
`not reverified`. Task 9 passed its Auditee privacy, exact-identity,
persistence/migration, responsive-browser, build, boundary, hygiene, and fresh
independent re-review gates. Its single bounded visual run failed with the exact
final count unavailable after reporter evidence was overwritten; the primitive
gallery passed, and the matrix was not rerun under the user's no-pixel-loop
direction. Task 10 passed its focused 101/101, full React 515/515, root/parity
107/107, typecheck, demo build, route/profile boundary, responsive Chromium
3/3, root-diff, diff-check, and process-hygiene gates. The main-agent
spec/code-quality review found no Critical or Important findings. The
2026-07-23 independent review of `a598571..8f8a252` passed focused Vitest 53/53,
typecheck, and range diff-check, found no Critical or Minor issues, and found
one Important fail-closed HTTP direct-load defect: profile-insensitive parent
resolution can mount demo-only Admin parents for two contextual routes. The
correction was implemented through RED → GREEN TDD: two real-`HttpBackend`
regressions reproduced the exceptions, then passed after parent resolution was
made profile-aware. Fresh correction gates passed focused Task 10 103/103, full
React 517/517, typecheck, demo and HTTP builds, HTTP artifact scan, exact
86-route/two-profile boundary, root/parity 107/107, root-oracle diff, and
diff-check. The correction is `verified locally`; its clean independent
re-review is `not run`, so Task 10 is not independently accepted yet. Its
single bounded visual matrix remains literally 1/40, with the primitive
gallery green and all 39 Admin route pairs non-green, and was not rerun. Task 11
subsequently passed its full interaction/accessibility boundary without claiming
an independent review. Task 12 executed the required full matrix once and
recorded its non-green visual result literally. A 2026-07-25 exact-runtime
follow-up verifies standalone baseline integrity, while the fresh visual
diagnostic remains non-green at 74/259. Plan 2 has not started.

## Objective

Deliver a complete stakeholder demo in React with the same 86 accepted screen
states, role navigation, visible behavior, lifecycle truth, and responsive
composition as the root demo. The result must be useful before the real backend
expansion, while making the backend boundary explicit enough that Plan 2 can
activate HTTP without redesigning components.

## Scope

- Preserve and continuously reverify the existing 17 React surfaces.
- Correct two inherited role-crossed mappings while preserving the 17-surface
  count: `ui-audit-009` is a CAA Inspector Finding Detail and `ui-audit-044` is
  a Department Manager Inspection Evidence surface. Lead review authority stays
  on `ui-audit-013` and `ui-audit-022`; a Lead route must not stand in for either
  source row.
- Add the remaining 69 accepted audit rows:
  - Inspector: `ui-audit-003`–`006`, `010`–`012` — 7 surfaces.
  - Lead Inspector: `ui-audit-014`–`021`, `023`–`026` — 12 surfaces.
  - Department Manager: `ui-audit-029`, `031`–`040`, `042`–`043`,
    `045`–`051` — 20 surfaces.
  - General Manager: `ui-audit-053`–`057` — 5 surfaces.
  - Executive Director: `ui-audit-060`–`065` — 6 surfaces.
  - Auditee: `ui-audit-067`–`073` — 7 surfaces.
  - Admin: `ui-audit-074`–`075`, `077`–`086` — 12 surfaces.
- Keep Finance Review and the existing 16 protected routes green.
- Make all 86 surfaces directly addressable by stable React URLs.
- Implement every demo action through `MockBackend`, navigation, form state, or
  explicit local state with a visible durable effect.
- Add tracked, hash-verified legacy baselines and decoded-pixel comparison for
  86 routes × 3 viewports.
- Produce synchronized English/Turkish evidence and update the plan lifecycle.

## Assumptions

- The accepted root demo and the 86-row audit remain the visual/behavior oracle.
- Plan 2 will implement real HTTP behavior for new capabilities. Until then,
  route metadata exposes new routes only in the `demo` profile and records the
  exact missing HTTP capability; the HTTP profile must not show a fake control.
- The existing 17 routes remain available in both demo and HTTP profiles.
- A route counted against an audit row must use that row's normalized source
  role. Shared lifecycle components may be reused only with role-safe
  projections; cross-role visual fixtures are not accepted parity evidence.
- AI Inspector Assistant is a deterministic advisory draft in this plan. It
  cannot create a Finding, set severity, close work, or make enforcement action.
- Screen labels may display truthful seed values, but shell, hierarchy, density,
  action placement, and responsive behavior must remain recognizable as the
  accepted demo.
- No root runtime CSS/JavaScript is imported by the React artifact.

## Global Constraints

- Follow `AGENTS.md` and
  `docs/product-specs/product-plan/MODULE_ARCHITECTURE.md`.
- Keep `Comment to Auditee` and `Internal CAA Note` structurally distinct.
- CAP acceptance never closes a Finding. Evidence acceptance and verification
  or explicit authorized closure is required.
- Preserve immutable CAP, Evidence, checklist-template, and report versions.
- Auditee routes expose only their organization and omit internal CAA data.
- Every visible button, link, input, select, tab, menu item, download, and row
  action must work or be visibly disabled with a specific reason.
- Do not add microservices, Next.js, a second UI, generic placeholder pages,
  external AI calls, real email, real storage, or real identity in this plan.
- Do not modify the root oracle except through a separately reviewed correction.
- Work on the current branch; do not create or switch branches.
- Preserve `.superpowers/`, `docs/demo-evidence/stakeholder/`, and `outputs/`.

## Ownership Boundaries

| Owner | Responsibility |
|---|---|
| Product/CAA Operations | Accept the 86-screen inventory, role purpose, lifecycle copy, and advisory-only boundaries |
| Frontend | Route registry, React components, mock capabilities, CSS ownership, accessibility, responsive behavior, and visual evidence |
| Backend plan owner | Consume the frozen capability contract in Plan 2 without changing screen purpose |
| QA | Semantic, visible-action, direct-route, responsive, keyboard, and decoded-pixel matrices |
| Records/Legal | No production approval in this plan; review remains a later gate |

## Route And File Structure

The route registry remains the single source of React reachability:

```ts
export type BuildProfileAvailability = "demo" | "http";
export type ScreenComponentKey = keyof typeof SCREEN_COMPONENT_REGISTRY;

export interface RouteContract {
  id: ReactSurfaceId;
  auditId: `ui-audit-${string}`;
  path: string;
  requiredRole: Role | null;
  placement: RoutePlacement;
  parentId: ReactSurfaceId | null;
  label: string;
  iconKey: IconKey;
  order: number;
  dataBoundary: DataBoundary;
  componentKey: ScreenComponentKey;
  availableProfiles: readonly BuildProfileAvailability[];
  blockedProfileReason?: string;
}
```

Feature files are grouped by domain, not by menu depth:

- `apps/web/src/features/communications/`
- `apps/web/src/features/calendar/`
- `apps/web/src/features/profile/`
- `apps/web/src/features/reports/`
- `apps/web/src/features/teams/`
- `apps/web/src/features/risk/`
- `apps/web/src/features/administration/`
- existing `assignments/`, `planning/`, `inspections/`, `checklists/`,
  `findings/`, `caps/`, `evidence/`, `organizations/`, and `admin/` folders.

Feature CSS remains in `apps/web/src/styles/features/` and is registered by
`apps/web/src/styles/style-ownership.test.ts`.

## Dependency Order

This is Plan 1 of 4. The sequence is intentionally serial: Plan 2 does not begin
until Tasks 1–12 are complete, this plan is `ready-for-verification`, and the
route/capability/action contracts are independently accepted. Plan 3 depends on
Plan 2's adapters and worker jobs. Plan 4 depends on the local full-stack gates
from Plans 1–3.

## Phases

1. **Contract and oracle foundation — Tasks 1–3:** freeze 86 routes, complete
   deterministic mock capabilities, and expand the visual oracle to 258 pairs.
2. **Source-faithful feature migration — Tasks 4–10:** migrate every remaining
   role surface while preserving existing routes and shared-shell behavior.
3. **Boundary and handoff — Tasks 11–12:** enforce complete interaction/
   accessibility coverage, run the full demo matrix, and record evidence.

---

### Task 1: Freeze The 86-Route React Contract

**Files**

- Modify `apps/web/src/app/route-contracts.ts`
- Modify `apps/web/src/app/route-contracts.test.ts`
- Create `apps/web/src/app/screen-component-registry.tsx`
- Modify `apps/web/src/app/router.tsx`
- Modify `apps/web/src/app/router.test.tsx`
- Modify `apps/web/src/parity/legacy-screen-manifest.ts`
- Modify `apps/web/src/parity/legacy-screen-manifest.test.ts`
- Modify `apps/web/tests/e2e/support/legacy-parity-fixtures.ts`
- Modify `tests/parity/behavior-ledger.json`
- Modify `tests/parity/react-legacy-parity.test.mjs`

**Interfaces**

- Produces exactly 86 unique `ReactSurfaceId` values and route contracts.
- Every route declares role, parent, placement, data boundary, profile
  availability, audit ID, and component key.
- Normalized audit-source role must equal `requiredRole` for all role-owned
  rows. Task 1 explicitly re-homes `ui-audit-009` from Lead to CAA Inspector and
  `ui-audit-044` from Lead to Department Manager, including paths, parents,
  fixtures, and direct-load guards.
- The 69 new routes use `availableProfiles: ["demo"]` and an exact Plan 2 HTTP
  activation reason. The 17 existing routes retain `["demo", "http"]`.

- [ ] Write failing tests that require 86 ordered source rows, 86 unique route
  IDs/paths, 0 legacy-only rows, exact 17 dual-profile routes, exact 69
  demo-only-until-Plan-2 routes, exact audit-role/route-role agreement, the two
  corrected inherited mappings, and no wildcard placeholder render.
- [ ] Run
  `npm --prefix apps/web test -- src/app/route-contracts.test.ts src/app/router.test.tsx src/parity/legacy-screen-manifest.test.ts`
  and confirm failure reports `expected 86`, not a syntax/setup failure.
- [ ] Implement the expanded unions, contracts, profile guard, and lazy screen
  component registry. A blocked HTTP direct load must render the real parent
  route plus an explicit unavailable-capability notice; it must not render the
  demo component or a generic placeholder.
- [ ] Run the focused tests, then
  `node --test tests/parity/react-legacy-parity.test.mjs`; expect all route and
  parity assertions green at `86/86`.
- [ ] Commit exactly `feat(ui): freeze full 86-screen route contract` after
  checking the task allowlist and cached diff.

### Task 2: Expand The Demo Backend Capability Contract

**Files**

- Modify `apps/web/src/backend/backend.ts`
- Modify `apps/web/src/backend/backend-contracts.ts`
- Modify `apps/web/src/mock/memory-mock-store.ts`
- Modify `apps/web/src/mock/mock-engine.ts`
- Modify `apps/web/src/mock/create-mock-backend.ts`
- Modify `apps/web/src/mock/seed-data.ts`
- Create `apps/web/src/mock/full-screen-scenario.test.ts`
- Modify `apps/web/tests/contract/mock-backend.test.ts`

**Interfaces**

Add composed capabilities named `communications`, `calendar`, `profiles`,
`teams`, `risk`, `documents`, `notifications`, `administration`, and
`assistantDrafts`. Each capability exposes typed query and command methods and
returns immutable projections. Commands accept `expectedRevision` and
`idempotencyKey` where they change canonical demo state.

- [ ] Write a failing contract test that loads all 86 screen projections from
  one seed and executes every declared visible command, including direct-ID,
  empty, denied, returned, overdue, and version-history states.
- [ ] Run
  `npm --prefix apps/web test -- src/mock/full-screen-scenario.test.ts tests/contract/mock-backend.test.ts`
  and confirm the new capability names are missing.
- [ ] Implement the capability interfaces, deterministic seed records,
  organization/role filters, revision checks, idempotent commands, notification
  records, document metadata, and advisory draft output. Do not duplicate
  existing Finding/CAP/Evidence state in new stores.
- [ ] Run focused tests and the existing canonical scenario contract; expect
  exact deterministic IDs and zero Auditee internal-field leakage.
- [ ] Commit exactly `feat(mock): cover full screen capability contract`.

### Task 3: Expand The Visual Oracle To 258 Pairs

**Files**

- Modify `apps/web/tests/e2e/support/legacy-parity-fixtures.ts`
- Modify `apps/web/tests/e2e/legacy-visual-parity.spec.ts`
- Modify `apps/web/tests/visual/visual-contract.test.ts`
- Modify `apps/web/scripts/verify-visual-baselines.mjs`
- Modify `apps/web/tests/visual-baselines/react-legacy-parity/baseline-manifest.json`
- Add 207 PNGs under
  `apps/web/tests/visual-baselines/react-legacy-parity/{desktop,tablet,mobile}/`
- Replace the six tracked role-crossed baselines for `ui-audit-009` and
  `ui-audit-044` with source-role-correct captures
- Modify `apps/web/scripts/assert-parity-boundary.mjs`

**Interfaces**

- Exactly 86 surfaces × 3 viewports = 258 legacy/React comparisons.
- The 207 new images cover the 69 new rows; the six replaced images correct the
  two inherited source-role mismatches and do not change the 86-surface count.
- Decoded RGBA threshold remains per-channel delta `40`, shell `<= 0.03`,
  predeclared content `<= 0.08`, masks explicit and `<= 5%`.

- [ ] Add red mutation fixtures for missing route, skipped viewport, absent
  shell/content region, changed comparator, untracked baseline, broad mask, and
  candidate/result attachment count below 258.
- [ ] Run the visual contract and boundary tests; confirm every mutation fails
  for its named reason.
- [ ] Capture the remaining accepted root states with pinned Chromium inputs,
  inspect every new baseline contact sheet, and update the hash manifest only
  through the reviewed baseline command.
- [ ] Run baseline verification and a no-op root recapture; expect 258 valid
  hashes, zero missing metadata, and no unauthorized root-source change.
- [ ] Commit exactly `test(ui): expand full screen visual oracle`.

### Task 4: Migrate The Seven Remaining Inspector Surfaces

**Files**

- Create `apps/web/src/features/findings/inspector-findings-page.tsx`
- Create `apps/web/src/features/communications/message-center-page.tsx`
- Create `apps/web/src/features/calendar/role-calendar-page.tsx`
- Create `apps/web/src/features/reports/inspector-reports-page.tsx`
- Create `apps/web/src/features/reports/closure-report-page.tsx`
- Create `apps/web/src/features/assistant/inspector-assistant-page.tsx`
- Create `apps/web/src/features/profile/profile-page.tsx`
- Create `apps/web/src/features/inspector/inspector-secondary-pages.test.tsx`
- Modify `apps/web/src/features/findings/finding-detail-page.tsx`
- Modify `apps/web/src/features/findings/finding-detail-page.test.tsx`
- Create `apps/web/src/styles/features/inspector-secondary.css`
- Modify `apps/web/src/styles/app.css`
- Modify `apps/web/src/styles/style-ownership.test.ts`

**Interfaces**

Covers new audit IDs `003`–`006`, `010`–`012` and corrects the inherited
`ui-audit-009` route so Finding Detail is owned, addressed, rendered, and
authorized as a CAA Inspector surface. Shared message/calendar/profile
components accept a role-safe projection; Inspector assistant consumes only
`assistantDrafts.createDraft` and always labels output as a draft.

- [x] Write seven failing tests for the new routes and a failing source-role
  regression for corrected `ui-audit-009`, covering purpose, owner, next action,
  Due Date, role scope, working actions, direct load, and mobile hierarchy.
- [x] Run the focused tests and the 21 new-route visual pairs; confirm expected red
  composition/action failures.
- [x] Port the exact root hierarchy and bounded selectors, connect all actions
  to the demo capability contract, and preserve the existing Inspector shell.
- [x] Run focused tests, all Inspector routes, visible-action inventory, and 11
  Inspector visual states × 3 viewports. Functional and contract gates are
  green; the exact visual command's truthful deviations are explicitly accepted
  by the user and recorded in the progress ledger and tech-debt tracker.
- [x] Commit disposition: not performed because the current execution explicitly
  forbids commits.

### Task 5: Migrate The Twelve Remaining Lead Inspector Surfaces

**Files**

- Create `apps/web/src/features/reports/lead-report-workspaces.tsx`
- Create `apps/web/src/features/reports/lead-report-workspaces.test.tsx`
- Create `apps/web/src/features/teams/audit-assignment-page.tsx`
- Create `apps/web/src/features/teams/question-assignment-page.tsx`
- Create `apps/web/src/features/teams/lead-assignment-pages.test.tsx`
- Reuse `message-center-page.tsx`, `role-calendar-page.tsx`, and `profile-page.tsx`
- Create `apps/web/src/features/reports/lead-analytics-page.tsx`
- Create `apps/web/src/styles/features/lead-secondary.css`
- Modify `apps/web/src/styles/app.css`
- Modify `apps/web/src/styles/style-ownership.test.ts`

**Interfaces**

Covers audit IDs `014`–`021`, `023`–`026`. Report state remains versioned and
follows Lead preparation/readiness boundaries; assignment commands preserve
exact Audit/question IDs and cannot change approval authority.

- [x] Write 12 failing direct-route and behavior tests, including returned
  Preliminary Report, readiness blockers, assignment workload, exact question
  scope, internal/auditee comment separation, and working document preview.
- [x] Run focused tests and 36 visual pairs; confirm red for missing surfaces.
- [x] Port source DOM/CSS and implement typed demo actions without changing the
  existing Assigned Audits and CAP Review surfaces.
- [x] Run all 14 Lead routes, keyboard/visible-action checks, and 42 Lead visual
  comparisons. Functional and review gates are green; the bounded 11/43 visual
  result and accepted stale-oracle/pixel deviations are recorded literally.
- [x] Commit disposition: not performed because the current execution explicitly
  forbids commits.

### Task 6: Migrate Department Manager Operational Workspaces

**Files**

- Create `apps/web/src/features/inspections/manager-audits-page.tsx`
- Create `apps/web/src/features/teams/inspection-team-page.tsx`
- Create `apps/web/src/features/findings/manager-findings-review-page.tsx`
- Create `apps/web/src/features/caps/manager-cap-monitoring-page.tsx`
- Create `apps/web/src/features/checklists/checklist-management-page.tsx`
- Create `apps/web/src/features/organizations/organization-detail-page.tsx`
- Create `apps/web/src/features/reports/manager-preliminary-review-page.tsx`
- Create `apps/web/src/features/caps/department-closure-review-page.tsx`
- Create `apps/web/src/features/management/manager-operational-pages.test.tsx`
- Modify `apps/web/src/features/evidence/evidence-review-page.tsx`
- Modify `apps/web/src/features/evidence/evidence-review-page.test.tsx`
- Create `apps/web/src/styles/features/manager-operations.css`
- Modify `apps/web/src/styles/app.css`
- Modify `apps/web/src/styles/style-ownership.test.ts`

**Interfaces**

Covers `ui-audit-029`, `032`–`035`, `042`, `045`–`046`. Decisions must call
the existing authoritative demo lifecycle commands and record exact actor,
reason, revision, report/Evidence version, and audit event.
This task also corrects inherited `ui-audit-044`: Inspection Evidence is a
Department Manager route/projection, not a Lead Inspector substitute. Lead CAP
and Evidence review remains inside the accepted Lead surfaces `013` and `022`.

- [x] Write failing tests for all eight new routes plus a source-role regression
  for corrected `ui-audit-044` and their denied/returned/direct-ID cases,
  including closure outcomes that keep partial/not-close Findings open.
- [x] Run focused tests and confirm missing-route red results.
- [x] Port exact operational register/dossier/action compositions and connect
  the typed demo commands; use responsive cards where the accepted root does.
- [x] Run the eight routes across desktop/tablet/mobile plus existing Manager
  routes and canonical lifecycle regression. Functional, responsive-browser,
  and independent review gates are green; the bounded 10/40 pixel result is
  recorded literally under the accepted no-loop disposition.
- [x] Commit disposition: not performed because the current execution explicitly
  forbids commits.

### Task 7: Migrate Department Manager Intelligence, Package, And Intake Workspaces

**Files**

- Create `apps/web/src/features/risk/manager-risk-workspaces.tsx`
- Create `apps/web/src/features/risk/manager-risk-workspaces.test.tsx`
- Create `apps/web/src/features/inspections/inspection-package-builder-page.tsx`
- Create `apps/web/src/features/planning/new-audit-wizard.tsx`
- Create `apps/web/src/features/planning/new-audit-wizard.test.tsx`
- Create `apps/web/src/styles/features/manager-intelligence.css`
- Create `apps/web/src/styles/features/planning-intake.css`
- Modify `apps/web/src/styles/app.css`
- Modify `apps/web/src/styles/style-ownership.test.ts`

**Interfaces**

Covers `ui-audit-031`, `036`–`040`, `043`, `047`–`051`. The five wizard
routes are one persisted draft with explicit step URLs. Submission creates a
Planning item, not an executable Audit; unannounced notice remains withheld.
Risk/Oversight Health values are indicators only.

- [x] Write failing tests for six risk/intelligence routes, package builder,
  five wizard steps, draft restart, validation, Back/Next, submission, zero
  budget Finance path, and unannounced privacy.
- [x] Run focused tests and 36 visual pairs; confirm missing composition/action red.
- [x] Port exact root compositions, create one Zod-validated wizard state, and
  call typed mock planning/package commands with deterministic revisions.
- [x] Run all 12 routes, planning lifecycle regression, visible actions, and
  responsive visuals. Functional, governance/privacy, responsive-browser, and
  independent review gates are green; the bounded pixel result and
  post-correction `not reverified` status are recorded literally.
- [x] Commit disposition: not performed because the current execution explicitly
  forbids commits.

### Task 8: Complete General Manager And Executive Director Workspaces

**Files**

- Extend `apps/web/src/features/planning/planning-workspaces.tsx`
- Create `apps/web/src/features/management/department-comparison-page.tsx`
- Create `apps/web/src/features/risk/executive-risk-page.tsx`
- Create `apps/web/src/features/reports/executive-report-workspaces.tsx`
- Create `apps/web/src/features/notifications/executive-notifications-page.tsx`
- Reuse `profile-page.tsx` for settings projections
- Create `apps/web/src/features/management/executive-workspaces.test.tsx`
- Create `apps/web/src/styles/features/executive-secondary.css`
- Modify `apps/web/src/styles/app.css`

**Interfaces**

Covers `ui-audit-053`–`057` and `060`–`065`. Planning preserves Department
Manager → Finance → General Manager → Executive Director → GM Release to
Department. Preliminary and Final Reports preserve Lead Inspector → Department
Manager → General Manager → Executive Director and bind every decision to the
exact immutable report version. Finance never participates in report approval.

- [x] Write 11 failing tests for decision queues, denied authority, return
  reasons, departments, risk indicator boundary, notifications, settings, and
  report preview/version identity.
- [x] Run focused tests and confirm the intended missing-route/behavior RED.
- [x] Port role-specific root compositions and connect demo capability commands.
- [x] Run all GM/Finance/Executive routes, canonical approval regressions,
  visible actions, and the bounded 42-pair visual matrix. Binding functional,
  authority, identity, responsive, build, boundary, and hygiene gates passed;
  the literal visual result is 15/42 route pairs plus the primitive gallery,
  with post-correction visuals `not reverified` under the user's no-loop
  direction.
- [x] Obtain fresh independent spec-compliance and code-quality approval after
  correcting all four Important review findings through RED → GREEN TDD.
- [x] Commit disposition: not performed because the current execution explicitly
  forbids commits.

### Task 9: Complete The Auditee Portal

**Files**

- Create `apps/web/src/features/auditee/inspection-coordination-page.tsx`
- Create `apps/web/src/features/reports/auditee-report-pages.tsx`
- Reuse `message-center-page.tsx` and `profile-page.tsx`
- Create `apps/web/src/features/documents/auditee-documents-page.tsx`
- Create `apps/web/src/features/auditee/auditee-secondary-pages.test.tsx`
- Create `apps/web/src/styles/features/auditee-secondary.css`
- Modify `apps/web/src/styles/app.css`

**Interfaces**

Covers `ui-audit-067`–`073`. Every projection is organization-scoped and
structurally excludes Internal CAA Note, CAA workload, risk scoring, other
organizations, and enforcement deliberations.

- [x] Write seven failing projection/route tests plus raw object scans for
  forbidden fields, direct-ID denial, report version, document version, message
  visibility, and truthful next action.
- [x] Run focused tests and the single bounded visual matrix; confirm the
  functional RED and record the literal non-green visual result without a
  pixel-only rerun.
- [x] Port exact Service Provider Portal compositions and connect only
  Auditee-safe demo capabilities.
- [x] Run all eight Auditee screens, canonical isolation tests, visible actions,
  and responsive desktop/tablet/mobile checks; verify zero forbidden fields and
  correct all four Important review findings through RED → GREEN TDD.
- [x] Commit disposition: not performed because the current execution explicitly
  forbids commits.

### Task 10: Complete Administration And Configuration Workspaces

**Files**

- Create `apps/web/src/features/admin/regulatory-library-page.tsx`
- Create `apps/web/src/features/admin/template-list-page.tsx`
- Create `apps/web/src/features/admin/question-bank-page.tsx`
- Create `apps/web/src/features/admin/checklist-builder-page.tsx`
- Create `apps/web/src/features/admin/template-version-history-page.tsx`
- Create `apps/web/src/features/admin/inspection-package-admin-page.tsx`
- Create `apps/web/src/features/admin/admin-reports-page.tsx`
- Create `apps/web/src/features/admin/users-roles-page.tsx`
- Extend `apps/web/src/features/admin/admin-configuration-page.tsx`
- Create `apps/web/src/features/admin/organization-master-data-page.tsx`
- Create `apps/web/src/features/admin/admin-audit-log-page.tsx`
- Create `apps/web/src/features/admin/admin-secondary-pages.test.tsx`
- Create `apps/web/src/styles/features/admin-secondary.css`
- Modify `apps/web/src/styles/app.css`

**Interfaces**

Covers `ui-audit-074`–`075`, `077`–`086`. Demo mutations create new immutable
versions. Regulatory content is labelled configured reference, not legal advice.
User provisioning is visibly demo-only until Plan 3 activates Keycloak admin.

- [x] Write 12 failing tests for list/detail/direct load, multiline questions,
  builder ordering, immutable publish/version history, user scope, organization
  master data, audit log filters, disabled production-only provisioning, and
  reference language.
- [x] Reproduce the checkpoint RED (10 TS18047 errors and 11/14 focused Admin
  tests) and run the single bounded visual matrix once; its literal result is
  1/40 with the primitive gallery green and all 39 Admin route pairs non-green.
- [x] Port the accepted Administration shell/compositions and implement all
  deterministic demo actions without mutating published versions.
- [x] Run all 13 Admin routes, accessible forms, visible actions, persistence,
  identity, authority, responsive, regression, build, boundary, root-diff, and
  hygiene checks; functional gates are green and the accepted non-green visual
  result is recorded without a pixel-only rerun.
- [x] Resolve the independent review's Important HTTP fail-closed direct-load
  finding through RED → GREEN TDD by making contextual parent resolution
  profile-aware and adding real `HttpBackend` regressions for the Inspection
  Package and Organization Detail routes.
- [x] Independent re-review disposition: `not run`. The resumed checkpoint
  accepts Tasks 1–10 as complete and does not claim an independent review of
  the Task 10 correction.
- [x] Commit disposition: the user explicitly authorized the focused Task 10
  commit before Task 11.

### Task 11: Enforce The 86-Surface Interaction And Accessibility Boundary

**Files**

- Modify `apps/web/tests/e2e/visible-action-contract.spec.ts`
- Modify `apps/web/tests/e2e/release-candidate-gates.spec.ts`
- Create `apps/web/tests/e2e/full-route-accessibility.spec.ts`
- Modify `apps/web/scripts/assert-parity-boundary.mjs`
- Modify `tests/parity/behavior-ledger.json`
- Modify `tests/parity/react-legacy-parity.test.mjs`

**Interfaces**

The action ledger must enumerate every non-native/local/backend action and the
test that proves its outcome. Only one visible navigation landmark is accessible
at a time; no off-canvas duplicate controls remain in the accessibility tree.

- [x] Add red mutation fixtures for an inert button, toast-only action,
  unlabelled control, fake dropdown, duplicate accessible navigation, missing
  disabled reason, broken deep link, and missing mobile viewport.
- [x] Run mutation tests and confirm every fixture fails closed.
- [x] Expand the inventory loop to 86 routes × desktop/tablet/mobile and add
  keyboard, dialog focus/return, heading/landmark, native form semantics,
  target-size, and document-overflow checks.
- [x] Run the complete demo browser matrix; expect 258 route checks, 258 action
  inventories, 0 console errors, and 0 unexplained disabled/inert controls.
- [x] Commit disposition: deferred during execution, then included in the
  user-authorized final Task 11–12 handoff.

Task 11 is `verified locally`: all eight mutation fixtures were observed
failing closed before the minimum corrections; the accessibility matrix passed
5/5 with exactly 258 responsive route checks and two focused dialog checks; the
visible-action matrix passed 3/3 with exactly 258 action inventories and
route-ordered owning-test evidence for all 86 surfaces. The final gates passed
602/602 Vitest tests, exact 86-route parity and behavior-ledger checks,
typecheck, `build:demo`, root-source diff, `git diff --check`, and process
hygiene. The main-agent spec-compliance review found one Important accessible
navigation-landmark gap and the separate main-agent code-quality review found
one Important action-evidence traceability gap; both were corrected through new
RED → GREEN cycles and fresh affected gates passed. No Critical or Important
finding remains. Independent review is `not run`.

### Task 12: Run The Full Demo Matrix And Prepare Handoff

**Files**

- Create `docs/demo-evidence/REACT_86_SCREEN_DEMO_2026-07-22.md`
- Summarize the React 86-screen status in
  `docs/demo-evidence/BUILD_SUMMARY.turkce.md`; no matching standalone Turkish
  companion file exists for the English React 86-screen evidence artifact.
- Modify `docs/demo-evidence/BUILD_SUMMARY.md`
- Modify `docs/demo-evidence/BUILD_SUMMARY.turkce.md`
- Modify `docs/index.md`
- Modify `MANIFEST.md`
- Modify `docs/exec-plans/index.md`
- Modify `docs/exec-plans/tech-debt-tracker.md`
- Modify this plan

- [x] Run clean install, typecheck, all Vitest, root oracle, demo build, app-shell
  scan, 86-route boundary, mock contracts, mock Playwright, visible actions,
  accessibility, baseline integrity, and 259 visual tests (primitive gallery +
  258 route/viewport pairs).
- [ ] Inspect 86 desktop, 86 tablet, and 86 mobile candidate images beside their
  accepted baselines; record audit ID, ratios, masks, semantic/geometry/action
  result, reviewer, and disposition.
- [x] Confirm the HTTP artifact still excludes root runtime and mock/seed input;
  new routes must remain visibly unavailable in HTTP until Plan 2 rather than
  silently using demo state.
- [x] Confirm every audit row's normalized role matches its React route and
  visual fixture, including source-role-correct `ui-audit-009` and `044`.
- [x] Write synchronized evidence with exact counts and set this plan to
  `ready-for-verification` only if all local demo gates pass.
- [x] Commit/push disposition: subsequently authorized by the user for the
  final Task 11–12 handoff; no branch change or deployment was authorized.

Task 12's non-visual required matrix passed: clean install, typecheck, 602/602
Vitest, 107/107 root/parity, demo and HTTP builds, app-shell and HTTP-artifact
scans, the exact 86-route/two-profile boundary, 28/28 mock Playwright, and 3/3
visible-action tests. The original one-shot visual matrix is `not verified` at
71/259: the primitive gallery passed 1/1 and route pairs passed 70/258, with 188
failures. The original run reported 157 decoded-pixel ratio, 69 semantic
substring, and 10 target-size failures with overlapping categories. The
target-size defects were fixed through a separate RED -> GREEN cycle and the
accessibility matrix then passed 5/5 with exactly 258 responsive route checks.
On 2026-07-25, standalone baseline integrity was rerun with exact Node 24.16.0,
Playwright 1.61.1, and Chromium 149.0.7827.55; it passed with 258 baseline PNGs
verified, and the current canonical UI-audit hash matches the accepted manifest
hash
`sha256:92a8ab06da1f87fd9e84b45b35fa5c3dc58aa78a6eb7f6f9c9652731e8f74967`.
A 2026-07-25 full visual diagnostic rerun on the same exact runtime remains
`not verified` at 74/259, with 185 failures. A complete
comparison-by-comparison manual image review is `not run`. No failed comparison
is claimed as passed, and no baseline, threshold, mask, semantic truth,
authority, or root oracle was weakened. The canonical evidence is
`docs/demo-evidence/REACT_86_SCREEN_DEMO_2026-07-22.md`.

## Required Verification Matrix

```bash
npm --prefix apps/web ci
npm --prefix apps/web run typecheck
npm --prefix apps/web test
node --test tests/*.test.js tests/parity/react-legacy-parity.test.mjs
npm --prefix apps/web run build:demo
npm --prefix apps/web run build:http
npm --prefix apps/web run check:app-shell
node apps/web/scripts/assert-http-artifact.mjs apps/web/dist/http
node apps/web/scripts/assert-parity-boundary.mjs
node apps/web/scripts/verify-visual-baselines.mjs
npm --prefix apps/web run test:e2e:mock
npm --prefix apps/web run test:e2e:visible-actions
npm --prefix apps/web run test:e2e:visual-parity
```

Final scope: 86 React routes, 0 legacy-only rows, 17 dual-profile routes, 69
demo-only routes awaiting Plan 2 activation, exactly 258 responsive route
checks, exactly 258 action inventories, and 259 one-shot visual tests.

## Risks And Controls

| Risk | Control |
|---|---|
| A generic React redesign replaces the accepted demo | Exact root DOM/CSS inspection, decoded-pixel thresholds, manual pair review |
| UI invents backend behavior | Frozen typed capability contract and explicit demo-only profile metadata |
| One huge component owns many screens | Domain feature folders and one-purpose route components |
| Shared shell change regresses completed roles | Rerun every previously migrated surface after shell/topbar/navigation changes |
| 258 baselines become self-authored evidence | Root-only capture profile, hash manifest, protected sources, mutation tests, manual contact sheets |
| Demo actions hide behind toasts | Visible-action ledger requires navigation, state, file, modal, or mock canonical mutation |
| Auditee privacy leaks through shared components | Structural projections and raw object/DOM forbidden-field scans |

## Dependencies

- Accepted root demo and `UI_SCREEN_AUDIT_2026-07-19.md`.
- The completed 17-route parity result in
  `REACT_LEGACY_UI_PARITY_2026-07-22.md`.
- Approved target architecture in `MODULE_ARCHITECTURE.md`.
- Plan 2 must not begin until Tasks 1–12 are complete, this plan is
  `ready-for-verification`, and the frozen route/capability/action contracts are
  independently accepted.

## Out Of Scope

- Real Go/PostgreSQL implementations for the new 69 routes.
- Keycloak provisioning/MFA activation, ClamAV, Mailpit, Gotenberg, Caddy, and
  full Compose application runtime.
- Monitoring, alarms, backup/DR, Terraform, Terragrunt, AWS, deployment, and
  production cutover.
- External AI provider, enforcement automation, or legal/regulatory approval.

## Execution Prompt

```text
Execute docs/exec-plans/active/2026-07-22-full-react-86-screen-migration-plan.md task by task with superpowers:executing-plans. Do not dispatch subagents unless I explicitly authorize them. Work on the current branch and preserve unrelated .superpowers/, docs/demo-evidence/stakeholder/, and outputs/ content.

Treat the intact root Vanilla demo and the 86-row UI audit as the visual and behavioral oracle. Expand React from 17 to 86 routes and make every route fully functional in deterministic demo/mock mode. Correct the inherited role-crossed mappings so ui-audit-009 is CAA Inspector and ui-audit-044 is Department Manager; require source-role/route-role equality for every counted row. New routes must be explicitly demo-only until the next backend plan activates their HTTP capabilities; never fall back to mock state in the HTTP build. Preserve the 17-surface count and rerun every protected surface after shared shell changes.

Use TDD and the exact task order. Every visible action must have a tested mock/navigation/local result or a specific disabled reason. Preserve lifecycle authority, immutable versions, Auditee organization isolation, Internal CAA Note separation, and advisory-only risk/assistant behavior. Expand the tracked decoded-pixel oracle to 86 routes × 3 viewports without broad masks or relaxed thresholds.

Before each commit inspect upstream, the task allowlist, cached names, the full cached diff, and diff check. Use the exact task commit message only when Git actions are separately authorized. Do not deploy, start Plan 2, claim HTTP completeness, remove the root demo, or claim production readiness. Finish with synchronized evidence and leave stakeholder acceptance as the next todo.
```
