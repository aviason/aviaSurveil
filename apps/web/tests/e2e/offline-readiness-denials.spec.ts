import { expect, test, type BrowserContext, type Page } from "@playwright/test";

import { OfflineStaticServer } from "./support/offline-static-server";

async function installStoragePolicy(
  context: BrowserContext,
  input: { persisted: boolean; persistResult: boolean; usage: number; quota: number },
): Promise<void> {
  await context.addInitScript((policy) => {
    Object.defineProperties(navigator.storage, {
      persisted: { configurable: true, value: async () => policy.persisted },
      persist: { configurable: true, value: async () => policy.persistResult },
      estimate: {
        configurable: true,
        value: async () => ({ usage: policy.usage, quota: policy.quota }),
      },
    });
  }, input);
}

function captureConsoleIssues(page: Page): string[] {
  const issues: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "warning" || message.type() === "error") {
      issues.push(`${message.type()}: ${message.text()}`);
    }
  });
  page.on("pageerror", (error) => issues.push(`pageerror: ${error.message}`));
  return issues;
}

test("managed-policy and persistence denial block only official offline checkout", async ({
  context,
  page,
}) => {
  const server = new OfflineStaticServer(4188);
  const consoleIssues = captureConsoleIssues(page);
  await installStoragePolicy(context, {
    persisted: false,
    persistResult: false,
    usage: 8 * 1024 * 1024,
    quota: 512 * 1024 * 1024,
  });
  try {
    await server.start();
    await page.goto(`${server.origin}/inspector/audits/AUD-2026-001`);
    await page.getByRole("button", { name: "Check out for offline use" }).click();
    await expect(page.locator('[data-readiness-code="managed-policy-unapproved"]')).toBeVisible();

    await page.getByLabel(/official browser\/version lane/i).check();
    await page.getByRole("button", { name: "Check out for offline use" }).click();
    await expect(
      page.locator('[data-readiness-code="ephemeral-or-unmanaged-storage"]'),
    ).toBeVisible();

    await page.getByLabel(/encrypted managed profile/i).check();
    await page.evaluate(async () => navigator.serviceWorker.ready);
    await page.getByRole("button", { name: "Check out for offline use" }).click();
    await expect(page.locator('[data-readiness-code="persistence-denied"]')).toContainText(
      /continue online-only/i,
    );
    await expect(page.getByText("Online inspection remains available")).toBeVisible();
    await expect(page.getByTestId("offline-package-status")).toHaveCount(0);
    expect(consoleIssues).toEqual([]);
  } finally {
    await server.stop().catch(() => undefined);
  }
});

test("advisory quota rejection reports required and available capacity", async ({
  context,
  page,
}) => {
  const server = new OfflineStaticServer(4189);
  const consoleIssues = captureConsoleIssues(page);
  await installStoragePolicy(context, {
    persisted: true,
    persistResult: true,
    usage: 96 * 1024 * 1024,
    quota: 100 * 1024 * 1024,
  });
  try {
    await server.start();
    await page.goto(`${server.origin}/inspector/audits/AUD-2026-001`);
    await page.getByLabel(/official browser\/version lane/i).check();
    await page.getByLabel(/encrypted managed profile/i).check();
    await page.evaluate(async () => navigator.serviceWorker.ready);
    await page.getByRole("button", { name: "Check out for offline use" }).click();
    const result = page.locator('[data-readiness-code="quota-insufficient"]');
    await expect(result).toBeVisible();
    await expect(result).toContainText(/Capacity estimate is advisory/i);
    await expect(page.getByTestId("offline-package-status")).toHaveCount(0);
    expect(consoleIssues).toEqual([]);
  } finally {
    await server.stop().catch(() => undefined);
  }
});
