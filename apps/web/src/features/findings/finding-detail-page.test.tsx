// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { MemoryRouter, Route, Routes } from "react-router-dom";
import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { completeMockChecklist } from "../../mock/test-checklist-fixtures";
import { FindingDetailPage } from "./finding-detail-page";

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

function renderPage(runtime: MockRuntime, findingId: string) {
  render(
    <AppProviders
      runtime={{
        backend: runtime.backend,
        backendForRole: runtime.backendForRole,
        buildProfile: "demo",
        environmentLabel: "test",
        identityMode: "demo-role-switch",
        subjectId: "USR-INSPECTOR-AMINA",
      }}
    >
      <ScenarioProvider>
        <MemoryRouter initialEntries={[`/inspector/findings/${findingId}`]}>
          <Routes>
            <Route path="/inspector/findings/:findingId" element={<FindingDetailPage />} />
          </Routes>
        </MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
}

describe("FindingDetailPage", () => {
  it("does not fabricate the CAB dossier from an unrelated Finding", async () => {
    const runtime = createMockBackendRuntime();

    renderPage(runtime, "FND-UNKNOWN-TEST");

    expect(await screen.findByRole("heading", { name: "Finding unavailable" })).toBeVisible();
    expect(screen.queryByText(/Finding CAB-2026-011/)).toBeNull();
    expect((await runtime.backendForRole("inspector").findings.get({ findingId: "FND-SKYCARGO-2026-099" })).findingNumber).toBe("CAR-2026-099");
  });

  it("direct-loads ui-audit-009 as a CAA Inspector dossier with source-role ownership", async () => {
    const runtime = createMockBackendRuntime();
    const finding = await seedFinding(runtime);

    renderPage(runtime, finding.id);

    const dossier = await screen.findByTestId("finding-dossier");
    expect(within(dossier).getAllByText(finding.findingNumber).length).toBeGreaterThanOrEqual(1);
    expect(within(dossier).getByText("CAA Inspector")).toBeVisible();
    expect(within(dossier).queryByText("Lead Inspector")).toBeNull();
    expect(within(dossier).getAllByText("WAITING_FOR_CAP").length).toBeGreaterThanOrEqual(1);
    for (const expected of [
      "Fly Namibia",
      "AUD-2026-001",
      "Level 1 Critical",
      "Overdue: 15 Jul 2026",
      /Non-Compliant response and required Inspector comment/,
      "Auditee to submit CAP",
    ]) {
      expect(within(dossier).getByText(expected)).toBeVisible();
    }
    expect(within(dossier).getByText(/Configured Cabin Inspection reference/)).toBeVisible();
    expect(screen.getByRole("list", { name: "Finding lifecycle" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Switch to Fly Namibia Auditee" })).toBeVisible();
    expect(screen.queryByRole("link", { name: "Switch to Fly Namibia Auditee" })).toBeNull();
  });

  it("offers the Inspector Assistant context link and an explicit Lead-authority CAP handoff", async () => {
    const runtime = createMockBackendRuntime();
    const finding = await seedFinding(runtime);
    await runtime.backendForRole("auditee").caps.submit({
      operationId: "OP-TEST-CAP-SUBMIT",
      findingId: finding.id,
      expectedFindingRevision: finding.revision,
      rootCause: "The equipment record and register were maintained separately.",
      correctiveAction: "Reconcile the sampled equipment record.",
      preventiveAction: "Use one controlled register.",
      responsiblePerson: "Fly Namibia Cabin Safety Manager",
      targetCompletionDate: "2026-07-15",
      commentToCaa: "Ready for Lead review.",
    });
    renderPage(runtime, finding.id);

    const dossier = await screen.findByTestId("finding-dossier");
    expect(within(dossier).getByRole("link", { name: "Open Inspector Assistant" })).toHaveAttribute(
      "href",
      `/inspector/assistant?findingId=${finding.id}`,
    );
    expect(within(dossier).queryByRole("button", { name: "Review CAP" })).toBeNull();
    expect(within(dossier).getByText("Review CAP requires Lead Inspector authority.")).toBeVisible();
    expect(within(dossier).getByRole("button", { name: "Switch to Lead Inspector for CAP Review" })).toBeVisible();
  });
});
