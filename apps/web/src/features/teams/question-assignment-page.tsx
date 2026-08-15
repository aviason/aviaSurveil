import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { AssignmentSummary, InspectionPackage } from "../../backend/backend";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";

export function QuestionAssignmentPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("leadInspector") ?? runtime.backend, [runtime]);
  const [assignment, setAssignment] = useState<AssignmentSummary | null>(null);
  const [inspectionPackage, setInspectionPackage] = useState<InspectionPackage | null>(null);
  const [section, setSection] = useState("all");
  const [query, setQuery] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    void backend.assignments.list({}).then(({ items }) => {
      const current = items.find((item) => item.packageId);
      if (!current?.packageId) throw new Error("No materialized canonical Audit is available for question assignment.");
      return backend.inspections.getPackage({ packageId: current.packageId }).then((packageView) => {
        if (packageView.auditId !== current.auditId) throw new Error("The execution package is not scoped to its server-owned Audit.");
        if (!cancelled) {
          setAssignment(current);
          setInspectionPackage(packageView);
        }
      });
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend]);

  const sections = [...new Set(inspectionPackage?.questions.map((question) => question.sectionId) ?? [])];
  const visibleQuestions = (inspectionPackage?.questions ?? []).filter((question) => {
    return (section === "all" || question.sectionId === section)
      && `${question.id} ${question.prompt} ${question.regulatoryReference ?? ""}`.toLowerCase().includes(query.toLowerCase());
  });
  const workload = (inspectionPackage?.questions ?? []).reduce<Record<string, number>>((counts, question) => {
    for (const subjectId of question.assignedInspectorUserIds) counts[subjectId] = (counts[subjectId] ?? 0) + 1;
    return counts;
  }, {});

  return <WorkspaceShell roleLabel="Lead Inspector" routeLabel="Lead Review">
    <div className="lead-secondary-page lead-question-assignment" data-audit-id={assignment?.auditId ?? inspectionPackage?.auditId ?? undefined} data-testid="lead-question-assignment-page">
      <header className="lead-secondary-header workbench-page-header"><div><Link className="lead-back-link" to={assignment ? `/lead-inspector/audits/${assignment.auditId}/assignment` : "/lead-inspector/lead-review"}>← Back to Assignment Overview</Link><h1>Assign Checklist Questions</h1><p>Assign exact backend question IDs to Inspectors for this Audit.</p></div></header>
      <CommandError message={error} />
      {inspectionPackage ? <>
        <section className="lead-fact-strip"><div><small>Inspection</small><strong>{inspectionPackage.title}</strong><span>{inspectionPackage.auditId}</span></div><div><small>Organization</small><strong>{inspectionPackage.organizationName}</strong></div><div><small>Package</small><strong>{inspectionPackage.id}</strong></div><div><small>Lead Inspector</small><strong>Current assignment</strong></div><div><small>Status</small><strong>{assignment?.status ?? "ASSIGNMENT"}</strong></div></section>
        <section className="lead-metric-grid"><article><span>Checklist Items</span><strong>{inspectionPackage.questions.length}</strong></article><article><span>Covered Items</span><strong>{inspectionPackage.questions.filter((question) => question.assignedInspectorUserIds.length > 0).length}</strong></article><article><span>Inspectors</span><strong>{Object.keys(workload).length}</strong></article><article><span>Sections</span><strong>{sections.length}</strong></article></section>
        <div className="lead-question-layout">
          <aside aria-label="Inspector workload" className="lead-panel lead-inspector-list" role="region"><h2>Inspectors</h2>{Object.entries(workload).map(([subjectId, count]) => <article data-subject-id={subjectId} key={subjectId}><strong>{subjectId}</strong><small>Assigned: {count} questions</small></article>)}</aside>
          <section aria-label="Checklist questions" className="lead-panel lead-question-table"><div className="lead-filter-row"><label>Section<select aria-label="Section" value={section} onChange={(event) => setSection(event.target.value)}><option value="all">All Sections</option>{sections.map((item) => <option key={item} value={item}>{item}</option>)}</select></label><label>Search<input aria-label="Search questions" type="search" value={query} onChange={(event) => setQuery(event.target.value)} /></label></div><div className="lead-question-rows">{visibleQuestions.map((question) => <article data-question-id={question.id} key={question.id}><div><small>{question.sectionId}{question.regulatoryReference ? ` · ${question.regulatoryReference}` : ""}</small><strong>{question.id}</strong><em>{question.prompt}</em></div><b>{question.assignedInspectorUserIds.join(", ") || "Unassigned"}</b></article>)}</div></section>
          <aside className="lead-panel lead-assignment-control"><h2>Pre-materialization coverage</h2><p>Question coverage is a preparation command and must be completed before the Audit package exists.</p>{assignment?.assignmentId ? <Link className="lead-button lead-button--primary" to={`/lead-inspector/audit-preparation?assignmentId=${encodeURIComponent(assignment.assignmentId)}`}>Open Lead preparation workspace</Link> : <p role="note">A server-owned assignment is required before coverage can be changed.</p>}<p>Question assignment cannot change report approval authority.</p></aside>
        </div>
      </> : <p>Loading exact checklist question scope…</p>}
    </div>
  </WorkspaceShell>;
}
