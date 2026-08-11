# Task 2 isolated provider scaffold evidence

Date: 2026-08-11

Status: `verified locally` for the isolated scaffold, configuration contract,
focused tests, Compose syntax, static Linux ARM64 build, and local ARM64 image
build. The result remains `candidate-only`; release is `release pending` and
`production-ready: not established`.

## Authorization and boundary

Option A — `github.com/zitadel/oidc/v3` v3.47.5 (`fb9fbfe`) was explicitly
authorized by the owner on 2026-08-11. The selected dependency is used only
inside `apps/auth`; application Go backends remain relying parties using
`coreos/go-oidc` plus `golang.org/x/oauth2`.

Task 2 creates no normal-profile route, API integration, database migration,
identity, production secret, or Keycloak change. The candidate Compose file is
opt-in, publishes no port, uses an internal network, mounts no Docker socket,
and expects only externally-created disposable secrets.

## Implemented scaffold

- `apps/auth/go.mod` and `go.sum` pin the selected provider library and its
  resolved dependency graph.
- `internal/config` requires the isolated candidate profile, exact issuer and
  listener, a dedicated PostgreSQL role/schema, read-only secret files, an
  RSA 2048–8192-bit signing key restricted to RS256, distinct non-zero 32-byte
  data/MFA keys, verified TLS SMTP, and bounded request/timeouts. Inline
  secrets, placeholders, shared application/Keycloak ownership, writable
  secret files, plaintext SMTP, weak keys, production, and normal profiles
  fail closed before listening.
- `internal/provider` records the selected `zitadel/oidc` route vocabulary but
  deliberately mounts no OIDC endpoint until later storage and protocol tasks.
- `internal/httpserver` exposes liveness and fail-closed readiness only;
  discovery returns 404 and readiness remains 503 until a later provider
  storage/runtime probe is wired.
- `internal/telemetry` emits JSON with credential-shaped attributes redacted.
- `apps/auth/Dockerfile` is a pinned static multi-stage build, runs as UID/GID
  `10001:10001`, and includes a liveness healthcheck. The opt-in Compose file
  adds read-only storage, `no-new-privileges`, all capabilities dropped,
  bounded PIDs/resources/logs, an internal network, and no socket mount.
- `apps/auth/NOTICE.md` records the direct Apache-2.0 dependency notice.

## Fresh verification

| Command | Result |
| --- | --- |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test ./... -count=1` (from `apps/auth`) | `verified locally` — all packages passed |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test -race ./... -count=1` | `verified locally` — all packages passed |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go vet ./...` | `verified locally` |
| `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ... ./cmd/auth` | `verified locally` — static ARM64 ELF, 6.5 MiB |
| `docker compose -f deploy/local/auth/compose.auth-candidate.yaml config --quiet` | `verified locally` |
| `docker build --platform linux/arm64 --file apps/auth/Dockerfile --target runtime ...` | `verified locally` — local image manifest `sha256:8ce1f635c1511f04b647763af03b8b62099486256f2ded64c317fa927582acc7` |
| `docker image inspect ...` | `verified locally` — `linux/arm64`, user `10001:10001`, healthcheck present |
| ARM64 image with no configuration, read-only/no-network/dropped-capability runtime | `verified locally` — exits 1 before listening; no secret output |

The direct host-process listener smoke is `blocked` by the sandbox's TCP bind
permission (`listen tcp 127.0.0.1:18083: bind: operation not permitted`). The
handler, configuration-positive/negative, and container fail-closed checks are
independent of that sandbox limitation. Full provider startup with a real
database, SMTP, key store, migrations, OIDC discovery, browser flow, and
normal Compose routing are `not run` by Task 2 scope.
