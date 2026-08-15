// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { GeneralManagerDashboardPage } from "./planning-workspaces";

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

function renderPage(runtime: MockRuntime) {
  return render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: "demo",
      environmentLabel: "test",
      identityMode: "demo-role-switch",
      subjectId: "USR-GM-OKAN",
    }}>
      <MemoryRouter initialEntries={["/general-manager/dashboard"]}><GeneralManagerDashboardPage /></MemoryRouter>
    </AppProviders>,
  );
}

describe("GeneralManagerDashboardPage", () => {
  it("loads the connected dashboard hierarchy and uses server-owned report identities", async () => {
    const runtime = createMockBackendRuntime();
    renderPage(runtime);
    const page = await screen.findByRole("heading", { name: "General Manager Dashboard" });
    expect(page).toBeVisible();
    expect(screen.getByRole("region", { name: "General Manager indicators" })).toBeVisible();
    expect(screen.getByRole("table", { name: "Department Overview" })).toBeVisible();
    expect(screen.getByRole("region", { name: "Risk Heat Map" })).toBeVisible();
    const queue = screen.getByRole("table", { name: "Report Review Queue" });
    const reports = (await runtime.backendForRole("gm").documents.list({})).items.filter((item) => item.kind === "REPORT");
    const report = reports.find((candidate) => candidate.id.startsWith("RPT-") || candidate.id.startsWith("PR-"));
    if (report) {
      expect(within(queue).getByText(report.id)).toBeVisible();
    } else {
      expect(within(queue).getByText("No report versions are available yet.")).toBeVisible();
    }
    expect(screen.getByText(/cannot issue, sign, lock, or close/i)).toBeVisible();
    expect(screen.queryByText(/Final authorized demo approval/i)).toBeNull();
  });

  it("renders an empty report queue from the backend without probing an obsolete fixture ID", async () => {
    const runtime = createMockBackendRuntime();
    const gm = runtime.backendForRole("gm");
    vi.spyOn(gm.documents, "list").mockResolvedValue({ items: [], nextCursor: null });
    const getVersion = vi.spyOn(gm.reports, "getVersion").mockRejectedValue(new Error("Not found."));
    renderPage(runtime);
    expect(await screen.findByText("No report versions are available yet.")).toBeVisible();
    expect(screen.queryByRole("alert")).toBeNull();
    expect(getVersion).not.toHaveBeenCalled();
  });

  it("does not synthesize a Planning item when the connected queue is empty", async () => {
    const runtime = createMockBackendRuntime();
    const gm = runtime.backendForRole("gm");
    vi.spyOn(gm.planning, "list").mockResolvedValue({ items: [], nextCursor: null });
    renderPage(runtime);
    const queue = await screen.findByRole("region", { name: "General Manager planning queue" });
    expect(within(queue).getByText("No server-owned Planning items are available.")).toBeVisible();
    expect(within(queue).queryByRole("button")).toBeNull();
  });
});
