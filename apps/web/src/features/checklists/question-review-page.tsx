import { useCallback, useEffect, useMemo, useState } from "react";

import { useApplicationRuntime } from "../../app/providers";
import type {
  CanonicalQuestionCatalogEntry,
  CanonicalQuestionReviewAction,
  CanonicalQuestionReviewCommandInput,
  CanonicalQuestionUsageClass,
} from "../../backend/backend";
import { CommandError, WorkspaceShell, errorMessage } from "../shared/workspace-shell";

const CATALOG_VERSION = "aga-preprod@1.0.0";
const PAGE_SIZE = 25;
const REASONS = ["MANAGER_SCOPE_DECISION", "SOURCE_MAPPING_REQUIRED", "CLASSIFICATION_EXPERT_REVIEW", "MANAGER_EXACT_RESOLUTION"] as const;

function operationId(prefix: string): string {
  return `${prefix}-${globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`}`;
}

export function QuestionReviewPage() {
  const runtime = useApplicationRuntime();
  const client = runtime.backend.canonicalQuestionReview;
  const [mode, setMode] = useState<CanonicalQuestionUsageClass>("PREPROD_EXERCISE");
  const [search, setSearch] = useState("");
  const [formCode, setFormCode] = useState("");
  const [domain, setDomain] = useState("");
  const [topic, setTopic] = useState("");
  const [riskBand, setRiskBand] = useState("");
  const [sourceGapState, setSourceGapState] = useState("");
  const [selectedFilter, setSelectedFilter] = useState<"all" | "selected" | "unselected">("all");
  const [scopeId, setScopeId] = useState("");
  const [cursor, setCursor] = useState<string | undefined>();
  const [previousCursors, setPreviousCursors] = useState<string[]>([]);
  const [items, setItems] = useState<CanonicalQuestionCatalogEntry[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [total, setTotal] = useState(0);
  const [selected, setSelected] = useState<CanonicalQuestionCatalogEntry | null>(null);
  const [reason, setReason] = useState<(typeof REASONS)[number]>(REASONS[0]);
  const [newDomain, setNewDomain] = useState("");
  const [newTopic, setNewTopic] = useState("");
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);

  const load = useCallback(async (signal?: AbortSignal) => {
    if (!client) {
      setError("Canonical Question Review is unavailable in this build profile.");
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const response = await client.reviewQueue({ mode, catalogVersion: CATALOG_VERSION, search, formCode, domain, topic, riskBand, sourceGapState, selected: selectedFilter, scopeId: scopeId || undefined, cursor, limit: PAGE_SIZE }, { signal });
      if (signal?.aborted) return;
      setItems(response.items);
      setNextCursor(response.nextCursor);
      setTotal(response.totalCount);
      setSelected(null);
      setError(null);
    } catch (cause) {
      if (!signal?.aborted) setError(errorMessage(cause));
    } finally {
      if (!signal?.aborted) setLoading(false);
    }
  }, [client, cursor, domain, formCode, mode, riskBand, search, selectedFilter, sourceGapState, scopeId, topic]);

  useEffect(() => {
    const controller = new AbortController();
    void load(controller.signal);
    return () => controller.abort();
  }, [load]);

  async function choose(row: CanonicalQuestionCatalogEntry) {
    setSelected(row);
    if (!client) return;
    try {
      const detail = await client.getQuestion({ catalogVersion: CATALOG_VERSION, usageClass: mode, questionVersionId: row.questionVersionId });
      setSelected(detail);
    } catch (cause) {
      setError(errorMessage(cause));
    }
  }

  async function decide(action: CanonicalQuestionReviewAction, extra: Partial<CanonicalQuestionReviewCommandInput> = {}) {
    if (!client || !selected) return;
    setBusy(true);
    setError(null);
    setStatus(null);
    try {
      const result = await client.command({ operationId: operationId("question-review"), idempotencyKey: operationId("question-review-idempotency"), mode, questionVersionId: selected.questionVersionId, action, reason, ...extra });
      setStatus(`${action} recorded${result.replayed ? " (idempotent replay)" : ""}.`);
      await load();
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  function resetFilters() {
    setSearch(""); setFormCode(""); setDomain(""); setTopic(""); setRiskBand(""); setSourceGapState(""); setSelectedFilter("all"); setScopeId(""); setCursor(undefined); setPreviousCursors([]);
  }

  const queueRange = useMemo(() => {
    const start = cursor ? Number(cursor) + 1 : 1;
    return items.length ? `${start}–${start + items.length - 1}` : "0";
  }, [cursor, items.length]);
  const exercise = mode === "PREPROD_EXERCISE";
  const authorityAvailable = false;
  const modeReason = exercise ? "PREPROD_EXERCISE review records cannot invoke technical approval or publication." : "Governed technical approval and publication remain on the existing candidate authority route.";

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel="Question Review">
      <div className="canonical-question-review" data-testid="canonical-question-review-page">
        <header className="canonical-question-review__header">
          <p className="eyebrow">Checklist Management · Question Review</p>
          <h1>Find → Compare → Decide</h1>
          <p>Review one bounded server page at a time. The Decision file records a canonical, attributed outcome; it never silently publishes a checklist.</p>
        </header>
        <section className="canonical-question-review__mode" aria-label="Question Review mode">
          <label>Review mode<select value={mode} onChange={(event) => { setMode(event.target.value as CanonicalQuestionUsageClass); setCursor(undefined); setPreviousCursors([]); }}><option value="PREPROD_EXERCISE">Preprod exercise</option><option value="GOVERNED_OPERATIONAL">Governed operational</option></select></label>
          <span className={exercise ? "canonical-boundary canonical-boundary--exercise" : "canonical-boundary"}>{exercise ? "Exercise / source mapping required" : "Governed / server eligibility required"}</span>
          <span role="note">{modeReason}</span>
        </section>
        <CommandError message={error} />
        {status ? <p className="canonical-question-review__status" role="status">{status}</p> : null}
        <section className="canonical-question-review__filters" aria-label="Question Review search and filters">
          <div className="canonical-question-review__filter-heading"><div><span className="eyebrow">Step 1 · Find</span><h2>Review queue</h2><p>Search and filters stay server-side; at most 25 rows are returned.</p></div><button disabled={busy || (!search && !formCode && !domain && !topic && !riskBand && !sourceGapState && selectedFilter === "all" && !scopeId)} onClick={resetFilters} type="button">Clear filters</button></div>
          <label>Search<input aria-label="Question Review search" value={search} onChange={(event) => { setSearch(event.target.value); setCursor(undefined); setPreviousCursors([]); }} placeholder="Form, proposal, or question identity" /></label>
          <div className="canonical-question-review__filter-grid"><label>Form<input aria-label="Question Review form filter" value={formCode} onChange={(event) => { setFormCode(event.target.value); setCursor(undefined); setPreviousCursors([]); }} /></label><label>Domain<input aria-label="Question Review domain filter" value={domain} onChange={(event) => { setDomain(event.target.value); setCursor(undefined); setPreviousCursors([]); }} /></label><label>Topic<input aria-label="Question Review topic filter" value={topic} onChange={(event) => { setTopic(event.target.value); setCursor(undefined); setPreviousCursors([]); }} /></label><label>Risk band<input aria-label="Question Review risk band filter" value={riskBand} onChange={(event) => { setRiskBand(event.target.value); setCursor(undefined); setPreviousCursors([]); }} /></label><label>Source gap<input aria-label="Question Review source gap filter" value={sourceGapState} onChange={(event) => { setSourceGapState(event.target.value); setCursor(undefined); setPreviousCursors([]); }} /></label><label>Selected state<select aria-label="Question Review selected filter" value={selectedFilter} onChange={(event) => { setSelectedFilter(event.target.value as typeof selectedFilter); setCursor(undefined); setPreviousCursors([]); }}><option value="all">All questions</option><option value="selected">Selected in scope</option><option value="unselected">Not selected</option></select></label><label>Scope draft ID<input aria-label="Question Review scope filter" value={scopeId} onChange={(event) => { setScopeId(event.target.value); setCursor(undefined); setPreviousCursors([]); }} placeholder="Optional" /></label></div>
        </section>
        <section className="canonical-question-review__facts" aria-label="Question Review summary"><article><span>Catalog</span><strong>{CATALOG_VERSION}</strong><small>{exercise ? "Disposable exercise boundary" : "Published operational boundary"}</small></article><article><span>Matching questions</span><strong>{total || items.length}</strong><small>server-owned count</small></article><article><span>Current page</span><strong>{queueRange}</strong><small>of bounded results</small></article><article><span>Page size</span><strong>25</strong><small>maximum text-bearing rows</small></article></section>
        <section className="canonical-question-review__workspace" aria-label="Question Review queue and Decision file">
          <section className="canonical-question-review__queue" aria-label="Question queue"><header><div><span className="eyebrow">Step 2 · Compare</span><h2>Question queue</h2></div><div className="canonical-question-review__pagination"><button disabled={!previousCursors.length || loading} onClick={() => { const history = [...previousCursors]; setCursor(history.pop()); setPreviousCursors(history); }} type="button">Previous</button><span aria-live="polite">{queueRange}</span><button disabled={!nextCursor || loading} onClick={() => { if (!nextCursor) return; setPreviousCursors((history) => [...history, cursor ?? ""]); setCursor(nextCursor); }} type="button">Next</button></div></header>{loading ? <p role="status">Loading server queue…</p> : null}<div className="canonical-question-review__rows" role="list">{items.map((row) => <article className={selected?.questionVersionId === row.questionVersionId ? "is-selected" : ""} key={row.questionVersionId} role="listitem"><button className="canonical-question-review__row-button" onClick={() => void choose(row)} type="button"><strong>{row.formCode} · {row.ordinal}</strong><span>{row.proposalId}</span><small>{row.proposedDomain ?? "Unclassified"} · {row.proposedTopic ?? "No topic"} · {row.sourceGapState}</small></button><details><summary>Technical identifiers</summary><dl><div><dt>Question version</dt><dd>{row.questionVersionId}</dd></div><div><dt>Digest</dt><dd>{row.questionDigest}</dd></div><div><dt>Source locator</dt><dd>{row.sourceLocator ?? "Not supplied"}</dd></div></dl></details></article>)}</div>{!loading && !items.length ? <p className="canonical-question-review__empty">No questions match these filters.</p> : null}</section>
          <aside className="canonical-question-review__dossier" aria-label="Selected question Decision file">{selected ? <><header><div><span className="eyebrow">Step 3 · Decide</span><h2>Decision file</h2></div><span className="canonical-selected-chip">Selected</span></header><blockquote>{selected.prompt ?? "Select the row to load the transient question wording."}</blockquote><dl className="canonical-question-review__details"><div><dt>Form / ordinal</dt><dd>{selected.formCode} · {selected.ordinal}</dd></div><div><dt>Domain</dt><dd>{selected.proposedDomain ?? "Not classified"}</dd></div><div><dt>Topic</dt><dd>{selected.proposedTopic ?? "No topic"}</dd></div><div><dt>Source gap</dt><dd>{selected.sourceGapState}</dd></div><div><dt>Publication</dt><dd>{selected.canPublish ? "Eligible only after governed authority" : "Unavailable"}</dd></div></dl><details><summary>Governance signals</summary><dl><div><dt>Question version</dt><dd>{selected.questionVersionId}</dd></div><div><dt>Prompt digest</dt><dd>{selected.questionDigest}</dd></div><div><dt>Configured reference</dt><dd>{selected.configuredReference ?? "Not supplied"}</dd></div><div><dt>Expected evidence</dt><dd>{selected.expectedEvidence ?? "Not supplied"}</dd></div></dl></details><section className="canonical-question-review__actions" aria-label="Question Review decisions"><label>Controlled reason<select aria-label="Controlled reason" value={reason} onChange={(event) => setReason(event.target.value as (typeof REASONS)[number])}>{REASONS.map((item) => <option key={item} value={item}>{item}</option>)}</select></label><div className="canonical-question-review__action-grid"><button disabled={busy} onClick={() => void decide("RETAIN")} type="button">Retain</button><button disabled={busy} onClick={() => void decide("INCLUDE")} type="button">Include</button><button disabled={busy} onClick={() => void decide("EXCLUDE")} type="button">Exclude</button><button disabled={busy} onClick={() => void decide("DEFER")} type="button">Defer</button></div><div className="canonical-question-review__reclassify"><label>New domain<input aria-label="New domain" value={newDomain} onChange={(event) => setNewDomain(event.target.value)} /></label><button disabled={busy || !newDomain.trim()} onClick={() => void decide("DOMAIN_RECLASSIFIED", { domain: newDomain.trim() })} type="button">Reclassify domain</button><label>Topic<input aria-label="New topic" value={newTopic} onChange={(event) => setNewTopic(event.target.value)} /></label><button disabled={busy || !newTopic.trim()} onClick={() => void decide("TOPIC_RECLASSIFIED", { topic: newTopic.trim() })} type="button">Update topic</button></div><div className="canonical-question-review__authority-actions"><button disabled={busy || !authorityAvailable} title={modeReason} onClick={() => void decide("TECHNICAL_APPROVE")} type="button">Technical approval</button><button disabled={busy || !authorityAvailable} title={modeReason} onClick={() => void decide("PUBLISH")} type="button">Publish</button></div><p className="canonical-question-review__disabled" role="note">{modeReason} Use the canonical candidate authority surface for governed technical approval and publication.</p></section></> : <div className="canonical-question-review__dossier-empty"><span className="eyebrow">Step 3 · Decide</span><h2>Open a question</h2><p>Select a queue row to load its transient wording and Decision file. No body is placed in the URL, browser storage, or telemetry.</p></div>}</aside>
        </section>
      </div>
    </WorkspaceShell>
  );
}
