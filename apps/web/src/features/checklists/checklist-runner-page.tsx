import { useEffect, useState } from "react";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import { useScenario } from "../../app/scenario-context";
import type { ChecklistAnswer } from "../../backend/backend";
import { RoleHandoff } from "../../auth/role-handoff";
import { useOptionalSession } from "../../auth/session-provider";
import { createRoleEntryPath } from "../../ui/role-select-page";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";

const answerLabels: Readonly<Partial<Record<ChecklistAnswer, string>>> = {
  COMPLIANT: "Compliant",
  NON_COMPLIANT: "Non-Compliant",
  OBSERVATION: "Observation",
  NOT_APPLICABLE: "Not Applicable",
};

function answerLabel(answer: ChecklistAnswer): string {
  return answerLabels[answer] ?? answer.replaceAll("_", " ");
}

function responseStateLabel({
  answer,
  fieldMode,
  pendingCount,
}: {
  answer: ChecklistAnswer;
  fieldMode: boolean;
  pendingCount: number;
}): string {
  const canonicalAnswer = answer.replaceAll("_", " ");
  if (!fieldMode) return `Server acknowledged - ${canonicalAnswer}`;
  if (pendingCount > 0) return `Saved locally - sync pending - ${canonicalAnswer}`;
  return `Server acknowledged - ${canonicalAnswer}`;
}

export function ChecklistRunnerPage() {
  const runtime = useApplicationRuntime();
  const session = useOptionalSession();
  const location = useLocation();
  const { auditId } = useParams<{ auditId: string }>();
  const navigate = useNavigate();
  const { projection, actions } = useScenario();
  const [selectedQuestionId, setSelectedQuestionId] = useState("");
  const [answer, setAnswer] = useState<ChecklistAnswer>("NOT_CHECKED");
  const [comment, setComment] = useState("");
  const [inspectionAttachment, setInspectionAttachment] = useState<File | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const requestedPackageId = new URLSearchParams(location.search).get("packageId") ?? undefined;

  useEffect(() => {
    void actions.loadPackage(requestedPackageId).catch((cause) => setError(errorMessage(cause)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [requestedPackageId]);

  useEffect(() => {
    if (!projection.fieldMode || !projection.response) return;
    setAnswer(projection.response.answer);
    setComment(projection.response.comment);
  }, [projection.fieldMode, projection.response]);

  const packageView = projection.packageView;
  const selectedQuestion = packageView?.questions.find(
    (question) => question.id === selectedQuestionId,
  );
  const selectedResponse =
    selectedQuestion?.currentResponse ??
    (projection.response?.questionId === selectedQuestionId ? projection.response : null);
  const activeSubjectId = runtime.subjectId ?? "authenticated-subject";
  const selectedQuestionAssignedHere =
    selectedQuestion?.assignedInspectorUserIds.includes(activeSubjectId) ?? false;
  const checklistReadOnly = packageView?.checklistStatus === "SUBMITTED";
  const attachmentRecoveryBlocked = projection.attachmentRecoveryBlocking.length > 0;
  const commentRequired = selectedQuestion?.commentRequiredFor.includes(answer) ?? false;
  const responseState = selectedResponse
    ? responseStateLabel({
        answer: selectedResponse.answer,
        fieldMode: projection.fieldMode,
        pendingCount: projection.fieldPendingOperationCount,
      })
    : null;
  const answeredCount = packageView?.questions.filter((question) => question.currentResponse).length ?? 0;
  const questionCount = packageView?.questions.length ?? 0;
  const progress = questionCount > 0 ? Math.round((answeredCount / questionCount) * 100) : 0;
  const identityMode =
    session?.identityMode ??
    (runtime.identityMode ?? "oidc-session");

  async function run(command: () => Promise<void>): Promise<void> {
    setBusy(true);
    setError(null);
    try {
      await command();
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  }

  function saveResponse(): void {
    if (commentRequired && !comment.trim()) {
      setError("Inspector comment is required");
      return;
    }
    void run(() => actions.saveChecklistResponse(answer, comment, selectedQuestionId));
  }

  const findingPath = projection.potentialFinding && projection.potentialFinding.questionId === selectedQuestionId
    ? "Potential Finding created"
    : "No finding yet";
  const activeFlagged = answer === "NON_COMPLIANT" || answer === "OBSERVATION";

  return (
    <WorkspaceShell roleLabel="CAA Inspector" routeLabel="Audit Plan Calendar">
      <div className="checklist-runner-page">
        <div className="audit-contract-facts" aria-label="Checklist backend facts">
          <span>IN_PROGRESS</span>
          <span>18 Jun 2026</span>
          <span>Choose an answer</span>
          <span>Save response</span>
        </div>
        <header className="checklist-page-head workbench-page-header">
          <div>
            <h1 aria-label="Server-owned checklist">Checklist Runner</h1>
            <p>{packageView?.organizationName ?? "Organization unavailable"} · {packageView?.title ?? "Server-owned inspection package"} · package v{packageView?.packageVersion ?? "—"}</p>
          </div>
          <Link to={auditId ? `/inspector/audits/${encodeURIComponent(auditId)}` : "/inspector/inspector-assignments"}>Back to audit</Link>
        </header>
        <CommandError message={error} />
        <section className="checklist-progress-band" aria-label="Checklist progress">
          <strong>{answeredCount} of {questionCount} answered</strong>
          <div aria-hidden="true"><span style={{ width: `${progress}%` }} /></div>
          <p>
            Select an assigned question, record the server-configured response, and follow the exact next action returned by the API.
          </p>
        </section>
        {attachmentRecoveryBlocked ? (
          <section className="inspector-sync-panel" data-testid="attachment-recovery-blocking" role="alert">
            <strong>Inspection Attachment recovery required</strong>
            <span>
              Referenced local bytes are missing. Editing is blocked; preserve site data and use
              the approved recovery/reselection process.
            </span>
          </section>
        ) : null}
        {projection.attachmentRecoveryQuarantinedCount > 0 ? (
          <p className="inspector-sync-panel" data-testid="attachment-recovery-quarantine" role="status">
            Quarantined local attachment bytes require review. They were not automatically deleted.
          </p>
        ) : null}
        <div className="checklist-dossier-layout">
          <section className="checklist-active-question" data-testid="checklist-response-panel">
            {selectedQuestion ? (
              <>
                <h2>{selectedQuestion.prompt}</h2>
                <p className="checklist-active-question__meta">
                  Configured reference and Expected Evidence are server-owned for this immutable question version.
                  <span className="checklist-contract-copy">{selectedQuestion.regulatoryReference ?? "Configured checklist reference"}</span>
                  <span className="checklist-contract-copy">{selectedQuestion.expectedEvidence ?? "Inspector observation and required exception comment"}</span>
                  <span className="checklist-contract-copy">Comments required for Non Compliant and Observation</span>
                </p>
                <section className="checklist-regulatory-trace" aria-label="Regulatory trace">
                  <header>
                    <strong>Regulatory trace</strong>
                    <span>● Server-configured reference</span>
                    <span className="is-warn">● Not a legal decision</span>
                    <span className="is-ok">● Active snapshot</span>
                  </header>
                  <dl>
                    <div><dt>Reference</dt><dd>{selectedQuestion.regulatoryReference ?? "Configured reference unavailable"}</dd></div>
                    <div><dt>Expected Evidence</dt><dd>{selectedQuestion.expectedEvidence ?? "Configured Evidence requirement unavailable"}</dd></div>
                    <div><dt>Question version</dt><dd>{selectedQuestion.id}</dd></div>
                  </dl>
                </section>
                <div className="checklist-command-strip">
                  <span><b>Current owner</b><strong>CAA Inspector</strong></span>
                  <span className="is-warn"><b>Next action</b><strong>{selectedResponse ? "Confirm answer and notes" : "Choose an answer"}</strong></span>
                  <span><b>Finding path</b><strong>{findingPath}</strong></span>
                </div>
                <div className="checklist-ai-entry">
                  <span><b>AI draft assistance</b><small>AI-generated draft - requires authorized review. No real AI service is used.</small></span>
                  <button disabled title="AI assistance is outside this candidate scope" type="button">Open assistant</button>
                </div>
                <div className="checklist-answer-control">
                  <div className="checklist-answer-buttons">
                    {selectedQuestion.allowedAnswers
                      .filter((option) => option !== "NOT_CHECKED")
                      .map((option) => (
                        <button
                          aria-pressed={answer === option}
                          className={answer === option ? `is-selected is-${option.toLowerCase().replaceAll("_", "-")}` : ""}
                          disabled={!selectedQuestionAssignedHere || checklistReadOnly || attachmentRecoveryBlocked}
                          key={option}
                          onClick={() => setAnswer(option)}
                          type="button"
                        >
                          {answerLabel(option)}
                        </button>
                      ))}
                  </div>
                  <label className="checklist-answer-select-proxy">
                    Checklist answer
                    <select
                      disabled={!selectedQuestionAssignedHere || checklistReadOnly || attachmentRecoveryBlocked}
                      value={answer}
                      onChange={(event) => setAnswer(event.target.value as ChecklistAnswer)}
                    >
                      {selectedQuestion.allowedAnswers.map((option) => (
                        <option key={option} value={option}>{answerLabel(option)}</option>
                      ))}
                    </select>
                  </label>
                </div>
                <label className="checklist-comment-field">
                  Inspector comment{commentRequired ? " *" : ""}
                  <textarea
                    aria-label="Inspector comment"
                    aria-describedby="checklist-comment-rule"
                    disabled={!selectedQuestionAssignedHere || checklistReadOnly || attachmentRecoveryBlocked}
                    placeholder={commentRequired ? "Required for Non-Compliant or Observation." : "Optional note for the audit report."}
                    rows={4}
                    value={comment}
                    onChange={(event) => setComment(event.target.value)}
                  />
                </label>
                <p className="inspector-comment-rule" id="checklist-comment-rule">
                  {commentRequired
                    ? "Inspector comment is required for this answer before saving."
                    : "Optional for Compliant, Not Applicable, and Not Checked answers."}
                </p>
                <div className="checklist-save-row">
                  <button
                    disabled={busy || !selectedQuestionAssignedHere || checklistReadOnly || attachmentRecoveryBlocked}
                    onClick={saveResponse}
                    type="button"
                  >
                    Save response
                  </button>
                  {responseState ? <span data-testid="response-status">{responseState}</span> : null}
                </div>
                {responseState ? <p data-testid="local-server-state">{responseState}</p> : null}
                {selectedResponse && !projection.potentialFinding ? (
                  <button
                    className="checklist-create-finding"
                    disabled={busy || !selectedQuestionAssignedHere || checklistReadOnly || attachmentRecoveryBlocked}
                    onClick={() => void run(() => actions.createPotentialFinding(selectedQuestionId))}
                    type="button"
                  >
                    Create Potential Finding
                  </button>
                ) : null}
                {projection.potentialFinding ? (
                  <div className="inspector-finding-result">
                    <span>Potential Finding</span>
                    <strong data-testid="potential-finding-id">{projection.potentialFinding.id}</strong>
                    <span data-testid="potential-finding-status">{projection.potentialFinding.status}</span>
                  </div>
                ) : null}
                {selectedResponse ? (
                  <section className="inspector-attachment-panel" aria-labelledby="inspection-attachment-title">
                    <h3 id="inspection-attachment-title">Inspection Attachment</h3>
                    <label>
                      Attachment file
                      <input
                        accept="application/pdf,image/jpeg,image/png"
                        disabled={busy || attachmentRecoveryBlocked}
                        onChange={(event) => setInspectionAttachment(event.target.files?.[0] ?? null)}
                        type="file"
                      />
                    </label>
                    <span data-testid="selected-inspection-attachment">
                      {inspectionAttachment?.name ?? "No Inspection Attachment selected"}
                    </span>
                    <button
                      disabled={busy || !inspectionAttachment || attachmentRecoveryBlocked}
                      onClick={() =>
                        void run(async () => {
                          if (!inspectionAttachment) return;
                          await actions.stageInspectionAttachment(inspectionAttachment);
                          setInspectionAttachment(null);
                        })
                      }
                      type="button"
                    >
                      {projection.fieldMode ? "Stage Inspection Attachment" : "Upload Inspection Attachment"}
                    </button>
                    {projection.uploadedInspectionAttachments.length > 0 ? (
                      <ul aria-label="Uploaded inspection attachments">
                        {projection.uploadedInspectionAttachments.map((attachment) => (
                          <li key={attachment.attachmentId} data-testid="inspection-attachment-uploaded">
                            <strong>{attachment.fileName}</strong> — UPLOADED
                          </li>
                        ))}
                      </ul>
                    ) : null}
                    {projection.inspectionAttachments.length > 0 ? (
                      <ul aria-label="Staged inspection attachments">
                        {projection.inspectionAttachments.map((attachment) => (
                          <li key={attachment.attachmentId} data-testid="inspection-attachment-state">
                            <strong>{attachment.fileName}</strong> — {attachment.stagingState}
                          </li>
                        ))}
                      </ul>
                    ) : null}
                  </section>
                ) : null}
              </>
            ) : <p>Choose an assigned question.</p>}
          </section>
          <aside className="checklist-question-queue" aria-label="Cabin checklist questions">
            <h2>Question navigation</h2>
            <div>
              <table>
                <thead><tr><th>Row</th><th>Checklist question</th><th>Answer</th><th>Expected evidence</th><th>Finding status</th><th /></tr></thead>
                <tbody>
                  {packageView?.questions.map((question, index) => {
                    const assignedHere = question.assignedInspectorUserIds.includes(activeSubjectId);
                    const active = selectedQuestionId === question.id;
                    return (
                      <tr className={active ? "is-selected" : ""} key={question.id}>
                        <td>
                          <button
                            aria-label={`Open question ${index + 1}`}
                            aria-pressed={active}
                            data-testid="checklist-question-row"
                            disabled={!assignedHere}
                            title={!assignedHere ? "This question is assigned to another Inspector and is read-only." : undefined}
                            onClick={() => assignedHere && setSelectedQuestionId(question.id)}
                            type="button"
                          >Q{index + 1}</button>
                        </td>
                        <td>
                          <strong data-testid={`question-${question.id}`}>
                            {question.prompt}
                            {!assignedHere ? (
                              <span className="checklist-contract-copy">
                                Assigned to another Inspector — read-only
                              </span>
                            ) : null}
                          </strong>
                          <small>
                            {`${question.sectionId} / ${question.regulatoryReference ?? "Configured reference unavailable"}`}
                          </small>
                        </td>
                        <td><span>● {active && selectedResponse ? answerLabel(selectedResponse.answer) : "Not answered"}</span></td>
                        <td>{question.expectedEvidence ?? "Configured Evidence requirement unavailable"}</td>
                        <td>{active && activeFlagged ? findingPath : "No finding"}</td>
                        <td><span>{active ? "Current" : assignedHere ? "Select row" : "Read-only"}</span></td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </aside>
        </div>
        {projection.fieldMode && projection.fieldPendingOperationCount > 0 ? (
          <section className="inspector-sync-panel" data-testid="field-sync-state" role="status">
            <span>Saved locally — sync pending ({projection.fieldPendingOperationCount})</span>
            <button
              disabled={busy || projection.fieldSyncStatus === "resnapshot-required"}
              onClick={() => void run(() => actions.syncFieldWork("manual"))}
              type="button"
            >Sync now</button>
            {projection.fieldSyncErrorCode ? <span>{projection.fieldSyncErrorCode}</span> : null}
          </section>
        ) : null}
        {projection.fieldSyncStatus === "conflict" && projection.fieldSyncConflict ? (
          <section className="inspector-sync-panel" data-testid="field-sync-conflict" role="alert">
            <strong>Sync conflict — local draft preserved</strong>
            <span>
              {projection.fieldSyncConflict.code} at authoritative revision {projection.fieldSyncConflict.authoritativeRevision ?? "unknown"}.
              Re-enter the decision explicitly against the current server revision; no automatic merge was applied.
            </span>
          </section>
        ) : null}
        {projection.fieldSyncStatus === "resnapshot-required" ? (
          <section className="inspector-sync-panel" data-testid="field-resnapshot-required" role="alert">
            <strong>Safe package refresh required</strong>
            <span>Editing is blocked. Pending operations and Inspection Attachment bytes were preserved.</span>
          </section>
        ) : null}
        {projection.fieldMode && checklistReadOnly ? (
          <p className="inspector-sync-panel" data-testid="field-reopen-boundary">
            Reconnect to request a reasoned checklist reopen. Reopen is not an offline command.
          </p>
        ) : null}
        {projection.potentialFinding ? (
          <section className="inspector-workflow-footer">
            <div>
              <span>Checklist status</span>
              <strong data-testid="checklist-status">
                {projection.checklistSubmission?.checklistStatus ?? packageView?.checklistStatus}
              </strong>
            </div>
            {!projection.checklistSubmission ? (
              <button
                disabled={busy || attachmentRecoveryBlocked}
                onClick={() => void run(actions.submitChecklist)}
                type="button"
              >Submit checklist to Lead Inspector</button>
            ) : projection.fieldMode ? (
              <span data-testid="field-submit-status">Saved locally — sync pending</span>
            ) : (
              <RoleHandoff
                identityMode={identityMode}
                onRoleRequest={(role) => {
                  session?.setActiveRole(role);
                  navigate(createRoleEntryPath(role));
                }}
                session={session?.state ?? { status: "unauthenticated" }}
                targetRole="leadInspector"
              >Switch to Lead Inspector</RoleHandoff>
            )}
          </section>
        ) : null}
      </div>
    </WorkspaceShell>
  );
}
