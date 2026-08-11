# Task 10 security, dependency, recovery, and ARM64 gates

Date: 2026-08-11

Status: `candidate-only`; local code/image gates are `verified locally`, but
the release result is `NO-GO` because independent security review, full
browser/recovery/backup qualification, and native ARM64 mixed-workload
capacity evidence are not available. `production-ready: not established`.

## Verified local gates

| Gate | Result |
| --- | --- |
| `gofmt` and `git diff --check` | `verified locally` |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test ./... -count=1` | `verified locally` — all auth packages passed |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test -race ./... -count=1` | `verified locally` — all auth race tests passed before and after the security dependency update |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go vet ./...` | `verified locally` |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test ./internal/identity -run '^$' -fuzz=FuzzIdentifierNormalizationNeverPanics -fuzztime=3s` | `verified locally` — 138,011 executions, 204 new interesting inputs, no panic/failure |
| Disposable PostgreSQL migration/identity/session suites | `verified locally` — PostgreSQL 17.6; task-owned container removed |
| `npm --prefix apps/web run typecheck` | `verified locally` |
| `npm --prefix apps/web test` | `verified locally` — 85 files, 751 tests passed |
| `npm --prefix apps/web run build:demo` and `build:http` | `verified locally` — both builds completed; existing chunk warnings only |
| API identity/administration suites with disposable HTTP fixtures | `verified locally` |
| `node --test tests/harness-docs-smoke.test.js` and Keycloak realm contract tests | `verified locally` |
| `bash scripts/check-compose-policy.sh deploy/local/auth/compose.auth-candidate.yaml` | `verified locally` — all 21 local policy checks passed |
| `go mod verify` (`apps/auth`) | `verified locally` |
| CycloneDX SBOM via `syft` | `verified locally` — CycloneDX 1.7, 119 components written to disposable `/private/tmp` output |
| `trivy image --quiet --severity HIGH,CRITICAL --exit-code 1 aviasurveil360/auth-candidate:local` | `verified locally` — zero HIGH/CRITICAL findings after upgrading `golang.org/x/text` from v0.37.0 to fixed v0.39.0 for CVE-2026-56852 |
| rebuilt candidate image inspection | `verified locally` — linux/arm64, user `10001:10001`, digest `sha256:8ce1f635c1511f04b647763af03b8b62099486256f2ded64c317fa927582acc7` |

## Not run or blocked release gates

- `apps/api` `go mod verify` was `blocked` by unavailable network access for a
  module-cache metadata lookup; the focused API packages and tests still
  compiled and passed with the existing cache.
- Full Playwright OIDC browser profile, accessibility, dependency-loss/
  restart, Mailpit delivery/retry, backup/restore, key-custody recovery,
  distributed limiter, and native ARM64 steady/login-burst/Argon2id/PDF/
  notification/recovery load measurements are `not run`.
- Current external vulnerability/license/secret/misconfiguration scanning for
  the full repository, provenance attestation, and an independent security
  review are `not run`.
- No production database, identity, secret, DNS, SMTP provider, deployment,
  traffic, migration, cutover, or Keycloak retirement action was performed.

Because required release evidence is missing, Keycloak remains the active
provider and rollback baseline. Task 11 remains separately unauthorized.

## Fresh continuation evidence (2026-08-11)

| Gate | Result |
| --- | --- |
| `go test ./...` and `go test -race ./...` in `apps/auth` with an isolated module/build cache | `verified locally` — all packages, including the loopback OIDC protocol harness, passed after the test runner was granted loopback binding. |
| `go vet ./...` and `go mod verify` in `apps/auth` | `verified locally` |
| `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./cmd/auth` | `verified locally` — static ARM64 executable built in a disposable path. |
| `bash scripts/check-compose-policy.sh deploy/local/auth/compose.auth-candidate.yaml` | `verified locally` — 21/21 policy checks passed. |
| Fresh local candidate OCI build | `verified locally` — `linux/arm64`, non-root `10001:10001`, local digest `sha256:af1e47f377481372d57288dca40ab242e8e24d79efd46fb7e84dfa32c8451feb`; the task-owned image was removed after inspection. |
| `go mod verify` in `apps/api` with an isolated module/build cache | `verified locally` |
| `scripts/test-auth-candidate-postgres.sh` | `verified locally` — a task-owned PostgreSQL 17.6 project passed identity lifecycle, durable session rotation/reuse/expiry/revocation, forward-only migrations, and encrypted mail-outbox retry/lease-recovery tests; its container, volume, and runtime directory were removed. |
| `scripts/test-auth-candidate-postgres.sh` MFA extension | `verified locally` — forward-only `000005_mfa.up.sql` and durable encrypted TOTP state, replay rejection, hashed recovery codes, bounded recovery failures, and reset passed in the same isolated project; it does not make an MFA route available. |
| `scripts/test-auth-candidate-postgres.sh` challenge extension | `verified locally` — forward-only `000006_challenges.up.sql` and durable hashed challenge state, subject/purpose binding, attempt budget, consume/invalidate/cleanup, and concurrent single-use consumption passed in the same isolated project; it does not make a challenge route available. |
| `scripts/test-auth-candidate-postgres.sh` OIDC storage extension | `verified locally` — forward-only `000007_oidc_runtime.up.sql` and durable selected-library storage passed exact-client authentication, encrypted signing-key retrieval, authorization-code state, and refresh rotation/reuse denial; it does not make an OIDC route available. |
| `scripts/test-auth-candidate-mailpit.sh` | `verified locally` — a task-owned Mailpit instance required STARTTLS and SMTP authentication; `TestMailpitSTARTTLSDelivery` verified certificate validation and receipt visibility through its HTTP API. The container and ephemeral certificate/secret directory were removed. |
| `scripts/test-auth-candidate-mailpit-outbox.sh` | `verified locally` — task-owned PostgreSQL plus authenticated, certificate-verified STARTTLS Mailpit exercised the encrypted outbox: a first transient failure became retryable, the retry delivered through SMTP, and Mailpit’s receipt API confirmed it. The Compose project, container, volume, and temp credentials were removed. |

The current full browser OIDC/MFA/recovery/reset/logout, backup/restore,
restart/dependency-loss, key-rotation or Keycloak rollback-traffic, and native
ARM64 mixed-load/capacity gates are `blocked` by the unmounted candidate
runtime routes and the opt-in candidate Compose topology that is not yet wired
to PostgreSQL/Mailpit. SBOM, current license inventory, vulnerability scan, and
signed provenance are `not run` for this fresh image because the required
`syft`, `trivy`, and provenance tooling are absent from this machine. These
results do not alter the independent-security-review `not run` status.

The task-owned candidate image and temporary ARM64 executable were removed.
Cleanup of `/private/tmp/avia-auth-go-mod-cache` is `blocked`: the elevated Go
download wrote root-owned files and both direct and task-owned-container removal
attempts returned permission denied. `/private/tmp/avia-api-go-mod-cache` was
removed. No container or browser process remains. The remaining cache path is
operational residue, not repository evidence or runtime state.

## Isolated browser attempt (2026-08-12)

The isolated browser OIDC/MFA qualification is `blocked`: the disposable,
task-owned browser profile reached the provider callback redirect but aborted
`/authorize/callback` with `net::ERR_ABORTED` before the registered local
callback. It is not `verified locally`; recovery/reset/logout browser evidence
is also `not run`. The candidate remains `candidate-only`, Keycloak remains
the serving and rollback baseline, and release remains `release pending`.

The task-owned runtime dependency-loss and restart check is `verified locally`:
stopping authenticated STARTTLS Mailpit changed readiness to HTTP 503 without
changing liveness; restoration and an auth-container restart returned readiness
to HTTP 200 using durable PostgreSQL state. This remains `candidate-only`;
browser OIDC/MFA/recovery/reset/logout remains `blocked`, and release remains
`release pending`.

`scripts/test-auth-candidate-backup-restore.sh` is `verified locally` for a
task-owned PostgreSQL dump, removal and restoration of only the disposable
`auth_identity` schema, and successful auth-runtime readiness plus synthetic
active-account presence after restore. This is `candidate-only`; browser
OIDC/MFA/recovery/reset/logout remains `blocked`, and release remains `release
pending`.

## Runtime topology continuation (2026-08-12)

The isolated runtime topology is `verified locally`: its task-owned generated
secrets, dedicated PostgreSQL, authenticated STARTTLS Mailpit, and auth service
started and returned private-network liveness, dependency-gated readiness, and
OIDC discovery. This evidence is `candidate-only`; browser flows, recovery,
reset, explicit logout, restart/dependency-loss, backup/restore, rollback
traffic, native ARM64 mixed-load/capacity, fresh SBOM/license/provenance, and
independent review are `not run`. Release remains `release pending` and
literal `NO-GO`; Keycloak remains the serving provider and rollback baseline.

The candidate runtime's generic password recovery request and its encrypted
outbox delivery to authenticated STARTTLS Mailpit are `verified locally` using
one disposable `.invalid` account; the raw one-time value was not emitted.
Reset-form consumption is `verified locally` in the durable provider suite;
browser evidence remains `not run`. This does not change the `candidate-only`,
`release pending`, literal `NO-GO` disposition.

## Browser qualification completion (2026-08-12)

The isolated browser candidate gate is `verified locally`: a task-owned
ephemeral Playwright context completed OIDC Authorization Code with S256 PKCE
and standard `form_post`, provider-owned password login, TOTP MFA, generic
recovery initiation, password reset, MFA reset, post-reset password login
without MFA, and explicit logout against the disposable PostgreSQL and
authenticated STARTTLS Mailpit topology. All task-owned browser and Compose
resources were removed. This gate remains `candidate-only`; rollback traffic,
native ARM64 mixed-load capacity, fresh SBOM/license/vulnerability/provenance,
independent security review, and release approval are `not run`. Keycloak
remains serving and the rollback baseline, so release remains `release pending`
with literal `NO-GO`.

After the browser qualification change, the full `apps/auth` `go test ./...`,
`go vet ./...`, and `go test -race ./...` are `verified locally` with
task-owned temporary Go caches. The Compose policy, harness-docs smoke, and
diff checks are also `verified locally`. These checks do not change any `not
run` release gate.

## Native ARM64 bounded auth mixed-load qualification (2026-08-12)

`scripts/test-auth-candidate-load.sh` is `verified locally` on the `arm64`
host using only the disposable auth candidate, PostgreSQL, and authenticated
STARTTLS Mailpit topology. It measured 590 ms for two concurrent successful
Authorization Code + S256 PKCE password logins at the configured two-operation
Argon2id capacity, two concurrent rejected unknown-account Argon2id attempts,
four recovery requests with matching Mailpit outbox receipts, and concurrent
readiness/discovery probes. The runner fails if any result violates the
configured Argon2id capacity boundary or recovery delivery count, then removes
all task-owned resources. This is `candidate-only`; complete gateway/API/worker/
PDF mixed-load capacity, release headroom, and sustained-swap evidence remain
`not run`. Independent security review and release approval remain `not run`;
Keycloak remains serving and the rollback baseline, so release remains `release
pending` with literal `NO-GO`.

## Candidate provider-form accessibility qualification (2026-08-12)

The isolated browser qualification is `verified locally` for semantic
accessibility of the provider-owned login, MFA, recovery, password-reset, and
MFA-reset forms: English document language, exactly one main landmark and
level-one heading, one submit button, and accessible label/control pairs with
required non-checkbox inputs. This remains `candidate-only`; complete product
accessibility and release accessibility evidence are `not run`. Keycloak remains
serving and the rollback baseline; release remains `release pending` with
literal `NO-GO`.

## Durable signing-key rotation qualification (2026-08-12)

The disposable PostgreSQL provider suite is `verified locally` for encrypted
RSA signing-key rotation, finite old-key JWKS overlap, and elapsed-overlap
retirement. This is `candidate-only`; key-custody operations, complete release
provenance, independent security review, and release approval remain `not run`.
Keycloak remains serving and the rollback baseline; release remains `release
pending` with literal `NO-GO`.

## Fresh module and web qualification (2026-08-12)

`go mod verify` in both `apps/auth` and `apps/api`, `npm --prefix apps/web run`
`typecheck`, and `npm --prefix apps/web test` are `verified locally`; the web
suite passed 85 files and 751 tests. `syft`, `trivy`, and `govulncheck` are not
installed, and `docker sbom` is an unknown command, so a fresh SBOM/license/
vulnerability result remains `not run`. This does not change independent
security review, provenance, Keycloak baseline, `candidate-only`, or `release
pending` status.

## Local/disposable gate closure audit (2026-08-12)

All technically solvable isolated candidate gates are `verified locally`:
durable provider state, login/MFA/recovery/reset/logout, Mailpit, dependency
loss/restart, backup/restore, browser protocol/provider-form semantics, bounded
ARM64 auth-only load, signing-key overlap/retirement, module/web integrity, and
focused/full/race/vet checks. Remaining authenticated provider administration,
application BFF E2E, organization denial, Keycloak rollback traffic, complete
gateway/API/worker/PDF capacity, complete product accessibility, fresh
SBOM/license/vulnerability/provenance, independent review, and release approval
are `not run` or `blocked` at their recorded scope. The result remains
`candidate-only`; Keycloak remains serving and rollback baseline, release is
`release pending`, and literal `NO-GO` remains.
