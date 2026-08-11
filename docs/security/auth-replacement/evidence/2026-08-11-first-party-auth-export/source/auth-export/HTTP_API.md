# HTTP and GraphQL API

## Active source boundary

The source runtime mounts:

- `POST /graphql` for every registration, login, refresh, logout, session, and
  password operation;
- `GET /.well-known/openid-configuration` for discovery-shaped metadata; and
- `GET /oauth2/jwks` for local RSA public keys.

The REST routes in `auth/source_reference/legacy_handler.go.txt` are dormant:
the runtime calls `RegisterMetadata`, not `Register`. They are included only to
show prior adapter shapes and must not be reported as active API support.

## Persisted auth operations

| Operation name | GraphQL field | Authentication |
|---|---|---|
| `Auth_Register` | `authRegister(input)` | Public |
| `Auth_Login` | `authLogin(input)` | Public |
| `Auth_Refresh` | `authRefresh(input)` | Public, possession of refresh value plus matching client/device strings |
| `Auth_Logout` | `authLogout(input)` | Bearer session required |
| `Auth_LogoutAll` | `authLogoutAll(input)` | Bearer session plus recent password step-up |
| `Auth_RevokeSession` | `authRevokeSession(sessionID)` | Bearer session plus recent password step-up; target is scoped to caller |
| `Auth_Sessions` | `authSessions` | Bearer session required |
| `Auth_StepUp` | `authStepUp(input)` | Bearer session plus password |
| `User_ChangePassword` | `changePassword(input)` | Bearer session plus recent password step-up and current password |
| `Auth_Viewer` | `viewer` | Bearer-authenticated viewer in the broader source schema |

The auth-only schema types are preserved in
`auth/source_reference/auth.graphqls.txt`. The full application schema is
excluded because it is unrelated to this export.

## Token transport

Login and refresh return an `AuthPayload` containing an access JWT, an opaque
refresh value, expiry timestamps, and user fields in JSON. Protected requests
use `Authorization: Bearer <access-token>`. Logout may also receive a refresh
value in the GraphQL body.

The source does not set `Cache-Control: no-store`, set cookies, implement CSRF,
or define a browser token-storage contract. AviaSurveil360 should not copy this
transport unchanged into React. Prefer a same-origin BFF with a server-managed
refresh credential and protected cookie, or explicitly accept and mitigate the
XSS exposure of browser-accessible tokens.

## Access JWT contract

Locally issued tokens contain `iss`, `sub`, `aud`, `azp`, `client_id`, `exp`,
`iat`, `auth_time`, `jti`, `sid`, `scope`, `role`, `user_id`, and
`auth_version`. The verifier requires `RS256`, a recognized `kid`, a valid RSA
signature, exact issuer, an allowed audience and client, unexpired time claims,
a random token/session identifier, and positive auth version. Middleware then
checks the PostgreSQL session and current account auth version.

This is a proprietary access-token contract. It is not an ID token, OAuth token
response, or proof of OIDC conformance.

## chi adapter

`examples/chi_mount.go` uses the `MethodFunc` subset implemented by chi's
router. It mounts only metadata/JWKS because the actual identity handlers must
be rewritten around AviaSurveil360's request models, security controls, and
browser architecture.

## Required HTTP controls before exposure

- Apply byte limits before JSON/GraphQL decoding and configure server read,
  write, header, and idle timeouts.
- Require the reviewed persisted-operation document set and apply query
  depth/complexity/input limits.
- Add password-operation rate/concurrency limits and generic failure timing.
- Set `Cache-Control: no-store` on credential responses.
- Define exact same-origin/CORS and cookie/CSRF behavior.
- Enforce active account and organization membership centrally before protected
  resolver execution.
- Keep health minimal and readiness dependent on all mandatory auth services.
