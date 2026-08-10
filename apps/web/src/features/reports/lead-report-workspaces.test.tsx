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
import { seedVisualRuntimeForPath } from "../../mock/seed-visual-runtime";

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
      <ScenarioProvider>
        <MemoryRouter initialEntries={[path]}>
          <AppRouter />
        </MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
  return runtime;
}

describe("Lead Inspector report workspaces", () => {
  it("direct-loads Preliminary Reports with immutable versions and a returned report reason", async () => {
    renderLeadRoute("/lead-inspector/preliminary-reports");

    const page = await screen.findByTestId("lead-preliminary-reports-page");
    expect(within(page).getByRole("heading", { level: 1, name: "Preliminary Reports" })).toBeVisible();
    expect(within(page).getByText("PR-2026-018")).toBeVisible();
    const returned = within(page).getByRole("article", { name: "Preliminary Report PR-2026-019 version 2" });
    expect(returned).toHaveAttribute("data-report-id", "PR-2026-019");
    expect(returned).toHaveAttribute("data-report-version", "2");
    expect(within(returned).getByText("Returned to Lead Inspector")).toBeVisible();
    expect(within(returned).getByText(/return reason:/i)).toHaveTextContent("Finding basis requires clarification");
  });

  it("direct-loads the exact PR-2026-018 workflow and separates auditee and internal comments", async () => {
    renderLeadRoute("/lead-inspector/preliminary-reports/PR-2026-018");

    const page = await screen.findByTestId("lead-preliminary-report-workflow-page");
    expect(within(page).getByRole("heading", { name: /Preliminary Report.*Cabin Inspection.*Fly Namibia/ })).toBeVisible();
    expect(page).toHaveAttribute("data-report-id", "PR-2026-018");
    expect(page).toHaveAttribute("data-audit-id", "AUD-2026-001");
    expect(within(page).getByText("1.0 (Working)")).toBeVisible();
    const auditeeComment = within(page).getByRole("region", { name: "Comment to Auditee" });
    const internalNote = within(page).getByRole("region", { name: "Internal CAA Note" });
    expect(auditeeComment).not.toBe(internalNote);
    expect(internalNote).toHaveTextContent(/CAA-only/i);
  });

  it("saves the PR-2026-018 working draft and opens a record-specific document preview", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/lead-inspector/preliminary-reports/PR-2026-018");
    const getFinding = vi.spyOn(runtime.backendForRole("leadInspector").findings, "get");
    renderLeadRoute("/lead-inspector/preliminary-reports/PR-2026-018", runtime);

    const page = await screen.findByTestId("lead-preliminary-report-workflow-page");
    const finding = await within(page).findByRole("article", { name: "Finding FND-CAB-2026-001" });
    expect(finding).toHaveTextContent("FND-CAB-2026-001");
    expect(page).not.toHaveTextContent("CAB-2026-011");
    expect(getFinding).toHaveBeenCalledWith({ findingId: "FND-CAB-2026-001" });
    await user.click(within(page).getByRole("button", { name: "Report Content" }));
    const content = within(page).getByRole("region", { name: "Report Content workspace" });
    const summary = within(content).getByLabelText("Executive Summary");
    await user.clear(summary);
    await user.type(summary, "Canonical FND-CAB-2026-001 summary for Lead review.");
    await user.clear(within(page).getByLabelText("Comment to Auditee text"));
    await user.type(within(page).getByLabelText("Comment to Auditee text"), "Auditee-visible canonical Finding comment.");
    await user.clear(within(page).getByLabelText("Internal CAA Note text"));
    await user.type(within(page).getByLabelText("Internal CAA Note text"), "Internal canonical Finding note.");
    await user.click(within(page).getByRole("button", { name: "Save Draft" }));
    const saved = within(page).getByRole("status");
    expect(saved).toHaveAttribute("data-durable-outcome", "preliminary-report-saved");
    expect(saved).toHaveTextContent("PR-2026-018 version 1 working draft saved");
    cleanup();
    renderLeadRoute("/lead-inspector/preliminary-reports/PR-2026-018", runtime);
    const remounted = await screen.findByTestId("lead-preliminary-report-workflow-page");
    await user.click(within(remounted).getByRole("button", { name: "Report Content" }));
    expect(within(remounted).getByLabelText("Executive Summary")).toHaveValue("Canonical FND-CAB-2026-001 summary for Lead review.");
    expect(within(remounted).getByLabelText("Comment to Auditee text")).toHaveValue("Auditee-visible canonical Finding comment.");
    expect(within(remounted).getByLabelText("Internal CAA Note text")).toHaveValue("Internal canonical Finding note.");
    await user.click(within(remounted).getByRole("button", { name: "Attachments" }));
    expect(within(remounted).getByRole("region", { name: "Attachment workspace" })).toHaveTextContent("PR-2026-018-working-draft.pdf");
    await user.click(within(remounted).getByRole("button", { name: "Review & Submit" }));
    expect(within(remounted).getByRole("region", { name: "Review workspace" })).toHaveTextContent("FND-CAB-2026-001");
    await user.click(within(remounted).getByRole("button", { name: "Preview working document" }));
    const preview = within(remounted).getByRole("region", { name: "PR-2026-018 working document preview" });
    expect(preview).toHaveTextContent("PRELIMINARY INSPECTION REPORT");
    expect(preview).toHaveTextContent("AUD-2026-001");
    expect(preview).toHaveTextContent("FND-CAB-2026-001");
    expect(preview).toHaveTextContent("Canonical FND-CAB-2026-001 summary for Lead review.");
  });

  it("direct-loads Final Reports without losing canonical report-version identity", async () => {
    renderLeadRoute("/lead-inspector/final-reports");

    const page = await screen.findByTestId("lead-final-reports-page");
    const report = await within(page).findByRole("article", { name: "Final Report RPT-CAB-2026-001" });
    expect(report).toHaveAttribute("data-report-version-id", "RPT-CAB-2026-001-V1");
    expect(report).toHaveAttribute("data-report-id", "RPT-CAB-2026-001");
    expect(within(report).getByText("RPT-CAB-2026-001")).toBeVisible();
    expect(within(report).getByText(/Version 1/)).toBeVisible();
    expect(within(report).getByRole("link", { name: "View RPT-CAB-2026-001 readiness" })).toHaveAttribute(
      "href",
      "/lead-inspector/final-reports/RPT-CAB-2026-001/readiness",
    );
  });

  it("renders a truthful empty Final Report list without requesting an obsolete fixture", async () => {
    const runtime = createMockBackendRuntime();
    const lead = runtime.backendForRole("leadInspector");
    vi.spyOn(lead.documents, "list").mockResolvedValue({ items: [], nextCursor: null });
    const getVersion = vi.spyOn(lead.reports, "getVersion").mockRejectedValue(new Error("Not found."));

    renderLeadRoute("/lead-inspector/final-reports", runtime);

    const page = await screen.findByTestId("lead-final-reports-page");
    expect(within(page).getByText("No Final Report versions are available yet.")).toBeVisible();
    expect(within(page).queryByRole("alert")).toBeNull();
    expect(getVersion).not.toHaveBeenCalled();
  });

  it("shows truthful readiness blockers and keeps approval authority outside Lead preparation", async () => {
    renderLeadRoute("/lead-inspector/final-reports/RPT-CAB-2026-001/readiness");

    const page = await screen.findByTestId("lead-final-report-readiness-page");
    await within(page).findByRole("region", { name: "Readiness blockers" });
    expect(page).toHaveAttribute("data-report-version-id", "RPT-CAB-2026-001-V1");
    expect(within(page).getByRole("heading", { name: "Final Report Preparation" })).toBeVisible();
    expect(within(page).getByRole("region", { name: "Readiness blockers" })).toHaveTextContent("Open Findings");
    expect(within(page).getByText(/CAP acceptance is not closure/i)).toBeVisible();
    expect(within(page).queryByRole("button", { name: /approve|issue|lock/i })).toBeNull();
    expect(within(page).getByRole("button", { name: "Prepare Final Report unavailable" })).toHaveAttribute(
      "title",
      "Report RPT-CAB-2026-001-V1 is at Executive Director Review; Lead Inspector preparation is read-only.",
    );
  });

  it("keeps Executive Director Review read-only and previews the same immutable backend version", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    const getVersion = vi.spyOn(runtime.backendForRole("leadInspector").reports, "getVersion");
    const decide = vi.spyOn(runtime.backendForRole("leadInspector").reports, "decide");
    renderLeadRoute("/lead-inspector/final-reports/RPT-CAB-2026-001/prepare", runtime);

    const page = await screen.findByTestId("lead-prepare-final-report-page");
    expect(await within(page).findByLabelText("Executive Summary")).toHaveAttribute("readonly");
    expect(within(page).getByRole("button", { name: "Save as Draft unavailable" })).toHaveAttribute(
      "title",
      "Report RPT-CAB-2026-001-V1 is at Executive Director Review; Lead Inspector cannot edit this immutable version.",
    );
    expect(getVersion).toHaveBeenCalledWith({ reportVersionId: "RPT-CAB-2026-001-V1" });
    expect(decide).not.toHaveBeenCalled();
    await user.click(within(page).getByRole("link", { name: "Preview Final Report" }));
    const documentPage = await screen.findByTestId("lead-final-report-document-page");
    await within(documentPage).findByText("RPT-CAB-2026-001-V1");
    expect(documentPage).toHaveAttribute(
      "data-report-version-id",
      "RPT-CAB-2026-001-V1",
    );
  });

  it("renders and prepares a download for the same Final Report document version", async () => {
    const user = userEvent.setup();
    renderLeadRoute("/lead-inspector/final-reports/RPT-CAB-2026-001/document");

    const page = await screen.findByTestId("lead-final-report-document-page");
    await within(page).findByText("RPT-CAB-2026-001-V1");
    expect(page).toHaveAttribute("data-report-version-id", "RPT-CAB-2026-001-V1");
    expect(within(page).getByRole("heading", { name: "Final Report" })).toBeVisible();
    expect(within(page).getByText("RPT-CAB-2026-001")).toBeVisible();
    expect(within(page).getByText("AUD-2026-001")).toBeVisible();
    const download = within(page).getByRole("link", { name: "Download report preview" });
    expect(download).toHaveClass(
      "workbench-page-header__action",
    );
    expect(download).toHaveAttribute("download", "AviaSurveil360_RPT-CAB-2026-001-V1_Report_Preview.txt");
    expect(decodeURIComponent(download.getAttribute("href") ?? "")).toContain("RPT-CAB-2026-001-V1");
    await user.click(download);
    expect(within(page).getByRole("status")).toHaveTextContent("RPT-CAB-2026-001-V1 preview prepared in the demo workspace");
  });

  it("navigates list to readiness, immutable snapshot, and document without Lead decision authority", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    const decide = vi.spyOn(runtime.backendForRole("leadInspector").reports, "decide");
    renderLeadRoute("/lead-inspector/final-reports", runtime);

    const list = await screen.findByTestId("lead-final-reports-page");
    await within(list).findByRole("article", { name: "Final Report RPT-CAB-2026-001" });
    expect(within(list).getByRole("link", { name: "View immutable preparation snapshot" })).toHaveAttribute("href", "/lead-inspector/final-reports/RPT-CAB-2026-001/prepare");
    expect(within(list).getByRole("link", { name: "View Final Report document" })).toHaveAttribute("href", "/lead-inspector/final-reports/RPT-CAB-2026-001/document");
    await user.click(within(list).getByRole("link", { name: "View RPT-CAB-2026-001 readiness" }));
    const readiness = await screen.findByTestId("lead-final-report-readiness-page");
    expect(within(readiness).getByRole("link", { name: "View immutable preparation snapshot" })).toHaveAttribute("href", "/lead-inspector/final-reports/RPT-CAB-2026-001/prepare");
    expect(within(readiness).getByRole("link", { name: "View Final Report document" })).toHaveAttribute("href", "/lead-inspector/final-reports/RPT-CAB-2026-001/document");
    await user.click(within(readiness).getByRole("link", { name: "View immutable preparation snapshot" }));
    const snapshot = await screen.findByTestId("lead-prepare-final-report-page");
    expect(await within(snapshot).findByLabelText("Executive Summary")).toHaveAttribute("readonly");
    await user.click(within(snapshot).getByRole("link", { name: "Preview Final Report" }));
    expect(await screen.findByTestId("lead-final-report-document-page")).toBeVisible();
    expect(decide).not.toHaveBeenCalled();
  });
});
