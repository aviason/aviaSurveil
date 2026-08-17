import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const sourceRoot = resolve(fileURLToPath(new URL(".", import.meta.url)), "..");

describe("app-shell version contract", () => {
  it("advances the service worker, offline readiness, and generated manifest together", () => {
    const serviceWorker = readFileSync(resolve(sourceRoot, "sw.ts"), "utf8");
    const offlineReadiness = readFileSync(resolve(sourceRoot, "offline/storage-readiness.ts"), "utf8");
    const viteConfig = readFileSync(resolve(sourceRoot, "../vite.config.ts"), "utf8");
    const manifestPlugin = readFileSync(resolve(sourceRoot, "../build/app-shell-manifest-plugin.ts"), "utf8");

    expect(serviceWorker).toContain('__AVIA_RELEASE_FINGERPRINT__');
    expect(serviceWorker).toContain('APP_SHELL_ACTIVATION_POLICY');
    expect(offlineReadiness).not.toMatch(/indexedDbSchemaVersion:\s*1,/);
    expect(viteConfig).toContain("app-shell-manifest-plugin");
    expect(manifestPlugin).toContain("canonicalAppShellManifestInput");
    expect(serviceWorker).not.toMatch(/indexedDbSchemaVersion:\s*1/);
  });
});
