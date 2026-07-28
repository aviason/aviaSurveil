#!/usr/bin/env bash
set -euo pipefail

umask 077

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_file="$repository_root/deploy/local/compose.yaml"
project_name="aviasurveil360-local-preprod"
profile_name="${1:-}"
profile_version="1.0.0"
task8_qualification="${AVIA_TASK8_PROFILE_QUALIFICATION:-false}"
qualification_started_epoch="$(date -u +%s)"
run_id="run-task7-connected-smoke"
evidence_directory=""
runtime_root=""
state_directory=""
control_store_directory=""
prepare_configuration_file=""
connected_configuration_file=""
authorization_file=""
seed_file=""
project_owned="false"
resource_sampler_pid=""
resource_sampler_marker=""

fail() {
  printf 'preprod-connected-scenarios: %s\n' "$*" >&2
  exit 1
}

compose_command() {
  docker compose \
    --project-name "$project_name" \
    --file "$compose_file" \
    --profile local-preprod-loader \
    "$@"
}

project_residue() {
  local containers
  local volumes
  local networks
  containers="$(
    docker ps --all \
      --filter "label=com.docker.compose.project=$project_name" \
      --format '{{.ID}}'
  )"
  volumes="$(
    docker volume ls \
      --filter "label=com.docker.compose.project=$project_name" \
      --format '{{.Name}}'
  )"
  networks="$(
    docker network ls \
      --filter "label=com.docker.compose.project=$project_name" \
      --format '{{.Name}}'
  )"
  if [[ -n "$containers" || -n "$volumes" || -n "$networks" ]]; then
    return 0
  fi
  return 1
}

start_resource_sampler() {
  local samples_file="$1"
  resource_sampler_marker="$runtime_root/resource-sampler.active"
  : >"$resource_sampler_marker"
  (
    while [[ -f "$resource_sampler_marker" ]]; do
      container_ids="$(
        docker ps \
          --filter "label=com.docker.compose.project=$project_name" \
          --filter "label=com.docker.compose.service=preprod-data-loader" \
          --format '{{.ID}}'
      )"
      if [[ -n "$container_ids" ]]; then
        docker stats \
          --no-stream \
          --format '{{json .}}' \
          $container_ids >>"$samples_file" 2>/dev/null || true
      fi
      sleep 0.25
    done
  ) &
  resource_sampler_pid="$!"
}

stop_resource_sampler() {
  if [[ -n "$resource_sampler_marker" ]]; then
    rm -f -- "$resource_sampler_marker"
  fi
  if [[ -n "$resource_sampler_pid" ]]; then
    wait "$resource_sampler_pid" 2>/dev/null || true
  fi
  resource_sampler_pid=""
  resource_sampler_marker=""
}

cleanup() {
  local status=$?
  trap - EXIT HUP INT TERM
  if [[ -n "$resource_sampler_marker" ]]; then
    rm -f -- "$resource_sampler_marker"
  fi
  if [[ -n "$resource_sampler_pid" ]]; then
    wait "$resource_sampler_pid" 2>/dev/null || true
  fi
  if [[ "$project_owned" == "true" ]]; then
    if [[ "$status" -ne 0 &&
      "$task8_qualification" == "true" &&
      -n "$evidence_directory" &&
      -d "$evidence_directory" &&
      ! -e "$evidence_directory/compose-failure.log" ]]; then
      (
        set -o noclobber
        compose_command logs --no-color \
          preprod-api preprod-migration \
          >"$evidence_directory/compose-failure.log" 2>&1
      ) || status=1
    fi
    if ! compose_command down --volumes --remove-orphans; then
      status=1
    fi
    if project_residue; then
      printf '%s\n' \
        "preprod-connected-scenarios: task-owned Compose residue remains" >&2
      status=1
    fi
  fi
  if [[ -n "$runtime_root" && -d "$runtime_root" ]]; then
    rm -rf -- "$runtime_root"
  fi
  if [[ "$status" -ne 0 &&
    "$task8_qualification" == "true" &&
    -n "$evidence_directory" &&
    -d "$evidence_directory" &&
    ! -e "$evidence_directory/failure.txt" ]]; then
    printf 'Task 8 profile qualification failed with exit %d.\n' \
      "$status" >"$evidence_directory/failure.txt"
  fi
  exit "$status"
}
trap cleanup EXIT HUP INT TERM

if [[ "$#" -ne 1 ]]; then
  fail "usage: $0 smoke|acceptance|realistic|stress"
fi
if [[ "$task8_qualification" == "true" ]]; then
  case "$profile_name" in
    smoke | acceptance | realistic | stress)
      ;;
    *)
      fail "Task 8 profile must be smoke|acceptance|realistic|stress"
      ;;
  esac
  run_id="${AVIA_TASK8_RUN_ID:-}"
  evidence_directory="${AVIA_TASK8_EVIDENCE_DIRECTORY:-}"
  [[ "$run_id" =~ ^run-task8-(smoke|acceptance|realistic|stress)-[a-z0-9-]+-[0-9]+$ ]] ||
    fail "AVIA_TASK8_RUN_ID is invalid"
  [[ "$evidence_directory" == \
    "$repository_root/docs/demo-evidence/preprod-data/$run_id" ]] ||
    fail "AVIA_TASK8_EVIDENCE_DIRECTORY is outside the create-only evidence root"
  [[ -d "$evidence_directory" ]] ||
    fail "Task 8 evidence directory does not exist"
  profile_version="${AVIA_TASK8_PROFILE_VERSION:-}"
  [[ "$profile_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] ||
    fail "AVIA_TASK8_PROFILE_VERSION is invalid"
elif [[ "$profile_name" != "smoke" ]]; then
  fail "usage: $0 smoke"
fi
command -v docker >/dev/null 2>&1 || fail "docker is required"
command -v node >/dev/null 2>&1 || fail "node is required"
command -v openssl >/dev/null 2>&1 || fail "openssl is required"
docker info >/dev/null 2>&1 || fail "docker daemon is unavailable"
if project_residue; then
  fail "pre-existing $project_name Compose resources must be resolved by their owner"
fi

runtime_root="$(
  mktemp -d \
    "${TMPDIR:-/tmp}/avia-preprod-connected-${profile_name}.XXXXXX"
)"
chmod 0700 "$runtime_root"
state_directory="$runtime_root/state"
control_store_directory="$state_directory/control-store"
prepare_configuration_file="$runtime_root/prepare-config.json"
connected_configuration_file="$runtime_root/connected-config.json"
authorization_file="$runtime_root/placeholder-authorization.json"
seed_file="$state_directory/secrets/preprod_loader_seed"

AVIA_PREPROD_STATE_DIR="$state_directory" \
  "$repository_root/scripts/init-local-preprod-namespace.sh"

printf '{}\n' >"$prepare_configuration_file"
printf '{}\n' >"$authorization_file"
chmod 0600 "$prepare_configuration_file" "$authorization_file" "$seed_file"

export AVIA_PREPROD_STATE_DIR="$state_directory"
export AVIA_PREPROD_CONTROL_STORE_DIR="$control_store_directory"
export AVIA_PREPROD_LOADER_CONFIG_FILE="$prepare_configuration_file"
export AVIA_PREPROD_AUTHORIZATION_FILE="$authorization_file"
export AVIA_PREPROD_SEED_FILE="$seed_file"

env GOCACHE="$runtime_root/go-cache-scenarios" \
  go -C "$repository_root/apps/api" test -count=1 \
  ./internal/preproddata/scenarios ./cmd/preprod-data-loader
node --test "$repository_root/tests/preprod-data-boundary.test.mjs"

build_services=(
  preprod-migration
  preprod-keycloak
  preprod-data-loader
)
if [[ "$task8_qualification" == "true" ]]; then
  build_services+=(preprod-api)
fi
compose_command build "${build_services[@]}"

project_owned="true"
runtime_services=(
  preprod-postgres \
  preprod-keycloak-postgres \
  preprod-mailpit \
  preprod-minio \
  preprod-keycloak
)
if [[ "$task8_qualification" == "true" ]]; then
  runtime_services+=(preprod-api)
fi
compose_command up --detach --wait --wait-timeout 240 \
  "${runtime_services[@]}"

postgres_system_identifier="$(
  compose_command exec --no-TTY preprod-postgres \
    psql \
      --username aviasurveil360_preprod_loader \
      --dbname aviasurveil360_local_preprod \
      --tuples-only \
      --no-align \
      --command 'SELECT system_identifier::text FROM pg_control_system()' |
    tr -d '[:space:]'
)"
[[ "$postgres_system_identifier" =~ ^[0-9]{10,24}$ ]] ||
  fail "invalid PostgreSQL system identifier"

code_digest="$(
  docker image inspect aviasurveil360/preprod-data-loader:local \
    --format '{{.Id}}'
)"
[[ "$code_digest" =~ ^sha256:[a-f0-9]{64}$ ]] ||
  fail "invalid loader image digest"

contract_digest="$(
  node --input-type=module - \
    "$repository_root/apps/api/internal/preproddata/profiles/profiles.go" \
    "$repository_root/apps/web/src/parity/legacy-screen-source.json" \
    "$repository_root/tests/parity/behavior-ledger.json" <<'NODE'
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";

const digest = createHash("sha256");
for (const path of process.argv.slice(2)) {
  digest.update(readFileSync(path));
}
process.stdout.write(`sha256:${digest.digest("hex")}`);
NODE
)"
[[ "$contract_digest" =~ ^sha256:[a-f0-9]{64}$ ]] ||
  fail "invalid scenario contract digest"

AVIA_TASK7_CONFIG_PATH="$prepare_configuration_file" \
AVIA_TASK7_RUN_ID="$run_id" \
AVIA_TASK7_PROFILE="$profile_name" \
AVIA_TASK7_PROFILE_VERSION="$profile_version" \
AVIA_TASK7_CODE_DIGEST="$code_digest" \
AVIA_TASK7_CONTRACT_DIGEST="$contract_digest" \
AVIA_TASK7_POSTGRES_SYSTEM_IDENTIFIER="$postgres_system_identifier" \
  node --input-type=module <<'NODE'
import { writeFileSync } from "node:fs";

const env = process.env;
const runId = env.AVIA_TASK7_RUN_ID;
const configuration = {
  environment: "local-preprod",
  profile: env.AVIA_TASK7_PROFILE,
  profileVersion: env.AVIA_TASK7_PROFILE_VERSION,
  runId,
  seedFile: "/run/secrets/preprod_seed",
  authorizationFile: "/run/secrets/preprod_loader_authorization",
  controlStoreDirectory: "/var/lib/aviasurveil360-preprod-control",
  intentFile:
    `/var/lib/aviasurveil360-preprod-control/intents/${runId}.json`,
  routeCatalogFile: "/app/catalog/legacy-screen-source.json",
  behaviorLedgerFile: "/app/catalog/behavior-ledger.json",
  codeDigest: env.AVIA_TASK7_CODE_DIGEST,
  contractDigest: env.AVIA_TASK7_CONTRACT_DIGEST,
  target: {
    environment: "local-preprod",
    databaseName: "aviasurveil360_local_preprod",
    databaseOwner: "aviasurveil360_preprod_loader",
    postgresSystemIdentifier: env.AVIA_TASK7_POSTGRES_SYSTEM_IDENTIFIER,
    postgresHost: "preprod-postgres",
    postgresPort: 5432,
    composeProject: "aviasurveil360-local-preprod",
    keycloakRealm: "aviasurveil360-local-preprod",
    keycloakDatabase: "keycloak_local_preprod",
    keycloakServiceClientId:
      "aviasurveil360-local-preprod-lifecycle",
    mailpitNamespace: "aviasurveil360-local-preprod",
    objectBucket: "aviasurveil360-local-preprod",
    objectPrefix: `runs/${runId}/`,
    loaderQueueNamespace: "aviasurveil360-local-preprod",
    profileName: env.AVIA_TASK7_PROFILE,
    profileVersion: env.AVIA_TASK7_PROFILE_VERSION,
    runId,
  },
};
writeFileSync(
  env.AVIA_TASK7_CONFIG_PATH,
  `${JSON.stringify(configuration, null, 2)}\n`,
  { mode: 0o600 },
);
NODE
chmod 0600 "$prepare_configuration_file"

"$repository_root/scripts/load-preprod-data.sh" prepare

intent_file="$control_store_directory/intents/$run_id.json"
[[ -f "$intent_file" ]] || fail "immutable intent file was not created"

AVIA_TASK7_INTENT_PATH="$intent_file" \
AVIA_TASK7_CONFIG_PATH="$connected_configuration_file" \
  node --input-type=module <<'NODE'
import { readFileSync, writeFileSync } from "node:fs";

const intent = JSON.parse(readFileSync(process.env.AVIA_TASK7_INTENT_PATH));
const configuration = {
  environment: "local-preprod",
  profile: intent.profile,
  profileVersion: intent.profileVersion,
  runId: intent.runId,
  seedFile: "/run/secrets/preprod_seed",
  authorizationFile: "/run/secrets/preprod_loader_authorization",
  controlStoreDirectory: "/var/lib/aviasurveil360-preprod-control",
  intentFile:
    `/var/lib/aviasurveil360-preprod-control/intents/${intent.runId}.json`,
  routeCatalogFile: "/app/catalog/legacy-screen-source.json",
  behaviorLedgerFile: "/app/catalog/behavior-ledger.json",
  codeDigest: intent.codeDigest,
  contractDigest: intent.contractDigest,
  target: intent.target,
};
writeFileSync(
  process.env.AVIA_TASK7_CONFIG_PATH,
  `${JSON.stringify(configuration, null, 2)}\n`,
  { mode: 0o600 },
);
NODE
chmod 0600 "$connected_configuration_file"
export AVIA_PREPROD_LOADER_CONFIG_FILE="$connected_configuration_file"

write_authorization() {
  local operation="$1"
  local output_path="$2"
  local issuer="$3"
  local token
  local nonce
  token="$(openssl rand -hex 32)"
  nonce="$(openssl rand -hex 16)"
  AVIA_TASK7_INTENT_PATH="$intent_file" \
  AVIA_TASK7_AUTHORIZATION_PATH="$output_path" \
  AVIA_TASK7_AUTHORIZATION_OPERATION="$operation" \
  AVIA_TASK7_AUTHORIZATION_ISSUER="$issuer" \
  AVIA_TASK7_AUTHORIZATION_TOKEN="$token" \
  AVIA_TASK7_AUTHORIZATION_NONCE="$nonce" \
    node --input-type=module <<'NODE'
import { readFileSync, writeFileSync } from "node:fs";

const env = process.env;
const intent = JSON.parse(readFileSync(env.AVIA_TASK7_INTENT_PATH));
const authorization = {
  schemaVersion: "preprod-operation-authorization/v1",
  token: env.AVIA_TASK7_AUTHORIZATION_TOKEN,
  operation: env.AVIA_TASK7_AUTHORIZATION_OPERATION,
  issuer: env.AVIA_TASK7_AUTHORIZATION_ISSUER,
  expiresAt: new Date(Date.now() + 10 * 60 * 1000).toISOString(),
  nonce: env.AVIA_TASK7_AUTHORIZATION_NONCE,
  runId: intent.runId,
  intentDigest: intent.intentDigest,
  targetFingerprintDigest: intent.targetFingerprintDigest,
};
writeFileSync(
  env.AVIA_TASK7_AUTHORIZATION_PATH,
  `${JSON.stringify(authorization)}\n`,
  { mode: 0o600 },
);
NODE
  chmod 0600 "$output_path"
  unset token nonce
}

authorization_file="$runtime_root/load-authorization.json"
authorization_issuer="plan-5-task-7-connected-smoke"
if [[ "$task8_qualification" == "true" ]]; then
  authorization_issuer="plan-5-task-8-${profile_name}"
fi
write_authorization \
  "LOAD_EMPTY_TARGET" \
  "$authorization_file" \
  "$authorization_issuer"
export AVIA_PREPROD_AUTHORIZATION_FILE="$authorization_file"

"$repository_root/scripts/load-preprod-data.sh" verify-authorization
generation_started_epoch="$(date -u +%s)"
resource_samples_file="$runtime_root/loader-resource-samples.ndjson"
if [[ "$task8_qualification" == "true" ]]; then
  start_resource_sampler "$resource_samples_file"
  interrupt_commands="$(
    AVIA_TASK8_INTENT_PATH="$intent_file" node --input-type=module <<'NODE'
import { readFileSync } from "node:fs";
const intent = JSON.parse(readFileSync(process.env.AVIA_TASK8_INTENT_PATH));
const batchSize = 128;
const commands =
  Math.ceil(intent.expectedCounts.organizations / batchSize) +
  Math.ceil(intent.expectedCounts.providerAccounts / batchSize);
if (!Number.isSafeInteger(commands) || commands < 2) {
  throw new Error("invalid qualification interruption boundary");
}
process.stdout.write(String(commands));
NODE
  )"
  export AVIA_PREPROD_QUALIFICATION_INTERRUPT_AFTER_COMMANDS="$interrupt_commands"
  set +e
  "$repository_root/scripts/load-preprod-data.sh" run-connected
  interrupted_status="$?"
  set -e
  unset AVIA_PREPROD_QUALIFICATION_INTERRUPT_AFTER_COMMANDS
  [[ "$interrupted_status" -ne 0 ]] ||
    fail "qualification interruption unexpectedly completed the run"

  AVIA_TASK8_CONTROL_STORE="$control_store_directory" \
  AVIA_TASK8_RUN_ID="$run_id" \
    node --input-type=module <<'NODE'
import { readdirSync, readFileSync } from "node:fs";
import path from "node:path";

const directory = path.join(
  process.env.AVIA_TASK8_CONTROL_STORE,
  "results",
  process.env.AVIA_TASK8_RUN_ID,
);
const results = readdirSync(directory)
  .filter((name) => name.endsWith(".json"))
  .map((name) => JSON.parse(readFileSync(path.join(directory, name))));
if (
  results.length !== 1 ||
  results[0].outcome !== "FAILED" ||
  !results[0].failures.includes("COMMAND_STREAM_FAILED")
) {
  throw new Error("controlled interruption evidence is not exact");
}
NODE

  resume_authorization_file="$runtime_root/resume-authorization.json"
  write_authorization \
    "RESUME_RUN" \
    "$resume_authorization_file" \
    "plan-5-task-8-${profile_name}-resume"
  authorization_file="$resume_authorization_file"
  export AVIA_PREPROD_AUTHORIZATION_FILE="$authorization_file"
  "$repository_root/scripts/load-preprod-data.sh" verify-authorization
  "$repository_root/scripts/load-preprod-data.sh" run-connected
  stop_resource_sampler
else
  "$repository_root/scripts/load-preprod-data.sh" run-connected
fi
generation_completed_epoch="$(date -u +%s)"
generation_seconds="$((generation_completed_epoch - generation_started_epoch))"

successful_result_binding="$control_store_directory/runs/$run_id.success"
[[ -f "$successful_result_binding" ]] ||
  fail "successful-result binding was not created"

AVIA_TASK7_INTENT_PATH="$intent_file" \
AVIA_TASK7_CONTROL_STORE="$control_store_directory" \
AVIA_TASK7_RUN_ID="$run_id" \
  node --input-type=module <<'NODE'
import { readFileSync } from "node:fs";
import path from "node:path";

const env = process.env;
const intent = JSON.parse(readFileSync(env.AVIA_TASK7_INTENT_PATH));
const resultDigest = readFileSync(
  path.join(env.AVIA_TASK7_CONTROL_STORE, "runs", `${env.AVIA_TASK7_RUN_ID}.success`),
  "utf8",
).trim();
const result = JSON.parse(readFileSync(path.join(
  env.AVIA_TASK7_CONTROL_STORE,
  "results",
  env.AVIA_TASK7_RUN_ID,
  `${resultDigest.replace(/^sha256:/u, "")}.json`,
)));
if (
  result.outcome !== "SUCCEEDED" ||
  result.runId !== intent.runId ||
  result.intentDigest !== intent.intentDigest ||
  result.resultDigest !== resultDigest
) {
  throw new Error("successful result identity differs from immutable intent");
}
const expectedFamilies = Object.keys(intent.expectedCounts).sort();
const actualFamilies = Object.keys(result.actualCounts).sort();
const digestFamilies = Object.keys(result.relationshipDigests).sort();
if (
  expectedFamilies.length !== 40 ||
  JSON.stringify(actualFamilies) !== JSON.stringify(expectedFamilies) ||
  JSON.stringify(digestFamilies) !== JSON.stringify(expectedFamilies)
) {
  throw new Error("result family catalogs differ from the frozen manifest");
}
for (const family of expectedFamilies) {
  if (
    result.actualCounts[family] !== intent.expectedCounts[family] ||
    !/^sha256:[a-f0-9]{64}$/u.test(result.relationshipDigests[family])
  ) {
    throw new Error(`result reconciliation differs for ${family}`);
  }
}
if (!Array.isArray(result.checkpoints) || result.checkpoints.length < 3) {
  throw new Error("successful result omits durable checkpoint evidence");
}
NODE

domain_count_sql="$(
  AVIA_TASK8_INTENT_PATH="$intent_file" node --input-type=module <<'NODE'
import { readFileSync } from "node:fs";

const intent = JSON.parse(readFileSync(process.env.AVIA_TASK8_INTENT_PATH));
const mappings = {
  organizations: ["organizations", "organizations"],
  providerAccounts: ["identity_references", "identity_references"],
  desiredMembershipVersions: [
    "desired_membership_versions",
    "desired_membership_versions",
  ],
  applicationProfiles: ["user_profiles", "user_profiles"],
  sessions: ["session_references", "session_references"],
  surveillancePlans: ["surveillance_plan_items", "surveillance_plan_items"],
  audits: ["inspections", "inspections"],
  assignments: ["audit_question_assignments", "audit_question_assignments"],
  checklistTemplates: ["template_masters", "template_masters"],
  checklistTemplateVersions: [
    "checklist_template_versions",
    "checklist_template_versions",
  ],
  checklistQuestions: ["question_versions", "question_versions"],
  inspectionPackages: ["inspection_packages", "inspection_packages"],
  checklistResponses: ["checklist_responses", "checklist_responses"],
  potentialFindings: ["potential_findings", "potential_findings"],
  findings: ["findings", "findings"],
  capRevisions: ["cap_revisions", "cap_revisions"],
  objectVersions: ["object_metadata", "object_metadata"],
  evidenceVersions: ["evidence_versions", "evidence_versions"],
  reviewDecisions: ["review_decisions", "review_decisions"],
  reportVersions: ["report_versions", "report_versions"],
  communications: ["communication_messages", "communication_messages"],
  notifications: ["notification_records", "notification_records"],
  auditEvents: ["audit_events", "audit_events"],
  outboxMessages: ["outbox_messages", "outbox_messages"],
  deliveryJobs: [
    "notification_delivery_jobs",
    "notification_delivery_jobs",
  ],
  renderJobs: ["document_render_jobs", "document_render_jobs"],
  calendarRecords: ["reminder_dispatches", "reminder_dispatches"],
  offlineGrants: ["offline_grants", "offline_grants"],
  syncChanges: ["authorized_sync_changes", "authorized_sync_changes"],
};
const values = Object.entries(mappings).map(([family, [name, table]]) => {
  const expected = intent.expectedCounts[family];
  if (!Number.isSafeInteger(expected) || expected < 0) {
    throw new Error(`missing exact count for ${family}`);
  }
  return `('${name}', (SELECT count(*) FROM ${table}), ${expected})`;
});
process.stdout.write(`
WITH actual(name, actual_count, expected_count) AS (
  VALUES ${values.join(",\n")}
)
SELECT CASE
  WHEN count(*) FILTER (WHERE actual_count <> expected_count) = 0
    THEN 'domain-counts-ok'
  ELSE string_agg(
    name || '=' || actual_count || '/' || expected_count,
    ',' ORDER BY name
  ) FILTER (WHERE actual_count <> expected_count)
END
FROM actual;
`);
NODE
)"
domain_count_result="$(
  compose_command exec --no-TTY preprod-postgres \
    psql \
      --username aviasurveil360_preprod_loader \
      --dbname aviasurveil360_local_preprod \
      --tuples-only \
      --no-align \
      --set ON_ERROR_STOP=1 \
      --command "$domain_count_sql" |
    tr -d '\r'
)"
[[ "$domain_count_result" == "domain-counts-ok" ]] ||
  fail "domain materialization mismatch: $domain_count_result"

privacy_result="$(
  compose_command exec --no-TTY preprod-postgres \
    psql \
      --username aviasurveil360_preprod_loader \
      --dbname aviasurveil360_local_preprod \
      --tuples-only \
      --no-align \
      --set ON_ERROR_STOP=1 \
      --command "
        WITH matrix AS (
          SELECT jsonb_array_elements(attributes -> 'privacyMatrix') AS item
          FROM preprod_loader.scenario_records
          WHERE run_id = '$run_id'
            AND family = 'routeDispositions'
            AND attributes ? 'privacyMatrix'
        ),
        checked AS (
          SELECT matrix.item, records.record_id, records.organization_id
          FROM matrix
          LEFT JOIN preprod_loader.scenario_records AS records
            ON records.run_id = '$run_id'
           AND records.family = 'routeDispositions'
           AND records.record_id = matrix.item ->> 'recordCanary'
        )
        SELECT
          count(*) || '|' ||
          count(DISTINCT item ->> 'surface') || '|' ||
          count(DISTINCT item ->> 'canaryClass') || '|' ||
          count(*) FILTER (
            WHERE record_id IS NOT NULL
              AND organization_id = item ->> 'sourceOrganizationId'
              AND item ->> 'sourceOrganizationId' <>
                item ->> 'actorOrganizationId'
              AND item ->> 'expectedResult' = 'denied-no-exposure'
          )
        FROM checked;
      " |
    tr -d '[:space:]'
)"
[[ "$privacy_result" == "45|15|3|45" ]] ||
  fail "privacy canary reconciliation mismatch: $privacy_result"

if [[ "$task8_qualification" == "true" ]]; then
  runtime_metrics="$(
    compose_command exec --no-TTY preprod-postgres \
      psql \
        --username aviasurveil360_preprod_loader \
        --dbname aviasurveil360_local_preprod \
        --tuples-only \
        --no-align \
        --set ON_ERROR_STOP=1 \
        --field-separator '|' \
        --command "
          SELECT
            pg_database_size(current_database()),
            pg_total_relation_size('preprod_loader.scenario_records'),
            pg_total_relation_size('object_metadata'),
            COALESCE((SELECT sum(size_bytes) FROM object_metadata), 0),
            COALESCE((
              SELECT max(extract(epoch FROM (
                COALESCE(
                  accepted_at,
                  next_attempt_at,
                  terminal_at,
                  updated_at
                ) - created_at
              )))
              FROM notification_delivery_jobs
            ), 0),
            (
              SELECT count(*)
              FROM identity_references
              WHERE lower(email) !~ '^[a-z0-9._+-]+@synthetic[.]invalid$'
            ) + (
              SELECT count(*)
              FROM preprod_loader.scenario_records
              WHERE btrim(decision_reason) !~ '^SYNTHETIC '
                 OR attributes::text ~* '\"(client_secret|access_token|refresh_token|totp_secret|private_key|api_key)\"[[:space:]]*:'
            );
        " |
      tr -d '[:space:]'
  )"
  IFS='|' read -r \
    database_bytes \
    scenario_relation_bytes \
    object_metadata_relation_bytes \
    object_bytes \
    worker_lag_seconds \
    privacy_findings <<<"$runtime_metrics"
  for numeric_value in \
    "$database_bytes" \
    "$scenario_relation_bytes" \
    "$object_metadata_relation_bytes" \
    "$object_bytes" \
    "$worker_lag_seconds" \
    "$privacy_findings"; do
    [[ "$numeric_value" =~ ^[0-9]+([.][0-9]+)?$ ]] ||
      fail "runtime metric is not numeric"
  done
  [[ "$privacy_findings" == "0" ]] ||
    fail "generated namespace privacy scan found $privacy_findings findings"

  postgres_disk_kib="$(
    compose_command exec --no-TTY preprod-postgres \
      sh -c "du -sk /var/lib/postgresql/data | cut -f1" |
      tr -d '[:space:]'
  )"
  keycloak_disk_kib="$(
    compose_command exec --no-TTY preprod-keycloak-postgres \
      sh -c "du -sk /var/lib/postgresql/data | cut -f1" |
      tr -d '[:space:]'
  )"
  object_disk_kib="$(
    compose_command exec --no-TTY preprod-minio \
      sh -c "du -sk /data | cut -f1" |
      tr -d '[:space:]'
  )"
  mailpit_disk_kib="$(
    compose_command exec --no-TTY preprod-mailpit \
      sh -c "du -sk /data | cut -f1" |
      tr -d '[:space:]'
  )"
  for disk_value in \
    "$postgres_disk_kib" \
    "$keycloak_disk_kib" \
    "$object_disk_kib" \
    "$mailpit_disk_kib"; do
    [[ "$disk_value" =~ ^[0-9]+$ ]] ||
      fail "namespace disk measurement is invalid"
  done
  namespace_disk_mib="$(((
    postgres_disk_kib +
    keycloak_disk_kib +
    object_disk_kib +
    mailpit_disk_kib +
    1023
  ) / 1024))"

  compose_command exec --no-TTY preprod-postgres \
    psql \
      --username aviasurveil360_preprod_loader \
      --dbname aviasurveil360_local_preprod \
      --tuples-only \
      --no-align \
      --set ON_ERROR_STOP=1 \
      --command "
        EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON)
        SELECT id, reference, organization_id, status, updated_at
        FROM findings
        ORDER BY updated_at DESC, id
        LIMIT 25;
      " >"$evidence_directory/query-plan.json"

  compose_command exec --no-TTY preprod-postgres \
    psql \
      --username aviasurveil360_preprod_loader \
      --dbname aviasurveil360_local_preprod \
      --tuples-only \
      --no-align \
      --set ON_ERROR_STOP=1 \
      --command "
        COPY (
          SELECT relname, seq_scan, idx_scan, n_live_tup,
                 pg_total_relation_size(relid) AS total_bytes
          FROM pg_stat_user_tables
          ORDER BY pg_total_relation_size(relid) DESC, relname
          LIMIT 20
        ) TO STDOUT WITH (FORMAT csv, HEADER true, DELIMITER E'\t');
      " >"$evidence_directory/top-tables.tsv"

  AVIA_TASK8_COMPOSE_FILE="$compose_file" \
  AVIA_TASK8_PROJECT="$project_name" \
  AVIA_TASK8_API_LATENCY_PATH="$evidence_directory/api-latency.json" \
    node --input-type=module <<'NODE'
import { spawnSync } from "node:child_process";
import { writeFileSync } from "node:fs";
import { performance } from "node:perf_hooks";

const samples = [];
for (let index = 0; index < 20; index += 1) {
  const started = performance.now();
  const result = spawnSync("docker", [
    "compose",
    "--project-name",
    process.env.AVIA_TASK8_PROJECT,
    "--file",
    process.env.AVIA_TASK8_COMPOSE_FILE,
    "--profile",
    "local-preprod-loader",
    "exec",
    "--no-TTY",
    "preprod-api",
    "wget",
    "--quiet",
    "--output-document=/dev/null",
    "http://127.0.0.1:8080/health/ready",
  ], { encoding: "utf8" });
  if (result.status !== 0) {
    throw new Error(`preprod API readiness probe failed: ${result.stderr}`);
  }
  samples.push(Number((performance.now() - started).toFixed(3)));
}
samples.sort((left, right) => left - right);
const percentile = (fraction) =>
  samples[Math.min(
    samples.length - 1,
    Math.ceil(samples.length * fraction) - 1,
  )];
writeFileSync(
  process.env.AVIA_TASK8_API_LATENCY_PATH,
  `${JSON.stringify({
    schemaVersion: "preprod-api-latency/v1",
    endpoint: "/health/ready",
    sampleCount: samples.length,
    unit: "milliseconds",
    p50: percentile(0.50),
    p95: percentile(0.95),
    maximum: samples.at(-1),
  }, null, 2)}\n`,
  { flag: "wx", mode: 0o600 },
);
NODE

  AVIA_TASK8_CONTRACT_PATH="$evidence_directory/contract.json" \
  AVIA_TASK8_SAMPLES_PATH="$resource_samples_file" \
  AVIA_TASK8_METRICS_PATH="$evidence_directory/runtime-metrics.json" \
  AVIA_TASK8_PROFILE="$profile_name" \
  AVIA_TASK8_PROFILE_VERSION="$profile_version" \
  AVIA_TASK8_GENERATION_SECONDS="$generation_seconds" \
  AVIA_TASK8_DATABASE_BYTES="$database_bytes" \
  AVIA_TASK8_SCENARIO_RELATION_BYTES="$scenario_relation_bytes" \
  AVIA_TASK8_OBJECT_METADATA_RELATION_BYTES="$object_metadata_relation_bytes" \
  AVIA_TASK8_OBJECT_BYTES="$object_bytes" \
  AVIA_TASK8_NAMESPACE_DISK_MIB="$namespace_disk_mib" \
  AVIA_TASK8_WORKER_LAG_SECONDS="$worker_lag_seconds" \
  AVIA_TASK8_PRIVACY_FINDINGS="$privacy_findings" \
    node --input-type=module <<'NODE'
import { readFileSync, writeFileSync } from "node:fs";

const unitScale = {
  B: 1,
  KiB: 1024,
  MiB: 1024 ** 2,
  GiB: 1024 ** 3,
};
const parseBytes = (source) => {
  const value = source.trim().split(/\s+/u)[0];
  const match = value.match(/^([0-9]+(?:[.][0-9]+)?)(B|KiB|MiB|GiB)$/u);
  if (!match) throw new Error(`unrecognized Docker memory value ${value}`);
  return Number(match[1]) * unitScale[match[2]];
};
const contract = JSON.parse(readFileSync(process.env.AVIA_TASK8_CONTRACT_PATH));
const envelope = contract.profile.resourceEnvelope;
const lines = readFileSync(process.env.AVIA_TASK8_SAMPLES_PATH, "utf8")
  .trim()
  .split("\n")
  .filter(Boolean)
  .map((line) => JSON.parse(line));
if (lines.length === 0) {
  throw new Error("loader resource sampler produced no measurements");
}
let peakMemoryBytes = 0;
let peakCpuCores = 0;
for (const sample of lines) {
  peakMemoryBytes = Math.max(peakMemoryBytes, parseBytes(sample.MemUsage));
  peakCpuCores = Math.max(
    peakCpuCores,
    Number.parseFloat(String(sample.CPUPerc).replace("%", "")) / 100,
  );
}
const metrics = {
  schemaVersion: "preprod-profile-runtime-metrics/v1",
  profile: process.env.AVIA_TASK8_PROFILE,
  profileVersion: process.env.AVIA_TASK8_PROFILE_VERSION,
  resourceSampleCount: lines.length,
  generationSeconds: Number(process.env.AVIA_TASK8_GENERATION_SECONDS),
  peakLoaderMemoryMiB: Number((peakMemoryBytes / 1024 ** 2).toFixed(3)),
  peakLoaderCpuCores: Number(peakCpuCores.toFixed(3)),
  databaseBytes: Number(process.env.AVIA_TASK8_DATABASE_BYTES),
  scenarioRelationBytes:
    Number(process.env.AVIA_TASK8_SCENARIO_RELATION_BYTES),
  objectMetadataRelationBytes:
    Number(process.env.AVIA_TASK8_OBJECT_METADATA_RELATION_BYTES),
  objectBytes: Number(process.env.AVIA_TASK8_OBJECT_BYTES),
  namespaceDiskMiB: Number(process.env.AVIA_TASK8_NAMESPACE_DISK_MIB),
  workerLagSeconds: Number(process.env.AVIA_TASK8_WORKER_LAG_SECONDS),
  privacyFindings: Number(process.env.AVIA_TASK8_PRIVACY_FINDINGS),
  resourceEnvelope: envelope,
};
const envelopeViolations = [];
if (metrics.peakLoaderMemoryMiB <= 0) {
  envelopeViolations.push("peak-loader-memory-non-positive");
}
if (metrics.peakLoaderCpuCores <= 0) {
  envelopeViolations.push("peak-loader-cpu-non-positive");
}
if (metrics.generationSeconds > envelope.durationSeconds) {
  envelopeViolations.push("duration-exceeded");
}
if (metrics.peakLoaderMemoryMiB > envelope.memoryMiB) {
  envelopeViolations.push("memory-exceeded");
}
if (metrics.peakLoaderCpuCores > envelope.cpuCores) {
  envelopeViolations.push("cpu-exceeded");
}
if (metrics.namespaceDiskMiB > envelope.diskMiB) {
  envelopeViolations.push("disk-exceeded");
}
if (metrics.objectBytes > envelope.objectBytes) {
  envelopeViolations.push("object-bytes-exceeded");
}
if (metrics.privacyFindings !== 0) {
  envelopeViolations.push("privacy-findings");
}
if (
  metrics.profile === "stress" &&
  metrics.objectBytes !== envelope.objectBytes
) {
  envelopeViolations.push("stress-object-bytes-not-exact");
}
metrics.withinEnvelope = envelopeViolations.length === 0;
metrics.envelopeViolations = envelopeViolations;
writeFileSync(
  process.env.AVIA_TASK8_METRICS_PATH,
  `${JSON.stringify(metrics, null, 2)}\n`,
  { flag: "wx", mode: 0o600 },
);
if (envelopeViolations.length > 0) {
  throw new Error(
    `profile exceeded its frozen resource or privacy envelope: ${
      envelopeViolations.join(",")
    }; metrics=${JSON.stringify(metrics)}`,
  );
}
NODE
fi

cleanup_started_epoch="$(date -u +%s)"
compose_command down --volumes --remove-orphans
if project_residue; then
  fail "whole-namespace cleanup left task-owned Compose residue"
fi

cleanup_authorization_file="$runtime_root/cleanup-authorization.json"
cleanup_issuer="plan-5-task-7-connected-smoke-cleanup"
if [[ "$task8_qualification" == "true" ]]; then
  cleanup_issuer="plan-5-task-8-${profile_name}-cleanup"
fi
write_authorization \
  "DROP_RECREATE_TARGET" \
  "$cleanup_authorization_file" \
  "$cleanup_issuer"
export AVIA_PREPROD_AUTHORIZATION_FILE="$cleanup_authorization_file"

"$repository_root/scripts/load-preprod-data.sh" record-cleanup
compose_command down --volumes --remove-orphans
if project_residue; then
  fail "offline cleanup attestation left task-owned Compose residue"
fi
cleanup_completed_epoch="$(date -u +%s)"
cleanup_seconds="$((cleanup_completed_epoch - cleanup_started_epoch))"
qualification_seconds="$((cleanup_completed_epoch - qualification_started_epoch))"

AVIA_TASK7_INTENT_PATH="$intent_file" \
AVIA_TASK7_CONTROL_STORE="$control_store_directory" \
AVIA_TASK7_RUN_ID="$run_id" \
AVIA_TASK8_QUALIFICATION="$task8_qualification" \
AVIA_TASK8_EVIDENCE_DIRECTORY="$evidence_directory" \
AVIA_TASK8_CLEANUP_SECONDS="$cleanup_seconds" \
AVIA_TASK8_QUALIFICATION_SECONDS="$qualification_seconds" \
  node --input-type=module <<'NODE'
import { readdirSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";

const env = process.env;
const intent = JSON.parse(readFileSync(env.AVIA_TASK7_INTENT_PATH));
const resultDigest = readFileSync(
  path.join(env.AVIA_TASK7_CONTROL_STORE, "runs", `${env.AVIA_TASK7_RUN_ID}.success`),
  "utf8",
).trim();
const cleanupDirectory = path.join(
  env.AVIA_TASK7_CONTROL_STORE,
  "cleanup",
  env.AVIA_TASK7_RUN_ID,
);
const cleanupFiles = readdirSync(cleanupDirectory).filter(
  (name) => name.endsWith(".json"),
);
if (cleanupFiles.length !== 1) {
  throw new Error(`cleanup attestation count = ${cleanupFiles.length}`);
}
const attestation = JSON.parse(readFileSync(
  path.join(cleanupDirectory, cleanupFiles[0]),
));
if (
  attestation.schemaVersion !== "preprod-cleanup-attestation/v1" ||
  attestation.runId !== intent.runId ||
  attestation.intentDigest !== intent.intentDigest ||
  attestation.resultDigest !== resultDigest ||
  attestation.targetFingerprintDigest !== intent.targetFingerprintDigest ||
  !/^sha256:[a-f0-9]{64}$/u.test(attestation.authorizationHash) ||
  !/^sha256:[a-f0-9]{64}$/u.test(attestation.attestationDigest) ||
  cleanupFiles[0] !==
    `${attestation.attestationDigest.replace(/^sha256:/u, "")}.json`
) {
  throw new Error("cleanup attestation does not bind exact retained evidence");
}
const authorizationDirectory = path.join(
  env.AVIA_TASK7_CONTROL_STORE,
  "authorizations",
);
const authorizations = readdirSync(authorizationDirectory)
  .filter((name) => name.endsWith(".json"))
  .map((name) => JSON.parse(readFileSync(
    path.join(authorizationDirectory, name),
  )));
const operations = authorizations.map(({ operation }) => operation).sort();
const expectedOperations = env.AVIA_TASK8_QUALIFICATION === "true"
  ? ["DROP_RECREATE_TARGET", "LOAD_EMPTY_TARGET", "RESUME_RUN"]
  : ["DROP_RECREATE_TARGET", "LOAD_EMPTY_TARGET"];
if (
  JSON.stringify(operations) !==
    JSON.stringify(expectedOperations) ||
  !authorizations.some(
    (authorization) =>
      authorization.operation === "DROP_RECREATE_TARGET" &&
      authorization.authorizationHash === attestation.authorizationHash,
  )
) {
  throw new Error("cleanup authorization evidence is incomplete");
}
if (env.AVIA_TASK8_QUALIFICATION === "true") {
  const resultDirectory = path.join(
    env.AVIA_TASK7_CONTROL_STORE,
    "results",
    env.AVIA_TASK7_RUN_ID,
  );
  const results = readdirSync(resultDirectory)
    .filter((name) => name.endsWith(".json"))
    .map((name) => JSON.parse(readFileSync(
      path.join(resultDirectory, name),
    )));
  if (
    results.length !== 2 ||
    !results.some(({ outcome, failures }) =>
      outcome === "FAILED" && failures.includes("COMMAND_STREAM_FAILED")
    ) ||
    !results.some(({ outcome, resultDigest: digest }) =>
      outcome === "SUCCEEDED" && digest === resultDigest
    )
  ) {
    throw new Error("qualification interruption/resume result history differs");
  }
  const checkpointDirectory = path.join(
    env.AVIA_TASK7_CONTROL_STORE,
    "checkpoints",
    env.AVIA_TASK7_RUN_ID,
  );
  const checkpoints = readdirSync(checkpointDirectory)
    .filter((name) => name.endsWith(".json"))
    .map((name) => JSON.parse(readFileSync(
      path.join(checkpointDirectory, name),
    )));
  writeFileSync(
    path.join(env.AVIA_TASK8_EVIDENCE_DIRECTORY, "control-evidence.json"),
    `${JSON.stringify({
      schemaVersion: "preprod-profile-control-evidence/v1",
      intent,
      results,
      checkpoints,
      authorizations,
      cleanupAttestation: attestation,
    }, null, 2)}\n`,
    { flag: "wx", mode: 0o600 },
  );
  const contract = JSON.parse(readFileSync(path.join(
    env.AVIA_TASK8_EVIDENCE_DIRECTORY,
    "contract.json",
  )));
  const cleanupSeconds = Number(env.AVIA_TASK8_CLEANUP_SECONDS);
  const qualificationSeconds = Number(env.AVIA_TASK8_QUALIFICATION_SECONDS);
  const qualificationEnvelopeSeconds =
    contract.profile.resourceEnvelope.qualificationSeconds;
  const envelopeViolations = [];
  if (!Number.isSafeInteger(cleanupSeconds) || cleanupSeconds < 0) {
    envelopeViolations.push("cleanup-measurement-invalid");
  } else if (
    cleanupSeconds > contract.profile.resourceEnvelope.cleanupSeconds
  ) {
    envelopeViolations.push("cleanup-exceeded");
  }
  if (
    qualificationEnvelopeSeconds !== undefined &&
    (
      !Number.isSafeInteger(qualificationSeconds) ||
      qualificationSeconds < 0
    )
  ) {
    envelopeViolations.push("qualification-measurement-invalid");
  } else if (
    qualificationEnvelopeSeconds !== undefined &&
    qualificationSeconds > qualificationEnvelopeSeconds
  ) {
    envelopeViolations.push("qualification-duration-exceeded");
  }
  writeFileSync(
    path.join(env.AVIA_TASK8_EVIDENCE_DIRECTORY, "cleanup-metrics.json"),
    `${JSON.stringify({
      schemaVersion: "preprod-profile-cleanup-metrics/v1",
      profile: contract.profile.name,
      profileVersion: contract.profile.version,
      cleanupSeconds,
      cleanupEnvelopeSeconds:
        contract.profile.resourceEnvelope.cleanupSeconds,
      qualificationSeconds,
      qualificationEnvelopeSeconds:
        qualificationEnvelopeSeconds ?? null,
      withinEnvelope: envelopeViolations.length === 0,
      envelopeViolations,
      residue: {
        containers: 0,
        volumes: 0,
        networks: 0,
      },
      mode: "whole-disposable-namespace",
      selectiveDeletionAllowed: false,
    }, null, 2)}\n`,
    { flag: "wx", mode: 0o600 },
  );
  if (envelopeViolations.length > 0) {
    throw new Error(
      `cleanup or qualification duration exceeded its frozen envelope: ${
        envelopeViolations.join(",")
      }`,
    );
  }
}
NODE

if [[ "$task8_qualification" == "true" ]]; then
  AVIA_TASK8_EVIDENCE_DIRECTORY="$evidence_directory" \
    node --input-type=module <<'NODE'
import { readdirSync, readFileSync } from "node:fs";
import path from "node:path";

const root = process.env.AVIA_TASK8_EVIDENCE_DIRECTORY;
const entries = readdirSync(root, { withFileTypes: true });
if (
  entries.length !== 7 ||
  entries.some((entry) => !entry.isFile()) ||
  !entries.some(({ name }) => name === "control-evidence.json") ||
  !entries.some(({ name }) => name === "runtime-metrics.json") ||
  !entries.some(({ name }) => name === "cleanup-metrics.json")
) {
  throw new Error("qualification evidence file set is incomplete");
}
const forbiddenKey = /^(password|totpsecret|recoverycode|provideractiontoken|accesstoken|refreshtoken|privatekey|apikey|clientsecret|token)$/u;
const inspect = (value) => {
  if (Array.isArray(value)) {
    value.forEach(inspect);
    return;
  }
  if (value === null || typeof value !== "object") return;
  for (const [key, nested] of Object.entries(value)) {
    const normalized = key.toLowerCase().replaceAll(/[-_. ]/gu, "");
    if (forbiddenKey.test(normalized)) {
      throw new Error(`forbidden evidence key ${key}`);
    }
    inspect(nested);
  }
};
for (const entry of entries) {
  const source = readFileSync(path.join(root, entry.name), "utf8");
  if (
    /-----BEGIN [A-Z ]*PRIVATE KEY-----|AKIA[0-9A-Z]{16}|ghp_[A-Za-z0-9]{20,}/u
      .test(source)
  ) {
    throw new Error(`secret material found in ${entry.name}`);
  }
  if (entry.name.endsWith(".json")) inspect(JSON.parse(source));
}
NODE
fi

project_owned="false"
if [[ "$task8_qualification" == "true" ]]; then
  printf '%s\n' \
    "Plan 5 Task 8 ${profile_name} profile: verified locally" \
    "profile=${profile_name}; families=40; routes=86; actions=306; roles=8" \
    "privacy=45/45; resume=verified locally; cleanup=verified locally; residue=0"
else
  printf '%s\n' \
    "Plan 5 Task 7 connected smoke scenarios: verified locally" \
    "profile=smoke; families=40; routes=86; actions=306; roles=8" \
    "privacy=45/45; target cleanup=verified locally; residue=0"
fi
