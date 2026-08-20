// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { AppRouter } from "../../app/router";
import { ScenarioProvider } from "../../app/scenario-context";
import type { DemoBackend } from "../../backend/backend";
import { createMockBackendPersistentRuntime, createMockBackendRuntime } from "../../mock/create-mock-backend";
import { catalogValueLabel } from "./planning-intake-formatters";

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

async function selectFirstAuthorizedScope(user: ReturnType<typeof userEvent.setup>) {
  const supplier = await screen.findByRole("combobox", { name: "Supplier / organization" });
  await waitFor(() => expect(supplier).toBeEnabled());
  const supplierValue = (supplier.querySelector("option") as HTMLOptionElement | null)?.value;
  if (!supplierValue) throw new Error("Expected an authorized supplier option");
  await user.selectOptions(supplier, supplierValue);
  await waitFor(() => expect(screen.getByRole("combobox", { name: "Provider scope" })).toBeEnabled());
  const provider = screen.getByRole("combobox", { name: "Provider scope" });
  await user.selectOptions(provider, (provider.querySelector("option") as HTMLOptionElement).value);
  const target = screen.getByRole("combobox", { name: "Regulated target" });
  await user.selectOptions(target, (target.querySelector("option") as HTMLOptionElement).value);
}

async function createDraft(user: ReturnType<typeof userEvent.setup>) {
  await selectFirstAuthorizedScope(user);
  expect(screen.queryByRole("button", { name: "Open audit setup for this supplier" })).toBeNull();
  expect(screen.queryByRole("button", { name: "Save draft" })).toBeNull();
  await user.click(screen.getByRole("button", { name: "Continue" }));
  await screen.findByRole("heading", { level: 2, name: "Purpose" });
  return screen.findByTestId("new-audit-wizard-page");
}

async function progressToChecklist(user: ReturnType<typeof userEvent.setup>, category = "Routine / Announced") {
  await createDraft(user);
  await user.selectOptions(await screen.findByRole("combobox", { name: /Inspection approach/ }), category);
  await user.type(await screen.findByLabelText("Purpose"), "Targeted apron safety verification");
  await user.click(screen.getByRole("button", { name: "Continue" }));
  await screen.findByRole("heading", { level: 2, name: "Schedule" });
  await user.type(await screen.findByLabelText("Planned date"), "2026-12-10");
  await user.type(await screen.findByLabelText("Location"), "Fly Namibia HQ");
  await user.click(screen.getByRole("button", { name: "Continue" }));
  await screen.findByRole("heading", { level: 2, name: "Checklist & budget" });
  return screen.findByTestId("new-audit-wizard-page");
}

async function confirmOneQuestion(user: ReturnType<typeof userEvent.setup>) {
  const [firstQuestion] = await screen.findAllByRole("checkbox", { name: /Select / });
  await user.click(firstQuestion);
  await user.click(screen.getByRole("button", { name: "Review selection" }));
  await screen.findByRole("dialog", { name: "Review selection" });
  await user.click(within(screen.getByRole("dialog", { name: "Review selection" })).getByRole("button", { name: "Confirm selection" }));
  await screen.findByText("Selection confirmed and saved to the server-owned scope.");
}

function selectedCountFrom(summary: HTMLElement): number {
  const text = within(summary).getByText(/[\d,]+ questions selected/).textContent ?? "0";
  return Number(text.match(/[\d,]+/)?.[0]?.replace(/,/g, "") ?? "0");
}

describe("New Inspection Planning intake", () => {
  it("keeps English protocol labels stable and makes the planned date keyboard-safe on mobile", async () => {
    expect(catalogValueLabel("INITIAL_CERTIFICATION")).toBe("Initial Certification");

    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-1");
    await createDraft(user);

    const riskCategory = screen.getByRole("combobox", { name: "Risk category" });
    expect(riskCategory).toHaveValue("Configured inspection risk");
    await user.selectOptions(riskCategory, "Safety-critical");
    expect(riskCategory).toHaveValue("Safety-critical");

    await user.type(await screen.findByLabelText("Purpose"), "Date control regression check");
    await user.click(screen.getByRole("button", { name: "Continue" }));
    const plannedDate = await screen.findByLabelText("Planned date");
    expect(plannedDate).toHaveAttribute("type", "text");
    expect(plannedDate).toHaveAttribute("inputmode", "text");
    expect(plannedDate).toHaveAttribute("enterkeyhint", "next");
    expect(plannedDate).toHaveAttribute("placeholder", "YYYY-MM-DD");
    const calendarPicker = screen.getByTestId("planning-intake-date-picker");
    expect(calendarPicker).toHaveAttribute("type", "date");
    await user.type(plannedDate, "20261210");
    expect(plannedDate).toHaveValue("2026-12-10");
    await user.type(await screen.findByLabelText("Location"), "Fly Namibia HQ");
    await user.click(plannedDate);
    await user.keyboard("{Enter}");
    await screen.findByRole("heading", { level: 2, name: "Checklist & budget" });
  });

  it("loads every authorized scope page before building the cascade", async () => {
    const runtime = createMockBackendRuntime();
    const catalog = runtime.backendForRole("manager").canonicalCatalog!;
    const original = catalog.listScopeOptions.bind(catalog);
    const allOptions = (await original({ limit: 25 })).items;
    const listScopeOptions = vi.spyOn(catalog, "listScopeOptions").mockImplementation(async (input) => input?.cursor
      ? { items: allOptions.slice(2), nextCursor: null }
      : { items: allOptions.slice(0, 2), nextCursor: "2" });
    renderWizardRoute("/department-manager/new-audit/step-1", runtime);

    const supplier = await screen.findByRole("combobox", { name: "Supplier / organization" });
    await waitFor(() => expect(supplier).toBeEnabled());
    expect(supplier.querySelectorAll("option")).toHaveLength(3);
    expect(listScopeOptions).toHaveBeenCalledTimes(2);
  });

  it("exposes the full authorized supplier cascade instead of a single fixture option", async () => {
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-1");

    const supplier = await screen.findByRole("combobox", { name: "Supplier / organization" });
    await waitFor(() => expect(supplier).toBeEnabled());
    expect(supplier.querySelectorAll("option")).toHaveLength(3);

    const provider = screen.getByRole("combobox", { name: "Provider scope" });
    expect(provider.querySelectorAll("option")).toHaveLength(3);
    await user.selectOptions(provider, "SCOPE-FLY-NAMIBIA-AERODROME");

    const target = screen.getByRole("combobox", { name: "Regulated target" });
    expect(target.querySelectorAll("option")).toHaveLength(3);
    await user.selectOptions(provider, "SCOPE-OPS-AOC-SOURCE-BOUND");
    expect(screen.getByRole("combobox", { name: "Inspection type" }).querySelectorAll("option")).toHaveLength(10);
  });

  it("starts at Basics without a draft and creates the server draft on the first valid Continue", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    const create = vi.spyOn(runtime.backendForRole("manager").planningIntake, "createDraft");
    renderWizardRoute("/department-manager/new-audit/step-1", runtime);
    await selectFirstAuthorizedScope(user);
    expect(screen.queryByTestId("new-audit-wizard-page")).toBeNull();
    expect(screen.getByRole("list", { name: "Planning intake steps" })).toHaveTextContent("1Basics2Purpose3Schedule4Checklist & budget5Review");
    await user.click(screen.getByRole("button", { name: "Continue" }));
    await screen.findByRole("heading", { level: 2, name: "Purpose" });
    expect(create).toHaveBeenCalledTimes(1);
    expect(await screen.findByTestId("new-audit-wizard-page")).toHaveAttribute("data-draft-id", expect.stringMatching(/^PLAN-DRAFT-/));
    expect(screen.getAllByText("Saved")[0]).toBeVisible();
  });

  it.each([
    "/department-manager/new-audit/step-2",
    "/department-manager/new-audit/step-3",
    "/department-manager/new-audit/step-4",
    "/department-manager/new-audit/step-5",
  ])("canonicalizes draftless later route %s to Basics", async (path) => {
    renderWizardRoute(path);
    await screen.findByRole("heading", { level: 2, name: "Basics" });
    expect(screen.queryByText(/Stage \d of 3/)).toBeNull();
    expect(screen.queryByRole("heading", { name: /Step [2345] of 5/ })).toBeNull();
  });

  it("keeps human scope labels and defaults the catalog recommendation after draft creation", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    const manager = runtime.backendForRole("manager");
    const listCatalog = vi.spyOn(manager.canonicalCatalog!, "listCatalog");
    renderWizardRoute("/department-manager/new-audit/step-1", runtime);
    await createDraft(user);
    expect(screen.getByRole("heading", { level: 2, name: "Purpose" })).toBeVisible();
    await user.type(await screen.findByLabelText("Purpose"), "Scope presentation contract");
    await user.click(screen.getByRole("button", { name: "Continue" }));
    await screen.findByRole("heading", { level: 2, name: "Schedule" });
    await user.type(await screen.findByLabelText("Planned date"), "2026-12-10");
    await user.type(await screen.findByLabelText("Location"), "Fly Namibia HQ");
    await user.click(screen.getByRole("button", { name: "Continue" }));
    await screen.findByRole("heading", { level: 2, name: "Checklist & budget" });
    await screen.findByText("Suggested questions");
    expect(screen.getByText(/matching questions · page 1/)).toBeVisible();
    expect(listCatalog.mock.calls.some(([input]) => input.includedByDefault === true && input.recommendationState === undefined)).toBe(true);
    expect(screen.queryByText(/qv:synthetic/)).toBeNull();
    expect(screen.getByRole("complementary", { name: "Inspection brief" })).toHaveTextContent("Fly Namibia");
  });

  it("returns Clear filters to the server suggestion and labels the full catalog explicitly", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    const catalog = runtime.backendForRole("manager").canonicalCatalog!;
    const originalListCatalog = catalog.listCatalog.bind(catalog);
    vi.spyOn(catalog, "listCatalog").mockImplementation(async (input, options) => {
      const page = await originalListCatalog(input, options);
      if (input.includedByDefault === true) return { ...page, totalCount: 1002 };
      if (input.includedByDefault === undefined && input.recommendationState === undefined) return { ...page, totalCount: 1310 };
      return page;
    });
    renderWizardRoute("/department-manager/new-audit/step-1", runtime);
    await progressToChecklist(user);

    expect(screen.getByRole("heading", { level: 3, name: "Suggested questions" })).toBeVisible();
    await waitFor(() => expect(screen.getByText("1,002 matching questions · page 1")).toBeVisible());
    const filters = screen.getByText("Advanced filters").closest("details");
    if (!filters) throw new Error("Advanced filters container is missing");
    await user.click(screen.getByText("Advanced filters"));
    const recommendation = screen.getByLabelText("Recommendation filter");
    await user.selectOptions(recommendation, "");
    await waitFor(() => expect(screen.getByRole("heading", { level: 3, name: "Full approved catalog" })).toBeVisible());
    await waitFor(() => expect(screen.getByText("1,310 matching questions · page 1")).toBeVisible());
    expect(filters).toHaveTextContent("1 active");
    expect(screen.getByText(/Showing the complete approved catalog\./)).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    await waitFor(() => expect(recommendation).toHaveValue("SUGGESTED_NOW"));
    expect(screen.getByRole("heading", { level: 3, name: "Suggested questions" })).toBeVisible();
    await waitFor(() => expect(screen.getByText("1,002 matching questions · page 1")).toBeVisible());
    expect(screen.queryByRole("heading", { level: 3, name: "Full approved catalog" })).toBeNull();
  });

  it("renders the exact no-history scope-filter golden IDs in suggested and full UI states", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime(() => "2026-08-18T12:00:00.000Z", "prior-audit-no-history-scope-filter");
    renderWizardRoute("/department-manager/new-audit/step-1", runtime);
    await progressToChecklist(user);
    const visibleIDs = () => [...document.querySelectorAll<HTMLElement>(".planning-intake-catalog-list > li")].map((row) => row.dataset.questionVersionId).sort();
    await waitFor(() => expect(visibleIDs()).toEqual([
      "Q-NO-HISTORY-IN-FOCUS-OPTIONAL",
      "Q-NO-HISTORY-OUTSIDE-FOCUS-MANDATORY",
    ]));
    await user.click(screen.getByText("Advanced filters"));
    await user.selectOptions(screen.getByLabelText("Recommendation filter"), "");
    await waitFor(() => expect(visibleIDs()).toEqual([
      "Q-NO-HISTORY-IN-FOCUS-OPTIONAL",
      "Q-NO-HISTORY-OUTSIDE-FOCUS-MANDATORY",
      "Q-NO-HISTORY-OUTSIDE-FOCUS-OPTIONAL",
    ]));
    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    await waitFor(() => expect(visibleIDs()).toEqual([
      "Q-NO-HISTORY-IN-FOCUS-OPTIONAL",
      "Q-NO-HISTORY-OUTSIDE-FOCUS-MANDATORY",
    ]));
  });

  it("adds suggested questions into the selected-tray and allows removing one entry", async () => {
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-4");
    const page = await progressToChecklist(user);
    const summary = within(page).getByRole("region", { name: "Selection summary" });
    expect(selectedCountFrom(summary)).toBe(0);
    await user.click(screen.getByRole("button", { name: "Use suggested questions" }));
    await waitFor(() => expect(selectedCountFrom(summary)).toBeGreaterThan(0));
    const afterAdd = selectedCountFrom(summary);
    expect(afterAdd).toBeGreaterThan(0);
    const selectedTray = within(summary).getByRole("region", { name: "Selected questions" });
    expect(selectedTray).toHaveTextContent("selected questions");
    await user.click(within(selectedTray).getAllByRole("button", { name: "View details" })[0]);
    const selectedQuestionDialog = await screen.findByRole("dialog", { name: "Question dossier" });
    expect(selectedQuestionDialog).toHaveTextContent("Question details");
    await user.click(within(selectedQuestionDialog).getByRole("button", { name: "Close" }));
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Question dossier" })).toBeNull());
    await user.click(within(selectedTray).getAllByRole("button", { name: "Remove" })[0]);
    await waitFor(() => expect(selectedCountFrom(summary)).toBe(afterAdd - 1));
    const uncheckedQuestion = screen.getAllByRole("checkbox", { name: /Select / }).find((checkbox) => !(checkbox as HTMLInputElement).checked);
    expect(uncheckedQuestion).toBeDefined();
    await user.click(uncheckedQuestion!);
    await waitFor(() => expect(selectedCountFrom(summary)).toBe(afterAdd));
    await user.click(screen.getByRole("button", { name: "Use suggested questions" }));
    await waitFor(() => expect(screen.getByText(/matched questions are already selected/)).toBeVisible());
    expect(screen.getByRole("button", { name: "Use suggested questions" })).toBeEnabled();
  });

  it("shows concrete recommendation reasons instead of generic omission rationale text", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime(() => "2026-08-18T12:00:00.000Z", "prior-audit-single-history");
    renderWizardRoute("/department-manager/new-audit/step-4", runtime);
    await progressToChecklist(user);
    expect(screen.queryByText("One clean Audit is not sufficient longitudinal evidence for omission.")).toBeNull();
    expect(screen.getAllByText(/Insufficient longitudinal history/).length).toBeGreaterThan(0);
    const summary = screen.getByRole("status", { name: "Prior-Audit recommendation summary" });
    expect(summary).toBeInTheDocument();
  });

  it("labels the historical suggestion default location and demo override guidance", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime(() => "2026-08-18T12:00:00.000Z", "prior-audit-multi-history");
    renderWizardRoute("/department-manager/new-audit/step-4", runtime);
    await progressToChecklist(user);
    const summary = await screen.findByRole("status", { name: "Prior-Audit recommendation summary" });
    expect(summary).toHaveTextContent("Default historical suggestion location: Windhoek International Airport (historical location snapshot).");
    expect(summary).toHaveTextContent("Demo note: Historical suggestion matching is pre-configured and ready with this default location.");
    expect(summary).toHaveTextContent("choose a different schedule location before running suggestion actions again");
  });

  it("shows the prior-Audit scenario, every withheld reason, and restores the exact history-deferred set", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime(() => "2026-08-18T12:00:00.000Z", "prior-audit-multi-history");
    renderWizardRoute("/department-manager/new-audit/step-4", runtime);
    await progressToChecklist(user);
    const summary = await screen.findByRole("status", { name: "Prior-Audit recommendation summary" });
    expect(summary).toHaveTextContent("3 comparable prior Audits");
    expect(summary).toHaveTextContent("1 withheld by history");
    expect(screen.getByRole("list", { name: "History-deferred questions" })).toHaveTextContent("validated-clean of 3 comparable Audits");
    const restore = screen.getByRole("button", { name: "Include all history-deferred questions" });
    await waitFor(() => expect(restore).toBeEnabled());
    await user.click(restore);
    expect(screen.getByText("1 history-deferred questions are included and ready for selection review.")).toBeVisible();
    expect(screen.getByRole("region", { name: "Selection summary" })).toHaveTextContent("1 questions selected");
  });

  it("exposes all advanced filters while keeping them collapsed initially", async () => {
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-4");
    await progressToChecklist(user);
    const filters = screen.getByText("Advanced filters").closest("details");
    expect(filters).not.toHaveAttribute("open");
    await user.click(screen.getByText("Advanced filters"));
    expect(screen.getByText("Form · Any")).toBeVisible();
    expect(screen.getByText("Domain · Any")).toBeVisible();
    expect(screen.getByText("Topic · Any")).toBeVisible();
    expect(screen.getByText("Risk · Any")).toBeVisible();
    expect(screen.getByText("Checklist focus · Any")).toBeVisible();
    expect(screen.getByLabelText("Source gap filter")).toBeVisible();
    expect(screen.getByLabelText("Recommendation filter")).toBeVisible();
    expect(screen.getByLabelText("Selected state filter")).toBeVisible();
    expect(screen.getByRole("button", { name: "Clear filters" })).toBeVisible();
    expect(screen.queryByRole("button", { name: "Preview next exact batch" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Confirm selection" })).toBeNull();
  });

  it("opens and closes the question dossier with Escape and returns focus", async () => {
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-4");
    await progressToChecklist(user);
    const trigger = (await screen.findAllByRole("button", { name: "View details" }))[0];
    await user.click(trigger);
    const dialog = await screen.findByRole("dialog", { name: "Question dossier" });
    expect(dialog).toHaveTextContent("Question details");
    await user.keyboard("{Escape}");
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Question dossier" })).toBeNull());
    expect(document.activeElement).toBe(trigger);
  });

  it("runs the bounded selection chain behind one review confirmation", async () => {
    const runtime = createMockBackendRuntime();
    const review = runtime.backendForRole("manager").canonicalCatalog!;
    const previewBatches: Array<{ count: number; expectedDigest: string }> = [];
    const commitBatches: Array<{ count: number; expectedDigest: string }> = [];
    const committedDigests: string[] = [];
    const originalPreview = review.previewSelection.bind(review);
    const originalCommit = review.commitSelection.bind(review);
    vi.spyOn(review, "previewSelection").mockImplementation(async (input, options) => { previewBatches.push({ count: input.questionVersionIds.length, expectedDigest: input.expectedSelectionDigest }); return originalPreview(input, options); });
    vi.spyOn(review, "commitSelection").mockImplementation(async (input, options) => { commitBatches.push({ count: input.questionVersionIds.length, expectedDigest: input.expectedSelectionDigest }); const receipt = await originalCommit(input, options); committedDigests.push(receipt.selection.selectionDigest); return receipt; });
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-4", runtime);
    await progressToChecklist(user);
    await user.click(screen.getByText("More selection actions"));
    await user.click(screen.getByRole("button", { name: "Add all matching eligible questions" }));
    await waitFor(() => expect(screen.getByText(/1,310 eligible questions are ready/)).toBeVisible());
    await user.click(screen.getByRole("button", { name: "Review selection" }));
    const dialog = screen.getByRole("dialog", { name: "Review selection" });
    await user.click(within(dialog).getByRole("button", { name: "Confirm selection" }));
    await screen.findByText("Selection confirmed and saved to the server-owned scope.");
    expect(previewBatches.map(({ count }) => count)).toEqual([500, 500, 310]);
    expect(commitBatches.map(({ count }) => count)).toEqual([500, 500, 310]);
    expect(commitBatches[1]?.expectedDigest).toBe(committedDigests[0]);
    expect(commitBatches[2]?.expectedDigest).toBe(committedDigests[1]);
  });

  it("shows inline budget validation, accepts zero, and removes Preview from review", async () => {
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-4");
    await progressToChecklist(user);
    await confirmOneQuestion(user);
    const budget = await screen.findByLabelText("Requested Budget");
    await user.clear(budget);
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByText("Requested budget is required")).toBeVisible();
    expect(budget).toHaveAttribute("aria-invalid", "true");
    expect(document.activeElement).toBe(budget);
    await user.type(budget, "0");
    await user.click(screen.getByRole("button", { name: "Continue" }));
    await screen.findByRole("heading", { level: 2, name: "Review" });
    expect(screen.queryByRole("button", { name: "Preview" })).toBeNull();
    await screen.findByRole("heading", { level: 3, name: "Notice & governance" });
    expect(document.body).toHaveTextContent("Requested budget");
  });

  it("serializes autosave and exposes a retryable save failure", async () => {
    const runtime = createMockBackendRuntime();
    const manager = runtime.backendForRole("manager");
    const originalSave = manager.planningIntake.saveDraft.bind(manager.planningIntake);
    let attempts = 0;
    vi.spyOn(manager.planningIntake, "saveDraft").mockImplementation(async (input) => {
      attempts += 1;
      if (attempts === 1) throw new Error("temporary save failure");
      return originalSave(input);
    });
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-1", runtime);
    await createDraft(user);
    await user.type(await screen.findByLabelText("Purpose"), "Autosaved purpose");
    await waitFor(() => expect(screen.getAllByText("Couldn't save")[0]).toBeVisible(), { timeout: 2000 });
    expect(screen.getAllByRole("button", { name: "Retry" })[0]).toBeVisible();
    await user.click(screen.getAllByRole("button", { name: "Retry" })[0]);
    await waitFor(() => expect(screen.getAllByText("Saved")[0]).toBeVisible(), { timeout: 2000 });
    expect(attempts).toBeGreaterThanOrEqual(2);
  });

  it("persists one exact draft across Back, autosave, unmount, and runtime restart", async () => {
    const user = userEvent.setup();
    const firstRuntime = createMockBackendPersistentRuntime(localStorage);
    renderWizardRoute("/department-manager/new-audit/step-1", firstRuntime);
    await createDraft(user);
    await user.type(await screen.findByLabelText("Purpose"), "Persisted targeted inspection purpose");
    await waitFor(() => expect(screen.getAllByText("Saved")[0]).toBeVisible(), { timeout: 2000 });
    const draftId = (await screen.findByTestId("new-audit-wizard-page")).getAttribute("data-draft-id");
    expect(draftId).toMatch(/^PLAN-DRAFT-/);
    cleanup();
    const restartedRuntime = createMockBackendPersistentRuntime(localStorage);
    renderWizardRoute(`/department-manager/new-audit/step-2?draftId=${encodeURIComponent(draftId ?? "")}`, restartedRuntime);
    expect(await screen.findByLabelText("Purpose")).toHaveValue("Persisted targeted inspection purpose");
    const draft = await restartedRuntime.backendForRole("manager").planningIntake.getDraft({ draftId: draftId ?? "" });
    expect(draft).toMatchObject({ id: draftId, purpose: "Persisted targeted inspection purpose" });
  });

  it("submits a zero-budget unannounced Planning item to Finance without creating an Audit or exposing notice", async () => {
    const runtime = createMockBackendRuntime();
    const manager = runtime.backendForRole("manager");
    const auditsBefore = (await manager.assignments.list({})).items.map((item) => item.auditId);
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-1", runtime);
    await progressToChecklist(user, "Ad Hoc / Unannounced");
    await confirmOneQuestion(user);
    await user.clear(await screen.findByLabelText("Requested Budget"));
    await user.type(await screen.findByLabelText("Requested Budget"), "0");
    await user.click(screen.getByRole("button", { name: "Continue" }));
    await screen.findByRole("heading", { level: 2, name: "Review" });
    await screen.findByRole("heading", { level: 3, name: "Notice & governance" });
    expect(screen.getAllByText(/Notice withheld/)[0]).toBeVisible();
    expect(screen.getByText(/does not create an Audit or start an Inspector assignment/)).toBeVisible();
    await user.click(screen.getByRole("button", { name: "Submit to Finance" }));
    const selectedRecord = await screen.findByTestId("planning-selected-record");
    const planningItemId = (await manager.planning.list({ limit: 20 })).items.find((item) => selectedRecord.textContent?.includes(item.title))?.id;
    expect(planningItemId).toBeTruthy();
    const submitted = (await manager.planning.list({ limit: 20 })).items.find((item) => item.id === planningItemId);
    expect(submitted).toMatchObject({ id: planningItemId, organizationId: "ORG-FLY-NAMIBIA", estimatedBudget: 0, status: "FINANCE_REVIEW", currentOwnerRole: "finance", nextAction: "Finance to review budget and resources", revision: 1 });
    expect((await manager.assignments.list({})).items.map((item) => item.auditId)).toEqual(auditsBefore);
    const auditee = runtime.backendForRole("auditee");
    await expect(auditee.planning.list({})).rejects.toThrow(/CAA planning access is required/i);
    expect(JSON.stringify(await auditee.calendar.list({}))).not.toContain(planningItemId);
    expect(JSON.stringify(await auditee.calendar.list({}))).not.toContain("Targeted apron safety verification");
  });

  it("keeps the withheld intake absent from Auditee DOM and projections after persistent runtime restart", async () => {
    const firstRuntime = createMockBackendPersistentRuntime(localStorage);
    const manager = firstRuntime.backendForRole("manager");
    const draft = await manager.planningIntake.getDraft({ draftId: "PLAN-DRAFT-2026-001" });
    const saved = await manager.planningIntake.saveDraft({ draftId: draft.id, expectedRevision: draft.revision, idempotencyKey: "PRIVACY-SAVE-PLAN-DRAFT-2026-001-R1", values: { ...draft, inspectionCategory: "Ad Hoc / Unannounced", noticePolicy: "WITHHELD", purpose: "WITHHELD-TARGETED-APRON-PURPOSE", riskCategory: "WITHHELD-INTERNAL-RISK-CATEGORY", location: "Fly Namibia HQ", requestedBudget: 0 } });
    await manager.planningIntake.submit({ draftId: saved.id, expectedRevision: saved.revision, idempotencyKey: "PRIVACY-SUBMIT-PLAN-DRAFT-2026-001-R2", planningItemId: "PLAN-2026-WITHHELD-001" });
    const restartedRuntime = createMockBackendPersistentRuntime(localStorage);
    renderWizardRoute("/auditee/service-provider-cap", restartedRuntime);
    await screen.findByTestId("auditee-scope");
    const forbidden = ["PLAN-2026-WITHHELD-001", "Ad Hoc / Unannounced — Fly Namibia", "WITHHELD-TARGETED-APRON-PURPOSE", "WITHHELD-INTERNAL-RISK-CATEGORY", "Department Manager initiated"];
    for (const value of forbidden) expect(document.body).not.toHaveTextContent(value);
    const auditee = restartedRuntime.backendForRole("auditee");
    const projection = JSON.stringify({ calendar: await auditee.calendar.list({}), communications: await auditee.communications.list({}), documents: await auditee.documents.list({}), notifications: await auditee.notifications.list({}) });
    for (const value of forbidden) expect(projection).not.toContain(value);
  });

  it("keeps legacy AGA recommendation controls absent from the canonical flow", async () => {
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-5");
    await createDraft(user);
    expect(screen.queryByRole("region", { name: "AGA recommendation" })).toBeNull();
    expect(screen.queryByRole("button", { name: /Create AGA recommendation/i })).toBeNull();
  });

  it.each([1440, 1024, 390])("keeps progress, form, brief, and actions ordered at %ipx", async (width) => {
    Object.defineProperty(window, "innerWidth", { configurable: true, value: width });
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-4");
    const page = await progressToChecklist(user);
    const rail = within(page).getByRole("list", { name: "Planning intake steps" });
    const form = within(page).getByRole("region", { name: "Planning intake form" });
    const brief = within(page).getByRole("complementary", { name: "Inspection brief" });
    const actions = within(page).getByRole("region", { name: "Planning intake actions" });
    expect(rail.compareDocumentPosition(form) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(form.compareDocumentPosition(brief) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(brief.compareDocumentPosition(actions) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(await screen.findByLabelText("Requested Budget")).toBeEnabled();
  });

  it("exposes planning intake commands to managers and denies Auditee access", async () => {
    const runtime = createMockBackendRuntime();
    const manager = runtime.backendForRole("manager") as DemoBackend;
    expect(manager.planningIntake).toBeDefined();
    expect(manager.planningIntake).toHaveProperty("getDraft");
    expect(manager.planningIntake).toHaveProperty("saveDraft");
    expect(manager.planningIntake).toHaveProperty("submit");
    await expect(runtime.backendForRole("auditee").planningIntake.getDraft({ draftId: "PLAN-DRAFT-2026-001" })).rejects.toThrow(/unavailable to this role|Department Manager/i);
  });
});
