import type { Role } from "../backend/backend";
import {
  REACT_ROUTE_CONTRACT_BY_AUDIT_ID,
  REACT_ROUTE_CONTRACT_BY_ID,
  type DataBoundary,
  type ReactSurfaceId,
} from "../app/route-contracts";
import legacyScreenSourceJson from "./legacy-screen-source.json";

export type LegacyDisposition = "react-parity";
export type VisualParityMode = "strict-shell" | "content-adapted";
export type { DataBoundary, ReactSurfaceId };

export interface LegacyScreenSourceTuple {
  auditId: string;
  role: string;
  screenName: string;
}

export interface LegacyScreenContract extends LegacyScreenSourceTuple {
  legacyView: string;
  legacyParams: Readonly<Record<string, string>>;
  reactSurfaceId: ReactSurfaceId;
  reactPath: string;
  disposition: LegacyDisposition;
  dataBoundary: DataBoundary;
  parityMode: VisualParityMode;
  productAuthority: readonly string[];
  sourceEvidence: readonly string[];
  referenceScreenshotIds: readonly string[];
  reason: string;
}

export interface ProductScreenCrosswalk {
  outcome: string;
  delivery: "react-parity";
  reactSurfaceIds: readonly ReactSurfaceId[];
  sourceAuditIds: readonly string[];
  disposition: string;
}

const sourceRows = legacyScreenSourceJson as readonly LegacyScreenSourceTuple[];
const defaultProductAuthority = ["AGENTS.md", "docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.md", "docs/product-specs/index.md"] as const;
const defaultSourceEvidence = ["docs/demo-evidence/UI_SCREEN_AUDIT_2026-07-19.md", "index.html", "css/styles.css", "js/app.js", "js/views.js"] as const;

export const LEGACY_SCREEN_SOURCE: readonly LegacyScreenSourceTuple[] = sourceRows;
const retiredRootOracleAuditIDs = new Set(["ui-audit-043"]);
export const LEGACY_SCREEN_MANIFEST: readonly LegacyScreenContract[] = sourceRows
  .filter((row) => !retiredRootOracleAuditIDs.has(row.auditId))
  .map((row) => {
  const route = REACT_ROUTE_CONTRACT_BY_AUDIT_ID.get(row.auditId);
  if (!route) throw new Error(`Missing route contract for ${row.auditId}.`);
  return {
    ...row,
    legacyView: row.screenName,
    legacyParams: { auditId: row.auditId },
    reactSurfaceId: route.id,
    reactPath: route.path,
    disposition: "react-parity",
    dataBoundary: route.dataBoundary,
    parityMode: row.auditId === "ui-audit-001" ? "strict-shell" : "content-adapted",
    productAuthority: defaultProductAuthority,
    sourceEvidence: defaultSourceEvidence,
    referenceScreenshotIds: [row.auditId],
    reason: `The frozen React route contract owns this ${row.screenName} surface.`,
  };
  });

export const PRODUCT_SCREEN_CROSSWALK: readonly ProductScreenCrosswalk[] = [
  { outcome: "Role switch / login", delivery: "react-parity", reactSurfaceIds: ["role-select"], sourceAuditIds: ["ui-audit-001"], disposition: "Delivered as the candidate session/role entry." },
  { outcome: "Manager Dashboard", delivery: "react-parity", reactSurfaceIds: ["manager-home"], sourceAuditIds: ["ui-audit-027"], disposition: "Delivered as Department Manager Dashboard." },
  { outcome: "Inspector Dashboard", delivery: "react-parity", reactSurfaceIds: ["inspector-home"], sourceAuditIds: ["ui-audit-002"], disposition: "Delivered as Inspector My Assignments." },
  { outcome: "Audit Plan Calendar", delivery: "react-parity", reactSurfaceIds: ["audit-plan"], sourceAuditIds: ["ui-audit-028"], disposition: "Delivered as Department Manager Planning." },
  { outcome: "Audit Detail", delivery: "react-parity", reactSurfaceIds: ["audit-detail"], sourceAuditIds: ["ui-audit-007"], disposition: "Delivered as contextual Inspector Audit Detail." },
  { outcome: "Checklist Runner", delivery: "react-parity", reactSurfaceIds: ["checklist-runner"], sourceAuditIds: ["ui-audit-008"], disposition: "Delivered with backend plus field local boundary." },
  { outcome: "Finding Detail with lifecycle stepper", delivery: "react-parity", reactSurfaceIds: ["finding-detail"], sourceAuditIds: ["ui-audit-009"], disposition: "Delivered as the CAA Inspector Finding lifecycle dossier." },
  { outcome: "Auditee My Findings", delivery: "react-parity", reactSurfaceIds: ["auditee-home"], sourceAuditIds: ["ui-audit-066"], disposition: "Delivered inside the Auditee Corrective Actions workspace." },
  { outcome: "CAP Submission Form", delivery: "react-parity", reactSurfaceIds: ["auditee-home"], sourceAuditIds: ["ui-audit-066"], disposition: "Delivered as a versioned CAP submission state." },
  { outcome: "Evidence Upload / Review", delivery: "react-parity", reactSurfaceIds: ["auditee-home", "evidence-review"], sourceAuditIds: ["ui-audit-066", "ui-audit-044"], disposition: "Delivered through Auditee evidence submission and Department Manager evidence review." },
  { outcome: "Closed Finding / Report Preview", delivery: "react-parity", reactSurfaceIds: ["finding-detail", "report-preview"], sourceAuditIds: ["ui-audit-009", "ui-audit-030"], disposition: "Delivered through the Finding lifecycle and report dossier." },
  { outcome: "Admin Checklist Template Preview", delivery: "react-parity", reactSurfaceIds: ["admin-home"], sourceAuditIds: ["ui-audit-076"], disposition: "Delivered as the Admin template preview route." },
  { outcome: "New Inspection Planning Intake", delivery: "react-parity", reactSurfaceIds: ["audit-plan", "new-audit-wizard-1", "new-audit-wizard-2", "new-audit-wizard-3", "new-audit-wizard-4", "new-audit-wizard-5"], sourceAuditIds: ["ui-audit-028", "ui-audit-047", "ui-audit-048", "ui-audit-049", "ui-audit-050", "ui-audit-051"], disposition: "Frozen as Department Manager demo routes pending their owning feature slice." },
];

export function routeRoleForSurface(surfaceId: ReactSurfaceId): Role | null {
  return REACT_ROUTE_CONTRACT_BY_ID.get(surfaceId)?.requiredRole ?? null;
}
