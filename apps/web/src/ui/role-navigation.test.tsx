// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";

import { REACT_ROUTE_CONTRACTS } from "../app/route-contracts";
import { RoleNavigation } from "./role-navigation";

afterEach(() => cleanup());

describe("RoleNavigation", () => {
  it.each(
    REACT_ROUTE_CONTRACTS
      .filter((route) => route.requiredRole)
      .map((route) => [route.id, route.requiredRole] as const),
  )("exposes exactly one accessible active primary item for %s", (routeId, requiredRole) => {
    render(
      <MemoryRouter>
        <RoleNavigation activeRole={requiredRole!} activeRouteId={routeId} />
      </MemoryRouter>,
    );

    const navigation = screen.getByRole("navigation", { name: "Primary role navigation" });
    expect(within(navigation).getAllByRole("link").filter((link) => link.hasAttribute("aria-current"))).toHaveLength(1);
  });

  it("derives primary navigation from the route registry in order", () => {
    render(
      <MemoryRouter>
        <RoleNavigation activeRole="manager" activeRouteId="manager-home" />
      </MemoryRouter>,
    );

    expect(screen.getAllByRole("link").map((link) => link.getAttribute("aria-label"))).toEqual([
      "Dashboard",
      "Planning",
      "Audits",
      "Reports Approval",
      "Risk Dashboard",
      "Inspection Team",
      "Findings Review",
      "CAP Monitoring",
      "Checklist Management",
    ]);
    expect(screen.getByText("Department Manager")).toBeVisible();
    expect(screen.getByRole("link", { name: "Dashboard" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Reports Approval" })).toHaveAttribute(
      "href",
      "/department-manager/preliminary-reports/:reportId",
    );
    expect(screen.getByRole("link", { name: "Risk Dashboard" })).toHaveAttribute(
      "href",
      "/department-manager/risk-dashboard",
    );
  });

  it("makes Manager Risk Dashboard reachable and the sole active desktop/mobile navigation item", () => {
    render(
      <MemoryRouter>
        <RoleNavigation activeRole="manager" activeRouteId="manager-risk-dashboard" />
      </MemoryRouter>,
    );

    const navigation = screen.getByRole("navigation", { name: "Primary role navigation" });
    expect(within(navigation).getByRole("link", { name: "Risk Dashboard" })).toHaveAttribute("aria-current", "page");
    expect(within(navigation).getAllByRole("link").filter((link) => link.hasAttribute("aria-current"))).toHaveLength(1);
  });

  it("maps Manager Inspection Evidence to its Findings Review parent with one active primary item", () => {
    render(
      <MemoryRouter>
        <RoleNavigation activeRole="manager" activeRouteId="evidence-review" />
      </MemoryRouter>,
    );

    const navigation = screen.getByRole("navigation", { name: "Primary role navigation" });
    expect(within(navigation).queryByRole("link", { name: "Evidence Review" })).toBeNull();
    expect(within(navigation).getByRole("link", { name: "Findings Review" })).toHaveAttribute("aria-current", "page");
    expect(within(navigation).getAllByRole("link").filter((link) => link.hasAttribute("aria-current"))).toHaveLength(1);
  });

  it("keeps contextual detail routes out of primary navigation while activating their parent", () => {
    render(
      <MemoryRouter>
        <RoleNavigation activeRole="inspector" activeRouteId="audit-detail" />
      </MemoryRouter>,
    );

    expect(screen.getAllByRole("link").map((link) => link.getAttribute("aria-label"))).toEqual([
      "My Assignments",
      "Findings",
      "Messages",
      "Calendar",
      "Reports",
    ]);
    expect(screen.queryByRole("link", { name: "Audit Detail" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "My Assignments" })).toHaveAttribute("aria-current", "page");
  });

  it("makes every primary Inspector workspace route reachable with its declared path", () => {
    render(
      <MemoryRouter>
        <RoleNavigation activeRole="inspector" activeRouteId="inspector-findings" />
      </MemoryRouter>,
    );

    expect(screen.getByRole("link", { name: "Findings" })).toHaveAttribute("href", "/inspector/findings");
    expect(screen.getByRole("link", { name: "Messages" })).toHaveAttribute("href", "/inspector/messages");
    expect(screen.getByRole("link", { name: "Calendar" })).toHaveAttribute("href", "/inspector/calendar");
    expect(screen.getByRole("link", { name: "Reports" })).toHaveAttribute("href", "/inspector/reports");
    expect(screen.getByRole("link", { name: "Findings" })).toHaveAttribute("aria-current", "page");
  });

  it("maps the contextual Inspector Profile to one accessible active primary item", () => {
    render(
      <MemoryRouter>
        <RoleNavigation activeRole="inspector" activeRouteId="inspector-profile" />
      </MemoryRouter>,
    );

    expect(screen.getAllByRole("link").filter((link) => link.hasAttribute("aria-current"))).toHaveLength(1);
    expect(screen.getByRole("link", { name: "My Assignments" })).toHaveAttribute("aria-current", "page");
  });

  it("does not link Lead Inspector navigation to the Manager-owned evidence route", () => {
    render(
      <MemoryRouter>
        <RoleNavigation activeRole="leadInspector" activeRouteId="lead-home" />
      </MemoryRouter>,
    );

    expect(screen.queryByRole("link", { name: "Evidence Review" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Evidence Review/ })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Assigned Audits" })).toHaveAttribute("href", "/lead-inspector/lead-review");
  });

  it("gives every Lead primary destination its exact path and one accessible active item", () => {
    const expected = [
      ["Preliminary Reports", "lead-preliminary-reports", "/lead-inspector/preliminary-reports"],
      ["Final Reports", "lead-final-reports", "/lead-inspector/final-reports"],
      ["Calendar", "lead-calendar", "/lead-inspector/calendar"],
      ["Messages", "lead-messages", "/lead-inspector/messages"],
      ["Analytics & Reports", "lead-analytics-reports", "/lead-inspector/analytics-reports"],
      ["Settings", "lead-settings", "/lead-inspector/settings"],
    ] as const;

    for (const [name, routeId, href] of expected) {
      const { unmount } = render(
        <MemoryRouter>
          <RoleNavigation activeRole="leadInspector" activeRouteId={routeId} />
        </MemoryRouter>,
      );
      const navigation = screen.getByRole("navigation", { name: "Primary role navigation" });
      expect(within(navigation).getByRole("link", { name })).toHaveAttribute("href", href);
      expect(within(navigation).getAllByRole("link").filter((link) => link.hasAttribute("aria-current"))).toHaveLength(1);
      expect(within(navigation).getByRole("link", { name })).toHaveAttribute("aria-current", "page");
      unmount();
    }
  });

  it("labels the Auditee navigation as the Service Provider Portal", () => {
    render(
      <MemoryRouter>
        <RoleNavigation activeRole="auditee" activeRouteId="auditee-home" />
      </MemoryRouter>,
    );

    expect(screen.getByText("Service Provider Portal")).toBeVisible();
    expect(screen.getByRole("link", { name: "Corrective Actions (CAP)" })).toHaveAttribute("aria-current", "page");
  });

  it("reproduces the grouped Administration navigation with Templates active", () => {
    render(
      <MemoryRouter>
        <RoleNavigation activeRole="admin" activeRouteId="admin-home" />
      </MemoryRouter>,
    );

    expect(screen.getAllByText("Administration")).toHaveLength(2);
    expect(screen.getByText("Regulations")).toBeVisible();
    expect(screen.getByText("Evidence & Documents")).toBeVisible();
    expect(screen.getByRole("link", { name: "Templates" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Organisation Master Data" })).toHaveAttribute("href", "/admin/organization-master-data");
    expect(screen.getByRole("link", { name: "Audit Log" })).toHaveAttribute("href", "/admin/audit-log");
  });
});
