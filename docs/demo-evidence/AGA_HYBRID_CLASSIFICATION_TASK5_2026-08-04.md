# AGA Hybrid Classification Demo Lifecycle — Task 5 Local Evidence

Status: `verified locally`; `candidate-only`; `release pending`; `production-ready: not established`.

## Implementation

The shared HTTP-only AGA classification workspace now has five fixed role
routes, server capability gating, POST-body pagination/filtering, bounded page
cache, separate sealed Base and nullable Draft-effective state, provider
eligibility, exact classification commands, immutable candidate/reword
successor actions, stale recovery, and Admin-only generation history/reset with
the three exact CAS fields and explicit confirmation. The existing demo raw
package panel and eager all-question loader remain unchanged; the demo entry
does not import the supplemental route or client.

The HTTP backend uses the existing request closure for all supplemental calls.
The artifact scanner was written test-first: its missing-module RED was
recorded before implementation, and its positive/negative fixture suite is now
green. The HTTP source-map policy emits no body-bearing maps; the demo retains
local maps for debugging.

## Verification

- Focused Vitest command: 6 files, 41 tests passed.
- Artifact scanner fixtures: 4/4 passed.
- `npm --prefix apps/web run typecheck`: passed.
- Demo build and scan: `build:demo` passed; scanner passed for 146 files and
  184 inputs.
- HTTP build and scans: `build:http`, the AGA workspace scanner, and
  `assert-http-artifact.mjs` passed; each HTTP scan reported 82 files and 183
  inputs.
- `git diff --check`: included in the task handoff gate and passed.

No browser session, external system, real database, deployment, or production
write was used. Live shared persistence/durability remains `not run` and is a
Task 9 gate.
