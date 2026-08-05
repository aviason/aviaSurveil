import { expect, test } from "@playwright/test";

import {
  loginQualificationAccount,
  logout,
  requiredEnvironment,
} from "./support/aga-candidate-demo";

const managerWorkspaceRoute = "/department-manager/aga-demo-workspace";
const adminWorkspaceRoute = "/admin/aga-demo-workspace";

function assertFixedWorkspaceLocation(page: import("@playwright/test").Page, route: string): void {
  const current = new URL(page.url());
  expect(current.pathname).toBe(route);
  expect(current.search).toBe("");
  expect(current.hash).toBe("");
  expect(page.url()).not.toMatch(/(?:inspection|finding|cap|evidence|draft|question)[=:]/iu);
}

test("Department Manager opens the bounded classification and lifecycle workspace", async ({ page }) => {
  test.setTimeout(120_000);
  await loginQualificationAccount(
    page,
    requiredEnvironment("AVIA_AGA_OIDC_MANAGER_USERNAME"),
    managerWorkspaceRoute,
  );
  await expect(page.getByRole("heading", { name: "AGA classification review" })).toBeVisible();
  assertFixedWorkspaceLocation(page, managerWorkspaceRoute);
  await expect(page.getByRole("link", { name: "Inspection lifecycle" })).toHaveAttribute(
    "href",
    `${managerWorkspaceRoute}/inspection`,
  );
  await expect(page.getByRole("button", { name: "Mark ready unavailable" })).toBeDisabled();
  await logout(page);
});

test("CAA Admin sees history and reset controls without changing the generation", async ({ page }) => {
  test.setTimeout(120_000);
  await loginQualificationAccount(
    page,
    requiredEnvironment("AVIA_AGA_OIDC_ADMIN_USERNAME"),
    adminWorkspaceRoute,
  );
  await expect(page.getByRole("heading", { name: "AGA classification review" })).toBeVisible();
  assertFixedWorkspaceLocation(page, adminWorkspaceRoute);
  await expect(page.getByRole("button", { name: "View generation history" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Admin generation history and reset" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Reset generation" })).toBeDisabled();
  await logout(page);
});
