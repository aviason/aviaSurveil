import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { useApplicationRuntime, useBackendForRole } from "../../app/providers";
import { RoleHandoff } from "../../auth/role-handoff";
import { useOptionalSession } from "../../auth/session-provider";
import type { FindingView, ManagerDashboardProjection, OrganizationSummary, PlanningItemView, ReportVersionView, Role } from "../../backend/backend";
import { createRoleEntryPath } from "../../ui/role-select-page";
import { discoverCAAReportVersions } from "./report-discovery";
import { CommandError, errorMessage, formatLocalDate, formatSeverity, WorkspaceShell } from "../shared/workspace-shell";

function useContinuation() {
  const runtime = useApplicationRuntime();
  const session = useOptionalSession();
  const navigate = useNavigate();
  return {
    identityMode: session?.identityMode ?? runtime.identityMode ?? "oidc-session",
    session: session?.state ?? { status: "unauthenticated" as const },
    request(role: Role) { session?.setActiveRole(role); navigate(createRoleEntryPath(role)); },
  };
}

export function ExecutiveDashboardPage() {
  const backend = useBackendForRole("executiveDirector");
  const handoff = useContinuation();
  const [dashboard, setDashboard] = useState<ManagerDashboardProjection | null>(null);
  const [findings, setFindings] = useState<FindingView[]>([]);
  const [organizations, setOrganizations] = useState<OrganizationSummary[]>([]);
  const [plans, setPlans] = useState<PlanningItemView[]>([]);
  const [reports, setReports] = useState<ReportVersionView[]>([]);
  const [selectedPlanId, setSelectedPlanId] = useState<string | null>(null);
  const [selectedReportId, setSelectedReportId] = useState<string | null>(null);
  const [reportOpen, setReportOpen] = useState(false);
  const [planOpen, setPlanOpen] = useState(false);
  const [reportReason, setReportReason] = useState("");
  const [planReason, setPlanReason] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    void Promise.all([
      backend.dashboards.getManagerProjection({}),
      backend.findings.list({ limit: 50 }),
      backend.organizations.list({ limit: 100 }),
      backend.planning.list({ limit: 20 }),
      discoverCAAReportVersions(backend, ["FINAL"]),
    ]).then(([nextDashboard, nextFindings, nextOrganizations, nextPlans, nextReports]) => {
      setDashboard(nextDashboard); setFindings(nextFindings.items); setOrganizations(nextOrganizations.items); setPlans(nextPlans.items); setReports(nextReports);
    }).catch((cause) => setError(errorMessage(cause)));
  }, [backend]);

  async function issueReport(): Promise<void> {
    if (!report) return;
    if (!reportReason.trim()) { setError("Report decision reason is required."); return; }
    setBusy(true); setError(null);
    try { const updated = await backend.reports.decide({ operationId: `EXEC-REPORT-${report.reportVersionId}-${report.revision}`, reportVersionId: report.reportVersionId, expectedReportVersionRevision: report.revision, decision: "ISSUE_AND_LOCK", reason: reportReason }); setReports((current) => current.map((candidate) => candidate.reportVersionId === updated.reportVersionId ? updated : candidate)); setReportReason(""); } catch (cause) { setError(errorMessage(cause)); } finally { setBusy(false); }
  }

  const plan = selectedPlanId ? plans.find((candidate) => candidate.id === selectedPlanId) ?? null : null;
  const report = selectedReportId ? reports.find((candidate) => candidate.reportVersionId === selectedReportId) ?? null : null;
  async function approvePlan(): Promise<void> {
    if (!plan) return;
    if (!planReason.trim()) { setError("Plan decision reason is required."); return; }
    setBusy(true); setError(null);
    try { const updated = await backend.planning.decide({ operationId: `EXEC-PLAN-${plan.id}-${plan.revision}`, planningItemId: plan.id, expectedPlanningRevision: plan.revision, decision: "APPROVE_PLAN", reason: planReason, expectedSubmittedScopeSnapshotId: plan.submittedScopeSnapshotId, expectedPlanningSnapshotDigest: plan.planningSnapshotDigest }); setPlans((current) => current.map((candidate) => candidate.id === updated.id ? updated : candidate)); setPlanReason(""); } catch (cause) { setError(errorMessage(cause)); } finally { setBusy(false); }
  }

  const pendingPlans = plans.filter((item) => item.status === "EXECUTIVE_DIRECTOR_REVIEW");
  const pendingReports = reports.filter((candidate) => candidate.status === "EXECUTIVE_DIRECTOR_REVIEW").length;
  const openFindings = findings.filter((finding) => finding.status !== "CLOSED");
  const overdue = openFindings.filter((finding) => finding.dueState === "OVERDUE");
  const kpis = [
    ["Total Audits", plans.length, "Server-owned planning records", ""],
    ["Audits in Progress", plans.filter((item) => item.status !== "RELEASED").length, "Scheduled, active, or follow-up", "info"],
    ["Pending Approval", pendingPlans.length + pendingReports, `${pendingPlans.length} plans · ${pendingReports} reports`, "warn"],
    ["Final Reports", reports.length, "All visible Final Report records", "ok"],
    ["Overdue Actions", overdue.length, "Open Finding Due Dates", "danger"],
    ["Closed This Period", dashboard?.closedFindings ?? 0, "Closed audits and Findings", "ok"],
  ] as const;
  const canonicalFinding = findings.find((finding) => report?.findingIds.includes(finding.id)) ?? null;

  return (
    <WorkspaceShell roleLabel="Executive Director" routeLabel="Executive Dashboard">
      <div className="authority-workspace executive-dashboard-page">
        <header className="authority-page-head workbench-page-header"><h1>Executive Director Dashboard</h1><p>Final decision workbench for surveillance plans and Final Reports.</p></header>
        <CommandError message={error} />
        <div className="authority-guardrails"><span>Final authorized approval</span><span>Decision recorded without signature assertion</span><span>No automatic enforcement or closure decision</span></div>
        <section aria-label="Executive overview" className="executive-kpi-grid">{kpis.map(([label, value, foot, tone]) => <article className={tone ? `is-${tone}` : ""} key={label}><span>{label}</span><b>{value}</b><small>{foot}</small></article>)}</section>
        <section className="executive-decision-grid"><section aria-label="Planning approvals" className="executive-panel"><header><div><span>Decision queue</span><h2>Planning approvals</h2></div><Link aria-label="View all Planning" to="/executive-director/planning">View all</Link></header>{pendingPlans.length ? pendingPlans.map((item) => <article className="executive-decision-row" key={item.id}><div><span>{item.id} · Cabin Safety</span><b>{item.title}</b><small>{item.organizationName} · Target {formatLocalDate(item.scheduledDate)}</small></div><div><span className="authority-badge is-warn">Pending Final Approval</span><button aria-label={`Review plan ${item.id}`} onClick={() => { setSelectedPlanId(item.id); setPlanOpen(true); }} type="button">Review plan</button></div></article>) : <div className="executive-empty"><b>No plans require an Executive Director decision.</b><span>Approved or returned items remain available in Planning.</span></div>}</section>
          <section aria-label="Final Report approvals" className="executive-panel"><header><div><span>Decision queue</span><h2>Final Report approvals</h2></div><Link aria-label="View all Final Reports" to="/executive-director/final-reports">View all</Link></header>{reports.length ? reports.map((candidate) => <article className="executive-decision-row" key={candidate.reportVersionId}><div><span>{candidate.reportVersionId} · {candidate.auditId}</span><b>{organizations.find((item) => item.id === candidate.organizationId)?.legalName ?? candidate.organizationId}</b><small>Server-owned immutable version {candidate.version}</small></div><div><span className="authority-badge is-warn">{candidate.status === "EXECUTIVE_DIRECTOR_REVIEW" ? "Pending Final Authorized Approval" : candidate.status}</span><button aria-label={`Review report ${candidate.reportVersionId}`} onClick={() => { setSelectedReportId(candidate.reportVersionId); setReportOpen(true); }} type="button">Review report</button></div></article>) : <div className="executive-empty"><b>No Final Report versions are available yet.</b><span>The queue will populate after an inspection completes the report lifecycle.</span></div>}</section></section>
        <section className="executive-lower-grid"><section aria-label="Department overview" className="executive-panel"><header><div><span>Portfolio context</span><h2>Department overview</h2></div></header><div className="executive-department-list">{["Airworthiness", "Cabin Safety", "Certification", "Licensing", "Ramp", "Security"].map((name, index) => <div key={name}><span><b>{name}</b><small>{index === 1 ? plans.length : 0} audits</small></span><span><b>{index === 1 ? plans.filter((item) => item.status !== "RELEASED").length : 0}</b><small>active</small></span><span><b>{index === 1 ? openFindings.length : 0}</b><small>open Findings</small></span></div>)}</div></section>
          <section aria-label="Overdue actions" className="executive-panel"><header><div><span>Due Date attention</span><h2>Overdue actions</h2></div></header>{overdue.length ? overdue.map((finding) => <div className="executive-overdue-row" key={finding.id}><span><b>{finding.findingNumber}</b><small>{finding.title}</small></span><span><b>{formatSeverity(finding.severity)}</b><small>{formatLocalDate(finding.dueDate)}</small></span></div>) : <div className="executive-empty"><b>No overdue actions.</b><span>Due Date monitoring remains informational.</span></div>}</section>
          <aside className="executive-risk-guardrail"><span>Oversight Health context</span><h2>Management indicator only</h2><p>Risk and workload summaries are informational only. They do not make an automatic legal, enforcement, certificate suspension, Finding closure, or audit closure decision.</p></aside></section>

        {planOpen && plan ? <section aria-label="Executive plan decision" className="executive-authority-decision"><header><span>Executive Director decision</span><h2>Final plan approval</h2></header><dl><div><dt>Status</dt><dd data-testid="planning-status">{plan.status}</dd></div><div><dt>Current owner</dt><dd>Executive Director</dd></div><div><dt>Target</dt><dd>{formatLocalDate(plan.scheduledDate)}</dd></div><div><dt>Revision</dt><dd>Revision {plan.revision}</dd></div></dl>{plan.status === "EXECUTIVE_DIRECTOR_REVIEW" ? <><label>Plan decision reason<textarea value={planReason} onChange={(event) => setPlanReason(event.target.value)} /></label><button disabled={busy} onClick={() => void approvePlan()} type="button">Approve Plan</button></> : null}{plan.status === "GM_RELEASE" ? <RoleHandoff identityMode={handoff.identityMode} session={handoff.session} targetRole="gm" onRoleRequest={handoff.request}>Continue as General Manager</RoleHandoff> : null}</section> : null}
        {reportOpen && report ? <section aria-label="Executive report decision" className="executive-authority-decision"><header><span>Executive Director decision</span><h2>Final Report authority</h2></header><dl><div><dt>Report</dt><dd>{report.reportVersionId}</dd></div><div><dt>Status</dt><dd data-testid="report-status">{report.status}</dd></div><div><dt>Revision</dt><dd>Revision {report.revision}</dd></div><div><dt>Content hash</dt><dd>{report.contentHash}</dd></div></dl>{report.status !== "LOCKED" ? <><label>Report decision reason<textarea value={reportReason} onChange={(event) => setReportReason(event.target.value)} /></label><button disabled={busy} onClick={() => void issueReport()} type="button">Issue and lock report</button></> : <div className="executive-lock-result"><strong data-testid="report-finding-status">{report.findingIds.length ? canonicalFinding?.status ?? `Linked Finding unavailable for ${report.reportVersionId}` : `No Findings linked — relationship unavailable for ${report.reportVersionId}`}</strong><span>Report issue did not close the separately created Finding</span></div>}</section> : null}
      </div>
    </WorkspaceShell>
  );
}
