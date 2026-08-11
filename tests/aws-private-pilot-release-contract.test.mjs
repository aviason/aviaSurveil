import assert from "node:assert/strict";
import { chmodSync, cpSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import path from "node:path";
import os from "node:os";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";

import {
  REQUIRED_BINDINGS,
  REQUIRED_SUBJECTS,
  digestPath,
  validateReleaseManifest,
} from "../scripts/lib/aws-private-pilot-release.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const command = path.join(root, "scripts/aws-private-pilot-release.sh");
const digest = (character) => `sha256:${character.repeat(64)}`;

const bindingPaths = {
  compose: "deploy/aws-private-pilot/compose.yaml",
  gatewayConfig: "deploy/aws-private-pilot/gateway/Caddyfile",
  systemdUnit: "deploy/aws-private-pilot/systemd/aviasurveil360-private-pilot.service",
  tunnelSystemdUnit: "deploy/aws-private-pilot/systemd/aviasurveil360-private-pilot-tunnel.service",
  tunnelHealthSystemdUnit: "deploy/aws-private-pilot/systemd/aviasurveil360-private-pilot-tunnel-health.service",
  tunnelHealthTimer: "deploy/aws-private-pilot/systemd/aviasurveil360-private-pilot-tunnel-health.timer",
  supervisor: "deploy/aws-private-pilot/runtime/supervisor.sh",
  ipv6Preflight: "deploy/aws-private-pilot/runtime/ipv6-preflight.sh",
  appEntrypoint: "deploy/aws-private-pilot/runtime/app-entrypoint.sh",
  databaseBootstrap: "deploy/aws-private-pilot/runtime/database-bootstrap.sh",
  keycloakEntrypoint: "deploy/aws-private-pilot/runtime/keycloak-entrypoint.sh",
  migrationSet: "apps/api/migrations",
  decisionContract: "docs/operations/AWS_PRIVATE_PILOT_DECISIONS.md",
  infrastructurePolicy: "infra/policies/aws-private-pilot.rego",
};

function fixture(t, options = {}) {
  const directory = path.join(os.tmpdir(), `aws-private-pilot-release-test-${process.pid}-${Math.random().toString(16).slice(2)}`);
  mkdirSync(directory, { recursive: true, mode: 0o700 });
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const fixtureRoot = path.join(directory, "repo");
  mkdirSync(fixtureRoot, { recursive: true, mode: 0o700 });
  cpSync(path.join(root, "scripts"), path.join(fixtureRoot, "scripts"), { recursive: true });
  for (const relativePath of Object.values(bindingPaths)) {
    const sourcePath = path.join(root, relativePath);
    const targetPath = path.join(fixtureRoot, relativePath);
    mkdirSync(path.dirname(targetPath), { recursive: true, mode: 0o700 });
    cpSync(sourcePath, targetPath, { recursive: true });
  }

  const subjects = Object.fromEntries(REQUIRED_SUBJECTS.map((name, index) => {
    const character = (index + 1).toString(16).slice(-1);
    return [name, {
      image: `111122223333.dkr-ecr.eu-central-1.on.aws/${name}@${digest(character)}`,
      platform: "linux/arm64",
      configDigest: digest("a"),
      sbomSha256: digest("b"),
      provenanceSha256: digest("c"),
      vulnerabilityEvidenceSha256: digest("d"),
    }];
  }));

  const lockPath = path.join(fixtureRoot, "fixture-image-lock.json");
  writeFileSync(lockPath, JSON.stringify({ schemaVersion: 1, resolved: true, architecture: "linux/arm64", subjects: Object.fromEntries(Object.entries(subjects).map(([name, subject]) => [name, subject.image])) }));
  chmodSync(lockPath, 0o600);

  const realmPath = path.join(fixtureRoot, "fixture-keycloak-realm.json");
  writeFileSync(realmPath, JSON.stringify({
    realm: "private-pilot",
    enabled: true,
    smtpServer: {
      host: "smtp.example.invalid",
      port: "587",
      from: "identity@example.invalid",
      auth: "true",
      user: "keycloak-private-pilot",
      password: "${KC_SMTP_PASSWORD}",
      starttls: "true",
      ssl: "false",
    },
  }));
  chmodSync(realmPath, 0o600);

  const gatewayArtifactPath = path.join(fixtureRoot, "fixture-gateway-static-artifact");
  mkdirSync(gatewayArtifactPath, { mode: 0o700 });
  writeFileSync(path.join(gatewayArtifactPath, "index.html"), "<!doctype html><html><body>fixture</body></html>\n");
  writeFileSync(path.join(gatewayArtifactPath, "asset-abc123.js"), "console.log('fixture');\n");

  const rdsCABundlePath = path.join(fixtureRoot, "fixture-aws-rds-global-bundle.pem");
  writeFileSync(rdsCABundlePath, "-----BEGIN CERTIFICATE-----\nfixture-public-ca-only\n-----END CERTIFICATE-----\n");
  chmodSync(rdsCABundlePath, 0o600);

  const runtimeEnvironmentPath = path.join(fixtureRoot, "fixture-runtime.env");
  const runtimeLines = Object.entries({
    AVIA_CLOUDFLARED_IMAGE: subjects.cloudflared.image,
    AVIA_GATEWAY_IMAGE: subjects.gateway.image,
    AVIA_API_IMAGE: subjects.api.image,
    AVIA_WORKER_IMAGE: subjects.worker.image,
    AVIA_KEYCLOAK_IMAGE: subjects.keycloak.image,
    AVIA_DATABASE_BOOTSTRAP_IMAGE: subjects["database-bootstrap"].image,
    AVIA_MIGRATION_IMAGE: subjects.migration.image,
    AVIA_AWS_ACCOUNT_ID: "111122223333",
    AVIA_AWS_REGION: "eu-central-1",
    AVIA_INSTANCE_ID: "i-0123456789abcdef0",
    AVIA_FORCE_IPV6: "true",
    AVIA_CLOUDFLARE_EDGE_IP_VERSION: "6",
    AVIA_CLOUDFLARE_EDGE_HOSTS: "region1.v2.argotunnel.com,region2.v2.argotunnel.com",
    AVIA_CLOUDFLARE_TUNNEL_TOKEN_PARAMETER_NAME: "/aviasurveil360/private-pilot/cloudflare/tunnel-token",
    AVIA_CLOUDFLARE_TUNNEL_TOKEN_FILE: "/run/aviasurveil360-private-pilot/secrets/cloudflare_tunnel_token",
    AVIA_RDS_CA_BUNDLE_FILE: "/opt/aviasurveil360/private-pilot/release/aws-rds-global-bundle.pem",
    ...Object.fromEntries([
      "APP_DATABASE_PASSWORD", "APP_MIGRATION_PASSWORD", "DATABASE_BOOTSTRAP_PASSWORD",
      "KEYCLOAK_DATABASE_PASSWORD", "OIDC_CLIENT_SECRET", "SESSION_ENCRYPTION_KEY", "KEYCLOAK_SERVICE_CLIENT_SECRET",
      "APP_SMTP_PASSWORD", "KEYCLOAK_SMTP_PASSWORD", "KEYCLOAK_REALM",
    ].map((name) => [`AVIA_${name}_FILE`, `/run/aviasurveil360-private-pilot/secrets/${name.toLowerCase()}`])),
  }).map(([name, value]) => `${name}=${value}`);
  writeFileSync(runtimeEnvironmentPath, `${runtimeLines.join("\n")}\n`);
  chmodSync(runtimeEnvironmentPath, 0o600);

  const bindings = Object.fromEntries(Object.entries(bindingPaths).map(([name, relativePath]) => [name, { path: relativePath, sha256: digestPath(path.join(fixtureRoot, relativePath)) }]));
  const lockRelativePath = path.relative(fixtureRoot, lockPath);
  bindings.imageLock = { path: lockRelativePath, sha256: digestPath(lockPath) };
  bindings.keycloakRealm = { path: path.relative(fixtureRoot, realmPath), sha256: digestPath(realmPath) };
  bindings.rdsCABundle = { path: path.relative(fixtureRoot, rdsCABundlePath), sha256: digestPath(rdsCABundlePath) };
  bindings.gatewayStaticArtifact = { path: path.relative(fixtureRoot, gatewayArtifactPath), sha256: digestPath(gatewayArtifactPath) };
  bindings.runtimeEnvironment = { path: path.relative(fixtureRoot, runtimeEnvironmentPath), sha256: digestPath(runtimeEnvironmentPath) };
  assert.deepEqual(Object.keys(bindings).sort(), [...REQUIRED_BINDINGS].sort());

  const now = Date.now();
  const manifest = {
    schemaVersion: 1,
    releaseId: "private-pilot-fixture-001",
    issuedAt: new Date(now - 60000).toISOString(),
    expiresAt: new Date(now + 86400000).toISOString(),
    artifactStatus: "candidate-only",
    releasePending: true,
    productionReady: false,
    architecture: "linux/arm64",
    sourceTreeSha256: digest("e"),
    subjects,
    bindings,
    database: {
      latestMigrationVersion: 44,
      forwardOnly: true,
      runtimeRole: "aviasurveil360_runtime",
      migrationRole: "aviasurveil360_migrator",
      migrationRoleLockedAfterUse: true,
    },
    rollback: {
      predecessorManifestSha256: digest("f"),
      nMinusOneCompatible: options.nMinusOneCompatible ?? false,
      compatibilityEvidenceSha256: digest("1"),
      incompatibleAction: "roll-forward-or-coordinated-restore",
    },
    targets: {
      awsProfile: "avia",
      awsAccountId: "111122223333",
      region: "eu-central-1",
      hostname: "pilot.example.invalid",
      instanceId: "i-0123456789abcdef0",
      databaseArn: "arn:aws:rds:eu-central-1:111122223333:db:avia-private-pilot",
      bucketArns: ["quarantine", "canonical", "attachments", "documents"].map((name) => `arn:aws:s3:::fixture-${name}`),
      cloudflareAccountId: "2".repeat(32),
      cloudflareZoneId: "1".repeat(32),
      cloudflareTunnelId: "12345678-1234-4123-8123-123456789abc",
      connectorParameterArn: "arn:aws:ssm:eu-central-1:111122223333:parameter/aviasurveil360/private-pilot/cloudflare/tunnel-token",
      terraformAddresses: ["aws_instance.runtime"],
      retainOrDestroyDecision: "retain",
    },
    authorization: {
      scope: "local-preparation-only",
      remoteActionsAuthorized: false,
      productionReleaseAuthorized: false,
    },
  };
  const manifestPath = path.join(fixtureRoot, "release-manifest.json");
  writeFileSync(manifestPath, JSON.stringify(manifest));
  chmodSync(manifestPath, 0o600);
  return { manifest, manifestPath, runtimeEnvironmentPath, fixtureRoot, commandPath: path.join(fixtureRoot, "scripts/aws-private-pilot-release.sh") };
}

function runRelease(fixtureResult, ...args) {
  return spawnSync(fixtureResult.commandPath, args, { cwd: fixtureResult.fixtureRoot, encoding: "utf8" });
}

test("complete digest-bound release candidate validates entirely offline", (t) => {
  const prepared = fixture(t);
  const { manifest, manifestPath } = prepared;
  assert.deepEqual(validateReleaseManifest(manifest, { root: prepared.fixtureRoot }), []);
  const result = runRelease(prepared, "validate", manifestPath);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /verified locally/u);
  assert.match(result.stdout, /candidate-only/u);
});

test("every command is dry-run by default and execution remains unauthorized", (t) => {
  const prepared = fixture(t);
  const { manifestPath } = prepared;
  for (const action of ["render", "build", "export", "bootstrap", "migrate", "start", "health", "drain", "retain", "destroy"]) {
    const result = runRelease(prepared, action, manifestPath);
    assert.equal(result.status, 0, `${action}: ${result.stderr}`);
    assert.match(result.stdout, /"mode": "dry-run"/u, action);
    assert.match(result.stdout, /"externalEvidence": "not run"/u, action);
  }
  const execute = runRelease(prepared, "start", manifestPath, "--execute");
  assert.equal(execute.status, 77);
  assert.match(execute.stderr, /Task 7 exact authorization/u);
});

test("binary rollback requires explicit N/N-1 compatibility evidence", (t) => {
  const blocked = fixture(t);
  const result = runRelease(blocked, "rollback", blocked.manifestPath);
  assert.equal(result.status, 78);
  assert.match(result.stderr, /roll-forward-or-coordinated-restore/u);

  const compatible = fixture(t, { nMinusOneCompatible: true });
  const allowedPlan = runRelease(compatible, "rollback", compatible.manifestPath);
  assert.equal(allowedPlan.status, 0, allowedPlan.stderr);
  assert.match(allowedPlan.stdout, /predecessor manifest/u);
});

test("changed files, stale input, mutable subjects, and remote authority fail closed", async (t) => {
  const mutations = {
    "changed-binding": [(manifest) => { manifest.bindings.compose.sha256 = digest("9"); }, /changed-binding:compose/u],
    stale: [(manifest) => { manifest.expiresAt = "2020-01-01T00:00:00.000Z"; }, /stale-release-input/u],
    mutable: [(manifest) => { manifest.subjects.api.image = "fixture.invalid/api:latest"; }, /mutable-image:api/u],
    amd64: [(manifest) => { manifest.subjects.worker.platform = "linux/amd64"; }, /wrong-platform:worker/u],
    remote: [(manifest) => { manifest.authorization.remoteActionsAuthorized = true; }, /remote-authority-forbidden/u],
    "default-aws-profile": [(manifest) => { manifest.targets.awsProfile = "default"; }, /aws-operator-profile-must-be-avia/u],
    "malformed-region": [(manifest) => { manifest.targets.region = "["; }, /unresolved-remote-targets/u],
    "cross-account-database": [(manifest) => { manifest.targets.databaseArn = manifest.targets.databaseArn.replace("111122223333", "444455556666"); }, /database-target-mismatch/u],
    "cross-account-connector-parameter": [(manifest) => { manifest.targets.connectorParameterArn = manifest.targets.connectorParameterArn.replace("111122223333", "444455556666"); }, /connector-parameter-target-mismatch/u],
    "extra-release-binding": [(manifest) => { manifest.bindings.unreviewed = manifest.bindings.compose; }, /invalid-release-bindings/u],
    "external-image": [(manifest) => { manifest.subjects.keycloak.image = `quay.io/keycloak/keycloak@${digest("8")}`; }, /non-target-ecr-image:keycloak/u],
    target: [(manifest) => { manifest.targets.instanceId = "pending"; }, /unresolved-remote-targets/u],
  };
  for (const [name, [mutate, expected]] of Object.entries(mutations)) {
    await t.test(name, (inner) => {
      const prepared = fixture(inner);
      const { manifest } = prepared;
      mutate(manifest);
      assert.match(validateReleaseManifest(manifest, { root: prepared.fixtureRoot }).join("\n"), expected);
    });
  }
});

test("Keycloak realm SMTP must use one verified encrypted mode and an environment placeholder", (t) => {
  const prepared = fixture(t);
  const { manifest } = prepared;
  const realmPath = path.join(prepared.fixtureRoot, manifest.bindings.keycloakRealm.path);
  const realm = JSON.parse(readFileSync(realmPath, "utf8"));
  realm.smtpServer.starttls = "false";
  realm.smtpServer.password = "plaintext-fixture-password";
  writeFileSync(realmPath, JSON.stringify(realm));
  manifest.bindings.keycloakRealm.sha256 = digestPath(realmPath);
  assert.ok(validateReleaseManifest(manifest, { root: prepared.fixtureRoot }).includes("insecure-keycloak-smtp-contract"));
});

test("runtime environment rejects AWS profiles, static credentials, and changed image subjects", async (t) => {
  for (const [name, line, expected] of [
    ["default-profile", "AWS_PROFILE=default", "runtime-aws-profile-or-static-credential-forbidden"],
    ["named-runtime-profile", "AWS_PROFILE=avia", "runtime-aws-profile-or-static-credential-forbidden"],
    ["static-key", "AWS_ACCESS_KEY_ID=fixture", "runtime-aws-profile-or-static-credential-forbidden"],
    ["provider-redirect", "AWS_CONTAINER_CREDENTIALS_FULL_URI=http://127.0.0.1/credentials", "runtime-aws-profile-or-static-credential-forbidden"],
    ["metadata-redirect", "AWS_EC2_METADATA_SERVICE_ENDPOINT=http://127.0.0.1/credentials", "runtime-aws-profile-or-static-credential-forbidden"],
    ["embedded-secret", "AVIA_SMTP_PASSWORD=fixture-secret", "runtime-aws-profile-or-static-credential-forbidden"],
    ["embedded-tunnel-token", "AVIA_CLOUDFLARE_TUNNEL_TOKEN=fixture-secret", "runtime-aws-profile-or-static-credential-forbidden"],
    ["changed-image", `AVIA_API_IMAGE=fixture.invalid/api@${digest("9")}`, "invalid-runtime-environment"],
  ]) {
    await t.test(name, (inner) => {
      const prepared = fixture(inner);
      const { manifest, runtimeEnvironmentPath } = prepared;
      writeFileSync(runtimeEnvironmentPath, `${readFileSync(runtimeEnvironmentPath, "utf8")}${line}\n`);
      manifest.bindings.runtimeEnvironment.sha256 = digestPath(runtimeEnvironmentPath);
      const errors = validateReleaseManifest(manifest, { root: prepared.fixtureRoot });
      assert.ok(
        errors.includes(expected) || (name === "changed-image" && errors.includes("runtime-image-mismatch:api")),
        errors.join(", "),
      );
    });
  }
});

test("database and supervisor scripts preserve bounded privilege and health ordering", () => {
  const bootstrap = readFileSync(path.join(root, "deploy/aws-private-pilot/runtime/database-bootstrap.sh"), "utf8");
  const app = readFileSync(path.join(root, "deploy/aws-private-pilot/runtime/app-entrypoint.sh"), "utf8");
  const compose = readFileSync(path.join(root, "deploy/aws-private-pilot/compose.yaml"), "utf8");
  const release = readFileSync(path.join(root, "scripts/aws-private-pilot-release.mjs"), "utf8");
  const supervisor = readFileSync(path.join(root, "deploy/aws-private-pilot/runtime/supervisor.sh"), "utf8");
  const unit = readFileSync(path.join(root, "deploy/aws-private-pilot/systemd/aviasurveil360-private-pilot.service"), "utf8");
  const tunnelUnit = readFileSync(path.join(root, "deploy/aws-private-pilot/systemd/aviasurveil360-private-pilot-tunnel.service"), "utf8");
  const runtimeCommands = [
    "apps/api/cmd/api/main.go",
    "apps/api/cmd/worker/main.go",
  ].map((file) => readFileSync(path.join(root, file), "utf8"));

  assert.match(bootstrap, /aviasurveil360_owner NOLOGIN/u);
  assert.match(bootstrap, /aviasurveil360_migrator LOGIN/u);
  assert.match(bootstrap, /ALTER ROLE aviasurveil360_migrator NOLOGIN PASSWORD NULL/u);
  assert.ok((bootstrap.match(/ALTER ROLE aviasurveil360_migrator NOLOGIN PASSWORD NULL/gu) ?? []).length >= 2);
  assert.doesNotMatch(bootstrap, /DATABASE aviasurveil360 OWNER aviasurveil360_runtime/u);
  assert.doesNotMatch(bootstrap, /--set=(?:app|migration|keycloak)_password/u);
  assert.doesNotMatch(app, /scheduler\)/u);
  assert.doesNotMatch(app, /sslmode=disable/u);
  assert.match(bootstrap, /\\getenv app_password AVIA_BOOTSTRAP_APP_PASSWORD/u);
  assert.match(app, /app_migration_password/u);
  assert.match(app, /sslrootcert=/u);
  assert.match(app, /pool_max_conns=/u);
  assert.match(app, /integer from 1 through 8/u);
  assert.match(app, /32-128 URL-safe characters/u);
  assert.match(compose, /AVIA_DATABASE_USER: aviasurveil360_migrator/u);
  assert.match(compose, /required reviewed AWS RDS CA bundle/u);
  assert.match(release, /AVIA_DATABASE_BOOTSTRAP_MODE=lockdown/u);
  assert.match(release, /AVIA_DATABASE_BOOTSTRAP_MODE=migration-enable/u);
  assert.ok(release.indexOf("AVIA_DATABASE_BOOTSTRAP_MODE=migration-enable") < release.indexOf("--profile migration run --rm migration"));
  assert.ok(release.indexOf("--profile migration run --rm migration") < release.indexOf("AVIA_DATABASE_BOOTSTRAP_MODE=lockdown"));
  assert.doesNotMatch(release, /\/usr\/local\/libexec/u);
  assert.match(supervisor, /gateway api worker keycloak/u);
  assert.match(supervisor, /private-pilot\/scripts\/aws-private-pilot-release\.sh/u);
  assert.match(supervisor, /required root-owned private-pilot file is not mode 0600/u);
  assert.doesNotMatch(supervisor, /AVIA_DATA_FEED_/u);
  assert.match(unit, /ExecStartPre=.*supervisor\.sh validate-runtime/u);
  assert.match(unit, /ExecStartPre=.*ipv6-preflight\.sh runtime/u);
  assert.match(unit, /ExecStartPost=.*supervisor\.sh health/u);
  assert.match(tunnelUnit, /supervisor\.sh materialize-tunnel-token/u);
  assert.match(tunnelUnit, /supervisor\.sh run-tunnel/u);
  for (const source of runtimeCommands) {
    assert.match(source, /RuntimeProfile != "aws-private-pilot"/u);
    assert.match(source, /RequiredMigrationVersion: migrations\.LatestVersion/u);
  }
});

test("the committed JSON schema carries immutable and non-production labels", () => {
  const schema = readFileSync(path.join(root, "deploy/aws-private-pilot/release-manifest.schema.json"), "utf8");
  assert.match(schema, /candidate-only/u);
  assert.match(schema, /linux\/arm64/u);
  assert.match(schema, /sha256:\[0-9a-f\]\{64\}/u);
  assert.match(schema, /remoteActionsAuthorized/u);
});
