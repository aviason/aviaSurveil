# Local Recovery Point Backup

This procedure creates same-host logically isolated first-party recovery
evidence. It remains `candidate-only` and `release pending`.

## Scope

The maintained backup boundary covers application PostgreSQL, first-party auth
PostgreSQL and its identity fingerprint, versioned application objects,
configuration references, and one checksum-bound recovery catalog.

A partial component set must never be published as complete. Backup objects and
published catalogs are immutable. Same-host evidence does not prove host-loss,
remote-site, account, region, legal-retention, or production recovery.

## Verification

Create and verify a disposable recovery profile:

```bash
./scripts/test-backup-profile.sh
```

Verify one existing exact recovery point:

```bash
export RECOVERY_POINT_ID="rp-YYYYMMDDTHHMMSSZ-name"
./scripts/verify-backup-catalog.sh "$RECOVERY_POINT_ID"
```

Exercise the isolated auth-only dump/remove/restore path:

```bash
./scripts/test-auth-candidate-backup-restore.sh
```

The application and identity fingerprints must cover authoritative membership,
first-party profile/authority, MFA, session, receipt, and revision state without
writing secret values to evidence.

## Failure handling

If freshness is the only fault, create a new point rather than changing an old
one. If any component fails, leave the incomplete point unpublished, capture
the component/error class, and correct the dependency before choosing a new ID.

Record point ID, component/catalog hashes, UTC timestamps, retention,
application and identity fingerprints, failure domain, and cleanup. Record
secret references only.

A coordinated full application/object/browser restore and measured RPO/RTO are
`not run` in the maintained topology. Do not infer them from component backup
or auth-only restore evidence.

Backup deletion, retention shortening, immutability changes, remote stores,
production credentials, or production restore require separate authorization.
