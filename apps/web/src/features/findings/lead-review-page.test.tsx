// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { MemoryRouter } from "react-router-dom";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import type { Role } from "../../backend/backend";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { completeMockChecklist } from "../../mock/test-checklist-fixtures";
import { LeadReviewPage } from "./lead-review-page";

afterEach(cleanup);

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

async function seedPotentialFinding(runtime: MockRuntime) {
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
  return potentialFinding;
}

function renderPage(
  runtime: MockRuntime,
  backendForRole = runtime.backendForRole,
) {
  render(
    <AppProviders
      runtime={{
        backend: runtime.backend,
        backendForRole,
        buildProfile: "demo",
        environmentLabel: "test",
        identityMode: "demo-role-switch",
        subjectId: "USR-LEAD-CANER",
      }}
    >
      <ScenarioProvider>
        <MemoryRouter initialEntries={["/lead-inspector/lead-review"]}>
          <LeadReviewPage />
        </MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
}

describe("LeadReviewPage", () => {
  it("direct-loads the persisted Lead queue, requires reasons, and converts with explicit CAP/Evidence requirements", async () => {
    const runtime = createMockBackendRuntime();
    await seedPotentialFinding(runtime);
    const leadBackend = runtime.backendForRole("leadInspector");
    const listSpy = vi.spyOn(leadBackend.potentialFindings, "list");
    const getSpy = vi.spyOn(leadBackend.potentialFindings, "get");

    renderPage(runtime);

    const queue = await screen.findByRole("table", {
      name: "Potential Findings awaiting Lead review",
    });
    expect(listSpy).toHaveBeenCalledWith({ status: "PENDING_LEAD_REVIEW" });
    expect(getSpy).toHaveBeenCalledWith({ potentialFindingId: "PF-2026-001" });
    expect(within(queue).getByRole("cell", { name: "PF-2026-001" })).toBeVisible();
    expect(within(queue).getByText("PENDING LEAD REVIEW")).toBeVisible();

    await userEvent.click(screen.getByRole("button", { name: "Return Potential Finding" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Lead decision reason is required");

    await userEvent.selectOptions(screen.getByLabelText("Finding severity"), "LEVEL_2_MAJOR");
    expect(screen.getByLabelText("CAP required")).toBeChecked();
    expect(screen.getByLabelText("Evidence required")).toBeChecked();
    await userEvent.click(screen.getByRole("button", { name: "Convert to Finding" }));

    const decision = await screen.findByTestId("lead-decision-result");
    expect(within(decision).getByTestId("finding-number")).toHaveTextContent("CAB-2026-001");
    expect(within(decision).getByText("CAP required")).toBeVisible();
    expect(within(decision).getByText("Evidence required")).toBeVisible();
  });

  it("does not expose Lead decision controls through forbidden role backends", async () => {
    const runtime = createMockBackendRuntime();
    await seedPotentialFinding(runtime);
    await expect(
      runtime.backendForRole("auditee" as Role).potentialFindings.list({
        status: "PENDING_LEAD_REVIEW",
      }),
    ).rejects.toThrow("Lead Inspector authority is required.");
  });

  it("applies no-CAP no-Evidence no-Due-Date defaults when Observation is selected", async () => {
    const runtime = createMockBackendRuntime();
    await seedPotentialFinding(runtime);
    const leadBackend = runtime.backendForRole("leadInspector");
    const decideSpy = vi.spyOn(leadBackend.potentialFindings, "decide");

    renderPage(runtime);
    await screen.findByTestId("potential-finding-dossier");
    await userEvent.selectOptions(screen.getByLabelText("Finding severity"), "OBSERVATION");

    expect(screen.getByLabelText("CAP required")).not.toBeChecked();
    expect(screen.getByLabelText("Evidence required")).not.toBeChecked();
    expect(screen.getByLabelText("Finding Due Date")).toHaveValue("");

    await userEvent.click(screen.getByRole("button", { name: "Convert to Finding" }));

    expect(decideSpy).toHaveBeenLastCalledWith({
      operationId: "OP-PF-CONVERT",
      potentialFindingId: "PF-2026-001",
      expectedPotentialFindingRevision: 1,
      decision: "CONVERT",
      severity: "OBSERVATION",
      capRequired: false,
      evidenceRequired: false,
      dueDate: null,
    });
    const decision = await screen.findByTestId("lead-decision-result");
    expect(within(decision).getByText("CAP not required")).toBeVisible();
    expect(within(decision).getByText("Evidence not required")).toBeVisible();
  });

  it("allows the Lead to explicitly configure an Observation Due Date", async () => {
    const runtime = createMockBackendRuntime();
    await seedPotentialFinding(runtime);
    const leadBackend = runtime.backendForRole("leadInspector");
    const decideSpy = vi.spyOn(leadBackend.potentialFindings, "decide");

    renderPage(runtime);
    await screen.findByTestId("potential-finding-dossier");
    await userEvent.selectOptions(screen.getByLabelText("Finding severity"), "OBSERVATION");
    const dueDate = screen.getByLabelText("Finding Due Date");
    expect(dueDate).not.toHaveAttribute("readonly");
    await userEvent.type(dueDate, "2026-08-01");
    await userEvent.click(screen.getByRole("button", { name: "Convert to Finding" }));

    expect(decideSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({
        severity: "OBSERVATION",
        dueDate: "2026-08-01",
      }),
    );
  });

  it("keeps a successful conversion visible when the runtime creates role adapters per call", async () => {
    const runtime = createMockBackendRuntime();
    await seedPotentialFinding(runtime);
    const backendForRole: MockRuntime["backendForRole"] = (role) => ({
      ...runtime.backendForRole(role),
    });

    renderPage(runtime, backendForRole);

    await screen.findByTestId("potential-finding-dossier");
    await userEvent.click(screen.getByRole("button", { name: "Convert to Finding" }));

    expect(await screen.findByTestId("finding-number")).toHaveTextContent("CAB-2026-001");
    expect(screen.queryByText("No authorized persisted Potential Findings awaiting Lead review")).not.toBeInTheDocument();
  });
});
