import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
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
