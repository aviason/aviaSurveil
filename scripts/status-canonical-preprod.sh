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
  printf 'canonical-preprod-status: %s\n' "$*" >&2
  exit 1
}

[[ -f "$metadata_file" ]] || fail "canonical local preprod is not initialized"
[[ "$https_port" =~ ^[0-9]+$ && "$https_port" -ge 1024 && "$https_port" -le 65535 ]] ||
  fail "AVIA_PREPROD_HTTPS_PORT must be a user-space TCP port"
[[ "$project_name" =~ ^aviasurveil360-(local-preprod(-[a-z0-9][a-z0-9-]*)?|task-[a-z0-9][a-z0-9-]*)$ ]] ||
  fail "AVIA_CANONICAL_PREPROD_PROJECT must identify one exact AviaSurveil360 local-preprod project"
command -v docker >/dev/null 2>&1 || fail "docker is required"
command -v node >/dev/null 2>&1 || fail "node is required"
metadata="$(node --input-type=module - "$metadata_file" "$project_name" "$state_root" "$web_origin" <<'NODE'
import { readFileSync } from "node:fs";
const metadata = JSON.parse(readFileSync(process.argv[2], "utf8"));
const [project, stateDirectory, webOrigin] = process.argv.slice(3);
if (
  metadata.schemaVersion !== "canonical-preprod-runtime/v1" ||
  metadata.project !== project ||
  metadata.stateDirectory !== stateDirectory ||
  metadata.webOrigin !== webOrigin ||
  metadata.profile !== "aga-preprod@1.0.0" ||
  metadata.identityNamespace !== "canonical-aga-preprod-exercise-v1" ||
  metadata.donorRuntime !== "disabled" ||
  metadata.externalPreprod !== "not run"
) throw new Error("canonical local preprod metadata is invalid");
process.stdout.write(JSON.stringify(metadata));
NODE
)"

compose() {
  AVIA_PREPROD_STATE_DIR="$state_root" \
  AVIA_PREPROD_PROFILE="aga-preprod@1.0.0" \
  AVIA_PREPROD_PROFILE_QUALIFICATION="true" \
  AVIA_PREPROD_IDENTITY_NAMESPACE="canonical-aga-preprod-exercise-v1" \
  AVIA_PREPROD_HTTPS_PORT="$https_port" \
    docker compose --project-name "$project_name" --file "$compose_file" \
      --profile local-preprod-loader "$@"
}

compose ps --format table
curl --fail --silent --show-error --insecure \
  --output /dev/null "$web_origin/health/ready" ||
  fail "canonical API readiness is not responding at $web_origin"
printf '%s\n' "$metadata"
printf 'canonical local preprod verified locally: gateway/API readiness is healthy; donor runtime disabled; external preprod not run.\n'
