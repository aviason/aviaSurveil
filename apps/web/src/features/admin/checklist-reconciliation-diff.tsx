import type { GovernedQuestionReconciliationView } from "../../backend/backend";

export function ChecklistReconciliationDiff({ items }: { items: GovernedQuestionReconciliationView[] }) {
  return (
    <section aria-label="Checklist reconciliation diff" data-testid="checklist-reconciliation-diff">
      <h2>Immutable reconciliation</h2>
      <table><thead><tr><th>Question</th><th>Wording</th><th>Applicability</th><th>Scope</th></tr></thead>
        <tbody>{items.map((item) => <tr key={item.legacyQuestionId}><td>{item.legacyQuestionId}</td><td>{item.wordingChanged ? `${item.legacyWording} → ${item.currentWording}` : "UNCHANGED"}</td><td>{item.applicabilityChanged ? `${item.legacyApplicability} → ${item.currentApplicability}` : "UNCHANGED"}</td><td>{item.scopeChanged ? `${item.legacyScopeClassification} → ${item.currentScopeClassification}` : "UNCHANGED"}</td></tr>)}</tbody>
      </table>
    </section>
  );
}
