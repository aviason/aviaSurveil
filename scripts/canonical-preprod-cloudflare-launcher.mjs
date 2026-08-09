#!/usr/bin/env node
import { spawn } from "node:child_process";
import {
  chmodSync,
  closeSync,
  existsSync,
  lstatSync,
  openSync,
  readdirSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { dirname, isAbsolute, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const repositoryRoot = resolve(scriptDirectory, "..");
const localRoot = resolve(repositoryRoot, ".local");
const [cloudflared, portText, runtimeRootInput] = process.argv.slice(2);

if (process.argv.length !== 5) {
  throw new Error("usage: canonical-preprod-cloudflare-launcher.mjs <cloudflared> <port> <runtime-root>");
}
if (!isAbsolute(cloudflared) || !statSync(cloudflared).isFile()) {
  throw new Error("cloudflared executable must be an absolute file path");
}
if (!/^[0-9]+$/u.test(portText)) {
  throw new Error("Quick Tunnel port must be numeric");
}
const port = Number(portText);
if (!Number.isSafeInteger(port) || port < 1024 || port > 65535) {
  throw new Error("Quick Tunnel port must be a user-space TCP port");
}
if (!isAbsolute(runtimeRootInput)) {
  throw new Error("Quick Tunnel runtime root must be absolute");
}
const runtimeRoot = resolve(runtimeRootInput);
if (runtimeRoot === localRoot || !runtimeRoot.startsWith(`${localRoot}/`)) {
  throw new Error("Quick Tunnel runtime root must be task-owned below repository .local");
}

const assertDirectory = (path) => {
  const entry = lstatSync(path);
  if (entry.isSymbolicLink() || !entry.isDirectory()) {
    throw new Error(`Quick Tunnel runtime path must be a real directory: ${path}`);
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
  throw new Error("Quick Tunnel cloudflared config directory must remain empty");
}

const pidPath = join(runtimeRoot, "cloudflared.pid");
const logPath = join(runtimeRoot, "cloudflared.log");
if (existsSync(pidPath)) {
  throw new Error("Quick Tunnel PID file already exists");
}

const localOrigin = `http://127.0.0.1:${port}`;
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
try {
  child = spawn(cloudflared, ["tunnel", "--url", localOrigin], {
    cwd: runtimeRoot,
    detached: true,
    env: environment,
    stdio: ["ignore", logFd, logFd],
  });
  await new Promise((resolveSpawn, rejectSpawn) => {
    child.once("error", rejectSpawn);
    child.once("spawn", resolveSpawn);
  });
  if (!Number.isSafeInteger(child.pid) || child.pid <= 0) {
    throw new Error("cloudflared did not provide a process ID");
  }
  writeFileSync(pidPath, `${child.pid}\n`, { mode: 0o600 });
  chmodSync(pidPath, 0o600);
  child.unref();
  process.stdout.write(`${child.pid}\n`);
} catch (error) {
  // This process just spawned the child and has not detached it without an
  // ownership record.  If writing that record failed, remove that exact child
  // rather than leaving an unauditable public exposure behind.
  if (Number.isSafeInteger(child?.pid) && child.pid > 0) {
    try {
      process.kill(child.pid, "SIGTERM");
    } catch {
      // A failed launch can already have exited; preserve the original error.
    }
  }
  throw error;
} finally {
  closeSync(logFd);
}
