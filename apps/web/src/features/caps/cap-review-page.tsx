import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { CapRevisionView, FindingView, ReviewCapOutput, Role } from "../../backend/backend";
import { RoleHandoff } from "../../auth/role-handoff";
import { useOptionalSession } from "../../auth/session-provider";
import { createRoleEntryPath } from "../../ui/role-select-page";
import { DueState } from "../../ui/workbench/due-state";
import { StatusPill, type StatusPillTone } from "../../ui/workbench/status-pill";
import {
  CommandError,
  errorMessage,
  formatLocalDate,
  formatSeverity,
  WorkspaceShell,
} from "../shared/workspace-shell";

function statusTone(status: string): StatusPillTone {
  if (status === "ACCEPTED" || status === "EVIDENCE_REQUIRED") return "success";
  if (status.includes("REJECTED") || status.includes("MORE_INFORMATION")) return "warning";
  return "neutral";
}

function latestRevision(items: CapRevisionView[]): CapRevisionView | null {
  return [...items].sort((left, right) => right.revision - left.revision)[0] ?? null;
}

function reviewSummary(revision: CapRevisionView): string {
  if (!revision.latestReview) return "No CAA review yet";
  return `${revision.latestReview.decision}: ${revision.latestReview.commentToAuditee}`;
}

export function CapReviewPage() {
  const runtime = useApplicationRuntime();
  const leadBackend = useMemo(
    () => runtime.backendForRole?.("leadInspector") ?? runtime.backend,
    [runtime],
  );
  const session = useOptionalSession();
  const navigate = useNavigate();
  const { findingId } = useParams<{ findingId: string }>();
  const [finding, setFinding] = useState<FindingView | null>(null);
  const [capRevisions, setCapRevisions] = useState<CapRevisionView[]>([]);
  const [targetRevision, setTargetRevision] = useState<CapRevisionView | null>(null);
  const [capReview, setCapReview] = useState<ReviewCapOutput | null>(null);
  const [commentToAuditee, setCommentToAuditee] = useState("");
  const [internalCaaNote, setInternalCaaNote] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [quickActionsOpen, setQuickActionsOpen] = useState(false);
  const [reviewListCollapsed, setReviewListCollapsed] = useState(false);
  const [revisionQuery, setRevisionQuery] = useState("");
  const identityMode =
    session?.identityMode ??
    (runtime.identityMode ?? "oidc-session");
  const handoffSession = session?.state ?? { status: "unauthenticated" as const };
  const visibleCapRevisions = useMemo(() => {
    const query = revisionQuery.trim().toLowerCase();
    return [...capRevisions].reverse().filter((revision) =>
      !query || [
        `CAP Revision ${revision.revision}`,
        revision.rootCause,
        revision.correctiveAction,
        revision.status,
      ].join(" ").toLowerCase().includes(query),
    );
  }, [capRevisions, revisionQuery]);

  async function loadReviewTarget(): Promise<void> {
    if (!findingId) throw new Error("The CAP review route does not contain a finding identity.");
    const loadedFinding = await leadBackend.findings.get({ findingId });
    const revisionList = await leadBackend.caps.listRevisions({ findingId: loadedFinding.id });
    const latest = latestRevision(revisionList.items);
    const selected = latest
      ? await leadBackend.caps.getRevision({ capRevisionId: latest.id })
      : null;
    setFinding(loadedFinding);
    setCapRevisions(revisionList.items);
    setTargetRevision(selected);
  }

  useEffect(() => {
    let cancelled = false;
    void loadReviewTarget().catch((cause) => {
      if (!cancelled) setError(errorMessage(cause));
    });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [findingId, leadBackend]);

  function requestRole(role: Role): void {
    session?.setActiveRole(role);
    navigate(createRoleEntryPath(role));
  }

  function requestInspectorFinding(role: Role): void {
    session?.setActiveRole(role);
    if (finding) navigate(`/inspector/findings/${encodeURIComponent(finding.id)}`);
  }

  async function review(decision: "ACCEPT" | "REJECT" | "REQUEST_MORE_INFORMATION"): Promise<void> {
    if (!finding || !targetRevision) return;
    if (!commentToAuditee.trim()) {
      setError("Comment to Auditee is required");
      return;
    }
    if (!internalCaaNote.trim()) {
      setError("Internal CAA Note is required");
      return;
    }
    if (commentToAuditee.trim() === internalCaaNote.trim()) {
      setError("Comment to Auditee and Internal CAA Note must be separate");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const result = await leadBackend.caps.review({
        operationId: `OP-CAP-${decision}-${crypto.randomUUID()}`,
        capRevisionId: targetRevision.id,
        expectedCapRevision: targetRevision.revision,
        findingId: finding.id,
        expectedFindingRevision: finding.revision,
        decision,
        commentToAuditee,
        internalCaaNote,
      });
      setCapReview(result);
      const loadedFinding = await leadBackend.findings.get({ findingId: finding.id });
      const revisionList = await leadBackend.caps.listRevisions({ findingId: finding.id });
      const selected = await leadBackend.caps.getRevision({ capRevisionId: targetRevision.id });
      setFinding(loadedFinding);
      setCapRevisions(revisionList.items);
      setTargetRevision(selected);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  function exportReview(): void {
    if (!finding) return;
    const body = [
      `CAP Review — ${finding.findingNumber}`,
      finding.title,
      ...capRevisions.map((revision) => [
        `Revision ${revision.revision} — ${revision.status}`,
        `Root Cause: ${revision.rootCause}`,
        `Corrective Action: ${revision.correctiveAction}`,
        `Preventive Action: ${revision.preventiveAction}`,
      ].join("\n")),
    ].join("\n\n");
    const url = URL.createObjectURL(new Blob([body], { type: "text/plain;charset=utf-8" }));
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `${finding.findingNumber}-cap-review.txt`;
    anchor.click();
    URL.revokeObjectURL(url);
  }

  return (
    <WorkspaceShell roleLabel="Lead Inspector" routeLabel="CAP Review">
      <CommandError message={error} />
      {finding ? (
        <div className="cap-root-page">
          <span className="finding-semantic-context">Lead Inspector · {finding.organizationName}</span>
          <header className="cap-root-command workbench-page-header">
            <div className="cap-root-command__main">
              <p className="cap-root-breadcrumb">Dashboard › Findings › CAP Review › {finding.findingNumber}</p>
              <h1>CAP Review (Lead Inspector) <span>● Waiting for Review</span></h1>
              <p>Review the corrective action plan and inspector assessment, then make your decision.</p>
            </div>
            <div className="cap-root-command__metrics">
              <div className="is-info"><span>Current owner</span><b>Lead Inspector</b></div>
              <div className="is-warn"><span>Next action</span><b>Recommend, return, or request evidence</b></div>
              <div className="is-danger"><span>Due Date</span><b>{formatLocalDate(finding.dueDate)}</b></div>
              <div className="is-info"><span>Status</span><b>CAP Submitted — Pending CAA Review</b></div>
            </div>
            <div className="cap-root-command__actions">
              <button className="lead-root-button" onClick={exportReview} type="button">Export Review</button>
              <div className="cap-root-action-menu">
                <button aria-expanded={quickActionsOpen} className="lead-root-button lead-root-button--primary" onClick={() => setQuickActionsOpen((value) => !value)} type="button">Actions⌄</button>
                {quickActionsOpen ? <div role="menu"><button onClick={() => document.getElementById("cap-decision")?.scrollIntoView({ block: "start" })} role="menuitem" type="button">Open decision panel</button></div> : null}
              </div>
            </div>
          </header>

          <div className="cap-root-layout">
            <aside className="cap-root-list-panel">
              <header><h2>CAP Review List</h2><button aria-expanded={!reviewListCollapsed} aria-label={`${reviewListCollapsed ? "Expand" : "Collapse"} CAP Review List`} className="cap-root-collapse" onClick={() => setReviewListCollapsed((value) => !value)} type="button">⌁</button></header>
              {!reviewListCollapsed ? <>
                <div className="cap-root-list-tabs"><b>Canonical CAPs ({capRevisions.length})</b></div>
                <label className="cap-root-search"><span className="finding-semantic-context">Search CAP revisions</span><input onChange={(event) => setRevisionQuery(event.target.value)} placeholder="Search..." type="search" value={revisionQuery} /><i>⌕</i></label>
                <div className="cap-root-list-items">
                {visibleCapRevisions.map((revision) => (
                  <button
                    aria-pressed={revision.id === targetRevision?.id}
                    className={`cap-root-list-card${revision.id === targetRevision?.id ? " is-active" : ""}`}
                    key={revision.id}
                    onClick={() => setTargetRevision(revision)}
                    type="button"
                  >
                    <span><b>CAP Revision {revision.revision}</b><em className={revision.status === "SUBMITTED" ? "is-medium" : "is-low"}>● {revision.status === "SUBMITTED" ? "Medium" : "Low"}</em></span>
                    <small>Finding: {finding.findingNumber}</small>
                    <small>Dept: Cabin Safety</small>
                    <small>Due: {formatLocalDate(finding.dueDate)}</small>
                    <small>Inspector: Caner Yildiz</small>
                  </button>
                ))}
                {visibleCapRevisions.length === 0 ? <p>No CAP revisions match this search.</p> : null}
                </div>
              </> : null}
            </aside>

            <div className="cap-root-main">
              <section className="cap-root-panel">
                <header>
                  <h2>Finding Information</h2>
                  <RoleHandoff
                    identityMode={identityMode}
                    onRoleRequest={requestInspectorFinding}
                    session={handoffSession}
                    targetRole="inspector"
                  >
                    Switch to CAA Inspector Finding
                  </RoleHandoff>
                </header>
                <dl className="cap-root-info-grid">
                  <div><dt>Audit / Inspection</dt><dd>{finding.auditId}</dd></div>
                  <div><dt>Department</dt><dd>Cabin Safety</dd></div>
                  <div><dt>Finding ID</dt><dd>{finding.findingNumber}</dd></div>
                  <div><dt>Severity</dt><dd>{formatSeverity(finding.severity)}</dd></div>
                  <div><dt>Regulatory reference</dt><dd>{finding.regulatoryReference ?? "Configured reference"}</dd></div>
                  <div><dt>Due Date</dt><dd><DueState dueDate={finding.dueDate} today={new Date().toISOString().slice(0, 10)} /></dd></div>
                  <div className="is-wide"><dt>Finding Title</dt><dd>{finding.title}</dd></div>
                  <div><dt>Finding Raised On</dt><dd>15 Jun 2026</dd></div>
                  <div className="is-full"><dt>Finding Description</dt><dd>{finding.description}</dd></div>
                </dl>
              </section>

              {targetRevision ? (
                <article className="cap-root-panel cap-root-plan" data-testid="cap-review-target">
                  <div className="cap-root-plan-head">
                    <h2>Corrective Action Plan (CAP) by Service Provider</h2>
                    <span>Revision {targetRevision.revision}</span>
                    <StatusPill label={targetRevision.status} tone={statusTone(targetRevision.status)} />
                  </div>
                  <h3>Root Cause</h3><p>{targetRevision.rootCause}</p>
                  <h3>Corrective Action</h3><p>{targetRevision.correctiveAction}</p>
                  <h3>Preventive Action</h3><p>{targetRevision.preventiveAction}</p>
                  <dl className="cap-root-plan-meta">
                    <div><dt>Responsible Person</dt><dd>{targetRevision.responsiblePerson}</dd></div>
                    <div><dt>Target Completion Date</dt><dd>{formatLocalDate(targetRevision.targetCompletionDate)}</dd></div>
                    <div><dt>CAP Status</dt><dd>{targetRevision.status}</dd></div>
                    <div><dt>Comment to CAA</dt><dd>{targetRevision.commentToCaa}</dd></div>
                  </dl>
                </article>
              ) : <article className="lead-review-empty"><h2>No CAP revision is pending CAA review</h2><p>The CAP review route can load only submitted immutable CAP revisions.</p></article>}

              <section className="lead-review-history" aria-label="CAP revision register">
                <table>
                  <caption>CAP revision history</caption>
                  <thead><tr><th scope="col">Revision</th><th scope="col">Status</th><th scope="col">Root cause</th><th scope="col">CAA review</th><th scope="col">Internal CAA Note</th></tr></thead>
                  <tbody>{capRevisions.map((revision) => <tr data-testid="cap-revision-row" key={revision.id}><td>Revision {revision.revision}</td><td>{revision.status}</td><td>{revision.rootCause}</td><td>{reviewSummary(revision)}</td><td>{revision.audience === "CAA" && revision.latestReview ? revision.latestReview.internalCaaNote : "Not recorded"}</td></tr>)}</tbody>
                </table>
              </section>
            </div>

            <aside className="cap-root-side">
              <section className="cap-root-panel cap-root-quality">
                <h2>Lead Inspector Review</h2>
                <div><span>CAP Quality</span><b className="cap-root-stars">★ ★ ★ ★ ☆</b></div>
                <div><span>Evidence Completeness</span><strong>92%</strong><i><span /></i></div>
                <div><span>Root Cause Addressed</span><strong className="is-ok">● High</strong></div>
              </section>
              {targetRevision ? (
                <section className="cap-root-panel cap-root-decision" id="cap-decision">
                  <h2>Lead Inspector Decision</h2>
                  <div className="cap-root-decision-options"><label><input defaultChecked name="cap-choice" type="radio" /> <span><b>Recommend Closure (Approve CAP)</b><small>The CAP is adequate and effective.</small></span></label><label><input name="cap-choice" type="radio" /> <span><b>Request Revision</b><small>CAP requires changes before approval.</small></span></label><label><input name="cap-choice" type="radio" /> <span><b>Request More Evidence</b><small>Additional evidence is required.</small></span></label></div>
                  <label>Comment to Auditee<textarea rows={4} value={commentToAuditee} onChange={(event) => setCommentToAuditee(event.target.value)} /></label>
                  <label>Internal CAA Note<textarea rows={4} value={internalCaaNote} onChange={(event) => setInternalCaaNote(event.target.value)} /></label>
                  <div className="cap-root-decision-actions"><button className="lead-root-button lead-root-button--primary" disabled={busy} onClick={() => void review("ACCEPT")} type="button">Accept CAP</button><button className="lead-root-button" disabled={busy} onClick={() => void review("REQUEST_MORE_INFORMATION")} type="button">Request More Information</button><button className="lead-root-button" disabled={busy} onClick={() => void review("REJECT")} type="button">Reject CAP</button></div>
                </section>
              ) : null}
            </aside>
          </div>

          {targetRevision ? (
            capReview && finding ? (
            <section className="lead-review-result">
              <strong data-testid="finding-status">{finding.status}</strong>
              <span data-testid="closure-state">Finding remains open</span>
              {finding.evidenceRequired ? (
                <span>CAP accepted; Evidence remains required before closure.</span>
              ) : (
                <span>CAP accepted; CAA verification remains required before closure.</span>
              )}
              <RoleHandoff
                identityMode={identityMode}
                onRoleRequest={requestRole}
                session={handoffSession}
                targetRole="executiveDirector"
              >
                Check report authority
              </RoleHandoff>
            </section>
            ) : null
          ) : null}
        </div>
      ) : (
        <article className="lead-review-empty">
          <h2>Finding unavailable</h2>
          <p>The CAP review route could not load this Finding for the Lead Inspector.</p>
        </article>
      )}
    </WorkspaceShell>
  );
}
