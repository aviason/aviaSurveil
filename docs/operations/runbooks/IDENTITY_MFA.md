# Local First-Party Identity And MFA Recovery

This runbook covers only the disposable first-party Go OIDC topology. The
result is `candidate-only`; release is `release pending`.

## Scope and ownership

Identity owns account, password, MFA, recovery, signing-key, provider-session,
and authority state in auth PostgreSQL. The application owns membership and BFF
session state in application PostgreSQL. Provider administration listens on
port 8081 only inside the Compose network and must never be exposed by the
gateway.

Use only synthetic `@synthetic.invalid` users. Real users, remote systems,
external SMTP, deployment, traffic, and production secrets are outside this
runbook.

## Start and observe

For the named canonical disposable profile:

```bash
make preprod-up
make preprod-status
```

For a uniquely named full profile:

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-identity-check"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
./scripts/local-stack.sh up full
./scripts/local-stack.sh check full
```

Public discovery is available only through the gateway's `/identity` prefix.
Port 8081 has no host publication and no gateway route. Never copy or print the
admin secret.

## Recovery and MFA behavior

Recovery initiation must return a generic response. Activation, password reset,
and MFA reset consume one-time state transactionally and are replay-safe.
Password changes and MFA resets increment `auth_revision` and revoke provider
sessions. Authority, disable/suspend, and activation changes revoke old
credentials and application sessions.

The canonical browser path must use accessible labels for username/email,
password, one-time code, recovery, password reset, and MFA reset. It must not
depend on provider-specific DOM selectors.

## Verification

```bash
go -C apps/auth test -count=1 ./...
go -C apps/api test -count=1 ./internal/identity ./internal/platform/session
./scripts/test-auth-candidate-runtime.sh
./scripts/test-preprod-identity-lifecycle.sh
make preprod-test-fault-restart
```

Record each command literally as `verified locally`, `not run`, or
`blocked`. Verify that browser storage contains no provider token, old BFF
sessions fail after lifecycle/revision changes, public admin paths are denied,
and task-owned containers, volumes, networks, browser profiles, and processes
are removed.

## Restart

Restart only the exact task-owned auth service:

```bash
docker compose \
  --project-name "$AVIA_LOCAL_PROJECT" \
  --file deploy/local/compose.yaml \
  --profile full \
  restart preprod-auth
./scripts/local-stack.sh check full
```

A restart must preserve durable accounts, authority, MFA, signing keys, and
sessions according to their explicit lifecycle. Dependency loss must make
readiness fail closed and recovery must restore readiness.

## Escalation and authorization

Escalate authority drift, replay conflict, credential leakage, unexpected
session survival, or public access to port 8081 to Identity and Security.
Production credential operations, real-user mutation, remote SMTP, deployment,
traffic, or external identity migration require separate explicit authority.
