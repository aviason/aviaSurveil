#!/bin/sh
set -eu

recovery_repository_root=$(
  CDPATH= cd -- "$(dirname -- "$0")/.." && pwd
)
recovery_state_directory=${AVIASURVEIL_LOCAL_STATE_DIR:-"$recovery_repository_root/.local/aviasurveil360"}
recovery_catalog_directory="$recovery_state_directory/recovery/catalog"
recovery_lock_directory="$recovery_state_directory/recovery/locks"
recovery_audit_path="$recovery_state_directory/recovery/backup-audit.jsonl"
recovery_local_compose="$recovery_repository_root/deploy/local/compose.yaml"
recovery_compose="$recovery_repository_root/deploy/recovery/compose.recovery.yaml"
recovery_project=${AVIA_LOCAL_PROJECT:-aviasurveil360-local}
recovery_active_lock=""

compose_recovery() {
  docker compose \
    --project-name "$recovery_project" \
    --file "$recovery_local_compose" \
    --file "$recovery_compose" \
    --profile full \
    --profile recovery \
    "$@"
}

validate_recovery_point_id() {
  recovery_point_id=$1
  if ! printf '%s\n' "$recovery_point_id" |
    grep -Eq '^rp-[a-z0-9TZ][a-z0-9TZ-]{5,95}$'; then
    echo "invalid recovery point identifier" >&2
    return 1
  fi
}

prepare_recovery_directories() {
  mkdir -p "$recovery_catalog_directory" "$recovery_lock_directory"
  chmod 0700 \
    "$recovery_state_directory/recovery" \
    "$recovery_catalog_directory" \
    "$recovery_lock_directory"
}

release_recovery_lock() {
  if [ -n "$recovery_active_lock" ]; then
    rmdir "$recovery_active_lock" 2>/dev/null || true
    recovery_active_lock=""
  fi
}

acquire_recovery_lock() {
  lock_name=$1
  prepare_recovery_directories
  recovery_active_lock="$recovery_lock_directory/$lock_name.lock"
  if ! mkdir "$recovery_active_lock" 2>/dev/null; then
    echo "recovery operation $lock_name is already running" >&2
    return 1
  fi
  trap release_recovery_lock EXIT HUP INT TERM
}

component_directory() {
  recovery_point_id=$1
  printf '%s\n' "$recovery_catalog_directory/$recovery_point_id/components"
}

prepare_component_directory() {
  recovery_point_id=$1
  directory=$(component_directory "$recovery_point_id")
  mkdir -p "$directory"
  chmod 0700 "$recovery_catalog_directory/$recovery_point_id" "$directory"
  printf '%s\n' "$directory"
}

record_backup_audit() {
  event=$1
  recovery_point_id=$2
  component=$3
  outcome=$4
  prepare_recovery_directories
  jq -cn \
    --arg occurredAt "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --arg event "$event" \
    --arg recoveryPointId "$recovery_point_id" \
    --arg component "$component" \
    --arg outcome "$outcome" \
    '{
      occurredAt: $occurredAt,
      event: $event,
      recoveryPointId: $recoveryPointId,
      component: $component,
      outcome: $outcome,
      artifactStatus: "candidate-only"
    }' >>"$recovery_audit_path"
  chmod 0600 "$recovery_audit_path"
}

pgbackrest_command() {
  service=$1
  config=$2
  stanza=$3
  shift 3
  compose_recovery exec --no-TTY "$service" \
    /usr/local/bin/pgbackrest-secret \
    "--config=$config" \
    "--stanza=$stanza" \
    "$@"
}
