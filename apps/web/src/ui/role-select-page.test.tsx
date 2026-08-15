// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { RoleSelectPage, ROLE_ENTRIES, createRoleEntryPath } from "./role-select-page";

afterEach(() => cleanup());

describe("RoleSelectPage", () => {
  it("renders all eight deterministic role cards with semantic icons and stable links", () => {
    render(
      <MemoryRouter>
        <RoleSelectPage mode="demo-role-switch" onRoleRequest={() => undefined} />
      </MemoryRouter>,
    );

    expect(screen.getByRole("heading", { name: "AviaSurveil360" })).toBeInTheDocument();
    expect(screen.getAllByRole("link")).toHaveLength(8);
    expect(screen.getAllByTestId("role-card-icon")).toHaveLength(8);
    expect(ROLE_ENTRIES.map(({ role, route }) => `${role}:${route}`)).toEqual([
      "inspector:inspector-assignments",
      "leadInspector:lead-review",
      "manager:dashboard",
      "gm:gm-dashboard",
      "finance:finance-review",
      "executiveDirector:executive-dashboard",
      "auditee:service-provider-cap",
      "admin:templates",
    ]);
    for (const entry of ROLE_ENTRIES) {
      const card = screen.getByRole("link", { name: new RegExp(entry.label) });
      expect(card).toHaveAttribute("href", createRoleEntryPath(entry.role));
      expect(card).toHaveAttribute("data-icon-key", entry.iconKey);
    }
  });

  it("uses a real link and reports the requested role when a card is activated", async () => {
    const user = userEvent.setup();
    const onRoleRequest = vi.fn();
    render(
      <MemoryRouter>
        <RoleSelectPage mode="canonical-test-role-switch" onRoleRequest={onRoleRequest} />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("link", { name: /CAA Inspector/i }));
    expect(onRoleRequest).toHaveBeenCalledWith("inspector");
  });

  it("moves keyboard focus to workspace selection from both discovery controls", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <RoleSelectPage mode="demo-role-switch" onRoleRequest={() => undefined} />
      </MemoryRouter>,
    );

    const workspaces = screen.getByRole("main");
    await user.click(screen.getByRole("button", { name: "Explore the local workspace" }));
    expect(workspaces).toHaveFocus();
    await user.click(screen.getByRole("button", { name: "Skip to workspace selection" }));
    expect(workspaces).toHaveFocus();
  });
});
