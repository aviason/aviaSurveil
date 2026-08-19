import { describe, expect, it } from "vitest";

import {
  legacyQuiescentReloadFingerprint,
  releaseFingerprintFromActivationMessage,
} from "./app-shell-update-protocol";

describe("app-shell update protocol", () => {
  it("keeps the real release identity alongside the legacy reload token", () => {
    const releaseFingerprint = `sha256:${"a".repeat(64)}`;
    const legacyToken = legacyQuiescentReloadFingerprint(releaseFingerprint);

    expect(legacyToken).not.toBe(releaseFingerprint);
    expect(releaseFingerprintFromActivationMessage({ fingerprint: legacyToken })).toBe(
      releaseFingerprint,
    );
    expect(releaseFingerprintFromActivationMessage({
      fingerprint: legacyToken,
      releaseFingerprint,
    })).toBe(releaseFingerprint);
  });
});
