# Task 11 — Canonical First-Party OIDC Cutover And Local Retirement

Date: 2026-08-12

## Authorization

The initial Task 11 authorization covered only the canonical disposable
local-preprod/demo topology. After that cutover was demonstrated, the owner
explicitly expanded repository-local scope and directed removal of every
remaining profile, adapter, local runtime, retired auxiliary package, rollback
baseline, and historical comparison artifact for the retired provider.

There is no serving production environment and there are no real users. No
remote system, DNS, external SMTP, deployment, traffic, or production secret is
authorized or touched.

Status: `candidate-only`. Release: `release pending`.

## Implemented first-party boundary

- Auth migration `000008_local_preprod_authority_admin.up.sql` persists
  provider profiles, application-authority mirrors, separate membership and
  auth revisions, admin receipts, authorization expiry, stable auth time,
  durable AMR, and cleanup state.
- Password, MFA, authority, disable/suspend, and activation changes revoke
  credentials and sessions. Recovery and activation mutations are
  transactional and replay-safe.
- Public OIDC/UI remains on 8080. Private provider administration remains on
  8081 behind a separate mounted bearer secret with constant-time comparison;
  the gateway never routes it.
- The API and worker use only the first-party provider-admin adapter. Login and
  protected requests compare token, provider, and application authority with
  exact organization, role, membership, membership revision, and auth revision.
- The canonical loader creates nine fresh synthetic `usr_...` subjects,
  activates them through the dedicated auth Mailpit, and persists exact
  application references and assignments. It is resumable and fails closed on
  drift.
- Maintained local Compose profiles use first-party auth PostgreSQL,
  authenticated mandatory-STARTTLS auth Mailpit, and the Go auth service.
  Public `/identity/*` routing strips the prefix and never exposes port 8081.
- Repository-local adapter code, profiles, test fixtures, image/config assets,
  retired auxiliary topology, rollback assumptions, and historical comparison
  evidence for the retired provider were removed under the expanded
  authorization.

## Verification record

The original canonical cutover evidence is `verified locally` for:

- provider-owned password login, TOTP MFA, recovery, password reset, MFA reset,
  logout, signing-key rotation, dependency loss/restart, and backup/restore;
- nine synthetic users reaching exact role/organization routes through the API
  BFF;
- no provider credentials in browser storage;
- no-membership and organization/role/revision mismatch denial;
- stale auth-revision and lifecycle rejection;
- public provider-admin route denial;
- canonical HTTPS and local HTTP browser matrices;
- task-owned cleanup.

After the expanded repository-local retirement:

- Auth focused, PostgreSQL, full, race, vet, and module gates: `verified locally`.
- API focused suites, disposable integration harness, DB-free full graph, vet,
  module verification, and bounded applicability race shards: `verified locally`.
- Web typecheck, full Vitest, demo/HTTP builds, and HTTP artifact scan:
  `verified locally`.
- Compose policy/config, generated contracts, SQLC, static profile contracts,
  and stale-reference/path scans: `verified locally`.
- Canonical HTTPS lifecycle, nine-user role panels/logout, dependency
  fault/restart, public-admin denial, and cleanup: `verified locally`.
- Canonical local HTTP lifecycle, nine-user role panels/logout, and cleanup:
  `verified locally`.
- HTTP backend contract transcript: `verified locally`. The coordination
  contract now resets a separate coherent pre-execution fixture, while the
  ordinary canonical execution fixture remains `IN_PROGRESS` for offline-grant
  qualification. The API transition guard remains fail-closed, and the
  disposable HTTP contract plus outbox drain completed 24/24 tests.
- AviaCore data-feed coverage register: `blocked` by unrelated dirty-tree
  migration, relation/column, and mutation-inventory drift; no coverage
  records were fabricated.
- Harness documentation smoke and final diff check: `verified locally`.
- Fresh SBOM/license/vulnerability/provenance tooling: `not run`.
- Remote/deployment/traffic/release activity: `not run`.

The canonical fault/restart gate initially exposed an identity readiness gap:
the API was probing OIDC discovery rather than the auth service's dynamic
`/health/ready` endpoint. The maintained local Compose health URL now probes
that readiness endpoint, and the complete fault matrix passed afterward.

No release or production claim follows from this evidence. The result remains
`candidate-only` and `release pending`.
