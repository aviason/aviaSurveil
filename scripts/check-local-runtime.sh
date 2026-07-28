#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIRECTORY="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIRECTORY}/.." && pwd)"
COMPOSE_FILE="${REPOSITORY_ROOT}/deploy/local/compose.yaml"
PROJECT="${AVIA_LOCAL_PROJECT:-}"
PROFILE="${AVIA_LOCAL_PROFILE:-full}"
STATE_DIRECTORY="${AVIASURVEIL_LOCAL_STATE_DIR:-}"
HTTPS_PORT="${AVIA_LOCAL_HTTPS_PORT:-8443}"
MODE="${1:-check}"

for required_command in curl docker node rg; do
  if ! command -v "${required_command}" >/dev/null 2>&1; then
    echo "required command is unavailable: ${required_command}" >&2
    exit 69
  fi
done

if [[ ! "${PROJECT}" =~ ^aviasurveil360-task-[a-z0-9][a-z0-9-]*$ ]]; then
  echo "AVIA_LOCAL_PROJECT must identify one exact task-owned project" >&2
  exit 64
fi

project_resource_ids() {
  {
    docker ps --all --quiet --filter "label=com.docker.compose.project=${PROJECT}"
    docker network ls --quiet --filter "label=com.docker.compose.project=${PROJECT}"
    docker volume ls --quiet --filter "label=com.docker.compose.project=${PROJECT}"
  } | sed '/^$/d'
}

if [[ "${MODE}" == "--assert-clean" ]]; then
  if project_resource_ids | grep -q .; then
    echo "task-owned runtime residue remains for ${PROJECT}" >&2
    exit 1
  fi
  echo "Task-owned runtime residue: zero"
  exit 0
fi
if [[ "${MODE}" != "check" ]]; then
  echo "usage: $0 [--assert-clean]" >&2
  exit 64
fi
if [[ "${PROFILE}" != "full" ]]; then
  echo "runtime failure matrix requires AVIA_LOCAL_PROFILE=full" >&2
  exit 64
fi
if [[ -z "${STATE_DIRECTORY}" ]] ||
  [[ "$(tr -d '\r\n' <"${STATE_DIRECTORY}/.compose-project-owner" 2>/dev/null || true)" != "${PROJECT}" ]]; then
  echo "runtime state is not owned by ${PROJECT}" >&2
  exit 1
fi

COMPOSE=(
  docker compose
  --project-name "${PROJECT}"
  --file "${COMPOSE_FILE}"
  --profile "${PROFILE}"
)
RUNTIME_ARTIFACT_DIRECTORY="$(mktemp -d /private/tmp/aviasurveil360-runtime-check.XXXXXX)"
cleanup_artifacts() {
  rm -rf -- "${RUNTIME_ARTIFACT_DIRECTORY}"
}
trap cleanup_artifacts EXIT HUP INT TERM

compose() {
  "${COMPOSE[@]}" "$@"
}

container_id() {
  compose ps --all --quiet "$1"
}

wait_for_service_health() {
  local service=$1
  local deadline=$((SECONDS + 180))
  local identifier status
  while ((SECONDS < deadline)); do
    identifier="$(container_id "${service}")"
    if [[ -n "${identifier}" ]]; then
      status="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "${identifier}")"
      if [[ "${status}" == "healthy" || "${status}" == "running" ]]; then
        return 0
      fi
    fi
    sleep 1
  done
  echo "${service} did not recover to a healthy/running state" >&2
  return 1
}

request_health() {
  local path=$1
  local output=$2
  curl --insecure --silent --show-error \
    --output "${output}" \
    --write-out '%{http_code}' \
    "https://localhost:${HTTPS_PORT}${path}"
}

wait_for_readiness() {
  local expected_code=$1
  local expected_status=$2
  local deadline=$((SECONDS + 90))
  local code
  while ((SECONDS < deadline)); do
    code="$(request_health /health/ready "${RUNTIME_ARTIFACT_DIRECTORY}/ready.json" || true)"
    if [[ "${code}" == "${expected_code}" ]] &&
      node -e '
        const report = JSON.parse(require("node:fs").readFileSync(process.argv[1], "utf8"));
        process.exit(report.status === process.argv[2] ? 0 : 1);
      ' "${RUNTIME_ARTIFACT_DIRECTORY}/ready.json" "${expected_status}"; then
      return 0
    fi
    sleep 1
  done
  echo "readiness did not reach ${expected_code}/${expected_status}" >&2
  return 1
}

assert_liveness() {
  local code
  code="$(request_health /health/live "${RUNTIME_ARTIFACT_DIRECTORY}/live.json")"
  if [[ "${code}" != "200" ]]; then
    echo "liveness changed with downstream service state" >&2
    return 1
  fi
}

assert_only_gateway_published() {
  local service identifier published
  while IFS= read -r service; do
    identifier="$(container_id "${service}")"
    [[ -n "${identifier}" ]] || continue
    published="$(docker inspect --format '{{range $port, $bindings := .NetworkSettings.Ports}}{{if $bindings}}{{$port}} {{end}}{{end}}' "${identifier}")"
    if [[ "${service}" == "gateway" ]]; then
      [[ -n "${published}" ]] || {
        echo "gateway has no published HTTPS port" >&2
        return 1
      }
    elif [[ -n "${published}" ]]; then
      echo "internal service ${service} has a published port" >&2
      return 1
    fi
  done < <(compose config --services)
  echo "Published runtime ports: gateway only"
}

assert_service_networks() {
  local service=$1
  local expected_networks=$2
  local identifier actual expected network
  identifier="$(container_id "${service}")"
  if [[ -z "${identifier}" ]]; then
    echo "missing container for network check: ${service}" >&2
    return 1
  fi
  actual="$(
    docker inspect \
      --format '{{range $network, $_ := .NetworkSettings.Networks}}{{$network}}{{"\n"}}{{end}}' \
      "${identifier}" |
      while IFS= read -r network; do
        [[ -n "${network}" ]] || continue
        printf '%s\n' "${network#"${PROJECT}_"}"
      done |
      sort
  )"
  expected="$(tr ' ' '\n' <<<"${expected_networks}" | sed '/^$/d' | sort)"
  if [[ "${actual}" != "${expected}" ]]; then
    echo "network membership mismatch for ${service}" >&2
    return 1
  fi
}

assert_exact_network_membership() {
  assert_service_networks gateway "edge frontend identity platform"
  assert_service_networks web-http "frontend"
  assert_service_networks api "database frontend identity platform"
  assert_service_networks worker "database identity platform"
  assert_service_networks scheduler "database"
  assert_service_networks migration "database"
  assert_service_networks postgres "database"
  assert_service_networks keycloak-postgres "identity-database"
  assert_service_networks keycloak "identity identity-database"
  assert_service_networks minio "platform"
  assert_service_networks clamav "platform signature-updates"
  assert_service_networks mailpit "identity platform"
  assert_service_networks gotenberg "platform"
  echo "Network membership: exact"
}

assert_no_orphan_containers() {
  local expected actual
  expected="$(compose config --services | sort)"
  actual="$(
    docker ps --all \
      --filter "label=com.docker.compose.project=${PROJECT}" \
      --format '{{.Label "com.docker.compose.service"}}' |
      sort -u
  )"
  while IFS= read -r service; do
    [[ -z "${service}" ]] && continue
    if ! grep -Fxq -- "${service}" <<<"${expected}"; then
      echo "orphan container detected for service ${service}" >&2
      return 1
    fi
  done <<<"${actual}"
  echo "Orphan containers: zero"
}

assert_no_secret_leakage() {
  local secret_path secret_name secret_value
  compose logs --no-color >"${RUNTIME_ARTIFACT_DIRECTORY}/compose.log"
  for secret_path in "${STATE_DIRECTORY}/secrets/"*; do
    [[ -f "${secret_path}" ]] || continue
    secret_name="${secret_path##*/}"
    secret_value="$(tr -d '\r\n' <"${secret_path}")"
    if [[ -n "${secret_value}" ]] &&
      rg --fixed-strings --quiet -- "${secret_value}" "${RUNTIME_ARTIFACT_DIRECTORY}"; then
      echo "generated secret found in runtime logs: ${secret_name}" >&2
      return 1
    fi
  done
  echo "Runtime log secret scan: zero generated-secret matches"
}

inject_dependency_failure() {
  local service=$1
  local status=$2
  local expected_code=$3
  echo "Injecting ${service} dependency loss"
  compose stop --timeout 15 "${service}"
  assert_liveness
  wait_for_readiness "${expected_code}" "${status}"
  compose start "${service}"
  wait_for_service_health "${service}"
  wait_for_readiness 200 ready
}

assert_worker_restart() {
  local identifier before after deadline
  identifier="$(container_id worker)"
  before="$(docker inspect --format '{{.RestartCount}}' "${identifier}")"
  compose exec --no-TTY worker sh -c '
    set -- $(cat /proc/1/task/1/children)
    if [ "$#" -ne 1 ]; then
      echo "expected one application child under container init" >&2
      exit 1
    fi
    kill -KILL "$1"
  '
  deadline=$((SECONDS + 60))
  while ((SECONDS < deadline)); do
    identifier="$(container_id worker)"
    if [[ -n "${identifier}" ]]; then
      after="$(docker inspect --format '{{.RestartCount}}' "${identifier}")"
      if ((after > before)); then
        wait_for_service_health worker
        echo "Worker crash restart: recovered"
        return 0
      fi
    fi
    sleep 1
  done
  echo "worker did not restart after an injected crash" >&2
  return 1
}

assert_only_gateway_published
assert_exact_network_membership
assert_no_orphan_containers
assert_liveness
wait_for_readiness 200 ready

if [[ "${AVIA_RUNTIME_FAILURE_MATRIX:-0}" == "1" ]]; then
  for required in postgres keycloak minio clamav; do
    inject_dependency_failure "${required}" not_ready 503
  done
  for optional in gotenberg mailpit; do
    inject_dependency_failure "${optional}" degraded 200
  done
  assert_worker_restart
fi

assert_no_secret_leakage
echo "Runtime isolation, failure, bounded shutdown, restart, and residue contracts passed"
