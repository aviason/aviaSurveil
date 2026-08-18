import { useEffect, useMemo, useState, type ReactNode } from "react";
import { Link } from "react-router-dom";

import { useScenario } from "../../app/scenario-context";
import { useApplicationRuntime } from "../../app/providers";
import { BackendHttpError } from "../../backend/http-backend";
import { DataRegister, type DataRegisterColumn } from "../../ui/workbench/data-register";
import { DueState } from "../../ui/workbench/due-state";
import { StatusPill, type StatusPillTone } from "../../ui/workbench/status-pill";
import { CommandError, errorMessage, formatLocalDate, WorkspaceShell } from "../shared/workspace-shell";

interface AssignmentRegisterRow extends Record<string, ReactNode> {
  audit: ReactNode;
  organization: ReactNode;
  status: ReactNode;
  dueDate: ReactNode;
  dueState: ReactNode;
  nextAction: ReactNode;
  rowId: string;
}

const assignmentColumns: readonly DataRegisterColumn<AssignmentRegisterRow>[] = [
  { key: "audit", header: "Audit", mobileRender: (row) => row.rowId },
  { key: "organization", header: "Organization" },
  { key: "status", header: "Status" },
  { key: "dueDate", header: "Due Date" },
  { key: "dueState", header: "Due state" },
  { key: "nextAction", header: "Next action" },
];

function statusTone(status: string): StatusPillTone {
  if (status.includes("SUBMITTED") || status.includes("COMPLETED")) return "success";
  if (status.includes("OVERDUE")) return "danger";
  if (status.includes("IN_PROGRESS")) return "warning";
  return "neutral";
}

function formatStatus(value: string): string {
  return value.replaceAll("_", " ");
}

type AssignmentGlyphKind = "document" | "clock" | "check" | "calendar" | "layers";

function AssignmentGlyph({ kind }: { kind: AssignmentGlyphKind }) {
  const path = {
    document: <><path d="M6 3.75h8l4 4V20.25H6z" /><path d="M14 3.75v4h4M9 12h6M9 15.5h6" /></>,
    clock: <><circle cx="12" cy="12" r="8.25" /><path d="M12 7.5v4.75l3.25 1.9" /></>,
    check: <><circle cx="12" cy="12" r="8.25" /><path d="m8.1 12.2 2.55 2.55 5.3-5.4" /></>,
    calendar: <><rect x="4.25" y="5.5" width="15.5" height="14" rx="1.75" /><path d="M7.5 3.75v3.5M16.5 3.75v3.5M4.25 9.25h15.5" /></>,
    layers: <><path d="m12 4 7.75 4-7.75 4-7.75-4z" /><path d="m4.25 12 7.75 4 7.75-4M4.25 16 12 20l7.75-4" /></>,
  }[kind];

  return (
    <svg aria-hidden="true" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.8">
      {path}
    </svg>
  );
}

function isUnavailableAssignmentProjection(error: unknown): boolean {
  return error instanceof BackendHttpError && error.status === 404;
}

export function InspectorAssignmentsPage() {
  const { projection, actions } = useScenario();
  const runtime = useApplicationRuntime();
  const [error, setError] = useState<string | null>(null);
  const [unavailableReason, setUnavailableReason] = useState<string | null>(null);
  const [startingAuditId, setStartingAuditId] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [typeFilter, setTypeFilter] = useState("all");
  const [organizationFilter, setOrganizationFilter] = useState("all");
  const [dateFilter, setDateFilter] = useState("all");

  useEffect(() => {
    void actions.loadAssignments().catch((cause) => {
      if (isUnavailableAssignmentProjection(cause)) {
        setError(null);
        setUnavailableReason("Assignments are not provisioned in this local AGA workspace.");
        return;
      }
      setError(errorMessage(cause));
    });
    // The injected Backend instance is fixed for the lifetime of this candidate shell.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const dueSoonCount = projection.assignments.filter(
    (assignment) => assignment.dueState === "DUE_SOON",
  ).length;
  const inProgressCount = projection.assignments.filter(
    (assignment) => assignment.status === "IN_PROGRESS",
  ).length;
  const primaryAssignment = projection.assignments[0] ?? null;
  const completedCount = projection.assignments.filter((assignment) => assignment.status.includes("COMPLETED")).length;
  const overdueCount = projection.assignments.filter((assignment) => assignment.dueState === "OVERDUE").length;
  const visibleAssignments = projection.assignments.filter((assignment) => {
    if (statusFilter === "OVERDUE" && assignment.dueState !== "OVERDUE") return false;
    if (statusFilter !== "all" && statusFilter !== "OVERDUE" && assignment.status !== statusFilter) return false;
    if (typeFilter === "cabin" && !assignment.title.toLowerCase().includes("cabin")) return false;
    if (organizationFilter !== "all" && assignment.organizationName !== organizationFilter) return false;
    if (dateFilter === "due-soon" && assignment.dueState !== "DUE_SOON") return false;
    const normalizedQuery = query.trim().toLowerCase();
    return !normalizedQuery || [assignment.auditId, assignment.title, assignment.organizationName]
      .join(" ")
      .toLowerCase()
      .includes(normalizedQuery);
  });

  function resetFilters(): void {
    setQuery("");
    setStatusFilter("all");
    setTypeFilter("all");
    setOrganizationFilter("all");
    setDateFilter("all");
  }

  async function startInspection(assignment: (typeof projection.assignments)[number]): Promise<void> {
    const workflow = (runtime.backendForRole?.("inspector") ?? runtime.backend).canonicalAuditWorkflow;
    if (!workflow || !assignment.inspectionRevision) {
      setError("Inspector start is unavailable until the server provides the materialized inspection revision.");
      return;
    }
    setStartingAuditId(assignment.auditId);
    setError(null);
    try {
      await workflow.start(assignment.auditId, {
        operationId: `START-${assignment.auditId}-${assignment.inspectionRevision}`,
        expectedInspectionRevision: assignment.inspectionRevision,
      });
      await actions.loadAssignments();
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setStartingAuditId(null);
    }
  }
  const filtersActive = statusFilter !== "all" ||
    query !== "" ||
    typeFilter !== "all" ||
    organizationFilter !== "all" ||
    dateFilter !== "all";
  const rows = useMemo<AssignmentRegisterRow[]>(
    () =>
      visibleAssignments.map((assignment) => ({
        rowId: assignment.auditId,
        audit: (
          <span className="inspector-register-audit">
            <strong aria-hidden="true">{assignment.title}</strong>
            <small>{assignment.auditId}</small>
          </span>
        ),
        organization: assignment.organizationName,
        status: (
          <StatusPill
            label={formatStatus(assignment.status)}
            tone={statusTone(assignment.status)}
          />
        ),
        dueDate: formatLocalDate(assignment.dueDate),
        dueState: <DueState dueDate={assignment.dueDate} today="2026-06-15" />,
        nextAction: (
          <span className="inspector-register-action">
            <span>{assignment.nextAction}</span>
            {assignment.status === "SCHEDULED" ? (
              <button className="primary-link" disabled={startingAuditId === assignment.auditId} onClick={() => void startInspection(assignment)} type="button">
                {startingAuditId === assignment.auditId ? "Starting…" : "Start inspection"}
              </button>
            ) : assignment.packageId && assignment.status === "IN_PROGRESS" ? (
              <Link className="primary-link" to={`/inspector/audits/${encodeURIComponent(assignment.auditId)}`}>
                Open {assignment.title}
              </Link>
            ) : (
              <span title={assignment.status === "AWAITING_AUDITEE_CONFIRMATION" ? "Auditee confirmation is required before Inspector start." : "Execution is not ready for this assignment."}>
                {assignment.status === "AWAITING_AUDITEE_CONFIRMATION" ? "Awaiting auditee confirmation" : "Preparation in progress"}
              </span>
            )}
          </span>
        ),
      })),
    [visibleAssignments],
  );

  return (
    <WorkspaceShell roleLabel="CAA Inspector" routeLabel="My Assignments">
      <div className="inspector-assignment-page">
        <header className="inspector-assignment-head workbench-page-header">
          <div>
            <h1>My Assignments</h1>
            <p>View and manage all audits and tasks assigned to you.</p>
          </div>
        </header>
      <CommandError message={error} />
      {unavailableReason ? <p className="inspector-empty-state" role="status">{unavailableReason}</p> : null}
      <section className="inspector-assignment-kpis" aria-label="Assignment attention">
        <button aria-pressed={statusFilter === "all"} className={statusFilter === "all" ? "is-active" : ""} onClick={() => setStatusFilter("all")} type="button">
          <span className="inspector-assignment-kpi__icon"><AssignmentGlyph kind="document" /></span><span><b>Open Assignments</b><strong>{projection.assignments.length}</strong><em>Audits</em></span>
        </button>
        <button aria-pressed={statusFilter === "IN_PROGRESS"} className={statusFilter === "IN_PROGRESS" ? "is-active is-warn" : "is-warn"} onClick={() => setStatusFilter("IN_PROGRESS")} type="button">
          <span className="inspector-assignment-kpi__icon"><AssignmentGlyph kind="clock" /></span><span><b>In Progress</b><strong>{inProgressCount}</strong><em>Audits</em></span>
        </button>
        <button aria-pressed={statusFilter === "COMPLETED"} className={statusFilter === "COMPLETED" ? "is-active is-ok" : "is-ok"} onClick={() => setStatusFilter("COMPLETED")} type="button">
          <span className="inspector-assignment-kpi__icon"><AssignmentGlyph kind="check" /></span><span><b>Completed</b><strong>{completedCount}</strong><em>Audits</em></span>
        </button>
        <button aria-pressed={statusFilter === "OVERDUE"} className={statusFilter === "OVERDUE" ? "is-active is-danger" : "is-danger"} onClick={() => setStatusFilter("OVERDUE")} type="button">
          <span className="inspector-assignment-kpi__icon"><AssignmentGlyph kind="calendar" /></span><span><b>Overdue</b><strong>{overdueCount}</strong><em>Audits</em></span>
        </button>
        <button aria-pressed={statusFilter === "all" && query === "" && typeFilter === "all" && organizationFilter === "all" && dateFilter === "all"} className="is-neutral" onClick={resetFilters} type="button">
          <span className="inspector-assignment-kpi__icon"><AssignmentGlyph kind="layers" /></span><span><b>Total Assigned</b><strong>{projection.assignments.length}</strong><em>Audits</em></span>
        </button>
      </section>
      <section className="inspector-assignment-filters" aria-label="Assignment filters">
        <label className="inspector-assignment-filter inspector-assignment-filter--search"><span>Search audits</span><input type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search audits..." /><b aria-hidden="true">⌕</b></label>
        <label className="inspector-assignment-filter"><span>Status</span><select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}><option value="all">All Status</option><option value="IN_PROGRESS">In Progress</option><option value="COMPLETED">Completed</option><option value="OVERDUE">Overdue</option></select></label>
        <label className="inspector-assignment-filter"><span>Type</span><select value={typeFilter} onChange={(event) => setTypeFilter(event.target.value)}><option value="all">All Types</option><option value="cabin">Cabin Safety</option></select></label>
        <label className="inspector-assignment-filter"><span>Organization</span><select value={organizationFilter} onChange={(event) => setOrganizationFilter(event.target.value)}><option value="all">All Organizations</option>{[...new Set(projection.assignments.map((assignment) => assignment.organizationName))].map((organization) => <option key={organization} value={organization}>{organization}</option>)}</select></label>
        <label className="inspector-assignment-filter"><span>Date</span><select value={dateFilter} onChange={(event) => setDateFilter(event.target.value)}><option value="all">Date Range</option><option value="due-soon">Due Soon</option></select></label>
        <button
          aria-label="Reset assignment filters"
          className="inspector-filter-action"
          disabled={!filtersActive}
          onClick={resetFilters}
          title={!filtersActive ? "Assignment filters are already at their defaults." : undefined}
          type="button"
        >↺ Reset</button>
      </section>
      <section className="inspector-register" aria-label="Assigned Audits">
        <DataRegister
          caption="Assigned Audits"
          columns={assignmentColumns}
          rowKey={(row) => row.rowId}
          rows={rows}
        />
      </section>
      {rows.length === 0 ? <p className="inspector-empty-state">No assignments match these filters.</p> : null}
      <span className="qualification-boundary">
        Inspector Workspace. Current owner: CAA Inspector. {dueSoonCount} Due Soon. Next action: {primaryAssignment?.nextAction ?? "None"}
      </span>
      </div>
    </WorkspaceShell>
  );
}
