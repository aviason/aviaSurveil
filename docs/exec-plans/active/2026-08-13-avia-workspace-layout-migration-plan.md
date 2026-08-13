# AviaWorkspace Layout Migration

Date: 2026-08-13
Last updated: 2026-08-13
Status: active
Release state: `candidate-only`; `release pending`

## Planning input

Two independent read-only planning passes were requested with
`gpt-5.6-sol` at `xhigh` reasoning: one for AviaSurveil and one for
AviaWorkspace. Their common recommendation was to make repository ownership,
durable shared boundaries, environment selection, connected Data, and release
selection explicit, while keeping live external qualification and deployment
behind their actual gates. This continuation implements the repository-local
parts of those decisions.

## Objective and user-visible outcome

Make the new AviaWorkspace layout authoritative for application ownership and
local composition while keeping the product honest about what is runnable.
AviaSurveil identifies itself as `aviason/aviaSurveil` and owns application
behavior and candidate-local validation. AviaWorkspace owns topology,
environment selection, release locks, and platform infrastructure. Shared
AviaAuth owns durable identity, and shared AviaData owns the local admission
boundary and retained source contracts. Local candidate implementation is
complete; live cross-repository qualification, released images, and cloud
deployment remain separate evidence states.

The immediate user-visible result is a safer local contract: disabled Data
configuration fails closed, Surveil's Workspace composition uses a private
file-backed Auth-admin target, and active execution plans point at the current
checkout. The result is not a deployment or production-readiness claim.

## Scope

In scope:

- AviaSurveil API configuration and focused contract tests for `AVIA_DATA`.
- Surveil local Compose declarations and preprod namespace initialization.
- Surveil component ownership metadata and architecture/package documentation.
- AviaWorkspace dev secret preparation and Auth/Surveil Compose wiring.
- Durable shared/auth source migration and private admin listener contract.
- Local AviaData PostgreSQL/SeaweedFS admission candidate and Surveil worker
  cutover.
- Coordinated AviaData source/fixture identity migration metadata and ledger.
- Immutable release-lock validation and exact environment target cutover.
- Workspace integration-gap documentation.
- Active AviaSurveil plan paths that still point at retired checkout locations.
- Local verification and literal blocker recording.

Out of scope:

- Branch operations, commits, pushes, deployment, traffic, or external writes.
- Cloud deployment, traffic changes, commits, pushes, or other external writes.
- Rewriting retained AviaData/AviaCore historical technical identifiers,
  source URLs, pinned revisions, fixtures, or evidence bytes. The coordinated
  migration is registration-only and does not mutate history.
- Renaming all environment values or deleting the `local-preprod` topology
  before an equivalent qualified disposable lane exists.
- Inventing released image digests, prepared demo data, or production evidence.

## Ownership and repository orientation

- `apps/surveil/` owns Surveil application behavior, application migrations,
  and candidate-local validation fixtures.
- `shared/auth/` owns the durable AviaAuth identity service and its private
  admin boundary. Its local implementation is present and unit-tested; live
  database/mail/browser qualification and release remain pending.
- `shared/data/` owns AviaData source and technical contracts. Retained
  `aviacore` identifiers and source-bound fixtures are versioned contracts,
  not branding-only strings.
- `workspace/` owns platform topology, environment selection, deployment
  targets, release locks, and customer infrastructure.

## Fixed decisions

### Application contract

`apps/surveil/component.json` is Workspace contract v1 metadata for the
`surveil` application. Its local Compose file is explicitly
`candidate-validation-only`, and `aviaWorkspace` is the deployment authority.
Runtime entries remain `candidate-only` or `blocked` until their dependencies
are independently qualified.

### AviaData boundary

`AVIA_DATA` accepts only `0` or `1`. Namibia dev now requires
`AVIA_DATA_MODE=local-candidate` and composes the internal PostgreSQL/SeaweedFS
admission service plus the Surveil worker loop. Other environments remain
release-gated; no external source or fallback to embedded data behavior is
permitted.

### Auth administration boundary

The target Workspace dev topology is:

- public identity/OIDC listener: `auth:8080`;
- private Auth administration listener: `auth:8081`;
- API/worker admin URL: `http://auth:8081/private/admin`;
- admin credential: create-only `auth_admin_secret`, mounted from a file.

The listener and authenticated admin API are implemented in shared/auth and
the Workspace wiring uses a file-backed secret. The local unit suite passes;
database/mail/browser E2E and released-image evidence remain pending.

### Historical identity references

Historical test output, source-bound AviaData fixtures, pinned revisions, and
evidence are preserved. New Workspace fixture registrations use the canonical
IDs from `metadata/source-identity-migration.v1.json`; migration `0009` is
forward-only and immutable.

## Ordered implementation

1. **Baseline and authority check — completed.**
   - Read the repository-local instructions for Surveil, AviaData, and
     Workspace.
   - Preserved the existing Surveil changes and unrelated Workspace changes.
   - Confirmed that the two `gpt-5.6-sol xhigh` planning passes were read-only.

2. **Explicit Data contract and local-candidate mode — implemented and verified locally.**
   - Added `Settings.DataEnabled` and strict `AVIA_DATA` parsing.
   - Rejected known feed configuration while `AVIA_DATA=0`.
   - Added `local-candidate` mode validation, internal endpoint validation,
     and focused configuration coverage.
   - Moved Namibia dev to the local candidate Data path with a dedicated
     payload-key secret; production-like environments remain release-gated.

3. **Application ownership metadata — implemented and verified locally.**
   - Declared candidate-local Compose scope and Workspace deployment authority
     in `component.json`.
   - Synchronized README, architecture, and manifest ownership statements.

4. **Durable shared Auth migration — implemented locally; live qualification pending.**
   - Added create-only `auth_admin_secret` generation to `prepare_dev`.
   - Added the secret to the Workspace base Compose contract and Auth/Surveil
     service mounts.
   - Moved the durable auth source and migrations from embedded Surveil into
     `shared/auth`; removed the embedded source and compatibility imports.
   - Implemented the private 8081 listener and file-backed admin secret
     contract; shared/auth `go test ./...` passes locally.

5. **Active plan path cleanup — implemented.**
   - Replaced stale local checkout paths in active plan execution prompts with
     `/Users/marlonjd/Developer/monorepos/avia/apps/surveil`.
   - Kept historical GitHub command output and source-bound evidence unchanged.

6. **Focused verification and handoff — completed locally; broad migration remains active.**
   - Ran the tests and static checks in the verification matrix below.
   - Recorded exact `verified locally`, `blocked`, `not run`, and
     `candidate-only` states.
   - Leave this plan active until shared runtime and release dependencies are
     separately qualified.

7. **Cross-repository component metadata enforcement — implemented and verified locally.**
   - Workspace now validates canonical repository and image identity, the
     exact four-environment runtime matrix, literal runtime statuses, and
     non-empty blocker reasons for every sibling component.
   - Application contracts must publish a real local Compose path,
     `candidate-validation-only` scope, and `aviaWorkspace` deployment
     authority.
   - The new gate found and repaired the missing canonical image identity in
     AviaData's component metadata without changing retained `aviacore`
     technical contracts.
   - Workspace README and the Surveil integration-gap record now distinguish
     completed static contracts from blocked durable runtime evidence.

8. **Local AviaData admission and worker cutover — implemented locally; E2E pending.**
   - Added a PostgreSQL-backed candidate admission service with SeaweedFS
     write-once receipts, durable state transitions, readiness, and the
     existing V3 contract boundary.
   - Added the Surveil local-candidate worker client, lease/outbox processing,
     exact internal endpoint checks, and Workspace dev composition.
   - Added migration `0009` plus canonical source/fixture registration
     metadata. Historical source IDs, fixture IDs, and evidence bytes were
     not rewritten.

9. **Environment and release-lock cutover — implemented locally; release pending.**
   - Namibia dev now requires the local Data admission candidate and exact
     `dev` target selection.
   - Demo, preprod, and prod lock files are immutable and tamper-evident via a
     canonical `lockDigest`; release checks fail closed for missing image
     digests.
   - Existing demo image values were preserved. Preprod/prod remain empty until
     independently published immutable images are supplied.

10. **Deployment boundary — intentionally not run.**
    - Static Compose and target validation are in scope.
    - Cloud deployment, traffic, image publication, and external release
      writes remain `not run` because the repository instructions require
      separate owner authorization and external credentials.

## Verification matrix

Run from the owning repository unless noted otherwise:

| Check | Expected observation | State |
|---|---|---|
| `go test ./internal/platform/config ./internal/datafeed ./cmd/worker` in Surveil | Data mode, local candidate client/config, and worker wiring tests pass | verified locally |
| `go test ./...` in shared/auth | Durable auth, private admin, migrations, and config tests pass; DB-dependent tests skip without the configured test database | verified locally |
| `node --test tests/local-compose-policy.test.mjs tests/preprod-data-boundary.test.mjs tests/local-runtime-contract.test.mjs` | Local Compose and preprod boundary contracts pass, 22/22 | verified locally |
| `PYTHONDONTWRITEBYTECODE=1 PYTHONPATH=src python3 -m unittest tests.test_aviasurveil_candidate_server tests.test_aviasurveil_phase2_3_ingestion tests.test_aviasurveil_phase2_4_ingestion` in AviaData | Local candidate mapping, manifest, and ingestion contract tests pass, 20/20 | verified locally |
| `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests -p 'test_*.py'` in Workspace | Workspace contract, component metadata, lock digest, and secret-preparation tests pass, 13/13 | verified locally |
| `python3 scripts/workspace.py validate` in Workspace | Catalog, exact target matrix, digest-bound lock, and infrastructure-route validation passes | verified locally |
| `make repo-check` in AviaData | Repository gate includes the migration metadata and no-secret checks; 46/46 focused tests pass | verified locally |
| `make compose-config TARGET=namibia/dev` in Workspace | Workspace dev Compose resolves the Data admission service, local worker mode, new secret, and 8081 admin declarations | verified locally |
| `make aviasurveil-phase2-4-check` in AviaData | Candidate static/tests pass, then the existing live PostgreSQL/SeaweedFS phase gate runs | blocked; Docker daemon unavailable (`DOCKER_DAEMON_UNAVAILABLE`) |
| `node --test tests/harness-docs-smoke.test.js` in AviaSurveil | Plan, index, tracker, and harness documentation links remain coherent | verified locally |
| `git diff --check` in each changed repository | No whitespace errors | verified locally |
| Active-path scan | No retired local checkout path in active plan instructions; historical evidence remains allowed | verified locally |
| Workspace operational doctor | Live auth/data/database/object-store readiness and cross-repository E2E | not run; candidate runtime requires Docker services and external release evidence |
| Shared AviaAuth ready/admin runtime | Durable implementation and cross-repository E2E | implementation verified locally; DB/mail/browser E2E and released image pending |
| Connected AviaData mode | Live PostgreSQL/SeaweedFS admission, endpoint/auth/image/fallback contract | candidate-only; live Docker E2E not run |
| Environment cutover and release locks | Exact target matrix and tamper-evident lock files | lock structure verified locally; image publication and cloud cutover release pending |
| Deployment | Cloud infrastructure/application deployment and traffic | not run; requires exact target authorization, credentials, and owner approval |

The local Compose test is candidate validation only. It cannot establish
remote deployment, production readiness, external identity, release
provenance, or customer-data safety.

## Acceptance criteria

The repository-local implementation slice is complete when:

- `AVIA_DATA` defaults to `0`, accepts only `0|1`, rejects feed settings while
  disabled, and fails closed for the unreleased enabled path.
- Namibia dev composes the local Data admission candidate and worker with
  file-backed secrets; non-dev release targets stay fail-closed until their
  published images and runtime evidence exist.
- component ownership and deployment authority are explicit and consistent in
  metadata and living docs.
- embedded `apps/auth` is removed, shared/auth owns the durable implementation,
  and Workspace generates/mounts the private 8081 admin secret.
- the source/fixture migration is forward-only, historical IDs are unchanged,
  and the migration metadata is synchronized across JSON, SQL, and ADR.
- cloud release locks are immutable and digest-bound, with missing image
  values still failing closed.
- active plan prompts use the current checkout path.
- focused tests and diff checks are `verified locally`, with every external or
  missing-runtime gate labeled literally.
- unrelated dirty files remain unchanged and no commit/push/deploy occurs.

The plan remains active only for the external qualification and release gates:
live cross-repository E2E, independently published image digests, exact target
release approval, and cloud deployment. Those gates cannot be claimed from
local source edits or Compose configuration.

## Risks, dependencies, idempotence, and recovery

- **Shared Auth dependency:** The durable implementation is now in shared/auth,
  but an image that lacks its database/mail/readiness behavior must not be
  promoted. Recovery is to keep release selection blocked until the candidate
  E2E suite passes.
- **Secret persistence:** `prepare_dev` uses create-only writes. Re-running it
  preserves existing values and regenerates only missing files. A disposable
  `.state` can be removed only through a separately authorized cleanup action;
  this plan does not delete it.
- **Data enablement:** Namibia dev uses only the explicitly named local
  candidate. No external source or compatibility fallback is retained; live
  release targets remain blocked until the connected contract is qualified.
- **Historical metadata:** Migration `0009` registers new canonical IDs without
  rewriting source-bound URLs, pinned revisions, fixture bytes, or evidence.
  Any broader namespace change requires a new coordinated versioned migration.
- **Dirty worktrees:** Existing infra, release-lock, environment, and other
  user changes are outside this plan and must remain untouched.

## Current outcome

Implemented locally in this thread:

- explicit Surveil Data mode validation and tests;
- local PostgreSQL/SeaweedFS AviaData admission candidate and worker loop;
- Surveil ownership/component metadata and docs;
- Workspace private-admin secret declaration and dev preparation;
- embedded auth removal and shared/auth durable source migration;
- forward-only AviaData source/fixture identity migration and ADR;
- immutable, digest-bound demo/preprod/prod release lock validation;
- Namibia dev environment cutover to the local Data candidate;
- active plan checkout-path cleanup;
- synchronized Workspace integration-gap status;
- fail-closed cross-repository component metadata enforcement and canonical
  AviaData image metadata;
- focused Go/Node/Python checks and static Compose/validator/lock checks.

Still blocked or intentionally deferred:

- live PostgreSQL/mail/browser and cross-repository E2E qualification;
- released immutable image publication and filling the release locks with
  verified digests (preprod/prod remain null by design);
- deleting/renaming the legacy local-preprod qualification model;
- cloud infrastructure/application deployment, traffic, and production
  evidence.

## Execution Prompt

Continue this plan in the current local repositories. Preserve unrelated dirty
files. Do not create, switch, rename, or delete branches; do not commit, push,
deploy, or perform remote/cost-bearing actions. Keep application behavior in
`apps/surveil`, platform composition and release authority in `workspace`,
durable identity in `shared/auth`, and source/data contracts in `shared/data`.
Run the focused verification matrix, repair only failures caused by this
plan, and update this plan, `docs/exec-plans/index.md`, and
`docs/exec-plans/tech-debt-tracker.md` with literal evidence states. Never
claim runtime readiness from declarative Compose wiring, and never rename
historical AviaData identifiers without a coordinated versioned contract
migration.
