# AviaSurveil360 Agent Guide

Keep this file as a routing map. Follow the linked authority for detail instead
of expanding this file.

## Repository Orientation

AviaSurveil360 is a CAA surveillance and oversight product with:

- an intact root HTML/CSS/Vanilla JavaScript demo that remains the legacy UI
  and behavior oracle;
- a `candidate-only` React/Vite application under `apps/web/`;
- a local Go/PostgreSQL candidate under `apps/api/`; and
- versioned product, plan, harness, and verification knowledge under `docs/`.

Local evidence does not establish production readiness or deployment.

## Canonical Routes

- [README](README.md) — package overview and local entry points.
- [Manifest](MANIFEST.md) — current repository inventory and scope boundary.
- [Documentation map](docs/index.md) — documentation collections.
- [Product specifications](docs/product-specs/index.md) — product behavior,
  terminology, roles, workflows, screens, and acceptance semantics.
- [Architecture](ARCHITECTURE.md) — runtime surfaces, dependency direction, and
  immutable boundaries.
- [Planning contract](docs/PLANS.md) — ExecPlan authoring and lifecycle rules.
- [Execution-plan index](docs/exec-plans/index.md) — current plan status and
  next todo.
- [Harness index](docs/agent-harness/index.md) — agent operating entrypoint.
- [Capability registry](docs/agent-harness/registry.md) — runnable commands and
  repository surfaces.
- [Verification matrix](docs/agent-harness/verification-matrix.md) — checks by
  risk and change type.
- [Output contract](docs/agent-harness/output-contract.md) — evidence labels
  and handoff language.
- [Entropy checklist](docs/agent-harness/entropy-cleanup-checklist.md) —
  recurring documentation and plan cleanup.
- [Build evidence](docs/demo-evidence/BUILD_SUMMARY.md) — historical local
  evidence and known limitations; read when evidence or status is in scope.

For migration work, read the matching active plan completely. The completed
React migration authority is the
[Full React 86-Screen Migration Plan](docs/exec-plans/completed/2026-07-22-full-react-86-screen-migration-plan.md).

## Durable Product Boundaries

- The core lifecycle is Audit Plan → Checklist → Finding → CAP → Evidence →
  CAA Review → Closure.
- CAP acceptance is not Finding closure. Closure requires accepted Evidence and
  verification, or an explicitly authorized and audit-recorded closure path.
- Preserve exact record, role, organization, route, and version identity.
- Auditee projections are organization-scoped and exclude Internal CAA Notes,
  other organizations, CAA workload, private risk scoring, and enforcement
  deliberations.
- Keep `Comment to Auditee` separate from `Internal CAA Note`.
- Evidence and published configuration history are append-only concepts; do
  not overwrite earlier versions.
- Regulatory material is a configured reference, finding basis, or expected
  Evidence—not legal advice or automatic enforcement.
- `Oversight Health Index` is advisory management information, never an
  automatic legal, enforcement, certificate, or closure decision.
- Visible controls must work, navigate exactly, create a local artifact, or be
  disabled with a specific reason. Do not use toast-only success or fake
  controls.

## Delivery And Change Boundaries

- Preserve the root demo as the accepted legacy oracle unless a task explicitly
  changes that oracle.
- Do not weaken accepted visual baselines, masks, thresholds, identity,
  authority, privacy, or semantic truth to manufacture parity.
- Keep demo, HTTP, and production claims distinct. Do not expose demo
  mock/seed state through the HTTP artifact.
- Do not begin a later numbered migration plan before its predecessor reaches
  the state required by the plan index.
- Preserve unrelated user changes and generated evidence.
- Work on the current branch. Do not create, switch, rename, or delete branches
  unless the user explicitly requests that branch action.
- Do not commit, push, deploy, initialize Git, or modify external/production
  systems unless the user explicitly authorizes that exact action.
- Use English for repository code, plans, evidence, and documentation. Turkish
  may be used in user-facing chat; do not create `.turkce.md` companion files.

## Planning

Use [docs/PLANS.md](docs/PLANS.md) for any multi-step implementation,
migration, readiness, or recovery work. Keep the active plan and
[plan index](docs/exec-plans/index.md) synchronized with actual progress.
Record durable unresolved facts in
[the technical-debt tracker](docs/exec-plans/tech-debt-tracker.md).

## Verification

Choose the smallest applicable gate from the
[verification matrix](docs/agent-harness/verification-matrix.md). Common
entry points are:

```bash
npm --prefix apps/web run typecheck
npm --prefix apps/web test
npm --prefix apps/web run build:demo
node --test tests/*.test.js tests/parity/react-legacy-parity.test.mjs
node tests/harness-docs-smoke.test.js
git diff --check
```

Browser work must use an isolated profile when practical and end with cleanup
of task-owned browser, Playwright, Vite, and test processes.

## Definition Of Done

- Requested behavior or documentation is implemented within the active scope.
- Relevant focused checks and required plan gates have fresh literal results.
- Role, identity, privacy, authority, persistence, mobile, and accessibility
  boundaries remain intact.
- Plan/index/evidence records match the actual final state.
- Claims use `verified locally`, `not run`, `blocked`, `candidate-only`,
  `release pending`, and `production-ready` literally.
- No unrelated work was overwritten and no unauthorized external action was
  taken.
