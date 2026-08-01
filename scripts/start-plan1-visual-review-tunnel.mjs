#!/usr/bin/env node
import { spawn } from "node:child_process";
import { appendFileSync, existsSync, mkdirSync, openSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, "..");
const reviewRoot = resolve(repoRoot, ".local/aviasurveil360/visual-review");
const runtimeDir = resolve(reviewRoot, "runtime");
const serverPidPath = resolve(runtimeDir, "server.pid");
const tunnelPidPath = resolve(runtimeDir, "cloudflared.pid");
const serverLogPath = resolve(runtimeDir, "server.log");
const tunnelLogPath = resolve(runtimeDir, "cloudflared.log");
const cloudflaredPath = "/opt/homebrew/opt/cloudflared/bin/cloudflared";
const port = Number(process.env.PORT || 4197);

mkdirSync(runtimeDir, { recursive: true });

function stopPid(path) {
  if (!existsSync(path)) return;
  const pid = Number(readFileSync(path, "utf8").trim());
  if (!Number.isFinite(pid) || pid <= 0) return;
  try {
    process.kill(pid, "SIGTERM");
  } catch {
    // The process may already be gone.
  }
}

function sleep(ms) {
  return new Promise((resolveSleep) => setTimeout(resolveSleep, ms));
}

async function waitFor(predicate, timeoutMs, label) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const value = await predicate();
    if (value) return value;
    await sleep(500);
  }
  throw new Error(`Timed out waiting for ${label}.`);
}

function startDetached(command, args, logPath, env = {}) {
  const startOffset = existsSync(logPath) ? statSync(logPath).size : 0;
  appendFileSync(logPath, `\n\n=== ${new Date().toISOString()} ${command} ${args.join(" ")} ===\n`);
  const logFd = openSync(logPath, "a");
  const child = spawn(command, args, {
    cwd: repoRoot,
    detached: true,
    env: { ...process.env, ...env },
    stdio: ["ignore", logFd, logFd],
  });
  child.unref();
  return { pid: child.pid, startOffset };
}

async function getJson(url) {
  try {
    const response = await fetch(url);
    if (!response.ok) return null;
    return await response.json();
  } catch {
    return null;
  }
}

function readLog(path, offset = 0) {
  return existsSync(path) ? readFileSync(path, "utf8").slice(offset) : "";
}

function lastTunnelUrlFromLog(log) {
  const matches = [...log.matchAll(/https:\/\/[a-z0-9-]+\.trycloudflare\.com/gi)].map((match) => match[0]);
  return matches.at(-1) ?? null;
}

stopPid(tunnelPidPath);
stopPid(serverPidPath);
await sleep(700);

const serverProcess = startDetached(process.execPath, ["scripts/plan1-visual-stakeholder-review.mjs"], serverLogPath, {
  PORT: String(port),
});
writeFileSync(serverPidPath, `${serverProcess.pid}\n`);

const localUrl = await waitFor(async () => {
  const state = await getJson(`http://127.0.0.1:${port}/api/state`);
  return state ? `http://127.0.0.1:${port}` : null;
}, 15_000, "local review server");

const tunnelProcess = startDetached(
  cloudflaredPath,
  ["tunnel", "--url", localUrl, "--no-autoupdate"],
  tunnelLogPath,
);
writeFileSync(tunnelPidPath, `${tunnelProcess.pid}\n`);

const publicUrl = await waitFor(() => {
  return lastTunnelUrlFromLog(readLog(tunnelLogPath, tunnelProcess.startOffset));
}, 120_000, "Cloudflare quick tunnel URL");

await waitFor(async () => {
  const state = await getJson(`${publicUrl}/api/state`);
  return state ? state : null;
}, 120_000, "public tunnel API");

console.log(publicUrl);
console.log(`serverPid=${serverProcess.pid}`);
console.log(`tunnelPid=${tunnelProcess.pid}`);
console.log(`serverLog=${serverLogPath}`);
console.log(`tunnelLog=${tunnelLogPath}`);
