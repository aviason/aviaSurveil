import assert from "node:assert/strict";
import { chmodSync, existsSync, mkdirSync, mkdtempSync, readFileSync, readFileSync as readBytes, rmSync, statSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { tmpdir } from "node:os";
import test from "node:test";

import { validateInventory } from "../scripts/build-aga-hybrid-forbidden-inventory.mjs";
import { issueAuthorization, validateAuthorization } from "../scripts/issue-aga-hybrid-connected-authorization.mjs";
import { checkSummary, createSyntheticLedger, finalizeSummary, validateLedger } from "../scripts/verify-aga-hybrid-demo-workspace-evidence.mjs";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const read = (relative) => readFileSync(resolve(root, relative), "utf8");

test("Task 3 workspace files and role boundary are closed", () => {
  for (const relative of [
    "apps/api/internal/preproddata/agademoworkspace/contract.go",
    "apps/api/internal/preproddata/agademoworkspace/postgres_roles.go",
    "apps/api/internal/preproddata/agademoworkspace/postgres_provision.go",
    "apps/api/internal/preproddata/agademoworkspace/postgres_store.go",
    "apps/api/internal/preproddata/agademoworkspace/store.go",
    "apps/api/internal/preproddata/agademoworkspace/fixture.go",
    "docs/product-specs/data-and-rules/aga-demo-workspace-fixture.schema.json",
    "tests/fixtures/aga-demo-workspace-authority-fixture-template.v1.json",
  ]) {
    assert.ok(existsSync(resolve(root, relative)), `missing ${relative}`);
  }
  const contract = read("apps/api/internal/preproddata/agademoworkspace/contract.go");
  const provision = read("apps/api/internal/preproddata/agademoworkspace/postgres_provision.go");
  for (const role of [
    "preprod_aga_demo_workspace_owner",
    "preprod_aga_demo_workspace_fixture_exporter",
    "preprod_aga_demo_workspace_loader",
    "preprod_aga_demo_workspace_reader",
    "preprod_aga_demo_workspace_command",
  ]) assert.match(contract, new RegExp(role, "u"));
  assert.match(provision, /classification_pass_records/u);
  assert.match(provision, /question_versions/u);
  assert.match(provision, /idempotency_responses/u);
  assert.doesNotMatch(contract, /preprod_aga_demo\./u);
  assert.doesNotMatch(contract, /Provider\s+table/u);
});

test("Task 3 Compose and secret names are explicit", () => {
  const compose = read("deploy/local/compose.yaml");
  for (const token of [
    "aga-demo-workspace-loader",
    "preprod-aga-demo-workspace-role-provisioner",
    "preprod-aga-demo-workspace-fixture-exporter",
    "preprod-aga-demo-workspace-loader",
    "preprod_aga_demo_workspace_fixture_exporter_database_password",
    "preprod_aga_demo_workspace_loader_database_password",
    "preprod_aga_demo_workspace_reader_database_password",
    "preprod_aga_demo_workspace_command_database_password",
  ]) assert.match(compose, new RegExp(token.replaceAll("-", "[-_]"), "u"));
  const dockerfile = read("apps/api/Dockerfile");
  for (const target of ["preprod-aga-demo-workspace-role-provisioner", "preprod-aga-demo-workspace-fixture-exporter", "preprod-aga-demo-workspace-loader"]) assert.match(dockerfile, new RegExp(target, "u"));
  const initializer = read("scripts/init-local-preprod-namespace.sh");
  for (const secret of ["preprod_aga_demo_workspace_fixture_exporter_database_password", "preprod_aga_demo_workspace_loader_database_password", "preprod_aga_demo_workspace_reader_database_password", "preprod_aga_demo_workspace_command_database_password"]) assert.match(initializer, new RegExp(secret, "u"));
  assert.doesNotMatch(read("scripts/init-local-secrets.sh"), /aga_demo_workspace/u);
  const secretsReadme = read("deploy/local/secrets/README.md");
  assert.match(secretsReadme, /workspace/u);
});

test("Task 3 fixture schema is closed and text-free", () => {
  const schema = JSON.parse(read("docs/product-specs/data-and-rules/aga-demo-workspace-fixture.schema.json"));
  assert.equal(schema.additionalProperties, false);
  const template = JSON.parse(read("tests/fixtures/aga-demo-workspace-authority-fixture-template.v1.json"));
  assert.equal(template.schemaVersion, "aga-demo-workspace-authority-fixture/v1");
  assert.equal(template.syntheticNamespace, "AGA_DEMO_ONLY");
  assert.equal(template.providerConfiguration.length, 20);
  assert.equal(template.scopes.length, 2);
  assert.equal(Object.prototype.hasOwnProperty.call(template, "subjectId"), false);
  assert.equal(Object.prototype.hasOwnProperty.call(template, "membershipId"), false);
});

test("Task 9 inventory, authorization, recovery, and evidence files are closed", () => {
  for (const relative of [
    "scripts/test-aga-hybrid-demo-workspace-connected.sh",
    "scripts/verify-aga-hybrid-demo-workspace-evidence.mjs",
    "scripts/build-aga-hybrid-forbidden-inventory.mjs",
    "scripts/issue-aga-hybrid-connected-authorization.mjs",
    "tests/fixtures/aga-hybrid-forbidden-object-inventory.v1.json",
    "tests/fixtures/aga-hybrid-connected-authorization.schema.json",
  ]) assert.ok(existsSync(resolve(root, relative)), `missing ${relative}`);
  const inventory = JSON.parse(read("tests/fixtures/aga-hybrid-forbidden-object-inventory.v1.json"));
  assert.doesNotThrow(() => validateInventory(inventory));
  const allObjects = Object.values(inventory.classes).flatMap((entry) => entry.objects ?? []);
  assert.equal(new Set(allObjects).size, allObjects.length);
  assert.ok(inventory.classes.FORBIDDEN_BUSINESS.objects.every((object) => !object.startsWith("preprod_aga_demo_workspace.")));
  assert.ok(inventory.classes.WORKSPACE_ALLOWED.objects.every((object) => object.startsWith("preprod_aga_demo_workspace.")));
  const authorizationSchema = JSON.parse(read("tests/fixtures/aga-hybrid-connected-authorization.schema.json"));
  assert.equal(authorizationSchema.additionalProperties, false);
  const harness = read("scripts/test-aga-hybrid-demo-workspace-connected.sh");
  assert.match(harness, /pending external authority/u);
  assert.doesNotMatch(harness, /issue-aga-hybrid-connected-authorization/u);
  assert.match(harness, /fault-matrix-prepare/u);
});

test("authorization envelopes are closed and atomically consumed", () => {
  const directory = mkdtempSync(`${tmpdir()}/aga-hybrid-auth-`);
  chmodSync(directory, 0o700);
  const output = resolve(directory, "authorization.json");
  const values = new Map([
    ["output", output],
    ["phase", "prepare"],
    ["issuer", "local-test-operator"],
    ["input-digest", `sha256:${"a".repeat(64)}`],
    ["code-digest", `sha256:${"b".repeat(64)}`],
    ["contract-digest", `sha256:${"c".repeat(64)}`],
    ["operations", "CREATE_BASE,QUALIFY_EXISTING_SYNTHETIC_OIDC,PIN_PRE_WORKSPACE_FORBIDDEN_BASELINE,PROVISION_EMPTY_WORKSPACE_CONTRACT,EXPORT_WORKSPACE_FIXTURE,PREPARE_LOAD_INTENTS,CLEANUP_ON_PREPARE_FAILURE"],
  ]);
  try {
    issueAuthorization(values);
    const authorization = JSON.parse(readFileSync(output, "utf8"));
    assert.equal(statSync(output).mode & 0o777, 0o600);
    assert.equal(authorization.tokens.length, authorization.operations.length);
    assert.equal(new Set(authorization.tokens).size, authorization.tokens.length);
    assert.doesNotThrow(() => validateAuthorization(authorization, new Date(authorization.issuedAt)));
    assert.throws(() => issueAuthorization(values), /OUTPUT_EXISTS/u);
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});

test("forbidden baseline predates workspace provisioning", () => {
  const inventory = JSON.parse(read("tests/fixtures/aga-hybrid-forbidden-object-inventory.v1.json"));
  const forbidden = new Set(inventory.classes.FORBIDDEN_BUSINESS.objects);
  const workspace = inventory.classes.WORKSPACE_ALLOWED.objects;
  assert.ok(workspace.every((object) => !forbidden.has(object)));
  assert.ok(inventory.coverage.migrationTableCount > inventory.coverage.workspaceObjectCount);
});

test("loader barriers precede credential revocation", () => {
  const harness = read("scripts/test-aga-hybrid-demo-workspace-connected.sh");
  const plan = read("docs/exec-plans/active/2026-08-03-aga-hybrid-classification-demo-lifecycle-plan.md");
  assert.ok(plan.indexOf("LOAD_SEAL_BARRIERS_COMPLETE") < plan.indexOf("CREDENTIALS_REVOKED"));
  assert.ok(harness.indexOf("RUN_WORKSPACE_LOAD_SEAL_BARRIERS") < harness.indexOf("CREDENTIALS_REVOKED"));
});

test("connected receipt qualification is explicit and fail-closed", () => {
  const harness = read("scripts/test-aga-hybrid-demo-workspace-connected.sh");
  for (const marker of [
    "prepare_connected_target",
    "validate_connected_receipts",
    "consume_authorization",
    "write_happy_ledger",
    "write_fault_ledger",
    "run_f1_probe",
    "run_f3_probe",
    "browserTestCount: 17",
    "credentialRevocationReceiptCount",
    "residueCount",
  ]) assert.match(harness, new RegExp(marker.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&"), "u"));
  assert.match(harness, /pending external authority/u);
  assert.doesNotMatch(harness, /AVIA_AGA_HYBRID_CONNECTED_RECEIPTS_FILE/u);
  assert.doesNotMatch(harness, /CONNECTED_EXECUTION_NOT_IMPLEMENTED/u);
  assert.doesNotMatch(harness, /issue-aga-hybrid-connected-authorization/u);
});

test("phase journal recovers every commit publication boundary", () => {
  const ledger = createSyntheticLedger("happy-path");
  assert.equal(ledger.phaseJournal.length, 14);
  assert.equal(ledger.phaseJournal[0].previousDigest, "GENESIS");
  for (let index = 1; index < ledger.phaseJournal.length; index += 1) {
    assert.equal(ledger.phaseJournal[index].previousDigest, ledger.phaseJournal[index - 1].receiptDigest);
  }
  assert.doesNotThrow(() => validateLedger(ledger, "happy-path"));
});

test("fault matrix manifest has the exact four connected cases", () => {
  const ledger = createSyntheticLedger("fault-matrix");
  assert.deepEqual(ledger.faultCases.map((entry) => entry.caseName), [
    "INHERITED_BASE_RECEIPT_GAP",
    "WORKSPACE_TRANSACTION_RECEIPT_GAP",
    "CONCURRENT_TOKEN_RESERVATION",
    "CLEANUP_RECEIPT_GAP",
  ]);
  assert.doesNotThrow(() => validateLedger(ledger, "fault-matrix"));
});

test("evidence finalizer is privacy-safe and check-only", () => {
  const directory = mkdtempSync(`${tmpdir()}/aga-hybrid-evidence-`);
  const happyDirectory = resolve(directory, "happy");
  const faultDirectory = resolve(directory, "fault");
  const summaryPath = resolve(directory, "summary.md");
  mkdirSync(happyDirectory, { mode: 0o700 });
  mkdirSync(faultDirectory, { mode: 0o700 });
  chmodSync(happyDirectory, 0o700);
  chmodSync(faultDirectory, 0o700);
  writeFileSync(resolve(happyDirectory, "ledger.json"), `${JSON.stringify(createSyntheticLedger("happy-path"))}\n`, { mode: 0o600 });
  writeFileSync(resolve(faultDirectory, "ledger.json"), `${JSON.stringify(createSyntheticLedger("fault-matrix"))}\n`, { mode: 0o600 });
  try {
    assert.doesNotThrow(() => validateLedger(JSON.parse(readBytes(resolve(happyDirectory, "ledger.json"), "utf8")), "happy-path"));
    assert.doesNotThrow(() => validateLedger(JSON.parse(readBytes(resolve(faultDirectory, "ledger.json"), "utf8")), "fault-matrix"));
    assert.throws(() => finalizeSummary(summaryPath, happyDirectory, faultDirectory), /PROVENANCE_MISSING|PROVENANCE_SYNTHETIC_LEDGER/u);
    assert.equal(existsSync(summaryPath), false);
    assert.throws(() => validateLedger({ ...createSyntheticLedger("happy-path"), privatePath: "/tmp/private-ledger" }, "happy-path"), /SENSITIVE_FIELD/u);
    assert.throws(() => validateLedger({ ...createSyntheticLedger("happy-path"), phaseJournal: [{ phase: "TARGET_CREATED", status: "COMPLETED", previousDigest: "GENESIS", receiptDigest: `sha256:${"0".repeat(64)}` }] }, "happy-path"), /HAPPY_PHASE_SET/u);
    assert.throws(() => checkSummary(summaryPath, resolve(directory, "missing-happy"), faultDirectory), /LEDGER_DIR_INVALID/u);
    const broken = JSON.parse(readBytes(resolve(happyDirectory, "ledger.json"), "utf8"));
    broken.phaseJournal[1].previousDigest = "GENESIS";
    writeFileSync(resolve(happyDirectory, "ledger.json"), `${JSON.stringify(broken)}\n`, { mode: 0o600 });
    assert.throws(() => checkSummary(summaryPath, happyDirectory, faultDirectory), /PHASE_RECEIPT_CHAIN|SUMMARY_DERIVATION/u);
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});
