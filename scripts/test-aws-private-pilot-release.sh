#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
node --test "$repo_root/tests/aws-private-pilot-release-contract.test.mjs"
printf '%s\n' 'verified locally: private-pilot release command contracts; every external action remains not run'
