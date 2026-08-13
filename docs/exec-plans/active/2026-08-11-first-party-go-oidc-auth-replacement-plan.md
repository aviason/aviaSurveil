# First-Party Go OIDC Authentication And Repository-Local Provider Retirement

Date: 2026-08-11
Last updated: 2026-08-13
Status: active
Release state: `candidate-only`; `release pending`

## Objective and user-visible outcome

Make the first-party Go OIDC service the only maintained identity provider in
this repository. The disposable local-preprod/demo topology must provision nine
fresh synthetic identities, authenticate them through the API BFF, enforce
exact organization/role/revision authority, and reject stale sessions after
lifecycle changes.

The repository must contain no active adapter, runtime profile, image,
configuration, test fixture, retired auxiliary topology, rollback baseline, or
historical evidence for the retired third-party provider. This expanded
repository-local retirement was explicitly authorized by the owner on
2026-08-12 after the canonical first-party cutover.

The result remains `candidate-only` and `release pending`.

## Scope

In scope:

- `../../shared/auth` public OIDC/UI on port 8080 and private provider
  administration on port 8081;
- provider persistence, authority, credentials, MFA, recovery, sessions,
  signing keys, idempotency receipts, audit, and cleanup;
- the API/worker first-party provider-admin adapter and BFF authority checks;
- the resumable nine-user canonical synthetic identity loader;
- all maintained local/demo/test Compose profiles and recovery support;
- repository-local retirement of the previous provider adapter, runtime,
  profiles, retired auxiliary packages, fixtures, tests, and evidence;
- documentation, policy, and harness synchronization;
- local verification and task-owned cleanup.

Out of scope:

- remote systems, DNS, external SMTP, deployment, traffic, production secrets,
  real users, and identity migration;
- commits, pushes, and branch operations;
- release approval and any production claim;
- rerunning or citing the retired security scan.

## Repository orientation

- `../../shared/auth/` owns first-party OIDC, credentials, MFA, recovery,
  signing keys, opaque claim projections, private administration, and auth
  PostgreSQL migrations.
- `apps/api/internal/identity/` owns neutral provider authority types and the
  first-party provider-admin client.
- `apps/api/internal/platform/session/` owns BFF sessions and exact revision
  validation.
- `apps/api/cmd/preprod-canonical-demo-identity-loader/` owns the disposable
  nine-user cross-database bootstrap.
- `deploy/local/` owns maintained local topology, gateway routing, generated
  disposable secrets, SMTP-free local-preprod activation, and policy.
- `deploy/recovery/` and the recovery scripts own local backup/fingerprint
  support for application and first-party auth PostgreSQL.
- `docs/security/auth-replacement/` contains only the current plan link and
  the Task 11 cutover/retirement evidence.

## Fixed contracts

### Provider authority

Each provider profile and application-authority mirror persists:

- opaque `usr_...` subject;
- membership ID;
- organization ID;
- exactly one role;
- state;
- membership revision;
- separate `auth_revision`.

Private ID-token claims are emitted only for an active, verified account with
active authority. The claims are `organization_id`, a one-element `roles`,
`membership_id`, `membership_revision`, and `auth_revision`.

### OIDC and credential behavior

Authorization requests and codes expire and are cleaned in bounded batches.
`auth_time` is stable for one authentication event. AMR is durable and starts
with `pwd`, followed by `otp` or `mfa` when applicable.
`offline_access` and refresh-token grants are disabled; the BFF never stores
or uses provider refresh tokens.

Password changes, MFA resets, authority changes, disable/suspend, and
activation revoke provider credentials and sessions. Reset and activation are
transactional consume-and-mutate operations with replay-safe terminal results.

### Private administration

Port 8081 is a separate listener and is never gateway-routed. It requires a
separate generated high-entropy bearer secret from a mounted file and verifies
it in constant time. Mutations require `Idempotency-Key`, bounded JSON,
canonical request hashing, expected/resulting revisions, typed neutral errors,
redacted audit records, and deterministic replay/conflict behavior.

### Application authority

There is one provider selection: first-party. There is no dual issuer or
fallback. Login compares token claims, fresh provider observation, and
application membership. The disposable local-preprod profile refreshes
provider authority on every protected request so lifecycle changes invalidate
stale BFF sessions immediately.

### Canonical bootstrap

The loader provisions nine new synthetic identities through the private admin
API and activates them through the private local-preprod admin endpoint. It never imports
subjects, passwords, or MFA state from the retired provider. It persists
returned subjects into application identity references, memberships, lifecycle
receipts, and required assignments. Replay is resumable and idempotent; partial
or drifted cross-database state fails closed.

## Ordered implementation

1. Documentation and authority boundary — completed.
   - Recorded the initial canonical-only authorization.
   - Recorded the later explicit repository-local retirement authorization.
   - Removed obsolete Task 0–10 comparison/export evidence.

2. Provider P0 persistence and OIDC behavior — completed.
   - Added forward-only auth migration
     `000008_local_preprod_authority_admin.up.sql`.
   - Added authority mirrors, operation receipts, expiring authorization state,
     durable AMR/auth-time, revocation, and transactional recovery/activation.

3. Private provider administration — completed.
   - Added the split 8081 server and neutral operation contracts.
   - Kept every admin route private from the gateway.

4. API, worker, BFF, and lifecycle integration — completed.
   - Added the first-party adapter and exact authority comparison.
   - Removed provider selection fallbacks and provider-specific lifecycle
     errors.
   - Removed the retired adapter and integration scenario.

5. Canonical bootstrap and topology — completed.
   - Added first-party auth PostgreSQL, SMTP-free local-preprod auth runtime, generated
     secret/TLS material, gateway prefix stripping, and the nine-user loader.
   - Updated HTTPS, local HTTP, status, fault/restart, and cleanup scripts.

6. Repository-local retirement — completed; requalification completed locally.
   - Removed remaining provider services, images,
     configs, adapter tests, runtime code, old loader, and rollback artifacts.
   - Removed stale standalone scheduler topology; reminder scheduling remains
     owned by the worker.
   - Rewrote recovery and image-policy surfaces for the maintained first-party
     topology.

7. Final verification and documentation synchronization — completed locally;
   release remains pending.
   - Re-ran the affected focused and full gates after the expanded retirement.
   - Recorded only observed results and cleaned task-owned processes/resources.

## Current verification record

| Gate | Literal result |
|---|---|
| Auth focused suites and PostgreSQL migration/identity/session/recovery evidence | `verified locally` |
| Auth full `go test`, race, vet, and module verification | `verified locally` |
| API focused identity/session/admin/loader/worker suites | `verified locally` |
| API disposable session-authority and integration harness | `verified locally` |
| API DB-free full graph, vet, and module verification | `verified locally` |
| API `agaapplicability` bounded race shards | `verified locally` |
| Web typecheck, full Vitest, demo/HTTP builds, and HTTP artifact scan | `verified locally` |
| Maintained Compose policy/config, contracts, SQLC, static profiles, and stale-reference scan | `verified locally` |
| Canonical HTTPS lifecycle, role panels/logout, dependency fault/restart, and cleanup | `verified locally` |
| Canonical local HTTP lifecycle, role panels/logout, and cleanup | `verified locally` |
| HTTP backend contract transcript | `verified locally` |
| AviaCore data-feed coverage register | `blocked` |
| Harness documentation smoke and final `git diff --check` | `verified locally` |
| Fresh SBOM/license/vulnerability/provenance tooling | `not run` |
| Remote/deployment/traffic/release activity | `not run` |

The HTTP backend contract initially exposed two different lifecycle phases in
one fixture: an awaiting Auditee assignment and an `IN_PROGRESS` inspection
required by offline-grant qualification. The API correctly rejected the
incompatible confirm command. The harness now resets the coordination test to
a separate pre-execution fixture (`AWAITING_AUDITEE_CONFIRMATION` for the
inspection and assignment, `NOT_STARTED` for the checklist) while retaining
the ordinary execution fixture for offline tests. The API state-machine guard
was preserved, the duplicate coordination projection caused by multiple
matching planning drafts was removed with a distinct projection query, and
the HTTP contract plus outbox drain are `verified locally` (24/24 contract
tests).

The AviaCore coverage test has four failures caused by pre-existing
dirty-tree registry drift: migration 000028 and 000030 hashes differ, later
migrations are absent from the closed register, persisted relation/column
dispositions are incomplete, and the authoritative mutation/event inventory is
stale. This is outside the auth retirement scope and remains `blocked` pending
an explicit coverage-register disposition update.

## Acceptance criteria

The plan can leave active status only when:

- active source, tests, maintained configuration, and living documentation have
  no reference to the retired provider or deleted auxiliary package;
- auth migration/admin/authority/OIDC focused tests and auth
  full/race/vet/module verification are `verified locally`;
- API adapter/session/lifecycle/loader focused tests and API
  full/race/vet/module verification are `verified locally`, or a literal
  environment blocker is recorded;
- web typecheck and full tests are `verified locally`;
- Compose policy/config checks are `verified locally`;
- fresh canonical HTTPS and local HTTP E2E, lifecycle, fault/restart, public
  admin-route denial, and cleanup are `verified locally`;
- harness documentation smoke and `git diff --check` are `verified locally`;
- plan, index, tracker, and Task 11 evidence agree;
- the result is still `candidate-only` and `release pending`.

## Risks and recovery

The identity namespace is disposable and contains no real users. The loader and
provider-admin receipts make interrupted bootstrap resumable. Namespace
initialization is create-only unless explicit rotation is requested.

Repository deletions remain recoverable from version control because no commit
or push is authorized. No remote rollback path is retained. Recovery testing is
limited to the maintained first-party PostgreSQL topology.

## Decisions and discoveries

- The owner explicitly superseded the former requirement to retain noncanonical
  provider profiles, adapter code, local provider assets, retired auxiliary
  material, and historical comparison evidence.
- The old standalone scheduler Compose role referenced a nonexistent command
  and Docker target. It was removed; reminder scheduling already runs in the
  worker.
- The old connected preprod data loader depended on the retired provider and
  duplicated the canonical loaders. It was removed; canonical AGA and identity
  loaders remain isolated.
- The AviaCore data-feed coverage register already contains broader dirty-tree
  drift unrelated to this retirement. Its closed-world test is currently
  `blocked` pending an explicit product disposition update; this plan does
  not invent those decisions.

## Execution Prompt

Continue in
`/Users/marlonjd/Developer/monorepos/avia/apps/surveil` on the current branch.
Preserve unrelated dirty-tree changes. Do not commit, push, deploy, touch remote
systems, rerun/cite the retired security scan, or claim anything beyond
`candidate-only` and `release pending`.

Finish living documentation and zero-reference cleanup, then run the smallest
focused gates followed by auth full/race/vet/module checks, API focused/full/
race/vet/module checks, web typecheck/full tests, Compose policy/config,
canonical HTTPS and local HTTP E2E, fault/restart/cleanup, harness docs smoke,
and `git diff --check`. Record unexecuted gates as `not run` and genuine
environment constraints as `blocked`.
