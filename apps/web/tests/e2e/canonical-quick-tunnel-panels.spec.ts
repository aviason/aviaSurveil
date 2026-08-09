import { expect, test, type Page } from "@playwright/test";

import {
  loginQualificationAccount,
  logout,
} from "./support/aga-candidate-demo";

type DemoRole =
  | "admin"
  | "auditee"
  | "executiveDirector"
  | "finance"
  | "gm"
  | "inspector"
  | "leadInspector"
  | "manager";

interface DemoAccount {
  username: string;
  displayName: string;
  organizationId: "CAA" | "ORG-FLY-NAMIBIA";
  role: DemoRole;
  homePath: string;
  heading: RegExp;
  navigationLabel: string;
}

const accounts: readonly DemoAccount[] = [
  {
    username: "aga-demo-admin@synthetic.invalid",
    displayName: "Administrator",
    organizationId: "CAA",
    role: "admin",
    homePath: "/admin/templates",
    heading: /Checklist Templates|Template Preview/u,
    navigationLabel: "Templates",
  },
  {
    username: "aga-demo-auditee-a@synthetic.invalid",
    displayName: "Auditee A",
    organizationId: "ORG-FLY-NAMIBIA",
    role: "auditee",
    homePath: "/auditee/service-provider-cap",
    heading: /Corrective Actions \(CAP\)/u,
    navigationLabel: "Corrective Actions",
  },
  {
    username: "aga-demo-auditee-b@synthetic.invalid",
    displayName: "Auditee B",
    organizationId: "ORG-FLY-NAMIBIA",
    role: "auditee",
    homePath: "/auditee/service-provider-cap",
    heading: /Corrective Actions \(CAP\)/u,
    navigationLabel: "Corrective Actions",
  },
  {
    username: "aga-demo-executive-director@synthetic.invalid",
    displayName: "Executive Director",
    organizationId: "CAA",
    role: "executiveDirector",
    homePath: "/executive-director/executive-dashboard",
    heading: /Executive Director Dashboard/u,
    navigationLabel: "Dashboard",
  },
  {
    username: "aga-demo-finance@synthetic.invalid",
    displayName: "Finance",
    organizationId: "CAA",
    role: "finance",
    homePath: "/finance/finance-review",
    heading: /Finance Review/u,
    navigationLabel: "Finance Review",
  },
  {
    username: "aga-demo-gm@synthetic.invalid",
    displayName: "General Manager",
    organizationId: "CAA",
    role: "gm",
    homePath: "/general-manager/gm-dashboard",
    heading: /General Manager Dashboard/u,
    navigationLabel: "General Manager",
  },
  {
    username: "aga-demo-inspector@synthetic.invalid",
    displayName: "Inspector",
    organizationId: "CAA",
    role: "inspector",
    homePath: "/inspector/inspector-assignments",
    heading: /My Assignments/u,
    navigationLabel: "My Assignments",
  },
  {
    username: "aga-demo-lead-inspector@synthetic.invalid",
    displayName: "Lead Inspector",
    organizationId: "CAA",
    role: "leadInspector",
    homePath: "/lead-inspector/lead-review",
    heading: /Assigned Audits/u,
    navigationLabel: "Assigned Audits",
  },
  {
    username: "aga-demo-manager@synthetic.invalid",
    displayName: "Department Manager",
    organizationId: "CAA",
    role: "manager",
    homePath: "/department-manager/dashboard",
    heading: /Department Manager Dashboard/u,
    navigationLabel: "Dashboard",
  },
] as const;

function requiredEnvironment(name: string): string {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`${name} is required`);
  return value;
}

function observeFatalBrowserFailures(page: Page, publicOrigin: string) {
  const pageErrors: string[] = [];
  const serverErrors: string[] = [];
  const requestFailures: string[] = [];
  page.on("pageerror", (error) => pageErrors.push(error.message));
  page.on("response", (response) => {
    const url = new URL(response.url());
    if (url.origin === publicOrigin && response.status() >= 500) {
      serverErrors.push(`${response.status()} ${url.pathname}`);
    }
  });
  page.on("requestfailed", (request) => {
    const url = new URL(request.url());
    const errorText = request.failure()?.errorText ?? "unknown";
    // This test deliberately crosses multiple SPA routes. Chromium cancels
    // in-flight reads from the page being replaced; those requests never
    // receive a product response. Connection, DNS, TLS, HTTP 5xx, page, auth,
    // and OIDC failures remain independently observed and fail the test.
    if (errorText === "net::ERR_ABORTED") return;
    if (url.origin === publicOrigin) {
      requestFailures.push(`${request.method()} ${url.pathname}: ${errorText}`);
    }
  });
  return { pageErrors, requestFailures, serverErrors };
}

async function ensureAppShellServiceWorkerControlsPage(page: Page): Promise<void> {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: /Sign in to AviaSurveil360/i })).toBeVisible();
  await page.evaluate(async () => {
    if (!("serviceWorker" in navigator)) {
      throw new Error("Service Worker is unavailable in the qualification browser");
    }
    await navigator.serviceWorker.ready;
  });
  await page.reload();
  await expect.poll(
    () => page.evaluate(() => Boolean(navigator.serviceWorker.controller)),
    { message: "the public application must be controlled by its real Service Worker" },
  ).toBe(true);
}

async function assertAuthenticatedPanel(page: Page, account: DemoAccount): Promise<void> {
  const publicOrigin = new URL(requiredEnvironment("AVIA_E2E_BASE_URL")).origin;

  await ensureAppShellServiceWorkerControlsPage(page);
  await loginQualificationAccount(page, account.username, account.homePath);
  // The authentication callback intentionally replaces the unauthenticated
  // session probe. Observe panel traffic only after that navigation boundary,
  // so its cancelled GET is not misreported as an application failure.
  const failures = observeFatalBrowserFailures(page, publicOrigin);
  await expect(page.getByTestId("application-shell")).toHaveAttribute(
    "data-active-role",
    account.role,
  );
  await expect(page.getByRole("heading", { name: account.heading, level: 1 })).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Primary role navigation" })).toContainText(
    account.navigationLabel,
  );

  const session = await page.evaluate(async () => {
    const response = await fetch("/auth/session", {
      cache: "no-store",
      credentials: "same-origin",
      headers: { Accept: "application/json" },
    });
    return { status: response.status, body: await response.json() };
  });
  expect(session.status).toBe(200);
  expect(session.body).toMatchObject({
    displayName: account.displayName,
    organizationId: account.organizationId,
    roles: [account.role],
  });
  expect(session.body.subjectId).toEqual(expect.any(String));

  const cookies = await page.context().cookies(publicOrigin);
  const sessionCookie = cookies.find((cookie) => cookie.name === "__Host-avia_session");
  const csrfCookie = cookies.find((cookie) => cookie.name === "__Host-avia_csrf");
  expect(sessionCookie).toMatchObject({
    secure: true,
    httpOnly: true,
    sameSite: "Strict",
    path: "/",
  });
  expect(csrfCookie).toMatchObject({
    secure: true,
    httpOnly: false,
    sameSite: "Strict",
    path: "/",
  });
  expect(cookies.some((cookie) => cookie.name === "avia_session")).toBe(false);
  expect(cookies.some((cookie) => cookie.name === "avia_csrf")).toBe(false);

  if (account.role === "manager") {
    await page.goto("/department-manager/checklist-management/question-review");
    await expect(page.getByRole("heading", { name: "Find → Compare → Decide", level: 1 })).toBeVisible();
    await expect(page.getByLabel("Question Review queue and Decision file")).toBeVisible();

    await page.goto("/department-manager/new-audit/step-1");
    await expect(page.getByRole("heading", { name: "New Inspection", level: 1 })).toBeVisible();
    await expect(page.getByLabel("Planning intake form")).toBeVisible();
    const scopeSelector = page.getByLabel("Organization, provider scope, and regulated target");
    await expect(scopeSelector).toBeEnabled();
    await scopeSelector.selectOption({ index: 1 });
    await expect(page.getByTestId("new-audit-wizard-page")).toBeVisible();

    await page.getByRole("button", { name: "Next", exact: true }).click();
    await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-2\?draftId=/u);
    await page.getByLabel("Purpose").fill("Privacy-safe Quick Tunnel browser qualification");
    await page.getByRole("button", { name: "Next", exact: true }).click();
    await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-3\?draftId=/u);
    await page.getByLabel("Planned Date").fill("2026-09-15");
    await page.getByLabel("Location").fill("Synthetic qualification location");
    await page.getByRole("button", { name: "Next", exact: true }).click();
    await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-4\?draftId=/u);

    const pagination = page.getByLabel("New Audit question pagination");
    await expect(pagination).toContainText("1310 matching questions");
    const firstQuestion = page.getByRole("checkbox", { name: /^Select /u }).first();
    await expect(firstQuestion).toBeEnabled();
    await firstQuestion.check();
    await page.getByRole("button", { name: "Preview exact batch" }).click();
    await expect(page.getByRole("status").filter({ hasText: "1 selected · ready to confirm" })).toBeVisible();
    await page.getByRole("button", { name: "Confirm selection" }).click();
    await expect(page.getByRole("status").filter({ hasText: "Exact question selection committed · 1 selected" })).toBeVisible();
    await page.getByRole("button", { name: "Next", exact: true }).click();
    await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-5\?draftId=/u);
    await expect(page.getByText("1 exact question versions")).toBeVisible();
  }

  expect(failures.pageErrors).toEqual([]);
  expect(failures.serverErrors).toEqual([]);
  expect(failures.requestFailures).toEqual([]);

  await logout(page);
  const loggedOutStatus = await page.evaluate(async () => (
    await fetch("/auth/session", { cache: "no-store", credentials: "same-origin" })
  ).status);
  expect(loggedOutStatus).toBe(401);
}

test.describe("canonical Quick Tunnel role panels", () => {
  test.describe.configure({ mode: "serial" });

  for (const account of accounts) {
    test(`${account.displayName} signs in through public OIDC and opens the exact role panel`, async ({ page }) => {
      test.setTimeout(120_000);
      await assertAuthenticatedPanel(page, account);
    });
  }
});
