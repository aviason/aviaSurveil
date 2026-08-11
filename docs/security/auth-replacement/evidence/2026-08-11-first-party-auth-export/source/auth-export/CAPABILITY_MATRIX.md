# Capability matrix

Every row uses one required literal status. “Verified locally” means a focused
local test exercised that exact boundary; source inspection alone is not
verification.

| Capability | Status | Evidence and boundary |
|---|---|---|
| Native user registration | implemented but not verified | Public GraphQL registration calls `Store.Register`, creates `app.users` plus `app.auth_accounts`, and assigns `applicant`. No database integration test was available. |
| Administrative provisioning | absent | No invite, activate, direct administrative create, or completed role-grant executor exists. Seed tooling is development-only and unsafe for provisioning. |
| Unique immutable subject identity | partial | New users receive random 256-bit `usr_` subjects with a unique index and JWT `sub`; no database immutability guard exists, and source backfill subjects were deterministic MD5 identifiers. |
| Username normalization | partial | Input is trimmed and uniqueness/login are case-insensitive. Unicode normalization, reserved-name policy, cross-field uniqueness, and canonical identifier storage are absent. |
| Email normalization | partial | Trim plus case-insensitive index only. No syntax/canonicalization policy. |
| Email verification | absent | No verification state, challenge, delivery, or activation flow. |
| Password hashing algorithm and parameters | implemented and verified locally | Argon2id v19, 64 MiB memory, three iterations, parallelism 1, 16-byte random salt, 32-byte output; focused unit and race tests passed. |
| Password hash parsing safety | partial | Format checks exist, but stored memory/iteration/key parameters have no upper bounds before allocation/work. |
| Password rehash policy | absent | No `NeedsRehash` or successful-login parameter upgrade. |
| Password strength and size policy | absent | Only nonempty input is required; no minimum, maximum, compromised-password, or reuse check. |
| Password history | absent | No history storage or comparison. |
| Password reset | absent | No reset endpoint, repository, mail flow, or token table. |
| Hashed single-use expiring reset token | absent | No reset token implementation exists. |
| Password change | implemented but not verified | Requires current password plus recent password step-up, hashes the new value, increments `auth_version`, and revokes other sessions. No database test exists. |
| Revoke all sessions on password change | partial | Other sessions and refreshes are revoked, but the current pre-change refresh family remains usable. |
| Login enumeration-resistant response text | partial | Unknown account and wrong password map to the same client error, but unknown accounts skip Argon2 work and remain timing-distinguishable. |
| Login throttling and brute-force protection | absent | No per-source/principal/global limit, backoff, lockout, or password-work concurrency bound on active GraphQL auth paths. |
| Server-side session state | implemented but not verified | JWT bearer requests with `sub`/`sid` are checked against PostgreSQL session/account state on every authenticated request. No PostgreSQL test was run. |
| Access token handling | implemented and verified locally | Proprietary RS256 JWT issuance and verification unit tests cover signature, algorithm rejection, claims, and JWKS. It is not OAuth/OIDC token issuance. |
| Refresh token handling | implemented but not verified | Random 32-byte opaque tokens, SHA-256 hashes at rest, row locking, rotation, family reuse revocation, device/client string binding. No database integration test. |
| Session rotation and fixation prevention | implemented but not verified | Login creates a fresh random session/family/token; refresh rotates atomically. Strict concurrent reuse can revoke a legitimate family. |
| Session idle timeout | implemented but not verified | Default 14 days for member and 12 hours for all non-member roles; idle extends on refresh/step-up, not access-token use. The `applicant` classification as staff-short-idle is likely unintended. |
| Session absolute timeout | implemented but not verified | Default 30 days in session and refresh records. |
| Session revocation | implemented but not verified | Logout, logout-all, selected revoke, password-change revocation, and refresh-family reuse response exist. |
| Secure HttpOnly/Secure/SameSite cookies | absent | No auth cookies are set; bearer and refresh values are returned in JSON. Browser storage and BFF design remain AviaSurveil360 responsibilities. |
| CSRF protection | absent | No cookie-based auth and no CSRF token/origin check. Bearer-only use is not classic cookie CSRF, but adding cookies requires a new defense. |
| CORS policy | absent | No CORS middleware or permissive headers were found; cross-origin browser behavior must be defined at Caddy/application level. |
| OIDC relying-party state/nonce/PKCE | absent | No external authorization-code flow exists. |
| Redirect URI validation | absent | No redirect-based login or redirect registry exists. |
| OIDC discovery provider | partial | Discovery-shaped JSON and local JWKS exist, but advertised flows/scopes are unsupported and required endpoints are missing. |
| Remote discovery/JWKS consumption | absent | No external issuer or Keycloak JWKS validation path. |
| Signing algorithm allowlist | implemented and verified locally | Verifier requires exact `RS256` and a known `kid`; algorithm-confusion tests passed. |
| Signing-key storage | partial | PEM or filesystem path is supported. No KMS/HSM/secret-manager integration or permission check is present. |
| Signing-key rotation | partial | One active private key plus restart-loaded previous public keys. No automatic/live rotation or expiry policy. A prior custom `kid` is not preserved automatically. |
| Signing-key strength validation | absent | RSA key parsing does not enforce a minimum modulus size. |
| TOTP enrollment and verification | absent | No implementation. Password step-up is not MFA. |
| TOTP replay prevention | absent | No implementation. |
| Recovery codes | absent | No generation, hashing, consumption, or regeneration. |
| MFA reset/admin recovery | absent | No factor recovery workflow or administrative controls. |
| WebAuthn/passkeys | absent | No implementation. |
| Display role | implemented but not verified | Role strings are duplicated in `app.users` and `auth_accounts`; the latter is preferred. |
| Permission assignment/revocation | partial | `staff_permissions` is read live, but reviewed admin mutations create proposals only; approval does not apply the role change. |
| Request-time permission revocation | implemented but not verified | Staff checks query active, non-revoked permission rows rather than trusting stale JWT role claims. |
| Organization membership | absent | No organization or tenant entity, membership, claim, or resolver exists. |
| Organization isolation | absent | No organization key is enforced in auth, session, role, or application queries. |
| Disabled user behavior | absent | No disabled state or fail-closed check. |
| Suspended user behavior | absent | No suspended state or fail-closed check. |
| Locked user behavior | absent | No lock state or login-attempt state. |
| Deleted user behavior | partial | Cleanup can revoke credentials/delete auth accounts, but deletion is an operator-run worker; `deletion_pending` is not checked by auth paths. |
| Login/logout/session/password audit | partial | Several success events and refresh-reuse events exist, but failures, registration, normal refresh, identifier changes, and recovery are missing. Event insert errors are ignored at call sites. |
| Administrative-change audit | partial | A separate staff audit exists for proposals, but assignment execution is absent and generic approval authorization is incomplete. |
| Redacted logs and telemetry | partial | No raw token/password logging was found in active auth code. Audit IP/user-agent hash fields remain empty, and a seed command logs a static demo password. |
| SMTP/email delivery | absent | No SMTP client, TLS/STARTTLS policy, templates, or mail queue. |
| English identity messages | partial | Hard-coded English API error codes/messages exist; no user-facing catalog or delivery templates. |
| Turkish identity messages | absent | No localization catalog/template. |
| French identity messages | absent | No localization catalog/template. |
| Portuguese identity messages | absent | No localization catalog/template. |
| Liveness | implemented but not verified | `/healthz` responds without checking auth dependencies. |
| Readiness | partial | `/readyz` checks only database ping; token-service construction errors are ignored, so signing can be unavailable while readiness succeeds. |
| Forward-only migrations | partial | Source runner applies lexically ordered SQL in transactions and has no down migrations. Export projection order is explicit, but neither was run against PostgreSQL. |
| Migration rollback | absent | Deliberately forward-only; no down/rollback behavior. |
| Migration checksums/immutability | absent | Source runner records filenames only and has no content checksums. |
| Auth unit tests | implemented and verified locally | Focused auth and password/JWT tests passed under Go 1.26.4. |
| Auth store integration tests | absent | No focused PostgreSQL session/refresh/store suite found. |
| Migration tests | absent | No migration execution test suite found. |
| Auth race tests | implemented and verified locally | `go test -race` passed for `internal/auth`; database concurrency paths were not exercised. |
| Auth fuzz tests | absent | No `func Fuzz` target exists. |
| Security regression tests | partial | JWT algorithm and auth middleware cases exist; throttling, reuse concurrency, tenant, account-state, recovery, MFA, and audit failure tests are absent. |
