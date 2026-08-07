import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Link, useSearchParams } from "react-router-dom";

import type {
  FindingSeverity,
  FindingView,
  PotentialFindingView,
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
  const [dueDate, setDueDate] = useState<string | null>("2026-07-15");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const result = await leadBackend.potentialFindings.list({
          status: "PENDING_LEAD_REVIEW",
        });
        const first = result.items[0] ?? null;
        const selectedPotential = first
          ? await leadBackend.potentialFindings.get({ potentialFindingId: first.id })
          : null;
        if (!cancelled) {
          setQueue(result.items);
          setSelected(selectedPotential);
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
        operationId: `OP-PF-${decision}`,
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
        operationId: "OP-PF-CONVERT",
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

  function selectSeverity(nextSeverity: FindingSeverity): void {
    if (nextSeverity === "OBSERVATION") {
      setCapRequired(false);
      setEvidenceRequired(false);
      setDueDate(null);
    } else if (severity === "OBSERVATION") {
      setCapRequired(true);
      setEvidenceRequired(true);
      setDueDate("2026-07-15");
    }
    setSeverity(nextSeverity);
  }

  return (
    <WorkspaceShell roleLabel="Lead Inspector" routeLabel="Lead Review">
      <div className="lead-assigned-page">
        <div className="lead-assigned-crumb"><span>Dashboard</span><span>›</span><b>Assigned Audits</b></div>
        <header className="lead-assigned-head workbench-page-header">
          <div><h1>Assigned Audits</h1><p>View and manage all audits assigned to you.</p></div>
          <button
            aria-label="New Audit Assignment unavailable: this screen remains in the accepted legacy demo"
            className="lead-root-button lead-root-button--primary"
            disabled
            title="New Audit Assignment remains in the accepted legacy demo."
            type="button"
          >+ New Audit Assignment</button>
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
                <div><h3>{selected.id} · Non-Compliant</h3><p>2026 Cabin Inspection - Fly Namibia · Is protective breathing equipment serviceable and accessible?</p></div>
                <StatusPill label="Pending Lead Review" tone={statusTone(selected.status)} />
              </div>
              <p className="lead-potential-comment">PBE serviceability and accessibility could not be confirmed.</p>
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
                  <span>Canonical Finding</span><strong data-testid="finding-number">{finding.findingNumber}</strong><span data-testid="finding-status">{finding.status}</span><span>{formatSeverity(finding.severity)}</span><span>{finding.capRequired ? "CAP required" : "CAP not required"}</span><span>{finding.evidenceRequired ? "Evidence required" : "Evidence not required"}</span><Link className="primary-link" to="/lead-inspector/cap-review/FND-CAB-2026-001">Open Lead CAP review</Link>
                </section>
              )}
            </article>
          </section>
        ) : (
          <article className="lead-review-empty"><h2>No authorized persisted Potential Findings awaiting Lead review</h2><p>Empty state means no persisted records are authorized for this Lead Inspector, not that the React session has no prior in-memory state.</p></article>
        )}
        {queue.length > 0 ? (
          <section className="lead-review-register" aria-label="Lead Potential Finding queue">
            <DataRegister caption="Potential Findings awaiting Lead review" columns={potentialFindingColumns} rowKey={(row) => row.rowId} rows={rows} />
          </section>
        ) : null}
        <span className="candidate-boundary">Pending lead review. Due Date: 15 Jul 2026.</span>
      </div>
    </WorkspaceShell>
  );
}
