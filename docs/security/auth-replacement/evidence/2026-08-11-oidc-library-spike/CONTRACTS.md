# Provider-neutral contracts and AS360-AUTH test map

Date: 2026-08-11

These contracts are frozen for the evidence gate only. They do not select a
library and do not authorize `apps/auth/`. The current Keycloak issuer remains
the serving and rollback baseline until a later owner decision and all gates
pass.

## Browser and RP contract

- The browser uses the existing same-origin API/BFF session and CSRF boundary;
  browser JavaScript never receives a provider refresh token.
- Authorization Code + PKCE S256 is mandatory. The RP generates and binds
  unpredictable `state` and `nonce`, validates both on the callback, and
  rejects a missing, mismatched, reused, or expired value.
- Redirect URIs are exact, pre-registered values. No wildcard, prefix, open
  redirect, fallback origin, or user-controlled post-logout URI is accepted.
- The issuer is one exact configured URL. Discovery issuer, token issuer,
  ID-token `iss`, and JWKS origin must match it exactly; no dual-issuer or
  token-verification fallback is permitted.
- The RP validates ID-token signature and algorithm allow-list, issuer,
  audience, authorized party where configured, time bounds, nonce, and an
  immutable opaque `sub`. Unknown `kid`, retired key, stale issuer, wrong
  audience, algorithm confusion, and malformed claims fail closed.
- RP-initiated logout accepts only a registered client/redirect and a
  subject-bound ID-token hint. It revokes the BFF/provider session and never
  trusts a caller-controlled subject or redirect.

## Provider and authority contract

The host/provider adapter must expose provider-neutral operations, not
Keycloak-shaped APIs:

| Boundary | Required behavior |
| --- | --- |
| Account provisioning | Idempotent operation ID, immutable provider subject, canonical identifier normalization, verification gate, organization/membership facts supplied by Avia authority |
| Authority observation | Read-only provider status, factor state, required actions, client/issuer/key version, and provider revision; stale or unavailable state fails closed for sensitive actions |
| Lifecycle | Lock, suspend, deactivate, deletion-pending, and tombstone state are explicit and checked by login, refresh, token use, and session validation |
| Required actions | Password change, verification, MFA/step-up, and recovery actions are single-use/attempt-bounded, auditable, and cannot be bypassed by refresh or client variation |
| MFA state | TOTP and hashed recovery codes have enrollment, challenge, attempt, replay, reset, and revocation semantics; WebAuthn remains a separate product decision |
| Session revocation | Password change, logout, lifecycle change, factor reset, refresh reuse, and provider auth-version changes revoke the exact affected sessions/families atomically |
| Recovery | Admin-assisted, operation-ID-bound, notification-safe, redacted, and subject/organization scoped; generic public outcomes prevent enumeration |
| Application authority | Roles, organizations, assignments, and CAA workload stay in Avia's application database and BFF policy; provider claims are inputs, never a second authorization authority |

## Database, secret, and operational contract

- Provider credential/factor/signing state uses a separate PostgreSQL role and
  schema or logical database. API and worker roles receive no provider
  signing, MFA, SMTP, or provider-admin secrets.
- Provider signing keys, active/overlap/retired `kid` records, MFA secrets,
  recovery material, client secrets, and SMTP credentials have independent
  secret identities and rotation/recovery procedures. Readiness fails closed
  when required material is missing, weak, placeholder, or inconsistent.
- The provider does not own Avia organizations, findings, CAPs, evidence,
  CAA decisions, or auditee projections. Those remain in Avia's canonical
  database and audit boundary.
- Every security-sensitive mutation has an idempotency key, expected revision,
  typed conflict, durable redacted audit event, and an explicit fail-closed or
  fail-alert result. Logs never contain credentials, tokens, recovery codes,
  raw bodies, or unbounded attacker strings.
- Readiness checks issuer/client/key/DB/secret dependencies without leaking
  their values. Liveness does not claim authentication readiness.
- Native ARM64 mixed-workload measurements must record steady/peak RSS, CPU,
  Argon2id queue/rejections, DB pools, latency, restart time, and headroom;
  no such production or release evidence exists in Task 1.

## AS360-OIDC-WEB-1 negative contract

The disposable run sheets cover the library-owned core where possible. Before
any provider is authorized for Task 2, the selected path must add the full
negative matrix below, including cases marked `not run` in the results.

| Contract | Required negative behavior | Task 1 evidence state |
| --- | --- | --- |
| Discovery/issuer | Wrong issuer, HTTPS/origin mismatch, missing endpoint, stale metadata, and discovery/JWKS origin drift fail closed | ZITADEL positive verified; framework gaps and Hydra full server `not run` |
| Client/redirect | Unknown client, wrong secret/auth method, unregistered exact redirect, wildcard/prefix redirect, and wrong post-logout redirect are rejected without redirecting to attacker input | Redirect negatives verified in ZITADEL/Fosite/Authelia; remaining cases `not run` |
| State | Missing, mismatched, replayed, expired, or cross-tab state is rejected by the RP | Positive echo checks only; negative state `not run` |
| Nonce | Missing, mismatched, replayed, or cross-client nonce is rejected by ID-token validation | Positive claim checks only; negative nonce `not run` |
| PKCE/code | Missing or wrong verifier, wrong method, wrong client/redirect, expired code, and replayed code fail; one predecessor cannot mint two tokens | Wrong verifier and replay verified in ZITADEL/Fosite/Authelia; remaining cases `not run` |
| ID token | Wrong algorithm, malformed signature, wrong issuer/audience/azp, missing subject, bad time, and wrong nonce fail | Positive verification verified; wrong issuer/audience/alg `not run` |
| JWKS/key ring | Unknown `kid`, stale cache, retired key, overlap boundary, restart, emergency rotation, and rollback validate only intended keys | ZITADEL JWKS positive; overlap/retirement and unknown `kid` `not run` |
| Logout | Unregistered redirect, wrong subject hint, cross-client hint, replayed logout, and stale session are rejected/revoked safely | ZITADEL redirect negative/default redirect; subject binding and replay `not run` |
| Claims/authority | Mutable email/role/organization cannot replace immutable subject; provider claims cannot bypass current Avia membership/revision | Not run; required in adapter/application gates |

## AS360-AUTH-001 through AS360-AUTH-030

The first 18 IDs are preserved from the first source review; 019–030 are the
second source's additional regression contracts. A library selection does not
close any finding by itself.

| ID | Frozen regression contract | Planned owning gate |
| --- | --- | --- |
| `AS360-AUTH-001` | Pending/unapproved accounts cannot cross the product-access gate | Provider status + API/BFF authorization |
| `AS360-AUTH-002` | Registration and recovery bound Argon2id memory, input, queue, and concurrency work before hashing | Provider admission/rate-limit/ARM64 |
| `AS360-AUTH-003` | No seeded/static privileged password or unsafe seed against a non-disposable DB | Bootstrap and secret gate |
| `AS360-AUTH-004` | Empty requested staff scope never matches a scoped grant; global grants are explicit | Application authority |
| `AS360-AUTH-005` | Unknown identifiers use constant-cost behavior; layered source/principal/global limits and generic outcomes prevent guessing | Provider abuse controls |
| `AS360-AUTH-006` | Password length, compromise/reuse/history, parameter rehash, and safe maximum are enforced consistently | Provider credential policy |
| `AS360-AUTH-007` | Password change revokes every old refresh family or atomically replaces the current one | DB concurrency/session gate |
| `AS360-AUTH-008` | Lock/suspend/deactivate/deletion states fail closed in login, refresh, token use, and session validation | Lifecycle/reconciliation |
| `AS360-AUTH-009` | Canonical identifiers are verified, typed, unambiguous, normalized, and never silently collide | Provisioning/recovery |
| `AS360-AUTH-010` | Every login-identifier change requires recent step-up, notification, and audit | MFA/step-up |
| `AS360-AUTH-011` | Security events are durable, redacted, complete, and have explicit failure semantics | Audit/observability |
| `AS360-AUTH-012` | Disallowed clients cannot leave durable orphan sessions/tokens | Client policy/transaction |
| `AS360-AUTH-013` | Configured step-up expiry is the single enforced server-derived boundary | Session/step-up |
| `AS360-AUTH-014` | Discovery is not advertised unless all claimed OIDC endpoints and checks exist | OIDC conformance |
| `AS360-AUTH-015` | Signing/readiness rejects weak, missing, placeholder, stale, or unrotatable key material | Key-ring/readiness |
| `AS360-AUTH-016` | Reachable dependencies have current advisory review and no unaccepted High/Critical result | Dependency/SBOM |
| `AS360-AUTH-017` | Organization/membership filters and claims isolate every repository and API path | Application authority |
| `AS360-AUTH-018` | Cross-user role enumeration requires explicit manager/admin authority | Application authorization |
| `AS360-AUTH-019` | Two refreshes of one predecessor cannot both succeed; reuse revokes the family deterministically | Refresh concurrency |
| `AS360-AUTH-020` | Logout revokes only the authenticated subject's session and never trusts caller-controlled token text | Logout/session binding |
| `AS360-AUTH-021` | Password change atomically revokes prior provider and application sessions | Credential lifecycle |
| `AS360-AUTH-022` | Concurrent security-state writes cannot restore stale token, lockout, password, factor, account, or auth-version state | Revision/CAS/race |
| `AS360-AUTH-023` | Required verification gates registration issuance, login, refresh, token use, and recovery consistently | Verification policy |
| `AS360-AUTH-024` | Every auth transport shares source/principal/device/global limits; trusted proxy identity is explicit; sensitive limiter failure closes | Abuse controls |
| `AS360-AUTH-025` | Long-lived connections cannot outlive expiry, auth-version change, suspension, reset, logout, or deactivation | Session/channel gate |
| `AS360-AUTH-026` | Key activation, overlap, retirement, cache, restart, and emergency rotation preserve only intended validation windows | Key-ring/recovery |
| `AS360-AUTH-027` | Legacy HMAC configuration cannot alter the RS256/OIDC verification path | Configuration isolation |
| `AS360-AUTH-028` | Concurrent client refreshes share one attempt, retain only the accepted successor, and clear terminal rejection | BFF/client refresh |
| `AS360-AUTH-029` | Verification and step-up codes have atomic attempt bounds/backoff without attacker-triggered permanent denial | MFA/recovery |
| `AS360-AUTH-030` | Body, identifier, password, and expensive-hash input are bounded before allocation/work | HTTP/provider admission |

All 30 contracts remain open for implementation/qualification. The Task 1
evidence records no closure, no selected library, and no authorization to move
to Task 2.
