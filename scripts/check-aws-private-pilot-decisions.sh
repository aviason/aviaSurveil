#!/usr/bin/env bash
set -euo pipefail

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
decision_file=${1:-${AVIA_AWS_PRIVATE_PILOT_DECISION_FILE:-"$repository_root/.local/aviasurveil360/aws-private-pilot/decision.json"}}
node_bin=${NODE_BIN:-node}

if [[ ! -e "$decision_file" ]]; then
  echo "missing-owner-input: decision file not found: $decision_file" >&2
  exit 64
fi

exec "$node_bin" "$repository_root/scripts/lib/aws-private-pilot-decision.mjs" "$decision_file"
