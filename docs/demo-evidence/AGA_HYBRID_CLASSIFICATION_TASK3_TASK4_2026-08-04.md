# AGA Hybrid Classification Demo Lifecycle — Tasks 3–4 Local Evidence

Status: `verified locally`; `candidate-only`; `release pending`; `production-ready: not established`.

## Task 3

The sibling `preprod_aga_demo_workspace` contract, least-privilege role DDL,
one-shot fixture/export/load commands, append-only memory store, Compose
profiles/secrets, and boundary tests were implemented. The focused Go package
and command tests, boundary Node test (3/3), `go vet`, Compose config, Compose
policy (21/21), and `git diff --check` passed locally.

The combined Task 3 Node command passed 21 tests and retained five pre-existing
Gate 0/Task 2 failures caused by the absent external System Design Matrix
workbook and accepted predecessor reconstruction diagnostics. No external
system or real database was accessed. Live PostgreSQL persistence, grants, and
zero-delta evidence remain `not run` and belong to Task 9.

## Task 4

The exact operation matrix, neutral CSRF-aware protector, workspace service,
separate reader/command tagged pools, fixed supplemental routes, OpenAPI
bundler behavior, generated Go/TypeScript contracts, and legacy five-route
boundary were implemented. Focused service/HTTP Go tests passed, including
`TestWorkspaceProtectorAuthenticatesCSRFFirst`, classification/lifecycle
authorization matrices, direct-ID neutral denial, idempotency-before-domain
lookup, and replay behavior.

The generated contract gate passed `npm --prefix apps/web run contracts:check`;
the focused OpenAPI suite passed 20/20; the default and `preproddemo` API tests
passed; and the tagged API binary built locally. The workspace lifecycle
capability intentionally reports unavailable until Task 7's append-only state
machine is implemented; it does not return a fake lifecycle success.

Connected role behavior, live persistence, browser execution, and canonical
zero-delta evidence remain `not run`.
