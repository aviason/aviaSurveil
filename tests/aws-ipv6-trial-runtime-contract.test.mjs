import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const composePath = path.join(repositoryRoot, "deploy/aws-ipv6-trial/compose.yaml");
const caddyPath = path.join(repositoryRoot, "deploy/aws-ipv6-trial/gateway/Caddyfile");
const dockerfilePath = path.join(repositoryRoot, "deploy/aws-ipv6-trial/gateway/Dockerfile");
const buildScriptPath = path.join(repositoryRoot, "scripts/build-aws-ipv6-trial-images.sh");
const runtimeScriptPath = path.join(repositoryRoot, "scripts/test-aws-ipv6-trial-runtime.sh");

function read(filePath) {
  assert.equal(existsSync(filePath), true, `${path.relative(repositoryRoot, filePath)} must exist`);
  return readFileSync(filePath, "utf8");
}

function serviceBlock(compose, name) {
  const start = compose.indexOf(`\n  ${name}:`);
  assert.ok(start >= 0, `${name} service must exist`);
  const tail = compose.slice(start + 1);
  const next = tail.search(/\n  [a-z0-9-]+:/u);
  return next >= 0 ? tail.slice(0, next) : tail;
}

test("milestone-1 Compose contains exactly cloudflared, internal gateway, and web-demo", () => {
  const compose = read(composePath);
  assert.match(compose, /^services:\n/mu);
  for (const service of ["cloudflared", "gateway", "web-demo"]) assert.match(compose, new RegExp(`^  ${service}:`, "mu"));
  assert.doesNotMatch(compose, /^  (?:api|worker|scheduler|keycloak|postgres|clamav|minio|gotenberg|mailpit):/mu);
  assert.match(compose, /trial-internal:\n\s+driver: bridge\n\s+internal: true/u);
  assert.match(compose, /TUNNEL_EDGE_IP_VERSION:\s*["']?6/u);
  assert.match(compose, /--token-file[\s\S]*\/run\/secrets\/tunnel-token/u);
  assert.match(compose, /cloudflared_config/u);
});

test("all services enforce ARM64, immutable image inputs, no host ports, and runtime hardening", () => {
  const compose = read(composePath);
  assert.equal((compose.match(/platform:\s*linux\/arm64/gu) ?? []).length, 3);
  assert.match(compose, /AVIA_TRIAL_CLOUDFLARED_IMAGE:\?[^}]*digest-bound/u);
  assert.match(compose, /AVIA_TRIAL_GATEWAY_IMAGE:\?[^}]*digest-bound/u);
  assert.match(compose, /AVIA_TRIAL_WEB_DEMO_IMAGE:\?[^}]*digest-bound/u);
  assert.doesNotMatch(compose, /platform:\s*linux\/amd64|qemu|rosetta|privileged:\s*true/u);
  assert.doesNotMatch(compose, /^\s+ports:/mu);
  assert.doesNotMatch(compose, /network_mode:\s*host/u);
  assert.equal((compose.match(/read_only:\s*true/gu) ?? []).length, 1);
  assert.equal((compose.match(/no-new-privileges:true/gu) ?? []).length, 1);
  assert.equal((compose.match(/- ALL/gu) ?? []).length, 1);
  assert.equal((compose.match(/healthcheck:/gu) ?? []).length, 3);
  assert.match(serviceBlock(compose, "cloudflared"), /- edge\n\s+- trial-internal/u);
  assert.match(serviceBlock(compose, "gateway"), /- trial-internal/u);
  assert.match(serviceBlock(compose, "web-demo"), /- trial-internal/u);
});

test("gateway is internal HTTP with no origin TLS bypass and bounded forwarded context", () => {
  const caddy = read(caddyPath);
  assert.match(caddy, /:8080\s*\{/u);
  assert.match(caddy, /reverse_proxy web-demo:8080/u);
  assert.match(caddy, /header_up CF-Connecting-IP/u);
  assert.match(caddy, /header_up X-Forwarded-Proto https/u);
  assert.doesNotMatch(caddy, /tls internal|noTLSVerify|https:\/\//iu);
  assert.match(caddy, /path \/health\/live \/health\/ready/u);
  assert.match(caddy, /respond "ok\\n" 200/u);
});

test("trial gateway image is pinned and built from a scratch runtime", () => {
  const dockerfile = read(dockerfilePath);
  assert.match(dockerfile, /^ARG GO_BUILD_IMAGE=.*@sha256:[0-9a-f]{64}$/mu);
  assert.match(dockerfile, /^FROM scratch AS gateway$/mu);
  assert.match(dockerfile, /^USER 1000:1000$/mu);
  assert.match(dockerfile, /^EXPOSE 8080$/mu);
  assert.doesNotMatch(dockerfile, /linux\/amd64|qemu|rosetta/u);
});

test("build and runtime scripts fail closed on non-ARM64, mutable images, missing scans, or broad cleanup", () => {
  const build = read(buildScriptPath);
  const runtime = read(runtimeScriptPath);
  assert.ok(build.includes('--platform "$platform"') || build.includes("--platform linux/arm64"));
  assert.match(build, /--provenance=mode=max/u);
  assert.match(build, /--sbom=/u);
  assert.match(build, /trivy image[\s\S]*HIGH,CRITICAL/u);
  assert.match(build, /@sha256:\[0-9a-f\]\{64\}/u);
  assert.match(build, /containerimage\.digest/u);
  assert.match(build, /architecture.*arm64|arm64.*architecture/u);
  assert.doesNotMatch(build, /docker\s+pull/u);
  assert.match(build, /DOCKER_DEFAULT_PLATFORM[^\n]*linux\/amd64/u);
  assert.doesNotMatch(build, /qemu|rosetta/u);
  assert.match(runtime, /--run/u);
  assert.match(runtime, /0600|0400/u);
  assert.match(runtime, /docker compose[\s\S]*down --volumes --remove-orphans/u);
  assert.match(runtime, /label=com\.docker\.compose\.project=/u);
  assert.doesNotMatch(runtime, /docker\s+(?:system|volume|network)\s+prune/u);
  assert.match(runtime, /thirtyMinuteBrowserLoop.*not run/u);
});
