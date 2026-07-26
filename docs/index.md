# AviaSurveil360 Docs Index

This is the canonical docs map for AviaSurveil360. Use it after the root
`AGENTS.md`, `README.md`, and `MANIFEST.md`.

## Core Surfaces

| Surface | Use |
|---|---|
| `agent-harness/index.md` | Agent routing, output contracts, verification, registry, and cleanup rules. |
| `exec-plans/index.md` | Active execution-plan status and one next concrete todo per plan. |
| `exec-plans/tech-debt-tracker.md` | Durable blockers, accepted risks, missing evidence, and technical debt. |
| `product-specs/index.md` | Product source documents for domain, workflow, UX, data, analytics, scenarios, and references. |
| `demo-handoff/` | Demo prompts, stakeholder acceptance criteria, and full-MVP handoff prompt. |
| `demo-evidence/BUILD_SUMMARY.md` | Current local demo evidence, known limitations, and production gaps. |
| `demo-evidence/REACT_MOCK_SLICE_2026-07-20.md` | Tasks 2-4 React mock candidate scope, canonical transcript, local gates, and explicit exclusions. |
| `demo-evidence/GO_POSTGRES_FOUNDATION_2026-07-21.md` | Task 9 one-module Go, forward-only PostgreSQL, generation, and local profile evidence. |
| `demo-evidence/CANONICAL_AUTHORITY_FOUNDATION_2026-07-21.md` | Task 10 domain authority, isolation, session/OIDC, idempotency, audit, and migration evidence. |
| `demo-evidence/BOUNDED_UPLOAD_AND_HTTP_PARITY_2026-07-21.md` | Task 11 private bounded upload, deterministic scan, live HTTP contract, and mock/HTTP parity evidence. |
| `demo-evidence/PWA_OFFLINE_READINESS_2026-07-21.md` | Task 6 app-shell-only cache, explicit readiness, restart survival, multi-client update, and server-stopped startup evidence. |
| `demo-evidence/INDEXEDDB_FIELD_STORAGE_2026-07-21.md` | Task 7 atomic subject-scoped field storage, causal outbox, v1 migration, and pending/in-flight browser restart evidence. |
| `demo-evidence/OPFS_INSPECTION_ATTACHMENT_RECOVERY_2026-07-21.md` | Task 8 manifest-first OPFS staging, startup reconciliation, no-delete policy, and server-stopped attachment restart evidence. |
| `demo-evidence/IDEMPOTENT_FOREGROUND_SYNC_2026-07-21.md` | Task 12 typed causal foreground sync, exact acknowledgement replay, authorized conflicts, attachment delivery, and restart recovery evidence. |
| `demo-evidence/FIRST_PRODUCTION_ROUTE_FAMILIES_2026-07-21.md` | Task 5 organization, planning authority/calendar, versioned configuration, reminder, audit-trail, and dual-profile responsive parity evidence. |
| `demo-evidence/LOCAL_RELEASE_CANDIDATE_2026-07-21.md` | Task 13 local `GO`, complete mock/HTTP/offline/security/restore matrix, dependency/SBOM review, and explicit production blockers. |
| `demo-evidence/REACT_LEGACY_UI_PARITY_2026-07-22.md` | Task 16 exact 17/69 scope, full candidate matrix, 51-pair decoded-pixel and manual review, OIDC/offline/recovery evidence, and stakeholder handoff. |
| `demo-evidence/REACT_86_SCREEN_DEMO_2026-07-22.md` | Full React Tasks 11–12 evidence: exact 86-route scope, 258 responsive/action inventories, 613 per-action records, 306/306 executed route controls, baseline integrity, final 89/259 visual result, and clean independent acceptance. |
| `demo-evidence/REACT_86_SCREEN_VISUAL_REVIEW_2026-07-25.md` | Final 258-pair React visual-review ledger with literal decoded-region ratios, zero-mask status, semantic/geometry/action result, reviewer, and disposition. |
| `../api/openapi/aviasurveil360.yaml` | Minimal versioned transport source for the authorized local candidate slices. |
| `../apps/web/` | Build-time-separated React/Vite mock and HTTP candidate entries with the canonical lifecycle, approved route families, PWA/readiness, atomic field storage, OPFS attachment recovery, and typed foreground sync. |
| `../apps/api/` | One-module Go API/worker candidate with canonical and planning authority, local OIDC/session, PostgreSQL stores, bounded upload/scan, configuration/audit projections, and typed sync services. |
| `../deploy/local/compose.test.yaml` | Pinned isolated PostgreSQL, Keycloak, and MinIO local verification profile. |

## Demo Boundary

AviaSurveil360 remains a planning pack with the intact frontend-only static
clickable demo plus a separate `candidate-only` React/Go vertical. A real local
Go/PostgreSQL HTTP path, pinned local Keycloak exchange, private MinIO upload,
deterministic scan worker, canonical and approved route-family mock/HTTP browser
parity, Task 6 app-shell/readiness/restart behavior, Task 7-8 field/attachment
recovery, Task 12 foreground sync, and the Task 13 complete local matrix are
`verified locally`. Task 16 verifies the initial 17 routed React surfaces
against the accepted root demo. The Full React migration now implements all 86
audited routes in deterministic demo mode: 17 are dual-profile and 69 remain
explicitly demo-only until Plan 2. Its 258-pair main-agent visual review is
complete, and the Task 10 correction plus Tasks 11–12 evidence have clean
independent acceptance. Plan 1 is `ready-for-verification`; Plan 2 is
sequencing-unblocked but remains unstarted and unauthorized. The result is
local `GO`, `candidate-only`, and `release pending`. These are not deployed
production services. The docs do not claim production OIDC/MFA,
production authorization operations, production storage/scanning or Evidence
records management, notification delivery, deployment, remote CI, cutover,
legacy removal, or production readiness. Those production actions remain
`blocked` pending explicit authorization and a separately approved
release/operations plan.
