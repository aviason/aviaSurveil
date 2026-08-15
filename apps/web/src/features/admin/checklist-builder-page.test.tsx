// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { ChecklistBuilderPage } from "./checklist-builder-page";

describe("Checklist Builder approved catalog boundary", () => {
  it("renders the server-owned approved catalog browser without an intake workflow", async () => {
    const runtime = createMockBackendRuntime();
    render(
      <AppProviders runtime={{ backend: runtime.backend, backendForRole: runtime.backendForRole, buildProfile: "demo", environmentLabel: "test", identityMode: "demo-role-switch", subjectId: "USR-ADMIN-ADA" }}>
        <ScenarioProvider><MemoryRouter><ChecklistBuilderPage /></MemoryRouter></ScenarioProvider>
      </AppProviders>,
    );
    expect(screen.getByRole("heading", { name: "Approved AGA Catalog" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /load approved questions/i })).toBeDisabled();
    expect(screen.getByText(/does not create an approval, publication, or internal review workflow/i)).toBeInTheDocument();
  });
});
