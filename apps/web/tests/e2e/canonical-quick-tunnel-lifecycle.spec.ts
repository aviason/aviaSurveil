import { expect, test, type Page } from "@playwright/test";
import { createHash, randomUUID } from "node:crypto";

import { loginQualificationAccount, logout, requiredEnvironment } from "./support/canonical-preprod-session";

const accounts = {
  auditeeA: "aga-demo-auditee-a@synthetic.invalid",
  executiveDirector: "aga-demo-executive-director@synthetic.invalid",
  finance: "aga-demo-finance@synthetic.invalid",
  gm: "aga-demo-gm@synthetic.invalid",
  inspector: "aga-demo-inspector@synthetic.invalid",
  lead: "aga-demo-lead-inspector@synthetic.invalid",
  manager: "aga-demo-manager@synthetic.invalid",
} as const;

interface SessionProjection {
  subjectId: string;
  organizationId: string;
  roles: string[];
}

interface InspectionPackageProjection {
  id: string;
  auditId: string;
  checklistRevision: number;
  checklistStatus: string;
  questions: Array<{ id: string }>;
}

interface FindingProjection {
  id: string;
  auditId: string;
  organizationId: string;
  status: string;
  revision: number;
}

interface ReportProjection {
  reportVersionId: string;
  reportId: string;
  status: string;
  revision: number;
}

interface DocumentProjection {
  id: string;
  renderStatus?: string;
  documentVersionId?: string;
  sha256?: string;
  rendererHash?: string;
  templateHash?: string;
  sourceHash?: string;
}

interface APIResult<T> {
  status: number;
  body: T;
  raw: string;
}

function operation(prefix: string): string {
  return `${prefix}-${randomUUID()}`;
}

async function apiRequest<T>(
  page: Page,
  method: "GET" | "POST" | "PUT",
  path: string,
  body?: unknown,
): Promise<APIResult<T>> {
  const result = await page.evaluate(async ({ requestMethod, requestPath, requestBody }) => {
    const command = typeof requestBody === "object" && requestBody !== null
      ? requestBody as Record<string, unknown>
      : null;
    const idempotencyKey = typeof command?.idempotencyKey === "string"
      ? command.idempotencyKey
      : "";
    const csrfEntry = document.cookie
      .split(";")
      .map((entry) => entry.trim())
      .find((entry) => entry.startsWith("__Host-avia_csrf=") || entry.startsWith("avia_csrf="));
    const csrf = csrfEntry ? decodeURIComponent(csrfEntry.slice(csrfEntry.indexOf("=") + 1)) : "";
    const response = await fetch(requestPath, {
      method: requestMethod,
      cache: "no-store",
      credentials: "same-origin",
      headers: {
        Accept: "application/json",
        ...(requestBody === undefined ? {} : { "Content-Type": "application/json" }),
        ...(idempotencyKey ? { "Idempotency-Key": idempotencyKey } : {}),
        ...(requestMethod === "GET" || !csrf ? {} : { "X-CSRF-Token": csrf }),
      },
      body: requestBody === undefined ? undefined : JSON.stringify(requestBody),
    });
    const raw = await response.text();
    let decoded: unknown = null;
    try { decoded = raw ? JSON.parse(raw) : null; } catch { decoded = raw; }
    return { status: response.status, body: decoded, raw };
  }, { requestMethod: method, requestPath: path, requestBody: body });
  return result as APIResult<T>;
}

async function requireAPI<T>(
  page: Page,
  method: "GET" | "POST" | "PUT",
  path: string,
  body?: unknown,
): Promise<T> {
  const result = await apiRequest<T>(page, method, path, body);
  if (result.status < 200 || result.status >= 300) {
    throw new Error(`${method} ${path} returned ${result.status}: ${result.raw}`);
  }
  return result.body;
}

async function signIn(page: Page, username: string, path: string): Promise<SessionProjection> {
  await loginQualificationAccount(page, username, path);
  const session = await requireAPI<SessionProjection>(page, "GET", "/auth/session");
  expect(session.subjectId).toEqual(expect.any(String));
  return session;
}

async function selectPlanningItem(page: Page, planningItemId: string): Promise<void> {
  const search = page.getByPlaceholder(/Plan ID, title, organization/u);
  await expect(search).toBeVisible();
  await search.fill(planningItemId);
  const row = page.getByRole("row").filter({ hasText: planningItemId });
  await expect(row).toBeVisible();
  const review = row.getByRole("button", { name: `Review ${planningItemId}`, exact: true });
  if (await review.isVisible() && await review.isEnabled()) {
    await review.click();
  } else {
    await expect(row.getByRole("button", { name: `${planningItemId} is already selected`, exact: true })).toBeDisabled();
  }
  await expect(page.locator("[data-planning-item-id]")).toHaveAttribute("data-planning-item-id", planningItemId);
}

async function selectFinancePlanningItem(page: Page, planningItemId: string): Promise<void> {
  const search = page.getByLabel("Search plans");
  await expect(search).toBeVisible();
  await search.fill(planningItemId);
  const row = page.getByRole("row").filter({ hasText: planningItemId });
  await expect(row).toBeVisible();
  const review = row.getByRole("button", { name: `Review ${planningItemId}`, exact: true });
  if (await review.isVisible() && await review.isEnabled()) {
    await review.click();
  } else {
    await expect(row.getByRole("button", { name: `${planningItemId} is already selected`, exact: true })).toBeDisabled();
  }
  await expect(page.locator(".finance-review-detail")).toContainText(planningItemId);
}

async function decideReport(
  page: Page,
  report: ReportProjection,
  decision: "FORWARD" | "ISSUE_AND_LOCK",
  reason: string,
): Promise<ReportProjection> {
  return requireAPI<ReportProjection>(page, "POST", `/api/v1/report-versions/${encodeURIComponent(report.reportVersionId)}/decisions`, {
    operationId: operation(`REPORT-${decision}`),
    reportVersionId: report.reportVersionId,
    expectedReportVersionRevision: report.revision,
    decision,
    reason,
  });
}

test.describe("canonical Quick Tunnel 1,310-question lifecycle", () => {
  test.describe.configure({ mode: "serial" });

  test("runs exact selection, approval, preparation, execution, CAP, Evidence, closure, and Final Report across real roles", async ({ browser, page }) => {
    test.setTimeout(1_500_000);

    const leadSession = await signIn(page, accounts.lead, "/lead-inspector/lead-review");
    expect(leadSession.roles).toEqual(["leadInspector"]);
    await logout(page);

    const inspectorSession = await signIn(page, accounts.inspector, "/inspector/inspector-assignments");
    expect(inspectorSession.roles).toEqual(["inspector"]);
    await logout(page);

    await signIn(page, accounts.manager, "/department-manager/new-audit/step-1");
    const donorWorkspace = await apiRequest(page, "POST", "/api/v1/preprod/aga-demo-workspace/classification/query", {});
    expect(donorWorkspace.status).toBe(404);
    const scopeSelector = page.getByLabel("Organization, provider scope, and regulated target");
    await expect(scopeSelector).toBeEnabled();
    await scopeSelector.selectOption({ index: 1 });
    await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-1\?draftId=/u);
    await page.getByRole("button", { name: "Next", exact: true }).click();
    await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-2\?draftId=/u);
    await page.getByLabel("Inspection Category").selectOption("Ad Hoc / Unannounced");
    await page.getByLabel("Purpose").fill("Privacy-safe connected lifecycle qualification");
    await page.getByLabel("Risk Category").fill("Synthetic connected qualification risk");
    await page.getByRole("button", { name: "Next", exact: true }).click();
    await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-3\?draftId=/u);
    await page.getByLabel("Planned Date").fill("2026-09-15");
    await page.getByLabel("Location").fill("Synthetic connected qualification location");
    await page.getByRole("button", { name: "Next", exact: true }).click();
    await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-4\?draftId=/u);
    await expect(page.getByLabel("New Audit question pagination")).toContainText("1310 matching questions", { timeout: 60_000 });
    await page.getByRole("button", { name: "Stage all matching eligible questions" }).click();
    await expect(page.getByRole("status").filter({ hasText: "1,310 eligible questions staged" })).toBeVisible({ timeout: 120_000 });
    for (const expectedCount of [500, 1_000, 1_310]) {
      await page.getByRole("button", { name: "Preview next exact batch" }).click();
      await expect(page.getByRole("status").filter({ hasText: `Preview: ${expectedCount} selected` })).toBeVisible({ timeout: 60_000 });
      await page.getByRole("button", { name: "Confirm selection" }).click();
      const committedStatus = expectedCount < 1_310
        ? `Exact batch committed · ${expectedCount.toLocaleString("en-US")} of 1,310 selected`
        : "Exact question selection committed · 1,310 selected";
      await expect(page.getByRole("status").filter({ hasText: committedStatus })).toBeVisible({ timeout: 60_000 });
    }
    await expect(page.getByText("1310 selected").first()).toBeVisible();
    await page.getByRole("button", { name: "Next", exact: true }).click();
    await expect(page).toHaveURL(/\/department-manager\/new-audit\/step-5\?draftId=/u);
    await expect(page.getByText(/1310 exact question versions/u)).toBeVisible();
    await page.getByRole("button", { name: "Submit for Finance Review" }).click();
    await expect(page).toHaveURL(/\/department-manager\/audit-plan\?planningItemId=/u, { timeout: 60_000 });
    const planningItemId = new URL(page.url()).searchParams.get("planningItemId");
    expect(planningItemId).toMatch(/^plan-intake-[a-f0-9]{24}$/u);
    await logout(page);

    await signIn(page, accounts.finance, "/finance/finance-review");
    await selectFinancePlanningItem(page, planningItemId!);
    await expect(page.getByTestId("planning-status")).toHaveText("FINANCE_REVIEW");
    await page.getByRole("button", { name: "Approve Budget" }).click();
    await page.getByLabel("Finance decision reason").fill("Synthetic budget approved for connected qualification.");
    await page.getByRole("button", { name: "Confirm Finance Decision" }).click();
    await expect(page.getByTestId("planning-status")).toHaveText("GM_REVIEW");
    await logout(page);

    await signIn(page, accounts.gm, "/general-manager/planning");
    await selectPlanningItem(page, planningItemId!);
    await page.getByLabel("General Manager decision reason").fill("Synthetic operational review completed.");
    await page.getByRole("button", { name: `Forward ${planningItemId} to Executive Director` }).click();
    await expect(page.getByTestId("planning-status")).toHaveText("EXECUTIVE_DIRECTOR_REVIEW");
    await logout(page);

    await signIn(page, accounts.executiveDirector, "/executive-director/planning");
    await selectPlanningItem(page, planningItemId!);
    await page.getByLabel("Executive Director plan decision reason").fill("Synthetic final Planning approval completed.");
    await page.getByRole("button", { name: `Approve plan ${planningItemId}` }).click();
    await expect(page.getByTestId("planning-status")).toHaveText("GM_RELEASE");
    await logout(page);

    await signIn(page, accounts.gm, "/general-manager/planning");
    await selectPlanningItem(page, planningItemId!);
    await page.getByLabel("General Manager decision reason").fill("Synthetic approved plan released to Department.");
    await page.getByRole("button", { name: `Release ${planningItemId} to Department Manager` }).click();
    await expect(page.getByTestId("planning-status")).toHaveText("RELEASED");
    await logout(page);

    await signIn(page, accounts.manager, `/department-manager/audit-plan?planningItemId=${encodeURIComponent(planningItemId!)}`);
    await expect(page.getByTestId("canonical-preparation-actions")).toBeVisible();
    await page.getByRole("button", { name: "Begin server-owned preparation" }).click();
    await expect(page.getByRole("status").filter({ hasText: "Server-owned preparation" })).toBeVisible();
    await page.getByLabel("Lead Inspector subject ID").fill(leadSession.subjectId);
    await page.getByRole("button", { name: "Assign Lead Inspector" }).click();
    const leadPreparationLink = page.getByRole("link", { name: "Open Lead preparation workspace" });
    await expect(leadPreparationLink).toBeVisible();
    const assignmentId = new URL((await leadPreparationLink.getAttribute("href"))!, requiredEnvironment("AVIA_E2E_BASE_URL")).searchParams.get("assignmentId");
    expect(assignmentId).toEqual(expect.any(String));
    await logout(page);

    await signIn(page, accounts.lead, `/lead-inspector/audit-preparation?assignmentId=${encodeURIComponent(assignmentId!)}`);
    await expect(page.getByText("1310").first()).toBeVisible();
    await page.getByLabel("Inspector subject IDs").fill(inspectorSession.subjectId);
    await page.getByRole("button", { name: "Preview exact team" }).click();
    await expect(page.getByLabel("Team assignment preview")).toBeVisible();
    await page.getByRole("button", { name: "Confirm team assignment" }).click();
    await expect(page.getByLabel("Coverage Inspector")).toBeEnabled();
    await page.getByLabel("Coverage Inspector").selectOption(inspectorSession.subjectId);
    await page.getByRole("button", { name: "Stage all released questions for Inspector" }).click();
    await expect(page.getByRole("status").filter({ hasText: "1,310 released questions staged" })).toBeVisible();
    for (const expectedCount of [500, 1_000, 1_310]) {
      await page.getByRole("button", { name: "Preview next coverage batch" }).click();
      await expect(page.getByLabel("Question coverage preview")).toBeVisible();
      await page.getByRole("button", { name: "Confirm question coverage batch" }).click();
      await expect(page.getByRole("status").filter({ hasText: `${expectedCount.toLocaleString("en-US")}` }).first()).toBeVisible({ timeout: 60_000 });
    }
    await expect(page.getByRole("status").filter({ hasText: "1,310 exact question assignments committed" })).toBeVisible();
    await logout(page);

    await signIn(page, accounts.manager, `/department-manager/audit-plan?planningItemId=${encodeURIComponent(planningItemId!)}`);
    await expect(page.getByRole("button", { name: "Confirm preparation" })).toBeEnabled();
    await page.getByRole("button", { name: "Confirm preparation" }).click();
    await expect(page.getByRole("button", { name: "Materialize canonical Audit" })).toBeEnabled();
    await page.getByRole("button", { name: "Materialize canonical Audit" }).click();
    const materializedStatus = page.getByRole("status").filter({ hasText: "Inspector start is available" });
    await expect(materializedStatus).toBeVisible({ timeout: 60_000 });
    const materializedText = await materializedStatus.textContent() ?? "";
    const materialized = /Inspection (\S+) · \S+ · package (\S+)\./u.exec(materializedText);
    expect(materialized).not.toBeNull();
    const auditId = materialized![1];
    const packageId = materialized![2];
    await logout(page);

    await signIn(page, accounts.inspector, "/inspector/inspector-assignments");
    const preStartPackage = await apiRequest(page, "GET", `/api/v1/inspection-packages/${encodeURIComponent(packageId)}`);
    expect(preStartPackage.status).toBeGreaterThanOrEqual(400);
    const assignmentRow = page.getByRole("row").filter({ hasText: auditId });
    await assignmentRow.getByRole("button", { name: "Start inspection" }).click();
    await expect(assignmentRow.getByRole("link", { name: /^Open /u })).toBeVisible();
    const detailPackageResponse = page.waitForResponse((response) => {
      const url = new URL(response.url());
      return decodeURIComponent(url.pathname) === `/api/v1/inspection-packages/${packageId}`;
    });
    await page.goto(`/inspector/audits/${encodeURIComponent(auditId)}`);
    const detailPackage = await detailPackageResponse;
    expect(
      detailPackage.status(),
      `Audit detail package request failed: ${await detailPackage.text()}`,
    ).toBe(200);
    await expect(page.getByTestId("audit-id")).toHaveText(auditId, { timeout: 30_000 });
    const runner = page.getByRole("link", { name: "Run Cabin checklist" });
    await expect(runner).toHaveAttribute("href", `/inspector/audits/${encodeURIComponent(auditId)}/checklist?packageId=${encodeURIComponent(packageId)}`);

    const inspectionPackage = await requireAPI<InspectionPackageProjection>(page, "GET", `/api/v1/inspection-packages/${encodeURIComponent(packageId)}`);
    expect(inspectionPackage.questions).toHaveLength(1_310);
    expect(inspectionPackage.checklistStatus).toBe("IN_PROGRESS");
    const responseIds = inspectionPackage.questions.map((_, index) => `e2e-response-${String(index + 1).padStart(4, "0")}-${randomUUID()}`);
    const bulkResult = await page.evaluate(async ({ auditIdentity, questions, ids }) => {
      const csrfEntry = document.cookie.split(";").map((entry) => entry.trim()).find((entry) => entry.startsWith("__Host-avia_csrf=") || entry.startsWith("avia_csrf="));
      const csrf = csrfEntry ? decodeURIComponent(csrfEntry.slice(csrfEntry.indexOf("=") + 1)) : "";
      const failures: string[] = [];
      let rateLimitRetries = 0;
      const wait = (milliseconds: number) => new Promise((resolve) => window.setTimeout(resolve, milliseconds));
      for (let start = 0; start < questions.length; start += 20) {
        await Promise.all(questions.slice(start, start + 20).map(async (question, offset) => {
          const index = start + offset;
          for (let attempt = 0; attempt < 8; attempt += 1) {
            const response = await fetch(`/api/v1/checklist-responses/${encodeURIComponent(ids[index])}`, {
              method: "PUT",
              credentials: "same-origin",
              headers: { Accept: "application/json", "Content-Type": "application/json", "X-CSRF-Token": csrf },
              body: JSON.stringify({
                operationId: `CHECKLIST-${index + 1}-${ids[index]}`,
                responseId: ids[index],
                auditId: auditIdentity,
                questionId: question.id,
                expectedResponseRevision: null,
                answer: index === 0 ? "NON_COMPLIANT" : "COMPLIANT",
                comment: index === 0 ? "Privacy-safe connected qualification exception." : "",
              }),
            });
            if (response.ok) return;
            const body = await response.text();
            if (response.status !== 429) {
              failures.push(`${index}:${response.status}:${body}`);
              return;
            }
            rateLimitRetries += 1;
            const rawRetryAfter = Number.parseInt(response.headers.get("Retry-After") ?? "1", 10);
            const retryAfterSeconds = Number.isFinite(rawRetryAfter)
              ? Math.min(65, Math.max(1, rawRetryAfter))
              : 1;
            await wait((retryAfterSeconds * 1_000) + 250);
          }
          failures.push(`${index}:429:rate limit retry budget exhausted`);
        }));
      }
      return { failures, rateLimitRetries };
    }, { auditIdentity: auditId, questions: inspectionPackage.questions, ids: responseIds });
    expect(bulkResult.failures).toEqual([]);

    const potential = await requireAPI<{ id: string; revision: number; status: string }>(page, "POST", "/api/v1/potential-findings", {
      operationId: operation("POTENTIAL-FINDING"),
      auditId,
      questionId: inspectionPackage.questions[0].id,
      checklistResponseId: responseIds[0],
      expectedChecklistResponseRevision: 1,
      title: "Privacy-safe connected qualification exception",
      description: "Synthetic execution exception used only for the disposable local lifecycle test.",
      requiredComment: "Auditee-facing synthetic qualification comment.",
      inspectionAttachmentIds: [],
    });
    expect(potential.status).toBe("PENDING_LEAD_REVIEW");
    const refreshedPackage = await requireAPI<InspectionPackageProjection>(page, "GET", `/api/v1/inspection-packages/${encodeURIComponent(packageId)}`);
    const submission = await requireAPI<{ checklistStatus: string }>(page, "POST", `/api/v1/checklists/${encodeURIComponent(auditId)}/submit`, {
      operationId: operation("SUBMIT-CHECKLIST"),
      auditId,
      expectedChecklistRevision: refreshedPackage.checklistRevision,
    });
    expect(submission.checklistStatus).toBe("SUBMITTED");
    await logout(page);

    const suffix = randomUUID();
    const preliminaryReport: ReportProjection = await signIn(page, accounts.lead, "/lead-inspector/preliminary-reports").then(async () => requireAPI(page, "POST", "/api/v1/report-versions", {
      operationId: operation("PRELIMINARY-CREATE"),
      idempotencyKey: operation("PRELIMINARY-IDEMPOTENCY"),
      expectedRevision: null,
      reportVersionId: `report-version:preliminary:${suffix}`,
      reportId: `report:preliminary:${suffix}`,
      auditId,
      kind: "PRELIMINARY",
      version: 1,
      status: "DEPARTMENT_REVIEW",
      findingIds: [],
      content: { title: "Privacy-safe Preliminary Report", summary: "Synthetic connected qualification only." },
    }));
    expect(preliminaryReport.status).toBe("DEPARTMENT_REVIEW");
    await logout(page);

    await signIn(page, accounts.manager, "/department-manager/dashboard");
    let preliminary = await decideReport(page, preliminaryReport, "FORWARD", "Department Preliminary review completed.");
    expect(preliminary.status).toBe("GM_REVIEW");
    await logout(page);
    await signIn(page, accounts.gm, "/general-manager/report-approvals");
    preliminary = await decideReport(page, preliminary, "FORWARD", "General Manager Preliminary review completed.");
    expect(preliminary.status).toBe("EXECUTIVE_DIRECTOR_REVIEW");
    await logout(page);
    await signIn(page, accounts.executiveDirector, "/executive-director/preliminary-reports");
    preliminary = await decideReport(page, preliminary, "ISSUE_AND_LOCK", "Executive Director issued the synthetic Preliminary Report.");
    expect(preliminary.status).toBe("LOCKED");
    await logout(page);

    await signIn(page, accounts.lead, "/lead-inspector/lead-review");
    const conversion = await requireAPI<{ potentialFinding: { status: string }; finding: FindingProjection }>(page, "POST", `/api/v1/potential-findings/${encodeURIComponent(potential.id)}/decisions`, {
      operationId: operation("CONVERT-POTENTIAL"),
      potentialFindingId: potential.id,
      expectedPotentialFindingRevision: potential.revision,
      decision: "CONVERT",
      severity: "LEVEL_3_MINOR",
      capRequired: true,
      evidenceRequired: true,
      dueDate: "2026-10-31",
    });
    expect(conversion.potentialFinding.status).toBe("CONVERTED");
    let finding = conversion.finding;
    expect(finding.status).toBe("WAITING_FOR_CAP");
    await logout(page);

    await signIn(page, accounts.auditeeA, "/auditee/service-provider-cap");
    const cap = await requireAPI<{ capRevisionId: string; capRevision: number; findingStatus: string; findingRevision: number }>(page, "POST", "/api/v1/caps", {
      operationId: operation("CAP-SUBMIT"),
      findingId: finding.id,
      expectedFindingRevision: finding.revision,
      rootCause: "Synthetic process-control gap.",
      correctiveAction: "Synthetic corrective control implemented.",
      preventiveAction: "Synthetic preventive review scheduled.",
      responsiblePerson: "Privacy-safe Auditee role",
      targetCompletionDate: "2026-10-15",
      commentToCaa: "Synthetic CAP for connected qualification.",
    });
    expect(cap.findingStatus).toBe("CAP_SUBMITTED");
    await logout(page);

    await signIn(page, accounts.lead, "/lead-inspector/lead-review");
    const capReview = await requireAPI<{ findingStatus: string; findingRevision: number }>(page, "POST", `/api/v1/caps/${encodeURIComponent(cap.capRevisionId)}/reviews`, {
      operationId: operation("CAP-ACCEPT"),
      capRevisionId: cap.capRevisionId,
      expectedCapRevision: cap.capRevision,
      findingId: finding.id,
      expectedFindingRevision: cap.findingRevision,
      decision: "ACCEPT",
      commentToAuditee: "CAP accepted; Evidence remains required.",
      internalCaaNote: "Synthetic internal qualification note; must not enter Auditee projections.",
    });
    expect(capReview.findingStatus).toBe("EVIDENCE_REQUIRED");
    await logout(page);

    await signIn(page, accounts.auditeeA, "/auditee/service-provider-cap");
    const evidenceBytes = Buffer.from("%PDF-1.4\n% privacy-safe connected qualification\n1 0 obj\n<<>>\nendobj\ntrailer\n<<>>\n%%EOF\n", "utf8");
    const evidenceDigest = `sha256:${createHash("sha256").update(evidenceBytes).digest("hex")}`;
    const upload = await requireAPI<{ uploadId: string; uploadUrl: string; requiredHeaders: Record<string, string> }>(page, "POST", "/api/v1/evidence/uploads", {
      operationId: operation("EVIDENCE-BEGIN"),
      findingId: finding.id,
      expectedFindingRevision: capReview.findingRevision,
      fileName: "privacy-safe-qualification.pdf",
      declaredMediaType: "application/pdf",
      byteSize: evidenceBytes.byteLength,
      sha256: evidenceDigest,
    });
    expect(new URL(upload.uploadUrl).protocol).toBe("https:");
    const putResult = await page.evaluate(async ({ url, headers, bytes }) => {
      const response = await fetch(url, { method: "PUT", headers, body: Uint8Array.from(bytes) });
      return { status: response.status, body: await response.text() };
    }, { url: upload.uploadUrl, headers: upload.requiredHeaders, bytes: [...evidenceBytes] });
    expect(putResult.status, putResult.body).toBeGreaterThanOrEqual(200);
    expect(putResult.status, putResult.body).toBeLessThan(300);
    const completedEvidence = await requireAPI<{ evidenceVersionId: string; scanState: string }>(page, "POST", `/api/v1/evidence/uploads/${encodeURIComponent(upload.uploadId)}/complete`, {
      operationId: operation("EVIDENCE-COMPLETE"),
      uploadId: upload.uploadId,
      sha256: evidenceDigest,
      byteSize: evidenceBytes.byteLength,
    });
    expect(completedEvidence.scanState).toBe("PENDING");
    let evidenceVersion: { id: string; scanState: string; reviewState: string; revision: number } | undefined;
    await expect.poll(async () => {
      const versions = await requireAPI<{ items: Array<{ id: string; scanState: string; reviewState: string; revision: number }> }>(page, "GET", `/api/v1/findings/${encodeURIComponent(finding.id)}/evidence`);
      evidenceVersion = versions.items.find((item) => item.id === completedEvidence.evidenceVersionId);
      return evidenceVersion?.scanState;
    }, { timeout: 90_000, intervals: [1_000, 2_000, 5_000] }).toBe("CLEAN");
    expect(evidenceVersion?.reviewState).toBe("PENDING_CAA_REVIEW");
    finding = await requireAPI<FindingProjection>(page, "GET", `/api/v1/findings/${encodeURIComponent(finding.id)}`);
    expect(finding.status).toBe("PENDING_CAA_REVIEW");
    await logout(page);

    await signIn(page, accounts.lead, "/lead-inspector/lead-review");
    const evidenceReview = await requireAPI<{ findingStatus: string; findingRevision: number }>(page, "POST", `/api/v1/evidence/${encodeURIComponent(evidenceVersion!.id)}/reviews`, {
      operationId: operation("EVIDENCE-CLOSE"),
      evidenceVersionId: evidenceVersion!.id,
      expectedEvidenceVersionRevision: evidenceVersion!.revision,
      findingId: finding.id,
      expectedFindingRevision: finding.revision,
      decision: "CLOSE",
      commentToAuditee: "Evidence accepted and verified; Finding closed on the evidence-verified basis.",
      internalCaaNote: "Synthetic internal verification note.",
    });
    expect(evidenceReview.findingStatus).toBe("CLOSED");
    finding = await requireAPI<FindingProjection>(page, "GET", `/api/v1/findings/${encodeURIComponent(finding.id)}`);
    expect(finding.status).toBe("CLOSED");
    await logout(page);

    await signIn(page, accounts.lead, "/lead-inspector/final-reports");
    let finalReport = await requireAPI<ReportProjection>(page, "POST", "/api/v1/report-versions", {
      operationId: operation("FINAL-CREATE"),
      idempotencyKey: operation("FINAL-IDEMPOTENCY"),
      expectedRevision: null,
      reportVersionId: `report-version:final:${suffix}`,
      reportId: `report:final:${suffix}`,
      auditId,
      kind: "FINAL",
      version: 1,
      status: "DEPARTMENT_REVIEW",
      findingIds: [finding.id],
      content: { title: "Privacy-safe Final Report", summary: "Synthetic connected lifecycle completed." },
    });
    await logout(page);
    await signIn(page, accounts.manager, "/department-manager/dashboard");
    finalReport = await decideReport(page, finalReport, "FORWARD", "Department Final Report review completed.");
    await logout(page);
    await signIn(page, accounts.gm, "/general-manager/report-approvals");
    finalReport = await decideReport(page, finalReport, "FORWARD", "General Manager Final Report review completed.");
    await logout(page);
    await signIn(page, accounts.executiveDirector, "/executive-director/final-reports");
    finalReport = await decideReport(page, finalReport, "ISSUE_AND_LOCK", "Executive Director issued the synthetic Final Report.");
    expect(finalReport.status).toBe("LOCKED");

    let finalDocument: DocumentProjection | undefined;
    await expect.poll(async () => {
      const documents = await requireAPI<{ items: DocumentProjection[] }>(page, "GET", "/api/v1/documents?organizationId=ORG-FLY-NAMIBIA&limit=100");
      finalDocument = documents.items.find((item) => item.id === finalReport.reportVersionId);
      return finalDocument?.renderStatus;
    }, { timeout: 120_000, intervals: [1_000, 2_000, 5_000] }).toBe("SUCCEEDED");
    expect(finalDocument?.documentVersionId).toEqual(expect.any(String));
    for (const digest of [finalDocument?.sha256, finalDocument?.rendererHash, finalDocument?.templateHash, finalDocument?.sourceHash]) {
      expect(digest).toMatch(/^sha256:[0-9a-f]{64}$/u);
    }
    await logout(page);

    await signIn(page, accounts.auditeeA, "/auditee/final-reports");
    const auditeeFindings = await requireAPI<{ items: unknown[] }>(page, "GET", "/api/v1/findings?limit=100");
    const auditeeReports = await requireAPI<{ items: unknown[] }>(page, "GET", "/api/v1/auditee/report-versions?kind=FINAL");
    expect(JSON.stringify({ auditeeFindings, auditeeReports })).not.toContain("internalCaaNote");
    expect(JSON.stringify(auditeeReports)).toContain(finalReport.reportVersionId);

    const managerNotificationContext = await browser.newContext({
      baseURL: requiredEnvironment("AVIA_E2E_BASE_URL"),
      ignoreHTTPSErrors: true,
    });
    const managerNotificationPage = await managerNotificationContext.newPage();
    try {
      await signIn(managerNotificationPage, accounts.manager, "/department-manager/dashboard");
      const deliverySubject = `Privacy-safe lifecycle delivery ${suffix}`;
      await requireAPI(page, "POST", "/api/v1/communications", {
        operationId: operation("AUDITEE-COMMUNICATION"),
        expectedRevision: null,
        idempotencyKey: operation("AUDITEE-COMMUNICATION-IDEMPOTENCY"),
        organizationId: "ORG-FLY-NAMIBIA",
        subject: deliverySubject,
        body: "Synthetic final-report delivery verification for the disposable local qualification.",
        audience: "CAA",
      });
      await expect.poll(async () => {
        const notifications = await requireAPI<{
          items: Array<{ body: string; emailDeliveryStatus: string }>;
        }>(managerNotificationPage, "GET", "/api/v1/notifications?limit=100");
        return notifications.items.find((item) => item.body.includes(deliverySubject))?.emailDeliveryStatus;
      }, { timeout: 60_000, intervals: [1_000, 2_000, 5_000] }).toBe("DELIVERED");
      await logout(managerNotificationPage);
    } finally {
      await managerNotificationContext.close();
    }
    await logout(page);

    const anonymousDownload = await apiRequest(page, "GET", `/api/v1/evidence/${encodeURIComponent(evidenceVersion!.id)}/download`);
    expect(anonymousDownload.status).toBe(401);
  });
});
