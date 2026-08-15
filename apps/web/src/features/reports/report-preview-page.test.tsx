// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { ReportPreviewPage } from "./report-preview-page";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

function renderPage() {
  const runtime = createMockBackendRuntime();
  const getVersion = vi.spyOn(runtime.backendForRole("manager").reports, "getVersion");
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
        <MemoryRouter initialEntries={["/department-manager/reports/RPT-CAB-2026-001-V1"]}>
          <Routes>
            <Route path="/department-manager/reports/:reportVersionId" element={<ReportPreviewPage />} />
          </Routes>
        </MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
  return { runtime, getVersion };
}

describe("ReportPreviewPage", () => {
  it("direct-loads the immutable report version and truthful manager authority state", async () => {
    const { getVersion } = renderPage();

    expect(await screen.findByRole("heading", { name: "Reports Approval" })).toBeVisible();
    expect(getVersion).toHaveBeenCalledWith({ reportVersionId: "RPT-CAB-2026-001-V1" });
    const dossier = screen.getByTestId("report-version-dossier");
    expect(within(dossier).getByText("RPT-CAB-2026-001-V1")).toBeVisible();
    expect(within(dossier).getByText("Version 1")).toBeVisible();
    expect(within(dossier).getByText("EXECUTIVE_DIRECTOR_REVIEW")).toBeVisible();
    expect(within(dossier).getByText("sha256:candidate-report-v1")).toBeVisible();
    expect(screen.getByText(/Department Manager cannot issue, sign, lock, or close/i)).toBeVisible();
    expect(screen.queryByRole("button", { name: /issue|sign|close/i })).toBeNull();
  });

  it("keeps queue, tabs, preview, and disabled download behavior functional and explicit", async () => {
    renderPage();
    const user = userEvent.setup();

    const queue = await screen.findByRole("table", { name: "Report Queue" });
    expect(within(queue).getByText("RPT-CAB-2026-001-V1")).toBeVisible();
    const search = screen.getByRole("button", { name: "Search reports unavailable" });
    const reset = screen.getByRole("button", { name: "Reset report filters unavailable" });
    expect(search).toBeDisabled();
    expect(search).toHaveAttribute("title", "The current report search is already applied.");
    expect(reset).toBeDisabled();
    expect(reset).toHaveAttribute("title", "Report filters are already at their defaults.");
    await user.type(screen.getByLabelText("Search reports"), "no matching report");
    expect(screen.getByRole("button", { name: "Search" })).toBeEnabled();
    await user.click(screen.getByRole("button", { name: "Search" }));
    expect(within(queue).getByText("No matching report versions.")).toBeVisible();
    expect(screen.getByRole("button", { name: "Reset" })).toBeEnabled();
    await user.selectOptions(screen.getByLabelText("Report type"), "inspection");
    await user.selectOptions(screen.getByLabelText("Report status"), "in-review");
    await user.click(screen.getByRole("button", { name: "Reset" }));
    expect(within(queue).getByText("RPT-CAB-2026-001-V1")).toBeVisible();
    expect(screen.getByLabelText("Report type")).toHaveValue("all");
    expect(screen.getByLabelText("Report status")).toHaveValue("all");
    expect(screen.getByRole("button", { name: "Reset report filters unavailable" })).toBeDisabled();
    expect(within(queue).queryByRole("button", { name: /Department review unavailable/i })).toBeNull();
    expect(screen.getByText(/Department Manager cannot issue, sign, lock, or close this report/i)).toBeVisible();
    await user.click(screen.getByRole("tab", { name: "Decision history" }));
    expect(screen.getByRole("tabpanel")).toHaveTextContent(/current immutable state/i);
    await user.click(screen.getByRole("button", { name: "Review Full Report" }));
    expect(screen.getByRole("dialog", { name: "Immutable report preview" })).toBeVisible();
    expect(screen.getByText(/This preview is read-only/i)).toBeVisible();
  });
});
