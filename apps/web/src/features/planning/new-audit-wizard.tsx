import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { z } from "zod";

import { useApplicationRuntime } from "../../app/providers";
import type {
  CanonicalAuditScopeOption,
  CanonicalApplicationType,
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

const stageDefinitions = [
  { number: 1, title: "Set up", description: "Scope, purpose, timing, and location" },
  { number: 2, title: "Choose questions", description: "Review and confirm the exact immutable subset" },
  { number: 3, title: "Review and submit", description: "Confirm the Planning record for Finance" },
] as const;

const selectionBatchLimit = 500;
const selectedTrayRenderLimit = 100;

type SelectionOperationKind = "ADD" | "REMOVE";

interface SelectionPreviewOperation {
  operationId: string;
  operationKind: SelectionOperationKind;
  questionVersionIds: string[];
}

type CatalogFacetOption = { value: string; count: number };

function catalogValueLabel(value: string): string {
  return value
    .toLocaleLowerCase()
    .split("_")
    .map((part) => part.charAt(0).toLocaleUpperCase() + part.slice(1))
    .join(" ");
}

function CatalogFacetPicker({
  label,
  ariaLabel,
  options,
  selected,
  onChange,
}: {
  label: string;
  ariaLabel: string;
  options: CatalogFacetOption[];
  selected: string[];
  onChange: (next: string[]) => void;
}) {
  return (
    <details className="planning-intake-facet-picker">
      <summary aria-label={ariaLabel}>{selected.length ? `${label} · ${selected.length} selected` : `${label} · Any`}</summary>
      <div className="planning-intake-facet-options" role="group" aria-label={ariaLabel}>
        {options.length ? options.map((option) => <label key={option.value}>
          <input checked={selected.includes(option.value)} onChange={(event) => onChange(event.target.checked ? [...selected, option.value] : selected.filter((value) => value !== option.value))} type="checkbox" />
          <span>{catalogValueLabel(option.value)}</span><small>{option.count.toLocaleString("en-US")}</small>
        </label>) : <p>No values in the current result set.</p>}
      </div>
    </details>
  );
}

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
    riskCategory: z.string().trim().min(1, "Risk category is required"),
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

function stageFromStep(step: number): number {
  if (step <= 3) return 1;
  return step === 4 ? 2 : 3;
}

function noticePolicyFor(category: PlanningIntakeInspectionCategory): PlanningIntakeDraftValues["noticePolicy"] {
  return category === "Ad Hoc / Unannounced" ? "WITHHELD" : "ADVANCE";
}

function noticeLabel(values: Pick<PlanningIntakeDraftValues, "noticePolicy">): string {
  return values.noticePolicy === "WITHHELD" ? "No Advance Notice (withheld)" : "Advance Notice Required";
}

function inspectionTypeFor(types: readonly CanonicalApplicationType[]): CanonicalApplicationType {
  const firstSupported = types.find((type) => ["RAMP", "CABIN", "RAMP_INSPECTION", "CABIN_INSPECTION"].includes(type));
  if (firstSupported) return firstSupported;
  throw new Error("The selected server-owned scope has no supported inspection type.");
}

async function selectionDigestFor(ids: readonly string[]): Promise<string> {
  const canonical = [...new Set(ids)].map((id, index) => `${index}\u0000${id}\n`).join("");
  const bytes = new TextEncoder().encode(canonical);
  const digest = await globalThis.crypto.subtle.digest("SHA-256", bytes);
  return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("");
}

function nextSelectionBatch(
  current: readonly string[],
  desired: readonly string[],
): { operationKind: SelectionOperationKind; questionVersionIds: string[] } | null {
  const currentSet = new Set(current);
  const desiredSet = new Set(desired);
  const removals = current.filter((id) => !desiredSet.has(id));
  if (removals.length) {
    return { operationKind: "REMOVE", questionVersionIds: removals.slice(0, selectionBatchLimit) };
  }
  const additions = desired.filter((id) => !currentSet.has(id));
  if (additions.length) {
    return { operationKind: "ADD", questionVersionIds: additions.slice(0, selectionBatchLimit) };
  }
  return null;
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
  return {
    ...draft,
    // The API requires a non-empty risk category at Finance submission. Keep
    // older/local drafts resumable while making that server requirement
    // visible and editable in the canonical wizard.
    riskCategory: draft.riskCategory || "Configured inspection risk",
    selectedQuestionVersionIds: [...(draft.selectedQuestionVersionIds ?? [])],
    requestedBudget: String(draft.requestedBudget),
  };
}

function commandValuesFor(values: PlanningIntakeFormValues): PlanningIntakeDraftValues {
  const result = requestedBudgetSchema.safeParse(values.requestedBudget);
  if (!result.success) throw new Error(result.error.issues[0]?.message ?? "Requested budget is invalid");
  // PlanningIntakeDraftView extends the command values with server-owned
  // identity/revision fields. Keep those fields out of strict JSON commands;
  // the API deliberately rejects client-authored id/revision/updatedAt values.
  return {
    organizationId: values.organizationId,
    organizationName: values.organizationName,
    applicationType: values.applicationType,
    domain: values.domain,
    inspectionCategory: values.inspectionCategory,
    noticePolicy: values.noticePolicy,
    purpose: values.purpose,
    triggerType: values.triggerType,
    riskCategory: values.riskCategory,
    plannedDate: values.plannedDate,
    mode: values.mode,
    location: values.location,
    templateVersionId: values.templateVersionId,
    scope: values.scope,
    catalogVersion: values.catalogVersion,
    scopeDraftId: values.scopeDraftId,
    selectionDigest: values.selectionDigest,
    selectedQuestionVersionIds: values.selectedQuestionVersionIds,
    estimatedResourceRequirement: values.estimatedResourceRequirement,
    formDistribution: values.formDistribution,
    domainDistribution: values.domainDistribution,
    providerScopeId: values.providerScopeId,
    regulatedTargetId: values.regulatedTargetId,
    requestedBudget: result.data,
    currency: values.currency,
  };
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
  const [catalogFormCode, setCatalogFormCode] = useState<string[]>([]);
  const [catalogDomain, setCatalogDomain] = useState<string[]>([]);
  const [catalogTopic, setCatalogTopic] = useState<string[]>([]);
  const [catalogRiskBand, setCatalogRiskBand] = useState<string[]>([]);
  const [catalogSourceGapState, setCatalogSourceGapState] = useState("");
  const [catalogChecklistFocus, setCatalogChecklistFocus] = useState<string[]>([]);
  const [catalogRecommendationState, setCatalogRecommendationState] = useState("");
  const [catalogSelectedFilter, setCatalogSelectedFilter] = useState<"all" | "selected" | "unselected">("all");
  const [catalogCursor, setCatalogCursor] = useState<string | undefined>();
  const [catalogPreviousCursors, setCatalogPreviousCursors] = useState<string[]>([]);
  const [catalogPageNumber, setCatalogPageNumber] = useState(1);
  const [catalogDetail, setCatalogDetail] = useState<CanonicalQuestionCatalogEntry | null>(null);
  const catalogDetailRequestRef = useRef(0);
  const skipDraftHydrationRef = useRef<string | null>(null);
  const [selectionPreview, setSelectionPreview] = useState<CanonicalSelectionPreview | null>(null);
  const [selectionPreviewOperation, setSelectionPreviewOperation] = useState<SelectionPreviewOperation | null>(null);
  const [serverSelectionSummary, setServerSelectionSummary] = useState<CanonicalSelectionPreview["preview"] | null>(null);
  const [pendingSelectionIds, setPendingSelectionIds] = useState<string[]>([]);
  const [selectionDirty, setSelectionDirty] = useState(false);
  const [scopeOptionLabel, setScopeOptionLabel] = useState<string | null>(null);
  const [scopeOptions, setScopeOptions] = useState<CanonicalAuditScopeOption[]>([]);
  // Before a draft exists, keep the scope tuple in local UI state only. The
  // server creates the opaque draft after the manager explicitly chooses the
  // supplier, provider scope, regulated target, and application type.
  const [pendingOrganizationId, setPendingOrganizationId] = useState("");
  const [pendingProviderScopeId, setPendingProviderScopeId] = useState("");
  const [pendingRegulatedTargetId, setPendingRegulatedTargetId] = useState("");
  const [pendingApplicationType, setPendingApplicationType] = useState<CanonicalApplicationType | "">("");
  // Catalog authority is returned by the server-owned scope selector. The
  // approved source is the only catalog that can be used for a new Audit.
  const [auditUsageClass, setAuditUsageClass] = useState<CanonicalQuestionUsageClass>("GOVERNED_OPERATIONAL");

  useEffect(() => {
    let cancelled = false;
    if (!backend.planningIntake) {
      setError("Planning intake commands are unavailable in this build profile.");
      return () => { cancelled = true; };
    }
    const requestedDraftId = new URLSearchParams(location.search).get("draftId");
    if (requestedDraftId && skipDraftHydrationRef.current === requestedDraftId) {
      // changeScope already received the authoritative server draft. The URL
      // update below intentionally re-enters this effect, but re-fetching the
      // same draft here can race a manager's immediate application-type
      // selection and restore the old type over the user's change.
      skipDraftHydrationRef.current = null;
      return () => { cancelled = true; };
    }
    const load = (async () => {
      if (!backend.canonicalCatalog) {
        throw new Error("Server-authorized audit scope selection is unavailable in this build profile.");
      }
      const optionPages: CanonicalAuditScopeOption[] = [];
      let cursor: string | undefined;
      do {
        const page = await backend.canonicalCatalog.listScopeOptions({ limit: 25, cursor });
        optionPages.push(...page.items);
        cursor = page.nextCursor ?? undefined;
      } while (cursor && optionPages.length < 1000);
      const options = { items: optionPages };
      if (!cancelled) {
        setScopeOptions(options.items);
        setAuditUsageClass("GOVERNED_OPERATIONAL");
        if (!requestedDraftId && options.items.length) {
          const first = options.items[0];
          setPendingOrganizationId(first.organizationId);
          setPendingProviderScopeId(first.providerScopeId);
          setPendingRegulatedTargetId(first.regulatedTargetId);
          setPendingApplicationType(inspectionTypeFor(first.inspectionTypes));
        }
      }
      if (requestedDraftId) {
        const loadedDraft = await backend.planningIntake.getDraft({ draftId: requestedDraftId });
        let loadedUsageClass: CanonicalQuestionUsageClass | undefined;
        if (!cancelled) {
          // Catalog versions are globally unique, so the server-enumerated
          // option is the authoritative usage-class pin for a saved draft.
          // Do not trust or rewrite a client-side usage value while resuming.
          loadedUsageClass = options.items.find((option) => option.catalogVersion === loadedDraft.catalogVersion)?.usageClass;
          const matchingOption = options.items.find((option) => option.catalogVersion === loadedDraft.catalogVersion &&
            option.usageClass === loadedUsageClass &&
            option.organizationId === loadedDraft.organizationId &&
            option.providerScopeId === loadedDraft.providerScopeId &&
            option.regulatedTargetId === loadedDraft.regulatedTargetId);
          if (!matchingOption) {
            throw new Error("The saved Planning draft no longer has an exact authorized catalog/scope/target option.");
          }
          setAuditUsageClass(matchingOption.usageClass);
          loadedUsageClass = matchingOption.usageClass;
        }
        return { draft: loadedDraft, usageClass: loadedUsageClass ?? "GOVERNED_OPERATIONAL" as CanonicalQuestionUsageClass };
      }
      // A new audit remains an uncommitted setup until the manager explicitly
      // chooses an authorized organization/provider scope/target.
      return { draft: null, usageClass: "GOVERNED_OPERATIONAL" as CanonicalQuestionUsageClass };
    })();
    void load.then(({ draft: loaded, usageClass }) => {
      if (!cancelled) {
        setAuditUsageClass(usageClass);
        if (loaded) {
          setDraft(loaded);
          setValues(formValuesFor(loaded));
          setPendingSelectionIds([...(loaded.selectedQuestionVersionIds ?? [])]);
          if (loaded.selectionDigest && loaded.formDistribution && loaded.domainDistribution && loaded.estimatedResourceRequirement !== undefined) {
            setServerSelectionSummary({
              selectionDigest: loaded.selectionDigest,
              selectedQuestionVersionIds: [...(loaded.selectedQuestionVersionIds ?? [])],
              selectedCount: loaded.selectedQuestionVersionIds?.length ?? 0,
              catalogVersion: loaded.catalogVersion ?? "",
              usageClass,
              formDistribution: loaded.formDistribution,
              domainDistribution: loaded.domainDistribution,
              estimatedResourceRequirement: loaded.estimatedResourceRequirement,
            });
          }
          setSelectionDirty(false);
          if (!scopeOptionLabel && loaded.organizationName) setScopeOptionLabel(loaded.organizationName);
        }
      }
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend, location.search]);

  useEffect(() => {
    if (!catalogDetail) return;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") closeCatalogDetail();
    };
    document.addEventListener("keydown", closeOnEscape);
    return () => document.removeEventListener("keydown", closeOnEscape);
  }, [catalogDetail]);

  function closeCatalogDetail() {
    // A dossier opens immediately with the visible row and is then replaced
    // by the server projection. In a slower public environment, that request
    // can resolve after the Manager has already closed the dossier. Advance
    // the request generation so a stale response cannot reopen the overlay
    // and intercept the next catalog interaction.
    catalogDetailRequestRef.current += 1;
    setCatalogDetail(null);
  }

  async function changeScope(option: (typeof scopeOptions)[number], requestedApplicationType?: CanonicalApplicationType) {
    if (!backend.planningIntake || (values && option.providerScopeId === values.providerScopeId && option.regulatedTargetId === values.regulatedTargetId)) return;
    setAuditUsageClass(option.usageClass);
    setBusy(true);
    setError(null);
    try {
      const nextValues: PlanningIntakeDraftValues = {
        ...(values ? commandValuesFor(values) : {
          organizationId: "",
          organizationName: "",
          applicationType: requestedApplicationType ?? inspectionTypeFor(option.inspectionTypes),
          domain: "Cabin Safety",
          inspectionCategory: "Routine / Announced" as const,
          noticePolicy: "ADVANCE" as const,
          purpose: "",
          triggerType: "Department Manager initiated",
          riskCategory: "Configured inspection risk",
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
        applicationType: requestedApplicationType ?? inspectionTypeFor(option.inspectionTypes),
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
      skipDraftHydrationRef.current = replacement.id;
      setValues(formValuesFor(replacement));
      setPendingSelectionIds([]);
      setSelectionDirty(false);
      setSelectionPreview(null);
      setSelectionPreviewOperation(null);
      setServerSelectionSummary(null);
      setScopeOptionLabel(`${option.organizationName} · ${option.providerTypeLabel} · ${option.targetLabel}`);
      setCatalogSearch("");
      setCatalogFormCode([]);
      setCatalogDomain([]);
      setCatalogTopic([]);
      setCatalogRiskBand([]);
      setCatalogSourceGapState("");
      setCatalogChecklistFocus([]);
      setCatalogRecommendationState("");
      setCatalogSelectedFilter("all");
      closeCatalogDetail();
      resetCatalogPage();
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

  function changeApplicationType(applicationType: string) {
    if (!values) return;
    if ((values.selectedQuestionVersionIds ?? []).length > 0 || pendingSelectionIds.length > 0) {
      setError("Clear the exact question selection before changing the application type.");
      return;
    }
    update("applicationType", applicationType);
    closeCatalogDetail();
    resetCatalogPage();
    setStatus(`Checklist suggestions will be evaluated for ${catalogValueLabel(applicationType)} and its matching prior-audit history.`);
  }

  useEffect(() => {
    if (step !== 4 || !values || !values.catalogVersion || !backend.canonicalCatalog) return;
    const controller = new AbortController();
    setCatalogBusy(true);
    void backend.canonicalCatalog.listCatalog({
      catalogVersion: values.catalogVersion,
      usageClass: auditUsageClass,
      search: catalogSearch || undefined,
      formCode: catalogFormCode || undefined,
      domain: catalogDomain || undefined,
      topic: catalogTopic || undefined,
      riskBand: catalogRiskBand || undefined,
      sourceGapState: catalogSourceGapState || undefined,
      checklistFocus: catalogChecklistFocus.length ? catalogChecklistFocus : undefined,
      recommendationState: catalogRecommendationState || undefined,
      selected: catalogSelectedFilter,
      scopeId: values.scopeDraftId || undefined,
      applicationType: values.applicationType as CanonicalApplicationType,
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
  }, [auditUsageClass, backend, catalogChecklistFocus, catalogCursor, catalogDomain, catalogFormCode, catalogRecommendationState, catalogRiskBand, catalogSearch, catalogSelectedFilter, catalogSourceGapState, catalogTopic, step, values?.applicationType, values?.catalogVersion, values?.scopeDraftId]);

  const selectionSummary = useMemo(() => {
    const selected = new Set(pendingSelectionIds);
    const serverSummary = serverSelectionSummary && serverSelectionSummary.selectedQuestionVersionIds.length === selected.size &&
      serverSelectionSummary.selectedQuestionVersionIds.every((id) => selected.has(id)) ? serverSelectionSummary : null;
    return {
      // Distribution and resource values are displayable only after the
      // canonical selection receipt is returned. Local catalog rows may be
      // incomplete or adversarial and are never treated as authoritative.
      formDistribution: serverSummary?.formDistribution ?? {},
      domainDistribution: serverSummary?.domainDistribution ?? {},
      complete: Boolean(serverSummary),
      estimatedResourceRequirement: serverSummary?.estimatedResourceRequirement,
    };
  }, [pendingSelectionIds, serverSelectionSummary]);

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

  function setPendingSelection(next: string[]) {
    const normalized = [...new Set(next)];
    setPendingSelectionIds(normalized);
    setSelectionDirty(true);
    setSelectionPreview(null);
    setSelectionPreviewOperation(null);
    setStatus("Selection changes are staged locally. Preview and confirm the exact batch before continuing.");
  }

  async function stageAllMatchingQuestions(recommendationOverride?: string) {
    if (busy || !values?.catalogVersion || !backend.canonicalCatalog) return;
    setBusy(true);
    setError(null);
    try {
      const ids: string[] = [];
      const seenIds = new Set<string>();
      const seenCursors = new Set<string>();
      let cursor: string | undefined;
      do {
        const page = await backend.canonicalCatalog.listCatalog({
          catalogVersion: values.catalogVersion,
          usageClass: auditUsageClass,
          search: catalogSearch || undefined,
          formCode: catalogFormCode.length ? catalogFormCode : undefined,
          domain: catalogDomain.length ? catalogDomain : undefined,
          topic: catalogTopic.length ? catalogTopic : undefined,
          riskBand: catalogRiskBand.length ? catalogRiskBand : undefined,
          sourceGapState: catalogSourceGapState || undefined,
          checklistFocus: catalogChecklistFocus.length ? catalogChecklistFocus : undefined,
          recommendationState: recommendationOverride || catalogRecommendationState || undefined,
          selected: catalogSelectedFilter,
          scopeId: values.scopeDraftId || undefined,
          applicationType: values.applicationType as CanonicalApplicationType,
          cursor,
          limit: 100,
        });
        for (const entry of page.items) {
          if (entry.canSelect && !seenIds.has(entry.questionVersionId)) {
            seenIds.add(entry.questionVersionId);
            ids.push(entry.questionVersionId);
          }
        }
        const nextCursor = page.nextCursor ?? undefined;
        if (nextCursor && seenCursors.has(nextCursor)) {
          throw new Error("Catalog pagination repeated a cursor while staging the exact question set.");
        }
        if (nextCursor) seenCursors.add(nextCursor);
        cursor = nextCursor;
      } while (cursor);
      if (!ids.length) throw new Error("No selectable questions match the current server-authorized filters.");
      const currentSelection = values.selectedQuestionVersionIds ?? [];
      setPendingSelection([...currentSelection, ...ids]);
      setStatus(`${ids.length.toLocaleString("en-US")} ${recommendationOverride ? "AI-suggested" : "eligible"} questions staged locally; commits run in batches of at most ${selectionBatchLimit}.`);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  async function previewQuestionSelection() {
    if (busy || !values || !draft || !backend.canonicalCatalog) return;
    if (!values.scopeDraftId) {
      setError("The server did not return a canonical scope identity; reload this draft before selecting questions.");
      return;
    }
    const batch = nextSelectionBatch(values.selectedQuestionVersionIds ?? [], pendingSelectionIds);
    if (!batch) {
      setSelectionDirty(false);
      setStatus("The staged selection already matches the server-owned exact selection.");
      return;
    }
    setBusy(true);
    try {
      const expectedSelectionDigest = values.selectionDigest || await selectionDigestFor([]);
      const previewOperationId = operationId(`SCOPE-${draft.id}-PREVIEW`);
      const previewReceipt = await backend.canonicalCatalog.previewSelection({
        scopeId: values.scopeDraftId,
        operationId: previewOperationId,
        idempotencyKey: previewOperationId,
			 expectedSelectionDigest,
			 questionVersionIds: batch.questionVersionIds,
			 operationKind: batch.operationKind,
        usageClass: auditUsageClass,
        filter: {},
      });
      setSelectionPreview(previewReceipt);
      setSelectionPreviewOperation({ operationId: previewOperationId, ...batch });
      setError(null);
      setStatus(`Exact selection preview ready · ${batch.operationKind} ${batch.questionVersionIds.length} · ${previewReceipt.preview.selectedCount.toLocaleString("en-US")} of ${pendingSelectionIds.length.toLocaleString("en-US")} target questions after commit.`);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  async function confirmQuestionSelection() {
    if (busy || !values || !draft || !backend.canonicalCatalog || !selectionPreview || !selectionPreviewOperation) return;
    if (!values.scopeDraftId) {
      setError("The server did not return a canonical scope identity; reload this draft before selecting questions.");
      return;
    }
    setBusy(true);
    try {
      const expectedSelectionDigest = values.selectionDigest || await selectionDigestFor([]);
      const operationIdValue = operationId(`SCOPE-${draft.id}-COMMIT`);
      const receipt = await backend.canonicalCatalog.commitSelection({
        scopeId: values.scopeDraftId,
        operationId: operationIdValue,
        previewOperationId: selectionPreviewOperation.operationId,
        idempotencyKey: operationIdValue,
			 expectedSelectionDigest,
			 questionVersionIds: selectionPreviewOperation.questionVersionIds,
			 operationKind: selectionPreviewOperation.operationKind,
        usageClass: auditUsageClass,
        filter: {},
      });
      setValues((current) => current ? {
        ...current,
        selectedQuestionVersionIds: receipt.selection.selectedQuestionVersionIds,
        selectionDigest: receipt.selection.selectionDigest,
        estimatedResourceRequirement: receipt.selection.estimatedResourceRequirement,
        formDistribution: receipt.selection.formDistribution,
        domainDistribution: receipt.selection.domainDistribution,
      } : current);
      setServerSelectionSummary(receipt.selection);
      setSelectionPreview(null);
      setSelectionPreviewOperation(null);
      setError(null);
      const remainingBatch = nextSelectionBatch(receipt.selection.selectedQuestionVersionIds, pendingSelectionIds);
      if (remainingBatch) {
        setSelectionDirty(true);
        const remaining = Math.max(0, pendingSelectionIds.length - receipt.selection.selectedCount);
        setStatus(`Exact batch committed · ${receipt.selection.selectedCount.toLocaleString("en-US")} of ${pendingSelectionIds.length.toLocaleString("en-US")} selected · ${remaining.toLocaleString("en-US")} remaining.`);
      } else {
        setPendingSelectionIds(receipt.selection.selectedQuestionVersionIds);
        setSelectionDirty(false);
        setStatus(`Exact question selection committed · ${receipt.selection.selectedCount.toLocaleString("en-US")} selected · ${receipt.selection.selectionDigest}`);
      }
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  function toggleQuestion(questionVersionId: string) {
    const selected = pendingSelectionIds.includes(questionVersionId);
    setPendingSelection(selected ? pendingSelectionIds.filter((id) => id !== questionVersionId) : [...pendingSelectionIds, questionVersionId]);
  }

  async function openCatalogDetail(question: CanonicalQuestionCatalogEntry) {
    const requestGeneration = ++catalogDetailRequestRef.current;
    setCatalogDetail(question);
    if (!values || !backend.canonicalCatalog) return;
    try {
      const detail = await backend.canonicalCatalog.getQuestion({
        catalogVersion: values.catalogVersion ?? "",
        usageClass: auditUsageClass,
        questionVersionId: question.questionVersionId,
        scopeId: values.scopeDraftId || undefined,
        applicationType: values.applicationType as CanonicalApplicationType,
      });
      if (requestGeneration === catalogDetailRequestRef.current) setCatalogDetail(detail);
    } catch (cause) {
      if (requestGeneration === catalogDetailRequestRef.current) setError(errorMessage(cause));
    }
  }

  async function saveDraft(nextValues = values): Promise<PlanningIntakeDraftView | null> {
    if (!backend.planningIntake || !draft || !nextValues) return null;
    if (selectionDirty) throw new Error("Confirm the exact question selection before saving this Planning draft.");
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
    if (direction > 0 && step === 4 && selectionDirty) {
      setError("Preview and confirm the exact question selection before continuing.");
      return;
    }
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
  const stage = stageFromStep(step);
  const stageDefinition = stageDefinitions[stage - 1] ?? stageDefinitions[0];
  const selectedScopeOption = values
    ? scopeOptions.find((item) => item.organizationId === values.organizationId && item.providerScopeId === values.providerScopeId && item.regulatedTargetId === values.regulatedTargetId) ?? null
    : null;
  const supplierOptions = [...new Map(scopeOptions.map((item) => [item.organizationId, item])).values()];
  const selectedSupplierOptions = values ? scopeOptions.filter((item) => item.organizationId === values.organizationId) : [];
  const providerScopeOptions = [...new Map(selectedSupplierOptions.map((item) => [item.providerScopeId, item])).values()];
  const regulatedTargetOptions = selectedSupplierOptions.filter((item) => item.providerScopeId === values?.providerScopeId);
  const pendingProviderOptions = scopeOptions.filter((item) => item.organizationId === pendingOrganizationId)
    .filter((item, index, all) => all.findIndex((candidate) => candidate.providerScopeId === item.providerScopeId) === index);
  const pendingTargetOptions = scopeOptions.filter((item) => item.organizationId === pendingOrganizationId && item.providerScopeId === pendingProviderScopeId);
  const pendingScopeOption = scopeOptions.find((item) => item.organizationId === pendingOrganizationId && item.providerScopeId === pendingProviderScopeId && item.regulatedTargetId === pendingRegulatedTargetId) ?? null;
  const pendingCanOpen = Boolean(pendingScopeOption && pendingApplicationType !== "" && pendingScopeOption.inspectionTypes.includes(pendingApplicationType));

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel={`New Audit Wizard ${step}`}>
      <div className="planning-intake-page" data-draft-id={draft?.id} data-testid={draft ? "new-audit-wizard-page" : undefined}>
        <header className="planning-intake-header workbench-page-header">
          <p className="eyebrow">Department planning intake</p>
          <h1>New Inspection</h1>
          <p>Create a governed Planning item. An executable Audit is created only after the accepted release and confirmation stage.</p>
        </header>
        <ol aria-label="Planning intake steps" className="planning-intake-steps">
          {stageDefinitions.map((item) => <li aria-current={item.number === stage ? "step" : undefined} className={item.number === stage ? "is-current" : item.number < stage ? "is-complete" : ""} key={item.number}><span>{item.number}</span><b>{item.title}</b><small>{item.description}</small></li>)}
        </ol>
        <CommandError message={error} />
        {status ? <p className="planning-intake-status" role="status">{status}</p> : null}
        <section aria-label="Planning intake form" className="planning-intake-form">
          <header><span>Stage {stage} of 3 · substep {step} of 5</span><h2>{stageDefinition.title} — Step {step} of 5 — {definition.title}</h2></header>
          {!values ? <div className="planning-intake-fields">
            <label>Supplier / organization
              <select aria-label="Supplier / organization" disabled={busy || !supplierOptions.length} value={pendingOrganizationId} onChange={(event) => {
                const organizationId = event.target.value;
                const firstProvider = scopeOptions.find((item) => item.organizationId === organizationId);
                setPendingOrganizationId(organizationId);
                setPendingProviderScopeId(firstProvider?.providerScopeId ?? "");
                setPendingRegulatedTargetId(firstProvider?.regulatedTargetId ?? "");
                setPendingApplicationType(firstProvider ? inspectionTypeFor(firstProvider.inspectionTypes) : "");
              }}>{supplierOptions.map((item) => <option key={item.organizationId} value={item.organizationId}>{item.organizationName}</option>)}</select>
            </label>
            <label>Provider scope
              <select aria-label="Provider scope" disabled={busy || !pendingProviderOptions.length} value={pendingProviderScopeId} onChange={(event) => {
                const providerScopeId = event.target.value;
                const firstTarget = scopeOptions.find((item) => item.organizationId === pendingOrganizationId && item.providerScopeId === providerScopeId);
                setPendingProviderScopeId(providerScopeId);
                setPendingRegulatedTargetId(firstTarget?.regulatedTargetId ?? "");
                setPendingApplicationType(firstTarget ? inspectionTypeFor(firstTarget.inspectionTypes) : "");
              }}>{pendingProviderOptions.map((item) => <option key={item.providerScopeId} value={item.providerScopeId}>{item.providerTypeLabel} · {item.providerScopeId}</option>)}</select>
            </label>
            <label>Regulated target
              <select aria-label="Regulated target" disabled={busy || !pendingTargetOptions.length} value={pendingRegulatedTargetId} onChange={(event) => {
                const regulatedTargetId = event.target.value;
                const option = scopeOptions.find((item) => item.organizationId === pendingOrganizationId && item.providerScopeId === pendingProviderScopeId && item.regulatedTargetId === regulatedTargetId);
                setPendingRegulatedTargetId(regulatedTargetId);
                setPendingApplicationType(option ? inspectionTypeFor(option.inspectionTypes) : "");
              }}>{pendingTargetOptions.map((item) => <option key={item.regulatedTargetId} value={item.regulatedTargetId}>{item.targetLabel}</option>)}</select>
            </label>
            <label>Application Type
              <select aria-label="Application Type" disabled={busy || !pendingScopeOption?.inspectionTypes.length} value={pendingApplicationType} onChange={(event) => setPendingApplicationType(event.target.value as CanonicalApplicationType)}>{(pendingScopeOption?.inspectionTypes ?? []).map((inspectionType) => <option key={inspectionType} value={inspectionType}>{catalogValueLabel(inspectionType)}</option>)}</select>
              <small>{(pendingScopeOption?.inspectionTypes.length ?? 0) > 1 ? "Recommendations and prior-audit history will follow this audit type." : pendingScopeOption ? `Only ${catalogValueLabel(pendingApplicationType || pendingScopeOption.inspectionTypes[0] || "")} is authorized for this supplier/provider scope.` : "Choose a supplier, provider scope, and regulated target first."}</small>
            </label>
            <div className="planning-intake-notice" role="note"><b>Assign the supplier before opening the Audit</b><span>{pendingScopeOption ? `${pendingScopeOption.organizationName} · ${pendingScopeOption.providerTypeLabel} · ${pendingScopeOption.targetLabel}` : "Choose the server-authorized supplier scope."}</span><small>The server will create the opaque Planning draft only after this exact supplier, provider scope, target, and type tuple is selected.</small></div>
            <button type="button" disabled={busy || !pendingCanOpen} onClick={() => { if (pendingScopeOption && pendingApplicationType) void changeScope(pendingScopeOption, pendingApplicationType); }}>Open audit setup for this supplier</button>
          </div> : null}
          {values && step === 1 ? <div className="planning-intake-fields">
            <label>Supplier / organization<select aria-label="Supplier / organization" disabled={busy || !supplierOptions.length} value={values.organizationId} onChange={(event) => { const option = scopeOptions.find((item) => item.organizationId === event.target.value); if (option) void changeScope(option); }}>{supplierOptions.map((item) => <option key={item.organizationId} value={item.organizationId}>{item.organizationName}</option>)}</select></label>
            <label>Provider scope<select aria-label="Provider scope" disabled={busy || !providerScopeOptions.length} value={values.providerScopeId} onChange={(event) => { const option = scopeOptions.find((item) => item.organizationId === values.organizationId && item.providerScopeId === event.target.value); if (option) void changeScope(option); }}>{providerScopeOptions.map((item) => <option key={item.providerScopeId} value={item.providerScopeId}>{item.providerTypeLabel} · {item.providerScopeId}</option>)}</select></label>
            <label>Regulated target<select aria-label="Regulated target" disabled={busy || !regulatedTargetOptions.length} value={values.regulatedTargetId} onChange={(event) => { const option = scopeOptions.find((item) => item.organizationId === values.organizationId && item.providerScopeId === values.providerScopeId && item.regulatedTargetId === event.target.value); if (option) void changeScope(option); }}>{regulatedTargetOptions.map((item) => <option key={item.regulatedTargetId} value={item.regulatedTargetId}>{item.targetLabel}</option>)}</select></label>
            <label>Application Type<select aria-label="Application Type" disabled={busy || !selectedScopeOption?.inspectionTypes.length} value={values.applicationType} onChange={(event) => changeApplicationType(event.target.value)}>{(selectedScopeOption?.inspectionTypes ?? []).map((inspectionType) => <option key={inspectionType} value={inspectionType}>{catalogValueLabel(inspectionType)}</option>)}</select><small>{(selectedScopeOption?.inspectionTypes.length ?? 0) > 1 ? "Choose the server-authorized audit type. Recommendations and prior-audit history follow this selection." : `Only ${catalogValueLabel(values.applicationType)} is authorized for this supplier/provider scope; choose another provider scope to use a different type.`}</small></label>
            <label>Domain<input aria-label="Domain" value={values.domain} onChange={(event) => update("domain", event.target.value)} /></label>
            <div className="planning-intake-notice" role="note"><b>Supplier assignment is part of this Audit</b><span>{scopeOptionLabel ?? `${values.organizationName || values.organizationId} · provider scope ${values.providerScopeId || "pending"} · target ${values.regulatedTargetId || "pending"}`}</span><small>Coordination and the executable Audit are later bound to this exact supplier organization and regulated target.</small></div>
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
              <div className="planning-intake-catalog-header"><div><span className="eyebrow">Approved source catalog · 1,310 immutable questions</span><h3>Build the checklist with guided suggestions</h3><p>AI enrichment is advisory only. Suggestions for <strong>{catalogValueLabel(values.applicationType)}</strong> use the selected supplier scope, prior locked Final history, risk, and recurrence. Keep or remove any valid question yourself.</p></div><span className="planning-intake-catalog-count" aria-live="polite">{pendingSelectionIds.length} selected{selectionDirty ? " · staged" : ""}</span></div>
              <div className="planning-intake-catalog-filters" aria-label="New Audit question search and filters">
                <label className="planning-intake-catalog-search">Search<input aria-label="New Audit question search" value={catalogSearch} onChange={(event) => { setCatalogSearch(event.target.value); resetCatalogPage(); }} placeholder="Search the question text, form, or identity" /></label>
                <CatalogFacetPicker ariaLabel="New Audit form filter" label="Form" options={catalogPage?.facets.forms ?? []} selected={catalogFormCode} onChange={(next) => { setCatalogFormCode(next); resetCatalogPage(); }} />
                <CatalogFacetPicker ariaLabel="New Audit domain filter" label="Domain" options={catalogPage?.facets.domains ?? []} selected={catalogDomain} onChange={(next) => { setCatalogDomain(next); resetCatalogPage(); }} />
                <CatalogFacetPicker ariaLabel="New Audit topic filter" label="Topic" options={catalogPage?.facets.topics ?? []} selected={catalogTopic} onChange={(next) => { setCatalogTopic(next); resetCatalogPage(); }} />
                <CatalogFacetPicker ariaLabel="New Audit risk tier filter" label="Risk tier" options={catalogPage?.facets.riskTiers ?? []} selected={catalogRiskBand} onChange={(next) => { setCatalogRiskBand(next); resetCatalogPage(); }} />
                <CatalogFacetPicker ariaLabel="New Audit checklist focus filter" label="Checklist focus" options={catalogPage?.facets.checklistFocuses ?? []} selected={catalogChecklistFocus} onChange={(next) => { setCatalogChecklistFocus(next); resetCatalogPage(); }} />
                <label>Source gap<select aria-label="New Audit source gap filter" value={catalogSourceGapState} onChange={(event) => { setCatalogSourceGapState(event.target.value); resetCatalogPage(); }}><option value="">Any source context</option><option value="OPTIONAL_ENRICHMENT_NOT_PROVIDED">Optional enrichment unavailable</option><option value="SOURCE_CONTEXT_INCOMPLETE">Source context incomplete</option></select></label>
                <label>Recommendation<select aria-label="New Audit recommendation filter" value={catalogRecommendationState} onChange={(event) => { setCatalogRecommendationState(event.target.value); resetCatalogPage(); }}><option value="">All advisory states</option>{(catalogPage?.facets.recommendationStates ?? []).map((option) => <option key={option.value} value={option.value}>{catalogValueLabel(option.value)} · {option.count.toLocaleString("en-US")}</option>)}</select></label>
                <label>Selected state<select aria-label="New Audit selected filter" value={catalogSelectedFilter} onChange={(event) => { setCatalogSelectedFilter(event.target.value as typeof catalogSelectedFilter); resetCatalogPage(); }}><option value="all">All questions</option><option value="selected">Selected in scope</option><option value="unselected">Not selected</option></select></label>
              </div>
              <div className="planning-intake-catalog-focus-note" role="note"><b>Suggestions for {catalogValueLabel(values.applicationType)}</b><span>Use “Stage suggested questions” to apply the existing deterministic AI advisory for this audit type. The full approved catalog remains available through the filters and “All advisory states”.</span></div>
              {catalogChecklistFocus.length ? <div className="planning-intake-catalog-focus-note" role="note"><b>{catalogChecklistFocus.map(catalogValueLabel).join(", ")}</b><span>Questions outside this focus are hidden from the default result. Clear the focus filter to inspect the full catalog.</span></div> : null}
              {catalogBusy ? <p role="status">Loading catalog page…</p> : null}
              {!catalogBusy && !catalogPage ? <p role="status">Catalog selection is unavailable in this build profile.</p> : null}
              {catalogPage ? <ul className="planning-intake-catalog-list">
                {catalogPage.items.map((question) => {
                  const checked = pendingSelectionIds.includes(question.questionVersionId);
                  return <li key={question.questionVersionId}><label><input aria-label={`Select ${question.formCode} ${question.ordinal}`} checked={checked} disabled={busy || !question.canSelect} onChange={() => toggleQuestion(question.questionVersionId)} title={!question.canSelect ? "This question is not selectable in the current server-authorized scope." : undefined} type="checkbox" /><span><b>{question.prompt ?? "Question prompt unavailable"}</b><small>{question.formCode} · item {question.ordinal} · {catalogValueLabel(question.aiAdvisory.domainCode)} · <code>{question.questionVersionId}</code></small><span className="planning-intake-question-meta"><em>{catalogValueLabel(question.aiAdvisory.riskTier)} risk</em><em>{catalogValueLabel(question.aiAdvisory.advisoryState)}</em>{question.aiAdvisory.recommendationReasonCodes.slice(0, 2).map((reason) => <em key={reason}>{catalogValueLabel(reason)}</em>)}</span></span></label><button className="planning-intake-question-detail" type="button" onClick={() => void openCatalogDetail(question)}>View dossier</button></li>;
                })}
              </ul> : null}
              <section aria-label="Selected question tray" className="planning-intake-selected-tray">
                <header><h4>Selected question tray</h4><span>{pendingSelectionIds.length} exact immutable versions</span></header>
                {pendingSelectionIds.length ? <><ul>{pendingSelectionIds.slice(0, selectedTrayRenderLimit).map((questionId) => <li key={questionId}><span>{questionId}</span><button type="button" disabled={busy} onClick={() => setPendingSelection(pendingSelectionIds.filter((id) => id !== questionId))}>Remove</button></li>)}</ul>{pendingSelectionIds.length > selectedTrayRenderLimit ? <p>{pendingSelectionIds.length - selectedTrayRenderLimit} additional exact identities remain staged; use the selected-state filter and pagination to inspect them without rendering all question bodies at once.</p> : null}</> : <p>No questions selected. Select at least one version to continue.</p>}
              </section>
              {catalogDetail ? <div className="planning-intake-dossier-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) closeCatalogDetail(); }}><section aria-label="Question dossier" aria-modal="true" className="planning-intake-question-dossier" role="dialog"><header><div><span className="eyebrow">Question dossier</span><h4>{catalogDetail.formCode} · item {catalogDetail.ordinal}</h4></div><button autoFocus type="button" onClick={closeCatalogDetail}>Close</button></header><p className="planning-intake-dossier-prompt">{catalogDetail.prompt ?? "Prompt unavailable in this profile."}</p><div className="planning-intake-question-meta"><em>{catalogValueLabel(catalogDetail.aiAdvisory.riskTier)} risk</em><em>{catalogValueLabel(catalogDetail.aiAdvisory.advisoryState)}</em>{catalogDetail.aiAdvisory.recommendationReasonCodes.map((reason) => <em key={reason}>{catalogValueLabel(reason)}</em>)}</div><p>{catalogDetail.aiAdvisory.previouslyVerifiedAt ? `Previously verified ${new Date(catalogDetail.aiAdvisory.previouslyVerifiedAt).toLocaleDateString("en-GB")}.` : "No prior locked Final verification is recorded for this question."}</p><dl><div><dt>Question version</dt><dd>{catalogDetail.questionVersionId}</dd></div><div><dt>Domain</dt><dd>{catalogValueLabel(catalogDetail.aiAdvisory.domainCode)}</dd></div><div><dt>Checklist focus</dt><dd>{catalogDetail.aiAdvisory.inspectionTypeCodes.map(catalogValueLabel).join(", ") || "Not classified"}</dd></div><div><dt>Reference</dt><dd>{catalogDetail.configuredReference ?? "Not configured"}</dd></div><div><dt>Expected evidence</dt><dd>{catalogDetail.expectedEvidence ?? "Not configured"}</dd></div><div><dt>Source context</dt><dd>{catalogDetail.aiAdvisory.externalApplicabilityUnresolved ? "Some applicability context is unresolved; this advisory does not block selection." : "Source context available"}</dd></div></dl></section></div> : null}
              {selectionPreview ? <p className="planning-intake-selection-preview" role="status">Preview: {selectionPreview.preview.selectedCount} selected · {selectionPreview.valid ? "ready to confirm" : selectionPreview.reason}</p> : null}
              <div className="planning-intake-selection-actions"><button type="button" disabled={busy || catalogBusy || !catalogPage} onClick={() => void stageAllMatchingQuestions("SUGGESTED_NOW")}>Stage suggested questions</button><button type="button" disabled={busy || catalogBusy || !catalogPage} onClick={() => void stageAllMatchingQuestions()}>Stage all matching eligible questions</button><button type="button" disabled={busy || !selectionDirty} onClick={() => void previewQuestionSelection()} title={!selectionDirty ? "Stage an Add, Remove, or tray change first." : undefined}>Preview next exact batch</button><button type="button" disabled={busy || !selectionPreview?.valid || !selectionPreviewOperation} onClick={() => void confirmQuestionSelection()} title={!selectionPreview?.valid ? "Preview the staged exact batch first." : undefined}>Confirm selection</button><button type="button" disabled={busy || !selectionDirty} onClick={() => { setPendingSelectionIds([...(values.selectedQuestionVersionIds ?? [])]); setSelectionDirty(false); setSelectionPreview(null); setSelectionPreviewOperation(null); setStatus("Staged selection changes were discarded."); }} title={!selectionDirty ? "There are no staged selection changes." : undefined}>Undo staged changes</button></div>
              <div className="planning-intake-catalog-pagination" aria-label="New Audit question pagination"><button disabled={catalogBusy || !catalogPreviousCursors.length} onClick={() => { const history = [...catalogPreviousCursors]; setCatalogCursor(history.pop()); setCatalogPreviousCursors(history); setCatalogPageNumber((value) => Math.max(1, value - 1)); }} type="button">Previous questions</button><button disabled={catalogBusy || (!catalogSearch && !catalogFormCode.length && !catalogDomain.length && !catalogTopic.length && !catalogRiskBand.length && !catalogSourceGapState && !catalogChecklistFocus.length && !catalogRecommendationState && catalogSelectedFilter === "all")} onClick={() => { setCatalogSearch(""); setCatalogFormCode([]); setCatalogDomain([]); setCatalogTopic([]); setCatalogRiskBand([]); setCatalogSourceGapState(""); setCatalogChecklistFocus([]); setCatalogRecommendationState(""); setCatalogSelectedFilter("all"); resetCatalogPage(); }} type="button">Clear filters</button><span aria-live="polite">{catalogPage?.totalCount ?? 0} matching questions · page {catalogPageNumber}</span><button disabled={catalogBusy || !catalogPage?.nextCursor} onClick={() => { if (!catalogPage?.nextCursor) return; setCatalogPreviousCursors((history) => [...history, catalogCursor ?? ""]); setCatalogCursor(catalogPage.nextCursor ?? undefined); setCatalogPageNumber((value) => value + 1); }} type="button">Next questions</button></div>
            </div>
            <label>Question Catalog Version<input aria-label="Question Catalog Version" readOnly value={values.catalogVersion ?? "Unavailable"} /></label>
            <label>Requested Budget<input aria-label="Requested Budget" min="0" type="number" value={values.requestedBudget} onChange={(event) => update("requestedBudget", event.target.value)} /></label>
            <label>Currency<select aria-label="Currency" value={values.currency} onChange={(event) => update("currency", event.target.value as PlanningIntakeDraftValues["currency"])}><option value="USD">USD</option><option value="EUR">EUR</option><option value="NAD">NAD</option></select></label>
            <div className="planning-intake-notice" role="note"><b>Finance Review is required even when the requested budget is zero.</b><span>The selection digest is retained with the Planning draft and cannot be replaced by a template identifier.</span></div>
          </div> : null}
          {values && step === 5 ? <div className="planning-intake-review">
            <dl><div><dt>Draft</dt><dd>{draft?.id}</dd></div><div><dt>Organization</dt><dd>{values.organizationName} · {values.organizationId}</dd></div><div><dt>Category</dt><dd>{values.inspectionCategory}</dd></div><div><dt>Purpose</dt><dd>{values.purpose || "Not provided"}</dd></div><div><dt>Planned work</dt><dd>{values.plannedDate} · {values.mode} · {values.location || "Location required"}</dd></div><div><dt>Question selection</dt><dd>{values.catalogVersion} · {values.selectedQuestionVersionIds?.length ?? 0} exact question versions · {values.selectionDigest || "Not frozen"}</dd></div><div><dt>Estimated resource requirement</dt><dd>{selectionSummary.estimatedResourceRequirement === undefined ? "Server summary unavailable; reload the exact selected scope." : `${selectionSummary.estimatedResourceRequirement} question-hours (server-derived from the exact selected set)`}</dd></div><div><dt>Form distribution</dt><dd>{selectionSummary.complete ? Object.entries(selectionSummary.formDistribution).map(([form, count]) => `${form}: ${count}`).join(" · ") || "None" : "Loading exact selected-question distribution…"}</dd></div><div><dt>Domain distribution</dt><dd>{selectionSummary.complete ? Object.entries(selectionSummary.domainDistribution).map(([domain, count]) => `${domain}: ${count}`).join(" · ") || "None" : "Loading exact selected-question distribution…"}</dd></div><div><dt>Requested budget</dt><dd>{values.requestedBudget} {values.currency}</dd></div><div><dt>Notice</dt><dd>{noticeLabel(values)}</dd></div></dl>
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
