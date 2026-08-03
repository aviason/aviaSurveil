import { expect, test } from "@playwright/test";

import {
  agaRoute,
  browserFetch,
  loginQualificationAccount,
  logout,
  recordQualificationPhase,
  requiredEnvironment,
} from "./support/aga-candidate-demo";

const labels = ["candidate-only", "release pending", "production-ready: not established"];

async function collectPages<T>(
  page: import("@playwright/test").Page,
  path: string,
): Promise<T[]> {
  const items: T[] = [];
  let cursor = "";
  for (;;) {
    const separator = path.includes("?") ? "&" : "?";
    const response = await browserFetch(
      page,
      `${path}${separator}limit=100${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
    );
    expect(response.status).toBe(200);
    expect(response.headers["cache-control"]).toBe("private, no-store");
    expect(response.headers.pragma).toBe("no-cache");
    expect(response.headers.vary).toBe("Cookie");
    const body = JSON.parse(response.body) as { items: T[]; nextCursor?: string | null };
    items.push(...body.items);
    if (!body.nextCursor) return items;
    cursor = body.nextCursor;
  }
}

test("CAA Admin reads the exact sealed AGA candidate projection without persistence", async ({ page }) => {
  test.setTimeout(240_000);
  await loginQualificationAccount(page, requiredEnvironment("AVIA_AGA_OIDC_ADMIN_USERNAME"));
  recordQualificationPhase("admin-login-complete");

  const consoleErrors: string[] = [];
  const failedRequests: string[] = [];
  const telemetryRequests: string[] = [];
  const errorResponseCategories: Array<"api" | "auth" | "vite" | "asset" | "web" | "external"> = [];
  page.on("console", (message) => {
    if (message.type() === "error") consoleErrors.push(message.text());
  });
  page.on("requestfailed", (request) => failedRequests.push(request.url()));
  page.on("request", (request) => {
    if (new URL(request.url()).pathname.startsWith("/otel/v1/")) {
      telemetryRequests.push(request.method());
    }
  });
  page.on("response", (response) => {
    if (response.status() < 400) return;
    const responseURL = new URL(response.url());
    const pageURL = new URL(page.url());
    if (responseURL.origin !== pageURL.origin) {
      errorResponseCategories.push("external");
    } else if (responseURL.pathname.startsWith("/api/")) {
      errorResponseCategories.push("api");
    } else if (responseURL.pathname.startsWith("/auth/")) {
      errorResponseCategories.push("auth");
    } else if (/^\/(?:@|src\/|node_modules\/)/u.test(responseURL.pathname)) {
      errorResponseCategories.push("vite");
    } else if (responseURL.pathname.startsWith("/assets/")) {
      errorResponseCategories.push("asset");
    } else {
      errorResponseCategories.push("web");
    }
  });

  recordQualificationPhase("admin-capability-requested");
  const capabilityResponse = await browserFetch(page, `${agaRoute}/capability`);
  expect(capabilityResponse.status).toBe(200);
  expect(JSON.parse(capabilityResponse.body)).toEqual({ available: true, labels });
  recordQualificationPhase("admin-capability-verified");
  recordQualificationPhase("admin-route-open");
  const panel = page.getByTestId("aga-candidate-demo-panel");
  await expect(panel).toBeVisible();
  recordQualificationPhase("admin-panel-visible");
  for (const label of labels) await expect(panel.getByText(label, { exact: true })).toBeVisible();
  await expect(panel.locator("dt", { hasText: "Forms" }).locator("+ dd")).toHaveText("52");
  await expect(panel.locator("dt", { hasText: "Candidate questions" }).locator("+ dd")).toHaveText("1310");
  await expect(panel.getByRole("list", { name: "Future source evidence requirements" }).getByRole("listitem")).toHaveCount(6);
  await expect(panel.getByRole("heading", { name: "Extraction review" })).toBeVisible();
  await expect(panel.getByRole("heading", { name: "Source-gap review" })).toBeVisible();
  await expect(panel.getByRole("heading", { name: "Risk-review blockers" })).toBeVisible();
  recordQualificationPhase("admin-summary-visible");

  const forms = await collectPages<{ code: string; title: string; questionCount: number; questionExtractionState: string }>(page, `${agaRoute}/forms`);
  expect(forms).toHaveLength(52);
  expect(forms.every((form) => form.title.length > 0 && Number.isInteger(form.questionCount))).toBe(true);
  expect(forms.filter((form) => form.questionExtractionState === "NO_PROTOCOL_QUESTION_BOUNDARY_DETECTED")).toHaveLength(21);
  recordQualificationPhase("admin-forms-read");
  const proposed = await collectPages<{ proposalId: string; text: string }>(
    page,
    `${agaRoute}/questions?sourceGapCategory=PROPOSAL_PRESENT_REVIEW_REQUIRED`,
  );
  const unmapped = await collectPages<{ proposalId: string; text: string }>(
    page,
    `${agaRoute}/questions?sourceGapCategory=UNMAPPED_NO_QUESTION_LEVEL_SOURCE_PROPOSAL`,
  );
  const blockers = await collectPages<{ proposalId: string; text: string }>(
    page,
    `${agaRoute}/questions?riskBand=PROPOSED_REVIEW_REQUIRED`,
  );
  expect(proposed).toHaveLength(1261);
  expect(unmapped).toHaveLength(49);
  expect(blockers).toHaveLength(14);
  recordQualificationPhase("admin-question-slices-read");

  const summaryResponse = await browserFetch(page, `${agaRoute}/summary`);
  expect(summaryResponse.status).toBe(200);
  const summary = JSON.parse(summaryResponse.body) as { packageDigest: string };
  const canaries = [summary.packageDigest, proposed[0]!.proposalId, proposed[0]!.text];
  const persisted = await page.evaluate(async () => {
    const values: string[] = [
      ...Object.values(localStorage),
      ...Object.values(sessionStorage),
    ];
    if ("databases" in indexedDB) {
      for (const info of await indexedDB.databases()) {
        if (!info.name) continue;
        const database = await new Promise<IDBDatabase>((resolve, reject) => {
          const request = indexedDB.open(info.name!);
          request.onsuccess = () => resolve(request.result);
          request.onerror = () => reject(request.error);
        });
        for (const storeName of database.objectStoreNames) {
          const records = await new Promise<unknown[]>((resolve, reject) => {
            const transaction = database.transaction(storeName, "readonly");
            const request = transaction.objectStore(storeName).getAll();
            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
          });
          values.push(JSON.stringify(records));
        }
        database.close();
      }
    }
    for (const cacheName of await caches.keys()) {
      const cache = await caches.open(cacheName);
      for (const request of await cache.keys()) {
        values.push(request.url);
        values.push(await (await cache.match(request))!.text());
      }
    }
    return values.join("\n");
  });
  for (const canary of canaries) expect(persisted).not.toContain(canary);
  recordQualificationPhase("admin-browser-storage-clear");

  for (const viewport of [
    { width: 1440, height: 900 },
    { width: 1024, height: 768 },
    { width: 390, height: 844 },
  ]) {
    await page.setViewportSize(viewport);
    await expect(panel).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    await page.keyboard.press("Tab");
    expect(await page.evaluate(() => document.activeElement !== document.body)).toBe(true);
  }
  recordQualificationPhase("admin-viewports-verified");

  expect(telemetryRequests).toEqual([]);
  recordQualificationPhase("admin-telemetry-silent");
  if (consoleErrors.length > 0) {
    const consoleText = consoleErrors.join("\n");
    if (consoleErrors.every((message) => message.startsWith("Failed to load resource:"))) {
      if (errorResponseCategories.includes("api")) {
        recordQualificationPhase("admin-console-api-resource-error");
      } else if (errorResponseCategories.includes("auth")) {
        recordQualificationPhase("admin-console-auth-resource-error");
      } else if (errorResponseCategories.includes("vite")) {
        recordQualificationPhase("admin-console-vite-resource-error");
      } else if (errorResponseCategories.includes("asset")) {
        recordQualificationPhase("admin-console-asset-resource-error");
      } else if (errorResponseCategories.includes("web")) {
        recordQualificationPhase("admin-console-other-web-resource-error");
      } else if (errorResponseCategories.includes("external")) {
        recordQualificationPhase("admin-console-external-resource-error");
      } else {
        recordQualificationPhase("admin-console-unknown-resource-error");
      }
    } else if (/Each child|Encountered two children|validateDOMNesting|cannot be a descendant/u.test(consoleText)) {
      recordQualificationPhase("admin-console-react-error");
    } else if (/Content Security Policy|violates the following directive|Refused to/u.test(consoleText)) {
      recordQualificationPhase("admin-console-csp-error");
    } else {
      recordQualificationPhase("admin-console-other-error");
    }
  }
  expect(consoleErrors).toEqual([]);
  recordQualificationPhase("admin-console-clean");
  expect(failedRequests).toEqual([]);
  recordQualificationPhase("admin-failed-requests-clean");
  recordQualificationPhase("admin-runtime-clean");
  await logout(page);
  recordQualificationPhase("admin-logout-request-complete");
  const staleCapability = await browserFetch(page, `${agaRoute}/capability`);
  expect(staleCapability.status).toBe(404);
  expect(staleCapability.body).toBe('{"error":"not found"}');
  recordQualificationPhase("admin-logout-session-revoked");
  await expect(page.getByTestId("aga-candidate-demo-panel")).toHaveCount(0);
  await expect(page.getByRole("heading", { name: /Sign in to AviaSurveil360/i })).toBeVisible();
  recordQualificationPhase("admin-logout-ui-cleared");
  await page.goto("/");
  await page.goBack();
  await expect(page.getByTestId("aga-candidate-demo-panel")).toHaveCount(0);
  await expect(page.getByRole("heading", { name: /Sign in to AviaSurveil360/i })).toBeVisible();
  recordQualificationPhase("admin-logout-history-clean");
  recordQualificationPhase("admin-logout-verified");
});
