import type {
  AssignmentSummary,
  AuditEventView,
  CapStatus,
  ChecklistTemplateVersionDetailView,
  ChecklistTemplateVersionView,
  EvidenceReviewState,
  EvidenceScanState,
  EvidenceUploadState,
  EvidenceVersionView,
  FindingView,
  InspectionQuestion,
  InspectionPackage,
  OrganizationSummary,
  PlanningItemView,
  PlanningIntakeDraftView,
  PlanningProposalDraftView,
  PlanningAuditPackageSetupView,
  PotentialFindingView,
  ReportVersionView,
  ReminderRuleView,
  AuthorizedSyncChange,
  CommunicationView,
  NotificationView,
  ProfileView,
  AdminRegulatoryMappingView,
  AdminRegulatoryReferenceView,
  AdminTemplateMasterView,
  AdminQuestionView,
  AdminTemplateView,
  AdminInspectionPackageView,
  AdminReportDefinitionView,
} from "../backend/backend";
import { REACT_ROUTE_CONTRACTS, type ReactSurfaceId } from "../app/route-contracts";

export interface MockCapRevision {
  id: string;
  capId: string;
  findingId: string;
  organizationId: string;
  version: number;
  revision: number;
  status: CapStatus;
  rootCause: string;
  correctiveAction: string;
  preventiveAction: string;
  responsiblePerson: string;
  targetCompletionDate: string;
  commentToCaa: string;
  commentToAuditee: string;
  internalCaaNote: string;
  reviewDecision: "ACCEPT" | "REJECT" | "REQUEST_MORE_INFORMATION" | null;
  submittedAt: string;
  reviewedAt: string | null;
}

export interface MockEvidenceVersion extends EvidenceVersionView {
  commentToAuditee: string;
}

export interface MockEvidenceReview {
  id: string;
  findingId: string;
  evidenceVersionId: string;
  decision: "CLOSE" | "PARTIALLY_CLOSE" | "NOT_CLOSE" | "REQUEST_MORE_INFORMATION";
  commentToAuditee: string;
  internalCaaNote: string;
  reviewedAt: string;
}

export interface MockCommunication extends CommunicationView {
  senderSubjectId: string;
}

export interface MockAuditeeCoordinationResponse {
  auditId: string;
  organizationId: string;
  status: "CONFIRMED" | "ALTERNATIVE_PROPOSED";
  alternativeDate: string | null;
  revision: number;
}

export interface MockReportPublicMetadata {
  kind: "PRELIMINARY" | "FINAL";
  responseDueDate: string | null;
  caaVisibleComment: string | null;
}

export interface MockEvidenceUpload {
  kind: "evidence";
  uploadId: string;
  findingId: string;
  organizationId: string;
  fileName: string;
  declaredMediaType: string;
  byteSize: number;
  sha256: string;
}

export interface MockInspectionAttachmentUpload {
  kind: "inspection-attachment";
  uploadId: string;
  inspectionAttachmentId: string;
  packageId: string;
  fileName: string;
  byteSize: number;
  sha256: string;
}

/** Route-owned presentation metadata only; canonical Finding/CAP/Evidence records remain in their existing stores. */
export interface MockScreenProjectionSeed {
  screenId: string;
  requiredRole: import("../backend/backend").Role | null;
  visibleActions: readonly import("../backend/backend").VisibleScreenAction[];
}

const action = (id: string, label: string, kind: import("../backend/backend").VisibleScreenAction["kind"], effect: import("../backend/backend").VisibleScreenAction["effect"]): import("../backend/backend").VisibleScreenAction => ({ id, label, kind, effect });

/** Exhaustive source-faithful action contract. It is intentionally not generated from labels or paths. */
export const SCREEN_VISIBLE_ACTIONS: Record<ReactSurfaceId, readonly import("../backend/backend").VisibleScreenAction[]> = {
  "role-select": [
    action("select-inspector", "Select CAA Inspector", "navigation", { type: "navigation", target: "/inspector/inspector-assignments" }),
    action("select-lead-inspector", "Select Lead Inspector", "navigation", { type: "navigation", target: "/lead-inspector/lead-review" }),
    action("select-manager", "Select Department Manager", "navigation", { type: "navigation", target: "/department-manager/dashboard" }),
    action("select-finance", "Select Finance", "navigation", { type: "navigation", target: "/finance/finance-review" }),
    action("select-gm", "Select General Manager", "navigation", { type: "navigation", target: "/general-manager/gm-dashboard" }),
    action("select-executive", "Select Executive Director", "navigation", { type: "navigation", target: "/executive-director/executive-dashboard" }),
    action("select-auditee", "Select Auditee", "navigation", { type: "navigation", target: "/auditee/service-provider-cap" }),
    action("select-admin", "Select Admin Preview", "navigation", { type: "navigation", target: "/admin/templates" }),
  ],
  "inspector-home": [action("open-assignment", "Open assigned Audit", "navigation", { type: "navigation", target: "/inspector/audits/AUD-2026-001" })],
  "inspector-findings": [action("open-finding", "Open Finding dossier", "navigation", { type: "navigation", target: "/inspector/findings/FND-CAB-2026-001" })],
  "inspector-messages": [action("compose-message", "Compose message", "modal", { type: "modal", dialog: "compose-message" })],
  "inspector-calendar": [action("open-calendar-item", "Open Audit queue item", "navigation", { type: "navigation", target: "/inspector/audits/AUD-2026-001" })],
  "inspector-reports": [action("preview-report", "Preview report", "filePreview", { type: "filePreview", file: "RPT-CAB-2026-001-V1" })],
  "audit-detail": [action("start-checklist", "Start checklist", "navigation", { type: "navigation", target: "/inspector/audits/AUD-2026-001/checklist" })],
  "checklist-runner": [action("save-response", "Save checklist response", "localProjection", { type: "localProjection", projection: "checklist-response" }), action("submit-checklist", "Submit checklist", "modal", { type: "modal", dialog: "submit-checklist" })],
  "finding-detail": [action("open-closure-report", "Open closure report", "navigation", { type: "navigation", target: "/inspector/closure-reports/CR-CAB-2026-001" })],
  "closure-report-preview": [action("download-closure-report", "Download closure report", "fileDownload", { type: "fileDownload", file: "CR-CAB-2026-001.pdf" })],
  "inspector-assistant": [action("draft-advisory", "Draft advisory", "capabilityDispatch", { type: "capabilityDispatch", capability: "assistantDrafts.createDraft" })],
  "inspector-profile": [action("save-profile", "Save profile", "modal", { type: "modal", dialog: "save-profile" })],
  "lead-home": [action("review-potential-finding", "Review Potential Finding", "navigation", { type: "navigation", target: "/lead-inspector/cap-review/FND-CAB-2026-001" })],
  "lead-preliminary-reports": [action("open-preliminary-report", "Open preliminary report", "navigation", { type: "navigation", target: "/lead-inspector/preliminary-reports/PR-2026-018" })],
  "lead-preliminary-report-workflow": [action("save-preliminary-draft", "Save Preliminary Report draft", "localProjection", { type: "localProjection", projection: "preliminary-report-draft" })],
  "lead-final-reports": [action("open-final-readiness", "Open final report readiness", "navigation", { type: "navigation", target: "/lead-inspector/final-reports/RPT-CAB-2026-001/readiness" })],
  "lead-final-report-readiness": [action("prepare-final", "Prepare final report", "navigation", { type: "navigation", target: "/lead-inspector/final-reports/RPT-CAB-2026-001/prepare" })],
  "lead-prepare-final-report": [action("save-draft", "Save draft", "localProjection", { type: "localProjection", projection: "final-report-draft" }), action("preview-report", "Preview report", "filePreview", { type: "filePreview", file: "RPT-CAB-2026-001-V1" })],
  "lead-final-report-document": [action("download-report", "Download report", "fileDownload", { type: "fileDownload", file: "RPT-CAB-2026-001.pdf" })],
  "lead-audit-assignment": [action("assign-inspector", "Assign Inspector", "modal", { type: "modal", dialog: "assign-inspector" })],
  "lead-checklist-question-assignment": [action("save-question-assignment", "Save question assignment", "localProjection", { type: "localProjection", projection: "question-assignment" })],
  "cap-review": [action("accept-cap", "Accept CAP", "modal", { type: "modal", dialog: "accept-cap", confirmCommand: { owner: "caps.review", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } }), action("request-cap-information", "Request more information", "modal", { type: "modal", dialog: "request-cap-information", confirmCommand: { owner: "caps.review", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } })],
  "lead-calendar": [action("open-lead-calendar-item", "Open review item", "navigation", { type: "navigation", target: "/lead-inspector/lead-review" })],
  "lead-messages": [action("compose-lead-message", "Compose message", "modal", { type: "modal", dialog: "compose-message" })],
  "lead-analytics-reports": [action("download-analytics", "Download analytics", "fileDownload", { type: "fileDownload", file: "lead-analytics.csv" })],
  "lead-settings": [action("save-lead-settings", "Save settings", "modal", { type: "modal", dialog: "save-settings" })],
  "manager-home": [action("open-overdue-finding", "Open overdue Finding", "navigation", { type: "navigation", target: "/department-manager/findings-review" })],
  "audit-plan": [action("create-audit", "Create audit", "navigation", { type: "navigation", target: "/department-manager/new-audit/step-1" })],
  "manager-audits": [action("open-manager-audit", "Open Audit dossier", "modal", { type: "modal", dialog: "manager-audit-dossier" })],
  "report-preview": [action("return-report", "Return report", "modal", { type: "modal", dialog: "return-report", confirmCommand: { owner: "reports.decide", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } }), action("forward-report", "Forward report", "modal", { type: "modal", dialog: "forward-report", confirmCommand: { owner: "reports.decide", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } })],
  "manager-risk-dashboard": [action("open-risk-profile", "Open organization risk profile", "navigation", { type: "navigation", target: "/department-manager/organizations/ORG-FLY-NAMIBIA/risk-profile" })],
  "manager-inspection-team": [action("open-team-member", "Open team member", "modal", { type: "modal", dialog: "team-member" })],
  "manager-findings-review": [action("open-evidence-review", "Open Evidence review", "navigation", { type: "navigation", target: "/department-manager/evidence/FND-CAB-2026-001" })],
  "manager-cap-monitoring": [action("open-cap-closure", "Open CAP closure review", "navigation", { type: "navigation", target: "/department-manager/findings/FND-CAB-2026-001/closure-review" })],
  "manager-checklist-management": [action("open-checklist-template", "Open checklist template", "modal", { type: "modal", dialog: "manager-checklist-template" })],
  "manager-safety-intelligence": [action("open-safety-intelligence", "Open safety intelligence", "localProjection", { type: "localProjection", projection: "safety-intelligence" })],
  "organization-risk-profile": [action("open-organization-finding", "Open organization Finding", "navigation", { type: "navigation", target: "/department-manager/findings-review" })],
  "manager-ssp-nasp": [action("open-ssp-nasp", "Open SSP / NASP item", "localProjection", { type: "localProjection", projection: "ssp-nasp" })],
  "manager-usoap-readiness": [action("open-usoap-gap", "Open USOAP gap", "localProjection", { type: "localProjection", projection: "usoap-gap" })],
  "manager-cap-effectiveness": [action("review-effectiveness", "Review CAP effectiveness", "modal", { type: "modal", dialog: "cap-effectiveness" })],
  "organization-registry": [action("open-organization", "Open organization", "navigation", { type: "navigation", target: "/department-manager/organizations/ORG-FLY-NAMIBIA" })],
  "organization-detail": [action("open-organization-history", "Open organization history", "localProjection", { type: "localProjection", projection: "organization-history" })],
  "evidence-review": [action("accept-evidence", "Accept Evidence", "modal", { type: "modal", dialog: "accept-evidence", confirmCommand: { owner: "evidence.review", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } }), action("request-evidence-information", "Request more information", "modal", { type: "modal", dialog: "request-evidence-information", confirmCommand: { owner: "evidence.review", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } })],
  "manager-preliminary-report-review": [action("return-preliminary-report", "Return preliminary report", "modal", { type: "modal", dialog: "return-preliminary-report", confirmCommand: { owner: "reports.decide", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } }), action("forward-preliminary-report", "Forward preliminary report", "modal", { type: "modal", dialog: "forward-preliminary-report", confirmCommand: { owner: "reports.decide", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } })],
  "manager-cap-closure-review": [action("authorize-closure", "Authorize closure", "modal", { type: "modal", dialog: "authorize-closure", confirmCommand: { owner: "findings.authorizedClose", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } })],
  "new-audit-wizard-1": [action("wizard-continue", "Continue", "navigation", { type: "navigation", target: "/department-manager/new-audit/step-2" }), action("wizard-cancel", "Cancel", "navigation", { type: "navigation", target: "/department-manager/audit-plan" })],
  "new-audit-wizard-2": [action("wizard-back", "Back", "navigation", { type: "navigation", target: "/department-manager/new-audit/step-1" }), action("wizard-continue", "Continue", "navigation", { type: "navigation", target: "/department-manager/new-audit/step-3" })],
  "new-audit-wizard-3": [action("wizard-back", "Back", "navigation", { type: "navigation", target: "/department-manager/new-audit/step-2" }), action("wizard-continue", "Continue", "navigation", { type: "navigation", target: "/department-manager/new-audit/step-4" })],
  "new-audit-wizard-4": [action("wizard-back", "Back", "navigation", { type: "navigation", target: "/department-manager/new-audit/step-3" }), action("wizard-review", "Continue to review", "navigation", { type: "navigation", target: "/department-manager/new-audit/step-5" })],
  "new-audit-wizard-5": [action("wizard-back", "Back", "navigation", { type: "navigation", target: "/department-manager/new-audit/step-4" }), action("wizard-submit", "Submit to Finance", "capabilityDispatch", { type: "capabilityDispatch", capability: "planningProposal.submit" })],
  "post-release-checklist-selection": [action("return-to-planning", "Return to Planning", "navigation", { type: "navigation", target: "/department-manager/audit-plan" })],
  "gm-home": [action("open-department-summary", "Open department summary", "localProjection", { type: "localProjection", projection: "department-summary" })],
  "gm-planning": [action("review-plan", "Review plan", "modal", { type: "modal", dialog: "review-plan" })],
  "gm-report-approvals": [action("forward-report", "Forward report", "modal", { type: "modal", dialog: "forward-report", confirmCommand: { owner: "reports.decide", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } }), action("return-report", "Return report", "modal", { type: "modal", dialog: "return-report", confirmCommand: { owner: "reports.decide", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } })],
  "gm-departments": [action("open-department", "Open department", "localProjection", { type: "localProjection", projection: "department" })],
  "gm-risk-dashboard": [action("open-summary-risk", "Open summary risk", "localProjection", { type: "localProjection", projection: "summary-risk" })],
  "gm-settings": [action("save-gm-settings", "Save settings", "modal", { type: "modal", dialog: "save-settings" })],
  "finance-home": [action("approve-budget", "Approve budget", "modal", { type: "modal", dialog: "approve-budget", confirmCommand: { owner: "planning.decide", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } }), action("return-budget", "Return budget", "modal", { type: "modal", dialog: "return-budget", confirmCommand: { owner: "planning.decide", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } })],
  "executive-home": [action("open-executive-report", "Open executive report", "navigation", { type: "navigation", target: "/executive-director/final-reports" })],
  "executive-planning": [action("approve-plan", "Approve plan", "modal", { type: "modal", dialog: "approve-plan", confirmCommand: { owner: "planning.decide", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } }), action("return-plan", "Return plan", "modal", { type: "modal", dialog: "return-plan", confirmCommand: { owner: "planning.decide", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } })],
  "executive-preliminary-reports": [action("issue-preliminary-report", "Issue preliminary report", "modal", { type: "modal", dialog: "issue-preliminary-report", confirmCommand: { owner: "reports.decide", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } })],
  "executive-final-reports": [action("issue-final-report", "Issue final report", "modal", { type: "modal", dialog: "issue-final-report", confirmCommand: { owner: "reports.decide", requiresRevision: true, requiresIdempotency: false, requiresOperationMetadata: true } })],
  "executive-report-preview": [action("download-executive-report", "Download report", "fileDownload", { type: "fileDownload", file: "RPT-CAB-2026-001.pdf" })],
  "executive-notifications": [action("open-executive-notification", "Open notification", "localProjection", { type: "localProjection", projection: "executive-notification" })],
  "executive-settings": [action("save-executive-settings", "Save settings", "modal", { type: "modal", dialog: "save-settings" })],
  "auditee-home": [action("respond-to-cap", "Respond to CAP", "modal", { type: "modal", dialog: "submit-cap" })],
  "auditee-inspection-coordination": [action("confirm-coordination", "Confirm coordination", "modal", { type: "modal", dialog: "confirm-coordination" })],
  "auditee-preliminary-reports": [action("open-auditee-preliminary-report", "Open report", "filePreview", { type: "filePreview", file: "PR-2026-018" })],
  "auditee-final-reports": [action("open-auditee-final-report", "Open report", "filePreview", { type: "filePreview", file: "RPT-CAB-2026-001" })],
  "auditee-report-preview": [action("download-auditee-report", "Download report", "fileDownload", { type: "fileDownload", file: "RPT-CAB-2026-001.pdf" })],
  "auditee-messages": [action("compose-auditee-message", "Compose message", "modal", { type: "modal", dialog: "compose-message" })],
  "auditee-documents": [action("download-document", "Download document", "fileDownload", { type: "fileDownload", file: "RPT-CAB-2026-001.pdf" })],
  "auditee-settings": [action("save-auditee-settings", "Save settings", "modal", { type: "modal", dialog: "save-settings" })],
  "admin-regulatory-library": [action("search-regulatory-library", "Search regulatory library", "localProjection", { type: "localProjection", projection: "regulatory-search" })],
  "admin-template-list": [action("open-template", "Open template", "navigation", { type: "navigation", target: "/admin/templates" })],
  "admin-home": [action("preview-template", "Preview template", "filePreview", { type: "filePreview", file: "CTV-CABIN-1" })],
  "admin-question-bank": [action("add-question", "Add question", "modal", { type: "modal", dialog: "add-question" })],
  "admin-checklist-builder": [action("save-checklist-builder", "Save checklist", "localProjection", { type: "localProjection", projection: "checklist-builder" })],
  "admin-version-history": [action("compare-template-version", "Compare versions", "localProjection", { type: "localProjection", projection: "template-diff" })],
  "admin-inspection-package-builder": [action("build-inspection-package", "Build inspection package", "modal", { type: "modal", dialog: "build-inspection-package" })],
  "admin-reports": [action("download-admin-report", "Download report", "fileDownload", { type: "fileDownload", file: "admin-report.csv" })],
  "admin-users-roles": [action("edit-user-role", "Edit user role", "modal", { type: "modal", dialog: "edit-user-role" })],
  "admin-configurations": [action("save-configuration", "Save configuration", "modal", { type: "modal", dialog: "save-configuration" })],
  "admin-organization-master-data": [action("open-master-organization", "Open organization", "navigation", { type: "navigation", target: "/admin/organization-master-data/ORG-FLY-NAMIBIA" })],
  "admin-organization-detail": [action("save-organization", "Save organization", "modal", { type: "modal", dialog: "save-organization" })],
  "admin-audit-log": [action("download-audit-log", "Download audit log", "fileDownload", { type: "fileDownload", file: "audit-log.csv" })],
};

export type MockUpload = MockEvidenceUpload | MockInspectionAttachmentUpload;

export interface MockState {
  adminWorkspace: {
    regulatoryReferences: AdminRegulatoryReferenceView[];
    templateMasters: AdminTemplateMasterView[];
    questions: Record<string, AdminQuestionView>;
    template: AdminTemplateView;
    inspectionPackage: AdminInspectionPackageView;
    reportDefinitions: AdminReportDefinitionView[];
    questionCounter: number;
  };
  assignments: AssignmentSummary[];
  organizations: OrganizationSummary[];
  planningItems: Record<string, PlanningItemView>;
  planningIntakeDrafts: Record<string, PlanningIntakeDraftView>;
  planningProposalDrafts: Record<string, PlanningProposalDraftView>;
  planningAuditPackageSetups: Record<string, PlanningAuditPackageSetupView>;
  checklistTemplateVersions: ChecklistTemplateVersionView[];
  checklistTemplateVersionDetails: Record<string, ChecklistTemplateVersionDetailView>;
  reminderRules: ReminderRuleView[];
  auditEvents: AuditEventView[];
  packages: Record<string, InspectionPackage>;
  checklistResponses: Record<string, import("../backend/backend").ChecklistResponseView>;
  potentialFindings: Record<string, PotentialFindingView>;
  findings: Record<string, FindingView>;
  capRevisions: MockCapRevision[];
  evidenceVersions: MockEvidenceVersion[];
  evidenceReviews: MockEvidenceReview[];
  uploads: Record<string, MockUpload>;
  reportVersions: Record<string, ReportVersionView>;
  reportPublicMetadata: Record<string, MockReportPublicMetadata>;
  auditeeCoordinationResponses: Record<string, MockAuditeeCoordinationResponse>;
  profiles: Record<string, ProfileView>;
  communications: MockCommunication[];
  notifications: NotificationView[];
  authorizedSyncChanges: AuthorizedSyncChange[];
  screenProjectionSeeds: Record<string, MockScreenProjectionSeed>;
  counters: {
    potentialFinding: number;
    finding: number;
    upload: number;
    evidenceReview: number;
    inspectionAttachment: number;
    auditEvent: number;
  };
}

const allowedAnswers = [
  "COMPLIANT",
  "NON_COMPLIANT",
  "OBSERVATION",
  "NOT_APPLICABLE",
  "NOT_CHECKED",
] as const;

const demoEmailDelivery = {
  emailDeliveryStatus: "NOT_CONFIGURED",
  emailDeliveryAttempts: 0,
  emailAcceptedAt: null,
  emailNextAttemptAt: null,
} as const;

const opsAocCabinRampMapping: AdminRegulatoryMappingView = {
  id: "RMAP-OPS-AOC-CABIN-RAMP-001",
  auditArea: "OPS",
  serviceProviderTypes: ["Air Operator (AOC)"],
  applicableRegulations: ["Part 91", "Part 92", "Part 121 / 127 / 135", "Part 140"],
  criticalElement: "CE-7",
  protocolQuestionId: "4.450",
  protocolQuestion: "Does the surveillance programme include risk-based ramp inspections of AOC holders and foreign air operators?",
  annexReferences: [
    "Annex 6 Part I 4.2.2.2",
    "Annex 6 Part I Appendix 5 §7",
    "Annex 6 Part I 4.2.12.1",
    "Annex 6 Part I 6.2.2",
    "Annex 6 Part I 6.16.1–6.16.3",
    "Annex 6 Part I 8.1.1",
  ],
  nationalReferences: [
    "NAMCAR 121.07.6–121.07.8",
    "NAMCAR 135.07.6–135.07.8",
    "NAMCAR 91.04.14, 91.04.16–91.04.17, 91.04.21–91.04.22, 91.04.24, 91.04.27–91.04.29",
    "NAMCAR 121.02.14, 121.05.21, 121.05.24–121.05.26, 121.05.30–121.05.38, 121.05.45, 121.05.48",
    "NAMCAR 135.05.17–135.05.18, 135.05.22, 135.05.25–135.05.26, 135.05.32–135.05.33",
    "NAMCAR 91.10.1, Part 121 Subpart 10, Part 135 Subpart 10, and Part 145",
  ],
  caaImplementationReference: "NCAA Operations surveillance / ramp-inspection procedure — controlled source not supplied",
  requirement: "The NCAA establishes and executes a risk-based ramp-surveillance programme that uses standardized inspection coverage, preserves inspection records, and follows identified discrepancies through the authorized safety-issue process.",
  verificationObjective: "During a scoped cabin/ramp inspection, verify that selected cabin safety equipment, restraints, seats, exits, placards, and associated control records are present, serviceable, accessible, and consistent with the operator's approved documentation.",
  expectedEvidence: [
    "Inspector observation recorded against the exact aircraft and inspection",
    "Approved operations manual, MEL, or controlled equipment reference",
    "Serviceability, maintenance, expiry, or defect-control record as applicable",
    "Permitted photo or attachment linked to the exact checklist response",
    "Inspection discrepancy and follow-up record when an exception is identified",
  ],
  whyIncluded: "OPS PQ 4.450 identifies risk-based ramp inspection and cabin/safety coverage. The supplied Annex 6 Part I crosswalk links the surveillance obligation and selected cabin/equipment SARPs to NAMCAR clauses. The six practical questions are a candidate decomposition for responsible Department Manager technical review, not an official compliance conclusion.",
  reviewStatus: "EXPERT_REVIEW_REQUIRED",
  sourceGap: "The controlled NCAA Operations surveillance/ramp-inspection procedure has not been identified or supplied. Applicability and the exact Part 127 / Part 140 linkages remain subject to responsible Department Manager technical review.",
  refreshPolicy: {
    sourceCollectionId: "NCAA-NAMCATS-ALL-PAGES",
    lastCheckedAt: "2026-07-28T20:46:37.914Z",
    nextReconciliationDate: "2027-01-28",
    nextExpertValidationDate: "2027-07-28",
    eventDrivenReview: true,
    reconciliationIntervalMonths: 6,
    expertValidationIntervalMonths: 12,
    sourceChangeState: "BASELINE_CAPTURED",
    updateMode: "PROPOSE_DRAFT_ONLY",
    documentCount: 58,
    manifestPath: "docs/regulatory-sources/ncaa-namcats-manifest.json",
    guardrails: [
      "A changed source creates a clause-impact review proposal; it never rewrites a published checklist.",
      "Source presence and extracted text do not establish applicability, legal interpretation, or expert validation.",
      "Existing audits remain pinned to their exact published checklist version.",
    ],
  },
  scopeRecommendation: {
    id: "SCOPE-REC-AUD-2026-001-CABIN-001",
    status: "ADVISORY_ONLY",
    historyState: "INSUFFICIENT_FOR_DEFERRAL",
    generatedAt: "2026-07-28T00:00:00Z",
    signals: [
      "The current candidate screen scenario contains overdue Finding FND-CAB-2026-001 and pending Evidence review for PBE serviceability.",
      "No validated two-clean-audit history window is available for question-level deferral.",
      "The regulatory mapping and controlled NCAA ramp-inspection procedure remain under expert review.",
    ],
    guardrails: [
      "Mandatory, safety-critical, newly changed, overdue, open-Finding, and unknown-history controls cannot be automatically omitted.",
      "No recorded problem is not evidence of compliance.",
      "Any scope reduction requires Department Manager approval, a recorded rationale, and an audit trail.",
      "A full-scope inspection remains due at least annually even after a validated clean history is established.",
    ],
    questionRecommendations: [
      {
        questionId: "CAB-GALLEY-001",
        classification: "ROTATIONAL_SAMPLE",
        rationale: "Keep the question in scope but use a representative sample unless aircraft condition, records, or prior findings indicate broader coverage.",
        historyBasis: "No validated clean-history window supports deferral; sampling is the maximum current reduction.",
        requiresManagerApproval: true,
      },
      {
        questionId: "CAB-LAV-001",
        classification: "ROTATIONAL_SAMPLE",
        rationale: "Sample lavatory equipment and placards, expanding to full coverage when fire-safety or maintenance signals are present.",
        historyBasis: "No validated clean-history window supports deferral; safety escalation remains mandatory.",
        requiresManagerApproval: true,
      },
      {
        questionId: "CAB-PAX-SEAT-001",
        classification: "ROTATIONAL_SAMPLE",
        rationale: "Use documented representative seat and restraint sampling, with immediate expansion when a defect is identified.",
        historyBasis: "No validated clean-history window supports deferral; the sampling floor must be recorded.",
        requiresManagerApproval: true,
      },
      {
        questionId: "CAB-EMEQ-PBE-001",
        classification: "FOCUSED_FULL",
        rationale: "Inspect the applicable PBE population and supporting records because the current scenario has an overdue related Finding and pending Evidence review.",
        historyBasis: "FND-CAB-2026-001 is overdue and the latest PBE Evidence remains pending CAA review.",
        requiresManagerApproval: true,
      },
      {
        questionId: "CAB-VID-CREW-SEAT-001",
        classification: "MANDATORY_CORE",
        rationale: "Crew restraint and exit-adjacent safety coverage stays in every inspection package.",
        historyBasis: "Safety-critical status overrides clean-history or efficiency signals.",
        requiresManagerApproval: true,
      },
      {
        questionId: "CAB-COCKPIT-GEN-001",
        classification: "MANDATORY_CORE",
        rationale: "Emergency-exit accessibility, markings, and visible cabin condition stay in every inspection package.",
        historyBasis: "Safety-critical status overrides clean-history or efficiency signals.",
        requiresManagerApproval: true,
      },
    ],
  },
  sources: [
    {
      id: "ICAO-PQ-OPS-2024-R1.1",
      title: "2024 USOAP CMA Protocol Questions — OPS",
      sourceType: "ICAO_PQ",
      version: "September 2024 Revision 1.1",
      status: "SUPPLIED_WORKING_COPY",
      locator: "04_OPS_2024 PQ Rev L FULL (en) · PQ 4.450",
      url: null,
    },
    {
      id: "NCAA-CC-ANNEX6-PARTI-A610",
      title: "Annex 6 Part I to NAMCAR compliance crosswalk",
      sourceType: "ANNEX_CROSSWALK",
      version: "Supplied working copy; edition not declared",
      status: "SUPPLIED_WORKING_COPY",
      locator: "CC.zip/CC/NAMB/Annex_NAMB_A610.docx · 4.2.2.2, 4.2.12.1, 6.2.2, 6.16, 8.1.1",
      url: null,
    },
    {
      id: "AVIASURVEIL360-AUDIT-AREA-MAPPING-2026-07-27",
      title: "AviaSurveil360 AI Regulatory Knowledge Mapping",
      sourceType: "AUDIT_AREA_TAXONOMY",
      version: "2026-07-27",
      status: "SUPPLIED_WORKING_COPY",
      locator: "AI Regulatory Mapping!A5:H5",
      url: null,
    },
    {
      id: "NCAA-NAMCARS-LIBRARY",
      title: "NCAA public NAMCARS download library",
      sourceType: "NATIONAL_LIBRARY",
      version: "Public library observed 2026-07-28",
      status: "PUBLIC_REFERENCE",
      locator: "NAMCARS public download index",
      url: "https://www.ncaa.com.na/downloads.php?pagetitle=NAMCARS",
    },
    {
      id: "NCAA-NAMCATS-LIBRARY",
      title: "NCAA public NAMCATS download library",
      sourceType: "NATIONAL_LIBRARY",
      version: "All-pages baseline synchronized 2026-07-28 · 57 unique PDFs + linked index",
      status: "PUBLIC_REFERENCE",
      locator: "docs/regulatory-sources/ncaa-namcats-manifest.json",
      url: "https://www.ncaa.com.na/downloads.php?pagetitle=NAMCATS",
    },
    {
      id: "NCAA-OPS-SURVEILLANCE-PROCEDURE",
      title: "NCAA Operations surveillance / ramp-inspection procedure",
      sourceType: "CAA_PROCEDURE",
      version: "Not supplied",
      status: "SOURCE_GAP",
      locator: "Controlled procedure/manual identity required",
      url: null,
    },
  ],
  proposedQuestions: [
    {
      id: "CAB-GALLEY-001",
      prompt: "Are galley restraints and stowage areas serviceable and secure?",
      verificationMethod: "Observe a representative sample and reconcile any defect with the approved MEL or defect-control record.",
      evidenceExamples: ["Inspector observation", "Permitted photo", "MEL or defect-control record when applicable"],
      whyIncluded: "PQ 4.450 calls for risk-based cabin/safety coverage; secure cabin stowage is a practical ramp-inspection verification point requiring expert confirmation against the operator and aircraft scope.",
    },
    {
      id: "CAB-LAV-001",
      prompt: "Are lavatory safety equipment and placards present and serviceable?",
      verificationMethod: "Observe the lavatory safety configuration and sample the applicable serviceability or inspection record.",
      evidenceExamples: ["Inspector observation", "Placard and equipment condition", "Serviceability or maintenance record"],
      whyIncluded: "Annex 6 Part I 6.2.2 and the mapped equipment clauses support cabin fire-safety and equipment checks within PQ 4.450 ramp coverage.",
    },
    {
      id: "CAB-PAX-SEAT-001",
      prompt: "Are passenger seats, belts, and adjacent fittings serviceable?",
      verificationMethod: "Sample passenger seats and restraints and reconcile exceptions with the approved defect-control process.",
      evidenceExamples: ["Seat and restraint observation", "Permitted photo", "MEL or deferred-defect record"],
      whyIncluded: "Annex 6 Part I 4.2.12.1 and 6.2.2 map passenger restraint and seat provisions to national clauses suitable for practical ramp verification.",
    },
    {
      id: "CAB-EMEQ-PBE-001",
      prompt: "Is the PBE installed, serviceable, accessible, and in compliance with configured cabin emergency equipment requirements?",
      verificationMethod: "Confirm installed position, accessibility, seal or expiry state, and the supporting serviceability record against the approved configuration.",
      evidenceExamples: ["PBE position confirmation", "Expiry or seal check", "Serviceability or maintenance record"],
      whyIncluded: "PQ 4.450 includes cabin/safety equipment within ramp inspection. The PBE-specific national clause and controlled inspection procedure still require expert validation.",
    },
    {
      id: "CAB-VID-CREW-SEAT-001",
      prompt: "Are cabin information displays and crew seats serviceable?",
      verificationMethod: "Observe the cabin information and crew-seat configuration, including harness condition and accessibility near required exits.",
      evidenceExamples: ["Crew-seat and harness observation", "Cabin information display or placard check", "Defect record when applicable"],
      whyIncluded: "Annex 6 Part I 6.16.1–6.16.3 maps cabin crew seat, harness, and exit-location requirements to NAMCAR 91/121 clauses.",
    },
    {
      id: "CAB-COCKPIT-GEN-001",
      prompt: "Are cabin general condition and emergency exits satisfactory?",
      verificationMethod: "Observe representative cabin condition, exit accessibility, markings, and visible safety-critical discrepancies.",
      evidenceExamples: ["Inspector observation", "Exit accessibility and marking check", "Permitted photo or discrepancy record"],
      whyIncluded: "PQ 4.450 explicitly includes cabin/safety in risk-based ramp inspections; Annex 6 passenger-information, equipment, and continuing-airworthiness provisions support the candidate decomposition.",
    },
  ],
};

function cabinQuestion(
  id: string,
  sectionId: string,
  prompt: string,
  assignedInspectorUserIds: string[],
  expectedEvidence = "Inspector observation and required exception comment",
): InspectionQuestion {
  return {
    id,
    sectionId,
    prompt,
    regulatoryReference: `Configured Cabin Inspection reference — ${sectionId}`,
    expectedEvidence,
    allowedAnswers: [...allowedAnswers],
    commentRequiredFor: ["NON_COMPLIANT", "OBSERVATION"],
    assignedInspectorUserIds,
    currentResponse: null,
  };
}

export function createCanonicalSeedState(now: string): MockState {
  const canonicalPackage: InspectionPackage = {
    id: "PKG-CAB-2026-001",
    auditId: "AUD-2026-001",
    organizationId: "ORG-FLY-NAMIBIA",
    organizationName: "Fly Namibia",
    title: "2026 Cabin Inspection - Fly Namibia",
    packageVersion: 1,
    schemaVersion: 1,
    protocolVersion: 1,
    templateVersionId: "CTV-CABIN-1",
    packageDigest: "sha256:candidate-cabin-package-v1",
    expiresAt: "2026-07-15T23:59:59.000Z",
    checklistStatus: "IN_PROGRESS",
    checklistRevision: 1,
    questions: [
      cabinQuestion(
        "CAB-GALLEY-001",
        "GALLEY",
        "Are galley restraints and stowage areas serviceable and secure?",
        ["USR-INSPECTOR-DAVID"],
      ),
      cabinQuestion(
        "CAB-LAV-001",
        "LAV",
        "Are lavatory safety equipment and placards present and serviceable?",
        ["USR-INSPECTOR-AMINA"],
      ),
      cabinQuestion(
        "CAB-PAX-SEAT-001",
        "PAX SEAT",
        "Are passenger seats, belts, and adjacent fittings serviceable?",
        ["USR-INSPECTOR-AMINA"],
      ),
      cabinQuestion(
        "CAB-EMEQ-PBE-001",
        "EM EQ / PBE",
        "Is the PBE installed, serviceable, accessible, and in compliance with configured cabin emergency equipment requirements?",
        ["USR-INSPECTOR-AMINA"],
        "PBE serviceability record and cabin position confirmation",
      ),
      cabinQuestion(
        "CAB-VID-CREW-SEAT-001",
        "VID+CREW SEAT",
        "Are cabin information displays and crew seats serviceable?",
        ["USR-INSPECTOR-AMINA"],
      ),
      cabinQuestion(
        "CAB-COCKPIT-GEN-001",
        "COCKPIT+CAB GEN COND+EXITS",
        "Are cabin general condition and emergency exits satisfactory?",
        ["USR-INSPECTOR-AMINA"],
      ),
    ],
  };

  const otherOrganizationFinding: FindingView = {
    id: "FND-SKYCARGO-2026-099",
    findingNumber: "CAR-2026-099",
    auditId: "AUD-2026-099",
    organizationId: "ORG-SKYCARGO",
    organizationName: "SkyCargo Air",
    title: "Cargo restraint record needs follow-up",
    description: "A configured cargo oversight check remains open.",
    regulatoryReference: "Configured cargo reference",
    findingBasis: "Configured check result",
    severity: "LEVEL_2_MAJOR",
    status: "OPEN",
    dueDate: "2026-07-30",
    dueState: "NOT_DUE",
    currentOwnerType: "AUDITEE",
    currentOwnerId: "ORG-SKYCARGO",
    currentOwnerRole: "auditee",
    nextAction: "SkyCargo Air to submit CAP",
    capRequired: true,
    evidenceRequired: true,
    repeatFinding: false,
    createdAt: now,
    issuedAt: now,
    closedAt: null,
    closureBasis: null,
    revision: 1,
  };

  return {
    adminWorkspace: {
      regulatoryReferences: [
        {
          id: "NAMCARS-CAB-001",
          title: "Configured Cabin Safety reference",
          version: "2026.1",
          status: "ACTIVE",
          effectiveDate: "2026-01-01",
          configuredRules: [
            "Configured reference for Cabin Inspection sampling",
            "Candidate-only trace from OPS PQ 4.450 / CE-7 to practical cabin/ramp verification",
            "AI draft cannot publish itself; responsible Department Manager technical review is required",
          ],
          changeHistory: [
            "2026-01-01 — Added to the mock regulatory library",
            "2026-07-28 — Added supplied-source OPS / Air Operator trace pilot",
          ],
          mappings: [opsAocCabinRampMapping],
        },
        {
          id: "NAMCARS-FOPS-004",
          title: "Configured Flight Operations reference",
          version: "2025.4",
          status: "SUPERSEDED",
          effectiveDate: "2025-10-01",
          configuredRules: ["Reference-only Flight Operations sampling metadata"],
          changeHistory: ["2026-01-01 — Superseded in demo data"],
          mappings: [],
        },
      ],
      templateMasters: [
        {
          id: "TPL-CABIN-2026",
          title: "Cabin Inspection checklist",
          publishedVersionId: "CTV-CABIN-1",
          status: "PUBLISHED",
          owner: "Department Manager",
          itemCount: canonicalPackage.questions.length,
          previewPath: "/admin/templates",
          disabledReason: null,
          revision: 1,
        },
        {
          id: "TPL-FOPS-2026",
          title: "Flight Operations checklist",
          publishedVersionId: "CTV-FOPS-1",
          status: "PUBLISHED",
          owner: "Department Manager",
          itemCount: 0,
          previewPath: null,
          disabledReason: "TPL-FOPS-2026 / CTV-FOPS-1 has no declared Template Preview route in Task 10.",
          revision: 1,
        },
      ],
      questions: Object.fromEntries(canonicalPackage.questions.map((question) => [question.id, {
        id: question.id,
        prompt: question.prompt,
        configuredReference: question.regulatoryReference ?? "No configured reference",
        expectedEvidence: question.expectedEvidence ?? "No expected Evidence configured",
        revision: 1,
      }])),
      template: {
        id: "TPL-CABIN-2026",
        publishedVersionId: "CTV-CABIN-1",
        versions: [{
          id: "CTV-CABIN-1",
          templateId: "TPL-CABIN-2026",
          version: 1,
          status: "PUBLISHED",
          owner: "Department Manager",
          creatorSubjectId: "USR-MANAGER-NORA",
          changeReason: "Initial immutable published Cabin Inspection version.",
          questionIds: canonicalPackage.questions.map((question) => question.id),
          revision: 1,
          createdAt: now,
        }],
        revision: 1,
      },
      inspectionPackage: {
        id: "PKG-CAB-2026-001",
        auditId: "AUD-2026-001",
        organizationId: "ORG-FLY-NAMIBIA",
        organizationName: "Fly Namibia",
        questionIds: canonicalPackage.questions.map((question) => question.id),
        configuredReferences: canonicalPackage.questions.map((question) => question.regulatoryReference ?? "No configured reference"),
        expectedEvidence: canonicalPackage.questions.map((question) => question.expectedEvidence ?? "No expected Evidence configured"),
        riskFocus: ["Emergency equipment serviceability", "PBE serviceability", "Cabin inspection CAP follow-up"],
      },
      reportDefinitions: [{
        id: "ADMIN-RPT-PACKAGE-001",
        title: "Inspection package configuration preview",
        description: "Typed mock report definition; this is not a real report or PDF engine.",
        packageFields: ["packageId", "auditId", "organizationId", "questionIds", "configuredReferences", "expectedEvidence", "riskFocus"],
        actionReason: "ADMIN-RPT-PACKAGE-001 generation is unavailable because Task 10 provides a typed browser-local preview only.",
      }],
      questionCounter: canonicalPackage.questions.length,
    },
    assignments: [
      {
        auditId: canonicalPackage.auditId,
        organizationId: canonicalPackage.organizationId,
        organizationName: canonicalPackage.organizationName,
        title: canonicalPackage.title,
        status: "IN_PROGRESS",
        dueDate: "2026-06-18",
        dueState: "DUE_SOON",
        nextAction: "Continue Cabin Inspection checklist",
        scheduledStartDate: "2026-06-15",
        currentOwnerId: "USR-INSPECTOR-AMINA",
        currentOwnerRole: "inspector",
        currentOwnerDisplayName: "Amina Inspector",
        inspectionNotice: "ROUTINE",
        caaReleasedToAuditee: true,
        noticeWithheld: false,
      },
      {
        auditId: "AUD-2026-099",
        organizationId: "ORG-SKYCARGO",
        organizationName: "SkyCargo Air",
        title: "Special Inspection",
        status: "IN_PROGRESS",
        dueDate: "2026-05-22",
        dueState: "OVERDUE",
        nextAction: "Continue checklist / prepare report",
        scheduledStartDate: "2026-05-20",
        currentOwnerId: "USR-INSPECTOR-DAVID",
        currentOwnerRole: "inspector",
        currentOwnerDisplayName: "David Inspector",
        inspectionNotice: "UNANNOUNCED",
        caaReleasedToAuditee: false,
        noticeWithheld: true,
      },
    ],
    organizations: [
      {
        id: "ORG-FLY-NAMIBIA",
        legalName: "Fly Namibia",
        organizationType: "OPERATOR",
        status: "ACTIVE",
        openFindingCount: 0,
        lastAuditDate: null,
        nextAuditDate: "2026-07-15",
        revision: 1,
      },
      {
        id: "ORG-SKYCARGO",
        legalName: "SkyCargo Air",
        organizationType: "OPERATOR",
        status: "ACTIVE",
        openFindingCount: 1,
        lastAuditDate: null,
        nextAuditDate: "2026-07-30",
        revision: 1,
      },
    ],
    planningItems: {
      "PLAN-2026-CAB-001": {
        id: "PLAN-2026-CAB-001",
        title: "2026 Cabin Surveillance — Fly Namibia",
        planYear: 2026,
        organizationId: "ORG-FLY-NAMIBIA",
        organizationName: "Fly Namibia",
        inspectionType: "CABIN",
        scheduledDate: "2026-07-15",
        estimatedBudget: 48000,
        status: "FINANCE_REVIEW",
        currentOwnerRole: "finance",
        nextAction: "Finance to review budget",
        revision: 1,
      },
    },
    planningIntakeDrafts: {
      "PLAN-DRAFT-2026-001": {
        id: "PLAN-DRAFT-2026-001",
        organizationId: "ORG-FLY-NAMIBIA",
        organizationName: "Fly Namibia",
        applicationType: "Continued Surveillance",
        domain: "Cabin Safety",
        inspectionCategory: "Routine / Announced",
        noticePolicy: "ADVANCE",
        purpose: "",
        triggerType: "Department Manager initiated",
        riskCategory: "",
        plannedDate: "2026-12-10",
        mode: "On-site",
        location: "",
        templateVersionId: "CTV-CABIN-1",
        scope: "",
        requestedBudget: 0,
        currency: "USD",
        revision: 1,
        submittedPlanningItemId: null,
        updatedAt: now,
      },
    },
    planningProposalDrafts: {},
    planningAuditPackageSetups: {},
    checklistTemplateVersions: [
      {
        id: "CTV-CABIN-1",
        templateId: "CABIN",
        title: "Cabin Inspection checklist",
        version: 1,
        status: "PUBLISHED",
        publishedAt: now,
        questionCount: 6,
      },
    ],
    checklistTemplateVersionDetails: {
      "CTV-CABIN-1": {
        id: "CTV-CABIN-1",
        templateId: "CABIN",
        title: "Cabin Inspection checklist",
        version: 1,
        status: "PUBLISHED",
        publishedAt: now,
        questionCount: canonicalPackage.questions.length,
        questions: canonicalPackage.questions.map((question) => ({
          id: question.id,
          sectionId: question.sectionId,
          prompt: question.prompt,
          regulatoryReference: question.regulatoryReference,
          expectedEvidence: question.expectedEvidence,
          allowedAnswers: [...question.allowedAnswers],
          commentRequiredFor: [...question.commentRequiredFor],
        })),
      },
    },
    reminderRules: [
      { id: "REM-30", label: "30 days before Due Date", offsetDays: 30, channel: "IN_APP", status: "ACTIVE", revision: 1 },
      { id: "REM-15", label: "15 days before Due Date", offsetDays: 15, channel: "IN_APP", status: "ACTIVE", revision: 1 },
      { id: "REM-7", label: "7 days before Due Date", offsetDays: 7, channel: "IN_APP", status: "ACTIVE", revision: 1 },
      { id: "REM-DUE", label: "On the Due Date", offsetDays: 0, channel: "IN_APP", status: "ACTIVE", revision: 1 },
      { id: "REM-OVERDUE", label: "After the Due Date", offsetDays: -1, channel: "IN_APP", status: "ACTIVE", revision: 1 },
    ],
    auditEvents: [
      {
        eventId: "AUDIT-REPORT-SEED-0001",
        occurredAt: now,
        actorRole: "manager",
        actorSubjectId: "USR-MANAGER-NORA",
        action: "report.decision_recorded",
        entityType: "report_version",
        entityId: "PR-2026-018-V0",
        beforeStatus: "DEPARTMENT_REVIEW",
        afterStatus: "RETURNED",
        reason: "Clarify Finding basis and supporting Evidence.",
        entityRevision: 1,
      },
      {
        eventId: "AUDIT-SYSTEM-SEED-0001",
        occurredAt: now,
        actorRole: null,
        actorSubjectId: null,
        action: "reminder.demo_trace_recorded",
        entityType: "reminder_rule",
        entityId: "REM-7",
        beforeStatus: "ACTIVE",
        afterStatus: "ACTIVE",
        reason: "Demo in-app reminder trace; no real delivery.",
        entityRevision: 1,
      },
    ],
    packages: { [canonicalPackage.id]: canonicalPackage },
    checklistResponses: {},
    potentialFindings: {},
    findings: {
      [otherOrganizationFinding.id]: { ...otherOrganizationFinding, dueDate: "2026-06-10", dueState: "OVERDUE" },
    },
    capRevisions: [
      {
        id: "CAP-CAR-2026-099-R1",
        capId: "CAP-CAR-2026-099",
        findingId: "FND-SKYCARGO-2026-099",
        organizationId: "ORG-SKYCARGO",
        version: 1,
        revision: 1,
        status: "SUPERSEDED",
        rootCause: "Initial configured root cause.",
        correctiveAction: "Initial configured corrective action.",
        preventiveAction: "Initial configured preventive action.",
        responsiblePerson: "SkyCargo Safety Manager",
        targetCompletionDate: "2026-06-01",
        commentToCaa: "Initial submission.",
        commentToAuditee: "Please clarify the prior action.",
        internalCaaNote: "Historical CAA review note.",
        reviewDecision: "REQUEST_MORE_INFORMATION",
        submittedAt: now,
        reviewedAt: now,
      },
      {
        id: "CAP-CAR-2026-099-R2",
        capId: "CAP-CAR-2026-099",
        findingId: "FND-SKYCARGO-2026-099",
        organizationId: "ORG-SKYCARGO",
        version: 2,
        revision: 2,
        status: "MORE_INFORMATION_REQUESTED",
        rootCause: "Updated configured root cause.",
        correctiveAction: "Updated configured corrective action.",
        preventiveAction: "Updated configured preventive action.",
        responsiblePerson: "SkyCargo Safety Manager",
        targetCompletionDate: "2026-06-10",
        commentToCaa: "Updated submission.",
        commentToAuditee: "Submit the requested record.",
        internalCaaNote: "Internal CAA note retained only for CAA.",
        reviewDecision: "REQUEST_MORE_INFORMATION",
        submittedAt: now,
        reviewedAt: now,
      },
    ],
    evidenceVersions: [
      { id: "EVD-CAR-2026-099-V1", findingId: "FND-SKYCARGO-2026-099", organizationId: "ORG-SKYCARGO", version: 1, fileName: "cargo-record-v1.pdf", submittedAt: now, uploadState: "UPLOADED", scanState: "CLEAN", reviewState: "MORE_INFORMATION_REQUESTED", revision: 1, commentToAuditee: "Historical evidence response." },
      { id: "EVD-CAR-2026-099-V2", findingId: "FND-SKYCARGO-2026-099", organizationId: "ORG-SKYCARGO", version: 2, fileName: "cargo-record-v2.pdf", submittedAt: now, uploadState: "UPLOADED", scanState: "CLEAN", reviewState: "PENDING_CAA_REVIEW", revision: 2, commentToAuditee: "Latest evidence pending review." },
    ],
    evidenceReviews: [],
    uploads: {},
    reportVersions: {
      "PR-2026-018-V0": {
        reportVersionId: "PR-2026-018-V0",
        reportId: "PR-2026-018",
        kind: "PRELIMINARY",
        organizationId: "ORG-FLY-NAMIBIA",
        auditId: "AUD-2026-001",
        findingIds: [],
        potentialFindingIds: [],
        potentialFindingRootDigest: "sha256:4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945",
        contentHash: "sha256:preliminary-report-v0-returned",
        version: 0,
        status: "RETURNED",
        revision: 1,
        issuedAt: null,
      },
      "PR-2026-018-V1": {
        reportVersionId: "PR-2026-018-V1",
        reportId: "PR-2026-018",
        kind: "PRELIMINARY",
        organizationId: "ORG-FLY-NAMIBIA",
        auditId: "AUD-2026-001",
        findingIds: [],
        potentialFindingIds: [],
        potentialFindingRootDigest: "sha256:4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945",
        contentHash: "sha256:preliminary-report-v1-department-review",
        version: 1,
        status: "DEPARTMENT_REVIEW",
        revision: 1,
        issuedAt: null,
      },
      "RPT-CAB-2026-001-V1": {
        reportVersionId: "RPT-CAB-2026-001-V1",
        reportId: "RPT-CAB-2026-001",
        kind: "FINAL",
        organizationId: "ORG-FLY-NAMIBIA",
        auditId: "AUD-2026-001",
        findingIds: [],
        potentialFindingIds: [],
        potentialFindingRootDigest: "sha256:4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945",
        contentHash: "sha256:candidate-report-v1",
        version: 1,
        status: "EXECUTIVE_DIRECTOR_REVIEW",
        revision: 1,
        issuedAt: null,
      },
    },
    reportPublicMetadata: {
      "PR-2026-018-V0": { kind: "PRELIMINARY", responseDueDate: null, caaVisibleComment: null },
      "PR-2026-018-V1": { kind: "PRELIMINARY", responseDueDate: null, caaVisibleComment: null },
      "RPT-CAB-2026-001-V1": { kind: "FINAL", responseDueDate: null, caaVisibleComment: null },
    },
    auditeeCoordinationResponses: {},
    profiles: {
      "USR-INSPECTOR-AMINA": {
        subjectId: "USR-INSPECTOR-AMINA",
        role: "inspector",
        organizationId: null,
        displayName: "Amina Inspector",
        revision: 1,
      },
      "USR-INSPECTOR-DAVID": {
        subjectId: "USR-INSPECTOR-DAVID",
        role: "inspector",
        organizationId: null,
        displayName: "David Inspector",
        revision: 1,
      },
      "USR-LEAD-CANER": { subjectId: "USR-LEAD-CANER", role: "leadInspector", organizationId: null, displayName: "Caner Lead Inspector", revision: 1 },
      "USR-MANAGER-NORA": { subjectId: "USR-MANAGER-NORA", role: "manager", organizationId: null, displayName: "Nora Department Manager", revision: 1 },
      "USR-GM-OMAR": { subjectId: "USR-GM-OMAR", role: "gm", organizationId: null, displayName: "Omar General Manager", revision: 1 },
      "USR-FINANCE-LINA": { subjectId: "USR-FINANCE-LINA", role: "finance", organizationId: null, displayName: "Lina Finance Reviewer", revision: 1 },
      "USR-ED-ZARA": { subjectId: "USR-ED-ZARA", role: "executiveDirector", organizationId: null, displayName: "Zara Executive Director", revision: 1 },
      "USR-AUDITEE-FLY": { subjectId: "USR-AUDITEE-FLY", role: "auditee", organizationId: "ORG-FLY-NAMIBIA", displayName: "Fly Namibia Auditee", revision: 1 },
      "USR-ADMIN-ADA": { subjectId: "USR-ADMIN-ADA", role: "admin", organizationId: null, displayName: "Ada Administrator", revision: 1 },
    },
    communications: [],
    notifications: Object.values({
      "USR-INSPECTOR-AMINA": { id: "NOT-INSPECTOR-001", subjectId: "USR-INSPECTOR-AMINA", title: "Checklist Due Soon", body: "Continue the Cabin Inspection checklist before its Due Date.", readAt: null, ...demoEmailDelivery, revision: 1 },
      "USR-LEAD-CANER": { id: "NOT-LEAD-001", subjectId: "USR-LEAD-CANER", title: "Potential Finding Review", body: "A deterministic Potential Finding is ready for Lead review.", readAt: null, ...demoEmailDelivery, revision: 1 },
      "USR-MANAGER-NORA": { id: "NOT-MANAGER-001", subjectId: "USR-MANAGER-NORA", title: "Overdue Finding", body: "A configured Finding requires management attention.", readAt: null, ...demoEmailDelivery, revision: 1 },
      "USR-AUDITEE-FLY": { id: "NOT-AUDITEE-001", subjectId: "USR-AUDITEE-FLY", title: "CAP Request", body: "Submit the requested CAP and expected Evidence.", readAt: null, ...demoEmailDelivery, revision: 1 },
    }).concat(
      Object.values({
        "USR-GM-OMAR": { id: "NOT-GM-001", subjectId: "USR-GM-OMAR", title: "Management View", body: "Review the surveillance overview.", readAt: null, ...demoEmailDelivery, revision: 1 },
        "USR-FINANCE-LINA": { id: "NOT-FINANCE-001", subjectId: "USR-FINANCE-LINA", title: "Finance Review", body: "Review the configured audit budget.", readAt: null, ...demoEmailDelivery, revision: 1 },
        "USR-ED-ZARA": { id: "NOT-EXEC-001", subjectId: "USR-ED-ZARA", title: "Report Review", body: "An executive report is ready for review.", readAt: null, ...demoEmailDelivery, revision: 1 },
        "USR-ADMIN-ADA": { id: "NOT-ADMIN-001", subjectId: "USR-ADMIN-ADA", title: "Template Library", body: "A checklist template version is available.", readAt: null, ...demoEmailDelivery, revision: 1 },
      }),
    ),
    authorizedSyncChanges: [],
    screenProjectionSeeds: Object.fromEntries(
      REACT_ROUTE_CONTRACTS.map((route) => {
        return [route.id, {
          screenId: route.id,
          requiredRole: route.requiredRole,
          visibleActions: SCREEN_VISIBLE_ACTIONS[route.id],
        } satisfies MockScreenProjectionSeed];
      }),
    ),
    counters: {
      potentialFinding: 1,
      finding: 1,
      upload: 1,
      evidenceReview: 1,
      inspectionAttachment: 1,
      auditEvent: 1,
    },
  };
}

/** Screen-fixture store only: preserves the canonical lifecycle seed where FND-CAB is created by Lead conversion. */
export function createFullScreenScenarioSeedState(now: string): MockState {
  const state = createCanonicalSeedState(now);
  state.findings["FND-CAB-2026-001"] = {
    ...state.findings["FND-SKYCARGO-2026-099"]!,
    id: "FND-CAB-2026-001",
    findingNumber: "CAB-SCREEN-2026-001",
    auditId: "AUD-2026-001",
    organizationId: "ORG-FLY-NAMIBIA",
    organizationName: "Fly Namibia",
    status: "EVIDENCE_MORE_INFORMATION_REQUESTED",
    dueDate: "2026-06-10",
    dueState: "OVERDUE",
    currentOwnerId: "ORG-FLY-NAMIBIA",
    currentOwnerRole: "auditee",
    nextAction: "Fly Namibia to provide additional Evidence",
  };
  state.capRevisions.push(
    { id: "CAP-CAB-SCREEN-R1", capId: "CAP-CAB-SCREEN", findingId: "FND-CAB-2026-001", organizationId: "ORG-FLY-NAMIBIA", version: 1, revision: 1, status: "SUPERSEDED", rootCause: "Historical root cause.", correctiveAction: "Historical corrective action.", preventiveAction: "Historical preventive action.", responsiblePerson: "Fly Namibia Safety Manager", targetCompletionDate: "2026-06-01", commentToCaa: "Historical submission.", commentToAuditee: "Update requested.", internalCaaNote: "Historical internal note.", reviewDecision: "REQUEST_MORE_INFORMATION", submittedAt: now, reviewedAt: now },
    { id: "CAP-CAB-SCREEN-R2", capId: "CAP-CAB-SCREEN", findingId: "FND-CAB-2026-001", organizationId: "ORG-FLY-NAMIBIA", version: 2, revision: 2, status: "MORE_INFORMATION_REQUESTED", rootCause: "Updated root cause.", correctiveAction: "Updated corrective action.", preventiveAction: "Updated preventive action.", responsiblePerson: "Fly Namibia Safety Manager", targetCompletionDate: "2026-06-10", commentToCaa: "Updated submission.", commentToAuditee: "Evidence required.", internalCaaNote: "Updated internal note.", reviewDecision: "REQUEST_MORE_INFORMATION", submittedAt: now, reviewedAt: now },
  );
  state.evidenceVersions.push(
    { id: "EVD-CAB-SCREEN-V1", findingId: "FND-CAB-2026-001", organizationId: "ORG-FLY-NAMIBIA", version: 1, fileName: "historical-evidence-v1.pdf", submittedAt: now, uploadState: "UPLOADED", scanState: "CLEAN", reviewState: "MORE_INFORMATION_REQUESTED", revision: 1, commentToAuditee: "Historical request." },
    { id: "EVD-CAB-SCREEN-V2", findingId: "FND-CAB-2026-001", organizationId: "ORG-FLY-NAMIBIA", version: 2, fileName: "historical-evidence-v2.pdf", submittedAt: now, uploadState: "UPLOADED", scanState: "CLEAN", reviewState: "PENDING_CAA_REVIEW", revision: 2, commentToAuditee: "Latest submission." },
  );
  state.reportVersions["RPT-CAB-2026-001-V0"] = { reportVersionId: "RPT-CAB-2026-001-V0", reportId: "RPT-CAB-2026-001", kind: "FINAL", organizationId: "ORG-FLY-NAMIBIA", auditId: "AUD-2026-001", findingIds: ["FND-CAB-2026-001"], potentialFindingIds: [], potentialFindingRootDigest: "sha256:4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945", contentHash: "sha256:screen-report-v0", version: 0, status: "RETURNED", revision: 1, issuedAt: null };
  state.reportPublicMetadata["RPT-CAB-2026-001-V0"] = { kind: "FINAL", responseDueDate: null, caaVisibleComment: null };
  state.reportVersions["RPT-CAB-2026-001-V1"]!.findingIds = ["FND-CAB-2026-001"];
  return state;
}

export function publicEvidenceVersion(version: MockEvidenceVersion): EvidenceVersionView {
  const {
    id,
    findingId,
    organizationId,
    version: versionNumber,
    fileName,
    submittedAt,
    uploadState,
    scanState,
    reviewState,
    revision,
  } = version;
  return {
    id,
    findingId,
    organizationId,
    version: versionNumber,
    fileName,
    submittedAt,
    uploadState: uploadState as EvidenceUploadState,
    scanState: scanState as EvidenceScanState,
    reviewState: reviewState as EvidenceReviewState,
    revision,
  };
}
