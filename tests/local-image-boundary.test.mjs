import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import path from "node:path";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);

const requiredFiles = [
  "apps/web/Dockerfile",
  "deploy/local/webserver/main.go",
  "apps/api/Dockerfile",
  "apps/auth/Dockerfile",
  "deploy/local/gateway/Dockerfile",
  "deploy/local/gateway/Caddyfile",
  "deploy/local/gateway/security-headers.caddy",
  "deploy/local/api/entrypoint.sh",
  "deploy/local/worker/entrypoint.sh",
  "scripts/build-local-images.sh",
  "scripts/generate-image-sboms.sh",
  "scripts/scan-local-images.sh",
];

function read(relativePath) {
  const absolutePath = path.join(repositoryRoot, relativePath);
  assert.ok(existsSync(absolutePath), `${relativePath} must exist`);
  return readFileSync(absolutePath, "utf8");
}

test("declares every Task 2 container and gateway boundary file", () => {
  for (const relativePath of requiredFiles) {
    assert.ok(
      existsSync(path.join(repositoryRoot, relativePath)),
      `${relativePath} must exist`,
    );
  }
});

test("web Dockerfile builds immutable demo and HTTP artifacts separately", () => {
  const dockerfile = read("apps/web/Dockerfile");
  assert.match(dockerfile, /^FROM .+ AS demo-build$/mu);
  assert.match(dockerfile, /^FROM .+ AS http-build$/mu);
  assert.match(dockerfile, /^FROM .+ AS demo$/mu);
  assert.match(dockerfile, /^FROM .+ AS http$/mu);
  assert.match(dockerfile, /\bnpm ci\b/u);
  assert.match(dockerfile, /\bnpm run build:demo\b/u);
  assert.match(dockerfile, /\bnpm run build:http\b/u);
  assert.doesNotMatch(dockerfile, /AVIA_HTTP_TEST_PROFILE=canonical/u);
  assert.match(dockerfile, /\bUSER\s+(?!0\b|root\b)\S+/u);
  assert.doesNotMatch(dockerfile, /\bCOPY\s+--from=.+node_modules\b/u);
});

test("web runtime is a bounded scratch server with an executable health probe", () => {
  const dockerfile = read("apps/web/Dockerfile");
  assert.match(dockerfile, /^FROM \$\{GO_BUILD_IMAGE\} AS web-server-build$/mu);
  assert.match(dockerfile, /^FROM scratch AS web-runtime$/mu);
  assert.match(dockerfile, /\/web-server/u);
  assert.doesNotMatch(dockerfile, /\bbusybox httpd\b/u);

  const server = read("deploy/local/webserver/main.go");
  assert.match(server, /signal\.NotifyContext/u);
  assert.match(server, /syscall\.SIGTERM/u);
  assert.match(server, /\.Shutdown\(/u);
  assert.match(server, /healthcheck/u);
  assert.match(server, /ReadHeaderTimeout/u);
});

test("the existing HTTP artifact contains no mock, seed, or test-profile input", () => {
  const inputManifestPath = path.join(
    repositoryRoot,
    "apps/web/dist/http/build-inputs.json",
  );
  if (!existsSync(inputManifestPath)) {
    return;
  }
  const manifest = JSON.parse(readFileSync(inputManifestPath, "utf8"));
  assert.equal(manifest.profile, "http");
  const inputs = manifest.inputs.join("\n");
  assert.doesNotMatch(inputs, /\/src\/(?:mock|entry\/http-test)(?:\/|\.tsx?$)/u);
  assert.doesNotMatch(inputs, /seed-(?:data|visual-runtime)/u);
  assert.doesNotMatch(inputs, /testprofile|test-profile/iu);
});

test("Go Dockerfile has reproducible api, worker, and migration targets", () => {
  const dockerfile = read("apps/api/Dockerfile");
  for (const target of ["api", "worker", "migration"]) {
    assert.match(dockerfile, new RegExp(`^FROM .+ AS ${target}$`, "mu"));
  }
  assert.match(dockerfile, /CGO_ENABLED=0/u);
  assert.match(dockerfile, /-trimpath/u);
  assert.match(dockerfile, /-buildvcs=false/u);
  assert.match(dockerfile, /-buildid=/u);
  assert.match(dockerfile, /\bUSER\s+(?!0\b|root\b)\S+/u);
  const runtimeStart = dockerfile.indexOf(" AS runtime-base");
  assert.notEqual(runtimeStart, -1, "Go runtime base stage must exist");
  assert.doesNotMatch(
    dockerfile.slice(runtimeStart),
    /\b(?:go build|gcc|git|npm)\b/u,
  );
});

test("runtime entrypoints read secret files without printing values and exec the process", () => {
  for (const relativePath of [
    "deploy/local/api/entrypoint.sh",
    "deploy/local/worker/entrypoint.sh",
  ]) {
    const entrypoint = read(relativePath);
    assert.match(entrypoint, /^#!\/bin\/sh$/mu);
    assert.match(entrypoint, /\bset -eu\b/u);
    assert.match(entrypoint, /\/run\/secrets\//u);
    assert.match(entrypoint, /\bexec\b/u);
    assert.doesNotMatch(entrypoint, /\bset -x\b/u);
    assert.doesNotMatch(
      entrypoint,
      /\becho\b[^\n]*(?:PASSWORD|SECRET|TOKEN|KEY|credential)/iu,
    );
  }
});

test("Caddy owns one HTTPS origin and exact same-origin upstream routes", () => {
  const caddyfile = read("deploy/local/gateway/Caddyfile");
  assert.match(caddyfile, /https:\/\/localhost:8443/u);
  assert.match(caddyfile, /^\s*skip_install_trust\s*$/mu);
  assert.match(caddyfile, /\btls internal\b/u);
  assert.match(caddyfile, /\bencode zstd gzip\b/u);
  assert.match(caddyfile, /\/api\/\*/u);
  assert.match(caddyfile, /\bstrip_prefix \/api\b/u);
  assert.match(caddyfile, /\/auth\/\*/u);
  assert.match(caddyfile, /\/health\/\*/u);
  assert.match(caddyfile, /\/identity\/\*/u);
  assert.match(caddyfile, /@test_routes path \/__test \/__test\/\*/u);
  assert.match(caddyfile, /handle @test_routes/u);
  assert.match(caddyfile, /\brespond 404\b/u);
  assert.match(caddyfile, /\breverse_proxy api:8080\b/u);
  assert.match(caddyfile, /\bhandle_path \/identity\/\*/u);
  assert.match(caddyfile, /\breverse_proxy preprod-auth:8080\b/u);
  assert.match(caddyfile, /\breverse_proxy web:8080\b/u);
  assert.doesNotMatch(caddyfile, /\brewrite @spa\b/u);

  const headers = read("deploy/local/gateway/security-headers.caddy");
  for (const requiredHeader of [
    "Strict-Transport-Security",
    "Content-Security-Policy",
    "X-Content-Type-Options",
    "Referrer-Policy",
    "Permissions-Policy",
  ]) {
    assert.match(headers, new RegExp(requiredHeader, "u"));
  }
  assert.match(caddyfile, /max-age=31536000, immutable/u);

  const compose = read("deploy/local/compose.yaml");
  assert.match(
    compose,
    /test: \[CMD, \/healthcheck, https:\/\/localhost:8443\/\]/u,
  );
});

test("Compose maps each local runtime service to its reviewed build target", () => {
  const rendered = JSON.parse(
    execFileSync(
      "docker",
      [
        "compose",
        "--file",
        path.join(repositoryRoot, "deploy/local/compose.yaml"),
        "--profile",
        "full",
        "--profile",
        "demo",
        "--profile",
        "test",
        "config",
        "--format",
        "json",
      ],
      { cwd: repositoryRoot, encoding: "utf8" },
    ),
  );
  const targets = {
    gateway: "gateway",
    "web-demo": "demo",
    "web-http": "http",
    api: "api",
    worker: "worker",
    "fixture-init": "migration",
  };
  for (const [serviceName, expectedTarget] of Object.entries(targets)) {
    const service = rendered.services[serviceName];
    assert.ok(service, `${serviceName} must be rendered`);
    assert.equal(service.build?.target, expectedTarget);
    assert.equal(service.read_only, true);
    assert.match(String(service.user), /^(?!0(?::|$)|root(?::|$)).+/u);
  }
});

test("ClamAV uses its reviewed amd64-only image on Apple Silicon hosts", () => {
  const rendered = JSON.parse(
    execFileSync(
      "docker",
      [
        "compose",
        "--file",
        path.join(repositoryRoot, "deploy/local/compose.yaml"),
        "--profile",
        "full",
        "config",
        "--format",
        "json",
      ],
      { cwd: repositoryRoot, encoding: "utf8" },
    ),
  );

  assert.equal(rendered.services.clamav.platform, "linux/amd64");
});

test("every full-profile Go process uses production configuration without test bypasses", () => {
  const rendered = JSON.parse(
    execFileSync(
      "docker",
      [
        "compose",
        "--file",
        path.join(repositoryRoot, "deploy/local/compose.yaml"),
        "--profile",
        "full",
        "config",
        "--format",
        "json",
      ],
      { cwd: repositoryRoot, encoding: "utf8" },
    ),
  );

  for (const serviceName of ["api", "worker"]) {
    const service = rendered.services[serviceName];
    assert.equal(
      service.environment.AVIA_ENVIRONMENT,
      "production",
      `${serviceName} must use production configuration`,
    );
    const serializedEnvironment = JSON.stringify(service.environment);
    assert.doesNotMatch(
      serializedEnvironment,
      /AVIA_(?:ENABLE_CANONICAL|CANONICAL_TEST|TEST_|DEV_SESSION)/u,
      `${serviceName} must not receive test or development bypasses`,
    );
  }
});

test("the HTTP browser config is same-origin under the Caddy API prefix", () => {
  const config = JSON.parse(
    read("apps/web/public/http/http-config.json"),
  );
  assert.equal(config.apiBaseUrl, "/api/");
  assert.doesNotMatch(JSON.stringify(config), /https?:\/\//u);
});
