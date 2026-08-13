import { expect, test, type Page } from "@playwright/test";

import {
  loginQualificationAccount,
  logout,
} from "./support/canonical-preprod-session";

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
  });
  await expect.poll(
    () => page.evaluate(async () => {
      const registration = await navigator.serviceWorker.getRegistration("/");
      const worker = registration?.active ?? registration?.waiting ?? registration?.installing;
      return {
        hasRegistration: Boolean(registration),
        secureContext: window.isSecureContext,
        state: worker?.state ?? null,
      };
    }),
    {
      message: "the real application Service Worker must install and activate",
      timeout: 30_000,
    },
  ).toEqual({ hasRegistration: true, secureContext: true, state: "activated" });
  await page.reload();
  await expect.poll(
    () => page.evaluate(() => Boolean(navigator.serviceWorker.controller)),
    { message: "the public application must be controlled by its real Service Worker" },
  ).toBe(true);
}

async function crawlPrimaryRolePages(page: Page, account: DemoAccount, serverErrors: readonly string[]): Promise<void> {
  const navigation = page.getByRole("navigation", { name: "Primary role navigation" });
  const paths = await navigation.locator('a[href^="/"]').evaluateAll((links) => [
    ...new Set(
      links
        .map((link) => link.getAttribute("href"))
        .filter((href): href is string => Boolean(href && href !== "/")),
    ),
  ]);
  expect(paths.length).toBeGreaterThan(0);
  for (const path of paths) {
    await page.goto(path);
    await expect(
      page.getByTestId("application-shell"),
      `${account.displayName} route ${path} did not render its application shell; current URL: ${page.url()}`,
    ).toHaveAttribute("data-active-role", account.role);
    await expect(page.getByTestId("route-loading")).toHaveCount(0, { timeout: 15_000 });
    await expect(page.getByText("Loading route...", { exact: true })).toHaveCount(0);
    await expect(page.getByRole("heading", { level: 1 }).first()).toBeVisible();
    const alertText = (await page.getByRole("alert").allTextContents())
      .map((text) => text.trim())
      .filter(Boolean);
    expect(
      alertText,
      `${account.displayName} route ${path} rendered an alert; server errors: ${serverErrors.join(", ")}`,
    ).toEqual([]);
  }
}

async function assertAuthenticatedPanel(page: Page, account: DemoAccount): Promise<void> {
  const publicOrigin = new URL(requiredEnvironment("AVIA_E2E_BASE_URL")).origin;
  const secureTransport = publicOrigin.startsWith("https://");
  const sessionCookieName = secureTransport ? "__Host-avia_session" : "avia_session";
  const csrfCookieName = secureTransport ? "__Host-avia_csrf" : "avia_csrf";
  const nonCanonicalSessionCookieName = secureTransport ? "avia_session" : "__Host-avia_session";
  const nonCanonicalCSRFCookieName = secureTransport ? "avia_csrf" : "__Host-avia_csrf";

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
  const sessionCookie = cookies.find((cookie) => cookie.name === sessionCookieName);
  const csrfCookie = cookies.find((cookie) => cookie.name === csrfCookieName);
  expect(sessionCookie).toMatchObject({
    secure: secureTransport,
    httpOnly: true,
    sameSite: "Strict",
    path: "/",
  });
  expect(csrfCookie).toMatchObject({
    secure: secureTransport,
    httpOnly: false,
    sameSite: "Strict",
    path: "/",
  });
  expect(cookies.some((cookie) => cookie.name === nonCanonicalSessionCookieName)).toBe(false);
  expect(cookies.some((cookie) => cookie.name === nonCanonicalCSRFCookieName)).toBe(false);

  const providerCredentialEntries = await page.evaluate(() => [
    ...Object.entries(localStorage),
    ...Object.entries(sessionStorage),
  ].map(([key, value]) => `${key}=${value}`));
  expect(
    providerCredentialEntries.filter((entry) => /(?:access|id|refresh)[_-]?token/i.test(entry)),
  ).toEqual([]);

  if (account.role === "admin") {
    const donorCapabilityStatus = await page.evaluate(async () => (
      await fetch("/api/v1/admin/governed-checklist/aga-candidate-demo/capability", {
        cache: "no-store",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
      })
    ).status);
    expect(donorCapabilityStatus).toBe(404);

    const shell = page.getByTestId("application-shell");
    await expect(shell).not.toHaveClass(/workspace-shell--admin-demo/u);
    await expect(page.getByText("DEMO", { exact: true })).toHaveCount(0);
    expect(await shell.evaluate((element) => ({
      paddingTop: getComputedStyle(element).paddingTop,
      ribbonHeight: getComputedStyle(element).getPropertyValue("--admin-ribbon-height").trim(),
      top: element.getBoundingClientRect().top,
    }))).toEqual({ paddingTop: "0px", ribbonHeight: "0px", top: 0 });

    await page.getByRole("link", { name: "Question Bank" }).click();
    await expect(page).toHaveURL(/\/admin\/question-bank$/u);
    await expect(page.getByRole("heading", { name: "Question Bank", level: 1 })).toBeVisible();
    await expect(page.getByTestId("route-loading")).toHaveCount(0);

    await page.getByRole("link", { name: "Regulatory Library" }).click();
    await expect(page).toHaveURL(/\/admin\/regulatory-library$/u);
    await expect(page.getByRole("heading", { name: "Regulatory Library", level: 1 })).toBeVisible();
    await expect(page.getByTestId("route-loading")).toHaveCount(0);

    await page.getByRole("link", { name: "Templates" }).click();
    await expect(page).toHaveURL(/\/admin\/template-library$/u);
    await expect(page.getByRole("heading", { name: /Checklist Templates|Template Preview/u, level: 1 })).toBeVisible();
    await expect(page.getByTestId("route-loading")).toHaveCount(0);
  }

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
    await page.getByRole("button", { name: "Preview next exact batch" }).click();
    await expect(page.getByRole("status").filter({ hasText: "1 selected · ready to confirm" })).toBeVisible();
    await page.getByRole("button", { name: "Confirm selection" }).click();
    await expect(page.getByRole("status").filter({ hasText: "Exact question selection committed · 1 selected" })).toBeVisible();
    await page.getByRole("button", { name: "Next", exact: true }).click();
    await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-5\?draftId=/u);
    await expect(page.getByText("1 exact question versions")).toBeVisible();
  }

  await crawlPrimaryRolePages(page, account, failures.serverErrors);

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

	test("logout ends provider SSO and forces credentials before switching accounts", async ({ page }) => {
		test.setTimeout(180_000);
		const admin = accounts.find((account) => account.role === "admin")!;
		const manager = accounts.find((account) => account.role === "manager")!;

		await ensureAppShellServiceWorkerControlsPage(page);
		await loginQualificationAccount(page, admin.username, admin.homePath);
		await expect(page.getByTestId("application-shell")).toHaveAttribute("data-active-role", "admin");

		await logout(page);
		await page.goto(manager.homePath);
		await expect(page.getByRole("heading", { name: /Sign in to AviaSurveil360/i })).toBeVisible();
		await page.getByRole("button", { name: "Sign in with organization identity" }).click();
		await expect(page.getByLabel(/username or email/i)).toBeVisible();
		await expect(page.getByLabel(/^password$/i)).toBeVisible();

		await page.getByLabel(/username or email/i).fill(manager.username);
		await page.getByLabel(/^password$/i).fill(requiredEnvironment("AVIA_AGA_OIDC_PASSWORD"));
		await page.getByRole("button", { name: "Continue" }).click();
		await expect(page).toHaveURL((url) => url.pathname === manager.homePath);
		await expect(page.getByTestId("application-shell")).toHaveAttribute("data-active-role", "manager");
		await expect(page.getByRole("heading", { name: manager.heading, level: 1 })).toBeVisible();
	});
});
