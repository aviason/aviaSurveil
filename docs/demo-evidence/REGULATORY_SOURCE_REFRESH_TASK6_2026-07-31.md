# Regulatory Source Refresh And Adaptive Checklists — Task 6 Local Evidence

Status: `verified locally`; `candidate-only`; `release pending`;
`production-ready: not established`.

## Scope boundary

This Task 6 evidence uses only the explicit internal `SYNTHETIC-OPS-AOC`
test profile and a synthetic legacy-checklist candidate fixture. It proves
local candidate workflow mechanics, transport parity, and fail-closed
governance; it does not establish legal authority, a real historical
checklist import, source-owner validation, technical approval of real content,
or publication of an official checklist.

The root legacy demo remains unchanged. A real historical checklist must enter
only as `EXISTING_CHECKLIST_CANDIDATE`; its wording, operational intent, and
result history are candidate input, never regulatory authority. The approved
current regulatory/controlled-CAA-procedure chain remains the sole authority.

## Delivered behavior

- Every generated and published question has the two canonical views:
  `scopeRecommendation` and `regulatoryTrace`. The resolved trace contains
  source identity/title, immutable version/SHA-256, locator/page/section/clause,
  source type, applicability, national reference, controlled CAA procedure
  mapping, verification objective, expected Evidence, currentness, and
  technical-review state. The scope view contains classification, signals,
  operational-history basis, rationale, guardrails, and approval/review state.
- A Draft may carry the literal `SOURCE_MAPPING_REQUIRED` state only for an
  `EXISTING_CHECKLIST_CANDIDATE`. It contains no partial citation or partial
  trace and cannot be submitted as validated, technically approved,
  automatically deferred, published, or materialized into an executable Audit
  package.
- Validation rejects missing origin/trace/classification/rationale/applicability/
  currentness/technical review, stale source lineage, unresolved source gaps,
  and automatic deferral for mandatory, safety-critical, changed, overdue, or
  unknown-history controls. Complete `FOCUSED_FULL` and
  `ROTATIONAL_SAMPLE` questions remain valid branches.
- Exact origins are restricted to `REGULATORY_TRACE`,
  `EXISTING_CHECKLIST_CANDIDATE`, and `HYBRID_RECONCILED`.
  `HYBRID_RECONCILED` retains the candidate-only legacy wording, intent,
  result-history, Evidence, applicability, and scope comparison while binding
  the new Draft to the current trace.
- Observed source rows are inert. An explicit, append-only source-currentness
  activation records the exact predecessor/current snapshot and hash, refuses
  historical-source reactivation, and creates an impact-review Draft for a
  source change. Candidate import may bind only to that committed activation;
  it cannot activate a source implicitly.
- A source version/hash change projects the old trace as `STALE`, blocks its
  publication, creates a distinct impact-review Draft, and leaves the old
  candidate snapshot, published version, and in-progress Audit package bytes
  unchanged.
- A rejected raw source-change import is atomic: its savepoint rollback leaves
  no generation-run, scope, source, partition, candidate, binding, or
  impact-Draft lineage behind. One impact-review Draft may link several
  independently immutable candidate roots.
- Department Manager technical approval and publication are separate,
  persisted decisions. A generic Admin and a Department Manager without a
  current assignment are denied. Auditee projections neither expose governed
  source/review/publication facts nor internal deliberation.
- The OpenAPI source, generated Go/TypeScript transports, semantic mock,
  canonical HTTP boundary, PostgreSQL persistence, and React Checklist Builder
  share the same question shape and literal source-gap semantics.
- The mock Regulatory Library projects V1 and V2 source rows from immutable
  candidate/run lineage; it does not attribute a V2 candidate, hash, or run to
  the V1 source row. The pre-seeded V1 currentness activation also replays its
  stored immutable receipt just as the HTTP/Go boundary does.

## Fresh verification

| Command or check | Result |
|---|---|
| `node scripts/regulatory/sync-ncaa-namcats.mjs` | `verified locally`: all 58 bounded local documents were reused; no new byte download was needed (605,250,466 bytes). |
| `node scripts/regulatory/sync-ncaa-namcats.mjs --verify-only` | `verified locally`: 58 bounded local source documents, 605,250,466 bytes. No refresh/download was performed. |
| `./scripts/check-contracts.sh` | `verified locally`: generated-contract checks passed, 16 pass / 0 fail. |
| `npm --prefix apps/web run typecheck` | `verified locally`: passed. |
| `npm --prefix apps/web test -- src/features/admin/admin-secondary-pages.test.tsx src/backend/transport-mappers.test.ts` | `verified locally`: 28 tests passed. |
| `npm --prefix apps/web run build:demo` | `verified locally`: passed. |
| `env GOCACHE=/private/tmp/avia-regulatory-go-cache go test ./internal/configuration ./internal/httpapi` | `verified locally`: passed. |
| `go test ./internal/httpapi ./internal/regulatory ./internal/checklistgovernance` | `verified locally`: passed. |
| `npm --prefix apps/web test -- --run src/backend/governed-task6-parity.test.ts src/backend/governed-checklist-http-parity.test.ts` | `verified locally`: 19 tests passed, including pre-seeded activation replay and exact V1/V2 source-library lineage. |
| `node --test api/openapi/tests/regulatory-question-governance-contract.test.mjs` | `verified locally`: 2 tests passed. |
| `node --test tests/checklist-management-smoke.test.js tests/manager-checklist-management-smoke.test.js tests/demo-boundary-smoke.test.js` | `verified locally`: 3 tests passed; the legacy-demo boundary remains intact. |
| `AVIA_HTTP_PROFILE_FOCUSED_E2E=regulatory-source-refresh ./scripts/test-http-profile.sh` | `verified locally`: 24-artifact inventory, canonical PostgreSQL behavioral tests (11.529s), clean reset, and HTTP Playwright at 1440×900 and 390×844 passed (2/2). The profile drained its outbox and removed its local containers, volumes, network, Vite, and runtime directory. |
| Rendered Checklist Builder review | `verified locally`: source-gap Draft visibly showed `SOURCE_MAPPING_REQUIRED` with disabled submission; the hybrid Draft visibly showed origin, comparison, trace/source version/hash/locator/applicability/currentness/review state, verification objective, and expected Evidence without console errors or horizontal overflow. |
| Independent read-only Task 6 implementation and delta review | Accepted: no Critical or Important finding. The reviewer did not run commands; command results in this table are owner-run local evidence. |
| `node tests/harness-docs-smoke.test.js` | `verified locally`: passed after plan, index, tracker, product-specification, and evidence synchronization. |
| `git diff --check` | `verified locally`: passed after final documentation synchronization. |
| Process and service residue check | `verified locally`: filtered process check found no Task 6 Playwright, Vite, HTTP-profile, source-refresh, or profile-container process; `docker ps` was empty. Existing unrelated user processes and the pre-existing dirty worktree were preserved. |

The canonical PostgreSQL tests inspect persisted candidate/status/decision/
publication/Audit rows, immutable snapshots and package bytes, trace hashes,
question identities, and publication digests. They do not rely only on labels
or UI snapshots. The source-gap, stale-source, hybrid-reconciliation, generic
Admin, current-department assignment, and Auditee denials each assert that
forbidden commands leave the relevant persisted effects absent.

## Remaining external decisions

The responsible NCAA source owner must validate current source authority,
Part 127/Part 140 applicability, controlled procedure mapping, and exact
question/Evidence interpretation. The responsible Department Manager must
then perform real technical approval and a separate real publication decision.
Those are `blocked` external-owner decisions, not local verification gaps.

Independent read-only Task 6 implementation and final-delta review are
accepted with no Critical or Important finding. Final documentation, harness,
diff, Docker, and task-process checks are `verified locally`. The final Git
status remains dirty because it was already dirty at task start; no unrelated
tracked or untracked work was reset, reverted, staged, committed, pushed, or
deleted.
