// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { InspectionPackageBuilderPage } from "./inspection-package-builder-page";

afterEach(cleanup);

describe("InspectionPackageBuilderPage", () => {
  it("persists the manager's draft revision while keeping runnable execution authority separate", async () => {
    const runtime = createMockBackendRuntime();
    render(<AppProviders runtime={{ backend: runtime.backend, backendForRole: runtime.backendForRole, buildProfile: "demo", environmentLabel: "test", identityMode: "demo-role-switch", subjectId: "USR-MANAGER-NORA" }}><ScenarioProvider><MemoryRouter><InspectionPackageBuilderPage /></MemoryRouter></ScenarioProvider></AppProviders>);
    const page = await screen.findByTestId("inspection-package-builder-page");
    const save = screen.getByRole("button", { name: "Save package draft" });
    const unavailable = screen.getByRole("button", { name: /Runnable checklist unavailable/ });
    expect(unavailable).toBeDisabled();
    expect(unavailable).toHaveAttribute("title", expect.stringMatching(/assigned CAA Inspector authority/));
    await userEvent.setup().clear(screen.getByLabelText("Risk focus"));
    await userEvent.setup().type(screen.getByLabelText("Risk focus"), "governed synthetic source change");
    await userEvent.setup().click(save);
    await waitFor(() => expect(screen.getByText(/Saved revision/)).toBeVisible());
    expect(page).toHaveTextContent("Expected Evidence");
  });
});
