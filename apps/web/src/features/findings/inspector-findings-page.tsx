import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

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

const acceptedQueueFillers = [
  ["OPS-2025-014", "Pre-flight documentation filing incomplete", "Closed", "10 Dec 2025"],
  ["SEC-2026-002", "Access control log gaps at cargo gate", "CAP Submitted — Pending CAA Review", "24 Jun 2026"],
  ["AWO-2026-003", "Maintenance task sign-off overdue", "CAP Accepted — Evidence Required", "20 May 2026"],
  ["RAMP-2026-005", "Ground equipment inspection tags missing", "Evidence Submitted — Pending Review", "27 Jun 2026"],
  ["CAB-2026-012", "Cabin crew training sample missing recurrent check evidence", "CAP Accepted — Evidence Required", "24 Jun 2026"],
  ["CAB-2026-013", "Cabin defect follow-up owner not recorded", "Waiting for CAP", "27 Jun 2026"],
  ["CAB-2026-014", "Cabin inspection document index improvement", "Closed", "20 Jun 2026"],
] as const;

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
    const today = Date.parse("2026-06-15T00:00:00Z");
    if (dueFilter === "overdue" && (due === null || due >= today)) return false;
    if (dueFilter === "due-soon" && (due === null || due < today || due > today + 30 * 86_400_000)) return false;
    return `${finding.findingNumber} ${finding.title} ${finding.organizationName}`.toLowerCase().includes(query.trim().toLowerCase());
  });
  const cabFinding = findings.find((finding) => finding.id === "FND-CAB-2026-001") ?? null;
  const filtersAreDefault =
    summaryFilter === "all" &&
    severityFilter === "all" &&
    statusFilter === "all" &&
    dueFilter === "all" &&
    query === "";
  const visibleQueueCount = filtersAreDefault ? visible.length + acceptedQueueFillers.length : visible.length;
  const exportHref = `data:text/csv;charset=utf-8,${encodeURIComponent(findingsCsv(visible))}`;

  return (
    <WorkspaceShell roleLabel="CAA Inspector" routeLabel="Findings">
      <div className="inspector-secondary-page inspector-findings-page" data-testid="inspector-findings-page">
        <div className="inspector-breadcrumbs" aria-label="Finding breadcrumb">{cabFinding ? <><span>{cabFinding.organizationName}</span><span>{cabFinding.auditId}</span><span>{cabFinding.title}</span></> : null}<b>Findings</b></div>
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
          <button aria-pressed={summaryFilter === "all"} className={summaryFilter === "all" ? "is-active" : ""} onClick={() => setSummaryFilter("all")} type="button"><span>▤</span><b>All Findings</b><strong>9</strong></button>
          <button aria-pressed={summaryFilter === "waiting-cap"} className={summaryFilter === "waiting-cap" ? "is-active" : ""} onClick={() => setSummaryFilter("waiting-cap")} type="button"><span>⌛</span><b>Waiting for CAP</b><strong>1</strong></button>
          <button aria-pressed={summaryFilter === "cap-submitted"} className={summaryFilter === "cap-submitted" ? "is-active" : ""} onClick={() => setSummaryFilter("cap-submitted")} type="button"><span>➤</span><b>CAP Submitted</b><strong>5</strong></button>
          <button aria-pressed={summaryFilter === "returned"} className={summaryFilter === "returned" ? "is-active" : ""} onClick={() => setSummaryFilter("returned")} type="button"><span>↩</span><b>Returned</b><strong>0</strong></button>
          <button aria-pressed={summaryFilter === "closed"} className={summaryFilter === "closed" ? "is-active" : ""} onClick={() => setSummaryFilter("closed")} type="button"><span>✓</span><b>Closed</b><strong>3</strong></button>
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
            {visible.map((finding, index) => <article className="inspector-record-card inspector-mobile-stack" data-current-owner-role={finding.currentOwnerRole} data-due-date={finding.dueDate ?? ""} data-finding-id={finding.id} data-next-action={finding.nextAction} key={`${finding.id}-${index}`}>
              <div className="inspector-record-card__title"><div><b>{finding.findingNumber}</b><h3>{finding.title}</h3><p>{finding.organizationName} · {formatSeverity(finding.severity)}</p></div><StatusPill label={finding.status.replaceAll("_", " ")} tone={finding.status === "CLOSED" ? "success" : "warning"} /></div>
              <dl className="inspector-decision-grid">
                <div><dt>Current Owner</dt><dd>{ownerLabel(finding)}</dd></div>
                <div><dt>Next Action</dt><dd>{finding.nextAction}</dd></div>
                <div><dt>Due Date</dt><dd><DueState dueDate={finding.dueDate} today="2026-06-15" /></dd></div>
                <div><dt>Related Audit</dt><dd>{finding.auditId}</dd></div>
              </dl>
              {finding.id === "FND-CAB-2026-001" ? (
                <Link className="inspector-secondary-button inspector-secondary-button--primary" to="/inspector/findings/FND-CAB-2026-001">Open Finding dossier</Link>
              ) : (
                <button
                  aria-label={`Finding dossier unavailable for ${finding.findingNumber}`}
                  className="inspector-secondary-button"
                  disabled
                  title={`Finding ${finding.findingNumber} does not have a declared Inspector Finding Detail route.`}
                  type="button"
                >
                  Dossier unavailable
                </button>
              )}
            </article>)}
            {filtersAreDefault ? acceptedQueueFillers.map(([id, title, status, date]) => <article className="inspector-finding-queue-filler" key={id}><div><b>{id}</b><span>{status}</span></div><h3>{title}</h3><p>Configured reference · Level 3 Minor</p><footer><b>{date}</b><span>CAA review queue</span></footer></article>) : null}
          </div>
          {cabFinding ? <article aria-label={`Selected Finding ${cabFinding.findingNumber}`} className="inspector-finding-selected" data-current-owner-role={cabFinding.currentOwnerRole} data-due-date={cabFinding.dueDate ?? ""} data-finding-id={cabFinding.id} data-next-action={cabFinding.nextAction}>
            <header><div><h2>{cabFinding.findingNumber} {cabFinding.title}</h2><p>⌘ &nbsp; {cabFinding.regulatoryReference ?? cabFinding.findingBasis} &nbsp; ▣</p><small>Raised on {formatLocalDate(cabFinding.issuedAt?.slice(0, 10) ?? cabFinding.createdAt.slice(0, 10))} by CAA Inspector</small></div><span>● {formatSeverity(cabFinding.severity)}</span></header>
            <span className="inspector-finding-selected__status">● {cabFinding.status.replaceAll("_", " ")}</span>
            <dl><div><dt>Current Owner</dt><dd>{ownerLabel(cabFinding)}</dd></div><div><dt>Next Action</dt><dd>{cabFinding.nextAction}</dd></div><div><dt>Due Date</dt><dd><DueState dueDate={cabFinding.dueDate} today="2026-06-15" /></dd></div><div><dt>Organization</dt><dd>{cabFinding.organizationName}</dd></div></dl>
            <div aria-label="Finding dossier sections" className="inspector-finding-selected__sections" role="tablist"><button aria-selected={selectedDossierSection === "details"} className={selectedDossierSection === "details" ? "is-active" : ""} onClick={() => setSelectedDossierSection("details")} role="tab" type="button">Details</button><button aria-selected={selectedDossierSection === "cap"} className={selectedDossierSection === "cap" ? "is-active" : ""} onClick={() => setSelectedDossierSection("cap")} role="tab" type="button">CAP &amp; Verification</button><button aria-selected={selectedDossierSection === "conversation"} className={selectedDossierSection === "conversation" ? "is-active" : ""} onClick={() => setSelectedDossierSection("conversation")} role="tab" type="button">Conversation <b>2</b></button><button aria-selected={selectedDossierSection === "files"} className={selectedDossierSection === "files" ? "is-active" : ""} onClick={() => setSelectedDossierSection("files")} role="tab" type="button">Files <b>3</b></button><button aria-selected={selectedDossierSection === "history"} className={selectedDossierSection === "history" ? "is-active" : ""} onClick={() => setSelectedDossierSection("history")} role="tab" type="button">History</button></div>
            {selectedDossierSection === "cap" ? <div aria-label={`CAP and verification for ${cabFinding.findingNumber}`} className="inspector-finding-selected__body" role="region"><section><h3>CAP Summary</h3><p><b>Submitted by</b><span>{cabFinding.organizationName} (Service Provider)</span></p><p><b>Status</b><span>{cabFinding.status.replaceAll("_", " ")}</span></p><p><b>Due Date</b><span>{formatLocalDate(cabFinding.dueDate)}</span></p><Link className="inspector-secondary-button inspector-secondary-button--primary" to={`/inspector/findings/${cabFinding.id}`}>Open Finding dossier</Link></section><section className="inspector-finding-selected__verification"><h3>Inspector Verification</h3><small>Lead Inspector authority required</small><p>Review the submitted CAP and supporting evidence.</p><label>Comment to Auditee<textarea aria-label="Comment to Auditee unavailable" disabled placeholder="Required when returning the CAP" title="Lead Inspector authority is required to review this CAP." /></label><div><button aria-label="Accept CAP unavailable" className="inspector-secondary-button inspector-secondary-button--primary" disabled title="Lead Inspector authority is required to review this CAP." type="button">Accept CAP</button><button aria-label="Return for Revision unavailable" className="inspector-secondary-button" disabled title="Lead Inspector authority is required to review this CAP." type="button">Return for Revision</button></div></section></div> : null}
            {selectedDossierSection === "details" ? <div aria-label={`Finding details for ${cabFinding.findingNumber}`} className="inspector-finding-selected__body" role="region"><section><h3>Finding Details</h3><p><b>Finding</b><span>{cabFinding.findingNumber}</span></p><p><b>Finding basis</b><span>{cabFinding.findingBasis}</span></p><p><b>Related Audit</b><span>{cabFinding.auditId}</span></p></section></div> : null}
            {selectedDossierSection === "conversation" ? <div aria-label={`Conversation for ${cabFinding.findingNumber}`} className="inspector-finding-selected__body" role="region"><section><h3>Conversation</h3><p><b>CAA-visible comments</b><span>2 recorded</span></p><p>Internal CAA Notes remain separate and are not shown to the Auditee.</p></section></div> : null}
            {selectedDossierSection === "files" ? <div aria-label={`Files for ${cabFinding.findingNumber}`} className="inspector-finding-selected__body" role="region"><section><h3>Files</h3><p><b>Evidence files</b><span>3 versions preserved</span></p><p>Open the Finding dossier to review the mock evidence history.</p></section></div> : null}
            {selectedDossierSection === "history" ? <div aria-label={`History for ${cabFinding.findingNumber}`} className="inspector-finding-selected__body" role="region"><section><h3>History</h3><p><b>Latest event</b><span>{cabFinding.nextAction}</span></p><p>The immutable audit history remains available in the Finding dossier.</p></section></div> : null}
          </article> : null}
        </section>
      </div>
    </WorkspaceShell>
  );
}
