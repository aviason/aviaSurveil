// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { planningItemLabel, recordReference } from "../shared/record-presentation";
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
  return runtime;
}

describe("AuditPlanCalendarPage", () => {
  it("direct-loads the governed command center and planning queue", async () => {
    const runtime = renderPage();
    const item = (await runtime.backendForRole("manager").planning.list({ limit: 20 })).items[0];
    if (!item) throw new Error("Expected a server-owned planning item.");
    await userEvent.click(await screen.findByRole("button", { name: `Open ${planningItemLabel(item.title, item.id)}` }));

    expect(await screen.findByRole("heading", { name: "Department Planning" })).toBeVisible();
    const commandCenter = await screen.findByTestId("planning-command-center");
    expect(within(commandCenter).getAllByText("Finance Review").length).toBeGreaterThan(0);
    expect(within(commandCenter).getByText(item.nextAction)).toBeVisible();
    expect(within(commandCenter).getByText(item.organizationName)).toBeVisible();
    expect(screen.getByRole("list", { name: "Planning decision path" })).toBeVisible();
    expect(screen.getByRole("table", { name: "Planning Queue" })).toBeVisible();
  });

  it("links the supported intake workspaces and marks the already selected queue record unavailable", async () => {
    const runtime = renderPage();
    const item = (await runtime.backendForRole("manager").planning.list({ limit: 20 })).items[0];
    if (!item) throw new Error("Expected a server-owned planning item.");
    await userEvent.click(await screen.findByRole("button", { name: `Open ${planningItemLabel(item.title, item.id)}` }));

    const open = await screen.findByRole("button", { name: `${planningItemLabel(item.title, item.id)} is already selected` });
    expect(open).toBeDisabled();
    expect(open).toHaveAttribute("title", "This Planning item is already open in the Planning command center.");
    expect(screen.getByTestId("planning-selected-record")).toHaveTextContent(recordReference("Plan", item.id));
    expect(screen.getByRole("link", { name: "New Inspection planning intake" })).toHaveAttribute("href", "/department-manager/new-audit/step-1");
    expect(screen.queryByRole("link", { name: "Open Inspection Package Builder" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "New Inspection planning intake" })).toHaveAttribute("href", "/department-manager/new-audit/step-1");
  });
});
