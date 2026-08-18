import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { CapRevisionView, FindingView } from "../../backend/backend";
import { CommandError, errorMessage, formatLocalDate, PageHeader, WorkspaceShell } from "../shared/workspace-shell";
import { recordReference } from "../shared/record-presentation";

interface CapRecord { finding: FindingView; revisions: CapRevisionView[] }

export function ManagerCapMonitoringPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("manager") ?? runtime.backend, [runtime]);
  const [records, setRecords] = useState<CapRecord[]>([]);
  const [selectedId, setSelectedId] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    void backend.findings.list({}).then(async ({ items }) => Promise.all(items.map(async (finding) => ({ finding, revisions: (await backend.caps.listRevisions({ findingId: finding.id })).items })))).then((loaded) => {
      if (!cancelled) { setRecords(loaded); setSelectedId(""); }
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend]);

  const selected = records.find((record) => record.finding.id === selectedId) ?? null;
  const latest = selected?.revisions.at(-1) ?? null;

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel="CAP Monitoring">
      <div className="manager-ops-page" data-testid="manager-cap-monitoring-page">
        <PageHeader eyebrow="Corrective action oversight" title="CAP Monitoring" description="Track immutable CAP revisions, target dates, Finding owner, and closure readiness without CAP review authority." />
        <CommandError message={error} />
        <section aria-label="CAP filters" className="manager-ops-filters"><label>Finding<select value={selectedId} onChange={(event) => setSelectedId(event.target.value)}>{records.map(({ finding }) => <option key={finding.id} value={finding.id}>{finding.findingNumber} · {finding.title}</option>)}</select></label></section>
        {selected ? <div className="manager-ops-layout"><section aria-label="CAP revision register" className="manager-ops-register">{selected.revisions.length ? selected.revisions.map((revision) => <article className="manager-ops-card" key={revision.id}><p className="eyebrow">CAP revision {revision.revision} · {recordReference("CAP", revision.id)}</p><h2>Corrective action plan</h2><p>{revision.status.replaceAll("_", " ")} · Target {formatLocalDate(revision.targetCompletionDate)}</p><p>{revision.correctiveAction}</p></article>) : <p>No CAP revision has been submitted for {selected.finding.findingNumber}.</p>}</section><section aria-label={`${selected.finding.findingNumber} CAP dossier`} className="manager-ops-dossier"><p className="eyebrow">Latest lifecycle state</p><h2>{selected.finding.title}</h2><p>{selected.finding.findingNumber}</p><dl className="manager-ops-facts"><div><dt>CAP status</dt><dd>{latest?.status.replaceAll("_", " ") ?? "Not submitted"}</dd></div><div><dt>Current owner</dt><dd>{selected.finding.currentOwnerType}</dd></div><div><dt>Next action</dt><dd>{selected.finding.nextAction}</dd></div><div><dt>Target</dt><dd>{latest ? formatLocalDate(latest.targetCompletionDate) : "Not set"}</dd></div></dl><Link to={`/department-manager/findings/${encodeURIComponent(selected.finding.id)}/closure-review`}>Open closure review</Link></section></div> : <p>Select a Finding to inspect its CAP revisions.</p>}
      </div>
    </WorkspaceShell>
  );
}
