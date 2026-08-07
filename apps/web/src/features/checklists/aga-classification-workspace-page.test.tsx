// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import type {
  AGADemoWorkspaceBackend,
  AGADemoWorkspaceQuery,
  AGADemoWorkspaceQueryResponse,
} from "../../backend/aga-demo-workspace";
import { BackendHttpError } from "../../backend/http-backend";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { AGADemoClassificationWorkspacePage } from "./aga-classification-workspace-page";

afterEach(() => cleanup());

const digest = "sha256:0000000000000000000000000000000000000000000000000000000000000000";
const generation = { generationId: "aga-ws-generation-0001", state: "ACTIVE", revision: 3, sealDigest: digest };
const draft = { draftId: "aga-ws-draft-0001", generationId: generation.generationId, revision: 4, contentDigest: digest, items: [{ currentLeaf: true, draftAgreementConfidence: "HIGH", draftReviewState: "AUTO_PRESELECTED", draftRecommendationState: "AUTO_PROPOSED_HIGH_CONFIDENCE", draftDisposition: "INCLUDE" }] };
const row = {
  identity: { packageVersion: "AGA_PACKAGE", packageJsonSha256: digest, formCode: "FSS-AGA-FORM-001", proposalId: "proposal-001", ordinal: 1, textDigest: digest },
  questionKey: "server-returned-question-key",
  questionRef: { questionOrigin: "SEALED_BASE" as const, packageVersion: "AGA_PACKAGE", packageJsonSha256: digest, formCode: "FSS-AGA-FORM-001", proposalId: "proposal-001", ordinal: 1, textDigest: digest },
  questionOrigin: "SEALED_BASE" as const,
  includeEligible: true,
  includeEligibilityReason: "ELIGIBLE_FOR_CURRENT_SIMULATION_SCOPE",
  projection: { mainDomainCode: "DOMAIN_A", topicCodes: ["TOPIC_A"], inspectionProfileCodes: ["PROFILE_A"], inspectionTypeCodes: ["TYPE_A"], canonicalTargetKind: "AERODROME_OPERATOR", targetProfileCode: "AERODROME_PROFILE", operationQualifiers: [], activityQualifiers: [], applicabilityDisposition: "APPLICABLE", evidenceExpectationCodes: [], externalInvolvements: [] },
  agreementConfidence: "HIGH",
  recommendationState: "AUTO_PROPOSED_HIGH_CONFIDENCE",
  governance: { sourceMappingState: "SOURCE_MAPPING_REQUIRED", sourceAuthorityState: "NOT_ATTESTED", riskClassificationState: "CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW", decisionState: "DECISION_NOT_SUPPLIED", extractionState: "EXTRACTED_CANDIDATE", questionSourceProposalGap: false, externalApplicabilityUnresolved: false, blockerCodes: [] },
  itemSemanticDigest: digest,
  candidateDigest: digest,
  challengeDigest: digest,
  draftAgreementConfidence: "HIGH",
  draftRecommendationState: "AUTO_PROPOSED_HIGH_CONFIDENCE",
  draftReviewState: "AUTO_PRESELECTED",
  draftDisposition: "INCLUDE",
};

function response(operation: string, overrides: Partial<AGADemoWorkspaceQueryResponse> = {}): AGADemoWorkspaceQueryResponse {
  return {
    operation,
    generation,
    itemCount: 1,
    lifecycleAvailable: false,
    ...overrides,
  } as AGADemoWorkspaceQueryResponse;
}

function createWorkspaceBackend(resetEnabled = false, rowOverrides: Partial<typeof row> = {}) {
  const responses: Record<string, AGADemoWorkspaceQueryResponse> = {
    GET_SUMMARY: response("GET_SUMMARY", { baseQuestionCount: 1310, draftIncludedCount: 1, draftExcludedCount: 0, draftDeferredCount: 0 }),
    GET_DRAFT: response("GET_DRAFT", { draft }),
    GET_PROVIDER_CONFIGURATION: response("GET_PROVIDER_CONFIGURATION", { providerConfiguration: [
      { providerTypeCode: "AERODROME_OPERATOR", disposition: "INSPECTED_SCOPE_ELIGIBLE", reasonCode: "CANDIDATE_PROFILE_BOUNDARY" },
      { providerTypeCode: "ANSP", disposition: "INVOLVEMENT_ONLY", reasonCode: "CANDIDATE_PROFILE_BOUNDARY" },
    ] }),
    GET_HISTORY: response("GET_HISTORY", { history: [generation] }),
    SEARCH_ITEMS: response("SEARCH_ITEMS", { items: [{ ...row, ...rowOverrides }], page: 0, pageSize: 25 }),
  };
  const classificationQuery = vi.fn(async (input: AGADemoWorkspaceQuery) => responses[input.operationId] ?? responses.SEARCH_ITEMS);
  const classificationCommand = vi.fn(async (input: never) => ({ operationId: "INCLUDE", replayed: false, lifecycleAvailable: false, draft, input }));
  const adminCommand = vi.fn(async (input: never) => ({ operationId: "RESET_GENERATION", replayed: false, lifecycleAvailable: false, generation, input }));
  const backend: AGADemoWorkspaceBackend = {
    capability: vi.fn().mockResolvedValue({ available: true, projection: resetEnabled ? "CAA_ADMIN" : "DEPARTMENT_MANAGER_SCOPED", classificationEnabled: true, recommendationEnabled: true, lifecycleEnabled: false, resetEnabled }),
    classificationQuery,
    classificationCommand,
    recommendationCommand: vi.fn(),
    lifecycleQuery: vi.fn(),
    lifecycleCommand: vi.fn(),
    adminCommand,
  };
  return { backend, classificationQuery, classificationCommand, adminCommand };
}

function renderPage(role: "manager" | "admin" = "manager", rowOverrides: Partial<typeof row> = {}, queryError?: Error) {
  const runtime = createMockBackendRuntime();
  const workspace = createWorkspaceBackend(role === "admin", rowOverrides);
  if (queryError) workspace.classificationQuery.mockRejectedValue(queryError);
  render(
    <AppProviders runtime={{ backend: { ...runtime.backend, agaDemoWorkspace: workspace.backend }, buildProfile: "http", environmentLabel: "test" }}>
      <MemoryRouter>
        <AGADemoClassificationWorkspacePage capability={{ available: true, projection: role === "admin" ? "CAA_ADMIN" : "DEPARTMENT_MANAGER_SCOPED", classificationEnabled: role !== "admin", recommendationEnabled: role !== "admin", lifecycleEnabled: false, resetEnabled: role === "admin" }} role={role} roleLabel={role === "admin" ? "Admin Preview" : "Department Manager"} />
      </MemoryRouter>
    </AppProviders>,
  );
  return workspace;
}

async function selectRow() {
  await screen.findByTestId("aga-classification-workspace-page");
  await screen.findByRole("button", { name: /open controls/i });
  await userEvent.setup().click(screen.getByRole("button", { name: /open controls/i }));
}

describe("AGA classification workspace page", () => {
  it("renders authorized supplemental workspace", async () => {
    renderPage();
    expect(await screen.findByTestId("aga-classification-workspace-page")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Classify AGA questions" })).toBeInTheDocument();
    expect(screen.getByTestId("aga-demo-guidance")).toHaveTextContent(/Find.*Compare.*Decide/s);
    expect(screen.getByText("1310")).toBeInTheDocument();
  });

  it("turns an unprovisioned 404 into a bounded local workspace state", async () => {
    renderPage("manager", {}, new BackendHttpError("Backend request failed with status 404", 404, null, null, null));
    expect(await screen.findByRole("alert")).toHaveTextContent("Classification inventory is not provisioned in this local AGA workspace.");
    expect(screen.queryByText("Backend request failed with status 404")).not.toBeInTheDocument();
  });

  it("routes the manager to the fixed server-owned inspection package builder", async () => {
    const workspace = renderPage();
    expect(await screen.findByRole("link", { name: /open inspection package builder/i })).toHaveAttribute("href", "/department-manager/aga-demo-workspace/inspection-package");
    expect(workspace.classificationCommand).not.toHaveBeenCalledWith(expect.objectContaining({ operationId: "PREVIEW_BATCH" }));
  });

  it("creates immutable wording successor", async () => {
    const workspace = renderPage();
    await selectRow();
    const user = userEvent.setup();
    await user.type(screen.getByRole("textbox", { name: /candidate or successor wording/i }), "A materially different successor wording");
    await user.click(screen.getByRole("button", { name: /create immutable wording successor/i }));
    await waitFor(() => expect(workspace.classificationCommand).toHaveBeenCalledWith(expect.objectContaining({ operationId: "REWORD_CANDIDATE", targetQuestionKey: row.questionKey, workspaceBody: "A materially different successor wording" })));
  });

  it("resolves every controlled proposal family", async () => {
    const workspace = renderPage();
    await selectRow();
    const user = userEvent.setup();
    for (const [index, [label, operation]] of [[0, [/use candidate proposal/i, "CANDIDATE"]], [1, [/use challenge proposal/i, "CHALLENGE"]], [2, [/set exact proposal/i, "SET_EXACT"]]] as const) {
      if (index > 0) await selectRow();
      const button = screen.getByRole("button", { name: label });
      await waitFor(() => expect(button).not.toBeDisabled());
      await user.click(button);
      await waitFor(() => expect(workspace.classificationCommand).toHaveBeenCalledWith(expect.objectContaining({ operationId: "RESOLVE_CLASSIFICATION_PROPOSALS", resolutionMode: operation })));
    }
  });

  it("shows provider eligibility", async () => {
    renderPage();
    expect(await screen.findByText(/1 inspected-scope eligible/i)).toBeInTheDocument();
    expect(screen.getByText(/ANSP · INVOLVEMENT_ONLY/i)).toBeInTheDocument();
  });

  it("fails closed for an ineligible server-returned Include row", async () => {
    const workspace = renderPage("manager", { includeEligible: false, includeEligibilityReason: "TARGET_PROFILE_MISMATCH" });
    await selectRow();
    const include = screen.getByRole("button", { name: /include selected item/i });
    expect(include).toBeDisabled();
    expect(include).toHaveAccessibleDescription(/TARGET_PROFILE_MISMATCH/);
    await userEvent.setup().click(include);
    expect(workspace.classificationCommand).not.toHaveBeenCalledWith(expect.objectContaining({ operationId: "INCLUDE" }));
  });

  it("admin resets generation with exact CAS", async () => {
    const workspace = renderPage("admin");
    const user = userEvent.setup();
    await screen.findByTestId("aga-workspace-admin-page");
    await user.click(screen.getByRole("button", { name: /view generation history/i }));
    expect((await screen.findAllByText(/aga-ws-generation-0001/)).length).toBeGreaterThan(0);
    await user.type(screen.getByRole("textbox", { name: /reset reason/i }), "Reset disposable test generation");
    await user.click(screen.getByRole("checkbox", { name: /confirm generation reset/i }));
    await user.click(screen.getByRole("button", { name: /reset generation/i }));
    await waitFor(() => expect(workspace.adminCommand).toHaveBeenCalled());
    expect(workspace.adminCommand.mock.calls[0]?.[0]).toMatchObject({ operationId: "RESET_GENERATION", expectedGenerationId: generation.generationId, expectedGenerationRevision: generation.revision, expectedGenerationSealDigest: generation.sealDigest });
    expect(workspace.classificationCommand).not.toHaveBeenCalled();
    expect(workspace.classificationQuery.mock.calls.map(([request]) => request.operationId)).not.toContain("GET_DRAFT");
    expect(workspace.classificationQuery.mock.calls.map(([request]) => request.operationId)).not.toContain("SEARCH_ITEMS");
  });

  it("purges sensitive state on BFCache restore and never writes browser storage", async () => {
    renderPage();
    await screen.findByTestId("aga-classification-workspace-page");
    expect(window.localStorage.length).toBe(0);
    expect(window.sessionStorage.length).toBe(0);
    const event = new Event("pageshow");
    Object.defineProperty(event, "persisted", { value: true });
    window.dispatchEvent(event);
    expect(await screen.findByText(/restored page was cleared/i)).toBeInTheDocument();
  });
});
