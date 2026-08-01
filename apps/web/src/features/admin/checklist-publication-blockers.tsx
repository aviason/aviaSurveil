import type { GovernedValidationIssue } from "../../backend/backend";

export function ChecklistPublicationBlockers({ issues }: { issues: GovernedValidationIssue[] }) {
  return <aside aria-label="Checklist publication blockers" data-testid="checklist-publication-blockers"><h2>Publication blockers</h2>{issues.length ? <ul>{issues.map((issue) => <li key={`${issue.code}-${issue.fieldPath}`}>{issue.code}: {issue.message}</li>)}</ul> : <p>No blockers recorded.</p>}</aside>;
}
