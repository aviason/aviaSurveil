#!/usr/bin/env bash
set -euo pipefail

umask 077

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
http_port="${AVIA_PREPROD_HTTP_PORT:-8085}"
project_name="${AVIA_CANONICAL_PREPROD_PROJECT:-aviasurveil360-local-preprod-cloudflare}"
state_root="${AVIA_CANONICAL_PREPROD_STATE_DIR:-$repository_root/.local/aviasurveil360-canonical-preprod-cloudflare}"
runtime_root="${AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR:-$repository_root/.local/aviasurveil360-canonical-preprod-cloudflare-tunnel}"
compose_file="$repository_root/deploy/local/compose.yaml"
compose_override="$repository_root/deploy/local/compose.local-http.yaml"
runtime_file="$runtime_root/runtime.json"
local_origin="http://127.0.0.1:${http_port}"

fail() {
  printf 'canonical-preprod-cloudflare-status: %s\n' "$*" >&2
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

metadata_value() {
  local expression="$1"
  node --input-type=module - "$runtime_file" "$expression" <<'NODE'
import { readFileSync } from "node:fs";
const [runtimeFile, expression] = process.argv.slice(2);
const metadata = JSON.parse(readFileSync(runtimeFile, "utf8"));
const value = expression.split(".").reduce((current, key) => current?.[key], metadata);
if (typeof value !== "string" && typeof value !== "number" && typeof value !== "boolean") {
  throw new Error(`runtime metadata field is missing: ${expression}`);
}
process.stdout.write(String(value));
NODE
}

validate_runtime_metadata() {
  node --input-type=module - \
    "$runtime_file" "$project_name" "$state_root" "$runtime_root" "$local_origin" <<'NODE'
import { readFileSync } from "node:fs";
const [runtimeFile, project, stateDirectory, runtimeDirectory, localOrigin] = process.argv.slice(2);
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
  metadata.project !== project ||
  metadata.stateDirectory !== stateDirectory ||
  metadata.runtimeDirectory !== runtimeDirectory ||
  metadata.localOrigin !== localOrigin ||
  metadata.cookieSecure !== true ||
  metadata.donorRuntime !== "disabled" ||
  metadata.externalPreprod !== "not run" ||
  publicOrigin.protocol !== "https:" ||
  publicOrigin.origin !== metadata.publicOrigin ||
  publicOrigin.pathname !== "/" ||
  publicOrigin.search ||
  publicOrigin.hash ||
  !hostname.test(publicOrigin.hostname) ||
  !Number.isSafeInteger(metadata.tunnel?.pid) ||
  metadata.tunnel?.pidFile !== `${runtimeDirectory}/cloudflared.pid` ||
  metadata.tunnel?.logFile !== `${runtimeDirectory}/cloudflared.log` ||
  metadata.tunnel?.localOrigin !== localOrigin ||
  typeof metadata.tunnel?.processCommand !== "string" ||
  !metadata.tunnel.processCommand.includes("cloudflared") ||
  !metadata.tunnel.processCommand.includes(`tunnel --url ${localOrigin}`)
) {
  throw new Error("canonical Cloudflare Quick Tunnel runtime metadata is invalid");
}
NODE
}

verify_public_discovery() {
  local public_origin="$1"
  node --input-type=module - "$public_origin" <<'NODE'
const origin = process.argv[2];
const issuer = `${origin}/identity/realms/aviasurveil360-local-preprod`;
const response = await fetch(`${issuer}/.well-known/openid-configuration`, {
  signal: AbortSignal.timeout(5000),
});
if (!response.ok) throw new Error(`OIDC discovery returned ${response.status}`);
const discovery = await response.json();
if (discovery.issuer !== issuer) throw new Error("OIDC issuer did not match the public Quick Tunnel origin");
NODE
}

[[ "$http_port" =~ ^[0-9]+$ && "$http_port" -ge 1024 && "$http_port" -le 65535 ]] ||
  fail "AVIA_PREPROD_HTTP_PORT must be a user-space TCP port"
[[ "$project_name" =~ ^aviasurveil360-local-preprod-cloudflare(-[a-z0-9][a-z0-9-]*)?$ ]] ||
  fail "AVIA_CANONICAL_PREPROD_PROJECT must be a task-owned Cloudflare disposable project"
[[ ! -L "$repository_root/.local" ]] || fail "$repository_root/.local must not be a symlink"
[[ ! -e "$repository_root/.local" || -d "$repository_root/.local" ]] ||
  fail "$repository_root/.local must be a directory when it exists"
validate_disposable_path "$state_root" "AVIA_CANONICAL_PREPROD_STATE_DIR"
validate_disposable_path "$runtime_root" "AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR"
[[ "$state_root" != "$runtime_root" ]] || fail "state and runtime roots must be distinct"
[[ -d "$state_root" && ! -L "$state_root" ]] || fail "state directory is missing: $state_root"
[[ -d "$runtime_root" && ! -L "$runtime_root" ]] || fail "runtime directory is missing: $runtime_root"
[[ -f "$runtime_file" && ! -L "$runtime_file" ]] || fail "runtime metadata is missing: $runtime_file"
[[ -f "$compose_file" && -f "$compose_override" ]] || fail "the local HTTP Quick Tunnel profile is incomplete"
command -v docker >/dev/null 2>&1 || fail "docker is required"
command -v node >/dev/null 2>&1 || fail "node is required"
command -v curl >/dev/null 2>&1 || fail "curl is required"
docker info >/dev/null 2>&1 || fail "docker daemon is unavailable"
validate_runtime_metadata

public_origin="$(metadata_value publicOrigin)"
tunnel_pid="$(metadata_value tunnel.pid)"
tunnel_command="$(metadata_value tunnel.processCommand)"
[[ "$tunnel_pid" =~ ^[0-9]+$ ]] || fail "Quick Tunnel PID is invalid"
kill -0 "$tunnel_pid" 2>/dev/null || fail "Cloudflare Quick Tunnel process is not running"
[[ "$(ps -p "$tunnel_pid" -o command= 2>/dev/null || true)" == "$tunnel_command" ]] ||
  fail "Cloudflare Quick Tunnel process identity changed"

curl --connect-timeout 2 --max-time 5 --fail --silent --show-error --output /dev/null "$local_origin/health/ready" ||
  fail "local API readiness is not responding at $local_origin"
curl --connect-timeout 2 --max-time 5 --fail --silent --show-error --output /dev/null "$public_origin/health/ready" ||
  fail "public Quick Tunnel API readiness is not responding"
verify_public_discovery "$public_origin"

compose_environment=(
  AVIA_PREPROD_STATE_DIR="$state_root"
  AVIA_PREPROD_PROFILE="aga-preprod@1.0.0"
  AVIA_PREPROD_PROFILE_QUALIFICATION="true"
  AVIA_PREPROD_IDENTITY_NAMESPACE="canonical-aga-preprod-exercise-v1"
  AVIA_PREPROD_TRANSPORT=http
  AVIA_PREPROD_HTTP_PORT="$http_port"
  AVIA_PREPROD_WEB_ORIGIN="$public_origin"
  AVIA_PREPROD_KEYCLOAK_PUBLIC_ORIGIN="$public_origin"
  AVIA_PREPROD_PUBLIC_HOST="${public_origin#*://}"
  AVIA_PREPROD_ORIGIN_SCHEME=https
  AVIA_PREPROD_PUBLIC_TLS=true
  AVIA_PREPROD_COOKIE_SECURE=true
)
compose_command=(
  docker compose --project-name "$project_name"
  --file "$compose_file"
  --file "$compose_override"
  --profile local-preprod-loader
)
demo_identity_counts="$(
  env "${compose_environment[@]}" "${compose_command[@]}" exec --no-TTY \
    preprod-postgres psql \
      --username aviasurveil360_preprod_loader \
      --dbname aviasurveil360_local_preprod \
      --tuples-only --no-align --field-separator '|' \
      --command "
        SELECT
          (SELECT count(*) FROM identity_references WHERE email LIKE '%@synthetic.invalid'),
          (SELECT count(*) FROM user_profiles WHERE subject_id IN (
            SELECT subject_id FROM identity_references WHERE email LIKE '%@synthetic.invalid'
          )),
          (SELECT count(*) FROM desired_membership_versions
            WHERE membership_id LIKE 'CANONICAL-DEMO-MEMBERSHIP-%'
              AND revision = 1 AND membership_state = 'ACTIVE'),
          (SELECT count(*) FROM caa_department_memberships
            WHERE root_id = 'CANONICAL-DEMO-DEPARTMENT-MANAGER'
              AND status = 'ACTIVE');
      " | tr -d '[:space:]'
)"
[[ "$demo_identity_counts" == "9|9|9|1" ]] ||
  fail "demo identity count mismatch: $demo_identity_counts"

env "${compose_environment[@]}" "${compose_command[@]}" ps --format table
printf 'canonical Cloudflare Quick Tunnel verified locally: public readiness, exact OIDC issuer, and nine demo identities are healthy; external preprod not run.\n'
