import type { PropsWithChildren } from "react";

import { DEMO_MOCK_STORAGE_KEY } from "../app/demo-persistence";
import type { Role } from "../backend/backend";
import type { ReactSurfaceId } from "../app/route-contracts";
import { ApplicationTopbar, type NotificationState, type ShellIdentityPresentation } from "./application-topbar";
import { CandidateBoundary } from "./candidate-boundary";
import { MobileNavigation } from "./mobile-navigation";
import { RoleNavigation } from "./role-navigation";

export type { NotificationState, ShellIdentityPresentation } from "./application-topbar";

export interface ApplicationShellProps {
  identity: ShellIdentityPresentation;
  activeRouteId: ReactSurfaceId;
  onRoleRequest(role: Role): void;
  onLogout(): void;
  notificationState: NotificationState;
  environmentLabel?: string;
}

export function ApplicationShell({
  identity,
  activeRouteId,
  onRoleRequest,
  onLogout,
  notificationState,
  environmentLabel = "Deterministic mock data",
  children,
}: PropsWithChildren<ApplicationShellProps>) {
  const auditeeChrome = identity.activeRole === "auditee";
  const managerChrome = identity.activeRole === "manager";
  const authorityChrome = ["gm", "finance", "executiveDirector"].includes(identity.activeRole);
  const adminChrome = identity.activeRole === "admin";
  const auditeeDemoChrome = auditeeChrome && identity.mode === "demo-role-switch";
  const managerDemoChrome = managerChrome && identity.mode === "demo-role-switch";
  const authorityDemoChrome = authorityChrome && identity.mode === "demo-role-switch";
  const adminDemoChrome = adminChrome && identity.mode === "demo-role-switch";
  const rootDemoChrome = auditeeDemoChrome || managerDemoChrome || authorityDemoChrome || adminDemoChrome;
  const compactNavigation = typeof window !== "undefined" && window.innerWidth <= 900;
  return (
    <main
      className={`workspace-shell${auditeeChrome ? " workspace-shell--auditee" : ""}${managerChrome ? " workspace-shell--manager" : ""}${authorityChrome ? ` workspace-shell--authority workspace-shell--${identity.activeRole}` : ""}${adminChrome ? " workspace-shell--admin" : ""}${auditeeDemoChrome ? " workspace-shell--auditee-demo" : ""}${managerDemoChrome ? " workspace-shell--manager-demo" : ""}${authorityDemoChrome ? " workspace-shell--authority-demo" : ""}${adminDemoChrome ? " workspace-shell--admin-demo" : ""}`}
      data-active-role={identity.activeRole}
      data-testid="application-shell"
    >
      {rootDemoChrome ? (
        <div className="auditee-demo-ribbon" role="status">
          <span className="auditee-demo-ribbon__dot" aria-hidden="true" />
          <strong>DEMO</strong>
          <span className="auditee-demo-ribbon__text">Frontend clickable prototype — mock data only. Frontend-only demo saved in this browser; no backend, database, AI service, regulatory ingestion, or real integrations.</span>
          <button
            className="auditee-demo-ribbon__reset"
            onClick={() => {
              window.localStorage.removeItem(DEMO_MOCK_STORAGE_KEY);
              window.location.reload();
            }}
            type="button"
          >
            <span>Reset demo</span>
          </button>
        </div>
      ) : null}
      <aside
        aria-hidden={compactNavigation || undefined}
        className="workspace-sidebar"
        inert={compactNavigation || undefined}
      >
        <div className="sidebar-brand">
          <span className="sidebar-brand-mark" aria-hidden="true">
            <span className="sidebar-brand-mark__wing sidebar-brand-mark__wing--primary" />
            <span className="sidebar-brand-mark__wing sidebar-brand-mark__wing--secondary" />
            <span className="sidebar-brand-mark__code">AS</span>
          </span>
          <span><strong>AviaSurveil360</strong><small>{auditeeChrome || managerChrome || authorityChrome || adminChrome ? "OVERSIGHT WORKBENCH" : "Aviation Audit System"}</small></span>
        </div>
        <RoleNavigation activeRole={identity.activeRole} activeRouteId={activeRouteId} />
        <div className="sidebar-footer">
          <button className="nav-item" onClick={onLogout} type="button">
            <span className="nav-item__icon" aria-hidden="true">
              <svg viewBox="0 0 24 24"><path d="M10 17 5 12l5-5" /><path d="M5 12h12" /><path d="M14 4h5v16h-5" /></svg>
            </span>
            <span>{(auditeeChrome || managerChrome || authorityChrome || adminChrome) && identity.mode !== "oidc-session" ? "Role select" : "Logout"}</span>
          </button>
          {rootDemoChrome ? <small>Demo data · frontend-only · saved in this browser</small> : null}
        </div>
      </aside>
      <section className="workspace-content">
        <span className="candidate-boundary" data-testid="active-role">
          {identity.activeRole === "inspector" ? "CAA Inspector" : identity.activeRole === "leadInspector" ? "Lead Inspector" : identity.activeRole}
        </span>
        <MobileNavigation activeRole={identity.activeRole} activeRouteId={activeRouteId} />
        <ApplicationTopbar
          activeRouteId={activeRouteId}
          identity={identity}
          onRoleRequest={onRoleRequest}
          onLogout={onLogout}
          notificationState={notificationState}
        />
        <CandidateBoundary mode={identity.mode} environmentLabel={environmentLabel} />
        {children}
      </section>
    </main>
  );
}
