import { expect, test, type Page } from "@playwright/test";

const apiURL = process.env.AVIA_HTTP_API_URL ?? "http://127.0.0.1:58081";
const token = process.env.AVIA_CANONICAL_TEST_TOKEN ?? "";

async function markBrowserRestartVerified(page: Page): Promise<void> {
  await page.evaluate(async () => {
    const database = await new Promise<IDBDatabase>((resolve, reject) => {
      const open = indexedDB.open("aviasurveil360-offline-foundation");
      open.onsuccess = () => resolve(open.result);
      open.onerror = () => reject(open.error);
    });
    await new Promise<void>((resolve, reject) => {
      const transaction = database.transaction("foundation", "readwrite");
      transaction.objectStore("foundation").put({
        key: "restart-canary",
        value: { bootId: "verified-prior-browser-boot", verified: false },
      });
      transaction.oncomplete = () => resolve();
      transaction.onerror = () => reject(transaction.error);
    });
    database.close();
  });
}

async function readOfflineGrantScope(page: Page): Promise<{
  grantId: string;
  deviceInstanceId: string;
  packageVersion: number;
  protocolVersion: number;
}> {
  return page.evaluate(async () => {
    const database = await new Promise<IDBDatabase>((resolve, reject) => {
      const open = indexedDB.open("aviasurveil360-offline-foundation");
      open.onsuccess = () => resolve(open.result);
      open.onerror = () => reject(open.error);
    });
    const grants = await new Promise<Array<{
      grantId: string;
      deviceInstanceId: string;
      packageVersion: number;
      protocolVersion: number;
    }>>((resolve, reject) => {
      const read = database.transaction("offlineGrants").objectStore("offlineGrants").getAll();
      read.onsuccess = () => resolve(read.result);
      read.onerror = () => reject(read.error);
    });
    database.close();
    if (grants.length !== 1 || !grants[0]) {
      throw new Error(`Expected one checked-out OfflineGrant, found ${grants.length}.`);
    }
    return grants[0];
  });
}

test.beforeEach(async ({ context, request }) => {
  const reset = await request.post(`${apiURL}/__test/reset`, {
    headers: { "x-avia-test-token": token },
  });
  expect(reset.ok()).toBe(true);
  await context.addInitScript(() => {
    Object.defineProperties(navigator.storage, {
      persisted: { configurable: true, value: async () => true },
      persist: { configurable: true, value: async () => true },
      estimate: {
        configurable: true,
        value: async () => ({ usage: 8 * 1024 * 1024, quota: 512 * 1024 * 1024 }),
      },
    });
    if (navigator.serviceWorker) {
      Object.defineProperty(navigator.serviceWorker, "ready", {
        configurable: true,
        value: Promise.resolve({}),
      });
    }
  });
});

test("offline field work survives a lost acknowledgement and foreground-syncs through the real HTTP backend", async ({
  context,
  page,
  request,
}) => {
  test.setTimeout(120_000);
  const consoleIssues: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error" || message.type() === "warning") {
      consoleIssues.push(`${message.type()}: ${message.text()}`);
    }
  });
  page.on("pageerror", (error) => consoleIssues.push(`pageerror: ${error.message}`));

  await page.goto("/inspector/audits/AUD-2026-001");
  await expect(page.getByTestId("audit-id")).toHaveText("AUD-2026-001");
  await markBrowserRestartVerified(page);
  await page.getByLabel(/official browser\/version lane/i).check();
  await page.getByLabel(/encrypted managed profile/i).check();
  await page.getByRole("button", { name: "Check out for offline use" }).click();
  await expect(page.locator('[data-readiness-code="ready"]')).toBeVisible();
  await expect(page.getByTestId("offline-package-status")).toContainText("PKG-CAB-2026-001");

  await page.getByRole("link", { name: "Run Cabin checklist" }).click();
  await expect(page.getByRole("heading", { name: "Cabin Inspection checklist" })).toBeVisible();
  await page.getByLabel("Checklist answer").selectOption("NON_COMPLIANT");
  await page.getByLabel("Inspector comment").fill("PBE record unavailable during offline field work.");
  await page.getByRole("button", { name: "Save response" }).click();
  await page.getByRole("button", { name: "Create Potential Finding" }).click();
  await expect(page.getByTestId("potential-finding-id")).toContainText("PF-LOCAL-");
  await page.getByLabel("Attachment file").setInputFiles({
    name: "pbe-serviceability.pdf",
    mimeType: "application/pdf",
    buffer: Buffer.from("%PDF-1.7\n1 0 obj\n<</Type/Catalog>>\nendobj\n%%EOF\n"),
  });
  await page.getByRole("button", { name: "Stage Inspection Attachment" }).click();
  await expect(page.getByTestId("inspection-attachment-state")).toContainText("ready");
  await context.setOffline(true);
  await page
    .getByLabel("Inspector comment")
    .fill("PBE record remains unavailable after the offline follow-up check.");
  await page.getByRole("button", { name: "Save response" }).click();
  await expect(page.getByTestId("field-sync-state")).toContainText("sync pending (3)");

  let loseFirstAcknowledgement = true;
  await page.route("**/v1/sync/operations", async (route) => {
    if (!loseFirstAcknowledgement) {
      await route.continue();
      return;
    }
    loseFirstAcknowledgement = false;
    const committed = await route.fetch();
    expect(committed.ok()).toBe(true);
    await route.abort("failed");
  });
  await context.setOffline(false);
  await expect(page.getByTestId("field-sync-state")).toContainText("NETWORK_OR_INFRASTRUCTURE");

  await page.getByRole("button", { name: "Sync now" }).click();
  await expect(page.getByTestId("potential-finding-id")).toHaveText("PF-2026-001");
  await expect(page.getByTestId("inspection-attachment-state")).toContainText("acknowledged");
  await expect(page.getByTestId("field-sync-state")).toHaveCount(0);

  const packageResponse = await request.get(`${apiURL}/v1/inspection-packages/PKG-CAB-2026-001`, {
    headers: {
      "x-avia-test-token": token,
      "x-avia-test-subject": "154ec5ac-6f97-4f55-916f-d2f142fc6211",
    },
  });
  expect(packageResponse.ok()).toBe(true);
  const inspectionPackage = await packageResponse.json();
  const question = inspectionPackage.questions.find(
    (candidate: { id: string }) => candidate.id === "CAB-EMEQ-PBE-001",
  );
  expect(question.currentResponse).toMatchObject({
    answer: "NON_COMPLIANT",
    revision: 1,
  });

  await context.setOffline(true);
  await page.getByLabel("Checklist answer").selectOption("OBSERVATION");
  await page.getByLabel("Inspector comment").fill("Foreground retry records the later offline edit.");
  await page.getByRole("button", { name: "Save response" }).click();
  await expect(page.getByTestId("field-sync-state")).toContainText("sync pending (1)");
  await context.setOffline(false);
  await expect(page.getByTestId("field-sync-state")).toHaveCount(0);
  await expect(page.getByTestId("response-status")).toContainText("Server acknowledged");
  await expect(page.getByTestId("response-status")).toContainText("OBSERVATION");

  const updatedPackageResponse = await request.get(
    `${apiURL}/v1/inspection-packages/PKG-CAB-2026-001`,
    {
      headers: {
        "x-avia-test-token": token,
        "x-avia-test-subject": "154ec5ac-6f97-4f55-916f-d2f142fc6211",
      },
    },
  );
  const updatedPackage = await updatedPackageResponse.json();
  const updatedQuestion = updatedPackage.questions.find(
    (candidate: { id: string }) => candidate.id === "CAB-EMEQ-PBE-001",
  );
  expect(updatedQuestion.currentResponse).toMatchObject({ answer: "OBSERVATION", revision: 2 });

  const localAttachment = await page.evaluate(async () => {
    const database = await new Promise<IDBDatabase>((resolve, reject) => {
      const open = indexedDB.open("aviasurveil360-offline-foundation");
      open.onsuccess = () => resolve(open.result);
      open.onerror = () => reject(open.error);
    });
    const rows = await new Promise<Array<Record<string, unknown>>>((resolve, reject) => {
      const read = database.transaction("attachmentManifests").objectStore("attachmentManifests").getAll();
      read.onsuccess = () => resolve(read.result as Array<Record<string, unknown>>);
      read.onerror = () => reject(read.error);
    });
    database.close();
    return rows[0];
  });
  expect(localAttachment).toMatchObject({
    stagingState: "acknowledged",
    syncState: "ACKNOWLEDGED",
    localBytesPresent: true,
    purgeEligibleAt: null,
  });
  const expectedNetworkErrors = consoleIssues.filter((issue) =>
    issue.includes("net::ERR_FAILED") || issue.includes("net::ERR_INTERNET_DISCONNECTED"),
  );
  expect(expectedNetworkErrors.length).toBeGreaterThan(0);
  expect(consoleIssues.filter((issue) =>
    !issue.includes("net::ERR_FAILED") && !issue.includes("net::ERR_INTERNET_DISCONNECTED"),
  )).toEqual([]);
});

test("a stale field response preserves the local draft until explicit re-entry resolves the conflict", async ({
  context,
  page,
  request,
}) => {
  test.setTimeout(120_000);
  const consoleIssues: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error" || message.type() === "warning") {
      consoleIssues.push(`${message.type()}: ${message.text()}`);
    }
  });
  page.on("pageerror", (error) => consoleIssues.push(`pageerror: ${error.message}`));

  await page.goto("/inspector/audits/AUD-2026-001");
  await markBrowserRestartVerified(page);
  await page.getByLabel(/official browser\/version lane/i).check();
  await page.getByLabel(/encrypted managed profile/i).check();
  await page.getByRole("button", { name: "Check out for offline use" }).click();
  await expect(page.locator('[data-readiness-code="ready"]')).toBeVisible();
  const offlineGrant = await readOfflineGrantScope(page);
  await page.getByRole("link", { name: "Run Cabin checklist" }).click();
  await expect(page.getByRole("heading", { name: "Cabin Inspection checklist" })).toBeVisible();

  await context.setOffline(true);
  await page.getByLabel("Checklist answer").selectOption("NON_COMPLIANT");
  await page.getByLabel("Inspector comment").fill("Local draft must survive the stale revision.");
  await page.getByRole("button", { name: "Save response" }).click();
  await expect(page.getByTestId("field-sync-state")).toContainText("sync pending (1)");

  const authoritativeMutation = await request.post(`${apiURL}/v1/sync/operations`, {
    headers: {
      "x-avia-test-token": token,
      "x-avia-test-subject": "154ec5ac-6f97-4f55-916f-d2f142fc6211",
    },
    data: {
      operation: {
        operationId: "OP-RC-AUTHORITATIVE-RESPONSE-001",
        protocolVersion: offlineGrant.protocolVersion,
        offlineGrantId: offlineGrant.grantId,
        packageId: "PKG-CAB-2026-001",
        packageVersion: offlineGrant.packageVersion,
        entityId: "RESP-CAB-EMEQ-PBE-001",
        commandType: "UPSERT_CHECKLIST_RESPONSE",
        baseRevision: null,
        deviceInstanceId: offlineGrant.deviceInstanceId,
        clientOccurredAt: "2026-07-21T09:00:00Z",
        payload: {
          auditId: "AUD-2026-001",
          questionId: "CAB-EMEQ-PBE-001",
          answer: "OBSERVATION",
          comment: "Authoritative concurrent field observation.",
        },
      },
    },
  });
  expect(authoritativeMutation.ok()).toBe(true);
  await expect(authoritativeMutation.json()).resolves.toMatchObject({
    status: "accepted",
    authoritativeRevision: 1,
  });

  await context.setOffline(false);
  const conflict = page.getByTestId("field-sync-conflict");
  await expect(conflict).toContainText("STALE_REVISION at authoritative revision 1");
  await expect(conflict).toContainText("local draft preserved");
  await expect(page.getByTestId("response-status")).toContainText("Saved locally");
  await expect(page.getByTestId("response-status")).toContainText("NON COMPLIANT");
  await expect(page.getByLabel("Inspector comment")).toHaveValue(
    "Local draft must survive the stale revision.",
  );

  await page
    .getByLabel("Inspector comment")
    .fill("Explicitly re-entered after reviewing authoritative revision 1.");
  await page.getByRole("button", { name: "Save response" }).click();
  await page.getByRole("button", { name: "Sync now" }).click();
  await expect(page.getByTestId("field-sync-conflict")).toHaveCount(0);
  await expect(page.getByTestId("field-sync-state")).toHaveCount(0);
  await expect(page.getByTestId("response-status")).toContainText("Server acknowledged");
  await expect(page.getByTestId("response-status")).toContainText("NON COMPLIANT");

  const packageResponse = await request.get(`${apiURL}/v1/inspection-packages/PKG-CAB-2026-001`, {
    headers: {
      "x-avia-test-token": token,
      "x-avia-test-subject": "154ec5ac-6f97-4f55-916f-d2f142fc6211",
    },
  });
  expect(packageResponse.ok()).toBe(true);
  const inspectionPackage = await packageResponse.json();
  const response = inspectionPackage.questions.find(
    (candidate: { id: string }) => candidate.id === "CAB-EMEQ-PBE-001",
  ).currentResponse;
  expect(response).toMatchObject({
    answer: "NON_COMPLIANT",
    comment: "Explicitly re-entered after reviewing authoritative revision 1.",
    revision: 2,
  });
  expect(consoleIssues).toEqual([]);
});
