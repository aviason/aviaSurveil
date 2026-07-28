# Preprod Identity And Data Foundation Evidence

**Evidence date:** 2026-07-28

**Plan:** Plan 5 — Identity And Realistic Data Foundation

**Task status:** `complete — verified locally and published`

**Artifact status:** `candidate-only`

**Release status:** `release pending`

## Predecessor Gate

Plan 5 Tasks 1–8 are complete, `verified locally`, published, and remotely
confirmed. Task 8 implementation/evidence was published as
`9a1b5cadff82a047bc0400acdb39f3fab2fbecab`; its publication metadata was
published as `552d5afde52774d1069e003de1c902dfe3aac540`. The active local
qualification set is `smoke@1.0.0`, `acceptance@1.0.0`,
`realistic@1.1.0`, and `stress@1.1.0`.

Task 9 implementation and final evidence were published as
`9d30576876406609577af4381718e8ac66e94e06`; the exact `origin/main`
revision was confirmed after the non-force push.

Owner directive `OWNER-DIRECTIVE-2026-07-28-P5T8-01` retains the unchanged
full-volume `realistic@1.0.0` and `stress@1.0.0` profiles as deferred
release-readiness endurance evidence with literal status `not run`.

## Strict RED

Before this evidence file was created:

| Command | Literal result |
|---|---|
| `test -f docs/demo-evidence/PREPROD_IDENTITY_AND_DATA_FOUNDATION_2026-07-27.md` | exit 1; required Task 9 final evidence artifact absent |

## Final Matrix

The matrix is complete. Literal results, including transient failures and the
exact owner-approved stress exception, remain recorded.

The first full Go race attempt passed all packages through
`preproddata/scenarios` but failed to build `tests/integration` because eight
canonical HTTP integration files were missing the `canonicaltest` build tag.
The strict focused Node RED was 7/8 with the same undefined symbols. A minimal
tag/helper separation fix now passes both graphs: the untagged boundary suite
is 8/8 and `go -C apps/api test -tags canonicaltest -run '^$'
./tests/integration` passed in 0.666 seconds. The second full-race attempt
passed `preproddata/scenarios` in 203.363 seconds and compiled integration, but
the live tests then failed because the required task-owned PostgreSQL and MinIO
endpoints were not running (`127.0.0.1:55432` and `127.0.0.1:59001`
connection refused). These attempts are not accepted matrix evidence; the
exact command requires an environment-backed clean rerun.

The environment-backed exact rerun passed every package, including
`internal/preproddata/scenarios` in 199.981 seconds and `tests/integration` in
55.441 seconds. After the successful Go process, the ad hoc outer shell
returned exit 1 because its cleanup function assigned to zsh's read-only
`status` parameter. This was a wrapper cleanup defect rather than a test
failure. The exact task-owned Docker project and generated runtime directory
were then explicitly removed; independent Docker queries returned zero
container, volume, or network residue, and protected PID 13055 remained alive.

The first local-full attempt accepted the prior digest/SBOM/scan manifest, then
failed closed when its worker became unhealthy. Its cleanup removed the exact
task-owned project and state. A separately scoped diagnostic reproduction
showed that the accepted API and worker images still identified source revision
`3acedc80c7c4545e715fa694ea1e2fbc6163447d`. Those binaries/entrypoints predated
the Plan 5 least-privilege Keycloak service-client transition: the stale API
required `AVIA_KEYCLOAK_ADMIN_USERNAME`, and the stale worker expected the
removed bootstrap-admin password mount. The diagnostic project was also fully
removed. Current-source image build, digest-bound SBOM generation,
HIGH/CRITICAL scanning, and a clean local-full rerun are required; neither
failed run is accepted matrix evidence.

The current-source prerequisite then completed 9/9 local image builds, 9/9
digest-bound CycloneDX SBOMs, and 9/9 fail-closed HIGH/CRITICAL scans. The next
local-full run proved the rebuilt API and worker healthy, then stopped before
browser evidence because the runtime checker expected Mailpit on `platform`
only. The maintained Compose policy intentionally requires `identity` for
Keycloak delivery plus `platform` for worker delivery. Focused strict RED was
6/7; after synchronizing the exact checker expectation to `identity platform`,
the runtime and Compose-policy suites passed 28/28. The failed full-profile
project and state were fully removed; this attempt is not accepted matrix
evidence.

The next clean run reached Playwright and failed because the bootstrap Admin
had no authoritative desired membership while the historical browser fixture
attempted to give one provider account seven CAA roles. The strict
local-profile contract RED passed 11/13. The harness was changed to create the
Admin membership through the authoritative one-shot identity setup and to use
eight distinct exact-role sessions. Its first live attempt then failed because
`docker compose cp` could not write into the read-only API root filesystem.
The next attempt used an auto-removed `compose run --rm --no-deps` setup
container with a read-only bind-mounted binary, but failed because the
bootstrap script and lifecycle both attempted to create the same Keycloak
email. Every failed project completed exact cleanup. The final ordering lets
the lifecycle create the Admin before the harness verifies exact CAA/Admin
provider authority and configures only the test password and
`CONFIGURE_TOTP`.

The first Task 9 OIDC-profile run passed one-shot identity setup and diagnosed
the active administrator as exact CAA/Admin authority in provider,
application, and session state. Its browser test then timed out after 360
seconds because the historical E2E selector expected the removed
`Submit provisioning` button. The maintained UI requires
`Review provisioning` followed by the explicit `Confirm Provision` dialog.
The RED run's generated-secret scan found zero matches and exact Docker cleanup
left zero residue. After synchronizing the E2E with that confirmation boundary,
Web typecheck, the focused Users/Roles suite (6/6), stale-selector scan, and
`git diff --check` passed. The clean OIDC rerun then exited 0: its one
real-browser MFA/provisioning/session-revocation test passed in 50.4 seconds,
the generated-secret/log scan found zero matches, and exact task-owned cleanup
completed.

The first Task 9 `realistic@1.1.0` rerun produced and reconciled every exact
count, relationship, privacy canary, and resume state but failed closed because
generation took 858 seconds and total qualification took 1,085/900 seconds.
Cleanup completed in 11 seconds with zero residue. The create-only run
`run-task8-realistic-20260728-154050-89149` is preserved as failed evidence.
Comparison with the accepted Task 8 generation time of 228 seconds exposed
redundant Keycloak round trips: every synthetic account fetched a fresh
service token and the same realm-role representation. The focused strict RED
reported `Keycloak service-token requests = 3, expected 1`. The endpoint now
reuses only an unexpired `expires_in`-bounded service token and validated
realm-role metadata while retaining exact per-account user/role
reconciliation. The focused test passed; Node boundary/scale tests passed
12/12, Go vet and `git diff --check` passed, and the changed scenario package
passed its complete race suite in 217.252 seconds before the clean rerun. That
rerun reduced generation to 716 seconds but still failed closed at 966/900
qualification seconds; cleanup completed in 17 seconds with zero residue.
Create-only run `run-task8-realistic-20260728-160743-98228` is retained. The
next focused RED reported `Keycloak user reads = 4, expected 2`: newly created
users were read both before and after role mapping although only the
post-mapping read can prove final authority. The implementation now performs
one post-mapping exact user/role check for a new account and preserves the
complete final global provider reconciliation. The focused test passed,
boundary/scale tests passed 12/12, Go vet and `git diff --check` passed, and
the scenario package passed under race in 198.129 seconds. The next clean
rerun again reconciled every exact count, relationship, privacy canary, and
resume state, but generation took 740 seconds and total qualification took
999/900 seconds. Cleanup completed in 16 seconds with zero residue. Create-only
run `run-task8-realistic-20260728-163103-6332` is retained as failed evidence.
The final focused RED reported
`Keycloak user reads after listed reconciliation = 4, expected 2`: the complete
`briefRepresentation=false` list used for namespace reconciliation was
discarded and every account was fetched again. Reconciliation now validates
that already-listed complete representation directly while retaining the
exact per-account realm-role read. The focused test passed, boundary/scale
tests passed 12/12, Go vet and `git diff --check` passed, and the complete
scenario package passed under race in 199.467 seconds. The single bounded
rerun then passed as
`run-task8-realistic-20260728-165658-16700`: generation completed in 672
seconds and total qualification in 872/900 seconds; seven bounded evidence
files, 326 resource samples, 26.17 MiB peak loader memory, 1.064 peak CPU
cores, privacy findings 0, every resource envelope, exact reconciliation,
controlled interruption/resume, and 10-second whole-namespace cleanup all
passed. Independent Docker inspection found zero task-owned residue and PID
13055 remained alive.

The fresh Task 9 `stress@1.1.0` run
`run-task8-stress-20260728-171259-25353` completed its authoritative data run
with outcome `SUCCEEDED`: exact reconciliation, every data family and
relationship digest, privacy findings 0, controlled interruption/resume,
resource measurement, and whole-namespace cleanup all passed. It nevertheless
failed closed on the owner duration gate because generation took 1,624 seconds
and total qualification took 1,819/1,800 seconds. Cleanup completed in 12
seconds with residue 0. The create-only run retains nine bounded files,
including its failure record; 785 resource samples measured 28.62 MiB peak
loader memory and 0.648 peak CPU cores. Independent Docker inspection found
zero task-owned residue and PID 13055 remained alive. No automatic rerun was
started. On 2026-07-28 the owner explicitly approved
`OWNER-DIRECTIVE-2026-07-28-P5T9-01`, accepting only this exact retained run's
19-second qualification overrun for the Task 9 matrix. The exception does not
change `stress@1.1.0`, its 1,800-second fail-closed envelope, its exact volume,
or any data, relationship, privacy, resume, resource, or cleanup gate. The run
continues to record exit 1 and `qualification-duration-exceeded` literally;
the owner decision makes a repeat stress run unnecessary for Task 9.

After the final Keycloak reconciliation changes, the exact
`go -C apps/api test -race -p 1 -count=1 ./...` command was rerun in a clean,
task-owned PostgreSQL/MinIO environment and exited 0. Every package passed;
`internal/preproddata/scenarios` completed in 147.408 seconds and
`tests/integration` in 42.686 seconds. The Bash cleanup wrapper exited 0,
removed its containers, volumes, network, and temporary runtime directory,
and independent Docker queries found zero task-owned residue. PID 13055
remained alive.

A separate final main-agent code and specification review inspected the full
Task 9 diff after the final matrix. It confirmed the expiry-bounded Keycloak
token cache, exact provider-user and realm-role reconciliation, eight
distinct authoritative role memberships/sessions, the normal HTTP no-seed and
no-loader boundary, unchanged root demo, create-only evidence history, and
literal owner-exception/non-claim language. It found no Critical or Important
finding.

| Command | Literal result |
|---|---|
| `./scripts/check-contracts.sh` | exit 0; 16/16 OpenAPI tests passed; generated Go/TypeScript drift check `contracts-check: ok` |
| `./scripts/check-sqlc.sh` | exit 0; `sqlc-check: ok` |
| `node --test tests/preprod-identity-data-contract.test.mjs` | exit 0; 26/26 passed |
| `node --test tests/preprod-data-boundary.test.mjs` | exit 0; 8/8 passed |
| `node --test tests/preprod-data-scale-contract.test.mjs` | exit 0; 4/4 passed |
| `go -C apps/api test -race -p 1 -count=1 ./...` | final exit 0 after all Task 9 Go changes in a clean task-owned PostgreSQL/MinIO environment; every package passed; `internal/preproddata/scenarios` 147.408 s and `tests/integration` 42.686 s; wrapper exit 0 and independent cleanup inspection found zero container/volume/network residue |
| `npm --prefix apps/web run typecheck` | exit 0 |
| `npm --prefix apps/web test` | exit 0; 64/64 files and 650/650 tests passed in 181.71 s; emitted four non-failing jsdom `Not implemented: navigation to another Document` warnings |
| `./scripts/test-preprod-identity-lifecycle.sh all-eight-roles` | exit 0; canonical-tag integration package passed in 5.036 s; all-eight-role identity lifecycle `verified locally`; task-owned containers, volumes, and network removed |
| `node --test tests/*.test.js tests/parity/react-legacy-parity.test.mjs` | exit 0; 108/108 passed in 10.334 s |
| `npm --prefix apps/web run build:demo` | exit 0; 272 modules transformed; built in 1.61 s; non-failing >500 kB chunk advisory retained |
| `npm --prefix apps/web run build:http` | exit 0; 270 modules transformed; built in 1.72 s; non-failing >500 kB chunk advisory retained |
| `./scripts/test-normal-artifact-boundary.sh` | exit 0; normal and canonical-tag HTTP API package checks passed; `normal-artifact-boundary: ok` |
| `./scripts/test-local-full-profile.sh` | exit 0; one Playwright test passed in 1.1 minutes; eight distinct exact-role sessions covered all 86 routes and ten scenario families; nine required Mailpit messages reconciled; worker restart and complete runtime check passed; JSON reported `tests=1`, `skipped=0`, `status=verified locally`; exact project cleanup and independent Docker query found zero task-owned residue; PID 13055 remained alive |
| `./scripts/test-http-oidc-profile.sh` | exit 0; identity setup passed; one real-browser OIDC/MFA/provisioning/session-revocation test passed in 50.4 s; generated-secret/log scan found zero matches; exact task-owned cleanup completed |
| `./scripts/test-preprod-data-profile.sh smoke` | exit 0; run `run-task8-smoke-20260728-152926-81065`; exact 40 families, 86 routes, 306 actions, eight roles, privacy 45/45, controlled interruption/resume, and cleanup passed; seven bounded evidence files; 8 resource samples, 26 s generation, 8.719 MiB peak loader memory, 0.118 CPU cores, 236 s qualification, 4 s cleanup, all envelopes true, residue 0 |
| `./scripts/test-preprod-data-profile.sh acceptance` | exit 0; run `run-task8-acceptance-20260728-153414-85184`; exact 40 families, 86 routes, 306 actions, eight roles, privacy 45/45, controlled interruption/resume, and cleanup passed; seven bounded evidence files; 98 resource samples, 210 s generation, 24.9 MiB peak loader memory, 0.573 CPU cores, 361 s qualification, 5 s cleanup, all envelopes true, residue 0 |
| `./scripts/test-preprod-data-profile.sh realistic` | exit 0; run `run-task8-realistic-20260728-165658-16700`; exact 40 families, 86 routes, 306 actions, eight roles, privacy 45/45, controlled interruption/resume, and cleanup passed; seven bounded evidence files; 326 resource samples, 672 s generation, 26.17 MiB peak loader memory, 1.064 CPU cores, 872/900 s qualification, 10 s cleanup, all envelopes true, residue 0 |
| `./scripts/test-preprod-data-profile.sh stress` | owner-approved exact-run exception `OWNER-DIRECTIVE-2026-07-28-P5T9-01`; retained exit-1 run `run-task8-stress-20260728-171259-25353` completed the authoritative run with exact data/reconciliation/privacy/resume/resource/cleanup success, 785 resource samples, 1,624 s generation, 28.62 MiB peak loader memory, 0.648 CPU cores, and 12 s cleanup with residue 0, then recorded `qualification-duration-exceeded` at 1,819/1,800 s; the owner accepted only this 19-second Task 9 overrun without changing the profile or any gate |
| `git diff --check` | exit 0 after final Plan 5 code, evidence, plan move, index, tracker, profile, manifest, and build-summary synchronization |

## Non-Claims

AWS, deployment, production changes, real PII, full-volume
`realistic@1.0.0`/`stress@1.0.0` endurance, release approval, and production
readiness remain `not run`. Task 9 is `verified locally` under the exact
owner-approved stress exception; the artifact remains `candidate-only` and
release remains `release pending`. Task 9 is published as
`9d30576876406609577af4381718e8ac66e94e06` and remotely confirmed.
