import { expect, test, type Locator, type Page } from "@playwright/test";

const viewports = [
  { id: "desktop", width: 1440, height: 900 },
  { id: "tablet", width: 1024, height: 768 },
  { id: "mobile", width: 390, height: 844 },
] as const;

const adminRoutes = [
  ["/admin/regulatory-library", "admin-regulatory-library-page", "Regulatory Library", "Candidate-only regulatory library"],
  ["/admin/template-library", "admin-template-list-page", "Templates", "CTV-CABIN-1"],
  ["/admin/templates", "admin-template-preview-page", "Templates", "CTV-CABIN-1"],
  ["/admin/question-bank", "admin-question-bank-page", "Question Bank", "CAB-GALLEY-001"],
  ["/admin/checklist-builder", "admin-checklist-builder-page", "Checklist Builder", "CTV-CABIN-1"],
  ["/admin/templates/TPL-CABIN-2026/history", "admin-version-history-page", "Version History", "CTV-CABIN-1"],
  ["/admin/inspection-package-builder", "admin-inspection-package-page", "Checklist Builder", "PKG-CAB-2026-001"],
  ["/admin/reports", "admin-reports-page", "Reports", "ADMIN-RPT-PACKAGE-001"],
  ["/admin/users-roles", "admin-users-roles-page", "Users / Roles", "USR-ADMIN-ADA"],
  ["/admin/configurations", "admin-configurations-page", "Configurations", "Oversight Health Index"],
  ["/admin/organization-master-data", "admin-organization-master-data-page", "Organisation Master Data", "ORG-FLY-NAMIBIA"],
  ["/admin/organization-master-data/ORG-FLY-NAMIBIA", "admin-organization-detail-page", "Organisation Master Data", "ORG-FLY-NAMIBIA"],
  ["/admin/audit-log", "admin-audit-log-page", "Audit Log", "AUDIT-REPORT-SEED-0001"],
] as const;

async function expectControlBounds(page: Page, target: Locator): Promise<void> {
  await target.scrollIntoViewIfNeeded();
  await expect(target).toBeVisible();
  const box = await target.boundingBox();
  const viewport = page.viewportSize();
  expect(box).not.toBeNull();
  expect(viewport).not.toBeNull();
  if (!box || !viewport) return;
  const identity = await target.evaluate((element) =>
    `${element.tagName.toLowerCase()}[${element.getAttribute("aria-label") ?? element.textContent?.trim().slice(0, 80) ?? "unlabelled"}]`,
  );
  expect(box.width, identity).toBeGreaterThanOrEqual(44);
  expect(box.height, identity).toBeGreaterThanOrEqual(44);
  expect(box.x, identity).toBeGreaterThanOrEqual(0);
  expect(Math.floor(box.x + box.width), identity).toBeLessThanOrEqual(viewport.width);
}

async function primaryNavigation(page: Page, width: number): Promise<Locator> {
  if (width > 900) {
    return page.locator(".workspace-sidebar").getByRole("navigation", { name: "Primary role navigation" });
  }
  const opener = page.getByRole("button", { name: "Open navigation" });
  await opener.click();
  await expect(opener).toHaveAttribute("aria-expanded", "true");
  return page.locator(".mobile-navigation__drawer").getByRole("navigation", { name: "Primary role navigation" });
}

for (const viewport of viewports) {
  test(`keeps all Administration workspaces exact and usable at ${viewport.width}px`, async ({ page }) => {
    await page.setViewportSize(viewport);
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });

    for (const [path, testId, activeLabel, marker] of adminRoutes) {
      await page.goto(path);
      const routePage = page.getByTestId(testId);
      await expect(routePage).toBeVisible();
      await expect(routePage).toContainText(marker);
      await expect.poll(() => page.evaluate(() =>
        document.documentElement.scrollWidth <= document.documentElement.clientWidth,
      )).toBe(true);
      const enabled = routePage.locator(
        "a[href]:visible, button:not(:disabled):visible, input:not(:disabled):visible, select:not(:disabled):visible, textarea:not(:disabled):visible",
      );
      for (let index = 0; index < await enabled.count(); index += 1) {
        await expectControlBounds(page, enabled.nth(index));
      }
      const navigation = await primaryNavigation(page, viewport.width);
      await expect(navigation.getByRole("link", { name: activeLabel })).toHaveAttribute("aria-current", "page");
      await expect(navigation.locator("a[aria-current='page']")).toHaveCount(1);
      if (viewport.width <= 900) {
        await navigation.getByRole("link", { name: activeLabel }).click();
        await expect(page.getByRole("dialog", { name: "Primary navigation" })).toBeHidden();
      }
    }

    await page.goto("/admin/regulatory-library");
    const regulatory = page.getByTestId("admin-regulatory-library-page");
    await regulatory.getByLabel("Search regulatory references").fill("NAMCARS-FOPS-004");
    await regulatory.getByLabel("Regulatory status").selectOption("SUPERSEDED");
    await expect(regulatory).toContainText("NAMCARS-FOPS-004");
    await expect(regulatory).not.toContainText("NAMCARS-CAB-001");
    await expect(regulatory).toContainText("Candidate-only configured reference");

    await page.goto("/admin/template-library");
    const templates = page.getByTestId("admin-template-list-page");
    await expect(templates.getByRole("link", { name: "Preview CTV-CABIN-1" })).toHaveAttribute("href", "/admin/templates");
    await expect(templates.getByRole("button", { name: /Open TPL-FOPS-2026 unavailable/ })).toBeDisabled();
    await templates.getByRole("link", { name: "Preview CTV-CABIN-1" }).click();
    await expect(page).toHaveURL(/\/admin\/templates$/);
    const preview = page.getByTestId("admin-template-preview-page");
    await expect(preview.getByRole("heading", { name: "Template Preview — Cabin Inspection" })).toBeVisible();
    await expect(preview.getByRole("link", { name: "Back to templates" })).toHaveAttribute("href", "/admin/template-library");
    await expect(preview.locator("[aria-label='Template section jumps']").getByRole("link")).toHaveCount(6);
    await preview.getByRole("link", { name: "Back to templates" }).click();
    await expect(page).toHaveURL(/\/admin\/template-library$/);

    await page.goto("/admin/question-bank");
    const questionBank = page.getByTestId("admin-question-bank-page");
    await questionBank.getByRole("button", { name: "Create question" }).click();
    await expect(questionBank.getByRole("alert")).toContainText("Question text is required");
    await questionBank.getByLabel("Question text").fill("Is the configured cabin record available?\nConfirm the expected Evidence version.");
    await questionBank.getByLabel("Configured reference").fill("Configured reference — CAB-RECORD");
    await questionBank.getByLabel("Expected Evidence").fill("Cabin record version");
    await expect(questionBank).toContainText("80 characters / 500");
    await questionBank.getByRole("button", { name: "Create question" }).click();
    await expect(questionBank.getByRole("status")).toContainText("Q-ADMIN-2026-007");
    await page.reload();
    await expect(page.getByTestId("admin-question-bank-page")).toContainText("Q-ADMIN-2026-007");
    await expect(page.getByTestId("admin-question-bank-page")).toContainText("Confirm the expected Evidence version.");

    await page.goto("/admin/checklist-builder");
    const builder = page.getByTestId("admin-checklist-builder-page");
    await expect(builder).toContainText("CTV-CABIN-1 · 6 exact questions · Revision 1");
    await builder.getByRole("button", { name: "Create working Draft" }).click();
    await expect(builder).toContainText("CTV-CABIN-DRAFT-2 · Revision 1 · Owner Admin Preview");
    await builder.getByLabel("Question to add").selectOption("Q-ADMIN-2026-007");
    await builder.getByRole("button", { name: "Add Q-ADMIN-2026-007 to CTV-CABIN-DRAFT-2" }).click();
    const addedQuestion = builder.locator(".admin-builder-list > li").filter({ hasText: "Q-ADMIN-2026-007" });
    await expect(addedQuestion).toBeVisible();
    await addedQuestion.getByRole("button", { name: "Move Q-ADMIN-2026-007 up in CTV-CABIN-DRAFT-2" }).click();
    await expect(builder).toContainText("CTV-CABIN-DRAFT-2 · Revision 3 · Owner Admin Preview");
    await page.reload();
    const remountedBuilder = page.getByTestId("admin-checklist-builder-page");
    await expect(remountedBuilder).toContainText("Q-ADMIN-2026-007");
    const remountedIds = await remountedBuilder.locator(".admin-builder-list small").allTextContents();
    expect(remountedIds.at(-2)).toBe("Q-ADMIN-2026-007");
    await expect(remountedBuilder).toContainText("CTV-CABIN-1 · 6 exact questions · Revision 1");
    await expect(remountedBuilder.getByRole("button", { name: /Publish CTV-CABIN-DRAFT-2 unavailable/ })).toBeDisabled();

    await page.goto("/admin/templates/TPL-CABIN-2026/history");
    const history = page.getByTestId("admin-version-history-page");
    await expect(history).toContainText("CTV-CABIN-1");
    await expect(history).toContainText("CTV-CABIN-DRAFT-2");
    await expect(history).toContainText("1 questions added, 0 removed, order unchanged");
    await expect(history.getByRole("button", { name: /Publish CTV-CABIN-DRAFT-2 unavailable/ })).toBeDisabled();

    await page.goto("/admin/inspection-package-builder");
    const adminPackage = page.getByTestId("admin-inspection-package-page");
    await expect(adminPackage).toContainText("PKG-CAB-2026-001");
    await expect(adminPackage).toContainText("AUD-2026-001");
    await expect(adminPackage).toContainText("ORG-FLY-NAMIBIA");
    await expect(adminPackage.getByRole("button", { name: /Run PKG-CAB-2026-001 unavailable/ })).toBeDisabled();

    await page.goto("/admin/reports");
    const reports = page.getByTestId("admin-reports-page");
    await reports.getByLabel("Search report definitions").fill("no matching definition");
    await expect(reports).toContainText("No matching report definition");
    await reports.getByLabel("Search report definitions").fill("package");
    await expect(reports).toContainText("ADMIN-RPT-PACKAGE-001");
    await expect(reports.getByRole("button", { name: /Generate ADMIN-RPT-PACKAGE-001 unavailable/ })).toBeDisabled();

    await page.goto("/admin/users-roles");
    const users = page.getByTestId("admin-users-roles-page");
    await users.getByLabel("Search users").fill("USR-AUDITEE-FLY");
    await users.getByLabel("User role").selectOption("auditee");
    await expect(users).toContainText("USR-AUDITEE-FLY");
    await expect(users).toContainText("ORG-FLY-NAMIBIA");
    await expect(users).toContainText("Not configured in demo");
    await expect(users.getByRole("button", { name: /Change role USR-AUDITEE-FLY unavailable.*Plan 3 Keycloak administration/ })).toBeDisabled();

    await page.goto("/admin/configurations");
    const configurations = page.getByTestId("admin-configurations-page");
    await expect(configurations).toContainText("No real email or SMS delivery is configured");
    await expect(configurations.getByRole("button", { name: /save/i })).toHaveCount(0);

    await page.goto("/admin/organization-master-data");
    const organizations = page.getByTestId("admin-organization-master-data-page");
    await organizations.getByLabel("Search organizations").fill("ORG-SKYCARGO");
    await expect(organizations).toContainText("ORG-SKYCARGO");
    await expect(organizations).not.toContainText("ORG-FLY-NAMIBIA");
    await expect(organizations.getByRole("button", { name: /Open ORG-SKYCARGO unavailable.*no declared contextual detail route/ })).toBeDisabled();
    await page.reload();
    const remountedOrganizations = page.getByTestId("admin-organization-master-data-page");
    await expect(remountedOrganizations).toContainText("ORG-FLY-NAMIBIA");
    await remountedOrganizations.getByRole("link", { name: "Open ORG-FLY-NAMIBIA" }).click();
    await expect(page).toHaveURL(/\/admin\/organization-master-data\/ORG-FLY-NAMIBIA$/);
    const organization = page.getByTestId("admin-organization-detail-page");
    await expect(organization).toContainText("Fly Namibia");
    await expect(organization).not.toContainText("ORG-SKYCARGO");
    await expect(organization.getByRole("button", { name: /Open risk ORG-FLY-NAMIBIA unavailable.*Department Manager-owned/ })).toBeDisabled();

    await page.goto("/admin/audit-log");
    const auditLog = page.getByTestId("admin-audit-log-page");
    await auditLog.getByLabel("Audit actor").fill("USR-MANAGER-NORA");
    await auditLog.getByLabel("Audit action").fill("report.decision_recorded");
    await auditLog.getByLabel("Audit entity").fill("PR-2026-018-V0");
    await auditLog.getByLabel("Audit origin").selectOption("MANUAL");
    await auditLog.getByLabel("Audit date text").fill("2026-06-15");
    await expect(auditLog).toContainText("AUDIT-REPORT-SEED-0001");
    await expect(auditLog).not.toContainText("AUDIT-SYSTEM-SEED-0001");
    await expect(auditLog).toContainText("DEPARTMENT_REVIEW → RETURNED");
    expect(consoleErrors).toEqual([]);
  });
}
