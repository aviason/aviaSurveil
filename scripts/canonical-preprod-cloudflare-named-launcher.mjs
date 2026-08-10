#!/usr/bin/env node
import { execFileSync, spawn } from "node:child_process";
import {
  chmodSync,
  closeSync,
  existsSync,
  lstatSync,
  openSync,
  readdirSync,
  statSync,
} from "node:fs";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { validateCloudflareTunnelToken } from "./validate-cloudflare-tunnel-token.mjs";
import {
  installOwnedChildSignalCleanup,
  recordOwnedChild,
  terminateOwnedChild,
} from "./lib/canonical-preprod-owned-child.mjs";

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const repositoryRoot = resolve(scriptDirectory, "..");
const localRoot = resolve(repositoryRoot, ".local");
const [cloudflared, runtimeRootInput, keychainService, keychainAccount] = process.argv.slice(2);

if (process.argv.length !== 6) {
  throw new Error(
    "usage: canonical-preprod-cloudflare-named-launcher.mjs <cloudflared> <runtime-root> <keychain-service> <keychain-account>",
  );
}
if (!isAbsolute(cloudflared) || !statSync(cloudflared).isFile()) {
  throw new Error("cloudflared executable must be an absolute file path");
}
if (!isAbsolute(runtimeRootInput)) {
  throw new Error("named Tunnel runtime root must be absolute");
}
if (!/^[A-Za-z0-9._:-]+$/u.test(keychainService)) {
  throw new Error("named Tunnel Keychain service contains unsupported characters");
}
if (!/^[A-Za-z0-9._:@-]+$/u.test(keychainAccount)) {
  throw new Error("named Tunnel Keychain account contains unsupported characters");
}

const runtimeRoot = resolve(runtimeRootInput);
if (runtimeRoot === localRoot || !runtimeRoot.startsWith(`${localRoot}/`)) {
  throw new Error("named Tunnel runtime root must be task-owned below repository .local");
}

const assertDirectory = (path) => {
  const entry = lstatSync(path);
  if (entry.isSymbolicLink() || !entry.isDirectory()) {
    throw new Error(`named Tunnel runtime path must be a real directory: ${path}`);
  }
};

assertDirectory(runtimeRoot);
const cloudflaredHome = join(runtimeRoot, "cloudflared-home");
const cloudflaredConfig = join(cloudflaredHome, ".cloudflared");
const xdgConfig = join(runtimeRoot, "xdg-config");
const xdgData = join(runtimeRoot, "xdg-data");
const tmpDirectory = join(runtimeRoot, "tmp");
for (const path of [cloudflaredHome, cloudflaredConfig, xdgConfig, xdgData, tmpDirectory]) {
  assertDirectory(path);
}
if (readdirSync(cloudflaredConfig).length !== 0) {
  throw new Error("named Tunnel cloudflared config directory must remain empty");
}

const pidPath = join(runtimeRoot, "cloudflared.pid");
const logPath = join(runtimeRoot, "cloudflared.log");
if (existsSync(pidPath)) {
  throw new Error("named Tunnel PID file already exists");
}

let keychainValue;
try {
  keychainValue = execFileSync(
    "/usr/bin/security",
    ["find-generic-password", "-a", keychainAccount, "-s", keychainService, "-w"],
    { encoding: "buffer", maxBuffer: 8192, stdio: ["ignore", "pipe", "pipe"] },
  );
} catch {
  throw new Error("named Tunnel connector token is unavailable in macOS Keychain");
}

let tokenLength;
try {
  ({ byteLength: tokenLength } = validateCloudflareTunnelToken(keychainValue));
} catch {
  keychainValue.fill(0);
  throw new Error(
    "named Tunnel connector token in macOS Keychain is malformed or truncated; store the complete eyJ... value",
  );
}
const tokenBytes = keychainValue.subarray(0, tokenLength);

const logFd = openSync(logPath, "a", 0o600);
chmodSync(logPath, 0o600);
const environment = {
  PATH: "/usr/bin:/bin",
  HOME: join(runtimeRoot, "cloudflared-home"),
  XDG_CONFIG_HOME: join(runtimeRoot, "xdg-config"),
  XDG_DATA_HOME: join(runtimeRoot, "xdg-data"),
  TMPDIR: join(runtimeRoot, "tmp"),
  NO_PROXY: "127.0.0.1,localhost",
};

let child;
let tokenPayload;
let tokenPipe;
let removeSignalCleanup;
try {
  child = spawn(
    cloudflared,
    ["tunnel", "--no-autoupdate", "--protocol", "http2", "run", "--token-file", "/dev/fd/3"],
    {
      cwd: runtimeRoot,
      detached: true,
      env: environment,
      stdio: ["ignore", logFd, logFd, "pipe"],
    },
  );
  await new Promise((resolveSpawn, rejectSpawn) => {
    child.once("error", rejectSpawn);
    child.once("spawn", resolveSpawn);
  });
  if (!Number.isSafeInteger(child.pid) || child.pid <= 0) {
    throw new Error("cloudflared did not provide a process ID");
  }

  // Record ownership before the connector receives its token. If this
  // launcher is interrupted at any later point, both its own signal handler
  // and the parent lifecycle trap can identify and remove the exact exposure.
  recordOwnedChild(pidPath, child.pid);
  removeSignalCleanup = installOwnedChildSignalCleanup(child, pidPath);

  tokenPayload = Buffer.alloc(tokenBytes.length + 1);
  tokenBytes.copy(tokenPayload);
  tokenPayload[tokenPayload.length - 1] = 0x0a;
  tokenPipe = child.stdio[3];
  await new Promise((resolveWrite, rejectWrite) => {
    tokenPipe.once("error", rejectWrite);
    tokenPipe.end(tokenPayload, resolveWrite);
  });
  // A detached child does not keep the launcher alive, but an inherited pipe
  // can. cloudflared has received EOF at this point, so close and unref the
  // parent handle before waiting for early token rejection.
  tokenPipe.destroy();
  tokenPipe.unref?.();

  const earlyExit = await new Promise((resolveEarlyExit) => {
    if (child.exitCode !== null || child.signalCode !== null) {
      resolveEarlyExit(true);
      return;
    }
    const onExit = () => {
      clearTimeout(timer);
      resolveEarlyExit(true);
    };
    const timer = setTimeout(() => {
      child.off("exit", onExit);
      resolveEarlyExit(false);
    }, 750);
    child.once("exit", onExit);
  });
  if (earlyExit) {
    throw new Error("cloudflared exited while validating the named Tunnel connector token");
  }

  removeSignalCleanup();
  removeSignalCleanup = undefined;
  child.unref();
  process.stdout.write(`${child.pid}\n`);
} catch (error) {
  removeSignalCleanup?.();
  await terminateOwnedChild(child, pidPath);
  throw error;
} finally {
  tokenPipe?.destroy();
  keychainValue.fill(0);
  tokenPayload?.fill(0);
  closeSync(logFd);
}
