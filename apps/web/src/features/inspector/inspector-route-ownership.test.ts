import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

const routeOwnerFiles = [
  "../findings/lead-review-page.tsx",
  "../findings/manager-dashboard-page.tsx",
  "../evidence/evidence-review-page.tsx",
  "../caps/cap-review-page.tsx",
] as const;

describe("Inspector Finding route ownership", () => {
  it("contains no stale Lead Inspector Finding-detail URLs", () => {
    const sources = routeOwnerFiles.map((file) => readFileSync(resolve(import.meta.dirname, file), "utf8"));
    expect(sources.join("\n")).not.toContain("/lead-inspector/findings/");
  });

  it("uses the declared Lead CAP route or an explicit role handoff for cross-role Finding access", () => {
    const leadReview = readFileSync(resolve(import.meta.dirname, "../findings/lead-review-page.tsx"), "utf8");
    const managerDashboard = readFileSync(resolve(import.meta.dirname, "../findings/manager-dashboard-page.tsx"), "utf8");
    const evidenceReview = readFileSync(resolve(import.meta.dirname, "../evidence/evidence-review-page.tsx"), "utf8");
    const capReview = readFileSync(resolve(import.meta.dirname, "../caps/cap-review-page.tsx"), "utf8");
    expect(leadReview).toContain("/lead-inspector/cap-review/");
    expect(managerDashboard).toContain("/department-manager/findings-review");
    expect(evidenceReview).toContain("/department-manager/findings-review");
    expect(capReview).toContain("Switch to CAA Inspector Finding");
    expect(capReview).toContain("targetRole=\"inspector\"");
  });
});
