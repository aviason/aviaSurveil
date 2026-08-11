# OIDC/OAuth compatibility report

## Conclusion

The project is not an OIDC Provider/Authorization Server. It is a proprietary RS256 JWT/session issuer with a discovery/JWKS facade used by API Gateway and AppSync as a resource-server consumer. It is also not an OIDC relying party/client: the mobile and GraphQL paths use email/password, API-key mutations and bearer tokens without browser redirects or OAuth libraries.

The discovery document and JWKS endpoint are metadata/public-key plumbing only. Their existence must not be interpreted as OIDC compliance.

## Provider feature evidence

| Feature | Classification | Evidence |
|---|---|---|
| OpenID Provider discovery | partial | `internal/jwks/handler.go` emits only issuer, JWKS URI, RS256, `id_token` response type and public subject type. Authorization/token/userinfo/revocation/logout/PKCE metadata is absent. `handler_test.go` tests JWKS, not discovery. |
| Authorization endpoint | absent | No `/authorize` route or handler. |
| Token endpoint | absent | `/auth/refresh` is custom JSON and does not implement OAuth token grants. |
| Authorization Code flow | absent | No authorization code, redemption, consent or redirect flow. |
| PKCE S256 | absent | No challenge/verifier parsing or S256 comparison. |
| State and nonce | absent | No protocol state/nonce store or validation. |
| Exact redirect URI validation | absent | No registered clients or redirect URI comparison. |
| ID token issuance | absent | JWT is an access/session token with custom claims; `AuthResponse` contains no `id_token`. |
| ID token validation | absent | No `acr`, `amr`, `auth_time`, `nonce`, `at_hash`, `c_hash` or ID-token validator. |
| Access token | partial | RS256 bearer JWT is issued and verified, but it is not exposed through an OAuth grant/token endpoint. |
| Refresh token | partial | Opaque refresh token rotates in a single user row; no OAuth client/family semantics. |
| Refresh rotation/reuse | partial | Hash/device mismatch clears the one stored session; atomic family-wide behavior and explicit old-token tests are absent. |
| JWKS publication | implemented but not verified | `/.well-known/jwks.json` is served and has a response test. |
| Signing-key rotation | absent | One loaded RSA key and one JWK; no overlap/retirement/rotation implementation. |
| UserInfo | absent | Product profile APIs are not `/userinfo` and do not have OIDC claim semantics. |
| Revocation | absent | No RFC 7009 endpoint; logout is proprietary token-version invalidation. |
| RP-initiated logout | absent | No OIDC logout endpoint or post-logout redirect validation. |
| Client registry/authentication | absent | No OAuth client records, secrets, JWT client auth, or dynamic registration. |
| Issuer/audience/azp | partial | Issuer and audience are checked by the local proprietary verifier; `azp` is not checked. AppSync identity parsing relies on external validation. |
| Scopes and claims | absent | No scope negotiation or standard claim selection; custom `email`, `tv`, `did` claims only. |
| Session and consent | partial | App privacy consents are recorded, but no OAuth authorization session or scope-consent behavior. |
| Negative protocol tests | absent | No protocol conformance/negative test suite. |

## Consumer boundary

Terraform configures API Gateway JWT authorization and AppSync `OPENID_CONNECT` using the service issuer/JWKS. `internal/graphql/event.go` consumes already-validated AppSync identity claims (`sub`, `email`, `tv`, `did`) and does not implement an OIDC client or independently validate issuer/audience/azp. The local HTTP adapter verifies the proprietary JWT directly. These are resource-server integrations, not provider or RP implementations.

## Compatibility blockers

Implementing a conforming provider would require a client registry, exact redirect URI checks, `/authorize` and `/token`, code+PKCE, state/nonce, ID-token semantics, UserInfo, scopes/claims, revocation/logout, consent/session state, client authentication, key rotation, and protocol tests. None are present in this export.
