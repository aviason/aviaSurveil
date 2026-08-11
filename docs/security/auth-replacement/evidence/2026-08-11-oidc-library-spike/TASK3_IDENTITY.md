# Task 3 identity, password, account-state, and abuse-control evidence

Date: 2026-08-11

Status: `verified locally` for the provider-neutral identity domain,
forward-only PostgreSQL identity schema, Argon2id admission/policy, dummy-hash
authentication path, account-state transitions, canonical identifier
uniqueness, trusted-proxy resolution, fail-closed throttling, and disposable
PostgreSQL lifecycle tests. The result remains `candidate-only`; no normal API
or Keycloak route consumes this provider yet.

## Implemented boundaries

- `internal/identity` allocates opaque `usr_` subjects, normalizes one typed
  email/username namespace, enforces global normalized-value uniqueness,
  rejects public registration, and keeps invitation, verification, activation,
  password, state, and revision changes separate.
- Account states are `invited`, `active`, `disabled`, `suspended`, `locked`,
  `deletion-pending`, and `deleted`. Mutations require the expected auth
  revision and fail closed on stale writes. Active credentials require a
  verified email.
- `internal/password` uses bounded Argon2id with a fixed dummy hash for
  unknown/malformed accounts, random salts, encoded parameter validation,
  maximum password bytes, bounded concurrent hashing admission, minimum
  policy, compromised-password hook, and current/history reuse denial.
- `internal/throttle` applies IP, canonical identifier, and device keys as one
  atomic admission decision. Limiter failure is fail-closed. Forwarded client
  identity is accepted only from configured gateway prefixes; untrusted or
  malformed forwarding cannot spoof the address.
- `internal/identity/postgres.go` mirrors the domain mutations using row locks,
  revision checks, atomic failure/lock accounting, password-history rows, and
  the same generic authentication outcomes.
- `migrations/000001_identity.up.sql` creates the separate `auth_identity`
  schema, account/identifier/history/invitation/throttle tables, immutable
  subject shape checks, active-email verification check, and cross-field
  normalized identifier uniqueness. The migration never creates a role or
  grants application/worker access.

## Fresh verification

| Command | Result |
| --- | --- |
| `go test ./... -count=1` (from `apps/auth`) | `verified locally` |
| `go test -race ./internal/password ./internal/throttle ./internal/identity ./migrations -count=1` | `verified locally` |
| `go vet ./...` | `verified locally` |
| `AVIA_AUTH_TEST_DATABASE_URL=... go test ./internal/identity ./migrations -run 'TestPostgreSQL\|TestIdentityMigration' -count=1 -v` against disposable PostgreSQL 17.6 | `verified locally` — migration idempotence, subject/check constraints, global duplicate identifier denial, invitation → verified → active, login, password change/history, stale revision, and suspension passed |
| `git diff --check` | `verified locally` |

The PostgreSQL test used a task-owned disposable container and synthetic
credentials/identities only; the container was removed after the run. No
application database, Keycloak database, production identity, or secret was
read or changed.

## Remaining scope

Provider OIDC routes, refresh families, MFA/recovery, audit, API adapter,
browser flow, distributed production limiter, backup/restore, and cutover are
`not run` and belong to later ordered tasks. The in-memory store remains a
unit-test adapter; the PostgreSQL store is the candidate persistence path but
is not wired into the Task 2 process until the later provider-runtime tasks.
