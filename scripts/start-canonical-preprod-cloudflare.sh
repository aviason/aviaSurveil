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
quick_tunnel_parser="$repository_root/scripts/canonical-preprod-quick-tunnel-url.mjs"
quick_tunnel_launcher="$repository_root/scripts/canonical-preprod-cloudflare-launcher.mjs"
local_origin="http://127.0.0.1:${http_port}"
realm_path="/identity/realms/aviasurveil360-local-preprod"
placeholder_log="$runtime_root/placeholder.log"
tunnel_log="$runtime_root/cloudflared.log"
placeholder_pid_file="$runtime_root/placeholder.pid"
tunnel_pid_file="$runtime_root/cloudflared.pid"
runtime_file="$runtime_root/runtime.json"
public_origin=""
placeholder_command=""
tunnel_command=""
cleanup_on_failure=true

fail() {
  printf 'canonical-preprod-cloudflare-up: %s\n' "$*" >&2
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

assert_no_project_residue() {
  local residue
  residue="$(
    docker ps --all --filter "label=com.docker.compose.project=$project_name" --format '{{.ID}}'
    docker volume ls --filter "label=com.docker.compose.project=$project_name" --format '{{.Name}}'
    docker network ls --filter "label=com.docker.compose.project=$project_name" --format '{{.Name}}'
  )"
  [[ -z "$residue" ]] || fail "task-owned Compose residue remains for $project_name"
}

process_command() {
  ps -p "$1" -o command= 2>/dev/null || true
}

capture_process_command() {
  local pid="$1"
  local command=""
  for _ in {1..40}; do
    command="$(process_command "$pid")"
    [[ -n "$command" ]] && {
      printf '%s' "$command"
      return 0
    }
    kill -0 "$pid" 2>/dev/null || return 1
    sleep 0.05
  done
  return 1
}

stop_owned_process() {
  local pid_file="$1"
  local expected_command="$2"
  local label="$3"
  [[ -f "$pid_file" ]] || return 0

  local pid
  pid="$(tr -d '[:space:]' <"$pid_file")"
  [[ "$pid" =~ ^[0-9]+$ ]] || {
    printf 'canonical-preprod-cloudflare-up: invalid %s PID file\n' "$label" >&2
    return 1
  }
  if ! kill -0 "$pid" 2>/dev/null; then
    rm -f -- "$pid_file"
    return 0
  fi

  local actual_command
  actual_command="$(process_command "$pid")"
  [[ -n "$expected_command" && "$actual_command" == "$expected_command" ]] || {
    printf 'canonical-preprod-cloudflare-up: refusing to stop an unverified %s process\n' "$label" >&2
    return 1
  }

  kill -TERM "$pid"
  for _ in {1..40}; do
    kill -0 "$pid" 2>/dev/null || break
    sleep 0.25
  done
  if kill -0 "$pid" 2>/dev/null; then
    actual_command="$(process_command "$pid")"
    [[ "$actual_command" == "$expected_command" ]] || {
      printf 'canonical-preprod-cloudflare-up: refusing to force-stop a replaced %s process\n' "$label" >&2
      return 1
    }
    kill -KILL "$pid"
  fi
  rm -f -- "$pid_file"
}

compose_down() {
  local origin="${public_origin:-http://localhost:${http_port}}"
  local public_host="${origin#*://}"
  local public_scheme="${origin%%://*}"
  local public_tls=false
  [[ "$origin" == https://* ]] && public_tls=true
  AVIA_PREPROD_STATE_DIR="$state_root" \
  AVIA_PREPROD_PROFILE="aga-preprod@1.0.0" \
  AVIA_PREPROD_PROFILE_QUALIFICATION="true" \
  AVIA_PREPROD_IDENTITY_NAMESPACE="canonical-aga-preprod-exercise-v1" \
  AVIA_PREPROD_TRANSPORT=http \
  AVIA_PREPROD_HTTP_PORT="$http_port" \
  AVIA_PREPROD_WEB_ORIGIN="$origin" \
  AVIA_PREPROD_KEYCLOAK_PUBLIC_ORIGIN="$origin" \
  AVIA_PREPROD_PUBLIC_HOST="$public_host" \
  AVIA_PREPROD_ORIGIN_SCHEME="$public_scheme" \
  AVIA_PREPROD_PUBLIC_TLS="$public_tls" \
  AVIA_PREPROD_COOKIE_SECURE="$public_tls" \
    docker compose --project-name "$project_name" \
      --file "$compose_file" \
      --file "$compose_override" \
      --profile local-preprod-loader down --volumes --remove-orphans
}

remove_owned_root() {
  local value="$1"
  [[ -e "$value" ]] || return 0
  [[ ! -L "$value" ]] || {
    printf 'canonical-preprod-cloudflare-up: refusing to remove symlinked state: %s\n' "$value" >&2
    return 1
  }
  rm -rf -- "$value"
}

cleanup() {
  local status=$?
  local exposure_removed=true
  trap - EXIT HUP INT TERM
  if [[ "$cleanup_on_failure" == true && "$status" -ne 0 ]]; then
    # Exposure is removed before any task-owned Compose state is deleted.
    stop_owned_process "$tunnel_pid_file" "$tunnel_command" cloudflared || exposure_removed=false
    stop_owned_process "$placeholder_pid_file" "$placeholder_command" placeholder || exposure_removed=false
    if [[ "$exposure_removed" == true ]]; then
      compose_down >/dev/null 2>&1 || true
      remove_owned_root "$state_root" || true
      remove_owned_root "$runtime_root" || true
    else
      printf 'canonical-preprod-cloudflare-up: exposure identity could not be verified; preserving exact task state for escalation\n' >&2
    fi
  fi
  exit "$status"
}

wait_for_http() {
  local url="$1"
  local label="$2"
  for _ in {1..120}; do
    if curl --connect-timeout 2 --max-time 5 --fail --silent --show-error --output /dev/null "$url"; then
      return 0
    fi
    sleep 0.5
  done
  fail "$label did not become ready: $url"
}

verify_public_discovery() {
  node --input-type=module - "$public_origin" "$realm_path" <<'NODE'
const [origin, realmPath] = process.argv.slice(2);
const expectedIssuer = `${origin}${realmPath}`;
const response = await fetch(`${expectedIssuer}/.well-known/openid-configuration`, {
  signal: AbortSignal.timeout(5000),
});
if (!response.ok) throw new Error(`OIDC discovery returned ${response.status}`);
const discovery = await response.json();
if (discovery.issuer !== expectedIssuer) {
  throw new Error(`OIDC issuer mismatch: expected ${expectedIssuer}, received ${discovery.issuer}`);
}
NODE
}

wait_for_public_discovery() {
  for _ in {1..120}; do
    if verify_public_discovery >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  verify_public_discovery || fail "public OIDC discovery did not publish the exact Quick Tunnel issuer"
}

[[ "$http_port" =~ ^[0-9]+$ && "$http_port" -ge 1024 && "$http_port" -le 65535 ]] ||
  fail "AVIA_PREPROD_HTTP_PORT must be a user-space TCP port"
[[ "$project_name" =~ ^aviasurveil360-local-preprod-cloudflare(-[a-z0-9][a-z0-9-]*)?$ ]] ||
  fail "AVIA_CANONICAL_PREPROD_PROJECT must be a task-owned Cloudflare disposable project"
[[ -f "$compose_file" && -f "$compose_override" && -f "$quick_tunnel_parser" && -f "$quick_tunnel_launcher" ]] ||
  fail "the local HTTP Quick Tunnel profile is incomplete"
[[ ! -L "$repository_root/.local" ]] || fail "$repository_root/.local must not be a symlink"
[[ ! -e "$repository_root/.local" || -d "$repository_root/.local" ]] ||
  fail "$repository_root/.local must be a directory when it exists"
validate_disposable_path "$state_root" "AVIA_CANONICAL_PREPROD_STATE_DIR"
validate_disposable_path "$runtime_root" "AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR"
[[ "$state_root" != "$runtime_root" ]] || fail "state and runtime roots must be distinct"
[[ ! -e "$state_root" && ! -L "$state_root" ]] ||
  fail "state directory already exists; stop/clean the exact disposable profile first: $state_root"
[[ ! -e "$runtime_root" && ! -L "$runtime_root" ]] ||
  fail "runtime directory already exists; refuse to reuse or kill stale task-owned resources: $runtime_root"
command -v docker >/dev/null 2>&1 || fail "docker is required"
command -v node >/dev/null 2>&1 || fail "node is required"
command -v curl >/dev/null 2>&1 || fail "curl is required"
cloudflared_binary="$(command -v cloudflared)"
[[ -n "$cloudflared_binary" ]] || fail "cloudflared is required for the temporary HTTPS URL"
docker info >/dev/null 2>&1 || fail "docker daemon is unavailable"
assert_no_project_residue

mkdir -p "$runtime_root/cloudflared-home/.cloudflared" \
  "$runtime_root/xdg-config" \
  "$runtime_root/xdg-data" \
  "$runtime_root/tmp"
# The task-owned .cloudflared directory deliberately stays empty: Quick Tunnel
# mode must not discover a developer config file, token, or named tunnel.
chmod 0700 "$runtime_root" "$runtime_root/cloudflared-home" \
  "$runtime_root/cloudflared-home/.cloudflared" "$runtime_root/xdg-config" \
  "$runtime_root/xdg-data" "$runtime_root/tmp"
trap cleanup EXIT HUP INT TERM

: >"$placeholder_log"
: >"$tunnel_log"
node "$repository_root/scripts/canonical-preprod-tunnel-placeholder.mjs" "$http_port" \
  >"$placeholder_log" 2>&1 < /dev/null &
placeholder_pid=$!
printf '%s\n' "$placeholder_pid" >"$placeholder_pid_file"
if ! placeholder_command="$(capture_process_command "$placeholder_pid")"; then
  fail "placeholder process did not expose a verifiable identity"
fi

for _ in {1..80}; do
  if curl --fail --silent --output /dev/null "$local_origin/__canonical_preprod_tunnel_placeholder_ready"; then
    break
  fi
  sleep 0.25
done
curl --fail --silent --output /dev/null "$local_origin/__canonical_preprod_tunnel_placeholder_ready" ||
  fail "placeholder origin did not become ready on port $http_port"

# No login, token, account, named tunnel, DNS, Access, or external-preprod
# configuration is used: this is only an anonymous trycloudflare Quick Tunnel.
# The launcher uses a new process group and detached stdio so closing this
# controlling shell cannot terminate the task-owned Cloudflare process.
if ! tunnel_pid="$(node "$quick_tunnel_launcher" "$cloudflared_binary" "$http_port" "$runtime_root")"; then
  fail "cloudflared detached launcher could not start the temporary URL; see $tunnel_log"
fi
[[ "$tunnel_pid" =~ ^[0-9]+$ ]] || fail "cloudflared detached launcher returned an invalid PID"
if ! tunnel_command="$(capture_process_command "$tunnel_pid")"; then
  fail "cloudflared process did not expose a verifiable identity"
fi

for _ in {1..240}; do
  if public_origin="$(node "$quick_tunnel_parser" --file "$tunnel_log" 2>/dev/null)"; then
    break
  fi
  if ! kill -0 "$tunnel_pid" 2>/dev/null; then
    fail "cloudflared exited before publishing a temporary URL; see $tunnel_log"
  fi
  sleep 0.5
done
[[ -n "$public_origin" ]] || {
  node "$quick_tunnel_parser" --file "$tunnel_log" >&2 || true
  fail "timed out waiting for exactly one Cloudflare Quick Tunnel URL; see $tunnel_log"
}

stop_owned_process "$placeholder_pid_file" "$placeholder_command" placeholder ||
  fail "placeholder process ownership could not be verified"
placeholder_command=""

AVIA_PREPROD_TRANSPORT=http \
AVIA_PREPROD_HTTP_PORT="$http_port" \
AVIA_PREPROD_WEB_ORIGIN="$public_origin" \
AVIA_PREPROD_PUBLIC_TLS=true \
AVIA_PREPROD_COOKIE_SECURE=true \
AVIA_CANONICAL_PREPROD_PROJECT="$project_name" \
AVIA_CANONICAL_PREPROD_STATE_DIR="$state_root" \
  "$repository_root/scripts/start-canonical-preprod-http.sh"

wait_for_http "$local_origin/health/ready" "local canonical API readiness"
wait_for_http "$public_origin/health/ready" "public Quick Tunnel API readiness"
wait_for_public_discovery

node --input-type=module - \
  "$runtime_file" "$project_name" "$state_root" "$runtime_root" "$local_origin" \
  "$public_origin" "$tunnel_pid" "$tunnel_pid_file" "$tunnel_log" \
  "http://127.0.0.1:${http_port}" "$tunnel_command" <<'NODE'
import { writeFileSync } from "node:fs";
const [
  runtimeFile,
  project,
  stateDirectory,
  runtimeDirectory,
  localOrigin,
  publicOrigin,
  pid,
  pidFile,
  logFile,
  localTunnelOrigin,
  processCommand,
] = process.argv.slice(2);
writeFileSync(
  runtimeFile,
  `${JSON.stringify({
    schemaVersion: "canonical-preprod-cloudflare-runtime/v1",
    project,
    stateDirectory,
    runtimeDirectory,
    localOrigin,
    publicOrigin,
    cookieSecure: true,
    tunnel: {
      pid: Number(pid),
      pidFile,
      logFile,
      localOrigin: localTunnelOrigin,
      processCommand,
    },
    donorRuntime: "disabled",
    externalPreprod: "not run",
  }, null, 2)}\n`,
  { mode: 0o600 },
);
NODE
chmod 0600 "$runtime_file"
cleanup_on_failure=false
printf 'Canonical local preprod is available through a disposable Cloudflare Quick Tunnel at %s\n' "$public_origin"
printf 'Local origin: %s (the random public URL is the secure-cookie/OIDC origin)\n' "$local_origin"
printf 'External preprod: not run\n'
