# Authentication and identity capability matrix

Status vocabulary is literal: `implemented and verified locally`, `implemented but not verified`, `partial`, `absent`, `blocked`, and `not applicable`. “Implemented” means source evidence exists; it does not imply protocol conformance or production readiness.

| Capability | Status | Evidence and boundary |
|---|---|---|
| Immutable user subject/UUID | implemented and verified locally | `internal/user/id.go`, `internal/user/model.go`; UUIDv7-style IDs are used as `sub`/user ID and full unit tests passed. |
| Canonical email/username identifier | implemented and verified locally | Email is normalized and uniqueness is transactionally guarded in `internal/auth/service.go` and `internal/platform/storage/dynamodb.go`; user/auth tests passed; no separate username field. |
| Registration | implemented and verified locally | Email/password registration, validation, verification token and consent recording in `internal/auth/service.go`; auth/full-suite tests passed. |
| Invitation flow | absent | No invitation model, token, endpoint, or test. |
| Administrative provisioning | absent | No admin auth/API or provisioning workflow in the boundary. |
| Password hashing and verification | implemented and verified locally | Argon2id with random salt and constant-time comparison in `pkg/security/password.go`; package and full-suite tests passed. |
| Password policy | implemented and verified locally | Minimum policy is enforced in auth service and covered by auth tests; no breached-password or history policy. |
| Password rehash/upgrade | absent | Stored Argon2 parameters are parsed, but no migration/rehash-on-login path was found. |
| Password history | absent | No history storage or reuse check. |
| Login throttling | implemented and verified locally | IP fixed-window limiter plus failed-login counters in user state; focused/full tests passed. |
| Enumeration resistance | partial | Login errors are generic in the normal path, but account/verification behavior and IP-only limiting need a dedicated review. |
| Account locking | implemented and verified locally | Failed-attempt counter and `LockedUntil` are persisted; auth and full-suite tests passed. |
| Access-token issuance | implemented and verified locally | Proprietary RS256 bearer JWT with issuer, audience, expiry, token version, email and device claims; JWT/auth tests passed. |
| Access-token verification | implemented and verified locally | Algorithm allow-list, signature, issuer/audience/expiry and database token-version/status checks; focused and race tests passed. |
| Refresh-token issuance | implemented and verified locally | Opaque `userID.secret` token; SHA-256 hash stored in the user record with idle/absolute expiry and device binding; auth tests passed. |
| Refresh entropy/transport | implemented and verified locally | Cryptographically random opaque secret over the JSON auth contract; auth/full tests passed, while HTTPS enforcement remains deployment-dependent. |
| Refresh hash/encryption at rest | implemented and verified locally | SHA-256 hash at rest; auth tests passed. Application-level encryption is not implemented and storage encryption is deployment responsibility. |
| Refresh family/session binding | partial | One refresh record per user and device binding exist; no explicit token-family table or multi-session family graph. |
| Client/device binding | implemented and verified locally | Device ID is checked and mismatch clears the stored session; auth handler tests passed. |
| Rotation atomicity | partial | Rotation overwrites one user row; no conditional compare-and-swap/family transaction evidence. |
| Concurrent refresh behavior | blocked | No explicit DynamoDB contention/conditional-write test; unit/full/race runs passed but do not prove concurrent persistence semantics. |
| Refresh reuse detection | partial | Source detects hash/device mismatch and clears the session; no explicit old-token race/reuse regression test was found. |
| Family-wide revocation | partial | Clearing the single stored session is effectively one-user-wide, but there is no token-family model. |
| Password-change revocation | absent | `ChangePassword` changes the hash but does not bump token version or close existing sessions; password reset does close sessions. |
| Account-state revocation | implemented and verified locally | Verify path checks active state and token version; deactivation/delete close sessions; full suite passed. |
| Absolute and idle refresh expiry | implemented and verified locally | Both are persisted and checked in refresh service; auth tests passed. |
| Refresh cleanup/retention | partial | User-row expiry fields exist; no dedicated cleanup/retention job for refresh state was found. |
| Refresh crash recovery | blocked | No crash/retry/concurrency test; persistence semantics are not verified locally. |
| Token leakage/logging controls | partial | Access tokens are returned to clients; dev mail/SMS adapters can log verification material. No comprehensive auth log-redaction test. |
| Server-side sessions/device sessions/limits | partial | A single refresh/device session is embedded in the user row; no multi-device session table or configurable session limit. |
| Logout/logout-all/selective revocation | partial | Logout bumps token version and clears the stored refresh state; no RFC 7009 endpoint or selective multi-session model. |
| Forced logout | implemented and verified locally | Token-version bump and account state close current bearer sessions; auth/full tests passed. |
| Password reset | implemented and verified locally | Hashed reset token, expiry and session closure are implemented and covered by auth tests. |
| Email verification | implemented and verified locally | Hashed verification token, expiry, resend and status checks are implemented and covered by auth tests. |
| Account activation/suspension/disabling/deletion | partial | Deactivation/reactivation and deletion-pending/purge exist and tests passed; no separate suspension/admin workflow. |
| TOTP MFA/replay prevention/recovery codes | absent | No TOTP, recovery-code, or MFA challenge implementation. |
| WebAuthn/passkeys | absent | No WebAuthn implementation or tests. |
| OIDC provider discovery | partial | Minimal unauthenticated discovery document exists; it omits required endpoint and flow metadata and has no complete discovery test. |
| OIDC authorization endpoint | absent | No `/authorize`. |
| OAuth token endpoint | absent | `/auth/refresh` is a proprietary JSON route, not `/token`. |
| Authorization Code flow | absent | No code issuance, redemption, client redirect, or tests. |
| PKCE S256 | absent | No `code_challenge`/verifier handling. |
| State/nonce handling | absent | No OIDC authorization state/nonce; unrelated messaging nonces do not count. |
| Exact registered redirect URI validation | absent | No client registry or redirect URI store. |
| ID-token issuance/validation | absent | Auth response has proprietary `token` and `refreshToken`, not `id_token`; no ID-token claim validation. |
| Access/refresh tokens (OAuth semantics) | partial | Proprietary access JWT and refresh token exist; OAuth token endpoint and standards semantics do not. |
| JWKS publication | implemented and verified locally | One RSA public key is served and handler tests cover JWKS response. |
| Signing-key rotation/overlap | absent | One loaded key/JWK; no key ring, overlap, retirement, or rotation test. |
| UserInfo | absent | App profile queries are not an OIDC UserInfo endpoint. |
| Revocation endpoint | absent | No `/revoke`; proprietary logout only. |
| RP-initiated logout | absent | No OIDC logout endpoint or `post_logout_redirect_uri`. |
| Client registry/client authentication | absent | API key/password gates are application auth, not OAuth client registration/authentication. |
| Issuer/audience validation | implemented and verified locally | Local JWT verifier validates configured issuer and audience; focused and race tests passed. |
| `azp` validation | absent | No authorized-party check. |
| Scope and claim handling | absent | No OAuth scopes or standard OIDC claim negotiation; custom email/tv/did only. |
| Session/consent behavior | partial | App privacy-consent ledger exists; no OAuth authorization session or scope consent screen. |
| Negative protocol tests | absent | JWT/middleware negative tests exist, but no OIDC/OAuth protocol suite. |
| Roles/permissions | absent | No role/permission model in auth boundary. |
| Organization/tenant membership/isolation | absent | No tenant or organization fields/authorization checks. |
| Audit events/redaction | absent | No dedicated auth audit event stream or redaction policy. |
| SMTP delivery | implemented and verified locally | SMTP and log mailers exist and package tests passed; localization and production delivery verification are absent. |
| Localized identity templates | absent | No locale/template registry. |
| CSRF/cookies/BFF/browser token storage | absent | Bearer/API-key GraphQL and native Keychain client; no cookie/BFF/CSRF implementation. |
| CORS policy | implemented but not verified | Terraform config permits wildcard origins; no browser security regression test. |
| PostgreSQL schema/migrations | not applicable | This project uses DynamoDB and has no PostgreSQL artifacts. |
| DynamoDB schema/concurrency/cleanup | partial | User/app tables, uniqueness transactions and purge code exist; optional DynamoDB Local contention/cleanup tests were not run. |
| Readiness/liveness | implemented and verified locally | Health routes and their package/full-suite tests passed; deployment verification was not run. |
| Telemetry/rate/resource limits | partial | IP rate limiter and analytics exist; no complete auth resource-limit or telemetry regression suite. |
| Unit tests | implemented and verified locally | Focused auth, adjacent and complete Go unit suites passed. |
| Race tests | implemented and verified locally | Focused auth-boundary `go test -race` passed; no repository-wide race target exists. |
| Fuzz tests | absent | No auth/OIDC fuzz targets found. |
| DynamoDB integration tests | blocked | DynamoDB Local tests are included but require external service/dependencies and were not run. |
| PostgreSQL integration tests | not applicable | No PostgreSQL implementation or test target exists. |
| Security regression tests | partial | JWT, password, middleware and selected auth regressions exist; no OIDC or full refresh-concurrency suite. |
