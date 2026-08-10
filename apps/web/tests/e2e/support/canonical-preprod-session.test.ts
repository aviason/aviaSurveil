import { describe, expect, it } from "vitest";

import { returnTargetMatches } from "./canonical-preprod-session";

describe("returnTargetMatches", () => {
  it("matches both the path and query string of an OIDC return target", () => {
    expect(
      returnTargetMatches(
        new URL("https://candidate.test/department-manager/audit-plan?planningItemId=plan-intake-123"),
        "/department-manager/audit-plan?planningItemId=plan-intake-123",
        "https://candidate.test",
      ),
    ).toBe(true);
    expect(
      returnTargetMatches(
        new URL("https://candidate.test/department-manager/audit-plan?planningItemId=plan-intake-other"),
        "/department-manager/audit-plan?planningItemId=plan-intake-123",
        "https://candidate.test",
      ),
    ).toBe(false);
  });
});
