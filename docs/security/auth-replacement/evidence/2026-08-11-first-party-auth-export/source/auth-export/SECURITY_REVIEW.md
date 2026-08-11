# Security review

This is a source-only review of the EMSI Go authentication and identity
boundary. It is not a production assessment, penetration test, or statement
about customer data. No external system or production data was accessed.

## Threat model

Protected assets are account subjects, password hashes, signing keys, access
tokens, refresh tokens, server-side sessions, permissions, identity attributes,
and security audit records. Trust boundaries are the unauthenticated GraphQL
registration/login/refresh operations, bearer middleware, JWT-to-database
session validation, PostgreSQL transactions, staff permission checks, and
deployment-provided keys/configuration. Relevant attackers are an
unauthenticated remote caller, a self-registered applicant, an ordinary
authenticated member, a scoped staff user, a holder of a stolen access or
refresh token, and a misconfigured operator.

The principal security objectives are correct authentication, immediate
credential revocation, bounded password work, fail-closed account status,
least-privilege authorization, tenant isolation, durable redacted auditing, and
signing-key confidentiality.

## Validated findings

### AS360-AUTH-001 — High — Pending applicants can cross the documented product-access gate

`Store.Register` assigns every public registration the `applicant` role, but the
source pending-user middleware does not classify `/graphql` as denied. Many
product resolvers require only an authenticated user ID. A self-registered,
unapproved account can therefore reach member GraphQL surfaces. This is a
source-application authorization defect, not something the isolated package
can correct.

Evidence: source `internal/auth/store.go:144-166`,
`internal/httpapi/server.go:197-254`, and product resolvers. AviaSurveil360 must
enforce active/provisioned status centrally for every protected operation and
keep a small explicit pre-approval allowlist.

### AS360-AUTH-002 — High — Public registration exposes unbounded Argon2id work

Public registration hashes the submitted password before database uniqueness
checks. There is no application rate limiter, global password-work semaphore,
or bounded GraphQL body admission. Each call can consume approximately 64 MiB
for three Argon2id iterations, including duplicate registrations that later
fail. Parallel requests can exhaust the single ARM64 runtime.

Evidence: `auth/password.go:25-41`, `auth/store.go:144-166`, and source GraphQL
resolver/middleware. Preserve the memory-hard hash and bound admission at both
the application and trusted Cloudflare/Caddy boundary.

### AS360-AUTH-003 — High — Source seed command contains a known privileged password

The source repository's runnable `cmd/seed-m9` utility embeds and logs a static
demo password, accepts any database URL, and can provision privileged accounts
without a fail-closed disposable-environment check. The seed utility and value
are excluded from this export, but their presence makes the source default
unsafe if an operator runs that command against a non-disposable database.

Remove the static credential, generate one-time secrets, require explicit
database/environment markers, and never ship the seed binary in production.

### AS360-AUTH-004 — High — Empty requested staff scope can authorize scoped grants globally

The source staff permission predicate treats an empty requested scope as a
wildcard. Multiple admin resolvers pass an empty scope, and generic proposal
review does not authorize against proposal type, target scope, or proposer.
Scoped staff may therefore cross delegated resource boundaries or approve
unrelated proposals. The staff workflow is not exported as reusable code.

AviaSurveil360 must model organization and resource scope explicitly. An empty
requested scope may match only an explicit global grant, never a scoped grant.

### AS360-AUTH-005 — Medium — Login guessing is unbounded and identifiers are timing-enumerable

Unknown identifiers return before Argon2id verification, while known accounts
perform the memory-hard password check and staff-permission hydration. Response
text is generic, but timing differs materially. No per-source, per-identifier,
or global attempt budget, progressive delay, or abuse audit exists.

Evidence: `auth/store.go:209-231`, `auth/store.go:827-852`, and
`auth/password.go:52-58`. Use a fixed dummy hash for unknown users and layered
rate limits without creating an account-lockout denial-of-service primitive.

### AS360-AUTH-006 — Medium — Password policy permits trivial and reused passwords

Registration and password change require only a nonempty value. There is no
minimum or safe maximum length, compromised-password control, current-password
reuse rejection, history, or encoded-parameter rehash policy. A one-character
password is accepted.

Evidence: `auth/store.go:144-166`, `auth/store.go:625-647`, and
`auth/password.go`. Centralize the policy across registration, change, and any
future reset flow. Add rehash-on-success when stored parameters are outdated.

### AS360-AUTH-007 — Medium — Password change leaves the current old refresh family usable

Password change increments `auth_version` and revokes every *other* session and
refresh token. The current refresh record has no issuance auth version. It can
rotate after the password change and receive an access token carrying the new
account auth version. A thief holding that refresh value plus the repeatable
device/client strings is not immediately evicted.

Evidence: `auth/store.go:625-684` and `auth/store.go:234-352`. Revoke every old
family; if the current device must remain signed in, replace its session and
family atomically and return the new credentials.

### AS360-AUTH-008 — Medium — Account lifecycle does not fail closed

There is no disabled, suspended, or locked state. The only lifecycle column is
`active`, `deletion_pending`, or `deleted_tombstone`, and login, refresh, and
session validation do not check it. Immediate deletion only queues a cleanup;
the credential-revoking cleanup worker is operator-run and partial failures are
not automatically retried.

Evidence: `migrations/001_auth_identity.sql`, `auth/store.go:234-315`,
`auth/store.go:412-452`, and `auth/store.go:827-852`. Add one fail-closed account
state used by all three paths and make cleanup durable and scheduled.

### AS360-AUTH-009 — Medium — Registration accepts unverified, ambiguous login identifiers

Registration immediately reserves caller-provided email and enables it for
login without ownership verification. Username and email are unique only
within their respective columns, but login searches both with an `OR`; one
account's username can equal another account's email and make lookup ambiguous.
Normalization is trim plus case-insensitive database comparison only.

Evidence: `auth/store.go:144-206`, `auth/store.go:827-852`, and the separate
indexes in `migrations/001_auth_identity.sql`. Use a single canonical,
type-aware identifier table and enable email login only after verification.

### AS360-AUTH-010 — Medium — Username can change without recent step-up

In the active source GraphQL profile mutation, email and phone changes require
recent password step-up but username changes do not. The store then updates the
same username used for login and emits no identity security event. A stolen
bearer session can rename the victim's login identifier.

Evidence: `auth/store.go:723-768` plus source GraphQL resolver
`schema.resolvers.go:258-299`. Require step-up for every login-identifier change,
notify through a verified channel, and audit a redacted change event.

### AS360-AUTH-011 — Medium — Security audit coverage and failure semantics are incomplete

Durable events cover successful login, logout, logout-all, selected revoke,
step-up, password change, and detected refresh reuse. Registration, failed
login, failed step-up, normal refresh, identifier changes, and recovery events
are absent. Call sites intentionally ignore several audit insert errors. The
schema's IP and user-agent hash fields are never populated.

Evidence: `auth/store.go:939-942` and `auth/store.go:1151-1167`. Introduce a
shared `AuditSink`, bounded reason codes, trusted-proxy-aware keyed hashes, and
explicit fail-closed/fail-alert behavior for security-critical events. Never
record credentials, tokens, raw bodies, or unbounded attacker strings.

### AS360-AUTH-012 — Medium — Disallowed client IDs create durable orphan sessions

Login verifies the password and commits the session, refresh record, and login
event before access-token issuance checks the client allowlist. A valid account
can repeatedly use a disallowed client ID: issuance fails, but the unreachable
database rows remain. No session-count cap or expired-row cleanup was found.

Evidence: `auth/store.go:889-959` and `auth/tokens.go:166-174`. Validate issuance
policy before committing or make issuance and persistence one fail-closed
operation; add caps and cleanup.

### AS360-AUTH-013 — Medium — Configured step-up TTL is not the enforced TTL

The store uses configured `StepUpTTL` to report `validUntil`, but GraphQL and
other authorization checks call `UserHasRecentStepUp` with zero, which defaults
to a hard-coded ten minutes. A configured shorter window is therefore reported
expired while still authorized.

Evidence: `auth/auth.go:56-72`, `auth/store.go:583-622`, and source GraphQL
session/password resolvers. Persist or inject a single server-derived expiry
boundary everywhere.

### AS360-AUTH-014 — Medium — OIDC metadata overstates protocol support

Discovery-shaped metadata advertises `openid`, `offline_access`, `RS256` ID
tokens, and `response_types_supported: [token]`. The service implements no
authorization endpoint, OAuth token endpoint, ID token, userinfo endpoint,
client authentication, consent, state, nonce, PKCE, or redirect validation.
Consumers can misclassify the service as an OIDC provider.

Evidence: `auth/tokens.go:276-297`. Do not publish OIDC discovery unless a
conforming server is implemented. For the native design, publish a separate
application metadata document.

### AS360-AUTH-015 — Medium — Signing and readiness configuration is incomplete

Signing keys may be supplied inline or by file, with no minimum RSA modulus
check, KMS/HSM integration, permission validation, automatic rotation, or live
reload. Previous keys receive a derived fingerprint `kid`, so a previous custom
`kid` is not preserved automatically. The source server discards token-service
construction errors and readiness checks only PostgreSQL. `AUTH_JWT_SECRET` is
loaded but unused. Development mode generates a new ephemeral 2048-bit key at
each startup.

Require explicit production issuer/audience/client/key configuration, reject
weak RSA keys, preserve prior key IDs, surface initialization errors, and make
readiness depend on usable signing/verifying state.

### AS360-AUTH-016 — Medium — Auth and database dependencies have known advisories

The offline source scan reported critical advisories for source dependency
`github.com/jackc/pgx/v5 v5.7.2`, fixed in 5.9.x, and several advisories against
`golang.org/x/crypto v0.51.0`, fixed in 0.52.0. The copied auth package imports
only `x/crypto/argon2`; the flagged SSH/OpenPGP paths appear unreachable from
this package, but that is an import-path inference rather than a call-graph
proof. The export retains source versions for faithful evaluation.

Upgrade and retest in the AviaSurveil360 repository rather than silently
changing the candidate export.

### AS360-AUTH-017 — Medium — No organization isolation exists

There is no organization/tenant table, membership, claim, repository filter,
or authorization resolver. Free-form staff permission `scope` is not a tenant
boundary and, in source, has a wildcard defect. This is a blocking absence for
an application-managed multi-organization system, not a demonstrated
cross-tenant exploit against the current single-context application.

### AS360-AUTH-018 — Low — Authenticated users can enumerate arbitrary role assignments

Source `AdminRoles` and `AdminUserRoles` resolvers require authentication but no
staff permission, and the latter accepts an arbitrary user ID. Normal profiles
omit these role fields. Any authenticated account can therefore enumerate role
labels for other users.

Require `permission_manager` or `super_admin` for cross-user role reads and use
a separate self-only endpoint if needed.

## Explicit negative results and absent controls

- No permissive CORS header or middleware was found. This is **absent**, not a
  verified safe deployment policy; Caddy/Cloudflare configuration was out of
  scope.
- No open redirect, redirect URI, outbound OIDC request, remote JWKS fetch,
  SQL injection, command injection, or direct token logging was validated in
  the reviewed auth boundary.
- Tokens are transported in GraphQL JSON and `Authorization: Bearer`; no auth
  cookies exist. Therefore cookie flags and classic cookie CSRF are **absent**,
  not implemented. A browser BFF/cookie design needs new controls.
- The optional `X-User-ID` development bypass is disabled by default and source
  config validation rejects it outside local/development/test. An
  internet-reachable deployment must still fail if it uses a development
  environment label with the flag enabled.
- Password reset, email verification, MFA, recovery codes, administrative
  recovery, and WebAuthn are absent; there is no insecure recovery endpoint to
  validate, but those missing controls block replacement.
- No plaintext credential, private signing key, SMTP credential, API token,
  production URL, or customer datum is included in this export.

## Required security gate before reuse

Do not integrate this candidate until AviaSurveil360 supplies and tests: a
standards decision for OIDC versus native BFF auth; organization-scoped data
access; verified provisioning/deactivation; throttled constant-cost password
flows; complete password/reset/MFA/passkey lifecycle; all-session credential
rotation; durable redacted auditing; safe SMTP; four-language identity content;
strict production configuration/readiness; dependency upgrades; and
PostgreSQL-backed concurrency/migration tests.
