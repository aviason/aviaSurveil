import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type {
  AGADemoWorkspaceCapability,
  AGADemoWorkspaceCommand,
  AGADemoWorkspaceDraft,
  AGADemoWorkspaceProviderConfiguration,
  AGADemoWorkspaceQuery,
  AGADemoWorkspaceQueryResponse,
} from "../../backend/aga-demo-workspace";
import type { Role } from "../../backend/backend";
import { BackendHttpError } from "../../backend/http-backend";
import { CommandError, PageHeader, WorkspaceShell, errorMessage } from "../shared/workspace-shell";

type WorkspaceRow = NonNullable<AGADemoWorkspaceQueryResponse["items"]>[number];
const PAGE_SIZE = 25;
const MAX_PAGE_CACHE = 4;
const REASON_OPTIONS = [
  "MANAGER_SCOPE_DECISION",
  "CLASSIFICATION_EXPERT_REVIEW",
  "MANAGER_EXACT_RESOLUTION",
  "SIMULATION_SOURCE_GAP_OVERRIDE",
  "SYNTHETIC_CANDIDATE_ADDED",
  "SYNTHETIC_CANDIDATE_REWORDED",
] as const;

type BooleanFilter = "all" | "true" | "false";

type Filters = {
  search: string;
  domainCode: string;
  topicCode: string;
  confidence: string;
  blocker: BooleanFilter;
  sourceGap: BooleanFilter;
  externalInvolvement: BooleanFilter;
  formCode: string;
  disposition: string;
};

const emptyFilters: Filters = {
  search: "",
  domainCode: "",
  topicCode: "",
  confidence: "",
  blocker: "all",
  sourceGap: "all",
  externalInvolvement: "all",
  formCode: "",
  disposition: "",
};

function queryFor(filters: Filters, page: number): AGADemoWorkspaceQuery {
  return {
    operationId: "SEARCH_ITEMS",
    page,
    pageSize: PAGE_SIZE,
    ...(filters.search.trim() ? { search: filters.search.trim() } : {}),
    ...(filters.domainCode.trim() ? { domainCode: filters.domainCode.trim() } : {}),
    ...(filters.topicCode.trim() ? { topicCode: filters.topicCode.trim() } : {}),
    ...(filters.confidence ? { confidence: filters.confidence } : {}),
    ...(filters.blocker !== "all" ? { blocker: filters.blocker } : {}),
    ...(filters.sourceGap !== "all" ? { sourceGap: filters.sourceGap } : {}),
    ...(filters.externalInvolvement !== "all" ? { externalInvolvement: filters.externalInvolvement } : {}),
    ...(filters.formCode.trim() ? { formCode: filters.formCode.trim() } : {}),
    ...(filters.disposition ? { disposition: filters.disposition as "INCLUDE" | "EXCLUDE" | "DEFER" | "UNSET" } : {}),
  };
}

function metadataOnlyResponse(response: AGADemoWorkspaceQueryResponse): AGADemoWorkspaceQueryResponse {
  const stripText = (row: WorkspaceRow): WorkspaceRow => {
    const { questionText: _questionText, questionTextDigest: _questionTextDigest, textOrigin: _textOrigin, ...metadata } = row;
    return metadata as WorkspaceRow;
  };
  return {
    ...response,
    items: response.items?.map(stripText),
    reviewItems: response.reviewItems?.map(stripText),
  };
}

function commandKey(operationId: string): string {
  if (typeof globalThis.crypto?.randomUUID !== "function") {
    throw new Error("This browser cannot create a secure idempotency key for the requested command.");
  }
  return `aga-demo-workspace-${operationId.toLowerCase()}-${globalThis.crypto.randomUUID()}`;
}

async function digestBody(value: string): Promise<string> {
  if (!globalThis.crypto?.subtle) return "";
  const bytes = new TextEncoder().encode(value);
  const digest = await globalThis.crypto.subtle.digest("SHA-256", bytes);
  return `sha256:${Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("")}`;
}

function currentDraftItemCount(draft: AGADemoWorkspaceDraft | null): number {
  return draft?.items?.filter((item) => item.currentLeaf !== false).length ?? 0;
}

function providerLabel(entry: AGADemoWorkspaceProviderConfiguration): string {
  return `${entry.providerTypeCode ?? "Unknown provider"} · ${entry.disposition ?? "Not configured"}`;
}

function conflictMessage(error: unknown): boolean {
  return /conflict|stale|compare-and-swap|cas|revision|digest/i.test(errorMessage(error));
}

function isWorkspaceNotFound(error: unknown): boolean {
  return error instanceof BackendHttpError && error.status === 404;
}

export function AGADemoClassificationWorkspacePage({
  capability,
  role,
  roleLabel,
}: {
  capability: AGADemoWorkspaceCapability;
  role: Role;
  roleLabel: string;
}) {
  const runtime = useApplicationRuntime();
  const client = runtime.backend.agaDemoWorkspace;
  const [filters, setFilters] = useState<Filters>(emptyFilters);
  const [page, setPage] = useState(0);
  const [summary, setSummary] = useState<AGADemoWorkspaceQueryResponse | null>(null);
  const [draft, setDraft] = useState<AGADemoWorkspaceDraft | null>(null);
  const [providers, setProviders] = useState<AGADemoWorkspaceProviderConfiguration[]>([]);
  const [pageResponse, setPageResponse] = useState<AGADemoWorkspaceQueryResponse | null>(null);
  const [selected, setSelected] = useState<WorkspaceRow | null>(null);
  const [history, setHistory] = useState<AGADemoWorkspaceQueryResponse["history"]>([]);
  const [status, setStatus] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [pending, setPending] = useState(false);
  const [refreshVersion, setRefreshVersion] = useState(0);
  const [reasonCode, setReasonCode] = useState<(typeof REASON_OPTIONS)[number]>("MANAGER_SCOPE_DECISION");
  const [mainDomainCode, setMainDomainCode] = useState("");
  const [topicCode, setTopicCode] = useState("");
  const [workspaceBody, setWorkspaceBody] = useState("");
  const [resetReason, setResetReason] = useState("");
  const [resetConfirmed, setResetConfirmed] = useState(false);
  const pageCache = useRef(new Map<string, AGADemoWorkspaceQueryResponse>());

  const filterKey = useMemo(() => JSON.stringify(filters), [filters]);
  const generation = pageResponse?.generation ?? summary?.generation ?? null;
  const draftRevision = draft?.revision ?? 0;
  const classificationAvailable = capability.classificationEnabled && Boolean(client);
  const adminReadResetAvailable = role === "admin" && capability.available && capability.resetEnabled && Boolean(client);
  const workspaceAvailable = classificationAvailable || adminReadResetAvailable;

  const refreshModels = useCallback(async (signal?: AbortSignal) => {
    if (!client || !workspaceAvailable) return;
    setLoading(true);
    if (!capability.classificationEnabled) {
      const summaryResponse = await client.classificationQuery({ operationId: "GET_SUMMARY" }, { signal });
      setSummary(summaryResponse);
      setDraft(null);
      setProviders([]);
      pageCache.current.clear();
      setPageResponse(null);
      setSelected(null);
      setRefreshVersion((value) => value + 1);
      setLoading(false);
      return;
    }
    const [summaryResponse, draftResponse, providerResponse] = await Promise.all([
      client.classificationQuery({ operationId: "GET_SUMMARY" }, { signal }),
      client.classificationQuery({ operationId: "GET_DRAFT" }, { signal }),
      client.classificationQuery({ operationId: "GET_PROVIDER_CONFIGURATION" }, { signal }),
    ]);
    setSummary(summaryResponse);
    setDraft(draftResponse.draft ?? null);
    setProviders(providerResponse.providerConfiguration ?? []);
    pageCache.current.clear();
    setPageResponse(null);
    setSelected(null);
    setRefreshVersion((value) => value + 1);
    setLoading(false);
  }, [capability.classificationEnabled, client, workspaceAvailable]);

  useEffect(() => {
    const controller = new AbortController();
    if (!workspaceAvailable) {
      setLoading(false);
      return () => controller.abort();
    }
    void refreshModels(controller.signal).catch((cause) => {
      if (!controller.signal.aborted) {
        setLoading(false);
        setError(isWorkspaceNotFound(cause)
          ? "Classification inventory is not provisioned in this local AGA workspace."
          : errorMessage(cause));
      }
    });
    return () => controller.abort();
  }, [refreshModels, workspaceAvailable]);

  useEffect(() => {
    const controller = new AbortController();
    if (!classificationAvailable || !client) return () => controller.abort();
    const key = `${filterKey}:${page}`;
    setPageResponse(null);
    setSelected(null);
    // Cached metadata is deliberately never promoted to the loaded page: it
    // contains no sealed question bodies. Returning Page 1 after Page 2 must
    // always refetch its server-authoritative text before rendering it.
    const cached = pageCache.current.get(key);
    void cached;
    setLoading(true);
    void client.classificationQuery(queryFor(filters, page), { signal: controller.signal })
      .then((response) => {
        if (controller.signal.aborted) return;
        pageCache.current.set(key, metadataOnlyResponse(response));
        while (pageCache.current.size > MAX_PAGE_CACHE) {
          const oldest = pageCache.current.keys().next().value;
          if (oldest === undefined) break;
          pageCache.current.delete(oldest);
        }
        setPageResponse(response);
        setLoading(false);
      })
      .catch((cause) => {
        if (!controller.signal.aborted) {
          setLoading(false);
          setError(isWorkspaceNotFound(cause)
            ? "Classification inventory is not provisioned in this local AGA workspace."
            : errorMessage(cause));
        }
      });
    return () => controller.abort();
  }, [classificationAvailable, client, filterKey, filters, page, refreshVersion]);

  useEffect(() => {
    setMainDomainCode(selected?.projection.mainDomainCode ?? "");
    setTopicCode(selected?.projection.topicCodes?.[0] ?? "");
  }, [selected]);

  useEffect(() => {
    const invalidateRestoredPage = (event: PageTransitionEvent) => {
      if (!event.persisted) return;
      pageCache.current.clear();
      setSummary(null);
      setDraft(null);
      setProviders([]);
      setPageResponse(null);
      setSelected(null);
      setStatus(null);
      setError("This restored page was cleared; reload the server-authoritative workspace to continue.");
    };
    window.addEventListener("pageshow", invalidateRestoredPage);
    return () => window.removeEventListener("pageshow", invalidateRestoredPage);
  }, []);

  const updateFilter = useCallback((name: keyof Filters, value: string) => {
    setFilters((current) => ({ ...current, [name]: value }));
    setPage(0);
  }, []);

  const clearFilters = useCallback(() => {
    setFilters(emptyFilters);
    setPage(0);
  }, []);

  const executeCommand = useCallback(async (
    operationId: AGADemoWorkspaceCommand["operationId"],
    extra: Partial<AGADemoWorkspaceCommand> = {},
  ) => {
    if (!client || !generation || !draft || !selected?.questionKey) {
      throw new Error("Select a server-returned question and load the current Draft before issuing a command.");
    }
    const idempotencyKey = commandKey(operationId);
    const command: AGADemoWorkspaceCommand = {
      operationId,
      idempotencyKey,
      expectedGenerationId: generation.generationId ?? "",
      expectedDraftRevision: draft.revision,
      expectedDraftContentDigest: draft.contentDigest,
      targetQuestionKey: selected.questionKey,
      reasonCode: reasonCode || undefined,
      ...extra,
    };
    setPending(true);
    setError(null);
    setStatus(null);
    try {
      const response = await client.classificationCommand(command);
      if (response.draft) setDraft(response.draft);
      if (response.generation) setSummary((current) => current ? { ...current, generation: response.generation! } : current);
      setStatus(`${operationId} recorded${response.replayed ? " (idempotent replay)" : ""}.`);
      pageCache.current.clear();
      setPageResponse(null);
      await refreshModels();
    } catch (cause) {
      setError(conflictMessage(cause) ? "The Draft changed on the server. The current generation and Draft were reloaded; review the selected row again." : errorMessage(cause));
      if (conflictMessage(cause)) await refreshModels();
      throw cause;
    } finally {
      setPending(false);
    }
  }, [client, draft, generation, reasonCode, refreshModels, selected]);

  const runAction = useCallback((operationId: AGADemoWorkspaceCommand["operationId"], extra?: Partial<AGADemoWorkspaceCommand>) => {
    void executeCommand(operationId, extra).catch(() => undefined);
  }, [executeCommand]);

  const createCandidate = useCallback(async () => {
    if (!workspaceBody.trim() || !selected) {
      setError("A non-empty candidate wording and a server-returned projection are required.");
      return;
    }
    const workspaceBodyDigest = await digestBody(workspaceBody);
    if (!workspaceBodyDigest) {
      setError("The browser could not create the required wording digest.");
      return;
    }
    runAction("ADD_CANDIDATE", {
      reasonCode: "SYNTHETIC_CANDIDATE_ADDED",
      workspaceBody,
      workspaceBodyDigest,
      exactProjection: selected.projection,
    });
  }, [runAction, selected, workspaceBody]);

  const rewordCandidate = useCallback(async () => {
    if (!workspaceBody.trim() || !selected) {
      setError("A non-empty successor wording is required; the browser never synthesizes a question identity.");
      return;
    }
    const workspaceBodyDigest = await digestBody(workspaceBody);
    if (!workspaceBodyDigest) {
      setError("The browser could not create the required wording digest.");
      return;
    }
    runAction("REWORD_CANDIDATE", {
      reasonCode: "SYNTHETIC_CANDIDATE_REWORDED",
      workspaceBody,
      workspaceBodyDigest,
    });
  }, [runAction, selected, workspaceBody]);

  const showHistory = useCallback(() => {
    if (!client || role !== "admin") return;
    void client.classificationQuery({ operationId: "GET_HISTORY" })
      .then((response) => setHistory(response.history ?? []))
      .catch((cause) => setError(errorMessage(cause)));
  }, [client, role]);

  const resetGeneration = useCallback(async () => {
    if (!client || role !== "admin" || !generation || !resetConfirmed || !resetReason.trim()) return;
    const idempotencyKey = commandKey("RESET_GENERATION");
    setPending(true);
    setError(null);
    setStatus(null);
    try {
      const response = await client.adminCommand({
        operationId: "RESET_GENERATION",
        idempotencyKey,
        expectedGenerationId: generation.generationId ?? "",
        expectedGenerationRevision: generation.revision,
        expectedGenerationSealDigest: generation.sealDigest,
        reasonCode: resetReason.trim(),
      });
      setStatus(`Generation reset recorded${response.replayed ? " (idempotent replay)" : ""}.`);
      setResetReason("");
      setResetConfirmed(false);
      await refreshModels();
    } catch (cause) {
      setError(conflictMessage(cause) ? "The generation changed before reset. The current generation was reloaded and no reset was applied." : errorMessage(cause));
      if (conflictMessage(cause)) await refreshModels();
    } finally {
      setPending(false);
    }
  }, [client, generation, refreshModels, resetConfirmed, resetReason, role]);

  if (!workspaceAvailable) {
    return (
      <WorkspaceShell roleLabel={roleLabel} routeLabel="AGA Demo Workspace">
        <section className="aga-workspace-unavailable" data-testid="aga-workspace-projection-unavailable" role="alert">
          <h1>AGA demo workspace</h1>
          <p>Classification review is not enabled for this projection. Lifecycle controls remain unavailable until the server declares them.</p>
        </section>
      </WorkspaceShell>
    );
  }

  if (adminReadResetAvailable && !classificationAvailable) {
    return (
      <WorkspaceShell roleLabel={roleLabel} routeLabel="AGA Demo Workspace">
        <main className="aga-classification-workspace" data-testid="aga-workspace-admin-page">
          <PageHeader
            eyebrow="Disposable local-preprod workspace"
            title="AGA generation administration"
            description="CAA administrators can inspect the current disposable generation, its history, and its explicitly authorized reset boundary. Classification review remains unavailable."
            action={<button className="aga-workspace-button" onClick={showHistory} type="button">View generation history</button>}
          />
          <CommandError message={error} />
          {status ? <p className="aga-workspace-status" role="status">{status}</p> : null}
          <section aria-label="Admin generation facts" className="aga-workspace-facts">
            <article><span>Current generation</span><strong>{generation?.generationId ?? "Not loaded"}</strong><small>sealed revision {generation?.revision ?? "—"}</small></article>
            <article><span>Admin capability</span><strong>Read and reset</strong><small>no classification mutations</small></article>
          </section>
          <section aria-label="Admin generation controls" className="aga-workspace-admin-panel">
            <h2>Admin generation history and reset</h2>
            <p>Reset is destructive to this disposable generation only and never writes canonical records.</p>
            {history?.length ? <ul>{history.map((entry) => <li key={entry.generationId}>{entry.generationId} · revision {entry.revision}</li>)}</ul> : <p>No history loaded.</p>}
            <label>Reset reason<input aria-label="Reset reason" value={resetReason} onChange={(event) => setResetReason(event.target.value)} /></label>
            <label className="aga-workspace-confirm"><input aria-label="Confirm generation reset" checked={resetConfirmed} onChange={(event) => setResetConfirmed(event.target.checked)} type="checkbox" /> I understand this resets the disposable generation.</label>
            <button disabled={!resetReason.trim() || !resetConfirmed || pending || !capability.resetEnabled} onClick={() => void resetGeneration()} type="button">Reset generation</button>
          </section>
        </main>
      </WorkspaceShell>
    );
  }

  const rows = pageResponse?.items ?? [];
  const total = pageResponse?.itemCount ?? summary?.itemCount ?? 0;
  const currentPageNumber = pageResponse?.page ?? page;
  const hasNextPage = pageResponse?.nextPage !== undefined;
  const inspectedScopeEligible = providers.filter((entry) => entry.disposition === "INSPECTED_SCOPE_ELIGIBLE");
  const involvementOnly = providers.filter((entry) => entry.disposition === "INVOLVEMENT_ONLY");
  const activeFilterCount = Object.entries(filters).filter(([key, value]) => {
    if (key === "blocker" || key === "sourceGap" || key === "externalInvolvement") return value !== "all";
    return Boolean(value);
  }).length;
  const firstVisibleRow = rows.length > 0 ? currentPageNumber * PAGE_SIZE + 1 : 0;
  const lastVisibleRow = rows.length > 0 ? firstVisibleRow + rows.length - 1 : 0;
  const generationLabel = generation ? `${generation.state ?? "ACTIVE"} · revision ${generation.revision}` : "Not loaded";
  const includeDisabledReason = !selected
    ? "Select a server-returned classification item before including it."
    : !selected.includeEligible
      ? selected.includeEligibilityReason
      : pending
        ? "The current command is pending."
        : "";

  return (
    <WorkspaceShell roleLabel={roleLabel} routeLabel="AGA Demo Workspace">
      <main className="aga-classification-workspace" data-testid="aga-classification-workspace-page">
        <PageHeader
          eyebrow="Department Manager · local AGA candidate"
          title="Classify AGA questions"
          description="A guided review queue for 1,310 sealed questions. Find one, understand the current proposal, then record a Draft decision."
          action={role === "admin" ? <button className="aga-workspace-button" onClick={showHistory} type="button">View generation history</button> : undefined}
        />
        <section aria-label="How to use this demo" className="aga-workspace-guide" data-testid="aga-demo-guidance">
          <div className="aga-workspace-guide__lede">
            <span className="aga-workspace-kicker">Start here</span>
            <h2>What am I looking at?</h2>
            <p>This is a safe local candidate. You are reviewing sealed question text and its proposed classification; nothing here publishes a regulatory decision.</p>
            <span className="aga-workspace-safe-state"><span aria-hidden="true" /> Local candidate only</span>
          </div>
          <ol className="aga-workspace-guide__steps">
            <li><span className="aga-workspace-step-number">1</span><div><strong>Find</strong><p>Search the queue or narrow it with filters.</p></div></li>
            <li><span className="aga-workspace-step-number">2</span><div><strong>Compare</strong><p>Open a row to read its sealed text and signals.</p></div></li>
            <li><span className="aga-workspace-step-number">3</span><div><strong>Decide</strong><p>Record one Draft action with a controlled reason.</p></div></li>
          </ol>
        </section>
        <CommandError message={error} />
        {status ? <p className="aga-workspace-status" role="status">{status}</p> : null}

        <section aria-label="Workspace generation and counts" className="aga-workspace-facts">
          <article><span>Questions in queue</span><strong>{summary?.baseQuestionCount ?? 0}</strong><small>sealed server inventory</small></article>
          <article><span>Current generation</span><strong>{generationLabel}</strong><small>{generation?.generationId ?? "Waiting for server"}</small></article>
          <article><span>Draft decisions</span><strong>{currentDraftItemCount(draft)}</strong><small>effective Draft projection</small></article>
          <article><span>Showing now</span><strong>{total ? `${firstVisibleRow}–${lastVisibleRow}` : "—"}</strong><small>{total ? `of ${total} matching rows` : "No rows loaded"}</small></article>
        </section>

        <section aria-label="Server filters" className="aga-workspace-filter-panel">
          <div className="aga-workspace-filter-head">
            <div><span className="aga-workspace-kicker">Step 1 · Find</span><h2>Review queue</h2><p>Search is sent to the server; the browser keeps only lightweight page metadata.</p></div>
            <div className="aga-workspace-filter-head__actions"><span className="aga-workspace-filter-count">{activeFilterCount ? `${activeFilterCount} filter${activeFilterCount === 1 ? "" : "s"} on` : "All questions"}</span><button className="aga-workspace-quiet-button" disabled={!activeFilterCount || loading} onClick={clearFilters} type="button">Clear filters</button></div>
          </div>
          <label className="aga-workspace-search-field"><span>Search by form, reference, digest, or code</span><input aria-label="AGA search" placeholder="Try FSS-AGA-FORM-002 or a question reference" value={filters.search} onChange={(event) => updateFilter("search", event.target.value)} /></label>
          <details className="aga-workspace-advanced-filters">
            <summary>More filters</summary>
            <div className="aga-workspace-filter-grid">
              <label>Domain<input aria-label="Domain filter" value={filters.domainCode} onChange={(event) => updateFilter("domainCode", event.target.value)} /></label>
              <label>Topic<input aria-label="Topic filter" value={filters.topicCode} onChange={(event) => updateFilter("topicCode", event.target.value)} /></label>
              <label>Form<input aria-label="Form filter" value={filters.formCode} onChange={(event) => updateFilter("formCode", event.target.value)} /></label>
              <label>Sealed confidence<select aria-label="Confidence filter" value={filters.confidence} onChange={(event) => updateFilter("confidence", event.target.value)}><option value="">All</option><option value="HIGH">HIGH</option><option value="MEDIUM">MEDIUM</option><option value="LOW">LOW</option></select></label>
              <label>Blocker<select aria-label="Blocker filter" value={filters.blocker} onChange={(event) => updateFilter("blocker", event.target.value)}><option value="all">All</option><option value="true">Has blocker</option><option value="false">No blocker</option></select></label>
              <label>Source gap<select aria-label="Source-gap filter" value={filters.sourceGap} onChange={(event) => updateFilter("sourceGap", event.target.value)}><option value="all">All</option><option value="true">Has gap</option><option value="false">No gap</option></select></label>
              <label>External involvement<select aria-label="External-involvement filter" value={filters.externalInvolvement} onChange={(event) => updateFilter("externalInvolvement", event.target.value)}><option value="all">All</option><option value="true">Unresolved</option><option value="false">Resolved</option></select></label>
              <label>Draft disposition<select aria-label="Disposition filter" value={filters.disposition} onChange={(event) => updateFilter("disposition", event.target.value)}><option value="">All</option><option value="INCLUDE">INCLUDE</option><option value="EXCLUDE">EXCLUDE</option><option value="DEFER">DEFER</option><option value="UNSET">UNSET</option></select></label>
            </div>
          </details>
        </section>

        <section aria-label="Sealed classification rows" className="aga-workspace-register">
          <div className="aga-workspace-register__head"><div><span className="aga-workspace-kicker">Step 2 · Compare</span><h2>Question queue</h2><p>Select a row to open its decision file. The full sealed text is fetched only for the active page.</p></div><div className="aga-workspace-pagination"><button disabled={currentPageNumber === 0 || loading} onClick={() => setPage((value) => Math.max(0, value - 1))} type="button">Previous</button><span>Page {currentPageNumber + 1}</span><button disabled={!hasNextPage || loading} onClick={() => setPage((value) => value + 1)} type="button">Next</button></div></div>
          {loading ? <div aria-live="polite" className="aga-workspace-queue-loading"><span /><span /><span /></div> : null}
          <div className="aga-workspace-review-grid">
            <section aria-label="Sealed question queue" className="aga-workspace-queue">
              <div className="aga-workspace-queue__head"><div><strong>{total || 0}</strong><span>matching questions</span></div><span>{firstVisibleRow && lastVisibleRow ? `${firstVisibleRow}–${lastVisibleRow}` : "No rows"}</span></div>
              <div className="aga-workspace-queue__list" role="list">
                {rows.map((row) => <article className="aga-workspace-queue-item" data-selected={selected?.questionKey === row.questionKey ? "true" : undefined} key={row.questionKey} role="listitem">
                  <div className="aga-workspace-queue-item__topline"><button className="aga-workspace-row-link" onClick={() => setSelected(row)} type="button">{row.identity.formCode} <span aria-hidden="true">·</span> {row.identity.ordinal}</button><span className="aga-workspace-status-chip">{row.agreementConfidence} confidence</span></div>
                  <button className="aga-workspace-queue-item__question" onClick={() => setSelected(row)} type="button">{row.questionText ?? "Sealed candidate text is not available in this projection."}</button>
                  <div className="aga-workspace-queue-item__signals"><span>{row.projection.mainDomainCode}</span><span>{(row.projection.topicCodes ?? []).join(" · ") || "No topic"}</span><span>{row.draftDisposition ?? "No Draft decision"}</span></div>
                  <div className="aga-workspace-queue-item__footer"><span>{row.textOrigin === "SEALED_BASE" ? "Sealed text · digest matched" : "Text projection not returned"}</span><button className="aga-workspace-quiet-button" aria-label={`Open controls for ${row.identity.formCode} ${row.identity.ordinal}`} onClick={() => setSelected(row)} type="button">Open controls</button></div>
                  <details className="aga-workspace-technical-details"><summary>Technical identifiers</summary><dl><div><dt>Proposal</dt><dd>{row.identity.proposalId}</dd></div><div><dt>Question key</dt><dd>{row.questionKey}</dd></div><div><dt>Candidate digest</dt><dd>{row.candidateDigest}</dd></div></dl></details>
                </article>)}
              </div>
              {!loading && rows.length === 0 ? <div className="aga-workspace-empty"><strong>No questions match these filters.</strong><p>Clear the filters to return to the sealed queue.</p><button className="aga-workspace-quiet-button" onClick={clearFilters} type="button">Clear filters</button></div> : null}
            </section>

            <aside aria-label="Selected question decision file" className="aga-workspace-dossier">
              {selected ? <>
                <div className="aga-workspace-dossier__head"><div><span className="aga-workspace-kicker">Step 3 · Decide</span><h2>Decision file</h2><p>One server-returned question, one auditable Draft action.</p></div><span className="aga-workspace-selected-chip">Selected</span></div>
                <blockquote className="aga-workspace-question-quote">{selected.questionText ?? "Sealed candidate text is not available in this projection."}</blockquote>
                <dl className="aga-workspace-dossier-facts"><div><dt>Domain</dt><dd>{selected.projection.mainDomainCode}</dd></div><div><dt>Topic</dt><dd>{(selected.projection.topicCodes ?? []).join(" · ") || "No topic"}</dd></div><div><dt>Proposal</dt><dd>{selected.recommendationState}</dd></div><div><dt>Draft state</dt><dd>{selected.draftDisposition ?? "Not set"}</dd></div><div><dt>Confidence</dt><dd>{selected.agreementConfidence}</dd></div><div><dt>Scope</dt><dd>{selected.includeEligible ? "Include eligible" : "Include blocked"}</dd></div></dl>
                <p className="aga-workspace-dossier-note">The server owns the classification signals. Your actions create attributed Draft successors in this local candidate only.</p>
                <details className="aga-workspace-technical-details"><summary>Show governance signals</summary><dl><div><dt>Candidate digest</dt><dd>{selected.candidateDigest}</dd></div><div><dt>Challenge digest</dt><dd>{selected.challengeDigest}</dd></div><div><dt>Source mapping</dt><dd>{selected.governance.sourceMappingState}</dd></div><div><dt>Decision state</dt><dd>{selected.governance.decisionState}</dd></div></dl></details>
                <section aria-label="Draft controls" className="aga-workspace-command-panel">
                  <div><span className="aga-workspace-kicker">Record a Draft decision</span><h3>What should happen to this question?</h3><p id="include-eligibility-reason" className="aga-workspace-boundary">{selected.includeEligible ? "This item is eligible for Include under the current server-owned simulation scope." : `Include unavailable: ${selected.includeEligibilityReason}`}</p></div>
                  <label>Controlled reason<select aria-label="Controlled reason" value={reasonCode} onChange={(event) => setReasonCode(event.target.value as (typeof REASON_OPTIONS)[number])}>{REASON_OPTIONS.map((reason) => <option key={reason} value={reason}>{reason}</option>)}</select></label>
                  <div className="aga-workspace-action-grid"><button disabled={!selected || pending} onClick={() => runAction("RETAIN")} type="button">Retain</button><button className="aga-workspace-action-primary" aria-label="Include selected item" aria-describedby="include-eligibility-reason" disabled={Boolean(includeDisabledReason)} onClick={() => runAction("INCLUDE")} title={includeDisabledReason || undefined} type="button">Include</button><button disabled={!selected || pending} onClick={() => runAction("EXCLUDE")} type="button">Exclude</button><button disabled={!selected || pending} onClick={() => runAction("DEFER")} type="button">Defer</button></div>
                  <div className="aga-workspace-advanced-actions"><button disabled={!selected || pending} onClick={() => runAction("RECLASSIFY_MAIN_DOMAIN", { mainDomainCode })} type="button">Reclassify domain</button><label>New domain<input aria-label="Main domain code" value={mainDomainCode} onChange={(event) => setMainDomainCode(event.target.value)} placeholder="DOMAIN_CODE" /></label><label>Topic code<input aria-label="Topic code" value={topicCode} onChange={(event) => setTopicCode(event.target.value)} placeholder="TOPIC_CODE" /></label><button disabled={!selected || !topicCode || pending} onClick={() => runAction("ADD_TOPIC", { topicCode })} type="button">Add topic</button><button disabled={!selected || !topicCode || pending} onClick={() => runAction("REMOVE_TOPIC", { topicCode })} type="button">Remove topic</button><button disabled={!selected || pending} onClick={() => runAction("RESOLVE_CLASSIFICATION_PROPOSALS", { resolutionMode: "CANDIDATE" })} type="button">Use candidate proposal</button><button disabled={!selected || pending} onClick={() => runAction("RESOLVE_CLASSIFICATION_PROPOSALS", { resolutionMode: "CHALLENGE" })} type="button">Use challenge proposal</button><button disabled={!selected || pending} onClick={() => runAction("RESOLVE_CLASSIFICATION_PROPOSALS", { resolutionMode: "SET_EXACT", exactProjection: selected.projection })} type="button">Set exact proposal</button></div>
                  {role === "manager" ? <Link className="aga-workspace-inspection-link" to="/department-manager/aga-demo-workspace/inspection-package">Next: open inspection package builder</Link> : null}
                  <label className="aga-workspace-successor-field">Candidate or successor wording<textarea aria-label="Candidate or successor wording" value={workspaceBody} onChange={(event) => setWorkspaceBody(event.target.value)} placeholder="Optional: write a controlled successor wording" /></label>
                  <div className="aga-workspace-successor-actions"><button disabled={!selected || !workspaceBody.trim() || pending} onClick={() => void createCandidate()} type="button">Add candidate</button><button disabled={!selected || !workspaceBody.trim() || pending} onClick={() => void rewordCandidate()} type="button">Create immutable wording successor</button></div>
                  <div className="aga-workspace-disabled-actions"><button aria-label="Mark ready unavailable" disabled title="Readiness requires the complete server-pinned provider scope digest and controlled readiness event." type="button">Mark ready unavailable</button><button aria-label="Technical approval unavailable" disabled title="Technical approval is not a classification Draft action in this workspace." type="button">Technical approval unavailable</button><button aria-label="Publication unavailable" disabled title="Publication is not available from this candidate workspace." type="button">Publication unavailable</button></div>
                </section>
              </> : <div className="aga-workspace-dossier-empty"><span className="aga-workspace-kicker">Step 3 · Decide</span><h2>Open a question to begin</h2><p>Choose any row in the queue. Its sealed wording, classification signals, and available Draft actions will appear here.</p><div className="aga-workspace-dossier-empty__hint"><span>1</span><strong>Select a row</strong><span aria-hidden="true">→</span><strong>Read its decision file</strong></div>{role === "manager" ? <Link className="aga-workspace-inspection-link" to="/department-manager/aga-demo-workspace/inspection-package">Open inspection package builder</Link> : null}</div>}
            </aside>
          </div>
        </section>

        <section aria-label="Provider eligibility" className="aga-workspace-panel">
          <div><span className="aga-workspace-kicker">Why scope matters</span><h2>Provider scope eligibility</h2><p>Classification confidence and provider eligibility are separate checks. Only the configured synthetic scope can be inspected in this candidate.</p></div>
          <div className="aga-workspace-provider-summary"><strong>{inspectedScopeEligible.length} inspected-scope eligible</strong><strong>{involvementOnly.length} involvement-only</strong></div>
          <ul>{providers.map((entry) => <li key={entry.providerTypeCode}>{providerLabel(entry)}<small>{entry.reasonCode}</small></li>)}</ul>
        </section>

        {role === "admin" ? <section aria-label="Admin generation controls" className="aga-workspace-admin-panel"><h2>Admin generation history and reset</h2><p>Reset is destructive to this disposable generation only and never writes canonical records.</p>{history?.length ? <ul>{history.map((entry) => <li key={entry.generationId}>{entry.generationId} · revision {entry.revision}</li>)}</ul> : <p>No history loaded.</p>}<label>Reset reason<input aria-label="Reset reason" value={resetReason} onChange={(event) => setResetReason(event.target.value)} /></label><label className="aga-workspace-confirm"><input aria-label="Confirm generation reset" checked={resetConfirmed} onChange={(event) => setResetConfirmed(event.target.checked)} type="checkbox" /> I understand this resets the disposable generation.</label><button disabled={!resetReason.trim() || !resetConfirmed || pending || !capability.resetEnabled} onClick={() => void resetGeneration()} type="button">Reset generation</button></section> : null}
      </main>
    </WorkspaceShell>
  );
}
