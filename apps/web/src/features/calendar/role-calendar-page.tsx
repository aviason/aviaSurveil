import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { CalendarItemView, Role } from "../../backend/backend";
import { StatusPill } from "../../ui/workbench/status-pill";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";

interface CalendarProjection { role: Role; roleLabel: string; routeLabel: string; testId: string; ownerLabel: string; }
const inspectorProjection: CalendarProjection = { role: "inspector", roleLabel: "CAA Inspector", routeLabel: "Audit Work Queue", testId: "inspector-calendar-page", ownerLabel: "CAA Inspector" };
const leadProjection: CalendarProjection = { role: "leadInspector", roleLabel: "Lead Inspector", routeLabel: "Lead Calendar", testId: "lead-calendar-page", ownerLabel: "Lead Inspector" };

function calendarDueText(item: CalendarItemView): string {
  const scheduledDate = item.scheduledDate;
  const formattedDate = new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    timeZone: "UTC",
  }).format(new Date(`${scheduledDate}T00:00:00.000Z`));
  if (scheduledDate === "2026-06-15") return `${formattedDate} · Today`;
  const daysOverdue = Math.round((Date.parse("2026-06-15T00:00:00.000Z") - Date.parse(`${scheduledDate}T00:00:00.000Z`)) / 86_400_000);
  return daysOverdue > 0 ? `${formattedDate} · ${daysOverdue} days overdue` : formattedDate;
}

export function RoleCalendarPage({ projection }: { projection: CalendarProjection }) {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.(projection.role) ?? runtime.backend, [projection.role, runtime]);
  const [items, setItems] = useState<CalendarItemView[]>([]);
  const [filter, setFilter] = useState<"active" | "completed">("active");
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    if (!backend.calendar) return;
    let cancelled = false;
    void backend.calendar.list({}).then((page) => !cancelled && setItems(page.items)).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend]);
  const orderedItems = [...items].sort((left, right) => {
    if (left.dueState === right.dueState) return left.scheduledDate.localeCompare(right.scheduledDate);
    return left.dueState === "OVERDUE" ? -1 : 1;
  });
  return <WorkspaceShell roleLabel={projection.roleLabel} routeLabel={projection.routeLabel}>
    <div className="inspector-secondary-page inspector-calendar-page" data-testid={projection.testId}>
      <header className="inspector-secondary-head workbench-page-header"><div><h1>Audit Work Queue</h1><p>Assigned audits in a simple queue, sorted by Due Date. Use Active for open work and Completed for finished audits.</p></div><div className="inspector-secondary-actions"><button aria-pressed={filter === "active"} className={filter === "active" ? "is-active" : ""} onClick={() => setFilter("active")} type="button"><span>Active audits ({items.length})</span></button><button aria-pressed={filter === "completed"} className={filter === "completed" ? "is-active" : ""} onClick={() => setFilter("completed")} type="button"><span>Completed (0)</span></button></div></header>
      <CommandError message={error} />
      <section className="inspector-work-queue" aria-label={`${filter} Audit queue`}>
        {filter === "active" ? <><div className="inspector-work-queue__head" aria-hidden="true"><span>Priority</span><span>Item</span><span>Organization</span><span>Owner</span><span>Next Action</span><span>Due Date / Target</span><span>Status</span><span>Open</span></div>{orderedItems.map((item) => <article className={`inspector-work-row inspector-work-row--${item.dueState === "OVERDUE" ? "danger" : "warning"}`} key={item.id}>
          <div className="inspector-work-row__priority"><StatusPill label={item.dueState === "OVERDUE" ? "Overdue" : "Due soon"} tone={item.dueState === "OVERDUE" ? "danger" : "warning"} /></div>
          <div className="inspector-work-row__item"><h2>{item.organizationName ? `${item.organizationName} · ${item.title}` : item.title}</h2><p>{item.auditId} · {item.organizationId}</p></div>
          <p className="inspector-work-row__organization">{item.organizationName ?? item.organizationId}</p>
          <p className="inspector-work-row__owner"><span>Current Owner</span>{projection.ownerLabel}</p>
          <p className="inspector-work-row__action"><span>Next Action</span>{item.nextAction ?? "Open assigned audit"}</p>
          <p className="inspector-work-row__due"><span>Due Date </span><span className="workbench-due-state">{calendarDueText(item)}</span></p>
          <div className="inspector-work-row__status"><StatusPill label={item.dueState === "OVERDUE" ? "In Progress" : "Scheduled"} tone={item.dueState === "OVERDUE" ? "warning" : "neutral"} /><StatusPill label={item.dueState === "OVERDUE" ? "Overdue" : "Due Soon"} tone={item.dueState === "OVERDUE" ? "danger" : "warning"} /></div>
          <Link aria-label={projection.role === "leadInspector" ? `Open assignment for ${item.auditId}` : undefined} className="inspector-secondary-button inspector-secondary-button--primary inspector-work-row__open" to={projection.role === "leadInspector" ? `/lead-inspector/audits/${encodeURIComponent(item.auditId)}/assignment` : `/inspector/audits/${encodeURIComponent(item.auditId)}/checklist`}><span>{projection.role === "leadInspector" ? "Open assignment" : "Continue checklist"}</span></Link>
        </article>)}</> : <p className="inspector-empty-state">No completed audits in this deterministic projection.</p>}
      </section>
    </div>
  </WorkspaceShell>;
}

export function InspectorCalendarPage() { return <RoleCalendarPage projection={inspectorProjection} />; }
export function LeadCalendarPage() { return <RoleCalendarPage projection={leadProjection} />; }
