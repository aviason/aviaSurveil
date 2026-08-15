// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { AppRouter } from "../../app/router";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { seedVisualRuntimeForPath } from "../../mock/seed-visual-runtime";

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

afterEach(() => {
  cleanup();
  Object.defineProperty(window, "innerWidth", { configurable: true, value: 1024 });
});

function renderLeadRoute(path: string, runtime: MockRuntime = createMockBackendRuntime()) {
  render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: "demo",
      environmentLabel: "test",
      identityMode: "demo-role-switch",
      subjectId: "USR-LEAD-CANER",
    }}>
      <ScenarioProvider><MemoryRouter initialEntries={[path]}><AppRouter /></MemoryRouter></ScenarioProvider>
    </AppProviders>,
  );
  return runtime;
}

describe("Lead Inspector assignment and secondary routes", () => {
  it("direct-loads the exact Audit assignment and keeps preparation tied to server-owned IDs", async () => {
    const runtime = createMockBackendRuntime();
    const assignments = await runtime.backendForRole("leadInspector").assignments.list({});
    const assignment = assignments.items.find((item) => item.auditId === "AUD-2026-001");
    if (!assignment) throw new Error("Expected the server-owned Lead assignment.");
    renderLeadRoute(`/lead-inspector/audits/${assignment.auditId}/assignment`, runtime);
    const page = await screen.findByTestId("lead-audit-assignment-page");
    expect(await within(page).findByText(assignment.auditId)).toBeVisible();
    expect(page).toHaveAttribute("data-audit-id", assignment.auditId);
    expect(within(page).getByRole("link", { name: "View Preliminary Reports" })).toHaveAttribute(
      "href",
      "/lead-inspector/preliminary-reports",
    );
    expect(page).not.toHaveTextContent(/localStorage|mock mutation|fixed.*question/i);
  });

  it("renders every question in the server-owned materialized package without a local assignment write path", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/lead-inspector/audits/AUD-2026-001/assignment");
    const lead = runtime.backendForRole("leadInspector");
    const assignment = (await lead.assignments.list({})).items.find((item) => item.packageId);
    if (!assignment?.packageId) throw new Error("Expected a materialized server-owned package.");
    const packageView = await lead.inspections.getPackage({ packageId: assignment.packageId });
    renderLeadRoute(`/lead-inspector/audits/${assignment.auditId}/checklist-questions`, runtime);
    const page = await screen.findByTestId("lead-question-assignment-page");
    const questions = await within(page).findByRole("region", { name: "Checklist questions" });
    for (const question of packageView.questions) expect(within(questions).getByText(question.id)).toBeVisible();
    expect(within(page).queryByRole("button", { name: "Assign Questions" })).toBeNull();
    expect(within(page).getByRole("link", { name: "Open Lead preparation workspace" })).toHaveAttribute(
      "href",
      `/lead-inspector/audit-preparation?assignmentId=${encodeURIComponent(`${assignment.auditId}:assignment`)}`,
    );
  });

  it("keeps Lead analytics advisory and truthful when the role has no risk capability", async () => {
    renderLeadRoute("/lead-inspector/analytics-reports");
    const page = await screen.findByTestId("lead-analytics-page");
    expect(within(page).getByRole("heading", { name: "Safety Intelligence Dashboard" })).toBeVisible();
    expect(within(page).queryByRole("alert")).toBeNull();
    expect(within(page).getByText("No server-owned signals match the current filter.")).toBeVisible();
    expect(page).not.toHaveTextContent(/Lead management summary|SkyCargo|FND-|localStorage/i);
  });

  it.each([900, 390])("keeps the assignment surface ordered at %ipx", async (width) => {
    Object.defineProperty(window, "innerWidth", { configurable: true, value: width });
    renderLeadRoute("/lead-inspector/audits/AUD-2026-001/assignment");
    const page = await screen.findByTestId("lead-audit-assignment-page");
    expect(within(page).getByRole("heading", { name: /Audit assignment|2026 Cabin Inspection/ })).toBeVisible();
    expect(page.querySelector(".workbench-page-header")).not.toBeNull();
  });
});
