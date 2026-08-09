import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const quickTunnelHostname = /^(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)\.trycloudflare\.com$/u;

const trimLogPunctuation = (value) => value.replace(/[),;.!?\]]+$/u, "");

/**
 * Returns the only HTTPS trycloudflare origin published in a cloudflared log.
 * A Quick Tunnel can reconnect, but it must not silently switch this local
 * profile to a different public origin after the API/OIDC configuration has
 * been materialized.
 */
export function extractQuickTunnelOrigin(log) {
  if (typeof log !== "string") {
    throw new TypeError("Quick Tunnel log must be text");
  }

  const candidates = (log.match(/https?:\/\/[^\s"'<>]+/giu) ?? [])
    .map(trimLogPunctuation)
    .filter((candidate) => candidate.toLowerCase().includes("trycloudflare.com"));

  if (candidates.length === 0) {
    throw new Error("cloudflared did not publish a trycloudflare HTTPS origin");
  }

  const origins = new Set();
  for (const candidate of candidates) {
    let parsed;
    try {
      parsed = new URL(candidate);
    } catch {
      throw new Error("cloudflared published an invalid Quick Tunnel URL");
    }

    if (parsed.protocol !== "https:") {
      throw new Error("Quick Tunnel URL must use HTTPS");
    }
    if (
      parsed.username ||
      parsed.password ||
      parsed.port ||
      parsed.pathname !== "/" ||
      parsed.search ||
      parsed.hash ||
      candidate !== parsed.origin ||
      !quickTunnelHostname.test(parsed.hostname)
    ) {
      throw new Error("Quick Tunnel URL must be a bare trycloudflare HTTPS origin");
    }
    origins.add(parsed.origin);
  }

  if (origins.size !== 1) {
    throw new Error("cloudflared published more than exactly one Quick Tunnel origin");
  }
  return [...origins][0];
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  if (process.argv.length !== 4 || process.argv[2] !== "--file") {
    throw new Error("usage: canonical-preprod-quick-tunnel-url.mjs --file <log-file>");
  }
  process.stdout.write(`${extractQuickTunnelOrigin(readFileSync(process.argv[3], "utf8"))}\n`);
}
