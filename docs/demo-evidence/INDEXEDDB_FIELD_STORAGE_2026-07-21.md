# IndexedDB Field Storage And Outbox Evidence — 2026-07-21

## Outcome

- Evidence status: `verified locally`
- Artifact status: `candidate-only`
- Release status: `release pending`
- Production deployment, traffic cutover, legacy removal, production MDM/local-data controls, and a `production-ready` claim: `blocked`
- Task 8 OPFS Inspection Attachment bytes, Task 12 network sync/conflict delivery, Task 5 wider route migration, and Task 13 final local release-candidate gate: `not run`

Task 7 replaces the Task 6 checkout-only IndexedDB foundation with the complete subject-scoped field working set and typed causal outbox. The intact root Vanilla JavaScript demo remains the behavioral reference. This task does not add OPFS attachment bytes or send an outbox operation to the server.

## Test-First Evidence

The field tests were first run against absent `db`, schema-migration, repository, and outbox modules. The red/green cycle then caught and corrected:

- a SHA-256 Promise that allowed a Dexie transaction to commit before its outbox row was written;
- an equivalent external-Promise risk during the released v1-to-v2 migration;
- an invalid recovery-test clock that attempted to accept a future-issued grant;
- local writes that could otherwise proceed after grant expiry;
- a second active response identity for one immutable package question; and
- React field-state polling that initially observed the pre-commit projection.

The final implementation uses Dexie transaction tracking for cryptographic work and tests every declared entity/outbox failure boundary.

## Verified Storage And Authority Boundary

- Schema v2 contains subject-compound stores for packages, offline grants, checklist responses, Potential Finding drafts, attachment manifests, outbox operations, and sync cursors while retaining the Task 6 foundation store for safe migration/recovery.
- Checkout validates the exact subject, organization, device, package ID/version/digest, assignment scope, positive N/N-1 package schema, protocol, issue time, grant expiry, and package expiry.
- Repository mutation methods never accept an actor ID. The repository is bound once to the session subject supplied at construction, and all primary keys and queries remain subject-scoped. The current canonical route supplies its fixed test subject; Task 5 still owns wider authenticated role/session wiring.
- A checklist response, Potential Finding draft, or checklist submission and its typed outbox operation commit atomically or both abort. Injected quota/termination failures leave the prior snapshot unchanged.
- `NOT_CHECKED` is persisted as a real answer. Immutable allowed-answer, assignment, and comment-required rules are enforced locally; another Inspector's question is disabled/read-only.
- Unsent response edits supersede earlier unsent rows. An `IN_FLIGHT` request body and digest remain unchanged; a later edit is stored as `BLOCKED_ON_DEPENDENCY` against that operation.
- Potential Finding and checklist-submit commands retain causal dependencies. Duplicate operation replay is idempotent, while reuse with a changed payload fails closed.
- Pull changes and their cursor commit atomically. Out-of-scope changes fail the transaction. Logout/user switch locks without deleting records; expiry locks, and revocation/corruption/incompatibility quarantines preserved recovery data.
- Submitted checklists are read-only locally. The UI states that a reasoned reopen requires reconnection and is not an offline command.
- No `localStorage` business-record write, mock-store write from a component, or logout deletion was introduced.

## Migration And Restart Recovery

- Every released schema is covered: the released Task 6 v1 checkout snapshot upgrades to the full v2 field schema while the legacy foundation record remains available for recovery.
- Failure injection at `before-expand`, `after-expand`, `after-copy`, and `before-contract` opens read-only recovery and preserves v1 data.
- Positive N/N-1 compatibility rejects zero, future, and older-than-N-1 schema versions.
- A dedicated persistent-Chrome test performs the required browser close/reopen, stops the origin server, commits an offline response, changes its first operation to `IN_FLIGHT`, commits a later causally blocked edit, closes Chrome, and reopens the same profile while the server remains stopped.
- The restarted field screen restores the latest response and exact `Saved locally — sync pending (2)` state. IndexedDB still contains one immutable in-flight operation, its blocked successor, and the pending response.
- The two Task 6 browser tests remain green: real server-stopped startup and two-page N/N-1 app-shell update preservation.

## Fresh Verification

The following completed with exit code `0` unless an audit result is explicitly stated:

- `npm --prefix apps/web run typecheck`
- `npm --prefix apps/web test`: 14 files, 97/97 tests
- focused FieldRepository suite: 20/20 tests
- `npm --prefix apps/web run test:e2e:offline`: 3/3 real persistent-Chrome tests
- `./scripts/test-http-profile.sh`: full Go race/live PostgreSQL, then-current local OIDC provider, and MinIO integration; OpenAPI 5/5; SQLC clean generation; React 97/97; live HTTP Backend contract 9/9; mock Playwright 1/1; HTTP Playwright 1/1; both builds; HTTP artifact isolation across 10 files and 81 inputs; task-owned dependency cleanup
- `npm --prefix apps/web run check:app-shell`: demo and HTTP artifacts pass across 10 files and 3 assets each
- Root Vanilla JavaScript smoke suite: 103/103
- `git diff --check`
- `npm --prefix apps/web audit --omit=dev --json`: 0 production dependency vulnerabilities

The full development-dependency audit is a verified local finding, not a passing gate: it reports two high-severity transitive development-tool findings in `js-yaml` through `@redocly/openapi-core`. Remediation is `not run` in Task 7 and is tracked for the Task 13 security gate. It is not present in the production dependency audit and does not change this Task 7 feature result.

Browser-process and local-container cleanup was checked after the offline and HTTP profiles. No task-owned Chrome, Playwright, Vite, API/worker process, container, network, or volume remained.

This evidence supports only the local Task 7 candidate. Task 8 manifest-first OPFS Inspection Attachment recovery is subsequently `verified locally`; Task 12 typed network sync is the next binding slice. Production release and cutover remain outside this authorization and `blocked`.
