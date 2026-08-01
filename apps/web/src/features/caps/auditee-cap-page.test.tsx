// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import type { FindingView } from "../../backend/backend";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { completeMockChecklist } from "../../mock/test-checklist-fixtures";
import { AuditeeCapPage } from "./auditee-cap-page";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

async function seedFinding(runtime: MockRuntime): Promise<FindingView> {
  const inspector = runtime.backendForRole("inspector");
  const packageView = await inspector.inspections.getPackage({ packageId: "PKG-CAB-2026-001" });
  const response = await inspector.inspections.upsertChecklistResponse({
    operationId: "OP-AUDITEE-TEST-RESPONSE",
    responseId: "RESP-CAB-EMEQ-PBE-001",
    auditId: packageView.auditId,
    questionId: "CAB-EMEQ-PBE-001",
    expectedResponseRevision: null,
    answer: "NON_COMPLIANT",
    comment: "PBE serviceability and accessibility could not be confirmed.",
  });
  const potentialFinding = await inspector.potentialFindings.create({
    operationId: "OP-AUDITEE-TEST-PF",
    auditId: packageView.auditId,
    questionId: "CAB-EMEQ-PBE-001",
    checklistResponseId: response.id,
    expectedChecklistResponseRevision: response.revision,
    title: "PBE serviceability and accessibility not confirmed",
    description: "The configured cabin check could not confirm PBE serviceability.",
    requiredComment: response.comment,
    inspectionAttachmentIds: [],
  });
  await completeMockChecklist(runtime, packageView.id);
  await inspector.inspections.submitChecklist({
    operationId: "OP-AUDITEE-TEST-CHECKLIST",
    auditId: packageView.auditId,
    expectedChecklistRevision: packageView.checklistRevision,
  });
  const result = await runtime.backendForRole("leadInspector").potentialFindings.decide({
    operationId: "OP-AUDITEE-TEST-CONVERT",
    potentialFindingId: potentialFinding.id,
    expectedPotentialFindingRevision: potentialFinding.revision,
    decision: "CONVERT",
    severity: "LEVEL_1_CRITICAL",
    capRequired: true,
    evidenceRequired: true,
    dueDate: "2026-07-15",
  });
  if (!result.finding) throw new Error("Expected canonical Finding.");
  return result.finding;
}

async function seedAcceptedSecondCap(runtime: MockRuntime): Promise<FindingView> {
  const lead = runtime.backendForRole("leadInspector");
  const auditee = runtime.backendForRole("auditee");
  let finding = await seedFinding(runtime);
  const first = await auditee.caps.submit({
    operationId: "OP-AUDITEE-TEST-CAP-R1",
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    rootCause: "Initial root cause retained for immutable history.",
    correctiveAction: "Initial corrective action.",
    preventiveAction: "Initial preventive action.",
    responsiblePerson: "Fly Namibia Cabin Safety Manager",
    targetCompletionDate: "2026-07-15",
    commentToCaa: "Initial CAP submitted for CAA review.",
  });
  finding = await lead.findings.get({ findingId: finding.id });
  await lead.caps.review({
    operationId: "OP-AUDITEE-TEST-CAP-R1-REVIEW",
    capRevisionId: first.capRevisionId,
    expectedCapRevision: first.capRevision,
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    decision: "REQUEST_MORE_INFORMATION",
    commentToAuditee: "Clarify how PBE position records will be sampled.",
    internalCaaNote: "Private CAA assessment must never enter the Auditee shape.",
  });
  finding = await auditee.findings.get({ findingId: finding.id });
  const second = await auditee.caps.submit({
    operationId: "OP-AUDITEE-TEST-CAP-R2",
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    rootCause: "Revised root cause with record reconciliation.",
    correctiveAction: "Replace affected PBE and update the cabin defect record.",
    preventiveAction: "Add supervisor review and monthly sampling.",
    responsiblePerson: "Fly Namibia Cabin Safety Manager",
    targetCompletionDate: "2026-07-20",
    commentToCaa: "Revised CAP submitted for CAA review.",
  });
  finding = await lead.findings.get({ findingId: finding.id });
  await lead.caps.review({
    operationId: "OP-AUDITEE-TEST-CAP-R2-REVIEW",
    capRevisionId: second.capRevisionId,
    expectedCapRevision: second.capRevision,
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    decision: "ACCEPT",
    commentToAuditee: "CAP accepted. Submit the required PBE Evidence.",
    internalCaaNote: "Private acceptance assessment.",
  });
  return auditee.findings.get({ findingId: finding.id });
}

async function seedEvidence(runtime: MockRuntime, finding: FindingView): Promise<void> {
  const auditee = runtime.backendForRole("auditee");
  const fileName = "Fly_Namibia_PBE_Serviceability_Record_CAB-2026-001.pdf";
  const upload = await auditee.evidence.beginUpload({
    operationId: "OP-AUDITEE-TEST-EVIDENCE-BEGIN",
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    fileName,
    declaredMediaType: "application/pdf",
    byteSize: 12,
    sha256: "sha256:auditee-test",
  });
  await auditee.evidence.completeUpload({
    operationId: "OP-AUDITEE-TEST-EVIDENCE-COMPLETE",
    uploadId: upload.uploadId,
    byteSize: 12,
    sha256: "sha256:auditee-test",
  });
}

function renderPage(runtime: MockRuntime, identityMode: "demo-role-switch" | "oidc-session" = "demo-role-switch") {
  return render(
    <AppProviders
      runtime={{
        backend: runtime.backend,
        backendForRole: runtime.backendForRole,
        buildProfile: identityMode === "oidc-session" ? "http" : "demo",
        environmentLabel: "test",
        identityMode,
        subjectId: "USR-AUDITEE-FLY-NAMIBIA",
      }}
    >
      <ScenarioProvider>
        <MemoryRouter initialEntries={["/auditee/service-provider-cap"]}>
          <AuditeeCapPage />
        </MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
}

describe("AuditeeCapPage", () => {
  it("direct-loads own-org Findings, immutable CAP revisions, public CAA comments, and Evidence history", async () => {
    const runtime = createMockBackendRuntime();
    const finding = await seedAcceptedSecondCap(runtime);
    await seedEvidence(runtime, finding);
    await expect(
      runtime.backendForRole("auditee").findings.get({ findingId: "FND-SKYCARGO-2026-099" }),
    ).rejects.toThrow(/unavailable to this Auditee organization/i);

    renderPage(runtime);

    const findings = await screen.findByRole("table", { name: "My Findings" });
    expect(within(findings).getByText("CAB-2026-001")).toBeVisible();
    const selectedRow = within(findings).getByText("CAB-2026-001").closest("tr");
    if (!selectedRow) throw new Error("Expected selected Auditee Finding row.");
    expect(within(selectedRow).getByRole("button", { name: "CAB-2026-001" })).toHaveAttribute("aria-pressed", "true");
    expect(within(selectedRow).getByRole("button", { name: "CAB-2026-001 is already selected" })).toBeDisabled();
    expect(within(selectedRow).getByRole("button", { name: "CAB-2026-001 is already selected" })).toHaveAttribute(
      "title",
      "CAB-2026-001 is already open in the Auditee Finding dossier.",
    );
    expect(screen.getByTestId("auditee-scope")).toHaveTextContent("Fly Namibia");
    const history = await screen.findByRole("table", { name: "CAP revision history" });
    expect(within(history).getAllByTestId("auditee-cap-revision-row")).toHaveLength(2);
    expect(within(history).getByText("Initial root cause retained for immutable history.")).toBeVisible();
    expect(within(history).getByText("Revised root cause with record reconciliation.")).toBeVisible();
    expect(within(history).getByText("Clarify how PBE position records will be sampled.")).toBeVisible();
    expect(screen.getByText("Fly_Namibia_PBE_Serviceability_Record_CAB-2026-001.pdf")).toBeVisible();
    expect(document.body).not.toHaveTextContent(
      /Private CAA assessment|Private acceptance assessment|Internal CAA Note|SkyCargo Air|Inspector workload|enforcement/i,
    );
  });

  it("submits every CAP field, hydrates the immutable revision, and uses a bounded role handoff", async () => {
    const runtime = createMockBackendRuntime();
    await seedFinding(runtime);
    renderPage(runtime);
    const user = userEvent.setup();

    await screen.findByRole("table", { name: "My Findings" });
    await user.type(screen.getByLabelText("Root cause", { exact: true }), "Complete root cause.");
    await user.type(screen.getByLabelText("Corrective action", { exact: true }), "Complete corrective action.");
    await user.type(screen.getByLabelText("Preventive action", { exact: true }), "Complete preventive action.");
    await user.type(screen.getByLabelText("Responsible person", { exact: true }), "Fly Namibia Safety Manager");
    await user.type(screen.getByLabelText("Target completion date", { exact: true }), "2026-07-15");
    await user.type(screen.getByLabelText("Comment to CAA", { exact: true }), "Complete public submission comment.");
    await user.click(screen.getByRole("button", { name: "Submit CAP" }));

    expect(await screen.findByTestId("finding-status")).toHaveTextContent("CAP_SUBMITTED");
    const history = await screen.findByRole("table", { name: "CAP revision history" });
    expect(within(history).getByText("Complete root cause.")).toBeVisible();
    expect(screen.getByRole("button", { name: "Switch to CAA CAP Review" })).toBeEnabled();

    cleanup();
    renderPage(runtime, "oidc-session");
    expect(await screen.findByRole("button", { name: "Switch to CAA CAP Review" })).toBeDisabled();
    expect(screen.getByText(/session does not include Lead Inspector authority/i)).toBeVisible();
  });

  it("labels deterministic demo Evidence truthfully and exposes only the selected filename", async () => {
    const runtime = createMockBackendRuntime();
    await seedAcceptedSecondCap(runtime);
    renderPage(runtime);
    const user = userEvent.setup();

    const input = await screen.findByLabelText("Deterministic demo Evidence file");
    const file = new File(["demo evidence"], "PBE-record.pdf", { type: "application/pdf" });
    await user.upload(input, file);

    expect(screen.getByTestId("selected-evidence-file")).toHaveTextContent("PBE-record.pdf");
    expect(document.body).not.toHaveTextContent("Mock Evidence file");
  });

  it("keeps official HTTP Evidence submission online-first", async () => {
    const runtime = createMockBackendRuntime();
    await seedAcceptedSecondCap(runtime);
    vi.spyOn(window.navigator, "onLine", "get").mockReturnValue(false);

    renderPage(runtime, "oidc-session");

    const input = await screen.findByLabelText("Evidence file");
    expect(input).toBeDisabled();
    expect(screen.getByRole("button", { name: "Submit Evidence version" })).toBeDisabled();
    expect(screen.getByRole("status")).toHaveTextContent(
      "Official Evidence submission requires an online connection.",
    );
    expect(document.body).not.toHaveTextContent(/Deterministic demo|Mock Evidence file/);
  });

  it("hands pending Evidence review to the declared Department Manager route authority", async () => {
    const runtime = createMockBackendRuntime();
    const finding = await seedAcceptedSecondCap(runtime);
    await seedEvidence(runtime, finding);

    renderPage(runtime, "oidc-session");

    expect(await screen.findByRole("button", { name: "Switch to Evidence Review" })).toBeDisabled();
    expect(screen.getByText(/session does not include Department Manager authority/i)).toBeVisible();
  });
});
