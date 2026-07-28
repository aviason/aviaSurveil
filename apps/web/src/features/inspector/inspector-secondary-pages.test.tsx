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

afterEach(cleanup);

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

function renderRoute(path: string, runtime: MockRuntime = createMockBackendRuntime()) {
  render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: "demo",
      environmentLabel: "test",
      identityMode: "demo-role-switch",
      subjectId: "USR-INSPECTOR-AMINA",
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

describe("Inspector secondary routes", () => {
  it("direct-loads Findings with the Inspector purpose, decision hierarchy, and working dossier action", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/findings");
    renderRoute("/inspector/findings", runtime);

    const page = await screen.findByTestId("inspector-findings-page");
    expect(within(page).getByRole("heading", { level: 1, name: "Findings" })).toBeVisible();
    expect(within(page).getByText("All findings and CAPs from this inspection")).toBeVisible();
    expect((await within(page).findAllByText("Current Owner")).length).toBeGreaterThan(0);
    expect(within(page).getAllByText("Next Action").length).toBeGreaterThan(0);
    expect(within(page).getAllByText("Due Date").length).toBeGreaterThan(0);
    expect(within(page).getByText("CAA Inspector workspace")).toBeInTheDocument();
    const selectedFinding = within(page).getByRole("article", { name: "Selected Finding CAB-2026-001" });
    expect(within(selectedFinding).getByRole("button", { name: "Accept CAP unavailable" })).toBeDisabled();
    expect(within(selectedFinding).getByRole("button", { name: "Return for Revision unavailable" })).toBeDisabled();
    await user.click(within(page).getAllByRole("link", { name: "Open Finding dossier" })[0]!);
    expect(await screen.findByRole("heading", { name: /Finding CAB-2026-011/ })).toBeVisible();
  });

  it("uses one exact backend Finding projection in the queue and selected dossier", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/findings");
    const expected = await runtime.backendForRole("inspector").findings.get({ findingId: "FND-CAB-2026-001" });
    renderRoute("/inspector/findings", runtime);

    const page = await screen.findByTestId("inspector-findings-page");
    const queueCard = within(page).getAllByRole("link", { name: "Open Finding dossier" })[0]!.closest("article");
    const selected = within(page).getByRole("article", { name: `Selected Finding ${expected.findingNumber}` });
    for (const projection of [queueCard, selected]) {
      expect(projection).toHaveAttribute("data-finding-id", expected.id);
      expect(projection).toHaveAttribute("data-current-owner-role", expected.currentOwnerRole);
      expect(projection).toHaveAttribute("data-next-action", expected.nextAction);
      expect(projection).toHaveAttribute("data-due-date", expected.dueDate);
    }
    expect(within(selected).getByText("Lead Inspector")).toBeVisible();
    expect(within(selected).getByText(expected.nextAction)).toBeVisible();
  });

  it("presents the accepted nine-record Finding queue total before filters are applied", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/findings");
    renderRoute("/inspector/findings", runtime);

    const page = await screen.findByTestId("inspector-findings-page");
    const queue = within(page).getByRole("region", { name: "Finding Queue" });
    expect(within(queue).getByText("9 findings")).toBeVisible();
  });

  it("exposes the selected Finding dossier sections as named navigation", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/findings");
    renderRoute("/inspector/findings", runtime);

    const page = await screen.findByTestId("inspector-findings-page");
    const selectedFinding = within(page).getByRole("article", { name: "Selected Finding CAB-2026-001" });
    expect(within(selectedFinding).getByRole("navigation", { name: "Finding dossier sections" })).toBeVisible();
  });

  it("exports the currently visible Finding queue as a downloadable CSV artifact", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/findings");
    renderRoute("/inspector/findings", runtime);

    const page = await screen.findByTestId("inspector-findings-page");
    const exportLink = within(page).getByRole("link", { name: /Export/ });
    expect(exportLink).toHaveAttribute("download", "AviaSurveil360_Inspector_Findings.csv");
    expect(exportLink.getAttribute("href")).toMatch(/^data:text\/csv;charset=utf-8,/);
    expect(decodeURIComponent(exportLink.getAttribute("href") ?? "")).toContain("CAB-2026-001");
  });

  it("preserves the exact backend Finding identity when the CAB dossier record is absent", async () => {
    renderRoute("/inspector/findings");

    const page = await screen.findByTestId("inspector-findings-page");
    expect(within(page).getByText("CAR-2026-099")).toBeVisible();
    expect(within(page).queryByText("CAB-2026-011")).toBeNull();
    expect(within(page).queryByRole("link", { name: "Open Finding dossier" })).toBeNull();
    expect(within(page).getByRole("button", { name: "Finding dossier unavailable for CAR-2026-099" })).toHaveAttribute(
      "title",
      "Finding CAR-2026-099 does not have a declared Inspector Finding Detail route.",
    );
  });

  it("switches every selected Finding dossier section to a visible record-specific outcome", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/findings");
    renderRoute("/inspector/findings", runtime);

    const page = await screen.findByTestId("inspector-findings-page");
    const selectedFinding = within(page).getByRole("article", { name: "Selected Finding CAB-2026-001" });
    for (const [tab, outcome] of [
      ["Details", "Finding details for CAB-2026-001"],
      ["Conversation 2", "Conversation for CAB-2026-001"],
      ["Files 3", "Files for CAB-2026-001"],
      ["History", "History for CAB-2026-001"],
      ["CAP & Verification", "CAP and verification for CAB-2026-001"],
    ] as const) {
      await user.click(within(selectedFinding).getByRole("button", { name: tab }));
      expect(within(selectedFinding).getByRole("region", { name: outcome })).toBeVisible();
    }
  });

  it("opens only the frozen CAB Finding row and explicitly disables every other record", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/findings");
    renderRoute("/inspector/findings", runtime);

    const page = await screen.findByTestId("inspector-findings-page");
    const records = await within(page).findAllByRole("article");
    const cab = records.find((record) => within(record).queryByText("CAB-2026-001"));
    const skyCargo = records.find((record) => within(record).queryByText("CAR-2026-099"));
    expect(cab).toBeDefined();
    expect(skyCargo).toBeDefined();
    expect(within(cab!).getByRole("link", { name: "Open Finding dossier" })).toHaveAttribute("href", "/inspector/findings/FND-CAB-2026-001");
    expect(within(skyCargo!).queryByRole("link")).toBeNull();
    expect(within(skyCargo!).getByRole("button", { name: "Finding dossier unavailable for CAR-2026-099" })).toHaveAttribute(
      "title",
      "Finding CAR-2026-099 does not have a declared Inspector Finding Detail route.",
    );
  });

  it("applies distinct KPI, CAP Level, CAP Status, Due Date, and Reset filters to visible records", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/findings");
    renderRoute("/inspector/findings", runtime);

    const page = await screen.findByTestId("inspector-findings-page");
    const queue = within(page).getByRole("region", { name: "Finding Queue" });
    const recordGrid = queue.querySelector<HTMLElement>(".inspector-record-grid");
    if (!recordGrid) throw new Error("Expected Finding record grid.");
    expect(within(page).getByRole("button", { name: "Reset Finding filters unavailable" })).toBeDisabled();
    expect(within(page).getByRole("button", { name: "Reset Finding filters unavailable" })).toHaveAttribute(
      "title",
      "Finding filters are already at their defaults.",
    );

    await user.click(within(page).getByRole("button", { name: /CAP Submitted/ }));
    expect(within(page).getByRole("button", { name: "Reset" })).toBeEnabled();
    expect(within(page).getByRole("button", { name: /CAP Submitted/ })).toHaveAttribute("aria-pressed", "true");
    expect(within(page).getByRole("button", { name: /All Findings/ })).toHaveAttribute("aria-pressed", "false");
    expect(within(queue).getByText("1 findings")).toBeVisible();
    expect(within(recordGrid).getByText("CAB-2026-001")).toBeVisible();
    expect(within(recordGrid).queryByText("CAR-2026-099")).toBeNull();

    await user.click(within(page).getByRole("button", { name: /Waiting for CAP/ }));
    expect(within(page).getByRole("button", { name: /Waiting for CAP/ })).toHaveAttribute("aria-pressed", "true");
    expect(within(queue).getByText("0 findings")).toBeVisible();

    await user.click(within(page).getByRole("button", { name: /All Findings/ }));
    await user.selectOptions(within(page).getByLabelText("CAP Level"), "LEVEL_2_MAJOR");
    expect(within(queue).getByText("1 findings")).toBeVisible();
    expect(within(recordGrid).getByText("CAR-2026-099")).toBeVisible();

    await user.selectOptions(within(page).getByLabelText("CAP Level"), "all");
    await user.selectOptions(within(page).getByLabelText("CAP Status"), "CAP_SUBMITTED");
    expect(within(queue).getByText("1 findings")).toBeVisible();
    expect(within(recordGrid).getByText("CAB-2026-001")).toBeVisible();
    expect(within(recordGrid).queryByText("CAR-2026-099")).toBeNull();

    await user.selectOptions(within(page).getByLabelText("CAP Status"), "all");
    await user.selectOptions(within(page).getByLabelText("Due Date"), "overdue");
    expect(within(queue).getByText("1 findings")).toBeVisible();
    expect(within(recordGrid).getByText("CAR-2026-099")).toBeVisible();

    await user.click(within(page).getByRole("button", { name: "Reset" }));
    expect(within(queue).getByText("9 findings")).toBeVisible();
    expect(within(page).getByLabelText("CAP Level")).toHaveValue("all");
    expect(within(page).getByLabelText("CAP Status")).toHaveValue("all");
    expect(within(page).getByLabelText("Due Date")).toHaveValue("all");
    expect(within(page).getByRole("button", { name: "Reset Finding filters unavailable" })).toBeDisabled();
  });

  it("direct-loads the role-safe Message Center and records a composed in-app message", async () => {
    const user = userEvent.setup();
    renderRoute("/inspector/messages");

    const page = await screen.findByTestId("inspector-messages-page");
    expect(within(page).getByRole("heading", { name: "Messages from the CAA" })).toBeVisible();
    expect(within(page).getByText(/In-app notifications/)).toBeVisible();
    expect(within(page).getByText(/CAA Inspector workspace only/)).toBeVisible();
    expect(await within(page).findByText("Email delivery: Not configured in demo")).toBeVisible();
    await user.click(within(page).getByRole("button", { name: "Compose message" }));
    await user.type(within(page).getByLabelText("Subject"), "Cabin Inspection follow-up");
    await user.type(within(page).getByLabelText("Message"), "Please review the configured evidence request.");
    await user.click(within(page).getByRole("button", { name: "Send in-app message" }));
    expect(await within(page).findByText("Message recorded in the demo workspace.")).toBeVisible();
  });

  it("scopes the Audit Work Queue to the signed-in Inspector and opens the exact checklist route", async () => {
    const user = userEvent.setup();
    renderRoute("/inspector/calendar");

    const page = await screen.findByTestId("inspector-calendar-page");
    expect(within(page).getByRole("heading", { name: "Audit Work Queue" })).toBeVisible();
    const cards = await within(page).findAllByRole("article");
    const flyNamibia = cards.find((card) => within(card).queryByText(/2026 Cabin Inspection/));
    expect(flyNamibia).toBeDefined();
    expect(within(page).queryByText(/SkyCargo Air/)).toBeNull();
    expect(within(flyNamibia!).getByText("Current Owner")).toBeVisible();
    expect(within(flyNamibia!).getByText("Next Action")).toBeVisible();
    expect(within(flyNamibia!).getByText("Due Date")).toBeVisible();
    const continueChecklist = within(flyNamibia!).getByRole("link", { name: "Continue checklist" });
    expect(continueChecklist).toHaveAttribute("href", "/inspector/audits/AUD-2026-001/checklist");
    await user.click(continueChecklist);
    expect(await screen.findByRole("heading", { name: "Cabin Inspection checklist" })).toBeVisible();
  });

  it("shows mobile-first decision order and a visible completed-filter outcome", async () => {
    const user = userEvent.setup();
    renderRoute("/inspector/calendar");
    const page = await screen.findByTestId("inspector-calendar-page");
    const card = (await within(page).findAllByRole("article"))[0]!;
    const title = within(card).getByRole("heading");
    const owner = within(card).getByText("Current Owner");
    const action = within(card).getByText("Next Action");
    const dueDate = within(card).getByText("Due Date");
    expect(title.compareDocumentPosition(owner) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(owner.compareDocumentPosition(action) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(action.compareDocumentPosition(dueDate) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(within(page).getByRole("button", { name: /Active audits/ })).toHaveAttribute("aria-pressed", "true");
    expect(within(page).getByRole("button", { name: /Completed/ })).toHaveAttribute("aria-pressed", "false");
    await user.click(within(page).getByRole("button", { name: /Completed/ }));
    expect(within(page).getByRole("button", { name: /Active audits/ })).toHaveAttribute("aria-pressed", "false");
    expect(within(page).getByRole("button", { name: /Completed/ })).toHaveAttribute("aria-pressed", "true");
    expect(within(page).getByText("No completed audits in this deterministic projection.")).toBeVisible();
  });

  it("keeps an open Finding report as a pending draft while preserving its report identity", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/closure-reports/CR-CAB-2026-001");
    renderRoute("/inspector/reports", runtime);

    const page = await screen.findByTestId("inspector-reports-page");
    expect(within(page).getByRole("heading", { name: "Reports" })).toBeVisible();
    expect(within(page).getByText("Report previews and historical closure reports (view only).")).toBeVisible();
    expect(within(page).getByRole("region", { name: "Report previews and historical reports" })).toBeVisible();
    expect(within(page).queryByRole("region", { name: "Past closure reports" })).toBeNull();
    expect(within(page).getAllByText("Next Action").length).toBeGreaterThan(0);
    expect(within(page).getAllByText("Due Date").length).toBeGreaterThan(0);
    const reports = within(page).getAllByRole("article");
    const cab = reports.find((report) => within(report).queryAllByText(/CAB-2026-011/).length > 0);
    expect(cab).toBeDefined();
    expect(within(cab!).queryByText("Closed")).toBeNull();
    expect(within(cab!).getAllByText(/CAP Submitted|Pending CAA Review|Draft preview/i).length).toBeGreaterThan(0);
    await user.click(within(cab!).getByRole("link", { name: "Preview CAB-2026-011 draft report" }));
    expect(await screen.findByRole("heading", { level: 1, name: "Finding Report — CAB-2026-011" })).toBeVisible();
    const preview = screen.getByTestId("closure-report-page");
    expect(within(preview).queryByText("Finding Closure Report")).toBeNull();
    expect(within(preview).getAllByText(/CAP Submitted|Pending CAA Review|Draft/i).length).toBeGreaterThan(0);

    cleanup();
    renderRoute("/inspector/reports");
    const remounted = await screen.findByTestId("inspector-reports-page");
    for (const report of within(remounted).getAllByRole("article").filter((row) => within(row).queryAllByText(/CAB-2026-011/).length === 0)) {
      const reportId = within(report).getByText(/^(?:CAB|OPS)-\d{4}-\d{3}/).textContent?.split(" · ")[0];
      expect(within(report).getByRole("button", { name: `Report preview unavailable for ${reportId}` })).toHaveAttribute(
        "title",
        `Report ${reportId} does not have a declared Inspector report-detail route.`,
      );
    }
  });

  it("labels a closed CAB record as a closure report instead of a draft", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/closure-reports/CR-CAB-2026-001");
    const manager = runtime.backendForRole("manager");
    const finding = await manager.findings.get({ findingId: "FND-CAB-2026-001" });
    await manager.findings.authorizedClose({
      operationId: "OP-REPORT-LABEL-CLOSE",
      findingId: finding.id,
      expectedFindingRevision: finding.revision,
      reason: "Authorized closure for lifecycle-label regression coverage.",
    });
    renderRoute("/inspector/reports", runtime);

    const page = await screen.findByTestId("inspector-reports-page");
    expect(await within(page).findByRole("region", { name: "Past closure reports" })).toBeVisible();
    expect(within(page).getByRole("link", { name: "Preview CAB-2026-011 closure report" })).toBeVisible();
    expect(within(page).queryByRole("link", { name: "Preview CAB-2026-011 draft report" })).toBeNull();
  });

  it("direct-loads the pending mock report and gives export a visible durable effect", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/closure-reports/CR-CAB-2026-001");
    renderRoute("/inspector/closure-reports/CR-CAB-2026-001", runtime);

    const page = await screen.findByTestId("closure-report-page");
    expect(within(page).getByRole("heading", { name: "Finding Report — CAB-2026-011" })).toBeVisible();
    expect(within(page).getByText("Finding Report Draft")).toBeVisible();
    expect(within(page).queryByText("Finding Closure Report")).toBeNull();
    expect(within(page).getByText(/mock — not a legally issued document/i)).toBeVisible();
    const download = within(page).getByRole("link", { name: "Download report preview" });
    expect(download).toHaveAttribute("download", "AviaSurveil360_CAB-2026-011_Report_Preview.txt");
    expect(decodeURIComponent(download.getAttribute("href") ?? "")).toContain("CAB-2026-001");
    await user.click(download);
    expect(within(page).getByRole("status")).toHaveTextContent("Mock report preview prepared for download");
  });

  it("direct-loads the advisory-only Inspector Assistant and labels generated output as Draft", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    const createDraft = vi.spyOn(runtime.backendForRole("inspector").assistantDrafts!, "createDraft");
    renderRoute("/inspector/assistant", runtime);

    const page = await screen.findByTestId("inspector-assistant-page");
    expect(within(page).getByRole("heading", { name: "AI Inspector Assistant Panel" })).toBeVisible();
    expect(within(page).getAllByText(/requires authorized review/i).length).toBeGreaterThan(0);
    expect(await within(page).findByText(/CAR-2026-099 · Cargo restraint record needs follow-up/)).toBeVisible();
    expect(within(page).getByText("SkyCargo Air")).toBeVisible();
    const suggestions = within(page).getByRole("region", { name: "Assistant suggestions" });
    expect(within(suggestions).getAllByText(/CAR-2026-099|Cargo restraint record needs follow-up|SkyCargo Air/).length).toBeGreaterThan(0);
    expect(suggestions).not.toHaveTextContent(/CAB EMEQ|PBE serviceability|RISK-ORG-XYZ|closure evidence/i);
    await user.clear(within(page).getByLabelText("Draft request"));
    await user.type(within(page).getByLabelText("Draft request"), "Summarize the configured finding basis.");
    await user.click(within(page).getByRole("button", { name: "Create Draft" }));
    const output = await within(page).findByRole("status", { name: "Assistant Draft" });
    expect(output).toHaveAttribute("data-durable-outcome", "assistant-draft");
    expect(createDraft).toHaveBeenCalledWith(expect.objectContaining({ findingId: "FND-SKYCARGO-2026-099" }));
    expect(output).toHaveTextContent("Draft");
    expect(output).toHaveTextContent("CAR-2026-099");
    expect(output).not.toHaveTextContent(/final decision|approved decision/i);
  });

  it("keeps the seeded CAB Assistant display and draft command bound to the same Finding", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/assistant");
    const createDraft = vi.spyOn(runtime.backendForRole("inspector").assistantDrafts!, "createDraft");
    renderRoute("/inspector/assistant", runtime);

    const page = await screen.findByTestId("inspector-assistant-page");
    expect(await within(page).findByText("CAB-2026-011 · Emergency equipment serviceability record incomplete")).toBeVisible();
    expect(within(page).getByText("Fly Namibia")).toBeVisible();
    await user.click(within(page).getByRole("button", { name: "Create Draft" }));
    expect(createDraft).toHaveBeenCalledWith(expect.objectContaining({ findingId: "FND-CAB-2026-001" }));
  });

  it("keeps a transitioned CAB lifecycle status and Draft command aligned to the exact Finding", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/assistant");
    const lead = runtime.backendForRole("leadInspector");
    const beforeReview = await lead.findings.get({ findingId: "FND-CAB-2026-001" });
    const cap = (await lead.caps.listRevisions({ findingId: beforeReview.id })).items.at(-1);
    if (!cap) throw new Error("Expected submitted CAB CAP revision.");
    await lead.caps.review({
      operationId: "OP-ASSISTANT-CAB-MORE-INFORMATION",
      capRevisionId: cap.id,
      expectedCapRevision: cap.revision,
      findingId: beforeReview.id,
      expectedFindingRevision: beforeReview.revision,
      decision: "REQUEST_MORE_INFORMATION",
      commentToAuditee: "Provide the missing CAP detail.",
      internalCaaNote: "CAB lifecycle transition regression coverage.",
    });
    const transitioned = await lead.findings.get({ findingId: beforeReview.id });
    const createDraft = vi.spyOn(runtime.backendForRole("inspector").assistantDrafts!, "createDraft");
    renderRoute("/inspector/assistant", runtime);

    const page = await screen.findByTestId("inspector-assistant-page");
    await within(page).findByText(/CAB-2026-011 · Emergency equipment serviceability record incomplete/);
    const context = page.querySelector<HTMLElement>(".inspector-assistant-context");
    if (!context) throw new Error("Expected Assistant Finding context.");
    expect(context).toHaveAttribute("data-finding-id", transitioned.id);
    expect(within(context).getByText(transitioned.status.replaceAll("_", " "))).toBeVisible();
    await user.click(within(page).getByRole("button", { name: "Create Draft" }));
    expect(createDraft).toHaveBeenCalledWith(expect.objectContaining({ findingId: transitioned.id }));
  });

  it("records an authorized review outcome for a visible Assistant suggestion", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/inspector/assistant");
    renderRoute("/inspector/assistant", runtime);

    const page = await screen.findByTestId("inspector-assistant-page");
    const suggestion = await within(page).findByRole("article", { name: "Draft finding language for PBE serviceability" });
    const accept = within(suggestion).getByRole("button", { name: /Accept draft:/ });
    const edit = within(suggestion).getByRole("button", { name: /Record edit:/ });
    const reject = within(suggestion).getByRole("button", { name: /Reject:/ });
    expect(accept).toHaveAttribute("aria-pressed", "false");
    expect(edit).toHaveAttribute("aria-pressed", "false");
    expect(reject).toHaveAttribute("aria-pressed", "false");
    await user.click(accept);
    expect(accept).toHaveAttribute("aria-pressed", "true");
    expect(edit).toHaveAttribute("aria-pressed", "false");
    expect(reject).toHaveAttribute("aria-pressed", "false");
    expect(within(suggestion).getByRole("status")).toHaveTextContent("Accepted by authorized Inspector");
  });

  it("direct-loads the Inspector Profile and persists a role-safe display-name update", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    renderRoute("/inspector/profile", runtime);

    const page = await screen.findByTestId("inspector-profile-page");
    expect(within(page).getByRole("heading", { name: "Profile" })).toBeVisible();
    expect(await within(page).findByText("Amina Inspector")).toBeVisible();
    expect(within(page).getByText("Inspector Workspace")).toBeVisible();
    expect(within(page).queryByLabelText("Display name")).toBeNull();
    await user.click(within(page).getByRole("button", { name: "Edit profile" }));
    expect(within(page).getByDisplayValue("Amina Inspector")).toBeVisible();
    await user.clear(within(page).getByLabelText("Display name"));
    await user.type(within(page).getByLabelText("Display name"), "Aylin Sezer Updated");
    await user.click(within(page).getByRole("button", { name: "Save profile" }));
    expect(await within(page).findByText("Profile saved in the demo workspace.")).toBeVisible();
    cleanup();
    renderRoute("/inspector/profile", runtime);
    const remounted = await screen.findByTestId("inspector-profile-page");
    expect(await within(remounted).findByText("Aylin Sezer Updated")).toBeVisible();
  });
});
