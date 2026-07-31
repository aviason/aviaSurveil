import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";

const root = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..");

test("root legacy demo retains checklist lifecycle controls without claiming governed publication", () => {
  const checklist = fs.readFileSync(path.join(root, "js/checklists.js"), "utf8");
  const inspection = fs.readFileSync(path.join(root, "js/inspection.js"), "utf8");
  assert.match(checklist, /createChecklistDraftVersion/);
  assert.match(checklist, /publishChecklistVersion/);
  assert.match(inspection, /Potential Finding|potentialFinding/i);
  assert.doesNotMatch(checklist, /SOURCE-SYNTHETIC-OPS-AOC|NCAA-CC-ANNEX6/);
});
