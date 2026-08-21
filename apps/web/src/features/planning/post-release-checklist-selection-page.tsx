import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useBackendForRole } from "../../app/providers";
import type {
  CanonicalApplicationType,
  CanonicalQuestionCatalogEntry,
  CanonicalQuestionCatalogPage,
  PlanningAuditPackageSetupView,
  PlanningItemView,
} from "../../backend/backend";
import { CommandError, errorMessage, formatLocalDate, WorkspaceShell } from "../shared/workspace-shell";

function operationId(prefix: string, planningItemId: string): string {
  return `${prefix}-${planningItemId}-${Date.now()}`;
}

function sameIds(left: readonly string[], right: readonly string[]): boolean {
  if (left.length !== right.length) return false;
  const rightSet = new Set(right);
  return left.every((id) => rightSet.has(id));
}

function questionLabel(question: CanonicalQuestionCatalogEntry): string {
  return question.prompt ?? `${question.formCode} item ${question.ordinal}`;
}

/**
 * The existing governed catalog selector is intentionally bounded to this
 * post-release route. New Audit planning never mounts it; exact identities
 * and historical recommendations only become available after RELEASED.
 */
export function PostReleaseChecklistSelectionPage() {
  const backend = useBackendForRole("manager");
  const navigate = useNavigate();
  const { planningItemId } = useParams<{ planningItemId: string }>();
  const proposal = backend.planningProposal;
  const catalog = backend.canonicalCatalog;
  const [item, setItem] = useState<PlanningItemView | null>(null);
  const [setup, setSetup] = useState<PlanningAuditPackageSetupView | null>(null);
  const [catalogPage, setCatalogPage] = useState<CanonicalQuestionCatalogPage | null>(null);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [committedIds, setCommittedIds] = useState<string[]>([]);
  const [query, setQuery] = useState("");
  const [formCode, setFormCode] = useState("");
  const [cursor, setCursor] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const selectionHydratedRef = useRef(false);

  useEffect(() => {
    let cancelled = false;
    if (!planningItemId) {
      setError("A released Planning item is required.");
      return () => { cancelled = true; };
    }
    void backend.planning.list({ limit: 100 }).then((output) => {
      if (cancelled) return;
      const selected = output.items.find((candidate) => candidate.id === planningItemId);
      if (!selected) throw new Error("The requested Planning item was not found.");
      setItem(selected);
    }).catch((cause) => {
      if (!cancelled) setError(errorMessage(cause));
    });
    return () => { cancelled = true; };
  }, [backend, planningItemId]);

  async function loadCatalog(nextCursor?: string, nextSetup = setup): Promise<void> {
    if (!catalog || !nextSetup) return;
    setLoading(true);
    setError(null);
    try {
      const [page, selectedPage] = await Promise.all([
        catalog.listCatalog({
          catalogVersion: nextSetup.catalogVersion,
          usageClass: "GOVERNED_OPERATIONAL",
          search: query.trim() || undefined,
          formCode: formCode.trim() || undefined,
          scopeId: nextSetup.scopeDraftId,
          applicationType: item?.inspectionType as CanonicalApplicationType,
          selected: "all",
          projection: "full",
          cursor: nextCursor ?? undefined,
          limit: 50,
        }),
        nextCursor ? Promise.resolve(null) : catalog.listCatalog({
          catalogVersion: nextSetup.catalogVersion,
          usageClass: "GOVERNED_OPERATIONAL",
          scopeId: nextSetup.scopeDraftId,
          selected: "selected",
          projection: "selection",
          limit: 500,
        }),
      ]);
      setCatalogPage((current) => nextCursor ? { ...page, items: [...(current?.items ?? []), ...page.items] } : page);
      setCursor(page.nextCursor);
      if (selectedPage && !selectionHydratedRef.current) {
        const ids = selectedPage.items.map((question) => question.questionVersionId);
        setCommittedIds(ids);
        setSelectedIds(ids);
        selectionHydratedRef.current = true;
      }
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (!item || item.status !== "RELEASED" || !proposal) return;
    let cancelled = false;
    void proposal.ensureAuditPackageSetup({
      operationId: operationId("AUDIT-PACKAGE-SETUP", item.id),
      idempotencyKey: operationId("AUDIT-PACKAGE-SETUP", item.id),
      planningItemId: item.id,
      expectedPlanningRevision: item.revision,
    }).then((nextSetup) => {
      if (cancelled) return;
      setSetup(nextSetup);
      return loadCatalog(undefined, nextSetup);
    }).catch((cause) => {
      if (!cancelled) setError(errorMessage(cause));
    });
    return () => { cancelled = true; };
    // The setup is ensured once for a released item. Search/filter changes
    // intentionally use the explicit reload control below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [item, proposal]);

  const dirty = useMemo(() => !sameIds(selectedIds, committedIds), [committedIds, selectedIds]);
  const released = item?.status === "RELEASED";
  const finalized = setup?.status === "FINALIZED";

  function toggle(question: CanonicalQuestionCatalogEntry): void {
    if (busy || finalized || !question.canSelect) return;
    setSelectedIds((current) => current.includes(question.questionVersionId)
      ? current.filter((id) => id !== question.questionVersionId)
      : [...current, question.questionVersionId]);
    setStatus(null);
  }

  async function confirmSelection(): Promise<void> {
    if (!catalog || !setup || !dirty) return;
    setBusy(true);
    setError(null);
    try {
      const previewOperationId = operationId("AUDIT-PACKAGE-SELECTION-PREVIEW", setup.planningItemId);
      const commitOperationId = operationId("AUDIT-PACKAGE-SELECTION", setup.planningItemId);
      const preview = await catalog.previewSelection({
        scopeId: setup.scopeDraftId,
        operationId: previewOperationId,
        idempotencyKey: previewOperationId,
        expectedSelectionDigest: setup.selectionDigest,
        questionVersionIds: selectedIds,
        operationKind: "REPLACE",
        usageClass: "GOVERNED_OPERATIONAL",
        filter: {},
      });
      if (!preview.valid) throw new Error(preview.reason || "The server rejected this selection.");
      const receipt = await catalog.commitSelection({
        scopeId: setup.scopeDraftId,
        operationId: commitOperationId,
        previewOperationId,
        idempotencyKey: commitOperationId,
        expectedSelectionDigest: setup.selectionDigest,
        questionVersionIds: selectedIds,
        operationKind: "REPLACE",
        usageClass: "GOVERNED_OPERATIONAL",
        filter: {},
      });
      const next = { ...setup, selectedCount: receipt.selection.selectedCount, selectionDigest: receipt.selection.selectionDigest };
      setSetup(next);
      setCommittedIds([...receipt.selection.selectedQuestionVersionIds]);
      setSelectedIds([...receipt.selection.selectedQuestionVersionIds]);
      setStatus("Selection confirmed by the server. Review the approved ceiling before finalization.");
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  async function finalize(): Promise<void> {
    if (!proposal || !item || !setup || dirty || finalized) return;
    setBusy(true);
    setError(null);
    try {
      const next = await proposal.finalizeAuditPackage({
        operationId: operationId("AUDIT-PACKAGE-FINALIZE", item.id),
        idempotencyKey: operationId("AUDIT-PACKAGE-FINALIZE", item.id),
        planningItemId: item.id,
        expectedPlanningRevision: item.revision,
        expectedSetupRevision: setup.revision,
        expectedSelectionDigest: setup.selectionDigest,
      });
      setSetup(next);
      setStatus("Audit-package scope finalized. Department Manager preparation can continue from the Planning workspace.");
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  return <WorkspaceShell roleLabel="Department Manager" routeLabel="Post-release Checklist Preparation">
    <div className="management-workspace planning-workspace post-release-checklist-page" data-testid="post-release-checklist-page">
      <header className="management-page-head workbench-page-header">
        <span>Post-release Audit preparation</span>
        <h1>Checklist selection</h1>
        <p>Review the governed catalog and confirm the exact Audit-package scope after Planning release. These identities are never part of New Audit approval.</p>
      </header>
      <CommandError message={error} />
      {item ? <>
        <section className="management-panel planning-post-release-context" aria-label="Released Planning context">
          <header><div><span>Released Planning context</span><h2>{item.title}</h2><p>{item.organizationName} · {formatLocalDate(item.scheduledDate)}</p></div><span className="planning-demo-badge is-info">{item.status.replaceAll("_", " ")}</span></header>
          {!released ? <p role="alert">Checklist selection is unavailable until this Planning item reaches RELEASED. Current owner: {item.currentOwnerRole}; next action: {item.nextAction}.</p> : setup ? <dl className="planning-post-release-facts"><div><dt>Approved checklist-item ceiling</dt><dd>{setup.approvedChecklistItemCeiling}</dd></div><div><dt>Selected</dt><dd>{selectedIds.length}</dd></div><div><dt>Package state</dt><dd>{setup.status.replaceAll("_", " ")}</dd></div><div><dt>Catalog root</dt><dd>{setup.catalogRootDigest}</dd></div></dl> : <p role="status">Ensuring the post-release Audit-package setup…</p>}
          {status ? <p className="planning-inline-status" role="status">{status}</p> : null}
          <button type="button" className="planning-intake-secondary" onClick={() => navigate("/department-manager/audit-plan")}>Return to Planning</button>
        </section>
        {released && setup ? <section className="management-panel planning-post-release-selector" aria-label="Governed checklist selector">
          <header><div><span>Current governed catalog</span><h2>Checklist items</h2><p>Recommendations are advisory input. The Department Manager confirms the immutable selection explicitly.</p></div><span aria-live="polite" className="planning-intake-catalog-count">{selectedIds.length} selected</span></header>
          {catalogPage?.recommendationSummary ? <aside className="planning-intake-recommendation-summary" aria-label="Prior-Audit recommendation summary"><header><div><span className="planning-intake-dialog-kicker">Prior-Audit history</span><h3>{catalogPage.recommendationSummary.comparableAuditCount ? `${catalogPage.recommendationSummary.comparableAuditCount} comparable prior Audits` : "No comparable prior Audits"}</h3></div><span>{catalogPage.recommendationSummary.historyDeferredCount} withheld by history</span></header><p>Historical recommendations remain a review signal. Location differences do not remove otherwise comparable history.</p></aside> : null}
          <div className="planning-post-release-filters"><label>Search<input aria-label="Search checklist items" value={query} onChange={(event) => setQuery(event.target.value)} /></label><label>Form code<input aria-label="Filter checklist form" value={formCode} onChange={(event) => setFormCode(event.target.value)} /></label><button type="button" disabled={loading} onClick={() => void loadCatalog()}>Apply filters</button></div>
          {catalogPage ? <ul className="planning-intake-catalog-list" aria-label="Governed checklist items">{catalogPage.items.map((question) => <li key={question.questionVersionId}><label><input aria-label={`Select ${question.formCode} item ${question.ordinal}`} type="checkbox" checked={selectedIds.includes(question.questionVersionId)} disabled={busy || finalized || !question.canSelect} onChange={() => toggle(question)} /><span><b className="planning-intake-question-prompt">{questionLabel(question)}</b><small className="planning-intake-question-info">{question.formCode} · item {question.ordinal} · {question.recommendation.recommendationState.replaceAll("_", " ")} · {question.recommendation.historyCount} prior Audits</small></span></label><details><summary>View details</summary><p>{question.expectedEvidence ?? "Expected evidence is server-defined for this governed item."}</p></details></li>)}</ul> : <p role="status">Loading the governed checklist catalog…</p>}
          {cursor ? <button type="button" disabled={loading} onClick={() => void loadCatalog(cursor)}>Load next catalog page</button> : null}
          <footer className="planning-post-release-actions"><p role="status">{dirty ? "Unconfirmed changes are local until you confirm the selection." : setup.status === "FINALIZED" ? setup.nextAction : setup.nextAction}</p><div><button type="button" className="planning-intake-secondary" disabled={busy || finalized || !dirty} onClick={() => { setSelectedIds([...committedIds]); setStatus("Unconfirmed changes were discarded."); }}>Undo changes</button><button type="button" className="planning-intake-primary" disabled={busy || finalized || !dirty} onClick={() => void confirmSelection()}>Confirm selection</button><button type="button" className="planning-intake-primary" disabled={busy || finalized || dirty || !setup.selectedCount} title={dirty ? "Confirm the current selection before finalization." : !setup.selectedCount ? "Select and confirm at least one checklist item." : undefined} onClick={() => void finalize()}>Finalize Audit package</button></div></footer>
        </section> : null}
      </> : !error ? <p role="status">Loading released Planning context…</p> : null}
    </div>
  </WorkspaceShell>;
}
