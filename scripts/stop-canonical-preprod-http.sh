#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
state_root="${AVIA_CANONICAL_PREPROD_STATE_DIR:-$repository_root/.local/aviasurveil360-canonical-preprod-http}"
project_name="${AVIA_CANONICAL_PREPROD_PROJECT:-aviasurveil360-local-preprod-http}"
compose_file="$repository_root/deploy/local/compose.yaml"
compose_override="$repository_root/deploy/local/compose.local-http.yaml"

[[ -d "$state_root" && ! -L "$state_root" ]] || {
  printf 'canonical-preprod-http-down: state directory is missing: %s\n' "$state_root" >&2
  exit 1
}
[[ -f "$state_root/runtime.json" ]] || {
  printf 'canonical-preprod-http-down: runtime metadata is missing: %s\n' "$state_root/runtime.json" >&2
  exit 1
}

AVIA_PREPROD_STATE_DIR="$state_root" \
AVIA_PREPROD_PROFILE="aga-preprod@1.0.0" \
AVIA_PREPROD_PROFILE_QUALIFICATION="true" \
AVIA_PREPROD_IDENTITY_NAMESPACE="canonical-aga-preprod-exercise-v1" \
AVIA_PREPROD_HTTP_PORT="${AVIA_PREPROD_HTTP_PORT:-8085}" \
  docker compose --project-name "$project_name" \
    --file "$compose_file" \
    --file "$compose_override" \
    --profile local-preprod-loader down "$@"

printf 'Canonical local HTTP preprod stopped; state retained at %s\n' "$state_root"
