import { useState } from "react";
import { Link, useParams } from "react-router-dom";

import { AdminError, AdminPage, DisabledAdminAction, useAdminLoad, useAdminWorkspace } from "./admin-workspace-shared";

export function OrganizationMasterDataPage() {
  const backend = useAdminWorkspace();
  const [search, setSearch] = useState("");
  const [organizationType, setOrganizationType] = useState("");
  const [status, setStatus] = useState("");
  const [scope, setScope] = useState("");
  const { data, error } = useAdminLoad(() => backend.listOrganizations({ search, organizationType, status, scope }), [backend, search, organizationType, status, scope]);
  return (
    <AdminPage testId="admin-organization-master-data-page" routeLabel="Organisation Master Data" title="Organisation Master Data" description="Exact typed organization records and contextual detail availability.">
      <section className="admin-filter-bar" aria-label="Organization master-data filters"><label>Search<input aria-label="Search organizations" onChange={(event) => setSearch(event.target.value)} value={search} /></label><label>Type<select aria-label="Organization type" onChange={(event) => setOrganizationType(event.target.value)} value={organizationType}><option value="">All types</option><option value="OPERATOR">Operator</option></select></label><label>Status<select aria-label="Organization status" onChange={(event) => setStatus(event.target.value)} value={status}><option value="">All statuses</option><option value="ACTIVE">Active</option></select></label><label>Scope<select aria-label="Organization scope" onChange={(event) => setScope(event.target.value)} value={scope}><option value="">All scopes</option><option value="CAA oversight">CAA oversight</option></select></label></section>
      <AdminError message={error} />
      <div className="admin-card-register admin-dense-register" role="list" aria-label="Organization master data">
        {data?.items.map((organization) => <article className="admin-record-card" key={organization.id} role="listitem"><header><div><b>{organization.legalName}</b><small>{organization.id}</small></div><span>{organization.status}</span></header><dl><div><dt>Type</dt><dd>{organization.organizationType}</dd></div><div><dt>Scope</dt><dd>{organization.scope}</dd></div></dl>{organization.detailAvailable ? <Link className="admin-action-link" aria-label={`Open ${organization.id}`} to={`/admin/organization-master-data/${organization.id}`}>Open exact record</Link> : <DisabledAdminAction label={`Open ${organization.id}`} reason={organization.disabledReason ?? `${organization.id} has no declared contextual detail route.`} />}</article>)}
      </div>
    </AdminPage>
  );
}

export function AdminOrganizationDetailPage() {
  const backend = useAdminWorkspace();
  const { organizationId } = useParams<{ organizationId: string }>();
  const { data, error } = useAdminLoad(() => organizationId ? backend.getOrganization({ organizationId }) : Promise.resolve(null), [backend, organizationId]);
  return (
    <AdminPage testId="admin-organization-detail-page" routeLabel="Admin Organization Detail" title="Organization Detail" description="Exact contextual master-data record under Organisation Master Data.">
      <AdminError message={error} />
      {data ? <section className="admin-record-card admin-organization-detail" aria-label={`Organization ${data.id}`}><header><div><b>{data.legalName}</b><small>{data.id}</small></div><span>{data.status}</span></header><dl><div><dt>Organization ID</dt><dd>{data.id}</dd></div><div><dt>Legal name</dt><dd>{data.legalName}</dd></div><div><dt>Organization type</dt><dd>{data.organizationType}</dd></div><div><dt>Scope</dt><dd>{data.scope}</dd></div></dl><p>Only typed master-data fields are shown. Contact and certificate values are not configured in demo.</p><DisabledAdminAction label={`Edit ${data.id}`} reason={`${data.id} editing is unavailable because Task 10 declares no organization mutation.`} /><DisabledAdminAction label={`Open risk ${data.id}`} reason={`${data.id} risk actions are Department Manager-owned and unavailable in Admin Preview.`} /></section> : null}
    </AdminPage>
  );
}
