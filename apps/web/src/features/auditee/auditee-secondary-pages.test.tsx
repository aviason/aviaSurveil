// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { AppRouter } from "../../app/router";
import { ScenarioProvider } from "../../app/scenario-context";
import { DEMO_MOCK_STORAGE_KEY } from "../../app/demo-persistence";
import type {
  CommunicationView,
  DemoBackend,
  DocumentMetadataView,
  LocalDate,
} from "../../backend/backend";
import {
  createMockBackendPersistentRuntime,
  createMockBackendRuntime,
} from "../../mock/create-mock-backend";
import { seedVisualRuntimeForPath } from "../../mock/seed-visual-runtime";
import type { MockState } from "../../mock/seed-data";

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

interface PersistedEnvelopeFixture {
  schemaVersion: number;
  state: MockState;
  operations: unknown[];
}

interface AuditeeCoordinationView {
  auditId: string;
  organizationId: string;
  organizationName: string;
  title: string;
  inspectionCategory: "Routine / Announced";
  scheduledStartDate: LocalDate;
  status: "AWAITING_AUDITEE_CONFIRMATION" | "CONFIRMED" | "ALTERNATIVE_PROPOSED";
  alternativeDate: LocalDate | null;
  nextAction: string;
  revision: number;
}

interface AuditeeReleasedReportView {
  reportVersionId: string;
  reportId: string;
  kind: "PRELIMINARY" | "FINAL";
  organizationId: string;
  auditId: string;
  findingIds: string[];
  version: number;
  status: "LOCKED";
  revision: number;
  issuedAt: string;
  responseDueDate: LocalDate | null;
  caaVisibleCommentState: "NO_COMMENT_RECORDED" | "RECORDED";
  caaVisibleComment: string | null;
}

interface AuditeeCapabilities {
  auditeeCoordination?: {
    list(input: Record<string, never>): Promise<{ items: AuditeeCoordinationView[]; nextCursor: null }>;
    respond(input: {
      auditId: string;
      organizationId: string;
      expectedRevision: number;
      idempotencyKey: string;
      decision: "CONFIRM" | "PROPOSE_ALTERNATIVE";
      alternativeDate: LocalDate | null;
    }): Promise<AuditeeCoordinationView>;
  };
  auditeeReports?: {
    listReleased(input: { kind?: "PRELIMINARY" | "FINAL" }): Promise<{ items: AuditeeReleasedReportView[]; nextCursor: null }>;
    getReleased(input: { reportVersionId: string }): Promise<AuditeeReleasedReportView>;
  };
}

function auditeeBackend(runtime: MockRuntime) {
  return runtime.backendForRole("auditee") as DemoBackend & AuditeeCapabilities;
}

function renderAuditeeRoute(path: string, runtime: MockRuntime = createMockBackendRuntime()) {
  render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: "demo",
      environmentLabel: "test",
      identityMode: "demo-role-switch",
      subjectId: "USR-AUDITEE-FLY",
    }}>
      <ScenarioProvider>
        <MemoryRouter initialEntries={[path]}><AppRouter /></MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
  return runtime;
}

async function lockReports(runtime: MockRuntime) {
  const manager = runtime.backendForRole("manager");
  const gm = runtime.backendForRole("gm");
  const executive = runtime.backendForRole("executiveDirector");

  const preliminary = await manager.reports.getVersion({ reportVersionId: "PR-2026-018-V1" });
  const preliminaryAtGm = await manager.reports.decide({
    operationId: "TASK9-MANAGER-PR",
    reportVersionId: preliminary.reportVersionId,
    expectedReportVersionRevision: preliminary.revision,
    decision: "FORWARD",
    reason: "Department Manager forwarded the exact Preliminary Report version.",
  });
  const preliminaryAtExecutive = await gm.reports.decide({
    operationId: "TASK9-GM-PR",
    reportVersionId: preliminaryAtGm.reportVersionId,
    expectedReportVersionRevision: preliminaryAtGm.revision,
    decision: "FORWARD",
    reason: "General Manager forwarded the exact Preliminary Report version.",
  });
  await executive.reports.decide({
    operationId: "TASK9-EXEC-PR",
    reportVersionId: preliminaryAtExecutive.reportVersionId,
    expectedReportVersionRevision: preliminaryAtExecutive.revision,
    decision: "ISSUE_AND_LOCK",
    reason: "Executive Director issued and locked the exact Preliminary Report version.",
  });

  const finalReport = await executive.reports.getVersion({ reportVersionId: "RPT-CAB-2026-001-V1" });
  await executive.reports.decide({
    operationId: "TASK9-EXEC-FINAL",
    reportVersionId: finalReport.reportVersionId,
    expectedReportVersionRevision: finalReport.revision,
    decision: "ISSUE_AND_LOCK",
    reason: "Executive Director issued and locked the exact Final Report version.",
  });
}

function readPersistedEnvelope(storage: Storage): PersistedEnvelopeFixture {
  const raw = storage.getItem(DEMO_MOCK_STORAGE_KEY);
  if (!raw) throw new Error("Expected a persisted mock envelope.");
  return JSON.parse(raw) as PersistedEnvelopeFixture;
}

function writePersistedEnvelope(storage: Storage, envelope: PersistedEnvelopeFixture): void {
  storage.setItem(DEMO_MOCK_STORAGE_KEY, JSON.stringify(envelope));
}

function addReleasedFinalReportFixture(
  envelope: PersistedEnvelopeFixture,
  reportVersionId = "RPT-CAB-2026-002-V1",
  reportId = "RPT-CAB-2026-002",
): void {
  const source = envelope.state.reportVersions["RPT-CAB-2026-001-V1"];
  if (!source) throw new Error("Expected the canonical Final Report fixture.");
  envelope.state.reportVersions[reportVersionId] = {
    ...source,
    reportVersionId,
    reportId,
    findingIds: [],
    contentHash: "sha256:second-released-final-report",
  };
  envelope.state.reportPublicMetadata[reportVersionId] = {
    kind: "FINAL",
    responseDueDate: null,
    caaVisibleComment: null,
  };
}

beforeEach(() => localStorage.clear());
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("Auditee secondary workspaces", () => {
  it("direct-loads all seven new routes and exposes every primary destination with one role-correct active item", async () => {
    const routes = [
      ["/auditee/inspection-coordination", "auditee-inspection-coordination-page", "Inspection Coordination", "Inspection Coordination"],
      ["/auditee/preliminary-reports", "auditee-preliminary-reports-page", "Preliminary Reports", "Preliminary Reports"],
      ["/auditee/final-reports", "auditee-final-reports-page", "Final Reports", "Final Reports"],
      ["/auditee/reports/RPT-CAB-2026-001", "auditee-report-preview-page", "Report Preview", "Final Reports"],
      ["/auditee/messages", "auditee-messages-page", "Messages from the CAA", "Messages"],
      ["/auditee/documents", "auditee-documents-page", "Documents", "Documents"],
      ["/auditee/settings", "auditee-settings-page", "Service Provider Settings", "Settings"],
    ] as const;

    for (const [path, testId, heading, activeLabel] of routes) {
      renderAuditeeRoute(path);
      const page = await screen.findByTestId(testId);
      expect(within(page).getByRole("heading", { level: 1, name: heading })).toBeVisible();
      expect(page.querySelector(".workbench-page-header"), `${path} must use the shared workbench page header`).not.toBeNull();
      expect(screen.queryByTestId("route-pending-implementation")).toBeNull();
      const navigation = screen.getByRole("navigation", { name: "Primary role navigation" });
      expect(within(navigation).getByRole("link", { name: activeLabel })).toHaveAttribute("aria-current", "page");
      expect(within(navigation).getAllByRole("link").filter((link) => link.hasAttribute("aria-current"))).toHaveLength(1);
      expect(screen.getByTestId("application-shell")).toHaveAttribute("data-active-role", "auditee");
      cleanup();
    }
  });

  it("projects only released Routine or Announced own-organization coordination and persists revision-safe idempotent responses", async () => {
    const runtime = createMockBackendRuntime();
    const auditee = auditeeBackend(runtime);
    expect(auditee.auditeeCoordination).toBeDefined();
    if (!auditee.auditeeCoordination) return;

    const initial = await auditee.auditeeCoordination.list({});
    expect(initial.items).toEqual([
      expect.objectContaining({
        auditId: "AUD-2026-001",
        organizationId: "ORG-FLY-NAMIBIA",
        organizationName: "Fly Namibia",
        inspectionCategory: "Routine / Announced",
        scheduledStartDate: "2026-06-15",
        status: "AWAITING_AUDITEE_CONFIRMATION",
        revision: 1,
        nextAction: "Confirm the proposed inspection date or propose an alternative date to the CAA.",
      }),
    ]);
    const raw = JSON.stringify(initial);
    expect(raw).not.toMatch(/AUD-2026-099|ORG-SKYCARGO|SkyCargo|USR-INSPECTOR|currentOwner|dueDate|noticeWithheld|UNANNOUNCED|AD_HOC/i);

    const confirmed = await auditee.auditeeCoordination.respond({
      auditId: "AUD-2026-001",
      organizationId: "ORG-FLY-NAMIBIA",
      expectedRevision: 1,
      idempotencyKey: "TASK9-CONFIRM-AUD-2026-001-R1",
      decision: "CONFIRM",
      alternativeDate: null,
    });
    expect(confirmed).toMatchObject({ status: "CONFIRMED", revision: 2, scheduledStartDate: "2026-06-15" });
    expect(await auditee.auditeeCoordination.respond({
      auditId: "AUD-2026-001",
      organizationId: "ORG-FLY-NAMIBIA",
      expectedRevision: 1,
      idempotencyKey: "TASK9-CONFIRM-AUD-2026-001-R1",
      decision: "CONFIRM",
      alternativeDate: null,
    })).toEqual(confirmed);
    await expect(auditee.auditeeCoordination.respond({
      auditId: "AUD-2026-001",
      organizationId: "ORG-FLY-NAMIBIA",
      expectedRevision: 1,
      idempotencyKey: "TASK9-STALE-AUD-2026-001-R1",
      decision: "PROPOSE_ALTERNATIVE",
      alternativeDate: "2026-06-22",
    })).rejects.toThrow(/revision conflict/i);
    await expect((runtime.backendForRole("manager") as DemoBackend & AuditeeCapabilities).auditeeCoordination?.list({})).rejects.toThrow(/Auditee/i);
  });

  it("releases typed Preliminary and Final report projections only at LOCKED without private fields or inferred relationships", async () => {
    const runtime = createMockBackendRuntime();
    const auditee = auditeeBackend(runtime);
    expect(auditee.auditeeReports).toBeDefined();
    if (!auditee.auditeeReports) return;

    expect((await auditee.auditeeReports.listReleased({})).items).toEqual([]);
    await expect(auditee.auditeeReports.getReleased({ reportVersionId: "PR-2026-018-V1" })).rejects.toThrow(/unavailable/i);
    await expect(auditee.auditeeReports.getReleased({ reportVersionId: "RPT-CAB-2026-001-V1" })).rejects.toThrow(/unavailable/i);

    await lockReports(runtime);
    const all = await auditee.auditeeReports.listReleased({});
    expect(all.items.map((report) => [report.reportVersionId, report.kind])).toEqual([
      ["PR-2026-018-V1", "PRELIMINARY"],
      ["RPT-CAB-2026-001-V1", "FINAL"],
    ]);
    expect(all.items.every((report) => report.status === "LOCKED" && report.organizationId === "ORG-FLY-NAMIBIA")).toBe(true);
    expect(all.items.every((report) => report.responseDueDate === null && report.caaVisibleCommentState === "NO_COMMENT_RECORDED" && report.caaVisibleComment === null)).toBe(true);
    expect(all.items.find((report) => report.kind === "FINAL")?.findingIds).toEqual([]);
    expect(JSON.stringify(all)).not.toMatch(/contentHash|sha256|Internal CAA|risk|workload|enforcement|ORG-SKYCARGO|SkyCargo/i);
    expect((await auditee.auditeeReports.listReleased({ kind: "PRELIMINARY" })).items.map((report) => report.reportVersionId)).toEqual(["PR-2026-018-V1"]);
  });

  it("keeps Preliminary preview on its exact version, final preview contextual, and downloads unable to close a Finding", async () => {
    const runtime = createMockBackendRuntime();
    await lockReports(runtime);
    renderAuditeeRoute("/auditee/preliminary-reports", runtime);
    const user = userEvent.setup();
    const preliminaryPage = await screen.findByTestId("auditee-preliminary-reports-page");
    expect(preliminaryPage).toHaveTextContent("PR-2026-018-V1");
    expect(preliminaryPage).toHaveTextContent("Response Due Date: Not configured");
    expect(preliminaryPage).toHaveTextContent("CAA-visible comment: No CAA-visible comment recorded");
    expect(within(preliminaryPage).getByRole("combobox", { name: "Release stage" })).toBeDisabled();
    await user.click(within(preliminaryPage).getByRole("button", { name: "Preview PR-2026-018-V1" }));
    expect(within(preliminaryPage).getByRole("region", { name: "Preliminary Report preview PR-2026-018-V1" })).toHaveTextContent("PR-2026-018-V1");
    expect(within(preliminaryPage).queryByText("RPT-CAB-2026-001-V1")).toBeNull();

    cleanup();
    renderAuditeeRoute("/auditee/reports/RPT-CAB-2026-001-V1", runtime);
    const finalPage = await screen.findByTestId("auditee-report-preview-page");
    expect(finalPage).toHaveAttribute("data-report-version-id", "RPT-CAB-2026-001-V1");
    expect(finalPage).toHaveTextContent("No Finding is linked to this report version.");
    expect(finalPage).not.toHaveTextContent(/contentHash|sha256:|FR-2025-009/i);
    expect(within(finalPage).getByRole("link", { name: "Findings Overview" })).toHaveAttribute("href", "#auditee-report-findings");
    expect(within(finalPage).getByRole("button", { name: "Download unavailable" })).toBeDisabled();
  });

  it("lists only safe own-organization released Reports and exact Evidence versions with public review results and denies direct unsafe documents", async () => {
    const runtime = createMockBackendRuntime();
    await seedVisualRuntimeForPath(runtime, "/department-manager/evidence/FND-CAB-2026-001");
    await lockReports(runtime);
    const auditee = auditeeBackend(runtime);
    const documents = await auditee.documents.list({ organizationId: "ORG-SKYCARGO" });
    expect(documents.items.every((document) => document.organizationId === "ORG-FLY-NAMIBIA")).toBe(true);
    expect(documents.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: "RPT-CAB-2026-001-V1", kind: "REPORT", version: 1, publicReviewResult: "RELEASED" }),
      expect.objectContaining({ kind: "EVIDENCE", version: 1, publicReviewResult: "PARTIALLY_ACCEPTED" }),
      expect.objectContaining({ kind: "EVIDENCE", version: 2, publicReviewResult: "PENDING_CAA_REVIEW" }),
    ]));
    expect(JSON.stringify(documents)).not.toMatch(/PR-2026-018-V0|contentHash|Internal CAA|ORG-SKYCARGO|SkyCargo/i);
    await expect(auditee.documents.open({ documentId: "PR-2026-018-V0" })).rejects.toThrow(/unavailable/i);
    await expect(auditee.documents.open({ documentId: "EVD-CAR-2026-099-V2" })).rejects.toThrow(/unavailable/i);
    await expect(auditee.documents.open({ documentId: "UNKNOWN-DOCUMENT" })).rejects.toThrow(/unavailable/i);

    renderAuditeeRoute("/auditee/documents", runtime);
    const page = await screen.findByTestId("auditee-documents-page");
    const exact = await within(page).findByRole("row", { name: /RPT-CAB-2026-001-V1/ });
    expect(within(exact).getByRole("button", { name: "Download RPT-CAB-2026-001-V1 unavailable" })).toBeDisabled();
    expect(document.body).not.toHaveTextContent(/ORG-SKYCARGO|SkyCargo|Internal CAA|inspector workload|risk score|enforcement/i);
  });

  it("sends and rehydrates only own-organization Auditee-visible communication without exposing a CAA-private subject", async () => {
    const runtime = createMockBackendRuntime();
    const auditee = auditeeBackend(runtime);
    const sent = await auditee.communications.send({
      expectedRevision: null,
      idempotencyKey: "MSG-AUDITEE-FLY-1",
      organizationId: "ORG-FLY-NAMIBIA",
      subject: "Coordination question",
      body: "Please confirm the entrance briefing time.",
      audience: "CAA",
    });
    expect(sent).toMatchObject({ direction: "AUDITEE_TO_CAA", organizationId: "ORG-FLY-NAMIBIA" } satisfies Partial<CommunicationView>);
    const listed = await auditee.communications.list({ organizationId: "ORG-SKYCARGO" });
    expect(listed.items).toEqual([expect.objectContaining({ id: "MSG-AUDITEE-FLY-1", direction: "AUDITEE_TO_CAA" })]);
    expect(JSON.stringify(listed)).not.toMatch(/senderSubjectId|USR-INSPECTOR|USR-MANAGER|Internal CAA Note|ORG-SKYCARGO|SkyCargo/i);

    renderAuditeeRoute("/auditee/messages", runtime);
    const page = await screen.findByTestId("auditee-messages-page");
    expect(page).toHaveTextContent("Auditee-visible communication only");
    expect(JSON.stringify(await auditee.communications.list({}))).toContain("Coordination question");
    expect(within(page).getByRole("button", { name: "Compose message to CAA" })).toBeEnabled();
    expect(within(page).queryByLabelText("Visibility")).toBeNull();
    expect(page).not.toHaveTextContent(/Internal CAA Note|CAA-only|private message/i);
  });

  it("persists subject-scoped profile updates after remount and keeps undeclared notification preferences read-only", async () => {
    const storage = window.localStorage;
    const runtime = createMockBackendPersistentRuntime(storage);
    const user = userEvent.setup();
    renderAuditeeRoute("/auditee/settings", runtime);
    const page = await screen.findByTestId("auditee-settings-page");
    expect(page).toHaveTextContent("Service Provider Settings");
    expect(await within(page).findByRole("region", { name: "Organization scope" })).toHaveTextContent("ORG-FLY-NAMIBIA");
    expect(await within(page).findByRole("region", { name: "Notification preferences" })).toHaveTextContent(/read-only|not configurable/i);
    for (const name of ["Due Date reminders", "Report release updates"]) {
      expect(within(page).getByRole("checkbox", { name })).toBeDisabled();
      expect(within(page).getByRole("checkbox", { name })).toHaveAttribute(
        "aria-describedby",
        "auditee-notification-disabled-reason",
      );
    }
    expect(within(page).getByText(/Notification preferences are read-only/i)).toHaveAttribute(
      "id",
      "auditee-notification-disabled-reason",
    );
    await user.click(within(page).getByRole("button", { name: "Edit profile" }));
    await user.clear(within(page).getByLabelText("Display name"));
    await user.type(within(page).getByLabelText("Display name"), "Fly Namibia Quality Lead");
    await user.click(within(page).getByRole("button", { name: "Save profile" }));
    expect(await within(page).findByRole("status")).toHaveTextContent("Profile saved");
    cleanup();

    renderAuditeeRoute("/auditee/settings", createMockBackendPersistentRuntime(storage));
    const remounted = await screen.findByTestId("auditee-settings-page");
    expect(remounted).toHaveTextContent("Fly Namibia Quality Lead");
    expect(JSON.stringify(await auditeeBackend(runtime).profiles.getMine({}))).not.toMatch(/Internal CAA|risk|workload|enforcement/i);
  });

  it("fails closed across legacy Auditee APIs and strips CAA-only owner, planning, notice, hash, and next-action fields from safe outputs", async () => {
    const runtime = createMockBackendRuntime();
    await lockReports(runtime);
    const auditee = auditeeBackend(runtime);

    const legacyReads = await Promise.allSettled([
      auditee.assignments.list({}),
      auditee.reports.getVersion({ reportVersionId: "RPT-CAB-2026-001-V1" }),
      auditee.planning.list({}),
    ]);
    expect.soft(legacyReads.map((result) => result.status)).toEqual(["rejected", "rejected", "rejected"]);

    const readable = {
      coordination: await auditee.auditeeCoordination?.list({}),
      reports: await auditee.auditeeReports?.listReleased({}),
      documents: await auditee.documents.list({}),
      messages: await auditee.communications.list({}),
      calendar: await auditee.calendar.list({}),
    };
    expect.soft(JSON.stringify(readable)).not.toMatch(
      /contentHash|currentOwnerId|currentOwnerRole|senderSubjectId|planningItems|noticeWithheld|Continue Cabin Inspection checklist/i,
    );
  });

  it("requires every Auditee report document to satisfy the exact public release predicate", async () => {
    const storage = window.localStorage;
    const runtime = createMockBackendPersistentRuntime(storage);
    await lockReports(runtime);
    const envelope = readPersistedEnvelope(storage);
    const source = envelope.state.reportVersions["RPT-CAB-2026-001-V1"]!;

    envelope.state.reportVersions["RPT-NO-METADATA-V1"] = {
      ...source,
      reportVersionId: "RPT-NO-METADATA-V1",
      reportId: "RPT-NO-METADATA",
      contentHash: "sha256:no-public-metadata",
    };
    envelope.state.reportVersions["RPT-NOT-ISSUED-V1"] = {
      ...source,
      reportVersionId: "RPT-NOT-ISSUED-V1",
      reportId: "RPT-NOT-ISSUED",
      contentHash: "sha256:not-issued",
      issuedAt: null,
    };
    envelope.state.reportPublicMetadata["RPT-NOT-ISSUED-V1"] = {
      kind: "FINAL",
      responseDueDate: null,
      caaVisibleComment: null,
    };
    envelope.state.reportVersions["RPT-FOREIGN-FINDING-V1"] = {
      ...source,
      reportVersionId: "RPT-FOREIGN-FINDING-V1",
      reportId: "RPT-FOREIGN-FINDING",
      contentHash: "sha256:foreign-finding",
      findingIds: ["FND-SKYCARGO-2026-099"],
    };
    envelope.state.reportPublicMetadata["RPT-FOREIGN-FINDING-V1"] = {
      kind: "FINAL",
      responseDueDate: null,
      caaVisibleComment: null,
    };
    writePersistedEnvelope(storage, envelope);

    const remounted = auditeeBackend(createMockBackendPersistentRuntime(storage));
    const reportDocuments = (await remounted.documents.list({})).items
      .filter((document) => document.kind === "REPORT")
      .map((document) => document.id)
      .sort();
    expect(reportDocuments).toEqual(["PR-2026-018-V1", "RPT-CAB-2026-001-V1"]);
    await expect(remounted.documents.open({ documentId: "RPT-NO-METADATA-V1" })).rejects.toThrow(/unavailable/i);
    await expect(remounted.documents.open({ documentId: "RPT-NOT-ISSUED-V1" })).rejects.toThrow(/unavailable/i);
    await expect(remounted.documents.open({ documentId: "RPT-FOREIGN-FINDING-V1" })).rejects.toThrow(/unavailable/i);
  });

  it("resets an older schema-v1 persisted envelope missing Task 9 state and safely hydrates Coordination, Reports, Messages, and Settings", async () => {
    const storage = window.localStorage;
    const runtime = createMockBackendPersistentRuntime(storage);
    const auditee = auditeeBackend(runtime);
    await auditee.communications.send({
      expectedRevision: null,
      idempotencyKey: "TASK9-LEGACY-ENVELOPE-WRITE",
      organizationId: "ORG-FLY-NAMIBIA",
      subject: "Legacy envelope fixture",
      body: "Persist a schema envelope before removing Task 9 fields.",
      audience: "CAA",
    });
    const envelope = readPersistedEnvelope(storage);
    envelope.schemaVersion = 1;
    const legacyState = envelope.state as Partial<MockState>;
    delete legacyState.reportPublicMetadata;
    delete legacyState.auditeeCoordinationResponses;
    writePersistedEnvelope(storage, envelope);

    const migrated = auditeeBackend(createMockBackendPersistentRuntime(storage));
    await expect(migrated.auditeeCoordination?.list({})).resolves.toMatchObject({
      items: [expect.objectContaining({ auditId: "AUD-2026-001", status: "AWAITING_AUDITEE_CONFIRMATION" })],
    });
    await expect(migrated.auditeeReports?.listReleased({})).resolves.toEqual({ items: [], nextCursor: null });
    await expect(migrated.communications.list({})).resolves.toEqual({
      items: [expect.objectContaining({
        id: "TASK9-LEGACY-ENVELOPE-WRITE",
        direction: "AUDITEE_TO_CAA",
        organizationId: "ORG-FLY-NAMIBIA",
      })],
      nextCursor: null,
    });
    await expect(migrated.profiles.getMine({})).resolves.toMatchObject({
      subjectId: "USR-AUDITEE-FLY",
      organizationId: "ORG-FLY-NAMIBIA",
      displayName: "Fly Namibia Auditee",
    });
  });

  it("persists an alternative coordination proposal and includes only public inbound plus own outbound messages after remount", async () => {
    const storage = window.localStorage;
    const runtime = createMockBackendPersistentRuntime(storage);
    const auditee = auditeeBackend(runtime);
    const manager = runtime.backendForRole("manager");

    const initial = (await auditee.auditeeCoordination?.list({}))?.items[0];
    expect(initial).toBeDefined();
    if (!initial || !auditee.auditeeCoordination) return;
    await auditee.auditeeCoordination.respond({
      auditId: initial.auditId,
      organizationId: initial.organizationId,
      expectedRevision: initial.revision,
      idempotencyKey: "TASK9-ALTERNATIVE-AUD-2026-001-R1",
      decision: "PROPOSE_ALTERNATIVE",
      alternativeDate: "2026-06-22",
    });
    await manager.communications.send({
      expectedRevision: null,
      idempotencyKey: "TASK9-CAA-PUBLIC-INBOUND",
      organizationId: "ORG-FLY-NAMIBIA",
      subject: "CAA-visible coordination reply",
      body: "The public coordination response is available.",
      audience: "AUDITEE",
    });
    await manager.communications.send({
      expectedRevision: null,
      idempotencyKey: "TASK9-CAA-PRIVATE-SAME-ORG",
      organizationId: "ORG-FLY-NAMIBIA",
      subject: "Internal CAA Note",
      body: "This same-organization message remains CAA-private.",
      audience: "CAA",
    });
    await auditee.communications.send({
      expectedRevision: null,
      idempotencyKey: "TASK9-AUDITEE-OUTBOUND",
      organizationId: "ORG-FLY-NAMIBIA",
      subject: "Auditee coordination response",
      body: "Fly Namibia proposes 22 June 2026.",
      audience: "CAA",
    });
    const envelope = readPersistedEnvelope(storage);
    envelope.state.communications.push(
      {
        id: "TASK9-MALFORMED-INBOUND-PAIR",
        organizationId: "ORG-FLY-NAMIBIA",
        subject: "Malformed inbound pair",
        body: "CAA_TO_AUDITEE must never be readable with a CAA-only audience.",
        audience: "CAA",
        direction: "CAA_TO_AUDITEE",
        senderSubjectId: "USR-MANAGER-NORA",
        revision: 1,
        createdAt: "2026-06-15T09:00:00.000Z",
      },
      {
        id: "TASK9-MALFORMED-OUTBOUND-PAIR",
        organizationId: "ORG-FLY-NAMIBIA",
        subject: "Malformed outbound pair",
        body: "AUDITEE_TO_CAA must never be readable with an Auditee audience.",
        audience: "AUDITEE",
        direction: "AUDITEE_TO_CAA",
        senderSubjectId: "USR-AUDITEE-FLY",
        revision: 1,
        createdAt: "2026-06-15T09:00:00.000Z",
      },
    );
    writePersistedEnvelope(storage, envelope);

    const remounted = auditeeBackend(createMockBackendPersistentRuntime(storage));
    expect((await remounted.auditeeCoordination?.list({}))?.items[0]).toMatchObject({
      status: "ALTERNATIVE_PROPOSED",
      alternativeDate: "2026-06-22",
      revision: 2,
    });
    const messages = await remounted.communications.list({});
    expect(messages.items.map((message) => [message.id, message.direction])).toEqual([
      ["TASK9-CAA-PUBLIC-INBOUND", "CAA_TO_AUDITEE"],
      ["TASK9-AUDITEE-OUTBOUND", "AUDITEE_TO_CAA"],
    ]);
    expect(JSON.stringify(messages)).not.toMatch(/TASK9-CAA-PRIVATE-SAME-ORG|TASK9-MALFORMED|Internal CAA Note|senderSubjectId|USR-MANAGER/i);
  });

  it("links only the canonical Final Report row to the contextual route and disables every other released Final with an exact reason", async () => {
    const storage = window.localStorage;
    const runtime = createMockBackendPersistentRuntime(storage);
    await lockReports(runtime);
    const envelope = readPersistedEnvelope(storage);
    addReleasedFinalReportFixture(envelope);
    addReleasedFinalReportFixture(envelope, "RPT-CAB-2026-001-V2", "RPT-CAB-2026-001");
    writePersistedEnvelope(storage, envelope);
    renderAuditeeRoute("/auditee/final-reports", createMockBackendPersistentRuntime(storage));

    const page = await screen.findByTestId("auditee-final-reports-page");
    const reportLinks = within(page).getAllByRole("link", { name: "Open report" });
    expect(reportLinks.length).toBeGreaterThan(0);
    expect(reportLinks.every((link) => link.getAttribute("href")?.startsWith("/auditee/reports/") ?? false)).toBe(true);
  });
});
