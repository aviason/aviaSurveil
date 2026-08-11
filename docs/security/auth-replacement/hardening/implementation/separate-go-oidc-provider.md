# Implementation Plan: Separate First-Party Go OIDC Provider

## Selected Design And Constraints

The selected design is a small, separately privileged Go OIDC provider. It
will reuse only reviewed concepts and tests from the two owner-provided exports
and will use a maintained authorization-server library for protocol mechanics.
The provider owns credentials, factors, provider sessions, OIDC state, signing keys, reset
and verification challenges, and identity security audit. AviaSurveil360 keeps
its existing application sessions, desired memberships, organization scope,
and domain authorization.

Keycloak remains the active provider until the replacement passes every gate.
No implementation phase authorizes AWS, Cloudflare, SMTP, production identity,
DNS, secret, migration, deployment, or traffic mutation.

## Task 1 Library Authorization Gate

The Task 1 comparison and disposable evidence are recorded in the
[OIDC library spike results](../../evidence/2026-08-11-oidc-library-spike/RESULTS.md)
and the [provider-neutral contracts](../../evidence/2026-08-11-oidc-library-spike/CONTRACTS.md).
As of 2026-08-11 the evidence is `verified locally`. The owner authorized
**Option A — `zitadel/oidc` v3.47.5 (`fb9fbfe`)** on 2026-08-11 with the
messages “tamamdır olur kabul” and “tamam evet”, accepting the previously
documented recommendation. This is the selected-library field for the
implementation. Task 2 may create `apps/auth/`, but only with this library.
No production, deployment, identity, secret, migration, traffic, or
Keycloak-retirement authority is implied.

The following exact authorization question was answered on 2026-08-11:

> Which option authorizes Task 2: (A) zitadel/oidc; (B) ory/fosite;
> (C) authelia.com/provider/oauth2; (D) Ory Hydra; (E) another
> evidence-backed option; (F) low-level JOSE/OAuth2 after explicit
> contract amendment; (G) libraryless/manual after explicit
> contract amendment and independent security review; or (H) stop/rework?

**Recorded answer:** A — `zitadel/oidc` v3.47.5 (`fb9fbfe`), authorized
2026-08-11.

Only the selected option proceeds. The comparison and disposable evidence for
the other options remain retained for audit and are not implementation
authorization.

## Source Revision And Drift Check

- Evidence archive SHA-256:
  `7fa982300440cb3e79d28bc0f7f22ebb59124bc9c125dededb22dea306fc7fb7`
- Export source revision:
  `60dbe494318106569f6d9dbea121d6b1c841ae95`
- Second evidence archive SHA-256:
  `5de123c9bd8a711889e85b1876329540dee49423f64568c0a4bade4b5a4ff79b`
- Second export source revision:
  `cfcf14a6de6a5e7c00ff116dd47e477dddc68c74`
- Combined evidence collection SHA-256:
  `06e71b74208ee3e703ecd096a3ba126b6dd3b4e1509850e8a06be38098a8b175`
- AviaSurveil360 HEAD at intake:
  `e36a395492f82b2bd4c79d2b580449fddbedf07e`
- Working-tree drift at intake: `present`

Before each implementation package, refresh the relevant current files and
record material drift in the ExecPlan. Never overwrite unrelated user changes.

## Affected Components

- New `apps/auth/` Go service and tests.
- New AviaSurveil360-owned forward-only identity migrations; the export
  migration is evaluation evidence only.
- Generic provider administration and observation interfaces in
  `apps/api/internal/identity/`.
- Existing OIDC/BFF/session code in `apps/api/internal/identity/`,
  `apps/api/internal/platform/session/`, and `apps/api/internal/httpapi/`.
- Identity UI and recovery flows under `apps/web/src/auth/`.
- Local Compose, gateway, secrets, SMTP test harness, runbooks, and eventually
  the separately authorized production surface.

## Ordered Work Packages

1. Freeze identity invariants and select a maintained OIDC library through a
   bounded interoperability, maintenance, license, and dependency spike.
2. Scaffold `apps/auth` with strict configuration, liveness/readiness,
   non-root image, separate database role, redacted telemetry, and no runtime
   integration.
3. Implement canonical identifiers, immutable subjects, account states,
   administrative provisioning, password policy/rehash/history, constant-cost
   failure, consistent verification gates, layered throttling across every
   public auth transport, trusted-proxy handling, and bounded Argon2id
   concurrency and input size.
4. Implement provider sessions and refresh families with issuance auth version,
   rotation, reuse detection, absolute/idle expiry, all-session revocation,
   conditional or row-lock concurrency, stale-write denial, cleanup, crash
   recovery, and PostgreSQL contention tests. Password change must atomically
   revoke all prior provider and application sessions.
5. Implement standards-conforming OIDC discovery, authorization code plus PKCE,
   exact clients/redirects, ID/access/refresh tokens, JWKS/key rotation,
   revocation, userinfo when required, and logout through the selected library.
   Logout must bind the authenticated subject and exact session; no identifier
   embedded in caller-controlled refresh text is authoritative. Remove any
   legacy HMAC configuration path.
6. Implement email verification, hashed single-use reset challenges, TOTP with
   replay denial, hashed recovery codes, administrative recovery, verified
   TLS/STARTTLS delivery, and `en`/`tr`/`fr`/`pt` catalogs.
7. Implement append-only redacted security audit and a narrow internal provider
   administration/observation API for enable, lock, required actions, factors,
   sessions, and subject state.
8. Replace Keycloak-specific API callers behind a provider-neutral interface;
   preserve exact application membership, role, organization, CSRF, BFF, and
   session behavior. Serialize concurrent client refresh work, clear the full
   local session after terminal rejection, and revalidate or disconnect any
   authenticated long-lived channel on expiry or revocation. Never enable
   dual-issuer fallback.
9. Add an isolated local replacement profile and synthetic identities with a
   distinct issuer. Prove login, logout, MFA, recovery, lifecycle, tenant
   denial, browser, restart, backup/restore, and rollback while Keycloak remains
   intact.
10. Run current dependency/security scans, independent review, protocol
    interoperability, fuzz/race/PostgreSQL integration, failure recovery, and
    native ARM64 mixed-capacity gates.
11. Only after explicit cutover authorization, migrate exact subjects through
    a reviewed identity manifest, require safe credential/factor enrollment,
    switch the exact issuer and gateway route, observe acceptance, and retain a
    bounded rollback window.
12. Remove Keycloak code, image, database, secrets, runbooks, and release
    subjects only after cutover acceptance and separate retention/destruction
    authorization.

## Compatibility And Migration

The current API remains an OIDC relying party. The replacement must satisfy its
Authorization Code plus PKCE, nonce, exact issuer, ID-token, logout, and claim
contracts or revise them deliberately with equivalent tests. Provider
administration becomes an application-owned interface rather than a Keycloak
API abstraction.

Subjects are never derived from mutable email or username. Synthetic identities
may be recreated with exact approved subjects. Real identity, password, TOTP,
or recovery migration is outside local implementation and requires an exact
inventory, owner decision, and separate authorization.

## Tactical Protections During Migration

- Keep Keycloak patched, configured, and recoverable.
- Use different issuer URLs, client IDs, keys, databases, and cookies.
- Configure each test environment for exactly one provider.
- Do not accept token verification fallback or share signing keys.
- Preserve current provider observation, membership revision, session
  revocation, CSRF, and offline subject-lock boundaries.
- Keep the imported candidate out of production builds until adapted code
  passes the mapped finding tests.

## Tests And Security Validation

- Regression coverage for `AS360-AUTH-001` through `AS360-AUTH-030` from the
  two imported security reviews and the
  [source comparison](../source-comparison.md).
- PostgreSQL execution and concurrency tests for every migration, login,
  refresh rotation/reuse, password change, lock/suspend/deactivate, factor
  consumption, audit failure, and cleanup path.
- OIDC positive and negative tests for discovery, exact redirects, state,
  nonce, PKCE, client authentication, algorithms, claims, time, JWKS
  activation/overlap/retirement, revocation, subject-bound logout, and stale
  HMAC configuration denial.
- Cross-transport rate-limit, verification-gate, WebSocket/long-lived-session,
  client refresh serialization, stale-write, body/password bound, and security
  log-redaction tests.
- Organization/role negative matrix and direct repository-scope tests.
- Unit, race, fuzz, integration, browser, accessibility, dependency, license,
  secret, image, and independent security review gates.

## Performance And Resource Benchmarks

Measure the existing Keycloak baseline and replacement on native ARM64 using
the complete gateway/API/worker/identity mixed workload. Record steady and peak
RSS, CPU, login and refresh latency, Argon2id queue depth/rejections, database
pool usage, restart time, and required headroom. Do not reduce Argon2id strength,
disable MFA, or hide a capacity failure with sustained swap.

## Rollout And Rollback

Local and preproduction rollout uses a distinct issuer and synthetic data.
Production cutover, if later authorized, changes one exact issuer/route wave and
does not accept old and new tokens simultaneously. Rollback invalidates new
sessions, restores the exact Keycloak issuer and immutable image, confirms
subject/application membership continuity, and records a security audit event.

## Acceptance Criteria

- Every imported High and Medium finding has a passing regression or an
  explicitly accepted residual risk; no finding is closed by proposal alone.
- OIDC interoperability, MFA/recovery, lifecycle, organization isolation,
  session revocation, audit, SMTP, backup/restore, and rollback gates pass.
- Current dependency and image scans contain no unaccepted reachable
  High/Critical result.
- Native ARM64 mixed workload meets the private-pilot headroom contract.
- Independent security review has no unresolved cutover blocker.
- Keycloak removal has separate explicit authorization and occurs only after
  successful cutover acceptance.

Until then the result remains `candidate-only`, release is `release pending`,
and `production-ready: not established`.

## Open Decisions

- The maintained authorization-server library is no longer open: Option A,
  `zitadel/oidc` v3.47.5 (`fb9fbfe`), was authorized on 2026-08-11. Any
  change requires a new evidence comparison and owner decision.
- Initial WebAuthn/passkey requirement.
- Provider database name and long-term backup/retention policy.
- Whether any non-synthetic identity migration is needed.
