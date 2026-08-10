// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import type { FindingView } from "../../backend/backend";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { completeMockChecklist } from "../../mock/test-checklist-fixtures";
import { ExecutiveDashboardPage } from "./executive-dashboard-page";

afterEach(cleanup);

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

function renderPage(runtime: MockRuntime) {
  return render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: "demo",
      environmentLabel: "test",
      identityMode: "demo-role-switch",
      subjectId: "USR-ED-UFUK",
    }}>
      <MemoryRouter initialEntries={["/executive-director/executive-dashboard"]}>
        <ExecutiveDashboardPage />
      </MemoryRouter>
    </AppProviders>,
  );
}

async function seedCanonicalFinding(runtime: MockRuntime): Promise<FindingView> {
  const inspector = runtime.backendForRole("inspector");
  const packageView = await inspector.inspections.getPackage({ packageId: "PKG-CAB-2026-001" });
  const response = await inspector.inspections.upsertChecklistResponse({
    operationId: "OP-EXEC-RESPONSE",
    responseId: "RESP-CAB-EMEQ-PBE-001",
    auditId: packageView.auditId,
    questionId: "CAB-EMEQ-PBE-001",
    expectedResponseRevision: null,
    answer: "NON_COMPLIANT",
    comment: "PBE serviceability could not be confirmed.",
  });
  const potential = await inspector.potentialFindings.create({
    operationId: "OP-EXEC-PF",
    auditId: packageView.auditId,
    questionId: "CAB-EMEQ-PBE-001",
    checklistResponseId: response.id,
    expectedChecklistResponseRevision: response.revision,
    title: "PBE serviceability not confirmed",
    description: "The cabin check could not confirm PBE serviceability.",
    requiredComment: response.comment,
    inspectionAttachmentIds: [],
  });
  await completeMockChecklist(runtime, packageView.id);
  await inspector.inspections.submitChecklist({
    operationId: "OP-EXEC-SUBMIT",
    auditId: packageView.auditId,
    expectedChecklistRevision: packageView.checklistRevision,
  });
  const result = await runtime.backendForRole("leadInspector").potentialFindings.decide({
    operationId: "OP-EXEC-CONVERT",
    potentialFindingId: potential.id,
    expectedPotentialFindingRevision: potential.revision,
    decision: "CONVERT",
    severity: "LEVEL_2_MAJOR",
    capRequired: true,
    evidenceRequired: true,
    dueDate: "2026-07-15",
  });
  if (!result.finding) throw new Error("Expected canonical Finding.");
  return result.finding;
}

async function advancePlanToExecutive(runtime: MockRuntime) {
  const finance = runtime.backendForRole("finance");
  const initial = (await finance.planning.list({ limit: 20 })).items[0]!;
  const atGm = await finance.planning.decide({
    operationId: "OP-EXEC-FINANCE-APPROVE",
    planningItemId: initial.id,
    expectedPlanningRevision: initial.revision,
    decision: "APPROVE_BUDGET",
    reason: "Budget approved for Executive test.",
  });
  return runtime.backendForRole("gm").planning.decide({
    operationId: "OP-EXEC-GM-FORWARD",
    planningItemId: atGm.id,
    expectedPlanningRevision: atGm.revision,
    decision: "FORWARD_FOR_FINAL_APPROVAL",
    reason: "GM forwarded the plan for final approval.",
  });
}

describe("ExecutiveDashboardPage", () => {
  it("direct-loads the distinct six-KPI, two-queue, and three-context Executive hierarchy", async () => {
    const runtime = createMockBackendRuntime();
    renderPage(runtime);

    expect(await screen.findByRole("heading", { name: "Executive Director Dashboard" })).toBeVisible();
    expect(screen.getByText("Final authorized demo approval")).toBeVisible();
    const overview = screen.getByRole("region", { name: "Executive overview" });
    expect(within(overview).getAllByRole("article")).toHaveLength(6);
    expect(screen.getByRole("region", { name: "Planning approvals" })).toBeVisible();
    expect(screen.getByRole("region", { name: "Final Report approvals" })).toBeVisible();
    expect(screen.getByRole("region", { name: "Department overview" })).toBeVisible();
    expect(screen.getByRole("region", { name: "Overdue actions" })).toBeVisible();
    expect(screen.getByRole("heading", { name: "Management indicator only" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Review report RPT-CAB-2026-001-V1" })).toBeEnabled();
    expect(screen.getByRole("link", { name: "View all Planning" })).toHaveAttribute(
      "href",
      "/executive-director/planning",
    );
    expect(screen.getByRole("link", { name: "View all Final Reports" })).toHaveAttribute(
      "href",
      "/executive-director/final-reports",
    );
  });

  it("renders truthful empty report state when the connected catalog has no report versions", async () => {
    const runtime = createMockBackendRuntime();
    const executive = runtime.backendForRole("executiveDirector");
    vi.spyOn(executive.documents, "list").mockResolvedValue({ items: [], nextCursor: null });
    const getVersion = vi.spyOn(executive.reports, "getVersion").mockRejectedValue(new Error("Not found."));

    renderPage(runtime);

    expect(await screen.findByRole("heading", { name: "Executive Director Dashboard" })).toBeVisible();
    expect(await screen.findByText("No Final Report versions are available yet.")).toBeVisible();
    expect(screen.queryByRole("alert")).toBeNull();
    expect(getVersion).not.toHaveBeenCalled();
  });

  it("issues and locks the immutable unlinked report without closing a separately created Finding", async () => {
    const runtime = createMockBackendRuntime();
    const finding = await seedCanonicalFinding(runtime);
    renderPage(runtime);
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: "Review report RPT-CAB-2026-001-V1" }));
    const decision = screen.getByRole("region", { name: "Executive report decision" });
    await user.click(within(decision).getByRole("button", { name: "Issue and lock report" }));
    expect(screen.getByRole("alert")).toHaveTextContent(/decision reason is required/i);
    await user.type(within(decision).getByLabelText("Report decision reason"), "Final report reviewed and authorized.");
    await user.click(within(decision).getByRole("button", { name: "Issue and lock report" }));

    expect(await screen.findByTestId("report-status")).toHaveTextContent("LOCKED");
    expect(screen.getByTestId("report-finding-status")).toHaveTextContent("No Findings linked — relationship unavailable for RPT-CAB-2026-001-V1");
    expect(screen.getByText("Report issue did not close the separately created Finding")).toBeVisible();
    expect((await runtime.backendForRole("leadInspector").findings.get({ findingId: finding.id })).status).toBe(finding.status);
    const locked = await runtime.backendForRole("executiveDirector").reports.getVersion({ reportVersionId: "RPT-CAB-2026-001-V1" });
    expect(locked).toMatchObject({ status: "LOCKED", revision: 2, findingIds: [] });
    expect(within(decision).queryByRole("button", { name: "Issue and lock report" })).toBeNull();
  });

  it("approves only an Executive-owned plan revision and hands release back to General Manager", async () => {
    const runtime = createMockBackendRuntime();
    await advancePlanToExecutive(runtime);
    renderPage(runtime);
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: "Review plan PLAN-2026-CAB-001" }));
    const decision = screen.getByRole("region", { name: "Executive plan decision" });
    expect(within(decision).getByTestId("planning-status")).toHaveTextContent("EXECUTIVE_DIRECTOR_REVIEW");
    expect(within(decision).getByText("Revision 3")).toBeVisible();
    await user.click(within(decision).getByRole("button", { name: "Approve Plan" }));
    expect(screen.getByRole("alert")).toHaveTextContent(/decision reason is required/i);
    await user.type(within(decision).getByLabelText("Plan decision reason"), "Final plan authority recorded.");
    await user.click(within(decision).getByRole("button", { name: "Approve Plan" }));

    expect(await within(decision).findByTestId("planning-status")).toHaveTextContent("GM_RELEASE");
    expect(within(decision).getByRole("button", { name: "Continue as General Manager" })).toBeEnabled();
  });
});
