# AviaSurveil360 First-Party OIDC Provider

`apps/auth` is the repository's maintained identity provider. It is a separate
Go trust boundary using `github.com/zitadel/oidc/v3`.

The current result is `candidate-only`; release is `release pending`.

## Ownership

The provider owns credentials, account verification, MFA, recovery,
authorization requests/codes, signing keys, provider sessions, provider
profiles, application-authority mirrors, idempotent private-admin receipts, and
redacted audit events in a dedicated PostgreSQL database.

Applications remain OIDC relying parties. The AviaSurveil360 API acts as a BFF,
keeps provider tokens server-side, and evaluates application membership in its
own PostgreSQL database.

## Listeners

- Port 8080 serves public OIDC, account UI, liveness, readiness, and discovery.
- Port 8081 serves private provider administration.
- The private listener requires a separate mounted high-entropy bearer secret
  and is never exposed by the gateway.
- Private mutations require `Idempotency-Key`, bounded JSON, canonical request
  hashing, expected/resulting revisions, and replay/conflict semantics.

## Local-preprod contract

The canonical disposable topology supplies all credentials through read-only
files, uses dedicated auth PostgreSQL, and sends privileged auth mail through a
separate Mailpit with authenticated mandatory STARTTLS.

The provider issues no refresh token and rejects `offline_access`. Private
authority claims are issued only for active, verified accounts with active
authority. Password, MFA, authority, disable/suspend, and activation mutations
revoke old credentials and sessions.

## Local verification

```bash
GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache go test ./...
./scripts/test-auth-candidate-runtime.sh
./scripts/test-auth-candidate-backup-restore.sh
```

Canonical application/BFF verification is owned by the active auth replacement
ExecPlan and `scripts/test-canonical-preprod-fault-restart.sh`.

No remote system, deployment, traffic, production secret, or real user is
authorized by this README.
