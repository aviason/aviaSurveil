import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";

import { useApplicationRuntime, useBackendForRole } from "../../app/providers";
import { RoleHandoff } from "../../auth/role-handoff";
import { useOptionalSession } from "../../auth/session-provider";
import type {
  FindingView,
  ManagerDashboardProjection,
  OrganizationSummary,
  PlanningAuditPackageSetupView,
  PlanningDecision,
  PlanningItemView,
  ReportVersionView,
  Role,
} from "../../backend/backend";
import { createRoleEntryPath } from "../../ui/role-select-page";
import { discoverCAAReportVersions } from "../reports/report-discovery";
import {
  CommandError,
  errorMessage,
  formatLocalDate,
  WorkspaceShell,
} from "../shared/workspace-shell";
import { planningItemLabel, recordReference } from "../shared/record-presentation";

const roleLabels: Record<Role, string> = {
  inspector: "CAA Inspector",
  leadInspector: "Lead Inspector",
  manager: "Department Manager",
  finance: "Finance Review",
  gm: "General Manager",
  executiveDirector: "Executive Director",
  auditee: "Auditee",
  admin: "Admin Preview",
};

function money(value: number): string {
  return new Intl.NumberFormat("en", { style: "currency", currency: "NAD", maximumFractionDigits: 0 }).format(value);
}

function planningStatusLabel(status: PlanningItemView["status"]): string {
  return status === "FINANCE_REVIEW" ? "Pending Finance Review" : status.replaceAll("_", " ");
}

function useRoleContinuation() {
  const runtime = useApplicationRuntime();
  const session = useOptionalSession();
  const navigate = useNavigate();
  return {
    identityMode: session?.identityMode ?? runtime.identityMode ?? "oidc-session",
    session: session?.state ?? { status: "unauthenticated" as const },
    request(role: Role) {
      session?.setActiveRole(role);
      navigate(createRoleEntryPath(role));
    },
  };
}

function ApprovalRail({ status }: { status: PlanningItemView["status"] }) {
  const stages = [
    ["DEPARTMENT_MANAGER", "Department Manager"],
    ["FINANCE_REVIEW", "Finance Review"],
    ["GM_REVIEW", "GM Review"],
    ["EXECUTIVE_DIRECTOR_REVIEW", "Executive Director"],
    ["GM_RELEASE", "GM Release"],
  ] as const;
  const currentStage = status === "RETURNED"
    ? "DEPARTMENT_MANAGER"
    : status === "RELEASED"
      ? null
      : status;
  const current = currentStage ? stages.findIndex(([stage]) => stage === currentStage) : stages.length;
  return <ol aria-label="Finance approval flow" className="authority-approval-rail">{stages.map(([stage, label], index) => <li aria-current={index === current ? "step" : undefined} className={index < current ? "done" : index === current ? "current" : ""} data-planning-stage={stage} key={stage}><span>{index < current ? "✓" : index + 1}</span><b>{label}</b></li>)}</ol>;
}

function PlanningProposalDossier({ item }: { item: PlanningItemView }) {
  if (!item.purpose && !item.requiredInspectorCount && !item.estimatedChecklistItemCount) return null;
  return <section aria-label="Planning proposal dossier" className="planning-proposal-dossier"><header><span>Immutable Planning proposal</span><h3>What was submitted for approval</h3></header><dl><div><dt>Provider scope</dt><dd>{item.providerScopeLabel ?? "Server-authorized scope"}</dd></div><div><dt>Regulated target</dt><dd>{item.regulatedTargetLabel ?? "Server-authorized target"}</dd></div><div><dt>Purpose</dt><dd>{item.purpose ?? "Not provided"}</dd></div><div><dt>Mode and location</dt><dd>{item.mode ?? "Not provided"}{item.locationLabel ? ` · ${item.locationLabel}` : item.meetingLink ? ` · ${item.meetingLink}` : ""}</dd></div><div><dt>Required inspectors</dt><dd>{item.requiredInspectorCount ?? "Not provided"}</dd></div><div><dt>Estimated checklist items</dt><dd>{item.estimatedChecklistItemCount ?? "Not provided"}</dd></div><div><dt>Workload basis</dt><dd>{item.workloadEstimate ? `${item.workloadEstimate.suggestedCount} suggested · safe range ${item.workloadEstimate.safeMinimum}–${item.workloadEstimate.safeMaximum}` : "Server estimate unavailable"}</dd></div><div><dt>Initiated by</dt><dd>{item.initiatedBy ?? "Department Manager"}</dd></div></dl><p>Exact checklist-item identities are not part of this approval dossier. They become available only in post-release Audit-package preparation.</p></section>;
}

export function FinanceReviewPage() {
  const backend = useBackendForRole("finance");
  const handoff = useRoleContinuation();
  const [items, setItems] = useState<PlanningItemView[]>([]);
  const [selected, setSelected] = useState<PlanningItemView | null>(null);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("pending");
  const [tab, setTab] = useState("summary");
  const [choice, setChoice] = useState<PlanningDecision | null>(null);
  const [reason, setReason] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    void backend.planning.list({ limit: 100 }).then((output) => {
      setItems(output.items);
    }).catch((cause) => setError(errorMessage(cause)));
  }, [backend]);

  const visible = useMemo(() => items.filter((item) => {
    const statusMatch = status === "all" || (status === "pending" && item.status === "FINANCE_REVIEW") || (status === "approved" && item.status !== "FINANCE_REVIEW") || (status === "returned" && item.status === "RETURNED");
    return statusMatch && (!query.trim() || [item.id, item.title, item.organizationName, item.inspectionType].join(" ").toLowerCase().includes(query.toLowerCase()));
  }), [items, query, status]);

  async function decide(): Promise<void> {
    if (!selected || !choice) return;
    if (!reason.trim()) { setError("Finance decision reason is required."); return; }
    setBusy(true); setError(null);
    try {
      const updated = await backend.planning.decide({ operationId: `FINANCE-${choice}-${selected.id}-${selected.revision}`, planningItemId: selected.id, expectedPlanningRevision: selected.revision, decision: choice, reason, expectedSubmittedScopeSnapshotId: selected.submittedScopeSnapshotId, expectedPlanningSnapshotDigest: selected.planningSnapshotDigest });
      setSelected(updated); setItems((current) => current.map((item) => item.id === updated.id ? updated : item)); setChoice(null); setReason("");
    } catch (cause) { setError(errorMessage(cause)); } finally { setBusy(false); }
  }

  return (
    <WorkspaceShell roleLabel="Finance Review" routeLabel="Finance Review">
      <div className="authority-workspace finance-review-page">
        <header className="authority-page-head workbench-page-header"><h1>Finance Review</h1><p>Approve the requested budget for General Manager review or return it to Department Manager for revision.</p></header>
        <CommandError message={error} />
        <div className="authority-guardrails"><span>Budget approval before GM Review</span><span>No plan signature or release</span><span>Server-recorded decision boundary</span></div>
        <section aria-label="Finance review summary" className="finance-summary-strip">
          <article><span>Pending Finance Review</span><b>{items.filter((item) => item.status === "FINANCE_REVIEW").length}</b></article>
          <article><span>Total Requested Budget</span><b>{money(items.reduce((sum, item) => sum + item.estimatedBudget, 0))}</b></article>
          <article><span>Approval path</span><b>Department Manager → Finance Review → GM Review → Executive Director → GM Release</b></article>
        </section>
        <div className="finance-filter-row"><label>Search<input aria-label="Search plans" onChange={(event) => setQuery(event.target.value)} placeholder="Plan, department, organization" value={query} /></label><label>Status<select aria-label="Finance status" onChange={(event) => setStatus(event.target.value)} value={status}><option value="pending">Pending</option><option value="approved">Approved</option><option value="returned">Returned</option><option value="all">All</option></select></label></div>
        <div className="finance-review-layout"><div className="finance-review-main">
          <section className="finance-review-queue"><h2>Review Queue</h2><div className="authority-table-scroll"><table aria-label="Finance Review Queue"><thead><tr><th>Plan</th><th>Department</th><th>Requested</th><th>Current Owner</th><th>Status</th><th>Action</th></tr></thead><tbody>{visible.map((item) => <tr className={selected?.id === item.id ? "is-selected" : ""} key={item.id}><td><b>{item.title}</b><small>{recordReference("Plan", item.id)}</small></td><td>Cabin Safety</td><td>{money(item.estimatedBudget)}</td><td>{roleLabels[item.currentOwnerRole]}</td><td><span className={`authority-badge${item.status === "FINANCE_REVIEW" ? " is-warn" : ""}`}>{planningStatusLabel(item.status)}</span></td><td><button aria-label={selected?.id === item.id ? `${planningItemLabel(item.title, item.id)} is already selected` : `Review ${planningItemLabel(item.title, item.id)}`} disabled={selected?.id === item.id} onClick={() => setSelected(item)} title={selected?.id === item.id ? "This Planning item is already open in the Finance dossier." : undefined} type="button">Review</button></td></tr>)}</tbody></table></div></section>
          {selected ? <section className="finance-review-detail"><div className="finance-review-title"><div><span>Selected Plan</span><h2>{selected.title}</h2><p>{recordReference("Plan", selected.id)} · {selected.organizationName}</p></div><span className={`authority-badge${selected.status === "FINANCE_REVIEW" ? " is-warn" : ""}`}>{planningStatusLabel(selected.status)}</span></div><PlanningProposalDossier item={selected} /><div className="finance-review-tabs" role="tablist" aria-label="Finance dossier sections">{[["summary", "Budget Summary"], ["breakdown", "Budget Breakdown"], ["documents", "Supporting Documents"], ["history", "Comments & History"]].map(([id, label]) => <button aria-selected={tab === id} className={tab === id ? "is-active" : ""} key={id} onClick={() => setTab(id)} role="tab" type="button">{label}</button>)}</div><div className="finance-review-panel" role="tabpanel">{tab === "summary" ? <><h2>Budget Summary</h2><div className="finance-summary-grid"><div><span>Requested Budget</span><b>{money(selected.estimatedBudget)}</b></div><div><span>Available for Plan</span><b>Not in contract</b></div><div><span>Remaining Annual Budget</span><b>Not in contract</b></div><div><span>Budget Reconciliation</span><b>{money(selected.estimatedBudget)}</b></div></div><h3>Resource justification</h3><p>{selected.nextAction}</p><div className="authority-callout"><b>Finance boundary:</b> Finance reviews budget and resource justification only.</div></> : <><h2>{tab === "breakdown" ? "Budget Breakdown" : tab === "documents" ? "Supporting Documents" : "Comments & History"}</h2><p>This detail is not represented by the current Backend contract.</p></>}</div></section> : null}
        </div><aside className="finance-review-side">{selected ? <><section className="finance-review-rail"><h2>Approval Flow</h2><ApprovalRail status={selected.status} /></section><section className="finance-review-decision"><div><span>Current owner</span><b data-testid="planning-owner">{roleLabels[selected.currentOwnerRole]}</b><small>{selected.nextAction}</small><i data-testid="planning-status">{selected.status}</i><em><span>Target {formatLocalDate(selected.scheduledDate)}</span><span>Revision {selected.revision}</span></em></div>{selected.status === "FINANCE_REVIEW" ? <><div className="finance-decision-buttons"><button disabled={busy} onClick={() => setChoice("APPROVE_BUDGET")} type="button">Approve Budget</button><button disabled={busy} onClick={() => setChoice("RETURN_FOR_REVISION")} type="button">Return for Revision</button></div>{choice ? <div className="finance-decision-form"><h3>{choice === "APPROVE_BUDGET" ? "Approve Budget" : "Return for Revision"}</h3><label>Finance decision reason<textarea value={reason} onChange={(event) => setReason(event.target.value)} /></label><button disabled={busy} onClick={() => void decide()} type="button">Confirm Finance Decision</button></div> : null}</> : null}{selected.status === "GM_REVIEW" ? <RoleHandoff identityMode={handoff.identityMode} session={handoff.session} targetRole="gm" onRoleRequest={handoff.request}>Continue as General Manager</RoleHandoff> : null}</section></> : null}</aside></div>
      </div>
    </WorkspaceShell>
  );
}

export function GeneralManagerDashboardPage() {
  const backend = useBackendForRole("gm");
  const handoff = useRoleContinuation();
  const [dashboard, setDashboard] = useState<ManagerDashboardProjection | null>(null);
  const [findings, setFindings] = useState<FindingView[]>([]);
  const [organizations, setOrganizations] = useState<OrganizationSummary[]>([]);
  const [plans, setPlans] = useState<PlanningItemView[]>([]);
  const [reports, setReports] = useState<ReportVersionView[]>([]);
  const [selectedPlanId, setSelectedPlanId] = useState<string | null>(null);
  const [choice, setChoice] = useState<PlanningDecision | null>(null);
  const [reason, setReason] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    void Promise.all([backend.dashboards.getManagerProjection({}), backend.findings.list({ limit: 50 }), backend.organizations.list({ limit: 100 }), backend.planning.list({ limit: 100 }), discoverCAAReportVersions(backend)]).then(([nextDashboard, nextFindings, nextOrganizations, nextPlans, nextReports]) => { setDashboard(nextDashboard); setFindings(nextFindings.items); setOrganizations(nextOrganizations.items); setPlans(nextPlans.items); setReports(nextReports); }).catch((cause) => setError(errorMessage(cause)));
  }, [backend]);
  const plan = selectedPlanId ? plans.find((candidate) => candidate.id === selectedPlanId) ?? null : null;
  async function decide(): Promise<void> {
    if (!plan || !choice) return;
    if (!reason.trim()) { setError("General Manager decision reason is required."); return; }
    setBusy(true); setError(null);
    try { const updated = await backend.planning.decide({ operationId: `GM-${choice}-${plan.id}-${plan.revision}`, planningItemId: plan.id, expectedPlanningRevision: plan.revision, decision: choice, reason, expectedSubmittedScopeSnapshotId: plan.submittedScopeSnapshotId, expectedPlanningSnapshotDigest: plan.planningSnapshotDigest }); setPlans((current) => current.map((item) => item.id === updated.id ? updated : item)); setChoice(null); setReason(""); } catch (cause) { setError(errorMessage(cause)); } finally { setBusy(false); }
  }
  const departments = ["Cabin Safety", "Security", "Airworthiness", "Ramp", "Unassigned", "Certification", "Licensing"];
  const highRisk = findings.filter((finding) => ["LEVEL_1_CRITICAL", "LEVEL_2_MAJOR"].includes(finding.severity)).length;
  const pendingReports = reports.filter((report) => report.status === "GM_REVIEW");
  const kpis = [["Pending Preliminary Reports", pendingReports.filter((report) => report.kind === "PRELIMINARY").length], ["Pending Final Reports", pendingReports.filter((report) => report.kind === "FINAL").length], ["High Risk Findings", highRisk], ["Reports Awaiting Your Approval", pendingReports.length], ["Overdue CAPs", dashboard?.overdueFindings ?? 0]] as const;
  return (
    <WorkspaceShell roleLabel="General Manager" routeLabel="GM Dashboard">
      <div className="authority-workspace gm-dashboard-page"><header className="authority-page-head workbench-page-header"><h1>General Manager Dashboard</h1><p>Review intermediate Preliminary and Final Report decisions, department exposure, high-risk findings, and overdue CAPs.</p><span className="qualification-boundary">Due</span></header><CommandError message={error} />
        <section aria-label="General Manager indicators" className="gm-kpis">{kpis.map(([label, value]) => <article key={label}><span>{label}</span><strong>{value}</strong></article>)}</section>
        <div className="gm-dashboard-grid"><section className="gm-panel"><header><span>Cross-department oversight</span><h2>Department Overview</h2></header><div className="gm-table"><table aria-label="Department Overview"><thead><tr><th>Department</th><th>Audits</th><th>Active</th><th>Findings</th><th>High</th><th>Medium</th><th>Overdue CAPs</th><th>Exposure</th></tr></thead><tbody>{departments.map((department, index) => <tr key={department}><td><b>{department}</b></td><td>{index === 0 ? plans.length : 0}</td><td>{index === 0 ? plans.filter((item) => item.status !== "RELEASED").length : 0}</td><td>{index === 0 ? findings.length : 0}</td><td>{index === 0 ? highRisk : 0}</td><td>0</td><td>{index === 0 ? dashboard?.overdueFindings ?? 0 : 0}</td><td><span className="gm-exposure-score">{index === 0 ? findings.length + plans.length : 0}</span></td></tr>)}</tbody></table></div><Link to="/general-manager/departments">View All Departments</Link></section>
          <section aria-label="Risk Heat Map" className="gm-panel gm-risk-heat"><header><span>Likelihood × Impact</span><h2>Risk Heat Map</h2></header><div className="gm-risk-matrix">{Array.from({ length: 25 }, (_, index) => { const score = (5 - Math.floor(index / 5)) * ((index % 5) + 1); return <div className={score >= 15 ? "is-critical" : score >= 10 ? "is-high" : score >= 5 ? "is-medium" : "is-low"} key={index}><b>{index === 22 ? highRisk : 0}</b><small>{score}</small></div>; })}</div><div className="gm-risk-axis"><span>Higher likelihood ↑</span><span>Impact →</span></div></section></div>
        <section className="gm-panel gm-dashboard-queue"><header><span>Intermediate review stage</span><h2>Report Review Queue</h2></header><div className="gm-table gm-approval-table"><table aria-label="Report Review Queue"><thead><tr><th>Report</th><th>Type</th><th>Organization</th><th>Status</th><th>Owner</th><th>Decision</th></tr></thead><tbody>{reports.length ? reports.map((report) => <tr key={report.reportVersionId}><td><b>{report.reportVersionId}</b><small>{report.auditId}</small></td><td>{report.kind === "PRELIMINARY" ? "Preliminary Report" : "Final Report"}</td><td>{organizations.find((organization) => organization.id === report.organizationId)?.legalName ?? report.organizationId}</td><td>{report.status}</td><td>{report.status === "GM_REVIEW" ? "General Manager" : report.status === "EXECUTIVE_DIRECTOR_REVIEW" || report.status === "LOCKED" ? "Executive Director" : "Department Manager"}</td><td>{report.status === "GM_REVIEW" ? <Link aria-label={`Open report ${report.reportVersionId}`} to="/general-manager/report-approvals">Open Report</Link> : <button aria-label={`Open report ${report.reportVersionId} unavailable`} disabled title={`Report version ${report.reportVersionId} is ${report.status}; General Manager can open exact report decisions only at GM_REVIEW.`} type="button">Open Report</button>}</td></tr>) : <tr><td colSpan={6}><b>No report versions are available yet.</b><small>The queue will populate after an inspection completes the report lifecycle.</small></td></tr>}</tbody></table></div></section>
        <section aria-label="General Manager planning queue" className="gm-panel gm-dashboard-queue"><header><span>Planning decisions</span><h2>Select a Planning item</h2></header>{plans.length ? <div className="gm-table"><table><thead><tr><th>Planning item</th><th>Organization</th><th>Status</th><th>Action</th></tr></thead><tbody>{plans.map((item) => <tr key={item.id}><td><b>{item.title}</b><small>{recordReference("Plan", item.id)}</small></td><td>{item.organizationName}</td><td>{planningStatusLabel(item.status)}</td><td><button aria-label={selectedPlanId === item.id ? `${planningItemLabel(item.title, item.id)} selected` : `Open ${planningItemLabel(item.title, item.id)}`} disabled={selectedPlanId === item.id} onClick={() => setSelectedPlanId(item.id)} type="button">{selectedPlanId === item.id ? "Selected" : "Open"}</button></td></tr>)}</tbody></table></div> : <p>No server-owned Planning items are available.</p>}</section>
        <p className="gm-authority-note"><b>Authority boundary:</b> General Manager review may return or forward a Preliminary or Final Report. General Manager cannot issue, sign, lock, or close a Finding.</p>
        {plan && ["GM_REVIEW", "GM_RELEASE", "EXECUTIVE_DIRECTOR_REVIEW", "RELEASED"].includes(plan.status) ? <section aria-label="General Manager planning decision" className="gm-planning-decision"><header><span>Planning authority</span><h2>{plan.title}</h2></header><PlanningProposalDossier item={plan} /><dl><div><dt>Status</dt><dd data-testid="planning-status">{plan.status}</dd></div><div><dt>Current owner</dt><dd data-testid="planning-owner">{roleLabels[plan.currentOwnerRole]}</dd></div><div><dt>Target</dt><dd>{formatLocalDate(plan.scheduledDate)}</dd></div><div><dt>Revision</dt><dd>Revision {plan.revision}</dd></div></dl>{["GM_REVIEW", "GM_RELEASE"].includes(plan.status) ? <><div className="gm-decision-buttons"><button disabled={busy} onClick={() => setChoice(plan.status === "GM_RELEASE" ? "RELEASE_PLAN" : "FORWARD_FOR_FINAL_APPROVAL")} type="button">{plan.status === "GM_RELEASE" ? "Release Plan" : "Forward to Executive Director"}</button><button disabled={busy} onClick={() => setChoice("RETURN_FOR_REVISION")} type="button">Return for Revision</button></div>{choice ? <div className="gm-decision-form"><label>General Manager decision reason<textarea value={reason} onChange={(event) => setReason(event.target.value)} /></label><button onClick={() => void decide()} type="button">Confirm General Manager Decision</button></div> : null}</> : null}{plan.status === "EXECUTIVE_DIRECTOR_REVIEW" ? <RoleHandoff identityMode={handoff.identityMode} session={handoff.session} targetRole="executiveDirector" onRoleRequest={handoff.request}>Continue as Executive Director</RoleHandoff> : null}</section> : null}
      </div>
    </WorkspaceShell>
  );
}

function PlanningGovernancePage({ role }: { role: "gm" | "executiveDirector" }) {
  const backend = useBackendForRole(role);
  const [items, setItems] = useState<PlanningItemView[]>([]);
  const [selectedId, setSelectedId] = useState("");
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [reason, setReason] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    void backend.planning.list({ limit: 50 }).then((output) => {
      setItems(output.items);
    }).catch((cause) => setError(errorMessage(cause)));
  }, [backend]);

  const selected = items.find((item) => item.id === selectedId) ?? null;
  const visible = useMemo(() => items.filter((item) => {
    const matchesQuery = !query.trim() || [item.id, item.title, item.organizationName, item.inspectionType]
      .join(" ").toLowerCase().includes(query.trim().toLowerCase());
    return matchesQuery && (statusFilter === "all" || item.status === statusFilter);
  }), [items, query, statusFilter]);
  const expectedStatus = role === "gm" ? ["GM_REVIEW", "GM_RELEASE"] : ["EXECUTIVE_DIRECTOR_REVIEW"];
  const canDecide = selected ? expectedStatus.includes(selected.status) : false;

  async function decide(decision: PlanningDecision): Promise<void> {
    if (!selected) return;
    if (!reason.trim()) {
      setError(`${role === "gm" ? "General Manager" : "Executive Director"} decision reason is required.`);
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const updated = await backend.planning.decide({
        operationId: `${role === "gm" ? "GM" : "EXEC"}-${decision}-${selected.id}-${selected.revision}`,
        planningItemId: selected.id,
        expectedPlanningRevision: selected.revision,
        decision,
        reason,
        expectedSubmittedScopeSnapshotId: selected.submittedScopeSnapshotId,
        expectedPlanningSnapshotDigest: selected.planningSnapshotDigest,
      });
      setItems((current) => current.map((item) => item.id === updated.id ? updated : item));
      setReason("");
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  const isGm = role === "gm";
  const testId = isGm ? "gm-planning-page" : "executive-planning-page";
  return (
    <WorkspaceShell roleLabel={isGm ? "General Manager" : "Executive Director"} routeLabel="Planning">
      <div
        className={`executive-workspace-page executive-planning-page${isGm ? " gm-planning-page" : ""}`}
        data-planning-item-id={selected?.id}
        data-planning-revision={selected?.revision}
        data-testid={testId}
      >
        <header className="authority-page-head workbench-page-header">
          <h1>Planning</h1>
          <p>{isGm ? "Review Finance-approved plans and release Executive-approved plans to Department." : "Review and decide surveillance plans after Department Manager, Finance, and General Manager review."}</p>
        </header>
        <CommandError message={error} />
        <div className="authority-guardrails">
          <span>{isGm ? "General Manager operational review" : "Executive Director final plan approval"}</span>
          <span>{isGm ? "No stage skipping" : "Decision recorded without signature assertion"}</span>
          <span>General Manager Release remains a separate next stage</span>
        </div>
        <section aria-label="Planning approval stages" className="executive-stage-grid">
          {["Department Manager", "Finance Review", "General Manager", "Executive Director", "General Manager Release"].map((label, index) => <article key={label}><span>{label}</span><b>{index + 1}</b></article>)}
        </section>
        <section aria-label="Planning filters" className="executive-planning-filter">
          <label>Search<input onChange={(event) => setQuery(event.target.value)} placeholder="Plan ID, title, organization" value={query} /></label>
          <label>Status<select onChange={(event) => setStatusFilter(event.target.value)} value={statusFilter}><option value="all">All statuses</option><option value="FINANCE_REVIEW">Finance Review</option><option value="GM_REVIEW">GM Review</option><option value="EXECUTIVE_DIRECTOR_REVIEW">Executive Director Review</option><option value="GM_RELEASE">GM Release</option><option value="RELEASED">Released</option><option value="RETURNED">Returned</option></select></label>
        </section>
        <div className="executive-planning-layout">
          <div className="executive-planning-main">
            <section className="executive-panel executive-planning-queue" aria-label="Planning register">
              <header><div><span>Planning register</span><h2>{visible.length} visible plan{visible.length === 1 ? "" : "s"}</h2></div></header>
              <div className="responsive-table-shell"><table><thead><tr><th>Plan</th><th>Organization</th><th>Target</th><th>Budget</th><th>Owner</th><th>Status</th><th>Action</th></tr></thead><tbody>{visible.map((item) => <tr className={item.id === selected?.id ? "is-selected" : ""} key={item.id}><td><b>{item.id}</b><small>{item.title}</small></td><td>{item.organizationName}</td><td>{formatLocalDate(item.scheduledDate)}</td><td>{money(item.estimatedBudget)}</td><td>{roleLabels[item.currentOwnerRole]}</td><td>{item.status}</td><td><button aria-label={item.id === selected?.id ? `${item.id} is already selected` : `Review ${item.id}`} disabled={item.id === selected?.id} onClick={() => setSelectedId(item.id)} title={item.id === selected?.id ? `${item.id} is already open in the Planning dossier.` : undefined} type="button">Review {item.id}</button></td></tr>)}</tbody></table></div>
              {!visible.length ? <p>No plans match these filters.</p> : null}
            </section>
            {selected ? <section className="executive-plan-detail" aria-label={`Selected plan ${selected.id}`}><header><div><span>Selected Planning item</span><h2>{selected.title}</h2><p>{selected.id} · {selected.organizationName}</p></div></header><PlanningProposalDossier item={selected} /><dl className="executive-definition-grid"><div><dt>Planning item</dt><dd>{selected.id}</dd></div><div><dt>Revision</dt><dd>{selected.revision}</dd></div><div><dt>Current owner</dt><dd>{roleLabels[selected.currentOwnerRole]}</dd></div><div><dt>Status</dt><dd data-testid="planning-status">{selected.status}</dd></div><div><dt>Next action</dt><dd>{selected.nextAction}</dd></div><div><dt>Budget</dt><dd>{money(selected.estimatedBudget)}</dd></div></dl><ol aria-label="Planning decision path" className="approval-rail">{["Department Manager", "Finance Review", "General Manager", "Executive Director", "General Manager Release"].map((label, index) => <li key={label}><span>{index + 1}</span><b>{label}</b></li>)}</ol></section> : null}
          </div>
          <aside>
            {selected ? <section className="executive-decision-panel" aria-label={`${isGm ? "General Manager" : "Executive Director"} planning decision`}>
              <span>{isGm ? "General Manager" : "Executive Director"} decision</span>
              <h2>{isGm ? selected.status === "GM_RELEASE" ? "Release approved plan" : "Operational plan review" : "Final plan approval"}</h2>
              {canDecide ? <>
                <label>{isGm ? "General Manager decision reason" : "Executive Director plan decision reason"}<textarea onChange={(event) => setReason(event.target.value)} value={reason} /></label>
                {isGm ? <>
                  <button disabled={busy} onClick={() => void decide(selected.status === "GM_RELEASE" ? "RELEASE_PLAN" : "FORWARD_FOR_FINAL_APPROVAL")} type="button">{selected.status === "GM_RELEASE" ? `Release ${selected.id} to Department Manager` : `Forward ${selected.id} to Executive Director`}</button>
                  <button disabled={busy} onClick={() => void decide("RETURN_FOR_REVISION")} type="button">Return {selected.id} for revision</button>
                </> : <>
                  <button disabled={busy} onClick={() => void decide("APPROVE_PLAN")} type="button">Approve plan {selected.id}</button>
                  <button disabled={busy} onClick={() => void decide("RETURN_FOR_REVISION")} type="button">Return {selected.id} to General Manager</button>
                </>}
              </> : <button aria-label={`Planning decision unavailable for ${selected.id}`} disabled title={`Planning item ${selected.id} is ${selected.status}; ${isGm ? "General Manager" : "Executive Director"} decision is unavailable at this stage.`} type="button">Decision unavailable</button>}
            </section> : null}
          </aside>
        </div>
      </div>
    </WorkspaceShell>
  );
}

export function GeneralManagerPlanningPage() { return <PlanningGovernancePage role="gm" />; }
export function ExecutivePlanningPage() { return <PlanningGovernancePage role="executiveDirector" />; }

export function AuditPlanCalendarPage() {
  const backend = useBackendForRole("manager");
  const workflow = backend.canonicalAuditWorkflow;
  const proposal = backend.planningProposal;
  const [searchParams] = useSearchParams();
  const [items, setItems] = useState<PlanningItemView[]>([]);
  const [selected, setSelected] = useState<PlanningItemView | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [preparation, setPreparation] = useState<Awaited<ReturnType<NonNullable<typeof workflow>["prepare"]>> | null>(null);
  const [assignment, setAssignment] = useState<Awaited<ReturnType<NonNullable<typeof workflow>["assignLead"]>> | null>(null);
  const [materialized, setMaterialized] = useState<Awaited<ReturnType<NonNullable<typeof workflow>["materialize"]>> | null>(null);
  const [leadSubjectId, setLeadSubjectId] = useState("");
  const [preparationConfirmed, setPreparationConfirmed] = useState(false);
  const [packageSetup, setPackageSetup] = useState<PlanningAuditPackageSetupView | null>(null);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    setPackageSetup(null);
    if (!selected || selected.status !== "RELEASED" || !proposal) return;
    let cancelled = false;
    void proposal.getAuditPackageSetup({ planningItemId: selected.id }).then((current) => {
      if (!cancelled) setPackageSetup(current);
    }).catch(() => {
      // A released proposal has no package setup until the Department Manager
      // opens the explicit checklist-selection route.
    });
    return () => { cancelled = true; };
  }, [proposal, selected?.id, selected?.status]);

  useEffect(() => {
    void backend.planning.list({ limit: 100 }).then((output) => {
      setItems(output.items);
      const requestedId = searchParams.get("planningItemId");
      setSelected(output.items.find((item) => item.id === requestedId) ?? null);
    }).catch((cause) => setError(errorMessage(cause)));
  }, [backend, searchParams]);

  useEffect(() => {
    setPreparation(null);
    setAssignment(null);
    setMaterialized(null);
    setPreparationConfirmed(false);
    setLeadSubjectId("");
    if (!selected || selected.status !== "RELEASED" || !workflow) return;
    let cancelled = false;
    void workflow.getPreparation({ planningItemId: selected.id }).then((current) => {
      if (cancelled) return;
      // PREPARATION has no Lead yet; keep the assignment projection in the
      // preparation branch so the DM can resume Lead assignment after a
      // browser/session restart. Later states render the assigned hand-off.
      setAssignment(current.status === "PREPARATION" ? null : current);
      setPreparation({
        assignmentId: current.id,
        planningItemId: selected.id,
        organizationId: current.organizationId,
        status: "PREPARATION",
        revision: current.revision,
      });
      // Hydrate the append-only confirmation receipt.  The assignment
      // revision pin is checked by the server, so a refresh can safely
      // unlock materialization only for the exact confirmed projection.
      setPreparationConfirmed(current.status === "QUESTIONS_ASSIGNED" &&
        Boolean(current.preparationId && current.preparationDigest &&
          current.preparationConfirmedAssignmentRevision === current.revision));
      setStatus(`Resumed server-owned preparation ${current.id} at ${current.status}; revision ${current.revision}.`);
    }).catch(() => {
      // A released plan may not have a preparation yet. The Begin command is
      // the explicit creation boundary and remains enabled in that case.
    });
    return () => { cancelled = true; };
  }, [selected?.id, selected?.status, workflow]);

  async function beginPreparation(): Promise<void> {
    if (!workflow || !selected) return;
    if (packageSetup?.status !== "FINALIZED") {
      setError("Finalize the post-release checklist selection before beginning preparation.");
      return;
    }
    setBusy(true); setError(null);
    try {
      const result = await workflow.prepare(selected.id, {
        operationId: `PREPARE-${selected.id}-${selected.revision}`,
        idempotencyKey: `PREPARE-${selected.id}-${selected.revision}`,
        expectedPlanningRevision: selected.revision,
      });
      setPreparation(result);
      setStatus(`Server-owned preparation ${result.assignmentId} created; no Inspection exists yet.`);
    } catch (cause) { setError(errorMessage(cause)); } finally { setBusy(false); }
  }

  async function assignLead(): Promise<void> {
    if (!workflow || !preparation?.assignmentId || !leadSubjectId.trim()) return;
    setBusy(true); setError(null);
    try {
      const result = await workflow.assignLead(preparation.assignmentId, {
        operationId: `LEAD-${preparation.assignmentId}-${preparation.revision}`,
        idempotencyKey: `LEAD-${preparation.assignmentId}-${preparation.revision}`,
        expectedInspectionRevision: preparation.revision,
        leadSubjectId: leadSubjectId.trim(),
      });
      setAssignment(result);
      setStatus(`Lead ${result.leadSubjectId} assigned from the released Planning date.`);
    } catch (cause) { setError(errorMessage(cause)); } finally { setBusy(false); }
  }

  async function confirmPreparation(): Promise<void> {
    if (!workflow || !assignment) return;
    setBusy(true); setError(null);
    try {
      const confirmation = await workflow.confirmPreparation(assignment.id, { operationId: `CONFIRM-${assignment.id}-${assignment.revision}`, idempotencyKey: `CONFIRM-${assignment.id}-${assignment.revision}`, expectedAssignmentRevision: assignment.revision });
      setStatus("Department Manager preparation confirmed; materialization is now available.");
      setPreparationConfirmed(true);
      setAssignment({ ...assignment, status: "QUESTIONS_ASSIGNED", revision: confirmation.revision,
        preparationId: confirmation.preparationId,
        preparationDigest: confirmation.preparationDigest,
        preparationConfirmedAt: confirmation.confirmedAt,
        preparationConfirmedAssignmentRevision: confirmation.confirmedAssignmentRevision,
      });
    } catch (cause) { setError(errorMessage(cause)); } finally { setBusy(false); }
  }

  async function materializeAudit(): Promise<void> {
    if (!workflow || !assignment) return;
    setBusy(true); setError(null);
    try {
      const result = await workflow.materialize(assignment.id, { operationId: `MATERIALIZE-${assignment.id}-${assignment.revision}`, idempotencyKey: `MATERIALIZE-${assignment.id}-${assignment.revision}`, expectedAssignmentRevision: assignment.revision });
      setMaterialized(result); setStatus(`Inspection ${result.inspectionId} and NOT_STARTED checklist created atomically.`);
    } catch (cause) { setError(errorMessage(cause)); } finally { setBusy(false); }
  }

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel="Department Planning">
      <div className="management-workspace planning-workspace">
        <header className="management-page-head workbench-page-header">
          <h1>Department Planning</h1>
          <p>Review governed Planning work and send a new Audit plan to Finance.</p>
          <div className="planning-parent-actions">
            <Link aria-label="New Audit planning intake" to="/department-manager/new-audit/step-1">New Audit</Link>
          </div>
        </header>
        <CommandError message={error} />
        {status ? <p className="planning-intake-status" role="status">{status}</p> : null}
        <p className="planning-governance-note">Planning authority remains with the Department Manager. Submission creates a Planning item for Finance; Inspector start remains a separate release-stage action.</p>
        {selected ? (
          <section className="planning-command-center management-panel" data-testid="planning-command-center">
            <header className="planning-command-center__head">
              <div className="planning-command-center__identity">
                <span className="planning-command-center__eyebrow">Planning record · {recordReference("Plan", selected.id)}</span>
                <h2>Planning command center</h2>
                <h3>{selected.title}</h3>
                <p>Governed Planning item for {selected.organizationName}; no executable Audit is created before accepted release and confirmation.</p>
              </div>
              <div aria-label="Plan state" className="planning-command-center__state">
                <span className="planning-demo-badge is-warn"><i />{roleLabels[selected.currentOwnerRole]}</span><span className="planning-demo-badge is-warn"><i />{planningStatusLabel(selected.status)}</span><span className="planning-demo-badge is-info"><i />{selected.inspectionType}</span>
                <i className="planning-raw-status" data-testid="planning-status">{selected.status}</i>
              </div>
            </header>
            <div className="planning-command-center__facts">
              <div><span>Organization &amp; Department</span><b>{selected.organizationName}</b><small>Cabin Safety</small></div>
              <div><span>Scope &amp; Risk Driver</span><b>{selected.inspectionType}</b><small>Configured in the exact Planning record</small></div>
              <div><span>Budget &amp; Resources</span><b>{new Intl.NumberFormat("en", { style: "currency", currency: "NAD", maximumFractionDigits: 0 }).format(selected.estimatedBudget)}</b><small>Finance Review remains required at zero budget</small></div>
              <div><span>Target &amp; Readiness</span><b>{formatLocalDate(selected.scheduledDate)}</b><small>{planningStatusLabel(selected.status)}</small></div>
            </div>
            <div aria-label="Current planning action" className="planning-command-center__action" role="status">
              <div><span>Current owner</span><b data-testid="planning-owner">{roleLabels[selected.currentOwnerRole]}</b></div>
              <div><span>Next action</span><b>{selected.nextAction}</b></div>
              <div><span>Blocking reason</span><b>{selected.status === "FINANCE_REVIEW" ? "Waiting for Finance Review decision." : "No unresolved blocking reason"}</b></div>
            </div>
            <div className="planning-command-center__path">
              <div className="planning-command-center__path-head"><span>Decision Path</span><small>Department submission through final approval</small></div>
              <ol aria-label="Planning decision path" className="approval-rail">
                {["Department Manager", "Finance Review", "General Manager", "Executive Director", "General Manager Release"].map((label, index) => {
                  const currentIndex = selected.status === "FINANCE_REVIEW" ? 1 : selected.status === "GM_REVIEW" ? 2 : selected.status === "EXECUTIVE_DIRECTOR_REVIEW" ? 3 : 4;
                  return (
                  <li className={`approval-step ${index === currentIndex ? "current" : index < currentIndex ? "done" : ""}`} key={label}>
                    <span className="approval-step__dot">{index < currentIndex ? "✓" : index === currentIndex ? "•" : index + 1}</span><span className="approval-step__label">{label}</span>
                  </li>
                );})}
              </ol>
            </div>
          </section>
        ) : null}

        {selected?.status === "RELEASED" ? <section aria-label="Canonical post-release preparation" className="planning-command-center management-panel" data-testid="canonical-preparation-actions">
          <header className="planning-command-center__head"><div><span className="planning-command-center__eyebrow">Post-release canonical workflow</span><h2>Prepare → Assign → Confirm → Materialize → Start</h2><p>These controls call server-owned commands. No inspection, package, or checklist exists until materialization.</p></div><span className="planning-demo-badge is-info">{workflow ? "Connected command boundary" : "Unavailable in this build profile"}</span></header>
          {!workflow ? <p role="note">Canonical preparation commands are unavailable in this build profile.</p> : <div className="planning-preparation-actions">
            <div><Link to={`/department-manager/planning/${encodeURIComponent(selected.id)}/setup/checklist`}>Open checklist selection</Link><p role="status">{packageSetup?.status === "FINALIZED" ? `Audit package finalized · ${packageSetup.selectedCount} checklist items` : "Checklist selection and Audit-package finalization are required before preparation."}</p></div>
            {!preparation ? <button disabled={busy || packageSetup?.status !== "FINALIZED"} onClick={() => void beginPreparation()} title={packageSetup?.status !== "FINALIZED" ? "Finalize the post-release checklist selection before beginning preparation." : undefined} type="button">Begin preparation</button> : <p role="status">{recordReference("Preparation", preparation.assignmentId)} · {preparation.status.replaceAll("_", " ")} · revision {preparation.revision}</p>}
            {preparation && !assignment ? <div><label>Lead Inspector subject ID<input aria-label="Lead Inspector subject ID" value={leadSubjectId} onChange={(event) => setLeadSubjectId(event.target.value)} placeholder="lead-subject" /></label><button disabled={busy || !leadSubjectId.trim()} onClick={() => void assignLead()} type="button">Assign Lead Inspector</button></div> : null}
            {assignment ? <><p role="status">{recordReference("Assignment", assignment.id)} · {assignment.status.replaceAll("_", " ")} · revision {assignment.revision}</p><p role="note">The assigned Lead Inspector completes team membership and per-question coverage in the Lead preparation workspace.</p><Link to={`/lead-inspector/audit-preparation?assignmentId=${encodeURIComponent(assignment.id)}`}>Open Lead preparation workspace</Link><button disabled={busy || preparationConfirmed || assignment.status !== "QUESTIONS_ASSIGNED"} onClick={() => void confirmPreparation()} title={preparationConfirmed ? "Preparation is already confirmed for this assignment revision." : assignment.status !== "QUESTIONS_ASSIGNED" ? "The Lead Inspector must complete team and question coverage first." : undefined} type="button">Confirm preparation</button><button disabled={busy || !preparationConfirmed} onClick={() => void materializeAudit()} title={!preparationConfirmed ? "Confirm preparation before materialization." : undefined} type="button">Create Audit</button></> : null}
            {materialized ? <p role="status">{recordReference("Audit", materialized.inspectionId)} · {materialized.status.replaceAll("_", " ")}. Inspector start is available from My Assignments after readiness.</p> : null}
          </div>}
        </section> : null}

        <section className="planning-queue management-panel">
          <div className="management-section-head"><div><span>Authorized register</span><h2>Planning Queue</h2></div><strong>{items.length} item</strong></div>
          <div className="management-table-scroll">
            <table aria-label="Planning Queue">
              <thead><tr><th>Planning Item</th><th>Organization</th><th>Type</th><th>Target</th><th>Budget</th><th>Owner</th><th>Status</th><th>Action</th></tr></thead>
              <tbody>{items.map((item) => (
                <tr className={selected?.id === item.id ? "is-selected" : ""} key={item.id}>
                  <td><b>{item.title}</b><small>{recordReference("Plan", item.id)}</small></td><td>{item.organizationName}</td><td>{item.inspectionType}</td><td>{formatLocalDate(item.scheduledDate)}</td>
                  <td>{new Intl.NumberFormat("en", { style: "currency", currency: "NAD", maximumFractionDigits: 0 }).format(item.estimatedBudget)}</td><td>{roleLabels[item.currentOwnerRole]}</td><td>{planningStatusLabel(item.status)}</td>
                  <td><button
                    aria-label={selected?.id === item.id ? `${planningItemLabel(item.title, item.id)} is already selected` : `Open ${planningItemLabel(item.title, item.id)}`}
                    disabled={selected?.id === item.id}
                    onClick={() => setSelected(item)}
                    title={selected?.id === item.id ? "This Planning item is already open in the Planning command center." : undefined}
                    type="button"
                  >
                    Open
                  </button></td>
                </tr>
              ))}</tbody>
            </table>
          </div>
          {selected ? <p className="planning-selected-record" data-testid="planning-selected-record">Selected plan: <b>{selected.title}</b> · {recordReference("Plan", selected.id)} · revision {selected.revision}</p> : null}
        </section>
      </div>
    </WorkspaceShell>
  );
}
