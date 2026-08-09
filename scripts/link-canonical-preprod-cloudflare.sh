#!/usr/bin/env bash
set -euo pipefail

umask 077

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
state_root="${AVIA_CANONICAL_PREPROD_STATE_DIR:-$repository_root/.local/aviasurveil360-canonical-preprod-cloudflare}"
runtime_root="${AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR:-$repository_root/.local/aviasurveil360-canonical-preprod-cloudflare-tunnel}"
runtime_file="$runtime_root/runtime.json"
start_script="$repository_root/scripts/start-canonical-preprod-cloudflare.sh"
status_script="$repository_root/scripts/status-canonical-preprod-cloudflare.sh"

fail() {
  printf 'canonical-preprod-cloudflare-link: %s\n' "$*" >&2
  exit 1
}

validate_disposable_path() {
  local value="$1"
  local label="$2"
  case "$value" in
    "$repository_root"/.local/*) ;;
    *) fail "$label must be an absolute path below $repository_root/.local" ;;
  esac
  [[ "$value" != "$repository_root/.local" && "$value" != "$repository_root/.local/" ]] ||
    fail "refusing the broad local state root for $label"
  [[ ! -L "$value" ]] || fail "$label must not be a symlink"
  [[ "$value" != *"//"* && "$value" != *"/./"* && "$value" != */. &&
    "$value" != *"/../"* && "$value" != */.. ]] ||
    fail "$label must be a canonical path without . or .. components"

  local remaining current component
  remaining="${value#"$repository_root/.local/"}"
  current="$repository_root/.local"
  while [[ -n "$remaining" ]]; do
    component="${remaining%%/*}"
    current="$current/$component"
    [[ ! -L "$current" ]] || fail "$label must not traverse a symlink"
    if [[ "$remaining" == */* ]]; then
      [[ ! -e "$current" || -d "$current" ]] ||
        fail "$label has a non-directory ancestor"
      remaining="${remaining#*/}"
    else
      remaining=""
    fi
  done
}

[[ "$#" == 0 ]] || fail "this helper accepts no arguments"
[[ -x "$start_script" && -x "$status_script" ]] || fail "Quick Tunnel lifecycle scripts are missing or not executable"
[[ ! -L "$repository_root/.local" ]] || fail "$repository_root/.local must not be a symlink"
[[ ! -e "$repository_root/.local" || -d "$repository_root/.local" ]] ||
  fail "$repository_root/.local must be a directory when it exists"
validate_disposable_path "$state_root" "AVIA_CANONICAL_PREPROD_STATE_DIR"
validate_disposable_path "$runtime_root" "AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR"
[[ "$state_root" != "$runtime_root" ]] || fail "state and runtime roots must be distinct"

if [[ -e "$state_root" || -L "$state_root" || -e "$runtime_root" || -L "$runtime_root" ]]; then
  [[ -d "$state_root" && ! -L "$state_root" && -d "$runtime_root" && ! -L "$runtime_root" ]] ||
    fail "refusing to reuse partial or stale disposable state"
  [[ -f "$runtime_file" && ! -L "$runtime_file" ]] ||
    fail "runtime metadata is missing; refusing to reuse partial or stale disposable state"
  "$status_script" >/dev/null
else
  "$start_script" >/dev/null
  "$status_script" >/dev/null
fi

node --input-type=module - "$runtime_file" "$state_root" "$runtime_root" <<'NODE'
import { readFileSync } from "node:fs";
const [runtimeFile, stateDirectory, runtimeDirectory] = process.argv.slice(2);
const metadata = JSON.parse(readFileSync(runtimeFile, "utf8"));
const hostname = /^(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)\.trycloudflare\.com$/u;
let publicOrigin;
try {
  publicOrigin = new URL(metadata.publicOrigin);
} catch {
  throw new Error("Quick Tunnel runtime metadata has no valid public origin");
}
if (
  metadata.schemaVersion !== "canonical-preprod-cloudflare-runtime/v1" ||
  metadata.stateDirectory !== stateDirectory ||
  metadata.runtimeDirectory !== runtimeDirectory ||
  metadata.cookieSecure !== true ||
  metadata.donorRuntime !== "disabled" ||
  metadata.externalPreprod !== "not run" ||
  publicOrigin.protocol !== "https:" ||
  publicOrigin.origin !== metadata.publicOrigin ||
  publicOrigin.pathname !== "/" ||
  publicOrigin.search ||
  publicOrigin.hash ||
  !hostname.test(publicOrigin.hostname)
) {
  throw new Error("canonical Cloudflare Quick Tunnel runtime metadata is invalid");
}
process.stdout.write(`${metadata.publicOrigin}\n`);
NODE
