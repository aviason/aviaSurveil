import assert from "node:assert/strict";
import { spawn, spawnSync } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { pathToFileURL } from "node:url";
import { test } from "node:test";
import { validateCloudflareTunnelToken } from "../scripts/validate-cloudflare-tunnel-token.mjs";

const read = (file) => readFileSync(file, "utf8");

test("named Tunnel token storage bypasses the 128-byte Keychain CLI prompt without exposing the secret", () => {
  const store = read("scripts/store-canonical-preprod-cloudflare-token.sh");
  const writer = read("scripts/store-cloudflare-tunnel-token-keychain.swift");

  assert.doesNotMatch(store, /security add-generic-password/u);
  assert.match(store, /read -r -s token <\/dev\/tty/u);
  assert.match(store, /read -r -s confirmation <\/dev\/tty/u);
  assert.match(store, /swiftc_binary/u);
  assert.match(store, /mktemp -d \/tmp\/aviasurveil360-cloudflare-keychain\.XXXXXX/u);
  assert.match(store, /printf '%s' "\$token" \| "\$writer_binary"/u);
  assert.match(store, /store-cloudflare-tunnel-token-keychain\.swift/u);
  assert.match(store, /accepts no arguments/u);
  assert.match(store, /security find-generic-password/u);
  assert.match(store, /security delete-generic-password/u);
  assert.match(store, /validate-cloudflare-tunnel-token\.mjs/u);
  assert.doesNotMatch(store, /TUNNEL_TOKEN|set\s+-x/u);
  assert.doesNotMatch(store, /(?:-w|-X)\s+"?\$token/u);

  assert.match(writer, /FileHandle\.standardInput\.readDataToEndOfFile/u);
  assert.match(writer, /SecItemAdd/u);
  assert.match(writer, /kSecValueData/u);
  assert.match(writer, /SecAccessCreate/u);
  assert.match(writer, /SecTrustedApplicationCreateFromPath\(\s*"\/usr\/bin\/security"/u);
  assert.doesNotMatch(writer, /Process\(|TUNNEL_TOKEN/u);
  assert.ok(
    store.indexOf('node "$token_validator"') < store.indexOf("security delete-generic-password"),
    "the complete connector payload must validate before replacing the exact Keychain item",
  );

  const argumentAttempt = spawnSync(
    "bash",
    ["scripts/store-canonical-preprod-cloudflare-token.sh", "must-not-be-accepted"],
    { cwd: process.cwd(), encoding: "utf8" },
  );
  assert.notEqual(argumentAttempt.status, 0);
  assert.match(argumentAttempt.stderr, /accepts no arguments/u);
});

test("connector-token validator accepts the encoded connector payload and rejects partial or copied input", () => {
  const validToken = Buffer.from(
    JSON.stringify({ a: "account-id", t: "tunnel-id", s: "connector-secret" }),
    "utf8",
  ).toString("base64");

  assert.deepEqual(validateCloudflareTunnelToken(Buffer.from(validToken)), {
    byteLength: validToken.length,
  });
  assert.deepEqual(validateCloudflareTunnelToken(Buffer.from(`${validToken}\n`)), {
    byteLength: validToken.length,
  });

  const invalidTokens = [
    validToken.slice(0, -8),
    `cloudflared service install ${validToken}`,
    Buffer.from("not-json", "utf8").toString("base64"),
    Buffer.from(JSON.stringify({ a: "account-id", t: "tunnel-id" }), "utf8").toString("base64"),
  ];
  for (const token of invalidTokens) {
    assert.throws(() => validateCloudflareTunnelToken(Buffer.from(token)), /malformed or truncated/u);
  }

  const cliFailure = spawnSync(
    process.execPath,
    ["scripts/validate-cloudflare-tunnel-token.mjs"],
    { cwd: process.cwd(), encoding: "utf8", input: `cloudflared service install ${validToken}` },
  );
  assert.notEqual(cliFailure.status, 0);
  assert.match(cliFailure.stderr, /malformed or truncated/u);
  assert.equal(cliFailure.stderr.includes(validToken), false);
});

test("named Tunnel launcher keeps the connector token out of argv, env, logs, and disk", () => {
  const launcher = read("scripts/canonical-preprod-cloudflare-named-launcher.mjs");

  assert.match(launcher, /execFileSync\(\s*"\/usr\/bin\/security"/u);
  assert.match(launcher, /find-generic-password/u);
  assert.match(launcher, /validateCloudflareTunnelToken/u);
  assert.match(
    launcher,
    /\["tunnel", "--no-autoupdate", "--protocol", "http2", "run", "--token-file", "\/dev\/fd\/3"\]/u,
  );
  assert.match(launcher, /stdio:\s*\["ignore", logFd, logFd, "pipe"\]/u);
  assert.match(launcher, /tokenPipe\.end\(tokenPayload/u);
  assert.match(launcher, /tokenPipe\.destroy\(\)/u);
  assert.match(launcher, /tokenPipe\.unref\?\.\(\)/u);
  assert.match(launcher, /detached:\s*true/u);
  assert.match(launcher, /child\.unref\(\)/u);
  assert.match(launcher, /keychainValue\.fill\(0\)/u);
  assert.match(launcher, /tokenPayload\?\.fill\(0\)/u);
  assert.match(launcher, /cloudflared exited while validating/u);
  assert.ok(
    launcher.indexOf("recordOwnedChild(pidPath, child.pid)") <
      launcher.indexOf("tokenPipe.end(tokenPayload"),
    "the exact connector PID must be durable before the token can create public exposure",
  );
  assert.match(launcher, /installOwnedChildSignalCleanup\(child, pidPath\)/u);
  assert.match(launcher, /await terminateOwnedChild\(child, pidPath\)/u);
  assert.doesNotMatch(launcher, /TUNNEL_TOKEN|process\.env|--token["']/u);
  assert.doesNotMatch(launcher, /writeFileSync\([^\n]*(?:token|keychain)/iu);
});

test("named Tunnel ownership guard removes a detached connector when its launcher is interrupted", async () => {
  const temporaryRoot = mkdtempSync(join(tmpdir(), "avia-named-launcher-signal-"));
  const fixturePath = join(temporaryRoot, "launcher-fixture.mjs");
  const pidPath = join(temporaryRoot, "cloudflared.pid");
  const childPidPath = join(temporaryRoot, "child.pid");
  const guardModule = pathToFileURL(
    resolve("scripts/lib/canonical-preprod-owned-child.mjs"),
  ).href;
  writeFileSync(
    fixturePath,
    `import { spawn } from "node:child_process";
import { writeFileSync } from "node:fs";
import { installOwnedChildSignalCleanup, recordOwnedChild } from ${JSON.stringify(guardModule)};
const [pidPath, childPidPath] = process.argv.slice(2);
const child = spawn(process.execPath, ["-e", "setInterval(() => {}, 1000)"], {
  detached: true,
  stdio: "ignore",
});
await new Promise((resolve, reject) => {
  child.once("spawn", resolve);
  child.once("error", reject);
});
recordOwnedChild(pidPath, child.pid);
installOwnedChildSignalCleanup(child, pidPath);
writeFileSync(childPidPath, String(child.pid));
process.stdout.write("ready\\n");
setInterval(() => {}, 1000);
`,
    { mode: 0o600 },
  );

  let launcher;
  let detachedChildPid;
  try {
    launcher = spawn(process.execPath, [fixturePath, pidPath, childPidPath], {
      stdio: ["ignore", "pipe", "pipe"],
    });
    await new Promise((resolveReady, rejectReady) => {
      const timer = setTimeout(() => rejectReady(new Error("fixture did not become ready")), 5_000);
      launcher.once("error", rejectReady);
      launcher.stdout.once("data", (chunk) => {
        if (chunk.toString("utf8").includes("ready")) {
          clearTimeout(timer);
          resolveReady();
        }
      });
    });
    detachedChildPid = Number(readFileSync(childPidPath, "utf8"));
    assert.equal(readFileSync(pidPath, "utf8").trim(), String(detachedChildPid));
    launcher.kill("SIGTERM");
    const exit = await new Promise((resolveExit) => {
      launcher.once("exit", (code, signal) => resolveExit({ code, signal }));
    });
    assert.deepEqual(exit, { code: 143, signal: null });
    assert.equal(existsSync(pidPath), false, "signal cleanup must remove the exact PID record");
    assert.throws(() => process.kill(detachedChildPid, 0), { code: "ESRCH" });
  } finally {
    if (launcher?.exitCode === null && launcher?.signalCode === null) {
      launcher.kill("SIGKILL");
    }
    if (Number.isSafeInteger(detachedChildPid)) {
      try {
        process.kill(detachedChildPid, "SIGKILL");
      } catch {
        // The expected path has already removed the detached fixture child.
      }
    }
    rmSync(temporaryRoot, { recursive: true, force: true });
  }
});

test("named demo Make targets use a separate exact local profile", () => {
  const makefile = read("Makefile");

  assert.match(makefile, /CLOUDFLARE_DEMO_HOSTNAME \?= demo\.aviasurveil\.com/u);
  assert.match(makefile, /CANONICAL_PREPROD_CLOUDFLARE_DEMO_HTTP_PORT \?= 8086/u);
  assert.match(makefile, /aviasurveil360-local-preprod-cloudflare-demo/u);
  for (const target of ["token", "up", "status", "users", "down"]) {
    assert.match(makefile, new RegExp("^preprod-cloudflare-demo-" + target + ":$", "mu"));
  }
  assert.match(makefile, /AVIA_PREPROD_CLOUDFLARE_MODE=named/u);
  assert.match(makefile, /scripts\/store-canonical-preprod-cloudflare-token\.sh/u);
  assert.match(makefile, /scripts\/start-canonical-preprod-cloudflare\.sh/u);
  assert.match(makefile, /scripts\/status-canonical-preprod-cloudflare\.sh/u);
  assert.match(makefile, /scripts\/stop-canonical-preprod-cloudflare\.sh/u);

  const dryRun = spawnSync("make", ["--dry-run", "preprod-cloudflare-demo-up"], {
    cwd: process.cwd(),
    encoding: "utf8",
  });
  assert.equal(dryRun.status, 0, dryRun.stderr);
  assert.match(dryRun.stdout, /AVIA_PREPROD_PUBLIC_HOSTNAME="demo\.aviasurveil\.com"/u);
  assert.match(dryRun.stdout, /AVIA_PREPROD_HTTP_PORT="8086"/u);
  assert.doesNotMatch(dryRun.stdout, /eyJ|TUNNEL_TOKEN/u);
});

test("named Tunnel lifecycle binds exact hostname, Keychain reference, process identity, and cleanup order", () => {
  const start = read("scripts/start-canonical-preprod-cloudflare.sh");
  const status = read("scripts/status-canonical-preprod-cloudflare.sh");
  const stop = read("scripts/stop-canonical-preprod-cloudflare.sh");
  const users = read("scripts/show-canonical-preprod-cloudflare-users.sh");

  assert.match(start, /AVIA_PREPROD_CLOUDFLARE_MODE/u);
  assert.match(start, /public_origin="https:\/\/\$public_hostname"/u);
  assert.match(start, /canonical-preprod-cloudflare-named-launcher\.mjs/u);
  assert.match(start, /validate-cloudflare-tunnel-token\.mjs/u);
  assert.match(start, /wait_for_public_dns_publication/u);
  assert.match(start, /verify the dashboard route targets \$local_origin/u);
  assert.match(start, /credentialReference/u);
  assert.match(start, /kind: "macos-keychain"/u);
  assert.match(start, /externalPreprod: "not run"/u);
  assert.match(start, /sanitized cloudflared log tail/u);
  assert.match(start, /REDACTED_CONNECTOR_TOKEN/u);
  assert.match(start, /local images are ready; starting the \$tunnel_label connector/u);
  assert.match(start, /connector is running; checking DNS and the public route/u);
  assert.match(start, /public route is ready; replacing the startup placeholder/u);
  assert.ok(
    start.indexOf('tunnel_command="$cloudflared_binary tunnel --no-autoupdate') <
      start.indexOf('node "$named_tunnel_launcher"'),
    "cleanup must know the exact named cloudflared identity before the launcher can be interrupted",
  );
  assert.ok(
    start.indexOf('node "$tunnel_token_validator"') < start.indexOf("prebuild_images\n"),
    "named Tunnel token validation must run before the Docker image build",
  );

  for (const script of [status, stop]) {
    assert.match(script, /runtimeMode = metadata\.tunnel\?\.mode \?\? "quick"/u);
    assert.match(script, /publicOrigin\.hostname === expectedHostname/u);
    assert.match(script, /credentialReference\?\.kind === "macos-keychain"/u);
    assert.match(script, /--token-file \/dev\/fd\/3/u);
    assert.match(script, /processCommand/u);
  }
  assert.ok(
    stop.lastIndexOf("stop_verified_tunnel") < stop.lastIndexOf("compose_down"),
    "the named public exposure must be removed before disposable services",
  );
  assert.ok(
    stop.lastIndexOf("stop_verified_tunnel") < stop.lastIndexOf("docker info"),
    "the named public exposure must be removable even when Docker is unavailable",
  );
  assert.match(stop, /public exposure stopped, but the Docker daemon is unavailable/u);
  assert.match(stop, /dashboard tunnel\/DNS configuration were retained/u);
  assert.match(users, /runtimeMode !== expectedMode/u);
  assert.match(users, /origin\.hostname === expectedHostname/u);
});

test("named mode rejects malformed hostnames before Keychain or Docker work", () => {
  const result = spawnSync("bash", ["scripts/start-canonical-preprod-cloudflare.sh"], {
    cwd: process.cwd(),
    encoding: "utf8",
    env: {
      ...process.env,
      AVIA_PREPROD_CLOUDFLARE_MODE: "named",
      AVIA_PREPROD_PUBLIC_HOSTNAME: "https://demo.aviasurveil.com/path",
      AVIA_CANONICAL_PREPROD_STATE_DIR: process.cwd() + "/.local/named-contract-state",
      AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR:
        process.cwd() + "/.local/named-contract-runtime",
    },
  });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /must be a lowercase DNS hostname/u);
  assert.doesNotMatch(result.stderr, /docker daemon|Compose residue/u);
});
