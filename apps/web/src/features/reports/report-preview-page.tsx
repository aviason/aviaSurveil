import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { useBackendForRole } from "../../app/providers";
import type { DocumentMetadataView, FindingView, ReportVersionView } from "../../backend/backend";
import { useDialogFocus } from "../../ui/dialog-focus";
import { CommandError, errorMessage, formatLocalDate, formatSeverity, WorkspaceShell } from "../shared/workspace-shell";

const tabs = ["Summary", "Findings", "Attachments", "Comments", "Decision history"] as const;
type ReportTab = typeof tabs[number];

export function ReportPreviewPage() {
  const backend = useBackendForRole("manager");
  const params = useParams();
  const reportVersionId = params.reportVersionId;
  const [report, setReport] = useState<ReportVersionView | null>(null);
  const [reportDocument, setReportDocument] = useState<DocumentMetadataView | null>(null);
  const [findings, setFindings] = useState<FindingView[]>([]);
  const [activeTab, setActiveTab] = useState<ReportTab>("Summary");
  const [previewOpen, setPreviewOpen] = useState(false);
  const [searchDraft, setSearchDraft] = useState("");
  const [search, setSearch] = useState("");
  const [typeFilter, setTypeFilter] = useState<"all" | "inspection">("all");
  const [statusFilter, setStatusFilter] = useState<"all" | "in-review">("all");
  const [reason, setReason] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const previewOpenerRef = useRef<HTMLButtonElement | null>(null);
  const previewDialogRef = useRef<HTMLDivElement | null>(null);
  const previewCloseRef = useRef<HTMLButtonElement | null>(null);
  const closePreview = () => {
    setPreviewOpen(false);
    previewOpenerRef.current?.focus();
  };
  useDialogFocus({
    containerRef: previewDialogRef,
    initialFocusRef: previewCloseRef,
    onClose: closePreview,
    open: previewOpen,
  });

  useEffect(() => {
    let cancelled = false;
    if (!reportVersionId) {
      setError("The report route does not contain a report version identity.");
      return () => { cancelled = true; };
    }
    void Promise.all([
      backend.reports.getVersion({ reportVersionId }),
      backend.documents.open({ documentId: reportVersionId }),
      backend.findings.list({ limit: 50 }),
    ]).then(([nextReport, nextDocument, findingOutput]) => {
      if (!cancelled) {
        setReport(nextReport);
        setReportDocument(nextDocument);
        setFindings(findingOutput.items.filter((finding) => nextReport.findingIds.includes(finding.id)));
      }
    }).catch((cause) => {
      if (!cancelled) setError(errorMessage(cause));
    });
    return () => { cancelled = true; };
  }, [backend, reportVersionId]);

  const queueVisible = useMemo(() => {
    const matchesSearch = !search.trim() || [report?.reportVersionId, report?.reportId, report?.organizationId, report?.auditId]
      .some((value) => value?.toLowerCase().includes(search.toLowerCase()));
    const matchesType = typeFilter === "all" || typeFilter === "inspection";
    const matchesStatus = statusFilter === "all" || report?.status.includes("REVIEW");
    return matchesSearch && matchesType && matchesStatus;
  }, [report, search, statusFilter, typeFilter]);
  const searchIsApplied = searchDraft.trim() === search.trim();
  const filtersAreDefault = searchDraft === "" && search === "" && typeFilter === "all" && statusFilter === "all";

  async function decide(decision: "RETURN" | "FORWARD"): Promise<void> {
    if (!report) return;
    if (!reason.trim()) {
      setError("Report decision reason is required.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      setReport(await backend.reports.decide({
        operationId: `REPORT-${decision}-${report.reportVersionId}-${report.revision}`,
        reportVersionId: report.reportVersionId,
        expectedReportVersionRevision: report.revision,
        decision,
        reason,
      }));
      setReason("");
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel="Reports Approval">
      <div className="management-workspace reports-workspace">
        <header className="management-page-head workbench-page-header"><h1>Reports Approval</h1><p>Review separate Preliminary and Final Report artifacts and record the Department Manager decision.</p></header>
        <CommandError message={error} />
        <div className="reports-approval-layout">
          <section className="report-queue management-panel">
            <div className="management-section-head"><div><span>Department workspace</span><h2>Report Queue</h2></div></div>
            <p className="report-queue__scope">Server-owned report version · {report?.reportVersionId ?? "Loading"}</p>
            <div className="report-queue__filters">
              <label><span>Search</span><input aria-label="Search reports" onChange={(event) => setSearchDraft(event.target.value)} placeholder="Report, audit, organization" value={searchDraft} /></label>
              <button
                aria-label={searchIsApplied ? "Search reports unavailable" : undefined}
                disabled={searchIsApplied}
                onClick={() => setSearch(searchDraft)}
                title={searchIsApplied ? "The current report search is already applied." : undefined}
                type="button"
              >
                Search
              </button>
              <label><span>Type</span><select aria-label="Report type" onChange={(event) => setTypeFilter(event.target.value as typeof typeFilter)} value={typeFilter}><option value="all">All types</option><option value="inspection">Inspection</option></select></label>
              <label><span>Status</span><select aria-label="Report status" onChange={(event) => setStatusFilter(event.target.value as typeof statusFilter)} value={statusFilter}><option value="all">All statuses</option><option value="in-review">In review</option></select></label>
              <button
                aria-label={filtersAreDefault ? "Reset report filters unavailable" : undefined}
                disabled={filtersAreDefault}
                onClick={() => { setSearch(""); setSearchDraft(""); setTypeFilter("all"); setStatusFilter("all"); }}
                title={filtersAreDefault ? "Report filters are already at their defaults." : undefined}
                type="button"
              >
                Reset
              </button>
            </div>
            <div className="report-queue__counts" aria-label="Report counts">
              {[["All", report ? 1 : 0], ["Department", report?.status === "DEPARTMENT_REVIEW" ? 1 : 0], ["GM", report?.status === "GM_REVIEW" ? 1 : 0], ["Executive", report?.status === "EXECUTIVE_DIRECTOR_REVIEW" ? 1 : 0], ["Returned", report?.status === "RETURNED" ? 1 : 0], ["Issued", ["ISSUED", "LOCKED"].includes(report?.status ?? "") ? 1 : 0]].map(([label, value]) => <span key={label}><b>{value}</b><small>{label}</small></span>)}
            </div>
            <div className="management-table-scroll">
              <table aria-label="Report Queue"><thead><tr><th>Report</th><th>Organization</th><th>Type</th><th>Status</th><th>Action</th></tr></thead><tbody>
                {report && queueVisible ? <tr className="is-selected"><td><b>{report.reportVersionId}</b><small>{report.auditId}</small></td><td>{report.organizationId}</td><td>Inspection report</td><td>{report.status}</td><td><div className="manager-record-actions"><button aria-controls="report-version-dossier" onClick={() => document.getElementById("report-version-dossier")?.focus()} type="button">Open</button></div></td></tr> : <tr><td colSpan={5}>No matching report versions.</td></tr>}
              </tbody></table>
            </div>
          </section>

          {report ? (
            <section className="report-dossier management-panel" data-testid="report-version-dossier" id="report-version-dossier" tabIndex={-1}>
              <header><div><span>Immutable selected version</span><h2>{report.organizationId} · Inspection report</h2><small>{report.reportVersionId}</small></div><strong data-testid="report-status">{report.status}</strong></header>
              <dl className="report-dossier__identity"><div><dt>Report ID</dt><dd>{report.reportId}</dd></div><div><dt>Version</dt><dd>Version {report.version}</dd></div><div><dt>Audit</dt><dd>{report.auditId}</dd></div><div><dt>Content hash</dt><dd>{report.contentHash}</dd></div></dl>
              <div className="report-tabs" role="tablist" aria-label="Report dossier sections">
                {tabs.map((tab) => <button aria-selected={activeTab === tab} key={tab} onClick={() => setActiveTab(tab)} role="tab" type="button">{tab}</button>)}
              </div>
              <div className="report-tab-panel" role="tabpanel">
                {activeTab === "Summary" ? <><h3>Inspection Report Preview</h3><p>This immutable report version summarizes the server-owned audit record and current Finding projection.</p></> : null}
                {activeTab === "Findings" ? findings.length ? <ul>{findings.map((finding) => <li key={finding.id}><b>{finding.findingNumber}</b> · {formatSeverity(finding.severity)} · Due {formatLocalDate(finding.dueDate)} · {finding.status.replaceAll("_", " ")}</li>)}</ul> : <p>No Finding is included.</p> : null}
                {activeTab === "Attachments" ? <p>Attachments are represented by the server-owned document metadata. Binary download is available when the renderer has produced a document.</p> : null}
                {activeTab === "Comments" ? <p>No additional CAA-visible comments are recorded for this version.</p> : null}
                {activeTab === "Decision history" ? <p>The current immutable state is <b>{report.status}</b>. Earlier report versions and decisions are preserved.</p> : null}
              </div>
              {findings.length ? <div className="report-conclusion">{findings.map((finding) => <p key={finding.id}><span data-testid={`report-finding-status-${finding.id}`}>{finding.status}</span><strong>{finding.closureBasis === "EVIDENCE_VERIFIED" ? "Evidence accepted and verified" : "Finding remains open"}</strong></p>)}</div> : null}
              {report.status === "DEPARTMENT_REVIEW" ? <div className="report-decision-panel"><label>Department Manager decision reason<textarea rows={3} value={reason} onChange={(event) => setReason(event.target.value)} /></label><div><button disabled={busy} onClick={() => void decide("RETURN")} type="button">Return for Revision</button><button disabled={busy} onClick={() => void decide("FORWARD")} type="button">Forward to General Manager</button></div></div> : <p className="management-authority-note">Department Manager cannot issue, sign, lock, or close this report. The current stage belongs to the authorized downstream role.</p>}
              <div className="report-dossier__actions"><button aria-label="Review Full Report" onClick={() => setPreviewOpen(true)} ref={previewOpenerRef} type="button">Preview Full Report</button>{reportDocument?.downloadUrl ? <a className="primary-link" href={reportDocument.downloadUrl} rel="noreferrer" target="_blank">Download generated document</a> : <span title="No generated document URL is available for this report version.">Generated document unavailable</span>}</div>
              <Link className="primary-link" to={`/auditee/reports/${encodeURIComponent(report.reportVersionId)}`}>View as Auditee</Link>
            </section>
          ) : null}
        </div>
        {previewOpen ? <div className="report-preview-dialog" ref={previewDialogRef} role="dialog" aria-modal="true" aria-label="Immutable report preview"><article><header><div><span>Immutable report preview</span><h2>Cabin Inspection Report</h2></div><button aria-label="Close report preview" onClick={closePreview} ref={previewCloseRef} type="button">×</button></header><p><b>{report?.reportVersionId}</b> · {report?.contentHash}</p><p>This preview is read-only. It does not issue, sign, lock, or close the report.</p></article></div> : null}
      </div>
    </WorkspaceShell>
  );
}
