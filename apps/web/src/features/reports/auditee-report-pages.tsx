import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { AuditeeReleasedReportView } from "../../backend/backend";
import { CommandError, errorMessage, formatLocalDate, WorkspaceShell } from "../shared/workspace-shell";

const FINAL_REPORT_VERSION_ID = "RPT-CAB-2026-001-V1";
const FINAL_REPORT_ID = "RPT-CAB-2026-001";
const FINAL_REPORT_ROUTE = "/auditee/reports/RPT-CAB-2026-001";

function responseDueDate(report: AuditeeReleasedReportView): string {
  return report.responseDueDate ? formatLocalDate(report.responseDueDate) : "Not configured";
}

function publicComment(report: AuditeeReleasedReportView): string {
  return report.caaVisibleCommentState === "RECORDED" && report.caaVisibleComment
    ? report.caaVisibleComment
    : "No CAA-visible comment recorded";
}

function downloadReport(report: AuditeeReleasedReportView, onDone: (status: string) => void) {
  const fileName = `${report.reportId}.pdf`;
  const body = [
    "AviaSurveil360 browser-local demo report",
    `Report: ${report.reportId}`,
    `Version: ${report.reportVersionId}`,
    `Audit: ${report.auditId}`,
    `Organization: ${report.organizationId}`,
    `Status: ${report.status}`,
  ].join("\n");
  const url = URL.createObjectURL(new Blob([body], { type: "application/pdf" }));
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = fileName;
  document.body.appendChild(anchor);
  anchor.click();
  document.body.removeChild(anchor);
  URL.revokeObjectURL(url);
  onDone(`${fileName} downloaded for exact immutable version ${report.reportVersionId}. Browser-local demo artifact only.`);
}

function finalReportAction(report: AuditeeReleasedReportView, mobile: boolean) {
  const isCanonicalContext = report.reportId === FINAL_REPORT_ID
    && report.reportVersionId === FINAL_REPORT_VERSION_ID;
  if (isCanonicalContext) {
    return <Link
      aria-label={`${mobile ? "Open mobile" : "Open"} ${report.reportVersionId}`}
      to={FINAL_REPORT_ROUTE}
    >Open report</Link>;
  }
  const reason = `${report.reportVersionId} cannot open the contextual route because it is reserved for ${FINAL_REPORT_VERSION_ID}.`;
  return <span className="auditee-report-action-unavailable">
    <button aria-label={`Open ${report.reportVersionId} unavailable`} disabled title={reason} type="button">Open unavailable</button>
    <small>{reason}</small>
  </span>;
}

function AuditeeReportRegister({ kind }: { kind: AuditeeReleasedReportView["kind"] }) {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("auditee") ?? runtime.backend, [runtime]);
  const [reports, setReports] = useState<AuditeeReleasedReportView[]>([]);
  const [query, setQuery] = useState("");
  const [selectedPreliminary, setSelectedPreliminary] = useState<AuditeeReleasedReportView | null>(null);
  const [status, setStatus] = useState("");
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    if (!backend.auditeeReports) return;
    let cancelled = false;
    void backend.auditeeReports.listReleased({ kind }).then((output) => {
      if (!cancelled) setReports(output.items);
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend, kind]);
  const filtered = reports.filter((report) => `${report.reportId} ${report.reportVersionId} ${report.auditId}`.toLowerCase().includes(query.toLowerCase()));
  const title = kind === "PRELIMINARY" ? "Preliminary Reports" : "Final Reports";
  const testId = kind === "PRELIMINARY" ? "auditee-preliminary-reports-page" : "auditee-final-reports-page";
  return <WorkspaceShell roleLabel="Auditee — Fly Namibia" routeLabel={title}>
    <div className="auditee-secondary-page auditee-reports-page" data-testid={testId}>
      <header className="auditee-secondary-head workbench-page-header"><div><span>Fly Namibia · released artifacts</span><h1>{title}</h1><p>Only exact report versions issued and locked by the Executive Director are visible.</p></div></header>
      <p className="auditee-safe-boundary">Report review and download never closes a Finding. Follow the displayed next action separately.</p>
      <CommandError message={error} />
      {status ? <p className="auditee-action-result" role="status">{status}</p> : null}
      <section className="auditee-secondary-filters" aria-label={`${title} filters`}><label>Search reports<input value={query} onChange={(event) => setQuery(event.target.value)} /></label><label>Release stage<select aria-describedby={`auditee-${kind.toLowerCase()}-release-stage-reason`} aria-label="Release stage" disabled value="LOCKED"><option value="LOCKED">Issued and locked</option></select><small id={`auditee-${kind.toLowerCase()}-release-stage-reason`}>Only the terminal LOCKED stage is available to Fly Namibia Auditee.</small></label></section>
      <section className="auditee-report-register" aria-label={`${title} register`}>
        <div className="responsive-table-shell"><table><thead><tr><th>Report Version</th><th>Audit</th><th>Release</th><th>Response Due Date</th><th>CAA-visible comment</th><th>Action</th></tr></thead><tbody>{filtered.map((report) => <tr key={report.reportVersionId}><td><b>{report.reportVersionId}</b><small>{report.reportId} · Version {report.version}</small></td><td>{report.auditId}</td><td>{report.status}</td><td>{responseDueDate(report)}</td><td>{publicComment(report)}</td><td>{kind === "PRELIMINARY" ? <button onClick={() => setSelectedPreliminary(report)} type="button">Preview {report.reportVersionId}</button> : finalReportAction(report, false)}</td></tr>)}</tbody></table></div>
        <div className="auditee-report-mobile-cards">{filtered.map((report) => <article key={report.reportVersionId}><h2>{report.reportVersionId}</h2><p>{report.auditId} · {report.status}</p><p>Response Due Date: {responseDueDate(report)}</p><p>CAA-visible comment: {publicComment(report)}</p>{kind === "PRELIMINARY" ? <button aria-label={`Preview mobile ${report.reportVersionId}`} onClick={() => setSelectedPreliminary(report)} type="button">Preview report</button> : finalReportAction(report, true)}</article>)}</div>
        {!filtered.length ? <p>No {title.toLowerCase()} are released for Fly Namibia.</p> : null}
      </section>
      {selectedPreliminary ? <section className="auditee-preliminary-preview" aria-label={`Preliminary Report preview ${selectedPreliminary.reportVersionId}`}><header><span>Browser-local preview</span><h2>{selectedPreliminary.reportVersionId}</h2><p>{selectedPreliminary.reportId} · {selectedPreliminary.auditId}</p></header><p>Response Due Date: {responseDueDate(selectedPreliminary)}</p><p>CAA-visible comment: {publicComment(selectedPreliminary)}</p><p>No Findings linked — relationship unavailable for {selectedPreliminary.reportVersionId}</p><button onClick={() => downloadReport(selectedPreliminary, setStatus)} type="button">Download {selectedPreliminary.reportVersionId}</button></section> : null}
    </div>
  </WorkspaceShell>;
}

export function AuditeePreliminaryReportsPage() { return <AuditeeReportRegister kind="PRELIMINARY" />; }
export function AuditeeFinalReportsPage() { return <AuditeeReportRegister kind="FINAL" />; }

export function AuditeeReportPreviewPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("auditee") ?? runtime.backend, [runtime]);
  const [report, setReport] = useState<AuditeeReleasedReportView | null>(null);
  const [status, setStatus] = useState("");
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    if (!backend.auditeeReports) return;
    let cancelled = false;
    void backend.auditeeReports.getReleased({ reportVersionId: FINAL_REPORT_VERSION_ID }).then((loaded) => {
      if (!cancelled) setReport(loaded);
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend]);
  return <WorkspaceShell roleLabel="Auditee — Fly Namibia" routeLabel="Final Report Preview">
    <div className="auditee-secondary-page auditee-report-preview-page" data-report-version-id={report?.reportVersionId} data-testid="auditee-report-preview-page">
      <header className="auditee-secondary-head auditee-preview-head workbench-page-header"><div><Link to="/auditee/final-reports">← Return to Final Reports</Link><span>Fly Namibia · browser-local preview</span><h1>Final Report</h1><p>{report?.reportVersionId ?? "Released report unavailable"}</p></div>{report ? <button onClick={() => downloadReport(report, setStatus)} type="button">Download {report.reportVersionId}</button> : <button disabled title={`${FINAL_REPORT_VERSION_ID} is unavailable until Executive Director issue and lock.`} type="button">Download unavailable</button>}</header>
      <CommandError message={error} />
      {status ? <p className="auditee-action-result" role="status">{status}</p> : null}
      {report ? <div className="auditee-report-preview-layout"><div aria-label="Report sections" className="auditee-report-sections"><a href="#auditee-report-summary">Executive Summary</a><a href="#auditee-report-overview">Inspection Overview</a><a href="#auditee-report-findings">Findings Overview</a></div><article><header><span>AviaSurveil360</span><h2>{report.reportId}</h2><p>{report.reportVersionId} · Version {report.version}</p></header><section id="auditee-report-summary"><h3>Executive Summary</h3><p>Issued and locked report shared with Fly Namibia.</p></section><section id="auditee-report-overview"><h3>Inspection Overview</h3><p>{report.auditId} · {report.organizationId}</p></section><section id="auditee-report-findings"><h3>Findings Overview</h3><p>{report.findingIds.length ? report.findingIds.join(", ") : `No Findings linked — relationship unavailable for ${report.reportVersionId}`}</p></section><footer><p>Response Due Date: {responseDueDate(report)}</p><p>CAA-visible comment: {publicComment(report)}</p><p>Report approval does not close a Finding.</p></footer></article></div> : <p>No released Final Report is available for this exact route.</p>}
    </div>
  </WorkspaceShell>;
}
