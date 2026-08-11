# Local Secret Rotation

This runbook uses a parallel fresh stack to replace local `candidate-only`
credentials. It is not production-ready and rotation is `not run` until the
new stack passes and the old stack remains available for rollback.

## Scope And Owner

Owner: Security

Escalation owner: Platform/Operations and Release authority

Scope includes file-backed local application, database, Keycloak, MinIO, SMTP,
session, OIDC, backup, Grafana, and recovery TLS credentials.

## Preconditions

- Identify the old exact project and state directory without reading secret
  values.
- Keep the repository-supported Node.js runtime available on `PATH` so the
  generated Keycloak realm can be built.
- Choose a different project, state directory, and HTTPS port for the new
  stack.
- Confirm capacity to keep old and new stacks isolated during validation.
- Record affected credential references and rotation owner.

## Symptoms

- A local secret may be exposed, expired, mismatched, or due for rotation.
- A service fails authentication after an attempted credential change.
- Generated secret material appears in logs or evidence.

## Safety Boundary

- Do not rotate database, identity, object-store, or encryption credentials in
  place with `init-local-secrets.sh --rotate`; dependent persistent services
  require coordinated changes.
- Never print, compare, or attach secret values.
- Keep the old stack unchanged until the new stack passes all checks.

## Diagnosis

Verify permissions and references without displaying file contents:

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-rotation-new"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
test ! -e "$AVIASURVEIL_LOCAL_STATE_DIR"
./scripts/check-local-image-evidence.sh full
```

## Expected Output

The new state path is absent before creation. Startup generates protected local
credential files, services authenticate through file references, and logs
contain no generated values.

## Reversible Mitigation

Create a parallel stack with fresh credentials:

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-rotation-new"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
export AVIA_LOCAL_HTTPS_PORT="28443"
export AVIA_LOCAL_PUBLIC_ORIGIN="https://localhost:$AVIA_LOCAL_HTTPS_PORT"
./scripts/local-stack.sh up full
./scripts/local-stack.sh check full
```

If validation fails, remove only the new project and continue using the
unchanged old local stack.

## Recovery Verification

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-rotation-new"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
./scripts/local-stack.sh check full
```

Require OIDC/TOTP, database, object, worker, SMTP, browser, secret-log scan, and
cleanup evidence before recording `verified locally`.

## Evidence Capture

Capture only credential reference names, old/new project identities, state
permissions, UTC timeline, service health, negative secret-log scan, owner, and
decision. Never capture secret values.

## Escalation

Escalate suspected exposure to Security immediately. Escalate dependency
authentication failure to Platform/Operations and its service owner. Release
authority decides retirement of the old candidate.

## Authorization Required

In-place persistent credential changes, encryption-key replacement, old-stack
destruction, production secret access, external secret managers, user
credential reset, and AWS secret operations require new explicit
authorization.

## AWS Private-Pilot Boundary

The private-pilot contracts separate the Cloudflare connector token,
application/Keycloak database credentials, OIDC/session credentials, SMTP
credentials. Dormant data-feed key material is not mounted by the private-pilot
runtime. There are no origin-auth header secrets. The connector lives in one
KMS-encrypted Standard SSM SecureString;
Terraform creates only the write-only non-runnable placeholder and must never
read the real token into plan or state. Rotation requires a separately
authorized SSM write, connector restart, four-connection health proof, and a
rollback/revocation window. Runtime refuses the placeholder and exposes the
token only as a connector-UID `0400` file without logging it. No production
secret value is committed or populated by local preparation. Future AWS secret
operations require profile `avia` in `eu-central-1` and separate exact
authorization; EC2 runtime access remains instance-profile scoped.
