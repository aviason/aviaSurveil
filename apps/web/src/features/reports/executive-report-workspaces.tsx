import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { useBackendForRole } from "../../app/providers";
import type { AuditEventView, ReportVersionView } from "../../backend/backend";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";

const PRELIMINARY_VERSION_ID = "PR-2026-018-V1";
const FINAL_VERSION_ID = "RPT-CAB-2026-001-V1";

function pdfEscape(value: string): string {
  return value
    .replace(/[^\x20-\x7E]/g, " ")
    .replace(/\\/g, "\\\\")
    .replace(/\(/g, "\\(")
    .replace(/\)/g, "\\)");
}

function buildReportPdf(report: ReportVersionView): string {
  const lines = [
    "AviaSurveil360 Final Report",
    `Report ID: ${report.reportId}`,
    `Report Version ID: ${report.reportVersionId}`,
    `Version: ${report.version}`,
    `Revision: ${report.revision}`,
    `Audit: ${report.auditId}`,
    `Organization: ${report.organizationId}`,
    `Status: ${report.status}`,
    `Content hash: ${report.contentHash}`,
  ];
  let y = 792;
  const content = lines.map((line, index) => {
    const size = index === 0 ? 18 : 10;
    const command = `BT /F1 ${size} Tf 54 ${y} Td (${pdfEscape(line)}) Tj ET\n`;
    y -= index === 0 ? 24 : 16;
    return command;
  }).join("");
  const objects = [
    "<< /Type /Catalog /Pages 2 0 R >>",
    "<< /Type /Pages /Kids [5 0 R] /Count 1 >>",
    "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
    `<< /Length ${content.length} >>\nstream\n${content}endstream`,
    "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 842] /Resources << /Font << /F1 3 0 R >> >> /Contents 4 0 R >>",
  ];
  let pdf = "%PDF-1.4\n";
  const offsets = [0];
  objects.forEach((body, index) => {
    offsets[index + 1] = pdf.length;
    pdf += `${index + 1} 0 obj\n${body}\nendobj\n`;
  });
  const xref = pdf.length;
  pdf += `xref\n0 ${objects.length + 1}\n0000000000 65535 f \n`;
  for (let index = 1; index <= objects.length; index += 1) {
    pdf += `${String(offsets[index]).padStart(10, "0")} 00000 n \n`;
  }
  return `${pdf}trailer\n<< /Size ${objects.length + 1} /Root 1 0 R >>\nstartxref\n${xref}\n%%EOF`;
}

function reportLabel(report: ReportVersionView): string {
  return `${report.reportId} · Version ${report.version}`;
}

function useReports(role: "gm" | "executiveDirector", reportVersionIds: readonly string[]) {
  const backend = useBackendForRole(role);
  const [reports, setReports] = useState<ReportVersionView[]>([]);
  const [events, setEvents] = useState<Record<string, AuditEventView[]>>({});
  const [error, setError] = useState<string | null>(null);

  async function loadEvents(reportVersionId: string) {
    const output = await backend.auditTrail.list({ entityType: "report_version", entityId: reportVersionId });
    setEvents((current) => ({ ...current, [reportVersionId]: output.items }));
  }

  useEffect(() => {
    let cancelled = false;
    void Promise.all(reportVersionIds.map((reportVersionId) => backend.reports.getVersion({ reportVersionId }))).then(async (loaded) => {
      if (cancelled) return;
      setReports(loaded);
      await Promise.all(loaded.map((report) => loadEvents(report.reportVersionId)));
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  // reportVersionIds are module constants at every call site.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [backend]);

  return { backend, reports, setReports, events, loadEvents, error, setError };
}

function DecisionHistory({ events }: { events: readonly AuditEventView[] }) {
  if (!events.length) return <p>No recorded report decision yet.</p>;
  return <ol className="manager-report-history">{events.slice().reverse().map((event) => <li key={event.eventId}><b>{event.actorRole ?? "System"} · revision {event.entityRevision ?? "—"}</b><span>{event.beforeStatus} → {event.afterStatus}</span><p>{event.reason ?? "No reason recorded"}</p></li>)}</ol>;
}

export function GeneralManagerReportApprovalsPage() {
  const { backend, reports, setReports, events, loadEvents, error, setError } = useReports("gm", [PRELIMINARY_VERSION_ID, FINAL_VERSION_ID]);
  const [selectedId, setSelectedId] = useState("");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const pending = reports.filter((report) => report.status === "GM_REVIEW");
  const selected = selectedId
    ? reports.find((report) => report.reportVersionId === selectedId) ?? null
    : pending[0] ?? null;

  async function decide(decision: "FORWARD" | "RETURN") {
    if (!selected) return;
    if (!reason.trim()) { setError("General Manager report decision reason is required."); return; }
    setBusy(true); setError(null);
    try {
      const updated = await backend.reports.decide({
        operationId: `GM-REPORT-${decision}-${selected.reportVersionId}-${selected.revision}`,
        reportVersionId: selected.reportVersionId,
        expectedReportVersionRevision: selected.revision,
        decision,
        reason,
      });
      setSelectedId(updated.reportVersionId);
      setReports((current) => current.map((report) => report.reportVersionId === updated.reportVersionId ? updated : report));
      await loadEvents(updated.reportVersionId);
      setReason("");
    } catch (cause) { setError(errorMessage(cause)); } finally { setBusy(false); }
  }

  return <WorkspaceShell roleLabel="General Manager" routeLabel="Report Approvals">
    <div className="executive-workspace-page gm-report-approvals-page" data-testid="gm-report-approvals-page">
      <header className="authority-page-head workbench-page-header"><h1>Report Approvals</h1><p>Review Department Manager-approved Preliminary and Final Reports before forwarding them to Executive Director.</p></header>
      <CommandError message={error} />
      <section aria-label="General Manager report indicators" className="gm-kpis"><article><span>Awaiting GM Review</span><strong>{pending.length}</strong></article><article><span>Preliminary Reports</span><strong>{reports.filter((report) => report.reportId.startsWith("PR-")).length}</strong></article><article><span>Final Reports</span><strong>{reports.filter((report) => report.reportId.startsWith("RPT-")).length}</strong></article><article><span>Issue authority</span><strong>Executive only</strong></article><article><span>Versions</span><strong>Immutable</strong></article></section>
      <div className="gm-approval-workspace">
        <section className="executive-panel" aria-label="General Manager report queue"><header><div><span>Intermediate review queue</span><h2>Reports Awaiting GM Review</h2></div></header>{pending.length ? <div className="responsive-table-shell"><table><thead><tr><th>Report Version</th><th>Type</th><th>Audit</th><th>Status</th><th>Revision</th><th>Action</th></tr></thead><tbody>{pending.map((report) => <tr key={report.reportVersionId}><td><b>{report.reportVersionId}</b><small>{report.reportId}</small></td><td>{report.reportId.startsWith("PR-") ? "Preliminary Report" : "Final Report"}</td><td>{report.auditId}</td><td>{report.status}</td><td>{report.revision}</td><td><button aria-label={selected?.reportVersionId === report.reportVersionId ? `${report.reportVersionId} is already selected` : `Open ${report.reportVersionId}`} disabled={selected?.reportVersionId === report.reportVersionId} onClick={() => setSelectedId(report.reportVersionId)} title={selected?.reportVersionId === report.reportVersionId ? `${report.reportVersionId} is already open in the General Manager report dossier.` : undefined} type="button">Open {report.reportVersionId}</button></td></tr>)}</tbody></table></div> : <div className="executive-empty"><b>No report versions are at GM_REVIEW.</b><span>{reports.map((report) => `${report.reportVersionId} remains ${report.status}`).join("; ")}. Department Manager must forward a Preliminary Report before General Manager review; an EXECUTIVE_DIRECTOR_REVIEW report remains with Executive Director.</span></div>}</section>
        {selected ? <section aria-label={`Selected report ${selected.reportVersionId}`} className="gm-report-detail" data-report-revision={selected.revision} data-report-version-id={selected.reportVersionId}><header><div><span>Selected immutable report</span><h2>{reportLabel(selected)}</h2><p>{selected.reportVersionId} · {selected.auditId}</p></div><b data-testid="report-status">{selected.status}</b></header><dl><div><dt>Report Version ID</dt><dd>{selected.reportVersionId}</dd></div><div><dt>Revision</dt><dd>{selected.revision}</dd></div><div><dt>Content hash</dt><dd>{selected.contentHash}</dd></div><div><dt>Current stage</dt><dd>{selected.status}</dd></div></dl><section><h3>Review History</h3><DecisionHistory events={events[selected.reportVersionId] ?? []} /></section>{selected.status === "GM_REVIEW" ? <><label>General Manager report decision reason<textarea onChange={(event) => setReason(event.target.value)} value={reason} /></label><div className="gm-detail-actions"><button disabled={busy} onClick={() => void decide("RETURN")} type="button">Return {selected.reportVersionId} to Department Manager</button><button disabled={busy} onClick={() => void decide("FORWARD")} type="button">Forward {selected.reportVersionId} to Executive Director</button></div></> : <button aria-label={`General Manager decision unavailable for ${selected.reportVersionId}`} disabled title={`Report version ${selected.reportVersionId} is ${selected.status}; General Manager decision is unavailable.`} type="button">Decision unavailable</button>}<p className="gm-final-rule"><b>Intermediate review rule:</b> General Manager may return or forward this exact version. Executive Director alone may issue, mock-sign, share, or lock it. Finance does not participate in report approval.</p></section> : null}
      </div>
    </div>
  </WorkspaceShell>;
}

function ExecutiveReportReviewPage({ kind }: { kind: "preliminary" | "final" }) {
  const reportVersionId = kind === "preliminary" ? PRELIMINARY_VERSION_ID : FINAL_VERSION_ID;
  const { backend, reports, setReports, events, loadEvents, error, setError } = useReports("executiveDirector", [reportVersionId]);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const report = reports[0] ?? null;
  const eligibleReport = report && ["EXECUTIVE_DIRECTOR_REVIEW", "LOCKED"].includes(report.status) ? report : null;
  const isPreliminary = kind === "preliminary";

  async function decide() {
    if (!report) return;
    if (!reason.trim()) { setError("Executive Director report decision reason is required."); return; }
    setBusy(true); setError(null);
    try {
      const updated = await backend.reports.decide({
        operationId: `EXEC-REPORT-ISSUE_AND_LOCK-${report.reportVersionId}-${report.revision}`,
        reportVersionId: report.reportVersionId,
        expectedReportVersionRevision: report.revision,
        decision: "ISSUE_AND_LOCK",
        reason,
      });
      setReports([updated]);
      await loadEvents(updated.reportVersionId);
      setReason("");
    } catch (cause) { setError(errorMessage(cause)); } finally { setBusy(false); }
  }

  const pageName = isPreliminary ? "Preliminary Reports" : "Final Reports";
  const pageTestId = isPreliminary ? "executive-preliminary-reports-page" : "executive-final-reports-page";
  const selectedName = `Selected ${isPreliminary ? "Preliminary" : "Final"} Report ${reportVersionId}`;
  return <WorkspaceShell roleLabel="Executive Director" routeLabel={pageName}>
    <div className={`executive-workspace-page ${isPreliminary ? "executive-preliminary-report-page" : "executive-final-report-page"}`} data-testid={pageTestId}>
      <header className="authority-page-head workbench-page-header"><h1>{pageName}</h1><p>Review exact immutable report versions forwarded through Department Manager and General Manager approval.</p></header>
      <CommandError message={error} />
      <div className="authority-guardrails"><span>Executive Director issue authority</span><span>Mock approval mark — no real e-signature</span><span>{isPreliminary ? "No enforcement action on Preliminary Reports" : "No automatic enforcement, CAP acceptance, or Finding closure"}</span></div>
      <section aria-label={`${pageName} status summary`} className="executive-report-summary"><article><span>Total</span><b>{eligibleReport ? 1 : 0}</b></article><article><span>Pending</span><b>{eligibleReport?.status === "EXECUTIVE_DIRECTOR_REVIEW" ? 1 : 0}</b></article><article><span>Issued / Locked</span><b>{eligibleReport?.status === "LOCKED" ? 1 : 0}</b></article><article><span>Returned</span><b>0</b></article></section>
      {eligibleReport ? <section aria-label={selectedName} className="executive-report-review-layout" data-report-revision={eligibleReport.revision} data-report-version-id={eligibleReport.reportVersionId}><div className="executive-report-review-main"><section className="executive-plan-detail"><header><div><span>{selectedName}</span><h2>{reportLabel(eligibleReport)}</h2><p>{eligibleReport.auditId} · revision {eligibleReport.revision}</p></div><b data-testid="report-status">{eligibleReport.status}</b></header><dl className="executive-definition-grid"><div><dt>Report ID</dt><dd>{eligibleReport.reportId}</dd></div><div><dt>Version ID</dt><dd>{eligibleReport.reportVersionId}</dd></div><div><dt>Version</dt><dd>{eligibleReport.version}</dd></div><div><dt>Revision</dt><dd>{eligibleReport.revision}</dd></div><div><dt>Content hash</dt><dd>{eligibleReport.contentHash}</dd></div><div><dt>Finance report stage</dt><dd>Not applicable</dd></div></dl>{!eligibleReport.findingIds.length ? <p className="executive-report-boundary">No Findings linked — relationship unavailable for {eligibleReport.reportVersionId}</p> : <p>{eligibleReport.findingIds.length} linked Finding{eligibleReport.findingIds.length === 1 ? "" : "s"}</p>}<section><h3>Decision History</h3><DecisionHistory events={events[eligibleReport.reportVersionId] ?? []} /></section>{!isPreliminary ? <Link to="/executive-director/reports/RPT-CAB-2026-001" aria-label={`Preview ${eligibleReport.reportVersionId}`}>Preview Full Report</Link> : null}</section></div><aside><section className="executive-report-decision"><span>Executive Director decision</span><h2>{isPreliminary ? "Preliminary Report authority" : "Final Report authority"}</h2>{eligibleReport.status === "EXECUTIVE_DIRECTOR_REVIEW" ? <><label>Executive Director report decision reason<textarea onChange={(event) => setReason(event.target.value)} value={reason} /></label><button disabled={busy} onClick={() => void decide()} type="button">Issue and lock {eligibleReport.reportVersionId}</button><button aria-label={`Return ${eligibleReport.reportVersionId} unavailable`} disabled title={`Report version ${eligibleReport.reportVersionId} is at Executive Director Review; the typed Plan 1 command permits issue and lock only. Return authority is not declared for Executive Director.`} type="button">Return unavailable</button></> : <button aria-label={`Executive decision unavailable for ${eligibleReport.reportVersionId}`} disabled title={`Report version ${eligibleReport.reportVersionId} is ${eligibleReport.status}; no further Executive Director decision is available.`} type="button">Decision recorded</button>}<p>Report approval never closes a Finding and never applies enforcement.</p></section></aside></section> : <section className="executive-empty"><b>No report version is eligible for Executive Director review.</b><span>{report?.status === "DEPARTMENT_REVIEW" ? `${report.reportVersionId} is DEPARTMENT_REVIEW. Department Manager must forward the exact version, then General Manager must forward it before Executive Director review.` : report?.status === "GM_REVIEW" ? `${report.reportVersionId} is GM_REVIEW. General Manager must forward the exact version before Executive Director review.` : report ? `${report.reportVersionId} is ${report.status}; the current source stage must complete its next action before Executive Director review.` : "The exact report version is unavailable."}</span></section>}
    </div>
  </WorkspaceShell>;
}

export function ExecutivePreliminaryReportsPage() { return <ExecutiveReportReviewPage kind="preliminary" />; }
export function ExecutiveFinalReportsPage() { return <ExecutiveReportReviewPage kind="final" />; }

export function ExecutiveReportPreviewPage() {
  const backend = useBackendForRole("executiveDirector");
  const [report, setReport] = useState<ReportVersionView | null>(null);
  const [zoom, setZoom] = useState(100);
  const [status, setStatus] = useState("");
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    void backend.reports.getVersion({ reportVersionId: FINAL_VERSION_ID }).then(setReport).catch((cause) => setError(errorMessage(cause)));
  }, [backend]);
  const facts = useMemo(() => report ? [report.reportVersionId, report.reportId, report.auditId, `Version ${report.version}`, `Revision ${report.revision}`, report.contentHash] : [], [report]);
  function downloadReport(): void {
    if (!report) return;
    const fileName = `${report.reportId}.pdf`;
    const url = URL.createObjectURL(new Blob([buildReportPdf(report)], { type: "application/pdf" }));
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = fileName;
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
    URL.revokeObjectURL(url);
    setStatus(`${fileName} downloaded for immutable report version ${report.reportVersionId}.`);
  }
  return <WorkspaceShell roleLabel="Executive Director" routeLabel="Final Report Preview">
    <div className="executive-report-preview-page" data-report-version-id={report?.reportVersionId} data-testid="executive-report-preview-page">
      <header className="executive-report-preview-head workbench-page-header"><div><Link to="/executive-director/final-reports">← Return to Final Report review</Link><span>Selected report · {report?.reportId ?? "Loading"}</span><h1>Final Report Preview</h1><p>State-backed browser preview · sample page 1</p></div><div className="executive-preview-actions"><div aria-label="Preview zoom" className="executive-zoom-control"><span>Zoom</span>{[75, 90, 100, 110].map((value) => <button aria-pressed={zoom === value} className={zoom === value ? "is-active" : ""} key={value} onClick={() => setZoom(value)} type="button">{value}%</button>)}</div><button aria-label={`Print ${FINAL_VERSION_ID} unavailable`} disabled title={`Printing ${FINAL_VERSION_ID} is unavailable in the frontend-only demo.`} type="button">Print unavailable</button><button disabled={!report} onClick={downloadReport} type="button">Download PDF</button></div></header>
      <CommandError message={error} />
      {status ? <p role="status">{status}</p> : null}
      {report ? <div className="executive-report-preview-layout"><aside className="executive-report-contents"><span>Contents</span><a href="#report-summary">1. Executive Summary</a><a href="#report-overview">2. Inspection Overview</a><a href="#report-findings">3. Findings Overview</a><div><b>Document facts</b>{facts.map((fact) => <small key={fact}>{fact}</small>)}</div></aside><div className="executive-report-canvas"><article className="executive-report-document" style={{ transform: `scale(${zoom / 100})`, transformOrigin: "top center" }}><header><span>AviaSurveil360</span><h2>Final Report</h2><p>{report.reportVersionId}</p></header><section id="report-summary"><h3>Executive Summary</h3><p>Immutable candidate report version prepared for Executive Director review.</p></section><section id="report-overview"><h3>Inspection Overview</h3><p>{report.auditId} · {report.organizationId}</p></section><section id="report-findings"><h3>Findings Overview</h3><p>{report.findingIds.length ? report.findingIds.join(", ") : `No Findings linked — relationship unavailable for ${report.reportVersionId}`}</p></section><footer><b>Status</b><span>{report.status}</span><b>Content hash</b><span>{report.contentHash}</span></footer></article></div></div> : null}
    </div>
  </WorkspaceShell>;
}
