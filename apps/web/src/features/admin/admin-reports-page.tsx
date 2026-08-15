import { useEffect, useState } from "react";

import type { AdminReportDefinitionView } from "../../backend/backend";
import { AdminError, AdminPage, DisabledAdminAction, useAdminLoad, useAdminWorkspace } from "./admin-workspace-shared";

export function AdminReportsPage() {
  const backend = useAdminWorkspace();
  const [search, setSearch] = useState("");
  const [selectedId, setSelectedId] = useState("");
  const { data, error } = useAdminLoad(() => backend.listReportDefinitions({ search }), [backend, search]);
  useEffect(() => { if (selectedId && !data?.items.some((item) => item.id === selectedId)) setSelectedId(""); }, [data, selectedId]);
  const selected: AdminReportDefinitionView | null = data?.items.find((item) => item.id === selectedId) ?? null;
  return (
    <AdminPage testId="admin-reports-page" routeLabel="Admin Reports" title="Admin Reports" description="Server-owned report-definition catalog and declared document boundaries.">
      <section className="admin-filter-bar" aria-label="Admin report filters"><label>Search<input aria-label="Search report definitions" onChange={(event) => setSearch(event.target.value)} value={search} /></label><label>Selected report<select aria-label="Selected report definition" onChange={(event) => setSelectedId(event.target.value)} value={selectedId}><option value="">Select a report definition</option>{data?.items.map((report) => <option key={report.id} value={report.id}>{report.id} — {report.title}</option>)}</select></label></section>
      <AdminError message={error} />
      {selected ? <section className="admin-record-card admin-report-preview" aria-label={`Report definition ${selected.id}`}><header><div><b>{selected.title}</b><small>{selected.id}</small></div><span>Server catalog</span></header><p>{selected.description}</p><h2>Declared package fields</h2><ul>{selected.packageFields.map((field) => <li key={field}>{field}</li>)}</ul><DisabledAdminAction label={`Generate ${selected.id}`} reason={selected.actionReason} /><DisabledAdminAction label={`Download ${selected.id}`} reason={`${selected.id} has no server-generated artifact available for this definition.`} /><DisabledAdminAction label={`Publish ${selected.id}`} reason={`${selected.id} has no publication command in the current API contract.`} /></section> : <p>Select a server-owned report definition to inspect it.</p>}
    </AdminPage>
  );
}
