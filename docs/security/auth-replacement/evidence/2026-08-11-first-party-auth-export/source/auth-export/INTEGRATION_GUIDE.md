# AviaSurveil360 integration guide

## Decision first

This export cannot satisfy existing Keycloak/OIDC contracts. AviaSurveil360
must choose one of two architectures before integration:

1. Retain Keycloak (or another conforming provider) for OIDC, MFA, recovery,
   federation, and service clients, and use only selected native primitives from
   this export; or
2. Deliberately replace OIDC with a first-party native/BFF design, rewrite the
   missing lifecycle/security controls, and migrate every issuer/client
   expectation. This is not a drop-in change.

Do not expose the current discovery-shaped endpoint as proof of OIDC support.

## Keycloak responsibility map

| Responsibility | Candidate path | Integration classification |
|---|---|---|
| Browser login | Reuse Argon2id verification and store concepts, but place the React client behind a same-origin chi BFF. Do not persist refresh tokens in browser JavaScript. | **Must be rewritten** at the browser/transport layer. |
| Browser logout | `Store.Logout`, `LogoutAll`, and selected revocation provide database operations. | **Needs an AviaSurveil360 adapter** and cookie/BFF semantics. |
| Current-session lookup | Bearer middleware plus `ValidateSession`; `ListSessions` returns device/session metadata. | **Needs an adapter** for chi context and Avia response models. |
| Immutable subject | Random `usr_` subject and unique index. | **Can be reused with a database immutability constraint**; migration/backfill needs rewrite. |
| TOTP MFA | None. Password step-up is not MFA. | **Must be implemented or retained in Keycloak**. |
| User provisioning | Public self-registration only; no complete administrative provisioning. | **Must be rewritten** using invitations/admin approval and verified identifiers. |
| User deactivation | No disabled/suspended/locked enforcement. | **Must be rewritten** and checked in login, refresh, and every session validation. |
| Role membership | Display role plus live `staff_permissions` reads. Assignment workflow is incomplete. | **Must be rewritten** as normalized application roles/permissions. |
| Organization membership | None. | **Must be implemented** and applied to every repository query/mutation. |
| Session revocation | Server-side session rows, auth version, refresh-family revocation. | **Reusable after fixing password-change/current-family and account-state gaps**. |
| Password reset | None. | **Must be implemented** with hashed, single-use, expiring tokens and safe SMTP. |
| Audit logging | Partial `auth_security_events`. | **Needs a durable `AuditSink` rewrite** with failure events and redacted source fingerprints. |
| API authentication middleware | `auth.Wrap` verifies bearer JWTs and validates server-side session state. | **Can be adapted** to chi middleware; remove the development header from production builds/config. |
| OIDC issuer/discovery/JWKS | Local JWKS and misleading discovery-shaped JSON only. | **Cannot replace Keycloak**. Keep a conforming provider or implement a standards-tested server/library. |
| Keycloak service-client administration | No client registry, client credentials, admin API, realm roles, or service accounts. | **Cannot be replaced by this export**; rewrite callers against application admin APIs or retain an IdP. |

## Code classification

### Reuse with no semantic change

- `auth/password.go` as the starting Argon2id encoder/verifier, after adding a
  separate policy and rehash decision around it.
- JWT RS256 primitives and exact-algorithm verification in `auth/tokens.go` for
  an explicitly proprietary access-token contract.
- Random 256-bit subject/session/refresh generation, refresh-token hashing,
  row-lock rotation, and reuse-family revocation concepts.
- Context user helpers where they match AviaSurveil360's request model.

“Reuse” does not mean production approval: dependency upgrades, RSA-size
validation, tests, and the reported control gaps remain required.

### Needs an AviaSurveil360 adapter

- `database/sql` calls can use pgx's stdlib adapter, but AviaSurveil360 should
  expose task-specific pgx repositories behind `UserRepository` and
  `SessionRepository` rather than coupling the modular monolith to EMSI tables.
- `examples/chi_mount.go` shows the subset of chi used for metadata/JWKS.
  Login/logout/current-session handlers need Avia request/response types.
- `AuditSink`, `MailSender`, `Clock`, and `RandomSource` should be injected at
  the application boundary for deterministic tests and operational policy.
- Existing role strings need translation into AviaSurveil360 application roles;
  never infer an organization from a global role.

### Must be rewritten

- Public registration, verified email/username identifier lifecycle, admin
  provisioning, deactivation, suspension/lock, and deletion enforcement.
- Browser transport. Prefer a same-origin BFF with `Secure`, `HttpOnly`, and an
  intentional `SameSite` setting, session rotation, Origin checks, and CSRF
  tokens for state-changing cookie requests. If bearer tokens remain in the
  React client, document and mitigate XSS/token-theft risk explicitly.
- Password policy, rehash, reset, notification, and recovery.
- TOTP enrollment/challenge/replay control, hashed recovery codes, recovery
  administration, and optionally WebAuthn/passkeys.
- Organization membership and repository-level tenant scoping.
- Role assignment/revocation and privileged workflow approval.
- OIDC/OAuth protocol behavior, if OIDC compatibility remains a requirement.
- Signing key custody, rotation, startup validation, and readiness.
- Locale selection and identity catalogs/templates for `en`, `tr`, `fr`, and
  `pt` (choose and document `pt-BR` versus `pt-PT` if product requirements care).

### Keycloak capabilities this export cannot replace

- Authorization Code flow, PKCE, state, nonce, redirect URI registration,
  consent/scopes, ID tokens, userinfo, token/introspection/revocation endpoints,
  dynamic or administrative client management, and client credentials.
- Federation/brokering, upstream social/enterprise identity, realm management,
  service accounts, and standards-compatible logout/session management.
- TOTP/recovery/WebAuthn, verified-email recovery, localized hosted identity UI,
  and mature administrative identity lifecycle.

## Proposed ports

The compileable declarations live in `auth/ports.go`. They are candidate-only;
the copied `Store` does not implement them.

- `UserRepository`: create/load by canonical identifier or subject, enforce
  account state, and atomically advance auth version.
- `SessionRepository`: create/validate/rotate/revoke sessions while enforcing
  subject, client, organization, account state, auth version, idle, and absolute
  expiry.
- `PasswordHasher`: hash, verify, and report `NeedsRehash`; policy remains a
  separate service.
- `MFARepository`: store verified factors, atomically consume a TOTP time window
  to prevent replay, consume hashed recovery codes, and audit resets.
- `AuditSink`: durable, redacted, bounded security events. Decide explicitly
  which events fail closed and which alert without blocking availability.
- `MailSender`: template/locale-based mail over externally validated TLS or
  STARTTLS; it must never accept raw HTML or secret-bearing log fields from
  handlers.
- `Clock`: authoritative UTC time for token/session/reset/MFA tests.
- `RandomSource`: cryptographically secure entropy in production and
  deterministic controlled entropy only in tests.
- `OrganizationMembershipResolver`: resolve active subject membership, roles,
  and permissions for exactly one organization.

## Suggested modular-monolith ownership

Keep packages narrow:

- `identity`: subjects, verified login identifiers, lifecycle status.
- `credentials`: password policy/hash and reset challenges.
- `sessions`: browser sessions, JWT access assertions, refresh rotation.
- `mfa`: TOTP/recovery codes/WebAuthn.
- `authorization`: roles, permissions, organization membership.
- `audit`: append-only redacted security events.
- `mail`: locale-aware template rendering and SMTP transport.
- `identityhttp`: chi handlers/middleware and React/BFF contract.

Each repository method must take the organization or deliberately operate on a
global identity. Avoid placing tenant selection solely in middleware context;
bind it into SQL predicates and unique constraints.

## PostgreSQL integration sequence

1. Create a new task-owned database and run `migrations/001_auth_identity.sql`
   only for evaluation.
2. Implement the ports with pgx transactions and add database integration tests
   for concurrent refresh, reuse, password change, deactivation, and audit
   failure.
3. Replace the compact projection with AviaSurveil360-owned forward-only
   migrations. Add canonical identifier, verification, reset, MFA,
   organization, membership, account-state, and audit tables.
4. Add constraints that make `subject` immutable, identifiers unambiguous, and
   tenant memberships uniquely scoped.
5. Add expired/revoked token cleanup and operational retention policies.

## Browser and edge boundary

Cloudflare Tunnel and Caddy are defense in depth, not substitutes for
application controls. Define the only trusted proxy hops, cap request bodies and
connection timeouts, rate-limit password operations at both layers, restrict
metrics/administration to the private plane, and make Caddy preserve one
canonical HTTPS origin. The application must validate trusted forwarded
addresses before producing audit fingerprints.

## Cutover compatibility

Existing Keycloak `sub` values must either remain immutable external identities
or be mapped once to new immutable subjects in an audited migration. Do not
derive new subjects from mutable email/username. During migration, never accept
both Keycloak and native tokens through an ambiguous fallback. Use an explicit
issuer/audience route boundary and remove the old verifier only after all
clients and service calls have moved.
