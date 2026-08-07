import { useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { z } from "zod";

import { useApplicationRuntime } from "../../app/providers";
import type {
  CanonicalAuditScopeOption,
  CanonicalQuestionCatalogPage,
  CanonicalQuestionCatalogEntry,
  CanonicalQuestionUsageClass,
  CanonicalSelectionPreview,
  PlanningIntakeDraftValues,
  PlanningIntakeDraftView,
  PlanningIntakeInspectionCategory,
} from "../../backend/backend";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";

const stepDefinitions = [
  { number: 1, title: "Inspection basics" },
  { number: 2, title: "Category and purpose" },
  { number: 3, title: "When and where" },
  { number: 4, title: "Choose questions and budget" },
  { number: 5, title: "Review and submit" },
] as const;

type PlanningIntakeFormValues = Omit<PlanningIntakeDraftValues, "requestedBudget"> & {
  requestedBudget: string;
};

const requestedBudgetSchema = z
  .string()
  .trim()
  .min(1, "Requested budget is required")
  .transform((value) => Number(value))
  .refine((value) => Number.isFinite(value) && value >= 0, "Requested budget must be zero or greater");

const stepSchemas = {
  1: z.object({
    organizationId: z.string().min(1, "Organization is required"),
    applicationType: z.string().min(1, "Application type is required"),
    domain: z.string().min(1, "Domain is required"),
  }),
  2: z.object({
    inspectionCategory: z.enum(["Routine / Announced", "Ad Hoc / Unannounced"]),
    purpose: z.string().trim().min(1, "Purpose is required"),
  }),
  3: z.object({
    plannedDate: z.string().min(1, "Planned date is required"),
    location: z.string().trim().min(1, "Location is required"),
  }),
  4: z.object({
    catalogVersion: z.string().min(1, "Question catalog is required"),
    selectedQuestionVersionIds: z.array(z.string()).min(1, "Select at least one question"),
    selectionDigest: z.string().min(1, "Question selection must be frozen"),
    requestedBudget: requestedBudgetSchema,
  }),
} as const;

function pathForStep(step: number, draftId?: string): string {
	return `/department-manager/new-audit/step-${step}${draftId ? `?draftId=${encodeURIComponent(draftId)}` : ""}`;
}

function operationId(prefix: string): string {
  return `${prefix}-${globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`}`.toUpperCase();
}

function stepFromPath(pathname: string): number {
  const candidate = Number(pathname.match(/step-(\d)$/)?.[1] ?? 1);
  return Math.min(5, Math.max(1, candidate));
}

function noticePolicyFor(category: PlanningIntakeInspectionCategory): PlanningIntakeDraftValues["noticePolicy"] {
  return category === "Ad Hoc / Unannounced" ? "WITHHELD" : "ADVANCE";
}

function noticeLabel(values: Pick<PlanningIntakeDraftValues, "noticePolicy">): string {
  return values.noticePolicy === "WITHHELD" ? "No Advance Notice (withheld)" : "Advance Notice Required";
}

async function selectionDigestFor(ids: readonly string[]): Promise<string> {
  const canonical = [...new Set(ids)].map((id, index) => `${index}\u0000${id}\n`).join("");
  const bytes = new TextEncoder().encode(canonical);
  const digest = await globalThis.crypto.subtle.digest("SHA-256", bytes);
  return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("");
}

function validationMessage(step: number, values: PlanningIntakeFormValues): string | null {
  if (step === 5) {
    for (const priorStep of [1, 2, 3, 4] as const) {
      const result = stepSchemas[priorStep].safeParse(values);
      if (!result.success) return result.error.issues[0]?.message ?? "Planning intake is incomplete";
    }
    return null;
  }
  const schema = stepSchemas[step as keyof typeof stepSchemas];
  if (!schema) return null;
  const result = schema.safeParse(values);
  return result.success ? null : result.error.issues[0]?.message ?? "Planning intake is incomplete";
}

function formValuesFor(draft: PlanningIntakeDraftView): PlanningIntakeFormValues {
  return { ...draft, selectedQuestionVersionIds: [...(draft.selectedQuestionVersionIds ?? [])], requestedBudget: String(draft.requestedBudget) };
}

function commandValuesFor(values: PlanningIntakeFormValues): PlanningIntakeDraftValues {
  const result = requestedBudgetSchema.safeParse(values.requestedBudget);
  if (!result.success) throw new Error(result.error.issues[0]?.message ?? "Requested budget is invalid");
  return { ...values, requestedBudget: result.data };
}

export function NewAuditWizardPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("manager") ?? runtime.backend, [runtime]);
  const navigate = useNavigate();
  const location = useLocation();
  const step = stepFromPath(location.pathname);
  const [draft, setDraft] = useState<PlanningIntakeDraftView | null>(null);
  const [values, setValues] = useState<PlanningIntakeFormValues | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [preview, setPreview] = useState(false);
  const [busy, setBusy] = useState(false);
  const [catalogPage, setCatalogPage] = useState<CanonicalQuestionCatalogPage | null>(null);
  const [catalogBusy, setCatalogBusy] = useState(false);
  const [catalogSearch, setCatalogSearch] = useState("");
  const [catalogFormCode, setCatalogFormCode] = useState("");
  const [catalogDomain, setCatalogDomain] = useState("");
  const [catalogTopic, setCatalogTopic] = useState("");
  const [catalogRiskBand, setCatalogRiskBand] = useState("");
  const [catalogSourceGapState, setCatalogSourceGapState] = useState("");
  const [catalogSelectedFilter, setCatalogSelectedFilter] = useState<"all" | "selected" | "unselected">("all");
  const [catalogCursor, setCatalogCursor] = useState<string | undefined>();
  const [catalogPreviousCursors, setCatalogPreviousCursors] = useState<string[]>([]);
  const [catalogPageNumber, setCatalogPageNumber] = useState(1);
  const [catalogDetail, setCatalogDetail] = useState<CanonicalQuestionCatalogEntry | null>(null);
  const [selectionPreview, setSelectionPreview] = useState<CanonicalSelectionPreview | null>(null);
  const [scopeOptionLabel, setScopeOptionLabel] = useState<string | null>(null);
  const [scopeOptions, setScopeOptions] = useState<CanonicalAuditScopeOption[]>([]);
  // Catalog authority is returned by the server-owned scope selector. The
  // normal API returns GOVERNED_OPERATIONAL; only the disposable local
  // preprod profile may return PREPROD_EXERCISE.
  const [auditUsageClass, setAuditUsageClass] = useState<CanonicalQuestionUsageClass>("GOVERNED_OPERATIONAL");

  useEffect(() => {
    let cancelled = false;
    if (!backend.planningIntake) {
      setError("Planning intake commands are unavailable in this build profile.");
      return () => { cancelled = true; };
    }
    const requestedDraftId = new URLSearchParams(location.search).get("draftId");
    const load = (async () => {
      if (!backend.canonicalQuestionReview) {
        throw new Error("Server-authorized audit scope selection is unavailable in this build profile.");
      }
      const optionPages: CanonicalAuditScopeOption[] = [];
      let cursor: string | undefined;
      do {
        const page = await backend.canonicalQuestionReview.listScopeOptions({ limit: 25, cursor });
        optionPages.push(...page.items);
        cursor = page.nextCursor ?? undefined;
      } while (cursor && optionPages.length < 1000);
      const options = { items: optionPages };
      if (!cancelled) {
        setScopeOptions(options.items);
        setAuditUsageClass(options.items[0]?.usageClass ?? "GOVERNED_OPERATIONAL");
      }
      if (requestedDraftId) {
        const loadedDraft = await backend.planningIntake.getDraft({ draftId: requestedDraftId });
        if (!cancelled) {
          // Catalog versions are globally unique, so the server-enumerated
          // option is the authoritative usage-class pin for a saved draft.
          // Do not trust or rewrite a client-side usage value while resuming.
          const loadedUsageClass = options.items.find((option) => option.catalogVersion === loadedDraft.catalogVersion)?.usageClass;
          const matchingOption = options.items.find((option) => option.catalogVersion === loadedDraft.catalogVersion &&
            option.usageClass === loadedUsageClass &&
            option.organizationId === loadedDraft.organizationId &&
            option.providerScopeId === loadedDraft.providerScopeId &&
            option.regulatedTargetId === loadedDraft.regulatedTargetId);
          if (!matchingOption) {
            throw new Error("The saved Planning draft no longer has an exact authorized catalog/scope/target option.");
          }
          setAuditUsageClass(matchingOption.usageClass);
        }
        return loadedDraft;
      }
      // A new audit remains an uncommitted setup until the manager explicitly
      // chooses an authorized organization/provider scope/target.
      return null;
    })();
    void load.then((loaded) => {
      if (!cancelled) {
        if (loaded) {
          setDraft(loaded);
          setValues(formValuesFor(loaded));
          if (!scopeOptionLabel && loaded.organizationName) setScopeOptionLabel(loaded.organizationName);
        }
      }
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend, location.search]);

  async function changeScope(option: (typeof scopeOptions)[number]) {
    if (!backend.planningIntake || (values && option.providerScopeId === values.providerScopeId && option.regulatedTargetId === values.regulatedTargetId)) return;
    setAuditUsageClass(option.usageClass);
    setBusy(true);
    setError(null);
    try {
      const nextValues: PlanningIntakeDraftValues = {
        ...(values ? commandValuesFor(values) : {
          organizationId: "",
          organizationName: "",
          applicationType: option.inspectionTypes[0] ?? "CABIN_INSPECTION",
          domain: "Cabin Safety",
          inspectionCategory: "Routine / Announced" as const,
          noticePolicy: "ADVANCE" as const,
          purpose: "",
          triggerType: "Department Manager initiated",
          riskCategory: "",
          plannedDate: "",
          mode: "On-site" as const,
          location: "",
          catalogVersion: "",
          scopeDraftId: "",
          selectionDigest: "",
          selectedQuestionVersionIds: [],
          requestedBudget: 0,
          currency: "USD",
          providerScopeId: "",
          regulatedTargetId: "",
        }),
        organizationId: option.organizationId,
        organizationName: option.organizationName,
        applicationType: option.inspectionTypes[0] ?? "CABIN_INSPECTION",
        catalogVersion: option.catalogVersion,
        providerScopeId: option.providerScopeId,
        regulatedTargetId: option.regulatedTargetId,
        scopeDraftId: "",
        selectionDigest: "",
        selectedQuestionVersionIds: [],
      };
      const operationIdValue = operationId("NEW-AUDIT-SCOPE");
      const replacement = await backend.planningIntake.createDraft({
        operationId: operationIdValue,
        idempotencyKey: operationIdValue,
        expectedRevision: null,
        values: nextValues,
      });
      setDraft(replacement);
      setValues(formValuesFor(replacement));
      setScopeOptionLabel(`${option.organizationName} · ${option.providerTypeLabel} · ${option.targetLabel}`);
      setStatus("A new server-owned draft was opened for the selected organization/provider scope/target.");
      // Preserve the requested step after the explicit scope choice. A direct
      // link to a later step still requires the same server-owned selection,
      // then resumes at that step instead of silently resetting the user.
      navigate(pathForStep(step, replacement.id), { replace: true });
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  useEffect(() => {
    if (step !== 4 || !values || !values.catalogVersion || !backend.canonicalQuestionReview) return;
    const controller = new AbortController();
    setCatalogBusy(true);
    void backend.canonicalQuestionReview.listCatalog({
      catalogVersion: values.catalogVersion,
      usageClass: auditUsageClass,
      search: catalogSearch || undefined,
      formCode: catalogFormCode || undefined,
      domain: catalogDomain || undefined,
      topic: catalogTopic || undefined,
      riskBand: catalogRiskBand || undefined,
      sourceGapState: catalogSourceGapState || undefined,
      selected: catalogSelectedFilter,
      scopeId: values.scopeDraftId || undefined,
      cursor: catalogCursor,
      limit: 25,
    }, { signal: controller.signal }).then((page) => {
      if (!controller.signal.aborted) setCatalogPage(page);
    }).catch((cause) => {
      if (!controller.signal.aborted) setError(errorMessage(cause));
    }).finally(() => {
      if (!controller.signal.aborted) setCatalogBusy(false);
    });
    return () => controller.abort();
  }, [auditUsageClass, backend, catalogCursor, catalogDomain, catalogFormCode, catalogRiskBand, catalogSearch, catalogSelectedFilter, catalogSourceGapState, catalogTopic, step, values?.catalogVersion, values?.scopeDraftId]);

  function resetCatalogPage() {
    setCatalogCursor(undefined);
    setCatalogPreviousCursors([]);
    setCatalogPageNumber(1);
  }

  function update<K extends keyof PlanningIntakeFormValues>(key: K, value: PlanningIntakeFormValues[K]) {
    setValues((current) => current ? { ...current, [key]: value } : current);
    setStatus(null);
  }

  function updateCategory(category: PlanningIntakeInspectionCategory) {
    setValues((current) => current ? {
      ...current,
      inspectionCategory: category,
      noticePolicy: noticePolicyFor(category),
    } : current);
    setStatus(null);
  }

  async function commitQuestionSelection(questionVersionIds: string[], operationKind: "ADD" | "REMOVE") {
    if (busy || !values || !draft || !backend.canonicalQuestionReview) return;
    if (!values.scopeDraftId) {
      setError("The server did not return a canonical scope identity; reload this draft before selecting questions.");
      return;
    }
    setBusy(true);
    try {
      const expectedSelectionDigest = values.selectionDigest || await selectionDigestFor([]);
      const previewOperationId = operationId(`SCOPE-${draft.id}-PREVIEW`);
      const previewReceipt = await backend.canonicalQuestionReview.previewSelection({
        scopeId: values.scopeDraftId,
        operationId: previewOperationId,
        idempotencyKey: previewOperationId,
			 expectedSelectionDigest,
			 questionVersionIds,
			 operationKind,
        usageClass: auditUsageClass,
        filter: {},
      });
      setSelectionPreview(previewReceipt);
      const operationIdValue = operationId(`SCOPE-${draft.id}-COMMIT`);
      const receipt = await backend.canonicalQuestionReview.commitSelection({
        scopeId: values.scopeDraftId,
        operationId: operationIdValue,
        previewOperationId,
        idempotencyKey: operationIdValue,
			 expectedSelectionDigest,
				 questionVersionIds,
			 operationKind,
        usageClass: auditUsageClass,
        filter: {},
      });
      setValues((current) => current ? {
        ...current,
        selectedQuestionVersionIds: receipt.selection.selectedQuestionVersionIds,
        selectionDigest: receipt.selection.selectionDigest,
      } : current);
      setError(null);
      setStatus(`Exact question selection committed · ${receipt.selection.selectedCount} selected · ${receipt.selection.selectionDigest}`);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  function toggleQuestion(questionVersionId: string) {
    if (!values) return;
    const selected = values.selectedQuestionVersionIds?.includes(questionVersionId) ?? false;
    void commitQuestionSelection([questionVersionId], selected ? "REMOVE" : "ADD");
  }

  async function openCatalogDetail(question: CanonicalQuestionCatalogEntry) {
    setCatalogDetail(question);
    if (!values || !backend.canonicalQuestionReview) return;
    try {
      const detail = await backend.canonicalQuestionReview.getQuestion({
        catalogVersion: values.catalogVersion ?? "",
        usageClass: auditUsageClass,
        questionVersionId: question.questionVersionId,
        scopeId: values.scopeDraftId || undefined,
      });
      setCatalogDetail(detail);
    } catch (cause) {
      setError(errorMessage(cause));
    }
  }

  async function saveDraft(nextValues = values): Promise<PlanningIntakeDraftView | null> {
    if (!backend.planningIntake || !draft || !nextValues) return null;
    const saved = await backend.planningIntake.saveDraft({
      draftId: draft.id,
      expectedRevision: draft.revision,
      idempotencyKey: `SAVE-${draft.id}-R${draft.revision}`,
      values: commandValuesFor(nextValues),
    });
    setDraft(saved);
    setValues(formValuesFor(saved));
    return saved;
  }

  async function move(direction: -1 | 1) {
    if (!values) return;
    const validationError = direction > 0 ? validationMessage(step, values) : null;
    if (validationError) {
      setError(validationError);
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const saved = await saveDraft();
      navigate(pathForStep(step + direction, saved?.id ?? draft?.id));
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  async function saveOnly() {
    setBusy(true);
    setError(null);
    try {
      const saved = await saveDraft();
      if (saved) {
        setStatus(`Draft saved · revision ${saved.revision}`);
        navigate(pathForStep(step, saved.id), { replace: true });
      }
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  async function submit() {
    if (!backend.planningIntake || !values) return;
    const validationError = validationMessage(5, values);
    if (validationError) {
      setError(validationError);
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const saved = await saveDraft();
      if (!saved) return;
      const output = await backend.planningIntake.submit({
        draftId: saved.id,
        expectedRevision: saved.revision,
        idempotencyKey: `SUBMIT-${saved.id}-R${saved.revision}`,
      });
      navigate(`/department-manager/audit-plan?planningItemId=${output.planningItem.id}`);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  const definition = stepDefinitions[step - 1] ?? stepDefinitions[0];

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel={`New Audit Wizard ${step}`}>
      <div className="planning-intake-page" data-draft-id={draft?.id} data-testid={draft ? "new-audit-wizard-page" : undefined}>
        <header className="planning-intake-header workbench-page-header">
          <p className="eyebrow">Department planning intake</p>
          <h1>New Inspection</h1>
          <p>Create a governed Planning item. An executable Audit is created only after the accepted release and confirmation stage.</p>
        </header>
        <ol aria-label="Planning intake steps" className="planning-intake-steps">
          {stepDefinitions.map((item) => <li aria-current={item.number === step ? "step" : undefined} className={item.number === step ? "is-current" : item.number < step ? "is-complete" : ""} key={item.number}><span>{item.number}</span><b>{item.title}</b></li>)}
        </ol>
        <CommandError message={error} />
        {status ? <p className="planning-intake-status" role="status">{status}</p> : null}
        <section aria-label="Planning intake form" className="planning-intake-form">
          <header><span>Step {step} of 5</span><h2>Step {step} of 5 — {definition.title}</h2></header>
          {!values ? <div className="planning-intake-fields">
            <label>Organization, provider scope, and regulated target
              <select aria-label="Organization, provider scope, and regulated target" disabled={busy || !scopeOptions.length} value="" onChange={(event) => { const option = scopeOptions.find((item) => `${item.providerScopeId}:${item.regulatedTargetId}` === event.target.value); if (option) void changeScope(option); }}>
                <option value="">Choose an authorized scope…</option>
                {scopeOptions.map((item) => <option key={`${item.providerScopeId}:${item.regulatedTargetId}`} value={`${item.providerScopeId}:${item.regulatedTargetId}`}>{item.organizationName} · {item.providerTypeLabel} · {item.targetLabel}</option>)}
              </select>
            </label>
            <div className="planning-intake-notice" role="note"><b>Selection is required</b><span>The server will create the opaque Planning draft only after this explicit Department Manager scope choice.</span></div>
          </div> : null}
          {values && step === 1 ? <div className="planning-intake-fields">
            <label>Organization<select aria-label="Organization" disabled={busy || !scopeOptions.length} value={`${values.providerScopeId}:${values.regulatedTargetId}`} onChange={(event) => { const option = scopeOptions.find((item) => `${item.providerScopeId}:${item.regulatedTargetId}` === event.target.value); if (option) void changeScope(option); }}><option value={`${values.providerScopeId}:${values.regulatedTargetId}`}>{values.organizationName || values.organizationId}</option>{scopeOptions.filter((item) => `${item.providerScopeId}:${item.regulatedTargetId}` !== `${values.providerScopeId}:${values.regulatedTargetId}`).map((item) => <option key={`${item.providerScopeId}:${item.regulatedTargetId}`} value={`${item.providerScopeId}:${item.regulatedTargetId}`}>{item.organizationName} · {item.providerTypeLabel} · {item.targetLabel}</option>)}</select></label>
            <label>Application Type<select aria-label="Application Type" disabled={busy || (scopeOptions.find((item) => item.providerScopeId === values.providerScopeId && item.regulatedTargetId === values.regulatedTargetId)?.inspectionTypes.length ?? 0) <= 1} value={values.applicationType} onChange={(event) => update("applicationType", event.target.value)}>{(scopeOptions.find((item) => item.providerScopeId === values.providerScopeId && item.regulatedTargetId === values.regulatedTargetId)?.inspectionTypes ?? []).map((inspectionType) => <option key={inspectionType} value={inspectionType}>{inspectionType}</option>)}</select><small>Server-enumerated by provider scope; changing it requires a new scope selection.</small></label>
            <label>Domain<input aria-label="Domain" value={values.domain} onChange={(event) => update("domain", event.target.value)} /></label>
            <div className="planning-intake-notice" role="note"><b>Server-authorized scope</b><span>{scopeOptionLabel ?? `${values.organizationName || values.organizationId} · provider scope ${values.providerScopeId || "pending"} · target ${values.regulatedTargetId || "pending"}`}</span></div>
          </div> : null}
          {values && step === 2 ? <div className="planning-intake-fields">
            <label>Inspection Category<select aria-label="Inspection Category" value={values.inspectionCategory} onChange={(event) => updateCategory(event.target.value as PlanningIntakeInspectionCategory)}><option value="Routine / Announced">Routine / Announced</option><option value="Ad Hoc / Unannounced">Ad Hoc / Unannounced</option></select></label>
            <label className="is-wide">Purpose<textarea aria-label="Purpose" value={values.purpose} onChange={(event) => update("purpose", event.target.value)} /></label>
            <label>Trigger Type<select aria-label="Trigger Type" value={values.triggerType} onChange={(event) => update("triggerType", event.target.value)}><option value="Department Manager initiated">Department Manager initiated</option><option value="Risk signal">Risk signal</option></select></label>
            <label>Risk Category<input aria-label="Risk Category" value={values.riskCategory} onChange={(event) => update("riskCategory", event.target.value)} /></label>
            <div className="planning-intake-notice" role="note"><b>{noticeLabel(values)}</b><span>{values.noticePolicy === "WITHHELD" ? "Organization notice remains withheld through this Planning stage." : "Advance notice applies after the accepted governance stage."}</span></div>
          </div> : null}
          {values && step === 3 ? <div className="planning-intake-fields">
            <label>Planned Date<input aria-label="Planned Date" type="date" value={values.plannedDate} onChange={(event) => update("plannedDate", event.target.value)} /></label>
            <label>Mode<select aria-label="Mode" value={values.mode} onChange={(event) => update("mode", event.target.value as PlanningIntakeDraftValues["mode"])}><option value="On-site">On-site</option><option value="Remote">Remote</option></select></label>
            <label className="is-wide">Location<input aria-label="Location" value={values.location} onChange={(event) => update("location", event.target.value)} /></label>
          </div> : null}
          {values && step === 4 ? <div className="planning-intake-fields">
            <div className="planning-intake-catalog" aria-label="Question catalog selection">
              <div className="planning-intake-catalog-header"><div><span className="eyebrow">{auditUsageClass === "PREPROD_EXERCISE" ? "Disposable preprod exercise boundary" : "Governed selection boundary"}</span><h3>Choose the exact question subset</h3><p>Only the selected immutable question versions are sent to Finance. No checklist template is selected here.</p></div><span className="planning-intake-catalog-count" aria-live="polite">{values.selectedQuestionVersionIds?.length ?? 0} selected</span></div>
              <div className="planning-intake-catalog-filters" aria-label="New Audit question search and filters">
                <label>Search<input aria-label="New Audit question search" value={catalogSearch} onChange={(event) => { setCatalogSearch(event.target.value); resetCatalogPage(); }} placeholder="Form, proposal, or question identity" /></label>
                <label>Form<input aria-label="New Audit form filter" value={catalogFormCode} onChange={(event) => { setCatalogFormCode(event.target.value); resetCatalogPage(); }} /></label>
                <label>Domain<input aria-label="New Audit domain filter" value={catalogDomain} onChange={(event) => { setCatalogDomain(event.target.value); resetCatalogPage(); }} /></label>
                <label>Topic<input aria-label="New Audit topic filter" value={catalogTopic} onChange={(event) => { setCatalogTopic(event.target.value); resetCatalogPage(); }} /></label>
                <label>Risk band<input aria-label="New Audit risk filter" value={catalogRiskBand} onChange={(event) => { setCatalogRiskBand(event.target.value); resetCatalogPage(); }} /></label>
                <label>Source gap<input aria-label="New Audit source gap filter" value={catalogSourceGapState} onChange={(event) => { setCatalogSourceGapState(event.target.value); resetCatalogPage(); }} /></label>
                <label>Selected state<select aria-label="New Audit selected filter" value={catalogSelectedFilter} onChange={(event) => { setCatalogSelectedFilter(event.target.value as typeof catalogSelectedFilter); resetCatalogPage(); }}><option value="all">All questions</option><option value="selected">Selected in scope</option><option value="unselected">Not selected</option></select></label>
              </div>
              {catalogBusy ? <p role="status">Loading catalog page…</p> : null}
              {!catalogBusy && !catalogPage ? <p role="status">Catalog selection is unavailable in this build profile.</p> : null}
              {catalogPage ? <ul className="planning-intake-catalog-list">
                {catalogPage.items.map((question) => {
                  const checked = values.selectedQuestionVersionIds?.includes(question.questionVersionId) ?? false;
                  return <li key={question.questionVersionId}><label><input aria-label={`Select ${question.formCode} ${question.ordinal}`} checked={checked} disabled={busy || !question.canSelect} onChange={() => toggleQuestion(question.questionVersionId)} title={!question.canSelect ? "This question is not selectable in the current server-authorized scope." : undefined} type="checkbox" /><span><b>{question.formCode} · item {question.ordinal}</b><small>{question.questionVersionId} · {question.proposedDomain ?? "Unclassified"}</small></span></label><button className="planning-intake-question-detail" type="button" onClick={() => void openCatalogDetail(question)}>View dossier</button></li>;
                })}
              </ul> : null}
              <section aria-label="Selected question tray" className="planning-intake-selected-tray">
                <header><h4>Selected question tray</h4><span>{values.selectedQuestionVersionIds?.length ?? 0} exact immutable versions</span></header>
                {(values.selectedQuestionVersionIds ?? []).length ? <ul>{(values.selectedQuestionVersionIds ?? []).map((questionId) => <li key={questionId}><span>{questionId}</span><button type="button" disabled={busy} onClick={() => void commitQuestionSelection([questionId], "REMOVE")}>Remove</button></li>)}</ul> : <p>No questions selected. Select at least one version to continue.</p>}
              </section>
              {catalogDetail ? <aside aria-label="Selected question dossier" className="planning-intake-question-dossier"><header><h4>Question dossier</h4><button type="button" onClick={() => setCatalogDetail(null)}>Close</button></header><strong>{catalogDetail.formCode} · item {catalogDetail.ordinal}</strong><p>{catalogDetail.prompt ?? "Prompt unavailable in this profile."}</p><dl><div><dt>Question version</dt><dd>{catalogDetail.questionVersionId}</dd></div><div><dt>Source gap</dt><dd>{catalogDetail.sourceGapState}</dd></div><div><dt>Reference</dt><dd>{catalogDetail.configuredReference ?? "Not configured"}</dd></div><div><dt>Expected Evidence</dt><dd>{catalogDetail.expectedEvidence ?? "Not configured"}</dd></div></dl></aside> : null}
              {selectionPreview ? <p className="planning-intake-selection-preview" role="status">Preview: {selectionPreview.preview.selectedCount} selected · {selectionPreview.valid ? "ready to commit" : selectionPreview.reason}</p> : null}
              <div className="planning-intake-catalog-pagination" aria-label="New Audit question pagination"><button disabled={catalogBusy || !catalogPreviousCursors.length} onClick={() => { const history = [...catalogPreviousCursors]; setCatalogCursor(history.pop()); setCatalogPreviousCursors(history); setCatalogPageNumber((value) => Math.max(1, value - 1)); }} type="button">Previous questions</button><button disabled={catalogBusy || (!catalogSearch && !catalogFormCode && !catalogDomain && !catalogTopic && !catalogRiskBand && !catalogSourceGapState && catalogSelectedFilter === "all")} onClick={() => { setCatalogSearch(""); setCatalogFormCode(""); setCatalogDomain(""); setCatalogTopic(""); setCatalogRiskBand(""); setCatalogSourceGapState(""); setCatalogSelectedFilter("all"); resetCatalogPage(); }} type="button">Clear filters</button><span aria-live="polite">{catalogPage?.totalCount ?? 0} matching questions · page {catalogPageNumber}</span><button disabled={catalogBusy || !catalogPage?.nextCursor} onClick={() => { if (!catalogPage?.nextCursor) return; setCatalogPreviousCursors((history) => [...history, catalogCursor ?? ""]); setCatalogCursor(catalogPage.nextCursor ?? undefined); setCatalogPageNumber((value) => value + 1); }} type="button">Next questions</button></div>
            </div>
            <label>Question Catalog Version<input aria-label="Question Catalog Version" readOnly value={values.catalogVersion ?? "Unavailable"} /></label>
            <label>Requested Budget<input aria-label="Requested Budget" min="0" type="number" value={values.requestedBudget} onChange={(event) => update("requestedBudget", event.target.value)} /></label>
            <label>Currency<select aria-label="Currency" value={values.currency} onChange={(event) => update("currency", event.target.value as PlanningIntakeDraftValues["currency"])}><option value="USD">USD</option><option value="EUR">EUR</option><option value="NAD">NAD</option></select></label>
            <div className="planning-intake-notice" role="note"><b>Finance Review is required even when the requested budget is zero.</b><span>The selection digest is retained with the Planning draft and cannot be replaced by a template identifier.</span></div>
          </div> : null}
          {values && step === 5 ? <div className="planning-intake-review">
            <dl><div><dt>Draft</dt><dd>{draft?.id}</dd></div><div><dt>Organization</dt><dd>{values.organizationName} · {values.organizationId}</dd></div><div><dt>Category</dt><dd>{values.inspectionCategory}</dd></div><div><dt>Purpose</dt><dd>{values.purpose || "Not provided"}</dd></div><div><dt>Planned work</dt><dd>{values.plannedDate} · {values.mode} · {values.location || "Location required"}</dd></div><div><dt>Question selection</dt><dd>{values.catalogVersion} · {values.selectedQuestionVersionIds?.length ?? 0} exact question versions · {values.selectionDigest || "Not frozen"}</dd></div><div><dt>Requested budget</dt><dd>{values.requestedBudget} {values.currency}</dd></div><div><dt>Notice</dt><dd>{noticeLabel(values)}</dd></div></dl>
            <div className="planning-intake-governance"><b>Department Manager → Finance Review → General Manager → Executive Director → General Manager Release</b><p>No executable Audit is created at this step. The submitted record remains a Planning item awaiting Finance Review.</p></div>
            {preview ? <article className="planning-intake-preview" aria-label="Planning intake preview"><p className="eyebrow">Durable in-screen preview</p><h3>{values.inspectionCategory} — {values.organizationName}</h3><p>{values.purpose}</p><small>{draft?.id} · revision {draft?.revision}</small></article> : null}
          </div> : null}
        </section>
        <section aria-label="Planning intake actions" className="planning-intake-actions">
          {step === 1 ? <button onClick={() => navigate("/department-manager/audit-plan")} type="button">Cancel</button> : <button disabled={busy} onClick={() => void move(-1)} type="button">Back</button>}
          {step === 1 ? <button disabled={busy || !values} onClick={() => void saveOnly()} type="button">Save draft</button> : null}
          {step < 5 ? <button disabled={busy || !values} onClick={() => void move(1)} type="button">Next</button> : null}
          {step === 5 ? <button disabled={busy || !values} onClick={() => setPreview((current) => !current)} type="button">Preview</button> : null}
          {step === 5 ? <button disabled={busy || !values} onClick={() => void submit()} type="button">Submit for Finance Review</button> : null}
        </section>
      </div>
    </WorkspaceShell>
  );
}
