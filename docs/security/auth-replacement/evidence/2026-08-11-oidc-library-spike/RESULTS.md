# Task 1 OIDC library comparison and disposable evidence

Date: 2026-08-11

Status: `verified locally` for the primary-source comparison, isolated ARM64
package checks, and the disposable tests recorded below. The result remains
`candidate-only`; production readiness is not established.

## selected-library

- **Option A — `zitadel/oidc` v3.47.5 (`fb9fbfe`)** is authorized for Task 2
  and later local implementation tasks.
- Authorization date: 2026-08-11.
- Owner authorization text: “tamamdır olur kabul”, followed by the explicit
  confirmation “tamam evet”, recorded as acceptance of the previously
  recommended Option A (`zitadel/oidc`).
- This authorization selects the library only; it does not authorize
  production identity, secrets, deployment, traffic, migration, cutover, or
  Keycloak retirement.

Task 2 may now create `apps/auth/` and proceed only with this selected library.
Keycloak remains the active provider and rollback baseline until every later
gate passes.

## Evidence boundary

The spike is disposable and lives only under
`/private/tmp/avia-auth-oidc-spike`. It contains no repository runtime code,
database migration, deployment artifact, secret, or `apps/auth/` directory.
The local host is `darwin/arm64` with Go 1.26.4. Exact test commands and
results are retained in the spike's per-option run sheets.

Primary sources used for the comparison:

| Option | Primary version/commit | License, maintenance, and security evidence |
| --- | --- | --- |
| `zitadel/oidc` | [`v3.47.5`](https://github.com/zitadel/oidc/releases/tag/v3.47.5), `fb9fbfe` | [OP/client README](https://github.com/zitadel/oidc/tree/v3.47.5), [Apache-2.0 license](https://github.com/zitadel/oidc/blob/v3.47.5/LICENSE), [repository security policy](https://github.com/zitadel/oidc/security/policy), signed current release with active tags |
| `ory/fosite` | [`v0.49.0`](https://github.com/ory/fosite/releases/tag/v0.49.0), `653c812` | [framework README](https://github.com/ory/fosite/blob/v0.49.0/README.md), [Apache-2.0 license](https://github.com/ory/fosite/blob/v0.49.0/LICENSE), [security policy](https://github.com/ory/fosite/security/policy), current release page and changelog |
| `authelia.com/provider/oauth2` | [`v0.2.40`](https://github.com/authelia/oauth2-provider/tree/v0.2.40), `8939c05` | [fork README](https://github.com/authelia/oauth2-provider/blob/v0.2.40/README.md), [Apache-2.0 license](https://github.com/authelia/oauth2-provider/blob/v0.2.40/LICENSE), active signed tags but [no GitHub release artifacts](https://github.com/authelia/oauth2-provider/releases); the parent [Authelia security policy](https://github.com/authelia/authelia/security/policy) is not a substitute for a fork-specific release process |
| Ory Hydra | [`v26.2.0`](https://github.com/ory/hydra/releases/tag/v26.2.0), `0b84568` | [server README](https://github.com/ory/hydra/blob/v26.2.0/README.md), [Apache-2.0 source license](https://github.com/ory/hydra/blob/v26.2.0/LICENSE), [current security policy](https://github.com/ory/hydra/security/policy), signed release changelog includes dependency/security fixes; Ory Enterprise is required for vendor SLAs, regular security releases, and CVE support |
| `coreos/go-oidc` | [`v3.20.0`](https://github.com/coreos/go-oidc/releases/tag/v3.20.0), `75dfa5c` | [RP README](https://github.com/coreos/go-oidc/blob/v3.20.0/README.md), [Apache-2.0 license](https://github.com/coreos/go-oidc/blob/v3.20.0/LICENSE), [security policy](https://github.com/coreos/go-oidc/security/policy), maintained RP release; GitHub's Security quality indicator was `0` at inspection, so dependency review remains required |
| Low-level JOSE/OAuth2 composition | No single version or commit; every primitive would need an independent pin | License and security posture would be the union of independently selected JOSE/OAuth2/HTTP components; no single maintainer owns the OP contract |
| Libraryless/manual | No version, commit, or library license | No maintained security boundary; all protocol and cryptographic behavior would become Avia-owned code |

## Option-by-option comparison

### `zitadel/oidc`

- **Version/commit and license:** v3.47.5, `fb9fbfe`, Apache-2.0. The release
  is signed and contains a recent loopback/post-logout redirect fix.
- **Maintenance and security:** active upstream release/tag cadence and a
  repository security policy. The README presents both an OIDC relying-party
  client and an OpenID Provider package under `pkg/op`; the example is not a
  production identity product.
- **OIDC/OP scope:** the broadest library-level fit in this comparison:
  provider endpoints, discovery, authorization-code processing, token and
  OIDC ID-token handling, JWKS, and OP logout hooks are available as host
  components. Account UI, user lifecycle, durable stores, and operational key
  custody remain host responsibilities.
- **Integration:** an isolated provider process can own the HTTP surface and
  adapt the existing API/BFF as an OIDC RP. Exact issuer and redirect registry,
  client policy, account provisioning, MFA/recovery, and application
  authorization stay behind provider-neutral interfaces.
- **DB/secret separation:** the package does not grant the API its signing,
  MFA, SMTP, or provider database authority. The host must supply durable
  storage and key custody; the disposable example uses in-memory users/clients
  and a generated key only.
- **ARM64/resources:** ARM64 package and protocol test passed locally. The
  dependency graph is predominantly Go; no resource measurement, RSA/key
  rotation stress, or mixed API/provider workload was run. The example is
  intentionally too small to establish production RSS or CPU headroom.
- **Advantages:** provider surface exists; current client and OP packages;
  clean Apache-2.0 terms; small disposable integration; existing Avia RP can
  verify its ID token through `coreos/go-oidc`.
- **Disadvantages:** host still owns identity persistence, login/consent UX,
  lifecycle, key-ring overlap, audit, and recovery. The upstream example's
  single generated signing key cannot prove rotation overlap.
- **Disqualify reasons:** none at the Task 1 evidence gate. The option remains
  candidate-only until owner authorization, a real storage/key-ring adapter,
  full negative matrix, ARM64 capacity evidence, and independent review.
- **Required tests and local result:** [`AS360-OIDC-WEB-1.md`](/private/tmp/avia-auth-oidc-spike/zitadel-oidc/AS360-OIDC-WEB-1.md)
  and `contract_test.go` passed discovery, exact issuer, code+PKCE, exact
  redirect negative, wrong-login negative, state, nonce, token exchange,
  ID-token verification, JWKS, logout redirect negative, and default logout
  locally. Key rotation overlap/retirement is explicitly `not run`.

### `ory/fosite`

- **Version/commit and license:** v0.49.0, `653c812`, Apache-2.0.
- **Maintenance and security:** current release and changelog plus a security
  policy. Its README calls it a security-first OAuth2/OIDC **framework** and
  explicitly puts TLS, storage, clients, user interaction, and server wiring
  on the integrator.
- **OIDC/OP scope:** RFC OAuth2/OIDC handlers, PKCE, code/implicit/hybrid
  flows, token issuance, and OIDC ID-token strategies. It does not publish
  discovery, provide a login/consent application, own JWKS/key rotation,
  implement RP logout, or supply an identity store as a complete OP.
- **Integration:** a reviewed HTTP adapter must route authorize/token/error
  handlers, enforce client and redirect policy, render login/consent, publish
  discovery/JWKS, and connect provider-neutral identity/session interfaces.
- **DB/secret separation:** storage and signing strategy are injected by the
  host. The example memory store is not a production persistence decision;
  provider database credentials and signing/MFA/SMTP secrets must stay outside
  the API and worker roles.
- **ARM64/resources:** the isolated package and adapter-shaped protocol test
  passed on ARM64. The dependency graph is broad (including `ristretto` and
  many indirect Ory/telemetry packages); no RSS, CPU, CGO, or mixed-workload
  measurement was run.
- **Advantages:** mature composable handlers, explicit storage/key strategy,
  useful negative-path hooks, Apache-2.0, and a large upstream integration
  test corpus.
- **Disadvantages:** high adapter ownership: discovery, key publication and
  overlap, login/consent, logout, audit, lifecycle, and persistence are all
  outside the library. A thin-looking adapter can accidentally become a
  second authentication product.
- **Disqualify reasons:** disqualified as a standalone server. It remains a
  candidate only if the owner authorizes the adapter boundary and accepts the
  added protocol/security review scope.
- **Required tests and local result:** [`AS360-OIDC-WEB-1.md`](/private/tmp/avia-auth-oidc-spike/ory-fosite/AS360-OIDC-WEB-1.md)
  and `protocol_test.go` passed exact redirect rejection, code+PKCE, state,
  nonce, token exchange, RS256 ID-token signature/issuer/audience/nonce
  verification, and code replay negative. Discovery, JWKS, logout, and key
  overlap intentionally remain `not run` because Fosite does not own them.

### `authelia.com/provider/oauth2`

- **Version/commit and license:** v0.2.40, `8939c05`, Apache-2.0. It is an
  Authelia-maintained hard fork of Fosite, not the Authelia server itself.
- **Maintenance and security:** active signed tags and recent dependency
  updates, but no GitHub release artifacts and no independent fork-specific
  advisory/release channel visible at inspection. Parent Authelia is a mature
  OIDC project; that history reduces but does not remove fork divergence risk.
- **OIDC/OP scope:** OAuth2/OIDC 1.0 handlers and protocol strategies with
  PKCE and related negative-test hooks. Like Fosite, it is a framework: no
  standalone discovery, login/consent UI, JWKS/key-ring service, durable
  identity store, or RP logout endpoint.
- **Integration:** requires the same reviewed HTTP, identity, discovery/JWKS,
  logout, and storage adapter as Fosite, plus a deliberate upgrade policy for
  the fork and its Go 1.25/toolchain 1.26.5 requirements.
- **DB/secret separation:** storage, client registry, and key material are
  host-injected; the memory store is only a disposable fixture. Provider DB,
  signing, MFA, and SMTP secrets must be process/role-separated from API and
  worker credentials.
- **ARM64/resources:** isolated package and adapter-shaped protocol test
  passed on ARM64. Its direct graph is smaller than Fosite's and uses
  `go-jose/v4` and non-generic `ristretto v0.2.0`; no resource or concurrency
  benchmark was run.
- **Advantages:** active fork, modern JOSE/crypto dependency set, explicit
  protocol handlers, and compatibility with the same compositional adapter
  pattern.
- **Disadvantages:** framework gap remains; fork API and release/security
  process can diverge from upstream Fosite; no standalone server or current
  release artifact was found.
- **Disqualify reasons:** disqualified as a standalone server and higher
  governance risk than upstream Fosite unless the owner explicitly accepts
  the fork maintenance boundary.
- **Required tests and local result:** [`AS360-OIDC-WEB-1.md`](/private/tmp/avia-auth-oidc-spike/authelia-oauth2/AS360-OIDC-WEB-1.md)
  and `protocol_test.go` passed exact redirect rejection, code+PKCE, state,
  nonce, token exchange, RS256 ID-token signature/issuer/audience/nonce
  verification, and code replay negative. Discovery, JWKS, logout, and key
  overlap are `not run` because they are host responsibilities.

### Ory Hydra

- **Version/commit and license:** v26.2.0, `0b84568`, Apache-2.0 source. Its
  [`go.mod`](https://github.com/ory/hydra/blob/v26.2.0/go.mod) uses
  `github.com/ory/hydra/v2`, but `v26.2.0` is not a valid
  semver for that module path; it is distributed as an official server rather
  than a current importable library.
- **Maintenance and security:** signed current release with dependency and
  security fixes in its changelog and a repository security policy. Vendor
  SLAs, regular security releases, and CVE support are an Ory Enterprise
  concern, so the operating/support decision must be explicit.
- **OIDC/OP scope:** hardened standalone OAuth2/OIDC provider with discovery,
  authorization/token endpoints, client management, consent/login bridges,
  JWKS, PKCE and certified-provider claims. It intentionally does not manage
  end-user accounts itself.
- **Integration:** run the official binary/container as a separate provider;
  provision SQL, a login/consent bridge, client registry, admin endpoint
  controls, and the existing API/BFF RP configuration. This is an operational
  integration, not a Go package embedding.
- **DB/secret separation:** Hydra owns SQL token/client/session state and
  provider signing/system secrets in its own process and database role. The
  login/consent bridge needs a separately reviewed authority boundary; Avia
  application authorization remains outside Hydra.
- **ARM64/resources:** current-server resource behavior was not measured. It
  is the heaviest option here because it adds a server, SQL, migrations,
  login/consent bridge, and operational endpoints. The compatibility `spec`
  package compiled on ARM64, but that is not a server capacity result.
- **Advantages:** complete server ownership of protocol endpoints, strong
  standards surface, established Ory operational model, and less custom
  protocol code in Avia.
- **Disadvantages:** multi-service operational footprint; SQL and bridge
  dependencies; enterprise support/licensing decision; current release cannot
  be imported using normal `/v2` semver; migration, secrets, and key rotation
  need provider-specific runbooks.
- **Disqualify reasons:** not disqualified by design, but the current-server
  AS360 smoke is blocked locally because Docker/OrbStack is unavailable and no
  disposable SQL/login-consent stack exists. It cannot be authorized from this
  evidence alone.
- **Required tests and local result:** [`AS360-OIDC-WEB-1.md`](/private/tmp/avia-auth-oidc-spike/ory-hydra/AS360-OIDC-WEB-1.md)
  records all discovery, code+PKCE, redirect/client, state/nonce, token,
  ID-token, JWKS/rotation, and logout checks as `not run / blocked`. The
  compatibility module compile passed; it is not a protocol pass.

### `coreos/go-oidc`

- **Version/commit and license:** v3.20.0, `75dfa5c`, Apache-2.0.
- **Maintenance and security:** current RP release and security policy. The
  repository has a small open-issue surface but the GitHub Security quality
  indicator was `0` at inspection; Avia must still run dependency and
  verification tests.
- **OIDC/OP scope:** relying-party client only: discovery, authorization-code
  exchange support through `x/oauth2`, ID-token verification, and remote JWKS.
  It is not an authorization server or OP.
- **Integration:** keep it at the existing API/BFF RP boundary; it is not a
  replacement provider. The current API already uses v3.15.0 and has focused
  discovery, issuer, PKCE, nonce, and JWKS rotation tests.
- **DB/secret separation:** no provider DB; the RP needs client credentials
  and verifier configuration only. Provider signing/MFA/SMTP authority remains
  elsewhere.
- **ARM64/resources:** small, pure-Go RP dependency and low expected memory;
  no OP workload applies and no new benchmark was run.
- **Advantages:** proven existing Avia RP integration, narrow API, maintained
  verification boundary.
- **Disadvantages:** cannot issue authorization codes, tokens, ID tokens, or
  manage users/clients; choosing it would leave the replacement provider
  unimplemented.
- **Disqualify reasons:** not applicable as the requested authorization-server
  option; it cannot satisfy the OP side of AS360-OIDC-WEB-1.
- **Required tests and local result:** retain the existing API RP matrix
  (discovery issuer, exact redirect, PKCE, nonce, ID-token claims, and JWKS
  rotation). No OP spike was created because the option has no OP surface.

### Low-level JOSE/OAuth2 composition

- **Version/commit and license:** no single version or commit; every JOSE,
  OAuth2, HTTP, discovery, and key library would need independent pins and
  notices. A composition is not itself a licensed maintained OP.
- **Maintenance and security:** fragmented advisories and responsibility;
  Avia would own protocol state-machine correctness, cryptographic selection,
  parser hardening, and interoperability drift.
- **OIDC/OP scope:** primitives only. Discovery, login/consent, code/PKCE,
  nonce/state, token exchange, claims, JWKS rotation, logout, replay controls,
  and storage semantics would be composed and reviewed by Avia.
- **Integration and DB/secrets:** maximum host ownership and blast radius;
  provider DB, signing/MFA/SMTP secret separation would still need to be
  designed from scratch.
- **ARM64/resources:** likely pure-Go if compatible primitives are chosen, but
  no meaningful estimate is possible before a full implementation; no spike
  was run.
- **Advantages:** narrow dependency surface and complete local control.
- **Disadvantages:** duplicates maintained protocol implementations and creates
  the largest security/review burden.
- **Disqualify reasons:** the active ExecPlan explicitly excludes custom
  authorization-code, JWT-parser, and cryptographic mechanics while maintained
  protocol libraries are available. Reconsideration requires an explicit
  contract amendment and independent security review.
- **Required tests:** complete AS360-OIDC-WEB-1 positive/negative matrix,
  RFC/OIDC conformance, parser fuzzing, algorithm confusion/issuer/audience
  tests, key-rotation/restart/rollback, race tests, and independent security
  review. No test was attempted under the current contract.

### Libraryless/manual solution

- **Version/commit and license:** none; there is no third-party maintenance or
  license boundary to evaluate.
- **Maintenance and security:** all protocol, cryptography, parser, storage,
  and incident response responsibility would be Avia-owned.
- **OIDC/OP scope:** none until every server surface is written manually.
- **Integration and DB/secrets:** maximum custom integration and secret/DB
  responsibility; no established separation exists.
- **ARM64/resources:** unknowable without implementation; no spike was run.
- **Advantages:** no dependency maintenance or external server footprint.
- **Disadvantages:** highest correctness and security risk, no upstream
  interoperability evidence, and no independent owner for urgent fixes.
- **Disqualify reasons:** explicitly excluded by the active ExecPlan;
  reconsideration requires an explicit contract amendment and independent
  security review.
- **Required tests:** same complete conformance, fuzz, race, rotation,
  recovery, ARM64, and independent-review gates as the low-level option. No
  test was attempted under the current contract.

## AS360-OIDC-WEB-1 result matrix

| Check | ZITADEL example | Fosite adapter | Authelia fork adapter | Hydra current server | `go-oidc` / low-level / manual |
| --- | --- | --- | --- | --- | --- |
| Discovery and exact issuer | verified locally | host-owned; `not run` | host-owned; `not run` | blocked / not run | RP-only or disqualified |
| Authorization Code + PKCE | verified locally | verified locally | verified locally | blocked / not run | RP-only or disqualified |
| Exact redirect and client negatives | redirect negative verified | redirect negative verified | redirect negative verified | blocked / not run | RP-only or disqualified |
| State and nonce | positive callback/claim checks verified | positive checks verified | positive checks verified | blocked / not run | RP-only or disqualified |
| Token exchange | verified locally | verified locally | verified locally | blocked / not run | RP-only or disqualified |
| ID-token verification | `go-oidc` verifier verified | disposable RS256 signature/claims verified | disposable RS256 signature/claims verified | blocked / not run | RP verifier only or disqualified |
| JWKS | endpoint verified | host-owned; `not run` | host-owned; `not run` | blocked / not run | RP fetch only or disqualified |
| Key overlap/retirement | skipped; `not run` | host key-ring; `not run` | host key-ring; `not run` | blocked / not run | disqualified/manual not permitted |
| RP-initiated logout | redirect negative and default redirect verified | host-owned; `not run` | host-owned; `not run` | blocked / not run | RP-only or disqualified |
| Wrong verifier, wrong login, or code replay | wrong verifier/login verified | wrong verifier and replay verified | wrong verifier and replay verified | blocked / not run | disqualified/manual not permitted |
| Unknown `kid`, wrong issuer/audience, subject-bound logout | not run or adapter-specific | not run | not run | blocked / not run | required before authorization |

This matrix deliberately distinguishes `verified locally` from `not run` and
`blocked`; no missing test has been turned into a selection recommendation.

## Decision state

Task 1 evidence is complete and Option A (`zitadel/oidc` v3.47.5,
`fb9fbfe`) was authorized by the owner on 2026-08-11. The dated decision is
also recorded in the active ExecPlan and implementation handoff. Only this
option may proceed to Task 2; the other options remain comparison evidence and
are not implementation dependencies.
