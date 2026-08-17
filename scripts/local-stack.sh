#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIRECTORY="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIRECTORY}/.." && pwd)"
COMPOSE_FILE="${REPOSITORY_ROOT}/deploy/local/compose.yaml"
COMMAND="${1:-status}"
PROFILE="${2:-${AVIA_LOCAL_PROFILE:-full}}"
LOCAL_TARGET="${AVIA_LOCAL_TARGET:-namibia/demo}"

case "${COMMAND}" in
  up | down | status | logs | check) ;;
  *)
    echo "usage: $0 {up|down|status|logs|check} [demo|full|test|recovery|tools]" >&2
    exit 64
    ;;
esac
case "${PROFILE}" in
  demo | full | test | recovery | tools) ;;
  *)
    echo "unsupported local profile: ${PROFILE}" >&2
    exit 64
    ;;
esac
case "${LOCAL_TARGET}" in
  namibia/dev | namibia/demo) ;;
  *)
    echo "AVIA_LOCAL_TARGET must be one exact local target: namibia/dev or namibia/demo" >&2
    exit 64
    ;;
esac
LOCAL_ENVIRONMENT="${LOCAL_TARGET##*/}"
LOCAL_MANIFEST_PREFIX="${AVIA_LOCAL_MANIFEST_PREFIX:-${LOCAL_ENVIRONMENT}}"
if [[ "${LOCAL_ENVIRONMENT}" == "dev" ]]; then
  LOCAL_RUNTIME_ENVIRONMENT="development"
  LOCAL_AUTH_ENVIRONMENT="dev"
  LOCAL_AUTH_PROFILE="${AVIA_LOCAL_AUTH_PROFILE:-standalone}"
  LOCAL_SERVER_MANAGED_CORS="true"
  LOCAL_SCANNER_MODE="${AVIA_LOCAL_SCANNER_MODE:-deterministic-test}"
else
  LOCAL_RUNTIME_ENVIRONMENT="demo"
  LOCAL_AUTH_ENVIRONMENT="demo"
  LOCAL_AUTH_PROFILE="${AVIA_LOCAL_AUTH_PROFILE:-first-party-demo}"
  # Demo must omit this development-only bypass entirely. An explicit
  # false value is still a forbidden cloud-environment override.
  LOCAL_SERVER_MANAGED_CORS=""
  LOCAL_SCANNER_MODE="${AVIA_LOCAL_SCANNER_MODE:-disabled}"
fi
LOCAL_OIDC_CLIENT_ID="${AVIA_LOCAL_OIDC_CLIENT_ID:-aviasurveil360-namibia-${LOCAL_ENVIRONMENT}-web}"
LOCAL_SIGNING_KEY_ID="${AVIA_LOCAL_SIGNING_KEY_ID:-avia-namibia-${LOCAL_ENVIRONMENT}-auth-signing-2026}"
export AVIA_LOCAL_TARGET LOCAL_ENVIRONMENT LOCAL_RUNTIME_ENVIRONMENT LOCAL_AUTH_ENVIRONMENT LOCAL_MANIFEST_PREFIX LOCAL_AUTH_PROFILE LOCAL_SERVER_MANAGED_CORS LOCAL_SCANNER_MODE LOCAL_OIDC_CLIENT_ID LOCAL_SIGNING_KEY_ID

if [[ -z "${AVIA_LOCAL_PROJECT:-}" ]]; then
  if [[ "${COMMAND}" != "up" ]]; then
    echo "AVIA_LOCAL_PROJECT is required for ${COMMAND}; refusing an ambiguous stack target" >&2
    exit 64
  fi
  AVIA_LOCAL_PROJECT="aviasurveil360-task-$(date -u +%Y%m%d%H%M%S)-$$"
fi
if [[ ! "${AVIA_LOCAL_PROJECT}" =~ ^aviasurveil360-task-[a-z0-9][a-z0-9-]*$ ]]; then
  echo "AVIA_LOCAL_PROJECT must be a unique aviasurveil360-task-* name" >&2
  exit 64
fi
export AVIA_LOCAL_PROJECT
export AVIA_LOCAL_PROFILE="${PROFILE}"

DEFAULT_STATE_PARENT="${REPOSITORY_ROOT}/.local/aviasurveil360/projects"
AVIASURVEIL_LOCAL_STATE_DIR="${AVIASURVEIL_LOCAL_STATE_DIR:-${DEFAULT_STATE_PARENT}/${AVIA_LOCAL_PROJECT}}"
if [[ "${AVIASURVEIL_LOCAL_STATE_DIR}" != /* ]]; then
  echo "AVIASURVEIL_LOCAL_STATE_DIR must be an absolute task-owned path" >&2
  exit 64
fi
export AVIASURVEIL_LOCAL_STATE_DIR

assert_local_port() {
  local name="$1" value="${2:-}"
  if [[ ! "$value" =~ ^[0-9]+$ ]] || (( value < 1024 || value > 65535 )); then
    echo "${name} must be an explicit numeric loopback port between 1024 and 65535" >&2
    exit 64
  fi
  python3 -c 'import socket,sys; sock=socket.socket(); sock.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1); sock.bind(("127.0.0.1",int(sys.argv[1]))); sock.close()' "$value" || {
    if docker ps --filter "label=com.docker.compose.project=${AVIA_LOCAL_PROJECT}" --format '{{.Ports}}' | grep -q ":${value}->"; then return 0; fi
    echo "${name} is not available on loopback: ${value}" >&2
    exit 64
  }
}

if [[ "${AVIA_LOCAL_STRICT_PORTS:-0}" == "1" && "${COMMAND}" == "up" ]]; then
  for port_name in AVIA_LOCAL_HTTPS_PORT AVIA_LOCAL_MAILPIT_SMTP_PORT AVIA_LOCAL_MAILPIT_UI_PORT; do
    if [[ -z "${!port_name:-}" ]]; then
      echo "${port_name} must be explicitly supplied for strict task-owned local runs" >&2
      exit 64
    fi
    assert_local_port "$port_name" "${!port_name}"
  done
fi
AVIA_BOOTSTRAP_MANIFEST_DIR="${AVIA_BOOTSTRAP_MANIFEST_DIR:-${REPOSITORY_ROOT}/../../deployments/namibia/manifests}"
AVIA_ROSTER_CREDENTIAL_DIRECTORY="${AVIA_ROSTER_CREDENTIAL_DIRECTORY:-${AVIASURVEIL_LOCAL_STATE_DIR}/roster-credentials}"
manifest_digest() {
  local path="$1"
  if command -v shasum >/dev/null 2>&1; then
    printf 'sha256:%s\n' "$(shasum -a 256 "${path}" | awk '{print $1}')"
  else
    printf 'sha256:%s\n' "$(sha256sum "${path}" | awk '{print $1}')"
  fi
}
export AVIA_BOOTSTRAP_MANIFEST_DIR AVIA_ROSTER_CREDENTIAL_DIRECTORY
export AVIA_FOUNDATION_MANIFEST_SHA256="${AVIA_FOUNDATION_MANIFEST_SHA256:-$(manifest_digest "${AVIA_BOOTSTRAP_MANIFEST_DIR}/${LOCAL_MANIFEST_PREFIX}-foundation.json")}"
export AVIA_ROSTER_MANIFEST_SHA256="${AVIA_ROSTER_MANIFEST_SHA256:-$(manifest_digest "${AVIA_BOOTSTRAP_MANIFEST_DIR}/${LOCAL_MANIFEST_PREFIX}-identity-roster.json")}"
export AVIA_CATALOG_MANIFEST_SHA256="${AVIA_CATALOG_MANIFEST_SHA256:-$(manifest_digest "${AVIA_BOOTSTRAP_MANIFEST_DIR}/${LOCAL_MANIFEST_PREFIX}-approved-catalog.json")}"
export AVIA_AI_RECOMMENDATION_ARTIFACT_SHA256="${AVIA_AI_RECOMMENDATION_ARTIFACT_SHA256:-$(manifest_digest "${REPOSITORY_ROOT}/deliverables/aga-ai-checklist-recommendations-v1/AGA_AI_CHECKLIST_RECOMMENDATIONS_V1.json")}"
OWNER_MARKER="${AVIASURVEIL_LOCAL_STATE_DIR}/.compose-project-owner"
PROFILE_MARKER="${AVIASURVEIL_LOCAL_STATE_DIR}/.compose-profile"

assert_owner() {
  if [[ ! -f "${OWNER_MARKER}" ]] ||
    [[ "$(tr -d '\r\n' <"${OWNER_MARKER}")" != "${AVIA_LOCAL_PROJECT}" ]]; then
    echo "state directory is not owned by Compose project ${AVIA_LOCAL_PROJECT}" >&2
    exit 1
  fi
}

initialize_owner() {
  if [[ -e "${AVIASURVEIL_LOCAL_STATE_DIR}" ]]; then
    assert_owner
  else
    mkdir -p "${AVIASURVEIL_LOCAL_STATE_DIR}"
    chmod 0700 "${AVIASURVEIL_LOCAL_STATE_DIR}"
    printf '%s\n' "${AVIA_LOCAL_PROJECT}" >"${OWNER_MARKER}"
    chmod 0600 "${OWNER_MARKER}"
  fi
  printf '%s\n' "${PROFILE}" >"${PROFILE_MARKER}"
  chmod 0600 "${PROFILE_MARKER}"
}

compose() {
  docker compose \
    --project-name "${AVIA_LOCAL_PROJECT}" \
    --file "${COMPOSE_FILE}" \
    --profile "${PROFILE}" \
    "$@"
}

case "${COMMAND}" in
  up)
    initialize_owner
    if [[ "${PROFILE}" =~ ^(full|test|recovery)$ ]] &&
      [[ ! -f "${AVIASURVEIL_LOCAL_STATE_DIR}/bootstrap.json" ]]; then
      AVIASURVEIL_LOCAL_STATE_DIR="${AVIASURVEIL_LOCAL_STATE_DIR}" \
        "${REPOSITORY_ROOT}/scripts/init-local-demo-bootstrap.sh"
    fi
    compose build
    compose up --detach --wait
    printf 'Local %s profile is running as project %s\n' "${PROFILE}" "${AVIA_LOCAL_PROJECT}"
    ;;
  down)
    assert_owner
    compose down --volumes --remove-orphans --timeout 15
    if docker ps --all --quiet \
      --filter "label=com.docker.compose.project=${AVIA_LOCAL_PROJECT}" |
      grep -q .; then
      echo "task-owned containers remain after down" >&2
      exit 1
    fi
    rm -rf -- "${AVIASURVEIL_LOCAL_STATE_DIR}"
    printf 'Removed task-owned project %s and its scoped state\n' "${AVIA_LOCAL_PROJECT}"
    ;;
  status)
    assert_owner
    compose ps --all
    ;;
  logs)
    assert_owner
    compose logs --no-color
    ;;
  check)
    assert_owner
    "${REPOSITORY_ROOT}/scripts/check-local-runtime.sh"
    ;;
esac
