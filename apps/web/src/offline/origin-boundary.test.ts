import { describe, expect, it } from "vitest";

import { assertExactOrigin } from "./origin-boundary";

describe("exact origin boundary", () => {
  it("accepts only the exact scheme/host/port and rejects aliases or redirects", () => {
    expect(assertExactOrigin("https://prod.example", "https://prod.example")).toMatchObject({ ok: true, origin: "https://prod.example" });
    expect(assertExactOrigin("https://alias.example", "https://prod.example")).toMatchObject({ ok: false, code: "ORIGIN_MISMATCH" });
    expect(assertExactOrigin("http://prod.example", "https://prod.example")).toMatchObject({ ok: false, code: "ORIGIN_MISMATCH" });
    expect(assertExactOrigin("https://prod.example", "")).toMatchObject({ ok: false, code: "ORIGIN_UNCONFIGURED" });
  });
});
