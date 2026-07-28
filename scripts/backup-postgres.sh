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

acquire_recovery_lock application-database
directory=$(prepare_component_directory "$recovery_point_id")
component_next="$directory/.applicationDatabase.json.next"
component_path="$directory/applicationDatabase.json"
info_path="$directory/.application-pgbackrest-info.json"
fingerprint_path="$directory/.application-fingerprint.json"

record_backup_audit started "$recovery_point_id" applicationDatabase pending

pgbackrest_command \
  postgres \
  /etc/pgbackrest/pgbackrest.conf \
  application \
  stanza-create
pgbackrest_command \
  postgres \
  /etc/pgbackrest/pgbackrest.conf \
  application \
  check
pgbackrest_command \
  postgres \
  /etc/pgbackrest/pgbackrest.conf \
  application \
  backup \
  "--type=$backup_type"
pgbackrest_command \
  postgres \
  /etc/pgbackrest/pgbackrest.conf \
  application \
  info \
  --output=json >"$info_path"

compose_recovery exec --no-TTY \
  --env AVIA_ENVIRONMENT=local-candidate \
  --env AVIA_ENABLE_RECOVERY_BACKUP=true \
  --env "AVIA_RECOVERY_POINT_ID=$recovery_point_id" \
  recovery-toolbox \
  /bin/sh -c '
    password=$(tr -d "\r\n" </run/secrets/app_database_password)
    export AVIA_DATABASE_URL="postgres://aviasurveil360:${password}@postgres:5432/aviasurveil360?sslmode=disable"
    exec /usr/local/bin/recovery-fingerprint
  ' >"$fingerprint_path"

jq -s \
  --arg backupType "$backup_type" \
  --arg capturedAt "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  '
    .[0] as $fingerprint |
    .[1][0] as $stanza |
    ($stanza.backup[-1] // error("pgBackRest returned no backup")) as $backup |
    $fingerprint |
    .applicationDatabase.pgBackRest = {
      stanza: $stanza.name,
      backupType: $backupType,
      label: $backup.label,
      prior: $backup.prior,
      reference: ($backup.reference // []),
      timestamp: $backup.timestamp,
      databaseSize: $backup.info.size,
      repositorySize: $backup.info.repository.size,
      capturedAt: $capturedAt
    }
  ' "$fingerprint_path" "$info_path" >"$component_next"

mv -f -- "$component_next" "$component_path"
chmod 0600 "$component_path"
rm -f -- "$info_path" "$fingerprint_path"
record_backup_audit completed "$recovery_point_id" applicationDatabase succeeded
echo "Application PostgreSQL backup component: verified locally"
