import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { FindingView } from "../../backend/backend";
import { CommandError, errorMessage, FindingFacts, PageHeader, WorkspaceShell } from "../shared/workspace-shell";

export function ManagerFindingsReviewPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("manager") ?? runtime.backend, [runtime]);
  const [findings, setFindings] = useState<FindingView[]>([]);
  const [status, setStatus] = useState("all");
  const [query, setQuery] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [searchParams] = useSearchParams();
  const requestedFindingId = searchParams.get("findingId");
  const requestedOrganizationId = searchParams.get("organizationId");

  useEffect(() => {
    let cancelled = false;
    void backend.findings.list({}).then(({ items }) => {
      if (!cancelled) {
        const scopedItems = requestedOrganizationId
          ? items.filter((item) => item.organizationId === requestedOrganizationId)
          : items;
        setFindings(scopedItems);
        setSelectedId(
          requestedFindingId && scopedItems.some((item) => item.id === requestedFindingId)
            ? requestedFindingId
            : null,
        );
      }
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend, requestedFindingId, requestedOrganizationId]);

  const visible = findings.filter((finding) => {
    const matchesStatus = status === "all" || finding.status === status;
    const haystack = `${finding.findingNumber} ${finding.title} ${finding.organizationName}`.toLowerCase();
    return matchesStatus && haystack.includes(query.toLowerCase());
  });
  const selected = findings.find((finding) => finding.id === selectedId) ?? null;

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel="Findings Review">
      <div className="manager-ops-page" data-testid="manager-findings-review-page">
        <PageHeader eyebrow="Lifecycle oversight" title="Findings Review" description="Review Finding ownership, CAP and Evidence state, Due Date, and the exact next action." />
        <CommandError message={error} />
        {requestedOrganizationId ? <p className="manager-ops-scope">Organization scope: {requestedOrganizationId}</p> : null}
        <section aria-label="Finding filters" className="manager-ops-filters">
          <label>Status<select value={status} onChange={(event) => setStatus(event.target.value)}><option value="all">All statuses</option>{[...new Set(findings.map((finding) => finding.status))].map((value) => <option key={value} value={value}>{value}</option>)}</select></label>
          <label>Search Findings<input type="search" value={query} onChange={(event) => setQuery(event.target.value)} /></label>
        </section>
        <div className="manager-ops-layout">
          <section aria-label="Finding register" className="manager-ops-register">
            {visible.map((finding) => (
              <article aria-label={`Finding ${finding.id}`} className="manager-ops-card" data-finding-id={finding.id} key={finding.id}>
                <button aria-pressed={selectedId === finding.id} className="manager-ops-record-select" onClick={() => setSelectedId(finding.id)} type="button"><span>{finding.findingNumber}</span><strong>{finding.title}</strong></button>
                <p>{finding.organizationName} · {finding.status}</p>
                <Link to={`/department-manager/evidence/${encodeURIComponent(finding.id)}`}>Open Evidence {finding.id}</Link>
              </article>
            ))}
            {visible.length === 0 ? <p>No Findings match the active Organization, status, and search scope.</p> : null}
          </section>
          <section aria-label="Selected Finding dossier" className="manager-ops-dossier">
            {selected ? <><p className="eyebrow">Exact Finding dossier</p><h2>{selected.id}</h2><FindingFacts finding={selected} /></> : <p>Select a Finding.</p>}
          </section>
        </div>
      </div>
    </WorkspaceShell>
  );
}
