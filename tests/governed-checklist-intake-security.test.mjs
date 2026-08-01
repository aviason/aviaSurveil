import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const paths = [
  "docs/product-specs/modules/CHECKLIST_BUILDER_AND_RUNNER.md",
  "docs/product-specs/modules/AUDIT_PLANNING.md",
  "docs/product-specs/modules/ADMIN_CONFIGURATION.md",
  "docs/product-specs/data-and-rules/CONCEPTUAL_DATA_MODEL.md",
  "docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md",
  "docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.md",
  "docs/product-specs/screen-specs/DEPARTMENT_MANAGER_WORKSPACES.md",
];

test("Gate 0 retains eight top-level roles and prevents authority/privacy bypasses", async () => {
  const specs = await Promise.all(paths.map((path) => readFile(path, "utf8")));
  const combined = specs.join("\n");
  const security = specs[4];

  assert.match(security, /\| Action \| Inspector \| Lead Inspector \| Department Manager \| Finance \| General Manager \| Executive Director \| Auditee \| Admin \|/);
  assert.doesNotMatch(combined, /\|[^\n]*\b(?:Technical Expert|Source Owner|Reviewer)\b[^\n]*\|/i);
  assert.match(combined, /functional assignments[\s\S]*not top-level roles/i);
  assert.match(combined, /Admin[\s\S]*cannot[\s\S]*technically approve[\s\S]*publish/i);
  assert.match(combined, /Auditee[\s\S]*cannot[\s\S]*intake[\s\S]*review[\s\S]*decision/i);
  assert.match(combined, /Auditee[\s\S]*Internal CAA Note/i);
  assert.match(combined, /publication[\s\S]*does not[\s\S]*Audit-package eligibility/i);
});

test("Gate 0 inventory supports only phased, fail-closed verification", async () => {
  const inventory = await readFile("scripts/verify-governed-checklist-test-inventory.mjs", "utf8");
  for (const phase of ["gate0", "task1", "task2", "task3", "task4", "task5", "task6", "task7", "task8", "task9", "final"]) {
    assert.match(inventory, new RegExp(`\\b${phase}\\b`));
  }
  assert.match(inventory, /unknown verification phase/i);
  assert.match(inventory, /tests\/aga-checklist-archive-inventory\.test\.mjs/);
  assert.match(inventory, /tests\/governed-checklist-intake-plan-contract\.test\.mjs/);
  assert.match(inventory, /tests\/governed-checklist-intake-security\.test\.mjs/);
  assert.match(inventory, /task9[\s\S]*blocked/i);
  assert.match(inventory, /real[\s\S]*(?:Form 048|slice)[\s\S]*authorization/i);
});
