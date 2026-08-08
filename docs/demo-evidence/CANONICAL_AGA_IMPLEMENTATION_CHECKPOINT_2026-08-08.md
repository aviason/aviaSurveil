# Canonical AGA Implementation Checkpoint — 2026-08-08

This is a local implementation checkpoint for the canonical AGA successor
plan. It is `candidate-only`; it is not stakeholder acceptance, release, or
production evidence. Task 10 external preprod deployment remains explicitly
`not run`.

## Verified locally

| Check | Result |
|---|---|
| OpenAPI bundle, generated Go/TypeScript contracts, and contract suite | `verified locally` — 16/16 checks passed |
| Canonical/OpenAPI/donor-boundary JS/MJS subset | `verified locally` — 70/70 tests passed |
| Focused Go packages (`internal/httpapi`, `internal/assignments`, `internal/application`, `migrations`) | `verified locally` |
| Empty-database and retained N-1 migration upgrade, including populated 000029 fixture and migration 38 | `verified locally` |
| React typecheck | `verified locally` |
| Full React test suite | `verified locally` — 91 files / 767 tests passed |
| Donor-free HTTP/Go artifact boundary and local-preprod runtime-role boundary | `verified locally` — normal API/worker/scheduler/migrate dependency and binary-marker scans, focused artifact/compose tests, and disposable PostgreSQL privilege probe |
| `git diff --check` | `verified locally` |

The checkpoint includes the server-owned preparation confirmation revision pin,
restart-safe Department Manager projection, multi-Inspector immutable coverage
key, cumulative governed successor review facts, controlled governed reasons,
disposable-scope authorization for exercise review commands, and the governed
virtual-candidate queue dispatch required for technical approval → publish.
Migration 38 reconciles pre-38 confirmed receipts with immutable assignment
revision successors; hydrated confirmations disable repeat submission. Question
Review now returns bounded append-only Draft history, New Audit summary values
are recomputed from immutable catalog memberships, the Lead handoff uses a
canonical pre-materialization route, donor workspace operations are absent from
the normal HTTP contract/client/artifact, and long-lived local-preprod services
use a provisioned non-owner database role.

## Not run or blocked

- Full `go -C apps/api test -count=1 ./...`: `blocked` — all non-integration
  packages passed, but integration packages could not connect to the disposable
  PostgreSQL endpoint at `127.0.0.1:55432` after the sandbox-authorized retry.
- Recursive root JS/MJS discovery: `blocked` by existing AGA/AviaCore/AWS and
  paused-donor fixture-contract families; canonical contract and harness smoke
  subsets pass.
- Connected HTTP/OIDC visible-action qualification: `blocked` by the existing
  fixture path requiring a canonical execution package and a missing visible
  action evidence row; its task-owned PostgreSQL/Keycloak/MinIO/Mailpit stack
  was cleaned up.
- Donor-disabled full OIDC multi-role qualification, real local
  MinIO/ClamAV/Gotenberg/Mailpit matrix, backup/restore, visual/browser
  viewport evidence, donor deletion/requalification, and stakeholder review:
  `not run`.
- External preprod deployment and all remote infrastructure actions: `not run`
  and unauthorized by this plan execution request.

Sol Ultra's latest independent read-only implementation review remains
`NOT ACCEPTED` pending a fresh post-remediation reread. The final Sol XHigh
code-boundary reread is `ACCEPTED` with 0 Critical / 0 Important findings; it
does not replace connected, visual, or stakeholder gates.
The local implementation remains `candidate-only` and `release pending`.
