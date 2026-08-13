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
  for (const service of ["api", "worker"]) {
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
  const provider = serviceBlock("preprod-auth");
  assert.match(
    api,
    /AVIA_OIDC_ISSUER_URL:\s+["']?https:\/\/localhost:\$\{AVIA_LOCAL_HTTPS_PORT:-8443\}\/identity["']?/,
  );
  assert.match(
    api,
    /AVIA_OIDC_DISCOVERY_URL:\s+http:\/\/preprod-auth:8080/,
  );
  assert.match(api, /AVIA_OIDC_DISCOVERY_PRIVATE_NETWORK:\s*["']true["']/);
  assert.match(provider, /AVIA_AUTH_ADMIN_HTTP_ADDRESS:\s+0\.0\.0\.0:8081/u);
  assert.doesNotMatch(provider, /ports:/u);

  const initializer = read("scripts/init-local-preprod-namespace.sh");
  assert.match(initializer, /preprod_auth_signing_key/u);
  assert.match(initializer, /preprod_auth_admin_secret/u);
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
    "postgres",
    "preprod-auth-postgres",
    "preprod-auth-mailpit",
    "preprod-auth",
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
    "preprod-auth",
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
  assert.match(
    script,
    /assert_service_networks preprod-auth "identity preprod-identity preprod-identity-database"/u,
    "runtime checker must preserve the provider's exact private network reachability",
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
