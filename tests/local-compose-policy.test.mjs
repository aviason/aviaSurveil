import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import {
  existsSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  statSync,
} from "node:fs";
import os from "node:os";
import path from "node:path";
import { test } from "node:test";
import { fileURLToPath, pathToFileURL } from "node:url";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const composePath = path.join(repositoryRoot, "deploy/local/compose.yaml");
const lockPath = path.join(repositoryRoot, "deploy/local/image-lock.json");
const policyPath = path.join(repositoryRoot, "deploy/local/compose-policy.json");
const validatorPath = path.join(repositoryRoot, "scripts/lib/local-compose-policy.mjs");
const localInitializer = path.join(repositoryRoot, "scripts/init-local-secrets.sh");
const authInitializer = path.join(repositoryRoot, "scripts/init-local-preprod-namespace.sh");

async function validator() {
  assert.ok(existsSync(validatorPath));
  return import(pathToFileURL(validatorPath));
}

function secureFixture() {
  const databaseImage = "postgres:17.6-alpine3.22@sha256:ef257d85f76e48da1c64832459b59fcaba1a4dac97bf5d7450c77753542eee94";
  return {
    profile: "full",
    compose: {
      services: {
        gateway: {
          image: "local/gateway",
          user: "10001:10001",
          read_only: true,
          ports: [{ host_ip: "127.0.0.1", published: "8443", target: 8443 }],
          networks: ["edge"],
          healthcheck: { test: ["CMD", "/healthcheck"] },
        },
        postgres: {
          image: databaseImage,
          user: "70:70",
          read_only: true,
          environment: { POSTGRES_PASSWORD_FILE: "/run/secrets/app_database_password" },
          secrets: [{ source: "app_database_password" }],
          volumes: [{ source: "app_database", target: "/var/lib/postgresql/data", type: "volume" }],
          networks: ["database"],
          healthcheck: { test: ["CMD", "pg_isready"] },
        },
        "auth-postgres": {
          image: databaseImage,
          user: "70:70",
          read_only: true,
          environment: { POSTGRES_PASSWORD_FILE: "/run/secrets/auth_database_password" },
          secrets: [{ source: "auth_database_password" }],
          volumes: [{ source: "auth_database", target: "/var/lib/postgresql/data", type: "volume" }],
          networks: ["auth-database"],
          healthcheck: { test: ["CMD", "pg_isready"] },
        },
      },
      networks: {
        edge: {},
        database: { internal: true },
        "auth-database": { internal: true },
      },
    },
    lock: { images: { postgres: { reference: databaseImage } } },
    policy: {
      browserPublishedPorts: {
        gateway: [{ hostIp: "127.0.0.1", published: 8443, target: 8443 }],
      },
      databaseIsolation: {
        applicationService: "postgres",
        identityService: "auth-postgres",
        dataTarget: "/var/lib/postgresql/data",
      },
      externalImageServices: ["postgres", "auth-postgres"],
      healthcheckRequiredServices: ["gateway", "postgres", "auth-postgres"],
      profileServices: { full: ["gateway", "postgres", "auth-postgres"] },
      publicNetworkServices: ["gateway"],
      secretMounts: {
        postgres: ["app_database_password"],
        "auth-postgres": ["auth_database_password"],
      },
      writableRootfsExceptions: {},
    },
  };
}

async function expectViolation(code, mutate) {
  const { validateComposePolicy } = await validator();
  const fixture = structuredClone(secureFixture());
  mutate(fixture);
  const violations = validateComposePolicy(fixture);
  assert.ok(violations.some((entry) => entry.code === code), JSON.stringify(violations));
}

test("policy rejects plaintext secrets, unpinned images, and internal ports", async () => {
  await expectViolation("PLAINTEXT_SECRET", ({ compose }) => {
    compose.services.postgres.environment.POSTGRES_PASSWORD = "unsafe";
  });
  await expectViolation("UNPINNED_EXTERNAL_IMAGE", ({ compose }) => {
    compose.services.postgres.image = "postgres:17";
  });
  await expectViolation("PUBLISHED_INTERNAL_PORT", ({ compose }) => {
    compose.services.postgres.ports = [{ host_ip: "127.0.0.1", published: "5432", target: 5432 }];
  });
});

test("policy rejects missing mounts, shared databases, and root runtimes", async () => {
  await expectViolation("MISSING_SECRET_MOUNT", ({ compose }) => {
    compose.services.postgres.secrets = [];
  });
  await expectViolation("SHARED_DATABASE", ({ compose }) => {
    compose.services["auth-postgres"].volumes[0].source = "app_database";
  });
  await expectViolation("ROOT_USER", ({ compose }) => {
    compose.services.gateway.user = "0:0";
  });
});

test("all maintained local profiles satisfy the checked Compose policy", async () => {
  const { validateComposePolicy } = await validator();
  const lock = JSON.parse(readFileSync(lockPath, "utf8"));
  const policy = JSON.parse(readFileSync(policyPath, "utf8"));
  for (const profile of ["demo", "full", "test", "recovery", "tools"]) {
    const compose = JSON.parse(execFileSync("docker", [
      "compose", "--file", composePath, "--profile", profile, "config", "--format", "json",
    ], { cwd: repositoryRoot, encoding: "utf8" }));
    assert.deepEqual(
      validateComposePolicy({ compose, lock, policy, profile }),
      [],
      `${profile} policy mismatch`,
    );
  }
});

test("local application secrets are private, create-safe, and rotatable", (t) => {
  const state = mkdtempSync(path.join(os.tmpdir(), "avia-local-secrets-"));
  t.after(() => rmSync(state, { recursive: true, force: true }));
  const env = { ...process.env, AVIASURVEIL_LOCAL_STATE_DIR: state };
  execFileSync("sh", [localInitializer], { cwd: repositoryRoot, env });
  const directory = path.join(state, "secrets");
  assert.equal(statSync(directory).mode & 0o777, 0o700);
  const required = [
    "app_database_password",
    "minio_api_access_key",
    "minio_api_secret_key",
    "session_encryption_key",
    "smtp_auth_file",
    "smtp_password",
  ];
  const original = new Map();
  for (const name of required) {
    const filename = path.join(directory, name);
    assert.equal(statSync(filename).mode & 0o777, 0o600);
    original.set(name, readFileSync(filename, "utf8"));
  }
  assert.throws(() => execFileSync("sh", [localInitializer], { cwd: repositoryRoot, env }));
  execFileSync("sh", [localInitializer, "--rotate"], { cwd: repositoryRoot, env });
  for (const [name, value] of original) {
    assert.notEqual(readFileSync(path.join(directory, name), "utf8"), value);
  }
});

test("first-party auth namespace creates separate keys, admin secret, and STARTTLS material", (t) => {
  const state = mkdtempSync(path.join(os.tmpdir(), "avia-auth-secrets-"));
  t.after(() => rmSync(state, { recursive: true, force: true }));
  const env = { ...process.env, AVIA_PREPROD_STATE_DIR: state };
  execFileSync("sh", [authInitializer], { cwd: repositoryRoot, env });
  const directory = path.join(state, "secrets");
  for (const name of [
    "preprod_auth_database_password",
    "preprod_auth_database_url",
    "preprod_auth_signing_key",
    "preprod_auth_data_encryption_key",
    "preprod_auth_mfa_key",
    "preprod_auth_admin_secret",
    "preprod_oidc_client_secret",
    "preprod_auth_smtp_auth_file",
    "preprod_auth_mailpit_ca",
    "preprod_auth_mailpit_cert",
    "preprod_auth_mailpit_key",
  ]) {
    assert.equal(statSync(path.join(directory, name)).mode & 0o777, 0o400, name);
  }
  assert.throws(() => execFileSync("sh", [authInitializer], { cwd: repositoryRoot, env }));
  assert.equal(JSON.parse(readFileSync(path.join(state, "namespace.json"), "utf8")).identityProvider, "first-party");
});

test("first-party provider has split public/admin listeners and no public admin route", () => {
  const compose = JSON.parse(execFileSync("docker", [
    "compose", "--file", composePath, "--profile", "full", "config", "--format", "json",
  ], { cwd: repositoryRoot, encoding: "utf8" }));
  const provider = compose.services["preprod-auth"];
  assert.equal(provider.environment.AVIA_AUTH_HTTP_ADDRESS, "0.0.0.0:8080");
  assert.equal(provider.environment.AVIA_AUTH_ADMIN_HTTP_ADDRESS, "0.0.0.0:8081");
  assert.equal("ports" in provider, false);
  assert.equal(provider.environment.AVIA_AUTH_ADMIN_SECRET_FILE, "/run/secrets/preprod_auth_admin_secret");
  const gateway = readFileSync(path.join(repositoryRoot, "deploy/local/gateway/Caddyfile"), "utf8");
  assert.match(gateway, /handle_path \/identity\/\*/u);
  assert.match(gateway, /reverse_proxy preprod-auth:8080/u);
  assert.doesNotMatch(gateway, /8081|\/admin/u);
});

test("API and worker use only the first-party administration boundary", () => {
  const compose = JSON.parse(execFileSync("docker", [
    "compose", "--file", composePath, "--profile", "full", "config", "--format", "json",
  ], { cwd: repositoryRoot, encoding: "utf8" }));
  for (const name of ["api", "worker"]) {
    const service = compose.services[name];
    assert.equal(service.environment.AVIA_FIRST_PARTY_ADMIN_URL, "http://preprod-auth:8081");
    assert.equal(service.environment.AVIA_FIRST_PARTY_ADMIN_SECRET_FILE, "/run/secrets/preprod_auth_admin_secret");
    assert.equal(service.environment.AVIA_OIDC_ISSUER_URL, "https://localhost:8443/identity");
  }
  assert.equal(compose.services.api.environment.AVIA_OIDC_DISCOVERY_URL, "http://preprod-auth:8080");
});
