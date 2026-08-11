# Authentication Source Comparison

## Decision

Use both owner-provided exports as candidate-only reference collections. Do not
select either as a deployable authentication service and do not change the
selected architecture. Build a separately privileged Go OIDC provider backed
by AviaSurveil360-owned PostgreSQL state and a maintained authorization-server
library; keep application authorization in the current API.

## Capability Selection

| Capability | First export | Kindred export | Avia adaptation decision |
|---|---|---|---|
| Immutable subject | Random identifiers | UUID subject | Define one opaque immutable provider subject; never derive it from email. |
| Password hashing | Argon2id concepts | Argon2id with encoded parameters | Adapt only after parameter, input-size, memory, concurrency, rehash, history, and dummy-hash controls are fixed. |
| Access token verification | Proprietary JWT | RS256 with algorithm, issuer, audience, time, token-version, and account-state checks | Keep the useful negative JWT tests, but use the maintained OIDC library and provider contracts for token issuance and verification. |
| Refresh state | PostgreSQL row-lock/family concepts | Hashed one-session record with idle/absolute expiry and device binding | Use Avia-owned PostgreSQL refresh families with conditional rotation, reuse revocation, multiple-session policy, crash recovery, and retention. |
| Client refresh behavior | Server-oriented | Swift actor serializes simultaneous refreshes and clears rejected sessions | Port the behavior to Avia's browser/BFF test contract where applicable; browser code must not receive provider refresh tokens. |
| Verification/reset | Incomplete | Hashed expiring email verification and password-reset challenges | Adapt the challenge model with single-use, attempt bounds, generic outcomes, durable delivery, and four-locale templates. |
| Account lifecycle | Incomplete | Lock, deactivate, deletion-pending, purge, and token-version revocation | Use as test input only; Avia account states and administrative authority remain canonical. |
| Abuse controls | Missing/incomplete | IP fixed-window and DynamoDB counter | Reimplement a trusted-proxy-aware, distributed, multi-key control with explicit fail-closed behavior for sensitive operations. |
| OIDC | Absent | Misleading minimal discovery/JWKS facade | Reuse neither protocol surface. Select and prove a maintained authorization-server library. |
| Persistence | PostgreSQL evaluation schema duplicates app authority | DynamoDB user row mixes security state | Import neither schema/store. Create focused PostgreSQL identity tables and roles. |
| Authorization | Duplicated users/permissions without Avia isolation | No role, permission, organization, or tenant model | Keep exact role/organization authority solely in Avia's application database and BFF/API policy. |
| Realtime sessions | Not material | WebSocket checked at connect only | Add per-event expiry/account/auth-version validation or durable disconnect-on-revocation semantics if Avia later has authenticated long-lived channels. |
| Signing keys | Incomplete | One RSA key/JWK with no overlap | Implement a key ring, stable `kid`, activation/overlap/retirement states, readiness checks, and rotation recovery tests. |

## Mandatory Regression Additions

The second export adds the following explicit regression contracts after the
first export's `AS360-AUTH-001` through `AS360-AUTH-018` set:

| ID | Required behavior |
|---|---|
| `AS360-AUTH-019` | Two refresh requests using one predecessor cannot both succeed; reuse revokes the affected family deterministically. |
| `AS360-AUTH-020` | Logout can revoke only a session owned by the authenticated subject and never trusts an identifier embedded in caller-controlled token text. |
| `AS360-AUTH-021` | Password change atomically revokes every prior provider and application session. |
| `AS360-AUTH-022` | Concurrent security-state changes cannot restore stale token, lockout, password, factor, account, or auth-version state. |
| `AS360-AUTH-023` | A required verification policy gates registration issuance, login, refresh, token use, and relevant recovery paths consistently. |
| `AS360-AUTH-024` | Every public auth transport has shared per-source, principal, device, and global limits; trusted proxy identity is explicit and sensitive-operation limiter failure is closed. |
| `AS360-AUTH-025` | Authenticated long-lived connections cannot outlive token expiry, auth-version change, suspension, password reset, logout, or deactivation. |
| `AS360-AUTH-026` | Signing-key activation, overlap, retirement, cache behavior, restart, and emergency rotation preserve only the intended validation window. |
| `AS360-AUTH-027` | Legacy HMAC configuration cannot enable, disable, or alter the RS256/OIDC verification path. |
| `AS360-AUTH-028` | Concurrent client requests share one refresh attempt, preserve only the accepted successor, and clear the complete local session after terminal rejection. |
| `AS360-AUTH-029` | Verification codes and authenticated step-up operations have atomic attempt bounds and backoff without attacker-triggered permanent denial. |
| `AS360-AUTH-030` | Body, identifier, password, and expensive-hash work are bounded before allocation or Argon2id execution. |

## Rejected Direct Reuse

DynamoDB-specific repositories, application GraphQL resolvers, product consent
records, AppSync/API Gateway wiring, mobile operational configuration, and
Kindred domain models are outside the replacement runtime. Keeping them in the
retained evidence tree supports review and provenance; it does not make them
AviaSurveil360 dependencies.

Both exports remain `candidate-only`. The comparative intake is
`verified locally`; implementation and protocol conformance are `not run`;
`production-ready: not established`.
