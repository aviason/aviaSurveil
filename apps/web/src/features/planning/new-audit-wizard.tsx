import { useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { z } from "zod";

import { useApplicationRuntime } from "../../app/providers";
import type {
  PlanningIntakeDraftValues,
  PlanningIntakeDraftView,
  PlanningIntakeInspectionCategory,
} from "../../backend/backend";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";

const DRAFT_ID = "PLAN-DRAFT-2026-001";
const SUBMITTED_PLAN_ID = "PLAN-2026-INTAKE-001";

const stepDefinitions = [
  { number: 1, title: "Inspection basics" },
  { number: 2, title: "Category and purpose" },
  { number: 3, title: "When and where" },
  { number: 4, title: "Checklist, scope and budget" },
  { number: 5, title: "Review and submit" },
] as const;

type PlanningIntakeFormValues = Omit<PlanningIntakeDraftValues, "requestedBudget"> & {
  requestedBudget: string;
};

type AgaRecommendationState = "hidden" | "checking" | "available" | "unavailable";

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
    templateVersionId: z.string().min(1, "Checklist template is required"),
    requestedBudget: requestedBudgetSchema,
  }),
} as const;

function pathForStep(step: number): string {
  return `/department-manager/new-audit/step-${step}`;
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
  return { ...draft, requestedBudget: String(draft.requestedBudget) };
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
  const [agaRecommendationState, setAgaRecommendationState] = useState<AgaRecommendationState>("hidden");

  useEffect(() => {
    const workspace = backend.agaDemoWorkspace;
    const controller = new AbortController();
    if (!workspace) {
      setAgaRecommendationState("hidden");
      return () => controller.abort();
    }
    setAgaRecommendationState("checking");
    void workspace.capability({ signal: controller.signal })
      .then((capability) => {
        if (controller.signal.aborted) return;
        setAgaRecommendationState(capability.available && capability.recommendationEnabled ? "available" : "unavailable");
      })
      .catch(() => {
        if (!controller.signal.aborted) setAgaRecommendationState("unavailable");
      });
    return () => controller.abort();
  }, [backend]);

  useEffect(() => {
    let cancelled = false;
    if (!backend.planningIntake) {
      setError("Planning intake commands are unavailable in this build profile.");
      return () => { cancelled = true; };
    }
    void backend.planningIntake.getDraft({ draftId: DRAFT_ID }).then((loaded) => {
      if (!cancelled) {
        setDraft(loaded);
        setValues(formValuesFor(loaded));
      }
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend]);

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
      await saveDraft();
      navigate(pathForStep(step + direction));
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
      if (saved) setStatus(`Draft saved · revision ${saved.revision}`);
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
        planningItemId: SUBMITTED_PLAN_ID,
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
            <label>Checklist Template<select aria-label="Checklist Template" value={values.templateVersionId} onChange={(event) => update("templateVersionId", event.target.value)}><option value="CTV-CABIN-1">Cabin Inspection Checklist · version 1</option></select></label>
            <label className="is-wide">Scope<textarea aria-label="Scope" value={values.scope} onChange={(event) => update("scope", event.target.value)} /></label>
            <label>Requested Budget<input aria-label="Requested Budget" min="0" type="number" value={values.requestedBudget} onChange={(event) => update("requestedBudget", event.target.value)} /></label>
            <label>Currency<select aria-label="Currency" value={values.currency} onChange={(event) => update("currency", event.target.value as PlanningIntakeDraftValues["currency"])}><option value="USD">USD</option><option value="EUR">EUR</option><option value="NAD">NAD</option></select></label>
            <div className="planning-intake-notice" role="note"><b>Finance Review is required even when the requested budget is zero.</b><span>Zero budget does not bypass the accepted governance chain.</span></div>
          </div> : null}
          {values && step === 5 ? <div className="planning-intake-review">
            <dl><div><dt>Draft</dt><dd>{draft?.id}</dd></div><div><dt>Organization</dt><dd>{values.organizationName} · {values.organizationId}</dd></div><div><dt>Category</dt><dd>{values.inspectionCategory}</dd></div><div><dt>Purpose</dt><dd>{values.purpose || "Not provided"}</dd></div><div><dt>Planned work</dt><dd>{values.plannedDate} · {values.mode} · {values.location || "Location required"}</dd></div><div><dt>Scope</dt><dd>{values.scope || "Not provided"}</dd></div><div><dt>Requested budget</dt><dd>{values.requestedBudget} {values.currency}</dd></div><div><dt>Notice</dt><dd>{noticeLabel(values)}</dd></div></dl>
            <div className="planning-intake-governance"><b>Department Manager → Finance Review → General Manager → Executive Director → General Manager Release</b><p>No executable Audit is created at this step. The submitted record remains a Planning item awaiting Finance Review.</p></div>
            {preview ? <article className="planning-intake-preview" aria-label="Planning intake preview"><p className="eyebrow">Durable in-screen preview</p><h3>{values.inspectionCategory} — {values.organizationName}</h3><p>{values.purpose}</p><small>{draft?.id} · revision {draft?.revision}</small></article> : null}
          </div> : null}
        </section>
        {step === 5 && agaRecommendationState === "available" ? <section aria-label="AGA recommendation" className="planning-intake-governance">
          <p className="eyebrow">Synthetic AGA workspace</p>
          <h2>Deterministic checklist recommendation</h2>
          <p role="status">Recommendation is unavailable until the authorized workspace supplies one current server-derived provider scope and target.</p>
          <button disabled title="Server-derived provider scope and target facts are required" type="button">Create AGA recommendation</button>
        </section> : null}
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
