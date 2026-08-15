import { useEffect, useMemo, useState } from "react";

import { useBackendForRole } from "../../app/providers";
import type { CanonicalAuditScopeOption, CanonicalQuestionCatalogEntry } from "../../backend/backend";
import { AdminError, AdminPage } from "./admin-workspace-shared";

export function ChecklistBuilderRoute() {
  return <ChecklistBuilderPage />;
}

export function ChecklistBuilderPage() {
  const backend = useBackendForRole("admin");
  const catalog = backend.canonicalCatalog;
  const [scopes, setScopes] = useState<CanonicalAuditScopeOption[]>([]);
  const [scopeId, setScopeId] = useState("");
  const [catalogVersion, setCatalogVersion] = useState("");
  const [questions, setQuestions] = useState<CanonicalQuestionCatalogEntry[]>([]);
  const [query, setQuery] = useState("");
  const [formCode, setFormCode] = useState("");
  const [cursor, setCursor] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const selectedScope = useMemo(() => scopes.find((scope) => scope.providerScopeId === scopeId) ?? null, [scopeId, scopes]);

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

  return <AdminPage testId="admin-checklist-builder-page" routeLabel="Checklist Builder" title="Approved AGA Catalog" description="Browse the source-approved operational catalog. This surface does not create an approval, publication, or internal review workflow.">
    <AdminError message={error} />
    <section className="admin-filter-bar" aria-label="Approved catalog filters">
      <label>Foundation scope<select aria-label="Foundation scope" value={scopeId} onChange={(event) => chooseScope(event.target.value)}><option value="">Choose an exact scope</option>{scopes.map((scope) => <option key={scope.providerScopeId} value={scope.providerScopeId}>{scope.organizationName} · {scope.targetLabel}</option>)}</select></label>
      <label>Search<input aria-label="Search approved questions" value={query} onChange={(event) => setQuery(event.target.value)} /></label>
      <label>Form code<input aria-label="Filter approved form" value={formCode} onChange={(event) => setFormCode(event.target.value)} /></label>
      <button disabled={loading || !scopeId} onClick={() => void loadCatalog()} type="button">{loading ? "Loading…" : "Load approved questions"}</button>
    </section>
    {selectedScope ? <p className="admin-scope-note">Scope {selectedScope.providerScopeId} · catalog {catalogVersion} · {selectedScope.usageClass}</p> : null}
    <section className="admin-card-register admin-dense-register" role="list" aria-label="Approved AGA catalog questions">
      {questions.map((question) => <article className="admin-record-card" key={question.questionVersionId} role="listitem"><header><div><b>{question.questionVersionId}</b><small>{question.formCode} · source ordinal {question.ordinal}</small></div><span>{question.sourceGapState}</span></header><p>{question.prompt ?? "Question prompt unavailable"}</p><dl><div><dt>Immutable digest</dt><dd>{question.questionDigest}</dd></div><div><dt>Configured reference</dt><dd>{question.configuredReference ?? "Not provided"}</dd></div><div><dt>Expected Evidence</dt><dd>{question.expectedEvidence ?? "Not provided"}</dd></div><div><dt>Selection</dt><dd>{question.canSelect ? "Available to authorized Manager" : "Unavailable"}</dd></div></dl></article>)}
      {!questions.length ? <p>Choose an exact scope and load the server-owned approved catalog. No question is implicitly selected.</p> : null}
    </section>
    {cursor ? <button disabled={loading} onClick={() => void loadCatalog(cursor)} type="button">Load next catalog page</button> : null}
  </AdminPage>;
}
