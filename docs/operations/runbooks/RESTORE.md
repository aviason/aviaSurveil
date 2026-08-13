# Isolated Restore Runbook

This runbook covers only maintained local component restore checks. The result
is `candidate-only`; release is `release pending`.

## Catalog validation

Use a unique source project and exact recovery-point ID:

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-recovery-source"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
export RECOVERY_POINT_ID="rp-YYYYMMDDTHHMMSSZ-name"
./scripts/verify-backup-catalog.sh "$RECOVERY_POINT_ID"
```

Catalog validation must fail before target mutation when a component, checksum,
retention field, identity fingerprint, or secret reference is missing or
inconsistent.

## First-party auth restore check

```bash
./scripts/test-auth-candidate-backup-restore.sh
```

This task-owned check dumps, removes, and restores only its disposable auth
schema, then verifies readiness and synthetic account state. It must clean its
containers, volumes, network, and temporary secret directory.

## Current limit

The prior full-stack restore harness is retired. Application/object/browser
restore and measured RPO/RTO are `not run`; do not claim them from catalog or
auth-only evidence.

Never target an active project, mount a source database or primary object
volume into a target, print credentials/MFA material, or use broad cleanup.
Production restore, remote stores, retention changes, real identities, shared
targets, and destructive source actions require separate explicit authority.
