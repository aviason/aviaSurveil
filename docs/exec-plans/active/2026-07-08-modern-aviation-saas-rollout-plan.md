# Modern Aviation SaaS Rollout Implementation Plan


**Goal:** Apply a modern, lively, demo-impressive Aviation SaaS visual direction across the AviaSurveil360 frontend-only demo without weakening the CAA oversight workflow or demo boundaries.

**Architecture:** Keep the current static HTML/CSS/Vanilla JavaScript architecture. Build a shared visual system in `css/styles.css`, then apply it through existing render helpers and route views in `js/views.js`, with small controller/data changes only when a visible state or demo interaction already exists. Verification must combine Node smoke tests with rendered browser checks for clipping, overlap, asset freshness, and interaction proof.

**Tech Stack:** HTML, CSS, Vanilla JavaScript, mock data, browser-local demo state, direct Node smoke tests, local static server, Browser/Playwright-style rendered QA.

**Status:** `active` — responsive blockers remediated and verified locally on 2026-07-19; broader visual-system rollout remains active.

## 2026-07-19 Responsive Blocker Remediation Checkpoint

The complete 2026-07-18 Playwright baseline was reviewed through its 255
screenshots and role/viewport contact sheets before implementation. That review
confirmed two actionable blockers: mobile Inspector Assignments compressed its
desktop columns into unreadable vertical fragments, and mobile Executive
Planning rendered a `1136px` document inside a `390px` viewport. The registered
route inventory also gained Executive Director `Preliminary Reports`, bringing
the current coverage inventory to 86 screen states.

The two dense workbenches now preserve semantic table markup on desktop and
render readable task/plan cards through tablet and phone widths. At tablet
width, cards use a two-column grid; at phone width, they use one column. Every
card keeps the operational identity, organization, status, owner or progress,
Due Date/target, and primary action visible. Executive Planning detail tabs are
contained within their panel and no longer widen the document.

Fresh local delta evidence:

- Baseline reviewed: 85 routes x 3 viewports, 255/255 screenshots captured on
  2026-07-18 with 0 capture errors and 0 console issues.
- Delta capture: Inspector Assignments, Executive Planning, and the newly
  registered Executive Preliminary Reports at `1440x1000`, `1024x900`, and
  `390x844` (9 route/viewport checks).
- Delta result: 0 document-overflow failures, 0 console warnings/errors, and
  correct route headings in all 9 checks.
- Mobile result: Inspector Assignments and Executive Planning both report
  `scrollWidth === innerWidth === 390`.
- Tablet result: both adapted registers report `scrollWidth === innerWidth ===
  1024`; Inspector and Executive rows render in two readable columns.
- Automated result: the full local Node suite passes 66/66; all `js/*.js`
  syntax checks and `git diff --check` pass.
- Baseline evidence: `/private/tmp/aviasurveil360-ui-audit-2026-07-18/`.
- Before/after and delta evidence:
  `/private/tmp/aviasurveil360-ui-qa-2026-07-19-before/` and
  `/private/tmp/aviasurveil360-ui-qa-2026-07-19-after/`.

This checkpoint is **verified locally** and remains **demo-only**. It does not
claim production accessibility certification, real-device validation,
deployment, or stakeholder sign-off.

## 2026-07-18 Visual Audit Refresh

The role-selection / login screen is now the approved visual reference for the
rest of the demo. The rollout must carry its confident navy command surface,
clean off-white workspace, aviation blue, restrained teal/gold accents, strong
type hierarchy, and precise spacing into the application shell without turning
operational screens into marketing pages.

Fresh local evidence:

- Local URL: `http://127.0.0.1:4173/index.html`
- Routes: 85, including the Service Provider `Inspection Coordination` route
- Viewports: desktop `1440x900`, tablet `1024x768`, mobile `390x844`
- Screenshots: 255 attempted, 255 captured
- Capture errors: 0
- Console warnings/errors: 0
- Role/view normalization mismatches: 0
- Near-empty screens: 0
- Document-level overflow: 1, isolated to mobile Executive Planning
- Screenshot index: `/private/tmp/aviasurveil360-ui-audit-2026-07-18/screenshot-index.json`
- Contact sheets: `/private/tmp/aviasurveil360-ui-audit-2026-07-18/contact-sheets/`
- Automated tests at audit start: 60 passed, 0 failed

The `ui-ux-pro-max` design-system search independently supports the existing
direction: authority navy, trust blue, light operational surfaces, restrained
gold accent, flat/touch-first information design, and minimal motion. Do not add
an external font dependency solely to follow its typography suggestion; the
current local/system font stack already supports the static and offline-friendly
demo boundary.

### Audit Decision

Do not recolor or reskin every page independently. The product already uses the
right base palette. The implementation problem is that the login screen's brand
confidence, hierarchy, and rhythm fade into generic white table/card surfaces
on some internal routes. Fix shared responsive and shell primitives first, then
upgrade the highest-value operational workbenches.

### Current Strong Reference Surfaces

- Role selection / login: master brand, color, typography, and spacing reference.
- Inspector Findings: strong queue + selected dossier + lifecycle composition.
- Lead Audit Assignment: strong lifecycle and Service Provider coordination context.
- Service Provider Inspection Coordination: clear external task, safe scope, and one primary response path.
- Admin Question Bank: effective selected-record/configuration-studio pattern.

### Prioritized Gap Ledger

| Priority | Surface | Evidence-backed gap | Planned response |
|---|---|---|---|
| Blocking | Executive Planning at `390x844` | The `1120px` planning table forces a `1136px` document inside a `390px` viewport. | Replace the mobile table with stacked plan rows/cards or a deliberate contained table shell; assert `scrollWidth === clientWidth` and preserve all decisions. |
| High | Inspector My Assignments at `390x844` | The dense desktop table compresses columns until labels and values break vertically, making the queue effectively unreadable even though page-level overflow is false. | Use the established mobile work-item row pattern with title, organization, status, Due Date, and one primary action; keep secondary fields behind expansion. |
| High | Nested operational tables | Manager Dashboard `Upcoming Audits` and some organization/report tables depend on narrow internal scrolling or partially hidden action columns. | Standardize sticky/contained action cells on tablet and replace tables with stacked rows on phone where scanning suffers. Add nested clipping assertions, not only document overflow checks. |
| High | Touch and type heuristics | The audit flags many sub-44px controls and sub-12px leaf labels across dense workbenches. Counts are heuristic, not accepted defect totals, but they show a systemic density risk. | Audit shared buttons, icon buttons, pagination, tabs, table actions, and mobile inputs; enforce 44px touch areas and readable mobile label tokens without inflating desktop density. |
| High | First-screen task hierarchy | Several long review/configuration routes place the primary decision below filters, summary cards, or long tables. | Introduce a shared command header/decision bar showing owner, next action, Due Date, status, and the single primary CTA. |
| Medium | Cross-role shell consistency | Navy/blue identity is consistent, but mobile header treatment, demo strip, role identity, and page-heading rhythm vary by role family. | Consolidate shell tokens and role-accent mapping while keeping one navigation model and predictable mobile header anatomy. |
| Medium | Manager dashboards | Information is complete but large repeated KPI/card grids feel more like a generic admin dashboard than an aviation command workbench. | Reduce equal-weight cards, promote one attention strip, and use risk/lifecycle trace elements as the distinctive management signature. |
| Medium | Dense configuration/report surfaces | Admin and governance screens are functional but often rely on white tables with similar visual weight. | Reuse the Question Bank configuration-studio and Findings dossier patterns instead of adding more decorative cards. |

### Revised Delivery Order

1. Responsive blockers and QA guards: Executive Planning, Inspector Assignments,
   nested action columns, touch targets, and asset freshness.
2. Shared design system and shell: tokens, command headers, role identity,
   navigation states, typography rhythm, and mobile header.
3. Signature operational workflows: Finding/CAP/Evidence, Lead Assignment,
   Inspection Coordination, approval/decision dossiers, and Manager attention.
4. Lower-frequency configuration and advanced screens.
5. Full 85-route desktop/tablet/mobile recapture and stakeholder review.

## Global Constraints

- Keep AviaSurveil360 as the canonical product name.
- Keep the first executable artifact frontend-only and demo-only.
- Do not add backend, database, API, real authentication, real authorization enforcement, real file upload/storage, real AI service, real regulatory ingestion, real notification service, production audit log, mobile/offline production app, framework migration, e-signature, or deployment unless explicitly requested.
- Preserve the Finding -> CAP -> Evidence -> CAA Review -> Closure workflow.
- CAP acceptance is not finding closure.
- Separate `Comment to Auditee` from `Internal CAA Note`.
- Keep auditee views limited to their own organization and CAA-visible information.
- Use careful regulatory language: `reference`, `configured check`, `expected evidence`, `finding basis`, and `review result`.
- Keep source code identifiers, comments, canonical docs, file names where practical, and implementation plans in English.
- Avoid decorative orbs, generic marketing heroes, one-note palettes, fake controls, and inert button-like UI.
- For any visible UI change, verify controls update real UI state or are visibly disabled/removed.
- Bump `index.html` CSS/JS asset query tokens whenever CSS or JS behavior changes.

---

## Objective

Move the full AviaSurveil360 demo from "functional oversight prototype" to a cohesive **Modern Aviation SaaS** experience: brighter, livelier, more polished, more memorable in stakeholder demos, and still credible for a civil aviation authority workbench.

The target feeling is **Aviation Command SaaS**:

- modern and energetic, not generic startup dashboard
- operational and scan-friendly, not marketing page
- premium and crisp, not heavy enterprise grey
- colorful through meaningful status and aviation accents, not decoration
- workflow-led, with Finding/CAP/Evidence as the signature experience

## Scope

In scope:

- Global visual language: color tokens, surfaces, elevation, typography rhythm, status styling, focus states, and motion/interaction feel.
- Shell and navigation: role selector, sidebar, topbar, notification/profile controls, active nav states, and role identity.
- Shared components: command headers, metric strips, status pills, file chips, table rows, action buttons, lifecycle steppers, evidence cards, regulatory trace blocks, and decision panels.
- Inspector surfaces: My Assignments, SMS Oversight Audit, Checklist Runner, Findings, Reports, Messages, Calendar/Audit Work Queue.
- Lead Inspector surfaces: Assigned Audits, assignment questions, preliminary/final reports, final report prepare, CAP review detail.
- Department Manager / GM / Finance / Executive surfaces: dashboard, planning, approvals, audit reports, organizations, findings, CAP reviews.
- Auditee / Service Provider portal: received reports, My CAPs, communications, documents, settings.
- Admin preview: templates, question bank, checklist versions, reports, users, settings, organizations, audit log, regulatory library.
- Demo QA harness improvements that prevent the recent classes of misses: text clipping, stale assets, collision with menu buttons, hidden primary actions, and visually fake controls.
- Documentation and evidence updates for plan/index state and demo-only limitations.

Out of scope:

- Production architecture or backend build.
- Real authentication, authorization, uploads, document storage, notifications, AI, regulatory ingestion, reporting engine, audit log, or legal/enforcement automation.
- Replacing the static architecture with React, Next.js, Tailwind, shadcn, or another framework.
- Broad IA/product rewrites outside visual and interaction polish.
- Branch operations, commits, pushes, PRs, or deployment unless explicitly requested.

## Assumptions

- The current app is a static demo with all primary screens already reachable.
- The latest baseline includes attachment mock download behavior and attachment filename clipping fixes.
- The user prefers "modern SaaS / lively / impressive" over "serious premium GovTech."
- The civil aviation authority domain still requires restraint: no playful metaphors, no consumer-app gamification, no legal/regulatory overclaims.
- Existing tests are useful but insufficient for visual quality; rendered checks must be added or run as part of implementation.
- There is no `package.json`; verification uses direct `node` commands and local browser/static-server QA.

## Visual Direction

Use this direction consistently:

| Dimension | Direction |
|---|---|
| Product personality | Aviation command platform, confident, live, precise |
| Background | Light operational canvas with subtle cool tint, not dark theme |
| Primary color | Aviation blue with cyan/teal accents |
| Status colors | Clear but calmer green, amber, red, neutral blue |
| Surfaces | Crisp white panels, soft borders, purposeful elevation |
| Navigation | Dark navy command rail with cleaner active states |
| Tables | Dense but polished, stronger row hover/selected states, better chips |
| Workflow visuals | Lifecycle steppers and evidence/version cards as hero components |
| Iconography | Replace emoji-like UI marks with consistent line icons or local icon primitives |
| Motion | Small hover/press/state transitions only; no decorative animation |

## File Map

Primary implementation files:

- `index.html` - asset token bump for every CSS/JS visual behavior change.
- `css/styles.css` - visual tokens, shell styling, component system, responsive rules, clipping guards.
- `js/views.js` - route markup, component composition, reusable visual patterns.
- `js/app.js` - action wiring only where visual state needs an existing behavior to surface.
- `js/data.js` - mock labels/icons/status metadata only when needed for visual consistency.
- `js/helpers.js` - shared badge/status/file/lifecycle helpers if existing helpers are better than duplicating markup.

Test and evidence files:

- `tests/premium-ui-remediation-smoke.test.js` - broad style/markup smoke guard for premium visual primitives.
- `tests/inspector-nav-smoke.test.js` - inspector shell, SMS audit, file chip, table, and assignment guards.
- `tests/lead-inspector-workspace-smoke.test.js` - lead review/report polish guards.
- `tests/service-provider-final-report-smoke.test.js` - auditee/service provider portal guards.
- `tests/demo-boundary-smoke.test.js` - demo-only boundary checks.
- `docs/demo-evidence/RESPONSIVE_QA_2026-07-08.md` or a new dated evidence note if the implementation run happens after this plan date.
- `docs/exec-plans/index.md` - plan tracking status and next todo.

## Phases

### Phase 0 - Baseline, Token Audit, And QA Gate

Purpose: freeze the current visual state, identify all shared primitives, and make the next QA run capable of catching visual defects before push.

- [x] **Step 1: Confirm clean baseline**

  Run:

  ```bash
  git status --short
  git log -3 --oneline
  ```

  Expected: only intentional local changes, if any; latest commits include the attachment download and clipping fixes.

- [x] **Step 2: Inventory visual primitives**

  Run:

  ```bash
  rg -n "badge|pill|status|inspection-file|workbench-command|lifecycle|regulatory|table|btn|iconbtn|sidebar|topbar" css/styles.css js/views.js js/helpers.js js/app.js
  ```

  Expected: a concise list of existing primitives to consolidate rather than duplicating a new design language.

- [x] **Step 3: Capture current route/viewport baseline**

  Use a local static server:

  ```bash
  python3 -m http.server 4173 --bind 127.0.0.1
  ```

  Render the critical route matrix at minimum:

  - Inspector: My Assignments, SMS Oversight Audit, Checklist Runner, Findings
  - Lead Inspector: Assigned Audits, Assignment Questions, Final Report Prepare
  - Manager: Dashboard, Planning, Safety Intelligence, CAP Reviews
  - Auditee: Received Reports, My CAPs, Documents
  - Admin: Templates, Question Bank, Users, Audit Log

  Required viewport set:

  - `1720 x 960`
  - `1366 x 768`
  - `1024 x 768`
  - `768 x 1024`
  - `390 x 844`

  Current result (2026-07-18): 85 routes across desktop, tablet, and mobile
  produced 255 screenshots with 0 capture errors, 0 console issues, and 0
  route mismatches. One real document-overflow defect remains on mobile
  Executive Planning. Evidence is under
  `/private/tmp/aviasurveil360-ui-audit-2026-07-18/`.

- [x] **Step 4: Add or update visual QA guard expectations**

  Extend existing smoke tests before visual edits where possible. At minimum, guards must assert:

  - `.inspection-file` cannot return to `white-space: nowrap`
  - attached file names render inside `.inspection-file__name`
  - `index.html` asset token changes from the previous production token
  - primary table/action columns do not have known narrow widths after fixes
  - key route markup includes the new shared visual primitives once they are introduced

  Expected: a future regression like clipped file names breaks a test or rendered metric.

  Current result (2026-07-19): the smoke contracts assert semantic row hooks,
  field labels, phone/tablet card breakpoints, two-column tablet grids,
  contained Executive detail tabs, and the shared asset token. The focused
  tests were written or updated before the responsive implementation and fail
  if the table compression or document-overflow guards are removed.

### Phase 1 - Shared Modern Aviation Design System

Purpose: create the reusable visual layer before touching every screen.

- [ ] **Step 1: Define global tokens in `css/styles.css`**

  Add or refactor CSS custom properties for:

  ```css
  :root {
    --aviation-blue: #2563eb;
    --aviation-blue-strong: #1d4ed8;
    --aviation-cyan: #0891b2;
    --aviation-teal: #0f766e;
    --signal-amber: #d97706;
    --signal-coral: #dc2626;
    --command-navy: #07182d;
    --surface-canvas: #f6f8fc;
    --surface-raised: #ffffff;
    --line-soft: #e6ebf3;
  }
  ```

  Use existing token names where practical to avoid a disruptive rewrite; only add new names when they clarify the new direction.

- [ ] **Step 2: Build shared component classes**

  Add classes for:

  - `.saas-command`
  - `.saas-command__meta`
  - `.saas-command__actions`
  - `.saas-metric-strip`
  - `.saas-status`
  - `.saas-file-chip`
  - `.saas-lifecycle`
  - `.saas-decision-panel`
  - `.saas-signal-card`
  - `.saas-empty-state`

  Expected: existing route views can adopt these classes without adding a framework.

- [ ] **Step 3: Replace emoji-like visible UI marks where practical**

  Search:

  ```bash
  rg -n "📎|📅|💾|🔔|⚠️|✅|↩️|🧾|📨|✈️|🏢|⚙️|🎖️|🏛️|💷|🧭|📊" js css index.html
  ```

  Replace visible UI emoji in buttons, cards, and status controls with:

  - existing local SVG icon helper where available
  - simple text-free icon spans when already used
  - consistent compact labels when an icon is not necessary

  Keep emoji only in seed notification data if removal would require broader copy changes; document any retained cases.

- [x] **Step 4: Add asset token**

  Change all CSS/JS query strings in `index.html` to a new shared token, for example:

  ```text
  20260708-modern-saas
  ```

  Expected: local and deployed browsers fetch the updated CSS/JS instead of cached assets.

  Current local token: `20260719-mobile-workbench-v1`.

### Phase 2 - Shell, Navigation, And Role Identity

Purpose: make every role feel part of one polished product.

- [ ] **Step 1: Upgrade sidebar and topbar**

  Modify `css/styles.css` and `js/app.js` shell markup only as needed:

  - richer active nav state
  - more polished role identity label
  - cleaner notification/profile cluster
  - consistent icon sizing
  - no topbar action overlap at desktop/tablet widths

- [ ] **Step 2: Upgrade login/role select**

  Modify `js/views.js` role select render:

  - present roles as modern experience cards
  - keep text concise
  - avoid marketing hero layout
  - make role cards clearly clickable

- [ ] **Step 3: Verify shell routes**

  Run:

  ```bash
  node tests/inspector-nav-smoke.test.js
  node tests/lead-inspector-nav-smoke.test.js
  node tests/governance-render-smoke.test.js
  ```

  Expected: nav routes remain reachable and no restricted role route reappears.

### Phase 3 - Inspector Experience

Purpose: make the primary demo journey feel like a premium field operations workbench.

- [ ] **Step 1: Modernize Inspector My Assignments**

  Modify `js/views.js` and `css/styles.css`:

  - top "next inspection" module becomes a command card
  - assignment table rows get stronger hover/selected states
  - status/progress/due date chips use the shared SaaS status language
  - primary `Continue` action remains first-viewport visible

- [ ] **Step 2: Modernize SMS Oversight Audit**

  Modify `viewInspectorAuditExecution`:

  - command header with audit identity, lifecycle, progress, and actions
  - attached files use `.saas-file-chip` or equivalent
  - comments and compliance controls align more cleanly
  - section list looks like navigation, not a plain list

- [ ] **Step 3: Modernize Checklist Runner**

  Modify the checklist runner view:

  - active question dossier becomes the main surface
  - expected evidence and regulatory trace are visually integrated
  - finding creation is a clear next action, not a generic button

- [ ] **Step 4: Verify Inspector flow**

  Run:

  ```bash
  node tests/inspector-nav-smoke.test.js
  node tests/inspection-execution-smoke.test.js
  node tests/checklist-comment-render-smoke.test.js
  ```

  Rendered checks:

  - SMS audit Section 2 attached filenames: `clippedCount = 0`
  - primary actions visible at `1720 x 960`, `1024 x 768`, and `390 x 844`
  - no horizontal document overflow

### Phase 4 - Finding, CAP, And Evidence Signature Workflow

Purpose: make the product differentiator visually obvious.

- [ ] **Step 1: Upgrade Finding Detail**

  Modify `viewFindingDetail` and related panels:

  - lifecycle rail is the most recognizable component on the page
  - current owner, next action, due date, severity, organization, and related audit are prominent
  - `Comment to Auditee` and `Internal CAA Note` are visually distinct

- [ ] **Step 2: Upgrade CAP review and evidence review**

  Apply shared decision panels:

  - CAP acceptance copy must still say evidence is required before closure
  - evidence review card must show version, filename, status, uploaded date, and review action
  - request-more-info action remains visible and auditee-visible reason is required

- [ ] **Step 3: Upgrade evidence/file UI**

  Use consistent file chips across:

  - checklist attachments
  - CAP evidence
  - final report attachments
  - Service Provider documents

  Every chip should show:

  - icon
  - readable filename
  - action state or source context where useful
  - no clipping against adjacent menu buttons

- [ ] **Step 4: Verify lifecycle rules**

  Run:

  ```bash
  node tests/demo-boundary-smoke.test.js
  node tests/table-first-workbench-smoke.test.js
  node tests/service-provider-final-report-smoke.test.js
  ```

  Expected: CAP acceptance still does not close a finding; evidence acceptance or authorized closure remains required.

### Phase 5 - Manager, Governance, And Safety Intelligence

Purpose: make management screens feel like a live command dashboard while staying actionable.

- [ ] **Step 1: Upgrade Manager Dashboard**

  Modify manager dashboard view:

  - top command strip with exposure, delay, workload, and action
  - fewer chart-like decorations
  - stronger "Needs Attention" list
  - every metric links to or visually points toward an action/list

- [ ] **Step 2: Upgrade Safety Intelligence and Organization Risk**

  Build signal cards:

  - risk driver
  - affected organization
  - evidence/finding link
  - recommended management action
  - status and urgency

- [ ] **Step 3: Upgrade planning and approval surfaces**

  Apply one approval package pattern to:

  - Department Manager planning
  - GM planning
  - Finance review
  - Executive Director approval
  - checklist approvals
  - audit report approvals

  Each page must show current owner, next action, blocking reason, approval path, and primary decision controls.

- [ ] **Step 4: Verify governance**

  Run:

  ```bash
  node tests/planning-render-smoke.test.js
  node tests/planning-release-smoke.test.js
  node tests/planning-workspace-smoke.test.js
  node tests/report-approval-smoke.test.js
  node tests/department-preliminary-review-smoke.test.js
  ```

  Expected: approval chain behavior and role ownership remain intact.

### Phase 6 - Lead Inspector And Report Workflow

Purpose: make report preparation and lead review feel high-value rather than table-heavy.

- [ ] **Step 1: Upgrade Assigned Audits**

  Add a modern assignment command area and selected audit summary, while preserving the table for scanning.

- [ ] **Step 2: Upgrade Assignment Questions**

  Use a better split between question selection, inspector assignment, and release controls.

- [ ] **Step 3: Upgrade Preliminary/Final Report surfaces**

  Apply report package styling:

  - report sections as navigable cards/tabs
  - approval path visible
  - attachments as file chips
  - final issue state clearly separated from draft state

- [ ] **Step 4: Verify lead workflow**

  Run:

  ```bash
  node tests/lead-inspector-workspace-smoke.test.js
  node tests/lead-inspector-nav-smoke.test.js
  node tests/department-preliminary-review-smoke.test.js
  ```

  Expected: lead routes remain reachable and report approval sequence remains intact.

### Phase 7 - Auditee / Service Provider Portal

Purpose: make the external-facing experience simple, lively, and trustworthy.

- [ ] **Step 1: Preserve and polish Inspection Coordination**

Keep the announced-inspection task structure as a reference external workflow:
proposed date, Lead Inspector, location, checklist package, visible organization
scope, confirm date, or propose alternative. Keep Ad Hoc / Unannounced
inspections undisclosed in advance. Do not add real email or calendar delivery.

- [ ] **Step 2: Upgrade Received Reports**

  Use document/report cards with clear CAA status and CAP obligations.

- [ ] **Step 3: Upgrade My CAPs**

  Keep the request-center grouping:

  - Required Now
  - Waiting CAA Review
  - Returned / More Information Requested
  - Closed / Archive

  Make each item show next action, due date, finding severity, and evidence state.

- [ ] **Step 4: Upgrade Documents and Communications**

  Use consistent file chips and message thread styling. Keep auditee privacy boundaries.

- [ ] **Step 5: Verify auditee boundaries**

  Run:

  ```bash
  node tests/table-first-workbench-smoke.test.js
  node tests/inspection-coordination-smoke.test.js
  node tests/service-provider-final-report-smoke.test.js
  node tests/demo-boundary-smoke.test.js
  ```

  Expected: no internal CAA notes, other organizations, inspector workload, internal scoring, or governance-only data leaks into auditee views.

### Phase 8 - Admin And Configuration Studio

Purpose: make Admin Preview feel configured and intentional, not a leftover table set.

- [ ] **Step 1: Upgrade Templates and Question Bank**

  Apply configuration studio pattern:

  - searchable list left
  - selected item preview/editor right
  - version/reference/evidence preview visible

- [ ] **Step 2: Upgrade Users, Settings, Organizations, Audit Log**

  Add selected record/event panels where useful; do not invent production admin capabilities.

- [ ] **Step 3: Upgrade Admin Reports**

  Either make reports a catalog/admin preview or remove role-mismatched report reuse from Admin Preview navigation.

- [ ] **Step 4: Verify admin**

  Run:

  ```bash
  node tests/checklist-management-smoke.test.js
  node tests/checklist-approval-smoke.test.js
  node tests/governance-render-smoke.test.js
  ```

  Expected: checklist/template configuration behavior remains mock-only but functional.

### Phase 9 - Full Rendered QA And Evidence

Purpose: prevent "looks okay in DOM" from passing when rendered UI is clipped, stale, or visually broken.

- [ ] **Step 1: Syntax and smoke tests**

  Run:

  ```bash
  git diff --check
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
  for f in tests/*.test.js; do node "$f" || exit 1; done
  ```

  Expected: no syntax errors and all smoke tests pass.

- [ ] **Step 2: Rendered route matrix**

  Run a route/viewport matrix against local static server:

  ```bash
  python3 -m http.server 4173 --bind 127.0.0.1
  ```

  Required matrix:

  - all 85 current routes represented in the 2026-07-18 screenshot index,
    including stateful Finding/Report previews and Inspection Coordination
  - viewports `1920 x 1080`, `1720 x 960`, `1366 x 768`, `1024 x 768`, `768 x 1024`, `390 x 844`

  Required assertions:

  - no console errors or warnings
  - no framework overlay
  - no document-level overflow
  - no clipped file chips, status pills, badges, topbar actions, or table action columns
  - no button/card text overflow
  - primary action visible in first viewport for decision screens
  - asset token in runtime CSS/JS URLs matches `index.html`

- [ ] **Step 3: Manual screenshot review**

  Inspect screenshots for:

  - visual consistency across roles
  - fancy/modern SaaS feeling without generic dashboard clutter
  - no emoji-style UI artifacts in core controls
  - no accidental one-note palette
  - no nested cards or over-decorated page sections

- [ ] **Step 4: Evidence update**

  Update demo evidence with:

  - local rendered matrix result
  - screenshot folder path if generated under `/private/tmp`
  - exact commands run
  - open risks using literal labels: `verified locally`, `not run`, `release pending`

## Verification

Minimum local verification for this rollout:

```bash
git diff --check
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
for f in tests/*.test.js; do node "$f" || exit 1; done
```

Rendered verification:

- Local static server at `http://127.0.0.1:4173/`.
- Desktop/tablet/mobile route matrix for all major role surfaces.
- Screenshot evidence for at least one representative route per role after final polish.
- Specific clipping metrics for known weak points:
  - Inspector SMS Oversight Audit attached files
  - Lead assignment question table
  - CAP/evidence decision panels
  - final report attachments
  - topbar action areas

Boundary verification:

- Confirm no backend/API/auth/upload/storage/email/notification/AI/regulatory-ingestion implementation was added.
- Confirm evidence/file upload remains mock filename or browser-generated demo content only.
- Confirm auditee privacy boundaries still hold.

## Risks

- Modern SaaS styling can become too generic. Keep aviation-specific command language and lifecycle visuals.
- More color can weaken risk/status meaning. Reserve red/amber/green for real status and risk semantics.
- Broad visual edits can break existing click flows. Run targeted route tests after every phase.
- Asset cache can hide CSS/JS changes after deploy. Bump `index.html` query token on every visual behavior change.
- Visual QA can pass page-level overflow but miss inner clipping. Include element-level `scrollWidth/clientWidth` checks.
- File/function size may grow further. Prefer small helper extraction only when it reduces real duplication and matches local patterns.

## Dependencies

- Static demo files: `index.html`, `css/styles.css`, `js/*.js`.
- Product source docs under `docs/product-specs/`.
- UX principles in `docs/product-specs/ux-plan/`.
- Workflow rules in `docs/product-specs/workflows/`.
- Existing smoke tests under `tests/`.
- Browser or Playwright-style rendered QA capability.
- Stakeholder review for final visual direction.

## Ownership Boundaries

- Codex may edit static demo HTML/CSS/JS, tests, and demo/plan docs.
- Codex must keep implementation demo-only and frontend-only.
- Codex must not create branches, commit, push, deploy, open PRs, or post GitHub comments unless explicitly requested in the current task.
- Product/stakeholder owner approves whether the final visual direction is acceptable.
- Regulatory/legal owners must review any future stronger regulatory claims; this plan does not create such claims.

## Explicit Out Of Scope

- Production backend, database, API, auth, authorization, real upload/storage, real notification service, real AI, real regulatory ingestion, production audit log, e-signature, mobile/offline production app.
- Framework migration.
- Real deployment or release.
- New legal, enforcement, certificate, security, medical, or operational obligations.
- Rewriting the product scope into a broad enterprise suite.

## Task Checklist

- [x] Phase 0: Baseline, token audit, and QA gate.
- [ ] Phase 1: Shared Modern Aviation design system.
- [ ] Phase 2: Shell, navigation, and role identity.
- [ ] Phase 3: Inspector experience.
- [ ] Phase 4: Finding, CAP, and Evidence signature workflow.
- [ ] Phase 5: Manager, governance, and Safety Intelligence.
- [ ] Phase 6: Lead Inspector and report workflow.
- [ ] Phase 7: Auditee / Service Provider portal.
- [ ] Phase 8: Admin and configuration studio.
- [ ] Phase 9: Full rendered QA and evidence.

## Execution Prompt

```text
Implement docs/exec-plans/active/2026-07-08-modern-aviation-saas-rollout-plan.md task-by-task using the role-selection/login screen as the master reference for brand, color, typography hierarchy, and spacing. Keep the Modern Aviation SaaS direction lively, polished, aviation-command oriented, demo-impressive, and credible for a civil aviation authority. First fix the evidence-backed responsive blockers: mobile Executive Planning document overflow, unreadable mobile Inspector Assignments columns, nested action-column visibility, and shared touch-target/type tokens. Then consolidate the shell and command-header system before upgrading Inspector, Finding/CAP/Evidence, Manager/Governance, Lead Inspector, Service Provider Inspection Coordination, Auditee, and Admin surfaces. Preserve the frontend-only HTML/CSS/Vanilla JavaScript architecture, mock/browser-local state, Finding -> CAP -> Evidence -> CAA Review -> Closure rules, announced-versus-unannounced coordination policy, and auditee privacy boundaries. Do not add backend, database, API, real authentication, real authorization enforcement, real upload/storage, real AI, real regulatory ingestion, real notifications, production audit log, external fonts, framework migration, branch operations, commit, push, deploy, PRs, or GitHub comments unless explicitly requested. Bump index.html asset tokens when CSS/JS changes. Verify with git diff --check, node --check for all js/*.js files, all tests/*.test.js, and a fresh stateful 85-route local rendered matrix at desktop/tablet/mobile widths with document and nested clipping checks, primary-action visibility, console review, screenshot inspection, and demo-boundary checks. Update docs/exec-plans/index.md and demo evidence with literal status labels such as verified locally, not run, release pending, and demo-only.
```
