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
  printf 'canonical-preprod-status: %s\n' "$*" >&2
  exit 1
}

[[ -f "$metadata_file" ]] || fail "canonical local preprod is not initialized"
[[ "$project_name" =~ ^aviasurveil360-(local-preprod(-[a-z0-9][a-z0-9-]*)?|task-[a-z0-9][a-z0-9-]*)$ ]] ||
  fail "AVIA_CANONICAL_PREPROD_PROJECT must identify one exact AviaSurveil360 local-preprod project"
command -v docker >/dev/null 2>&1 || fail "docker is required"
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
  !["http:", "https:"].includes(origin.protocol) ||
  origin.pathname !== "/" || origin.search || origin.hash ||
  metadata.identityProvider !== "first-party" ||
  metadata.profile !== "aga-preprod@1.0.0" ||
  metadata.identityNamespace !== "canonical-aga-preprod-exercise-v1" ||
  metadata.donorRuntime !== "disabled" ||
  metadata.externalPreprod !== "not run"
) throw new Error("canonical local preprod metadata is invalid");
process.stdout.write(`${metadata.webOrigin}\t${origin.protocol.slice(0, -1)}\t${origin.host}\n`);
NODE
)" || fail "canonical local preprod metadata is invalid"
IFS=$'\t' read -r web_origin origin_scheme public_host <<<"$metadata_values"
[[ "$origin_scheme" == "http" || "$origin_scheme" == "https" ]] || fail "canonical local preprod metadata has an invalid transport"
if [[ "$origin_scheme" == "http" ]]; then
  compose_override="$repository_root/deploy/local/compose.local-http.yaml"
else
  compose_override=""
  [[ "$https_port" =~ ^[0-9]+$ && "$https_port" -ge 1024 && "$https_port" -le 65535 ]] ||
    fail "AVIA_PREPROD_HTTPS_PORT must be a user-space TCP port"
fi

metadata="$(node --input-type=module - "$metadata_file" <<'NODE'
import { readFileSync } from "node:fs";
process.stdout.write(JSON.stringify(JSON.parse(readFileSync(process.argv[2], "utf8"))));
NODE
)"

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

compose ps --format table
curl_args=(--fail --silent --show-error --output /dev/null)
[[ "$origin_scheme" == "https" ]] && curl_args+=(--insecure)
curl "${curl_args[@]}" "$web_origin/health/ready" ||
  fail "canonical API readiness is not responding at $web_origin"
printf '%s\n' "$metadata"
printf 'canonical local preprod verified locally: first-party gateway/API readiness is healthy; donor runtime disabled; external preprod not run.\n'
