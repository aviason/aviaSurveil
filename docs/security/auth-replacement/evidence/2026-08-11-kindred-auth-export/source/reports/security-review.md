# Candidate security review

This review is an offline, source-based assessment of the exported boundary. It is not a penetration test, deployment review, or production approval. Findings were corroborated against the copied source and the independent inventory/security review; blocked test execution is recorded rather than inferred as a pass.

## Findings

### High: unauthenticated REST logout can target an arbitrary user identifier

`internal/auth/handler.go` registers the legacy REST logout route with rate limiting but without the bearer-auth middleware. `internal/auth/service.go` derives a user ID from the refresh-token prefix and bumps that user's token version/clears the refresh state without first proving the supplied token hash belongs to that user. A caller who learns or guesses a user ID can submit a fabricated `userID.anything` value and cause session invalidation (denial of service). GraphQL logout is protected, but the REST route remains a candidate risk. Remediation: require authenticated identity and verify the refresh hash/device before any revocation; use an opaque server-side lookup rather than trusting a user-ID prefix.

### High: password change does not invalidate existing sessions

The ordinary password-change path replaces `PasswordHash` but does not bump `TokenVersion` or close refresh state. Password reset and account lifecycle paths do close sessions. A stolen bearer or refresh token can therefore remain usable after a user changes their password. Remediation: perform an atomic token-version/session-family invalidation on every password change and add a regression test for access and refresh tokens issued before the change.

### High: no signing-key rotation or overlap

Runtime wiring loads one RSA key, exposes one JWK, and has no old-key overlap, retirement, rotation schedule, or rotation test. Terraform documentation notes that replacing the key invalidates outstanding access tokens. This is a key-availability and incident-response blocker for any production issuer.

### High: not an OIDC provider despite metadata endpoints

Only minimal discovery/JWKS metadata is served. Authorization code, PKCE, redirect URI registration, state/nonce, ID tokens, UserInfo, revocation, RP logout, client authentication, scopes and protocol negative tests are absent. Consumers must not treat the metadata as a conforming provider.

### Medium: refresh state is single-row and not concurrency-safe by evidence

One refresh hash/expiry/device is stored on the user record. Rotation reads and overwrites this state, but no conditional compare-and-swap, token-family table, concurrent refresh test, crash recovery test, or explicit old-token replay regression was found. This is insufficient evidence for robust family-wide reuse detection and multi-device behavior.

### High: concurrent full-row writes can lose security state

The user repository rewrites the complete user item without a prior-version or field-scoped compare-and-swap. Refresh rotation, logout, failed-login/lockout accounting, phone verification, password updates and other lifecycle mutations can race; logout also ignores an update error. A stale write can restore an old refresh hash/token version, lose lockout state, or overwrite a newer password. Remediation: use conditional field updates or an optimistic version for every security-state mutation and test contention.

### Medium: rate limiting can fail open and trusts forwarded identity headers

The fixed-window limiter can allow requests when its backing store errors, and the default client-IP extraction trusts `X-Forwarded-For`/`X-Real-IP`. Unless a trusted proxy overwrites these headers, an attacker can evade the IP limit. Remediation: fail closed for sensitive auth operations, enforce trusted proxy configuration, and add account/email/device throttles.

### Medium: legacy HMAC configuration/documentation conflicts with RS256 runtime

README/OpenAPI describe HMAC bearer tokens and Terraform still carries `TOKEN_SECRET` plumbing, while the Go signer/verifier uses RS256 and configured issuer/audience. The mismatch can cause unsafe deployment assumptions or an accidentally disabled edge authorizer when the JWT key SSM path is empty. Remove stale secret plumbing/docs and make the edge-auth configuration mandatory and testable.

### Medium: wildcard CORS with a bearer-only browser boundary

Reviewed Terraform permits wildcard origins. There is no cookie/BFF/CSRF boundary, so any future browser use of bearer tokens would need explicit origin policy and storage guidance. Native mobile Keychain storage is documented; browser token storage is not implemented.

### High: phone OTP and authenticated step-up paths lack attempt throttles

Six-digit phone verification codes remain valid until expiry without an attempt counter, and authenticated password step-up/change/deactivate/delete routes are not wrapped by the auth limiter. A stolen bearer can make unlimited OTP guesses or brute-force a weak old password. Add atomic attempt counters, account/destination/device limits, and lockout/backoff for step-up operations.

### Medium: account-state responses and timing can enumerate users

Duplicate registration and lockout responses are explicit, and unknown-user login can return before the expensive Argon2 verification while known-user failures perform it. Normalize public errors/timing where enumeration resistance is required, using a dummy hash for unknown accounts.

### Low/medium: development delivery adapters expose sensitive verification material

The log mailer/SMS sender intentionally logs password-reset or verification metadata for local development, and the SMTP body contains reset material. Ensure these adapters cannot be selected in production, redact tokens/codes from structured logs, and add a redaction test. No customer values or operational delivery configuration is included in this export.

### Medium: no administrative authorization, tenancy, or auth audit boundary

No role/permission/organization/tenant model, admin provisioning/invitation flow, or dedicated auth audit event/redaction stream was found. Account lifecycle and privacy-consent events exist but do not provide administrative security isolation.

### High: registration can issue a session before required email verification

The registration path creates an unverified user and immediately issues access/refresh tokens. The configured verification requirement is enforced on login but is not consistently enforced by the registration-issued session, token verification, or refresh path. If that configuration is enabled, this is a candidate bypass. Remediation: gate initial issuance, verification, and refresh on the same account policy and add a test with verification required.

### High: GraphQL public auth mutations bypass the REST limiter

The IP limiter is wired to the REST router, while AppSync API-key mutations expose login, registration, refresh, verification and reset operations without the same application limiter. This enables brute force, lockout and reset/resend abuse through the GraphQL path. Add shared distributed per-IP/account/device limits at the GraphQL boundary and test each public mutation.

### High: WebSocket sessions are not revalidated after connect

The copied `internal/messagews` handler verifies a bearer only at `$connect`, stores a 24-hour connection/subscription TTL, and then authorizes later events from the connection row without checking JWT expiry, token version, account status or TTL. Logout, password reset and deactivation do not force-disconnect these rows. Revalidate on every event or disconnect on revocation, and filter fanout by expiry.

### Medium: expensive password hashing has no effective upper bound

Argon2id is deliberately expensive, but handlers and GraphQL decoding do not impose a small maximum password/body size. Large concurrent public requests can consume memory/CPU, especially where GraphQL rate controls are absent. Cap request bodies and password length before hashing and add resource-limit tests.

### Medium: broad shared infrastructure role increases auth-data blast radius

The deployment uses a shared Lambda role for API, GraphQL, websocket, workers and purge paths with access to users/app tables and object storage. A compromise in a lower-trust worker could read or mutate password/refresh hashes and token versions. Split roles and scope table/index/object permissions per workload.

## Positive controls observed

- RS256 algorithm allow-list and rejection of HS256 algorithm confusion in JWT tests.
- Issuer, audience, expiry, signature, token-version and account-status checks in the proprietary verifier path.
- Argon2id password hashing with random salts and constant-time verification.
- Hashed verification/reset/refresh tokens and idle/absolute refresh expiry fields.
- Failed-login counters/lockout, email normalization/uniqueness transactions, and session closure on reset/deactivation/delete.
- Native client stores bearer material in Keychain rather than a browser cookie/local-storage path.

These controls are source evidence; the complete Go unit suite and focused auth race run passed, while the listed concurrency/deployment gaps remain unverified.

## Coverage limitations

No production deployment, live endpoint, external identity provider, database dump, PostgreSQL system, fuzz run, or DynamoDB Local run was available. The focused auth race run and complete Go unit suite passed; the export scan is limited to the final sanitized tree and ZIP and cannot prove absence of secrets outside the selected files.
