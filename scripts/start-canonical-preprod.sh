#!/usr/bin/env bash
set -euo pipefail

umask 077

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
state_root="${AVIA_CANONICAL_PREPROD_STATE_DIR:-$repository_root/.local/aviasurveil360-canonical-preprod}"
compose_file="$repository_root/deploy/local/compose.yaml"
transport="${AVIA_PREPROD_TRANSPORT:-https}"
project_name="${AVIA_CANONICAL_PREPROD_PROJECT:-aviasurveil360-local-preprod}"
https_port="${AVIA_PREPROD_HTTPS_PORT:-8445}"
http_port="${AVIA_PREPROD_HTTP_PORT:-8085}"
compose_override="${AVIA_PREPROD_COMPOSE_OVERRIDE:-}"
metadata_file="$state_root/runtime.json"
public_host=""
public_tls="false"
skip_build="${AVIA_PREPROD_SKIP_BUILD:-false}"

fail() {
  printf 'canonical-preprod-up: %s\n' "$*" >&2
  exit 1
}

validate_origin() {
  node --input-type=module - "$1" <<'NODE'
const raw = process.argv[2];
let value;
try {
  value = new URL(raw);
} catch {
  throw new Error("origin is not a URL");
}
if (
  !["http:", "https:"].includes(value.protocol) ||
  value.username ||
  value.password ||
  value.pathname !== "/" ||
  value.search ||
  value.hash ||
  value.origin !== raw
) {
  throw new Error("origin must be an exact HTTP(S) origin without credentials, path, query, or fragment");
}
process.stdout.write(`${value.protocol}\t${value.host}`);
NODE
}

case "$state_root" in
  /*) ;;
  *) fail "AVIA_CANONICAL_PREPROD_STATE_DIR must be absolute" ;;
esac
[[ "$state_root" != "/" && "$state_root" != /tmp && "$state_root" != /private/tmp ]] ||
  fail "refusing a broad disposable state directory"
[[ ! -L "$state_root" ]] || fail "AVIA_CANONICAL_PREPROD_STATE_DIR must not be a symlink"
command -v node >/dev/null 2>&1 || fail "node is required"
[[ "$project_name" =~ ^aviasurveil360-(local-preprod(-[a-z0-9][a-z0-9-]*)?|task-[a-z0-9][a-z0-9-]*)$ ]] ||
  fail "AVIA_CANONICAL_PREPROD_PROJECT must identify one exact AviaSurveil360 local-preprod project"
case "$transport" in
  https)
    [[ "$https_port" =~ ^[0-9]+$ && "$https_port" -ge 1024 && "$https_port" -le 65535 ]] ||
      fail "AVIA_PREPROD_HTTPS_PORT must be a user-space TCP port"
    [[ -z "$compose_override" ]] ||
      fail "canonical HTTPS transport does not accept a Compose override"
    web_origin="${AVIA_PREPROD_WEB_ORIGIN:-https://localhost:${https_port}}"
    [[ "$web_origin" == "https://localhost:${https_port}" ]] ||
      fail "canonical HTTPS transport requires https://localhost:${https_port}"
    ;;
  http)
    [[ "$http_port" =~ ^[0-9]+$ && "$http_port" -ge 1024 && "$http_port" -le 65535 ]] ||
      fail "AVIA_PREPROD_HTTP_PORT must be a user-space TCP port"
    http_override="$repository_root/deploy/local/compose.local-http.yaml"
    [[ "$compose_override" == "$http_override" && -f "$compose_override" && ! -L "$compose_override" ]] ||
      fail "HTTP transport requires the task-owned deploy/local/compose.local-http.yaml override"
    web_origin="${AVIA_PREPROD_WEB_ORIGIN:-http://localhost:${http_port}}"
    ;;
  *)
    fail "AVIA_PREPROD_TRANSPORT must be https or http"
    ;;
esac
origin_parts="$(validate_origin "$web_origin")" ||
  fail "AVIA_PREPROD_WEB_ORIGIN must be an absolute HTTP(S) origin without a path"
IFS=$'\t' read -r origin_scheme public_host <<<"$origin_parts"
[[ -n "$public_host" ]] || fail "AVIA_PREPROD_WEB_ORIGIN must include a host"
origin_scheme="${origin_scheme%:}"
[[ "$origin_scheme" == "http" || "$origin_scheme" == "https" ]] ||
  fail "AVIA_PREPROD_WEB_ORIGIN must use HTTP or HTTPS"
[[ "$origin_scheme" == "https" ]] && public_tls="true"
cookie_secure="${AVIA_PREPROD_COOKIE_SECURE:-$public_tls}"
case "$cookie_secure" in
  true|false) ;;
  *) fail "AVIA_PREPROD_COOKIE_SECURE must be true or false" ;;
esac
case "$skip_build" in
  true|false) ;;
  *) fail "AVIA_PREPROD_SKIP_BUILD must be true or false" ;;
esac
[[ "$cookie_secure" == "$public_tls" ]] ||
  fail "AVIA_PREPROD_COOKIE_SECURE must match the public origin TLS mode"
[[ -f "$compose_file" ]] || fail "Compose file is missing"
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
  AVIA_PREPROD_HTTP_PORT="$http_port" \
  AVIA_PREPROD_WEB_ORIGIN="$web_origin" \
  AVIA_PREPROD_PUBLIC_HOST="$public_host" \
  AVIA_PREPROD_ORIGIN_SCHEME="$origin_scheme" \
  AVIA_PREPROD_PUBLIC_TLS="$public_tls" \
  AVIA_PREPROD_COOKIE_SECURE="$cookie_secure" \
    docker compose "${compose_args[@]}" \
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
AVIA_PREPROD_ORIGIN_SCHEME="$origin_scheme" \
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

if [[ "$skip_build" == false ]]; then
  compose build \
    preprod-migration \
    preprod-normal-runtime-role-provisioner \
    preprod-canonical-aga-loader \
    preprod-canonical-demo-identity-loader \
    preprod-clamav \
    preprod-gotenberg \
    preprod-auth \
    preprod-minio \
    preprod-mailpit \
    preprod-api \
    preprod-worker \
    preprod-web-http \
    preprod-gateway
fi

compose up --detach --wait --wait-timeout 300 \
  preprod-postgres \
  preprod-auth-postgres \
  preprod-auth-mailpit \
  preprod-auth \
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

compose up --detach preprod-canonical-demo-identity-loader
compose wait preprod-canonical-demo-identity-loader
identity_loader_container="$(compose ps -aq preprod-canonical-demo-identity-loader)"
[[ -n "$identity_loader_container" ]] || fail "canonical demo identity loader disappeared before completion"
identity_loader_exit_code="$(docker inspect --format '{{.State.ExitCode}}' "$identity_loader_container")"
[[ "$identity_loader_exit_code" == "0" ]] ||
  fail "canonical demo identity loader exited with status $identity_loader_exit_code"
unset identity_loader_container identity_loader_exit_code

demo_identity_counts="$(
  compose exec --no-TTY preprod-postgres psql \
    --username aviasurveil360_preprod_loader \
    --dbname aviasurveil360_local_preprod \
    --tuples-only --no-align --field-separator '|' \
    --command "
      SELECT
        (SELECT count(*) FROM identity_references WHERE email LIKE '%@synthetic.invalid'),
        (SELECT count(*) FROM user_profiles WHERE subject_id IN (
          SELECT subject_id FROM identity_references WHERE email LIKE '%@synthetic.invalid'
        )),
        (SELECT count(*) FROM desired_membership_versions
          WHERE membership_id LIKE 'CANONICAL-DEMO-MEMBERSHIP-%'
            AND revision = 1 AND membership_state = 'ACTIVE'),
        (SELECT count(*) FROM caa_department_memberships
          WHERE root_id = 'CANONICAL-DEMO-DEPARTMENT-MANAGER'
            AND status = 'ACTIVE');
    " | tr -d '[:space:]'
)"
[[ "$demo_identity_counts" == "9|9|9|1" ]] ||
  fail "demo identity seed count mismatch: $demo_identity_counts"
unset demo_identity_counts

compose up --detach --wait --wait-timeout 300 \
  preprod-normal-runtime-role-provisioner \
  preprod-canonical-aga-loader \
  preprod-api \
  preprod-worker \
  preprod-gateway

cat >"$metadata_file" <<EOF
{
  "schemaVersion": "canonical-preprod-runtime/v2",
  "project": "$project_name",
  "stateDirectory": "$state_root",
  "profile": "aga-preprod@1.0.0",
  "identityNamespace": "canonical-aga-preprod-exercise-v1",
  "identityProvider": "first-party",
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
