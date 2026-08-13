# Local Secret Rotation

Local secret rotation uses a parallel fresh task-owned namespace. The result is
`candidate-only`; release is `release pending`.

## Scope

The generated local set includes application/auth database credentials,
first-party signing/data-encryption/MFA keys, OIDC client and private-admin
secrets, application and auth SMTP credentials/certificates, session keys,
MinIO credentials, backup credentials, and recovery TLS material.

Never rotate a persistent namespace in place. Never print, compare, or attach
secret values.

## Parallel replacement

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-rotation-new"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
export AVIA_LOCAL_HTTPS_PORT="28443"
./scripts/local-stack.sh up full
./scripts/local-stack.sh check full
```

Initialization is create-only. It generates the first-party auth namespace
under the task-owned state root, protects secret files, and refuses partial or
unexpected existing material.

Verify OIDC/password/MFA/recovery, private-admin denial from the public origin,
database/object/worker/SMTP health, negative secret-log scans, and exact cleanup
before recording `verified locally`. If validation fails, remove only the new
project and keep the prior local namespace unchanged.

Capture only reference names, project identities, permissions, UTC timeline,
health, negative log scan, owner, and disposition. In-place encryption-key
replacement, old namespace destruction, user credential resets, production
secrets, external secret managers, remote systems, deployment, or traffic
require separate explicit authority.
