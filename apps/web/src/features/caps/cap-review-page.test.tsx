// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { MemoryRouter } from "react-router-dom";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { completeMockChecklist } from "../../mock/test-checklist-fixtures";
import { CapReviewPage } from "./cap-review-page";

afterEach(cleanup);

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

async function seedFinding(runtime: MockRuntime) {
  const inspector = runtime.backendForRole("inspector");
  const packageView = await inspector.inspections.getPackage({ packageId: "PKG-CAB-2026-001" });
  const response = await inspector.inspections.upsertChecklistResponse({
    operationId: "OP-TEST-RESPONSE",
    responseId: "RESP-CAB-EMEQ-PBE-001",
    auditId: "AUD-2026-001",
    questionId: "CAB-EMEQ-PBE-001",
    expectedResponseRevision: null,
    answer: "NON_COMPLIANT",
    comment: "PBE serviceability and accessibility could not be confirmed.",
  });
  const potentialFinding = await inspector.potentialFindings.create({
    operationId: "OP-TEST-PF",
    auditId: "AUD-2026-001",
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
    operationId: "OP-TEST-CHECKLIST",
    auditId: packageView.auditId,
    expectedChecklistRevision: packageView.checklistRevision,
  });
  const result = await runtime.backendForRole("leadInspector").potentialFindings.decide({
    operationId: "OP-TEST-CONVERT",
    potentialFindingId: potentialFinding.id,
    expectedPotentialFindingRevision: potentialFinding.revision,
    decision: "CONVERT",
    severity: "LEVEL_1_CRITICAL",
    capRequired: true,
    evidenceRequired: true,
    dueDate: "2026-07-15",
  });
  if (!result.finding) throw new Error("Expected seeded Finding.");
  return result.finding;
}

async function seedTwoCapRevisions(runtime: MockRuntime) {
  const lead = runtime.backendForRole("leadInspector");
  const auditee = runtime.backendForRole("auditee");
  let finding = await seedFinding(runtime);
  const first = await auditee.caps.submit({
    operationId: "OP-TEST-CAP-R1",
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
    operationId: "OP-TEST-CAP-R1-REVIEW",
    capRevisionId: first.capRevisionId,
    expectedCapRevision: first.capRevision,
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    decision: "REQUEST_MORE_INFORMATION",
    commentToAuditee: "Clarify how PBE position records will be sampled.",
    internalCaaNote: "Internal CAA note for revision 1.",
  });
  finding = await lead.findings.get({ findingId: finding.id });
  const second = await auditee.caps.submit({
    operationId: "OP-TEST-CAP-R2",
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    rootCause: "Revised root cause with record reconciliation.",
    correctiveAction: "Replace affected PBE and update the cabin defect record.",
    preventiveAction: "Add supervisor review and monthly sampling.",
    responsiblePerson: "Fly Namibia Cabin Safety Manager",
    targetCompletionDate: "2026-07-20",
    commentToCaa: "Revised CAP submitted for CAA review.",
  });
  return { finding: await lead.findings.get({ findingId: finding.id }), second };
}

function renderPage(runtime: MockRuntime) {
  render(
    <AppProviders
      runtime={{
        backend: runtime.backend,
        backendForRole: runtime.backendForRole,
        buildProfile: "demo",
        environmentLabel: "test",
        identityMode: "demo-role-switch",
        subjectId: "USR-LEAD-CANER",
      }}
    >
      <ScenarioProvider>
        <MemoryRouter initialEntries={["/lead-inspector/cap-review/FND-CAB-2026-001"]}>
          <CapReviewPage />
        </MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
}

describe("CapReviewPage", () => {
  it("direct-loads Finding plus immutable CAP revisions and reviews the latest without projection.capSubmission", async () => {
    const runtime = createMockBackendRuntime();
    await seedTwoCapRevisions(runtime);
    const leadCaps = runtime.backendForRole("leadInspector").caps;
    const listSpy = vi.spyOn(leadCaps, "listRevisions");
    const getSpy = vi.spyOn(leadCaps, "getRevision");

    renderPage(runtime);

    const history = await screen.findByRole("table", { name: "CAP revision history" });
    expect(listSpy).toHaveBeenCalledWith({ findingId: "FND-CAB-2026-001" });
    expect(getSpy).toHaveBeenCalledWith({ capRevisionId: "CAP-CAB-2026-001-R2" });
    expect(within(history).getAllByTestId("cap-revision-row")).toHaveLength(2);
    expect(within(history).getByText("Initial root cause retained for immutable history.")).toBeVisible();
    expect(
      within(history).getByText(/Clarify how PBE position records will be sampled\./),
    ).toBeVisible();
    expect(within(history).getByText("Internal CAA note for revision 1.")).toBeVisible();

    const target = screen.getByTestId("cap-review-target");
    expect(within(target).getByText("Revision 2")).toBeVisible();
    expect(within(target).getByText("Revised root cause with record reconciliation.")).toBeVisible();

    await userEvent.click(screen.getByRole("button", { name: "Request More Information" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Comment to Auditee is required");

    await userEvent.clear(screen.getByLabelText("Comment to Auditee"));
    await userEvent.type(
      screen.getByLabelText("Comment to Auditee"),
      "CAP accepted. Submit the required PBE serviceability record.",
    );
    await userEvent.clear(screen.getByLabelText("Internal CAA Note"));
    await userEvent.type(
      screen.getByLabelText("Internal CAA Note"),
      "CAP actions are credible; Evidence verification remains required.",
    );
    await userEvent.click(screen.getByRole("button", { name: "Accept CAP" }));

    expect(await screen.findByTestId("finding-status")).toHaveTextContent("EVIDENCE_REQUIRED");
    expect(screen.getByTestId("closure-state")).toHaveTextContent("Finding remains open");
    expect(screen.getByText("CAP accepted; Evidence remains required before closure.")).toBeVisible();
  });
});
