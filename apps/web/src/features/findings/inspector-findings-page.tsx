import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { FindingView } from "../../backend/backend";
import { DueState } from "../../ui/workbench/due-state";
import { StatusPill } from "../../ui/workbench/status-pill";
import { CommandError, errorMessage, formatLocalDate, formatSeverity, WorkspaceShell } from "../shared/workspace-shell";

function ownerLabel(finding: FindingView): string {
  if (finding.currentOwnerType === "AUDITEE") return finding.organizationName;
  if (finding.currentOwnerRole === "leadInspector") return "Lead Inspector";
  if (finding.currentOwnerRole === "manager") return "Department Manager";
  if (finding.currentOwnerRole === "inspector") return "CAA Inspector";
  return "CAA";
}

type SelectedDossierSection = "details" | "cap" | "conversation" | "files" | "history";

function csvCell(value: string | null | undefined): string {
  return `"${(value ?? "").replaceAll("\"", "\"\"")}"`;
}

function findingsCsv(findings: FindingView[]): string {
  const header = ["Finding", "Title", "Organization", "Severity", "Status", "Due Date", "Current Owner", "Next Action"];
  const rows = findings.map((finding) => [
    finding.findingNumber,
    finding.title,
    finding.organizationName,
    formatSeverity(finding.severity),
    finding.status.replaceAll("_", " "),
    finding.dueDate,
    ownerLabel(finding),
    finding.nextAction,
  ]);
  return [header, ...rows].map((row) => row.map(csvCell).join(",")).join("\n");
}

export function InspectorFindingsPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("inspector") ?? runtime.backend, [runtime]);
  const [searchParams] = useSearchParams();
  const [findings, setFindings] = useState<FindingView[]>([]);
  const [query, setQuery] = useState("");
  const [summaryFilter, setSummaryFilter] = useState("all");
  const [severityFilter, setSeverityFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");
  const [dueFilter, setDueFilter] = useState("all");
  const [exportStatus, setExportStatus] = useState("");
  const [selectedDossierSection, setSelectedDossierSection] = useState<SelectedDossierSection>("cap");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    void backend.findings.list({ limit: 50 }).then(({ items }) => {
      if (!cancelled) setFindings(items);
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend]);

  const visible = findings.filter((finding) => {
    if (summaryFilter === "waiting-cap" && finding.status !== "WAITING_FOR_CAP") return false;
    if (summaryFilter === "cap-submitted" && finding.status !== "CAP_SUBMITTED") return false;
    if (summaryFilter === "returned" && !finding.status.includes("MORE_INFORMATION_REQUESTED")) return false;
    if (summaryFilter === "closed" && finding.status !== "CLOSED") return false;
    if (severityFilter !== "all" && finding.severity !== severityFilter) return false;
    if (statusFilter === "open" && finding.status === "CLOSED") return false;
    if (statusFilter !== "all" && statusFilter !== "open" && finding.status !== statusFilter) return false;
    const due = finding.dueDate ? Date.parse(`${finding.dueDate}T00:00:00Z`) : null;
    const today = Date.parse(`${new Date().toISOString().slice(0, 10)}T00:00:00Z`);
    if (dueFilter === "overdue" && (due === null || due >= today)) return false;
    if (dueFilter === "due-soon" && (due === null || due < today || due > today + 30 * 86_400_000)) return false;
    return `${finding.findingNumber} ${finding.title} ${finding.organizationName}`.toLowerCase().includes(query.trim().toLowerCase());
  });
  const selectedFinding = findings.find((finding) => finding.id === searchParams.get("findingId")) ?? null;
  const filtersAreDefault = summaryFilter === "all" && severityFilter === "all" && statusFilter === "all" && dueFilter === "all" && query === "";
  const visibleQueueCount = visible.length;
  const exportHref = `data:text/csv;charset=utf-8,${encodeURIComponent(findingsCsv(visible))}`;

  return (
    <WorkspaceShell roleLabel="CAA Inspector" routeLabel="Findings">
      <div className="inspector-secondary-page inspector-findings-page" data-testid="inspector-findings-page">
        <div className="inspector-breadcrumbs" aria-label="Finding breadcrumb">{selectedFinding ? <><span>{selectedFinding.organizationName}</span><span>{selectedFinding.auditId}</span><span>{selectedFinding.title}</span></> : null}<b>Findings</b></div>
        <header className="inspector-secondary-head workbench-page-header">
          <div><span className="inspector-secondary-scope">CAA Inspector workspace</span><h1>Findings</h1><p>All findings and CAPs from this inspection</p></div>
          <div className="inspector-secondary-actions">
            <a download="AviaSurveil360_Inspector_Findings.csv" href={exportHref} onClick={() => setExportStatus(`Findings CSV prepared for ${visible.length} records.`)}><span>⇩ Export</span></a>
            <button type="button" onClick={() => setStatusFilter((current) => current === "all" ? "open" : "all")}><span>▽ Filter</span></button>
            <button aria-label="New Finding unavailable: Create Finding begins from an inspected checklist response." disabled title="Create Finding begins from an inspected checklist response." type="button"><span><span aria-hidden="true">🔒</span> New Finding</span></button>
          </div>
        </header>
        <CommandError message={error} />
        {exportStatus ? <p role="status" className="inspector-action-result">{exportStatus}</p> : null}
        <section className="inspector-finding-kpis" aria-label="Finding summary">
          <button aria-pressed={summaryFilter === "all"} className={summaryFilter === "all" ? "is-active" : ""} onClick={() => setSummaryFilter("all")} type="button"><span>▤</span><b>All Findings</b><strong>{findings.length}</strong></button>
          <button aria-pressed={summaryFilter === "waiting-cap"} className={summaryFilter === "waiting-cap" ? "is-active" : ""} onClick={() => setSummaryFilter("waiting-cap")} type="button"><span>⌛</span><b>Waiting for CAP</b><strong>{findings.filter((finding) => finding.status === "WAITING_FOR_CAP").length}</strong></button>
          <button aria-pressed={summaryFilter === "cap-submitted"} className={summaryFilter === "cap-submitted" ? "is-active" : ""} onClick={() => setSummaryFilter("cap-submitted")} type="button"><span>➤</span><b>CAP Submitted</b><strong>{findings.filter((finding) => finding.status === "CAP_SUBMITTED").length}</strong></button>
          <button aria-pressed={summaryFilter === "returned"} className={summaryFilter === "returned" ? "is-active" : ""} onClick={() => setSummaryFilter("returned")} type="button"><span>↩</span><b>Returned</b><strong>{findings.filter((finding) => finding.status.includes("MORE_INFORMATION_REQUESTED")).length}</strong></button>
          <button aria-pressed={summaryFilter === "closed"} className={summaryFilter === "closed" ? "is-active" : ""} onClick={() => setSummaryFilter("closed")} type="button"><span>✓</span><b>Closed</b><strong>{findings.filter((finding) => finding.status === "CLOSED").length}</strong></button>
        </section>
        <section className="inspector-finding-filters" aria-label="Finding filters">
          <label className="inspector-secondary-search"><span>Search Findings</span><span className="inspector-search-control"><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search by ID, title, or keyword..." type="search" /><span aria-hidden="true">⌕</span></span></label>
          <label><span>CAP Level</span><select value={severityFilter} onChange={(event) => setSeverityFilter(event.target.value)}><option value="all">All Levels</option><option value="LEVEL_1_CRITICAL">Level 1 Critical</option><option value="LEVEL_2_MAJOR">Level 2 Major</option><option value="LEVEL_3_MINOR">Level 3 Minor</option></select></label>
          <label><span>CAP Status</span><select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}><option value="all">All Statuses</option><option value="open">Open</option><option value="WAITING_FOR_CAP">Waiting for CAP</option><option value="CAP_SUBMITTED">CAP Submitted</option><option value="CLOSED">Closed</option></select></label>
          <label><span>Due Date</span><select value={dueFilter} onChange={(event) => setDueFilter(event.target.value)}><option value="all">All Due Dates</option><option value="due-soon">Due Soon</option><option value="overdue">Overdue</option></select></label>
          <button
            aria-label={filtersAreDefault ? "Reset Finding filters unavailable" : undefined}
            disabled={filtersAreDefault}
            onClick={() => { setSummaryFilter("all"); setSeverityFilter("all"); setStatusFilter("all"); setDueFilter("all"); setQuery(""); }}
            title={filtersAreDefault ? "Finding filters are already at their defaults." : undefined}
            type="button"
          >
            Reset
          </button>
        </section>
        <section className="inspector-secondary-register" aria-label="Finding Queue">
          <header><h2>Finding Queue</h2><span>{visibleQueueCount} findings</span></header>
          <div className="inspector-record-grid">
            {visible.map((finding) => <article className="inspector-record-card inspector-mobile-stack" data-current-owner-role={finding.currentOwnerRole} data-due-date={finding.dueDate ?? ""} data-finding-id={finding.id} data-next-action={finding.nextAction} key={finding.id}>
              <div className="inspector-record-card__title"><div><b>{finding.findingNumber}</b><h3>{finding.title}</h3><p>{finding.organizationName} · {formatSeverity(finding.severity)}</p></div><StatusPill label={finding.status.replaceAll("_", " ")} tone={finding.status === "CLOSED" ? "success" : "warning"} /></div>
              <dl className="inspector-decision-grid">
                <div><dt>Current Owner</dt><dd>{ownerLabel(finding)}</dd></div>
                <div><dt>Next Action</dt><dd>{finding.nextAction}</dd></div>
                <div><dt>Due Date</dt><dd><DueState dueDate={finding.dueDate} today={new Date().toISOString().slice(0, 10)} /></dd></div>
                <div><dt>Related Audit</dt><dd>{finding.auditId}</dd></div>
              </dl>
              <Link className="inspector-secondary-button inspector-secondary-button--primary" to={`/inspector/findings/${encodeURIComponent(finding.id)}`}>Open Finding dossier</Link>
            </article>)}
          </div>
          {selectedFinding ? <article aria-label={`Selected Finding ${selectedFinding.findingNumber}`} className="inspector-finding-selected" data-current-owner-role={selectedFinding.currentOwnerRole} data-due-date={selectedFinding.dueDate ?? ""} data-finding-id={selectedFinding.id} data-next-action={selectedFinding.nextAction}>
            <header><div><h2>{selectedFinding.findingNumber} {selectedFinding.title}</h2><p>⌘ &nbsp; {selectedFinding.regulatoryReference ?? selectedFinding.findingBasis} &nbsp; ▣</p><small>Raised on {formatLocalDate(selectedFinding.issuedAt?.slice(0, 10) ?? selectedFinding.createdAt.slice(0, 10))} by CAA Inspector</small></div><span>● {formatSeverity(selectedFinding.severity)}</span></header>
            <span className="inspector-finding-selected__status">● {selectedFinding.status.replaceAll("_", " ")}</span>
            <dl><div><dt>Current Owner</dt><dd>{ownerLabel(selectedFinding)}</dd></div><div><dt>Next Action</dt><dd>{selectedFinding.nextAction}</dd></div><div><dt>Due Date</dt><dd><DueState dueDate={selectedFinding.dueDate} today={new Date().toISOString().slice(0, 10)} /></dd></div><div><dt>Organization</dt><dd>{selectedFinding.organizationName}</dd></div></dl>
            <div aria-label="Finding dossier sections" className="inspector-finding-selected__sections" role="tablist"><button aria-selected={selectedDossierSection === "details"} className={selectedDossierSection === "details" ? "is-active" : ""} onClick={() => setSelectedDossierSection("details")} role="tab" type="button">Details</button><button aria-selected={selectedDossierSection === "cap"} className={selectedDossierSection === "cap" ? "is-active" : ""} onClick={() => setSelectedDossierSection("cap")} role="tab" type="button">CAP &amp; Verification</button><button aria-selected={selectedDossierSection === "conversation"} className={selectedDossierSection === "conversation" ? "is-active" : ""} onClick={() => setSelectedDossierSection("conversation")} role="tab" type="button">Conversation <b>2</b></button><button aria-selected={selectedDossierSection === "files"} className={selectedDossierSection === "files" ? "is-active" : ""} onClick={() => setSelectedDossierSection("files")} role="tab" type="button">Files <b>3</b></button><button aria-selected={selectedDossierSection === "history"} className={selectedDossierSection === "history" ? "is-active" : ""} onClick={() => setSelectedDossierSection("history")} role="tab" type="button">History</button></div>
            {selectedDossierSection === "cap" ? <div aria-label={`CAP and verification for ${selectedFinding.findingNumber}`} className="inspector-finding-selected__body" role="region"><section><h3>CAP Summary</h3><p><b>Submitted by</b><span>{selectedFinding.organizationName} (Service Provider)</span></p><p><b>Status</b><span>{selectedFinding.status.replaceAll("_", " ")}</span></p><p><b>Due Date</b><span>{formatLocalDate(selectedFinding.dueDate)}</span></p><Link className="inspector-secondary-button inspector-secondary-button--primary" to={`/inspector/findings/${selectedFinding.id}`}>Open Finding dossier</Link></section><section className="inspector-finding-selected__verification"><h3>Inspector Verification</h3><small>Lead Inspector authority required</small><p>Review the submitted CAP and supporting evidence.</p><label>Comment to Auditee<textarea aria-label="Comment to Auditee unavailable" disabled placeholder="Required when returning the CAP" title="Lead Inspector authority is required to review this CAP." /></label><div><button aria-label="Accept CAP unavailable" className="inspector-secondary-button inspector-secondary-button--primary" disabled title="Lead Inspector authority is required to review this CAP." type="button">Accept CAP</button><button aria-label="Return for Revision unavailable" className="inspector-secondary-button" disabled title="Lead Inspector authority is required to review this CAP." type="button">Return for Revision</button></div></section></div> : null}
            {selectedDossierSection === "details" ? <div aria-label={`Finding details for ${selectedFinding.findingNumber}`} className="inspector-finding-selected__body" role="region"><section><h3>Finding Details</h3><p><b>Finding</b><span>{selectedFinding.findingNumber}</span></p><p><b>Finding basis</b><span>{selectedFinding.findingBasis}</span></p><p><b>Related Audit</b><span>{selectedFinding.auditId}</span></p></section></div> : null}
            {selectedDossierSection === "conversation" ? <div aria-label={`Conversation for ${selectedFinding.findingNumber}`} className="inspector-finding-selected__body" role="region"><section><h3>Conversation</h3><p><b>CAA-visible comments</b><span>2 recorded</span></p><p>Internal CAA Notes remain separate and are not shown to the Auditee.</p></section></div> : null}
            {selectedDossierSection === "files" ? <div aria-label={`Files for ${selectedFinding.findingNumber}`} className="inspector-finding-selected__body" role="region"><section><h3>Files</h3><p><b>Evidence files</b><span>Immutable versions are preserved by the server.</span></p><p>Open the Finding dossier to review the server-owned Evidence history.</p></section></div> : null}
            {selectedDossierSection === "history" ? <div aria-label={`History for ${selectedFinding.findingNumber}`} className="inspector-finding-selected__body" role="region"><section><h3>History</h3><p><b>Latest event</b><span>{selectedFinding.nextAction}</span></p><p>The immutable audit history remains available in the Finding dossier.</p></section></div> : null}
          </article> : null}
        </section>
      </div>
    </WorkspaceShell>
  );
}
