import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { useBackendForRole } from "../../app/providers";
import type { OrganizationSummary } from "../../backend/backend";
import {
  CommandError,
  errorMessage,
  formatLocalDate,
  WorkspaceShell,
} from "../shared/workspace-shell";

function organizationType(value: string): string {
  if (value === "OPERATOR") return "Operator / Service Provider";
  return value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (match) => match.toUpperCase());
}

export function OrganizationRegistryPage() {
  const backend = useBackendForRole("manager");
  const [organizations, setOrganizations] = useState<OrganizationSummary[]>([]);
  const [selected, setSelected] = useState<OrganizationSummary | null>(null);
  const [error, setError] = useState<string | null>(null);

  function organizationAction(organization: OrganizationSummary) {
    return <Link aria-label={`Open organization ${organization.id}`} to={`/department-manager/organizations/${encodeURIComponent(organization.id)}`}>Open record</Link>;
  }

  useEffect(() => {
    let cancelled = false;
    void backend.organizations.list({ limit: 100 }).then((output) => {
      if (!cancelled) setOrganizations(output.items);
    }).catch((cause) => {
      if (!cancelled) setError(errorMessage(cause));
    });
    return () => {
      cancelled = true;
    };
  }, [backend]);

  return (
    <WorkspaceShell roleLabel="Department Manager" routeLabel="Organization Registry">
      <div className="management-workspace organization-registry-page">
        <header className="management-page-head workbench-page-header">
          <h1>Organizations</h1>
          <p>Regulated organizations under surveillance.</p>
        </header>
        <CommandError message={error} />
        <section className="organization-register management-panel">
          <div className="management-table-scroll">
            <table aria-label="Organizations">
              <thead><tr><th>Organization</th><th>Type</th><th>Open Findings</th><th>Status</th><th>Last Audit</th><th>Next Audit</th><th>Action</th></tr></thead>
              <tbody>
                {organizations.map((organization) => (
                  <tr data-testid="organization-row" key={organization.id}>
                    <td><b>{organization.legalName}</b></td>
                    <td>{organizationType(organization.organizationType)}</td>
                    <td>{organization.openFindingCount}</td>
                    <td><span className={`management-status is-${organization.status.toLowerCase()}`}>{organization.status}</span></td>
                    <td>{formatLocalDate(organization.lastAuditDate)}</td>
                    <td>{formatLocalDate(organization.nextAuditDate)}</td>
                    <td><div className="manager-record-actions"><button aria-label={`Open ${organization.legalName}`} onClick={() => setSelected(organization)} type="button">Inspect summary</button>{organizationAction(organization)}</div></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        <section aria-label="Responsive organization records" className="organization-mobile-records">
          {organizations.map((organization) => (
            <article aria-label={organization.legalName} data-testid="organization-mobile-record" key={organization.id}>
              <header><h2>{organization.legalName}</h2><span>{organization.status}</span></header>
              <dl><div><dt>Type</dt><dd>{organizationType(organization.organizationType)}</dd></div><div><dt>Open Findings</dt><dd>{organization.openFindingCount}</dd></div><div><dt>Next Audit</dt><dd>{formatLocalDate(organization.nextAuditDate)}</dd></div></dl>
              <div className="manager-record-actions"><button aria-label={`Open ${organization.legalName} mobile record`} onClick={() => setSelected(organization)} type="button">Inspect summary</button>{organizationAction(organization)}</div>
            </article>
          ))}
        </section>

        {selected ? (
          <aside className="organization-dossier management-panel" data-testid="organization-dossier">
            <header><div><span>Selected organization</span><h2>{selected.legalName}</h2></div><strong>{selected.status}</strong></header>
            <dl>
              <div><dt>Organization ID</dt><dd>{selected.id}</dd></div>
              <div><dt>Type</dt><dd>{organizationType(selected.organizationType)}</dd></div>
              <div><dt>Open Findings</dt><dd>{selected.openFindingCount}</dd></div>
              <div><dt>Last Audit</dt><dd>{formatLocalDate(selected.lastAuditDate)}</dd></div>
              <div><dt>Next Audit</dt><dd>{formatLocalDate(selected.nextAuditDate)}</dd></div>
              <div><dt>Revision</dt><dd>{selected.revision}</dd></div>
            </dl>
            <p>Read-only authorized organization projection. Editing and organization administration are outside this workspace surface.</p>
          </aside>
        ) : null}
      </div>
    </WorkspaceShell>
  );
}
