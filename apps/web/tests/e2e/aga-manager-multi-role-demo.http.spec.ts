import { expect, test, type Locator, type Page, type Response } from "@playwright/test";
import { createHash } from "node:crypto";

import {
  loginQualificationAccount,
  logout,
  requiredEnvironment,
} from "./support/aga-candidate-demo";

const managerPackageRoute = "/department-manager/aga-demo-workspace/inspection-package";
const inspectorRoute = "/inspector/aga-demo-workspace";
const leadRoute = "/lead-inspector/aga-demo-workspace";
const reviewerRoute = "/caa-reviewer/aga-demo-workspace";
const auditeeRoute = "/auditee/aga-demo-workspace";
const workspaceQueryPath = "/v1/preprod/aga-demo-workspace/classification/query";

type Candidate = {
  identity: { formCode: string; textDigest: string; proposalId: string; ordinal: number };
  text: string;
  projection: Record<string, unknown>;
  governance: Record<string, unknown>;
  draftDisposition?: string;
};

function formCode(number: number): string {
  return `FSS-AGA-FORM-${String(number).padStart(3, "0")}`;
}

const candidateFormCodes = [
  ...Array.from({ length: 34 }, (_, index) => formCode(index + 1)),
  "FSS-AGA-FORM-035A",
  ...Array.from({ length: 13 }, (_, index) => formCode(index + 36)),
  ...Array.from({ length: 4 }, (_, index) => formCode(index + 50)),
];

function digest(value: string): string {
  return `sha256:${createHash("sha256").update(value).digest("hex")}`;
}

function equalQualifiers(left: unknown, right: unknown): boolean {
  const normalize = (value: unknown) => (Array.isArray(value) ? value.map((entry) => `${(entry as { key?: string }).key ?? ""}\u0000${(entry as { value?: string }).value ?? ""}`).sort() : []);
  return JSON.stringify(normalize(left)) === JSON.stringify(normalize(right));
}

function isEligible(candidate: Candidate, setup: Record<string, unknown>): boolean {
  const projection = candidate.projection;
  const profileCodes = Array.isArray(projection.inspectionProfileCodes) ? projection.inspectionProfileCodes : [];
  const typeCodes = Array.isArray(projection.inspectionTypeCodes) ? projection.inspectionTypeCodes : [];
  const disposition = projection.applicabilityDisposition;
  return projection.canonicalTargetKind === setup.canonicalTargetKind &&
    projection.targetProfileCode === setup.targetProfileCode &&
    profileCodes.includes(setup.inspectionProfileCode) &&
    typeCodes.includes(setup.inspectionTypeCode) &&
    equalQualifiers(projection.operationQualifiers, setup.operationQualifiers) &&
    equalQualifiers(projection.activityQualifiers, setup.activityQualifiers) &&
    ["APPLICABLE", "CONDITIONAL_ON_CONFIGURATION", "CONDITIONAL_ON_FACILITY", "CONDITIONAL_ON_OPERATION"].includes(String(disposition));
}

async function waitForWorkspaceQuery(page: Page): Promise<Response> {
  return page.waitForResponse((response) => {
    if (!response.url().includes(workspaceQueryPath) || !response.ok()) return false;
    try {
      return response.request().postDataJSON()?.operationId === "SEARCH_ITEMS";
    } catch {
      return false;
    }
  });
}

async function waitForWorkspaceCommand(page: Page, operationId: string): Promise<Response> {
  return page.waitForResponse((response) => {
    if (!response.url().includes("/v1/preprod/aga-demo-workspace/classification/commands")) return false;
    try {
      return response.request().postDataJSON()?.operationId === operationId;
    } catch {
      return false;
    }
  });
}

async function clickWorkspaceCommandWithRateLimit(page: Page, operationId: string, button: Locator): Promise<Response> {
  for (let attempt = 0; attempt < 2; attempt += 1) {
    const responseWait = waitForWorkspaceCommand(page, operationId);
    await button.click();
    const response = await responseWait;
    if (response.status() !== 429 || attempt === 1) return response;
    const retryAfterSeconds = Number(response.headers()["retry-after"] ?? "60");
    const waitMilliseconds = Math.max(1_000, (Number.isFinite(retryAfterSeconds) ? retryAfterSeconds : 60) * 1_000 + 250);
    await page.waitForTimeout(waitMilliseconds);
  }
  throw new Error(`workspace command ${operationId} did not produce a response`);
}

async function runVisibleBatch(
  page: Page,
  input: { formCode?: string; disposition: "UNSET" | "INCLUDE" | "EXCLUDE"; action: "INCLUDE" | "EXCLUDE"; search?: string; reason?: "MANAGER_SCOPE_DECISION" | "SIMULATION_SOURCE_GAP_OVERRIDE" },
): Promise<number> {
  await page.getByLabel("Batch search").fill(input.search ?? "");
  await page.getByLabel("Batch form filter").fill(input.formCode ?? "");
  await page.getByLabel("Batch disposition filter").selectOption(input.disposition);
  await page.getByLabel("Batch action").selectOption(input.action);
  await page.getByLabel("Batch reason").selectOption(input.reason ?? "MANAGER_SCOPE_DECISION");
  const previewResult = await clickWorkspaceCommandWithRateLimit(page, "PREVIEW_BATCH", page.getByRole("button", { name: "Create server preview" }));
  if (!previewResult.ok()) {
    throw new Error(`PREVIEW_BATCH failed: status=${previewResult.status()} body=${await previewResult.text()}`);
  }
  const previewBody = await previewResult.json() as { batchPreview?: { count?: number; previewDigest?: string } };
  const serverPreview = previewBody.batchPreview;
  const count = serverPreview?.count ?? -1;
  expect(serverPreview?.previewDigest).toMatch(/^sha256:[a-f0-9]{64}$/u);
  await expect(page.getByText(serverPreview?.previewDigest ?? "", { exact: true })).toBeVisible({ timeout: 30_000 });
  expect(count).toBeLessThanOrEqual(500);
  if (count > 0) {
    const executeResult = await clickWorkspaceCommandWithRateLimit(page, "EXECUTE_BATCH", page.getByRole("button", { name: "Confirm simulation disposition" }));
    if (!executeResult.ok()) {
      throw new Error(`EXECUTE_BATCH failed: status=${executeResult.status()} body=${await executeResult.text()}`);
    }
    await expect(page.getByRole("status")).toContainText("Confirmed simulation disposition", { timeout: 30_000 });
  }
  return count;
}

async function assertNoBrowserPersistence(page: Page): Promise<void> {
  const retained = await page.evaluate(async () => ({
    localStorage: Object.keys(localStorage),
    sessionStorage: Object.keys(sessionStorage),
    indexedDb: "databases" in indexedDB ? (await indexedDB.databases()).map((entry) => entry.name).filter(Boolean) : [],
    cacheStorage: "caches" in window ? await caches.keys() : [],
  }));
  expect(retained.localStorage).toEqual([]);
  expect(retained.sessionStorage).toEqual([]);
  expect(retained.indexedDb).toEqual([]);
  expect(retained.cacheStorage.every((name) => name === "aviasurveil360-app-shell-v1")).toBe(true);
}

test("Department Manager releases a bounded AGA package through the complete multi-role lifecycle", async ({ page }) => {
  test.setTimeout(30 * 60_000);
  const candidates = new Map<string, Candidate>();
  let setup: Record<string, unknown> | null = null;
  const responseTasks: Promise<void>[] = [];
  page.on("response", (response) => {
    if (!response.url().includes(workspaceQueryPath) || !response.ok()) return;
    responseTasks.push((async () => {
      let body: Record<string, unknown>;
      try {
        body = await response.json() as Record<string, unknown>;
      } catch {
        return;
      }
      if (body.operation === "GET_SIMULATION_SETUP" && body.simulationSetup && typeof body.simulationSetup === "object") {
        setup = body.simulationSetup as Record<string, unknown>;
      }
      if (body.operation !== "SEARCH_ITEMS") return;
      const rows = Array.isArray(body.items) ? body.items : [];
      expect(rows.length).toBeLessThanOrEqual(25);
      for (const value of rows) {
        if (!value || typeof value !== "object") continue;
        const row = value as Record<string, unknown>;
        const identity = row.identity as Candidate["identity"];
        const text = row.questionText;
        const textDigest = row.questionTextDigest;
        if (typeof text !== "string" || typeof textDigest !== "string" || !identity || typeof identity.textDigest !== "string") continue;
        expect(digest(text)).toBe(textDigest);
        expect(textDigest).toBe(identity.textDigest);
        const candidate: Candidate = {
          identity,
          text,
          projection: (row.projection ?? {}) as Record<string, unknown>,
          governance: (row.governance ?? {}) as Record<string, unknown>,
          draftDisposition: typeof row.draftDisposition === "string" ? row.draftDisposition : undefined,
        };
        candidates.set(`${identity.formCode}\u0000${identity.proposalId}\u0000${identity.ordinal}\u0000${identity.textDigest}`, candidate);
      }
    })());
  });

  await loginQualificationAccount(page, requiredEnvironment("AVIA_AGA_OIDC_MANAGER_USERNAME"), managerPackageRoute);
  await expect(page.getByRole("heading", { name: "AGA inspection package builder" })).toBeVisible();
  await expect(page.getByText(/1,310 candidate AGA questions; bounded pages only/i)).toBeVisible();

  const inventory = page.locator("section[aria-label='Bounded candidate inventory']");
  const nextPage = inventory.getByRole("button", { name: "Next page" });
  for (let pageNumber = 0; pageNumber < 53; pageNumber += 1) {
    await expect(inventory.locator("tbody tr").first()).toBeVisible({ timeout: 30_000 });
    if (pageNumber < 52) {
      const response = waitForWorkspaceQuery(page);
      await nextPage.click();
      await response;
      await expect(inventory.locator("tbody tr").first()).toBeVisible({ timeout: 30_000 });
    }
  }
  await expect(nextPage).toBeDisabled();
  await page.getByLabel("Package inventory search").fill("FSS-AGA-FORM-053");
  await expect(inventory.locator("tbody tr").first()).toBeVisible({ timeout: 30_000 });
  const blankSearchResponse = waitForWorkspaceQuery(page);
  await page.getByLabel("Package inventory search").fill("");
  await blankSearchResponse;
  await Promise.all(responseTasks);
  expect(candidates.size).toBe(1310);
  expect(setup).not.toBeNull();
  const eligible = [...candidates.values()].filter((candidate) => isEligible(candidate, setup!));
  expect(eligible.length).toBeGreaterThanOrEqual(1);
  const selected = eligible.find((candidate) => candidate.draftDisposition !== "INCLUDE");
  expect(selected).toBeDefined();
  const selectedCandidate = selected!;
  const bodySearch = selectedCandidate.text.trim().split(/\s+/u).slice(0, 8).join(" ");
  expect(bodySearch.length).toBeGreaterThan(8);

  await page.getByRole("button", { name: "2 · Package preview" }).click();
  let excluded = 0;
  for (const code of candidateFormCodes) {
    excluded += await runVisibleBatch(page, { formCode: code, disposition: "UNSET", action: "EXCLUDE" });
  }
  let replacedInitialIncludes = 0;
  for (const code of candidateFormCodes) {
    replacedInitialIncludes += await runVisibleBatch(page, { formCode: code, disposition: "INCLUDE", action: "EXCLUDE" });
  }
  expect(excluded + replacedInitialIncludes).toBe(1310);
  const included = await runVisibleBatch(page, {
    formCode: selectedCandidate.identity.formCode,
    disposition: "EXCLUDE",
    action: "INCLUDE",
    search: bodySearch,
    reason: Boolean(selectedCandidate.governance.questionSourceProposalGap) ? "SIMULATION_SOURCE_GAP_OVERRIDE" : "MANAGER_SCOPE_DECISION",
  });
  expect(included).toBe(1);

  await page.getByRole("button", { name: "3 · Simulation release" }).click();
  const ready = page.getByRole("button", { name: "Mark ready for demo simulation" });
  await expect(ready).toBeEnabled({ timeout: 30_000 });
  await ready.click();
  await expect(page.getByRole("status")).toContainText("marked ready for demo simulation", { timeout: 30_000 });
  const createRecommendation = page.getByRole("button", { name: "Create synthetic recommendation" });
  await expect(createRecommendation).toBeEnabled({ timeout: 30_000 });
  await createRecommendation.click();
  await expect(page.getByRole("status").filter({ hasText: "Synthetic recommendation is current" })).toBeVisible({ timeout: 30_000 });
  await page.getByLabel("Inspector selection", { exact: true }).selectOption({ index: 1 });
  await page.getByLabel("Lead Inspector selection", { exact: true }).selectOption({ index: 1 });
  await page.getByRole("button", { name: "Release synthetic inspection" }).click();
  await expect(page.getByRole("heading", { name: "Released immutable inspection snapshot" })).toBeVisible({ timeout: 30_000 });
  await assertNoBrowserPersistence(page);
  await logout(page);

  await loginQualificationAccount(page, requiredEnvironment("AVIA_AGA_OIDC_INSPECTOR_USERNAME"), `${inspectorRoute}/inspection`);
  await expect(page.getByRole("heading", { name: "Inspection and checklist responses" })).toBeVisible({ timeout: 30_000 });
  await page.getByRole("button", { name: "Start inspection" }).click();
  const question = page.locator("article[aria-label^='Checklist question ']").first();
  await expect(question).toBeVisible({ timeout: 30_000 });
  await question.getByLabel(/NON_COMPLIANT$/u).check();
  await page.getByLabel("Comment to Auditee").fill("Synthetic non-compliant response requires corrective action.");
  await question.getByRole("button", { name: "Record response" }).click();
  await expect(page.getByRole("status")).toContainText("RECORD_RESPONSE recorded", { timeout: 30_000 });
  await question.getByRole("button", { name: "Propose Potential Finding" }).click();
  await expect(page.getByRole("status")).toContainText("CREATE_POTENTIAL_FINDING recorded", { timeout: 30_000 });
  await page.getByRole("button", { name: "Submit checklist" }).click();
  await expect(page.getByRole("status")).toContainText("SUBMIT_CHECKLIST recorded", { timeout: 30_000 });
  await logout(page);

  await loginQualificationAccount(page, requiredEnvironment("AVIA_AGA_OIDC_LEAD_INSPECTOR_USERNAME"), `${leadRoute}/potential-findings`);
  await expect(page.getByRole("heading", { name: "Potential Finding review" })).toBeVisible();
  await page.getByRole("button", { name: "Convert to Finding" }).click();
  await expect(page.getByRole("status")).toContainText("CONVERT_POTENTIAL_FINDING recorded", { timeout: 30_000 });
  await logout(page);

  await loginQualificationAccount(page, requiredEnvironment("AVIA_AGA_OIDC_AUDITEE_USERNAME"), `${auditeeRoute}/caps-evidence`);
  await expect(page.getByRole("heading", { name: "CAP, Evidence, and closure" })).toBeVisible();
  await page.getByLabel("CAP root cause").fill("Synthetic process-control root cause.");
  await page.getByLabel("CAP corrective action").fill("Synthetic corrective action.");
  await page.getByLabel("CAP preventive action").fill("Synthetic preventive action.");
  await page.getByLabel("CAP responsible person").fill("Synthetic accountable provider role");
  await page.getByLabel("Lifecycle Comment to Auditee").fill("Synthetic CAP submitted for CAA review.");
  await page.getByRole("button", { name: "Submit CAP revision" }).click();
  await expect(page.getByRole("status")).toContainText("SUBMIT_CAP_REVISION recorded", { timeout: 30_000 });
  await logout(page);

  await loginQualificationAccount(page, requiredEnvironment("AVIA_AGA_OIDC_LEAD_INSPECTOR_USERNAME"), `${reviewerRoute}/caps-evidence`);
  await expect(page.getByRole("heading", { name: "CAP, Evidence, and closure" })).toBeVisible();
  await page.getByLabel("Lifecycle Comment to Auditee").fill("CAP accepted after CAA Reviewer review.");
  await page.getByLabel("Internal CAA Note").fill("Internal CAA review note remains private.");
  await page.getByRole("button", { name: "Review CAP" }).click();
  await expect(page.getByRole("status")).toContainText("REVIEW_CAP recorded", { timeout: 30_000 });
  await expect(page.locator("section[aria-label='Finding selection']")).toContainText("EVIDENCE_REQUIRED");
  await expect(page.locator("section[aria-label='Finding selection']")).not.toContainText("CLOSED");
  await logout(page);

  await loginQualificationAccount(page, requiredEnvironment("AVIA_AGA_OIDC_AUDITEE_USERNAME"), `${auditeeRoute}/caps-evidence`);
  await page.getByLabel("Evidence filename").fill("synthetic-evidence.pdf");
  await page.getByLabel("Lifecycle Comment to Auditee").fill("Synthetic Evidence submitted for verification.");
  await page.getByRole("button", { name: "Submit Evidence version" }).click();
  await expect(page.getByRole("status")).toContainText("SUBMIT_EVIDENCE_VERSION recorded", { timeout: 30_000 });
  await logout(page);

  await loginQualificationAccount(page, requiredEnvironment("AVIA_AGA_OIDC_LEAD_INSPECTOR_USERNAME"), `${leadRoute}/caps-evidence`);
  await page.getByLabel("Lifecycle Comment to Auditee").fill("Evidence verified and finding closed.");
  await page.getByLabel("Internal CAA Note").fill("Internal verification note remains private.");
  await page.getByRole("button", { name: "Verify Evidence" }).click();
  await expect(page.getByRole("status")).toContainText("VERIFY_EVIDENCE recorded", { timeout: 30_000 });
  await expect(page.locator("section[aria-label='Finding selection']")).toContainText("CLOSED");
  await assertNoBrowserPersistence(page);
  await logout(page);
});
