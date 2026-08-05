import { expect, test } from "@playwright/test";

import {
  loginQualificationAccount,
  logout,
  requiredEnvironment,
} from "./support/aga-candidate-demo";

test("lifecycle routes retain no identifiers in browser URL or Web Storage", async ({ page }) => {
  test.setTimeout(120_000);
  await loginQualificationAccount(
    page,
    requiredEnvironment("AVIA_AGA_OIDC_INSPECTOR_USERNAME"),
    "/inspector/aga-demo-workspace/inspection",
  );
  const location = new URL(page.url());
  expect(location.search).toBe("");
  expect(location.hash).toBe("");
  expect(page.url()).not.toMatch(/(?:aga-ws|inspection|finding|cap|evidence|questionKey|digest)[=:]/iu);
  const retained = await page.evaluate(async () => ({
    localStorage: Object.keys(localStorage),
    sessionStorage: Object.keys(sessionStorage),
    indexedDb: "databases" in indexedDB ? (await indexedDB.databases()).map((entry) => entry.name).filter(Boolean) : [],
  }));
  expect(retained.localStorage).toEqual([]);
  expect(retained.sessionStorage).toEqual([]);
  expect(retained.indexedDb).toEqual([]);
  await logout(page);
});

test("BFCache restoration leaves the lifecycle projection cleared", async ({ page }) => {
  test.setTimeout(120_000);
  await loginQualificationAccount(
    page,
    requiredEnvironment("AVIA_AGA_OIDC_INSPECTOR_USERNAME"),
    "/inspector/aga-demo-workspace/inspection",
  );
  await page.evaluate(() => {
    window.dispatchEvent(new PageTransitionEvent("pageshow", { persisted: true }));
  });
  await expect(page.getByRole("alert")).toContainText(/restored page was cleared/i);
  await logout(page);
});
