#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
state_root="${AVIA_CANONICAL_PREPROD_STATE_DIR:-$repository_root/.local/aviasurveil360-canonical-preprod-cloudflare}"
runtime_root="${AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR:-$repository_root/.local/aviasurveil360-canonical-preprod-cloudflare-tunnel}"
tunnel_mode="${AVIA_PREPROD_CLOUDFLARE_MODE:-quick}"
public_hostname="${AVIA_PREPROD_PUBLIC_HOSTNAME:-demo.aviasurveil.com}"
keychain_service="${AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_SERVICE:-com.aviasurveil360.cloudflare-tunnel}"
keychain_account="${AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_ACCOUNT:-$public_hostname}"
runtime_file="$runtime_root/runtime.json"
fixture_file="$repository_root/deploy/local/fixtures/canonical-preprod-demo-identities.json"
password_file="$state_root/secrets/preprod_aga_demo_oidc_qualification_password"
status_script="$repository_root/scripts/status-canonical-preprod-cloudflare.sh"

fail() {
  printf 'canonical-preprod-cloudflare-users: %s\n' "$*" >&2
  exit 1
}

AVIA_CANONICAL_PREPROD_STATE_DIR="$state_root" \
AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR="$runtime_root" \
AVIA_PREPROD_CLOUDFLARE_MODE="$tunnel_mode" \
AVIA_PREPROD_PUBLIC_HOSTNAME="$public_hostname" \
AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_SERVICE="$keychain_service" \
AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_ACCOUNT="$keychain_account" \
  bash "$status_script" >/dev/null

for file in "$runtime_file" "$fixture_file" "$password_file"; do
  [[ -f "$file" && ! -L "$file" ]] || fail "required verified input is missing or symlinked: $file"
done

node --input-type=module - \
  "$runtime_file" "$fixture_file" "$password_file" "$tunnel_mode" "$public_hostname" <<'NODE'
import { readFileSync } from "node:fs";

const [runtimePath, fixturePath, passwordPath, expectedMode, expectedHostname] = process.argv.slice(2);
const runtime = JSON.parse(readFileSync(runtimePath, "utf8"));
const fixture = JSON.parse(readFileSync(fixturePath, "utf8"));
const password = readFileSync(passwordPath, "utf8").trim();
const origin = new URL(runtime.publicOrigin);
const runtimeMode = runtime.tunnel?.mode ?? "quick";
const hostnameValid = expectedMode === "quick"
  ? /^[a-z0-9-]+\.trycloudflare\.com$/u.test(origin.hostname)
  : origin.hostname === expectedHostname;
if (
  runtime.schemaVersion !== "canonical-preprod-cloudflare-runtime/v1" ||
  origin.protocol !== "https:" ||
  runtimeMode !== expectedMode ||
  !hostnameValid ||
  fixture.schemaVersion !== "canonical-preprod-demo-identities/v1" ||
  !Array.isArray(fixture.users) || fixture.users.length !== 9 ||
  !password || /[\r\n]/u.test(password)
) {
  throw new Error("verified Cloudflare Tunnel login metadata is invalid");
}

process.stdout.write(`URL: ${origin.origin}\n`);
process.stdout.write(`Password (all demo users): ${password}\n`);
process.stdout.write("\nRole\tUsername\n");
for (const user of fixture.users) {
  process.stdout.write(`${user.role}\t${user.email}\n`);
}
NODE
