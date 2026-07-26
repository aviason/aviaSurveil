import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { describe, expect, it } from "vitest";

import { resolveBuildProfile } from "./build-profile";

describe("build-time backend profile", () => {
  it("accepts only explicit demo and HTTP values", () => {
    expect(resolveBuildProfile("demo")).toBe("demo");
    expect(resolveBuildProfile("http")).toBe("http");
    expect(() => resolveBuildProfile("production")).toThrow(/exactly 'demo' or 'http'/);
  });

  it("keeps backend-family imports separated at the two entries", () => {
    const root = path.resolve(import.meta.dirname, "..");
    const demoEntry = fs.readFileSync(path.join(root, "entry/demo.tsx"), "utf8");
    const httpEntry = fs.readFileSync(path.join(root, "entry/http.tsx"), "utf8");
    expect(demoEntry).toMatch(/createMockBackend/);
    expect(demoEntry).not.toMatch(/createHttpBackend/);
    expect(httpEntry).toMatch(/createHttpBackend/);
    expect(httpEntry).not.toMatch(/src\/mock|createMockBackend|seed-data/);
  });

  it("fails closed under route, action, import, artifact, and visual harness mutations", () => {
    const repositoryRoot = path.resolve(import.meta.dirname, "../../../..");
    const script = path.join(repositoryRoot, "apps/web/scripts/assert-parity-boundary.mjs");
    const mutations = [
      ["undeclared-route", /undeclared React path/],
      ["inert-button", /inert button/],
      ["remove-action-harness-contract", /visible action harness.*durable outcome/i],
      ["skip-stateful-control-execution", /stateful and form controls/i],
      ["skip-mobile-command-execution", /mobile-only route controls/i],
      ["toast-only-action", /toast-only action/],
      ["unlabelled-control", /accessible name/],
      ["remove-focus-indicator-contract", /visible keyboard focus indicator/i],
      ["fake-dropdown", /native select semantics/],
      ["duplicate-accessible-navigation", /one accessible primary navigation/],
      ["missing-disabled-reason", /record-specific disabled reason/],
      ["broken-deep-link", /deep link/],
      ["missing-mobile-viewport", /mobile viewport/],
      ["broad-root-import", /protected root runtime code/],
      ["http-mock-import", /forbidden mock\/test input/],
      ["remove-shell-assertion", /missing fail-closed contract.*workspace-sidebar/],
      ["remove-content-assertion", /missing fail-closed contract.*workbench-page-header/],
      ["compressed-byte-comparator", /decoded pixels, not compressed PNG bytes/],
      ["skip-viewport", /missing fail-closed contract.*VISUAL_VIEWPORTS/],
    ] as const;

    for (const [mutation, expected] of mutations) {
      const result = spawnSync(process.execPath, [script], {
        cwd: repositoryRoot,
        encoding: "utf8",
        env: {
          ...process.env,
          AVIA_BOUNDARY_MUTATION: mutation,
          AVIA_BOUNDARY_SOURCE_ONLY: "1",
        },
      });
      expect(result.status, `${mutation} unexpectedly passed`).not.toBe(0);
      expect(`${result.stdout}\n${result.stderr}`).toMatch(expected);
    }
  });
});
