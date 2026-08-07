#!/usr/bin/env bash
set -euo pipefail

# Starts the disposable, API-backed AGA demo for local interactive use. The
# connected qualification harness remains the authority for target creation,
# workspace loading, and API startup; this wrapper only issues short-lived
# local operator documents for the exact target it just created and serves the
# HTTP artifact after the API health gate passes.

umask 077

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
state_root="${AVIA_AGA_DEMO_STATE_DIR:-$repository_root/apps/web/.local/aga-demo}"
private_root="$state_root/private"
ledger_directory="$state_root/ledger"
prepare_authorization="$state_root/prepare-authorization.json"
qualification_authorization="$state_root/qualification-authorization.json"
overlay_load_authorization="$state_root/overlay-load-authorization.json"
metadata_file="$state_root/state.json"
api_port="${AVIA_PREPROD_AGA_API_PORT:-58081}"
oidc_port="${AVIA_PREPROD_AGA_OIDC_PORT:-58082}"
web_port="${AVIA_PREPROD_AGA_WEB_PORT:-4174}"
oidc_host="${AVIA_PREPROD_AGA_OIDC_HOST:-127.0.0.1}"
web_origin="${AVIA_PREPROD_AGA_DEMO_WEB_ORIGIN:-http://127.0.0.1:${web_port}}"
web_origin="${web_origin%/}"
web_image="${AVIA_PREPROD_AGA_WEB_IMAGE:-node:24.16.0-alpine3.23@sha256:2bdb65ed1dab192432bc31c95f94155ca5ad7fc1392fb7eb7526ab682fa5bf14}"
web_container_name="${AVIA_PREPROD_AGA_WEB_CONTAINER_NAME:-aviasurveil360-local-preprod-aga-web}"
node_command="${DEMO_NODE:-node}"
node_path=""
state_directory=""
web_container_id=""
cleanup_on_failure=true

package_file="$repository_root/deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip"
classification_directory="$repository_root/deliverables/aga-question-classification-candidate-2026-08-03"
provider_catalog_file="$repository_root/docs/regulatory-sources/catalogs/service-provider-catalog.v1.json"

fail() {
  printf 'aga-demo-start: %s\n' "$*" >&2
  exit 1
}

cleanup() {
  local result=$?
  trap - EXIT HUP INT TERM
  if [[ "$cleanup_on_failure" == true && "$result" -ne 0 ]]; then
    if [[ -n "$web_container_id" ]]; then
      docker rm --force "$web_container_name" >/dev/null 2>&1 || true
    fi
    if [[ -n "$state_directory" && -d "$state_directory" ]]; then
      AVIA_PREPROD_STATE_DIR="$state_directory" \
	      AVIA_PREPROD_AGA_OIDC_HOST="$oidc_host" \
	      AVIA_PREPROD_AGA_OIDC_PORT="$oidc_port" \
	      AVIA_PREPROD_AGA_API_PORT="$api_port" \
	      AVIA_PREPROD_AGA_DEMO_WEB_ORIGIN="$web_origin" \
	        docker compose --project-name aviasurveil360-local-preprod \
          --file "$repository_root/deploy/local/compose.yaml" \
          --profile local-preprod-loader \
          --profile aga-candidate-demo \
          --profile aga-candidate-demo-oidc-fixture \
          --profile aga-demo-workspace-loader \
          --profile preproddemo down --volumes --remove-orphans >/dev/null 2>&1 || true
    fi
    rm -rf -- "$state_root"
  fi
  exit "$result"
}
trap cleanup EXIT HUP INT TERM

export AVIA_PREPROD_AGA_DEMO_WEB_ORIGIN="$web_origin"

resolve_node() {
  if [[ "$node_command" = /* ]]; then
    [[ -x "$node_command" ]] || fail "DEMO_NODE is not executable: $node_command"
    node_path="$node_command"
  else
    node_path="$(command -v "$node_command" 2>/dev/null || true)"
    [[ -n "$node_path" ]] || fail "Node.js is required; set DEMO_NODE to its absolute path"
  fi
  export PATH="$(dirname "$node_path"):$PATH"
}

private_directory() {
  local path="$1"
  [[ -d "$path" && ! -L "$path" ]] || fail "private directory is missing: $path"
  [[ "$(stat -f '%Lp' "$path")" == 700 ]] || fail "private directory must be mode 0700: $path"
}

private_file() {
  local path="$1"
  [[ -f "$path" && ! -L "$path" ]] || fail "private file is missing: $path"
  [[ "$(stat -f '%Lp' "$path")" == 600 ]] || fail "private file must be mode 0600: $path"
}

compose_command() {
  [[ -n "$state_directory" ]] || fail "preprod state directory is not known"
  AVIA_PREPROD_STATE_DIR="$state_directory" \
	  AVIA_PREPROD_AGA_OIDC_HOST="$oidc_host" \
	  AVIA_PREPROD_AGA_OIDC_PORT="$oidc_port" \
	  AVIA_PREPROD_AGA_API_PORT="$api_port" \
	  AVIA_PREPROD_AGA_DEMO_WEB_ORIGIN="$web_origin" \
	    docker compose --project-name aviasurveil360-local-preprod \
      --file "$repository_root/deploy/local/compose.yaml" \
      --profile local-preprod-loader \
      --profile aga-candidate-demo \
      --profile aga-candidate-demo-oidc-fixture \
      --profile aga-demo-workspace-loader \
      --profile preproddemo "$@"
}

hash_file() {
  shasum -a 256 "$1" | awk '{print "sha256:" $1}'
}

hash_contract_inputs() {
  "$node_path" --input-type=module - \
    "$repository_root/apps/api/internal/preproddata/agacandidatedemo/contract.go" \
    "$repository_root/api/openapi/source/paths/platform.json" \
    "$repository_root/api/openapi/source/schemas/platform.json" <<'NODE'
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
const digest = createHash("sha256");
for (const path of process.argv.slice(2)) digest.update(readFileSync(path));
process.stdout.write(`sha256:${digest.digest("hex")}\n`);
NODE
}

issue_prepare_authorization() {
  local input_digest code_digest contract_digest operations
  input_digest="$(hash_file "$package_file")"
  code_digest="$(hash_file "$repository_root/apps/api/internal/preproddata/agacandidatedemo/contract.go")"
  contract_digest="$(hash_contract_inputs)"
  operations="CREATE_BASE,QUALIFY_EXISTING_SYNTHETIC_OIDC,PIN_PRE_WORKSPACE_FORBIDDEN_BASELINE,PROVISION_EMPTY_WORKSPACE_CONTRACT,EXPORT_WORKSPACE_FIXTURE,PREPARE_LOAD_INTENTS,CLEANUP_ON_PREPARE_FAILURE"
  "$node_path" "$repository_root/scripts/issue-aga-hybrid-connected-authorization.mjs" \
    --output "$prepare_authorization" \
    --issuer local-make-aga-demo \
    --phase prepare \
    --input-digest "$input_digest" \
    --code-digest "$code_digest" \
    --contract-digest "$contract_digest" \
    --operations "$operations" >"$state_root/prepare-authorization.log"
  chmod 600 "$state_root/prepare-authorization.log"
  private_file "$prepare_authorization"
}

run_prepare() {
  local result
  set +e
  AVIA_AGA_HYBRID_PRIVATE_ROOT="$private_root" \
  AVIA_AGA_HYBRID_PREPARE_AUTHORIZATION_FILE="$prepare_authorization" \
  AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR="$ledger_directory" \
  AVIA_AGA_DEMO_PACKAGE_FILE="$package_file" \
  AVIA_AGA_CLASSIFICATION_CANDIDATE_DIR="$classification_directory" \
  AVIA_AGA_PROVIDER_CATALOG_FILE="$provider_catalog_file" \
  AVIA_PREPROD_AGA_OIDC_HOST="$oidc_host" \
  AVIA_PREPROD_AGA_OIDC_PORT="$oidc_port" \
  AVIA_PREPROD_AGA_API_PORT="$api_port" \
  AVIA_PREPROD_AGA_DEMO_WEB_ORIGIN="$web_origin" \
    bash "$repository_root/scripts/test-aga-hybrid-demo-workspace-connected.sh" prepare \
      >"$state_root/prepare.log" 2>&1
  result=$?
  set -e
  [[ "$result" -eq 2 ]] || {
    sed -n '1,80p' "$state_root/prepare.log" >&2 || true
    fail "connected target preparation failed (exit $result); see $state_root/prepare.log"
  }
  grep -Fq 'pending external authority: target-bound qualification bundle is required' "$state_root/prepare.log" || \
    fail "connected preparation did not reach its target-bound handoff"
}

read_handoff_facts() {
  local handoff="$private_root/base-handoff/handoff.json"
  private_file "$handoff"
  state_directory="$("$node_path" -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).stateDirectory' "$handoff")"
  local runtime_root run_id
  runtime_root="$("$node_path" -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).runtimeRoot' "$handoff")"
  run_id="$("$node_path" -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).runId' "$handoff")"
  [[ "$state_directory" = /* && "$runtime_root" = /* && "$run_id" =~ ^[a-z0-9][a-z0-9-]{5,95}$ ]] || fail "prepared handoff is invalid"
  private_directory "$state_directory"
  printf '%s\n' "$state_directory" >"$state_root/state-directory.txt"
  chmod 600 "$state_root/state-directory.txt"
  printf '%s\n' "$runtime_root" >"$state_root/runtime-root.txt"
  chmod 600 "$state_root/runtime-root.txt"
}

write_demo_config() {
  local output="$1" handoff="$private_root/base-handoff/handoff.json" runtime_config
  runtime_config="$("$node_path" -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).runtimeRoot + "/connected-config.json"' "$handoff")"
  "$node_path" --input-type=module - "$output" "$handoff" "$runtime_config" <<'NODE'
import { closeSync, fsyncSync, openSync, readFileSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";
const [output, handoffPath, predecessorConfigPath] = process.argv.slice(2);
const handoff = JSON.parse(readFileSync(handoffPath, "utf8"));
const predecessor = JSON.parse(readFileSync(predecessorConfigPath, "utf8"));
const value = {
  environment: "local-preprod",
  runId: `${handoff.runId}-aga-demo`,
  createdAt: new Date().toISOString(),
  packageFile: "/run/input/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip",
  controlStoreDirectory: "/var/lib/aviasurveil360-preprod-control",
  authorizationFile: "/run/secrets/aga_demo_authorization",
  baseEvidenceFile: "/run/evidence/base-result.json",
  writerPasswordFile: "/run/secrets/preprod_aga_demo_writer_database_password",
  codeDigest: predecessor.codeDigest,
  contractDigest: predecessor.contractDigest,
  target: { ...handoff.databaseTarget, overlaySchema: "preprod_aga_demo" },
};
const descriptor = openSync(output, "wx", 0o600);
try { writeFileSync(descriptor, `${JSON.stringify(value, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
const parent = openSync(dirname(output), "r");
try { fsyncSync(parent); } finally { closeSync(parent); }
NODE
  private_file "$output"
}

prepare_overlay_intent() {
  local config="$private_root/aga-demo-config.json"
  local control_store="$private_root/aga-control-store"
  write_demo_config "$config"
  AVIA_PREPROD_STATE_DIR="$state_directory" \
  AVIA_PREPROD_AGA_OIDC_HOST="$oidc_host" \
  AVIA_PREPROD_AGA_OIDC_PORT="$oidc_port" \
  AVIA_PREPROD_AGA_API_PORT="$api_port" \
  AVIA_PREPROD_AGA_DEMO_WEB_ORIGIN="$web_origin" \
  AVIA_AGA_DEMO_CONFIG_FILE="$config" \
  AVIA_AGA_DEMO_PACKAGE_FILE="$package_file" \
  AVIA_AGA_DEMO_BASE_EVIDENCE_FILE="$private_root/base-handoff/base-result.json" \
  AVIA_AGA_DEMO_CONTROL_STORE_DIR="$control_store" \
    bash "$repository_root/scripts/load-aga-candidate-demo.sh" prepare-aga-demo \
      >"$state_root/overlay-prepare.log" 2>&1
  chmod 600 "$state_root/overlay-prepare.log"
  private_directory "$control_store"
  local intent_path
  intent_path="$("$node_path" --input-type=module - "$control_store/aga-demo/intents" <<'NODE'
import { readdirSync, readFileSync } from "node:fs";
const directory = process.argv[2];
const entries = readdirSync(directory).filter((name) => name.endsWith(".json"));
const matches = entries.filter((name) => {
  const value = JSON.parse(readFileSync(`${directory}/${name}`, "utf8"));
  return value.operation === "LOAD_AGA_CANDIDATE_DEMO_OVERLAY";
});
if (matches.length !== 1) throw new Error("expected one overlay intent");
process.stdout.write(`${directory}/${matches[0]}\n`);
NODE
  )"
  private_file "$intent_path"
  printf '%s\n' "$intent_path" >"$state_root/overlay-intent-path.txt"
  chmod 600 "$state_root/overlay-intent-path.txt"
  "$node_path" "$repository_root/scripts/issue-preprod-aga-demo-operation-authorization.mjs" \
    --intent "$intent_path" \
    --output "$overlay_load_authorization" \
    --operation LOAD_AGA_CANDIDATE_DEMO_OVERLAY \
    --issuer local-make-aga-demo \
    --expires-seconds 900 >"$state_root/overlay-load-authorization.log"
  chmod 600 "$state_root/overlay-load-authorization.log"
  private_file "$overlay_load_authorization"
}

issue_qualification_authorization() {
  local intent="$private_root/qualification-intent.json"
  local input_digest code_digest contract_digest target_digest intent_digest journal_digest bundle_digest operations
  private_file "$intent"
  input_digest="$(hash_file "$package_file")"
  code_digest="$(hash_file "$repository_root/apps/api/internal/preproddata/agacandidatedemo/contract.go")"
  contract_digest="$(hash_contract_inputs)"
  target_digest="$("$node_path" -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).targetFingerprintDigest' "$intent")"
  intent_digest="$("$node_path" -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).intentDigest' "$intent")"
  journal_digest="$("$node_path" -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).journalDigest' "$intent")"
  bundle_digest="$("$node_path" -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).bundleDigest' "$intent")"
  operations="LOAD_AGA_CANDIDATE_DEMO_OVERLAY,RUN_WORKSPACE_LOAD_SEAL_BARRIERS,LOAD_AND_SEAL_AGA_DEMO_WORKSPACE,RUN_CONNECTED_AGA_HYBRID_QUALIFICATION,CLEANUP_AGA_CANDIDATE_DEMO_OVERLAY,CLEANUP_WHOLE_DISPOSABLE_NAMESPACE"
  "$node_path" "$repository_root/scripts/issue-aga-hybrid-connected-authorization.mjs" \
    --output "$qualification_authorization" \
    --issuer local-make-aga-demo \
    --phase qualification \
    --target-mode EXACT_TARGET \
    --target-fingerprint "$target_digest" \
    --intent-digest "$intent_digest" \
    --journal-digest "$journal_digest" \
    --bundle-digest "$bundle_digest" \
    --input-digest "$input_digest" \
    --code-digest "$code_digest" \
    --contract-digest "$contract_digest" \
    --operations "$operations" >"$state_root/qualification-authorization.log"
  chmod 600 "$state_root/qualification-authorization.log"
  private_file "$qualification_authorization"
}

run_qualification() {
  local config="$private_root/aga-demo-config.json"
  local control_store="$private_root/aga-control-store"
  AVIA_AGA_HYBRID_PRIVATE_ROOT="$private_root" \
  AVIA_AGA_HYBRID_QUALIFICATION_AUTHORIZATION_BUNDLE_FILE="$qualification_authorization" \
  AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR="$ledger_directory" \
  AVIA_AGA_DEMO_PACKAGE_FILE="$package_file" \
  AVIA_AGA_CLASSIFICATION_CANDIDATE_DIR="$classification_directory" \
  AVIA_AGA_PROVIDER_CATALOG_FILE="$provider_catalog_file" \
  AVIA_AGA_DEMO_CONFIG_FILE="$config" \
  AVIA_AGA_DEMO_CONTROL_STORE_DIR="$control_store" \
  AVIA_AGA_DEMO_BASE_EVIDENCE_FILE="$private_root/base-handoff/base-result.json" \
  AVIA_AGA_DEMO_LOAD_AUTHORIZATION_FILE="$overlay_load_authorization" \
  AVIA_AGA_DEMO_PREPARED_INTENT=1 \
  AVIA_PREPROD_STATE_DIR="$state_directory" \
  AVIA_PREPROD_AGA_OIDC_HOST="$oidc_host" \
  AVIA_PREPROD_AGA_OIDC_PORT="$oidc_port" \
  AVIA_PREPROD_AGA_API_PORT="$api_port" \
  AVIA_PREPROD_AGA_DEMO_WEB_ORIGIN="$web_origin" \
    bash "$repository_root/scripts/test-aga-hybrid-demo-workspace-connected.sh" serve \
      >"$state_root/qualification.log" 2>&1
  chmod 600 "$state_root/qualification.log"
}

verify_question_count() {
  local raw count
  raw="$(compose_command exec --no-TTY preprod-postgres psql \
    --username aviasurveil360_preprod_loader \
    --dbname aviasurveil360_local_preprod \
    --tuples-only --no-align --command \
    'SELECT count(*) FROM preprod_aga_demo_workspace.classification_items' | tr -d '[:space:]')"
  count="$raw"
  [[ "$count" == 1310 ]] || fail "API workspace question count is $count, expected 1310"
  printf '%s\n' "$count" >"$state_root/question-count.txt"
  chmod 600 "$state_root/question-count.txt"
}

start_web() {
  local build_log="$state_root/web-build.log"
  local web_log="$state_root/web.log"
  (
    cd "$repository_root/apps/web"
    AVIA_BUILD_PROFILE=http AVIA_HTTP_API_TARGET="http://127.0.0.1:${api_port}" \
      "$node_path" node_modules/typescript/bin/tsc -b
    VITE_AVIA_DISABLE_BROWSER_TELEMETRY=1 AVIA_BUILD_PROFILE=http AVIA_HTTP_TEST_PROFILE= AVIA_HTTP_API_TARGET="http://127.0.0.1:${api_port}" \
      "$node_path" node_modules/vite/bin/vite.js build
  ) >"$build_log" 2>&1 || {
    sed -n '1,160p' "$build_log" >&2 || true
    fail "HTTP artifact build failed; see $build_log"
  }
  if docker container inspect "$web_container_name" >/dev/null 2>&1; then
    fail "web container name is already in use: $web_container_name"
  fi
  web_container_id="$(docker run --detach \
    --name "$web_container_name" \
    --label com.aviasurveil360.component=aga-demo-web \
    --label com.aviasurveil360.repository="$repository_root" \
    --publish "127.0.0.1:${web_port}:${web_port}" \
    --add-host host.docker.internal:host-gateway \
    --env AVIA_BUILD_PROFILE=http \
    --env AVIA_HTTP_API_TARGET="http://host.docker.internal:${api_port}" \
    --env AVIA_AGA_WEB_ROOT=/app/dist/http \
    --env AVIA_AGA_WEB_PORT="$web_port" \
    --volume "$repository_root/apps/web:/app:ro" \
    --workdir /app \
    "$web_image" \
    scripts/serve-aga-http.mjs)"
  printf '%s\n' "$web_container_id" >"$state_root/web.container-id"
  chmod 600 "$state_root/web.container-id"
  for _ in $(seq 1 160); do
    if curl --fail --silent --output /dev/null "http://127.0.0.1:${web_port}/"; then return 0; fi
    if [[ "$(docker inspect -f '{{.State.Running}}' "$web_container_name" 2>/dev/null || true)" != true ]]; then
      docker logs --tail 120 "$web_container_name" >"$web_log" 2>&1 || true
      sed -n '1,120p' "$web_log" >&2 || true
      fail "HTTP demo server exited during startup; see $web_log"
    fi
    sleep 0.25
  done
  docker logs --tail 120 "$web_container_name" >"$web_log" 2>&1 || true
  fail "HTTP demo server did not become ready; see $web_log"
}

write_metadata() {
  local manager_username admin_username auditee_username password
  manager_username="$("$node_path" --input-type=module - "$private_root/browser-accounts.json" <<'NODE'
import { readFileSync } from "node:fs";
const accounts = JSON.parse(readFileSync(process.argv[2], "utf8")).accounts;
const account = accounts.find(({ slot }) => slot === "DEPARTMENT_MANAGER");
if (!account?.username) throw new Error("manager account missing");
process.stdout.write(`${account.username}\n`);
NODE
  )"
  admin_username="$("$node_path" --input-type=module - "$private_root/browser-accounts.json" <<'NODE'
import { readFileSync } from "node:fs";
const accounts = JSON.parse(readFileSync(process.argv[2], "utf8")).accounts;
const account = accounts.find(({ slot }) => slot === "CAA_ADMIN");
if (!account?.username) throw new Error("admin account missing");
process.stdout.write(`${account.username}\n`);
NODE
  )"
  auditee_username="$("$node_path" --input-type=module - "$private_root/browser-accounts.json" <<'NODE'
import { readFileSync } from "node:fs";
const accounts = JSON.parse(readFileSync(process.argv[2], "utf8")).accounts;
const account = accounts.find(({ slot }) => slot === "AUDITEE_MATCHING");
if (!account?.username) throw new Error("matching auditee account missing");
process.stdout.write(`${account.username}\n`);
NODE
  )"
  password="$(tr -d '\r\n' <"$state_directory/secrets/preprod_aga_demo_oidc_qualification_password")"
  [[ ${#password} -ge 24 ]] || fail "OIDC qualification password is unexpectedly short"
  "$node_path" --input-type=module - "$metadata_file" "$private_root" "$state_directory" "$web_container_name" "$web_container_id" "$web_image" "$api_port" "$oidc_port" "$web_origin" "$oidc_host" "$manager_username" "$admin_username" "$auditee_username" <<'NODE'
import { closeSync, fsyncSync, openSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";
const [output, privateRoot, stateDirectory, webContainerName, webContainerId, webImage, apiPort, oidcPort, webOrigin, oidcHost, managerUsername, adminUsername, auditeeUsername] = process.argv.slice(2);
const value = {
  schemaVersion: "aga-demo-runtime/v1",
  status: "running",
  privateRoot,
  stateDirectory,
  webContainerName,
  webContainerId,
  webImage,
  apiUrl: `http://127.0.0.1:${apiPort}`,
  oidcUrl: `http://${oidcHost}:${oidcPort}/identity/realms/aviasurveil360-local-preprod/account`,
  webUrl: `${webOrigin}/department-manager/aga-demo-workspace`,
  questionCount: 1310,
  managerUsername,
  adminUsername,
  auditeeUsername,
  startedAt: new Date().toISOString(),
};
const descriptor = openSync(output, "wx", 0o600);
try { writeFileSync(descriptor, `${JSON.stringify(value, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
const parent = openSync(dirname(output), "r");
try { fsyncSync(parent); } finally { closeSync(parent); }
NODE
  private_file "$metadata_file"
  printf '\nAGA API demo hazır.\nURL: %s/\nSorular: 1,310 (API workspace)\n\nGiriş için aynı parolayı kullan:\n  Department Manager: %s\n  CAA Admin:          %s\n  Matching Auditee:   %s\n  Parola:             %s\n\nKapatmak: make aga-demo-down\nDurum:     make aga-demo-status\n' \
    "$web_origin/department-manager/aga-demo-workspace" "$manager_username" "$admin_username" "$auditee_username" "$password"
}

[[ "$state_root" = /* ]] || fail "AVIA_AGA_DEMO_STATE_DIR must be absolute"
[[ -f "$package_file" && -d "$classification_directory" && -f "$provider_catalog_file" ]] || fail "AGA demo inputs are missing"
command -v docker >/dev/null 2>&1 || fail "Docker is required"
command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v shasum >/dev/null 2>&1 || fail "shasum is required"
resolve_node

if [[ -f "$metadata_file" ]]; then
  fail "AGA API demo state already exists; run make aga-demo-status or make aga-demo-down"
fi
if curl --fail --silent --output /dev/null "http://127.0.0.1:${web_port}/"; then
  fail "HTTP port ${web_port} is already in use"
fi
if curl --fail --silent --output /dev/null "http://127.0.0.1:${api_port}/health/ready"; then
  fail "API port ${api_port} is already in use"
fi

mkdir -p "$state_root" "$private_root" "$ledger_directory"
chmod 700 "$state_root" "$private_root" "$ledger_directory"
private_directory "$state_root"
private_directory "$private_root"
private_directory "$ledger_directory"

issue_prepare_authorization
run_prepare
read_handoff_facts
prepare_overlay_intent
issue_qualification_authorization
run_qualification

for _ in $(seq 1 80); do
  if curl --fail --silent --output /dev/null "http://127.0.0.1:${api_port}/health/ready"; then break; fi
  sleep 0.25
done
curl --fail --silent --output /dev/null "http://127.0.0.1:${api_port}/health/ready" || fail "API health gate did not pass"
verify_question_count
start_web

cleanup_on_failure=false
write_metadata
trap - EXIT HUP INT TERM
