# AviaSurveil360 Build Summary

Last updated: 2026-08-13

AviaSurveil360 contains the intact root static demo and a separate React/Go
candidate. The maintained runtime is local and disposable.

Status: `candidate-only`. Release: `release pending`.

## Maintained runtime

- `apps/web`: build-separated React/Vite demo and HTTP artifacts.
- `apps/api`: Go API, consolidated worker/reminder controller, migrations,
  application PostgreSQL authority, BFF sessions, object storage, scanning,
  email, and document workflows.
- `apps/auth`: first-party Go OIDC, password/MFA/recovery, provider sessions,
  signing keys, authority mirrors, private administration, and dedicated auth
  PostgreSQL.
- `deploy/local/compose.yaml`: maintained demo/full/local-preprod services,
  including separate application and privileged auth Mailpit services.
- `apps/api/cmd/preprod-canonical-demo-identity-loader`: resumable nine-user
  synthetic identity bootstrap.
- `apps/api/cmd/preprod-canonical-aga-loader`: canonical 1,310-question
  exercise catalog bootstrap.

The public gateway strips `/identity/*` before sending requests to auth port
8080. Private administration on port 8081 is internal-only and has no gateway
or host publication.

## Identity cutover

The canonical disposable first-party cutover is `verified locally` for nine
fresh synthetic subjects, exact role/organization routes, browser token-storage
absence, logout/account switching, stale revision rejection, lifecycle
changes, recovery, MFA reset, public admin denial, dependency loss/restart,
fault recovery, and HTTPS/local-HTTP browser matrices.

The owner subsequently authorized repository-local retirement of the previous
identity implementation. Its remaining adapter, profiles, runtime assets, AWS
deployment package, fixtures, rollback assumptions, and historical comparison
evidence were removed. Post-retirement auth/API/web and canonical HTTPS/HTTP
requalification are `verified locally`.

See:

- [active first-party OIDC plan](../exec-plans/active/2026-08-11-first-party-go-oidc-auth-replacement-plan.md)
- [Task 11 evidence](../security/auth-replacement/evidence/2026-08-11-oidc-library-spike/TASK11_LOCAL_PREPROD_CUTOVER.md)

## Current local verification

The following are `verified locally` after topology retirement:

- auth focused/PostgreSQL/full/race/vet/module verification;
- API focused suites, disposable integration, DB-free full graph, vet/module,
  and bounded applicability race shards;
- web typecheck, full tests, demo/HTTP builds, and HTTP artifact scan;
- maintained Compose policy/config, generated contracts, SQLC, static profile
  contracts, and stale-reference/path scans;
- canonical HTTPS lifecycle, role panels/logout, dependency fault/restart, and
  cleanup;
- canonical local HTTP lifecycle, role panels/logout, and cleanup.

The HTTP backend contract transcript is `verified locally`: its coordination
case now uses a separate coherent pre-execution fixture, while the canonical
execution seed remains `IN_PROGRESS` for offline-grant qualification. The API
transition guard remains fail-closed. The AviaCore data-feed coverage register
is `blocked` by unrelated dirty-tree migration, relation/column, and
mutation-inventory drift.
Fresh SBOM/license/vulnerability/provenance evidence and remote/deployment/
traffic/release activity are `not run`.

Harness documentation smoke and the final diff check are `verified locally`.

## Boundary

No remote system, DNS, external SMTP, deployment, traffic, production secret,
or real user is touched or authorized. No commit, push, or branch operation is
part of this work. The result remains `candidate-only` and `release pending`.
