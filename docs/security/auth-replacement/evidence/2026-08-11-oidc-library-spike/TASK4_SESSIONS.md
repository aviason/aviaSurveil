# Task 4 provider sessions and refresh families

Date: 2026-08-11

Status: `verified locally` for the in-memory and PostgreSQL session adapters,
refresh-family rotation, revocation callbacks, migration boundaries, race
coverage, and disposable integration database. The result remains
`candidate-only`; the provider is not serving application traffic.

## Implemented boundary

- `internal/session` issues opaque `ses_`/`fam_` identifiers and returns a raw
  refresh credential only at issuance/rotation. Stored credentials are
  SHA-256 hashes; fingerprint material is an HMAC with a provider-owned key.
- Client and exact redirect validation happens before session writes. The
  authorizer checks the immutable subject and auth revision before issuance or
  rotation.
- Rotation locks one family, marks the predecessor used, advances generation,
  caps idle expiry by the absolute expiry, and rejects wrong client,
  fingerprint, stale authorization, expired, revoked, or reused credentials.
  Reuse revokes the complete family.
- Family counts are bounded per subject. Logout, password changes, factor
  changes, lockout, suspension, and deactivation call `RevokeAllSessions`.
- `migrations/000002_sessions.up.sql` stores provider sessions, refresh
  families, and one-use token history with state/length/check constraints.
  The PostgreSQL adapter uses row locks and performs the same checks and
  cleanup durably.

## Fresh verification

| Command | Result |
| --- | --- |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test ./... -count=1` | `verified locally` — all auth packages, including provider HTTP harness, passed |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test -race ./... -count=1` | `verified locally` — concurrent in-memory rotation/reuse and all package races passed |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go vet ./...` | `verified locally` |
| `AVIA_AUTH_TEST_DATABASE_URL=... go test ./internal/session ./internal/identity ./migrations -run '^TestPostgreSQL' -count=1 -v` | `verified locally` — durable rotation/reuse, expiry/revocation, lifecycle revocation, and migration idempotence passed against disposable PostgreSQL 17.6 |
| task-owned PostgreSQL container cleanup | `verified locally` — `avia-auth-pg-task4` removed after the run |

Native ARM64 capacity, multi-instance distributed cleanup, crash/restart
recovery, and independent security review are `not run`; they are later
qualification/release gates, not claims established by this task.
