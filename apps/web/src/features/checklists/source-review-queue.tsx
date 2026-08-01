import type { GovernedSourceReviewQueuePage } from "../../backend/backend";

export function SourceReviewQueue({ page }: { page: GovernedSourceReviewQueuePage }) {
  return <section aria-label="Scoped source review queue" data-testid="source-review-queue"><h2>Source review queue</h2><p>Only current source-owner assignment scope is shown.</p><ul>{page.items.map((item, index) => <li key={index}>{"candidateId" in item ? item.candidateId : item.sourceVersionId}</li>)}</ul></section>;
}
