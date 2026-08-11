# Task 7 security audit and provider administration

Date: 2026-08-11

Status: `verified locally` for the bounded in-memory audit/admin components,
provider-neutral API boundary, exact client registry, RS256 key-ring policy,
and migration constraints. The result remains `candidate-only`; no internal
admin endpoint is exposed and no provider-admin secret is active.

## Implemented boundary

- `internal/audit` appends bounded events, clones snapshots, validates opaque
  subjects, and redacts keys containing password, secret, token, private-key,
  MFA, recovery, cookie, authorization, or code material before storage.
- `internal/admin` requires exact client IDs and exact HTTPS/loopback redirect
  values (no wildcards, prefixes, fragments, or arbitrary post-logout URLs),
  stores only client-secret hashes, and supports active/revoked states.
- `KeyRing` accepts RS256 RSA keys of at least 2048 bits, permits one active
  key plus explicit overlap/retired states, and bounds rotation/retirement.
- `apps/api/internal/identity/provider_admin.go` defines the provider-neutral
  lifecycle/directory boundary. The current Keycloak adapter satisfies it as
  the retained migration baseline; application callers no longer need a
  Keycloak-shaped interface.
- `migrations/000003_provider_admin_audit.up.sql` creates exact client/key and
  append-only audit tables with state, key-size/hash, active-key uniqueness,
  subject, and JSON-object constraints.

## Fresh verification

| Command | Result |
| --- | --- |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test ./internal/admin ./internal/audit ./migrations -count=1` | `verified locally` — client/redirect, secret-hash, key-ring, redaction, bounded append, and migration boundary tests passed |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test -race ./internal/admin ./internal/audit ./migrations -count=1` | `verified locally` |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-api-go-cache go test ./internal/identity ./internal/administration -run 'TestProviderAdminBoundaryIsProviderNeutral\|^$' -count=1` | `verified locally` — the retained Keycloak adapter compiles against the provider-neutral boundary |

Durable audit append/fail-closed policy, authenticated provider-admin HTTP
handlers, authorization matrices, operational audit export, and independent
security review are `not run`; those are required before any production
authority or cutover claim.
