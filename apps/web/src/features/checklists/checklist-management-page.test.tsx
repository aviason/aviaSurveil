// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { ChecklistManagementPage } from "./checklist-management-page";

describe("ChecklistManagementPage approved catalog boundary", () => {
  it("renders a read-only server-owned catalog register", () => {
    const runtime = createMockBackendRuntime();
    render(
      <AppProviders runtime={{ backend: runtime.backend, backendForRole: runtime.backendForRole, buildProfile: "demo", environmentLabel: "test", identityMode: "demo-role-switch", subjectId: "USR-MANAGER-NORA" }}>
        <ScenarioProvider><MemoryRouter><ChecklistManagementPage /></MemoryRouter></ScenarioProvider>
      </AppProviders>,
    );
    expect(screen.getByRole("heading", { name: "Checklist Management" })).toBeInTheDocument();
    expect(screen.getByText(/New Audit workflow selects the exact subset/i)).toBeInTheDocument();
    expect(screen.queryByText(/governed candidate|blocked-generation|publication reason/i)).toBeNull();
  });
});
