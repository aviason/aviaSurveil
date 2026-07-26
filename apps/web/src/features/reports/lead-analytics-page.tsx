import { useMemo, useState } from "react";

import { useApplicationRuntime } from "../../app/providers";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";

interface AdvisorySignal {
  organizationId: string;
  organizationName: string;
  mockRisk: number;
  overdue: boolean;
  repeat: boolean;
  owner: string;
  nextAction: string;
  blockingReason: string;
  drivers: readonly string[];
}

const signals = Object.freeze<readonly AdvisorySignal[]>([
  Object.freeze({ organizationId: "ORG-SKYCARGO", organizationName: "SkyCargo Air", mockRisk: 80, overdue: true, repeat: false, owner: "CAA Manager", nextAction: "Prioritize Evidence review and configured security follow-up", blockingReason: "Evidence remains pending on an overdue Finding.", drivers: ["Overdue Finding", "Evidence review pending"] }),
  Object.freeze({ organizationId: "ORG-FLY-NAMIBIA", organizationName: "Fly Namibia", mockRisk: 55, overdue: false, repeat: true, owner: "Lead Inspector", nextAction: "Review Cabin Inspection assignment workload", blockingReason: "Exact checklist questions still require workload confirmation.", drivers: ["Cabin Inspection scheduled", "Repeat indicator for review"] }),
]);

function csvCell(value: string | number | boolean): string {
  return `"${String(value).replaceAll("\"", "\"\"")}"`;
}

function analyticsCsv(rows: readonly AdvisorySignal[]): string {
  const header = ["Organization ID", "Organization", "Mock Risk", "Overdue", "Repeat", "Current Owner", "Next Action", "Blocking Reason"];
  const values = rows.map((signal) => [
    signal.organizationId,
    signal.organizationName,
    signal.mockRisk,
    signal.overdue,
    signal.repeat,
    signal.owner,
    signal.nextAction,
    signal.blockingReason,
  ]);
  return [header, ...values].map((row) => row.map(csvCell).join(",")).join("\n");
}

export function LeadAnalyticsPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("leadInspector") ?? runtime.backend, [runtime]);
  const [filter, setFilter] = useState<"all" | "overdue" | "repeat">("all");
  const [status, setStatus] = useState("");
  const [error, setError] = useState<string | null>(null);
  const visible = signals.filter((signal) => filter === "all" || (filter === "overdue" ? signal.overdue : signal.repeat));
  const csvHref = `data:text/csv;charset=utf-8,${encodeURIComponent(analyticsCsv(visible))}`;
  async function prepareCsv() {
    try {
      await backend.administration?.invokeVisibleAction({ screenId: "lead-analytics-reports", actionId: "download-analytics" });
      setStatus(`lead-analytics.csv prepared with ${visible.map((signal) => signal.organizationId).join(", ")}.`);
    } catch (cause) { setError(errorMessage(cause)); }
  }
  return <WorkspaceShell roleLabel="Lead Inspector" routeLabel="Lead Analytics & Reports">
    <div className="lead-secondary-page lead-analytics-page" data-testid="lead-analytics-page">
      <header className="lead-secondary-header workbench-page-header"><div><h1>Safety Intelligence Dashboard</h1><p>Decide which risk, delay, or workload issue needs management attention today.</p></div></header>
      <section aria-label="Lead management summary" className="lead-metric-grid lead-analytics-summary"><article><span>Open Findings</span><strong>2</strong></article><article><span>Overdue Findings</span><strong>1</strong></article><article><span>Pending CAP Reviews</span><strong>1</strong></article><article><span>Assigned Audits</span><strong>2</strong></article></section>
      <div className="lead-advisory-strip"><span>Demo data</span><span>Mock risk indicator</span><strong>Not a legal decision</strong><span>Frontend-only demo</span></div>
      <div aria-label="Analytics filters" className="lead-action-row lead-analytics-filters"><button aria-pressed={filter === "all"} className={filter === "all" ? "lead-button lead-button--primary" : "lead-button"} onClick={() => setFilter("all")} type="button">All signals</button><button aria-pressed={filter === "overdue"} className={filter === "overdue" ? "lead-button lead-button--primary" : "lead-button"} onClick={() => setFilter("overdue")} type="button">Overdue</button><button aria-pressed={filter === "repeat"} className={filter === "repeat" ? "lead-button lead-button--primary" : "lead-button"} onClick={() => setFilter("repeat")} type="button">Repeat</button></div>
      <CommandError message={error} />
      <section aria-label="Management attention" className="lead-panel lead-attention-command"><div><h2>Management attention</h2><p>Prioritize record-specific review; this advisory indicator cannot enforce, close, approve, issue, sign, or lock work.</p><div className="lead-attention-items">{visible.map((signal) => <article data-organization-id={signal.organizationId} key={signal.organizationId}><h3>{signal.organizationName}</h3><dl><div><dt>Current Owner</dt><dd>{signal.owner}</dd></div><div><dt>Next Action</dt><dd>{signal.nextAction}</dd></div><div><dt>Blocking Reason</dt><dd>{signal.blockingReason}</dd></div></dl></article>)}</div></div><a className="lead-button lead-button--primary" download="AviaSurveil360_Lead_Analytics.csv" href={csvHref} onClick={() => void prepareCsv()}>Download analytics CSV</a></section>
      {status ? <p className="lead-action-result" role="status">{status}</p> : null}
      <section aria-label="Management Signal Dossiers" className="lead-signal-list"><h2>Management Signal Dossiers</h2>{visible.map((signal) => <article data-organization-id={signal.organizationId} key={signal.organizationId}><div className="lead-risk-score"><small>Mock risk</small><strong>{signal.mockRisk}</strong></div><div><h3>{signal.organizationName}</h3><p><b>Recommended action:</b> {signal.nextAction}</p><p><b>Blocking reason:</b> {signal.blockingReason}</p><div>{signal.drivers.map((driver) => <span key={driver}>{driver}</span>)}</div></div><dl><div><dt>Current Owner</dt><dd>{signal.owner}</dd></div><div><dt>Organization</dt><dd>{signal.organizationId}</dd></div><div><dt>Overdue</dt><dd>{signal.overdue ? "Yes" : "No"}</dd></div></dl><button aria-label={`Risk profile unavailable for ${signal.organizationId}`} className="lead-button" disabled title={`Organization ${signal.organizationId} has no declared Lead Inspector risk-profile route.`} type="button">Profile unavailable</button></article>)}</section>
    </div>
  </WorkspaceShell>;
}
