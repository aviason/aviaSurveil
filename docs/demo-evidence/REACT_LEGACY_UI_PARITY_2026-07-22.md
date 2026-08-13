# React Legacy UI Parity Evidence — 22 July 2026

## Result

Task 16 is `verified locally`. The React/Go application remains
`candidate-only`; release is `release pending`. All 17 routed React surfaces
passed the desktop, tablet, and mobile visual, semantic, geometry, and visible
action gates. Manual review of all 51 legacy/React viewport pairs found each
React surface recognizably derived from the accepted root Vanilla demo rather
than a newly invented interface.

This is not stakeholder acceptance and it is not a `production-ready` claim.
The root demo remains intact. The other 69 accepted audit rows remain
legacy-only. Production deployment, traffic cutover, legacy removal, production
Identity/MFA, records policy, monitoring/on-call, and disaster recovery remain
`blocked` pending separate authorization and evidence.

## Scope And Implementation Truth

- The frozen source inventory contains 86 ordered audit rows: 17
  `react-parity` surfaces and 69 explicit legacy-only rows.
- The 17-route registry includes role selection plus 16 protected routes. No
  placeholder route was created for the remaining 69 screens.
- Potential Finding list/get, immutable CAP revision list/get, and checklist
  template version detail are real typed read verticals in mock and HTTP modes.
- The normal HTTP lane uses same-origin OIDC session, CSRF, authenticated roles,
  expiry, and logout. The deterministic canonical-header lane remains isolated.
- Auditee projections remain organization-scoped and structurally omit Internal
  CAA Note, workload, internal risk, and enforcement deliberation fields.
- CAP acceptance does not close a Finding. Official Evidence remains
  online-first and versioned; offline files remain Inspection Attachments.
- The HTTP artifact excludes root runtime CSS/JavaScript, mock/seed modules, and
  canonical-test inputs.

Task 16 corrected three stale verification assumptions without changing product
behavior: OIDC logout is scoped to the open Profile menu, the offline asset test
checks the accepted role-selector image elements, and the recovery drill
explicitly enables its canonical seed. These files were added to the reviewed
Task 16 allowlist.

## Verification Matrix

| Gate | Result |
|---|---|
| Clean install | `npm --prefix apps/web ci`: passed; 158 packages installed |
| OpenAPI/generated contracts | `./scripts/check-contracts.sh`: 8/8 passed; lint and generation passed |
| SQLC | `./scripts/check-sqlc.sh`: passed, `sqlc-check: ok` |
| Contract examples | 7/7 passed |
| Go authority/security | `go test -race -p 1 -count=1 ./...` passed inside the canonical HTTP profile after its required PostgreSQL, then-current local OIDC provider, and MinIO bring-up |
| React type/unit/contract | typecheck passed; Vitest 47 files / 282 tests passed |
| Builds and boundaries | demo and HTTP builds passed; both app-shell scans passed at 24 files / 16 assets; HTTP artifact passed at 24 files / 109 inputs; parity boundary passed at 17 routes / 2 profiles |
| Baseline integrity | all 51 tracked PNG hashes and metadata passed |
| Root oracle and ledger | 107/107 passed |
| Mock browser | 8/8 passed |
| Visible actions | 3/3 viewport tests passed; every test inventoried all 17 surfaces |
| Canonical HTTP | complete profile passed: Go race/integration, contracts, SQLC, React 282/282, HTTP contract 14/14, mock 8/8, HTTP 10/10, and worker/outbox observability |
| Normal OIDC | 1/1 passed, including external-provider login, session projection, role route, CSRF mutation, expiry boundary, and logout |
| Real offline | 7/7 passed |
| Recovery | PostgreSQL and exact private-object backup/delete/restore passed; candidate-only drill |
| Visual parity | 52/52 passed: primitive gallery plus 51 route/viewport comparisons |
| Dependencies | full and production-only npm audits both passed with 0 vulnerabilities |
| Cleanup | task-owned browser, server, container, network, and volume resources were removed |

The standalone Go command was also attempted before service orchestration. It
first encountered sandbox local-port restrictions and then correctly failed
closed when the required services were absent. That attempt is not green
evidence. The authoritative result is the same race command executed by
`scripts/test-http-profile.sh` after deterministic service bring-up.

Transient browser runs that ended in operating-system resource kills or stale
test selectors are likewise not counted as green evidence. Each affected full
command was rerun after the bounded harness correction and passed as recorded
above.

## Recovery Evidence

- PostgreSQL dump SHA-256:
  `8908103b07f78b5b098db7d01c1b186f57226e91ed20f80d1f4f47dd550dab3a`
- Restored private object: exact 47 bytes.
- Restored object SHA-256:
  `ba47f0913c1d12b747062e178b1e346a80a1bf8be2f4b645d08cf0d3cc12d08d`

This local recovery rehearsal is `candidate-only`; it does not prove production
RPO/RTO, disaster recovery, retention, or legal-hold operations.

## Visual Environment And Integrity

- Baseline source commit: `e31117b6b48a1a4549f44de4f18ba7da2fd1d340`
- Playwright `1.61.1`; Chromium `149.0.7827.55`; Node `24.16.0`
- Platform: macOS/Darwin arm64, release `25.5.0`
- Viewports: desktop `1440×900`, tablet `1024×768`, mobile `390×844`
- Browser inputs: pinned locale, timezone, color scheme, reduced motion, device
  scale, font readiness, and image readiness.
- Comparison: decoded RGBA pixels, maximum per-channel delta `40`.
- Limits: shell regions `<= 0.03`; only predeclared adapted content regions
  `<= 0.08`.
- Masks: 0 across all 51 pairs; maximum masked ratio `0`.
- Maximum observed shell ratios: sidebar `0.02968`, topbar `0.02945`.
- Maximum observed adapted ratios: content-header `0.06612`, content `0.07980`.
- Maximum strict role-selector viewport ratio: `0.00309`.
- Evidence emitted by the final run: 51 candidate viewport PNGs and 51
  machine-readable region-result records. Eighteen contact sheets were reviewed
  manually.

Tracked source hashes:

| Source | SHA-256 |
|---|---|
| Root `index.html` | `1b02a3a2f5bb459f43f8da8896b401c65336dab3f43f6aff8f6cb53132358574` |
| Root `css/styles.css` | `b8fc4b99934702fa22ad844ebf70026b9b70d94d9106a7926c39377808f757f6` |
| Root `js/app.js` | `ba68a7fde9ccd2fd246317c54fa6b778382f4d26485527761cd4ef3cccfe4c50` |
| Root `js/views.js` | `a580203b8e1e9fad53ffbfd68200a8e3dfd8919e4c20def662b0355782e12359` |
| Root `js/data.js` | `429cd3af4b92e3fd7faaf8ae787c207137940f399118a3e278f1c6c351da4df2` |
| UI audit document | `92a8ab06da1f87fd9e84b45b35fa5c3dc58aa78a6eb7f6f9c9652731e8f74967` |
| Web lockfile | `6d8de594a8e58754c0486465f0c0b368d37de99304722d51654a48397cd05743` |

## Per-Route Reviewer Record

For each table row the baseline is
`apps/web/tests/visual-baselines/react-legacy-parity/{viewport}/{surface}.png`;
the final React attachment uses `{viewport}/{surface}.png`. Ratios are ordered
`S/T/H/C` for sidebar, topbar, content-header, and content. Mobile has `T/C`.
`V` is the full viewport ratio for the strict role selector. Every viewport in
every row has mask count `0`, masked ratio `0`, geometry `passed`, semantics
`passed`, and visible actions `passed` (`G/S/A = P/P/P`). Reviewer: Codex
primary agent. Disposition for every row: `accepted-root-demo-parity`.

| Audit | Surface | Mode | Desktop ratios | Tablet ratios | Mobile ratios | G/S/A | Recognizable accepted demo? |
|---|---|---|---|---|---|---|---|
| ui-audit-076 | admin-home | adapted | S .00010 / T .02945 / H .01061 / C .07297 | S .01407 / T .01629 / H .01736 / C .06698 | T .02894 / C .06561 | P/P/P | Yes |
| ui-audit-007 | audit-detail | adapted | S .00117 / T .00156 / H .01180 / C .04849 | S .00138 / T .00156 / H .01799 / C .04286 | T .02125 / C .06447 | P/P/P | Yes |
| ui-audit-028 | audit-plan | adapted | S .00407 / T .00775 / H .00362 / C .05895 | S .00722 / T .01767 / H .01781 / C .06008 | T .02830 / C .07646 | P/P/P | Yes |
| ui-audit-066 | auditee-home | adapted | S .00085 / T .02885 / H .00939 / C .06955 | S .00344 / T .01629 / H .01736 / C .06446 | T .02653 / C .07191 | P/P/P | Yes |
| ui-audit-022 | cap-review | adapted | S .00334 / T .00096 / H .00282 / C .06682 | S .00392 / T .00096 / H .00429 / C .06790 | T .02041 / C .04169 | P/P/P | Yes |
| ui-audit-008 | checklist-runner | adapted | S .00117 / T .00156 / H .06368 / C .07980 | S .00138 / T .00907 / H .06612 / C .07200 | T .02125 / C .07547 | P/P/P | Yes |
| ui-audit-044 | evidence-review | adapted | S .00330 / T .00096 / H .03479 / C .03827 | S .00387 / T .00096 / H .02088 / C .04764 | T .02041 / C .06392 | P/P/P | Yes |
| ui-audit-059 | executive-home | adapted | S .02286 / T .02819 / H .00919 / C .06121 | S .02968 / T .01629 / H .01736 / C .06838 | T .02806 / C .07267 | P/P/P | Yes |
| ui-audit-058 | finance-home | adapted | S .01120 / T .02849 / H .00900 / C .07362 | S .02646 / T .01767 / H .01781 / C .05885 | T .02934 / C .06999 | P/P/P | Yes |
| ui-audit-009 | finding-detail | adapted | S .00334 / T .00096 / H .03723 / C .07772 | S .00392 / T .00096 / H .05673 / C .05566 | T .02041 / C .07197 | P/P/P | Yes |
| ui-audit-052 | gm-home | adapted | S .02342 / T .02855 / H .00916 / C .07137 | S .02965 / T .01629 / H .01736 / C .04302 | T .02806 / C .06952 | P/P/P | Yes |
| ui-audit-002 | inspector-home | adapted | S .00118 / T .00156 / H .00034 / C .06051 | S .00138 / T .00156 / H .00051 / C .02279 | T .02125 / C .03506 | P/P/P | Yes |
| ui-audit-013 | lead-home | adapted | S .00334 / T .00096 / H .00021 / C .04121 | S .00392 / T .00096 / H .00031 / C .03369 | T .02041 / C .05484 | P/P/P | Yes |
| ui-audit-027 | manager-home | adapted | S .00405 / T .00775 / H .00261 / C .06974 | S .00720 / T .01767 / H .01781 / C .07750 | T .02830 / C .07639 | P/P/P | Yes |
| ui-audit-041 | organization-registry | adapted | S .00395 / T .00775 / H .00371 / C .01435 | S .00708 / T .01767 / H .01781 / C .02646 | T .02830 / C .04745 | P/P/P | Yes |
| ui-audit-030 | report-preview | adapted | S .00404 / T .00775 / H .00381 / C .07750 | S .00719 / T .02121 / H .01897 / C .07894 | T .02830 / C .06587 | P/P/P | Yes |
| ui-audit-001 | role-select | strict | V .00000 | V .00000 | V .00309 | P/P/P | Yes |

The manual review covered all 17 desktop images, all 17 tablet images, and all
17 mobile images. It confirmed the accepted DEMO ribbon, role-specific
sidebar/topbar, typography, spacing, dense cards/tables, lifecycle hierarchy,
and responsive stacking. Differences are limited to truthful backend seed
values/counts inside predeclared adapted regions.

## Cleanup And Handoff

The canonical HTTP, OIDC, and recovery scripts removed their task-owned API,
worker, Vite/static server, PostgreSQL, the then-current local OIDC provider, MinIO, network, and volume
resources. The final browser process inspection found no task-owned Playwright,
webdriver, remote-debugging Chromium, or Vite residue. Unrelated user processes
and the preserved untracked workspace paths were not touched.

Plan status after Task 16: `ready-for-verification`. The next concrete todo is
explicit stakeholder review and acceptance of the 17-surface React result. Only
after that acceptance may this remediation plan be considered for completion;
production release/cutover still requires a separate approved operations plan.
