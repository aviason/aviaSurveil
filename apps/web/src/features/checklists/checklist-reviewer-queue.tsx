import type { GovernedReviewerQueuePage } from "../../backend/backend";

export function ChecklistReviewerQueue({ page }: { page: GovernedReviewerQueuePage }) {
  return <section aria-label="Scoped checklist reviewer queue" data-testid="checklist-reviewer-queue"><h2>Reviewer queue</h2><p>Comments are internal and non-binding; they never become technical approval.</p><ul>{page.items.map((item) => <li key={item.candidateId}>{item.candidateId} · {item.status}</li>)}</ul></section>;
}
