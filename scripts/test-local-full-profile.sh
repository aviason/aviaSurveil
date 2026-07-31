#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIRECTORY="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIRECTORY}/.." && pwd)"
COMPOSE_FILE="${REPOSITORY_ROOT}/deploy/local/compose.yaml"
RUNTIME_DIRECTORY="$(mktemp -d /private/tmp/aviasurveil360-plan3-full.XXXXXX)"
AVIA_LOCAL_PROJECT="aviasurveil360-task-plan3-full-$(date -u +%Y%m%d%H%M%S)-$$"
AVIASURVEIL_LOCAL_STATE_DIR="${RUNTIME_DIRECTORY}/local-state"
AVIA_LOCAL_HTTPS_PORT="${AVIA_LOCAL_FULL_HTTPS_PORT:-$((28443 + RANDOM % 10000))}"
PLAYWRIGHT_REPORT="${RUNTIME_DIRECTORY}/playwright-report.json"
SUMMARY_PATH="${RUNTIME_DIRECTORY}/summary.json"
MAILPIT_REPORT="${RUNTIME_DIRECTORY}/mailpit-messages.json"
IDENTITY_SETUP_BINARY="${RUNTIME_DIRECTORY}/identitysetup.test"
SUMMARY_CONTENT_TYPE="application/json"
APPLICATION_ADMIN_USERNAME="local.admin.$(openssl rand -hex 6)@example.test"
APPLICATION_ADMIN_PASSWORD="$(openssl rand -hex 20)Aa1!"
STACK_STARTED="false"

export AVIA_LOCAL_PROJECT AVIASURVEIL_LOCAL_STATE_DIR AVIA_LOCAL_HTTPS_PORT
export AVIA_LOCAL_PUBLIC_ORIGIN="https://localhost:${AVIA_LOCAL_HTTPS_PORT}"
export COMPOSE_PROGRESS=plain

read_runtime_secret() {
  tr -d '\r\n' <"${AVIASURVEIL_LOCAL_STATE_DIR}/secrets/$1"
}

compose() {
  docker compose \
    --project-name "${AVIA_LOCAL_PROJECT}" \
    --file "${COMPOSE_FILE}" \
    --profile full \
    "$@"
}

force_remove_task_owned_residue() {
  local resource_id
  local status=0
  while IFS= read -r resource_id; do
    if [[ -n "${resource_id}" ]] &&
      ! docker rm --force "${resource_id}"; then
      status=1
    fi
  done < <(
    docker ps --all --quiet \
      --filter "label=com.docker.compose.project=${AVIA_LOCAL_PROJECT}"
  )
  while IFS= read -r resource_id; do
    if [[ -n "${resource_id}" ]] &&
      ! docker volume rm "${resource_id}"; then
      status=1
    fi
  done < <(
    docker volume ls --quiet \
      --filter "label=com.docker.compose.project=${AVIA_LOCAL_PROJECT}"
  )
  while IFS= read -r resource_id; do
    if [[ -n "${resource_id}" ]] &&
      ! docker network rm "${resource_id}"; then
      status=1
    fi
  done < <(
    docker network ls --quiet \
      --filter "label=com.docker.compose.project=${AVIA_LOCAL_PROJECT}"
  )
  while IFS= read -r resource_id; do
    if [[ -n "${resource_id}" ]] &&
      [[ "${resource_id}" == "${AVIA_LOCAL_PROJECT}_"* ]] &&
      ! docker volume rm "${resource_id}"; then
      status=1
    fi
  done < <(
    docker volume ls --quiet \
      --filter "name=^${AVIA_LOCAL_PROJECT}_"
  )
  while IFS= read -r resource_id; do
    if [[ -n "${resource_id}" ]] &&
      [[ "${resource_id}" == "${AVIA_LOCAL_PROJECT}_"* ]] &&
      ! docker network rm "${resource_id}"; then
      status=1
    fi
  done < <(
    docker network ls --quiet \
      --filter "name=^${AVIA_LOCAL_PROJECT}_"
  )
  return "${status}"
}

assert_no_task_owned_residue() {
  if docker ps --all --quiet \
    --filter "label=com.docker.compose.project=${AVIA_LOCAL_PROJECT}" |
    grep -q .; then
    echo "task-owned residue: containers remain for ${AVIA_LOCAL_PROJECT}" >&2
    return 1
  fi
  if docker volume ls --quiet \
    --filter "label=com.docker.compose.project=${AVIA_LOCAL_PROJECT}" |
    grep -q .; then
    echo "task-owned residue: volumes remain for ${AVIA_LOCAL_PROJECT}" >&2
    return 1
  fi
  if docker network ls --quiet \
    --filter "label=com.docker.compose.project=${AVIA_LOCAL_PROJECT}" |
    grep -q .; then
    echo "task-owned residue: networks remain for ${AVIA_LOCAL_PROJECT}" >&2
    return 1
  fi
  if docker volume ls --quiet \
    --filter "name=^${AVIA_LOCAL_PROJECT}_" |
    grep -q .; then
    echo "task-owned residue: unlabeled volumes remain for ${AVIA_LOCAL_PROJECT}" >&2
    return 1
  fi
  if docker network ls --quiet \
    --filter "name=^${AVIA_LOCAL_PROJECT}_" |
    grep -q .; then
    echo "task-owned residue: unlabeled networks remain for ${AVIA_LOCAL_PROJECT}" >&2
    return 1
  fi
}

cleanup() {
  local status=$?
  trap - EXIT
  set +e
  if [[ "${STACK_STARTED}" == "true" &&
    -f "${AVIASURVEIL_LOCAL_STATE_DIR}/.compose-project-owner" ]]; then
    "${REPOSITORY_ROOT}/scripts/local-stack.sh" down full
    if [[ $? -ne 0 ]]; then
      echo "normal task-owned Compose cleanup failed; applying exact-label fallback" >&2
    fi
  fi
  force_remove_task_owned_residue
  if [[ $? -ne 0 ]]; then
    status=1
  fi
  assert_no_task_owned_residue
  if [[ $? -ne 0 ]]; then
    status=1
  fi
  rm -rf -- "${RUNTIME_DIRECTORY}"
  exit "${status}"
}
trap cleanup EXIT

"${REPOSITORY_ROOT}/scripts/check-local-image-evidence.sh" full
STACK_STARTED="true"
"${REPOSITORY_ROOT}/scripts/local-stack.sh" up full
"${REPOSITORY_ROOT}/scripts/check-local-runtime.sh"
compose stop worker

KEYCLOAK_BASE_URL="https://localhost:${AVIA_LOCAL_HTTPS_PORT}/identity"
KEYCLOAK_ADMIN_USERNAME="local-bootstrap-admin"
KEYCLOAK_ADMIN_PASSWORD="$(read_runtime_secret keycloak_bootstrap_admin_password)"

RUNTIME_ARCHITECTURE="$(
  docker image inspect aviasurveil360/api:local --format '{{.Architecture}}'
)"
case "${RUNTIME_ARCHITECTURE}" in
  amd64 | arm64) ;;
  *)
    echo "unsupported local API image architecture: ${RUNTIME_ARCHITECTURE}" >&2
    exit 1
    ;;
esac
CGO_ENABLED=0 GOOS=linux GOARCH="${RUNTIME_ARCHITECTURE}" \
  go -C "${REPOSITORY_ROOT}/apps/api" test \
  -c -tags canonicaltest \
  -o "${IDENTITY_SETUP_BINARY}" \
  ./tests/identitysetup
compose run \
  --rm \
  --no-deps \
  --volume "${IDENTITY_SETUP_BINARY}:/tmp/identitysetup.test:ro" \
  --entrypoint /bin/sh \
  api -ec '
  database_password="$(tr -d "\r\n" </run/secrets/app_database_password)"
  export AVIA_TEST_DATABASE_URL="postgres://aviasurveil360:${database_password}@postgres:5432/aviasurveil360?sslmode=disable"
  unset database_password
  exec /tmp/identitysetup.test \
    -test.run "^TestGate0BootstrapHistoricalChecklistForFullProfile$" \
    -test.count=1
' historical-checklist-bootstrap
compose run \
  --rm \
  --no-deps \
  --volume "${IDENTITY_SETUP_BINARY}:/tmp/identitysetup.test:ro" \
  --entrypoint /bin/sh \
  api -ec '
  database_password="$(tr -d "\r\n" </run/secrets/app_database_password)"
  service_client_secret="$(
    tr -d "\r\n" </run/secrets/keycloak_service_client_secret
  )"
  export AVIA_TEST_DATABASE_URL="postgres://aviasurveil360:${database_password}@postgres:5432/aviasurveil360?sslmode=disable"
  export AVIA_OIDC_TEST_ADMIN_USERNAME="$1"
  export AVIA_TEST_OIDC_ISSUER_URL="$2"
  export AVIA_OIDC_TEST_KEYCLOAK_BASE_URL="http://keycloak:8080/identity"
  export AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET="${service_client_secret}"
  unset database_password service_client_secret
  exec /tmp/identitysetup.test \
    -test.run "^TestTask4PrepareOIDCHarnessApplicationAdministrator$" \
    -test.count=1
' identity-setup \
  "${APPLICATION_ADMIN_USERNAME}" \
  "${AVIA_LOCAL_PUBLIC_ORIGIN}/identity/realms/aviasurveil360"

KEYCLOAK_ADMIN_TOKEN="$(
  curl --fail --silent --show-error --insecure \
    --request POST \
    "${KEYCLOAK_BASE_URL}/realms/master/protocol/openid-connect/token" \
    --header "Content-Type: application/x-www-form-urlencoded" \
    --data-urlencode "client_id=admin-cli" \
    --data-urlencode "grant_type=password" \
    --data-urlencode "username=${KEYCLOAK_ADMIN_USERNAME}" \
    --data-urlencode "password=${KEYCLOAK_ADMIN_PASSWORD}" |
    node -e 'let body="";process.stdin.on("data",chunk=>body+=chunk);process.stdin.on("end",()=>{const value=JSON.parse(body);if(!value.access_token)process.exit(1);process.stdout.write(value.access_token);});'
)"
APPLICATION_ADMIN_SUBJECT_ID="$(
  curl --fail --silent --show-error --insecure \
    --get \
    "${KEYCLOAK_BASE_URL}/admin/realms/aviasurveil360/users" \
    --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
    --data-urlencode "email=${APPLICATION_ADMIN_USERNAME}" \
    --data-urlencode "exact=true" |
    node -e 'let body="";process.stdin.on("data",chunk=>body+=chunk);process.stdin.on("end",()=>{const users=JSON.parse(body);if(users.length!==1||!users[0].id)process.exit(1);process.stdout.write(users[0].id);});'
)"
curl --fail --silent --show-error --insecure \
  "${KEYCLOAK_BASE_URL}/admin/realms/aviasurveil360/users/${APPLICATION_ADMIN_SUBJECT_ID}" \
  --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
  --output "${RUNTIME_DIRECTORY}/application-admin.json"
curl --fail --silent --show-error --insecure \
  "${KEYCLOAK_BASE_URL}/admin/realms/aviasurveil360/users/${APPLICATION_ADMIN_SUBJECT_ID}/role-mappings/realm" \
  --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
  --output "${RUNTIME_DIRECTORY}/application-admin-roles.json"
node -e '
  const fs = require("node:fs");
  const user = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
  const roles = JSON.parse(fs.readFileSync(process.argv[2], "utf8"))
    .map(({ name }) => name)
    .filter((name) => [
      "inspector", "leadInspector", "manager", "finance", "gm",
      "executiveDirector", "auditee", "admin",
    ].includes(name))
    .sort();
  if (
    user.enabled !== true ||
    user.email !== process.argv[3] ||
    user.attributes?.organization_id?.length !== 1 ||
    user.attributes.organization_id[0] !== "CAA" ||
    roles.length !== 1 ||
    roles[0] !== "admin"
  ) {
    process.stderr.write("lifecycle-created Admin authority is not exact\n");
    process.exit(1);
  }
' \
  "${RUNTIME_DIRECTORY}/application-admin.json" \
  "${RUNTIME_DIRECTORY}/application-admin-roles.json" \
  "${APPLICATION_ADMIN_USERNAME}"
node -e '
  process.stdout.write(JSON.stringify({
    emailVerified: true,
    requiredActions: ["CONFIGURE_TOTP"],
  }));
' |
  curl --fail --silent --show-error --insecure \
    --request PUT \
    "${KEYCLOAK_BASE_URL}/admin/realms/aviasurveil360/users/${APPLICATION_ADMIN_SUBJECT_ID}" \
    --header "Authorization: Bearer ${KEYCLOAK_ADMIN_TOKEN}" \
    --header "Content-Type: application/json" \
    --data-binary @- \
    --output /dev/null
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
unset KEYCLOAK_ADMIN_TOKEN

compose start worker
compose up --detach --wait worker

export AVIA_E2E_PROFILE=local-full
export AVIA_E2E_BASE_URL="https://localhost:${AVIA_LOCAL_HTTPS_PORT}"
export AVIA_E2E_IGNORE_HTTPS_ERRORS=1
export AVIA_PLAYWRIGHT_OUTPUT_DIR="${RUNTIME_DIRECTORY}/playwright-results"
export PLAYWRIGHT_JSON_OUTPUT_NAME="${PLAYWRIGHT_REPORT}"
export AVIA_LOCAL_FULL_ADMIN_USERNAME="${APPLICATION_ADMIN_USERNAME}"
export AVIA_LOCAL_FULL_ADMIN_PASSWORD="${APPLICATION_ADMIN_PASSWORD}"
export AVIA_LOCAL_FULL_ADMIN_SUBJECT_ID="${APPLICATION_ADMIN_SUBJECT_ID}"
export AVIA_LOCAL_FULL_KEYCLOAK_BASE_URL="${KEYCLOAK_BASE_URL}"
export AVIA_LOCAL_FULL_KEYCLOAK_ADMIN_USERNAME="${KEYCLOAK_ADMIN_USERNAME}"
export AVIA_LOCAL_FULL_KEYCLOAK_ADMIN_PASSWORD="${KEYCLOAK_ADMIN_PASSWORD}"
export AVIA_LOCAL_FULL_COMPOSE_PROJECT="${AVIA_LOCAL_PROJECT}"

(
  cd "${REPOSITORY_ROOT}/apps/web"
  npx playwright test --project=local-full --forbid-only --reporter=json
)

compose exec --no-TTY api \
  wget --quiet --output-document=- http://mailpit:8025/api/v1/messages \
  >"${MAILPIT_REPORT}"
node -e '
  const fs = require("node:fs");
  const payload = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
  const messages = payload.messages ?? payload.Messages;
  if (!Array.isArray(messages)) {
    process.stderr.write("Mailpit response omitted the messages collection\n");
    process.exit(1);
  }
  const proof = process.argv[2];
  const match = messages.find((message) =>
    [message.Subject, message.subject, message.Snippet, message.snippet]
      .filter((value) => typeof value === "string")
      .some((value) => value.includes(proof))
  );
  if (!match) {
    process.stderr.write(`Mailpit omitted the expected application delivery: ${proof}\n`);
    process.exit(1);
  }
  process.stdout.write(
    `Mailpit accepted ${messages.length} message(s), including ${proof}\n`
  );
' "${MAILPIT_REPORT}" "Plan 3 full-profile SMTP delivery"

compose restart worker
compose up --detach --wait worker
"${REPOSITORY_ROOT}/scripts/check-local-runtime.sh"

compose logs --no-color >"${RUNTIME_DIRECTORY}/docker-runtime.log"
for secret_value in \
  "${APPLICATION_ADMIN_PASSWORD}" \
  "${KEYCLOAK_ADMIN_PASSWORD}"; do
  if rg --fixed-strings --quiet -- "${secret_value}" \
    "${PLAYWRIGHT_REPORT}" \
    "${RUNTIME_DIRECTORY}/playwright-results" \
    "${MAILPIT_REPORT}" \
    "${RUNTIME_DIRECTORY}/docker-runtime.log"; then
    echo "generated secret found in full-profile artifacts" >&2
    exit 1
  fi
done

node -e '
  const fs = require("node:fs");
  const report = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
  const tests = [];
  const visit = (suite) => {
    for (const spec of suite.specs ?? []) {
      for (const test of spec.tests ?? []) tests.push(test);
    }
    for (const child of suite.suites ?? []) visit(child);
  };
  for (const suite of report.suites ?? []) visit(suite);
  const skipped = tests.filter((test) =>
    test.status === "skipped" ||
    (test.results ?? []).some((result) => result.status === "skipped")
  ).length;
  const summary = {
    contentType: process.argv[3],
    profile: "full",
    composeProject: process.env.AVIA_LOCAL_PROJECT,
    expectedDirectLoads: 86,
    expectedScenarioFamilies: 10,
    tests: tests.length,
    skipped,
    status: skipped === 0 ? "verified locally" : "blocked",
  };
  fs.writeFileSync(process.argv[2], `${JSON.stringify(summary, null, 2)}\n`);
  process.stdout.write(`${JSON.stringify(summary)}\n`);
  if (skipped !== 0 || tests.length === 0) process.exit(1);
' "${PLAYWRIGHT_REPORT}" "${SUMMARY_PATH}" "${SUMMARY_CONTENT_TYPE}"

echo "Clean full profile verified locally; cleanup will assert zero task-owned residue."
