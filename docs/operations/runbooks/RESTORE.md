# Isolated Restore Runbook

This procedure is `candidate-only`, is exercised as `verified locally`, and is
not production-ready. It restores one complete local recovery point into a new
Docker Compose project; it does not certify host-loss recovery.

## Scope And Owner

Owner: Platform/Operations

Escalation owner: Security and Release authority

Scope is one complete local recovery point restored into a new, exact,
task-owned project with independent state and HTTPS port.

## Preconditions

- A complete catalog exists under the source local state directory and its
  application database, identity database, identity fingerprint, object
  manifest, and configuration-reference components are present.
- The source recovery-only MinIO service is healthy and the accepted recovery
  image digest has current SBOM and HIGH/CRITICAL scan evidence.
- The operator has unique restore project, state-directory, HTTPS-port, and
  evidence-path values.
- The browser proof account password and separately protected TOTP seed are
  available for the local drill.

## Symptoms

- A restore is required to validate one recovery point.
- The latest catalog is absent, partial, corrupt, expired, or fingerprint
  inconsistent.
- A restored dependency, worker, API restart, TOTP login, role scope, or route
  load fails.

## Safety Boundary

- Never use an active Compose project name or state directory as a restore
  target.
- Never mount the source application database, Keycloak database, or primary
  object-store volume. The restore reads only the logically isolated backup
  store, the immutable catalog, and required file-based secret references.
- Never use broad Docker cleanup. Cleanup is restricted to the exact restore
  project label.
- A checksum mismatch, absent component, identity mismatch, browser failure, or
  cleanup failure makes the drill unsuccessful. Partial success is forbidden.

## Procedure

Validate the named catalog, start one uniquely named isolated target, restore
the application database, identity database, exact object versions, and
configuration references, then run the fingerprint, worker, restart, MFA/role,
and 86-route checks. Stop before target creation if catalog validation fails.

## Diagnosis

Validate one complete catalog and source project before creating the target:

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-recovery-source"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
export RECOVERY_POINT_ID="rp-20260726T120000Z-ops"
./scripts/verify-backup-catalog.sh "$RECOVERY_POINT_ID"
```

## Expected Output

The evidence is labeled `verified locally` and `candidate-only`. It contains
start/end timestamps, selected recovery time, separate database and object RPO,
RTO, calculated data loss, expected/actual fingerprints, five scenario
results, an unskipped 86-route browser result, and cleanup status.

## Reversible Mitigation

Restore into an isolated target only:

```bash
export AVIA_RESTORE_SOURCE_PROJECT="aviasurveil360-task-recovery-source"
export AVIA_RESTORE_SOURCE_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_RESTORE_SOURCE_PROJECT"
export AVIA_RESTORE_PROJECT="aviasurveil360-restore-ops-example"
export AVIA_RESTORE_STATE_DIR="/private/tmp/$AVIA_RESTORE_PROJECT"
export AVIA_RESTORE_HTTPS_PORT="29443"
export AVIA_RESTORE_EVIDENCE_PATH="/private/tmp/$AVIA_RESTORE_PROJECT-evidence.json"
./scripts/restore-isolated-stack.sh "rp-20260726T120000Z-ops" "$AVIA_RESTORE_PROJECT"
```

The command removes only the exact target resources. It leaves the source
unchanged and fails closed before target mutation for invalid catalogs.

## Recovery Verification

Require exact application, identity, object, and configuration fingerprints;
restored worker completion; API restart; normal OIDC/TOTP role scope; 86 direct
route loads; bounded RPO/RTO; and zero target residue.

## Evidence Capture

Preserve the JSON evidence outside the temporary target state. Capture recovery
point, source/target identities, timestamps, fingerprints, RPO/RTO, scenario
results, browser route count, corruption response, and cleanup status. Exclude
credentials and TOTP material.

## Expected Evidence

The retained evidence must match the selected complete recovery point, record
the exact source and isolated target identities, prove all recovery checks, and
carry the literal `verified locally` and `candidate-only` labels.

## Cleanup

Remove only the exact restore Compose project, its task-owned state directory,
and its temporary browser output. Confirm no container, volume, network, or
temporary credential remains for that project; never prune shared Docker state.

## Escalation

Escalate missing or corrupt components to Platform/Operations and Security.
Escalate identity/TOTP or role-scope mismatch to Identity and Security. Any
records-policy question goes to Records/Legal.

## Authorization Required

Production restore, remote backup store, retention change, destructive source
action, shared target, identity change, and AWS recovery require new explicit
authorization.
