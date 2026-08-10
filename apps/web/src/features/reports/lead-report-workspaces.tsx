import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { FindingView, ReportApprovalStatus, ReportVersionView } from "../../backend/backend";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";
import { readPreliminaryReportDraft, savePreliminaryReportDraft } from "../shared/lead-workspace-state";
import { discoverCAAReportVersions } from "./report-discovery";

const FINAL_REPORT_VERSION_ID = "RPT-CAB-2026-001-V1";
const FINAL_REPORT_ID = "RPT-CAB-2026-001";
const FINAL_AUDIT_ID = "AUD-2026-001";

interface PreliminaryReportVersion {
  reportId: string;
  auditId: string;
  organization: string;
  inspection: string;
  version: number;
  state: "DEPARTMENT_REVIEW" | "RETURNED";
  updatedAt: string;
  returnReason: string | null;
}

const preliminaryReportVersions = Object.freeze<readonly PreliminaryReportVersion[]>([
  Object.freeze({ reportId: "PR-2026-018", auditId: FINAL_AUDIT_ID, organization: "Fly Namibia", inspection: "Cabin Inspection", version: 1, state: "DEPARTMENT_REVIEW", updatedAt: "2026-07-09 10:30", returnReason: null }),
  Object.freeze({ reportId: "PR-2026-019", auditId: "AUD-2026-099", organization: "SkyCargo Air", inspection: "Special Inspection", version: 2, state: "RETURNED", updatedAt: "2026-07-11 09:15", returnReason: "Finding basis requires clarification" }),
]);

function reportStatusLabel(status: ReportApprovalStatus): string {
  return status.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function useLeadFinalReport(): { report: ReportVersionView | null; error: string | null; loaded: boolean } {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("leadInspector") ?? runtime.backend, [runtime]);
  const [report, setReport] = useState<ReportVersionView | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loaded, setLoaded] = useState(false);
  useEffect(() => {
    let cancelled = false;
    void discoverCAAReportVersions(backend, ["FINAL"]).then((reports) => {
      if (!cancelled) setReport(reports[0] ?? null);
    }).catch((cause) => {
      if (!cancelled) setError(errorMessage(cause));
    }).finally(() => {
      if (!cancelled) setLoaded(true);
    });
    return () => { cancelled = true; };
  }, [backend]);
  return { report, error, loaded };
}

function LeadReportShell({ routeLabel, children }: { routeLabel: string; children: React.ReactNode }) {
  return <WorkspaceShell roleLabel="Lead Inspector" routeLabel={routeLabel}>{children}</WorkspaceShell>;
}

export function LeadPreliminaryReportsPage() {
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<"all" | PreliminaryReportVersion["state"]>("all");
  const visible = preliminaryReportVersions.filter((report) => {
    const matchesQuery = `${report.reportId} ${report.auditId} ${report.organization} ${report.inspection}`.toLowerCase().includes(query.toLowerCase());
    return matchesQuery && (status === "all" || report.state === status);
  });
  return <LeadReportShell routeLabel="Lead Preliminary Reports">
    <div className="lead-secondary-page lead-preliminary-page" data-testid="lead-preliminary-reports-page">
      <header className="lead-secondary-header workbench-page-header"><div><h1>Preliminary Reports</h1><p>View and manage all preliminary inspection reports.</p></div></header>
      <section className="lead-metric-grid" aria-label="Preliminary Report summary">
        <article><span>Draft</span><strong>0</strong></article>
        <article><span>In Review</span><strong>1</strong></article>
        <article><span>Approved</span><strong>0</strong></article>
        <article><span>Returned</span><strong>1</strong></article>
        <article><span>Released / Closed</span><strong>0</strong></article>
        <article><span>Total</span><strong>{preliminaryReportVersions.length}</strong></article>
      </section>
      <div className="lead-filter-row">
        <label>Search reports<input type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search reports..." /></label>
        <label>Status<select value={status} onChange={(event) => setStatus(event.target.value as typeof status)}><option value="all">All Status</option><option value="DEPARTMENT_REVIEW">Department Review</option><option value="RETURNED">Returned to Lead Inspector</option></select></label>
      </div>
      <section className="lead-record-list" aria-label="Preliminary Report versions">
        {visible.map((report) => <article aria-label={`Preliminary Report ${report.reportId} version ${report.version}`} className="lead-report-row" data-report-id={report.reportId} data-report-version={report.version} key={`${report.reportId}-V${report.version}`}>
          <div><small>Report ID</small><strong>{report.reportId}</strong><span>{report.auditId} · {report.inspection}</span></div>
          <div><small>Organization</small><strong>{report.organization}</strong></div>
          <div><small>Status</small><strong className={`lead-state lead-state--${report.state.toLowerCase()}`}>{report.state === "RETURNED" ? "Returned to Lead Inspector" : "Department Review"}</strong>{report.returnReason ? <span>Return reason: {report.returnReason}</span> : <span>Current Owner: Department Manager</span>}</div>
          <div><small>Last Updated</small><strong>{report.updatedAt}</strong></div>
          {report.reportId === "PR-2026-018" ? <Link className="lead-button lead-button--primary" to="/lead-inspector/preliminary-reports/PR-2026-018">Open report package</Link> : <button aria-label={`Report package unavailable for ${report.reportId}`} className="lead-button" disabled title={`Preliminary Report ${report.reportId} has no declared Lead Inspector workflow route.`} type="button">Package unavailable</button>}
        </article>)}
      </section>
    </div>
  </LeadReportShell>;
}

export function LeadPreliminaryReportWorkflowPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("leadInspector") ?? runtime.backend, [runtime]);
  const initialDraft = useMemo(() => readPreliminaryReportDraft(backend), [backend]);
  const [activeStep, setActiveStep] = useState(initialDraft.activeStep);
  const [saved, setSaved] = useState("");
  const [preview, setPreview] = useState(false);
  const [executiveSummary, setExecutiveSummary] = useState(initialDraft.executiveSummary);
  const [auditeeComment, setAuditeeComment] = useState(initialDraft.commentToAuditee);
  const [internalNote, setInternalNote] = useState(initialDraft.internalCaaNote);
  const [finding, setFinding] = useState<FindingView | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    let cancelled = false;
    void backend.findings.get({ findingId: "FND-CAB-2026-001" }).then((value) => {
      if (!cancelled) setFinding(value);
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend]);
  async function saveDraft() {
    await backend.administration?.invokeVisibleAction({ screenId: "lead-preliminary-report-workflow", actionId: "save-preliminary-draft" });
    savePreliminaryReportDraft(backend, {
      reportId: "PR-2026-018",
      auditId: FINAL_AUDIT_ID,
      version: 1,
      activeStep,
      executiveSummary,
      commentToAuditee: auditeeComment,
      internalCaaNote: internalNote,
      saved: true,
    });
    setSaved("PR-2026-018 version 1 working draft saved in the demo workspace.");
  }
  return <LeadReportShell routeLabel="Lead Preliminary Reports">
    <div className="lead-secondary-page lead-preliminary-workflow" data-audit-id={FINAL_AUDIT_ID} data-report-id="PR-2026-018" data-testid="lead-preliminary-report-workflow-page">
      <header className="lead-secondary-header workbench-page-header"><div><p className="lead-breadcrumb">Preliminary Reports › PR-2026-018 › Inspection &amp; Findings</p><h1>Preliminary Report — Cabin Inspection — Fly Namibia</h1><span className="lead-state">Department Review</span></div></header>
      <CommandError message={error} />
      <section className="lead-fact-strip" aria-label="Preliminary Report identity"><div><small>Inspection</small><strong>Cabin Inspection</strong><span>{FINAL_AUDIT_ID}</span></div><div><small>Organization</small><strong>Fly Namibia</strong></div><div><small>Inspection Dates</small><strong>15 Jun 2026</strong></div><div><small>Lead Inspector</small><strong>Caner Yildiz</strong></div><div><small>Report Version</small><strong>1.0 (Working)</strong></div></section>
      <div aria-label="Preliminary Report preparation steps" className="lead-stepper">
        {(["inspection", "content", "attachments", "review"] as const).map((step, index) => { const label = step === "inspection" ? "Inspection & Findings" : step === "content" ? "Report Content" : step === "attachments" ? "Attachments" : "Review & Submit"; return <button aria-current={activeStep === step ? "step" : undefined} aria-label={label} className={activeStep === step ? "is-active" : ""} key={step} onClick={() => setActiveStep(step)} type="button"><span aria-hidden="true">{index + 1}</span>{label}</button>; })}
      </div>
      {activeStep === "inspection" ? <section aria-label="Inspection & Findings workspace" className="lead-panel lead-step-workspace"><h2>Inspection Overview</h2><dl className="lead-detail-grid"><div><dt>Audit ID</dt><dd>{FINAL_AUDIT_ID}</dd></div><div><dt>Inspection Type</dt><dd>Cabin Inspection</dd></div><div><dt>Organization</dt><dd>Fly Namibia</dd></div><div><dt>Checklist Template</dt><dd>CTV-CABIN-1</dd></div></dl><h2>Authorized Findings (1)</h2>{finding ? <article aria-label={`Finding ${finding.id}`} className="lead-finding-record" data-finding-id={finding.id}><strong>{finding.id}</strong><span>{finding.findingNumber}</span><h3>{finding.title}</h3><p>{finding.findingBasis}</p><dl className="lead-detail-grid"><div><dt>Status</dt><dd>{finding.status}</dd></div><div><dt>Current Owner</dt><dd>{finding.currentOwnerRole}</dd></div><div><dt>Next Action</dt><dd>{finding.nextAction}</dd></div><div><dt>Due Date</dt><dd>{finding.dueDate ?? "Not set"}</dd></div></dl></article> : <p>Loading canonical Finding…</p>}<p>Report issue does not close this Finding.</p></section> : null}
      {activeStep === "content" ? <section aria-label="Report Content workspace" className="lead-panel lead-step-workspace"><h2>Report Content</h2><label>Executive Summary<textarea aria-label="Executive Summary" value={executiveSummary} onChange={(event) => setExecutiveSummary(event.target.value)} /></label><p>The summary remains a working Draft for PR-2026-018 version 1.</p></section> : null}
      {activeStep === "attachments" ? <section aria-label="Attachment workspace" className="lead-panel lead-step-workspace"><h2>Attachments</h2><article className="lead-attachment-record"><strong>PR-2026-018-working-draft.pdf</strong><span>Mock filename only · no real upload or storage</span></article><button className="lead-button" disabled title="PR-2026-018 has no declared real upload capability in this frontend-only demo." type="button">Upload unavailable for PR-2026-018</button></section> : null}
      {activeStep === "review" ? <section aria-label="Review workspace" className="lead-panel lead-step-workspace"><h2>Review &amp; Submit</h2><dl className="lead-detail-grid"><div><dt>Report</dt><dd>PR-2026-018 · Version 1 Draft</dd></div><div><dt>Audit</dt><dd>{FINAL_AUDIT_ID}</dd></div><div><dt>Finding</dt><dd>{finding?.id ?? "Loading"}</dd></div><div><dt>Finding count</dt><dd>{finding ? 1 : 0}</dd></div></dl><p>Submission cannot approve, issue, sign, lock, or close a Finding.</p></section> : null}
      <div className="lead-comment-grid">
        <section aria-label="Comment to Auditee"><h2>Comment to Auditee</h2><textarea aria-label="Comment to Auditee text" value={auditeeComment} onChange={(event) => setAuditeeComment(event.target.value)} /></section>
        <section aria-label="Internal CAA Note"><h2>Internal CAA Note</h2><textarea aria-label="Internal CAA Note text" value={internalNote} onChange={(event) => setInternalNote(event.target.value)} /><p>CAA-only. Never included in an Auditee projection.</p></section>
      </div>
      <div className="lead-action-row"><button className="lead-button" onClick={() => void saveDraft()} type="button">Save Draft</button><button className="lead-button lead-button--primary" onClick={() => setPreview((value) => !value)} type="button">Preview working document</button></div>
      {saved ? <p className="lead-action-result" data-durable-outcome="preliminary-report-saved" role="status">{saved}</p> : null}
      {preview ? <section aria-label="PR-2026-018 working document preview" className="lead-document-preview"><small>WORKING DRAFT · VERSION 1</small><h2>PRELIMINARY INSPECTION REPORT</h2><p>Report PR-2026-018 · Audit {FINAL_AUDIT_ID} · Fly Namibia</p><p>Finding {finding?.id ?? "loading"}</p><p>{executiveSummary}</p><p>{auditeeComment}</p></section> : null}
    </div>
  </LeadReportShell>;
}

export function LeadFinalReportsPage() {
  const { report, error, loaded } = useLeadFinalReport();
  return <LeadReportShell routeLabel="Lead Final Reports"><div className="lead-secondary-page lead-final-list" data-testid="lead-final-reports-page"><header className="lead-secondary-header workbench-page-header"><div><h1>Final Reports</h1><p>View and manage final reports without changing approval authority.</p></div></header><CommandError message={error} />{report ? <><section className="lead-metric-grid" aria-label="Final Report summary"><article><span>All Reports</span><strong>1</strong></article><article><span>Ready for Preparation</span><strong>0</strong></article><article><span>Waiting Approval</span><strong>1</strong></article></section><article aria-label={`Final Report ${report.reportId}`} className="lead-report-row" data-report-id={report.reportId} data-report-version-id={report.reportVersionId}><div><small>Report ID</small><strong>{report.reportId}</strong><span>{report.auditId} · Version {report.version}</span></div><div><small>Organization</small><strong>{report.organizationId}</strong></div><div><small>Status</small><strong>{reportStatusLabel(report.status)}</strong><span>Approval authority remains with the current stage owner.</span></div><div className="lead-record-actions"><Link className="lead-button" to={`/lead-inspector/final-reports/${report.reportId}/readiness`}>View {report.reportId} readiness</Link><Link className="lead-button" to={`/lead-inspector/final-reports/${report.reportId}/prepare`}>View immutable preparation snapshot</Link><Link className="lead-button" to={`/lead-inspector/final-reports/${report.reportId}/document`}>View Final Report document</Link></div></article></> : loaded && !error ? <section className="lead-panel"><h2>No Final Report versions are available yet.</h2><p>The list will populate after the canonical report lifecycle creates an immutable Final Report version.</p><button disabled title="No immutable Final Report version exists for Lead Inspector review." type="button">Report unavailable</button></section> : !error ? <p>Loading Final Report versions…</p> : null}</div></LeadReportShell>;
}

export function LeadFinalReportReadinessPage() {
  const { report, error } = useLeadFinalReport();
  const disabledReason = `Report ${FINAL_REPORT_VERSION_ID} is at Executive Director Review; Lead Inspector preparation is read-only.`;
  return <LeadReportShell routeLabel="Lead Final Reports"><div className="lead-secondary-page lead-final-readiness" data-report-version-id={report?.reportVersionId} data-testid="lead-final-report-readiness-page"><header className="lead-secondary-header workbench-page-header"><div><p className="lead-breadcrumb">Final Reports › {FINAL_REPORT_ID} › Readiness</p><h1>Final Report Preparation</h1><p>{FINAL_REPORT_ID} · Fly Namibia</p></div></header><CommandError message={error} />{report ? <><section className="lead-fact-strip"><div><small>Report ID</small><strong>{report.reportId}</strong></div><div><small>Audit</small><strong>{report.auditId}</strong></div><div><small>Version</small><strong>{report.version}</strong></div><div><small>Status</small><strong>{reportStatusLabel(report.status)}</strong></div></section><div className="lead-workflow-grid"><section className="lead-panel"><h2>Findings Summary</h2><section aria-label="Readiness blockers" className="lead-summary-cards"><article><span>Linked Findings</span><strong>{report.findingIds.length}</strong></article><article className="is-danger"><span>Open Findings</span><strong>{report.findingIds.length}</strong></article><article><span>CAP Implementation</span><strong>Required</strong></article><article><span>Attachments</span><strong>Mock filenames only</strong></article></section><p>CAP acceptance is not closure. Evidence verification or authorized closure remains required.</p></section><aside className="lead-panel"><h2>Approval Path</h2><ol><li>Lead Inspector preparation — completed</li><li>Department Manager review — completed</li><li>General Manager review — completed</li><li>Executive Director Review — current</li></ol></aside></div><div className="lead-action-row"><Link className="lead-button" to={`/lead-inspector/final-reports/${report.reportId}/prepare`}>View immutable preparation snapshot</Link><Link className="lead-button" to={`/lead-inspector/final-reports/${report.reportId}/document`}>View Final Report document</Link><button aria-label="Prepare Final Report unavailable" className="lead-button lead-button--primary" disabled title={disabledReason} type="button">Prepare Final Report unavailable</button></div></> : <p>Loading exact report version…</p>}</div></LeadReportShell>;
}

export function LeadPrepareFinalReportPage() {
  const { report, error } = useLeadFinalReport();
  const reason = `Report ${FINAL_REPORT_VERSION_ID} is at Executive Director Review; Lead Inspector cannot edit this immutable version.`;
  return <LeadReportShell routeLabel="Lead Final Reports"><div className="lead-secondary-page lead-final-editor" data-report-version-id={report?.reportVersionId} data-testid="lead-prepare-final-report-page"><header className="lead-secondary-header workbench-page-header"><div><p className="lead-breadcrumb">Final Reports › {FINAL_REPORT_ID} › Prepare</p><h1>Report Content</h1><p>Exact immutable version preview for Fly Namibia</p></div></header><CommandError message={error} />{report ? <><section className="lead-fact-strip"><div><small>Report ID</small><strong>{report.reportId}</strong></div><div><small>Inspection ID</small><strong>{report.auditId}</strong></div><div><small>Report Version</small><strong>{report.version}</strong></div><div><small>Status</small><strong>{reportStatusLabel(report.status)}</strong></div></section><div className="lead-editor-layout"><aside className="lead-panel"><h2>Report Sections</h2><ol><li>Executive Summary</li><li>Inspection Overview</li><li>Findings Summary</li><li>CAP Implementation Summary</li><li>Conclusions</li></ol></aside><section className="lead-panel"><h2>1. Executive Summary</h2><textarea aria-label="Executive Summary" readOnly value="The exact Final Report version is already in Executive Director Review. Lead Inspector editing is locked." /><div className="lead-action-row"><button aria-label="Save as Draft unavailable" className="lead-button" disabled title={reason} type="button">Save as Draft unavailable</button><Link className="lead-button lead-button--primary" to={`/lead-inspector/final-reports/${report.reportId}/document`}>Preview Final Report</Link></div></section><aside className="lead-panel"><h2>Report Progress</h2><strong>Immutable version {report.version}</strong><p>Lead Inspector cannot issue, sign, or lock this report.</p></aside></div></> : <p>Loading exact report version…</p>}</div></LeadReportShell>;
}

export function LeadFinalReportDocumentPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("leadInspector") ?? runtime.backend, [runtime]);
  const { report, error } = useLeadFinalReport();
  const [status, setStatus] = useState("");
  async function prepareDownload() {
    await backend.administration?.invokeVisibleAction({ screenId: "lead-final-report-document", actionId: "download-report" });
    setStatus(`${FINAL_REPORT_VERSION_ID} preview prepared in the demo workspace.`);
  }
  const reportPreview = [
    "AviaSurveil360 demo Final Report preview",
    `Report ID: ${report?.reportId ?? FINAL_REPORT_ID}`,
    `Report Version ID: ${report?.reportVersionId ?? FINAL_REPORT_VERSION_ID}`,
    `Audit ID: ${report?.auditId ?? FINAL_AUDIT_ID}`,
    `Status: ${report ? reportStatusLabel(report.status) : "Loading"}`,
    "This is a mock preview, not an issued or signed report.",
  ].join("\n");
  const reportPreviewHref = `data:text/plain;charset=utf-8,${encodeURIComponent(reportPreview)}`;
  return <LeadReportShell routeLabel="Lead Final Reports"><div className="lead-secondary-page lead-final-document" data-report-version-id={report?.reportVersionId} data-testid="lead-final-report-document-page"><header className="lead-secondary-header workbench-page-header"><div><p className="lead-breadcrumb">Final Reports › {FINAL_REPORT_ID}</p><h1>Final Report</h1><p>{FINAL_REPORT_ID} · Fly Namibia</p></div><a className="lead-button workbench-page-header__action" download={`AviaSurveil360_${FINAL_REPORT_VERSION_ID}_Report_Preview.txt`} href={reportPreviewHref} onClick={() => void prepareDownload()}>Download report preview</a></header><CommandError message={error} />{status ? <p className="lead-action-result" role="status">{status}</p> : null}{report ? <article className="lead-document-preview"><header><strong>AviaSurveil360</strong><span>FINAL REPORT · DEMO-ONLY</span></header><h2>Cabin Inspection Final Report</h2><p>{report.reportId} · Version {report.version}</p><dl className="lead-detail-grid"><div><dt>Report ID</dt><dd>{report.reportId}</dd></div><div><dt>Report Version ID</dt><dd>{report.reportVersionId}</dd></div><div><dt>Audit ID</dt><dd>{report.auditId}</dd></div><div><dt>Organization</dt><dd>Fly Namibia</dd></div><div><dt>Current Status</dt><dd>{reportStatusLabel(report.status)}</dd></div></dl><h3>1. Executive Summary</h3><p>This immutable demo report remains in its current approval stage. Rendering it does not close Findings.</p></article> : <p>Loading exact report version…</p>}</div></LeadReportShell>;
}
