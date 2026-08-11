# AWS Private-Pilot Task 8 Runtime Simplification Evidence

**Evidence date:** 11 August 2026  
**Plan:** [AWS Single-AZ ARM64 Private-Pilot Production Preparation](../exec-plans/active/2026-08-10-aws-single-az-arm64-private-pilot-production-plan.md)  
**Scope:** Task 8 Layers 8.0–8.5 local implementation and requalification only  
**Result:** the simplified runtime contracts and available focused gates are
`verified locally`; provider-backed, release, external identity/data, remote
smoke/recovery, and native ARM64 capacity gates are `not run` or `blocked`.
The artifact remains `candidate-only`, release is `release pending`, and
`production-ready: not established`.

No AWS, Cloudflare, SMTP, DNS, registry, provider-backed Terraform/Terragrunt,
remote-state/lock, publication, secret, apply, RDS, identity/data, remote
smoke, recovery, release, traffic, rollback, retain/destroy, or residue action
was performed for Task 8. The local `deploy/local` MinIO/ClamAV/Mailpit harness
was preserved.

## Immutable inventories

Layer 8.0 created two separate immutable inventories with identical schema and
named per-layer deltas. The legacy file was not rewritten:

| Inventory | File SHA-256 | canonical aggregate | key cardinality |
|---|---|---|---|
| Legacy predecessor | `ed82bf3f04c787d75300c5d61c64608b4b062f8530fe347b05e7fc08e08756f5` | `sha256:e1e26c6d4050c3eaf57e1b6352bd1a3cbc4462d83c94a2c83d04be66d4d6351e` | 8 long-running roles, 11 release subjects, 7 ECR repositories, 10 log groups |
| Final target | `2f0dd4b2fac09888c8ca216d2c9b5f04f87e27cbf54a9a374a108196ffaeb39d` | `sha256:77ee4da38a9a5ca9260a7d90e554f41f05cb20c67ae54d30e43ca88d430605b9` | 4 long-running roles, 7 release subjects, 5 ECR repositories, 8 runtime secret containers, 7 log groups |

The final target roles are `gateway`, `api`, `worker`, and `keycloak`; bounded
jobs are database bootstrap, migration, and Keycloak bootstrap. The target
release subjects are `cloudflared`, `gateway`, `api`, `worker`, `keycloak`,
`database-bootstrap`, and `migration`. The named 8.1–8.5 deltas are stored in
both inventory files and are checked by the inventory generator.

## Layer results

### Layer 8.1 — data-feed runtime retirement

`verified locally` for API/application lifecycle decoupling, command and
private-pilot Compose/Docker/release/IaC removal, and dormant-source retention.
`MaterializeInspection` and `StartInspection` no longer construct or query
data-feed events, require a writer, environment, table, secret, or egress.
Existing `datafeed_*` migrations, triggers, tables, and historical rows remain
unchanged. Dormant packages/tests and the recovery runbook are explicitly not
reachable from the private-pilot runtime. The focused config/application/API/
worker source checks pass in the disposable Go test copy. IPv6-bound dormant
datafeed tests are `blocked` in this sandbox because their `httptest` listener
cannot bind `[::1]:0`; AviaCore contract tests are `blocked` because the
external `/private/integrations/aviacore` fixture is unavailable. ClamAV
scanner tests requiring `127.0.0.1:0` are likewise `blocked` by the sandbox.

### Layer 8.2 — worker reminder controller

`verified locally`. The standalone scheduler command/runtime surface is gone.
The worker owns a separately supervised controller with one startup cycle,
injected ticks, fixed deadlines, deterministic bounded keyset batches, no
overlap, per-candidate failure isolation, later-tick retry, redacted error
telemetry, and coordinated WaitGroup cancellation. The advisory lock,
transaction, uniqueness, notification lease, and retry semantics remain
unchanged. `go test -race ./cmd/worker ./internal/application/... ./internal/worker/...`
passed in the disposable source copy, including the controller deadline,
single-flight, and shutdown tests. Database-backed reminder integration is
`not run` because no task-owned database was started for this evidence pass.

### Layer 8.3 — gateway-embedded React artifact

`verified locally` for the HTTP artifact and contract sources. In a disposable
workspace copy, `npm run typecheck` passed; the focused service-worker/app-shell
suite passed 26/26; `npm run build:http` passed; the HTTP artifact scan passed
79 files/181 inputs; and the app-shell scan passed 79 files/73 assets. The
artifact has network-only, no-store runtime configuration and shell manifest
fetches, obsolete-cache deletion, direct `/index.html` support, immutable
fingerprinted assets, and no SPA fallback for missing assets. The private-pilot
Compose config passed. A task-owned gateway container matrix and ARM64 Docker
build are `not run` because they would require an unavailable release image
and no external registry/network action is authorized.

### Layer 8.4 — native Go PDF

`verified locally`. `github.com/signintech/gopdf v0.38.0` is pinned and regular
and bold Noto Sans are embedded with checksum/source metadata. The native
renderer emits canonical, privacy-safe report narrative rather than an
identifier-only fixture, and enqueue snapshots bind renderer/layout/module/
font/source provenance. Lease renewal/finalization is generation-fenced;
unfinished legacy jobs receive append-only disposition and native jobs have
five-attempt dead-letter/manual-retry rules.

Task-owned output: `/private/tmp/avia-task8-report-final.pdf` (SHA-256
`5ebcd8f42edd4916208103fe4f0eacb8adc305c2ad968b34bc8842ce31be2402`). The
bundled `pdfinfo` reports one A4 page (595x842), no encryption or JavaScript,
fixed epoch metadata, and producer `avia-native-gopdf gopdf@v0.38.0`.
Bundled `pdftotext` extracted actual Turkish, English, French (`résultats
vérifiables`, `la clôture`) and Portuguese (`ação`) narrative, including the
finding and recommendation text. `pdftoppm` rendered page 1 to
`/private/tmp/avia-task8-report-final-pages/page-1.png` (SHA-256
`e0eebc568fd790028554306cd3dc78edf008f049d61331a89331c27154965dbc`); visual
inspection found no clipping, overlap, glyph substitution, or page-break defect.
The focused document,
application, and worker Go tests passed in the disposable source copy.

### Layer 8.5 — reconciliation

`verified locally` for the desired-state reconciliation: four Compose roles,
seven release subjects, five ECR repositories, eight runtime Secrets Manager
containers plus separate RDS bootstrap and SSM tunnel identities, and seven
log groups. Private-pilot decision, Compose, infrastructure, and release
contract tests pass after fixture isolation. Fixture-backed `docker compose
... config --format json` passed; the bare command without the required fixture
environment is `not run`.
Provider initialization/planning, image publication/signing, vulnerability
attestation, external SMTP, deployment, rollback, and residue gates are
`not run`; native ARM64 capacity is `blocked` because no resolved immutable
release image set and no owner overlay are available. No capacity GO or
`NO-GO for t4g.small` is recorded.

## Complete local gate attempts

- `node tests/harness-docs-smoke.test.js` and `git diff --check` — `verified
  locally`.
- `./scripts/test-aws-private-pilot-compose.sh static` — 6/6 passed;
  `verified locally`.
- `./scripts/check-aws-private-pilot-infrastructure.sh` — 23/23 passed;
  provider-backed plan `not run`; `verified locally` for source contracts.
- `./scripts/test-aws-private-pilot-release.sh` — 28/28 passed; every external
  action remains `not run`.
- The complete Go `./...` attempt in a disposable copy reached the Task 8
  packages, but is `blocked` overall by historical missing deliverables,
  unavailable `/private/integrations/aviacore` fixtures, dormant data-feed
  IPv6 `httptest` bind restrictions, and local SMTP/ClamAV/identity listener
  restrictions. The Task 8 focused Go race packages remain green above.
- The complete React Vitest suite was attempted in a disposable copy and is
  `blocked`/not complete because existing navigation tests repeatedly emit
  `Not implemented: navigation to another Document`; the process was stopped
  after the bounded attempt. The Task 8 focused 26/26 suite and both builds
  remain green above.
- The local PostgreSQL/MinIO/ClamAV/Mailpit harness was attempted and is
  `blocked` by sandbox permission to the local Docker socket. No containers,
  volumes, or networks were created by that attempt.
- A bare production Compose config invocation is `not run` without its
  required private fixture environment; the contract's fixture-backed Docker
  config path passed.

## Historical supersession records

The new record supersedes runtime claims only; it does not alter historical
files. Their recorded hashes are:

- `AWS_PRIVATE_PILOT_LOCAL_PREPARATION_2026-08-10.md` —
  `08a6b662ae9255f5c68d9dd081e1bb27f6ee136d117f82ee0241c6c9d096b78b`;
- `AWS_PRIVATE_PILOT_TASK7_DISCOVERY_2026-08-11.md` —
  `b9fc0c27c41b4a4adaf962aabf66310af38a29a432fbf1d830cf57e1df8fa6b2`.

Those files remain byte-preserved predecessor evidence. This record and the
living plan/index/tracker are the only places that describe their Task 8
runtime supersession.

## Final status vocabulary

Local code and contract evidence above is `verified locally`. Unavailable
prerequisites are labeled `blocked` or `not run`; no old pre-amendment result
is inherited. The implementation is `candidate-only`, release is `release
pending`, and `production-ready: not established`.
