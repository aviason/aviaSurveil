import { createHash } from "node:crypto";
import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { relative, resolve, sep } from "node:path";
import { inflateSync } from "node:zlib";

import { expect, type Page } from "@playwright/test";

import { REACT_ROUTE_CONTRACTS, type ReactSurfaceId } from "../../../src/app/route-contracts";
import type { VisualParityMode } from "../../../src/parity/legacy-screen-manifest";

export const VISUAL_FIXED_TIME_ISO = "2026-06-15T09:00:00.000Z";
export const VISUAL_BASELINE_VERSION = "react-legacy-parity-v1";
export const VISUAL_BASELINE_ROOT = "apps/web/tests/visual-baselines/react-legacy-parity";
export const VISUAL_LEGACY_BASE_URL = "http://127.0.0.1:4173";
export const VISUAL_REACT_BASE_URL = "http://127.0.0.1:4174";
export const VISUAL_MAX_CHANNEL_DELTA = 40;

export type VisualViewportId = "desktop" | "tablet" | "mobile";
export type VisualRegionFilter = "all" | "shell";

export interface Size {
  width: number;
  height: number;
}

export interface Rect extends Size {
  x: number;
  y: number;
}

export interface RegionRect extends Rect {
  id: string;
}

export interface RectMask {
  selector: string;
  rationale: string;
  expectedCount: number;
  rects: Rect[];
}

export interface VisualFrame extends Size {
  data: Uint8ClampedArray;
}

export interface VisualComparisonResult {
  passed: boolean;
  diffPixels: number;
  comparedPixels: number;
  diffPixelRatio: number;
}

export interface VisualRegionComparisonContract {
  region: RegionRect;
  maxDiffPixelRatio: number;
  maxChannelDelta: number;
}

export interface LegacyRouteFixture {
  role: string | null;
  view: string;
  params: Record<string, string>;
}

export interface LegacyVisualState {
  wizardStep?: number;
  preliminaryWorkflow?: boolean;
}

export interface VisualSurfaceFixture {
  id: ReactSurfaceId;
  auditId: string;
  parityMode: VisualParityMode;
  reactPath: string;
  legacy: LegacyRouteFixture;
  stableSelector: string;
  expectedHeading: string;
  expectedRoleText: string | null;
  expectedOwnerText: string | null;
  expectedNextActionText: string | null;
  expectedStatusText: string | null;
  expectedDueDateText: string | null;
  expectedPrimaryActionText: string | null;
  expectedPrivacyAbsence: string[];
  expectedSemanticMarker: string;
  contentAdaptationReason?: string;
  masks: RectMask[];
  legacyState?: LegacyVisualState;
}

export interface BaselineManifestSource {
  commit: string;
  files: Record<string, string>;
  auditDocument: {
    path: string;
    sha256: string;
  };
  packageLockSha256: string;
}

export interface BaselineManifestEnvironment {
  playwrightVersion: string;
  chromiumVersion: string;
  nodeVersion: string;
  platform: NodeJS.Platform | string;
  arch: NodeJS.Architecture | string;
  osRelease: string;
}

export interface BaselineManifestItem {
  auditId: string;
  surfaceId: ReactSurfaceId;
  parityMode: VisualParityMode;
  viewport: VisualViewportId | string;
  viewportSize: Size;
  file: string;
  sha256: string;
  sourceRoute: {
    legacyView: string;
    legacyParams: Record<string, string>;
    reactPath: string;
  };
}

export interface BaselineManifest {
  schemaVersion: 1;
  generatedAt: string;
  baselineVersion: string;
  surfaceCount: number;
  viewportCount: number;
  source: BaselineManifestSource;
  environment: BaselineManifestEnvironment;
  items: BaselineManifestItem[];
}

export interface ValidateBaselineManifestOptions {
  baselineDir: string;
  expectedSource?: BaselineManifestSource;
  expectedEnvironment?: BaselineManifestEnvironment;
  expectedSurfaces?: ReadonlyMap<ReactSurfaceId, VisualSurfaceFixture>;
  expectedViewports?: ReadonlyMap<string, Size>;
}

export const VISUAL_VIEWPORTS = [
  { id: "desktop", width: 1440, height: 900 },
  { id: "tablet", width: 1024, height: 768 },
  { id: "mobile", width: 390, height: 844 },
] as const;

export const VISUAL_VIEWPORT_BY_ID = new Map<string, Size>(
  VISUAL_VIEWPORTS.map((viewport) => [viewport.id, viewport]),
);

export const LEGACY_SOURCE_HASH_FILES = [
  "index.html",
  "css/styles.css",
  "js/app.js",
  "js/views.js",
  "js/data.js",
] as const;

export const LEGACY_AUDIT_DOCUMENT = "docs/demo-evidence/UI_SCREEN_AUDIT_2026-07-19.md";
export const APP_PACKAGE_LOCK = "apps/web/package-lock.json";

const CORE_VISUAL_SURFACES: readonly VisualSurfaceFixture[] = [
  {
    id: "role-select",
    auditId: "ui-audit-001",
    parityMode: "strict-shell",
    reactPath: "/",
    legacy: { role: null, view: "login", params: {} },
    stableSelector: ".login-selector",
    expectedHeading: "AviaSurveil360",
    expectedSemanticMarker: "Choose your workspace",
    expectedRoleText: null,
    expectedOwnerText: null,
    expectedNextActionText: null,
    expectedStatusText: null,
    expectedDueDateText: null,
    expectedPrimaryActionText: "Explore the clickable demo",
    expectedPrivacyAbsence: ["Internal CAA Note", "Inspector workload", "enforcement deliberation"],
    masks: [],
  },
  {
    id: "inspector-home",
    auditId: "ui-audit-002",
    parityMode: "content-adapted",
    reactPath: "/inspector/inspector-assignments",
    legacy: { role: "inspector", view: "inspector-assignments", params: {} },
    stableSelector: "main.content",
    expectedHeading: "My Assignments",
    expectedSemanticMarker: "PR-2026-018",
    expectedRoleText: "Inspector",
    expectedOwnerText: "Aylin Sezer",
    expectedNextActionText: "Start",
    expectedStatusText: "In Progress",
    expectedDueDateText: "15 Jun 2026",
    expectedPrimaryActionText: "Open",
    expectedPrivacyAbsence: ["Internal CAA Note", "enforcement deliberation"],
    contentAdaptationReason: "React may bind the same assignment queue to backend-shaped cards.",
    masks: [],
  },
  {
    id: "lead-home",
    auditId: "ui-audit-013",
    parityMode: "content-adapted",
    reactPath: "/lead-inspector/lead-review",
    legacy: { role: "leadInspector", view: "lead-review", params: {} },
    stableSelector: "main.content",
    expectedHeading: "Assigned Audits",
    expectedSemanticMarker: "PF-2026-001",
    expectedRoleText: "Lead Inspector",
    expectedOwnerText: "Caner Yildiz",
    expectedNextActionText: "Convert to Finding",
    expectedStatusText: "Pending lead review",
    expectedDueDateText: "15 Jul 2026",
    expectedPrimaryActionText: "Convert to Finding",
    expectedPrivacyAbsence: ["SkyCargo Air", "enforcement deliberation"],
    contentAdaptationReason: "Potential Finding review uses the accepted lead queue pattern.",
    masks: [],
  },
  {
    id: "manager-home",
    auditId: "ui-audit-027",
    parityMode: "content-adapted",
    reactPath: "/department-manager/dashboard",
    legacy: { role: "manager", view: "dashboard", params: {} },
    stableSelector: "main.content",
    expectedHeading: "Dashboard",
    expectedSemanticMarker: "Open Findings",
    expectedRoleText: "Department Manager",
    expectedOwnerText: "Mehmet Kaya",
    expectedNextActionText: "overdue",
    expectedStatusText: "Open",
    expectedDueDateText: "Due",
    expectedPrimaryActionText: "Open",
    expectedPrivacyAbsence: ["enforcement deliberation"],
    contentAdaptationReason: "Manager dashboard may adapt KPI content while preserving workbench hierarchy.",
    masks: [],
  },
  {
    id: "gm-home",
    auditId: "ui-audit-052",
    parityMode: "content-adapted",
    reactPath: "/general-manager/gm-dashboard",
    legacy: { role: "gm", view: "gm-dashboard", params: {} },
    stableSelector: "main.content",
    expectedHeading: "General Manager Dashboard",
    expectedSemanticMarker: "Department Overview",
    expectedRoleText: "General Manager",
    expectedOwnerText: "Okan Demir",
    expectedNextActionText: "review",
    expectedStatusText: "Pending",
    expectedDueDateText: "Due",
    expectedPrimaryActionText: "Open",
    expectedPrivacyAbsence: ["Internal CAA Note", "enforcement deliberation"],
    contentAdaptationReason: "Authority dashboard content can adapt to the current candidate planning read model.",
    masks: [],
  },
  {
    id: "finance-home",
    auditId: "ui-audit-058",
    parityMode: "content-adapted",
    reactPath: "/finance/finance-review",
    legacy: { role: "finance", view: "finance-review", params: {} },
    stableSelector: "main.content",
    expectedHeading: "Finance Review",
    expectedSemanticMarker: "PLAN-2026-Q3-CABIN",
    expectedRoleText: "Finance Review",
    expectedOwnerText: "Derya Acar",
    expectedNextActionText: "budget",
    expectedStatusText: "Finance",
    expectedDueDateText: "Target",
    expectedPrimaryActionText: "Approve",
    expectedPrivacyAbsence: ["Internal CAA Note", "enforcement deliberation"],
    contentAdaptationReason: "Finance review preserves approval composition with candidate planning state.",
    masks: [],
  },
  {
    id: "executive-home",
    auditId: "ui-audit-059",
    parityMode: "content-adapted",
    reactPath: "/executive-director/executive-dashboard",
    legacy: { role: "executiveDirector", view: "executive-dashboard", params: {} },
    stableSelector: "main.content",
    expectedHeading: "Executive Director Dashboard",
    expectedSemanticMarker: "FR-2026-022",
    expectedRoleText: "Executive Director",
    expectedOwnerText: "Ufuk Aslan",
    expectedNextActionText: "approval",
    expectedStatusText: "Pending",
    expectedDueDateText: "Due",
    expectedPrimaryActionText: "Approve",
    expectedPrivacyAbsence: ["Internal CAA Note", "Inspector workload"],
    contentAdaptationReason: "Executive dashboard preserves final decision hierarchy with candidate data.",
    masks: [],
  },
  {
    id: "auditee-home",
    auditId: "ui-audit-066",
    parityMode: "content-adapted",
    reactPath: "/auditee/service-provider-cap",
    legacy: { role: "auditee", view: "service-provider-cap", params: {} },
    stableSelector: "main.content",
    expectedHeading: "Corrective Actions",
    expectedSemanticMarker: "CAB-2026-011",
    expectedRoleText: "Service Provider Portal",
    expectedOwnerText: "Fly Namibia",
    expectedNextActionText: "Submit",
    expectedStatusText: "CAP",
    expectedDueDateText: "Due Date",
    expectedPrimaryActionText: "Submit",
    expectedPrivacyAbsence: ["Internal CAA Note", "SkyCargo Air", "Inspector workload"],
    contentAdaptationReason: "Auditee route combines My Findings, CAP, and Evidence request states.",
    masks: [],
  },
  {
    id: "admin-home",
    auditId: "ui-audit-076",
    parityMode: "content-adapted",
    reactPath: "/admin/templates",
    legacy: { role: "admin", view: "template-preview", params: { id: "TPL-CABIN-2026" } },
    stableSelector: "main.content",
    expectedHeading: "Template Preview",
    expectedSemanticMarker: "TPL-CABIN-2026",
    expectedRoleText: "Administration",
    expectedOwnerText: "Cabin Safety",
    expectedNextActionText: "Expected evidence",
    expectedStatusText: "Published",
    expectedDueDateText: null,
    expectedPrimaryActionText: "Back to templates",
    expectedPrivacyAbsence: ["Inspector workload", "enforcement deliberation"],
    contentAdaptationReason: "Admin home renders the approved published-template preview slice.",
    masks: [],
  },
  {
    id: "audit-detail",
    auditId: "ui-audit-007",
    parityMode: "content-adapted",
    reactPath: "/inspector/audits/AUD-2026-001",
    legacy: { role: "inspector", view: "audit-detail", params: { auditId: "AUD-2026-001" } },
    stableSelector: "main.content",
    expectedHeading: "2026 Cabin Inspection",
    expectedSemanticMarker: "INS-2026-001",
    expectedRoleText: "Inspector",
    expectedOwnerText: "Aylin Sezer",
    expectedNextActionText: "Run",
    expectedStatusText: "In Progress",
    expectedDueDateText: "15 Jun 2026",
    expectedPrimaryActionText: "Run",
    expectedPrivacyAbsence: ["Internal CAA Note", "enforcement deliberation"],
    contentAdaptationReason: "Audit detail keeps the same inspection context and action path.",
    masks: [],
  },
  {
    id: "checklist-runner",
    auditId: "ui-audit-008",
    parityMode: "content-adapted",
    reactPath: "/inspector/audits/AUD-2026-001/checklist",
    legacy: {
      role: "inspector",
      view: "checklist",
      params: { auditId: "AUD-2026-001", questionId: "cab-em-eq-pbe" },
    },
    stableSelector: "main.content",
    expectedHeading: "Cabin Inspection",
    expectedSemanticMarker: "CAB EMEQ PBE",
    expectedRoleText: "Inspector",
    expectedOwnerText: "Aylin Sezer",
    expectedNextActionText: "Submit to Lead Inspector",
    expectedStatusText: "In Progress",
    expectedDueDateText: "15 Jun 2026",
    expectedPrimaryActionText: "Submit",
    expectedPrivacyAbsence: ["Internal CAA Note", "enforcement deliberation"],
    contentAdaptationReason: "Checklist runner uses candidate field state while retaining row/action composition.",
    masks: [],
  },
  {
    id: "organization-registry",
    auditId: "ui-audit-041",
    parityMode: "content-adapted",
    reactPath: "/department-manager/organizations",
    legacy: { role: "manager", view: "organizations", params: {} },
    stableSelector: "main.content",
    expectedHeading: "Organizations",
    expectedSemanticMarker: "Fly Namibia",
    expectedRoleText: "Department Manager",
    expectedOwnerText: "Fly Namibia",
    expectedNextActionText: "Open",
    expectedStatusText: "Findings",
    expectedDueDateText: null,
    expectedPrimaryActionText: "Open",
    expectedPrivacyAbsence: ["Internal CAA Note", "enforcement deliberation"],
    contentAdaptationReason: "Registry can adapt row counts while preserving organization workbench layout.",
    masks: [],
  },
  {
    id: "audit-plan",
    auditId: "ui-audit-028",
    parityMode: "content-adapted",
    reactPath: "/department-manager/audit-plan",
    legacy: { role: "manager", view: "planning", params: {} },
    stableSelector: "main.content",
    expectedHeading: "Planning",
    expectedSemanticMarker: "PLAN-2026-Q3-CABIN",
    expectedRoleText: "Department Manager",
    expectedOwnerText: "Department Manager",
    expectedNextActionText: "Finance",
    expectedStatusText: "Planning",
    expectedDueDateText: "Target",
    expectedPrimaryActionText: "Open",
    expectedPrivacyAbsence: ["Internal CAA Note", "enforcement deliberation"],
    contentAdaptationReason: "Planning maps to the candidate read-only Audit Plan Calendar state.",
    masks: [],
  },
  {
    id: "finding-detail",
    auditId: "ui-audit-009",
    parityMode: "content-adapted",
    reactPath: "/inspector/findings/FND-CAB-2026-001",
    legacy: { role: "inspector", view: "finding", params: { findingId: "CAB-2026-011" } },
    stableSelector: "main.content",
    expectedHeading: "Finding CAB-2026-011",
    expectedSemanticMarker: "CAB-2026-011",
    expectedRoleText: "Inspector",
    expectedOwnerText: "Fly Namibia",
    expectedNextActionText: "Review CAP",
    expectedStatusText: "CAP Submitted",
    expectedDueDateText: "15 Jul 2026",
    expectedPrimaryActionText: "Review CAP",
    expectedPrivacyAbsence: ["SkyCargo Air", "enforcement deliberation"],
    contentAdaptationReason: "A seeded legacy finding supplies the shared lifecycle dossier without live mutation.",
    masks: [],
  },
  {
    id: "cap-review",
    auditId: "ui-audit-022",
    parityMode: "content-adapted",
    reactPath: "/lead-inspector/cap-review/FND-CAB-2026-001",
    legacy: { role: "leadInspector", view: "cap-review-detail", params: { findingId: "CAB-2026-011" } },
    stableSelector: "main.content",
    expectedHeading: "CAP Review",
    expectedSemanticMarker: "CAB-2026-011",
    expectedRoleText: "Lead Inspector",
    expectedOwnerText: "Fly Namibia",
    expectedNextActionText: "Accept",
    expectedStatusText: "CAP Submitted",
    expectedDueDateText: "15 Jul 2026",
    expectedPrimaryActionText: "Accept",
    expectedPrivacyAbsence: ["SkyCargo Air", "enforcement deliberation"],
    contentAdaptationReason: "CAP review detail uses a seeded submitted CAP while preserving immutable revision semantics.",
    masks: [],
  },
  {
    id: "evidence-review",
    auditId: "ui-audit-044",
    parityMode: "content-adapted",
    reactPath: "/department-manager/evidence/FND-CAB-2026-001",
    legacy: { role: "manager", view: "findings", params: { filter: "evreview" } },
    stableSelector: "main.content",
    expectedHeading: "Findings",
    expectedSemanticMarker: "RAMP-2026-005",
    expectedRoleText: "Department Manager",
    expectedOwnerText: "Fly Namibia",
    expectedNextActionText: "Review evidence",
    expectedStatusText: "Evidence",
    expectedDueDateText: "Due",
    expectedPrimaryActionText: "Review",
    expectedPrivacyAbsence: ["SkyCargo Air", "enforcement deliberation"],
    contentAdaptationReason: "Evidence review maps to the accepted evidence waiting-review queue pattern.",
    masks: [],
  },
  {
    id: "report-preview",
    auditId: "ui-audit-030",
    parityMode: "content-adapted",
    reactPath: "/department-manager/reports/RPT-CAB-2026-001-V1",
    legacy: { role: "manager", view: "reports-approval", params: { reportId: "PR-2026-018" } },
    stableSelector: "main.content",
    expectedHeading: "Reports Approval",
    expectedSemanticMarker: "PR-2026-018",
    expectedRoleText: "Department Manager",
    expectedOwnerText: "Fly Namibia",
    expectedNextActionText: "Review",
    expectedStatusText: "Report",
    expectedDueDateText: null,
    expectedPrimaryActionText: "Preview",
    expectedPrivacyAbsence: ["SkyCargo Air", "Inspector workload"],
    contentAdaptationReason: "Report preview uses the accepted immutable report dossier composition.",
    masks: [],
  },
] as const;

/**
 * The root demo has no URL router. These are the reviewed in-memory states used
 * to drive its immutable UI-audit oracle. Keep the role and view together: a
 * route may never borrow another role's legacy capture.
 */
interface ReviewedLegacyAssertion {
  expectedHeading: string;
  expectedSemanticMarker: string;
  legacyState?: LegacyVisualState;
}

const LEGACY_ROUTE_FIXTURES: Partial<Record<ReactSurfaceId, LegacyRouteFixture>> = {
  "inspector-findings": { role: "inspector", view: "findings", params: {} },
  "inspector-messages": { role: "inspector", view: "messages", params: {} },
  "inspector-calendar": { role: "inspector", view: "calendar", params: {} },
  "inspector-reports": { role: "inspector", view: "reports", params: {} },
  "closure-report-preview": { role: "inspector", view: "report", params: { findingId: "CAB-2026-011" } },
  "inspector-assistant": { role: "inspector", view: "ai-assistant", params: { sourceView: "finding", findingId: "CAB-2026-011" } },
  "inspector-profile": { role: "inspector", view: "profile", params: {} },
  "lead-preliminary-reports": { role: "leadInspector", view: "audit-reports", params: { filter: "preliminary" } },
  "lead-preliminary-report-workflow": { role: "leadInspector", view: "audit-reports", params: { filter: "preliminary", reportId: "PR-2026-018" } },
  "lead-final-reports": { role: "leadInspector", view: "audit-reports", params: { filter: "final" } },
  "lead-final-report-readiness": { role: "leadInspector", view: "audit-reports", params: { filter: "final", finalReportId: "FR-2026-018" } },
  "lead-prepare-final-report": { role: "leadInspector", view: "final-report-prepare", params: { reportId: "FR-2026-018" } },
  "lead-final-report-document": { role: "leadInspector", view: "final-report-view", params: { reportId: "FR-2026-018" } },
  "lead-audit-assignment": { role: "leadInspector", view: "lead-assignment", params: { auditId: "AUD-2026-001" } },
  "lead-checklist-question-assignment": { role: "leadInspector", view: "lead-assignment-questions", params: { auditId: "AUD-2026-001" } },
  "lead-calendar": { role: "leadInspector", view: "calendar", params: {} },
  "lead-messages": { role: "leadInspector", view: "messages", params: {} },
  "lead-analytics-reports": { role: "leadInspector", view: "safety-intelligence", params: {} },
  "lead-settings": { role: "leadInspector", view: "settings", params: {} },
  "manager-audits": { role: "manager", view: "calendar", params: {} },
  "manager-risk-dashboard": { role: "manager", view: "manager-risk", params: {} },
  "manager-inspection-team": { role: "manager", view: "inspection-team", params: {} },
  "manager-findings-review": { role: "manager", view: "findings-review", params: {} },
  "manager-cap-monitoring": { role: "manager", view: "cap-monitoring", params: {} },
  "manager-checklist-management": { role: "manager", view: "manager-checklists", params: {} },
  "manager-safety-intelligence": { role: "manager", view: "safety-intelligence", params: {} },
  "organization-risk-profile": { role: "manager", view: "org-risk", params: { orgId: "ORG-XYZ" } },
  "manager-ssp-nasp": { role: "manager", view: "ssp-nasp", params: {} },
  "manager-usoap-readiness": { role: "manager", view: "usoap-readiness", params: {} },
  "manager-cap-effectiveness": { role: "manager", view: "cap-effectiveness", params: {} },
  "organization-detail": { role: "manager", view: "org-detail", params: { orgId: "ORG-XYZ" } },
  "inspection-package-builder": { role: "manager", view: "package-builder", params: {} },
  "manager-preliminary-report-review": { role: "manager", view: "audit-reports", params: { filter: "preliminary" } },
  "manager-cap-closure-review": { role: "manager", view: "unit-manager-review", params: { findingId: "CAB-2026-011" } },
  "new-audit-wizard-1": { role: "manager", view: "wizard", params: { step: "1" } },
  "new-audit-wizard-2": { role: "manager", view: "wizard", params: { step: "2" } },
  "new-audit-wizard-3": { role: "manager", view: "wizard", params: { step: "3" } },
  "new-audit-wizard-4": { role: "manager", view: "wizard", params: { step: "4" } },
  "new-audit-wizard-5": { role: "manager", view: "wizard", params: { step: "5" } },
  "gm-planning": { role: "gm", view: "planning", params: {} },
  "gm-report-approvals": { role: "gm", view: "gm-report-approvals", params: {} },
  "gm-departments": { role: "gm", view: "gm-departments", params: {} },
  "gm-risk-dashboard": { role: "gm", view: "gm-risk", params: {} },
  "gm-settings": { role: "gm", view: "settings", params: {} },
  "executive-planning": { role: "executiveDirector", view: "executive-planning", params: {} },
  "executive-preliminary-reports": { role: "executiveDirector", view: "executive-preliminary-reports", params: {} },
  "executive-final-reports": { role: "executiveDirector", view: "executive-final-reports", params: {} },
  "executive-report-preview": { role: "executiveDirector", view: "executive-report-preview", params: { reportId: "FR-2026-022" } },
  "executive-notifications": { role: "executiveDirector", view: "executive-notifications", params: {} },
  "executive-settings": { role: "executiveDirector", view: "settings", params: {} },
  "auditee-inspection-coordination": { role: "auditee", view: "service-provider-inspection-coordination", params: {} },
  "auditee-preliminary-reports": { role: "auditee", view: "service-provider-preliminary-reports", params: {} },
  "auditee-final-reports": { role: "auditee", view: "service-provider-final-reports", params: {} },
  "auditee-report-preview": { role: "auditee", view: "service-provider-report-preview", params: { reportId: "FR-2025-009" } },
  "auditee-messages": { role: "auditee", view: "messages", params: {} },
  "auditee-documents": { role: "auditee", view: "reports", params: { filter: "documents" } },
  "auditee-settings": { role: "auditee", view: "settings", params: {} },
  "admin-regulatory-library": { role: "admin", view: "regulatory-library", params: {} },
  "admin-template-list": { role: "admin", view: "templates", params: {} },
  "admin-question-bank": { role: "admin", view: "question-bank", params: {} },
  "admin-checklist-builder": { role: "admin", view: "checklist-builder", params: {} },
  "admin-version-history": { role: "admin", view: "checklist-versions", params: {} },
  "admin-inspection-package-builder": { role: "admin", view: "package-builder", params: {} },
  "admin-reports": { role: "admin", view: "reports", params: {} },
  "admin-users-roles": { role: "admin", view: "users", params: {} },
  "admin-configurations": { role: "admin", view: "settings", params: {} },
  "admin-organization-master-data": { role: "admin", view: "organizations", params: {} },
  "admin-organization-detail": { role: "admin", view: "org-detail", params: { orgId: "ORG-XYZ" } },
  "admin-audit-log": { role: "admin", view: "auditlog", params: {} },
};

const REVIEWED_SOURCE_ASSERTIONS: Partial<Record<ReactSurfaceId, ReviewedLegacyAssertion>> = {
  "inspector-findings": { expectedHeading: "Findings", expectedSemanticMarker: "CAB-2026-011" },
  "inspector-messages": { expectedHeading: "Messages from the CAA", expectedSemanticMarker: "In-app notifications" },
  "inspector-calendar": { expectedHeading: "Audit Work Queue", expectedSemanticMarker: "Fly Namibia · Cabin Inspection" },
  "inspector-reports": { expectedHeading: "Reports", expectedSemanticMarker: "Past closure reports" },
  "closure-report-preview": { expectedHeading: "Finding Report", expectedSemanticMarker: "CAB-2026-011" },
  "inspector-assistant": { expectedHeading: "AI Inspector Assistant Panel", expectedSemanticMarker: "CAB-2026-011 · Emergency equipment serviceability record incomplete" },
  "inspector-profile": { expectedHeading: "Profile", expectedSemanticMarker: "Aylin Sezer" },
  "lead-preliminary-reports": { expectedHeading: "Preliminary Reports", expectedSemanticMarker: "PR-2026-018" },
  "lead-preliminary-report-workflow": { expectedHeading: "Preliminary Report", expectedSemanticMarker: "Inspection Overview", legacyState: { preliminaryWorkflow: true } },
  "lead-final-reports": { expectedHeading: "Final Reports", expectedSemanticMarker: "FR-2026-018" },
  "lead-final-report-readiness": { expectedHeading: "Final Report Preparation", expectedSemanticMarker: "FR-2026-018" },
  "lead-prepare-final-report": { expectedHeading: "Report Content", expectedSemanticMarker: "Report Progress" },
  "lead-final-report-document": { expectedHeading: "Final Report", expectedSemanticMarker: "Final Report" },
  "lead-audit-assignment": { expectedHeading: "Cabin Inspection", expectedSemanticMarker: "Assigned Audits" },
  "lead-checklist-question-assignment": { expectedHeading: "Assign Checklist Questions", expectedSemanticMarker: "cab-em-eq-pbe" },
  "lead-calendar": { expectedHeading: "Audit Work Queue", expectedSemanticMarker: "Fly Namibia · Cabin Inspection" },
  "lead-messages": { expectedHeading: "Messages from the CAA", expectedSemanticMarker: "In-app notifications" },
  "lead-analytics-reports": { expectedHeading: "Safety Intelligence Dashboard", expectedSemanticMarker: "Management attention" },
  "lead-settings": { expectedHeading: "Settings", expectedSemanticMarker: "Configured rules" },
  "manager-audits": { expectedHeading: "Audit Work Queue", expectedSemanticMarker: "Fly Namibia · Cabin Inspection" },
  "manager-risk-dashboard": { expectedHeading: "Risk Dashboard", expectedSemanticMarker: "Risk" },
  "manager-inspection-team": { expectedHeading: "Inspection Team", expectedSemanticMarker: "Inspection Team" },
  "manager-findings-review": { expectedHeading: "Findings Review", expectedSemanticMarker: "CAB-2026-011" },
  "manager-cap-monitoring": { expectedHeading: "CAP Monitoring", expectedSemanticMarker: "CAP" },
  "manager-checklist-management": { expectedHeading: "Checklist Management", expectedSemanticMarker: "Configured" },
  "manager-safety-intelligence": { expectedHeading: "Safety Intelligence Dashboard", expectedSemanticMarker: "Management attention" },
  "organization-risk-profile": { expectedHeading: "Organization Risk Profile", expectedSemanticMarker: "Fly Namibia" },
  "manager-ssp-nasp": { expectedHeading: "SSP/NASP Management Dashboard", expectedSemanticMarker: "NASP-ACT-001" },
  "manager-usoap-readiness": { expectedHeading: "USOAP Readiness Workspace", expectedSemanticMarker: "Missing evidence" },
  "manager-cap-effectiveness": { expectedHeading: "CAP Effectiveness", expectedSemanticMarker: "Effectiveness" },
  "organization-detail": { expectedHeading: "Fly Namibia", expectedSemanticMarker: "Organization" },
  "inspection-package-builder": { expectedHeading: "Dynamic Inspection Package Builder", expectedSemanticMarker: "Package Draft" },
  "manager-preliminary-report-review": { expectedHeading: "Preliminary Report", expectedSemanticMarker: "Preliminary Report" },
  "manager-cap-closure-review": { expectedHeading: "Department Manager Review", expectedSemanticMarker: "CAB-2026-011" },
  "new-audit-wizard-1": { expectedHeading: "New Inspection", expectedSemanticMarker: "Inspection basics", legacyState: { wizardStep: 1 } },
  "new-audit-wizard-2": { expectedHeading: "New Inspection", expectedSemanticMarker: "Category and purpose", legacyState: { wizardStep: 2 } },
  "new-audit-wizard-3": { expectedHeading: "New Inspection", expectedSemanticMarker: "When and where", legacyState: { wizardStep: 3 } },
  "new-audit-wizard-4": { expectedHeading: "New Inspection", expectedSemanticMarker: "Checklist, scope and budget", legacyState: { wizardStep: 4 } },
  "new-audit-wizard-5": { expectedHeading: "New Inspection", expectedSemanticMarker: "Review and submit", legacyState: { wizardStep: 5 } },
  "gm-planning": { expectedHeading: "Planning", expectedSemanticMarker: "Planning" },
  "gm-report-approvals": { expectedHeading: "Report Approvals", expectedSemanticMarker: "Report" },
  "gm-departments": { expectedHeading: "Departments", expectedSemanticMarker: "Department" },
  "gm-risk-dashboard": { expectedHeading: "Cross-Department Risk Dashboard", expectedSemanticMarker: "Risk" },
  "gm-settings": { expectedHeading: "Settings", expectedSemanticMarker: "Configured rules" },
  "executive-planning": { expectedHeading: "Planning", expectedSemanticMarker: "Planning" },
  "executive-preliminary-reports": { expectedHeading: "Preliminary Reports", expectedSemanticMarker: "Preliminary" },
  "executive-final-reports": { expectedHeading: "Final Reports", expectedSemanticMarker: "Final Report" },
  "executive-report-preview": { expectedHeading: "Final Report Preview", expectedSemanticMarker: "FR-2026-022" },
  "executive-notifications": { expectedHeading: "Notifications", expectedSemanticMarker: "Notifications" },
  "executive-settings": { expectedHeading: "Settings", expectedSemanticMarker: "Configured rules" },
  "auditee-inspection-coordination": { expectedHeading: "Inspection Coordination", expectedSemanticMarker: "Fly Namibia" },
  "auditee-preliminary-reports": { expectedHeading: "Preliminary Reports", expectedSemanticMarker: "Fly Namibia" },
  "auditee-final-reports": { expectedHeading: "Final Reports", expectedSemanticMarker: "Fly Namibia" },
  "auditee-report-preview": { expectedHeading: "Final Report", expectedSemanticMarker: "FR-2025-009" },
  "auditee-messages": { expectedHeading: "Messages from the CAA", expectedSemanticMarker: "Fly Namibia" },
  "auditee-documents": {
    expectedHeading: "Documents",
    expectedSemanticMarker: "Released Reports and submitted Evidence",
  },
  "auditee-settings": { expectedHeading: "Service Provider Settings", expectedSemanticMarker: "Privacy boundary" },
  "admin-regulatory-library": { expectedHeading: "Regulatory Library", expectedSemanticMarker: "Mock regulatory library" },
  "admin-template-list": { expectedHeading: "Checklist Templates", expectedSemanticMarker: "Template" },
  "admin-question-bank": { expectedHeading: "Question Bank", expectedSemanticMarker: "Question" },
  "admin-checklist-builder": { expectedHeading: "Checklist Builder", expectedSemanticMarker: "Checklist" },
  "admin-version-history": { expectedHeading: "Version History — Cabin Inspection", expectedSemanticMarker: "v1.1" },
  "admin-inspection-package-builder": { expectedHeading: "Dynamic Inspection Package Builder", expectedSemanticMarker: "Package Draft" },
  "admin-reports": { expectedHeading: "Reports", expectedSemanticMarker: "Report" },
  "admin-users-roles": { expectedHeading: "Users", expectedSemanticMarker: "Role" },
  "admin-configurations": { expectedHeading: "Settings", expectedSemanticMarker: "Configured rules" },
  "admin-organization-master-data": { expectedHeading: "Organizations", expectedSemanticMarker: "Fly Namibia" },
  "admin-organization-detail": { expectedHeading: "Fly Namibia", expectedSemanticMarker: "Organization" },
  "admin-audit-log": { expectedHeading: "Audit Log", expectedSemanticMarker: "Audit" },
};

const roleFromRoute = (id: ReactSurfaceId) => REACT_ROUTE_CONTRACTS.find((route) => route.id === id)?.requiredRole ?? null;

const GENERATED_VISUAL_SURFACES: readonly VisualSurfaceFixture[] = REACT_ROUTE_CONTRACTS
  .filter((route) => !CORE_VISUAL_SURFACES.some((surface) => surface.id === route.id))
  .map((route) => {
    const legacy = LEGACY_ROUTE_FIXTURES[route.id];
    const assertion = REVIEWED_SOURCE_ASSERTIONS[route.id];
    if (!legacy) throw new Error(`Missing reviewed legacy visual fixture for ${route.id}.`);
    if (!assertion?.expectedHeading || !assertion.expectedSemanticMarker) {
      throw new Error(`Missing reviewed legacy semantic assertion for ${route.id}.`);
    }
    if (legacy.role !== roleFromRoute(route.id)) {
      throw new Error(`Legacy visual fixture role mismatch for ${route.id}.`);
    }
    return {
      id: route.id,
      auditId: route.auditId,
      parityMode: "content-adapted",
      reactPath: route.path,
      legacy,
      stableSelector: "main.content",
      expectedHeading: assertion.expectedHeading,
      expectedRoleText: null,
      expectedOwnerText: null,
      expectedNextActionText: null,
      expectedStatusText: null,
      expectedDueDateText: null,
      expectedPrimaryActionText: null,
      expectedPrivacyAbsence: [],
      expectedSemanticMarker: assertion.expectedSemanticMarker,
      contentAdaptationReason: "Route is present in the accepted static-demo visual oracle; React implementation follows in its owning feature task.",
      masks: [],
      legacyState: assertion.legacyState,
    };
  });

export const VISUAL_SURFACES: readonly VisualSurfaceFixture[] = [
  ...CORE_VISUAL_SURFACES,
  ...GENERATED_VISUAL_SURFACES,
].sort((left, right) => Number(left.auditId.slice(-3)) - Number(right.auditId.slice(-3)));

export const VISUAL_SURFACE_BY_ID = new Map<ReactSurfaceId, VisualSurfaceFixture>(
  VISUAL_SURFACES.map((surface) => [surface.id, surface]),
);

type ReactSurfaceSemanticFields = Pick<
  VisualSurfaceFixture,
  | "expectedHeading"
  | "expectedSemanticMarker"
  | "expectedOwnerText"
  | "expectedNextActionText"
  | "expectedStatusText"
  | "expectedDueDateText"
  | "expectedPrimaryActionText"
>;

/**
 * Legacy fixtures stay immutable because they own the accepted root-demo
 * captures. React assertions resolve separately against canonical candidate
 * records and the current workbench copy.
 */
const REACT_SURFACE_SEMANTIC_OVERRIDES = {
  "inspector-home": {
    expectedSemanticMarker: "AUD-2026-001",
    expectedOwnerText: "CAA Inspector",
    expectedNextActionText: "Continue Cabin Inspection checklist",
    expectedStatusText: "IN PROGRESS",
    expectedDueDateText: "18 Jun 2026",
  },
  "audit-detail": {
    expectedSemanticMarker: "AUD-2026-001",
    expectedOwnerText: "CAA Inspector",
    expectedStatusText: "IN_PROGRESS",
    expectedDueDateText: "18 Jun 2026",
  },
  "checklist-runner": {
    expectedOwnerText: "CAA Inspector",
    expectedNextActionText: "Choose an answer",
    expectedStatusText: "IN_PROGRESS",
    expectedDueDateText: "18 Jun 2026",
    expectedPrimaryActionText: "Save response",
  },
  "finding-detail": {
    expectedDueDateText: "19 Jun 2026",
    expectedNextActionText: "Lead Inspector to review CAP",
    expectedPrimaryActionText: "Open CAP review handoff",
  },
  "inspector-findings": {
    expectedHeading: "Findings",
    expectedSemanticMarker: "CAR-2026-099",
  },
  "lead-home": {
    expectedSemanticMarker: "Lead decision required",
  },
  "inspector-calendar": {
    expectedHeading: "Audit Work Queue",
    expectedSemanticMarker: "AUD-2026-001",
  },
  "inspector-reports": {
    expectedHeading: "Reports",
    expectedSemanticMarker: "Past Reports",
  },
  "lead-final-reports": {
    expectedHeading: "Final Reports",
    expectedSemanticMarker: "RPT-CAB-2026-001",
  },
  "lead-final-report-readiness": {
    expectedHeading: "Final Report Preparation",
    expectedSemanticMarker: "RPT-CAB-2026-001",
  },
  "lead-checklist-question-assignment": {
    expectedHeading: "Assign Checklist Questions",
    expectedSemanticMarker: "CAB-EMEQ-PBE-001",
  },
  "lead-calendar": {
    expectedHeading: "Audit Work Queue",
    expectedSemanticMarker: "AUD-2026-001",
  },
  "manager-findings-review": {
    expectedHeading: "Findings Review",
    expectedSemanticMarker: "FND-CAB-2026-001",
  },
  "manager-cap-closure-review": {
    expectedHeading: "Department Manager Review",
    expectedSemanticMarker: "FND-CAB-2026-001",
  },
  "audit-plan": {
    expectedHeading: "Department Planning",
    expectedSemanticMarker: "PLAN-2026-CAB-001",
  },
  "cap-review": {
    expectedHeading: "CAP Review (Lead Inspector)",
    expectedSemanticMarker: "CAB-2026-001",
  },
  "evidence-review": {
    expectedHeading: "Findings",
    expectedSemanticMarker: "FND-CAB-2026-001",
  },
  "finance-home": {
    expectedHeading: "Finance Review",
    expectedSemanticMarker: "PLAN-2026-CAB-001",
  },
  "executive-home": {
    expectedHeading: "Executive Director Dashboard",
    expectedSemanticMarker: "RPT-CAB-2026-001-V1",
  },
  "executive-report-preview": {
    expectedHeading: "Final Report Preview",
    expectedSemanticMarker: "RPT-CAB-2026-001-V1",
  },
  "auditee-home": {
    expectedHeading: "Corrective Actions (CAP)",
    expectedSemanticMarker: "CAB-2026-001",
  },
  "auditee-report-preview": {
    expectedHeading: "Final Report",
    expectedSemanticMarker: "RPT-CAB-2026-001-V1",
  },
  "auditee-settings": {
    expectedHeading: "Service Provider Settings",
    expectedSemanticMarker: "ORG-FLY-NAMIBIA",
  },
  "admin-home": {
    expectedHeading: "Template Preview — Cabin Inspection",
    expectedSemanticMarker: "CTV-CABIN-1",
  },
  "admin-version-history": {
    expectedHeading: "Version History",
    expectedSemanticMarker: "CTV-CABIN-1",
  },
  "admin-inspection-package-builder": {
    expectedHeading: "Inspection Package Builder",
    expectedSemanticMarker: "PKG-CAB-2026-001",
  },
  "admin-configurations": {
    expectedHeading: "Configurations",
    expectedSemanticMarker: "Configured demo rules",
  },
  "admin-organization-master-data": {
    expectedHeading: "Organisation Master Data",
    expectedSemanticMarker: "ORG-FLY-NAMIBIA",
  },
} satisfies Partial<Record<ReactSurfaceId, Partial<ReactSurfaceSemanticFields>>>;

export function resolveReactSurfaceSemantics(surface: VisualSurfaceFixture): VisualSurfaceFixture {
  const override = REACT_SURFACE_SEMANTIC_OVERRIDES[surface.id as keyof typeof REACT_SURFACE_SEMANTIC_OVERRIDES];
  return override ? { ...surface, ...override } : surface;
}

export function hashBytes(bytes: Uint8Array | Buffer | string): string {
  return `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
}

export function solidFrame(width: number, height: number, color: readonly [number, number, number, number]): VisualFrame {
  const data = new Uint8ClampedArray(width * height * 4);
  for (let offset = 0; offset < data.length; offset += 4) {
    data[offset] = color[0];
    data[offset + 1] = color[1];
    data[offset + 2] = color[2];
    data[offset + 3] = color[3];
  }
  return { width, height, data };
}

export function patchedFrame(
  frame: VisualFrame,
  rect: Rect,
  color: readonly [number, number, number, number],
): VisualFrame {
  const next = { ...frame, data: new Uint8ClampedArray(frame.data) };
  for (let y = rect.y; y < rect.y + rect.height; y += 1) {
    for (let x = rect.x; x < rect.x + rect.width; x += 1) {
      if (x < 0 || y < 0 || x >= frame.width || y >= frame.height) continue;
      const offset = (y * frame.width + x) * 4;
      next.data[offset] = color[0];
      next.data[offset + 1] = color[1];
      next.data[offset + 2] = color[2];
      next.data[offset + 3] = color[3];
    }
  }
  return next;
}

function rectContains(rect: Rect, x: number, y: number): boolean {
  return x >= rect.x && y >= rect.y && x < rect.x + rect.width && y < rect.y + rect.height;
}

function isMasked(masks: readonly RectMask[], x: number, y: number): boolean {
  return masks.some((mask) => mask.rects.some((rect) => rectContains(rect, x, y)));
}

export function compareVisualFrames(
  baseline: VisualFrame,
  candidate: VisualFrame,
  options: {
    region: RegionRect;
    masks: readonly RectMask[];
    maxDiffPixelRatio: number;
    maxChannelDelta: number;
  },
): VisualComparisonResult {
  if (baseline.width !== candidate.width || baseline.height !== candidate.height) {
    throw new Error("Visual frames must have identical dimensions.");
  }
  validateMaskContract(options.masks, baseline);
  let diffPixels = 0;
  let comparedPixels = 0;
  const xEnd = Math.min(options.region.x + options.region.width, baseline.width);
  const yEnd = Math.min(options.region.y + options.region.height, baseline.height);
  for (let y = Math.max(0, options.region.y); y < yEnd; y += 1) {
    for (let x = Math.max(0, options.region.x); x < xEnd; x += 1) {
      if (isMasked(options.masks, x, y)) continue;
      comparedPixels += 1;
      const offset = (y * baseline.width + x) * 4;
      const redDelta = Math.abs(baseline.data[offset] - candidate.data[offset]);
      const greenDelta = Math.abs(baseline.data[offset + 1] - candidate.data[offset + 1]);
      const blueDelta = Math.abs(baseline.data[offset + 2] - candidate.data[offset + 2]);
      const alphaDelta = Math.abs(baseline.data[offset + 3] - candidate.data[offset + 3]);
      if (
        redDelta > options.maxChannelDelta ||
        greenDelta > options.maxChannelDelta ||
        blueDelta > options.maxChannelDelta ||
        alphaDelta > 0
      ) {
        diffPixels += 1;
      }
    }
  }
  const diffPixelRatio = comparedPixels ? diffPixels / comparedPixels : 0;
  return {
    passed: diffPixelRatio <= options.maxDiffPixelRatio,
    diffPixels,
    comparedPixels,
    diffPixelRatio,
  };
}

function paethPredictor(left: number, up: number, upperLeft: number): number {
  const prediction = left + up - upperLeft;
  const leftDistance = Math.abs(prediction - left);
  const upDistance = Math.abs(prediction - up);
  const upperLeftDistance = Math.abs(prediction - upperLeft);
  if (leftDistance <= upDistance && leftDistance <= upperLeftDistance) return left;
  if (upDistance <= upperLeftDistance) return up;
  return upperLeft;
}

export function decodePngFrame(input: Uint8Array): VisualFrame {
  const bytes = Buffer.from(input);
  const signature = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
  if (bytes.length < signature.length || !bytes.subarray(0, 8).equals(signature)) {
    throw new Error("Visual comparator requires a valid PNG input.");
  }

  let width = 0;
  let height = 0;
  let bitDepth = 0;
  let colorType = -1;
  let interlaceMethod = -1;
  const idatChunks: Buffer[] = [];
  let offset = 8;
  while (offset + 12 <= bytes.length) {
    const length = bytes.readUInt32BE(offset);
    const type = bytes.toString("ascii", offset + 4, offset + 8);
    const dataStart = offset + 8;
    const dataEnd = dataStart + length;
    if (dataEnd + 4 > bytes.length) throw new Error(`Malformed PNG ${type} chunk.`);
    const data = bytes.subarray(dataStart, dataEnd);
    if (type === "IHDR") {
      width = data.readUInt32BE(0);
      height = data.readUInt32BE(4);
      bitDepth = data[8] ?? 0;
      colorType = data[9] ?? -1;
      if ((data[10] ?? -1) !== 0 || (data[11] ?? -1) !== 0) {
        throw new Error("Unsupported PNG compression or filter method.");
      }
      interlaceMethod = data[12] ?? -1;
    } else if (type === "IDAT") {
      idatChunks.push(data);
    } else if (type === "IEND") {
      break;
    }
    offset = dataEnd + 4;
  }

  if (!width || !height || !idatChunks.length) throw new Error("PNG is missing IHDR or IDAT data.");
  if (bitDepth !== 8 || interlaceMethod !== 0) {
    throw new Error("Visual comparator supports only non-interlaced 8-bit PNGs.");
  }
  const channelsByColorType: Record<number, number> = { 0: 1, 2: 3, 4: 2, 6: 4 };
  const channels = channelsByColorType[colorType];
  if (!channels) throw new Error(`Unsupported PNG color type ${colorType}.`);

  const inflated = inflateSync(Buffer.concat(idatChunks));
  const stride = width * channels;
  const expectedLength = height * (stride + 1);
  if (inflated.length !== expectedLength) {
    throw new Error(`Unexpected PNG scanline length ${inflated.length}; expected ${expectedLength}.`);
  }
  const reconstructed = Buffer.alloc(height * stride);
  let sourceOffset = 0;
  for (let row = 0; row < height; row += 1) {
    const filter = inflated[sourceOffset] ?? -1;
    sourceOffset += 1;
    const rowOffset = row * stride;
    for (let column = 0; column < stride; column += 1) {
      const raw = inflated[sourceOffset + column] ?? 0;
      const left = column >= channels ? reconstructed[rowOffset + column - channels] ?? 0 : 0;
      const up = row > 0 ? reconstructed[rowOffset - stride + column] ?? 0 : 0;
      const upperLeft = row > 0 && column >= channels
        ? reconstructed[rowOffset - stride + column - channels] ?? 0
        : 0;
      let predictor = 0;
      if (filter === 1) predictor = left;
      else if (filter === 2) predictor = up;
      else if (filter === 3) predictor = Math.floor((left + up) / 2);
      else if (filter === 4) predictor = paethPredictor(left, up, upperLeft);
      else if (filter !== 0) throw new Error(`Unsupported PNG row filter ${filter}.`);
      reconstructed[rowOffset + column] = (raw + predictor) & 0xff;
    }
    sourceOffset += stride;
  }

  const rgba = new Uint8ClampedArray(width * height * 4);
  for (let pixel = 0; pixel < width * height; pixel += 1) {
    const source = pixel * channels;
    const target = pixel * 4;
    if (colorType === 6) {
      rgba[target] = reconstructed[source] ?? 0;
      rgba[target + 1] = reconstructed[source + 1] ?? 0;
      rgba[target + 2] = reconstructed[source + 2] ?? 0;
      rgba[target + 3] = reconstructed[source + 3] ?? 255;
    } else if (colorType === 2) {
      rgba[target] = reconstructed[source] ?? 0;
      rgba[target + 1] = reconstructed[source + 1] ?? 0;
      rgba[target + 2] = reconstructed[source + 2] ?? 0;
      rgba[target + 3] = 255;
    } else {
      const grey = reconstructed[source] ?? 0;
      rgba[target] = grey;
      rgba[target + 1] = grey;
      rgba[target + 2] = grey;
      rgba[target + 3] = colorType === 4 ? reconstructed[source + 1] ?? 255 : 255;
    }
  }
  return { width, height, data: rgba };
}

export function visualComparisonRegions(
  viewport: Size & { id?: string },
  parityMode: VisualParityMode,
): VisualRegionComparisonContract[] {
  if (parityMode === "strict-shell") {
    return [{
      region: { id: "viewport", x: 0, y: 0, width: viewport.width, height: viewport.height },
      maxDiffPixelRatio: 0.03,
      maxChannelDelta: VISUAL_MAX_CHANNEL_DELTA,
    }];
  }
  const desktop = viewport.width > 900;
  const sidebarWidth = desktop ? 230 : 0;
  const topbarHeight = desktop ? 64 : 52;
  const regions: VisualRegionComparisonContract[] = [];
  if (desktop) {
    regions.push({
      region: { id: "sidebar", x: 0, y: 0, width: sidebarWidth, height: viewport.height },
      maxDiffPixelRatio: 0.03,
      maxChannelDelta: VISUAL_MAX_CHANNEL_DELTA,
    });
  }
  const shellControlWidth = desktop ? Math.min(260, viewport.width - sidebarWidth) : viewport.width;
  regions.push({
    region: {
      id: "topbar",
      x: viewport.width - shellControlWidth,
      y: 0,
      width: shellControlWidth,
      height: Math.min(topbarHeight, viewport.height),
    },
    maxDiffPixelRatio: 0.03,
    maxChannelDelta: VISUAL_MAX_CHANNEL_DELTA,
  });
  if (desktop) {
    regions.push({
      region: {
        id: "content-header",
        x: sidebarWidth,
        y: 0,
        width: Math.max(0, viewport.width - sidebarWidth),
        height: Math.min(topbarHeight, viewport.height),
      },
      maxDiffPixelRatio: 0.08,
      maxChannelDelta: VISUAL_MAX_CHANNEL_DELTA,
    });
  }
  regions.push({
    region: {
      id: "content",
      x: sidebarWidth,
      y: Math.min(topbarHeight, viewport.height),
      width: viewport.width - sidebarWidth,
      height: Math.max(0, viewport.height - topbarHeight),
    },
    maxDiffPixelRatio: 0.08,
    maxChannelDelta: VISUAL_MAX_CHANNEL_DELTA,
  });
  return regions;
}

export function assertVisualSpecFailClosed(source: string): void {
  if (source.includes("byteDiffRatio")) {
    throw new Error("Visual spec must use decoded-pixel comparison, not compressed PNG byte ratios.");
  }
  for (const required of ["decodePngFrame", "compareVisualFrames", "visualComparisonRegions"]) {
    if (!source.includes(required)) throw new Error(`Visual spec is missing decoded-pixel contract ${required}.`);
  }
  const adaptedBranch = /surface\.parityMode\s*===\s*["']content-adapted["'][\s\S]{0,5000}?\breturn\s*;/m;
  if (adaptedBranch.test(source)) {
    throw new Error("Visual spec contains a content-adapted comparison bypass.");
  }
  if (!source.includes("react-candidate-viewport") || !source.includes("decoded-pixel-region-results")) {
    throw new Error("Visual spec must attach candidate PNG and decoded-pixel region results.");
  }
}

function rejectBroadSelector(selector: string): void {
  const normalized = selector.trim().toLowerCase();
  const broadSelectors = new Set([
    "*",
    "html",
    "body",
    "#root",
    "#app-root",
    ".shell",
    ".sidebar",
    ".topbar",
    ".main",
    ".content",
    ".workspace-shell",
    ".workspace-sidebar",
    ".workspace-content",
    ".work-content",
  ]);
  if (broadSelectors.has(normalized)) {
    throw new Error(`Broad mask selector rejected: ${selector}`);
  }
  if (/(^|[\s>+~])(?:html|body|#root|#app-root|\.shell|\.sidebar|\.topbar|\.workspace-shell|\.workspace-sidebar|\.workspace-content|\.work-content)(?:$|[\s>+~.#:\[])/.test(normalized)) {
    throw new Error(`Broad ancestor mask selector rejected: ${selector}`);
  }
}

export function validateMaskContract(masks: readonly RectMask[], viewport: Size): void {
  const covered = new Uint8Array(viewport.width * viewport.height);
  for (const mask of masks) {
    rejectBroadSelector(mask.selector);
    if (!mask.rationale || mask.rationale.trim().length < 12) {
      throw new Error(`Mask ${mask.selector} must include a written rationale.`);
    }
    if (!Number.isInteger(mask.expectedCount) || mask.expectedCount < 1) {
      throw new Error(`Mask ${mask.selector} must name an exact positive expected count.`);
    }
    if (!mask.rects.length) {
      throw new Error(`Mask ${mask.selector} did not resolve to any rectangles.`);
    }
    for (const rect of mask.rects) {
      if (rect.width <= 0 || rect.height <= 0) {
        throw new Error(`Mask ${mask.selector} resolved to an empty rectangle.`);
      }
      const xStart = Math.max(0, Math.floor(rect.x));
      const yStart = Math.max(0, Math.floor(rect.y));
      const xEnd = Math.min(viewport.width, Math.ceil(rect.x + rect.width));
      const yEnd = Math.min(viewport.height, Math.ceil(rect.y + rect.height));
      for (let y = yStart; y < yEnd; y += 1) {
        for (let x = xStart; x < xEnd; x += 1) {
          covered[y * viewport.width + x] = 1;
        }
      }
    }
  }
  const coveredPixels = covered.reduce((total, value) => total + value, 0);
  const ratio = coveredPixels / (viewport.width * viewport.height);
  if (ratio > 0.05) {
    throw new Error(`Mask coverage ${ratio.toFixed(4)} exceeds the 5% viewport limit.`);
  }
}

export function validateGeometrySnapshot(
  baseline: Record<string, Rect>,
  candidate: Record<string, Rect>,
  tolerancePx = 2,
): void {
  for (const key of ["shell", "sidebar", "topbar", "content"] as const) {
    if (!baseline[key] || !candidate[key]) {
      throw new Error(`${key} geometry is missing from the snapshot.`);
    }
    for (const field of ["x", "y", "width", "height"] as const) {
      const delta = Math.abs(baseline[key][field] - candidate[key][field]);
      if (delta > tolerancePx) {
        throw new Error(`${key} geometry ${field} shifted by ${delta}px.`);
      }
    }
  }
}

export function assertViewportScreenshotContract(input: {
  fullPage: boolean;
  viewport: Size;
  imageSize: Size;
}): void {
  if (input.fullPage) {
    throw new Error("Comparator screenshots must use fullPage:false.");
  }
  if (input.viewport.width !== input.imageSize.width || input.viewport.height !== input.imageSize.height) {
    throw new Error(
      `Comparator screenshot size ${input.imageSize.width}x${input.imageSize.height} does not match viewport ${input.viewport.width}x${input.viewport.height}.`,
    );
  }
}

export function assertVisualPairAttachments(
  attachments: readonly { name: string }[],
): void {
  const candidateCount = attachments.filter(
    (attachment) => attachment.name === "react-candidate-viewport",
  ).length;
  if (candidateCount !== 1) {
    throw new Error(
      `Visual pair must include exactly one React candidate attachment; got ${candidateCount}.`,
    );
  }

  const resultCount = attachments.filter(
    (attachment) => attachment.name === "decoded-pixel-region-results",
  ).length;
  if (resultCount !== 1) {
    throw new Error(
      `Visual pair must include exactly one decoded-pixel result attachment; got ${resultCount}.`,
    );
  }
}

export function assertBaselineUpdateMode(input: {
  command: string;
  env: Pick<NodeJS.ProcessEnv, "AVIA_UPDATE_LEGACY_BASELINES">;
}): void {
  const wantsUpdate = input.env.AVIA_UPDATE_LEGACY_BASELINES === "1";
  if (wantsUpdate && input.command !== "visual:baseline:update") {
    throw new Error("Only visual:baseline:update may rewrite visual baselines.");
  }
  if (input.command === "visual:baseline:update" && !wantsUpdate) {
    throw new Error("visual:baseline:update requires AVIA_UPDATE_LEGACY_BASELINES=1.");
  }
}

export function resolveFocusedSurfaces(value = process.env.AVIA_VISUAL_SURFACES): VisualSurfaceFixture[] {
  if (value === undefined) return [...VISUAL_SURFACES];
  if (value.trim() === "") throw new Error("AVIA_VISUAL_SURFACES cannot be empty.");
  const seen = new Set<string>();
  return value.split(",").map((raw) => {
    const id = raw.trim();
    if (!id) throw new Error("AVIA_VISUAL_SURFACES contains an empty surface id.");
    if (seen.has(id)) throw new Error(`Duplicate visual surface filter: ${id}`);
    seen.add(id);
    const surface = VISUAL_SURFACE_BY_ID.get(id as ReactSurfaceId);
    if (!surface) throw new Error(`Unknown visual surface filter: ${id}`);
    return surface;
  });
}

export function resolveVisualRegions(
  value = process.env.AVIA_VISUAL_REGIONS,
  options: { allowShellOnly?: boolean } = {},
): VisualRegionFilter[] {
  if (value === undefined) return ["all"];
  if (value.trim() === "") throw new Error("AVIA_VISUAL_REGIONS cannot be empty.");
  if (value === "shell") {
    if (!options.allowShellOnly) {
      throw new Error("AVIA_VISUAL_REGIONS=shell is reserved for Task 6 shell checkpoint.");
    }
    return ["shell"];
  }
  throw new Error(`Unsupported AVIA_VISUAL_REGIONS value: ${value}`);
}

function safeBaselinePath(baselineDir: string, relativeFile: string): string {
  if (relativeFile.startsWith("/") || relativeFile.includes("..") || relativeFile.split(/[\\/]/).some((part) => part === "")) {
    throw new Error(`Unsafe baseline path: ${relativeFile}`);
  }
  const absolute = resolve(baselineDir, relativeFile);
  const rel = relative(resolve(baselineDir), absolute);
  if (rel.startsWith("..") || rel === "" || rel.split(sep)[0] === "..") {
    throw new Error(`Baseline path escapes root: ${relativeFile}`);
  }
  return absolute;
}

function listPngFiles(dir: string, base = dir): string[] {
  if (!existsSync(dir)) return [];
  const entries = readdirSync(dir).sort();
  const files: string[] = [];
  for (const entry of entries) {
    const absolute = resolve(dir, entry);
    const rel = relative(base, absolute).split(sep).join("/");
    const stat = statSync(absolute);
    if (stat.isDirectory()) {
      files.push(...listPngFiles(absolute, base));
    } else if (entry.endsWith(".png")) {
      files.push(rel);
    }
  }
  return files;
}

function compareRecord(label: string, actual: Record<string, string>, expected: Record<string, string>): void {
  const actualKeys = Object.keys(actual).sort();
  const expectedKeys = Object.keys(expected).sort();
  if (actualKeys.join("\n") !== expectedKeys.join("\n")) {
    throw new Error(`${label} metadata keys mismatch.`);
  }
  for (const key of expectedKeys) {
    if (actual[key] !== expected[key]) {
      throw new Error(`${label} metadata mismatch for ${key}.`);
    }
  }
}

export function validateBaselineManifest(manifest: BaselineManifest, options: ValidateBaselineManifestOptions): void {
  if (manifest.schemaVersion !== 1) throw new Error("Unsupported baseline manifest schema.");
  if (manifest.baselineVersion !== VISUAL_BASELINE_VERSION) {
    throw new Error(`Unexpected baseline version: ${manifest.baselineVersion}`);
  }
  if (!Array.isArray(manifest.items) || !manifest.items.length) {
    throw new Error("Baseline manifest has no items.");
  }
  const surfaceIds = new Set(manifest.items.map((item) => item.surfaceId));
  const viewportIds = new Set(manifest.items.map((item) => item.viewport));
  if (manifest.surfaceCount !== surfaceIds.size) {
    throw new Error("Baseline manifest surface count is stale.");
  }
  if (manifest.viewportCount !== viewportIds.size) {
    throw new Error("Baseline manifest viewport count is stale.");
  }

  if (options.expectedSource) {
    compareRecord("source", manifest.source.files, options.expectedSource.files);
    if (manifest.source.auditDocument.path !== options.expectedSource.auditDocument.path) {
      throw new Error("source metadata mismatch for audit document path.");
    }
    if (manifest.source.auditDocument.sha256 !== options.expectedSource.auditDocument.sha256) {
      throw new Error("source metadata mismatch for audit document hash.");
    }
    if (manifest.source.packageLockSha256 !== options.expectedSource.packageLockSha256) {
      throw new Error("package-lock metadata mismatch.");
    }
  }
  if (options.expectedEnvironment) {
    for (const key of Object.keys(options.expectedEnvironment) as Array<keyof BaselineManifestEnvironment>) {
      if (manifest.environment[key] !== options.expectedEnvironment[key]) {
        throw new Error(`${key} metadata mismatch.`);
      }
    }
  }

  const seenKeys = new Set<string>();
  const seenFiles = new Set<string>();
  for (const item of manifest.items) {
    const key = `${item.surfaceId}/${item.viewport}`;
    if (seenKeys.has(key)) throw new Error(`Duplicate baseline item: ${key}`);
    seenKeys.add(key);
    if (seenFiles.has(item.file)) throw new Error(`Duplicate baseline file: ${item.file}`);
    seenFiles.add(item.file);

    const surface = options.expectedSurfaces?.get(item.surfaceId);
    if (surface) {
      if (item.auditId !== surface.auditId) throw new Error(`Route mismatch for ${item.surfaceId}: audit id.`);
      if (item.parityMode !== surface.parityMode) throw new Error(`Route mismatch for ${item.surfaceId}: parity mode.`);
      if (item.sourceRoute.reactPath !== surface.reactPath) throw new Error(`Route mismatch for ${item.surfaceId}: React path.`);
      if (item.sourceRoute.legacyView !== surface.legacy.view) throw new Error(`Route mismatch for ${item.surfaceId}: legacy view.`);
      if (JSON.stringify(item.sourceRoute.legacyParams) !== JSON.stringify(surface.legacy.params)) {
        throw new Error(`Route mismatch for ${item.surfaceId}: legacy params.`);
      }
    }

    const viewport = options.expectedViewports?.get(item.viewport);
    if (viewport && (item.viewportSize.width !== viewport.width || item.viewportSize.height !== viewport.height)) {
      throw new Error(`Viewport metadata mismatch for ${item.surfaceId}/${item.viewport}.`);
    }

    const absolute = safeBaselinePath(options.baselineDir, item.file);
    if (!existsSync(absolute)) throw new Error(`Missing baseline PNG: ${item.file}`);
    const realHash = hashBytes(readFileSync(absolute));
    if (realHash !== item.sha256) {
      throw new Error(`Baseline hash drift for ${item.file}.`);
    }
  }

  const extraFiles = listPngFiles(options.baselineDir).filter((file) => !seenFiles.has(file));
  if (extraFiles.length) {
    throw new Error(`Unexpected extra baseline PNG file(s): ${extraFiles.join(", ")}`);
  }
}

export function byteDiffRatio(baseline: Uint8Array, candidate: Uint8Array): number {
  const max = Math.max(baseline.length, candidate.length);
  if (!max) return 0;
  let diff = Math.abs(baseline.length - candidate.length);
  const shared = Math.min(baseline.length, candidate.length);
  for (let index = 0; index < shared; index += 1) {
    if (baseline[index] !== candidate[index]) diff += 1;
  }
  return diff / max;
}

export function maxDiffForSurface(surface: VisualSurfaceFixture): number {
  return surface.parityMode === "strict-shell" ? 0.03 : 0.08;
}

export async function installDeterministicPageState(page: Page): Promise<void> {
  await page.clock.install({ time: new Date(VISUAL_FIXED_TIME_ISO) });
  await page.addInitScript(() => {
    window.localStorage.clear();
    window.sessionStorage.clear();
    window.sessionStorage.setItem("avia-route-matrix-fixtures", "1");
  });
}

export async function clearOriginStorage(page: Page): Promise<void> {
  await page.evaluate(async () => {
    window.localStorage.clear();
    window.sessionStorage.clear();
    if ("caches" in window) {
      for (const key of await window.caches.keys()) await window.caches.delete(key);
    }
    if ("serviceWorker" in navigator) {
      const registrations = await navigator.serviceWorker.getRegistrations();
      await Promise.all(registrations.map((registration) => registration.unregister()));
    }
  });
}

export async function disableVisualNoise(page: Page): Promise<void> {
  await page.addStyleTag({
    content: `
      *, *::before, *::after {
        animation-delay: 0s !important;
        animation-duration: 0s !important;
        animation-iteration-count: 1 !important;
        caret-color: transparent !important;
        scroll-behavior: auto !important;
        transition-delay: 0s !important;
        transition-duration: 0s !important;
      }
      *::selection { background: transparent !important; color: inherit !important; }
    `,
  });
}

export async function waitForStableVisualState(page: Page, stableSelector: string): Promise<void> {
  await page.waitForLoadState("domcontentloaded");
  await page.waitForLoadState("networkidle", { timeout: 5_000 }).catch(() => undefined);
  await page.waitForFunction(() => document.fonts.ready.then(() => true));
  await page.evaluate(async () => {
    const images = [...document.images];
    await Promise.all(images.map((image) => image.decode().catch(() => undefined)));
  });
  await expect(page.locator(stableSelector).first()).toBeVisible();
}

export async function driveLegacySurface(page: Page, surface: VisualSurfaceFixture): Promise<void> {
  await page.goto(VISUAL_LEGACY_BASE_URL);
  await clearOriginStorage(page);
  await page.reload({ waitUntil: "domcontentloaded" });
  await page.evaluate((fixture) => {
    const win = window as typeof window & {
      clearDemoState?: () => void;
      initializeState?: () => void;
      normalizeViewForRole?: () => void;
      roleCanOpenView?: () => boolean;
      render?: () => void;
      state?: {
        role: string | null;
        view: string;
        params: Record<string, string>;
        selectedFilters?: Record<string, string>;
        potentialFindings?: Array<Record<string, unknown>>;
        managerReportsUi?: { selectedReportId?: string; tab?: string };
        serviceProviderUi?: { cap?: { selectedFindingId?: string } };
        leadPreliminaryReportsUi?: { mode?: string; selectedReportId?: string };
        wizard?: Record<string, unknown> | null;
        ui?: { notifOpen?: boolean; menuOpen?: boolean };
      };
    };
    win.clearDemoState?.();
    win.initializeState?.();
    if (!win.state) throw new Error("Legacy state was not initialized.");
    win.state.role = fixture.role;
    win.state.view = fixture.view;
    win.state.params = { ...fixture.params };
    if (fixture.view === "lead-review" && !fixture.params.auditId) {
      win.state.potentialFindings = [{
        id: "PF-2026-001",
        auditId: "AUD-2026-001",
        orgId: "ORG-XYZ",
        questionId: "cab-em-eq-pbe",
        checklistText: "Is protective breathing equipment serviceable and accessible?",
        result: "noncompliant",
        comment: "PBE serviceability and accessibility could not be confirmed.",
        evidenceFiles: [],
        status: "pending_lead_review",
        createdBy: "Aylin Sezer",
        createdDate: "2026-06-15",
        leadDecision: null,
        findingId: null,
      }];
    }
    if (fixture.params.filter) {
      win.state.selectedFilters = { ...(win.state.selectedFilters ?? {}), [fixture.view]: fixture.params.filter };
      if (fixture.view === "findings") win.state.selectedFilters.findings = fixture.params.filter;
    }
    if (fixture.params.reportId && win.state.managerReportsUi) {
      win.state.managerReportsUi.selectedReportId = fixture.params.reportId;
      win.state.managerReportsUi.tab = "summary";
    }
    if (fixture.params.findingId && win.state.serviceProviderUi?.cap) {
      win.state.serviceProviderUi.cap.selectedFindingId = fixture.params.findingId;
    }
    if (fixture.legacyState?.preliminaryWorkflow && win.state.leadPreliminaryReportsUi) {
      win.state.leadPreliminaryReportsUi.mode = "workflow";
      win.state.leadPreliminaryReportsUi.selectedReportId = fixture.params.reportId ?? "PR-2026-018";
    }
    if (fixture.legacyState?.wizardStep) {
      win.state.wizard = {
        step: fixture.legacyState.wizardStep,
        orgId: "ORG-XYZ",
        type: "Cabin Inspection",
        domain: "Cabin Safety",
        inspectionCategory: "Routine / Announced",
        noticePolicy: "advance",
        purpose: "",
        triggerType: "Department Manager initiated",
        riskCategory: "",
        date: "2026-12-10",
        mode: "On-site",
        location: "",
        templateId: "TPL-CABIN-2026",
        scope: "",
        currency: "USD",
        requestedBudget: "0",
      };
    }
    if (win.state.ui) {
      win.state.ui.notifOpen = false;
      win.state.ui.menuOpen = false;
    }
    // The UI audit contract addresses hidden legacy states directly. Both
    // guards are deliberately bypassed only for this synchronous render so
    // source-role captures cannot silently fall back to a role home screen.
    const normalize = win.normalizeViewForRole;
    const canOpen = win.roleCanOpenView;
    win.normalizeViewForRole = () => undefined;
    win.roleCanOpenView = () => true;
    try {
      win.render?.();
    } finally {
      win.normalizeViewForRole = normalize;
      win.roleCanOpenView = canOpen;
    }
  }, { ...surface.legacy, legacyState: surface.legacyState });
  await disableVisualNoise(page);
  await waitForStableVisualState(page, surface.stableSelector);
  await assertLegacyCaptureState(page, surface);
}

export async function driveReactSurface(page: Page, surface: VisualSurfaceFixture): Promise<void> {
  await page.goto(`${VISUAL_REACT_BASE_URL}${surface.reactPath}`);
  await clearOriginStorage(page).catch(() => undefined);
  await disableVisualNoise(page);
  await waitForStableVisualState(page, surface.id === "role-select" ? ".login-selector" : "main");
  await assertSurfaceSemantics(page, resolveReactSurfaceSemantics(surface));
  await expect(page.locator("body")).not.toContainText(
    /Finding unavailable|Loading exact|route could not load|route-pending-implementation/i,
  );
}

export async function assertSurfaceSemantics(page: Page, surface: VisualSurfaceFixture): Promise<void> {
  const body = page.locator("body");
  await expect(body).toContainText(surface.expectedHeading);
  await expect(body).toContainText(surface.expectedSemanticMarker);
  if (surface.expectedRoleText) await expect(body).toContainText(surface.expectedRoleText);
  if (surface.expectedOwnerText) await expect(body).toContainText(surface.expectedOwnerText);
  if (surface.expectedNextActionText) await expect(body).toContainText(surface.expectedNextActionText);
  if (surface.expectedStatusText) await expect(body).toContainText(surface.expectedStatusText);
  if (surface.expectedDueDateText) await expect(body).toContainText(surface.expectedDueDateText);
  if (surface.expectedPrimaryActionText) await expect(body).toContainText(surface.expectedPrimaryActionText);
  for (const forbidden of surface.expectedPrivacyAbsence) {
    await expect(body).not.toContainText(forbidden);
  }
}

/** The guarded root baseline updater reaches this assertion through driveLegacySurface. */
export async function assertLegacyCaptureState(page: Page, surface: VisualSurfaceFixture): Promise<void> {
  const body = page.locator("body");
  await expect(body).toContainText(surface.expectedHeading);
  await expect(body).toContainText(surface.expectedSemanticMarker);
}
