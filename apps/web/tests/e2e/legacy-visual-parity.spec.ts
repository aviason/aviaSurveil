import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { expect, test } from "@playwright/test";

import {
  assertBaselineUpdateMode,
  assertVisualPairAttachments,
  assertViewportScreenshotContract,
  assertSurfaceSemantics,
  compareVisualFrames,
  decodePngFrame,
  driveReactSurface,
  installDeterministicPageState,
  resolveFocusedSurfaces,
  resolveReactSurfaceSemantics,
  validateBaselineManifest,
  visualComparisonRegions,
  VISUAL_BASELINE_ROOT,
  VISUAL_SURFACE_BY_ID,
  VISUAL_SURFACES,
  VISUAL_VIEWPORTS,
  VISUAL_VIEWPORT_BY_ID,
  type BaselineManifest,
} from "./support/legacy-parity-fixtures";

const appRoot = process.cwd();
const repoRoot = resolve(appRoot, "../..");
const baselineRoot = resolve(repoRoot, VISUAL_BASELINE_ROOT);
const manifestPath = resolve(baselineRoot, "baseline-manifest.json");
const surfaces = resolveFocusedSurfaces();

assertBaselineUpdateMode({
  command: "test:e2e:visual-parity",
  env: { AVIA_UPDATE_LEGACY_BASELINES: process.env.AVIA_UPDATE_LEGACY_BASELINES },
});

function readManifest(): BaselineManifest {
  const manifest = JSON.parse(readFileSync(manifestPath, "utf8")) as BaselineManifest;
  validateBaselineManifest(manifest, {
    baselineDir: baselineRoot,
    expectedSurfaces: VISUAL_SURFACE_BY_ID,
    expectedViewports: VISUAL_VIEWPORT_BY_ID,
  });
  return manifest;
}

const manifest = readManifest();
const expectedVisualPairCount = VISUAL_SURFACES.length * VISUAL_VIEWPORTS.length;

test.afterAll(() => {
  expect(VISUAL_SURFACES).toHaveLength(85);
  expect(expectedVisualPairCount).toBe(255);
});

const workbenchPrimitiveGallery = `
  <main class="workspace-content" data-testid="workbench-primitive-gallery">
    <header class="workbench-page-header" data-primitive="page-header">
      <div class="workbench-page-header__main">
        <p class="eyebrow">Geometry gallery</p>
        <h1>Oversight Workbench Primitives</h1>
        <p class="workspace-purpose">Stable primitive geometry for feature migrations.</p>
        <dl class="workbench-fact-grid" data-primitive="fact-grid">
          <div class="workbench-fact-grid__item"><dt>Owner</dt><dd>CAA Inspector</dd></div>
          <div class="workbench-fact-grid__item"><dt>Next action</dt><dd>Review evidence</dd></div>
          <div class="workbench-fact-grid__item"><dt>Status</dt><dd>In progress</dd></div>
          <div class="workbench-fact-grid__item"><dt>Due Date</dt><dd>24 Jul 2026</dd></div>
        </dl>
      </div>
      <button class="primary-button" type="button">Run checklist</button>
    </header>
    <section class="workbench-data-register" data-primitive="data-register">
      <table>
        <caption>CAP revisions</caption>
        <thead><tr><th scope="col">Revision</th><th scope="col">Owner</th><th scope="col">Decision</th></tr></thead>
        <tbody><tr><td>1</td><td>Fly Namibia</td><td>Needs CAA review</td></tr></tbody>
      </table>
      <div class="workbench-record-list">
        <article class="workbench-record-card" aria-label="CAP-REV-1">
          <h3>CAP-REV-1</h3>
          <dl><div><dt>Decision</dt><dd>Needs CAA review</dd></div></dl>
        </article>
      </div>
    </section>
    <ol aria-label="Finding lifecycle" class="workbench-lifecycle-stepper" data-primitive="lifecycle-stepper">
      <li data-state="complete"><span class="workbench-lifecycle-stepper__marker"></span><span>Finding issued</span></li>
      <li aria-current="step" data-state="current"><span class="workbench-lifecycle-stepper__marker"></span><span>CAP accepted</span></li>
      <li data-state="pending"><span class="workbench-lifecycle-stepper__marker"></span><span>Finding closed</span></li>
    </ol>
    <section class="workbench-decision-panel" data-primitive="decision-panel">
      <h2>CAA decision</h2>
      <div class="workbench-decision-panel__actions">
        <div class="workbench-decision-panel__action"><button class="primary-button" type="button">Accept evidence</button></div>
        <div class="workbench-decision-panel__action"><button class="primary-button" disabled type="button">Request more information</button><p>Auditee evidence is missing.</p></div>
      </div>
    </section>
    <section class="workbench-empty-error-state workbench-empty-error-state--error" data-primitive="empty-error-state">
      <p class="eyebrow">Error state</p>
      <h2>Audit Detail: Could not load audit</h2>
    </section>
  </main>
`;

test("checks shared workbench primitive geometry gallery", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await installDeterministicPageState(page);
  await page.goto("/");
  await page.evaluate((html) => {
    document.body.innerHTML = html;
  }, workbenchPrimitiveGallery);
  await expect(page.getByTestId("workbench-primitive-gallery")).toBeVisible();
  await expect(page.getByRole("table", { name: "CAP revisions" })).toBeVisible();
  await expect(page.getByRole("article", { name: "CAP-REV-1" })).toBeHidden();
  const desktopBoxes = await page.locator("[data-primitive]").evaluateAll((elements) =>
    elements.map((element) => {
      const rect = element.getBoundingClientRect();
      return { height: rect.height, width: rect.width };
    }),
  );
  expect(desktopBoxes.every((box) => box.width > 0 && box.height > 0)).toBe(true);
  const desktopTargets = await page.locator("button:visible").evaluateAll((elements) =>
    elements.map((element) => {
      const rect = element.getBoundingClientRect();
      return { height: rect.height, width: rect.width };
    }),
  );
  expect(desktopTargets.every((target) => target.width >= 44 && target.height >= 44)).toBe(true);

  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.getByRole("table", { name: "CAP revisions" })).toBeHidden();
  await expect(page.getByRole("article", { name: "CAP-REV-1" })).toBeVisible();
});

for (const viewport of VISUAL_VIEWPORTS) {
  for (const surface of surfaces) {
    test(`checks visual contract for ${surface.id} at ${viewport.id}`, async ({ page }, testInfo) => {
      const baselineItem = manifest.items.find(
        (item) => item.surfaceId === surface.id && item.viewport === viewport.id,
      );
      if (!baselineItem) throw new Error(`Missing baseline item for ${surface.id}/${viewport.id}.`);

      await page.setViewportSize(viewport);
      await installDeterministicPageState(page);
      await driveReactSurface(page, surface);
      const screenshot = await page.screenshot({ fullPage: false });
      assertViewportScreenshotContract({
        fullPage: false,
        viewport,
        imageSize: baselineItem.viewportSize,
      });

      const baseline = readFileSync(resolve(baselineRoot, baselineItem.file));
      await testInfo.attach("react-candidate-viewport", {
        body: screenshot,
        contentType: "image/png",
      });
      const baselineFrame = decodePngFrame(baseline);
      const candidateFrame = decodePngFrame(screenshot);
      const comparisons = visualComparisonRegions(viewport, surface.parityMode).map((contract) => {
        const result = compareVisualFrames(baselineFrame, candidateFrame, {
          region: contract.region,
          masks: surface.masks,
          maxDiffPixelRatio: contract.maxDiffPixelRatio,
          maxChannelDelta: contract.maxChannelDelta,
        });
        return {
          region: contract.region,
          maxDiffPixelRatio: contract.maxDiffPixelRatio,
          maxChannelDelta: contract.maxChannelDelta,
          ...result,
        };
      });
      await testInfo.attach("decoded-pixel-region-results", {
        body: JSON.stringify(
          {
            surfaceId: surface.id,
            viewport: viewport.id,
            parityMode: surface.parityMode,
            baseline: baselineItem.file,
            comparisons,
          },
          null,
          2,
        ),
        contentType: "application/json",
      });
      assertVisualPairAttachments(testInfo.attachments);
      for (const comparison of comparisons) {
        expect.soft(
          comparison.passed,
          `${surface.id}/${viewport.id}/${comparison.region.id} ratio ${comparison.diffPixelRatio.toFixed(5)} max ${comparison.maxDiffPixelRatio.toFixed(2)}`,
        ).toBe(true);
      }
      if (surface.parityMode === "content-adapted") {
        await assertSurfaceSemantics(page, resolveReactSurfaceSemantics(surface));
        await expect(page.locator("[data-testid='application-shell']")).toBeVisible();
        await expect(page.locator(".workspace-sidebar")).toBeVisible();
        await expect(page.locator(".application-topbar")).toBeVisible();
        await expect(page.locator(".workspace-content")).toBeVisible();
        await expect(page.locator(".workbench-page-header")).toBeVisible();
        if (surface.id === "inspector-findings") {
          const dossierLink = page.getByRole("link", { name: "Open Finding dossier" });
          await expect(dossierLink).toBeVisible();
          await dossierLink.click();
          await expect(page).toHaveURL(/\/inspector\/findings\/FND-CAB-2026-001$/);
          await page.getByRole("link", { name: "Open Inspector Assistant" }).click();
          await expect(page).toHaveURL(/\/inspector\/assistant$/);
          if (viewport.id === "mobile") {
            await page.getByRole("button", { name: "Open navigation" }).click();
            const drawer = page.getByRole("dialog", { name: "Primary navigation" });
            await expect(drawer).toBeVisible();
            await drawer.getByRole("link", { name: "Findings" }).click();
            await expect(drawer).toBeHidden();
            await expect(page).toHaveURL(/\/inspector\/findings$/);
          }
        }
        const touchTargets = await page.locator("a:visible, button:visible").evaluateAll((elements) =>
          elements.map((element) => {
            const rect = element.getBoundingClientRect();
            return { width: rect.width, height: rect.height, text: element.textContent?.trim() ?? "" };
          }),
        );
        expect(
          touchTargets.every((target) => target.width >= 44 && target.height >= 44),
          `undersized touch targets: ${JSON.stringify(touchTargets.filter((target) => target.width < 44 || target.height < 44))}`,
        ).toBe(true);
        if (surface.id === "inspector-home") {
          if (viewport.width <= 1100) {
            await expect(page.getByRole("article", { name: "AUD-2026-001" })).toBeVisible();
          } else {
            await expect(page.getByRole("table", { name: "Assigned Audits" })).toBeVisible();
          }
        }
        if (surface.id === "audit-detail") {
          await expect(page.getByTestId("audit-dossier")).toBeVisible();
          await expect(page.getByTestId("offline-readiness-panel")).toBeVisible();
        }
        if (surface.id === "checklist-runner") {
          await expect(page.getByTestId("checklist-question-row")).toHaveCount(6);
          await expect(page.getByTestId("checklist-response-panel")).toBeVisible();
        }
        if (surface.id === "auditee-home") {
          await expect(page.getByTestId("auditee-scope")).toContainText("Fly Namibia");
          await expect(page.getByRole("table", { name: "My Findings" })).toBeVisible();
          await expect(page.getByTestId("auditee-selected-finding")).toBeVisible();
          await expect(page.locator("body")).not.toContainText(
            /SkyCargo Air|Internal CAA Note|Inspector workload|internal risk|enforcement deliberation/i,
          );
        }
      }
    });
  }
}
