import { describe, expect, it, vi } from "vitest";

import { detectStaleDocument } from "./stale-document-guard";

describe("stale document guard", () => {
  it("detects an app entry asset that is absent from the current shell manifest", async () => {
    await expect(detectStaleDocument({
      entryURL: "https://candidate.test/assets/app-old123.js",
      loadManifest: async () => ({
        releaseFingerprint: `sha256:${"a".repeat(64)}`,
        files: [{ url: "/assets/app-new456.js" }],
      }),
    })).resolves.toBe(`sha256:${"a".repeat(64)}`);
  });

  it("does not navigate when the entry asset belongs to the current shell", async () => {
    const replace = vi.fn();
    const fingerprint = await detectStaleDocument({
      entryURL: "https://candidate.test/assets/app-current.js",
      loadManifest: async () => ({
        releaseFingerprint: `sha256:${"b".repeat(64)}`,
        files: [{ url: "/assets/app-current.js" }],
      }),
    });
    expect(fingerprint).toBeNull();
    expect(replace).not.toHaveBeenCalled();
  });
});
