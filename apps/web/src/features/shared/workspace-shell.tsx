import { useCallback, useState, type PropsWithChildren, type ReactNode } from "react";
import { useNavigate } from "react-router-dom";

import type { FindingSeverity, FindingView, Role } from "../../backend/backend";
import { useApplicationRuntime } from "../../app/providers";
import type { ReactSurfaceId } from "../../app/route-contracts";
import { useOptionalSession } from "../../auth/session-provider";
import { ApplicationShell, type NotificationState, type ShellIdentityPresentation } from "../../ui/application-shell";
import { ROLE_ENTRIES, createRoleEntryPath } from "../../ui/role-select-page";

export function formatLocalDate(value: string | null): string {
  if (!value) return "Not set";
  const [year, month, day] = value.split("-").map(Number);
  return new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    timeZone: "UTC",
  }).format(new Date(Date.UTC(year, month - 1, day)));
}

export function formatSeverity(value: FindingSeverity): string {
  const labels: Record<FindingSeverity, string> = {
    LEVEL_1_CRITICAL: "Level 1 Critical",
    LEVEL_2_MAJOR: "Level 2 Major",
    LEVEL_3_MINOR: "Level 3 Minor",
    OBSERVATION: "Observation",
  };
  return labels[value];
}

export function WorkspaceShell({
  roleLabel,
  routeLabel,
  children,
}: PropsWithChildren<{ roleLabel: string; routeLabel: string }>) {
  const { buildProfile, environmentLabel, beforeSubjectChange } = useApplicationRuntime();
  const session = useOptionalSession();
  const navigate = useNavigate();
  const [logoutError, setLogoutError] = useState<string | null>(null);
  const routeRole = roleForLabel(roleLabel);
  const activeRouteId = routeForLabel(routeLabel, routeRole);
  const authenticatedSession =
    session?.state.status === "authenticated" ? session.state : null;
  const activeRole =
    authenticatedSession?.session.roles.includes(routeRole)
      ? routeRole
      : authenticatedSession?.activeRole ?? routeRole;
  const fallbackMode = buildProfile === "http" ? "canonical-test-role-switch" : "demo-role-switch";
  const mode =
    authenticatedSession
      ? session?.identityMode ?? fallbackMode
      : fallbackMode;
  const identity: ShellIdentityPresentation = {
    mode,
    displayName: authenticatedSession?.session.displayName ?? deterministicDisplayName(activeRole),
    organizationLabel:
      authenticatedSession?.session.organizationId ??
      (activeRole === "auditee" ? "Fly Namibia" : "Namibia Civil Aviation Authority"),
    activeRole,
    availableRoles: authenticatedSession?.session.roles ?? ROLE_ENTRIES.map((entry) => entry.role),
  };
  const notificationState: NotificationState =
    buildProfile === "http"
      ? {
          kind: "unavailable",
          reason: "Notification delivery is not connected in this candidate.",
        }
      : {
          kind: "local",
          unreadCount:
            activeRole === "finance" || activeRole === "auditee" || activeRole === "manager"
              ? 1
              : activeRole === "gm" || activeRole === "executiveDirector" || activeRole === "admin"
                ? 0
                : 2,
          onOpen: () => undefined,
        };
  const handleLogout = useCallback(() => {
    if (mode !== "oidc-session" || !authenticatedSession || !session) {
      navigate("/");
      return;
    }
    setLogoutError(null);
    void (async () => {
      try {
        await beforeSubjectChange?.("LOGOUT");
        await session.logout();
        navigate("/");
      } catch {
        setLogoutError("Local sign-out could not be completed. Your authenticated session remains active.");
      }
    })();
  }, [authenticatedSession, beforeSubjectChange, mode, navigate, session]);
  return (
    <>
      <ApplicationShell
        activeRouteId={activeRouteId}
        environmentLabel={buildProfile === "demo" ? "Deterministic mock data" : environmentLabel}
        identity={identity}
        notificationState={notificationState}
        onLogout={handleLogout}
        onRoleRequest={(role) => {
          session?.setActiveRole(role);
          navigate(createRoleEntryPath(role));
        }}
      >
        {logoutError ? <p className="command-error" role="alert">{logoutError}</p> : null}
        {children}
      </ApplicationShell>
    </>
  );
}

const roleLabels: Record<string, Role> = {
  "CAA Inspector": "inspector",
  "Lead Inspector": "leadInspector",
  "Department Manager": "manager",
  "General Manager": "gm",
  "Finance Review": "finance",
  "Executive Director": "executiveDirector",
  "Auditee — Fly Namibia": "auditee",
  "Admin Preview": "admin",
};

const deterministicNames: Record<Role, string> = {
  inspector: "Aylin Sezer",
  leadInspector: "Caner Yildiz",
  manager: "Mehmet Kaya",
  gm: "Okan Demir",
  finance: "Derya Acar",
  executiveDirector: "Ufuk Aslan",
  auditee: "Fly Namibia Quality Manager",
  admin: "System Admin",
};

function deterministicDisplayName(role: Role): string {
  return deterministicNames[role];
}

const routeLabels: Record<string, ReactSurfaceId> = {
  "My Assignments": "inspector-home",
  "Lead Review": "lead-home",
  Dashboard: "manager-home",
  "GM Dashboard": "gm-home",
  "Finance Review": "finance-home",
  "Executive Dashboard": "executive-home",
  "Corrective Actions": "auditee-home",
  Templates: "admin-home",
  "Audit Detail": "audit-detail",
  "Checklist Runner": "checklist-runner",
  "Organization Registry": "organization-registry",
  "Audit Plan Calendar": "audit-plan",
  "Department Planning": "audit-plan",
  "Finding Detail": "finding-detail",
  Findings: "inspector-findings",
  Messages: "inspector-messages",
  "Audit Work Queue": "inspector-calendar",
  "Inspector Reports": "inspector-reports",
  "Closure Report Preview": "closure-report-preview",
  "AI Inspector Assistant": "inspector-assistant",
  Profile: "inspector-profile",
  "Lead Preliminary Reports": "lead-preliminary-reports",
  "Lead Final Reports": "lead-final-reports",
  "Lead Calendar": "lead-calendar",
  "Lead Messages": "lead-messages",
  "Lead Analytics & Reports": "lead-analytics-reports",
  "Lead Settings": "lead-settings",
  "CAP Review": "cap-review",
  "Inspection Evidence": "evidence-review",
  "Report Preview": "report-preview",
  "Reports Approval": "report-preview",
  "Manager Audits": "manager-audits",
  "Inspection Team": "manager-inspection-team",
  "Findings Review": "manager-findings-review",
  "CAP Monitoring": "manager-cap-monitoring",
  "Checklist Management": "manager-checklist-management",
  "Organization Detail": "organization-detail",
  "Preliminary Report Review": "manager-preliminary-report-review",
  "Department Manager Review": "manager-cap-closure-review",
  "Risk Dashboard": "manager-risk-dashboard",
  "Safety Intelligence": "manager-safety-intelligence",
  "Organization Risk Profile": "organization-risk-profile",
  "SSP / NASP": "manager-ssp-nasp",
  "USOAP Readiness": "manager-usoap-readiness",
  "CAP Effectiveness": "manager-cap-effectiveness",
  "Inspection Package Builder": "inspection-package-builder",
  "New Audit Wizard 1": "new-audit-wizard-1",
  "New Audit Wizard 2": "new-audit-wizard-2",
  "New Audit Wizard 3": "new-audit-wizard-3",
  "New Audit Wizard 4": "new-audit-wizard-4",
  "New Audit Wizard 5": "new-audit-wizard-5",
};

const roleRouteLabels: Partial<Record<Role, Record<string, ReactSurfaceId>>> = {
  inspector: {
    "Report Preview": "closure-report-preview",
  },
  auditee: {
    "Inspection Coordination": "auditee-inspection-coordination",
    "Preliminary Reports": "auditee-preliminary-reports",
    "Final Reports": "auditee-final-reports",
    "Final Report Preview": "auditee-report-preview",
    "Auditee Messages": "auditee-messages",
    Documents: "auditee-documents",
    "Auditee Settings": "auditee-settings",
  },
  gm: {
    Planning: "gm-planning",
    "Report Approvals": "gm-report-approvals",
    Departments: "gm-departments",
    "Risk Dashboard": "gm-risk-dashboard",
    Settings: "gm-settings",
  },
  executiveDirector: {
    Planning: "executive-planning",
    "Preliminary Reports": "executive-preliminary-reports",
    "Final Reports": "executive-final-reports",
    "Final Report Preview": "executive-report-preview",
    Notifications: "executive-notifications",
    Settings: "executive-settings",
  },
  admin: {
    "Regulatory Library": "admin-regulatory-library",
    "Template List": "admin-template-list",
    "Template Preview": "admin-home",
    "Question Bank": "admin-question-bank",
    "Checklist Builder": "admin-checklist-builder",
    "Version History": "admin-version-history",
    "Admin Inspection Package Builder": "admin-inspection-package-builder",
    "Admin Reports": "admin-reports",
    "Users / Roles": "admin-users-roles",
    Configurations: "admin-configurations",
    "Organisation Master Data": "admin-organization-master-data",
    "Admin Organization Detail": "admin-organization-detail",
    "Audit Log": "admin-audit-log",
  },
};

function roleForLabel(label: string): Role {
  return roleLabels[label] ?? "inspector";
}

function routeForLabel(label: string, role: Role): ReactSurfaceId {
  const roleRoute = roleRouteLabels[role]?.[label];
  if (roleRoute) return roleRoute;
  const direct = routeLabels[label];
  if (direct) return direct;
  return ROLE_ENTRIES.find((entry) => entry.role === role)?.routeId ?? "inspector-home";
}

export function PageHeader({
  eyebrow,
  title,
  description,
  action,
}: {
  eyebrow: string;
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <header className="page-header workbench-page-header">
      <div>
        <p className="eyebrow">{eyebrow}</p>
        <h1>{title}</h1>
        <p className="workspace-purpose">{description}</p>
      </div>
      {action ? <div className="page-header__action">{action}</div> : null}
    </header>
  );
}

export function StatusPill({ children }: PropsWithChildren) {
  return <span className="status-pill">{children}</span>;
}

export function FindingFacts({ finding }: { finding: FindingView }) {
  return (
    <dl className="fact-grid">
      <div><dt>Finding</dt><dd>{finding.findingNumber}</dd></div>
      <div><dt>Status</dt><dd>{finding.status}</dd></div>
      <div><dt>Severity</dt><dd>{formatSeverity(finding.severity)}</dd></div>
      <div><dt>Organization</dt><dd>{finding.organizationName}</dd></div>
      <div><dt>Related Audit</dt><dd>{finding.auditId}</dd></div>
      <div><dt>Due Date</dt><dd>{formatLocalDate(finding.dueDate)}</dd></div>
      <div><dt>Current owner</dt><dd>{finding.currentOwnerType}</dd></div>
      <div><dt>Next action</dt><dd>{finding.nextAction}</dd></div>
    </dl>
  );
}

export function CommandError({ message }: { message: string | null }) {
  return message ? <p className="command-error" role="alert">{message}</p> : null;
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "The candidate action could not be completed.";
}
