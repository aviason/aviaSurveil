import { useEffect, useMemo, useState } from "react";

import { useApplicationRuntime } from "../../app/providers";
import type { AuditeeCoordinationView } from "../../backend/backend";
import { CommandError, errorMessage, formatLocalDate, WorkspaceShell } from "../shared/workspace-shell";

export function AuditeeInspectionCoordinationPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("auditee") ?? runtime.backend, [runtime]);
  const [items, setItems] = useState<AuditeeCoordinationView[]>([]);
  const [alternativeDate, setAlternativeDate] = useState("");
  const [statusFilter, setStatusFilter] = useState("ALL");
  const [status, setStatus] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!backend.auditeeCoordination) return;
    let cancelled = false;
    void backend.auditeeCoordination.list({}).then((output) => {
      if (!cancelled) setItems(output.items);
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend]);

  const visible = items.filter((item) => statusFilter === "ALL" || item.status === statusFilter);

  async function respond(item: AuditeeCoordinationView, decision: "CONFIRM" | "PROPOSE_ALTERNATIVE") {
    if (!backend.auditeeCoordination) return;
    try {
      const updated = await backend.auditeeCoordination.respond({
        auditId: item.auditId,
        organizationId: item.organizationId,
        expectedRevision: item.revision,
        idempotencyKey: `COORD-${item.auditId}-${decision}-${item.revision}`,
        decision,
        alternativeDate: decision === "PROPOSE_ALTERNATIVE" ? alternativeDate : null,
      });
      setItems((current) => current.map((candidate) => candidate.auditId === updated.auditId ? updated : candidate));
      setStatus(decision === "CONFIRM"
        ? `${updated.auditId} proposed date confirmed.`
        : `${updated.auditId} alternative date ${updated.alternativeDate} submitted to CAA; acceptance remains pending.`);
      setAlternativeDate("");
      setError(null);
    } catch (cause) { setError(errorMessage(cause)); }
  }

  return <WorkspaceShell roleLabel="Auditee" routeLabel="Inspection Coordination">
    <div className="auditee-secondary-page auditee-coordination-page" data-testid="auditee-inspection-coordination-page">
      <header className="auditee-secondary-head workbench-page-header">
        <div><span>Organization-scoped coordination</span><h1>Inspection Coordination</h1><p>The CAA expects the authorized Auditee organization to confirm the proposed inspection date or propose an alternative date next.</p></div>
      </header>
      <p className="auditee-safe-boundary">Only CAA-released Routine / Announced inspections for the authorized Auditee organization appear here.</p>
      <CommandError message={error} />
      {status ? <p className="auditee-action-result" role="status">{status}</p> : null}
      <section className="auditee-secondary-filters" aria-label="Inspection coordination filters">
        <label>Status<select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}><option value="ALL">All visible coordination</option><option value="AWAITING_AUDITEE_CONFIRMATION">Awaiting response</option><option value="CONFIRMED">Confirmed</option><option value="ALTERNATIVE_PROPOSED">Alternative proposed</option></select></label>
      </section>
      <section className="auditee-card-register" aria-label="Released inspection coordination">
        {visible.map((item) => <article key={item.auditId} data-audit-id={item.auditId} data-revision={item.revision}>
          <header><div><span>{item.inspectionCategory}</span><h2>{item.title}</h2><p>{item.auditId} · {item.organizationName}</p></div><b>{item.status}</b></header>
          <dl><div><dt>Proposed date</dt><dd>{formatLocalDate(item.scheduledStartDate)}</dd></div>{item.alternativeDate ? <div><dt>Alternative date</dt><dd>{formatLocalDate(item.alternativeDate)}</dd></div> : null}<div><dt>Exact organization</dt><dd>{item.organizationId}</dd></div><div><dt>Next action</dt><dd>{item.nextAction}</dd></div><div><dt>Revision</dt><dd>{item.revision}</dd></div></dl>
          {item.status === "AWAITING_AUDITEE_CONFIRMATION" ? <div className="auditee-coordination-actions">
            <button onClick={() => void respond(item, "CONFIRM")} type="button">Confirm Proposed Date</button>
            <label>Alternative date<input min={item.scheduledStartDate} type="date" value={alternativeDate} onChange={(event) => setAlternativeDate(event.target.value)} /></label>
            <button disabled={!alternativeDate} onClick={() => void respond(item, "PROPOSE_ALTERNATIVE")} title={!alternativeDate ? `Select an alternative date for ${item.auditId} before submitting.` : undefined} type="button">Propose Alternative Date</button>
          </div> : <p>{item.nextAction}</p>}
        </article>)}
        {!visible.length ? <p>No released inspections match this filter.</p> : null}
      </section>
    </div>
  </WorkspaceShell>;
}
