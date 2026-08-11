#!/usr/bin/env bash
set -euo pipefail

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
mode=${1:-static}

if [[ "$mode" != "static" ]]; then
  echo "unsupported mode: only offline static verification is authorized" >&2
  exit 64
fi

node --test "$repository_root/tests/aws-private-pilot-compose-contract.test.mjs"
echo "verified locally: aws-private-pilot production Compose contract"
