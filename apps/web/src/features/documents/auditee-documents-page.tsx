import { useEffect, useMemo, useState } from "react";

import { useApplicationRuntime } from "../../app/providers";
import type { DocumentMetadataView } from "../../backend/backend";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";

function downloadDocument(document: DocumentMetadataView, onDone: (value: string) => void) {
  if (!document.downloadUrl || document.renderStatus !== "SUCCEEDED") {
    throw new Error(`Generated Document ${document.id} is not ready for download.`);
  }
  const anchor = window.document.createElement("a");
  anchor.href = document.downloadUrl;
  anchor.download = document.downloadFileName ?? document.title;
  anchor.rel = "noopener";
  window.document.body.appendChild(anchor);
  anchor.click();
  window.document.body.removeChild(anchor);
  onDone(`${document.downloadFileName ?? document.title} downloaded as the authorized immutable generated Document ${document.documentVersionId ?? document.id}.`);
}

export function AuditeeDocumentsPage() {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.("auditee") ?? runtime.backend, [runtime]);
  const [documents, setDocuments] = useState<DocumentMetadataView[]>([]);
  const [kind, setKind] = useState<"ALL" | DocumentMetadataView["kind"]>("ALL");
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("");
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    if (!backend.documents) return;
    let cancelled = false;
    void backend.documents.list({}).then((output) => {
      if (!cancelled) setDocuments(output.items);
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend]);
  const filtered = documents.filter((document) => (kind === "ALL" || document.kind === kind) && `${document.id} ${document.title}`.toLowerCase().includes(query.toLowerCase()));
  async function openAndDownload(document: DocumentMetadataView) {
    if (!backend.documents) return;
    try {
      const exact = await backend.documents.open({ documentId: document.id });
      downloadDocument(exact, setStatus);
      setError(null);
    } catch (cause) { setError(errorMessage(cause)); }
  }
  return <WorkspaceShell roleLabel="Auditee" routeLabel="Documents">
    <div className="auditee-secondary-page auditee-documents-page" data-testid="auditee-documents-page">
      <header className="auditee-secondary-head workbench-page-header"><div><span>Organization-scoped documents</span><h1>Documents</h1><p>Released Reports and submitted Evidence appear with exact immutable versions and public review results.</p></div></header>
      <p className="auditee-safe-boundary">Private generated documents are downloaded only through short-lived authorized URLs. Rendering does not approve, sign, close, or confer legal validity.</p>
      <CommandError message={error} />
      {status ? <p className="auditee-action-result" role="status">{status}</p> : null}
      <section className="auditee-secondary-filters" aria-label="Document filters"><label>Document type<select value={kind} onChange={(event) => setKind(event.target.value as typeof kind)}><option value="ALL">All documents</option><option value="REPORT">Reports</option><option value="EVIDENCE">Evidence</option></select></label><label>Search exact document<input value={query} onChange={(event) => setQuery(event.target.value)} /></label></section>
      <section className="auditee-document-register" aria-label="Auditee documents"><div className="responsive-table-shell"><table><thead><tr><th>Document</th><th>Type</th><th>Version</th><th>Review result</th><th>Action</th></tr></thead><tbody>{filtered.map((document) => { const available = Boolean(document.downloadUrl) && document.renderStatus === "SUCCEEDED"; const unavailableReason = `Generated document state is ${document.renderStatus?.toLowerCase() ?? "not available"}.`; return <tr key={document.id}><td><b>{document.id}</b><small>{document.title}</small></td><td>{document.kind === "REPORT" ? "Generated Report" : "Evidence"}</td><td>Version {document.version}</td><td>{document.publicReviewResult ?? "Not available"}</td><td>{available ? <button onClick={() => void openAndDownload(document)} type="button">Download {document.id}</button> : <button aria-label={`Download ${document.id} unavailable`} disabled title={unavailableReason} type="button">Download unavailable</button>}</td></tr>; })}</tbody></table></div>{!filtered.length ? <p>No safe documents match this filter.</p> : null}</section>
    </div>
  </WorkspaceShell>;
}
