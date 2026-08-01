import type { ExistingChecklistCandidateView } from "../../backend/backend";

export function ChecklistCandidateReview({ candidate }: { candidate: ExistingChecklistCandidateView }) {
  return (
    <section aria-label="Existing checklist candidate review" data-testid="checklist-candidate-review">
      <h2>Existing checklist candidate</h2>
      <p><b>Origin:</b> {candidate.origin}</p>
      <p><b>Currentness:</b> {candidate.candidateCurrentness}</p>
      <ol>{candidate.questions.map((question) => <li key={question.questionId}>{question.wording}</li>)}</ol>
      <p role="note">Candidate wording and history are not regulatory source authority.</p>
    </section>
  );
}
