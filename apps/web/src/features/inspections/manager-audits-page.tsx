import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { AssignmentSummary } from "../../backend/backend";
import { CommandError, errorMessage, formatLocalDate, PageHeader, WorkspaceShell } from "../shared/workspace-shell";

export function ManagerAuditsPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("manager") ?? runtime.backend, [runtime]);
  const [audits, setAudits] = useState<AssignmentSummary[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [status, setStatus] = useState("all");
  const [error, setError] = useState<string | null>(null);
  const [searchParams] = useSearchParams();
  const requestedAuditId = searchParams.get("auditId");

  useEffect(() => {
    let cancelled = false;
    void backend.assignments.list({}).then(({ items }) => {
      if (!cancelled) {
        setAudits(items);
        setSelectedId(requestedAuditId && items.some((item) => item.auditId === requestedAuditId) ? requestedAuditId : null);
      }
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend, requestedAuditId]);

  const visible = status === "all" ? audits : audits.filter((audit) => audit.status === status);
  const selected = audits.find((audit) => audit.auditId === selectedId) ?? null;

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel="Manager Audits">
      <div className="manager-ops-page" data-testid="manager-audits-page">
        <PageHeader eyebrow="Department operations" title="Audit Work Queue" description="Open the exact Audit dossier, owner, Due Date, and next action without borrowing Inspector execution authority." />
        <CommandError message={error} />
        <section aria-label="Audit filters" className="manager-ops-filters">
          <label>Status<select value={status} onChange={(event) => setStatus(event.target.value)}><option value="all">All statuses</option><option value="IN_PROGRESS">In progress</option></select></label>
        </section>
        <div className="manager-ops-layout">
          <section aria-label="Audit register" className="manager-ops-register">
            {visible.map((audit) => (
              <article className="manager-ops-card" key={audit.auditId}>
                <div><p className="eyebrow">{audit.status}</p><h2>{audit.auditId}</h2><p>{audit.organizationName} · {audit.title}</p></div>
                <dl><div><dt>Due Date</dt><dd>{formatLocalDate(audit.dueDate)}</dd></div><div><dt>Next action</dt><dd>{audit.nextAction}</dd></div></dl>
                <button onClick={() => setSelectedId(audit.auditId)} type="button">Open Audit {audit.auditId}</button>
              </article>
            ))}
          </section>
          {selected ? (
            <section aria-label={`Audit ${selected.auditId} dossier`} className="manager-ops-dossier">
              <p className="eyebrow">Exact Audit dossier</p><h2>{selected.auditId}</h2><p>{selected.title}</p>
              <dl className="manager-ops-facts"><div><dt>Organization</dt><dd>{selected.organizationName}</dd></div><div><dt>Status</dt><dd>{selected.status}</dd></div><div><dt>Current owner</dt><dd>{selected.currentOwnerRole === "inspector" ? "CAA Inspector" : selected.currentOwnerRole ?? "Owner unavailable"} · {selected.currentOwnerDisplayName ?? selected.currentOwnerId ?? "No typed owner"}{selected.currentOwnerId ? ` · ${selected.currentOwnerId}` : ""}</dd></div><div><dt>Next action</dt><dd>{selected.nextAction}</dd></div><div><dt>Due Date</dt><dd>{formatLocalDate(selected.dueDate)}</dd></div></dl>
              <button aria-label={`Checklist unavailable for ${selected.auditId}`} disabled title={`Audit ${selected.auditId} has no declared Department Manager checklist-execution route.`} type="button">{selected.nextAction}</button>
            </section>
          ) : null}
        </div>
      </div>
    </WorkspaceShell>
  );
}
