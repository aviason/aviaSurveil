import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { FindingView } from "../../backend/backend";
import { CommandError, errorMessage, formatLocalDate, formatSeverity, WorkspaceShell } from "../shared/workspace-shell";

export function ClosureReportPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("inspector") ?? runtime.backend, [runtime]);
  const navigate = useNavigate();
  const [status, setStatus] = useState("");
  const [finding, setFinding] = useState<FindingView | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    let cancelled = false;
    void Promise.all([
      backend.administration?.getScreenProjection({ screenId: "closure-report-preview" }) ?? Promise.resolve(null),
      backend.findings.get({ findingId: "FND-CAB-2026-001" }),
    ]).then(([, result]) => !cancelled && setFinding(result)).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend]);
  async function exportReport() {
    try {
      await backend.administration?.invokeVisibleAction({ screenId: "closure-report-preview", actionId: "download-closure-report" });
      setStatus("Mock report preview prepared for download. No real document was generated or stored.");
    } catch (cause) { setError(errorMessage(cause)); }
  }
  const isClosed = finding?.status === "CLOSED";
  const lifecycleLabel = finding ? finding.status.replaceAll("_", " ") : "DRAFT PREVIEW";
  const reportPreview = [
    "AviaSurveil360 demo report preview",
    "Report reference: CAB-2026-011",
    `Finding: ${finding?.findingNumber ?? "Pending Finding data"}`,
    `Organization: ${finding?.organizationName ?? "Pending Finding data"}`,
    `Status: ${lifecycleLabel}`,
    "This is a mock preview, not a legally issued document.",
  ].join("\n");
  const reportPreviewHref = `data:text/plain;charset=utf-8,${encodeURIComponent(reportPreview)}`;
  return <WorkspaceShell roleLabel="CAA Inspector" routeLabel="Report Preview">
    <div className="inspector-secondary-page closure-report-page" data-testid="closure-report-page">
      <header className="inspector-secondary-head workbench-page-header"><div><h1>Finding Report — CAB-2026-011</h1><p>Report preview (mock — not a legally issued document).</p></div><div className="inspector-secondary-actions"><button onClick={() => navigate("/inspector/reports")} type="button"><span>Back</span></button><a className="is-active" download="AviaSurveil360_CAB-2026-011_Report_Preview.txt" href={reportPreviewHref} onClick={() => void exportReport()}><span>Download report preview</span></a></div></header>
      <CommandError message={error} />
      {status ? <p className="inspector-action-result" role="status">{status}</p> : null}
      <article className="closure-report-sheet">
        <header><span className="closure-report-mark sidebar-brand-mark" aria-hidden="true"><span className="sidebar-brand-mark__wing sidebar-brand-mark__wing--primary" /><span className="sidebar-brand-mark__wing sidebar-brand-mark__wing--secondary" /><span className="sidebar-brand-mark__code">AS</span></span><div><small>Civil Aviation Authority · AviaSurveil360</small><h2>{isClosed ? "Finding Closure Report" : "Finding Report Draft"}</h2><p>Generated 15 Jun 2026 · Report reference CAB-2026-011</p></div><span>{lifecycleLabel}</span></header>
        <section><h3>Finding Summary</h3><dl><dt>Finding ID</dt><dd>{finding?.findingNumber ?? "Pending Finding data"}</dd><dt>Title</dt><dd>{finding?.title ?? "Pending Finding data"}</dd><dt>Organization</dt><dd>{finding?.organizationName ?? "Pending Finding data"}</dd><dt>Severity</dt><dd>{finding ? formatSeverity(finding.severity) : "Pending Finding data"}</dd><dt>Related audit</dt><dd>{finding?.auditId ?? "Pending Finding data"}</dd><dt>Regulatory reference</dt><dd>{finding?.regulatoryReference ?? finding?.findingBasis ?? "Pending Finding data"}</dd><dt>Issued</dt><dd>{formatLocalDate(finding?.issuedAt?.slice(0, 10) ?? null)}</dd><dt>Due Date</dt><dd>{formatLocalDate(finding?.dueDate ?? null)}</dd></dl></section>
        <section><h3>Corrective Action Plan</h3><dl><dt>Root cause</dt><dd>The equipment register and maintenance record were maintained in separate files.</dd><dt>Corrective action</dt><dd>Reconciled the sampled position against the current serviceability register.</dd><dt>Preventive action</dt><dd>Added a single register review before each cabin inspection.</dd><dt>Responsible</dt><dd>Fly Namibia Cabin Safety Manager</dd><dt>Target date</dt><dd>18 Jun 2026</dd></dl></section>
        <section><h3>Evidence</h3><dl><dt>Evidence</dt><dd>None recorded</dd></dl></section>
        <section><h3>Audit Trail</h3><p>No audit-log entries.</p></section>
        <footer>{isClosed ? "This is a demo closure report preview." : "This draft is not a closure report because the Finding remains open."} No real document is generated, stored or signed.</footer>
      </article>
    </div>
  </WorkspaceShell>;
}
