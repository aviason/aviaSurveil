// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { ApplicationShell, type ShellIdentityPresentation } from "./application-shell";

afterEach(() => {
  cleanup();
  Object.defineProperty(window, "innerWidth", { configurable: true, value: 1024 });
});

const identity: ShellIdentityPresentation = {
  mode: "demo-role-switch",
  displayName: "Asha Inspector",
  organizationLabel: "Namibia Civil Aviation Authority",
  activeRole: "inspector",
  availableRoles: ["inspector", "leadInspector", "manager"],
};

describe("ApplicationShell", () => {
  function LocationProbe() {
    return <output data-testid="location-probe">{useLocation().pathname}</output>;
  }
  it("composes candidate boundary, registry navigation, topbar, and supplied content", () => {
    render(
      <MemoryRouter>
        <ApplicationShell
          identity={identity}
          activeRouteId="audit-detail"
          onRoleRequest={() => undefined}
          onLogout={() => undefined}
          notificationState={{ kind: "local", unreadCount: 1, onOpen: () => undefined }}
        >
          <h1>Audit Detail</h1>
        </ApplicationShell>
      </MemoryRouter>,
    );

    expect(screen.getByTestId("application-shell")).toBeInTheDocument();
    expect(screen.getByText("Candidate-only")).toBeInTheDocument();
    expect(screen.getByText("No production-readiness claim")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "My Assignments" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("heading", { name: "Audit Detail" })).toBeInTheDocument();
  });

  it("supports mobile menu Escape close and focus restoration", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <ApplicationShell
          identity={identity}
          activeRouteId="inspector-home"
          onRoleRequest={() => undefined}
          onLogout={() => undefined}
          notificationState={{ kind: "local", unreadCount: 0, onOpen: () => undefined }}
        >
          <p>Assignments</p>
        </ApplicationShell>
      </MemoryRouter>,
    );

    const opener = screen.getByRole("button", { name: "Open navigation" });
    await user.click(opener);
    expect(screen.getByRole("dialog", { name: "Primary navigation" })).toBeInTheDocument();
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog", { name: "Primary navigation" })).not.toBeInTheDocument();
    expect(opener).toHaveFocus();
  });

  it("removes the collapsed desktop sidebar from mobile keyboard navigation", () => {
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 390 });
    const { container } = render(
      <MemoryRouter>
        <ApplicationShell
          identity={identity}
          activeRouteId="inspector-home"
          onRoleRequest={() => undefined}
          onLogout={() => undefined}
          notificationState={{ kind: "local", unreadCount: 0, onOpen: () => undefined }}
        >
          <p>Assignments</p>
        </ApplicationShell>
      </MemoryRouter>,
    );

    const collapsedSidebar = container.querySelector("aside.workspace-sidebar");
    expect(collapsedSidebar).toHaveAttribute("aria-hidden", "true");
    expect(collapsedSidebar).toHaveAttribute("inert");
  });

  it("exposes all Inspector workspace destinations from the mobile navigation drawer", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <ApplicationShell
          identity={identity}
          activeRouteId="inspector-findings"
          onRoleRequest={() => undefined}
          onLogout={() => undefined}
          notificationState={{ kind: "local", unreadCount: 0, onOpen: () => undefined }}
        >
          <h1>Findings</h1>
        </ApplicationShell>
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: "Open navigation" }));
    const drawer = screen.getByRole("dialog", { name: "Primary navigation" });
    for (const [name, href] of [
      ["My Assignments", "/inspector/inspector-assignments"],
      ["Findings", "/inspector/findings"],
      ["Messages", "/inspector/messages"],
      ["Calendar", "/inspector/calendar"],
      ["Reports", "/inspector/reports"],
    ] as const) {
      expect(within(drawer).getByRole("link", { name })).toHaveAttribute("href", href);
    }
  });

  it("transitions desktop and mobile Lead navigation to exact paths with one accessible active item", async () => {
    const user = userEvent.setup();
    const leadIdentity: ShellIdentityPresentation = {
      ...identity,
      activeRole: "leadInspector",
      displayName: "Caner Yildiz",
      availableRoles: ["leadInspector", "inspector"],
    };
    render(
      <MemoryRouter initialEntries={["/lead-inspector/preliminary-reports"]}>
        <LocationProbe />
        <ApplicationShell
          identity={leadIdentity}
          activeRouteId="lead-preliminary-reports"
          onRoleRequest={() => undefined}
          onLogout={() => undefined}
          notificationState={{ kind: "local", unreadCount: 0, onOpen: () => undefined }}
        ><h1>Preliminary Reports</h1></ApplicationShell>
      </MemoryRouter>,
    );

    const desktopNavigation = screen.getByRole("navigation", { name: "Primary role navigation" });
    expect(within(desktopNavigation).getAllByRole("link").filter((link) => link.hasAttribute("aria-current"))).toHaveLength(1);
    await user.click(within(desktopNavigation).getByRole("link", { name: "Final Reports" }));
    expect(screen.getByTestId("location-probe")).toHaveTextContent("/lead-inspector/final-reports");

    cleanup();
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 390 });
    render(
      <MemoryRouter initialEntries={["/lead-inspector/final-reports"]}>
        <LocationProbe />
        <ApplicationShell
          identity={leadIdentity}
          activeRouteId="lead-final-reports"
          onRoleRequest={() => undefined}
          onLogout={() => undefined}
          notificationState={{ kind: "local", unreadCount: 0, onOpen: () => undefined }}
        ><h1>Final Reports</h1></ApplicationShell>
      </MemoryRouter>,
    );
    await user.click(screen.getByRole("button", { name: "Open navigation" }));
    const drawer = screen.getByRole("dialog", { name: "Primary navigation" });
    const drawerNavigation = within(drawer).getByRole("navigation", { name: "Primary role navigation" });
    expect(within(drawerNavigation).getAllByRole("link").filter((link) => link.hasAttribute("aria-current"))).toHaveLength(1);
    await user.click(within(drawerNavigation).getByRole("link", { name: "Calendar" }));
    expect(screen.getByTestId("location-probe")).toHaveTextContent("/lead-inspector/calendar");
    expect(screen.queryByRole("dialog", { name: "Primary navigation" })).not.toBeInTheDocument();
  });

  it("wires visible role and logout controls to callbacks", async () => {
    const user = userEvent.setup();
    const onRoleRequest = vi.fn();
    const onLogout = vi.fn();
    render(
      <MemoryRouter>
        <ApplicationShell
          identity={identity}
          activeRouteId="inspector-home"
          onRoleRequest={onRoleRequest}
          onLogout={onLogout}
          notificationState={{ kind: "local", unreadCount: 0, onOpen: () => undefined }}
        >
          <p>Assignments</p>
        </ApplicationShell>
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: /Asha Inspector/i }));
    await user.click(screen.getByRole("button", { name: "Department Manager" }));
    await user.click(within(screen.getByRole("menu", { name: "Profile menu" })).getByRole("button", { name: "Logout" }));
    expect(onRoleRequest).toHaveBeenCalledWith("manager");
    expect(onLogout).toHaveBeenCalledOnce();
  });

  it("renders the accepted Service Provider Portal chrome only for the Auditee demo", () => {
    render(
      <MemoryRouter>
        <ApplicationShell
          identity={{
            ...identity,
            activeRole: "auditee",
            displayName: "Fly Namibia Quality Manager",
            organizationLabel: "Fly Namibia",
            availableRoles: ["auditee", "leadInspector"],
          }}
          activeRouteId="auditee-home"
          onRoleRequest={() => undefined}
          onLogout={() => undefined}
          notificationState={{ kind: "local", unreadCount: 1, onOpen: () => undefined }}
        >
          <h1>Corrective Actions (CAP)</h1>
        </ApplicationShell>
      </MemoryRouter>,
    );

    expect(screen.getByTestId("application-shell")).toHaveClass("workspace-shell--auditee-demo");
    expect(screen.getByRole("status")).toHaveTextContent("Frontend clickable prototype");
    expect(screen.getAllByText("Service Provider Portal").length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: "Role select" })).toBeEnabled();
  });

  it("renders the accepted Department Manager demo chrome without changing inspector chrome", () => {
    const { rerender } = render(
      <MemoryRouter>
        <ApplicationShell
          identity={{ ...identity, activeRole: "manager", displayName: "Mehmet Kaya" }}
          activeRouteId="manager-home"
          onRoleRequest={() => undefined}
          onLogout={() => undefined}
          notificationState={{ kind: "local", unreadCount: 1, onOpen: () => undefined }}
        ><h1>Department Manager Dashboard</h1></ApplicationShell>
      </MemoryRouter>,
    );

    expect(screen.getByTestId("application-shell")).toHaveClass("workspace-shell--manager-demo");
    expect(screen.getByText("OVERSIGHT WORKBENCH")).toBeVisible();
    expect(screen.getAllByText("Department Manager").length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: "Role select" })).toBeEnabled();

    rerender(
      <MemoryRouter>
        <ApplicationShell
          identity={identity}
          activeRouteId="inspector-home"
          onRoleRequest={() => undefined}
          onLogout={() => undefined}
          notificationState={{ kind: "local", unreadCount: 1, onOpen: () => undefined }}
        ><h1>My Assignments</h1></ApplicationShell>
      </MemoryRouter>,
    );
    expect(screen.getByTestId("application-shell")).not.toHaveClass("workspace-shell--manager");
  });

  it("renders the accepted Administration demo chrome and grouped sidebar", () => {
    render(
      <MemoryRouter>
        <ApplicationShell
          identity={{ ...identity, activeRole: "admin", displayName: "System Admin", availableRoles: ["admin", "manager"] }}
          activeRouteId="admin-home"
          onRoleRequest={() => undefined}
          onLogout={() => undefined}
          notificationState={{ kind: "local", unreadCount: 0, onOpen: () => undefined }}
        ><h1>Template Preview — Cabin Inspection</h1></ApplicationShell>
      </MemoryRouter>,
    );

    expect(screen.getByTestId("application-shell")).toHaveClass("workspace-shell--admin-demo");
    expect(screen.getByText("OVERSIGHT WORKBENCH")).toBeVisible();
    expect(screen.getByRole("link", { name: "Templates" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("button", { name: "Role select" })).toBeEnabled();
  });

  it("does not reserve demo ribbon chrome for an OIDC Administration session", () => {
    render(
      <MemoryRouter>
        <ApplicationShell
          identity={{ ...identity, mode: "oidc-session", activeRole: "admin", displayName: "System Admin", availableRoles: ["admin"] }}
          activeRouteId="admin-question-bank"
          onRoleRequest={() => undefined}
          onLogout={() => undefined}
          notificationState={{ kind: "local", unreadCount: 0, onOpen: () => undefined }}
        ><h1>Question Bank</h1></ApplicationShell>
      </MemoryRouter>,
    );

    expect(screen.getByTestId("application-shell")).toHaveClass("workspace-shell--admin");
    expect(screen.getByTestId("application-shell")).not.toHaveClass("workspace-shell--admin-demo");
    expect(screen.queryByText("DEMO")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Logout" })).toBeEnabled();
  });
});
