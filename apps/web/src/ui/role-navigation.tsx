import { Fragment, type ReactNode } from "react";
import { Link } from "react-router-dom";

import type { Role } from "../backend/backend";
import {
  REACT_ROUTE_CONTRACTS,
  REACT_ROUTE_CONTRACT_BY_ID,
  type ReactSurfaceId,
  type RouteContract,
} from "../app/route-contracts";

export function activePrimaryRouteId(activeRouteId: ReactSurfaceId): ReactSurfaceId {
  if (
    activeRouteId === "finding-detail" ||
    activeRouteId === "inspector-assistant"
  ) return "inspector-findings";
  if (activeRouteId === "closure-report-preview") return "inspector-reports";
  if (activeRouteId === "organization-risk-profile") return "manager-risk-dashboard";
  if (
    activeRouteId === "organization-registry" ||
    activeRouteId === "organization-detail"
  ) return "manager-home";
  if (activeRouteId === "executive-report-preview") return "executive-final-reports";
  if (activeRouteId === "admin-home") return "admin-template-list";
  if (activeRouteId === "admin-inspection-package-builder") return "admin-checklist-builder";
  if (activeRouteId === "admin-organization-detail") return "admin-organization-master-data";
  if (activeRouteId === "report-preview") return activeRouteId;
  let current: ReactSurfaceId = activeRouteId;
  const visited = new Set<ReactSurfaceId>();
  while (!visited.has(current)) {
    visited.add(current);
    const parentId: ReactSurfaceId | null =
      REACT_ROUTE_CONTRACT_BY_ID.get(current)?.parentId ?? null;
    if (!parentId) return current;
    current = parentId;
  }
  return activeRouteId;
}

export function primaryRoutesForRole(role: Role): readonly RouteContract[] {
  return REACT_ROUTE_CONTRACTS
    .filter((contract) => contract.placement === "primary" && contract.requiredRole === role)
    .sort((left, right) => left.order - right.order);
}

type NavigationIcon =
  | "dashboard"
  | "assignment"
  | "cap"
  | "finding"
  | "evidence"
  | "message"
  | "calendar"
  | "report"
  | "analytics"
  | "settings"
  | "settingsRoot"
  | "shield"
  | "planning"
  | "organization"
  | "users"
  | "help"
  | "history"
  | "listCheck"
  | "building";

interface AcceptedNavigationItem {
  label: string;
  icon: NavigationIcon;
  routeId?: ReactSurfaceId;
  targetRouteId?: ReactSurfaceId;
  badge?: string;
  section?: string;
}

const ACCEPTED_NAVIGATION: Readonly<Record<Role, readonly AcceptedNavigationItem[]>> = {
  inspector: [
    { label: "Dashboard", icon: "dashboard" },
    { label: "My Assignments", icon: "assignment", routeId: "inspector-home", badge: "8" },
    { label: "Findings", icon: "finding", routeId: "inspector-findings", badge: "14" },
    { label: "Evidence Review", icon: "evidence", badge: "3" },
    { label: "Messages", icon: "message", routeId: "inspector-messages", badge: "2" },
    { label: "Calendar", icon: "calendar", routeId: "inspector-calendar", badge: "2" },
    { label: "Reports", icon: "report", routeId: "inspector-reports" },
  ],
  leadInspector: [
    { label: "Assigned Audits", icon: "assignment", routeId: "lead-home" },
    { label: "Evidence Review", icon: "evidence" },
    { label: "Preliminary Reports", icon: "report", routeId: "lead-preliminary-reports", badge: "6" },
    { label: "Final Reports", icon: "report", routeId: "lead-final-reports", badge: "4" },
    { label: "Calendar", icon: "calendar", routeId: "lead-calendar", badge: "5" },
    { label: "Messages", icon: "message", routeId: "lead-messages" },
    { label: "Analytics & Reports", icon: "analytics", routeId: "lead-analytics-reports" },
    { label: "Settings", icon: "settings", routeId: "lead-settings" },
  ],
  manager: [
    { label: "Dashboard", icon: "dashboard", routeId: "manager-home" },
    { label: "Planning", icon: "planning", routeId: "audit-plan" },
    { label: "Audits", icon: "assignment", routeId: "manager-audits" },
    { label: "Reports Approval", icon: "report", routeId: "report-preview", targetRouteId: "manager-preliminary-report-review", badge: "2" },
    { label: "Risk Dashboard", icon: "analytics", routeId: "manager-risk-dashboard" },
    { label: "Inspection Team", icon: "assignment", routeId: "manager-inspection-team" },
    { label: "Findings Review", icon: "finding", routeId: "manager-findings-review" },
    { label: "CAP Monitoring", icon: "evidence", routeId: "manager-cap-monitoring" },
    { label: "Checklist Management", icon: "settings", routeId: "manager-checklist-management" },
  ],
  gm: [
    { label: "Dashboard", icon: "dashboard", routeId: "gm-home" },
    { label: "Planning", icon: "planning", routeId: "gm-planning" },
    { label: "Report Approvals", icon: "report", routeId: "gm-report-approvals" },
    { label: "Departments", icon: "dashboard", routeId: "gm-departments" },
    { label: "Risk Dashboard", icon: "shield", routeId: "gm-risk-dashboard" },
    { label: "Settings", icon: "settings", routeId: "gm-settings" },
  ],
  finance: [{ label: "Finance Review", icon: "dashboard", routeId: "finance-home" }],
  executiveDirector: [
    { label: "Dashboard", icon: "dashboard", routeId: "executive-home" },
    { label: "Planning", icon: "planning", routeId: "executive-planning" },
    { label: "Preliminary Reports", icon: "report", routeId: "executive-preliminary-reports" },
    { label: "Final Reports", icon: "report", routeId: "executive-final-reports" },
    { label: "Notifications", icon: "dashboard", routeId: "executive-notifications" },
    { label: "Settings", icon: "settings", routeId: "executive-settings" },
  ],
  auditee: [
    { label: "Inspection Coordination", icon: "assignment", routeId: "auditee-inspection-coordination", badge: "1" },
    { label: "Corrective Actions (CAP)", icon: "cap", routeId: "auditee-home", badge: "4" },
    { label: "Preliminary Reports", icon: "report", routeId: "auditee-preliminary-reports" },
    { label: "Final Reports", icon: "report", routeId: "auditee-final-reports" },
    { label: "Messages", icon: "message", routeId: "auditee-messages", badge: "1" },
    { label: "Documents", icon: "evidence", routeId: "auditee-documents" },
    { label: "Settings", icon: "settingsRoot", routeId: "auditee-settings" },
  ],
  admin: [
    { label: "Regulatory Library", icon: "dashboard", routeId: "admin-regulatory-library", section: "Regulations" },
    { label: "Templates", icon: "listCheck", routeId: "admin-template-list", section: "Evidence & Documents" },
    { label: "Checklist Builder", icon: "listCheck", routeId: "admin-checklist-builder" },
    { label: "Question Bank", icon: "help", routeId: "admin-question-bank" },
    { label: "Version History", icon: "history", routeId: "admin-version-history" },
    { label: "Reports", icon: "report", routeId: "admin-reports" },
    { label: "Users / Roles", icon: "users", routeId: "admin-users-roles", section: "Administration" },
    { label: "Configurations", icon: "settingsRoot", routeId: "admin-configurations" },
    { label: "Organisation Master Data", icon: "building", routeId: "admin-organization-master-data" },
    { label: "Audit Log", icon: "history", routeId: "admin-audit-log" },
  ],
};

function NavigationGlyph({ icon }: { icon: NavigationIcon }) {
  const paths: Readonly<Record<NavigationIcon, ReactNode>> = {
    dashboard: <><path d="M4 5h7v6H4z" /><path d="M13 5h7v4h-7z" /><path d="M13 11h7v8h-7z" /><path d="M4 13h7v6H4z" /></>,
    assignment: <><path d="M9 5h6" /><path d="M9 3h6l1 2h3v16H5V5h3l1-2z" /><path d="M9 11h6" /><path d="M9 15h4" /></>,
    cap: <><path d="M9 5h6" /><path d="M9 3h6l1 2h3v16H5V5h3l1-2z" /><path d="m8.5 14.5 2 2 5-5" /></>,
    finding: <><path d="M6 3h8l4 4v5" /><path d="M14 3v5h5" /><path d="M6 3v18h7" /><path d="M9 11h4" /><path d="M9 15h2" /><path d="m17.5 19.5 3 3" /><path d="M16 19a4 4 0 1 0 0-8 4 4 0 0 0 0 8z" /></>,
    evidence: <path d="m21 11-9 9a6 6 0 0 1-8.5-8.5l9.5-9.5a4 4 0 0 1 5.7 5.7l-9.5 9.5a2 2 0 0 1-2.8-2.8l8.7-8.7" />,
    message: <><path d="M4 6h16v12H4z" /><path d="m4 7 8 6 8-6" /></>,
    calendar: <><path d="M7 3v4" /><path d="M17 3v4" /><path d="M4 8h16" /><path d="M5 5h14v16H5z" /></>,
    report: <><path d="M14 3H7v18h10V8z" /><path d="M14 3v5h5" /><path d="M9 13h6" /><path d="M9 17h4" /></>,
    analytics: <><path d="M4 19V5" /><path d="M4 19h16" /><path d="M8 16v-5" /><path d="M12 16V8" /><path d="M16 16v-8" /></>,
    settings: <><circle cx="12" cy="12" r="3" /><path d="M19 13.5v-3l-2-.7-.8-1.9.9-1.9-2.1-2.1-1.9.9-1.9-.8-.7-2h-3l-.7 2-1.9.8-1.9-.9L3.9 6l.9 1.9L4 9.8l-2 .7v3l2 .7.8 1.9-.9 1.9L6 20.1l1.9-.9 1.9.8.7 2h3l.7-2 1.9-.8 1.9.9 2.1-2.1-.9-1.9.8-1.9z" /></>,
    settingsRoot: <><path d="M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z" /><path d="M19.4 15a8 8 0 0 0 .1-1.5l2-1.5-2-3.5-2.4 1a8 8 0 0 0-1.3-.8L15.5 6h-7l-.3 2.7a8 8 0 0 0-1.3.8l-2.4-1-2 3.5 2 1.5A8 8 0 0 0 4.6 15l-2 1.5 2 3.5 2.4-1a8 8 0 0 0 1.3.8l.3 2.7h7l.3-2.7a8 8 0 0 0 1.3-.8l2.4 1 2-3.5z" /></>,
    shield: <><path d="M12 3 19 6v5c0 4.6-2.8 8-7 10-4.2-2-7-5.4-7-10V6z" /><path d="M12 7v9" /></>,
    planning: <><path d="M5 4h14v17H5z" /><path d="M8 2v5" /><path d="M16 2v5" /><path d="M8 11h8" /><path d="M8 15h6" /></>,
    organization: <><path d="M4 21V7l8-4 8 4v14" /><path d="M8 10h2" /><path d="M14 10h2" /><path d="M8 14h2" /><path d="M14 14h2" /><path d="M10 21v-3h4v3" /></>,
    users: <><path d="M16 11a3 3 0 1 0 0-6" /><path d="M8 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6z" /><path d="M2 19a6 6 0 0 1 12 0" /><path d="M14 15a5 5 0 0 1 8 4" /></>,
    help: <><path d="M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18z" /><path d="M9.5 9a2.5 2.5 0 1 1 4.2 1.8c-.9.8-1.7 1.2-1.7 2.7" /><path d="M12 17h.01" /></>,
    history: <><path d="M3 12a9 9 0 1 0 3-6.7" /><path d="M3 5v6h6" /><path d="M12 7v5l3 2" /></>,
    listCheck: <><path d="m4 7 2 2 3-4" /><path d="M11 7h9" /><path d="m4 15 2 2 3-4" /><path d="M11 15h9" /></>,
    building: <><path d="M5 21V3h10v18" /><path d="M15 9h4v12" /><path d="M8 7h3" /><path d="M8 11h3" /><path d="M8 15h3" /><path d="M3 21h18" /></>,
  };
  return <span className="nav-item__icon" aria-hidden="true"><svg viewBox="0 0 24 24">{paths[icon]}</svg></span>;
}

export function RoleNavigation({
  activeRole,
  activeRouteId,
  onNavigate,
}: {
  activeRole: Role;
  activeRouteId: ReactSurfaceId;
  onNavigate?: () => void;
}) {
  const activePrimary = activePrimaryRouteId(activeRouteId);
  return (
    <nav className="role-navigation" aria-label="Primary role navigation">
      {activeRole === "auditee" ? <p className="role-navigation__experience">Service Provider Portal</p> : null}
      {activeRole === "manager" ? <p className="role-navigation__experience">Department Manager</p> : null}
      {activeRole === "gm" ? <p className="role-navigation__experience">General Manager</p> : null}
      {activeRole === "finance" ? <p className="role-navigation__experience">Finance Review</p> : null}
      {activeRole === "executiveDirector" ? <p className="role-navigation__experience">Executive Director</p> : null}
      {activeRole === "admin" ? <p className="role-navigation__experience">Administration</p> : null}
      {ACCEPTED_NAVIGATION[activeRole].map((item) => {
        const route = item.routeId ? REACT_ROUTE_CONTRACT_BY_ID.get(item.routeId) : null;
        const targetRoute = item.targetRouteId ? REACT_ROUTE_CONTRACT_BY_ID.get(item.targetRouteId) : route;
        const active =
          item.routeId === activePrimary ||
          (activeRole === "inspector" && activeRouteId === "audit-plan" && item.label === "Calendar");
        const visuallyActive = active && activeRouteId !== "inspector-profile";
        const content = <><NavigationGlyph icon={item.icon} /><span>{item.label}</span>{item.badge ? <span className="nav-item__badge">{item.badge}</span> : null}</>;
        return (
          <Fragment key={item.label}>
            {item.section ? <p className="role-navigation__section">{item.section}</p> : null}
            {route ? (
              <Link aria-label={item.label} className={`nav-item${visuallyActive ? " active" : ""}`} to={targetRoute?.path ?? route.path} aria-current={active ? "page" : undefined} onClick={onNavigate}>
                {content}
              </Link>
            ) : (
              <button
                aria-label={`${item.label} unavailable: this screen remains in the accepted legacy demo`}
                className={`nav-item${active ? " active" : ""}`}
                disabled
                title="This screen remains in the accepted legacy demo."
                type="button"
              >
                {content}
              </button>
            )}
          </Fragment>
        );
      })}
    </nav>
  );
}
