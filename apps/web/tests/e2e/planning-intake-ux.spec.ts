import { expect, test } from "@playwright/test";

test("Department Manager creates and submits a New Audit Planning proposal", async ({ page }) => {
  test.setTimeout(45_000);
  await page.setViewportSize({ width: 390, height: 844 });
  const consoleIssues: string[] = [];
  page.on("console", (message) => { if (message.type() === "error" || message.type() === "warning") consoleIssues.push(`${message.type()}: ${message.text()}`); });
  page.on("pageerror", (error) => consoleIssues.push(`pageerror: ${error.message}`));

  await page.goto("/department-manager/new-audit/step-1");
  await expect(page.getByRole("heading", { name: "New Audit" })).toBeVisible();
  await expect(page.getByLabel("Inspected Organization", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Continue", exact: true })).toHaveCount(1);
  await expect(page.getByText("Open audit setup for this supplier", { exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Save draft", exact: true })).toHaveCount(0);
  await expect(page.locator(".planning-intake-brief__mobile > summary")).toContainText("Audit plan summary");

  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-2\?draftId=/);
  await expect(page.getByLabel("Purpose", { exact: true })).toBeVisible();
  await page.getByLabel("Purpose", { exact: true }).fill("Verify the operating controls remain effective before the scheduled surveillance visit.");
  await page.getByRole("button", { name: "Continue", exact: true }).click();

  await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-3\?draftId=/);
  await page.getByLabel("Planned date", { exact: true }).fill("2026-12-10");
  await page.getByRole("button", { name: "Continue", exact: true }).click();

  await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-4\?draftId=/);
  await expect(page.getByLabel("Required inspectors", { exact: true })).toBeVisible();
  await expect(page.getByLabel("Estimated checklist items", { exact: true })).toBeVisible();
  await page.getByLabel("Requested budget", { exact: true }).fill("0");
  await page.getByRole("button", { name: "Browse checklist items", exact: true }).click();
  const preview = page.getByRole("dialog", { name: "Checklist item preview" });
  await expect(preview).toBeVisible();
  await expect(preview).toContainText("does not select or freeze checklist items");
  await preview.getByRole("button", { name: "Close", exact: true }).click();
  await page.getByRole("button", { name: "Continue to review", exact: true }).click();

  await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-5\?draftId=/);
  await expect(page.getByRole("heading", { name: "Approval context" })).toBeVisible();
  await expect(page.getByText(/does not create an executable Audit/)).toBeVisible();
  await expect(page.getByRole("button", { name: "Submit to Finance", exact: true })).toBeEnabled();
  await page.getByRole("button", { name: "Submit to Finance", exact: true }).click();
  await expect(page).toHaveURL(/\/department-manager\/audit-plan\?planningItemId=/);
  await expect(page.getByTestId("planning-selected-record")).toContainText("Selected plan:");
  expect(consoleIssues).toEqual([]);
});
