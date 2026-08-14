# AviaSurveil360 Agent Harness Entropy Cleanup Checklist

Use this checklist for small, reviewable cleanup passes. It prevents stale
instructions, duplicated runbooks, plan-index drift, and evidence-label drift
from becoming the next agent's default context.

## Current Cleanup Items

| Item | Status | Owner | Evidence / next action |
|---|---|---|---|
| Canonical harness package exists under `docs/agent-harness/`. | verified locally | Agent executing readiness plan | Verified with `node tests/harness-docs-smoke.test.js`. |
| Old runbook is a compatibility pointer only. | verified locally | Harness maintainer | `../demo-handoff/AGENT_HARNESS_RUNBOOK.md` no longer duplicates operating rules. |
| `AGENTS.md` stays short and map-like. | verified locally | Harness maintainer | Reduced from 318 to 121 lines; adaptive harness audit/check report 0 errors and 0 warnings. |
| Repository documentation stays English-only. | recurring check | Future agents | Do not recreate `.turkce.md` companion files; Turkish remains available in chat handoffs. |
| `MANIFEST.md` lists new harness docs and smoke test. | verified locally | Agent executing readiness plan | Manifest Agent Harness and Smoke Tests sections. |
| Active plan index matches actual harness status. | ready-for-verification | Plan owner / future agents | Maintenance repair is verified locally; user sign-off remains the next lifecycle action. |
| Superseded partial adaptation remains historical, not completed by inference. | accepted current risk | Future agents | Keep continuation note linked to readiness plan. |
| Historical Plans 2–4 stakeholder links resolve through immutable provenance, not a restored deleted blob. | verified locally | Harness maintainer | `stakeholder/PLANS2_4_STAKEHOLDER_DISPOSITION_2026-07-28.md` is a provenance-only replacement with blob and deleting commit identities. |
| Custom harness authority map, operating loop, coverage, and certification routes stay complete. | verified locally | Harness maintainer | `make harness-maintenance` checks the routes without creating evidence or certification records. |
| Data's `DEBT-001` retirement stays owner-controlled. | recurring check | Integration owner | Do not recreate `shared/data/docs/agent-harness`, certification trees, or HMAC snapshots from Surveil. |
| Demo summary keeps local evidence separate from production gaps. | recurring check | Future agents | `../demo-evidence/BUILD_SUMMARY.md` after visible demo verification. |

## Recurring Drift Symptoms

- `AGENTS.md` grows into a long encyclopedia instead of routing agents to
  focused docs.
- Compatibility pointers grow back into duplicate runbooks.
- `../exec-plans/index.md` keeps an old next todo after implementation evidence
  changes.
- A plan is moved to `completed/` without user/stakeholder sign-off.
- The demo is described with production-scope language after only local checks.
- README or MANIFEST omits current prototype, test, plan, or harness files.
- Browser evidence uses hidden navigation labels instead of visible target
  content.
- A root test instruction assumes a root `package.json`; legacy root checks use
  direct Node commands while React checks use `npm --prefix apps/web`.

## Cleanup Rules

- Prefer one targeted edit to the authoritative surface over repeating the same
  warning across many files.
- If a recurring issue can be checked mechanically, add or extend a direct Node
  smoke test using built-ins only.
- Keep durable blockers and accepted risks in
  `../exec-plans/tech-debt-tracker.md` when they must survive the current plan.
- Preserve unrelated dirty worktree changes; classify them rather than
  reverting them.
- Use `verified locally`, `blocked`, and `not run` literally in status
  readouts.
- Keep demo-only and production-readiness boundaries explicit whenever a task
  touches role visibility, evidence, upload, AI, regulatory, audit-log,
  notification, offline, reporting, or security language.

## Next Adoption Check

After this readiness plan reaches `ready-for-verification`, use this package on
one small future AviaSurveil360 task and confirm the agent can answer before
editing:

- What source rule controls the task?
- What output contract applies?
- Which verification level applies?
- Where will evidence be recorded?
- What is explicitly out of scope?

If the package causes confusion, fix `index.md` or `output-contract.md` first.
Do not expand `AGENTS.md` unless a concise pointer is missing.
