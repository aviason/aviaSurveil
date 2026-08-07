import type { BackendPrincipal, GovernedValidationIssue, Role } from "./backend";

export type DemoCapabilityName =
  | "communications" | "calendar" | "profiles" | "teams" | "risk" | "documents"
  | "notifications" | "administration" | "assistantDrafts" | "planningIntake"
  | "packageDrafts" | "auditeeCoordination" | "auditeeReports" | "adminWorkspace"
  | "governedChecklistReview" | "governedChecklistIntake" | "canonicalQuestionReview"
  | "canonicalAuditWorkflow";

export const DEMO_CAPABILITY_PERMISSION_MATRIX: Readonly<Record<Role, readonly DemoCapabilityName[]>> = {
  inspector: ["communications", "calendar", "profiles", "documents", "notifications", "administration", "assistantDrafts", "canonicalAuditWorkflow"],
  leadInspector: ["communications", "calendar", "profiles", "documents", "notifications", "administration", "assistantDrafts", "canonicalAuditWorkflow"],
  manager: ["communications", "calendar", "profiles", "teams", "risk", "documents", "notifications", "administration", "planningIntake", "packageDrafts", "governedChecklistReview", "canonicalQuestionReview", "canonicalAuditWorkflow"],
  finance: ["profiles", "notifications", "administration"],
  gm: ["profiles", "notifications", "administration"],
  executiveDirector: ["profiles", "notifications", "administration"],
  auditee: ["communications", "calendar", "profiles", "documents", "notifications", "administration", "auditeeCoordination", "auditeeReports"],
  admin: ["profiles", "teams", "documents", "notifications", "administration", "adminWorkspace", "governedChecklistIntake", "canonicalQuestionReview"],
};

export function requireDemoCapability(principal: BackendPrincipal, capability: DemoCapabilityName): void {
  if (!DEMO_CAPABILITY_PERMISSION_MATRIX[principal.role].includes(capability)) {
    throw new BackendAuthorizationInvariantError(`${capability} is unavailable to this role in the demo capability boundary.`);
  }
}

export class BackendInvariantError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "BackendInvariantError";
  }
}

export class BackendAuthorizationInvariantError extends BackendInvariantError {
  constructor(message: string) {
    super(message);
    this.name = "BackendAuthorizationInvariantError";
  }
}

export class BackendConflictError extends BackendInvariantError {
  constructor(message: string) {
    super(message);
    this.name = "BackendConflictError";
  }
}

export class GovernedValidationError extends BackendInvariantError {
  constructor(readonly issues: GovernedValidationIssue[]) {
    super(issues.map((issue) => `${issue.fieldPath}: ${issue.message}`).join("; "));
    this.name = "GovernedValidationError";
  }
}

export class OperationIdReuseError extends BackendInvariantError {
  constructor(operationId: string) {
    super(`Operation ID ${operationId} was reused with a different semantic payload.`);
    this.name = "OperationIdReuseError";
  }
}

export function requireNonEmpty(value: string, label: string): string {
  const normalized = value.trim();
  if (!normalized) throw new BackendInvariantError(`${label} is required.`);
  return normalized;
}

export function requireRole(
  principal: BackendPrincipal,
  allowedRoles: readonly Role[],
  message: string,
): void {
  if (!allowedRoles.includes(principal.role)) {
    throw new BackendAuthorizationInvariantError(message);
  }
}

export function requireRevision(actual: number, expected: number | null, label: string): void {
  if (actual !== expected) {
    throw new BackendConflictError(`${label} revision conflict: expected ${expected}, received ${actual}.`);
  }
}

export function cloneValue<T>(value: T): T {
  return structuredClone(value);
}

export function canonicalJson(value: unknown): string {
  function normalize(candidate: unknown): unknown {
    if (Array.isArray(candidate)) return candidate.map(normalize);
    if (candidate && typeof candidate === "object") {
      return Object.fromEntries(
        Object.entries(candidate)
          .sort(([left], [right]) => left.localeCompare(right))
          .map(([key, child]) => [key, normalize(child)]),
      );
    }
    return candidate;
  }
  return JSON.stringify(normalize(value));
}
