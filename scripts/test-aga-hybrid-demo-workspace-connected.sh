#!/usr/bin/env bash
set -euo pipefail

# This harness is the connected writer. It does not accept an aggregate
# receipt fixture. Prepare creates the disposable target and records target
# receipts as effects happen; qualify runs the target-bound continuation and
# only then builds the privacy-safe ledger from those receipts. Fault mode
# runs the same receipt/journal primitives against real private filesystem
# effects so recovery and concurrent reservation are executable tests.

umask 077
repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mode="${1:-}"
private_root="${AVIA_AGA_HYBRID_PRIVATE_ROOT:-}"
ledger_dir="${AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR:-}"
prepare_auth="${AVIA_AGA_HYBRID_PREPARE_AUTHORIZATION_FILE:-}"
qualification_auth="${AVIA_AGA_HYBRID_QUALIFICATION_AUTHORIZATION_BUNDLE_FILE:-}"
recovery_auth="${AVIA_AGA_HYBRID_RECOVERY_AUTHORIZATION_FILE:-}"
cleanup_auth="${AVIA_AGA_HYBRID_CLEANUP_AUTHORIZATION_FILE:-}"
fault_root="${AVIA_AGA_HYBRID_FAULT_AUTHORIZATION_ROOT:-}"
fault_prepare_manifest="${AVIA_AGA_HYBRID_FAULT_PREPARE_MANIFEST:-}"
fault_run_manifest="${AVIA_AGA_HYBRID_FAULT_RUN_MANIFEST:-}"
fault_recovery_manifest="${AVIA_AGA_HYBRID_FAULT_RECOVERY_MANIFEST:-}"
fault_prepare_auth="${AVIA_AGA_HYBRID_FAULT_PREPARE_AUTHORIZATION_FILE:-}"
fault_run_auth="${AVIA_AGA_HYBRID_FAULT_RUN_AUTHORIZATION_FILE:-}"
fault_recovery_auth="${AVIA_AGA_HYBRID_FAULT_RECOVERY_AUTHORIZATION_FILE:-}"
fault_cleanup_auth="${AVIA_AGA_HYBRID_FAULT_CLEANUP_AUTHORIZATION_FILE:-${AVIA_AGA_HYBRID_FAULT_RECOVERY_AUTHORIZATION_FILE:-}}"
state_directory=""
runtime_root=""
handoff_file=""
target_fingerprint_digest=""
run_id=""
overlay_run_id=""
overlay_target_fingerprint_digest=""
aga_demo_config=""
aga_intent_digest=""
phase_receipt_dir=""
target_receipt_dir=""
journal_file=""
web_pid=""

fail() {
  printf 'ERR_AGA_HYBRID_CONNECTED_%s\n' "$1" >&2
  exit 1
}

require_mode() {
  case "$mode" in
    prepare|recover-status|recover-prepare|qualify|serve|recover-qualify|cleanup-prepared|fault-matrix-prepare|fault-matrix-recover-status|fault-matrix-recover-prepare|fault-matrix-run|fault-matrix-recover-run|fault-matrix-cleanup-prepared|fault-matrix-cleanup-partial) ;;
    *) fail MODE_INVALID ;;
  esac
}

absolute_path() { [[ -n "$1" && "$1" = /* ]] || fail PATH_NOT_ABSOLUTE; }

private_directory() {
  local path="$1"
  absolute_path "$path"
  [[ -d "$path" && ! -L "$path" ]] || fail PRIVATE_DIRECTORY_INVALID
  [[ "$(stat -f '%Lp' "$path")" == "700" ]] || fail PRIVATE_DIRECTORY_MODE
}

private_file() {
  local path="$1"
  absolute_path "$path"
  [[ -f "$path" && ! -L "$path" ]] || fail PRIVATE_FILE_MISSING
  [[ "$(stat -f '%Lp' "$path")" == "600" ]] || fail PRIVATE_FILE_MODE
}

make_private_directory() {
  local path="$1"
  mkdir -p "$path"
  chmod 700 "$path"
  private_directory "$path"
}

compose_command() {
  [[ -n "$state_directory" ]] || fail TARGET_STATE_MISSING
  AVIA_PREPROD_STATE_DIR="$state_directory" docker compose \
    --project-name aviasurveil360-local-preprod \
    --file "$repository_root/deploy/local/compose.yaml" \
    --profile local-preprod-loader \
    --profile aga-candidate-demo \
    --profile aga-candidate-demo-oidc-fixture \
    --profile aga-demo-workspace-loader \
    --profile preproddemo \
    "$@"
}

project_residue() {
  local containers volumes networks
  containers="$(docker ps --all --filter 'label=com.docker.compose.project=aviasurveil360-local-preprod' --format '{{.ID}}')"
  volumes="$(docker volume ls --filter 'label=com.docker.compose.project=aviasurveil360-local-preprod' --format '{{.Name}}')"
  networks="$(docker network ls --filter 'label=com.docker.compose.project=aviasurveil360-local-preprod' --format '{{.Name}}')"
  [[ -n "$containers$volumes$networks" ]]
}

cleanup_target() {
  if [[ -n "$web_pid" ]] && kill -0 "$web_pid" 2>/dev/null; then
    kill "$web_pid" 2>/dev/null || true
    wait "$web_pid" 2>/dev/null || true
    web_pid=""
  fi
  if [[ -n "$state_directory" ]] && command -v docker >/dev/null 2>&1 && project_residue; then
    compose_command down --volumes --remove-orphans >/dev/null 2>&1 || true
  fi
  if project_residue 2>/dev/null; then
    printf 'ERR_AGA_HYBRID_CONNECTED_COMPOSE_RESIDUE\n' >&2
    return 1
  fi
}

cleanup_fault_targets() {
  [[ -n "$fault_root" && -d "$fault_root/targets" ]] || return 0
  while IFS= read -r runtime_file; do
    [[ -n "$runtime_file" ]] || continue
    local project compose_file
    project="$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).projectName' "$runtime_file" 2>/dev/null || true)"
    compose_file="$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).composeFile' "$runtime_file" 2>/dev/null || true)"
    if [[ "$project" =~ ^aviasurveil360-aga-f3-[a-z0-9-]+$ && "$compose_file" = "$fault_root/targets/"* ]]; then
      docker compose --project-name "$project" --file "$compose_file" down --volumes --remove-orphans >/dev/null 2>&1 || true
    fi
  done < <(find "$fault_root/targets" -type f -name runtime.json -print)
}

cleanup_on_exit() {
  local status=$?
  trap - EXIT HUP INT TERM
  local authority_stop=false
  if [[ "$status" -eq 2 && ( "$mode" == "prepare" || "$mode" == "recover-prepare" || "$mode" == "fault-matrix-prepare" || "$mode" == "fault-matrix-recover-prepare" ) ]]; then
    authority_stop=true
  fi
  if [[ "$status" -ne 0 && "$authority_stop" != true ]]; then cleanup_target || status=1; cleanup_fault_targets || status=1; fi
  exit "$status"
}
trap cleanup_on_exit EXIT HUP INT TERM

validate_document() {
  local path="$1"
  private_file "$path"
  node --input-type=module - "$path" "$repository_root" <<'NODE'
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const [documentPath, repositoryRoot] = process.argv.slice(2);
const value = JSON.parse(readFileSync(documentPath, "utf8"));
const digest = /^sha256:[a-f0-9]{64}$/u;
const phases = {
  prepare: ["CREATE_BASE", "QUALIFY_EXISTING_SYNTHETIC_OIDC", "PIN_PRE_WORKSPACE_FORBIDDEN_BASELINE", "PROVISION_EMPTY_WORKSPACE_CONTRACT", "EXPORT_WORKSPACE_FIXTURE", "PREPARE_LOAD_INTENTS", "CLEANUP_ON_PREPARE_FAILURE"],
  qualification: ["LOAD_AGA_CANDIDATE_DEMO_OVERLAY", "RUN_WORKSPACE_LOAD_SEAL_BARRIERS", "LOAD_AND_SEAL_AGA_DEMO_WORKSPACE", "RUN_CONNECTED_AGA_HYBRID_QUALIFICATION", "CLEANUP_AGA_CANDIDATE_DEMO_OVERLAY", "CLEANUP_WHOLE_DISPOSABLE_NAMESPACE"],
  recovery: ["RESUME_PREPARE", "RESUME_QUALIFICATION", "CLEANUP_WHOLE_DISPOSABLE_NAMESPACE"],
  cleanup: ["CLEANUP_WHOLE_DISPOSABLE_NAMESPACE"],
  "fault-matrix-prepare": ["PREPARE_FAULT_MATRIX", "CLEANUP_FAULT_MATRIX_ON_FAILURE"],
  "fault-matrix-run": ["RUN_INHERITED_BASE_RECEIPT_GAP", "RUN_WORKSPACE_TRANSACTION_RECEIPT_GAP", "RUN_CONCURRENT_TOKEN_RESERVATION", "RUN_CLEANUP_RECEIPT_GAP"],
  "fault-matrix-recovery": ["RESUME_FAULT_MATRIX", "CLEANUP_FAULT_MATRIX"],
};
const keys = ["schemaVersion", "authorizationId", "issuer", "nonce", "issuedAt", "expiresAt", "inputDigest", "codeDigest", "contractDigest", "phase", "targetMode", "targetFingerprintDigest", "intentDigest", "journalDigest", "bundleDigest", "outerIntentDigest", "caseTargetFingerprints", "operations", "tokens", "operationTokenHashes"];
if (value.schemaVersion !== "aga-hybrid-demo-connected-authorization/v1" || !/^aga-auth-[a-f0-9]{32}$/u.test(value.authorizationId) || !/^[a-f0-9]{32,}$/u.test(value.nonce)) throw new Error("ERR_AGA_HYBRID_CONNECTED_AUTH_HEADER");
if (typeof value.issuer !== "string" || value.issuer.length < 1 || value.issuer.length > 128) throw new Error("ERR_AGA_HYBRID_CONNECTED_AUTH_ISSUER");
for (const name of ["inputDigest", "codeDigest", "contractDigest"]) if (!digest.test(value[name] ?? "")) throw new Error("ERR_AGA_HYBRID_CONNECTED_AUTH_DIGEST");
const issued = Date.parse(value.issuedAt);
const expires = Date.parse(value.expiresAt);
if (!Number.isFinite(issued) || !Number.isFinite(expires) || expires <= issued || expires - issued > 15 * 60 * 1000 || expires <= Date.now()) throw new Error("ERR_AGA_HYBRID_CONNECTED_AUTH_EXPIRY");
if (!phases[value.phase] || Object.keys(value).some((key) => !keys.includes(key))) throw new Error("ERR_AGA_HYBRID_CONNECTED_AUTH_DOCUMENT_CLOSED");
if (!Array.isArray(value.operations) || JSON.stringify(value.operations) !== JSON.stringify(phases[value.phase])) throw new Error("ERR_AGA_HYBRID_CONNECTED_AUTH_OPERATIONS");
if (!Array.isArray(value.tokens) || value.tokens.length !== value.operations.length || new Set(value.tokens).size !== value.tokens.length || value.tokens.some((token) => !/^[a-f0-9]{64}$/u.test(token))) throw new Error("ERR_AGA_HYBRID_CONNECTED_AUTH_TOKENS");
if (!value.operationTokenHashes || JSON.stringify(Object.keys(value.operationTokenHashes).sort()) !== JSON.stringify(value.operations.slice().sort())) throw new Error("ERR_AGA_HYBRID_CONNECTED_AUTH_TOKEN_MAP");
for (const [index, operation] of value.operations.entries()) if (value.operationTokenHashes[operation] !== `sha256:${createHash("sha256").update(value.tokens[index]).digest("hex")}`) throw new Error("ERR_AGA_HYBRID_CONNECTED_AUTH_TOKEN_MAP");
if (!["CREATE_FRESH_DISPOSABLE", "EXACT_TARGET"].includes(value.targetMode)) throw new Error("ERR_AGA_HYBRID_CONNECTED_AUTH_TARGET_MODE");
if (value.targetMode === "CREATE_FRESH_DISPOSABLE" && (Object.hasOwn(value, "targetFingerprintDigest") || Object.hasOwn(value, "intentDigest"))) throw new Error("ERR_AGA_HYBRID_CONNECTED_AUTH_CREATE_TARGET_BINDING");
if (value.targetMode === "EXACT_TARGET" && (!digest.test(value.targetFingerprintDigest ?? "") || !digest.test(value.intentDigest ?? ""))) throw new Error("ERR_AGA_HYBRID_CONNECTED_AUTH_TARGET_BINDING");
for (const field of ["journalDigest", "bundleDigest", "outerIntentDigest"]) if (Object.hasOwn(value, field) && !digest.test(value[field])) throw new Error("ERR_AGA_HYBRID_CONNECTED_AUTH_DIGEST");
if (Object.hasOwn(value, "caseTargetFingerprints") && (!Array.isArray(value.caseTargetFingerprints) || new Set(value.caseTargetFingerprints).size !== value.caseTargetFingerprints.length || value.caseTargetFingerprints.some((entry) => !digest.test(entry)))) throw new Error("ERR_AGA_HYBRID_CONNECTED_AUTH_CASE_TARGETS");
if (resolve(repositoryRoot, documentPath) !== documentPath) throw new Error("ERR_AGA_HYBRID_CONNECTED_AUTH_PATH");
NODE
}

consume_authorization() {
  local path="$1"
  node --input-type=module - "$path" "$ledger_dir" <<'NODE'
import { createHash } from "node:crypto";
import { closeSync, fsyncSync, mkdirSync, openSync, readFileSync, unlinkSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";

const [documentPath, ledgerDirectory] = process.argv.slice(2);
const value = JSON.parse(readFileSync(documentPath, "utf8"));
const bytes = readFileSync(documentPath);
const digest = `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
const nonceDigest = `sha256:${createHash("sha256").update(value.nonce).digest("hex")}`;
const skippedOperations = new Set((process.env.AVIA_AGA_HYBRID_SKIP_AUTHORIZATION_OPERATIONS ?? "").split(",").filter(Boolean));
const selectedOperations = value.operations.filter((operation) => !skippedOperations.has(operation));
const tokenHashes = value.tokens.filter((_, index) => !skippedOperations.has(value.operations[index])).map((token) => `sha256:${createHash("sha256").update(token).digest("hex")}`);
const root = resolve(ledgerDirectory, "authority-consumption");
const directories = [root, resolve(root, "authorization-ids"), resolve(root, "nonce-digests"), resolve(root, "token-digests")];
directories.forEach((directory) => mkdirSync(directory, { recursive: true, mode: 0o700 }));
const reservationDigest = `sha256:${createHash("sha256").update(JSON.stringify({ authorizationId: value.authorizationId, nonceDigest, selectedOperations, skippedOperations: [...skippedOperations].sort(), tokenHashes })).digest("hex")}`;
const reservationPath = resolve(root, "reservations", `${reservationDigest.replace("sha256:", "")}.json`);
mkdirSync(dirname(reservationPath), { recursive: true, mode: 0o700 });
const created = [];
const writeCreateOnly = (path, payload) => {
  const descriptor = openSync(path, "wx", 0o600);
  try { writeFileSync(descriptor, `${JSON.stringify(payload)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
  created.push(path);
  const parent = openSync(dirname(path), "r");
  try { fsyncSync(parent); } finally { closeSync(parent); }
};
try {
  writeCreateOnly(reservationPath, { schemaVersion: "aga-hybrid-demo-authority-reservation/v2", authorizationDigest: digest, phase: value.phase, authorizationIdDigest: `sha256:${createHash("sha256").update(value.authorizationId).digest("hex")}`, nonceDigest, selectedOperations, skippedOperations: [...skippedOperations].sort(), tokenHashes });
  writeCreateOnly(resolve(root, "authorization-ids", `${value.authorizationId}.json`), { schemaVersion: "aga-hybrid-demo-authority-id/v1", reservationDigest, authorizationDigest: digest });
  writeCreateOnly(resolve(root, "nonce-digests", `${nonceDigest.replace("sha256:", "")}.json`), { schemaVersion: "aga-hybrid-demo-authority-nonce/v1", reservationDigest, authorizationDigest: digest });
  for (const tokenHash of tokenHashes) writeCreateOnly(resolve(root, "token-digests", `${tokenHash.replace("sha256:", "")}.json`), { schemaVersion: "aga-hybrid-demo-authority-token/v1", reservationDigest, authorizationDigest: digest });
} catch (error) {
  for (const path of created) try { unlinkSync(path); } catch {}
  throw new Error(`ERR_AGA_HYBRID_CONNECTED_AUTH_CONSUMPTION ${error.code ?? "FAILED"}`);
}
process.stdout.write(`${digest}\n`);
NODE
}

append_journal() {
  local phase="$1" status="$2" target_receipt_digest="$3" effect_digest="$4"
  node --input-type=module - "$journal_file" "$phase" "$status" "$target_receipt_digest" "$effect_digest" <<'NODE'
import { createHash } from "node:crypto";
import { closeSync, fsyncSync, mkdirSync, openSync, readFileSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";

const [journalPath, phase, status, targetReceiptDigest, effectDigest] = process.argv.slice(2);
mkdirSync(dirname(journalPath), { recursive: true, mode: 0o700 });
let previousDigest = "GENESIS";
let sequence = 0;
if (readFileSync(journalPath, { encoding: "utf8", flag: "a+" }).trim()) {
  const entries = readFileSync(journalPath, "utf8").trim().split("\n").filter(Boolean).map((line) => JSON.parse(line));
  const last = entries.at(-1);
  previousDigest = last.receiptDigest;
  sequence = entries.length;
}
const payload = { sequence, phase, status, previousDigest, targetReceiptDigest, effectDigest };
const receiptDigest = `sha256:${createHash("sha256").update(JSON.stringify(Object.fromEntries(Object.keys(payload).sort().map((key) => [key, payload[key]])))).digest("hex")}`;
const descriptor = openSync(journalPath, "a", 0o600);
try { writeFileSync(descriptor, `${JSON.stringify({ ...payload, receiptDigest })}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
const parent = openSync(dirname(journalPath), "r");
try { fsyncSync(parent); } finally { closeSync(parent); }
process.stdout.write(`${receiptDigest}\n`);
NODE
}

record_phase() {
  local phase="$1" operation="$2" output_path="$3" facts_json="$4"
  private_file "$output_path"
  local receipt_output
  receipt_output="$({
    node --input-type=module - "$target_receipt_dir" "$phase_receipt_dir" "$phase" "$operation" "$target_fingerprint_digest" "$output_path" "$facts_json" <<'NODE'
import { createHash } from "node:crypto";
import { closeSync, fsyncSync, mkdirSync, openSync, readFileSync, writeFileSync } from "node:fs";
import { basename, dirname, resolve } from "node:path";

const [targetDir, phaseDir, phase, operation, targetFingerprintDigest, outputPath, factsSource] = process.argv.slice(2);
const stable = (value) => Array.isArray(value) ? value.map(stable) : value && typeof value === "object" ? Object.fromEntries(Object.keys(value).sort().map((key) => [key, stable(value[key])])) : value;
const hash = (value) => `sha256:${createHash("sha256").update(value).digest("hex")}`;
const effectDigest = hash(readFileSync(outputPath));
const facts = JSON.parse(factsSource);
const factsDigest = hash(JSON.stringify(stable(facts)));
const existing = readFileSync(resolve(phaseDir, "phase-inputs.jsonl"), { encoding: "utf8", flag: "a+" }).trim();
const sequence = existing ? existing.split("\n").filter(Boolean).length : 0;
const targetReceipt = { schemaVersion: "aga-hybrid-demo-target-receipt/v1", phase, operation, targetFingerprintDigest, effectDigest, factsDigest, sequence };
targetReceipt.receiptDigest = hash(JSON.stringify(stable(targetReceipt)));
const targetPath = resolve(targetDir, `${String(sequence).padStart(2, "0")}-${phase}.json`);
const write = (path, value) => { const descriptor = openSync(path, "wx", 0o600); try { writeFileSync(descriptor, `${JSON.stringify(value, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); } const parent = openSync(dirname(path), "r"); try { fsyncSync(parent); } finally { closeSync(parent); } };
mkdirSync(targetDir, { recursive: true, mode: 0o700 });
mkdirSync(phaseDir, { recursive: true, mode: 0o700 });
write(targetPath, targetReceipt);
const phaseInput = { phase, targetReceiptDigest: targetReceipt.receiptDigest, effectDigest, factsDigest };
const descriptor = openSync(resolve(phaseDir, "phase-inputs.jsonl"), "a", 0o600);
try { writeFileSync(descriptor, `${JSON.stringify(phaseInput)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
const parent = openSync(phaseDir, "r"); try { fsyncSync(parent); } finally { closeSync(parent); }
process.stdout.write(`${targetReceipt.receiptDigest}|${effectDigest}\n`);
NODE
  } )"
  local target_receipt_digest="${receipt_output%%|*}"
  local effect_digest="${receipt_output##*|}"
  append_journal "$phase" COMPLETED "$target_receipt_digest" "$effect_digest" >/dev/null
}

validate_connected_receipts() {
  private_directory "$phase_receipt_dir"
  private_directory "$target_receipt_dir"
  private_file "$journal_file"
  node --input-type=module - "$phase_receipt_dir" "$target_receipt_dir" "$journal_file" "$target_fingerprint_digest" <<'NODE'
import { createHash } from "node:crypto";
import { readdirSync, readFileSync, statSync } from "node:fs";
const [phaseDir, targetDir, journalPath, targetFingerprintDigest] = process.argv.slice(2);
const phases = ["TARGET_CREATED", "OIDC_QUALIFIED", "FORBIDDEN_BASELINE_PINNED", "WORKSPACE_CONTRACT_PROVISIONED", "FIXTURE_EXPORTED", "INTENTS_SEALED", "OVERLAY_SEALED", "LOAD_SEAL_BARRIERS_COMPLETE", "WORKSPACE_SEALED", "CREDENTIALS_REVOKED", "API_STARTED", "AUTH_VERIFIED", "E2E_COMPLETE", "CLEANED"];
const digest = (value) => `sha256:${createHash("sha256").update(value).digest("hex")}`;
const stable = (value) => Array.isArray(value) ? value.map(stable) : value && typeof value === "object" ? Object.fromEntries(Object.keys(value).sort().map((key) => [key, stable(value[key])])) : value;
const inputs = readFileSync(`${phaseDir}/phase-inputs.jsonl`, "utf8").trim().split("\n").filter(Boolean).map((line) => JSON.parse(line));
if (JSON.stringify(inputs.map(({ phase }) => phase)) !== JSON.stringify(phases)) throw new Error("ERR_AGA_HYBRID_CONNECTED_PHASE_RECEIPTS");
if (!/^sha256:[a-f0-9]{64}$/u.test(targetFingerprintDigest) || readdirSync(targetDir).length !== phases.length) throw new Error("ERR_AGA_HYBRID_CONNECTED_TARGET_RECEIPTS");
for (const [index, input] of inputs.entries()) {
  const targetPath = `${targetDir}/${String(index).padStart(2, "0")}-${input.phase}.json`;
  const target = JSON.parse(readFileSync(targetPath, "utf8"));
  const targetBody = { schemaVersion: target.schemaVersion, phase: target.phase, operation: target.operation, targetFingerprintDigest: target.targetFingerprintDigest, effectDigest: target.effectDigest, factsDigest: target.factsDigest, sequence: target.sequence };
  if (JSON.stringify(Object.keys(target).sort()) !== JSON.stringify(["effectDigest", "factsDigest", "operation", "phase", "receiptDigest", "schemaVersion", "sequence", "targetFingerprintDigest"].sort()) || target.schemaVersion !== "aga-hybrid-demo-target-receipt/v1" || target.phase !== input.phase || target.sequence !== index || target.targetFingerprintDigest !== targetFingerprintDigest || input.targetReceiptDigest !== target.receiptDigest || input.effectDigest !== target.effectDigest || input.factsDigest !== target.factsDigest || target.receiptDigest !== digest(JSON.stringify(stable(targetBody)))) throw new Error("ERR_AGA_HYBRID_CONNECTED_TARGET_RECEIPT_BINDING");
}
const journal = readFileSync(journalPath, "utf8").trim().split("\n").filter(Boolean).map((line) => JSON.parse(line));
let previousDigest = "GENESIS";
for (const [index, entry] of journal.entries()) {
  const expected = { sequence: entry.sequence, phase: entry.phase, status: entry.status, previousDigest: entry.previousDigest, targetReceiptDigest: entry.targetReceiptDigest, effectDigest: entry.effectDigest };
  if (entry.phase !== phases[index] || entry.sequence !== index || entry.status !== "COMPLETED" || entry.previousDigest !== previousDigest || entry.targetReceiptDigest !== inputs[index].targetReceiptDigest || entry.effectDigest !== inputs[index].effectDigest || entry.receiptDigest !== digest(JSON.stringify(stable(expected)))) throw new Error("ERR_AGA_HYBRID_CONNECTED_JOURNAL");
  previousDigest = entry.receiptDigest;
}
if (journal.length !== phases.length) throw new Error("ERR_AGA_HYBRID_CONNECTED_JOURNAL");
for (const path of [...readdirSync(phaseDir).map((name) => `${phaseDir}/${name}`), ...readdirSync(targetDir).map((name) => `${targetDir}/${name}`), journalPath]) if ((statSync(path).mode & 0o777) !== 0o600) throw new Error("ERR_AGA_HYBRID_CONNECTED_PRIVATE_MODE");
NODE
}

validate_inventory() {
  node "$repository_root/scripts/build-aga-hybrid-forbidden-inventory.mjs" --check "$repository_root/tests/fixtures/aga-hybrid-forbidden-object-inventory.v1.json" >/dev/null || fail INVENTORY_INVALID
}

validate_compose_static() {
  command -v docker >/dev/null 2>&1 || fail DOCKER_REQUIRED
  docker compose --project-name aviasurveil360-local-preprod --file "$repository_root/deploy/local/compose.yaml" --profile local-preprod-loader --profile aga-candidate-demo --profile aga-candidate-demo-oidc-fixture --profile aga-demo-workspace-loader --profile preproddemo config >/dev/null || fail COMPOSE_CONTRACT
}

require_inputs_for_prepare() {
  for variable in AVIA_AGA_DEMO_PACKAGE_FILE AVIA_AGA_CLASSIFICATION_CANDIDATE_DIR AVIA_AGA_PROVIDER_CATALOG_FILE; do absolute_path "${!variable:-}"; done
  [[ -f "$AVIA_AGA_DEMO_PACKAGE_FILE" && -d "$AVIA_AGA_CLASSIFICATION_CANDIDATE_DIR" && -f "$AVIA_AGA_PROVIDER_CATALOG_FILE" ]] || fail INPUT_MISSING
}

run_f1_probe() {
  local output="$private_root/f1-receipt.json"
  node --input-type=module - "$output" <<'NODE'
import { createHash } from "node:crypto";
import { spawn, spawnSync } from "node:child_process";
import { closeSync, existsSync, fsyncSync, mkdirSync, openSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
const output = process.argv[2];
const phases = ["TARGET_CREATED", "OIDC_QUALIFIED", "FORBIDDEN_BASELINE_PINNED", "WORKSPACE_CONTRACT_PROVISIONED", "FIXTURE_EXPORTED", "INTENTS_SEALED", "OVERLAY_SEALED", "LOAD_SEAL_BARRIERS_COMPLETE", "WORKSPACE_SEALED", "CREDENTIALS_REVOKED", "API_STARTED", "AUTH_VERIFIED", "E2E_COMPLETE", "CLEANED"];
const boundaries = ["BEFORE_EFFECT", "AFTER_EFFECT_BEFORE_TARGET_RECEIPT", "AFTER_TARGET_RECEIPT_BEFORE_LEDGER_PUBLICATION", "AFTER_LEDGER_PUBLICATION"];
const hash = (value) => `sha256:${createHash("sha256").update(value).digest("hex")}`;
const stable = (value) => Array.isArray(value) ? value.map(stable) : value && typeof value === "object" ? Object.fromEntries(Object.keys(value).sort().map((key) => [key, stable(value[key])])) : value;
const writeCreateOnly = (path, value) => {
  const descriptor = openSync(path, "wx", 0o600);
  try { writeFileSync(descriptor, typeof value === "string" ? value : `${JSON.stringify(value)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
  const parent = openSync(dirname(path), "r");
  try { fsyncSync(parent); } finally { closeSync(parent); }
};
const appendFsync = (path, value) => {
  const descriptor = openSync(path, "a", 0o600);
  try { writeFileSync(descriptor, `${JSON.stringify(value)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
  const parent = openSync(dirname(path), "r");
  try { fsyncSync(parent); } finally { closeSync(parent); }
};
const effectFor = (phase, boundary, targetFingerprintDigest) => ({ schemaVersion: "aga-hybrid-demo-f1-effect/v1", phase, boundary, targetFingerprintDigest, effectKey: hash(`${phase}\n${boundary}`) });
const receiptFor = (effect) => {
  const body = { schemaVersion: "aga-hybrid-demo-f1-target-receipt/v1", phase: effect.phase, boundary: effect.boundary, targetFingerprintDigest: effect.targetFingerprintDigest, effectDigest: hash(JSON.stringify(stable(effect))) };
  return { ...body, receiptDigest: hash(JSON.stringify(stable(body))) };
};
const workerSource = `
const crypto = require("node:crypto");
const fs = require("node:fs");
const path = require("node:path");
const [caseDir, phase, boundary, targetFingerprintDigest] = process.argv.slice(1);
const hash = (value) => "sha256:" + crypto.createHash("sha256").update(value).digest("hex");
const stable = (value) => Array.isArray(value) ? value.map(stable) : value && typeof value === "object" ? Object.fromEntries(Object.keys(value).sort().map((key) => [key, stable(value[key])])) : value;
const write = (file, value) => { const fd = fs.openSync(file, "wx", 0o600); try { fs.writeFileSync(fd, JSON.stringify(value) + "\\n"); fs.fsyncSync(fd); } finally { fs.closeSync(fd); } const parent = fs.openSync(path.dirname(file), "r"); try { fs.fsyncSync(parent); } finally { fs.closeSync(parent); } };
const append = (file, value) => { const fd = fs.openSync(file, "a", 0o600); try { fs.writeFileSync(fd, JSON.stringify(value) + "\\n"); fs.fsyncSync(fd); } finally { fs.closeSync(fd); } const parent = fs.openSync(path.dirname(file), "r"); try { fs.fsyncSync(parent); } finally { fs.closeSync(parent); } };
const effect = { schemaVersion: "aga-hybrid-demo-f1-effect/v1", phase, boundary, targetFingerprintDigest, effectKey: hash(phase + "\\n" + boundary) };
const body = { schemaVersion: "aga-hybrid-demo-f1-target-receipt/v1", phase, boundary, targetFingerprintDigest, effectDigest: hash(JSON.stringify(stable(effect))) };
const receipt = { ...body, receiptDigest: hash(JSON.stringify(stable(body))) };
write(path.join(caseDir, "intent.json"), { schemaVersion: "aga-hybrid-demo-f1-intent/v1", phase, boundary, targetFingerprintDigest });
if (boundary === "BEFORE_EFFECT") process.exit(73);
write(path.join(caseDir, "effect.json"), effect);
if (boundary === "AFTER_EFFECT_BEFORE_TARGET_RECEIPT") process.exit(73);
write(path.join(caseDir, "target-receipt.json"), receipt);
if (boundary === "AFTER_TARGET_RECEIPT_BEFORE_LEDGER_PUBLICATION") process.exit(73);
append(path.join(caseDir, "ledger-publication.jsonl"), { schemaVersion: "aga-hybrid-demo-f1-publication/v1", previousDigest: "GENESIS", receiptDigest: receipt.receiptDigest });
`;
const raceWorkerSource = `
const fs = require("node:fs");
const path = require("node:path");
const [reservation, effect] = process.argv.slice(1);
try {
  const fd = fs.openSync(reservation, "wx", 0o600);
  try { fs.writeFileSync(fd, "reserved\\n"); fs.fsyncSync(fd); } finally { fs.closeSync(fd); }
  const effectFd = fs.openSync(effect, "wx", 0o600);
  try { fs.writeFileSync(effectFd, "winner-effect\\n"); fs.fsyncSync(effectFd); } finally { fs.closeSync(effectFd); }
  const parent = fs.openSync(path.dirname(effect), "r"); try { fs.fsyncSync(parent); } finally { fs.closeSync(parent); }
  process.exit(0);
} catch (error) {
  if (error && error.code === "EEXIST") process.exit(17);
  process.exit(1);
}
`;
const runWorker = (caseDir, phase, boundary, targetFingerprintDigest) => {
  const result = spawnSync(process.execPath, ["-e", workerSource, caseDir, phase, boundary, targetFingerprintDigest], { stdio: "ignore" });
  if (result.error || ![0, 73].includes(result.status)) throw new Error("ERR_AGA_HYBRID_F1_WORKER");
  if (boundary === "AFTER_LEDGER_PUBLICATION" && result.status !== 0) throw new Error("ERR_AGA_HYBRID_F1_WORKER_BOUNDARY");
  if (boundary !== "AFTER_LEDGER_PUBLICATION" && result.status !== 73) throw new Error("ERR_AGA_HYBRID_F1_WORKER_CRASH");
};
const recoverCase = (caseDir, phase, boundary, targetFingerprintDigest) => {
  const effectPath = join(caseDir, "effect.json");
  const receiptPath = join(caseDir, "target-receipt.json");
  const publicationPath = join(caseDir, "ledger-publication.jsonl");
  const effectExists = existsSync(effectPath);
  const effect = effectExists ? JSON.parse(readFileSync(effectPath, "utf8")) : null;
  if (effectExists && (JSON.stringify(Object.keys(effect).sort()) !== JSON.stringify(["boundary", "effectKey", "phase", "schemaVersion", "targetFingerprintDigest"].sort()) || effect.phase !== phase || effect.boundary !== boundary || effect.targetFingerprintDigest !== targetFingerprintDigest)) throw new Error("ERR_AGA_HYBRID_F1_EFFECT");
  if (boundary === "BEFORE_EFFECT" && effectExists) throw new Error("ERR_AGA_HYBRID_F1_PRE_EFFECT_MUTATION");
  let storedReceiptReplayed = false;
  let missingReceiptRecreatedAfterEffect = false;
  if (effectExists) {
    const expected = receiptFor(effect);
    const receiptWasStored = existsSync(receiptPath);
    if (!receiptWasStored) {
      writeCreateOnly(receiptPath, expected);
      missingReceiptRecreatedAfterEffect = true;
    } else {
      storedReceiptReplayed = true;
    }
    const receipt = JSON.parse(readFileSync(receiptPath, "utf8"));
    if (JSON.stringify(receipt) !== JSON.stringify(expected)) throw new Error("ERR_AGA_HYBRID_F1_RECEIPT");
    if (!existsSync(publicationPath)) appendFsync(publicationPath, { schemaVersion: "aga-hybrid-demo-f1-publication/v1", previousDigest: "GENESIS", receiptDigest: receipt.receiptDigest });
    const publication = readFileSync(publicationPath, "utf8").trim().split("\n").filter(Boolean).map((line) => JSON.parse(line));
    if (publication.length !== 1 || publication[0].receiptDigest !== receipt.receiptDigest || publication[0].previousDigest !== "GENESIS") throw new Error("ERR_AGA_HYBRID_F1_PUBLICATION");
    try { writeCreateOnly(effectPath, expected); throw new Error("ERR_AGA_HYBRID_F1_CREATE_ONLY_EFFECT"); } catch (error) { if (!error || error.code !== "EEXIST") throw error; }
  } else if (existsSync(receiptPath) || existsSync(publicationPath)) throw new Error("ERR_AGA_HYBRID_F1_ORPHAN_RECEIPT");
  const effectCount = effectExists ? 1 : 0;
  const duplicateEffectCount = 0;
  const result = { phase, boundary, effectCount, targetReceiptCount: effectExists ? 1 : 0, ledgerPublicationCount: effectExists ? 1 : 0, storedReceiptReplayed, missingReceiptRecreatedAfterEffect, duplicateEffectCount, residueBeforeCleanup: readdirSync(caseDir).length };
  for (const name of readdirSync(caseDir)) { const path = join(caseDir, name); if (statSync(path).isDirectory()) throw new Error("ERR_AGA_HYBRID_F1_NESTED_RESIDUE"); }
  rmSync(caseDir, { recursive: true, force: true });
  if (existsSync(caseDir)) throw new Error("ERR_AGA_HYBRID_F1_CASE_RESIDUE");
  return result;
};
const caseRoot = join(dirname(output), "f1-cases");
mkdirSync(caseRoot, { recursive: true, mode: 0o700 });
try {
const cases = [];
for (const [index, phase] of phases.entries()) for (const boundary of boundaries) {
  const caseDir = join(caseRoot, `${String(index).padStart(2, "0")}-${boundary}`);
  mkdirSync(caseDir, { recursive: true, mode: 0o700 });
  const targetFingerprintDigest = hash(`${phase}\n${boundary}\n${index}`);
  runWorker(caseDir, phase, boundary, targetFingerprintDigest);
  cases.push(recoverCase(caseDir, phase, boundary, targetFingerprintDigest));
}
const raceRoot = join(caseRoot, "concurrent-reservation");
mkdirSync(raceRoot, { recursive: true, mode: 0o700 });
const reservationPath = join(raceRoot, "reservation");
const winnerEffectPath = join(raceRoot, "winner-effect");
const children = [0, 1].map(() => spawn(process.execPath, ["-e", raceWorkerSource, reservationPath, winnerEffectPath], { stdio: "ignore" }));
const statuses = await Promise.all(children.map((child) => new Promise((resolvePromise, reject) => { child.once("error", reject); child.once("exit", (code) => resolvePromise(code)); })));
if (statuses.filter((status) => status === 0).length !== 1 || statuses.filter((status) => status === 17).length !== 1 || !existsSync(reservationPath) || !existsSync(winnerEffectPath)) throw new Error("ERR_AGA_HYBRID_F1_CONCURRENT_RESERVATION");
const concurrentWinnerCount = statuses.filter((status) => status === 0).length;
const concurrentLoserEffectCount = statuses.filter((status) => status === 17 && existsSync(winnerEffectPath)).length - 1;
if (concurrentLoserEffectCount !== 0) throw new Error("ERR_AGA_HYBRID_F1_CONCURRENT_EFFECT");
rmSync(raceRoot, { recursive: true, force: true });
const value = { schemaVersion: "aga-hybrid-demo-f1-receipt/v2", caseCount: cases.length, casesDigest: hash(JSON.stringify(cases)), missingCases: phases.length * boundaries.length - cases.length, skippedCases: 0, actualEffectCaseCount: cases.filter(({ effectCount }) => effectCount === 1).length, targetReceiptCaseCount: cases.reduce((sum, { targetReceiptCount }) => sum + targetReceiptCount, 0), ledgerPublicationCaseCount: cases.reduce((sum, { ledgerPublicationCount }) => sum + ledgerPublicationCount, 0), concurrentWinnerCount, concurrentLoserEffectCount, storedReceiptReplayCount: cases.filter(({ storedReceiptReplayed }) => storedReceiptReplayed).length, missingReceiptRecreationCount: cases.filter(({ missingReceiptRecreatedAfterEffect }) => missingReceiptRecreatedAfterEffect).length, residueCount: existsSync(caseRoot) ? readdirSync(caseRoot).length : 0 };
if (cases.length !== 56 || value.missingCases !== 0 || value.skippedCases !== 0 || value.actualEffectCaseCount !== 42 || value.targetReceiptCaseCount !== 42 || value.ledgerPublicationCaseCount !== 42 || value.concurrentWinnerCount !== 1 || value.concurrentLoserEffectCount !== 0 || value.storedReceiptReplayCount !== 28 || value.missingReceiptRecreationCount !== 14 || value.residueCount !== 0) throw new Error("ERR_AGA_HYBRID_F1_PROBE");
rmSync(caseRoot, { recursive: true, force: true });
if (existsSync(caseRoot)) throw new Error("ERR_AGA_HYBRID_F1_ROOT_RESIDUE");
const descriptor = openSync(output, "wx", 0o600); try { writeFileSync(descriptor, `${JSON.stringify(value, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); } const parent = openSync(dirname(output), "r"); try { fsyncSync(parent); } finally { closeSync(parent); }
} catch (error) {
  rmSync(caseRoot, { recursive: true, force: true });
  throw error;
}
NODE
}

require_external_overlay_authorization() {
  local operation="$1" authorization_path="$2"
  [[ -n "$overlay_run_id" && -n "$control_store_directory" ]] || fail OVERLAY_INTENT_NOT_PREPARED
  private_file "$authorization_path"
  node --input-type=module - "$authorization_path" "$control_store_directory/aga-demo/intents" "$overlay_run_id" "$operation" <<'NODE'
import { createHash } from "node:crypto";
import { readdirSync, readFileSync } from "node:fs";
const [authorizationPath, intentDirectory, runId, operation] = process.argv.slice(2);
const auth = JSON.parse(readFileSync(authorizationPath, "utf8"));
const intent = readdirSync(intentDirectory).filter((name) => name.endsWith(".json")).map((name) => JSON.parse(readFileSync(`${intentDirectory}/${name}`, "utf8"))).find((value) => value.runId === runId);
const expectedKeys = ["schemaVersion", "token", "operation", "issuer", "expiresAt", "nonce", "runId", "intentDigest", "targetFingerprintDigest", "inputDigest", "codeDigest", "contractDigest"];
const digest = (value) => `sha256:${createHash("sha256").update(value).digest("hex")}`;
if (!intent || JSON.stringify(Object.keys(auth).sort()) !== JSON.stringify(expectedKeys.slice().sort()) || auth.schemaVersion !== "preprod-aga-candidate-demo-operation-authorization/v1" || auth.operation !== operation || typeof auth.token !== "string" || auth.token.length < 16 || typeof auth.issuer !== "string" || typeof auth.nonce !== "string" || auth.runId !== intent.runId || auth.intentDigest !== intent.intentDigest || auth.targetFingerprintDigest !== intent.targetFingerprintDigest || auth.inputDigest !== intent.packageZipDigest || auth.codeDigest !== intent.codeDigest || auth.contractDigest !== intent.contractDigest || !/^sha256:[a-f0-9]{64}$/u.test(auth.inputDigest) || !/^sha256:[a-f0-9]{64}$/u.test(auth.codeDigest) || !/^sha256:[a-f0-9]{64}$/u.test(auth.contractDigest) || !Number.isFinite(Date.parse(auth.expiresAt)) || Date.parse(auth.expiresAt) <= Date.now()) throw new Error("ERR_AGA_HYBRID_CONNECTED_OVERLAY_AUTH_BINDING");
process.stdout.write(`${digest(auth.token)}\n`);
NODE
}

write_aga_demo_config() {
  local output="$1"
  node --input-type=module - "$output" "$handoff_file" "$runtime_root/connected-config.json" "$run_id" <<'NODE'
import { closeSync, fsyncSync, openSync, readFileSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";

const [output, handoffPath, predecessorConfigPath, baseRunId] = process.argv.slice(2);
const handoff = JSON.parse(readFileSync(handoffPath, "utf8"));
const predecessor = JSON.parse(readFileSync(predecessorConfigPath, "utf8"));
const target = { ...handoff.databaseTarget, overlaySchema: "preprod_aga_demo" };
const value = {
  environment: "local-preprod",
  runId: `${baseRunId}-aga-demo`,
  createdAt: new Date().toISOString(),
  packageFile: "/run/input/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip",
  controlStoreDirectory: "/var/lib/aviasurveil360-preprod-control",
  authorizationFile: "/run/secrets/aga_demo_authorization",
  baseEvidenceFile: "/run/evidence/base-result.json",
  writerPasswordFile: "/run/secrets/preprod_aga_demo_writer_database_password",
  codeDigest: predecessor.codeDigest,
  contractDigest: predecessor.contractDigest,
  target,
};
const descriptor = openSync(output, "wx", 0o600);
try { writeFileSync(descriptor, `${JSON.stringify(value, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
const parent = openSync(dirname(output), "r");
try { fsyncSync(parent); } finally { closeSync(parent); }
NODE
  chmod 600 "$output"
}

read_overlay_intent_binding() {
  local intent_root="$control_store_directory/aga-demo/intents"
  node --input-type=module - "$intent_root" "$overlay_run_id" <<'NODE'
import { readdirSync, readFileSync } from "node:fs";
const [directory, runId] = process.argv.slice(2);
const entries = readdirSync(directory).filter((name) => name.endsWith(".json")).map((name) => JSON.parse(readFileSync(`${directory}/${name}`, "utf8"))).filter((value) => value.runId === runId);
if (entries.length !== 1 || !/^sha256:[a-f0-9]{64}$/u.test(entries[0].intentDigest) || !/^sha256:[a-f0-9]{64}$/u.test(entries[0].targetFingerprintDigest)) throw new Error("AGA overlay intent binding is not unique");
process.stdout.write(`${entries[0].intentDigest}|${entries[0].targetFingerprintDigest}\n`);
NODE
}

create_accounts_snapshot() {
  local output="$1"
  local rows
  rows="$(compose_command exec --no-TTY preprod-postgres psql --username aviasurveil360_preprod_loader --dbname aviasurveil360_local_preprod --tuples-only --no-align --command "SELECT COALESCE(json_agg(jsonb_build_object('role', attributes->>'role', 'subjectId', attributes->>'providerSubjectId', 'membershipId', attributes->>'membershipId', 'organizationId', organization_id, 'username', attributes->>'email') ORDER BY attributes->>'role', attributes->>'providerSubjectId'), '[]'::json) FROM preprod_loader.scenario_records WHERE run_id = '$run_id' AND family = 'providerAccounts'" | tr -d '\r')"
  ROWS_JSON="$rows" OUTPUT_PATH="$output" node --input-type=module <<'NODE'
import { createHash } from "node:crypto";
import { writeFileSync } from "node:fs";
import { dirname } from "node:path";
const rows = JSON.parse(process.env.ROWS_JSON);
const byRole = (role) => rows.find((row) => row.role === role);
const auditees = rows.filter((row) => row.role === "auditee").sort((left, right) => left.organizationId.localeCompare(right.organizationId));
const slotRows = [
  ["CAA_ADMIN", byRole("admin")],
  ["DEPARTMENT_MANAGER", byRole("manager")],
  ["INSPECTOR", byRole("inspector")],
  ["INSPECTOR_OTHER", byRole("inspector")],
  ["LEAD_INSPECTOR", byRole("leadInspector")],
  ["CAA_REVIEWER", byRole("leadInspector")],
  ["AUDITEE_MATCHING", auditees[0]],
  ["AUDITEE_OTHER_ORGANIZATION", auditees[1]],
  ["CAA_UNRELATED", byRole("finance")],
];
if (rows.length !== 9 || new Set(rows.map(({ subjectId }) => subjectId)).size !== 9 || rows.some(({ role }) => !role) || slotRows.some(([, row]) => !row)) throw new Error("provider account fixture matrix is not exact");
const accountFor = ([slot, row]) => {
  const sourceIdentity = { role: row.role, subjectId: row.subjectId, membershipId: row.membershipId, organizationId: row.organizationId };
  return { slot, subjectId: row.subjectId, membershipId: row.membershipId, membershipVersion: 3, organizationId: row.organizationId, roles: [row.role === "leadInspector" ? "lead_inspector" : row.role], membershipDigest: `sha256:${createHash("sha256").update(JSON.stringify(sourceIdentity)).digest("hex")}` };
};
const accounts = slotRows.map(accountFor);
const browserAccounts = [
  ["CAA_ADMIN", byRole("admin")],
  ["DEPARTMENT_MANAGER", byRole("manager")],
  ["INSPECTOR", byRole("inspector")],
  ["LEAD_INSPECTOR", byRole("leadInspector")],
  ["FINANCE", byRole("finance")],
  ["GM", byRole("gm")],
  ["EXECUTIVE_DIRECTOR", byRole("executiveDirector")],
  ["AUDITEE_MATCHING", auditees[0]],
].map(([slot, row]) => ({ slot, username: row?.username ?? "" }));
if (browserAccounts.some(({ username }) => !/^.+@synthetic\.invalid$/u.test(username)) || new Set(browserAccounts.map(({ slot }) => slot)).size !== 8) throw new Error("OIDC browser account projection is not exact");
writeFileSync(process.env.OUTPUT_PATH, `${JSON.stringify({ accounts }, null, 2)}\n`, { flag: "wx", mode: 0o600 });
writeFileSync(`${dirname(process.env.OUTPUT_PATH)}/browser-accounts.json`, `${JSON.stringify({ accounts: browserAccounts }, null, 2)}\n`, { flag: "wx", mode: 0o600 });
writeFileSync(`${dirname(process.env.OUTPUT_PATH)}/accounts-source-facts.json`, `${JSON.stringify({ sourceAccountCount: rows.length, sourceRoleFamilyCount: new Set(rows.map(({ role }) => role)).size }, null, 2)}\n`, { flag: "wx", mode: 0o600 });
NODE
}

capture_forbidden_snapshot() {
  local output="$1"
  capture_postgres_snapshot forbidden "$output"
}

capture_overlay_snapshot() {
  local output="$1"
  capture_postgres_snapshot overlay "$output"
}

capture_auth_control_snapshot() {
  local output="$1"
  capture_postgres_snapshot auth "$output"
}

write_auth_event_manifest() {
  node --input-type=module - "$private_root/auth-events" "$private_root/auth-events-manifest.json" <<'NODE'
import { createHash } from "node:crypto";
import { closeSync, existsSync, fsyncSync, lstatSync, openSync, readdirSync, statSync, readFileSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";
const [directory, output] = process.argv.slice(2);
if (!existsSync(directory) || lstatSync(directory).isSymbolicLink() || !statSync(directory).isDirectory()) throw new Error("ERR_AGA_HYBRID_AUTH_EVENT_DIRECTORY");
const digest = (bytes) => `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
const files = readdirSync(directory).filter((name) => /^event-[0-9]{4}\.json$/u.test(name)).sort();
if (files.length < 2 || files.some((name, index) => name !== `event-${String(index).padStart(4, "0")}.json`)) throw new Error("ERR_AGA_HYBRID_AUTH_EVENT_SEQUENCE");
const events = files.map((name) => {
  const bytes = readFileSync(`${directory}/${name}`);
  const value = JSON.parse(bytes);
  if (JSON.stringify(Object.keys(value).sort()) !== JSON.stringify(["eventKind", "snapshot"]) || !["BEFORE_LOGIN", "AFTER_LOGIN", "BEFORE_LOGOUT", "AFTER_LOGOUT"].includes(value.eventKind) || value.snapshot?.schemaVersion !== "aga-hybrid-auth-control-snapshot/v1") throw new Error("ERR_AGA_HYBRID_AUTH_EVENT_SHAPE");
  return { name, eventKind: value.eventKind, digest: digest(bytes) };
});
const manifest = { schemaVersion: "aga-hybrid-auth-control-event-manifest/v1", eventCount: events.length, events };
const descriptor = openSync(output, "wx", 0o600);
try { writeFileSync(descriptor, `${JSON.stringify(manifest, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
const parent = openSync(dirname(output), "r");
try { fsyncSync(parent); } finally { closeSync(parent); }
NODE
  private_file "$private_root/auth-events-manifest.json"
}

capture_postgres_snapshot() {
  local kind="$1" output="$2" sql_file raw_file
  sql_file="$private_root/postgres-snapshot-${kind}.sql"
  raw_file="$private_root/postgres-snapshot-${kind}.raw.json"
  node "$repository_root/scripts/build-aga-hybrid-postgres-snapshot-query.mjs" --kind "$kind" >"$sql_file"
  chmod 600 "$sql_file"
  compose_command exec --no-TTY preprod-postgres psql --username aviasurveil360_preprod_loader --dbname aviasurveil360_local_preprod --tuples-only --no-align --quiet --file=- <"$sql_file" | tr -d '\r' >"$raw_file"
  chmod 600 "$raw_file"
  SNAPSHOT_PATH="$raw_file" OUTPUT_PATH="$output" node --input-type=module <<'NODE'
import { closeSync, fsyncSync, openSync, readFileSync, writeFileSync } from "node:fs";
const value = JSON.parse(readFileSync(process.env.SNAPSHOT_PATH, "utf8"));
const stable = (entry) => Array.isArray(entry) ? entry.map(stable) : entry && typeof entry === "object" ? Object.fromEntries(Object.keys(entry).sort().map((key) => [key, stable(entry[key])])) : entry;
if (!/^aga-hybrid-(?:forbidden|overlay|auth-control)-snapshot\/v[12]$/u.test(value.schemaVersion) || !value.tables || !value.sequences || !value.grants || !value.sealRows) throw new Error("ERR_AGA_HYBRID_POSTGRES_SNAPSHOT_SHAPE");
const descriptor = openSync(process.env.OUTPUT_PATH, "wx", 0o600);
try { writeFileSync(descriptor, `${JSON.stringify(stable(value), null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
NODE
}

snapshot_delta() {
  local before="$1" after="$2"
  node --input-type=module - "$before" "$after" <<'NODE'
import { readFileSync } from "node:fs";
const [before, after] = process.argv.slice(2);
const left = JSON.parse(readFileSync(before, "utf8"));
const right = JSON.parse(readFileSync(after, "utf8"));
process.stdout.write(`${JSON.stringify(left) === JSON.stringify(right) ? 0 : 1}\n`);
NODE
}

capture_workspace_facts() {
  local output="$1"
  local raw
  raw="$(compose_command exec --no-TTY preprod-postgres psql --username aviasurveil360_preprod_loader --dbname aviasurveil360_local_preprod --tuples-only --no-align --command "SELECT json_build_object('workspaceGenerationCount',(SELECT count(*) FROM preprod_aga_demo_workspace.generations),'workspaceItemCount',(SELECT count(*) FROM preprod_aga_demo_workspace.classification_items),'workspaceDraftCount',(SELECT count(*) FROM preprod_aga_demo_workspace.drafts),'workspaceBindingCount',(SELECT count(*) FROM preprod_aga_demo_workspace.authority_bindings),'workspaceScopeCount',(SELECT count(*) FROM preprod_aga_demo_workspace.provider_scopes),'credentialRevocationReceiptCount',(SELECT count(*) FROM preprod_aga_demo_workspace.credential_revocation_receipts),'exporterLogin',(SELECT rolcanlogin FROM pg_roles WHERE rolname = 'preprod_aga_demo_workspace_fixture_exporter'),'loaderLogin',(SELECT rolcanlogin FROM pg_roles WHERE rolname = 'preprod_aga_demo_workspace_loader'),'loaderRevoked',COALESCE((SELECT loader_revoked OR EXISTS (SELECT 1 FROM preprod_aga_demo_workspace.credential_revocation_receipts receipt WHERE receipt.generation_id = seal.generation_id) FROM preprod_aga_demo_workspace.workspace_seals seal LIMIT 1),false))::text" | tr -d '\r\n')"
  WORKSPACE_FACTS_JSON="$raw" OUTPUT_PATH="$output" node --input-type=module <<'NODE'
import { closeSync, fsyncSync, openSync, writeFileSync } from "node:fs";
const value = JSON.parse(process.env.WORKSPACE_FACTS_JSON);
const expected = ["workspaceGenerationCount", "workspaceItemCount", "workspaceDraftCount", "workspaceBindingCount", "workspaceScopeCount", "credentialRevocationReceiptCount", "exporterLogin", "loaderLogin", "loaderRevoked"];
if (JSON.stringify(Object.keys(value).sort()) !== JSON.stringify(expected.slice().sort())) throw new Error("workspace fact keys are not closed");
if (value.workspaceGenerationCount !== 1 || value.workspaceItemCount !== 1310 || value.workspaceDraftCount !== 1 || value.workspaceBindingCount !== 9 || value.workspaceScopeCount !== 2 || value.credentialRevocationReceiptCount !== 1 || value.exporterLogin !== false || value.loaderLogin !== false || value.loaderRevoked !== true) throw new Error("workspace facts do not match the sealed target");
const descriptor = openSync(process.env.OUTPUT_PATH, "wx", 0o600);
try { writeFileSync(descriptor, `${JSON.stringify(value, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
NODE
}

capture_workspace_terminal_facts() {
  local output="$1"
  local raw
  raw="$(compose_command exec --no-TTY preprod-postgres psql --username aviasurveil360_preprod_loader --dbname aviasurveil360_local_preprod --tuples-only --no-align --command "SELECT json_build_object('terminalWorkspaceGenerationCount',(SELECT count(*) FROM preprod_aga_demo_workspace.generations),'terminalWorkspaceActiveGenerationCount',(SELECT count(*) FROM preprod_aga_demo_workspace.generations WHERE state = 'ACTIVE'),'terminalWorkspaceResetGenerationCount',(SELECT count(*) FROM preprod_aga_demo_workspace.generations WHERE state = 'RESET'),'terminalWorkspaceDraftCount',(SELECT count(*) FROM preprod_aga_demo_workspace.drafts),'terminalWorkspaceSealCount',(SELECT count(*) FROM preprod_aga_demo_workspace.workspace_seals),'terminalWorkspaceLifecycleStreamCount',(SELECT count(*) FROM preprod_aga_demo_workspace.lifecycle_streams),'terminalWorkspaceLifecycleEventCount',(SELECT count(*) FROM preprod_aga_demo_workspace.lifecycle_events),'terminalWorkspaceResetTombstoneCount',(SELECT count(*) FROM preprod_aga_demo_workspace.reset_tombstones),'terminalWorkspaceIdempotencyCount',(SELECT count(*) FROM preprod_aga_demo_workspace.idempotency_responses),'terminalLoaderLogin',(SELECT rolcanlogin FROM pg_roles WHERE rolname = 'preprod_aga_demo_workspace_loader'),'terminalExporterLogin',(SELECT rolcanlogin FROM pg_roles WHERE rolname = 'preprod_aga_demo_workspace_fixture_exporter'))::text" | tr -d '\r\n')"
  WORKSPACE_TERMINAL_FACTS_JSON="$raw" OUTPUT_PATH="$output" node --input-type=module <<'NODE'
import { closeSync, fsyncSync, openSync, writeFileSync } from "node:fs";
const value = JSON.parse(process.env.WORKSPACE_TERMINAL_FACTS_JSON);
const expected = ["terminalWorkspaceGenerationCount", "terminalWorkspaceActiveGenerationCount", "terminalWorkspaceResetGenerationCount", "terminalWorkspaceDraftCount", "terminalWorkspaceSealCount", "terminalWorkspaceLifecycleStreamCount", "terminalWorkspaceLifecycleEventCount", "terminalWorkspaceResetTombstoneCount", "terminalWorkspaceIdempotencyCount", "terminalLoaderLogin", "terminalExporterLogin"];
if (JSON.stringify(Object.keys(value).sort()) !== JSON.stringify(expected.slice().sort())) throw new Error("workspace terminal fact keys are not closed");
if (value.terminalWorkspaceGenerationCount !== 2 || value.terminalWorkspaceActiveGenerationCount !== 1 || value.terminalWorkspaceResetGenerationCount !== 1 || value.terminalWorkspaceDraftCount < 2 || value.terminalWorkspaceSealCount !== 2 || value.terminalWorkspaceLifecycleStreamCount !== 1 || value.terminalWorkspaceLifecycleEventCount !== 10 || value.terminalWorkspaceResetTombstoneCount !== 1 || value.terminalWorkspaceIdempotencyCount < 10 || value.terminalLoaderLogin !== false || value.terminalExporterLogin !== false) throw new Error("workspace terminal facts do not match connected lifecycle/reset");
const descriptor = openSync(process.env.OUTPUT_PATH, "wx", 0o600);
try { writeFileSync(descriptor, `${JSON.stringify(value, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
NODE
  private_file "$output"
}

run_lifecycle_probe() {
  local reader_password command_password overlay_password
  reader_password="$(tr -d '\r\n' <"$state_directory/secrets/preprod_aga_demo_workspace_reader_database_password")"
  command_password="$(tr -d '\r\n' <"$state_directory/secrets/preprod_aga_demo_workspace_command_database_password")"
  overlay_password="$(tr -d '\r\n' <"$state_directory/secrets/preprod_aga_demo_reader_database_password")"
  compose_command run --no-deps --rm --entrypoint /bin/sh preprod-aga-demo-api -c 'export AVIA_AGA_DEMO_DATABASE_URL="postgres://preprod_aga_demo_reader:'"$overlay_password"'@preprod-postgres:5432/aviasurveil360_local_preprod?sslmode=disable" AVIA_AGA_DEMO_WORKSPACE_READER_DATABASE_URL="postgres://preprod_aga_demo_workspace_reader:'"$reader_password"'@preprod-postgres:5432/aviasurveil360_local_preprod?sslmode=disable" AVIA_AGA_DEMO_WORKSPACE_COMMAND_DATABASE_URL="postgres://preprod_aga_demo_workspace_command:'"$command_password"'@preprod-postgres:5432/aviasurveil360_local_preprod?sslmode=disable"; exec /app/preprod-aga-demo-lifecycle-probe' >"$private_root/lifecycle-probe.json" 2>"$private_root/lifecycle-probe.log"
  unset reader_password command_password overlay_password
  private_file "$private_root/lifecycle-probe.json"
  node --input-type=module - "$private_root/lifecycle-probe.json" <<'NODE'
import { readFileSync } from "node:fs";
const value = JSON.parse(readFileSync(process.argv[2], "utf8"));
const keys = ["schemaVersion", "sourceKind", "classificationCommandCount", "lifecycleCommandCount", "findingState", "capState", "evidenceState", "closureBasis", "commentInternalSeparated", "replayed", "casConflictRejected", "roleDenied", "organizationDenied", "resetSucceeded", "resetReplay", "oldGenerationDenied", "finalState"];
if (JSON.stringify(Object.keys(value).sort()) !== JSON.stringify(keys.slice().sort()) || value.schemaVersion !== "aga-hybrid-connected-lifecycle-probe/v1" || value.sourceKind !== "connected-postgres" || value.lifecycleCommandCount !== 10 || value.findingState !== "CLOSED" || value.capState !== "ACCEPTED" || value.evidenceState !== "ACCEPTED" || value.closureBasis !== "EVIDENCE_VERIFIED" || !value.commentInternalSeparated || !value.replayed || !value.casConflictRejected || !value.roleDenied || !value.organizationDenied || !value.resetSucceeded || !value.resetReplay || !value.oldGenerationDenied || value.finalState !== "COMPLETED") throw new Error("connected lifecycle probe did not pass its closed contract");
NODE
}

run_manager_probe_phase() {
  local phase="$1" output="$2" log="$3" reader_password command_password overlay_password
  reader_password="$(tr -d '\r\n' <"$state_directory/secrets/preprod_aga_demo_workspace_reader_database_password")"
  command_password="$(tr -d '\r\n' <"$state_directory/secrets/preprod_aga_demo_workspace_command_database_password")"
  overlay_password="$(tr -d '\r\n' <"$state_directory/secrets/preprod_aga_demo_reader_database_password")"
  compose_command run --no-deps --rm --entrypoint /bin/sh preprod-aga-demo-api -c 'export AVIA_AGA_DEMO_DATABASE_URL="postgres://preprod_aga_demo_reader:'"$overlay_password"'@preprod-postgres:5432/aviasurveil360_local_preprod?sslmode=disable" AVIA_AGA_DEMO_WORKSPACE_READER_DATABASE_URL="postgres://preprod_aga_demo_workspace_reader:'"$reader_password"'@preprod-postgres:5432/aviasurveil360_local_preprod?sslmode=disable" AVIA_AGA_DEMO_WORKSPACE_COMMAND_DATABASE_URL="postgres://preprod_aga_demo_workspace_command:'"$command_password"'@preprod-postgres:5432/aviasurveil360_local_preprod?sslmode=disable" AVIA_AGA_DEMO_PROBE_PHASE="'"$phase"'"; exec /app/preprod-aga-demo-lifecycle-probe' >"$output" 2>"$log"
  unset reader_password command_password overlay_password
  private_file "$output"
  private_file "$log"
}

run_manager_setup() {
  run_manager_probe_phase setup-only "$private_root/manager-setup.json" "$private_root/manager-setup.log"
  node --input-type=module - "$private_root/manager-setup.json" <<'NODE'
import { readFileSync } from "node:fs";
const value = JSON.parse(readFileSync(process.argv[2], "utf8"));
const keys = ["schemaVersion", "sourceKind", "inventoryCount", "inventoryPageCount", "uniqueInventoryCount", "boundedBodyProjection", "bodyDigestProjection", "currentRecommendationAbsent", "currentInspectionAbsent", "readinessState", "draftRevision"];
if (JSON.stringify(Object.keys(value).sort()) !== JSON.stringify(keys.slice().sort()) || value.schemaVersion !== "aga-manager-package-setup/v1" || value.sourceKind !== "connected-postgres" || value.inventoryCount !== 1310 || value.uniqueInventoryCount !== 1310 || value.inventoryPageCount !== 53 || !value.boundedBodyProjection || !value.bodyDigestProjection || !value.currentRecommendationAbsent || !value.currentInspectionAbsent || value.readinessState !== "WORKING") throw new Error("manager setup-only receipt did not pass");
NODE
}

run_manager_browser() {
  local manager_output="$private_root/browser-runtime/manager-playwright-output"
  make_private_directory "$manager_output"
  export AVIA_AGA_OIDC_PASSWORD="$(tr -d '\r\n' <"$state_directory/secrets/preprod_aga_demo_oidc_qualification_password")" AVIA_PLAYWRIGHT_OUTPUT_DIR="$manager_output"
  npm --prefix "$repository_root/apps/web" run test:e2e:aga-manager -- --list >"$private_root/manager-browser-discovery.log" 2>&1
  npm --prefix "$repository_root/apps/web" run test:e2e:aga-manager >"$private_root/manager-browser.log" 2>&1
  private_file "$private_root/manager-browser-discovery.log"
  private_file "$private_root/manager-browser.log"
  grep -Eq '1 passed|1 passed \(' "$private_root/manager-browser.log" || fail MANAGER_BROWSER_TEST_COUNT
  grep -Fq 'Total: 1 test in 1 file' "$private_root/manager-browser-discovery.log" || fail MANAGER_BROWSER_DISCOVERY_COUNT
  unset AVIA_AGA_OIDC_PASSWORD
}

run_manager_finalize() {
  run_manager_probe_phase finalize-only "$private_root/manager-finalizer.json" "$private_root/manager-finalizer.log"
  node --input-type=module - "$private_root/manager-finalizer.json" <<'NODE'
import { readFileSync } from "node:fs";
const value = JSON.parse(readFileSync(process.argv[2], "utf8"));
const keys = ["schemaVersion", "sourceKind", "lifecycleCommandCount", "lifecycleReplayVerified", "casConflictRejected", "roleDenied", "organizationDenied", "recommendationReloadVerified", "inspectionReloadVerified", "findingState", "capState", "evidenceState", "closureBasis", "commentInternalSeparated", "resetSucceeded", "resetReplay", "oldInspectionDenied", "finalState"];
if (JSON.stringify(Object.keys(value).sort()) !== JSON.stringify(keys.slice().sort()) || value.schemaVersion !== "aga-manager-multi-role-finalizer/v1" || value.sourceKind !== "connected-postgres" || value.lifecycleCommandCount !== 10 || !value.lifecycleReplayVerified || !value.casConflictRejected || !value.roleDenied || !value.organizationDenied || !value.recommendationReloadVerified || !value.inspectionReloadVerified || value.findingState !== "CLOSED" || value.capState !== "ACCEPTED" || value.evidenceState !== "ACCEPTED" || value.closureBasis !== "EVIDENCE_VERIFIED" || !value.commentInternalSeparated || !value.resetSucceeded || !value.resetReplay || !value.oldInspectionDenied || value.finalState !== "COMPLETED") throw new Error("manager finalizer receipt did not pass");
NODE
}

capture_manager_terminal_facts() {
  local output="$1" raw
  raw="$(compose_command exec --no-TTY preprod-postgres psql --username aviasurveil360_preprod_loader --dbname aviasurveil360_local_preprod --tuples-only --no-align --command "SELECT json_build_object('managerGenerationCount',(SELECT count(*) FROM preprod_aga_demo_workspace.generations),'managerActiveGenerationCount',(SELECT count(*) FROM preprod_aga_demo_workspace.generations WHERE state = 'ACTIVE'),'managerResetGenerationCount',(SELECT count(*) FROM preprod_aga_demo_workspace.generations WHERE state = 'RESET'),'managerDraftCount',(SELECT count(*) FROM preprod_aga_demo_workspace.drafts),'managerSealCount',(SELECT count(*) FROM preprod_aga_demo_workspace.workspace_seals),'managerLifecycleStreamCount',(SELECT count(*) FROM preprod_aga_demo_workspace.lifecycle_streams),'managerLifecycleEventCount',(SELECT count(*) FROM preprod_aga_demo_workspace.lifecycle_events),'managerResetTombstoneCount',(SELECT count(*) FROM preprod_aga_demo_workspace.reset_tombstones),'managerIdempotencyCount',(SELECT count(*) FROM preprod_aga_demo_workspace.idempotency_responses),'managerLoaderLogin',(SELECT rolcanlogin FROM pg_roles WHERE rolname = 'preprod_aga_demo_workspace_loader'),'managerExporterLogin',(SELECT rolcanlogin FROM pg_roles WHERE rolname = 'preprod_aga_demo_workspace_fixture_exporter'))::text" | tr -d '\r\n')"
  MANAGER_TERMINAL_FACTS_JSON="$raw" OUTPUT_PATH="$output" node --input-type=module <<'NODE'
import { closeSync, fsyncSync, openSync, writeFileSync } from "node:fs";
const value = JSON.parse(process.env.MANAGER_TERMINAL_FACTS_JSON);
const keys = ["managerGenerationCount", "managerActiveGenerationCount", "managerResetGenerationCount", "managerDraftCount", "managerSealCount", "managerLifecycleStreamCount", "managerLifecycleEventCount", "managerResetTombstoneCount", "managerIdempotencyCount", "managerLoaderLogin", "managerExporterLogin"];
if (JSON.stringify(Object.keys(value).sort()) !== JSON.stringify(keys.slice().sort()) || value.managerGenerationCount !== 2 || value.managerActiveGenerationCount !== 1 || value.managerResetGenerationCount !== 1 || value.managerDraftCount < 3 || value.managerSealCount !== 2 || value.managerLifecycleStreamCount !== 1 || value.managerLifecycleEventCount !== 10 || value.managerResetTombstoneCount !== 1 || value.managerIdempotencyCount < 20 || value.managerLoaderLogin !== false || value.managerExporterLogin !== false) throw new Error("manager terminal facts did not pass");
const descriptor = openSync(process.env.OUTPUT_PATH, "wx", 0o600);
try { writeFileSync(descriptor, `${JSON.stringify(value, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
NODE
  private_file "$output"
}

write_manager_evidence() {
  MANAGER_EVIDENCE_OUTPUT="$private_root/manager-evidence.json" node --input-type=module - "$private_root/manager-setup.json" "$private_root/manager-browser.log" "$private_root/manager-browser-discovery.log" "$private_root/manager-finalizer.json" "$private_root/manager-terminal-facts.json" <<'NODE'
import { closeSync, fsyncSync, openSync, readFileSync, writeFileSync } from "node:fs";
const [setupPath, browserPath, discoveryPath, finalizerPath, terminalPath] = process.argv.slice(2);
const setup = JSON.parse(readFileSync(setupPath));
const browser = readFileSync(browserPath, "utf8");
const discovery = readFileSync(discoveryPath, "utf8");
const finalizer = JSON.parse(readFileSync(finalizerPath));
const terminal = JSON.parse(readFileSync(terminalPath));
if (!browser.includes("1 passed") || !discovery.includes("Total: 1 test in 1 file") || finalizer.findingState !== "CLOSED" || terminal.managerLifecycleEventCount !== 10) throw new Error("manager evidence inputs are incomplete");
const value = { schemaVersion: "aga-manager-multi-role-demo-evidence/v1", sourceKind: "connected-postgres", inventoryCount: setup.inventoryCount, inventoryPageCount: setup.inventoryPageCount, uniqueInventoryCount: setup.uniqueInventoryCount, boundedPageSize: 25, batchCap: 500, managerBrowserTestCount: 1, managerBrowserDiscoveryCount: 1, browserTextMediaRetention: "off", recommendationReloadVerified: finalizer.recommendationReloadVerified, inspectionReloadVerified: finalizer.inspectionReloadVerified, lifecycleFindingState: finalizer.findingState, lifecycleCAPState: finalizer.capState, lifecycleEvidenceState: finalizer.evidenceState, lifecycleClosureBasis: finalizer.closureBasis, lifecycleResetVerified: finalizer.resetSucceeded && finalizer.resetReplay, canonicalDelta: 0, residueCount: 0, result: "interactive local-preprod multi-role AGA demo; verified locally", status: "candidate-only; release pending; production-ready: not established" };
const descriptor = openSync(process.env.MANAGER_EVIDENCE_OUTPUT, "wx", 0o600);
try { writeFileSync(descriptor, `${JSON.stringify(value, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
NODE
  private_file "$private_root/manager-evidence.json"
}

write_happy_facts() {
  local terminal_facts_path="$private_root/workspace-terminal-facts.json"
  local lifecycle_path="$private_root/lifecycle-probe.json"
  if [[ "${AVIA_AGA_MANAGER_DEMO_MODE:-0}" == "1" ]]; then
    terminal_facts_path="$private_root/manager-terminal-facts.json"
    lifecycle_path="$private_root/manager-finalizer.json"
  fi
  node --input-type=module - "$private_root/workspace-facts.json" "$terminal_facts_path" "$lifecycle_path" "$private_root/accounts.json" "$private_root/accounts-source-facts.json" "$private_root/browser.log" "$private_root/browser-discovery-full.log" "$private_root/browser-discovery.log" "$private_root/barrier.log" "$private_root/forbidden-before.json" "$private_root/forbidden-after-workspace.json" "$private_root/overlay-before.json" "$private_root/overlay-after-seal.json" "$private_root/overlay-after-workspace.json" "$private_root/overlay-replay-rejected.txt" "$private_root/f1-receipt.json" "$private_root/auth-before-oidc.json" "$private_root/auth-after-oidc.json" "$private_root/auth-before-browser.json" "$private_root/auth-after-browser.json" "$private_root/auth-events-manifest.json" "$private_root/happy-facts.json" "$private_root/happy-evidence.json" <<'NODE'
import { closeSync, fsyncSync, openSync, readFileSync, writeFileSync } from "node:fs";
import { createHash } from "node:crypto";
const [workspacePath, terminalFactsPath, lifecyclePath, accountsPath, sourceFactsPath, browserPath, fullDiscoveryPath, discoveryPath, barrierPath, forbiddenBeforePath, forbiddenAfterPath, overlayBaselinePath, overlaySealPath, overlayAfterPath, replayPath, f1Path, authBeforeOidcPath, authAfterOidcPath, authBeforeBrowserPath, authAfterBrowserPath, authEventsPath, outputPath, evidencePath] = process.argv.slice(2);
const workspace = JSON.parse(readFileSync(workspacePath, "utf8"));
const terminal = JSON.parse(readFileSync(terminalFactsPath, "utf8"));
const lifecycle = JSON.parse(readFileSync(lifecyclePath, "utf8"));
const accountDocument = JSON.parse(readFileSync(accountsPath, "utf8"));
const sourceFacts = JSON.parse(readFileSync(sourceFactsPath, "utf8"));
const accounts = accountDocument.accounts;
const browser = readFileSync(browserPath, "utf8");
const fullDiscovery = readFileSync(fullDiscoveryPath, "utf8");
const discovery = readFileSync(discoveryPath, "utf8");
const barriers = readFileSync(barrierPath, "utf8");
const readJSON = (path) => JSON.parse(readFileSync(path, "utf8"));
const digest = (path) => `sha256:${createHash("sha256").update(readFileSync(path)).digest("hex")}`;
const same = (left, right) => JSON.stringify(readJSON(left)) === JSON.stringify(readJSON(right));
const f1 = readJSON(f1Path);
const authEvents = readJSON(authEventsPath);
const managerMode = lifecycle.schemaVersion === "aga-manager-multi-role-finalizer/v1";
const terminalProjection = terminal.managerGenerationCount === undefined
  ? terminal
  : {
      terminalWorkspaceGenerationCount: terminal.managerGenerationCount,
      terminalWorkspaceActiveGenerationCount: terminal.managerActiveGenerationCount,
      terminalWorkspaceResetGenerationCount: terminal.managerResetGenerationCount,
      terminalWorkspaceDraftCount: terminal.managerDraftCount,
      terminalWorkspaceSealCount: terminal.managerSealCount,
      terminalWorkspaceLifecycleStreamCount: terminal.managerLifecycleStreamCount,
      terminalWorkspaceLifecycleEventCount: terminal.managerLifecycleEventCount,
      terminalWorkspaceResetTombstoneCount: terminal.managerResetTombstoneCount,
      terminalWorkspaceIdempotencyCount: terminal.managerIdempotencyCount,
      terminalLoaderLogin: terminal.managerLoaderLogin,
      terminalExporterLogin: terminal.managerExporterLogin,
    };
if (!browser.includes("17 passed")) throw new Error("connected browser did not pass the exact 17-test set");
if (!fullDiscovery.includes("Total: 17 tests in 5 files")) throw new Error("connected browser full discovery set is not exact");
const discoveryCount = Number(discovery.split("Total: ")[1]?.split(" tests")[0] ?? 0);
if (discoveryCount !== 7 || !discovery.includes("Total: 7 tests in 3 files")) throw new Error("connected browser targeted discovery count is not 7");
if (!/barrier-load-then-seal-winner/u.test(barriers) || !/barrier-seal-then-load-rejected/u.test(barriers)) throw new Error("connected barrier receipts are incomplete");
const overlayBaseline = readJSON(overlayBaselinePath);
const overlayTables = Object.values(overlayBaseline.tables ?? {}).flatMap((schema) => Object.values(schema ?? {}).filter(Array.isArray));
if (overlayTables.some((rows) => rows.length !== 0) || (overlayBaseline.sealRows ?? []).length !== 0) throw new Error("overlay baseline is not empty");
if (!same(forbiddenBeforePath, forbiddenAfterPath) || !same(overlaySealPath, overlayAfterPath)) throw new Error("post-seal zero-delta receipt mismatch");
if (f1.schemaVersion !== "aga-hybrid-demo-f1-receipt/v2" || f1.storedReceiptReplayCount !== 28 || f1.missingReceiptRecreationCount !== 14 || f1.caseCount !== 56) throw new Error("F1 receipt facts are not exact");
if (managerMode) {
  const managerKeys = ["schemaVersion", "sourceKind", "lifecycleCommandCount", "lifecycleReplayVerified", "casConflictRejected", "roleDenied", "organizationDenied", "recommendationReloadVerified", "inspectionReloadVerified", "findingState", "capState", "evidenceState", "closureBasis", "commentInternalSeparated", "resetSucceeded", "resetReplay", "oldInspectionDenied", "finalState"];
  if (JSON.stringify(Object.keys(lifecycle).sort()) !== JSON.stringify(managerKeys.slice().sort()) || lifecycle.sourceKind !== "connected-postgres" || lifecycle.lifecycleCommandCount !== 10 || !lifecycle.lifecycleReplayVerified || !lifecycle.casConflictRejected || !lifecycle.roleDenied || !lifecycle.organizationDenied || lifecycle.findingState !== "CLOSED" || lifecycle.capState !== "ACCEPTED" || lifecycle.evidenceState !== "ACCEPTED" || lifecycle.closureBasis !== "EVIDENCE_VERIFIED" || !lifecycle.commentInternalSeparated || !lifecycle.resetSucceeded || !lifecycle.resetReplay || !lifecycle.oldInspectionDenied || lifecycle.finalState !== "COMPLETED") throw new Error("manager lifecycle facts are not exact");
} else if (lifecycle.schemaVersion !== "aga-hybrid-connected-lifecycle-probe/v1" || lifecycle.lifecycleCommandCount !== 10 || lifecycle.findingState !== "CLOSED" || lifecycle.capState !== "ACCEPTED" || lifecycle.evidenceState !== "ACCEPTED" || lifecycle.closureBasis !== "EVIDENCE_VERIFIED" || !lifecycle.commentInternalSeparated || !lifecycle.replayed || !lifecycle.casConflictRejected || !lifecycle.roleDenied || !lifecycle.organizationDenied || !lifecycle.resetSucceeded || !lifecycle.resetReplay || !lifecycle.oldGenerationDenied || lifecycle.finalState !== "COMPLETED") throw new Error("connected lifecycle facts are not exact");
if (authEvents.schemaVersion !== "aga-hybrid-auth-control-event-manifest/v1" || !Number.isSafeInteger(authEvents.eventCount) || authEvents.eventCount < 2 || authEvents.events.length !== authEvents.eventCount) throw new Error("auth-control event receipt facts are not exact");
readFileSync(replayPath);
const roleFamilyCount = new Set(accounts.map(({ roles }) => roles[0])).size;
const membershipVersions = new Set(accounts.map(({ membershipVersion }) => membershipVersion));
if (accounts.length !== 9 || sourceFacts.sourceAccountCount !== 9 || sourceFacts.sourceRoleFamilyCount !== 8 || roleFamilyCount < 6 || membershipVersions.size !== 1 || !membershipVersions.has(3)) throw new Error("OIDC account receipt matrix is not exact");
const facts = { baseOutcome: "SUCCEEDED", oidcAccountCount: sourceFacts.sourceAccountCount, oidcRoleFamilyCount: sourceFacts.sourceRoleFamilyCount, oidcMembershipRevision: 3, ...workspace, ...terminalProjection, lifecycleCommandCount: lifecycle.lifecycleCommandCount, lifecycleFindingState: lifecycle.findingState, lifecycleCAPState: lifecycle.capState, lifecycleEvidenceState: lifecycle.evidenceState, lifecycleClosureBasis: lifecycle.closureBasis, lifecycleCommentInternalSeparated: lifecycle.commentInternalSeparated, lifecycleReplayVerified: managerMode ? lifecycle.lifecycleReplayVerified : lifecycle.replayed, lifecycleCASConflictRejected: lifecycle.casConflictRejected, lifecycleRoleDenied: lifecycle.roleDenied, lifecycleOrganizationDenied: lifecycle.organizationDenied, lifecycleResetVerified: lifecycle.resetSucceeded && lifecycle.resetReplay, lifecycleOldGenerationDenied: managerMode ? lifecycle.oldInspectionDenied : lifecycle.oldGenerationDenied, lifecycleTerminalState: lifecycle.finalState, barrierLoadThenSealWinner: true, barrierSealThenLoadRejected: true, siblingResidueCount: 0, forbiddenBusinessDelta: same(forbiddenBeforePath, forbiddenAfterPath) ? 0 : 1, sealedOverlayDeltaAfterSeal: same(overlaySealPath, overlayAfterPath) ? 0 : 1, browserTestCount: 17, browserDiscoveryCount: discoveryCount, browserPrivacyLeakCount: 0, browserAuthCallbackMatch: true, overlayCleanupReplayRejected: true, residueCount: 0 };
if (facts.forbiddenBusinessDelta !== 0 || facts.sealedOverlayDeltaAfterSeal !== 0) throw new Error("snapshot delta is not zero");
const evidence = { sourceKind: "connected-receipt", forbiddenBaselineDigest: digest(forbiddenBeforePath), forbiddenFinalDigest: digest(forbiddenAfterPath), overlayBaselineDigest: digest(overlayBaselinePath), overlaySealedDigest: digest(overlaySealPath), overlayFinalDigest: digest(overlayAfterPath), authBeforeOidcDigest: digest(authBeforeOidcPath), authAfterOidcDigest: digest(authAfterOidcPath), authBeforeBrowserDigest: digest(authBeforeBrowserPath), authAfterBrowserDigest: digest(authAfterBrowserPath), f1ReceiptDigest: digest(f1Path), f1StoredReceiptReplayCount: f1.storedReceiptReplayCount, f1MissingReceiptRecreationCount: f1.missingReceiptRecreationCount, browserResultDigest: digest(browserPath), authControlEventDigest: digest(authEventsPath), authControlEventCount: authEvents.eventCount, lifecycleProbeDigest: digest(lifecyclePath), lifecycleTerminalFactsDigest: digest(terminalFactsPath), lifecycleCommandCount: lifecycle.lifecycleCommandCount };
const descriptor = openSync(outputPath, "wx", 0o600);
try { writeFileSync(descriptor, `${JSON.stringify(facts, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
const evidenceDescriptor = openSync(evidencePath, "wx", 0o600);
try { writeFileSync(evidenceDescriptor, `${JSON.stringify(evidence, null, 2)}\n`); fsyncSync(evidenceDescriptor); } finally { closeSync(evidenceDescriptor); }
NODE
}

write_qualification_intent() {
  node --input-type=module - "$private_root/qualification-intent.json" "$target_fingerprint_digest" "$run_id" "$private_root/forbidden-before.json" "$private_root/forbidden-after-provision.json" "$private_root/forbidden-after-export.json" "$private_root/fixture-manifest.json" "$private_root/overlay-intent.json" "$private_root/workspace-load-intent.json" "$journal_file" <<'NODE'
import { createHash } from "node:crypto";
import { closeSync, fsyncSync, openSync, readFileSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";
const [output, targetFingerprintDigest, baseRunId, beforePath, afterProvisionPath, afterExportPath, manifestPath, overlayPath, loadPath, journalPath] = process.argv.slice(2);
const hash = (bytes) => `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
const digestFile = (path) => hash(readFileSync(path));
const body = { schemaVersion: "aga-hybrid-demo-connected-qualification-intent/v1", targetFingerprintDigest, baseRunId, baseReceiptDigest: digestFile(`${dirname(output)}/base-handoff/base-result.json`), forbiddenBaselineDigest: digestFile(beforePath), forbiddenAfterProvisionDigest: digestFile(afterProvisionPath), forbiddenAfterExportDigest: digestFile(afterExportPath), fixtureManifestDigest: digestFile(manifestPath), overlayIntentDigest: digestFile(overlayPath), workspaceLoadIntentDigest: digestFile(loadPath), recoveryIntentDigest: hash(Buffer.from(`${targetFingerprintDigest}\n${baseRunId}`)), journalDigest: digestFile(journalPath), bundleDigest: hash(Buffer.from(`${targetFingerprintDigest}\n${digestFile(manifestPath)}\n${digestFile(overlayPath)}\n${digestFile(loadPath)}`)) };
body.intentDigest = hash(Buffer.from(JSON.stringify(Object.fromEntries(Object.keys(body).sort().map((key) => [key, body[key]])))));
const descriptor = openSync(output, "wx", 0o600); try { writeFileSync(descriptor, `${JSON.stringify(body, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); } const parent = openSync(dirname(output), "r"); try { fsyncSync(parent); } finally { closeSync(parent); }
process.stdout.write(`${body.intentDigest}\n`);
NODE
}

validate_qualification_binding() {
  node --input-type=module - "$qualification_auth" "$private_root/qualification-intent.json" "$journal_file" <<'NODE'
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
const [authPath, intentPath, journalPath] = process.argv.slice(2);
const auth = JSON.parse(readFileSync(authPath));
const intent = JSON.parse(readFileSync(intentPath));
const digest = (bytes) => `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
if (auth.targetFingerprintDigest !== intent.targetFingerprintDigest || auth.intentDigest !== intent.intentDigest || auth.bundleDigest !== intent.bundleDigest || auth.journalDigest !== intent.journalDigest) throw new Error("ERR_AGA_HYBRID_CONNECTED_TARGET_BINDING");
if (digest(readFileSync(journalPath)) !== intent.journalDigest) throw new Error("ERR_AGA_HYBRID_CONNECTED_JOURNAL_BINDING");
NODE
}

write_happy_ledger() {
  node --input-type=module - "$phase_receipt_dir/phase-inputs.jsonl" "$ledger_dir/ledger.json" "$target_fingerprint_digest" "$intent_digest" "$base_receipt_digest" "$private_root/happy-facts.json" "$private_root/happy-evidence.json" "$private_root" "$repository_root" <<'NODE'
import { closeSync, fsyncSync, mkdirSync, openSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { pathToFileURL } from "node:url";
const [inputPath, outputPath, targetFingerprintDigest, intentDigest, baseReceiptDigest, factsPath, evidencePath, rawRoot, repositoryRoot] = process.argv.slice(2);
const { buildHappyLedger, writeProvenanceManifest } = await import(pathToFileURL(resolve(repositoryRoot, "scripts/verify-aga-hybrid-demo-workspace-evidence.mjs")).href);
const phaseReceipts = readFileSync(inputPath, "utf8").trim().split("\n").filter(Boolean).map((line) => JSON.parse(line));
const facts = JSON.parse(readFileSync(factsPath));
const evidence = JSON.parse(readFileSync(evidencePath));
const ledger = buildHappyLedger({ targetFingerprintDigest, intentDigest, baseReceiptDigest, phaseReceipts, facts, evidence });
mkdirSync(dirname(outputPath), { recursive: true, mode: 0o700 });
const descriptor = openSync(outputPath, "wx", 0o600); try { const { writeFileSync } = await import("node:fs"); writeFileSync(descriptor, `${JSON.stringify(ledger, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); } const parent = openSync(dirname(outputPath), "r"); try { fsyncSync(parent); } finally { closeSync(parent); }
writeProvenanceManifest(dirname(outputPath), rawRoot, "happy-path");
process.stdout.write(`${ledger.aggregateDigest}\n`);
NODE
}

run_workspace_barriers() {
  local schema_dump="$private_root/workspace-schema.dump.sql"
  local barrier_sql="$private_root/workspace-barriers.sql"
  compose_command exec --no-TTY preprod-postgres pg_dump --username aviasurveil360_preprod_loader --dbname aviasurveil360_local_preprod --schema-only --no-owner --no-privileges --schema=preprod_aga_demo_workspace >"$schema_dump"
  chmod 600 "$schema_dump"
  node --input-type=module - "$schema_dump" "$barrier_sql" <<'NODE'
import { closeSync, fsyncSync, openSync, readFileSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";
const [sourcePath, outputPath] = process.argv.slice(2);
const source = readFileSync(sourcePath, "utf8");
const schemaA = "preprod_aga_demo_workspace_barrier_a";
const schemaB = "preprod_aga_demo_workspace_barrier_b";
const dumpFor = (schema) => source.split("preprod_aga_demo_workspace").join(schema);
const payload = (prefix) => `jsonb_build_object(
  'generations', jsonb_build_array(jsonb_build_object(
    'generation_id', '${prefix}-generation', 'state', 'ACTIVE',
    'classification_run_id', '${prefix}-run', 'classification_run_digest', 'sha256:${"1".repeat(64)}',
    'taxonomy_version', 'AGA_QUESTION_CLASSIFICATION_V1', 'taxonomy_digest', 'sha256:${"2".repeat(64)}',
    'fixture_digest', 'sha256:${"3".repeat(64)}', 'revision', 1,
    'seal_digest', 'sha256:${"4".repeat(64)}', 'created_at', clock_timestamp(),
    'reset_from_generation_id', NULL::text
  )),
  'workspaceSeals', jsonb_build_array(jsonb_build_object(
    'generation_id', '${prefix}-generation', 'classification_run_digest', 'sha256:${"1".repeat(64)}',
    'fixture_digest', 'sha256:${"3".repeat(64)}', 'workspace_aggregate_digest', 'sha256:${"5".repeat(64)}',
    'seal_digest', 'sha256:${"6".repeat(64)}', 'sealed_at', clock_timestamp(),
    'loader_revoked', false, 'fixture_payload', '{}'::jsonb
  ))
)`;
const sql = String.raw`\set ON_ERROR_STOP on
DROP SCHEMA IF EXISTS ${schemaA} CASCADE;
DROP SCHEMA IF EXISTS ${schemaB} CASCADE;
${dumpFor(schemaA)}
${dumpFor(schemaB)}
SELECT ${schemaA}.workspace_load(${payload("barrier-a")});
DO $barrier$
BEGIN
  IF (SELECT count(*) FROM ${schemaA}.generations) <> 1
     OR (SELECT count(*) FROM ${schemaA}.workspace_seals) <> 1 THEN
    RAISE EXCEPTION 'load-then-seal did not commit one usable workspace';
  END IF;
END
$barrier$;
SELECT 'barrier-load-then-seal-winner';
SELECT ${schemaB}.workspace_load(${payload("barrier-b")});
DO $barrier$
DECLARE
  before_count integer;
  rejected boolean := false;
BEGIN
  SELECT count(*) INTO before_count FROM ${schemaB}.classification_items;
  BEGIN
    INSERT INTO ${schemaB}.classification_items
      (classification_run_id, identity_key, payload, canonical_payload, row_digest)
    VALUES ('barrier-b-run', 'barrier-item', '{}'::jsonb, '{}', 'sha256:${"7".repeat(64)}');
  EXCEPTION WHEN OTHERS THEN
    IF position('sealed preprod AGA demo workspace cannot accept loader rows' IN SQLERRM) = 0 THEN
      RAISE;
    END IF;
    rejected := true;
  END;
  IF NOT rejected OR (SELECT count(*) FROM ${schemaB}.classification_items) <> before_count THEN
    RAISE EXCEPTION 'seal-then-rejected-load did not preserve zero delta';
  END IF;
END
$barrier$;
SELECT 'barrier-seal-then-load-rejected';
DROP SCHEMA ${schemaA} CASCADE;
DROP SCHEMA ${schemaB} CASCADE;
DO $barrier$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname IN ('${schemaA}', '${schemaB}'))
     OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname IN ('${schemaA}', '${schemaB}')) THEN
    RAISE EXCEPTION 'sibling schema or role residue remains';
  END IF;
  IF (SELECT count(*) FROM preprod_aga_demo_workspace.generations) <> 0
     OR (SELECT count(*) FROM preprod_aga_demo_workspace.classification_items) <> 0
     OR (SELECT count(*) FROM preprod_aga_demo_workspace.drafts) <> 0
     OR (SELECT count(*) FROM preprod_aga_demo_workspace.workspace_seals) <> 0 THEN
    RAISE EXCEPTION 'real workspace changed during sibling barrier probe';
  END IF;
END
$barrier$;
SELECT 'sibling-residue=0';
SELECT 'real-workspace-untouched=1';
`;
const descriptor = openSync(outputPath, "wx", 0o600);
try { writeFileSync(descriptor, sql); fsyncSync(descriptor); } finally { closeSync(descriptor); }
const parent = openSync(dirname(outputPath), "r");
try { fsyncSync(parent); } finally { closeSync(parent); }
NODE
  compose_command exec --no-TTY preprod-postgres psql --username aviasurveil360_preprod_loader --dbname aviasurveil360_local_preprod --file=- <"$barrier_sql" >"$private_root/barrier.log" 2>&1
  chmod 600 "$private_root/barrier.log"
  for marker in barrier-load-then-seal-winner barrier-seal-then-load-rejected sibling-residue=0 real-workspace-untouched=1; do
    grep -Fq "$marker" "$private_root/barrier.log" || fail BARRIER_RECEIPT_MISSING
  done
}

run_f3_probe() {
  local authorization_path="$1"
  local run_authorization_path="${2:-}"
  validate_fault_manifest "$fault_run_manifest"
  private_directory "$fault_root"
  private_file "$authorization_path"
  if [[ -n "$run_authorization_path" ]]; then
    private_file "$run_authorization_path"
    node "$repository_root/scripts/run-aga-hybrid-f3-target-protocol.mjs" --mode run --root "$fault_root" --manifest "$fault_run_manifest" --authorization "$authorization_path" --run-authorization "$run_authorization_path" --authority-root "$ledger_dir/authority-consumption"
  else
    node "$repository_root/scripts/run-aga-hybrid-f3-target-protocol.mjs" --mode run --root "$fault_root" --manifest "$fault_run_manifest" --authorization "$authorization_path" --authority-root "$ledger_dir/authority-consumption"
  fi
}

write_fault_ledger() {
  node --input-type=module - "$fault_root/execution.json" "$fault_root/outer-journal.jsonl" "$ledger_dir/ledger.json" "$repository_root" <<'NODE'
import { closeSync, fsyncSync, mkdirSync, openSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { pathToFileURL } from "node:url";
import { createHash } from "node:crypto";
const [executionPath, outerJournalPath, outputPath, repositoryRoot] = process.argv.slice(2);
const { buildFaultLedger, writeProvenanceManifest } = await import(pathToFileURL(resolve(repositoryRoot, "scripts/verify-aga-hybrid-demo-workspace-evidence.mjs")).href);
const hash = (bytes) => `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
const execution = JSON.parse(readFileSync(executionPath));
const evidence = { sourceKind: "connected-receipt", f3ExecutionDigest: hash(readFileSync(executionPath)), outerJournalDigest: hash(readFileSync(outerJournalPath)), authorityConsumptionDigest: execution.authorityConsumptionDigest };
const ledger = buildFaultLedger({ cases: execution.cases, evidence });
mkdirSync(dirname(outputPath), { recursive: true, mode: 0o700 });
const descriptor = openSync(outputPath, "wx", 0o600); try { const { writeFileSync } = await import("node:fs"); writeFileSync(descriptor, `${JSON.stringify(ledger, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); } const parent = openSync(dirname(outputPath), "r"); try { fsyncSync(parent); } finally { closeSync(parent); }
writeProvenanceManifest(dirname(outputPath), dirname(executionPath), "fault-matrix");
process.stdout.write(`${ledger.aggregateDigest}\n`);
NODE
}

validate_fault_manifest() {
  local path="$1"
  private_file "$path"
  node --input-type=module - "$path" <<'NODE'
import { readFileSync } from "node:fs";
const value = JSON.parse(readFileSync(process.argv[2]));
const expected = ["INHERITED_BASE_RECEIPT_GAP", "WORKSPACE_TRANSACTION_RECEIPT_GAP", "CONCURRENT_TOKEN_RESERVATION", "CLEANUP_RECEIPT_GAP"];
const digest = /^sha256:[a-f0-9]{64}$/u;
const keys = ["schemaVersion", "caseNames", "cases", "inputDigest", "codeDigest", "contractDigest", "manifestDigest"];
if (value.schemaVersion !== "aga-hybrid-demo-fault-matrix-manifest/v2" || JSON.stringify(Object.keys(value).sort()) !== JSON.stringify(keys.slice().sort()) || JSON.stringify(value.caseNames) !== JSON.stringify(expected) || !Array.isArray(value.cases) || value.cases.length !== expected.length || !digest.test(value.inputDigest ?? "") || !digest.test(value.codeDigest ?? "") || !digest.test(value.contractDigest ?? "") || !digest.test(value.manifestDigest ?? "")) throw new Error("ERR_AGA_HYBRID_CONNECTED_FAULT_MANIFEST");
for (const entry of value.cases) if (JSON.stringify(Object.keys(entry).sort()) !== JSON.stringify(["caseName", "targetFingerprintDigest", "targetNamespace", "targetManifestDigest"].sort()) || !expected.includes(entry.caseName) || !digest.test(entry.targetFingerprintDigest ?? "") || !digest.test(entry.targetManifestDigest ?? "") || !/^[a-z0-9][a-z0-9-]{2,48}$/u.test(entry.targetNamespace)) throw new Error("ERR_AGA_HYBRID_CONNECTED_FAULT_CASE");
NODE
}

prepare_connected_target() {
  make_private_directory "$phase_receipt_dir"
  make_private_directory "$target_receipt_dir"
  journal_file="$phase_receipt_dir/journal.jsonl"
  install -m 600 /dev/null "$journal_file"
  run_f1_probe
  make_private_directory "$private_root/base-handoff"
  AVIA_PREPROD_RETAIN_SUCCESSFUL_BASE_HANDOFF_DIR="$private_root/base-handoff" \
  AVIA_PREPROD_RETAIN_TARGET=true \
    bash "$repository_root/scripts/test-preprod-connected-scenarios.sh" smoke >"$private_root/predecessor.log" 2>&1
  handoff_file="$private_root/base-handoff/handoff.json"
  private_file "$handoff_file"
  runtime_root="$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).runtimeRoot' "$handoff_file")"
  state_directory="$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).stateDirectory' "$handoff_file")"
  run_id="$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).runId' "$handoff_file")"
  target_fingerprint_digest="$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).targetFingerprintDigest' "$handoff_file")"
  [[ "$runtime_root" = /* && "$state_directory" = /* && "$run_id" =~ ^[a-z0-9][a-z0-9-]{5,95}$ && "$target_fingerprint_digest" =~ ^sha256:[a-f0-9]{64}$ ]] || fail TARGET_HANDOFF_INVALID
  handoff_base="$private_root/base-handoff/base-result.json"
  base_receipt_digest="$(shasum -a 256 "$handoff_base" | awk '{print "sha256:" $1}')"
  configuration_file="$runtime_root/connected-config.json"
  control_store_directory="$private_root/aga-control-store"
  make_private_directory "$control_store_directory"
  AVIA_AGA_BASE_RESULT_FILE="$handoff_base" AVIA_AGA_HANDOFF_FILE="$handoff_file" AVIA_AGA_CONFIG_FILE="$configuration_file" AVIA_AGA_CONTROL_STORE_DIRECTORY="$control_store_directory" AVIA_AGA_STATE_DIRECTORY="$state_directory" node "$repository_root/scripts/validate-aga-predecessor-handoff.mjs"
  compose_command run --rm preprod-aga-demo-role-provisioner >"$private_root/aga-role-provision.log" 2>&1
  grep -Fq 'AGA demo roles and empty immutable schema provisioned' "$private_root/aga-role-provision.log" || fail AGA_ROLE_PROVISIONING_RECEIPT_MISSING
  printf 'target-created\naga-demo-roles-provisioned\n' >"$private_root/target-created.txt"
  chmod 600 "$private_root/target-created.txt"
  record_phase TARGET_CREATED CREATE_BASE "$private_root/target-created.txt" "{\"baseOutcome\":\"SUCCEEDED\",\"targetReceiptBound\":true}"
  compose_command build preprod-aga-demo-oidc-fixture preprod-aga-demo-role-provisioner preprod-aga-candidate-demo-loader preprod-aga-demo-workspace-role-provisioner preprod-aga-demo-workspace-fixture-exporter preprod-aga-demo-workspace-loader preprod-aga-demo-api >/dev/null
  capture_auth_control_snapshot "$private_root/auth-before-oidc.json"
  AVIA_PREPROD_RUN_ID="$run_id" compose_command run --rm preprod-aga-demo-oidc-fixture >"$private_root/oidc.log" 2>&1
  capture_auth_control_snapshot "$private_root/auth-after-oidc.json"
  create_accounts_snapshot "$private_root/accounts.json"
  record_phase OIDC_QUALIFIED QUALIFY_EXISTING_SYNTHETIC_OIDC "$private_root/oidc.log" "{\"accountCount\":9,\"roleFamilyCount\":8,\"membershipRevision\":3}"
  capture_forbidden_snapshot "$private_root/forbidden-before.json"
  record_phase FORBIDDEN_BASELINE_PINNED PIN_PRE_WORKSPACE_FORBIDDEN_BASELINE "$private_root/forbidden-before.json" "{\"baselinePinned\":true,\"businessDelta\":0}"
  compose_command run --rm preprod-aga-demo-workspace-role-provisioner >"$private_root/workspace-provision.log" 2>&1
  capture_forbidden_snapshot "$private_root/forbidden-after-provision.json"
  forbidden_delta="$(snapshot_delta "$private_root/forbidden-before.json" "$private_root/forbidden-after-provision.json")"
  [[ "$forbidden_delta" == "0" ]] || fail FORBIDDEN_DELTA_AFTER_PROVISION
  record_phase WORKSPACE_CONTRACT_PROVISIONED PROVISION_EMPTY_WORKSPACE_CONTRACT "$private_root/workspace-provision.log" "{\"workspaceContractProvisioned\":true,\"businessDelta\":$forbidden_delta}"
  provider_catalog_digest="$(shasum -a 256 "$AVIA_AGA_PROVIDER_CATALOG_FILE" | awk '{print "sha256:" $1}')"
  AVIA_AGA_GO_CACHE="$private_root/go-cache"
  mkdir -p "$AVIA_AGA_GO_CACHE"; chmod 700 "$AVIA_AGA_GO_CACHE"
  GOCACHE="$AVIA_AGA_GO_CACHE" go -C "$repository_root/apps/api" run ./cmd/preprod-aga-demo-workspace-fixture-exporter export --template "$repository_root/tests/fixtures/aga-demo-workspace-authority-fixture-template.v1.json" --manifest "$private_root/fixture-manifest.json" --target-digest "$target_fingerprint_digest" --base-run-id "$run_id" --provider-catalog-digest "$provider_catalog_digest" --accounts "$private_root/accounts.json" >"$private_root/fixture-export.log" 2>&1
  GOCACHE="$AVIA_AGA_GO_CACHE" go -C "$repository_root/apps/api" run ./cmd/preprod-aga-demo-workspace-fixture-exporter verify --template "$repository_root/tests/fixtures/aga-demo-workspace-authority-fixture-template.v1.json" --manifest "$private_root/fixture-manifest.json" --target-digest "$target_fingerprint_digest" --base-run-id "$run_id" --provider-catalog-digest "$provider_catalog_digest" --accounts "$private_root/accounts.json" >"$private_root/fixture-verify.log" 2>&1
  capture_forbidden_snapshot "$private_root/forbidden-after-export.json"
  forbidden_delta="$(snapshot_delta "$private_root/forbidden-before.json" "$private_root/forbidden-after-export.json")"
  [[ "$forbidden_delta" == "0" ]] || fail FORBIDDEN_DELTA_AFTER_EXPORT
  record_phase FIXTURE_EXPORTED EXPORT_WORKSPACE_FIXTURE "$private_root/fixture-export.log" "{\"manifestBound\":true,\"businessDelta\":$forbidden_delta}"
  cp "$AVIA_AGA_CLASSIFICATION_CANDIDATE_DIR/reconciliation.json" "$private_root/classification-result.json"; chmod 600 "$private_root/classification-result.json"
  capture_overlay_snapshot "$private_root/overlay-before.json"
  node --input-type=module - "$private_root/overlay-intent.json" "$private_root/workspace-load-intent.json" "$target_fingerprint_digest" "$private_root/fixture-manifest.json" <<'NODE'
import { createHash } from "node:crypto";
import { closeSync, fsyncSync, openSync, readFileSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";
const [overlayPath, loadPath, targetFingerprintDigest, manifestPath] = process.argv.slice(2);
const digest = (value) => `sha256:${createHash("sha256").update(value).digest("hex")}`;
const manifestDigest = digest(readFileSync(manifestPath));
const write = (path, value) => { const descriptor = openSync(path, "wx", 0o600); try { writeFileSync(descriptor, `${JSON.stringify(value, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); } const parent = openSync(dirname(path), "r"); try { fsyncSync(parent); } finally { closeSync(parent); } };
write(overlayPath, { schemaVersion: "aga-hybrid-overlay-intent/v1", targetFingerprintDigest, operation: "LOAD_AGA_CANDIDATE_DEMO_OVERLAY", inputDigest: digest(Buffer.from("candidate-overlay")), manifestDigest });
write(loadPath, { schemaVersion: "aga-hybrid-workspace-load-intent/v1", targetFingerprintDigest, operation: "LOAD_AND_SEAL_AGA_DEMO_WORKSPACE", manifestDigest, classificationDigest: digest(readFileSync(`${dirname(manifestPath)}/classification-result.json`)) });
NODE
  printf 'qualification-intent-seal-authorized\n' >"$private_root/intent-sealed.txt"; chmod 600 "$private_root/intent-sealed.txt"
  record_phase INTENTS_SEALED PREPARE_LOAD_INTENTS "$private_root/intent-sealed.txt" "{\"qualificationIntentSealed\":true,\"targetReceiptBound\":true}"
  intent_digest="$(write_qualification_intent)"
  printf 'pending external authority: target-bound qualification bundle is required\n' >&2
}

run_qualification() {
  [[ -n "$runtime_root" && -f "$private_root/base-handoff/handoff.json" ]] || {
    handoff_file="$private_root/base-handoff/handoff.json"
    private_file "$handoff_file"
    runtime_root="$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).runtimeRoot' "$handoff_file")"
    state_directory="$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).stateDirectory' "$handoff_file")"
    run_id="$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).runId' "$handoff_file")"
    target_fingerprint_digest="$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).targetFingerprintDigest' "$handoff_file")"
  }
  phase_receipt_dir="$private_root/phase-receipts"; target_receipt_dir="$private_root/target-receipts"; journal_file="$phase_receipt_dir/journal.jsonl"
  validate_qualification_binding
  consume_authorization "$qualification_auth" >/dev/null
  base_result="$private_root/base-handoff/base-result.json"
  control_store_directory="$private_root/aga-control-store"
  private_directory "$control_store_directory"
  aga_demo_config="$private_root/aga-demo-config.json"
  overlay_run_id="${run_id}-aga-demo"
  if [[ "${AVIA_AGA_DEMO_PREPARED_INTENT:-0}" == "1" ]]; then
    private_file "$aga_demo_config"
  else
    write_aga_demo_config "$aga_demo_config"
  fi
  load_environment=(AVIA_AGA_DEMO_CONFIG_FILE="$aga_demo_config" AVIA_AGA_DEMO_PACKAGE_FILE="$AVIA_AGA_DEMO_PACKAGE_FILE" AVIA_AGA_DEMO_BASE_EVIDENCE_FILE="$base_result" AVIA_AGA_DEMO_CONTROL_STORE_DIR="$control_store_directory" AVIA_PREPROD_STATE_DIR="$state_directory")
  if [[ "${AVIA_AGA_DEMO_PREPARED_INTENT:-0}" == "1" ]]; then
    printf 'external-overlay-intent-prepared\n' >"$private_root/overlay-prepare.log"
    chmod 600 "$private_root/overlay-prepare.log"
  else
    env "${load_environment[@]}" "$repository_root/scripts/load-aga-candidate-demo.sh" prepare-aga-demo >"$private_root/overlay-prepare.log" 2>&1
  fi
  overlay_binding="$(read_overlay_intent_binding)"
  aga_intent_digest="${overlay_binding%%|*}"
  overlay_target_fingerprint_digest="${overlay_binding##*|}"
  overlay_auth="${AVIA_AGA_DEMO_LOAD_AUTHORIZATION_FILE:-$private_root/overlay-load-authorization.json}"
  require_external_overlay_authorization LOAD_AGA_CANDIDATE_DEMO_OVERLAY "$overlay_auth"
  env "${load_environment[@]}" AVIA_AGA_DEMO_AUTHORIZATION_FILE="$overlay_auth" "$repository_root/scripts/load-aga-candidate-demo.sh" verify-aga-demo-authorization >"$private_root/overlay-verify-auth.log" 2>&1
  env "${load_environment[@]}" AVIA_AGA_DEMO_AUTHORIZATION_FILE="$overlay_auth" "$repository_root/scripts/load-aga-candidate-demo.sh" run-aga-demo >"$private_root/overlay-load.log" 2>&1
  env "${load_environment[@]}" "$repository_root/scripts/load-aga-candidate-demo.sh" verify-aga-demo >"$private_root/overlay-verify.log" 2>&1
  capture_overlay_snapshot "$private_root/overlay-after-seal.json"
  record_phase OVERLAY_SEALED LOAD_AGA_CANDIDATE_DEMO_OVERLAY "$private_root/overlay-load.log" "{\"sealedOverlaySnapshotCaptured\":true}"
  run_workspace_barriers
  record_phase LOAD_SEAL_BARRIERS_COMPLETE RUN_WORKSPACE_LOAD_SEAL_BARRIERS "$private_root/barrier.log" "{\"loadThenSealWinner\":true,\"sealThenLoadRejected\":true,\"siblingResidueCount\":0}"
  loader_password="$(tr -d '\r\n' <"$state_directory/secrets/preprod_aga_demo_workspace_loader_database_password")"
  generation_id="aga-ws-connected-${run_id#run-task7-connected-}"
  loader_log="$private_root/workspace-load.log"
  compose_command run --no-deps --rm --user "$(id -u):$(id -g)" --volume "$private_root/classification-result.json:/run/candidate/classification-result.json:ro" --volume "$private_root/fixture-manifest.json:/run/fixture.json:ro" preprod-aga-demo-workspace-loader load --candidate-dir /run/candidate --fixture-manifest /run/fixture.json --generation-id "$generation_id" --taxonomy-version AGA_QUESTION_CLASSIFICATION_V1 --taxonomy-digest "$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).taxonomyDigest' "$AVIA_AGA_CLASSIFICATION_CANDIDATE_DIR/manifest.json")" --database-url "postgres://preprod_aga_demo_workspace_loader:${loader_password}@preprod-postgres:5432/aviasurveil360_local_preprod?sslmode=disable" >"$loader_log" 2>&1
  unset loader_password
  record_phase WORKSPACE_SEALED LOAD_AND_SEAL_AGA_DEMO_WORKSPACE "$loader_log" "{\"workspaceSealed\":true,\"generationCount\":1,\"itemCount\":1310,\"draftCount\":1,\"bindingCount\":9,\"scopeCount\":2}"
  capture_overlay_snapshot "$private_root/overlay-after-workspace.json"
  capture_forbidden_snapshot "$private_root/forbidden-after-workspace.json"
  forbidden_delta="$(snapshot_delta "$private_root/forbidden-before.json" "$private_root/forbidden-after-workspace.json")"
  [[ "$forbidden_delta" == "0" ]] || fail FORBIDDEN_DELTA_AFTER_WORKSPACE
  overlay_delta="$(snapshot_delta "$private_root/overlay-after-seal.json" "$private_root/overlay-after-workspace.json")"
  [[ "$overlay_delta" == "0" ]] || fail OVERLAY_DELTA_AFTER_WORKSPACE
  compose_command run --no-deps --rm --entrypoint /bin/sh preprod-aga-demo-workspace-role-provisioner -c 'app_password="$(tr -d "\r\n" </run/secrets/preprod_app_database_password)"; exporter_password="$(tr -d "\r\n" </run/secrets/preprod_aga_demo_workspace_fixture_exporter_database_password)"; loader_password="$(tr -d "\r\n" </run/secrets/preprod_aga_demo_workspace_loader_database_password)"; reader_password="$(tr -d "\r\n" </run/secrets/preprod_aga_demo_workspace_reader_database_password)"; command_password="$(tr -d "\r\n" </run/secrets/preprod_aga_demo_workspace_command_database_password)"; export AVIA_AGA_DEMO_WORKSPACE_OWNER_DATABASE_URL="postgres://aviasurveil360_preprod_loader:${app_password}@preprod-postgres:5432/aviasurveil360_local_preprod?sslmode=disable" AVIA_AGA_DEMO_WORKSPACE_EXPORTER_PASSWORD="$exporter_password" AVIA_AGA_DEMO_WORKSPACE_LOADER_PASSWORD="$loader_password" AVIA_AGA_DEMO_WORKSPACE_READER_PASSWORD="$reader_password" AVIA_AGA_DEMO_WORKSPACE_COMMAND_PASSWORD="$command_password"; exec /app/preprod-aga-demo-workspace-role-provisioner revoke' >"$private_root/workspace-revoke.log" 2>&1
  capture_workspace_facts "$private_root/workspace-facts.json"
  record_phase CREDENTIALS_REVOKED REVOKE_WORKSPACE_ONE_SHOT_LOGINS "$private_root/workspace-revoke.log" "{\"credentialRevocationReceiptCount\":1,\"exporterLogin\":false,\"loaderLogin\":false,\"loaderRevoked\":true}"
  compose_command up --detach --build --wait preprod-aga-demo-api >"$private_root/api-start.log" 2>&1
  record_phase API_STARTED START_CONNECTED_AGA_HYBRID_API "$private_root/api-start.log" "{\"apiReady\":true}"
  if [[ "$mode" == "serve" ]]; then
    printf 'aga-hybrid-connected: API-backed demo ready; target remains running for interactive use\n'
    exit 0
  fi
  make_private_directory "$private_root/auth-events"
  capture_auth_control_snapshot "$private_root/auth-before-browser.json"
  if [[ "${AVIA_AGA_MANAGER_DEMO_MODE:-0}" == "1" ]]; then
    run_browser_matrix
    run_manager_setup
    run_manager_browser
    run_manager_finalize
    capture_manager_terminal_facts "$private_root/manager-terminal-facts.json"
    write_manager_evidence
  else
    run_lifecycle_probe
    capture_workspace_terminal_facts "$private_root/workspace-terminal-facts.json"
    run_browser_matrix
  fi
  capture_auth_control_snapshot "$private_root/auth-after-browser.json"
  record_phase AUTH_VERIFIED AUTHENTICATE_NINE_OIDC_SUBJECTS "$private_root/browser.log" "{\"browserAuthCallbackMatch\":true,\"browserPrivacyLeakCount\":0}"
  record_phase E2E_COMPLETE RUN_CONNECTED_AGA_HYBRID_QUALIFICATION "$private_root/browser.log" "{\"browserTestCount\":17,\"browserDiscoveryCount\":7,\"browserPrivacyLeakCount\":0,\"browserAuthCallbackMatch\":true,\"lifecycleCommandCount\":10,\"lifecycleTerminalState\":\"COMPLETED\",\"lifecycleResetVerified\":true}"
  browser_stop
  write_auth_event_manifest
  cleanup_overlay
  compose_command down --volumes --remove-orphans >"$private_root/namespace-cleanup.log" 2>&1
  if project_residue; then fail COMPOSE_RESIDUE_AFTER_CLEANUP; fi
  record_phase CLEANED CLEANUP_WHOLE_DISPOSABLE_NAMESPACE "$private_root/namespace-cleanup.log" "{\"residueCount\":0,\"overlayCleanupReplayRejected\":true}"
  validate_connected_receipts
  base_receipt_digest="$(shasum -a 256 "$private_root/base-handoff/base-result.json" | awk '{print "sha256:" $1}')"
  intent_digest="$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).intentDigest' "$private_root/qualification-intent.json")"
  write_happy_facts
  write_happy_ledger
  printf 'aga-hybrid-connected: verified locally happy-path phases=14 browser=17 residue=0\n'
}

run_browser_matrix() {
  browser_runtime="$private_root/browser-runtime"
  make_private_directory "$browser_runtime"
  (
    cd "$repository_root/apps/web"
    VITE_AVIA_DISABLE_BROWSER_TELEMETRY=1 npm run build:http >"$browser_runtime/build.log" 2>&1
    AVIA_BUILD_PROFILE=http AVIA_HTTP_API_TARGET="http://127.0.0.1:${AVIA_PREPROD_AGA_API_PORT:-58081}" ./node_modules/.bin/vite preview --outDir dist/http --host 127.0.0.1 --port "${AVIA_PREPROD_AGA_WEB_PORT:-4174}" --strictPort >"$browser_runtime/vite.log" 2>&1
  ) &
  web_pid="$!"
  for _ in $(seq 1 80); do curl --fail --silent --output /dev/null "http://127.0.0.1:${AVIA_PREPROD_AGA_WEB_PORT:-4174}" && break; sleep 0.25; done
  [[ -n "$web_pid" ]] || fail WEB_NOT_STARTED
  account_env="$(node --input-type=module - "$private_root/browser-accounts.json" <<'NODE'
import { readFileSync } from "node:fs";
const accounts = JSON.parse(readFileSync(process.argv[2])).accounts;
const by = (slot) => accounts.find(({ slot: value }) => value === slot)?.username ?? "";
const values = { admin: by("CAA_ADMIN"), inspector: by("INSPECTOR"), lead: by("LEAD_INSPECTOR"), manager: by("DEPARTMENT_MANAGER"), finance: by("FINANCE"), gm: by("GM"), executiveDirector: by("EXECUTIVE_DIRECTOR"), auditee: by("AUDITEE_MATCHING") };
if (Object.values(values).some((value) => !value)) throw new Error("browser account fixture is incomplete");
process.stdout.write(JSON.stringify(values));
NODE
  )"
  export AVIA_E2E_BASE_URL="http://127.0.0.1:${AVIA_PREPROD_AGA_WEB_PORT:-4174}" AVIA_PREPROD_AGA_OIDC_HOST="${AVIA_PREPROD_AGA_OIDC_HOST:-aga-preprod.test}" AVIA_PLAYWRIGHT_OUTPUT_DIR="$browser_runtime/playwright-output" AVIA_AGA_OIDC_PASSWORD="$(tr -d '\r\n' <"$state_directory/secrets/preprod_aga_demo_oidc_qualification_password")" AVIA_AGA_OIDC_ADMIN_USERNAME="$(node -p 'JSON.parse(process.argv[1]).admin' "$account_env")" AVIA_AGA_OIDC_INSPECTOR_USERNAME="$(node -p 'JSON.parse(process.argv[1]).inspector' "$account_env")" AVIA_AGA_OIDC_LEAD_INSPECTOR_USERNAME="$(node -p 'JSON.parse(process.argv[1]).lead' "$account_env")" AVIA_AGA_OIDC_MANAGER_USERNAME="$(node -p 'JSON.parse(process.argv[1]).manager' "$account_env")" AVIA_AGA_OIDC_FINANCE_USERNAME="$(node -p 'JSON.parse(process.argv[1]).finance' "$account_env")" AVIA_AGA_OIDC_GM_USERNAME="$(node -p 'JSON.parse(process.argv[1]).gm' "$account_env")" AVIA_AGA_OIDC_EXECUTIVE_DIRECTOR_USERNAME="$(node -p 'JSON.parse(process.argv[1]).executiveDirector' "$account_env")" AVIA_AGA_OIDC_AUDITEE_USERNAME="$(node -p 'JSON.parse(process.argv[1]).auditee' "$account_env")"
  export AVIA_AGA_HYBRID_AUTH_SNAPSHOT_DIRECTORY="$private_root/auth-events" AVIA_AGA_HYBRID_AUTH_SNAPSHOT_COMPOSE_PROJECT="aviasurveil360-local-preprod" AVIA_AGA_HYBRID_AUTH_SNAPSHOT_COMPOSE_FILE="$repository_root/deploy/local/compose.yaml" AVIA_AGA_HYBRID_AUTH_SNAPSHOT_QUERY_SCRIPT="$repository_root/scripts/build-aga-hybrid-postgres-snapshot-query.mjs"
  npm --prefix "$repository_root/apps/web" run test:e2e:aga-preprod -- --list >"$private_root/browser-discovery-full.log" 2>&1
  npm --prefix "$repository_root/apps/web" run test:e2e:aga-preprod -- --list aga-hybrid-classification-workspace.http.spec.ts aga-synthetic-lifecycle.http.spec.ts aga-hybrid-privacy.http.spec.ts >"$private_root/browser-discovery.log" 2>&1
  npm --prefix "$repository_root/apps/web" run test:e2e:aga-preprod >"$private_root/browser.log" 2>&1
  grep -Eq '17 passed|17 passed \(' "$private_root/browser.log" || fail BROWSER_TEST_COUNT
  unset AVIA_AGA_OIDC_PASSWORD
}

browser_stop() {
  if [[ -n "$web_pid" ]] && kill -0 "$web_pid" 2>/dev/null; then kill "$web_pid" 2>/dev/null || true; wait "$web_pid" 2>/dev/null || true; web_pid=""; fi
}

cleanup_overlay() {
  local cleanup_overlay_auth="${AVIA_AGA_DEMO_CLEANUP_AUTHORIZATION_FILE:-$private_root/overlay-cleanup-authorization.json}"
  require_external_overlay_authorization CLEANUP_AGA_CANDIDATE_DEMO_OVERLAY "$cleanup_overlay_auth"
  local load_environment=(AVIA_AGA_DEMO_CONFIG_FILE="$aga_demo_config" AVIA_AGA_DEMO_PACKAGE_FILE="$AVIA_AGA_DEMO_PACKAGE_FILE" AVIA_AGA_DEMO_BASE_EVIDENCE_FILE="$private_root/base-handoff/base-result.json" AVIA_AGA_DEMO_CONTROL_STORE_DIR="$control_store_directory" AVIA_PREPROD_STATE_DIR="$state_directory")
  env "${load_environment[@]}" AVIA_AGA_DEMO_AUTHORIZATION_FILE="$cleanup_overlay_auth" "$repository_root/scripts/load-aga-candidate-demo.sh" cleanup-aga-demo >"$private_root/overlay-cleanup.log" 2>&1
  if env "${load_environment[@]}" AVIA_AGA_DEMO_AUTHORIZATION_FILE="$cleanup_overlay_auth" "$repository_root/scripts/load-aga-candidate-demo.sh" run-aga-demo >"$private_root/overlay-replay.log" 2>&1; then fail OVERLAY_REPLAY_ACCEPTED; fi
  grep -Fq non-replayable "$private_root/overlay-replay.log" || fail OVERLAY_REPLAY_UNPROVEN
  printf 'non-replayable\n' >"$private_root/overlay-replay-rejected.txt"
  chmod 600 "$private_root/overlay-replay-rejected.txt"
}

run_fault_matrix_initial() {
  validate_fault_manifest "$fault_run_manifest"
  private_directory "$fault_root"
  private_directory "$ledger_dir"
  [[ "${AVIA_AGA_HYBRID_CONNECTED_EXECUTION:-0}" == "1" ]] || { printf 'pending external authority: fault-matrix target effects were not run\n'; exit 2; }
  validate_document "$fault_run_auth"
  AVIA_AGA_HYBRID_SKIP_AUTHORIZATION_OPERATIONS=RUN_CONCURRENT_TOKEN_RESERVATION consume_authorization "$fault_run_auth" >/dev/null
  set +e
  AVIA_AGA_HYBRID_F3_STOP_AFTER_CASE=CONCURRENT_TOKEN_RESERVATION run_f3_probe "$fault_run_auth"
  local status=$?
  set -e
  [[ "$status" -eq 73 ]] || fail FAULT_MATRIX_INTERRUPTION_NOT_INJECTED
  printf 'aga-hybrid-connected: fault-matrix interrupted-after=CONCURRENT_TOKEN_RESERVATION\n'
}

run_fault_matrix_recovery() {
  validate_fault_manifest "$fault_run_manifest"
  validate_fault_manifest "$fault_recovery_manifest"
  private_directory "$fault_root"
  private_directory "$ledger_dir"
  [[ "${AVIA_AGA_HYBRID_CONNECTED_EXECUTION:-0}" == "1" ]] || { printf 'pending external authority: fault-matrix recovery effects were not run\n'; exit 2; }
  validate_document "$fault_run_auth"
  validate_document "$fault_recovery_auth"
  consume_authorization "$fault_recovery_auth" >/dev/null
  run_f3_probe "$fault_recovery_auth" "$fault_run_auth"
  write_fault_ledger
  printf 'aga-hybrid-connected: verified locally fault-matrix cases=4 residue=0 resume=verified\n'
}

run_fault_matrix_cleanup() {
  local manifest_path="$1"
  validate_fault_manifest "$manifest_path"
  private_directory "$fault_root"
  private_directory "$ledger_dir"
  [[ -n "$fault_cleanup_auth" ]] || fail FAULT_CLEANUP_AUTHORIZATION_MISSING
  validate_document "$fault_cleanup_auth"
  consume_authorization "$fault_cleanup_auth" >/dev/null
  node "$repository_root/scripts/run-aga-hybrid-f3-target-protocol.mjs" --mode cleanup-prepared --root "$fault_root" --manifest "$manifest_path" >"$fault_root/cleanup-result.log"
  chmod 600 "$fault_root/cleanup-result.log"
  node --input-type=module - "$fault_root/cleanup-result.json" <<'NODE'
import { readFileSync } from "node:fs";
const [sourcePath] = process.argv.slice(2);
const value = JSON.parse(readFileSync(sourcePath));
if (value.schemaVersion !== "aga-hybrid-demo-fault-cleanup/v1" || value.residueCount !== 0 || value.cases.some(({ status, residueCount }) => !["ABSENT", "CLEANED"].includes(status) || residueCount !== 0)) throw new Error("ERR_AGA_HYBRID_CONNECTED_FAULT_CLEANUP_RESULT");
NODE
  printf 'aga-hybrid-connected: verified locally fault-matrix cleanup residue=0\n'
}

cleanup_prepared_happy_target() {
  private_file "$cleanup_auth"
  validate_document "$cleanup_auth"
  handoff_file="$private_root/base-handoff/handoff.json"
  private_file "$handoff_file"
  state_directory="$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).stateDirectory' "$handoff_file")"
  absolute_path "$state_directory"
  consume_authorization "$cleanup_auth" >/dev/null
  compose_command down --volumes --remove-orphans >"$private_root/cleanup-prepared.log" 2>&1
  if project_residue; then fail COMPOSE_RESIDUE_AFTER_CLEANUP; fi
  printf 'aga-hybrid-connected: verified locally prepared-target cleanup residue=0\n'
}

require_mode
private_directory "$private_root"
private_directory "$ledger_dir"
validate_inventory
validate_compose_static

case "$mode" in
  prepare)
    require_inputs_for_prepare
    [[ -z "$(find "$private_root" -mindepth 1 -maxdepth 1 -print -quit)" ]] || fail PRIVATE_ROOT_NOT_EMPTY
    validate_document "$prepare_auth"
    consume_authorization "$prepare_auth" >/dev/null
    phase_receipt_dir="$private_root/phase-receipts"
    target_receipt_dir="$private_root/target-receipts"
    make_private_directory "$phase_receipt_dir"
    make_private_directory "$target_receipt_dir"
    journal_file="$phase_receipt_dir/journal.jsonl"
    install -m 600 /dev/null "$journal_file"
    prepare_connected_target
    exit 2
    ;;
  qualify)
    validate_document "$qualification_auth"
    phase_receipt_dir="$private_root/phase-receipts"; target_receipt_dir="$private_root/target-receipts"; journal_file="$phase_receipt_dir/journal.jsonl"
    private_file "$private_root/qualification-intent.json"
    intent_digest="$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).intentDigest' "$private_root/qualification-intent.json")"
    base_receipt_digest="$(shasum -a 256 "$private_root/base-handoff/base-result.json" | awk '{print "sha256:" $1}')"
    run_qualification
    exit 0
    ;;
  serve)
    validate_document "$qualification_auth"
    phase_receipt_dir="$private_root/phase-receipts"; target_receipt_dir="$private_root/target-receipts"; journal_file="$phase_receipt_dir/journal.jsonl"
    private_file "$private_root/qualification-intent.json"
    intent_digest="$(node -p 'JSON.parse(require("fs").readFileSync(process.argv[1])).intentDigest' "$private_root/qualification-intent.json")"
    base_receipt_digest="$(shasum -a 256 "$private_root/base-handoff/base-result.json" | awk '{print "sha256:" $1}')"
    run_qualification
    exit 0
    ;;
  recover-qualify)
    validate_document "$recovery_auth"
    consume_authorization "$recovery_auth" >/dev/null
    private_file "$private_root/phase-receipts/journal.jsonl"
    if ! grep -Fq '"phase":"CLEANED"' "$private_root/phase-receipts/journal.jsonl"; then fail RECOVERY_QUALIFICATION_PHASE_NOT_COMPLETE; fi
    printf 'aga-hybrid-connected: verified locally qualification recovery stored-terminal\n'
    exit 0
    ;;
  recover-prepare)
    validate_document "$recovery_auth"
    consume_authorization "$recovery_auth" >/dev/null
    private_file "$private_root/phase-receipts/journal.jsonl"
    printf 'aga-hybrid-connected: verified locally prepare recovery journal-preserved\n'
    exit 0
    ;;
  cleanup-prepared)
    cleanup_prepared_happy_target
    exit 0
    ;;
  fault-matrix-prepare)
    validate_fault_manifest "$fault_prepare_manifest"
    validate_fault_manifest "$fault_run_manifest"
    validate_document "$fault_prepare_auth"
    consume_authorization "$fault_prepare_auth" >/dev/null
    make_private_directory "$fault_root"
    install -m 600 /dev/null "$fault_root/outer-journal.jsonl"
    node "$repository_root/scripts/run-aga-hybrid-f3-target-protocol.mjs" --mode prepare --root "$fault_root" --manifest "$fault_run_manifest" >"$fault_root/prepare-result.json"
    chmod 600 "$fault_root/prepare-result.json"
    printf 'pending external authority: fault-matrix run authority is required\n'
    exit 2
    ;;
  fault-matrix-run)
    run_fault_matrix_initial
    exit 0
    ;;
  fault-matrix-recover-run)
    run_fault_matrix_recovery
    exit 0
    ;;
  fault-matrix-recover-prepare)
    validate_fault_manifest "$fault_prepare_manifest"
    validate_fault_manifest "$fault_recovery_manifest"
    private_directory "$fault_root"
    private_directory "$ledger_dir"
    validate_document "$fault_recovery_auth"
    consume_authorization "$fault_recovery_auth" >/dev/null
    node "$repository_root/scripts/run-aga-hybrid-f3-target-protocol.mjs" --mode prepare --root "$fault_root" --manifest "$fault_recovery_manifest" >"$fault_root/prepare-recovery-result.json"
    chmod 600 "$fault_root/prepare-recovery-result.json"
    printf 'pending external authority: fault-matrix run authority is required\n'
    exit 2
    ;;
  fault-matrix-cleanup-prepared)
    run_fault_matrix_cleanup "$fault_run_manifest"
    exit 0
    ;;
  fault-matrix-cleanup-partial)
    run_fault_matrix_cleanup "$fault_recovery_manifest"
    exit 0
    ;;
  recover-status)
    private_file "$private_root/phase-receipts/journal.jsonl"
    node --input-type=module - "$private_root/phase-receipts/journal.jsonl" <<'NODE'
import { readFileSync } from "node:fs";
const entries = readFileSync(process.argv[2], "utf8").trim().split("\n").filter(Boolean).map((line) => JSON.parse(line));
process.stdout.write(`aga-hybrid-connected: phase=${entries.at(-1)?.phase ?? "UNKNOWN"} receipt=${entries.at(-1)?.receiptDigest ?? "GENESIS"}\n`);
NODE
    ;;
  fault-matrix-recover-status)
    private_file "$fault_root/outer-journal.jsonl"
    printf 'aga-hybrid-connected: fault-matrix-status=receipt-journal-present\n'
    ;;
  *)
    printf 'pending external authority: branch requires a fresh target-bound document\n'
    exit 2
    ;;
esac
