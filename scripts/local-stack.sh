#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIRECTORY="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIRECTORY}/.." && pwd)"
COMPOSE_FILE="${REPOSITORY_ROOT}/deploy/local/compose.yaml"
COMMAND="${1:-status}"
PROFILE="${2:-${AVIA_LOCAL_PROFILE:-full}}"

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
AVIA_PREPROD_STATE_DIR="${AVIA_PREPROD_STATE_DIR:-${AVIASURVEIL_LOCAL_STATE_DIR}/first-party-auth}"
AVIA_PREPROD_WEB_ORIGIN="${AVIA_PREPROD_WEB_ORIGIN:-https://localhost:${AVIA_LOCAL_HTTPS_PORT:-8443}}"
export AVIA_PREPROD_STATE_DIR AVIA_PREPROD_WEB_ORIGIN
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
    if [[ "${PROFILE}" != "demo" ]] &&
      [[ ! -f "${AVIASURVEIL_LOCAL_STATE_DIR}/secrets/app_database_password" ]]; then
      "${REPOSITORY_ROOT}/scripts/init-local-secrets.sh"
    fi
    if [[ "${PROFILE}" =~ ^(demo|full)$ ]] &&
      [[ ! -f "${AVIA_PREPROD_STATE_DIR}/namespace.json" ]]; then
      "${REPOSITORY_ROOT}/scripts/init-local-preprod-namespace.sh"
    fi
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
