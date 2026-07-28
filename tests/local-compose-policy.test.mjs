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
import { test } from "node:test";
import { fileURLToPath, pathToFileURL } from "node:url";
import path from "node:path";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const composePath = path.join(repositoryRoot, "deploy/local/compose.yaml");
const testComposePath = path.join(
  repositoryRoot,
  "deploy/local/compose.test.yaml",
);
const lockPath = path.join(repositoryRoot, "deploy/local/image-lock.json");
const policyPath = path.join(repositoryRoot, "deploy/local/compose-policy.json");
const validatorPath = path.join(
  repositoryRoot,
  "scripts/lib/local-compose-policy.mjs",
);
const secretInitializerPath = path.join(
  repositoryRoot,
  "scripts/init-local-secrets.sh",
);
const oidcHarnessPath = path.join(
  repositoryRoot,
  "scripts/test-http-oidc-profile.sh",
);
const workerEntrypointPath = path.join(
  repositoryRoot,
  "deploy/local/worker/entrypoint.sh",
);
const policyCheckerPath = path.join(
  repositoryRoot,
  "scripts/check-compose-policy.sh",
);
const sopsConfigPath = path.join(repositoryRoot, ".sops.yaml");
const encryptedApplicationConfigPath = path.join(
  repositoryRoot,
  "deploy/local/config/application.enc.yaml",
);
const oidcVerificationPaths = [
  "apps/api/tests/integration/oidc_keycloak_test.go",
  "apps/web/tests/e2e/oidc-session.spec.ts",
  "apps/web/tests/e2e/oidc-mfa-provisioning.spec.ts",
];

async function loadValidator() {
  assert.ok(
    existsSync(validatorPath),
    "local Compose policy validator must exist",
  );
  return import(pathToFileURL(validatorPath));
}

function secureFixture() {
  return {
    profile: "full",
    compose: {
      services: {
        gateway: {
          image: "local/aviasurveil360-gateway:plan3",
          user: "10001:10001",
          read_only: true,
          ports: [
            {
              host_ip: "127.0.0.1",
              published: "8443",
              target: 8443,
            },
          ],
          networks: ["edge", "frontend"],
          healthcheck: { test: ["CMD", "/usr/bin/healthcheck"] },
        },
        postgres: {
          image:
            "postgres:17.6-alpine3.22@sha256:ef257d85f76e48da1c64832459b59fcaba1a4dac97bf5d7450c77753542eee94",
          user: "70:70",
          read_only: true,
          environment: {
            POSTGRES_PASSWORD_FILE: "/run/secrets/app_database_password",
          },
          secrets: [{ source: "app_database_password" }],
          volumes: [
            {
              source: "app_database",
              target: "/var/lib/postgresql/data",
              type: "volume",
            },
          ],
          networks: ["database"],
          healthcheck: { test: ["CMD", "pg_isready"] },
        },
        "keycloak-postgres": {
          image:
            "postgres:17.6-alpine3.22@sha256:ef257d85f76e48da1c64832459b59fcaba1a4dac97bf5d7450c77753542eee94",
          user: "70:70",
          read_only: true,
          environment: {
            POSTGRES_PASSWORD_FILE:
              "/run/secrets/keycloak_database_password",
          },
          secrets: [{ source: "keycloak_database_password" }],
          volumes: [
            {
              source: "keycloak_database",
              target: "/var/lib/postgresql/data",
              type: "volume",
            },
          ],
          networks: ["identity-database"],
          healthcheck: { test: ["CMD", "pg_isready"] },
        },
      },
      networks: {
        edge: {},
        frontend: { internal: true },
        database: { internal: true },
        "identity-database": { internal: true },
      },
    },
    lock: {
      images: {
        postgres: {
          reference:
            "postgres:17.6-alpine3.22@sha256:ef257d85f76e48da1c64832459b59fcaba1a4dac97bf5d7450c77753542eee94",
        },
      },
    },
    policy: {
      browserPublishedPorts: {
        gateway: [{ hostIp: "127.0.0.1", published: 8443, target: 8443 }],
      },
      databaseIsolation: {
        applicationService: "postgres",
        identityService: "keycloak-postgres",
        dataTarget: "/var/lib/postgresql/data",
      },
      externalImageServices: ["postgres", "keycloak-postgres"],
      healthcheckRequiredServices: [
        "gateway",
        "postgres",
        "keycloak-postgres",
      ],
      profileServices: {
        full: ["gateway", "postgres", "keycloak-postgres"],
      },
      publicNetworkServices: ["gateway"],
      secretMounts: {
        postgres: ["app_database_password"],
        "keycloak-postgres": ["keycloak_database_password"],
      },
      writableRootfsExceptions: {},
    },
  };
}

async function expectViolation(code, mutate) {
  const { validateComposePolicy } = await loadValidator();
  const fixture = structuredClone(secureFixture());
  mutate(fixture);
  const violations = validateComposePolicy(fixture);
  assert.ok(
    violations.some((violation) => violation.code === code),
    `expected ${code}, received ${JSON.stringify(violations)}`,
  );
}

test("rejects plaintext secret values", async () => {
  await expectViolation("PLAINTEXT_SECRET", ({ compose }) => {
    compose.services.postgres.environment.POSTGRES_PASSWORD =
      "unsafe-default-password";
  });
});

test("OIDC verification contains no committed plaintext test password", () => {
  const forbiddenPassword = [
    "Local",
    "Inspector",
    "Pass",
    "123",
    "!",
  ].join("");
  for (const relativePath of oidcVerificationPaths) {
    assert.equal(
      readFileSync(path.join(repositoryRoot, relativePath), "utf8").includes(
        forbiddenPassword,
      ),
      false,
      `${relativePath} must obtain credentials from a task-owned runtime environment`,
    );
  }
});

test("rejects missing required secret mounts", async () => {
  await expectViolation("MISSING_SECRET_MOUNT", ({ compose }) => {
    compose.services.postgres.secrets = [];
  });
});

test("rejects external images without immutable digests", async () => {
  await expectViolation("UNPINNED_EXTERNAL_IMAGE", ({ compose }) => {
    compose.services.postgres.image = "postgres:17.6-alpine3.22";
  });
});

test("rejects published internal service ports", async () => {
  await expectViolation("PUBLISHED_INTERNAL_PORT", ({ compose }) => {
    compose.services.postgres.ports = [
      { host_ip: "127.0.0.1", published: "5432", target: 5432 },
    ];
  });
});

test("rejects shared application and identity database storage", async () => {
  await expectViolation("SHARED_DATABASE", ({ compose }) => {
    compose.services["keycloak-postgres"].volumes[0].source =
      "app_database";
  });
});

test("rejects internal services attached to unrestricted networks", async () => {
  await expectViolation("UNRESTRICTED_NETWORK", ({ compose }) => {
    compose.services.postgres.networks.push("edge");
  });
});

test("allows only the exact reviewed egress network for a named service", async () => {
  const { validateComposePolicy } = await loadValidator();
  const fixture = structuredClone(secureFixture());
  fixture.compose.networks.signature_updates = {};
  fixture.compose.services.postgres.networks.push("signature_updates");
  fixture.policy.egressNetworkServices = {
    postgres: ["signature_updates"],
  };
  const approved = validateComposePolicy(fixture);
  assert.equal(
    approved.some((violation) => violation.code === "UNRESTRICTED_NETWORK"),
    false,
    JSON.stringify(approved),
  );

  fixture.compose.services.postgres.networks.push("unreviewed_egress");
  fixture.compose.networks.unreviewed_egress = {};
  const rejected = validateComposePolicy(fixture);
  assert.ok(
    rejected.some(
      (violation) =>
        violation.code === "UNRESTRICTED_NETWORK" &&
        violation.message.includes("unreviewed_egress"),
    ),
    JSON.stringify(rejected),
  );
});

test("rejects root runtime users", async () => {
  await expectViolation("ROOT_USER", ({ compose }) => {
    compose.services.postgres.user = "0";
  });
});

test("rejects runtime services without an explicit non-root user", async () => {
  await expectViolation("ROOT_USER", ({ compose }) => {
    delete compose.services.postgres.user;
  });
});

test("rejects writable root filesystems without a reviewed reason", async () => {
  await expectViolation("WRITABLE_ROOTFS_EXCEPTION", ({ compose }) => {
    compose.services.postgres.read_only = false;
  });
});

test("rejects missing service health checks", async () => {
  await expectViolation("MISSING_HEALTHCHECK", ({ compose }) => {
    delete compose.services.postgres.healthcheck;
  });
});

test("rejects missing services from the selected profile topology", async () => {
  await expectViolation("MISSING_PROFILE_SERVICE", ({ compose }) => {
    delete compose.services.postgres;
  });
});

test("rejects unexpected services in the selected profile topology", async () => {
  await expectViolation("UNEXPECTED_PROFILE_SERVICE", ({ compose }) => {
    compose.services["debug-console"] = {
      image: "local/debug-console:plan3",
      user: "10001:10001",
      read_only: true,
      networks: ["frontend"],
      healthcheck: { test: ["CMD", "/usr/bin/healthcheck"] },
    };
  });
});

test("the canonical Compose profiles satisfy the secure local policy", async () => {
  assert.ok(existsSync(composePath), "deploy/local/compose.yaml must exist");
  assert.ok(existsSync(lockPath), "deploy/local/image-lock.json must exist");
  assert.ok(existsSync(policyPath), "deploy/local/compose-policy.json must exist");

  const { validateComposePolicy } = await loadValidator();
  const lock = JSON.parse(readFileSync(lockPath, "utf8"));
  const policy = JSON.parse(readFileSync(policyPath, "utf8"));

  for (const profile of ["demo", "full", "test", "recovery", "tools"]) {
    const rendered = execFileSync(
      "docker",
      [
        "compose",
        "--file",
        composePath,
        "--profile",
        profile,
        "config",
        "--format",
        "json",
      ],
      { cwd: repositoryRoot, encoding: "utf8" },
    );
    const compose = JSON.parse(rendered);
    const violations = validateComposePolicy({
      compose,
      lock,
      policy,
      profile,
    });
    assert.deepEqual(
      violations,
      [],
      `${profile} profile policy violations:\n${JSON.stringify(violations, null, 2)}`,
    );
  }
});

test("the secret initializer is private, non-destructive, and explicitly rotatable", (t) => {
  assert.ok(
    existsSync(secretInitializerPath),
    "scripts/init-local-secrets.sh must exist",
  );
  const localState = mkdtempSync(
    path.join(os.tmpdir(), "aviasurveil360-local-secrets-"),
  );
  t.after(() => rmSync(localState, { recursive: true, force: true }));
  const environment = {
    ...process.env,
    AVIASURVEIL_LOCAL_STATE_DIR: localState,
  };
  const expectedSecretPatterns = new Map([
    ["app_database_password", /^[a-f0-9]{64}$/],
    ["keycloak_bootstrap_admin_password", /^[a-f0-9]{64}$/],
    ["keycloak_database_password", /^[a-f0-9]{64}$/],
    ["keycloak_service_client_secret", /^[a-f0-9]{64}$/],
    ["minio_root_password", /^[a-f0-9]{64}$/],
    ["minio_root_user", /^[a-f0-9]{20}$/],
    ["oidc_client_secret", /^[a-f0-9]{64}$/],
    ["session_encryption_key", /^[A-Za-z0-9+/]{43}=$/],
    ["smtp_auth_file", /^aviasurveil360:[a-f0-9]{64}$/],
    ["smtp_password", /^[a-f0-9]{64}$/],
  ]);

  execFileSync("sh", [secretInitializerPath], {
    cwd: repositoryRoot,
    env: environment,
    stdio: "pipe",
  });
  const secretDirectory = path.join(localState, "secrets");
  const runtimeRealmPath = path.join(localState, "keycloak", "realm.json");
  const initialValues = new Map();
  for (const [filename, expectedPattern] of expectedSecretPatterns) {
    const filenamePath = path.join(secretDirectory, filename);
    assert.ok(existsSync(filenamePath), `${filename} must be generated`);
    assert.equal(
      statSync(filenamePath).mode & 0o777,
      0o600,
      `${filename} must be mode 0600`,
    );
    const value = readFileSync(filenamePath, "utf8").trim();
    assert.match(value, expectedPattern);
    initialValues.set(filename, value);
  }
  assert.equal(statSync(secretDirectory).mode & 0o777, 0o700);
  assert.equal(
    initialValues.get("smtp_auth_file"),
    `aviasurveil360:${initialValues.get("smtp_password")}`,
  );
  assert.ok(existsSync(runtimeRealmPath), "runtime Keycloak realm must be generated");
  assert.equal(statSync(runtimeRealmPath).mode & 0o777, 0o600);
  const initialRuntimeRealm = JSON.parse(
    readFileSync(runtimeRealmPath, "utf8"),
  );
  const initialWebClient = initialRuntimeRealm.clients.find(
    ({ clientId }) => clientId === "aviasurveil360-web",
  );
  assert.equal(
    initialWebClient.secret,
    initialValues.get("oidc_client_secret"),
  );
  const initialServiceClient = initialRuntimeRealm.clients.find(
    ({ clientId }) => clientId === "aviasurveil360-lifecycle",
  );
  assert.equal(
    initialServiceClient.secret,
    initialValues.get("keycloak_service_client_secret"),
  );
  assert.equal(
    JSON.stringify(initialRuntimeRealm).includes(
      "__AVIA_OIDC_CLIENT_SECRET__",
    ),
    false,
  );

  assert.throws(
    () =>
      execFileSync("sh", [secretInitializerPath], {
        cwd: repositoryRoot,
        env: environment,
        stdio: "pipe",
      }),
    /Command failed/,
  );

  execFileSync("sh", [secretInitializerPath, "--rotate"], {
    cwd: repositoryRoot,
    env: environment,
    stdio: "pipe",
  });
  for (const filename of expectedSecretPatterns.keys()) {
    const rotated = readFileSync(
      path.join(secretDirectory, filename),
      "utf8",
    ).trim();
    assert.notEqual(rotated, initialValues.get(filename));
  }
  const rotatedRuntimeRealm = JSON.parse(
    readFileSync(runtimeRealmPath, "utf8"),
  );
  const rotatedWebClient = rotatedRuntimeRealm.clients.find(
    ({ clientId }) => clientId === "aviasurveil360-web",
  );
  assert.equal(
    rotatedWebClient.secret,
    readFileSync(
      path.join(secretDirectory, "oidc_client_secret"),
      "utf8",
    ).trim(),
  );
  const rotatedServiceClient = rotatedRuntimeRealm.clients.find(
    ({ clientId }) => clientId === "aviasurveil360-lifecycle",
  );
  assert.equal(
    rotatedServiceClient.secret,
    readFileSync(
      path.join(secretDirectory, "keycloak_service_client_secret"),
      "utf8",
    ).trim(),
  );
  assert.notEqual(rotatedWebClient.secret, initialWebClient.secret);
  assert.equal(
    readFileSync(path.join(secretDirectory, "smtp_auth_file"), "utf8").trim(),
    `aviasurveil360:${readFileSync(
      path.join(secretDirectory, "smtp_password"),
      "utf8",
    ).trim()}`,
  );
});

test("Mailpit SMTP is private, authenticated, and reachable by the worker and Keycloak", (t) => {
  const localState = mkdtempSync(
    path.join(os.tmpdir(), "aviasurveil360-mailpit-compose-"),
  );
  t.after(() => rmSync(localState, { recursive: true, force: true }));
  const environment = {
    ...process.env,
    AVIASURVEIL_LOCAL_STATE_DIR: localState,
  };
  execFileSync("sh", [secretInitializerPath], {
    cwd: repositoryRoot,
    env: environment,
    stdio: "pipe",
  });
  const rendered = execFileSync(
    "docker",
    [
      "compose",
      "--file",
      composePath,
      "--profile",
      "full",
      "config",
      "--format",
      "json",
    ],
    { cwd: repositoryRoot, encoding: "utf8", env: environment },
  );
  const compose = JSON.parse(rendered);
  const worker = compose.services.worker;
  const keycloak = compose.services.keycloak;
  const mailpit = compose.services.mailpit;
  assert.equal(worker.environment.AVIA_SMTP_ADDRESS, "mailpit:1025");
  assert.equal(
    worker.environment.AVIA_SMTP_FROM,
    "no-reply@aviasurveil360.local",
  );
  assert.equal(worker.environment.AVIA_SMTP_USERNAME, "aviasurveil360");
  assert.equal(
    worker.environment.AVIA_SMTP_PASSWORD_FILE,
    "/run/secrets/smtp_password",
  );
  assert.equal(worker.environment.AVIA_SMTP_PRIVATE_NETWORK, "true");
  assert.equal("AVIA_SMTP_PASSWORD" in worker.environment, false);
  assert.ok(worker.depends_on.mailpit);
  assert.equal(
    worker.secrets.some(({ source }) => source === "smtp_password"),
    true,
  );
  assert.equal(
    compose.services.api.secrets.some(
      ({ source }) => source === "smtp_password",
    ),
    false,
  );
  assert.deepEqual(
    Object.entries(compose.services)
      .filter(([, service]) =>
        (service.secrets ?? []).some(
          ({ source }) => source === "smtp_password",
        ),
      )
      .map(([serviceName]) => serviceName),
    ["worker"],
  );
  assert.equal(
    mailpit.environment.MP_SMTP_AUTH_FILE,
    "/run/secrets/smtp_auth_file",
  );
  assert.equal(mailpit.environment.MP_SMTP_AUTH_ALLOW_INSECURE, "true");
  assert.equal(mailpit.environment.MP_SMTP_DISABLE_RDNS, "true");
  assert.equal(
    mailpit.secrets.some(({ source }) => source === "smtp_auth_file"),
    true,
  );
  assert.equal("ports" in mailpit, false);
  assert.equal("identity" in mailpit.networks, true);
  assert.equal("identity" in keycloak.networks, true);
  assert.equal("platform" in mailpit.networks, true);
  assert.equal("platform" in worker.networks, true);
  const toolsRendered = execFileSync(
    "docker",
    [
      "compose",
      "--file",
      composePath,
      "--profile",
      "tools",
      "config",
      "--format",
      "json",
    ],
    { cwd: repositoryRoot, encoding: "utf8", env: environment },
  );
  const toolsCompose = JSON.parse(toolsRendered);
  const toolsPorts = toolsCompose.services["mailpit-tools"].ports;
  assert.deepEqual(
    toolsPorts.map(({ host_ip, published, target }) => ({
      hostIp: host_ip,
      published: Number(published),
      target,
    })),
    [
      { hostIp: "127.0.0.1", published: 1025, target: 1025 },
      { hostIp: "127.0.0.1", published: 8025, target: 8025 },
    ],
  );
});

test("the full profile imports only the generated realm and reads Keycloak credentials from mounted files", (t) => {
  const localState = mkdtempSync(
    path.join(os.tmpdir(), "aviasurveil360-keycloak-compose-"),
  );
  t.after(() => rmSync(localState, { recursive: true, force: true }));
  const environment = {
    ...process.env,
    AVIASURVEIL_LOCAL_STATE_DIR: localState,
  };
  execFileSync("sh", [secretInitializerPath], {
    cwd: repositoryRoot,
    env: environment,
    stdio: "pipe",
  });

  const rendered = execFileSync(
    "docker",
    [
      "compose",
      "--file",
      composePath,
      "--profile",
      "full",
      "config",
      "--format",
      "json",
    ],
    {
      cwd: repositoryRoot,
      encoding: "utf8",
      env: environment,
    },
  );
  const compose = JSON.parse(rendered);
  const keycloak = compose.services.keycloak;
  assert.deepEqual(keycloak.entrypoint, [
    "/bin/sh",
    "/opt/aviasurveil360/keycloak-entrypoint.sh",
  ]);
  assert.ok(keycloak.command.includes("--import-realm"));
  assert.equal(
    keycloak.environment.KC_DB_PASSWORD_FILE,
    "/run/secrets/keycloak_database_password",
  );
  assert.equal(
    keycloak.environment.KC_BOOTSTRAP_ADMIN_PASSWORD_FILE,
    "/run/secrets/keycloak_bootstrap_admin_password",
  );
  assert.equal("KC_DB_PASSWORD" in keycloak.environment, false);
  assert.equal(
    "KC_BOOTSTRAP_ADMIN_PASSWORD" in keycloak.environment,
    false,
  );

  const secretTargets = new Map(
    keycloak.secrets.map(({ source, target }) => [source, target]),
  );
  assert.equal(
    secretTargets.get("keycloak_realm"),
    "/opt/keycloak/data/import/realm.json",
  );
  assert.equal(
    compose.secrets.keycloak_realm.file,
    path.join(localState, "keycloak", "realm.json"),
  );
  const entrypointConfig = keycloak.configs.find(
    ({ source }) => source === "keycloak_entrypoint",
  );
  assert.equal(
    entrypointConfig?.target,
    "/opt/aviasurveil360/keycloak-entrypoint.sh",
  );
  assert.match(
    keycloak.healthcheck.test.join(" "),
    /\/identity\/health\/ready/u,
  );

  const worker = compose.services.worker;
  assert.equal(
    worker.environment.AVIA_KEYCLOAK_ADMIN_URL,
    "http://keycloak:8080/identity",
  );
  assert.equal(worker.environment.AVIA_KEYCLOAK_REALM, "aviasurveil360");
  assert.equal(
    worker.environment.AVIA_KEYCLOAK_SERVICE_CLIENT_ID,
    "aviasurveil360-lifecycle",
  );
  assert.equal(
    worker.environment.AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET_FILE,
    "/run/secrets/keycloak_service_client_secret",
  );
  assert.equal(
    "AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET" in worker.environment,
    false,
  );
  assert.equal("AVIA_KEYCLOAK_ADMIN_USERNAME" in worker.environment, false);
  assert.equal("AVIA_KEYCLOAK_ADMIN_PASSWORD_FILE" in worker.environment, false);
  const workerEntrypoint = readFileSync(workerEntrypointPath, "utf8");
  assert.match(
    workerEntrypoint,
    /AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET_FILE/u,
  );
  assert.match(
    workerEntrypoint,
    /export AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET/u,
  );
});

test("the OIDC harness uses generated secrets and production-mode Keycloak", () => {
  const source = readFileSync(testComposePath, "utf8");
  assert.match(source, /^\s*keycloak-postgres:\s*$/mu);
  assert.match(source, /image:\s+aviasurveil360\/keycloak:local/u);
  assert.match(source, /target:\s+keycloak/u);
  assert.match(source, /KC_DB_PASSWORD_FILE:/u);
  assert.match(source, /KC_BOOTSTRAP_ADMIN_PASSWORD_FILE:/u);
  assert.match(source, /keycloak_realm/u);
  assert.match(source, /--optimized/u);
  assert.doesNotMatch(source, /\bstart-dev\b/u);
  assert.doesNotMatch(
    source,
    /^\s*(?:POSTGRES_PASSWORD|KC_BOOTSTRAP_ADMIN_PASSWORD|MINIO_ROOT_PASSWORD):/mu,
  );
  assert.doesNotMatch(source, /\.\/keycloak\/realm\.json/u);

  const harness = readFileSync(oidcHarnessPath, "utf8");
  assert.match(harness, /scan_runtime_artifacts_for_secret_leaks/u);
  assert.match(harness, /\brg\s+--fixed-strings\s+--quiet\b/u);
  assert.match(harness, /docker-runtime\.log/u);
  assert.doesNotMatch(
    harness,
    /\brg\s+--fixed-strings\s+--line-number\b/u,
  );
});

test("the policy checker never prints a secret-bearing match", () => {
  const checker = readFileSync(policyCheckerPath, "utf8");
  assert.match(checker, /\brg\s+--quiet\b/u);
  assert.doesNotMatch(checker, /\brg\s+--line-number\b/u);
});

test("the SOPS rule covers the maintained source and produces no plaintext config", () => {
  const sopsConfig = readFileSync(sopsConfigPath, "utf8");
  const pathRule = sopsConfig.match(/path_regex:\s*(.+)/)?.[1]?.trim();
  assert.ok(pathRule, "SOPS creation rule must declare path_regex");
  assert.match(
    "deploy/local/config/application.enc.yaml",
    new RegExp(pathRule),
  );
  assert.doesNotMatch(
    "deploy/local/config/application.example.yaml",
    new RegExp(pathRule),
  );
  assert.ok(
    existsSync(encryptedApplicationConfigPath),
    "encrypted application config must exist",
  );
  const encryptedConfig = readFileSync(encryptedApplicationConfigPath, "utf8");
  assert.match(encryptedConfig, /^version:\s+ENC\[/m);
  assert.match(encryptedConfig, /^http:\s*\n\s+listenAddress:\s+ENC\[/m);
  assert.match(encryptedConfig, /^\s+publicOrigin:\s+ENC\[/m);
  assert.doesNotMatch(encryptedConfig, /https:\/\/localhost:8443/);
  assert.match(encryptedConfig, /^sops:\s*$/m);
});
