import { createHmac, createHash, randomBytes } from "node:crypto";

import {
  expect,
  request as playwrightRequest,
  test,
  type APIRequestContext,
  type BrowserContext,
  type Page,
} from "@playwright/test";

import { REACT_ROUTE_CONTRACTS } from "../../src/app/route-contracts";
import { FULL_PLATFORM_SCENARIO_FAMILIES } from "../contract/full-platform-backend.contract";

interface AppResponse<T> {
  status: number;
  body: T;
  etag: string | null;
}

interface LifecycleView {
  id: string;
  subjectId: string | null;
  status: string;
  failureReason: string | null;
}

interface PlanningItem {
  id: string;
  status: string;
  revision: number;
}

interface FindingView {
  id: string;
  revision: number;
  status: string;
}

type FullRole =
  | "inspector"
  | "leadInspector"
  | "manager"
  | "finance"
  | "gm"
  | "executiveDirector"
  | "auditee"
  | "admin";

interface ProvisionedPersona {
  context: BrowserContext;
  page: Page;
  role: Exclude<FullRole, "admin">;
  subjectID: string;
}

type RolePages = Record<FullRole, Page>;
type WorkflowSubjects = Pick<
  Record<FullRole, string>,
  "inspector" | "leadInspector"
>;

const applicationOrigin = requiredEnvironment("AVIA_E2E_BASE_URL");
const mailpitProofSubject = "Plan 3 full-profile SMTP delivery";

function requiredEnvironment(name: string): string {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`${name} is required`);
  return value;
}

function decodeBase32(value: string): Buffer {
  const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
  let bits = "";
  for (const character of value.toUpperCase().replace(/[^A-Z2-7]/g, "")) {
    const index = alphabet.indexOf(character);
    if (index < 0) throw new Error("Keycloak returned an invalid TOTP secret");
    bits += index.toString(2).padStart(5, "0");
  }
  const bytes: number[] = [];
  for (let index = 0; index + 8 <= bits.length; index += 8) {
    bytes.push(Number.parseInt(bits.slice(index, index + 8), 2));
  }
  return Buffer.from(bytes);
}

function totp(secret: string, now = Date.now()): string {
  const counter = BigInt(Math.floor(now / 30_000));
  const message = Buffer.alloc(8);
  message.writeBigUInt64BE(counter);
  const digest = createHmac("sha1", decodeBase32(secret)).update(message).digest();
  const offset = digest[digest.length - 1]! & 0x0f;
  const value = (
    ((digest[offset]! & 0x7f) << 24) |
    ((digest[offset + 1]! & 0xff) << 16) |
    ((digest[offset + 2]! & 0xff) << 8) |
    (digest[offset + 3]! & 0xff)
  ) % 1_000_000;
  return value.toString().padStart(6, "0");
}

async function avoidTotpBoundary(page: Page): Promise<void> {
  const remainingSeconds = 30 - (Math.floor(Date.now() / 1000) % 30);
  if (remainingSeconds < 4) {
    await page.waitForTimeout((remainingSeconds + 1) * 1000);
  }
}

async function beginLogin(
  page: Page,
  username: string,
  password: string,
): Promise<void> {
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: /Sign in to AviaSurveil360/i }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Sign in with organization identity" })
    .click();
  await page.getByLabel(/username or email/i).fill(username);
  await page.locator('input[name="password"]').fill(password);
  await page.getByRole("button", { name: /sign in/i }).click();
}

async function enrollTotp(
  page: Page,
  username: string,
  password: string,
  expectedRole: string,
): Promise<string> {
  await beginLogin(page, username, password);
  await expect(page.locator("#kc-totp-settings")).toBeVisible();
  await page.locator("#mode-manual").click();
  const secret = (await page.locator("#kc-totp-secret-key").innerText())
    .replace(/\s+/g, "");
  expect(secret).not.toBe("");
  await avoidTotpBoundary(page);
  await page.locator('input[name="totp"]').fill(totp(secret));
  const label = page.locator('input[name="userLabel"]');
  if (await label.isVisible()) await label.fill("Plan 3 clean full profile");
  await page.locator('button[type="submit"], input[type="submit"]').click();
  await expect.poll(() => new URL(page.url()).origin, { timeout: 60_000 })
    .toBe(applicationOrigin);
  await expect.poll(async () => {
    return page.evaluate(async () => {
      const response = await fetch("/auth/session", { credentials: "same-origin" });
      if (!response.ok) return [];
      const body = await response.json() as { roles?: string[] };
      return body.roles ?? [];
    });
  }).toContain(expectedRole);
  return secret;
}

async function appRequest<T>(
  page: Page,
  method: "GET" | "POST" | "PUT",
  path: string,
  body?: unknown,
  extraHeaders: Record<string, string> = {},
): Promise<AppResponse<T>> {
  return page.evaluate(
    async ({ method: requestMethod, path: requestPath, body: requestBody, extraHeaders: headers }) => {
      const csrf = document.cookie
        .split("; ")
        .find((entry) => entry.startsWith("__Host-avia_csrf="))
        ?.split("=")[1];
      const requestHeaders = new Headers({
        Accept: "application/json",
        ...headers,
      });
      if (requestBody !== undefined) {
        requestHeaders.set("content-type", "application/json");
        requestHeaders.set("x-csrf-token", decodeURIComponent(csrf ?? ""));
      }
      const response = await fetch(requestPath, {
        method: requestMethod,
        credentials: "same-origin",
        headers: requestHeaders,
        body: requestBody === undefined ? undefined : JSON.stringify(requestBody),
      });
      const text = await response.text();
      return {
        status: response.status,
        body: text ? JSON.parse(text) : null,
        etag: response.headers.get("etag"),
      };
    },
    { method, path, body, extraHeaders },
  ) as Promise<AppResponse<T>>;
}

function commandHeaders(key: string, revision?: number): Record<string, string> {
  return revision === undefined
    ? { "Idempotency-Key": key }
    : { "Idempotency-Key": key, "If-Match": `"rev-${revision}"` };
}

function expectStatus<T>(
  response: AppResponse<T>,
  status: number,
  label: string,
): T {
  expect(response.status, `${label}: ${JSON.stringify(response.body)}`).toBe(status);
  return response.body;
}

async function keycloakAdminToken(request: APIRequestContext): Promise<string> {
  const response = await request.post(
    `${requiredEnvironment("AVIA_LOCAL_FULL_KEYCLOAK_BASE_URL")}/realms/master/protocol/openid-connect/token`,
    {
      form: {
        client_id: "admin-cli",
        grant_type: "password",
        username: requiredEnvironment("AVIA_LOCAL_FULL_KEYCLOAK_ADMIN_USERNAME"),
        password: requiredEnvironment("AVIA_LOCAL_FULL_KEYCLOAK_ADMIN_PASSWORD"),
      },
    },
  );
  expect(response.status()).toBe(200);
  const body = await response.json() as { access_token?: string };
  expect(body.access_token).toBeTruthy();
  return body.access_token!;
}

async function configureProvisionedLogin(
  request: APIRequestContext,
  token: string,
  subjectID: string,
  password: string,
): Promise<void> {
  const passwordResponse = await request.put(
    `${requiredEnvironment("AVIA_LOCAL_FULL_KEYCLOAK_BASE_URL")}/admin/realms/aviasurveil360/users/${subjectID}/reset-password`,
    {
      headers: { Authorization: `Bearer ${token}` },
      data: { type: "password", temporary: false, value: password },
    },
  );
  expect(passwordResponse.status()).toBe(204);
  const requiredActionResponse = await request.put(
    `${requiredEnvironment("AVIA_LOCAL_FULL_KEYCLOAK_BASE_URL")}/admin/realms/aviasurveil360/users/${subjectID}`,
    {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        emailVerified: true,
        requiredActions: ["CONFIGURE_TOTP"],
      },
    },
  );
  expect(requiredActionResponse.status()).toBe(204);
}

async function provisionRole(
  adminPage: Page,
  keycloakRequest: APIRequestContext,
  role: Exclude<FullRole, "admin">,
  organizationID: string,
): Promise<ProvisionedPersona> {
  const email = `full.${role.toLowerCase()}.${randomBytes(6).toString("hex")}@example.test`;
  const password = `${randomBytes(18).toString("hex")}Aa1!`;
  const key = `full-${role.toLowerCase()}-${randomBytes(8).toString("hex")}`;
  const accepted = expectStatus(
    await appRequest<LifecycleView>(
      adminPage,
      "POST",
      "/api/v1/admin/user-lifecycle-requests",
      {
        operationId: key,
        idempotencyKey: key,
        subjectId: null,
        action: "PROVISION",
        roles: [role],
        organizationId: organizationID,
        email,
        displayName: `Clean Full Profile ${role}`,
        reason: `Create the isolated ${role} authority for the clean full profile.`,
      },
      commandHeaders(key),
    ),
    202,
    `provision ${role}`,
  );
  let lifecycle = accepted;
  await expect.poll(async () => {
    const response = await appRequest<LifecycleView>(
      adminPage,
      "GET",
      `/api/v1/admin/user-lifecycle-requests/${accepted.id}`,
    );
    expect(response.status).toBe(200);
    lifecycle = response.body;
    return lifecycle.status;
  }, { timeout: 90_000 }).toBe("SUCCEEDED");
  expect(lifecycle.failureReason).toBeNull();
  expect(lifecycle.subjectId).toBeTruthy();

  const token = await keycloakAdminToken(keycloakRequest);
  await configureProvisionedLogin(
    keycloakRequest,
    token,
    lifecycle.subjectId!,
    password,
  );

  const context = await adminPage.context().browser()!.newContext({
    baseURL: applicationOrigin,
    ignoreHTTPSErrors: true,
  });
  const page = await context.newPage();
  await enrollTotp(page, email, password, role);
  return { context, page, role, subjectID: lifecycle.subjectId! };
}

async function proveSMTPNotification(
  adminPage: Page,
  auditeePage: Page,
): Promise<void> {
  const key = `full-mailpit-${randomBytes(8).toString("hex")}`;
  expectStatus(
    await appRequest(
      adminPage,
      "POST",
      "/api/v1/communications",
      {
        operationId: key,
        expectedRevision: null,
        idempotencyKey: key,
        organizationId: "ORG-FLY-NAMIBIA",
        subject: mailpitProofSubject,
        body: "Normal application communication delivered through private SMTP.",
        audience: "AUDITEE",
      },
      commandHeaders(key),
    ),
    201,
    "send SMTP-backed Auditee communication",
  );

  let notification: {
    body: string;
    emailDeliveryStatus: string;
    emailDeliveryAttempts: number;
    emailAcceptedAt: string | null;
  } | undefined;
  await expect.poll(async () => {
    const response = await appRequest<{
      items: Array<{
        body: string;
        emailDeliveryStatus: string;
        emailDeliveryAttempts: number;
        emailAcceptedAt: string | null;
      }>;
    }>(auditeePage, "GET", "/api/v1/notifications");
    expect(response.status).toBe(200);
    notification = response.body.items.find((item) =>
      item.body.includes(mailpitProofSubject)
    );
    return notification?.emailDeliveryStatus;
  }, { timeout: 90_000 }).toBe("DELIVERED");
  expect(notification?.emailDeliveryAttempts).toBeGreaterThanOrEqual(1);
  expect(notification?.emailAcceptedAt).toBeTruthy();
}

async function createAdminPrerequisites(adminPage: Page): Promise<string[]> {
  expectStatus(
    await appRequest(
      adminPage,
      "POST",
      "/api/v1/admin/organizations",
      {
        operationId: "full-create-organization",
        idempotencyKey: "full-create-organization",
        expectedRevision: null,
        organizationId: "ORG-FLY-NAMIBIA",
        legalName: "Fly Namibia",
        organizationType: "OPERATOR",
      },
      commandHeaders("full-create-organization"),
    ),
    201,
    "create organization",
  );

  const questionIDs = [
    "CAB-EMEQ-PBE-001",
    "CAB-LAV-001",
    "CAB-PAX-SEAT-001",
    "CAB-VID-CREW-SEAT-001",
    "CAB-GALLEY-001",
    "CAB-DOORS-001",
  ];
  expectStatus(
    await appRequest(
      adminPage,
      "POST",
      "/api/v1/admin/reminder-rules",
      {
        operationId: "full-create-reminder",
        idempotencyKey: "full-create-reminder",
        expectedRevision: null,
        ruleId: "REM-OVERDUE",
        label: "Overdue Finding reminder",
        offsetDays: -1,
        channel: "IN_APP_AND_EMAIL",
      },
      commandHeaders("full-create-reminder"),
    ),
    201,
    "create reminder rule",
  );
  return questionIDs;
}

async function createCleanApplicationState(
  rolePages: RolePages,
  subjects: WorkflowSubjects,
  questionIDs: string[],
): Promise<void> {
  const values = {
    organizationId: "ORG-FLY-NAMIBIA",
    organizationName: "Fly Namibia",
    applicationType: "Continued Surveillance",
    domain: "Cabin Safety",
    inspectionCategory: "Ad Hoc / Unannounced",
    noticePolicy: "WITHHELD",
    purpose: "Risk-triggered clean full-profile cabin inspection.",
    triggerType: "Authorized local verification",
    riskCategory: "Cabin Safety",
    plannedDate: "2026-07-25",
    mode: "On-site",
    location: "Windhoek",
    templateVersionId: "CTV-CABIN-1",
    scope: "Cabin safety and emergency equipment.",
    requestedBudget: 0,
    currency: "USD",
  };
  expectStatus(
    await appRequest(
      rolePages.manager,
      "POST",
      "/api/v1/planning/intake-drafts",
      {
        operationId: "full-create-intake",
        idempotencyKey: "full-create-intake",
        expectedRevision: null,
        draftId: "PLAN-DRAFT-2026-001",
        values,
      },
      commandHeaders("full-create-intake"),
    ),
    201,
    "create planning intake",
  );
  const submitted = expectStatus<{ planningItem: PlanningItem }>(
    await appRequest(
      rolePages.manager,
      "POST",
      "/api/v1/planning/intake-drafts/PLAN-DRAFT-2026-001/submissions",
      {
        operationId: "full-submit-intake",
        expectedRevision: 1,
        idempotencyKey: "full-submit-intake",
        draftId: "PLAN-DRAFT-2026-001",
        planningItemId: "PLAN-2026-AD-HOC-001",
      },
      commandHeaders("full-submit-intake", 1),
    ),
    200,
    "submit planning intake",
  );
  let planning = submitted.planningItem;
  for (const [page, decision, reason] of [
    [rolePages.finance, "APPROVE_BUDGET", "No external budget is required."],
    [
      rolePages.gm,
      "FORWARD_FOR_FINAL_APPROVAL",
      "Forward the exact plan revision.",
    ],
    [
      rolePages.executiveDirector,
      "APPROVE_PLAN",
      "Approve the authorized plan revision.",
    ],
    [rolePages.gm, "RELEASE_PLAN", "Release the authorized plan."],
  ] as const) {
    planning = expectStatus<PlanningItem>(
      await appRequest(
        page,
        "POST",
        `/api/v1/planning/items/${planning.id}/decisions`,
        {
          operationId: `full-plan-${decision.toLowerCase()}`,
          planningItemId: planning.id,
          expectedPlanningRevision: planning.revision,
          decision,
          reason,
        },
      ),
      200,
      `planning ${decision}`,
    );
  }
  expect(planning).toMatchObject({ status: "RELEASED", revision: 5 });

  expectStatus(
    await appRequest(
      rolePages.manager,
      "POST",
      "/api/v1/audit-workspaces",
      {
        operationId: "full-create-audit-workspace",
        idempotencyKey: "full-create-audit-workspace",
        planningItemId: planning.id,
        expectedPlanningRevision: planning.revision,
        auditId: "AUD-2026-001",
        assignmentId: "ASG-AUD-2026-001",
        packageId: "PKG-CAB-2026-001",
        packageDraftId: "PKG-AUD-2026-001-CABIN",
        templateId: "TPL-CABIN-2026",
        templateVersionId: "CTV-CABIN-1",
        leadInspectorSubjectId: subjects.leadInspector,
        memberSubjectIds: [subjects.inspector],
        scheduledStartDate: "2026-07-25",
        scheduledEndDate: "2026-07-27",
        expiresAt: "2026-08-25T00:00:00Z",
        questions: questionIDs.map((questionId) => ({
          questionId,
          prompt: questionId,
          configuredReference: questionId,
          expectedEvidence: [questionId],
          assignedInspectorSubjectIds: [subjects.inspector],
        })),
      },
      commandHeaders("full-create-audit-workspace"),
    ),
    201,
    "create audit workspace",
  );
}

async function createAndCloseFinding(
  inspectorPage: Page,
  leadInspectorPage: Page,
  auditeePage: Page,
): Promise<{ finding: FindingView; evidenceVersionID: string }> {
  const response = expectStatus<{ id: string; revision: number; comment: string }>(
    await appRequest(
      inspectorPage,
      "PUT",
      "/api/v1/checklist-responses/RESP-CAB-EMEQ-PBE-001",
      {
        operationId: "full-checklist-response",
        responseId: "RESP-CAB-EMEQ-PBE-001",
        auditId: "AUD-2026-001",
        questionId: "CAB-EMEQ-PBE-001",
        expectedResponseRevision: null,
        answer: "NON_COMPLIANT",
        comment: "PBE serviceability requires exact Evidence.",
      },
    ),
    200,
    "record checklist response",
  );
  const potential = expectStatus<{ id: string; revision: number }>(
    await appRequest(
      inspectorPage,
      "POST",
      "/api/v1/potential-findings",
      {
        operationId: "full-create-potential",
        auditId: "AUD-2026-001",
        questionId: "CAB-EMEQ-PBE-001",
        checklistResponseId: response.id,
        expectedChecklistResponseRevision: response.revision,
        title: "PBE serviceability not confirmed",
        description: "The configured cabin control requires Evidence.",
        requiredComment: response.comment,
        inspectionAttachmentIds: [],
      },
    ),
    201,
    "create Potential Finding",
  );
  const converted = expectStatus<{ finding: FindingView }>(
    await appRequest(
      leadInspectorPage,
      "POST",
      `/api/v1/potential-findings/${potential.id}/decisions`,
      {
        operationId: "full-convert-potential",
        potentialFindingId: potential.id,
        expectedPotentialFindingRevision: potential.revision,
        decision: "CONVERT",
        severity: "LEVEL_1_CRITICAL",
        capRequired: true,
        evidenceRequired: true,
        dueDate: "2026-07-23",
      },
    ),
    200,
    "convert Potential Finding",
  );
  let finding = converted.finding;

  const cap = expectStatus<{
    capRevisionId: string;
    capRevision: number;
    findingRevision: number;
  }>(
    await appRequest(
      auditeePage,
      "POST",
      "/api/v1/caps",
      {
        operationId: "full-submit-cap",
        findingId: finding.id,
        expectedFindingRevision: finding.revision,
        rootCause: "The position record was not reconciled.",
        correctiveAction: "Service the PBE and reconcile its position.",
        preventiveAction: "Add recorded monthly supervisor sampling.",
        responsiblePerson: "Fly Namibia Cabin Safety Manager",
        targetCompletionDate: "2026-07-25",
        commentToCaa: "CAP submitted through the Auditee organization boundary.",
      },
    ),
    201,
    "submit CAP",
  );
  const accepted = expectStatus<{ findingRevision: number; findingStatus: string }>(
    await appRequest(
      inspectorPage,
      "POST",
      `/api/v1/caps/${cap.capRevisionId}/reviews`,
      {
        operationId: "full-accept-cap",
        capRevisionId: cap.capRevisionId,
        expectedCapRevision: cap.capRevision,
        findingId: finding.id,
        expectedFindingRevision: cap.findingRevision,
        decision: "ACCEPT",
        commentToAuditee: "CAP accepted; exact Evidence remains required.",
        internalCaaNote: "CAP acceptance is not Finding closure.",
      },
    ),
    200,
    "accept CAP without closure",
  );
  expect(accepted.findingStatus).not.toBe("CLOSED");
  finding = { ...finding, revision: accepted.findingRevision, status: accepted.findingStatus };

  const body = Buffer.from(
    "%PDF-1.7\n1 0 obj\n<</Type/Catalog/Label(Clean full scan)>>\nendobj\n%%EOF\n",
  );
  const sha256 = `sha256:${createHash("sha256").update(body).digest("hex")}`;
  const upload = expectStatus<{
    uploadId: string;
    uploadUrl: string;
    requiredHeaders: Record<string, string>;
  }>(
    await appRequest(
      auditeePage,
      "POST",
      "/api/v1/evidence/uploads",
      {
        operationId: "full-evidence-begin",
        findingId: finding.id,
        expectedFindingRevision: finding.revision,
        fileName: "PBE_Serviceability_Record.pdf",
        declaredMediaType: "application/pdf",
        byteSize: body.byteLength,
        sha256,
      },
    ),
    201,
    "begin MinIO Evidence object upload",
  );
  const objectUpload = await auditeePage.request.put(upload.uploadUrl, {
    headers: upload.requiredHeaders,
    data: body,
  });
  expect(objectUpload.ok()).toBe(true);
  const completed = expectStatus<{ evidenceVersionId: string }>(
    await appRequest(
      auditeePage,
      "POST",
      `/api/v1/evidence/uploads/${upload.uploadId}/complete`,
      {
        operationId: "full-evidence-complete",
        uploadId: upload.uploadId,
        sha256,
        byteSize: body.byteLength,
      },
    ),
    200,
    "complete private versioned object upload",
  );
  let evidence: { id: string; revision: number; scanState: string } | undefined;
  await expect.poll(async () => {
    const versions = expectStatus<{ items: Array<{
      id: string;
      revision: number;
      scanState: string;
    }> }>(
      await appRequest(
        auditeePage,
        "GET",
        `/api/v1/findings/${finding.id}/evidence`,
      ),
      200,
      "poll ClamAV Evidence scan",
    );
    evidence = versions.items.find(({ id }) => id === completed.evidenceVersionId);
    return evidence?.scanState;
  }, { timeout: 120_000 }).toBe("CLEAN");

  const reviewableFinding = expectStatus<FindingView>(
    await appRequest(
      inspectorPage,
      "GET",
      `/api/v1/findings/${finding.id}`,
    ),
    200,
    "read current Finding revision after ClamAV processing",
  );
  expect(reviewableFinding.status).toBe("PENDING_CAA_REVIEW");
  const closed = expectStatus<{ findingRevision: number; findingStatus: string }>(
    await appRequest(
      inspectorPage,
      "POST",
      `/api/v1/evidence/${evidence!.id}/reviews`,
      {
        operationId: "full-close-with-scan-clean-evidence",
        evidenceVersionId: evidence!.id,
        expectedEvidenceVersionRevision: evidence!.revision,
        findingId: finding.id,
        expectedFindingRevision: reviewableFinding.revision,
        decision: "CLOSE",
        commentToAuditee: "The exact scan-clean Evidence version is accepted.",
        internalCaaNote: "ClamAV scan-clean Evidence verified before closure.",
      },
    ),
    200,
    "close Finding with exact scan-clean Evidence",
  );
  expect(closed.findingStatus).toBe("CLOSED");
  return {
    finding: { ...finding, revision: closed.findingRevision, status: closed.findingStatus },
    evidenceVersionID: evidence!.id,
  };
}

async function proveReportAndPDF(
  leadInspectorPage: Page,
  managerPage: Page,
  generalManagerPage: Page,
  executiveDirectorPage: Page,
  findingID: string,
): Promise<void> {
  const report = expectStatus<{
    reportVersionId: string;
    revision: number;
    status: string;
  }>(
    await appRequest(
      leadInspectorPage,
      "POST",
      "/api/v1/report-versions",
      {
        operationId: "full-create-final-report",
        idempotencyKey: "full-create-final-report",
        expectedRevision: null,
        reportVersionId: "RPT-CAB-2026-001-V1",
        reportId: "RPT-CAB-2026-001",
        auditId: "AUD-2026-001",
        kind: "FINAL",
        version: 1,
        status: "DEPARTMENT_REVIEW",
        findingIds: [findingID],
        content: {
          title: "Clean Full Profile Final Report",
          conclusion: "Authority remains explicit and Evidence-backed.",
        },
      },
      commandHeaders("full-create-final-report"),
    ),
    201,
    "create immutable Final Report version",
  );
  let current = report;
  for (const [page, decision, reason] of [
    [managerPage, "FORWARD", "Forward Department review."],
    [generalManagerPage, "FORWARD", "Forward General Manager review."],
    [
      executiveDirectorPage,
      "ISSUE_AND_LOCK",
      "Issue the exact immutable Final Report.",
    ],
  ] as const) {
    current = expectStatus(
      await appRequest<typeof current>(
        page,
        "POST",
        `/api/v1/report-versions/${current.reportVersionId}/decisions`,
        {
          operationId: `full-report-${current.revision}-${decision}`,
          reportVersionId: current.reportVersionId,
          expectedReportVersionRevision: current.revision,
          decision,
          reason,
        },
      ),
      200,
      `report ${decision}`,
    );
  }
  expect(current.status).toBe("LOCKED");

  await expect.poll(async () => {
    const documents = expectStatus<{ items: Array<{
      id: string;
      renderStatus?: string;
    }> }>(
      await appRequest(executiveDirectorPage, "GET", "/api/v1/documents"),
      200,
      "poll Gotenberg PDF document",
    );
    return documents.items.find(({ id }) => id === current.reportVersionId)?.renderStatus;
  }, { timeout: 120_000 }).toBe("SUCCEEDED");

  const document = expectStatus<{
    documentVersionId: string;
    downloadFileName: string;
    downloadUrl: string;
    sha256: string;
    rendererHash: string;
    templateHash: string;
    sourceHash: string;
  }>(
    await appRequest(
      executiveDirectorPage,
      "GET",
      `/api/v1/documents/${current.reportVersionId}`,
    ),
    200,
    "authorize rendered Gotenberg PDF download",
  );
  expect(document.documentVersionId).toBeTruthy();
  expect(document.downloadFileName).toMatch(/\.pdf$/);
  expect(document.sha256).toMatch(/^sha256:[0-9a-f]{64}$/);
  expect(document.rendererHash).toBe(
    "sha256:56c47f7b913f3b978554115a0191c4a9dcc2558f9090f27f3f13f28a7c2f8329",
  );
  expect(document.templateHash).toMatch(/^sha256:[0-9a-f]{64}$/);
  expect(document.sourceHash).toMatch(/^sha256:[0-9a-f]{64}$/);
  const pdf = await executiveDirectorPage.request.get(document.downloadUrl);
  expect(pdf.status()).toBe(200);
  expect(pdf.headers()["content-type"]).toContain("application/pdf");
  expect((await pdf.body()).subarray(0, 5).toString("utf8")).toBe("%PDF-");
}

async function proveDirectLoads(rolePages: RolePages): Promise<string[]> {
  expect(REACT_ROUTE_CONTRACTS).toHaveLength(86);
  const loaded: string[] = [];
  for (const route of REACT_ROUTE_CONTRACTS) {
    const page = rolePages[route.requiredRole ?? "admin"];
    const response = await page.goto(route.path, { waitUntil: "domcontentloaded" });
    expect(response?.status(), `direct load ${route.path}`).toBe(200);
    await expect(page.locator("main")).toHaveCount(1);
    expect(new URL(page.url()).pathname).toBe(route.path);
    loaded.push(route.id);
  }
  expect(loaded).toHaveLength(86);
  expect(new Set(loaded).size).toBe(86);
  return loaded;
}

test("clean full profile proves 86 HTTP routes and ten real-service scenario families", async ({
  page: adminPage,
}) => {
  test.setTimeout(900_000);
  expect(FULL_PLATFORM_SCENARIO_FAMILIES).toHaveLength(10);
  const scenarioProofs = new Set<string>();
  const keycloakRequest = await playwrightRequest.newContext({
    ignoreHTTPSErrors: true,
  });
  const roleContexts: BrowserContext[] = [];
  try {
    await enrollTotp(
      adminPage,
      requiredEnvironment("AVIA_LOCAL_FULL_ADMIN_USERNAME"),
      requiredEnvironment("AVIA_LOCAL_FULL_ADMIN_PASSWORD"),
      "admin",
    );
    const missingTestBoundary = await appRequest(
      adminPage,
      "POST",
      ["/__test", "reset"].join("/"),
      {},
    );
    expect(missingTestBoundary.status, "/__test/* must return 404 in full mode").toBe(404);

    const questionIDs = await createAdminPrerequisites(adminPage);
    const personas = new Map<
      Exclude<FullRole, "admin">,
      ProvisionedPersona
    >();
    for (const role of [
      "inspector",
      "leadInspector",
      "manager",
      "finance",
      "gm",
      "executiveDirector",
      "auditee",
    ] as const) {
      const persona = await provisionRole(
        adminPage,
        keycloakRequest,
        role,
        role === "auditee" ? "ORG-FLY-NAMIBIA" : "CAA",
      );
      personas.set(role, persona);
      roleContexts.push(persona.context);
    }
    const persona = (
      role: Exclude<FullRole, "admin">,
    ): ProvisionedPersona => {
      const value = personas.get(role);
      if (!value) throw new Error(`missing ${role} full-profile persona`);
      return value;
    };
    const rolePages: RolePages = {
      admin: adminPage,
      inspector: persona("inspector").page,
      leadInspector: persona("leadInspector").page,
      manager: persona("manager").page,
      finance: persona("finance").page,
      gm: persona("gm").page,
      executiveDirector: persona("executiveDirector").page,
      auditee: persona("auditee").page,
    };
    await createCleanApplicationState(
      rolePages,
      {
        inspector: persona("inspector").subjectID,
        leadInspector: persona("leadInspector").subjectID,
      },
      questionIDs,
    );
    scenarioProofs.add("ad-hoc-planning-to-assignment");
    scenarioProofs.add("configuration-and-immutable-package-snapshot");

    await proveSMTPNotification(rolePages.inspector, rolePages.auditee);
    const { finding, evidenceVersionID } = await createAndCloseFinding(
      rolePages.inspector,
      rolePages.leadInspector,
      rolePages.auditee,
    );
    expect(evidenceVersionID).toBeTruthy();
    scenarioProofs.add("routine-inspection-to-closure");
    scenarioProofs.add("checklist-and-potential-finding-authority");
    scenarioProofs.add("cap-evidence-and-closure-authority");

    await proveReportAndPDF(
      rolePages.leadInspector,
      rolePages.manager,
      rolePages.gm,
      rolePages.executiveDirector,
      finding.id,
    );
    scenarioProofs.add("preliminary-and-final-report-authority");

    const organizations = expectStatus<{ items: unknown[] }>(
      await appRequest(adminPage, "GET", "/api/v1/organizations"),
      200,
      "organization projection",
    );
    expect(organizations.items.length).toBeGreaterThanOrEqual(1);
    scenarioProofs.add("organization-and-platform-projections");

    const risk = expectStatus<{ advisoryOnly?: boolean }>(
      await appRequest(rolePages.manager, "GET", "/api/v1/risk/overview"),
      200,
      "advisory risk projection",
    );
    expect(risk).toBeTruthy();
    scenarioProofs.add("advisory-management-projections");

    const checkout = expectStatus<{ offlineGrant: { grantId: string } }>(
      await appRequest(
        rolePages.inspector,
        "POST",
        "/api/v1/inspection-packages/PKG-CAB-2026-001/checkout",
        {
          operationId: "full-offline-checkout",
          packageId: "PKG-CAB-2026-001",
          expectedPackageVersion: 1,
          deviceInstanceId: "plan3-clean-full-device",
        },
      ),
      200,
      "offline recovery checkout",
    );
    expect(checkout.offlineGrant.grantId).toBeTruthy();
    scenarioProofs.add("offline-causal-sync-and-session-boundaries");

    const guidance = expectStatus<{
      advisoryOnly: boolean;
      prohibitedActions: string[];
    }>(
      await appRequest(
        rolePages.inspector,
        "GET",
        "/api/v1/assistant/guidance",
      ),
      200,
      "assistant advisory boundary",
    );
    expect(guidance.advisoryOnly).toBe(true);
    expect(guidance.prohibitedActions.length).toBeGreaterThan(0);
    const draft = expectStatus<{
      advisoryOnly: boolean;
      canCreateFinding: boolean;
      canSetSeverity: boolean;
      canCloseFinding: boolean;
    }>(
      await appRequest(
        rolePages.inspector,
        "POST",
        "/api/v1/assistant/drafts",
        {
          operationId: "full-assistant-draft",
          expectedRevision: null,
          idempotencyKey: "full-assistant-draft",
          findingId: finding.id,
          prompt: "Summarize the accepted Evidence without changing authority.",
        },
        commandHeaders("full-assistant-draft"),
      ),
      201,
      "assistant advisory draft",
    );
    expect(draft).toMatchObject({
      advisoryOnly: true,
      canCreateFinding: false,
      canSetSeverity: false,
      canCloseFinding: false,
    });
    scenarioProofs.add("advisory-draft-without-canonical-mutation");

    await proveDirectLoads(rolePages);
    expect([...scenarioProofs].sort()).toEqual(
      [...FULL_PLATFORM_SCENARIO_FAMILIES].sort(),
    );
    expect([...scenarioProofs]).toHaveLength(10);

    await test.info().attach("local-full-platform-summary", {
      body: JSON.stringify({
        directLoads: 86,
        scenarioFamilies: [...scenarioProofs],
        identity: "eight exact-role production-mode Keycloak CONFIGURE_TOTP sessions",
        evidence: "private MinIO object and real ClamAV scan-clean gating",
        notifications: "private Mailpit SMTP is inspected by the profile script",
        documents: "real Gotenberg PDF rendered into private versioned object storage",
      }, null, 2),
      contentType: "application/json",
    });
  } finally {
    await Promise.all(roleContexts.map((context) => context.close()));
    await keycloakRequest.dispose();
  }
});
