import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import path from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const read = (relativePath) => readFileSync(path.join(repositoryRoot, relativePath), "utf8");

function serviceBlock(compose, name) {
  return compose.match(new RegExp(`^  ${name}:\\n(?<service>(?:^    .*\\n|^\\n)*)`, "mu"))?.groups?.service ?? "";
}

test("canonical local-preprod owns one isolated first-party identity namespace", () => {
  const compose = read("deploy/local/compose.yaml");
  for (const name of [
    "preprod-auth-postgres",
    "preprod-auth-mailpit-volume-init",
    "preprod-auth-mailpit",
    "preprod-auth",
    "preprod-canonical-aga-loader",
    "preprod-canonical-demo-identity-loader",
  ]) {
    const block = serviceBlock(compose, name);
    assert.notEqual(block, "", name);
    assert.match(block, /local-preprod-loader/u, name);
  }
  assert.doesNotMatch(compose, new RegExp(["key", "cloak"].join(""), "iu"));
  assert.match(compose, /^  preprod-auth-database:\s*$/mu);
  assert.match(compose, /^  preprod-auth-mailpit-data:\s*$/mu);
});

test("public identity routing strips the issuer prefix and never exposes admin port", () => {
  for (const relativePath of [
    "deploy/local/gateway/Caddyfile.preprod",
    "deploy/local/gateway/Caddyfile.preprod.http",
    "deploy/local/gateway/Caddyfile",
  ]) {
    const gateway = read(relativePath);
    assert.match(gateway, /handle_path \/identity\/\*/u, relativePath);
    assert.match(gateway, /reverse_proxy (?:preprod-auth|\{env\.AVIA_IDENTITY_UPSTREAM\}):8080/u, relativePath);
    assert.doesNotMatch(gateway, /8081|\/admin/u, relativePath);
  }
});

test("first-party runtime has separate public and internal admin listeners", () => {
  const compose = JSON.parse(execFileSync("docker", [
    "compose",
    "--file",
    path.join(repositoryRoot, "deploy/local/compose.yaml"),
    "--profile",
    "local-preprod-loader",
    "config",
    "--format",
    "json",
  ], { cwd: repositoryRoot, encoding: "utf8" }));
  const provider = compose.services["preprod-auth"];
  assert.equal(provider.environment.AVIA_AUTH_HTTP_ADDRESS, "0.0.0.0:8080");
  assert.equal(provider.environment.AVIA_AUTH_ADMIN_HTTP_ADDRESS, "0.0.0.0:8081");
  assert.equal(provider.environment.AVIA_AUTH_ISSUER_URL, "https://localhost:8445/identity");
  assert.equal("ports" in provider, false);
  assert.deepEqual(Object.keys(provider.networks).sort(), [
    "identity",
    "preprod-identity",
    "preprod-identity-database",
  ]);
});

test("namespace initializer generates all privileged auth material create-only", () => {
  const initializer = read("scripts/init-local-preprod-namespace.sh");
  for (const marker of [
    "preprod_auth_database_password",
    "preprod_auth_signing_key",
    "preprod_auth_data_encryption_key",
    "preprod_auth_mfa_key",
    "preprod_auth_admin_secret",
    "preprod_oidc_client_secret",
    "preprod_auth_smtp_password",
    "preprod_auth_mailpit_cert",
    "preprod_auth_mailpit_key",
    "identityProvider\": \"first-party",
  ]) {
    assert.ok(initializer.includes(marker), marker);
  }
  assert.match(initializer, /chmod 0400/u);
  assert.doesNotMatch(initializer, /--rotate/u);
});

test("canonical API waits for private discovery before startup", () => {
  const service = serviceBlock(read("deploy/local/compose.yaml"), "preprod-api");
  assert.match(service, /AVIA_OIDC_DISCOVERY_URL:\s*http:\/\/preprod-auth:8080/u);
  assert.match(service, /oidc_wait_attempt=0/u);
  assert.match(service, /openid-configuration/u);
  assert.ok(service.indexOf("oidc_wait_attempt=0") < service.indexOf("exec /app/api"));
});

test("canonical API and worker use non-owner application credentials and first-party admin", () => {
  const compose = read("deploy/local/compose.yaml");
  for (const name of ["preprod-api", "preprod-worker"]) {
    const service = serviceBlock(compose, name);
    assert.match(service, /preprod-normal-runtime-role-provisioner:[^]*?service_completed_successfully/u, name);
    assert.match(service, /AVIA_DATABASE_USER:\s*preprod_normal_api/u, name);
    assert.match(service, /preprod_normal_api_database_password/u, name);
    assert.doesNotMatch(service, /preprod_app_database_password/u, name);
    assert.match(service, /AVIA_FIRST_PARTY_ADMIN_URL:\s*http:\/\/preprod-auth:8081/u, name);
    assert.match(service, /AVIA_FIRST_PARTY_ADMIN_SECRET_FILE:\s*\/run\/secrets\/preprod_auth_admin_secret/u, name);
  }
});

test("canonical synthetic identity loader is one-shot and authority-aware", () => {
  const dockerfile = read("apps/api/Dockerfile");
  const loader = serviceBlock(read("deploy/local/compose.yaml"), "preprod-canonical-demo-identity-loader");
  assert.match(dockerfile, /\.\/cmd\/preprod-canonical-demo-identity-loader/u);
  assert.match(dockerfile, /AS preprod-canonical-demo-identity-loader/u);
  assert.match(loader, /preprod-auth:[^]*?condition: service_healthy/u);
  assert.match(loader, /preprod-auth-mailpit:[^]*?condition: service_healthy/u);
  assert.match(loader, /AVIA_FIRST_PARTY_ADMIN_URL/u);
  assert.match(loader, /preprod_auth_admin_secret/u);
  assert.match(loader, /restart:\s*"no"/u);
});

test("preprod object policy limits loader access to the run prefix", () => {
  const initializer = read("deploy/local/minio/preprod-init.sh");
  const source = initializer.match(/cat >\/tmp\/preprod-loader-policy\.json <<'EOF'\n(?<policy>[\s\S]*?)\nEOF/u)?.groups?.policy;
  assert.ok(source);
  const policy = JSON.parse(source);
  const list = policy.Statement.find((statement) => {
    const actions = Array.isArray(statement.Action) ? statement.Action : [statement.Action];
    return actions.includes("s3:ListBucket");
  });
  assert.deepEqual(list.Condition, { StringLike: { "s3:prefix": ["runs/*"] } });
  assert.deepEqual(list.Resource, ["arn:aws:s3:::aviasurveil360-local-preprod"]);
});
