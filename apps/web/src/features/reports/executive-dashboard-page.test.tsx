// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { ExecutiveDashboardPage } from "./executive-dashboard-page";

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

afterEach(cleanup);

function renderPage(runtime: MockRuntime) {
  render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: "demo",
      environmentLabel: "test",
      identityMode: "demo-role-switch",
      subjectId: "USR-ED-UFUK",
    }}>
      <MemoryRouter initialEntries={["/executive-director/dashboard"]}><ExecutiveDashboardPage /></MemoryRouter>
    </AppProviders>,
  );
}

describe("ExecutiveDashboardPage", () => {
  it("loads the six-KPI decision surface from real planning, report, Finding, and organization queries", async () => {
    const runtime = createMockBackendRuntime();
    renderPage(runtime);
    expect(await screen.findByRole("heading", { name: "Executive Director Dashboard" })).toBeVisible();
    expect(screen.getByRole("region", { name: "Executive overview" })).toBeVisible();
    expect(screen.getByRole("region", { name: "Planning approvals" })).toBeVisible();
    expect(screen.getByRole("region", { name: "Final Report approvals" })).toBeVisible();
    expect(screen.getByRole("region", { name: "Department overview" })).toBeVisible();
    expect(screen.getByRole("region", { name: "Overdue actions" })).toBeVisible();
    expect(screen.getByText(/No automatic enforcement or closure decision/i)).toBeVisible();
    expect(document.body).not.toHaveTextContent(/demo approval|mock mutation|localStorage/i);
  });

  it("opens and issues the exact server-owned Final Report without closing a linked Finding", async () => {
    const runtime = createMockBackendRuntime();
    const reports = (await runtime.backendForRole("executiveDirector").documents.list({})).items.filter((item) => item.kind === "REPORT");
    const report = reports.find((candidate) => candidate.id.startsWith("RPT-"));
    if (!report) throw new Error("Expected a server-owned Final Report document.");
    const reportVersion = await runtime.backendForRole("executiveDirector").reports.getVersion({ reportVersionId: report.id });
    renderPage(runtime);
    const page = await screen.findByRole("heading", { name: "Executive Director Dashboard" });
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: `Review report ${reportVersion.reportVersionId}` }));
    const decision = await screen.findByRole("region", { name: "Executive report decision" });
    expect(within(decision).getByText(reportVersion.reportVersionId)).toBeVisible();
    if (reportVersion.status !== "LOCKED") {
      await user.type(within(decision).getByLabelText("Report decision reason"), "Issue the exact Final Report version.");
      await user.click(within(decision).getByRole("button", { name: "Issue and lock report" }));
      await expect.poll(async () => (await runtime.backendForRole("executiveDirector").reports.getVersion({ reportVersionId: reportVersion.reportVersionId })).status).toBe("LOCKED");
    }
    expect(decision).toHaveTextContent(/Report issue did not close the separately created Finding|No Findings linked|OPEN|CLOSED/i);
    expect(page).not.toHaveTextContent("Fixed CAB report");
  });

  it("does not synthesize an Executive-owned planning revision when the queue is empty", async () => {
    const runtime = createMockBackendRuntime();
    vi.spyOn(runtime.backendForRole("executiveDirector").planning, "list").mockResolvedValue({ items: [], nextCursor: null });
    renderPage(runtime);
    const queue = await screen.findByRole("region", { name: "Planning approvals" });
    expect(within(queue).getByText("No plans require an Executive Director decision.")).toBeVisible();
    expect(within(queue).queryByRole("button")).toBeNull();
  });
});
