import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
  type RefObject,
} from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { z } from "zod";

import { useApplicationRuntime } from "../../app/providers";
import type {
  CanonicalApplicationType,
  CanonicalAuditScopeOption,
  CanonicalQuestionCatalogEntry,
  CanonicalQuestionCatalogPage,
  CanonicalQuestionUsageClass,
  CanonicalSelectionDigest,
  PlanningIntakeDraftValues,
  PlanningIntakeDraftView,
  PlanningIntakeInspectionCategory,
} from "../../backend/backend";
import {
  CommandError,
  errorMessage,
  formatLocalDate,
  WorkspaceShell,
} from "../shared/workspace-shell";
import { catalogValueLabel } from "./planning-intake-formatters";

const stepDefinitions = [
  { number: 1, label: "Basics", description: "Choose the authorized inspection scope." },
  { number: 2, label: "Purpose", description: "Set the operational reason and notice consequence." },
  { number: 3, label: "Schedule", description: "Set when and where the inspection will take place." },
  { number: 4, label: "Checklist & budget", description: "Review suggested questions and resources." },
  { number: 5, label: "Review", description: "Confirm the Planning item before Finance." },
] as const;

const selectionBatchLimit = 500;
const defaultCatalogRecommendationState = "SUGGESTED_NOW";
const riskCategoryOptions = [
  "Configured inspection risk",
  "Safety-critical",
  "High operational",
  "Control assurance",
  "Review required",
] as const;

type SelectionOperationKind = "ADD" | "REMOVE";
type AutosaveState = "clean" | "dirty" | "saving" | "saved" | "error";
type FieldKey =
  | "organizationId"
  | "providerScopeId"
  | "regulatedTargetId"
  | "applicationType"
  | "inspectionCategory"
  | "purpose"
  | "riskCategory"
  | "plannedDate"
  | "location"
  | "selectedQuestionVersionIds"
  | "requestedBudget";
type FieldErrors = Partial<Record<FieldKey, string>>;

interface PlanningIntakeFormValues extends Omit<PlanningIntakeDraftValues, "requestedBudget"> {
  requestedBudget: string;
}

interface SelectionProgress {
  completed: number;
  total: number;
  error: string | null;
}

interface SelectionBatchOperation {
  signature: string;
  previewOperationId: string;
  commitOperationId: string;
}

type CatalogFacetOption = { value: string; count: number };

const requestedBudgetSchema = z
  .string()
  .trim()
  .min(1, "Requested budget is required")
  .transform((value) => Number(value))
  .refine((value) => Number.isFinite(value) && value >= 0, "Requested budget must be zero or greater");

const stepSchemas = {
  1: z.object({
    organizationId: z.string().min(1, "Organization is required"),
    applicationType: z.string().min(1, "Inspection type is required"),
    domain: z.string().min(1, "Inspection domain is required"),
  }),
  2: z.object({
    inspectionCategory: z.enum(["Routine / Announced", "Ad Hoc / Unannounced"]),
    purpose: z.string().trim().min(1, "Purpose is required"),
    riskCategory: z.string().trim().min(1, "Risk category is required"),
  }),
  3: z.object({
    plannedDate: z.string().min(1, "Planned date is required").regex(/^\d{4}-\d{2}-\d{2}$/, "Use YYYY-MM-DD for the planned date"),
    location: z.string().trim().min(1, "Location is required"),
  }),
  4: z.object({
    catalogVersion: z.string().min(1, "Question catalog is required"),
    selectedQuestionVersionIds: z.array(z.string()).min(1, "Select at least one question"),
    selectionDigest: z.string().min(1, "Question selection must be confirmed"),
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

function riskCategoryOptionsFor(value: string): string[] {
  return value && !riskCategoryOptions.includes(value as (typeof riskCategoryOptions)[number])
    ? [value, ...riskCategoryOptions]
    : [...riskCategoryOptions];
}

function readableLocalDate(value: string | undefined): string {
  if (!value) return "To be set";
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return value;
  try {
    return formatLocalDate(value);
  } catch {
    return value;
  }
}

function normalizeDateInput(value: string): string {
  const digits = value.replace(/\D/g, "").slice(0, 8);
  if (digits.length <= 4) return digits;
  if (digits.length <= 6) return `${digits.slice(0, 4)}-${digits.slice(4)}`;
  return `${digits.slice(0, 4)}-${digits.slice(4, 6)}-${digits.slice(6)}`;
}

function noticePolicyFor(category: PlanningIntakeInspectionCategory): PlanningIntakeDraftValues["noticePolicy"] {
  return category === "Ad Hoc / Unannounced" ? "WITHHELD" : "ADVANCE";
}

function noticeLabel(category: PlanningIntakeInspectionCategory): string {
  return category === "Ad Hoc / Unannounced" ? "Notice withheld" : "Advance notice";
}

function inspectionTypeFor(types: readonly CanonicalApplicationType[]): CanonicalApplicationType {
  const firstSupported = types.find((type) => [
    "RAMP",
    "CABIN",
    "RAMP_INSPECTION",
    "CABIN_INSPECTION",
    "CHANGE_APPROVAL",
    "DOCUMENT_AND_RECORD_REVIEW",
    "FOLLOW_UP",
    "INITIAL_CERTIFICATION",
    "ON_SITE_INSPECTION",
    "PERIODIC_SURVEILLANCE",
    "RENEWAL",
    "SPECIAL_PURPOSE",
  ].includes(type));
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
  if (removals.length) return { operationKind: "REMOVE", questionVersionIds: removals.slice(0, selectionBatchLimit) };
  const additions = desired.filter((id) => !currentSet.has(id));
  if (additions.length) return { operationKind: "ADD", questionVersionIds: additions.slice(0, selectionBatchLimit) };
  return null;
}

function selectionChangeCount(current: readonly string[], desired: readonly string[]): number {
  const currentSet = new Set(current);
  const desiredSet = new Set(desired);
  return current.filter((id) => !desiredSet.has(id)).length + desired.filter((id) => !currentSet.has(id)).length;
}

function formValuesFor(draft: PlanningIntakeDraftView): PlanningIntakeFormValues {
  return {
    ...draft,
    riskCategory: draft.riskCategory || "Configured inspection risk",
    selectedQuestionVersionIds: [...(draft.selectedQuestionVersionIds ?? [])],
    requestedBudget: String(draft.requestedBudget),
  };
}

function commandValuesFor(values: PlanningIntakeFormValues): PlanningIntakeDraftValues {
  const result = requestedBudgetSchema.safeParse(values.requestedBudget);
  if (!result.success) throw new Error(result.error.issues[0]?.message ?? "Requested budget is invalid");
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

function defaultDraftValues(option: CanonicalAuditScopeOption, applicationType: CanonicalApplicationType): PlanningIntakeDraftValues {
  return {
    organizationId: option.organizationId,
    organizationName: option.organizationName,
    applicationType,
    domain: "Cabin Safety",
    inspectionCategory: "Routine / Announced",
    noticePolicy: "ADVANCE",
    purpose: "",
    triggerType: "Department Manager initiated",
    riskCategory: "Configured inspection risk",
    plannedDate: "",
    mode: "On-site",
    location: "",
    catalogVersion: option.catalogVersion,
    scopeDraftId: "",
    selectionDigest: "",
    selectedQuestionVersionIds: [],
    providerScopeId: option.providerScopeId,
    regulatedTargetId: option.regulatedTargetId,
    requestedBudget: 0,
    currency: "USD",
  };
}

function FieldError({ id, message }: { id: string; message?: string }): ReactNode {
  return message ? <span className="planning-intake-field-error" id={id} role="alert">{message}</span> : null;
}

function RequiredMark(): ReactNode {
  return <span aria-hidden="true" className="planning-intake-required">*</span>;
}

function PlanningDateField({
  value,
  error,
  onBlur,
  onChange,
  onNext,
}: {
  value: string;
  error?: string;
  onBlur: () => void;
  onChange: (value: string) => void;
  onNext: () => void;
}) {
  const datePickerRef = useRef<HTMLInputElement | null>(null);

  const openDatePicker = () => {
    const picker = datePickerRef.current;
    if (!picker) return;

    picker.focus();
    if (typeof picker.showPicker === "function") {
      picker.showPicker();
      return;
    }

    picker.click();
  };

  return (
    <div className="planning-intake-date-control">
      <input
        aria-describedby={error ? "planning-intake-plannedDate-error" : undefined}
        aria-invalid={Boolean(error)}
        aria-label="Planned date"
        autoComplete="off"
        enterKeyHint="next"
        id="planning-intake-plannedDate"
        inputMode="text"
        maxLength={10}
        onBlur={onBlur}
        onChange={(event) => onChange(normalizeDateInput(event.target.value))}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            onNext();
          }
        }}
        placeholder="YYYY-MM-DD"
        type="text"
        value={value}
      />
      <button
        aria-label="Open planned date calendar"
        className="planning-intake-date-calendar"
        onClick={openDatePicker}
        type="button"
      >
        <span aria-hidden="true">📅</span>
      </button>
      <input
        aria-hidden="true"
        aria-label="Planned date picker"
        ref={datePickerRef}
        data-testid="planning-intake-date-picker"
        className="planning-intake-date-picker"
        tabIndex={-1}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            onNext();
          }
        }}
        type="date"
        value={value}
      />
    </div>
  );
}

function PlanningIntakeProgress({ step }: { step: number }) {
  return (
    <ol aria-label="Planning intake steps" className="planning-intake-steps">
      {stepDefinitions.map((definition) => (
        <li aria-current={definition.number === step ? "step" : undefined} className={definition.number === step ? "is-current" : definition.number < step ? "is-complete" : ""} key={definition.number}>
          <span>{definition.number < step ? "✓" : definition.number}</span><b>{definition.label}</b>
        </li>
      ))}
    </ol>
  );
}

function ValidationSummary({ errors, onFocus }: { errors: FieldErrors; onFocus: (field: FieldKey) => void }) {
  const entries = Object.entries(errors) as Array<[FieldKey, string]>;
  if (entries.length < 2) return null;
  return (
    <div className="planning-intake-error-summary" role="alert">
      <b>Review the highlighted fields</b>
      <ul>{entries.map(([field, message]) => <li key={field}><button type="button" onClick={() => onFocus(field)}>{message}</button></li>)}</ul>
    </div>
  );
}

function AutosaveIndicator({ state, error, onRetry }: { state: AutosaveState; error: string | null; onRetry: () => void }) {
  if (state === "error") return <span className="planning-intake-autosave is-error"><span role="alert">Couldn't save</span><button type="button" onClick={onRetry}>Retry</button>{error ? <small>{error}</small> : null}</span>;
  const label = state === "saving" ? "Saving…" : state === "dirty" ? "Not saved" : state === "saved" ? "Saved" : "Not saved";
  return <span className={`planning-intake-autosave is-${state}`}>{label}</span>;
}

interface BriefProps {
  values: PlanningIntakeFormValues | null;
  scopeOption: CanonicalAuditScopeOption | null;
  pendingScopeOption: CanonicalAuditScopeOption | null;
  pendingApplicationType: CanonicalApplicationType | "";
  selectedCount: number;
  autosaveState: AutosaveState;
  autosaveError: string | null;
  onRetry: () => void;
}

function InspectionBrief({ values, scopeOption, pendingScopeOption, pendingApplicationType, selectedCount, autosaveState, autosaveError, onRetry }: BriefProps) {
  const activeScopeOption = scopeOption ?? pendingScopeOption;
  const organization = values?.organizationName || pendingScopeOption?.organizationName || "Choose a supplier";
  const provider = activeScopeOption?.providerTypeLabel ?? (values ? "Authorized provider scope" : "Choose a provider scope");
  const target = activeScopeOption?.targetLabel ?? (values ? "Authorized regulated target" : "Choose a regulated target");
  const applicationType = catalogValueLabel(values?.applicationType || pendingApplicationType || "");
  const category = values?.inspectionCategory ?? "To be set";
  const notice = values ? noticeLabel(values.inspectionCategory) : "To be completed later";
  const plannedDate = readableLocalDate(values?.plannedDate);
  const mode = values?.mode || "To be set";
  const location = values?.location || "To be set";
  const state = values ? autosaveState : "clean";
  const details = (
    <dl className="planning-intake-brief__facts">
      <div><dt>Supplier / organization</dt><dd>{organization}</dd></div>
      <div><dt>Provider scope</dt><dd>{provider}</dd></div>
      <div><dt>Regulated target</dt><dd>{target}</dd></div>
      <div><dt>Inspection type</dt><dd>{applicationType || "To be set"}</dd></div>
      <div><dt>Inspection approach</dt><dd>{category}</dd></div>
      <div><dt>Notice policy</dt><dd>{notice}</dd></div>
      <div><dt>Planned date</dt><dd>{plannedDate}</dd></div>
      <div><dt>Mode</dt><dd>{mode}</dd></div>
      <div><dt>Location</dt><dd>{location}</dd></div>
      <div><dt>Questions selected</dt><dd>{selectedCount.toLocaleString("en-US")}</dd></div>
      <div><dt>Requested budget</dt><dd>{values ? `${values.requestedBudget || "0"} ${values.currency}` : "To be set"}</dd></div>
    </dl>
  );
  return (
    <aside aria-label="Inspection brief" className="planning-intake-brief">
      <div className="planning-intake-brief__desktop"><header><h2>Inspection brief</h2><AutosaveIndicator state={state} error={autosaveError} onRetry={onRetry} /></header>{details}</div>
      <details className="planning-intake-brief__mobile"><summary><span>Inspection brief · {organization}</span><AutosaveIndicator state={state} error={autosaveError} onRetry={onRetry} /></summary>{details}</details>
    </aside>
  );
}

function CatalogFacetPicker({ label, ariaLabel, options, selected, onChange }: { label: string; ariaLabel: string; options: CatalogFacetOption[]; selected: string[]; onChange: (next: string[]) => void }) {
  return (
    <details className="planning-intake-facet-picker">
      <summary aria-label={ariaLabel}>{selected.length ? `${label} · ${selected.length} selected` : `${label} · Any`}</summary>
      <div className="planning-intake-facet-options" role="group" aria-label={ariaLabel}>{options.length ? options.map((option) => <label key={option.value}><input checked={selected.includes(option.value)} onChange={(event) => onChange(event.target.checked ? [...selected, option.value] : selected.filter((value) => value !== option.value))} type="checkbox" /><span>{catalogValueLabel(option.value)}</span><small>{option.count.toLocaleString("en-US")}</small></label>) : <p>No values in the current result set.</p>}</div>
    </details>
  );
}

function useDialogFocus(dialogRef: RefObject<HTMLElement | null>, onClose: () => void, returnFocusRef: RefObject<HTMLElement | null>) {
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;
  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return undefined;
    const focusableSelector = "button:not(:disabled), [href], input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex=\"-1\"])";
    dialog.querySelector<HTMLElement>("[data-autofocus]")?.focus();
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") { event.preventDefault(); onCloseRef.current(); return; }
      if (event.key !== "Tab") return;
      const focusable = [...dialog.querySelectorAll<HTMLElement>(focusableSelector)];
      if (!focusable.length) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
    };
    dialog.addEventListener("keydown", onKeyDown);
    return () => {
      dialog.removeEventListener("keydown", onKeyDown);
      const returnFocus = returnFocusRef.current;
      window.setTimeout(() => returnFocus?.focus(), 0);
    };
  }, [dialogRef, returnFocusRef]);
}

function QuestionDossier({ question, onClose, returnFocusRef }: { question: CanonicalQuestionCatalogEntry; onClose: () => void; returnFocusRef: RefObject<HTMLElement | null> }) {
  const dialogRef = useRef<HTMLElement | null>(null);
  useDialogFocus(dialogRef, onClose, returnFocusRef);
  return (
    <div className="planning-intake-dossier-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose(); }}>
      <section aria-label="Question dossier" aria-modal="true" className="planning-intake-question-dossier" ref={dialogRef} role="dialog">
        <header><div><span className="planning-intake-dialog-kicker">Question details</span><h2>{question.formCode} · item {question.ordinal}</h2></div><button data-autofocus type="button" onClick={onClose}>Close</button></header>
        <p className="planning-intake-dossier-prompt">{question.prompt ?? "Question prompt unavailable in this profile."}</p>
        <div className="planning-intake-question-meta"><span>{catalogValueLabel(question.aiAdvisory.riskTier)} risk</span><span>{catalogValueLabel(question.recommendation.recommendationState)}</span><span>{question.recommendation.historyCount.toLocaleString("en-US")} comparable Audits</span>{question.recommendation.signalCodes.map((reason) => <span key={reason}>{catalogValueLabel(reason)}</span>)}</div>
        <p>{question.recommendation.rationale}</p>
        <p>{question.recommendation.canDefer ? "This optional question may be deferred with an explicit manager reason." : "This question remains protected by the server recommendation floor."}</p>
        <p>{question.aiAdvisory.previouslyVerifiedAt ? `Previously verified ${new Date(question.aiAdvisory.previouslyVerifiedAt).toLocaleDateString("en-GB")}.` : "No prior locked Final verification is recorded for this question."}</p>
        <details className="planning-intake-technical-details"><summary>Technical question details</summary><dl><div><dt>Question version</dt><dd>{question.questionVersionId}</dd></div><div><dt>Domain</dt><dd>{catalogValueLabel(question.aiAdvisory.domainCode)}</dd></div><div><dt>Checklist focus</dt><dd>{question.aiAdvisory.inspectionTypeCodes.map(catalogValueLabel).join(", ") || "Not classified"}</dd></div><div><dt>Reference</dt><dd>{question.configuredReference ?? "Not configured"}</dd></div><div><dt>Expected evidence</dt><dd>{question.expectedEvidence ?? "Not configured"}</dd></div><div><dt>Source context</dt><dd>{question.aiAdvisory.externalApplicabilityUnresolved ? "Some applicability context is unresolved; this advisory does not block selection." : "Source context available"}</dd></div></dl></details>
      </section>
    </div>
  );
}

function SelectionReviewDialog({ selectedCount, additions, removals, total, progress, onConfirm, onClose, onRetry, returnFocusRef, busy }: { selectedCount: number; additions: number; removals: number; total: number; progress: SelectionProgress; onConfirm: () => void; onClose: () => void; onRetry: () => void; returnFocusRef: RefObject<HTMLElement | null>; busy: boolean }) {
  const dialogRef = useRef<HTMLElement | null>(null);
  useDialogFocus(dialogRef, onClose, returnFocusRef);
  return (
    <div className="planning-intake-dossier-backdrop" role="presentation">
      <section aria-label="Review selection" aria-modal="true" className="planning-intake-selection-dialog" ref={dialogRef} role="dialog">
        <header><div><span className="planning-intake-dialog-kicker">Selection review</span><h2>Review selection</h2></div><button data-autofocus type="button" onClick={onClose}>Close</button></header>
        <p>Confirm one selection decision. Required bounded batches run behind this review and every immutable receipt remains server-owned.</p>
        <dl className="planning-intake-selection-dialog__facts"><div><dt>Questions selected</dt><dd>{selectedCount.toLocaleString("en-US")}</dd></div><div><dt>Additions</dt><dd>{additions.toLocaleString("en-US")}</dd></div><div><dt>Removals</dt><dd>{removals.toLocaleString("en-US")}</dd></div></dl>
        {total ? <p className="planning-intake-selection-progress" role="status">{progress.completed.toLocaleString("en-US")} of {total.toLocaleString("en-US")} confirmed</p> : <p role="status">No selection changes are waiting for confirmation.</p>}
        {progress.error ? <p className="planning-intake-field-error" role="alert">{progress.error}</p> : null}
        <footer><button type="button" onClick={onClose} disabled={busy}>Back to checklist</button>{progress.error ? <button type="button" onClick={onRetry} disabled={busy}>Retry confirmation</button> : <button data-autofocus="confirm" type="button" onClick={onConfirm} disabled={busy || !total}>Confirm selection</button>}</footer>
      </section>
    </div>
  );
}

export function NewAuditWizardPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("manager") ?? runtime.backend, [runtime]);
  const navigate = useNavigate();
  const location = useLocation();
  const requestedDraftId = new URLSearchParams(location.search).get("draftId");
  const requestedStep = stepFromPath(location.pathname);
  // A queryless deep link cannot load server-owned downstream state. Keep the
  // requested URL stable while presenting the only safe entry state; the
  // first valid Continue still creates the draft and advances normally.
  const step = requestedDraftId ? requestedStep : 1;
  const [draft, setDraft] = useState<PlanningIntakeDraftView | null>(null);
  const [values, setValues] = useState<PlanningIntakeFormValues | null>(null);
  const [scopeOptions, setScopeOptions] = useState<CanonicalAuditScopeOption[]>([]);
  const [pendingOrganizationId, setPendingOrganizationId] = useState("");
  const [pendingProviderScopeId, setPendingProviderScopeId] = useState("");
  const [pendingRegulatedTargetId, setPendingRegulatedTargetId] = useState("");
  const [pendingApplicationType, setPendingApplicationType] = useState<CanonicalApplicationType | "">("");
  const [auditUsageClass, setAuditUsageClass] = useState<CanonicalQuestionUsageClass>("GOVERNED_OPERATIONAL");
  const [serverError, setServerError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [busy, setBusy] = useState(false);
  const routeRedirecting = false;
  const [autosaveState, setAutosaveState] = useState<AutosaveState>("clean");
  const [autosaveError, setAutosaveError] = useState<string | null>(null);
  const [catalogPage, setCatalogPage] = useState<CanonicalQuestionCatalogPage | null>(null);
  const [catalogBusy, setCatalogBusy] = useState(false);
  const [historyDeferredQuestions, setHistoryDeferredQuestions] = useState<CanonicalQuestionCatalogEntry[]>([]);
  const [historyDeferredBusy, setHistoryDeferredBusy] = useState(false);
  const [historyDeferredError, setHistoryDeferredError] = useState<string | null>(null);
  const [catalogSearch, setCatalogSearch] = useState("");
  const [catalogFormCode, setCatalogFormCode] = useState<string[]>([]);
  const [catalogDomain, setCatalogDomain] = useState<string[]>([]);
  const [catalogTopic, setCatalogTopic] = useState<string[]>([]);
  const [catalogRiskBand, setCatalogRiskBand] = useState<string[]>([]);
  const [catalogSourceGapState, setCatalogSourceGapState] = useState("");
  const [catalogChecklistFocus, setCatalogChecklistFocus] = useState<string[]>([]);
  const [catalogRecommendationState, setCatalogRecommendationState] = useState(defaultCatalogRecommendationState);
  const [catalogSelectedFilter, setCatalogSelectedFilter] = useState<"all" | "selected" | "unselected">("all");
  const [catalogCursor, setCatalogCursor] = useState<string | undefined>();
  const [catalogPreviousCursors, setCatalogPreviousCursors] = useState<string[]>([]);
  const [catalogPageNumber, setCatalogPageNumber] = useState(1);
  const [catalogDetail, setCatalogDetail] = useState<CanonicalQuestionCatalogEntry | null>(null);
  const [pendingSelectionIds, setPendingSelectionIds] = useState<string[]>([]);
  const [selectionDirty, setSelectionDirty] = useState(false);
  const [serverSelectionSummary, setServerSelectionSummary] = useState<CanonicalSelectionDigest | null>(null);
  const [selectionReviewOpen, setSelectionReviewOpen] = useState(false);
  const [selectionProgress, setSelectionProgress] = useState<SelectionProgress>({ completed: 0, total: 0, error: null });
  const draftRef = useRef<PlanningIntakeDraftView | null>(null);
  const valuesRef = useRef<PlanningIntakeFormValues | null>(null);
  const autosaveQueueRef = useRef<PlanningIntakeFormValues | null>(null);
  const autosaveFlightRef = useRef<Promise<PlanningIntakeDraftView> | null>(null);
  const autosaveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const autosaveKeyRef = useRef<string | null>(null);
  const autosaveSequenceRef = useRef(0);
  const createDraftOperationRef = useRef<string | null>(null);
  const selectionBatchOperationRef = useRef<SelectionBatchOperation | null>(null);
  const selectionWorkTotalRef = useRef(0);
  const selectionReviewTriggerRef = useRef<HTMLElement | null>(null);
  const catalogTriggerRef = useRef<HTMLElement | null>(null);
  const catalogDetailRequestRef = useRef(0);
  useEffect(() => { draftRef.current = draft; }, [draft]);
  useEffect(() => { valuesRef.current = values; }, [values]);
  useEffect(() => () => { if (autosaveTimerRef.current) clearTimeout(autosaveTimerRef.current); }, []);

  useEffect(() => {
    if (step > 1 && !requestedDraftId) return undefined;
    let cancelled = false;
    if (!backend.planningIntake || !backend.canonicalCatalog) { setServerError("Planning intake commands are unavailable in this build profile."); return () => { cancelled = true; }; }
    const load = async () => {
      const options: CanonicalAuditScopeOption[] = [];
      const seenCursors = new Set<string>();
      let cursor: string | undefined;
      do {
        const page = await backend.canonicalCatalog!.listScopeOptions({ limit: 25, cursor });
        options.push(...page.items);
        const nextCursor = page.nextCursor ?? undefined;
        if (!nextCursor) break;
        if (seenCursors.has(nextCursor)) throw new Error("Authorized scope pagination repeated a cursor.");
        seenCursors.add(nextCursor);
        cursor = nextCursor;
      } while (options.length < 1000);
      if (cursor && options.length >= 1000) throw new Error("Authorized scope options exceeded the bounded page limit.");
      if (cancelled) return;
      setScopeOptions(options);
      setAuditUsageClass(options[0]?.usageClass ?? "GOVERNED_OPERATIONAL");
      if (!requestedDraftId) {
        const first = options[0];
        if (first) {
          setPendingOrganizationId(first.organizationId);
          setPendingProviderScopeId(first.providerScopeId);
          setPendingRegulatedTargetId(first.regulatedTargetId);
          setPendingApplicationType(inspectionTypeFor(first.inspectionTypes));
        }
        return;
      }
      const loaded = await backend.planningIntake!.getDraft({ draftId: requestedDraftId });
      if (cancelled) return;
      const matchingOption = options.find((option) => option.catalogVersion === loaded.catalogVersion && option.organizationId === loaded.organizationId && option.providerScopeId === loaded.providerScopeId && option.regulatedTargetId === loaded.regulatedTargetId);
      if (!matchingOption) throw new Error("The saved Planning draft no longer has an exact authorized catalog/scope/target option.");
      setAuditUsageClass(matchingOption.usageClass);
      setPendingOrganizationId(loaded.organizationId);
      setPendingProviderScopeId(loaded.providerScopeId ?? matchingOption.providerScopeId);
      setPendingRegulatedTargetId(loaded.regulatedTargetId ?? matchingOption.regulatedTargetId);
      setPendingApplicationType(loaded.applicationType as CanonicalApplicationType);
      setDraft(loaded);
      const hydrated = formValuesFor(loaded);
      setValues(hydrated);
      valuesRef.current = hydrated;
      setPendingSelectionIds([...(loaded.selectedQuestionVersionIds ?? [])]);
      setSelectionDirty(false);
      if (loaded.selectionDigest && loaded.formDistribution && loaded.domainDistribution && loaded.estimatedResourceRequirement !== undefined) {
        setServerSelectionSummary({ selectionDigest: loaded.selectionDigest, selectedQuestionVersionIds: [...(loaded.selectedQuestionVersionIds ?? [])], selectedCount: loaded.selectedQuestionVersionIds?.length ?? 0, catalogVersion: loaded.catalogVersion ?? "", usageClass: matchingOption.usageClass, formDistribution: loaded.formDistribution, domainDistribution: loaded.domainDistribution, estimatedResourceRequirement: loaded.estimatedResourceRequirement });
      }
      setAutosaveState("saved");
    };
    void load().catch((cause) => {
      if (cancelled) return;
      if (requestedDraftId) { setStatus("This Planning draft is no longer available. Start again from Basics."); navigate(pathForStep(1), { replace: true }); }
      else setServerError(errorMessage(cause));
    });
    return () => { cancelled = true; };
  }, [backend, navigate, requestedDraftId, step]);

  useEffect(() => {
    if (step !== 4 || !values?.catalogVersion || !backend.canonicalCatalog) return undefined;
    const controller = new AbortController();
    setCatalogBusy(true);
    void backend.canonicalCatalog.listCatalog({ catalogVersion: values.catalogVersion, usageClass: auditUsageClass, search: catalogSearch || undefined, formCode: catalogFormCode.length ? catalogFormCode : undefined, domain: catalogDomain.length ? catalogDomain : undefined, topic: catalogTopic.length ? catalogTopic : undefined, riskBand: catalogRiskBand.length ? catalogRiskBand : undefined, sourceGapState: catalogSourceGapState || undefined, checklistFocus: catalogChecklistFocus.length ? catalogChecklistFocus : undefined, recommendationState: catalogRecommendationState && catalogRecommendationState !== defaultCatalogRecommendationState ? catalogRecommendationState : undefined, includedByDefault: catalogRecommendationState === defaultCatalogRecommendationState ? true : undefined, selected: catalogSelectedFilter, scopeId: values.scopeDraftId || undefined, applicationType: values.applicationType as CanonicalApplicationType, cursor: catalogCursor, limit: 25 }, { signal: controller.signal }).then((page) => { if (!controller.signal.aborted) setCatalogPage(page); }).catch((cause) => { if (!controller.signal.aborted) setServerError(errorMessage(cause)); }).finally(() => { if (!controller.signal.aborted) setCatalogBusy(false); });
    return () => controller.abort();
  }, [auditUsageClass, backend, catalogChecklistFocus, catalogCursor, catalogDomain, catalogFormCode, catalogRecommendationState, catalogRiskBand, catalogSearch, catalogSelectedFilter, catalogSourceGapState, catalogTopic, step, values?.applicationType, values?.catalogVersion, values?.scopeDraftId]);

  useEffect(() => {
    if (step !== 4 || !values?.catalogVersion || !values.scopeDraftId || !backend.canonicalCatalog) {
      setHistoryDeferredQuestions([]);
      setHistoryDeferredError(null);
      return undefined;
    }
    const controller = new AbortController();
    setHistoryDeferredBusy(true);
    setHistoryDeferredError(null);
    void (async () => {
      const questions: CanonicalQuestionCatalogEntry[] = [];
      const seenIds = new Set<string>();
      const seenCursors = new Set<string>();
      let cursor: string | undefined;
      do {
        const page = await backend.canonicalCatalog!.listCatalog({
          catalogVersion: values.catalogVersion || "",
          usageClass: auditUsageClass,
          recommendationState: "RECENTLY_VERIFIED",
          includedByDefault: false,
          scopeId: values.scopeDraftId,
          applicationType: values.applicationType as CanonicalApplicationType,
          cursor,
          limit: 2000,
          projection: "selection",
        }, { signal: controller.signal });
        for (const question of page.items) {
          if (seenIds.has(question.questionVersionId)) throw new Error("History-deferred catalog repeated a question version.");
          seenIds.add(question.questionVersionId);
          if (question.canSelect && question.recommendation.recommendationState === "RECENTLY_VERIFIED" && question.recommendation.classification === "DEFER_ELIGIBLE" && question.recommendation.canDefer && !question.recommendation.includedByDefault) questions.push(question);
        }
        const nextCursor = page.nextCursor ?? undefined;
        if (nextCursor && seenCursors.has(nextCursor)) throw new Error("History-deferred catalog repeated a cursor.");
        if (nextCursor) seenCursors.add(nextCursor);
        cursor = nextCursor;
      } while (cursor);
      if (!controller.signal.aborted) setHistoryDeferredQuestions(questions);
    })().catch((cause) => {
      if (!controller.signal.aborted) {
        setHistoryDeferredQuestions([]);
        setHistoryDeferredError(errorMessage(cause));
      }
    }).finally(() => {
      if (!controller.signal.aborted) setHistoryDeferredBusy(false);
    });
    return () => controller.abort();
  }, [auditUsageClass, backend, step, values?.applicationType, values?.catalogVersion, values?.scopeDraftId]);

  useEffect(() => {
    const background = document.querySelector<HTMLElement>(".planning-intake-page");
    if (!background) return undefined;
    if (selectionReviewOpen || catalogDetail) { background.setAttribute("inert", ""); background.setAttribute("aria-hidden", "true"); }
    else { background.removeAttribute("inert"); background.removeAttribute("aria-hidden"); }
    return () => { background.removeAttribute("inert"); background.removeAttribute("aria-hidden"); };
  }, [catalogDetail, selectionReviewOpen]);

  const selectedScopeOption = useMemo(() => values ? scopeOptions.find((option) => option.organizationId === values.organizationId && option.providerScopeId === values.providerScopeId && option.regulatedTargetId === values.regulatedTargetId) ?? null : null, [scopeOptions, values]);
  const pendingScopeOption = useMemo(() => scopeOptions.find((option) => option.organizationId === pendingOrganizationId && option.providerScopeId === pendingProviderScopeId && option.regulatedTargetId === pendingRegulatedTargetId) ?? null, [pendingOrganizationId, pendingProviderScopeId, pendingRegulatedTargetId, scopeOptions]);
  const supplierOptions = useMemo(() => [...new Map(scopeOptions.map((option) => [option.organizationId, option])).values()], [scopeOptions]);
  const pendingProviderOptions = useMemo(() => scopeOptions.filter((option) => option.organizationId === pendingOrganizationId).filter((option, index, all) => all.findIndex((candidate) => candidate.providerScopeId === option.providerScopeId) === index), [pendingOrganizationId, scopeOptions]);
  const pendingTargetOptions = useMemo(() => scopeOptions.filter((option) => option.organizationId === pendingOrganizationId && option.providerScopeId === pendingProviderScopeId), [pendingOrganizationId, pendingProviderScopeId, scopeOptions]);
  const selectedProviderOptions = useMemo(() => scopeOptions.filter((option) => option.organizationId === values?.organizationId).filter((option, index, all) => all.findIndex((candidate) => candidate.providerScopeId === option.providerScopeId) === index), [scopeOptions, values?.organizationId]);
  const selectedTargetOptions = useMemo(() => scopeOptions.filter((option) => option.organizationId === values?.organizationId && option.providerScopeId === values?.providerScopeId), [scopeOptions, values?.organizationId, values?.providerScopeId]);
  const activeFilterCount = [catalogFormCode.length, catalogDomain.length, catalogTopic.length, catalogRiskBand.length, catalogSourceGapState ? 1 : 0, catalogChecklistFocus.length, catalogRecommendationState !== defaultCatalogRecommendationState ? 1 : 0, catalogSelectedFilter !== "all" ? 1 : 0].reduce((sum, count) => sum + count, 0);
  const selectionDelta = useMemo(() => { const current = new Set(values?.selectedQuestionVersionIds ?? []); const desired = new Set(pendingSelectionIds); return { additions: pendingSelectionIds.filter((id) => !current.has(id)).length, removals: (values?.selectedQuestionVersionIds ?? []).filter((id) => !desired.has(id)).length, selectedCount: pendingSelectionIds.length }; }, [pendingSelectionIds, values?.selectedQuestionVersionIds]);
  const selectionSummary = useMemo(() => { const serverSummary = serverSelectionSummary && serverSelectionSummary.selectedQuestionVersionIds.length === pendingSelectionIds.length && serverSelectionSummary.selectedQuestionVersionIds.every((id) => pendingSelectionIds.includes(id)) ? serverSelectionSummary : null; return { complete: Boolean(serverSummary), estimatedResourceRequirement: serverSummary?.estimatedResourceRequirement, formDistribution: serverSummary?.formDistribution ?? {}, domainDistribution: serverSummary?.domainDistribution ?? {} }; }, [pendingSelectionIds, serverSelectionSummary]);
  const recommendationSummary = catalogPage?.recommendationSummary ?? null;
  const historyScenarioLabel = recommendationSummary ? recommendationSummary.comparableAuditCount === 0 ? "No comparable prior Audits" : recommendationSummary.comparableAuditCount === 1 ? "One comparable prior Audit" : `${recommendationSummary.comparableAuditCount.toLocaleString("en-US")} comparable prior Audits` : "Recommendation summary unavailable";
  const historyDeferredReady = Boolean(recommendationSummary && !historyDeferredBusy && !historyDeferredError && historyDeferredQuestions.length === recommendationSummary.historyDeferredCount);
  const fullCatalogSelected = catalogRecommendationState === "";
  const catalogHeading = fullCatalogSelected ? "Full approved catalog" : "Suggested questions";
  const catalogDescription = fullCatalogSelected
    ? "Showing the complete approved catalog. Selection remains an explicit Department Manager decision."
    : "The catalog starts with the server's current recommendation. Selection remains an explicit Department Manager decision.";

  function resetCatalogPage() { setCatalogCursor(undefined); setCatalogPreviousCursors([]); setCatalogPageNumber(1); setCatalogPage(null); }
  function clearFieldError(field: FieldKey) { setFieldErrors((current) => { if (!current[field]) return current; const next = { ...current }; delete next[field]; return next; }); }
  function focusField(field: FieldKey) { document.getElementById(`planning-intake-${field}`)?.focus(); }
  function setDraftState(next: PlanningIntakeDraftView) { draftRef.current = next; setDraft(next); }
  function setFormValues(next: PlanningIntakeFormValues) { valuesRef.current = next; setValues(next); }

  async function saveNextQueued(): Promise<PlanningIntakeDraftView | null> {
    const currentDraft = draftRef.current;
    const queued = autosaveQueueRef.current;
    if (!currentDraft || !queued || !backend.planningIntake) return currentDraft;
    autosaveQueueRef.current = null;
    let valuesToSave: PlanningIntakeDraftValues;
    try { valuesToSave = commandValuesFor(queued); } catch (cause) { setAutosaveState("dirty"); setAutosaveError(errorMessage(cause)); return currentDraft; }
    const idempotencyKey = autosaveKeyRef.current ?? `SAVE-${currentDraft.id}-R${currentDraft.revision}-${++autosaveSequenceRef.current}`;
    autosaveKeyRef.current = idempotencyKey;
    setAutosaveState("saving");
    setAutosaveError(null);
    const request = backend.planningIntake.saveDraft({ draftId: currentDraft.id, expectedRevision: currentDraft.revision, idempotencyKey, values: valuesToSave });
    autosaveFlightRef.current = request;
    try {
      const saved = await request;
      autosaveFlightRef.current = null;
      autosaveKeyRef.current = null;
      setDraftState(saved);
      if (valuesRef.current === queued && !autosaveQueueRef.current) setFormValues(formValuesFor(saved));
      setAutosaveState(autosaveQueueRef.current ? "dirty" : "saved");
      return saved;
    } catch (cause) {
      autosaveFlightRef.current = null;
      if (!autosaveQueueRef.current) autosaveQueueRef.current = queued;
      setAutosaveState("error");
      setAutosaveError(errorMessage(cause));
      throw cause;
    }
  }

  async function flushAutosave(nextValues = valuesRef.current): Promise<PlanningIntakeDraftView | null> {
    if (!draftRef.current || !nextValues) return draftRef.current;
    autosaveQueueRef.current = nextValues;
    if (autosaveTimerRef.current) clearTimeout(autosaveTimerRef.current);
    let latest = draftRef.current;
    while (autosaveFlightRef.current || autosaveQueueRef.current) { if (autosaveFlightRef.current) latest = await autosaveFlightRef.current; else latest = (await saveNextQueued()) ?? latest; }
    return latest;
  }

  function queueAutosave(nextValues: PlanningIntakeFormValues) {
    if (!draftRef.current || !backend.planningIntake) return;
    autosaveQueueRef.current = nextValues;
    setAutosaveState("dirty");
    setAutosaveError(null);
    if (autosaveTimerRef.current) clearTimeout(autosaveTimerRef.current);
    autosaveTimerRef.current = setTimeout(() => { autosaveTimerRef.current = null; void flushAutosave().catch(() => undefined); }, 650);
  }

  function setFormValuesWithAutosave(updater: (current: PlanningIntakeFormValues) => PlanningIntakeFormValues) { const current = valuesRef.current; if (!current) return; const next = updater(current); setFormValues(next); queueAutosave(next); }
  function update<K extends keyof PlanningIntakeFormValues>(key: K, value: PlanningIntakeFormValues[K]) { setFormValuesWithAutosave((current) => ({ ...current, [key]: value })); clearFieldError(key as FieldKey); setStatus(null); }
  function updateCategory(category: PlanningIntakeInspectionCategory) { setFormValuesWithAutosave((current) => ({ ...current, inspectionCategory: category, noticePolicy: noticePolicyFor(category) })); clearFieldError("inspectionCategory"); setStatus(null); }
  function updatePendingOrganization(organizationId: string) { const first = scopeOptions.find((option) => option.organizationId === organizationId); setPendingOrganizationId(organizationId); setPendingProviderScopeId(first?.providerScopeId ?? ""); setPendingRegulatedTargetId(first?.regulatedTargetId ?? ""); setPendingApplicationType(first ? inspectionTypeFor(first.inspectionTypes) : ""); clearFieldError("organizationId"); }
  function updatePendingProvider(providerScopeId: string) { const first = scopeOptions.find((option) => option.organizationId === pendingOrganizationId && option.providerScopeId === providerScopeId); setPendingProviderScopeId(providerScopeId); setPendingRegulatedTargetId(first?.regulatedTargetId ?? ""); setPendingApplicationType(first ? inspectionTypeFor(first.inspectionTypes) : ""); clearFieldError("providerScopeId"); }
  function updatePendingTarget(regulatedTargetId: string) { const option = scopeOptions.find((candidate) => candidate.organizationId === pendingOrganizationId && candidate.providerScopeId === pendingProviderScopeId && candidate.regulatedTargetId === regulatedTargetId); setPendingRegulatedTargetId(regulatedTargetId); setPendingApplicationType(option ? inspectionTypeFor(option.inspectionTypes) : ""); clearFieldError("regulatedTargetId"); }
  function resetCatalogState() { setCatalogSearch(""); setCatalogFormCode([]); setCatalogDomain([]); setCatalogTopic([]); setCatalogRiskBand([]); setCatalogSourceGapState(""); setCatalogChecklistFocus([]); setCatalogRecommendationState(defaultCatalogRecommendationState); setCatalogSelectedFilter("all"); resetCatalogPage(); setCatalogDetail(null); }

  async function createDraftForScope(option: CanonicalAuditScopeOption, applicationType: CanonicalApplicationType, replaceExisting = false) {
    if (!backend.planningIntake) return;
    if (replaceExisting && valuesRef.current && (valuesRef.current.purpose || valuesRef.current.plannedDate || valuesRef.current.location || pendingSelectionIds.length) && !globalThis.confirm("Changing the inspection scope replaces the current purpose, schedule, and checklist context. Continue?")) return;
    setBusy(true); setServerError(null);
    try {
      const base = replaceExisting && valuesRef.current ? { ...defaultDraftValues(option, applicationType), inspectionCategory: valuesRef.current.inspectionCategory, noticePolicy: valuesRef.current.noticePolicy, triggerType: valuesRef.current.triggerType, riskCategory: valuesRef.current.riskCategory, requestedBudget: Number(valuesRef.current.requestedBudget) || 0, currency: valuesRef.current.currency } : defaultDraftValues(option, applicationType);
      const operationIdValue = createDraftOperationRef.current ?? operationId(replaceExisting ? "PLANNING-SCOPE-REPLACE" : "PLANNING-DRAFT-CREATE");
      createDraftOperationRef.current = operationIdValue;
      const created = await backend.planningIntake.createDraft({ operationId: operationIdValue, idempotencyKey: operationIdValue, expectedRevision: null, values: base });
      createDraftOperationRef.current = null;
      setDraftState(created);
      setFormValues(formValuesFor(created));
      setPendingSelectionIds([]); setSelectionDirty(false); setServerSelectionSummary(null); setAutosaveState("saved"); setAutosaveError(null); resetCatalogState(); setStatus("Saved");
      navigate(pathForStep(replaceExisting ? step : 2, created.id), { replace: true });
    } catch (cause) { setServerError(errorMessage(cause)); } finally { setBusy(false); }
  }

  async function handleExistingScopeChange(option: CanonicalAuditScopeOption) { if (!valuesRef.current || (option.providerScopeId === valuesRef.current.providerScopeId && option.regulatedTargetId === valuesRef.current.regulatedTargetId)) return; await createDraftForScope(option, valuesRef.current.applicationType as CanonicalApplicationType, true); }
  async function changeApplicationType(applicationType: string) { if (!valuesRef.current || applicationType === valuesRef.current.applicationType) return; const hasDownstream = Boolean(valuesRef.current.purpose || valuesRef.current.plannedDate || valuesRef.current.location || pendingSelectionIds.length); if (hasDownstream && !globalThis.confirm("Changing the inspection type replaces the current checklist context. Continue?")) return; if (hasDownstream && selectedScopeOption) { await createDraftForScope(selectedScopeOption, applicationType as CanonicalApplicationType, true); return; } update("applicationType", applicationType); resetCatalogState(); }

  function validationCandidate(currentStep: number): PlanningIntakeFormValues | null { if (!valuesRef.current) return null; if (currentStep !== 4) return valuesRef.current; return { ...valuesRef.current, selectedQuestionVersionIds: [...pendingSelectionIds], selectionDigest: selectionDirty ? valuesRef.current.selectionDigest || "pending" : valuesRef.current.selectionDigest }; }
  function errorsForStep(currentStep: number, candidate: unknown): FieldErrors { if (!candidate) return {}; const result = (stepSchemas[currentStep as keyof typeof stepSchemas] as z.ZodType).safeParse(candidate); if (result.success) return {}; const next: FieldErrors = {}; for (const issue of result.error.issues) { const field = issue.path[0] as FieldKey | undefined; if (field && !next[field]) next[field] = issue.message; } return next; }
  function validateStep(currentStep: number): boolean { const next = errorsForStep(currentStep, validationCandidate(currentStep)); setFieldErrors(next); const first = Object.keys(next)[0] as FieldKey | undefined; if (first) window.setTimeout(() => focusField(first), 0); return !first; }
  function validateAllSteps(): boolean { const next: FieldErrors = {}; for (const currentStep of [1, 2, 3, 4] as const) Object.assign(next, errorsForStep(currentStep, validationCandidate(currentStep))); setFieldErrors(next); const first = Object.keys(next)[0] as FieldKey | undefined; if (first) window.setTimeout(() => focusField(first), 0); return !first; }
  function validatePendingBasics(): boolean { const next = errorsForStep(1, { organizationId: pendingOrganizationId, applicationType: pendingApplicationType, domain: "Cabin Safety" }); setFieldErrors(next); const first = Object.keys(next)[0] as FieldKey | undefined; if (first) window.setTimeout(() => focusField(first), 0); return !first; }
  function validateField(field: FieldKey) { const candidate: unknown = step === 1 && !valuesRef.current ? { organizationId: pendingOrganizationId, applicationType: pendingApplicationType, domain: "Cabin Safety" } : validationCandidate(step === 5 ? 4 : step); const errors = errorsForStep(step === 5 ? 4 : step, candidate); setFieldErrors((current) => { const next = { ...current }; if (errors[field]) next[field] = errors[field]; else delete next[field]; return next; }); }
  async function retryAutosave() { try { await flushAutosave(); } catch { /* the brief keeps the retry state visible */ } }

  function openSelectionReview(element?: HTMLElement) { selectionReviewTriggerRef.current = element ?? document.activeElement as HTMLElement | null; const current = valuesRef.current?.selectedQuestionVersionIds ?? []; const total = selectionChangeCount(current, pendingSelectionIds); selectionWorkTotalRef.current = total; setSelectionProgress({ completed: 0, total, error: null }); setSelectionReviewOpen(true); }
  function closeSelectionReview() { setSelectionReviewOpen(false); setSelectionProgress({ completed: 0, total: 0, error: null }); }
  async function confirmSelectionReview() {
    if (!valuesRef.current || !draftRef.current || !backend.canonicalCatalog || !valuesRef.current.scopeDraftId || !selectionDirty) { closeSelectionReview(); return; }
    const target = [...pendingSelectionIds];
    let committed = [...(valuesRef.current.selectedQuestionVersionIds ?? [])];
    const total = selectionWorkTotalRef.current || selectionChangeCount(committed, target);
    let completed = total - selectionChangeCount(committed, target);
    setBusy(true); setSelectionProgress({ completed, total, error: null });
    try {
      while (true) {
        const batch = nextSelectionBatch(committed, target);
        if (!batch) break;
        const expectedSelectionDigest = valuesRef.current.selectionDigest || await selectionDigestFor(committed);
        const signature = `${batch.operationKind}:${expectedSelectionDigest}:${batch.questionVersionIds.join("|")}`;
        if (selectionBatchOperationRef.current?.signature !== signature) selectionBatchOperationRef.current = { signature, previewOperationId: operationId("SELECTION-PREVIEW"), commitOperationId: operationId("SELECTION-COMMIT") };
        const operations = selectionBatchOperationRef.current;
        const preview = await backend.canonicalCatalog.previewSelection({ scopeId: valuesRef.current.scopeDraftId, operationId: operations.previewOperationId, idempotencyKey: operations.previewOperationId, expectedSelectionDigest, questionVersionIds: batch.questionVersionIds, operationKind: batch.operationKind, usageClass: auditUsageClass, filter: {} });
        if (!preview.valid) throw new Error(preview.reason || "The server rejected this selection.");
        const receipt = await backend.canonicalCatalog.commitSelection({ scopeId: valuesRef.current.scopeDraftId, operationId: operations.commitOperationId, previewOperationId: operations.previewOperationId, idempotencyKey: operations.commitOperationId, expectedSelectionDigest, questionVersionIds: batch.questionVersionIds, operationKind: batch.operationKind, usageClass: auditUsageClass, filter: {} });
        selectionBatchOperationRef.current = null;
        committed = receipt.selection.selectedQuestionVersionIds;
        const nextValues: PlanningIntakeFormValues = { ...valuesRef.current, selectedQuestionVersionIds: [...committed], selectionDigest: receipt.selection.selectionDigest, estimatedResourceRequirement: receipt.selection.estimatedResourceRequirement, formDistribution: receipt.selection.formDistribution, domainDistribution: receipt.selection.domainDistribution };
        setFormValues(nextValues); setServerSelectionSummary(receipt.selection); completed = total - selectionChangeCount(committed, target); setSelectionProgress({ completed, total, error: null });
      }
      setPendingSelectionIds(target); setSelectionDirty(false); setSelectionProgress({ completed: total, total, error: null }); setStatus("Selection confirmed and saved to the server-owned scope."); if (valuesRef.current) queueAutosave(valuesRef.current); setSelectionReviewOpen(false);
    } catch (cause) { setSelectionProgress({ completed, total, error: errorMessage(cause) }); setStatus("Selection confirmation stopped. Completed receipts remain intact; retry to resume."); }
    finally { setBusy(false); }
  }
  function retrySelectionConfirmation() { setSelectionProgress((current) => ({ ...current, error: null })); void confirmSelectionReview(); }
  function toggleQuestion(questionVersionId: string) { const next = pendingSelectionIds.includes(questionVersionId) ? pendingSelectionIds.filter((id) => id !== questionVersionId) : [...pendingSelectionIds, questionVersionId]; setPendingSelectionIds([...new Set(next)]); setSelectionDirty(true); setStatus("Selection changes are ready for review."); clearFieldError("selectedQuestionVersionIds"); }

  function restoreHistoryDeferredQuestions() {
    const eligibleIds = historyDeferredQuestions
      .filter((question) => question.canSelect && question.recommendation.recommendationState === "RECENTLY_VERIFIED" && question.recommendation.classification === "DEFER_ELIGIBLE" && question.recommendation.canDefer && !question.recommendation.includedByDefault)
      .map((question) => question.questionVersionId);
    if (!eligibleIds.length) {
      setStatus("There are no history-deferred optional questions to restore.");
      return;
    }
    setPendingSelectionIds((current) => [...new Set([...current, ...eligibleIds])]);
    setSelectionDirty(true);
    setStatus(`${eligibleIds.length.toLocaleString("en-US")} history-deferred questions are included and ready for selection review.`);
    clearFieldError("selectedQuestionVersionIds");
  }

  async function addAllMatchingQuestions(recommendationOverride?: string) {
    if (busy || !valuesRef.current || !backend.canonicalCatalog) return;
    setBusy(true); setServerError(null);
    try {
      const ids: string[] = []; const seenIds = new Set<string>(); const seenCursors = new Set<string>(); let cursor: string | undefined;
      do {
        const page = await backend.canonicalCatalog.listCatalog({ catalogVersion: valuesRef.current.catalogVersion || "", usageClass: auditUsageClass, search: catalogSearch || undefined, formCode: catalogFormCode.length ? catalogFormCode : undefined, domain: catalogDomain.length ? catalogDomain : undefined, topic: catalogTopic.length ? catalogTopic : undefined, riskBand: catalogRiskBand.length ? catalogRiskBand : undefined, sourceGapState: catalogSourceGapState || undefined, checklistFocus: catalogChecklistFocus.length ? catalogChecklistFocus : undefined, recommendationState: recommendationOverride && recommendationOverride !== defaultCatalogRecommendationState ? recommendationOverride : undefined, includedByDefault: recommendationOverride === defaultCatalogRecommendationState ? true : undefined, selected: catalogSelectedFilter, scopeId: valuesRef.current.scopeDraftId || undefined, applicationType: valuesRef.current.applicationType as CanonicalApplicationType, cursor, limit: 100, projection: "selection" });
        for (const entry of page.items) if (entry.canSelect && !seenIds.has(entry.questionVersionId)) { seenIds.add(entry.questionVersionId); ids.push(entry.questionVersionId); }
        const nextCursor = page.nextCursor ?? undefined; if (nextCursor && seenCursors.has(nextCursor)) throw new Error("Catalog pagination repeated a cursor while collecting the exact question set."); if (nextCursor) seenCursors.add(nextCursor); cursor = nextCursor;
      } while (cursor);
      if (!ids.length) throw new Error("No selectable questions match the current server-authorized filters.");
      setPendingSelectionIds((current) => [...new Set([...current, ...ids])]); setSelectionDirty(true); setStatus(`${ids.length.toLocaleString("en-US")} ${recommendationOverride ? "suggested" : "eligible"} questions are ready for selection review.`);
    } catch (cause) { setServerError(errorMessage(cause)); } finally { setBusy(false); }
  }

  async function openCatalogDetail(question: CanonicalQuestionCatalogEntry, trigger?: HTMLElement) { const requestId = ++catalogDetailRequestRef.current; catalogTriggerRef.current = trigger ?? document.activeElement as HTMLElement | null; setCatalogDetail(question); if (!valuesRef.current || !backend.canonicalCatalog) return; try { const detail = await backend.canonicalCatalog.getQuestion({ catalogVersion: valuesRef.current.catalogVersion || "", usageClass: auditUsageClass, questionVersionId: question.questionVersionId, scopeId: valuesRef.current.scopeDraftId || undefined, applicationType: valuesRef.current.applicationType as CanonicalApplicationType }); if (catalogDetailRequestRef.current === requestId) setCatalogDetail(detail); } catch (cause) { if (catalogDetailRequestRef.current === requestId) setServerError(errorMessage(cause)); } }
  function closeCatalogDetail() { catalogDetailRequestRef.current += 1; setCatalogDetail(null); }

  async function continueFromStep() {
    if (step === 1 && !valuesRef.current) { if (!validatePendingBasics() || !pendingScopeOption || !pendingApplicationType) return; await createDraftForScope(pendingScopeOption, pendingApplicationType); return; }
    if (!valuesRef.current) return;
    if (!validateStep(step === 5 ? 4 : step)) return;
    if (step === 4 && selectionDirty) { openSelectionReview(); return; }
    setBusy(true); setServerError(null);
    try { const saved = await flushAutosave(valuesRef.current); navigate(pathForStep(step + 1, saved?.id ?? draftRef.current?.id)); } catch (cause) { setServerError(errorMessage(cause)); } finally { setBusy(false); }
  }
  async function moveBack() { if (!valuesRef.current || step <= 1) return; setBusy(true); try { const saved = await flushAutosave(valuesRef.current); navigate(pathForStep(step - 1, saved?.id ?? draftRef.current?.id)); } catch (cause) { setServerError(errorMessage(cause)); } finally { setBusy(false); } }
  function cancel() { const needsConfirmation = Boolean(draftRef.current && (autosaveState === "dirty" || autosaveState === "saving" || autosaveState === "error" || selectionDirty)); if (needsConfirmation && !globalThis.confirm("You have changes that are not fully saved. Leave this intake?")) return; navigate("/department-manager/audit-plan"); }
  async function submit() { if (!draftRef.current || !valuesRef.current || !backend.planningIntake) return; if (!validateAllSteps() || selectionDirty) { if (selectionDirty) openSelectionReview(); return; } setBusy(true); setServerError(null); try { const saved = await flushAutosave(valuesRef.current); if (!saved) return; const output = await backend.planningIntake.submit({ draftId: saved.id, expectedRevision: saved.revision, idempotencyKey: `SUBMIT-${saved.id}-R${saved.revision}` }); navigate(`/department-manager/audit-plan?planningItemId=${encodeURIComponent(output.planningItem.id)}`); } catch (cause) { setServerError(errorMessage(cause)); } finally { setBusy(false); } }
  function editStep(targetStep: number) { navigate(pathForStep(targetStep, draftRef.current?.id)); }

  const definition = stepDefinitions[step - 1] ?? stepDefinitions[0];
  const canCreateDraft = Boolean(pendingScopeOption && pendingApplicationType && scopeOptions.length);
  const currentSelectedCount = pendingSelectionIds.length;
  const useReviewPrimary = step === 4 && selectionDirty;
  const actionLabel = busy ? (step === 1 && !values ? "Creating draft…" : step === 5 ? "Submitting…" : "Saving…") : "Continue";

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel={`New Audit Wizard ${step}`}>
      <div className="planning-intake-page" data-draft-id={draft?.id} data-testid={draft ? "new-audit-wizard-page" : undefined}>
        <header className="planning-intake-header workbench-page-header"><div><h1>New Inspection</h1><p>Create a governed Planning item. An executable Audit is created only after the accepted release and confirmation stage.</p></div></header>
        <PlanningIntakeProgress step={step} />
        {serverError ? <CommandError message={serverError} /> : null}
        {status ? <p className="planning-intake-status" role="status">{status}</p> : null}
        <ValidationSummary errors={fieldErrors} onFocus={focusField} />
        <div className="planning-intake-layout">
          <section aria-label="Planning intake form" className="planning-intake-form">
            <header className="planning-intake-form__header"><span>Step {step} of 5</span><h2>{definition.label}</h2><p>{definition.description}</p></header>
            {routeRedirecting ? <p className="planning-intake-loading" role="status">Returning to Basics…</p> : null}
            {!routeRedirecting && step > 1 && !values ? <p className="planning-intake-loading" role="status">Loading saved Planning draft…</p> : null}
            {!routeRedirecting && step === 1 && !values ? <div className="planning-intake-fields planning-intake-scope-fields">
              <label htmlFor="planning-intake-organizationId">Supplier / organization <RequiredMark /><select aria-label="Supplier / organization" id="planning-intake-organizationId" aria-invalid={Boolean(fieldErrors.organizationId)} aria-describedby={fieldErrors.organizationId ? "planning-intake-organizationId-error" : undefined} disabled={busy || !supplierOptions.length} value={pendingOrganizationId} onBlur={() => validateField("organizationId")} onChange={(event) => updatePendingOrganization(event.target.value)}>{supplierOptions.map((option) => <option key={option.organizationId} value={option.organizationId}>{option.organizationName}</option>)}</select><small>Select the supplier or organization that will be inspected.</small><FieldError id="planning-intake-organizationId-error" message={fieldErrors.organizationId} /></label>
              <label htmlFor="planning-intake-providerScopeId">Provider scope <RequiredMark /><select aria-label="Provider scope" id="planning-intake-providerScopeId" aria-invalid={Boolean(fieldErrors.providerScopeId)} aria-describedby={fieldErrors.providerScopeId ? "planning-intake-providerScopeId-error" : undefined} disabled={busy || !pendingProviderOptions.length} value={pendingProviderScopeId} onBlur={() => validateField("providerScopeId")} onChange={(event) => updatePendingProvider(event.target.value)}>{pendingProviderOptions.map((option) => <option key={option.providerScopeId} value={option.providerScopeId}>{option.providerTypeLabel}</option>)}</select><small>Choose the authorized aviation provider scope.</small><FieldError id="planning-intake-providerScopeId-error" message={fieldErrors.providerScopeId} /></label>
              <label htmlFor="planning-intake-regulatedTargetId">Regulated target <RequiredMark /><select aria-label="Regulated target" id="planning-intake-regulatedTargetId" aria-invalid={Boolean(fieldErrors.regulatedTargetId)} aria-describedby={fieldErrors.regulatedTargetId ? "planning-intake-regulatedTargetId-error" : undefined} disabled={busy || !pendingTargetOptions.length} value={pendingRegulatedTargetId} onBlur={() => validateField("regulatedTargetId")} onChange={(event) => updatePendingTarget(event.target.value)}>{pendingTargetOptions.map((option) => <option key={option.regulatedTargetId} value={option.regulatedTargetId}>{option.targetLabel}</option>)}</select><small>Select the regulated target applicable to this inspection.</small><FieldError id="planning-intake-regulatedTargetId-error" message={fieldErrors.regulatedTargetId} /></label>
              <label htmlFor="planning-intake-applicationType">Inspection type <RequiredMark /><select id="planning-intake-applicationType" aria-label="Inspection type" aria-invalid={Boolean(fieldErrors.applicationType)} aria-describedby={fieldErrors.applicationType ? "planning-intake-applicationType-error" : undefined} disabled={busy || !pendingScopeOption?.inspectionTypes.length} value={pendingApplicationType} onBlur={() => validateField("applicationType")} onChange={(event) => { setPendingApplicationType(event.target.value as CanonicalApplicationType); clearFieldError("applicationType"); }}>{(pendingScopeOption?.inspectionTypes ?? []).map((type) => <option key={type} value={type}>{catalogValueLabel(type)}</option>)}</select><small>Recommendations and prior-audit history follow this server-authorized type.</small><FieldError id="planning-intake-applicationType-error" message={fieldErrors.applicationType} /></label>
              <p className="planning-intake-boundary-note" role="note">Continuing creates a Planning draft; it does not create an Audit.</p>
            </div> : null}
            {!routeRedirecting && step === 1 && values ? <div className="planning-intake-fields planning-intake-scope-fields">
              <label htmlFor="planning-intake-existing-organization">Supplier / organization <RequiredMark /><select id="planning-intake-existing-organization" value={values.organizationId} onChange={(event) => { const option = scopeOptions.find((candidate) => candidate.organizationId === event.target.value); if (option) void handleExistingScopeChange(option); }}>{supplierOptions.map((option) => <option key={option.organizationId} value={option.organizationId}>{option.organizationName}</option>)}</select></label>
              <label htmlFor="planning-intake-existing-provider">Provider scope <RequiredMark /><select id="planning-intake-existing-provider" value={values.providerScopeId ?? ""} onChange={(event) => { const option = scopeOptions.find((candidate) => candidate.organizationId === values.organizationId && candidate.providerScopeId === event.target.value); if (option) void handleExistingScopeChange(option); }}>{selectedProviderOptions.map((option) => <option key={option.providerScopeId} value={option.providerScopeId}>{option.providerTypeLabel}</option>)}</select></label>
              <label htmlFor="planning-intake-existing-target">Regulated target <RequiredMark /><select id="planning-intake-existing-target" value={values.regulatedTargetId ?? ""} onChange={(event) => { const option = scopeOptions.find((candidate) => candidate.organizationId === values.organizationId && candidate.providerScopeId === values.providerScopeId && candidate.regulatedTargetId === event.target.value); if (option) void handleExistingScopeChange(option); }}>{selectedTargetOptions.map((option) => <option key={option.regulatedTargetId} value={option.regulatedTargetId}>{option.targetLabel}</option>)}</select></label>
              <label htmlFor="planning-intake-existing-application">Inspection type <RequiredMark /><select id="planning-intake-existing-application" value={values.applicationType} onChange={(event) => void changeApplicationType(event.target.value)}>{(selectedScopeOption?.inspectionTypes ?? []).map((type) => <option key={type} value={type}>{catalogValueLabel(type)}</option>)}</select><small>Changing the authorized type may replace later checklist context.</small></label>
              <p className="planning-intake-boundary-note" role="note">Continuing creates a Planning draft; it does not create an Audit.</p>
            </div> : null}
            {!routeRedirecting && values && step === 2 ? <div className="planning-intake-fields">
              <label htmlFor="planning-intake-inspectionCategory">Inspection approach <RequiredMark /><select aria-label="Inspection approach" id="planning-intake-inspectionCategory" value={values.inspectionCategory} onChange={(event) => updateCategory(event.target.value as PlanningIntakeInspectionCategory)}><option value="Routine / Announced">Routine / Announced</option><option value="Ad Hoc / Unannounced">Ad Hoc / Unannounced</option></select><small>{values.inspectionCategory === "Ad Hoc / Unannounced" ? "The supplier notice remains withheld through Planning." : "Advance notice applies after the accepted governance stage."}</small></label>
              <label className="is-wide" htmlFor="planning-intake-purpose">Purpose <RequiredMark /><textarea aria-label="Purpose" id="planning-intake-purpose" aria-invalid={Boolean(fieldErrors.purpose)} aria-describedby={fieldErrors.purpose ? "planning-intake-purpose-error" : undefined} value={values.purpose} onBlur={() => validateField("purpose")} onChange={(event) => update("purpose", event.target.value)} /><small>Describe why this inspection is being undertaken and what the Department Manager needs to establish.</small><FieldError id="planning-intake-purpose-error" message={fieldErrors.purpose} /></label>
              <label htmlFor="planning-intake-triggerType">Trigger type<input id="planning-intake-triggerType" readOnly value={values.triggerType} /><small>Configured from the Planning authority.</small></label>
              <label htmlFor="planning-intake-riskCategory">Risk category <RequiredMark /><select aria-label="Risk category" id="planning-intake-riskCategory" aria-invalid={Boolean(fieldErrors.riskCategory)} aria-describedby={fieldErrors.riskCategory ? "planning-intake-riskCategory-error" : undefined} value={values.riskCategory} onBlur={() => validateField("riskCategory")} onChange={(event) => update("riskCategory", event.target.value)}>{riskCategoryOptionsFor(values.riskCategory).map((option) => <option key={option} value={option}>{option}</option>)}</select><small>Select the configured risk category for this inspection.</small><FieldError id="planning-intake-riskCategory-error" message={fieldErrors.riskCategory} /></label>
              <p className="planning-intake-boundary-note" role="note"><b>{noticeLabel(values.inspectionCategory)}</b><span>{values.inspectionCategory === "Ad Hoc / Unannounced" ? "Organization notice remains withheld through this Planning stage." : "Notice is derived from the selected inspection approach."}</span></p>
            </div> : null}
            {!routeRedirecting && values && step === 3 ? <div className="planning-intake-fields">
              <label htmlFor="planning-intake-plannedDate">Planned date <RequiredMark /><PlanningDateField error={fieldErrors.plannedDate} onBlur={() => validateField("plannedDate")} onChange={(value) => update("plannedDate", value)} onNext={() => void continueFromStep()} value={values.plannedDate} /><small>Enter YYYY-MM-DD or open the calendar.</small><FieldError id="planning-intake-plannedDate-error" message={fieldErrors.plannedDate} /></label>
              <label htmlFor="planning-intake-mode">Mode<select id="planning-intake-mode" value={values.mode} onChange={(event) => update("mode", event.target.value as PlanningIntakeDraftValues["mode"])}><option value="On-site">On-site</option><option value="Remote">Remote</option></select><small>Choose how the inspection will be conducted.</small></label>
              <label className="is-wide" htmlFor="planning-intake-location">Location <RequiredMark /><input aria-label="Location" id="planning-intake-location" aria-invalid={Boolean(fieldErrors.location)} aria-describedby={fieldErrors.location ? "planning-intake-location-error" : undefined} value={values.location} onBlur={() => validateField("location")} onChange={(event) => update("location", event.target.value)} /><small>Specify the airport, facility, or other inspection location.</small><FieldError id="planning-intake-location-error" message={fieldErrors.location} /></label>
            </div> : null}
            {!routeRedirecting && values && step === 4 ? <div className="planning-intake-checklist-step">
              <section aria-label="Question catalog selection" className="planning-intake-catalog">
                <header className="planning-intake-catalog-header"><div><span className="planning-intake-dialog-kicker">{fullCatalogSelected ? "Approved catalog" : "Server recommendation"}</span><h3>{catalogHeading}</h3><p>{catalogDescription}</p></div><span className="planning-intake-catalog-count" aria-live="polite">{pendingSelectionIds.length.toLocaleString("en-US")} selected</span></header>
                {recommendationSummary ? <aside aria-label="Prior-Audit recommendation summary" className="planning-intake-recommendation-summary" role="status">
                  <header><div><span className="planning-intake-dialog-kicker">Prior-Audit history</span><h4>{historyScenarioLabel}</h4></div><span>{recommendationSummary.historyDeferredCount.toLocaleString("en-US")} withheld by history</span></header>
                  <p>{recommendationSummary.comparableAuditCount === 0 ? "No comparable history was found for this exact organization, provider scope, regulated target, location, and audit type. Suggested questions are still constrained by authorized applicability and the selected audit-type focus." : `The server compared this scope against ${recommendationSummary.comparableAuditCount.toLocaleString("en-US")} immutable FINAL/LOCKED Audit${recommendationSummary.comparableAuditCount === 1 ? "" : "s"} in the fixed ${recommendationSummary.historyWindowMonths}-month history window.`}</p>
                  <dl className="planning-intake-recommendation-summary__facts"><div><dt>Scope</dt><dd>{recommendationSummary.organizationLabel} · {recommendationSummary.providerScopeLabel} · {recommendationSummary.regulatedTargetLabel}</dd></div><div><dt>Location</dt><dd>{recommendationSummary.locationLabel || "Not specified"}</dd></div><div><dt>Audit type focus</dt><dd>{recommendationSummary.focusConfigured ? `${catalogValueLabel(recommendationSummary.focusType ?? recommendationSummary.auditTypeLabel)} · ${recommendationSummary.focusInspectionTypeCodes.map(catalogValueLabel).join(", ")}` : "Not configured"}</dd></div></dl>
                  {historyDeferredBusy ? <p className="planning-intake-loading" role="status">Loading every history-deferred question and its reason…</p> : null}
                  {historyDeferredError ? <p className="planning-intake-error" role="alert">History-deferred questions could not be loaded: {historyDeferredError}</p> : null}
                  {recommendationSummary.historyDeferredCount > 0 ? <>
                    <p className="planning-intake-recommendation-summary__warning"><b>Why these questions are not suggested now:</b> comparable Audits repeatedly satisfied these optional controls within their recurrence interval. They remain selectable in the full approved catalog.</p>
                    <ul aria-label="History-deferred questions" className="planning-intake-recommendation-summary__list">{historyDeferredQuestions.map((question) => { const cleanCount = question.recommendation.validatedCleanAuditCount; const comparableCount = question.recommendation.comparableAuditCount; return <li key={question.questionVersionId}><b>{question.prompt ?? `${question.formCode} item ${question.ordinal}`}</b><small>{question.formCode} · item {question.ordinal} · {cleanCount.toLocaleString("en-US")} validated-clean of {comparableCount.toLocaleString("en-US")} comparable Audits{question.recommendation.lastValidatedCleanAt ? ` · last clean ${question.recommendation.lastValidatedCleanAt.slice(0, 10)}` : ""}</small><span>{question.recommendation.rationale}</span></li>; })}</ul>
                    <button disabled={busy || !historyDeferredReady} type="button" onClick={restoreHistoryDeferredQuestions}>Include all history-deferred questions</button>
                    {!historyDeferredBusy && !historyDeferredError && !historyDeferredReady ? <p className="planning-intake-error" role="alert">The complete history-deferred list is not ready yet; the restore action stays disabled until the server count matches.</p> : null}
                  </> : <p className="planning-intake-recommendation-summary__clear">No optional questions were withheld by comparable history. Risk-protected and uncertain questions remain suggested.</p>}
                </aside> : null}
                <label className="planning-intake-catalog-search" htmlFor="planning-intake-catalog-search">Search questions<input id="planning-intake-catalog-search" aria-label="Search questions" value={catalogSearch} onChange={(event) => { setCatalogSearch(event.target.value); resetCatalogPage(); }} placeholder="Search question text, form, or item reference" /></label>
                <details className="planning-intake-advanced-filters"><summary>Advanced filters <span>{activeFilterCount} active</span></summary><div className="planning-intake-catalog-filters" aria-label="Advanced question filters">
                  <CatalogFacetPicker ariaLabel="Form filter" label="Form" options={catalogPage?.facets.forms ?? []} selected={catalogFormCode} onChange={(next) => { setCatalogFormCode(next); resetCatalogPage(); }} />
                  <CatalogFacetPicker ariaLabel="Domain filter" label="Domain" options={catalogPage?.facets.domains ?? []} selected={catalogDomain} onChange={(next) => { setCatalogDomain(next); resetCatalogPage(); }} />
                  <CatalogFacetPicker ariaLabel="Topic filter" label="Topic" options={catalogPage?.facets.topics ?? []} selected={catalogTopic} onChange={(next) => { setCatalogTopic(next); resetCatalogPage(); }} />
                  <CatalogFacetPicker ariaLabel="Risk filter" label="Risk" options={catalogPage?.facets.riskTiers ?? []} selected={catalogRiskBand} onChange={(next) => { setCatalogRiskBand(next); resetCatalogPage(); }} />
                  <CatalogFacetPicker ariaLabel="Checklist focus filter" label="Checklist focus" options={catalogPage?.facets.checklistFocuses ?? []} selected={catalogChecklistFocus} onChange={(next) => { setCatalogChecklistFocus(next); resetCatalogPage(); }} />
                  <label htmlFor="planning-intake-source-gap">Source gap<select id="planning-intake-source-gap" aria-label="Source gap filter" value={catalogSourceGapState} onChange={(event) => { setCatalogSourceGapState(event.target.value); resetCatalogPage(); }}><option value="">Any source context</option><option value="OPTIONAL_ENRICHMENT_NOT_PROVIDED">Optional enrichment unavailable</option><option value="SOURCE_CONTEXT_INCOMPLETE">Source context incomplete</option></select></label>
                  <label htmlFor="planning-intake-recommendation">Recommendation<select id="planning-intake-recommendation" aria-label="Recommendation filter" value={catalogRecommendationState} onChange={(event) => { setCatalogRecommendationState(event.target.value); resetCatalogPage(); }}><option value="">Full approved catalog</option><option value={defaultCatalogRecommendationState}>Suggested now (server included-by-default)</option>{(catalogPage?.facets.recommendationStates ?? []).filter((option) => option.value !== defaultCatalogRecommendationState).map((option) => <option key={option.value} value={option.value}>{catalogValueLabel(option.value)} · {option.count.toLocaleString("en-US")}</option>)}</select></label>
                  <label htmlFor="planning-intake-selected-state">Selected state<select id="planning-intake-selected-state" aria-label="Selected state filter" value={catalogSelectedFilter} onChange={(event) => { setCatalogSelectedFilter(event.target.value as typeof catalogSelectedFilter); resetCatalogPage(); }}><option value="all">All questions</option><option value="selected">Selected in scope</option><option value="unselected">Not selected</option></select></label>
                </div><button className="planning-intake-text-action" disabled={!activeFilterCount && catalogRecommendationState === defaultCatalogRecommendationState} type="button" onClick={() => { setCatalogFormCode([]); setCatalogDomain([]); setCatalogTopic([]); setCatalogRiskBand([]); setCatalogSourceGapState(""); setCatalogChecklistFocus([]); setCatalogRecommendationState(defaultCatalogRecommendationState); setCatalogSelectedFilter("all"); resetCatalogPage(); }}>Clear filters</button></details>
                <p className="planning-intake-result-count" aria-live="polite">{catalogPage?.totalCount.toLocaleString("en-US") ?? "0"} matching questions · page {catalogPageNumber}</p>
                {catalogBusy ? <p className="planning-intake-loading" role="status">Loading {fullCatalogSelected ? "full approved catalog" : "suggested questions"}…</p> : null}
                {!catalogBusy && !catalogPage ? <p className="planning-intake-loading" role="status">Catalog selection is unavailable in this build profile.</p> : null}
                {catalogPage ? <ul className="planning-intake-catalog-list">{catalogPage.items.map((question) => { const checked = pendingSelectionIds.includes(question.questionVersionId); const prompt = question.prompt ?? "Question prompt unavailable"; const questionInfo = [question.formCode, `item ${question.ordinal}`, `${catalogValueLabel(question.aiAdvisory.riskTier)} risk`, catalogValueLabel(question.recommendation.recommendationState), `${question.recommendation.historyCount.toLocaleString("en-US")} prior Audits`, ...question.recommendation.signalCodes.slice(0, 1).map(catalogValueLabel)].join(" · "); return <li data-question-version-id={question.questionVersionId} key={question.questionVersionId}><label><input aria-label={`Select ${question.formCode} item ${question.ordinal}`} checked={checked} disabled={busy || !question.canSelect} onChange={() => toggleQuestion(question.questionVersionId)} title={!question.canSelect ? "This question is not selectable in the current server-authorized scope." : undefined} type="checkbox" /><span><b className="planning-intake-question-prompt" title={prompt}>{prompt}</b><small className="planning-intake-question-info">{questionInfo}</small><small>{question.recommendation.rationale}</small></span></label><button className="planning-intake-question-detail" type="button" onClick={(event) => void openCatalogDetail(question, event.currentTarget)}>View details</button></li>; })}</ul> : null}
                <div className="planning-intake-selection-actions"><button disabled={busy || catalogBusy || !catalogPage} type="button" onClick={() => void addAllMatchingQuestions(defaultCatalogRecommendationState)}>Use suggested questions</button><details className="planning-intake-more-actions"><summary>More selection actions</summary><button disabled={busy || catalogBusy || !catalogPage} type="button" onClick={() => void addAllMatchingQuestions()}>Add all matching eligible questions</button></details></div>
                <section aria-label="Selection summary" className="planning-intake-selection-summary"><header><div><h3>Selection summary</h3><p>{selectionDelta.selectedCount.toLocaleString("en-US")} questions selected</p></div><div className="planning-intake-selection-summary__metrics"><span>Additions <b>{selectionDelta.additions.toLocaleString("en-US")}</b></span><span>Removals <b>{selectionDelta.removals.toLocaleString("en-US")}</b></span><span>Resource <b>{selectionSummary.complete ? `${selectionSummary.estimatedResourceRequirement ?? 0} question-hours` : "Server-derived after confirmation"}</b></span></div></header>{fieldErrors.selectedQuestionVersionIds ? <FieldError id="planning-intake-selectedQuestionVersionIds-error" message={fieldErrors.selectedQuestionVersionIds} /> : null}<div><button className={useReviewPrimary ? "planning-intake-primary" : "planning-intake-secondary"} disabled={busy || !selectionDirty} type="button" onClick={(event) => openSelectionReview(event.currentTarget)}>Review selection</button><button className="planning-intake-text-action" disabled={busy || !selectionDirty} type="button" onClick={() => { setPendingSelectionIds([...(values.selectedQuestionVersionIds ?? [])]); setSelectionDirty(false); setSelectionProgress({ completed: 0, total: 0, error: null }); setStatus("Selection changes were discarded."); }}>Undo changes</button></div></section>
                <div className="planning-intake-catalog-pagination" aria-label="Question pagination"><button disabled={catalogBusy || !catalogPreviousCursors.length} type="button" onClick={() => { const history = [...catalogPreviousCursors]; setCatalogCursor(history.pop()); setCatalogPreviousCursors(history); setCatalogPageNumber((current) => Math.max(1, current - 1)); }}>Previous questions</button><span aria-live="polite">Page {catalogPageNumber} · {catalogPage?.totalCount.toLocaleString("en-US") ?? "0"} matching</span><button disabled={catalogBusy || !catalogPage?.nextCursor} type="button" onClick={() => { if (!catalogPage?.nextCursor) return; setCatalogPreviousCursors((history) => [...history, catalogCursor ?? ""]); setCatalogCursor(catalogPage.nextCursor ?? undefined); setCatalogPageNumber((current) => current + 1); }}>Next questions</button></div>
              </section>
              <section aria-label="Resources" className="planning-intake-resources"><header><h3>Resources</h3><p>Finance Review is required even when the requested budget is zero.</p></header><label htmlFor="planning-intake-requestedBudget">Requested budget <RequiredMark /><input id="planning-intake-requestedBudget" aria-label="Requested Budget" aria-invalid={Boolean(fieldErrors.requestedBudget)} aria-describedby={fieldErrors.requestedBudget ? "planning-intake-requestedBudget-error" : undefined} min="0" type="number" value={values.requestedBudget} onBlur={() => validateField("requestedBudget")} onChange={(event) => update("requestedBudget", event.target.value)} /><FieldError id="planning-intake-requestedBudget-error" message={fieldErrors.requestedBudget} /></label><label htmlFor="planning-intake-currency">Currency<select id="planning-intake-currency" value={values.currency} onChange={(event) => update("currency", event.target.value as PlanningIntakeDraftValues["currency"])}><option value="USD">USD</option><option value="EUR">EUR</option><option value="NAD">NAD</option></select></label></section>
            </div> : null}
            {!routeRedirecting && values && step === 5 ? <div className="planning-intake-review">
              <section className="planning-intake-review-section"><header><h3>Inspection</h3><button type="button" onClick={() => editStep(1)}>Edit</button></header><dl><div><dt>Supplier / organization</dt><dd>{values.organizationName}</dd></div><div><dt>Provider scope</dt><dd>{selectedScopeOption?.providerTypeLabel ?? "Authorized scope"}</dd></div><div><dt>Regulated target</dt><dd>{selectedScopeOption?.targetLabel ?? "Authorized target"}</dd></div><div><dt>Inspection type</dt><dd>{catalogValueLabel(values.applicationType)}</dd></div><div><dt>Purpose</dt><dd>{values.purpose}</dd></div></dl></section>
              <section className="planning-intake-review-section"><header><h3>Schedule</h3><button type="button" onClick={() => editStep(3)}>Edit</button></header><dl><div><dt>Planned date</dt><dd>{readableLocalDate(values.plannedDate)}</dd></div><div><dt>Mode</dt><dd>{values.mode}</dd></div><div><dt>Location</dt><dd>{values.location}</dd></div></dl></section>
              <section className="planning-intake-review-section"><header><h3>Checklist</h3><button type="button" onClick={() => editStep(4)}>Edit</button></header><dl><div><dt>Questions selected</dt><dd>{pendingSelectionIds.length.toLocaleString("en-US")}</dd></div><div><dt>Resource requirement</dt><dd>{selectionSummary.complete ? `${selectionSummary.estimatedResourceRequirement ?? 0} question-hours` : "Server-derived selection summary unavailable"}</dd></div><div><dt>Form distribution</dt><dd>{selectionSummary.complete ? Object.entries(selectionSummary.formDistribution).map(([form, count]) => `${form}: ${count}`).join(" · ") || "None" : "Available after exact confirmation"}</dd></div></dl></section>
              <section className="planning-intake-review-section"><header><h3>Resources</h3><button type="button" onClick={() => editStep(4)}>Edit</button></header><dl><div><dt>Requested budget</dt><dd>{values.requestedBudget} {values.currency}</dd></div></dl></section>
              <section className="planning-intake-review-section"><header><h3>Notice &amp; governance</h3><button type="button" onClick={() => editStep(2)}>Edit</button></header><p>{values.inspectionCategory} · {noticeLabel(values.inspectionCategory)}</p><p>Submit creates a Planning item for Finance Review. It does not create an Audit or start an Inspector assignment.</p><p className="planning-intake-governance-path">Department Manager → Finance Review → General Manager → Executive Director → General Manager Release</p></section>
            </div> : null}
          </section>
          <InspectionBrief values={values} scopeOption={selectedScopeOption} pendingScopeOption={pendingScopeOption} pendingApplicationType={pendingApplicationType} selectedCount={currentSelectedCount} autosaveState={autosaveState} autosaveError={autosaveError} onRetry={() => void retryAutosave()} />
        </div>
        <section aria-label="Planning intake actions" className="planning-intake-actions"><div className="planning-intake-actions__secondary">{step === 1 ? <button className="planning-intake-secondary" type="button" onClick={cancel}>Cancel</button> : <button className="planning-intake-secondary" disabled={busy || !values} type="button" onClick={() => void moveBack()}>Back</button>}{autosaveState === "error" && draft ? <button className="planning-intake-text-action" type="button" onClick={() => void retryAutosave()}>Retry save</button> : null}</div><div className="planning-intake-actions__primary">{step < 5 ? <button className={useReviewPrimary ? "planning-intake-secondary" : "planning-intake-primary"} disabled={busy || !values && step > 1 || (step === 1 && !values && !canCreateDraft)} type="button" onClick={() => void continueFromStep()}>{actionLabel}</button> : <button className="planning-intake-primary" disabled={busy || !values} type="button" onClick={() => void submit()}>Submit to Finance</button>}</div></section>
      </div>
      {selectionReviewOpen ? <SelectionReviewDialog selectedCount={selectionDelta.selectedCount} additions={selectionDelta.additions} removals={selectionDelta.removals} total={selectionProgress.total} progress={selectionProgress} busy={busy} onConfirm={() => void confirmSelectionReview()} onRetry={retrySelectionConfirmation} onClose={closeSelectionReview} returnFocusRef={selectionReviewTriggerRef} /> : null}
      {catalogDetail ? <QuestionDossier question={catalogDetail} onClose={closeCatalogDetail} returnFocusRef={catalogTriggerRef} /> : null}
    </WorkspaceShell>
  );
}
