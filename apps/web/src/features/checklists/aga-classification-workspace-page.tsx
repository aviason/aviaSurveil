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
        setError(errorMessage(cause));
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
          setError(errorMessage(cause));
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
          eyebrow="Disposable local-preprod workspace"
          title="AGA classification review"
          description="Review the sealed classification projection and create attributed Draft successors. Base package content remains outside browser persistence and telemetry."
          action={role === "admin" ? <button className="aga-workspace-button" onClick={showHistory} type="button">View generation history</button> : undefined}
        />
        <CommandError message={error} />
        {status ? <p className="aga-workspace-status" role="status">{status}</p> : null}

        <section aria-label="Workspace generation and counts" className="aga-workspace-facts">
          <article><span>Current generation</span><strong>{generation?.generationId ?? "Not loaded"}</strong><small>sealed revision {generation?.revision ?? "—"}</small></article>
          <article><span>Base inventory</span><strong>{summary?.baseQuestionCount ?? 0}</strong><small>server-sealed rows</small></article>
          <article><span>Draft items</span><strong>{currentDraftItemCount(draft)}</strong><small>effective Draft projection</small></article>
          <article><span>Filtered rows</span><strong>{total}</strong><small>page {currentPageNumber + 1}</small></article>
        </section>

        <section aria-label="Server filters" className="aga-workspace-filter-panel">
          <h2>Classification inventory</h2>
          <p>Search and every filter are sent in the POST body. The browser retains at most four fetched pages.</p>
          <div className="aga-workspace-filter-grid">
            <label>Search code or digest<input aria-label="AGA search" value={filters.search} onChange={(event) => updateFilter("search", event.target.value)} /></label>
            <label>Domain<input aria-label="Domain filter" value={filters.domainCode} onChange={(event) => updateFilter("domainCode", event.target.value)} /></label>
            <label>Topic<input aria-label="Topic filter" value={filters.topicCode} onChange={(event) => updateFilter("topicCode", event.target.value)} /></label>
            <label>Form<input aria-label="Form filter" value={filters.formCode} onChange={(event) => updateFilter("formCode", event.target.value)} /></label>
            <label>Sealed confidence<select aria-label="Confidence filter" value={filters.confidence} onChange={(event) => updateFilter("confidence", event.target.value)}><option value="">All</option><option value="HIGH">HIGH</option><option value="MEDIUM">MEDIUM</option><option value="LOW">LOW</option></select></label>
            <label>Blocker<select aria-label="Blocker filter" value={filters.blocker} onChange={(event) => updateFilter("blocker", event.target.value)}><option value="all">All</option><option value="true">Has blocker</option><option value="false">No blocker</option></select></label>
            <label>Source gap<select aria-label="Source-gap filter" value={filters.sourceGap} onChange={(event) => updateFilter("sourceGap", event.target.value)}><option value="all">All</option><option value="true">Has gap</option><option value="false">No gap</option></select></label>
            <label>External involvement<select aria-label="External-involvement filter" value={filters.externalInvolvement} onChange={(event) => updateFilter("externalInvolvement", event.target.value)}><option value="all">All</option><option value="true">Unresolved</option><option value="false">Resolved</option></select></label>
            <label>Draft disposition<select aria-label="Disposition filter" value={filters.disposition} onChange={(event) => updateFilter("disposition", event.target.value)}><option value="">All</option><option value="INCLUDE">INCLUDE</option><option value="EXCLUDE">EXCLUDE</option><option value="DEFER">DEFER</option><option value="UNSET">UNSET</option></select></label>
          </div>
        </section>

        <section aria-label="Sealed classification rows" className="aga-workspace-register">
          <div className="aga-workspace-register__head"><div><h2>Sealed base classification</h2><p>Authorized roles receive only the active 25-row sealed text page; the four-page cache stores metadata without bodies.</p></div><div className="aga-workspace-pagination"><button disabled={currentPageNumber === 0 || loading} onClick={() => setPage((value) => Math.max(0, value - 1))} type="button">Previous page</button><span>Page {currentPageNumber + 1}</span><button disabled={!hasNextPage || loading} onClick={() => setPage((value) => value + 1)} type="button">Next page</button></div></div>
          {loading ? <p role="status">Loading server page…</p> : null}
          <div className="aga-workspace-table-wrap"><table><caption className="sr-only">Sealed AGA classification page</caption><thead><tr><th scope="col">Server reference</th><th scope="col">Sealed candidate text</th><th scope="col">Classification</th><th scope="col">Confidence / provenance</th><th scope="col">Draft-effective state</th><th scope="col">Review</th></tr></thead><tbody>{rows.map((row) => <tr data-selected={selected?.questionKey === row.questionKey ? "true" : undefined} key={row.questionKey}><th scope="row"><button className="aga-workspace-row-link" onClick={() => setSelected(row)} type="button">{row.identity.formCode} · {row.identity.ordinal}</button><small>{row.identity.proposalId} · {row.questionKey}</small></th><td><p className="aga-workspace-question-text">{row.questionText ?? "Sealed candidate text is not available in this projection."}</p><small>{row.textOrigin === "SEALED_BASE" ? "sealed candidate text · digest matched" : "text projection not returned"}</small></td><td><strong>{row.projection.mainDomainCode}</strong><small>{(row.projection.topicCodes ?? []).join(" · ") || "No topic"}</small><small>{row.recommendationState}</small></td><td><strong>{row.agreementConfidence}</strong><small>candidate {row.candidateDigest}</small><small>challenge {row.challengeDigest}</small></td><td><strong>{row.draftAgreementConfidence ?? "Nullable / not set"}</strong><small>{row.draftReviewState || "Review state not set"}</small><small>{row.draftDisposition ?? "Disposition not set"}</small></td><td><button onClick={() => setSelected(row)} type="button">Open controls</button></td></tr>)}</tbody></table></div>
          {!loading && rows.length === 0 ? <p className="aga-workspace-empty">No sealed rows match the server filters.</p> : null}
        </section>

        <section aria-label="Provider eligibility" className="aga-workspace-panel">
          <h2>Provider scope eligibility</h2>
          <p>Connected provider configuration is sealed separately from classification confidence. Only the configured synthetic scope can be inspected.</p>
          <div className="aga-workspace-provider-summary"><strong>{inspectedScopeEligible.length} inspected-scope eligible</strong><strong>{involvementOnly.length} involvement-only</strong></div>
          <ul>{providers.map((entry) => <li key={entry.providerTypeCode}>{providerLabel(entry)}<small>{entry.reasonCode}</small></li>)}</ul>
        </section>

        <section aria-label="Draft controls" className="aga-workspace-command-panel">
          <h2>Draft controls</h2>
          <p>Every action uses the current generation, Draft revision, content digest, and a server-returned question reference.</p>
          <p id="include-eligibility-reason" className="aga-workspace-boundary">{selected ? selected.includeEligible ? "Selected item is eligible for Include under the current server-owned simulation scope." : `Include unavailable: ${selected.includeEligibilityReason}` : "Select a server-returned item to receive its server-owned Include eligibility."}</p>
          <label>Controlled reason<select aria-label="Controlled reason" value={reasonCode} onChange={(event) => setReasonCode(event.target.value as (typeof REASON_OPTIONS)[number])}>{REASON_OPTIONS.map((reason) => <option key={reason} value={reason}>{reason}</option>)}</select></label>
          <div className="aga-workspace-actions">
            <button disabled={!selected || pending} onClick={() => runAction("RETAIN")} type="button">Retain selected item</button>
            <button aria-describedby="include-eligibility-reason" disabled={Boolean(includeDisabledReason)} onClick={() => runAction("INCLUDE")} title={includeDisabledReason || undefined} type="button">Include selected item</button>
            <button disabled={!selected || pending} onClick={() => runAction("EXCLUDE")} type="button">Exclude selected item</button>
            <button disabled={!selected || pending} onClick={() => runAction("DEFER")} type="button">Defer selected item</button>
            <button disabled={!selected || pending} onClick={() => runAction("RECLASSIFY_MAIN_DOMAIN", { mainDomainCode })} type="button">Reclassify main domain</button>
            <button disabled={!selected || !topicCode || pending} onClick={() => runAction("ADD_TOPIC", { topicCode })} type="button">Add topic</button>
            <button disabled={!selected || !topicCode || pending} onClick={() => runAction("REMOVE_TOPIC", { topicCode })} type="button">Remove topic</button>
            <button disabled={!selected || pending} onClick={() => runAction("RESOLVE_CLASSIFICATION_PROPOSALS", { resolutionMode: "CANDIDATE" })} type="button">Use candidate proposal</button>
            <button disabled={!selected || pending} onClick={() => runAction("RESOLVE_CLASSIFICATION_PROPOSALS", { resolutionMode: "CHALLENGE" })} type="button">Use challenge proposal</button>
            <button disabled={!selected || pending} onClick={() => runAction("RESOLVE_CLASSIFICATION_PROPOSALS", { resolutionMode: "SET_EXACT", exactProjection: selected?.projection })} type="button">Set exact proposal</button>
            {role === "manager" ? <Link className="aga-workspace-button" to="/department-manager/aga-demo-workspace/inspection-package">Open inspection package builder</Link> : null}
          </div>
          <label>Candidate or successor wording<textarea aria-label="Candidate or successor wording" value={workspaceBody} onChange={(event) => setWorkspaceBody(event.target.value)} /></label>
          <div className="aga-workspace-actions"><button disabled={!selected || !workspaceBody.trim() || pending} onClick={() => void createCandidate()} type="button">Add candidate</button><button disabled={!selected || !workspaceBody.trim() || pending} onClick={() => void rewordCandidate()} type="button">Create immutable wording successor</button><button aria-label="Mark ready unavailable" disabled title="Readiness requires the complete server-pinned provider scope digest and controlled readiness event." type="button">Mark ready unavailable</button></div>
          <div className="aga-workspace-disabled-actions"><button aria-label="Technical approval unavailable" disabled title="Technical approval is not a classification Draft action in this workspace.">Technical approval unavailable</button><button aria-label="Publication unavailable" disabled title="Publication is not available from this candidate workspace.">Publication unavailable</button></div>
        </section>

        {role === "admin" ? <section aria-label="Admin generation controls" className="aga-workspace-admin-panel"><h2>Admin generation history and reset</h2><p>Reset is destructive to this disposable generation only and never writes canonical records.</p>{history?.length ? <ul>{history.map((entry) => <li key={entry.generationId}>{entry.generationId} · revision {entry.revision}</li>)}</ul> : <p>No history loaded.</p>}<label>Reset reason<input aria-label="Reset reason" value={resetReason} onChange={(event) => setResetReason(event.target.value)} /></label><label className="aga-workspace-confirm"><input aria-label="Confirm generation reset" checked={resetConfirmed} onChange={(event) => setResetConfirmed(event.target.checked)} type="checkbox" /> I understand this resets the disposable generation.</label><button disabled={!resetReason.trim() || !resetConfirmed || pending || !capability.resetEnabled} onClick={() => void resetGeneration()} type="button">Reset generation</button></section> : null}
      </main>
    </WorkspaceShell>
  );
}
