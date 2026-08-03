#!/usr/bin/env bash
set -euo pipefail

# This is a deliberately narrow, connected qualification harness. It cannot
# create a base profile: the caller must supply an exact, private predecessor
# result and its private retained-target handoff. AGA operations are confined
# to the immutable PostgreSQL overlay and every exit tears down the disposable
# namespace rather than selectively repairing it.

umask 077

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_file="$repository_root/deploy/local/compose.yaml"
project_name="aviasurveil360-local-preprod"
package_file="${AVIA_AGA_DEMO_PACKAGE_FILE:-}"
base_result_file="${AVIA_AGA_DEMO_BASE_RESULT_FILE:-}"
load_authorization_file="${AVIA_AGA_DEMO_LOAD_AUTHORIZATION_FILE:-}"
cleanup_authorization_file="${AVIA_AGA_DEMO_CLEANUP_AUTHORIZATION_FILE:-}"
configuration_file="${AVIA_AGA_DEMO_CONFIG_FILE:-}"
control_store_directory="${AVIA_AGA_DEMO_CONTROL_STORE_DIR:-}"
state_directory="${AVIA_PREPROD_STATE_DIR:-}"
evidence_directory="${AVIA_AGA_DEMO_EVIDENCE_DIRECTORY:-}"
handoff_file="${AVIA_AGA_DEMO_HANDOFF_FILE:-$(dirname "${AVIA_AGA_DEMO_BASE_RESULT_FILE:-/nonexistent}")/handoff.json}"
cleanup_needed="false"
web_pid=""
runtime_directory=""

fail() {
  printf 'test-aga-candidate-preprod-demo-connected: %s\n' "$*" >&2
  exit 1
}

compose_command() {
  AVIA_PREPROD_STATE_DIR="$state_directory" docker compose \
    --project-name "$project_name" \
    --file "$compose_file" \
    --profile local-preprod-loader \
    --profile aga-candidate-demo \
    --profile aga-candidate-demo-oidc-fixture \
    --profile preproddemo \
    "$@"
}

project_residue() {
  local containers volumes networks
  containers="$(docker ps --all --filter "label=com.docker.compose.project=$project_name" --format '{{.ID}}')"
  volumes="$(docker volume ls --filter "label=com.docker.compose.project=$project_name" --format '{{.Name}}')"
  networks="$(docker network ls --filter "label=com.docker.compose.project=$project_name" --format '{{.Name}}')"
  [[ -n "$containers$volumes$networks" ]]
}

cleanup() {
  local status=$?
  trap - EXIT HUP INT TERM
  if [[ -n "$web_pid" ]] && kill -0 "$web_pid" 2>/dev/null; then
    kill "$web_pid" 2>/dev/null || true
    wait "$web_pid" 2>/dev/null || true
  fi
  if [[ "$cleanup_needed" == "true" ]]; then
    compose_command down --volumes --remove-orphans || status=1
    if project_residue; then
      printf 'test-aga-candidate-preprod-demo-connected: task-owned Compose residue remains\n' >&2
      status=1
    fi
  fi
  if [[ -n "$runtime_directory" && -d "$runtime_directory" ]]; then
    rm -rf -- "$runtime_directory"
  fi
  exit "$status"
}
trap cleanup EXIT HUP INT TERM

for name in package_file base_result_file load_authorization_file cleanup_authorization_file configuration_file control_store_directory state_directory evidence_directory handoff_file; do
  value="${!name}"
  [[ -n "$value" && "$value" = /* ]] || fail "$name must be an absolute path"
done
for path in "$package_file" "$base_result_file" "$load_authorization_file" "$cleanup_authorization_file" "$configuration_file" "$handoff_file"; do
  [[ -f "$path" && ! -L "$path" ]] || fail "required private file is missing or a symlink"
done
[[ -d "$control_store_directory" && ! -L "$control_store_directory" ]] || fail "control store is missing or a symlink"
[[ -d "$state_directory" && ! -L "$state_directory" ]] || fail "state directory is missing or a symlink"
[[ -d "$evidence_directory" && ! -L "$evidence_directory" ]] || fail "evidence directory is missing or a symlink"
[[ "$(find "$evidence_directory" -mindepth 1 -maxdepth 1 -print -quit)" == "" ]] || fail "evidence directory must be empty"
for path in "$base_result_file" "$load_authorization_file" "$cleanup_authorization_file" "$configuration_file" "$handoff_file"; do
  [[ "$(stat -f '%Lp' "$path")" == "600" ]] || fail "private input does not have mode 0600"
done
[[ "$(stat -f '%Lp' "$evidence_directory")" == "700" ]] || fail "evidence directory does not have mode 0700"

command -v docker >/dev/null 2>&1 || fail "docker is required"
command -v jq >/dev/null 2>&1 || fail "jq is required"
docker info >/dev/null 2>&1 || fail "docker daemon is unavailable"
project_residue || fail "exact disposable predecessor target is not running"
cleanup_needed="true"
runtime_directory="$(mktemp -d "${TMPDIR:-/tmp}/avia-aga-demo-browser.XXXXXX")"
chmod 0700 "$runtime_directory"

compose_command build preprod-aga-candidate-demo-loader preprod-aga-demo-role-provisioner preprod-aga-demo-oidc-fixture
code_digest="$(
  docker run --rm --network none --entrypoint sha256sum \
    aviasurveil360/preprod-aga-candidate-demo-loader:local \
    /app/preprod-aga-candidate-demo-loader |
    awk '{print "sha256:" $1}'
)"
contract_digest="$(
  node --input-type=module - \
    "$repository_root/apps/api/internal/preproddata/agacandidatedemo/contract.go" \
    "$repository_root/api/openapi/source/paths/platform.json" \
    "$repository_root/api/openapi/source/schemas/platform.json" <<'NODE'
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";

const digest = createHash("sha256");
for (const path of process.argv.slice(2)) digest.update(readFileSync(path));
process.stdout.write(`sha256:${digest.digest("hex")}`);
NODE
)"
[[ "$code_digest" =~ ^sha256:[a-f0-9]{64}$ &&
  "$contract_digest" =~ ^sha256:[a-f0-9]{64}$ ]] ||
  fail "tagged code or contract digest is invalid"
[[ "$(jq -er '.codeDigest' "$configuration_file")" == "$code_digest" ]] ||
  fail "tagged loader code digest mismatch"
[[ "$(jq -er '.contractDigest' "$configuration_file")" == "$contract_digest" ]] ||
  fail "AGA demo contract digest mismatch"
compose_command run --rm preprod-aga-demo-role-provisioner \
  | tee "$evidence_directory/role-provision.log"

run_id="$(jq -er '.runId' "$base_result_file")"
base_target_digest="$(jq -er '.targetFingerprintDigest' "$base_result_file")"
[[ "$(jq -r '.outcome' "$base_result_file")" == "SUCCEEDED" ]] || fail "base result is not successful"
[[ "$(jq -r '.disposable' "$base_result_file")" == "true" ]] || fail "base result is not disposable"
[[ "$run_id" =~ ^[a-z0-9][a-z0-9-]{5,95}$ ]] || fail "base run id is invalid"
[[ "$base_target_digest" =~ ^sha256:[a-f0-9]{64}$ ]] || fail "base target digest is invalid"

AVIA_PREPROD_RUN_ID="$run_id" compose_command run --rm preprod-aga-demo-oidc-fixture \
  | tee "$evidence_directory/oidc-qualification-fixture.log"

AVIA_AGA_BASE_RESULT_FILE="$base_result_file" \
AVIA_AGA_HANDOFF_FILE="$handoff_file" \
AVIA_AGA_CONFIG_FILE="$configuration_file" \
AVIA_AGA_CONTROL_STORE_DIRECTORY="$control_store_directory" \
AVIA_AGA_STATE_DIRECTORY="$state_directory" \
  node "$repository_root/scripts/validate-aga-predecessor-handoff.mjs"

snapshot() {
  local output_path="$1"
  local database_digest keycloak_users mailpit_digest minio_digest
  database_digest="$(
    compose_command exec --no-TTY preprod-postgres \
      psql --username aviasurveil360_preprod_loader \
      --dbname aviasurveil360_local_preprod --tuples-only --no-align \
      --command "SELECT schemaname || '.' || relname || ':' || n_tup_ins || ':' || n_tup_upd || ':' || n_tup_del FROM pg_stat_user_tables WHERE schemaname <> 'preprod_aga_demo' ORDER BY schemaname, relname" |
      shasum -a 256 | awk '{print "sha256:" $1}'
  )"
  keycloak_users="$(
    compose_command exec --no-TTY preprod-keycloak-postgres \
      psql --username keycloak_preprod --dbname keycloak_local_preprod \
      --tuples-only --no-align --command 'SELECT count(*) FROM user_entity' |
      tr -d '[:space:]'
  )"
  [[ "$keycloak_users" =~ ^[0-9]+$ ]] || fail "Keycloak user snapshot is invalid"
  mailpit_digest="$(
    compose_command exec --no-TTY preprod-mailpit \
      wget --quiet --output-document=- http://127.0.0.1:8025/api/v1/messages |
      shasum -a 256 | awk '{print "sha256:" $1}'
  )"
  minio_digest="$(
    compose_command exec --no-TTY preprod-minio \
      mc ls --recursive --versions local-preprod/aviasurveil360-local-preprod |
      shasum -a 256 | awk '{print "sha256:" $1}'
  )"
  [[ "$database_digest" =~ ^sha256:[a-f0-9]{64}$ && "$mailpit_digest" =~ ^sha256:[a-f0-9]{64}$ && "$minio_digest" =~ ^sha256:[a-f0-9]{64}$ ]] || fail "forbidden-system snapshot digest is invalid"
  AVIA_AGA_SNAPSHOT_PATH="$output_path" \
  AVIA_AGA_DATABASE_DIGEST="$database_digest" \
  AVIA_AGA_KEYCLOAK_USERS="$keycloak_users" \
  AVIA_AGA_MAILPIT_DIGEST="$mailpit_digest" \
  AVIA_AGA_MINIO_DIGEST="$minio_digest" \
    node --input-type=module <<'NODE'
import { writeFileSync } from 'node:fs';
writeFileSync(process.env.AVIA_AGA_SNAPSHOT_PATH, `${JSON.stringify({
  schemaVersion: 'preprod-aga-candidate-demo-forbidden-snapshot/v1',
  databaseNonOverlayStatsDigest: process.env.AVIA_AGA_DATABASE_DIGEST,
  keycloakUserCount: Number(process.env.AVIA_AGA_KEYCLOAK_USERS),
  mailpitStateDigest: process.env.AVIA_AGA_MAILPIT_DIGEST,
  minioObjectVersionDigest: process.env.AVIA_AGA_MINIO_DIGEST,
})}\n`, { flag: 'wx', mode: 0o600 });
NODE
}

safe_session_diagnostic() {
  local state callback_digest stored_digest callback_count stored_count request_digest request_count request_seen request_matched api_diagnostic
  state="$(
    compose_command exec --no-TTY preprod-postgres \
      psql --username aviasurveil360_preprod_loader \
      --dbname aviasurveil360_local_preprod --tuples-only --no-align \
      --command "
        SELECT session.authority_state || '|' || sync.drift_state || '|' ||
               COALESCE((
                 SELECT event.details->>'reasonCode'
                 FROM audit_events event
                 WHERE event.entity_type = 'SESSION'
                   AND event.entity_id = session.id
                   AND event.action IN (
                     'SESSION_AUTHORITY_DENIED',
                     'SESSION_AUTHORITY_REVOCATION_PENDING'
                   )
                 ORDER BY event.occurred_at DESC
                 LIMIT 1
               ), 'NONE') || '|' ||
               CASE WHEN session.revoked_at IS NULL THEN 'NOT_REVOKED' ELSE 'REVOKED' END || '|' ||
               CASE WHEN CURRENT_TIMESTAMP < session.expires_at THEN 'IDLE_VALID' ELSE 'IDLE_EXPIRED' END || '|' ||
               CASE WHEN CURRENT_TIMESTAMP < session.absolute_expires_at THEN 'ABSOLUTE_VALID' ELSE 'ABSOLUTE_EXPIRED' END
        FROM session_references session
        JOIN desired_membership_sync sync
          ON sync.membership_id = session.membership_id
        ORDER BY session.created_at DESC
        LIMIT 1
      " 2>/dev/null | tr -d '[:space:]'
  )" || state=""
  case "$state" in
    ACTIVE\|IN_SYNC\|NONE\|NOT_REVOKED\|IDLE_VALID\|ABSOLUTE_VALID)
      printf 'test-aga-candidate-preprod-demo-connected: server session diagnostic=active-valid\n' >&2
      ;;
    ACTIVE\|*)
      printf 'test-aga-candidate-preprod-demo-connected: server session diagnostic=active-invalid\n' >&2
      ;;
    REVOCATION_PENDING\|*\|*)
      printf 'test-aga-candidate-preprod-demo-connected: server session diagnostic=revocation-pending\n' >&2
      ;;
    DENIED_STALE_AUTHORITY\|*\|*)
      printf 'test-aga-candidate-preprod-demo-connected: server session diagnostic=denied\n' >&2
      ;;
    *)
      printf 'test-aga-candidate-preprod-demo-connected: server session diagnostic=unclassified\n' >&2
      ;;
  esac
  callback_digest="$(jq -er '.callback' "$runtime_directory/browser-session-digests.json" 2>/dev/null || true)"
  stored_digest="$(jq -er '.stored' "$runtime_directory/browser-session-digests.json" 2>/dev/null || true)"
  if [[ "$callback_digest" =~ ^[a-f0-9]{64}$ ]]; then
    callback_count="$(
      compose_command exec --no-TTY preprod-postgres \
        psql --username aviasurveil360_preprod_loader \
        --dbname aviasurveil360_local_preprod --tuples-only --no-align \
        --command "SELECT COUNT(*) FROM session_references WHERE session_token_hash = '$callback_digest'" \
        2>/dev/null | tr -d '[:space:]'
    )" || callback_count=""
  else
    callback_count=""
  fi
  if [[ "$stored_digest" =~ ^[a-f0-9]{64}$ ]]; then
    stored_count="$(
      compose_command exec --no-TTY preprod-postgres \
        psql --username aviasurveil360_preprod_loader \
        --dbname aviasurveil360_local_preprod --tuples-only --no-align \
        --command "SELECT COUNT(*) FROM session_references WHERE session_token_hash = '$stored_digest'" \
        2>/dev/null | tr -d '[:space:]'
    )" || stored_count=""
  else
    stored_count=""
  fi
  case "$callback_count" in
    1) printf 'test-aga-candidate-preprod-demo-connected: browser callback digest diagnostic=match\n' >&2 ;;
    0) printf 'test-aga-candidate-preprod-demo-connected: browser callback digest diagnostic=mismatch\n' >&2 ;;
    *) printf 'test-aga-candidate-preprod-demo-connected: browser callback digest diagnostic=unclassified\n' >&2 ;;
  esac
  case "$stored_count" in
    1) printf 'test-aga-candidate-preprod-demo-connected: browser session digest diagnostic=match\n' >&2 ;;
    0) printf 'test-aga-candidate-preprod-demo-connected: browser session digest diagnostic=mismatch\n' >&2 ;;
    *) printf 'test-aga-candidate-preprod-demo-connected: browser session digest diagnostic=unclassified\n' >&2 ;;
  esac
  request_seen="false"
  request_matched="false"
  while IFS= read -r request_digest; do
    [[ "$request_digest" =~ ^[a-f0-9]{64}$ ]] || continue
    request_seen="true"
    request_count="$(
      compose_command exec --no-TTY preprod-postgres \
        psql --username aviasurveil360_preprod_loader \
        --dbname aviasurveil360_local_preprod --tuples-only --no-align \
        --command "SELECT COUNT(*) FROM session_references WHERE session_token_hash = '$request_digest'" \
        2>/dev/null | tr -d '[:space:]'
    )" || request_count=""
    if [[ "$request_count" == "1" ]]; then
      request_matched="true"
    fi
  done < <(jq -r '.requests[]?' "$runtime_directory/browser-session-digests.json" 2>/dev/null || true)
  if [[ "$request_matched" == "true" ]]; then
    printf 'test-aga-candidate-preprod-demo-connected: browser request digest diagnostic=match\n' >&2
  elif [[ "$request_seen" == "true" ]]; then
    printf 'test-aga-candidate-preprod-demo-connected: browser request digest diagnostic=mismatch\n' >&2
  else
    printf 'test-aga-candidate-preprod-demo-connected: browser request digest diagnostic=unclassified\n' >&2
  fi
  api_diagnostic="$(
    compose_command logs --no-color preprod-aga-demo-api 2>/dev/null |
      sed -n \
        -e 's/.*"msg":"browser session authentication rejected".*"diagnostic":"\([^"]*\)".*/\1/p' \
        -e 's/.*browser session authentication rejected.*diagnostic=\([^ ]*\).*/\1/p' |
      tail -n 1
  )" || api_diagnostic=""
  case "$api_diagnostic" in
    missing-token|session-not-found|session-read|expired-or-revoked|unbound-authority|revocation-pending|missing-membership|authority-read|authority-mismatch|invalid-observation-time|provider-unavailable|provider-drift|observation-refresh|identity-reference|profile-read|invalid-role|idle-refresh|transaction|unauthenticated|context-expired|postgres-privilege-public-schema|postgres-privilege-trace-context|postgres-privilege-audit-sequence|postgres-privilege-session-references|postgres-privilege-identity-references|postgres-privilege-user-profiles|postgres-privilege-membership-versions|postgres-privilege-membership-sync|postgres-privilege-login-states|postgres-privilege-audit-events|postgres-insufficient-privilege|postgres-integrity|postgres-connection|postgres|department-assignment|internal)
      printf 'test-aga-candidate-preprod-demo-connected: API authentication diagnostic=%s\n' "$api_diagnostic" >&2
      ;;
    *)
      printf 'test-aga-candidate-preprod-demo-connected: API authentication diagnostic=unclassified\n' >&2
      ;;
  esac
}

load_environment=(
  AVIA_AGA_DEMO_CONFIG_FILE="$configuration_file"
  AVIA_AGA_DEMO_PACKAGE_FILE="$package_file"
  AVIA_AGA_DEMO_BASE_EVIDENCE_FILE="$base_result_file"
  AVIA_AGA_DEMO_CONTROL_STORE_DIR="$control_store_directory"
  AVIA_PREPROD_STATE_DIR="$state_directory"
)

snapshot "$evidence_directory/forbidden-before.json"
env "${load_environment[@]}" \
  AVIA_AGA_DEMO_AUTHORIZATION_FILE="$load_authorization_file" \
  "$repository_root/scripts/load-aga-candidate-demo.sh" prepare-aga-demo | tee "$evidence_directory/prepare.log"
env "${load_environment[@]}" \
  AVIA_AGA_DEMO_AUTHORIZATION_FILE="$load_authorization_file" \
  "$repository_root/scripts/load-aga-candidate-demo.sh" verify-aga-demo-authorization | tee "$evidence_directory/authorization.log"
env "${load_environment[@]}" \
  AVIA_AGA_DEMO_AUTHORIZATION_FILE="$load_authorization_file" \
  "$repository_root/scripts/load-aga-candidate-demo.sh" run-aga-demo | tee "$evidence_directory/load.log"
env "${load_environment[@]}" \
  "$repository_root/scripts/load-aga-candidate-demo.sh" verify-aga-demo | tee "$evidence_directory/verify.log"
compose_command run --rm preprod-aga-demo-role-provisioner verify-least-privilege \
  | tee "$evidence_directory/least-privilege.log"
compose_command run --rm preprod-aga-demo-role-provisioner verify-sealed-projection \
  | tee "$evidence_directory/sealed-projection.log"
snapshot "$evidence_directory/forbidden-after.json"
cmp "$evidence_directory/forbidden-before.json" "$evidence_directory/forbidden-after.json" >/dev/null || fail "forbidden-system delta detected"

account_rows="$(
  compose_command exec --no-TTY preprod-postgres \
    psql --username aviasurveil360_preprod_loader \
    --dbname aviasurveil360_local_preprod --tuples-only --no-align \
    --field-separator '|' \
    --command "SELECT attributes->>'role', attributes->>'email' FROM preprod_loader.scenario_records WHERE run_id = '$run_id' AND family = 'providerAccounts' ORDER BY attributes->>'role', attributes->>'email'"
)"
[[ "$(printf '%s\n' "$account_rows" | sed '/^[[:space:]]*$/d' | wc -l | tr -d '[:space:]')" == "9" ]] || fail "OIDC qualification account count mismatch"
admin_username=""
inspector_username=""
lead_inspector_username=""
manager_username=""
finance_username=""
gm_username=""
executive_director_username=""
auditee_username=""
while IFS='|' read -r role email; do
  [[ "$email" == *@synthetic.invalid ]] || fail "OIDC qualification email boundary mismatch"
  case "$role" in
    admin) admin_username="$email" ;;
    inspector) inspector_username="$email" ;;
    leadInspector) lead_inspector_username="$email" ;;
    manager) manager_username="$email" ;;
    finance) finance_username="$email" ;;
    gm) gm_username="$email" ;;
    executiveDirector) executive_director_username="$email" ;;
    auditee) [[ -n "$auditee_username" ]] || auditee_username="$email" ;;
    *) fail "OIDC qualification role boundary mismatch" ;;
  esac
done <<EOF
$account_rows
EOF
for username in "$admin_username" "$inspector_username" "$lead_inspector_username" "$manager_username" "$finance_username" "$gm_username" "$executive_director_username" "$auditee_username"; do
  [[ -n "$username" ]] || fail "OIDC qualification role matrix is incomplete"
done
unset account_rows role email

compose_command up --detach --build --wait preprod-aga-demo-api
(
  cd "$repository_root/apps/web"
  VITE_AVIA_DISABLE_BROWSER_TELEMETRY=1 \
    npm run build:http
  AVIA_BUILD_PROFILE=http \
  AVIA_HTTP_API_TARGET="http://127.0.0.1:${AVIA_PREPROD_AGA_API_PORT:-58081}" \
    ./node_modules/.bin/vite preview --outDir dist/http \
      --host 127.0.0.1 --port "${AVIA_PREPROD_AGA_WEB_PORT:-4174}" --strictPort
) >"$runtime_directory/vite.log" 2>&1 &
web_pid="$!"
web_ready="false"
for _ in $(seq 1 60); do
  if curl --fail --silent --output /dev/null "http://127.0.0.1:${AVIA_PREPROD_AGA_WEB_PORT:-4174}"; then
    web_ready="true"
    break
  fi
  sleep 0.25
done
[[ "$web_ready" == "true" ]] || fail "ordinary React HTTP artifact did not become ready"

browser_environment=(
  AVIA_E2E_BASE_URL="http://127.0.0.1:${AVIA_PREPROD_AGA_WEB_PORT:-4174}"
  AVIA_PREPROD_AGA_OIDC_HOST="${AVIA_PREPROD_AGA_OIDC_HOST:-aga-preprod.test}"
  AVIA_PLAYWRIGHT_OUTPUT_DIR="$runtime_directory/playwright-output"
  AVIA_AGA_BROWSER_PHASE_FILE="$runtime_directory/browser-phase.log"
  AVIA_AGA_BROWSER_SESSION_DIGEST_FILE="$runtime_directory/browser-session-digests.json"
  AVIA_AGA_OIDC_PASSWORD="$(tr -d '\r\n' <"$state_directory/secrets/preprod_aga_demo_oidc_qualification_password")"
  AVIA_AGA_OIDC_ADMIN_USERNAME="$admin_username"
  AVIA_AGA_OIDC_INSPECTOR_USERNAME="$inspector_username"
  AVIA_AGA_OIDC_LEAD_INSPECTOR_USERNAME="$lead_inspector_username"
  AVIA_AGA_OIDC_MANAGER_USERNAME="$manager_username"
  AVIA_AGA_OIDC_FINANCE_USERNAME="$finance_username"
  AVIA_AGA_OIDC_GM_USERNAME="$gm_username"
  AVIA_AGA_OIDC_EXECUTIVE_DIRECTOR_USERNAME="$executive_director_username"
  AVIA_AGA_OIDC_AUDITEE_USERNAME="$auditee_username"
)
install -m 0600 /dev/null "$runtime_directory/browser-phase.log"
install -m 0600 /dev/null "$runtime_directory/browser-session-digests.json"
env "${browser_environment[@]}" npm --prefix "$repository_root/apps/web" run test:e2e:aga-preprod -- --list \
  | tee "$evidence_directory/playwright-list.log"
if env "${browser_environment[@]}" npm --prefix "$repository_root/apps/web" run test:e2e:aga-preprod \
  >"$runtime_directory/playwright.log" 2>&1; then
  printf 'AGA candidate demo isolated-browser matrix passed: tests=10 viewports=1440x900,1024x768,390x844 retainedMedia=0\n' \
    >"$evidence_directory/playwright.log"
else
  safe_phase="$(tail -n 1 "$runtime_directory/browser-phase.log" 2>/dev/null || true)"
  if [[ "$safe_phase" =~ ^(login-gate-visible|provider-page-open|provider-username-filled|provider-password-filled|provider-submit-complete|oidc-provider-error|oidc-provider-retained|oidc-callback-invalid-state|oidc-callback-invalid-token|oidc-callback-stale-authority|oidc-callback-session-failure|oidc-callback-other-client-error|oidc-callback-server-error|oidc-callback-success-retained|oidc-web-return-mismatch|oidc-unexpected-location|oidc-cookie-pair-present|oidc-cookie-pair-incomplete|oidc-callback-complete|anonymous-neutral-verified|denied-login-complete|denied-session-cookie-sent|denied-session-cookie-not-sent|denied-session-active-before|denied-session-lost-before|denied-capability-requested|denied-capability-received|denied-neutral-verified|denied-session-active-after|denied-session-lost-after|denied-logout-complete|denied-stale-session-verified|mutation-login-complete|mutation-methods-verified|mutation-routes-verified|mutation-logout-complete|admin-login-complete|logout-csrf-cookie-missing|logout-csrf-rejected|logout-session-missing|logout-server-failure|logout-unexpected-status|admin-capability-requested|admin-capability-verified|admin-route-open|admin-panel-visible|admin-summary-visible|admin-forms-read|admin-question-slices-read|admin-browser-storage-clear|admin-viewports-verified|admin-telemetry-silent|admin-console-api-resource-error|admin-console-auth-resource-error|admin-console-vite-resource-error|admin-console-asset-resource-error|admin-console-other-web-resource-error|admin-console-external-resource-error|admin-console-unknown-resource-error|admin-console-react-error|admin-console-csp-error|admin-console-other-error|admin-console-clean|admin-failed-requests-clean|admin-runtime-clean|admin-logout-request-complete|admin-logout-session-revoked|admin-logout-ui-cleared|admin-logout-history-clean|admin-logout-verified)$ ]]; then
    printf 'test-aga-candidate-preprod-demo-connected: isolated-browser qualification failed at safe phase=%s\n' "$safe_phase" >&2
  else
    printf 'test-aga-candidate-preprod-demo-connected: isolated-browser qualification failed at safe phase=none\n' >&2
  fi
  safe_session_diagnostic
  fail "isolated-browser qualification failed; private task-owned output was not retained"
fi
unset browser_environment AVIA_AGA_OIDC_PASSWORD
kill "$web_pid" 2>/dev/null || true
wait "$web_pid" 2>/dev/null || true
web_pid=""
compose_command stop preprod-aga-demo-api >/dev/null

env "${load_environment[@]}" \
  AVIA_AGA_DEMO_AUTHORIZATION_FILE="$cleanup_authorization_file" \
  "$repository_root/scripts/load-aga-candidate-demo.sh" cleanup-aga-demo | tee "$evidence_directory/cleanup.log"
if env "${load_environment[@]}" \
  AVIA_AGA_DEMO_AUTHORIZATION_FILE="$load_authorization_file" \
  "$repository_root/scripts/load-aga-candidate-demo.sh" run-aga-demo >"$evidence_directory/replay.log" 2>&1; then
  fail "load replay after cleanup tombstone unexpectedly succeeded"
fi
grep -Fq 'non-replayable' "$evidence_directory/replay.log" || fail "replay rejection did not prove cleanup tombstone"

printf '%s\n' 'candidate-only' 'release pending' 'production-ready: not established' \
  >"$evidence_directory/labels.txt"
compose_command down --volumes --remove-orphans
project_residue && fail "task-owned Compose residue remains after whole-namespace cleanup"
cleanup_needed="false"
printf 'Whole disposable namespace cleanup verified: database=0 keycloak=0 mailpit=0 minio=0 queue=0 process=0 browser-cache=0\n' \
  >"$evidence_directory/namespace-cleanup.log"
chmod 0600 "$evidence_directory"/*
printf 'test-aga-candidate-preprod-demo-connected: verified local disposable overlay and whole-namespace cleanup\n'
