import { expect, test, type Page } from "@playwright/test";
import { createHttpBackend } from "../../src/backend/http-backend";
import { createCanonicalTestFetch } from "../../src/test-profile/http-test-boundary";

import {
  expectCanonicalScenarioTranscript,
  type CanonicalScenarioTranscript,
} from "./support/scenario-transcript";

const apiURL = process.env.AVIA_HTTP_API_URL ?? "http://127.0.0.1:58081";
const testToken = process.env.AVIA_CANONICAL_TEST_TOKEN ?? "";

async function submitEvidence(page: Page, version: number, fileName: string): Promise<void> {
  await page.getByTestId("evidence-file").setInputFiles({
    name: fileName,
    mimeType: "application/pdf",
    buffer: Buffer.from(`%PDF-1.7\n1 0 obj\n<</Type/Catalog/Version(${version})>>\nendobj\n%%EOF\n`),
  });
  await expect(page.getByTestId("selected-evidence-file")).toHaveText(fileName);
  await page.getByRole("button", { name: "Submit Evidence version" }).click();
  await expect(page.getByTestId("evidence-version-count")).toHaveText(String(version));
}

async function recordEvidenceDecision(
  page: Page,
  decision: "PARTIALLY_CLOSE" | "NOT_CLOSE" | "CLOSE",
  publicComment: string,
  internalNote: string,
): Promise<void> {
  await page.getByLabel("Evidence review decision").selectOption(decision);
  await page.getByLabel("Comment to Auditee").fill(publicComment);
  await page.getByLabel("Internal CAA Note").fill(internalNote);
  await page.getByRole("button", { name: "Record Evidence review" }).click();
}

test.beforeEach(async ({ request }, testInfo) => {
  if (testInfo.project.name !== "http") return;
  const response = await request.post(`${apiURL}/__test/reset`, {
    headers: { "x-avia-test-token": testToken },
  });
  expect(response.ok()).toBe(true);
});

test("canonical Cabin Inspection lifecycle is backend-shaped and organization-safe", async ({
  page,
  request,
}, testInfo) => {
  test.setTimeout(90_000);
  const consoleIssues: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error" || message.type() === "warning") {
      consoleIssues.push(`${message.type()}: ${message.text()}`);
    }
  });
  page.on("pageerror", (error) => consoleIssues.push(`pageerror: ${error.message}`));
  let captureAuditeeTransport = false;
  const auditeeTransportPayloads: string[] = [];
  page.on("response", async (response) => {
    if (!captureAuditeeTransport || testInfo.project.name !== "http") return;
    if (response.request().headers()["x-avia-test-subject"] !== "USR-AUDITEE-FLY") return;
    if (!/\/v1\/(?:findings|organizations)|\/cap-revisions|\/evidence/.test(response.url())) return;
    if (!response.headers()["content-type"]?.includes("json")) return;
    try {
      auditeeTransportPayloads.push(await response.text());
    } catch {
      // A navigation can dispose an already-observed response; completed route assertions remain authoritative.
    }
  });

  if (testInfo.project.name === "http") {
    // The canonical HTTP package is server-assigned to the canonical UUID
    // subject.  Use that exact server-owned identity for fixture prefill;
    // the human-facing role label remains a UI concern and must not be used
    // to bypass the assignment boundary.
    const otherInspector = createHttpBackend(
      { apiBaseUrl: apiURL, environmentLabel: "Canonical HTTP fixture" },
      { fetchImplementation: createCanonicalTestFetch("154ec5ac-6f97-4f55-916f-d2f142fc6211", testToken) },
    );
    const packageView = await otherInspector.inspections.getPackage({ packageId: "PKG-CAB-2026-001" });
    for (const question of packageView.questions) {
      if (question.id === "CAB-EMEQ-PBE-001") continue;
      await otherInspector.inspections.upsertChecklistResponse({
        operationId: `OP-E2E-PREFILL-${question.id}`,
        responseId: `RESP-${question.id}`,
        auditId: packageView.auditId,
        questionId: question.id,
        expectedResponseRevision: null,
        answer: "COMPLIANT",
        comment: "",
      });
    }
  } else {
    await page.goto("/");
    const fixtureResponses = await page.evaluate(async () => {
      if (!window.__aviaCompleteCanonicalChecklistForTest) throw new Error("Mock checklist fixture seam is unavailable.");
      return window.__aviaCompleteCanonicalChecklistForTest();
    });
    expect(fixtureResponses).toEqual(expect.arrayContaining([
      { id: "CAB-EMEQ-PBE-001", assignedInspectorUserIds: ["USR-INSPECTOR-AMINA"], currentResponse: null },
    ]));
  }

  await page.goto("/");
  await expect(page.getByRole("heading", { name: "AviaSurveil360" })).toBeVisible();
  await expect(page.getByRole("link", { name: /CAA Inspector/i })).toBeVisible();
  await page.getByRole("link", { name: /CAA Inspector/i }).click();

  await expect(page).toHaveURL(/\/inspector\/inspector-assignments$/);
  await expect(page.getByRole("heading", { name: "My Assignments" })).toBeVisible();
  await expect(page.getByText("2026 Cabin Inspection - Fly Namibia")).toBeVisible();
  await page.getByRole("link", { name: "Open Cabin Inspection" }).click();

  await expect(page.getByTestId("audit-id")).toHaveText("AUD-2026-001");
  await page.getByRole("link", { name: "Run Cabin checklist" }).click();
  await expect(page.getByRole("heading", { name: "Cabin Inspection checklist" })).toBeVisible();
  await expect(page.getByTestId("checklist-question-row")).toHaveCount(6);
  await expect(page.getByTestId("question-CAB-GALLEY-001")).toContainText(
    "Assigned to another Inspector — read-only",
  );

  await page.getByRole("button", { name: "Open question 4" }).click();
  await page.getByLabel("Checklist answer").selectOption("NON_COMPLIANT");
  await page
    .getByLabel("Inspector comment")
    .fill("PBE serviceability and accessibility could not be confirmed for this Audit.");
  await page.getByRole("button", { name: "Save response" }).click();
  await expect(page.getByTestId("response-status")).toContainText("Server acknowledged");
  await expect(page.getByTestId("response-status")).toContainText("NON COMPLIANT");
  await page.getByRole("button", { name: "Create Potential Finding" }).click();
  await expect(page.getByTestId("potential-finding-id")).toHaveText("PF-2026-001");
  await expect(page.getByTestId("potential-finding-status")).toHaveText(
    "PENDING_LEAD_REVIEW",
  );
  await expect(page.getByRole("button", { name: "Switch to Lead Inspector" })).toHaveCount(0);
  await expect(page.getByTestId("active-role")).toHaveText("CAA Inspector");
  await page.getByRole("button", { name: "Submit checklist to Lead Inspector" }).click();
  await expect(page.getByTestId("checklist-status")).toHaveText("SUBMITTED");
  await page.getByRole("button", { name: "Switch to Lead Inspector" }).click();

  await expect(page).toHaveURL(/\/lead-inspector\/lead-review$/);
  await expect(page.getByTestId("potential-finding-dossier")).toContainText("PF-2026-001");
  await page.getByLabel("Finding severity").selectOption("LEVEL_1_CRITICAL");
  await expect(page.getByLabel("CAP required")).toBeChecked();
  await expect(page.getByLabel("Evidence required")).toBeChecked();
  await page.getByRole("button", { name: "Convert to Finding" }).click();
  await expect(page.getByTestId("finding-number")).toHaveText("CAB-2026-001");
  await expect(page.getByTestId("finding-status")).toHaveText("WAITING_FOR_CAP");
  await page.getByRole("link", { name: "Open Lead CAP review" }).click();
  await expect(page.getByRole("heading", { level: 1, name: /^CAP Review/ })).toBeVisible();
  await page.getByRole("button", { name: "Switch to CAA Inspector Finding" }).click();
  await expect(page).toHaveURL(/\/inspector\/findings\/FND-CAB-2026-001$/);

  for (const expected of [
    "Fly Namibia",
    "AUD-2026-001",
    "Level 1 Critical",
    "Auditee to submit CAP",
    "15 Jul 2026",
  ]) {
    await expect(page.getByTestId("finding-dossier")).toContainText(expected);
  }
  captureAuditeeTransport = true;
  await page.getByRole("button", { name: "Switch to Fly Namibia Auditee" }).click();

  await expect(page).toHaveURL(/\/auditee\/service-provider-cap$/);
  await expect(page.getByRole("heading", { name: "Corrective Actions (CAP)" })).toBeVisible();
  await expect(page.getByTestId("auditee-scope")).toContainText("Fly Namibia");
  await expect(page.locator("body")).not.toContainText("SkyCargo Air");
  await expect(page.locator("body")).not.toContainText("Internal CAA Note");
  await page.reload();
  await expect(page.getByRole("heading", { name: "Corrective Actions (CAP)" })).toBeVisible();
  await expect(page.getByTestId("auditee-scope")).toContainText("Fly Namibia");
  await expect(page.getByLabel("Root cause", { exact: true })).toBeVisible();
  expect(auditeeTransportPayloads.join("\n")).not.toMatch(
    /SkyCargo Air|internalCaaNote|Internal CAA Note|Inspector workload|internal risk|enforcement deliberation|reviewerPrivate/i,
  );
  await page
    .getByLabel("Root cause", { exact: true })
    .fill(
      "Pre-flight cabin equipment serviceability checks did not reconcile the PBE position with the deferred defect list.",
    );
  await page
    .getByLabel("Corrective action", { exact: true })
    .fill(
      "Replace or service the affected PBE, update the cabin defect record, and confirm serviceability before release.",
    );
  await page
    .getByLabel("Preventive action", { exact: true })
    .fill(
      "Add a supervisor review of emergency equipment checks and monthly sampling of PBE serviceability records.",
    );
  await page.getByLabel("Responsible person", { exact: true }).fill("Fly Namibia Cabin Safety Manager");
  await page.getByLabel("Target completion date", { exact: true }).fill("2026-07-15");
  await page.getByLabel("Comment to CAA", { exact: true }).fill("CAP submitted for CAA review.");
  await page.getByRole("button", { name: "Submit CAP" }).click();
  await expect(page.getByTestId("finding-status")).toContainText("CAP_SUBMITTED");
  await expect(page.getByTestId("auditee-cap-revision-row")).toHaveCount(1);
  await page.reload();
  await expect(page.getByTestId("finding-status")).toContainText("CAP_SUBMITTED");
  await expect(page.getByTestId("auditee-cap-revision-row")).toHaveCount(1);
  await expect(page.getByRole("table", { name: "CAP revision history" })).toContainText(
    "Pre-flight cabin equipment serviceability checks",
  );
  await expect(page.getByRole("button", { name: /Accept CAP/i })).toHaveCount(0);
  captureAuditeeTransport = false;
  await page.getByRole("button", { name: "Switch to CAA CAP Review" }).click();

  await expect(page.getByRole("heading", { level: 1, name: /^CAP Review/ })).toBeVisible();
  await page
    .getByLabel("Comment to Auditee")
    .fill("Clarify how the PBE position records will be reconciled and sampled.");
  await page
    .getByLabel("Internal CAA Note")
    .fill("Revision 1 lacks the private sampling assessment.");
  await page.getByRole("button", { name: "Request More Information" }).click();
  await expect(page.getByTestId("finding-status")).toHaveText("CAP_MORE_INFORMATION_REQUESTED");

  await page.getByRole("link", { name: "Switch role" }).click();
  captureAuditeeTransport = true;
  await page.getByRole("link", { name: /Auditee/i }).click();
  await expect(page.getByTestId("finding-status")).toContainText("CAP_MORE_INFORMATION_REQUESTED");
  await expect(page.getByText("Clarify how the PBE position records will be reconciled and sampled.").first()).toBeVisible();
  await expect(page.locator("body")).not.toContainText("Revision 1 lacks the private sampling assessment.");
  await page
    .getByRole("textbox", { name: "Root cause", exact: true })
    .fill("Revised root cause reconciles the PBE position with the deferred defect list and sampling controls.");
  await page
    .getByRole("textbox", { name: "Corrective action", exact: true })
    .fill("Service the affected PBE, reconcile its position, and update the cabin defect record before release.");
  await page
    .getByRole("textbox", { name: "Preventive action", exact: true })
    .fill("Add supervisor reconciliation and monthly signed sampling of PBE position and serviceability records.");
  await page.getByRole("textbox", { name: "Responsible person", exact: true }).fill("Fly Namibia Cabin Safety Manager");
  await page.locator(".auditee-cap-form input[type='date']").fill("2026-07-20");
  await page.getByRole("textbox", { name: "Comment to CAA", exact: true }).fill("Revised CAP submitted with the requested sampling detail.");
  await page.getByRole("button", { name: "Submit CAP" }).click();
  await expect(page.getByTestId("finding-status")).toContainText("CAP_SUBMITTED");
  await expect(page.getByTestId("auditee-cap-revision-row")).toHaveCount(2);
  const auditeeCapHistory = page.getByRole("table", { name: "CAP revision history" });
  await expect(auditeeCapHistory).toContainText("Pre-flight cabin equipment serviceability checks");
  await expect(auditeeCapHistory).toContainText("Revised root cause reconciles the PBE position");
  await page.reload();
  await expect(page.getByTestId("auditee-cap-revision-row")).toHaveCount(2);
  expect(auditeeTransportPayloads.join("\n")).not.toMatch(
    /SkyCargo Air|internalCaaNote|Internal CAA Note|Inspector workload|internal risk|enforcement deliberation|reviewerPrivate/i,
  );

  captureAuditeeTransport = false;
  await page.getByRole("button", { name: "Switch to CAA CAP Review" }).click();
  await expect(page.getByTestId("cap-review-target")).toContainText("Revision 2");
  await page
    .getByLabel("Comment to Auditee")
    .fill("CAP accepted. Submit the required PBE serviceability record.");
  await page
    .getByLabel("Internal CAA Note")
    .fill("CAP revision 2 is credible; Evidence verification remains required.");
  await page.getByRole("button", { name: "Accept CAP" }).click();
  await expect(page.getByTestId("finding-status")).toHaveText("EVIDENCE_REQUIRED");
  await expect(page.getByTestId("closure-state")).toHaveText("Finding remains open");

  if (testInfo.project.name === "http") {
    const decide = async (
      subjectId: string,
      operationId: string,
      expectedReportVersionRevision: number,
    ) => {
      const response = await request.post(
        `${apiURL}/v1/report-versions/RPT-CAB-2026-001-V1/decisions`,
        {
          headers: {
            "x-avia-test-token": testToken,
            "x-avia-test-subject": subjectId,
          },
          data: {
            operationId,
            reportVersionId: "RPT-CAB-2026-001-V1",
            expectedReportVersionRevision,
            decision: "FORWARD",
            reason: "Advance the exact Final Report through its authorized review chain.",
          },
        },
      );
      expect(response.ok()).toBe(true);
    };
    await decide("USR-MANAGER-NORA", "OP-CANONICAL-FINAL-DM-FORWARD", 1);
    await decide("USR-GM-OMAR", "OP-CANONICAL-FINAL-GM-FORWARD", 2);
  }
  await page.getByRole("button", { name: "Check report authority" }).click();
  await expect(page.getByRole("heading", { name: "Executive Director Dashboard" })).toBeVisible();
  await page.getByRole("button", { name: /Review report/i }).click();
  await page.getByLabel("Report decision reason").fill("Issue the exact candidate report version.");
  await page.getByRole("button", { name: "Issue and lock report" }).click();
  await expect(page.getByTestId("report-status")).toHaveText("LOCKED");
  await expect(page.getByTestId("report-finding-status")).toHaveText(
    "No Findings linked — relationship unavailable for RPT-CAB-2026-001-V1",
  );
  await expect(page.getByText("Report issue did not close the separately created Finding")).toBeVisible();

  await page.getByLabel("Experience").selectOption("manager");
  await expect(page.getByRole("heading", { name: "Department Manager Dashboard" })).toBeVisible();
  await page.getByRole("button", { name: "Use authorized closure" }).click();
  await expect(page.getByRole("alert")).toHaveText("Authorized closure reason is required.");
  await expect(page.getByTestId("manager-canonical-status")).toHaveText("EVIDENCE_REQUIRED");
  await page.getByRole("link", { name: "Return to Fly Namibia Evidence" }).click();
  captureAuditeeTransport = true;

  await submitEvidence(
    page,
    1,
    "Fly_Namibia_PBE_Serviceability_Record_CAB-2026-001.pdf",
  );
  captureAuditeeTransport = false;
  await page.getByRole("button", { name: "Switch to Evidence Review" }).click();
  await expect(page.getByTestId("reviewing-evidence-version")).toHaveText("Version 1");
  await recordEvidenceDecision(
    page,
    "PARTIALLY_CLOSE",
    "The serviceability record is accepted; provide cabin position confirmation.",
    "Version 1 covers serviceability but not accessibility.",
  );
  await expect(page.getByTestId("finding-status")).toHaveText(
    "EVIDENCE_MORE_INFORMATION_REQUESTED",
  );
  await expect(page.getByTestId("closure-state")).toHaveText("Finding remains open");

  captureAuditeeTransport = true;
  await page.getByRole("button", { name: "Return to Auditee Evidence" }).click();
  await submitEvidence(
    page,
    2,
    "Fly_Namibia_PBE_Position_Confirmation_CAB-2026-001.pdf",
  );
  captureAuditeeTransport = false;
  await page.getByRole("button", { name: "Switch to Evidence Review" }).click();
  await expect(page.getByTestId("reviewing-evidence-version")).toHaveText("Version 2");
  await recordEvidenceDecision(
    page,
    "NOT_CLOSE",
    "The position image is insufficient; provide the signed cabin record.",
    "Version 2 lacks the signed accountable record.",
  );
  await expect(page.getByTestId("finding-status")).toHaveText(
    "EVIDENCE_MORE_INFORMATION_REQUESTED",
  );
  await expect(page.getByTestId("closure-state")).toHaveText("Finding remains open");

  captureAuditeeTransport = true;
  await page.getByRole("button", { name: "Return to Auditee Evidence" }).click();
  await submitEvidence(page, 3, "Fly_Namibia_PBE_Signed_Record_CAB-2026-001.pdf");
  captureAuditeeTransport = false;
  await page.getByRole("button", { name: "Switch to Evidence Review" }).click();
  await expect(page.getByTestId("reviewing-evidence-version")).toHaveText("Version 3");
  await recordEvidenceDecision(
    page,
    "CLOSE",
    "Evidence accepted and verified.",
    "Version 3 completes the configured PBE verification basis.",
  );
  await expect(page.getByTestId("finding-status")).toHaveText("CLOSED");
  await expect(page.getByTestId("closure-state")).toHaveText("Finding closed");
  await expect(page.getByTestId("closure-basis")).toHaveText("EVIDENCE_VERIFIED");
  await expect(page.getByTestId("evidence-history-row")).toHaveCount(3);

  await page.getByRole("button", { name: "Open updated Manager Dashboard" }).click();
  await expect(page.getByTestId("manager-closed-findings")).toHaveText("1");
  await expect(page.getByTestId("manager-canonical-status")).toHaveText("CLOSED");
  await page.goto("/department-manager/reports/RPT-CAB-2026-001-V1");
  await expect(page.getByRole("heading", { name: "Reports Approval" })).toBeVisible();
  await expect(page.getByTestId("report-status")).toHaveText("LOCKED");
  await page.getByRole("tab", { name: "Findings" }).click();
  await expect(page.getByRole("tabpanel")).toContainText("No Finding is included.");
  await expect(page.getByTestId("report-finding-status")).toHaveCount(0);

  captureAuditeeTransport = true;
  await page.getByRole("link", { name: "View as Fly Namibia Auditee" }).click();
  await expect(page.getByTestId("finding-status")).toContainText("CLOSED");
  await expect(page.getByTestId("evidence-version-count")).toHaveText("3");
  const finalAuditeeBody = await page.locator("body").innerText();
  const auditeeForbiddenDataPresent = /SkyCargo Air|Internal CAA Note|Inspector workload|internal risk|enforcement deliberation/i.test(
    finalAuditeeBody,
  );
  expect(auditeeForbiddenDataPresent).toBe(false);
  expect(auditeeTransportPayloads.join("\n")).not.toMatch(
    /SkyCargo Air|internalCaaNote|Internal CAA Note|Inspector workload|internal risk|enforcement deliberation|reviewerPrivate/i,
  );

  const transcript: CanonicalScenarioTranscript = {
    auditId: "AUD-2026-001",
    questionId: "CAB-EMEQ-PBE-001",
    potentialFindingId: "PF-2026-001",
    findingNumber: "CAB-2026-001",
    afterConversion: "WAITING_FOR_CAP",
    afterCapSubmission: "CAP_SUBMITTED",
    afterCapAcceptance: "EVIDENCE_REQUIRED",
    afterPartialClose: "EVIDENCE_MORE_INFORMATION_REQUESTED",
    afterNotClose: "EVIDENCE_MORE_INFORMATION_REQUESTED",
    afterEvidenceClose: "CLOSED",
    closureBasis: "EVIDENCE_VERIFIED",
    evidenceVersions: 3,
    reportStatus: "LOCKED",
    reportFindingStatus: "EVIDENCE_REQUIRED",
    managerClosedFindings: 1,
    auditeeForbiddenDataPresent,
  };
  expectCanonicalScenarioTranscript(transcript);
  await testInfo.attach("canonical-scenario-transcript", {
    body: JSON.stringify(transcript, null, 2),
    contentType: "application/json",
  });
  expect(consoleIssues).toEqual([]);
});
