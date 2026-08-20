import { expect, test, type Locator, type Page } from "@playwright/test";

const viewports = [
  { id: "desktop", width: 1440, height: 900 },
  { id: "tablet", width: 1024, height: 768 },
  { id: "mobile", width: 390, height: 844 },
] as const;

const taskSevenRoutes = [
  ["/department-manager/risk-dashboard", "manager-risk-dashboard-page"],
  ["/department-manager/safety-intelligence", "manager-safety-intelligence-page"],
  ["/department-manager/organizations/ORG-FLY-NAMIBIA/risk-profile", "organization-risk-profile-page"],
  ["/department-manager/ssp-nasp", "manager-ssp-nasp-page"],
  ["/department-manager/usoap-readiness", "manager-usoap-readiness-page"],
  ["/department-manager/cap-effectiveness", "manager-cap-effectiveness-page"],
  ["/department-manager/new-audit/step-1", "new-audit-wizard-page"],
  ["/department-manager/new-audit/step-2", "new-audit-wizard-page"],
  ["/department-manager/new-audit/step-3", "new-audit-wizard-page"],
  ["/department-manager/new-audit/step-4", "new-audit-wizard-page"],
  ["/department-manager/new-audit/step-5", "new-audit-wizard-page"],
] as const;

async function expectInsideViewport(page: Page, target: Locator): Promise<void> {
  await target.scrollIntoViewIfNeeded();
  await expect(target).toBeVisible();
  const box = await target.boundingBox();
  const viewport = page.viewportSize();
  expect(box).not.toBeNull();
  expect(viewport).not.toBeNull();
  if (!box || !viewport) return;
  expect(box.x).toBeGreaterThanOrEqual(0);
  expect(box.y).toBeGreaterThanOrEqual(0);
  expect(Math.floor(box.x + box.width)).toBeLessThanOrEqual(viewport.width);
  expect(Math.floor(box.y + box.height)).toBeLessThanOrEqual(viewport.height);
}

async function managerNavigation(page: Page, width: number): Promise<Locator> {
  if (width > 900) return page.locator(".workspace-sidebar").getByRole("navigation", { name: "Primary role navigation" });
  const opener = page.getByRole("button", { name: "Open navigation" });
  await opener.click();
  await expect(opener).toHaveAttribute("aria-expanded", "true");
  return page.locator(".mobile-navigation__drawer").getByRole("navigation", { name: "Primary role navigation" });
}

for (const viewport of viewports) {
  test(`keeps Manager intelligence, package, and one intake draft usable at ${viewport.width}px`, async ({ page }) => {
    await page.setViewportSize(viewport);
    for (const [route, testId] of taskSevenRoutes) {
      await page.goto(route);
      const routePage = page.getByTestId(testId);
      await expect(routePage).toBeVisible();
      await expect(routePage.locator("a[href], button, input, select, textarea").first()).toBeVisible();
      await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    }

    await page.goto("/department-manager/risk-dashboard");
    const riskPage = page.getByTestId("manager-risk-dashboard-page");
    await expect(riskPage).toBeVisible();
    for (const label of ["Date Range", "Department", "Inspection", "Risk Level"]) {
      await expect(riskPage.getByLabel(label)).toBeEnabled();
    }
    await riskPage.getByLabel("Risk Level").selectOption("HIGH");
    await expect(riskPage).toContainText("No Finding records match the active risk filters.");
    await riskPage.getByRole("button", { name: "Reset filters" }).click();
    await expect(riskPage).toContainText("FND-SKYCARGO-2026-099");
    await expect(riskPage.getByRole("link", { name: "Export CSV" })).toHaveAttribute("download", /\.csv$/);

    const navigation = await managerNavigation(page, viewport.width);
    await expect(navigation.getByRole("link", { name: "Risk Dashboard" })).toHaveAttribute("aria-current", "page");
    await expect(navigation.locator("a[aria-current='page']")).toHaveCount(1);
    if (viewport.width <= 900) await page.keyboard.press("Escape");

    const safetyLink = riskPage.getByRole("link", { name: "Open Safety Intelligence" });
    await expectInsideViewport(page, safetyLink);
    await safetyLink.click();
    await expect(page).toHaveURL(/\/department-manager\/safety-intelligence$/);
    const safetyPage = page.getByTestId("manager-safety-intelligence-page");
    await expect(safetyPage).toBeVisible();
    await safetyPage.getByLabel("Signal filter").selectOption("overdue");
    await expect(safetyPage.getByTestId("active-signal-filter")).toHaveText("overdue");
    await safetyPage.getByLabel("Signal filter").selectOption("all");
    await safetyPage.getByRole("link", { name: "Open risk profile for ORG-FLY-NAMIBIA" }).click();
    await expect(page).toHaveURL(/\/department-manager\/organizations\/ORG-FLY-NAMIBIA\/risk-profile$/);
    await expect(page.getByTestId("organization-risk-profile-page")).toContainText("ORG-FLY-NAMIBIA");

    for (const [route, heading] of [
      ["/department-manager/ssp-nasp", "SSP/NASP Management Dashboard"],
      ["/department-manager/usoap-readiness", "USOAP Readiness Workspace"],
      ["/department-manager/cap-effectiveness", "CAP Effectiveness"],
    ] as const) {
      await page.goto(route);
      await expect(page.getByRole("heading", { level: 1, name: heading })).toBeVisible();
      await expectInsideViewport(page, page.getByRole("link", { name: "Back to Risk Dashboard" }));
    }

    await page.goto("/department-manager/audit-plan");
    const intakeLink = page.getByRole("link", { name: "New Audit planning intake" });
    await expectInsideViewport(page, intakeLink);
    await intakeLink.click();
    await page.getByRole("button", { name: "Continue", exact: true }).click();
    await expect(page.getByLabel("Purpose", { exact: true })).toBeVisible();
    await page.getByLabel("Purpose", { exact: true }).fill("Targeted operational control verification");
    await page.getByRole("button", { name: "Continue", exact: true }).click();
    await expect(page.getByLabel("Planned date", { exact: true })).toBeVisible();
    await page.getByLabel("Planned date", { exact: true }).fill("2026-12-10");
    await page.getByRole("button", { name: "Continue", exact: true }).click();
    await expect(page.getByLabel("Required inspectors", { exact: true })).toBeVisible();
    await page.getByLabel("Requested budget", { exact: true }).fill("0");
    await page.getByRole("button", { name: "Continue to review", exact: true }).click();
    const reviewPage = page.getByTestId("new-audit-wizard-page");
    await expect(reviewPage).toContainText("Approval context");
    const submit = reviewPage.getByRole("button", { name: "Submit to Finance" });
    await expectInsideViewport(page, submit);
    await submit.click();
    await expect(page).toHaveURL(/\/department-manager\/audit-plan\?planningItemId=/);
    await expect(page.getByTestId("planning-selected-record")).toContainText("Selected plan:");

  });
}
