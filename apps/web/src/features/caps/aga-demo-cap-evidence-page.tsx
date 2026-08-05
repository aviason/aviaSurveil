import { useEffect, useMemo, useState } from "react";

import {
  LifecycleAction,
  LifecyclePageFrame,
  LifecycleUnavailable,
  lifecycleDisabledReason,
  useLifecycleWorkspace,
  type AGADemoLifecyclePageProps,
} from "../inspections/aga-demo-inspection-page";

function latestFinding(projection: NonNullable<ReturnType<typeof useLifecycleWorkspace>["projection"]>, findingId: string) {
  return projection.findings.filter((finding) => finding.findingId === findingId).sort((left, right) => right.revision - left.revision)[0];
}

function latestCAP(projection: NonNullable<ReturnType<typeof useLifecycleWorkspace>["projection"]>, findingId: string) {
  return projection.capRevisions.filter((cap) => cap.findingId === findingId).sort((left, right) => right.revision - left.revision)[0];
}

function latestEvidence(projection: NonNullable<ReturnType<typeof useLifecycleWorkspace>["projection"]>, findingId: string) {
  return projection.evidenceVersions.filter((evidence) => evidence.findingId === findingId).sort((left, right) => right.version - left.version)[0];
}

export function AGADemoCAPEvidencePage({
  capability,
  role,
  roleLabel,
  initialProjection,
  inspectionId,
  onProjectionChange,
}: AGADemoLifecyclePageProps) {
  const workspace = useLifecycleWorkspace({ capability, role, initialProjection, inspectionId, onProjectionChange });
  const projection = workspace.projection;
  const [selectedFindingId, setSelectedFindingId] = useState("");
  const [rootCause, setRootCause] = useState("");
  const [correctiveAction, setCorrectiveAction] = useState("");
  const [preventiveAction, setPreventiveAction] = useState("");
  const [responsiblePerson, setResponsiblePerson] = useState("");
  const [evidenceFileName, setEvidenceFileName] = useState("");
  const [commentToAuditee, setCommentToAuditee] = useState("");
  const [internalCaaNote, setInternalCaaNote] = useState("");
  const [capOutcome, setCapOutcome] = useState<"ACCEPT" | "REJECT" | "MORE_INFORMATION_REQUESTED">("ACCEPT");
  const [evidenceOutcome, setEvidenceOutcome] = useState<"CLOSE" | "PARTIALLY_CLOSE" | "NOT_CLOSE" | "REQUEST_MORE_INFORMATION">("CLOSE");

  const findingIds = useMemo(() => [...new Set(projection?.findings.map((finding) => finding.findingId) ?? [])], [projection]);
  useEffect(() => {
    if (!selectedFindingId && findingIds[0]) setSelectedFindingId(findingIds[0]);
    if (selectedFindingId && !findingIds.includes(selectedFindingId)) setSelectedFindingId(findingIds[0] ?? "");
  }, [findingIds, selectedFindingId]);

  if (!projection) {
    return (
      <LifecyclePageFrame
        description="CAP submission, CAA review, Evidence metadata, verification, and authorized closure remain separate synthetic artifacts."
        error={workspace.error}
        eyebrow="Synthetic CAP and Evidence lifecycle"
        roleLabel={roleLabel}
        status={workspace.status}
        testId="aga-demo-cap-evidence-page"
        title="CAP, Evidence, and closure"
      >
        <LifecycleUnavailable capability={capability} client={workspace.client} projection={projection} loading={workspace.loading} />
      </LifecyclePageFrame>
    );
  }

  const finding = latestFinding(projection, selectedFindingId);
  const cap = finding ? latestCAP(projection, finding.findingId) : undefined;
  const evidence = finding ? latestEvidence(projection, finding.findingId) : undefined;
  const auditeeAllowed = role === "auditee";
  const reviewerAllowed = role === "leadInspector" || role === "manager";
  const verifierAllowed = role === "inspector" || role === "leadInspector";
  const managerAllowed = role === "manager";
  const findingReason = !finding ? "Choose a server-returned Finding before issuing a lifecycle command." : "";
  const capFormComplete = rootCause.trim() && correctiveAction.trim() && preventiveAction.trim() && responsiblePerson.trim() && commentToAuditee.trim();
  const evidenceFormComplete = evidenceFileName.trim() && commentToAuditee.trim();
  const capSubmitReason = lifecycleDisabledReason(capability, workspace.client, projection, auditeeAllowed, finding?.capRequired && (finding.state === "WAITING_FOR_CAP" || finding.state === "CAP_REJECTED" || finding.state === "CAP_MORE_INFORMATION_REQUESTED") ? (capFormComplete ? "" : "CAP fields and Comment to Auditee are required.") : "This Finding is not awaiting a CAP revision.") ?? "";
  const capReviewReason = lifecycleDisabledReason(capability, workspace.client, projection, reviewerAllowed, cap?.state === "PENDING_CAA_REVIEW" ? (commentToAuditee.trim() && internalCaaNote.trim() ? "" : "CAP review requires separate Comment to Auditee and Internal CAA Note fields.") : "The current CAP is not pending CAA review.") ?? "";
  const evidenceSubmitReason = lifecycleDisabledReason(capability, workspace.client, projection, auditeeAllowed, finding?.evidenceRequired && (finding.state === "EVIDENCE_REQUIRED" || finding.state === "EVIDENCE_MORE_INFORMATION_REQUESTED") ? (evidenceFormComplete ? "" : "Evidence filename and Comment to Auditee are required.") : "This Finding is not awaiting Evidence.") ?? "";
  const verifyReason = lifecycleDisabledReason(capability, workspace.client, projection, verifierAllowed, evidence?.reviewState === "PENDING_CAA_REVIEW" ? (commentToAuditee.trim() && internalCaaNote.trim() ? "" : "Evidence verification requires separate Comment to Auditee and Internal CAA Note fields.") : "The current Evidence version is not pending verification.") ?? "";
  const closeReason = lifecycleDisabledReason(capability, workspace.client, projection, managerAllowed, finding?.state === "PENDING_CLOSURE" ? "" : "Authorized closure is available only from PENDING_CLOSURE after accepted Evidence or the explicit no-Evidence branch.") ?? "";

  return (
    <LifecyclePageFrame
      description="Matching Service Provider submits immutable CAP/Evidence metadata; the CAA reviewer, verifier, and scoped Manager retain separate authorities."
      error={workspace.error}
      eyebrow="Synthetic CAP and Evidence lifecycle"
      roleLabel={roleLabel}
      status={workspace.status}
      testId="aga-demo-cap-evidence-page"
      title="CAP, Evidence, and closure"
    >
      <section aria-label="Finding selection" className="aga-lifecycle-facts">
        <article><span>Findings</span><strong>{findingIds.length}</strong><small>Conversion is the Lead gate</small></article>
        <article><span>Selected Finding</span><strong>{finding?.findingId ?? "Not selected"}</strong><small>{finding?.state ?? "No server-returned Finding"}</small></article>
        <article><span>CAP</span><strong>{cap?.state ?? "Not submitted"}</strong><small>{cap ? `revision ${cap.revision}` : "Append-only revisions"}</small></article>
        <article><span>Evidence</span><strong>{evidence?.reviewState ?? "Not submitted"}</strong><small>{evidence ? `version ${evidence.version}` : "Metadata only"}</small></article>
      </section>
      <section aria-label="Finding lifecycle selector" className="aga-lifecycle-panel">
        <label>Server-returned Finding<select aria-label="Finding" value={selectedFindingId} onChange={(event) => setSelectedFindingId(event.target.value)}><option value="">Select a Finding</option>{findingIds.map((findingId) => <option key={findingId} value={findingId}>{findingId}</option>)}</select></label>
        {finding ? <p className="aga-lifecycle-boundary">CAP required: {String(finding.capRequired)} · Evidence required: {String(finding.evidenceRequired)} · Due Date required: {String(finding.dueDateRequired)} · next action: {finding.nextAction}</p> : <p>Selecting a Finding is required before any command can be enabled.</p>}
      </section>
      <section aria-label="CAP submission" className="aga-lifecycle-panel">
        <h2>Service Provider CAP submission</h2>
        <p>CAP submission creates a new immutable revision and never closes the Finding.</p>
        <label>Root cause<textarea aria-label="CAP root cause" value={rootCause} onChange={(event) => setRootCause(event.target.value)} /></label>
        <label>Corrective action<textarea aria-label="CAP corrective action" value={correctiveAction} onChange={(event) => setCorrectiveAction(event.target.value)} /></label>
        <label>Preventive action<textarea aria-label="CAP preventive action" value={preventiveAction} onChange={(event) => setPreventiveAction(event.target.value)} /></label>
        <label>Responsible person<input aria-label="CAP responsible person" value={responsiblePerson} onChange={(event) => setResponsiblePerson(event.target.value)} /></label>
        <LifecycleAction actionId="submit-cap" disabled={Boolean(capSubmitReason) || workspace.pending} label="Submit CAP revision" onClick={() => void workspace.runCommand("SUBMIT_CAP_REVISION", { findingId: selectedFindingId, rootCause, correctiveAction, preventiveAction, responsiblePerson, commentToAuditee }).catch(() => undefined)} reason={capSubmitReason || "The command is pending."} />
      </section>
      <section aria-label="CAA CAP review" className="aga-lifecycle-panel">
        <h2>CAA CAP review</h2>
        <label>CAP outcome<select aria-label="CAP outcome" value={capOutcome} onChange={(event) => setCapOutcome(event.target.value as typeof capOutcome)}><option value="ACCEPT">ACCEPT</option><option value="REJECT">REJECT</option><option value="MORE_INFORMATION_REQUESTED">MORE_INFORMATION_REQUESTED</option></select></label>
        <LifecycleAction actionId="review-cap" disabled={Boolean(capReviewReason) || workspace.pending} label="Review CAP" onClick={() => void workspace.runCommand("REVIEW_CAP", { findingId: selectedFindingId, outcome: capOutcome, commentToAuditee, internalCaaNote }).catch(() => undefined)} reason={capReviewReason || "The command is pending."} />
      </section>
      <section aria-label="Evidence submission" className="aga-lifecycle-panel">
        <h2>Service Provider Evidence metadata</h2>
        <p>No file bytes are uploaded by this candidate page; the server stores an append-only metadata version.</p>
        <label>Evidence filename<input aria-label="Evidence filename" value={evidenceFileName} onChange={(event) => setEvidenceFileName(event.target.value)} /></label>
        <LifecycleAction actionId="submit-evidence" disabled={Boolean(evidenceSubmitReason) || workspace.pending} label="Submit Evidence version" onClick={() => void workspace.runCommand("SUBMIT_EVIDENCE_VERSION", { findingId: selectedFindingId, evidenceFileName, commentToAuditee }).catch(() => undefined)} reason={evidenceSubmitReason || "The command is pending."} />
      </section>
      <section aria-label="Evidence verification" className="aga-lifecycle-panel">
        <h2>CAA Evidence verification</h2>
        <label>Evidence outcome<select aria-label="Evidence outcome" value={evidenceOutcome} onChange={(event) => setEvidenceOutcome(event.target.value as typeof evidenceOutcome)}><option value="CLOSE">CLOSE</option><option value="PARTIALLY_CLOSE">PARTIALLY_CLOSE</option><option value="NOT_CLOSE">NOT_CLOSE</option><option value="REQUEST_MORE_INFORMATION">REQUEST_MORE_INFORMATION</option></select></label>
        <label>Comment to Auditee<textarea aria-label="Lifecycle Comment to Auditee" value={commentToAuditee} onChange={(event) => setCommentToAuditee(event.target.value)} /></label>
        {(reviewerAllowed || verifierAllowed) ? <label>Internal CAA Note<textarea aria-label="Internal CAA Note" value={internalCaaNote} onChange={(event) => setInternalCaaNote(event.target.value)} /></label> : <p id="auditee-internal-note-reason">Internal CAA notes are not exposed to the Auditee projection.</p>}
        <LifecycleAction actionId="verify-evidence" disabled={Boolean(verifyReason) || workspace.pending} label="Verify Evidence" onClick={() => void workspace.runCommand("VERIFY_EVIDENCE", { findingId: selectedFindingId, outcome: evidenceOutcome, commentToAuditee, internalCaaNote }).catch(() => undefined)} reason={verifyReason || "The command is pending."} />
      </section>
      <section aria-label="Authorized closure" className="aga-lifecycle-panel">
        <h2>Separate authorized closure</h2>
        <p>CAP acceptance and Evidence verification do not silently create this command. A scoped Department Manager must authorize closure from PENDING_CLOSURE.</p>
        <LifecycleAction actionId="authorized-close" disabled={Boolean(closeReason) || workspace.pending} label="Authorize Finding closure" onClick={() => void workspace.runCommand("AUTHORIZED_CLOSE", { findingId: selectedFindingId, reasonCode: "MANAGER_AUTHORIZED_CLOSURE" }).catch(() => undefined)} reason={closeReason || "The command is pending."} />
      </section>
      <section aria-label="CAP and Evidence history" className="aga-lifecycle-register">
        <h2>Append-only CAP and Evidence history</h2>
        <ul>{projection.capRevisions.map((revision) => <li key={revision.capId}><strong>CAP {revision.state}</strong><span>{revision.capId} · {revision.findingId} · revision {revision.revision}</span></li>)}{projection.evidenceVersions.map((version) => <li key={version.evidenceId}><strong>Evidence {version.reviewState}</strong><span>{version.evidenceId} · {version.findingId} · version {version.version}</span></li>)}</ul>
      </section>
    </LifecyclePageFrame>
  );
}
