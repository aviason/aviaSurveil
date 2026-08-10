// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";
import "fake-indexeddb/auto";

import { MemoryRouter, Route, Routes } from "react-router-dom";
import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { AuditDetailPage } from "./audit-detail-page";

afterEach(cleanup);

function renderPage(
  runtime = createMockBackendRuntime(),
  initialEntry = "/inspector/audits/AUD-2026-001",
) {
  render(
    <AppProviders
      runtime={{
        backend: runtime.backend,
        backendForRole: runtime.backendForRole,
        buildProfile: "demo",
        environmentLabel: "test",
        identityMode: "demo-role-switch",
        subjectId: "USR-INSPECTOR-AMINA",
      }}
    >
      <ScenarioProvider>
        <MemoryRouter initialEntries={[initialEntry]}>
          <Routes>
            <Route path="/inspector/audits/:auditId" element={<AuditDetailPage />} />
          </Routes>
        </MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
}

describe("AuditDetailPage", () => {
  it("renders an audit dossier with owner, due date, package state, offline eligibility, and runner path", async () => {
    renderPage();

    const dossier = await screen.findByTestId("audit-dossier");
    for (const expected of [
      "Fly Namibia",
      "CABIN",
      "IN_PROGRESS",
      "CAA Inspector",
      "Due Date: 18 Jun 2026",
      "PKG-CAB-2026-001",
      "6",
      "Offline eligible",
    ]) {
      expect((await within(dossier).findAllByText(expected))[0]).toBeVisible();
    }
    expect(within(dossier).getByRole("link", { name: "Run Cabin checklist" })).toHaveAttribute(
      "href",
      "/inspector/audits/AUD-2026-001/checklist?packageId=PKG-CAB-2026-001",
    );
    expect(await screen.findByTestId("offline-readiness-panel")).toBeVisible();
  });

  it("encodes server-owned Audit and package identities in the checklist runner link", async () => {
    const runtime = createMockBackendRuntime();
    const backend = runtime.backendForRole("inspector");
    const baselineAssignments = await backend.assignments.list({});
    const baselinePackage = await backend.inspections.getPackage({ packageId: "PKG-CAB-2026-001" });
    const auditId = "inspection:assignment:plan-intake-42";
    const packageId = "inspection-package:inspection:assignment:plan-intake-42:1";
    vi.spyOn(backend.assignments, "list").mockResolvedValue({
      ...baselineAssignments,
      items: baselineAssignments.items.map((item, index) => index === 0
        ? { ...item, auditId, packageId }
        : item),
    });
    vi.spyOn(backend.inspections, "getPackage").mockResolvedValue({
      ...baselinePackage,
      id: packageId,
      auditId,
    });

    renderPage(runtime, `/inspector/audits/${encodeURIComponent(auditId)}`);

    expect(await screen.findByTestId("audit-id")).toHaveTextContent(auditId);
    expect(screen.getByRole("link", { name: "Run Cabin checklist" })).toHaveAttribute(
      "href",
      `/inspector/audits/${encodeURIComponent(auditId)}/checklist?packageId=${encodeURIComponent(packageId)}`,
    );
  });
});
