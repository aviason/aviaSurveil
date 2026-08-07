import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import { CANONICAL_QUESTION_REVIEW_PATH } from "../../app/route-contracts";
import type {
  DepartmentManagerGovernedReviewCommandInput,
  DepartmentManagerGovernedReviewItem,
  GovernedBlockedGenerationResult,
  GovernedPublicationView,
  InspectionPackage,
} from "../../backend/backend";
import { EXACT_BLOCKED_REAL_OPS_AOC_REQUEST } from "../../backend/governed-synthetic-profile";
import { CommandError, errorMessage, PageHeader, WorkspaceShell } from "../shared/workspace-shell";

function activeReviewQueueItems(items: DepartmentManagerGovernedReviewItem[]): DepartmentManagerGovernedReviewItem[] {
  return items.filter((item) => ["DEPARTMENT_REVIEW", "TECHNICALLY_APPROVED"].includes(item.candidate.status));
}

type ReviewAction = "return" | "reject" | "approve";

export function ChecklistManagementPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("manager") ?? runtime.backend, [runtime]);
  const [inspectionPackage, setInspectionPackage] = useState<InspectionPackage | null>(null);
  const [reviewQueue, setReviewQueue] = useState<DepartmentManagerGovernedReviewItem[]>([]);
  const [reviewItem, setReviewItem] = useState<DepartmentManagerGovernedReviewItem | null>(null);
  const [publication, setPublication] = useState<GovernedPublicationView | null>(null);
  const [blockedGeneration, setBlockedGeneration] = useState<GovernedBlockedGenerationResult | null>(null);
  const [reason, setReason] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    void Promise.all([
      backend.inspections.getPackage({ packageId: "PKG-CAB-2026-001" }),
      backend.governedChecklistReview.listQueue({}),
      backend.governedChecklistReview.validateBlockedGeneration({
        operationId: "GOVERNED-BLOCKED-GENREQ-OPS-AOC-0001",
        idempotencyKey: "GOVERNED-BLOCKED-GENREQ-OPS-AOC-0001",
        generationRequest: EXACT_BLOCKED_REAL_OPS_AOC_REQUEST,
      }),
    ]).then(([loadedPackage, queue, blocked]) => {
      if (!cancelled) {
        setInspectionPackage(loadedPackage);
        const activeItems = activeReviewQueueItems(queue.items);
        setReviewQueue(activeItems);
        setReviewItem(activeItems[0] ?? null);
        setBlockedGeneration(blocked);
      }
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend]);

  const selectCandidate = useCallback(async (candidateId: string) => {
    setPending(true);
    setError(null);
    try {
      const detail = await backend.governedChecklistReview.getCandidate({ candidateId });
      setReviewItem(detail);
      setReason("");
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setPending(false);
    }
  }, [backend]);

  const commandFor = useCallback((action: string): DepartmentManagerGovernedReviewCommandInput | null => {
    if (!reviewItem || !reason.trim()) return null;
    const candidate = reviewItem.candidate;
    const operationId = [
      "GOVERNED", action.toLocaleUpperCase(), candidate.candidateId,
      String(candidate.revision), candidate.contentDigest.slice(-12),
    ].join("-");
    return {
      operationId,
      idempotencyKey: operationId,
      candidateId: candidate.candidateId,
      expectedRevision: candidate.revision,
      expectedContentDigest: candidate.contentDigest,
      reason: reason.trim(),
    };
  }, [reason, reviewItem]);

  const decide = useCallback(async (action: ReviewAction) => {
    const command = commandFor(action);
    if (!command) {
      setError("A decision reason is required.");
      return;
    }
    setPending(true);
    setError(null);
    try {
      const candidate = await backend.governedChecklistReview[action](command);
      let detail: DepartmentManagerGovernedReviewItem;
      try {
        detail = await backend.governedChecklistReview.getCandidate({ candidateId: candidate.candidateId });
      } catch {
        detail = { ...reviewItem!, candidate };
      }
      const queue = await backend.governedChecklistReview.listQueue({});
      setReviewQueue(activeReviewQueueItems(queue.items));
      setReviewItem(detail);
      setReason("");
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setPending(false);
    }
  }, [backend, commandFor, reviewItem]);

  const publish = useCallback(async () => {
    const command = commandFor("publish");
    if (!command) {
      setError("A publication reason is required.");
      return;
    }
    setPending(true);
    setError(null);
    try {
      const published = await backend.governedChecklistReview.publish(command);
      const detail = await backend.governedChecklistReview.getCandidate({
        candidateId: published.candidateId,
      });
      const queue = await backend.governedChecklistReview.listQueue({});
      setPublication(published);
      setReviewQueue(queue.items);
      setReviewItem(detail);
      setReason("");
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setPending(false);
    }
  }, [backend, commandFor]);

  const hasOwnApproval = reviewItem?.decisions.some((decision) =>
    decision.decision === "TECHNICALLY_APPROVED" &&
    decision.actorSubjectId === runtime.subjectId) ?? false;

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel="Checklist Management">
      <div className="manager-ops-page" data-testid="manager-checklist-management-page">
        <PageHeader
          eyebrow="Governed configuration"
          title="Checklist Management"
          description="Review exact source lineage and owner scope, record an attributed technical decision, then publish in a separate controlled step."
          action={<Link className="button" to={CANONICAL_QUESTION_REVIEW_PATH}>Question Review</Link>}
        />
        <CommandError message={error} />

        <section
          aria-label="Governed checklist review"
          className="manager-ops-layout"
          data-testid="governed-checklist-review"
        >
          {reviewQueue.length > 0 ? (
            <nav aria-label="Authorized governed checklist queue" className="button-row">
              {reviewQueue.map((item) => (
                <button
                  aria-current={reviewItem?.candidate.candidateId === item.candidate.candidateId ? "true" : undefined}
                  aria-label={`Select governed candidate ${item.candidate.candidateId}`}
                  disabled={pending}
                  key={item.candidate.candidateId}
                  onClick={() => void selectCandidate(item.candidate.candidateId)}
                  type="button"
                >
                  {item.candidate.candidateId} · {item.candidate.status}
                </button>
              ))}
            </nav>
          ) : null}
          {reviewItem ? (
            <>
              <div className="manager-ops-register">
                <article className="manager-ops-card">
                  <p className="eyebrow">Current exact candidate</p>
                  <h2>{reviewItem.candidate.candidateId}</h2>
                  <p><span className="status-pill">{reviewItem.candidate.status}</span></p>
                  <dl>
                    <div><dt>Revision</dt><dd>{reviewItem.candidate.revision}</dd></div>
                    <div><dt>Content digest</dt><dd>{reviewItem.candidate.contentDigest}</dd></div>
                    <div><dt>Generation run</dt><dd>{reviewItem.candidate.generationRunId}</dd></div>
                  </dl>
                </article>

                <article className="manager-ops-card">
                  <p className="eyebrow">Persisted source lineage</p>
                  <h3>Sources and clauses</h3>
                  {reviewItem.candidate.sourceSnapshots.map((source) => (
                    <dl key={`${source.sourceId}-${source.clauseId}`}>
                      <div><dt>Source</dt><dd>{source.sourceIdentity}</dd></div>
                      <div><dt>Clause</dt><dd>{source.locator}</dd></div>
                      <div><dt>Hash</dt><dd>{source.sourceHash}</dd></div>
                    </dl>
                  ))}
                </article>

                <article className="manager-ops-card">
                  <p className="eyebrow">Exact required owners</p>
                  <h3>Approval scope</h3>
                  {reviewItem.requiredOwners.map((owner) => (
                    <dl key={`${owner.departmentId}-${owner.organizationalUnitId}`}>
                      <div><dt>Department</dt><dd>{owner.departmentId}</dd></div>
                      <div><dt>Unit</dt><dd>{owner.organizationalUnitId}</dd></div>
                      <div><dt>Approval</dt><dd>{owner.approvalRequired ? "Required" : "Not required"}</dd></div>
                    </dl>
                  ))}
                </article>
              </div>

              <div className="manager-ops-dossier">
                <p className="eyebrow">Review controls</p>
                <h2>Decisions and blockers</h2>
                <section aria-label="Blocking issues">
                  <h3>Blocking issues</h3>
                  {reviewItem.blockingIssues.length === 0
                    ? <p>No blocking issues</p>
                    : reviewItem.blockingIssues.map((issue) => (
                      <article key={`${issue.code}-${issue.fieldPath}`}>
                        <strong>{issue.code}</strong>
                        <p>{issue.message}</p>
                        <small>{issue.sourceIdentity} · {issue.locator}</small>
                      </article>
                    ))}
                </section>
                <section aria-label="Recorded decisions">
                  <h3>Recorded decisions</h3>
                  {reviewItem.decisions.length === 0
                    ? <p>No decisions recorded</p>
                    : reviewItem.decisions.map((decision) => (
                      <article key={decision.operationId}>
                        <strong>{decision.decision}</strong>
                        <p>{decision.actorSubjectId}</p>
                        <small>{decision.actorDepartmentId} · {decision.reason}</small>
                      </article>
                    ))}
                </section>
                <label>
                  Decision reason
                  <textarea
                    disabled={pending}
                    onChange={(event) => setReason(event.target.value)}
                    rows={4}
                    value={reason}
                  />
                </label>
                {reviewItem.candidate.status === "DEPARTMENT_REVIEW" ? (
                  <div className="button-row">
                    <button disabled={pending} onClick={() => void decide("return")} type="button">Return for revision</button>
                    <button disabled={pending} onClick={() => void decide("reject")} type="button">Reject candidate</button>
                    {!hasOwnApproval ? (
                      <button disabled={pending || reviewItem.blockingIssues.length > 0} onClick={() => void decide("approve")} type="button">
                        Technically approve
                      </button>
                    ) : null}
                  </div>
                ) : null}
                {reviewItem.candidate.status === "TECHNICALLY_APPROVED" ? (
                  <button disabled={pending} onClick={() => void publish()} type="button">
                    Publish checklist version
                  </button>
                ) : null}
                {publication ? (
                  <div role="status">
                    <strong>{publication.templateVersionId}</strong>
                    <p>Published as immutable Checklist Template Version in a separate publication decision.</p>
                  </div>
                ) : null}
              </div>
            </>
          ) : (
            <>
              <article className="manager-ops-card">
                <p className="eyebrow">Department review queue</p>
                <h2>No current governed candidates</h2>
                <p>Only candidates owned by the manager’s current exact department and organizational unit appear here.</p>
              </article>
              {blockedGeneration ? (
                <article className="manager-ops-card" data-testid="blocked-governed-generation">
                  <p className="eyebrow">Source-bound generation validation</p>
                  <h3>{blockedGeneration.requestId}</h3>
                  <p><span className="status-pill">{blockedGeneration.status}</span></p>
                  <ul>
                    {blockedGeneration.blockingIssues.map((issue) => (
                      <li key={issue.gapId}>
                        <strong>{issue.gapId}</strong> · {issue.reason}
                      </li>
                    ))}
                  </ul>
                </article>
              ) : null}
            </>
          )}
        </section>

        {inspectionPackage ? (
          <section aria-label="Published checklist reference" className="manager-ops-layout">
            <div aria-label="Published Checklist questions" className="manager-ops-register">
              <article className="manager-ops-card">
                <p className="eyebrow">Published package reference · version {inspectionPackage.packageVersion}</p>
                <h2>{inspectionPackage.title}</h2>
                <p>{inspectionPackage.questions.length} configured questions</p>
              </article>
              {inspectionPackage.questions.map((question) => (
                <article className="manager-ops-card" key={question.id}>
                  <p className="eyebrow">{question.sectionId}</p>
                  <h3>{question.id}</h3>
                  <p>{question.prompt}</p>
                  <small>{question.expectedEvidence ?? "No expected Evidence configured"}</small>
                </article>
              ))}
            </div>
            <aside aria-label="Checklist configuration boundary" className="manager-ops-dossier">
              <p className="eyebrow">Published immutable reference</p>
              <h2>{inspectionPackage.templateVersionId}</h2>
              <p>Package {inspectionPackage.id} binds this published version to {inspectionPackage.auditId}.</p>
              <button
                aria-label={`Edit ${inspectionPackage.templateVersionId} unavailable`}
                disabled
                title={`Checklist Template Version ${inspectionPackage.templateVersionId} is published and no Department Manager draft-mutation command is declared in Plan 1.`}
                type="button"
              >
                Edit unavailable
              </button>
            </aside>
          </section>
        ) : null}
      </div>
    </WorkspaceShell>
  );
}
