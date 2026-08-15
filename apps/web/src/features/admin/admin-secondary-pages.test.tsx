// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { ChecklistBuilderPage } from "./checklist-builder-page";
import { InspectionPackageAdminPage } from "./inspection-package-admin-page";

function renderAdmin(component: ReactNode) {
  const runtime = createMockBackendRuntime();
  render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: "demo",
      environmentLabel: "test",
      identityMode: "demo-role-switch",
      subjectId: "USR-ADMIN-ADA",
    }}>
      <ScenarioProvider>
        <MemoryRouter>{component}</MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
}

afterEach(() => cleanup());

describe("Admin approved catalog surfaces", () => {
  it("exposes the complete approved catalog browser without candidate governance controls", () => {
    renderAdmin(<ChecklistBuilderPage />);
    expect(screen.getByRole("heading", { name: "Approved AGA Catalog" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /load approved questions/i })).toBeDisabled();
    expect(screen.getByText(/does not create an approval, publication, or internal review workflow/i)).toBeInTheDocument();
    expect(screen.queryByText(/governed candidate|technical approval|bulk approval/i)).not.toBeInTheDocument();
  });

  it("keeps the historical inspection-package entry on the same read-only catalog boundary", () => {
    renderAdmin(<InspectionPackageAdminPage />);
    expect(screen.getByRole("heading", { name: "Approved AGA Catalog" })).toBeInTheDocument();
    expect(screen.queryByText(/Inspection Package Builder/i)).not.toBeInTheDocument();
  });
});
