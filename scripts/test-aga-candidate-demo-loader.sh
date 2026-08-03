#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
go -C "$repository_root/apps/api" test ./cmd/preprod-aga-candidate-demo-loader -run AGACandidateDemo
node --test "$repository_root/tests/preprod-data-boundary.test.mjs"
