import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { Link } from "react-router-dom";

import type { Role, ChecklistAnswer } from "../../backend/backend";
import type {
  AGADemoWorkspaceBackend,
  AGADemoWorkspaceCapability,
  AGADemoWorkspaceCommand,
  AGADemoWorkspaceLifecycleCAAProjection,
  AGADemoWorkspaceLifecycleAuditeeProjection,
  AGADemoWorkspaceLifecycleProjection,
  AGADemoWorkspaceQuestionTextPage,
  AGADemoWorkspaceQueryResponse,
} from "../../backend/aga-demo-workspace";
import { BackendHttpError } from "../../backend/http-backend";
import { useApplicationRuntime } from "../../app/providers";
import { CommandError, PageHeader, StatusPill, WorkspaceShell, errorMessage } from "../shared/workspace-shell";

export type AGADemoLifecycleProjection =
  | AGADemoWorkspaceLifecycleProjection
  | AGADemoWorkspaceLifecycleCAAProjection
  | AGADemoWorkspaceLifecycleAuditeeProjection;

export interface AGADemoLifecyclePageProps {
  capability: AGADemoWorkspaceCapability;
  role: Role;
  roleLabel: string;
  /** Test and parent-route injection only; production routes never invent this object. */
  initialProjection?: AGADemoLifecycleProjection;
  /** A server-returned identifier held in transient React memory, never URL/query state. */
  inspectionId?: string;
  onProjectionChange?: (projection: AGADemoLifecycleProjection | null) => void;
}

export interface LifecycleWorkspaceState {
  client: AGADemoWorkspaceBackend | undefined;
  projection: AGADemoLifecycleProjection | null;
  questionPage: AGADemoWorkspaceQuestionTextPage | null;
  questionPageLoading: boolean;
  loading: boolean;
  pending: boolean;
  status: string | null;
  error: string | null;
  runCommand: (
    operationId: AGADemoWorkspaceCommand["operationId"],
    extra?: Partial<AGADemoWorkspaceCommand>,
  ) => Promise<AGADemoLifecycleProjection | null>;
  loadQuestionPage: (page: number) => Promise<void>;
}

const NO_LIFECYCLE_CONTEXT =
  "No server-returned synthetic inspection is available in this session. The page will not invent a lifecycle identifier; use an authorized server setup/release response first.";

function isLifecycleNotFound(error: unknown): boolean {
  return error instanceof BackendHttpError && error.status === 404;
}

export function useLifecycleWorkspace({
  capability,
  role,
  initialProjection,
  inspectionId,
  onProjectionChange,
}: Pick<AGADemoLifecyclePageProps, "capability" | "role" | "initialProjection" | "inspectionId" | "onProjectionChange">): LifecycleWorkspaceState {
  const runtime = useApplicationRuntime();
  const client = runtime.backend.agaDemoWorkspace;
  const [projection, setProjection] = useState<AGADemoLifecycleProjection | null>(initialProjection ?? null);
  const [questionPage, setQuestionPage] = useState<AGADemoWorkspaceQuestionTextPage | null>(null);
  const [loading, setLoading] = useState(Boolean(inspectionId && !initialProjection));
  const [questionPageLoading, setQuestionPageLoading] = useState(false);
  const [pending, setPending] = useState(false);
  const [status, setStatus] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const subjectRef = useRef(runtime.subjectId);

  const updateProjection = useCallback((next: AGADemoLifecycleProjection | null) => {
    setProjection(next);
    onProjectionChange?.(next);
  }, [onProjectionChange]);

  useEffect(() => {
    if (initialProjection) updateProjection(initialProjection);
  }, [initialProjection, updateProjection]);

  useEffect(() => {
    if (subjectRef.current === runtime.subjectId) return;
    subjectRef.current = runtime.subjectId;
    updateProjection(null);
    setQuestionPage(null);
    setStatus(null);
    setError("The authenticated principal changed; the previous lifecycle projection was cleared.");
  }, [runtime.subjectId, updateProjection]);

  useEffect(() => {
    const invalidateRestoredPage = (event: PageTransitionEvent) => {
      if (!event.persisted) return;
      updateProjection(null);
      setStatus(null);
      setError("This restored page was cleared; reload the server-authoritative lifecycle object to continue.");
    };
    window.addEventListener("pageshow", invalidateRestoredPage);
    return () => window.removeEventListener("pageshow", invalidateRestoredPage);
  }, [updateProjection]);

  useEffect(() => {
    const controller = new AbortController();
    if (initialProjection || !client || !capability.lifecycleEnabled) {
      setLoading(false);
      return () => controller.abort();
    }
    const operationId = inspectionId
      ? role === "manager" || role === "leadInspector" || role === "admin" ? "GET_ROLE_HISTORY" : "GET_INSPECTION"
      : "GET_CURRENT_INSPECTION";
    const request: Parameters<AGADemoWorkspaceBackend["lifecycleQuery"]>[0] = {
      operationId,
      ...(inspectionId ? { inspectionId } : {}),
    };
    setLoading(Boolean(inspectionId));
    setQuestionPage(null);
    void Promise.resolve(client.lifecycleQuery(request, { signal: controller.signal }))
      .then((response: AGADemoWorkspaceQueryResponse) => {
        if (controller.signal.aborted) return;
        if (!response) {
          updateProjection(null);
          setLoading(false);
          return;
        }
      const next = response.lifecycleCaa ?? response.lifecycleAuditee ?? response.lifecycle ?? null;
        updateProjection(next);
        setLoading(false);
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return;
        setLoading(false);
        if (isLifecycleNotFound(cause)) {
          updateProjection(null);
          setQuestionPage(null);
          setStatus(null);
          setError(null);
          return;
        }
        setError(errorMessage(cause));
      });
    return () => controller.abort();
  }, [capability.lifecycleEnabled, client, initialProjection, inspectionId, role, updateProjection]);

  const loadQuestionPage = useCallback(async (page: number) => {
    if (!client || !projection || initialProjection || role === "auditee" || !capability.lifecycleEnabled) return;
    setQuestionPageLoading(true);
    try {
      const response = await client.lifecycleQuery({ operationId: "GET_INSPECTION_QUESTION_PAGE", inspectionId: projection.inspectionId, page, pageSize: 25 });
      setQuestionPage(response.questionPage ?? null);
    } catch (cause) {
      if (isLifecycleNotFound(cause)) {
        updateProjection(null);
        setQuestionPage(null);
        setStatus(null);
        setError(null);
        return;
      }
      setError(errorMessage(cause));
      throw cause;
    } finally {
      setQuestionPageLoading(false);
    }
  }, [capability.lifecycleEnabled, client, initialProjection, projection, role, updateProjection]);

  useEffect(() => {
    if (!projection || initialProjection || role === "auditee") return;
    void loadQuestionPage(0).catch(() => undefined);
  }, [initialProjection, loadQuestionPage, projection, role]);

  const runCommand = useCallback(async (
    operationId: AGADemoWorkspaceCommand["operationId"],
    extra: Partial<AGADemoWorkspaceCommand> = {},
  ): Promise<AGADemoLifecycleProjection | null> => {
    if (!client) throw new Error("The HTTP workspace extension is unavailable in this build profile.");
    if (!capability.lifecycleEnabled) throw new Error("The server did not declare synthetic lifecycle capability for this principal.");
    if (!projection) throw new Error(NO_LIFECYCLE_CONTEXT);
    const command: AGADemoWorkspaceCommand = {
      operationId,
      idempotencyKey: `AGA-DEMO-UI-${operationId}-${projection.inspectionId}-R${projection.revision}`,
      expectedGenerationId: projection.generationId,
      expectedLifecycleRevision: projection.revision,
      expectedLifecycleDigest: projection.digest,
      inspectionId: projection.inspectionId,
      ...extra,
    };
    setPending(true);
    setStatus(null);
    setError(null);
    try {
      const response = await client.lifecycleCommand(command);
      const next = response.lifecycleAuditee ?? response.lifecycle ?? null;
      if (next) updateProjection(next);
      setStatus(`${operationId} recorded${response.replayed ? " (idempotent replay)" : ""}.`);
      return next;
    } catch (cause) {
      if (isLifecycleNotFound(cause)) {
        updateProjection(null);
        setQuestionPage(null);
        setStatus(null);
        setError("The server-returned lifecycle object is no longer available. Reload the authorized inspection before continuing.");
        throw cause;
      }
      setError(errorMessage(cause));
      throw cause;
    } finally {
      setPending(false);
    }
  }, [capability.lifecycleEnabled, client, projection, updateProjection]);

  return { client, projection, questionPage, questionPageLoading, loading, pending, status, error, runCommand, loadQuestionPage };
}

export function lifecycleDisabledReason(
  capability: AGADemoWorkspaceCapability,
  client: AGADemoWorkspaceBackend | undefined,
  projection: AGADemoLifecycleProjection | null,
  allowedRole: boolean,
  detail: string,
): string | null {
  if (!client) return "The HTTP workspace extension is unavailable in this build profile.";
  if (!capability.available || !capability.lifecycleEnabled) return "The server did not declare lifecycle capability for this principal.";
  if (!projection) return NO_LIFECYCLE_CONTEXT;
  if (!allowedRole) return "This operation is disabled because the authenticated role is not its assigned lifecycle actor.";
  return detail || null;
}

export function LifecycleAction({
  actionId,
  label,
  disabled,
  reason,
  onClick,
}: {
  actionId: string;
  label: string;
  disabled: boolean;
  reason: string;
  onClick?: () => void;
}) {
  const reasonId = `${actionId}-disabled-reason`;
  return (
    <>
      <button aria-describedby={disabled ? reasonId : undefined} aria-label={label} disabled={disabled} onClick={onClick} title={disabled ? reason : undefined} type="button">
        {label}
      </button>
      {disabled ? <span className="sr-only" id={reasonId}>{reason}</span> : null}
    </>
  );
}

export function LifecyclePageFrame({
  roleLabel,
  testId,
  eyebrow,
  title,
  description,
  error,
  status,
  children,
}: {
  roleLabel: string;
  testId: string;
  eyebrow: string;
  title: string;
  description: string;
  error: string | null;
  status: string | null;
  children: ReactNode;
}) {
  return (
    <WorkspaceShell roleLabel={roleLabel} routeLabel="AGA Demo Workspace">
      <main className="aga-lifecycle-page" data-testid={testId}>
        <PageHeader eyebrow={eyebrow} title={title} description={description} />
        <CommandError message={error} />
        {status ? <p className="aga-workspace-status" role="status">{status}</p> : null}
        {children}
      </main>
    </WorkspaceShell>
  );
}

export function LifecycleUnavailable({
  capability,
  client,
  projection,
  loading,
  role,
}: {
  capability: AGADemoWorkspaceCapability;
  client: AGADemoWorkspaceBackend | undefined;
  projection: AGADemoLifecycleProjection | null;
  loading: boolean;
  role: Role;
}) {
  const reason = loading
    ? "Loading the server-returned lifecycle projection."
    : lifecycleDisabledReason(capability, client, projection, true, "") ?? NO_LIFECYCLE_CONTEXT;
  return (
    <section aria-label="Synthetic lifecycle context" className="aga-lifecycle-unavailable" role={loading ? undefined : "status"}>
      <h2>Server-bound lifecycle context</h2>
      <p>{reason}</p>
      {!loading && role === "manager" ? (
        <div className="aga-lifecycle-next-step">
          <div>
            <span className="aga-lifecycle-next-step__eyebrow">Manager next step</span>
            <strong>Build and release the synthetic inspection first</strong>
            <p>Classify the sealed questions, create the package recommendation, then hand the released snapshot to the Inspector and Lead Inspector.</p>
          </div>
          <Link className="aga-lifecycle-next-step__link" to="/department-manager/aga-demo-workspace/inspection-package">Open package builder</Link>
        </div>
      ) : null}
      <LifecycleAction actionId="load-authorized-inspection" disabled label="Load authorized inspection" reason={reason} />
    </section>
  );
}

function ManagerSimulationSetup({
  capability,
  client,
  projection,
}: {
  capability: AGADemoWorkspaceCapability;
  client: AGADemoWorkspaceBackend | undefined;
  projection: AGADemoLifecycleProjection | null;
}) {
  const recommendationReason = !capability.recommendationEnabled
    ? "The server did not declare recommendation capability for this principal."
    : "Creation requires one exact server-returned provider scope, typed target, current Draft, taxonomy/run, and readiness pin; the browser never invents these facts.";
  const releaseReason = projection
    ? "This inspection is already represented by a server-returned lifecycle projection; no second simulation release is created in the browser."
    : "Release requires an immutable server recommendation snapshot and assigned binding pins; no lifecycle identifier is created in the browser.";
  return (
    <section aria-label="Department Manager simulation setup" className="aga-lifecycle-panel">
      <h2>Department Manager simulation setup</h2>
      <p>Recommendation creation and simulation release remain separate from technical approval and publication.</p>
      <div className="aga-lifecycle-actions">
        {projection ? <Link className="aga-lifecycle-primary-link" to="/department-manager/aga-demo-workspace/inspection-package">Open package builder</Link> : null}
        <LifecycleAction actionId="create-aga-recommendation" disabled label="Create AGA recommendation" reason={recommendationReason} />
        <LifecycleAction actionId="release-synthetic-simulation" disabled label="Release synthetic simulation" reason={releaseReason} />
      </div>
      <p className="aga-lifecycle-boundary">These two buttons stay disabled on the lifecycle page by design. The package builder is the single place where the server-owned Draft, recommendation, and release pins are prepared.</p>
      {!client ? <span className="sr-only">HTTP workspace extension unavailable.</span> : null}
    </section>
  );
}

function latestResponseForQuestion(
  projection: Exclude<AGADemoLifecycleProjection, AGADemoWorkspaceLifecycleAuditeeProjection>,
  questionKey: string,
): ChecklistAnswer {
  const response = projection.responses
    .filter((candidate) => candidate.questionKey === questionKey)
    .sort((left, right) => right.revision - left.revision)[0];
  return response?.answer ?? "NOT_CHECKED";
}

function isCAAProjection(
  projection: AGADemoLifecycleProjection,
): projection is AGADemoWorkspaceLifecycleCAAProjection {
  return "roleHistory" in projection;
}

function isAuditeeProjection(
  projection: AGADemoLifecycleProjection,
): projection is AGADemoWorkspaceLifecycleAuditeeProjection {
  return "publicOwnerLabel" in projection;
}

const checklistAnswers: readonly ChecklistAnswer[] = ["COMPLIANT", "NON_COMPLIANT", "OBSERVATION", "NOT_APPLICABLE", "NOT_CHECKED"];

export function AGADemoInspectionPage({
  capability,
  role,
  roleLabel,
  initialProjection,
  inspectionId,
  onProjectionChange,
}: AGADemoLifecyclePageProps) {
  const workspace = useLifecycleWorkspace({ capability, role, initialProjection, inspectionId, onProjectionChange });
  const [answers, setAnswers] = useState<Record<string, ChecklistAnswer>>({});
  const [comment, setComment] = useState("");
  const [reopenExplanation, setReopenExplanation] = useState("");
  const [responsePending, setResponsePending] = useState<string | null>(null);

  useEffect(() => {
    const inspectionProjection = workspace.projection;
    if (!inspectionProjection || isAuditeeProjection(inspectionProjection)) return;
    setAnswers(Object.fromEntries(inspectionProjection.questions.map((question) => [question.questionKey, latestResponseForQuestion(inspectionProjection, question.questionKey)])));
  }, [workspace.projection]);

  const recordResponse = async (questionKey: string) => {
    setResponsePending(questionKey);
    try {
      await workspace.runCommand("RECORD_RESPONSE", {
        targetQuestionKey: questionKey,
        answer: answers[questionKey] ?? "NOT_CHECKED",
        commentToAuditee: comment.trim(),
      });
    } catch {
      // The command error is rendered by the shared lifecycle state.
    } finally {
      setResponsePending(null);
    }
  };

  const proposeFinding = async (questionKey: string) => {
    setResponsePending(questionKey);
    try {
      await workspace.runCommand("CREATE_POTENTIAL_FINDING", {
        targetQuestionKey: questionKey,
        answer: answers[questionKey] ?? "NOT_CHECKED",
        commentToAuditee: comment.trim(),
      });
    } catch {
      // The command error is rendered by the shared lifecycle state.
    } finally {
      setResponsePending(null);
    }
  };

  const projection = workspace.projection;
  if (projection && isAuditeeProjection(projection)) {
    return (
      <LifecyclePageFrame
        description="The Auditee receives only the organization-scoped CAP and Evidence projection; inspection questions, responses, and Potential Findings remain CAA-only."
        error={workspace.error}
        eyebrow="Synthetic CAP and Evidence lifecycle"
        roleLabel={roleLabel}
        status={workspace.status}
        testId="aga-demo-inspection-page"
        title="Auditee lifecycle boundary"
      >
        <section aria-label="Auditee public owner" className="aga-lifecycle-boundary">
          <strong>{projection.publicOwnerLabel}</strong>
          <p>Use the CAP and Evidence route for the released organization-scoped lifecycle context.</p>
        </section>
      </LifecyclePageFrame>
    );
  }
  const questionTextByKey = new Map((workspace.questionPage?.items ?? []).map((question) => [question.questionKey, question.questionText]));
  const visibleQuestions = projection && workspace.questionPage
    ? projection.questions.filter((question) => questionTextByKey.has(question.questionKey))
    : initialProjection
      ? projection?.questions.slice(0, 25) ?? []
      : [];
  return (
    <LifecyclePageFrame
      description="Answer the server-pinned checklist and propose a Potential Finding without creating a direct Finding."
      error={workspace.error}
      eyebrow="Synthetic inspection lifecycle"
      roleLabel={roleLabel}
      status={workspace.status}
      testId="aga-demo-inspection-page"
      title="Inspection and checklist responses"
    >
      {!projection ? <>
        {role === "manager" ? <ManagerSimulationSetup capability={capability} client={workspace.client} projection={projection} /> : null}
        <LifecycleUnavailable capability={capability} client={workspace.client} projection={projection} loading={workspace.loading} role={role} />
        <section aria-label="Inspection actions" className="aga-lifecycle-actions">
          <LifecycleAction
            actionId="start-inspection"
            disabled
            label="Start inspection"
            reason={lifecycleDisabledReason(capability, workspace.client, projection, role === "inspector", "A server-returned inspection projection is required before starting.") ?? "A server-returned inspection projection is required before starting."}
          />
        </section>
      </> : (
        <>
          {role === "manager" ? <ManagerSimulationSetup capability={capability} client={workspace.client} projection={projection} /> : null}
          <section aria-label="Inspection identity" className="aga-lifecycle-facts">
            <article><span>Inspection state</span><strong>{projection.state}</strong><small>{projection.nextAction}</small></article>
            <article><span>Current owner</span><strong>{projection.currentOwnerRole}</strong><small>revision {projection.revision}</small></article>
            <article><span>Questions</span><strong>{projection.questions.length}</strong><small>{projection.responses.length} response versions</small></article>
            <article><span>Potential Findings</span><strong>{projection.potentialFindings.length}</strong><small>Lead review remains separate</small></article>
          </section>
          {isCAAProjection(projection) ? (
            <section aria-label="CAA role history" className="aga-lifecycle-register">
              <h2>CAA-only role history</h2>
              <p>Binding and membership history is visible only in this CAA projection.</p>
              <ul>
                {projection.roleHistory.map((event) => <li key={`${event.role}-${event.occurredAt}`}><strong>{event.role}</strong><span>{event.action} · {event.occurredAt}</span></li>)}
              </ul>
            </section>
          ) : null}
          <section aria-label="Inspection actions" className="aga-lifecycle-actions">
            <LifecycleAction
              actionId="start-inspection"
              disabled={Boolean(lifecycleDisabledReason(capability, workspace.client, projection, role === "inspector", projection.state === "READY" ? "" : "The inspection is not in READY state.")) || workspace.pending}
              label="Start inspection"
              onClick={() => void workspace.runCommand("START_INSPECTION").catch(() => undefined)}
              reason={lifecycleDisabledReason(capability, workspace.client, projection, role === "inspector", projection.state === "READY" ? "" : "The inspection is not in READY state.") ?? "The command is pending."}
            />
            <LifecycleAction
              actionId="submit-checklist"
              disabled={Boolean(lifecycleDisabledReason(capability, workspace.client, projection, role === "inspector", projection.state === "IN_PROGRESS" ? "" : "Checklist submission is available only while the Inspector is executing the inspection.")) || workspace.pending}
              label="Submit checklist"
              onClick={() => void workspace.runCommand("SUBMIT_CHECKLIST").catch(() => undefined)}
              reason={lifecycleDisabledReason(capability, workspace.client, projection, role === "inspector", projection.state === "IN_PROGRESS" ? "" : "Checklist submission is available only while the Inspector is executing the inspection.") ?? "The command is pending."}
            />
            <LifecycleAction
              actionId="reopen-checklist"
              disabled={Boolean(lifecycleDisabledReason(capability, workspace.client, projection, role === "inspector" || role === "leadInspector", projection.state === "SUBMITTED" || projection.state === "COMPLETED" ? (reopenExplanation.trim() ? "" : "A bounded reopen explanation is required.") : "Reopen is available only from SUBMITTED or COMPLETED.")) || workspace.pending}
              label="Reopen checklist"
              onClick={() => void workspace.runCommand("REOPEN_CHECKLIST", { reasonCode: "REOPEN_FOR_REVIEW", reasonExplanation: reopenExplanation.trim() }).catch(() => undefined)}
              reason={lifecycleDisabledReason(capability, workspace.client, projection, role === "inspector" || role === "leadInspector", projection.state === "SUBMITTED" || projection.state === "COMPLETED" ? (reopenExplanation.trim() ? "" : "A bounded reopen explanation is required.") : "Reopen is available only from SUBMITTED or COMPLETED.") ?? "The command is pending."}
            />
          </section>
          {(role === "inspector" || role === "leadInspector") ? <label>Reopen explanation<textarea aria-label="Reopen explanation" maxLength={1000} value={reopenExplanation} onChange={(event) => setReopenExplanation(event.target.value)} /></label> : null}
          <section aria-label="Checklist questions" className="aga-lifecycle-question-list">
            <div className="aga-lifecycle-question-list__heading"><div><h2>Server-pinned questions</h2><p>Question text is composed only for this active authorized page; the browser never stores the sealed body or invents a question identity.</p></div>{workspace.questionPage ? <div className="aga-workspace-pagination"><button disabled={workspace.questionPage.page === 0 || workspace.questionPageLoading} onClick={() => void workspace.loadQuestionPage(Math.max(0, workspace.questionPage!.page - 1))} type="button">Previous question page</button><span>Page {workspace.questionPage.page + 1}</span><button disabled={workspace.questionPage.nextPage === undefined || workspace.questionPageLoading} onClick={() => void workspace.loadQuestionPage(workspace.questionPage!.nextPage ?? workspace.questionPage!.page + 1)} type="button">Next question page</button></div> : null}</div>
            {workspace.questionPageLoading ? <p role="status">Loading the authorized transient question page…</p> : null}
            <label>Comment to Auditee<textarea aria-label="Comment to Auditee" value={comment} onChange={(event) => setComment(event.target.value)} /></label>
            {visibleQuestions.map((question) => {
              const answer = answers[question.questionKey] ?? "NOT_CHECKED";
              const eligible = answer === "NON_COMPLIANT" || answer === "OBSERVATION";
              const roleAllowed = role === "inspector";
              const stateAllowed = projection.state === "IN_PROGRESS";
              const recordReason = lifecycleDisabledReason(capability, workspace.client, projection, roleAllowed, stateAllowed ? "" : "Responses are available only while the inspection is IN_PROGRESS.") ?? "The command is pending.";
              const findingReason = lifecycleDisabledReason(capability, workspace.client, projection, roleAllowed, eligible ? (comment.trim() ? "" : "A Potential Finding requires a non-empty Comment to Auditee.") : "A Potential Finding requires NON_COMPLIANT or OBSERVATION.") ?? "The command is pending.";
              return (
                <article aria-label={`Checklist question ${question.questionKey}`} className="aga-lifecycle-question" key={question.questionKey}>
                  <div><span>Question {question.rootSequence}</span><strong>{question.questionKey}</strong><small>{question.projection.mainDomainCode ?? "Domain unavailable"} · {question.projection.applicabilityDisposition ?? "Applicability unavailable"}</small>{questionTextByKey.has(question.questionKey) ? <p className="aga-lifecycle-question-text">{questionTextByKey.get(question.questionKey)}</p> : <small>Authorized transient text is not present in this page.</small>}</div>
                  <fieldset>
                    <legend>Answer for {question.questionKey}</legend>
                    {checklistAnswers.map((candidate) => <label key={candidate}><input aria-label={`${question.questionKey} ${candidate}`} checked={answer === candidate} disabled={!roleAllowed || !stateAllowed || workspace.pending} name={`answer-${question.rootSequence}`} onChange={() => setAnswers((current) => ({ ...current, [question.questionKey]: candidate }))} type="radio" />{candidate}</label>)}
                  </fieldset>
                  <div className="aga-lifecycle-actions">
                    <LifecycleAction actionId={`record-response-${question.questionKey}`} disabled={Boolean(recordReason && recordReason !== "The command is pending.") || workspace.pending || responsePending === question.questionKey} label="Record response" onClick={() => void recordResponse(question.questionKey)} reason={recordReason} />
                    <LifecycleAction actionId={`propose-finding-${question.questionKey}`} disabled={Boolean(findingReason && findingReason !== "The command is pending.") || workspace.pending || responsePending === question.questionKey} label="Propose Potential Finding" onClick={() => void proposeFinding(question.questionKey)} reason={findingReason} />
                  </div>
                </article>
              );
            })}
          </section>
          <section aria-label="Lifecycle boundary" className="aga-lifecycle-boundary">
            <StatusPill>{projection.state}</StatusPill>
            <p>Inspector response and Potential Finding creation are synthetic, append-only lifecycle events. Lead conversion is required before a Finding exists.</p>
          </section>
        </>
      )}
    </LifecyclePageFrame>
  );
}
