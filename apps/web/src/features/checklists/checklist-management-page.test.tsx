// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import {
  EXACT_BLOCKED_REAL_OPS_AOC_REQUEST,
  SYNTHETIC_GOVERNED_BUNDLE,
} from "../../backend/governed-synthetic-profile";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { ChecklistManagementPage } from "./checklist-management-page";

afterEach(cleanup);

describe("ChecklistManagementPage governed manager review", () => {
  it("shows zero review actions for the exact blocked real controlled-procedure OPS/AOC profile", async () => {
    const runtime = createMockBackendRuntime();
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
          <MemoryRouter initialEntries={["/department-manager/checklist-management"]}>
            <ChecklistManagementPage />
          </MemoryRouter>
        </ScenarioProvider>
      </AppProviders>,
    );
    const review = await screen.findByTestId("governed-checklist-review");
    expect(within(review).getByRole("heading", {
      name: "No current governed candidates",
    })).toBeVisible();
    expect(within(review).queryByRole("button", { name: "Technically approve" })).toBeNull();
    expect(within(review).queryByRole("button", { name: "Publish checklist version" })).toBeNull();
    expect(within(review).queryByRole("button", { name: /Select governed candidate/ })).toBeNull();
    const blocked = within(review).getByTestId("blocked-governed-generation");
    expect(within(blocked).getByRole("heading", {
      name: EXACT_BLOCKED_REAL_OPS_AOC_REQUEST.requestId,
    })).toBeVisible();
    for (const issue of EXACT_BLOCKED_REAL_OPS_AOC_REQUEST.unresolvedSourceGaps) {
      expect(within(blocked).getByText(issue.gapId)).toBeVisible();
    }
  });

  it("renders every authorized queue item and preserves the exact selected detail identity", async () => {
    const runtime = createMockBackendRuntime();
    const admin = runtime.backendForRole("admin").adminWorkspace;
    const imported = await admin.importGovernedGenerationRun({
      operationId: "TASK6-UI-QUEUE-IMPORT",
      idempotencyKey: "TASK6-UI-QUEUE-IMPORT",
      candidateBundle: SYNTHETIC_GOVERNED_BUNDLE,
    });
    const submitted = await admin.submitGovernedCandidateReview({
      operationId: "TASK6-UI-QUEUE-SUBMIT",
      idempotencyKey: "TASK6-UI-QUEUE-SUBMIT",
      candidateId: imported.candidate!.candidateId,
      expectedRevision: imported.candidate!.revision,
      expectedContentDigest: imported.candidate!.contentDigest,
      reason: "Submit exact candidate for queue selection.",
    });
    const managerBackend = runtime.backendForRole("manager");
    const first = await managerBackend.governedChecklistReview.getCandidate({
      candidateId: submitted.candidateId,
    });
    const second = {
      ...structuredClone(first),
      candidate: {
        ...structuredClone(first.candidate),
        candidateId: "CAND-SYNTHETIC-OPS-AOC-SECOND",
        candidateRootId: "CAND-SYNTHETIC-OPS-AOC-SECOND",
      },
    };
    const selected: string[] = [];
    const queueBackend = {
      ...managerBackend,
      governedChecklistReview: {
        ...managerBackend.governedChecklistReview,
        listQueue: async () => ({ items: [first, second] }),
        getCandidate: async ({ candidateId }: { candidateId: string }) => {
          selected.push(candidateId);
          return candidateId === second.candidate.candidateId ? second : first;
        },
      },
    };

    render(
      <AppProviders runtime={{
        backend: runtime.backend,
        backendForRole: (role) => role === "manager" ? queueBackend : runtime.backendForRole(role),
        buildProfile: "demo",
        environmentLabel: "test",
        identityMode: "demo-role-switch",
        subjectId: "USR-MANAGER-NORA",
      }}>
        <ScenarioProvider>
          <MemoryRouter initialEntries={["/department-manager/checklist-management"]}>
            <ChecklistManagementPage />
          </MemoryRouter>
        </ScenarioProvider>
      </AppProviders>,
    );

    const review = await screen.findByTestId("governed-checklist-review");
    expect(within(review).getByRole("button", {
      name: `Select governed candidate ${first.candidate.candidateId}`,
    })).toBeVisible();
    const secondSelector = within(review).getByRole("button", {
      name: `Select governed candidate ${second.candidate.candidateId}`,
    });
    expect(secondSelector).toBeVisible();
    await userEvent.setup().click(secondSelector);
    await waitFor(() => expect(selected).toEqual([second.candidate.candidateId]));
    expect(within(review).getByRole("heading", {
      name: second.candidate.candidateId,
    })).toBeVisible();
  });

  it("shows exact lineage, owners, decisions, blockers, and keeps approval separate from publication", async () => {
    const runtime = createMockBackendRuntime();
    const admin = runtime.backendForRole("admin").adminWorkspace;
    const imported = await admin.importGovernedGenerationRun({
      operationId: "TASK6-UI-IMPORT",
      idempotencyKey: "TASK6-UI-IMPORT",
      candidateBundle: SYNTHETIC_GOVERNED_BUNDLE,
    });
    const candidate = imported.candidate!;
    await admin.submitGovernedCandidateReview({
      operationId: "TASK6-UI-SUBMIT",
      idempotencyKey: "TASK6-UI-SUBMIT",
      candidateId: candidate.candidateId,
      expectedRevision: candidate.revision,
      expectedContentDigest: candidate.contentDigest,
      reason: "Submit exact candidate for UI review.",
    });

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
          <MemoryRouter initialEntries={["/department-manager/checklist-management"]}>
            <ChecklistManagementPage />
          </MemoryRouter>
        </ScenarioProvider>
      </AppProviders>,
    );

    const review = await screen.findByTestId("governed-checklist-review");
    expect(within(review).getByText(candidate.candidateId)).toBeVisible();
    expect(within(review).getByText("SYNTHETIC-OPS-AOC")).toBeVisible();
    expect(within(review).getByText("Synthetic OPS/AOC 1")).toBeVisible();
    expect(within(review).getAllByText("FLIGHT_OPERATIONS_INSPECTORATE")).toHaveLength(2);
    expect(within(review).getByText("No blocking issues")).toBeVisible();
    expect(within(review).getByText("No decisions recorded")).toBeVisible();
    expect(within(review).getByRole("button", { name: "Technically approve" })).toBeEnabled();
    expect(within(review).queryByRole("button", { name: "Publish checklist version" })).toBeNull();

    const user = userEvent.setup();
    await user.type(within(review).getByLabelText("Decision reason"), "Exact source lineage and owner scope verified.");
    await user.click(within(review).getByRole("button", { name: "Technically approve" }));
    await waitFor(() => expect(within(review).getAllByText("TECHNICALLY_APPROVED").length).toBeGreaterThan(0));
    expect(within(review).getByText("USR-MANAGER-NORA")).toBeVisible();
    expect(within(review).queryByRole("button", { name: "Technically approve" })).toBeNull();
    expect(within(review).getByRole("button", { name: "Publish checklist version" })).toBeEnabled();

    await user.clear(within(review).getByLabelText("Decision reason"));
    await user.type(within(review).getByLabelText("Decision reason"), "Publish the separately approved exact candidate.");
    await user.click(within(review).getByRole("button", { name: "Publish checklist version" }));
    expect(await within(review).findByText(/CTV-GOV-/)).toBeVisible();
    expect(within(review).getByText(/published as immutable Checklist Template Version/i)).toBeVisible();
    expect(within(review).queryByRole("button", {
      name: `Select governed candidate ${candidate.candidateId}`,
    })).toBeNull();
    expect(within(review).getByText("PUBLISHED")).toBeVisible();
  });

  it.each([
    ["Return for revision", "RETURNED"],
    ["Reject candidate", "REJECTED"],
  ] as const)("refreshes the active queue after %s while preserving terminal detail", async (buttonName, status) => {
    const runtime = createMockBackendRuntime();
    const admin = runtime.backendForRole("admin").adminWorkspace;
    const imported = await admin.importGovernedGenerationRun({
      operationId: `TASK6-UI-${status}-IMPORT`,
      idempotencyKey: `TASK6-UI-${status}-IMPORT`,
      candidateBundle: SYNTHETIC_GOVERNED_BUNDLE,
    });
    const candidate = imported.candidate!;
    await admin.submitGovernedCandidateReview({
      operationId: `TASK6-UI-${status}-SUBMIT`,
      idempotencyKey: `TASK6-UI-${status}-SUBMIT`,
      candidateId: candidate.candidateId,
      expectedRevision: candidate.revision,
      expectedContentDigest: candidate.contentDigest,
      reason: `Submit exact candidate for ${status}.`,
    });
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
          <MemoryRouter initialEntries={["/department-manager/checklist-management"]}>
            <ChecklistManagementPage />
          </MemoryRouter>
        </ScenarioProvider>
      </AppProviders>,
    );
    const review = await screen.findByTestId("governed-checklist-review");
    const user = userEvent.setup();
    await user.type(
      within(review).getByLabelText("Decision reason"),
      `Persist exact ${status} terminal detail.`,
    );
    await user.click(within(review).getByRole("button", { name: buttonName }));
    await waitFor(() => expect(within(review).queryByRole("button", {
      name: `Select governed candidate ${candidate.candidateId}`,
    })).toBeNull());
    expect(within(review).getAllByText(status).length).toBeGreaterThan(0);
    expect(within(review).getByText((_content, element) =>
      element?.tagName === "SMALL" &&
      element.textContent?.endsWith(`Persist exact ${status} terminal detail.`) === true,
    )).toBeVisible();
    expect(within(review).queryByRole("button", { name: "Technically approve" })).toBeNull();
    expect(within(review).queryByRole("button", { name: "Publish checklist version" })).toBeNull();
  });
});
