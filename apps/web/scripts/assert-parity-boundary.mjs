#!/usr/bin/env node
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import {
  assertAppShellArtifact,
  BRAND_ASSET_BASENAMES,
} from "./assert-app-shell-artifact.mjs";
import { assertHttpArtifact } from "./assert-http-artifact.mjs";

const scriptPath = fileURLToPath(import.meta.url);
const defaultRepositoryRoot = path.resolve(path.dirname(scriptPath), "../../..");

const EXPECTED_ROUTE_COUNT = 85;
const EXPECTED_VISUAL_PAIR_COUNT = 255;

const HTTP_FORBIDDEN_INPUTS = [
  /[/\\]src[/\\]mock[/\\]/i,
  /[/\\]seed-data(?:\.[cm]?[jt]s)?$/i,
  /[/\\](?:entry[/\\])?http-test\.[cm]?[jt]sx?$/i,
  /[/\\]test-profile[/\\]/i,
];

function filesBelow(directory, predicate) {
  if (!fs.existsSync(directory)) return [];
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const absolute = path.join(directory, entry.name);
    if (entry.isDirectory()) return filesBelow(absolute, predicate);
    return predicate(absolute) ? [absolute] : [];
  });
}

function normalized(relativePath) {
  return relativePath.split(path.sep).join("/");
}

function mutateInteractionSource(relativePath, source, mutation) {
  if (
    mutation === "remove-action-harness-contract" &&
    relativePath.endsWith("tests/e2e/visible-action-contract.spec.ts")
  ) {
    return source.replace("        await assertDurableControlOutcome(page, surface, control);", "");
  }
  if (
    mutation === "skip-stateful-control-execution" &&
    relativePath.endsWith("tests/e2e/visible-action-contract.spec.ts")
  ) {
    return source.replace(
      "    const routeCommands = controls.filter(isExecutableRouteControl);",
      "    const routeCommands = controls.filter((control) => control.tag === \"BUTTON\" && !hasAccessibleState(control) && !control.ariaControls);",
    );
  }
  if (
    mutation === "skip-mobile-command-execution" &&
    relativePath.endsWith("tests/e2e/visible-action-contract.spec.ts")
  ) {
    return source.replace(
      "  const executionViewports = [VISUAL_VIEWPORTS[0], VISUAL_VIEWPORTS[2]];",
      "  const executionViewports = [VISUAL_VIEWPORTS[0]];",
    );
  }
  if (
    mutation === "toast-only-action" &&
    relativePath.endsWith("tests/e2e/visible-action-contract.spec.ts")
  ) {
    return source.replace(
      "\"[role='status']:not([data-durable-outcome])\",",
      "\"[data-transient-status-never-matches]\",",
    );
  }
  if (
    mutation === "unlabelled-control" &&
    relativePath.endsWith("tests/e2e/visible-action-contract.spec.ts")
  ) {
    return source.replace("expect.soft(unnamed,", "expect.soft([],");
  }
  if (
    mutation === "missing-disabled-reason" &&
    relativePath.endsWith("tests/e2e/visible-action-contract.spec.ts")
  ) {
    return source.replace(
      'expect(normalize(current!.disabledReason)).not.toBe("");',
      "expect(true).toBe(true);",
    );
  }
  if (
    mutation === "duplicate-accessible-navigation" &&
    relativePath.endsWith("tests/e2e/full-route-accessibility.spec.ts")
  ) {
    return source.replace("  await expect(navigation).toHaveCount(1);", "");
  }
  if (
    mutation === "broken-deep-link" &&
    relativePath.endsWith("tests/e2e/full-route-accessibility.spec.ts")
  ) {
    return source.replace(
      "expect.soft(pathname(page), `${surface.id}/${viewport.id} preserves its exact deep link`).toBe(surface.reactPath);",
      "expect.soft(pathname(page), `${surface.id}/${viewport.id} preserves its exact deep link`).toBe(pathname(page));",
    );
  }
  if (
    mutation === "missing-mobile-viewport" &&
    relativePath.endsWith("tests/e2e/support/legacy-parity-fixtures.ts")
  ) {
    return source.replace('  { id: "mobile", width: 390, height: 844 },\n', "");
  }
  if (
    mutation === "remove-focus-indicator-contract" &&
    relativePath.endsWith("src/styles/reset.css")
  ) {
    return source.replace(
      /button:focus-visible,[\s\S]*?outline-offset: 2px;\n}\n/,
      "",
    );
  }
  if (
    mutation === "fake-dropdown" &&
    relativePath.endsWith("src/features/reports/report-preview-page.tsx")
  ) {
    return source.replace(
      '<select aria-label="Report type"',
      '<div role="combobox" aria-label="Report type"',
    );
  }
  return source;
}

function assertInteractionSourceBoundary(repositoryRoot, mutation) {
  const requiredPaths = [
    "apps/web/tests/e2e/visible-action-contract.spec.ts",
    "apps/web/tests/e2e/full-route-accessibility.spec.ts",
    "apps/web/tests/e2e/support/legacy-parity-fixtures.ts",
    "apps/web/src/styles/reset.css",
  ];
  const productionFiles = filesBelow(path.join(repositoryRoot, "apps/web/src"), (file) =>
    /\.[cm]?[jt]sx?$/.test(file) && !/\.(?:test|spec)\.[cm]?[jt]sx?$/.test(file),
  );
  const sources = new Map(
    [...requiredPaths.map((relativePath) => path.join(repositoryRoot, relativePath)), ...productionFiles]
      .map((absolute) => {
        const relativePath = normalized(path.relative(repositoryRoot, absolute));
        assert.ok(fs.existsSync(absolute), `Interaction boundary source is missing: ${relativePath}`);
        const source = fs.readFileSync(absolute, "utf8");
        return [relativePath, mutateInteractionSource(relativePath, source, mutation)];
      }),
  );

  const visibleActionSource = sources.get("apps/web/tests/e2e/visible-action-contract.spec.ts");
  const accessibilitySource = sources.get("apps/web/tests/e2e/full-route-accessibility.spec.ts");
  const fixtureSource = sources.get("apps/web/tests/e2e/support/legacy-parity-fixtures.ts");
  const resetSource = sources.get("apps/web/src/styles/reset.css");
  assert.ok(visibleActionSource, "Visible action harness source is missing.");
  assert.ok(accessibilitySource, "Full-route accessibility harness source is missing.");
  assert.ok(fixtureSource, "Responsive route fixture source is missing.");
  assert.ok(resetSource, "Shared focus-style source is missing.");

  assert.ok(
    visibleActionSource.includes("await assertDurableControlOutcome(page, surface, control);"),
    "Visible action harness must execute every active route command's durable outcome.",
  );
  assert.ok(
    visibleActionSource.includes("const routeCommands = controls.filter(isExecutableRouteControl);") &&
      visibleActionSource.includes("assertNativeFormControlOutcome") &&
      visibleActionSource.includes("assertAccessibleStateOutcome") &&
      visibleActionSource.includes("assertControlledSurfaceOutcome"),
    "Visible action harness must execute stateful and form controls with exact postconditions.",
  );
  assert.ok(
    visibleActionSource.includes(
      "const executionViewports = [VISUAL_VIEWPORTS[0], VISUAL_VIEWPORTS[2]];",
    ),
    "Visible action harness must execute mobile-only route controls.",
  );
  assert.ok(
    visibleActionSource.includes("\"[role='status']:not([data-durable-outcome])\","),
    "Visible action contract rejects a toast-only action.",
  );
  assert.ok(
    visibleActionSource.includes("expect.soft(unnamed,"),
    "Every visible control must have an accessible name.",
  );
  assert.ok(
    visibleActionSource.includes('expect(normalize(current!.disabledReason)).not.toBe("");'),
    "Every disabled control must expose a record-specific disabled reason.",
  );
  assert.ok(
    accessibilitySource.includes("  await expect(navigation).toHaveCount(1);"),
    "Each routed workspace must expose exactly one accessible primary navigation.",
  );
  assert.ok(
    accessibilitySource.includes(
      "expect.soft(pathname(page), `${surface.id}/${viewport.id} preserves its exact deep link`).toBe(surface.reactPath);",
    ),
    "Every declared deep link must resolve to its exact path.",
  );
  assert.ok(
    accessibilitySource.includes("focusIndicatorVisible") &&
      /:focus-visible[\s\S]*?outline:\s*[1-9]/.test(resetSource),
    "Every sequential target must expose a visible keyboard focus indicator.",
  );

  const viewportBlock = fixtureSource.match(/export const VISUAL_VIEWPORTS = \[([\s\S]*?)\] as const;/);
  assert.ok(viewportBlock, "The interaction viewport registry is missing.");
  const viewportIds = [...viewportBlock[1].matchAll(/\bid:\s*"([^"]+)"/g)].map((match) => match[1]);
  assert.deepEqual(
    viewportIds,
    ["desktop", "tablet", "mobile"],
    "The interaction matrix must include the mobile viewport.",
  );

  for (const [relativePath, source] of sources) {
    if (!relativePath.startsWith("apps/web/src/")) continue;
    assert.doesNotMatch(
      source,
      /<(?:button|div|input|span)\b[^>]*\brole\s*=\s*["'](?:combobox|listbox)["']/i,
      `Dropdown controls must use native select semantics: ${relativePath}`,
    );
  }
}

function mutateSource(relativePath, source, mutation) {
  if (mutation === "missing-route" && relativePath.endsWith("src/app/route-contracts.ts")) {
    return source.replace(/  \{ auditId: "ui-audit-086"[^\n]*\n/, "");
  }
  if (mutation === "remove-http-profile" && relativePath.endsWith("src/app/route-contracts.ts")) {
    return source.replace('availableProfiles: ["demo", "http"],', 'availableProfiles: ["demo"],');
  }
  if (mutation === "restore-blocked-profile-reason" && relativePath.endsWith("src/app/route-contracts.ts")) {
    return source.replace(
      'availableProfiles: ["demo", "http"],',
      'availableProfiles: ["demo", "http"],\n  blockedProfileReason: "Stale HTTP block.",',
    );
  }
  if (mutation === "undeclared-route" && relativePath.endsWith("src/app/router.tsx")) {
    return `${source}\nconst boundaryRouteMutation = <Route path=\"/scope-leak\" element={<div />} />;\n`;
  }
  if (mutation === "inert-button" && relativePath.endsWith("src/app/router.tsx")) {
    return `${source}\nconst boundaryButtonMutation = <button type=\"button\">Do nothing</button>;\n`;
  }
  if (mutation === "broad-root-import" && relativePath.endsWith("src/app/router.tsx")) {
    return `${source}\nimport \"../../../../css/styles.css\";\n`;
  }
  return source;
}

function mutateVisualSource(source, mutation) {
  if (mutation === "remove-shell-assertion") {
    return source.replace('await expect(page.locator(".workspace-sidebar")).toBeVisible();', "");
  }
  if (mutation === "remove-content-assertion") {
    return source.replace('await expect(page.locator(".workbench-page-header")).toBeVisible();', "");
  }
  if (mutation === "compressed-byte-comparator") {
    return source.replaceAll("compareVisualFrames", "byteDiffRatio");
  }
  if (mutation === "skip-viewport") {
    return source.replace(
      "for (const viewport of VISUAL_VIEWPORTS)",
      "for (const viewport of VISUAL_VIEWPORTS.slice(0, 2))",
    );
  }
  if (mutation === "remove-candidate-attachment") {
    return source.replace(
      'testInfo.attach("react-candidate-viewport"',
      'testInfo.attach("removed-react-candidate-viewport"',
    );
  }
  if (mutation === "remove-result-attachment") {
    return source.replace(
      'testInfo.attach("decoded-pixel-region-results"',
      'testInfo.attach("removed-decoded-pixel-region-results"',
    );
  }
  return source;
}

function extractRouteContractPaths(source) {
  return [...source.matchAll(/\bpath:\s*"([^"]+)"/g)].map((match) => match[1]);
}

function assertSourceBoundary(repositoryRoot, mutation) {
  const sourceRoot = path.join(repositoryRoot, "apps/web/src");
  const sourceFiles = filesBelow(sourceRoot, (file) =>
    /\.[cm]?[jt]sx?$/.test(file) && !/\.(?:test|spec)\.[cm]?[jt]sx?$/.test(file),
  );
  const sources = new Map(sourceFiles.map((absolute) => {
    const relativePath = normalized(path.relative(repositoryRoot, absolute));
    const original = fs.readFileSync(absolute, "utf8");
    return [relativePath, mutateSource(relativePath, original, mutation)];
  }));

  const routeContracts = sources.get("apps/web/src/app/route-contracts.ts");
  const router = sources.get("apps/web/src/app/router.tsx");
  assert.ok(routeContracts, "Typed React route registry is missing.");
  assert.ok(router, "React router source is missing.");
  const declaredRoutePaths = extractRouteContractPaths(routeContracts);
  assert.equal(
    declaredRoutePaths.length,
    EXPECTED_ROUTE_COUNT,
    "React route registry must remain the exact ordered 85-surface set.",
  );
  assert.equal(new Set(declaredRoutePaths).size, EXPECTED_ROUTE_COUNT, "React route registry contains duplicate paths.");
  assert.match(
    routeContracts,
    /availableProfiles:\s*\["demo",\s*"http"\]/,
    "All 85 routes must be dual-profile.",
  );
  assert.doesNotMatch(
    routeContracts,
    /blockedProfileReason:\s*["']/,
    "Activated routes must not retain a blocked profile reason.",
  );

  const declaredPaths = new Set(declaredRoutePaths);
  for (const match of router.matchAll(/<Route\b[^>]*\bpath="([^"]+)"/g)) {
    assert.ok(
      match[1] === "*" || declaredPaths.has(match[1]),
      `Router contains undeclared React path: ${match[1]}`,
    );
  }
  assert.match(router, /REACT_ROUTE_CONTRACTS\.filter\(/, "Routes must be generated from the declared registry.");
  assert.doesNotMatch(
    router,
    /RoleEntryPlaceholder|candidate React entry route|coming soon/i,
    "Router contains a visible placeholder route or label.",
  );

  for (const [relativePath, source] of sources) {
    assert.doesNotMatch(
      source,
      /(?:from\s*|import\s*)["'][^"']*(?:[/\\]css[/\\]styles\.css|[/\\]js[/\\])[^"']*["']/i,
      `React source imports protected root runtime code: ${relativePath}`,
    );
    for (const openingTag of source.matchAll(/<button\b([^>]*)>/g)) {
      const attributes = openingTag[1];
      const actionable = /\bon[A-Z]\w*\s*=|\bdisabled(?:\s*=|\s|$)|\btype\s*=\s*["'](?:submit|reset)["']|\bformAction\s*=|\{/i.test(attributes);
      assert.ok(actionable, `React source contains an inert button in ${relativePath}: ${openingTag[0]}`);
    }
  }

  const brandRegistrySource = sources.get("apps/web/src/ui/brand-assets.ts");
  assert.ok(brandRegistrySource, "Semantic brand registry is missing.");
  for (const basename of BRAND_ASSET_BASENAMES) {
    assert.match(
      brandRegistrySource,
      new RegExp(`/${basename.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\\.`),
      `Semantic brand registry is missing ${basename}.`,
    );
  }
}

export function assertBuildInputBoundary(profile, inputs) {
  assert.ok(Array.isArray(inputs), `${profile} build-input inventory must be an array.`);
  for (const input of inputs) {
    assert.doesNotMatch(input, /[/\\]css[/\\]styles\.css$/i, `${profile} build imports root CSS: ${input}`);
    assert.doesNotMatch(input, /[/\\]js[/\\]/i, `${profile} build imports root JavaScript: ${input}`);
    if (profile === "http") {
      for (const forbidden of HTTP_FORBIDDEN_INPUTS) {
        assert.doesNotMatch(input, forbidden, `HTTP build imports forbidden mock/test input: ${input}`);
      }
    }
  }
}

function assertBuildBoundary(repositoryRoot, mutation) {
  for (const profile of ["demo", "http"]) {
    const artifactRoot = path.join(repositoryRoot, `apps/web/dist/${profile}`);
    assertAppShellArtifact(artifactRoot);
    const inputsPath = path.join(artifactRoot, "build-inputs.json");
    const viteManifestPath = path.join(artifactRoot, ".vite/manifest.json");
    assert.ok(fs.existsSync(inputsPath), `${profile} build-inputs.json is missing.`);
    assert.ok(fs.existsSync(viteManifestPath), `${profile} Vite manifest is missing.`);
    const buildInputs = JSON.parse(fs.readFileSync(inputsPath, "utf8"));
    assert.equal(buildInputs.profile, profile, `${profile} input profile is stale.`);
    const inputs = [...buildInputs.inputs];
    if (profile === "http" && mutation === "http-mock-import") {
      inputs.push(path.join(repositoryRoot, "apps/web/src/mock/seed-data.ts"));
    }
    assertBuildInputBoundary(profile, inputs);
    const viteManifest = fs.readFileSync(viteManifestPath, "utf8");
    assert.doesNotMatch(viteManifest, /(?:src[/\\]mock|seed-data|css[/\\]styles\.css|[/\\]js[/\\])/i);
  }
  assertHttpArtifact(path.join(repositoryRoot, "apps/web/dist/http"));
}

export function assertVisualHarnessSource(source) {
  assert.doesNotMatch(source, /byteDiffRatio/, "Visual parity must compare decoded pixels, not compressed PNG bytes.");
  for (const required of [
    "decodePngFrame",
    "compareVisualFrames",
    "visualComparisonRegions",
    "const surfaces = resolveFocusedSurfaces()",
    "for (const viewport of VISUAL_VIEWPORTS)",
    "for (const surface of surfaces)",
    'testInfo.attach("react-candidate-viewport"',
    'testInfo.attach("decoded-pixel-region-results"',
    'await expect(page.locator(".workspace-sidebar")).toBeVisible();',
    'await expect(page.locator(".application-topbar")).toBeVisible();',
    'await expect(page.locator(".workspace-content")).toBeVisible();',
    'await expect(page.locator(".workbench-page-header")).toBeVisible();',
    "const expectedVisualPairCount = VISUAL_SURFACES.length * VISUAL_VIEWPORTS.length;",
    "expect(expectedVisualPairCount).toBe(255);",
    "assertVisualPairAttachments(testInfo.attachments);",
  ]) {
    assert.ok(source.includes(required), `Visual parity harness is missing fail-closed contract: ${required}`);
  }
  for (const bypass of ["resolveVisualRegions", "shellOnly", "AVIA_VISUAL_REGIONS"]) {
    assert.ok(!source.includes(bypass), `Visual parity harness can bypass the 255-pair matrix via ${bypass}.`);
  }
  assert.match(source, /for \(const comparison of comparisons\)[\s\S]*?comparison\.passed/, "Decoded region results are not asserted.");
}

export function assertParityBoundary(options = {}) {
  const repositoryRoot = path.resolve(options.repositoryRoot ?? defaultRepositoryRoot);
  const mutation = options.mutation ?? null;
  assertInteractionSourceBoundary(repositoryRoot, mutation);
  assertSourceBoundary(repositoryRoot, mutation);

  const visualSpecPath = path.join(repositoryRoot, "apps/web/tests/e2e/legacy-visual-parity.spec.ts");
  assert.ok(fs.existsSync(visualSpecPath), "Visual parity spec is missing.");
  const visualSource = mutateVisualSource(fs.readFileSync(visualSpecPath, "utf8"), mutation);
  assertVisualHarnessSource(visualSource);

  if (options.requireBuilds !== false) {
    assertBuildBoundary(repositoryRoot, mutation);
  } else if (mutation === "http-mock-import") {
    assertBuildInputBoundary("http", [path.join(repositoryRoot, "apps/web/src/mock/seed-data.ts")]);
  }

  return { routes: EXPECTED_ROUTE_COUNT, profiles: options.requireBuilds === false ? 0 : 2 };
}

const invokedPath = process.argv[1] ? path.resolve(process.argv[1]) : null;
if (invokedPath === scriptPath) {
  const result = assertParityBoundary({
    mutation: process.env.AVIA_BOUNDARY_MUTATION,
    requireBuilds: process.env.AVIA_BOUNDARY_SOURCE_ONLY !== "1",
  });
  console.log(`parity-boundary-scan: ok (${result.routes} routes, ${result.profiles} build profiles)`);
}
