import { test } from "@playwright/test";

test.describe("governed checklist intake (HTTP)", () => {
  test.skip("blocked: live PostgreSQL/object-store/parser dependencies are not available in this candidate workspace", async () => {
    // The focused HTTP contract is covered by the fetch parity test; this
    // browser profile remains an explicit blocked external dependency.
  });
});
