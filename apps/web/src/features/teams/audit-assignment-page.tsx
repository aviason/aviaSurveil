import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { AssignmentSummary, InspectionPackage } from "../../backend/backend";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";

const AUDIT_ID = "AUD-2026-001";
const PACKAGE_ID = "PKG-CAB-2026-001";

const inspectorNames: Readonly<Record<string, string>> = {
  "USR-INSPECTOR-AMINA": "Amina Inspector",
  "USR-INSPECTOR-DAVID": "David Inspector",
};

export function AuditAssignmentPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("leadInspector") ?? runtime.backend, [runtime]);
  const [assignment, setAssignment] = useState<AssignmentSummary | null>(null);
  const [inspectionPackage, setInspectionPackage] = useState<InspectionPackage | null>(null);
  const [showWorkload, setShowWorkload] = useState(true);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    let cancelled = false;
    void Promise.all([backend.assignments.list({}), backend.inspections.getPackage({ packageId: PACKAGE_ID })]).then(([assignments, packageView]) => {
      const exactAssignment = assignments.items.find((item) => item.auditId === AUDIT_ID);
      if (!exactAssignment || packageView.auditId !== AUDIT_ID) throw new Error(`Audit ${AUDIT_ID} assignment package is unavailable.`);
      if (!cancelled) { setAssignment(exactAssignment); setInspectionPackage(packageView); }
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend]);
  const workload = useMemo(() => {
    const counts = new Map<string, number>();
    for (const question of inspectionPackage?.questions ?? []) {
      for (const subjectId of question.assignedInspectorUserIds) counts.set(subjectId, (counts.get(subjectId) ?? 0) + 1);
    }
    return [...counts.entries()];
  }, [inspectionPackage]);
  return <WorkspaceShell roleLabel="Lead Inspector" routeLabel="Lead Review">
    <div className="lead-secondary-page lead-audit-assignment" data-audit-id={AUDIT_ID} data-testid="lead-audit-assignment-page">
      <header className="lead-secondary-header workbench-page-header"><div><Link className="lead-back-link" to="/lead-inspector/lead-review">← Back to Assigned Audits</Link><h1>Cabin Inspection</h1><p>{AUDIT_ID}</p></div><Link className="lead-button workbench-page-header__action" to="/lead-inspector/preliminary-reports/PR-2026-018">View Preliminary Report</Link></header>
      <CommandError message={error} />
      {assignment && inspectionPackage ? <>
        <section aria-label="Audit assignment summary" className="lead-fact-strip"><div><small>Organization</small><strong>{assignment.organizationName}</strong><span>{assignment.organizationId}</span></div><div><small>Inspection Type</small><strong>Cabin Inspection</strong><span>{assignment.title}</span></div><div><small>Current Owner</small><strong>Lead Inspector</strong></div><div><small>Next Action</small><strong>Assign exact checklist questions</strong></div><div><small>Due Date</small><strong>{assignment.dueDate}</strong></div></section>
        <div className="lead-assignment-progress" aria-label="Assignment progress"><span className="is-complete">Planning · Completed</span><span className="is-complete">Approval · Completed</span><span className="is-active">Assignment · In Progress</span><span>Execution · Pending</span></div>
        <div className="lead-workflow-grid">
          <section className="lead-panel"><h2>Assignment Overview</h2><dl className="lead-detail-grid"><div><dt>Checklist</dt><dd>{inspectionPackage.title}</dd></div><div><dt>Runnable Questions</dt><dd>{inspectionPackage.questions.length}</dd></div><div><dt>Package Version</dt><dd>{inspectionPackage.packageVersion}</dd></div><div><dt>Assignment Status</dt><dd>{assignment.status}</dd></div></dl><Link className="lead-button lead-button--primary" to={`/lead-inspector/audits/${assignment.auditId}/checklist-questions`}>Assign Checklist Questions</Link></section>
          <section className="lead-panel"><h2>Inspection Scope</h2><dl className="lead-detail-grid"><div><dt>Sections</dt><dd>{new Set(inspectionPackage.questions.map((question) => question.sectionId)).size}</dd></div><div><dt>Template</dt><dd>{inspectionPackage.templateVersionId}</dd></div><div><dt>Lead Inspector</dt><dd>Caner Yildiz</dd></div></dl></section>
          <section className="lead-panel"><h2>Next Steps</h2><button className="lead-button" onClick={() => setShowWorkload((value) => !value)} type="button">{showWorkload ? "Hide" : "View"} Workload Summary</button><p>Assignment changes cannot approve, issue, sign, or lock reports.</p></section>
        </div>
        {showWorkload ? <section aria-label="Inspector workload" className="lead-panel lead-workload"><h2>Inspector workload</h2>{workload.map(([subjectId, count]) => <article data-subject-id={subjectId} key={subjectId}><strong>{inspectorNames[subjectId] ?? subjectId}</strong><span>{subjectId}</span><b>{count} exact questions</b></article>)}</section> : null}
      </> : <p>Loading exact Audit assignment…</p>}
    </div>
  </WorkspaceShell>;
}
