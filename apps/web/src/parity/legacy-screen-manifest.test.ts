import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { describe, expect, it } from "vitest";

import { REACT_ROUTE_CONTRACT_BY_ID } from "../app/route-contracts";
import {
  LEGACY_SCREEN_MANIFEST,
  LEGACY_SCREEN_SOURCE,
  PRODUCT_SCREEN_CROSSWALK,
  routeRoleForSurface,
} from "./legacy-screen-manifest";

interface AuditTuple {
  auditId: string;
  role: string;
  screenName: string;
}

const repoRoot = resolve(import.meta.dirname, "../../../..");
const auditMarkdown = readFileSync(
  resolve(repoRoot, "docs/demo-evidence/UI_SCREEN_AUDIT_2026-07-19.md"),
  "utf8",
);

function normalizeCell(value: string): string {
  return value.replace(/\s+/g, " ").trim();
}

function extractAuditRows(): AuditTuple[] {
  const start = auditMarkdown.indexOf("## Preserved pre-remediation screen findings");
  const end = auditMarkdown.indexOf("## Prioritized issue list");
  expect(start).toBeGreaterThan(-1);
  expect(end).toBeGreaterThan(start);
  return auditMarkdown
    .slice(start, end)
    .split(/\r?\n/)
    .flatMap((line): AuditTuple[] => {
      if (!line.startsWith("|")) return [];
      const cells = line
        .split("|")
        .slice(1, -1)
        .map(normalizeCell);
      if (cells.length !== 6 || !/^\d+$/.test(cells[0])) return [];
      return [
        {
          auditId: `ui-audit-${cells[0].padStart(3, "0")}`,
          role: cells[1],
          screenName: cells[2],
        },
      ];
    });
}

describe("legacy screen manifest", () => {
  it("matches the exact ordered 86 rows from the protected audit document", () => {
    const expected = extractAuditRows();
    expect(expected).toHaveLength(86);
    expect(LEGACY_SCREEN_SOURCE).toEqual(expected);
    expect(LEGACY_SCREEN_MANIFEST.map(({ auditId, role, screenName }) => ({
      auditId,
      role,
      screenName,
    }))).toEqual(expected.filter(({ auditId }) => auditId !== "ui-audit-043"));
    expect(LEGACY_SCREEN_MANIFEST.map(({ auditId }) => auditId)).toEqual(
      Array.from({ length: 86 }, (_, index) => `ui-audit-${String(index + 1).padStart(3, "0")}`)
        .filter((auditId) => auditId !== "ui-audit-043"),
    );
  });

  it("keeps the protected 86-row oracle while retiring the fixed-ID package builder from React", () => {
    const reactRows = LEGACY_SCREEN_MANIFEST.filter(
      ({ disposition }) => disposition === "react-parity",
    );
    const legacyRows = LEGACY_SCREEN_MANIFEST.filter(
      ({ disposition }) => disposition !== "react-parity",
    );

    expect(reactRows).toHaveLength(85);
    expect(legacyRows).toHaveLength(0);
    expect(reactRows.map(({ auditId }) => auditId)).toEqual(
      LEGACY_SCREEN_SOURCE.filter(({ auditId }) => auditId !== "ui-audit-043").map(({ auditId }) => auditId),
    );
  });

  it("keeps every React row aligned with the typed route registry", () => {
    for (const row of LEGACY_SCREEN_MANIFEST) {
      expect(row.productAuthority.length).toBeGreaterThan(0);
      expect(row.sourceEvidence.length).toBeGreaterThan(0);
      expect(row.reason.trim()).not.toBe("");
      if (row.disposition !== "react-parity") continue;

      expect(row.reactSurfaceId).not.toBeNull();
      const route = REACT_ROUTE_CONTRACT_BY_ID.get(row.reactSurfaceId!);
      expect(route).toBeDefined();
      expect(row.reactPath).toBe(route?.path);
      expect(row.dataBoundary).toBe(route?.dataBoundary);
      expect(routeRoleForSurface(row.reactSurfaceId!)).toBe(route?.requiredRole);
      expect(row.referenceScreenshotIds).toEqual([row.auditId]);
    }
  });

  it("maps every current repo-required screen outcome into the active route contract", () => {
    expect(PRODUCT_SCREEN_CROSSWALK.map(({ outcome }) => outcome)).toEqual([
      "Role switch / login",
      "Manager Dashboard",
      "Inspector Dashboard",
      "Audit Plan Calendar",
      "Audit Detail",
      "Checklist Runner",
      "Finding Detail with lifecycle stepper",
      "Auditee My Findings",
      "CAP Submission Form",
      "Evidence Upload / Review",
      "Closed Finding / Report Preview",
      "Admin Checklist Template Preview",
      "New Inspection Planning Intake",
    ]);

    const intake = PRODUCT_SCREEN_CROSSWALK.find(
      ({ outcome }) => outcome === "New Inspection Planning Intake",
    );
    expect(intake).toEqual({
      outcome: "New Inspection Planning Intake",
      delivery: "react-parity",
      reactSurfaceIds: [
        "audit-plan",
        "new-audit-wizard-1",
        "new-audit-wizard-2",
        "new-audit-wizard-3",
        "new-audit-wizard-4",
        "new-audit-wizard-5",
      ],
      sourceAuditIds: [
        "ui-audit-028",
        "ui-audit-047",
        "ui-audit-048",
        "ui-audit-049",
        "ui-audit-050",
        "ui-audit-051",
      ],
      disposition: "Frozen as Department Manager demo routes pending their owning feature slice.",
    });

    const wizardRows = LEGACY_SCREEN_MANIFEST.filter(
      ({ auditId }) => auditId >= "ui-audit-047" && auditId <= "ui-audit-051",
    );
    expect(wizardRows).toHaveLength(5);
    expect(wizardRows.every(({ disposition, reactPath }) => disposition === "react-parity" && reactPath.startsWith("/department-manager/new-audit/step-"))).toBe(true);
  });
});
