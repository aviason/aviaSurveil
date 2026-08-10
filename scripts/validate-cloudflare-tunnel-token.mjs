#!/usr/bin/env node
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";

const MAX_TOKEN_BYTES = 8192;
const REQUIRED_CONNECTOR_FIELDS = ["a", "t", "s"];

const invalidToken = () =>
  new Error(
    "Cloudflare Tunnel connector token is malformed or truncated; paste only the complete eyJ... value",
  );

export function validateCloudflareTunnelToken(tokenBytes) {
  if (!Buffer.isBuffer(tokenBytes) || tokenBytes.length === 0 || tokenBytes.length > MAX_TOKEN_BYTES) {
    throw invalidToken();
  }

  let byteLength = tokenBytes.length;
  if (byteLength > 0 && tokenBytes[byteLength - 1] === 0x0a) byteLength -= 1;
  if (byteLength > 0 && tokenBytes[byteLength - 1] === 0x0d) byteLength -= 1;
  if (byteLength === 0) throw invalidToken();

  const token = tokenBytes.toString("ascii", 0, byteLength);
  if (
    !token.startsWith("eyJ") ||
    !/^[A-Za-z0-9+/_-]+={0,2}$/u.test(token) ||
    tokenBytes.subarray(0, byteLength).some((byte) => byte <= 0x20 || byte >= 0x7f)
  ) {
    throw invalidToken();
  }

  const standardBase64 = token.replaceAll("-", "+").replaceAll("_", "/");
  const unpadded = standardBase64.replace(/=+$/u, "");
  const suppliedPadding = standardBase64.length - unpadded.length;
  if (unpadded.length % 4 === 1) throw invalidToken();
  const expectedPadding = (4 - (unpadded.length % 4)) % 4;
  if (suppliedPadding !== 0 && suppliedPadding !== expectedPadding) throw invalidToken();

  const padded = `${unpadded}${"=".repeat(expectedPadding)}`;
  const decoded = Buffer.from(padded, "base64");
  try {
    if (decoded.toString("base64").replace(/=+$/u, "") !== unpadded) {
      throw invalidToken();
    }

    let connector;
    try {
      connector = JSON.parse(decoded.toString("utf8"));
    } catch {
      throw invalidToken();
    }
    if (connector === null || typeof connector !== "object" || Array.isArray(connector)) {
      throw invalidToken();
    }
    for (const field of REQUIRED_CONNECTOR_FIELDS) {
      if (typeof connector[field] !== "string" || connector[field].length === 0) {
        throw invalidToken();
      }
    }
  } finally {
    decoded.fill(0);
  }

  return { byteLength };
}

async function main() {
  const chunks = [];
  let totalBytes = 0;
  try {
    for await (const chunk of process.stdin) {
      const bytes = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);
      chunks.push(bytes);
      totalBytes += bytes.length;
      if (totalBytes > MAX_TOKEN_BYTES) throw invalidToken();
    }

    const tokenBytes = Buffer.concat(chunks, totalBytes);
    try {
      validateCloudflareTunnelToken(tokenBytes);
    } finally {
      tokenBytes.fill(0);
    }
  } catch {
    process.stderr.write(
      "cloudflare-tunnel-token: connector token is malformed or truncated; paste only the complete eyJ... value\n",
    );
    process.exitCode = 1;
  } finally {
    for (const chunk of chunks) chunk.fill(0);
  }
}

const modulePath = fileURLToPath(import.meta.url);
if (process.argv[1] && resolve(process.argv[1]) === modulePath) {
  await main();
}
