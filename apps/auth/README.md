# AviaSurveil360 first-party OIDC provider scaffold

This directory is the isolated Go provider authorized on 2026-08-11 with
`github.com/zitadel/oidc/v3` v3.47.5. Task 2 began as a liveness-only,
fail-closed scaffold. Tasks 3–10 now provide candidate-only identity,
session, OIDC protocol, MFA/recovery, audit/admin, API-boundary, and synthetic
qualification components. The normal process and Compose profiles still do
not serve those candidate routes.

The provider is a separate trust boundary. It owns future credential, factor,
signing-key, identity-database, SMTP, and provider-session authority. The
ordinary API and worker do not receive these secrets. Each application will
remain an OIDC relying party using `coreos/go-oidc` and `golang.org/x/oauth2`,
while its own BFF keeps browser sessions and application authorization.

## Required startup contract

All secret values are supplied through read-only files. Inline credential
environment variables are rejected. The scaffold requires:

- `AVIA_AUTH_ENVIRONMENT=local-candidate` or `test`;
- `AVIA_AUTH_PROFILE=isolated-candidate`;
- an explicit loopback/local issuer, address, dedicated PostgreSQL URL file,
  role, and non-public schema;
- an RSA 2048–8192-bit private signing key, stable key ID, and
  `AVIA_AUTH_SIGNING_ALGORITHM=RS256`;
- distinct, non-zero 32-byte data-encryption and MFA key files; and
- a verified mailbox, SMTP host, STARTTLS or implicit TLS mode, and SMTP
  password file.

Placeholder, shared application/Keycloak database ownership, writable secret
files, plaintext SMTP, weak keys, unbounded request settings, and production or
normal-profile startup are rejected before the listener starts.

The isolated candidate Compose file is
[`deploy/local/auth/compose.auth-candidate.yaml`](../../deploy/local/auth/compose.auth-candidate.yaml).
It is never included by the normal local profiles, publishes no port, mounts
no Docker socket, and expects externally-created disposable Compose secrets.

## Local verification

```bash
GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test ./...
```

`/health/live` returns 200 after the process starts. `/health/ready` remains
503 until a separately authorized runtime wiring supplies validated durable
provider storage and key/SMTP readiness; candidate unit/protocol tests do not
change that boundary.
