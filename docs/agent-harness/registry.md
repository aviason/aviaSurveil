# AviaSurveil360 Agent Harness Registry

This registry tells agents where to look before editing and where to record
evidence afterward. It is an inventory, not a new product specification.

## Instruction Surfaces

| Surface | Purpose |
|---|---|
| `../../AGENTS.md` | Concise canonical router for scope, product boundaries, plans, verification, and change control. |
| `../../ARCHITECTURE.md` | Runtime surfaces, dependency direction, and high-risk invariants. |
| `../PLANS.md` | Repository-native ExecPlan contract and lifecycle. |
| `../../CLAUDE.md` | Thin Claude adapter that routes back to `AGENTS.md`. |
| `index.md` | Canonical harness entrypoint. |
| `output-contract.md` | Response shape, status labels, evidence wording, and forbidden claims. |
| `verification-matrix.md` | Local command ladder and risk-based verification. |
| `entropy-cleanup-checklist.md` | Drift and cleanup queue for future harness maintenance. |

## Product Source Documents

| Folder | Use when |
|---|---|
| `../product-specs/research-and-positioning/` | Market context, positioning, and product decisions. |
| `../product-specs/product-plan/` | Product vision, MVP scope, roadmap, and module architecture. |
| `../product-specs/ux-plan/` | UX principles, navigation, and role information architecture. |
| `../product-specs/workflows/` | Surveillance, checklist, Finding, CAP, evidence, reminders, and escalation workflows. |
| `../product-specs/modules/` | Module-level fields, states, actions, rules, and acceptance criteria. |
| `../product-specs/screen-specs/` | Screen inventory and form-level expectations. |
| `../product-specs/data-and-rules/` | Conceptual data model, statuses, permissions, visibility, and security rules. |
| `../product-specs/analytics/` | Oversight Health Index, KPI, and report rules. |
| `../demo-handoff/` | Demo prompt, acceptance criteria, full-MVP prompt, and applied runbook. |
| `../product-specs/scenarios/` | Demo scenario and edge-case replay paths. |
| `../product-specs/references/` | Glossary, terminology, and source notes. |
| `../regulatory-sources/` | Public-source manifest, refresh boundary, ignored full-text locators, and compact source-bound derived assessments. |

## Plans, Evidence, And Notes

| Surface | Use |
|---|---|
| `../exec-plans/index.md` | Active plan status and one next concrete todo per active plan. |
| `../exec-plans/active/2026-06-29-agent-harness-readiness-completion-plan.md` | Current harness completion plan and execution prompt. |
| `../exec-plans/active/2026-06-29-aviasurveil-harness-engineering-adaptation-plan.md` | Historical partial-adaptation record superseded by the readiness completion plan. |
| `../exec-plans/active/2026-07-20-react-vite-pwa-go-offline-first-production-plan.md` | Active authorized local-candidate task order, scope boundaries, decisions, and next todo. |
| `../exec-plans/active/2026-07-28-regulatory-source-refresh-adaptive-checklists-plan.md` | Public NAMCATS synchronization, OCR, refresh policy, adaptive checklist scope, and derived-context plan. |
| `../exec-plans/active/2026-07-31-governed-aga-checklist-intake-and-official-source-authoring-plan.md` | Candidate-only AGA intake/authoring contract, phased inventory, external archive verifier, and real-owner handoff. |
| `../exec-plans/tech-debt-tracker.md` | Durable blocker, accepted-risk, missing-evidence, and technical-debt tracker. |
| `../demo-evidence/BUILD_SUMMARY.md` | Current demo evidence, local verification status, and production gaps. |
| `../demo-evidence/PWA_OFFLINE_READINESS_2026-07-21.md` | Task 6 app-shell, readiness, restart, multi-client update, and actual offline startup evidence. |

## Static Demo Surfaces

| Surface | Purpose |
|---|---|
| `../../index.html` | Static demo entrypoint. |
| `../../css/styles.css` | Demo layout, responsive behavior, and visual treatment. |
| `../../js/data.js` | Mock data and browser-only demo persistence boundary. |
| `../../js/helpers.js` | Shared helpers, visibility helpers, and rendering helpers. |
| `../../js/approval.js` | Shared mock approval-chain primitive. |
| `../../js/planning.js` | Planning approval and audit-preparation demo logic. |
| `../../js/checklists.js` | Checklist management demo logic. |
| `../../js/inspection.js` | Inspection execution and Potential Finding demo logic. |
| `../../js/reports.js` | Preliminary/final report approval demo logic. |
| `../../js/views.js` | Static demo screen rendering. |
| `../../js/app.js` | Role routing, UI actions, mock interactions, and bootstrapping. |

## Local Smoke Tests

There is no root `package.json`; root legacy tests run directly with `node`.
The React candidate uses `apps/web/package.json`.

| Test | Main coverage |
|---|---|
| `../../tests/harness-docs-smoke.test.js` | Harness package structure, links, labels, and forbidden readiness claims. |
| `../../tests/demo-boundary-smoke.test.js` | Auditee isolation, CAP closure boundary, mock evidence filename-only behavior. |
| `../../tests/approval-smoke.test.js` | Shared approval-chain behavior. |
| `../../tests/checklist-approval-smoke.test.js` | Checklist approval workflow. |
| `../../tests/checklist-management-smoke.test.js` | Checklist management behavior. |
| `../../tests/governance-render-smoke.test.js` | Governance render surfaces. |
| `../../tests/inspection-execution-smoke.test.js` | Inspection execution and Finding lifecycle behavior. |
| `../../tests/planning-render-smoke.test.js` | Planning approval rendering. |
| `../../tests/planning-release-smoke.test.js` | Planning release behavior. |
| `../../tests/report-approval-smoke.test.js` | Report approval workflow. |
| `../../tests/audit-work-queue-smoke.test.js` | Inspector work queue behavior. |
| `../../tests/ncaa-regulatory-source-sync.test.mjs` | Bounded three-page NCAA source discovery and page-level extraction merge behavior. |
| `../../tests/ncaa-regulatory-derived-context.test.mjs` | Part 127 / Part 140 source hashes, locators, evidence pages, applicability dispositions, six-question implications, and human gates. |
| `../../tests/preprod-data-boundary.test.mjs` | Isolated one-shot loader topology, immutable authorization/control-store boundaries, MinIO policy, migration config, normal-artifact exclusion, and networkless cleanup recorder. |
| `../../tests/canonical-preprod-quick-tunnel.test.mjs` | Canonical HTTPS/default isolation, loopback HTTP override, anonymous Quick Tunnel parser/command boundary, public-origin runtime wiring, and destructive ownership-validated cleanup contract. |
| `../../tests/canonical-preprod-named-tunnel.test.mjs` | Unbounded hidden terminal input, Security-framework Keychain storage, pipe-only connector delivery, exact named-host/runtime identity, separate Make profile, and exposure-first cleanup contract. |
| `../../apps/web/tests/e2e/canonical-quick-tunnel-panels.spec.ts` | Service-Worker-controlled public OIDC login with visible provider fields, exact role/organization projection, Secure cookie pair, nine role homes, Question Review, and 1,310-question New Audit selection over the approved disposable Quick Tunnel. |
| `../../tests/aga-checklist-archive-inventory.test.mjs` | Read-only `AGA_CHECKLIST_ARCHIVE` stream/hash/central-directory inventory; never extracts or copies source bytes. |
| `../../tests/governed-checklist-intake-plan-contract.test.mjs` | Frozen lifecycle, authority, role/privacy, source-trace, and AGA limit contract. |
| `../../tests/governed-checklist-intake-security.test.mjs` | Phased fail-closed inventory and security-boundary contract. |

## Task-To-Source Routing

| Task type | Read first | Record result in |
|---|---|---|
| Harness readiness | `index.md`, completion plan, `../exec-plans/index.md` | Completion plan, plan index, this package. |
| Status readout | `../exec-plans/index.md`, relevant plan, `../demo-evidence/BUILD_SUMMARY.md` | Chat only unless repo status is stale. |
| Product docs | Relevant product spec folder and `../product-specs/references/` | Canonical English document and focused verification evidence. |
| Prototype behavior | Active plan, relevant workflow/module docs, static demo files | Targeted tests, demo summary when evidence changes. |
| Role visibility | `../product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md`, auditee portal docs | Targeted tests and demo summary if behavior changes. |
| Finding/CAP/Evidence | Workflow docs and module docs for findings, CAP, evidence | Targeted tests and plan/evidence summary. |
| UI/visual QA | UX docs, screen specs, active plan | Screenshot/browser evidence and demo summary when accepted. |
| Plan lifecycle | `../../AGENTS.md`, `../exec-plans/index.md`, target plan | Plan file, index row, and tech-debt tracker entry if a durable gap exists. |
| Regulatory source assessment | `../regulatory-sources/README.md`, tracked manifest, applicable derived JSON/Markdown, conceptual data model, active regulatory plan | Derived assessment, focused source-context test, plan/index, and durable tracker; keep full text in the ignored local vault. |
| Governed AGA intake/authoring | The active AGA plan, product specs, OpenAPI source, `../agent-harness/verification-matrix.md`, and fresh Git status | Run the phase inventory, path-driven archive verifier, bounded candidate/security tests, focused Go/React/HTTP checks, cleanup assertion, and metadata-only evidence. Keep real Form 048, owner/manager, connected-runtime, and expansion decisions `blocked`. |
| Connected preprod data | Plan 5, `../product-specs/data-and-rules/PREPROD_IDENTITY_AND_DATA_PROFILE.md`, loader/scenario packages | `./scripts/test-preprod-data-profile.sh smoke`, then `acceptance`, `realistic`, and `stress`; active versions are `smoke@1.0.0`, `acceptance@1.0.0`, `realistic@1.1.0`, and `stress@1.1.0`. Full-volume realistic/stress `1.0.0` endurance remains retained and `not run`. Synchronize Plan 5 evidence, plan index, and tracker. |
| Disposable Quick Tunnel | `../../docs/operations/runbooks/START_STOP.md`, active canonical AGA plan, `../../tests/canonical-preprod-quick-tunnel.test.mjs` | Run static/compose checks by default. Start only after explicit current public-exposure authorization; use `make preprod-cloudflare-test-panels` for the isolated nine-role browser gate and record it separately as `verified locally`, `blocked`, or `not run`. It never authorizes named tunnels, DNS, Access, AWS, or Task 10. |
| Named local-origin Cloudflare Tunnel | `../../docs/operations/runbooks/START_STOP.md`, active canonical AGA plan, `../../tests/canonical-preprod-named-tunnel.test.mjs` | Run static/security/dry-run checks before asking the operator to store the complete tunnel-scoped connector token. Confirm malformed/truncated credentials fail before image builds and diagnostics redact connector values. `make preprod-cloudflare-demo-up` requires explicit current public-exposure authorization plus a pre-created `demo.aviasurveil.com` remotely managed tunnel route. Record the live public readiness/OIDC/browser result separately as `verified locally`, `blocked`, or `not run`; this path does not authorize Cloudflare account/DNS/Access mutation, AWS, or Task 10 external preprod. |

## Authorization Boundary

The repository now contains explicitly authorized `candidate-only` React/Go,
local-database, local-OIDC, local-object-storage, upload/scan, and PWA-readiness
slices recorded by the active production-transition plan. Do not expand these
into production integrations, production deployment, real notification or
regulatory ingestion, hosted automation/remote CI, branch changes, PRs, or
GitHub comments without exact current authorization. Commit and push are allowed
only where the current user request explicitly requires them.
