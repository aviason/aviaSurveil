#!/usr/bin/env node

// Operator-side F3 manifest preparation. It creates target identities only;
// token issuance remains a separate operation in issue-aga-hybrid-connected-
// authorization.mjs.
import { createHash } from "node:crypto";
import { closeSync, fsyncSync, mkdirSync, openSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";

const CASES = ["INHERITED_BASE_RECEIPT_GAP", "WORKSPACE_TRANSACTION_RECEIPT_GAP", "CONCURRENT_TOKEN_RESERVATION", "CLEANUP_RECEIPT_GAP"];
const digest = (value) => `sha256:${createHash("sha256").update(value).digest("hex")}`;
const stable = (value) => Array.isArray(value) ? value.map(stable) : value && typeof value === "object" ? Object.fromEntries(Object.keys(value).sort().map((key) => [key, stable(value[key])])) : value;
const values = new Map();
for (let index = 0; index < process.argv.slice(2).length; index += 2) {
  const key = process.argv[index + 2];
  const value = process.argv[index + 3];
  if (!key?.startsWith("--") || value === undefined || values.has(key)) throw new Error("ERR_AGA_HYBRID_F3_MANIFEST_ARGUMENTS");
  values.set(key, value);
}
try {
  const output = resolve(values.get("--output") ?? "");
  const inputDigest = values.get("--input-digest");
  const codeDigest = values.get("--code-digest");
  const contractDigest = values.get("--contract-digest");
  if (!output.startsWith("/") || !/^sha256:[a-f0-9]{64}$/u.test(inputDigest ?? "") || !/^sha256:[a-f0-9]{64}$/u.test(codeDigest ?? "") || !/^sha256:[a-f0-9]{64}$/u.test(contractDigest ?? "")) throw new Error("ERR_AGA_HYBRID_F3_MANIFEST_INPUT");
  const cases = CASES.map((caseName) => {
    const targetNamespace = `aga-f3-${caseName.toLowerCase().replaceAll("_", "-")}`;
    const targetFingerprintDigest = digest(`AGA-F3-TARGET-V3\n${caseName}\n${targetNamespace}\n${inputDigest}\n${codeDigest}\n${contractDigest}`);
    const targetBody = { schemaVersion: "aga-hybrid-f3-target-manifest/v3", caseName, targetFingerprintDigest, targetNamespace, targetProjectName: `aviasurveil360-aga-f3-${targetNamespace}`, targetDatabaseName: "aga_f3", targetNamespaceDigest: digest(`AGA-F3-NAMESPACE-V3\n${targetNamespace}`), inputDigest, codeDigest, contractDigest };
    return { caseName, targetFingerprintDigest, targetNamespace, targetManifestDigest: digest(JSON.stringify(stable(targetBody))) };
  });
  const body = { schemaVersion: "aga-hybrid-demo-fault-matrix-manifest/v2", caseNames: CASES, cases, inputDigest, codeDigest, contractDigest };
  const value = { ...body, manifestDigest: digest(JSON.stringify(stable(body))) };
  mkdirSync(dirname(output), { recursive: true, mode: 0o700 });
  const descriptor = openSync(output, "wx", 0o600);
  try { writeFileSync(descriptor, `${JSON.stringify(value, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
  const parent = openSync(dirname(output), "r");
  try { fsyncSync(parent); } finally { closeSync(parent); }
  process.stdout.write(`aga-hybrid-f3-manifest: wrote digest=${value.manifestDigest}\n`);
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : "ERR_AGA_HYBRID_F3_MANIFEST_UNEXPECTED"}\n`);
  process.exitCode = 1;
}
