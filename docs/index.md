# AviaSurveil360 Docs Index

This is the canonical docs map for AviaSurveil360. Use it after the root
`AGENTS.md`, `README.md`, and `MANIFEST.md`.

## Core Surfaces

| Surface | Use |
|---|---|
| `../ARCHITECTURE.md` | Runtime surfaces, dependency direction, and high-risk invariants. |
| `PLANS.md` | Repository-native ExecPlan contract and lifecycle. |
| `agent-harness/index.md` | Agent routing, output contracts, verification, registry, and cleanup rules. |
| `operations/index.md` | Candidate-only objectives, telemetry, alerts, ownership, operational runbooks, and drill evidence. |
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
| `demo-evidence/stakeholder/PLAN1_VISUAL_CODEX_TRIAGE_2026-07-27.md` | Historical Codex triage for all 170 retained pixel failures: 160 high-confidence Accept recommendations and 10 Fix recommendations, including nine records that required explicit manual review. |
| `demo-evidence/stakeholder/PLAN1_VISUAL_STAKEHOLDER_DECISIONS_2026-07-27.md` | Completed Plan 1 stakeholder closeout ledger: 170/170 dispositions resolved, all 10 Fix records `fixed-verified-locally`, and zero manual decisions remaining. |
| `demo-evidence/stakeholder/PLANS2_4_STAKEHOLDER_DISPOSITION_2026-07-28.md` | Combined Plans 2–4 local stakeholder acceptance, canonical evidence basis, preserved historical verification boundaries, residual owner decisions, explicit AWS/release/deployment/production exclusions, and the separate Plan 5 Task 2 authorization gate. |
| `demo-evidence/FULL_BACKEND_SCENARIO_PARITY_2026-07-22.md` | Full Backend Tasks 1–12 evidence: 86 dual-profile routes, 28 Backend slices, 10 scenario families, 45 proofs, final matrix, reviews, and preserved Plan 1 gaps. |
| `demo-evidence/LOCAL_PRODUCTION_LIKE_SERVICES_2026-07-22.md` | Local production-like services evidence: scanned runtime digests, clean 86-route demo/full profiles, 10 scenario families, real MFA/MinIO/ClamAV/Mailpit/Gotenberg proofs, failure/restart, and cleanup. |
| `demo-evidence/LOCAL_RELIABILITY_AND_DR_2026-07-22.md` | Local reliability, observability, alert, dual-database backup, exact isolated restore, RPO/RTO, runbook, image/SBOM/scan, Terraform, Terragrunt, and explicit AWS `not run` evidence. |
| `demo-evidence/PLAN5_IDENTITY_DATA_FOUNDATION_2026-07-28.md` | Plan 5 Tasks 1–4 contract, normal/canonical-test artifact split, append-only membership, least-privilege Keycloak service client, provider-backed directory, revision-guarded invitation/recovery lifecycle, exact desired/provider/token/session authority, live OIDC/MFA/revocation evidence, strict RED results, fresh local verification, and explicit Tasks 5–9 boundaries. |
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
`verified locally`. The Full React candidate now owns all 86 routes in demo and
HTTP profiles, Full Backend Tasks 1–12 pass exact parity for 10 scenario
families and 45 proofs, and Local Production-Like Services Tasks 1–9 are
`verified locally`. Plan 1's 258-pair main-agent visual review is complete, its
Task 10 correction plus Tasks 11–12 evidence have clean independent acceptance,
and its retained pixel failures remain literal. Its visual stakeholder
disposition is 170/170 complete: 160 high-confidence Codex Accept records have
aggregate user authorization and all 10 Fix records are
`fixed-verified-locally`, including the Manager heatmap, Organization score,
and GM risk-matrix outcomes. Plan 1 is `completed`; manual review remaining is
zero. Plan 4
Tasks 1–9 and Task 11
add locally verified telemetry, all eight alerts, separate application/identity
backup chains, two exact isolated restores, candidate RPO/RTO, owner-scoped
runbooks, nine image/SBOM/scan bindings, and offline Terraform/Terragrunt policy
gates. Plans 1–4 are completed local `candidate-only` milestones under the
28 July 2026 combined stakeholder disposition. AWS Task 10 is optional,
unauthorized, and literally `not run`. The result is local `GO`,
`candidate-only`, and `release pending`.
These are not deployed production services. The docs do not
claim production identity federation,
production authorization operations, production storage/scanning or Evidence
records management, notification delivery, deployment, remote CI, cutover,
legacy removal, or production readiness. Those production actions remain
`blocked` pending explicit authorization and a separately approved
release/operations plan.
