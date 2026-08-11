# Task 5 standards-conforming OIDC provider candidate

Date: 2026-08-11

Status: `verified locally` for the isolated `zitadel/oidc` provider candidate,
its exact client/redirect policy, Authorization Code + PKCE S256 flow, RS256
ID-token/JWKS output, refresh rotation, RP logout redirect, and protocol
negative cases. The result remains `candidate-only`; the candidate is not
wired to the normal API or Keycloak route.

## Implemented boundary

- `internal/provider/candidate.go` constructs the selected library's OP with
  one exact web client, one exact redirect, one exact post-logout redirect,
  RS256, a stable `kid`, and no request-object or insecure grant fallback.
- The candidate `/authorize` boundary requires `code_challenge_method=S256`
  for every code request. Its login harness authorizes only the configured
  synthetic subject and returns control to the library callback.
- The storage adapter keeps authorization requests/codes single-use,
  returns only opaque tokens, hashes refresh credentials, and rejects replay,
  wrong client, wrong redirect, and wrong PKCE verifier through the library
  protocol path.
- The existing API RP remains separate and uses `coreos/go-oidc/v3` plus
  `golang.org/x/oauth2`; it validates discovery issuer, state/nonce/PKCE,
  RS256/JWKS, claim authority, clock bounds, and provider logout origin.

## Fresh verification

| Command | Result |
| --- | --- |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test ./internal/provider -count=1 -v` | `verified locally` — discovery, endpoints, JWKS `kid`/alg/use, code + PKCE, ID-token nonce/claims/signature, code replay, refresh rotation/reuse, and logout passed |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test ./... -count=1` | `verified locally` — provider candidate and all auth packages passed |
| `GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-api-go-cache go test ./internal/identity -run 'TestRemoteOIDCProvider\|TestTask4RemoteOIDCProvider' -count=1 -v` | `verified locally` — API RP discovery, PKCE, verified claims, private discovery/public issuer binding, JWKS rotation, expiry, and clock-boundary negatives passed |

Wrong redirect and missing-PKCE requests return a client error without an
attacker-controlled redirect. Unknown/retired key, full browser/real SMTP,
key-ring overlap through the OP, and independent security review are `not run`
or belong to later release gates; no production OIDC or dual-issuer fallback is
claimed.

## Fresh durable storage continuation (2026-08-11)

`internal/provider.PostgresStorage` now implements the selected
`zitadel/oidc` storage contract against the privileged `auth_identity` schema.
Forward-only `000007_oidc_runtime.up.sql` stores authorization and refresh
credentials only as SHA-256 hashes, keeps access/revocation state durable, and
uses encrypted signing-key material with exact persisted client records.
`scripts/test-auth-candidate-postgres.sh` is `verified locally` for client
secret rejection, encrypted key retrieval, authorization-code state, and
refresh rotation/reuse denial. It remains `candidate-only`: no OIDC, login,
MFA, recovery, reset, or logout HTTP route is mounted.

The isolated `RuntimeCandidate` now owns `/login` and `/mfa` against the
durable adapters. Its password path stages, but does not authorize, the OIDC
request when MFA is enabled; TOTP or a hashed recovery code must complete
before the library callback. The disposable PostgreSQL suite is `verified
locally` for both no-factor login and MFA-required login. `cmd/auth` still
mounts only health routes, so no browser/runtime qualification is implied.

## Runtime continuation (2026-08-12)

`cmd/auth` now initializes the durable PostgreSQL identity, MFA, OIDC storage,
and admission limiter before mounting `RuntimeCandidate`; readiness is opened
only after that initialization. `scripts/test-auth-candidate-runtime.sh` is
`verified locally` for an isolated PostgreSQL, authenticated STARTTLS Mailpit,
and auth topology plus private-network liveness/readiness/discovery checks.
The runner removed all task-owned containers, volumes, and secret material.
This remains `candidate-only`: provider-owned recovery, reset, and explicit
logout handlers and browser qualification are `not run`; Keycloak remains the
serving provider and rollback baseline and release remains `release pending`.

## Durable signing-key rotation qualification (2026-08-12)

`scripts/test-auth-candidate-postgres.sh` is `verified locally` for the
PostgreSQL signing-key ring: a newly encrypted RSA key becomes active, the old
key stays in finite JWKS overlap, and an elapsed overlap is retired. Invalid,
same-ID, and unbounded-overlap rotation requests are rejected. This remains
`candidate-only`; runtime key-custody operations and release evidence are `not
run`. Keycloak remains serving and the rollback baseline; release remains
`release pending`.
