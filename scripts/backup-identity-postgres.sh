#!/bin/sh
set -eu

script_directory=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$script_directory/lib/recovery-backup.sh"

if [ "$#" -ne 2 ]; then
  echo "usage: $0 RECOVERY_POINT_ID {full|diff|incr}" >&2
  exit 64
fi

recovery_point_id=$1
backup_type=$2
validate_recovery_point_id "$recovery_point_id"
case "$backup_type" in
  full | diff | incr) ;;
  *)
    echo "backup type must be full, diff, or incr" >&2
    exit 64
    ;;
esac

acquire_recovery_lock identity-database
directory=$(prepare_component_directory "$recovery_point_id")
component_next="$directory/.identityDatabase.json.next"
component_path="$directory/identityDatabase.json"
info_path="$directory/.identity-pgbackrest-info.json"
fingerprint_path="$directory/.identity-fingerprint.json"

record_backup_audit started "$recovery_point_id" identityDatabase pending

pgbackrest_command \
  preprod-auth-postgres \
  /etc/pgbackrest/pgbackrest.conf \
  identity \
  stanza-create
pgbackrest_command \
  preprod-auth-postgres \
  /etc/pgbackrest/pgbackrest.conf \
  identity \
  check
pgbackrest_command \
  preprod-auth-postgres \
  /etc/pgbackrest/pgbackrest.conf \
  identity \
  backup \
  "--type=$backup_type"
pgbackrest_command \
  preprod-auth-postgres \
  /etc/pgbackrest/pgbackrest.conf \
  identity \
  info \
  --output=json >"$info_path"

compose_recovery exec --no-TTY \
  --env AVIA_ENVIRONMENT=local-candidate \
  --env AVIA_ENABLE_RECOVERY_BACKUP=true \
  --env "AVIA_RECOVERY_POINT_ID=$recovery_point_id" \
  recovery-toolbox \
  /bin/sh /opt/aviasurveil360/identity-recovery-fingerprint.sh \
  >"$fingerprint_path"

jq -s \
  --arg backupType "$backup_type" \
  --arg capturedAt "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  '
    .[0] as $fingerprint |
    .[1][0] as $stanza |
    ($stanza.backup[-1] // error("pgBackRest returned no backup")) as $backup |
    {
      identityDatabase: {
        status: "verified locally",
        artifactStatus: "candidate-only",
        recoveryPointId: $fingerprint.identityFingerprint.recoveryPointId,
        pgBackRest: {
          stanza: $stanza.name,
          backupType: $backupType,
          label: $backup.label,
          prior: $backup.prior,
          reference: ($backup.reference // []),
          timestamp: $backup.timestamp,
          databaseSize: $backup.info.size,
          repositorySize: $backup.info.repository.size,
          capturedAt: $capturedAt
        },
        sha256: $fingerprint.identityFingerprint.sha256
      },
      identityFingerprint: $fingerprint.identityFingerprint
    }
  ' "$fingerprint_path" "$info_path" >"$component_next"

mv -f -- "$component_next" "$component_path"
chmod 0600 "$component_path"
rm -f -- "$info_path" "$fingerprint_path"
record_backup_audit completed "$recovery_point_id" identityDatabase succeeded
echo "Identity PostgreSQL backup component: verified locally"
