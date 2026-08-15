import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { useBackendForRole } from "../../app/providers";
import type { CanonicalAuditScopeOption, CanonicalQuestionCatalogEntry } from "../../backend/backend";
import { CommandError, WorkspaceShell } from "../shared/workspace-shell";

/**
 * The manager-facing catalog register is deliberately read-only. Question
 * selection belongs to the server-owned New Audit draft, where the exact
 * immutable IDs and catalog root are frozen into that Audit's scope snapshot.
 */
export function ChecklistManagementPage() {
  const backend = useBackendForRole("manager");
  const catalog = backend.canonicalCatalog;
  const [scopes, setScopes] = useState<CanonicalAuditScopeOption[]>([]);
  const [scopeId, setScopeId] = useState("");
  const [catalogVersion, setCatalogVersion] = useState("");
  const [questions, setQuestions] = useState<CanonicalQuestionCatalogEntry[]>([]);
  const [query, setQuery] = useState("");
  const [formCode, setFormCode] = useState("");
  const [cursor, setCursor] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const selectedScope = useMemo(
    () => scopes.find((scope) => scope.providerScopeId === scopeId) ?? null,
    [scopeId, scopes],
  );

  useEffect(() => {
    let cancelled = false;
    if (!catalog) {
      setError("Approved catalog is unavailable for this role.");
      return () => { cancelled = true; };
    }
    void catalog.listScopeOptions({ limit: 100 }).then((page) => {
      if (!cancelled) setScopes(page.items);
    }).catch((cause) => {
      if (!cancelled) setError(cause instanceof Error ? cause.message : "Approved catalog scope could not be loaded.");
    });
    return () => { cancelled = true; };
  }, [catalog]);

  async function loadCatalog(nextCursor?: string) {
    if (!catalog || !selectedScope || !catalogVersion) {
      setError("Choose an exact active scope and catalog version before loading approved questions.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const page = await catalog.listCatalog({
        catalogVersion,
        usageClass: "GOVERNED_OPERATIONAL",
        search: query || undefined,
        formCode: formCode || undefined,
        scopeId: selectedScope.providerScopeId,
        cursor: nextCursor,
        limit: 100,
      });
      setQuestions((current) => nextCursor ? [...current, ...page.items] : page.items);
      setCursor(page.nextCursor);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Approved catalog could not be loaded.");
    } finally {
      setLoading(false);
    }
  }

  function chooseScope(value: string) {
    const scope = scopes.find((candidate) => candidate.providerScopeId === value);
    setScopeId(value);
    setCatalogVersion(scope?.catalogVersion ?? "");
    setQuestions([]);
    setCursor(null);
  }

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel="Checklist Management">
      <div className="manager-ops-page" data-testid="manager-checklist-management-page">
        <header className="authority-page-head workbench-page-header">
          <p className="eyebrow">Approved AGA catalog</p>
          <h1>Checklist Management</h1>
          <p>Browse all source-approved questions. The New Audit workflow selects the exact subset and records its immutable scope snapshot.</p>
        </header>
        <CommandError message={error} />
        <section className="manager-ops-layout" aria-label="Approved catalog register">
          <div className="manager-filter-row">
            <label>Foundation scope<select aria-label="Foundation scope" value={scopeId} onChange={(event) => chooseScope(event.target.value)}><option value="">Choose an exact scope</option>{scopes.map((scope) => <option key={scope.providerScopeId} value={scope.providerScopeId}>{scope.organizationName} · {scope.targetLabel}</option>)}</select></label>
            <label>Search<input aria-label="Search approved questions" value={query} onChange={(event) => setQuery(event.target.value)} /></label>
            <label>Form code<input aria-label="Filter approved form" value={formCode} onChange={(event) => setFormCode(event.target.value)} /></label>
            <button disabled={loading || !scopeId} onClick={() => void loadCatalog()} type="button">{loading ? "Loading…" : "Load approved questions"}</button>
          </div>
          {selectedScope ? <p>Scope {selectedScope.providerScopeId} · catalog {catalogVersion} · {selectedScope.usageClass} · {questions.length} loaded</p> : null}
          <p><Link to="/department-manager/new-audit/step-4">Open New Audit question selection</Link></p>
          <section className="admin-card-register admin-dense-register" role="list" aria-label="Approved AGA catalog questions">
            {questions.map((question) => <article className="admin-record-card" key={question.questionVersionId} role="listitem"><header><div><b>{question.questionVersionId}</b><small>{question.formCode} · source ordinal {question.ordinal}</small></div><span>{question.sourceGapState}</span></header><p>{question.prompt ?? "Question prompt unavailable"}</p><dl><div><dt>Immutable digest</dt><dd>{question.questionDigest}</dd></div><div><dt>Configured reference</dt><dd>{question.configuredReference ?? "Not provided"}</dd></div><div><dt>Expected Evidence</dt><dd>{question.expectedEvidence ?? "Not provided"}</dd></div><div><dt>Selection</dt><dd>{question.canSelect ? "Available to authorized Manager" : "Unavailable"}</dd></div></dl></article>)}
            {!questions.length ? <p>Choose an exact scope and load the server-owned approved catalog. No question is implicitly selected.</p> : null}
          </section>
          {cursor ? <button disabled={loading} onClick={() => void loadCatalog(cursor)} type="button">Load next catalog page</button> : null}
        </section>
      </div>
    </WorkspaceShell>
  );
}
