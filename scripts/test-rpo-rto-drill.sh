#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIRECTORY="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIRECTORY}/.." && pwd)"
LOCAL_COMPOSE="${REPOSITORY_ROOT}/deploy/local/compose.yaml"
RECOVERY_COMPOSE="${REPOSITORY_ROOT}/deploy/recovery/compose.recovery.yaml"
RUNTIME_DIRECTORY="$(mktemp -d /private/tmp/aviasurveil360-plan4-drill.XXXXXX)"
AVIA_LOCAL_PROJECT="aviasurveil360-task-plan4-drill-$(date -u +%Y%m%d%H%M%S)-$$"
AVIASURVEIL_LOCAL_STATE_DIR="${RUNTIME_DIRECTORY}/source-state"
SOURCE_HTTPS_PORT="${AVIA_DRILL_HTTPS_PORT:-$((30000 + RANDOM % 10000))}"
RESTORE_1_HTTPS_PORT=$((SOURCE_HTTPS_PORT + 1))
RESTORE_2_HTTPS_PORT=$((SOURCE_HTTPS_PORT + 2))
APPLICATION_ADMIN_USERNAME="restore.admin.$(openssl rand -hex 6)@example.test"
APPLICATION_ADMIN_PASSWORD="$(openssl rand -hex 20)Aa1!"
APPLICATION_ADMIN_SUBJECT_ID=""
EXPECTED_ORGANIZATION_ID="CAA"
EXPECTED_ROLES="admin,executiveDirector,finance,gm,inspector,leadInspector,manager"
TOTP_SECRET_FILE="${RUNTIME_DIRECTORY}/restored-totp-secret"
STACK_STARTED=false

export AVIA_LOCAL_PROJECT AVIASURVEIL_LOCAL_STATE_DIR
export AVIA_LOCAL_HTTPS_PORT="${SOURCE_HTTPS_PORT}"
export AVIA_LOCAL_PUBLIC_ORIGIN="https://localhost:${SOURCE_HTTPS_PORT}"
export COMPOSE_PROGRESS=plain

compose() {
  docker compose \
    --project-name "${AVIA_LOCAL_PROJECT}" \
    --file "${LOCAL_COMPOSE}" \
    --file "${RECOVERY_COMPOSE}" \
    --profile full \
    --profile recovery \
    "$@"
}

remove_source_resources() {
  local resource_id
  local status=0
  while IFS= read -r resource_id; do
    [[ -z "${resource_id}" ]] ||
      docker rm --force "${resource_id}" >/dev/null || status=1
  done < <(
    docker ps --all --quiet \
      --filter "label=com.docker.compose.project=${AVIA_LOCAL_PROJECT}"
  )
  while IFS= read -r resource_id; do
    [[ -z "${resource_id}" ]] ||
      docker volume rm "${resource_id}" >/dev/null || status=1
  done < <(
    docker volume ls --quiet \
      --filter "label=com.docker.compose.project=${AVIA_LOCAL_PROJECT}"
  )
  while IFS= read -r resource_id; do
    [[ -z "${resource_id}" ]] ||
      docker network rm "${resource_id}" >/dev/null || status=1
  done < <(
    docker network ls --quiet \
      --filter "label=com.docker.compose.project=${AVIA_LOCAL_PROJECT}"
  )
  return "${status}"
}

assert_no_project_residue() {
  local project="$1"
  if {
    docker ps --all --quiet \
      --filter "label=com.docker.compose.project=${project}"
    docker volume ls --quiet \
      --filter "label=com.docker.compose.project=${project}"
    docker network ls --quiet \
      --filter "label=com.docker.compose.project=${project}"
  } | sed '/^$/d' | grep -q .; then
    echo "task-owned residue remains for ${project}" >&2
    return 1
  fi
}

cleanup() {
  local status=$?
  trap - EXIT
  set +e
  if [[ "${STACK_STARTED}" == "true" ]]; then
    compose down --volumes --remove-orphans --timeout 15 >/dev/null
  fi
  remove_source_resources || status=1
  assert_no_project_residue "${AVIA_LOCAL_PROJECT}" || status=1
  rm -rf -- "${RUNTIME_DIRECTORY}"
  exit "${status}"
}
trap cleanup EXIT

read_runtime_secret() {
  tr -d '\r\n' <"${AVIASURVEIL_LOCAL_STATE_DIR}/secrets/$1"
}

mkdir -p "${AVIASURVEIL_LOCAL_STATE_DIR}"
chmod 0700 "${AVIASURVEIL_LOCAL_STATE_DIR}"
printf '%s\n' "${AVIA_LOCAL_PROJECT}" \
  >"${AVIASURVEIL_LOCAL_STATE_DIR}/.compose-project-owner"
chmod 0600 "${AVIASURVEIL_LOCAL_STATE_DIR}/.compose-project-owner"

"${SCRIPT_DIRECTORY}/init-local-secrets.sh"
"${SCRIPT_DIRECTORY}/check-local-image-evidence.sh" full
"${SCRIPT_DIRECTORY}/check-local-image-evidence.sh" recovery
STACK_STARTED=true
compose up --detach --wait

KEYCLOAK_BASE_URL="https://localhost:${SOURCE_HTTPS_PORT}/identity"
KEYCLOAK_ADMIN_USERNAME="local-bootstrap-admin"
KEYCLOAK_ADMIN_PASSWORD="$(read_runtime_secret keycloak_bootstrap_admin_password)"
KEYCLOAK_ADMIN_TOKEN="$(
  curl --fail --silent --show-error --insecure \
    --request POST \
    "${KEYCLOAK_BASE_URL}/realms/master/protocol/openid-connect/token" \
    --header "Content-Type: application/x-www-form-urlencoded" \
    --data-urlencode "client_id=admin-cli" \
    --data-urlencode "grant_type=password" \
    --data-urlencode "username=${KEYCLOAK_ADMIN_USERNAME}" \
    --data-urlencode "password=${KEYCLOAK_ADMIN_PASSWORD}" |
    node -e '
      let body = "";
      process.stdin.on("data", (chunk) => body += chunk);
      process.stdin.on("end", () => {
        const value = JSON.parse(body);
        if (!value.access_token) process.exit(1);
        process.stdout.write(value.access_token);
      });
    '
)"

node -e '
  process.stdout.write(JSON.stringify({
    username: process.argv[1],
    email: process.argv[1],
    firstName: "Restore",
    lastName: "Drill Administrator",
    enabled: true,
    emailVerified: true,
    attributes: { organization_id: ["CAA"] },
    requiredActions: ["CONFIGURE_TOTP"],
  }));
' "${APPLICATION_ADMIN_USERNAME}" |
  curl --fail --silent --show-error --insecure \
    --request POST \
    "${KEYCLOAK_BASE_URL}/admin/realms/aviasurveil360/users" \
    --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
    --header "Content-Type: application/json" \
    --data-binary @- \
    --output /dev/null

APPLICATION_ADMIN_SUBJECT_ID="$(
  curl --fail --silent --show-error --insecure \
    --get \
    "${KEYCLOAK_BASE_URL}/admin/realms/aviasurveil360/users" \
    --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
    --data-urlencode "email=${APPLICATION_ADMIN_USERNAME}" \
    --data-urlencode "exact=true" |
    node -e '
      let body = "";
      process.stdin.on("data", (chunk) => body += chunk);
      process.stdin.on("end", () => {
        const users = JSON.parse(body);
        if (users.length !== 1 || !users[0].id) process.exit(1);
        process.stdout.write(users[0].id);
      });
    '
)"
[[ "${APPLICATION_ADMIN_SUBJECT_ID}" =~ ^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$ ]] || {
  echo "Keycloak returned an invalid application administrator subject ID" >&2
  exit 1
}

for role in inspector leadInspector manager gm finance executiveDirector admin; do
  role_json="$(
    curl --fail --silent --show-error --insecure \
      "${KEYCLOAK_BASE_URL}/admin/realms/aviasurveil360/roles/${role}" \
      --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}"
  )"
  node -e \
    'process.stdout.write(JSON.stringify([JSON.parse(process.argv[1])]))' \
    "${role_json}" |
    curl --fail --silent --show-error --insecure \
      --request POST \
      "${KEYCLOAK_BASE_URL}/admin/realms/aviasurveil360/users/${APPLICATION_ADMIN_SUBJECT_ID}/role-mappings/realm" \
      --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
      --header "Content-Type: application/json" \
      --data-binary @- \
      --output /dev/null
done

node -e '
  process.stdout.write(JSON.stringify({
    type: "password",
    temporary: false,
    value: process.argv[1],
  }));
' "${APPLICATION_ADMIN_PASSWORD}" |
  curl --fail --silent --show-error --insecure \
    --request PUT \
    "${KEYCLOAK_BASE_URL}/admin/realms/aviasurveil360/users/${APPLICATION_ADMIN_SUBJECT_ID}/reset-password" \
    --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
    --header "Content-Type: application/json" \
    --data-binary @- \
    --output /dev/null

client_path="${RUNTIME_DIRECTORY}/keycloak-client.json"
client_updated_path="${RUNTIME_DIRECTORY}/keycloak-client-updated.json"
curl --fail --silent --show-error --insecure \
  --get \
  "${KEYCLOAK_BASE_URL}/admin/realms/aviasurveil360/clients" \
  --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
  --data-urlencode "clientId=aviasurveil360-web" >"${client_path}"
node -e '
  const fs = require("node:fs");
  const clients = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
  if (clients.length !== 1 || !clients[0].id) process.exit(1);
  const client = clients[0];
  const origins = process.argv.slice(3);
  client.redirectUris = [
    ...new Set([...(client.redirectUris ?? []), ...origins.map((origin) => `${origin}/auth/callback`)]),
  ];
  client.webOrigins = [
    ...new Set([...(client.webOrigins ?? []), ...origins]),
  ];
  fs.writeFileSync(process.argv[2], `${JSON.stringify(client)}\n`, { mode: 0o600 });
' \
  "${client_path}" \
  "${client_updated_path}" \
  "https://localhost:${RESTORE_1_HTTPS_PORT}" \
  "https://localhost:${RESTORE_2_HTTPS_PORT}"
client_id="$(node -e '
  const value = JSON.parse(require("node:fs").readFileSync(process.argv[1], "utf8"));
  process.stdout.write(value.id);
' "${client_updated_path}")"
curl --fail --silent --show-error --insecure \
  --request PUT \
  "${KEYCLOAK_BASE_URL}/admin/realms/aviasurveil360/clients/${client_id}" \
  --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
  --header "Content-Type: application/json" \
  --data-binary "@${client_updated_path}" \
  --output /dev/null
unset KEYCLOAK_ADMIN_TOKEN role_json

compose stop worker
export AVIA_E2E_PROFILE=restored-platform
export AVIA_E2E_BASE_URL="https://localhost:${SOURCE_HTTPS_PORT}"
export AVIA_E2E_IGNORE_HTTPS_ERRORS=1
export AVIA_PLAYWRIGHT_OUTPUT_DIR="${RUNTIME_DIRECTORY}/prepare-playwright"
export PLAYWRIGHT_JSON_OUTPUT_NAME="${RUNTIME_DIRECTORY}/prepare-report.json"
export AVIA_RESTORED_MODE=prepare
export AVIA_RESTORED_USERNAME="${APPLICATION_ADMIN_USERNAME}"
export AVIA_RESTORED_PASSWORD="${APPLICATION_ADMIN_PASSWORD}"
export AVIA_RESTORED_EXPECTED_ORGANIZATION_ID="${EXPECTED_ORGANIZATION_ID}"
export AVIA_RESTORED_EXPECTED_ROLES="${EXPECTED_ROLES}"
export AVIA_RESTORED_TOTP_SECRET_FILE="${TOTP_SECRET_FILE}"
(
  cd "${REPOSITORY_ROOT}/apps/web"
  ./node_modules/.bin/playwright test \
    --project=restored-platform \
    --forbid-only \
    --reporter=json
)
[[ -s "${TOTP_SECRET_FILE}" ]] || {
  echo "source TOTP enrollment did not produce a protected seed" >&2
  exit 1
}
chmod 0600 "${TOTP_SECRET_FILE}"
TOTP_SECRET="$(tr -d '\r\n' <"${TOTP_SECRET_FILE}")"

identity_email="$(
  compose exec --no-TTY postgres \
    psql --username aviasurveil360 --dbname aviasurveil360 \
    --tuples-only --no-align \
    --command \
    "SELECT COALESCE(email, '') FROM identity_references WHERE subject_id = '${APPLICATION_ADMIN_SUBJECT_ID}'"
)"
[[ "${identity_email}" == "${APPLICATION_ADMIN_USERNAME}" ]] || {
  echo "restored notification recipient identity was not synchronized" >&2
  exit 1
}
compose exec --no-TTY postgres \
  psql --username aviasurveil360 --dbname aviasurveil360 \
  --set ON_ERROR_STOP=1 \
  --command "
    BEGIN;
    INSERT INTO notification_records (
      id, recipient_subject_id, organization_id, title, body,
      related_entity_type, related_entity_id, deduplication_key,
      revision, created_at
    ) VALUES (
      'plan4-restore-backlog-notification',
      '${APPLICATION_ADMIN_SUBJECT_ID}',
      'CAA',
      'Plan 4 restored worker backlog',
      'This local candidate email must complete after isolated restore.',
      'RECOVERY_DRILL',
      'plan4-restore-backlog',
      'plan4-restore-backlog',
      1,
      now()
    );
    INSERT INTO outbox_messages (
      id, topic, aggregate_type, aggregate_id, payload, available_at,
      idempotency_key, operation_id, correlation_id
    ) VALUES (
      'plan4-restore-backlog-outbox',
      'notification.email_requested',
      'NOTIFICATION',
      'plan4-restore-backlog-notification',
      jsonb_build_object(
        'id', 'plan4-restore-backlog-notification',
        'title', 'Plan 4 restored worker backlog'
      ),
      now(),
      'notification-email:plan4-restore-backlog',
      'plan4-restore-backlog',
      'plan4-restore-backlog'
    );
    INSERT INTO notification_delivery_jobs (
      id, notification_id, recipient_subject_id, channel, status,
      idempotency_key, outbox_message_id, attempt_count,
      created_at, updated_at
    ) VALUES (
      'plan4-restore-backlog-delivery',
      'plan4-restore-backlog-notification',
      '${APPLICATION_ADMIN_SUBJECT_ID}',
      'EMAIL',
      'PENDING',
      'notification-email:plan4-restore-backlog',
      'plan4-restore-backlog-outbox',
      0,
      now(),
      now()
    );
    COMMIT;
  " >/dev/null
initial_backlog="$(
  compose exec --no-TTY postgres \
    psql --username aviasurveil360 --dbname aviasurveil360 \
    --tuples-only --no-align --command "
      SELECT COUNT(*)
      FROM notification_delivery_jobs job
      JOIN outbox_messages outbox ON outbox.id = job.outbox_message_id
      WHERE job.id = 'plan4-restore-backlog-delivery'
        AND job.status = 'PENDING'
        AND outbox.id = 'plan4-restore-backlog-outbox'
        AND outbox.topic = 'notification.email_requested'
        AND outbox.delivered_at IS NULL
        AND outbox.terminal_state IS NULL
    "
)"
[[ "${initial_backlog}" == "1" ]] || {
  echo "initial worker backlog was not exactly one pending item" >&2
  exit 1
}

write_object_version() {
  local content="$1"
  local marker="$2"
  compose exec --no-TTY minio /bin/sh -c '
    root_user=$(tr -d "\r\n" </run/secrets/minio_root_user)
    root_password=$(tr -d "\r\n" </run/secrets/minio_root_password)
    mc alias set primary http://127.0.0.1:9000 "$root_user" "$root_password" >/dev/null
    printf "%s\n" "$1" >/tmp/recovery-drill.txt
    mc cp --attr "drill-marker=$2" /tmp/recovery-drill.txt \
      primary/evidence-clean/recovery-fixtures/rpo-rto-drill.txt >/dev/null
    rm -f /tmp/recovery-drill.txt
    unset root_user root_password
  ' shell "${content}" "${marker}"
}

write_object_version "Plan 4 restore drill version one" "drill-1"
first_point="rp-$(date -u +%Y%m%dT%H%M%SZ)-drill1"
"${SCRIPT_DIRECTORY}/verify-backup-catalog.sh" \
  --create "${first_point}" full

compose exec --no-TTY postgres \
  psql \
  --username aviasurveil360 \
  --dbname aviasurveil360 \
  --set ON_ERROR_STOP=1 \
  --command "
    INSERT INTO audit_events (
      event_id,
      occurred_at,
      action,
      entity_type,
      entity_id,
      details
    ) VALUES (
      'plan4-rpo-rto-controlled-${first_point}',
      now(),
      'recovery.drill.changed',
      'RecoveryDrill',
      '${first_point}',
      '{\"artifactStatus\":\"candidate-only\"}'::jsonb
    )
  " >/dev/null
write_object_version "Plan 4 restore drill version two" "drill-2"
sleep 1
second_point="rp-$(date -u +%Y%m%dT%H%M%SZ)-drill2"
"${SCRIPT_DIRECTORY}/verify-backup-catalog.sh" \
  --create "${second_point}" incr

first_catalog="${AVIASURVEIL_LOCAL_STATE_DIR}/recovery/catalog/${first_point}/catalog.json"
second_catalog="${AVIASURVEIL_LOCAL_STATE_DIR}/recovery/catalog/${second_point}/catalog.json"
[[ "$(jq -er '.status' "${first_catalog}")" == "complete" ]]
[[ "$(jq -er '.status' "${second_catalog}")" == "complete" ]]
[[ "$(
  jq -er '.applicationDatabase.sha256' "${first_catalog}"
)" != "$(
  jq -er '.applicationDatabase.sha256' "${second_catalog}"
)" ]] || {
  echo "controlled database change did not alter the fingerprint" >&2
  exit 1
}
[[ "$(
  jq -er '.applicationObjects.manifestSha256' "${first_catalog}"
)" != "$(
  jq -er '.applicationObjects.manifestSha256' "${second_catalog}"
)" ]] || {
  echo "controlled object change did not alter the fingerprint" >&2
  exit 1
}

run_complete_drill() {
  local label="$1"
  local point="$2"
  local port="$3"
  local project="aviasurveil360-restore-${label}-$(date -u +%H%M%S)-$$"
  local state="${RUNTIME_DIRECTORY}/${label}-state"
  local evidence="${RUNTIME_DIRECTORY}/${label}-evidence.json"

  env \
    AVIA_RESTORE_SOURCE_PROJECT="${AVIA_LOCAL_PROJECT}" \
    AVIA_RESTORE_SOURCE_STATE_DIR="${AVIASURVEIL_LOCAL_STATE_DIR}" \
    AVIA_RESTORE_PROJECT="${project}" \
    AVIA_RESTORE_STATE_DIR="${state}" \
    AVIA_RESTORE_HTTPS_PORT="${port}" \
    AVIA_RESTORE_EVIDENCE_PATH="${evidence}" \
    AVIA_RESTORE_BROWSER_USERNAME="${APPLICATION_ADMIN_USERNAME}" \
    AVIA_RESTORE_BROWSER_PASSWORD="${APPLICATION_ADMIN_PASSWORD}" \
    AVIA_RESTORE_BROWSER_TOTP_SECRET="${TOTP_SECRET}" \
    AVIA_RESTORE_EXPECTED_ORGANIZATION_ID="${EXPECTED_ORGANIZATION_ID}" \
    AVIA_RESTORE_EXPECTED_ROLES="${EXPECTED_ROLES}" \
    "${SCRIPT_DIRECTORY}/restore-isolated-stack.sh" \
      "${point}" "${project}"

  jq -e '
    .status == "verified locally" and
    .artifactStatus == "candidate-only" and
    .productionCertification == false and
    .databaseRpoSeconds <= 900 and
    .objectRpoSeconds <= 900 and
    .rtoSeconds <= 3600 and
    .fingerprints.application.expected == .fingerprints.application.actual and
    .fingerprints.identity.expected == .fingerprints.identity.actual and
    .fingerprints.objects.expected == .fingerprints.objects.actual and
    .browserSmoke.directLoads == 86 and
    .browserSmoke.totpLogin == true and
    .browserSmoke.skipped == 0 and
    .cleanupStatus == "verified locally" and
    ([.scenarioResults[].status] | all(. == "verified locally"))
  ' "${evidence}" >/dev/null
  if (( $(jq -r '.databaseRpoSeconds' "${evidence}") > 900 )); then
    echo "database RPO exceeded 900 seconds" >&2
    exit 1
  fi
  if (( $(jq -r '.objectRpoSeconds' "${evidence}") > 900 )); then
    echo "object RPO exceeded 900 seconds" >&2
    exit 1
  fi
  if (( $(jq -r '.rtoSeconds' "${evidence}") > 3600 )); then
    echo "RTO exceeded 3600 seconds" >&2
    exit 1
  fi
  echo "drillRecoveryPoint=$(jq -r '.recoveryPointId' "${evidence}")"
  echo "applicationFingerprint=$(jq -r '.fingerprints.application.actual' "${evidence}")"
  echo "identityFingerprint=$(jq -r '.fingerprints.identity.actual' "${evidence}")"
  echo "objectFingerprint=$(jq -r '.fingerprints.objects.actual' "${evidence}")"
  assert_no_project_residue "${project}"
  LAST_DRILL_EVIDENCE="${evidence}"
}

LAST_DRILL_EVIDENCE=""
run_complete_drill drill-1 "${second_point}" "${RESTORE_1_HTTPS_PORT}"
drill_1_evidence="${LAST_DRILL_EVIDENCE}"

corrupt_catalog="${RUNTIME_DIRECTORY}/latest-corrupt-catalog.json"
jq '.applicationDatabase.sha256 = ("0" * 64)' \
  "${second_catalog}" >"${corrupt_catalog}"
corrupt_project="aviasurveil360-restore-corrupt-$(date -u +%H%M%S)-$$"
if env \
  AVIA_RESTORE_SOURCE_PROJECT="${AVIA_LOCAL_PROJECT}" \
  AVIA_RESTORE_SOURCE_STATE_DIR="${AVIASURVEIL_LOCAL_STATE_DIR}" \
  AVIA_RESTORE_PROJECT="${corrupt_project}" \
  AVIA_RESTORE_STATE_DIR="${RUNTIME_DIRECTORY}/corrupt-state" \
  AVIA_RESTORE_HTTPS_PORT="$((RESTORE_2_HTTPS_PORT + 1))" \
  AVIA_RESTORE_EVIDENCE_PATH="${RUNTIME_DIRECTORY}/corrupt-evidence.json" \
  AVIA_RESTORE_CATALOG_PATH="${corrupt_catalog}" \
  AVIA_RESTORE_BROWSER_USERNAME="${APPLICATION_ADMIN_USERNAME}" \
  AVIA_RESTORE_BROWSER_PASSWORD="${APPLICATION_ADMIN_PASSWORD}" \
  AVIA_RESTORE_BROWSER_TOTP_SECRET="${TOTP_SECRET}" \
  AVIA_RESTORE_EXPECTED_ORGANIZATION_ID="${EXPECTED_ORGANIZATION_ID}" \
  AVIA_RESTORE_EXPECTED_ROLES="${EXPECTED_ROLES}" \
  "${SCRIPT_DIRECTORY}/restore-isolated-stack.sh" \
    "${second_point}" "${corrupt_project}" \
    >"${RUNTIME_DIRECTORY}/corrupt-stdout.log" \
    2>"${RUNTIME_DIRECTORY}/corrupt-stderr.log"; then
  echo "latest-backup-corruption-fallback: corrupt catalog was accepted" >&2
  exit 1
fi
rg --fixed-strings "catalog checksum mismatch" \
  "${RUNTIME_DIRECTORY}/corrupt-stderr.log" >/dev/null
assert_no_project_residue "${corrupt_project}"
echo "latest-backup-corruption-fallback: corrupt latest catalog refused"

run_complete_drill drill-2 "${first_point}" "${RESTORE_2_HTTPS_PORT}"
drill_2_evidence="${LAST_DRILL_EVIDENCE}"

aggregate="${RUNTIME_DIRECTORY}/drill-summary.json"
jq -s \
  '{
    status: "verified locally",
    artifactStatus: "candidate-only",
    productionCertification: false,
    drills: .,
    completeDrills: length,
    corruptionFallback: "verified locally",
    cleanupStatus: "verified locally"
  }' \
  "${drill_1_evidence}" \
  "${drill_2_evidence}" >"${aggregate}"
jq -e '
  .completeDrills == 2 and
  .corruptionFallback == "verified locally" and
  ([.drills[].databaseRpoSeconds] | all(. <= 900)) and
  ([.drills[].objectRpoSeconds] | all(. <= 900)) and
  ([.drills[].rtoSeconds] | all(. <= 3600)) and
  ([.drills[].browserSmoke.directLoads] | all(. == 86)) and
  ([.drills[].browserSmoke.totpLogin] | all(. == true)) and
  ([.drills[].cleanupStatus] | all(. == "verified locally"))
' "${aggregate}" >/dev/null

echo "Two complete isolated recovery drills: verified locally"
echo "Candidate database/object RPO <= 900 seconds: verified locally"
echo "Candidate RTO <= 3600 seconds: verified locally"
echo "Restored OIDC/TOTP role scope and 86 routes: verified locally"
echo "Drill cleanup: zero isolated residue"
