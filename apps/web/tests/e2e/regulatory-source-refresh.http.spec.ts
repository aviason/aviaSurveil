import { expect, test } from "@playwright/test";

const apiURL = process.env.AVIA_HTTP_API_URL ?? "http://127.0.0.1:58081";
const testToken = process.env.AVIA_CANONICAL_TEST_TOKEN ?? "";

for (const viewport of [{ width: 1440, height: 900 }, { width: 390, height: 844 }]) {
  test(`HTTP legacy source-gap reconciliation remains candidate-only at ${viewport.width}x${viewport.height}`, async ({ page, request }) => {
    const issues: string[] = [];
    const commands: string[] = [];
    page.on("console", (message) => { if (["error", "warning"].includes(message.type())) issues.push(message.text()); });
    page.on("pageerror", (error) => issues.push(error.message));
    page.on("request", (outgoing) => {
      if (outgoing.method() !== "POST" || !outgoing.url().includes("/v1/")) return;
      commands.push(new URL(outgoing.url()).pathname.replace(/^\/api(?=\/v1\/)/, ""));
    });
    await page.setViewportSize(viewport);

    const reset = await request.post(`${apiURL}/__test/reset`, {
      headers: { "x-avia-test-token": testToken },
    });
    expect(reset.ok()).toBe(true);

    await page.goto("/department-manager/checklist-management");
    await page.getByRole("combobox", { name: "Experience" }).selectOption("admin");
    if (viewport.width < 600) await page.getByRole("button", { name: "Open navigation" }).click();
    await page.getByRole("link", { name: "Checklist Builder", exact: true }).click();
    await expect(page.getByTestId("admin-checklist-builder-page")).toBeVisible();
    await page.getByRole("button", { name: "Load candidate-only legacy source-gap Draft" }).click();

    await expect(page.getByText("SOURCE_MAPPING_REQUIRED").first()).toBeVisible();
    await expect(page.getByText("EXISTING CHECKLIST CANDIDATE").first()).toBeVisible();
    await expect(page.getByRole("button", { name: "Submit exact candidate for department review" })).toBeDisabled();
    await expect(page.getByText(/cannot be validated, automatically deferred, included in an executable published Audit package, or published/i)).toBeVisible();

    await page.getByRole("button", { name: "Activate source change and create reconciliation Draft" }).click();
    await expect(page.getByText("HYBRID RECONCILED").first()).toBeVisible();
    await expect(page.getByText("Legacy candidate comparison")).toBeVisible();
    await expect(page.getByText("Synthetic test-profile impact source")).toBeVisible();
    await expect(page.getByText("TECHNICAL REVIEW REQUIRED").first()).toBeVisible();
    await expect(page.getByText("Current-source activation:")).toBeVisible();
    await expect(page.getByRole("button", { name: "Submit exact candidate for department review" })).toBeEnabled();

    expect(commands.filter((path) => path === "/v1/admin/governed-checklist/generation-runs")).toHaveLength(2);
    expect(commands.filter((path) => path === "/v1/admin/governed-checklist/source-currentness-activations")).toHaveLength(1);
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    expect(issues).toEqual([]);
  });
}
