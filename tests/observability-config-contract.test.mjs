import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const observabilityRoot = path.join(repositoryRoot, "deploy/observability");
const composePath = path.join(
  observabilityRoot,
  "compose.observability.yaml",
);
const localComposePath = path.join(repositoryRoot, "deploy/local/compose.yaml");
const collectorPath = path.join(observabilityRoot, "otel-collector.yaml");
const prometheusPath = path.join(observabilityRoot, "prometheus.yaml");
const rulesPath = path.join(
  observabilityRoot,
  "rules/aviasurveil360.yaml",
);
const alertmanagerPath = path.join(observabilityRoot, "alertmanager.yaml");
const lokiPath = path.join(observabilityRoot, "loki.yaml");
const tempoPath = path.join(observabilityRoot, "tempo.yaml");
const datasourcePath = path.join(
  observabilityRoot,
  "grafana/provisioning/datasources/aviasurveil360.yaml",
);
const dashboardProviderPath = path.join(
  observabilityRoot,
  "grafana/provisioning/dashboards/aviasurveil360.yaml",
);
const dashboardPath = path.join(
  observabilityRoot,
  "grafana/dashboards/aviasurveil360-overview.json",
);
const profileScriptPath = path.join(
  repositoryRoot,
  "scripts/test-observability-profile.sh",
);

const requiredServices = [
  "alertmanager",
  "grafana",
  "loki",
  "otel-collector",
  "prometheus",
  "tempo",
];

function readRequired(filePath) {
  assert.ok(
    existsSync(filePath),
    `${path.relative(repositoryRoot, filePath)} must exist`,
  );
  return readFileSync(filePath, "utf8");
}

function composeConfig() {
  readRequired(composePath);
  const output = execFileSync(
    "docker",
    [
      "compose",
      "--file",
      localComposePath,
      "--file",
      composePath,
      "--profile",
      "full",
      "--profile",
      "observability",
      "config",
      "--format",
      "json",
    ],
    {
      cwd: repositoryRoot,
      encoding: "utf8",
      env: {
        ...process.env,
        AVIASURVEIL_LOCAL_STATE_DIR:
          "/private/tmp/aviasurveil360-observability-contract",
      },
    },
  );
  return JSON.parse(output);
}

test("observability profile declares only immutable-digest service images", () => {
  const compose = composeConfig();
  for (const serviceName of requiredServices) {
    const service = compose.services[serviceName];
    assert.ok(service, `${serviceName} service is required`);
    assert.deepEqual(
      service.profiles,
      ["observability"],
      `${serviceName} must be opt-in`,
    );
    assert.match(
      service.image,
      /@sha256:[a-f0-9]{64}$/,
      `${serviceName} image must use an immutable digest`,
    );
  }
});

test("observability services are bounded, non-root, read-only, and healthy", () => {
  const compose = composeConfig();
  for (const serviceName of requiredServices) {
    const service = compose.services[serviceName];
    assert.equal(service.read_only, true, `${serviceName} rootfs must be read-only`);
    assert.match(String(service.user), /^[1-9]\d*:[0-9]+$/);
    assert.ok(service.healthcheck?.test, `${serviceName} needs a healthcheck`);
    assert.ok(
      Number(service.pids_limit) > 0,
      `${serviceName} needs a PID limit`,
    );
    assert.ok(
      service.deploy?.resources?.limits?.memory,
      `${serviceName} needs a memory limit`,
    );
    assert.ok(
      service.deploy?.resources?.limits?.cpus,
      `${serviceName} needs a CPU limit`,
    );
  }

  const composeSource = readRequired(composePath);
  assert.match(composeSource, /retention\.time=\d+[dh]/);
  assert.match(readRequired(lokiPath), /retention_period:\s*\d+[dh]/);
  assert.match(readRequired(tempoPath), /block_retention:\s*\d+[dh]/);
});

test("Grafana is authenticated and provisions Git-owned data sources and dashboards", () => {
  const compose = composeConfig();
  const grafana = compose.services.grafana;
  assert.equal(grafana.environment.GF_AUTH_ANONYMOUS_ENABLED, "false");
  assert.equal(grafana.environment.GF_USERS_ALLOW_SIGN_UP, "false");
  assert.match(
    grafana.environment.GF_SECURITY_ADMIN_PASSWORD__FILE,
    /^\/run\/secrets\//,
  );
  assert.ok(
    grafana.secrets.some(({ source }) => source === "grafana_admin_password"),
  );

  const datasources = readRequired(datasourcePath);
  for (const expected of ["Prometheus", "Loki", "Tempo"]) {
    assert.match(datasources, new RegExp(`name:\\s*${expected}\\b`));
  }
  assert.match(readRequired(dashboardProviderPath), /disableDeletion:\s*true/);
  const dashboard = JSON.parse(readRequired(dashboardPath));
  assert.equal(dashboard.editable, false);
  assert.ok(dashboard.panels.length >= 6);
});

test("collector accepts OTLP and applies bounded allowlisting plus redaction", () => {
  const source = readRequired(collectorPath);
  assert.match(source, /otlp:/);
  assert.match(source, /memory_limiter:/);
  assert.match(source, /batch:/);
  assert.match(source, /transform\/redaction:/);
  assert.match(source, /filter\/allowlisted_attributes:/);
  for (const forbiddenKey of [
    "authorization",
    "cookie",
    "password",
    "session",
    "token",
    "internal_caa_note",
    "evidence_bytes",
    "message_body",
  ]) {
    assert.match(source, new RegExp(forbiddenKey));
  }
  assert.match(source, /service:/);
  assert.match(source, /^\s+pipelines:\s*$/m);
  assert.match(source, /traces:/);
  assert.match(source, /metrics:/);
  assert.match(source, /logs:/);
});

test("Prometheus loads bounded retention and the complete owned alert catalog", () => {
  const prometheus = readRequired(prometheusPath);
  assert.match(prometheus, /rule_files:/);
  assert.match(prometheus, /aviasurveil360\.yaml/);
  assert.match(prometheus, /alertmanagers:/);

  const rules = readRequired(rulesPath);
  const alertNames = [...rules.matchAll(/^\s*-\s+alert:\s*(\S+)\s*$/gm)].map(
    ([, name]) => name,
  );
  assert.equal(alertNames.length, 8);
  assert.equal(new Set(alertNames).size, 8);
  for (const alertName of alertNames) {
    const start = rules.indexOf(`- alert: ${alertName}`);
    const next = rules.indexOf("\n  - alert:", start + 1);
    const block = rules.slice(start, next === -1 ? undefined : next);
    assert.match(block, /^\s+for:\s*\d+[ms]\s*$/m);
    assert.match(block, /^\s+owner:\s*.+$/m);
    assert.match(block, /^\s+severity:\s*(warning|critical)\s*$/m);
    assert.match(block, /^\s+dedup_key:\s*.+$/m);
    assert.match(block, /^\s+runbook_url:\s*docs\/operations\/runbooks\/.+\.md\s*$/m);
    assert.match(block, /^\s+recovery_condition:\s*.+$/m);
  }
});

test("Alertmanager groups, inhibits derivative alerts, and sends resolved mail", () => {
  const source = readRequired(alertmanagerPath);
  assert.match(source, /group_by:\s*\[[^\]]*alertname[^\]]*service[^\]]*severity[^\]]*\]/);
  assert.match(source, /group_wait:\s*\d+s/);
  assert.match(source, /group_interval:\s*\d+[sm]/);
  assert.match(source, /repeat_interval:\s*\d+h/);
  assert.match(source, /inhibit_rules:/);
  assert.match(source, /required-dependency-down/);
  assert.match(source, /smtp_smarthost:\s*mailpit:1025/);
  assert.match(source, /smtp_auth_username:\s*aviasurveil360/);
  assert.match(source, /smtp_auth_password_file:\s*\/run\/secrets\/smtp_password/);
  assert.match(source, /send_resolved:\s*true/);
});

test("observability backends publish no ports and use the Caddy HTTPS origin", () => {
  const compose = composeConfig();
  for (const serviceName of requiredServices) {
    const ports = compose.services[serviceName].ports ?? [];
    assert.deepEqual(ports, [], `${serviceName} must not publish ports`);
  }
  assert.equal(
    compose.services.grafana.environment.GF_SERVER_SERVE_FROM_SUB_PATH,
    "true",
  );
  assert.match(
    compose.services.grafana.environment.GF_SERVER_ROOT_URL,
    /^https:\/\/localhost:\d+\/operations\/$/,
  );
  assert.equal(compose.networks.observability.internal, true);
  const gatewayConfigMount = compose.services.gateway.volumes.find(
    (volume) => volume.target === "/etc/caddy/Caddyfile",
  );
  assert.equal(gatewayConfigMount?.type, "bind");
  assert.equal(gatewayConfigMount?.read_only, true);
  const gateway = readRequired(
    path.join(repositoryRoot, "deploy/local/gateway/Caddyfile"),
  );
  assert.match(gateway, /handle\s+\/operations\/\*/);
  assert.doesNotMatch(gateway, /handle_path\s+\/operations\/\*/);
  assert.match(gateway, /reverse_proxy\s+grafana:3000/);
});

test("profile harness proves fixtures, correlation, recovery, persistence, and cleanup", () => {
  const source = readRequired(profileScriptPath);
  assert.match(source, /aviasurveil360-task-plan4-observability-/);
  assert.match(source, /--profile\s+observability/);
  assert.match(source, /scripts\/init-local-secrets\.sh/);
  assert.match(source, /api-read-latency/);
  assert.match(source, /api-command-latency/);
  assert.match(source, /outbox-ready-warning/);
  assert.match(source, /outbox-ready-critical/);
  assert.match(source, /worker-attempts/);
  assert.match(source, /backup-incremental-age/);
  assert.match(source, /backup-full-age/);
  assert.match(source, /required-dependency-down/);
  assert.match(source, /traceId|TRACE_ID_HEX/);
  assert.match(source, /correlation\.id|CORRELATION_ID/);
  assert.match(source, /Mailpit/);
  assert.match(source, /send_resolved/);
  assert.match(source, /restart/);
  assert.match(source, /assert_no_task_owned_residue/);
  assert.doesNotMatch(source, /docker\s+(?:system\s+)?prune/);
  assert.doesNotMatch(source, /docker\s+(?:container|volume|network)?\s*rm\s+(?:-f\s+)?\$\{/);
});

test("profile harness sends OTLP trace and span identifiers as hex", () => {
  const source = readRequired(profileScriptPath);
  assert.match(source, /const traceId = traceHex;/);
  assert.match(source, /const spanId = spanHex;/);
  assert.doesNotMatch(
    source,
    /Buffer\.from\((?:traceHex|spanHex),\s*"hex"\)\.toString\("base64"\)/,
  );
});
