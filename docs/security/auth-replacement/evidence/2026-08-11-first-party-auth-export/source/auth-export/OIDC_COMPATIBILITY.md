# OIDC and Keycloak compatibility

## Classification

The source is **native password/session authentication with a proprietary JWT
issuer**. It is neither an OIDC relying party/client nor a conforming OIDC/OAuth
authorization server.

Publishing a JWKS does not make a service an authorization server. Publishing
discovery-shaped JSON does not make it OpenID Connect compliant.

## Protocol inventory

| OIDC/OAuth element | Status | Source behavior |
|---|---|---|
| External issuer discovery | absent | No discovery fetch or external issuer configuration. |
| Remote JWKS fetch/cache/refresh | absent | Verifies only locally configured RSA keys. |
| Authorization endpoint | absent | No browser redirect or consent flow. |
| OAuth token endpoint | absent | Login/refresh are proprietary GraphQL operations. |
| Authorization Code grant | absent | No code issuance/exchange. |
| PKCE | absent | No code challenge/verifier. |
| `state` | absent | No authorization redirect flow. |
| OIDC `nonce` | absent | No ID token or authorization request. |
| Redirect URI registration/validation | absent | No client/redirect registry. |
| ID token | absent | Issued JWT is an application access token. |
| UserInfo endpoint | absent | Dormant legacy user-info REST code is not OIDC UserInfo and is not mounted. |
| Client authentication/credentials | absent | Client IDs are a static allowlist, not registered OAuth clients; no service accounts or client secret flow. |
| Scopes/consent | absent | Metadata advertises scopes, but no scope authorization or consent system exists. |
| Token introspection | absent | Resource middleware validates local JWT/session state directly. |
| OAuth token revocation endpoint | absent | Proprietary logout/session mutations revoke local state. |
| RP-initiated/back-channel logout | absent | No OIDC logout protocol. |
| Discovery-shaped metadata | partial | Issuer/JWKS plus unsupported response/scopes/ID-token claims are advertised. |
| JWKS publication | implemented and verified locally | Local active and previous RSA public keys are emitted and unit-tested. |

## Existing Keycloak expectations

Any AviaSurveil360 component that currently expects the following will break if
pointed at this export:

- issuer discovery containing `authorization_endpoint`, `token_endpoint`, and
  supported grant/client authentication metadata;
- authorization-code redirects with PKCE, state, nonce, and registered redirect
  URIs;
- ID tokens and UserInfo claims;
- realm/client roles or protocol mappers;
- Keycloak admin/service-client endpoints and client credentials;
- federated logout, MFA/recovery, account console, or localized hosted pages.

The existing Keycloak issuer and this native issuer must not share an issuer
URL. Verifiers must not try one and silently fall back to the other.

## Safe options

### Keep OIDC

Retain Keycloak or adopt a mature conforming Go-compatible authorization server
instead of extending these primitives into a home-grown protocol server. The
export can still inform password/session or application authorization design,
but the IdP remains the identity protocol authority.

### Deliberately leave OIDC

Use a first-party same-origin BFF/native session contract, remove OIDC discovery
claims, replace all Keycloak SDK/admin calls, and migrate clients explicitly.
Document the proprietary token claims and key lifecycle as an internal API.
This choice still requires implementing all missing identity controls in the
security review.

## Metadata mismatch

The source `OpenIDConfiguration` advertises:

- `response_types_supported: ["token"]`;
- `scopes_supported: ["openid", "profile", "offline_access"]`; and
- `id_token_signing_alg_values_supported: ["RS256"]`.

There is no endpoint capable of processing that response type, no OIDC scope
flow, and no ID token issuance. AviaSurveil360 must not copy these claims. If a
native metadata endpoint is useful, give it a non-OIDC path and schema.
