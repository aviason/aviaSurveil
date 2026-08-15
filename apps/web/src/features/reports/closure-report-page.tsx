import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { DocumentMetadataView, FindingView, ReportVersionView } from "../../backend/backend";
import { CommandError, errorMessage, formatLocalDate, formatSeverity, WorkspaceShell } from "../shared/workspace-shell";

export function ClosureReportPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("inspector") ?? runtime.backend, [runtime]);
  const navigate = useNavigate();
  const { closureReportId } = useParams();
  const [report, setReport] = useState<ReportVersionView | null>(null);
  const [document, setDocument] = useState<DocumentMetadataView | null>(null);
  const [findings, setFindings] = useState<FindingView[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    if (!closureReportId) {
      setError("The report route does not contain a server-owned report version identity.");
      return () => { cancelled = true; };
    }
    void Promise.all([
      backend.reports.getVersion({ reportVersionId: closureReportId }),
      backend.documents.open({ documentId: closureReportId }),
      backend.findings.list({ limit: 100 }),
    ]).then(([nextReport, nextDocument, findingPage]) => {
      if (cancelled) return;
      setReport(nextReport);
      setDocument(nextDocument);
      setFindings(findingPage.items.filter((finding) => nextReport.findingIds.includes(finding.id)));
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend, closureReportId]);

  const lifecycleLabel = report?.status.replaceAll("_", " ") ?? "LOADING";
  return <WorkspaceShell roleLabel="CAA Inspector" routeLabel="Report Preview">
    <div className="inspector-secondary-page closure-report-page" data-report-version-id={report?.reportVersionId} data-testid="closure-report-page">
      <header className="inspector-secondary-head workbench-page-header"><div><h1>{report ? `Report ${report.reportId}` : "Report Preview"}</h1><p>Server-owned immutable report version · {report?.reportVersionId ?? "Loading"}</p></div><div className="inspector-secondary-actions"><button onClick={() => navigate("/inspector/reports")} type="button"><span>Back</span></button>{document?.downloadUrl ? <a className="is-active" href={document.downloadUrl} rel="noreferrer" target="_blank"><span>Open generated document</span></a> : <span title="The server has not made a generated document download available for this report version.">Generated document unavailable</span>}</div></header>
      <CommandError message={error} />
      {report ? <article className="closure-report-sheet">
        <header><span className="closure-report-mark sidebar-brand-mark" aria-hidden="true"><span className="sidebar-brand-mark__wing sidebar-brand-mark__wing--primary" /><span className="sidebar-brand-mark__wing sidebar-brand-mark__wing--secondary" /><span className="sidebar-brand-mark__code">AS</span></span><div><small>Civil Aviation Authority · AviaSurveil360</small><h2>Immutable Report Version</h2><p>{report.reportVersionId} · created {formatLocalDate(document?.createdAt?.slice(0, 10) ?? report.issuedAt?.slice(0, 10) ?? null)}</p></div><span>{lifecycleLabel}</span></header>
        <section><h3>Report Identity</h3><dl><dt>Report ID</dt><dd>{report.reportId}</dd><dt>Report version</dt><dd>{report.reportVersionId} · version {report.version}</dd><dt>Audit</dt><dd>{report.auditId}</dd><dt>Organization</dt><dd>{report.organizationId}</dd><dt>Content hash</dt><dd>{report.contentHash}</dd><dt>Revision</dt><dd>{report.revision}</dd></dl></section>
        <section><h3>Findings</h3>{findings.length ? <ul>{findings.map((finding) => <li key={finding.id}><Link to={`/inspector/findings/${encodeURIComponent(finding.id)}`}>{finding.findingNumber}</Link> · {finding.title} · {formatSeverity(finding.severity)} · {finding.status.replaceAll("_", " ")}</li>)}</ul> : <p>No Finding is linked to this report version.</p>}</section>
        <section><h3>Evidence and closure state</h3><p>Evidence review and Finding closure remain separate server-owned state transitions. This report view does not create, accept, or close either state.</p>{document ? <dl><dt>Document render</dt><dd>{document.renderStatus ?? "Not submitted"}</dd><dt>Document hash</dt><dd>{document.sha256 ?? "Not available"}</dd></dl> : null}</section>
        <section><h3>Decision history</h3><p>Current state: {lifecycleLabel}. Earlier immutable versions and decision events remain available through the audit trail.</p></section>
        <footer>This view renders server-owned report metadata; it is not a locally generated or signed legal document.</footer>
      </article> : null}
    </div>
  </WorkspaceShell>;
}
