import { expect, test } from "@playwright/test";

test("Department Manager creates and submits a governed inspection brief", async ({ page }) => {
  test.setTimeout(45_000);
  await page.setViewportSize({ width: 390, height: 844 });
  const consoleIssues: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error" || message.type() === "warning") {
      consoleIssues.push(`${message.type()}: ${message.text()}`);
    }
  });
  page.on("pageerror", (error) => consoleIssues.push(`pageerror: ${error.message}`));

  await page.goto("/department-manager/new-audit/step-1");
  await expect(page.getByRole("heading", { name: "New Inspection" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Continue", exact: true })).toHaveCount(1);
  await expect(page.getByText("Open audit setup for this supplier", { exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Save draft", exact: true })).toHaveCount(0);

  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-2\?draftId=/);
  await expect(page.getByText("Saved", { exact: true }).first()).toHaveCount(1);

  await page.getByLabel("Purpose", { exact: true }).fill("Verify cabin safety controls before the scheduled surveillance visit.");
  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-3\?draftId=/);
  await page.getByLabel("Planned date", { exact: true }).fill("2026-12-10");
  await page.getByLabel("Location", { exact: true }).fill("Windhoek");
  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-4\?draftId=/);

  const advancedFilters = page.locator(".planning-intake-advanced-filters");
  await expect(advancedFilters).not.toHaveAttribute("open", "");
  await advancedFilters.locator("> summary").click();
  await expect(page.getByLabel("Source gap filter")).toBeVisible();
  await expect(page.getByLabel("Recommendation filter")).toBeVisible();
  await expect(page.getByLabel("Selected state filter")).toBeVisible();
  await page.getByLabel("Recommendation filter").selectOption("");
  await expect(page.getByRole("heading", { name: "Full approved catalog", exact: true })).toBeVisible();
  await expect(advancedFilters).toContainText("1 active");
  await expect(page.getByText("Showing the complete approved catalog.", { exact: false })).toBeVisible();
  await page.getByRole("button", { name: "Clear filters", exact: true }).click();
  await expect(page.getByLabel("Recommendation filter")).toHaveValue("SUGGESTED_NOW");
  await expect(page.getByRole("heading", { name: "Suggested questions", exact: true })).toBeVisible();
  await advancedFilters.locator("> summary").click();

  await page.getByRole("checkbox", { name: /Select / }).first().check();
  await page.getByRole("button", { name: "Review selection", exact: true }).click();
  const selectionDialog = page.getByRole("dialog", { name: "Review selection" });
  await expect(selectionDialog).toBeVisible();
  await selectionDialog.getByRole("button", { name: "Confirm selection" }).click();
  await expect(page.getByText("Selection confirmed and saved to the server-owned scope.")).toBeVisible();

  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-5\?draftId=/);
  await expect(page.getByText("0 USD", { exact: true }).first()).toBeVisible();
  await expect(page.getByRole("button", { name: "Preview", exact: true })).toHaveCount(0);
  await expect(page.getByText(/Submit creates a Planning item for Finance Review\./)).toBeVisible();
  await expect(page.getByText("Create Audit", { exact: true })).toHaveCount(0);
  await expect(page.getByText("Start inspection", { exact: true })).toHaveCount(0);

  await page.getByRole("button", { name: "Submit to Finance", exact: true }).click();
  await expect(page).toHaveURL(/\/department-manager\/audit-plan\?planningItemId=/);
  await expect(page.getByTestId("planning-selected-record")).toContainText("Selected plan:");
  await expect(page.getByTestId("planning-selected-record")).toContainText("Plan #");
  await expect(page.getByTestId("planning-selected-record")).toContainText("Fly Namibia");
  expect(consoleIssues).toEqual([]);
});
