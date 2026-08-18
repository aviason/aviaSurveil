import { chromium, type BrowserContext, type Page } from "@playwright/test";

export async function launchManagedPersistentContext(profileDirectory: string): Promise<BrowserContext> {
  const context = await chromium.launchPersistentContext(profileDirectory, {
    channel: "chrome",
    headless: true,
    viewport: { width: 1440, height: 900 },
    serviceWorkers: "allow",
  });
  await context.addInitScript(() => {
    Object.defineProperties(navigator.storage, {
      persisted: { configurable: true, value: async () => true },
      persist: { configurable: true, value: async () => true },
      estimate: {
        configurable: true,
        value: async () => ({ usage: 8 * 1024 * 1024, quota: 512 * 1024 * 1024 }),
      },
    });
  });
  return context;
}

export async function activePage(context: BrowserContext): Promise<Page> {
  const page = context.pages()[0] ?? (await context.newPage());
  return page;
}

export async function attestAndRequestCheckout(page: Page): Promise<void> {
  await page.getByLabel(/official browser\/version lane/i).check();
  await page.getByLabel(/encrypted managed profile/i).check();
  await page.getByRole("button", { name: "Check out for offline use" }).click();
}
