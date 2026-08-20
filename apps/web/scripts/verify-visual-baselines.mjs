#!/usr/bin/env node
import { createHash } from "node:crypto";
import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { release } from "node:os";
import { dirname, relative, resolve, sep } from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

import { chromium } from "playwright";

const scriptDir = fileURLToPath(new URL(".", import.meta.url));
const appRoot = resolve(scriptDir, "..");
const repoRoot = resolve(scriptDir, "../../..");
const baselineDir = resolve(appRoot, "tests/visual-baselines/react-legacy-parity");
const manifestPath = resolve(baselineDir, "baseline-manifest.json");
const baselineVersion = "react-legacy-parity-v1";
const fixedTimeIso = "2026-06-15T09:00:00.000Z";

const viewports = {
  desktop: { width: 1440, height: 900 },
  tablet: { width: 1024, height: 768 },
  mobile: { width: 390, height: 844 },
};

const legacyExpectedSurfaces = {
  "role-select": {
    auditId: "ui-audit-001",
    parityMode: "strict-shell",
    reactPath: "/",
    legacyView: "login",
    legacyParams: {},
  },
  "inspector-home": {
    auditId: "ui-audit-002",
    parityMode: "content-adapted",
    reactPath: "/inspector/inspector-assignments",
    legacyView: "inspector-assignments",
    legacyParams: {},
  },
  "lead-home": {
    auditId: "ui-audit-013",
    parityMode: "content-adapted",
    reactPath: "/lead-inspector/lead-review",
    legacyView: "lead-review",
    legacyParams: {},
  },
  "manager-home": {
    auditId: "ui-audit-027",
    parityMode: "content-adapted",
    reactPath: "/department-manager/dashboard",
    legacyView: "dashboard",
    legacyParams: {},
  },
  "gm-home": {
    auditId: "ui-audit-052",
    parityMode: "content-adapted",
    reactPath: "/general-manager/gm-dashboard",
    legacyView: "gm-dashboard",
    legacyParams: {},
  },
  "finance-home": {
    auditId: "ui-audit-058",
    parityMode: "content-adapted",
    reactPath: "/finance/finance-review",
    legacyView: "finance-review",
    legacyParams: {},
  },
  "executive-home": {
    auditId: "ui-audit-059",
    parityMode: "content-adapted",
    reactPath: "/executive-director/executive-dashboard",
    legacyView: "executive-dashboard",
    legacyParams: {},
  },
  "auditee-home": {
    auditId: "ui-audit-066",
    parityMode: "content-adapted",
    reactPath: "/auditee/service-provider-cap",
    legacyView: "service-provider-cap",
    legacyParams: {},
  },
  "admin-home": {
    auditId: "ui-audit-076",
    parityMode: "content-adapted",
    reactPath: "/admin/templates",
    legacyView: "template-preview",
    legacyParams: { id: "TPL-CABIN-2026" },
  },
  "audit-detail": {
    auditId: "ui-audit-007",
    parityMode: "content-adapted",
    reactPath: "/inspector/audits/AUD-2026-001",
    legacyView: "audit-detail",
    legacyParams: { auditId: "AUD-2026-001" },
  },
  "checklist-runner": {
    auditId: "ui-audit-008",
    parityMode: "content-adapted",
    reactPath: "/inspector/audits/AUD-2026-001/checklist",
    legacyView: "checklist",
    legacyParams: { auditId: "AUD-2026-001", questionId: "cab-em-eq-pbe" },
  },
  "organization-registry": {
    auditId: "ui-audit-041",
    parityMode: "content-adapted",
    reactPath: "/department-manager/organizations",
    legacyView: "organizations",
    legacyParams: {},
  },
  "audit-plan": {
    auditId: "ui-audit-028",
    parityMode: "content-adapted",
    reactPath: "/department-manager/audit-plan",
    legacyView: "planning",
    legacyParams: {},
  },
  "finding-detail": {
    auditId: "ui-audit-009",
    parityMode: "content-adapted",
    reactPath: "/inspector/findings/FND-CAB-2026-001",
    legacyView: "finding",
    legacyParams: { findingId: "CAB-2026-011" },
  },
  "cap-review": {
    auditId: "ui-audit-022",
    parityMode: "content-adapted",
    reactPath: "/lead-inspector/cap-review/FND-CAB-2026-001",
    legacyView: "cap-review-detail",
    legacyParams: { findingId: "CAB-2026-011" },
  },
  "evidence-review": {
    auditId: "ui-audit-044",
    parityMode: "content-adapted",
    reactPath: "/department-manager/evidence/FND-CAB-2026-001",
    legacyView: "findings",
    legacyParams: { filter: "evreview" },
  },
  "report-preview": {
    auditId: "ui-audit-030",
    parityMode: "content-adapted",
    reactPath: "/department-manager/reports/RPT-CAB-2026-001-V1",
    legacyView: "reports-approval",
    legacyParams: { reportId: "PR-2026-018" },
  },
};

const additionalLegacyExpectedSurfaces = {
  "inspector-findings": { legacyView: "findings", legacyParams: {} }, "inspector-messages": { legacyView: "messages", legacyParams: {} }, "inspector-calendar": { legacyView: "calendar", legacyParams: {} }, "inspector-reports": { legacyView: "reports", legacyParams: {} }, "closure-report-preview": { legacyView: "report", legacyParams: { findingId: "CAB-2026-011" } }, "inspector-assistant": { legacyView: "ai-assistant", legacyParams: { sourceView: "finding", findingId: "CAB-2026-011" } }, "inspector-profile": { legacyView: "profile", legacyParams: {} },
  "lead-preliminary-reports": { legacyView: "audit-reports", legacyParams: { filter: "preliminary" } }, "lead-preliminary-report-workflow": { legacyView: "audit-reports", legacyParams: { filter: "preliminary", reportId: "PR-2026-018" } }, "lead-final-reports": { legacyView: "audit-reports", legacyParams: { filter: "final" } }, "lead-final-report-readiness": { legacyView: "audit-reports", legacyParams: { filter: "final", finalReportId: "FR-2026-018" } }, "lead-prepare-final-report": { legacyView: "final-report-prepare", legacyParams: { reportId: "FR-2026-018" } }, "lead-final-report-document": { legacyView: "final-report-view", legacyParams: { reportId: "FR-2026-018" } }, "lead-audit-assignment": { legacyView: "lead-assignment", legacyParams: { auditId: "AUD-2026-001" } }, "lead-checklist-question-assignment": { legacyView: "lead-assignment-questions", legacyParams: { auditId: "AUD-2026-001" } }, "lead-calendar": { legacyView: "calendar", legacyParams: {} }, "lead-messages": { legacyView: "messages", legacyParams: {} }, "lead-analytics-reports": { legacyView: "safety-intelligence", legacyParams: {} }, "lead-settings": { legacyView: "settings", legacyParams: {} },
  "manager-audits": { legacyView: "calendar", legacyParams: {} }, "manager-risk-dashboard": { legacyView: "manager-risk", legacyParams: {} }, "manager-inspection-team": { legacyView: "inspection-team", legacyParams: {} }, "manager-findings-review": { legacyView: "findings-review", legacyParams: {} }, "manager-cap-monitoring": { legacyView: "cap-monitoring", legacyParams: {} }, "manager-checklist-management": { legacyView: "manager-checklists", legacyParams: {} }, "manager-safety-intelligence": { legacyView: "safety-intelligence", legacyParams: {} }, "organization-risk-profile": { legacyView: "org-risk", legacyParams: { orgId: "ORG-XYZ" } }, "manager-ssp-nasp": { legacyView: "ssp-nasp", legacyParams: {} }, "manager-usoap-readiness": { legacyView: "usoap-readiness", legacyParams: {} }, "manager-cap-effectiveness": { legacyView: "cap-effectiveness", legacyParams: {} }, "organization-detail": { legacyView: "org-detail", legacyParams: { orgId: "ORG-XYZ" } }, "inspection-package-builder": { legacyView: "package-builder", legacyParams: {} }, "manager-preliminary-report-review": { legacyView: "audit-reports", legacyParams: { filter: "preliminary" } }, "manager-cap-closure-review": { legacyView: "unit-manager-review", legacyParams: { findingId: "CAB-2026-011" } },
  "new-audit-wizard-1": { legacyView: "wizard", legacyParams: { step: "1" } }, "new-audit-wizard-2": { legacyView: "wizard", legacyParams: { step: "2" } }, "new-audit-wizard-3": { legacyView: "wizard", legacyParams: { step: "3" } }, "new-audit-wizard-4": { legacyView: "wizard", legacyParams: { step: "4" } }, "new-audit-wizard-5": { legacyView: "wizard", legacyParams: { step: "5" } },
  "gm-planning": { legacyView: "planning", legacyParams: {} }, "gm-report-approvals": { legacyView: "gm-report-approvals", legacyParams: {} }, "gm-departments": { legacyView: "gm-departments", legacyParams: {} }, "gm-risk-dashboard": { legacyView: "gm-risk", legacyParams: {} }, "gm-settings": { legacyView: "settings", legacyParams: {} }, "executive-planning": { legacyView: "executive-planning", legacyParams: {} }, "executive-preliminary-reports": { legacyView: "executive-preliminary-reports", legacyParams: {} }, "executive-final-reports": { legacyView: "executive-final-reports", legacyParams: {} }, "executive-report-preview": { legacyView: "executive-report-preview", legacyParams: { reportId: "FR-2026-022" } }, "executive-notifications": { legacyView: "executive-notifications", legacyParams: {} }, "executive-settings": { legacyView: "settings", legacyParams: {} },
  "auditee-inspection-coordination": { legacyView: "service-provider-inspection-coordination", legacyParams: {} }, "auditee-preliminary-reports": { legacyView: "service-provider-preliminary-reports", legacyParams: {} }, "auditee-final-reports": { legacyView: "service-provider-final-reports", legacyParams: {} }, "auditee-report-preview": { legacyView: "service-provider-report-preview", legacyParams: { reportId: "FR-2025-009" } }, "auditee-messages": { legacyView: "messages", legacyParams: {} }, "auditee-documents": { legacyView: "reports", legacyParams: { filter: "documents" } }, "auditee-settings": { legacyView: "settings", legacyParams: {} },
  "admin-regulatory-library": { legacyView: "regulatory-library", legacyParams: {} }, "admin-template-list": { legacyView: "templates", legacyParams: {} }, "admin-question-bank": { legacyView: "question-bank", legacyParams: {} }, "admin-checklist-builder": { legacyView: "checklist-builder", legacyParams: {} }, "admin-version-history": { legacyView: "checklist-versions", legacyParams: {} }, "admin-inspection-package-builder": { legacyView: "package-builder", legacyParams: {} }, "admin-reports": { legacyView: "reports", legacyParams: {} }, "admin-users-roles": { legacyView: "users", legacyParams: {} }, "admin-configurations": { legacyView: "settings", legacyParams: {} }, "admin-organization-master-data": { legacyView: "organizations", legacyParams: {} }, "admin-organization-detail": { legacyView: "org-detail", legacyParams: { orgId: "ORG-XYZ" } }, "admin-audit-log": { legacyView: "auditlog", legacyParams: {} },
};

// This independent table is deliberately not imported from the capture fixtures.
// It makes the source-role equality check observable in the baseline verifier,
// even though the generated manifest records only the legacy view and params.
const legacyRoleBySurface = {
  "role-select": null,
  "inspector-home": "inspector", "audit-detail": "inspector", "checklist-runner": "inspector", "finding-detail": "inspector", "inspector-findings": "inspector", "inspector-messages": "inspector", "inspector-calendar": "inspector", "inspector-reports": "inspector", "closure-report-preview": "inspector", "inspector-assistant": "inspector", "inspector-profile": "inspector",
  "lead-home": "leadInspector", "cap-review": "leadInspector", "lead-preliminary-reports": "leadInspector", "lead-preliminary-report-workflow": "leadInspector", "lead-final-reports": "leadInspector", "lead-final-report-readiness": "leadInspector", "lead-prepare-final-report": "leadInspector", "lead-final-report-document": "leadInspector", "lead-audit-assignment": "leadInspector", "lead-checklist-question-assignment": "leadInspector", "lead-calendar": "leadInspector", "lead-messages": "leadInspector", "lead-analytics-reports": "leadInspector", "lead-settings": "leadInspector",
  "manager-home": "manager", "organization-registry": "manager", "audit-plan": "manager", "evidence-review": "manager", "report-preview": "manager", "manager-audits": "manager", "manager-risk-dashboard": "manager", "manager-inspection-team": "manager", "manager-findings-review": "manager", "manager-cap-monitoring": "manager", "manager-checklist-management": "manager", "manager-safety-intelligence": "manager", "organization-risk-profile": "manager", "manager-ssp-nasp": "manager", "manager-usoap-readiness": "manager", "manager-cap-effectiveness": "manager", "organization-detail": "manager", "inspection-package-builder": "manager", "manager-preliminary-report-review": "manager", "manager-cap-closure-review": "manager", "new-audit-wizard-1": "manager", "new-audit-wizard-2": "manager", "new-audit-wizard-3": "manager", "new-audit-wizard-4": "manager", "new-audit-wizard-5": "manager",
  "gm-home": "gm", "gm-planning": "gm", "gm-report-approvals": "gm", "gm-departments": "gm", "gm-risk-dashboard": "gm", "gm-settings": "gm",
  "finance-home": "finance",
  "executive-home": "executiveDirector", "executive-planning": "executiveDirector", "executive-preliminary-reports": "executiveDirector", "executive-final-reports": "executiveDirector", "executive-report-preview": "executiveDirector", "executive-notifications": "executiveDirector", "executive-settings": "executiveDirector",
  "auditee-home": "auditee", "auditee-inspection-coordination": "auditee", "auditee-preliminary-reports": "auditee", "auditee-final-reports": "auditee", "auditee-report-preview": "auditee", "auditee-messages": "auditee", "auditee-documents": "auditee", "auditee-settings": "auditee",
  "admin-home": "admin", "admin-regulatory-library": "admin", "admin-template-list": "admin", "admin-question-bank": "admin", "admin-checklist-builder": "admin", "admin-version-history": "admin", "admin-inspection-package-builder": "admin", "admin-reports": "admin", "admin-users-roles": "admin", "admin-configurations": "admin", "admin-organization-master-data": "admin", "admin-organization-detail": "admin", "admin-audit-log": "admin",
};

const routeContractSource = readFileSync(resolve(appRoot, "src/app/route-contracts.ts"), "utf8");
const auditSource = JSON.parse(readFileSync(resolve(appRoot, "src/parity/legacy-screen-source.json"), "utf8"));
const roleByAuditSource = {
  Global: null,
  "CAA Inspector": "inspector",
  "Lead Inspector": "leadInspector",
  "Department Manager": "manager",
  "General Manager": "gm",
  Finance: "finance",
  "Executive Director": "executiveDirector",
  Auditee: "auditee",
  "Admin Preview": "admin",
};

const expectedLegacyFixtures = { ...legacyExpectedSurfaces, ...additionalLegacyExpectedSurfaces };
if (Object.keys(expectedLegacyFixtures).length !== 86) {
  fail(`Visual baseline verifier must declare the exact 86 legacy source states; got ${Object.keys(expectedLegacyFixtures).length}.`);
}
if (Object.keys(legacyRoleBySurface).length !== 86) {
  fail(`Visual baseline verifier must declare the exact 86 legacy source roles; got ${Object.keys(legacyRoleBySurface).length}.`);
}
const expectedSurfaces = Object.fromEntries(
  [...routeContractSource.matchAll(/\{ auditId: "([^"]+)", id: "([^"]+)", path: "([^"]+)", requiredRole: (null|"[^"]+")/g)]
    .filter((match) => match[2] !== "post-release-checklist-selection")
    .map((match) => [
    match[2],
    {
      auditId: match[1],
      parityMode: match[1] === "ui-audit-001" ? "strict-shell" : "content-adapted",
      reactPath: match[3],
      requiredRole: match[4] === "null" ? null : match[4].slice(1, -1),
    },
  ]),
);
if (Object.keys(expectedSurfaces).length !== 85) {
  fail(`Visual baseline verifier could not derive the exact 85 legacy-route visual contract; got ${Object.keys(expectedSurfaces).length}.`);
}
for (const source of auditSource) {
  if (source.auditId === "ui-audit-043") continue;
  const surface = Object.values(expectedSurfaces).find((candidate) => candidate.auditId === source.auditId);
  if (!surface) fail(`Audit source ${source.auditId} has no React route contract.`);
  if (surface.requiredRole !== roleByAuditSource[source.role]) {
    fail(`Source-role/route-role mismatch for ${source.auditId}.`);
  }
}
for (const [surfaceId, surface] of Object.entries(expectedSurfaces)) {
  if (!(surfaceId in legacyRoleBySurface)) fail(`Missing expected legacy role for ${surfaceId}.`);
  if (legacyRoleBySurface[surfaceId] !== surface.requiredRole) {
    fail(`Expected legacy role/route-role mismatch for ${surfaceId}.`);
  }
}

const sourceFiles = [
  "index.html",
  "css/styles.css",
  "js/app.js",
  "js/views.js",
  "js/data.js",
];

function fail(message) {
  throw new Error(message);
}

function hashBytes(bytes) {
  return `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
}

function hashFile(relativePath) {
  return hashBytes(readFileSync(resolve(repoRoot, relativePath)));
}

function safeBaselinePath(relativeFile) {
  if (
    relativeFile.startsWith("/") ||
    relativeFile.includes("..") ||
    relativeFile.split(/[\\/]/).some((part) => part === "")
  ) {
    fail(`Unsafe baseline path: ${relativeFile}`);
  }
  const absolute = resolve(baselineDir, relativeFile);
  const rel = relative(baselineDir, absolute);
  if (rel.startsWith("..") || rel === "" || rel.split(sep)[0] === "..") {
    fail(`Baseline path escapes root: ${relativeFile}`);
  }
  return absolute;
}

function listPngFiles(dir, base = dir) {
  if (!existsSync(dir)) return [];
  const files = [];
  for (const entry of readdirSync(dir).sort()) {
    const absolute = resolve(dir, entry);
    const stat = statSync(absolute);
    if (stat.isDirectory()) files.push(...listPngFiles(absolute, base));
    else if (entry.endsWith(".png")) files.push(relative(base, absolute).split(sep).join("/"));
  }
  return files;
}

function assertNotIgnored(absolutePath) {
  const result = spawnSync("git", ["check-ignore", "-q", relative(repoRoot, absolutePath)], {
    cwd: repoRoot,
    stdio: "ignore",
  });
  if (result.status === 0) fail(`Visual baseline path is ignored by git: ${relative(repoRoot, absolutePath)}`);
  if (result.status !== 1) fail(`git check-ignore failed for ${relative(repoRoot, absolutePath)}`);
}

function currentSourceMetadata() {
  const files = {};
  for (const file of sourceFiles) files[file] = hashFile(file);
  return {
    files,
    auditDocument: {
      path: "docs/demo-evidence/UI_SCREEN_AUDIT_2026-07-19.md",
      sha256: hashFile("docs/demo-evidence/UI_SCREEN_AUDIT_2026-07-19.md"),
    },
    packageLockSha256: hashFile("apps/web/package-lock.json"),
  };
}

async function currentEnvironmentMetadata() {
  const playwrightPackage = JSON.parse(
    readFileSync(resolve(appRoot, "node_modules/@playwright/test/package.json"), "utf8"),
  );
  const browser = await chromium.launch({ headless: true });
  try {
    return {
      playwrightVersion: playwrightPackage.version,
      chromiumVersion: browser.version(),
      nodeVersion: process.versions.node,
      platform: process.platform,
      arch: process.arch,
      osRelease: release(),
    };
  } finally {
    await browser.close();
  }
}

function assertRecordEqual(label, actual, expected) {
  const actualKeys = Object.keys(actual).sort();
  const expectedKeys = Object.keys(expected).sort();
  if (actualKeys.join("\n") !== expectedKeys.join("\n")) {
    fail(`${label} metadata keys mismatch.`);
  }
  for (const key of expectedKeys) {
    if (actual[key] !== expected[key]) fail(`${label} metadata mismatch for ${key}.`);
  }
}

function assertObjectEqual(label, actual, expected) {
  const actualJson = JSON.stringify(actual);
  const expectedJson = JSON.stringify(expected);
  if (actualJson !== expectedJson) fail(`${label} mismatch.`);
}

async function verify() {
  if (process.env.AVIA_UPDATE_LEGACY_BASELINES === "1") {
    fail("verify-visual-baselines.mjs is read-only and must not run in baseline update mode.");
  }
  if (!existsSync(manifestPath)) fail("Missing visual baseline manifest.");
  assertNotIgnored(manifestPath);

  const manifest = JSON.parse(readFileSync(manifestPath, "utf8"));
  if (manifest.schemaVersion !== 1) fail("Unsupported baseline manifest schema.");
  if (manifest.generatedAt !== fixedTimeIso) fail("Baseline manifest generatedAt is not deterministic.");
  if (manifest.baselineVersion !== baselineVersion) fail(`Unexpected baseline version: ${manifest.baselineVersion}`);
  if (manifest.surfaceCount !== 86) fail("Baseline manifest must record exactly 86 surfaces.");
  if (manifest.viewportCount !== 3) fail("Baseline manifest must record exactly 3 viewports.");
  if (!Array.isArray(manifest.items) || manifest.items.length !== 258) {
    fail(`Baseline manifest must list exactly 258 PNGs; got ${manifest.items?.length ?? 0}.`);
  }
  if (!/^[0-9a-f]{40}$/.test(manifest.source?.commit ?? "")) {
    fail("Baseline manifest source commit must be a full git SHA.");
  }

  const expectedSource = currentSourceMetadata();
  assertRecordEqual("source", manifest.source.files, expectedSource.files);
  if (manifest.source.auditDocument.path !== expectedSource.auditDocument.path) {
    fail("source metadata mismatch for audit document path.");
  }
  if (manifest.source.auditDocument.sha256 !== expectedSource.auditDocument.sha256) {
    fail("source metadata mismatch for audit document hash.");
  }
  if (manifest.source.packageLockSha256 !== expectedSource.packageLockSha256) {
    fail("package-lock metadata mismatch.");
  }

  const expectedEnvironment = await currentEnvironmentMetadata();
  for (const key of Object.keys(expectedEnvironment)) {
    if (manifest.environment[key] !== expectedEnvironment[key]) {
      fail(`${key} metadata mismatch.`);
    }
  }

  const seenKeys = new Set();
  const seenFiles = new Set();
  for (const item of manifest.items) {
    const key = `${item.surfaceId}/${item.viewport}`;
    if (seenKeys.has(key)) fail(`Duplicate baseline item: ${key}`);
    seenKeys.add(key);
    if (seenFiles.has(item.file)) fail(`Duplicate baseline file: ${item.file}`);
    seenFiles.add(item.file);

    const surface = expectedSurfaces[item.surfaceId];
    const legacy = expectedLegacyFixtures[item.surfaceId];
    if (!surface) fail(`Unexpected baseline surface: ${item.surfaceId}`);
    if (!legacy) fail(`Missing expected legacy source state for ${item.surfaceId}`);
    if (!viewports[item.viewport]) fail(`Unexpected baseline viewport: ${item.viewport}`);
    if (item.auditId !== surface.auditId) fail(`Route mismatch for ${item.surfaceId}: audit id.`);
    if (item.parityMode !== surface.parityMode) fail(`Route mismatch for ${item.surfaceId}: parity mode.`);
    if (item.sourceRoute.reactPath !== surface.reactPath) fail(`Route mismatch for ${item.surfaceId}: React path.`);
    if (item.sourceRoute.legacyView !== legacy.legacyView) fail(`Route mismatch for ${item.surfaceId}: legacy view.`);
    assertObjectEqual(`Route mismatch for ${item.surfaceId}: legacy params`, item.sourceRoute.legacyParams, legacy.legacyParams);
    assertObjectEqual(`Viewport metadata for ${key}`, item.viewportSize, viewports[item.viewport]);

    const absolute = safeBaselinePath(item.file);
    assertNotIgnored(absolute);
    if (!existsSync(absolute)) fail(`Missing baseline PNG: ${item.file}`);
    const realHash = hashBytes(readFileSync(absolute));
    if (realHash !== item.sha256) fail(`Baseline hash drift for ${item.file}.`);
  }

  const expectedKeys = [];
  for (const surfaceId of Object.keys(expectedSurfaces)) {
    for (const viewportId of Object.keys(viewports)) expectedKeys.push(`${surfaceId}/${viewportId}`);
  }
  expectedKeys.sort();
  const actualKeys = [...seenKeys].sort();
  if (actualKeys.join("\n") !== expectedKeys.join("\n")) {
    fail("Baseline manifest does not contain the exact 86x3 surface matrix.");
  }

  const extraFiles = listPngFiles(baselineDir).filter((file) => !seenFiles.has(file));
  if (extraFiles.length) fail(`Unexpected extra baseline PNG file(s): ${extraFiles.join(", ")}`);

  process.stdout.write(`Verified ${manifest.items.length} visual baseline PNGs in ${relative(repoRoot, baselineDir)}.\n`);
}

verify().catch((error) => {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`);
  process.exitCode = 1;
});
