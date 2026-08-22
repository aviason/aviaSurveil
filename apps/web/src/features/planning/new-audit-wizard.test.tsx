// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { AppRouter } from "../../app/router";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendPersistentRuntime, createMockBackendRuntime } from "../../mock/create-mock-backend";

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

beforeEach(() => localStorage.clear());
afterEach(cleanup);

function renderWizardRoute(path: string, runtime: MockRuntime = createMockBackendRuntime()) {
  render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: "demo",
      environmentLabel: "test",
      identityMode: "demo-role-switch",
      subjectId: "USR-MANAGER-NORA",
    }}>
      <ScenarioProvider>
        <MemoryRouter initialEntries={[path]}><AppRouter /></MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
  return runtime;
}

async function createDraft(user: ReturnType<typeof userEvent.setup>) {
  await screen.findByRole("combobox", { name: "Inspected Organization" });
  await waitFor(() => expect(screen.getByRole("button", { name: "Continue" })).toBeEnabled());
  await user.click(screen.getByRole("button", { name: "Continue" }));
  await screen.findByRole("heading", { level: 2, name: "Purpose" });
  await screen.findByRole("textbox", { name: "Purpose" });
}

async function completeThroughResources(user: ReturnType<typeof userEvent.setup>) {
  await createDraft(user);
  await user.type(screen.getByRole("textbox", { name: "Purpose" }), "Confirm the operating controls remain effective.");
  await user.click(screen.getByRole("button", { name: "Continue" }));
  await screen.findByRole("heading", { level: 2, name: "Schedule" });
  await screen.findByLabelText("Planned date");
  fireEvent.change(screen.getByLabelText("Planned date"), { target: { value: "2026-12-10" } });
  await user.click(screen.getByRole("button", { name: "Continue" }));
  await screen.findByRole("heading", { level: 2, name: "Resources & budget" });
  await screen.findByRole("spinbutton", { name: "Required inspectors" });
}

describe("New Audit planning proposal", () => {
  it("uses the current shell with one five-step New Audit rhythm and user-owned fields", async () => {
    renderWizardRoute("/department-manager/new-audit/step-1");
    await screen.findByRole("heading", { level: 1, name: "New Audit" });
    expect(screen.getByRole("combobox", { name: "Inspected Organization" })).toBeVisible();
    expect(screen.getByRole("combobox", { name: "Inspection type" })).toBeVisible();
    expect(screen.getByRole("list", { name: "New Audit steps" })).toHaveTextContent("1Scope2Purpose3Schedule4Resources & budget5Review");
    expect(screen.getByRole("complementary", { name: "Audit plan summary" })).toHaveTextContent("Inspected Organization");
    expect(screen.queryByText("New Inspection")).toBeNull();
    expect(screen.queryByText(/Trigger type|Risk category|Inspection approach|Domain/)).toBeNull();
    expect(screen.queryByRole("button", { name: /Open audit setup|Save draft|Next/ })).toBeNull();
  });

  it("creates one proposal draft from the first valid Continue and renders auto-resolved scope facts", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    const create = vi.spyOn(runtime.backendForRole("manager").planningProposal!, "createDraft");
    renderWizardRoute("/department-manager/new-audit/step-1", runtime);
    const organization = await screen.findByRole("combobox", { name: "Inspected Organization" });
    await waitFor(() => expect(organization).toBeEnabled());
    await user.selectOptions(organization, "ORG-SKYCARGO");
    await waitFor(() => expect(screen.getAllByText("Automatically selected")[0]).toBeVisible());
    expect(screen.queryByRole("combobox", { name: "Provider scope" })).toBeNull();
    expect(screen.queryByRole("combobox", { name: "Regulated target" })).toBeNull();
    await user.click(screen.getByRole("button", { name: "Continue" }));
    await screen.findByRole("heading", { level: 2, name: "Purpose" });
    expect(create).toHaveBeenCalledTimes(1);
  });

  it.each([
    "/department-manager/new-audit/step-2",
    "/department-manager/new-audit/step-3",
    "/department-manager/new-audit/step-4",
    "/department-manager/new-audit/step-5",
  ])("keeps a draftless deep link safe at Scope: %s", async (path) => {
    renderWizardRoute(path);
    await screen.findByRole("heading", { level: 2, name: "Scope" });
    expect(screen.getByRole("heading", { level: 1, name: "New Audit" })).toBeVisible();
    expect(screen.queryByText(/Loading saved New Audit draft/)).toBeNull();
  });

  it("supports server-managed purpose presets without making the text read-only", async () => {
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-1");
    await createDraft(user);
    const purpose = screen.getByRole("textbox", { name: "Purpose" });
    await user.selectOptions(screen.getByRole("combobox", { name: "Purpose preset" }), "PURPOSE-ROUTINE-SURVEILLANCE");
    expect(purpose).toHaveValue("Complete a planned surveillance review of the approved operating context and confirm that required controls remain effective.");
    await user.clear(purpose);
    await user.type(purpose, "A manager-edited purpose.");
    expect(purpose).toHaveValue("A manager-edited purpose.");
    expect(screen.queryByText("Department Manager initiated")).toBeNull();
  });

  it("validates Schedule inline and switches between conditional On-site and Remote controls", async () => {
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-1");
    await createDraft(user);
    await user.type(screen.getByRole("textbox", { name: "Purpose" }), "Schedule validation");
    await user.click(screen.getByRole("button", { name: "Continue" }));
    await screen.findByRole("heading", { level: 2, name: "Schedule" });
    await user.click(screen.getByRole("button", { name: "Continue" }));
    const date = screen.getByLabelText("Planned date");
    expect(screen.getByText("Planned date is required")).toBeVisible();
    expect(document.activeElement).toBe(date);
    fireEvent.change(date, { target: { value: "2026-12-10" } });
    await user.click(screen.getByRole("radio", { name: "Remote" }));
    expect(screen.queryByText("Target-derived canonical location")).toBeNull();
    expect(screen.queryByLabelText("Enter another location")).toBeNull();
    await user.click(screen.getByRole("button", { name: "Add online meeting link" }));
    await user.type(screen.getByLabelText("Online meeting link"), "not-a-link");
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(screen.getByText("Use an HTTP(S) meeting link")).toBeVisible();
  });

  it("keeps Resources focused on capacity, workload estimate, and budget", async () => {
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-1");
    await completeThroughResources(user);
    expect(screen.getByRole("spinbutton", { name: "Required inspectors" })).toHaveValue(2);
    expect(screen.getByRole("spinbutton", { name: "Estimated checklist items" })).toBeVisible();
    expect(screen.getByRole("spinbutton", { name: "Requested budget" })).toBeVisible();
    expect(screen.getByText(/Suggested \d+; safe range/)).toBeVisible();
    expect(screen.queryByText(/Use suggested questions|Review selection|Question catalog/)).toBeNull();
    expect(screen.queryByRole("checkbox")).toBeNull();
  });

  it("opens a read-only checklist preview and never exposes selection controls", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    const listCatalog = vi.spyOn(runtime.backendForRole("manager").canonicalCatalog!, "listCatalog");
    renderWizardRoute("/department-manager/new-audit/step-1", runtime);
    await completeThroughResources(user);
    const trigger = screen.getByRole("button", { name: "Browse checklist items" });
    await user.click(trigger);
    const dialog = await screen.findByRole("dialog", { name: "Checklist item preview" });
    await waitFor(() => expect(listCatalog).toHaveBeenCalledWith(expect.objectContaining({ checklistFocus: ["ON_SITE_INSPECTION", "PERIODIC_SURVEILLANCE"] }), expect.anything()));
    const previewRequest = listCatalog.mock.calls.find(([input]) => input.projection === "full")?.[0];
    expect(previewRequest?.applicationType).toBeUndefined();
    expect(listCatalog).toHaveBeenCalledWith(expect.objectContaining({ projection: "full" }), expect.anything());
    expect(within(dialog).getByText(/candidate questions/)).not.toHaveTextContent("0 candidate questions");
    expect(dialog).toHaveTextContent("does not select or freeze the planned checklist");
    expect(within(dialog).queryByRole("checkbox")).toBeNull();
    expect(within(dialog).getByRole("button", { name: "Use visible count as estimate" })).toBeEnabled();
    await user.keyboard("{Escape}");
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Checklist item preview" })).toBeNull());
    expect(document.activeElement).toBe(trigger);
    await user.click(trigger);
    const reopenedDialog = await screen.findByRole("dialog", { name: "Checklist item preview" });
    await user.click(within(reopenedDialog).getByRole("button", { name: "Close" }));
    expect(screen.queryByRole("dialog", { name: "Checklist item preview" })).toBeNull();
  });

  it("shows non-blocking capacity/range warnings and accepts zero budget", async () => {
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-1");
    await completeThroughResources(user);
    await user.clear(screen.getByRole("spinbutton", { name: "Required inspectors" }));
    await user.type(screen.getByRole("spinbutton", { name: "Required inspectors" }), "8");
    await user.clear(screen.getByRole("spinbutton", { name: "Estimated checklist items" }));
    await user.type(screen.getByRole("spinbutton", { name: "Estimated checklist items" }), "1");
    expect(screen.getByText(/above the current eligible roster/)).toBeVisible();
    expect(screen.getByText(/outside the server-suggested safe range/)).toBeVisible();
    await user.type(screen.getByRole("spinbutton", { name: "Requested budget" }), "0");
    await user.click(screen.getByRole("button", { name: "Continue to review" }));
    await screen.findByRole("heading", { level: 2, name: "Review" });
    await screen.findByRole("heading", { level: 3, name: "Approval context" });
    expect(screen.getAllByText("0 USD")[0]).toBeVisible();
  });

  it("renders a decision-ready Review and submits a Planning item without creating an Audit", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    const manager = runtime.backendForRole("manager");
    const auditsBefore = (await manager.assignments.list({})).items.length;
    renderWizardRoute("/department-manager/new-audit/step-1", runtime);
    await completeThroughResources(user);
    await user.type(screen.getByRole("spinbutton", { name: "Requested budget" }), "0");
    await user.click(screen.getByRole("button", { name: "Continue to review" }));
    await screen.findByRole("heading", { level: 2, name: "Review" });
    await screen.findByRole("heading", { level: 3, name: "Approval context" });
    expect(screen.getByText(/does not create an executable Audit/)).toBeVisible();
    expect(screen.getByRole("button", { name: "Submit to Finance" })).toBeEnabled();
    await user.click(screen.getByRole("button", { name: "Submit to Finance" }));
    await screen.findByRole("heading", { name: "Department Planning" });
    expect((await manager.assignments.list({})).items.length).toBe(auditsBefore);
    expect(await screen.findByTestId("planning-selected-record")).toBeVisible();
  });

  it("serializes proposal autosave and exposes a retryable error", async () => {
    const runtime = createMockBackendRuntime();
    const proposal = runtime.backendForRole("manager").planningProposal!;
    const originalSave = proposal.saveDraft.bind(proposal);
    let attempts = 0;
    vi.spyOn(proposal, "saveDraft").mockImplementation(async (input) => { attempts += 1; if (attempts === 1) throw new Error("temporary save failure"); return originalSave(input); });
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-1", runtime);
    await createDraft(user);
    await user.type(screen.getByRole("textbox", { name: "Purpose" }), "Autosaved purpose");
    await waitFor(() => expect(screen.getAllByText("Couldn’t save")[0]).toBeVisible(), { timeout: 2500 });
    await user.click(screen.getAllByRole("button", { name: "Retry" })[0]);
    await waitFor(() => expect(screen.getAllByText("Saved")[0]).toBeVisible(), { timeout: 2500 });
    expect(attempts).toBeGreaterThanOrEqual(2);
  });

  it("persists the proposal across unmount and runtime restart", async () => {
    const user = userEvent.setup();
    const firstRuntime = createMockBackendPersistentRuntime(localStorage);
    renderWizardRoute("/department-manager/new-audit/step-1", firstRuntime);
    await createDraft(user);
    await user.type(screen.getByRole("textbox", { name: "Purpose" }), "Persisted proposal purpose");
    await waitFor(() => expect(screen.getAllByText("Saved")[0]).toBeVisible(), { timeout: 2500 });
    const draftId = (await screen.findByTestId("new-audit-wizard-page")).getAttribute("data-draft-id");
    cleanup();
    const restartedRuntime = createMockBackendPersistentRuntime(localStorage);
    renderWizardRoute(`/department-manager/new-audit/step-2?draftId=${encodeURIComponent(draftId ?? "")}`, restartedRuntime);
    expect(await screen.findByRole("textbox", { name: "Purpose" })).toHaveValue("Persisted proposal purpose");
    expect((await restartedRuntime.backendForRole("manager").planningProposal!.getDraft({ draftId: draftId ?? "" })).purpose).toBe("Persisted proposal purpose");
  });
});
