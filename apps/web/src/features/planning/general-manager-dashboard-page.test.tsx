// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { GeneralManagerDashboardPage } from "./planning-workspaces";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

function renderPage(runtime: MockRuntime, identityMode: "demo-role-switch" | "oidc-session" = "demo-role-switch") {
  return render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: identityMode === "oidc-session" ? "http" : "demo",
      environmentLabel: "test",
      identityMode,
      subjectId: "USR-GM-OKAN",
    }}>
      <MemoryRouter initialEntries={["/general-manager/gm-dashboard"]}>
        <GeneralManagerDashboardPage />
      </MemoryRouter>
    </AppProviders>,
  );
}

async function advancePlanToGeneralManager(runtime: MockRuntime) {
  const finance = runtime.backendForRole("finance");
  const item = (await finance.planning.list({ limit: 20 })).items[0]!;
  return finance.planning.decide({
    operationId: "OP-GM-TEST-FINANCE-APPROVE",
    planningItemId: item.id,
    expectedPlanningRevision: item.revision,
    decision: "APPROVE_BUDGET",
    reason: "Budget reviewed for GM test.",
  });
}

describe("GeneralManagerDashboardPage", () => {
  it("direct-loads the distinct GM KPI, department, heat-map, and report-review hierarchy", async () => {
    const runtime = createMockBackendRuntime();
    renderPage(runtime);

    expect(await screen.findByRole("heading", { name: "General Manager Dashboard" })).toBeVisible();
    const indicators = screen.getByRole("region", { name: "General Manager indicators" });
    expect(within(indicators).getAllByRole("article")).toHaveLength(5);
    expect(screen.getByRole("table", { name: "Department Overview" })).toBeVisible();
    expect(screen.getByRole("region", { name: "Risk Heat Map" })).toBeVisible();
    expect(screen.getByRole("table", { name: "Report Review Queue" })).toBeVisible();
    expect(screen.getByRole("link", { name: "View All Departments" })).toHaveAttribute(
      "href",
      "/general-manager/departments",
    );
    expect(screen.getByRole("button", { name: "Open report RPT-CAB-2026-001-V1 unavailable" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Open report RPT-CAB-2026-001-V1 unavailable" })).toHaveAttribute(
      "title",
      "Report version RPT-CAB-2026-001-V1 is EXECUTIVE_DIRECTOR_REVIEW; General Manager can open exact report decisions only at GM_REVIEW.",
    );
    expect(screen.getByText(/General Manager review may return or forward/i)).toBeVisible();
    expect(screen.getByText(/cannot issue, sign, lock, or close/i)).toBeVisible();
    expect(screen.queryByRole("button", { name: /issue|sign|lock|close Finding/i })).toBeNull();
  });

  it("renders a truthful empty report queue when connected storage has no report versions", async () => {
    const runtime = createMockBackendRuntime();
    const gm = runtime.backendForRole("gm");
    vi.spyOn(gm.documents, "list").mockResolvedValue({ items: [], nextCursor: null });
    const getVersion = vi.spyOn(gm.reports, "getVersion").mockRejectedValue(new Error("Not found."));

    renderPage(runtime);

    expect(await screen.findByText("No report versions are available yet.")).toBeVisible();
    expect(screen.queryByRole("alert")).toBeNull();
    expect(getVersion).not.toHaveBeenCalled();
  });

  it("forwards only a GM-owned planning revision and bounds the Executive Director handoff", async () => {
    const runtime = createMockBackendRuntime();
    await advancePlanToGeneralManager(runtime);
    renderPage(runtime);
    const user = userEvent.setup();

    const decision = await screen.findByRole("region", { name: "General Manager planning decision" });
    expect(within(decision).getByTestId("planning-status")).toHaveTextContent("GM_REVIEW");
    expect(within(decision).getByTestId("planning-owner")).toHaveTextContent("General Manager");
    expect(within(decision).getByText("Revision 2")).toBeVisible();
    await user.click(within(decision).getByRole("button", { name: "Forward to Executive Director" }));
    await user.click(within(decision).getByRole("button", { name: "Confirm General Manager Decision" }));
    expect(screen.getByRole("alert")).toHaveTextContent(/decision reason is required/i);

    await user.type(within(decision).getByLabelText("General Manager decision reason"), "Operational scope reviewed.");
    await user.click(within(decision).getByRole("button", { name: "Confirm General Manager Decision" }));
    expect(await within(decision).findByTestId("planning-status")).toHaveTextContent("EXECUTIVE_DIRECTOR_REVIEW");
    expect(within(decision).getByRole("button", { name: "Continue as Executive Director" })).toBeEnabled();

    cleanup();
    renderPage(runtime, "oidc-session");
    expect(await screen.findByRole("button", { name: "Continue as Executive Director" })).toBeDisabled();
    expect(screen.getByText(/session does not include Executive Director authority/i)).toBeVisible();
  });

  it("cannot use report issue authority at the current immutable report stage", async () => {
    const runtime = createMockBackendRuntime();
    const report = await runtime.backendForRole("gm").reports.getVersion({ reportVersionId: "RPT-CAB-2026-001-V1" });
    await expect(runtime.backendForRole("gm").reports.decide({
      operationId: "OP-GM-ILLEGAL-ISSUE",
      reportVersionId: report.reportVersionId,
      expectedReportVersionRevision: report.revision,
      decision: "ISSUE_AND_LOCK",
      reason: "GM must not issue this report.",
    })).rejects.toThrow(/cannot perform|General Manager|stage/i);
  });
});
