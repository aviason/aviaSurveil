#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
state_root="${AVIA_AGA_DEMO_STATE_DIR:-$repository_root/apps/web/.local/aga-demo}"
metadata_file="$state_root/state.json"
node_command="${DEMO_NODE:-node}"

if [[ ! -f "$metadata_file" ]]; then
  printf 'AGA API demo is not running (no state at %s)\n' "$state_root" >&2
  exit 1
fi

if [[ "$node_command" = /* ]]; then
  [[ -x "$node_command" ]] || { printf 'DEMO_NODE is not executable: %s\n' "$node_command" >&2; exit 1; }
  node_path="$node_command"
else
  node_path="$(command -v "$node_command" 2>/dev/null || true)"
  [[ -n "$node_path" ]] || { printf 'Node.js is required to read demo state; set DEMO_NODE\n' >&2; exit 1; }
fi

metadata="$("$node_path" --input-type=module - "$metadata_file" <<'NODE'
import { readFileSync } from "node:fs";
const value = JSON.parse(readFileSync(process.argv[2], "utf8"));
if (value.schemaVersion !== "aga-demo-runtime/v1") throw new Error("invalid AGA demo state");
process.stdout.write(JSON.stringify(value));
NODE
)"
web_url="$(printf '%s' "$metadata" | "$node_path" -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>process.stdout.write(JSON.parse(s).webUrl))')"
api_url="$(printf '%s' "$metadata" | "$node_path" -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>process.stdout.write(JSON.parse(s).apiUrl))')"
question_count="$(printf '%s' "$metadata" | "$node_path" -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>process.stdout.write(String(JSON.parse(s).questionCount)))')"
web_container_name="$(printf '%s' "$metadata" | "$node_path" -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>process.stdout.write(JSON.parse(s).webContainerName))')"

curl --fail --silent --output /dev/null "$api_url/health/ready" || { printf 'AGA API demo API is not responding at %s\n' "$api_url" >&2; exit 1; }
[[ "$(docker inspect -f '{{.State.Running}}' "$web_container_name" 2>/dev/null || true)" = true ]] || { printf 'AGA API demo web container is not running: %s\n' "$web_container_name" >&2; exit 1; }
curl --fail --silent --output /dev/null "$web_url" || { printf 'AGA API demo web is not responding at %s\n' "$web_url" >&2; exit 1; }
printf 'AGA API demo responding\nURL: %s\nAPI: %s\nQuestions: %s\n' "$web_url" "$api_url" "$question_count"
