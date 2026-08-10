// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { AppRouter } from "../../app/router";
import { ScenarioProvider } from "../../app/scenario-context";
import {
  createMockBackendPersistentRuntime,
  createMockBackendRuntime,
} from "../../mock/create-mock-backend";
import { seedVisualRuntimeForPath } from "../../mock/seed-visual-runtime";

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

beforeEach(() => localStorage.clear());
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

const subjectByRole: Record<"gm" | "executiveDirector", string> = {
  gm: "USR-GM-OMAR",
  executiveDirector: "USR-ED-ZARA",
};

function renderGovernanceRoute(
  path: string,
  role: "gm" | "executiveDirector",
  runtime: MockRuntime = createMockBackendRuntime(),
) {
  render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: "demo",
      environmentLabel: "test",
      identityMode: "demo-role-switch",
      subjectId: subjectByRole[role],
    }}>
      <ScenarioProvider>
        <MemoryRouter initialEntries={[path]}><AppRouter /></MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
  return runtime;
}

async function advancePlanTo(runtime: MockRuntime, target: "GM_REVIEW" | "EXECUTIVE_DIRECTOR_REVIEW") {
  const finance = runtime.backendForRole("finance");
  const initial = (await finance.planning.list({ limit: 20 })).items[0]!;
  const atGm = await finance.planning.decide({
    operationId: `TASK8-FINANCE-${target}`,
    planningItemId: initial.id,
    expectedPlanningRevision: initial.revision,
    decision: "APPROVE_BUDGET",
    reason: "Finance approved the exact planning revision for Task 8.",
  });
  if (target === "GM_REVIEW") return atGm;
  return runtime.backendForRole("gm").planning.decide({
    operationId: `TASK8-GM-${target}`,
    planningItemId: atGm.id,
    expectedPlanningRevision: atGm.revision,
    decision: "FORWARD_FOR_FINAL_APPROVAL",
    reason: "General Manager forwarded the exact planning revision.",
  });
}

async function advancePreliminaryReportTo(runtime: MockRuntime, target: "GM_REVIEW" | "EXECUTIVE_DIRECTOR_REVIEW") {
  const manager = runtime.backendForRole("manager");
  const initial = await manager.reports.getVersion({ reportVersionId: "PR-2026-018-V1" });
  const atGm = await manager.reports.decide({
    operationId: `TASK8-MANAGER-PR-${target}`,
    reportVersionId: initial.reportVersionId,
    expectedReportVersionRevision: initial.revision,
    decision: "FORWARD",
    reason: "Department Manager forwarded immutable Preliminary Report version 1.",
  });
  if (target === "GM_REVIEW") return atGm;
  return runtime.backendForRole("gm").reports.decide({
    operationId: `TASK8-GM-PR-${target}`,
    reportVersionId: atGm.reportVersionId,
    expectedReportVersionRevision: atGm.revision,
    decision: "FORWARD",
    reason: "General Manager forwarded immutable Preliminary Report version 1.",
  });
}

describe("General Manager and Executive Director workspaces", () => {
  it("direct-loads all eleven source-role routes and makes every primary route reachable with one active navigation item", async () => {
    const routes = [
      ["/general-manager/planning", "gm", "gm-planning-page", "Planning"],
      ["/general-manager/report-approvals", "gm", "gm-report-approvals-page", "Report Approvals"],
      ["/general-manager/departments", "gm", "gm-departments-page", "Departments"],
      ["/general-manager/risk-dashboard", "gm", "gm-risk-dashboard-page", "Cross-Department Risk Dashboard"],
      ["/general-manager/settings", "gm", "gm-settings-page", "Settings"],
      ["/executive-director/planning", "executiveDirector", "executive-planning-page", "Planning"],
      ["/executive-director/preliminary-reports", "executiveDirector", "executive-preliminary-reports-page", "Preliminary Reports"],
      ["/executive-director/final-reports", "executiveDirector", "executive-final-reports-page", "Final Reports"],
      ["/executive-director/reports/RPT-CAB-2026-001", "executiveDirector", "executive-report-preview-page", "Final Report Preview"],
      ["/executive-director/notifications", "executiveDirector", "executive-notifications-page", "Notifications"],
      ["/executive-director/settings", "executiveDirector", "executive-settings-page", "Settings"],
    ] as const;

    for (const [path, role, testId, heading] of routes) {
      renderGovernanceRoute(path, role);
      const page = await screen.findByTestId(testId);
      expect(within(page).getByRole("heading", { level: 1, name: heading })).toBeVisible();
      expect(page.querySelector(".workbench-page-header"), `${path} must use the shared workbench page header`).not.toBeNull();
      expect(screen.getByTestId("application-shell")).toHaveAttribute("data-active-role", role);
      expect(screen.queryByTestId("route-pending-implementation")).toBeNull();
      const navigation = screen.getByRole("navigation", { name: "Primary role navigation" });
      expect(
        within(navigation).getAllByRole("link").filter((link) => link.hasAttribute("aria-current")),
        `${path} must expose exactly one active primary navigation link`,
      ).toHaveLength(1);
      cleanup();
    }
  });

  it("lets General Manager decide only the exact GM-owned Planning revision and keeps release as a later GM stage", async () => {
    const runtime = createMockBackendRuntime();
    await advancePlanTo(runtime, "GM_REVIEW");
    const user = userEvent.setup();
    renderGovernanceRoute("/general-manager/planning", "gm", runtime);

    const page = await screen.findByTestId("gm-planning-page");
    await within(page).findByTestId("planning-status");
    expect(page).toHaveAttribute("data-planning-item-id", "PLAN-2026-CAB-001");
    expect(page).toHaveAttribute("data-planning-revision", "2");
    expect(within(page).getByTestId("planning-status")).toHaveTextContent("GM_REVIEW");
    expect(within(page).getByRole("button", { name: "PLAN-2026-CAB-001 is already selected" })).toBeDisabled();
    expect(within(page).getByRole("button", { name: "PLAN-2026-CAB-001 is already selected" })).toHaveAttribute(
      "title",
      "PLAN-2026-CAB-001 is already open in the Planning dossier.",
    );
    await user.click(within(page).getByRole("button", { name: "Forward PLAN-2026-CAB-001 to Executive Director" }));
    expect(within(page).getByRole("alert")).toHaveTextContent("General Manager decision reason is required");
    await user.type(within(page).getByLabelText("General Manager decision reason"), "Operational scope and Finance review confirmed.");
    await user.click(within(page).getByRole("button", { name: "Forward PLAN-2026-CAB-001 to Executive Director" }));
    expect(await within(page).findByTestId("planning-status")).toHaveTextContent("EXECUTIVE_DIRECTOR_REVIEW");
    expect(page).toHaveTextContent("General Manager Release remains a separate next stage");
  });

  it("preserves the complete Planning chain, zero-budget Finance gate, exact actors, reasons, revisions, and audit events", async () => {
    const runtime = createMockBackendRuntime();
    const manager = runtime.backendForRole("manager");
    const draft = await manager.planningIntake.getDraft({ draftId: "PLAN-DRAFT-2026-001" });
    const saved = await manager.planningIntake.saveDraft({
      draftId: draft.id,
      expectedRevision: draft.revision,
      idempotencyKey: "TASK8-ZERO-BUDGET-DRAFT",
      values: { ...draft, purpose: "Zero-budget governance test", location: "CAA Office", requestedBudget: 0 },
    });
    const submitted = await manager.planningIntake.submit({
      draftId: saved.id,
      expectedRevision: saved.revision,
      idempotencyKey: "TASK8-ZERO-BUDGET-SUBMIT",
      planningItemId: "PLAN-2026-ZERO-001",
    });
    expect(submitted.planningItem).toMatchObject({ estimatedBudget: 0, status: "FINANCE_REVIEW", currentOwnerRole: "finance", revision: 1 });
    await expect(runtime.backendForRole("gm").planning.decide({
      operationId: "TASK8-ZERO-SKIP-FINANCE",
      planningItemId: submitted.planningItem.id,
      expectedPlanningRevision: submitted.planningItem.revision,
      decision: "FORWARD_FOR_FINAL_APPROVAL",
      reason: "Attempted stage skip.",
    })).rejects.toThrow(/not at General Manager review/i);

    const finance = await runtime.backendForRole("finance").planning.decide({
      operationId: "TASK8-ZERO-FINANCE",
      planningItemId: submitted.planningItem.id,
      expectedPlanningRevision: submitted.planningItem.revision,
      decision: "APPROVE_BUDGET",
      reason: "Zero budget reviewed; Finance stage remains mandatory.",
    });
    const gm = await runtime.backendForRole("gm").planning.decide({
      operationId: "TASK8-ZERO-GM",
      planningItemId: finance.id,
      expectedPlanningRevision: finance.revision,
      decision: "FORWARD_FOR_FINAL_APPROVAL",
      reason: "General Manager reviewed exact revision 2.",
    });
    const executive = await runtime.backendForRole("executiveDirector").planning.decide({
      operationId: "TASK8-ZERO-EXEC",
      planningItemId: gm.id,
      expectedPlanningRevision: gm.revision,
      decision: "APPROVE_PLAN",
      reason: "Executive Director approved exact revision 3.",
    });
    const released = await runtime.backendForRole("gm").planning.decide({
      operationId: "TASK8-ZERO-RELEASE",
      planningItemId: executive.id,
      expectedPlanningRevision: executive.revision,
      decision: "RELEASE_PLAN",
      reason: "General Manager released exact revision 4 to Department.",
    });
    expect(released).toMatchObject({ status: "RELEASED", currentOwnerRole: "manager", revision: 5 });
    const events = await runtime.backendForRole("gm").auditTrail.list({ entityType: "SURVEILLANCE_PLAN", entityId: released.id });
    expect(events.items.map((event) => [event.actorRole, event.actorSubjectId, event.beforeStatus, event.afterStatus, event.reason, event.entityRevision])).toEqual([
      ["manager", "USR-MANAGER-NORA", "DRAFT", "FINANCE_REVIEW", "Routine / Announced; notice advance", 1],
      ["finance", "USR-FINANCE-LINA", "FINANCE_REVIEW", "GM_REVIEW", "Zero budget reviewed; Finance stage remains mandatory.", 2],
      ["gm", "USR-GM-OMAR", "GM_REVIEW", "EXECUTIVE_DIRECTOR_REVIEW", "General Manager reviewed exact revision 2.", 3],
      ["executiveDirector", "USR-ED-ZARA", "EXECUTIVE_DIRECTOR_REVIEW", "GM_RELEASE", "Executive Director approved exact revision 3.", 4],
      ["gm", "USR-GM-OMAR", "GM_RELEASE", "RELEASED", "General Manager released exact revision 4 to Department.", 5],
    ]);

    renderGovernanceRoute("/executive-director/planning", "executiveDirector", runtime);
    expect(await screen.findByTestId("executive-planning-page")).toHaveTextContent("PLAN-2026-ZERO-001");
  });

  it("moves an exact Preliminary Report version Manager to GM to Executive without Finance report authority", async () => {
    const runtime = createMockBackendRuntime();
    await advancePreliminaryReportTo(runtime, "GM_REVIEW");
    const user = userEvent.setup();
    renderGovernanceRoute("/general-manager/report-approvals", "gm", runtime);

    const page = await screen.findByTestId("gm-report-approvals-page");
    const selected = await within(page).findByRole("region", { name: "Selected report PR-2026-018-V1" });
    expect(selected).toHaveAttribute("data-report-version-id", "PR-2026-018-V1");
    expect(selected).toHaveAttribute("data-report-revision", "2");
    const selectedQueueAction = within(page).getByRole("button", { name: "PR-2026-018-V1 is already selected" });
    expect(selectedQueueAction).toBeDisabled();
    expect(selectedQueueAction).toHaveAttribute(
      "title",
      "PR-2026-018-V1 is already open in the General Manager report dossier.",
    );
    expect(page).not.toHaveTextContent(/Finance Review.*Report/i);
    await expect(runtime.backendForRole("finance").reports.decide({
      operationId: "TASK8-FINANCE-REPORT-DENIED",
      reportVersionId: "PR-2026-018-V1",
      expectedReportVersionRevision: 2,
      decision: "FORWARD",
      reason: "Finance must not participate in report approval.",
    })).rejects.toThrow(/role or report stage|cannot perform/i);
    await user.type(within(selected).getByLabelText("General Manager report decision reason"), "Exact immutable Preliminary Report version reviewed.");
    await user.click(within(selected).getByRole("button", { name: "Forward PR-2026-018-V1 to Executive Director" }));
    expect(await within(selected).findByTestId("report-status")).toHaveTextContent("EXECUTIVE_DIRECTOR_REVIEW");
  });

  it("shows only stage-eligible immutable report versions in the GM and Executive Director queues", async () => {
    const runtime = createMockBackendRuntime();
    renderGovernanceRoute("/general-manager/report-approvals", "gm", runtime);

    const gmPage = await screen.findByTestId("gm-report-approvals-page");
    expect(gmPage).toHaveTextContent("No report versions are at GM_REVIEW");
    expect(gmPage).toHaveTextContent("PR-2026-018-V1 remains DEPARTMENT_REVIEW");
    expect(within(gmPage).queryByRole("region", { name: /Selected report/ })).toBeNull();

    cleanup();
    renderGovernanceRoute("/executive-director/preliminary-reports", "executiveDirector", runtime);
    const executivePage = await screen.findByTestId("executive-preliminary-reports-page");
    expect(executivePage).toHaveTextContent(
      "PR-2026-018-V1 is DEPARTMENT_REVIEW. Department Manager must forward the exact version, then General Manager must forward it before Executive Director review.",
    );
    expect(within(executivePage).queryByRole("region", { name: /Selected Preliminary Report/ })).toBeNull();
    expect(executivePage).not.toHaveTextContent("Decision recorded");
  });

  it("renders connected report workspaces as empty without requesting obsolete fixture identities", async () => {
    const runtime = createMockBackendRuntime();
    const executive = runtime.backendForRole("executiveDirector");
    vi.spyOn(executive.documents, "list").mockResolvedValue({ items: [], nextCursor: null });
    const getVersion = vi.spyOn(executive.reports, "getVersion").mockRejectedValue(new Error("Not found."));

    renderGovernanceRoute("/executive-director/final-reports", "executiveDirector", runtime);

    const page = await screen.findByTestId("executive-final-reports-page");
    expect(page).toHaveTextContent("No Final Report versions are available yet.");
    expect(within(page).queryByRole("alert")).toBeNull();
    expect(getVersion).not.toHaveBeenCalled();
  });

  it("keeps a General Manager return reason visible and bound to the exact immutable report version and audit event", async () => {
    const runtime = createMockBackendRuntime();
    await advancePreliminaryReportTo(runtime, "GM_REVIEW");
    const user = userEvent.setup();
    renderGovernanceRoute("/general-manager/report-approvals", "gm", runtime);

    const page = await screen.findByTestId("gm-report-approvals-page");
    const selected = within(page).getByRole("region", { name: "Selected report PR-2026-018-V1" });
    await user.type(within(selected).getByLabelText("General Manager report decision reason"), "Return version 1 because the Finding basis needs clarification.");
    await user.click(within(selected).getByRole("button", { name: "Return PR-2026-018-V1 to Department Manager" }));
    expect(await within(selected).findByTestId("report-status")).toHaveTextContent("RETURNED");
    expect(selected).toHaveTextContent("Return version 1 because the Finding basis needs clarification.");
    expect(selected).toHaveTextContent("PR-2026-018-V1");
    const events = await runtime.backendForRole("gm").auditTrail.list({ entityType: "report_version", entityId: "PR-2026-018-V1" });
    expect(events.items.at(-1)).toMatchObject({
      actorRole: "gm",
      actorSubjectId: "USR-GM-OMAR",
      entityId: "PR-2026-018-V1",
      afterStatus: "RETURNED",
      reason: "Return version 1 because the Finding basis needs clarification.",
      entityRevision: 3,
    });
  });

  it("compares departments from truthful projections without exposing Department Manager editing controls", async () => {
    renderGovernanceRoute("/general-manager/departments", "gm");
    const page = await screen.findByTestId("gm-departments-page");
    const table = within(page).getByRole("table", { name: "Department comparison" });
    expect(within(table).getByText("Cabin Safety")).toBeVisible();
    expect(within(table).getByText(/Fly Namibia/)).toBeVisible();
    expect(page).toHaveTextContent("Cross-department oversight");
    expect(page).not.toHaveTextContent(/assign inspector|edit checklist|approve CAP/i);
    expect(within(page).getByRole("button", { name: "Open Cabin Safety department summary" })).toBeEnabled();
  });

  it("renders cross-department risk as decision support only and never as an automatic legal or closure determination", async () => {
    renderGovernanceRoute("/general-manager/risk-dashboard", "gm");
    const page = await screen.findByTestId("gm-risk-dashboard-page");
    expect(within(page).getByRole("region", { name: "Oversight Health indicators" })).toBeVisible();
    const matrix = within(page).getByRole("region", { name: "Risk Exposure Matrix" });
    expect(matrix.querySelectorAll("[data-matrix-cell]")).toHaveLength(25);
    expect(matrix.querySelectorAll(".is-low, .is-medium, .is-high, .is-critical")).toHaveLength(25);
    expect(page).toHaveTextContent("Management indicator only");
    expect(page).toHaveTextContent(/does not (make|trigger) an automatic legal, enforcement, certificate, suspension, compliance, or closure decision/i);
    expect(page).not.toHaveTextContent(/automatically (close|suspend|revoke|certify)/i);
    expect(within(page).getByRole("button", { name: "Open risk Finding FND-SKYCARGO-2026-099 unavailable" })).toHaveAttribute(
      "title",
      "Finding FND-SKYCARGO-2026-099 has no declared General Manager detail route in Plan 1.",
    );
  });

  it("lets Executive Director decide only the exact Executive-owned Planning revision and returns release authority to GM", async () => {
    const runtime = createMockBackendRuntime();
    await advancePlanTo(runtime, "EXECUTIVE_DIRECTOR_REVIEW");
    const user = userEvent.setup();
    renderGovernanceRoute("/executive-director/planning", "executiveDirector", runtime);

    const page = await screen.findByTestId("executive-planning-page");
    expect(page).toHaveAttribute("data-planning-item-id", "PLAN-2026-CAB-001");
    expect(page).toHaveAttribute("data-planning-revision", "3");
    await user.type(within(page).getByLabelText("Executive Director plan decision reason"), "Final plan authority recorded for revision 3.");
    await user.click(within(page).getByRole("button", { name: "Approve plan PLAN-2026-CAB-001" }));
    expect(await within(page).findByTestId("planning-status")).toHaveTextContent("GM_RELEASE");
    expect(page).toHaveTextContent("General Manager to release approved plan");
    expect(page).toHaveTextContent("Decision recorded without signature assertion");
  });

  it("issues Preliminary Reports only after Manager and GM review and keeps unsupported Executive return visibly disabled for version 1", async () => {
    const runtime = createMockBackendRuntime();
    await advancePreliminaryReportTo(runtime, "EXECUTIVE_DIRECTOR_REVIEW");
    const user = userEvent.setup();
    renderGovernanceRoute("/executive-director/preliminary-reports", "executiveDirector", runtime);

    const page = await screen.findByTestId("executive-preliminary-reports-page");
    const selected = await within(page).findByRole("region", { name: "Selected Preliminary Report PR-2026-018-V1" });
    expect(selected).toHaveAttribute("data-report-version-id", "PR-2026-018-V1");
    expect(selected).toHaveAttribute("data-report-revision", "3");
    expect(within(selected).getByRole("button", { name: "Return PR-2026-018-V1 unavailable" })).toHaveAttribute(
      "title",
      "Report version PR-2026-018-V1 is at Executive Director Review; the typed Plan 1 command permits issue and lock only. Return authority is not declared for Executive Director.",
    );
    await user.type(within(selected).getByLabelText("Executive Director report decision reason"), "Issue exact Preliminary Report version 1.");
    await user.click(within(selected).getByRole("button", { name: "Issue and lock PR-2026-018-V1" }));
    expect(await within(selected).findByTestId("report-status")).toHaveTextContent("LOCKED");
    const events = await runtime.backendForRole("executiveDirector").auditTrail.list({ entityType: "report_version", entityId: "PR-2026-018-V1" });
    expect(events.items.map((event) => event.actorRole)).toEqual(["manager", "gm", "executiveDirector"]);
    expect(events.items.at(-1)?.entityRevision).toBe(4);
  });

  it("keeps the canonical Final Report fixture explicitly unlinked and issuing it never closes a separately created Finding", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/findings/FND-CAB-2026-001");
    const findingBefore = await runtime.backendForRole("leadInspector").findings.get({ findingId: "FND-CAB-2026-001" });
    const reportBefore = await runtime.backendForRole("executiveDirector").reports.getVersion({ reportVersionId: "RPT-CAB-2026-001-V1" });
    expect(reportBefore.findingIds).toEqual([]);
    const user = userEvent.setup();
    renderGovernanceRoute("/executive-director/final-reports", "executiveDirector", runtime);

    const page = await screen.findByTestId("executive-final-reports-page");
    const selected = within(page).getByRole("region", { name: "Selected Final Report RPT-CAB-2026-001-V1" });
    expect(selected).toHaveTextContent("No Findings linked — relationship unavailable for RPT-CAB-2026-001-V1");
    expect(within(selected).getByRole("link", { name: "Preview RPT-CAB-2026-001-V1" })).toHaveAttribute(
      "href",
      "/executive-director/reports/RPT-CAB-2026-001",
    );
    await user.type(within(selected).getByLabelText("Executive Director report decision reason"), "Issue and lock exact immutable Final Report version 1.");
    await user.click(within(selected).getByRole("button", { name: "Issue and lock RPT-CAB-2026-001-V1" }));
    expect(await within(selected).findByTestId("report-status")).toHaveTextContent("LOCKED");
    expect(await runtime.backendForRole("leadInspector").findings.get({ findingId: findingBefore.id })).toMatchObject({ status: findingBefore.status, revision: findingBefore.revision });

    await user.click(within(selected).getByRole("link", { name: "Preview RPT-CAB-2026-001-V1" }));
    const preview = await screen.findByTestId("executive-report-preview-page");
    expect(preview).toHaveAttribute("data-report-version-id", "RPT-CAB-2026-001-V1");
    expect(preview).toHaveTextContent("No Findings linked — relationship unavailable for RPT-CAB-2026-001-V1");
  });

  it("downloads the exact immutable Final Report identity with the frozen PDF file name", async () => {
    const nativeUrl = globalThis.URL;
    const createObjectURL = vi.fn(() => "blob:task8-report");
    const revokeObjectURL = vi.fn();
    class DownloadURL extends nativeUrl {}
    Object.defineProperties(DownloadURL, {
      createObjectURL: { configurable: true, value: createObjectURL },
      revokeObjectURL: { configurable: true, value: revokeObjectURL },
    });
    vi.stubGlobal("URL", DownloadURL);
    let clickedAnchor: HTMLAnchorElement | undefined;
    vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(function captureDownload(this: HTMLAnchorElement) {
      clickedAnchor = this;
    });
    const user = userEvent.setup();
    renderGovernanceRoute("/executive-director/reports/RPT-CAB-2026-001", "executiveDirector");

    const preview = await screen.findByTestId("executive-report-preview-page");
    expect(preview).toHaveAttribute("data-report-version-id", "RPT-CAB-2026-001-V1");
    await user.click(within(preview).getByRole("button", { name: "Download PDF" }));

    expect(createObjectURL).toHaveBeenCalledWith(expect.any(Blob));
    expect(clickedAnchor?.download).toBe("RPT-CAB-2026-001.pdf");
    expect(clickedAnchor?.href).toContain("blob:task8-report");
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:task8-report");
    expect(within(preview).getByRole("status")).toHaveTextContent("RPT-CAB-2026-001.pdf");
    expect(within(preview).getByRole("status")).toHaveTextContent("RPT-CAB-2026-001-V1");
  });

  it("marks exact Executive notifications read and hydrates role-safe GM and Executive settings after remount at every viewport", async () => {
    const runtime = createMockBackendPersistentRuntime(localStorage);
    const user = userEvent.setup();
    renderGovernanceRoute("/executive-director/notifications", "executiveDirector", runtime);
    const notifications = await screen.findByTestId("executive-notifications-page");
    const record = await within(notifications).findByRole("article", { name: "Notification NOT-EXEC-001" });
    expect(record).toHaveTextContent("Email delivery: Not configured in demo");
    await user.click(within(record).getByRole("button", { name: "Mark NOT-EXEC-001 read" }));
    await screen.findByText("Read · revision 2");
    expect(record).toHaveTextContent("Read · revision 2");
    expect(await runtime.backendForRole("executiveDirector").notifications.list({})).toEqual(expect.objectContaining({
      items: [expect.objectContaining({ id: "NOT-EXEC-001", revision: 2, readAt: "2026-06-15T09:00:00.000Z" })],
    }));

    cleanup();
    renderGovernanceRoute("/executive-director/notifications", "executiveDirector", runtime);
    expect(within(await screen.findByTestId("executive-notifications-page")).getByRole("article", { name: "Notification NOT-EXEC-001" })).toHaveTextContent("Read");

    cleanup();
    renderGovernanceRoute("/general-manager/settings", "gm", runtime);
    const gm = await screen.findByTestId("gm-settings-page");
    await user.click(within(gm).getByRole("button", { name: "Edit profile" }));
    await user.clear(within(gm).getByLabelText("Display name"));
    await user.type(within(gm).getByLabelText("Display name"), "Omar GM Updated");
    await user.click(within(gm).getByRole("button", { name: "Save profile" }));
    expect(await within(gm).findByText("Omar GM Updated")).toBeVisible();
    cleanup();
    renderGovernanceRoute("/general-manager/settings", "gm", runtime);
    expect(await screen.findByTestId("gm-settings-page")).toHaveTextContent("Omar GM Updated");

    cleanup();
    renderGovernanceRoute("/executive-director/settings", "executiveDirector", runtime);
    const executiveSettings = await screen.findByTestId("executive-settings-page");
    await user.click(within(executiveSettings).getByRole("button", { name: "Edit profile" }));
    await user.clear(within(executiveSettings).getByLabelText("Display name"));
    await user.type(within(executiveSettings).getByLabelText("Display name"), "Zara ED Updated");
    await user.click(within(executiveSettings).getByRole("button", { name: "Save profile" }));
    expect(await within(executiveSettings).findByText("Zara ED Updated")).toBeVisible();

    cleanup();
    renderGovernanceRoute("/executive-director/settings", "executiveDirector", runtime);
    expect(await screen.findByTestId("executive-settings-page")).toHaveTextContent("Zara ED Updated");
    cleanup();
    renderGovernanceRoute("/general-manager/settings", "gm", runtime);
    const remountedGm = await screen.findByTestId("gm-settings-page");
    expect(remountedGm).toHaveTextContent("Omar GM Updated");
    expect(remountedGm).not.toHaveTextContent("Zara ED Updated");

    cleanup();
    for (const width of [1440, 1024, 390]) {
      Object.defineProperty(window, "innerWidth", { configurable: true, value: width });
      renderGovernanceRoute("/executive-director/settings", "executiveDirector", runtime);
      const executive = await screen.findByTestId("executive-settings-page");
      expect(within(executive).getByRole("heading", { level: 1, name: "Settings" })).toBeVisible();
      expect(executive).toHaveTextContent("Zara ED Updated");
      expect(executive).not.toHaveTextContent("Omar GM Updated");
      expect(within(executive).getByRole("button", { name: "Edit profile" })).toBeEnabled();
      cleanup();
    }
  });
});
