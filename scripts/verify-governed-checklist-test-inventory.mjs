#!/usr/bin/env node
// Fail-closed phased governed-checklist inventory. It checks only the artifacts
// owned through the requested phase. The final phase additionally asks the
// configured Vitest and Playwright runners to discover their executable tests.
import { execFile } from "node:child_process";
import { readFile, stat } from "node:fs/promises";
import { resolve } from "node:path";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);
const root = process.cwd();
const phases = ["gate0", "task1", "task2", "task3", "task4", "task5", "task6", "task7", "task8", "task9", "final"];
const artifact = (kind, path, minimumSuites = 0) => ({ kind, path, minimumSuites });

const gate0 = [
  artifact("spec", "docs/product-specs/modules/CHECKLIST_BUILDER_AND_RUNNER.md"),
  artifact("spec", "docs/product-specs/modules/AUDIT_PLANNING.md"),
  artifact("spec", "docs/product-specs/modules/ADMIN_CONFIGURATION.md"),
  artifact("spec", "docs/product-specs/data-and-rules/CONCEPTUAL_DATA_MODEL.md"),
  artifact("spec", "docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md"),
  artifact("spec", "docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.md"),
  artifact("spec", "docs/product-specs/screen-specs/DEPARTMENT_MANAGER_WORKSPACES.md"),
  artifact("script", "scripts/verify-governed-checklist-test-inventory.mjs"),
  artifact("node", "tests/governed-checklist-intake-plan-contract.test.mjs", 1),
  artifact("node", "tests/governed-checklist-intake-security.test.mjs", 1),
  artifact("node", "tests/aga-checklist-archive-inventory.test.mjs", 1),
];

const phaseAdditions = {
  task1: [artifact("openapi", "api/openapi/tests/governed-checklist-intake-contract.test.mjs", 1), artifact("openapi", "api/openapi/tests/governed-checklist-authoring-contract.test.mjs", 1)],
  task2: [artifact("migration", "apps/api/migrations/000028_governed_checklist_intake_and_authoring.up.sql"), artifact("go", "apps/api/tests/integration/governed_checklist_intake_migration_test.go", 1)],
  task3: [artifact("go", "apps/api/internal/checklistintake/archive_test.go", 1), artifact("go", "apps/api/internal/httpapi/governed_checklist_intake_api_test.go", 1), artifact("profile", "scripts/test-governed-checklist-intake-profile.sh")],
  task4: [artifact("go", "apps/api/tests/integration/aga_form_048_candidate_intake_test.go", 1), artifact("go", "apps/api/internal/checklistintake/extraction_test.go", 1), artifact("go", "apps/api/internal/checklistintake/extraction_review_test.go", 1), artifact("script", "scripts/regulatory/inventory-checklist-archive.mjs")],
  task5: [artifact("go", "apps/api/tests/integration/governed_checklist_dual_authoring_test.go", 1), artifact("example", "api/openapi/examples/canonical/governed-official-source-draft.json")],
  task6: [artifact("go", "apps/api/tests/integration/aga_form_048_reconciliation_test.go", 1), artifact("go", "apps/api/internal/regulatory/reconciliation_test.go", 1), artifact("go", "apps/api/internal/regulatory/review_queue_test.go", 1), artifact("go", "apps/api/internal/regulatory/review_comments_test.go", 1)],
  task7: [artifact("go", "apps/api/tests/integration/governed_checklist_review_publication_test.go", 1), artifact("go", "apps/api/internal/checklistgovernance/eligibility_test.go", 1), artifact("go", "apps/api/internal/httpapi/governed_checklist_transport_mapping_test.go", 1), artifact("node", "tests/governed-checklist-discriminator-contract.test.mjs", 1)],
  task8: [artifact("react", "apps/web/src/backend/governed-checklist-intake-parity.test.ts", 1), artifact("react", "apps/web/src/features/admin/checklist-builder-page.test.tsx", 1), artifact("react", "apps/web/src/features/admin/checklist-intake-panel.test.tsx", 1), artifact("react", "apps/web/src/features/admin/checklist-candidate-review.test.tsx", 1), artifact("react", "apps/web/src/features/admin/checklist-draft-editor.test.tsx", 1), artifact("react", "apps/web/src/features/admin/checklist-reconciliation-diff.test.tsx", 1), artifact("react", "apps/web/src/features/admin/checklist-publication-blockers.test.tsx", 1), artifact("react", "apps/web/src/features/checklists/source-review-queue.test.tsx", 1), artifact("react", "apps/web/src/features/checklists/checklist-reviewer-queue.test.tsx", 1), artifact("playwright", "apps/web/tests/e2e/governed-checklist-intake.http.spec.ts", 1)],
  task9: [artifact("go", "apps/api/tests/integration/governed_checklist_intake_recovery_test.go", 1)],
};
const phaseBlockers = {
  task9: "blocked: the real Form 048 Admin identity/28 boundary packet, immutable candidate/source-gap Draft, and named expansion authorization remain external dependencies",
};

const legacyTask9 = [
  artifact("schema", "docs/regulatory-sources/schemas/regulatory-generation-request.schema.json"),
  artifact("schema", "docs/regulatory-sources/schemas/regulatory-generation-candidate-bundle.schema.json"),
  artifact("migration", "apps/api/migrations/000021_regulatory_checklist_governance.up.sql"),
  artifact("migration", "apps/api/migrations/000021_regulatory_checklist_governance.down.sql"),
  artifact("script", "scripts/regulatory/prepare-checklist-generation.mjs"),
  artifact("script", "scripts/regulatory/validate-checklist-candidate.mjs"),
  artifact("script", "scripts/regulatory/import-checklist-candidate.mjs"),
  artifact("node", "tests/service-provider-catalog.test.mjs", 1),
  artifact("node", "tests/regulatory-generation-contracts.test.mjs", 1),
  artifact("node", "tests/governed-checklist-lifecycle-smoke.test.mjs", 1),
  artifact("openapi", "api/openapi/tests/governed-checklist-contract.test.mjs", 1),
  artifact("openapi", "api/openapi/tests/governed-checklist-publication-boundary.test.mjs", 1),
  artifact("go", "apps/api/internal/regulatory/generation_test.go", 1),
  artifact("go", "apps/api/internal/checklistgovernance/applicability_test.go", 1),
  artifact("go", "apps/api/tests/integration/governed_checklist_task6_test.go", 1),
  artifact("go", "apps/api/tests/integration/governed_checklist_task7_test.go", 1),
  artifact("go", "apps/api/tests/integration/governed_checklist_task8_test.go", 1),
  artifact("go", "apps/api/tests/integration/governed_checklist_task9_test.go", 1),
  artifact("go", "apps/api/tests/integration/planning_assignment_scenario_test.go", 1),
  artifact("react", "apps/web/src/backend/governed-checklist-http-parity.test.ts", 1),
  artifact("react", "apps/web/src/features/checklists/checklist-management-page.test.tsx", 1),
  artifact("react", "apps/web/src/features/planning/new-audit-wizard.test.tsx", 1),
  artifact("playwright", "apps/web/tests/e2e/regulatory-checklist-governance.spec.ts", 1),
  artifact("playwright", "apps/web/tests/e2e/regulatory-checklist-governance.http.spec.ts", 1),
];

function requestedPhase(args) {
  if (args.length === 0) return "legacy";
  if (args.length !== 2 || args[0] !== "--phase" || !phases.includes(args[1])) {
    throw new Error(`unknown verification phase; expected one of: ${phases.join(", ")}`);
  }
  return args[1];
}

function artifactsThrough(phase) {
  if (phase === "legacy") return legacyTask9;
  const selected = [...gate0];
  for (const name of phases.slice(1, phases.indexOf(phase) + 1)) selected.push(...(phaseAdditions[name] ?? []));
  return phase === "final" ? [...selected, ...legacyTask9] : selected;
}

function countSuites(kind, content) {
  if (kind === "go") return (content.match(/^func Test\w+\(/gm) ?? []).length;
  return (content.match(/\b(?:test|it|describe)\s*\(/g) ?? []).length;
}

async function discoverFinalRunners() {
  const runnerRoot = resolve(root, "apps/web");
  const commands = [
    ["npm", ["exec", "vitest", "--", "list", "src/backend/governed-checklist-intake-parity.test.ts", "src/features/admin/checklist-builder-page.test.tsx", "src/features/admin/checklist-governed-intake-components.test.tsx"]],
    ["npm", ["exec", "playwright", "--", "test", "--list", "tests/e2e/governed-checklist-intake.spec.ts", "tests/e2e/governed-checklist-intake.http.spec.ts"]],
  ];
  for (const [command, args] of commands) {
    const { stdout } = await execFileAsync(command, args, { cwd: runnerRoot });
    if (!stdout.trim()) throw new Error(`${command} ${args.join(" ")} discovered no tests`);
    process.stdout.write(`RUNNER_DISCOVERY ${command} ${args.join(" ")} PASS\n`);
  }
}

let phase;
try {
  phase = requestedPhase(process.argv.slice(2));
} catch (error) {
  process.stderr.write(`${error.message}\n`);
  process.exitCode = 1;
  process.exit();
}

const missing = [];
const zeroSuites = [];
for (const { kind, path, minimumSuites } of artifactsThrough(phase)) {
  try {
    const metadata = await stat(resolve(root, path));
    if (!metadata.isFile()) throw new Error("not a regular file");
    const found = minimumSuites === 0 ? "present" : countSuites(kind, await readFile(resolve(root, path), "utf8"));
    process.stdout.write(`INVENTORY ${kind} ${path} expected>=${minimumSuites} observed=${found}\n`);
    if (typeof found === "number" && found < minimumSuites) zeroSuites.push(path);
  } catch {
    process.stdout.write(`MISSING ${kind} ${path}\n`);
    missing.push(path);
  }
}

if (missing.length || zeroSuites.length) {
  process.stderr.write(`Governed checklist verification inventory FAILED phase=${phase}: missing=${missing.length} zero-suite=${zeroSuites.length}\n`);
  for (const path of missing) process.stderr.write(`MISSING REQUIRED PATH: ${path}\n`);
  for (const path of zeroSuites) process.stderr.write(`ZERO REQUIRED SUITES: ${path}\n`);
  process.exitCode = 1;
} else if (phase === "final") {
  try {
    await discoverFinalRunners();
    if (phaseBlockers.task9) {
      process.stderr.write(`Governed checklist verification inventory BLOCKED phase=final: ${phaseBlockers.task9}\n`);
      process.exitCode = 2;
    }
  } catch (error) {
    process.stderr.write(`Governed checklist verification inventory FAILED phase=final: ${error.message}\n`);
    process.exitCode = 1;
  }
} else if (phaseBlockers[phase]) {
  process.stderr.write(`Governed checklist verification inventory BLOCKED phase=${phase}: ${phaseBlockers[phase]}\n`);
  process.exitCode = 2;
} else {
  process.stdout.write(`Governed checklist verification inventory PASS phase=${phase}: ${artifactsThrough(phase).length} required artifacts.\n`);
}
