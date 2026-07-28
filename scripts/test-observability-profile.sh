#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIRECTORY="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd "${SCRIPT_DIRECTORY}/.." && pwd)"
LOCAL_COMPOSE_FILE="${REPOSITORY_ROOT}/deploy/local/compose.yaml"
OBSERVABILITY_COMPOSE_FILE="${REPOSITORY_ROOT}/deploy/observability/compose.observability.yaml"
RUNTIME_DIRECTORY="$(mktemp -d /private/tmp/aviasurveil360-plan4-observability.XXXXXX)"
AVIA_LOCAL_PROJECT="aviasurveil360-task-plan4-observability-$(date -u +%Y%m%d%H%M%S)-$$"
AVIASURVEIL_LOCAL_STATE_DIR="${RUNTIME_DIRECTORY}/local-state"
AVIA_LOCAL_HTTPS_PORT="${AVIA_OBSERVABILITY_HTTPS_PORT:-$((38443 + RANDOM % 10000))}"
STACK_STARTED=false

export AVIA_LOCAL_PROJECT AVIASURVEIL_LOCAL_STATE_DIR
export AVIA_LOCAL_HTTPS_PORT
export AVIA_LOCAL_PUBLIC_ORIGIN="https://localhost:${AVIA_LOCAL_HTTPS_PORT}"
export COMPOSE_PROGRESS=plain

COMPOSE=(
  docker compose
  --project-name "${AVIA_LOCAL_PROJECT}"
  --file "${LOCAL_COMPOSE_FILE}"
  --file "${OBSERVABILITY_COMPOSE_FILE}"
  --profile full
  --profile observability
)

compose() {
  "${COMPOSE[@]}" "$@"
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
  return "${status}"
}

assert_no_task_owned_residue() {
  local kind
  for kind in container volume network; do
    case "${kind}" in
      container)
        docker ps --all --quiet \
          --filter "label=com.docker.compose.project=${AVIA_LOCAL_PROJECT}"
        ;;
      volume)
        docker volume ls --quiet \
          --filter "label=com.docker.compose.project=${AVIA_LOCAL_PROJECT}"
        ;;
      network)
        docker network ls --quiet \
          --filter "label=com.docker.compose.project=${AVIA_LOCAL_PROJECT}"
        ;;
    esac
  done |
    if grep -q .; then
      echo "task-owned Compose residue remains for ${AVIA_LOCAL_PROJECT}" >&2
      return 1
    fi
  echo "Task-owned observability residue: zero"
}

cleanup() {
  local status=$?
  trap - EXIT
  set +e
  if [[ "${STACK_STARTED}" == true ]]; then
    compose down --volumes --remove-orphans --timeout 15
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
trap cleanup EXIT HUP INT TERM

wait_for() {
  local description=$1
  local command=$2
  local deadline=$((SECONDS + 120))
  until eval "${command}"; do
    if ((SECONDS >= deadline)); then
      echo "timed out waiting for ${description}" >&2
      return 1
    fi
    sleep 2
  done
}

internal_get() {
  compose exec --no-TTY api \
    wget --quiet --output-document=- "$1"
}

mailpit_count() {
  internal_get http://mailpit:8025/api/v1/messages |
    node -e '
      let body = "";
      process.stdin.on("data", (chunk) => body += chunk);
      process.stdin.on("end", () => {
        const payload = JSON.parse(body);
        const messages = payload.messages ?? payload.Messages ?? [];
        process.stdout.write(String(messages.length));
      });
    '
}

wait_for_mailpit_count() {
  local minimum=$1
  local deadline=$((SECONDS + 90))
  local current
  while ((SECONDS < deadline)); do
    current="$(mailpit_count)"
    if ((current >= minimum)); then
      return 0
    fi
    sleep 2
  done
  echo "Mailpit did not reach ${minimum} messages" >&2
  return 1
}

post_alerts() {
  local payload_path=$1
  local payload
  payload="$(<"${payload_path}")"
  compose exec --no-TTY api \
    wget \
    --quiet \
    --output-document=- \
    --header "Content-Type: application/json" \
    --post-data "${payload}" \
    http://alertmanager:9093/api/v2/alerts \
    >/dev/null
}

mkdir -p "${AVIASURVEIL_LOCAL_STATE_DIR}"
chmod 0700 "${AVIASURVEIL_LOCAL_STATE_DIR}"
printf '%s\n' "${AVIA_LOCAL_PROJECT}" \
  >"${AVIASURVEIL_LOCAL_STATE_DIR}/.compose-project-owner"
chmod 0600 "${AVIASURVEIL_LOCAL_STATE_DIR}/.compose-project-owner"
"${REPOSITORY_ROOT}/scripts/init-local-secrets.sh"

"${REPOSITORY_ROOT}/scripts/check-local-image-evidence.sh" full

for image in \
  "otel/opentelemetry-collector-contrib:0.153.0@sha256:93aad750175cbf1a973ae1c5886c3371f4d800f61be25cdd26870b8441ffe9fa" \
  "prom/prometheus:v3.12.0@sha256:69f5241418838263316593f7274a304b095c40bcf22e57272865da91bd60a8ac" \
  "grafana/grafana:13.0.2@sha256:0b35c0b9de28b45b6d07b92c8b09a7b53559c9cacde2a535b0179b7ace673411" \
  "grafana/loki:3.7.2@sha256:191d4fdfb7264f16989f0a57f320872620a5a7c2ceeec6229212c4190ec49b86" \
  "grafana/tempo:2.10.5@sha256:ee21727732c7a7199cb71c3eee9153bbf23f9b0b87619f0555a0cf21a67f1a33" \
  "prom/alertmanager:v0.32.1@sha256:51a825c2a40acc3e338fdd00d622e01ec090f72be2b3ea46be0839cd47a4d286"; do
  if ! docker image inspect "${image}" >/dev/null 2>&1; then
    docker pull "${image}"
  fi
done

STACK_STARTED=true
compose up --detach --wait

compose exec --no-TTY prometheus \
  /bin/promtool check config /etc/prometheus/prometheus.yaml
compose exec --no-TTY alertmanager \
  /bin/amtool check-config /etc/alertmanager/alertmanager.yaml

published_services=""
while IFS= read -r service; do
  identifier="$(compose ps --all --quiet "${service}")"
  [[ -n "${identifier}" ]] || continue
  published="$(
    docker inspect \
      --format '{{range $port, $bindings := .NetworkSettings.Ports}}{{range $bindings}}{{if .HostPort}}{{.HostIp}}:{{.HostPort}}->{{$port}}{{"\n"}}{{end}}{{end}}{{end}}' \
      "${identifier}"
  )"
  if [[ -n "${published}" ]]; then
    if [[ "${service}" != "gateway" && "${service}" != "grafana" ]]; then
      echo "internal service ${service} has a published port: ${published}" >&2
      exit 1
    fi
    if grep -Evq '^127\.0\.0\.1:' <<<"${published}"; then
      echo "${service} published beyond loopback: ${published}" >&2
      exit 1
    fi
    published_services+="${service}"$'\n'
  fi
done < <(compose config --services | sort)
published_services="${published_services%$'\n'}"
if [[ "${published_services}" != "gateway" ]]; then
  echo "unexpected published services: ${published_services}" >&2
  exit 1
fi
echo "Published observability ports: Caddy gateway only"

GRAFANA_PASSWORD="$(
  tr -d '\r\n' \
    <"${AVIASURVEIL_LOCAL_STATE_DIR}/secrets/grafana_admin_password"
)"
echo "Checking authenticated Grafana dashboard provisioning"
wait_for \
  "authenticated Grafana dashboard provisioning" \
  "curl --fail --silent --show-error --insecure --user local-observability-admin:'${GRAFANA_PASSWORD}' https://localhost:${AVIA_LOCAL_HTTPS_PORT}/operations/api/dashboards/uid/aviasurveil360-overview >${RUNTIME_DIRECTORY}/grafana-dashboard.json"
node -e '
  const fs = require("node:fs");
  const dashboard = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
  if (dashboard.dashboard?.uid !== "aviasurveil360-overview") process.exit(1);
  if ((dashboard.dashboard?.panels ?? []).length < 6) process.exit(1);
' "${RUNTIME_DIRECTORY}/grafana-dashboard.json"
unset GRAFANA_PASSWORD

TRACE_ID_HEX="00112233445566778899aabbccddeeff"
SPAN_ID_HEX="0011223344556677"
CORRELATION_ID="plan4-correlation-001"
SERVICE_INSTANCE_ID="plan4-observability-fixture"
export TRACE_ID_HEX SPAN_ID_HEX CORRELATION_ID SERVICE_INSTANCE_ID

node -e '
  const fs = require("node:fs");
  const directory = process.argv[1];
  const traceHex = process.env.TRACE_ID_HEX;
  const spanHex = process.env.SPAN_ID_HEX;
  const traceId = traceHex;
  const spanId = spanHex;
  const now = BigInt(Date.now()) * 1000000n;
  const resource = {
    attributes: [
      { key: "service.name", value: { stringValue: "aviasurveil360-fixture" } },
      { key: "service.version", value: { stringValue: "plan4" } },
      { key: "service.instance.id", value: { stringValue: process.env.SERVICE_INSTANCE_ID } },
      { key: "deployment.environment.name", value: { stringValue: "local-candidate" } },
      { key: "authorization", value: { stringValue: "must-be-removed" } }
    ]
  };
  const attrs = (entries) => Object.entries(entries).map(([key, value]) => ({
    key,
    value: typeof value === "number" ? { doubleValue: value } : { stringValue: value }
  }));
  fs.writeFileSync(`${directory}/traces.json`, JSON.stringify({
    resourceSpans: [{
      resource,
      scopeSpans: [{
        scope: { name: "plan4-observability-fixture" },
        spans: [{
          traceId,
          spanId,
          name: "http.server.request",
          kind: 2,
          startTimeUnixNano: String(now - 5000000n),
          endTimeUnixNano: String(now),
          attributes: attrs({
            "http.request.method": "GET",
            "http.route": "/health/ready",
            "http.response.status_code": 200,
            "operation.class": "read",
            "module": "platform",
            "outcome.class": "succeeded",
            "correlation.id": process.env.CORRELATION_ID,
            "password": "must-be-removed"
          }),
          status: { code: 1 }
        }]
      }]
    }]
  }));
  fs.writeFileSync(`${directory}/metrics.json`, JSON.stringify({
    resourceMetrics: [{
      resource,
      scopeMetrics: [{
        scope: { name: "plan4-observability-fixture" },
        metrics: [
          ["dependency.health", 1, {
            "dependency.name": "fixture",
            required: "true",
            "outcome.class": "succeeded"
          }],
          ["backup.recovery_point.age", 60, {
            "backup.stanza": "local",
            "backup.type": "incremental",
            "outcome.class": "succeeded"
          }],
          ["browser.web_vital", 120, {
            "route.id": "dashboard",
            "build.profile": "http",
            "web_vital.name": "LCP",
            rating: "good"
          }]
        ].map(([name, value, attributes]) => ({
          name,
          unit: name.includes("age") ? "s" : "1",
          gauge: {
            dataPoints: [{
              timeUnixNano: String(now),
              asDouble: value,
              attributes: attrs(attributes)
            }]
          }
        }))
      }]
    }]
  }));
  fs.writeFileSync(`${directory}/logs.json`, JSON.stringify({
    resourceLogs: [{
      resource,
      scopeLogs: [{
        scope: { name: "plan4-observability-fixture" },
        logRecords: [{
          timeUnixNano: String(now),
          severityText: "INFO",
          body: { stringValue: "browser.error.handled" },
          traceId,
          spanId,
          attributes: attrs({
            "route.id": "dashboard",
            "build.profile": "http",
            "error.class": "FixtureError",
            "outcome.class": "handled",
            "correlation.id": process.env.CORRELATION_ID,
            "message_body": "must-be-removed"
          })
        }]
      }]
    }]
  }));
' "${RUNTIME_DIRECTORY}"

for signal in traces metrics logs; do
  echo "Submitting OTLP ${signal} fixture"
  curl \
    --fail \
    --silent \
    --show-error \
    --insecure \
    --header "Content-Type: application/json" \
    --data-binary "@${RUNTIME_DIRECTORY}/${signal}.json" \
    "https://localhost:${AVIA_LOCAL_HTTPS_PORT}/otel/v1/${signal}" \
    >/dev/null
done

wait_for \
  "Prometheus metric correlation" \
  "internal_get 'http://prometheus:9090/api/v1/query?query=aviasurveil_dependency_health' >${RUNTIME_DIRECTORY}/prometheus.json && node -e 'const p=require(\"${RUNTIME_DIRECTORY}/prometheus.json\");process.exit(p.data?.result?.some((r)=>r.metric?.service_instance_id===\"${SERVICE_INSTANCE_ID}\")?0:1)'"
wait_for \
  "Tempo trace correlation" \
  "internal_get 'http://tempo:3200/api/traces/${TRACE_ID_HEX}' >${RUNTIME_DIRECTORY}/tempo.json && rg --quiet --fixed-strings '${CORRELATION_ID}' '${RUNTIME_DIRECTORY}/tempo.json'"
wait_for \
  "Loki log correlation" \
  "internal_get 'http://loki:3100/loki/api/v1/query_range?query=%7Bservice_name%3D%22aviasurveil360-fixture%22%7D' >${RUNTIME_DIRECTORY}/loki.json && rg --quiet --fixed-strings '${CORRELATION_ID}' '${RUNTIME_DIRECTORY}/loki.json'"

if rg --fixed-strings --quiet \
  "must-be-removed" \
  "${RUNTIME_DIRECTORY}/prometheus.json" \
  "${RUNTIME_DIRECTORY}/tempo.json" \
  "${RUNTIME_DIRECTORY}/loki.json"; then
  echo "collector redaction leaked a forbidden fixture value" >&2
  exit 1
fi
echo "Trace/log/metric correlation and redaction: verified locally"

node -e '
  const fs = require("node:fs");
  const path = process.argv[1];
  const only = process.argv[2] || "";
  const now = new Date();
  const endsAt = new Date(now.getTime() + 15 * 60 * 1000);
  const fixtures = [
    ["AviaAPIReadLatency", "api-read-latency", "warning", "Backend", "api", "api-read-latency"],
    ["AviaAPICommandLatency", "api-command-latency", "warning", "Backend", "api", "api-command-latency"],
    ["AviaOutboxReadyWarning", "outbox-ready-warning", "warning", "Backend", "worker", "outbox-ready"],
    ["AviaOutboxReadyCritical", "outbox-ready-critical", "critical", "Backend", "worker", "outbox-ready"],
    ["AviaWorkerAttempts", "worker-attempts", "critical", "Backend", "worker", "worker-attempts"],
    ["AviaBackupIncrementalAge", "backup-incremental-age", "warning", "Platform-Operations", "backup", "backup-age"],
    ["AviaBackupFullAge", "backup-full-age", "critical", "Platform-Operations", "backup", "backup-age"],
    ["AviaRequiredDependencyDown", "required-dependency-down", "critical", "Platform-Operations", "full-stack", "full-stack-dependency"]
  ].filter(([, id]) => only === "" || id === only);
  fs.writeFileSync(path, JSON.stringify(fixtures.map(([alertname, id, severity, owner, service, dedup]) => ({
    labels: { alertname, catalog_id: id, severity, owner, service, dedup_key: dedup },
    annotations: {
      summary: `Plan 4 fixture ${id}`,
      runbook_url: "docs/operations/runbooks/INCIDENT_RESPONSE.md",
      recovery_condition: "Fixture returns to its bounded healthy state."
    },
    startsAt: now.toISOString(),
    endsAt: endsAt.toISOString(),
    generatorURL: "http://prometheus:9090/graph"
  }))));
' "${RUNTIME_DIRECTORY}/alert-read.json" api-read-latency

BASELINE_MAIL_COUNT="$(mailpit_count)"
post_alerts "${RUNTIME_DIRECTORY}/alert-read.json"
post_alerts "${RUNTIME_DIRECTORY}/alert-read.json"
wait_for_mailpit_count "$((BASELINE_MAIL_COUNT + 1))"
sleep 8
if (( "$(mailpit_count)" != BASELINE_MAIL_COUNT + 1 )); then
  echo "duplicate alert fixture produced an alert storm" >&2
  exit 1
fi
echo "Alertmanager duplicate grouping: one Mailpit delivery"

node -e '
  const fs = require("node:fs");
  const source = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
  const ended = new Date(Date.now() - 1000).toISOString();
  for (const alert of source) alert.endsAt = ended;
  fs.writeFileSync(process.argv[2], JSON.stringify(source));
' \
  "${RUNTIME_DIRECTORY}/alert-read.json" \
  "${RUNTIME_DIRECTORY}/alert-read-resolved.json"
post_alerts "${RUNTIME_DIRECTORY}/alert-read-resolved.json"
wait_for_mailpit_count "$((BASELINE_MAIL_COUNT + 2))"
echo "Alertmanager send_resolved recovery message: verified locally"

node -e '
  const fs = require("node:fs");
  const path = process.argv[1];
  const now = new Date();
  const endsAt = new Date(now.getTime() + 15 * 60 * 1000);
  const fixtures = [
    ["AviaAPICommandLatency", "api-command-latency", "warning", "Backend", "api", "api-command-latency"],
    ["AviaOutboxReadyWarning", "outbox-ready-warning", "warning", "Backend", "worker", "outbox-ready"],
    ["AviaOutboxReadyCritical", "outbox-ready-critical", "critical", "Backend", "worker", "outbox-ready"],
    ["AviaWorkerAttempts", "worker-attempts", "critical", "Backend", "worker", "worker-attempts"],
    ["AviaBackupIncrementalAge", "backup-incremental-age", "warning", "Platform-Operations", "backup", "backup-age"],
    ["AviaBackupFullAge", "backup-full-age", "critical", "Platform-Operations", "backup", "backup-age"],
    ["AviaRequiredDependencyDown", "required-dependency-down", "critical", "Platform-Operations", "full-stack", "full-stack-dependency"]
  ];
  fs.writeFileSync(path, JSON.stringify(fixtures.map(([alertname, id, severity, owner, service, dedup]) => ({
    labels: { alertname, catalog_id: id, severity, owner, service, dedup_key: dedup },
    annotations: {
      summary: `Plan 4 fixture ${id}`,
      runbook_url: "docs/operations/runbooks/INCIDENT_RESPONSE.md",
      recovery_condition: "Fixture returns to its bounded healthy state."
    },
    startsAt: now.toISOString(),
    endsAt: endsAt.toISOString()
  }))));
' "${RUNTIME_DIRECTORY}/alerts-remaining.json"
post_alerts "${RUNTIME_DIRECTORY}/alerts-remaining.json"
post_alerts "${RUNTIME_DIRECTORY}/alerts-remaining.json"

wait_for \
  "all remaining Alertmanager fixtures" \
  "internal_get http://alertmanager:9093/api/v2/alerts >${RUNTIME_DIRECTORY}/active-alerts.json && node -e 'const fs=require(\"node:fs\");const a=JSON.parse(fs.readFileSync(process.argv[1],\"utf8\"));const ids=new Set(a.map((v)=>v.labels?.catalog_id));process.exit([\"api-command-latency\",\"outbox-ready-warning\",\"outbox-ready-critical\",\"worker-attempts\",\"backup-incremental-age\",\"backup-full-age\",\"required-dependency-down\"].every((id)=>ids.has(id))?0:1)' '${RUNTIME_DIRECTORY}/active-alerts.json'"
echo "Every catalog alert fixture reached Alertmanager"

compose restart prometheus loki tempo grafana
compose up --detach --wait prometheus loki tempo grafana

wait_for \
  "Prometheus persistence after restart" \
  "internal_get 'http://prometheus:9090/api/v1/query?query=aviasurveil_dependency_health' >${RUNTIME_DIRECTORY}/prometheus-after-restart.json && rg --quiet --fixed-strings '${SERVICE_INSTANCE_ID}' '${RUNTIME_DIRECTORY}/prometheus-after-restart.json'"
wait_for \
  "Tempo persistence after restart" \
  "internal_get 'http://tempo:3200/api/traces/${TRACE_ID_HEX}' >${RUNTIME_DIRECTORY}/tempo-after-restart.json && rg --quiet --fixed-strings '${CORRELATION_ID}' '${RUNTIME_DIRECTORY}/tempo-after-restart.json'"
wait_for \
  "Loki persistence after restart" \
  "internal_get 'http://loki:3100/loki/api/v1/query_range?query=%7Bservice_name%3D%22aviasurveil360-fixture%22%7D' >${RUNTIME_DIRECTORY}/loki-after-restart.json && rg --quiet --fixed-strings '${CORRELATION_ID}' '${RUNTIME_DIRECTORY}/loki-after-restart.json'"
echo "Prometheus, Tempo, and Loki persistence across restart: verified locally"

compose logs --no-color >"${RUNTIME_DIRECTORY}/compose.log"
for secret_path in "${AVIASURVEIL_LOCAL_STATE_DIR}/secrets/"*; do
  [[ -f "${secret_path}" ]] || continue
  secret_value="$(tr -d '\r\n' <"${secret_path}")"
  if [[ -n "${secret_value}" ]] &&
    rg --fixed-strings --quiet -- "${secret_value}" "${RUNTIME_DIRECTORY}/compose.log"; then
    echo "generated secret leaked into observability logs" >&2
    exit 1
  fi
done
if ! rg --quiet 'send_resolved:\s*true' \
  "${REPOSITORY_ROOT}/deploy/observability/alertmanager.yaml"; then
  echo "Alertmanager recovery delivery was disabled" >&2
  exit 1
fi

echo "Local observability profile: verified locally"
