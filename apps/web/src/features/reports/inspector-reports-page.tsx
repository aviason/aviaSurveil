import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { DocumentMetadataView, ReportVersionView } from "../../backend/backend";
import { StatusPill } from "../../ui/workbench/status-pill";
import { CommandError, errorMessage, formatLocalDate, WorkspaceShell } from "../shared/workspace-shell";

function statusTone(status: ReportVersionView["status"]): "success" | "warning" | "neutral" {
  if (status === "LOCKED" || status === "ISSUED") return "success";
  if (status === "RETURNED") return "warning";
  return "neutral";
}

export function InspectorReportsPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("inspector") ?? runtime.backend, [runtime]);
  const [documents, setDocuments] = useState<DocumentMetadataView[]>([]);
  const [reports, setReports] = useState<ReportVersionView[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    void backend.documents.list({}).then(async (documentPage) => {
      const reportDocuments = documentPage.items.filter((item) => item.kind === "REPORT");
      const reportVersions = await Promise.all(
        reportDocuments.map((document) => backend.reports.getVersion({ reportVersionId: document.id })),
      );
      if (!cancelled) {
        setDocuments(reportDocuments);
        setReports(reportVersions);
      }
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend]);

  return <WorkspaceShell roleLabel="CAA Inspector" routeLabel="Inspector Reports">
    <div className="inspector-secondary-page" data-testid="inspector-reports-page">
      <header className="inspector-secondary-head workbench-page-header"><div><h1>Reports</h1><p>Server-owned report versions and generated documents (view only).</p></div></header>
      <CommandError message={error} />
      <h2 className="inspector-section-heading">Available Reports</h2>
      <section className="inspector-work-queue inspector-report-queue" aria-label="Available report versions">
        <div className="inspector-work-queue__head" aria-hidden="true"><span>Status</span><span>Report</span><span>Organization</span><span>Audit</span><span>Version</span><span>Created</span><span>Document</span><span>Open</span></div>
        {reports.length ? reports.map((report) => {
          const document = documents.find((item) => item.id === report.reportVersionId);
          const statusLabel = report.status.replaceAll("_", " ");
          const tone = statusTone(report.status);
          return <article className={`inspector-work-row inspector-work-row--${tone === "success" ? "success" : tone === "warning" ? "warning" : "neutral"}`} data-report-version-id={report.reportVersionId} key={report.reportVersionId}>
            <div className="inspector-work-row__priority"><StatusPill label={statusLabel} tone={tone} /></div>
            <div className="inspector-work-row__item"><h3>{document?.title ?? report.reportId}</h3><p>{report.reportVersionId}</p></div>
            <p className="inspector-work-row__organization">{report.organizationId}</p>
            <p className="inspector-work-row__owner">{report.auditId}</p>
            <p className="inspector-work-row__action">Version {report.version} · revision {report.revision}</p>
            <p aria-label={`Created ${report.reportVersionId}`} className="inspector-work-row__due">{formatLocalDate(document?.createdAt?.slice(0, 10) ?? null)}</p>
            <div className="inspector-work-row__status"><StatusPill label={document?.renderStatus ?? "Metadata available"} tone={document?.renderStatus === "SUCCEEDED" ? "success" : "neutral"} /></div>
            <Link aria-label={`Open ${report.reportVersionId} report`} className="inspector-secondary-button inspector-work-row__open" to={`/inspector/closure-reports/${encodeURIComponent(report.reportVersionId)}`}><span>View report</span></Link>
          </article>;
        }) : <p className="inspector-empty-state">No server-owned report versions are available for this Inspector session.</p>}
      </section>
    </div>
  </WorkspaceShell>;
}
