import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { DocumentMetadataView, FindingView, ReportApprovalStatus, ReportVersionView } from "../../backend/backend";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";
import { discoverCAAReportVersions, type ReportKind } from "./report-discovery";

function reportStatusLabel(status: ReportApprovalStatus): string {
  return status.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function LeadReportShell({ routeLabel, children }: { routeLabel: string; children: React.ReactNode }) {
  return <WorkspaceShell roleLabel="Lead Inspector" routeLabel={routeLabel}>{children}</WorkspaceShell>;
}

function useLeadBackend() {
  const runtime = useApplicationRuntime();
  return useMemo(() => runtime.backendForRole?.("leadInspector") ?? runtime.backend, [runtime]);
}

function useReportVersion(routeReportVersionId: string | undefined) {
  const backend = useLeadBackend();
  const [report, setReport] = useState<ReportVersionView | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    let cancelled = false;
    setReport(null);
    if (!routeReportVersionId) {
      setError("The report route does not contain a report version identity.");
      return () => { cancelled = true; };
    }
    setError(null);
    void backend.reports.getVersion({ reportVersionId: routeReportVersionId }).then((loaded) => {
      if (!cancelled) setReport(loaded);
    }).catch((cause) => {
      if (!cancelled) setError(errorMessage(cause));
    });
    return () => { cancelled = true; };
  }, [backend, routeReportVersionId]);
  return { backend, report, error };
}

function useReportFindings(report: ReportVersionView | null) {
  const backend = useLeadBackend();
  const [findings, setFindings] = useState<FindingView[]>([]);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    let cancelled = false;
    if (!report) {
      setFindings([]);
      return () => { cancelled = true; };
    }
    void backend.findings.list({ limit: 100 }).then(({ items }) => {
      if (!cancelled) setFindings(items.filter((finding) => report.findingIds.includes(finding.id)));
    }).catch((cause) => {
      if (!cancelled) setError(errorMessage(cause));
    });
    return () => { cancelled = true; };
  }, [backend, report]);
  return { findings, error };
}

function reportMatches(report: ReportVersionView, query: string, status: "all" | ReportApprovalStatus): boolean {
  const haystack = [report.reportVersionId, report.reportId, report.auditId, report.organizationId].join(" ").toLowerCase();
  return haystack.includes(query.trim().toLowerCase()) && (status === "all" || report.status === status);
}

function useReportList(kind: ReportKind) {
  const backend = useLeadBackend();
  const [reports, setReports] = useState<ReportVersionView[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loaded, setLoaded] = useState(false);
  useEffect(() => {
    let cancelled = false;
    void discoverCAAReportVersions(backend, [kind]).then((loadedReports) => {
      if (!cancelled) setReports(loadedReports);
    }).catch((cause) => {
      if (!cancelled) setError(errorMessage(cause));
    }).finally(() => {
      if (!cancelled) setLoaded(true);
    });
    return () => { cancelled = true; };
  }, [backend, kind]);
  return { reports, error, loaded };
}

function opaqueReportIdentity(prefix: string): string {
  return `${prefix}:${globalThis.crypto.randomUUID()}`;
}

function finalReportContent(auditId: string, findings: readonly FindingView[]) {
  return {
    schema: "avia.report-content/v1",
    languageTag: "en",
    title: "Final Inspection Report",
    executiveSummary: "The canonical inspection lifecycle is complete and all linked Findings have passed Evidence verification and closure.",
    scope: `Canonical inspection ${auditId}`,
    methodology: "The released question set, governed Preliminary Report, CAP review, Evidence scan/review, and authorized closure records were evaluated in sequence.",
    sections: [
      {
        id: "lifecycle-summary",
        heading: "Inspection lifecycle",
        paragraphs: ["This Final Report is prepared from immutable server-owned lifecycle records after every linked Finding reached CLOSED."],
      },
    ],
    findings: findings.map((finding) => ({
      findingId: finding.id,
      reference: finding.findingNumber,
      title: finding.title,
      narrative: finding.findingBasis || finding.description,
      regulatoryBasis: finding.regulatoryReference ? [finding.regulatoryReference] : [],
    })),
    conclusion: "The canonical inspection record is complete for Final Report approval.",
    recommendations: ["Retain the immutable inspection, CAP, Evidence, closure, and report artifacts under the released record."],
  };
}

function FinalReportCreator() {
  const backend = useLeadBackend();
  const [closedFindings, setClosedFindings] = useState<FindingView[]>([]);
  const [auditId, setAuditId] = useState("");
  const [created, setCreated] = useState<ReportVersionView | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    void backend.findings.list({ status: "CLOSED", limit: 100 }).then(({ items }) => {
      if (!cancelled) {
        setClosedFindings(items);
      }
    }).catch((cause) => {
      if (!cancelled) setError(errorMessage(cause));
    });
    return () => { cancelled = true; };
  }, [backend]);

  const auditOptions = useMemo(() => [...new Set(closedFindings.map((finding) => finding.auditId))].sort(), [closedFindings]);
  const selectedFindings = closedFindings.filter((finding) => finding.auditId === auditId);

  async function create(): Promise<void> {
    if (!auditId || !selectedFindings.length) {
      setError("Select an exact Audit with at least one closed Finding.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const reportVersionId = opaqueReportIdentity("report-version");
      const report = await backend.reports.create({
        operationId: `CREATE-FINAL-${reportVersionId}`,
        idempotencyKey: `CREATE-FINAL-${reportVersionId}`,
        expectedRevision: null,
        reportVersionId,
        reportId: opaqueReportIdentity("report"),
        auditId,
        kind: "FINAL",
        version: 1,
        status: "DEPARTMENT_REVIEW",
        findingIds: selectedFindings.map((finding) => finding.id),
        content: finalReportContent(auditId, selectedFindings),
      });
      setCreated(report);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  return <section aria-label="Create Final Report" className="lead-panel lead-final-report-creator" data-testid="final-report-creator">
    <h2>Prepare Final Report</h2>
    <p>Only server-owned closed Findings are eligible. The Final Report remains in Department Review until the declared approval chain completes.</p>
    <CommandError message={error} />
    <label>Closed Audit<select aria-label="Final Report audit" disabled={busy || !auditOptions.length} onChange={(event) => setAuditId(event.target.value)} value={auditId}><option value="">Select an exact closed Audit…</option>{auditOptions.map((option) => <option key={option} value={option}>{option}</option>)}</select></label>
    <p>{selectedFindings.length} closed Finding{selectedFindings.length === 1 ? "" : "s"} selected for {auditId || "the exact Audit"}.</p>
    <button className="lead-button lead-button--primary" disabled={busy || !selectedFindings.length} onClick={() => void create()} title={selectedFindings.length ? undefined : "Select at least one closed Finding before creating a Final Report."} type="button">Create Final Report</button>
    {created ? <article aria-label="Created Final Report" className="lead-report-creation-receipt" data-testid="created-final-report"><h3>Final Report created</h3><p data-testid="final-report-version-id">{created.reportVersionId}</p><span data-testid="final-report-status">{created.status}</span></article> : null}
  </section>;
}

export function LeadPreliminaryReportsPage() {
  const { reports, error, loaded } = useReportList("PRELIMINARY");
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<"all" | ReportApprovalStatus>("all");
  const visible = reports.filter((report) => reportMatches(report, query, status));
  return <LeadReportShell routeLabel="Lead Preliminary Reports">
    <div className="lead-secondary-page lead-preliminary-page" data-testid="lead-preliminary-reports-page">
      <header className="lead-secondary-header workbench-page-header"><div><h1>Preliminary Reports</h1><p>Server-owned Preliminary Report versions available to the Lead Inspector.</p></div></header>
      <CommandError message={error} />
      <section className="lead-metric-grid" aria-label="Preliminary Report summary"><article><span>Department Review</span><strong>{reports.filter((report) => report.status === "DEPARTMENT_REVIEW").length}</strong></article><article><span>Returned</span><strong>{reports.filter((report) => report.status === "RETURNED").length}</strong></article><article><span>Issued</span><strong>{reports.filter((report) => ["ISSUED", "LOCKED"].includes(report.status)).length}</strong></article><article><span>Total</span><strong>{reports.length}</strong></article></section>
      <div className="lead-filter-row"><label>Search reports<input type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search report, audit, or organization" /></label><label>Status<select value={status} onChange={(event) => setStatus(event.target.value as typeof status)}><option value="all">All statuses</option>{(["DEPARTMENT_REVIEW", "RETURNED", "ISSUED", "LOCKED"] as const).map((value) => <option key={value} value={value}>{reportStatusLabel(value)}</option>)}</select></label></div>
      <section className="lead-record-list" aria-label="Preliminary Report versions">{visible.map((report) => <article aria-label={`Preliminary Report ${report.reportVersionId}`} className="lead-report-row" data-report-id={report.reportId} data-report-version={report.version} key={report.reportVersionId}><div><small>Report version</small><strong>{report.reportVersionId}</strong><span>{report.reportId} · {report.auditId}</span></div><div><small>Organization</small><strong>{report.organizationId}</strong></div><div><small>Status</small><strong className={`lead-state lead-state--${report.status.toLowerCase()}`}>{reportStatusLabel(report.status)}</strong><span>Server-owned approval state</span></div><div><small>Version</small><strong>{report.version}</strong><span>{report.issuedAt ?? "Not issued"}</span></div><Link className="lead-button lead-button--primary" to={`/lead-inspector/preliminary-reports/${encodeURIComponent(report.reportVersionId)}`}>Open report package</Link></article>)}{loaded && !error && visible.length === 0 ? <p>No matching Preliminary Report versions are available.</p> : null}</section>
    </div>
  </LeadReportShell>;
}

export function LeadPreliminaryReportWorkflowPage() {
  const { reportId: routeReportId } = useParams<{ reportId: string }>();
  const { report, error } = useReportVersion(routeReportId);
  const { findings, error: findingError } = useReportFindings(report);
  const [preview, setPreview] = useState(false);
  return <LeadReportShell routeLabel="Lead Preliminary Reports"><div className="lead-secondary-page lead-preliminary-workflow" data-audit-id={report?.auditId} data-report-id={report?.reportId} data-testid="lead-preliminary-report-workflow-page"><header className="lead-secondary-header workbench-page-header"><div><p className="lead-breadcrumb">Preliminary Reports › {report?.reportVersionId ?? "Loading"}</p><h1>Preliminary Report</h1><span className="lead-state">{report ? reportStatusLabel(report.status) : "Loading"}</span></div></header><CommandError message={error ?? findingError} />{report ? <><section className="lead-fact-strip" aria-label="Preliminary Report identity"><div><small>Report</small><strong>{report.reportId}</strong><span>{report.reportVersionId}</span></div><div><small>Audit</small><strong>{report.auditId}</strong></div><div><small>Organization</small><strong>{report.organizationId}</strong></div><div><small>Version</small><strong>{report.version}</strong></div><div><small>Content hash</small><strong>{report.contentHash}</strong></div></section><section aria-label="Preliminary Report findings" className="lead-panel lead-step-workspace"><h2>Linked Findings ({findings.length})</h2>{findings.length ? findings.map((finding) => <article aria-label={`Finding ${finding.id}`} className="lead-finding-record" data-finding-id={finding.id} key={finding.id}><strong>{finding.id}</strong><span>{finding.findingNumber}</span><h3>{finding.title}</h3><p>{finding.findingBasis}</p><dl className="lead-detail-grid"><div><dt>Status</dt><dd>{finding.status}</dd></div><div><dt>Current Owner</dt><dd>{finding.currentOwnerRole}</dd></div><div><dt>Next Action</dt><dd>{finding.nextAction}</dd></div></dl></article>) : <p>No Finding is linked to this report version.</p>}<p>Report review does not close a Finding.</p></section><section className="lead-comment-grid"><article aria-label="Immutable report content"><h2>Immutable content</h2><p>The report content is server-owned and identified by {report.contentHash}. This surface does not create a browser-local Draft.</p></article><article aria-label="Report decision boundary"><h2>Decision boundary</h2><p>The current approval-stage owner must decide this version. Lead Inspector preparation remains read-only.</p></article></section><div className="lead-action-row"><button className="lead-button lead-button--primary" onClick={() => setPreview((value) => !value)} type="button">Preview server version</button></div>{preview ? <section aria-label="Immutable Preliminary Report preview" className="lead-document-preview"><small>SERVER-OWNED VERSION</small><h2>Preliminary Report</h2><p>{report.reportVersionId} · {report.auditId} · {report.organizationId}</p><p>{report.contentHash}</p><p>Rendering this version does not issue, sign, lock, or close a Finding.</p></section> : null}</> : !error ? <p>Loading exact report version…</p> : null}</div></LeadReportShell>;
}

export function LeadFinalReportsPage() {
  const { reports, error, loaded } = useReportList("FINAL");
  return <LeadReportShell routeLabel="Lead Final Reports"><div className="lead-secondary-page lead-final-list" data-testid="lead-final-reports-page"><header className="lead-secondary-header workbench-page-header"><div><h1>Final Reports</h1><p>View server-owned Final Report versions without changing approval authority.</p></div></header><CommandError message={error} /><FinalReportCreator />{reports.length ? <section className="lead-record-list" aria-label="Final Report versions">{reports.map((report) => <article aria-label={`Final Report ${report.reportVersionId}`} className="lead-report-row" data-report-id={report.reportId} data-report-version-id={report.reportVersionId} key={report.reportVersionId}><div><small>Report version</small><strong>{report.reportVersionId}</strong><span>{report.reportId} · {report.auditId}</span></div><div><small>Organization</small><strong>{report.organizationId}</strong></div><div><small>Status</small><strong>{reportStatusLabel(report.status)}</strong></div><div className="lead-record-actions"><Link className="lead-button" to={`/lead-inspector/final-reports/${encodeURIComponent(report.reportVersionId)}/readiness`}>View readiness</Link><Link className="lead-button" to={`/lead-inspector/final-reports/${encodeURIComponent(report.reportVersionId)}/prepare`}>View preparation snapshot</Link><Link className="lead-button" to={`/lead-inspector/final-reports/${encodeURIComponent(report.reportVersionId)}/document`}>View document</Link></div></article>)}</section> : loaded && !error ? <section className="lead-panel"><h2>No Final Report versions are available.</h2><p>The list will populate after the report lifecycle creates a server-owned Final Report version.</p></section> : !error ? <p>Loading Final Report versions…</p> : null}</div></LeadReportShell>;
}

function FinalReportPage({ mode }: { mode: "readiness" | "prepare" | "document" }) {
  const { reportId: routeReportId } = useParams<{ reportId: string }>();
  const { backend, report, error } = useReportVersion(routeReportId);
  const { findings, error: findingError } = useReportFindings(report);
  const [document, setDocument] = useState<DocumentMetadataView | null>(null);
  const [documentError, setDocumentError] = useState<string | null>(null);
  useEffect(() => {
    let cancelled = false;
    if (mode !== "document" || !report) return () => { cancelled = true; };
    void backend.documents.list({ organizationId: report.organizationId }).then(({ items }) => {
      const match = items.find((candidate) => candidate.kind === "REPORT" && (candidate.id === report.reportVersionId || candidate.id === report.reportId));
      if (!cancelled) setDocument(match ?? null);
    }).catch((cause) => {
      if (!cancelled) setDocumentError(errorMessage(cause));
    });
    return () => { cancelled = true; };
  }, [backend, mode, report]);
  return <LeadReportShell routeLabel="Lead Final Reports"><div className={`lead-secondary-page lead-final-${mode}`} data-report-version-id={report?.reportVersionId} data-testid={`lead-final-report-${mode}-page`}><header className="lead-secondary-header workbench-page-header"><div><p className="lead-breadcrumb"><span>Final Reports</span> › {report?.reportVersionId ?? "Loading"}</p><h1>{mode === "readiness" ? "Final Report Readiness" : mode === "prepare" ? "Final Report Preparation" : "Final Report Document"}</h1><p>{report?.organizationId ?? "Server-owned report version"}</p></div></header><CommandError message={error ?? findingError ?? documentError} />{report ? <><section className="lead-fact-strip"><div><small>Report ID</small><strong>{report.reportId}</strong></div><div><small>Version ID</small><strong>{report.reportVersionId}</strong></div><div><small>Audit</small><strong>{report.auditId}</strong></div><div><small>Status</small><strong>{reportStatusLabel(report.status)}</strong></div><div><small>Content hash</small><strong>{report.contentHash}</strong></div></section>{mode === "readiness" ? <div className="lead-workflow-grid"><section className="lead-panel"><h2>Findings Summary</h2><section aria-label="Readiness facts" className="lead-summary-cards"><article><span>Linked Findings</span><strong>{findings.length}</strong></article><article><span>Open Findings</span><strong>{findings.filter((finding) => finding.status !== "CLOSED").length}</strong></article><article><span>Closed Findings</span><strong>{findings.filter((finding) => finding.status === "CLOSED").length}</strong></article></section><p>CAP acceptance is not closure. Evidence verification must be complete before Finding closure.</p></section><aside className="lead-panel"><h2>Approval state</h2><p>{reportStatusLabel(report.status)}</p><p>Lead Inspector cannot issue or lock this immutable version.</p></aside></div> : null}{mode === "prepare" ? <section className="lead-editor-layout"><aside className="lead-panel"><h2>Report sections</h2><ol><li>Executive Summary</li><li>Inspection Overview</li><li>Findings Summary</li><li>CAP Implementation Summary</li><li>Conclusions</li></ol></aside><section className="lead-panel"><h2>Immutable preparation snapshot</h2><p>{report.reportVersionId} is server-owned. Content hash: {report.contentHash}.</p><p>Lead Inspector editing is not available for this version; no local Draft or mock mutation is created.</p><Link className="lead-button lead-button--primary" to={`/lead-inspector/final-reports/${encodeURIComponent(report.reportVersionId)}/document`}>View server document</Link></section></section> : null}{mode === "document" ? <article className="lead-document-preview"><header><strong>AviaSurveil360</strong><span>FINAL REPORT · SERVER-OWNED</span></header><h2>Final Report</h2><p>{report.reportId} · Version {report.version}</p><dl className="lead-detail-grid"><div><dt>Report version</dt><dd>{report.reportVersionId}</dd></div><div><dt>Audit</dt><dd>{report.auditId}</dd></div><div><dt>Organization</dt><dd>{report.organizationId}</dd></div><div><dt>Status</dt><dd>{reportStatusLabel(report.status)}</dd></div></dl><p>Rendering this immutable version does not close Findings.</p>{document?.downloadUrl ? <a className="lead-button" href={document.downloadUrl} rel="noreferrer" target="_blank">Open generated document</a> : <p>No generated document URL is available for this report version.</p>}</article> : null}</> : !error ? <p>Loading exact report version…</p> : null}</div></LeadReportShell>;
}

export function LeadFinalReportReadinessPage() { return <FinalReportPage mode="readiness" />; }
export function LeadPrepareFinalReportPage() { return <FinalReportPage mode="prepare" />; }
export function LeadFinalReportDocumentPage() { return <FinalReportPage mode="document" />; }
