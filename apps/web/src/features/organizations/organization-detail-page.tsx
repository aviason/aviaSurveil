import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { useApplicationRuntime } from "../../app/providers";
import type { AssignmentSummary, FindingView, ManagerDashboardProjection, OrganizationSummary } from "../../backend/backend";
import { CommandError, errorMessage, formatLocalDate, PageHeader, WorkspaceShell } from "../shared/workspace-shell";

interface OrganizationDetail { organization: OrganizationSummary; audits: AssignmentSummary[]; findings: FindingView[]; dashboard: ManagerDashboardProjection }

export function OrganizationDetailPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("manager") ?? runtime.backend, [runtime]);
  const { organizationId } = useParams<{ organizationId: string }>();
  const [detail, setDetail] = useState<OrganizationDetail | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    if (!organizationId) {
      setError("The organization route does not contain a server-owned organization identity.");
      return () => { cancelled = true; };
    }
    void Promise.all([backend.organizations.list({}), backend.assignments.list({}), backend.findings.list({}), backend.dashboards.getManagerProjection({ organizationId })]).then(([organizations, audits, findings, dashboard]) => {
      const organization = organizations.items.find((item) => item.id === organizationId);
      if (!organization) throw new Error(`Organization ${organizationId} was not found.`);
      if (!cancelled) setDetail({ organization, audits: audits.items.filter((audit) => audit.organizationId === organization.id), findings: findings.items.filter((finding) => finding.organizationId === organization.id), dashboard });
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend, organizationId]);

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel="Organization Detail">
      <div className="manager-ops-page" data-testid="manager-organization-detail-page">
        {detail ? <><PageHeader eyebrow="Organization oversight" title={detail.organization.legalName} description="Review exact Audit and Finding history for this organization without crossing organization boundaries." /><CommandError message={error} /><section aria-label="Organization summary" className="manager-ops-metrics"><article><span>Status</span><strong>{detail.organization.status}</strong></article><article><span>Open Findings</span><strong>{detail.dashboard.openFindings}</strong></article><article><span>Overdue</span><strong>{detail.dashboard.overdueFindings}</strong></article><article><span>Next Audit</span><strong>{formatLocalDate(detail.organization.nextAuditDate)}</strong></article></section><div className="manager-ops-layout"><section aria-label="Organization Audit history" className="manager-ops-register"><h2>Audit history</h2>{detail.audits.map((audit) => <article className="manager-ops-card" key={audit.auditId}><h3>{audit.auditId}</h3><p>{audit.title} · {audit.status}</p><p>Next action: {audit.nextAction}</p></article>)}</section><section aria-label="Organization Finding history" className="manager-ops-dossier"><h2>Finding history</h2>{detail.findings.length ? detail.findings.map((finding) => <article key={finding.id}><strong>{finding.id}</strong><p>{finding.status} · {finding.nextAction}</p><Link to={`/department-manager/findings-review?findingId=${encodeURIComponent(finding.id)}`}>Open {finding.id} in Findings Review</Link></article>) : <p>No current Finding is present in this runtime.</p>}</section></div></> : <><PageHeader eyebrow="Organization oversight" title="Organization details" description="Loading exact organization history." /><CommandError message={error} /></>}
      </div>
    </WorkspaceShell>
  );
}
