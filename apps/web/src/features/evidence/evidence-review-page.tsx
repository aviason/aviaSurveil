import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type {
  CapRevisionView,
  EvidenceVersionView,
  FindingView,
  ReviewEvidenceOutput,
  Role,
} from "../../backend/backend";
import { RoleHandoff } from "../../auth/role-handoff";
import { useOptionalSession } from "../../auth/session-provider";
import { createRoleEntryPath } from "../../ui/role-select-page";
import { StatusPill, type StatusPillTone } from "../../ui/workbench/status-pill";
import {
  CommandError,
  errorMessage,
  formatLocalDate,
  formatSeverity,
  WorkspaceShell,
} from "../shared/workspace-shell";

type EvidenceDecision = "CLOSE" | "PARTIALLY_CLOSE" | "NOT_CLOSE" | "REQUEST_MORE_INFORMATION";

function statusTone(status: string): StatusPillTone {
  if (status === "CLOSED" || status === "ACCEPTED") return "success";
  if (status.includes("MORE_INFORMATION") || status === "REJECTED") return "warning";
  return "neutral";
}

function submittedDate(instant: string): string {
  return new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "short",
    timeZone: "UTC",
    year: "numeric",
  }).format(new Date(instant));
}

export function EvidenceReviewPage() {
  const runtime = useApplicationRuntime();
  const managerBackend = useMemo(
    () => runtime.backendForRole?.("manager") ?? runtime.backend,
    [runtime],
  );
  const session = useOptionalSession();
  const navigate = useNavigate();
  const [finding, setFinding] = useState<FindingView | null>(null);
  const [capRevisions, setCapRevisions] = useState<CapRevisionView[]>([]);
  const [evidenceVersions, setEvidenceVersions] = useState<EvidenceVersionView[]>([]);
  const [selectedEvidenceVersionId, setSelectedEvidenceVersionId] = useState<string | null>(null);
  const [evidenceReview, setEvidenceReview] = useState<ReviewEvidenceOutput | null>(null);
  const [decision, setDecision] = useState<EvidenceDecision>("PARTIALLY_CLOSE");
  const [commentToAuditee, setCommentToAuditee] = useState("");
  const [internalCaaNote, setInternalCaaNote] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const latest = useMemo(() => evidenceVersions.at(-1) ?? null, [evidenceVersions]);
  const selectedEvidence = useMemo(
    () => evidenceVersions.find((version) => version.id === selectedEvidenceVersionId) ?? null,
    [evidenceVersions, selectedEvidenceVersionId],
  );
  const identityMode =
    session?.identityMode ??
    (runtime.buildProfile === "http" ? "canonical-test-role-switch" : "demo-role-switch");
  const handoffSession = session?.state ?? { status: "unauthenticated" as const };

  async function loadReviewTarget(): Promise<void> {
    const loadedFinding = await managerBackend.findings.get({ findingId: "FND-CAB-2026-001" });
    const [versions, caps] = await Promise.all([
      managerBackend.evidence.listVersions({ findingId: loadedFinding.id }),
      managerBackend.caps.listRevisions({ findingId: loadedFinding.id }),
    ]);
    setFinding(loadedFinding);
    setEvidenceVersions(versions);
    setSelectedEvidenceVersionId(versions.at(-1)?.id ?? null);
    setCapRevisions(caps.items);
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
  }, [managerBackend]);

  function requestRole(role: Role): void {
    session?.setActiveRole(role);
    navigate(createRoleEntryPath(role));
  }

  async function recordReview(): Promise<void> {
    if (!finding || !selectedEvidence || reviewDisabledReason) return;
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
      const review = await managerBackend.evidence.review({
        operationId: `OP-EVIDENCE-${selectedEvidence.id}-R${selectedEvidence.revision}-${decision}`,
        evidenceVersionId: selectedEvidence.id,
        expectedEvidenceVersionRevision: selectedEvidence.revision,
        findingId: finding.id,
        expectedFindingRevision: finding.revision,
        decision,
        commentToAuditee,
        internalCaaNote,
      });
      const [loadedVersions, loadedFinding] = await Promise.all([
        managerBackend.evidence.listVersions({ findingId: finding.id }),
        managerBackend.findings.get({ findingId: finding.id }),
      ]);
      setEvidenceReview(review);
      setEvidenceVersions(loadedVersions);
      setSelectedEvidenceVersionId(review.evidenceVersionId);
      setFinding(loadedFinding);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  const closed = finding?.status === "CLOSED";
  const latestCap = capRevisions.at(-1) ?? null;

  const reviewDisabledReason = useMemo(() => {
    if (!selectedEvidence) return "No Evidence version is selected.";
    if (latest && selectedEvidence.id !== latest.id) {
      return `Evidence version ${selectedEvidence.id} is historical; only exact latest version ${latest.id} can be reviewed.`;
    }
    if (selectedEvidence.scanState !== "CLEAN") {
      return `Evidence version ${selectedEvidence.id} has scan state ${selectedEvidence.scanState}; only CLEAN Evidence can be reviewed.`;
    }
    if (selectedEvidence.reviewState !== "PENDING_CAA_REVIEW") {
      return `Evidence version ${selectedEvidence.id} is ${selectedEvidence.reviewState} and is no longer pending CAA review.`;
    }
    return null;
  }, [latest, selectedEvidence]);

  const pendingEvidenceCount = evidenceVersions.filter((version) => version.scanState === "CLEAN" && version.reviewState === "PENDING_CAA_REVIEW").length;
  const pendingCapCount = capRevisions.filter((cap) => cap.status === "SUBMITTED" || cap.status === "PENDING_CAA_REVIEW").length;
  const waitingFindingCount = latest && latest.scanState === "CLEAN" && latest.reviewState === "PENDING_CAA_REVIEW" ? 1 : 0;
  const ownerLabel = finding ? `${finding.currentOwnerRole === "leadInspector" ? "Lead Inspector" : finding.currentOwnerRole === "inspector" ? "CAA Inspector" : finding.currentOwnerRole === "manager" ? "Department Manager" : finding.currentOwnerRole === "auditee" ? "Auditee" : "Owner"} · ${finding.currentOwnerId}` : "Owner unavailable";

  function openReviewPanel(evidenceVersionId: string): void {
    setSelectedEvidenceVersionId(evidenceVersionId);
    const target = document.getElementById("evidence-review-target");
    if (typeof target?.scrollIntoView === "function") target.scrollIntoView({ block: "start" });
  }

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel="Inspection Evidence">
      {finding ? (
        <div className="evidence-root-page" data-testid="manager-inspection-evidence-page">
          <CommandError message={error} />
          <span className="finding-semantic-context">Department Manager · {finding.id} · {finding.organizationName} · Review evidence · Evidence · Due</span>
          <header className="evidence-root-head workbench-page-header">
            <div>
              <h1>Findings</h1>
              <p>Follow findings, CAP responses, evidence, owner, due date, and next action in one workspace.</p>
            </div>
            <div aria-label="Finding filters" className="evidence-root-filters">
              <button disabled title="The all-findings register remains in the accepted legacy demo." type="button"><span>All Findings</span></button>
              <button disabled title="The open-findings register remains in the accepted legacy demo." type="button"><span>Open Findings</span></button>
              <button disabled title="The overdue-findings register remains in the accepted legacy demo." type="button"><span>Overdue Findings</span></button>
              <button aria-label="Open CAP / Provider Review" onClick={() => navigate("/department-manager/findings-review")} type="button"><span>CAP / Provider Review</span></button>
              <button aria-current="page" disabled title="Evidence Waiting Review is the current filter." type="button"><span>Evidence Waiting Review</span></button>
              <button disabled title="The Due Soon findings register remains in the accepted legacy demo." type="button"><span>Findings Due Soon</span></button>
              <button disabled title="The critical-findings register remains in the accepted legacy demo." type="button"><span>Critical Findings</span></button>
              <button disabled title="The closed-findings register remains in the accepted legacy demo." type="button"><span>Closed Findings</span></button>
            </div>
          </header>

          <div className="evidence-root-attention">
            <div className="is-info"><span>Evidence Waiting Review</span><b>{waitingFindingCount} {waitingFindingCount === 1 ? "finding" : "findings"}</b></div>
            <div className="is-warn"><span>CAP review rows</span><b>{pendingCapCount}</b></div>
            <div className="is-warn"><span>Evidence review rows</span><b>{pendingEvidenceCount}</b></div>
          </div>

          <div className="evidence-root-table-wrap">
            <table className="evidence-root-table">
              <thead><tr><th>Priority</th><th>Item</th><th>Organization</th><th>Owner</th><th>Next Action</th><th>Due Date / Target</th><th>Status</th><th>Open</th></tr></thead>
              <tbody>
                <tr className="is-parent is-attention">
                  <td data-col="priority"><span className="evidence-root-priority is-warn">Waiting review</span></td>
                  <td data-col="item"><b>{finding.title}</b><small>{finding.findingNumber} · {formatSeverity(finding.severity)}</small></td>
                  <td data-col="org">{finding.organizationName}</td>
                  <td data-col="owner" data-label="Owner:">{ownerLabel}</td>
                  <td data-col="next" data-label="Next:"><b>{finding.nextAction}</b></td>
                  <td data-col="due">Due Date {formatLocalDate(finding.dueDate)}</td>
                  <td data-col="status"><span className="evidence-root-status">● {finding.status}</span></td>
                  <td data-col="actions"><button onClick={() => navigate(`/department-manager/findings-review?findingId=${finding.id}`)} type="button">Open Findings Review</button></td>
                </tr>
                {latestCap ? (
                  <tr className="is-child">
                    <td data-col="priority"><span className="evidence-root-priority is-info">CAP state</span></td>
                    <td data-col="item"><b><span>Subitem</span> Corrective Action Plan</b><small>{latestCap.correctiveAction}</small></td>
                    <td data-col="org">{finding.organizationName}</td>
                    <td data-col="owner" data-label="Owner:">{ownerLabel}</td>
                    <td data-col="next" data-label="Next:"><b>CAP accepted; evidence still required before closure</b></td>
                    <td data-col="due">Target {formatLocalDate(latestCap.targetCompletionDate)}</td>
                    <td data-col="status"><span className="evidence-root-status">● {latestCap.status}</span></td>
                    <td data-col="actions"><span>See Finding row</span></td>
                  </tr>
                ) : null}
                {evidenceVersions.map((version) => (
                  <tr className={`is-child${version.id === latest?.id ? " is-attention" : ""}`} key={version.id}>
                    <td data-col="priority"><span className={`evidence-root-priority ${version.reviewState === "PENDING_CAA_REVIEW" ? "is-warn" : "is-info"}`}>{version.reviewState === "PENDING_CAA_REVIEW" ? "Waiting review" : "Reviewed"}</span></td>
                    <td data-col="item"><b><span>Subitem</span> Evidence v{version.version}</b><small>{version.fileName}</small></td>
                    <td data-col="org">{finding.organizationName}</td>
                    <td data-col="owner" data-label="Owner:">{ownerLabel}</td>
                    <td data-col="next" data-label="Next:"><b>{version.reviewState === "PENDING_CAA_REVIEW" ? finding.nextAction : "Review recorded for this immutable version"}</b></td>
                    <td data-col="due">Uploaded {submittedDate(version.submittedAt)}</td>
                    <td data-col="status"><span className="evidence-root-status">● {version.reviewState === "PENDING_CAA_REVIEW" ? "Uploaded" : version.reviewState}</span></td>
                    <td data-col="actions"><button aria-label={`${version.reviewState === "PENDING_CAA_REVIEW" && version.id === latest?.id ? "Review" : "View"} Evidence version ${version.id}`} className={version.id === latest?.id ? "is-primary" : ""} onClick={() => openReviewPanel(version.id)} type="button">{version.reviewState === "PENDING_CAA_REVIEW" && version.id === latest?.id ? "Review evidence" : "View version"}</button></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <article className="lead-review-dossier evidence-root-review" data-testid="evidence-review-target" id="evidence-review-target">
            <div className="lead-review-dossier__head">
              <div>
                <p className="eyebrow">Selected immutable Evidence version</p>
                <h2 data-testid="reviewing-evidence-version">
                  {selectedEvidence ? `Version ${selectedEvidence.version}` : "No version"}
                </h2>
              </div>
              {selectedEvidence ? (
                <StatusPill label={selectedEvidence.reviewState} tone={statusTone(selectedEvidence.reviewState)} />
              ) : null}
            </div>
            {selectedEvidence ? <p>{selectedEvidence.id} · {selectedEvidence.fileName}</p> : <p>No Evidence version is selected.</p>}
            <ol aria-label="Evidence version history" className="lead-review-version-list">
              {evidenceVersions.map((version) => (
                <li data-testid="evidence-history-row" key={version.id}>
                  <strong>Version {version.version}</strong>
                  <span>{version.fileName}</span>
                  <span>{version.scanState}</span>
                  <span>{version.reviewState}</span>
                </li>
              ))}
            </ol>
            <div className="lead-review-decision-grid">
              <label className="lead-review-field-span">
                Evidence review decision
                <select
                  value={decision}
                  onChange={(event) => setDecision(event.target.value as EvidenceDecision)}
                >
                  <option value="PARTIALLY_CLOSE">Partially Close</option>
                  <option value="NOT_CLOSE">Not Close</option>
                  <option value="REQUEST_MORE_INFORMATION">Request More Information</option>
                  <option value="CLOSE">Close</option>
                </select>
              </label>
              <label>
                Comment to Auditee
                <textarea
                  rows={4}
                  value={commentToAuditee}
                  onChange={(event) => setCommentToAuditee(event.target.value)}
                />
              </label>
              <label>
                Internal CAA Note
                <textarea
                  rows={4}
                  value={internalCaaNote}
                  onChange={(event) => setInternalCaaNote(event.target.value)}
                />
              </label>
              <button
                className="primary-button lead-review-field-span"
                disabled={busy || Boolean(reviewDisabledReason)}
                onClick={() => void recordReview()}
                title={reviewDisabledReason ?? undefined}
                type="button"
              >
                Record Evidence review
              </button>
            </div>
          </article>
          {evidenceReview ? (
            <section className="lead-review-result">
              <strong data-testid="finding-status">{finding.status}</strong>
              <span data-testid="closure-state">{closed ? "Finding closed" : "Finding remains open"}</span>
              {closed ? <span data-testid="closure-basis">{finding.closureBasis}</span> : null}
              {closed ? (
                <RoleHandoff
                  identityMode={identityMode}
                  onRoleRequest={requestRole}
                  session={handoffSession}
                  targetRole="manager"
                >
                  Open updated Manager Dashboard
                </RoleHandoff>
              ) : (
                <RoleHandoff
                  identityMode={identityMode}
                  onRoleRequest={requestRole}
                  session={handoffSession}
                  targetRole="auditee"
                >
                  Return to Auditee Evidence
                </RoleHandoff>
              )}
              <Link to={`/department-manager/findings/${finding.id}/closure-review`}>
                Open Department CAP Closure Review for {finding.id}
              </Link>
            </section>
          ) : null}
        </div>
      ) : (
        <article className="lead-review-empty">
          <h2>Finding unavailable</h2>
          <p>The Evidence review route could not load this Finding for the Department Manager.</p>
        </article>
      )}
    </WorkspaceShell>
  );
}
