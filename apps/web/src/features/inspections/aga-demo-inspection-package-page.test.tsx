// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import type {
  AGADemoWorkspaceBackend,
  AGADemoWorkspaceBatchPreview,
  AGADemoWorkspaceCapability,
  AGADemoWorkspaceClassificationReviewItem,
  AGADemoWorkspaceCommand,
  AGADemoWorkspaceDraft,
  AGADemoWorkspaceLifecycleProjection,
  AGADemoWorkspaceQuery,
  AGADemoWorkspaceQueryResponse,
  AGADemoWorkspaceSimulationSetup,
} from "../../backend/aga-demo-workspace";
import { BackendHttpError } from "../../backend/http-backend";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { AGADemoInspectionPackagePage } from "./aga-demo-inspection-package-page";

afterEach(() => cleanup());

const digest = `sha256:${"0".repeat(64)}`;
const generation = { generationId: "aga-ws-generation-0001", state: "ACTIVE", revision: 2, sealDigest: digest };
const capability: AGADemoWorkspaceCapability = { available: true, projection: "DEPARTMENT_MANAGER_SCOPED", classificationEnabled: true, recommendationEnabled: true, lifecycleEnabled: true, resetEnabled: false };
const draft: AGADemoWorkspaceDraft = {
  draftId: "aga-ws-draft-0001",
  generationId: generation.generationId,
  revision: 4,
  contentDigest: digest,
  items: [{ currentLeaf: true, draftDisposition: "UNSET" }],
};
const setupBase: AGADemoWorkspaceSimulationSetup = {
  generationId: generation.generationId,
  generationRevision: generation.revision,
  generationSealDigest: generation.sealDigest,
  draftId: draft.draftId!,
  draftRevision: draft.revision!,
  draftContentDigest: draft.contentDigest!,
  taxonomyVersion: "aga-taxonomy-2026.08",
  taxonomyDigest: digest,
  classificationRunId: "aga-classification-run-demo-001",
  classificationRunDigest: digest,
  organizationLabel: "Fly Namibia",
  providerLabel: "Aerodrome operator",
  targetLabel: "Windhoek Hosea Kutako",
  providerScopeRootId: "scope-root-001",
  providerScopeId: "scope-001",
  providerScopeVersion: 1,
  providerTypeId: "provider-type-aerodrome-operator",
  providerTypeCode: "AERODROME_OPERATOR",
  departmentId: "department-001",
  organizationalUnitId: "unit-001",
  targetId: "target-001",
  canonicalTargetKind: "AERODROME_OPERATOR",
  targetProfileCode: "AERODROME_PROFILE",
  inspectionProfileCode: "AGA_PROFILE",
  inspectionTypeCode: "AGA_INSPECTION",
  operationQualifiers: [],
  activityQualifiers: [],
  effectiveAt: "2026-08-05T09:00:00Z",
  providerScopeDigest: digest,
  readinessState: "WORKING",
  currentLeafCount: 1,
  includedCount: 0,
  excludedCount: 0,
  deferredCount: 0,
  unsetCount: 1,
  includedEligibleCount: 0,
  includedIneligibleCount: 0,
  includedBlockerCount: 0,
  includedSourceGapCount: 0,
  formDistribution: { "FSS-AGA-FORM-001": 1 },
  domainDistribution: { "DOMAIN-A": 1 },
  topicDistribution: { "TOPIC-A": 1 },
  inspectorChoices: [{ selectionPin: "inspector-pin-001", label: "CAA Inspector · AGA Demo", role: "INSPECTOR" }],
  leadChoices: [{ selectionPin: "lead-pin-001", label: "Lead Inspector · AGA Demo", role: "LEAD_INSPECTOR" }],
  simulationSetupDigest: digest,
};
const row: AGADemoWorkspaceClassificationReviewItem = {
  identity: { packageVersion: "AGA_PACKAGE", packageJsonSha256: digest, formCode: "FSS-AGA-FORM-001", proposalId: "proposal-001", ordinal: 1, textDigest: digest },
  questionKey: "question-001",
  questionRef: { questionOrigin: "SEALED_BASE" as const, packageVersion: "AGA_PACKAGE", packageJsonSha256: digest, formCode: "FSS-AGA-FORM-001", proposalId: "proposal-001", ordinal: 1, textDigest: digest },
  questionOrigin: "SEALED_BASE" as const,
  includeEligible: true,
  includeEligibilityReason: "ELIGIBLE_FOR_CURRENT_SIMULATION_SCOPE",
  projection: { mainDomainCode: "DOMAIN-A", topicCodes: ["TOPIC-A"] },
  agreementConfidence: "HIGH",
  recommendationState: "AUTO_PROPOSED_HIGH_CONFIDENCE",
  governance: { sourceMappingState: "MAPPED", sourceAuthorityState: "ATTESTED", riskClassificationState: "CANDIDATE", decisionState: "NOT_DECIDED", extractionState: "EXTRACTED", questionSourceProposalGap: false, externalApplicabilityUnresolved: false, blockerCodes: [] },
  itemSemanticDigest: digest,
  candidateDigest: digest,
  challengeDigest: digest,
  draftDisposition: "UNSET",
  questionText: "Is the aerodrome operator maintaining the required safety management records?",
  questionTextDigest: digest,
  textOrigin: "SEALED_BASE",
};

function queryResponse(operation: string, overrides: Partial<AGADemoWorkspaceQueryResponse> = {}): AGADemoWorkspaceQueryResponse {
  return { operation, generation, itemCount: 1, lifecycleAvailable: true, ...overrides } as AGADemoWorkspaceQueryResponse;
}

function lifecycle(): AGADemoWorkspaceLifecycleProjection {
  return {
    inspectionId: "inspection-001",
    generationId: generation.generationId,
    organizationId: "CAA",
    providerScopeId: "scope-001",
    state: "READY",
    revision: 1,
    questions: [],
    responses: [],
    potentialFindings: [],
    findings: [],
    capRevisions: [],
    evidenceVersions: [],
    verificationDecisions: [],
    currentOwnerRole: "Assigned Inspector",
    nextAction: "Start inspection",
    updatedAt: "2026-08-05T09:00:00Z",
    digest,
  } as AGADemoWorkspaceLifecycleProjection;
}

function createBackend(previewOverrides: Partial<AGADemoWorkspaceBatchPreview> = {}) {
  let setup = setupBase;
  let currentDraft = draft;
  let currentRecommendation: AGADemoWorkspaceQueryResponse["recommendationSnapshot"];
  let currentInspection: AGADemoWorkspaceLifecycleProjection | null = null;
  const classificationQuery = vi.fn(async (request: AGADemoWorkspaceQuery) => {
    if (request.operationId === "GET_SIMULATION_SETUP") return queryResponse(request.operationId, { simulationSetup: setup });
    if (request.operationId === "GET_DRAFT") return queryResponse(request.operationId, { draft: currentDraft });
    if (request.operationId === "SEARCH_ITEMS") return queryResponse(request.operationId, { items: [row], page: request.page ?? 0, pageSize: 25 });
    return queryResponse(request.operationId, { recommendationSnapshot: currentRecommendation });
  });
  const classificationCommand = vi.fn(async (command: AGADemoWorkspaceCommand) => {
    if (command.operationId === "EXECUTE_BATCH") {
      currentDraft = { ...currentDraft, revision: currentDraft.revision! + 1, items: [{ currentLeaf: true, draftDisposition: "INCLUDE" }] };
      setup = { ...setup, draftRevision: currentDraft.revision!, draftContentDigest: digest, includedCount: 1, unsetCount: 0, includedEligibleCount: 1 };
      return { operationId: command.operationId, replayed: false, lifecycleAvailable: true, draft: currentDraft };
    }
    if (command.operationId === "MARK_READY_FOR_DEMO_SIMULATION") {
      setup = { ...setup, readinessState: "READY_FOR_DEMO_SIMULATION", readinessEventDigest: digest };
    }
    const batchPreview: AGADemoWorkspaceBatchPreview = { previewId: "server-preview-001", generationId: generation.generationId, draftId: currentDraft.draftId!, draftRevision: currentDraft.revision!, draftContentDigest: currentDraft.contentDigest!, classificationRunDigest: digest, filter: { formCode: "FSS-AGA-FORM-001", disposition: "UNSET" as const }, filterDigest: digest, affectedIdentityDigest: digest, action: "INCLUDE" as const, reasonCode: "MANAGER_SCOPE_DECISION", count: 1, currentDisposition: { include: 0, exclude: 0, defer: 0, unset: 1 }, eligibleCount: 1, ineligibleCount: 0, blockerCount: 0, sourceGapCount: 0, expiresAt: "2026-08-05T10:00:00Z", previewDigest: digest, ...previewOverrides };
    return { operationId: command.operationId, replayed: false, lifecycleAvailable: true, batchPreview };
  });
  const recommendationCommand = vi.fn(async (command: AGADemoWorkspaceCommand) => {
    if (command.operationId === "CREATE_RECOMMENDATION") {
      currentRecommendation = { recommendation: { recommendationId: "recommendation-001", revision: 1, draftId: currentDraft.draftId!, draftRevision: currentDraft.revision!, digest, items: [] } as never, createdAt: "2026-08-05T09:00:00Z", snapshotDigest: digest } as never;
      setup = { ...setup, recommendationId: "recommendation-001", recommendationRevision: 1, recommendationDigest: digest };
      return { operationId: command.operationId, replayed: false, lifecycleAvailable: true, recommendation: currentRecommendation };
    }
    currentInspection = lifecycle();
    return { operationId: "CREATE_INSPECTION", replayed: false, lifecycleAvailable: true, lifecycle: currentInspection };
  });
  const backend: AGADemoWorkspaceBackend = {
    capability: vi.fn().mockResolvedValue(capability),
    classificationQuery,
    classificationCommand,
    recommendationCommand,
    lifecycleQuery: vi.fn().mockImplementation(async () => queryResponse("GET_CURRENT_INSPECTION", { ...(currentInspection ? { currentInspection, lifecycle: currentInspection } : {}) })),
    lifecycleCommand: vi.fn(),
    adminCommand: vi.fn(),
  };
  return { backend, classificationQuery, classificationCommand, recommendationCommand };
}

function renderPage(backend: AGADemoWorkspaceBackend) {
  const runtime = createMockBackendRuntime();
  render(
    <AppProviders runtime={{ ...runtime, backend: { ...runtime.backend, agaDemoWorkspace: backend }, buildProfile: "http", environmentLabel: "test", subjectId: "manager-aga-demo" }}>
      <MemoryRouter><AGADemoInspectionPackagePage capability={capability} role="manager" roleLabel="Department Manager" /></MemoryRouter>
    </AppProviders>,
  );
}

describe("AGA Department Manager inspection package builder", () => {
  it("renders a bounded local state when the package inventory is not provisioned", async () => {
    const { backend, classificationQuery } = createBackend();
    classificationQuery.mockRejectedValue(new BackendHttpError("Backend request failed with status 404", 404, null, null, null));
    renderPage(backend);
    expect(await screen.findByRole("alert")).toHaveTextContent("The inspection package inventory is not provisioned in this local AGA workspace.");
    expect(screen.queryByText("Backend request failed with status 404")).not.toBeInTheDocument();
  });

  it("uses a server-issued digest-bound preview and exact atomic confirmation without a client preview id", async () => {
    const { backend, classificationCommand } = createBackend();
    renderPage(backend);
    const user = userEvent.setup();
    await screen.findByText("1,310 candidate AGA questions; bounded pages only");
    await user.click(screen.getByRole("button", { name: /Package preview/ }));
    await user.click(screen.getByRole("button", { name: "Create server preview" }));
    await waitFor(() => expect(classificationCommand).toHaveBeenCalledWith(expect.objectContaining({ operationId: "PREVIEW_BATCH", simulationSetupDigest: digest, batchFilter: { formCode: "FSS-AGA-FORM-001", disposition: "UNSET" } })));
    const previewCommand = classificationCommand.mock.calls.find(([command]) => command.operationId === "PREVIEW_BATCH")?.[0] as AGADemoWorkspaceCommand;
    expect(previewCommand).not.toHaveProperty("previewId");
    expect(await screen.findByText(/server-preview-001/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Confirm simulation disposition" }));
    await waitFor(() => expect(classificationCommand).toHaveBeenCalledWith(expect.objectContaining({ operationId: "EXECUTE_BATCH", previewId: "server-preview-001", previewDigest: digest, simulationSetupDigest: digest })));
  });

  it("returns filtered inventory to the first page after searching from a later page", async () => {
    const { backend, classificationQuery } = createBackend();
    classificationQuery.mockImplementation(async (request: AGADemoWorkspaceQuery) => {
      if (request.operationId === "GET_SIMULATION_SETUP") return queryResponse(request.operationId, { simulationSetup: setupBase });
      if (request.operationId === "GET_DRAFT") return queryResponse(request.operationId, { draft });
      const page = request.page ?? 0;
      const searched = Boolean(request.search);
      return queryResponse(request.operationId, {
        items: [row],
        page,
        pageSize: 25,
        nextPage: !searched && page === 0 ? 1 : undefined,
      });
    });
    renderPage(backend);
    const user = userEvent.setup();
    await screen.findByText("1,310 candidate AGA questions; bounded pages only");
    await user.click(screen.getByRole("button", { name: "Next page" }));
    await waitFor(() => expect(classificationQuery).toHaveBeenCalledWith(expect.objectContaining({ operationId: "SEARCH_ITEMS", page: 1 }), expect.anything()));
    await user.type(screen.getByRole("textbox", { name: "Package inventory search" }), "FSS-AGA-FORM-053");
    await waitFor(() => expect(classificationQuery).toHaveBeenCalledWith(expect.objectContaining({ operationId: "SEARCH_ITEMS", page: 0, search: "FSS-AGA-FORM-053" }), expect.anything()));
    expect(await screen.findByText(row.questionText!)).toBeInTheDocument();
  });

  it("blocks confirmation of a mixed eligible and ineligible Include preview", async () => {
    const { backend, classificationCommand } = createBackend({ count: 2, eligibleCount: 1, ineligibleCount: 1 });
    renderPage(backend);
    const user = userEvent.setup();
    await screen.findByText("1,310 candidate AGA questions; bounded pages only");
    await user.click(screen.getByRole("button", { name: /Package preview/ }));
    await user.click(screen.getByRole("button", { name: "Create server preview" }));
    const confirm = await screen.findByRole("button", { name: "Confirm simulation disposition" });
    expect(confirm).toBeDisabled();
    expect(confirm).toHaveAccessibleDescription(/includes ineligible rows/i);
    await user.click(confirm);
    expect(classificationCommand).not.toHaveBeenCalledWith(expect.objectContaining({ operationId: "EXECUTE_BATCH" }));
  });

  it("completes readiness, recommendation, and role-pinned release with no browser readiness identifier", async () => {
    const { backend, classificationCommand, recommendationCommand } = createBackend();
    renderPage(backend);
    const user = userEvent.setup();
    await screen.findByText("1,310 candidate AGA questions; bounded pages only");
    await user.click(screen.getByRole("button", { name: /Package preview/ }));
    await user.click(screen.getByRole("button", { name: "Create server preview" }));
    await user.click(await screen.findByRole("button", { name: "Confirm simulation disposition" }));
    await waitFor(() => expect(classificationCommand).toHaveBeenCalledWith(expect.objectContaining({ operationId: "EXECUTE_BATCH" })));
    await user.click(screen.getByRole("button", { name: "3 · Simulation release" }));
    await user.click(screen.getByRole("button", { name: "Mark ready for demo simulation" }));
    await waitFor(() => expect(classificationCommand).toHaveBeenCalledWith(expect.objectContaining({ operationId: "MARK_READY_FOR_DEMO_SIMULATION", simulationSetupDigest: digest })));
    const readyCommand = classificationCommand.mock.calls.find(([command]) => command.operationId === "MARK_READY_FOR_DEMO_SIMULATION")?.[0] as AGADemoWorkspaceCommand;
    expect(readyCommand).not.toHaveProperty("readinessEventId");
    await user.click(screen.getByRole("button", { name: "Create synthetic recommendation" }));
    await waitFor(() => expect(recommendationCommand).toHaveBeenCalledWith(expect.objectContaining({ operationId: "CREATE_RECOMMENDATION", simulationSetupDigest: digest, draftId: draft.draftId })));
    await user.selectOptions(screen.getByRole("combobox", { name: "Inspector selection" }), "inspector-pin-001");
    await user.selectOptions(screen.getByRole("combobox", { name: "Lead Inspector selection" }), "lead-pin-001");
    await user.click(screen.getByRole("button", { name: "Release synthetic inspection" }));
    await waitFor(() => expect(recommendationCommand).toHaveBeenCalledWith(expect.objectContaining({ operationId: "CREATE_INSPECTION", simulationSetupDigest: digest, inspectorSelectionPin: "inspector-pin-001", leadSelectionPin: "lead-pin-001" })));
    expect(await screen.findByText(/Released immutable inspection snapshot/)).toBeInTheDocument();
    expect(localStorage.getItem("aga-demo-inspection-package")).toBeNull();
  });
});
