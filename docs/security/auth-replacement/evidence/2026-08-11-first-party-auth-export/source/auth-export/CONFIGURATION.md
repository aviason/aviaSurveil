# Configuration

No environment file is included. Values below are names and source semantics,
not deployable settings. Secrets must come from AviaSurveil360's approved secret
store and must never be committed, logged, placed in URLs, or copied into the
export.

| Name | Source behavior | AviaSurveil360 requirement |
|---|---|---|
| `APP_ENV` | Defaults to `development`; exact development mode enables ephemeral key generation. | Normalize once and allow only an explicit enum. Fail startup for any production-incompatible value. |
| `DATABASE_URL` | PostgreSQL DSN consumed by the source database package. | Inject through a secret store; require TLS and the least-privileged application role. Never print the DSN. |
| `AUTH_ISSUER_URL` | Defaults inside `TokenService` to an HTTP localhost issuer. | Required, canonical external HTTPS issuer if proprietary JWTs remain. Do not rely on the default. |
| `AUTH_JWT_PRIVATE_KEY_PEM` | Inline RSA private PEM. | Avoid inline environment material when file/secret-volume/KMS custody is available. Never include it in an export. |
| `AUTH_JWT_PRIVATE_KEY_FILE` | Path to RSA private PEM. | Required alternative to inline PEM; restrict filesystem ownership/mode and validate at startup. |
| `AUTH_JWT_KEY_ID` | Active `kid`; otherwise derived from the public key. | Stable, explicit, unique ID tied to a rotation record. |
| `AUTH_ACCESS_TOKEN_AUDIENCE` | CSV; source falls back to app-sync client, then `emsi-api`. | Required explicit Avia audience; reject EMSI defaults. |
| `AUTH_APPSYNC_CLIENT_ID` | Optional client added to audience/allowlist. | Remove unless Avia has a separately reviewed equivalent. |
| `AUTH_ALLOWED_CLIENT_IDS` | CSV; source defaults to EMSI native client IDs. | Required explicit allowlist. Validate before session creation. This is not an OAuth client registry. |
| `AUTH_JWT_PREVIOUS_PUBLIC_KEYS_PEM` | PEM bundle of previous RSA public keys. | Rotation overlap only. The source derives each previous `kid`; preserve configured historical IDs in a redesigned key ring. |
| `AUTH_JWT_PREVIOUS_PUBLIC_KEYS_FILE` | File alternative for previous public keys. | Use protected, versioned public key material; fail startup on parse errors. |
| `AUTH_ALLOW_DEV_HEADER` | Enables `X-User-ID`/`X-User-Role`; defaults false and source config rejects it outside local/development/test. | Compile out or hard-reject for the private EC2 runtime. A development `APP_ENV` must never be internet-reachable. |
| `AUTH_ACCESS_TOKEN_TTL_SECONDS` | Default 900 seconds. | Choose explicitly; source default is reasonable only with server-side validation and a protected refresh path. |
| `AUTH_REFRESH_TOKEN_TTL_SECONDS` | Default 2,592,000 seconds (30 days). | Choose an absolute lifetime based on product risk; revoke all pre-change credentials. |
| `AUTH_MEMBER_REFRESH_IDLE_TTL_SECONDS` | Default 1,209,600 seconds (14 days). | Make active-user classification explicit and test idle extension semantics. |
| `AUTH_STAFF_REFRESH_IDLE_TTL_SECONDS` | Default 43,200 seconds (12 hours). | Define privileged roles from authorization data, not “anything other than member.” |
| `AUTH_STEP_UP_TTL_SECONDS` | Default 600 seconds. | Fix enforcement so every sensitive operation uses this exact value. |
| `AUTH_JWT_SECRET` | Loaded by source config but unused by the RS256 token service. | Remove; do not create a misleading fallback or use it as an HMAC key. |
| `GRAPHQL_PERSISTED_ONLY` | Source default depends on an unnormalized environment string and explicit false is accepted. | Require true for exposed GraphQL, validate fail-closed, and also apply body/depth/complexity limits. |
| `AVIASURVEIL360_AUTH_TEST_DATABASE_URL` | Export integration-test convention only. | Point only to a disposable task-owned database whose database name contains `aviasurveil360_auth_test`. |

## Required startup validation

Before constructing handlers, validate all of the following and return a fatal
startup error:

- production environment identity and persisted-query policy;
- HTTPS canonical issuer, exact audience, and nonempty allowed clients;
- one usable RSA private key of an approved minimum size and a consistent key
  ring with stable `kid` values;
- database connectivity, schema version, and required auth tables;
- `AUTH_ALLOW_DEV_HEADER=false`;
- a configured `AuditSink`, `MailSender`, clock, cryptographic random source,
  password-work limiter, and organization resolver;
- locale catalogs/templates for English, Turkish, French, and Portuguese;
- no inline or file value accidentally contains an example/test placeholder.

Readiness must fail if signing, verification, database/schema, audit, or any
mandatory identity dependency is unavailable. Liveness should remain a narrow
process-health check.

## SMTP boundary to add

The source has no mail support. AviaSurveil360 must require an external SMTP
host and port, authenticated credentials from secret storage, a verified sender,
hostname verification, modern TLS, and either implicit TLS or mandatory
STARTTLS with downgrade failure. Do not support opportunistic plaintext
fallback. Add bounded timeouts, queue/retry policy, redacted delivery audit, and
templates selected from `en`, `tr`, `fr`, and `pt` catalogs. Do not put reset
tokens in logs, telemetry, subject lines, or query strings observed by third
parties beyond the necessary one-time reset URL.

## Edge notes

Trust forwarded client addresses only from the known Caddy/Cloudflare hop chain.
Set request-body, header, read, write, and idle limits in the Go server as well
as at the edge. Keep metrics and administrative endpoints off the public
tunnel. No CORS policy is exported; prefer the same origin for the React/Vite
client and API, or define a strict fixed-origin policy with credential semantics
that match the chosen browser architecture.
