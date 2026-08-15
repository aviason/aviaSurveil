import { type ChangeEvent, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import { useScenario } from "../../app/scenario-context";
import type { CapRevisionView, FindingStatus, Role } from "../../backend/backend";
import { RoleHandoff } from "../../auth/role-handoff";
import { useOptionalSession } from "../../auth/session-provider";
import { createRoleEntryPath } from "../../ui/role-select-page";
import {
  CommandError,
  errorMessage,
  formatLocalDate,
  formatSeverity,
  WorkspaceShell,
} from "../shared/workspace-shell";
import {
  createAuditeeFindingProjection,
  type AuditeeFindingProjection,
} from "./auditee-projection";

interface CapDraft {
  rootCause: string;
  correctiveAction: string;
  preventiveAction: string;
  responsiblePerson: string;
  targetCompletionDate: string;
  commentToCaa: string;
}

type FindingGroup = "all" | "open" | "in-progress" | "awaiting-review" | "closed";

const emptyCap: CapDraft = {
  rootCause: "",
  correctiveAction: "",
  preventiveAction: "",
  responsiblePerson: "",
  targetCompletionDate: "",
  commentToCaa: "",
};

const inProgressStatuses = new Set<FindingStatus>([
  "CAP_SUBMITTED",
  "CAP_ACCEPTED",
  "CAP_MORE_INFORMATION_REQUESTED",
  "EVIDENCE_REQUIRED",
  "EVIDENCE_SUBMITTED",
  "EVIDENCE_MORE_INFORMATION_REQUESTED",
  "PENDING_CAA_REVIEW",
  "PENDING_CLOSURE",
]);

const awaitingReviewStatuses = new Set<FindingStatus>([
  "CAP_SUBMITTED",
  "EVIDENCE_SUBMITTED",
  "PENDING_CAA_REVIEW",
  "PENDING_CLOSURE",
]);

function statusLabel(status: FindingStatus): string {
  const labels: Record<FindingStatus, string> = {
    DRAFT: "Draft",
    OPEN: "Open",
    WAITING_FOR_CAP: "Waiting for CAP",
    CAP_SUBMITTED: "CAP Submitted — Pending CAA Review",
    CAP_ACCEPTED: "CAP Accepted",
    CAP_REJECTED: "CAP Rejected",
    CAP_MORE_INFORMATION_REQUESTED: "CAP — More Information Requested",
    EVIDENCE_REQUIRED: "CAP Accepted — Evidence Required",
    EVIDENCE_SUBMITTED: "Evidence Submitted",
    PENDING_CAA_REVIEW: "Evidence — Pending CAA Review",
    EVIDENCE_MORE_INFORMATION_REQUESTED: "Evidence — More Information Requested",
    PENDING_CLOSURE: "Pending Closure",
    CLOSED: "Closed",
    ESCALATED: "Escalated",
  };
  return labels[status];
}

function lifecycleProgress(status: FindingStatus): { label: string; percent: number } {
  if (status === "CLOSED") return { label: "Closed", percent: 100 };
  if (["PENDING_CLOSURE", "PENDING_CAA_REVIEW", "EVIDENCE_SUBMITTED"].includes(status)) {
    return { label: "Evidence review", percent: 82 };
  }
  if (["EVIDENCE_REQUIRED", "EVIDENCE_MORE_INFORMATION_REQUESTED", "CAP_ACCEPTED"].includes(status)) {
    return { label: "Evidence required", percent: 66 };
  }
  if (["CAP_SUBMITTED", "CAP_MORE_INFORMATION_REQUESTED", "CAP_REJECTED"].includes(status)) {
    return { label: "CAP review", percent: 45 };
  }
  return { label: "CAP required", percent: 22 };
}

function matchesGroup(finding: AuditeeFindingProjection, group: FindingGroup): boolean {
  if (group === "all") return true;
  if (group === "closed") return finding.status === "CLOSED";
  if (group === "open") return finding.status !== "CLOSED";
  if (group === "in-progress") return inProgressStatuses.has(finding.status);
  return awaitingReviewStatuses.has(finding.status);
}

function latestRevision(items: CapRevisionView[]): CapRevisionView | null {
  return [...items].sort((left, right) => right.revision - left.revision)[0] ?? null;
}

function publicReviewComment(revision: CapRevisionView): string | null {
  return revision.latestReview?.commentToAuditee ?? null;
}

export function AuditeeCapPage() {
  const runtime = useApplicationRuntime();
  const session = useOptionalSession();
  const navigate = useNavigate();
  const { projection, actions } = useScenario();
  const [cap, setCap] = useState<CapDraft>(emptyCap);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [group, setGroup] = useState<FindingGroup>("all");
  const [auditFilter, setAuditFilter] = useState("all");
  const [severityFilter, setSeverityFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");
  const [query, setQuery] = useState("");
  const [online, setOnline] = useState(() => typeof navigator === "undefined" || navigator.onLine);
  const identityMode =
    session?.identityMode ?? runtime.identityMode ??
    (runtime.identityMode ?? "oidc-session");
  const handoffSession = session?.state ?? { status: "unauthenticated" as const };

  useEffect(() => {
    const updateOnlineState = () => setOnline(navigator.onLine);
    window.addEventListener("online", updateOnlineState);
    window.addEventListener("offline", updateOnlineState);
    return () => {
      window.removeEventListener("online", updateOnlineState);
      window.removeEventListener("offline", updateOnlineState);
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    void actions.loadAuditeeFindings().catch((cause) => {
      if (!cancelled) setError(errorMessage(cause));
    });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const publicFindings = useMemo(
    () => projection.auditeeFindings.map(createAuditeeFindingProjection),
    [projection.auditeeFindings],
  );
  const publicFinding = useMemo(
    () => projection.finding ? createAuditeeFindingProjection(projection.finding) : null,
    [projection.finding],
  );
  const organizationName =
    publicFinding?.organizationName ??
    projection.auditeeOrganizationName ??
    "Authorized organization";
  const audits = useMemo(
    () => [...new Set(publicFindings.map((finding) => finding.auditId))],
    [publicFindings],
  );
  const statuses = useMemo(
    () => [...new Set(publicFindings.map((finding) => finding.status))],
    [publicFindings],
  );
  const counts = useMemo(() => ({
    all: publicFindings.length,
    open: publicFindings.filter((finding) => matchesGroup(finding, "open")).length,
    "in-progress": publicFindings.filter((finding) => matchesGroup(finding, "in-progress")).length,
    "awaiting-review": publicFindings.filter((finding) => matchesGroup(finding, "awaiting-review")).length,
    closed: publicFindings.filter((finding) => matchesGroup(finding, "closed")).length,
  }), [publicFindings]);
  const visibleFindings = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return publicFindings.filter((finding) =>
      matchesGroup(finding, group) &&
      (auditFilter === "all" || finding.auditId === auditFilter) &&
      (severityFilter === "all" || finding.severity === severityFilter) &&
      (statusFilter === "all" || finding.status === statusFilter) &&
      (!normalized || [
        finding.findingNumber,
        finding.auditId,
        finding.title,
        statusLabel(finding.status),
      ].join(" ").toLowerCase().includes(normalized)),
    );
  }, [auditFilter, group, publicFindings, query, severityFilter, statusFilter]);
  const currentProgress = publicFinding ? lifecycleProgress(publicFinding.status) : null;
  const canSubmitCap =
    projection.finding?.status === "WAITING_FOR_CAP" ||
    projection.finding?.status === "CAP_MORE_INFORMATION_REQUESTED";
  const canSubmitEvidence =
    projection.finding?.status === "EVIDENCE_REQUIRED" ||
    projection.finding?.status === "EVIDENCE_MORE_INFORMATION_REQUESTED";
  const capRevision = latestRevision(projection.capRevisions);

  useEffect(() => {
    if (!canSubmitCap || !capRevision) {
      if (!canSubmitCap) setCap(emptyCap);
      return;
    }
    setCap({
      rootCause: capRevision.rootCause,
      correctiveAction: capRevision.correctiveAction,
      preventiveAction: capRevision.preventiveAction,
      responsiblePerson: capRevision.responsiblePerson,
      targetCompletionDate: capRevision.targetCompletionDate,
      commentToCaa: capRevision.commentToCaa,
    });
  }, [canSubmitCap, capRevision]);

  function updateCap(field: keyof CapDraft, value: string): void {
    setCap((current) => ({ ...current, [field]: value }));
  }

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

  function chooseFile(event: ChangeEvent<HTMLInputElement>): void {
    setSelectedFile(event.target.files?.[0] ?? null);
  }

  function requestCapReview(role: Role): void {
    session?.setActiveRole(role);
    navigate(projection.finding ? `/lead-inspector/cap-review/${projection.finding.id}` : createRoleEntryPath(role));
  }

  function requestEvidenceReview(role: Role): void {
    session?.setActiveRole(role);
    navigate(projection.finding ? `/department-manager/evidence/${projection.finding.id}` : createRoleEntryPath(role));
  }

  function focusResponsePackage(): void {
    document.getElementById("auditee-response-work-package")?.scrollIntoView({ behavior: "smooth" });
  }

  return (
    <WorkspaceShell roleLabel="Auditee" routeLabel="Corrective Actions">
      <div className="auditee-workspace" data-testid="auditee-page">
        <span className="visually-hidden">Service Provider Portal</span>
        <header className="workbench-page-header auditee-page-header">
          <div className="workbench-page-header__main">
            <h1>Corrective Actions (CAP)</h1>
            <p className="workspace-purpose">
              Review exactly what the CAA needs from {organizationName}, the configured Due Date, and the next lifecycle action.
            </p>
          </div>
        </header>

        <p className="auditee-scope-note" data-testid="auditee-scope">
          <span aria-hidden="true">🔒</span> Organization scope: {organizationName}. This portal shows only records explicitly released to your organization.
        </p>
        <CommandError message={error} />

        <section className="auditee-metrics" aria-label="Corrective Action summary">
          {([
            ["all", "Total"],
            ["open", "Open"],
            ["in-progress", "In Progress"],
            ["awaiting-review", "Awaiting Review"],
            ["closed", "Closed"],
          ] as const).map(([key, label]) => (
            <button
              aria-pressed={group === key}
              className={`auditee-metric auditee-metric--${key}${group === key ? " is-active" : ""}`}
              key={key}
              onClick={() => setGroup(key)}
              type="button"
            >
              <span>{label}</span>
              <strong>{counts[key]}</strong>
            </button>
          ))}
        </section>

        <div aria-label="Finding status groups" className="auditee-status-tabs">
          {([
            ["all", "Total"],
            ["open", "Open"],
            ["in-progress", "In Progress"],
            ["awaiting-review", "Awaiting Review"],
            ["closed", "Closed"],
          ] as const).map(([key, label]) => (
            <button
              aria-current={group === key ? "page" : undefined}
              aria-pressed={group === key}
              className={group === key ? "is-active" : ""}
              key={key}
              onClick={() => setGroup(key)}
              type="button"
            >
              {label} <b>{counts[key]}</b>
            </button>
          ))}
        </div>

        <section className="auditee-filters" aria-label="Filter My Findings">
          <label>
            Audit / Inspection
            <select value={auditFilter} onChange={(event) => setAuditFilter(event.target.value)}>
              <option value="all">All Audits / Inspections</option>
              {audits.map((auditId) => <option key={auditId} value={auditId}>{auditId}</option>)}
            </select>
          </label>
          <label>
            Level
            <select value={severityFilter} onChange={(event) => setSeverityFilter(event.target.value)}>
              <option value="all">All Levels</option>
              <option value="LEVEL_1_CRITICAL">Level 1 Critical</option>
              <option value="LEVEL_2_MAJOR">Level 2 Major</option>
              <option value="LEVEL_3_MINOR">Level 3 Minor</option>
              <option value="OBSERVATION">Observation</option>
            </select>
          </label>
          <label>
            Status
            <select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}>
              <option value="all">All Statuses</option>
              {statuses.map((status) => <option key={status} value={status}>{statusLabel(status)}</option>)}
            </select>
          </label>
          <label className="auditee-filter-search">
            Search
            <input
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Finding, audit, status..."
              type="search"
              value={query}
            />
          </label>
        </section>

        <div className="auditee-register-layout">
          <section className="auditee-table-card">
            <div className="auditee-table-scroll">
              <table aria-label="My Findings" className="auditee-findings-table">
                <thead>
                  <tr>
                    <th>Finding ID</th>
                    <th>Audit/Inspection</th>
                    <th>Finding Title</th>
                    <th>Level</th>
                    <th>Status</th>
                    <th>Due Date</th>
                    <th>Progress</th>
                    <th>Action</th>
                  </tr>
                </thead>
                <tbody>
                  {visibleFindings.length ? visibleFindings.map((finding) => {
                    const progress = lifecycleProgress(finding.status);
                    const selected = finding.id === publicFinding?.id;
                    return (
                      <tr className={selected ? "is-selected" : ""} key={finding.id}>
                        <td>
                          <button
                            aria-pressed={selected}
                            className="auditee-record-link"
                            onClick={() => void run(() => actions.selectAuditeeFinding(finding.id))}
                            type="button"
                          >
                            {finding.findingNumber}
                          </button>
                        </td>
                        <td>{finding.auditId}</td>
                        <td>{finding.title}</td>
                        <td>{formatSeverity(finding.severity)}</td>
                        <td>{statusLabel(finding.status)}</td>
                        <td>{formatLocalDate(finding.dueDate)}</td>
                        <td>
                          <span className="auditee-table-progress"><i style={{ width: `${progress.percent}%` }} /></span>
                          {progress.label}
                        </td>
                        <td>
                          <button
                            aria-label={selected ? `${finding.findingNumber} is already selected` : undefined}
                            disabled={selected}
                            onClick={() => void run(() => actions.selectAuditeeFinding(finding.id))}
                            title={selected ? `${finding.findingNumber} is already open in the Auditee Finding dossier.` : undefined}
                            type="button"
                          >
                            {finding.status === "CLOSED" ? "View" : "Select"}
                          </button>
                        </td>
                      </tr>
                    );
                  }) : (
                    <tr><td colSpan={8}><p className="auditee-empty">No Corrective Actions match these filters.</p></td></tr>
                  )}
                </tbody>
              </table>
            </div>
          </section>

          {publicFinding && currentProgress ? (
            <aside className="auditee-dossier" data-testid="auditee-selected-finding">
              <div className="auditee-dossier__head">
                <div><span>Selected Finding</span><h2>{publicFinding.findingNumber}</h2></div>
                <strong data-testid="finding-status">
                  {statusLabel(publicFinding.status)}
                  <span className="visually-hidden"> {publicFinding.status}</span>
                </strong>
              </div>
              <h3>{publicFinding.title}</h3>
              <p>{publicFinding.description}</p>
              <dl>
                <dt>Finding ID</dt><dd>{publicFinding.findingNumber}</dd>
                <dt>Level</dt><dd>{formatSeverity(publicFinding.severity)}</dd>
                <dt>Audit / Inspection</dt><dd>{publicFinding.auditId}</dd>
                <dt>Due Date</dt><dd>{formatLocalDate(publicFinding.dueDate)}</dd>
                <dt>Current owner</dt><dd>{publicFinding.currentOwnerType === "CAA" ? "CAA" : organizationName}</dd>
                <dt>Next action</dt><dd>{publicFinding.nextAction}</dd>
              </dl>
              <div className="auditee-progress">
                <span><b>Lifecycle progress</b><em>{currentProgress.label}</em></span>
                <div><i style={{ width: `${currentProgress.percent}%` }} /></div>
              </div>
              <h3>CAP / Evidence timeline</h3>
              <ol className="auditee-mini-timeline" aria-label="Finding lifecycle">
                <li className="is-done">Finding issued</li>
                <li className={projection.capRevisions.length ? "is-done" : "is-current"}>CAP submitted</li>
                <li className={canSubmitEvidence || projection.evidenceVersions.length ? "is-current" : ""}>Evidence reviewed</li>
                <li className={publicFinding.status === "CLOSED" ? "is-done" : ""}>Finding closed</li>
              </ol>
              <h3>CAA-visible comments</h3>
              {projection.capRevisions.some((revision) => publicReviewComment(revision)) ? (
                <div className="auditee-comment-list">
                  {projection.capRevisions.map((revision) => publicReviewComment(revision) ? (
                    <p key={revision.id}><b>CAP revision {revision.revision}</b>{publicReviewComment(revision)}</p>
                  ) : null)}
                </div>
              ) : <p>No CAA-visible comments.</p>}
              <h3>Evidence versions</h3>
              {projection.evidenceVersions.length ? (
                <div className="auditee-evidence-list">
                  {projection.evidenceVersions.map((version) => (
                    <p key={version.id}><b>v{version.version} · {version.fileName}</b><span>{version.reviewState}</span></p>
                  ))}
                </div>
              ) : <p>No Evidence versions submitted.</p>}
              <button
                aria-controls="auditee-response-work-package"
                className="primary-button auditee-dossier__action"
                onClick={focusResponsePackage}
                type="button"
              >
                {canSubmitCap || canSubmitEvidence ? "Respond" : "View Status"}
              </button>
              <p>CAP acceptance does not close this Finding. Required Evidence must be accepted or an authorized closure must be recorded.</p>
            </aside>
          ) : (
            <aside className="auditee-dossier"><p className="auditee-empty">Select a Corrective Action record.</p></aside>
          )}
        </div>

        {publicFinding ? (
          <section className="auditee-response-work-package" id="auditee-response-work-package">
            <header>
              <div><span>Selected work package</span><h2>CAP and Evidence response</h2></div>
              <strong>{publicFinding.findingNumber}</strong>
            </header>

            {canSubmitCap ? (
              <section className="auditee-response-panel">
                <div className="auditee-section-heading">
                  <div><span>Auditee submission</span><h3>CAP revision {projection.capRevisions.length + 1}</h3></div>
                  <small>CAA review follows submission</small>
                </div>
                <div className="auditee-cap-form">
                  <label className="field-span-2">Root cause<textarea rows={4} value={cap.rootCause} onChange={(event) => updateCap("rootCause", event.target.value)} /></label>
                  <label className="field-span-2">Corrective action<textarea rows={4} value={cap.correctiveAction} onChange={(event) => updateCap("correctiveAction", event.target.value)} /></label>
                  <label className="field-span-2">Preventive action<textarea rows={4} value={cap.preventiveAction} onChange={(event) => updateCap("preventiveAction", event.target.value)} /></label>
                  <label>Responsible person<input value={cap.responsiblePerson} onChange={(event) => updateCap("responsiblePerson", event.target.value)} /></label>
                  <label>Target completion date<input type="date" value={cap.targetCompletionDate} onChange={(event) => updateCap("targetCompletionDate", event.target.value)} /></label>
                  <label className="field-span-2">Comment to CAA<textarea rows={3} value={cap.commentToCaa} onChange={(event) => updateCap("commentToCaa", event.target.value)} /></label>
                </div>
                <div className="auditee-response-actions">
                  <button className="primary-button" disabled={busy} onClick={() => void run(() => actions.submitCap(cap))} type="button">Submit CAP</button>
                </div>
              </section>
            ) : null}

            {projection.capRevisions.length ? (
              <section className="auditee-response-panel">
                <div className="auditee-section-heading">
                  <div><span>Immutable history</span><h3>CAP revision history</h3></div>
                  <strong>{projection.capRevisions.length}</strong>
                </div>
                <div className="auditee-table-scroll">
                  <table aria-label="CAP revision history" className="auditee-revision-table">
                    <thead><tr><th>Revision</th><th>Root cause</th><th>Corrective action</th><th>Preventive action</th><th>Responsible person</th><th>Target</th><th>Comment to CAA</th><th>CAA-visible review</th></tr></thead>
                    <tbody>
                      {projection.capRevisions.map((revision) => (
                        <tr data-testid="auditee-cap-revision-row" key={revision.id}>
                          <td>Revision {revision.revision}</td>
                          <td>{revision.rootCause}</td>
                          <td>{revision.correctiveAction}</td>
                          <td>{revision.preventiveAction}</td>
                          <td>{revision.responsiblePerson}</td>
                          <td>{formatLocalDate(revision.targetCompletionDate)}</td>
                          <td>{revision.commentToCaa}</td>
                          <td>{publicReviewComment(revision) ?? "Awaiting CAA review"}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </section>
            ) : null}

            {(canSubmitEvidence || projection.evidenceVersions.length > 0 || projection.finding?.status === "CLOSED") ? (
              <section className="auditee-response-panel">
                <div className="auditee-section-heading">
                  <div><span>Immutable history</span><h3>Evidence versions</h3></div>
                  <strong data-testid="evidence-version-count">{projection.evidenceVersions.length}</strong>
                </div>
                {projection.evidenceVersions.length ? (
                  <ol className="auditee-version-list">
                    {projection.evidenceVersions.map((version) => (
                      <li key={version.id}><strong>Version {version.version}</strong><span>{version.fileName}</span><span>{version.reviewState}</span></li>
                    ))}
                  </ol>
                ) : <p>No Evidence version submitted yet.</p>}
                {canSubmitEvidence ? (
                  <div className="auditee-upload-panel">
                    <label>
                      Evidence file
                      <input data-testid="evidence-file" disabled={!online} type="file" accept="application/pdf" onChange={chooseFile} />
                    </label>
                    <span data-testid="selected-evidence-file">{selectedFile?.name ?? "No file selected"}</span>
                    {!online ? <small role="status">Official Evidence submission requires an online connection.</small> : null}
                    <button className="primary-button" disabled={busy || !selectedFile || !online} onClick={() => selectedFile && void run(() => actions.submitEvidence(selectedFile))} type="button">Submit Evidence version</button>
                  </div>
                ) : null}
              </section>
            ) : null}

            {projection.finding?.status === "CAP_SUBMITTED" ? (
              <footer className="auditee-handoff">
                <div><span>Current owner</span><strong>CAA CAP Review</strong></div>
                <RoleHandoff identityMode={identityMode} session={handoffSession} targetRole="leadInspector" onRoleRequest={requestCapReview}>Switch to CAA CAP Review</RoleHandoff>
              </footer>
            ) : null}
            {projection.finding?.status === "PENDING_CAA_REVIEW" ? (
              <footer className="auditee-handoff">
                <div><span>Current owner</span><strong>CAA Evidence Review</strong></div>
                <RoleHandoff identityMode={identityMode} session={handoffSession} targetRole="leadInspector" onRoleRequest={requestEvidenceReview}>Switch to Evidence Review</RoleHandoff>
              </footer>
            ) : null}
          </section>
        ) : null}
      </div>
    </WorkspaceShell>
  );
}
