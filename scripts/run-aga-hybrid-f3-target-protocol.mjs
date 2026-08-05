#!/usr/bin/env node

// F3 is a connected, target-bound recovery protocol. Each case owns a fresh
// Compose project and a fresh PostgreSQL database. The target process writes
// the effect and target receipt through PostgreSQL; the parent only publishes
// redacted receipt digests after the target has been stopped and cleaned.
import { createHash, randomBytes } from "node:crypto";
import {
  closeSync,
  existsSync,
  fsyncSync,
  lstatSync,
  mkdirSync,
  openSync,
  readFileSync,
  readdirSync,
  rmSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { spawn, spawnSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const CASES = Object.freeze([
  "INHERITED_BASE_RECEIPT_GAP",
  "WORKSPACE_TRANSACTION_RECEIPT_GAP",
  "CONCURRENT_TOKEN_RESERVATION",
  "CLEANUP_RECEIPT_GAP",
]);
const OPERATIONS = Object.freeze([
  "RUN_INHERITED_BASE_RECEIPT_GAP",
  "RUN_WORKSPACE_TRANSACTION_RECEIPT_GAP",
  "RUN_CONCURRENT_TOKEN_RESERVATION",
  "RUN_CLEANUP_RECEIPT_GAP",
]);
const RECOVERY_OPERATIONS = Object.freeze(["RESUME_FAULT_MATRIX", "CLEANUP_FAULT_MATRIX"]);
const EXECUTION_ORDER = Object.freeze([
  "CONCURRENT_TOKEN_RESERVATION",
  "INHERITED_BASE_RECEIPT_GAP",
  "WORKSPACE_TRANSACTION_RECEIPT_GAP",
  "CLEANUP_RECEIPT_GAP",
]);
const DIGEST = /^sha256:[a-f0-9]{64}$/u;
const POSTGRES_IMAGE = "postgres:17.6-alpine3.22@sha256:ef257d85f76e48da1c64832459b59fcaba1a4dac97bf5d7450c77753542eee94";
const SCRIPT_PATH = fileURLToPath(import.meta.url);
const root = resolve(dirname(SCRIPT_PATH), "..");

function fail(message) {
  throw new Error(`ERR_AGA_HYBRID_F3 ${message}`);
}

function digest(value) {
  return `sha256:${createHash("sha256").update(value).digest("hex")}`;
}

function stable(value) {
  if (Array.isArray(value)) return value.map(stable);
  if (value && typeof value === "object") return Object.fromEntries(Object.keys(value).sort().map((key) => [key, stable(value[key])]));
  return value;
}

class F3Interrupted extends Error {
  constructor(caseName) {
    super(`INTERRUPTED_AFTER_${caseName}`);
    this.exitCode = 73;
  }
}

function writeOnce(path, value) {
  mkdirSync(dirname(path), { recursive: true, mode: 0o700 });
  const descriptor = openSync(path, "wx", 0o600);
  try {
    writeFileSync(descriptor, Buffer.isBuffer(value) ? value : typeof value === "string" ? value : `${JSON.stringify(value, null, 2)}\n`);
    fsyncSync(descriptor);
  } finally {
    closeSync(descriptor);
  }
  const parent = openSync(dirname(path), "r");
  try { fsyncSync(parent); } finally { closeSync(parent); }
}

function readJSON(path) {
  if (!existsSync(path) || lstatSync(path).isSymbolicLink() || (statSync(path).mode & 0o777) !== 0o600) fail("PRIVATE_FILE");
  return JSON.parse(readFileSync(path, "utf8"));
}

function parseArgs(argv) {
  const values = new Map();
  for (let index = 0; index < argv.length; index += 2) {
    if (!argv[index]?.startsWith("--") || argv[index + 1] === undefined || values.has(argv[index])) fail("ARGUMENTS");
    values.set(argv[index].slice(2), argv[index + 1]);
  }
  return values;
}

function validateManifest(manifest) {
  const keys = ["schemaVersion", "caseNames", "cases", "inputDigest", "codeDigest", "contractDigest", "manifestDigest"];
  if (!manifest || JSON.stringify(Object.keys(manifest).sort()) !== JSON.stringify(keys.slice().sort()) || manifest.schemaVersion !== "aga-hybrid-demo-fault-matrix-manifest/v2" || JSON.stringify(manifest.caseNames) !== JSON.stringify(CASES) || !Array.isArray(manifest.cases) || manifest.cases.length !== CASES.length || !DIGEST.test(manifest.inputDigest ?? "") || !DIGEST.test(manifest.codeDigest ?? "") || !DIGEST.test(manifest.contractDigest ?? "") || !DIGEST.test(manifest.manifestDigest ?? "")) fail("MANIFEST_HEADER");
  const seen = new Set();
  for (const entry of manifest.cases) {
    const entryKeys = ["caseName", "targetFingerprintDigest", "targetNamespace", "targetManifestDigest"];
    if (JSON.stringify(Object.keys(entry).sort()) !== JSON.stringify(entryKeys.slice().sort()) || !CASES.includes(entry.caseName) || seen.has(entry.caseName) || !DIGEST.test(entry.targetFingerprintDigest ?? "") || !DIGEST.test(entry.targetManifestDigest ?? "") || !/^[a-z0-9][a-z0-9-]{2,48}$/u.test(entry.targetNamespace)) fail("MANIFEST_CASE");
    const expectedTargetFingerprint = digest(`AGA-F3-TARGET-V3\n${entry.caseName}\n${entry.targetNamespace}\n${manifest.inputDigest}\n${manifest.codeDigest}\n${manifest.contractDigest}`);
    const expectedTargetManifest = targetManifest(manifest, entry);
    if (entry.targetFingerprintDigest !== expectedTargetFingerprint || entry.targetManifestDigest !== expectedTargetManifest.targetManifestDigest) fail("MANIFEST_TARGET_BINDING");
    seen.add(entry.caseName);
  }
  if (seen.size !== CASES.length) fail("MANIFEST_CASE_SET");
  const body = { schemaVersion: manifest.schemaVersion, caseNames: manifest.caseNames, cases: manifest.cases, inputDigest: manifest.inputDigest, codeDigest: manifest.codeDigest, contractDigest: manifest.contractDigest };
  if (manifest.manifestDigest !== digest(JSON.stringify(stable(body)))) fail("MANIFEST_DIGEST");
  return manifest;
}

function targetSetDigest(manifest) {
  return digest(JSON.stringify(stable(manifest.cases.map(({ caseName, targetFingerprintDigest, targetNamespace }) => ({ caseName, targetFingerprintDigest, targetNamespace })))));
}

function outerIntentDigest(manifest) {
  return digest(`AGA-F3-OUTER-INTENT-V3\n${manifest.manifestDigest}\n${targetSetDigest(manifest)}`);
}

function validateAuthorization(authorization, manifest) {
  const expectedPhase = authorization?.phase;
  if (!["fault-matrix-run", "fault-matrix-recovery"].includes(expectedPhase)) fail("AUTH_PHASE");
  const keys = ["schemaVersion", "authorizationId", "issuer", "nonce", "issuedAt", "expiresAt", "inputDigest", "codeDigest", "contractDigest", "phase", "targetMode", "targetFingerprintDigest", "intentDigest", "journalDigest", "bundleDigest", "outerIntentDigest", "caseTargetFingerprints", "operations", "tokens", "operationTokenHashes"];
  if (!authorization || authorization.schemaVersion !== "aga-hybrid-demo-connected-authorization/v1" || JSON.stringify(Object.keys(authorization).sort()) !== JSON.stringify(keys.slice().sort()) || !DIGEST.test(authorization.inputDigest ?? "") || !DIGEST.test(authorization.codeDigest ?? "") || !DIGEST.test(authorization.contractDigest ?? "") || !DIGEST.test(authorization.targetFingerprintDigest ?? "") || !DIGEST.test(authorization.intentDigest ?? "") || !DIGEST.test(authorization.bundleDigest ?? "") || !DIGEST.test(authorization.outerIntentDigest ?? "") || authorization.inputDigest !== manifest.inputDigest || authorization.codeDigest !== manifest.codeDigest || authorization.contractDigest !== manifest.contractDigest || authorization.targetFingerprintDigest !== targetSetDigest(manifest) || authorization.intentDigest !== outerIntentDigest(manifest) || authorization.bundleDigest !== manifest.manifestDigest || authorization.outerIntentDigest !== outerIntentDigest(manifest)) fail("AUTH_HEADER");
  const expiresAt = Date.parse(authorization.expiresAt);
  const issuedAt = Date.parse(authorization.issuedAt);
  if (!Number.isFinite(issuedAt) || !Number.isFinite(expiresAt) || expiresAt <= Date.now() || expiresAt <= issuedAt || expiresAt - issuedAt > 15 * 60 * 1000) fail("AUTH_EXPIRY");
  const expectedOperations = expectedPhase === "fault-matrix-run" ? OPERATIONS : RECOVERY_OPERATIONS;
  if (JSON.stringify(authorization.operations) !== JSON.stringify(expectedOperations) || !Array.isArray(authorization.tokens) || authorization.tokens.length !== expectedOperations.length || !Array.isArray(authorization.caseTargetFingerprints) || JSON.stringify(authorization.caseTargetFingerprints) !== JSON.stringify(manifest.cases.map(({ targetFingerprintDigest }) => targetFingerprintDigest))) fail("AUTH_CASE_BINDING");
  if (expectedPhase === "fault-matrix-recovery" && !DIGEST.test(authorization.journalDigest ?? "")) fail("AUTH_JOURNAL_BINDING");
  for (const [index, operation] of expectedOperations.entries()) {
    const token = authorization.tokens[index];
    if (!/^[a-f0-9]{64}$/u.test(token) || authorization.operationTokenHashes?.[operation] !== digest(token)) fail("AUTH_TOKEN_BINDING");
  }
  return authorization;
}

function targetProjectName(namespace) {
  return `aviasurveil360-aga-f3-${namespace}`;
}

function targetManifest(manifest, entry) {
  const body = {
    schemaVersion: "aga-hybrid-f3-target-manifest/v3",
    caseName: entry.caseName,
    targetFingerprintDigest: entry.targetFingerprintDigest,
    targetNamespace: entry.targetNamespace,
    targetProjectName: targetProjectName(entry.targetNamespace),
    targetDatabaseName: "aga_f3",
    targetNamespaceDigest: digest(`AGA-F3-NAMESPACE-V3\n${entry.targetNamespace}`),
    inputDigest: manifest.inputDigest,
    codeDigest: manifest.codeDigest,
    contractDigest: manifest.contractDigest,
  };
  return { ...body, targetManifestDigest: digest(JSON.stringify(stable(body))) };
}

function compose(runtime, args, input) {
  const result = spawnSync("docker", ["compose", "--project-name", runtime.projectName, "--file", runtime.composeFile, ...args], { encoding: "utf8", input, stdio: [input === undefined ? "ignore" : "pipe", "pipe", "pipe"] });
  if (result.error) throw result.error;
  if (result.status !== 0) fail(`COMPOSE_${args[0] ?? "COMMAND"}`);
  return result.stdout ?? "";
}

function composeResidue(projectName) {
  const query = (type) => {
    const result = spawnSync("docker", [type === "containers" ? "ps" : type === "volumes" ? "volume" : "network", ...(type === "containers" ? ["--all"] : ["ls"]), "--filter", `label=com.docker.compose.project=${projectName}`, "--format", "{{.ID}}"], { encoding: "utf8" });
    return result.status === 0 ? result.stdout.trim() : "unknown";
  };
  return { containers: query("containers"), volumes: query("volumes"), networks: query("networks") };
}

function quoteSQL(value) {
  if (typeof value !== "string" || /[\u0000]/u.test(value)) fail("SQL_VALUE");
  return `'${value.replaceAll("'", "''")}'`;
}

function targetSQL(runtime, sql) {
  const result = compose(runtime, ["exec", "--no-TTY", "target-db", "psql", "--username", "postgres", "--dbname", runtime.databaseName, "--tuples-only", "--no-align", "--quiet", "--set", "ON_ERROR_STOP=1", "--command", sql]);
  return result.trim();
}

function prepareCompose(targetRoot, manifest, entry) {
  const projectName = targetProjectName(entry.targetNamespace);
  const password = randomBytes(24).toString("hex");
  const composeFile = resolve(targetRoot, "compose.yaml");
  const composeSource = `services:\n  target-db:\n    image: ${POSTGRES_IMAGE}\n    environment:\n      POSTGRES_DB: aga_f3\n      POSTGRES_USER: postgres\n      POSTGRES_PASSWORD: ${password}\n    volumes:\n      - target-db-data:/var/lib/postgresql/data\n    healthcheck:\n      test: [CMD-SHELL, pg_isready -U postgres -d aga_f3]\n      interval: 1s\n      timeout: 3s\n      retries: 30\nvolumes:\n  target-db-data:\n`;
  writeOnce(composeFile, composeSource);
  const runtime = { schemaVersion: "aga-hybrid-f3-target-runtime/v1", projectName, composeFile, databaseName: "aga_f3" };
  writeOnce(resolve(targetRoot, "runtime.json"), runtime);
  compose(runtime, ["up", "--detach", "--wait"]);
  targetSQL(runtime, "CREATE SCHEMA IF NOT EXISTS aga_f3; CREATE TABLE IF NOT EXISTS aga_f3.operation_effects (token_hash text PRIMARY KEY, case_name text NOT NULL, target_fingerprint_digest text NOT NULL, target_manifest_digest text NOT NULL, input_digest text NOT NULL, code_digest text NOT NULL, contract_digest text NOT NULL, authorization_id_digest text NOT NULL, authorization_digest text NOT NULL, operation text NOT NULL, effect_digest text NOT NULL, target_receipt jsonb, created_at timestamptz NOT NULL DEFAULT clock_timestamp()); CREATE TABLE IF NOT EXISTS aga_f3.recovery_receipts (recovery_digest text PRIMARY KEY, token_hash text NOT NULL, case_name text NOT NULL, origin text NOT NULL, target_receipt jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp());");
  return runtime;
}

function targetArgs(targetMode, targetRoot, manifestPath, operation, token, binding = {}) {
  const args = [SCRIPT_PATH, "--target-mode", targetMode, "--target-root", targetRoot, "--manifest", manifestPath, "--operation", operation, "--token", token];
  for (const [key, value] of Object.entries(binding)) if (value !== undefined && value !== "") args.push(`--${key}`, value);
  return args;
}

function runTarget(targetRoot, manifestPath, operation, token, targetMode, binding) {
  const result = spawnSync(process.execPath, targetArgs(targetMode, targetRoot, manifestPath, operation, token, binding), { encoding: "utf8" });
  if (result.error) throw result.error;
  return { status: result.status, stdout: result.stdout, stderr: result.stderr };
}

function targetProcess(values) {
  const targetRoot = resolve(values.get("target-root"));
  const manifest = readJSON(resolve(values.get("manifest")));
  const runtime = readJSON(resolve(targetRoot, "runtime.json"));
  const operation = values.get("operation");
  const token = values.get("token");
  const mode = values.get("target-mode");
  const authorizationId = values.get("authorized-authorization-id");
  const authorizationDigest = values.get("authorized-authorization-digest");
  const authorizedBinding = {
    caseName: values.get("authorized-case-name"),
    targetFingerprintDigest: values.get("authorized-target-fingerprint"),
    targetNamespace: values.get("authorized-target-namespace"),
    targetManifestDigest: values.get("authorized-target-manifest-digest"),
    inputDigest: values.get("authorized-input-digest"),
    codeDigest: values.get("authorized-code-digest"),
    contractDigest: values.get("authorized-contract-digest"),
  };
  const targetBody = {
    schemaVersion: manifest.schemaVersion,
    caseName: manifest.caseName,
    targetFingerprintDigest: manifest.targetFingerprintDigest,
    targetNamespace: manifest.targetNamespace,
    targetProjectName: manifest.targetProjectName,
    targetDatabaseName: manifest.targetDatabaseName,
    targetNamespaceDigest: manifest.targetNamespaceDigest,
    inputDigest: manifest.inputDigest,
    codeDigest: manifest.codeDigest,
    contractDigest: manifest.contractDigest,
  };
  const expectedTargetManifest = targetManifest({ inputDigest: manifest.inputDigest, codeDigest: manifest.codeDigest, contractDigest: manifest.contractDigest }, manifest);
  const namespaceSeal = readJSON(resolve(targetRoot, "namespace-seal.json"));
  if (manifest.schemaVersion !== "aga-hybrid-f3-target-manifest/v3" || JSON.stringify(Object.keys(manifest).sort()) !== JSON.stringify([...Object.keys(targetBody), "targetManifestDigest"].sort()) || !DIGEST.test(manifest.targetFingerprintDigest ?? "") || !DIGEST.test(manifest.targetManifestDigest ?? "") || manifest.targetProjectName !== runtime.projectName || manifest.targetDatabaseName !== runtime.databaseName || manifest.targetManifestDigest !== digest(JSON.stringify(stable(targetBody))) || JSON.stringify(stable(manifest)) !== JSON.stringify(stable(expectedTargetManifest)) || JSON.stringify(Object.keys(namespaceSeal).sort()) !== JSON.stringify(["schemaVersion", "caseName", "targetFingerprintDigest", "targetManifestDigest", "namespaceDigest"].sort()) || namespaceSeal.schemaVersion !== "aga-hybrid-f3-namespace-seal/v2" || namespaceSeal.caseName !== manifest.caseName || namespaceSeal.targetFingerprintDigest !== manifest.targetFingerprintDigest || namespaceSeal.targetManifestDigest !== manifest.targetManifestDigest || namespaceSeal.namespaceDigest !== manifest.targetNamespaceDigest || JSON.stringify(authorizedBinding) !== JSON.stringify({ caseName: manifest.caseName, targetFingerprintDigest: manifest.targetFingerprintDigest, targetNamespace: manifest.targetNamespace, targetManifestDigest: manifest.targetManifestDigest, inputDigest: manifest.inputDigest, codeDigest: manifest.codeDigest, contractDigest: manifest.contractDigest }) || !/^aga-auth-[a-f0-9]{32}$/u.test(authorizationId ?? "") || !DIGEST.test(authorizationDigest ?? "") || !OPERATIONS.includes(operation) || !/^[a-f0-9]{64}$/u.test(token ?? "")) fail("TARGET_AUTHORITY_BINDING");
  const tokenHash = digest(token);
  const authorizationIdDigest = digest(authorizationId);
  const effect = { schemaVersion: "aga-hybrid-f3-target-effect/v4", caseName: manifest.caseName, targetFingerprintDigest: manifest.targetFingerprintDigest, targetManifestDigest: manifest.targetManifestDigest, inputDigest: manifest.inputDigest, codeDigest: manifest.codeDigest, contractDigest: manifest.contractDigest, authorizationIdDigest, authorizationDigest, operation, tokenHash, effectDigest: digest(`AGA-F3-EFFECT-V4\n${manifest.targetManifestDigest}\n${manifest.inputDigest}\n${manifest.codeDigest}\n${manifest.contractDigest}\n${authorizationDigest}\n${operation}`) };
  const effectDigest = effect.effectDigest;
  const transaction = { ...effect, schemaVersion: "aga-hybrid-f3-target-transaction/v4" };
  const transactionDigest = digest(JSON.stringify(stable(transaction)));
  const targetReceipt = { schemaVersion: "aga-hybrid-f3-target-result/v4", caseName: manifest.caseName, targetFingerprintDigest: manifest.targetFingerprintDigest, targetManifestDigest: manifest.targetManifestDigest, inputDigest: manifest.inputDigest, codeDigest: manifest.codeDigest, contractDigest: manifest.contractDigest, authorizationIdDigest, authorizationDigest, operation, tokenHash, transactionDigest, resultDigest: digest(`AGA-F3-RESULT-V4\n${transactionDigest}`), origin: "stored-target-result" };
  const transactionPath = resolve(targetRoot, "transactions/operation.json");
  const resultPath = resolve(targetRoot, "transactions/result.json");
  const recoveryPath = resolve(targetRoot, "transactions/recovery.json");
  const cleanupPath = resolve(targetRoot, "transactions/cleanup.json");
  const cleanupReplayPath = resolve(targetRoot, "transactions/cleanup-replay-rejected.json");
  const authorityAttemptId = values.get("authority-attempt") ?? "single";
  const authorityRacePath = resolve(targetRoot, `transactions/authority-race-${authorityAttemptId}.json`);
  const tokenSQL = quoteSQL(tokenHash);
  if (mode === "apply") {
    if (manifest.caseName === "CONCURRENT_TOKEN_RESERVATION") {
      const authorityRoot = resolve(values.get("authority-root") ?? "");
      if (!authorityRoot.startsWith("/") || !existsSync(authorityRoot) || lstatSync(authorityRoot).isSymbolicLink() || (statSync(authorityRoot).mode & 0o777) !== 0o700) fail("AUTHORITY_ROOT");
      const reservationDirectory = resolve(authorityRoot, "target-token-reservations");
      mkdirSync(reservationDirectory, { recursive: true, mode: 0o700 });
      const reservationPath = resolve(reservationDirectory, `${tokenHash.replace("sha256:", "")}.json`);
      const reservation = { schemaVersion: "aga-hybrid-f3-shared-token-reservation/v1", caseName: manifest.caseName, targetFingerprintDigest: manifest.targetFingerprintDigest, targetManifestDigest: manifest.targetManifestDigest, operation, tokenHash, authorizationIdDigest, authorizationDigest, reservationDigest: digest(`${manifest.caseName}\n${operation}\n${tokenHash}\n${authorizationDigest}`) };
      try {
        writeOnce(reservationPath, reservation);
        writeOnce(authorityRacePath, { schemaVersion: "aga-hybrid-f3-authority-race-receipt/v1", caseName: manifest.caseName, attemptId: authorityAttemptId, status: "WON", tokenHash, reservationDigest: reservation.reservationDigest });
      } catch (error) {
        if (error?.code !== "EEXIST") throw error;
        if (!existsSync(authorityRacePath)) writeOnce(authorityRacePath, { schemaVersion: "aga-hybrid-f3-authority-race-receipt/v1", caseName: manifest.caseName, attemptId: authorityAttemptId, status: "LOST", tokenHash, reservationDigest: reservation.reservationDigest });
        process.exit(17);
      }
    }
    const storedReceipt = manifest.caseName === "CONCURRENT_TOKEN_RESERVATION" || manifest.caseName === "CLEANUP_RECEIPT_GAP";
    const receiptSQL = storedReceipt ? `${quoteSQL(JSON.stringify(targetReceipt))}::jsonb` : "NULL::jsonb";
    const inserted = targetSQL(runtime, `BEGIN; INSERT INTO aga_f3.operation_effects (token_hash,case_name,target_fingerprint_digest,target_manifest_digest,input_digest,code_digest,contract_digest,authorization_id_digest,authorization_digest,operation,effect_digest,target_receipt) VALUES (${tokenSQL},${quoteSQL(manifest.caseName)},${quoteSQL(manifest.targetFingerprintDigest)},${quoteSQL(manifest.targetManifestDigest)},${quoteSQL(manifest.inputDigest)},${quoteSQL(manifest.codeDigest)},${quoteSQL(manifest.contractDigest)},${quoteSQL(authorizationIdDigest)},${quoteSQL(authorizationDigest)},${quoteSQL(operation)},${quoteSQL(effectDigest)},${receiptSQL}) ON CONFLICT (token_hash) DO NOTHING RETURNING token_hash; COMMIT;`);
    if (!inserted) process.exit(17);
    writeOnce(transactionPath, transaction);
    if (storedReceipt) writeOnce(resultPath, targetReceipt);
    if (manifest.caseName !== "CONCURRENT_TOKEN_RESERVATION") process.exit(73);
    process.stdout.write("EFFECT_AND_TARGET_RECEIPT_COMMITTED\n");
    return;
  }
  if (mode === "recover") {
    const rows = targetSQL(runtime, `SELECT json_build_object('tokenHash',token_hash,'caseName',case_name,'targetFingerprintDigest',target_fingerprint_digest,'targetManifestDigest',target_manifest_digest,'inputDigest',input_digest,'codeDigest',code_digest,'contractDigest',contract_digest,'authorizationIdDigest',authorization_id_digest,'authorizationDigest',authorization_digest,'operation',operation,'effectDigest',effect_digest,'targetReceipt',target_receipt)::text FROM aga_f3.operation_effects WHERE token_hash=${tokenSQL};`);
    if (!rows) fail("TARGET_EFFECT_MISSING");
    const stored = JSON.parse(rows);
    if (stored.caseName !== manifest.caseName || stored.targetFingerprintDigest !== manifest.targetFingerprintDigest || stored.targetManifestDigest !== manifest.targetManifestDigest || stored.inputDigest !== manifest.inputDigest || stored.codeDigest !== manifest.codeDigest || stored.contractDigest !== manifest.contractDigest || stored.authorizationIdDigest !== authorizationIdDigest || stored.authorizationDigest !== authorizationDigest || stored.operation !== operation || stored.effectDigest !== effectDigest) fail("RECOVERY_BINDING");
    const storedTransaction = { schemaVersion: "aga-hybrid-f3-target-transaction/v4", caseName: manifest.caseName, targetFingerprintDigest: manifest.targetFingerprintDigest, targetManifestDigest: manifest.targetManifestDigest, inputDigest: manifest.inputDigest, codeDigest: manifest.codeDigest, contractDigest: manifest.contractDigest, authorizationIdDigest, authorizationDigest, operation, tokenHash, effectDigest };
    const transactionValue = existsSync(transactionPath) ? readJSON(transactionPath) : storedTransaction;
    if (JSON.stringify(stable(transactionValue)) !== JSON.stringify(stable(storedTransaction))) fail("RECOVERY_TRANSACTION");
    if (!existsSync(transactionPath)) writeOnce(transactionPath, transactionValue);
    let origin = "stored-target-result";
    let result = stored.targetReceipt;
    if (!result) {
      origin = "recreated-after-effect";
      result = { ...targetReceipt, origin };
      const recoveryDigest = digest(`${manifest.caseName}\n${result.resultDigest}\n${origin}`);
      targetSQL(runtime, `INSERT INTO aga_f3.recovery_receipts (recovery_digest,token_hash,case_name,origin,target_receipt) VALUES (${quoteSQL(recoveryDigest)},${tokenSQL},${quoteSQL(manifest.caseName)},${quoteSQL(origin)},${quoteSQL(JSON.stringify(result))}::jsonb) ON CONFLICT (recovery_digest) DO NOTHING;`);
    }
    const normalizedResult = { ...result, origin: "stored-target-result" };
    if (JSON.stringify(stable(normalizedResult)) !== JSON.stringify(stable({ ...targetReceipt, origin: "stored-target-result" }))) fail("RECOVERY_RECEIPT");
    if (existsSync(resultPath)) {
      const existingResult = readJSON(resultPath);
      if (JSON.stringify(stable(existingResult)) !== JSON.stringify(stable(result))) fail("RECOVERY_RESULT");
    } else {
      writeOnce(resultPath, result);
    }
    writeOnce(recoveryPath, { schemaVersion: "aga-hybrid-f3-recovery-receipt/v4", caseName: manifest.caseName, targetFingerprintDigest: manifest.targetFingerprintDigest, transactionDigest, resultDigest: result.resultDigest, origin, appendOnlyRecoveryReceipt: origin === "recreated-after-effect", recoveryDigest: digest(`${manifest.caseName}\n${result.resultDigest}\n${origin}`) });
    process.stdout.write(`${JSON.stringify({ origin, targetReceiptDigest: result.resultDigest })}\n`);
    return;
  }
  if (mode === "cleanup") {
    if (!existsSync(transactionPath) || !existsSync(resultPath)) fail("CLEANUP_INPUT");
    if (existsSync(cleanupPath)) {
      if (!existsSync(cleanupReplayPath)) writeOnce(cleanupReplayPath, { schemaVersion: "aga-hybrid-f3-cleanup-replay-rejection/v1", caseName: manifest.caseName, targetFingerprintDigest: manifest.targetFingerprintDigest, operation, tokenHash, status: "REJECTED", reason: "NON_REPLAYABLE", cleanupDigest: readJSON(cleanupPath).cleanupDigest });
      process.exit(17);
    }
    const transactionValue = readJSON(transactionPath);
    const result = readJSON(resultPath);
    if (transactionValue.effectDigest !== effectDigest || result.targetFingerprintDigest !== manifest.targetFingerprintDigest) fail("CLEANUP_BINDING");
    writeOnce(cleanupPath, { schemaVersion: "aga-hybrid-f3-cleanup-receipt/v3", caseName: manifest.caseName, targetFingerprintDigest: manifest.targetFingerprintDigest, transactionDigest, resultDigest: result.resultDigest, cleanupDigest: digest(`${manifest.caseName}\n${result.resultDigest}\nCLEANED`) });
    process.stdout.write("CLEANUP_RECEIPT_COMMITTED\n");
    return;
  }
  fail("TARGET_MODE");
}

function prepare(values) {
  const faultRoot = resolve(values.get("root"));
  const manifest = validateManifest(readJSON(resolve(values.get("manifest"))));
  const targetsRoot = resolve(faultRoot, "targets");
  mkdirSync(targetsRoot, { recursive: true, mode: 0o700 });
  const prepared = [];
  try {
    for (const entry of manifest.cases) {
      const targetRoot = resolve(targetsRoot, entry.targetNamespace);
      if (existsSync(targetRoot)) {
        const existingManifest = readJSON(resolve(targetRoot, "target-manifest.json"));
        const existingRuntime = readJSON(resolve(targetRoot, "runtime.json"));
        const expectedManifest = targetManifest(manifest, entry);
        if (JSON.stringify(stable(existingManifest)) !== JSON.stringify(stable(expectedManifest)) || existingRuntime.projectName !== targetProjectName(entry.targetNamespace) || existingRuntime.databaseName !== "aga_f3" || Object.values(composeResidue(existingRuntime.projectName)).every((value) => value === "")) fail("TARGET_ALREADY_EXISTS");
        prepared.push(entry);
        continue;
      }
      prepared.push(entry);
      mkdirSync(targetRoot, { recursive: true, mode: 0o700 });
      mkdirSync(resolve(targetRoot, "transactions"), { recursive: true, mode: 0o700 });
      const preparedManifest = targetManifest(manifest, entry);
      writeOnce(resolve(targetRoot, "target-manifest.json"), preparedManifest);
      writeOnce(resolve(targetRoot, "namespace-seal.json"), { schemaVersion: "aga-hybrid-f3-namespace-seal/v2", caseName: entry.caseName, targetFingerprintDigest: entry.targetFingerprintDigest, targetManifestDigest: preparedManifest.targetManifestDigest, namespaceDigest: preparedManifest.targetNamespaceDigest });
      prepareCompose(targetRoot, manifest, entry);
    }
  } catch (error) {
    for (const entry of prepared) {
      const runtimePath = resolve(targetsRoot, entry.targetNamespace, "runtime.json");
      if (existsSync(runtimePath)) {
        try { compose(readJSON(runtimePath), ["down", "--volumes", "--remove-orphans"]); } catch {}
      }
    }
    throw error;
  }
  return { schemaVersion: "aga-hybrid-f3-prepared/v3", caseCount: CASES.length, targetFingerprintDigests: manifest.cases.map(({ targetFingerprintDigest }) => targetFingerprintDigest), targetProjectCount: CASES.length };
}

function cleanupPrepared(values) {
  const faultRoot = resolve(values.get("root"));
  const manifest = validateManifest(readJSON(resolve(values.get("manifest"))));
  const targetsRoot = resolve(faultRoot, "targets");
  const cases = [];
  for (const entry of manifest.cases) {
    const targetRoot = resolve(targetsRoot, entry.targetNamespace);
    const runtimePath = resolve(targetRoot, "runtime.json");
    if (!existsSync(targetRoot)) {
      cases.push({ caseName: entry.caseName, targetFingerprintDigest: entry.targetFingerprintDigest, status: "ABSENT", residueCount: 0 });
      continue;
    }
    if (!existsSync(runtimePath)) fail("CLEANUP_RUNTIME_MISSING");
    const runtime = readJSON(runtimePath);
    if (runtime.projectName !== targetProjectName(entry.targetNamespace) || runtime.databaseName !== "aga_f3") fail("CLEANUP_TARGET_BINDING");
    try { compose(runtime, ["down", "--volumes", "--remove-orphans"]); } catch (error) { throw error; }
    const residue = composeResidue(runtime.projectName);
    if (Object.values(residue).some((value) => value !== "")) fail("CLEANUP_TARGET_RESIDUE");
    rmSync(targetRoot, { recursive: true, force: true });
    if (existsSync(targetRoot)) fail("CLEANUP_TARGET_DIRECTORY_RESIDUE");
    cases.push({ caseName: entry.caseName, targetFingerprintDigest: entry.targetFingerprintDigest, status: "CLEANED", residueCount: 0 });
  }
  const output = { schemaVersion: "aga-hybrid-demo-fault-cleanup/v1", targetMode: "compose-postgres", cases, residueCount: cases.reduce((total, entry) => total + entry.residueCount, 0) };
  writeOnce(resolve(faultRoot, "cleanup-result.json"), output);
  return output;
}

function copyTargetRecords(targetRoot, destination) {
  mkdirSync(destination, { recursive: true, mode: 0o700 });
  for (const name of ["target-manifest.json", "namespace-seal.json"]) writeOnce(resolve(destination, name), readFileSync(resolve(targetRoot, name)));
  const sourceDirectory = resolve(targetRoot, "transactions");
  for (const name of readdirSync(sourceDirectory).sort()) {
    const source = resolve(sourceDirectory, name);
    if (lstatSync(source).isSymbolicLink() || !statSync(source).isFile() || !/^[A-Za-z0-9_-]+\.json$/u.test(name)) fail("TARGET_RECORD");
    writeOnce(resolve(destination, "transactions", name), readFileSync(source));
  }
}

function appendOuterJournal(path, entries, phase, targetReceiptDigest, effectDigest) {
  const previousDigest = entries.at(-1)?.receiptDigest ?? "GENESIS";
  const payload = { sequence: entries.length, phase, status: "COMPLETED", previousDigest, targetReceiptDigest, effectDigest };
  const receiptDigest = digest(JSON.stringify(stable(payload)));
  const entry = { ...payload, receiptDigest };
  const descriptor = openSync(path, "a", 0o600);
  try { writeFileSync(descriptor, `${JSON.stringify(entry)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
  const parent = openSync(dirname(path), "r");
  try { fsyncSync(parent); } finally { closeSync(parent); }
  entries.push(entry);
}

function caseJournal(caseName, targetManifestDigest, namespaceSealDigest, transactionDigest, transactionEffectDigest, resultDigest, resultBytesDigest, cleanupDigest, residueDigest) {
  const phases = ["CASE_INTENT", "TARGET_EFFECT", "TARGET_RECEIPT", "LEDGER_PUBLICATION"];
  const receipts = [
    { targetReceiptDigest: targetManifestDigest, effectDigest: namespaceSealDigest },
    { targetReceiptDigest: transactionDigest, effectDigest: transactionEffectDigest },
    { targetReceiptDigest: resultDigest, effectDigest: resultBytesDigest },
    { targetReceiptDigest: resultDigest, effectDigest: digest(`${cleanupDigest}\n${residueDigest}`) },
  ];
  let previousDigest = "GENESIS";
  return phases.map((phase, sequence) => {
    const payload = { sequence, phase, status: "COMPLETED", previousDigest, ...receipts[sequence] };
    const receiptDigest = digest(JSON.stringify(stable(payload)));
    previousDigest = receiptDigest;
    return { ...payload, receiptDigest };
  });
}

function readOuterJournal(path) {
  if (!existsSync(path)) return [];
  const text = readFileSync(path, "utf8");
  if (!text.trim()) return [];
  const entries = text.trim().split("\n").map((line) => JSON.parse(line));
  let previousDigest = "GENESIS";
  for (const [index, entry] of entries.entries()) {
    const payload = { sequence: entry.sequence, phase: entry.phase, status: entry.status, previousDigest: entry.previousDigest, targetReceiptDigest: entry.targetReceiptDigest, effectDigest: entry.effectDigest };
    if (entry.sequence !== index || entry.previousDigest !== previousDigest || entry.status !== "COMPLETED" || !DIGEST.test(entry.targetReceiptDigest ?? "") || !DIGEST.test(entry.effectDigest ?? "") || entry.receiptDigest !== digest(JSON.stringify(stable(payload)))) fail("OUTER_JOURNAL");
    previousDigest = entry.receiptDigest;
  }
  return entries;
}

function completedOperations(entries) {
  const completed = new Set();
  for (const entry of entries) {
    if (OPERATIONS.includes(entry.phase)) {
      if (completed.has(entry.phase)) fail("OUTER_JOURNAL_DUPLICATE");
      completed.add(entry.phase);
    }
  }
  return completed;
}

function caseFactsFromRaw(caseName, caseRaw) {
  const transaction = readJSON(resolve(caseRaw, "transactions/operation.json"));
  const result = readJSON(resolve(caseRaw, "transactions/result.json"));
  const recovery = readJSON(resolve(caseRaw, "transactions/recovery.json"));
  const cleanupReceipt = readJSON(resolve(caseRaw, "transactions/cleanup.json"));
  const targetManifest = readJSON(resolve(caseRaw, "target-manifest.json"));
  const targetResidue = readJSON(resolve(caseRaw, "target-residue.json"));
  const raceReceipts = caseName === "CONCURRENT_TOKEN_RESERVATION"
    ? readdirSync(resolve(caseRaw, "transactions")).filter((name) => /^authority-race-[a-z0-9-]+\.json$/u.test(name)).map((name) => readJSON(resolve(caseRaw, "transactions", name)))
    : [];
  const caseFacts = caseName === "CONCURRENT_TOKEN_RESERVATION"
    ? { sharedTokenPreReserved: false, winnerCount: raceReceipts.filter(({ status }) => status === "WON").length, loserEffectCount: Math.max(targetResidue.databaseRowsBeforeCleanup - raceReceipts.filter(({ status }) => status === "WON").length, 0) }
    : caseName === "CLEANUP_RECEIPT_GAP"
      ? { storedCleanupReceiptReplayed: readJSON(resolve(caseRaw, "transactions/cleanup-replay-rejected.json")).status !== "REJECTED", effectCount: targetResidue.databaseRowsBeforeCleanup, duplicateEffectCount: Math.max(targetResidue.databaseRowsBeforeCleanup - 1, 0) }
      : { storedReceiptReplayed: recovery.origin === "stored-target-result", missingReceiptRecreatedAfterEffect: recovery.origin === "recreated-after-effect", effectCount: targetResidue.databaseRowsBeforeCleanup, duplicateEffectCount: Math.max(targetResidue.databaseRowsBeforeCleanup - 1, 0) };
  if (targetResidue.terminalState !== "CLEANED" || targetResidue.residueCount !== 0) fail("CASE_NOT_CLEAN");
  return {
    caseName,
    targetFingerprintDigest: targetManifest.targetFingerprintDigest,
    targetNamespaceDigest: targetManifest.targetNamespaceDigest,
    targetReceiptDigest: result.resultDigest,
    caseFacts,
    journal: readFileSync(resolve(caseRaw, "journal.jsonl"), "utf8").trim().split("\n").filter(Boolean).map((line) => JSON.parse(line)),
    targetRecordDigest: digest(readFileSync(resolve(caseRaw, "transactions/operation.json"))),
    cleanupReceiptDigest: cleanupReceipt.cleanupDigest,
    terminalState: targetResidue.terminalState,
    residueCount: targetResidue.residueCount,
  };
}

function targetBinding(manifest, entry, authorizationBinding) {
  const preparedManifest = targetManifest(manifest, entry);
  return {
    "authorized-authorization-id": authorizationBinding.authorizationId,
    "authorized-authorization-digest": authorizationBinding.authorizationDigest,
    "authorized-case-name": entry.caseName,
    "authorized-target-fingerprint": entry.targetFingerprintDigest,
    "authorized-target-namespace": entry.targetNamespace,
    "authorized-target-manifest-digest": preparedManifest.targetManifestDigest,
    "authorized-input-digest": manifest.inputDigest,
    "authorized-code-digest": manifest.codeDigest,
    "authorized-contract-digest": manifest.contractDigest,
  };
}

async function runCase(faultRoot, manifest, entry, token, operation, authorizationBinding, authorityRoot) {
  const targetRoot = resolve(faultRoot, "targets", entry.targetNamespace);
  const targetManifestPath = resolve(targetRoot, "target-manifest.json");
  if (!existsSync(targetRoot) || !existsSync(targetManifestPath)) fail("TARGET_MISSING");
  const binding = targetBinding(manifest, entry, authorizationBinding);
  let runtime;
  try {
    runtime = readJSON(resolve(targetRoot, "runtime.json"));
    const first = entry.caseName === "CONCURRENT_TOKEN_RESERVATION"
      ? await awaitConcurrentApply(targetRoot, targetManifestPath, operation, token, binding, authorityRoot)
      : [runTarget(targetRoot, targetManifestPath, operation, token, "apply", binding)];
    const winnerCount = first.filter((result) => result.status === 0 || result.status === 73).length;
    const loserCount = first.filter((result) => result.status === 17).length;
    if (entry.caseName === "CONCURRENT_TOKEN_RESERVATION" && (winnerCount !== 1 || loserCount !== 1)) throw new Error(`ERR_AGA_HYBRID_F3 CONCURRENT_RESULT statuses=${first.map(({ status }) => status).join(",")} diagnostics=${first.map(({ stderr }) => (stderr ?? "").trim().slice(0, 240)).join("|")}`);
    if (entry.caseName !== "CONCURRENT_TOKEN_RESERVATION" && first[0].status !== 73) fail("INTERRUPTION_BOUNDARY");
    const recovered = runTarget(targetRoot, targetManifestPath, operation, token, "recover", binding);
    if (recovered.status !== 0) fail("RECOVERY_PROCESS");
    const cleanup = runTarget(targetRoot, targetManifestPath, operation, token, "cleanup", binding);
    if (cleanup.status !== 0) fail("CLEANUP_PROCESS");
    const cleanupReplay = entry.caseName === "CLEANUP_RECEIPT_GAP" ? runTarget(targetRoot, targetManifestPath, operation, token, "cleanup", binding) : { status: 17 };
    if (entry.caseName === "CLEANUP_RECEIPT_GAP" && cleanupReplay.status !== 17) fail("CLEANUP_REPLAY");
    const caseRaw = resolve(faultRoot, "cases", entry.caseName);
    copyTargetRecords(targetRoot, caseRaw);
    const transaction = readJSON(resolve(caseRaw, "transactions/operation.json"));
    const result = readJSON(resolve(caseRaw, "transactions/result.json"));
    const databaseRowsBeforeCleanup = Number(targetSQL(runtime, "SELECT count(*) FROM aga_f3.operation_effects;"));
    const recoveryRowsBeforeCleanup = Number(targetSQL(runtime, "SELECT count(*) FROM aga_f3.recovery_receipts;"));
    if (databaseRowsBeforeCleanup !== 1) fail("TARGET_ROW_COUNT");
    const expectedRecoveryRows = entry.caseName === "INHERITED_BASE_RECEIPT_GAP" || entry.caseName === "WORKSPACE_TRANSACTION_RECEIPT_GAP" ? 1 : 0;
    if (recoveryRowsBeforeCleanup !== expectedRecoveryRows) fail("RECOVERY_ROW_COUNT");
    const residueBeforeDown = composeResidue(runtime.projectName);
    compose(runtime, ["down", "--volumes", "--remove-orphans"]);
    const residueAfterDown = composeResidue(runtime.projectName);
    const residueCount = Object.values(residueAfterDown).filter((value) => value !== "").length;
    if (residueCount !== 0) fail("TARGET_COMPOSE_RESIDUE");
    writeOnce(resolve(caseRaw, "target-residue.json"), { schemaVersion: "aga-hybrid-f3-target-residue/v2", containers: residueAfterDown.containers === "" ? 0 : 1, volumes: residueAfterDown.volumes === "" ? 0 : 1, networks: residueAfterDown.networks === "" ? 0 : 1, databaseRowsBeforeCleanup, recoveryRowsBeforeCleanup, composeResidueBeforeDown: residueBeforeDown, composeResidueAfterDown: residueAfterDown, terminalState: "CLEANED", residueCount });
    const cleanupReceipt = readJSON(resolve(caseRaw, "transactions/cleanup.json"));
    const journal = caseJournal(entry.caseName, digest(readFileSync(resolve(caseRaw, "target-manifest.json"))), digest(readFileSync(resolve(caseRaw, "namespace-seal.json"))), transactionDigest(transaction), transaction.effectDigest, result.resultDigest, digest(readFileSync(resolve(caseRaw, "transactions/result.json"))), cleanupReceipt.cleanupDigest, digest(readFileSync(resolve(caseRaw, "target-residue.json"))));
    writeOnce(resolve(caseRaw, "journal.jsonl"), `${journal.map((record) => JSON.stringify(record)).join("\n")}\n`);
    rmSync(targetRoot, { recursive: true, force: true });
    if (existsSync(targetRoot)) fail("TARGET_RESIDUE");
    return caseFactsFromRaw(entry.caseName, caseRaw);
  } catch (error) {
    if (runtime && existsSync(runtime.composeFile)) {
      try { compose(runtime, ["down", "--volumes", "--remove-orphans"]); } catch {}
    }
    rmSync(targetRoot, { recursive: true, force: true });
    throw error;
  }
}

function transactionDigest(transaction) {
  return digest(JSON.stringify(stable(transaction)));
}

function awaitConcurrentApply(targetRoot, targetManifestPath, operation, token, binding, authorityRoot) {
  return Promise.all([0, 1].map((attempt) => new Promise((resolvePromise, reject) => {
    const child = spawn(process.execPath, targetArgs("apply", targetRoot, targetManifestPath, operation, token, { ...binding, "authority-root": authorityRoot, "authority-attempt": `attempt-${attempt}` }), { stdio: ["ignore", "pipe", "pipe"] });
    let stdout = "";
    let stderr = "";
    child.stdout.on("data", (value) => { stdout += value; });
    child.stderr.on("data", (value) => { stderr += value; });
    child.on("error", reject);
    child.on("exit", (status) => resolvePromise({ status, stdout, stderr }));
  })));
}

async function run(values) {
  const faultRoot = resolve(values.get("root"));
  const manifest = validateManifest(readJSON(resolve(values.get("manifest"))));
  const authorizationPath = resolve(values.get("authorization"));
  const authorization = validateAuthorization(readJSON(authorizationPath), manifest);
  const originalAuthorization = authorization.phase === "fault-matrix-run"
    ? authorization
    : validateAuthorization(readJSON(resolve(values.get("run-authorization"))), manifest);
  if (authorization.phase === "fault-matrix-recovery" && originalAuthorization.phase !== "fault-matrix-run") fail("RUN_AUTH_PHASE");
  const journalPath = resolve(faultRoot, "outer-journal.jsonl");
  const entries = readOuterJournal(journalPath);
  if (authorization.phase === "fault-matrix-run" && entries.length !== 0) fail("RUN_MUST_START_WITH_EMPTY_OUTER_JOURNAL");
  if (authorization.phase === "fault-matrix-recovery") {
    if (authorization.journalDigest !== digest(readFileSync(journalPath))) fail("RECOVERY_JOURNAL_BINDING");
    if (authorization.outerIntentDigest !== outerIntentDigest(manifest)) fail("RECOVERY_INTENT_BINDING");
  }
  const originalAuthorizationPath = authorization.phase === "fault-matrix-run" ? authorizationPath : resolve(values.get("run-authorization"));
  const authorizationBinding = { authorizationId: originalAuthorization.authorizationId, authorizationDigest: digest(readFileSync(originalAuthorizationPath)) };
  const authorityRoot = resolve(values.get("authority-root") ?? "");
  if (!authorityRoot.startsWith("/") || !existsSync(authorityRoot) || lstatSync(authorityRoot).isSymbolicLink() || (statSync(authorityRoot).mode & 0o777) !== 0o700) fail("AUTHORITY_ROOT");
  const completed = completedOperations(entries);
  const executionByCase = new Map();
  for (const entry of manifest.cases) if (completed.has(`RUN_${entry.caseName}`)) executionByCase.set(entry.caseName, caseFactsFromRaw(entry.caseName, resolve(faultRoot, "cases", entry.caseName)));
  const stopAfter = authorization.phase === "fault-matrix-run" ? process.env.AVIA_AGA_HYBRID_F3_STOP_AFTER_CASE : "";
  for (const caseName of EXECUTION_ORDER) {
    const entry = manifest.cases.find((candidate) => candidate.caseName === caseName);
    const operationIndex = manifest.cases.findIndex((candidate) => candidate.caseName === caseName);
    const operation = OPERATIONS[operationIndex];
    if (completed.has(operation)) continue;
    const caseRaw = resolve(faultRoot, "cases", entry.caseName);
    const caseResult = authorization.phase === "fault-matrix-recovery" && existsSync(resolve(caseRaw, "journal.jsonl"))
      ? caseFactsFromRaw(entry.caseName, caseRaw)
      : await runCase(faultRoot, manifest, entry, originalAuthorization.tokens[operationIndex], operation, authorizationBinding, authorityRoot);
    const result = readJSON(resolve(faultRoot, "cases", entry.caseName, "transactions", "result.json"));
    const transaction = readJSON(resolve(faultRoot, "cases", entry.caseName, "transactions", "operation.json"));
    executionByCase.set(entry.caseName, caseResult);
    if (stopAfter && stopAfter === entry.caseName) throw new F3Interrupted(entry.caseName);
    appendOuterJournal(journalPath, entries, operation, result.resultDigest, transaction.effectDigest);
  }
  if (executionByCase.size !== CASES.length) fail("EXECUTION_INCOMPLETE");
  const execution = manifest.cases.map((entry) => executionByCase.get(entry.caseName));
  const authorityFiles = [];
  const collectAuthorityFiles = (directory, prefix = "") => {
    for (const name of readdirSync(directory).sort()) {
      const path = resolve(directory, name);
      if (lstatSync(path).isSymbolicLink()) fail("AUTHORITY_RESIDUE");
      if (statSync(path).isDirectory()) collectAuthorityFiles(path, `${prefix}${name}/`);
      else authorityFiles.push({ name: `${prefix}${name}`, digest: digest(readFileSync(path)) });
    }
  };
  collectAuthorityFiles(authorityRoot);
  const authorityConsumptionDigest = digest(JSON.stringify(stable(authorityFiles)));
  const output = { schemaVersion: "aga-hybrid-demo-fault-execution/v4", targetMode: "compose-postgres", cases: execution, outerJournalDigest: digest(readFileSync(journalPath)), authorityConsumptionDigest, authorityConsumptionFileCount: authorityFiles.length, caseCount: execution.length };
  writeOnce(resolve(faultRoot, "execution.json"), output);
  return output;
}

try {
  const values = parseArgs(process.argv.slice(2));
  const targetOperationMode = values.get("target-mode");
  const mode = values.get("mode") ?? (["apply", "recover", "cleanup"].includes(targetOperationMode ?? "") ? "target" : targetOperationMode);
  if (mode === "target") targetProcess(values);
  else if (mode === "prepare") process.stdout.write(`${JSON.stringify(prepare(values))}\n`);
  else if (mode === "cleanup-prepared") process.stdout.write(`${JSON.stringify(cleanupPrepared(values))}\n`);
  else if (mode === "run") process.stdout.write(`${JSON.stringify(await run(values))}\n`);
  else fail("MODE");
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : "ERR_AGA_HYBRID_F3 UNEXPECTED"}\n`);
  process.exitCode = error?.exitCode ?? 1;
}
