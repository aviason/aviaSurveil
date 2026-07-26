// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { AuditPlanCalendarPage } from "./planning-workspaces";

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
        subjectId: "USR-MANAGER-MEHMET",
      }}
    >
      <MemoryRouter initialEntries={["/department-manager/audit-plan"]}>
        <AuditPlanCalendarPage />
      </MemoryRouter>
    </AppProviders>,
  );
}

describe("AuditPlanCalendarPage", () => {
  it("direct-loads the governed command center and planning queue", async () => {
    renderPage();

    expect(await screen.findByRole("heading", { name: "Department Planning" })).toBeVisible();
    const commandCenter = await screen.findByTestId("planning-command-center");
    expect(within(commandCenter).getAllByText("Finance Review").length).toBeGreaterThan(0);
    expect(within(commandCenter).getByText("Finance to review budget")).toBeVisible();
    expect(within(commandCenter).getByText("15 Jul 2026")).toBeVisible();
    expect(screen.getByRole("list", { name: "Planning decision path" })).toBeVisible();
    expect(screen.getByRole("table", { name: "Planning Queue" })).toBeVisible();
  });

  it("links the supported intake workspaces and marks the already selected queue record unavailable", async () => {
    renderPage();

    const open = await screen.findByRole("button", { name: "PLAN-2026-CAB-001 is already selected" });
    expect(open).toBeDisabled();
    expect(open).toHaveAttribute("title", "PLAN-2026-CAB-001 is already open in the Planning command center.");
    expect(screen.getByTestId("planning-selected-record")).toHaveTextContent("PLAN-2026-CAB-001");
    expect(screen.getByRole("link", { name: "New Inspection planning intake" })).toHaveAttribute("href", "/department-manager/new-audit/step-1");
    expect(screen.getByRole("link", { name: "Open Inspection Package Builder" })).toHaveAttribute("href", "/department-manager/inspection-package-builder");
  });
});
