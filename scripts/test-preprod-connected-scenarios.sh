#!/usr/bin/env bash
set -euo pipefail

umask 077

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_file="$repository_root/deploy/local/compose.yaml"
project_name="aviasurveil360-local-preprod"
profile_name="${1:-}"
profile_version="1.0.0"
run_id="run-task7-connected-smoke"
runtime_root=""
state_directory=""
control_store_directory=""
prepare_configuration_file=""
connected_configuration_file=""
authorization_file=""
seed_file=""
project_owned="false"

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

cleanup() {
  local status=$?
  trap - EXIT HUP INT TERM
  if [[ "$project_owned" == "true" ]]; then
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
  exit "$status"
}
trap cleanup EXIT HUP INT TERM

if [[ "$#" -ne 1 || "$profile_name" != "smoke" ]]; then
  fail "usage: $0 smoke"
fi
command -v docker >/dev/null 2>&1 || fail "docker is required"
command -v node >/dev/null 2>&1 || fail "node is required"
command -v openssl >/dev/null 2>&1 || fail "openssl is required"
docker info >/dev/null 2>&1 || fail "docker daemon is unavailable"
if project_residue; then
  fail "pre-existing $project_name Compose resources must be resolved by their owner"
fi

runtime_root="$(mktemp -d "${TMPDIR:-/tmp}/avia-task7-connected-smoke.XXXXXX")"
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

compose_command build \
  preprod-migration \
  preprod-keycloak \
  preprod-data-loader

project_owned="true"
compose_command up --detach --wait --wait-timeout 240 \
  preprod-postgres \
  preprod-keycloak-postgres \
  preprod-mailpit \
  preprod-minio \
  preprod-keycloak

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
write_authorization \
  "LOAD_EMPTY_TARGET" \
  "$authorization_file" \
  "plan-5-task-7-connected-smoke"
export AVIA_PREPROD_AUTHORIZATION_FILE="$authorization_file"

"$repository_root/scripts/load-preprod-data.sh" verify-authorization
"$repository_root/scripts/load-preprod-data.sh" run-connected

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

domain_count_result="$(
  compose_command exec --no-TTY preprod-postgres \
    psql \
      --username aviasurveil360_preprod_loader \
      --dbname aviasurveil360_local_preprod \
      --tuples-only \
      --no-align \
      --set ON_ERROR_STOP=1 \
      --command "
        WITH actual(name, actual_count, expected_count) AS (
          VALUES
            ('organizations', (SELECT count(*) FROM organizations), 3),
            ('identity_references', (SELECT count(*) FROM identity_references), 9),
            ('desired_membership_versions', (SELECT count(*) FROM desired_membership_versions), 18),
            ('user_profiles', (SELECT count(*) FROM user_profiles), 9),
            ('session_references', (SELECT count(*) FROM session_references), 18),
            ('surveillance_plan_items', (SELECT count(*) FROM surveillance_plan_items), 4),
            ('inspections', (SELECT count(*) FROM inspections), 2),
            ('audit_question_assignments', (SELECT count(*) FROM audit_question_assignments), 3),
            ('template_masters', (SELECT count(*) FROM template_masters), 4),
            ('checklist_template_versions', (SELECT count(*) FROM checklist_template_versions), 6),
            ('question_versions', (SELECT count(*) FROM question_versions), 24),
            ('inspection_packages', (SELECT count(*) FROM inspection_packages), 2),
            ('checklist_responses', (SELECT count(*) FROM checklist_responses), 24),
            ('potential_findings', (SELECT count(*) FROM potential_findings), 12),
            ('findings', (SELECT count(*) FROM findings), 8),
            ('cap_revisions', (SELECT count(*) FROM cap_revisions), 12),
            ('evidence_versions', (SELECT count(*) FROM evidence_versions), 16),
            ('review_decisions', (SELECT count(*) FROM review_decisions), 16),
            ('report_versions', (SELECT count(*) FROM report_versions), 6),
            ('communication_messages', (SELECT count(*) FROM communication_messages), 16),
            ('notification_records', (SELECT count(*) FROM notification_records), 24),
            ('audit_events', (SELECT count(*) FROM audit_events), 250),
            ('outbox_messages', (SELECT count(*) FROM outbox_messages), 80),
            ('notification_delivery_jobs', (SELECT count(*) FROM notification_delivery_jobs), 48),
            ('object_metadata', (SELECT count(*) FROM object_metadata), 24),
            ('document_render_jobs', (SELECT count(*) FROM document_render_jobs), 6),
            ('reminder_dispatches', (SELECT count(*) FROM reminder_dispatches), 20),
            ('offline_grants', (SELECT count(*) FROM offline_grants), 4),
            ('authorized_sync_changes', (SELECT count(*) FROM authorized_sync_changes), 120)
        )
        SELECT CASE
          WHEN count(*) FILTER (WHERE actual_count <> expected_count) = 0
            THEN 'domain-counts-ok'
          ELSE string_agg(
            name || '=' || actual_count || '/' || expected_count,
            ','
            ORDER BY name
          ) FILTER (WHERE actual_count <> expected_count)
        END
        FROM actual;
      " |
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

compose_command down --volumes --remove-orphans
if project_residue; then
  fail "whole-namespace cleanup left task-owned Compose residue"
fi

cleanup_authorization_file="$runtime_root/cleanup-authorization.json"
write_authorization \
  "DROP_RECREATE_TARGET" \
  "$cleanup_authorization_file" \
  "plan-5-task-7-connected-smoke-cleanup"
export AVIA_PREPROD_AUTHORIZATION_FILE="$cleanup_authorization_file"

"$repository_root/scripts/load-preprod-data.sh" record-cleanup
compose_command down --volumes --remove-orphans
if project_residue; then
  fail "offline cleanup attestation left task-owned Compose residue"
fi

AVIA_TASK7_INTENT_PATH="$intent_file" \
AVIA_TASK7_CONTROL_STORE="$control_store_directory" \
AVIA_TASK7_RUN_ID="$run_id" \
  node --input-type=module <<'NODE'
import { readdirSync, readFileSync } from "node:fs";
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
if (
  JSON.stringify(operations) !==
    JSON.stringify(["DROP_RECREATE_TARGET", "LOAD_EMPTY_TARGET"]) ||
  !authorizations.some(
    (authorization) =>
      authorization.operation === "DROP_RECREATE_TARGET" &&
      authorization.authorizationHash === attestation.authorizationHash,
  )
) {
  throw new Error("cleanup authorization evidence is incomplete");
}
NODE

project_owned="false"
printf '%s\n' \
  "Plan 5 Task 7 connected smoke scenarios: verified locally" \
  "profile=smoke; families=40; routes=86; actions=306; roles=8" \
  "privacy=45/45; target cleanup=verified locally; residue=0"
