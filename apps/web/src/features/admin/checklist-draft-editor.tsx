import type { GovernedCandidateView } from "../../backend/backend";

export function ChecklistDraftEditor({ draft }: { draft: GovernedCandidateView }) {
  const blocked = draft.questions.some((question) => question.regulatoryTrace.state === "SOURCE_MAPPING_REQUIRED");
  return (
    <section aria-label="Governed checklist Draft editor" data-testid="checklist-draft-editor">
      <h2>Governed Draft</h2>
      <p><b>Lineage:</b> {draft.lineage.lineageType}</p>
      <p><b>Status:</b> {draft.status}</p>
      <p><b>Candidate digest:</b> {draft.contentDigest}</p>
      {blocked ? <p role="alert">SOURCE_MAPPING_REQUIRED — technical approval and publication are disabled.</p> : null}
      <ol>{draft.questions.map((question) => <li key={question.questionId}>{question.prompt}</li>)}</ol>
    </section>
  );
}
