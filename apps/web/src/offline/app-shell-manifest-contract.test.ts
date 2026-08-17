import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

import { canonicalAppShellManifestInput } from "./app-shell-manifest-contract";

describe("app-shell canonical manifest vectors", () => {
  it("matches the independent TypeScript test vector", () => {
    const vectorPath = resolve(process.cwd(), "build/app-shell-manifest-test-vectors.json");
    const [vector] = JSON.parse(readFileSync(vectorPath, "utf8")) as Array<{ manifestInput: Parameters<typeof canonicalAppShellManifestInput>[0]; expectedReleaseFingerprint: string }>;
    const fingerprint = `sha256:${createHash("sha256").update(canonicalAppShellManifestInput(vector.manifestInput)).digest("hex")}`;
    expect(fingerprint).toBe(vector.expectedReleaseFingerprint);
  });
});
