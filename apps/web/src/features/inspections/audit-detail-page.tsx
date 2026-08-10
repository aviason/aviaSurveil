import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import { useScenario } from "../../app/scenario-context";
import { CommandError, errorMessage, formatLocalDate, WorkspaceShell } from "../shared/workspace-shell";
import { OfflineReadinessPanel } from "./offline-readiness-panel";

const sections: readonly { id: string; label: string; note?: string }[] = [
  { id: "GALLEY", label: "Galley" },
  { id: "LAV", label: "Lavatories" },
  { id: "PAX SEAT", label: "Passenger Seats" },
  { id: "EM EQ / PBE", label: "Emergency Equipment", note: "EM EQ / PBE" },
  { id: "VID+CREW SEAT", label: "Video + Crew Seat" },
  { id: "COCKPIT+CAB GEN COND+EXITS", label: "Cockpit, Cabin General Condition + Exits" },
];

export function AuditDetailPage() {
  const runtime = useApplicationRuntime();
  const { auditId: requestedAuditId } = useParams<{ auditId: string }>();
  const { projection, actions } = useScenario();
  const [assignment, setAssignment] = useState<{ auditId: string; packageId?: string | null; dueDate: string | null; currentOwnerDisplayName?: string | null } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [draftSaved, setDraftSaved] = useState(false);
  const [selectedSectionId, setSelectedSectionId] = useState("EM EQ / PBE");

  useEffect(() => {
    const backend = runtime.backendForRole?.("inspector") ?? runtime.backend;
    void backend.assignments.list({}).then(({ items }) => {
      const current = items.find((item) => item.auditId === requestedAuditId && item.packageId);
      if (!current?.packageId) throw new Error("No executable canonical Audit package is available for this Inspector.");
      setAssignment(current);
      return actions.loadPackage(current.packageId);
    }).catch((cause) => setError(errorMessage(cause)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [requestedAuditId]);

  const packageView = projection.packageView;
  const auditTitle = packageView?.title ?? "Audit package unavailable";
  const organizationName = packageView?.organizationName ?? "Organization unavailable";
  const checklistQuestionCount = packageView?.questions.length ?? 6;
  const selectedSectionIndex = Math.max(0, sections.findIndex((section) => section.id === selectedSectionId));
  const selectedSection = sections[selectedSectionIndex];
  const selectedQuestion = packageView?.questions.find((question) =>
    question.sectionId === selectedSectionId ||
    (selectedSectionId === "EM EQ / PBE" && question.id === "CAB-EMEQ-PBE-001"),
  );
  const previousSection = sections[selectedSectionIndex - 1];
  const nextSection = sections[selectedSectionIndex + 1];
  const checklistRunnerPath = packageView
    ? `/inspector/audits/${encodeURIComponent(packageView.auditId)}/checklist?packageId=${encodeURIComponent(packageView.id)}`
    : "/inspector/inspector-assignments";

  function downloadChecklist(): void {
    const body = [
      auditTitle,
      `Inspection ID: ${packageView?.auditId ?? "Unavailable"}`,
      ...(packageView?.questions.map((question, index) => `${index + 1}. ${question.prompt}`) ?? []),
    ].join("\n");
    const url = URL.createObjectURL(new Blob([body], { type: "text/plain;charset=utf-8" }));
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `${packageView?.auditId ?? "audit"}-checklist.txt`;
    anchor.click();
    URL.revokeObjectURL(url);
  }

  return (
    <WorkspaceShell roleLabel="CAA Inspector" routeLabel="Audit Plan Calendar">
      <div className="inspection-exec" data-testid="audit-dossier">
        <div className="audit-contract-facts" aria-label="Audit backend facts">
          <span>CABIN</span>
          <span>{packageView?.checklistStatus ?? "UNAVAILABLE"}</span>
          <span>CAA Inspector</span>
          <span>Due Date: {formatLocalDate(assignment?.dueDate ?? null)}</span>
          <span>{packageView?.id ?? "Package unavailable"}</span>
          <span>{checklistQuestionCount}</span>
          <span>Offline eligible</span>
          <span>Run assigned Cabin checklist</span>
        </div>
        <Link className="inspection-back" to="/inspector/inspector-assignments">
          ← Back to Inspections
        </Link>
        <header className="inspection-exec__head workbench-page-header">
          <div>
            <h1>{auditTitle}</h1>
            <div className="inspection-title-meta">
              <span>{organizationName}</span>
              <span>Cabin Inspection</span>
              <span>Canonical package v{packageView?.packageVersion ?? "—"}</span>
            </div>
            <p className="inspection-assignment-context">
              Inspector scope: <strong>{assignment?.currentOwnerDisplayName ?? "Current assigned Inspector"}</strong>. Other Inspectors&apos; assigned
              questions are visible but read-only.
            </p>
            <div className="inspection-status-line">
              <span className="inspection-badge">● In Progress</span>
              {draftSaved ? <span className="inspection-save-state">Draft saved</span> : null}
            </div>
          </div>
          <div className="inspection-exec__actions">
            <button aria-label="Download checklist" onClick={downloadChecklist} type="button">⇩&nbsp; Download Checklist</button>
            <button aria-label="Save draft" onClick={() => setDraftSaved(true)} type="button">💾&nbsp; Save Draft</button>
            <Link
              aria-label="Run Cabin checklist"
              className="inspection-submit-action"
              to={checklistRunnerPath}
            >
              ➤&nbsp; Submit to Lead Inspector
            </Link>
          </div>
        </header>
        <CommandError message={error} />
        <section className="inspection-mobile-command" aria-label="Audit next action">
          <div><span>Current owner</span><strong>{assignment?.currentOwnerDisplayName ?? "Current assigned Inspector"}</strong></div>
          <div className="is-warn"><span>Next action</span><strong>Complete checklist sections</strong></div>
          <div><span>Due Date</span><strong>{formatLocalDate(assignment?.dueDate ?? null)}</strong></div>
          <div><span>Status</span><strong>{packageView?.checklistStatus ?? "Unavailable"}</strong></div>
          <Link aria-label="Submit to Lead Inspector" to={checklistRunnerPath}>
            Submit to Lead Inspector
          </Link>
        </section>
        <section className="inspection-summary-card" aria-label="Inspection summary">
          <div className="inspection-summary-item">
            <span className="inspection-summary-icon">📅</span>
            <div>
              <span>Inspection ID</span>
              <b data-testid="audit-id">{packageView?.auditId ?? "Unavailable"}</b>
            </div>
          </div>
          <div className="inspection-summary-item">
            <span className="inspection-summary-icon">📅</span>
            <div><span>Start Date</span><b>{formatLocalDate(assignment?.dueDate ?? null)}</b></div>
          </div>
          <div className="inspection-summary-item">
            <span className="inspection-summary-icon">📅</span>
            <div><span>End Date</span><b>{formatLocalDate(assignment?.dueDate ?? null)}</b></div>
          </div>
          <div className="inspection-summary-item inspection-summary-item--wide">
            <div><span>Checklist Progress</span><b>0 / {checklistQuestionCount} (0%)</b></div>
            <div className="inspection-progress" aria-hidden="true"><span /></div>
          </div>
        </section>
        <div className="inspection-workspace">
          <aside className="inspection-side">
            <section className="inspection-panel">
              <h2>Checklist Sections</h2>
              <div className="inspection-sections">
                {sections.map((section, index) => (
                  <button
                    aria-pressed={section.id === selectedSectionId}
                    className={`inspection-section${section.id === selectedSectionId ? " is-active" : ""}`}
                    key={section.id}
                    onClick={() => setSelectedSectionId(section.id)}
                    type="button"
                  >
                    <span id={`inspection-section-${index + 1}`}>
                      {index + 1}. {section.label}
                      {section.note ? <small>{section.note}</small> : null}
                    </span>
                    <b>0 / 1</b>
                  </button>
                ))}
              </div>
            </section>
            <section className="inspection-panel inspection-legend">
              <h2>Legend</h2>
              <p><span className="is-ok">✓</span> Compliant</p>
              <p><span className="is-danger">×</span> Non-Compliant</p>
              <p><span className="is-warn">◎</span> Observation</p>
              <p><span className="is-neutral">−</span> Not Applicable</p>
            </section>
          </aside>
          <section className="inspection-card" id="inspection-section-panel">
            <header className="inspection-card__head">
              <h2>{selectedSectionIndex + 1}. {selectedSection.label}</h2>
              <span>0 / 1 Completed&nbsp;&nbsp;⌃</span>
            </header>
            <div className="inspection-table-wrap">
              <table className="inspection-table">
                <thead>
                  <tr><th>No.</th><th>Checklist Item</th><th>Compliance</th><th>Comments</th><th>Attached File</th><th /></tr>
                </thead>
                <tbody>
                  <tr>
                    <td>{selectedSectionIndex + 1}.1</td>
                    <td>
                      <strong>{selectedQuestion?.prompt ?? `${selectedSection.label} checklist question is not included in this candidate package.`}</strong>
                      <small>{selectedQuestion?.regulatoryReference ?? "No configured reference is available in this candidate package."}</small>
                      <em>{selectedQuestion ? "Assigned to you" : "Read-only package summary"}</em>
                    </td>
                    <td>
                      <select aria-label={`Compliance for ${selectedSectionIndex + 1}.1`} disabled title="This package summary is read-only; run the checklist to record compliance." defaultValue="NOT_APPLICABLE">
                        <option value="NOT_APPLICABLE">Not Applicable</option>
                      </select>
                    </td>
                    <td><textarea aria-label="Comments for 4.1" disabled placeholder="Comment required for exception results" title="This package summary is read-only; run the checklist to add a comment." /></td>
                    <td><button disabled title="No Inspection Attachment is attached to this read-only package summary." type="button">🔗 No file attached</button></td>
                    <td>⋮</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <footer className="inspection-bottom-nav">
              <button aria-controls="inspection-section-panel" aria-label={previousSection ? `Previous section: ${previousSection.label}` : "Previous section unavailable"} disabled={!previousSection} onClick={() => previousSection && setSelectedSectionId(previousSection.id)} title={!previousSection ? "The first checklist section is already selected." : undefined} type="button">
                ← {previousSection?.label ?? "Previous Section"}
              </button>
              <span>Next Section</span>
              <button aria-controls="inspection-section-panel" aria-label={nextSection ? `Next section: ${nextSection.label}` : "Next section unavailable"} disabled={!nextSection} onClick={() => nextSection && setSelectedSectionId(nextSection.id)} title={!nextSection ? "The final checklist section is already selected." : undefined} type="button">
                {nextSection ? `${selectedSectionIndex + 2}. ${nextSection.label} →` : "Checklist complete"}
              </button>
            </footer>
          </section>
        </div>
        {packageView ? (
          <div className="inspection-offline-readiness">
            <OfflineReadinessPanel
              inspectionPackage={packageView}
              subjectId={runtime.subjectId ?? "USR-INSPECTOR-AMINA"}
            />
          </div>
        ) : null}
      </div>
    </WorkspaceShell>
  );
}
