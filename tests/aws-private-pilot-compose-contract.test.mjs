import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const composePath = path.join(root, "deploy/aws-private-pilot/compose.yaml");
const source = readFileSync(composePath, "utf8");
const longRunning = ["gateway", "api", "worker", "keycloak"];
const jobs = ["database-bootstrap", "migration", "keycloak-bootstrap"];

function fixtureEnvironment() {
  const digest = (name, c) => `fixture.invalid/${name}@sha256:${c.repeat(64)}`;
  const values = {
    AVIA_GATEWAY_IMAGE: digest("gateway", "a"), AVIA_API_IMAGE: digest("api", "c"),
    AVIA_WORKER_IMAGE: digest("worker", "d"), AVIA_KEYCLOAK_IMAGE: digest("keycloak", "1"),
    AVIA_DATABASE_BOOTSTRAP_IMAGE: digest("database-bootstrap", "3"), AVIA_MIGRATION_IMAGE: digest("migration", "4"),
    AVIA_DATABASE_HOST: "db.private.invalid", AVIA_DATABASE_BOOTSTRAP_USER: "bootstrap", AVIA_AWS_REGION: "eu-central-1",
    AVIA_RDS_CA_BUNDLE_FILE: "/private/fixture/aws-rds-global-bundle.pem",
    AVIA_QUARANTINE_BUCKET: "fixture-quarantine", AVIA_CLEAN_BUCKET: "fixture-clean", AVIA_ATTACHMENTS_BUCKET: "fixture-attachments", AVIA_DOCUMENTS_BUCKET: "fixture-documents",
    AVIA_PUBLIC_ORIGIN: "https://pilot.example.invalid", AVIA_PUBLIC_HOSTNAME: "pilot.example.invalid",
    AVIA_OIDC_ISSUER_URL: "https://pilot.example.invalid/identity/realms/pilot", AVIA_OIDC_CLIENT_ID: "pilot", AVIA_KEYCLOAK_REALM: "pilot", AVIA_KEYCLOAK_SERVICE_CLIENT_ID: "api",
    AVIA_SMTP_HOSTNAME: "smtp.example.invalid", AVIA_SMTP_PORT: "587", AVIA_SMTP_FROM: "no-reply@example.invalid", AVIA_SMTP_USERNAME: "pilot", AVIA_SMTP_TRANSPORT: "starttls", AVIA_SMTP_TLS_SERVER_NAME: "smtp.example.invalid",
  };
  for (const name of ["APP_DATABASE_PASSWORD", "APP_MIGRATION_PASSWORD", "DATABASE_BOOTSTRAP_PASSWORD", "KEYCLOAK_DATABASE_PASSWORD", "OIDC_CLIENT_SECRET", "SESSION_ENCRYPTION_KEY", "KEYCLOAK_SERVICE_CLIENT_SECRET", "APP_SMTP_PASSWORD", "KEYCLOAK_SMTP_PASSWORD", "KEYCLOAK_REALM"]) {
    values[`AVIA_${name}_FILE`] = `/private/fixture/${name.toLowerCase()}`;
  }
  return { ...process.env, ...values };
}

function render() {
  const result = spawnSync("docker", ["compose", "--file", composePath, "--profile", "bootstrap", "--profile", "migration", "config", "--format", "json"], {
    cwd: root, encoding: "utf8", env: fixtureEnvironment(),
  });
  assert.equal(result.status, 0, result.stderr);
  return JSON.parse(result.stdout);
}

test("production surface contains only approved runtime roles and bounded jobs", () => {
  const config = render();
  assert.deepEqual(Object.keys(config.services).sort(), [...longRunning, ...jobs].sort());
  for (const forbidden of ["cloudflared", "postgres", "keycloak-postgres", "minio", "mailpit", "clamav", "backup-minio", "prometheus", "grafana", "loki", "tempo", "alertmanager", "fixture", "loader", "volume-init"]) {
    assert.equal(Object.hasOwn(config.services, forbidden), false, forbidden);
  }
  for (const job of jobs) assert.equal(config.services[job].restart, "no");
});

test("every container is immutable ARM64 non-root and bounded", () => {
  const config = render();
  for (const [name, service] of Object.entries(config.services)) {
    assert.match(service.image, /@sha256:[0-9a-f]{64}$/u, name);
    assert.equal(service.platform, "linux/arm64", name);
    assert.notEqual(service.user, "0:0", name);
    assert.equal(service.read_only, true, name);
    assert.deepEqual(service.cap_drop, ["ALL"], name);
    assert.ok(service.security_opt.includes("no-new-privileges:true"), name);
    assert.ok(service.pids_limit > 0, name);
    assert.ok(service.deploy?.resources?.limits?.memory, name);
    assert.ok(service.deploy?.resources?.limits?.cpus, name);
    if (longRunning.includes(name)) assert.ok(service.healthcheck, name);
    else assert.equal(service.restart, "no", name);
    assert.ok(service.logging?.options?.["max-size"], name);
    assert.doesNotMatch(JSON.stringify(service), /docker\.sock|privileged|network_mode.*host/u, name);
  }
});

test("only gateway publishes a host port and internal services stay private", () => {
  const config = render();
  const published = Object.entries(config.services).filter(([, service]) => Array.isArray(service.ports) && service.ports.length > 0);
  assert.deepEqual(published.map(([name]) => name), ["gateway"]);
  assert.deepEqual(config.services.gateway.ports, [{ mode: "ingress", target: 8080, published: "8080", protocol: "tcp", host_ip: "127.0.0.1" }]);
  assert.equal(config.networks.frontend.internal, true);
  assert.equal(config.networks.application.internal, true);
  assert.equal(config.networks.identity.internal, true);
  assert.ok(Object.hasOwn(config.services.gateway.networks, "identity"));
  assert.ok(Object.hasOwn(config.services.keycloak.networks, "identity"));
  assert.equal(Object.hasOwn(config.services, "gotenberg"), false);
  assert.equal(config.networks.egress.enable_ipv6, true);
  assert.ok(config.networks.egress.ipam.config.some(({ subnet }) => subnet === "fd36:6176:6961:360::/64"));
});

test("AWS profile has no static object credentials or local fallback", () => {
  assert.match(source, /AVIA_RUNTIME_PROFILE: aws-private-pilot/u);
  assert.match(source, /AVIA_OBJECT_STORE_MODE: aws-s3/u);
  assert.match(source, /AVIA_SCANNER_MODE: guardduty-s3/u);
  assert.doesNotMatch(source, /AVIA_OBJECT_STORE_(?:ACCESS_KEY|SECRET_KEY)/u);
  assert.doesNotMatch(source, /\/var\/run\/docker\.sock/u);
  assert.doesNotMatch(source, /host\.docker\.internal|host-gateway/u);
  assert.match(source, /required external SMTP hostname/u);
});

test("every RDS client requires verify-full with the reviewed CA binding", () => {
  const config = render();
  const caTarget = "/etc/ssl/certs/aws-rds-global-bundle.pem";
  for (const name of ["api", "worker", "keycloak", "database-bootstrap", "migration", "keycloak-bootstrap"]) {
    assert.ok(config.services[name].configs.some((entry) => entry.target === caTarget), name);
  }
  assert.equal(config.services.api.environment.AVIA_DATABASE_SSLMODE, "verify-full");
  assert.equal(config.services.api.environment.AVIA_DATABASE_SSLROOTCERT, caTarget);
  assert.equal(config.services.api.environment.AVIA_DATABASE_MAX_CONNECTIONS, "4");
  assert.equal(config.services.api.environment.AVIA_OIDC_DISCOVERY_URL, "http://keycloak:8080/identity/realms/pilot");
  assert.equal(config.services.api.environment.AVIA_OIDC_DISCOVERY_PRIVATE_NETWORK, "true");
  assert.equal(config.services.worker.environment.AVIA_OIDC_DISCOVERY_URL, "http://keycloak:8080/identity/realms/pilot");
  assert.equal(config.services["database-bootstrap"].environment.PGSSLMODE, "verify-full");
  assert.equal(config.services["database-bootstrap"].environment.PGSSLROOTCERT, caTarget);
  assert.match(config.services.keycloak.environment.KC_DB_URL, /sslmode=verify-full&sslrootcert=\/etc\/ssl\/certs\/aws-rds-global-bundle\.pem/u);
  assert.equal(config.services.keycloak.environment.KC_DB_POOL_MAX_SIZE, "8");
  assert.equal(config.services.keycloak.environment.KC_HTTP_MANAGEMENT_RELATIVE_PATH, "/identity");
  assert.match(config.services.keycloak.healthcheck.test.join(" "), /GET \/identity\/health\/ready/u);
  assert.match(config.services.keycloak.environment.JAVA_OPTS_APPEND, /mail\.smtp\.ssl\.checkserveridentity=true/u);
  assert.match(config.services.keycloak.environment.JAVA_OPTS_APPEND, /mail\.smtp\.starttls\.required=true/u);
  assert.match(source, /required reviewed AWS RDS CA bundle/u);
});

test("gateway and systemd contracts fail closed", () => {
  const caddy = readFileSync(path.join(root, "deploy/aws-private-pilot/gateway/Caddyfile"), "utf8");
  const unit = readFileSync(path.join(root, "deploy/aws-private-pilot/systemd/aviasurveil360-private-pilot.service"), "utf8");
  const tunnelUnit = readFileSync(path.join(root, "deploy/aws-private-pilot/systemd/aviasurveil360-private-pilot-tunnel.service"), "utf8");
  const healthUnit = readFileSync(path.join(root, "deploy/aws-private-pilot/systemd/aviasurveil360-private-pilot-tunnel-health.service"), "utf8");
  const healthTimer = readFileSync(path.join(root, "deploy/aws-private-pilot/systemd/aviasurveil360-private-pilot-tunnel-health.timer"), "utf8");
  const supervisor = readFileSync(path.join(root, "deploy/aws-private-pilot/runtime/supervisor.sh"), "utf8");
  assert.match(caddy, /@wrong_host/u);
  assert.match(caddy, /@tunnel_health/u);
  assert.doesNotMatch(caddy, /origin.auth|ORIGIN_AUTH|alb.health/iu);
  assert.match(caddy, /identity_management/u);
  assert.doesNotMatch(caddy, /gotenberg:3000|chromium/iu);
  assert.match(unit, /docker compose/u);
  assert.match(unit, /Environment=DOCKER_CONFIG=\/etc\/aviasurveil360\/private-pilot\/docker/u);
  assert.match(unit, /Environment=AWS_ECR_DISABLE_CACHE=true/u);
  assert.match(unit, /Environment=AWS_USE_DUALSTACK_ENDPOINT=true/u);
  assert.match(unit, /ipv6-preflight\.sh runtime/u);
  assert.match(unit, /RuntimeDirectory=aviasurveil360-private-pilot/u);
  assert.match(unit, /NoNewPrivileges=yes/u);
  assert.match(unit, /ProtectSystem=strict/u);
  assert.doesNotMatch(unit, /install|curl|aws /u);
  assert.match(tunnelUnit, /supervisor\.sh materialize-tunnel-token/u);
  assert.match(tunnelUnit, /supervisor\.sh validate-tunnel/u);
  assert.match(tunnelUnit, /supervisor\.sh run-tunnel/u);
  assert.match(tunnelUnit, /Restart=always/u);
  assert.match(tunnelUnit, /NoNewPrivileges=yes/u);
  assert.match(tunnelUnit, /ProtectSystem=strict/u);
  assert.match(healthUnit, /supervisor\.sh tunnel-health/u);
  assert.match(healthTimer, /OnUnitActiveSec=1min/u);
  assert.match(supervisor, /--network host/u);
  assert.match(supervisor, /--platform linux\/arm64/u);
  assert.match(supervisor, /--edge-ip-version 6/u);
  assert.match(supervisor, /--protocol auto/u);
  assert.match(supervisor, /--cap-drop ALL/u);
  assert.match(supervisor, /--security-opt no-new-privileges:true/u);
  assert.match(supervisor, /--pids-limit 128/u);
  assert.match(supervisor, /--memory-swap 128m/u);
  assert.match(supervisor, /CloudflaredTunnelHAConnections/u);
  assert.match(supervisor, /connections.*-lt 4/u);
  const runtimeValidation = /validate-runtime\)([\s\S]*?)\n    ;;/u.exec(supervisor)?.[1] ?? "";
  assert.doesNotMatch(runtimeValidation, /AVIA_APP_MIGRATION_PASSWORD_FILE|AVIA_DATABASE_BOOTSTRAP_PASSWORD_FILE|AVIA_KEYCLOAK_REALM_FILE/u);
  assert.doesNotMatch(supervisor, /docker\.sock/u);
});
