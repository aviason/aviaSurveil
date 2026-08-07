// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { AppRouter } from "../../app/router";
import { ScenarioProvider } from "../../app/scenario-context";
import type { DemoBackend } from "../../backend/backend";
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

async function selectFirstAuthorizedScope(user: ReturnType<typeof userEvent.setup>) {
  const select = await screen.findByRole("combobox", { name: "Organization, provider scope, and regulated target" });
  await waitFor(() => expect(select).toBeEnabled());
  await waitFor(() => expect(select.querySelectorAll("option").length).toBeGreaterThan(1));
  const option = select.querySelectorAll("option")[1] as HTMLOptionElement | undefined;
  if (!option?.value) throw new Error("Expected an authorized scope option");
  await user.selectOptions(select, option.value);
  return screen.findByTestId("new-audit-wizard-page");
}

async function progressToStepFive(user: ReturnType<typeof userEvent.setup>) {
  await selectFirstAuthorizedScope(user);
  await screen.findByRole("heading", { level: 2, name: /Step 1 of 5/ });
  await user.click(screen.getByRole("button", { name: "Next" }));
  await screen.findByRole("heading", { level: 2, name: /Step 2 of 5/ });
  await screen.findByTestId("new-audit-wizard-page");
  await user.selectOptions(screen.getByLabelText("Inspection Category"), "Ad Hoc / Unannounced");
  await user.type(screen.getByLabelText("Purpose"), "Targeted apron safety verification");
  await user.click(screen.getByRole("button", { name: "Next" }));
  await screen.findByRole("heading", { level: 2, name: /Step 3 of 5/ });
  await screen.findByTestId("new-audit-wizard-page");
  await user.type(screen.getByLabelText("Planned Date"), "2026-12-10");
  await user.type(screen.getByLabelText("Location"), "Fly Namibia HQ");
  await user.click(screen.getByRole("button", { name: "Next" }));
  await screen.findByRole("heading", { level: 2, name: /Step 4 of 5/ });
  await screen.findByTestId("new-audit-wizard-page");
  const [firstQuestion] = await screen.findAllByRole("checkbox", { name: /Select / });
  if (!firstQuestion) throw new Error("Expected at least one selectable question");
  await user.click(firstQuestion);
  await screen.findByText(/Exact question selection committed/);
  await user.clear(screen.getByLabelText("Requested Budget"));
  await user.type(screen.getByLabelText("Requested Budget"), "0");
  await user.click(screen.getByRole("button", { name: "Next" }));
  await screen.findByRole("heading", { level: 2, name: /Step 5 of 5/ });
}

describe("New Inspection Planning intake", () => {
  it.each([
    ["/department-manager/new-audit/step-1", "Inspection basics"],
    ["/department-manager/new-audit/step-2", "Category and purpose"],
    ["/department-manager/new-audit/step-3", "When and where"],
    ["/department-manager/new-audit/step-4", "Choose questions and budget"],
    ["/department-manager/new-audit/step-5", "Review and submit"],
  ])("direct-loads %s as one explicit persisted draft step", async (path, marker) => {
    const user = userEvent.setup();
    renderWizardRoute(path);
    const page = await selectFirstAuthorizedScope(user);
    expect(within(page).getByRole("heading", { level: 1, name: "New Inspection" })).toBeVisible();
    expect(page).toHaveTextContent(marker);
    expect(page.getAttribute("data-draft-id")).toMatch(/^PLAN-DRAFT-/);
    expect(screen.queryByTestId("route-pending-implementation")).toBeNull();
  });

  it("exposes explicit demo-only planning intake commands and denies Auditee access", async () => {
    const runtime = createMockBackendRuntime();
    const manager = runtime.backendForRole("manager") as DemoBackend;
    expect(manager.planningIntake).toBeDefined();
    expect(manager.planningIntake).toHaveProperty("getDraft");
    expect(manager.planningIntake).toHaveProperty("saveDraft");
    expect(manager.planningIntake).toHaveProperty("submit");
    await expect(runtime.backendForRole("auditee").planningIntake.getDraft({ draftId: "PLAN-DRAFT-2026-001" })).rejects.toThrow(
      /unavailable to this role|Department Manager/i,
    );
  });

  it("validates required prior-step data without losing the draft", async () => {
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-2");
    await selectFirstAuthorizedScope(user);
    await screen.findByRole("heading", { level: 2, name: /Step 2 of 5/ });
    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Purpose is required");
    expect(screen.getByRole("heading", { level: 2, name: /Step 2 of 5/ })).toBeVisible();
    expect(screen.getByLabelText("Purpose")).toHaveValue("");
  });

  it("keeps the requested budget as raw input so blank is invalid while literal zero remains valid", async () => {
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-4");
    await selectFirstAuthorizedScope(user);
    await screen.findByRole("heading", { level: 2, name: /Step 4 of 5/ });
    const budget = screen.getByLabelText("Requested Budget");
    await user.click((await screen.findAllByRole("checkbox", { name: /Select / }))[0]);
    await screen.findByText(/Exact question selection committed/);

    await user.clear(budget);
    expect(budget).toHaveValue(null);
    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Requested budget is required");
    expect(screen.getByRole("heading", { level: 2, name: /Step 4 of 5/ })).toBeVisible();

    await user.type(budget, "0");
    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(await screen.findByRole("heading", { level: 2, name: /Step 5 of 5/ })).toBeVisible();
    expect(await screen.findByTestId("new-audit-wizard-page")).toHaveTextContent("0 USD");
  });

  it("persists one exact draft across Back/Next, unmount, and runtime restart", async () => {
    const user = userEvent.setup();
    const firstRuntime = createMockBackendPersistentRuntime(localStorage);
    renderWizardRoute("/department-manager/new-audit/step-1", firstRuntime);
    await selectFirstAuthorizedScope(user);
    await screen.findByRole("heading", { level: 2, name: /Step 1 of 5/ });
    await user.click(screen.getByRole("button", { name: "Next" }));
    await screen.findByRole("heading", { level: 2, name: /Step 2 of 5/ });
    const draftId = (await screen.findByTestId("new-audit-wizard-page")).getAttribute("data-draft-id");
    expect(draftId).toMatch(/^PLAN-DRAFT-/);
    await user.type(screen.getByLabelText("Purpose"), "Persisted targeted inspection purpose");
    await user.click(screen.getByRole("button", { name: "Next" }));
    await screen.findByRole("heading", { level: 2, name: /Step 3 of 5/ });
    await user.click(screen.getByRole("button", { name: "Back" }));
    expect(await screen.findByLabelText("Purpose")).toHaveValue("Persisted targeted inspection purpose");

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
    await progressToStepFive(user);

    const page = await screen.findByTestId("new-audit-wizard-page");
    expect(page).toHaveTextContent("No Advance Notice (withheld)");
    expect(page).toHaveTextContent("Department Manager → Finance Review → General Manager → Executive Director → General Manager Release");
    expect(page).toHaveTextContent("No executable Audit is created at this step");
    await user.click(within(page).getByRole("button", { name: "Submit for Finance Review" }));
    const selectedRecord = await screen.findByTestId("planning-selected-record");
    const planningItemId = selectedRecord.textContent?.match(/PLAN-[A-Z0-9-]+/)?.[0];
    expect(planningItemId).toBeTruthy();

    const submitted = (await manager.planning.list({ limit: 20 })).items.find((item) => item.id === planningItemId);
    expect(submitted).toMatchObject({
      id: planningItemId,
      organizationId: "ORG-FLY-NAMIBIA",
      estimatedBudget: 0,
      status: "FINANCE_REVIEW",
      currentOwnerRole: "finance",
      nextAction: "Finance to review budget and resources",
      revision: 1,
    });
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
    const saved = await manager.planningIntake.saveDraft({
      draftId: draft.id,
      expectedRevision: draft.revision,
      idempotencyKey: "PRIVACY-SAVE-PLAN-DRAFT-2026-001-R1",
      values: {
        ...draft,
        inspectionCategory: "Ad Hoc / Unannounced",
        noticePolicy: "WITHHELD",
        purpose: "WITHHELD-TARGETED-APRON-PURPOSE",
        riskCategory: "WITHHELD-INTERNAL-RISK-CATEGORY",
        location: "Fly Namibia HQ",
        requestedBudget: 0,
      },
    });
    await manager.planningIntake.submit({
      draftId: saved.id,
      expectedRevision: saved.revision,
      idempotencyKey: "PRIVACY-SUBMIT-PLAN-DRAFT-2026-001-R2",
      planningItemId: "PLAN-2026-WITHHELD-001",
    });

    const restartedRuntime = createMockBackendPersistentRuntime(localStorage);
    renderWizardRoute("/auditee/service-provider-cap", restartedRuntime);
    await screen.findByTestId("auditee-scope");
    const forbidden = [
      "PLAN-2026-WITHHELD-001",
      "Ad Hoc / Unannounced — Fly Namibia",
      "WITHHELD-TARGETED-APRON-PURPOSE",
      "WITHHELD-INTERNAL-RISK-CATEGORY",
      "Department Manager initiated",
    ];
    for (const value of forbidden) expect(document.body).not.toHaveTextContent(value);

    const auditee = restartedRuntime.backendForRole("auditee");
    const auditeeProjection = JSON.stringify({
      calendar: await auditee.calendar.list({}),
      communications: await auditee.communications.list({}),
      documents: await auditee.documents.list({}),
      notifications: await auditee.notifications.list({}),
    });
    for (const value of forbidden) expect(auditeeProjection).not.toContain(value);
  });

  it("keeps legacy AGA recommendation controls absent from the canonical New Audit flow", async () => {
    const runtime = createMockBackendRuntime();
    renderWizardRoute("/department-manager/new-audit/step-5", runtime);
    await selectFirstAuthorizedScope(userEvent.setup());
    await screen.findByRole("heading", { level: 2, name: /Step 5 of 5/ });
    expect(screen.queryByRole("region", { name: "AGA recommendation" })).toBeNull();
    expect(screen.queryByRole("button", { name: /Create AGA recommendation/i })).toBeNull();
  });

  it.each([1440, 1024, 390])("keeps step rail, form, and actions ordered at %ipx", async (width) => {
    Object.defineProperty(window, "innerWidth", { configurable: true, value: width });
    const user = userEvent.setup();
    renderWizardRoute("/department-manager/new-audit/step-4");
    const page = await selectFirstAuthorizedScope(user);
    const rail = within(page).getByRole("list", { name: "Planning intake steps" });
    const form = within(page).getByRole("region", { name: "Planning intake form" });
    const actions = within(page).getByRole("region", { name: "Planning intake actions" });
    expect(rail.compareDocumentPosition(form) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(form.compareDocumentPosition(actions) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    await waitFor(() => expect(within(page).getByLabelText("Requested Budget")).toBeEnabled());
  });
});
