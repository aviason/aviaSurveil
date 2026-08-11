import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { test } from "node:test";

const read = (file) => readFileSync(file, "utf8");

test("canonical HTTPS remains the default and HTTP is an explicit isolated override", () => {
  const canonical = read("scripts/start-canonical-preprod.sh");
  const http = read("scripts/start-canonical-preprod-http.sh");

  assert.match(canonical, /transport="\$\{AVIA_PREPROD_TRANSPORT:-https\}"/u);
  assert.match(canonical, /https_port="\$\{AVIA_PREPROD_HTTPS_PORT:-8445\}"/u);
  assert.match(canonical, /AVIA_PREPROD_HTTPS_PORT must be a user-space TCP port/u);
  assert.match(canonical, /canonical HTTPS transport does not accept a Compose override/u);
  assert.match(canonical, /HTTP transport requires the task-owned deploy\/local\/compose\.local-http\.yaml override/u);
  assert.match(canonical, /AVIA_PREPROD_WEB_ORIGIN must be an absolute HTTP\(S\) origin without a path/u);
  assert.match(canonical, /AVIA_PREPROD_COOKIE_SECURE/u);
  assert.match(canonical, /origin_scheme="\$\{origin_scheme%:\}"/u);
  assert.match(canonical, /AVIA_PREPROD_ORIGIN_SCHEME="\$origin_scheme"/u);
  assert.match(
    canonical,
    /compose up --detach --wait --wait-timeout 600 preprod-keycloak/u,
    "Keycloak must keep its real readiness gate while allowing a cold optimized start",
  );
  assert.doesNotMatch(canonical, /cloudflared|trycloudflare/u);

  assert.match(http, /export AVIA_PREPROD_TRANSPORT=http/u);
  assert.match(http, /aviasurveil360-local-preprod-http/u);
  assert.match(http, /compose\.local-http\.yaml/u);
  assert.doesNotMatch(http, /trycloudflare|cloudflared/u);
});

test("transport public-origin and cookie mode validation fail closed before Docker", () => {
  const canonicalOverride = spawnSync("bash", ["scripts/start-canonical-preprod.sh"], {
    cwd: process.cwd(),
    encoding: "utf8",
    env: {
      ...process.env,
      AVIA_PREPROD_TRANSPORT: "https",
      AVIA_PREPROD_HTTPS_PORT: "8445",
      AVIA_PREPROD_WEB_ORIGIN: "https://localhost:8445",
      AVIA_PREPROD_COMPOSE_OVERRIDE: `${process.cwd()}/deploy/local/compose.local-http.yaml`,
    },
  });
  assert.notEqual(canonicalOverride.status, 0);
  assert.match(
    canonicalOverride.stderr,
    /canonical HTTPS transport does not accept a Compose override/u,
  );

  const traversal = spawnSync("bash", ["scripts/start-canonical-preprod-cloudflare.sh"], {
    cwd: process.cwd(),
    encoding: "utf8",
    env: {
      ...process.env,
      AVIA_CANONICAL_PREPROD_STATE_DIR: `${process.cwd()}/.local/../not-task-owned`,
    },
  });
  assert.notEqual(traversal.status, 0);
  assert.match(
    traversal.stderr,
    /AVIA_CANONICAL_PREPROD_STATE_DIR must be a canonical path without \. or \.\. components/u,
  );

  const common = {
    ...process.env,
    AVIA_PREPROD_TRANSPORT: "http",
    AVIA_PREPROD_COMPOSE_OVERRIDE: `${process.cwd()}/deploy/local/compose.local-http.yaml`,
  };
  const invalidOrigin = spawnSync("bash", ["scripts/start-canonical-preprod.sh"], {
    cwd: process.cwd(),
    encoding: "utf8",
    env: {
      ...common,
      AVIA_PREPROD_WEB_ORIGIN: "https://invalid.trycloudflare.com/path",
    },
  });
  assert.notEqual(invalidOrigin.status, 0);
  assert.match(
    invalidOrigin.stderr,
    /AVIA_PREPROD_WEB_ORIGIN must be an absolute HTTP\(S\) origin without a path/u,
  );

  const invalidCookie = spawnSync("bash", ["scripts/start-canonical-preprod.sh"], {
    cwd: process.cwd(),
    encoding: "utf8",
    env: {
      ...common,
      AVIA_PREPROD_WEB_ORIGIN: "https://valid.trycloudflare.com",
      AVIA_PREPROD_COOKIE_SECURE: "false",
    },
  });
  assert.notEqual(invalidCookie.status, 0);
  assert.match(
    invalidCookie.stderr,
    /AVIA_PREPROD_COOKIE_SECURE must match the public origin TLS mode/u,
  );
});

test("Quick Tunnel is anonymous, detached from developer Cloudflare state, and parses one strict HTTPS origin", async () => {
  const start = read("scripts/start-canonical-preprod-cloudflare.sh");
  const canonical = read("scripts/start-canonical-preprod.sh");
  const placeholder = read("scripts/canonical-preprod-tunnel-placeholder.mjs");
  const launcher = read("scripts/canonical-preprod-cloudflare-launcher.mjs");
  const parserSource = read("scripts/canonical-preprod-quick-tunnel-url.mjs");
  const { extractQuickTunnelOrigin } = await import(
    "../scripts/canonical-preprod-quick-tunnel-url.mjs"
  );

  assert.match(start, /canonical-preprod-quick-tunnel-url\.mjs/u);
  assert.match(start, /canonical-preprod-cloudflare-launcher\.mjs/u);
  assert.match(
    start,
    /node "\$quick_tunnel_launcher" "\$cloudflared_binary" "\$http_port" "\$runtime_root"/u,
  );
  assert.match(start, /prebuild_images/u);
  assert.ok(
    start.indexOf("\nprebuild_images\n") >= 0 &&
      start.indexOf("\nprebuild_images\n") < start.indexOf('node "$quick_tunnel_launcher"'),
    "all local images must be built before the short-lived anonymous tunnel is launched",
  );
  assert.match(start, /AVIA_PREPROD_SKIP_BUILD=true/u);
  assert.match(canonical, /skip_build="\$\{AVIA_PREPROD_SKIP_BUILD:-false\}"/u);
  assert.doesNotMatch(start, /\bnohup\b/u);
  assert.match(
    launcher,
    /spawn\(\s*cloudflared,\s*\["tunnel", "--url", localOrigin, "--protocol", "http2"\]/u,
    "Quick Tunnel must use TCP HTTP/2 because UDP/QUIC is not stable on the qualification network",
  );
  assert.match(launcher, /detached:\s*true/u);
  assert.match(launcher, /stdio:\s*\["ignore", logFd, logFd\]/u);
  assert.match(launcher, /child\.unref\(\)/u);
  assert.match(launcher, /HOME:\s*join\(runtimeRoot, "cloudflared-home"\)/u);
  assert.match(launcher, /XDG_CONFIG_HOME:\s*join\(runtimeRoot, "xdg-config"\)/u);
  assert.doesNotMatch(
    launcher,
    /process\.env|--token|\b(?:login|create|route|dns|access)\b|TUNNEL_TOKEN|api\.cloudflare|aws|AWS/iu,
  );
  assert.match(parserSource, /trycloudflare\.com/u);
  assert.match(placeholder, /127\.0\.0\.1/u);
  assert.match(placeholder, /"cache-control": "no-store"/u);
  assert.match(placeholder, /http-equiv="refresh" content="3"/u);
  assert.match(
    start,
    /cloudflare-dns\.com\/dns-query/u,
    "the wrapper must wait for authoritative Quick Tunnel DNS before touching the system resolver",
  );
  assert.match(start, /\[1, 28\]\.includes\(answer\.type\)/u);
  assert.doesNotMatch(start, /\[1, 5, 28\]\.includes\(answer\.type\)/u);
  assert.ok(
    start.indexOf("wait_for_public_dns_publication") <
      start.indexOf('wait_for_http "$public_origin/__canonical_preprod_tunnel_placeholder_ready"'),
    "authoritative DNS publication must precede the first public placeholder probe",
  );
  const quickLaunchBranch = start.match(
    /if \[\[ "\$tunnel_mode" == quick \]\]; then\n([\s\S]*?)\nelse\n/u,
  )?.[1];
  assert.ok(quickLaunchBranch, "the explicit Quick Tunnel launch branch must remain present");
  assert.doesNotMatch(
    quickLaunchBranch,
    /cloudflared[^\n]*(?:login|create|route|dns|access|token)|--token|TUNNEL_TOKEN/iu,
  );
  assert.doesNotMatch(start, /TUNNEL_TOKEN|api\.cloudflare|aws|AWS/u);

  assert.equal(
    extractQuickTunnelOrigin("INF Your quick Tunnel has been created! Visit it at https://bird-12.trycloudflare.com"),
    "https://bird-12.trycloudflare.com",
  );
  assert.throws(
    () => extractQuickTunnelOrigin("https://one.trycloudflare.com https://two.trycloudflare.com"),
    /exactly one/u,
  );
  assert.throws(
    () => extractQuickTunnelOrigin("http://one.trycloudflare.com"),
    /HTTPS/u,
  );
  assert.throws(
    () => extractQuickTunnelOrigin("https://one.trycloudflare.com/path"),
    /origin/u,
  );
  assert.throws(
    () => extractQuickTunnelOrigin("https://one.example.com"),
    /trycloudflare/u,
  );
});

test("HTTP override accepts a random Quick Tunnel host and wires the public HTTPS origin end to end", () => {
  const override = read("deploy/local/compose.local-http.yaml");
  const caddy = read("deploy/local/gateway/Caddyfile.preprod.http");
  const canonicalCaddy = read("deploy/local/gateway/Caddyfile.preprod");

  assert.match(override, /host_ip:\s*127\.0\.0\.1/u);
  assert.match(override, /ports:\s*!override/u);
  assert.match(
    override,
    /preprod-keycloak:[\s\S]*?ports:\s*!override\s*\[\]/u,
    "the Quick Tunnel profile must not inherit Keycloak's direct host port",
  );
  assert.doesNotMatch(override, /published:\s*["']?58082/u);
  assert.match(override, /--proxy-headers=xforwarded/u);
  assert.match(override, /AVIA_COOKIE_SECURE/u);
  assert.match(override, /AVIA_OIDC_ISSUER_URL/u);
  assert.match(override, /AVIA_OIDC_REDIRECT_URL/u);
  assert.match(override, /AVIA_OBJECT_STORE_PUBLIC_ENDPOINT/u);
  assert.match(override, /AVIA_OBJECT_STORE_PUBLIC_TLS/u);
  assert.match(override, /AVIA_OBJECT_STORE_CORS_ORIGINS/u);
  assert.match(
    override,
    /AVIA_PREPROD_ORIGIN_SCHEME:\s*"\$\{AVIA_PREPROD_ORIGIN_SCHEME:-http\}"/u,
  );
  assert.match(override, /\$\{AVIA_PREPROD_WEB_ORIGIN/u);
  assert.match(override, /\$\{AVIA_PREPROD_PUBLIC_HOST/u);
  assert.match(caddy, /^:8085 \{/mu, "the HTTP gateway must accept the random Tunnel Host");
  assert.match(caddy, /import security_headers_http/u);
  assert.equal(
    (caddy.match(/header_up X-Forwarded-Proto \{\$AVIA_PREPROD_ORIGIN_SCHEME\}/gu) ?? []).length,
    5,
    "API, auth, health, identity, and object paths must receive the configured public scheme",
  );
  assert.doesNotMatch(caddy, /https:\/\//u);
  assert.doesNotMatch(canonicalCaddy, /trycloudflare|:8085/u);
});

test("rendered HTTP Quick Tunnel config never publishes Keycloak directly", (t) => {
  const rendered = spawnSync(
    "docker",
    [
      "compose",
      "--project-name",
      "aviasurveil360-local-preprod-cloudflare-contract",
      "--file",
      "deploy/local/compose.yaml",
      "--file",
      "deploy/local/compose.local-http.yaml",
      "--profile",
      "local-preprod-loader",
      "config",
      "--format",
      "json",
    ],
    {
      cwd: process.cwd(),
      encoding: "utf8",
      env: {
        ...process.env,
        AVIA_PREPROD_STATE_DIR: "/private/tmp/aviasurveil360-quick-tunnel-contract",
        AVIA_PREPROD_PROFILE: "aga-preprod@1.0.0",
        AVIA_PREPROD_PROFILE_QUALIFICATION: "true",
        AVIA_PREPROD_IDENTITY_NAMESPACE: "canonical-aga-preprod-exercise-v1",
        AVIA_PREPROD_TRANSPORT: "http",
        AVIA_PREPROD_HTTP_PORT: "18085",
        AVIA_PREPROD_WEB_ORIGIN: "https://fixture-quick-tunnel.trycloudflare.com",
        AVIA_PREPROD_KEYCLOAK_PUBLIC_ORIGIN: "https://fixture-quick-tunnel.trycloudflare.com",
        AVIA_PREPROD_PUBLIC_HOST: "fixture-quick-tunnel.trycloudflare.com",
        AVIA_PREPROD_ORIGIN_SCHEME: "https",
        AVIA_PREPROD_PUBLIC_TLS: "true",
        AVIA_PREPROD_COOKIE_SECURE: "true",
      },
    },
  );
  if (rendered.error?.code === "ENOENT") {
    t.skip("docker compose is unavailable for the rendered-config contract");
    return;
  }
  assert.equal(rendered.status, 0, rendered.stderr);
  const config = JSON.parse(rendered.stdout);
  assert.deepEqual(config.services["preprod-keycloak"].ports ?? [], []);
  assert.equal(
    config.services["preprod-gateway"].ports?.[0]?.host_ip,
    "127.0.0.1",
  );
  assert.equal(config.services["preprod-gateway"].ports?.[0]?.published, "18085");
  assert.equal(config.services["preprod-gateway"].environment?.AVIA_PREPROD_ORIGIN_SCHEME, "https");
});

test("Quick Tunnel lifecycle validates ownership and removes every task-owned resource", () => {
  const start = read("scripts/start-canonical-preprod-cloudflare.sh");
  const stop = read("scripts/stop-canonical-preprod-cloudflare.sh");
  const status = read("scripts/status-canonical-preprod-cloudflare.sh");

  assert.match(start, /! -L "\$state_root"/u);
  assert.match(start, /! -L "\$runtime_root"/u);
  assert.match(start, /state directory already exists/u);
  assert.match(start, /runtime directory already exists/u);
  assert.match(start, /health\/ready/u);
  assert.match(start, /openid-configuration/u);
  assert.match(start, /issuer/u);
  assert.match(start, /chmod 0600 "\$runtime_file"/u);

  assert.match(stop, /canonical-preprod-cloudflare-runtime\/v1/u);
  assert.match(stop, /localOrigin/u);
  assert.match(stop, /publicOrigin/u);
  assert.match(stop, /kill -TERM/u);
  assert.match(stop, /kill -KILL/u);
  assert.match(stop, /--volumes --remove-orphans/u);
  assert.match(stop, /com\.docker\.compose\.project/u);
  assert.match(stop, /rm -rf -- "\$state_root"/u);
  assert.match(stop, /rm -rf -- "\$runtime_root"/u);
  assert.doesNotMatch(stop, /retained at/u);
  assert.doesNotMatch(stop, /killall|pkill/u);
  assert.match(status, /canonical-preprod-cloudflare-runtime\/v1/u);
  assert.match(status, /verify_public_discovery "\$public_origin"/u);
  for (const script of [start, stop, status, read("scripts/link-canonical-preprod-cloudflare.sh")]) {
    assert.match(script, /must be a canonical path without \. or \.\. components/u);
    assert.match(script, /must not traverse a symlink/u);
  }
});

test("Make link helper reuses only a healthy exact Quick Tunnel runtime and prints one public URL", () => {
  const makefile = read("Makefile");
  const link = read("scripts/link-canonical-preprod-cloudflare.sh");

  assert.match(makefile, /^preprod-cloudflare-link:$/mu);
  assert.match(makefile, /preprod-cloudflare-link Print or start the disposable Quick Tunnel URL/u);
  assert.match(makefile, /scripts\/link-canonical-preprod-cloudflare\.sh/u);
  assert.match(link, /status-canonical-preprod-cloudflare\.sh/u);
  assert.match(link, /start-canonical-preprod-cloudflare\.sh/u);
  assert.match(link, /runtime metadata is missing/u);
  assert.match(link, /refusing to reuse partial or stale disposable state/u);
  assert.match(link, /canonical-preprod-cloudflare-runtime\/v1/u);
  assert.match(link, /process\.stdout\.write\(`\$\{metadata\.publicOrigin\}\\n`\)/u);
  assert.doesNotMatch(link, /\b(?:kill|pkill|killall)\b/u);
  assert.doesNotMatch(link, /cloudflared\s+(?:login|create|route|dns|access)|TUNNEL_TOKEN|api\.cloudflare|aws|AWS/iu);
});

test("canonical Quick Tunnel provisions the exact privacy-safe multi-role login matrix", () => {
  const fixture = JSON.parse(
    read("deploy/local/fixtures/canonical-preprod-demo-identities.json"),
  );
  const identityLoader = read(
    "apps/api/cmd/preprod-canonical-demo-identity-loader/main.go",
  );
  const compose = read("deploy/local/compose.yaml");
  const canonicalStart = read("scripts/start-canonical-preprod.sh");
  const status = read("scripts/status-canonical-preprod-cloudflare.sh");

  assert.equal(fixture.schemaVersion, "canonical-preprod-demo-identities/v1");
  assert.equal(fixture.users.length, 9);
  assert.deepEqual(
    fixture.users.map((user) => user.role).sort(),
    [
      "admin",
      "auditee",
      "auditee",
      "executiveDirector",
      "finance",
      "gm",
      "inspector",
      "leadInspector",
      "manager",
    ],
  );
  assert.equal(new Set(fixture.users.map((user) => user.scenarioId)).size, 9);
  assert.equal(new Set(fixture.users.map((user) => user.membershipId)).size, 9);
  assert.equal(new Set(fixture.users.map((user) => user.email)).size, 9);
  for (const user of fixture.users) {
    assert.match(user.scenarioId, /^synthetic-[a-z0-9-]+$/u);
    assert.match(user.membershipId, /^CANONICAL-DEMO-MEMBERSHIP-[A-Z0-9-]+$/u);
    assert.match(user.email, /^[a-z0-9-]+@synthetic\.invalid$/u);
    assert.ok(["CAA", "ORG-FLY-NAMIBIA"].includes(user.organizationId));
    assert.doesNotMatch(JSON.stringify(user), /password|secret|token/iu);
  }

  assert.match(identityLoader, /EnsureProviderAccount/u);
  assert.match(identityLoader, /QualifyExistingProviderAccounts/u);
  assert.match(identityLoader, /user_lifecycle_requests/u);
  assert.match(identityLoader, /desired_membership_versions/u);
  assert.match(identityLoader, /desired_membership_sync/u);
  assert.match(identityLoader, /caa_department_memberships/u);
  assert.match(compose, /preprod-canonical-demo-identity-loader:/u);
  assert.match(compose, /preprod_canonical_demo_oidc_qualification_password/u);
  assert.match(canonicalStart, /preprod-canonical-demo-identity-loader/u);
  assert.match(canonicalStart, /demo identity seed count mismatch/u);
  assert.match(status, /demo identity count mismatch/u);
});

test("Quick Tunnel exposes a repeatable browser qualification command for every demo role", () => {
  const makefile = read("Makefile");
  const runner = read("scripts/test-canonical-preprod-cloudflare-panels.sh");
  const users = read("scripts/show-canonical-preprod-cloudflare-users.sh");
  const spec = read("apps/web/tests/e2e/canonical-quick-tunnel-panels.spec.ts");
  const playwright = read("apps/web/playwright.config.ts");

  assert.match(makefile, /^preprod-cloudflare-test-panels:$/mu);
  assert.match(makefile, /^preprod-cloudflare-users:$/mu);
  assert.match(makefile, /scripts\/test-canonical-preprod-cloudflare-panels\.sh/u);
  assert.match(runner, /status-canonical-preprod-cloudflare\.sh/u);
  assert.match(runner, /preprod_canonical_demo_oidc_qualification_password/u);
  assert.match(runner, /AVIA_E2E_PROFILE=canonical-quick-tunnel/u);
  assert.match(runner, /canonical-quick-tunnel-panels/u);
  assert.match(users, /canonical-preprod-demo-identities\.json/u);
  assert.match(users, /Password \(all demo users\)/u);
  assert.match(users, /status-canonical-preprod-cloudflare\.sh/u);
  assert.match(playwright, /canonical-quick-tunnel/u);
  assert.match(playwright, /serviceWorkers:\s*["']allow["']/u);
  assert.doesNotMatch(playwright, /name:\s*["']canonical-quick-tunnel["'][\s\S]{0,500}serviceWorkers:\s*["']block["']/u);
  assert.match(spec, /navigator\.serviceWorker\.getRegistration/u);
  assert.match(spec, /state: "activated"/u);
  assert.doesNotMatch(spec, /await navigator\.serviceWorker\.ready;/u);
  assert.match(spec, /navigator\.serviceWorker\.controller/u);
  assert.match(spec, /__Host-avia_session/u);
  assert.match(spec, /__Host-avia_csrf/u);
  assert.match(spec, /secure/u);
  assert.match(spec, /httpOnly/u);
  assert.match(spec, /Find → Compare → Decide/u);
  assert.match(spec, /New Inspection/u);
  for (const role of [
    "admin",
    "auditee",
    "executiveDirector",
    "finance",
    "gm",
    "inspector",
    "leadInspector",
    "manager",
  ]) {
    assert.match(spec, new RegExp(`role: “${role}”|role: "${role}"`, "u"));
  }
  assert.doesNotMatch(runner, /cat .*(?:password|secret)|set -x/u);
});
