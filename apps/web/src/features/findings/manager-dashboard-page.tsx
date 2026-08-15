import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { useBackendForRole } from "../../app/providers";
import type {
  FindingView,
  ManagerDashboardProjection,
  OrganizationSummary,
  PlanningItemView,
} from "../../backend/backend";
import {
  CommandError,
  errorMessage,
  formatLocalDate,
  formatSeverity,
  WorkspaceShell,
} from "../shared/workspace-shell";

interface ManagerWorkspaceState {
  dashboard: ManagerDashboardProjection;
  findings: FindingView[];
  organizations: OrganizationSummary[];
  plans: PlanningItemView[];
}

const taskLinks = [
  {
    title: "Planning",
    description: "Track governed surveillance plans and their current approval owner.",
    href: "/department-manager/audit-plan",
    action: "Open Planning",
  },
  {
    title: "Organizations",
    description: "Review regulated organizations and current oversight activity.",
    href: "/department-manager/organizations",
    action: "Open Organizations",
  },
  {
    title: "Audits",
    description: "Open the exact Audit work queue, owner, Due Date, and next action.",
    href: "/department-manager/audits",
    action: "Open Audits",
  },
  {
    title: "Inspection Team",
    description: "Review Audit-scoped Lead and assigned Inspector teams.",
    href: "/department-manager/inspection-team",
    action: "Open Inspection Team",
  },
  {
    title: "Findings Review",
    description: "Review Finding ownership and exact CAP and Evidence state.",
    href: "/department-manager/findings-review",
    action: "Open Findings Review",
  },
  {
    title: "CAP Monitoring",
    description: "Monitor exact CAP revisions without borrowing Inspector review authority.",
    href: "/department-manager/cap-monitoring",
    action: "Open CAP Monitoring",
  },
  {
    title: "Checklist Management",
    description: "Review published checklist versions and configured questions.",
    href: "/department-manager/checklist-management",
    action: "Open Checklist Management",
  },
  {
    title: "Reports Approval",
    description: "Open a server-owned report version from its exact workflow handoff.",
    href: "/department-manager/audits",
    action: "Open Reports Approval",
  },
] as const;

const unavailableTasks = [
  ["Risk Dashboard", "Manager Risk Dashboard has no declared Task 6 route."],
] as const;

function statusLabel(status: string): string {
  return status.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (value) => value.toUpperCase());
}

export function ManagerDashboardPage() {
  const backend = useBackendForRole("manager");
  const [state, setState] = useState<ManagerWorkspaceState | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    void Promise.all([
      backend.dashboards.getManagerProjection({}),
      backend.findings.list({ limit: 50 }),
      backend.organizations.list({ limit: 100 }),
      backend.planning.list({ limit: 20 }),
    ]).then(([dashboard, findings, organizations, plans]) => {
      if (!cancelled) {
        setState({
          dashboard,
          findings: findings.items,
          organizations: organizations.items,
          plans: plans.items,
        });
      }
    }).catch((cause) => {
      if (!cancelled) setError(errorMessage(cause));
    });
    return () => {
      cancelled = true;
    };
  }, [backend]);

  const dashboard = state?.dashboard;
  const openFindings = state?.findings.filter((finding) => finding.status !== "CLOSED") ?? [];
  const indicators = [
    ["Total Audits", state?.plans.length ?? 0, `${state?.plans.filter((item) => item.status !== "RELEASED").length ?? 0} in planning`],
    ["Reports Awaiting Approval", dashboard?.pendingReportReviews ?? 0, "Immutable report versions"],
    ["Open Findings", dashboard?.openFindings ?? 0, "CAA review and auditee action"],
    ["CAPs In Progress", dashboard?.pendingCapReviews ?? 0, "CAP acceptance is not closure"],
    ["Overdue CAPs", dashboard?.overdueFindings ?? 0, "overdue management follow-up"],
    ["Inspection Team", state?.organizations.length ?? 0, "Authorized organization scope"],
  ] as const;

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel="Dashboard">
      <div className="management-workspace manager-dashboard-page">
        <header className="management-page-head workbench-page-header">
          <h1>Department Manager Dashboard</h1>
          <p>Operational planning and oversight for the authorized department, report decisions, Findings, CAPs, and inspection teams.</p>
        </header>
        <CommandError message={error} />
        <span aria-hidden="true" className="planning-raw-status" data-testid="manager-closed-findings">{dashboard?.closedFindings ?? 0}</span>

        <section aria-label="Management indicators" className="manager-dashboard-kpis">
          {indicators.map(([label, value, detail], index) => (
            <article className={`is-${["info", "warn", "danger", "info", "danger", "ok"][index]}`} key={label}>
              <span>{label}</span><strong>{value}</strong><small>{detail}</small>
            </article>
          ))}
        </section>

        <section className="manager-dashboard-tasks">
          <div className="management-section-head"><div><span>Manager workspace</span><h2>What needs attention?</h2></div></div>
          <div className="manager-dashboard-task-grid">
            {taskLinks.map((task) => (
              <Link aria-label={task.action} className="manager-dashboard-task" key={task.title} to={task.href}>
                <span aria-hidden="true">□</span><div><b>{task.title}</b><small>{task.description}</small></div><em>Open →</em>
              </Link>
            ))}
            {unavailableTasks.map(([title, reasonText]) => (
              <button aria-label={`${title} unavailable`} className="manager-dashboard-task" disabled key={title} title={reasonText} type="button">
                <span aria-hidden="true">□</span><div><b>{title}</b><small>{reasonText}</small></div><em>Unavailable</em>
              </button>
            ))}
          </div>
        </section>

        <div className="manager-dashboard-register-grid">
          <section className="management-panel">
            <div className="management-section-head"><div><span>Priority review</span><h2>Recent High-Risk Findings</h2></div></div>
            <div className="management-table-scroll">
              <table aria-label="Priority Findings">
                <thead><tr><th>Finding</th><th>Organization</th><th>Severity</th><th>Current Owner</th><th>Next Action</th><th>Due Date</th><th>Action</th></tr></thead>
                <tbody>
                  {openFindings.length ? openFindings.map((finding) => (
                    <tr key={finding.id}>
                      <td><b>{finding.findingNumber}</b><small>{finding.title}</small></td>
                      <td>{finding.organizationName}</td>
                      <td>{formatSeverity(finding.severity)}</td>
                      <td>{finding.currentOwnerType === "CAA" ? "CAA Inspector" : finding.organizationName}</td>
                      <td>{finding.nextAction}</td>
                      <td>{formatLocalDate(finding.dueDate)}</td>
                      <td><Link to={`/department-manager/findings-review?findingId=${finding.id}`}>Open {finding.id} in Findings Review</Link></td>
                    </tr>
                  )) : <tr><td colSpan={7}>No open high-risk Findings require attention.</td></tr>}
                </tbody>
              </table>
            </div>
          </section>
          <section className="management-panel">
            <div className="management-section-head"><div><span>Surveillance schedule</span><h2>Upcoming Audits</h2></div></div>
            <div className="management-table-scroll">
              <table aria-label="Upcoming Surveillance">
                <thead><tr><th>Plan</th><th>Organization</th><th>Type</th><th>Target</th><th>Status</th><th>Action</th></tr></thead>
                <tbody>
                  {state?.plans.map((plan) => (
                    <tr key={plan.id}>
                      <td><b>{plan.id}</b><small>{plan.title}</small></td><td>{plan.organizationName}</td><td>{plan.inspectionType}</td>
                      <td>{formatLocalDate(plan.scheduledDate)}</td><td>{statusLabel(plan.status)}</td>
                      <td><Link to={`/department-manager/audits?auditId=${plan.id}`}>Open Audit {plan.id}</Link></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        </div>

        <p className="management-advisory">Oversight Health Index is advisory; it does not trigger automatic legal, enforcement, certificate, or Finding-closure decisions.</p>

        <section className="management-panel" aria-label="Finding selection boundary">
          <p>Select an exact Finding from the register above to review CAP, Evidence, and closure state. No Finding is implicitly selected.</p>
        </section>
      </div>
    </WorkspaceShell>
  );
}
