#!/usr/bin/env bash
set -euo pipefail

umask 077
repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
state_root="${AVIA_AGA_DEMO_STATE_DIR:-$repository_root/apps/web/.local/aga-demo}"
metadata_file="$state_root/state.json"
node_command="${DEMO_NODE:-node}"

fail() {
  printf 'aga-demo-stop: %s\n' "$*" >&2
  exit 1
}

if [[ ! -f "$metadata_file" ]]; then
  printf 'No API-backed AGA demo state is recorded at %s\n' "$state_root"
  exit 0
fi

if [[ "$node_command" = /* ]]; then
  [[ -x "$node_command" ]] || fail "DEMO_NODE is not executable: $node_command"
  node_path="$node_command"
else
  node_path="$(command -v "$node_command" 2>/dev/null || true)"
  [[ -n "$node_path" ]] || fail "Node.js is required to read demo state; set DEMO_NODE"
fi

read_metadata() {
  "$node_path" --input-type=module - "$metadata_file" <<'NODE'
import { readFileSync } from "node:fs";
const value = JSON.parse(readFileSync(process.argv[2], "utf8"));
if (value.schemaVersion !== "aga-demo-runtime/v1" || typeof value.stateDirectory !== "string" || !value.stateDirectory.startsWith("/") || typeof value.webContainerName !== "string" || typeof value.webContainerId !== "string" || !/^[a-f0-9]{12,64}$/u.test(value.webContainerId)) throw new Error("invalid AGA demo state");
process.stdout.write(`${JSON.stringify(value)}\n`);
NODE
}

metadata="$(read_metadata)"
state_directory="$(printf '%s' "$metadata" | "$node_path" -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>process.stdout.write(JSON.parse(s).stateDirectory+"\n"))')"
web_container_name="$(printf '%s' "$metadata" | "$node_path" -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>process.stdout.write(JSON.parse(s).webContainerName+"\n"))')"
web_container_id="$(printf '%s' "$metadata" | "$node_path" -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>process.stdout.write(JSON.parse(s).webContainerId+"\n"))')"
api_port="$(printf '%s' "$metadata" | "$node_path" -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>process.stdout.write(new URL(JSON.parse(s).apiUrl).port+"\n"))')"
oidc_host="$(printf '%s' "$metadata" | "$node_path" -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>process.stdout.write(new URL(JSON.parse(s).oidcUrl).hostname+"\n"))')"
oidc_port="$(printf '%s' "$metadata" | "$node_path" -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>process.stdout.write(new URL(JSON.parse(s).oidcUrl).port+"\n"))')"

if docker container inspect "$web_container_name" >/dev/null 2>&1; then
  actual_container_id="$(docker inspect -f '{{.Id}}' "$web_container_name")"
  component="$(docker inspect -f '{{index .Config.Labels "com.aviasurveil360.component"}}' "$web_container_name")"
  [[ "$actual_container_id" = "$web_container_id" && "$component" = "aga-demo-web" ]] || fail "web container identity does not match demo state: $web_container_name"
  docker rm --force "$web_container_name"
fi

[[ -d "$state_directory" && ! -L "$state_directory" ]] || fail "preprod state directory is missing: $state_directory"
AVIA_PREPROD_STATE_DIR="$state_directory" \
AVIA_PREPROD_AGA_OIDC_HOST="$oidc_host" \
AVIA_PREPROD_AGA_OIDC_PORT="$oidc_port" \
AVIA_PREPROD_AGA_API_PORT="$api_port" \
  docker compose --project-name aviasurveil360-local-preprod \
    --file "$repository_root/deploy/local/compose.yaml" \
    --profile local-preprod-loader \
    --profile aga-candidate-demo \
    --profile aga-candidate-demo-oidc-fixture \
    --profile aga-demo-workspace-loader \
    --profile preproddemo down --volumes --remove-orphans

residue="$(docker ps --all --filter 'label=com.docker.compose.project=aviasurveil360-local-preprod' --format '{{.ID}}'; docker volume ls --filter 'label=com.docker.compose.project=aviasurveil360-local-preprod' --format '{{.Name}}'; docker network ls --filter 'label=com.docker.compose.project=aviasurveil360-local-preprod' --format '{{.Name}}')"
[[ -z "$residue" ]] || fail "Compose residue remains; state was retained at $state_root"

rm -rf -- "$state_root"
printf 'AGA API demo stopped; disposable containers, volumes, networks, and temporary credentials removed.\n'
