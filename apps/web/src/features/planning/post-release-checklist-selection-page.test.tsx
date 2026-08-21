// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { PostReleaseChecklistSelectionPage } from "./post-release-checklist-selection-page";

beforeEach(() => localStorage.clear());
afterEach(cleanup);

async function createReleasedProposal(runtime: ReturnType<typeof createMockBackendRuntime>): Promise<string> {
  const manager = runtime.backendForRole("manager");
  const proposal = manager.planningProposal!;
  const scope = (await proposal.listScopeOptions({ limit: 10 })).items[0]!;
  const location = (await proposal.listLocations({ organizationId: scope.organizationId, regulatedTargetId: scope.regulatedTargetId }))[0]!;
  const estimate = await proposal.getWorkloadEstimate({
    operationId: "TEST-ESTIMATE",
    idempotencyKey: "TEST-ESTIMATE",
    organizationId: scope.organizationId,
    providerScopeId: scope.providerScopeId,
    regulatedTargetId: scope.regulatedTargetId,
    inspectionType: "RAMP_INSPECTION",
  });
  const draft = await proposal.createDraft({
    operationId: "TEST-DRAFT",
    idempotencyKey: "TEST-DRAFT",
    values: {
      organizationId: scope.organizationId,
      providerScopeId: scope.providerScopeId,
      regulatedTargetId: scope.regulatedTargetId,
      inspectionType: "RAMP_INSPECTION",
      purpose: "Confirm post-release package preparation boundary.",
      plannedDate: "2026-12-10",
      mode: "On-site",
      locationInput: { kind: "CANONICAL", locationId: location.id },
      requiredInspectorCount: 2,
      estimatedChecklistItemCount: 10,
      workloadEstimateId: estimate.estimateId,
      workloadEstimateDigest: estimate.estimateDigest,
      requestedBudget: 0,
      currency: "NAD",
    },
  });
  const submitted = await proposal.submit({ operationId: "TEST-SUBMIT", idempotencyKey: "TEST-SUBMIT", draftId: draft.id, expectedRevision: draft.revision });
  let item = submitted.planningItem;
  item = await runtime.backendForRole("finance").planning.decide({ operationId: "TEST-FINANCE", planningItemId: item.id, expectedPlanningRevision: item.revision, decision: "APPROVE_BUDGET", reason: "Budget boundary verified.", expectedSubmittedScopeSnapshotId: "", expectedPlanningSnapshotDigest: item.planningSnapshotDigest });
  item = await runtime.backendForRole("gm").planning.decide({ operationId: "TEST-GM", planningItemId: item.id, expectedPlanningRevision: item.revision, decision: "FORWARD_FOR_FINAL_APPROVAL", reason: "Operational review complete.", expectedSubmittedScopeSnapshotId: "", expectedPlanningSnapshotDigest: item.planningSnapshotDigest });
  item = await runtime.backendForRole("executiveDirector").planning.decide({ operationId: "TEST-EXECUTIVE", planningItemId: item.id, expectedPlanningRevision: item.revision, decision: "APPROVE_PLAN", reason: "Final approval complete.", expectedSubmittedScopeSnapshotId: "", expectedPlanningSnapshotDigest: item.planningSnapshotDigest });
  item = await runtime.backendForRole("gm").planning.decide({ operationId: "TEST-RELEASE", planningItemId: item.id, expectedPlanningRevision: item.revision, decision: "RELEASE_PLAN", reason: "Release boundary complete.", expectedSubmittedScopeSnapshotId: "", expectedPlanningSnapshotDigest: item.planningSnapshotDigest });
  return item.id;
}

function renderPage(runtime: ReturnType<typeof createMockBackendRuntime>, planningItemId: string) {
  render(
    <AppProviders runtime={{ backend: runtime.backend, backendForRole: runtime.backendForRole, buildProfile: "demo", environmentLabel: "test", identityMode: "demo-role-switch", subjectId: "USR-MANAGER-NORA" }}>
      <ScenarioProvider>
        <MemoryRouter initialEntries={[`/department-manager/planning/${planningItemId}/setup/checklist`]}>
          <Routes><Route path="/department-manager/planning/:planningItemId/setup/checklist" element={<PostReleaseChecklistSelectionPage />} /></Routes>
        </MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
}

describe("post-release checklist preparation", () => {
  it("opens only after release and finalizes a bounded exact selection", async () => {
    const runtime = createMockBackendRuntime();
    const planningItemId = await createReleasedProposal(runtime);
    const user = userEvent.setup();
    renderPage(runtime, planningItemId);

    expect(await screen.findByRole("heading", { level: 1, name: "Checklist selection" })).toBeVisible();
    await screen.findByRole("heading", { level: 2, name: "Checklist items" });
    const firstCheckbox = await screen.findByRole("checkbox", { name: "Select SYNTH-FORM-001 item 1" });
    await user.click(firstCheckbox);
    await user.type(screen.getByLabelText("Search checklist items"), "SYNTH-FORM-001");
    await user.click(screen.getByRole("button", { name: "Apply filters" }));
    await waitFor(() => expect(screen.getByRole("checkbox", { name: "Select SYNTH-FORM-001 item 1" })).toBeChecked());
    await user.click(screen.getByRole("button", { name: "Confirm selection" }));
    await waitFor(() => expect(screen.getByText(/Selection confirmed by the server/)).toBeVisible());
    await user.click(screen.getByRole("button", { name: "Finalize Audit package" }));
    await waitFor(() => expect(screen.getByText(/Audit-package scope finalized/)).toBeVisible());
    expect(screen.getByText("FINALIZED")).toBeVisible();
    expect(screen.getByRole("checkbox", { name: "Select SYNTH-FORM-001 item 1" })).toBeDisabled();
  });
});
