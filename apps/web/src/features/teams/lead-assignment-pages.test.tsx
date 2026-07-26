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
    expect(page).toHaveAttribute("data-audit-id", "AUD-2026-001");
    expect(within(page).getByRole("heading", { name: "Cabin Inspection" })).toBeVisible();
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
    expect(within(page).getByRole("link", { name: "View Preliminary Report" })).toHaveAttribute(
      "href",
      "/lead-inspector/preliminary-reports/PR-2026-018",
    );
    expect(within(page).getByRole("link", { name: "View Preliminary Report" })).toHaveClass(
      "workbench-page-header__action",
    );
  });

  it("loads all six exact question IDs and applies a real section filter", async () => {
    const user = userEvent.setup();
    renderLeadRoute("/lead-inspector/audits/AUD-2026-001/checklist-questions");

    const page = await screen.findByTestId("lead-question-assignment-page");
    expect(page).toHaveAttribute("data-audit-id", "AUD-2026-001");
    await within(page).findByText("CAB-GALLEY-001");
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

  it("preserves selected question IDs in the local assignment and never invokes report approval", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    const decide = vi.spyOn(runtime.backendForRole("leadInspector").reports, "decide");
    renderLeadRoute("/lead-inspector/audits/AUD-2026-001/checklist-questions", runtime);

    const page = await screen.findByTestId("lead-question-assignment-page");
    expect(within(page).getByRole("button", { name: "Assign Questions" })).toHaveAttribute(
      "title",
      "Select at least one question from AUD-2026-001 before assigning.",
    );
    await user.click(within(page).getByLabelText("Select question CAB-GALLEY-001"));
    await user.click(within(page).getByLabelText("Select question CAB-EMEQ-PBE-001"));
    await user.selectOptions(within(page).getByLabelText("Assign To"), "USR-INSPECTOR-DAVID");
    await user.clear(within(page).getByLabelText("Assignment Due Date"));
    await user.type(within(page).getByLabelText("Assignment Due Date"), "2026-07-31");
    await user.selectOptions(within(page).getByLabelText("Assignment Priority"), "CRITICAL");
    await user.type(within(page).getByLabelText("Assignment Instructions"), "Verify exact emergency equipment records.");
    await user.click(within(page).getByRole("button", { name: "Assign Questions" }));
    const result = within(page).getByRole("status");
    expect(result).toHaveTextContent("AUD-2026-001");
    expect(result).toHaveTextContent("CAB-GALLEY-001, CAB-EMEQ-PBE-001");
    expect(result).toHaveTextContent("USR-INSPECTOR-DAVID");
    const saved = within(page).getByRole("region", { name: "Saved question assignments" });
    expect(saved).toHaveTextContent("CAB-GALLEY-001");
    expect(saved).toHaveTextContent("CAB-EMEQ-PBE-001");
    expect(saved).toHaveTextContent("2026-07-31");
    expect(saved).toHaveTextContent("Critical");
    expect(saved).toHaveTextContent("Verify exact emergency equipment records.");
    expect(within(page).getByRole("region", { name: "Inspector workload" })).toHaveTextContent("David Inspector");
    expect(decide).not.toHaveBeenCalled();

    cleanup();
    renderLeadRoute("/lead-inspector/audits/AUD-2026-001/checklist-questions", runtime);
    const remounted = await screen.findByTestId("lead-question-assignment-page");
    const persisted = await within(remounted).findByRole("region", { name: "Saved question assignments" });
    expect(persisted).toHaveTextContent("AUD-2026-001");
    expect(persisted).toHaveTextContent("PKG-CAB-2026-001");
    expect(persisted).toHaveTextContent("USR-INSPECTOR-DAVID");
    await user.selectOptions(within(remounted).getByLabelText("Priority filter"), "CRITICAL");
    const questionRegion = within(remounted).getByRole("region", { name: "Checklist questions" });
    expect(within(questionRegion).getByText("CAB-GALLEY-001")).toBeVisible();
    expect(within(questionRegion).queryByText("CAB-LAV-001")).toBeNull();
    await user.selectOptions(within(remounted).getByLabelText("Status filter"), "SAVED");
    expect(within(questionRegion).getByText("CAB-EMEQ-PBE-001")).toBeVisible();
    expect(within(remounted).getByLabelText("Department filter")).toBeEnabled();
    expect(within(remounted).getByLabelText("Risk filter")).toBeEnabled();
  });

  it("reuses role-safe Calendar and Messages with exact routes and separated visibility", async () => {
    const user = userEvent.setup();
    renderLeadRoute("/lead-inspector/calendar");
    const calendar = await screen.findByTestId("lead-calendar-page");
    expect(await within(calendar).findByRole("link", { name: "Open assignment for AUD-2026-001" })).toHaveAttribute(
      "href",
      "/lead-inspector/audits/AUD-2026-001/assignment",
    );
    expect(within(calendar).getByRole("button", { name: "Assignment unavailable for AUD-2026-099" })).toHaveAttribute(
      "title",
      "Audit AUD-2026-099 has no declared Lead Inspector assignment route.",
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
