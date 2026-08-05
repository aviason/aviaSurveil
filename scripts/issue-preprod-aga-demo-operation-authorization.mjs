#!/usr/bin/env node

// This is the operator-side issuer for the legacy candidate-overlay control
// plane. The connected harness deliberately does not import or call this
// module: every overlay operation must receive a separately issued token.
import { createHash, randomBytes } from "node:crypto";
import { closeSync, fsyncSync, lstatSync, mkdirSync, openSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const DIGEST = /^sha256:[a-f0-9]{64}$/u;
const OPERATIONS = new Set(["LOAD_AGA_CANDIDATE_DEMO_OVERLAY", "CLEANUP_AGA_CANDIDATE_DEMO_OVERLAY"]);
const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");

function fail(message) {
  throw new Error(`ERR_AGA_DEMO_OPERATOR_AUTH ${message}`);
}

function parseArgs(argv) {
  const values = new Map();
  for (let index = 0; index < argv.length; index += 2) {
    const key = argv[index];
    const value = argv[index + 1];
    if (!key?.startsWith("--") || value === undefined || values.has(key)) fail("ARGUMENTS");
    values.set(key.slice(2), value);
  }
  return values;
}

function privateAbsolute(path, label) {
  if (!path?.startsWith("/")) fail(`${label}_PATH`);
  const resolved = resolve(path);
  if (resolved !== path || resolved.startsWith(`${root}/`)) fail(`${label}_SCOPE`);
  return resolved;
}

function digest(value) {
  return `sha256:${createHash("sha256").update(value).digest("hex")}`;
}

const values = parseArgs(process.argv.slice(2));
try {
  const intentPath = privateAbsolute(values.get("intent"), "INTENT");
  const outputPath = privateAbsolute(values.get("output"), "OUTPUT");
  const operation = values.get("operation");
  const issuer = values.get("issuer");
  if (!OPERATIONS.has(operation) || !issuer || issuer.length > 128 || !/^[A-Za-z0-9._:-]+$/u.test(issuer)) fail("HEADER");
  const intentStat = lstatSync(intentPath);
  if (!intentStat.isFile() || intentStat.isSymbolicLink() || intentStat.mode & 0o077) fail("INTENT_FILE");
  const intent = JSON.parse(readFileSync(intentPath, "utf8"));
  for (const field of ["runId", "intentDigest", "targetFingerprintDigest", "packageZipDigest", "codeDigest", "contractDigest"]) {
    if (typeof intent[field] !== "string") fail(`INTENT_${field}`);
  }
  for (const field of ["intentDigest", "targetFingerprintDigest", "packageZipDigest", "codeDigest", "contractDigest"]) if (!DIGEST.test(intent[field])) fail(`INTENT_${field}`);
  const expiresSeconds = Number(values.get("expires-seconds") ?? "600");
  if (!Number.isInteger(expiresSeconds) || expiresSeconds < 30 || expiresSeconds > 900) fail("EXPIRY");
  const now = new Date();
  const authorization = {
    schemaVersion: "preprod-aga-candidate-demo-operation-authorization/v1",
    token: randomBytes(32).toString("hex"),
    operation,
    issuer,
    expiresAt: new Date(now.getTime() + expiresSeconds * 1000).toISOString(),
    nonce: randomBytes(16).toString("hex"),
    runId: intent.runId,
    intentDigest: intent.intentDigest,
    targetFingerprintDigest: intent.targetFingerprintDigest,
    inputDigest: intent.packageZipDigest,
    codeDigest: intent.codeDigest,
    contractDigest: intent.contractDigest,
  };
  mkdirSync(dirname(outputPath), { recursive: true, mode: 0o700 });
  const descriptor = openSync(outputPath, "wx", 0o600);
  try { writeFileSync(descriptor, `${JSON.stringify(authorization)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
  const parent = openSync(dirname(outputPath), "r");
  try { fsyncSync(parent); } finally { closeSync(parent); }
  process.stdout.write(`${JSON.stringify({ schemaVersion: "aga-demo-operator-issuance-receipt/v1", operation, authorizationHash: digest(authorization.token), intentDigest: intent.intentDigest, targetFingerprintDigest: intent.targetFingerprintDigest })}\n`);
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : "ERR_AGA_DEMO_OPERATOR_AUTH UNEXPECTED"}\n`);
  process.exitCode = 1;
}
