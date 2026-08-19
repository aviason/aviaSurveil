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

  it("does not widen an empty-history suggested view to the full 1,310-row catalog", async () => {
    const runtime = createMockBackendRuntime();
    const manager = runtime.backendForRole("manager");
    const common = { catalogVersion: "aga-approved-source@2.0.0", usageClass: "GOVERNED_OPERATIONAL" as const, limit: 2000, applicationType: "RAMP_INSPECTION" as const };
    const suggested = await manager.canonicalCatalog!.listCatalog({ ...common, includedByDefault: true });
    const full = await manager.canonicalCatalog!.listCatalog(common);
    expect(suggested.recommendationSummary.comparableAuditCount).toBe(0);
    expect(suggested.totalCount).toBeLessThan(full.totalCount);
    expect(full.totalCount).toBe(1310);
    expect(suggested.items.every((item) => item.recommendation.includedByDefault)).toBe(true);
  });

  it("matches the exact no-history scope-filter golden IDs across mock list projections", async () => {
    const profile = "prior-audit-no-history-scope-filter" as const;
    const fixture = priorAuditRecommendationFixtures[profile];
    const runtime = createMockBackendRuntime(() => "2026-08-18T12:00:00.000Z", profile);
    const manager = runtime.backendForRole("manager");
    const common = { catalogVersion: "aga-approved-source@2.0.0", usageClass: "GOVERNED_OPERATIONAL" as const, limit: 2000, applicationType: "RAMP_INSPECTION" as const };
    const suggested = await manager.canonicalCatalog!.listCatalog({ ...common, includedByDefault: true });
    const full = await manager.canonicalCatalog!.listCatalog(common);
    expect(fixture.priorAuditIds).toEqual([]);
    expect(fixture.declaredQuestionVersionIds?.slice().sort()).toEqual([
      "Q-NO-HISTORY-IN-FOCUS-OPTIONAL",
      "Q-NO-HISTORY-OUTSIDE-FOCUS-MANDATORY",
      "Q-NO-HISTORY-OUTSIDE-FOCUS-OPTIONAL",
      "Q-NO-HISTORY-WRONG-GENERAL-TYPE",
      "Q-NO-HISTORY-WRONG-PROVIDER",
      "Q-NO-HISTORY-WRONG-TARGET",
    ]);
    expect(fixture.excludedQuestionVersionIds?.slice().sort()).toEqual([
      "Q-NO-HISTORY-WRONG-GENERAL-TYPE",
      "Q-NO-HISTORY-WRONG-PROVIDER",
      "Q-NO-HISTORY-WRONG-TARGET",
    ]);
    expect(suggested.items.map((item) => item.questionVersionId).sort()).toEqual([
      "Q-NO-HISTORY-IN-FOCUS-OPTIONAL",
      "Q-NO-HISTORY-OUTSIDE-FOCUS-MANDATORY",
    ]);
    expect(full.items.map((item) => item.questionVersionId).sort()).toEqual([
      "Q-NO-HISTORY-IN-FOCUS-OPTIONAL",
      "Q-NO-HISTORY-OUTSIDE-FOCUS-MANDATORY",
      "Q-NO-HISTORY-OUTSIDE-FOCUS-OPTIONAL",
    ]);
    expect(suggested.totalCount).toBe(2);
    expect(full.totalCount).toBe(3);
    expect(full.items.map((item) => item.questionVersionId)).not.toEqual(expect.arrayContaining([
      "Q-NO-HISTORY-WRONG-PROVIDER",
      "Q-NO-HISTORY-WRONG-TARGET",
      "Q-NO-HISTORY-WRONG-GENERAL-TYPE",
    ]));
  });
});
