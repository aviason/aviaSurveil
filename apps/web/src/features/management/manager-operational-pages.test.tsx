// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { REACT_ROUTE_CONTRACT_BY_ID } from "../../app/route-contracts";
import { AppRouter } from "../../app/router";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { seedVisualRuntimeForPath } from "../../mock/seed-visual-runtime";

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

function renderManagerRoute(path: string, runtime: MockRuntime = createMockBackendRuntime()) {
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

describe("Department Manager operational workspaces", () => {
  it.each([
    ["/department-manager/audits", "manager-audits-page", "Audit Work Queue"],
    ["/department-manager/inspection-team", "manager-inspection-team-page", "Inspection Team"],
    ["/department-manager/findings-review", "manager-findings-review-page", "Findings Review"],
    ["/department-manager/cap-monitoring", "manager-cap-monitoring-page", "CAP Monitoring"],
    ["/department-manager/checklist-management", "manager-checklist-management-page", "Checklist Management"],
    ["/department-manager/organizations/ORG-FLY-NAMIBIA", "manager-organization-detail-page", "Fly Namibia"],
    ["/department-manager/preliminary-reports/PR-2026-018", "manager-preliminary-review-page", "Preliminary Report Review"],
    ["/department-manager/findings/FND-CAB-2026-001/closure-review", "manager-closure-review-page", "Department Manager Review"],
  ])("direct-loads %s as the real Manager workspace", async (path, testId, heading) => {
    const runtime = createMockBackendRuntime();
    if (path.includes("FND-CAB-2026-001")) await seedVisualRuntimeForPath(runtime, path);
    renderManagerRoute(path, runtime);

    const page = await screen.findByTestId(testId);
    expect(within(page).getByRole("heading", { level: 1, name: heading })).toBeVisible();
    expect(screen.getByTestId("application-shell")).toHaveAttribute("data-active-role", "manager");
    expect(screen.queryByTestId("route-pending-implementation")).toBeNull();
  });

  it("preserves exact source-role ownership for ui-audit-044 without a Lead shell substitute", async () => {
    const contract = REACT_ROUTE_CONTRACT_BY_ID.get("evidence-review");
    expect(contract).toMatchObject({
      auditId: "ui-audit-044",
      path: "/department-manager/evidence/FND-CAB-2026-001",
      requiredRole: "manager",
      parentId: "manager-findings-review",
    });

    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/department-manager/evidence/FND-CAB-2026-001");
    renderManagerRoute("/department-manager/evidence/FND-CAB-2026-001", runtime);

    const page = await screen.findByTestId("manager-inspection-evidence-page");
    expect(screen.getByTestId("application-shell")).toHaveAttribute("data-active-role", "manager");
    expect(page).toHaveTextContent("FND-CAB-2026-001");
    expect(page).toHaveTextContent("EV-CAB-2026-001-V2");
    expect(within(page).getByRole("button", { name: "Record Evidence review" })).toBeEnabled();
    expect(page).not.toHaveTextContent("Lead Inspector workspace");
    const navigation = screen.getByRole("navigation", { name: "Primary role navigation" });
    expect(within(navigation).getByRole("link", { name: "Findings Review" })).toHaveAttribute("aria-current", "page");
    expect(within(navigation).queryByRole("link", { name: "Evidence Review" })).toBeNull();
    expect(within(navigation).getAllByRole("link").filter((link) => link.hasAttribute("aria-current"))).toHaveLength(1);
  });

  it("honors an exact Audit query, projects Inspector ownership, and disables undeclared checklist continuation by Audit ID", async () => {
    renderManagerRoute("/department-manager/audits?auditId=AUD-2026-001");
    const page = await screen.findByTestId("manager-audits-page");
    expect(await within(page).findByRole("region", { name: "Audit AUD-2026-001 dossier" })).toBeVisible();
    expect(within(page).getByText("AUD-2026-099")).toBeVisible();
    const dossier = within(page).getByRole("region", { name: "Audit AUD-2026-001 dossier" });
    expect(dossier).toHaveTextContent("Fly Namibia");
    expect(dossier).toHaveTextContent("Amina Inspector");
    expect(dossier).toHaveTextContent("USR-INSPECTOR-AMINA");
    expect(dossier).not.toHaveTextContent("Department Manager oversight");
    expect(dossier).toHaveTextContent("Continue Cabin Inspection checklist");
    expect(within(dossier).getByRole("button", { name: "Checklist unavailable for AUD-2026-001" })).toHaveAttribute(
      "title",
      "Audit AUD-2026-001 has no declared Department Manager checklist-execution route.",
    );
  });

  it("projects Inspection Team as an exact Audit register and dossier without borrowing mutation authority", async () => {
    const user = userEvent.setup();
    renderManagerRoute("/department-manager/inspection-team");
    const page = await screen.findByTestId("manager-inspection-team-page");
    await within(page).findByText("AUD-2026-001");
    expect(within(page).getByText("AUD-2026-099")).toBeVisible();
    expect(within(page).queryByText("USR-ADMIN-ADA")).toBeNull();
    await user.click(within(page).getByRole("button", { name: "Open Inspection Team for AUD-2026-001" }));
    const dossier = within(page).getByRole("region", { name: "Inspection Team for Audit AUD-2026-001" });
    expect(dossier).toHaveTextContent("ORG-FLY-NAMIBIA");
    expect(dossier).toHaveTextContent("USR-LEAD-CANER");
    expect(dossier).toHaveTextContent("USR-INSPECTOR-AMINA");
    expect(within(dossier).getByRole("link", { name: "Open Audit AUD-2026-001" })).toHaveAttribute(
      "href",
      "/department-manager/audits?auditId=AUD-2026-001",
    );
    await user.click(within(dossier).getByRole("tab", { name: "Assignments" }));
    expect(within(dossier).getByRole("tabpanel")).toHaveTextContent("CAB-EMEQ-PBE-001");
    await user.click(within(dossier).getByRole("tab", { name: "Documents" }));
    expect(within(dossier).getByRole("tabpanel")).toHaveTextContent("RPT-CAB-2026-001-V1");
    await user.click(within(dossier).getByRole("tab", { name: "History" }));
    expect(within(dossier).getByRole("tabpanel")).toHaveTextContent("Audit team projection opened");
    expect(within(dossier).getByRole("button", { name: "Edit Inspection Team for AUD-2026-001 unavailable" })).toHaveAttribute(
      "title",
      "Audit AUD-2026-001 has no declared Department Manager team-mutation command in Plan 1.",
    );
  });

  it("carries an exact Organization Finding ID into Findings Review and honors the selection", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/department-manager/evidence/FND-CAB-2026-001");
    renderManagerRoute("/department-manager/organizations/ORG-FLY-NAMIBIA", runtime);
    const organizationPage = await screen.findByTestId("manager-organization-detail-page");
    expect(within(organizationPage).getByRole("link", { name: "Open FND-CAB-2026-001 in Findings Review" })).toHaveAttribute(
      "href",
      "/department-manager/findings-review?findingId=FND-CAB-2026-001",
    );

    cleanup();
    renderManagerRoute("/department-manager/findings-review?findingId=FND-CAB-2026-001", runtime);
    const findingsPage = await screen.findByTestId("manager-findings-review-page");
    expect(within(findingsPage).getByRole("region", { name: "Selected Finding dossier" })).toHaveTextContent("FND-CAB-2026-001");
  });

  it("enforces organizationId together with findingId in Findings Review", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/department-manager/evidence/FND-CAB-2026-001");

    renderManagerRoute(
      "/department-manager/findings-review?organizationId=ORG-FLY-NAMIBIA&findingId=FND-SKYCARGO-2026-099",
      runtime,
    );
    const page = await screen.findByTestId("manager-findings-review-page");
    const register = within(page).getByRole("region", { name: "Finding register" });
    expect(await within(register).findByRole("article", { name: "Finding FND-CAB-2026-001" })).toBeVisible();
    expect(within(register).queryByRole("article", { name: "Finding FND-SKYCARGO-2026-099" })).toBeNull();
    expect(within(page).getByRole("region", { name: "Selected Finding dossier" })).toHaveTextContent("FND-CAB-2026-001");
    expect(page).toHaveTextContent("Organization scope: ORG-FLY-NAMIBIA");
  });

  it("keeps Findings and CAP actions bound to exact records or visibly disabled", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/department-manager/evidence/FND-CAB-2026-001");
    const user = userEvent.setup();
    renderManagerRoute("/department-manager/findings-review", runtime);
    const findingsPage = await screen.findByTestId("manager-findings-review-page");
    const canonical = await within(findingsPage).findByRole("article", { name: "Finding FND-CAB-2026-001" });
    expect(canonical).toHaveAttribute("data-finding-id", "FND-CAB-2026-001");
    expect(within(canonical).getByRole("button", { name: /CAB-2026-001/ })).toHaveAttribute("aria-pressed", "true");
    const cargo = within(findingsPage).getByRole("article", { name: "Finding FND-SKYCARGO-2026-099" });
    expect(within(cargo).getByRole("button", { name: /CAR-2026-099/ })).toHaveAttribute("aria-pressed", "false");
    expect(within(canonical).getByRole("link", { name: "Open Evidence FND-CAB-2026-001" })).toHaveAttribute(
      "href",
      "/department-manager/evidence/FND-CAB-2026-001",
    );

    cleanup();
    renderManagerRoute("/department-manager/cap-monitoring", runtime);
    const capPage = await screen.findByTestId("manager-cap-monitoring-page");
    await user.selectOptions(within(capPage).getByLabelText("Finding"), "FND-SKYCARGO-2026-099");
    expect(within(capPage).getByRole("button", { name: "Closure review unavailable for FND-SKYCARGO-2026-099" })).toHaveAttribute(
      "title",
      "Finding FND-SKYCARGO-2026-099 has no declared Department Manager closure-review route.",
    );
  });

  it("shows the published checklist version read-only with a record-specific edit reason", async () => {
    renderManagerRoute("/department-manager/checklist-management");
    const page = await screen.findByTestId("manager-checklist-management-page");
    expect(await within(page).findByText("CTV-CABIN-1")).toBeVisible();
    expect(within(page).getByText("CAB-EMEQ-PBE-001")).toBeVisible();
    expect(within(page).getByRole("button", { name: "Edit CTV-CABIN-1 unavailable" })).toHaveAttribute(
      "title",
      "Checklist Template Version CTV-CABIN-1 is published and no Department Manager draft-mutation command is declared in Plan 1.",
    );
  });

  it("forwards the exact Department-review Preliminary Report and records actor, reason, revision, version, and audit event", async () => {
    const runtime = createMockBackendRuntime();
    const user = userEvent.setup();
    renderManagerRoute("/department-manager/preliminary-reports/PR-2026-018", runtime);
    const page = await screen.findByTestId("manager-preliminary-review-page");
    expect(await within(page).findByText("PR-2026-018-V1")).toBeVisible();
    expect(within(page).getByText("PR-2026-018-V0")).toBeVisible();
    expect(within(page).getByText("Clarify Finding basis and supporting Evidence.")).toBeVisible();
    await user.type(within(page).getByLabelText("Department Manager decision reason"), "Department review complete for immutable version 1.");
    await user.click(within(page).getByRole("button", { name: "Forward PR-2026-018-V1 to General Manager" }));
    expect(await within(page).findByTestId("manager-preliminary-status")).toHaveTextContent("GM_REVIEW");

    const events = await runtime.backendForRole("manager").auditTrail.list({ entityType: "report_version", entityId: "PR-2026-018-V1" });
    expect(events.items.at(-1)).toMatchObject({
      actorSubjectId: "USR-MANAGER-NORA",
      actorRole: "manager",
      action: "report.decision_recorded",
      entityType: "report_version",
      entityId: "PR-2026-018-V1",
      beforeStatus: "DEPARTMENT_REVIEW",
      afterStatus: "GM_REVIEW",
      reason: "Department review complete for immutable version 1.",
      entityRevision: 2,
    });
    expect(await runtime.backendForRole("manager").reports.getVersion({ reportVersionId: "PR-2026-018-V1" })).toMatchObject({ revision: 2, version: 1 });
  });

  it("renders a truthful empty Preliminary Report review when connected storage has no versions", async () => {
    const runtime = createMockBackendRuntime();
    const manager = runtime.backendForRole("manager");
    vi.spyOn(manager.documents, "list").mockResolvedValue({ items: [], nextCursor: null });
    const getVersion = vi.spyOn(manager.reports, "getVersion").mockRejectedValue(new Error("Not found."));

    renderManagerRoute("/department-manager/preliminary-reports/PR-2026-018", runtime);

    const page = await screen.findByTestId("manager-preliminary-review-page");
    expect(within(page).getByText("No Preliminary Report versions are available yet.")).toBeVisible();
    expect(within(page).queryByRole("alert")).toBeNull();
    expect(getVersion).not.toHaveBeenCalled();
  });

  it("records authorized closure separately with exact Finding identity, reason, actor, revision, and audit event", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/department-manager/evidence/FND-CAB-2026-001");
    const user = userEvent.setup();
    renderManagerRoute("/department-manager/findings/FND-CAB-2026-001/closure-review", runtime);
    const page = await screen.findByTestId("manager-closure-review-page");
    expect(await within(page).findByText("FND-CAB-2026-001")).toBeVisible();
    await user.click(within(page).getByRole("button", { name: "Authorize closure" }));
    expect(await within(page).findByRole("alert")).toHaveTextContent("Authorized closure reason is required");
    await user.type(within(page).getByLabelText("Authorized closure reason"), "Department authority confirms the explicit alternate closure basis.");
    await user.click(within(page).getByRole("button", { name: "Authorize closure" }));
    expect(await within(page).findByTestId("manager-closure-status")).toHaveTextContent("CLOSED");
    expect(within(page).getByTestId("manager-closure-basis")).toHaveTextContent("AUTHORIZED");

    const events = await runtime.backendForRole("manager").auditTrail.list({ entityType: "finding", entityId: "FND-CAB-2026-001" });
    expect(events.items.at(-1)).toMatchObject({
      actorSubjectId: "USR-MANAGER-NORA",
      actorRole: "manager",
      action: "finding.authorized_closure",
      entityType: "finding",
      entityId: "FND-CAB-2026-001",
      reason: "Department authority confirms the explicit alternate closure basis.",
      entityRevision: 9,
    });
  });

  it("denies Manager CAP review and records Manager not-close Evidence outcomes without closing the Finding", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/department-manager/evidence/FND-CAB-2026-001");
    const manager = runtime.backendForRole("manager");
    await expect(manager.caps.review({
      operationId: "OP-MANAGER-CAP-DENIED",
      capRevisionId: "CAP-CAB-SCREEN-R2",
      expectedCapRevision: 2,
      findingId: "FND-CAB-2026-001",
      expectedFindingRevision: 1,
      decision: "ACCEPT",
      commentToAuditee: "Not authorized.",
      internalCaaNote: "Manager must not borrow Lead authority.",
    })).rejects.toThrow("CAA Inspector or Lead Inspector authority is required to review CAP");

    const findingBefore = await manager.findings.get({ findingId: "FND-CAB-2026-001" });
    const latestEvidence = (await manager.evidence.listVersions({ findingId: findingBefore.id })).at(-1);
    if (!latestEvidence) throw new Error("Expected a latest Evidence version.");
    await manager.evidence.review({
      operationId: "OP-MANAGER-NOT-CLOSE",
      evidenceVersionId: latestEvidence.id,
      expectedEvidenceVersionRevision: latestEvidence.revision,
      findingId: findingBefore.id,
      expectedFindingRevision: findingBefore.revision,
      decision: "NOT_CLOSE",
      commentToAuditee: "The cabin position confirmation remains incomplete.",
      internalCaaNote: "Department review keeps the Finding open for exact Evidence version 2.",
    });
    const findingAfter = await manager.findings.get({ findingId: findingBefore.id });
    expect(findingAfter).toMatchObject({
      id: "FND-CAB-2026-001",
      status: "EVIDENCE_MORE_INFORMATION_REQUESTED",
      closedAt: null,
      closureBasis: null,
    });
    const events = await manager.auditTrail.list({ entityType: "finding", entityId: findingBefore.id });
    expect(events.items.at(-1)).toMatchObject({
      actorSubjectId: "USR-MANAGER-NORA",
      actorRole: "manager",
      action: "evidence.reviewed",
      entityType: "finding",
      entityId: findingBefore.id,
      beforeStatus: "PENDING_CAA_REVIEW",
      afterStatus: "EVIDENCE_MORE_INFORMATION_REQUESTED",
      reason: "The cabin position confirmation remains incomplete.",
      entityRevision: findingAfter.revision,
    });

    const latestAfter = (await manager.evidence.listVersions({ findingId: findingAfter.id })).at(-1);
    if (!latestAfter) throw new Error("Expected the reviewed Evidence version.");
    await expect(manager.evidence.review({
      operationId: "OP-MANAGER-INVALID-EVIDENCE-STAGE",
      evidenceVersionId: latestAfter.id,
      expectedEvidenceVersionRevision: latestAfter.revision,
      findingId: findingAfter.id,
      expectedFindingRevision: findingAfter.revision,
      decision: "CLOSE",
      commentToAuditee: "A repeated decision must be denied.",
      internalCaaNote: "The immutable review is already recorded.",
    })).rejects.toThrow("Evidence version is not pending CAA review");
  });

  it("records both live Department Manager RETURN and FORWARD report branches with exact actor and revision", async () => {
    const returnedRuntime = createMockBackendRuntime();
    const returnedManager = returnedRuntime.backendForRole("manager");
    const returnedReport = await returnedManager.reports.decide({
      operationId: "OP-REPORT-RETURN-PR-2026-018-V1-R1",
      reportVersionId: "PR-2026-018-V1",
      expectedReportVersionRevision: 1,
      decision: "RETURN",
      reason: "Return exact Preliminary Report version 1.",
    });
    expect(returnedReport).toMatchObject({ status: "RETURNED", revision: 2 });
    expect((await returnedManager.auditTrail.list({ entityType: "report_version", entityId: "PR-2026-018-V1" })).items.at(-1)).toMatchObject({
      actorSubjectId: "USR-MANAGER-NORA",
      actorRole: "manager",
      beforeStatus: "DEPARTMENT_REVIEW",
      afterStatus: "RETURNED",
      entityRevision: 2,
    });

    const forwardedRuntime = createMockBackendRuntime();
    const forwardedManager = forwardedRuntime.backendForRole("manager");
    await forwardedManager.reports.decide({
      operationId: "OP-REPORT-FORWARD-PR-2026-018-V1-R1",
      reportVersionId: "PR-2026-018-V1",
      expectedReportVersionRevision: 1,
      decision: "FORWARD",
      reason: "Forward exact Preliminary Report version 1.",
    });
    expect((await forwardedManager.auditTrail.list({ entityType: "report_version", entityId: "PR-2026-018-V1" })).items.at(-1)).toMatchObject({
      actorSubjectId: "USR-MANAGER-NORA",
      actorRole: "manager",
      beforeStatus: "DEPARTMENT_REVIEW",
      afterStatus: "GM_REVIEW",
      entityRevision: 2,
    });
  });

  it("denies stale or mismatched Manager Evidence decision identity", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/department-manager/evidence/FND-CAB-2026-001");
    const manager = runtime.backendForRole("manager");
    const finding = await manager.findings.get({ findingId: "FND-CAB-2026-001" });
    const versions = await manager.evidence.listVersions({ findingId: finding.id });
    const previous = versions[0];
    const latest = versions.at(-1);
    if (!previous || !latest) throw new Error("Expected immutable Evidence history.");

    await expect(manager.evidence.review({
      operationId: "OP-MANAGER-EVIDENCE-STALE",
      evidenceVersionId: latest.id,
      expectedEvidenceVersionRevision: latest.revision - 1,
      findingId: finding.id,
      expectedFindingRevision: finding.revision,
      decision: "CLOSE",
      commentToAuditee: "Stale revision attempt.",
      internalCaaNote: "Stale revision must fail closed.",
    })).rejects.toThrow("Evidence version revision conflict");

    await expect(manager.evidence.review({
      operationId: "OP-MANAGER-EVIDENCE-WRONG-VERSION",
      evidenceVersionId: previous.id,
      expectedEvidenceVersionRevision: previous.revision,
      findingId: finding.id,
      expectedFindingRevision: finding.revision,
      decision: "CLOSE",
      commentToAuditee: "Wrong version attempt.",
      internalCaaNote: "Only the exact latest version can be reviewed.",
    })).rejects.toThrow("Evidence review must target the exact latest version");
  });

  it.each([1440, 1024, 390])("keeps Manager mobile hierarchy and filters usable at %ipx", async (width) => {
    Object.defineProperty(window, "innerWidth", { configurable: true, value: width });
    renderManagerRoute("/department-manager/findings-review");
    const page = await screen.findByTestId("manager-findings-review-page");
    const filters = within(page).getByRole("region", { name: "Finding filters" });
    const register = within(page).getByRole("region", { name: "Finding register" });
    const dossier = within(page).getByRole("region", { name: "Selected Finding dossier" });
    expect(filters.compareDocumentPosition(register) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(register.compareDocumentPosition(dossier) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    await waitFor(() => expect(within(page).getByLabelText("Status")).toBeEnabled());
  });
});
