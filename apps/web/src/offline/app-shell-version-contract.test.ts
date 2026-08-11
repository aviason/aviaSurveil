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

    expect(serviceWorker).toContain('AVIA_APP_SHELL_VERSION:000009');
    expect(offlineReadiness).toMatch(/appShellVersion:\s*9,/);
    expect(viteConfig).toMatch(/appShellVersion:\s*9,/);
  });
});
