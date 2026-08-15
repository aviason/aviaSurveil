import { useEffect, useMemo, useState } from "react";

import { useApplicationRuntime } from "../../app/providers";
import type { FindingView, OrganizationSummary, RiskOverviewView } from "../../backend/backend";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";

interface AdvisorySignal {
  organizationId: string;
  organizationName: string;
  riskScore: number | null;
  overdue: boolean;
  repeat: boolean;
  owner: string;
  nextAction: string;
  blockingReason: string;
  drivers: readonly string[];
}

function csvCell(value: string | number | boolean | null): string {
  return `"${String(value ?? "").replaceAll("\"", "\"\"")}"`;
}

function analyticsCsv(rows: readonly AdvisorySignal[]): string {
  const header = ["Organization ID", "Organization", "Advisory Risk", "Overdue", "Repeat", "Current Owner", "Next Action", "Blocking Reason"];
  const values = rows.map((signal) => [signal.organizationId, signal.organizationName, signal.riskScore, signal.overdue, signal.repeat, signal.owner, signal.nextAction, signal.blockingReason]);
  return [header, ...values].map((row) => row.map(csvCell).join(",")).join("\n");
}

function signalFor(organization: OrganizationSummary, overview: RiskOverviewView | null, findings: FindingView[]): AdvisorySignal {
  const scoped = findings.filter((finding) => finding.organizationId === organization.id);
  const active = scoped.find((finding) => finding.status !== "CLOSED") ?? null;
  return {
    organizationId: organization.id,
    organizationName: organization.legalName,
    riskScore: overview?.advisoryHealth?.score ?? null,
    overdue: (overview?.overdueFindingCount ?? 0) > 0,
    repeat: (overview?.repeatFindingCount ?? 0) > 0,
    owner: active?.currentOwnerRole ?? "No active Finding",
    nextAction: active?.nextAction ?? "Review the organization record and current planning state",
    blockingReason: active ? `Finding ${active.id} remains ${active.status}.` : "No active Finding record is available.",
    drivers: active ? [active.id, active.status] : ["No active Finding"],
  };
}

export function LeadAnalyticsPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("leadInspector") ?? runtime.backend, [runtime]);
  const [signals, setSignals] = useState<AdvisorySignal[]>([]);
  const [filter, setFilter] = useState<"all" | "overdue" | "repeat">("all");
  const [status, setStatus] = useState("");
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    let cancelled = false;
    void Promise.all([
      backend.organizations.list({ limit: 100 }),
      backend.findings.list({ limit: 100 }),
    ]).then(async ([organizations, findings]) => {
      const overviews = await Promise.all(organizations.items.map(async (organization) => [
        organization.id,
        await backend.risk.getOverview({ organizationId: organization.id }),
      ] as const));
      const overviewByOrganization = new Map(overviews);
      if (!cancelled) setSignals(organizations.items.map((organization) => signalFor(organization, overviewByOrganization.get(organization.id) ?? null, findings.items)));
    }).catch((cause) => {
      if (!cancelled) setError(errorMessage(cause));
    });
    return () => { cancelled = true; };
  }, [backend]);

  const visible = signals.filter((signal) => filter === "all" || (filter === "overdue" ? signal.overdue : signal.repeat));
  const csvHref = `data:text/csv;charset=utf-8,${encodeURIComponent(analyticsCsv(visible))}`;
  async function prepareCsv() {
    try {
      await backend.administration?.invokeVisibleAction({ screenId: "lead-analytics-reports", actionId: "download-analytics" });
      setStatus(`lead-analytics.csv prepared with ${visible.length} organization records.`);
    } catch (cause) { setError(errorMessage(cause)); }
  }
  return <WorkspaceShell roleLabel="Lead Inspector" routeLabel="Lead Analytics & Reports">
    <div className="lead-secondary-page lead-analytics-page" data-testid="lead-analytics-page">
      <header className="lead-secondary-header workbench-page-header"><div><h1>Safety Intelligence Dashboard</h1><p>Review server-owned risk, delay, and workload signals without creating an approval or closure decision.</p></div></header>
      <div aria-label="Analytics filters" className="lead-action-row lead-analytics-filters"><button aria-pressed={filter === "all"} className={filter === "all" ? "lead-button lead-button--primary" : "lead-button"} onClick={() => setFilter("all")} type="button">All signals</button><button aria-pressed={filter === "overdue"} className={filter === "overdue" ? "lead-button lead-button--primary" : "lead-button"} onClick={() => setFilter("overdue")} type="button">Overdue</button><button aria-pressed={filter === "repeat"} className={filter === "repeat" ? "lead-button lead-button--primary" : "lead-button"} onClick={() => setFilter("repeat")} type="button">Repeat</button></div>
      <CommandError message={error} />
      <section aria-label="Management attention" className="lead-panel lead-attention-command"><div><h2>Management attention</h2><p>Prioritize record-specific review; this advisory indicator cannot enforce, close, approve, issue, sign, or lock work.</p><div className="lead-attention-items">{visible.map((signal) => <article data-organization-id={signal.organizationId} key={signal.organizationId}><h3>{signal.organizationName}</h3><dl><div><dt>Current Owner</dt><dd>{signal.owner}</dd></div><div><dt>Next Action</dt><dd>{signal.nextAction}</dd></div><div><dt>Blocking Reason</dt><dd>{signal.blockingReason}</dd></div></dl></article>)}</div></div><a className="lead-button lead-button--primary" download="AviaSurveil360_Lead_Analytics.csv" href={csvHref} onClick={() => void prepareCsv()}>Download analytics CSV</a></section>
      {status ? <p className="lead-action-result" role="status">{status}</p> : null}
      <section aria-label="Management Signal Dossiers" className="lead-signal-list"><h2>Management Signal Dossiers</h2>{visible.map((signal) => <article data-organization-id={signal.organizationId} key={signal.organizationId}><div className="lead-risk-score"><small>Advisory risk</small><strong>{signal.riskScore ?? "—"}</strong></div><div><h3>{signal.organizationName}</h3><p><b>Recommended action:</b> {signal.nextAction}</p><p><b>Blocking reason:</b> {signal.blockingReason}</p><div>{signal.drivers.map((driver) => <span key={driver}>{driver}</span>)}</div></div><dl><div><dt>Current Owner</dt><dd>{signal.owner}</dd></div><div><dt>Organization</dt><dd>{signal.organizationId}</dd></div><div><dt>Overdue</dt><dd>{signal.overdue ? "Yes" : "No"}</dd></div></dl></article>)}{!visible.length ? <p>No server-owned signals match the current filter.</p> : null}</section>
    </div>
  </WorkspaceShell>;
}
