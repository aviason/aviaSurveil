// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import type { AGADemoWorkspaceBackend, AGADemoWorkspaceLifecycleProjection } from "../../backend/aga-demo-workspace";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { ScenarioProvider } from "../../app/scenario-context";
import { AGADemoPotentialFindingPage } from "./aga-demo-potential-finding-page";

afterEach(() => cleanup());

const capability = { available: true, projection: "LEAD_SCOPED", classificationEnabled: false, recommendationEnabled: false, lifecycleEnabled: true, resetEnabled: false };

function projection(): AGADemoWorkspaceLifecycleProjection {
  return {
    inspectionId: "inspection-test-2",
    generationId: "generation-test-2",
    organizationId: "organization-test-2",
    providerScopeId: "scope-test-2",
    state: "SUBMITTED",
    revision: 4,
    questions: [],
    responses: [],
    potentialFindings: [{ potentialFindingId: "potential-test-1", rootId: "root-test-1", version: 1, inspectionId: "inspection-test-2", questionKey: "base\\u001fquestion-test-2", responseId: "response-test-1", responseRevision: 1, responseDigest: "sha256:" + "2".repeat(64), answer: "NON_COMPLIANT", commentToAuditee: "Follow-up is required.", state: "PENDING_LEAD_REVIEW", createdAt: "2026-08-04T13:00:00Z", digest: "sha256:" + "3".repeat(64) }],
    findings: [],
    capRevisions: [],
    evidenceVersions: [],
    verificationDecisions: [],
    currentOwnerRole: "Lead Inspector",
    nextAction: "Review Potential Findings",
    updatedAt: "2026-08-04T13:00:00Z",
    digest: "sha256:" + "4".repeat(64),
  } as AGADemoWorkspaceLifecycleProjection;
}

function renderPage(client: AGADemoWorkspaceBackend, initial = projection()) {
  const runtime = createMockBackendRuntime();
  render(
    <AppProviders runtime={{ ...runtime, backend: { ...runtime.backend, agaDemoWorkspace: client }, buildProfile: "http", environmentLabel: "test", subjectId: "lead-test-subject" }}>
      <ScenarioProvider><MemoryRouter><AGADemoPotentialFindingPage capability={capability} role="leadInspector" roleLabel="Lead Inspector" initialProjection={initial} /></MemoryRouter></ScenarioProvider>
    </AppProviders>,
  );
}

function lifecycleClient(current: AGADemoWorkspaceLifecycleProjection): AGADemoWorkspaceBackend {
  return {
    capability: vi.fn().mockResolvedValue(capability),
    classificationQuery: vi.fn(),
    classificationCommand: vi.fn(),
    recommendationCommand: vi.fn(),
    lifecycleQuery: vi.fn(),
    lifecycleCommand: vi.fn().mockResolvedValue({ operationId: "CONVERT_POTENTIAL_FINDING", replayed: false, lifecycleAvailable: true, lifecycle: current }),
    adminCommand: vi.fn(),
  };
}

describe("AGA synthetic Potential Finding page", () => {
  it("converts a server-returned Potential Finding only after explicit Lead choices", async () => {
    const client = lifecycleClient(projection());
    renderPage(client);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Convert to Finding" }));
    await waitFor(() => expect(client.lifecycleCommand).toHaveBeenCalledWith(expect.objectContaining({ operationId: "CONVERT_POTENTIAL_FINDING", potentialFindingId: "potential-test-1", reasonCode: "LEAD_REVIEW_DECISION", severity: "MAJOR", capRequired: true, evidenceRequired: true, dueDateRequired: false })));
  });

  it("keeps conversion unavailable when Lead review has no selected server object", () => {
    const current = projection();
    current.potentialFindings = [];
    const client = lifecycleClient(current);
    renderPage(client, current);
    expect(screen.getByRole("button", { name: "Convert to Finding" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Convert to Finding" })).toHaveAccessibleDescription(/Choose a server-returned Potential Finding/i);
  });

  it("exposes separate Lead return and dismiss transitions", async () => {
    const client = lifecycleClient(projection());
    renderPage(client);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Return for correction" }));
    await waitFor(() => expect(client.lifecycleCommand).toHaveBeenCalledWith(expect.objectContaining({ operationId: "RETURN_POTENTIAL_FINDING", potentialFindingId: "potential-test-1", reasonCode: "LEAD_REVIEW_DECISION" })));

    vi.mocked(client.lifecycleCommand).mockClear();
    await user.click(screen.getByRole("button", { name: "Dismiss Potential Finding" }));
    await waitFor(() => expect(client.lifecycleCommand).toHaveBeenCalledWith(expect.objectContaining({ operationId: "DISMISS_POTENTIAL_FINDING", potentialFindingId: "potential-test-1", reasonCode: "LEAD_REVIEW_DECISION" })));
  });
});
