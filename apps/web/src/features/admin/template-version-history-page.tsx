import type { AdminTemplateVersionView } from "../../backend/backend";
import { useParams } from "react-router-dom";
import { AdminError, AdminPage, DisabledAdminAction, useAdminLoad, useAdminWorkspace } from "./admin-workspace-shared";

function exactDiff(version: AdminTemplateVersionView, published: AdminTemplateVersionView): string {
  const added = version.questionIds.filter((id) => !published.questionIds.includes(id)).length;
  const removed = published.questionIds.filter((id) => !version.questionIds.includes(id)).length;
  const commonOrder = version.questionIds.filter((id) => published.questionIds.includes(id)).join("|");
  const publishedOrder = published.questionIds.filter((id) => version.questionIds.includes(id)).join("|");
  return `${added} questions added, ${removed} removed, order ${commonOrder === publishedOrder ? "unchanged" : "changed"}.`;
}

export function TemplateVersionHistoryPage() {
  const backend = useAdminWorkspace();
  const { templateId = "" } = useParams<{ templateId: string }>();
  const mastersLoad = useAdminLoad(() => backend.listTemplateMasters({}), [backend]);
  const masterAvailable = mastersLoad.data?.items.some((item) => item.id === templateId) ?? false;
  const { data, error } = useAdminLoad(
    () => masterAvailable ? backend.getTemplate({ templateId }) : Promise.resolve(null),
    [backend, masterAvailable],
  );
  const published = data?.versions.find((version) => version.id === data.publishedVersionId) ?? null;
  return (
    <AdminPage testId="admin-version-history-page" routeLabel="Version History" title="Version History" description={`Append-only version history for exact template master ${templateId || "selected route"}.`}>
      <AdminError message={mastersLoad.error ?? error} />
      {mastersLoad.data && !masterAvailable ? (
        <section className="admin-record-card" aria-label="Version history availability">
          <h2>No version history is available for {templateId || "this template"}</h2>
          <p>The exact server-owned template master is not present in this environment.</p>
          <DisabledAdminAction
            label="Compare versions"
            reason={`${templateId || "This template"} has no server-owned version history in this environment.`}
          />
        </section>
      ) : null}
      <div className="admin-card-register" role="list" aria-label={`${templateId || "Template"} version history`}>
        {published && data?.versions.map((version) => <article className="admin-record-card" key={version.id} role="listitem"><header><div><b>{version.id}</b><small>Version {version.version}</small></div><span>{version.status}</span></header><dl><div><dt>Creator</dt><dd>{version.creatorSubjectId}</dd></div><div><dt>Current owner</dt><dd>{version.owner}</dd></div><div><dt>Revision</dt><dd>{version.revision}</dd></div><div><dt>Created</dt><dd>{version.createdAt}</dd></div></dl><p><strong>Change reason:</strong> {version.changeReason}</p><p><strong>Exact diff:</strong> {exactDiff(version, published)}</p>{version.status === "DRAFT" ? <DisabledAdminAction label={`Publish ${version.id}`} reason={`${version.id} is DRAFT and Department Manager owns publishing after approval; Admin Preview cannot publish or submit it.`} /> : <DisabledAdminAction label={`Edit ${version.id}`} reason={`${version.id} is PUBLISHED and historical versions are append-only.`} />}</article>)}
      </div>
    </AdminPage>
  );
}
