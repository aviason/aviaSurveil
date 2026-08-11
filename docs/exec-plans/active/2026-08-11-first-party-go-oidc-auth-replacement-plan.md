# First-Party Go OIDC Authentication Replacement ExecPlan

This ExecPlan is a living document. Keep `Progress`, `Decision Log`,
`Discoveries`, and `Outcome` synchronized with actual work. Follow
[`docs/PLANS.md`](../../PLANS.md), the repository
[`AGENTS.md`](../../../AGENTS.md), and the literal evidence vocabulary in the
[`output contract`](../../agent-harness/output-contract.md).

## Status

- Plan status: `active`
- Current result: Tasks 0–10 are `verified locally` for their isolated,
  candidate-only implementation and local gates; Task 10 records `NO-GO` for
  release because independent review and native ARM64 capacity evidence are
  `not run`. Option A (`zitadel/oidc` v3.47.5, `fb9fbfe`) was authorized on
  2026-08-11.
- Current provider: Keycloak remains required and must not be removed or
  weakened during development.
- Product status: `candidate-only`; release is `release pending`;
  `production-ready: not established`.
- Next concrete todo: classify the remaining native ARM64 mixed-load,
  rollback-traffic, fresh SBOM/license/vulnerability/provenance, and external
  review gates without changing the Keycloak serving or rollback baseline.
  PostgreSQL and verified TLS Mailpit topology. Keycloak remains retained; no
  production or external-state action is authorized by this plan.

## Objective

Replace the Keycloak runtime, after complete local and later separately
authorized release qualification, with a small separately privileged Go OIDC
provider built from selected first-party authentication concepts and tests,
with a maintained authorization-server library. Preserve AviaSurveil360's existing
same-origin BFF, encrypted browser sessions, CSRF controls, immutable subject
identity, exact organization isolation, application-owned role authorization,
MFA, lifecycle revocation, recovery, redacted audit, and rollback behavior.

## User-Visible Outcome

Users continue to sign in and out through the same AviaSurveil360 origin. They
receive standards-based OIDC login, TOTP MFA, recovery codes, verified email,
password reset, session management, and localized identity messages in English,
Turkish, French, and Portuguese. Application screens continue to expose only
the exact role and organization authorized by server-owned membership state.

The long-term runtime replaces the Keycloak JVM with one bounded non-root ARM64
Go service. No user should be able to distinguish the provider replacement by
weaker authentication, changed subject identity, broader authorization, lost
history, or an unsafe browser-token contract.

## Scope

This plan owns:

- integrity-retained intake, comparison, and security mapping for both
  owner-provided source collections;
- the first-party Go provider architecture and implementation;
- canonical identifiers, passwords, account state, provider sessions, OIDC,
  signing-key rotation, TOTP, recovery codes, verification/reset, SMTP-backed
  identity messages, security audit, and provider administration;
- provider-neutral adapters in the current API while preserving application
  membership and session boundaries;
- local PostgreSQL, Mailpit, Compose, gateway, browser, recovery, image, and
  native ARM64 verification;
- synthetic subject migration and explicit rollback qualification; and
- later removal of Keycloak only after all gates and a separate cutover and
  retention/destruction authorization.

## Explicit Exclusions

- No public self-registration.
- No automatic federation, social login, LDAP, SAML, dynamic client
  registration, or third-party identity brokerage.
- No custom cryptographic algorithms, JWT parser, authorization-code protocol,
  or hand-written OIDC replacement when a maintained library owns the boundary.
- No direct application of the imported evaluation migration.
- No direct import of the Kindred DynamoDB repository, full-row user updates,
  AppSync/API Gateway wiring, or application/mobile domain code.
- No import of the export's `app.users`, `staff_permissions`, city, profile, or
  application authorization model.
- No ambiguous Keycloak/native dual-issuer fallback.
- No real password, TOTP seed, recovery code, token, private key, production
  identity, or customer data in repository evidence or local fixtures.
- No AWS, Cloudflare, SMTP-provider, DNS, certificate, secret, RDS, production
  migration, identity load, deployment, traffic, cutover, rollback, retention,
  destruction, or residue action without separate exact authorization.
- WebAuthn/passkeys are not a first-cutover commitment until the owner records
  that separate product decision; TOTP and hashed recovery codes are mandatory.

## Owner Decisions And Assumptions

- On 2026-08-11 the owner stated that the exported application code is entirely
  their own and may be copied, modified, and integrated. Third-party dependency
  licenses still apply.
- Both retained source collections are candidate evidence. Neither is a
  deployable OIDC provider or permission to enable imported runtime code.
- The selected direction is a separate Go provider, not identity code embedded
  into the ordinary API. This keeps credentials, factors, signing keys, and
  provider sessions outside the normal API/worker blast radius.
- Keycloak remains the serving and rollback provider until cutover acceptance.
- The existing API remains an OIDC relying party and continues to own encrypted
  browser sessions and application authorization.
- The imported source is reference input. Reuse means reviewed adaptation, not
  bulk copy into a runtime build.

## Repository Orientation And Evidence

- Raw intake receipt:
  [`docs/security/auth-replacement/evidence/2026-08-11-first-party-auth-export/IMPORT_RECEIPT.md`](../../security/auth-replacement/evidence/2026-08-11-first-party-auth-export/IMPORT_RECEIPT.md)
- Retained exact ZIP SHA-256:
  `7fa982300440cb3e79d28bc0f7f22ebb59124bc9c125dededb22dea306fc7fb7`
- Second raw intake receipt:
  [`docs/security/auth-replacement/evidence/2026-08-11-kindred-auth-export/IMPORT_RECEIPT.md`](../../security/auth-replacement/evidence/2026-08-11-kindred-auth-export/IMPORT_RECEIPT.md)
- Second retained exact ZIP SHA-256:
  `5de123c9bd8a711889e85b1876329540dee49423f64568c0a4bade4b5a4ff79b`
- Source selection matrix:
  [`source-comparison.md`](../../security/auth-replacement/hardening/source-comparison.md)
- Task 1 OIDC comparison and disposable evidence:
  [`RESULTS.md`](../../security/auth-replacement/evidence/2026-08-11-oidc-library-spike/RESULTS.md)
- Task 1 provider-neutral contracts:
  [`CONTRACTS.md`](../../security/auth-replacement/evidence/2026-08-11-oidc-library-spike/CONTRACTS.md)
- Imported security review:
  [`SECURITY_REVIEW.md`](../../security/auth-replacement/evidence/2026-08-11-first-party-auth-export/source/auth-export/SECURITY_REVIEW.md)
- Hardening portfolio:
  [`hardening.md`](../../security/auth-replacement/hardening/hardening.md)
- Selected implementation handoff:
  [`separate-go-oidc-provider.md`](../../security/auth-replacement/hardening/implementation/separate-go-oidc-provider.md)
- Current OIDC relying party: `apps/api/internal/identity/oidc_remote.go`
- Current provider administration: `apps/api/internal/identity/keycloak_admin.go`
- Current application sessions: `apps/api/internal/platform/session/`
- Current BFF endpoints: `apps/api/internal/httpapi/auth.go`
- Current browser client: `apps/web/src/auth/`
- Current identity contract:
  `docs/product-specs/data-and-rules/PREPROD_IDENTITY_AND_DATA_PROFILE.md`
- Current recovery runbook: `docs/operations/runbooks/IDENTITY_MFA.md`

## Target Trust Boundary

```mermaid
flowchart LR
    Browser["Browser"] --> Gateway["Gateway"]
    Gateway --> API["Go API and same-origin BFF"]
    Gateway --> Auth["First-party Go OIDC provider"]
    API --> Auth
    API --> AppDB["Application database\nsessions and memberships"]
    Auth --> AuthDB["Identity database\ncredentials, factors and OIDC state"]
    Auth --> Keys["Scoped signing and MFA secrets"]
    Auth --> SMTP["Verified TLS SMTP"]
```

The provider establishes authentication and one immutable subject. The API
remains the application authorization authority. Provider role/organization
observations, if retained for reconciliation, must match the exact current
server-owned desired membership; they never bypass database-scoped policy.

## Binding Security Invariants

1. Every protected request resolves one immutable subject and one exact active
   application membership before domain access.
2. CAA roles belong only to `CAA`; an Auditee belongs to exactly one non-CAA
   organization; invalid or multiple authority fails closed.
3. Public self-registration is disabled. Provisioning is administrative,
   revisioned, audited, and invitation-based.
4. Password verification is enumeration-resistant and bounded by input,
   per-source, per-principal, global rate, and Argon2id concurrency limits.
5. Password change, reset, refresh reuse, suspension, lock, deactivation, role
   change, organization transfer, and MFA reset revoke every affected provider
   and application session.
6. OIDC uses exact issuer, registered client, exact redirect URI,
   Authorization Code plus PKCE, state, nonce, short-lived codes, ID tokens,
   algorithm allowlists, stable `kid`, and controlled key rotation.
7. Keycloak and replacement issuers, keys, clients, databases, cookies, and
   sessions are distinct. Verification never falls back between them.
8. TOTP secrets are protected, time-window replay is rejected, and recovery
   codes are random, hashed, single-use, regenerated only after step-up, and
   never logged.
9. Verification and reset challenges are random, hashed at rest, single-use,
   expiring, attempt-bounded, and delivered only over verified TLS/STARTTLS.
10. Security audit is append-only, bounded, redacted, and explicit about which
    failures block the operation versus alert asynchronously.
11. Normal API and worker code cannot access password hashes, raw factor
    secrets, reset values, issuer private keys, or identity SMTP credentials.
12. No implementation result is called production-ready without current
    protocol, dependency, independent security, recovery, and ARM64 evidence.

## Ordered Tasks

### Task 0 — Preserve And Classify The Source Input

Status: `verified locally`.

- Verify both owner-provided ZIP digests.
- Reject traversal, absolute, duplicate, corrupt, or symlink entries.
- Retain the exact archive and extracted bytes outside runtime builds.
- Record the owner code-ownership decision and candidate-only boundary.
- Inspect both exports' reviews, capability matrices, persistence, protocol
  claims, dependencies, server/client primitives, and current Avia identity
  boundaries.
- Select the separate Go provider architecture and record alternatives.

Acceptance: both retained archives match their supplied digests; extracted
bytes match the inspected trees; no imported runtime is enabled.

### Task 1 — Freeze Protocol And Provider-Neutral Contracts

Status: `verified locally`; Option A (`zitadel/oidc` v3.47.5, `fb9fbfe`) was
authorized by the owner on 2026-08-11.

- Compare maintained Go authorization-server libraries using primary source,
  current maintenance, license, protocol ownership, testability, dependency,
  and ARM64 criteria.
- Build a disposable interoperability spike for discovery, Authorization Code
  plus PKCE, exact redirects, state, nonce, token exchange, ID-token
  verification, JWKS, key rotation overlap, and RP-initiated logout.
- Define provider-neutral interfaces for account provisioning, authority
  observation, lock/suspend/deactivate, required actions, MFA state, session
  revocation, and recovery without preserving Keycloak-shaped APIs.
- Freeze browser, issuer, claim, admin, audit, failure, readiness, and secret
  contracts. Map `AS360-AUTH-001` through `AS360-AUTH-030` to tests. The second
  collection adds conditional refresh, subject-bound logout, password-change
  revocation, security-state concurrency, verification-gate, cross-transport
  throttling, long-lived-session, key-rotation, HMAC-drift, client-refresh,
  OTP-attempt, and expensive-input regression contracts.

Acceptance: one maintained library is selected with evidence; the current API
can consume the spike without insecure issuer handling; the contract review has
no unresolved High design blocker.

Task 1 evidence gate (2026-08-11): the comparison, disposable spikes, and
provider-neutral contracts are `verified locally`. The owner authorized Option
A (`zitadel/oidc` v3.47.5, `fb9fbfe`) on 2026-08-11, so the one-library
selection acceptance is recorded. Only that option may be used for Task 2 and
later implementation; Keycloak remains active and no production authority is
implied.

### Task 2 — Scaffold The Isolated Provider

Status: `verified locally`.

- Create `apps/auth/` as a separate Go module or repository-consistent module
  surface after checking current module conventions.
- Add strict typed configuration, startup validation, liveness, readiness,
  redacted telemetry, non-root ARM64 image, no-new-privileges, dropped
  capabilities, bounded PIDs/resources/logs, and no Docker socket.
- Give the provider a separate database role/schema or logical database,
  signing/MFA secret set, and SMTP identity. Normal API and worker roles receive
  none of those secrets.
- Upgrade imported dependency concepts to current reviewed versions and retain
  required notices.

Acceptance: the empty provider starts only with complete safe configuration,
fails closed on missing/placeholder/weak key material, and is not routed by a
normal profile.

### Task 3 — Identity, Password, Account-State, And Abuse Controls

Status: `verified locally`.

- Implement immutable opaque subjects and one canonical type-aware identifier
  table with verified email state and cross-field uniqueness.
- Implement invited, active, disabled, suspended, locked,
  deletion-pending, and deleted states as required by product policy; every
  login, refresh, token, session, and admin path checks state fail closed.
- Adapt Argon2id only after bounding encoded parameters, password size,
  concurrency, and admission. Add policy, current-password reuse denial,
  history, compromised-password decision, and rehash-on-success.
- Use a fixed dummy hash and generic outcomes for unknown identifiers. Add edge
  and application throttling on every public auth transport without creating
  attacker-triggered permanent lockout. Trust forwarded client identity only
  from the configured gateway and fail closed when sensitive-operation limiter
  state cannot be established.
- Remove public registration. Add revisioned administrative invitation and
  activation.

Acceptance: focused unit/race/PostgreSQL tests close the mapped password,
registration, identifier, account-state, throttling, and authorization defects.

### Task 4 — Provider Sessions And Refresh Families

Status: `verified locally`.

- Implement cryptographically random provider sessions and refresh families,
  hashes at rest, row-lock rotation, reuse detection, absolute and idle expiry,
  issuance auth version, bounded session counts, and durable cleanup.
- Make every security-state mutation conditional or revision checked so stale
  refresh, password, lockout, lifecycle, or factor writes cannot restore prior
  authority.
- Make password/lifecycle/factor changes revoke all prior families. If the
  current browser remains signed in, replace its family atomically and return
  only the new credential.
- Validate clients before writing a session and avoid orphan rows.
- Define trusted-proxy-aware keyed fingerprints without retaining raw
  attacker-controlled user-agent or address values.

Acceptance: concurrent refresh/reuse, password change, lock, suspend,
deactivate, cleanup, and crash recovery pass PostgreSQL integration and race
tests.

### Task 5 — Standards-Conforming OIDC Provider

Status: `verified locally` for the isolated candidate protocol harness; full
durable provider/runtime integration remains `not run`.

- Implement discovery, exact client and redirect registry, Authorization Code
  plus PKCE, state/nonce binding, ID/access/refresh tokens, userinfo only when
  required, JWKS, stable key IDs and overlap rotation, revocation, and logout
  through the selected maintained library.
- Keep authorization codes short-lived, single-use, client/redirect/PKCE-bound,
  and hashed or library-protected at rest.
- Reject weak signing keys, unsupported algorithms, insecure production
  issuer URLs, clock anomalies beyond an explicit skew, and initialization
  errors. Readiness includes signing and verification state.
- Do not copy the candidate's discovery-shaped proprietary metadata.
- Bind logout/revocation to the authenticated subject and exact session, and
  make legacy HMAC configuration incapable of altering the RS256/OIDC path.

Acceptance: positive and negative protocol/interoperability suites pass against
the current API OIDC client; no dual issuer or algorithm fallback exists.

### Task 6 — MFA, Recovery, Verification, Mail, And Localization

Status: `verified locally` for bounded local components, encrypted PostgreSQL
outbox retry/lease recovery, durable encrypted PostgreSQL MFA state, and
disposable verified TLS/STARTTLS Mailpit delivery. Isolated runtime and browser
recovery/reset qualification is `verified locally`.

- Implement TOTP enrollment/challenge with encrypted secrets and replay-window
  consumption, plus random hashed single-use recovery codes.
- Implement verified-email invitation, verification, reset, factor recovery,
  and administrative recovery using expiring hashed attempt-bounded challenges.
- Reuse the project's verified TLS/STARTTLS SMTP transport through a narrow
  identity mail port with separate credentials and durable retry/audit policy.
- Add identity catalogs and safe templates for `en`, `tr`, `fr`, and `pt`;
  confirm whether Portuguese is `pt-PT`, `pt-BR`, or shared neutral content.
- Never place tokens in logs, telemetry, subject lines, or unrelated URLs.

Acceptance: enrollment, replay denial, recovery-code consumption, reset reuse,
expiry, throttling, SMTP downgrade denial, delivery retry, and four-locale
rendering pass focused and Mailpit integration tests.

### Task 7 — Security Audit And Provider Administration

Status: `verified locally` for bounded audit/admin components and the neutral
API boundary; authenticated provider-admin HTTP handlers remain `not run`.

- Implement an append-only redacted event schema for registration denial,
  invite, verification, login success/failure, refresh/reuse, logout, password,
  reset, identifier, MFA, factor recovery, lock, lifecycle, session, client,
  key, and admin events.
- Decide and test fail-closed versus alert-only behavior per event class.
- Implement a narrow authenticated internal API for provisioning, provider
  observation, account state, required actions, MFA state, and revocation.
- Scope provider administration through least privilege and exact subject
  identity; no browser receives provider admin authority.

Acceptance: missing/failed audit and authorization paths behave exactly as the
contract states; credentials and unbounded attacker input never enter events.

### Task 8 — Adapt AviaSurveil360 Without Weakening Authorization

Status: `verified locally` for the provider-neutral API boundary, existing BFF
contract, and web refresh coordinator; full replacement-provider E2E wiring is
`not run`.

- Replace `KeycloakAdminClient` ownership with provider-neutral interfaces and
  implementations while Keycloak remains supported only as the active
  migration baseline.
- Preserve current BFF cookies, CSRF, login-state/nonce/PKCE, provider logout,
  application sessions, authority observations, membership revisions, offline
  subject lock, and repository-level organization predicates.
- Keep organization and role authority server-owned. Claims must match the
  expected membership when claim reconciliation remains enabled.
- Add provider-independent API and React flows for login, logout, sessions,
  password, verification, TOTP, recovery codes, reset, and admin lifecycle.
- Serialize concurrent refresh work at every applicable client boundary and
  clear the complete local session after terminal refresh rejection. Revalidate
  or disconnect authenticated long-lived channels on token expiry, auth-version
  change, suspension, password reset, logout, and deactivation.

Acceptance: the existing role/organization/privacy matrix and negative tests
pass against both explicit profiles, never through fallback.

### Task 9 — Isolated Local Qualification And Synthetic Migration

Status: `verified locally` for the deterministic synthetic qualification
harness, isolated browser OIDC/MFA/recovery/reset/logout, dependency-loss,
restart, and backup/restore. Durable signing-key rotation/overlap/retirement and
candidate provider-form accessibility semantics are `verified locally`.
Keycloak rollback traffic, organization denial, and broader lifecycle
qualification are `not run`.

- Add a distinct local profile with a different issuer, client, database, keys,
  cookies, ports, and synthetic identities. Keep the normal Keycloak profile
  unchanged.
- Create exact opaque synthetic subjects from an approved manifest; never
  derive subjects from mutable identifiers.
- Run complete browser login/logout/TOTP/recovery/reset/session/lifecycle,
  organization denial, accessibility, restart, dependency-loss, backup,
  restore, key-rotation, and rollback scenarios.
- Prove that a bootstrap or break-glass provider identity without application
  membership cannot obtain application authority.

Acceptance: full synthetic qualification is `verified locally`, task-owned
state is removed, and Keycloak remains a tested rollback provider.

### Task 10 — Security, Dependency, Recovery, And ARM64 Gates

Status: `verified locally` for local code/image/dependency gates; release is
literal `NO-GO` because independent security and native ARM64 workload evidence
are `not run`.

- Run formatting, vet, unit, PostgreSQL integration, race, fuzz, protocol,
  browser, accessibility, contract, migration, backup/restore, and failure
  suites.
- Run current dependency, license, secret, misconfiguration, SBOM/provenance,
  and image scans. Validate reachability; do not dismiss findings by package
  name alone.
- Obtain independent security review of authentication, OIDC, MFA, recovery,
  organization, migration, key custody, audit, and rollback paths.
- Measure the complete native ARM64 gateway/API/worker/identity workload under
  steady, login burst, Argon2id abuse, PDF, notification, and recovery load.

Acceptance: no unresolved cutover blocker; required headroom passes without
weakened security or sustained swap. Otherwise record literal `NO-GO` and keep
Keycloak.

### Task 11 — Approval-Bound Cutover And Keycloak Retirement

Status: `not run`; requires separate exact authorization.

- Inventory whether any non-synthetic identity exists. If yes, create a
  separate exact migration and recovery plan; never improvise password or TOTP
  export.
- Review one immutable release, migration, issuer/route, smoke, rollback,
  observation, and retention bundle.
- Obtain separate exact authorization for each production identity, secret,
  migration, deployment, cutover, smoke, recovery, rollback, retention,
  destruction, and residue wave.
- Remove Keycloak runtime, database, secrets, image subjects, code, and
  runbooks only after cutover acceptance and retention/destruction approval.

Acceptance: separately authorized real evidence passes. Until then Keycloak is
not retired and no production-ready claim is permitted.

## Verification Commands And Expected Observations

Commands evolve as `apps/auth` is created. Each task must record exact commands
and fresh results. The minimum expected local gate includes:

```bash
gofmt -w apps/auth
go test ./...
go test -race ./...
go vet ./...
npm --prefix apps/web run typecheck
npm --prefix apps/web test
node --test deploy/local/keycloak/realm-contract.test.mjs
./scripts/test-http-oidc-profile.sh
git diff --check
```

Run Go commands from the relevant module and use task-owned cache/database
paths. Add focused auth commands before broad repository gates. Expected
observations are zero formatting/vet/test errors, exact negative OIDC and
organization denials, no secret output, and zero task-owned runtime residue.
Commands that require code not yet created remain `not run`; do not manufacture
passing placeholders.

## Risks And Mitigations

- **Protocol correctness:** use a maintained library, interoperability and
  negative tests, and independent review.
- **New provider ownership:** maintain complete rotation, recovery, backup,
  audit, monitoring, and incident runbooks before cutover.
- **Tenant drift:** keep application authorization and repository scoping in
  AviaSurveil360; reconcile provider observations exactly.
- **Credential abuse and t4g.small exhaustion:** layer rate limits, bounded
  Argon2id work, queue visibility, and native mixed-load measurement.
- **Migration identity loss:** retain exact subjects and append-only mappings;
  never derive identity from email or username.
- **Rollback ambiguity:** distinct issuers and one configured verifier at a
  time; retain Keycloak until post-cutover acceptance.
- **Dirty-tree conflict:** refresh each touched file, preserve unrelated user
  changes, and stop if overlapping intent cannot be reconciled safely.

## Idempotence And Recovery

- Source intake is content-addressed and must not be rewritten.
- Migrations are forward-only, checksum-recorded, transactionally applied, and
  tested on disposable PostgreSQL before use.
- Provisioning, invitation, reset, factor, session, and provider-admin commands
  use stable operation IDs and expected revisions; replay returns the prior
  result or a typed conflict without duplicate delivery or authority.
- Refresh rotation and factor consumption use row locks/unique constraints so
  a crash cannot issue two valid successors.
- Key and issuer rollback is a reviewed configuration/release operation, not a
  verifier fallback.
- Any failed migration or cutover retains Keycloak data and immutable images;
  no cleanup/destruction occurs before explicit retention disposition.

## Progress

- [x] 2026-08-11 — Owner confirmed first-party source ownership and authorized
  repository intake and a separate Auth Replacement ExecPlan.
- [x] 2026-08-11 — Both ZIP SHA-256 values matched the supplied digests;
  archives and exact extractions retained under
  `docs/security/auth-replacement/evidence/`.
- [x] 2026-08-11 — Retained-copy archive, high-signal secret, offline module,
  auth/export race, vet, hardening JSON, harness-docs, and diff checks passed;
  PostgreSQL, current vulnerability, OIDC, integration, and ARM64 gates remain
  `not run`.
- [x] 2026-08-11 — Export and current Avia identity boundaries inspected;
  direct schema/runtime import rejected.
- [x] 2026-08-11 — Security hardening portfolio selected the separate Go OIDC
  provider with Keycloak retained as baseline and rollback.
- [x] 2026-08-11 — Second source manifest, archive structure, extraction, and
  high-signal secret checks passed. Its fresh focused Go test was `blocked` by
  unavailable offline module versions; DynamoDB Local, OIDC, fuzz, mobile, and
  ARM64 gates remain `not run`.
- [x] 2026-08-11 — Comparative review retained RS256/JWKS negative-test,
  lifecycle, distributed-limit, and client refresh-serialization concepts;
  direct DynamoDB/runtime reuse was rejected and twelve regression contracts
  were added.
- [x] 2026-08-11 — Task 1 primary-source comparison, ARM64 disposable spikes,
  AS360-OIDC-WEB-1 core/negative checks, provider-neutral contracts, and the
  `AS360-AUTH-001` through `AS360-AUTH-030` map are recorded in the [spike
  results](../../security/auth-replacement/evidence/2026-08-11-oidc-library-spike/RESULTS.md)
  and [contracts](../../security/auth-replacement/evidence/2026-08-11-oidc-library-spike/CONTRACTS.md).
  ZITADEL, Fosite, and Authelia core checks passed locally; Hydra's current
  server smoke is blocked by unavailable SQL/daemon; framework-owned
  discovery/JWKS/logout and key overlap remain `not run` where stated.
- [x] 2026-08-11 — Task 2 isolated `apps/auth` scaffold is `verified locally`:
  typed fail-closed configuration, secret-file and database/SMTP separation,
  liveness/readiness boundary, redacted telemetry, selected-library pin,
  non-root ARM64 image, opt-in hardened Compose profile, focused unit/race/vet,
  static ARM64 build, Compose syntax, and local ARM64 image inspection are
  recorded in [Task 2 evidence](../../security/auth-replacement/evidence/2026-08-11-oidc-library-spike/TASK2_SCAFFOLD.md).
  Direct host-process listener smoke is `blocked` by sandbox TCP-bind
  permission; provider storage and OIDC routes remain intentionally absent.
- [x] 2026-08-11 — Task 3 identity, password, account-state, and abuse
  controls are `verified locally`: opaque subjects, canonical identifiers,
  revisioned lifecycle, Argon2id bounds/dummy path/history, fail-closed
  trusted-proxy throttling, forward-only identity schema, and disposable
  PostgreSQL lifecycle/constraint tests are recorded in [Task 3 evidence](../../security/auth-replacement/evidence/2026-08-11-oidc-library-spike/TASK3_IDENTITY.md).
- [x] 2026-08-11 — Task 4 provider sessions and refresh families are
  `verified locally`: bounded opaque sessions, durable row-lock rotation,
  reuse/revocation, lifecycle callbacks, race tests, and disposable
  PostgreSQL evidence are recorded in [Task 4 evidence](../../security/auth-replacement/evidence/2026-08-11-oidc-library-spike/TASK4_SESSIONS.md).
- [x] 2026-08-11 — Tasks 5–9 are `verified locally` within their scoped
  candidate boundaries: [Task 5](../../security/auth-replacement/evidence/2026-08-11-oidc-library-spike/TASK5_OIDC_PROVIDER.md), [Task 6](../../security/auth-replacement/evidence/2026-08-11-oidc-library-spike/TASK6_MFA_RECOVERY.md), [Task 7](../../security/auth-replacement/evidence/2026-08-11-oidc-library-spike/TASK7_AUDIT_ADMIN.md), [Task 8](../../security/auth-replacement/evidence/2026-08-11-oidc-library-spike/TASK8_API_WEB_ADAPTATION.md), and [Task 9](../../security/auth-replacement/evidence/2026-08-11-oidc-library-spike/TASK9_QUALIFICATION.md).
- [x] 2026-08-11 — Task 10 local security/dependency/image/build gates are
  `verified locally`; release is literal `NO-GO` pending independent review,
  complete recovery/browser qualification, and native ARM64 workload
  evidence.
- [x] 2026-08-11 — Fresh candidate-only continuation checks are `verified
  locally`: full auth unit/race/protocol tests, vet, auth/API module integrity,
  ARM64 cross-build, hardened Compose policy, and a disposable native ARM64
  image inspection. Full browser OIDC/MFA/recovery/reset/logout, Mailpit retry,
  backup/restore, restart/dependency-loss, rollback traffic, and mixed-load
  capacity are `blocked` by the current runtime's health-only route surface and
  candidate Compose topology not yet wired to PostgreSQL or Mailpit. SBOM/license/
  vulnerability/provenance tooling for the fresh image is `not run`. Keycloak
  remains the required serving and rollback baseline. Task-owned image and
  binary cleanup is complete; root-owned temporary module-cache cleanup is
  `blocked` by host file permissions.
- [x] 2026-08-11 — `scripts/test-auth-candidate-postgres.sh` now creates an
  isolated disposable PostgreSQL namespace, runs the auth identity/session/
  migration integration suite, and removes its container, volume, and runtime
  directory. This gate is `verified locally`; it does not supply the still
  missing browser, Mailpit, restore, rollback-traffic, or mixed-load evidence.
- [x] 2026-08-11 — `scripts/test-auth-candidate-mailpit.sh` starts a
  disposable authenticated Mailpit instance with a task-owned certificate and
  mandatory STARTTLS. `TestMailpitSTARTTLSDelivery` is `verified locally` for
  TLS-verified delivery and receipt visibility; the runner removes its
  container and ephemeral credential material. Durable retry/outbox wiring is
  still absent from the candidate runtime.
- [x] 2026-08-11 — Added forward-only migration `000004_mail_outbox.up.sql`
  and an encrypted-at-rest PostgreSQL mail outbox. Its bounded retry, lease
  ownership/recovery, and terminal delivery state are `verified locally` in a
  disposable PostgreSQL project. `scripts/test-auth-candidate-mailpit-outbox.sh`
  then exercised a planned transient failure followed by authenticated,
  certificate-verified STARTTLS delivery to disposable Mailpit and receipt API
  confirmation. The runner removed all task-owned containers, volumes, and
  temporary credentials. This remains `candidate-only`; no route is mounted
  and the remaining browser/runtime qualification is `blocked`.
- [x] 2026-08-11 — Added forward-only migration `000005_mfa.up.sql` and the
  PostgreSQL MFA adapter. It stores AES-GCM-protected TOTP factors, durable
  monotonic replay counters, and only hashed single-use recovery codes; row
  locking bounds recovery failures and reset deletes the factor state. The
  focused unit/migration tests and `scripts/test-auth-candidate-postgres.sh`
  are `verified locally`. This remains `candidate-only`; no runtime route
  consumes it and browser qualification is `blocked`.
- [x] 2026-08-11 — Added forward-only migration `000006_challenges.up.sql`
  and the PostgreSQL challenge adapter. It persists only token hashes with
  exact subject/purpose binding, expiry, attempt budgets, single-use and
  invalidation state. Consume/reject transitions use row locks; concurrent
  consumption yields one success and one used/replay result. Focused tests and
  `scripts/test-auth-candidate-postgres.sh` are `verified locally`. This
  remains `candidate-only`; no runtime route consumes it and browser
  qualification is `blocked`.
- [x] 2026-08-11 — Added forward-only migration `000007_oidc_runtime.up.sql`
  and a PostgreSQL `zitadel/oidc` storage adapter. Authorization and refresh
  credentials persist only as hashes; exact clients, access/revocation state,
  and encrypted signing-key material are durable. The disposable PostgreSQL
  test passed client authentication, encrypted key load, authorization-code
  state, and refresh rotation/reuse denial `verified locally`. This remains
  `candidate-only`; the liveness-only HTTP server does not mount this adapter.
- [x] 2026-08-11 — Added isolated provider-owned login and MFA handlers over
  the durable stores. A successful password authentication stages a subject;
  an enabled TOTP factor or recovery code must complete before the OIDC
  callback is authorized. The disposable PostgreSQL suite is `verified locally`
  for password login and the MFA-required path. These handlers are not mounted
  by `cmd/auth`, so the candidate runtime remains `blocked` from browser
  qualification.
- [ ] Task 11 — Separately authorized cutover and retirement.

## Decision Log

- 2026-08-11: Source licensing blocker closed by the owner's explicit ownership
  statement; third-party notices remain required.
- 2026-08-11: Preserve exact export as immutable evidence rather than placing
  known-defective candidate code directly in an application build.
- 2026-08-11: Preserve the second exact export beside the first and adapt
  selected concepts only. Do not import its DynamoDB persistence, minimal
  discovery facade, or application/mobile domain runtime.
- 2026-08-11: Preserve OIDC and the existing same-origin BFF rather than moving
  browser refresh tokens into JavaScript.
- 2026-08-11: Prefer a separately privileged Go provider over embedding
  credential/MFA/signing authority in the API.
- 2026-08-11: Keep Keycloak serving until parity, migration, independent
  security, recovery, rollback, and ARM64 gates pass.
- 2026-08-11: TOTP and recovery codes are mandatory; WebAuthn remains an open
  product decision.
- 2026-08-11: Task 1 evidence is complete and locally verified. The owner
  authorized Option A (`zitadel/oidc` v3.47.5, `fb9fbfe`) with “tamamdır olur
  kabul” and “tamam evet”; this dated decision authorizes only local Task 2–10
  implementation with that library. No production, deployment, identity,
  migration, traffic, or Keycloak-retirement action is authorized.
- 2026-08-11: Tasks 4–10 were completed sequentially as isolated local
  candidate work using only Option A. Keycloak remains the serving/rollback
  baseline; Task 10 is a local-gate `NO-GO` for release, and Task 11 still
  requires separate exact authorization.

## Discoveries

- The export is native password/session authentication with proprietary JWTs,
  not an OIDC client or conforming OIDC authorization server.
- The second export is also proprietary authentication, despite an RS256
  issuer and discovery/JWKS facade. It is neither a conforming OIDC provider nor
  an OIDC relying party.
- Its strongest reusable elements are Argon2id, random identifiers, exact JWT
  algorithm checks, row-lock refresh rotation, and family-reuse concepts.
- Its evaluation schema duplicates application user and permission concerns
  already owned more strongly by AviaSurveil360; importing it would create an
  unsafe second authorization authority.
- AviaSurveil360 already has a robust BFF/session/CSRF and desired-membership
  foundation. The replacement can focus on provider credentials, factors,
  OIDC, recovery, and provider administration.
- Password change refresh-family survival, unbounded Argon2id work, incomplete
  audit, absent tenant isolation, and misleading discovery are explicit
  regression targets, not accepted debt.
- The second export adds concrete stale-write, cross-account logout,
  verification-gate, GraphQL rate-limit, WebSocket revocation, key-rotation,
  legacy-HMAC, OTP-attempt, and body-size regression targets. Its Swift refresh
  actor is useful client-behavior evidence but does not change the BFF rule that
  browser JavaScript never holds provider refresh tokens.
- `zitadel/oidc` exposes an OP-capable library surface; Fosite and
  `authelia.com/provider/oauth2` expose protocol frameworks and require a
  reviewed host adapter for discovery, JWKS/key rotation, login/consent, and
  logout. They must not be described as standalone providers.
- Ory Hydra is a current standalone server rather than an importable current
  `/v2` module: its `v26.2.0` tag cannot be selected as a valid `/v2` semver.
  The compatibility schema compiled on ARM64, but the current SQL plus
  login/consent server smoke was blocked by unavailable Docker/OrbStack.
- The ARM64 package checks passed in isolated disposable modules. No result is
  a production RSS/CPU/capacity claim; key overlap, restart, mixed workload,
  and independent security gates remain `not run`.
- Task 2's provider scaffold is intentionally liveness-only: the selected
  library dependency is pinned, but no OIDC endpoint, identity storage,
  migration, database connection, SMTP delivery, or normal Compose route is
  enabled before its later contract task.
- Task 3 now has both an in-memory domain adapter for deterministic unit/race
  tests and a PostgreSQL adapter backed by the separate `auth_identity`
  schema. Public registration remains disabled; invitation/verification/
  activation and all security-state mutations use opaque subjects and expected
  revisions. OIDC/session/MFA routes still do not consume this adapter.
- Task 4 durable refresh families and lifecycle revocation now share auth
  revision checks; concurrent predecessor presentation cannot mint two
  successors.
- Task 5's provider candidate is a library-owned protocol harness with a
  narrow AS360 S256-PKCE boundary; durable storage, browser UI, and key-ring
  operations remain host-owned and candidate-only.
- Task 8 keeps browser refresh credentials server-side. Only a single
  same-origin refresh promise is shared in React, and terminal rejection
  clears the local session.
- Task 10's first image scan exposed CVE-2026-56852 in `golang.org/x/text`
  v0.37.0; the dependency was upgraded to v0.39.0, the ARM64 image rebuilt,
  and the HIGH/CRITICAL scan then passed with zero findings.
- A fresh local continuation confirms the candidate server still deliberately
  exposes only health routes. Its protocol, identity, MFA, recovery, mail, and
  administration components are not runtime-wired; therefore the missing
  browser/recovery/capacity evidence is a structural `blocked` gate rather than
  a test runner omission.
- The durable MFA adapter is isolated in the privileged `auth_identity` schema.
  It is not a claim that the liveness-only auth server now offers MFA, recovery,
  or reset routes.

## Outcome

Tasks 0–10 are `verified locally` for isolated candidate implementation and
local evidence: both source collections are preserved, integrity-bound,
classified, compared, and linked to the selected separate-provider
architecture; the Task 1 library evidence and contracts are retained in the
security package. Option A (`zitadel/oidc` v3.47.5, `fb9fbfe`) was authorized on
2026-08-11. Task 10 records a release `NO-GO` because independent security and
native ARM64 workload gates remain `not run`.
Production, deployment, identity migration, traffic, and Keycloak retirement
remain unauthorized. The result is `candidate-only`, release is `release
pending`, and `production-ready: not established`. Isolated runtime browser,
dependency-loss, restart, and backup/restore evidence is `verified locally`;
it does not change Task 11 authorization.

## Execution Prompt

Continue
`docs/exec-plans/active/2026-08-11-first-party-go-oidc-auth-replacement-plan.md`
from its next incomplete task. Read `AGENTS.md`, `docs/PLANS.md`, this complete
plan, the plan index, the technical-debt tracker, `ARCHITECTURE.md`, both raw
auth export receipts, their complete retained review/integration documents,
the source comparison, the
hardening portfolio and selected implementation handoff, the current OIDC,
Keycloak administration, session/BFF, React auth, identity profile, MFA
runbook, Compose, gateway, SMTP, migration, recovery, and AWS private-pilot
runtime sources before editing. Preserve all unrelated dirty-tree changes.

Tasks 0–10 have been executed sequentially. Preserve the recorded evidence,
refresh any stale command/result rows only when a fresh check is actually run,
and do not begin Task 11 without separate exact authorization. Do not
bulk-copy either imported store or apply the evaluation migration. Keep
Keycloak active and do not enable dual-issuer fallback. Use `apply_patch` for
manual edits, run focused tests, and update plan/index/tracker evidence
literally.

Do not stage, commit, push, deploy, publish, mutate AWS/Cloudflare/SMTP, touch
production/customer identity or data, migrate RDS, change DNS/certificates or
secrets, cut over traffic, remove Keycloak, or destroy/retain provider state
without separate exact authorization.

## Browser qualification attempt (2026-08-12)

The isolated browser runner used a task-owned persistent Playwright profile and
the disposable Compose topology. Discovery, `/authorize`, provider-owned login,
and MFA form submission reached the provider callback redirect. The browser
then aborted its GET to `/authorize/callback` with `net::ERR_ABORTED` before it
reached the local callback. The OIDC browser qualification is `blocked`; it is
not `verified locally`. No recovery/reset/logout browser claim follows from
this attempt. All task-owned containers, volumes, temporary secrets, browser
profile, and browser processes were removed. The candidate remains
`candidate-only`, Keycloak remains serving and rollback baseline, and release
remains `release pending`.

## Dependency-loss and restart qualification (2026-08-12)

The isolated runtime runner stopped only task-owned authenticated STARTTLS
Mailpit and observed the running auth service return HTTP 503 readiness while
liveness remained available. After Mailpit was restored, readiness recovered.
The runner then restarted the auth service and observed readiness recover again
against the same durable PostgreSQL state. This is `verified locally` and
`candidate-only`; browser callback qualification remains `blocked`, Keycloak
remains serving and rollback baseline, and release remains `release pending`.

## Backup and restore qualification (2026-08-12)

`scripts/test-auth-candidate-backup-restore.sh` created a task-owned PostgreSQL
dump, stopped only the disposable auth service, removed only its disposable
`auth_identity` schema, restored the dump, and restarted the runtime. Readiness
returned and the synthetic active account remained present. This is `verified
locally` and `candidate-only`; it does not establish any external recovery
objective. Browser callback qualification remains `blocked`, Keycloak remains
serving and rollback baseline, and release remains `release pending`.

## Runtime continuation (2026-08-12)

`cmd/auth` now applies the privileged migrations and initializes durable
PostgreSQL identity, MFA, OIDC storage, and admission limiting before it mounts
the isolated `RuntimeCandidate`; readiness opens only after those checks pass.
The opt-in candidate Compose topology now has its own PostgreSQL and
authenticated STARTTLS Mailpit services and consumes generated task-owned file
secrets. `scripts/test-auth-candidate-runtime.sh` built and started that
three-service topology, then queried liveness, readiness, and OIDC discovery
from the private network `verified locally`; its containers, volumes, and
secret directory were removed. This is `candidate-only`. Provider-owned
recovery, reset, and explicit logout handlers, browser qualification, and the
later restart/restore/rollback/capacity gates are `not run`. Keycloak remains
the serving and rollback baseline; release remains `release pending`.

`/recover/password` is now mounted on that isolated runtime. The same runner
seeded one synthetic `.invalid` active account, submitted its generic recovery
request, and observed the encrypted outbox worker deliver to authenticated
STARTTLS Mailpit `verified locally`; it did not print the one-time value. The
provider PostgreSQL suite now exercises those token-consumption password
and MFA reset handlers plus the narrow `/logout` entry point `verified locally`:
the reset password authenticates, the MFA factor is deleted, and logout only
redirects to the selected provider's end-session endpoint. Browser workflows
remain `not run`. This remains `candidate-only`, literal `NO-GO`, and `release
pending`.

## Browser qualification completion (2026-08-12)

`scripts/test-auth-candidate-browser.sh` is `verified locally` in a fresh,
disposable PostgreSQL/authenticated STARTTLS Mailpit topology and an isolated
ephemeral Playwright browser context. It completed OIDC Authorization Code with
S256 PKCE and standard `form_post` response delivery, provider-owned password
login, TOTP MFA, token exchange, generic recovery initiation, password reset,
MFA reset, a post-reset password login that did not request MFA, and explicit
logout. The runner removed its task-owned browser, containers, volumes, and
temporary secrets. This evidence is `candidate-only`; it does not test rollback
traffic, native ARM64 mixed-load capacity, fresh SBOM/license/vulnerability/
provenance, independent review, or release approval, which are `not run`.
Keycloak remains serving and the rollback baseline, and release remains
`release pending` with literal `NO-GO`.

## Native ARM64 bounded auth mixed-load qualification (2026-08-12)

On this `arm64` host, `scripts/test-auth-candidate-load.sh` is `verified
locally` in the task-owned candidate PostgreSQL/authenticated STARTTLS Mailpit
topology. It completed two concurrent successful Authorization Code + S256 PKCE
password logins at the configured Argon2id capacity, two concurrent rejected
unknown-account Argon2id attempts, four recovery requests with matching outbox
Mailpit receipts, and concurrent readiness/discovery probes in 590 ms. The
runner enforces the hasher's configured two-operation Argon2id ceiling rather
than weakening it for load, and removed its task-owned resources. This is
`candidate-only`; it is not the complete gateway/API/worker/PDF system workload,
so native ARM64 mixed-load/capacity for the release gate remains `not run`.
Keycloak remains serving and the rollback baseline; release remains `release
pending` with literal `NO-GO`.

## Candidate provider-form accessibility qualification (2026-08-12)

The isolated browser runner is `verified locally` for the provider-owned login,
MFA, recovery, password-reset, and MFA-reset forms: each has English document
language, exactly one main landmark and level-one heading, one submit control,
and accessible label/control pairs with required non-checkbox inputs. This is
`candidate-only` provider-form semantics, not the complete product
accessibility matrix, which remains `not run`. Keycloak remains serving and the
rollback baseline; release remains `release pending` with literal `NO-GO`.

## Fresh module and web qualification (2026-08-12)

`go mod verify` in `apps/auth` and `apps/api`, web typecheck, and the web
Vitest suite (85 files / 751 tests) are `verified locally`. `syft`, `trivy`,
and `govulncheck` are absent, while `docker sbom` is unavailable; a fresh
SBOM/license/vulnerability result therefore remains `not run`. This preserves
the `candidate-only`, Keycloak-retained, `release pending` literal `NO-GO`
disposition.

## Local/disposable gate closure audit (2026-08-12)

The technically solvable isolated candidate gates are `verified locally`:
durable identity/MFA/challenge/OIDC/outbox state; provider login, MFA, recovery,
reset, logout, signing-key rotation; authenticated STARTTLS Mailpit delivery;
readiness dependency loss and restart; backup/restore; browser protocol and
provider-form semantic accessibility; bounded native ARM64 auth-only load;
module/web integrity; and focused/full/race/vet checks.

The remaining items are not a smaller local substitute: authenticated provider
administration and application BFF E2E require the application-owned authority
boundary and a separately evidenced candidate API/web profile; organization
denial requires that same membership authority; Keycloak rollback traffic would
touch the serving baseline; complete gateway/API/worker/PDF capacity and
product accessibility require the wider system topology; SBOM/license/
vulnerability/provenance tools are unavailable; and independent review/release
approval are external. These remain `not run` or `blocked` as recorded. No
candidate-only result permits removing, weakening, or routing traffic away from
Keycloak. Release remains `release pending` with literal `NO-GO`.

## Durable signing-key rotation qualification (2026-08-12)

`scripts/test-auth-candidate-postgres.sh` is `verified locally` for the durable
candidate key ring. It encrypted a new RSA private key at rest, moved the old
active key to a finite overlap, returned both public keys from the storage JWKS
set, then retired the elapsed overlap and returned only the new key. Rotation
rejects an invalid, same-ID, or unbounded-overlap request. This is
`candidate-only` storage evidence; key custody, operational rotation approval,
and release provenance remain `not run`. Keycloak remains serving and the
rollback baseline; release remains `release pending` with literal `NO-GO`.
