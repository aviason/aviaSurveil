// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { MemoryRouter } from "react-router-dom";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { BackendHttpError } from "../../backend/http-backend";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { InspectorAssignmentsPage } from "./inspector-assignments-page";

afterEach(cleanup);

function renderPage(runtime = createMockBackendRuntime()) {
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
        <MemoryRouter initialEntries={["/inspector/inspector-assignments"]}>
          <InspectorAssignmentsPage />
        </MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
  return runtime;
}

describe("InspectorAssignmentsPage", () => {
  it("renders the decision-first assignment register and equivalent mobile card fields", async () => {
    const user = userEvent.setup();
    renderPage();

    const register = await screen.findByRole("table", { name: "Assigned Audits" });
    for (const column of ["Audit", "Organization", "Status", "Due Date", "Due state", "Next action"]) {
      expect(within(register).getByRole("columnheader", { name: column })).toBeVisible();
    }
    expect(within(register).getByRole("cell", { name: "AUD-2026-001" })).toBeVisible();
    expect(within(register).getByRole("cell", { name: "Fly Namibia" })).toBeVisible();
    expect(within(register).getByText("Due Soon: 18 Jun 2026")).toBeVisible();
    expect(within(register).getByRole("link", { name: "Open 2026 Cabin Inspection - Fly Namibia" })).toBeVisible();

    const mobileCard = screen.getByRole("article", { name: "AUD-2026-001" });
    expect(within(mobileCard).getByText("Due state")).toBeVisible();
    expect(within(mobileCard).getByText("Continue Cabin Inspection checklist")).toBeVisible();

    const reset = screen.getByRole("button", { name: "Reset assignment filters" });
    expect(reset).toBeDisabled();
    await user.type(screen.getByPlaceholderText("Search audits..."), "Fly");
    expect(reset).toBeEnabled();
    await user.click(reset);
    expect(screen.getByPlaceholderText("Search audits...")).toHaveValue("");
    expect(reset).toBeDisabled();
  });

  it("treats an unavailable local assignment projection as an explicit empty state", async () => {
    const runtime = createMockBackendRuntime();
    vi.spyOn(runtime.backendForRole("inspector").assignments, "list").mockRejectedValue(
      new BackendHttpError("Backend request failed with status 404", 404, null, null, null),
    );
    renderPage(runtime);

    const status = await screen.findByText("Assignments are not provisioned in this local AGA workspace.");
    expect(status).toHaveAttribute("role", "status");
    expect(screen.getByText("No assignments match these filters.")).toBeVisible();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.queryByText(/Backend request failed with status 404/i)).not.toBeInTheDocument();
  });
});
