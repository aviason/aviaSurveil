import type { Role } from "../backend/backend";

export type DataBoundary = "session" | "backend" | "backend+field";
export type RoutePlacement = "primary" | "contextual" | "none";
export type BuildProfileAvailability = "demo" | "http";
export type IconKey =
  | "assignments"
  | "leadReview"
  | "dashboard"
  | "planning"
  | "organizations"
  | "finance"
  | "reports"
  | "templates"
  | "profile"
  | "notifications"
  | "logout"
  | "menu";

/** Canonical successor routes that are deliberately outside the frozen 86-screen legacy parity set. */
export const CANONICAL_QUESTION_REVIEW_PATH = "/department-manager/checklist-management/question-review";

interface RouteSeed {
  auditId: `ui-audit-${string}`;
  id: string;
  path: string;
  requiredRole: Role | null;
  placement: RoutePlacement;
  parentId: string | null;
  label: string;
  iconKey: IconKey;
  dataBoundary: DataBoundary;
}

const route = <const Seed extends RouteSeed>(seed: Seed, order: number) => ({
  ...seed,
  componentKey: seed.id,
  order,
  availableProfiles: ["demo", "http"],
} as const);

const ROUTE_SEEDS = [
  { auditId: "ui-audit-001", id: "role-select", path: "/", requiredRole: null, placement: "none", parentId: null, label: "Role Selection", iconKey: "profile", dataBoundary: "session" },
  { auditId: "ui-audit-002", id: "inspector-home", path: "/inspector/inspector-assignments", requiredRole: "inspector", placement: "primary", parentId: null, label: "My Assignments", iconKey: "assignments", dataBoundary: "backend" },
  { auditId: "ui-audit-003", id: "inspector-findings", path: "/inspector/findings", requiredRole: "inspector", placement: "primary", parentId: null, label: "Findings", iconKey: "assignments", dataBoundary: "backend" },
  { auditId: "ui-audit-004", id: "inspector-messages", path: "/inspector/messages", requiredRole: "inspector", placement: "primary", parentId: null, label: "Messages", iconKey: "notifications", dataBoundary: "backend" },
  { auditId: "ui-audit-005", id: "inspector-calendar", path: "/inspector/calendar", requiredRole: "inspector", placement: "primary", parentId: null, label: "Calendar / Audit Queue", iconKey: "planning", dataBoundary: "backend" },
  { auditId: "ui-audit-006", id: "inspector-reports", path: "/inspector/reports", requiredRole: "inspector", placement: "primary", parentId: null, label: "Reports", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-007", id: "audit-detail", path: "/inspector/audits/AUD-2026-001", requiredRole: "inspector", placement: "contextual", parentId: "inspector-home", label: "Audit Detail", iconKey: "assignments", dataBoundary: "backend" },
  { auditId: "ui-audit-008", id: "checklist-runner", path: "/inspector/audits/AUD-2026-001/checklist", requiredRole: "inspector", placement: "contextual", parentId: "audit-detail", label: "Checklist Runner", iconKey: "assignments", dataBoundary: "backend+field" },
  { auditId: "ui-audit-009", id: "finding-detail", path: "/inspector/findings/FND-CAB-2026-001", requiredRole: "inspector", placement: "contextual", parentId: "inspector-findings", label: "Finding Detail", iconKey: "assignments", dataBoundary: "backend" },
  { auditId: "ui-audit-010", id: "closure-report-preview", path: "/inspector/closure-reports/CR-CAB-2026-001", requiredRole: "inspector", placement: "contextual", parentId: "finding-detail", label: "Closure Report Preview", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-011", id: "inspector-assistant", path: "/inspector/assistant", requiredRole: "inspector", placement: "contextual", parentId: "finding-detail", label: "AI Inspector Assistant", iconKey: "assignments", dataBoundary: "backend" },
  { auditId: "ui-audit-012", id: "inspector-profile", path: "/inspector/profile", requiredRole: "inspector", placement: "contextual", parentId: "inspector-home", label: "Profile", iconKey: "profile", dataBoundary: "backend" },
  { auditId: "ui-audit-013", id: "lead-home", path: "/lead-inspector/lead-review", requiredRole: "leadInspector", placement: "primary", parentId: null, label: "Lead Review", iconKey: "leadReview", dataBoundary: "backend" },
  { auditId: "ui-audit-014", id: "lead-preliminary-reports", path: "/lead-inspector/preliminary-reports", requiredRole: "leadInspector", placement: "primary", parentId: null, label: "Preliminary Reports", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-015", id: "lead-preliminary-report-workflow", path: "/lead-inspector/preliminary-reports/PR-2026-018", requiredRole: "leadInspector", placement: "contextual", parentId: "lead-preliminary-reports", label: "Preliminary Report Workflow", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-016", id: "lead-final-reports", path: "/lead-inspector/final-reports", requiredRole: "leadInspector", placement: "primary", parentId: null, label: "Final Reports", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-017", id: "lead-final-report-readiness", path: "/lead-inspector/final-reports/RPT-CAB-2026-001/readiness", requiredRole: "leadInspector", placement: "contextual", parentId: "lead-final-reports", label: "Final Report Readiness", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-018", id: "lead-prepare-final-report", path: "/lead-inspector/final-reports/RPT-CAB-2026-001/prepare", requiredRole: "leadInspector", placement: "contextual", parentId: "lead-final-reports", label: "Prepare Final Report", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-019", id: "lead-final-report-document", path: "/lead-inspector/final-reports/RPT-CAB-2026-001/document", requiredRole: "leadInspector", placement: "contextual", parentId: "lead-final-reports", label: "Final Report Document", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-020", id: "lead-audit-assignment", path: "/lead-inspector/audits/AUD-2026-001/assignment", requiredRole: "leadInspector", placement: "contextual", parentId: "lead-home", label: "Audit Assignment", iconKey: "assignments", dataBoundary: "backend" },
  { auditId: "ui-audit-021", id: "lead-checklist-question-assignment", path: "/lead-inspector/audits/AUD-2026-001/checklist-questions", requiredRole: "leadInspector", placement: "contextual", parentId: "lead-home", label: "Assign Checklist Questions", iconKey: "templates", dataBoundary: "backend" },
  { auditId: "ui-audit-022", id: "cap-review", path: "/lead-inspector/cap-review/FND-CAB-2026-001", requiredRole: "leadInspector", placement: "contextual", parentId: "lead-home", label: "CAP Review", iconKey: "leadReview", dataBoundary: "backend" },
  { auditId: "ui-audit-023", id: "lead-calendar", path: "/lead-inspector/calendar", requiredRole: "leadInspector", placement: "primary", parentId: null, label: "Calendar", iconKey: "planning", dataBoundary: "backend" },
  { auditId: "ui-audit-024", id: "lead-messages", path: "/lead-inspector/messages", requiredRole: "leadInspector", placement: "primary", parentId: null, label: "Messages", iconKey: "notifications", dataBoundary: "backend" },
  { auditId: "ui-audit-025", id: "lead-analytics-reports", path: "/lead-inspector/analytics-reports", requiredRole: "leadInspector", placement: "primary", parentId: null, label: "Analytics & Reports", iconKey: "dashboard", dataBoundary: "backend" },
  { auditId: "ui-audit-026", id: "lead-settings", path: "/lead-inspector/settings", requiredRole: "leadInspector", placement: "primary", parentId: null, label: "Settings", iconKey: "profile", dataBoundary: "backend" },
  { auditId: "ui-audit-027", id: "manager-home", path: "/department-manager/dashboard", requiredRole: "manager", placement: "primary", parentId: null, label: "Dashboard", iconKey: "dashboard", dataBoundary: "backend" },
  { auditId: "ui-audit-028", id: "audit-plan", path: "/department-manager/audit-plan", requiredRole: "manager", placement: "primary", parentId: null, label: "Audit Plan Calendar", iconKey: "planning", dataBoundary: "backend" },
  { auditId: "ui-audit-029", id: "manager-audits", path: "/department-manager/audits", requiredRole: "manager", placement: "primary", parentId: null, label: "Audits", iconKey: "assignments", dataBoundary: "backend" },
  { auditId: "ui-audit-030", id: "report-preview", path: "/department-manager/reports/RPT-CAB-2026-001-V1", requiredRole: "manager", placement: "primary", parentId: null, label: "Report Preview", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-031", id: "manager-risk-dashboard", path: "/department-manager/risk-dashboard", requiredRole: "manager", placement: "primary", parentId: null, label: "Risk Dashboard", iconKey: "dashboard", dataBoundary: "backend" },
  { auditId: "ui-audit-032", id: "manager-inspection-team", path: "/department-manager/inspection-team", requiredRole: "manager", placement: "primary", parentId: null, label: "Inspection Team", iconKey: "assignments", dataBoundary: "backend" },
  { auditId: "ui-audit-033", id: "manager-findings-review", path: "/department-manager/findings-review", requiredRole: "manager", placement: "primary", parentId: null, label: "Findings Review", iconKey: "leadReview", dataBoundary: "backend" },
  { auditId: "ui-audit-034", id: "manager-cap-monitoring", path: "/department-manager/cap-monitoring", requiredRole: "manager", placement: "primary", parentId: null, label: "CAP Monitoring", iconKey: "leadReview", dataBoundary: "backend" },
  { auditId: "ui-audit-035", id: "manager-checklist-management", path: "/department-manager/checklist-management", requiredRole: "manager", placement: "primary", parentId: null, label: "Checklist Management", iconKey: "templates", dataBoundary: "backend" },
  { auditId: "ui-audit-036", id: "manager-safety-intelligence", path: "/department-manager/safety-intelligence", requiredRole: "manager", placement: "contextual", parentId: "manager-risk-dashboard", label: "Safety Intelligence", iconKey: "dashboard", dataBoundary: "backend" },
  { auditId: "ui-audit-037", id: "organization-risk-profile", path: "/department-manager/organizations/ORG-FLY-NAMIBIA/risk-profile", requiredRole: "manager", placement: "contextual", parentId: "organization-registry", label: "Organization Risk Profile", iconKey: "organizations", dataBoundary: "backend" },
  { auditId: "ui-audit-038", id: "manager-ssp-nasp", path: "/department-manager/ssp-nasp", requiredRole: "manager", placement: "contextual", parentId: "manager-risk-dashboard", label: "SSP / NASP", iconKey: "dashboard", dataBoundary: "backend" },
  { auditId: "ui-audit-039", id: "manager-usoap-readiness", path: "/department-manager/usoap-readiness", requiredRole: "manager", placement: "contextual", parentId: "manager-risk-dashboard", label: "USOAP Readiness", iconKey: "dashboard", dataBoundary: "backend" },
  { auditId: "ui-audit-040", id: "manager-cap-effectiveness", path: "/department-manager/cap-effectiveness", requiredRole: "manager", placement: "contextual", parentId: "manager-cap-monitoring", label: "CAP Effectiveness", iconKey: "leadReview", dataBoundary: "backend" },
  { auditId: "ui-audit-041", id: "organization-registry", path: "/department-manager/organizations", requiredRole: "manager", placement: "primary", parentId: null, label: "Organizations", iconKey: "organizations", dataBoundary: "backend" },
  { auditId: "ui-audit-042", id: "organization-detail", path: "/department-manager/organizations/ORG-FLY-NAMIBIA", requiredRole: "manager", placement: "contextual", parentId: "organization-registry", label: "Organization Detail", iconKey: "organizations", dataBoundary: "backend" },
  { auditId: "ui-audit-043", id: "inspection-package-builder", path: "/department-manager/inspection-package-builder", requiredRole: "manager", placement: "contextual", parentId: "audit-plan", label: "Inspection Package Builder", iconKey: "templates", dataBoundary: "backend" },
  { auditId: "ui-audit-044", id: "evidence-review", path: "/department-manager/evidence/FND-CAB-2026-001", requiredRole: "manager", placement: "contextual", parentId: "manager-findings-review", label: "Inspection Evidence", iconKey: "leadReview", dataBoundary: "backend" },
  { auditId: "ui-audit-045", id: "manager-preliminary-report-review", path: "/department-manager/preliminary-reports/PR-2026-018", requiredRole: "manager", placement: "contextual", parentId: "report-preview", label: "Preliminary Report Review", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-046", id: "manager-cap-closure-review", path: "/department-manager/findings/FND-CAB-2026-001/closure-review", requiredRole: "manager", placement: "contextual", parentId: "manager-cap-monitoring", label: "Department CAP Closure Review", iconKey: "leadReview", dataBoundary: "backend" },
  { auditId: "ui-audit-047", id: "new-audit-wizard-1", path: "/department-manager/new-audit/step-1", requiredRole: "manager", placement: "contextual", parentId: "audit-plan", label: "New Audit Wizard 1", iconKey: "planning", dataBoundary: "backend" },
  { auditId: "ui-audit-048", id: "new-audit-wizard-2", path: "/department-manager/new-audit/step-2", requiredRole: "manager", placement: "contextual", parentId: "audit-plan", label: "New Audit Wizard 2", iconKey: "planning", dataBoundary: "backend" },
  { auditId: "ui-audit-049", id: "new-audit-wizard-3", path: "/department-manager/new-audit/step-3", requiredRole: "manager", placement: "contextual", parentId: "audit-plan", label: "New Audit Wizard 3", iconKey: "planning", dataBoundary: "backend" },
  { auditId: "ui-audit-050", id: "new-audit-wizard-4", path: "/department-manager/new-audit/step-4", requiredRole: "manager", placement: "contextual", parentId: "audit-plan", label: "New Audit Wizard 4", iconKey: "planning", dataBoundary: "backend" },
  { auditId: "ui-audit-051", id: "new-audit-wizard-5", path: "/department-manager/new-audit/step-5", requiredRole: "manager", placement: "contextual", parentId: "audit-plan", label: "New Audit Wizard 5", iconKey: "planning", dataBoundary: "backend" },
  { auditId: "ui-audit-052", id: "gm-home", path: "/general-manager/gm-dashboard", requiredRole: "gm", placement: "primary", parentId: null, label: "General Manager", iconKey: "dashboard", dataBoundary: "backend" },
  { auditId: "ui-audit-053", id: "gm-planning", path: "/general-manager/planning", requiredRole: "gm", placement: "primary", parentId: null, label: "Planning", iconKey: "planning", dataBoundary: "backend" },
  { auditId: "ui-audit-054", id: "gm-report-approvals", path: "/general-manager/report-approvals", requiredRole: "gm", placement: "primary", parentId: null, label: "Report Approvals", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-055", id: "gm-departments", path: "/general-manager/departments", requiredRole: "gm", placement: "primary", parentId: null, label: "Departments", iconKey: "dashboard", dataBoundary: "backend" },
  { auditId: "ui-audit-056", id: "gm-risk-dashboard", path: "/general-manager/risk-dashboard", requiredRole: "gm", placement: "primary", parentId: null, label: "Risk Dashboard", iconKey: "dashboard", dataBoundary: "backend" },
  { auditId: "ui-audit-057", id: "gm-settings", path: "/general-manager/settings", requiredRole: "gm", placement: "primary", parentId: null, label: "Settings", iconKey: "profile", dataBoundary: "backend" },
  { auditId: "ui-audit-058", id: "finance-home", path: "/finance/finance-review", requiredRole: "finance", placement: "primary", parentId: null, label: "Finance Review", iconKey: "finance", dataBoundary: "backend" },
  { auditId: "ui-audit-059", id: "executive-home", path: "/executive-director/executive-dashboard", requiredRole: "executiveDirector", placement: "primary", parentId: null, label: "Executive Dashboard", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-060", id: "executive-planning", path: "/executive-director/planning", requiredRole: "executiveDirector", placement: "primary", parentId: null, label: "Executive Planning", iconKey: "planning", dataBoundary: "backend" },
  { auditId: "ui-audit-061", id: "executive-preliminary-reports", path: "/executive-director/preliminary-reports", requiredRole: "executiveDirector", placement: "primary", parentId: null, label: "Preliminary Reports", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-062", id: "executive-final-reports", path: "/executive-director/final-reports", requiredRole: "executiveDirector", placement: "primary", parentId: null, label: "Executive Final Reports", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-063", id: "executive-report-preview", path: "/executive-director/reports/RPT-CAB-2026-001", requiredRole: "executiveDirector", placement: "contextual", parentId: "executive-final-reports", label: "Executive Report Preview", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-064", id: "executive-notifications", path: "/executive-director/notifications", requiredRole: "executiveDirector", placement: "primary", parentId: null, label: "Executive Notifications", iconKey: "notifications", dataBoundary: "backend" },
  { auditId: "ui-audit-065", id: "executive-settings", path: "/executive-director/settings", requiredRole: "executiveDirector", placement: "primary", parentId: null, label: "Settings", iconKey: "profile", dataBoundary: "backend" },
  { auditId: "ui-audit-066", id: "auditee-home", path: "/auditee/service-provider-cap", requiredRole: "auditee", placement: "primary", parentId: null, label: "Corrective Actions", iconKey: "assignments", dataBoundary: "backend" },
  { auditId: "ui-audit-067", id: "auditee-inspection-coordination", path: "/auditee/inspection-coordination", requiredRole: "auditee", placement: "primary", parentId: null, label: "Inspection Coordination", iconKey: "planning", dataBoundary: "backend" },
  { auditId: "ui-audit-068", id: "auditee-preliminary-reports", path: "/auditee/preliminary-reports", requiredRole: "auditee", placement: "primary", parentId: null, label: "Preliminary Reports", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-069", id: "auditee-final-reports", path: "/auditee/final-reports", requiredRole: "auditee", placement: "primary", parentId: null, label: "Final Reports", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-070", id: "auditee-report-preview", path: "/auditee/reports/RPT-CAB-2026-001", requiredRole: "auditee", placement: "contextual", parentId: "auditee-final-reports", label: "Report Preview", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-071", id: "auditee-messages", path: "/auditee/messages", requiredRole: "auditee", placement: "primary", parentId: null, label: "Messages", iconKey: "notifications", dataBoundary: "backend" },
  { auditId: "ui-audit-072", id: "auditee-documents", path: "/auditee/documents", requiredRole: "auditee", placement: "primary", parentId: null, label: "Documents", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-073", id: "auditee-settings", path: "/auditee/settings", requiredRole: "auditee", placement: "primary", parentId: null, label: "Settings", iconKey: "profile", dataBoundary: "backend" },
  { auditId: "ui-audit-074", id: "admin-regulatory-library", path: "/admin/regulatory-library", requiredRole: "admin", placement: "primary", parentId: null, label: "Regulatory Library", iconKey: "templates", dataBoundary: "backend" },
  { auditId: "ui-audit-075", id: "admin-template-list", path: "/admin/template-library", requiredRole: "admin", placement: "primary", parentId: null, label: "Templates", iconKey: "templates", dataBoundary: "backend" },
  { auditId: "ui-audit-076", id: "admin-home", path: "/admin/templates", requiredRole: "admin", placement: "contextual", parentId: "admin-template-list", label: "Template Preview", iconKey: "templates", dataBoundary: "backend" },
  { auditId: "ui-audit-077", id: "admin-question-bank", path: "/admin/question-bank", requiredRole: "admin", placement: "primary", parentId: null, label: "Question Bank", iconKey: "templates", dataBoundary: "backend" },
  { auditId: "ui-audit-078", id: "admin-checklist-builder", path: "/admin/checklist-builder", requiredRole: "admin", placement: "primary", parentId: null, label: "Checklist Builder", iconKey: "templates", dataBoundary: "backend" },
  { auditId: "ui-audit-079", id: "admin-version-history", path: "/admin/templates/TPL-CABIN-2026/history", requiredRole: "admin", placement: "primary", parentId: null, label: "Version History", iconKey: "templates", dataBoundary: "backend" },
  { auditId: "ui-audit-080", id: "admin-inspection-package-builder", path: "/admin/inspection-package-builder", requiredRole: "admin", placement: "contextual", parentId: "admin-checklist-builder", label: "Inspection Package Builder", iconKey: "templates", dataBoundary: "backend" },
  { auditId: "ui-audit-081", id: "admin-reports", path: "/admin/reports", requiredRole: "admin", placement: "primary", parentId: null, label: "Reports", iconKey: "reports", dataBoundary: "backend" },
  { auditId: "ui-audit-082", id: "admin-users-roles", path: "/admin/users-roles", requiredRole: "admin", placement: "primary", parentId: null, label: "Users / Roles", iconKey: "profile", dataBoundary: "backend" },
  { auditId: "ui-audit-083", id: "admin-configurations", path: "/admin/configurations", requiredRole: "admin", placement: "primary", parentId: null, label: "Configurations", iconKey: "templates", dataBoundary: "backend" },
  { auditId: "ui-audit-084", id: "admin-organization-master-data", path: "/admin/organization-master-data", requiredRole: "admin", placement: "primary", parentId: null, label: "Organisation Master Data", iconKey: "organizations", dataBoundary: "backend" },
  { auditId: "ui-audit-085", id: "admin-organization-detail", path: "/admin/organization-master-data/ORG-FLY-NAMIBIA", requiredRole: "admin", placement: "contextual", parentId: "admin-organization-master-data", label: "Organization Detail", iconKey: "organizations", dataBoundary: "backend" },
  { auditId: "ui-audit-086", id: "admin-audit-log", path: "/admin/audit-log", requiredRole: "admin", placement: "primary", parentId: null, label: "Audit Log", iconKey: "reports", dataBoundary: "backend" },
 ] as const satisfies readonly RouteSeed[];

export type ReactSurfaceId = (typeof ROUTE_SEEDS)[number]["id"];
export type ScreenComponentKey = ReactSurfaceId;
export interface RouteContract extends Omit<RouteSeed, "id" | "parentId"> {
  id: ReactSurfaceId;
  parentId: ReactSurfaceId | null;
  componentKey: ScreenComponentKey;
  order: number;
  availableProfiles: readonly BuildProfileAvailability[];
  blockedProfileReason?: string;
}

export const REACT_ROUTE_CONTRACTS = ROUTE_SEEDS.map(route) as readonly RouteContract[];

export const REACT_ROUTE_CONTRACT_BY_ID = new Map<ReactSurfaceId, RouteContract>(
  REACT_ROUTE_CONTRACTS.map((contract) => [contract.id, contract]),
);

export const REACT_ROUTE_CONTRACT_BY_AUDIT_ID = new Map<string, RouteContract>(
  REACT_ROUTE_CONTRACTS.map((contract) => [contract.auditId, contract]),
);
