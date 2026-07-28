#!/usr/bin/env bash
set -euo pipefail

umask 077

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_file="$repository_root/deploy/local/compose.yaml"
configuration_file="${AVIA_PREPROD_LOADER_CONFIG_FILE:-}"
authorization_file="${AVIA_PREPROD_AUTHORIZATION_FILE:-}"
seed_file="${AVIA_PREPROD_SEED_FILE:-}"
operation="${1:-}"

fail() {
  printf 'preprod-data-loader: %s\n' "$*" >&2
  exit 1
}

if [[ "$operation" != "prepare" && "$operation" != "verify-authorization" ]]; then
  fail "usage: $0 prepare|verify-authorization"
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

if [[ "$operation" == "verify-authorization" ]]; then
  [[ -n "$authorization_file" ]] ||
    fail "AVIA_PREPROD_AUTHORIZATION_FILE is required"
  [[ -f "$authorization_file" ]] ||
    fail "authorization file does not exist"
  chmod 600 "$authorization_file"
fi

export AVIA_PREPROD_LOADER_CONFIG_FILE="$configuration_file"
export AVIA_PREPROD_AUTHORIZATION_FILE="$authorization_file"
export AVIA_PREPROD_SEED_FILE="$seed_file"

docker compose \
  --project-name aviasurveil360-local-preprod \
  --file "$compose_file" \
  --profile local-preprod-loader \
  run --rm preprod-data-loader \
  "$operation" /run/config/preprod-loader.json
