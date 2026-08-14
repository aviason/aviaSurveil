# AviaSurveil360 Agent Harness Index

This is the canonical entrypoint for AviaSurveil360 agent harness work. Read it
after `../../AGENTS.md` when the task touches repo implementation, plans,
verification, readiness, handoff, or demo evidence.

The harness does not change product scope. AviaSurveil360 contains the intact
root static demo plus separately bounded React/Go candidates. Local candidate
evidence does not establish deployment or production readiness.

The legacy static demo remains a behavior oracle where its boundary is stated;
the candidate application is React/Vite in `apps/web/` plus Go/PostgreSQL in
`apps/api/`. First-party AviaAuth is the separate `../../shared/auth/` OIDC
candidate surface. Select its focused Go/Compose checks from the verification
matrix instead of describing the repository as frontend-only.

## Source Hierarchy

1. [`../../AGENTS.md`](../../AGENTS.md) is the concise instruction router.
2. [`../../ARCHITECTURE.md`](../../ARCHITECTURE.md) maps runtime and knowledge
   boundaries.
3. [`../product-specs/index.md`](../product-specs/index.md) routes product
   behavior and terminology.
4. [`../PLANS.md`](../PLANS.md) defines the plan contract, while
   [`../exec-plans/index.md`](../exec-plans/index.md) tracks current state.
5. This package defines the machine-readable authority map, environment,
   operating loop, output, registry, verification, coverage, certification,
   evidence, and cleanup rules.
6. [`../demo-evidence/BUILD_SUMMARY.md`](../demo-evidence/BUILD_SUMMARY.md)
   records historical local evidence and known gaps.

## Harness Package Map

| File | Use it for |
|---|---|
| `output-contract.md` | Final readout shape, status labels, evidence language, and forbidden claims. |
| `config.json` | Supported machine-readable authority paths for native harness consumers. |
| `registry.md` | Repo surface inventory and task-to-source routing. |
| `environment-contract.md` | Local isolation, lifecycle, external-fixture, and recovery boundaries. |
| `operating-loop.md` | Read → change → verify → record → escalate sequence. |
| `verification-matrix.md` | Local command ladder by risk level. |
| `coverage.md` | Canonical 31-capability inventory plus local dispositions. |
| `certification.md` | Explicit non-certifying source/fixture/commit/HMAC/Data stop boundaries. |
| `evidence/` | Future authorized local evidence only; it contains no certification record now. |
| `entropy-cleanup-checklist.md` | Drift, stale-claim, plan-index, and evidence-label cleanup. |
| `../exec-plans/index.md` | Active plan state and one concrete next todo per active plan. |
| `../exec-plans/tech-debt-tracker.md` | Durable blockers, accepted risks, missing evidence, and technical debt. |
| `../demo-evidence/BUILD_SUMMARY.md` | Current demo evidence, local verification status, and limitations. |

## Task Routing Summary

| Task | First source | Harness rule |
|---|---|---|
| Status or readiness readout | `../exec-plans/index.md`, `../demo-evidence/BUILD_SUMMARY.md` | Use `output-contract.md`; separate local proof from production scope. |
| Plan creation or execution | `../../AGENTS.md`, `../exec-plans/index.md`, nearest plan | Keep the index row and next todo synchronized with the actual result. |
| Docs-only product update | Relevant product specs and glossary | Keep English canonical documentation and careful regulatory wording. |
| Static demo behavior | Relevant workflow/module docs, `../demo-evidence/BUILD_SUMMARY.md`, targeted tests | Use the smallest local verification level that covers the changed path. |
| Role, visibility, CAP, evidence, upload, AI, or regulatory copy | `../product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md` plus workflow/module docs | Treat as boundary-sensitive and review demo-only labels. |
| Harness maintenance | This package, manifest, plan index | Prefer one focused harness doc update over broad AGENTS expansion. |
| Harness certification question | `certification.md`, active plan, Workspace integration contract | Stop at the named authority boundary; do not create commits, HMAC keys, or Data harness files. |

## Demo-Only Boundary

Do not add or claim backend, database, API, real authentication, real
authorization enforcement, real upload/storage, real AI service, real
regulatory ingestion, real notification service, production audit-log behavior,
remote CI, hosted runners, paid automation, framework migration, branch actions,
commits, pushes, PRs, or GitHub comments unless the user explicitly asks for
that exact action.

Local verification is the default: direct `node`, `git diff --check`, `rg`, and
local browser QA only when a visible workflow requires it.

## Before Editing Checklist

- Which `AGENTS.md` rule or active plan controls this task?
- Which product source doc defines the domain behavior?
- Which harness output contract applies?
- Which verification level applies?
- Where will evidence or status be recorded?
- What is explicitly out of scope?

If those answers are unclear, stop and read the relevant source document before
editing.
