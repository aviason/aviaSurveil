#!/usr/bin/env node
// Fail-closed Task 9 preflight. It deliberately starts no test runner: an
// omitted governed schema, test, or browser spec must stop aggregate evidence
// before a partial Node/Vitest/Playwright invocation can look successful.
import { readFile, stat } from "node:fs/promises";
import { resolve } from "node:path";

const root = process.cwd();
const required = [
  ["schema", "docs/regulatory-sources/schemas/regulatory-generation-request.schema.json", 0],
  ["schema", "docs/regulatory-sources/schemas/regulatory-generation-candidate-bundle.schema.json", 0],
  ["migration", "apps/api/migrations/000021_regulatory_checklist_governance.up.sql", 0],
  ["migration", "apps/api/migrations/000021_regulatory_checklist_governance.down.sql", 0],
  ["script", "scripts/regulatory/prepare-checklist-generation.mjs", 0],
  ["script", "scripts/regulatory/validate-checklist-candidate.mjs", 0],
  ["script", "scripts/regulatory/import-checklist-candidate.mjs", 0],
  ["script", "scripts/verify-governed-checklist-test-inventory.mjs", 0],
  ["node", "tests/service-provider-catalog.test.mjs", 1],
  ["node", "tests/regulatory-generation-contracts.test.mjs", 1],
  ["node", "tests/governed-checklist-lifecycle-smoke.test.mjs", 1],
  ["openapi", "api/openapi/tests/governed-checklist-contract.test.mjs", 1],
  ["openapi", "api/openapi/tests/governed-checklist-publication-boundary.test.mjs", 1],
  ["go", "apps/api/internal/regulatory/generation_test.go", 1],
  ["go", "apps/api/internal/checklistgovernance/applicability_test.go", 1],
  ["go", "apps/api/tests/integration/governed_checklist_task6_test.go", 1],
  ["go", "apps/api/tests/integration/governed_checklist_task7_test.go", 1],
  ["go", "apps/api/tests/integration/governed_checklist_task8_test.go", 1],
  ["go", "apps/api/tests/integration/governed_checklist_task9_test.go", 1],
  ["react", "apps/web/src/backend/governed-checklist-http-parity.test.ts", 1],
  ["react", "apps/web/src/features/checklists/checklist-management-page.test.tsx", 1],
  ["react", "apps/web/src/features/inspections/inspection-package-builder-page.test.tsx", 1],
  ["playwright", "apps/web/tests/e2e/regulatory-checklist-governance.spec.ts", 1],
  ["playwright", "apps/web/tests/e2e/regulatory-checklist-governance.http.spec.ts", 1],
];

function countSuites(kind, content) {
  if (kind === "go") return (content.match(/^func Test\w+\(/gm) ?? []).length;
  return (content.match(/\b(?:test|it|describe)\s*\(/g) ?? []).length;
}

const missing = [];
const zeroSuites = [];
for (const [kind, relativePath, minimumSuites] of required) {
  const absolutePath = resolve(root, relativePath);
  try {
    const metadata = await stat(absolutePath);
    if (!metadata.isFile()) throw new Error("not a regular file");
    const content = minimumSuites === 0 ? "" : await readFile(absolutePath, "utf8");
    const found = minimumSuites === 0 ? "present" : countSuites(kind, content);
    process.stdout.write(`INVENTORY ${kind} ${relativePath} expected>=${minimumSuites} observed=${found}\n`);
    if (typeof found === "number" && found < minimumSuites) zeroSuites.push(relativePath);
  } catch {
    process.stdout.write(`MISSING ${kind} ${relativePath}\n`);
    missing.push(relativePath);
  }
}

if (missing.length || zeroSuites.length) {
  process.stderr.write(`Governed checklist verification inventory FAILED before runners: missing=${missing.length} zero-suite=${zeroSuites.length}\n`);
  for (const path of missing) process.stderr.write(`MISSING REQUIRED PATH: ${path}\n`);
  for (const path of zeroSuites) process.stderr.write(`ZERO REQUIRED SUITES: ${path}\n`);
  process.exitCode = 1;
} else {
  process.stdout.write(`Governed checklist verification inventory PASS: ${required.length} required artifacts.\n`);
}
