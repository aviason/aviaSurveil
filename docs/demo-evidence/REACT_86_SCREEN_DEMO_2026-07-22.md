# Full React 86-Screen Demo Migration Evidence

**Evidence date:** 23 July 2026; visual-gap resolution updated 25 July 2026

**Plan:** [Full React 86-Screen Migration](../exec-plans/active/2026-07-22-full-react-86-screen-migration-plan.md)

**Artifact boundary:** local `candidate-only` React/Vite demo and HTTP build

**Plan status:** `ready-for-verification`

**Release status:** `release pending`; production deployment remains `blocked`

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

All eight required interaction mutations plus four acceptance-review
mutations were observed failing closed at RED against the actual Playwright
harness, fixture registry, and production sources:

1. inert button
2. toast-only action
3. unlabelled control
4. fake dropdown
5. duplicate accessible navigation
6. missing disabled reason
7. broken deep link
8. missing mobile viewport
9. removed action-harness contract
10. skipped stateful/form-control execution
11. removed visible-focus contract
12. skipped mobile-only command execution

The GREEN result is:

| Gate | Literal result |
|---|---|
| Responsive route inventory | 258/258: 86 desktop, 86 tablet, 86 mobile |
| Visible-action inventory | 258/258: 86 desktop, 86 tablet, 86 mobile |
| Exact per-action evidence | 613/613: 496 route-specific plus 117 shared records |
| Executed route controls | 306/306: 303 desktop-visible plus 3 mobile-only controls |
| Route-control-free surfaces | exactly 2: `executive-preliminary-reports` and `admin-configurations` |
| Accessibility Playwright file | 5/5 passed |
| Visible-action Playwright file | 4/4 passed |
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
control. The helper now requires both dimensions. The accessibility runner
traverses the complete Tab sequence, checks focus visibility, verifies inert
mobile-sidebar behavior, and retains focus-trap/return checks. Fresh checks
passed all 258 responsive routes with a 24px desktop/tablet and 44px mobile
boundary.

The main-agent spec-compliance and code-quality reviews found actionable
navigation, evidence, stateful-control, focus-indicator, and mobile-only
execution gaps. Each was fixed through a new RED → GREEN cycle. The final
independent Task 11 re-review accepted the 613-record ledger and exact
desktop/mobile execution of all 306 route controls with no remaining Critical,
Important, or Minor finding.

## Task 12 Required Verification Matrix

The original Task 12 commands were run in the required order. The complete
visual matrix was originally run exactly once for the Task 12 handoff; a
2026-07-25 exact-runtime diagnostic rerun is recorded below.

| Command | Literal result |
|---|---|
| `npm --prefix apps/web ci` | passed; 158 packages installed |
| `npm --prefix apps/web run typecheck` | passed |
| `npm --prefix apps/web test` | original Task 12 run passed: 58 files, 602/602 tests; two consecutive final 2026-07-25 reruns passed: 58 files, 607/607 tests each |
| `node --test tests/*.test.js tests/parity/react-legacy-parity.test.mjs` | passed: 107/107 tests |
| `npm --prefix apps/web run build:demo` | passed: 252 modules; bundle-size warning retained |
| `npm --prefix apps/web run build:http` | passed: 250 modules; bundle-size warning retained |
| `npm --prefix apps/web run check:app-shell` | passed for demo and HTTP: 144 files / 76 assets each |
| `node apps/web/scripts/assert-http-artifact.mjs apps/web/dist/http` | passed: 144 artifact files and 152 input files checked |
| `node apps/web/scripts/assert-parity-boundary.mjs` | passed: 86 routes and two build profiles |
| `node apps/web/scripts/verify-visual-baselines.mjs` | original Task 12 run failed: `source metadata mismatch for audit document hash.`; 2026-07-25 exact-runtime rerun passed: 258 baseline PNGs verified |
| `npm --prefix apps/web run test:e2e:mock` | initial RED 12/28; final corrected GREEN 30/30 |
| `npm --prefix apps/web run test:e2e:visible-actions` | passed: 4/4 with exactly 258 action inventories, 613 exact per-action evidence records, and 306/306 executable route controls |
| `npm --prefix apps/web run test:e2e:visual-parity` | original Task 12 run: 71/259 passed and 188/259 failed; first 2026-07-25 diagnostic: 74/259 passed and 185/259 failed; final corrected exact-runtime run: 89/259 passed and 170/259 retained pixel failures |

The mock Playwright RED was caused by stale E2E assumptions about the single
accessible navigation landmark, canonical Preliminary Report progression,
source-correct Lead ownership, the Finance planning stage, link semantics, and
screen-only Finding data. Production behavior was not changed to satisfy stale
fixtures. The tests were aligned with the already verified canonical contracts,
then the final complete mock matrix passed 30/30.

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
remained `not verified` at 74/259, with 185 failures. That intermediate
checkpoint contained decoded-pixel ratio and semantic
substring mismatches against the accepted oracle. The final corrected result
and complete manual comparison-by-comparison image review are recorded in the
gap-resolution section below.

No baseline was regenerated, no mask was broadened, no threshold was relaxed,
and no semantic identity, authority, record, role, or action was altered to
manufacture a visual pass.

## 25 July Visual-Gap Resolution

The follow-up kept the accepted root oracle, baseline manifest, decoded-pixel
thresholds, and zero-mask contract intact.

RED -> GREEN corrections addressed the real harness/UI defects:

1. worker-global attachment counters were replaced with a pair-local
   fail-closed guard requiring exactly one candidate PNG and one decoded-region
   result per route/viewport test;
2. immutable legacy capture semantics were separated from canonical React
   candidate semantics, removing 69 stale legacy-text failures without changing
   the root fixture;
3. Auditee and Executive report previews adopted the shared page-header
   contract, and their heading metadata spacing was corrected across desktop,
   tablet, and mobile;
4. the accessibility gate now traverses the complete keyboard order and checks
   mobile-sidebar inertness, focus visibility, topbar separation, touch targets,
   and overflow;
5. every active route command now has exact source-backed action evidence and a
   durable visible outcome; and
6. unavailable Report Queue filters now render with truthful disabled styling.

The final exact-runtime result is:

- Primitive gallery: 1/1 passed.
- Route/viewport pairs: 88/258 passed and 170/258 retained decoded-pixel
  failures.
- Overall: 89/259 passed and 170/259 failed.
- Candidate PNG attachments: 258/258, exactly one per pair.
- Decoded-region attachments: 258/258, exactly one per pair.
- Semantic errors: 0.
- Other non-pixel errors: 0.
- Masks: 0.

The main agent inspected all 11 contact sheets covering the 86 desktop, 86
tablet, and 86 mobile candidates, then opened selected full-size candidates
across Inspector, Lead Inspector, Manager, Auditee, Evidence, and report
surfaces. No blank route, broken shell, incoherent overlap, unintended
clipping, missing primary purpose/action, or Auditee cross-organization
disclosure was found. The final mobile report-preview candidate was inspected
again after the disabled-filter correction.

The 170 non-green pairs remain literal pixel failures. Their main-agent
disposition accepts them only as intentional content-adapted React
compositions; it does not convert them into decoded-pixel passes. The complete
per-pair ratios, masks, semantic/geometry/action result, reviewer, and
disposition are in
[React 86-Screen Visual Review Ledger](REACT_86_SCREEN_VISUAL_REVIEW_2026-07-25.md).
The independent Task 12 review accepted the literal visual evidence, artifact
hash, 258/258 candidate and region-result counts, zero semantic/non-pixel
errors, and retained 170 pixel failures without weakening the oracle.

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

Task 12 execution, visual-gap resolution, main-agent evidence preparation, and
independent acceptance are complete. The plan is `ready-for-verification`
while the pixel gate remains literally non-green:

1. baseline integrity is now `verified locally` with the exact runtime and 258
   baseline PNGs;
2. the final exact-runtime matrix is `not verified` at 89/259, with 170
   decoded-pixel failures retained;
3. all 11 candidate contact sheets and selected full-size candidates were
   manually reviewed, and the independent Task 12 reviewer accepted the
   evidence while preserving all 170 failures as failures.

The plan is therefore `ready-for-verification`. The Task 10 correction and
Tasks 11–12 evidence received clean independent acceptance on 2026-07-25.
Plan 2 is now sequencing-unblocked but remains unstarted and unauthorized until
the user directs its first task. Stakeholder review/sign-off remains the next
todo before this plan can move to `completed`.
