# Canonical AGA Implementation Checkpoint — 2026-08-08

This is a local implementation checkpoint for the canonical AGA successor
plan. It is `candidate-only`; it is not stakeholder acceptance, release, or
production evidence. Task 10 external preprod deployment remains explicitly
`not run`.

## Verified locally

| Check | Result |
|---|---|
| OpenAPI bundle, generated Go/TypeScript contracts, and contract suite | `verified locally` — 16/16 checks passed |
| Focused Go packages (`internal/httpapi`, `internal/assignments`, `internal/application`, `migrations`) | `verified locally` |
| Empty-database and retained N-1 migration upgrade, including populated 000029 fixture and migration 38 | `verified locally` |
| React typecheck | `verified locally` |
| `git diff --check` | `verified locally` |

The checkpoint includes the server-owned preparation confirmation revision pin,
restart-safe Department Manager projection, multi-Inspector immutable coverage
key, cumulative governed successor review facts, controlled governed reasons,
disposable-scope authorization for exercise review commands, and the governed
virtual-candidate queue dispatch required for technical approval → publish.
Migration 38 reconciles pre-38 confirmed receipts with immutable assignment
revision successors; hydrated confirmations disable repeat submission.

## Not run or blocked

- Full `go -C apps/api test -count=1 ./...`: `not run` after a long-running
  local attempt.
- Full React test suite: `blocked` by existing fixture-contract failures in
  unrelated admin/executive workspace families.
- Recursive root JS/MJS discovery: `blocked` by existing AGA/AviaCore/AWS
  fixture-contract families.
- Donor-disabled connected OIDC multi-role qualification, real local
  MinIO/ClamAV/Gotenberg/Mailpit matrix, backup/restore, visual/browser
  viewport evidence, donor deletion/requalification, and stakeholder review:
  `not run`.
- External preprod deployment and all remote infrastructure actions: `not run`
  and unauthorized by this plan execution request.

Sol Ultra's independent read-only implementation review remains `NOT ACCEPTED`
until the remaining Important findings and the affected connected/evidence
gates are resolved and rerun.
