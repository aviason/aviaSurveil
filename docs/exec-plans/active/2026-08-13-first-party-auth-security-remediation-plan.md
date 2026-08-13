# First-Party Authentication Security Remediation

Date: 2026-08-13
Last updated: 2026-08-13
Status: active
Implementation state: Gate 0 complete; Work packages 1–7 and AUTH-SEC-06/11 follow-ups complete
Release state: candidate-only; release pending
Predecessor: 2026-08-11-first-party-go-oidc-auth-replacement-plan.md
Implementation baseline revision: aa9f63f2bef60866b6d53ba78fa14f7de6f561c3
Original audit base revision: 74b84cf597e102ecae30b3d1cfc96ec84cde0480
Source state: auth remediation is committed and pushed on `main`; preserve the
remaining unrelated AviaCore changes

## Objective and user-visible outcome

Remediate the validated first-party authentication findings without replacing
the selected Go OIDC provider, weakening privacy behavior, or introducing a
second identity path. The resulting candidate must:

- bind each BFF OIDC state to the initiating browser;
- reject abusive authorization, login, MFA, and recovery traffic before it can
  create unbounded durable state or consume unbounded password-hashing work;
- keep attacker-generated credential failures from changing account lifecycle,
  auth revision, or existing session validity;
- make MFA and recovery attempt budgets durable, time-bounded, and safe under
  concurrency;
- bind every pending authorization request and code to the account auth
  revision that authenticated it;
- issue at most one useful recovery challenge and one pending recovery delivery
  for one subject and purpose during a cooldown;
- use one owned challenge-token validation boundary; and
- preserve exact issuer-prefixed routing behind the gateway.

This plan does not itself close a finding. A finding is fixed only after its
mapped implementation, regression test, required local gates, and final
security diff review pass. The result remains candidate-only and release
pending.

## Evidence basis and current observations

The plan is derived from the 2026-08-12 first-party auth source audit and a
fresh inspection of the current working tree at the source revision above. The
original scan lived in a temporary directory that no longer exists on
2026-08-13, so this document assigns stable local identifiers AUTH-SEC-01
through AUTH-SEC-11 and AUTH-INT-01. These identifiers are planning keys, not
claims that the temporary scan artifact remains available or sealed.

Current source inspection establishes:

- apps/auth/internal/provider/runtime.go accepts repeated TOTP attempts for one
  staged authorization request and redirects MFA to the root-relative /mfa
  path.
- apps/auth/internal/identity/provider_authority.go performs password policy
  and Argon2id work before it validates activation or password-recovery
  credentials.
- apps/api/internal/httpapi/security.go keys auth admission from RemoteAddr,
  while the deployed gateway makes that peer the shared proxy.
- apps/auth/internal/identity/postgres.go changes an account to locked,
  increments auth_revision, and invokes session revocation after password
  failures.
- apps/auth/internal/mfa/postgres.go persists a recovery failure count without
  a recovery time boundary.
- apps/auth/internal/provider/postgres.go does not persist the authenticating
  auth_revision on pending authorization requests or authorization codes.
- apps/auth/internal/provider/runtime.go can create a new challenge and mail
  delivery for every eligible anonymous recovery request.
- apps/api/internal/platform/session/manager.go consumes OIDC state by state
  alone, without a browser-held binding secret.
- public authorization and BFF login-state allocation rely primarily on later
  cleanup and do not have a transactionally enforced outstanding-state ceiling.
- the current auth baseline already uses challenge.DigestToken in the password and
  MFA recovery consumers. AUTH-SEC-07 must therefore begin with revalidation
  and preserve or strengthen that candidate correction; it must not blindly
  duplicate it.

## Scope

In scope:

- apps/auth credential, throttle, challenge, MFA, mail, provider, runtime,
  configuration, migration, and focused qualification surfaces;
- apps/api BFF auth handlers, session manager, login-state persistence,
  application auth admission, migrations, and focused integration tests;
- deploy/local gateway and Compose contracts needed to preserve the same-origin
  browser-binding and public/private route split;
- forward-only auth migration 000009 and API migration 000046;
- regression, concurrency, migration, local lifecycle, fault/restart, and
  security diff verification;
- plan, plan index, and durable evidence synchronization.

Out of scope:

- another identity provider, compatibility fallback, refresh-token enablement,
  or migration of real identities;
- remote systems, DNS, external SMTP, deployment, traffic, production secrets,
  production data, or release approval;
- unrelated API, web, AviaCore, AGA, or documentation cleanup;
- branches, commits, staging, pushes, or destructive working-tree cleanup;
- weakening PKCE, nonce, cookie, private-admin, authority, audit, or privacy
  controls to make tests pass.

## Assumptions and ownership boundaries

- The first-party Go OIDC provider is the only maintained identity
  implementation.
- The current namespace is disposable and contains synthetic candidate data,
  but implementation must still use forward-only migrations and safe
  concurrency semantics.
- The API owns BFF login state and browser cookies. The auth service owns
  passwords, MFA, challenges, provider authorization state, provider sessions,
  recovery mail coordination, and auth-side admission.
- A browser-binding cookie is an anti-CSRF and fairness input, not an
  authentication credential. Possession never grants identity or authority.
- No security decision may trust Forwarded, X-Forwarded-For, X-Real-IP, or the
  gateway socket peer as an end-user identity. The gateway must overwrite
  forwarding headers for observability, but auth admission uses server-issued
  browser binding plus durable global, identifier, subject, request, and client
  rules.
- Error behavior must remain account-enumeration neutral. Recovery always
  returns the same no-store accepted response, including rate-limit and backend
  rejection paths that can be handled safely.
- Existing user changes in the working tree are authoritative. The implementer
  must capture a task baseline and never reset, restore, or overwrite them.

## Selected design

We will consolidate auth abuse policy around durable, operation-specific rule
sets and state-machine invariants. The same browser-binding secret used to bind
OIDC state provides a per-browser fairness key. It is combined with a durable
global emergency ceiling and, when available, normalized identifier, subject,
client, and authorization-request keys. A global rule is evaluated in the same
transaction as attacker-variable keys so attacker-created buckets have a
bounded creation rate and a bounded retention period.

Credential failures become temporary authentication backoff only. They do not
change lifecycle state, auth_revision, or existing sessions. Pending OIDC state
captures auth_revision and is revalidated at every transition that can produce
a code or token. Challenge validation is centralized so token digest, purpose,
subject, state, expiry, attempt, and one-time consumption checks cannot drift
between password and MFA consumers.

Alternatives considered:

1. Local handler guards only. This has the smallest initial diff but retains
   duplicated challenge logic and inconsistent rate-limit ownership. It is
   rejected because the current findings already demonstrate control drift.
2. Durable state-bound auth controls in the existing API and auth services.
   This is selected because it removes the shared-proxy assumption, preserves
   the current deployment boundary, and gives every finding a testable owner.
3. A new edge admission service or Redis dependency. This could centralize
   policy across services, but adds a new privileged runtime and operational
   failure mode for a candidate that already has PostgreSQL. It is deferred
   unless measured load proves the selected PostgreSQL path inadequate.

### Engineering tradeoffs

| Dimension | Expected direction | Basis | Validation |
|---|---|---|---|
| Security | improves | source-derived | Re-run every mapped abuse, replay, cross-browser, and revocation test. |
| Performance | mixed | source-derived | Sensitive requests gain small PostgreSQL transactions; invalid activation/reset requests avoid Argon2id. Run focused load and concurrency tests. |
| Memory | neutral | source-derived | No new service or unbounded in-memory cache; only bounded row/index additions. Inspect row ceilings and cleanup tests. |
| Reliability | improves | source-derived | Admission fails closed and state transitions become atomic; PostgreSQL remains a required dependency. Exercise database loss and restart. |
| Operability | modest cost | hypothetical | Add redacted counters for denial and cleanup classes; verify no secrets or identifiers are logged. |
| Migration | moderate | source-derived | Two forward-only migrations and interface changes cross auth/API tests. Exercise fresh and already-migrated databases. |
| Rollback | candidate rebuild | source-derived | Do not re-enable vulnerable behavior. Stop the candidate and restore/recreate the disposable namespace if forward repair is not possible. |

## Fixed security invariants

1. OIDC state can be consumed only with the state value and the matching
   browser-held secret; a wrong browser cannot consume the legitimate flow.
2. No public path creates an authorization request, BFF login state, challenge,
   mail row, or attacker-variable throttle row before a global admission rule
   succeeds.
3. Every allocation path has a transactionally enforced outstanding-state cap
   in addition to periodic bounded cleanup.
4. A password failure never changes account lifecycle state, auth_revision, or
   provider/application session state.
5. Temporary password and recovery-code backoff expires automatically, cannot
   be extended indefinitely by requests made while it is active, and can be
   applied again after a fresh failure window.
6. Every submitted MFA code consumes one durable attempt reservation for the
   exact authorization request. Parallel requests cannot exceed the budget.
7. Invalid, expired, consumed, wrong-purpose, and wrong-subject activation or
   recovery credentials are rejected before password policy history checks or
   Argon2id hash work.
8. A valid token is revalidated and consumed in the same transaction as the
   protected mutation; prevalidation alone never authorizes a mutation.
9. Challenge token digesting and usable-state checks have one owner. Identity
   and MFA consumers do not reproduce token hashing or raw challenge SQL.
10. A pending request or code cannot outlive the auth_revision that performed
    password/MFA authentication. Revocation removes pending artifacts as
    cleanup, while revision checks remain the race-safe enforcement.
11. One subject and recovery purpose has at most one active challenge and one
    pending delivery during its cooldown. Unknown-account requests create no
    subject, challenge, or mail state.
12. Public recovery responses and logs do not disclose account existence,
    address, subject, token, cookie, state, code, or throttle key.
13. MFA and recovery redirects are derived from the configured issuer prefix;
    no root-relative handler path bypasses /identity behind the gateway.

## Repository orientation and affected interfaces

- apps/auth/internal/throttle owns durable multi-rule admission and cleanup.
- apps/auth/internal/identity owns password verification and temporary account
  backoff semantics.
- apps/auth/internal/mfa owns TOTP replay protection and time-bounded recovery
  code failure state.
- apps/auth/internal/challenge owns challenge digest, validation, row locking,
  attempts, one-active-per-purpose policy, and transactional consumption.
- apps/auth/internal/mail owns encrypted delivery preparation, transactional
  enqueue/deduplication, retry, lease, and retention cleanup.
- apps/auth/internal/provider owns authorization-request admission, MFA attempt
  reservations, auth_revision binding, code/token transition checks, recovery
  coordination, and issuer-prefixed routes.
- apps/api/internal/httpapi owns browser cookie transport and removes
  /auth/login from the socket-peer limiter.
- apps/api/internal/platform/session owns browser-bound OIDC login state,
  durable API-side admission, outstanding-state caps, and cleanup.
- deploy/local/gateway owns same-origin routing and canonical overwrite of
  forwarding headers; port 8081 remains private.

Expected primary files include:

- apps/auth/migrations/000009_first_party_auth_security_hardening.up.sql
- apps/auth/migrations/migrations.go
- apps/auth/internal/throttle/limiter.go
- apps/auth/internal/throttle/postgres.go
- apps/auth/internal/identity/store.go
- apps/auth/internal/identity/postgres.go
- apps/auth/internal/identity/provider_authority.go
- apps/auth/internal/challenge/store.go
- apps/auth/internal/challenge/postgres.go
- apps/auth/internal/mfa/mfa.go
- apps/auth/internal/mfa/postgres.go
- apps/auth/internal/mfa/recovery.go
- apps/auth/internal/mail/outbox.go
- apps/auth/internal/provider/runtime.go
- apps/auth/internal/provider/postgres.go
- apps/auth/internal/provider/candidate.go
- apps/auth/cmd/auth/main.go
- apps/api/migrations/000046_oidc_login_state_security.up.sql
- apps/api/migrations/migrations.go
- apps/api/internal/httpapi/auth.go
- apps/api/internal/httpapi/security.go
- apps/api/internal/platform/session/manager.go
- deploy/local/gateway/Caddyfile
- deploy/local/gateway/Caddyfile.preprod
- deploy/local/gateway/Caddyfile.preprod.http

This list is a boundary, not permission to rewrite every file. Touch only files
needed by the accepted work packages and their tests.

## Finding-to-work mapping

| ID | Finding | Work packages | Required proof |
|---|---|---|---|
| AUTH-SEC-01 | TOTP has no attempt limit | 1, 3 | Durable exact-request budget, concurrent reservation test, expiry/restart test. |
| AUTH-SEC-02 | Activation/reset performs Argon2id before token validation | 4 | Invalid token short-circuits before password work; valid token is rechecked and consumed atomically. |
| AUTH-SEC-03 | Gateway collapses auth throttles into shared proxy buckets | 1, 2 | Auth admission no longer keys end-user policy from RemoteAddr or forwarded headers; separate browser keys do not share the normal quota. |
| AUTH-SEC-04 | Expired password lockout does not enter a correct fresh failure window | 3 | Backoff expiry, fresh counting, repeat backoff, and successful-login tests. |
| AUTH-SEC-05 | Pending auth requests/codes survive security revision and revocation | 1, 5 | Password change, MFA reset, disable, and revoke between each OIDC transition fail closed. |
| AUTH-SEC-06 | Anonymous recovery can grow challenge and mail state | 1, 5 | Known-account storm coalesces to one active challenge/delivery; unknown storm creates none; retention remains bounded. |
| AUTH-SEC-07 | Challenge issuer/consumer digest can drift | 4 | One challenge validation API and cross-package password/MFA consume tests. Revalidate the current DigestToken candidate first. |
| AUTH-SEC-08 | OIDC state is not browser-bound | 1, 2 | Wrong/missing browser binding fails without consuming the legitimate state; correct binding succeeds once; replay fails. |
| AUTH-SEC-09 | Recovery-code failures permanently lock valid codes | 1, 3 | Time-bounded lock, non-extension during lock, post-expiry valid-code success, and concurrent failure tests. |
| AUTH-SEC-10 | Invalid passwords can lock a victim and revoke sessions | 3 | Wrong passwords never change lifecycle/auth_revision or call session revocation; existing sessions remain valid. |
| AUTH-SEC-11 | Public authorization state can outpace cleanup | 1, 2, 5 | Pre-allocation admission, exact concurrent cap, bounded cleanup, and restart tests for provider and API state. |
| AUTH-INT-01 | MFA redirect loses the issuer /identity prefix | 6 | Redirect equals the issuer-derived MFA URL in direct and gateway-prefixed tests. |

## Ordered work packages

### Gate 0 — establish the task-owned baseline and revalidate

Before any source edit:

1. Read AGENTS.md, docs/PLANS.md, this plan, the predecessor auth plan, the
   verification matrix, and the output contract.
2. Record branch, HEAD, git status, and focused diffs for every intended target.
   Copy the intended target files to a task-owned directory under /private/tmp
   and record SHA-256 hashes. Do not add the baseline copies to the repository.
3. Run the smallest current auth/API focused tests. Record failures as
   pre-existing or task-related from evidence; do not reset the tree.
4. Reopen every source location in the finding map. Mark a finding candidate
   already-fixed only when the current path and a focused test establish it.
   AUTH-SEC-07 is expected to require this treatment.
5. Update Current progress and Discoveries in this plan before implementation.

Gate exit: a reproducible task-owned baseline exists, no user change was
overwritten, and every finding has a current source owner.

### Work package 1 — forward-only schema and explicit policies

Add auth migration 000009 and register version 9 in
apps/auth/migrations/migrations.go. The migration must:

- extend oidc_auth_requests with nullable authenticating auth_revision,
  nonnegative MFA attempt count, fixed positive MFA attempt limit, and explicit
  invalidation timestamp/reason;
- add indexes for subject/revision lookup, outstanding request admission, and
  bounded cleanup;
- add recovery_locked_until and recovery failure-window state to mfa_factors;
- deterministically invalidate all but the newest active challenge per
  subject/purpose, then add a partial unique active-challenge index;
- add a non-PII delivery dedupe key and a partial unique index for pending mail
  states;
- add cleanup indexes for throttle and mail retention;
- preserve historical consumed, invalidated, delivered, and terminal rows; and
- use constraints that reject impossible counters, timestamps, and reasons.

Add API migration 000046 and set migrations.LatestVersion to 46. The migration
must:

- remove only expired ephemeral login states, then add a required
  browser_binding_hash to oidc_login_states without inventing a value for a
  live flow;
- add expiry/outstanding indexes;
- add a small durable OIDC login-admission table for global and browser rules;
  and
- add cleanup indexes and constraints that bound counters and key sizes.

Replace the auth limiter's one-limit-for-all-keys interface with an
operation-specific rule type containing a namespaced hashed key, window, and
limit. Evaluate all rules in one transaction, lock keys in stable order, apply
the global rule before inserting attacker-variable keys, allow at most a small
fixed rule count, and fail closed on PostgreSQL errors. Configuration owns all
limits, cooldowns, caps, and retention values; tests use short deterministic
values. Do not add a compatibility limiter.

Work-package tests:

- fresh migration and 8-to-9 / 45-to-46 migration tests;
- constraint and deterministic duplicate-normalization tests;
- concurrent multi-rule admission with no partial increments;
- global denial creates no attacker-variable bucket;
- bounded cleanup deletes at most its requested limit.

### Work package 2 — browser-bound OIDC state and pre-allocation admission

In the API:

- create a 32-byte random browser-binding secret at /auth/login when a
  well-formed current binding cookie is absent;
- use __Host-avia_login with Secure, HttpOnly, SameSite=Lax, Path=/, no Domain,
  and the login-state TTL; use avia_login with Secure=false only in the
  explicit loopback HTTP candidate;
- persist only a domain-separated hash of the binding with each login state;
- require raw state plus the binding cookie when consuming state;
- make a wrong or missing binding return the same invalid-state response
  without deleting the legitimate state;
- delete exactly one state on correct consumption and reject replay;
- refresh the short binding-cookie TTL at login so parallel tabs can share the
  same browser binding during the bounded window;
- apply durable global and browser admission plus an advisory-lock or
  row-lock-protected outstanding-state cap before inserting oidc_login_states;
  and
- remove /auth/login from applicationRateLimiter's socketPeer policy. Keep
  unrelated mutation policy unchanged unless a focused test proves it shares
  the same auth defect.

In the auth provider:

- apply durable global, registered-client, browser-binding, and
  authorization-request admission before provider.ServeHTTP can create an auth
  request;
- apply global, browser, normalized-identifier, and subject rules before
  password work, preserving unknown-account timing and response behavior;
- apply global, subject, and exact-request rules before MFA verification;
- enforce an outstanding auth-request cap transactionally inside
  CreateAuthRequest so concurrent processes cannot overrun it; and
- never use raw forwarding headers or the shared gateway peer as the normal
  user quota. Gateway configs must delete caller-supplied forwarding headers
  and write canonical values for observability only.

Attacker-variable throttle keys remain acceptable only because the global rule
limits their creation rate and cleanup limits their retention. Hash keys with
domain separation and never persist raw email, username, cookie, state, code,
or IP.

Work-package tests:

- two browser bindings receive independent normal quotas;
- a shared gateway RemoteAddr does not merge their auth quota;
- spoofed forwarding headers do not change admission identity;
- wrong/missing binding cannot consume state, correct binding later can, and
  replay cannot;
- concurrent provider/API allocation reaches the exact cap and no higher;
- expired-state and throttle cleanup is bounded and restart-safe.

### Work package 3 — credential, TOTP, and recovery-code failure semantics

Password failures:

- keep the account lifecycle state unchanged;
- never increment auth_revision and never invoke RevokeAllSessions because of
  an invalid password;
- use locked_until only as a bounded authentication backoff projection;
- do not extend an active backoff on each attacker request;
- after expiry, begin a fresh failure window and allow a later threshold to
  establish a new bounded backoff;
- clear stale failure state only after a successful authentication or an
  explicit credential/lifecycle mutation; and
- keep the in-memory candidate and PostgreSQL store behavior identical.

TOTP:

- reserve one attempt atomically on the exact oidc_auth_requests row before
  verifying either TOTP or a recovery code;
- permit the configured final attempt, then invalidate the request when that
  attempt fails;
- reject attempts after invalidation, expiry, completion, subject mismatch, or
  auth_revision drift;
- preserve the existing TOTP replay counter; and
- make restart and parallel submissions share the same budget.

Recovery codes:

- replace permanent recovery_failures lockout with a failure window and
  recovery_locked_until;
- reject during the active lock without extending it;
- reset the window after expiry and on successful single-use consumption; and
- keep code hashing, one-time deletion, and MFA reset revision semantics
  unchanged.

Work-package tests:

- wrong passwords cannot change lifecycle, auth_revision, or session rows;
- current provider/application sessions remain valid after password failures;
- backoff expires, counts restart, backoff can recur, and correct credentials
  succeed after expiry;
- exactly N MFA submissions are reserved under parallel load and N+1 fails;
- a valid TOTP below the limit completes once and replay remains rejected;
- recovery-code lock expires and a still-valid code succeeds afterward;
- memory/PostgreSQL parity and race tests pass.

### Work package 4 — cheap prevalidation and one challenge boundary

Move challenge digest, subject, purpose, state, expiry, attempt, row-lock, and
consume logic behind apps/auth/internal/challenge. Provide:

- a read-only prevalidation operation for cheap rejection before password
  work;
- a transaction-bound validation handle for final FOR UPDATE revalidation;
- transaction-bound consume and rejection methods that accept only a handle
  created by the challenge package; and
- one-active-per-subject/purpose issuance semantics.

Identity and MFA packages must stop hashing challenge tokens and stop querying
identity_challenges directly. Once no caller needs it, make the digest helper
private to the challenge package. Invitation activation must use the same
two-phase shape through an invitation-owned validator: cheap token/state/expiry
prevalidation, then password policy/history and Argon2id, then final locked
revalidation and atomic consume/mutate.

Do not hold a database row lock while running Argon2id. A second request with a
valid token may perform duplicate bounded work, but only one can pass final
revalidation and commit.

Work-package tests:

- random, malformed, wrong-subject, wrong-purpose, expired, consumed, and
  locked credentials fail before password policy/history/hash work;
- weak password plus invalid token returns the generic invalid-token class,
  proving token prevalidation is first;
- valid token plus invalid password does not consume the token;
- two valid concurrent submissions perform at most one mutation;
- password and MFA recovery use the same challenge boundary and digest;
- replay and purpose substitution fail.

### Work package 5 — revision-bound pending auth, recovery coalescing, cleanup

Revision-bound OIDC state:

- stage subject and the exact current auth_revision together after password
  authentication;
- require that revision to match an active, verified account before MFA,
  Authorize, authorization-code lookup, code exchange, and access-token
  creation;
- make a request without a captured revision ineligible once a subject has
  been staged;
- have RevokeAllSessions delete or explicitly invalidate pending requests and
  their cascade-linked codes for the subject in the same cleanup transaction;
  and
- retain revision checks as the race-safe control if cleanup races with code
  creation or exchange.

Recovery coordination:

- add one coordinator that begins a transaction, applies recovery admission,
  looks up the normalized email, enforces account eligibility and cooldown,
  creates one active challenge, and transactionally enqueues one encrypted
  deduplicated delivery;
- return the same accepted response for unknown, malformed, throttled,
  ineligible, coalesced, and successful requests;
- create no challenge or mail row for unknown accounts;
- do not enqueue a new link while an older delivery for the same
  subject/purpose is queued, leased, or retryable;
- invalidate a challenge in the same transaction if its mail row cannot be
  prepared; and
- never reuse or recover a raw token from storage.

Maintenance:

- extend the auth maintenance loop with bounded cleanup for expired provider
  requests/codes, challenges, stale throttle buckets, and retained
  delivered/terminal mail;
- make challenge cleanup bounded instead of an unbounded delete;
- clean API login states and admission buckets in bounded batches;
- log only class-level counts and errors; and
- verify that dependency failure cannot become a tight retry loop.

Work-package tests:

- password change, MFA reset, disable/suspend, and explicit revoke at every
  password-to-MFA-to-code-to-token boundary invalidates the stale flow;
- cleanup races cannot produce a token from an old revision;
- 100 concurrent known-account recovery requests yield at most one active
  challenge and one pending delivery for the cooldown;
- an equivalent unknown-account storm yields no subject/challenge/mail rows and
  the same external response shape;
- delivered/terminal retention and every cleanup batch respect the limit;
- mail lease/retry and challenge state remain coherent across restart.

### Work package 6 — issuer routing, redacted observability, focused integration

- Replace the root-relative MFA redirect with
  providerEndpoint(runtime.issuer, "mfa") plus the encoded request ID.
- Keep login, callback, logout, activation, password recovery, and MFA recovery
  paths issuer-derived.
- Return Retry-After for explicit 429 responses without echoing a bucket key.
- Add redacted counters/log fields for operation class, admitted/denied reason,
  MFA exhaustion, revision mismatch, recovery coalescing, and cleanup count.
- Add gateway tests that /identity/mfa is reachable while public port 8081
  administration remains unreachable.
- Update qualification fixtures to carry the browser-binding cookie without
  weakening Secure cookie behavior outside the explicit loopback HTTP profile.

Gate exit: focused direct-runtime and gateway-prefixed login, MFA, recovery,
logout, and stale-revision scenarios pass.

### Work package 7 — full verification and final security review

Run formatting first, then the narrowest tests, then broader gates. Do not
change a limit, assertion, privacy response, route, or security invariant just
to manufacture a pass.

Focused commands:

~~~bash
go -C apps/auth test -count=1 ./internal/throttle ./internal/challenge ./internal/identity ./internal/mfa ./internal/mail ./internal/provider ./migrations
go -C apps/api test -count=1 ./internal/httpapi ./internal/platform/session ./migrations
go -C apps/auth test -race -count=1 ./internal/throttle ./internal/challenge ./internal/identity ./internal/mfa ./internal/mail ./internal/provider
go -C apps/api test -race -count=1 ./internal/httpapi ./internal/platform/session
~~~

Required repository lane:

~~~bash
go -C apps/auth test -count=1 ./...
go -C apps/auth test -race -count=1 ./...
go -C apps/auth vet ./...
go -C apps/auth mod verify
go -C apps/api test -count=1 ./internal/identity ./internal/platform/session ./internal/httpapi ./cmd/preprod-canonical-demo-identity-loader
go -C apps/api test -count=1 ./...
go -C apps/api vet ./...
go -C apps/api mod verify
node --test tests/local-compose-policy.test.mjs tests/preprod-data-boundary.test.mjs
docker compose --file deploy/local/compose.yaml config --quiet
./scripts/test-auth-candidate-postgres.sh
./scripts/test-auth-candidate-runtime.sh
./scripts/test-auth-candidate-mailpit-outbox.sh
./scripts/test-preprod-identity-lifecycle.sh
./scripts/test-canonical-preprod-fault-restart.sh
node tests/harness-docs-smoke.test.js
git diff --check
~~~

Expected observations:

- all focused abuse, concurrency, migration, and replay tests pass;
- the provider/API public path and private administration remain split;
- HTTPS and explicit loopback HTTP login/callback/MFA/logout work;
- public port 8081 denial, dependency loss, restart, and task-owned cleanup pass;
- no new raw identity, browser-binding, state, code, token, or throttle value is
  present in logs or database fixtures;
- unrelated pre-existing failures are reported literally, never repaired
  outside scope; and
- the result remains candidate-only and release pending.

After functional verification:

1. Produce a task-owned patch by comparing the Gate 0 file copies with current
   files. Review every changed source-like file and test.
2. Run codex-security:security-diff-scan against the current working-tree diff
   from HEAD. Use the Gate 0 receipt to distinguish this task from pre-existing
   dirty-tree changes; disclose that scope limitation.
3. Reproduce and fix every confirmed task-introduced Critical, High, or Medium
   issue, rerun affected gates, and repeat the final review on the new target.
4. Write a finding disposition table with fixed, already-fixed-and-preserved,
   blocked, or not-fixed. Do not label a finding fixed from source inspection
   alone.

## Acceptance criteria

The plan may move to ready-for-verification only when:

- every AUTH-SEC and AUTH-INT row has an implemented control and passing mapped
  regression test, or a literal blocked disposition approved by the user;
- invalid password traffic cannot alter lifecycle state, auth_revision, or
  existing sessions;
- TOTP, recovery-code, authorization, login-state, challenge, mail, and
  throttle limits remain correct under concurrency and restart;
- invalid activation/reset credentials take the cheap path before password
  history and Argon2id work;
- OIDC state is bound to the initiating browser and remains one-use;
- all pending authorization transitions enforce current auth_revision;
- recovery is enumeration-neutral and durable state is capped and retained for
  a bounded period;
- every public route remains issuer/gateway correct and private admin remains
  private;
- migration tests cover fresh and predecessor databases;
- the focused, repository-lane, lifecycle, fault/restart, cleanup, diff-check,
  and final security review results are recorded literally;
- plan and index agree; and
- no remote, release, or production-ready claim is made.

## Rollout, rollback, idempotence, and recovery

Rollout is one local candidate rebuild:

1. stop only task-owned candidate services;
2. back up or fingerprint the disposable auth and API databases if needed for
   migration evidence;
3. apply API 000046 and auth 000009 through their existing ledgers;
4. start the same topology and run focused smoke before full lifecycle gates;
5. observe denial/cleanup counters without recording sensitive values; and
6. clean task-owned services and temporary baseline artifacts after evidence is
   captured.

Migrations are forward-only and ledger-idempotent. Recovery issuance,
admission, attempt reservation, revocation cleanup, and maintenance must be
transactionally replay-safe. Re-running cleanup may remove more eligible rows
but may never exceed one configured batch.

Do not roll back by restoring the vulnerable runtime or weakening schema
constraints. If a migration or runtime cannot be repaired safely, stop the
candidate and recreate the disposable namespace from the predecessor schema,
then apply a corrected forward migration. Repository edits remain recoverable
from version control, but no reset, checkout, branch operation, commit, or push
is authorized.

## Risks and dependencies

- PostgreSQL admission adds work to public auth paths. Keep transactions short,
  indexes selective, rule counts fixed, and Argon2id outside row locks.
- A global emergency ceiling can still deny new login starts during a sustained
  attack. It must be high enough for qualification traffic and paired with
  browser/identifier/subject rules so it is not the ordinary user quota.
- Browser-binding cookies share the public origin. Strict cookie attributes,
  hashed persistence, short TTL, and no authority semantics are mandatory.
- Two-database OIDC coordination cannot be one transaction. Browser binding
  protects the API state; provider auth_revision binding and one-time code
  semantics protect the provider transition.
- Recovery mail/challenge atomicity crosses package ownership but not database
  ownership. Use transaction-aware package methods rather than raw SQL in the
  HTTP handler.
- The auth replacement work was committed as aa9f63f while this plan was being
  written. Remaining unrelated AviaCore edits and any later user changes still
  require baseline receipts and narrow patch review.
- Existing AviaCore registry drift is unrelated and must not be repaired by
  this plan.

## Current progress

- [x] Source audit findings converted to stable local identifiers.
- [x] Current source owners and principal failure modes re-inspected.
- [x] Selected design, work ordering, tests, and acceptance criteria planned.
- [x] Implementation handoff prompt written.
- [x] Gate 0 task-owned baseline and current focused test results recorded.
- [x] Work packages 1 through 6 implemented; focused source and contract tests pass locally.
- [x] Work package 7 functional, repository-lane, connected-runtime, fault/restart, and cleanup verification completed locally.
- [x] Task-owned diff review and pre-remediation Codex Security diff scan completed; the sealed snapshot reported 3 Medium and 6 Low findings with no Critical or High finding.
- [x] Confirmed Medium findings were remediated locally: API/provider bootstrap outstanding partitions and bounded throttle-cleanup throughput; affected focused, race, full-package, connected-runtime, and fault/restart checks were rerun.
- [x] Post-remediation Codex Security diff scan completed and sealed with 6 Low residuals, no Medium/High/Critical findings, complete coverage, and zero deferred rows.
- [x] AUTH-SEC-06 follow-up completed: delivered recovery dedupe is gated by a usable challenge, expired active rows are invalidated transactionally before replacement, and the connected PostgreSQL replacement regression passed.
- [x] AUTH-SEC-11 follow-up completed: API/provider bootstrap partitions and 100-state caps pass 20/100/101 burst, consume/delete release, and forged-cookie regression coverage; connected API and PostgreSQL harnesses passed.
- [x] Recovery maintenance follow-up completed: challenge and retained-delivery cleanup now drain two bounded 1,000-row batches per minute, exceeding the two-purpose 1,200-per-minute recovery admission ceiling without introducing an unbounded delete. Focused challenge/mail/command tests and vet passed; the existing provider listener test remains sandbox-blocked.
- [x] Final follow-up fixes completed: absolute recovery URLs preserve issuer paths; durable and memory provider stores enforce one authorization code per request and bounded expired-code cleanup; API and throttle cleanup recheck refreshed rows; focused and connected checks were rerun.
- [x] Fresh current-tree security diff scan `ed7d4d6c-c227-47ce-9843-778b601a2611` sealed with complete coverage, zero deferred rows, three Low residuals, and no Medium/High/Critical findings.

## Finding disposition

| ID | Disposition | Evidence and remaining boundary |
|---|---|---|
| AUTH-SEC-01 | fixed — verified locally | Durable exact-request MFA/TOTP attempt reservation, replay rejection, and race coverage pass. |
| AUTH-SEC-02 | fixed — verified locally | Challenge prevalidation rejects invalid credentials before Argon2/password policy work; valid-token mutation is transaction-bound. |
| AUTH-SEC-03 | fixed — verified locally | Gateway forwarding headers are canonicalized and auth quotas use operation-specific server-side keys; gateway/Compose contracts pass. |
| AUTH-SEC-04 | fixed — verified locally | Password backoff expiry and fresh failure-window behavior pass identity focused, race, and connected PostgreSQL checks. |
| AUTH-SEC-05 | fixed — verified locally | Pending provider requests/codes are revision-bound and invalidated across password, MFA, authority, and session revocation transitions. |
| AUTH-SEC-06 | fixed — verified locally | Delivered recovery dedupe suppresses only while the challenge remains usable; expired active challenges are invalidated in the same transaction before replacement. Connected PostgreSQL outbox/challenge regression passed. |
| AUTH-SEC-07 | fixed — verified locally and preserved | The challenge package owns digest, expiry, attempt, row-lock, and consume/reject validation; cross-package tests pass. |
| AUTH-SEC-08 | fixed — verified locally | API OIDC state stores a binding hash, rejects wrong/missing bindings without consuming state, and accepts one correct consume. |
| AUTH-SEC-09 | fixed — verified locally | Recovery-code failure lock is time-bounded and non-extending while locked; post-expiry success and race checks pass. |
| AUTH-SEC-10 | fixed — verified locally | Invalid passwords do not change lifecycle/auth_revision or revoke existing sessions; identity/session tests pass. |
| AUTH-SEC-11 | fixed — verified locally | API and provider admission reserve 20 bootstrap states within a 100-state outstanding cap, admit bound starts while bootstrap is saturated, reject the 101st, and release capacity on consume/delete. Browser bindings are server-signed and forged cookies remain bootstrap traffic. Focused/race, connected PostgreSQL, and connected API burst tests passed. |
| AUTH-INT-01 | fixed — verified locally | MFA redirect derives the issuer-prefixed path; direct and gateway-prefixed route tests pass. |

## Decisions and discoveries

- Decision: use durable operation-specific PostgreSQL admission already
  available to the auth boundary; do not add Redis or a new edge service.
- Decision: do not trust forwarded headers or shared gateway peer identity for
  auth quotas. Use server-issued browser binding with global and
  identifier/subject/request controls.
- Decision: password failures are authentication backoff, not lifecycle or
  session revocation events.
- Decision: preserve enumeration neutrality even when recovery is throttled or
  coalesced.
- Decision: centralize challenge validation instead of retaining duplicated raw
  SQL around a shared digest helper.
- Implementation evidence: challenge issuance, prevalidation, final locked
  revalidation, consume, rejection, and cleanup now have one package-owned
  boundary; identity and MFA callers no longer hash challenge tokens or query
  challenge rows directly.
- Implementation evidence: API login state is bound to a short-lived browser
  secret, durable global/browser admission, and an outstanding-state cap; the
  API maintenance loop cleans states and admission buckets in bounded batches.
- Implementation evidence: auth maintenance now cleans provider state,
  challenges, exact-window throttle buckets, and retained terminal mail in
  bounded batches, with class-only cleanup logs.
- Implementation evidence: recovery issuance uses a transaction-aware
  challenge/mail coordinator, non-PII dedupe keys, and coalesces pending or
  delivered recovery work while replacing stale terminal-delivery challenges.
- Follow-up evidence (2026-08-13): recovery dedupe now checks delivered
  delivery state together with current challenge usability; every active
  challenge row, including an expired row still covered by the partial unique
  index, is invalidated before replacement in the same transaction. The
  connected PostgreSQL OIDC-state test creates one delivered and one fresh
  queued delivery after expiry.
- Post-scan remediation evidence (2026-08-13): API login-state admission now
  marks bootstrap rows and reserves the majority of the outstanding pool for
  already-bound starts; provider memory and PostgreSQL admission apply the same
  bounded anonymous partition. Auth throttle maintenance now drains two bounded
  10,000-row batches per minute, above the configured attacker-variable
  admission rate.
- Follow-up fairness evidence (2026-08-13): API login bindings are HMAC-signed
  with the configured first-party OIDC client secret and the provider accepts
  only valid signatures. Connected tests cover forged-cookie rejection, 20
  bootstrap states, bound admission at the reserved capacity, 101st denial,
  and consume/delete capacity release.
- Recovery cleanup follow-up evidence (2026-08-13): auth maintenance drains
  two bounded 1,000-row challenge batches and two bounded 1,000-row retained
  delivery batches per minute. This is above the two-purpose recovery
  admission ceiling of 1,200 accepted requests per minute while preserving the
  configured bounded-delete rule. Challenge, mail, auth-command tests, and
  vet passed locally; the existing provider discovery listener test was blocked
  by sandbox IPv6 bind permissions.
- Security-diff evidence (2026-08-13): pre-remediation scan
  `445b98f1-9aca-415b-96db-3e32b8773bfb` sealed with 9 reportable findings
  (3 Medium, 6 Low); post-remediation scan
  `8878490b-f785-4c9d-aa07-97878d7daf9a` sealed with 6 Low residuals, no
  Medium/High/Critical findings, complete coverage, and zero deferred rows.
  The post-remediation report is recorded at
  `/private/var/folders/nv/m6v_mydj3s38m58q4xrp4k1r0000gn/T/codex-security-scans-7AHrXd/aviaSurveil360/aa9f63f2bef60866b6d53ba78fa14f7de6f561c3_20260813T094621Z_uw0tv9u9/report.md`.
- Final current-tree security-diff evidence (2026-08-13): scan
  `ed7d4d6c-c227-47ce-9843-778b601a2611` is sealed with complete coverage,
  zero deferred rows, three Low findings, and no Medium/High/Critical findings.
  The readable report is recorded at
  `/private/var/folders/nv/m6v_mydj3s38m58q4xrp4k1r0000gn/T/codex-security-scans-zsEs0d/aviaSurveil360/aa9f63f2bef60866b6d53ba78fa14f7de6f561c3_20260813T181115Z_uum6xpji/report.md`.
  Remaining Low risks are conditional subject-throttle response discrepancy,
  shared missing-browser password quota, and shared missing-browser provider
  quota. These are candidate-only follow-ups, not release approval or
  production evidence.
- Gate 0 evidence (2026-08-13): branch `main`, HEAD
  `aa9f63f2bef60866b6d53ba78fa14f7de6f561c3`, and the initial tracked diff are
  recorded under `/private/tmp/avia-auth-security-gate0.3JhCRS`. Existing
  user-owned changes are limited to
  `docs/product-specs/data-and-rules/aviacore-data-feed-coverage.json`,
  `tests/aviacore-data-feed-coverage.test.mjs`, and this untracked plan; the
  auth/API/gateway target baseline had no diff. Baseline source copies and
  SHA-256 hashes are recorded in `baseline-sha256.txt`.
- Gate 0 focused evidence: API
  `go -C apps/api test -count=1 ./internal/httpapi ./internal/platform/session ./migrations`
  is `verified locally`. Auth throttle, challenge, identity, MFA, mail, and
  migrations passed in
  `go -C apps/auth test -count=1 ./internal/throttle ./internal/challenge ./internal/identity ./internal/mfa ./internal/mail ./internal/provider ./migrations`
  except the provider package could not bind the sandbox IPv6 loopback
  listener. The isolated rerun
  `go -C apps/auth test -count=1 ./internal/provider` with local loopback
  socket access is `verified locally`; the first failure is environmental and
  not a source regression.
- Gate 0 source revalidation findings are now addressed by the WP1–WP6
  implementation. The original receipt remains the basis for separating the
  unrelated AviaCore changes from this task-owned diff.
- Focused implementation evidence (2026-08-13): auth targeted packages,
  migration contracts, API targeted packages, issuer-prefix provider tests,
  auth/API race suites, `go vet`, `go mod verify`, gateway Node contract tests,
  Compose config validation, harness-docs smoke, and `git diff --check` are
  `verified locally`.
- Repository-lane evidence (2026-08-13): `go -C apps/auth test -count=1 ./...`,
  `go -C apps/auth test -race -count=1 ./...`, both auth/API focused race
  suites, `go vet`, and `go mod verify` are `verified locally`. API
  `go -C apps/api test -count=1 ./...` passed all non-integration packages but
  is `blocked` at the repository integration package by the unavailable
  default disposable PostgreSQL at `127.0.0.1:55432`; this is an existing
  environment lane, not a source assertion failure. The task-owned
  `./scripts/test-preprod-identity-lifecycle.sh session-authority` reran the
  API session, identity, administration, httpapi, and integration packages
  against disposable PostgreSQL/Mailpit/object storage and is `verified
  locally`.
- Connected execution evidence (2026-08-13):
  `./scripts/test-auth-candidate-postgres.sh`,
  `./scripts/test-auth-candidate-runtime.sh`, and the corrected
  `./scripts/test-auth-candidate-mailpit-outbox.sh` are `verified locally`.
  The Mailpit harness now invokes the pinned image's `/mailpit readyz`
  entrypoint; both STARTTLS delivery and outbox retry tests passed. The
  canonical fault/restart harness is `verified locally`: the lifecycle test
  passed 1/1, role panels passed 10/10, dependency-loss injection recovered,
  worker crash restart recovered, and task-owned cleanup asserted zero
  residue. External, release, deployment, and production evidence remain
  `not run`; the result remains candidate-only and release pending.

## Outcome notes

Gate 0 complete; implementation, final local security review, commit, and push
are complete. The unrelated AviaCore edits listed above remain preserved and
unstaged. WP1–WP7 implementation and the AUTH-SEC-06/11 follow-up verification
are complete. The current-tree security diff is sealed with three Low residual
risks and no Medium/High/Critical findings. No deployment, production traffic,
release approval, or production-ready claim was made; the result remains
candidate-only and release pending.

Planning-document verification on 2026-08-13:

| Command | Result |
|---|---|
| git diff --check | verified locally |
| Required harness-reference rg check from the docs-only verification lane | verified locally |
| node tests/harness-docs-smoke.test.js | verified locally |

## Execution Prompt

Target model: GPT-5.6 Luna Max Speed when that model is available; otherwise
use GPT-5.6 Sol at max reasoning. You are the sole implementation writer.

Work in /Users/marlonjd/Developer/monorepo/aviaSurveil360 on the current branch.
Read AGENTS.md, docs/PLANS.md,
docs/exec-plans/active/2026-08-13-first-party-auth-security-remediation-plan.md,
the predecessor first-party auth plan, the verification matrix, and the output
contract completely before editing. Use codex-security:fix-finding for the
implementation and codex-security:security-diff-scan only after functional
verification.

Implement the selected design in the plan exactly in work-package order. Do not
create or switch branches. Do not stage, commit, push, deploy, touch remote
systems, or modify production/external data. Treat every current or later
working-tree change as user-owned. Before editing, create the Gate 0 task
baseline under /private/tmp with status, focused diffs, file copies, and
SHA-256 hashes. Never use git reset, git checkout, or broad cleanup. Preserve
all unrelated changes and stop if an overlapping user edit cannot be merged
safely.

First revalidate AUTH-SEC-01 through AUTH-SEC-11 and AUTH-INT-01 against current
source. If a candidate fix already exists, preserve it and add the missing
owned boundary and regression test; do not duplicate or revert it. Update the
plan's progress, discoveries, decisions, and exact evidence at each material
transition.

Then implement:

- auth migration 000009 and API migration 000046 with fresh/predecessor
  migration tests;
- browser-bound one-use OIDC state and durable API/provider pre-allocation
  admission;
- multi-rule durable throttling that does not treat proxy RemoteAddr or
  forwarding headers as end-user identity;
- password failure semantics that never change lifecycle, auth_revision, or
  existing sessions;
- exact-request TOTP attempt reservations and time-bounded recovery-code
  backoff;
- cheap activation/recovery token prevalidation followed by final atomic
  revalidation/consume, with all challenge validation owned by the challenge
  package;
- auth_revision binding across password, MFA, authorization code, exchange,
  token creation, and revocation;
- enumeration-neutral, coalesced, transactionally consistent recovery
  challenge/mail issuance and bounded retention cleanup; and
- issuer-prefixed MFA routing and redacted observability.

Keep changes minimal and repository-native. Do not add a compatibility path,
fallback provider, external cache, or speculative abstraction. Do not hold
database locks during Argon2id. Do not weaken PKCE, nonce, cookie flags,
private-admin isolation, authority checks, privacy responses, or test
assertions. Never log raw email, username, subject, browser binding, state,
authorization code, recovery token, MFA code, password, or throttle key.

Run each work package's focused tests before continuing, then run every
applicable command in Work package 7. Use an isolated browser profile for
browser tests and clean task-owned browser, Vite, test, and Compose processes.
Record exact commands and literal results as verified locally, not run, or
blocked. Do not claim production readiness.

After tests, compare the Gate 0 copies with final files to obtain the task-owned
patch. Review every changed source-like file, then run the security diff scan
over the current working-tree diff from HEAD. Use the baseline receipt to
separate task changes from pre-existing dirty-tree changes. Fix confirmed
task-introduced Critical, High, or Medium findings, rerun affected tests, and
repeat the final review. Finish with a finding disposition table, changed-file
summary, exact verification results, remaining risks, cleanup status, and the
literal candidate-only / release-pending boundary.
