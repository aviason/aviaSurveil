import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const ledgerPath = path.join(repositoryRoot, "tests/parity/behavior-ledger.json");

function readLedger() {
  assert.ok(fs.existsSync(ledgerPath), "Behavior ledger is missing");
  return JSON.parse(fs.readFileSync(ledgerPath, "utf8"));
}

function referencedTests(entry) {
  return [entry.legacyTest, entry.reactTest].flat().filter(Boolean);
}

test("the behavior ledger covers the authorized first-production route families", () => {
  const ledger = readLedger();
  assert.equal(ledger.product, "AviaSurveil360");
  assert.equal(ledger.version, 4);
  assert.equal(ledger.legacyRemovalAllowed, false);
  assert.ok(ledger.entries.length >= 15, "Route actions must extend the canonical scenario and role entries");

  const canonicalScenario = ledger.entries.find((entry) => entry.id === "canonical-cabin-scenario");
  assert.ok(canonicalScenario, "Canonical Cabin scenario entry is required");
  assert.equal(canonicalScenario.classification, "first-production");
  assert.equal(canonicalScenario.expectedStatus, "CLOSED");
  assert.equal(
    canonicalScenario.visibilityInvariant,
    "auditee-never-receives-internal-caa-note",
  );
  assert.ok(
    canonicalScenario.reactTest.includes("apps/web/tests/e2e/canonical-scenario.spec.ts"),
    "The canonical React entry must reference its normalized Playwright transcript",
  );

  const roleEntries = ledger.entries.filter((entry) => entry.action === "enter-role");
  assert.deepEqual(
    roleEntries.map((entry) => `${entry.role}:${entry.route}`),
    [
      "inspector:inspector-assignments",
      "leadInspector:lead-review",
      "manager:dashboard",
      "gm:gm-dashboard",
      "finance:finance-review",
      "executiveDirector:executive-dashboard",
      "auditee:service-provider-cap",
      "admin:templates",
    ],
  );

  const routeActions = new Set(ledger.entries.map((entry) => entry.action));
  for (const action of [
    "view-organization-registry",
    "approve-planning-budget",
    "forward-plan-for-final-approval",
    "approve-plan",
    "release-plan",
    "view-versioned-configuration-and-audit-trail",
  ]) {
    assert.ok(routeActions.has(action), `Missing executable route action: ${action}`);
  }
});

test("the parity ledger freezes the 86/0 route scope and visible action ownership", () => {
  const ledger = readLedger();
  const expectedInteractionVerifiedSurfaceIds = ledger.reactScope.reactParitySurfaceIds;
  assert.equal(ledger.reactScope.reactParitySurfaceIds.length, 86);
  assert.equal(new Set(ledger.reactScope.reactParitySurfaceIds).size, 86);
  assert.equal(ledger.reactScope.legacyOnlyRows, 0);
  assert.equal(ledger.reactScope.legacyOnlyReactPath, "not-applicable");
  assert.equal(ledger.reactScope.legacyRemovalAllowed, false);
  assert.equal(ledger.reactScope.interactionVerifiedSurfaceIds.length, 86);
  assert.equal(ledger.reactScope.presentationCorrectionPendingSurfaceIds.length, 0);
  assert.equal(ledger.reactScope.routeContractOnlySurfaceIds.length, 0);
  assert.deepEqual(ledger.reactScope.interactionVerifiedSurfaceIds, expectedInteractionVerifiedSurfaceIds);
  assert.deepEqual(ledger.reactScope.presentationCorrectionPendingSurfaceIds, []);
  assert.deepEqual(ledger.reactScope.routeContractOnlySurfaceIds, []);
  assert.deepEqual(ledger.interactionMatrix, {
    viewports: ["desktop", "tablet", "mobile"],
    responsiveRouteChecks: 258,
    actionInventories: 258,
    evidence: [
      "apps/web/tests/e2e/full-route-accessibility.spec.ts",
      "apps/web/tests/e2e/visible-action-contract.spec.ts",
    ],
  });
  const actionEvidenceSurfaceIds = ledger.actionEvidenceGroups.flatMap((group) => group.surfaceIds);
  assert.equal(actionEvidenceSurfaceIds.length, 86);
  assert.equal(new Set(actionEvidenceSurfaceIds).size, 86);
  assert.deepEqual(actionEvidenceSurfaceIds, ledger.reactScope.reactParitySurfaceIds);
  for (const group of ledger.actionEvidenceGroups) {
    assert.ok(group.surfaceIds.length > 0);
    assert.ok(fs.existsSync(path.join(repositoryRoot, group.evidence)));
  }
  assert.ok(Array.isArray(ledger.actionEvidence), "Per-action evidence ledger is required");
  assert.ok(ledger.actionEvidence.length > 600, "Per-action evidence must enumerate every route and shared-shell control");
  const actionEvidenceKeys = new Set();
  const perActionSurfaceIds = new Set();
  for (const action of ledger.actionEvidence) {
    assert.ok(action.surfaceId === "*" || ledger.reactScope.reactParitySurfaceIds.includes(action.surfaceId));
    assert.ok(["route", "shell", "mobile-navigation"].includes(action.scope));
    assert.ok(Array.isArray(action.viewports) && action.viewports.length > 0);
    assert.ok(action.viewports.every((viewport) => ledger.interactionMatrix.viewports.includes(viewport)));
    if (action.surfaceId === "*") assert.notEqual(action.scope, "route");
    assert.ok(action.controlKey);
    assert.ok(action.durableEffect.length > 20);
    assert.ok(fs.existsSync(path.join(repositoryRoot, action.evidence)));
    assert.ok(action.assertion);
    if (action.scope === "route") {
      const expectedExecutableAssertion = {
        "verified-form-behavior": "assertNativeFormControlOutcome",
        "verified-visible-state": "assertAccessibleStateOutcome",
        "verified-tab-state": "assertAccessibleStateOutcome",
        "verified-controlled-state": "assertControlledSurfaceOutcome",
      }[action.boundary];
      if (expectedExecutableAssertion) {
        assert.equal(
          action.assertion,
          expectedExecutableAssertion,
          `${action.surfaceId}/${action.controlKey} must name its exact executable postcondition`,
        );
      }
    }
    const evidenceSource = fs.readFileSync(path.join(repositoryRoot, action.evidence), "utf8");
    assert.match(evidenceSource, new RegExp(action.assertion));
    const key = `${action.surfaceId}:${action.scope}:${action.controlKey}`;
    assert.ok(!actionEvidenceKeys.has(key), `Duplicate per-action evidence: ${key}`);
    actionEvidenceKeys.add(key);
    if (action.surfaceId !== "*") perActionSurfaceIds.add(action.surfaceId);
  }
  assert.deepEqual(
    new Set(ledger.reactScope.routeControlFreeSurfaceIds),
    new Set(["executive-preliminary-reports", "admin-configurations"]),
  );
  assert.deepEqual(
    new Set([...perActionSurfaceIds, ...ledger.reactScope.routeControlFreeSurfaceIds]),
    new Set(ledger.reactScope.reactParitySurfaceIds),
  );
  assert.equal(
    [...ledger.reactScope.routeControlFreeSurfaceIds].filter((surfaceId) =>
      perActionSurfaceIds.has(surfaceId)
    ).length,
    0,
    "Route-control-free surfaces must not claim route-action evidence",
  );
  const evidenceCategories = [
    ...ledger.reactScope.interactionVerifiedSurfaceIds,
    ...ledger.reactScope.presentationCorrectionPendingSurfaceIds,
    ...ledger.reactScope.routeContractOnlySurfaceIds,
  ];
  assert.equal(new Set(evidenceCategories).size, 86, "Evidence categories must be disjoint");
  assert.deepEqual(new Set(evidenceCategories), new Set(ledger.reactScope.reactParitySurfaceIds));
  assert.ok(ledger.visibleActionOwnership.length >= 3);
  for (const ownership of ledger.visibleActionOwnership) {
    assert.ok(ownership.id);
    assert.ok(ownership.surfaceIds.length > 0);
    assert.doesNotThrow(() => new RegExp(ownership.namePattern));
    assert.ok([
      "typed-mutation",
      "navigation",
      "selection-or-filter",
      "browser-local-artifact",
    ].includes(ownership.boundary));
    assert.ok(ownership.durableEffect.length > 20);
    assert.ok(fs.existsSync(path.join(repositoryRoot, ownership.evidence)));
    if (ownership.id === "declared-route-and-workflow-actions") {
      assert.deepEqual(ownership.surfaceIds, ledger.reactScope.interactionVerifiedSurfaceIds);
    }
    if (ownership.id === "shell-reset" || ownership.id === "shell-role-exit") {
      assert.deepEqual(
        ownership.surfaceIds,
        ledger.reactScope.interactionVerifiedSurfaceIds.filter((surfaceId) => surfaceId !== "role-select"),
      );
    }
  }
});

test("every ledger entry is executable and declares the parity contract", () => {
  const ledger = readLedger();
  const requiredFields = [
    "id",
    "classification",
    "role",
    "route",
    "action",
    "entityIdRule",
    "expectedStatus",
    "expectedOwner",
    "visibilityInvariant",
    "legacyTest",
    "reactTest",
    "acceptedDifference",
  ];

  for (const entry of ledger.entries) {
    for (const field of requiredFields) {
      assert.ok(Object.hasOwn(entry, field), `${entry.id ?? "entry"} is missing ${field}`);
    }
    assert.ok(["first-production", "later", "demo-only"].includes(entry.classification));
    for (const relativePath of referencedTests(entry)) {
      assert.ok(
        fs.existsSync(path.join(repositoryRoot, relativePath)),
        `${entry.id} references a missing executable test: ${relativePath}`,
      );
    }
  }
});

test("the legacy Vanilla demo remains the removal-blocking behavior oracle", () => {
  const ledger = readLedger();
  assert.equal(ledger.legacyRemovalAllowed, false);
  for (const requiredPath of ["index.html", "css/styles.css", "js/app.js", "js/data.js"]) {
    assert.ok(fs.existsSync(path.join(repositoryRoot, requiredPath)), `${requiredPath} must remain intact`);
  }
});

test("the full-platform parity gate freezes ten unskipped mock and HTTP scenario families", () => {
  const contractPath = path.join(
    repositoryRoot,
    "apps/web/tests/contract/full-platform-backend.contract.ts",
  );
  const scenarioPath = path.join(
    repositoryRoot,
    "apps/web/tests/e2e/full-platform-scenarios.spec.ts",
  );
  const resetCommandPath = path.join(
    repositoryRoot,
    "apps/api/cmd/test-profile-reset/main.go",
  );
  const resetScriptPath = path.join(repositoryRoot, "scripts/reset-test-profile.sh");
  const denialsPath = path.join(
    repositoryRoot,
    "apps/api/tests/integration/full_platform_denials_test.go",
  );
  const boundaryPath = path.join(
    repositoryRoot,
    "apps/api/internal/httpapi/test_profile_boundary_test.go",
  );
  for (const requiredPath of [
    contractPath,
    scenarioPath,
    resetCommandPath,
    resetScriptPath,
    denialsPath,
    boundaryPath,
  ]) {
    assert.ok(fs.existsSync(requiredPath), `Missing Task 11 parity surface: ${requiredPath}`);
  }

  const contract = fs.readFileSync(contractPath, "utf8");
  for (const family of [
    "routine-inspection-to-closure",
    "ad-hoc-planning-to-assignment",
    "checklist-and-potential-finding-authority",
    "cap-evidence-and-closure-authority",
    "preliminary-and-final-report-authority",
    "configuration-and-immutable-package-snapshot",
    "organization-and-platform-projections",
    "advisory-management-projections",
    "offline-causal-sync-and-session-boundaries",
    "advisory-draft-without-canonical-mutation",
  ]) {
    assert.match(contract, new RegExp(`["']${family}["']`), `Missing scenario family ${family}`);
  }
  for (const transcriptField of [
    "entityIds",
    "revisions",
    "statuses",
    "owners",
    "roles",
    "organizationIds",
    "versions",
    "auditEventTypes",
    "notificationJobs",
    "documentJobs",
    "denials",
    "dashboardProjections",
  ]) {
    assert.match(contract, new RegExp(`\\b${transcriptField}\\b`), `Missing transcript field ${transcriptField}`);
  }
  assert.doesNotMatch(contract, /\.skip\b|test\.fixme\b|describe\.skip\b/);

  const scenario = fs.readFileSync(scenarioPath, "utf8");
  assert.match(scenario, /runFullPlatformScenarios/);
  assert.match(scenario, /FULL_PLATFORM_EXPECTED_TRANSCRIPT/);
  assert.doesNotMatch(scenario, /\.skip\b|test\.fixme\b/);

  const playwright = fs.readFileSync(path.join(repositoryRoot, "apps/web/playwright.config.ts"), "utf8");
  const scenarioMatches = playwright.match(/e2e\/full-platform-scenarios\.spec\.ts/g) ?? [];
  assert.equal(scenarioMatches.length, 2, "Full-platform scenarios must run in mock and HTTP projects");

  const resetScript = fs.readFileSync(resetScriptPath, "utf8");
  assert.match(resetScript, /cmd\/test-profile-reset/);
  const boundary = fs.readFileSync(boundaryPath, "utf8");
  assert.match(boundary, /StatusNotFound/);
  assert.match(boundary, /\/__test\/reset/);
});
