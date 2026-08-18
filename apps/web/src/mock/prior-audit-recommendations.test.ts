import { describe, expect, it } from "vitest";

import {
  evaluatePriorAuditRecommendations,
  fullCatalogPriorAuditQuestionIds,
  priorAuditRecommendationFixtures,
  suggestedPriorAuditQuestionIds,
} from "./prior-audit-recommendations";
import { createMockBackendRuntime } from "./create-mock-backend";

describe("prior-audit recommendation fixtures", () => {
  it("matches the multi-history golden oracle", () => {
    const suggested = suggestedPriorAuditQuestionIds("prior-audit-multi-history");
    const omitted = Object.keys(priorAuditRecommendationFixtures["prior-audit-multi-history"].recommendations).filter((id) => !suggested.includes(id));
    expect(priorAuditRecommendationFixtures["prior-audit-multi-history"].priorAuditIds).toHaveLength(3);
    expect(suggested).toHaveLength(7);
    expect(omitted).toEqual(["qv:aga-approved-source-v2:FSS-AGA-FORM-002:all-forms-preview-002-0005"]);
    expect(fullCatalogPriorAuditQuestionIds("prior-audit-multi-history")).toHaveLength(8);
    expect(priorAuditRecommendationFixtures["prior-audit-multi-history"].recommendations[omitted[0]]).toMatchObject({
      recommendationState: "RECENTLY_VERIFIED",
      classification: "DEFER_ELIGIBLE",
      includedByDefault: false,
      canDefer: true,
    });
  });

  it("keeps a single clean Audit uncertain and suggested", () => {
    const clean = priorAuditRecommendationFixtures["prior-audit-single-history"].recommendations["qv:aga-approved-source-v2:FSS-AGA-FORM-002:all-forms-preview-002-0015"];
    expect(priorAuditRecommendationFixtures["prior-audit-single-history"].priorAuditIds).toEqual(["AUD-PRIOR-SINGLE-001"]);
    expect(clean).toMatchObject({ recommendationState: "UNCERTAIN_SIGNAL", includedByDefault: true, canDefer: false });
    expect(evaluatePriorAuditRecommendations("prior-audit-single-history")).toHaveLength(5);
  });

  it("keeps the omitted multi-history question selectable through the full mock catalog", async () => {
    const runtime = createMockBackendRuntime(() => "2026-08-18T12:00:00.000Z", "prior-audit-multi-history");
    const manager = runtime.backendForRole("manager");
    const common = { catalogVersion: "aga-approved-source@2.0.0", usageClass: "GOVERNED_OPERATIONAL" as const, limit: 100 };
    const suggested = await manager.canonicalCatalog!.listCatalog({ ...common, includedByDefault: true });
    const full = await manager.canonicalCatalog!.listCatalog(common);
    expect(suggested.totalCount).toBe(7);
    expect(full.totalCount).toBe(8);
    expect(full.items.map((item) => item.questionVersionId)).toContain("qv:aga-approved-source-v2:FSS-AGA-FORM-002:all-forms-preview-002-0005");
    expect(suggested.items.map((item) => item.questionVersionId)).not.toContain("qv:aga-approved-source-v2:FSS-AGA-FORM-002:all-forms-preview-002-0005");
  });
});
