# Local Recovery Point Backup

This procedure creates same-host logically isolated `candidate-only` recovery
evidence. It is not production-ready and remains `not run` until its catalog
and components verify.

## Scope And Owner

Owner: Platform/Operations

Escalation owner: Security and Records/Legal

Scope includes application PostgreSQL, Keycloak PostgreSQL plus identity
fingerprint, versioned application objects, configuration references, and one
immutable recovery catalog.

## Preconditions

- The exact source full/recovery project is healthy and owned by the operator.
- Accepted backup images have current digest, SBOM, and vulnerability evidence.
- Select a unique `rp-YYYYMMDDTHHMMSSZ-name` recovery-point ID.
- Confirm local retention capacity and that no component lock is active.

## Symptoms

- Incremental recovery-point age exceeds 30 minutes.
- Full or differential recovery data exceeds 26 hours.
- A component, checksum, retention value, identity fingerprint, or catalog is
  absent or inconsistent.

## Safety Boundary

- A partial component set must never be published as complete.
- Backup objects and published catalogs are immutable.
- Same-host logical isolation does not prove host-loss, remote-site, account,
  region, or legal-retention recovery.

## Diagnosis

Create and verify a disposable isolated backup profile:

```bash
./scripts/test-backup-profile.sh
```

For an owned recovery stack, verify one exact catalog without recreating
components:

```bash
export RECOVERY_POINT_ID="rp-20260726T120000Z-ops"
./scripts/verify-backup-catalog.sh "$RECOVERY_POINT_ID"
```

## Expected Output

Each component reports `verified locally`; the catalog is complete,
`candidate-only`, checksum-bound to the exact recovery-point ID, retained, and
labeled `same-host-logically-isolated`.

## Reversible Mitigation

When freshness is the only fault, create a new point instead of changing an
older point:

```bash
export RECOVERY_POINT_ID="rp-20260726T120000Z-ops"
./scripts/verify-backup-catalog.sh --create "$RECOVERY_POINT_ID" incr
```

If any component fails, leave the incomplete component directory unpublished,
capture evidence, and correct the dependency before choosing a new ID.

## Recovery Verification

```bash
./scripts/test-rpo-rto-drill.sh
```

Require exact database, identity, object, and configuration fingerprints,
bounded RPO/RTO, restored worker and browser checks, corruption fallback, and
zero residue before recording `verified locally`.

## Evidence Capture

Capture recovery-point ID, component and catalog hashes, backup type and
labels, UTC timestamps, retention timestamp, database/object RPO, RTO, failure
domain, and cleanup result. Record secret references, never secret values.

## Escalation

Escalate component or freshness failure to Platform/Operations; integrity or
secret concerns to Security; retention and legal-hold questions to
Records/Legal.

## Authorization Required

Backup deletion, retention shortening, immutability change, remote store
configuration, production credentials, production restore, and AWS resources
require new explicit authorization.

## AWS Private-Pilot Boundary

The local contract models 14-day RDS PITR and 35-day AWS Backup coverage for
the one RDS instance and versioned S3 buckets. It does not prove restore,
cross-failure-domain durability, or the stated RPO/RTO. Those checks remain
`not run` and require separate Task 7 authorization. Any future operator-side
AWS command must select profile `avia`; `default` is forbidden.
