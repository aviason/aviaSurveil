#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export AVIA_PREPROD_TRANSPORT=http
export AVIA_PREPROD_HTTP_PORT="${AVIA_PREPROD_HTTP_PORT:-8085}"
export AVIA_CANONICAL_PREPROD_PROJECT="${AVIA_CANONICAL_PREPROD_PROJECT:-aviasurveil360-local-preprod-http}"
export AVIA_CANONICAL_PREPROD_STATE_DIR="${AVIA_CANONICAL_PREPROD_STATE_DIR:-$repository_root/.local/aviasurveil360-canonical-preprod-http}"
export AVIA_PREPROD_COMPOSE_OVERRIDE="$repository_root/deploy/local/compose.local-http.yaml"

exec "$repository_root/scripts/start-canonical-preprod.sh" "$@"
