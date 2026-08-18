/**
 * Technical aggregate identifiers remain the source of truth for commands and
 * routing. This helper intentionally exposes only a short, stable reference
 * where a person needs to distinguish otherwise similar records.
 */
export function recordReference(kind: string, id: string): string {
  const datedReference = id.match(/(\d{4}-\d{3})$/)?.[1];
  if (datedReference) return `${kind} ${datedReference}`;

  const compactId = id.replace(/[^a-z0-9]/gi, "").toUpperCase();
  return `${kind} #${compactId.slice(-5) || "—"}`;
}

export function planningItemLabel(title: string, id: string): string {
  return `${title} · ${recordReference("Plan", id)}`;
}

export function auditLabel(title: string, id: string): string {
  return `${title} · ${recordReference("Audit", id)}`;
}

export function potentialFindingLabel(title: string, id: string): string {
  return `${title} · ${recordReference("Potential finding", id)}`;
}
