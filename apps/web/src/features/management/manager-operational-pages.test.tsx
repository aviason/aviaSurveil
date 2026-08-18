// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { REACT_ROUTE_CONTRACT_BY_ID } from "../../app/route-contracts";
import { AppRouter } from "../../app/router";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { seedVisualRuntimeForPath } from "../../mock/seed-visual-runtime";

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

afterEach(cleanup);

function renderManagerRoute(path: string, runtime: MockRuntime = createMockBackendRuntime()) {
  render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: "demo",
      environmentLabel: "test",
      identityMode: "demo-role-switch",
      subjectId: "USR-MANAGER-NORA",
    }}>
      <ScenarioProvider><MemoryRouter initialEntries={[path]}><AppRouter /></MemoryRouter></ScenarioProvider>
    </AppProviders>,
  );
  return runtime;
}

describe("Department Manager operational workspaces", () => {
  it.each([
    ["/department-manager/audits", "manager-audits-page", "Audit Work Queue"],
    ["/department-manager/inspection-team", "manager-inspection-team-page", "Inspection Team"],
    ["/department-manager/findings-review", "manager-findings-review-page", "Findings Review"],
    ["/department-manager/cap-monitoring", "manager-cap-monitoring-page", "CAP Monitoring"],
    ["/department-manager/checklist-management", "manager-checklist-management-page", "Checklist Management"],
    ["/department-manager/organizations/ORG-FLY-NAMIBIA", "manager-organization-detail-page", "Fly Namibia"],
  ] as const)("direct-loads %s through the Manager-owned route", async (path, testId, heading) => {
    renderManagerRoute(path);
    const page = await screen.findByTestId(testId);
    expect(await within(page).findByRole("heading", { level: 1, name: heading })).toBeVisible();
    expect(screen.getByTestId("application-shell")).toHaveAttribute("data-active-role", "manager");
    expect(screen.queryByTestId("route-pending-implementation")).toBeNull();
  });

  it("keeps the evidence route Manager-owned and binds it to the exact Finding ID", async () => {
    const contract = REACT_ROUTE_CONTRACT_BY_ID.get("evidence-review");
    expect(contract).toMatchObject({
      auditId: "ui-audit-044",
      path: "/department-manager/evidence/:findingId",
      requiredRole: "manager",
      parentId: "manager-findings-review",
    });
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/department-manager/evidence/FND-CAB-2026-001");
    renderManagerRoute("/department-manager/evidence/FND-CAB-2026-001", runtime);
    const page = await screen.findByTestId("manager-inspection-evidence-page");
    expect(page).toHaveTextContent("FND-CAB-2026-001");
    expect(page).toHaveTextContent("EV-CAB-2026-001-V2");
    expect(page).not.toHaveTextContent("Lead Inspector workspace");
  });

  it("does not implicitly select a Finding and uses the server-owned selection when requested", async () => {
    const runtime = createMockBackendRuntime();
    const finding = (await runtime.backendForRole("manager").findings.list({ limit: 50 })).items.find(Boolean);
    if (!finding) throw new Error("Expected the server-owned target Finding.");
    renderManagerRoute("/department-manager/findings-review", runtime);
    const page = await screen.findByTestId("manager-findings-review-page");
    expect(within(page).getByText("Select a Finding.")).toBeVisible();
    await userEvent.click(within(page).getByRole("button", { name: new RegExp(finding.findingNumber) }));
    expect(within(page).getByRole("heading", { name: finding.title })).toBeVisible();
    expect(within(page).getByRole("link", { name: "Open Evidence" })).toHaveAttribute(
      "href",
      `/department-manager/evidence/${finding.id}`,
    );
  });

  it("browses approved catalog pages from an exact scope without review or publication actions", async () => {
    const runtime = createMockBackendRuntime();
    const manager = runtime.backendForRole("manager");
    const scopes = await manager.canonicalCatalog!.listScopeOptions({ limit: 20 });
    const scope = scopes.items[0];
    if (!scope) throw new Error("Expected an approved catalog scope.");
    renderManagerRoute("/department-manager/checklist-management", runtime);
    const page = await screen.findByTestId("manager-checklist-management-page");
    const user = userEvent.setup();
    await user.selectOptions(within(page).getByLabelText("Foundation scope"), scope.providerScopeId);
    await user.click(within(page).getByRole("button", { name: "Load approved questions" }));
    expect(await within(page).findByRole("list", { name: "Approved AGA catalog questions" })).toBeVisible();
    expect(within(page).getByText(/loaded/)).toBeVisible();
    expect(page).not.toHaveTextContent(/candidate|exercise|PREPROD_EXERCISE|publish|approval/i);
    expect(within(page).getByRole("link", { name: "Open New Audit question selection" })).toHaveAttribute(
      "href",
      "/department-manager/new-audit/step-4",
    );
  });
});
