// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";

import { AppProviders } from "../../app/providers";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { OrganizationRegistryPage } from "./organization-registry-page";

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
        subjectId: "USR-MANAGER-MEHMET",
      }}
    >
      <MemoryRouter initialEntries={["/department-manager/organizations"]}>
        <OrganizationRegistryPage />
      </MemoryRouter>
    </AppProviders>,
  );
}

describe("OrganizationRegistryPage", () => {
  it("renders an authorized dense register and equivalent responsive records", async () => {
    renderPage();

    const register = await screen.findByRole("table", { name: "Organizations" });
    for (const column of ["Organization", "Type", "Open Findings", "Status", "Last Audit", "Next Audit", "Action"]) {
      expect(within(register).getByRole("columnheader", { name: column })).toBeVisible();
    }
    expect(within(register).getByText("Fly Namibia")).toBeVisible();
    expect(within(register).getByText("SkyCargo Air")).toBeVisible();

    const cards = screen.getAllByTestId("organization-mobile-record");
    expect(cards).toHaveLength(2);
    expect(within(cards[0]).getByText("Open Findings")).toBeVisible();
    expect(within(cards[0]).getByText("Next Audit")).toBeVisible();
  });

  it("opens a local read-only dossier without implying edit or create authority", async () => {
    renderPage();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: "Open Fly Namibia" }));
    const dossier = screen.getByTestId("organization-dossier");
    expect(within(dossier).getByRole("heading", { name: "Fly Namibia" })).toBeVisible();
    expect(within(dossier).getByText("ACTIVE")).toBeVisible();
    expect(screen.queryByRole("button", { name: /edit|create|delete/i })).toBeNull();
  });

  it("links each server-owned organization to its exact child route without a fixture-specific fallback", async () => {
    renderPage();

    const register = await screen.findByRole("table", { name: "Organizations" });
    expect(within(register).getByRole("link", { name: "Open organization ORG-FLY-NAMIBIA" })).toHaveAttribute(
      "href",
      "/department-manager/organizations/ORG-FLY-NAMIBIA",
    );
    expect(within(register).getByRole("link", { name: "Open organization ORG-SKYCARGO" })).toHaveAttribute(
      "href",
      "/department-manager/organizations/ORG-SKYCARGO",
    );
  });
});
