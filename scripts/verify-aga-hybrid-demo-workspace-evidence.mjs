#!/usr/bin/env node

import { createHash } from "node:crypto";
import {
  closeSync,
  copyFileSync,
  existsSync,
  fsyncSync,
  lstatSync,
  mkdirSync,
  openSync,
  readdirSync,
  readFileSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
export const HAPPY_PHASES = Object.freeze([
  "TARGET_CREATED",
  "OIDC_QUALIFIED",
  "FORBIDDEN_BASELINE_PINNED",
  "WORKSPACE_CONTRACT_PROVISIONED",
  "FIXTURE_EXPORTED",
  "INTENTS_SEALED",
  "OVERLAY_SEALED",
  "LOAD_SEAL_BARRIERS_COMPLETE",
  "WORKSPACE_SEALED",
  "CREDENTIALS_REVOKED",
  "API_STARTED",
  "AUTH_VERIFIED",
  "E2E_COMPLETE",
  "CLEANED",
]);
export const FAULT_CASES = Object.freeze([
  "INHERITED_BASE_RECEIPT_GAP",
  "WORKSPACE_TRANSACTION_RECEIPT_GAP",
  "CONCURRENT_TOKEN_RESERVATION",
  "CLEANUP_RECEIPT_GAP",
]);
const DIGEST = /^sha256:[a-f0-9]{64}$/u;
const HAPPY_KEYS = Object.freeze([
  "baseOutcome",
  "oidcAccountCount",
  "oidcRoleFamilyCount",
  "oidcMembershipRevision",
  "workspaceGenerationCount",
  "workspaceItemCount",
  "workspaceDraftCount",
  "workspaceBindingCount",
  "workspaceScopeCount",
  "credentialRevocationReceiptCount",
  "exporterLogin",
  "loaderLogin",
  "loaderRevoked",
  "terminalWorkspaceGenerationCount",
  "terminalWorkspaceActiveGenerationCount",
  "terminalWorkspaceResetGenerationCount",
  "terminalWorkspaceDraftCount",
  "terminalWorkspaceSealCount",
  "terminalWorkspaceLifecycleStreamCount",
  "terminalWorkspaceLifecycleEventCount",
  "terminalWorkspaceResetTombstoneCount",
  "terminalWorkspaceIdempotencyCount",
  "terminalLoaderLogin",
  "terminalExporterLogin",
  "lifecycleCommandCount",
  "lifecycleFindingState",
  "lifecycleCAPState",
  "lifecycleEvidenceState",
  "lifecycleClosureBasis",
  "lifecycleCommentInternalSeparated",
  "lifecycleReplayVerified",
  "lifecycleCASConflictRejected",
  "lifecycleRoleDenied",
  "lifecycleOrganizationDenied",
  "lifecycleResetVerified",
  "lifecycleOldGenerationDenied",
  "lifecycleTerminalState",
  "barrierLoadThenSealWinner",
  "barrierSealThenLoadRejected",
  "siblingResidueCount",
  "forbiddenBusinessDelta",
  "sealedOverlayDeltaAfterSeal",
  "browserTestCount",
  "browserDiscoveryCount",
  "browserPrivacyLeakCount",
  "browserAuthCallbackMatch",
  "overlayCleanupReplayRejected",
  "residueCount",
]);
const EXPECTED_HAPPY_FACTS = Object.freeze({
  baseOutcome: "SUCCEEDED",
  oidcAccountCount: 9,
  oidcRoleFamilyCount: 8,
  oidcMembershipRevision: 3,
  workspaceGenerationCount: 1,
  workspaceItemCount: 1310,
  workspaceDraftCount: 1,
  workspaceBindingCount: 9,
  workspaceScopeCount: 2,
  credentialRevocationReceiptCount: 1,
  exporterLogin: false,
  loaderLogin: false,
  loaderRevoked: true,
  terminalWorkspaceGenerationCount: 2,
  terminalWorkspaceActiveGenerationCount: 1,
  terminalWorkspaceResetGenerationCount: 1,
  terminalWorkspaceSealCount: 2,
  terminalWorkspaceLifecycleStreamCount: 1,
  terminalWorkspaceLifecycleEventCount: 10,
  terminalWorkspaceResetTombstoneCount: 1,
  terminalLoaderLogin: false,
  terminalExporterLogin: false,
  lifecycleCommandCount: 10,
  lifecycleFindingState: "CLOSED",
  lifecycleCAPState: "ACCEPTED",
  lifecycleEvidenceState: "ACCEPTED",
  lifecycleClosureBasis: "EVIDENCE_VERIFIED",
  lifecycleCommentInternalSeparated: true,
  lifecycleReplayVerified: true,
  lifecycleCASConflictRejected: true,
  lifecycleRoleDenied: true,
  lifecycleOrganizationDenied: true,
  lifecycleResetVerified: true,
  lifecycleOldGenerationDenied: true,
  lifecycleTerminalState: "COMPLETED",
  barrierLoadThenSealWinner: true,
  barrierSealThenLoadRejected: true,
  siblingResidueCount: 0,
  forbiddenBusinessDelta: 0,
  sealedOverlayDeltaAfterSeal: 0,
  browserTestCount: 17,
  browserDiscoveryCount: 7,
  browserPrivacyLeakCount: 0,
  browserAuthCallbackMatch: true,
  overlayCleanupReplayRejected: true,
  residueCount: 0,
});
const HAPPY_EVIDENCE_KEYS = Object.freeze([
  "sourceKind",
  "forbiddenBaselineDigest",
  "forbiddenFinalDigest",
  "overlayBaselineDigest",
  "overlaySealedDigest",
  "overlayFinalDigest",
  "authBeforeOidcDigest",
  "authAfterOidcDigest",
  "authBeforeBrowserDigest",
  "authAfterBrowserDigest",
  "f1ReceiptDigest",
  "f1StoredReceiptReplayCount",
  "f1MissingReceiptRecreationCount",
  "browserResultDigest",
  "authControlEventDigest",
  "authControlEventCount",
  "lifecycleProbeDigest",
  "lifecycleTerminalFactsDigest",
  "lifecycleCommandCount",
]);
const FAULT_EVIDENCE_KEYS = Object.freeze(["sourceKind", "f3ExecutionDigest", "outerJournalDigest", "authorityConsumptionDigest"]);

function hash(value) {
  return `sha256:${createHash("sha256").update(value).digest("hex")}`;
}

export function digestBytes(bytes) {
  return hash(bytes);
}

export function stable(value) {
  if (Array.isArray(value)) return value.map(stable);
  if (value && typeof value === "object") {
    return Object.fromEntries(Object.keys(value).sort().map((key) => [key, stable(value[key])]));
  }
  return value;
}

function fail(code, detail = "") {
  throw new Error(`ERR_AGA_HYBRID_EVIDENCE_${code}${detail ? ` ${detail}` : ""}`);
}

function exactKeys(value, keys, code) {
  if (JSON.stringify(Object.keys(value ?? {}).sort()) !== JSON.stringify([...keys].sort())) fail(code);
}

function validateDigest(value, code) {
  if (!DIGEST.test(value ?? "")) fail(code);
}

function privacyScan(value, path = []) {
  if (Array.isArray(value)) {
    value.forEach((entry, index) => privacyScan(entry, [...path, String(index)]));
    return;
  }
  if (!value || typeof value !== "object") {
    if (typeof value !== "string") return;
    if (/(?:^|[\\/])(Users|private|tmp)(?:[\\/]|$)/u.test(value)) fail("SENSITIVE_TOKEN", path.join("."));
    if (/(?:question(?:Key|Text)?|originalText|subject|membership|proposalId|inspectionId|findingId|capId|evidenceId|authorizationId|targetId|draftId|generationId|providerScopeId)/iu.test(value)) {
      if (!FAULT_CASES.includes(value)) fail("SENSITIVE_TOKEN", path.join("."));
    }
    return;
  }
  for (const [key, entry] of Object.entries(value)) {
    if (/(?:^|_)(?:path|root|directory|file|log|secret)(?:$|_)/iu.test(key) || /(?:Path|Root|Directory|File|Log|Secret)$/u.test(key)) fail("SENSITIVE_FIELD", [...path, key].join("."));
    if (/^(?:subject|membership|question|originalText|proposalId|inspectionId|findingId|capId|evidenceId|authorizationId|targetId|draftId|generationId|providerScopeId)(?:Id|Text)?$/iu.test(key)) fail("SENSITIVE_FIELD", [...path, key].join("."));
    privacyScan(entry, [...path, key]);
  }
}

function privacyScanRawFile(name, bytes) {
  const text = bytes.toString("utf8");
  if (/(?:\/Users\/|\/private\/tmp\/|\/var\/folders\/|Bearer\s+[A-Za-z0-9._-]+|password\s*[:=])/iu.test(text)) fail("RAW_SENSITIVE_TOKEN", name);
  const rawSensitiveScan = (value, path = []) => {
    if (Array.isArray(value)) {
      value.forEach((entry, index) => rawSensitiveScan(entry, [...path, String(index)]));
      return;
    }
    if (!value || typeof value !== "object") return;
    for (const [key, entry] of Object.entries(value)) {
      if (/(?:secret|password|privateKey|accessToken)/iu.test(key)) fail("RAW_SENSITIVE_FIELD", [...path, key].join("."));
      rawSensitiveScan(entry, [...path, key]);
    }
  };
  if (/\.jsonl$/u.test(name)) {
    text.trim().split("\n").filter(Boolean).forEach((line, index) => rawSensitiveScan(JSON.parse(line), [name, String(index)]));
  } else if (/\.json$/u.test(name)) {
    rawSensitiveScan(JSON.parse(text), [name]);
  }
}

function validateHappyFacts(facts) {
  exactKeys(facts, HAPPY_KEYS, "HAPPY_FACT_KEYS");
  for (const [key, expected] of Object.entries(EXPECTED_HAPPY_FACTS)) if (facts[key] !== expected) fail("HAPPY_FACTS", key);
  if (facts.workspaceDraftCount !== 1 || !Number.isSafeInteger(facts.terminalWorkspaceDraftCount) || facts.terminalWorkspaceDraftCount < 2 || !Number.isSafeInteger(facts.terminalWorkspaceIdempotencyCount) || facts.terminalWorkspaceIdempotencyCount < 10) fail("HAPPY_WORKSPACE_FACTS");
}

function validateHappyEvidence(evidence) {
  exactKeys(evidence, HAPPY_EVIDENCE_KEYS, "HAPPY_EVIDENCE_KEYS");
  if (!["connected-receipt", "synthetic"].includes(evidence.sourceKind)) fail("HAPPY_EVIDENCE_SOURCE");
  for (const field of HAPPY_EVIDENCE_KEYS.filter((key) => key.endsWith("Digest"))) validateDigest(evidence[field], `HAPPY_EVIDENCE_${field}`);
  if (evidence.f1StoredReceiptReplayCount !== 28 || evidence.f1MissingReceiptRecreationCount !== 14) fail("HAPPY_EVIDENCE_F1_COUNTS");
  if (!Number.isSafeInteger(evidence.authControlEventCount) || evidence.authControlEventCount < 2) fail("HAPPY_EVIDENCE_AUTH_EVENTS");
  if (evidence.lifecycleCommandCount !== 10) fail("HAPPY_EVIDENCE_LIFECYCLE");
}

function validateFaultEvidence(evidence) {
  exactKeys(evidence, FAULT_EVIDENCE_KEYS, "FAULT_EVIDENCE_KEYS");
  if (!["connected-receipt", "synthetic"].includes(evidence.sourceKind)) fail("FAULT_EVIDENCE_SOURCE");
  for (const field of FAULT_EVIDENCE_KEYS.filter((key) => key.endsWith("Digest"))) validateDigest(evidence[field], `FAULT_EVIDENCE_${field}`);
}

function phaseReceiptPayload(entry) {
  return {
    previousDigest: entry.previousDigest,
    phase: entry.phase,
    status: entry.status,
    targetFingerprintDigest: entry.targetFingerprintDigest,
    targetReceiptDigest: entry.targetReceiptDigest,
    effectDigest: entry.effectDigest,
    factsDigest: entry.factsDigest,
  };
}

function validatePhaseJournal(ledger) {
  if (!Array.isArray(ledger.phaseJournal) || JSON.stringify(ledger.phaseJournal.map(({ phase }) => phase)) !== JSON.stringify(HAPPY_PHASES)) fail("HAPPY_PHASE_SET");
  let previousDigest = "GENESIS";
  for (const entry of ledger.phaseJournal) {
    exactKeys(entry, ["phase", "status", "targetFingerprintDigest", "targetReceiptDigest", "effectDigest", "factsDigest", "previousDigest", "receiptDigest"], "PHASE_ENTRY_CLOSED");
    if (entry.status !== "COMPLETED" || entry.previousDigest !== previousDigest) fail("PHASE_RECEIPT_CHAIN");
    if (entry.targetFingerprintDigest !== ledger.targetFingerprintDigest) fail("PHASE_TARGET_BINDING");
    for (const [field, code] of [["targetFingerprintDigest", "PHASE_TARGET_DIGEST"], ["targetReceiptDigest", "PHASE_TARGET_RECEIPT"], ["effectDigest", "PHASE_EFFECT_DIGEST"], ["factsDigest", "PHASE_FACTS_DIGEST"]]) validateDigest(entry[field], code);
    const expectedReceipt = hash(JSON.stringify(stable(phaseReceiptPayload(entry))));
    if (entry.receiptDigest !== expectedReceipt) fail("PHASE_HASH_CHAIN");
    previousDigest = entry.receiptDigest;
  }
}

function journalRecordPayload(entry) {
  return {
    sequence: entry.sequence,
    phase: entry.phase,
    status: entry.status,
    previousDigest: entry.previousDigest,
    targetReceiptDigest: entry.targetReceiptDigest,
    effectDigest: entry.effectDigest,
  };
}

function validateFaultJournal(journal) {
  if (!Array.isArray(journal) || journal.length < 4) fail("FAULT_JOURNAL_SET");
  let previousDigest = "GENESIS";
  for (let index = 0; index < journal.length; index += 1) {
    const entry = journal[index];
    exactKeys(entry, ["sequence", "phase", "status", "previousDigest", "targetReceiptDigest", "effectDigest", "receiptDigest"], "FAULT_JOURNAL_ENTRY_CLOSED");
    if (entry.sequence !== index || entry.previousDigest !== previousDigest) fail("FAULT_JOURNAL_CHAIN");
    if (!Number.isSafeInteger(entry.sequence) || entry.sequence < 0 || typeof entry.phase !== "string" || typeof entry.status !== "string") fail("FAULT_JOURNAL_ENTRY");
    validateDigest(entry.targetReceiptDigest, "FAULT_JOURNAL_TARGET_RECEIPT");
    validateDigest(entry.effectDigest, "FAULT_JOURNAL_EFFECT");
    const expected = hash(JSON.stringify(stable(journalRecordPayload(entry))));
    if (entry.receiptDigest !== expected) fail("FAULT_JOURNAL_DIGEST");
    previousDigest = entry.receiptDigest;
  }
}

const FAULT_EXPECTED_FACTS = Object.freeze({
  INHERITED_BASE_RECEIPT_GAP: { storedReceiptReplayed: false, missingReceiptRecreatedAfterEffect: true, effectCount: 1, duplicateEffectCount: 0 },
  WORKSPACE_TRANSACTION_RECEIPT_GAP: { storedReceiptReplayed: false, missingReceiptRecreatedAfterEffect: true, effectCount: 1, duplicateEffectCount: 0 },
  CONCURRENT_TOKEN_RESERVATION: { sharedTokenPreReserved: false, winnerCount: 1, loserEffectCount: 0 },
  CLEANUP_RECEIPT_GAP: { storedCleanupReceiptReplayed: false, effectCount: 1, duplicateEffectCount: 0 },
});

function validateFaultCases(ledger) {
  if (!Array.isArray(ledger.faultCases) || JSON.stringify(ledger.faultCases.map(({ caseName }) => caseName)) !== JSON.stringify(FAULT_CASES)) fail("FAULT_CASE_SET");
  for (const entry of ledger.faultCases) {
    exactKeys(entry, ["caseName", "targetFingerprintDigest", "terminalState", "residueCount", "targetReceiptDigest", "caseFacts", "caseFactsDigest", "journal", "journalDigest", "caseDigest"], "FAULT_CASE_CLOSED");
    if (!FAULT_EXPECTED_FACTS[entry.caseName] || entry.terminalState !== "CLEANED" || entry.residueCount !== 0) fail("FAULT_CASE_NOT_CLEAN");
    validateDigest(entry.targetFingerprintDigest, "FAULT_CASE_TARGET");
    validateDigest(entry.targetReceiptDigest, "FAULT_CASE_TARGET_RECEIPT");
    if (JSON.stringify(entry.caseFacts) !== JSON.stringify(FAULT_EXPECTED_FACTS[entry.caseName])) fail("FAULT_CASE_FACTS");
    if (entry.caseFactsDigest !== hash(JSON.stringify(stable(entry.caseFacts)))) fail("FAULT_CASE_FACTS_DIGEST");
    validateFaultJournal(entry.journal);
    if (entry.journalDigest !== hash(JSON.stringify(stable(entry.journal)))) fail("FAULT_CASE_JOURNAL_DIGEST");
    const casePayload = {
      caseName: entry.caseName,
      targetFingerprintDigest: entry.targetFingerprintDigest,
      terminalState: entry.terminalState,
      residueCount: entry.residueCount,
      targetReceiptDigest: entry.targetReceiptDigest,
      caseFactsDigest: entry.caseFactsDigest,
      journalDigest: entry.journalDigest,
    };
    if (entry.caseDigest !== hash(JSON.stringify(stable(casePayload)))) fail("FAULT_CASE_DIGEST");
  }
}

export function validateLedger(ledger, kind) {
  if (!ledger || ledger.schemaVersion !== "aga-hybrid-demo-workspace-ledger/v2" || ledger.ledgerKind !== kind) fail("LEDGER_HEADER");
  privacyScan(ledger);
  if (kind === "happy-path") {
    exactKeys(ledger, ["schemaVersion", "ledgerKind", "terminalState", "residueCount", "targetFingerprintDigest", "intentDigest", "baseReceiptDigest", "phaseJournal", "facts", "evidence", "aggregateDigest"], "HAPPY_LEDGER_CLOSED");
    if (ledger.terminalState !== "CLEANED" || ledger.residueCount !== 0) fail("LEDGER_NOT_CLEAN");
    validateDigest(ledger.targetFingerprintDigest, "HAPPY_TARGET");
    validateDigest(ledger.intentDigest, "HAPPY_INTENT");
    validateDigest(ledger.baseReceiptDigest, "HAPPY_BASE_RECEIPT");
    validateHappyFacts(ledger.facts);
    validateHappyEvidence(ledger.evidence);
    validatePhaseJournal(ledger);
    if (ledger.aggregateDigest !== hash(JSON.stringify(stable({ phaseJournal: ledger.phaseJournal, facts: ledger.facts, evidence: ledger.evidence })))) fail("HAPPY_AGGREGATE_DIGEST");
  } else if (kind === "fault-matrix") {
    exactKeys(ledger, ["schemaVersion", "ledgerKind", "terminalState", "residueCount", "faultCases", "aggregateFacts", "evidence", "aggregateDigest"], "FAULT_LEDGER_CLOSED");
    if (ledger.terminalState !== "CLEANED" || ledger.residueCount !== 0) fail("LEDGER_NOT_CLEAN");
    validateFaultCases(ledger);
    validateFaultEvidence(ledger.evidence);
    const expectedFacts = Object.fromEntries(FAULT_CASES.map((caseName) => [caseName, ledger.faultCases.find((entry) => entry.caseName === caseName).caseFacts]));
    if (JSON.stringify(ledger.aggregateFacts) !== JSON.stringify(expectedFacts)) fail("FAULT_AGGREGATE_FACTS");
    if (ledger.aggregateDigest !== hash(JSON.stringify(stable({ faultCases: ledger.faultCases, aggregateFacts: ledger.aggregateFacts, evidence: ledger.evidence })))) fail("FAULT_AGGREGATE_DIGEST");
  } else {
    fail("LEDGER_KIND");
  }
  return true;
}

function readLedger(directory, name) {
  if (!directory || !directory.startsWith("/")) fail("LEDGER_DIR_ABSOLUTE");
  if (!existsSync(directory) || lstatSync(directory).isSymbolicLink() || (statSync(directory).mode & 0o777) !== 0o700) fail("LEDGER_DIR_INVALID");
  const path = resolve(directory, name);
  if (!existsSync(path) || lstatSync(path).isSymbolicLink() || (statSync(path).mode & 0o777) !== 0o600) fail("LEDGER_MISSING");
  const ledger = JSON.parse(readFileSync(path, "utf8"));
  validateLedger(ledger, ledger.ledgerKind);
  if (ledger.evidence?.sourceKind === "synthetic") fail("PROVENANCE_SYNTHETIC_LEDGER");
  const provenancePath = resolve(directory, "provenance.json");
  if (!existsSync(provenancePath) || lstatSync(provenancePath).isSymbolicLink() || (statSync(provenancePath).mode & 0o777) !== 0o600) fail("PROVENANCE_MISSING");
  const provenance = JSON.parse(readFileSync(provenancePath, "utf8"));
  validateProvenance(directory, provenance, ledger);
  return ledger;
}

function validateProvenance(directory, provenance, ledger) {
  exactKeys(provenance, ["schemaVersion", "ledgerKind", "manifestDigest", "signature", "rawReferences"], "PROVENANCE_CLOSED");
  if (provenance.schemaVersion !== "aga-hybrid-demo-workspace-provenance/v2" || provenance.ledgerKind !== ledger.ledgerKind || provenance.signature !== provenance.manifestDigest || !Array.isArray(provenance.rawReferences) || provenance.rawReferences.length === 0) fail("PROVENANCE_HEADER");
  const references = provenance.rawReferences;
  const seen = new Set();
  for (const reference of references) {
    exactKeys(reference, ["name", "role", "digest"], "PROVENANCE_REFERENCE_CLOSED");
    if (!/^raw\/[A-Za-z0-9._/-]+$/u.test(reference.name) || reference.name.includes("..") || seen.has(reference.name)) fail("PROVENANCE_REFERENCE_PATH");
    seen.add(reference.name);
    validateDigest(reference.digest, "PROVENANCE_REFERENCE_DIGEST");
    const target = resolve(directory, reference.name);
    if (!target.startsWith(`${resolve(directory, "raw")}/`) || !existsSync(target) || lstatSync(target).isSymbolicLink() || (statSync(target).mode & 0o777) !== 0o600) fail("PROVENANCE_RAW_FILE");
    const bytes = readFileSync(target);
    if (digestBytes(bytes) !== reference.digest) fail("PROVENANCE_RAW_DIGEST");
    privacyScanRawFile(reference.name, bytes);
  }
  const body = { ledgerKind: provenance.ledgerKind, rawReferences: references };
  if (provenance.manifestDigest !== hash(JSON.stringify(stable(body)))) fail("PROVENANCE_MANIFEST_DIGEST");
  const requiredRoles = ledger.ledgerKind === "happy-path"
    ? ["phase-receipts", "target-receipts", "consumed-authority", "f1", "forbidden-snapshot", "overlay-snapshot", "auth-snapshot", "browser-result", "lifecycle-probe", "workspace-terminal-facts"]
    : ["outer-journal", "case-target-receipts", "consumed-authority", "f3-execution"];
  for (const role of requiredRoles) if (!references.some((reference) => reference.role === role)) fail("PROVENANCE_ROLE_MISSING", role);
  if (ledger.ledgerKind === "happy-path" && ledger.evidence.sourceKind !== "connected-receipt") fail("PROVENANCE_SYNTHETIC_LEDGER");
  if (ledger.ledgerKind === "fault-matrix" && ledger.evidence.sourceKind !== "connected-receipt") fail("PROVENANCE_SYNTHETIC_LEDGER");
  const referenceDigest = (name) => references.find((reference) => reference.name === `raw/${name}`)?.digest;
  if (ledger.ledgerKind === "happy-path") {
    const bindings = [
      ["forbiddenBaselineDigest", "forbidden-before.json"],
      ["forbiddenFinalDigest", "forbidden-after-workspace.json"],
      ["overlayBaselineDigest", "overlay-before.json"],
      ["overlaySealedDigest", "overlay-after-seal.json"],
      ["overlayFinalDigest", "overlay-after-workspace.json"],
      ["authBeforeOidcDigest", "auth-before-oidc.json"],
      ["authAfterOidcDigest", "auth-after-oidc.json"],
      ["authBeforeBrowserDigest", "auth-before-browser.json"],
      ["authAfterBrowserDigest", "auth-after-browser.json"],
      ["f1ReceiptDigest", "f1-receipt.json"],
      ["browserResultDigest", "browser.log"],
      ["authControlEventDigest", "auth-events-manifest.json"],
      ["lifecycleProbeDigest", "lifecycle-probe.json"],
      ["lifecycleTerminalFactsDigest", "workspace-terminal-facts.json"],
    ];
    for (const [field, name] of bindings) if (ledger.evidence[field] !== referenceDigest(name)) fail("PROVENANCE_EVIDENCE_BINDING", field);
    const f1Reference = references.find((reference) => reference.name === "raw/f1-receipt.json");
    const f1 = f1Reference ? JSON.parse(readFileSync(resolve(directory, f1Reference.name), "utf8")) : null;
    if (!f1 || f1.schemaVersion !== "aga-hybrid-demo-f1-receipt/v2" || f1.caseCount !== 56 || f1.storedReceiptReplayCount !== ledger.evidence.f1StoredReceiptReplayCount || f1.missingReceiptRecreationCount !== ledger.evidence.f1MissingReceiptRecreationCount) fail("PROVENANCE_F1_BINDING");
    const phaseInputReference = references.find((reference) => reference.name === "raw/phase-receipts/phase-inputs.jsonl");
    const phaseInputLines = phaseInputReference ? readFileSync(resolve(directory, phaseInputReference.name), "utf8").trim().split("\n").filter(Boolean).map((line) => JSON.parse(line)) : [];
    if (phaseInputLines.length !== ledger.phaseJournal.length || phaseInputLines.some((input, index) => input.phase !== ledger.phaseJournal[index].phase || input.targetReceiptDigest !== ledger.phaseJournal[index].targetReceiptDigest || input.effectDigest !== ledger.phaseJournal[index].effectDigest || input.factsDigest !== ledger.phaseJournal[index].factsDigest)) fail("PROVENANCE_PHASE_BINDING");
    const authManifestReference = references.find((reference) => reference.name === "raw/auth-events-manifest.json");
    const authManifest = authManifestReference ? JSON.parse(readFileSync(resolve(directory, authManifestReference.name), "utf8")) : null;
    if (!authManifest || authManifest.schemaVersion !== "aga-hybrid-auth-control-event-manifest/v1" || authManifest.eventCount !== ledger.evidence.authControlEventCount || !Array.isArray(authManifest.events) || authManifest.events.length !== authManifest.eventCount) fail("PROVENANCE_AUTH_EVENTS");
    const lifecycleReference = references.find((reference) => reference.name === "raw/lifecycle-probe.json");
    const lifecycle = lifecycleReference ? JSON.parse(readFileSync(resolve(directory, lifecycleReference.name), "utf8")) : null;
    if (!lifecycle || lifecycle.schemaVersion !== "aga-hybrid-connected-lifecycle-probe/v1" || lifecycle.lifecycleCommandCount !== ledger.evidence.lifecycleCommandCount || lifecycle.findingState !== "CLOSED" || lifecycle.capState !== "ACCEPTED" || lifecycle.evidenceState !== "ACCEPTED" || lifecycle.closureBasis !== "EVIDENCE_VERIFIED" || !lifecycle.resetSucceeded || !lifecycle.resetReplay || !lifecycle.oldGenerationDenied) fail("PROVENANCE_LIFECYCLE");
    const terminalReference = references.find((reference) => reference.name === "raw/workspace-terminal-facts.json");
    const terminal = terminalReference ? JSON.parse(readFileSync(resolve(directory, terminalReference.name), "utf8")) : null;
    if (!terminal || terminal.terminalWorkspaceGenerationCount !== 2 || terminal.terminalWorkspaceActiveGenerationCount !== 1 || terminal.terminalWorkspaceResetGenerationCount !== 1 || terminal.terminalWorkspaceLifecycleEventCount !== 10 || terminal.terminalWorkspaceResetTombstoneCount !== 1) fail("PROVENANCE_TERMINAL_WORKSPACE");
  } else {
    if (ledger.evidence.f3ExecutionDigest !== referenceDigest("execution.json") || ledger.evidence.outerJournalDigest !== referenceDigest("outer-journal.jsonl")) fail("PROVENANCE_F3_BINDING");
    const executionReference = references.find((reference) => reference.name === "raw/execution.json");
    const execution = executionReference ? JSON.parse(readFileSync(resolve(directory, executionReference.name), "utf8")) : null;
    if (!execution || execution.schemaVersion !== "aga-hybrid-demo-fault-execution/v4" || execution.targetMode !== "compose-postgres" || execution.caseCount !== FAULT_CASES.length || ledger.evidence.authorityConsumptionDigest !== execution.authorityConsumptionDigest || !Number.isSafeInteger(execution.authorityConsumptionFileCount) || execution.authorityConsumptionFileCount < 1) fail("PROVENANCE_F3_EXECUTION");
    const authorityReferences = references.filter((reference) => reference.role === "consumed-authority").map((reference) => ({ name: reference.name.replace(/^raw\/consumed-authority\//u, ""), digest: reference.digest })).sort((left, right) => left.name.localeCompare(right.name));
    if (hash(JSON.stringify(stable(authorityReferences))) !== execution.authorityConsumptionDigest) fail("PROVENANCE_F3_AUTHORITY_BINDING");
    for (const caseName of FAULT_CASES) {
      const caseReference = references.find((reference) => reference.name === `raw/cases/${caseName}/target-residue.json`);
      const targetReference = references.find((reference) => reference.name === `raw/cases/${caseName}/target-manifest.json`);
      const executionCase = execution.cases.find((entry) => entry.caseName === caseName);
      if (!caseReference || !targetReference || !executionCase) fail("PROVENANCE_F3_CASE_BINDING", caseName);
      const residue = JSON.parse(readFileSync(resolve(directory, caseReference.name), "utf8"));
      const target = JSON.parse(readFileSync(resolve(directory, targetReference.name), "utf8"));
      if (residue.terminalState !== executionCase.terminalState || residue.residueCount !== executionCase.residueCount || target.targetFingerprintDigest !== executionCase.targetFingerprintDigest) fail("PROVENANCE_F3_CASE_RECEIPT", caseName);
    }
  }
}

function copyRawReference(rawDirectory, source, name, role, references) {
  if (!existsSync(source) || lstatSync(source).isSymbolicLink()) fail("PROVENANCE_SOURCE_MISSING", name);
  const sourceStat = statSync(source);
  if (!sourceStat.isFile() || (sourceStat.mode & 0o777) !== 0o600) fail("PROVENANCE_SOURCE_MODE", name);
  const target = resolve(rawDirectory, name);
  mkdirSync(dirname(target), { recursive: true, mode: 0o700 });
  copyFileSync(source, target);
  writeFileSync(target, readFileSync(source), { mode: 0o600 });
  references.push({ name: `raw/${name}`, role, digest: digestBytes(readFileSync(target)) });
}

function copyRawDirectory(rawDirectory, sourceDirectory, name, role, references) {
  if (!existsSync(sourceDirectory) || lstatSync(sourceDirectory).isSymbolicLink() || !statSync(sourceDirectory).isDirectory()) fail("PROVENANCE_SOURCE_DIRECTORY", name);
  for (const entry of readdirSync(sourceDirectory).sort()) {
    const source = resolve(sourceDirectory, entry);
    const sourceStat = lstatSync(source);
    if (sourceStat.isSymbolicLink()) fail("PROVENANCE_SOURCE_ENTRY", entry);
    if (sourceStat.isDirectory()) {
      if ((sourceStat.mode & 0o777) !== 0o700) fail("PROVENANCE_SOURCE_DIRECTORY_MODE", entry);
      copyRawDirectory(rawDirectory, source, `${name}/${entry}`, role, references);
      continue;
    }
    if (!sourceStat.isFile()) fail("PROVENANCE_SOURCE_ENTRY", entry);
    copyRawReference(rawDirectory, source, `${name}/${entry}`, role, references);
  }
}

export function writeProvenanceManifest(ledgerDirectory, rawRoot, ledgerKind) {
  if (!ledgerDirectory?.startsWith("/") || !rawRoot?.startsWith("/")) fail("PROVENANCE_ABSOLUTE");
  const rawDirectory = resolve(ledgerDirectory, "raw");
  mkdirSync(rawDirectory, { recursive: true, mode: 0o700 });
  const references = [];
  if (ledgerKind === "happy-path") {
    copyRawReference(rawDirectory, resolve(rawRoot, "phase-receipts/phase-inputs.jsonl"), "phase-receipts/phase-inputs.jsonl", "phase-receipts", references);
    copyRawReference(rawDirectory, resolve(rawRoot, "phase-receipts/journal.jsonl"), "phase-receipts/journal.jsonl", "phase-receipts", references);
    copyRawDirectory(rawDirectory, resolve(rawRoot, "target-receipts"), "target-receipts", "target-receipts", references);
    copyRawDirectory(rawDirectory, resolve(ledgerDirectory, "authority-consumption"), "consumed-authority", "consumed-authority", references);
    copyRawReference(rawDirectory, resolve(rawRoot, "f1-receipt.json"), "f1-receipt.json", "f1", references);
    for (const name of ["forbidden-before.json", "forbidden-after-workspace.json"]) copyRawReference(rawDirectory, resolve(rawRoot, name), name, "forbidden-snapshot", references);
    for (const name of ["overlay-before.json", "overlay-after-seal.json", "overlay-after-workspace.json"]) copyRawReference(rawDirectory, resolve(rawRoot, name), name, "overlay-snapshot", references);
    for (const name of ["auth-before-oidc.json", "auth-after-oidc.json", "auth-before-browser.json", "auth-after-browser.json", "auth-events-manifest.json"]) copyRawReference(rawDirectory, resolve(rawRoot, name), name, "auth-snapshot", references);
    copyRawDirectory(rawDirectory, resolve(rawRoot, "auth-events"), "auth-events", "auth-snapshot", references);
    copyRawReference(rawDirectory, resolve(rawRoot, "lifecycle-probe.json"), "lifecycle-probe.json", "lifecycle-probe", references);
    copyRawReference(rawDirectory, resolve(rawRoot, "workspace-terminal-facts.json"), "workspace-terminal-facts.json", "workspace-terminal-facts", references);
    for (const name of ["browser.log", "browser-discovery-full.log", "browser-discovery.log"]) copyRawReference(rawDirectory, resolve(rawRoot, name), name, "browser-result", references);
  } else if (ledgerKind === "fault-matrix") {
    copyRawReference(rawDirectory, resolve(rawRoot, "outer-journal.jsonl"), "outer-journal.jsonl", "outer-journal", references);
    copyRawReference(rawDirectory, resolve(rawRoot, "execution.json"), "execution.json", "f3-execution", references);
    copyRawDirectory(rawDirectory, resolve(rawRoot, "cases"), "cases", "case-target-receipts", references);
    copyRawDirectory(rawDirectory, resolve(ledgerDirectory, "authority-consumption"), "consumed-authority", "consumed-authority", references);
  } else {
    fail("PROVENANCE_KIND");
  }
  references.sort((left, right) => left.name.localeCompare(right.name));
  const body = { ledgerKind, rawReferences: references };
  const manifest = { schemaVersion: "aga-hybrid-demo-workspace-provenance/v2", ledgerKind, manifestDigest: hash(JSON.stringify(stable(body))), signature: hash(JSON.stringify(stable(body))), rawReferences: references };
  const output = resolve(ledgerDirectory, "provenance.json");
  const descriptor = openSync(output, "wx", 0o600);
  try { writeFileSync(descriptor, `${JSON.stringify(manifest, null, 2)}\n`); fsyncSync(descriptor); } finally { closeSync(descriptor); }
  return manifest;
}

function summaryFrom(happy, fault) {
  validateLedger(happy, "happy-path");
  validateLedger(fault, "fault-matrix");
  const facts = happy.facts;
  return {
    schemaVersion: "aga-hybrid-demo-workspace-evidence/v1",
    status: "verified locally",
    productStatus: "candidate-only",
    release: "release pending",
    productionReady: "not established",
    happyLedgerDigest: happy.aggregateDigest,
    faultLedgerDigest: fault.aggregateDigest,
    happyPhaseCount: happy.phaseJournal.length,
    faultCaseCount: fault.faultCases.length,
    connectedBrowser: facts.browserTestCount === 17 ? "verified locally" : "not run",
    browserTestCount: facts.browserTestCount,
    browserDiscoveryCount: facts.browserDiscoveryCount,
    browserPrivacyLeakCount: facts.browserPrivacyLeakCount,
    browserAuthCallbackMatch: facts.browserAuthCallbackMatch,
    lifecycleCommandCount: facts.lifecycleCommandCount,
    lifecycleTerminalState: facts.lifecycleTerminalState,
    lifecycleClosureBasis: facts.lifecycleClosureBasis,
    lifecycleResetVerified: facts.lifecycleResetVerified,
    forbiddenBusinessDelta: facts.forbiddenBusinessDelta,
    sealedOverlayDeltaAfterSeal: facts.sealedOverlayDeltaAfterSeal,
    residueCount: Math.max(happy.residueCount, fault.residueCount, facts.residueCount, ...fault.faultCases.map((entry) => entry.residueCount)),
  };
}

export function finalizeSummary(summaryPath, happyDirectory, faultDirectory) {
  const happy = readLedger(happyDirectory, "ledger.json");
  const fault = readLedger(faultDirectory, "ledger.json");
  const summary = summaryFrom(happy, fault);
  privacyScan(summary);
  const target = resolve(root, summaryPath);
  if (existsSync(target)) fail("SUMMARY_EXISTS");
  mkdirSync(dirname(target), { recursive: true });
  const bytes = Buffer.from(`${JSON.stringify(summary, null, 2)}\n`, "utf8");
  const descriptor = openSync(target, "wx", 0o644);
  try {
    writeFileSync(descriptor, bytes);
    fsyncSync(descriptor);
  } finally {
    closeSync(descriptor);
  }
  const parentDescriptor = openSync(dirname(target), "r");
  try { fsyncSync(parentDescriptor); } finally { closeSync(parentDescriptor); }
  return summary;
}

export function checkSummary(summaryPath, happyDirectory = process.env.AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR, faultDirectory = process.env.AVIA_AGA_HYBRID_FAULT_AUTHORIZATION_LEDGER_DIR) {
  const happy = readLedger(happyDirectory, "ledger.json");
  const fault = readLedger(faultDirectory, "ledger.json");
  const target = resolve(root, summaryPath);
  if (!existsSync(target) || lstatSync(target).isSymbolicLink() || (statSync(target).mode & 0o777) !== 0o644) fail("SUMMARY_MISSING");
  const before = { bytes: readFileSync(target), stat: statSync(target) };
  const summary = JSON.parse(before.bytes.toString("utf8"));
  const expected = summaryFrom(happy, fault);
  exactKeys(summary, Object.keys(expected), "SUMMARY_CLOSED");
  if (JSON.stringify(summary) !== JSON.stringify(expected)) fail("SUMMARY_DERIVATION");
  privacyScan(summary);
  const after = { bytes: readFileSync(target), stat: statSync(target) };
  if (!before.bytes.equals(after.bytes) || before.stat.size !== after.stat.size || before.stat.ino !== after.stat.ino || before.stat.mtimeMs !== after.stat.mtimeMs) fail("SUMMARY_MUTATED");
  return true;
}

function phaseReceipt(phase, targetFingerprintDigest, previousDigest, targetReceiptDigest, effectDigest, factsDigest) {
  const entry = { phase, status: "COMPLETED", targetFingerprintDigest, targetReceiptDigest, effectDigest, factsDigest, previousDigest };
  return { ...entry, receiptDigest: hash(JSON.stringify(stable(phaseReceiptPayload(entry)))) };
}

export function buildHappyLedger({ targetFingerprintDigest, intentDigest, baseReceiptDigest, phaseReceipts, facts, evidence }) {
  if (JSON.stringify(phaseReceipts?.map(({ phase }) => phase)) !== JSON.stringify(HAPPY_PHASES)) fail("HAPPY_BUILD_PHASES");
  let previousDigest = "GENESIS";
  const phaseJournal = phaseReceipts.map((receipt) => {
    validateDigest(receipt.targetReceiptDigest, "HAPPY_BUILD_TARGET_RECEIPT");
    validateDigest(receipt.effectDigest, "HAPPY_BUILD_EFFECT");
    validateDigest(receipt.factsDigest, "HAPPY_BUILD_FACTS");
    const entry = phaseReceipt(receipt.phase, targetFingerprintDigest, previousDigest, receipt.targetReceiptDigest, receipt.effectDigest, receipt.factsDigest);
    previousDigest = entry.receiptDigest;
    return entry;
  });
  const ledger = { schemaVersion: "aga-hybrid-demo-workspace-ledger/v2", ledgerKind: "happy-path", terminalState: "CLEANED", residueCount: facts.residueCount, targetFingerprintDigest, intentDigest, baseReceiptDigest, phaseJournal, facts, evidence, aggregateDigest: hash(JSON.stringify(stable({ phaseJournal, facts, evidence }))) };
  validateLedger(ledger, "happy-path");
  return ledger;
}

function faultRecord(caseName, targetFingerprintDigest, terminalState, residueCount, caseFacts, journal) {
  const caseFactsDigest = hash(JSON.stringify(stable(caseFacts)));
  const targetReceiptDigest = journal.at(-1).targetReceiptDigest;
  const journalDigest = hash(JSON.stringify(stable(journal)));
  const casePayload = { caseName, targetFingerprintDigest, terminalState, residueCount, targetReceiptDigest, caseFactsDigest, journalDigest };
  return { caseName, targetFingerprintDigest, terminalState, residueCount, targetReceiptDigest, caseFacts, caseFactsDigest, journal, journalDigest, caseDigest: hash(JSON.stringify(stable(casePayload))) };
}

export function buildFaultLedger({ cases, evidence }) {
  if (JSON.stringify(cases?.map(({ caseName }) => caseName)) !== JSON.stringify(FAULT_CASES)) fail("FAULT_BUILD_CASES");
  const faultCases = cases.map((entry) => {
    if (!Object.hasOwn(entry, "terminalState") || !Object.hasOwn(entry, "residueCount")) fail("FAULT_BUILD_TERMINAL_FACTS", entry.caseName);
    return faultRecord(entry.caseName, entry.targetFingerprintDigest, entry.terminalState, entry.residueCount, entry.caseFacts, entry.journal);
  });
  const aggregateFacts = Object.fromEntries(faultCases.map((entry) => [entry.caseName, entry.caseFacts]));
  const terminalState = faultCases.every((entry) => entry.terminalState === "CLEANED") ? "CLEANED" : "INTERRUPTED";
  const residueCount = faultCases.reduce((total, entry) => total + entry.residueCount, 0);
  const ledger = { schemaVersion: "aga-hybrid-demo-workspace-ledger/v2", ledgerKind: "fault-matrix", terminalState, residueCount, faultCases, aggregateFacts, evidence, aggregateDigest: hash(JSON.stringify(stable({ faultCases, aggregateFacts, evidence }))) };
  validateLedger(ledger, "fault-matrix");
  return ledger;
}

export function createSyntheticLedger(kind) {
  const targetFingerprintDigest = hash(`synthetic-target-${kind}`);
  const intentDigest = hash(`synthetic-intent-${kind}`);
  const baseReceiptDigest = hash(`synthetic-base-${kind}`);
  if (kind === "happy-path") {
    let previousDigest = "GENESIS";
    const phaseReceipts = HAPPY_PHASES.map((phase) => {
      const targetReceiptDigest = hash(`synthetic-target-receipt-${phase}`);
      const effectDigest = hash(`synthetic-effect-${phase}`);
      const factsDigest = hash(`synthetic-facts-${phase}`);
      const entry = { phase, targetReceiptDigest, effectDigest, factsDigest };
      previousDigest = phaseReceipt(phase, targetFingerprintDigest, previousDigest, targetReceiptDigest, effectDigest, factsDigest).receiptDigest;
      return entry;
    });
    const evidence = {
      sourceKind: "synthetic",
      forbiddenBaselineDigest: hash("synthetic-forbidden-baseline"), forbiddenFinalDigest: hash("synthetic-forbidden-final"),
      overlayBaselineDigest: hash("synthetic-overlay-baseline"), overlaySealedDigest: hash("synthetic-overlay-sealed"), overlayFinalDigest: hash("synthetic-overlay-final"),
      authBeforeOidcDigest: hash("synthetic-auth-before-oidc"), authAfterOidcDigest: hash("synthetic-auth-after-oidc"),
      authBeforeBrowserDigest: hash("synthetic-auth-before-browser"), authAfterBrowserDigest: hash("synthetic-auth-after-browser"),
      f1ReceiptDigest: hash("synthetic-f1"), f1StoredReceiptReplayCount: 28, f1MissingReceiptRecreationCount: 14, browserResultDigest: hash("synthetic-browser"), authControlEventDigest: hash("synthetic-auth-events"), authControlEventCount: 2, lifecycleProbeDigest: hash("synthetic-lifecycle-probe"), lifecycleTerminalFactsDigest: hash("synthetic-terminal-facts"), lifecycleCommandCount: 10,
    };
    return buildHappyLedger({ targetFingerprintDigest, intentDigest, baseReceiptDigest, phaseReceipts, facts: { ...EXPECTED_HAPPY_FACTS, terminalWorkspaceDraftCount: 2, terminalWorkspaceIdempotencyCount: 10 }, evidence });
  }
  if (kind === "fault-matrix") {
    const cases = FAULT_CASES.map((caseName, index) => {
      let previousDigest = "GENESIS";
      const journal = ["CASE_INTENT", "TARGET_EFFECT", "TARGET_RECEIPT", "LEDGER_PUBLICATION"].map((phase, sequence) => {
        const targetReceiptDigest = hash(`synthetic-${caseName}-${phase}`);
        const effectDigest = hash(`synthetic-effect-${caseName}-${phase}`);
        const entry = { sequence, phase, status: "COMPLETED", previousDigest, targetReceiptDigest, effectDigest };
        entry.receiptDigest = hash(JSON.stringify(stable(journalRecordPayload(entry))));
        previousDigest = entry.receiptDigest;
        return entry;
      });
      return { caseName, targetFingerprintDigest: hash(`synthetic-fault-target-${index}`), terminalState: "CLEANED", residueCount: 0, caseFacts: { ...FAULT_EXPECTED_FACTS[caseName] }, journal };
    });
    return buildFaultLedger({ cases, evidence: { sourceKind: "synthetic", f3ExecutionDigest: hash("synthetic-f3"), outerJournalDigest: hash("synthetic-outer-journal"), authorityConsumptionDigest: hash("synthetic-authority") } });
  }
  fail("SYNTHETIC_KIND");
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

if (process.argv[1] && fileURLToPath(import.meta.url) === resolve(process.argv[1])) {
  try {
    const values = parseArgs(process.argv.slice(2));
    if (values.has("finalize-summary")) {
      finalizeSummary(values.get("finalize-summary"), process.env.AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR, process.env.AVIA_AGA_HYBRID_FAULT_AUTHORIZATION_LEDGER_DIR);
      process.stdout.write("aga-hybrid-evidence: finalized privacy-safe summary\n");
    } else if (values.has("check-summary")) {
      checkSummary(values.get("check-summary"));
      process.stdout.write("aga-hybrid-evidence: summary check passed without mutation\n");
    } else {
      fail("MODE_REQUIRED");
    }
  } catch (error) {
    process.stderr.write(`${error instanceof Error ? error.message : "ERR_AGA_HYBRID_EVIDENCE_UNEXPECTED"}\n`);
    process.exitCode = 1;
  }
}
