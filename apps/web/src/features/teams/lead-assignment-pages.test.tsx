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
      <ScenarioProvider>
        <MemoryRouter initialEntries={[path]}><AppRouter /></MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
  return runtime;
}

describe("Lead Inspector assignment and secondary routes", () => {
  it("direct-loads the exact Audit assignment with workload and record-specific transitions", async () => {
    renderLeadRoute("/lead-inspector/audits/AUD-2026-001/assignment");

    const page = await screen.findByTestId("lead-audit-assignment-page");
    expect(await within(page).findByRole("heading", { name: "2026 Cabin Inspection - Fly Namibia" })).toBeVisible();
    expect(screen.getByTestId("lead-audit-assignment-page")).toHaveAttribute("data-audit-id", "AUD-2026-001");
    expect(await within(page).findByRole("region", { name: "Inspector workload" })).toHaveTextContent("Amina Inspector");
    const summary = await within(page).findByRole("region", { name: "Audit assignment summary" });
    const owner = within(summary).getByText("Current Owner");
    const nextAction = within(summary).getByText("Next Action");
    const dueDate = within(summary).getByText("Due Date");
    expect(owner.compareDocumentPosition(nextAction) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(nextAction.compareDocumentPosition(dueDate) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(within(page).getByRole("link", { name: "Assign Checklist Questions" })).toHaveAttribute(
      "href",
      "/lead-inspector/audits/AUD-2026-001/checklist-questions",
    );
    expect(within(page).getByRole("link", { name: "View Preliminary Reports" })).toHaveAttribute(
      "href",
      "/lead-inspector/preliminary-reports",
    );
    expect(within(page).getByRole("link", { name: "View Preliminary Reports" })).toHaveClass(
      "workbench-page-header__action",
    );
  });

  it("loads all six exact question IDs and applies a real section filter", async () => {
    const user = userEvent.setup();
    renderLeadRoute("/lead-inspector/audits/AUD-2026-001/checklist-questions");

    const page = await screen.findByTestId("lead-question-assignment-page");
    await within(page).findByText("CAB-GALLEY-001");
    expect(screen.getByTestId("lead-question-assignment-page")).toHaveAttribute("data-audit-id", "AUD-2026-001");
    for (const questionId of [
      "CAB-GALLEY-001",
      "CAB-LAV-001",
      "CAB-PAX-SEAT-001",
      "CAB-EMEQ-PBE-001",
      "CAB-VID-CREW-SEAT-001",
      "CAB-COCKPIT-GEN-001",
    ]) expect(within(page).getByText(questionId)).toBeVisible();
    await user.selectOptions(within(page).getByLabelText("Section"), "EM EQ / PBE");
    expect(within(page).getByText("CAB-EMEQ-PBE-001")).toBeVisible();
    expect(within(page).queryByText("CAB-GALLEY-001")).toBeNull();
  });

  it("keeps materialized question views read-only and hands preparation to the canonical route", async () => {
    const runtime = createMockBackendRuntime();
    const decide = vi.spyOn(runtime.backendForRole("leadInspector").reports, "decide");
    renderLeadRoute("/lead-inspector/audits/AUD-2026-001/checklist-questions", runtime);

    const page = await screen.findByTestId("lead-question-assignment-page");
    expect(within(page).queryByRole("button", { name: "Assign Questions" })).toBeNull();
    expect(within(page).getByRole("link", { name: "Open Lead preparation workspace" })).toHaveAttribute(
      "href",
      "/lead-inspector/audit-preparation?assignmentId=AUD-2026-001%3Aassignment",
    );
    expect(within(page).getByText(/question assignment cannot change report approval authority/i)).toBeVisible();
    expect(decide).not.toHaveBeenCalled();
  });

  it("reuses role-safe Calendar and Messages with exact routes and separated visibility", async () => {
    const user = userEvent.setup();
    renderLeadRoute("/lead-inspector/calendar");
    const calendar = await screen.findByTestId("lead-calendar-page");
    expect(await within(calendar).findByRole("link", { name: "Open assignment for AUD-2026-001" })).toHaveAttribute(
      "href",
      "/lead-inspector/audits/AUD-2026-001/assignment",
    );
    expect(within(calendar).getByRole("link", { name: "Open assignment for AUD-2026-099" })).toHaveAttribute(
      "href",
      "/lead-inspector/audits/AUD-2026-099/assignment",
    );

    cleanup();
    renderLeadRoute("/lead-inspector/messages");
    const messages = await screen.findByTestId("lead-messages-page");
    expect(within(messages).getByRole("region", { name: "Internal CAA Note messages" })).toBeVisible();
    expect(within(messages).getByRole("region", { name: "Comment to Auditee messages" })).toBeVisible();
    await user.click(within(messages).getByRole("button", { name: "Compose message" }));
    await user.selectOptions(within(messages).getByLabelText("Visibility"), "AUDITEE");
    await user.type(within(messages).getByLabelText("Subject"), "PR-2026-018 clarification");
    await user.type(within(messages).getByLabelText("Message"), "Please confirm the configured report fact.");
    await user.click(within(messages).getByRole("button", { name: "Send in-app message" }));
    expect(within(messages).getByRole("region", { name: "Comment to Auditee messages" })).toHaveTextContent("PR-2026-018 clarification");
  });

  it("direct-loads advisory Analytics and reusable Settings with visible action outcomes", async () => {
    const user = userEvent.setup();
    renderLeadRoute("/lead-inspector/analytics-reports");
    const analytics = await screen.findByTestId("lead-analytics-page");
    expect(within(analytics).getByRole("heading", { name: "Safety Intelligence Dashboard" })).toBeVisible();
    expect(within(analytics).getByText(/not a legal decision/i)).toBeVisible();
    await user.click(within(analytics).getByRole("button", { name: "Repeat" }));
    expect(within(analytics).getByRole("button", { name: "Repeat" })).toHaveAttribute("aria-pressed", "true");
    expect(within(analytics).getByRole("button", { name: "All signals" })).toHaveAttribute("aria-pressed", "false");
    const download = within(analytics).getByRole("link", { name: "Download analytics CSV" });
    expect(download).toHaveAttribute("download", "AviaSurveil360_Lead_Analytics.csv");
    expect(decodeURIComponent(download.getAttribute("href") ?? "")).toContain("ORG-FLY-NAMIBIA");
    expect(decodeURIComponent(download.getAttribute("href") ?? "")).not.toContain("ORG-SKYCARGO");
    await user.click(download);
    expect(within(analytics).getByRole("status")).toHaveTextContent("lead-analytics.csv prepared");

    cleanup();
    const runtime = createMockBackendRuntime();
    renderLeadRoute("/lead-inspector/settings", runtime);
    const settings = await screen.findByTestId("lead-settings-page");
    expect(within(settings).getByRole("heading", { name: "Settings" })).toBeVisible();
    expect(within(settings).getAllByText(/Configured rules/).length).toBeGreaterThan(0);
    await user.click(within(settings).getByRole("button", { name: "Edit profile" }));
    await user.clear(within(settings).getByLabelText("Display name"));
    await user.type(within(settings).getByLabelText("Display name"), "Caner Yildiz");
    await user.click(within(settings).getByRole("button", { name: "Save profile" }));
    expect(within(settings).getByRole("status")).toHaveTextContent("Profile saved in the demo workspace");
    cleanup();
    renderLeadRoute("/lead-inspector/settings", runtime);
    const remounted = await screen.findByTestId("lead-settings-page");
    expect(within(remounted).getByText("Caner Yildiz")).toBeVisible();
  });

  it.each([1440, 900, 390])("keeps Lead analytics hierarchy and advisory controls usable at %ipx", async (width) => {
    Object.defineProperty(window, "innerWidth", { configurable: true, value: width });
    renderLeadRoute("/lead-inspector/analytics-reports");
    const analytics = await screen.findByTestId("lead-analytics-page");
    const summary = within(analytics).getByRole("region", { name: "Lead management summary" });
    const attention = within(analytics).getByRole("region", { name: "Management attention" });
    const dossiers = within(analytics).getByRole("region", { name: "Management Signal Dossiers" });
    expect(summary.compareDocumentPosition(attention) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(attention.compareDocumentPosition(dossiers) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(summary).toHaveTextContent("Open Findings");
    expect(attention).toHaveTextContent("Current Owner");
    expect(attention).toHaveTextContent("Next Action");
    expect(attention).toHaveTextContent("Blocking Reason");
    for (const organizationId of ["ORG-SKYCARGO", "ORG-FLY-NAMIBIA"]) {
      expect(within(analytics).getByRole("button", { name: `Risk profile unavailable for ${organizationId}` })).toHaveAttribute(
        "title",
        `Organization ${organizationId} has no declared Lead Inspector risk-profile route.`,
      );
    }
    expect(within(analytics).getByRole("link", { name: "Download analytics CSV" })).toHaveAttribute(
      "download",
      "AviaSurveil360_Lead_Analytics.csv",
    );
  });
});
