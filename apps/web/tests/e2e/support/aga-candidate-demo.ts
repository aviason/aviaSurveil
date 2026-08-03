import { expect, type Page } from "@playwright/test";
import { createHash } from "node:crypto";
import { appendFileSync, writeFileSync } from "node:fs";

export const agaRoute = "/api/v1/admin/governed-checklist/aga-candidate-demo";

export function requiredEnvironment(name: string): string {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`${name} is required`);
  return value;
}

type QualificationPhase =
  | "login-gate-visible"
  | "provider-page-open"
  | "provider-username-filled"
  | "provider-password-filled"
  | "provider-submit-complete"
  | "oidc-provider-error"
  | "oidc-provider-retained"
  | "oidc-callback-invalid-state"
  | "oidc-callback-invalid-token"
  | "oidc-callback-stale-authority"
  | "oidc-callback-session-failure"
  | "oidc-callback-other-client-error"
  | "oidc-callback-server-error"
  | "oidc-callback-success-retained"
  | "oidc-web-return-mismatch"
  | "oidc-unexpected-location"
  | "oidc-callback-complete"
  | "oidc-cookie-pair-present"
  | "oidc-cookie-pair-incomplete"
  | "anonymous-neutral-verified"
  | "denied-login-complete"
  | "denied-session-cookie-sent"
  | "denied-session-cookie-not-sent"
  | "denied-session-active-before"
  | "denied-session-lost-before"
  | "denied-capability-requested"
  | "denied-capability-received"
  | "denied-neutral-verified"
  | "denied-session-active-after"
  | "denied-session-lost-after"
  | "denied-logout-complete"
  | "denied-stale-session-verified"
  | "mutation-login-complete"
  | "mutation-methods-verified"
  | "mutation-routes-verified"
  | "mutation-logout-complete"
  | "admin-login-complete"
  | "logout-csrf-cookie-missing"
  | "logout-csrf-rejected"
  | "logout-session-missing"
  | "logout-server-failure"
  | "logout-unexpected-status"
  | "admin-capability-requested"
  | "admin-capability-verified"
  | "admin-route-open"
  | "admin-panel-visible"
  | "admin-summary-visible"
  | "admin-forms-read"
  | "admin-question-slices-read"
  | "admin-browser-storage-clear"
  | "admin-viewports-verified"
  | "admin-telemetry-silent"
  | "admin-console-api-resource-error"
  | "admin-console-auth-resource-error"
  | "admin-console-vite-resource-error"
  | "admin-console-asset-resource-error"
  | "admin-console-other-web-resource-error"
  | "admin-console-external-resource-error"
  | "admin-console-unknown-resource-error"
  | "admin-console-react-error"
  | "admin-console-csp-error"
  | "admin-console-other-error"
  | "admin-console-clean"
  | "admin-failed-requests-clean"
  | "admin-runtime-clean"
  | "admin-logout-request-complete"
  | "admin-logout-session-revoked"
  | "admin-logout-ui-cleared"
  | "admin-logout-history-clean"
  | "admin-logout-verified";

export function recordQualificationPhase(phase: QualificationPhase): void {
  const path = process.env.AVIA_AGA_BROWSER_PHASE_FILE?.trim();
  if (!path) return;
  appendFileSync(path, `${phase}\n`, { encoding: "utf8", mode: 0o600 });
}

export async function loginQualificationAccount(
  page: Page,
  username: string,
  returnTo = "/admin/checklist-builder",
): Promise<void> {
  await page.goto(returnTo);
  await expect(page.getByRole("heading", { name: /Sign in to AviaSurveil360/i })).toBeVisible();
  recordQualificationPhase("login-gate-visible");
  await page.getByRole("button", { name: "Sign in with organization identity" }).click();
  recordQualificationPhase("provider-page-open");
  await page.locator("#username").fill(username);
  recordQualificationPhase("provider-username-filled");
  await page.locator("#password").fill(
    requiredEnvironment("AVIA_AGA_OIDC_PASSWORD"),
  );
  recordQualificationPhase("provider-password-filled");
  let callbackStatus: number | undefined;
  let callbackHTTPResponse: import("@playwright/test").Response | undefined;
  const sessionRequestDigestReads: Array<Promise<string>> = [];
  const callbackResponse = (response: import("@playwright/test").Response) => {
    const url = new URL(response.url());
    if (url.origin === new URL(requiredEnvironment("AVIA_E2E_BASE_URL")).origin && url.pathname === "/auth/callback") {
      callbackStatus = response.status();
      callbackHTTPResponse = response;
    }
  };
  const sessionRequest = (request: import("@playwright/test").Request) => {
    const url = new URL(request.url());
    if (url.origin !== new URL(requiredEnvironment("AVIA_E2E_BASE_URL")).origin || url.pathname !== "/auth/session") {
      return;
    }
    sessionRequestDigestReads.push(request.allHeaders().then((headers) => {
      const token = /(?:^|; )__Host-avia_session=([^;]+)/u.exec(
        headers.cookie ?? "",
      )?.[1];
      return token ? createHash("sha256").update(token).digest("hex") : "";
    }));
  };
  page.on("response", callbackResponse);
  page.on("request", sessionRequest);
  const sessionProjectionResponse = page.waitForResponse(
    (response) => {
      const url = new URL(response.url());
      return url.origin === new URL(requiredEnvironment("AVIA_E2E_BASE_URL")).origin &&
        url.pathname === "/auth/session";
    },
    { timeout: 10_000 },
  ).catch(() => undefined);
  await page.locator("#kc-login").click();
  recordQualificationPhase("provider-submit-complete");
  try {
    await expect(page).toHaveURL((url) => url.pathname === returnTo);
  } catch (error) {
    const current = new URL(page.url());
    const webOrigin = new URL(requiredEnvironment("AVIA_E2E_BASE_URL")).origin;
    if (current.hostname === requiredEnvironment("AVIA_PREPROD_AGA_OIDC_HOST")) {
      const providerRejected = await page.locator(".alert-error, #input-error").count() > 0;
      if (providerRejected) {
        recordQualificationPhase("oidc-provider-error");
      } else {
        recordQualificationPhase("oidc-provider-retained");
      }
    } else if (current.origin === webOrigin && current.pathname === "/auth/callback") {
      const callbackBody = await page.locator("body").textContent() ?? "";
      if (callbackBody.includes("INVALID_OIDC_STATE")) {
        recordQualificationPhase("oidc-callback-invalid-state");
      } else if (callbackBody.includes("INVALID_OIDC_TOKEN")) {
        recordQualificationPhase("oidc-callback-invalid-token");
      } else if (callbackBody.includes("STALE_AUTHORITY")) {
        recordQualificationPhase("oidc-callback-stale-authority");
      } else if (callbackBody.includes("SESSION_CREATE_FAILED")) {
        recordQualificationPhase("oidc-callback-session-failure");
      } else if ((callbackStatus ?? 0) >= 500) {
        recordQualificationPhase("oidc-callback-server-error");
      } else if ((callbackStatus ?? 0) >= 400) {
        recordQualificationPhase("oidc-callback-other-client-error");
      } else {
        recordQualificationPhase("oidc-callback-success-retained");
      }
    } else if (current.origin === webOrigin) {
      recordQualificationPhase("oidc-web-return-mismatch");
    } else {
      recordQualificationPhase("oidc-unexpected-location");
    }
    throw error;
  } finally {
    page.off("response", callbackResponse);
  }
  await sessionProjectionResponse;
  page.off("request", sessionRequest);
  const sessionRequestDigests = await Promise.all(sessionRequestDigestReads);
  const callbackCookies = await page.context().cookies(
    requiredEnvironment("AVIA_E2E_BASE_URL"),
  );
  const sessionCookie = callbackCookies.find(
    (cookie) => cookie.name === "__Host-avia_session",
  );
  const hasCSRFCookie = callbackCookies.some(
    (cookie) => cookie.name === "__Host-avia_csrf",
  );
  if (sessionCookie && hasCSRFCookie) {
    recordQualificationPhase("oidc-cookie-pair-present");
  } else {
    recordQualificationPhase("oidc-cookie-pair-incomplete");
  }
  const digestPath = process.env.AVIA_AGA_BROWSER_SESSION_DIGEST_FILE?.trim();
  if (digestPath && callbackHTTPResponse) {
    const callbackHeaders = await callbackHTTPResponse.headersArray();
    const callbackToken = callbackHeaders
      .filter((header) => header.name.toLowerCase() === "set-cookie")
      .map((header) => /^__Host-avia_session=([^;]+)/u.exec(header.value)?.[1])
      .find((value): value is string => Boolean(value));
    writeFileSync(digestPath, `${JSON.stringify({
      callback: callbackToken
        ? createHash("sha256").update(callbackToken).digest("hex")
        : "",
      stored: sessionCookie
        ? createHash("sha256").update(sessionCookie.value).digest("hex")
        : "",
      requests: sessionRequestDigests,
    })}\n`, { encoding: "utf8", mode: 0o600 });
  }
  recordQualificationPhase("oidc-callback-complete");
}

export async function browserFetch(
  page: Page,
  path: string,
  init?: { method?: string },
): Promise<{ status: number; body: string; headers: Record<string, string> }> {
  return page.evaluate(async ({ requestPath, requestMethod }) => {
    const response = await fetch(requestPath, {
      method: requestMethod,
      credentials: "same-origin",
      cache: "no-store",
      headers: { Accept: "application/json" },
    });
    return {
      status: response.status,
      body: await response.text(),
      headers: Object.fromEntries(response.headers.entries()),
    };
  }, { requestPath: path, requestMethod: init?.method ?? "GET" });
}

export async function logout(page: Page): Promise<void> {
  const result = await page.evaluate(async () => {
    const csrf = document.cookie
      .split(";")
      .map((entry) => entry.trim())
      .find((entry) => entry.startsWith("__Host-avia_csrf="))
      ?.slice("__Host-avia_csrf=".length);
    const response = await fetch("/auth/logout", {
      method: "POST",
      credentials: "same-origin",
      headers: csrf ? { "X-CSRF-Token": decodeURIComponent(csrf) } : {},
    });
    const body = await response.text();
    if (response.status === 204) {
      window.dispatchEvent(new CustomEvent("avia:authentication-lost"));
    }
    return {
      status: response.status,
      csrfPresent: Boolean(csrf),
      problem: body.includes("CSRF_INVALID")
        ? "csrf"
        : body.includes("UNAUTHENTICATED")
          ? "session"
          : "other",
    };
  });
  if (!result.csrfPresent) {
    recordQualificationPhase("logout-csrf-cookie-missing");
    throw new Error("Logout revocation failed");
  }
  if (result.status === 204) return;
  if (result.problem === "csrf") {
    recordQualificationPhase("logout-csrf-rejected");
  } else if (result.problem === "session") {
    recordQualificationPhase("logout-session-missing");
  } else if (result.status >= 500) {
    recordQualificationPhase("logout-server-failure");
  } else {
    recordQualificationPhase("logout-unexpected-status");
  }
  throw new Error("Logout revocation failed");
}
