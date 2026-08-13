# AviaSurveil360 Package Manifest

This repository is a planning pack plus an intact **frontend-only static
clickable demo** and a separate `candidate-only` React/Go application. It is
not a production system.

Candidate boundary: a local Go/PostgreSQL API/worker, a separate first-party
Go OIDC service with dedicated auth PostgreSQL and authenticated STARTTLS
Mailpit, private versioned MinIO storage, real ClamAV scanning, application
Mailpit SMTP, Gotenberg PDF rendering, complete normalized MockBackend/HttpBackend
scenarios, PWA/readiness, atomic offline field/outbox persistence,
manifest-first OPFS Inspection Attachment recovery, typed foreground sync, and
all 85 current React routes in demo and HTTP are `verified locally`; the
retired fixed-ID package-builder route remains only in the historical
86-screen baseline. The Full React
migration's Task 10 correction and Tasks 11–12 evidence have clean independent
acceptance; its full visual gate remains literally `not verified`. Plan 4 adds
local telemetry and alerting, separate application/identity backup chains,
exact isolated restore and candidate RPO/RTO drills, owner-scoped runbooks,
digest-bound recovery images, and offline Terraform/Terragrunt policy gates.
Plan 1 visual stakeholder disposition is 170/170 complete; all 10 Fix records
are `fixed-verified-locally` and zero manual decisions remain.
The fresh strict baseline verifier is separately `not verified` on later
UI-audit metadata drift, while a read-only content check verifies all 258/258
baseline PNG hashes; the accepted manifest was not changed.
Plans 1–4 are completed local `candidate-only` milestones. AWS Task 10 is
optional, unauthorized, and literally `not run`.
The old connected identity/data loader and its disposable evidence were
retired after the canonical successor and first-party identity loader became
the maintained local-preprod path. The sealed package reader needed by the
canonical AGA loader remains under canonical import ownership; no duplicate
stakeholder runtime remains in the package.
The active Canonical AGA Preprod End-To-End Product successor is separately
`candidate-only`: its canonical Question Review history, immutable New Audit
selection summary, pre-materialization Lead handoff, donor-free normal HTTP
contract/artifact/binary, non-owner local-preprod runtime role, and connected
OIDC hero lifecycle through Final Report issue are `verified locally`; the
privacy-safe manual viewport qualification is recorded as accepted, while full
recursive gates remain `not run` or `blocked`. Its final Task 8 disposable
matrix verified transaction/fault/concurrency, cold-restart fingerprint
preservation, post-restart multi-role OIDC, dependency loss/recovery, worker
crash restart, donor fail-closed behavior, secret-log denial, and zero residue.
Sol XHigh accepted the implemented pre-deletion code boundary with 0
Critical/0 Important. The user then selected Task 9 `delete`; 153 tracked donor
files were removed and the full Go/React/build/artifact plus connected
fault/restart matrix passed again with no donor fallback and zero residue. The
manual three-viewport review was accepted on 2026-08-11. External preprod is no longer part
of that plan; it is `not run` in a separate paused follow-up ExecPlan.
The root Vanilla demo remains intact.

## Root Files

- `AGENTS.md` — repo-local agent instructions and source-of-truth routing.
- `CLAUDE.md` — thin Claude adapter to the canonical agent guide.
- `ARCHITECTURE.md` — runtime surfaces, dependency direction, and high-risk
  invariants.
- `README.md` — package overview.
- `MANIFEST.md` — this package inventory.
- `index.html` — frontend-only static clickable demo entry point.

## Static Prototype

- `css/styles.css` — demo styling and responsive behavior.
- `js/data.js` — mock data, status maps, and browser-only demo persistence
  boundary.
- `js/helpers.js` — shared helpers, role visibility helpers, status helpers,
  demo notifications, and rendering helpers.
- `js/approval.js` — shared mock approval-chain primitive.
- `js/planning.js` — planning approval and audit-preparation demo logic.
- `js/checklists.js` — checklist management demo logic.
- `js/inspection.js` — inspection execution and Potential Finding demo logic.
- `js/reports.js` — preliminary/final report approval demo logic.
- `js/manager-workspaces.js` — Department Manager workspace state normalization
  and lookup selectors.
- `js/work-items.js` — shared work-item shaping plus deterministic,
  organization-scoped browser-local reminder and manager-attention records.
- `js/views.js` — static demo screen rendering.
- `js/app.js` — role routing, UI action handling, mock interactions, and demo
  bootstrapping.

## Versioned Contract, React Candidate, And Go Candidate

- `api/openapi/aviasurveil360.yaml` — generated versioned full-platform
  transport contract for all frozen local candidate slices.
- `integrations/aviacore/contracts/` and
  `integrations/aviacore/contract-lock.json` — exact Task 3B local mirror of
  the independently accepted AviaCore v3 producer contract, source-path/SHA-256
  inventory, aggregate root, behavioral identity, and separate authorization
  identity. It is `candidate-only`; it is not a producer runtime or Phase 2.3
  implementation.
- `scripts/check-aviacore-contracts.sh` and
  `scripts/generate-aviacore-contract-types.mjs` — read-only default
  mirror/lock/codegen drift gates; replacing the mirror requires the explicit
  Task 3B update environment authorization.
- `apps/api/internal/aviacorecontract/v3/` — checked generated payload types
  and a local Go serializer/validator that runs the locked positive, negative,
  and branch-matrix contract vectors without importing AviaCore implementation
  code.
- `apps/api/internal/datafeed/`, migrations `000022`/`000023`/`000024`,
  `apps/api/cmd/data-feed-worker/`, `apps/api/cmd/data-feed-replay/`,
  `apps/api/cmd/data-feed-backfill/`, `apps/api/cmd/data-feed-reconcile/`,
  `scripts/test-aviacore-data-feed-publisher.sh`, and
  `scripts/test-aviacore-feed-recovery.sh` — Task 4–6 v3 event construction,
  AES-GCM payload envelope, immutable event/attempt history, fenced source and
  replay-only delivery lanes, exact digest-bound direct-mTLS publication,
  approval-bound replay/source-consistent backfill, and fail-closed synthetic
  manifest reconciliation. This is `candidate-only`; connected AviaCore
  ingestion, Phase 2.3, coordinated recovery, release, and production use are
  not established.
- `api/openapi/source/` — six deterministic source fragments for the bundled
  OpenAPI artifact.
- `api/openapi/examples/full-platform/` — closed-schema examples for the full
  platform contract.
- `api/openapi/examples/canonical/` — canonical closed-schema request and
  response examples.
- `api/openapi/tests/contract-examples.test.mjs` — OpenAPI example and Auditee
  projection checks.
- `scripts/generate-contracts.sh` and `scripts/check-contracts.sh` — checked
  TypeScript generation, lint, example validation, and drift detection.
- `tests/parity/behavior-ledger.json` — version 4 exact 86-route behavior,
  dual-profile visible-action ownership, and 613-record per-action evidence
  ledger.
- `tests/parity/react-legacy-parity.test.mjs` — executable ledger and intact
  legacy-oracle checks.
- `apps/web/` — React + TypeScript + Vite candidate with build-time-separated
  demo and HTTP entries.
- `apps/web/src/backend/` — one capability-composed `Backend`, thin typed HTTP
  adapter, transport mapping, and boundary invariants.
- `apps/web/src/mock/` — deterministic `MemoryMockStore`, mock seed, and
  `MockBackend`; reachable only from the demo build entry.
- `apps/web/src/features/` — canonical Cabin Inspection assignments,
  inspection, checklist, Finding, CAP, Evidence, report, and dashboard routes.
- `apps/web/src/app/csp-policy.ts` — build-profile-aware CSP source; production
  artifacts exclude unsafe inline/eval and wildcard sources.
- `apps/web/src/sw.ts` — version-fenced app-shell-only Service Worker; it does
  not cache authenticated API or business-record responses.
- `apps/web/src/offline/storage-readiness.ts` — thirteen-result explicit
  managed-profile gate, browser storage canaries, restart proof, exact grant
  checks, and foundation checkout snapshot boundary.
- `apps/web/src/offline/update-coordinator.ts` — positive N/N-1 compatibility,
  pending-work deferral, cross-tab owner lock/broadcast, migration pause,
  read-only recovery, and shell-only rollback policy.
- `apps/web/src/offline/opfs-inspection-attachment-store.ts` — manifest-first
  OPFS staging, bounded writes, Worker hashing, verified promotion, and disabled
  purge boundary for field Inspection Attachments.
- `apps/web/src/offline/attachment-recovery.ts` — startup manifest/path
  reconciliation, blocking missing-byte detection, quarantine metadata, and
  no-automatic-delete recovery.
- `apps/web/src/features/inspections/offline-readiness-panel.tsx` — explicit
  policy attestations, online fallback, advisory capacity, checkout result, and
  site-data-loss messaging.
- `apps/web/tests/contract/` — reusable backend contract executed against the
  deterministic mock harness and the seeded live HTTP profile.
- `apps/web/tests/e2e/canonical-scenario.spec.ts` — normalized Cabin lifecycle
  and organization-isolation browser scenario, executed unchanged under mock
  and HTTP Playwright projects.
- `apps/web/tests/e2e/full-platform-scenarios.spec.ts` — exact normalized
  transcript parity for 10 scenario families and 45 required proofs.
- `apps/web/tests/e2e/offline-*.spec.ts` — dedicated persistent-profile Chrome
  restart/server-stop startup and two-client update/site-data recovery checks.
- `apps/web/tests/e2e/release-candidate-gates.spec.ts` — dual-profile role,
  stable-reset, literal-boundary, keyboard, focus, and target-size gate.
- `apps/web/tests/e2e/legacy-visual-parity.spec.ts` — decoded-pixel primitive
  gallery plus 258 route/viewport comparisons with candidate PNG and region
  result attachments.
- `apps/web/tests/e2e/visible-action-contract.spec.ts` — accessible visible
  action inventory across all 86 surfaces at desktop, tablet, and mobile plus
  durable-outcome execution for every active route command.
- `apps/web/tests/e2e/brand-app-shell-restart.spec.ts` — stopped-origin accepted
  brand/app-shell asset recovery.
- `apps/web/tests/e2e/offline-readiness-denials.spec.ts` — real-browser
  managed-policy, persistence-denied, online-fallback, and quota checks.
- `apps/web/scripts/assert-app-shell-artifact.mjs` — generated manifest/asset,
  version marker, and forbidden Service Worker behavior gate.
- `apps/web/scripts/assert-http-artifact.mjs` — HTTP build input/public-artifact
  exclusion gate for mock, seed, and test-profile code plus app-shell policy.
- `apps/web/scripts/assert-parity-boundary.mjs` — exact route/source/build,
  comparator, viewport, attachment, and inert-control fail-closed boundary.
- `apps/web/scripts/verify-visual-baselines.mjs` — 258-image baseline manifest,
  environment, and SHA-256 verifier.
- `apps/api/go.mod` — the single Go module and pinned runtime dependencies.
- `apps/api/cmd/api/` and `apps/api/cmd/worker/` — production-shaped HTTP and
  observable worker command entry points.
- `apps/api/cmd/local-recovery-drill/` — fail-closed, test-environment-only
  exact object backup/delete/restore verifier; not a production command.
- `apps/api/internal/httpapi/security.go` — API security headers and bounded
  in-memory local-candidate rate-limit classes.
- `apps/api/internal/` — canonical domain/authority modules, module-owned
  PostgreSQL stores, same-origin OIDC/session boundary, private object-store
  adapter, Evidence and Inspection Attachment upload services, real local
  ClamAV/Gotenberg/Mailpit adapters, deterministic test scanner, and
  fail-closed local test profile.
- `apps/auth/` — first-party public OIDC/UI, private provider administration,
  credentials, MFA, recovery, signing keys, provider sessions, authority
  mirrors, idempotency receipts, redacted audit, and forward-only migrations.
- `apps/api/cmd/preprod-canonical-demo-identity-loader/` — resumable
  nine-user first-party bootstrap with fresh opaque subjects and exact
  cross-database identity/membership/assignment reconciliation.
- `apps/api/cmd/preprod-canonical-aga-loader/` and
  `apps/api/internal/preproddata/canonicalaga/` — canonical disposable exercise
  catalog loader, immutable import/provenance boundary, and sealed package
  reader extracted before physical donor deletion. These packages do not mount
  a duplicate stakeholder lifecycle in normal runtime artifacts.
- `apps/api/internal/httpapi/generated/` — checked generated Go OpenAPI types.
- `apps/api/migrations/` — forward-only PostgreSQL foundation, authority, and
  Evidence upload migrations with retained N-1 verification.
- `apps/api/sqlc.yaml` and module-owned `queries.sql` / generated store output —
  checked SQLC source and drift-controlled persistence boundaries.
- `apps/api/tests/integration/` — live PostgreSQL, MinIO, authority,
  upload, worker recovery/failure/timeout, migration, generation, and cleanup
  tests.
- `deploy/local/compose.test.yaml` — digest-pinned, isolated local PostgreSQL,
  Mailpit, ClamAV, and MinIO verification services.
- `deploy/local/compose.yaml` — profile-scoped local HTTPS gateway, React
  demo/HTTP artifacts, API/worker, first-party auth, separate application and
  auth databases, separate application and privileged auth Mailpit services,
  MinIO, ClamAV, and Gotenberg topology.
- `deploy/local/compose.local-http.yaml`,
  `deploy/local/gateway/Caddyfile.preprod.http`, and
  the default Quick Tunnel mode in `scripts/*canonical-preprod-cloudflare*` —
  separate loopback-only HTTP gateway and ownership-validated disposable
  anonymous qualification path. It carries one strict random public HTTPS
  origin into first-party OIDC, Secure cookies, private-object TLS, and exact CORS
  without modifying the canonical HTTPS gateway or configuring named
  Cloudflare, DNS, Access, AWS, or external preprod resources.
- `make preprod-cloudflare-demo-*`,
  `scripts/store-canonical-preprod-cloudflare-token.sh`,
  `scripts/store-cloudflare-tunnel-token-keychain.swift`,
  `scripts/validate-cloudflare-tunnel-token.mjs`, and
  `scripts/canonical-preprod-cloudflare-named-launcher.mjs` — explicit
  `demo.aviasurveil.com` local-origin publication path. It uses separate local
  state/project/port ownership, stores the tunnel-scoped connector credential
  only in macOS Keychain through a hidden unbounded terminal read and a native
  Security-framework write, validates its encoded connector structure before
  image builds, and supplies it to `cloudflared` through an inherited
  `/dev/fd/3` pipe rather than argv, environment, logs, or a token file. The
  operator creates the remotely managed tunnel/public-hostname route in the
  Cloudflare dashboard; repository code performs no account/DNS/Access
  mutation and external preprod remains `not run`.
- `tests/canonical-preprod-quick-tunnel.test.mjs` and
  `scripts/canonical-preprod-quick-tunnel-url.mjs` — static anonymous-tunnel,
  strict-origin, transport-separation, public-runtime-wiring, and cleanup
  contracts for the optional disposable profile.
- `tests/canonical-preprod-named-tunnel.test.mjs` — Keychain hidden-prompt,
  pipe-only connector delivery, exact named-host/runtime identity, separate
  Make profile, and fail-closed cleanup contracts for the optional stable
  hostname.
- `apps/web/tests/e2e/canonical-quick-tunnel-panels.spec.ts`,
  `scripts/test-canonical-preprod-cloudflare-panels.sh`, and
  `scripts/show-canonical-preprod-cloudflare-users.sh` — isolated nine-account
  public OIDC/Secure-cookie/role-panel qualification, Department Manager
  Question Review and 1,310-question New Audit selection, plus the validated
  task-owned credential handoff command.
- `scripts/test-canonical-preprod-fault-restart.sh` and
  `tests/canonical-preprod-fault-restart.test.mjs` — exact disposable Task 8
  local-HTTPS transaction/fault/concurrency, 1,310-question lifecycle, cold
  restart fingerprint, post-restart multi-role OIDC, required/optional
  dependency loss, worker crash recovery, donor/log denial, and zero-residue
  qualification boundary.
- `tests/local-compose-policy.test.mjs` — fail-closed local topology, image,
  secret, network, health, and Mailpit wiring contract.
- `tests/local-runtime-contract.test.mjs` — liveness/readiness, migration,
  resource, restart, exact-network, leakage, and cleanup contract.
- `scripts/local-stack.sh` and `scripts/check-local-runtime.sh` — exact
  task-owned Compose lifecycle plus required/optional failure, crash restart,
  secret-log, network, publishing, orphan, and residue verification.
- `scripts/test-http-profile.sh` — fresh Go race/generation, live API/worker,
  React contract/build, mock/HTTP Playwright, worker/outbox drain assertion,
  and task-owned cleanup profile.
- `scripts/test-preprod-connected-scenarios.sh` and
  `tests/preprod-data-boundary.test.mjs` — Task 7 real-service smoke load,
  provider-assigned subject binding, exact domain/privacy reconciliation,
  separately authorized networkless cleanup attestation, normal-artifact
  exclusion, and zero-residue gate.
- `scripts/test-canonical-preprod-fault-restart.sh` and
  `tests/canonical-preprod-fault-restart.test.mjs` — exact disposable canonical
  transaction/concurrency, 1,310-question OIDC lifecycle, cold-restart
  fingerprint, dependency loss/recovery, worker restart, donor-denial, and
  zero-residue gate used before and after physical donor deletion.
- `scripts/test-local-recovery.sh` — isolated local PostgreSQL dump/restore and
  exact private object backup/restore drill with dedicated cleanup.
- `scripts/plan1-visual-stakeholder-review.mjs` — local-only Plan 1 visual
  decision UI, decision-ledger synchronizer, and focused surface rerun helper.
- `scripts/write-plan1-visual-codex-triage.mjs` and
  `scripts/build-plan1-visual-triage-contact-sheets.mjs` — fail-closed
  170-record triage and local contact-sheet generators.
- `deploy/observability/` — candidate-only OpenTelemetry Collector,
  Prometheus, Grafana, Loki, Tempo, Alertmanager, dashboards, and alert rules.
- `deploy/recovery/` — separate encrypted application/first-party-auth
  pgBackRest topology, logically isolated backup object store, and recovery
  image.
- `apps/api/cmd/object-backup/` and `apps/api/cmd/recovery-fingerprint/` —
  deterministic object manifest and authoritative application recovery
  fingerprint commands.
- `scripts/test-observability-profile.sh` and
  `scripts/verify-backup-catalog.sh` — isolated alert and recovery-catalog
  verification profiles.
- `docs/operations/` — candidate service objectives, telemetry/alert contracts,
  ownership, operational runbooks, and gated AWS trial decisions.
- `infra/terraform/` — reusable AWS modules, bootstrap and secure local
  fixtures, native tests, and version/provider locks.
- `infra/terragrunt/` — explicit-owner AWS trial component graph and
  non-deployable validate/plan fixtures.
- `infra/policies/aws-plan.rego` and `scripts/check-terragrunt.sh` — fail-closed
  fixture plan and policy gates that perform no AWS apply or destroy.

## Smoke Tests

There is no root `package.json`; root legacy checks use Node directly. The
separate `apps/web/package.json` owns the React candidate commands.

- `tests/approval-smoke.test.js`
- `tests/audit-work-queue-smoke.test.js`
- `tests/browser-scenario-contract-smoke.test.js` — role/action authority,
  canonical CAP/Finding mutation, reason-required reopen, and closure-label
  regression coverage.
- `tests/checklist-approval-smoke.test.js`
- `tests/checklist-comment-render-smoke.test.js`
- `tests/checklist-management-smoke.test.js`
- `tests/demo-boundary-smoke.test.js`
- `tests/department-manager-findings-smoke.test.js`
- `tests/department-manager-state-smoke.test.js`
- `tests/department-preliminary-review-smoke.test.js`
- `tests/executive-director-workspace-smoke.test.js`
- `tests/finance-review-workspace-smoke.test.js`
- `tests/general-manager-workspace-smoke.test.js`
- `tests/governance-render-smoke.test.js`
- `tests/harness-docs-smoke.test.js`
- `tests/inspection-execution-smoke.test.js`
- `tests/inspection-coordination-smoke.test.js`
- `tests/inspection-lifecycle-alignment-smoke.test.js`
- `tests/inspection-team-smoke.test.js`
- `tests/unannounced-inspection-intake-smoke.test.js` — implemented focused
  coverage for Department Manager Planning intake, notice-policy persistence,
  governed materialization, idempotency, and Service Provider privacy.
- `tests/inspector-nav-smoke.test.js`
- `tests/lead-inspector-nav-smoke.test.js`
- `tests/lead-inspector-workspace-smoke.test.js`
- `tests/manager-cap-monitoring-smoke.test.js`
- `tests/manager-checklist-management-smoke.test.js`
- `tests/manager-navigation-dashboard-smoke.test.js`
- `tests/manager-report-pdf-smoke.test.js`
- `tests/manager-reports-approval-smoke.test.js`
- `tests/manager-risk-dashboard-smoke.test.js`
- `tests/manager-workspace-responsive-smoke.test.js`
- `tests/planning-release-smoke.test.js`
- `tests/planning-render-smoke.test.js`
- `tests/planning-workspace-smoke.test.js`
- `tests/premium-ui-remediation-smoke.test.js`
- `tests/report-approval-smoke.test.js`
- `tests/scenario-integrity-regression.test.js` — exact-Audit checklist,
  Potential Finding, Observation, closure, and deterministic reminder contract.
- `tests/service-provider-final-report-smoke.test.js`
- `tests/service-provider-portal-smoke.test.js`
- `tests/stakeholder-readiness-regressions.test.js`
- `tests/table-first-workbench-smoke.test.js`
- `tests/ui-screenshot-audit-remediation-smoke.test.js` — focused responsive,
  interaction, and truthful-control contract for the 86-screen visual audit.

## Agent Harness

- `docs/index.md` — canonical docs map for agent, plan, product, demo handoff,
  and demo evidence surfaces.
- `docs/PLANS.md` — repository-native ExecPlan contract and lifecycle.
- `docs/exec-plans/completed/2026-08-07-canonical-aga-preprod-end-to-end-product-plan.md`
  — completed local canonical demo plan: Tasks 1–9 and 11 are `verified
  locally`, the user-selected physical donor deletion passed requalification,
  and the manual three-viewport stakeholder review was accepted on 2026-08-11.
- `docs/exec-plans/active/2026-08-10-canonical-aga-external-preprod-release-plan.md`
  — separate paused remote-preprod follow-up; every remote action remains
  `not run` and it does not block local demo closeout.
- `docs/agent-harness/index.md` — canonical harness entrypoint for future
  agents.
- `docs/agent-harness/output-contract.md` — required status, evidence, and
  final-readout contract.
- `docs/agent-harness/registry.md` — source, plan, evidence, static demo, and
  local test registry.
- `docs/agent-harness/verification-matrix.md` — local-only verification ladder
  for docs, JS, workflow, UI, and boundary-sensitive tasks.
- `docs/agent-harness/entropy-cleanup-checklist.md` — drift and cleanup tracker
  for stale harness instructions, evidence labels, plan state, and package
  truth.

## Build Evidence And Handoff

- `docs/demo-evidence/BUILD_SUMMARY.md` — canonical demo evidence, verification
  status, and known limitations.
- `docs/demo-evidence/CANONICAL_AGA_IMPLEMENTATION_CHECKPOINT_2026-08-08.md` — current canonical AGA candidate implementation checkpoint; Tasks 8–9 and the accepted manual viewport milestone are recorded while external preprod is tracked separately.
- `docs/demo-evidence/canonical-aga-manual-review-2026-08-10/` — privacy-safe New Audit and Question Review stakeholder handoff with nine exact viewport images and its digest-bound capture receipt.
- `docs/demo-evidence/stakeholder/CANONICAL_AGA_STAKEHOLDER_DISPOSITION_2026-08-11.md` — accepted local manual-review disposition with release and production exclusions preserved.
- `docs/demo-evidence/GOVERNED_SERVICE_PROVIDER_CHECKLIST_2026-07-29.md` —
  governed-checklist local inventory, authority, browser, and recovery evidence
  with the synthetic-only claim boundary.
- `docs/demo-evidence/UI_SCREEN_AUDIT_2026-07-19.md` — canonical 86-screen
  desktop, tablet, and mobile visual-audit evidence.
- `docs/demo-evidence/BROWSER_SCENARIO_INTEGRITY_2026-07-20.md` — canonical
  real-click browser matrix, automated gate, console, screenshot, and cleanup evidence.
- `docs/demo-evidence/REACT_MOCK_SLICE_2026-07-20.md` — canonical Tasks 2-4
  React mock slice scope, transcript, local verification, and evidence limits.
- `docs/demo-evidence/GO_POSTGRES_FOUNDATION_2026-07-21.md` —
  Task 9 Go/PostgreSQL candidate foundation evidence.
- `docs/demo-evidence/CANONICAL_AUTHORITY_FOUNDATION_2026-07-21.md` — Task 10
  authority, OIDC/session, isolation, and audit evidence.
- `docs/demo-evidence/BOUNDED_UPLOAD_AND_HTTP_PARITY_2026-07-21.md` — Task 11
  bounded upload/scan, live `HttpBackend`, and shared
  mock/HTTP scenario evidence.
- `docs/demo-evidence/PWA_OFFLINE_READINESS_2026-07-21.md` —
  Task 6 app-shell caching, explicit readiness, restart survival, multi-client
  update, and actual server-stopped startup evidence.
- `docs/demo-evidence/INDEXEDDB_FIELD_STORAGE_2026-07-21.md` —
  Task 7 atomic subject-scoped field storage, causal outbox, migration, and
  pending/in-flight restart-recovery evidence.
- `docs/demo-evidence/OPFS_INSPECTION_ATTACHMENT_RECOVERY_2026-07-21.md` —
  Task 8 manifest-first OPFS staging, startup reconciliation,
  no-delete policy, and server-stopped attachment restart evidence.
- `docs/demo-evidence/IDEMPOTENT_FOREGROUND_SYNC_2026-07-21.md` — Task 12
  one-operation causal sync, exact replay, typed
  conflict, authorized pull, and foreground recovery evidence.
- `docs/demo-evidence/FIRST_PRODUCTION_ROUTE_FAMILIES_2026-07-21.md` — Task 5
  approved route-family and responsive dual-profile
  parity evidence.
- `docs/demo-evidence/LOCAL_RELEASE_CANDIDATE_2026-07-21.md` —
  Task 13 local `GO`, complete verification matrix, dependency/SBOM review,
  restore rehearsal, and explicit production blockers.
- `docs/demo-evidence/REACT_LEGACY_UI_PARITY_2026-07-22.md` —
  Task 16 exact 17/69 scope, complete local matrix, normal OIDC, offline/recovery,
  51-pair decoded-pixel/manual parity review, and stakeholder handoff.
- `docs/demo-evidence/REACT_86_SCREEN_DEMO_2026-07-22.md` —
  Full React Tasks 11–12 exact 86-route scope, 258 responsive and action
  inventories, 613 exact evidence records, 306/306 executed route controls,
  literal original and final visual results, 2026-07-25 baseline-integrity
  pass, main-agent pair review, and clean independent acceptance.
- `docs/demo-evidence/REACT_86_SCREEN_VISUAL_REVIEW_2026-07-25.md` —
  final 258-pair decoded-region ratio, zero-mask, semantic/geometry/action,
  reviewer, and disposition ledger.
- `docs/demo-evidence/stakeholder/PLAN1_VISUAL_CODEX_TRIAGE_2026-07-27.{md,json}` —
  all 170 retained pixel failures triaged into 160 high-confidence Accept
  recommendations and 10 Fix recommendations, including nine records that
  required explicit manual review.
- `docs/demo-evidence/stakeholder/PLAN1_VISUAL_STAKEHOLDER_DECISIONS_2026-07-27.{md,json}` —
  aggregate authorization, individual decisions, all 10 Fix implementations
  verified locally, 170/170 resolved dispositions, and zero manual decisions
  remaining.
- `docs/demo-evidence/FULL_BACKEND_SCENARIO_PARITY_2026-07-22.md` —
  Full Backend Tasks 1–12 exact contract/persistence/capability coverage,
  86 dual-profile routes, 10 scenario families, 45 proofs, final matrix,
  review verdicts, and preserved Plan 1 gaps.
- `docs/demo-handoff/ACCEPTANCE_CRITERIA_AND_FEEDBACK.md`
- `docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md`
- `docs/demo-handoff/CODEX_DEMO_ONLY_PROMPT.md`
- `docs/demo-handoff/FULL_MVP_BUILD_PROMPT_LATER.md`

## Product Source Documents

- `docs/product-specs/index.md` — product specs map and reading order.
- `docs/product-specs/research-and-positioning/` — market research and product
  positioning.
- `docs/product-specs/product-plan/` — product vision, MVP scope, roadmap, and module
  architecture.
- `docs/product-specs/ux-plan/` — UX principles and navigation/information architecture.
- `docs/product-specs/workflows/` — surveillance, checklist, Finding/CAP/Evidence, and
  reminder workflows.
- `docs/product-specs/modules/` — module-level planning for audit planning, checklist
  builder, findings, CAP, evidence, auditee portal, dashboards, notifications,
  organization registry, and admin configuration.
- `docs/product-specs/screen-specs/` — screen inventory and form specs.
- `docs/product-specs/data-and-rules/` — conceptual data model, status, permission,
  security, and audit rules.
- `docs/product-specs/analytics/` — Oversight Health Index, KPIs, and report catalog.
- `docs/product-specs/scenarios/` — demo scenario and other domain scenarios.
- `docs/product-specs/references/` — glossary and source notes.

Repository documentation is English-only. Turkish explanations are delivered
in user-facing handoffs rather than duplicate companion files.

## Execution Plans

- `docs/PLANS.md` — repository-native plan contract.
- `docs/exec-plans/index.md` — active execution-plan tracking index.
- `docs/exec-plans/active/` — living plans.

## Execution Plan Archive And Tracker

- `docs/exec-plans/completed/index.md` — completed and archived execution-plan
  records.
- `docs/exec-plans/completed/2026-06-29-governance-browser-qa-mobile-blocker.md`
- `docs/exec-plans/completed/2026-07-22-full-react-86-screen-migration-plan.md`
- `docs/exec-plans/completed/2026-07-22-full-backend-scenario-parity-plan.md`
- `docs/exec-plans/tech-debt-tracker.md` — durable blocker, handoff,
  accepted-risk, missing-evidence, and technical-debt tracker.

## Production Boundary

The files above support stakeholder feedback, local demo verification, and a
`candidate-only` React/Go vertical. They prove the scoped local HTTP/API,
authority, audit-event, private upload, deterministic scan, scenario contracts,
PWA/readiness, atomic field storage, OPFS attachment recovery, the historical
17-surface root-demo parity checkpoint, and the current 86-route dual-profile
backend candidate recorded in Task evidence. Plan 3 Tasks 1–9 additionally
prove local production-mode identity/MFA, private versioned storage, real
malware scanning, immutable PDF rendering, and authenticated Mailpit SMTP
delivery with retry/restart evidence, plus bounded concurrent readiness,
failure recovery, exact network membership, two clean 86-route demo/full
profile repetitions, and zero-residue stack ownership.
They do not prove production identity
federation, external email delivery, production storage/scanning or records
operations, regulatory or enforcement approval, production sync, deployment,
cutover, release, or production readiness.

The first 22 July 2026 follow-up plan now implements 86/86 React demo routes.
Its standalone baseline-integrity gate is now `verified locally` as of
2026-07-25, while its full visual gate remains literally `not verified` at
89/259 with 170/258 retained pixel failures. Plan 1 is `completed` after clean
independent acceptance and explicit stakeholder closeout. Its visual
stakeholder disposition is 170/170 complete: 160 high-confidence Codex Accept
records have aggregate user authorization and all 10 Fix records are
`fixed-verified-locally`, including the Manager heatmap, Organization score,
and GM risk-matrix outcomes. Manual review remaining is zero. Plan 2 implements
all 86 HTTP routes and complete mock/Go/PostgreSQL scenario parity. Plan 3
Tasks 1–9 are `verified locally`; the required
matrix, two final clean demo/full repetitions, and separate final main-agent
reviews pass. Plan 4 Tasks 1–9 and Task 11 are also `verified locally`: the
clean 86-route/10-scenario profile, all eight alert fixtures, exact
dual-database/object recovery, two isolated RPO/RTO drills, ten runbooks, nine
image/SBOM/scan bindings, Terraform/Terragrunt fixtures, IaC policy gates, and
zero-residue checks pass. Plans 1–4 are completed local `candidate-only`
milestones under the combined 28 July 2026 stakeholder disposition. The
platform remains `candidate-only` and `release pending`. Deployment and
production readiness are `not run`. AWS Task 10 remains optional,
unauthorized, and literally `not run`.
