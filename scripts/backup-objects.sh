#!/bin/sh
set -eu

script_directory=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$script_directory/lib/recovery-backup.sh"

if [ "$#" -ne 1 ]; then
  echo "usage: $0 RECOVERY_POINT_ID" >&2
  exit 64
fi

recovery_point_id=$1
validate_recovery_point_id "$recovery_point_id"
acquire_recovery_lock application-objects
directory=$(prepare_component_directory "$recovery_point_id")
component_next="$directory/.applicationObjects.json.next"
component_path="$directory/applicationObjects.json"

record_backup_audit started "$recovery_point_id" applicationObjects pending

compose_recovery exec --no-TTY \
  --env AVIA_ENVIRONMENT=local-candidate \
  --env AVIA_ENABLE_RECOVERY_BACKUP=true \
  recovery-toolbox \
  /usr/local/bin/object-backup \
  --recovery-point "$recovery_point_id" \
  >"$component_next"

jq -e \
  --arg recoveryPointId "$recovery_point_id" \
  '
    .applicationObjects.status == "verified locally" and
    .applicationObjects.artifactStatus == "candidate-only" and
    .applicationObjects.recoveryPointId == $recoveryPointId and
    (.applicationObjects.manifestSha256 | test("^[a-f0-9]{64}$"))
  ' "$component_next" >/dev/null

mv -f -- "$component_next" "$component_path"
chmod 0600 "$component_path"
record_backup_audit completed "$recovery_point_id" applicationObjects succeeded
echo "Application-object versions and recovery-point manifest: verified locally"
