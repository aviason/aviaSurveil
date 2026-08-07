import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type {
  AGADemoWorkspaceBatchFilter,
  AGADemoWorkspaceBatchPreview,
  AGADemoWorkspaceCapability,
  AGADemoWorkspaceClassificationReviewItem,
  AGADemoWorkspaceCommand,
  AGADemoWorkspaceDraft,
  AGADemoWorkspaceLifecycleProjection,
  AGADemoWorkspaceQueryResponse,
  AGADemoWorkspaceRecommendationSnapshot,
  AGADemoWorkspaceSimulationSetup,
} from "../../backend/aga-demo-workspace";
import { BackendHttpError } from "../../backend/http-backend";
import type { Role } from "../../backend/backend";
import { CommandError, PageHeader, StatusPill, WorkspaceShell, errorMessage } from "../shared/workspace-shell";

const PAGE_SIZE = 25;
const MAX_BATCH_SIZE = 500;
const INITIAL_BATCH_FILTER: AGADemoWorkspaceBatchFilter = {
  disposition: "UNSET",
};

type PackageStage = "inventory" | "preview" | "release";
type ReviewRow = AGADemoWorkspaceClassificationReviewItem;

function identityValue(row: ReviewRow, key: string): string | number {
  const identity = row.identity as unknown as Record<string, unknown>;
  const value = identity[key];
  return typeof value === "string" || typeof value === "number" ? value : "—";
}

function commandKey(operationId: string): string {
  if (typeof globalThis.crypto?.randomUUID !== "function") {
    throw new Error("This browser cannot create a secure idempotency key for the requested command.");
  }
  return `aga-demo-package-${operationId.toLowerCase()}-${globalThis.crypto.randomUUID()}`;
}

function packageErrorMessage(error: unknown): string {
  return error instanceof BackendHttpError && error.status === 404
    ? "The inspection package inventory is not provisioned in this local AGA workspace."
    : errorMessage(error);
}

function sameBatchFilter(left: AGADemoWorkspaceBatchFilter, right: AGADemoWorkspaceBatchFilter): boolean {
  return left.search === right.search
    && left.domainCode === right.domainCode
    && left.topicCode === right.topicCode
    && left.confidence === right.confidence
    && left.blocker === right.blocker
    && left.sourceGap === right.sourceGap
    && left.externalInvolvement === right.externalInvolvement
    && left.formCode === right.formCode
    && left.disposition === right.disposition;
}

function previewMatchesControls(
  preview: AGADemoWorkspaceBatchPreview | null,
  filter: AGADemoWorkspaceBatchFilter,
  action: "INCLUDE" | "EXCLUDE" | "DEFER",
  reasonCode: string,
): boolean {
  if (!preview) return false;
  return sameBatchFilter(preview.filter, filter)
    && preview.action === action
    && preview.reasonCode === reasonCode;
}

function previewConfirmationReason(
  preview: AGADemoWorkspaceBatchPreview | null,
  controlsMatch: boolean,
): string {
  if (!preview) return "Create a current server preview before confirming a disposition.";
  if (!controlsMatch) return "The visible filter, action, or reason changed after this preview. Create a new preview.";
  if (preview.count === 0) return "The server preview has no matching rows to confirm.";
  if (preview.count > MAX_BATCH_SIZE) return "Narrow the filter; the server will not chunk or silently truncate a batch above 500 rows.";
  if (preview.action === "INCLUDE" && preview.eligibleCount === 0) return "Every matching row is ineligible for Include under the current server-owned scope.";
  if (preview.action === "INCLUDE" && preview.ineligibleCount > 0) return "The server preview includes ineligible rows. Narrow the filter to an entirely eligible Include set before confirming.";
  return "";
}

function currentDraftDispositionCount(draft: AGADemoWorkspaceDraft | null): { include: number; exclude: number; defer: number; unset: number } {
  const counts = { include: 0, exclude: 0, defer: 0, unset: 0 };
  for (const item of draft?.items ?? []) {
    if (item.currentLeaf === false) continue;
    switch (item.draftDisposition) {
      case "INCLUDE": counts.include += 1; break;
      case "EXCLUDE": counts.exclude += 1; break;
      case "DEFER": counts.defer += 1; break;
      default: counts.unset += 1;
    }
  }
  return counts;
}

function previewSummary(preview: AGADemoWorkspaceBatchPreview | null): string {
  if (!preview) return "No server-issued preview has been confirmed.";
  return `${preview.count} exact matches · ${preview.eligibleCount} eligible · ${preview.ineligibleCount} ineligible · expires ${preview.expiresAt}`;
}

function readinessReason(setup: AGADemoWorkspaceSimulationSetup | null): string {
  if (!setup) return "Load the current server-owned simulation setup first.";
  if (setup.readinessState !== "WORKING") return "The current Draft is already ready or is not in a writable WORKING state.";
  if (setup.unsetCount > 0) return `${setup.unsetCount} current Draft leaves still need an explicit INCLUDE, EXCLUDE, or DEFER disposition.`;
  if (setup.includedCount < 1) return "At least one explicitly included question is required for the synthetic demo.";
  if (setup.includedIneligibleCount > 0) return `${setup.includedIneligibleCount} included question(s) are ineligible for the exact provider, target, profile, type, or qualifier scope.`;
  return "";
}

export function AGADemoInspectionPackagePage({
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
  const [stage, setStage] = useState<PackageStage>("inventory");
  const [setup, setSetup] = useState<AGADemoWorkspaceSimulationSetup | null>(null);
  const [draft, setDraft] = useState<AGADemoWorkspaceDraft | null>(null);
  const [inventory, setInventory] = useState<AGADemoWorkspaceQueryResponse | null>(null);
  const [recommendation, setRecommendation] = useState<AGADemoWorkspaceRecommendationSnapshot | null>(null);
  const [releasedInspection, setReleasedInspection] = useState<AGADemoWorkspaceLifecycleProjection | null>(null);
  const [batchFilter, setBatchFilter] = useState<AGADemoWorkspaceBatchFilter>(INITIAL_BATCH_FILTER);
  const [batchAction, setBatchAction] = useState<"INCLUDE" | "EXCLUDE" | "DEFER">("INCLUDE");
  const [reasonCode, setReasonCode] = useState<"MANAGER_SCOPE_DECISION" | "SIMULATION_SOURCE_GAP_OVERRIDE">("MANAGER_SCOPE_DECISION");
  const [preview, setPreview] = useState<AGADemoWorkspaceBatchPreview | null>(null);
  const [inventorySearch, setInventorySearch] = useState("");
  const [inventoryPage, setInventoryPage] = useState(0);
  const [inspectorPin, setInspectorPin] = useState("");
  const [leadPin, setLeadPin] = useState("");
  const [pending, setPending] = useState(false);
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const formFilterWasEdited = useRef(false);

  const draftCounts = useMemo(() => currentDraftDispositionCount(draft), [draft]);

  const refresh = useCallback(async (signal?: AbortSignal) => {
    if (!client || role !== "manager") return;
    setLoading(true);
    setError(null);
    try {
      const [setupResponse, draftResponse, inventoryResponse] = await Promise.all([
        client.classificationQuery({ operationId: "GET_SIMULATION_SETUP" }, { signal }),
        client.classificationQuery({ operationId: "GET_DRAFT" }, { signal }),
        client.classificationQuery({ operationId: "SEARCH_ITEMS", page: inventoryPage, pageSize: PAGE_SIZE, ...(inventorySearch.trim() ? { search: inventorySearch.trim() } : {}) }, { signal }),
      ]);
      setSetup(setupResponse.simulationSetup ?? null);
      setDraft(draftResponse.draft ?? null);
      setInventory(inventoryResponse);
      if (stage === "release" && client.recommendationQuery) {
        try {
          const current = await client.recommendationQuery({ operationId: "GET_CURRENT_RECOMMENDATION" }, { signal });
          setRecommendation(current.recommendationSnapshot ?? null);
        } catch {
          // A failed authoritative refresh is not evidence that an earlier
          // release artifact remains current after a reset or scope change.
          setRecommendation(null);
        }
      } else if (stage !== "release") {
        setRecommendation(null);
      }
      if (stage === "release") {
        try {
          const currentInspection = await client.lifecycleQuery({ operationId: "GET_CURRENT_INSPECTION" }, { signal });
          setReleasedInspection(currentInspection.currentInspection ?? currentInspection.lifecycle ?? null);
        } catch {
          // Do not retain an inspection that the current authoritative read
          // could not bind to this generation and current role.
          setReleasedInspection(null);
        }
      } else {
        setReleasedInspection(null);
      }
    } catch (cause) {
      // The three core state reads are one server-authoritative snapshot. Do
      // not render a mixture of old controls and a failed current refresh.
      setSetup(null);
      setDraft(null);
      setInventory(null);
      setRecommendation(null);
      setReleasedInspection(null);
      throw cause;
    } finally {
      setLoading(false);
    }
  }, [client, inventoryPage, inventorySearch, role, stage]);

  useEffect(() => {
    const controller = new AbortController();
    void refresh(controller.signal).catch((cause: unknown) => {
      if (controller.signal.aborted) return;
      setLoading(false);
      setError(packageErrorMessage(cause));
    });
    return () => controller.abort();
  }, [refresh]);

  useEffect(() => {
    if (formFilterWasEdited.current || batchFilter.formCode || !inventory?.items?.length) return;
    const firstFormCode = identityValue(inventory.items[0]!, "formCode");
    if (typeof firstFormCode !== "string" || firstFormCode === "—") return;
    setBatchFilter((current) => current.formCode ? current : { ...current, formCode: firstFormCode });
  }, [batchFilter.formCode, inventory?.items]);

  const updateBatchFilter = useCallback((field: keyof AGADemoWorkspaceBatchFilter, value: string) => {
    if (field === "formCode") formFilterWasEdited.current = true;
    setBatchFilter((current) => ({ ...current, [field]: value }));
    setPreview(null);
  }, []);

  const updateBatchAction = useCallback((action: "INCLUDE" | "EXCLUDE" | "DEFER") => {
    setBatchAction(action);
    setPreview(null);
  }, []);

  const updateReasonCode = useCallback((nextReasonCode: "MANAGER_SCOPE_DECISION" | "SIMULATION_SOURCE_GAP_OVERRIDE") => {
    setReasonCode(nextReasonCode);
    setPreview(null);
  }, []);

  const issueCommand = useCallback(async (
    operationId: AGADemoWorkspaceCommand["operationId"],
    fields: Partial<AGADemoWorkspaceCommand>,
  ) => {
    if (!client || !setup || !draft) throw new Error("The current server-owned setup and Draft are required.");
    const command: AGADemoWorkspaceCommand = {
      operationId,
      idempotencyKey: commandKey(operationId),
      expectedGenerationId: setup.generationId,
      expectedDraftRevision: draft.revision,
      expectedDraftContentDigest: draft.contentDigest,
      ...fields,
    };
    setPending(true);
    setError(null);
    setStatus(null);
    try {
      const response = operationId === "CREATE_RECOMMENDATION"
        ? await client.recommendationCommand(command)
        : await client.classificationCommand(command);
      if (response.draft) setDraft(response.draft);
      return response;
    } finally {
      setPending(false);
    }
  }, [client, draft, setup]);

  const runPreview = useCallback(async () => {
    if (!setup || !draft) return;
    try {
      const response = await issueCommand("PREVIEW_BATCH", {
        batchFilter,
        batchAction,
        simulationSetupDigest: setup.simulationSetupDigest,
        reasonCode,
      });
      setPreview(response.batchPreview ?? null);
      setStage("preview");
      setStatus(response.batchPreview ? `Server-issued preview returned: ${previewSummary(response.batchPreview)}` : "The server returned no batch preview.");
    } catch (cause) {
      setError(packageErrorMessage(cause));
    }
  }, [batchAction, batchFilter, draft, issueCommand, reasonCode, setup]);

  const executePreview = useCallback(async () => {
    if (!setup || !draft || !preview || previewConfirmationReason(preview, previewMatchesControls(preview, batchFilter, batchAction, reasonCode))) return;
    try {
      await issueCommand("EXECUTE_BATCH", {
        previewId: preview.previewId,
        previewDigest: preview.previewDigest,
        batchFilter: preview.filter,
        batchAction: preview.action,
        simulationSetupDigest: setup.simulationSetupDigest,
        reasonCode: preview.reasonCode,
      });
      setPreview(null);
      setStatus("Confirmed simulation disposition appended as one atomic Draft successor.");
      await refresh();
    } catch (cause) {
      setError(packageErrorMessage(cause));
    }
  }, [batchAction, batchFilter, draft, issueCommand, preview, reasonCode, refresh, setup]);

  const markReady = useCallback(async () => {
    if (!setup || !draft || readinessReason(setup)) return;
    try {
      await issueCommand("MARK_READY_FOR_DEMO_SIMULATION", {
        simulationSetupDigest: setup.simulationSetupDigest,
        reasonCode: setup.includedSourceGapCount > 0 ? "SIMULATION_SOURCE_GAP_OVERRIDE" : reasonCode,
      });
      await refresh();
      setStage("release");
      setStatus("Draft marked ready for demo simulation with a server-issued readiness event.");
    } catch (cause) {
      setError(packageErrorMessage(cause));
    }
  }, [draft, issueCommand, reasonCode, refresh, setup]);

  const createRecommendation = useCallback(async () => {
    if (!setup || !draft || setup.readinessState !== "READY_FOR_DEMO_SIMULATION") return;
    try {
      const response = await issueCommand("CREATE_RECOMMENDATION", {
        draftId: draft.draftId,
        draftRevision: draft.revision,
        draftContentDigest: draft.contentDigest,
        simulationSetupDigest: setup.simulationSetupDigest,
        reasonCode: "MANAGER_SCOPE_DECISION",
      });
      const currentRecommendation = response.recommendation;
      if (currentRecommendation) {
        setRecommendation(currentRecommendation);
        // The command response is server-accepted, so it can immediately pin
        // the next release command while the follow-up read reconciles the
        // wider setup projection.
        setSetup((current) => current ? {
          ...current,
          recommendationId: currentRecommendation.recommendation.recommendationId,
          recommendationRevision: currentRecommendation.recommendation.revision,
          recommendationDigest: currentRecommendation.recommendation.digest,
        } : current);
      } else {
        setRecommendation(null);
      }
      await refresh();
      setStage("release");
      setStatus(response.replayed ? "The existing synthetic recommendation was returned by idempotent replay." : "Synthetic recommendation created from the exact eligible included set.");
    } catch (cause) {
      setError(packageErrorMessage(cause));
    }
  }, [draft, issueCommand, refresh, setup]);

  const releaseInspection = useCallback(async () => {
    if (!setup || !draft || !inspectorPin || !leadPin || !setup.recommendationId) return;
    try {
      // A recommendation is committed through a separate append-only command.
      // Re-read the setup immediately before release so the command carries
      // the server-issued digest that includes that recommendation, rather
      // than a render captured before the reader projection caught up.
      const latestSetupResponse = await client?.classificationQuery({ operationId: "GET_SIMULATION_SETUP" });
      const latestSetup = latestSetupResponse?.simulationSetup;
      if (!latestSetup?.recommendationId || latestSetup.recommendationId !== setup.recommendationId || !latestSetup.simulationSetupDigest) {
        throw new Error("The server-owned recommendation setup is still being reconciled. Retry release after the current setup refresh completes.");
      }
      setSetup(latestSetup);
      const response = await client?.recommendationCommand({
        operationId: "CREATE_INSPECTION",
        idempotencyKey: commandKey("CREATE_INSPECTION"),
        expectedGenerationId: latestSetup.generationId,
        expectedRecommendationRevision: latestSetup.recommendationRevision,
        simulationSetupDigest: latestSetup.simulationSetupDigest,
        inspectorSelectionPin: inspectorPin,
        leadSelectionPin: leadPin,
      });
      if (response?.lifecycle) setReleasedInspection(response.lifecycle);
      await refresh();
      setStage("release");
      setStatus(response?.replayed ? "The existing synthetic inspection was returned by idempotent replay." : "Synthetic inspection released with immutable question references.");
    } catch (cause) {
      setError(packageErrorMessage(cause));
    }
  }, [client, draft, inspectorPin, leadPin, refresh, setup]);

  const rows = inventory?.items ?? [];
  const hasNext = inventory?.nextPage !== undefined;
  const currentSetup = setup;
  const readyReason = readinessReason(currentSetup);
  const controlsMatchPreview = previewMatchesControls(preview, batchFilter, batchAction, reasonCode);
  const confirmPreviewReason = previewConfirmationReason(preview, controlsMatchPreview);

  if (role !== "manager" || !capability.classificationEnabled || !client) {
    return (
      <WorkspaceShell roleLabel={roleLabel} routeLabel="Inspection Package Builder">
        <main className="aga-package-page" data-testid="aga-demo-inspection-package-page">
          <PageHeader eyebrow="Candidate-only local-preprod package" title="Inspection package builder unavailable" description="This fixed route is reserved for the exactly bound Department Manager projection." />
          <p className="aga-package-boundary" role="alert">The server did not declare the Manager package capability for this session.</p>
        </main>
      </WorkspaceShell>
    );
  }

  return (
    <WorkspaceShell roleLabel={roleLabel} routeLabel="Inspection Package Builder">
      <main className="aga-package-page" data-testid="aga-demo-inspection-package-page">
        <PageHeader
          eyebrow="Interactive local-preprod · candidate-only"
          title="AGA inspection package builder"
          description="Build one synthetic inspection from explicit simulation dispositions. This package never approves, publishes, or mutates canonical Audit Plan, Checklist, Finding, CAP, or Evidence stores."
          action={<Link className="aga-workspace-button" to="/department-manager/aga-demo-workspace">Back to inventory</Link>}
        />
        <CommandError message={error} />
        {status ? <p className="aga-workspace-status" role="status">{status}</p> : null}

        <nav aria-label="Inspection package stages" className="aga-package-stage-nav">
          {(["inventory", "preview", "release"] as const).map((candidate) => <button aria-current={stage === candidate ? "step" : undefined} className={stage === candidate ? "active" : ""} key={candidate} onClick={() => setStage(candidate)} type="button">{candidate === "inventory" ? "1 · Inventory and disposition" : candidate === "preview" ? "2 · Package preview" : "3 · Simulation release"}</button>)}
        </nav>

        <section aria-label="Simulation truth and Draft counts" className="aga-package-facts">
          <article><span>Sealed candidate inventory</span><strong>{inventory?.itemCount ?? 0}</strong><small>1,310 candidate AGA questions; bounded pages only</small></article>
          <article><span>Current Draft leaves</span><strong>{currentSetup?.currentLeafCount ?? draft?.items?.length ?? 0}</strong><small>explicit disposition required for every current leaf</small></article>
          <article><span>Included / excluded / deferred</span><strong>{currentSetup?.includedCount ?? draftCounts.include} / {currentSetup?.excludedCount ?? draftCounts.exclude} / {currentSetup?.deferredCount ?? draftCounts.defer}</strong><small>simulation disposition, not approval</small></article>
          <article><span>Release state</span><strong>{currentSetup?.readinessState ?? "Loading"}</strong><small>candidate-only · release pending</small></article>
        </section>

        {stage === "inventory" ? <section aria-label="Bounded candidate inventory" className="aga-package-panel">
          <div className="aga-package-panel__heading"><div><h2>Reach the complete sealed inventory</h2><p>Each request contains at most 25 original bodies. Search, page, and text remain transient in this component; no URL or browser storage state is used.</p></div><div className="aga-workspace-pagination"><button disabled={inventoryPage === 0 || loading} onClick={() => setInventoryPage((value) => Math.max(0, value - 1))} type="button">Previous page</button><span>Page {(inventory?.page ?? inventoryPage) + 1}</span><button disabled={!hasNext || loading} onClick={() => setInventoryPage((value) => value + 1)} type="button">Next page</button></div></div>
          <label>Search sealed candidate text or identity<input aria-label="Package inventory search" value={inventorySearch} onChange={(event) => { setInventorySearch(event.target.value); setInventoryPage(0); }} /></label>
          {loading ? <p role="status">Loading current server page…</p> : null}
          <div className="aga-package-table-wrap"><table><caption className="sr-only">Bounded sealed AGA candidate inventory</caption><thead><tr><th scope="col">Form / ordinal</th><th scope="col">Sealed candidate text</th><th scope="col">Classification and governance</th><th scope="col">Disposition</th></tr></thead><tbody>{rows.map((row) => <tr key={row.questionKey}><th scope="row">{identityValue(row, "formCode")} · {identityValue(row, "ordinal")}<small>{identityValue(row, "proposalId")}</small></th><td><p>{row.questionText ?? "Text is not available in this projection."}</p><small>{row.questionTextDigest ? "digest-matched sealed text" : "text-free projection"}</small></td><td><strong>{row.projection.mainDomainCode ?? "Domain unavailable"}</strong><small>{(row.projection.topicCodes ?? []).join(" · ") || "No topic"}</small><small>{row.governance?.questionSourceProposalGap ? "source mapping required" : "source gap not present in projection"}</small></td><td><StatusPill>{row.draftDisposition ?? "UNSET"}</StatusPill></td></tr>)}</tbody></table></div>
          {!loading && rows.length === 0 ? <p className="aga-package-boundary">No sealed candidate rows match this transient search.</p> : null}
        </section> : null}

        {stage === "preview" ? <section aria-label="Exact server batch preview" className="aga-package-panel">
          <h2>Exact server batch preview</h2>
          <p>Batch actions operate on the complete server filter result, never the visible 25-row page. A preview is identity/digest-only, expires, and must be confirmed before one atomic Draft append.</p>
          <div className="aga-package-filter-grid">
            <label>Question text search<input aria-label="Batch search" value={batchFilter.search ?? ""} onChange={(event) => updateBatchFilter("search", event.target.value)} /></label>
            <label>Form filter<input aria-label="Batch form filter" value={batchFilter.formCode ?? ""} onChange={(event) => updateBatchFilter("formCode", event.target.value)} /></label>
            <label>Domain filter<input aria-label="Batch domain filter" value={batchFilter.domainCode ?? ""} onChange={(event) => updateBatchFilter("domainCode", event.target.value)} /></label>
            <label>Topic filter<input aria-label="Batch topic filter" value={batchFilter.topicCode ?? ""} onChange={(event) => updateBatchFilter("topicCode", event.target.value)} /></label>
            <label>Current disposition<select aria-label="Batch disposition filter" value={batchFilter.disposition ?? "UNSET"} onChange={(event) => updateBatchFilter("disposition", event.target.value)}><option value="UNSET">UNSET</option><option value="INCLUDE">INCLUDE</option><option value="EXCLUDE">EXCLUDE</option><option value="DEFER">DEFER</option></select></label>
            <label>Action<select aria-label="Batch action" value={batchAction} onChange={(event) => updateBatchAction(event.target.value as typeof batchAction)}><option value="INCLUDE">INCLUDE</option><option value="EXCLUDE">EXCLUDE</option><option value="DEFER">DEFER</option></select></label>
            <label>Controlled reason<select aria-label="Batch reason" value={reasonCode} onChange={(event) => updateReasonCode(event.target.value as typeof reasonCode)}><option value="MANAGER_SCOPE_DECISION">MANAGER_SCOPE_DECISION</option><option value="SIMULATION_SOURCE_GAP_OVERRIDE">SIMULATION_SOURCE_GAP_OVERRIDE</option></select></label>
          </div>
          <div className="aga-package-actions"><button disabled={pending || loading || !setup || !draft} onClick={() => void runPreview()} type="button">Create server preview</button><button disabled={pending || loading || Boolean(confirmPreviewReason)} onClick={() => void executePreview()} title={confirmPreviewReason || undefined} type="button">Confirm simulation disposition</button></div>
          <p className="aga-package-boundary">{previewSummary(preview)} {confirmPreviewReason && preview ? confirmPreviewReason : ""}</p>
          {preview ? <dl className="aga-package-preview-grid"><div><dt>Preview ID</dt><dd>{preview.previewId}</dd></div><div><dt>Preview digest</dt><dd>{preview.previewDigest}</dd></div><div><dt>Frozen command</dt><dd>{preview.action} · {preview.reasonCode} · {preview.filter.formCode || "all forms"}</dd></div><div><dt>Current disposition</dt><dd>{preview.currentDisposition.include} include · {preview.currentDisposition.exclude} exclude · {preview.currentDisposition.defer} defer · {preview.currentDisposition.unset} unset</dd></div><div><dt>Governance signals</dt><dd>{preview.blockerCount} blockers · {preview.sourceGapCount} source gaps</dd></div></dl> : null}
        </section> : null}

        {stage === "release" ? <section aria-label="Synthetic simulation release" className="aga-package-panel">
          <h2>Package preview and simulation release</h2>
          <p>All release controls use the current setup digest. The server re-resolves scope, target, Draft, readiness, recommendation, and role bindings at commit time.</p>
          <dl className="aga-package-preview-grid"><div><dt>Provider / target</dt><dd>{setup?.providerLabel ?? "—"} · {setup?.targetLabel ?? "—"}</dd></div><div><dt>Scope/profile</dt><dd>{setup?.providerScopeId ?? "—"} · {setup?.inspectionProfileCode ?? "—"} · {setup?.inspectionTypeCode ?? "—"}</dd></div><div><dt>Eligibility</dt><dd>{setup?.includedEligibleCount ?? 0} eligible included · {setup?.includedIneligibleCount ?? 0} included but ineligible</dd></div><div><dt>Governance</dt><dd>{setup?.includedBlockerCount ?? 0} included blockers · {setup?.includedSourceGapCount ?? 0} included source gaps</dd></div></dl>
          <div className="aga-package-boundary"><strong>Required invariant</strong><p>{readyReason || "The current Draft satisfies the visible readiness preconditions."}</p><p>CAP acceptance is not Finding closure. Normal closure requires accepted Evidence and assigned Inspector/Lead VERIFY_EVIDENCE with CLOSE.</p></div>
          <div className="aga-package-actions"><button aria-describedby="ready-reason" disabled={pending || Boolean(readyReason)} onClick={() => void markReady()} type="button">Mark ready for demo simulation</button><span className="sr-only" id="ready-reason">{readyReason || "Ready for server validation."}</span><button disabled={pending || setup?.readinessState !== "READY_FOR_DEMO_SIMULATION" || Boolean(recommendation)} onClick={() => void createRecommendation()} type="button">Create synthetic recommendation</button></div>
          <p className="aga-package-boundary">Readiness and recommendation are synthetic local-preprod artifacts; neither is a technical approval or publication.</p>
          {setup?.inspectorChoices?.length ? <fieldset><legend>Explicit Inspector selection</legend><select aria-label="Inspector selection" value={inspectorPin} onChange={(event) => setInspectorPin(event.target.value)}><option value="">Choose one server-returned Inspector</option>{setup.inspectorChoices.map((choice) => <option key={choice.selectionPin} value={choice.selectionPin}>{choice.label}</option>)}</select></fieldset> : null}
          {setup?.leadChoices?.length ? <fieldset><legend>Explicit Lead Inspector selection</legend><select aria-label="Lead Inspector selection" value={leadPin} onChange={(event) => setLeadPin(event.target.value)}><option value="">Choose one server-returned Lead Inspector</option>{setup.leadChoices.map((choice) => <option key={choice.selectionPin} value={choice.selectionPin}>{choice.label}</option>)}</select></fieldset> : null}
          <button disabled={pending || !recommendation || !inspectorPin || !leadPin || !setup?.simulationSetupDigest} onClick={() => void releaseInspection()} type="button">Release synthetic inspection</button>
          {recommendation ? <p role="status">Synthetic recommendation is current: revision {recommendation.recommendation.revision}, digest {recommendation.snapshotDigest}.</p> : null}
          {releasedInspection ? <section aria-label="Released synthetic inspection" className="aga-package-released"><h3>Released immutable inspection snapshot</h3><p>{releasedInspection.questions.length} immutable question reference(s) · revision {releasedInspection.revision} · {releasedInspection.digest}</p><nav aria-label="Synthetic role handoff" className="aga-package-handoff"><Link to="/department-manager/aga-demo-workspace/inspection">Manager snapshot</Link><Link to="/inspector/aga-demo-workspace/inspection">Inspector handoff</Link><Link to="/lead-inspector/aga-demo-workspace/inspection">Lead Inspector handoff</Link><Link to="/caa-reviewer/aga-demo-workspace/caps-evidence">CAA Reviewer CAP review</Link><Link to="/auditee/aga-demo-workspace/caps-evidence">Auditee public CAP and Evidence</Link></nav></section> : null}
        </section> : null}

        <section aria-label="Package governance labels" className="aga-package-boundary"><strong>Release labels</strong><p>1,310 candidate AGA questions · sealed candidate text · simulation disposition · synthetic recommendation · synthetic inspection · candidate-only · release pending · production-ready: not established.</p></section>
      </main>
    </WorkspaceShell>
  );
}
