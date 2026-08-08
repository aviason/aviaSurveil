import { describe, expect, it } from "vitest";

import { createHttpBackend } from "./http-backend";

describe("AGA demo workspace HTTP client", () => {
  it("does not expose the donor workspace capability in the normal HTTP graph", () => {
    const backend = createHttpBackend({ apiBaseUrl: "/", environmentLabel: "Test" });
    expect(backend.agaDemoWorkspace).toBeUndefined();
  });
});
