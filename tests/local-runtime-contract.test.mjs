import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  chmodSync,
  existsSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import path from "node:path";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const read = (relativePath) =>
  readFileSync(path.join(repositoryRoot, relativePath), "utf8");

const compose = read("deploy/local/compose.yaml");
const stackScriptPath = path.join(repositoryRoot, "scripts/local-stack.sh");
const runtimeCheckPath = path.join(
  repositoryRoot,
  "scripts/check-local-runtime.sh",
);
const failureE2EPath = path.join(
  repositoryRoot,
  "apps/web/tests/e2e/local-service-failures.http.spec.ts",
);

function serviceBlock(service) {
  const match = compose.match(
    new RegExp(
      `^  ${service}:\\n([\\s\\S]*?)(?=^  [a-z][a-z0-9-]*:\\n|^configs:\\n)`,
      "m",
    ),
  );
  assert.ok(match, `Compose service ${service} must exist`);
  return match[0];
}

test("full runtime waits for one-shot migration and named required dependencies", () => {
  assert.match(compose, /^\s{2}migration:\s*$/m);
  assert.match(compose, /migration:\s*\n(?:.*\n)*?\s+condition: service_completed_successfully/m);
  for (const service of ["api", "worker", "scheduler"]) {
    assert.match(
      serviceBlock(service),
      /depends_on:[\s\S]*?migration:[\s\S]*?condition: service_completed_successfully/,
      `${service} must wait for the one-shot migration`,
    );
  }
  assert.match(serviceBlock("api"), /health\/ready/);
});

test("full OIDC runtime separates public issuer from private discovery", () => {
  const api = serviceBlock("api");
  const keycloak = serviceBlock("keycloak");
  const scheduler = serviceBlock("scheduler");
  assert.match(
    api,
    /AVIA_OIDC_ISSUER_URL:\s+["']?https:\/\/localhost:\$\{AVIA_LOCAL_HTTPS_PORT:-8443\}\/identity\/realms\/aviasurveil360["']?/,
  );
  assert.match(
    api,
    /AVIA_OIDC_DISCOVERY_URL:\s+http:\/\/keycloak:8080\/identity\/realms\/aviasurveil360/,
  );
  assert.match(api, /AVIA_OIDC_DISCOVERY_PRIVATE_NETWORK:\s*["']true["']/);
  assert.match(keycloak, /--hostname-backchannel-dynamic=true/);
  assert.doesNotMatch(scheduler, /OIDC|oidc|session_encryption_key/);

  const initializer = read("scripts/init-local-secrets.sh");
  assert.match(initializer, /AVIA_LOCAL_HTTPS_PORT/);
  assert.match(initializer, /--public-origin/);
});

test("runtime services declare bounded shutdown, restart, resources, and process limits", () => {
  assert.match(compose, /stop_grace_period:\s*15s/);
  assert.match(compose, /pids_limit:\s*256/);
  assert.match(compose, /resources:\s*\n\s+limits:\s*\n\s+cpus:\s*["']?2/);
  for (const service of [
    "gateway",
    "web-demo",
    "web-http",
    "api",
    "worker",
    "scheduler",
    "postgres",
    "keycloak-postgres",
    "keycloak",
    "minio",
    "clamav",
    "mailpit",
    "gotenberg",
  ]) {
    assert.match(
      serviceBlock(service),
      /restart: unless-stopped/,
      `${service} must declare restart behavior`,
    );
  }
});

test("stack wrapper owns only a validated unique project and cleanup scope", () => {
  assert.ok(existsSync(stackScriptPath), "scripts/local-stack.sh must exist");
  const script = read("scripts/local-stack.sh");
  for (const required of [
    "AVIA_LOCAL_PROJECT",
    "aviasurveil360-task-",
    "--project-name",
    "--remove-orphans",
    "down",
    "status",
  ]) {
    assert.ok(script.includes(required), `stack wrapper omitted ${required}`);
  }
  assert.doesNotMatch(script, /docker\s+(?:stop|rm|volume rm|network rm)\s+\$\(/);
  assert.doesNotMatch(script, /docker\s+system\s+prune/);
});

test("runtime checker covers failure, leakage, isolation, and residue contracts", () => {
  assert.ok(existsSync(runtimeCheckPath), "runtime checker must exist");
  const script = read("scripts/check-local-runtime.sh");
  for (const required of [
    "postgres",
    "keycloak",
    "minio",
    "clamav",
    "gotenberg",
    "mailpit",
    "worker",
    "health/ready",
    "health/live",
    "docker compose",
    "logs",
    "secret",
    "orphan",
    "published",
    "residue",
    "assert_exact_network_membership",
    "Network membership: exact",
  ]) {
    assert.ok(
      script.toLowerCase().includes(required.toLowerCase()),
      `runtime checker omitted ${required}`,
    );
  }
  assert.doesNotMatch(
    script,
    /compose\s+kill/u,
    "worker crash injection must not be treated as a manual Docker stop",
  );
  assert.match(
    script,
    /compose\s+exec[\s\S]*kill\s+-KILL/u,
    "worker crash injection must kill the application process inside the container",
  );
  assert.match(
    script,
    /\/proc\/1\/task\/1\/children/u,
    "worker crash injection must target the init process's direct application child without a pidof race",
  );
});

test("runtime checker fails closed before evidence collection when rg is unavailable", () => {
  const commandDirectory = mkdtempSync(
    path.join(tmpdir(), "aviasurveil360-runtime-contract-"),
  );
  try {
    for (const command of ["docker", "node"]) {
      const commandPath = path.join(commandDirectory, command);
      writeFileSync(commandPath, "#!/bin/sh\nexit 0\n");
      chmodSync(commandPath, 0o755);
    }
    const result = spawnSync("/bin/bash", [runtimeCheckPath], {
      encoding: "utf8",
      env: {
        PATH: `${commandDirectory}:/usr/bin:/bin`,
      },
    });
    assert.equal(result.status, 69);
    assert.match(result.stderr, /required command is unavailable: rg/);
    assert.doesNotMatch(result.stdout, /verified|passed|zero generated-secret/i);
  } finally {
    rmSync(commandDirectory, { force: true, recursive: true });
  }
});

test("HTTP failure browser scenario is registered and contains no test-route fallback", () => {
  assert.ok(existsSync(failureE2EPath), "local service failure E2E must exist");
  const source = read(
    "apps/web/tests/e2e/local-service-failures.http.spec.ts",
  );
  assert.match(source, /degraded|unavailable/i);
  assert.doesNotMatch(source, /\/__test\//);
  const config = read("apps/web/playwright.config.ts");
  assert.ok(
    config.includes("e2e/local-service-failures.http.spec.ts"),
    "HTTP Playwright project must register service-failure coverage",
  );
});
