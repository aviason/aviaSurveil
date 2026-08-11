#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
AVIA_AUTH_CANDIDATE_BROWSER_QUALIFICATION=1 \
  exec "$repository_root/scripts/test-auth-candidate-runtime.sh"
