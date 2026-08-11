# AviaSurveil360 authentication export

Status: **candidate integration input; not production-ready; not a Keycloak replacement**.

This source-only package was extracted from the EMSI Go API at revision
`60dbe494318106569f6d9dbea121d6b1c841ae95`. It contains the smallest cohesive
native password, JWT, refresh-session, and database-store implementation needed
for technical evaluation. It contains no repository history, environment file,
private key, credential, production URL, customer data, binary, cache, or
database dump.

## Accurate classification

The implementation is classification **3: native password/session
authentication without acting as an OIDC provider**. More precisely, it issues
proprietary RS256 access JWTs and opaque rotating refresh tokens through custom
GraphQL operations. It is not an OIDC relying party/client and does not consume
Keycloak tokens. It exposes discovery-shaped metadata and JWKS, but it lacks
the authorization, OAuth token, ID-token, userinfo, redirect, state, nonce, and
PKCE flows required of an OIDC/OAuth authorization server.

Consequently, it cannot safely replace Keycloak for AviaSurveil360 as exported.

## Export contents

- `auth/`: exact copies of the source auth middleware, Argon2id implementation,
  PostgreSQL store, token service, and their focused unit tests. `ports.go` is a
  clearly marked candidate interface proposal added only to this export.
- `auth/source_reference/`: non-compiling source references for the dormant
  legacy HTTP handler, GraphQL mapping, and auth-only schema contract.
- `migrations/`: one forward-only current-state projection, in exact export
  execution order, plus its source-migration mapping.
- `tests/`: export boundary and migration checks. The integration-tag test
  refuses any DSN not named as a task-owned auth test database.
- `fixtures/`: synthetic identities using the reserved `example.invalid`
  domain and no passwords or tokens.
- `openapi/` and `examples/`: metadata/JWKS contract and minimal chi-compatible
  mount adapter.

The source GraphQL resolver, application role workflow, privacy worker, and
runtime configuration are documented but not copied as compiled dependencies:
they are coupled to unrelated EMSI modules and contain controls that must be
rewritten for AviaSurveil360.

## Verification record

All commands were run locally against source-only or export-only data. No
external service or production/customer data was used.

| Scope | Exact command | Literal result |
|---|---|---|
| Source formatting | `gofmt -l .` | **passed**; exit 0, no files listed |
| Source auth unit tests | `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/private/tmp/aviasurveil360-source-go-cache go test -modfile=/private/tmp/aviasurveil360-source-auth.mod ./internal/auth` | **passed**; `ok emsi_go_api/internal/auth 0.855s` |
| Source auth race tests | `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/private/tmp/aviasurveil360-source-go-cache go test -race -modfile=/private/tmp/aviasurveil360-source-auth.mod ./internal/auth` | **passed**; `ok emsi_go_api/internal/auth 2.388s` |
| Source auth vet | `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/private/tmp/aviasurveil360-source-go-cache go vet -modfile=/private/tmp/aviasurveil360-source-auth.mod ./internal/auth` | **passed**; exit 0, no diagnostics |
| Whole source tests | `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/private/tmp/aviasurveil360-source-go-cache go test ./...` | **blocked**; the local module cache lacked required dependency archives and network lookup was deliberately disabled |
| Whole source vet | `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/private/tmp/aviasurveil360-source-go-cache go vet ./...` | **blocked**; required modules were unavailable in the local download cache |
| Source integration tests | PostgreSQL-backed auth/store tests | **blocked**; no task-owned PostgreSQL test database was configured and no such focused source tests exist |
| Source migration tests | Auth migration execution against PostgreSQL | **blocked**; no task-owned PostgreSQL test database was configured and no migration test suite exists |
| Source fuzz tests | `rg -n '^func Fuzz' internal/auth` | **not run**; no auth fuzz targets exist |
| Source dependency/secret/misconfiguration scan | `trivy fs --skip-db-update --skip-java-db-update --skip-check-update --skip-version-check --offline-scan --no-progress --scanners vuln,secret,misconfig --format json --output /private/tmp/aviasurveil360-source-trivy-offline.json .` | **passed with findings**; no secret finding; vulnerable dependencies reported, including critical `pgx/v5` advisories fixed in newer releases |
| Export formatting | `gofmt -l auth examples tests` | **passed**; exit 0, no files listed |
| Export unit tests | `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/private/tmp/aviasurveil360-export-test-cache go test -modfile=/private/tmp/aviasurveil360-export-verify.mod ./...` | **passed**; auth and export tests passed, examples compiled |
| Export race tests | `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/private/tmp/aviasurveil360-export-race-cache go test -race -modfile=/private/tmp/aviasurveil360-export-verify.mod ./auth ./tests` | **passed**; auth and export tests passed under the race detector |
| Export vet | `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/private/tmp/aviasurveil360-export-vet-cache go vet -modfile=/private/tmp/aviasurveil360-export-verify.mod ./...` | **passed**; exit 0, no diagnostics |
| Portable module dependency resolution | `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOCACHE=/private/tmp/aviasurveil360-export-direct-cache go test ./...` | **blocked**; the local module download cache lacked the dependency archive and external lookup was disabled. The verification modfile changed only dependency resolution to already-extracted local source; it did not change exported code. |
| Export dependency/secret/misconfiguration scan | `trivy fs --skip-db-update --skip-java-db-update --skip-check-update --skip-version-check --offline-scan --no-progress --scanners vuln,secret,misconfig --format json --output /private/tmp/aviasurveil360-export-trivy-offline.json .` | **passed with findings**; zero secret findings; 14 `x/crypto v0.51.0` advisories were reported, fixed in `v0.52.0`. Export dependency listing shows only `argon2` and `blake2b`, not the flagged SSH/OpenPGP packages. |
| Export forbidden-file/content scan | Explicit file, directory, private-key, token, credential-bearing DSN, and non-reserved URL patterns over the export tree | **passed** before and after manifest creation; zero forbidden files/directories, probable secret hits, or unexpected URLs |
| Candidate ZIP integrity/list scan | `unzip -t /private/tmp/aviasurveil360-auth-export.zip`, `zipinfo -1 /private/tmp/aviasurveil360-auth-export.zip`, path/symlink/duplicate validation, manifest checksum verification, and `diff -qr` against the export tree | **passed**; 30 files, 38 total file/directory entries, no corrupt, duplicate, absolute, traversal, symlink, missing, extra, or changed entry |
| Candidate ZIP content scan | Extract to a task-owned temporary directory, run the same explicit forbidden-content checks, then `trivy fs --skip-db-update --skip-java-db-update --skip-check-update --skip-version-check --offline-scan --no-progress --scanners vuln,secret,misconfig ...` | **passed**; zero forbidden files/directories, probable secrets, Trivy secret findings, or unexpected non-reserved URLs. Dependency advisories remain as documented. |

The source full-module checks are recorded as blocked rather than passed or
failed because the requested no-external-systems boundary prevented dependency
download and the local cache was incomplete.

## Highest-impact blockers

1. No conforming OAuth/OIDC authorization server, so existing issuer,
   discovery, JWKS, ID-token, authorization-code, and service-client contracts
   cannot be replaced unchanged.
2. No TOTP, recovery codes, WebAuthn, password reset, email verification,
   secure SMTP, administrative recovery, or four-language identity messaging.
3. No organization/tenant model or query-level isolation contract.
4. No disabled, suspended, or locked account enforcement; deletion-pending is
   not checked during login, refresh, or session validation.
5. No application login/registration rate limits, fixed-cost unknown-user
   verification, password strength policy, history, or rehash policy.
6. Password change leaves the current pre-change refresh family usable.
7. Security auditing is incomplete and several event writes are fail-open.
8. The source role assignment workflow is proposal-only and has authorization
   defects; it is not an administrative provisioning system.
9. The compact export migration is an evaluation projection and was not run
   against PostgreSQL.
10. Repository redistribution licensing is unclear because no root source
    license was found.

Read `CAPABILITY_MATRIX.md`, `SECURITY_REVIEW.md`, and `INTEGRATION_GUIDE.md`
before compiling or adapting the package.
