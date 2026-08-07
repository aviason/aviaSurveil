import type { Backend } from "../../backend/backend";

export type PreliminaryReportStep = "inspection" | "content" | "attachments" | "review";
export type AssignmentPriority = "ROUTINE" | "HIGH" | "CRITICAL";

export interface PreliminaryReportDraft {
  reportId: "PR-2026-018";
  auditId: "AUD-2026-001";
  version: 1;
  activeStep: PreliminaryReportStep;
  executiveSummary: string;
  commentToAuditee: string;
  internalCaaNote: string;
  saved: boolean;
}

export interface SavedQuestionAssignment {
  auditId: string;
  packageId: string;
  questionId: string;
  inspectorSubjectId: string;
  dueDate: string;
  priority: AssignmentPriority;
  instructions: string;
}

interface LeadWorkspaceState {
  preliminaryDraft: PreliminaryReportDraft;
  questionAssignments: Map<string, SavedQuestionAssignment>;
}

const workspaceByBackend = new WeakMap<Backend, LeadWorkspaceState>();

function stateFor(backend: Backend): LeadWorkspaceState {
  const existing = workspaceByBackend.get(backend);
  if (existing) return existing;
  const created: LeadWorkspaceState = {
    preliminaryDraft: {
      reportId: "PR-2026-018",
      auditId: "AUD-2026-001",
      version: 1,
      activeStep: "inspection",
      executiveSummary: "Canonical Finding facts are ready for Lead Inspector report preparation.",
      commentToAuditee: "Confirm the configured report facts before authorized release.",
      internalCaaNote: "CAA-only review note for PR-2026-018.",
      saved: false,
    },
    questionAssignments: new Map(),
  };
  workspaceByBackend.set(backend, created);
  return created;
}

export function readPreliminaryReportDraft(backend: Backend): PreliminaryReportDraft {
  return { ...stateFor(backend).preliminaryDraft };
}

export function savePreliminaryReportDraft(backend: Backend, draft: PreliminaryReportDraft): PreliminaryReportDraft {
  const saved = { ...draft, saved: true };
  stateFor(backend).preliminaryDraft = saved;
  return { ...saved };
}

export function readQuestionAssignments(backend: Backend): ReadonlyMap<string, SavedQuestionAssignment> {
  return new Map([...stateFor(backend).questionAssignments].map(([id, assignment]) => [id, { ...assignment }]));
}

export function saveQuestionAssignments(backend: Backend, assignments: readonly SavedQuestionAssignment[]): ReadonlyMap<string, SavedQuestionAssignment> {
  const state = stateFor(backend);
  for (const assignment of assignments) state.questionAssignments.set(assignment.questionId, { ...assignment });
  return readQuestionAssignments(backend);
}
