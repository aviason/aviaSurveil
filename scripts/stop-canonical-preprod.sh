#!/usr/bin/env bash
set -euo pipefail

umask 077

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
state_root="${AVIA_CANONICAL_PREPROD_STATE_DIR:-$repository_root/.local/aviasurveil360-canonical-preprod}"
compose_file="$repository_root/deploy/local/compose.yaml"
project_name="${AVIA_CANONICAL_PREPROD_PROJECT:-aviasurveil360-local-preprod}"
https_port="${AVIA_PREPROD_HTTPS_PORT:-8445}"
metadata_file="$state_root/runtime.json"

fail() {
  printf 'canonical-preprod-down: %s\n' "$*" >&2
  exit 1
}

[[ -f "$metadata_file" ]] || {
  printf 'Canonical local preprod is not running (no state at %s)\n' "$state_root"
  exit 0
}
[[ "$project_name" =~ ^aviasurveil360-(local-preprod(-[a-z0-9][a-z0-9-]*)?|task-[a-z0-9][a-z0-9-]*)$ ]] ||
  fail "AVIA_CANONICAL_PREPROD_PROJECT must identify one exact AviaSurveil360 local-preprod project"
command -v node >/dev/null 2>&1 || fail "node is required"
metadata_values="$(node --input-type=module - "$metadata_file" "$project_name" "$state_root" <<'NODE'
import { readFileSync } from "node:fs";
const metadata = JSON.parse(readFileSync(process.argv[2], "utf8"));
const [project, stateDirectory] = process.argv.slice(3);
const origin = new URL(metadata.webOrigin);
if (
  metadata.schemaVersion !== "canonical-preprod-runtime/v2" ||
  metadata.project !== project ||
  metadata.stateDirectory !== stateDirectory ||
  metadata.identityProvider !== "first-party" ||
  !["http:", "https:"].includes(origin.protocol) ||
  origin.pathname !== "/" || origin.search || origin.hash
) throw new Error("canonical local preprod metadata is invalid");
process.stdout.write(`${metadata.webOrigin}\t${origin.protocol.slice(0, -1)}\t${origin.host}\n`);
NODE
)" || fail "canonical local preprod metadata is invalid"
IFS=$'\t' read -r web_origin origin_scheme public_host <<<"$metadata_values"
if [[ "$origin_scheme" == "http" ]]; then
  compose_override="$repository_root/deploy/local/compose.local-http.yaml"
else
  compose_override=""
fi

command -v docker >/dev/null 2>&1 || fail "docker is required"
docker info >/dev/null 2>&1 || fail "docker daemon is unavailable"
compose() {
  local compose_args=(--project-name "$project_name" --file "$compose_file")
  if [[ -n "$compose_override" ]]; then
    compose_args+=(--file "$compose_override")
  fi
  AVIA_PREPROD_STATE_DIR="$state_root" \
  AVIA_PREPROD_PROFILE="aga-preprod@1.0.0" \
  AVIA_PREPROD_PROFILE_QUALIFICATION="true" \
  AVIA_PREPROD_IDENTITY_NAMESPACE="canonical-aga-preprod-exercise-v1" \
  AVIA_PREPROD_HTTPS_PORT="$https_port" \
  AVIA_PREPROD_WEB_ORIGIN="$web_origin" \
  AVIA_PREPROD_PUBLIC_HOST="$public_host" \
  AVIA_PREPROD_ORIGIN_SCHEME="$origin_scheme" \
  AVIA_PREPROD_PUBLIC_TLS="$([[ "$origin_scheme" == https ]] && echo true || echo false)" \
    docker compose "${compose_args[@]}" --profile local-preprod-loader "$@"
}

compose down --volumes --remove-orphans
residue="$(
  docker ps --all --filter "label=com.docker.compose.project=$project_name" --format '{{.ID}}';
  docker volume ls --filter "label=com.docker.compose.project=$project_name" --format '{{.Name}}';
  docker network ls --filter "label=com.docker.compose.project=$project_name" --format '{{.Name}}'
)"
[[ -z "$residue" ]] || fail "task-owned Compose residue remains"
rm -rf -- "$state_root"
printf 'Canonical local preprod stopped; disposable state, containers, volumes, and networks removed.\n'
