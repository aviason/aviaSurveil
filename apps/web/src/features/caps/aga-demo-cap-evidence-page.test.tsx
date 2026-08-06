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
import { AGADemoCAPEvidencePage } from "./aga-demo-cap-evidence-page";

afterEach(() => cleanup());

const capability = { available: true, projection: "AUDITEE_SCOPED", classificationEnabled: false, recommendationEnabled: false, lifecycleEnabled: true, resetEnabled: false };

function projection(): AGADemoWorkspaceLifecycleProjection {
  return {
    inspectionId: "inspection-test-3",
    generationId: "generation-test-3",
    organizationId: "organization-test-3",
    providerScopeId: "scope-test-3",
    state: "SUBMITTED",
    revision: 5,
    questions: [],
    responses: [],
    potentialFindings: [],
    findings: [{ findingId: "finding-test-1", potentialFindingRootId: "root-test-1", inspectionId: "inspection-test-3", questionKey: "base\\u001fquestion-test-3", severity: "MAJOR", state: "WAITING_FOR_CAP", nextAction: "SUBMIT_CAP_REVISION", capRequired: true, evidenceRequired: true, dueDateRequired: false, revision: 1, createdAt: "2026-08-04T13:00:00Z", digest: "sha256:" + "5".repeat(64) }],
    capRevisions: [],
    evidenceVersions: [],
    verificationDecisions: [],
    currentOwnerRole: "Service Provider",
    nextAction: "Submit CAP revision",
    updatedAt: "2026-08-04T13:00:00Z",
    digest: "sha256:" + "6".repeat(64),
  } as AGADemoWorkspaceLifecycleProjection;
}

function renderPage(client: AGADemoWorkspaceBackend, role: "auditee" | "manager" = "auditee", initial = projection()) {
  const runtime = createMockBackendRuntime();
  render(
    <AppProviders runtime={{ ...runtime, backend: { ...runtime.backend, agaDemoWorkspace: client }, buildProfile: "http", environmentLabel: "test", subjectId: `${role}-test-subject` }}>
      <ScenarioProvider><MemoryRouter><AGADemoCAPEvidencePage capability={capability} role={role} roleLabel={role === "auditee" ? "Auditee — Fly Namibia" : "Department Manager"} initialProjection={initial} /></MemoryRouter></ScenarioProvider>
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
    lifecycleCommand: vi.fn().mockResolvedValue({ operationId: "SUBMIT_CAP_REVISION", replayed: false, lifecycleAvailable: true, lifecycle: current }),
    adminCommand: vi.fn(),
  };
}

describe("AGA synthetic CAP and Evidence page", () => {
	it("uses the final append-only CAP state when one revision records submission and pending review", async () => {
		const current = projection();
		current.findings[0] = { ...current.findings[0]!, state: "PENDING_CAA_REVIEW", nextAction: "REVIEW_CAP" };
		current.capRevisions = [
			{ capId: "cap-test-1", findingId: "finding-test-1", revision: 1, state: "SUBMITTED", rootCause: "root", correctiveAction: "correct", preventiveAction: "prevent", responsiblePerson: "owner", createdAt: "2026-08-04T13:00:00Z", digest: "sha256:" + "7".repeat(64) },
			{ capId: "cap-test-1", findingId: "finding-test-1", revision: 1, state: "PENDING_CAA_REVIEW", rootCause: "root", correctiveAction: "correct", preventiveAction: "prevent", responsiblePerson: "owner", createdAt: "2026-08-04T13:00:01Z", digest: "sha256:" + "8".repeat(64) },
		];
		const client = lifecycleClient(current);
		renderPage(client, "manager", current);
		const user = userEvent.setup();
		await user.type(screen.getByRole("textbox", { name: "Lifecycle Comment to Auditee" }), "CAP review comment");
		await user.type(screen.getByRole("textbox", { name: "Internal CAA Note" }), "Private review note");
		await waitFor(() => expect(screen.getByRole("button", { name: "Review CAP" })).toBeEnabled());
	});

	it("submits CAP metadata as the matching Auditee without exposing Internal CAA Notes", async () => {
    const client = lifecycleClient(projection());
    renderPage(client);
    const user = userEvent.setup();
    await user.type(screen.getByRole("textbox", { name: "CAP root cause" }), "Synthetic root cause.");
    await user.type(screen.getByRole("textbox", { name: "CAP corrective action" }), "Synthetic corrective action.");
    await user.type(screen.getByRole("textbox", { name: "CAP preventive action" }), "Synthetic preventive action.");
    await user.type(screen.getByRole("textbox", { name: "CAP responsible person" }), "Synthetic owner.");
    await user.type(screen.getByRole("textbox", { name: "Lifecycle Comment to Auditee" }), "CAP submitted for review.");
    expect(screen.queryByRole("textbox", { name: "Internal CAA Note" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Review CAP" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Verify Evidence" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Authorize Finding closure" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Append-only CAP and Evidence history" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Submit CAP revision" }));
    await waitFor(() => expect(client.lifecycleCommand).toHaveBeenCalledWith(expect.objectContaining({ operationId: "SUBMIT_CAP_REVISION", findingId: "finding-test-1", rootCause: "Synthetic root cause.", commentToAuditee: "CAP submitted for review." })));
  });

  it("keeps authorized closure disabled until the server projection is PENDING_CLOSURE", () => {
    const client = lifecycleClient(projection());
    renderPage(client, "manager");
    expect(screen.getByRole("button", { name: "Authorize Finding closure" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Authorize Finding closure" })).toHaveAccessibleDescription(/PENDING_CLOSURE/i);
  });

  it("keeps CAP acceptance separate from authorized closure", async () => {
    const current = projection();
    current.state = "COMPLETED";
    current.findings[0]!.state = "PENDING_CLOSURE";
    current.nextAction = "Authorize Finding closure";
    const client = lifecycleClient(current);
    renderPage(client, "manager", current);
    const user = userEvent.setup();
    const closeButton = screen.getByRole("button", { name: "Authorize Finding closure" });
    expect(closeButton).toBeDisabled();
    await user.type(screen.getByRole("textbox", { name: "Authorized closure explanation" }), "The Manager reviewed the synthetic closure basis.");
    await waitFor(() => expect(closeButton).toBeEnabled());
    await user.click(closeButton);
    await waitFor(() => expect(client.lifecycleCommand).toHaveBeenCalledWith(expect.objectContaining({ operationId: "AUTHORIZED_CLOSE", findingId: "finding-test-1", reasonCode: "MANAGER_AUTHORIZED_CLOSURE", reasonExplanation: "The Manager reviewed the synthetic closure basis." })));
    expect(client.lifecycleCommand).not.toHaveBeenCalledWith(expect.objectContaining({ operationId: "VERIFY_EVIDENCE" }));
  });
});
