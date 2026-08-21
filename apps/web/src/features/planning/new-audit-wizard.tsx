import { useEffect, useMemo, useRef, useState, type ReactNode, type RefObject } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { z } from "zod";

import { useApplicationRuntime } from "../../app/providers";
import type {
  CanonicalApplicationType,
  CanonicalAuditScopeOption,
  CanonicalQuestionCatalogEntry,
  PlanningIntakeNoticePolicy,
  PlanningLocationOption,
  PlanningProposalDraftValues,
  PlanningProposalDraftView,
  PlanningProposalLocationInput,
  PlanningPurposePreset,
  PlanningResolvedLocation,
  PlanningWorkloadEstimate,
} from "../../backend/backend";
import { CommandError, errorMessage, formatLocalDate, WorkspaceShell } from "../shared/workspace-shell";
import { catalogValueLabel } from "./planning-intake-formatters";

const stepDefinitions = [
  { number: 1, label: "Scope", description: "Who or what will be inspected, and under which inspection type?" },
  { number: 2, label: "Purpose", description: "Why is this Audit being planned?" },
  { number: 3, label: "Schedule", description: "When and how will it take place?" },
  { number: 4, label: "Resources & budget", description: "What capacity and budget should Finance approve?" },
  { number: 5, label: "Review", description: "Is this the plan Finance and later approvers should review?" },
] as const;

type AutosaveState = "clean" | "dirty" | "saving" | "saved" | "error";
type FieldKey = "organizationId" | "providerScopeId" | "regulatedTargetId" | "inspectionType" | "purpose" | "plannedDate" | "location" | "meetingLink" | "requiredInspectorCount" | "estimatedChecklistItemCount" | "requestedBudget";
type FieldErrors = Partial<Record<FieldKey, string>>;

interface NewAuditFormValues extends Omit<PlanningProposalDraftValues, "requestedBudget" | "requiredInspectorCount" | "estimatedChecklistItemCount"> {
  requestedBudget: string;
  requiredInspectorCount: string;
  estimatedChecklistItemCount: string;
  meetingLink: string;
}

const budgetSchema = z.string().trim().min(1, "Requested budget is required").refine((value) => {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed >= 0;
}, "Requested budget must be zero or greater");

const stepSchemas: Record<number, z.ZodTypeAny> = {
  1: z.object({
    organizationId: z.string().min(1, "Inspected Organization is required"),
    providerScopeId: z.string().min(1, "Provider scope is required"),
    regulatedTargetId: z.string().min(1, "Regulated target is required"),
    inspectionType: z.string().min(1, "Inspection type is required"),
  }),
  2: z.object({ purpose: z.string().trim().min(1, "Purpose is required") }),
  3: z.object({
    plannedDate: z.string().min(1, "Planned date is required").regex(/^\d{4}-\d{2}-\d{2}$/, "Enter a valid planned date"),
    mode: z.enum(["On-site", "Remote"]),
    location: z.string().optional(),
    meetingLink: z.string().optional(),
  }).superRefine((value, context) => {
    if (value.mode === "On-site" && !value.location?.trim()) context.addIssue({ code: z.ZodIssueCode.custom, path: ["location"], message: "Location is required for an on-site Audit" });
    if (value.mode === "Remote" && value.meetingLink?.trim() && !/^https?:\/\//i.test(value.meetingLink.trim())) context.addIssue({ code: z.ZodIssueCode.custom, path: ["meetingLink"], message: "Use an HTTP(S) meeting link" });
  }),
  4: z.object({
    requiredInspectorCount: z.string().trim().min(1, "Required inspectors is required").refine((value) => Number.isInteger(Number(value)) && Number(value) > 0, "Enter a positive inspector count"),
    estimatedChecklistItemCount: z.string().trim().min(1, "Estimated checklist items is required").refine((value) => Number.isInteger(Number(value)) && Number(value) > 0, "Enter a positive checklist-item estimate"),
    requestedBudget: budgetSchema,
  }),
};

function operationId(prefix: string): string {
  return `${prefix}-${globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`}`.toUpperCase();
}

function stepFromPath(pathname: string): number {
  return Math.min(5, Math.max(1, Number(pathname.match(/step-(\d)$/)?.[1] ?? 1)));
}

function pathForStep(step: number, draftId?: string): string {
  return `/department-manager/new-audit/step-${step}${draftId ? `?draftId=${encodeURIComponent(draftId)}` : ""}`;
}

function readableDate(value: string | undefined): string {
  if (!value) return "Not set";
  try { return formatLocalDate(value); } catch { return value; }
}

function noticeLabel(policy: PlanningIntakeNoticePolicy): string {
  return policy === "WITHHELD" ? "Notice withheld until the authorized release boundary" : "Advance notice applies after the authorized release boundary";
}

function formValuesForDraft(draft: PlanningProposalDraftView): NewAuditFormValues {
  return {
    organizationId: draft.organizationId,
    providerScopeId: draft.providerScopeId,
    regulatedTargetId: draft.regulatedTargetId,
    inspectionType: draft.inspectionType,
    purpose: draft.purpose,
    purposePresetId: draft.purposePresetId,
    plannedDate: draft.plannedDate,
    mode: draft.mode,
    locationInput: draft.location
      ? draft.location.kind === "CANONICAL" && draft.location.locationId
        ? { kind: "CANONICAL", locationId: draft.location.locationId }
        : { kind: "NEW", proposedLabel: draft.location.label, acceptedResolutionToken: `HYDRATED-${draft.id}` }
      : undefined,
    meetingLink: draft.meetingLink ?? "",
    requiredInspectorCount: String(draft.requiredInspectorCount),
    estimatedChecklistItemCount: String(draft.estimatedChecklistItemCount),
    workloadEstimateId: draft.workloadEstimateId,
    workloadEstimateDigest: draft.workloadEstimateDigest,
    requestedBudget: draft.requestedBudget === null ? "" : String(draft.requestedBudget),
    currency: draft.currency,
  };
}

function valuesForCommand(values: NewAuditFormValues): PlanningProposalDraftValues {
  return {
    organizationId: values.organizationId,
    providerScopeId: values.providerScopeId,
    regulatedTargetId: values.regulatedTargetId,
    inspectionType: values.inspectionType,
    purpose: values.purpose.trim(),
    purposePresetId: values.purposePresetId,
    plannedDate: values.plannedDate,
    mode: values.mode,
    locationInput: values.locationInput,
    meetingLink: values.meetingLink.trim() || undefined,
    requiredInspectorCount: Number(values.requiredInspectorCount),
    estimatedChecklistItemCount: Number(values.estimatedChecklistItemCount),
    workloadEstimateId: values.workloadEstimateId,
    workloadEstimateDigest: values.workloadEstimateDigest,
    requestedBudget: values.requestedBudget.trim() === "" ? null : Number(values.requestedBudget),
    currency: values.currency,
  };
}

function FieldError({ id, message }: { id: string; message?: string }): ReactNode {
  return message ? <span className="planning-intake-field-error" id={id} role="alert">{message}</span> : null;
}

function RequiredMark(): ReactNode { return <span aria-hidden="true" className="planning-intake-required">*</span>; }

function ValidationSummary({ errors, onFocus }: { errors: FieldErrors; onFocus: (field: FieldKey) => void }) {
  const entries = Object.entries(errors) as Array<[FieldKey, string]>;
  if (entries.length < 2) return null;
  return <div className="planning-intake-error-summary" role="alert"><b>Review the highlighted fields</b><ul>{entries.map(([field, message]) => <li key={field}><button type="button" onClick={() => onFocus(field)}>{message}</button></li>)}</ul></div>;
}

function AutosaveIndicator({ state, error, onRetry }: { state: AutosaveState; error: string | null; onRetry: () => void }) {
  if (state === "error") return <span className="planning-intake-autosave is-error"><span role="alert">Couldn’t save</span><button type="button" onClick={onRetry}>Retry</button>{error ? <small>{error}</small> : null}</span>;
  const label = state === "saving" ? "Saving…" : state === "dirty" ? "Not saved" : state === "saved" ? "Saved" : "Not saved";
  return <span className={`planning-intake-autosave is-${state}`}>{label}</span>;
}

function PlanningIntakeProgress({ step }: { step: number }) {
  return <div className="planning-intake-progress" aria-label="New Audit progress">
    <ol className="planning-intake-progress__desktop" aria-label="New Audit steps">{stepDefinitions.map((definition) => <li aria-current={definition.number === step ? "step" : undefined} className={definition.number === step ? "is-current" : definition.number < step ? "is-complete" : ""} key={definition.number}><span>{definition.number < step ? "✓" : definition.number}</span><b>{definition.label}</b></li>)}</ol>
    <div className="planning-intake-progress__mobile"><p><strong>Step {step} of 5</strong><span> · {stepDefinitions[step - 1]?.label}</span></p><details><summary>View all steps</summary><ol>{stepDefinitions.map((definition) => <li className={definition.number === step ? "is-current" : definition.number < step ? "is-complete" : ""} key={definition.number}><span>{definition.number < step ? "✓" : definition.number}</span><b>{definition.label}</b></li>)}</ol></details></div>
  </div>;
}

function AuditPlanSummary({ draft, values, option, estimate, autosaveState, autosaveError, onRetry }: { draft: PlanningProposalDraftView | null; values: NewAuditFormValues | null; option: CanonicalAuditScopeOption | null; estimate: PlanningWorkloadEstimate | null; autosaveState: AutosaveState; autosaveError: string | null; onRetry: () => void }) {
  const organization = draft?.organizationName || option?.organizationName || "Choose an organization";
  const facts: Array<[string, string]> = [];
  const provider = draft?.providerScopeLabel || option?.providerTypeLabel;
  const target = draft?.regulatedTargetLabel || option?.targetLabel;
  if (provider) facts.push(["Provider scope", provider]);
  if (target) facts.push(["Regulated target", target]);
  if (values?.inspectionType) facts.push(["Inspection type", catalogValueLabel(values.inspectionType)]);
  if (values?.purpose.trim()) facts.push(["Purpose", values.purpose.trim()]);
  if (values?.plannedDate) facts.push(["Planned date", readableDate(values.plannedDate)]);
  if (values?.mode) facts.push(["Mode", values.mode]);
  if (values?.mode === "On-site" && draft?.location?.label) facts.push(["Location", draft.location.label]);
  if (values?.mode === "Remote" && values.meetingLink.trim()) facts.push(["Meeting link", values.meetingLink.trim()]);
  if (values?.requiredInspectorCount) facts.push(["Required inspectors", values.requiredInspectorCount]);
  if (values?.estimatedChecklistItemCount) facts.push(["Estimated checklist items", values.estimatedChecklistItemCount]);
  if (values?.requestedBudget.trim()) facts.push(["Requested budget", `${values.requestedBudget} ${values.currency}`]);
  const body = <dl className="planning-intake-brief__facts"><div><dt>Inspected Organization</dt><dd>{organization}</dd></div>{facts.map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}{estimate && values?.mode ? <div><dt>Workload basis</dt><dd>{estimate.basisLabel}</dd></div> : null}</dl>;
  const header = <header><div><span className="planning-intake-brief__eyebrow">Decision context</span><h2>Audit plan summary</h2></div><AutosaveIndicator state={draft ? autosaveState : "clean"} error={autosaveError} onRetry={onRetry} /></header>;
  return <aside aria-label="Audit plan summary" className="planning-intake-brief"><div className="planning-intake-brief__desktop">{header}{body}</div><details className="planning-intake-brief__mobile"><summary><span>Audit plan summary · {organization}</span><AutosaveIndicator state={draft ? autosaveState : "clean"} error={autosaveError} onRetry={onRetry} /></summary>{body}</details></aside>;
}

function ScopeChoice({ label, value, automatic, children }: { label: string; value: string; automatic?: boolean; children?: ReactNode }) {
  return <div className="planning-intake-scope-choice"><div><span>{label}</span><strong>{value || "Select an option"}</strong>{automatic ? <small>Automatically selected</small> : null}</div>{children}</div>;
}

function WorkloadPreview({ open, onClose, rows, busy, query, onQuery, total, onUseCount, returnFocusRef }: { open: boolean; onClose: () => void; rows: CanonicalQuestionCatalogEntry[]; busy: boolean; query: string; onQuery: (value: string) => void; total: number; onUseCount: () => void; returnFocusRef: RefObject<HTMLElement | null> }) {
  const dialogRef = useRef<HTMLElement | null>(null);
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;
  useEffect(() => {
    if (!open) return undefined;
    const backgroundNodes = [document.querySelector<HTMLElement>(".planning-intake-layout"), document.querySelector<HTMLElement>(".planning-intake-actions")].filter((node): node is HTMLElement => Boolean(node));
    backgroundNodes.forEach((node) => { node.setAttribute("inert", ""); node.setAttribute("aria-hidden", "true"); });
    dialogRef.current?.querySelector<HTMLElement>("button, input")?.focus();
    const focusableSelector = "button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex=\"-1\"])";
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") { event.preventDefault(); onCloseRef.current(); return; }
      if (event.key !== "Tab" || !dialogRef.current) return;
      const focusable = [...dialogRef.current.querySelectorAll<HTMLElement>(focusableSelector)];
      if (!focusable.length) return;
      const first = focusable[0]; const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
    };
    dialogRef.current?.addEventListener("keydown", onKeyDown);
    return () => {
      dialogRef.current?.removeEventListener("keydown", onKeyDown);
      backgroundNodes.forEach((node) => { node.removeAttribute("inert"); node.removeAttribute("aria-hidden"); });
      window.setTimeout(() => returnFocusRef.current?.focus(), 0);
    };
  }, [open, returnFocusRef]);
  if (!open) return null;
  return <div className="planning-intake-dossier-backdrop" role="presentation"><section className="planning-intake-workload-drawer" ref={dialogRef} role="dialog" aria-modal="true" aria-label="Checklist item preview"><header><div><span className="planning-intake-dialog-kicker">Read-only preview</span><h2>Browse checklist items</h2></div><button type="button" onClick={onClose}>Close</button></header><p>This preview helps estimate volume. It does not select or freeze checklist items.</p><label htmlFor="planning-intake-preview-search">Search checklist items<input id="planning-intake-preview-search" value={query} onChange={(event) => onQuery(event.target.value)} placeholder="Search item text or reference" /></label><p className="planning-intake-preview-count" aria-live="polite">{total.toLocaleString("en-US")} matching items</p>{busy ? <p className="planning-intake-loading" role="status">Loading preview…</p> : <ul className="planning-intake-preview-list">{rows.map((row) => <li key={row.questionVersionId}><strong>{row.formCode} · item {row.ordinal}</strong><span>{row.prompt ?? "Checklist item text unavailable"}</span></li>)}</ul>}<footer><button className="planning-intake-secondary" type="button" onClick={onClose}>Close preview</button><button className="planning-intake-primary" type="button" onClick={onUseCount} disabled={busy || total === 0}>Use this count</button></footer></section></div>;
}

function NewAuditWizardPage() {
  const runtime = useApplicationRuntime();
  const managerBackend = runtime.backendForRole?.("manager") ?? runtime.backend;
  const proposal = managerBackend.planningProposal ?? runtime.backend.planningProposal;
  const canonicalCatalog = managerBackend.canonicalCatalog ?? runtime.backend.canonicalCatalog;
  const navigate = useNavigate();
  const location = useLocation();
  const requestedDraftId = new URLSearchParams(location.search).get("draftId");
  const requestedStep = stepFromPath(location.pathname);
  const step = requestedDraftId ? requestedStep : 1;

  const [scopeOptions, setScopeOptions] = useState<CanonicalAuditScopeOption[]>([]);
  const [purposePresets, setPurposePresets] = useState<PlanningPurposePreset[]>([]);
  const [locations, setLocations] = useState<PlanningLocationOption[]>([]);
  const [pendingOrganizationId, setPendingOrganizationId] = useState("");
  const [pendingProviderScopeId, setPendingProviderScopeId] = useState("");
  const [pendingRegulatedTargetId, setPendingRegulatedTargetId] = useState("");
  const [pendingInspectionType, setPendingInspectionType] = useState("");
  const [draft, setDraft] = useState<PlanningProposalDraftView | null>(null);
  const [values, setValues] = useState<NewAuditFormValues | null>(null);
  const [estimate, setEstimate] = useState<PlanningWorkloadEstimate | null>(null);
  const [serverError, setServerError] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [busy, setBusy] = useState(false);
  const [autosaveState, setAutosaveState] = useState<AutosaveState>("clean");
  const [autosaveError, setAutosaveError] = useState<string | null>(null);
  const [locationEditing, setLocationEditing] = useState(false);
  const [manualLocation, setManualLocation] = useState("");
  const [meetingLinkOpen, setMeetingLinkOpen] = useState(false);
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewRows, setPreviewRows] = useState<CanonicalQuestionCatalogEntry[]>([]);
  const [previewQuery, setPreviewQuery] = useState("");
  const [previewTotal, setPreviewTotal] = useState(0);
  const [previewBusy, setPreviewBusy] = useState(false);
  const previewTriggerRef = useRef<HTMLElement | null>(null);
  const headingRef = useRef<HTMLHeadingElement | null>(null);
  const valuesRef = useRef<NewAuditFormValues | null>(null);
  const draftRef = useRef<PlanningProposalDraftView | null>(null);
  const autosaveQueueRef = useRef<NewAuditFormValues | null>(null);
  const autosaveFlightRef = useRef<Promise<PlanningProposalDraftView> | null>(null);
  const autosaveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const autosaveKeyRef = useRef<string | null>(null);
  const autosaveSequenceRef = useRef(0);

  const setDraftState = (next: PlanningProposalDraftView) => { draftRef.current = next; setDraft(next); setEstimate(next.workloadEstimate); };
  const setFormState = (next: NewAuditFormValues | null) => { valuesRef.current = next; setValues(next); };
  const selectedOption = useMemo(() => {
    const organizationId = values?.organizationId ?? pendingOrganizationId;
    const providerScopeId = values?.providerScopeId ?? pendingProviderScopeId;
    const regulatedTargetId = values?.regulatedTargetId ?? pendingRegulatedTargetId;
    return scopeOptions.find((option) => option.organizationId === organizationId && option.providerScopeId === providerScopeId && option.regulatedTargetId === regulatedTargetId) ?? null;
  }, [pendingOrganizationId, pendingProviderScopeId, pendingRegulatedTargetId, scopeOptions, values]);
  const organizationOptions = useMemo(() => [...new Map(scopeOptions.map((option) => [option.organizationId, option])).values()], [scopeOptions]);
  const providerOptions = useMemo(() => scopeOptions.filter((option) => option.organizationId === (values?.organizationId ?? pendingOrganizationId)).filter((option, index, all) => all.findIndex((candidate) => candidate.providerScopeId === option.providerScopeId) === index), [pendingOrganizationId, scopeOptions, values?.organizationId]);
  const targetOptions = useMemo(() => scopeOptions.filter((option) => option.organizationId === (values?.organizationId ?? pendingOrganizationId) && option.providerScopeId === (values?.providerScopeId ?? pendingProviderScopeId)), [pendingOrganizationId, pendingProviderScopeId, scopeOptions, values?.organizationId, values?.providerScopeId]);
  const inspectionTypeOptions = selectedOption?.inspectionTypes ?? [];
  const currentDefinition = stepDefinitions[step - 1] ?? stepDefinitions[0];

  useEffect(() => { draftRef.current = draft; }, [draft]);
  useEffect(() => { valuesRef.current = values; }, [values]);
  useEffect(() => () => { if (autosaveTimerRef.current) clearTimeout(autosaveTimerRef.current); }, []);
  useEffect(() => { headingRef.current?.focus(); }, [step, requestedDraftId]);

  useEffect(() => {
    let cancelled = false;
    if (!proposal) { setServerError("New Audit planning is unavailable in this build profile."); return () => { cancelled = true; }; }
    if (requestedDraftId && draftRef.current?.id === requestedDraftId) return () => { cancelled = true; };
    void Promise.all([proposal.listScopeOptions({ limit: 25 }), proposal.listPurposePresets()]).then(async ([scopePage, presets]) => {
      if (cancelled) return;
      setScopeOptions(scopePage.items); setPurposePresets(presets);
      if (requestedDraftId) {
        const loaded = await proposal.getDraft({ draftId: requestedDraftId });
        if (cancelled) return;
        setDraftState(loaded); setFormState(formValuesForDraft(loaded));
        setPendingOrganizationId(loaded.organizationId); setPendingProviderScopeId(loaded.providerScopeId); setPendingRegulatedTargetId(loaded.regulatedTargetId); setPendingInspectionType(loaded.inspectionType);
        const loadedLocations = await proposal.listLocations({ organizationId: loaded.organizationId, regulatedTargetId: loaded.regulatedTargetId });
        if (!cancelled) { setLocations(loadedLocations); setAutosaveState("saved"); }
      } else {
        const first = scopePage.items[0];
        if (first) {
          setPendingOrganizationId(first.organizationId); setPendingProviderScopeId(first.providerScopeId); setPendingRegulatedTargetId(first.regulatedTargetId); setPendingInspectionType(first.inspectionTypes[0] ?? "");
          const firstLocations = await proposal.listLocations({ organizationId: first.organizationId, regulatedTargetId: first.regulatedTargetId });
          if (!cancelled) setLocations(firstLocations);
        }
      }
    }).catch((cause) => { if (!cancelled) { if (requestedDraftId) navigate(pathForStep(1), { replace: true }); setServerError(errorMessage(cause)); } });
    return () => { cancelled = true; };
  }, [navigate, proposal, requestedDraftId]);

  useEffect(() => {
    if (!previewOpen || !canonicalCatalog || !estimate || !values) return undefined;
    const controller = new AbortController(); setPreviewBusy(true);
    void canonicalCatalog.listCatalog({ catalogVersion: estimate.catalogVersion, usageClass: "GOVERNED_OPERATIONAL", search: previewQuery || undefined, applicationType: values.inspectionType as CanonicalApplicationType, limit: 50, projection: "selection" }, { signal: controller.signal }).then((page) => { if (!controller.signal.aborted) { setPreviewRows(page.items.slice(0, 50)); setPreviewTotal(page.totalCount); } }).catch((cause) => { if (!controller.signal.aborted) setServerError(errorMessage(cause)); }).finally(() => { if (!controller.signal.aborted) setPreviewBusy(false); });
    return () => controller.abort();
  }, [canonicalCatalog, estimate, previewOpen, previewQuery, values]);

  function focusField(field: FieldKey) { document.getElementById(`planning-intake-${field}`)?.focus(); }
  function clearFieldError(field: FieldKey) { setFieldErrors((current) => { if (!current[field]) return current; const next = { ...current }; delete next[field]; return next; }); }
  function updateForm<K extends keyof NewAuditFormValues>(key: K, value: NewAuditFormValues[K]) {
    const current = valuesRef.current; if (!current) return;
    const next = { ...current, [key]: value }; setFormState(next); queueAutosave(next); clearFieldError(key as FieldKey); setStatus(null);
  }
  function errorsForStep(currentStep: number, candidate: unknown): FieldErrors {
    const schema = stepSchemas[currentStep]; if (!schema) return {};
    const result = schema.safeParse(candidate); if (result.success) return {};
    const next: FieldErrors = {}; for (const issue of result.error.issues) { const field = issue.path[0] as FieldKey | undefined; if (field && !next[field]) next[field] = issue.message; } return next;
  }
  function candidateForStep(currentStep: number): Record<string, unknown> | null {
    const current = valuesRef.current;
    if (!current) return currentStep === 1 ? { organizationId: pendingOrganizationId, providerScopeId: pendingProviderScopeId, regulatedTargetId: pendingRegulatedTargetId, inspectionType: pendingInspectionType } : null;
    return currentStep === 3 ? { plannedDate: current.plannedDate, mode: current.mode, location: current.mode === "On-site" ? draftRef.current?.location?.label : "", meetingLink: current.meetingLink } : { ...current };
  }
  function validateStep(currentStep: number): boolean {
    const next = errorsForStep(currentStep, candidateForStep(currentStep)); setFieldErrors(next); const first = Object.keys(next)[0] as FieldKey | undefined; if (first) window.setTimeout(() => focusField(first), 0); return !first;
  }
  function validateAll(): boolean {
    const next: FieldErrors = {}; for (const currentStep of [1, 2, 3, 4]) Object.assign(next, errorsForStep(currentStep, candidateForStep(currentStep))); setFieldErrors(next); const first = Object.keys(next)[0] as FieldKey | undefined; if (first) window.setTimeout(() => focusField(first), 0); return !first;
  }

  async function saveQueued(): Promise<PlanningProposalDraftView | null> {
    const currentDraft = draftRef.current; const queued = autosaveQueueRef.current; if (!currentDraft || !queued || !proposal) return currentDraft;
    autosaveQueueRef.current = null; setAutosaveState("saving"); setAutosaveError(null);
    const idempotencyKey = autosaveKeyRef.current ?? `SAVE-${currentDraft.id}-R${currentDraft.revision}-${++autosaveSequenceRef.current}`; autosaveKeyRef.current = idempotencyKey;
    const request = proposal.saveDraft({ draftId: currentDraft.id, expectedRevision: currentDraft.revision, idempotencyKey, operationId: idempotencyKey, values: valuesForCommand(queued) }); autosaveFlightRef.current = request;
    try { const saved = await request; autosaveFlightRef.current = null; autosaveKeyRef.current = null; setDraftState(saved); if (valuesRef.current && !autosaveQueueRef.current) setFormState(formValuesForDraft(saved)); setAutosaveState(autosaveQueueRef.current ? "dirty" : "saved"); return saved; } catch (cause) { autosaveFlightRef.current = null; if (!autosaveQueueRef.current) autosaveQueueRef.current = queued; setAutosaveState("error"); setAutosaveError(errorMessage(cause)); throw cause; }
  }
  async function flushAutosave(nextValues = valuesRef.current): Promise<PlanningProposalDraftView | null> {
    if (!draftRef.current || !nextValues) return draftRef.current; autosaveQueueRef.current = nextValues; if (autosaveTimerRef.current) clearTimeout(autosaveTimerRef.current); let latest = draftRef.current;
    while (autosaveFlightRef.current || autosaveQueueRef.current) latest = (autosaveFlightRef.current ? await autosaveFlightRef.current : await saveQueued()) ?? latest; return latest;
  }
  function queueAutosave(next: NewAuditFormValues) {
    if (!draftRef.current || !proposal) return; autosaveQueueRef.current = next; setAutosaveState("dirty"); setAutosaveError(null); if (autosaveTimerRef.current) clearTimeout(autosaveTimerRef.current); autosaveTimerRef.current = setTimeout(() => { autosaveTimerRef.current = null; void flushAutosave().catch(() => undefined); }, 650);
  }
  async function retryAutosave() { try { await flushAutosave(); } catch { /* the visible retry state remains */ } }

  async function resolveManualLocation() {
    if (!proposal || !valuesRef.current || !manualLocation.trim()) return;
    try {
      const resolutionOperationId = operationId("PLANNING-LOCATION-RESOLVE");
      const resolution = await proposal.resolveLocation({ operationId: resolutionOperationId, idempotencyKey: resolutionOperationId, organizationId: valuesRef.current.organizationId, regulatedTargetId: valuesRef.current.regulatedTargetId, proposedLabel: manualLocation.trim() });
      const locationInput: PlanningProposalLocationInput = resolution.outcome === "CANONICAL" && resolution.location ? { kind: "CANONICAL", locationId: resolution.location.id } : { kind: "NEW", proposedLabel: manualLocation.trim(), acceptedResolutionToken: resolution.acceptedResolutionToken };
      updateForm("locationInput", locationInput); if (resolution.location) setLocations((current) => [resolution.location!, ...current.filter((location) => location.id !== resolution.location?.id)]); setLocationEditing(false); setStatus(resolution.message);
    } catch (cause) { setServerError(errorMessage(cause)); }
  }

  async function updateScopeSelection(next: { organizationId: string; providerScopeId: string; regulatedTargetId: string; inspectionType: string }) {
    const current = valuesRef.current; const hasDownstream = Boolean(current?.purpose.trim() || current?.plannedDate || current?.requestedBudget.trim());
    if (current && hasDownstream && !globalThis.confirm("Changing the inspected scope recalculates the location and workload estimate while retaining your authored purpose, schedule, and budget. Continue?")) return;
    setPendingOrganizationId(next.organizationId); setPendingProviderScopeId(next.providerScopeId); setPendingRegulatedTargetId(next.regulatedTargetId); setPendingInspectionType(next.inspectionType);
    if (!current || !proposal) return;
    try {
      const estimateOperationId = operationId("PLANNING-WORKLOAD-ESTIMATE");
      const nextEstimate = await proposal.getWorkloadEstimate({ ...next, operationId: estimateOperationId, idempotencyKey: estimateOperationId }); const nextLocations = await proposal.listLocations({ organizationId: next.organizationId, regulatedTargetId: next.regulatedTargetId }); setLocations(nextLocations); setEstimate(nextEstimate);
      updateForm("organizationId", next.organizationId); updateForm("providerScopeId", next.providerScopeId); updateForm("regulatedTargetId", next.regulatedTargetId); updateForm("inspectionType", next.inspectionType); updateForm("workloadEstimateId", nextEstimate.estimateId); updateForm("workloadEstimateDigest", nextEstimate.estimateDigest); updateForm("estimatedChecklistItemCount", String(nextEstimate.suggestedCount)); updateForm("locationInput", current.mode === "On-site" && nextLocations[0] ? { kind: "CANONICAL", locationId: nextLocations[0].id } : undefined);
    } catch (cause) { setServerError(errorMessage(cause)); }
  }

  async function createDraft() {
    if (!proposal || !selectedOption || !pendingInspectionType || !validateStep(1)) return;
    setBusy(true); setServerError(null);
    try {
      const estimateOperationId = operationId("PLANNING-WORKLOAD-ESTIMATE");
      const nextEstimate = await proposal.getWorkloadEstimate({ operationId: estimateOperationId, idempotencyKey: estimateOperationId, organizationId: selectedOption.organizationId, providerScopeId: selectedOption.providerScopeId, regulatedTargetId: selectedOption.regulatedTargetId, inspectionType: pendingInspectionType });
      const nextLocations = await proposal.listLocations({ organizationId: selectedOption.organizationId, regulatedTargetId: selectedOption.regulatedTargetId });
      const initialValues: PlanningProposalDraftValues = { organizationId: selectedOption.organizationId, providerScopeId: selectedOption.providerScopeId, regulatedTargetId: selectedOption.regulatedTargetId, inspectionType: pendingInspectionType, purpose: "", plannedDate: "", mode: "On-site", locationInput: nextLocations[0] ? { kind: "CANONICAL", locationId: nextLocations[0].id } : undefined, meetingLink: undefined, requiredInspectorCount: 2, estimatedChecklistItemCount: nextEstimate.suggestedCount, workloadEstimateId: nextEstimate.estimateId, workloadEstimateDigest: nextEstimate.estimateDigest, requestedBudget: null, currency: "USD" };
      const createOperationId = operationId("PLANNING-PROPOSAL-CREATE"); const created = await proposal.createDraft({ operationId: createOperationId, idempotencyKey: createOperationId, expectedRevision: null, values: initialValues });
      setLocations(nextLocations); setDraftState(created); setFormState(formValuesForDraft(created)); setAutosaveState("saved"); setStatus("Draft created. Your changes will save automatically."); navigate(pathForStep(2, created.id), { replace: true });
    } catch (cause) { setServerError(errorMessage(cause)); } finally { setBusy(false); }
  }
  async function continueFromStep() {
    if (step === 1 && !draftRef.current) { await createDraft(); return; }
    if (!valuesRef.current || !draftRef.current || !validateStep(step)) return;
    setBusy(true); setServerError(null); try { const saved = await flushAutosave(valuesRef.current); navigate(pathForStep(step + 1, saved?.id ?? draftRef.current?.id)); } catch (cause) { setServerError(errorMessage(cause)); } finally { setBusy(false); }
  }
  async function moveBack() {
    if (step <= 1 || !valuesRef.current || !draftRef.current) { navigate("/department-manager/audit-plan"); return; }
    setBusy(true); try { const saved = await flushAutosave(valuesRef.current); navigate(pathForStep(step - 1, saved?.id ?? draftRef.current.id)); } catch (cause) { setServerError(errorMessage(cause)); } finally { setBusy(false); }
  }
  function cancel() { const needsConfirmation = Boolean(draftRef.current && (autosaveState === "dirty" || autosaveState === "saving" || autosaveState === "error")); if (needsConfirmation && !globalThis.confirm("You have changes that are not fully saved. Leave New Audit?")) return; navigate("/department-manager/audit-plan"); }
  async function submit() {
    if (!proposal || !draftRef.current || !valuesRef.current || !validateAll()) return;
    setBusy(true); setServerError(null); try { const saved = await flushAutosave(valuesRef.current); if (!saved) return; const submitOperationId = operationId("PLANNING-PROPOSAL-SUBMIT"); const output = await proposal.submit({ draftId: saved.id, expectedRevision: saved.revision, idempotencyKey: submitOperationId, operationId: submitOperationId }); navigate(`/department-manager/audit-plan?planningItemId=${encodeURIComponent(output.planningItem.id)}`); } catch (cause) { setServerError(errorMessage(cause)); } finally { setBusy(false); }
  }
  function editStep(targetStep: number) { navigate(pathForStep(targetStep, draftRef.current?.id)); }

  const currentLocation: PlanningResolvedLocation | null = draft?.location ?? null;
  const workloadWarning = estimate && values ? Number(values.estimatedChecklistItemCount) < estimate.safeMinimum || Number(values.estimatedChecklistItemCount) > estimate.safeMaximum : false;
  const rosterWarning = estimate && values ? Number(values.requiredInspectorCount) > estimate.eligibleRosterCount : false;
  const actionLabel = busy ? (step === 1 && !draft ? "Creating draft…" : step === 5 ? "Submitting…" : "Saving…") : step === 4 ? "Continue to review" : "Continue";

  return <WorkspaceShell roleLabel="Department Manager" routeLabel={`New Audit · ${currentDefinition.label}`}>
    <div className="planning-intake-page" data-draft-id={draft?.id} data-testid="new-audit-wizard-page">
      <header className="planning-intake-header workbench-page-header"><div><span className="planning-intake-eyebrow">Department Manager planning</span><h1>New Audit</h1><p>Create a finance-reviewable plan. Submission creates a Planning item, not an executable Audit or final checklist.</p></div></header>
      <PlanningIntakeProgress step={step} />
      {serverError ? <CommandError message={serverError} /> : null}
      {status ? <p className="planning-intake-status" role="status">{status}</p> : null}
      <ValidationSummary errors={fieldErrors} onFocus={focusField} />
      <div className="planning-intake-layout">
        <section aria-label="New Audit form" className="planning-intake-form">
          <header className="planning-intake-form__header"><span>Step {step} of 5</span><h2 ref={headingRef} tabIndex={-1}>{currentDefinition.label}</h2><p aria-live="polite">{currentDefinition.description}</p></header>
          {!proposal ? <p className="planning-intake-loading" role="status">New Audit planning is unavailable in this build profile.</p> : null}
          {proposal && !scopeOptions.length ? <p className="planning-intake-loading" role="status">Loading authorized scope…</p> : null}
          {step > 1 && !values ? <p className="planning-intake-loading" role="status">Loading saved New Audit draft…</p> : null}

          {step === 1 && !values ? <div className="planning-intake-fields planning-intake-scope-fields">
            <label htmlFor="planning-intake-organizationId">Inspected Organization <RequiredMark /><select id="planning-intake-organizationId" aria-label="Inspected Organization" aria-invalid={Boolean(fieldErrors.organizationId)} aria-describedby={fieldErrors.organizationId ? "planning-intake-organizationId-error" : undefined} disabled={busy || !organizationOptions.length} value={pendingOrganizationId} onChange={(event) => { const first = scopeOptions.find((option) => option.organizationId === event.target.value); if (first) void updateScopeSelection({ organizationId: first.organizationId, providerScopeId: first.providerScopeId, regulatedTargetId: first.regulatedTargetId, inspectionType: first.inspectionTypes[0] ?? "" }); }}>{organizationOptions.map((option) => <option key={option.organizationId} value={option.organizationId}>{option.organizationName}</option>)}</select><small>Choose the organization whose operation or site is in scope for this Audit.</small><FieldError id="planning-intake-organizationId-error" message={fieldErrors.organizationId} /></label>
            <fieldset><legend>Operation / site</legend><div className="planning-intake-scope-facts"><ScopeChoice label="Provider scope" value={selectedOption?.providerTypeLabel ?? ""} automatic={providerOptions.length <= 1}>{providerOptions.length > 1 ? <select id="planning-intake-providerScopeId" aria-label="Provider scope" value={pendingProviderScopeId} onChange={(event) => { const next = scopeOptions.find((option) => option.organizationId === pendingOrganizationId && option.providerScopeId === event.target.value); if (next) void updateScopeSelection({ organizationId: next.organizationId, providerScopeId: next.providerScopeId, regulatedTargetId: next.regulatedTargetId, inspectionType: next.inspectionTypes[0] ?? "" }); }}>{providerOptions.map((option) => <option key={option.providerScopeId} value={option.providerScopeId}>{option.providerTypeLabel}</option>)}</select> : null}</ScopeChoice><ScopeChoice label="Regulated target" value={selectedOption?.targetLabel ?? ""} automatic={targetOptions.length <= 1}>{targetOptions.length > 1 ? <select id="planning-intake-regulatedTargetId" aria-label="Regulated target" value={pendingRegulatedTargetId} onChange={(event) => { const next = scopeOptions.find((option) => option.organizationId === pendingOrganizationId && option.providerScopeId === pendingProviderScopeId && option.regulatedTargetId === event.target.value); if (next) void updateScopeSelection({ organizationId: next.organizationId, providerScopeId: next.providerScopeId, regulatedTargetId: next.regulatedTargetId, inspectionType: next.inspectionTypes[0] ?? "" }); }}>{targetOptions.map((option) => <option key={option.regulatedTargetId} value={option.regulatedTargetId}>{option.targetLabel}</option>)}</select> : null}</ScopeChoice></div></fieldset>
            <label htmlFor="planning-intake-inspectionType">Inspection type <RequiredMark /><select id="planning-intake-inspectionType" aria-label="Inspection type" aria-invalid={Boolean(fieldErrors.inspectionType)} aria-describedby={fieldErrors.inspectionType ? "planning-intake-inspectionType-error" : undefined} disabled={busy || !inspectionTypeOptions.length} value={pendingInspectionType} onChange={(event) => { setPendingInspectionType(event.target.value); clearFieldError("inspectionType"); }}>{inspectionTypeOptions.map((type) => <option key={type} value={type}>{catalogValueLabel(type)}</option>)}</select><small>The inspection type affects the authorized workload estimate and later history.</small><FieldError id="planning-intake-inspectionType-error" message={fieldErrors.inspectionType} /></label>
            <p className="planning-intake-boundary-note" role="note">Continuing creates a Planning draft only. Checklist selection and named inspector assignment happen after the required approval and release boundary.</p>
          </div> : null}

          {step === 1 && values ? <div className="planning-intake-fields planning-intake-scope-fields">
            <label htmlFor="planning-intake-existing-organization">Inspected Organization <RequiredMark /><select id="planning-intake-existing-organization" value={values.organizationId} onChange={(event) => { const option = scopeOptions.find((candidate) => candidate.organizationId === event.target.value); if (option) void updateScopeSelection({ organizationId: option.organizationId, providerScopeId: option.providerScopeId, regulatedTargetId: option.regulatedTargetId, inspectionType: option.inspectionTypes[0] ?? values.inspectionType }); }}>{organizationOptions.map((option) => <option key={option.organizationId} value={option.organizationId}>{option.organizationName}</option>)}</select></label>
            <fieldset><legend>Operation / site</legend><div className="planning-intake-scope-facts"><ScopeChoice label="Provider scope" value={draft?.providerScopeLabel ?? selectedOption?.providerTypeLabel ?? ""} automatic={providerOptions.length <= 1}>{providerOptions.length > 1 ? <select id="planning-intake-providerScopeId" aria-label="Provider scope" value={values.providerScopeId} onChange={(event) => { const next = scopeOptions.find((option) => option.organizationId === values.organizationId && option.providerScopeId === event.target.value); if (next) void updateScopeSelection({ organizationId: next.organizationId, providerScopeId: next.providerScopeId, regulatedTargetId: next.regulatedTargetId, inspectionType: next.inspectionTypes[0] ?? values.inspectionType }); }}>{providerOptions.map((option) => <option key={option.providerScopeId} value={option.providerScopeId}>{option.providerTypeLabel}</option>)}</select> : null}</ScopeChoice><ScopeChoice label="Regulated target" value={draft?.regulatedTargetLabel ?? selectedOption?.targetLabel ?? ""} automatic={targetOptions.length <= 1}>{targetOptions.length > 1 ? <select id="planning-intake-regulatedTargetId" aria-label="Regulated target" value={values.regulatedTargetId} onChange={(event) => { const next = scopeOptions.find((option) => option.organizationId === values.organizationId && option.providerScopeId === values.providerScopeId && option.regulatedTargetId === event.target.value); if (next) void updateScopeSelection({ organizationId: next.organizationId, providerScopeId: next.providerScopeId, regulatedTargetId: next.regulatedTargetId, inspectionType: next.inspectionTypes[0] ?? values.inspectionType }); }}>{targetOptions.map((option) => <option key={option.regulatedTargetId} value={option.regulatedTargetId}>{option.targetLabel}</option>)}</select> : null}</ScopeChoice></div></fieldset>
            <label htmlFor="planning-intake-existing-inspectionType">Inspection type <RequiredMark /><select id="planning-intake-existing-inspectionType" value={values.inspectionType} onChange={(event) => void updateScopeSelection({ organizationId: values.organizationId, providerScopeId: values.providerScopeId, regulatedTargetId: values.regulatedTargetId, inspectionType: event.target.value })}>{inspectionTypeOptions.map((type) => <option key={type} value={type}>{catalogValueLabel(type)}</option>)}</select></label>
            <p className="planning-intake-boundary-note" role="note">Scope changes recalculate derived location and workload facts. Your authored purpose, schedule, and budget are retained.</p>
          </div> : null}

          {step === 2 && values ? <div className="planning-intake-fields planning-intake-purpose-fields">
            <label htmlFor="planning-intake-purposePreset">Start from a purpose <select id="planning-intake-purposePreset" aria-label="Purpose preset" value={values.purposePresetId ?? ""} onChange={(event) => { const preset = purposePresets.find((candidate) => candidate.id === event.target.value); if (!preset) { updateForm("purposePresetId", undefined); return; } if (values.purpose.trim() && values.purpose.trim() !== preset.purpose.trim() && !globalThis.confirm("Replace the current purpose text with this preset?")) return; updateForm("purposePresetId", preset.id); updateForm("purpose", preset.purpose); }}><option value="">Custom purpose</option>{purposePresets.map((preset) => <option key={preset.id} value={preset.id}>{preset.label}</option>)}</select><small>Presets are maintained by the server and remain editable after selection.</small></label>
            <label className="is-wide" htmlFor="planning-intake-purpose">Purpose <RequiredMark /><textarea id="planning-intake-purpose" aria-label="Purpose" aria-invalid={Boolean(fieldErrors.purpose)} aria-describedby={fieldErrors.purpose ? "planning-intake-purpose-error" : undefined} value={values.purpose} onBlur={() => setFieldErrors((current) => ({ ...current, purpose: errorsForStep(2, values).purpose }))} onChange={(event) => updateForm("purpose", event.target.value)} /><small>Describe the operational reason and what the Department Manager needs to establish. This text remains internal to the authorized Planning workflow.</small><FieldError id="planning-intake-purpose-error" message={fieldErrors.purpose} /></label>
            <p className="planning-intake-boundary-note" role="note">The initiator is recorded by the server as <b>Department Manager</b>.</p>
          </div> : null}

          {step === 3 && values ? <div className="planning-intake-fields planning-intake-schedule-fields">
            <label htmlFor="planning-intake-plannedDate">Planned date <RequiredMark /><input id="planning-intake-plannedDate" aria-label="Planned date" aria-invalid={Boolean(fieldErrors.plannedDate)} aria-describedby={fieldErrors.plannedDate ? "planning-intake-plannedDate-error" : undefined} type="date" value={values.plannedDate} onBlur={() => setFieldErrors((current) => ({ ...current, plannedDate: errorsForStep(3, candidateForStep(3)).plannedDate }))} onChange={(event) => updateForm("plannedDate", event.target.value)} /><small>Use the local date for the planned inspection activity.</small><FieldError id="planning-intake-plannedDate-error" message={fieldErrors.plannedDate} /></label>
            <fieldset className="planning-intake-mode-group"><legend>Mode <RequiredMark /></legend><div className="planning-intake-radio-row"><label><input checked={values.mode === "On-site"} name="planning-intake-mode" onChange={() => updateForm("mode", "On-site")} type="radio" />On-site</label><label><input checked={values.mode === "Remote"} name="planning-intake-mode" onChange={() => updateForm("mode", "Remote")} type="radio" />Remote</label></div><small>{values.mode === "On-site" ? "Location is required for an on-site Audit." : "Location is hidden while Remote is selected."}</small></fieldset>
            {values.mode === "On-site" ? <div className="planning-intake-location-field"><span className="planning-intake-label">Location <RequiredMark /></span>{currentLocation ? <div className="planning-intake-location-display"><div><strong>{currentLocation.label}</strong><small>{currentLocation.source === "TARGET_DEFAULT" ? "Target-derived canonical location" : currentLocation.source === "MANUAL" ? "Manual location for Finance review" : "Previously used canonical location"}</small></div><button type="button" className="planning-intake-text-action" onClick={() => setLocationEditing((current) => !current)}>{locationEditing ? "Close" : "Edit"}</button></div> : <button className="planning-intake-secondary planning-intake-add-location" type="button" onClick={() => setLocationEditing(true)}>Add location</button>}{locationEditing ? <div className="planning-intake-location-editor"><label htmlFor="planning-intake-canonicalLocation">Use a saved location<select id="planning-intake-canonicalLocation" aria-label="Saved location" value={values.locationInput?.kind === "CANONICAL" ? values.locationInput.locationId : "NEW"} onChange={(event) => { const selected = locations.find((location) => location.id === event.target.value); if (selected) { updateForm("locationInput", { kind: "CANONICAL", locationId: selected.id }); setLocationEditing(false); } else setManualLocation(""); }}><option value="NEW">Enter another location</option>{locations.map((location) => <option key={location.id} value={location.id}>{location.label}</option>)}</select></label><label htmlFor="planning-intake-location">Enter another location<input id="planning-intake-location" aria-label="Enter another location" aria-invalid={Boolean(fieldErrors.location)} aria-describedby={fieldErrors.location ? "planning-intake-location-error" : undefined} value={manualLocation} onChange={(event) => setManualLocation(event.target.value)} onBlur={() => { if (manualLocation.trim()) void resolveManualLocation(); }} placeholder="Location label" /><small>Likely aliases are matched to a canonical location before a new label is accepted.</small><FieldError id="planning-intake-location-error" message={fieldErrors.location} /></label><button type="button" className="planning-intake-secondary" onClick={() => void resolveManualLocation()} disabled={!manualLocation.trim()}>Use this location</button></div> : null}{fieldErrors.location ? <FieldError id="planning-intake-location-error" message={fieldErrors.location} /> : null}</div> : null}
            {values.mode === "Remote" ? <div className="planning-intake-disclosure"><button type="button" className="planning-intake-text-action" aria-expanded={meetingLinkOpen} onClick={() => setMeetingLinkOpen((current) => !current)}>{meetingLinkOpen ? "Hide online meeting link" : "Add online meeting link"}</button>{meetingLinkOpen ? <label htmlFor="planning-intake-meetingLink">Online meeting link <input id="planning-intake-meetingLink" aria-label="Online meeting link" aria-invalid={Boolean(fieldErrors.meetingLink)} aria-describedby={fieldErrors.meetingLink ? "planning-intake-meetingLink-error" : undefined} type="url" inputMode="url" value={values.meetingLink} onBlur={() => setFieldErrors((current) => ({ ...current, meetingLink: errorsForStep(3, candidateForStep(3)).meetingLink }))} onChange={(event) => updateForm("meetingLink", event.target.value)} placeholder="https://…" /><FieldError id="planning-intake-meetingLink-error" message={fieldErrors.meetingLink} /></label> : null}</div> : null}
          </div> : null}

          {step === 4 && values && estimate ? <div className="planning-intake-fields planning-intake-resources-step">
            <section className="planning-intake-open-section"><header><div><span className="planning-intake-section-kicker">Capacity</span><h3>Resources</h3></div><p>Give Finance a clear staffing assumption without treating the current roster as a hard schedule limit.</p></header><label htmlFor="planning-intake-requiredInspectorCount">Required inspectors <RequiredMark /><input id="planning-intake-requiredInspectorCount" aria-label="Required inspectors" aria-invalid={Boolean(fieldErrors.requiredInspectorCount)} aria-describedby={fieldErrors.requiredInspectorCount ? "planning-intake-requiredInspectorCount-error" : undefined} min="1" step="1" type="number" value={values.requiredInspectorCount} onBlur={() => setFieldErrors((current) => ({ ...current, requiredInspectorCount: errorsForStep(4, values).requiredInspectorCount }))} onChange={(event) => updateForm("requiredInspectorCount", event.target.value)} /><small>{estimate.eligibleRosterCount} eligible roster members were found. Requests above that count remain reviewable by Finance.</small><FieldError id="planning-intake-requiredInspectorCount-error" message={fieldErrors.requiredInspectorCount} /></label>{rosterWarning ? <p className="planning-intake-warning" role="status">This request is above the current eligible roster count. Finance will review the capacity assumption; it is not a hard scheduling error.</p> : null}</section>
            <section className="planning-intake-open-section"><header><div><span className="planning-intake-section-kicker">Workload</span><h3>Estimated checklist items</h3></div><p>{estimate.basisLabel}</p></header><label htmlFor="planning-intake-estimatedChecklistItemCount">Estimated checklist items <RequiredMark /><input id="planning-intake-estimatedChecklistItemCount" aria-label="Estimated checklist items" aria-invalid={Boolean(fieldErrors.estimatedChecklistItemCount)} aria-describedby={fieldErrors.estimatedChecklistItemCount ? "planning-intake-estimatedChecklistItemCount-error" : undefined} min="1" step="1" type="number" value={values.estimatedChecklistItemCount} onBlur={() => setFieldErrors((current) => ({ ...current, estimatedChecklistItemCount: errorsForStep(4, values).estimatedChecklistItemCount }))} onChange={(event) => updateForm("estimatedChecklistItemCount", event.target.value)} /><small>Suggested {estimate.suggestedCount}; safe range {estimate.safeMinimum}–{estimate.safeMaximum}; {estimate.applicableItemCount} applicable items in the governed catalog.</small><FieldError id="planning-intake-estimatedChecklistItemCount-error" message={fieldErrors.estimatedChecklistItemCount} /></label>{workloadWarning ? <p className="planning-intake-warning" role="status">This estimate is outside the server-suggested safe range. It will remain unchanged and Finance will review the entered value.</p> : null}<button className="planning-intake-secondary" type="button" onClick={(event) => { previewTriggerRef.current = event.currentTarget; setPreviewOpen(true); }}>Browse checklist items</button></section>
            <section className="planning-intake-open-section planning-intake-budget-section"><header><div><span className="planning-intake-section-kicker">Approval request</span><h3>Budget</h3></div><p>Blank budget is invalid. A literal zero still enters Finance Review.</p></header><div className="planning-intake-budget-fields"><label htmlFor="planning-intake-requestedBudget">Requested budget <RequiredMark /><input id="planning-intake-requestedBudget" aria-label="Requested budget" aria-invalid={Boolean(fieldErrors.requestedBudget)} aria-describedby={fieldErrors.requestedBudget ? "planning-intake-requestedBudget-error" : undefined} min="0" inputMode="decimal" type="number" value={values.requestedBudget} onBlur={() => setFieldErrors((current) => ({ ...current, requestedBudget: errorsForStep(4, values).requestedBudget }))} onChange={(event) => updateForm("requestedBudget", event.target.value)} /><FieldError id="planning-intake-requestedBudget-error" message={fieldErrors.requestedBudget} /></label><label htmlFor="planning-intake-currency">Currency <select id="planning-intake-currency" value={values.currency} onChange={(event) => updateForm("currency", event.target.value as NewAuditFormValues["currency"])}><option value="USD">USD</option><option value="EUR">EUR</option><option value="NAD">NAD</option></select></label></div></section>
          </div> : null}

          {step === 5 && values && draft ? <div className="planning-intake-review">
            <section className="planning-intake-review-section"><header><h3>Scope</h3><button type="button" onClick={() => editStep(1)}>Edit</button></header><dl><div><dt>Inspected Organization</dt><dd>{draft.organizationName}</dd></div><div><dt>Provider scope</dt><dd>{draft.providerScopeLabel}</dd></div><div><dt>Regulated target</dt><dd>{draft.regulatedTargetLabel}</dd></div><div><dt>Inspection type</dt><dd>{catalogValueLabel(values.inspectionType)}</dd></div></dl></section>
            <section className="planning-intake-review-section"><header><h3>Purpose</h3><button type="button" onClick={() => editStep(2)}>Edit</button></header><p className="planning-intake-review-copy">{values.purpose}</p></section>
            <section className="planning-intake-review-section"><header><h3>Schedule</h3><button type="button" onClick={() => editStep(3)}>Edit</button></header><dl><div><dt>Planned date</dt><dd>{readableDate(values.plannedDate)}</dd></div><div><dt>Mode</dt><dd>{values.mode}</dd></div>{values.mode === "On-site" ? <div><dt>Location</dt><dd>{currentLocation?.label ?? "Not set"}</dd></div> : <div><dt>Online meeting</dt><dd>{values.meetingLink || "Not added"}</dd></div>}</dl></section>
            <section className="planning-intake-review-section"><header><h3>Resources and budget</h3><button type="button" onClick={() => editStep(4)}>Edit</button></header><dl><div><dt>Required inspectors</dt><dd>{values.requiredInspectorCount}</dd></div><div><dt>Estimated checklist items</dt><dd>{values.estimatedChecklistItemCount}</dd></div><div><dt>Server suggestion</dt><dd>{estimate?.suggestedCount} items · safe range {estimate?.safeMinimum}–{estimate?.safeMaximum}</dd></div><div><dt>Requested budget</dt><dd>{values.requestedBudget} {values.currency}</dd></div></dl></section>
            <section className="planning-intake-review-section"><header><h3>Approval context</h3><button type="button" onClick={() => editStep(2)}>Edit</button></header><p>Initiated by {draft.initiatedBy}. {noticeLabel(draft.noticePolicy)}.</p><p>Submit creates a Planning item for Finance Review. It does not create an executable Audit, final checklist, or Inspector assignment.</p><p className="planning-intake-governance-path">Department Manager → Finance Review → General Manager → Executive Director → General Manager Release</p></section>
          </div> : null}
        </section>
        <AuditPlanSummary draft={draft} values={values} option={selectedOption} estimate={estimate} autosaveState={autosaveState} autosaveError={autosaveError} onRetry={() => void retryAutosave()} />
      </div>
      <section aria-label="New Audit actions" className="planning-intake-actions"><div className="planning-intake-actions__secondary">{step === 1 ? <button className="planning-intake-secondary" type="button" onClick={cancel}>Cancel</button> : <button className="planning-intake-secondary" type="button" disabled={busy || !values} onClick={() => void moveBack()}>Back</button>}{autosaveState === "error" && draft ? <button className="planning-intake-text-action" type="button" onClick={() => void retryAutosave()}>Retry save</button> : null}</div><div className="planning-intake-actions__primary">{step < 5 ? <button className="planning-intake-primary" type="button" disabled={busy || (step > 1 && !values) || (step === 1 && !selectedOption)} onClick={() => void continueFromStep()}>{actionLabel}</button> : <button className="planning-intake-primary" type="button" disabled={busy || !values || !draft} onClick={() => void submit()}>Submit to Finance</button>}</div></section>
      <WorkloadPreview open={previewOpen} onClose={() => setPreviewOpen(false)} rows={previewRows} busy={previewBusy} query={previewQuery} onQuery={setPreviewQuery} total={previewTotal} returnFocusRef={previewTriggerRef} onUseCount={() => { if (values) updateForm("estimatedChecklistItemCount", String(previewTotal)); setStatus(`Estimated checklist items set to ${previewTotal.toLocaleString("en-US")}.`); setPreviewOpen(false); }} />
    </div>
  </WorkspaceShell>;
}

export { NewAuditWizardPage };
