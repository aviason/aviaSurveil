import assert from "node:assert/strict";
import {
  chmodSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { test } from "node:test";

const read = (path) => readFileSync(path, "utf8");

const serviceBlock = (compose, name) =>
  compose.match(
    new RegExp(
      `^  ${name}:\\n(?<service>(?:^    .*\\n|^\\n)*)`,
      "mu",
    ),
  )?.groups?.service ?? "";

test("preprod loader is a separate one-shot artifact with no normal runtime surface", () => {
  const dockerfile = read("apps/api/Dockerfile");
  const compose = read("deploy/local/compose.yaml");
  const script = read("scripts/load-preprod-data.sh");
  const apiMain = read("apps/api/cmd/api/main.go");
  const normalProfile = read("apps/api/cmd/api/profile_normal.go");
  const loaderService = serviceBlock(compose, "preprod-data-loader");

  assert.match(
    dockerfile,
    /go build[^]*?-o \/out\/preprod-data-loader \.\/cmd\/preprod-data-loader/u,
  );
  assert.match(
    dockerfile,
    /FROM runtime-base AS preprod-data-loader[^]*?\/out\/preprod-data-loader/u,
  );
  assert.match(compose, /^\s{2}preprod-data-loader:\s*$/mu);
  assert.match(loaderService, /profiles:\s*\[local-preprod-loader\]/u);
  assert.match(loaderService, /target:\s*preprod-data-loader/u);
  assert.match(loaderService, /preprod_loader_authorization/u);
  assert.match(
    loaderService,
    /target:\s*\/var\/lib\/aviasurveil360-preprod-control/u,
  );
  assert.doesNotMatch(
    loaderService,
    /profiles:\s*\[[^\]]*(?:demo|full|test|recovery)/u,
  );

  assert.match(script, /umask 077/u);
  assert.match(script, /chmod 600/u);
  assert.match(script, /local-preprod-loader/u);
  assert.doesNotMatch(script, /--(?:token|authorization-token)\b/u);
  assert.doesNotMatch(
    script,
    /export\s+\w*(?:TOKEN|AUTHORIZATION_TOKEN)=/u,
  );

  assert.doesNotMatch(apiMain, /preproddata|preprod-data-loader/iu);
  assert.doesNotMatch(normalProfile, /preproddata|preprod-data-loader/iu);
});

test("AGA candidate demo loader has only the isolated PostgreSQL capability", () => {
  const dockerfile = read("apps/api/Dockerfile");
  const compose = read("deploy/local/compose.yaml");
  const wrapper = read("scripts/load-aga-candidate-demo.sh");
  const service = serviceBlock(compose, "preprod-aga-candidate-demo-loader");

  assert.match(dockerfile, /preprod-aga-candidate-demo-loader \.\/cmd\/preprod-aga-candidate-demo-loader/u);
  assert.match(service, /profiles:\s*\[aga-candidate-demo\]/u);
  assert.match(service, /preprod-postgres:[^]*?condition:\s*service_healthy/u);
  assert.match(service, /preprod_aga_demo_writer_database_password/u);
  assert.match(service, /preprod-app-database/u);
  assert.doesNotMatch(service, /keycloak|minio|mailpit|smtp|worker|scheduler|object|queue|lifecycle/iu);
  assert.match(wrapper, /umask 077/u);
  assert.match(wrapper, /run\/config\/aga-demo\.json/u);
  assert.match(wrapper, /run\/input\/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01\.zip/u);
  assert.doesNotMatch(wrapper, /AUTHORIZATION_TOKEN|--token/iu);
});

test("tagged AGA demo API uses separate normal and sealed-reader credentials", () => {
  const compose = read("deploy/local/compose.yaml");
  const service = serviceBlock(compose, "preprod-aga-demo-api");
  assert.match(service, /profiles:\s*\[preproddemo\]/u);
  assert.match(service, /target:\s*preprod-aga-demo-api/u);
  assert.match(service, /AVIA_DATABASE_USER:\s*preprod_normal_api/u);
  assert.match(service, /preprod_aga_demo_reader_database_password/u);
  assert.match(service, /AVIA_AGA_DEMO_DATABASE_URL/u);
  assert.match(service, /AVIA_OIDC_ISSUER_URL/u);
  assert.match(service, /preprod-keycloak/u);
  assert.doesNotMatch(service, /preprod_aga_demo_writer_database_password|minio|mailpit|smtp|worker|scheduler|queue/u);
});

test("disposable OIDC qualification is a separate predecessor fixture", () => {
  const compose = read("deploy/local/compose.yaml");
  const fixture = serviceBlock(compose, "preprod-aga-demo-oidc-fixture");
  const loader = serviceBlock(compose, "preprod-aga-candidate-demo-loader");
  assert.match(fixture, /aga-candidate-demo-oidc-fixture/u);
  assert.match(fixture, /preprod-keycloak/u);
  assert.match(fixture, /preprod_aga_demo_oidc_qualification_password/u);
  assert.doesNotMatch(fixture, /preprod_aga_demo_(?:reader|writer)_database_password/u);
  assert.doesNotMatch(loader, /keycloak|oidc|qualification|session_encryption/u);
});

test("one-shot profile owns a complete isolated disposable namespace", () => {
  const compose = read("deploy/local/compose.yaml");
  const namespaceInitializer = read("scripts/init-local-preprod-namespace.sh");
  const objectInitializer = read("deploy/local/minio/preprod-init.sh");
  const realmBuilder = read("deploy/local/keycloak/build-realm.mjs");
  const loaderService = serviceBlock(compose, "preprod-data-loader");

  for (const serviceName of [
    "preprod-postgres",
    "preprod-migration",
    "preprod-keycloak-postgres",
    "preprod-keycloak",
    "preprod-mailpit-volume-init",
    "preprod-mailpit",
    "preprod-minio-volume-init",
    "preprod-minio",
  ]) {
    const block = serviceBlock(compose, serviceName);
    assert.notEqual(block, "", `${serviceName} must exist`);
    assert.match(
      block,
      /profiles:\s*\[local-preprod-loader\]/u,
      `${serviceName} must be opt-in`,
    );
  }

  assert.match(loaderService, /preprod-migration:\s*\n\s+condition: service_completed_successfully/u);
  assert.match(loaderService, /preprod-keycloak:\s*\n\s+condition: service_healthy/u);
  assert.match(loaderService, /preprod-mailpit:\s*\n\s+condition: service_healthy/u);
  assert.match(loaderService, /preprod-minio:\s*\n\s+condition: service_healthy/u);
  assert.match(
    loaderService,
    /type:\s*bind[^]*?AVIA_PREPROD_CONTROL_STORE_DIR/u,
  );
  assert.doesNotMatch(loaderService, /preprod-loader-control-store:/u);

  for (const volume of [
    "preprod-app-database",
    "preprod-keycloak-database",
    "preprod-object-store",
    "preprod-mailpit-data",
  ]) {
    assert.match(compose, new RegExp(`^  ${volume}:\\s*$`, "mu"));
  }
  assert.match(
    compose,
    /preprod_keycloak_realm:[^]*?AVIA_PREPROD_STATE_DIR/u,
  );
  assert.match(
    namespaceInitializer,
    /outside-disposable-target/u,
  );
  assert.doesNotMatch(namespaceInitializer, /--rotate|mv -f/u);
  assert.match(
    namespaceInitializer,
    /--smtp-user aviasurveil360-preprod/u,
  );
  assert.match(realmBuilder, /realm\.smtpServer\.user = smtpUser/u);
  assert.match(objectInitializer, /aviasurveil360-local-preprod/u);
  assert.match(objectInitializer, /runs\/\*/u);
  assert.doesNotMatch(objectInitializer, /evidence-quarantine|evidence-clean/u);
});

test("preprod object policy scopes only ListBucket with the run prefix", () => {
  const initializer = read("deploy/local/minio/preprod-init.sh");
  const policySource = initializer.match(
    /cat >\/tmp\/preprod-loader-policy\.json <<'EOF'\n(?<policy>[\s\S]*?)\nEOF/u,
  )?.groups?.policy;
  assert.ok(policySource, "preprod loader policy is missing");
  const policy = JSON.parse(policySource);
  const statements = policy.Statement;
  const actionList = (statement) =>
    Array.isArray(statement.Action) ? statement.Action : [statement.Action];
  const location = statements.find((statement) =>
    actionList(statement).includes("s3:GetBucketLocation"));
  const list = statements.find((statement) =>
    actionList(statement).includes("s3:ListBucket"));

  assert.deepEqual(actionList(location), ["s3:GetBucketLocation"]);
  assert.equal(location.Condition, undefined);
  assert.deepEqual(actionList(list), ["s3:ListBucket"]);
  assert.deepEqual(list.Condition, {
    StringLike: {
      "s3:prefix": ["runs/*"],
    },
  });
});

test("preprod migration uses the supported database-only config mode", () => {
  const compose = read("deploy/local/compose.yaml");
  const migration = serviceBlock(compose, "preprod-migration");
  assert.match(migration, /AVIA_ENVIRONMENT:\s*development/u);
  assert.match(
    migration,
    /AVIA_DATABASE_NAME:\s*aviasurveil360_local_preprod/u,
  );
  assert.match(
    migration,
    /AVIA_DATABASE_USER:\s*aviasurveil360_preprod_loader/u,
  );
  assert.doesNotMatch(
    migration,
    /AVIA_(?:ENABLE_CANONICAL|TEST_PRINCIPAL|TEST_SESSION)/u,
  );
});

test("loader source declares exact target, authorization, and append-only boundaries", () => {
  const loader = [
    read("apps/api/internal/preproddata/loader.go"),
    read("apps/api/internal/preproddata/manifest.go"),
    read("apps/api/internal/preproddata/authorization.go"),
  ].join("\n");
  const controlStore = read("apps/api/internal/preproddata/control_store.go");
  const commandMain = read("apps/api/cmd/preprod-data-loader/main.go");

  for (const marker of [
    "local-preprod",
    "LOAD_EMPTY_TARGET",
    "RESUME_RUN",
    "DROP_RECREATE_TARGET",
    "PostgresSystemIdentifier",
    "KeycloakRealm",
    "KeycloakDatabase",
    "KeycloakServiceClientID",
    "MailpitNamespace",
    "ObjectBucket",
    "ObjectPrefix",
    "IntentDigest",
  ]) {
    assert.match(loader, new RegExp(marker, "u"));
  }

  assert.match(loader, /type CommandBoundary interface/u);
  assert.match(loader, /type CommandStream interface/u);
  assert.match(loader, /Next\(context\.Context\)/u);
  assert.match(loader, /Preflight/u);
  assert.match(loader, /Apply/u);
  assert.match(loader, /Reconcile/u);
  assert.doesNotMatch(loader, /Commands\s+\[\]AuthoritativeCommand/u);
  assert.doesNotMatch(loader, /DELETE\s+FROM/iu);
  assert.doesNotMatch(loader, /delete.*run.?id/iu);

  assert.match(controlStore, /O_CREATE\s*\|\s*os\.O_EXCL/u);
  assert.match(controlStore, /authorization[^]*?sha256/iu);
  assert.match(controlStore, /intent/u);
  assert.match(controlStore, /result/u);
  assert.match(controlStore, /checkpoint/u);
  assert.match(controlStore, /cleanup/u);
  assert.doesNotMatch(controlStore, /os\.Truncate|O_TRUNC/u);

  assert.match(commandMain, /AuthorizationFile/u);
  assert.match(commandMain, /ControlStoreDirectory/u);
  assert.doesNotMatch(commandMain, /flag\.\w+\([^)]*(?:token|secret)/iu);
});

test("normal artifact guard positively excludes the new loader package", () => {
  const boundary = read("scripts/test-normal-artifact-boundary.sh");
  assert.match(boundary, /internal\/preproddata/u);
  assert.match(boundary, /preprod-data-loader/u);
  assert.match(
    boundary,
    /normal_packages=\([^]*?"\.\/cmd\/api"[^]*?"\.\/cmd\/worker"[^]*?"\.\/cmd\/reminder-scheduler"[^]*?"\.\/cmd\/migrate"/u,
  );
});

test("normal untagged Go test graph excludes canonical-test-only integration files", () => {
  const root = mkdtempSync(path.join(tmpdir(), "avia-normal-go-graph-"));
  try {
    const goCache = path.join(root, "go-cache");
    const goTmp = path.join(root, "go-tmp");
    mkdirSync(goCache);
    mkdirSync(goTmp);
    const result = spawnSync(
      "go",
      [
        "-C",
        "apps/api",
        "test",
        "-run",
        "^$",
        "./tests/integration",
      ],
      {
        cwd: process.cwd(),
        encoding: "utf8",
        env: {
          ...process.env,
          GOCACHE: goCache,
          GOTMPDIR: goTmp,
        },
      },
    );
    assert.equal(
      result.status,
      0,
      `stdout=${result.stdout}\nstderr=${result.stderr}`,
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("cleanup recorder runs offline without restarting disposable dependencies", () => {
  const root = mkdtempSync(path.join(tmpdir(), "avia-task7-cleanup-wrapper-"));
  try {
    const dockerArgs = path.join(root, "docker-args.txt");
    const configuration = path.join(root, "loader-config.json");
    const authorization = path.join(root, "cleanup-authorization.json");
    const seed = path.join(root, "seed");
    const controlStore = path.join(root, "control-store");
    mkdirSync(controlStore, { mode: 0o700 });
    writeFileSync(configuration, "{}\n", { mode: 0o600 });
    writeFileSync(authorization, "{}\n", { mode: 0o600 });
    writeFileSync(seed, "synthetic-test-seed\n", { mode: 0o600 });
    writeFileSync(
      path.join(root, "docker"),
      "#!/bin/sh\nprintf '%s\\n' \"$@\" >\"$AVIA_FAKE_DOCKER_ARGS_FILE\"\n",
      { mode: 0o700 },
    );
    chmodSync(root, 0o700);
    // The script resolves docker through PATH, so expose only the fake executable.
    const result = spawnSync(
      "bash",
      ["scripts/load-preprod-data.sh", "record-cleanup"],
      {
        cwd: process.cwd(),
        encoding: "utf8",
        env: {
          ...process.env,
          AVIA_FAKE_DOCKER_ARGS_FILE: dockerArgs,
          AVIA_PREPROD_LOADER_CONFIG_FILE: configuration,
          AVIA_PREPROD_AUTHORIZATION_FILE: authorization,
          AVIA_PREPROD_SEED_FILE: seed,
          AVIA_PREPROD_CONTROL_STORE_DIR: controlStore,
          PATH: `${root}:${process.env.PATH}`,
        },
      },
    );
    assert.equal(
      result.status,
      0,
      `stdout=${result.stdout}\nstderr=${result.stderr}`,
    );
    const argumentsSeen = readFileSync(dockerArgs, "utf8").trim().split("\n");
    assert.equal(argumentsSeen[0], "run", argumentsSeen.join(" "));
    assert.equal(argumentsSeen.includes("compose"), false, argumentsSeen.join(" "));
    assert.ok(argumentsSeen.includes("--network"), argumentsSeen.join(" "));
    assert.ok(argumentsSeen.includes("none"), argumentsSeen.join(" "));
    assert.ok(
      argumentsSeen.includes(
        `${controlStore}:/var/lib/aviasurveil360-preprod-control:rw`,
      ),
      argumentsSeen.join(" "),
    );
    assert.deepEqual(
      argumentsSeen.slice(-2),
      ["record-cleanup", "/run/config/preprod-loader.json"],
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});
