import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { AssignmentSummary, InspectionPackage } from "../../backend/backend";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";
import {
  type AssignmentPriority,
  readQuestionAssignments,
  type SavedQuestionAssignment,
} from "../shared/lead-workspace-state";

const inspectorNames: Readonly<Record<string, string>> = {
  "USR-INSPECTOR-AMINA": "Amina Inspector",
  "USR-INSPECTOR-DAVID": "David Inspector",
};

interface QuestionFacet {
  department: "CABIN_SAFETY" | "FLIGHT_OPERATIONS";
  risk: "ROUTINE" | "HIGH" | "CRITICAL";
}

const questionFacets: Readonly<Record<string, QuestionFacet>> = {
  "CAB-GALLEY-001": { department: "CABIN_SAFETY", risk: "HIGH" },
  "CAB-LAV-001": { department: "CABIN_SAFETY", risk: "ROUTINE" },
  "CAB-PAX-SEAT-001": { department: "CABIN_SAFETY", risk: "HIGH" },
  "CAB-EMEQ-PBE-001": { department: "CABIN_SAFETY", risk: "CRITICAL" },
  "CAB-VID-CREW-SEAT-001": { department: "FLIGHT_OPERATIONS", risk: "HIGH" },
  "CAB-COCKPIT-GEN-001": { department: "FLIGHT_OPERATIONS", risk: "CRITICAL" },
};

function priorityLabel(priority: AssignmentPriority): string {
  return priority.toLowerCase().replace(/^./, (letter) => letter.toUpperCase());
}

export function QuestionAssignmentPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("leadInspector") ?? runtime.backend, [runtime]);
  const [assignment, setAssignment] = useState<AssignmentSummary | null>(null);
  const [inspectionPackage, setInspectionPackage] = useState<InspectionPackage | null>(null);
  const [section, setSection] = useState("all");
  const [query, setQuery] = useState("");
  const [priorityFilter, setPriorityFilter] = useState("all");
  const [departmentFilter, setDepartmentFilter] = useState("all");
  const [riskFilter, setRiskFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");
  const [savedAssignments, setSavedAssignments] = useState<ReadonlyMap<string, SavedQuestionAssignment>>(() => readQuestionAssignments(backend));
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
    const saved = savedAssignments.get(question.id);
    const facet = questionFacets[question.id];
    return (section === "all" || question.sectionId === section)
      && `${question.id} ${question.prompt}`.toLowerCase().includes(query.toLowerCase())
      && (priorityFilter === "all" || saved?.priority === priorityFilter)
      && (departmentFilter === "all" || facet?.department === departmentFilter)
      && (riskFilter === "all" || facet?.risk === riskFilter)
      && (statusFilter === "all" || (statusFilter === "SAVED" ? Boolean(saved) : !saved));
  });
  const workload = (inspectionPackage?.questions ?? []).reduce<Record<string, number>>((counts, question) => {
    const subjectId = savedAssignments.get(question.id)?.inspectorSubjectId ?? question.assignedInspectorUserIds[0];
    if (subjectId) counts[subjectId] = (counts[subjectId] ?? 0) + 1;
    return counts;
  }, {});

  return <WorkspaceShell roleLabel="Lead Inspector" routeLabel="Lead Review">
    <div className="lead-secondary-page lead-question-assignment" data-audit-id={assignment?.auditId ?? inspectionPackage?.auditId ?? undefined} data-testid="lead-question-assignment-page">
      <header className="lead-secondary-header workbench-page-header"><div><Link className="lead-back-link" to={assignment ? `/lead-inspector/audits/${assignment.auditId}/assignment` : "/lead-inspector/lead-review"}>← Back to Assignment Overview</Link><h1>Assign Checklist Questions</h1><p>Assign exact backend question IDs to Inspectors for this Audit.</p></div></header>
      <CommandError message={error} />
      {inspectionPackage ? <>
        <section className="lead-fact-strip"><div><small>Inspection</small><strong>{inspectionPackage.title}</strong><span>{inspectionPackage.auditId}</span></div><div><small>Organization</small><strong>{inspectionPackage.organizationName}</strong></div><div><small>Package</small><strong>{inspectionPackage.id}</strong></div><div><small>Lead Inspector</small><strong>Current assignment</strong></div><div><small>Status</small><strong>{assignment?.status ?? "ASSIGNMENT"}</strong></div></section>
        <section className="lead-metric-grid"><article><span>Checklist Items</span><strong>{inspectionPackage.questions.length}</strong></article><article><span>Covered Items</span><strong>{inspectionPackage.questions.filter((question) => question.assignedInspectorUserIds.length > 0).length}</strong></article><article><span>Inspectors</span><strong>{Object.keys(workload).length}</strong></article><article><span>Sections</span><strong>{sections.length}</strong></article></section>
        <div className="lead-filter-row lead-assignment-filters">
          <label>Department<select aria-label="Department filter" value={departmentFilter} onChange={(event) => setDepartmentFilter(event.target.value)}><option value="all">All Departments</option><option value="CABIN_SAFETY">Cabin Safety</option><option value="FLIGHT_OPERATIONS">Flight Operations</option></select></label>
          <label>Risk<select aria-label="Risk filter" value={riskFilter} onChange={(event) => setRiskFilter(event.target.value)}><option value="all">All Risks</option><option value="CRITICAL">Critical</option><option value="HIGH">High</option><option value="ROUTINE">Routine</option></select></label>
          <label>Priority<select aria-label="Priority filter" value={priorityFilter} onChange={(event) => setPriorityFilter(event.target.value)}><option value="all">All Priorities</option><option value="CRITICAL">Critical</option><option value="HIGH">High</option><option value="ROUTINE">Routine</option></select></label>
          <label>Status<select aria-label="Status filter" value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}><option value="all">All Statuses</option><option value="SAVED">Saved assignment</option><option value="UNASSIGNED">No saved assignment</option></select></label>
        </div>
        <div className="lead-question-layout">
          <aside aria-label="Inspector workload" className="lead-panel lead-inspector-list" role="region"><h2>Inspectors</h2>{Object.entries(workload).map(([subjectId, count]) => <article data-subject-id={subjectId} key={subjectId}><strong>{inspectorNames[subjectId] ?? subjectId}</strong><span>{subjectId}</span><small>Assigned: {count} questions</small></article>)}</aside>
          <section aria-label="Checklist questions" className="lead-panel lead-question-table"><div className="lead-filter-row"><label>Section<select aria-label="Section" value={section} onChange={(event) => setSection(event.target.value)}><option value="all">All Sections</option>{sections.map((item) => <option key={item} value={item}>{item}</option>)}</select></label><label>Search<input aria-label="Search questions" type="search" value={query} onChange={(event) => setQuery(event.target.value)} /></label></div><div className="lead-question-rows">{visibleQuestions.map((question) => { const saved = savedAssignments.get(question.id); return <article data-question-id={question.id} key={question.id}><div><small>{question.sectionId} · {questionFacets[question.id]?.department.replaceAll("_", " ")} · {questionFacets[question.id]?.risk}</small><strong>{question.id}</strong><em>{question.prompt}</em></div><b>{saved ? `${saved.inspectorSubjectId} · ${saved.dueDate} · ${priorityLabel(saved.priority)}` : question.assignedInspectorUserIds.join(", ")}</b></article>; })}</div></section>
          <aside className="lead-panel lead-assignment-control"><h2>Pre-materialization coverage</h2><p>Question coverage is a preparation command and must be completed before the Audit package exists.</p>{assignment?.assignmentId ? <Link className="lead-button lead-button--primary" to={`/lead-inspector/audit-preparation?assignmentId=${encodeURIComponent(assignment.assignmentId)}`}>Open Lead preparation workspace</Link> : <p role="note">A server-owned assignment is required before coverage can be changed.</p>}<p>Question assignment cannot change report approval authority.</p></aside>
        </div>
        {savedAssignments.size ? <section aria-label="Saved question assignments" className="lead-panel lead-saved-assignments"><h2>Saved question assignments</h2>{[...savedAssignments.values()].map((assignment) => <article data-question-id={assignment.questionId} key={assignment.questionId}><strong>{assignment.questionId}</strong><span>{assignment.auditId} · {assignment.packageId}</span><span>{assignment.inspectorSubjectId}</span><span>{assignment.dueDate} · {priorityLabel(assignment.priority)}</span><p>{assignment.instructions || "No additional instructions."}</p></article>)}</section> : null}
      </> : <p>Loading exact checklist question scope…</p>}
    </div>
  </WorkspaceShell>;
}
