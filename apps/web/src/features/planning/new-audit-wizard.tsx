import { useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { z } from "zod";

import { useApplicationRuntime } from "../../app/providers";
import type {
  CanonicalQuestionCatalogPage,
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
  const canonical = [...new Set(ids)].sort().join("\n");
  const bytes = new TextEncoder().encode(canonical ? `${canonical}\n` : "");
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

  useEffect(() => {
    let cancelled = false;
    if (!backend.planningIntake) {
      setError("Planning intake commands are unavailable in this build profile.");
      return () => { cancelled = true; };
    }
    const requestedDraftId = new URLSearchParams(location.search).get("draftId");
    const operationId = `NEW-AUDIT-${requestedDraftId ?? Date.now()}`;
    const load = requestedDraftId
      ? backend.planningIntake.getDraft({ draftId: requestedDraftId })
      : backend.planningIntake.createDraft({
        operationId,
        idempotencyKey: operationId,
        expectedRevision: null,
        values: {
          organizationId: "ORG-FLY-NAMIBIA",
          organizationName: "",
          applicationType: "Continued Surveillance",
          domain: "Cabin Safety",
          inspectionCategory: "Routine / Announced",
          noticePolicy: "ADVANCE",
          purpose: "",
          triggerType: "Department Manager initiated",
          riskCategory: "",
          plannedDate: "",
          mode: "On-site",
          location: "",
          catalogVersion: "aga-preprod@1.0.0",
          scopeDraftId: "",
          selectionDigest: "",
          selectedQuestionVersionIds: [],
          requestedBudget: 0,
          currency: "USD",
        },
      });
    void load.then((loaded) => {
      if (!cancelled) {
        setDraft(loaded);
        setValues(formValuesFor(loaded));
      }
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend, location.search]);

  useEffect(() => {
    if (step !== 4 || !values || !backend.canonicalQuestionReview) return;
    const controller = new AbortController();
    setCatalogBusy(true);
    void backend.canonicalQuestionReview.listCatalog({
      catalogVersion: values.catalogVersion || "aga-preprod@1.0.0",
      usageClass: "PREPROD_EXERCISE",
      limit: 25,
    }, { signal: controller.signal }).then((page) => {
      if (!controller.signal.aborted) setCatalogPage(page);
    }).catch((cause) => {
      if (!controller.signal.aborted) setError(errorMessage(cause));
    }).finally(() => {
      if (!controller.signal.aborted) setCatalogBusy(false);
    });
    return () => controller.abort();
  }, [backend, step, values?.catalogVersion]);

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

async function toggleQuestion(questionVersionId: string) {
    if (!values) return;
    const selected = new Set(values.selectedQuestionVersionIds ?? []);
    if (selected.has(questionVersionId)) selected.delete(questionVersionId);
    else selected.add(questionVersionId);
    const ids = [...selected].sort();
    const digest = await selectionDigestFor(ids);
    setValues((current) => current ? {
      ...current,
      selectedQuestionVersionIds: ids,
      selectionDigest: digest,
      scopeDraftId: current.scopeDraftId || draft?.id || "",
    } : current);
    setStatus(`Exact question selection frozen locally · ${ids.length} selected · ${digest}`);
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
          {!values ? <p>Loading Planning intake draft…</p> : null}
          {values && step === 1 ? <div className="planning-intake-fields">
            <label>Organization<select aria-label="Organization" value={values.organizationId} onChange={(event) => update("organizationId", event.target.value)}><option value="ORG-FLY-NAMIBIA">Fly Namibia</option></select></label>
            <label>Application Type<select aria-label="Application Type" value={values.applicationType} onChange={(event) => update("applicationType", event.target.value)}><option value="Continued Surveillance">Continued Surveillance</option></select></label>
            <label>Domain<select aria-label="Domain" value={values.domain} onChange={(event) => update("domain", event.target.value)}><option value="Cabin Safety">Cabin Safety</option></select></label>
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
              <div className="planning-intake-catalog-header"><div><span className="eyebrow">Governed selection boundary</span><h3>Choose the exact question subset</h3><p>Only the selected immutable question versions are sent to Finance. No checklist template is selected here.</p></div><span className="planning-intake-catalog-count" aria-live="polite">{values.selectedQuestionVersionIds?.length ?? 0} selected</span></div>
              {catalogBusy ? <p role="status">Loading catalog page…</p> : null}
              {!catalogBusy && !catalogPage ? <p role="status">Catalog selection is unavailable in this build profile.</p> : null}
              {catalogPage ? <ul className="planning-intake-catalog-list">
                {catalogPage.items.map((question) => {
                  const checked = values.selectedQuestionVersionIds?.includes(question.questionVersionId) ?? false;
                  return <li key={question.questionVersionId}><label><input aria-label={`Select ${question.formCode} ${question.ordinal}`} checked={checked} onChange={() => void toggleQuestion(question.questionVersionId)} type="checkbox" /><span><b>{question.formCode} · item {question.ordinal}</b><small>{question.questionVersionId} · {question.proposedDomain ?? "Unclassified"}</small></span></label></li>;
                })}
              </ul> : null}
              {catalogPage?.nextCursor ? <p className="planning-intake-catalog-note">Showing the first governed page of {catalogPage.totalCount} available questions. Search and filters remain available in Checklist Management → Question Review.</p> : null}
            </div>
            <label>Question Catalog Version<input aria-label="Question Catalog Version" readOnly value={values.catalogVersion ?? "aga-preprod@1.0.0"} /></label>
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
