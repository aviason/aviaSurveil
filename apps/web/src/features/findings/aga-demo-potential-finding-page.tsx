import { useEffect, useState } from "react";

import {
  LifecycleAction,
  LifecyclePageFrame,
  LifecycleUnavailable,
  lifecycleDisabledReason,
  useLifecycleWorkspace,
  type AGADemoLifecyclePageProps,
} from "../inspections/aga-demo-inspection-page";

export function AGADemoPotentialFindingPage({
  capability,
  role,
  roleLabel,
  initialProjection,
  inspectionId,
  onProjectionChange,
}: AGADemoLifecyclePageProps) {
  const workspace = useLifecycleWorkspace({ capability, role, initialProjection, inspectionId, onProjectionChange });
  const projection = workspace.projection;
  const [selectedPotentialId, setSelectedPotentialId] = useState("");
  const [reasonCode, setReasonCode] = useState("LEAD_REVIEW_DECISION");
  const [severity, setSeverity] = useState("MAJOR");
  const [capRequired, setCapRequired] = useState(true);
  const [evidenceRequired, setEvidenceRequired] = useState(true);
  const [dueDateRequired, setDueDateRequired] = useState(false);
  const [dueDate, setDueDate] = useState("");

  useEffect(() => {
    if (projection && "publicOwnerLabel" in projection) {
      setSelectedPotentialId("");
      return;
    }
    const pending = projection?.potentialFindings.find((potential) => potential.state === "PENDING_LEAD_REVIEW");
    setSelectedPotentialId(pending?.potentialFindingId ?? "");
  }, [projection]);

  if (projection && "publicOwnerLabel" in projection) {
    return (
      <LifecyclePageFrame
        description="Potential Findings are a CAA-only review artifact and are not exposed through the Auditee projection."
        error={workspace.error}
        eyebrow="Synthetic finding lifecycle"
        roleLabel={roleLabel}
        status={workspace.status}
        testId="aga-demo-potential-finding-page"
        title="Potential Finding review"
      >
        <section aria-label="Auditee finding boundary" className="aga-lifecycle-boundary">
          <p>Use the organization-scoped CAP and Evidence workflow for released Finding follow-up.</p>
        </section>
      </LifecyclePageFrame>
    );
  }

  if (!projection) {
    return (
      <LifecyclePageFrame
        description="Lead review is the required human boundary between a Potential Finding and a Finding."
        error={workspace.error}
        eyebrow="Synthetic finding lifecycle"
        roleLabel={roleLabel}
        status={workspace.status}
        testId="aga-demo-potential-finding-page"
        title="Potential Finding review"
      >
        <LifecycleUnavailable capability={capability} client={workspace.client} projection={projection} loading={workspace.loading} />
        <section aria-label="Lead Potential Finding actions" className="aga-lifecycle-actions">
          <LifecycleAction
            actionId="return-potential-finding"
            disabled
            label="Return for correction"
            reason={lifecycleDisabledReason(capability, workspace.client, projection, role === "leadInspector", "A server-returned Potential Finding projection is required before a Lead decision.") ?? "A server-returned Potential Finding projection is required before a Lead decision."}
          />
          <LifecycleAction
            actionId="dismiss-potential-finding"
            disabled
            label="Dismiss Potential Finding"
            reason={lifecycleDisabledReason(capability, workspace.client, projection, role === "leadInspector", "A server-returned Potential Finding projection is required before a Lead decision.") ?? "A server-returned Potential Finding projection is required before a Lead decision."}
          />
          <LifecycleAction
            actionId="convert-potential-finding"
            disabled
            label="Convert to Finding"
            reason={lifecycleDisabledReason(capability, workspace.client, projection, role === "leadInspector", "A server-returned Potential Finding projection is required before conversion.") ?? "A server-returned Potential Finding projection is required before conversion."}
          />
        </section>
      </LifecyclePageFrame>
    );
  }

  const selected = projection.potentialFindings.find((potential) => potential.potentialFindingId === selectedPotentialId);
  const leadAllowed = role === "leadInspector";
  const stateAllowed = projection.state === "SUBMITTED";
  const selectedReason = selected ? "" : "Choose a server-returned Potential Finding before issuing a Lead decision.";
  const commonReason = lifecycleDisabledReason(capability, workspace.client, projection, leadAllowed, stateAllowed ? selectedReason : "Lead review is available after the Inspector submits the checklist.") ?? "";
  const convertReason = commonReason
    ? commonReason
    : !reasonCode.trim()
      ? "A non-empty reason is required for conversion."
      : dueDateRequired && !dueDate
        ? "A due date is required when due-date tracking is selected."
        : "";
  const dueDateValue = dueDate ? `${dueDate}T00:00:00Z` : undefined;

  return (
    <LifecyclePageFrame
      description="Lead Inspector owns the return, dismiss, or conversion decision. Conversion creates the Finding with explicit CAP, Evidence, severity, and due-date choices."
      error={workspace.error}
      eyebrow="Synthetic finding lifecycle"
      roleLabel={roleLabel}
      status={workspace.status}
      testId="aga-demo-potential-finding-page"
      title="Potential Finding review"
    >
      <section aria-label="Potential Finding queue" className="aga-lifecycle-facts">
        <article><span>Inspection state</span><strong>{projection.state}</strong><small>{projection.nextAction}</small></article>
        <article><span>Potential Finding versions</span><strong>{projection.potentialFindings.length}</strong><small>Returned versions remain immutable</small></article>
        <article><span>Converted Findings</span><strong>{projection.findings.length}</strong><small>Lead conversion required</small></article>
      </section>
      <section aria-label="Lead Potential Finding actions" className="aga-lifecycle-panel">
        <h2>Lead decision</h2>
        <label>Server-returned Potential Finding<select aria-label="Potential Finding" value={selectedPotentialId} onChange={(event) => setSelectedPotentialId(event.target.value)}><option value="">Select a Potential Finding</option>{projection.potentialFindings.filter((potential) => potential.state === "PENDING_LEAD_REVIEW").map((potential) => <option key={potential.potentialFindingId} value={potential.potentialFindingId}>{potential.potentialFindingId} · {potential.answer}</option>)}</select></label>
        <label>Decision reason<input aria-label="Lead decision reason" value={reasonCode} onChange={(event) => setReasonCode(event.target.value)} /></label>
        <div className="aga-lifecycle-actions">
          <LifecycleAction actionId="return-potential-finding" disabled={Boolean(commonReason) || workspace.pending || !reasonCode.trim()} label="Return for correction" onClick={() => void workspace.runCommand("RETURN_POTENTIAL_FINDING", { potentialFindingId: selectedPotentialId, reasonCode }).catch(() => undefined)} reason={commonReason || "A non-empty reason is required."} />
          <LifecycleAction actionId="dismiss-potential-finding" disabled={Boolean(commonReason) || workspace.pending || !reasonCode.trim()} label="Dismiss Potential Finding" onClick={() => void workspace.runCommand("DISMISS_POTENTIAL_FINDING", { potentialFindingId: selectedPotentialId, reasonCode }).catch(() => undefined)} reason={commonReason || "A non-empty reason is required."} />
        </div>
        <fieldset>
          <legend>Finding conversion choices</legend>
          <label>Severity<select aria-label="Finding severity" value={severity} onChange={(event) => setSeverity(event.target.value)}><option value="CRITICAL">CRITICAL</option><option value="MAJOR">MAJOR</option><option value="MINOR">MINOR</option><option value="OBSERVATION">OBSERVATION</option></select></label>
          <label><input aria-label="CAP required" checked={capRequired} onChange={(event) => setCapRequired(event.target.checked)} type="checkbox" /> CAP required</label>
          <label><input aria-label="Evidence required" checked={evidenceRequired} onChange={(event) => setEvidenceRequired(event.target.checked)} type="checkbox" /> Evidence required</label>
          <label><input aria-label="Due date required" checked={dueDateRequired} onChange={(event) => setDueDateRequired(event.target.checked)} type="checkbox" /> Due Date required</label>
          <label>Due Date<input aria-label="Finding due date" disabled={!dueDateRequired} required={dueDateRequired} type="date" value={dueDate} onChange={(event) => setDueDate(event.target.value)} /></label>
        </fieldset>
        <LifecycleAction
          actionId="convert-potential-finding"
          disabled={Boolean(convertReason) || workspace.pending}
          label="Convert to Finding"
          onClick={() => void workspace.runCommand("CONVERT_POTENTIAL_FINDING", { potentialFindingId: selectedPotentialId, reasonCode, severity, capRequired, evidenceRequired, dueDateRequired, dueDate: dueDateValue }).catch(() => undefined)}
          reason={convertReason || "The command is pending."}
        />
        {selected ? <p className="aga-lifecycle-boundary">Selected {selected.potentialFindingId} is version {selected.version}; the server owns its response and digest pins.</p> : null}
      </section>
      <section aria-label="Potential Finding history" className="aga-lifecycle-register">
        <h2>Append-only Potential Finding history</h2>
        <ul>{projection.potentialFindings.map((potential) => <li key={potential.potentialFindingId}><strong>{potential.state}</strong><span>{potential.potentialFindingId} · version {potential.version}</span><small>{potential.commentToAuditee}</small></li>)}</ul>
      </section>
    </LifecyclePageFrame>
  );
}
