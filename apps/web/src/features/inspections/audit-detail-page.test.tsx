// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";
import "fake-indexeddb/auto";

import { MemoryRouter } from "react-router-dom";
import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { AuditDetailPage } from "./audit-detail-page";

afterEach(cleanup);

function renderPage() {
  const runtime = createMockBackendRuntime();
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
        <MemoryRouter initialEntries={["/inspector/audits/AUD-2026-001"]}>
          <AuditDetailPage />
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
      "/inspector/audits/AUD-2026-001/checklist",
    );
    expect(await screen.findByTestId("offline-readiness-panel")).toBeVisible();
  });
});
