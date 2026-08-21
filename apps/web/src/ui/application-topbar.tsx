import { useState } from "react";

import type { Role } from "../backend/backend";
import type { ReactSurfaceId } from "../app/route-contracts";
import type { RoleSelectionMode } from "./role-select-page";
import { roleLabel } from "./role-select-page";

export interface ShellIdentityPresentation {
  mode: RoleSelectionMode;
  displayName: string;
  organizationLabel: string;
  activeRole: Role;
  availableRoles: readonly Role[];
}

export type NotificationState =
  | { kind: "local"; unreadCount: number; onOpen(): void }
  | { kind: "unavailable"; reason: string };

function NotificationIcon() {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24">
      <path d="M18 9a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9" />
      <path d="M10 21h4" />
    </svg>
  );
}

function initials(name: string): string {
  return name.split(/\s+/).map((part) => part[0] ?? "").join("").slice(0, 2).toUpperCase();
}

export function ApplicationTopbar({
  identity,
  onRoleRequest,
  onLogout,
  notificationState,
  activeRouteId,
}: {
  identity: ShellIdentityPresentation;
  onRoleRequest(role: Role): void;
  onLogout(): void;
  notificationState: NotificationState;
  activeRouteId?: ReactSurfaceId;
}) {
  const [profileOpen, setProfileOpen] = useState(false);
  const [notificationsOpen, setNotificationsOpen] = useState(false);
  const [helpOpen, setHelpOpen] = useState(false);
  const auditeeChrome = identity.activeRole === "auditee";
  const managerChrome = identity.activeRole === "manager";
  const authorityChrome = ["gm", "finance", "executiveDirector"].includes(identity.activeRole);
  const adminChrome = identity.activeRole === "admin";
  const avatarInitials = adminChrome ? "AP" : initials(identity.displayName);
  if (auditeeChrome || managerChrome || authorityChrome || adminChrome) {
    const managerCrumbs: Partial<Record<ReactSurfaceId, string>> = {
      "manager-home": "Dashboard",
      "organization-registry": "Dashboard  ›  Organizations",
      "audit-plan": "Dashboard  ›  Planning",
      "new-audit-wizard-1": "Planning  ›  New Audit",
      "new-audit-wizard-2": "Planning  ›  New Audit",
      "new-audit-wizard-3": "Planning  ›  New Audit",
      "new-audit-wizard-4": "Planning  ›  New Audit",
      "new-audit-wizard-5": "Planning  ›  New Audit",
      "report-preview": "Dashboard  ›  Reports Approval",
    };
    const authorityCrumbs: Partial<Record<ReactSurfaceId, string>> = { "gm-home": "General Manager Dashboard", "finance-home": "Finance Review", "executive-home": "Executive Director Dashboard" };
    const auditeeCrumbs: Partial<Record<ReactSurfaceId, string>> = {
      "auditee-home": "Corrective Actions (CAP)",
      "auditee-inspection-coordination": "Inspection Coordination",
      "auditee-preliminary-reports": "Preliminary Reports",
      "auditee-final-reports": "Final Reports",
      "auditee-report-preview": "Final Reports  ›  Report Preview",
      "auditee-messages": "Messages",
      "auditee-documents": "Documents",
      "auditee-settings": "Settings",
    };
    const adminCrumbs: Partial<Record<ReactSurfaceId, string>> = {
      "admin-regulatory-library": "Regulatory Library",
      "admin-template-list": "Checklist Templates",
      "admin-home": "Templates  ›  Template Preview — Cabin Inspection",
      "admin-question-bank": "Question Bank",
      "admin-checklist-builder": "Checklist Builder",
      "admin-version-history": "Version History",
      "admin-inspection-package-builder": "Checklist Builder  ›  Inspection Package Builder",
      "admin-reports": "Admin Reports",
      "admin-users-roles": "Users / Roles",
      "admin-configurations": "Configurations",
      "admin-organization-master-data": "Organisation Master Data",
      "admin-organization-detail": "Organisation Master Data  ›  Organization Detail",
      "admin-audit-log": "Audit Log",
    };
    const routeCrumbs = auditeeChrome
      ? auditeeCrumbs[activeRouteId ?? "auditee-home"] ?? "Service Provider Portal"
      : managerChrome
        ? managerCrumbs[activeRouteId ?? "manager-home"] ?? "Dashboard"
        : adminChrome
          ? adminCrumbs[activeRouteId ?? "admin-template-list"] ?? "Administration"
          : authorityCrumbs[activeRouteId ?? "gm-home"] ?? roleLabel(identity.activeRole);
    return (
      <header className={`application-topbar application-topbar--root ${auditeeChrome ? "application-topbar--auditee auditee-root-topbar" : managerChrome ? "application-topbar--manager manager-root-topbar" : adminChrome ? "application-topbar--admin admin-root-topbar" : "application-topbar--authority authority-root-topbar"}`}>
        <div className="auditee-root-topbar__crumbs"><b>{routeCrumbs}</b></div>
        <div className="auditee-root-topbar__spacer" />
        {identity.mode !== "oidc-session" ? <label className="auditee-root-topbar__experience">
          <span>Experience</span>
          <select aria-label="Experience" onChange={(event) => onRoleRequest(event.target.value as Role)} value={identity.activeRole}>
            {identity.availableRoles.map((role) => <option key={role} value={role}>{role === "auditee" ? "Service Provider Portal - Service Provider" : role === "admin" ? "Administration" : roleLabel(role)}</option>)}
          </select>
        </label> : null}
        <div className="auditee-root-topbar__notification">
          {notificationState.kind === "local" ? (
            <button
              aria-expanded={notificationsOpen}
              aria-label="Notifications"
              className="auditee-root-topbar__icon"
              onClick={() => {
                notificationState.onOpen();
                setNotificationsOpen((value) => !value);
              }}
              type="button"
            >
              <NotificationIcon />
              {notificationState.unreadCount ? <span className="auditee-root-topbar__badge">{notificationState.unreadCount}</span> : null}
            </button>
          ) : (
            <button
              aria-label={`Notifications unavailable: ${notificationState.reason}`}
              className="auditee-root-topbar__icon"
              disabled
              title={notificationState.reason}
              type="button"
            >
              <NotificationIcon />
            </button>
          )}
          {notificationsOpen ? <p className="topbar-popover" role="status">{notificationState.kind === "local" ? `${notificationState.unreadCount} local notification updates` : notificationState.reason}</p> : null}
        </div>
        <div className="auditee-root-topbar__who">
          <span className={`auditee-root-topbar__avatar is-${identity.activeRole}`}>{avatarInitials}</span>
          <span className="auditee-root-topbar__identity">
            <strong>{identity.displayName}</strong>
            <small>{auditeeChrome ? `Service Provider Portal · ${identity.organizationLabel}` : adminChrome ? "Administration" : roleLabel(identity.activeRole)}</small>
          </span>
        </div>
      </header>
    );
  }
  return (
    <header className="application-topbar">
      <button
        aria-expanded={helpOpen}
        aria-label="Help"
        className="topbar-icon-button"
        onClick={() => setHelpOpen((value) => !value)}
        type="button"
      >
        ?
      </button>
      {helpOpen ? <p className="topbar-popover" role="status">Workspace help is available in this workbench.</p> : null}
      <div className="topbar-control">
        {notificationState.kind === "local" ? (
          <button
            aria-expanded={notificationsOpen}
            aria-label="Notifications"
            className="topbar-icon-button"
            type="button"
            onClick={() => {
              notificationState.onOpen();
              setNotificationsOpen((value) => !value);
            }}
          >
            <NotificationIcon />
            {notificationState.unreadCount ? <span className="topbar-notification-dot">{auditeeChrome ? notificationState.unreadCount : null}</span> : null}
          </button>
        ) : (
          <>
            <button
              aria-label={`Notifications unavailable: ${notificationState.reason}`}
              aria-describedby="topbar-notification-unavailable"
              className="topbar-icon-button"
              disabled
              title={notificationState.reason}
              type="button"
            >
              <NotificationIcon />
            </button>
            <span className="topbar-unavailable-reason" id="topbar-notification-unavailable">{notificationState.reason}</span>
          </>
        )}
        {notificationsOpen ? <p className="topbar-popover" role="status">{notificationState.kind === "local" ? `${notificationState.unreadCount} local notification updates` : notificationState.reason}</p> : null}
      </div>
      <div className="topbar-control">
        <button
          aria-expanded={profileOpen}
          className="topbar-profile"
          onClick={() => setProfileOpen((value) => !value)}
          type="button"
        >
          <span className="topbar-avatar">{initials(identity.displayName)}</span>
          {auditeeChrome ? (
            <span className="topbar-profile__identity">
              <strong>{identity.displayName}</strong>
              <small>Service Provider Portal · {identity.organizationLabel}</small>
            </span>
          ) : <span>{identity.displayName}</span>}
          <span className="topbar-profile__chevron" aria-hidden="true">⌄</span>
        </button>
        {identity.mode === "oidc-session" ? null : (
          <a
            aria-label="Switch role"
            className="topbar-switch-role"
            href="/"
            onClick={(event) => {
              event.preventDefault();
              onLogout();
            }}
            title="Switch role"
          />
        )}
        {profileOpen ? (
          <div className="topbar-profile-menu" role="menu" aria-label="Profile menu">
            <p>{identity.organizationLabel}</p>
            <p>{roleLabel(identity.activeRole)}</p>
            {identity.mode !== "oidc-session" ? identity.availableRoles.filter((role) => role !== identity.activeRole).map((role) => (
              <button key={role} onClick={() => onRoleRequest(role)} type="button">{roleLabel(role)}</button>
            )) : null}
            <button onClick={onLogout} type="button">Logout</button>
          </div>
        ) : null}
      </div>
    </header>
  );
}
