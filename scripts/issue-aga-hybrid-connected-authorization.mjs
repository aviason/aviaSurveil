#!/usr/bin/env node

import { createHash, randomBytes } from "node:crypto";
import { existsSync, lstatSync, openSync, writeFileSync, fsyncSync, closeSync, chmodSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const MAX_LIFETIME_MS = 15 * 60 * 1000;
const AUTHORIZATION_KEYS = Object.freeze([
  "schemaVersion",
  "authorizationId",
  "issuer",
  "nonce",
  "issuedAt",
  "expiresAt",
  "inputDigest",
  "codeDigest",
  "contractDigest",
  "phase",
  "targetMode",
  "targetFingerprintDigest",
  "intentDigest",
  "journalDigest",
  "bundleDigest",
  "outerIntentDigest",
  "caseTargetFingerprints",
  "operations",
  "tokens",
  "operationTokenHashes",
]);
const PHASE_OPERATIONS = Object.freeze({
  prepare: [
    "CREATE_BASE",
    "QUALIFY_EXISTING_SYNTHETIC_OIDC",
    "PIN_PRE_WORKSPACE_FORBIDDEN_BASELINE",
    "PROVISION_EMPTY_WORKSPACE_CONTRACT",
    "EXPORT_WORKSPACE_FIXTURE",
    "PREPARE_LOAD_INTENTS",
    "CLEANUP_ON_PREPARE_FAILURE",
  ],
  qualification: [
    "LOAD_AGA_CANDIDATE_DEMO_OVERLAY",
    "RUN_WORKSPACE_LOAD_SEAL_BARRIERS",
    "LOAD_AND_SEAL_AGA_DEMO_WORKSPACE",
    "RUN_CONNECTED_AGA_HYBRID_QUALIFICATION",
    "CLEANUP_AGA_CANDIDATE_DEMO_OVERLAY",
    "CLEANUP_WHOLE_DISPOSABLE_NAMESPACE",
  ],
  recovery: ["RESUME_PREPARE", "RESUME_QUALIFICATION", "CLEANUP_WHOLE_DISPOSABLE_NAMESPACE"],
  cleanup: ["CLEANUP_WHOLE_DISPOSABLE_NAMESPACE"],
  "fault-matrix-prepare": ["PREPARE_FAULT_MATRIX", "CLEANUP_FAULT_MATRIX_ON_FAILURE"],
  "fault-matrix-run": [
    "RUN_INHERITED_BASE_RECEIPT_GAP",
    "RUN_WORKSPACE_TRANSACTION_RECEIPT_GAP",
    "RUN_CONCURRENT_TOKEN_RESERVATION",
    "RUN_CLEANUP_RECEIPT_GAP",
  ],
  "fault-matrix-recovery": ["RESUME_FAULT_MATRIX", "CLEANUP_FAULT_MATRIX"],
});

function fail(code, detail = "") {
  const suffix = detail ? ` ${detail}` : "";
  throw new Error(`ERR_AGA_HYBRID_AUTHORIZATION_${code}${suffix}`);
}

function parseArguments(argv) {
  const values = new Map();
  for (let index = 0; index < argv.length; index += 2) {
    const key = argv[index];
    const value = argv[index + 1];
    if (!key?.startsWith("--") || value === undefined || values.has(key)) fail("ARGUMENTS_INVALID");
    values.set(key.slice(2), value);
  }
  return values;
}

function digest(value) {
  return `sha256:${createHash("sha256").update(value).digest("hex")}`;
}

function requireDigest(values, name) {
  const value = values.get(name);
  if (!/^sha256:[a-f0-9]{64}$/u.test(value ?? "")) fail("DIGEST_INVALID", name);
  return value;
}

function exactOperations(phase, values) {
  const expected = PHASE_OPERATIONS[phase];
  if (!expected) fail("PHASE_INVALID", phase);
  const supplied = (values.get("operations") ?? "").split(",").filter(Boolean);
  if (supplied.length !== expected.length || supplied.some((operation, index) => operation !== expected[index])) fail("OPERATIONS_INVALID", phase);
  return expected;
}

function validatePrivateOutput(output) {
  if (!output || !output.startsWith("/")) fail("OUTPUT_NOT_ABSOLUTE");
  const parent = dirname(output);
  if (!existsSync(parent) || lstatSync(parent).isSymbolicLink() || (lstatSync(parent).mode & 0o777) !== 0o700) fail("OUTPUT_PARENT_INVALID");
  if (existsSync(output)) fail("OUTPUT_EXISTS");
}

function authorizationFrom(values) {
  const phase = values.get("phase");
  if (!phase) fail("PHASE_REQUIRED");
  const operations = exactOperations(phase, values);
  const targetMode = values.get("target-mode") ?? (phase === "prepare" || phase === "fault-matrix-prepare" ? "CREATE_FRESH_DISPOSABLE" : "EXACT_TARGET");
  if (!["CREATE_FRESH_DISPOSABLE", "EXACT_TARGET"].includes(targetMode)) fail("TARGET_MODE_INVALID");
  if (targetMode === "CREATE_FRESH_DISPOSABLE" && (values.has("target-fingerprint") || values.has("intent-digest"))) fail("CREATE_TARGET_MUST_NOT_BE_BOUND");
  if (targetMode === "EXACT_TARGET" && (!/^sha256:[a-f0-9]{64}$/u.test(values.get("target-fingerprint") ?? "") || !/^sha256:[a-f0-9]{64}$/u.test(values.get("intent-digest") ?? ""))) fail("TARGET_BINDING_REQUIRED");
  const issuedAt = new Date();
  const expiresAt = new Date(issuedAt.getTime() + MAX_LIFETIME_MS - 1000);
  const tokens = operations.map(() => randomBytes(32).toString("hex"));
  const operationTokenHashes = Object.fromEntries(operations.map((operation, index) => [operation, digest(tokens[index])]));
  const output = {
    schemaVersion: "aga-hybrid-demo-connected-authorization/v1",
    authorizationId: `aga-auth-${randomBytes(16).toString("hex")}`,
    issuer: values.get("issuer") || fail("ISSUER_REQUIRED"),
    nonce: randomBytes(16).toString("hex"),
    issuedAt: issuedAt.toISOString(),
    expiresAt: expiresAt.toISOString(),
    inputDigest: requireDigest(values, "input-digest"),
    codeDigest: requireDigest(values, "code-digest"),
    contractDigest: requireDigest(values, "contract-digest"),
    phase,
    targetMode,
    operations,
    tokens,
    operationTokenHashes,
  };
  if (targetMode === "EXACT_TARGET") {
    output.targetFingerprintDigest = values.get("target-fingerprint");
    output.intentDigest = values.get("intent-digest");
  }
  for (const [key, name] of [["journal-digest", "journalDigest"], ["bundle-digest", "bundleDigest"], ["outer-intent-digest", "outerIntentDigest"]]) {
    if (values.has(key)) output[name] = requireDigest(values, key);
  }
  if (values.has("case-targets")) {
    const caseTargetFingerprints = values.get("case-targets").split(",").filter(Boolean);
    if (caseTargetFingerprints.some((value) => !/^sha256:[a-f0-9]{64}$/u.test(value)) || new Set(caseTargetFingerprints).size !== caseTargetFingerprints.length) fail("CASE_TARGETS_INVALID");
    output.caseTargetFingerprints = caseTargetFingerprints;
  }
  return output;
}

export function validateAuthorization(value, now = new Date()) {
  if (!value || value.schemaVersion !== "aga-hybrid-demo-connected-authorization/v1" || !PHASE_OPERATIONS[value.phase]) fail("DOCUMENT_INVALID");
  if (JSON.stringify(Object.keys(value).sort()) !== JSON.stringify(AUTHORIZATION_KEYS.filter((key) => Object.hasOwn(value, key)).sort())) fail("DOCUMENT_NOT_CLOSED");
  if (!/^aga-auth-[a-f0-9]{32}$/u.test(value.authorizationId) || !/^[a-f0-9]{32,}$/u.test(value.nonce)) fail("IDENTITY_INVALID");
  if (typeof value.issuer !== "string" || value.issuer.length < 1 || value.issuer.length > 128) fail("ISSUER_INVALID");
  for (const field of ["inputDigest", "codeDigest", "contractDigest"]) requireDigest(new Map([[field, value[field]]]), field);
  if (!Array.isArray(value.operations) || value.operations.join("\u0000") !== PHASE_OPERATIONS[value.phase].join("\u0000")) fail("OPERATIONS_INVALID");
  if (!Array.isArray(value.tokens) || value.tokens.length !== value.operations.length || new Set(value.tokens).size !== value.tokens.length || value.tokens.some((token) => !/^[a-f0-9]{64}$/u.test(token))) fail("TOKENS_INVALID");
  if (!value.operationTokenHashes || typeof value.operationTokenHashes !== "object" || Array.isArray(value.operationTokenHashes) || JSON.stringify(Object.keys(value.operationTokenHashes).sort()) !== JSON.stringify(value.operations.slice().sort())) fail("TOKEN_MAP_INVALID");
  const issuedAt = Date.parse(value.issuedAt);
  const expiresAt = Date.parse(value.expiresAt);
  if (!Number.isFinite(issuedAt) || !Number.isFinite(expiresAt) || expiresAt <= now.getTime() || expiresAt <= issuedAt || expiresAt - issuedAt > MAX_LIFETIME_MS) fail("EXPIRY_INVALID");
  if (!["CREATE_FRESH_DISPOSABLE", "EXACT_TARGET"].includes(value.targetMode)) fail("TARGET_MODE_INVALID");
  if (value.targetMode === "CREATE_FRESH_DISPOSABLE" && (Object.hasOwn(value, "targetFingerprintDigest") || Object.hasOwn(value, "intentDigest"))) fail("CREATE_TARGET_MUST_NOT_BE_BOUND");
  if (value.targetMode === "EXACT_TARGET" && (!/^sha256:[a-f0-9]{64}$/u.test(value.targetFingerprintDigest ?? "") || !/^sha256:[a-f0-9]{64}$/u.test(value.intentDigest ?? ""))) fail("TARGET_BINDING_REQUIRED");
  for (const field of ["journalDigest", "bundleDigest", "outerIntentDigest"]) if (Object.hasOwn(value, field)) requireDigest(new Map([[field, value[field]]]), field);
  if (Object.hasOwn(value, "caseTargetFingerprints") && (!Array.isArray(value.caseTargetFingerprints) || new Set(value.caseTargetFingerprints).size !== value.caseTargetFingerprints.length || value.caseTargetFingerprints.some((fingerprint) => !/^sha256:[a-f0-9]{64}$/u.test(fingerprint)))) fail("CASE_TARGETS_INVALID");
  for (const [index, operation] of value.operations.entries()) if (value.operationTokenHashes?.[operation] !== digest(value.tokens[index])) fail("TOKEN_HASH_MISMATCH", operation);
  return true;
}

export function issueAuthorization(values) {
  const outputPath = values.get("output");
  validatePrivateOutput(outputPath);
  const authorization = authorizationFrom(values);
  validateAuthorization(authorization, new Date(authorization.issuedAt));
  const bytes = Buffer.from(`${JSON.stringify(authorization, null, 2)}\n`, "utf8");
  const descriptor = openSync(outputPath, "wx", 0o600);
  try {
    writeFileSync(descriptor, bytes);
    fsyncSync(descriptor);
  } finally {
    closeSync(descriptor);
  }
  chmodSync(outputPath, 0o600);
  const parentDescriptor = openSync(dirname(outputPath), "r");
  try { fsyncSync(parentDescriptor); } finally { closeSync(parentDescriptor); }
  return { digest: digest(bytes), phase: authorization.phase, operationCount: authorization.operations.length };
}

if (process.argv[1] && fileURLToPath(import.meta.url) === resolve(process.argv[1])) {
  try {
    const result = issueAuthorization(parseArguments(process.argv.slice(2)));
    process.stdout.write(`aga-hybrid-authorization: issued digest=${result.digest} phase=${result.phase} operations=${result.operationCount}\n`);
  } catch (error) {
    process.stderr.write(`${error instanceof Error ? error.message : "ERR_AGA_HYBRID_AUTHORIZATION_UNEXPECTED"}\n`);
    process.exitCode = 1;
  }
}
