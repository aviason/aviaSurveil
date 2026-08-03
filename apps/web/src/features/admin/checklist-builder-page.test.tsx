// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { ChecklistBuilderPage } from "./checklist-builder-page";

describe("Checklist Builder governed intake boundary", () => {
  it("renders the candidate-only intake panel with an explicit external blocker", () => {
    const runtime = createMockBackendRuntime();
    render(
      <AppProviders runtime={{ backend: runtime.backend, backendForRole: runtime.backendForRole, buildProfile: "demo", environmentLabel: "test", identityMode: "demo-role-switch", subjectId: "USR-ADMIN-ADA" }}>
        <ScenarioProvider><MemoryRouter><ChecklistBuilderPage /></MemoryRouter></ScenarioProvider>
      </AppProviders>,
    );
    expect(screen.getByTestId("checklist-intake-panel")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /receive candidate-only archive/i })).toBeDisabled();
    expect(screen.getByText(/external, read-only dependency/i)).toBeInTheDocument();
    expect(screen.queryByTestId("aga-candidate-demo-panel")).not.toBeInTheDocument();
  });
});
