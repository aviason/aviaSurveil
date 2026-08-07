import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { AssignmentSummary, CanonicalAssignmentView, InspectionPackage } from "../../backend/backend";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";

const inspectorNames: Readonly<Record<string, string>> = {
  "USR-INSPECTOR-AMINA": "Amina Inspector",
  "USR-INSPECTOR-DAVID": "David Inspector",
};

export function AuditAssignmentPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("leadInspector") ?? runtime.backend, [runtime]);
  const [searchParams] = useSearchParams();
  const [assignment, setAssignment] = useState<AssignmentSummary | null>(null);
  const [inspectionPackage, setInspectionPackage] = useState<InspectionPackage | null>(null);
  const [preparation, setPreparation] = useState<CanonicalAssignmentView | null>(null);
  const [memberSubjectIds, setMemberSubjectIds] = useState("");
  const [questionCoverage, setQuestionCoverage] = useState("");
  const [busy, setBusy] = useState(false);
  const [showWorkload, setShowWorkload] = useState(true);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    let cancelled = false;
    const workflow = backend.canonicalAuditWorkflow;
    if (workflow) {
      void workflow.getPreparation({ assignmentId: searchParams.get("assignmentId") ?? undefined }).then((item) => {
        if (!cancelled) {
          setPreparation(item);
          setMemberSubjectIds(item.memberSubjectIds.filter((subjectId) => subjectId !== item.leadSubjectId).join(", "));
          setQuestionCoverage(item.questionAssignments.map((row) => `${row.questionId}:${row.subjectId}`).join("\n"));
        }
      }).catch(() => {
        if (cancelled) return;
        void backend.assignments.list({}).then(({ items }) => {
          const exactAssignment = items.find((item) => item.packageId);
          if (!exactAssignment?.packageId) throw new Error("No materialized canonical Audit assignment is available.");
          return backend.inspections.getPackage({ packageId: exactAssignment.packageId }).then((packageView) => {
            if (packageView.auditId !== exactAssignment.auditId) throw new Error("The execution package is not scoped to its server-owned Audit.");
            if (!cancelled) { setAssignment(exactAssignment); setInspectionPackage(packageView); }
          });
        }).catch((cause) => !cancelled && setError(errorMessage(cause)));
      });
      return () => { cancelled = true; };
    }
    void backend.assignments.list({}).then(({ items }) => {
      const exactAssignment = items.find((item) => item.packageId);
      if (!exactAssignment?.packageId) throw new Error("No materialized canonical Audit assignment is available.");
      return backend.inspections.getPackage({ packageId: exactAssignment.packageId }).then((packageView) => {
        if (packageView.auditId !== exactAssignment.auditId) throw new Error("The execution package is not scoped to its server-owned Audit.");
        if (!cancelled) { setAssignment(exactAssignment); setInspectionPackage(packageView); }
      });
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend, searchParams]);

  async function saveTeam(): Promise<void> {
    if (!preparation || !backend.canonicalAuditWorkflow) return;
    const members = memberSubjectIds.split(/[,\n]/).map((value) => value.trim()).filter(Boolean);
    if (!members.length) { setError("Enter at least one Inspector subject ID."); return; }
    setBusy(true); setError(null);
    try {
      const next = await backend.canonicalAuditWorkflow.assignTeam(preparation.id, {
        operationId: `TEAM-${preparation.id}-${preparation.revision}`,
        idempotencyKey: `TEAM-${preparation.id}-${preparation.revision}`,
        expectedRevision: preparation.revision,
        memberSubjectIds: members,
      });
      setPreparation(next);
    } catch (cause) { setError(errorMessage(cause)); } finally { setBusy(false); }
  }

  async function saveCoverage(): Promise<void> {
    if (!preparation || !backend.canonicalAuditWorkflow) return;
    const rows = questionCoverage.split(/[,\n]/).map((value) => value.trim()).filter(Boolean).map((value) => {
      const [questionId, subjectId] = value.split(":").map((part) => part.trim());
      return { questionId, subjectId };
    }).filter((row) => row.questionId && row.subjectId);
    if (!rows.length) { setError("Enter coverage as questionVersionId:subjectId, one per line."); return; }
    setBusy(true); setError(null);
    try {
      const next = await backend.canonicalAuditWorkflow.assignQuestionCoverage(preparation.id, {
        operationId: `COVERAGE-${preparation.id}-${preparation.revision}`,
        idempotencyKey: `COVERAGE-${preparation.id}-${preparation.revision}`,
        expectedRevision: preparation.revision,
        questionAssignments: rows,
      });
      setPreparation(next);
    } catch (cause) { setError(errorMessage(cause)); } finally { setBusy(false); }
  }
  const workload = useMemo(() => {
    const counts = new Map<string, number>();
    for (const question of inspectionPackage?.questions ?? []) {
      for (const subjectId of question.assignedInspectorUserIds) counts.set(subjectId, (counts.get(subjectId) ?? 0) + 1);
    }
    return [...counts.entries()];
  }, [inspectionPackage]);
  return <WorkspaceShell roleLabel="Lead Inspector" routeLabel="Lead Review">
    <div className="lead-secondary-page lead-audit-assignment" data-audit-id={assignment?.auditId ?? inspectionPackage?.auditId ?? undefined} data-testid="lead-audit-assignment-page">
      <header className="lead-secondary-header workbench-page-header"><div><Link className="lead-back-link" to="/lead-inspector/lead-review">← Back to Assigned Audits</Link><h1>{assignment?.title ?? "Audit assignment"}</h1><p>{assignment?.auditId ?? "Server-owned Audit"}</p></div><Link className="lead-button workbench-page-header__action" to="/lead-inspector/preliminary-reports">View Preliminary Reports</Link></header>
      <CommandError message={error} />
      {preparation ? <>
        <section aria-label="Pre-materialization preparation" className="lead-panel lead-preparation-panel">
          <h2>Pre-materialization preparation</h2>
          <p>Lead Inspector authority is required before the Department Manager can confirm and materialize this Audit. The execution package is intentionally unavailable until then.</p>
          <dl className="lead-detail-grid"><div><dt>Assignment</dt><dd>{preparation.id}</dd></div><div><dt>Status</dt><dd>{preparation.status}</dd></div><div><dt>Released questions</dt><dd>{preparation.selectedQuestionVersionIds?.length ?? 0}</dd></div><div><dt>Revision</dt><dd>{preparation.revision}</dd></div></dl>
          <label>Inspector subject IDs<input aria-label="Inspector subject IDs" value={memberSubjectIds} onChange={(event) => setMemberSubjectIds(event.target.value)} placeholder="inspector-a, inspector-b" /></label>
          <button className="lead-button lead-button--primary" disabled={busy || preparation.status !== "LEAD_ASSIGNED"} onClick={() => void saveTeam()} title={preparation.status !== "LEAD_ASSIGNED" ? "Team membership is already assigned or the preparation is no longer editable." : undefined} type="button">Assign exact team</button>
          <p>Released question versions: {(preparation.selectedQuestionVersionIds ?? []).join(", ") || "none returned"}</p>
          <label>Per-question coverage<textarea aria-label="Per-question coverage" value={questionCoverage} onChange={(event) => setQuestionCoverage(event.target.value)} placeholder="questionVersionId:subjectId" /></label>
          <button className="lead-button lead-button--primary" disabled={busy || (preparation.status !== "TEAM_ASSIGNED" && preparation.status !== "QUESTIONS_ASSIGNED")} onClick={() => void saveCoverage()} title={preparation.status !== "TEAM_ASSIGNED" && preparation.status !== "QUESTIONS_ASSIGNED" ? "Assign the exact team before coverage." : undefined} type="button">Save question coverage</button>
          <p role="status">After every released question has coverage, return to the Department Manager for preparation confirmation.</p>
        </section>
      </> : assignment && inspectionPackage ? <>
        <section aria-label="Audit assignment summary" className="lead-fact-strip"><div><small>Organization</small><strong>{assignment.organizationName}</strong><span>{assignment.organizationId}</span></div><div><small>Inspection Type</small><strong>{inspectionPackage.title}</strong><span>{assignment.title}</span></div><div><small>Current Owner</small><strong>Lead Inspector</strong></div><div><small>Next Action</small><strong>Assign exact checklist questions</strong></div><div><small>Due Date</small><strong>{assignment.dueDate}</strong></div></section>
        <div className="lead-assignment-progress" aria-label="Assignment progress"><span className="is-complete">Planning · Completed</span><span className="is-complete">Approval · Completed</span><span className="is-active">Assignment · In Progress</span><span>Execution · Pending</span></div>
        <div className="lead-workflow-grid">
          <section className="lead-panel"><h2>Assignment Overview</h2><dl className="lead-detail-grid"><div><dt>Checklist</dt><dd>{inspectionPackage.title}</dd></div><div><dt>Runnable Questions</dt><dd>{inspectionPackage.questions.length}</dd></div><div><dt>Package Version</dt><dd>{inspectionPackage.packageVersion}</dd></div><div><dt>Assignment Status</dt><dd>{assignment.status}</dd></div></dl><Link className="lead-button lead-button--primary" to={`/lead-inspector/audits/${assignment.auditId}/checklist-questions`}>Assign Checklist Questions</Link></section>
          <section className="lead-panel"><h2>Inspection Scope</h2><dl className="lead-detail-grid"><div><dt>Sections</dt><dd>{new Set(inspectionPackage.questions.map((question) => question.sectionId)).size}</dd></div><div><dt>Template</dt><dd>{inspectionPackage.templateVersionId}</dd></div><div><dt>Lead Inspector</dt><dd>Current assignment</dd></div></dl></section>
          <section className="lead-panel"><h2>Next Steps</h2><button className="lead-button" onClick={() => setShowWorkload((value) => !value)} type="button">{showWorkload ? "Hide" : "View"} Workload Summary</button><p>Assignment changes cannot approve, issue, sign, or lock reports.</p></section>
        </div>
        {showWorkload ? <section aria-label="Inspector workload" className="lead-panel lead-workload"><h2>Inspector workload</h2>{workload.map(([subjectId, count]) => <article data-subject-id={subjectId} key={subjectId}><strong>{inspectorNames[subjectId] ?? subjectId}</strong><span>{subjectId}</span><b>{count} exact questions</b></article>)}</section> : null}
      </> : <p>Loading exact Audit assignment…</p>}
    </div>
  </WorkspaceShell>;
}
