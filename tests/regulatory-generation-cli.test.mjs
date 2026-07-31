import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";
import { validateRealOPSAOCRequest } from "../scripts/regulatory/checklist-generation-contracts.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const run = (script, ...args) => spawnSync("node", [path.join(root, "scripts/regulatory", script), ...args], { cwd: root, encoding: "utf8" });
const runWithEnv = (environment, script, ...args) => spawnSync("node", [path.join(root, "scripts/regulatory", script), ...args], { cwd: root, encoding: "utf8", env: { ...process.env, ...environment } });
const fixture = "docs/regulatory-sources/fixtures/synthetic-ops-aoc-generation-candidate.v1.json";

test("Task 4 CLI prepares the bounded real pilot request but never manufactures authority", () => {
  const result = run("prepare-checklist-generation.mjs", "--request", "GENREQ-OPS-AOC-0001");
  assert.equal(result.status, 0, result.stderr);
  const request = JSON.parse(result.stdout);
  assert.equal(request.requestId, "GENREQ-OPS-AOC-0001");
  assert.equal(request.unresolvedSourceGaps.length, 3);
  assert.match(request.unresolvedSourceGaps.map((gap) => gap.reason).join(" "), /Part 140|Part 127|controlled NCAA/i);
  assert.doesNotThrow(() => validateRealOPSAOCRequest(request));
  request.sourceSnapshots[0].clauseLocators = ["Fabricated locator"];
  assert.throws(() => validateRealOPSAOCRequest(request), /source, hash, clause, or locator/);
});

test("Task 4 CLI validates the explicit synthetic result and rejects a real pilot import", () => {
  const synthetic = run("validate-checklist-candidate.mjs", fixture);
  assert.equal(synthetic.status, 0, synthetic.stderr);
  assert.match(synthetic.stdout, /validated/i);

  const real = run("import-checklist-candidate.mjs", fixture, "--request", "GENREQ-OPS-AOC-0001");
  assert.notEqual(real.status, 0);
  assert.match(real.stderr, /blocked|unresolved/i);
});

test("Task 4 CLI rejects remote and non-test database profiles before connecting", () => {
  for (const databaseURL of [
    "postgres://aviasurveil:task4-local-only@example.com:5432/avia_task4_fix2?sslmode=disable",
    "postgres://aviasurveil:task4-local-only@127.0.0.1:55448/production?sslmode=disable",
  ]) {
    const result = runWithEnv({ AVIA_REGULATORY_TEST_MODE: "1", AVIA_REGULATORY_DATABASE_URL: databaseURL }, "import-checklist-candidate.mjs", fixture);
    assert.notEqual(result.status, 0, result.stdout);
    assert.match(result.stderr, /loopback test-only|blocked/i);
  }
});
