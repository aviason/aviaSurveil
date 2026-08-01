import type { ChecklistImportBatchView } from "../../backend/backend";

export interface ChecklistIntakePanelProps {
  batch: ChecklistImportBatchView | null;
  onReceive?: () => void;
  disabledReason?: string | null;
}

export function ChecklistIntakePanel({ batch, onReceive, disabledReason }: ChecklistIntakePanelProps) {
  return (
    <section aria-label="Candidate-only checklist intake" data-testid="checklist-intake-panel">
      <h2>AGA checklist intake</h2>
      <p>Candidate-only inventory. It does not establish regulatory authority, approval, publication, or execution.</p>
      {batch ? (
        <dl>
          <div><dt>Receipt</dt><dd>{batch.importBatchId}</dd></div>
          <div><dt>Status</dt><dd>{batch.status}</dd></div>
          <div><dt>Files</dt><dd>{batch.fileCount}</dd></div>
          <div><dt>Blocking issues</dt><dd>{batch.blockingIssues.join(" · ") || "None recorded"}</dd></div>
        </dl>
      ) : <p>No archive receipt has been created.</p>}
      <button type="button" onClick={onReceive} disabled={!onReceive || Boolean(disabledReason)} title={disabledReason ?? undefined}>
        Receive candidate-only archive
      </button>
      {disabledReason ? <p role="status">{disabledReason}</p> : null}
    </section>
  );
}
