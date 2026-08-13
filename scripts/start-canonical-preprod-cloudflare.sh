#!/usr/bin/env bash
set -euo pipefail

umask 077

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
http_port="${AVIA_PREPROD_HTTP_PORT:-8085}"
project_name="${AVIA_CANONICAL_PREPROD_PROJECT:-aviasurveil360-local-preprod-cloudflare}"
state_root="${AVIA_CANONICAL_PREPROD_STATE_DIR:-$repository_root/.local/aviasurveil360-canonical-preprod-cloudflare}"
runtime_root="${AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR:-$repository_root/.local/aviasurveil360-canonical-preprod-cloudflare-tunnel}"
tunnel_mode="${AVIA_PREPROD_CLOUDFLARE_MODE:-quick}"
public_hostname="${AVIA_PREPROD_PUBLIC_HOSTNAME:-demo.aviasurveil.com}"
keychain_service="${AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_SERVICE:-com.aviasurveil360.cloudflare-tunnel}"
keychain_account="${AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_ACCOUNT:-$public_hostname}"
compose_file="$repository_root/deploy/local/compose.yaml"
compose_override="$repository_root/deploy/local/compose.local-http.yaml"
quick_tunnel_parser="$repository_root/scripts/canonical-preprod-quick-tunnel-url.mjs"
quick_tunnel_launcher="$repository_root/scripts/canonical-preprod-cloudflare-launcher.mjs"
named_tunnel_launcher="$repository_root/scripts/canonical-preprod-cloudflare-named-launcher.mjs"
tunnel_token_validator="$repository_root/scripts/validate-cloudflare-tunnel-token.mjs"
local_origin="http://127.0.0.1:${http_port}"
discovery_path="/identity/.well-known/openid-configuration"
placeholder_log="$runtime_root/placeholder.log"
tunnel_log="$runtime_root/cloudflared.log"
placeholder_pid_file="$runtime_root/placeholder.pid"
tunnel_pid_file="$runtime_root/cloudflared.pid"
runtime_file="$runtime_root/runtime.json"
public_origin=""
tunnel_label="Cloudflare Quick Tunnel"
placeholder_command=""
tunnel_command=""
actual_tunnel_command=""
cleanup_on_failure=true

fail() {
  printf 'canonical-preprod-cloudflare-up: %s\n' "$*" >&2
  exit 1
}

progress() {
  printf 'canonical-preprod-cloudflare-up: %s\n' "$*"
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

print_sanitized_tunnel_log() {
  [[ -s "$tunnel_log" ]] || return 0
  node --input-type=module - "$tunnel_log" <<'NODE' || true
import { readFileSync } from "node:fs";

const [logPath] = process.argv.slice(2);
let tail = readFileSync(logPath, "utf8").slice(-65536);
tail = tail
  .replace(/eyJ[A-Za-z0-9+/_=-]{16,}/gu, "[REDACTED_CONNECTOR_TOKEN]")
  .replace(/(--token\s+)(\S+)/giu, "$1[REDACTED_CONNECTOR_TOKEN]");
const lines = tail.split(/\r?\n/u).slice(-40).join("\n").trim();
if (lines) {
  process.stderr.write(`canonical-preprod-cloudflare-up: sanitized cloudflared log tail:\n${lines}\n`);
}
NODE
}

fail_with_tunnel_log() {
  local message="$1"
  print_sanitized_tunnel_log
  fail "$message"
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
  AVIA_PREPROD_PUBLIC_HOST="$public_host" \
  AVIA_PREPROD_ORIGIN_SCHEME="$public_scheme" \
  AVIA_PREPROD_PUBLIC_TLS="$public_tls" \
  AVIA_PREPROD_COOKIE_SECURE="$public_tls" \
    docker compose --project-name "$project_name" \
      --file "$compose_file" \
      --file "$compose_override" \
      --profile local-preprod-loader down --volumes --remove-orphans
}

prebuild_images() {
  local build_origin="https://prebuild.invalid"
  AVIA_PREPROD_STATE_DIR="$state_root" \
  AVIA_PREPROD_PROFILE="aga-preprod@1.0.0" \
  AVIA_PREPROD_PROFILE_QUALIFICATION="true" \
  AVIA_PREPROD_IDENTITY_NAMESPACE="canonical-aga-preprod-exercise-v1" \
  AVIA_PREPROD_TRANSPORT=http \
  AVIA_PREPROD_HTTP_PORT="$http_port" \
  AVIA_PREPROD_WEB_ORIGIN="$build_origin" \
  AVIA_PREPROD_PUBLIC_HOST="prebuild.invalid" \
  AVIA_PREPROD_ORIGIN_SCHEME=https \
  AVIA_PREPROD_PUBLIC_TLS=true \
  AVIA_PREPROD_COOKIE_SECURE=true \
    docker compose --project-name "$project_name" \
      --file "$compose_file" \
      --file "$compose_override" \
      --profile local-preprod-loader build \
        preprod-migration \
        preprod-normal-runtime-role-provisioner \
        preprod-canonical-aga-loader \
        preprod-canonical-demo-identity-loader \
        preprod-auth \
        preprod-minio \
        preprod-api \
        preprod-worker \
        preprod-web-http \
        preprod-gateway
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

wait_for_public_dns_publication() {
  node --input-type=module - "$public_origin" <<'NODE'
const [origin] = process.argv.slice(2);
const hostname = new URL(origin).hostname;
const endpoint = new URL("https://cloudflare-dns.com/dns-query");
endpoint.searchParams.set("name", hostname);
endpoint.searchParams.set("type", "A");
const deadline = Date.now() + 120_000;

while (Date.now() < deadline) {
  try {
    const response = await fetch(endpoint, {
      headers: { Accept: "application/dns-json" },
      signal: AbortSignal.timeout(5_000),
    });
    if (response.ok) {
      const payload = await response.json();
      const answers = Array.isArray(payload.Answer) ? payload.Answer : [];
      if (payload.Status === 0 && answers.some((answer) => [1, 28].includes(answer.type))) {
        process.exit(0);
      }
    }
  } catch {
    // A new anonymous hostname may not be visible at the edge yet. Retry the
    // read-only authoritative query without poisoning the OS resolver cache.
  }
  await new Promise((resolve) => setTimeout(resolve, 500));
}
process.exit(1);
NODE
}

verify_public_discovery() {
  node --input-type=module - "$public_origin" "$discovery_path" <<'NODE'
const [origin, discoveryPath] = process.argv.slice(2);
const expectedIssuer = `${origin}/identity`;
const response = await fetch(`${origin}${discoveryPath}`, {
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
  verify_public_discovery || fail "public OIDC discovery did not publish the exact configured issuer"
}

[[ "$http_port" =~ ^[0-9]+$ && "$http_port" -ge 1024 && "$http_port" -le 65535 ]] ||
  fail "AVIA_PREPROD_HTTP_PORT must be a user-space TCP port"
[[ "$project_name" =~ ^aviasurveil360-local-preprod-cloudflare(-[a-z0-9][a-z0-9-]*)?$ ]] ||
  fail "AVIA_CANONICAL_PREPROD_PROJECT must be a task-owned Cloudflare disposable project"
case "$tunnel_mode" in
  quick)
    [[ -f "$quick_tunnel_parser" && -f "$quick_tunnel_launcher" ]] ||
      fail "the local HTTP Quick Tunnel profile is incomplete"
    ;;
  named)
    tunnel_label="Cloudflare named Tunnel"
    [[ "$public_hostname" =~ ^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$ ]] ||
      fail "AVIA_PREPROD_PUBLIC_HOSTNAME must be a lowercase DNS hostname without a scheme, port, or path"
    [[ "$keychain_service" =~ ^[A-Za-z0-9._:-]+$ ]] ||
      fail "AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_SERVICE contains unsupported characters"
    [[ "$keychain_account" =~ ^[A-Za-z0-9._:@-]+$ ]] ||
      fail "AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_ACCOUNT contains unsupported characters"
    [[ -f "$named_tunnel_launcher" ]] || fail "the named Tunnel launcher is missing"
    [[ -f "$tunnel_token_validator" ]] || fail "the connector-token validator is missing"
    [[ -x /usr/bin/security ]] || fail "macOS Keychain is required for the named Tunnel profile"
    /usr/bin/security find-generic-password \
      -a "$keychain_account" \
      -s "$keychain_service" >/dev/null 2>&1 ||
      fail "named Tunnel connector credential is missing; run make preprod-cloudflare-demo-token first"
    public_origin="https://$public_hostname"
    ;;
  *) fail "AVIA_PREPROD_CLOUDFLARE_MODE must be quick or named" ;;
esac
[[ -f "$compose_file" && -f "$compose_override" ]] ||
  fail "the local HTTP Cloudflare profile is incomplete"
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
if [[ "$tunnel_mode" == named ]]; then
  if ! /usr/bin/security find-generic-password \
    -a "$keychain_account" \
    -s "$keychain_service" \
    -w | node "$tunnel_token_validator"; then
    fail "named Tunnel connector credential is malformed or truncated; run make preprod-cloudflare-demo-token and paste the complete eyJ... value"
  fi
fi
docker info >/dev/null 2>&1 || fail "docker daemon is unavailable"
assert_no_project_residue

# Build every local image before creating the short-lived anonymous public
# endpoint. The random URL is then spent only on startup and qualification,
# never on compiling the current worktree.
prebuild_images
progress "local images are ready; starting the $tunnel_label connector and startup placeholder"

mkdir -p "$runtime_root/cloudflared-home/.cloudflared" \
  "$runtime_root/xdg-config" \
  "$runtime_root/xdg-data" \
  "$runtime_root/tmp"
# The task-owned .cloudflared directory deliberately stays empty. Quick mode
# must not discover developer state, and named mode is remotely configured and
# receives its connector credential through a one-shot inherited pipe.
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

# The launcher uses a new process group and detached stdio so closing this
# controlling shell cannot terminate the task-owned Cloudflare process. Quick
# mode remains anonymous; named mode retrieves only its tunnel-scoped connector
# credential from Keychain and passes it through an inherited pipe.
if [[ "$tunnel_mode" == quick ]]; then
  tunnel_command="$cloudflared_binary tunnel --url $local_origin --protocol http2"
  if ! tunnel_pid="$(node "$quick_tunnel_launcher" "$cloudflared_binary" "$http_port" "$runtime_root")"; then
    fail_with_tunnel_log "cloudflared detached launcher could not start the temporary URL"
  fi
else
  tunnel_command="$cloudflared_binary tunnel --no-autoupdate --protocol http2 run --token-file /dev/fd/3"
  if ! tunnel_pid="$(node "$named_tunnel_launcher" \
    "$cloudflared_binary" "$runtime_root" "$keychain_service" "$keychain_account")"; then
    fail_with_tunnel_log "named Tunnel detached launcher could not start; verify the stored connector token and the dashboard Tunnel status"
  fi
fi
[[ "$tunnel_pid" =~ ^[0-9]+$ ]] || fail "cloudflared detached launcher returned an invalid PID"
if ! actual_tunnel_command="$(capture_process_command "$tunnel_pid")"; then
  fail_with_tunnel_log "cloudflared exited before its process identity could be verified"
fi
[[ "$actual_tunnel_command" == "$tunnel_command" ]] ||
  fail "cloudflared process identity does not match the exact launcher command"
tunnel_command="$actual_tunnel_command"
progress "$tunnel_label connector is running; checking DNS and the public route"

if [[ "$tunnel_mode" == quick ]]; then
  for _ in {1..240}; do
    if public_origin="$(node "$quick_tunnel_parser" --file "$tunnel_log" 2>/dev/null)"; then
      break
    fi
    if ! kill -0 "$tunnel_pid" 2>/dev/null; then
      fail_with_tunnel_log "cloudflared exited before publishing a temporary URL"
    fi
    sleep 0.5
  done
  [[ -n "$public_origin" ]] || {
    node "$quick_tunnel_parser" --file "$tunnel_log" >&2 || true
    fail "timed out waiting for exactly one Cloudflare Quick Tunnel URL; see $tunnel_log"
  }
fi

# Query Cloudflare DNS over HTTPS before the first system-resolver lookup. A
# premature NXDOMAIN can otherwise remain cached locally after Cloudflare has
# published the newly generated random hostname.
wait_for_public_dns_publication ||
  fail "Cloudflare did not publish authoritative DNS for the configured URL: $public_origin"
wait_for_http "$public_origin/__canonical_preprod_tunnel_placeholder_ready" \
  "public $tunnel_label placeholder readiness (verify the dashboard route targets $local_origin)"
progress "public route is ready; replacing the startup placeholder with canonical local preprod"

stop_owned_process "$placeholder_pid_file" "$placeholder_command" placeholder ||
  fail "placeholder process ownership could not be verified"
placeholder_command=""

AVIA_PREPROD_TRANSPORT=http \
AVIA_PREPROD_HTTP_PORT="$http_port" \
AVIA_PREPROD_WEB_ORIGIN="$public_origin" \
AVIA_PREPROD_PUBLIC_TLS=true \
AVIA_PREPROD_COOKIE_SECURE=true \
AVIA_PREPROD_SKIP_BUILD=true \
AVIA_CANONICAL_PREPROD_PROJECT="$project_name" \
AVIA_CANONICAL_PREPROD_STATE_DIR="$state_root" \
  "$repository_root/scripts/start-canonical-preprod-http.sh"

wait_for_http "$local_origin/health/ready" "local canonical API readiness"
wait_for_http "$public_origin/health/ready" "public $tunnel_label API readiness"
wait_for_public_discovery

node --input-type=module - \
  "$runtime_file" "$project_name" "$state_root" "$runtime_root" "$local_origin" \
  "$public_origin" "$tunnel_pid" "$tunnel_pid_file" "$tunnel_log" \
  "http://127.0.0.1:${http_port}" "$tunnel_command" "$tunnel_mode" \
  "$public_hostname" "$keychain_service" "$keychain_account" <<'NODE'
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
  tunnelMode,
  publicHostname,
  keychainService,
  keychainAccount,
] = process.argv.slice(2);
const credentialReference = tunnelMode === "named"
  ? { kind: "macos-keychain", service: keychainService, account: keychainAccount }
  : undefined;
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
      mode: tunnelMode,
      pid: Number(pid),
      pidFile,
      logFile,
      localOrigin: localTunnelOrigin,
      processCommand,
      ...(credentialReference ? { publicHostname, credentialReference } : {}),
    },
    donorRuntime: "disabled",
    externalPreprod: "not run",
  }, null, 2)}\n`,
  { mode: 0o600 },
);
NODE
chmod 0600 "$runtime_file"
cleanup_on_failure=false
if [[ "$tunnel_mode" == quick ]]; then
  printf 'Canonical local preprod is available through a disposable Cloudflare Quick Tunnel at %s\n' "$public_origin"
  printf 'Local origin: %s (the random public URL is the secure-cookie/OIDC origin)\n' "$local_origin"
else
  printf 'Canonical local preprod is available through the named Cloudflare Tunnel at %s\n' "$public_origin"
  printf 'Local origin: %s (the named public URL is the secure-cookie/OIDC origin)\n' "$local_origin"
fi
printf 'External preprod: not run\n'
