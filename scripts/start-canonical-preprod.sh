#!/usr/bin/env bash
set -euo pipefail

umask 077

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
state_root="${AVIA_CANONICAL_PREPROD_STATE_DIR:-$repository_root/.local/aviasurveil360-canonical-preprod}"
compose_file="$repository_root/deploy/local/compose.yaml"
project_name="aviasurveil360-local-preprod"
https_port="${AVIA_PREPROD_HTTPS_PORT:-8445}"
web_origin="https://localhost:${https_port}"
metadata_file="$state_root/runtime.json"

fail() {
  printf 'canonical-preprod-up: %s\n' "$*" >&2
  exit 1
}

case "$state_root" in
  /*) ;;
  *) fail "AVIA_CANONICAL_PREPROD_STATE_DIR must be absolute" ;;
esac
[[ "$state_root" != "/" && "$state_root" != /tmp && "$state_root" != /private/tmp ]] ||
  fail "refusing a broad disposable state directory"
[[ "$https_port" == "8445" ]] ||
  fail "canonical preprod currently requires AVIA_PREPROD_HTTPS_PORT=8445"
[[ -f "$compose_file" ]] || fail "Compose file is missing"
command -v docker >/dev/null 2>&1 || fail "docker is required"
command -v node >/dev/null 2>&1 || fail "node is required"
docker info >/dev/null 2>&1 || fail "docker daemon is unavailable"

compose() {
  AVIA_PREPROD_STATE_DIR="$state_root" \
  AVIA_PREPROD_PROFILE="aga-preprod@1.0.0" \
  AVIA_PREPROD_PROFILE_QUALIFICATION="true" \
  AVIA_PREPROD_IDENTITY_NAMESPACE="canonical-aga-preprod-exercise-v1" \
  AVIA_PREPROD_HTTPS_PORT="$https_port" \
  AVIA_PREPROD_WEB_ORIGIN="$web_origin" \
  AVIA_PREPROD_KEYCLOAK_PUBLIC_ORIGIN="$web_origin" \
    docker compose --project-name "$project_name" --file "$compose_file" \
      --profile local-preprod-loader "$@"
}

if [[ -f "$metadata_file" ]]; then
  printf 'Canonical local preprod is already initialized at %s\n' "$state_root"
  exit 0
fi
if docker ps --all --filter "label=com.docker.compose.project=$project_name" \
  --format '{{.ID}}' | grep -q .; then
  fail "existing $project_name resources are not owned by canonical-preprod-up"
fi
if [[ -e "$state_root" ]]; then
  fail "state directory exists without canonical-preprod metadata: $state_root"
fi

mkdir -p "$state_root"
chmod 0700 "$state_root"
AVIA_PREPROD_STATE_DIR="$state_root" \
AVIA_PREPROD_WEB_ORIGIN="$web_origin" \
AVIA_PREPROD_KEYCLOAK_PUBLIC_ORIGIN="$web_origin" \
  "$repository_root/scripts/init-local-preprod-namespace.sh"

cleanup_on_failure=true
cleanup() {
  local status=$?
  trap - EXIT HUP INT TERM
  if [[ "$cleanup_on_failure" == true && "$status" -ne 0 ]]; then
    compose down --volumes --remove-orphans >/dev/null 2>&1 || true
    rm -rf -- "$state_root"
  fi
  exit "$status"
}
trap cleanup EXIT HUP INT TERM

compose build \
  preprod-migration \
  preprod-normal-runtime-role-provisioner \
  preprod-canonical-aga-loader \
  preprod-clamav \
  preprod-gotenberg \
  preprod-keycloak \
  preprod-minio \
  preprod-mailpit \
  preprod-api \
  preprod-worker \
  preprod-scheduler \
  preprod-web-http \
  preprod-gateway

compose up --detach --wait --wait-timeout 300 \
  preprod-postgres \
  preprod-keycloak-postgres \
  preprod-mailpit \
  preprod-minio \
  preprod-clamav \
  preprod-gotenberg \
  preprod-web-http

compose up --detach preprod-migration
compose wait preprod-migration
migration_container="$(compose ps -aq preprod-migration)"
[[ -n "$migration_container" ]] || fail "preprod-migration container disappeared before completion"
migration_exit_code="$(docker inspect --format '{{.State.ExitCode}}' "$migration_container")"
[[ "$migration_exit_code" == "0" ]] ||
  fail "preprod-migration exited with status $migration_exit_code"
unset migration_container migration_exit_code

# The canonical loader requires an explicit active scope/target applicability
# pair.  This is a privacy-safe, task-owned foundation for the disposable
# exercise namespace; it is never used by normal or external environments.
compose exec --no-TTY preprod-postgres psql \
  --username aviasurveil360_preprod_loader \
  --dbname aviasurveil360_local_preprod \
  --command "\
    INSERT INTO organizations (id, legal_name, organization_type, status)\
    VALUES ('ORG-FLY-NAMIBIA', 'Synthetic Fly Namibia Demo', 'OPERATOR', 'ACTIVE')\
    ON CONFLICT (id) DO NOTHING;\
    INSERT INTO regulated_targets (id, target_kind, organization_id)\
    VALUES ('TARGET-OPS-AOC-SOURCE-BOUND', 'ORGANIZATION', 'ORG-FLY-NAMIBIA')\
    ON CONFLICT (id) DO NOTHING;\
    INSERT INTO organization_service_provider_scopes\
      (id, organization_id, service_provider_type_id, authorization_identifier,\
       status, effective_from, primary_target_id)\
    VALUES ('SCOPE-OPS-AOC-SOURCE-BOUND', 'ORG-FLY-NAMIBIA', 'AIR_OPERATOR',\
            'AOC-FLY-NAMIBIA-SOURCE-BOUND', 'ACTIVE', '2025-01-01',\
            'TARGET-OPS-AOC-SOURCE-BOUND')\
    ON CONFLICT (id) DO NOTHING;"

compose up --detach --wait --wait-timeout 300 \
  preprod-normal-runtime-role-provisioner \
  preprod-canonical-aga-loader \
  preprod-keycloak \
  preprod-api \
  preprod-worker \
  preprod-scheduler \
  preprod-gateway

cat >"$metadata_file" <<EOF
{
  "schemaVersion": "canonical-preprod-runtime/v1",
  "project": "$project_name",
  "stateDirectory": "$state_root",
  "profile": "aga-preprod@1.0.0",
  "identityNamespace": "canonical-aga-preprod-exercise-v1",
  "webOrigin": "$web_origin",
  "apiHealth": "$web_origin/health/ready",
  "donorRuntime": "disabled",
  "externalPreprod": "not run"
}
EOF
chmod 0600 "$metadata_file"
cleanup_on_failure=false
printf 'Canonical local preprod is running at %s\n' "$web_origin"
printf 'Exercise catalog: aga-preprod@1.0.0 (disposable namespace only)\n'
