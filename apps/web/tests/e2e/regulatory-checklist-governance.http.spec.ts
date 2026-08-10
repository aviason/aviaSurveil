import { expect, test } from "@playwright/test";

const apiURL = process.env.AVIA_HTTP_API_URL ?? "http://127.0.0.1:58081";
const testToken = process.env.AVIA_CANONICAL_TEST_TOKEN ?? "";

for (const viewport of [{ width: 1440, height: 900 }, { width: 390, height: 844 }]) {
  test(`HTTP governed checklist publication boundary is accessible and overflow-free at ${viewport.width}x${viewport.height}`, async ({ page, request }) => {
    const issues: string[] = [];
    const commands: string[] = [];
    page.on("console", (message) => { if (["error", "warning"].includes(message.type())) issues.push(message.text()); });
    page.on("pageerror", (error) => issues.push(error.message));
    page.on("request", (outgoing) => {
      const pathname = new URL(outgoing.url()).pathname;
      if (outgoing.method() === "POST" && pathname.startsWith("/api/v1/")) {
        commands.push(pathname.replace(/^\/api(?=\/v1\/)/, ""));
      }
    });
    await page.setViewportSize(viewport);

    const reset = await request.post(`${apiURL}/__test/reset`, {
      headers: { "x-avia-test-token": testToken },
    });
    expect(reset.ok()).toBe(true);

    await page.goto("/department-manager/checklist-management");
    await expect(page.getByTestId("governed-checklist-review")).toBeVisible();
    await expect(page.getByRole("heading", { name: "No current governed candidates" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Question Review" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Technically approve" })).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Publish checklist version" })).toHaveCount(0);

    await page.getByRole("combobox", { name: "Experience" }).selectOption("admin");
    if (viewport.width < 600) await page.getByRole("button", { name: "Open navigation" }).click();
    await page.getByRole("link", { name: "Checklist Builder", exact: true }).click();
    await expect(page.getByTestId("admin-checklist-builder-page")).toBeVisible();
    await page.getByRole("button", { name: "Import synthetic governed candidate" }).click();
    await expect(page.getByText("SYNTHETIC-OPS-AOC").first()).toBeVisible();
    await page.getByRole("button", { name: "Inspect governed generation run" }).click();
    await expect(page.getByRole("button", { name: "Submit exact candidate for department review" })).toBeEnabled();
    await page.getByRole("button", { name: "Submit exact candidate for department review" }).click();
    await expect(page.getByText("Submitted for department review. Admin has no technical approval or publication action.")).toBeVisible();

    await page.getByRole("combobox", { name: "Experience" }).selectOption("manager");
    if (viewport.width < 600) await page.getByRole("button", { name: "Open navigation" }).click();
    await page.getByRole("link", { name: "Checklist Management", exact: true }).click();
    const review = page.getByTestId("governed-checklist-review");
    await expect(review).toContainText("DEPARTMENT_REVIEW");
    await review.getByLabel("Decision reason").fill("Technically approve only the exact synthetic candidate.");
    await review.getByRole("button", { name: "Technically approve" }).click();
    await expect(review).toContainText("TECHNICALLY_APPROVED");
    await review.getByLabel("Decision reason").fill("Publish only the separately approved synthetic revision.");
    await review.getByRole("button", { name: "Publish checklist version" }).click();
    await expect(review).toContainText("PUBLISHED");
    await expect(review.getByText(/Published as immutable Checklist Template Version/i)).toBeVisible();

    expect(commands).toEqual([
      "/v1/admin/governed-checklist/generation-runs",
      "/v1/admin/governed-checklist/candidates/CAND-SYNTHETIC-OPS-AOC-0001/submissions",
      "/v1/department-manager/governed-checklist/candidates/CAND-SYNTHETIC-OPS-AOC-0001/technical-approvals",
      "/v1/department-manager/governed-checklist/candidates/CAND-SYNTHETIC-OPS-AOC-0001/publications",
    ]);
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    expect(issues).toEqual([]);
  });
}
