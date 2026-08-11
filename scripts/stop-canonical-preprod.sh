#!/usr/bin/env bash
set -euo pipefail

umask 077

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
state_root="${AVIA_CANONICAL_PREPROD_STATE_DIR:-$repository_root/.local/aviasurveil360-canonical-preprod}"
compose_file="$repository_root/deploy/local/compose.yaml"
project_name="${AVIA_CANONICAL_PREPROD_PROJECT:-aviasurveil360-local-preprod}"
https_port="${AVIA_PREPROD_HTTPS_PORT:-8445}"
metadata_file="$state_root/runtime.json"
web_origin="https://localhost:${https_port}"

fail() {
  printf 'canonical-preprod-down: %s\n' "$*" >&2
  exit 1
}

[[ -f "$metadata_file" ]] || {
  printf 'Canonical local preprod is not running (no state at %s)\n' "$state_root"
  exit 0
}
[[ "$https_port" =~ ^[0-9]+$ && "$https_port" -ge 1024 && "$https_port" -le 65535 ]] ||
  fail "AVIA_PREPROD_HTTPS_PORT must be a user-space TCP port"
[[ "$project_name" =~ ^aviasurveil360-(local-preprod(-[a-z0-9][a-z0-9-]*)?|task-[a-z0-9][a-z0-9-]*)$ ]] ||
  fail "AVIA_CANONICAL_PREPROD_PROJECT must identify one exact AviaSurveil360 local-preprod project"
command -v node >/dev/null 2>&1 || fail "node is required"
node --input-type=module - "$metadata_file" "$project_name" "$state_root" "$web_origin" <<'NODE'
import { readFileSync } from "node:fs";
const metadata = JSON.parse(readFileSync(process.argv[2], "utf8"));
const [project, stateDirectory, webOrigin] = process.argv.slice(3);
if (
  metadata.schemaVersion !== "canonical-preprod-runtime/v1" ||
  metadata.project !== project ||
  metadata.stateDirectory !== stateDirectory ||
  metadata.webOrigin !== webOrigin
) throw new Error("canonical local preprod metadata is invalid");
NODE

command -v docker >/dev/null 2>&1 || fail "docker is required"
docker info >/dev/null 2>&1 || fail "docker daemon is unavailable"
compose() {
  AVIA_PREPROD_STATE_DIR="$state_root" \
  AVIA_PREPROD_PROFILE="aga-preprod@1.0.0" \
  AVIA_PREPROD_PROFILE_QUALIFICATION="true" \
  AVIA_PREPROD_IDENTITY_NAMESPACE="canonical-aga-preprod-exercise-v1" \
  AVIA_PREPROD_HTTPS_PORT="$https_port" \
    docker compose --project-name "$project_name" --file "$compose_file" \
      --profile local-preprod-loader "$@"
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
