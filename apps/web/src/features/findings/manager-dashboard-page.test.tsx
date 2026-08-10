// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { ManagerDashboardPage } from "./manager-dashboard-page";

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
        subjectId: "USR-MANAGER-MEHMET",
      }}
    >
      <ScenarioProvider>
        <MemoryRouter initialEntries={["/department-manager/dashboard"]}>
          <ManagerDashboardPage />
        </MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
  return runtime;
}

describe("ManagerDashboardPage", () => {
  it("direct-loads the accepted attention hierarchy with advisory management indicators", async () => {
    const runtime = renderPage();

    expect(await screen.findByRole("heading", { name: "Department Manager Dashboard" })).toBeVisible();
    const indicators = screen.getByRole("region", { name: "Management indicators" });
    expect(within(indicators).getAllByRole("article")).toHaveLength(6);
    expect(screen.getByRole("heading", { name: "What needs attention?" })).toBeVisible();
    expect(screen.getByRole("table", { name: "Priority Findings" })).toBeVisible();
    expect(screen.getByRole("table", { name: "Upcoming Surveillance" })).toBeVisible();
    expect(screen.getByText(/Oversight Health Index is advisory/i)).toBeVisible();
    expect(screen.getByText(/does not trigger automatic legal, enforcement, certificate, or Finding-closure decisions/i)).toBeVisible();

    await expect(runtime.backendForRole("finance").dashboards.getManagerProjection({})).rejects.toThrow(
      /management authority/i,
    );
  });

  it("keeps the manager workspace controls on real registered routes", async () => {
    renderPage();
    await screen.findByRole("heading", { name: "What needs attention?" });

    expect(screen.getByRole("link", { name: "Open Planning" })).toHaveAttribute(
      "href",
      "/department-manager/audit-plan",
    );
    expect(screen.getByRole("link", { name: "Open Organizations" })).toHaveAttribute(
      "href",
      "/department-manager/organizations",
    );
    expect(screen.getByRole("link", { name: "Open Audits" })).toHaveAttribute(
      "href",
      "/department-manager/audits",
    );
    expect(screen.getByRole("link", { name: "Open Inspection Team" })).toHaveAttribute(
      "href",
      "/department-manager/inspection-team",
    );
    expect(screen.getByRole("link", { name: "Open Findings Review" })).toHaveAttribute(
      "href",
      "/department-manager/findings-review",
    );
    expect(screen.getByRole("link", { name: "Open CAP Monitoring" })).toHaveAttribute(
      "href",
      "/department-manager/cap-monitoring",
    );
    expect(screen.getByRole("link", { name: "Open Checklist Management" })).toHaveAttribute(
      "href",
      "/department-manager/checklist-management",
    );
    expect(screen.getByRole("link", { name: "Open Reports Approval" })).toHaveAttribute(
      "href",
      "/department-manager/preliminary-reports/PR-2026-018",
    );
    expect(screen.getByRole("button", { name: "Risk Dashboard unavailable" })).toHaveAttribute(
      "title",
      "Manager Risk Dashboard has no declared Task 6 route.",
    );
    expect(screen.queryByRole("button", { name: /automatic enforcement/i })).toBeNull();
  });

  it("loads a clean profile without requesting a synthetic fixed report version", async () => {
    const runtime = createMockBackendRuntime();
    const getVersion = vi.spyOn(runtime.backendForRole("manager").reports, "getVersion")
      .mockRejectedValue(new Error("Not found."));

    renderPage(runtime);

    expect(await screen.findByRole("heading", { name: "Department Manager Dashboard" })).toBeVisible();
    expect(await screen.findByText("Reports Awaiting Approval")).toBeVisible();
    expect(screen.queryByRole("alert")).toBeNull();
    expect(getVersion).not.toHaveBeenCalled();
  });
});
