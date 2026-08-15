// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";
import "fake-indexeddb/auto";

import { MemoryRouter, Route, Routes } from "react-router-dom";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { ChecklistRunnerPage } from "./checklist-runner-page";

afterEach(cleanup);

function renderPage() {
  const runtime = createMockBackendRuntime();
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
        <MemoryRouter initialEntries={["/inspector/audits/AUD-2026-001/checklist?packageId=PKG-CAB-2026-001"]}>
          <Routes>
            <Route path="/inspector/audits/:auditId/checklist" element={<ChecklistRunnerPage />} />
          </Routes>
        </MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
}

describe("ChecklistRunnerPage", () => {
  it("preserves question references, required comment behavior, attachment filename, and handoff boundary", async () => {
    renderPage();

    expect(await screen.findAllByTestId("checklist-question-row")).toHaveLength(6);
    await userEvent.click(screen.getByRole("button", { name: "Open question 4" }));
    const responsePanel = screen.getByTestId("checklist-response-panel");
    expect(within(responsePanel).getAllByText("Configured Cabin Inspection reference — EM EQ / PBE").length).toBeGreaterThan(0);
    expect(within(responsePanel).getAllByText("PBE serviceability record and cabin position confirmation").length).toBeGreaterThan(0);
    expect(within(responsePanel).getByText(/Comments required for Non Compliant and Observation/i)).toBeVisible();

    await userEvent.selectOptions(screen.getByLabelText("Checklist answer"), "NON_COMPLIANT");
    await userEvent.click(screen.getByRole("button", { name: "Save response" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Inspector comment is required");

    await userEvent.type(
      screen.getByLabelText("Inspector comment"),
      "PBE serviceability and accessibility could not be confirmed.",
    );
    await userEvent.click(screen.getByRole("button", { name: "Save response" }));
    await waitFor(() => expect(screen.getByTestId("response-status")).toHaveTextContent("Server acknowledged"));

    await userEvent.click(screen.getByRole("button", { name: "Create Potential Finding" }));
    expect(await screen.findByTestId("potential-finding-id")).toHaveTextContent("PF-2026-001");
    expect(screen.queryByRole("link", { name: "Switch to Lead Inspector" })).toBeNull();

    const attachment = new File(["inspection bytes"], "pbe-serviceability.pdf", {
      type: "application/pdf",
    });
    await userEvent.upload(screen.getByLabelText("Attachment file"), attachment);
    expect(screen.getByTestId("selected-inspection-attachment")).toHaveTextContent("pbe-serviceability.pdf");
    expect(screen.getByTestId("local-server-state")).toHaveTextContent("Server acknowledged");
  });
});
