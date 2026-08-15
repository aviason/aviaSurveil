import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { FindingView, Role } from "../../backend/backend";
import { RoleHandoff } from "../../auth/role-handoff";
import { useOptionalSession } from "../../auth/session-provider";
import { createRoleEntryPath } from "../../ui/role-select-page";
import { DueState } from "../../ui/workbench/due-state";
import { LifecycleStepper } from "../../ui/workbench/lifecycle-stepper";
import {
  CommandError,
  errorMessage,
  formatLocalDate,
  formatSeverity,
  WorkspaceShell,
} from "../shared/workspace-shell";

const lifecycleStages = [
  { id: "finding-issued", label: "Finding Issued" },
  { id: "cap-submitted", label: "CAP Submitted" },
  { id: "cap-reviewed", label: "CAP Reviewed" },
  { id: "evidence-submitted", label: "Evidence Submitted" },
  { id: "evidence-accepted", label: "Evidence Accepted" },
  { id: "closed", label: "Closed" },
] as const;

function lifecycleStageFor(finding: FindingView): string {
  if (finding.status === "CLOSED") return "closed";
  if (finding.status === "PENDING_CAA_REVIEW") return "evidence-accepted";
  if (finding.status === "EVIDENCE_SUBMITTED") return "evidence-submitted";
  if (
    finding.status === "EVIDENCE_REQUIRED" ||
    finding.status === "EVIDENCE_MORE_INFORMATION_REQUESTED"
  ) return "cap-reviewed";
  if (
    finding.status === "CAP_SUBMITTED" ||
    finding.status === "CAP_ACCEPTED" ||
    finding.status === "CAP_REJECTED" ||
    finding.status === "CAP_MORE_INFORMATION_REQUESTED"
  ) return "cap-submitted";
  return "finding-issued";
}

function ownerLabel(finding: FindingView): string {
  return finding.currentOwnerType === "AUDITEE" ? finding.organizationName : "CAA Inspector";
}

function ownerDetailLabel(finding: FindingView): string {
  return finding.currentOwnerType === "AUDITEE"
    ? `Auditee — ${finding.organizationName}`
    : "CAA Inspector";
}

function statusLabel(status: FindingView["status"]): string {
  const labels: Partial<Record<FindingView["status"], string>> = {
    WAITING_FOR_CAP: "Waiting for CAP",
    CAP_SUBMITTED: "CAP Submitted — Pending CAA Review",
    CAP_ACCEPTED: "CAP Accepted",
    CAP_REJECTED: "CAP Rejected",
    CAP_MORE_INFORMATION_REQUESTED: "CAP — More Information Requested",
    EVIDENCE_REQUIRED: "Evidence Required",
    EVIDENCE_SUBMITTED: "Evidence Submitted — Pending Review",
    EVIDENCE_MORE_INFORMATION_REQUESTED: "Evidence — More Information Requested",
    PENDING_CAA_REVIEW: "Pending CAA Review",
    CLOSED: "Closed",
  };
  return labels[status] ?? status;
}

function primaryActionLabel(finding: FindingView): string {
  if (finding.status === "CAP_SUBMITTED") return "Open CAP review handoff";
  if (finding.status === "EVIDENCE_SUBMITTED") return "Review evidence";
  return "View finding";
}

function displayNextAction(finding: FindingView): string {
  if (finding.status === "CAP_SUBMITTED") return "Lead Inspector to review CAP";
  if (finding.status === "EVIDENCE_SUBMITTED") return "Review evidence";
  return finding.nextAction;
}

function dueTiming(finding: FindingView): string {
  if (!finding.dueDate) return "";
  const due = Date.parse(`${finding.dueDate}T00:00:00Z`);
  const today = Date.parse(`${new Date().toISOString().slice(0, 10)}T00:00:00Z`);
  const days = Math.round((due - today) / 86_400_000);
  if (days < 0) return ` (${Math.abs(days)} days overdue)`;
  if (days === 0) return " (Due today)";
  return ` (Due in ${days} days)`;
}

export function FindingDetailPage() {
  const runtime = useApplicationRuntime();
  const inspectorBackend = useMemo(
    () => runtime.backendForRole?.("inspector") ?? runtime.backend,
    [runtime],
  );
  const session = useOptionalSession();
  const navigate = useNavigate();
  const { findingId } = useParams<{ findingId: string }>();
  const [finding, setFinding] = useState<FindingView | null>(null);
  const [error, setError] = useState<string | null>(null);
  const identityMode =
    session?.identityMode ??
    (runtime.identityMode ?? "oidc-session");
  const handoffSession = session?.state ?? { status: "unauthenticated" as const };

  useEffect(() => {
    let cancelled = false;
    if (!findingId) {
      setError("The route does not contain a finding identity.");
      return () => { cancelled = true; };
    }
    void inspectorBackend.findings
      .get({ findingId })
      .then((loaded) => {
        if (!cancelled) setFinding(loaded);
      })
      .catch((cause) => {
        if (!cancelled) setError(errorMessage(cause));
      });
    return () => {
      cancelled = true;
    };
  }, [findingId, inspectorBackend]);

  function requestRole(role: Role): void {
    session?.setActiveRole(role);
    navigate(createRoleEntryPath(role));
  }

  function requestLeadCapReview(role: Role): void {
    session?.setActiveRole(role);
    if (finding) navigate(`/lead-inspector/cap-review/${encodeURIComponent(finding.id)}`);
  }

  function activatePrimaryAction(): void {
    if (!finding) return;
    document.getElementById(finding.status === "CAP_SUBMITTED" ? "finding-cap" : "finding-details")?.scrollIntoView({ block: "start" });
  }

  const displayFindingNumber = finding?.findingNumber;

  return (
    <WorkspaceShell roleLabel="CAA Inspector" routeLabel="Finding Detail">
      <CommandError message={error} />
      {finding ? (
        <div className="finding-root-page" data-testid="finding-dossier">
          <span className="finding-semantic-context">CAA Inspector</span>
          <header className="finding-root-head workbench-page-header">
            <div>
              <h1>Finding {displayFindingNumber}</h1>
              <p>{finding.title}</p>
            </div>
            <button
              className="lead-root-button"
              onClick={() => navigate("/inspector/findings")}
              type="button"
            >
              Back to findings
            </button>
          </header>

          <section className="finding-root-next" aria-label="Next action">
            <span aria-hidden="true">→</span>
            <p>
              <b>Next action:</b> {displayNextAction(finding)} · Owner: {ownerLabel(finding)} · {" "}
              <b>Due Date:</b> {formatLocalDate(finding.dueDate)}{dueTiming(finding)}
            </p>
            <button
              aria-label={finding.status === "CAP_SUBMITTED" ? "Open CAP review handoff" : primaryActionLabel(finding)}
              className="lead-root-button lead-root-button--primary"
              onClick={activatePrimaryAction}
              type="button"
            >
              {finding.status === "CAP_SUBMITTED" ? <><span>Review CAP</span><span className="finding-semantic-context">Open CAP review handoff</span></> : primaryActionLabel(finding)}
            </button>
          </section>

          <section className="finding-mobile-summary" aria-label="Mobile decision summary">
            <div><span>Current owner</span><b>{ownerDetailLabel(finding)}</b></div>
            <div><span>Next action</span><b>{finding.status === "CAP_SUBMITTED" ? displayNextAction(finding) : `Next: ${displayNextAction(finding)}`}</b></div>
            <div><span>Due Date</span><b>{formatLocalDate(finding.dueDate)}</b></div>
            <div><span>Status</span><b>{statusLabel(finding.status)}</b></div>
            <button aria-label={finding.status === "CAP_SUBMITTED" ? "Open CAP review handoff" : primaryActionLabel(finding)} className="lead-root-button lead-root-button--primary" onClick={activatePrimaryAction} type="button">
              {finding.status === "CAP_SUBMITTED" ? <><span>Review CAP</span><span className="finding-semantic-context">Open CAP review handoff</span></> : primaryActionLabel(finding)}
            </button>
          </section>

          <section className="finding-caa-actions">
            <span>⚖️ <b>CAA actions</b> — nudge the Auditee with a traceable reminder. Department Manager owns any explicit authorized-closure decision; Evidence-backed closure remains the normal path.</span>
          </section>

          <section className="finding-root-lifecycle">
            <LifecycleStepper
              ariaLabel="Finding lifecycle"
              currentStageId={lifecycleStageFor(finding)}
              stages={lifecycleStages}
            />
            <p>CAP accepted is not closure - a finding closes only after evidence is accepted, verification is completed, or an authorized closure is recorded.</p>
          </section>

          <div className="finding-dossier-grid">
            <div className="finding-dossier-stack">
              <section className="finding-dossier-panel" id="finding-details">
                <header><h2>Finding Details</h2></header>
                <div className="finding-dossier-panel__body">
                  <dl className="finding-meta-grid">
                    <div><dt>Status</dt><dd>{statusLabel(finding.status)}</dd><dd className="finding-raw-status">{finding.status}</dd></div>
                    <div><dt>Severity</dt><dd>{formatSeverity(finding.severity)}</dd></div>
                    <div><dt>Current owner</dt><dd>{ownerDetailLabel(finding)}</dd></div>
                    <div><dt>Next action</dt><dd>{displayNextAction(finding)}</dd></div>
                    <div><dt>Due Date</dt><dd><DueState dueDate={finding.dueDate} today={new Date().toISOString().slice(0, 10)} />{dueTiming(finding)}</dd></div>
                    <div><dt>Organization</dt><dd>{finding.organizationName}</dd></div>
                    <div><dt>Related audit</dt><dd>{finding.auditId}</dd></div>
                    <div><dt>Finding</dt><dd>{displayFindingNumber}</dd></div>
                  </dl>
                  <div className="finding-detail-copy">
                    <span>Description</span>
                    <p>{finding.description}</p>
                    <span>Finding basis / regulatory reference</span>
                    <p>{finding.regulatoryReference ?? "Configured regulatory reference"}</p>
                    <p>{finding.findingBasis}</p>
                    <Link className="lead-root-button" to={`/inspector/assistant?findingId=${encodeURIComponent(finding.id)}`}>Open Inspector Assistant</Link>
                  </div>
                </div>
              </section>

              <section className="finding-dossier-panel" id="finding-cap">
                <header><div><h2>Corrective Action Plan (CAP)</h2><small>CAP acceptance does not close the finding</small></div></header>
                <div className="finding-dossier-panel__body">
                  <p>{finding.status === "WAITING_FOR_CAP" ? "No CAP submitted yet. Waiting for the auditee." : "The latest immutable CAP revision is available for CAA review."}</p>
                  <div className="lead-review-handoff">
                    {finding.status === "CAP_SUBMITTED" ? (
                      <>
                        <p>Review CAP requires Lead Inspector authority.</p>
                        <RoleHandoff
                          identityMode={identityMode}
                          onRoleRequest={requestLeadCapReview}
                          session={handoffSession}
                          targetRole="leadInspector"
                        >
                          Switch to Lead Inspector for CAP Review
                        </RoleHandoff>
                      </>
                    ) : (
                      <RoleHandoff
                        identityMode={identityMode}
                        onRoleRequest={requestRole}
                        session={handoffSession}
                        targetRole="auditee"
                      >
                        Switch to {finding.organizationName} Auditee
                      </RoleHandoff>
                    )}
                  </div>
                </div>
              </section>

              <section className="finding-dossier-panel">
                <header><div><h2>Evidence</h2><small>Version history is preserved</small></div></header>
                <div className="finding-dossier-panel__body"><p>No Evidence version is available on this Finding yet.</p></div>
              </section>
            </div>

            <aside className="finding-dossier-stack">
              <section className="finding-dossier-panel">
                <header><h2>Comments to Auditee</h2></header>
                <div className="finding-dossier-panel__body"><p>No comments to Auditee yet.</p></div>
              </section>
              <section className="finding-dossier-panel">
                <header><h2>Reminder &amp; Escalation History</h2></header>
                <div className="finding-dossier-panel__body"><p>No reminder stage has been recorded for this Finding.</p></div>
              </section>
              <section className="finding-dossier-panel">
                <header><h2>Audit Trail</h2></header>
                <div className="finding-dossier-panel__body"><p>The canonical audit trail remains available through the backend record.</p></div>
              </section>
            </aside>
          </div>
        </div>
      ) : (
        <article className="lead-review-empty">
          <h2>Finding unavailable</h2>
          <p>The Finding is not available to this CAA Inspector backend projection.</p>
        </article>
      )}
    </WorkspaceShell>
  );
}
