// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { AppRouter } from "../../app/router";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;
type GovernanceRole = "gm" | "executiveDirector";

const subjects: Record<GovernanceRole, string> = {
  gm: "USR-GM-OMAR",
  executiveDirector: "USR-ED-ZARA",
};

afterEach(cleanup);

function renderRoute(path: string, role: GovernanceRole, runtime: MockRuntime = createMockBackendRuntime()) {
  render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: "demo",
      environmentLabel: "test",
      identityMode: "demo-role-switch",
      subjectId: subjects[role],
    }}>
      <ScenarioProvider>
        <MemoryRouter initialEntries={[path]}><AppRouter /></MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
  return runtime;
}

async function movePlanToGmReview(runtime: MockRuntime) {
  const finance = runtime.backendForRole("finance");
  const item = (await finance.planning.list({ limit: 50 })).items.find((candidate) => candidate.status === "FINANCE_REVIEW");
  if (!item) throw new Error("Expected a Finance-owned planning item.");
  return finance.planning.decide({
    operationId: `TEST-FINANCE-${item.id}-${item.revision}`,
    planningItemId: item.id,
    expectedPlanningRevision: item.revision,
    decision: "APPROVE_BUDGET",
    reason: "Budget review completed for the exact server-owned planning revision.",
  });
}

async function moveReportToGmReview(runtime: MockRuntime) {
  const manager = runtime.backendForRole("manager");
  const report = await manager.reports.getVersion({ reportVersionId: "PR-2026-018-V1" });
  if (report.status !== "DEPARTMENT_REVIEW") throw new Error(`Expected Department Manager report review, got ${report.status}.`);
  return manager.reports.decide({
    operationId: `TEST-MANAGER-${report.reportVersionId}-${report.revision}`,
    reportVersionId: report.reportVersionId,
    expectedReportVersionRevision: report.revision,
    decision: "FORWARD",
    reason: "Department Manager forwarded the exact immutable Preliminary Report version.",
  });
}

describe("General Manager and Executive Director workspaces", () => {
  it.each([
    ["/general-manager/planning", "gm", "gm-planning-page", "Planning"],
    ["/general-manager/report-approvals", "gm", "gm-report-approvals-page", "Report Approvals"],
    ["/general-manager/risk-dashboard", "gm", "gm-risk-dashboard-page", "Cross-Department Risk Dashboard"],
    ["/executive-director/planning", "executiveDirector", "executive-planning-page", "Planning"],
    ["/executive-director/preliminary-reports", "executiveDirector", "executive-preliminary-reports-page", "Preliminary Reports"],
    ["/executive-director/final-reports", "executiveDirector", "executive-final-reports-page", "Final Reports"],
  ] as const)("direct-loads %s through the declared role route", async (path, role, testId, heading) => {
    renderRoute(path, role);
    const page = await screen.findByTestId(testId);
    expect(await within(page).findByRole("heading", { level: 1, name: heading })).toBeVisible();
    expect(screen.getByTestId("application-shell")).toHaveAttribute("data-active-role", role);
    expect(screen.queryByTestId("route-pending-implementation")).toBeNull();
  });

  it("selects a server-owned Planning revision before the GM command and preserves the next owner", async () => {
    const runtime = createMockBackendRuntime();
    const atGm = await movePlanToGmReview(runtime);
    renderRoute("/general-manager/planning", "gm", runtime);
    const page = await screen.findByTestId("gm-planning-page");
    const user = userEvent.setup();
    await user.click(within(page).getByRole("button", { name: `Review ${atGm.id}` }));
    const selected = await within(page).findByRole("region", { name: `Selected plan ${atGm.id}` });
    expect(within(selected).getByTestId("planning-status")).toHaveTextContent("GM_REVIEW");
    await user.type(within(page).getByLabelText("General Manager decision reason"), "Forwarding the exact reviewed plan.");
    await user.click(within(page).getByRole("button", { name: `Forward ${atGm.id} to Executive Director` }));
    await expect.poll(async () => (await runtime.backendForRole("gm").planning.list({ limit: 50 })).items.find((item) => item.id === atGm.id)?.status).toBe("EXECUTIVE_DIRECTOR_REVIEW");
    expect(page).not.toHaveTextContent(/automatic.*closure|localStorage|demo candidate/i);
  });

  it("moves an exact report version through GM review without granting issue or lock authority", async () => {
    const runtime = createMockBackendRuntime();
    const atGm = await moveReportToGmReview(runtime);
    renderRoute("/general-manager/report-approvals", "gm", runtime);
    const page = await screen.findByTestId("gm-report-approvals-page");
    const user = userEvent.setup();
    await user.click(within(page).getByRole("button", { name: `Open ${atGm.reportVersionId}` }));
    const dossier = await within(page).findByRole("region", { name: `Selected report ${atGm.reportVersionId}` });
    expect(within(dossier).getByTestId("report-status")).toHaveTextContent("GM_REVIEW");
    await user.type(within(dossier).getByLabelText("General Manager report decision reason"), "Forwarding the exact report version.");
    await user.click(within(dossier).getByRole("button", { name: `Forward ${atGm.reportVersionId} to Executive Director` }));
    await expect.poll(async () => (await runtime.backendForRole("gm").reports.getVersion({ reportVersionId: atGm.reportVersionId })).status).toBe("EXECUTIVE_DIRECTOR_REVIEW");
    expect(within(dossier).queryByRole("button", { name: /issue|lock/i })).toBeNull();
  });

  it("lets Executive Director issue and lock only the exact Executive-owned report version", async () => {
    const runtime = createMockBackendRuntime();
    const atGm = await moveReportToGmReview(runtime);
    const atExecutive = await runtime.backendForRole("gm").reports.decide({
      operationId: `TEST-GM-${atGm.reportVersionId}-${atGm.revision}`,
      reportVersionId: atGm.reportVersionId,
      expectedReportVersionRevision: atGm.revision,
      decision: "FORWARD",
      reason: "General Manager forwarded the exact immutable report version.",
    });
    renderRoute("/executive-director/preliminary-reports", "executiveDirector", runtime);
    const page = await screen.findByTestId("executive-preliminary-reports-page");
    const user = userEvent.setup();
    await user.click(within(page).getByRole("button", { name: `Open ${atExecutive.reportVersionId}` }));
    const dossier = await within(page).findByRole("region", { name: `Selected Preliminary Report ${atExecutive.reportVersionId}` });
    await user.type(within(dossier).getByLabelText("Executive Director report decision reason"), "Issuing the exact report version.");
    await user.click(within(dossier).getByRole("button", { name: `Issue and lock ${atExecutive.reportVersionId}` }));
    await expect.poll(async () => (await runtime.backendForRole("executiveDirector").reports.getVersion({ reportVersionId: atExecutive.reportVersionId })).status).toBe("LOCKED");
  });
});
