# Full React 86-Screen Demo Migration Evidence

**Evidence date:** 23 July 2026

**Plan:** [Full React 86-Screen Migration](../exec-plans/active/2026-07-22-full-react-86-screen-migration-plan.md)

**Artifact boundary:** local `candidate-only` React/Vite demo and HTTP build

**Plan status:** `active`

**Release status:** `blocked`

This record covers Tasks 11 and 12. Tasks 1–10 are preserved as completed.
The Task 11–12 handoff commit and push were subsequently authorized by the
user. No branch was created or switched, nothing was deployed, and Plan 2 was
not started.

## Scope Result

- Exactly 86 React routes are registered; zero legacy-only rows remain.
- Exactly 17 routes are available in both demo and HTTP profiles.
- Exactly 69 routes are demo-only.
- Every demo-only route uses the exact blocked reason:
  `HTTP capability is unavailable until Plan 2 activates this route.`
- Source-role and route-role equality is enforced for all 86 rows.
- `ui-audit-009` remains CAA Inspector-owned.
- `ui-audit-044` remains Department Manager-owned.
- Root runtime and mock/seed inputs remain excluded from the HTTP artifact.
- Plan 2 routes remain unavailable in HTTP instead of silently using demo
  state.
- The root HTML/CSS/JavaScript oracle was not modified.
- Accepted baselines, decoded-pixel thresholds, and masks were not weakened or
  replaced.

## Task 11 Interaction And Accessibility Boundary

Task 11 is `verified locally`.

All eight required mutation fixtures were first observed failing closed at RED:

1. inert button
2. toast-only action
3. unlabelled control
4. fake dropdown
5. duplicate accessible navigation
6. missing disabled reason
7. broken deep link
8. missing mobile viewport

The GREEN result is:

| Gate | Literal result |
|---|---|
| Responsive route inventory | 258/258: 86 desktop, 86 tablet, 86 mobile |
| Visible-action inventory | 258/258: 86 desktop, 86 tablet, 86 mobile |
| Accessibility Playwright file | 5/5 passed |
| Visible-action Playwright file | 3/3 passed |
| Dialog and mobile-navigation focus checks | 2/2 passed |
| Console errors | 0 |
| Unexplained inert controls | 0 |
| Disabled controls without record-specific reasons | 0 |
| Document-overflow failures | 0 |
| Accessible navigation landmarks | exactly one per checked route |
| Accessible active navigation items | exactly one per checked route |

The original target-size helper incorrectly accepted a control when either its
width or height met the minimum. A new RED run exposed the defect plus shared
undersized links, topbar selectors, filters, and the role-selection skip
control. The helper now requires both dimensions. Fresh checks passed all 258
responsive routes with a 24px desktop/tablet and 44px mobile boundary.

The main-agent spec-compliance review found one Important accessible-navigation
landmark gap. The separate main-agent code-quality review found one Important
action-evidence traceability gap. Both were fixed through new RED → GREEN
cycles. No Critical or Important finding remains. Independent review is
`not run`.

## Task 12 Required Verification Matrix

The original Task 12 commands were run in the required order. The complete
visual matrix was originally run exactly once for the Task 12 handoff; a
2026-07-25 exact-runtime diagnostic rerun is recorded below.

| Command | Literal result |
|---|---|
| `npm --prefix apps/web ci` | passed; 158 packages installed |
| `npm --prefix apps/web run typecheck` | passed |
| `npm --prefix apps/web test` | passed: 58 files, 602/602 tests |
| `node --test tests/*.test.js tests/parity/react-legacy-parity.test.mjs` | passed: 107/107 tests |
| `npm --prefix apps/web run build:demo` | passed: 252 modules; bundle-size warning retained |
| `npm --prefix apps/web run build:http` | passed: 250 modules; bundle-size warning retained |
| `npm --prefix apps/web run check:app-shell` | passed for demo and HTTP: 144 files / 76 assets each |
| `node apps/web/scripts/assert-http-artifact.mjs apps/web/dist/http` | passed: 144 artifact files and 152 input files checked |
| `node apps/web/scripts/assert-parity-boundary.mjs` | passed: 86 routes and two build profiles |
| `node apps/web/scripts/verify-visual-baselines.mjs` | original Task 12 run failed: `source metadata mismatch for audit document hash.`; 2026-07-25 exact-runtime rerun passed: 258 baseline PNGs verified |
| `npm --prefix apps/web run test:e2e:mock` | initial RED 12/28; corrected GREEN 28/28 |
| `npm --prefix apps/web run test:e2e:visible-actions` | passed: 3/3 with exactly 258 action inventories |
| `npm --prefix apps/web run test:e2e:visual-parity` | original Task 12 run: 71/259 passed and 188/259 failed; 2026-07-25 exact-runtime diagnostic rerun: 74/259 passed and 185/259 failed |

The mock Playwright RED was caused by stale E2E assumptions about the single
accessible navigation landmark, canonical Preliminary Report progression,
source-correct Lead ownership, the Finance planning stage, link semantics, and
screen-only Finding data. Production behavior was not changed to satisfy stale
fixtures. The tests were aligned with the already verified canonical contracts,
then the complete mock matrix passed 28/28.

## One-Shot Visual Result

The literal result is `not verified`:

- Primitive gallery: 1/1 passed.
- Route/viewport comparisons: 70/258 passed and 188/258 failed.
- Total: 71/259 passed and 188/259 failed.
- 157 failed comparison reports included decoded-pixel ratio deviations.
- 69 failed comparison reports included semantic substring mismatches.
- 10 failed comparison reports included undersized touch targets at the time of
  the one-shot visual run.
- No comparison reported document overflow or a console error.

These categories overlap and therefore must not be added together. The 157
decoded-pixel reports retain the prior `accepted visual deviation`
no-pixel-chasing disposition only where no semantic or functional defect is
present; they are not converted into passes. The ten target-size defects were
fixed through the separate RED -> GREEN accessibility cycle, which subsequently
passed 258/258 responsive route checks. On 2026-07-25, a full visual diagnostic
rerun with exact Node 24.16.0, Playwright 1.61.1, and Chromium 149.0.7827.55
remained `not verified` at 74/259, with 185 failures. The latest failure pattern
still contains decoded-pixel ratio and semantic substring mismatches against the
accepted oracle. A complete manual comparison-by-comparison image review is
`not run`.

No baseline was regenerated, no mask was broadened, no threshold was relaxed,
and no semantic identity, authority, record, role, or action was altered to
manufacture a visual pass.

## Baseline Integrity Diagnosis

Standalone baseline integrity is now `verified locally` as of 2026-07-25.

The manifest records:

`sha256:92a8ab06da1f87fd9e84b45b35fa5c3dc58aa78a6eb7f6f9c9652731e8f74967`

The current canonical English UI audit hashes to:

`sha256:92a8ab06da1f87fd9e84b45b35fa5c3dc58aa78a6eb7f6f9c9652731e8f74967`

The 2026-07-25 exact-runtime rerun used Node 24.16.0, Playwright 1.61.1, and
Chromium 149.0.7827.55 and reported 258 verified baseline PNGs. The accepted
baseline manifest was not changed merely to make verification pass.

## Handoff

Task 12 execution and evidence preparation are complete, but the plan cannot be
set to `ready-for-verification` because the visual gate remains non-green:

1. baseline integrity is now `verified locally` with the exact runtime and 258
   baseline PNGs;
2. the original one-shot visual matrix is `not verified` at 71/259, and the
   2026-07-25 exact-runtime diagnostic rerun remains `not verified` at 74/259.

The plan therefore remains `active`. Plan 2 is `blocked` until this plan becomes
`ready-for-verification` and the frozen route/capability/action handoff is
independently accepted. Independent review for Tasks 11–12 is `not run`.
