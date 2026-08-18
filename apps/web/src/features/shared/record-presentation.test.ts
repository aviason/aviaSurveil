import { describe, expect, it } from "vitest";

import { auditLabel, planningItemLabel, potentialFindingLabel, recordReference } from "./record-presentation";

describe("record presentation", () => {
  it("uses short, stable references instead of technical aggregate identifiers", () => {
    expect(recordReference("Plan", "plan-intake-08170c7cc0cc018834ef74ec")).toBe("Plan #F74EC");
    expect(recordReference("Audit", "inspection:assignment:plan-intake-08170c7cc0cc018834ef74ec")).toBe("Audit #F74EC");
    expect(recordReference("Finding", "FINDING-2026-001")).toBe("Finding 2026-001");
  });

  it("makes the business title the primary record label", () => {
    expect(planningItemLabel("Routine / Announced — Namibia AGA Qualification Operator", "plan-intake-abcde")).toBe("Routine / Announced — Namibia AGA Qualification Operator · Plan #ABCDE");
    expect(auditLabel("Cabin safety inspection", "inspection:assignment:abcde")).toBe("Cabin safety inspection · Audit #ABCDE");
    expect(potentialFindingLabel("PBE serviceability not confirmed", "potential-finding-abcde")).toBe("PBE serviceability not confirmed · Potential finding #ABCDE");
  });
});
