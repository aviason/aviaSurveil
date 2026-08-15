// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { AppRouter } from "../../app/router";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
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

async function reportDocument(runtime: MockRuntime, prefix: "PR-" | "RPT-") {
  const document = (await runtime.backendForRole("leadInspector").documents.list({})).items.find((item) => item.kind === "REPORT" && item.id.startsWith(prefix));
  if (!document) throw new Error(`Expected a server-owned ${prefix} report document.`);
  return runtime.backendForRole("leadInspector").reports.getVersion({ reportVersionId: document.id });
}

describe("Lead Inspector report workspaces", () => {
  it("lists server-owned Preliminary Report versions and exposes exact route links", async () => {
    const runtime = createMockBackendRuntime();
    const report = await reportDocument(runtime, "PR-");
    renderLeadRoute("/lead-inspector/preliminary-reports", runtime);
    const page = await screen.findByTestId("lead-preliminary-reports-page");
    expect(within(page).getByRole("heading", { name: "Preliminary Reports" })).toBeVisible();
    const row = await within(page).findByRole("article", { name: `Preliminary Report ${report.reportVersionId}` });
    expect(within(row).getByText(report.reportVersionId)).toBeVisible();
    expect(within(row).getByRole("link", { name: "Open report package" })).toHaveAttribute(
      "href",
      `/lead-inspector/preliminary-reports/${report.reportVersionId}`,
    );
  });

  it("opens the exact Preliminary Report version as a read-only server snapshot", async () => {
    const runtime = createMockBackendRuntime();
    const report = await reportDocument(runtime, "PR-");
    renderLeadRoute(`/lead-inspector/preliminary-reports/${report.reportVersionId}`, runtime);
    const page = await screen.findByTestId("lead-preliminary-report-workflow-page");
    await within(page).findByText(report.reportVersionId);
    expect(page).toHaveAttribute("data-report-id", report.reportId);
    expect(page).toHaveAttribute("data-audit-id", report.auditId);
    expect(within(page).getByText(report.reportVersionId)).toBeVisible();
    expect(within(page).getByRole("button", { name: "Preview server version" })).toBeEnabled();
    expect(page).not.toHaveTextContent(/Save Draft|mock mutation|localStorage|Internal CAA Note/i);
    await userEvent.click(within(page).getByRole("button", { name: "Preview server version" }));
    expect(within(page).getByRole("region", { name: "Immutable Preliminary Report preview" })).toHaveTextContent(report.reportVersionId);
  });

  it("renders truthful empty Final Report state when the connected document query is empty", async () => {
    const runtime = createMockBackendRuntime();
    vi.spyOn(runtime.backendForRole("leadInspector").documents, "list").mockResolvedValue({ items: [], nextCursor: null });
    renderLeadRoute("/lead-inspector/final-reports", runtime);
    const page = await screen.findByTestId("lead-final-reports-page");
    expect(await within(page).findByText("No Final Report versions are available.")).toBeVisible();
    expect(page).not.toHaveTextContent(/RPT-CAB|PR-2026|candidate|exercise/i);
  });

  it("keeps the exact Final Report readiness and document routes read-only", async () => {
    const runtime = createMockBackendRuntime();
    const report = await reportDocument(runtime, "RPT-");
    renderLeadRoute(`/lead-inspector/final-reports/${report.reportVersionId}/readiness`, runtime);
    const readiness = await screen.findByTestId("lead-final-report-readiness-page");
    expect(await within(readiness).findByText(report.reportVersionId)).toBeVisible();
    expect(within(readiness).getByText(/Lead Inspector cannot issue or lock/i)).toBeVisible();
    expect(within(readiness).queryByRole("button", { name: /approve|issue|lock report/i })).toBeNull();
  });
});
