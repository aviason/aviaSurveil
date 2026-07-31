#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <producer-manifest.json> <aviacore-manifest.json>" >&2
  exit 64
fi
if [[ ! -r "$1" || ! -r "$2" ]]; then
  echo "both producer and AviaCore manifests must be readable files" >&2
  exit 64
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
PRODUCER_MANIFEST="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
AVIACORE_MANIFEST="$(cd "$(dirname "$2")" && pwd)/$(basename "$2")"

go -C "${REPOSITORY_ROOT}/apps/api" run ./cmd/data-feed-reconcile "${PRODUCER_MANIFEST}" "${AVIACORE_MANIFEST}"
