import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Link, useSearchParams } from "react-router-dom";

import type {
  FindingSeverity,
  FindingView,
  PotentialFindingView,
  ReportVersionView,
} from "../../backend/backend";
import { useApplicationRuntime } from "../../app/providers";
import { DataRegister, type DataRegisterColumn } from "../../ui/workbench/data-register";
import { StatusPill, type StatusPillTone } from "../../ui/workbench/status-pill";
import {
  CommandError,
  errorMessage,
  formatSeverity,
  WorkspaceShell,
} from "../shared/workspace-shell";
import { AuditAssignmentPage } from "../teams/audit-assignment-page";

interface PotentialFindingRow extends Record<string, ReactNode> {
  id: ReactNode;
  audit: ReactNode;
  question: ReactNode;
  status: ReactNode;
  nextAction: ReactNode;
  rowId: string;
}

const potentialFindingColumns: readonly DataRegisterColumn<PotentialFindingRow>[] = [
  { key: "id", header: "Potential Finding" },
  { key: "audit", header: "Audit" },
  { key: "question", header: "Question" },
  { key: "status", header: "Status" },
  { key: "nextAction", header: "Next action" },
];

function statusLabel(value: string): string {
  return value.replaceAll("_", " ");
}

function statusTone(value: string): StatusPillTone {
  if (value.includes("CONVERTED")) return "success";
  if (value.includes("PENDING")) return "warning";
  if (value.includes("RETURNED")) return "warning";
  if (value.includes("DISMISSED")) return "danger";
  return "neutral";
}

function newOpaqueIdentity(prefix: string): string {
  return `${prefix}:${globalThis.crypto.randomUUID()}`;
}

const preliminaryReportContent = (auditId: string) => ({
  schema: "avia.report-content/v1",
  languageTag: "en",
  title: "Preliminary Inspection Report",
  executiveSummary: "The released inspection checklist has been executed and is ready for governed report review.",
  scope: `Canonical inspection ${auditId}`,
  methodology: "The assigned Inspector executed the immutable released question set and recorded server-owned responses.",
  sections: [{
    id: "execution-summary",
    heading: "Inspection execution",
    paragraphs: ["This Preliminary Report records the submitted checklist boundary. Finding conversion remains a later governed action."],
  }],
  findings: [],
  conclusion: "The inspection evidence is ready for the declared Preliminary Report approval path.",
  recommendations: ["Review and issue this immutable Preliminary Report before converting any Potential Finding."],
});

/**
 * The Lead preparation hand-off shares the role-owned Lead workspace route.
 * Keeping the assignment identity in the query makes the route server-owned
 * instead of embedding a fixture audit id in the manager's link.
 */
export function LeadReviewPage() {
  const [searchParams] = useSearchParams();
  return searchParams.get("assignmentId") ? <AuditAssignmentPage /> : <LeadReviewQueuePage />;
}

function LeadReviewQueuePage() {
  const runtime = useApplicationRuntime();
  const leadBackend = useMemo(
    () => runtime.backendForRole?.("leadInspector") ?? runtime.backend,
    [runtime],
  );
  const [queue, setQueue] = useState<PotentialFindingView[]>([]);
  const [selected, setSelected] = useState<PotentialFindingView | null>(null);
  const [finding, setFinding] = useState<FindingView | null>(null);
  const [reason, setReason] = useState("");
  const [severity, setSeverity] = useState<FindingSeverity>("LEVEL_1_CRITICAL");
  const [capRequired, setCapRequired] = useState(true);
  const [evidenceRequired, setEvidenceRequired] = useState(true);
  const [dueDate, setDueDate] = useState<string | null>(() => new Date(Date.now() + 30 * 86_400_000).toISOString().slice(0, 10));
  const [preliminaryReport, setPreliminaryReport] = useState<ReportVersionView | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const result = await leadBackend.potentialFindings.list({
          status: "PENDING_LEAD_REVIEW",
        });
        if (!cancelled) {
          setQueue(result.items);
          setSelected(null);
          setFinding(null);
        }
      } catch (cause) {
        if (!cancelled) setError(errorMessage(cause));
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [leadBackend]);

  async function selectPotentialFinding(id: string): Promise<void> {
    setBusy(true);
    setError(null);
    try {
      setSelected(await leadBackend.potentialFindings.get({ potentialFindingId: id }));
      setFinding(null);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  const rows = useMemo<PotentialFindingRow[]>(
    () =>
      queue.map((item) => ({
        rowId: item.id,
        id: item.id,
        audit: item.auditId,
        question: item.questionId,
        status: <StatusPill label={statusLabel(item.status)} tone={statusTone(item.status)} />,
        nextAction: item.convertedFindingId ? "Open Finding dossier" : "Lead decision required",
      })),
    [queue],
  );

  async function decide(decision: "RETURN" | "DISMISS"): Promise<void> {
    if (!selected) return;
    if (!reason.trim()) {
      setError("Lead decision reason is required");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const result = await leadBackend.potentialFindings.decide({
        operationId: `POTENTIAL-FINDING-${selected.id}-${selected.revision}-${decision}`,
        potentialFindingId: selected.id,
        expectedPotentialFindingRevision: selected.revision,
        decision,
        reason,
      });
      setSelected(result.potentialFinding);
      setQueue((current) => current.filter((item) => item.id !== selected.id));
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  async function convert(): Promise<void> {
    if (!selected) return;
    setBusy(true);
    setError(null);
    try {
      const result = await leadBackend.potentialFindings.decide({
        operationId: `POTENTIAL-FINDING-${selected.id}-${selected.revision}-CONVERT`,
        potentialFindingId: selected.id,
        expectedPotentialFindingRevision: selected.revision,
        decision: "CONVERT",
        severity,
        capRequired,
        evidenceRequired,
        dueDate,
      });
      setSelected(result.potentialFinding);
      setFinding(result.finding);
      setQueue((current) => current.filter((item) => item.id !== selected.id));
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  async function createPreliminaryReport(): Promise<void> {
    if (!selected) return;
    setBusy(true);
    setError(null);
    try {
      const reportId = newOpaqueIdentity("report");
      const reportVersionId = newOpaqueIdentity("report-version");
      const created = await leadBackend.reports.create({
        operationId: `CREATE-PRELIMINARY-${reportVersionId}`,
        idempotencyKey: `CREATE-PRELIMINARY-${reportVersionId}`,
        expectedRevision: null,
        reportVersionId,
        reportId,
        auditId: selected.auditId,
        kind: "PRELIMINARY",
        version: 1,
        status: "DEPARTMENT_REVIEW",
        findingIds: [],
        content: preliminaryReportContent(selected.auditId),
      });
      setPreliminaryReport(created);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  function selectSeverity(nextSeverity: FindingSeverity): void {
    if (nextSeverity === "OBSERVATION") {
      setCapRequired(false);
      setEvidenceRequired(false);
      setDueDate(null);
    } else if (severity === "OBSERVATION") {
      setCapRequired(true);
      setEvidenceRequired(true);
        setDueDate(new Date(Date.now() + 30 * 86_400_000).toISOString().slice(0, 10));
    }
    setSeverity(nextSeverity);
  }

  return (
    <WorkspaceShell roleLabel="Lead Inspector" routeLabel="Lead Review">
      <div className="lead-assigned-page">
        <div className="lead-assigned-crumb"><span>Dashboard</span><span>›</span><b>Assigned Audits</b></div>
        <header className="lead-assigned-head workbench-page-header">
          <div><h1>Assigned Audits</h1><p>View and manage server-owned audits assigned to you. New assignments are created by the Department Manager.</p></div>
        </header>
        <section className="lead-assigned-kpis" aria-label="Lead assignment summary">
          {[
            ["Total Assigned", String(queue.length), "100", "total", "▣"],
            ["In Progress", String(queue.length), "100", "progress", "◷"],
            ["Reports", "0", "0", "draft", "▤"],
            ["Upcoming", "0", "0", "pending", "➤"],
            ["Overdue", "0", "0", "overdue", "!"],
          ].map(([title, value, percent, tone, icon]) => (
            <article className={`lead-assigned-card is-${tone}`} key={title}>
              <span className="lead-assigned-card__icon" aria-hidden="true">{icon}</span>
              <span className="lead-assigned-card__body"><span>{title}</span><strong>{value}</strong><em> Audits</em><small>{percent}% of all</small><i><span style={{ width: `${percent}%` }} /></i></span>
            </article>
          ))}
        </section>
        <CommandError message={error} />
        {selected ? (
          <section className="lead-potential-panel" data-testid="potential-finding-dossier">
            <header><h2>Pending Inspector Finding Decisions</h2><span>Lead only</span></header>
              <article className="lead-potential-card">
              <div className="lead-potential-card__head">
                <div><h3>{selected.id}</h3><p>{selected.title} · {selected.organizationId}</p></div>
                <StatusPill label="Pending Lead Review" tone={statusTone(selected.status)} />
              </div>
              <p className="lead-potential-comment">{selected.description}</p>
              {!preliminaryReport ? <div className="lead-report-preparation">
                <h3>Preliminary Report</h3>
                <p>Create the immutable Preliminary Report after the Inspector submits the checklist. Finding issuance remains blocked until this version is approved and issued.</p>
                <button className="lead-root-button" disabled={busy} onClick={() => void createPreliminaryReport()} type="button">Create Preliminary Report</button>
              </div> : <section aria-label="Created Preliminary Report" className="lead-report-preparation" data-testid="created-preliminary-report">
                <h3>Preliminary Report created</h3>
                <p data-testid="preliminary-report-version-id">{preliminaryReport.reportVersionId}</p>
                <span data-testid="preliminary-report-status">{preliminaryReport.status}</span>
                <Link className="primary-link" to={`/lead-inspector/preliminary-reports/${encodeURIComponent(preliminaryReport.reportVersionId)}`}>Open Preliminary Report</Link>
              </section>}
              {!finding ? (
                <div className="lead-potential-form">
                  <label>Finding title<input readOnly title="Finding title is sourced from the persisted Potential Finding." value={selected.title} /></label>
                  <label>Lead severity <span aria-hidden="true">*</span><select aria-label="Finding severity" value={severity} onChange={(event) => selectSeverity(event.target.value as FindingSeverity)}><option value="LEVEL_1_CRITICAL">Level 1 Critical</option><option value="LEVEL_2_MAJOR">Level 2 Major</option><option value="LEVEL_3_MINOR">Level 3 Minor</option><option value="OBSERVATION">Observation</option></select></label>
                  <div className="lead-potential-checks">
                    <label><input checked={capRequired} onChange={(event) => setCapRequired(event.target.checked)} type="checkbox" /> CAP required</label>
                    <label><input checked={evidenceRequired} onChange={(event) => setEvidenceRequired(event.target.checked)} type="checkbox" /> Evidence required</label>
                  </div>
                  <label>Due Date<input aria-label="Finding Due Date" onChange={(event) => setDueDate(event.target.value || null)} type="date" value={dueDate ?? ""} /><small>Observation defaults clear CAP, Evidence, and Due Date; the Lead Inspector may explicitly enable them.</small></label>
                  <label>Reason for return/dismissal<textarea aria-label="Lead decision reason" onChange={(event) => setReason(event.target.value)} placeholder="Required only for Return or Dismiss." rows={3} value={reason} /></label>
                  <div className="lead-potential-actions">
                    <button className="lead-root-button lead-root-button--primary" disabled={busy} onClick={() => void convert()} type="button">Convert to Finding</button>
                    <button className="lead-root-button" disabled={busy} onClick={() => void decide("RETURN")} type="button">Return Potential Finding</button>
                    <button className="lead-root-button lead-root-button--danger" disabled={busy} onClick={() => void decide("DISMISS")} type="button">Dismiss Potential Finding</button>
                  </div>
                </div>
              ) : (
                <section className="lead-review-result" data-testid="lead-decision-result">
                  <span>Finding</span><strong data-testid="finding-number">{finding.findingNumber}</strong><span data-testid="finding-status">{finding.status}</span><span>{formatSeverity(finding.severity)}</span><span>{finding.capRequired ? "CAP required" : "CAP not required"}</span><span>{finding.evidenceRequired ? "Evidence required" : "Evidence not required"}</span><Link className="primary-link" to={`/lead-inspector/cap-review/${encodeURIComponent(finding.id)}`}>Open Lead CAP review</Link>
                </section>
              )}
            </article>
          </section>
        ) : (
          <article className="lead-review-empty"><h2>No authorized persisted Potential Findings awaiting Lead review</h2><p>Empty state means no persisted records are authorized for this Lead Inspector, not that the React session has no prior in-memory state.</p></article>
        )}
        {queue.length > 0 ? (
        <section className="lead-review-register" aria-label="Lead Potential Finding queue">
            <div className="button-row">
              {queue.map((item) => <button disabled={busy} key={item.id} onClick={() => void selectPotentialFinding(item.id)} type="button">{item.id} · {statusLabel(item.status)}</button>)}
            </div>
            <DataRegister caption="Potential Findings awaiting Lead review" columns={potentialFindingColumns} rowKey={(row) => row.rowId} rows={rows} />
        </section>
      ) : null}
        <span className="qualification-boundary">Pending Lead review uses the persisted Potential Finding state.</span>
      </div>
    </WorkspaceShell>
  );
}
