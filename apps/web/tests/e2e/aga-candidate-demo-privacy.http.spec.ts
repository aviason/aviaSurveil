import { expect, test } from "@playwright/test";

import {
  agaRoute,
  browserFetch,
  loginQualificationAccount,
  logout,
  recordQualificationPhase,
  requiredEnvironment,
} from "./support/aga-candidate-demo";

const deniedAccounts = [
  "INSPECTOR",
  "LEAD_INSPECTOR",
  "MANAGER",
  "FINANCE",
  "GM",
  "EXECUTIVE_DIRECTOR",
  "AUDITEE",
] as const;

const neutralBody = '{"error":"not found"}';
const neutralLength = String(neutralBody.length);

test("anonymous receives the neutral no-store response", async ({ page }) => {
  await page.goto(requiredEnvironment("AVIA_E2E_BASE_URL"));
  const anonymous = await browserFetch(page, `${agaRoute}/capability`);
  expect(anonymous.status).toBe(404);
  expect(anonymous.body).toBe(neutralBody);
  expect(anonymous.headers["cache-control"]).toBe("private, no-store");
  expect(anonymous.headers.pragma).toBe("no-cache");
  expect(anonymous.headers.vary).toBe("Cookie");
  expect(anonymous.headers["content-length"]).toBe(neutralLength);
  expect(anonymous.body).not.toMatch(/candidate-only|release pending|production-ready/u);
  recordQualificationPhase("anonymous-neutral-verified");
});

for (const accountName of deniedAccounts) {
  test(`${accountName} receives the same neutral response and loses access after logout`, async ({ page }) => {
    test.setTimeout(120_000);
    await loginQualificationAccount(
      page,
      requiredEnvironment(`AVIA_AGA_OIDC_${accountName}_USERNAME`),
    );
    recordQualificationPhase("denied-login-complete");
    const sessionCookie = (await page.context().cookies(
      requiredEnvironment("AVIA_E2E_BASE_URL"),
    )).find((cookie) => cookie.name === "__Host-avia_session" || cookie.name === "avia_session");
    const sessionRequestPromise = page.waitForRequest(
      (request) => new URL(request.url()).pathname === "/auth/session",
    );
    const sessionBeforePromise = browserFetch(page, "/auth/session");
    const sessionRequest = await sessionRequestPromise;
    const sessionRequestHeaders = await sessionRequest.allHeaders();
    const sessionBefore = await sessionBeforePromise;
    const sessionCookieSent = Boolean(
      sessionCookie && sessionRequestHeaders.cookie?.split("; ").includes(
        `${sessionCookie.name}=${sessionCookie.value}`,
      ),
    );
    if (sessionCookieSent) {
      recordQualificationPhase("denied-session-cookie-sent");
    } else {
      recordQualificationPhase("denied-session-cookie-not-sent");
    }
    recordQualificationPhase(
      sessionBefore.status === 200
        ? "denied-session-active-before"
        : "denied-session-lost-before",
    );
    expect(sessionBefore.status).toBe(200);
    recordQualificationPhase("denied-capability-requested");
    const denied = await browserFetch(page, `${agaRoute}/forms/NOT-A-REAL-FORM?limit=invalid`);
    recordQualificationPhase("denied-capability-received");
    expect(denied.status).toBe(404);
    expect(denied.headers["cache-control"]).toBe("private, no-store");
    expect(denied.headers.pragma).toBe("no-cache");
    expect(denied.headers.vary).toBe("Cookie");
    expect(denied.headers["content-length"]).toBe(neutralLength);
    expect(denied.body).toBe(neutralBody);
    expect(denied.body).not.toMatch(/candidate-only|release pending|production-ready|AGA|form/u);
    recordQualificationPhase("denied-neutral-verified");
    const sessionAfter = await browserFetch(page, "/auth/session");
    recordQualificationPhase(
      sessionAfter.status === 200
        ? "denied-session-active-after"
        : "denied-session-lost-after",
    );
    expect(sessionAfter.status).toBe(200);
    await logout(page);
    recordQualificationPhase("denied-logout-complete");
    const stale = await browserFetch(page, `${agaRoute}/capability`);
    expect(stale.status).toBe(404);
    expect(stale.headers["content-length"]).toBe(neutralLength);
    expect(stale.body).toBe(neutralBody);
    recordQualificationPhase("denied-stale-session-verified");
  });
}

test("the tagged API has no candidate mutation or export route", async ({ page }) => {
  await loginQualificationAccount(page, requiredEnvironment("AVIA_AGA_OIDC_ADMIN_USERNAME"));
  recordQualificationPhase("mutation-login-complete");
  for (const method of ["POST", "PUT", "PATCH", "DELETE"]) {
    const response = await browserFetch(page, `${agaRoute}/summary`, { method });
    expect([404, 405]).toContain(response.status);
    expect(response.headers["cache-control"]).toBe("private, no-store");
  }
  recordQualificationPhase("mutation-methods-verified");
  for (const suffix of ["/export", "/publish", "/approve", "/attest", "/assign"]) {
    const response = await browserFetch(page, `${agaRoute}${suffix}`);
    expect(response.status).toBe(404);
    expect(response.body).not.toMatch(/candidate-only|release pending|production-ready/u);
  }
  recordQualificationPhase("mutation-routes-verified");
  await logout(page);
  recordQualificationPhase("mutation-logout-complete");
});
