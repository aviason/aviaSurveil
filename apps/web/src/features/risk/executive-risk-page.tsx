import { useEffect, useMemo, useState } from "react";

import { useBackendForRole } from "../../app/providers";
import type { FindingView } from "../../backend/backend";
import { CommandError, errorMessage, formatSeverity, WorkspaceShell } from "../shared/workspace-shell";
import { RiskHeatmap } from "./risk-heatmap";

export function ExecutiveRiskPage() {
  const backend = useBackendForRole("gm");
  const [findings, setFindings] = useState<FindingView[]>([]);
  const [filter, setFilter] = useState("all");
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    void backend.findings.list({ limit: 100 }).then((output) => setFindings(output.items)).catch((cause) => setError(errorMessage(cause)));
  }, [backend]);
  const visible = useMemo(() => findings.filter((finding) => filter === "all" || finding.dueState === filter), [filter, findings]);
  const high = findings.filter((finding) => ["LEVEL_1_CRITICAL", "LEVEL_2_MAJOR"].includes(finding.severity)).length;

  return <WorkspaceShell roleLabel="General Manager" routeLabel="Risk Dashboard">
    <div className="executive-workspace-page gm-risk-dashboard-page" data-testid="gm-risk-dashboard-page">
      <header className="authority-page-head workbench-page-header"><h1>Cross-Department Risk Dashboard</h1><p>Review aggregated department exposure without team, checklist, or lifecycle editing controls.</p></header>
      <CommandError message={error} />
      <section aria-label="Oversight Health indicators" className="gm-kpis"><article><span>Open Findings</span><strong>{findings.filter((finding) => finding.status !== "CLOSED").length}</strong></article><article><span>High Risk Findings</span><strong>{high}</strong></article><article><span>Overdue CAPs</span><strong>{findings.filter((finding) => finding.dueState === "OVERDUE").length}</strong></article><article><span>Repeat Findings</span><strong>{findings.filter((finding) => finding.repeatFinding).length}</strong></article><article><span>Oversight Health</span><strong>Indicator</strong></article></section>
      <RiskHeatmap className="gm-panel gm-risk-heat gm-risk-dashboard-heat" records={findings} />
      <section aria-label="Risk filters" className="executive-report-filter"><label>Due state<select onChange={(event) => setFilter(event.target.value)} value={filter}><option value="all">All</option><option value="OVERDUE">Overdue</option><option value="DUE_SOON">Due Soon</option><option value="NOT_DUE">Not Due</option></select></label></section>
      <section className="executive-panel"><header><div><span>Priority visibility</span><h2>Recent High-Risk Findings</h2></div></header><div className="responsive-table-shell"><table aria-label="Cross-department risk Findings"><thead><tr><th>Finding</th><th>Organization</th><th>Severity</th><th>Status</th><th>Due State</th><th>Action</th></tr></thead><tbody>{visible.map((finding) => <tr key={finding.id}><td><b>{finding.id}</b><small>{finding.title}</small></td><td>{finding.organizationName}</td><td>{formatSeverity(finding.severity)}</td><td>{finding.status}</td><td>{finding.dueState}</td><td><button aria-label={`Open risk Finding ${finding.id} unavailable`} disabled title={`Finding ${finding.id} has no declared General Manager detail route in Plan 1.`} type="button">Open unavailable</button></td></tr>)}</tbody></table></div></section>
      <p className="executive-risk-guardrail"><b>Management indicator only:</b> This decision-support projection does not make an automatic legal, enforcement, certificate, suspension, compliance, or closure decision.</p>
    </div>
  </WorkspaceShell>;
}
