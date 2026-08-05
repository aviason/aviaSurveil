import { expect, test } from "@playwright/test";

import {
  loginQualificationAccount,
  logout,
  requiredEnvironment,
} from "./support/aga-candidate-demo";

const inspectorRoute = "/inspector/aga-demo-workspace";
const leadRoute = "/lead-inspector/aga-demo-workspace";
const auditeeRoute = "/auditee/aga-demo-workspace";

test("Inspector lifecycle controls stay bound to the authorized server projection", async ({ page }) => {
  test.setTimeout(120_000);
  await loginQualificationAccount(
    page,
    requiredEnvironment("AVIA_AGA_OIDC_INSPECTOR_USERNAME"),
    `${inspectorRoute}/inspection`,
  );
  await expect(page.getByRole("heading", { name: "Inspection and checklist responses" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Load authorized inspection" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Start inspection" })).toBeDisabled();
  await logout(page);
});

test("Lead review exposes separate Potential Finding decisions", async ({ page }) => {
  test.setTimeout(120_000);
  await loginQualificationAccount(
    page,
    requiredEnvironment("AVIA_AGA_OIDC_LEAD_INSPECTOR_USERNAME"),
    `${leadRoute}/potential-findings`,
  );
  await expect(page.getByRole("heading", { name: "Potential Finding review" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Return for correction" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Dismiss Potential Finding" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Convert to Finding" })).toBeDisabled();
  await logout(page);
});

test("Auditee CAP and Evidence view omits the Internal CAA Note field", async ({ page }) => {
  test.setTimeout(120_000);
  await loginQualificationAccount(
    page,
    requiredEnvironment("AVIA_AGA_OIDC_AUDITEE_USERNAME"),
    `${auditeeRoute}/caps-evidence`,
  );
  await expect(page.getByRole("heading", { name: "CAP, Evidence, and closure" })).toBeVisible();
  await expect(page.getByLabel("Internal CAA Note")).toHaveCount(0);
  await expect(page.getByText(/CAP acceptance and Evidence verification do not silently create/i)).toHaveCount(0);
  await logout(page);
});
