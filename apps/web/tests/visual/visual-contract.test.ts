import { spawnSync } from "node:child_process";
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { describe, expect, it } from "vitest";
import { REACT_ROUTE_CONTRACT_BY_ID } from "../../src/app/route-contracts";

import {
  assertBaselineUpdateMode,
  assertVisualSpecFailClosed,
  assertViewportScreenshotContract,
  compareVisualFrames,
  decodePngFrame,
  hashBytes,
  patchedFrame,
  resolveFocusedSurfaces,
  resolveVisualRegions,
  solidFrame,
  validateBaselineManifest,
  validateGeometrySnapshot,
  validateMaskContract,
  visualComparisonRegions,
  VISUAL_MAX_CHANNEL_DELTA,
  VISUAL_SURFACES,
  VISUAL_VIEWPORTS,
  type BaselineManifest,
  type RectMask,
} from "../e2e/support/legacy-parity-fixtures";

function withTempDir<T>(callback: (dir: string) => T): T {
  const dir = mkdtempSync(join(tmpdir(), "aviasurveil360-visual-contract-"));
  try {
    return callback(dir);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
}

function singleItemManifest(fileSha256: string): BaselineManifest {
  return {
    schemaVersion: 1,
    generatedAt: "2026-06-15T09:00:00.000Z",
    baselineVersion: "react-legacy-parity-v1",
    surfaceCount: 1,
    viewportCount: 1,
    source: {
      commit: "candidate-source",
      files: {
        "index.html": "sha256:source-index",
        "css/styles.css": "sha256:source-css",
        "js/app.js": "sha256:source-app",
        "js/views.js": "sha256:source-views",
        "js/data.js": "sha256:source-data",
      },
      auditDocument: {
        path: "docs/demo-evidence/UI_SCREEN_AUDIT_2026-07-19.md",
        sha256: "sha256:audit-doc",
      },
      packageLockSha256: "sha256:package-lock",
    },
    environment: {
      playwrightVersion: "1.61.1",
      chromiumVersion: "chromium-test",
      nodeVersion: "26.0.0",
      platform: "darwin",
      arch: "arm64",
      osRelease: "contract",
    },
    items: [
      {
        auditId: "ui-audit-001",
        surfaceId: "role-select",
        parityMode: "strict-shell",
        viewport: "desktop",
        viewportSize: { width: 100, height: 100 },
        file: "desktop/role-select.png",
        sha256: fileSha256,
      sourceRoute: {
          legacyView: "login",
          legacyParams: {},
          reactPath: "/",
        },
      },
    ],
  };
}

describe("visual parity contract", () => {
  it("fails a visual pair when either required attachment is missing or duplicated", async () => {
    const fixtures = await import("../e2e/support/legacy-parity-fixtures");
    const assertVisualPairAttachments = Reflect.get(fixtures, "assertVisualPairAttachments");

    expect(assertVisualPairAttachments).toBeTypeOf("function");
    expect(() =>
      assertVisualPairAttachments([
        { name: "react-candidate-viewport" },
        { name: "decoded-pixel-region-results" },
      ]),
    ).not.toThrow();
    expect(() =>
      assertVisualPairAttachments([{ name: "decoded-pixel-region-results" }]),
    ).toThrow(/exactly one react candidate/i);
    expect(() =>
      assertVisualPairAttachments([
        { name: "react-candidate-viewport" },
        { name: "decoded-pixel-region-results" },
        { name: "decoded-pixel-region-results" },
      ]),
    ).toThrow(/exactly one decoded-pixel result/i);
  });

  it("keeps legacy capture semantics immutable while resolving React candidate semantics", async () => {
    const fixtures = await import("../e2e/support/legacy-parity-fixtures");
    const resolveReactSurfaceSemantics = Reflect.get(fixtures, "resolveReactSurfaceSemantics");
    const surfaceById = (id: string) => {
      const surface = VISUAL_SURFACES.find((item) => item.id === id);
      if (!surface) throw new Error(`Missing visual fixture for ${id}.`);
      return surface;
    };
    const inspectorFindings = surfaceById("inspector-findings");
    const leadHome = surfaceById("lead-home");
    const adminConfigurations = surfaceById("admin-configurations");
    const auditPlan = surfaceById("audit-plan");
    const auditeeReport = surfaceById("auditee-report-preview");
    const inspectorHome = surfaceById("inspector-home");

    expect(resolveReactSurfaceSemantics).toBeTypeOf("function");
    expect(inspectorFindings).toMatchObject({ expectedSemanticMarker: "CAB-2026-011" });
    expect(resolveReactSurfaceSemantics(inspectorFindings)).toMatchObject({
      expectedHeading: "Findings",
      expectedSemanticMarker: "CAR-2026-099",
    });
    expect(resolveReactSurfaceSemantics(leadHome)).toMatchObject({
      expectedSemanticMarker: "Lead decision required",
    });
    expect(resolveReactSurfaceSemantics(adminConfigurations)).toMatchObject({
      expectedHeading: "Configurations",
      expectedSemanticMarker: "Configured demo rules",
    });
    expect(resolveReactSurfaceSemantics(auditPlan)).toMatchObject({
      expectedSemanticMarker: "Plan #AB001",
    });
    expect(resolveReactSurfaceSemantics(auditeeReport)).toMatchObject({
      expectedSemanticMarker: "Released report unavailable",
    });
    expect(resolveReactSurfaceSemantics(inspectorHome)).toMatchObject({
      expectedSemanticMarker: "AUD-2026-001",
      expectedOwnerText: "CAA Inspector",
      expectedNextActionText: "Continue Cabin Inspection checklist",
    });
  });

  it("freezes the 85 active surfaces by three viewports while retaining the root oracle separately", () => {
    expect(VISUAL_SURFACES).toHaveLength(85);
    expect(VISUAL_VIEWPORTS).toHaveLength(3);
    expect(VISUAL_SURFACES.length * VISUAL_VIEWPORTS.length).toBe(255);
    expect(new Set(VISUAL_SURFACES.map((surface) => surface.id)).size).toBe(85);
    expect(new Set(VISUAL_SURFACES.map((surface) => surface.auditId)).size).toBe(85);
    expect(VISUAL_SURFACES.every((surface) => surface.expectedHeading.trim().length > 0)).toBe(true);
    expect(VISUAL_SURFACES.every((surface) => surface.expectedSemanticMarker?.trim().length)).toBe(true);
    expect(VISUAL_SURFACES.filter((surface) => surface.id !== "role-select").every((surface) => surface.legacy.role)).toBe(true);
    for (const surface of VISUAL_SURFACES) {
      expect(surface.legacy.role).toBe(REACT_ROUTE_CONTRACT_BY_ID.get(surface.id)?.requiredRole ?? null);
    }
    expect(VISUAL_SURFACES.find((surface) => surface.auditId === "ui-audit-009")).toMatchObject({
      id: "finding-detail",
      reactPath: "/inspector/findings/FND-CAB-2026-001",
      legacy: { role: "inspector" },
    });
    expect(VISUAL_SURFACES.find((surface) => surface.auditId === "ui-audit-044")).toMatchObject({
      id: "evidence-review",
      reactPath: "/department-manager/evidence/FND-CAB-2026-001",
      legacy: { role: "manager" },
    });
    expect(VISUAL_SURFACES.find((surface) => surface.id === "admin-inspection-package-builder")).toMatchObject({
      legacy: { role: "admin", view: "package-builder" },
      expectedHeading: "Approved AGA Catalog",
    });
    expect(VISUAL_SURFACES.find((surface) => surface.id === "admin-organization-detail")).toMatchObject({
      legacy: { role: "admin", view: "org-detail", params: { orgId: "ORG-XYZ" } },
    });
    expect(VISUAL_SURFACES.find((surface) => surface.id === "inspector-calendar")).toMatchObject({
      expectedSemanticMarker: "Fly Namibia · Cabin Inspection",
    });
    expect(VISUAL_SURFACES.find((surface) => surface.id === "manager-preliminary-report-review")).toMatchObject({
      legacy: { role: "manager", view: "audit-reports", params: { filter: "preliminary" } },
      expectedHeading: "Reports Approval",
      expectedSemanticMarker: "PR-2026-018-V1",
    });
    for (const step of [1, 2, 3, 4, 5]) {
      expect(VISUAL_SURFACES.find((surface) => surface.id === `new-audit-wizard-${step}`)).toMatchObject({
        legacyState: { wizardStep: step },
      });
    }
    expect(VISUAL_SURFACES.find((surface) => surface.id === "lead-preliminary-report-workflow")).toMatchObject({
      legacyState: { preliminaryWorkflow: true },
      expectedSemanticMarker: "Report version was not found.",
    });
    expect(VISUAL_SURFACES.find((surface) => surface.id === "lead-final-report-readiness")).toMatchObject({
      legacy: { role: "leadInspector", view: "audit-reports", params: { filter: "final", finalReportId: "FR-2026-018" } },
      expectedHeading: "Final Report Readiness",
      expectedSemanticMarker: "Report version was not found.",
    });
    expect(VISUAL_SURFACES.find((surface) => surface.id === "lead-prepare-final-report")).toMatchObject({
      legacy: { role: "leadInspector", view: "final-report-prepare", params: { reportId: "FR-2026-018" } },
      expectedHeading: "Final Report Preparation",
      expectedSemanticMarker: "Report version was not found.",
    });
    expect(VISUAL_SURFACES.find((surface) => surface.id === "lead-final-report-document")).toMatchObject({
      legacy: { role: "leadInspector", view: "final-report-view", params: { reportId: "FR-2026-018" } },
    });
    expect(VISUAL_SURFACES.find((surface) => surface.id === "executive-report-preview")).toMatchObject({
      legacy: { role: "executiveDirector", view: "executive-report-preview", params: { reportId: "FR-2026-022" } },
      expectedHeading: "Final Report Preview",
      expectedSemanticMarker: "Report version was not found.",
    });
    expect(VISUAL_SURFACES.find((surface) => surface.id === "auditee-report-preview")).toMatchObject({
      legacy: { role: "auditee", view: "service-provider-report-preview", params: { reportId: "FR-2025-009" } },
      expectedHeading: "Report Preview",
      expectedSemanticMarker: "Released report unavailable",
    });
  });

  it("fails every named visual harness mutation before the browser comparison can run", () => {
    const repositoryRoot = join(process.cwd(), "../..");
    const script = join(repositoryRoot, "apps/web/scripts/assert-parity-boundary.mjs");
    const mutations = [
      ["missing-route", /85-surface/],
      ["remove-http-profile", /All 85 routes must be dual-profile/],
      ["restore-blocked-profile-reason", /must not retain a blocked profile reason/],
      ["skip-viewport", /VISUAL_VIEWPORTS/],
      ["remove-shell-assertion", /workspace-sidebar/],
      ["remove-content-assertion", /workbench-page-header/],
      ["compressed-byte-comparator", /decoded pixels, not compressed PNG bytes/],
      ["remove-candidate-attachment", /react-candidate-viewport/],
      ["remove-result-attachment", /decoded-pixel-region-results/],
    ] as const;

    for (const [mutation, reason] of mutations) {
      const result = spawnSync(process.execPath, [script], {
        cwd: repositoryRoot,
        encoding: "utf8",
        env: { ...process.env, AVIA_BOUNDARY_MUTATION: mutation, AVIA_BOUNDARY_SOURCE_ONLY: "1" },
      });
      expect(result.status, `${mutation} unexpectedly passed`).not.toBe(0);
      expect(`${result.stdout}\n${result.stderr}`).toMatch(reason);
    }
  });
  it("decodes tracked PNGs to RGBA pixels and defines fail-closed shell/content regions", () => {
    const baseline = readFileSync(
      join(process.cwd(), "tests/visual-baselines/react-legacy-parity/desktop/inspector-home.png"),
    );
    const frame = decodePngFrame(baseline);
    const regions = visualComparisonRegions(
      { id: "desktop", width: 1440, height: 900 },
      "content-adapted",
    );

    expect(frame).toMatchObject({ width: 1440, height: 900 });
    expect(frame.data).toHaveLength(1440 * 900 * 4);
    expect(regions.map((item) => [item.region.id, item.maxDiffPixelRatio])).toEqual([
      ["sidebar", 0.03],
      ["topbar", 0.03],
      ["content-header", 0.08],
      ["content", 0.08],
    ]);
    expect(regions.find((item) => item.region.id === "topbar")?.region).toEqual({
      id: "topbar",
      x: 1180,
      y: 0,
      width: 260,
      height: 64,
    });
    expect(regions.find((item) => item.region.id === "content-header")?.region).toEqual({
      id: "content-header",
      x: 230,
      y: 0,
      width: 1210,
      height: 64,
    });
  });

  it("rejects a production visual spec that compares compressed bytes or bypasses adapted regions", () => {
    const specPath = join(process.cwd(), "tests/e2e/legacy-visual-parity.spec.ts");
    expect(() => assertVisualSpecFailClosed(readFileSync(specPath, "utf8"))).not.toThrow();

    expect(() =>
      assertVisualSpecFailClosed(`
        const ratio = byteDiffRatio(baseline, screenshot);
        if (surface.parityMode === "content-adapted") return;
      `),
    ).toThrow(/decoded-pixel|bypass/i);
  });

  it("supports scoped visual execution while preserving the active 85 by 3 invariant", () => {
    const spec = readFileSync(join(process.cwd(), "tests/e2e/legacy-visual-parity.spec.ts"), "utf8");
    expect(spec).toContain("const surfaces = resolveFocusedSurfaces()");
    expect(spec).toContain("const expectedVisualPairCount = VISUAL_SURFACES.length * VISUAL_VIEWPORTS.length");
    expect(spec).toContain("expect(expectedVisualPairCount).toBe(255)");
    expect(spec).toContain("assertVisualPairAttachments(testInfo.attachments)");
    expect(spec).not.toContain("reactCandidateAttachmentCount");
    expect(spec).not.toContain("decodedRegionResultAttachmentCount");
  });

  it("fails an unmasked deterministic patch outside the strict region ratio", () => {
    const baseline = solidFrame(100, 100, [11, 12, 13, 255]);
    const candidate = patchedFrame(baseline, { x: 0, y: 0, width: 20, height: 20 }, [240, 240, 240, 255]);

    const result = compareVisualFrames(baseline, candidate, {
      region: { id: "viewport", x: 0, y: 0, width: 100, height: 100 },
      masks: [],
      maxDiffPixelRatio: 0.03,
      maxChannelDelta: VISUAL_MAX_CHANNEL_DELTA,
    });

    expect(result.passed).toBe(false);
    expect(result.diffPixelRatio).toBeCloseTo(0.04, 3);
  });

  it("tolerates a perturbation only inside one allowlisted dynamic leaf", () => {
    const baseline = solidFrame(100, 100, [11, 12, 13, 255]);
    const mask: RectMask = {
      selector: "[data-visual-dynamic='clock']",
      rationale: "Clock text is deterministic in normal capture and isolated for contract testing.",
      expectedCount: 1,
      rects: [{ x: 10, y: 10, width: 10, height: 10 }],
    };

    const insideCandidate = patchedFrame(
      baseline,
      { x: 10, y: 10, width: 10, height: 10 },
      [240, 240, 240, 255],
    );
    const outsideCandidate = patchedFrame(
      baseline,
      { x: 30, y: 10, width: 20, height: 20 },
      [240, 240, 240, 255],
    );

    expect(
      compareVisualFrames(baseline, insideCandidate, {
        region: { id: "viewport", x: 0, y: 0, width: 100, height: 100 },
        masks: [mask],
        maxDiffPixelRatio: 0.03,
        maxChannelDelta: VISUAL_MAX_CHANNEL_DELTA,
      }).passed,
    ).toBe(true);
    expect(
      compareVisualFrames(baseline, outsideCandidate, {
        region: { id: "viewport", x: 0, y: 0, width: 100, height: 100 },
        masks: [mask],
        maxDiffPixelRatio: 0.03,
        maxChannelDelta: VISUAL_MAX_CHANNEL_DELTA,
      }).passed,
    ).toBe(false);
  });

  it("ignores bounded Chromium gradient dithering but still counts a visible channel change", () => {
    const baseline = solidFrame(10, 10, [8, 29, 54, 255]);
    const dithered = solidFrame(10, 10, [9, 30, 54, 255]);
    const changed = solidFrame(10, 10, [49, 29, 54, 255]);
    const options = {
      region: { id: "viewport", x: 0, y: 0, width: 10, height: 10 },
      masks: [],
      maxDiffPixelRatio: 0.03,
      maxChannelDelta: VISUAL_MAX_CHANNEL_DELTA,
    } as const;

    expect(compareVisualFrames(baseline, dithered, options).diffPixelRatio).toBe(0);
    expect(compareVisualFrames(baseline, changed, options).diffPixelRatio).toBe(1);
  });

  it("fails a four-pixel shell geometry shift", () => {
    expect(() =>
      validateGeometrySnapshot(
        {
          shell: { x: 0, y: 0, width: 100, height: 100 },
          sidebar: { x: 0, y: 0, width: 24, height: 100 },
          topbar: { x: 24, y: 0, width: 76, height: 12 },
          content: { x: 24, y: 12, width: 76, height: 88 },
        },
        {
          shell: { x: 4, y: 0, width: 100, height: 100 },
          sidebar: { x: 0, y: 0, width: 24, height: 100 },
          topbar: { x: 24, y: 0, width: 76, height: 12 },
          content: { x: 24, y: 12, width: 76, height: 88 },
        },
      ),
    ).toThrow(/shell geometry/i);
  });

  it("rejects broad masks and masks covering more than five percent of the viewport", () => {
    const broadMask: RectMask = {
      selector: "body",
      rationale: "Too broad.",
      expectedCount: 1,
      rects: [{ x: 0, y: 0, width: 10, height: 10 }],
    };
    const largeMask: RectMask = {
      selector: "[data-visual-dynamic='large']",
      rationale: "This intentionally covers too much viewport area.",
      expectedCount: 1,
      rects: [{ x: 0, y: 0, width: 30, height: 30 }],
    };

    expect(() => validateMaskContract([broadMask], { width: 100, height: 100 })).toThrow(/broad/i);
    expect(() => validateMaskContract([largeMask], { width: 100, height: 100 })).toThrow(/5%/);
  });

  it("fails missing baselines, altered PNG hashes, stale source hashes, stale lockfile hashes, wrong metadata, and full-page input", () => {
    withTempDir((dir) => {
      const pngPath = join(dir, "desktop", "role-select.png");
      const manifest = singleItemManifest("sha256:missing");

      expect(() =>
        validateBaselineManifest(manifest, {
          baselineDir: dir,
          expectedSource: manifest.source,
          expectedEnvironment: manifest.environment,
        }),
      ).toThrow(/missing baseline/i);

      mkdirSync(join(dir, "desktop"), { recursive: true });
      writeFileSync(pngPath, Buffer.from("png-v1"));
      expect(() =>
        validateBaselineManifest(manifest, {
          baselineDir: dir,
          expectedSource: manifest.source,
          expectedEnvironment: manifest.environment,
        }),
      ).toThrow(/hash drift/i);

      const realHash = hashBytes(Buffer.from("png-v1"));
      const validManifest = singleItemManifest(realHash);
      expect(() =>
        validateBaselineManifest(validManifest, {
          baselineDir: dir,
          expectedSource: manifest.source,
          expectedEnvironment: manifest.environment,
        }),
      ).not.toThrow();

      writeFileSync(join(dir, "desktop", "untracked.png"), Buffer.from("untracked"));
      expect(() =>
        validateBaselineManifest(validManifest, {
          baselineDir: dir,
          expectedSource: manifest.source,
          expectedEnvironment: manifest.environment,
        }),
      ).toThrow(/unexpected extra baseline/i);
      rmSync(join(dir, "desktop", "untracked.png"));

      expect(() =>
        validateBaselineManifest(singleItemManifest(realHash), {
          baselineDir: dir,
          expectedSource: {
            ...manifest.source,
            files: { ...manifest.source.files, "index.html": "sha256:stale" },
          },
          expectedEnvironment: manifest.environment,
        }),
      ).toThrow(/source metadata/i);

      expect(() =>
        validateBaselineManifest(singleItemManifest(realHash), {
          baselineDir: dir,
          expectedSource: { ...manifest.source, packageLockSha256: "sha256:stale-lock" },
          expectedEnvironment: manifest.environment,
        }),
      ).toThrow(/package-lock/i);

      expect(() =>
        validateBaselineManifest(singleItemManifest(realHash), {
          baselineDir: dir,
          expectedSource: manifest.source,
          expectedEnvironment: { ...manifest.environment, platform: "linux" },
        }),
      ).toThrow(/platform/i);

      expect(() =>
        assertViewportScreenshotContract({
          fullPage: true,
          viewport: { width: 100, height: 100 },
          imageSize: { width: 100, height: 240 },
        }),
      ).toThrow(/fullPage:false/i);
    });
  });

  it("allows baseline writes only through the guarded update command", () => {
    expect(() =>
      assertBaselineUpdateMode({
        command: "test:e2e:visual-parity",
        env: { AVIA_UPDATE_LEGACY_BASELINES: "1" },
      }),
    ).toThrow(/visual:baseline:update/i);

    expect(() =>
      assertBaselineUpdateMode({
        command: "visual:baseline:update",
        env: { AVIA_UPDATE_LEGACY_BASELINES: "1" },
      }),
    ).not.toThrow();
  });

  it("rejects unknown, duplicate, empty surface filters and unsupported region filters", () => {
    expect(() => resolveFocusedSurfaces("role-select,role-select")).toThrow(/duplicate/i);
    expect(() => resolveFocusedSurfaces("")).toThrow(/empty/i);
    expect(() => resolveFocusedSurfaces("not-a-surface")).toThrow(/unknown/i);

    expect(resolveFocusedSurfaces("role-select").map((surface) => surface.id)).toEqual(["role-select"]);
    expect(resolveVisualRegions("shell", { allowShellOnly: true })).toEqual(["shell"]);
    expect(() => resolveVisualRegions("shell")).toThrow(/Task 6/i);
    expect(() => resolveVisualRegions("content")).toThrow(/unsupported/i);
  });
});
