# Data model

`migrations/001_auth_identity.sql` is a compact evaluation projection of the
current auth-related schema, not a migration to apply to an existing
AviaSurveil360 database.

## Exported tables

### `app.users`

Application profile row keyed by numeric `id`. It stores mutable username,
email, phone, display fields, free-form JSON, a display role, and the source
account-deletion status. The copied auth store does not enforce
`account_status`.

### `app.auth_accounts`

One password account per user. `subject` is the JWT identity and has a unique
index; application registration creates a random `usr_` value. Username and
email have separate case-insensitive unique indexes. `auth_version` invalidates
old access tokens when incremented. There is no database guard making subject
immutable and no verified identifier/status/password-history fields.

### `app.auth_sessions`

Server-side session keyed by a random opaque text ID. It stores user/client
metadata, the current refresh-token hash, absolute and idle expiry, auth time,
last step-up, last use, and revocation data. `user_agent` is raw caller-provided
text. There is no organization key, issuance auth version, session cap, or
retention/cleanup policy.

### `app.auth_refresh_tokens`

Append-style rotating refresh records with token SHA-256 hash, session, user,
family, client/device signal, absolute/idle expiry, rotation/replacement, and
revocation fields. The device signal is an unkeyed hash of caller-repeatable
device and client strings; it is not proof-of-possession or attestation.

### `app.auth_security_events`

Identity event rows with optional user/session, event type, client, two
fingerprint fields, JSON metadata, and timestamp. Current writes leave the IP
and user-agent hash fields empty and omit important failure/lifecycle events.
This is not a complete append-only audit subsystem.

### `app.staff_permissions`

Live permission strings with a free-form scope, active flag, grant/revocation
timestamps, and optional granter. The copied store uses it only to hydrate
authority codes. It is not an organization membership model, and the source
assignment workflow does not complete grants/revocations.

### `app.cities` and `app.schema_migrations`

`cities` is retained only because the copied `UserByID`/search queries resolve a
profile city. `schema_migrations` records the export migration name. Neither is
an organization boundary.

## Relationships

```text
app.users 1---1 app.auth_accounts
    |
    +---* app.auth_sessions 1---* app.auth_refresh_tokens
    |             |
    |             +---* app.auth_security_events (nullable session/user)
    |
    +---* app.staff_permissions
```

## Required AviaSurveil360 additions

- Canonical login identifier rows with type, normalized value, verification
  state/time, and one uniqueness domain across every identifier accepted by
  login.
- Explicit account status with fail-closed transitions for invited, active,
  disabled, suspended, locked, deletion-pending, and deleted states as product
  policy requires.
- Password history/rehash metadata and hashed, single-use, expiring
  verification/reset challenges with attempt budgets.
- MFA factors, encrypted TOTP secrets, consumed time windows, hashed recovery
  codes, WebAuthn credentials/counters, and audited recovery state.
- Organizations and memberships with organization-scoped roles/permissions,
  plus organization keys in every tenant-owned table and SQL predicate.
- Refresh issuance `auth_version`, session/token cleanup state, and retention.
- Append-only audit records with bounded event schema and redacted trusted
  source fingerprints.

## Migration behavior

The source migration runner orders filenames lexically, opens a transaction per
file, executes the file, and records the filename. It is forward-only. It has no
down migration or checksum validation. Its session-level PostgreSQL advisory
lock is acquired through a connection pool, so the unlock is not proven to run
on the same connection; AviaSurveil360 should use a dedicated connection or a
transaction advisory lock.

The export has one migration, so its exact order is unambiguous. It was not run
against PostgreSQL during this task and remains **blocked** from database-level
verification.
