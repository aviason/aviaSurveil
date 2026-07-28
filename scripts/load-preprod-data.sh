#!/usr/bin/env bash
set -euo pipefail

umask 077

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_file="$repository_root/deploy/local/compose.yaml"
configuration_file="${AVIA_PREPROD_LOADER_CONFIG_FILE:-}"
authorization_file="${AVIA_PREPROD_AUTHORIZATION_FILE:-}"
seed_file="${AVIA_PREPROD_SEED_FILE:-}"
control_store_directory="${AVIA_PREPROD_CONTROL_STORE_DIR:-}"
operation="${1:-}"

fail() {
  printf 'preprod-data-loader: %s\n' "$*" >&2
  exit 1
}

if [[ "$operation" != "prepare" &&
  "$operation" != "verify-authorization" &&
  "$operation" != "run-connected" &&
  "$operation" != "record-cleanup" ]]; then
  fail "usage: $0 prepare|verify-authorization|run-connected|record-cleanup"
fi
[[ -n "$configuration_file" ]] ||
  fail "AVIA_PREPROD_LOADER_CONFIG_FILE is required"
[[ -n "$seed_file" ]] ||
  fail "AVIA_PREPROD_SEED_FILE is required"
[[ -f "$configuration_file" ]] ||
  fail "loader configuration file does not exist"
[[ -f "$seed_file" ]] ||
  fail "seed file does not exist"
chmod 600 "$configuration_file" "$seed_file"

if [[ "$operation" == "verify-authorization" ||
  "$operation" == "run-connected" ||
  "$operation" == "record-cleanup" ]]; then
  [[ -n "$authorization_file" ]] ||
    fail "AVIA_PREPROD_AUTHORIZATION_FILE is required"
  [[ -f "$authorization_file" ]] ||
    fail "authorization file does not exist"
  chmod 600 "$authorization_file"
fi

export AVIA_PREPROD_LOADER_CONFIG_FILE="$configuration_file"
export AVIA_PREPROD_AUTHORIZATION_FILE="$authorization_file"
export AVIA_PREPROD_SEED_FILE="$seed_file"

if [[ "$operation" == "record-cleanup" ]]; then
  [[ -n "$control_store_directory" ]] ||
    fail "AVIA_PREPROD_CONTROL_STORE_DIR is required"
  [[ -d "$control_store_directory" ]] ||
    fail "control-store directory does not exist"
  docker run \
    --rm \
    --network none \
    --read-only \
    --security-opt no-new-privileges:true \
    --cap-drop ALL \
    --pids-limit 64 \
    --tmpfs /tmp:rw,noexec,nosuid,size=16m \
    --volume "$configuration_file:/run/config/preprod-loader.json:ro" \
    --volume \
      "$authorization_file:/run/secrets/preprod_loader_authorization:ro" \
    --volume \
      "$control_store_directory:/var/lib/aviasurveil360-preprod-control:rw" \
    aviasurveil360/preprod-data-loader:local \
    record-cleanup /run/config/preprod-loader.json
  exit 0
fi

compose_arguments=(
  compose
  --project-name aviasurveil360-local-preprod
  --file "$compose_file"
  --profile local-preprod-loader
  run
  --rm
)

docker "${compose_arguments[@]}" preprod-data-loader \
  "$operation" /run/config/preprod-loader.json
