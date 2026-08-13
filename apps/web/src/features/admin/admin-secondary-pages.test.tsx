// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { act, cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DEMO_MOCK_STORAGE_KEY } from "../../app/demo-persistence";
import { AppProviders } from "../../app/providers";
import { AppRouter } from "../../app/router";
import { ScenarioProvider } from "../../app/scenario-context";
import type { AdminRegulatoryReferenceView, AuditEventView, DemoBackend } from "../../backend/backend";
import {
  SYNTHETIC_EDITED_RATIONALE,
  SYNTHETIC_GOVERNED_BUNDLE,
} from "../../backend/governed-synthetic-profile";
import {
  createMockBackendPersistentRuntime,
  createMockBackendRuntime,
} from "../../mock/create-mock-backend";
import type { MockState } from "../../mock/seed-data";
import { ChecklistBuilderPage } from "./checklist-builder-page";
import { QuestionBankPage } from "./question-bank-page";
import { useAdminLoad, useAdminWorkspace } from "./admin-workspace-shared";

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

interface AdminQuestion {
  id: string;
  prompt: string;
  configuredReference: string;
  expectedEvidence: string;
  revision: number;
}

interface AdminTemplateVersion {
  id: string;
  templateId: "TPL-CABIN-2026";
  version: number;
  status: "PUBLISHED" | "DRAFT";
  owner: "Department Manager" | "Admin Preview";
  creatorSubjectId: string;
  changeReason: string;
  questionIds: string[];
  revision: number;
}

interface AdminWorkspaceCapability {
  listRegulatoryReferences(input: { search?: string; status?: string }): Promise<{ items: AdminRegulatoryReferenceView[] }>;
  listTemplateMasters(input: Record<string, never>): Promise<{ items: Array<{ id: string; publishedVersionId: string; owner: string; itemCount: number }> }>;
  listQuestions(input: { search?: string }): Promise<{ items: AdminQuestion[] }>;
  createQuestion(input: { prompt: string; configuredReference: string; expectedEvidence: string; expectedRevision: number | null; idempotencyKey: string }): Promise<AdminQuestion>;
  getTemplate(input: { templateId: string }): Promise<{ id: string; publishedVersionId: string; versions: AdminTemplateVersion[]; revision: number }>;
  createDraft(input: { templateId: string; expectedRevision: number; idempotencyKey: string; changeReason: string }): Promise<AdminTemplateVersion>;
  addDraftQuestion(input: { templateId: string; draftVersionId: string; questionId: string; expectedRevision: number; idempotencyKey: string }): Promise<AdminTemplateVersion>;
  moveDraftQuestion(input: { templateId: string; draftVersionId: string; questionId: string; direction: "UP" | "DOWN"; expectedRevision: number; idempotencyKey: string }): Promise<AdminTemplateVersion>;
  getInspectionPackage(input: { packageId: string }): Promise<{ id: string; auditId: string; organizationId: string; questionIds: string[]; configuredReferences: string[]; expectedEvidence: string[]; riskFocus: string[] }>;
  listReportDefinitions(input: { search?: string }): Promise<{ items: Array<{ id: string; packageFields: string[]; actionReason: string }> }>;
  listAccessDirectory(input: { search?: string; role?: string }): Promise<{ items: Array<{ subjectId: string; roles: string[]; organizationId: string | null; email: "Not configured in demo"; mfaEnrolled: boolean; mfaState: string; requiredActions: string[]; invitationState: string; accountStatus: "Not configured in demo"; applicationProfileState: string; membershipId: string | null; membershipState: string; membershipRevision: number; membershipDrift: string; lastSuccessfulSessionAt: string | null; providerObservedAt: string }> }>;
  listOrganizations(input: { search?: string; organizationType?: string; status?: string; scope?: string }): Promise<{ items: Array<{ id: string; legalName: string; organizationType: string; status: string; scope: string; detailAvailable: boolean; disabledReason: string | null }> }>;
  getOrganization(input: { organizationId: string }): Promise<{ id: string; legalName: string; organizationType: string; status: string; scope: string }>;
  listAuditEvents(input: { actor?: string; action?: string; entity?: string; system?: string; dateText?: string }): Promise<{ items: AuditEventView[] }>;
}

type AdminBackend = DemoBackend & { adminWorkspace?: AdminWorkspaceCapability };

function adminBackend(runtime: MockRuntime): AdminBackend {
  return runtime.backendForRole("admin") as AdminBackend;
}

function requireAdminWorkspace(runtime: MockRuntime): AdminWorkspaceCapability {
  const capability = adminBackend(runtime).adminWorkspace;
  expect(capability).toBeDefined();
  if (!capability) throw new Error("Admin workspace capability is missing.");
  return capability;
}

function renderAdminRoute(path: string, runtime: MockRuntime = createMockBackendRuntime()) {
  const view = render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: "demo",
      environmentLabel: "test",
      identityMode: "demo-role-switch",
      subjectId: "USR-ADMIN-ADA",
    }}>
      <ScenarioProvider>
        <MemoryRouter initialEntries={[path]}><AppRouter /></MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
  return { runtime, ...view };
}

function renderAdminComponent(component: ReactNode, runtime: MockRuntime) {
  return render(
    <AppProviders runtime={{ backend: runtime.backend, backendForRole: runtime.backendForRole, buildProfile: "demo", environmentLabel: "test", identityMode: "demo-role-switch", subjectId: "USR-ADMIN-ADA" }}>
      <MemoryRouter>{component}</MemoryRouter>
    </AppProviders>,
  );
}

function AdminCapabilityStabilityProbe() {
  const backend = useAdminWorkspace();
  const result = useAdminLoad(() => backend.listTemplateMasters({}), [backend]);
  return <p>{result.data ? "ready" : "loading"}</p>;
}

beforeEach(() => localStorage.clear());
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("Admin secondary workspaces", () => {
  it("keeps one Admin capability across runtime-wrapper rerenders", async () => {
    const runtime = createMockBackendRuntime();
    const admin = requireAdminWorkspace(runtime);
    const listTemplateMasters = vi.spyOn(admin, "listTemplateMasters");
    listTemplateMasters.mockResolvedValueOnce({ items: [] });
    listTemplateMasters.mockImplementation(() => new Promise(() => undefined));
    const backendForRole: MockRuntime["backendForRole"] = (role) => {
      const backend = runtime.backendForRole(role);
      if (role !== "admin" || !backend.adminWorkspace) return backend;
      return { ...backend, adminWorkspace: { ...backend.adminWorkspace } };
    };

    render(
      <AppProviders runtime={{
        backend: runtime.backend,
        backendForRole,
        buildProfile: "demo",
        environmentLabel: "test",
        identityMode: "demo-role-switch",
        subjectId: "USR-ADMIN-ADA",
      }}>
        <MemoryRouter><AdminCapabilityStabilityProbe /></MemoryRouter>
      </AppProviders>,
    );

    expect(await screen.findByText("ready")).toBeVisible();
    await act(async () => Promise.resolve());
    expect(listTemplateMasters).toHaveBeenCalledTimes(1);
  });

  it("direct-loads all 13 Admin routes with complete shell navigation, contextual active mapping, and route-specific breadcrumbs", async () => {
    const routes = [
      ["/admin/regulatory-library", "admin-regulatory-library-page", "Regulatory Library", "Regulatory Library"],
      ["/admin/template-library", "admin-template-list-page", "Checklist Templates", "Templates"],
      ["/admin/templates", "admin-template-preview-page", "Template Preview — Cabin Inspection", "Templates"],
      ["/admin/question-bank", "admin-question-bank-page", "Question Bank", "Question Bank"],
      ["/admin/checklist-builder", "admin-checklist-builder-page", "Checklist Builder", "Checklist Builder"],
      ["/admin/templates/TPL-CABIN-2026/history", "admin-version-history-page", "Version History", "Version History"],
      ["/admin/inspection-package-builder", "admin-inspection-package-page", "Inspection Package Builder", "Checklist Builder"],
      ["/admin/reports", "admin-reports-page", "Admin Reports", "Reports"],
      ["/admin/users-roles", "admin-users-roles-page", "Users / Roles", "Users / Roles"],
      ["/admin/configurations", "admin-configurations-page", "Configurations", "Configurations"],
      ["/admin/organization-master-data", "admin-organization-master-data-page", "Organisation Master Data", "Organisation Master Data"],
      ["/admin/organization-master-data/ORG-FLY-NAMIBIA", "admin-organization-detail-page", "Organization Detail", "Organisation Master Data"],
      ["/admin/audit-log", "admin-audit-log-page", "Audit Log", "Audit Log"],
    ] as const;

    for (const [path, testId, heading, activeLabel] of routes) {
      renderAdminRoute(path);
      const page = await screen.findByTestId(testId);
      expect(within(page).getByRole("heading", { level: 1, name: heading })).toBeVisible();
      expect(screen.queryByTestId("route-pending-implementation")).toBeNull();
      const navigation = screen.getByRole("navigation", { name: "Primary role navigation" });
      expect(within(navigation).getByRole("link", { name: activeLabel })).toHaveAttribute("aria-current", "page");
      expect(within(navigation).getAllByRole("link").filter((link) => link.hasAttribute("aria-current"))).toHaveLength(1);
      expect(screen.getByTestId("application-shell")).toHaveAttribute("data-active-role", "admin");
      expect(screen.getByText(new RegExp(heading.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")), { selector: ".auditee-root-topbar__crumbs b" })).toBeVisible();
      cleanup();
    }
  });

  it("fails every non-Admin typed workspace query before returning Admin data", async () => {
    const runtime = createMockBackendRuntime();
    const admin = requireAdminWorkspace(runtime);
    await expect(admin.listQuestions({})).resolves.toBeDefined();
    for (const role of ["inspector", "leadInspector", "manager", "gm", "finance", "executiveDirector", "auditee"] as const) {
      const capability = (runtime.backendForRole(role) as AdminBackend).adminWorkspace;
      expect(capability).toBeDefined();
      await expect(capability!.listQuestions({})).rejects.toThrow(/Admin/i);
      await expect(capability!.createQuestion({
        prompt: "This denied command must not create a record.",
        configuredReference: "Configured reference — DENIED",
        expectedEvidence: "Expected Evidence — denied",
        expectedRevision: null,
        idempotencyKey: `TASK10-DENIED-${role}`,
      })).rejects.toThrow(/Admin/i);
    }
    expect((await admin.listQuestions({ search: "This denied command" })).items).toEqual([]);
  });

  it("provides a read-only Regulatory Library with working filters and all four persistent guardrails", async () => {
    const runtime = createMockBackendRuntime();
    const capability = requireAdminWorkspace(runtime);
    const filtered = await capability.listRegulatoryReferences({ search: "NAMCARS-CAB-001", status: "ACTIVE" });
    expect(filtered.items).toEqual([expect.objectContaining({ id: "NAMCARS-CAB-001", version: "2026.1", status: "ACTIVE", effectiveDate: "2026-01-01" })]);
    expect(filtered.items[0]!.configuredRules.length).toBeGreaterThan(0);
    expect(filtered.items[0]!.changeHistory.length).toBeGreaterThan(0);
    expect(filtered.items[0]!.mappings).toEqual([
      expect.objectContaining({
        id: "RMAP-OPS-AOC-CABIN-RAMP-001",
        auditArea: "OPS",
        serviceProviderTypes: ["Air Operator (AOC)"],
        criticalElement: "CE-7",
        protocolQuestionId: "4.450",
        reviewStatus: "EXPERT_REVIEW_REQUIRED",
        sourceGap: expect.stringMatching(
          /controlled NCAA Operations surveillance\/ramp-inspection procedure/i,
        ),
        refreshPolicy: expect.objectContaining({
          sourceCollectionId: "NCAA-NAMCATS-ALL-PAGES",
          reconciliationIntervalMonths: 6,
          expertValidationIntervalMonths: 12,
          documentCount: 58,
          updateMode: "PROPOSE_DRAFT_ONLY",
        }),
        scopeRecommendation: expect.objectContaining({
          status: "ADVISORY_ONLY",
          historyState: "INSUFFICIENT_FOR_DEFERRAL",
          questionRecommendations: expect.arrayContaining([
            expect.objectContaining({
              questionId: "CAB-EMEQ-PBE-001",
              classification: "FOCUSED_FULL",
            }),
          ]),
        }),
        proposedQuestions: expect.arrayContaining([
          expect.objectContaining({ id: "CAB-EMEQ-PBE-001" }),
        ]),
      }),
    ]);
    expect(filtered.items[0]!.mappings[0]!.proposedQuestions).toHaveLength(6);
    renderAdminRoute("/admin/regulatory-library", runtime);
    const page = await screen.findByTestId("admin-regulatory-library-page");
    expect(page).toHaveTextContent("Candidate-only regulatory library");
    expect(page).toHaveTextContent("Public source baseline captured locally");
    expect(page).toHaveTextContent("Not a legal decision");
    expect(page).toHaveTextContent("No autonomous publication");
    expect(await within(page).findByText("RMAP-OPS-AOC-CABIN-RAMP-001")).toBeVisible();
    expect(page).toHaveTextContent(/PQ 4\.450 · CE-7/);
    expect(page).toHaveTextContent(/EXPERT REVIEW REQUIRED/);
    expect(page).toHaveTextContent(/Controlled-source gap/);
    expect(page).toHaveTextContent(/Event-driven review with scheduled reconciliation/);
    expect(page).toHaveTextContent(/58 public files/);
    expect(page).toHaveTextContent(/2027-01-28 · every 6 months/);
    expect(page).toHaveTextContent(/PROPOSE DRAFT ONLY/);
    expect(within(page).getAllByRole("listitem").filter((item) => item.textContent?.includes("CAB-")).length).toBeGreaterThanOrEqual(6);
    const user = userEvent.setup();
    await user.selectOptions(within(page).getByRole("combobox", { name: "Regulatory service provider" }), "Air Operator (AOC)");
    expect(await within(page).findByText("RMAP-OPS-AOC-CABIN-RAMP-001")).toBeVisible();
  });

  it("imports, inspects, edits, adopts, and submits only the exact governed current leaf", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    const governed = runtime.backendForRole("admin").adminWorkspace;
    const importSpy = vi.spyOn(governed, "importGovernedGenerationRun");
    const editSpy = vi.spyOn(governed, "createGovernedCandidateRevision");
    const submitSpy = vi.spyOn(governed, "submitGovernedCandidateReview");
    const sources = await governed.listGovernedSources({});
    expect(sources.items[0]).toEqual(expect.objectContaining({
      sourceHash: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
      clauseLocator: "Synthetic OPS/AOC 1",
      partitions: [expect.objectContaining({ role: "GENERATION_INPUT", stableRowIdentity: "CC:SYNTHETIC:OPS:AOC:1" })],
    }));
    expect(sources.items[1].partitions[0].role).toBe("BLIND_HOLDOUT");
    renderAdminComponent(<ChecklistBuilderPage />, runtime);
    const page = await screen.findByLabelText("Governed candidate editor");
    await user.click(within(page).getByRole("button", { name: "Import synthetic governed candidate" }));
    expect(importSpy).toHaveBeenCalledWith({
      operationId: "ADMIN-SYNTHETIC-GOVERNED-IMPORT",
      idempotencyKey: "ADMIN-SYNTHETIC-GOVERNED-IMPORT",
      candidateBundle: SYNTHETIC_GOVERNED_BUNDLE,
    });
    expect((await within(page).findAllByText(/GENRUN-SYNTHETIC-OPS-AOC-0001/)).length).toBeGreaterThan(0);
    expect(page).toHaveTextContent("GENREQ-SYNTHETIC-OPS-AOC-0001");
    expect(page).toHaveTextContent("regulatory-checklist-v1");
    expect(page).toHaveTextContent("deterministic-regulatory-fixture");
    expect(page).toHaveTextContent(/Synthetic OPS\/AOC 1/);
    expect(page).toHaveTextContent("Scope classification");
    expect(page).toHaveTextContent("MANDATORY CORE");
    expect(page).toHaveTextContent("Inclusion / deferral rationale");
    expect(page).toHaveTextContent("Synthetic test-profile source");
    expect(page).toHaveTextContent("Source version / hash");
    expect(page).toHaveTextContent("Page / locator / clause");
    expect(page).toHaveTextContent("Applicability");
    expect(page).toHaveTextContent("Currentness / technical review");
    expect(page).toHaveTextContent("Verification objective");
    expect(page).toHaveTextContent("Question origin");
    const originalDigest = "sha256:377598cb1bee5388b19c9d7d4de34f1ff9f6b16b7ac1d2ff6cc5d96af798ad19";
    expect(page).toHaveTextContent(originalDigest);
    const rationale = within(page).getByRole("textbox", { name: "Mapping rationale" });
    await user.clear(rationale);
    await user.type(rationale, "Synthetic test-profile rationale reviewed by an Admin without changing the controlled source claim.");
    const reason = within(page).getByRole("textbox", { name: "Change reason" });
    await user.clear(reason);
    await user.type(reason, "Apply the single controlled synthetic alternative.");
    expect(within(page).getByRole("button", { name: "Create immutable edited revision" })).toBeEnabled();
    await user.click(within(page).getByRole("button", { name: "Create immutable edited revision" }));
    expect(editSpy).toHaveBeenCalledWith(expect.objectContaining({
      candidateId: SYNTHETIC_GOVERNED_BUNDLE.candidateBundleId,
      expectedRevision: 1,
      expectedContentDigest: originalDigest,
      changeReason: "Apply the single controlled synthetic alternative.",
      mappings: [expect.objectContaining({ rationale: SYNTHETIC_EDITED_RATIONALE })],
      questions: SYNTHETIC_GOVERNED_BUNDLE.inspectionChecklist.questions,
      requiredOwners: [{
        departmentId: "FLIGHT_OPERATIONS_INSPECTORATE",
        organizationalUnitId: "FLIGHT_OPERATIONS_INSPECTORATE",
        approvalRequired: true,
      }],
    }));
    expect(await within(page).findByText(/revision 2/)).toBeVisible();
    expect(within(page).getByText("Imported run output digest").nextSibling).toHaveTextContent(originalDigest);
    const successorLine = within(page).getByText(/CAND-EDIT-.*revision 2/);
    expect(successorLine).toHaveTextContent("GENERATED_DRAFT");
    const successor = await editSpy.mock.results[0]!.value;
    expect(within(page).getByText("Current candidate content digest").nextSibling).toHaveTextContent(successor.contentDigest);
    expect(successor.contentDigest).not.toBe(originalDigest);
    expect(within(page).getByRole("button", { name: "Submit exact candidate for department review" })).toBeEnabled();
    await user.click(within(page).getByRole("button", { name: "Submit exact candidate for department review" }));
    expect(submitSpy).toHaveBeenCalledWith(expect.objectContaining({
      candidateId: successor.candidateId,
      expectedRevision: 2,
      expectedContentDigest: successor.contentDigest,
    }));
    expect(await within(page).findByText(/DEPARTMENT_REVIEW/)).toBeVisible();
    expect(page).toHaveTextContent("Submitted for department review");
    expect(within(page).queryByRole("button", { name: /approve|publish/i })).toBeNull();
  });

  it("renders literal source gaps and hybrid legacy comparisons without presenting either as authority", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    const governed = runtime.backendForRole("admin").adminWorkspace;
    const candidate = await governed.getGovernedCandidate({
      candidateId: SYNTHETIC_GOVERNED_BUNDLE.candidateBundleId,
    });
    const sourceGap = structuredClone(candidate.questions[0]!);
    sourceGap.questionId = "Q-LEGACY-SOURCE-GAP";
    sourceGap.origin = "EXISTING_CHECKLIST_CANDIDATE";
    sourceGap.citations = [];
    sourceGap.mandatoryCore = false;
    sourceGap.safetyCritical = false;
    sourceGap.scopeRecommendation = {
      ...sourceGap.scopeRecommendation,
      classification: "FOCUSED_FULL",
      guardrails: {
        ...sourceGap.scopeRecommendation.guardrails,
        mandatoryControl: false,
        safetyCritical: false,
        unknownHistory: true,
      },
    };
    sourceGap.regulatoryTrace = {
      state: "SOURCE_MAPPING_REQUIRED",
      mappingReviewState: "SOURCE_MAPPING_REQUIRED",
      currentnessState: "SOURCE_MAPPING_REQUIRED",
      technicalReviewState: "NOT_AVAILABLE",
    };
    sourceGap.reconciliation = null;

    const hybrid = structuredClone(candidate.questions[0]!);
    hybrid.questionId = "Q-HYBRID-RECONCILED";
    hybrid.origin = "HYBRID_RECONCILED";
    hybrid.mandatoryCore = false;
    hybrid.safetyCritical = false;
    hybrid.scopeRecommendation = {
      ...hybrid.scopeRecommendation,
      classification: "ROTATIONAL_SAMPLE",
      rationale: "Use the current controlled trace as a rotational sample after Department Manager technical review.",
      guardrails: {
        ...hybrid.scopeRecommendation.guardrails,
        mandatoryControl: false,
        safetyCritical: false,
        unknownHistory: false,
        sourceChanged: true,
      },
    };
    hybrid.reconciliation = {
      legacyQuestionId: "LEGACY-CABIN-001",
      legacyWording: "Historical cabin checklist wording",
      legacyOperationalIntent: "Candidate-only cabin safety observation",
      legacyResultHistory: "Unverified candidate outcome history",
      legacyExpectedEvidence: ["Historical checklist note"],
      legacyApplicability: "UNKNOWN_CANDIDATE_INPUT",
      legacyScopeClassification: "UNKNOWN_CANDIDATE_INPUT",
      currentWording: hybrid.prompt,
      currentExpectedEvidence: [...hybrid.expectedEvidence],
      currentApplicability: hybrid.regulatoryTrace.applicability!,
      currentScopeClassification: "ROTATIONAL_SAMPLE",
      wordingChanged: true,
      evidenceChanged: true,
      applicabilityChanged: true,
      scopeChanged: true,
    };
    vi.spyOn(governed, "getGovernedCandidate").mockResolvedValue({
      ...candidate,
      questions: [sourceGap, hybrid],
    });

    renderAdminComponent(<ChecklistBuilderPage />, runtime);
    const page = await screen.findByLabelText("Governed candidate editor");
    await user.click(within(page).getByRole("button", { name: "Import synthetic governed candidate" }));
    await within(page).findByText("SOURCE_MAPPING_REQUIRED");
    expect(page).toHaveTextContent("SOURCE_MAPPING_REQUIRED");
    expect(page).toHaveTextContent("cannot be validated");
    expect(page).toHaveTextContent("HYBRID RECONCILED");
    expect(page).toHaveTextContent("Legacy candidate comparison");
    expect(page).toHaveTextContent("Historical cabin checklist wording");
    expect(page).toHaveTextContent("ROTATIONAL SAMPLE");
  });

  it("repairs a candidate-only legacy source gap only by importing a separate current-source reconciliation Draft", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    renderAdminComponent(<ChecklistBuilderPage />, runtime);
    const page = await screen.findByLabelText("Governed candidate editor");

    await user.click(within(page).getByRole("button", { name: "Load candidate-only legacy source-gap Draft" }));
    expect(await within(page).findByText("SOURCE_MAPPING_REQUIRED")).toBeVisible();
    expect(page).toHaveTextContent("candidate-only legacy checklist Draft");
    expect(within(page).getByRole("button", { name: "Submit exact candidate for department review" })).toBeDisabled();
    expect(within(page).getByRole("button", { name: "Activate source change and create reconciliation Draft" })).toBeEnabled();

    await user.click(within(page).getByRole("button", { name: "Activate source change and create reconciliation Draft" }));
    expect((await within(page).findAllByText(/CAND-SYNTHETIC-HYBRID-RECONCILED-0004/)).length).toBeGreaterThan(0);
    expect(page).toHaveTextContent("Current-source activation:");
    expect(page).toHaveTextContent("Synthetic test-profile impact source");
    expect(page).toHaveTextContent("Legacy candidate comparison");
    await user.click(within(page).getByRole("button", { name: "Create immutable edited revision" }));
    expect(await within(page).findByText(/revision 2/)).toBeVisible();
    expect(within(page).getByRole("button", { name: "Submit exact candidate for department review" })).toBeEnabled();
  });

  it("does not query a candidate before import, then inspects, edits, and submits the exact current leaf", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    const governed = runtime.backendForRole("admin").adminWorkspace;
    const originalGetCandidate = governed.getGovernedCandidate.bind(governed);
    const originalImport = governed.importGovernedGenerationRun.bind(governed);
    let imported = false;
    vi.spyOn(governed, "getGovernedCandidate").mockImplementation(async (input) => {
      if (!imported) throw new Error("HTTP 404: governed candidate does not exist.");
      return originalGetCandidate(input);
    });
    vi.spyOn(governed, "importGovernedGenerationRun").mockImplementation(async (input) => {
      const result = await originalImport(input);
      imported = true;
      return result;
    });
    const editSpy = vi.spyOn(governed, "createGovernedCandidateRevision");
    const submitSpy = vi.spyOn(governed, "submitGovernedCandidateReview");

    renderAdminComponent(<ChecklistBuilderPage />, runtime);
    const page = await screen.findByLabelText("Governed candidate editor");
    expect(page).not.toHaveTextContent("HTTP 404: governed candidate does not exist.");
    expect(governed.getGovernedCandidate).not.toHaveBeenCalled();
    expect(within(page).getByRole("button", { name: "Import synthetic governed candidate" })).toBeEnabled();

    await user.click(within(page).getByRole("button", { name: "Import synthetic governed candidate" }));
    expect(await within(page).findByText(/revision 1/)).toBeVisible();
    await user.click(within(page).getByRole("button", { name: "Inspect governed generation run" }));
    const originalDigest = "sha256:377598cb1bee5388b19c9d7d4de34f1ff9f6b16b7ac1d2ff6cc5d96af798ad19";
    expect(within(page).getByText("Imported run output digest").nextSibling).toHaveTextContent(originalDigest);

    const rationale = within(page).getByRole("textbox", { name: "Mapping rationale" });
    await user.clear(rationale);
    await user.type(rationale, SYNTHETIC_EDITED_RATIONALE);
    await user.click(within(page).getByRole("button", { name: "Create immutable edited revision" }));
    const successor = await editSpy.mock.results[0]!.value;
    expect(await within(page).findByText(/revision 2/)).toBeVisible();
    expect(within(page).getByText("Imported run output digest").nextSibling).toHaveTextContent(originalDigest);
    expect(within(page).getByText("Current candidate content digest").nextSibling).toHaveTextContent(successor.contentDigest);

    await user.click(within(page).getByRole("button", { name: "Submit exact candidate for department review" }));
    expect(submitSpy).toHaveBeenCalledWith(expect.objectContaining({
      candidateId: successor.candidateId,
      expectedRevision: successor.revision,
      expectedContentDigest: successor.contentDigest,
    }));
    expect(await within(page).findByText(/DEPARTMENT_REVIEW/)).toBeVisible();
    expect(within(page).queryByRole("button", { name: /approve|publish/i })).toBeNull();
  });

  it("does not overwrite an in-progress rationale edit when the post-import reload finishes", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    const governed = runtime.backendForRole("admin").adminWorkspace;
    const originalGetCandidate = governed.getGovernedCandidate.bind(governed);
    const originalImport = governed.importGovernedGenerationRun.bind(governed);
    let delayNextCandidateRead = false;
    let importCount = 0;
    let releaseCandidateRead = () => {};
    let reportCandidateReadStarted = () => {};
    let reportCandidateReadFinished = () => {};
    const candidateReadGate = new Promise<void>((resolve) => {
      releaseCandidateRead = resolve;
    });
    const candidateReadStarted = new Promise<void>((resolve) => {
      reportCandidateReadStarted = resolve;
    });
    const candidateReadFinished = new Promise<void>((resolve) => {
      reportCandidateReadFinished = resolve;
    });
    vi.spyOn(governed, "getGovernedCandidate").mockImplementation(async (input) => {
      if (!delayNextCandidateRead) return originalGetCandidate(input);
      delayNextCandidateRead = false;
      reportCandidateReadStarted();
      await candidateReadGate;
      const result = await originalGetCandidate(input);
      reportCandidateReadFinished();
      return result;
    });
    vi.spyOn(governed, "importGovernedGenerationRun").mockImplementation(async (input) => {
      const result = await originalImport(input);
      importCount += 1;
      delayNextCandidateRead = importCount > 1;
      return result;
    });
    const editSpy = vi.spyOn(governed, "createGovernedCandidateRevision");

    renderAdminComponent(<ChecklistBuilderPage />, runtime);
    const page = await screen.findByLabelText("Governed candidate editor");
    await user.click(within(page).getByRole("button", { name: "Import synthetic governed candidate" }));
    await within(page).findByRole("textbox", { name: "Mapping rationale" });
    await user.click(within(page).getByRole("button", { name: "Import synthetic governed candidate" }));
    await candidateReadStarted;

    const rationale = within(page).getByRole("textbox", { name: "Mapping rationale" });
    await user.clear(rationale);
    await user.type(rationale, SYNTHETIC_EDITED_RATIONALE);
    await act(async () => {
      releaseCandidateRead();
      await candidateReadFinished;
    });

    expect(rationale).toHaveValue(SYNTHETIC_EDITED_RATIONALE);
    await user.click(within(page).getByRole("button", { name: "Create immutable edited revision" }));
    expect(editSpy).toHaveBeenCalledWith(expect.objectContaining({
      mappings: [expect.objectContaining({ rationale: SYNTHETIC_EDITED_RATIONALE })],
    }));
  });

  it("inspects the exact persisted safe failed generation run without a candidate", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    renderAdminComponent(<ChecklistBuilderPage />, runtime);
    const page = await screen.findByLabelText("Governed candidate editor");
    const runId = within(page).getByRole("textbox", { name: "Generation run ID" });
    await user.clear(runId);
    await user.type(runId, "GENRUN-FAILED-SYNTHETIC-INSPECTION");
    await user.click(within(page).getByRole("button", { name: "Inspect governed generation run" }));
    expect(await within(page).findByText(/GENRUN-FAILED-SYNTHETIC-INSPECTION/)).toBeVisible();
    expect(page).toHaveTextContent("GENREQ-FAILED-SYNTHETIC-INSPECTION");
    expect(page).toHaveTextContent("VALIDATION_FAILED");
    expect(page).toHaveTextContent("Exact synthetic failed-run inspection fixture");
    expect(page).not.toHaveTextContent("CAND-FAILED-SYNTHETIC");
    expect(within(page).queryByRole("button", { name: /approve|publish/i })).toBeNull();
  });

  it("renders exact field and source validation feedback for an unsupported governed edit", async () => {
    const user = userEvent.setup();
    const runtime = createMockBackendRuntime();
    renderAdminComponent(<ChecklistBuilderPage />, runtime);
    const page = await screen.findByLabelText("Governed candidate editor");
    await user.click(within(page).getByRole("button", { name: "Import synthetic governed candidate" }));
    await within(page).findByRole("textbox", { name: "Mapping rationale" });
    const rationale = within(page).getByRole("textbox", { name: "Mapping rationale" });
    await user.clear(rationale);
    await user.type(rationale, "Invent unsupported regulatory prose.");
    await user.click(within(page).getByRole("button", { name: "Create immutable edited revision" }));
    const alert = await within(page).findByRole("alert");
    expect(alert).toHaveTextContent("mappings[0].rationale");
    expect(alert).toHaveTextContent("SYNTHETIC-OPS-AOC");
    expect(alert).toHaveTextContent("CLAUSE-SYNTHETIC-OPS-AOC-1");
  });

  it("preserves the exact TPL-CABIN-2026 master to immutable CTV-CABIN-1 relationship and exact unsupported-row reasons", async () => {
    const runtime = createMockBackendRuntime();
    const capability = requireAdminWorkspace(runtime);
    const catalog = await capability.listTemplateMasters({});
    expect(catalog.items).toContainEqual(expect.objectContaining({ id: "TPL-CABIN-2026", publishedVersionId: "CTV-CABIN-1", owner: "Department Manager", itemCount: 6 }));
    renderAdminRoute("/admin/template-library", runtime);
    const page = await screen.findByTestId("admin-template-list-page");
    expect(within(page).getByRole("link", { name: /Preview CTV-CABIN-1/ })).toHaveAttribute("href", "/admin/templates");
    expect(within(page).getByRole("button", { name: /TPL-FOPS-2026 unavailable/ })).toBeDisabled();
    expect(page).toHaveTextContent(/TPL-FOPS-2026.*no declared Template Preview route/i);
  });

  it("validates and persists multiline Question Bank creation with an exact generated ID and configured-reference wording", async () => {
    const storage = localStorage;
    const runtime = createMockBackendPersistentRuntime(storage);
    renderAdminComponent(<QuestionBankPage />, runtime);
    const user = userEvent.setup();
    const page = await screen.findByTestId("admin-question-bank-page");
    const prompt = within(page).getByRole("textbox", { name: "Question text" });
    expect(prompt.tagName).toBe("TEXTAREA");
    await user.type(prompt, "   ");
    await user.click(within(page).getByRole("button", { name: "Create question" }));
    expect(await within(page).findByRole("alert")).toHaveTextContent(/Question text is required/);
    await expect(requireAdminWorkspace(runtime).createQuestion({
      prompt: "x".repeat(501),
      configuredReference: "Configured reference — TOO-LONG",
      expectedEvidence: "Expected Evidence — too long",
      expectedRevision: null,
      idempotencyKey: "TASK10-QUESTION-TOO-LONG",
    })).rejects.toThrow(/500 characters or fewer/);
    await user.clear(prompt);
    await user.type(prompt, "Is the configured cabin record available?\nConfirm the expected Evidence version.");
    await user.type(within(page).getByRole("textbox", { name: "Configured reference" }), "Configured reference — CAB-RECORD");
    await user.type(within(page).getByRole("textbox", { name: "Expected Evidence" }), "Cabin record version");
    expect(within(page).getByText(/80 characters/)).toBeVisible();
    await user.click(within(page).getByRole("button", { name: "Create question" }));
    expect(await within(page).findByText("Q-ADMIN-2026-007", { selector: ".admin-success b" })).toBeVisible();
    expect(await within(page).findByText("Q-ADMIN-2026-007", { selector: ".admin-record-card small" })).toBeVisible();
    cleanup();
    renderAdminRoute("/admin/question-bank", createMockBackendPersistentRuntime(storage));
    expect(await screen.findByText("Q-ADMIN-2026-007", { selector: ".admin-record-card small" })).toBeVisible();
    expect(screen.getByText(/Is the configured cabin record available\?[\s\S]*Confirm the expected Evidence version\./)).toBeVisible();
    expect(document.body).toHaveTextContent(/configured reference/i);
    expect(document.body).toHaveTextContent(/expected Evidence/i);
    expect(document.body).toHaveTextContent(/reference only/i);
  });

  it("keys Question Bank creation by every semantic field and clears an added Draft selection", async () => {
    const runtime = createMockBackendRuntime();
    renderAdminComponent(<QuestionBankPage />, runtime);
    const user = userEvent.setup();
    const prompt = await screen.findByRole("textbox", { name: "Question text" });
    const reference = screen.getByRole("textbox", { name: "Configured reference" });
    const evidence = screen.getByRole("textbox", { name: "Expected Evidence" });
    await user.type(prompt, "Is the exact cabin record configured?");
    await user.type(reference, "Configured reference — CAB-EXACT");
    await user.type(evidence, "Expected Evidence version one");
    await user.click(screen.getByRole("button", { name: "Create question" }));
    expect(await screen.findByText("Q-ADMIN-2026-007", { selector: ".admin-success b" })).toBeVisible();
    await user.clear(evidence);
    await user.type(evidence, "Expected Evidence version two");
    await user.click(screen.getByRole("button", { name: "Create question" }));
    expect(await screen.findByText("Q-ADMIN-2026-008", { selector: ".admin-success b" })).toBeVisible();
    cleanup();

    const capability = requireAdminWorkspace(runtime);
    const master = await capability.getTemplate({ templateId: "TPL-CABIN-2026" });
    await capability.createDraft({ templateId: master.id, expectedRevision: master.revision, idempotencyKey: "TASK10-SELECTION-DRAFT", changeReason: "Selection reset regression." });
    renderAdminComponent(<ChecklistBuilderPage />, runtime);
    const select = await screen.findByRole("combobox", { name: "Question to add" });
    expect(await screen.findByRole("heading", { name: "OPS · Air Operator (AOC) cabin/ramp pilot" })).toBeVisible();
    expect(screen.getByRole("heading", { name: "Recommended scope for the current scenario" })).toBeVisible();
    expect(screen.getByText(/INSUFFICIENT FOR DEFERRAL/)).toBeVisible();
    expect(screen.getAllByText(/FOCUSED FULL/).length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText(/ROTATIONAL SAMPLE/).length).toBeGreaterThanOrEqual(3);
    expect(screen.getAllByText(/MANDATORY CORE/).length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText(/No recorded problem is not evidence of compliance/)).toBeVisible();
    expect(screen.getByText(/6 of 6 published questions traced/i)).toBeVisible();
    expect(screen.getAllByText(/Regulatory trace · PQ 4\.450 · CE-7/).length).toBeGreaterThanOrEqual(6);
    await user.selectOptions(select, "Q-ADMIN-2026-007");
    await user.click(screen.getByRole("button", { name: /Add Q-ADMIN-2026-007/ }));
    expect(await screen.findByText("Q-ADMIN-2026-007", { selector: ".admin-builder-list small" })).toBeVisible();
    expect(select).toHaveValue("");
    expect(screen.getByRole("button", { name: /Add question to CTV-CABIN-DRAFT-2 unavailable/ })).toBeDisabled();
  });

  it("creates one revision-safe idempotent Draft while leaving CTV-CABIN-1 byte-for-byte unchanged", async () => {
    const runtime = createMockBackendRuntime();
    const capability = requireAdminWorkspace(runtime);
    const publishedBefore = JSON.stringify(await runtime.backendForRole("admin").configuration.getChecklistTemplateVersion({ templateVersionId: "CTV-CABIN-1" }));
    const master = await capability.getTemplate({ templateId: "TPL-CABIN-2026" });
    const input = { templateId: master.id, expectedRevision: master.revision, idempotencyKey: "TASK10-DRAFT-TPL-CABIN-2026-R1", changeReason: "Prepare a working checklist version." };
    const draft = await capability.createDraft(input);
    expect(draft).toMatchObject({ id: "CTV-CABIN-DRAFT-2", templateId: "TPL-CABIN-2026", version: 2, status: "DRAFT", owner: "Admin Preview", revision: 1 });
    expect(draft.questionIds).toHaveLength(6);
    expect(await capability.createDraft(input)).toEqual(draft);
    await expect(capability.createDraft({ ...input, idempotencyKey: "TASK10-STALE-DRAFT", expectedRevision: master.revision })).rejects.toThrow(/revision conflict/i);
    expect(JSON.stringify(await runtime.backendForRole("admin").configuration.getChecklistTemplateVersion({ templateVersionId: "CTV-CABIN-1" }))).toBe(publishedBefore);
  });

  it("fails closed without an available template master instead of requesting a fixture-only ID", async () => {
    const runtime = createMockBackendRuntime();
    const capability = requireAdminWorkspace(runtime);
    vi.spyOn(capability, "listTemplateMasters").mockResolvedValue({ items: [] });
    const getTemplate = vi.spyOn(capability, "getTemplate");

    renderAdminComponent(<ChecklistBuilderPage />, runtime);

    const page = await screen.findByTestId("admin-checklist-builder-page");
    expect(await within(page).findByText(/No checklist template master is available/i)).toBeVisible();
    expect(within(page).queryByRole("alert")).not.toBeInTheDocument();
    expect(getTemplate).not.toHaveBeenCalled();
    expect(within(page).getByRole("button", { name: /Create working Draft unavailable/i })).toBeDisabled();
  });

  it("shows a truthful empty history when the route template identity is absent", async () => {
    const runtime = createMockBackendRuntime();
    const capability = requireAdminWorkspace(runtime);
    vi.spyOn(capability, "listTemplateMasters").mockResolvedValue({ items: [] });
    const getTemplate = vi.spyOn(capability, "getTemplate");

    renderAdminRoute("/admin/templates/TPL-CABIN-2026/history", runtime);

    const page = await screen.findByTestId("admin-version-history-page");
    expect(await within(page).findByText(/No version history is available for TPL-CABIN-2026/i)).toBeVisible();
    expect(within(page).queryByRole("alert")).not.toBeInTheDocument();
    expect(getTemplate).not.toHaveBeenCalled();
    expect(within(page).getByRole("button", { name: /Compare versions unavailable/i })).toBeDisabled();
  });

  it("adds and reorders exact Draft questions without mutating a published array and records append-only history", async () => {
    const runtime = createMockBackendRuntime();
    const capability = requireAdminWorkspace(runtime);
    const question = await capability.createQuestion({ prompt: "Is the demo cabin reference recorded?", configuredReference: "Configured reference — CAB-DEMO", expectedEvidence: "Expected Evidence — cabin record", expectedRevision: null, idempotencyKey: "TASK10-Q-ADD" });
    const master = await capability.getTemplate({ templateId: "TPL-CABIN-2026" });
    const draft = await capability.createDraft({ templateId: master.id, expectedRevision: master.revision, idempotencyKey: "TASK10-DRAFT-ADD", changeReason: "Add an exact question." });
    const added = await capability.addDraftQuestion({ templateId: master.id, draftVersionId: draft.id, questionId: question.id, expectedRevision: draft.revision, idempotencyKey: "TASK10-ADD-Q" });
    expect(added.questionIds.at(-1)).toBe(question.id);
    const moved = await capability.moveDraftQuestion({ templateId: master.id, draftVersionId: draft.id, questionId: question.id, direction: "UP", expectedRevision: added.revision, idempotencyKey: "TASK10-MOVE-Q" });
    expect(moved.questionIds.at(-2)).toBe(question.id);
    const history = await capability.getTemplate({ templateId: master.id });
    expect(history.versions.map((version) => version.id)).toEqual(["CTV-CABIN-1", "CTV-CABIN-DRAFT-2"]);
    expect(history.versions[0]!.questionIds).not.toContain(question.id);
  });

  it("persists exact Draft mutations and idempotent operations across remount without changing CTV-CABIN-1", async () => {
    const storage = localStorage;
    const first = createMockBackendPersistentRuntime(storage);
    const capability = requireAdminWorkspace(first);
    const publishedBefore = JSON.stringify((await capability.getTemplate({ templateId: "TPL-CABIN-2026" })).versions[0]);
    const question = await capability.createQuestion({
      prompt: "Is the persisted exact reference visible after remount?",
      configuredReference: "Configured reference — PERSISTED-EXACT",
      expectedEvidence: "Expected Evidence — persisted exact record",
      expectedRevision: null,
      idempotencyKey: "TASK10-PERSIST-Q",
    });
    const master = await capability.getTemplate({ templateId: "TPL-CABIN-2026" });
    const draft = await capability.createDraft({
      templateId: master.id,
      expectedRevision: master.revision,
      idempotencyKey: "TASK10-PERSIST-DRAFT",
      changeReason: "Prove exact Draft persistence and immutable history.",
    });
    const addInput = {
      templateId: master.id,
      draftVersionId: draft.id,
      questionId: question.id,
      expectedRevision: draft.revision,
      idempotencyKey: "TASK10-PERSIST-ADD",
    };
    const added = await capability.addDraftQuestion(addInput);
    expect(await capability.addDraftQuestion(addInput)).toEqual(added);
    const moveInput = {
      templateId: master.id,
      draftVersionId: draft.id,
      questionId: question.id,
      direction: "UP" as const,
      expectedRevision: added.revision,
      idempotencyKey: "TASK10-PERSIST-MOVE",
    };
    const moved = await capability.moveDraftQuestion(moveInput);
    expect(await capability.moveDraftQuestion(moveInput)).toEqual(moved);
    await expect(capability.moveDraftQuestion({ ...moveInput, direction: "DOWN" })).rejects.toThrow(/operation id|idempotency/i);

    const second = createMockBackendPersistentRuntime(storage);
    const remounted = requireAdminWorkspace(second);
    const template = await remounted.getTemplate({ templateId: "TPL-CABIN-2026" });
    expect(JSON.stringify(template.versions[0])).toBe(publishedBefore);
    expect(template.versions.map((version) => version.id)).toEqual(["CTV-CABIN-1", "CTV-CABIN-DRAFT-2"]);
    expect(template.versions[1]!.questionIds.at(-2)).toBe(question.id);
    const adminEvents = (await remounted.listAuditEvents({ actor: "USR-ADMIN-ADA" })).items
      .filter((event) => event.action.startsWith("admin."));
    expect(adminEvents.map((event) => event.action)).toEqual([
      "admin.question_created",
      "admin.template_draft_created",
      "admin.template_question_added",
      "admin.template_question_reordered",
    ]);
    expect(new Set(adminEvents.map((event) => event.eventId)).size).toBe(4);
  });

  it("shows append-only version identity, ownership, exact diff, and disabled Department Manager publish authority", async () => {
    const runtime = createMockBackendRuntime();
    const capability = requireAdminWorkspace(runtime);
    const master = await capability.getTemplate({ templateId: "TPL-CABIN-2026" });
    await capability.createDraft({ templateId: master.id, expectedRevision: master.revision, idempotencyKey: "TASK10-HISTORY-DRAFT", changeReason: "Prepare Admin configuration changes." });
    renderAdminRoute("/admin/templates/TPL-CABIN-2026/history", runtime);
    const page = await screen.findByTestId("admin-version-history-page");
    expect(await within(page).findByText("CTV-CABIN-1")).toBeVisible();
    expect(await within(page).findByText("CTV-CABIN-DRAFT-2")).toBeVisible();
    expect(page).toHaveTextContent("Admin Preview");
    expect(page).toHaveTextContent("Department Manager");
    expect(page).toHaveTextContent(/0 questions added, 0 removed, order unchanged/i);
    const publish = within(page).getByRole("button", { name: /Publish CTV-CABIN-DRAFT-2 unavailable/ });
    expect(publish).toBeDisabled();
    expect(page).toHaveTextContent(/Department Manager owns publishing after approval/i);
  });

  it("projects the exact immutable Admin package without crossing into Inspector execution", async () => {
    const runtime = createMockBackendRuntime();
    const capability = requireAdminWorkspace(runtime);
    const projection = await capability.getInspectionPackage({ packageId: "PKG-CAB-2026-001" });
    expect(projection).toMatchObject({ id: "PKG-CAB-2026-001", auditId: "AUD-2026-001", organizationId: "ORG-FLY-NAMIBIA" });
    expect(projection.questionIds).toContain("CAB-EMEQ-PBE-001");
    expect(projection.configuredReferences.length).toBeGreaterThan(0);
    expect(projection.expectedEvidence.length).toBeGreaterThan(0);
    expect(projection.riskFocus.length).toBeGreaterThan(0);
    renderAdminRoute("/admin/inspection-package-builder", runtime);
    const page = await screen.findByTestId("admin-inspection-package-page");
    expect(await within(page).findByRole("button", { name: /Run PKG-CAB-2026-001 unavailable/ })).toBeDisabled();
    expect(page).toHaveTextContent(/Admin configuration preview.*not Inspector execution/i);
  });

  it("keeps Reports local-only, Users exact-scoped, and provider provisioning visibly disabled", async () => {
    const runtime = createMockBackendRuntime();
    const capability = requireAdminWorkspace(runtime);
    expect((await capability.listReportDefinitions({ search: "package" })).items).toEqual([expect.objectContaining({ id: "ADMIN-RPT-PACKAGE-001", packageFields: expect.arrayContaining(["packageId", "auditId", "organizationId"]) })]);
    const directory = await capability.listAccessDirectory({ search: "USR-AUDITEE-FLY", role: "auditee" });
    expect(directory.items).toEqual([expect.objectContaining({ subjectId: "USR-AUDITEE-FLY", roles: ["auditee"], organizationId: "ORG-FLY-NAMIBIA", email: "Not configured in demo", mfaState: "Not configured in demo", invitationState: "Not configured in demo", accountStatus: "Not configured in demo" })]);
    renderAdminRoute("/admin/users-roles", runtime);
    const page = await screen.findByTestId("admin-users-roles-page");
    const disabled = within(page).getAllByRole("button", { name: /unavailable/ });
    expect(disabled.length).toBeGreaterThan(0);
    expect(disabled.every((button) => button.hasAttribute("disabled"))).toBe(true);
    expect(page).toHaveTextContent(/configured identity-provider administration/);
  });

  it("separates demo Configurations from production integrations and keeps advisory-only lifecycle language", async () => {
    renderAdminRoute("/admin/configurations");
    const page = await screen.findByTestId("admin-configurations-page");
    expect(page).toHaveTextContent("Level 1 Critical");
    expect(page).toHaveTextContent("Due Date");
    expect(page).toHaveTextContent(/Oversight Health Index.*advisory/i);
    expect(page).toHaveTextContent(/in-app reminder/i);
    expect(page).toHaveTextContent(/No real email or SMS/i);
    expect(within(page).queryByRole("button", { name: /save/i })).toBeNull();
  });

  it("preserves exact organization list/detail identity and migrates an older persistent envelope without losing prior profile state", async () => {
    const storage = localStorage;
    const first = createMockBackendPersistentRuntime(storage);
    const profile = await first.backendForRole("inspector").profiles.updateMine({ displayName: "Persisted Inspector", expectedRevision: 1, idempotencyKey: "TASK10-PROFILE-PERSIST" });
    expect(profile.displayName).toBe("Persisted Inspector");
    const envelope = JSON.parse(storage.getItem(DEMO_MOCK_STORAGE_KEY)!) as { schemaVersion: number; state: Partial<MockState> & Record<string, unknown>; operations: unknown[] };
    envelope.schemaVersion = 2;
    delete envelope.state.adminWorkspace;
    storage.setItem(DEMO_MOCK_STORAGE_KEY, JSON.stringify(envelope));
    const runtime = createMockBackendPersistentRuntime(storage);
    expect((await runtime.backendForRole("inspector").profiles.getMine({})).displayName).toBe("Persisted Inspector");
    const capability = requireAdminWorkspace(runtime);
    const organizations = await capability.listOrganizations({ status: "ACTIVE", scope: "CAA oversight" });
    expect(organizations.items.map((organization) => organization.id)).toEqual(["ORG-FLY-NAMIBIA", "ORG-SKYCARGO"]);
    expect(organizations.items.find((organization) => organization.id === "ORG-SKYCARGO")).toMatchObject({ detailAvailable: false, disabledReason: expect.stringMatching(/ORG-SKYCARGO.*no declared contextual detail route/i) });
    expect(await capability.getOrganization({ organizationId: "ORG-FLY-NAMIBIA" })).toMatchObject({ id: "ORG-FLY-NAMIBIA", legalName: "Fly Namibia" });
    await expect(capability.getOrganization({ organizationId: "ORG-SKYCARGO" })).rejects.toThrow(/ORG-SKYCARGO/);
  });

  it("filters an append-only demo trace by exact actor/action/entity/system/date text without deleting identity", async () => {
    const runtime = createMockBackendRuntime();
    const capability = requireAdminWorkspace(runtime);
    const before = await capability.listAuditEvents({});
    const exact = await capability.listAuditEvents({ actor: "USR-MANAGER-NORA", action: "report.decision_recorded", entity: "PR-2026-018-V0", system: "MANUAL", dateText: "2026-06-15" });
    expect(exact.items).toEqual([expect.objectContaining({ eventId: "AUDIT-REPORT-SEED-0001", actorSubjectId: "USR-MANAGER-NORA", actorRole: "manager", action: "report.decision_recorded", entityId: "PR-2026-018-V0", beforeStatus: "DEPARTMENT_REVIEW", afterStatus: "RETURNED", reason: "Clarify Finding basis and supporting Evidence.", entityRevision: 1, occurredAt: "2026-06-15T09:00:00.000Z" })]);
    expect((await capability.listAuditEvents({})).items).toEqual(before.items);
    expect((await capability.listAuditEvents({ system: "SYSTEM" })).items).toEqual([
      expect.objectContaining({ eventId: "AUDIT-SYSTEM-SEED-0001", actorRole: null, actorSubjectId: null }),
    ]);
    renderAdminRoute("/admin/audit-log", runtime);
    const page = await screen.findByTestId("admin-audit-log-page");
    expect(page).toHaveTextContent(/demo trace/i);
    expect(page).toHaveTextContent(/not a production audit trail/i);
  });
});
