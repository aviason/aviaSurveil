import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { AuditeeReleasedReportView, DocumentMetadataView } from "../../backend/backend";
import { CommandError, errorMessage, formatLocalDate, WorkspaceShell } from "../shared/workspace-shell";

function responseDueDate(report: AuditeeReleasedReportView): string {
  return report.responseDueDate ? formatLocalDate(report.responseDueDate) : "Not configured";
}

function publicComment(report: AuditeeReleasedReportView): string {
  return report.caaVisibleCommentState === "RECORDED" && report.caaVisibleComment
    ? report.caaVisibleComment
    : "No CAA-visible comment recorded";
}

function useAuditeeBackend() {
  const runtime = useApplicationRuntime();
  return useMemo(() => runtime.backendForRole?.("auditee") ?? runtime.backend, [runtime]);
}

function useReportDocuments(backend: ReturnType<typeof useAuditeeBackend>) {
  const [documents, setDocuments] = useState<DocumentMetadataView[]>([]);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    let cancelled = false;
    void backend.documents.list({}).then((page) => {
      if (!cancelled) setDocuments(page.items.filter((item) => item.kind === "REPORT"));
    }).catch((cause) => {
      if (!cancelled) setError(errorMessage(cause));
    });
    return () => { cancelled = true; };
  }, [backend]);
  return { documents, error };
}

function documentForReport(documents: DocumentMetadataView[], report: AuditeeReleasedReportView): DocumentMetadataView | null {
  return documents.find((document) =>
    document.id === report.reportVersionId ||
    document.documentVersionId === report.reportVersionId,
  ) ?? null;
}

function ReportDownload({
  report,
  document,
}: {
  report: AuditeeReleasedReportView;
  document: DocumentMetadataView | null;
}) {
  if (!document?.downloadUrl) {
    return <button disabled title="The server has not published a generated document for this immutable report version yet." type="button">Download unavailable</button>;
  }
  return <a download={document.downloadFileName ?? `${report.reportVersionId}.pdf`} href={document.downloadUrl} rel="noreferrer" target="_blank">Download {report.reportVersionId}</a>;
}

function AuditeeReportRegister({ kind }: { kind: AuditeeReleasedReportView["kind"] }) {
  const backend = useAuditeeBackend();
  const [reports, setReports] = useState<AuditeeReleasedReportView[]>([]);
  const [query, setQuery] = useState("");
  const [selectedPreliminary, setSelectedPreliminary] = useState<AuditeeReleasedReportView | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { documents, error: documentError } = useReportDocuments(backend);

  useEffect(() => {
    let cancelled = false;
    if (!backend.auditeeReports) return () => { cancelled = true; };
    void backend.auditeeReports.listReleased({ kind }).then((output) => {
      if (!cancelled) setReports(output.items);
    }).catch((cause) => {
      if (!cancelled) setError(errorMessage(cause));
    });
    return () => { cancelled = true; };
  }, [backend, kind]);

  const filtered = reports.filter((report) =>
    `${report.reportId} ${report.reportVersionId} ${report.auditId}`.toLowerCase().includes(query.trim().toLowerCase()),
  );
  const title = kind === "PRELIMINARY" ? "Preliminary Reports" : "Final Reports";
  const testId = kind === "PRELIMINARY" ? "auditee-preliminary-reports-page" : "auditee-final-reports-page";
  return <WorkspaceShell roleLabel="Auditee" routeLabel={title}>
    <div className="auditee-secondary-page auditee-reports-page" data-testid={testId}>
      <header className="auditee-secondary-head workbench-page-header"><div><span>Organization-scoped released artifacts</span><h1>{title}</h1><p>Only exact report versions issued and locked by the Executive Director are visible to this Auditee session.</p></div></header>
      <p className="auditee-safe-boundary">Report review and download never closes a Finding. Follow the displayed next action separately.</p>
      <CommandError message={error ?? documentError} />
      <section className="auditee-secondary-filters" aria-label={`${title} filters`}><label>Search reports<input value={query} onChange={(event) => setQuery(event.target.value)} /></label><label>Release stage<select aria-label="Release stage" disabled value="LOCKED"><option value="LOCKED">Issued and locked</option></select><small>Only the terminal LOCKED stage is available to Auditee.</small></label></section>
      <section className="auditee-report-register" aria-label={`${title} register`}>
        <div className="responsive-table-shell"><table><thead><tr><th>Report Version</th><th>Audit</th><th>Release</th><th>Response Due Date</th><th>CAA-visible comment</th><th>Action</th></tr></thead><tbody>{filtered.map((report) => <tr key={report.reportVersionId}><td><b>{report.reportVersionId}</b><small>{report.reportId} · Version {report.version}</small></td><td>{report.auditId}</td><td>{report.status}</td><td>{responseDueDate(report)}</td><td>{publicComment(report)}</td><td>{kind === "PRELIMINARY" ? <button onClick={() => setSelectedPreliminary(report)} type="button">Preview {report.reportVersionId}</button> : <Link to={`/auditee/reports/${encodeURIComponent(report.reportVersionId)}`}>Open report</Link>}</td></tr>)}</tbody></table></div>
        <div className="auditee-report-mobile-cards">{filtered.map((report) => <article key={report.reportVersionId}><h2>{report.reportVersionId}</h2><p>{report.auditId} · {report.status}</p><p>Response Due Date: {responseDueDate(report)}</p><p>CAA-visible comment: {publicComment(report)}</p>{kind === "PRELIMINARY" ? <button onClick={() => setSelectedPreliminary(report)} type="button">Preview report</button> : <Link to={`/auditee/reports/${encodeURIComponent(report.reportVersionId)}`}>Open report</Link>}</article>)}</div>
        {!filtered.length ? <p>No {title.toLowerCase()} are released for this Auditee organization.</p> : null}
      </section>
      {selectedPreliminary ? <section className="auditee-preliminary-preview" aria-label={`Preliminary Report preview ${selectedPreliminary.reportVersionId}`}><header><span>Server-owned preview</span><h2>{selectedPreliminary.reportVersionId}</h2><p>{selectedPreliminary.reportId} · {selectedPreliminary.auditId}</p></header><p>Response Due Date: {responseDueDate(selectedPreliminary)}</p><p>CAA-visible comment: {publicComment(selectedPreliminary)}</p><p>{selectedPreliminary.findingIds.length ? `Finding IDs: ${selectedPreliminary.findingIds.join(", ")}` : "No Finding is linked to this report version."}</p><ReportDownload report={selectedPreliminary} document={documentForReport(documents, selectedPreliminary)} /></section> : null}
    </div>
  </WorkspaceShell>;
}

export function AuditeePreliminaryReportsPage() { return <AuditeeReportRegister kind="PRELIMINARY" />; }
export function AuditeeFinalReportsPage() { return <AuditeeReportRegister kind="FINAL" />; }

export function AuditeeReportPreviewPage() {
  const backend = useAuditeeBackend();
  const { reportVersionId } = useParams<{ reportVersionId: string }>();
  const [report, setReport] = useState<AuditeeReleasedReportView | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { documents, error: documentError } = useReportDocuments(backend);

  useEffect(() => {
    let cancelled = false;
    if (!reportVersionId || !backend.auditeeReports) {
      setError("The report route does not contain a server-owned report version identity.");
      return () => { cancelled = true; };
    }
    void backend.auditeeReports.getReleased({ reportVersionId }).then((loaded) => {
      if (!cancelled) setReport(loaded);
    }).catch((cause) => {
      if (!cancelled) setError(errorMessage(cause));
    });
    return () => { cancelled = true; };
  }, [backend, reportVersionId]);

  const reportDocument = report ? documentForReport(documents, report) : null;
  return <WorkspaceShell roleLabel="Auditee" routeLabel="Final Report Preview">
    <div className="auditee-secondary-page auditee-report-preview-page" data-report-version-id={report?.reportVersionId} data-testid="auditee-report-preview-page">
      <header className="auditee-secondary-head auditee-preview-head workbench-page-header"><div><Link to="/auditee/final-reports">← Return to Final Reports</Link><span>{report?.organizationId ?? "Organization scope"} · server preview</span><h1>Report Preview</h1><p>{report?.reportVersionId ?? "Released report unavailable"}</p></div>{report ? <ReportDownload report={report} document={reportDocument} /> : <button disabled title="The server has not published a released report for this immutable report route." type="button">Download unavailable</button>}</header>
      <CommandError message={error ?? documentError} />
      {report ? <div className="auditee-report-preview-layout"><div aria-label="Report sections" className="auditee-report-sections"><a href="#auditee-report-summary">Executive Summary</a><a href="#auditee-report-overview">Inspection Overview</a><a href="#auditee-report-findings">Findings Overview</a></div><article><header><span>AviaSurveil360</span><h2>{report.reportId}</h2><p>{report.reportVersionId} · Version {report.version}</p></header><section id="auditee-report-summary"><h3>Executive Summary</h3><p>Issued and locked report shared with the authorized Auditee organization.</p></section><section id="auditee-report-overview"><h3>Inspection Overview</h3><p>{report.auditId} · {report.organizationId}</p></section><section id="auditee-report-findings"><h3>Findings Overview</h3><p>{report.findingIds.length ? report.findingIds.join(", ") : "No Finding is linked to this report version."}</p></section><footer><p>Response Due Date: {responseDueDate(report)}</p><p>CAA-visible comment: {publicComment(report)}</p><p>Report approval does not close a Finding.</p></footer></article></div> : <p>No released report is available for this exact server-owned route.</p>}
    </div>
  </WorkspaceShell>;
}
