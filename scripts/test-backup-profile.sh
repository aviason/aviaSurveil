#!/bin/sh
set -eu

script_directory=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repository_root=$(CDPATH= cd -- "$script_directory/.." && pwd)
local_compose="$repository_root/deploy/local/compose.yaml"
recovery_compose="$repository_root/deploy/recovery/compose.recovery.yaml"
runtime_directory=$(
  mktemp -d /private/tmp/aviasurveil360-plan4-backup.XXXXXX
)
AVIA_LOCAL_PROJECT="aviasurveil360-task-plan4-backup-$(date -u +%Y%m%d%H%M%S)-$$"
AVIASURVEIL_LOCAL_STATE_DIR="$runtime_directory/local-state"
AVIA_LOCAL_HTTPS_PORT=${AVIA_BACKUP_HTTPS_PORT:-$((48443 + $$ % 1000))}
AVIA_LOCAL_PUBLIC_ORIGIN="https://localhost:$AVIA_LOCAL_HTTPS_PORT"
stack_started=false

export AVIA_LOCAL_PROJECT
export AVIASURVEIL_LOCAL_STATE_DIR
export AVIA_LOCAL_HTTPS_PORT
export AVIA_LOCAL_PUBLIC_ORIGIN
export COMPOSE_PROGRESS=plain

compose() {
  docker compose \
    --project-name "$AVIA_LOCAL_PROJECT" \
    --file "$local_compose" \
    --file "$recovery_compose" \
    --profile full \
    --profile recovery \
    "$@"
}

force_remove_task_owned_residue() {
  residue_status=0
  for resource_id in $(
    docker ps --all --quiet \
      --filter "label=com.docker.compose.project=$AVIA_LOCAL_PROJECT"
  ); do
    docker rm --force "$resource_id" || residue_status=1
  done
  for resource_id in $(
    docker volume ls --quiet \
      --filter "label=com.docker.compose.project=$AVIA_LOCAL_PROJECT"
  ); do
    docker volume rm "$resource_id" || residue_status=1
  done
  for resource_id in $(
    docker network ls --quiet \
      --filter "label=com.docker.compose.project=$AVIA_LOCAL_PROJECT"
  ); do
    docker network rm "$resource_id" || residue_status=1
  done
  return "$residue_status"
}

assert_no_task_owned_residue() {
  residue=$(
    {
      docker ps --all --quiet \
        --filter "label=com.docker.compose.project=$AVIA_LOCAL_PROJECT"
      docker volume ls --quiet \
        --filter "label=com.docker.compose.project=$AVIA_LOCAL_PROJECT"
      docker network ls --quiet \
        --filter "label=com.docker.compose.project=$AVIA_LOCAL_PROJECT"
    } | sed '/^$/d'
  )
  if [ -n "$residue" ]; then
    echo "task-owned recovery Compose residue remains" >&2
    return 1
  fi
  echo "Task-owned backup profile residue: zero"
}

cleanup() {
  cleanup_status=$?
  trap - EXIT HUP INT TERM
  set +e
  if [ "$stack_started" = true ]; then
    compose down --volumes --remove-orphans --timeout 15
  fi
  force_remove_task_owned_residue || cleanup_status=1
  assert_no_task_owned_residue || cleanup_status=1
  rm -rf -- "$runtime_directory"
  exit "$cleanup_status"
}
trap cleanup EXIT HUP INT TERM

mkdir -p "$AVIASURVEIL_LOCAL_STATE_DIR"
chmod 0700 "$AVIASURVEIL_LOCAL_STATE_DIR"
printf '%s\n' "$AVIA_LOCAL_PROJECT" \
  >"$AVIASURVEIL_LOCAL_STATE_DIR/.compose-project-owner"
chmod 0600 "$AVIASURVEIL_LOCAL_STATE_DIR/.compose-project-owner"

"$script_directory/init-local-secrets.sh"
"$script_directory/check-local-image-evidence.sh" full
"$script_directory/check-local-image-evidence.sh" recovery

stack_started=true
compose up --detach --wait

first_point="rp-$(date -u +%Y%m%dT%H%M%SZ)-full"
"$script_directory/verify-backup-catalog.sh" \
  --create "$first_point" full

# Controlled database change: an audit-only recovery fixture with no product
# lifecycle effect makes the second application fingerprint distinct.
compose exec --no-TTY postgres \
  psql \
  --username aviasurveil360 \
  --dbname aviasurveil360 \
  --set ON_ERROR_STOP=1 \
  --command "
    INSERT INTO audit_events (
      event_id,
      occurred_at,
      action,
      entity_type,
      entity_id,
      details
    ) VALUES (
      'plan4-backup-controlled-$first_point',
      now(),
      'recovery.fixture.recorded',
      'RecoveryFixture',
      '$first_point',
      '{\"artifactStatus\":\"candidate-only\"}'::jsonb
    )
  " >/dev/null

# Controlled object change: a versioned sentinel proves exact key/version,
# checksum, and manifest selection in the second recovery point.
compose exec --no-TTY minio /bin/sh -c '
  root_user=$(tr -d "\r\n" </run/secrets/minio_root_user)
  root_password=$(tr -d "\r\n" </run/secrets/minio_root_password)
  mc alias set primary http://127.0.0.1:9000 "$root_user" "$root_password" >/dev/null
  printf "%s\n" "Plan 4 controlled object change" >/tmp/recovery-change.txt
  mc cp /tmp/recovery-change.txt \
    primary/evidence-quarantine/recovery-fixtures/controlled-change.txt >/dev/null
  rm -f /tmp/recovery-change.txt
  unset root_user root_password
'

sleep 1
second_point="rp-$(date -u +%Y%m%dT%H%M%SZ)-incr"
"$script_directory/verify-backup-catalog.sh" \
  --create "$second_point" incr

first_catalog="$AVIASURVEIL_LOCAL_STATE_DIR/recovery/catalog/$first_point/catalog.json"
second_catalog="$AVIASURVEIL_LOCAL_STATE_DIR/recovery/catalog/$second_point/catalog.json"

jq -e '
  .status == "complete" and
  .artifactStatus == "candidate-only" and
  .failureDomain == "same-host-logically-isolated" and
  .productionHostLossCovered == false and
  .applicationDatabase.pgBackRest.backupType == "full"
' "$first_catalog" >/dev/null
jq -e '
  .status == "complete" and
  .artifactStatus == "candidate-only" and
  .applicationDatabase.pgBackRest.backupType == "incr" and
  (.applicationObjects.objectCount >= 1)
' "$second_catalog" >/dev/null

first_database_sha=$(
  jq -er '.applicationDatabase.sha256' "$first_catalog"
)
second_database_sha=$(
  jq -er '.applicationDatabase.sha256' "$second_catalog"
)
if [ "$first_database_sha" = "$second_database_sha" ]; then
  echo "controlled database change did not change the recovery fingerprint" >&2
  exit 1
fi

first_manifest_sha=$(
  jq -er '.applicationObjects.manifestSha256' "$first_catalog"
)
second_manifest_sha=$(
  jq -er '.applicationObjects.manifestSha256' "$second_catalog"
)
if [ "$first_manifest_sha" = "$second_manifest_sha" ]; then
  echo "controlled object change did not change the exact-version manifest" >&2
  exit 1
fi

if [ "$(
  wc -l <"$AVIASURVEIL_LOCAL_STATE_DIR/recovery/metrics.jsonl" |
    tr -d ' '
)" -ne 2 ]; then
  echo "expected one recovery-point age metric per complete catalog" >&2
  exit 1
fi

echo "Two recovery points around controlled database/object changes: verified locally"
echo "Application full-to-incremental pgBackRest chain: verified locally"
echo "Object version, ETag, SHA-256, metadata, and retention manifests: verified locally"
echo "Backup artifact status: candidate-only"
echo "Backup failure domain: same-host-logically-isolated"
