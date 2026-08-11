import { expect, test } from "@playwright/test";

import { VISUAL_SURFACES } from "./support/legacy-parity-fixtures";

test("clean Docker demo directly loads all 85 active routes", async ({ page }) => {
  test.setTimeout(300_000);
  expect(VISUAL_SURFACES).toHaveLength(85);
  const loaded: string[] = [];

  for (const surface of VISUAL_SURFACES) {
    const response = await page.goto(surface.reactPath, {
      waitUntil: "domcontentloaded",
    });
    expect(response?.status(), `direct load ${surface.reactPath}`).toBe(200);
    await expect(page.locator("main")).toHaveCount(1);
    expect(new URL(page.url()).pathname).toBe(surface.reactPath);
    loaded.push(surface.id);
  }

  expect(loaded).toHaveLength(85);
  expect(new Set(loaded).size).toBe(85);
});
