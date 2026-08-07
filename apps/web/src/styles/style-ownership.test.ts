import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

import { BRAND_ASSET_SOURCES } from "../ui/brand-assets";

const styleRoot = fileURLToPath(new URL(".", import.meta.url));
const styleFiles = [
  "app.css",
  "reset.css",
  "tokens.css",
  "base.css",
  "shell.css",
  "primitives.css",
  "features/inspector.css",
  "features/inspector-secondary.css",
  "features/lead-review.css",
  "features/lead-secondary.css",
  "features/manager-operations.css",
  "features/manager-intelligence.css",
  "features/planning-intake.css",
  "features/auditee.css",
  "features/auditee-secondary.css",
  "features/management.css",
  "features/executive-review.css",
  "features/executive-secondary.css",
  "features/admin.css",
  "features/admin-secondary.css",
  "utilities.css",
  "responsive.css",
] as const;

const expectedAppCss = `@layer reset, tokens, base, shell, primitives, features, utilities, responsive;
@import "./reset.css" layer(reset);
@import "./tokens.css" layer(tokens);
@import "./base.css" layer(base);
@import "./shell.css" layer(shell);
@import "./primitives.css" layer(primitives);
@import "./features/inspector.css" layer(features);
@import "./features/inspector-secondary.css" layer(features);
@import "./features/lead-review.css" layer(features);
@import "./features/lead-secondary.css" layer(features);
@import "./features/manager-operations.css" layer(features);
@import "./features/manager-intelligence.css" layer(features);
@import "./features/planning-intake.css" layer(features);
@import "./features/auditee.css" layer(features);
@import "./features/auditee-secondary.css" layer(features);
@import "./features/management.css" layer(features);
@import "./features/executive-review.css" layer(features);
@import "./features/executive-secondary.css" layer(features);
@import "./features/admin.css" layer(features);
@import "./features/admin-secondary.css" layer(features);
@import "../features/admin/aga-candidate-demo.css" layer(features);
@import "../features/checklists/aga-classification-workspace-page.css" layer(features);
@import "../features/checklists/question-review-page.css" layer(features);
@import "../features/inspections/aga-demo-lifecycle.css" layer(features);
@import "../features/inspections/aga-demo-inspection-package.css" layer(features);
@import "./utilities.css" layer(utilities);
@import "./responsive.css" layer(responsive);
`;

function readStyle(fileName: (typeof styleFiles)[number]): string {
  return readFileSync(resolve(styleRoot, fileName), "utf8");
}

function collectSelectors(source: string): string[] {
  const withoutComments = source.replace(/\/\*[\s\S]*?\*\//g, "");
  const selectors: string[] = [];
  const rulePattern = /([^{}@][^{}]*)\{/g;
  let match: RegExpExecArray | null;
  while ((match = rulePattern.exec(withoutComments))) {
    selectors.push(
      match[1]
        .split(",")
        .map((selector) => selector.trim().replace(/\s+/g, " "))
        .filter((selector) => Boolean(selector) && !selector.startsWith("@"))
        .sort()
        .join(", "),
    );
  }
  return selectors.filter(Boolean);
}

describe("CSS layer and ownership contract", () => {
  it("keeps app.css as the binding layer declaration plus ordered imports only", () => {
    expect(readStyle("app.css")).toBe(expectedAppCss);
  });

  it("keeps Task 5 Lead back links touch-safe and analytics below the desktop topbar control area", () => {
    const leadSecondary = readStyle("features/lead-secondary.css");
    expect(leadSecondary).toMatch(/\.lead-back-link\s*\{[^}]*min-width:\s*44px;[^}]*min-height:\s*44px;/s);
    expect(leadSecondary).toMatch(/\.lead-analytics-page \.lead-secondary-header\s*\{[^}]*max-width:\s*calc\(100% - 220px\);/s);
  });

  it("keeps global document selectors owned only by base.css", () => {
    const forbiddenGlobalSelector = /(^|,\s*)(?:html|body|#root)(?:\s|,|\{|$)/;
    for (const fileName of styleFiles.filter((fileName) => fileName !== "base.css")) {
      expect(readStyle(fileName)).not.toMatch(forbiddenGlobalSelector);
    }
    expect(readStyle("base.css")).toMatch(/body\s*\{/);
    expect(readStyle("base.css")).toMatch(/#root\s*\{/);
  });

  it("uses the accepted system workbench font and limits DM Sans to login role selection", () => {
    expect(readStyle("base.css")).toMatch(/font-family:\s*var\(--avia-workbench-font-family\)/);
    const dmSansOccurrences = [
      ...new Set(
        styleFiles.flatMap((fileName) =>
          [...readStyle(fileName).matchAll(/DM Sans/g)].map(() => fileName),
        ),
      ),
    ];
    expect(dmSansOccurrences).toEqual(["tokens.css"]);
    expect(readStyle("shell.css")).toMatch(
      /\.role-select-page\s*\{[\s\S]*font-family:\s*var\(--avia-login-font-family\)/,
    );
  });

  it("keeps CSS asset URLs synchronized with the typed brand registry", () => {
    const approvedSources = new Set<string>([
      BRAND_ASSET_SOURCES.mark,
      BRAND_ASSET_SOURCES.loginTexture,
      BRAND_ASSET_SOURCES.dmSansVariable,
      ...Object.values(BRAND_ASSET_SOURCES.icons),
    ]);
    const cssSources = styleFiles.flatMap((fileName) =>
      [...readStyle(fileName).matchAll(/url\(["']?([^"')]+)["']?\)/g)].map((match) => match[1]),
    );
    expect(cssSources.length).toBeGreaterThanOrEqual(3);
    for (const sourcePath of cssSources) {
      expect(approvedSources.has(sourcePath)).toBe(true);
    }
  });

  it("rejects root demo runtime imports, unlayered app imports, important overrides, and duplicate selectors", () => {
    const seenSelectors = new Map<string, string>();
    for (const fileName of styleFiles) {
      const source = readStyle(fileName);
      expect(source).not.toMatch(/css\/styles\.css|@import\s+["'][^"']+["'](?!\s+layer\()/);
      expect(source).not.toMatch(/!important/);
      if (fileName === "responsive.css") continue;
      for (const selector of collectSelectors(source)) {
        const owner = seenSelectors.get(selector);
        expect(owner, `${selector} duplicated in ${owner} and ${fileName}`).toBeUndefined();
        seenSelectors.set(selector, fileName);
      }
    }
  });
});
