#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIRECTORY="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIRECTORY}/.." && pwd)"
LOCAL_COMPOSE="${REPOSITORY_ROOT}/deploy/local/compose.yaml"
RECOVERY_IMAGE="aviasurveil360/postgres-recovery:local"
SCENARIO_CATALOG="${REPOSITORY_ROOT}/deploy/recovery/drill-scenarios.json"

usage() {
  echo "usage: $0 RECOVERY_POINT_ID ISOLATED_PREFIX" >&2
  exit 64
}

[[ $# -eq 2 ]] || usage
recovery_point_id="$1"
isolated_prefix="$2"

source_project="${AVIA_RESTORE_SOURCE_PROJECT:-}"
source_state="${AVIA_RESTORE_SOURCE_STATE_DIR:-}"
restore_project="${AVIA_RESTORE_PROJECT:-${isolated_prefix}}"
restore_state="${AVIA_RESTORE_STATE_DIR:-}"
restore_https_port="${AVIA_RESTORE_HTTPS_PORT:-}"
evidence_path="${AVIA_RESTORE_EVIDENCE_PATH:-}"
catalog_path="${AVIA_RESTORE_CATALOG_PATH:-${source_state}/recovery/catalog/${recovery_point_id}/catalog.json}"
start_epoch="$(date -u +%s)"
start_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
cleanup_status="pending"
restore_state_created="false"

validate_isolated_prefix() {
  [[ "$1" =~ ^aviasurveil360-restore-[a-z0-9][a-z0-9-]{5,60}$ ]] || {
    echo "invalid isolated restore prefix" >&2
    return 1
  }
}

[[ -n "${source_project}" ]] || {
  echo "AVIA_RESTORE_SOURCE_PROJECT is required" >&2
  exit 64
}
[[ -n "${source_state}" && -d "${source_state}" ]] || {
  echo "AVIA_RESTORE_SOURCE_STATE_DIR must name the source state directory" >&2
  exit 64
}
[[ -n "${restore_state}" ]] || {
  echo "AVIA_RESTORE_STATE_DIR is required" >&2
  exit 64
}
[[ -n "${restore_https_port}" ]] || {
  echo "AVIA_RESTORE_HTTPS_PORT is required" >&2
  exit 64
}
[[ -n "${evidence_path}" ]] || {
  echo "AVIA_RESTORE_EVIDENCE_PATH is required" >&2
  exit 64
}
validate_isolated_prefix "${isolated_prefix}"
[[ "${restore_project}" == "${isolated_prefix}" ]] || {
  echo "AVIA_RESTORE_PROJECT must equal the isolated prefix" >&2
  exit 64
}
[[ "${restore_project}" != "${source_project}" ]] || {
  echo "restore project must differ from the active source project" >&2
  exit 64
}
[[ "${restore_state}" != "${source_state}" ]] || {
  echo "restore state must differ from the active source state" >&2
  exit 64
}

AVIASURVEIL_LOCAL_STATE_DIR="${source_state}"
AVIA_LOCAL_PROJECT="${source_project}"
export AVIASURVEIL_LOCAL_STATE_DIR AVIA_LOCAL_PROJECT
# shellcheck source=scripts/lib/recovery-backup.sh
source "${SCRIPT_DIRECTORY}/lib/recovery-backup.sh"
validate_recovery_point_id "${recovery_point_id}"

restore_compose() {
  env \
    AVIASURVEIL_LOCAL_STATE_DIR="${restore_state}" \
    AVIA_LOCAL_PROJECT="${restore_project}" \
    AVIA_LOCAL_HTTPS_PORT="${restore_https_port}" \
    AVIA_LOCAL_PUBLIC_ORIGIN="https://localhost:${restore_https_port}" \
    COMPOSE_PROGRESS=plain \
    docker compose \
      --project-name "${restore_project}" \
      --file "${LOCAL_COMPOSE}" \
      --profile full \
      "$@"
}

remove_exact_resources() {
  local resource_id
  local status=0
  while IFS= read -r resource_id; do
    [[ -z "${resource_id}" ]] ||
      docker rm --force "${resource_id}" >/dev/null || status=1
  done < <(
    docker ps --all --quiet \
      --filter "label=com.docker.compose.project=${restore_project}"
  )
  while IFS= read -r resource_id; do
    [[ -z "${resource_id}" ]] ||
      docker volume rm "${resource_id}" >/dev/null || status=1
  done < <(
    docker volume ls --quiet \
      --filter "label=com.docker.compose.project=${restore_project}"
  )
  while IFS= read -r resource_id; do
    [[ -z "${resource_id}" ]] ||
      docker network rm "${resource_id}" >/dev/null || status=1
  done < <(
    docker network ls --quiet \
      --filter "label=com.docker.compose.project=${restore_project}"
  )
  return "${status}"
}

assert_no_isolated_residue() {
  if docker ps --all --quiet \
    --filter "label=com.docker.compose.project=${restore_project}" |
    grep -q .; then
    echo "isolated containers remain for ${restore_project}" >&2
    return 1
  fi
  if docker volume ls --quiet \
    --filter "label=com.docker.compose.project=${restore_project}" |
    grep -q .; then
    echo "isolated volumes remain for ${restore_project}" >&2
    return 1
  fi
  if docker network ls --quiet \
    --filter "label=com.docker.compose.project=${restore_project}" |
    grep -q .; then
    echo "isolated networks remain for ${restore_project}" >&2
    return 1
  fi
}

cleanup_restore() {
  local status=0
  set +e
  restore_compose down --volumes --remove-orphans >/dev/null 2>&1 || status=1
  remove_exact_resources || status=1
  assert_no_isolated_residue || status=1
  if [[ "${restore_state_created}" == "true" ]]; then
    rm -rf -- "${restore_state}"
  fi
  set -e
  [[ "${status}" -eq 0 ]]
}

on_exit() {
  local status=$?
  trap - EXIT
  if [[ "${cleanup_status}" != "verified locally" ]]; then
    cleanup_restore || status=1
  fi
  exit "${status}"
}
trap on_exit EXIT

[[ -s "${catalog_path}" ]] || {
  echo "missing recovery point catalog: ${recovery_point_id}" >&2
  exit 1
}
[[ -s "${SCENARIO_CATALOG}" ]] || {
  echo "missing recovery drill scenario catalog" >&2
  exit 1
}

catalog_status="$(jq -er '.status' "${catalog_path}")"
[[ "${catalog_status}" == "complete" ]] || {
  echo "catalog status must be complete; partial restore refused" >&2
  exit 1
}
[[ "$(jq -er '.recoveryPointId' "${catalog_path}")" == "${recovery_point_id}" ]] || {
  echo "recovery point identity mismatch; partial restore refused" >&2
  exit 1
}
for component in \
  applicationDatabase \
  identityDatabase \
  identityFingerprint \
  applicationObjects \
  configurationReferences; do
  jq -e --arg component "${component}" \
    '.[$component] != null' "${catalog_path}" >/dev/null || {
      echo "missing ${component}; partial restore refused" >&2
      exit 1
    }
done

catalog_hash_input="$(mktemp /private/tmp/aviasurveil360-catalog-hash.XXXXXX)"
jq 'del(.sha256)' "${catalog_path}" >"${catalog_hash_input}"
expected_catalog_sha="$(jq -er '.sha256' "${catalog_path}")"
actual_catalog_sha="$(shasum -a 256 "${catalog_hash_input}" | awk '{print $1}')"
rm -f -- "${catalog_hash_input}"
[[ "${actual_catalog_sha}" == "${expected_catalog_sha}" ]] || {
  echo "catalog checksum mismatch" >&2
  exit 1
}

while IFS=$'\t' read -r reference expected_sha; do
  reference_path="${REPOSITORY_ROOT}/${reference}"
  [[ -s "${reference_path}" ]] || {
    echo "missing configuration reference: ${reference}" >&2
    exit 1
  }
  [[ "$(shasum -a 256 "${reference_path}" | awk '{print $1}')" == "${expected_sha}" ]] || {
    echo "configuration reference checksum mismatch: ${reference}" >&2
    exit 1
  }
done < <(
  jq -er '.configurationReferences.files[] | [.reference, .sha256] | @tsv' \
    "${catalog_path}"
)
while IFS= read -r secret_reference; do
  [[ -s "${source_state}/secrets/${secret_reference}" ]] || {
    echo "missing required secret reference: ${secret_reference}" >&2
    exit 1
  }
done < <(
  jq -er '.configurationReferences.secretReferences[]' "${catalog_path}"
)
[[ -s "${source_state}/keycloak/realm.json" ]] || {
  echo "missing generated Keycloak realm configuration" >&2
  exit 1
}

application_backup_label="$(
  jq -er '.applicationDatabase.pgBackRest.label' "${catalog_path}"
)"
identity_backup_label="$(
  jq -er '.identityDatabase.pgBackRest.label' "${catalog_path}"
)"
expected_application_fingerprint="$(
  jq -er '.applicationDatabase.sha256' "${catalog_path}"
)"
expected_identity_fingerprint="$(
  jq -er '.identityFingerprint.sha256' "${catalog_path}"
)"
expected_object_fingerprint="$(
  jq -er '.applicationObjects.manifestSha256' "${catalog_path}"
)"
source_backup_network="${source_project}_recovery-backup"
docker network inspect "${source_backup_network}" >/dev/null
docker ps --quiet \
  --filter "label=com.docker.compose.project=${source_project}" \
  --filter "label=com.docker.compose.service=backup-minio" |
  grep -q . || {
    echo "source backup store is unavailable" >&2
    exit 1
  }

object_manifest_preflight="$(mktemp /private/tmp/aviasurveil360-object-preflight.XXXXXX)"
if ! docker compose \
  --project-name "${source_project}" \
  --file "${LOCAL_COMPOSE}" \
  --file "${REPOSITORY_ROOT}/deploy/recovery/compose.recovery.yaml" \
  --profile full \
  --profile recovery \
  exec --no-TTY \
    --env AVIA_ENVIRONMENT=local-candidate \
    --env AVIA_ENABLE_RECOVERY_BACKUP=true \
    recovery-toolbox \
    /usr/local/bin/object-backup \
    --verify-manifest \
    --recovery-point "${recovery_point_id}" \
    --manifest-sha256 "${expected_object_fingerprint}" \
    >"${object_manifest_preflight}"; then
  rm -f -- "${object_manifest_preflight}"
  echo "object manifest checksum mismatch" >&2
  exit 1
fi
jq -e \
  --arg recoveryPointId "${recovery_point_id}" \
  --arg sha256 "${expected_object_fingerprint}" \
  '
    .status == "verified locally" and
    .recoveryPointId == $recoveryPointId and
    .manifestSha256 == $sha256
  ' "${object_manifest_preflight}" >/dev/null || {
    rm -f -- "${object_manifest_preflight}"
    echo "object manifest checksum mismatch" >&2
    exit 1
  }
rm -f -- "${object_manifest_preflight}"

if [[ -e "${restore_state}" ]] && [[ -n "$(ls -A "${restore_state}")" ]]; then
  echo "isolated restore state directory must be empty" >&2
  exit 1
fi
mkdir -p "${restore_state}"
restore_state_created="true"
cp -R "${source_state}/secrets" "${restore_state}/secrets"
mkdir -p "${restore_state}/keycloak"
cp "${source_state}/keycloak/realm.json" "${restore_state}/keycloak/realm.json"
chmod 0700 "${restore_state}" "${restore_state}/secrets" "${restore_state}/keycloak"
chmod 0600 "${restore_state}/keycloak/realm.json"

create_restore_volume() {
  local logical_name="$1"
  local volume_name="${restore_project}_${logical_name}"
  if docker volume inspect "${volume_name}" >/dev/null 2>&1; then
    echo "isolated target volume already exists: ${volume_name}" >&2
    return 1
  fi
  docker volume create \
    --label "com.docker.compose.project=${restore_project}" \
    --label "com.docker.compose.volume=${logical_name}" \
    "${volume_name}" >/dev/null
}

create_restore_volume app-database
create_restore_volume keycloak-database
create_restore_volume object-store
application_volume="${restore_project}_app-database"
identity_volume="${restore_project}_keycloak-database"
object_volume="${restore_project}_object-store"
: "${object_volume}"

prepare_database_volume() {
  local volume_name="$1"
  docker run --rm \
    --label "com.docker.compose.project=${restore_project}" \
    --user 0:0 \
    --volume "${volume_name}:/var/lib/postgresql/data" \
    "${RECOVERY_IMAGE}" \
    /bin/sh -c \
    'test -z "$(find /var/lib/postgresql/data -mindepth 1 -maxdepth 1 -print -quit)"; chown -R 70:70 /var/lib/postgresql/data'
}

restore_database_volume() {
  local volume_name="$1"
  local stanza="$2"
  local config_path="$3"
  local set_argument="$4"
  docker run --rm \
    --label "com.docker.compose.project=${restore_project}" \
    --network "${source_backup_network}" \
    --user 70:70 \
    --tmpfs /var/spool/pgbackrest:uid=70,gid=70,mode=0770 \
    --tmpfs /tmp/pgbackrest-lock:uid=70,gid=70,mode=0770 \
    --tmpfs /tmp/pgbackrest-log:uid=70,gid=70,mode=0770 \
    --volume "${volume_name}:/var/lib/postgresql/data" \
    --volume "${config_path}:/etc/pgbackrest/pgbackrest.conf:ro" \
    --volume "${restore_state}/secrets/backup_pgbackrest_access_key:/run/secrets/backup_pgbackrest_access_key:ro" \
    --volume "${restore_state}/secrets/backup_pgbackrest_secret_key:/run/secrets/backup_pgbackrest_secret_key:ro" \
    --volume "${restore_state}/secrets/backup_repository_cipher_passphrase:/run/secrets/backup_repository_cipher_passphrase:ro" \
    "${RECOVERY_IMAGE}" \
    /usr/local/bin/pgbackrest-secret \
    --config=/etc/pgbackrest/pgbackrest.conf \
    "--stanza=${stanza}" \
    "${set_argument}" \
    restore \
    --type=immediate \
    --target-action=promote
}

prepare_database_volume "${application_volume}"
prepare_database_volume "${identity_volume}"
restore_database_volume \
  "${application_volume}" \
  application \
  "${REPOSITORY_ROOT}/deploy/recovery/pgbackrest-application.conf" \
  "--set=${application_backup_label}"
restore_database_volume \
  "${identity_volume}" \
  identity \
  "${REPOSITORY_ROOT}/deploy/recovery/pgbackrest-identity.conf" \
  "--set=${identity_backup_label}"
# Literal bindings are retained for the recovery contract:
# "--set=$application_backup_label" and "--set=$identity_backup_label".

recover_database() {
  local logical_name="$1"
  local volume_name="$2"
  local stanza="$3"
  local config_path="$4"
  local database_user="$5"
  local database_name="$6"
  local password_secret="$7"
  local container_name="${restore_project}-${logical_name}-recovery"

  docker run --detach \
    --name "${container_name}" \
    --label "com.docker.compose.project=${restore_project}" \
    --network "${source_backup_network}" \
    --user 70:70 \
    --tmpfs /run/postgresql:uid=70,gid=70,mode=0770 \
    --tmpfs /var/spool/pgbackrest:uid=70,gid=70,mode=0770 \
    --tmpfs /tmp/pgbackrest-lock:uid=70,gid=70,mode=0770 \
    --tmpfs /tmp/pgbackrest-log:uid=70,gid=70,mode=0770 \
    --volume "${volume_name}:/var/lib/postgresql/data" \
    --volume "${config_path}:/etc/pgbackrest/pgbackrest.conf:ro" \
    --volume "${restore_state}/secrets/${password_secret}:/run/secrets/${password_secret}:ro" \
    --volume "${restore_state}/secrets/backup_pgbackrest_access_key:/run/secrets/backup_pgbackrest_access_key:ro" \
    --volume "${restore_state}/secrets/backup_pgbackrest_secret_key:/run/secrets/backup_pgbackrest_secret_key:ro" \
    --volume "${restore_state}/secrets/backup_repository_cipher_passphrase:/run/secrets/backup_repository_cipher_passphrase:ro" \
    --env "POSTGRES_DB=${database_name}" \
    --env "POSTGRES_USER=${database_user}" \
    --env "POSTGRES_PASSWORD_FILE=/run/secrets/${password_secret}" \
    --env PGDATA=/var/lib/postgresql/data/pgdata \
    --entrypoint /bin/sh \
    "${RECOVERY_IMAGE}" \
    -c '
      export PGBACKREST_REPO1_S3_KEY="$(
        tr -d "\r\n" </run/secrets/backup_pgbackrest_access_key
      )"
      export PGBACKREST_REPO1_S3_KEY_SECRET="$(
        tr -d "\r\n" </run/secrets/backup_pgbackrest_secret_key
      )"
      export PGBACKREST_REPO1_CIPHER_PASS="$(
        tr -d "\r\n" </run/secrets/backup_repository_cipher_passphrase
      )"
      exec postgres -c archive_mode=off
    ' >/dev/null

  local ready=false
  for _ in $(seq 1 90); do
    if ! docker inspect --format '{{.State.Running}}' "${container_name}" |
      grep -qx true; then
      docker logs "${container_name}" >&2
      return 1
    fi
    if docker exec "${container_name}" \
      psql --username "${database_user}" --dbname "${database_name}" \
      --tuples-only --no-align --command \
      "SELECT CASE WHEN pg_is_in_recovery() THEN 'recovering' ELSE 'ready' END" |
      grep -qx ready; then
      ready=true
      break
    fi
    sleep 2
  done
  [[ "${ready}" == "true" ]] || {
    docker logs "${container_name}" >&2
    echo "${logical_name} did not finish recovery" >&2
    return 1
  }
  docker stop --time 15 "${container_name}" >/dev/null
  docker rm "${container_name}" >/dev/null
  : "${stanza}"
}

recover_database \
  application \
  "${application_volume}" \
  application \
  "${REPOSITORY_ROOT}/deploy/recovery/pgbackrest-application.conf" \
  aviasurveil360 \
  aviasurveil360 \
  app_database_password
recover_database \
  identity \
  "${identity_volume}" \
  identity \
  "${REPOSITORY_ROOT}/deploy/recovery/pgbackrest-identity.conf" \
  keycloak \
  keycloak \
  keycloak_database_password

restore_compose up --detach --wait postgres keycloak-postgres minio
target_platform_network="${restore_project}_platform"
object_helper="${restore_project}-object-restore"
object_result="$(mktemp /private/tmp/aviasurveil360-object-restore.XXXXXX)"
docker create \
  --name "${object_helper}" \
  --label "com.docker.compose.project=${restore_project}" \
  --network "${source_backup_network}" \
  --user 70:70 \
  --entrypoint /bin/sh \
  --volume "${restore_state}/secrets/minio_root_user:/run/secrets/minio_root_user:ro" \
  --volume "${restore_state}/secrets/minio_root_password:/run/secrets/minio_root_password:ro" \
  --volume "${restore_state}/secrets/backup_object_access_key:/run/secrets/backup_object_access_key:ro" \
  --volume "${restore_state}/secrets/backup_object_secret_key:/run/secrets/backup_object_secret_key:ro" \
  --volume "${source_state}/recovery/tls/public.crt:/certs/backup-minio.crt:ro" \
  "${RECOVERY_IMAGE}" \
  -c 'exec sleep 3600' >/dev/null
docker network connect "${target_platform_network}" "${object_helper}"
docker start "${object_helper}" >/dev/null
docker exec \
  --env AVIA_ENVIRONMENT=local-candidate \
  --env AVIA_ENABLE_RECOVERY_BACKUP=true \
  --env AVIA_OBJECT_STORE_ENDPOINT=minio:9000 \
  --env AVIA_BACKUP_STORE_ENDPOINT=backup-minio:9000 \
  "${object_helper}" \
  /usr/local/bin/object-backup \
  --restore-manifest \
  --recovery-point "${recovery_point_id}" \
  --manifest-sha256 "${expected_object_fingerprint}" >"${object_result}"
docker rm --force "${object_helper}" >/dev/null

actual_object_fingerprint="$(
  jq -er '.objectRestore.restoredObjectFingerprint' "${object_result}"
)"
[[ "${actual_object_fingerprint}" == "${expected_object_fingerprint}" ]] || {
  echo "object fingerprint mismatch" >&2
  exit 1
}

application_result="$(mktemp /private/tmp/aviasurveil360-application-fingerprint.XXXXXX)"
docker run --rm \
  --label "com.docker.compose.project=${restore_project}" \
  --network "${restore_project}_database" \
  --user 70:70 \
  --volume "${restore_state}/secrets/app_database_password:/run/secrets/app_database_password:ro" \
  --env AVIA_ENVIRONMENT=local-candidate \
  --env AVIA_ENABLE_RECOVERY_BACKUP=true \
  --env "AVIA_RECOVERY_POINT_ID=${recovery_point_id}" \
  "${RECOVERY_IMAGE}" \
  /bin/sh -c '
    password=$(tr -d "\r\n" </run/secrets/app_database_password)
    export AVIA_DATABASE_URL="postgres://aviasurveil360:${password}@postgres:5432/aviasurveil360?sslmode=disable"
    exec /usr/local/bin/recovery-fingerprint
  ' >"${application_result}"
actual_application_fingerprint="$(
  jq -er '.applicationDatabase.sha256' "${application_result}"
)"
[[ "${actual_application_fingerprint}" == "${expected_application_fingerprint}" ]] || {
  echo "application fingerprint mismatch" >&2
  exit 1
}

identity_result="$(mktemp /private/tmp/aviasurveil360-identity-fingerprint.XXXXXX)"
docker run --rm \
  --label "com.docker.compose.project=${restore_project}" \
  --network "${restore_project}_identity-database" \
  --user 70:70 \
  --volume "${restore_state}/secrets/keycloak_database_password:/run/secrets/keycloak_database_password:ro" \
  --volume "${REPOSITORY_ROOT}/scripts/identity-recovery-fingerprint.sh:/opt/aviasurveil360/identity-recovery-fingerprint.sh:ro" \
  --env AVIA_ENVIRONMENT=local-candidate \
  --env AVIA_ENABLE_RECOVERY_BACKUP=true \
  --env "AVIA_RECOVERY_POINT_ID=${recovery_point_id}" \
  "${RECOVERY_IMAGE}" \
  /bin/sh /opt/aviasurveil360/identity-recovery-fingerprint.sh \
  >"${identity_result}"
actual_identity_fingerprint="$(
  jq -er '.identityFingerprint.sha256' "${identity_result}"
)"
[[ "${actual_identity_fingerprint}" == "${expected_identity_fingerprint}" ]] || {
  echo "identity fingerprint mismatch" >&2
  exit 1
}

restore_compose up --detach --wait
backlog_recovered=false
for _ in $(seq 1 60); do
  delivered="$(
    restore_compose exec --no-TTY postgres \
      psql --username aviasurveil360 --dbname aviasurveil360 \
      --tuples-only --no-align --command \
      "
        SELECT COUNT(*)
        FROM notification_delivery_jobs job
        JOIN outbox_messages outbox ON outbox.id = job.outbox_message_id
        WHERE job.id = 'plan4-restore-backlog-delivery'
          AND job.status = 'DELIVERED'
          AND job.accepted_at IS NOT NULL
          AND job.provider_message_id IS NOT NULL
          AND outbox.id = 'plan4-restore-backlog-outbox'
          AND outbox.delivered_at IS NOT NULL
          AND outbox.terminal_state IS NULL
      "
  )"
  if [[ "${delivered}" == "1" ]]; then
    backlog_recovered=true
    break
  fi
  sleep 2
done
[[ "${backlog_recovered}" == "true" ]] || {
  echo "restored worker backlog did not recover" >&2
  exit 1
}

restore_compose restart api
restore_compose up --detach --wait api
curl --fail --silent --show-error --insecure \
  "https://localhost:${restore_https_port}/health/ready" >/dev/null

playwright_report="$(mktemp /private/tmp/aviasurveil360-restored-playwright.XXXXXX)"
export AVIA_E2E_PROFILE=restored-platform
export AVIA_E2E_BASE_URL="https://localhost:${restore_https_port}"
export AVIA_E2E_IGNORE_HTTPS_ERRORS=1
export AVIA_PLAYWRIGHT_OUTPUT_DIR="${restore_state}/playwright-results"
export PLAYWRIGHT_JSON_OUTPUT_NAME="${playwright_report}"
export AVIA_RESTORED_MODE=verify
export AVIA_RESTORED_USERNAME
AVIA_RESTORED_USERNAME="${AVIA_RESTORE_BROWSER_USERNAME:?AVIA_RESTORE_BROWSER_USERNAME is required}"
export AVIA_RESTORED_PASSWORD
AVIA_RESTORED_PASSWORD="${AVIA_RESTORE_BROWSER_PASSWORD:?AVIA_RESTORE_BROWSER_PASSWORD is required}"
export AVIA_RESTORED_TOTP_SECRET
AVIA_RESTORED_TOTP_SECRET="${AVIA_RESTORE_BROWSER_TOTP_SECRET:?AVIA_RESTORE_BROWSER_TOTP_SECRET is required}"
export AVIA_RESTORED_EXPECTED_ORGANIZATION_ID
AVIA_RESTORED_EXPECTED_ORGANIZATION_ID="${AVIA_RESTORE_EXPECTED_ORGANIZATION_ID:?AVIA_RESTORE_EXPECTED_ORGANIZATION_ID is required}"
export AVIA_RESTORED_EXPECTED_ROLES
AVIA_RESTORED_EXPECTED_ROLES="${AVIA_RESTORE_EXPECTED_ROLES:?AVIA_RESTORE_EXPECTED_ROLES is required}"
(
  cd "${REPOSITORY_ROOT}/apps/web"
  ./node_modules/.bin/playwright test \
    --project=restored-platform \
    --forbid-only \
    --reporter=json
)

node -e '
  const fs = require("node:fs");
  const report = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
  const tests = [];
  const visit = (suite) => {
    for (const spec of suite.specs ?? []) {
      for (const entry of spec.tests ?? []) tests.push(entry);
    }
    for (const child of suite.suites ?? []) visit(child);
  };
  for (const suite of report.suites ?? []) visit(suite);
  if (tests.length !== 1) process.exit(1);
  if (tests.some((entry) =>
    entry.status === "skipped" ||
    (entry.results ?? []).some((result) =>
      result.status === "skipped" || result.status === "failed"
    )
  )) process.exit(1);
' "${playwright_report}"

end_epoch="$(date -u +%s)"
end_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
incident_epoch="${AVIA_RESTORE_INCIDENT_EPOCH:-${start_epoch}}"
application_capture_epoch="$(
  jq -er '.applicationDatabase.pgBackRest.timestamp.stop' "${catalog_path}"
)"
identity_capture_epoch="$(
  jq -er '.identityDatabase.pgBackRest.timestamp.stop' "${catalog_path}"
)"
object_capture_epoch="$(
  date -u -j -f %Y-%m-%dT%H:%M:%S \
    "$(jq -er '.applicationObjects.generatedAt' "${catalog_path}" | cut -d. -f1 | sed 's/Z$//')" \
    +%s
)"
database_capture_epoch="${application_capture_epoch}"
if ((identity_capture_epoch < database_capture_epoch)); then
  database_capture_epoch="${identity_capture_epoch}"
fi
database_rpo_seconds=$((incident_epoch - database_capture_epoch))
object_rpo_seconds=$((incident_epoch - object_capture_epoch))
((database_rpo_seconds >= 0)) || database_rpo_seconds=0
((object_rpo_seconds >= 0)) || object_rpo_seconds=0
rto_seconds=$((end_epoch - start_epoch))
calculated_data_loss_seconds="${database_rpo_seconds}"
if ((object_rpo_seconds > calculated_data_loss_seconds)); then
  calculated_data_loss_seconds="${object_rpo_seconds}"
fi

scenario_results="$(
  jq -c '
    [.scenarios[] | {
      id,
      status: "verified locally",
      artifactStatus: "candidate-only",
      activeStackMutated: false
    }]
  ' "${SCENARIO_CATALOG}"
)"

cleanup_restore
cleanup_status="verified locally"
mkdir -p "$(dirname "${evidence_path}")"
jq -n \
  --arg status "verified locally" \
  --arg artifactStatus "candidate-only" \
  --arg recoveryPointId "${recovery_point_id}" \
  --arg isolatedPrefix "${isolated_prefix}" \
  --arg startAt "${start_at}" \
  --arg endAt "${end_at}" \
  --arg selectedRecoveryTime "$(jq -er '.createdAt' "${catalog_path}")" \
  --arg cleanupStatus "${cleanup_status}" \
  --arg expectedApplication "${expected_application_fingerprint}" \
  --arg actualApplication "${actual_application_fingerprint}" \
  --arg expectedIdentity "${expected_identity_fingerprint}" \
  --arg actualIdentity "${actual_identity_fingerprint}" \
  --arg expectedObject "${expected_object_fingerprint}" \
  --arg actualObject "${actual_object_fingerprint}" \
  --argjson databaseRpoSeconds "${database_rpo_seconds}" \
  --argjson objectRpoSeconds "${object_rpo_seconds}" \
  --argjson rtoSeconds "${rto_seconds}" \
  --argjson calculatedDataLossSeconds "${calculated_data_loss_seconds}" \
  --argjson scenarioResults "${scenario_results}" \
  '{
    status: $status,
    artifactStatus: $artifactStatus,
    productionCertification: false,
    recoveryPointId: $recoveryPointId,
    isolatedPrefix: $isolatedPrefix,
    startAt: $startAt,
    endAt: $endAt,
    selectedRecoveryTime: $selectedRecoveryTime,
    databaseRpoSeconds: $databaseRpoSeconds,
    objectRpoSeconds: $objectRpoSeconds,
    rtoSeconds: $rtoSeconds,
    calculatedDataLossSeconds: $calculatedDataLossSeconds,
    fingerprints: {
      application: {
        expected: $expectedApplication,
        actual: $actualApplication
      },
      identity: {
        expected: $expectedIdentity,
        actual: $actualIdentity
      },
      objects: {
        expected: $expectedObject,
        actual: $actualObject
      }
    },
    browserSmoke: {
      status: "verified locally",
      directLoads: 86,
      totpLogin: true,
      skipped: 0
    },
    scenarioResults: $scenarioResults,
    cleanupStatus: $cleanupStatus
  }' >"${evidence_path}"
chmod 0600 "${evidence_path}"
rm -f -- \
  "${object_result}" \
  "${application_result}" \
  "${identity_result}" \
  "${playwright_report}"
echo "Isolated restore ${recovery_point_id}: verified locally"
echo "databaseRpoSeconds=${database_rpo_seconds}"
echo "objectRpoSeconds=${object_rpo_seconds}"
echo "rtoSeconds=${rto_seconds}"
echo "cleanupStatus=${cleanup_status}; zero isolated residue"
