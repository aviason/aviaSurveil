import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { CapRevisionView, EvidenceVersionView, FindingView } from "../../backend/backend";
import { CommandError, errorMessage, FindingFacts, PageHeader, WorkspaceShell } from "../shared/workspace-shell";

export function DepartmentClosureReviewPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("manager") ?? runtime.backend, [runtime]);
  const { findingId } = useParams<{ findingId: string }>();
  const [finding, setFinding] = useState<FindingView | null>(null);
  const [caps, setCaps] = useState<CapRevisionView[]>([]);
  const [evidence, setEvidence] = useState<EvidenceVersionView[]>([]);
  const [reason, setReason] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    if (!findingId) {
      setError("The closure review route does not contain a finding identity.");
      return () => { cancelled = true; };
    }
    void backend.findings.get({ findingId }).then(async (loaded) => {
      const [capHistory, evidenceHistory] = await Promise.all([backend.caps.listRevisions({ findingId: loaded.id }), backend.evidence.listVersions({ findingId: loaded.id })]);
      if (!cancelled) { setFinding(loaded); setCaps(capHistory.items); setEvidence(evidenceHistory); }
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend, findingId]);

  async function authorizeClosure() {
    if (!finding) return;
    if (!reason.trim()) { setError("Authorized closure reason is required"); return; }
    setBusy(true); setError(null);
    try {
      const updated = await backend.findings.authorizedClose({ operationId: `OP-MANAGER-AUTHORIZED-CLOSE-${finding.id}-${finding.revision}`, findingId: finding.id, expectedFindingRevision: finding.revision, reason });
      setFinding(updated);
    } catch (cause) { setError(errorMessage(cause)); } finally { setBusy(false); }
  }

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel="Department Manager Review">
      <div className="manager-ops-page" data-testid="manager-closure-review-page">
        <PageHeader eyebrow="Separate closure authority" title="Department Manager Review" description="Review immutable Finding, CAP, and Evidence identity before using the explicit authorized closure path." />
        <CommandError message={error} />
        {finding ? <div className="manager-ops-layout"><section aria-label="Closure evidence package" className="manager-ops-register"><article className="manager-ops-card"><h2>Finding dossier</h2><FindingFacts finding={finding} /></article><article className="manager-ops-card"><p className="eyebrow">Immutable CAP history</p><h2>{caps.length} revisions</h2>{caps.map((cap) => <p key={cap.id}>{cap.id} · {cap.status}</p>)}</article><article className="manager-ops-card"><p className="eyebrow">Immutable Evidence history</p><h2>{evidence.length} versions</h2>{evidence.map((version) => <p key={version.id}>{version.id} · {version.reviewState}</p>)}</article></section><section aria-label={`Authorized closure ${finding.id}`} className="manager-ops-dossier"><p className="eyebrow">Exact Finding command</p><h2>{finding.id}</h2><dl className="manager-ops-facts"><div><dt>Status</dt><dd data-testid="manager-closure-status">{finding.status}</dd></div><div><dt>Closure basis</dt><dd data-testid="manager-closure-basis">{finding.closureBasis ?? "Not recorded"}</dd></div><div><dt>Revision</dt><dd>{finding.revision}</dd></div><div><dt>Authority</dt><dd>Department Manager</dd></div></dl>{finding.status !== "CLOSED" ? <><label>Authorized closure reason<textarea rows={5} value={reason} onChange={(event) => setReason(event.target.value)} /></label><button disabled={busy} onClick={() => void authorizeClosure()} type="button">Authorize closure</button></> : <p>Authorized closure is audit-logged. CAP acceptance was not used as Finding closure.</p>}<Link to={`/department-manager/evidence/${finding.id}`}>Return to Evidence {finding.id}</Link></section></div> : null}
      </div>
    </WorkspaceShell>
  );
}
