#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
state_root="${AVIA_CANONICAL_PREPROD_STATE_DIR:-$repository_root/.local/aviasurveil360-canonical-preprod-cloudflare}"
runtime_root="${AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR:-$repository_root/.local/aviasurveil360-canonical-preprod-cloudflare-tunnel}"
status_script="$repository_root/scripts/status-canonical-preprod-cloudflare.sh"
runtime_file="$runtime_root/runtime.json"
password_file="$state_root/secrets/preprod_canonical_demo_oidc_qualification_password"

fail() {
  printf 'canonical-preprod-cloudflare-panels: %s\n' "$*" >&2
  exit 1
}

[[ -f "$status_script" ]] || fail "status helper is missing"
AVIA_CANONICAL_PREPROD_STATE_DIR="$state_root" \
AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR="$runtime_root" \
  bash "$status_script" >/dev/null

[[ -f "$runtime_file" && ! -L "$runtime_file" ]] || fail "verified runtime metadata is missing"
[[ -f "$password_file" && ! -L "$password_file" ]] || fail "demo qualification password is missing"

public_origin="$(node -e '
  const { readFileSync } = require("node:fs");
  const metadata = JSON.parse(readFileSync(process.argv[1], "utf8"));
  const origin = new URL(metadata.publicOrigin);
  if (metadata.schemaVersion !== "canonical-preprod-cloudflare-runtime/v1" ||
      origin.protocol !== "https:" ||
      !/^[a-z0-9-]+\.trycloudflare\.com$/u.test(origin.hostname) ||
      origin.pathname !== "/" || origin.search || origin.hash) process.exit(65);
  process.stdout.write(origin.origin);
' "$runtime_file")" || fail "runtime public origin is invalid"
public_host="${public_origin#https://}"
qualification_password="$(tr -d '\r\n' <"$password_file")"
[[ -n "$qualification_password" ]] || fail "demo qualification password is empty"

output_root="$(mktemp -d /private/tmp/aviasurveil360-canonical-quick-tunnel-e2e.XXXXXX)"
cleanup() {
  case "$output_root" in
    /private/tmp/aviasurveil360-canonical-quick-tunnel-e2e.*) rm -rf -- "$output_root" ;;
    *) printf 'canonical-preprod-cloudflare-panels: refusing unsafe output cleanup\n' >&2 ;;
  esac
}
trap cleanup EXIT

(
  cd "$repository_root/apps/web"
  AVIA_E2E_PROFILE=canonical-quick-tunnel \
  AVIA_E2E_BASE_URL="$public_origin" \
  AVIA_PREPROD_OIDC_HOST="$public_host" \
  AVIA_AGA_OIDC_PASSWORD="$qualification_password" \
  AVIA_PLAYWRIGHT_OUTPUT_DIR="$output_root" \
    ./node_modules/.bin/playwright test \
      tests/e2e/canonical-quick-tunnel-panels.spec.ts \
      --project=canonical-quick-tunnel
)

printf 'canonical Quick Tunnel browser panels verified locally at %s; external preprod not run.\n' "$public_origin"
