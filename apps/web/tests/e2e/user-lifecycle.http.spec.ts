import { expect, test, type Locator, type Page } from "@playwright/test";

const apiURL = process.env.AVIA_HTTP_API_URL ?? "http://127.0.0.1:58081";
const token = process.env.AVIA_CANONICAL_TEST_TOKEN ?? "";

test.beforeEach(async ({ request }) => {
  const response = await request.post(`${apiURL}/__test/reset`, {
    headers: { "x-avia-test-token": token },
  });
  expect(response.ok()).toBe(true);
});

async function waitForSucceeded(page: Page): Promise<Locator> {
  const status = page.getByRole("status", {
    name: "Lifecycle request status",
  });
  await expect(status).toBeVisible();
  await expect.poll(async () => {
    let value = (await status.locator("header > span").textContent())?.trim();
    if (value !== "SUCCEEDED") {
      const refresh = status.getByRole("button", {
        name: /Refresh (?:provisioning|lifecycle) status|Retry lifecycle status/,
      });
      if (await refresh.isEnabled()) await refresh.click();
    }
    await page.waitForTimeout(150);
    value = (await status.locator("header > span").textContent())?.trim();
    const summary = (await status.locator(
      ".admin-lifecycle-status__summary",
    ).textContent())?.trim();
    return `${value}:${summary}`;
  }, {
    intervals: [100, 200, 300, 500],
    timeout: 30_000,
  }).toBe("SUCCEEDED:Succeeded");
  return status;
}

async function expectWithinViewport(page: Page, target: Locator): Promise<void> {
  await target.scrollIntoViewIfNeeded();
  const box = await target.boundingBox();
  const viewport = page.viewportSize();
  expect(box).not.toBeNull();
  expect(viewport).not.toBeNull();
  if (!box || !viewport) return;
  expect(box.x).toBeGreaterThanOrEqual(0);
  expect(box.y).toBeGreaterThanOrEqual(0);
  expect(Math.ceil(box.x + box.width)).toBeLessThanOrEqual(viewport.width);
  expect(Math.ceil(box.y + box.height)).toBeLessThanOrEqual(viewport.height);
}

test("user lifecycle stays reasoned, revision-bound, keyboard-safe, and responsive over HTTP", async ({
  page,
}) => {
  const consoleIssues: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "warning" || message.type() === "error") {
      consoleIssues.push(`${message.type()}: ${message.text()}`);
    }
  });
  page.on("pageerror", (error) => {
    consoleIssues.push(`pageerror: ${error.message}`);
  });

  await page.goto("/admin/users-roles");
  const workspace = page.getByTestId("admin-users-roles-page");
  await expect(workspace).toBeVisible();
  await expect(workspace.getByRole("status")).toContainText(
    /Loading user directory|No accounts match/,
  );

  await workspace.getByRole("button", { name: "Create user" }).click();
  await workspace.getByLabel("Provisioning email").fill(
    "task5.lifecycle@example.test",
  );
  await workspace.getByLabel("Provisioning display name").fill(
    "Task Five Inspector",
  );
  await workspace.getByLabel("Provisioning reason").fill(
    "Approved focused HTTP lifecycle proof.",
  );
  await workspace.getByLabel("Provisioning organization").fill("CAA");
  await workspace.getByLabel("Provisioning role").selectOption("inspector");
  await workspace.getByRole("button", { name: "Review provisioning" }).click();

  let dialog = page.getByRole("dialog", {
    name: "Confirm Provision for task5.lifecycle@example.test",
  });
  await expect(dialog).toBeVisible();
  await expect(dialog.getByRole("button", {
    name: "Close confirmation",
  })).toBeFocused();
  await expect(dialog).toContainText("Approved focused HTTP lifecycle proof.");
  await dialog.getByRole("button", { name: "Confirm Provision" }).click();
  await waitForSucceeded(page);

  await workspace.getByLabel("Search users").fill(
    "task5.lifecycle@example.test",
  );
  const account = workspace.getByRole("listitem").filter({
    hasText: "task5.lifecycle@example.test",
  });
  await expect(account).toBeVisible();
  for (const fact of [
    "Provider account",
    "Application profile",
    "Role",
    "Organization",
    "Invitation",
    "MFA",
    "Required actions",
    "Desired membership",
    "Authority alignment",
    "Last successful session",
    "Provider observed",
  ]) {
    await expect(account.getByText(fact, { exact: true })).toBeVisible();
  }
  await expect(account).toContainText("inspector");
  await expect(account).toContainText("CAA");
  await expect(account.getByRole("button", {
    name: /Reset MFA .* unavailable: MFA reset requires an enrolled provider authenticator\./,
  })).toBeDisabled();

  const updateRole = account.getByRole("button", { name: /^Update role / });
  await updateRole.click();
  dialog = page.getByRole("dialog", {
    name: /Confirm Update role for /,
  });
  await dialog.getByLabel("New role").selectOption("manager");
  await dialog.getByLabel("Action reason").fill(
    "Approved exact CAA role replacement.",
  );
  await dialog.getByRole("button", { name: "Confirm Update role" }).click();
  await waitForSucceeded(page);
  await expect(account).toContainText("manager");

  const forceLogout = account.getByRole("button", { name: /^Force logout / });
  await forceLogout.click();
  dialog = page.getByRole("dialog", {
    name: /Confirm Force logout for /,
  });
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  await expect(forceLogout).toBeFocused();

  for (const viewport of [
    { width: 1440, height: 900 },
    { width: 1024, height: 768 },
    { width: 390, height: 844 },
  ]) {
    await page.setViewportSize(viewport);
    await expect.poll(() => page.evaluate(() =>
      document.documentElement.scrollWidth <=
        document.documentElement.clientWidth,
    )).toBe(true);
    await expectWithinViewport(page, workspace.getByLabel("Search users"));
    await expectWithinViewport(page, forceLogout);
  }

  await forceLogout.click();
  dialog = page.getByRole("dialog", {
    name: /Confirm Force logout for /,
  });
  await expectWithinViewport(page, dialog.locator("form"));
  await dialog.getByLabel("Action reason").fill(
    "Approved focused session revocation proof.",
  );
  await dialog.getByRole("button", { name: "Confirm Force logout" }).click();
  await waitForSucceeded(page);

  expect(consoleIssues).toEqual([]);
});
