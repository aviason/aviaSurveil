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
