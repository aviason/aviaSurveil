// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import type {
  AGADemoWorkspaceBackend,
  AGADemoWorkspaceCommand,
  AGADemoWorkspaceLifecycleAuditeeProjection,
  AGADemoWorkspaceLifecycleCAAProjection,
  AGADemoWorkspaceLifecycleProjection,
} from "../../backend/aga-demo-workspace";
import { BackendHttpError } from "../../backend/http-backend";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { ScenarioProvider } from "../../app/scenario-context";
import { AGADemoInspectionPage } from "./aga-demo-inspection-page";

afterEach(() => cleanup());

const capability = { available: true, projection: "INSPECTOR_SCOPED", classificationEnabled: false, recommendationEnabled: false, lifecycleEnabled: true, resetEnabled: false };
const questionKey = "base\u001fquestion-test-1";

function projection(state: AGADemoWorkspaceLifecycleProjection["state"] = "IN_PROGRESS"): AGADemoWorkspaceLifecycleProjection {
  return {
    inspectionId: "inspection-test-1",
    generationId: "generation-test-1",
    organizationId: "organization-test-1",
    providerScopeId: "scope-test-1",
    state,
    revision: 3,
    questions: [{ questionKey, questionRef: { origin: "BASE", rootSequence: 1 }, rootSequence: 1, projection: { mainDomainCode: "DOMAIN_TEST" } }],
    responses: [],
    potentialFindings: [],
    findings: [],
    capRevisions: [],
    evidenceVersions: [],
    verificationDecisions: [],
    currentOwnerRole: "Assigned Inspector",
    nextAction: "Record checklist response",
    updatedAt: "2026-08-04T13:00:00Z",
    digest: "sha256:" + "1".repeat(64),
  } as AGADemoWorkspaceLifecycleProjection;
}

function renderPage(
  client: AGADemoWorkspaceBackend,
  role: "inspector" | "leadInspector" | "manager" | "auditee" = "inspector",
  initial: AGADemoWorkspaceLifecycleProjection | AGADemoWorkspaceLifecycleCAAProjection | AGADemoWorkspaceLifecycleAuditeeProjection | null = projection(),
  capabilityFor = capability,
) {
  const runtime = createMockBackendRuntime();
  render(
    <AppProviders runtime={{ ...runtime, backend: { ...runtime.backend, agaDemoWorkspace: client }, buildProfile: "http", environmentLabel: "test", subjectId: `subject-${role}` }}>
      <ScenarioProvider><MemoryRouter><AGADemoInspectionPage capability={capabilityFor} role={role} roleLabel={role === "inspector" ? "CAA Inspector" : role === "leadInspector" ? "Lead Inspector" : role === "manager" ? "Department Manager" : "Auditee — Fly Namibia"} initialProjection={initial ?? undefined} /></MemoryRouter></ScenarioProvider>
    </AppProviders>,
  );
}

function caaProjection(): AGADemoWorkspaceLifecycleCAAProjection {
  return {
    ...projection(),
    recommendationId: "recommendation-test-1",
    recommendationDigest: `sha256:${"8".repeat(64)}`,
    inspector: { bindingId: "binding-inspector-1", bindingRevision: 1 },
    lead: { bindingId: "binding-lead-1", bindingRevision: 1 },
    auditee: { bindingId: "binding-auditee-1", bindingRevision: 1 },
    roleHistory: [{ role: "INSPECTOR", action: "PINNED", occurredAt: "2026-08-04T13:00:00Z" }],
  } as AGADemoWorkspaceLifecycleCAAProjection;
}

function auditeeProjection(): AGADemoWorkspaceLifecycleAuditeeProjection {
  return {
    inspectionId: "inspection-test-1",
    generationId: "generation-test-1",
    organizationId: "organization-test-1",
    state: "IN_PROGRESS",
    revision: 3,
    findings: [],
    capRevisions: [],
    evidenceVersions: [],
    verificationDecisions: [],
    nextAction: "Submit CAP revision",
    publicOwnerLabel: "Fly Namibia · Service Provider",
    updatedAt: "2026-08-04T13:00:00Z",
    digest: "sha256:" + "1".repeat(64),
  } as AGADemoWorkspaceLifecycleAuditeeProjection;
}

function lifecycleClient(current: AGADemoWorkspaceLifecycleProjection | AGADemoWorkspaceLifecycleCAAProjection | AGADemoWorkspaceLifecycleAuditeeProjection): AGADemoWorkspaceBackend {
  let latest = current;

  return {
    capability: vi.fn().mockResolvedValue(capability),
    classificationQuery: vi.fn(),
    classificationCommand: vi.fn(),
    recommendationCommand: vi.fn(),
    lifecycleQuery: vi.fn(),
    lifecycleCommand: vi.fn().mockImplementation(async (command: AGADemoWorkspaceCommand) => {
      if (command.operationId === "RECORD_RESPONSE" && !("publicOwnerLabel" in latest)) {
        latest = {
          ...latest,
          responses: [
            {
              responseId: "response-test-1",
              questionKey,
              revision: 1,
              answer: "OBSERVATION",
              commentToAuditee: "Synthetic response requires follow-up.",
              createdAt: "2026-08-04T10:00:00Z",
              responseDigest: `sha256:${"7".repeat(64)}`,
            },
          ],
        };
      }

      if ("publicOwnerLabel" in latest) {
        return {
          operationId: command.operationId,
          replayed: false,
          lifecycleAvailable: true,
          lifecycleAuditee: latest,
        };
      }
      return {
        operationId: command.operationId,
        replayed: false,
        lifecycleAvailable: true,
        lifecycle: latest,
      };
    }),
    adminCommand: vi.fn(),
  };
}

describe("AGA synthetic inspection lifecycle page", () => {
  it("records an exact answer and proposes only an eligible Potential Finding", async () => {
    const client = lifecycleClient(projection());
    renderPage(client);
    const user = userEvent.setup();
    await user.click(screen.getByRole("radio", { name: `${questionKey} OBSERVATION` }));
    await user.type(screen.getByRole("textbox", { name: "Comment to Auditee" }), "Synthetic response requires follow-up.");
    await user.click(screen.getByRole("button", { name: "Record response" }));
    await waitFor(() => expect(client.lifecycleCommand).toHaveBeenCalledWith(expect.objectContaining({ operationId: "RECORD_RESPONSE", inspectionId: "inspection-test-1", targetQuestionKey: questionKey, answer: "OBSERVATION", commentToAuditee: "Synthetic response requires follow-up." })));
    await user.click(screen.getByRole("button", { name: "Propose Potential Finding" }));
    await waitFor(() => expect(client.lifecycleCommand).toHaveBeenCalledWith(expect.objectContaining({ operationId: "CREATE_POTENTIAL_FINDING", targetQuestionKey: questionKey, answer: "OBSERVATION" })));
  });

  it("keeps lifecycle controls disabled with a precise reason when no server object is present", () => {
    const client = lifecycleClient(projection());
    renderPage(client, "inspector", null);
    expect(screen.getByRole("status")).toHaveTextContent(/No server-returned synthetic inspection/i);
    expect(screen.getByRole("button", { name: "Load authorized inspection" })).toBeDisabled();
  });

  it("renders an empty lifecycle state instead of leaking an expected 404", async () => {
    const client = lifecycleClient(projection());
    vi.mocked(client.lifecycleQuery).mockRejectedValue(new BackendHttpError("Backend request failed with status 404", 404, null, null, null));
    renderPage(client, "inspector", null);

    await waitFor(() => expect(client.lifecycleQuery).toHaveBeenCalledWith(expect.objectContaining({ operationId: "GET_CURRENT_INSPECTION" }), expect.anything()));
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent(/No server-returned synthetic inspection/i));
    expect(screen.queryByText(/Backend request failed with status 404/i)).not.toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("keeps Manager recommendation and simulation release fail-closed until pinned facts arrive", () => {
    const client = lifecycleClient(projection());
    renderPage(client, "manager", null, { ...capability, recommendationEnabled: true });
    expect(screen.getByRole("link", { name: "Open package builder" })).toHaveAttribute("href", "/department-manager/aga-demo-workspace/inspection-package");
    expect(screen.getByRole("button", { name: "Create AGA recommendation" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Create AGA recommendation" })).toHaveAccessibleDescription(/server-returned provider scope.*typed target/i);
    expect(screen.getByRole("button", { name: "Release synthetic simulation" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Release synthetic simulation" })).toHaveAccessibleDescription(/immutable server recommendation snapshot/i);
  });

  it("keeps response controls role-bound and supports keyboard answer selection on a narrow viewport", async () => {
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 390 });
    const client = lifecycleClient(projection());
    renderPage(client, "leadInspector");
    expect(screen.getByRole("radio", { name: `${questionKey} OBSERVATION` })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Record response" })).toHaveAccessibleDescription(/authenticated role is not its assigned lifecycle actor/i);

    cleanup();
    renderPage(lifecycleClient(projection()));
    const user = userEvent.setup();
    const compliant = screen.getByRole("radio", { name: `${questionKey} COMPLIANT` });
    compliant.focus();
    await user.keyboard(" ");
    expect(compliant).toBeChecked();
    expect(screen.getByRole("button", { name: "Record response" })).toBeEnabled();
  });

  it("requires and sends a bounded reopen explanation", async () => {
    const client = lifecycleClient(projection("SUBMITTED"));
    renderPage(client, "inspector", projection("SUBMITTED"));
    const user = userEvent.setup();
    const reopen = screen.getByRole("button", { name: "Reopen checklist" });
    expect(reopen).toBeDisabled();
    expect(reopen).toHaveAccessibleDescription(/bounded reopen explanation/i);
    await user.type(screen.getByRole("textbox", { name: "Reopen explanation" }), "The checklist needs a focused correction.");
    await waitFor(() => expect(reopen).toBeEnabled());
    await user.click(reopen);
    await waitFor(() => expect(client.lifecycleCommand).toHaveBeenCalledWith(expect.objectContaining({ operationId: "REOPEN_CHECKLIST", reasonCode: "REOPEN_FOR_REVIEW", reasonExplanation: "The checklist needs a focused correction." })));
  });

  it("shows CAA-only role history while the Auditee receives only its public owner label", () => {
    renderPage(lifecycleClient(caaProjection()), "leadInspector", caaProjection());
    expect(screen.getByRole("region", { name: "CAA role history" })).toHaveTextContent("INSPECTOR");
    expect(screen.getByRole("region", { name: "CAA role history" })).toHaveTextContent("PINNED");

    cleanup();
    const publicProjection = auditeeProjection();
    renderPage(lifecycleClient(publicProjection), "auditee", publicProjection);
    expect(screen.getByRole("region", { name: "Auditee public owner" })).toHaveTextContent("Fly Namibia · Service Provider");
    expect(screen.queryByRole("region", { name: "CAA role history" })).not.toBeInTheDocument();
    expect(screen.queryByText("binding-lead-1")).not.toBeInTheDocument();
  });

  it("clears a retained in-memory projection on BFCache restoration", async () => {
    const client = lifecycleClient(projection());
    renderPage(client);
    expect(screen.queryByRole("button", { name: "Load authorized inspection" })).not.toBeInTheDocument();
    const event = new Event("pageshow");
    Object.defineProperty(event, "persisted", { value: true });
    window.dispatchEvent(event);
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent(/restored page was cleared/i));
  });
});
