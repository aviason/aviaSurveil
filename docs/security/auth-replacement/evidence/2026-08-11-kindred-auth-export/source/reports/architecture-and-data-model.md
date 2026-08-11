# Architecture and data model

## Boundary

The runtime loads one RSA private key, derives a thumbprint `kid`, builds a proprietary JWT signer/verifier and one-key JWKS document, then registers custom REST auth routes, GraphQL local routes, and public JWKS/discovery routes. API Gateway/AppSync consume the issuer metadata; they are not implemented inside this service.

```text
email/password + API key
          |
          v
   auth.Service + DynamoDB user row ----> mail/SMS adapters
          |             |
          |             +--> consent ledger / analytics
          v
 RS256 access JWT + opaque refresh token
          |
   local bearer middleware / AppSync identity adapter
          |
          v
 application GraphQL resolvers
```

The mobile source-reference client stores access and refresh tokens in Keychain and sends bearer headers. It does not use browser redirects, cookies, BFF sessions, PKCE, or ID tokens.

## User record and invariants

The DynamoDB user record contains an immutable UUID, normalized email, display name, Argon2id password hash, verification/reset token hashes and expiries, failed-login/lock state, account status, token version, and one refresh hash with idle/absolute expiry and device binding. Transactional email/phone uniqueness locks are used by the storage implementation. There is no role, permission, organization, tenant, OAuth client, consent-session, token-family, or multi-device-session table.

`TokenVersion` is checked after JWT parsing against the database. Logout and selected lifecycle operations bump it and clear the stored refresh state. Password reset closes sessions; the current password-change path changes only the hash, leaving an identified revocation gap.

## Lifecycle and privacy

Registration, verification/resend, login lockout, password reset/change, phone verification/change, deactivation/reactivation, deletion-pending and purge are in the copied auth/account-purge sources. The analytics consent ledger records application privacy purposes and deletion markers. It is not an OIDC scope-consent store. SMTP and SMS adapters are included; localized templates and production delivery controls are not implemented.

## Persistence and deployment

The project uses DynamoDB and optional DynamoDB Local tests. There are no PostgreSQL migrations, schema files, or PostgreSQL integration tests. Terraform exposes health/JWKS/discovery and can configure API Gateway JWT/AppSync OIDC consumer integrations. The sanitized infrastructure reference records route and configuration evidence without operational URLs or secret values.

## Compile boundary

The selected files are a cohesive review/export set, not a claim that the reduced tree is a standalone module. `source-reference/dependency-map.md` identifies omitted application packages needed by storage, GraphQL resolvers, test fakes, and deployment wiring. Those dependencies are unrelated business-domain code or contain excluded operational material; their auth-relevant contracts are documented rather than silently dropped.
