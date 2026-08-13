#!/usr/bin/env bash
set -euo pipefail

umask 077

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
compose_file="$repository_root/deploy/local/compose.yaml"
test_compose_file="$repository_root/deploy/local/compose.test.yaml"
runtime_root="$(mktemp -d /private/tmp/aviasurveil360-canonical-task8-fault.XXXXXX)"
state_root="$runtime_root/preprod-state"
artifact_root="$runtime_root/evidence"
integration_runtime="$runtime_root/integration-runtime"
project_name="aviasurveil360-task-canonical-preprod-fault-$(date -u +%Y%m%d%H%M%S)-$$"
integration_project="aviasurveil360-task8-negative-${RANDOM}-$$"
skip_build="${AVIA_TASK8_SKIP_BUILD:-false}"
stack_started=false
integration_started=false

fail() {
  printf 'canonical-preprod-fault-restart: %s\n' "$*" >&2
  exit 1
}

choose_loopback_port() {
  node --input-type=module <<'NODE'
import net from "node:net";
const server = net.createServer();
server.unref();
server.on("error", (error) => { throw error; });
server.listen(0, "127.0.0.1", () => {
  const address = server.address();
  if (typeof address !== "object" || address === null) process.exit(70);
  process.stdout.write(String(address.port));
  server.close();
});
NODE
}

https_port="${AVIA_TASK8_HTTPS_PORT:-$(choose_loopback_port)}"
integration_port="${AVIA_TASK8_POSTGRES_PORT:-$(choose_loopback_port)}"
web_origin="https://localhost:${https_port}"

for value in "$https_port" "$integration_port"; do
  [[ "$value" =~ ^[0-9]+$ && "$value" -ge 1024 && "$value" -le 65535 ]] ||
    fail "all task-owned ports must be user-space TCP ports"
done
[[ "$https_port" != "$integration_port" ]] ||
  fail "task-owned ports must be distinct"
case "$skip_build" in
  true|false) ;;
  *) fail "AVIA_TASK8_SKIP_BUILD must be true or false" ;;
esac
mkdir -p "$artifact_root"

compose() {
  AVIA_PREPROD_STATE_DIR="$state_root" \
  AVIA_PREPROD_PROFILE="aga-preprod@1.0.0" \
  AVIA_PREPROD_PROFILE_QUALIFICATION="true" \
  AVIA_PREPROD_IDENTITY_NAMESPACE="canonical-aga-preprod-exercise-v1" \
  AVIA_PREPROD_TRANSPORT=https \
  AVIA_PREPROD_HTTPS_PORT="$https_port" \
  AVIA_PREPROD_WEB_ORIGIN="$web_origin" \
  AVIA_PREPROD_PUBLIC_HOST="localhost:${https_port}" \
  AVIA_PREPROD_ORIGIN_SCHEME=https \
  AVIA_PREPROD_PUBLIC_TLS=true \
  AVIA_PREPROD_COOKIE_SECURE=true \
    docker compose --project-name "$project_name" --file "$compose_file" \
      --profile local-preprod-loader "$@"
}

test_compose() {
  AVIA_TEST_RUNTIME_DIR="$integration_runtime" \
  AVIA_TEST_POSTGRES_PORT="$integration_port" \
    docker compose --project-name "$integration_project" \
      --file "$test_compose_file" "$@"
}

project_residue() {
  local project="$1"
  {
    docker ps --all --quiet --filter "label=com.docker.compose.project=$project"
    docker volume ls --quiet --filter "label=com.docker.compose.project=$project"
    docker network ls --quiet --filter "label=com.docker.compose.project=$project"
  } | sed '/^$/d'
}

assert_no_project_residue() {
  local project="$1"
  if project_residue "$project" | grep -q .; then
    fail "task-owned Compose residue remains for $project"
  fi
}

cleanup() {
  local status=$?
  local process_snapshot process_residue
  trap - EXIT HUP INT TERM
  set +e
  if [[ "$stack_started" == true ]] || project_residue "$project_name" | grep -q .; then
    compose down --volumes --remove-orphans >/dev/null 2>&1
  fi
  if [[ "$integration_started" == true ]] || project_residue "$integration_project" | grep -q .; then
    test_compose down --volumes --remove-orphans >/dev/null 2>&1
  fi
  if project_residue "$project_name" | grep -q .; then
    printf 'canonical-preprod-fault-restart: task-owned Compose residue remains for %s\n' "$project_name" >&2
    status=1
  fi
  if project_residue "$integration_project" | grep -q .; then
    printf 'canonical-preprod-fault-restart: task-owned Compose residue remains for %s\n' "$integration_project" >&2
    status=1
  fi
  process_snapshot="$(ps -axo pid=,command=)"
  process_residue="$(printf '%s\n' "$process_snapshot" | awk -v owner="$$" -v marker="$runtime_root" '$1 != owner && index($0, marker) > 0')"
  if [[ -n "$process_residue" ]]; then
    printf 'canonical-preprod-fault-restart: task-owned process residue remains\n' >&2
    printf '%s\n' "$process_residue" >&2
    status=1
  fi
  case "$runtime_root" in
    /private/tmp/aviasurveil360-canonical-task8-fault.*) rm -rf -- "$runtime_root" ;;
    *) printf 'canonical-preprod-fault-restart: refusing unsafe runtime cleanup\n' >&2; status=1 ;;
  esac
  exit "$status"
}
trap cleanup EXIT HUP INT TERM

container_id() {
  compose ps --all --quiet "$1"
}

wait_for_service_health() {
  local service="$1"
  local deadline=$((SECONDS + 360))
  local identifier status
  while ((SECONDS < deadline)); do
    identifier="$(container_id "$service")"
    if [[ -n "$identifier" ]]; then
      status="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$identifier")"
      if [[ "$status" == healthy || "$status" == running ]]; then
        return 0
      fi
    fi
    sleep 1
  done
  fail "$service did not recover to a healthy/running state"
}

request_health() {
  local path="$1"
  local output="$2"
  curl --insecure --silent --show-error --output "$output" --write-out '%{http_code}' \
    "$web_origin$path"
}

wait_for_readiness() {
  local expected_code="$1"
  local expected_status="$2"
  local deadline=$((SECONDS + 180))
  local code
  while ((SECONDS < deadline)); do
    code="$(request_health /health/ready "$artifact_root/readiness.json" || true)"
    if [[ "$code" == "$expected_code" ]] &&
      node --input-type=module - "$artifact_root/readiness.json" "$expected_status" <<'NODE'
import { readFileSync } from "node:fs";
const report = JSON.parse(readFileSync(process.argv[2], "utf8"));
process.exit(report.status === process.argv[3] ? 0 : 1);
NODE
    then
      return 0
    fi
    sleep 1
  done
  fail "readiness did not reach $expected_code/$expected_status"
}

assert_liveness() {
  local code
  code="$(request_health /health/live "$artifact_root/liveness.json")"
  [[ "$code" == 200 ]] || fail "liveness changed with downstream dependency state"
}

inject_dependency_failure() {
  local service="$1"
  local expected_status="$2"
  local expected_code="$3"
  printf 'Injecting dependency loss: %s\n' "$service"
  compose stop --timeout 20 "$service"
  assert_liveness
  wait_for_readiness "$expected_code" "$expected_status"
  compose start "$service"
  wait_for_service_health "$service"
  wait_for_readiness 200 ready
}

assert_worker_restart() {
  local identifier before after deadline
  identifier="$(container_id preprod-worker)"
  before="$(docker inspect --format '{{.RestartCount}}' "$identifier")"
  compose exec --no-TTY preprod-worker sh -c '
    set -- $(cat /proc/1/task/1/children)
    if [ "$#" -ne 1 ]; then
      echo "expected one worker child under container init" >&2
      exit 1
    fi
    kill -KILL "$1"
  '
  deadline=$((SECONDS + 90))
  while ((SECONDS < deadline)); do
    identifier="$(container_id preprod-worker)"
    after="$(docker inspect --format '{{.RestartCount}}' "$identifier")"
    if ((after > before)); then
      wait_for_service_health preprod-worker
      printf 'Worker crash restart recovered: %s -> %s\n' "$before" "$after"
      return 0
    fi
    sleep 1
  done
  fail "preprod-worker did not restart after an injected crash"
}

psql_value() {
  local sql="$1"
  compose exec --no-TTY preprod-postgres psql \
    --username aviasurveil360_preprod_loader \
    --dbname aviasurveil360_local_preprod \
    --tuples-only --no-align --command "$sql" | tr -d '\r'
}

fingerprint_database() {
  local destination="$1"
  local table value
  : >"$destination"
  for table in \
    schema_migrations organizations canonical_question_catalogs \
    canonical_question_catalog_memberships planning_intake_drafts \
    surveillance_plan_items canonical_audit_scope_drafts \
    canonical_audit_scope_selection_questions canonical_audit_scope_snapshots \
    canonical_audit_scope_snapshot_questions audit_assignments audit_team_members \
    audit_question_assignments canonical_audit_preparation_snapshots \
    canonical_audit_preparation_questions inspections inspection_packages \
    inspection_checklists checklist_responses potential_findings findings \
    cap_revisions evidence_versions object_metadata report_versions \
    report_decisions document_records document_versions audit_events \
    command_transaction_links outbox_messages
  do
    value="$(psql_value "SELECT count(*)::text || '|' || COALESCE(jsonb_agg(to_jsonb(row_value) ORDER BY to_jsonb(row_value)::text), '[]'::jsonb)::text FROM $table AS row_value;")"
    printf '%s|%s\n' "$table" "$value" >>"$destination"
  done
  shasum -a 256 "$destination" | awk '{print $1}'
}

wait_for_database_quiescence() {
  local destination="$1"
  local deadline=$((SECONDS + 180))
  local probe="$artifact_root/fingerprint-quiescence.txt"
  local current previous="" active_leases stable_rounds=0
  while ((SECONDS < deadline)); do
    current="$(fingerprint_database "$probe")"
    active_leases="$(psql_value "SELECT count(*) FROM outbox_messages WHERE lease_owner IS NOT NULL AND (lease_expires_at IS NULL OR lease_expires_at > now());")"
    active_leases="${active_leases//[[:space:]]/}"
    if [[ "$current" == "$previous" && "$active_leases" == 0 ]]; then
      stable_rounds=$((stable_rounds + 1))
    else
      stable_rounds=0
    fi
    if ((stable_rounds >= 2)); then
      cp "$probe" "$destination"
      printf '%s\n' "$current"
      return 0
    fi
    previous="$current"
    sleep 3
  done
  fail "authoritative database state did not stabilize before restart fingerprinting"
}

assert_donor_unavailable() {
  local method path status
  while IFS='|' read -r method path; do
    status="$(curl --insecure --silent --show-error --output /dev/null --write-out '%{http_code}' \
      --request "$method" "$web_origin$path")"
    [[ "$status" =~ ^(401|404)$ ]] ||
      fail "pre-auth donor boundary failed closed: $method $path returned $status"
  done <<'EOF'
POST|/api/v1/preprod/aga-demo-workspace/classification/query
GET|/api/v1/admin/governed-checklist/aga-candidate-demo/capability
EOF
}

assert_public_provider_admin_unavailable() {
  local path status
  for path in /identity/v1/directory /identity/admin/v1/directory; do
    status="$(curl --insecure --silent --show-error --output /dev/null --write-out '%{http_code}' \
      "$web_origin$path")"
    [[ "$status" == 404 ]] ||
      fail "public provider-admin route was exposed: $path returned $status"
  done
}

assert_runtime_log_secrets_absent() {
  local secret_path secret_value
  compose logs --no-color >"$artifact_root/compose.log"
  for secret_path in "$state_root"/secrets/*; do
    [[ -f "$secret_path" ]] || continue
    secret_value="$(LC_ALL=C tr -d '\r\n' <"$secret_path")"
    if [[ -n "$secret_value" ]] && LC_ALL=C grep -aFq -- "$secret_value" "$artifact_root/compose.log"; then
      fail "generated secret appeared in runtime logs: ${secret_path##*/}"
    fi
  done
}

run_playwright_file() {
  local file="$1"
  local output="$2"
  (
    cd "$repository_root/apps/web"
    AVIA_E2E_PROFILE=canonical-quick-tunnel \
    AVIA_E2E_BASE_URL="$web_origin" \
    AVIA_E2E_IGNORE_HTTPS_ERRORS=1 \
    AVIA_AGA_OIDC_PASSWORD="$(tr -d '\r\n' <"$state_root/secrets/preprod_canonical_demo_oidc_qualification_password")" \
    AVIA_PLAYWRIGHT_OUTPUT_DIR="$output" \
      ./node_modules/.bin/playwright test "$file" --project=canonical-quick-tunnel
  )
}

run_negative_transaction_matrix() {
  . "$repository_root/scripts/lib/init-local-test-runtime.sh"
  initialize_local_test_runtime "$integration_runtime" "http://127.0.0.1:4174" "$repository_root"
  test_compose up --detach --wait postgres
  integration_started=true
  local database_password
  database_password="$(tr -d '\r\n' <"$integration_runtime/secrets/app_database_password")"
  AVIA_TEST_DATABASE_URL="postgres://aviasurveil:${database_password}@127.0.0.1:${integration_port}/aviasurveil?sslmode=disable" \
  GOCACHE="${AVIA_TASK8_GOCACHE:-/private/tmp/avia-go-cache}" \
    go -C "$repository_root/apps/api" test -tags canonicaltest -p 1 -count=1 \
      ./tests/integration \
      -run '^(TestRoutinePlanningReturnReentryAndAssignmentMaterialization|TestAdHocPlanningWithholdsAuditeeNoticeAfterMaterialization|TestPlanningAssignmentHTTPContractsAndNoticePrivacy|TestLostAcknowledgementReplaysOneCanonicalMutationAndTransitionEnvelope|TestFullPlatformRoleOrganizationAndPrivacyDenials|TestFullFindingLifecycleAuthority|TestDataFeedWriterIsAtomicEncryptedAndOperationIdempotent|TestFieldSyncPushIsTransactionalIdempotentAndServerAuthorized|TestAuthorizedCleanStateCreationIsTransactionalAndIdempotent|TestTask5FixRound2SerializesConcurrentRootCommands|TestTask6ConcurrentIdenticalApprovalReplaysOneExactEffect)$'
  test_compose down --volumes --remove-orphans
  integration_started=false
  assert_no_project_residue "$integration_project"
}

run_negative_transaction_matrix

AVIA_CANONICAL_PREPROD_STATE_DIR="$state_root" \
AVIA_CANONICAL_PREPROD_PROJECT="$project_name" \
AVIA_PREPROD_HTTPS_PORT="$https_port" \
AVIA_PREPROD_WEB_ORIGIN="$web_origin" \
AVIA_PREPROD_SKIP_BUILD="$skip_build" \
  bash "$repository_root/scripts/start-canonical-preprod.sh"
stack_started=true

wait_for_readiness 200 ready
assert_liveness
assert_donor_unavailable
assert_public_provider_admin_unavailable

run_playwright_file tests/e2e/canonical-quick-tunnel-lifecycle.spec.ts "$artifact_root/lifecycle"

before_fingerprint="$(wait_for_database_quiescence "$artifact_root/fingerprint-before.txt")"

compose stop --timeout 30 \
  preprod-gateway preprod-api preprod-worker preprod-auth \
  preprod-web-http preprod-minio preprod-postgres preprod-auth-postgres
compose start \
  preprod-postgres preprod-auth-postgres preprod-minio preprod-web-http
for service in preprod-postgres preprod-auth-postgres preprod-minio preprod-web-http; do
  wait_for_service_health "$service"
done
compose start preprod-auth
wait_for_service_health preprod-auth
compose start preprod-api preprod-worker preprod-gateway
for service in preprod-api preprod-worker preprod-gateway; do
  wait_for_service_health "$service"
done
wait_for_readiness 200 ready
assert_donor_unavailable
assert_public_provider_admin_unavailable
after_fingerprint="$(wait_for_database_quiescence "$artifact_root/fingerprint-after.txt")"
[[ "$after_fingerprint" == "$before_fingerprint" ]] ||
  fail "fingerprint changed across cold restart: $before_fingerprint != $after_fingerprint"

run_playwright_file tests/e2e/canonical-quick-tunnel-panels.spec.ts "$artifact_root/panels-after-restart"

for service in preprod-postgres preprod-auth-postgres preprod-auth preprod-minio; do
  inject_dependency_failure "$service" not_ready 503
done
assert_worker_restart
assert_runtime_log_secrets_absent
assert_donor_unavailable
assert_public_provider_admin_unavailable
wait_for_readiness 200 ready

printf 'Task 8 canonical negative/fault/restart matrix verified locally; fingerprint=%s; donor disabled; cleanup will assert zero residue.\n' "$after_fingerprint"
