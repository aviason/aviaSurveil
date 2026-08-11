import { type ComponentType, type LazyExoticComponent } from "react";

import { REACT_ROUTE_CONTRACTS, type ReactSurfaceId, type ScreenComponentKey } from "./route-contracts";
import { lazyRoute as lazy } from "./route-loader";

export type ScreenComponent = LazyExoticComponent<ComponentType>;
export type ScreenComponentEntry =
  | { status: "implemented"; component: ScreenComponent }
  | { status: "router-owned"; component?: undefined }
  | { status: "pending"; component?: undefined };

const implementedScreens: Partial<Record<ReactSurfaceId, ScreenComponent>> = {
  "inspector-home": lazy(async () => ({ default: (await import("../features/assignments/inspector-assignments-page")).InspectorAssignmentsPage })),
  "lead-home": lazy(async () => ({ default: (await import("../features/findings/lead-review-page")).LeadReviewPage })),
  "manager-home": lazy(async () => ({ default: (await import("../features/findings/manager-dashboard-page")).ManagerDashboardPage })),
  "gm-home": lazy(async () => ({ default: (await import("../features/planning/planning-workspaces")).GeneralManagerDashboardPage })),
  "finance-home": lazy(async () => ({ default: (await import("../features/planning/planning-workspaces")).FinanceReviewPage })),
  "executive-home": lazy(async () => ({ default: (await import("../features/reports/executive-dashboard-page")).ExecutiveDashboardPage })),
  "auditee-home": lazy(async () => ({ default: (await import("../features/caps/auditee-cap-page")).AuditeeCapPage })),
  "admin-home": lazy(async () => ({ default: (await import("../features/admin/admin-configuration-page")).AdminConfigurationPage })),
  "audit-detail": lazy(async () => ({ default: (await import("../features/inspections/audit-detail-page")).AuditDetailPage })),
  "checklist-runner": lazy(async () => ({ default: (await import("../features/checklists/checklist-runner-page")).ChecklistRunnerPage })),
  "organization-registry": lazy(async () => ({ default: (await import("../features/organizations/organization-registry-page")).OrganizationRegistryPage })),
  "audit-plan": lazy(async () => ({ default: (await import("../features/planning/planning-workspaces")).AuditPlanCalendarPage })),
  "finding-detail": lazy(async () => ({ default: (await import("../features/findings/finding-detail-page")).FindingDetailPage })),
  "inspector-findings": lazy(async () => ({ default: (await import("../features/findings/inspector-findings-page")).InspectorFindingsPage })),
  "inspector-messages": lazy(async () => ({ default: (await import("../features/communications/message-center-page")).InspectorMessageCenterPage })),
  "inspector-calendar": lazy(async () => ({ default: (await import("../features/calendar/role-calendar-page")).InspectorCalendarPage })),
  "inspector-reports": lazy(async () => ({ default: (await import("../features/reports/inspector-reports-page")).InspectorReportsPage })),
  "closure-report-preview": lazy(async () => ({ default: (await import("../features/reports/closure-report-page")).ClosureReportPage })),
  "inspector-assistant": lazy(async () => ({ default: (await import("../features/assistant/inspector-assistant-page")).InspectorAssistantPage })),
  "inspector-profile": lazy(async () => ({ default: (await import("../features/profile/profile-page")).InspectorProfilePage })),
  "lead-preliminary-reports": lazy(async () => ({ default: (await import("../features/reports/lead-report-workspaces")).LeadPreliminaryReportsPage })),
  "lead-preliminary-report-workflow": lazy(async () => ({ default: (await import("../features/reports/lead-report-workspaces")).LeadPreliminaryReportWorkflowPage })),
  "lead-final-reports": lazy(async () => ({ default: (await import("../features/reports/lead-report-workspaces")).LeadFinalReportsPage })),
  "lead-final-report-readiness": lazy(async () => ({ default: (await import("../features/reports/lead-report-workspaces")).LeadFinalReportReadinessPage })),
  "lead-prepare-final-report": lazy(async () => ({ default: (await import("../features/reports/lead-report-workspaces")).LeadPrepareFinalReportPage })),
  "lead-final-report-document": lazy(async () => ({ default: (await import("../features/reports/lead-report-workspaces")).LeadFinalReportDocumentPage })),
  "lead-audit-assignment": lazy(async () => ({ default: (await import("../features/teams/audit-assignment-page")).AuditAssignmentPage })),
  "lead-checklist-question-assignment": lazy(async () => ({ default: (await import("../features/teams/question-assignment-page")).QuestionAssignmentPage })),
  "lead-calendar": lazy(async () => ({ default: (await import("../features/calendar/role-calendar-page")).LeadCalendarPage })),
  "lead-messages": lazy(async () => ({ default: (await import("../features/communications/message-center-page")).LeadMessageCenterPage })),
  "lead-analytics-reports": lazy(async () => ({ default: (await import("../features/reports/lead-analytics-page")).LeadAnalyticsPage })),
  "lead-settings": lazy(async () => ({ default: (await import("../features/profile/profile-page")).LeadSettingsPage })),
  "cap-review": lazy(async () => ({ default: (await import("../features/caps/cap-review-page")).CapReviewPage })),
  "evidence-review": lazy(async () => ({ default: (await import("../features/evidence/evidence-review-page")).EvidenceReviewPage })),
  "report-preview": lazy(async () => ({ default: (await import("../features/reports/report-preview-page")).ReportPreviewPage })),
  "manager-audits": lazy(async () => ({ default: (await import("../features/inspections/manager-audits-page")).ManagerAuditsPage })),
  "manager-inspection-team": lazy(async () => ({ default: (await import("../features/teams/inspection-team-page")).InspectionTeamPage })),
  "manager-findings-review": lazy(async () => ({ default: (await import("../features/findings/manager-findings-review-page")).ManagerFindingsReviewPage })),
  "manager-cap-monitoring": lazy(async () => ({ default: (await import("../features/caps/manager-cap-monitoring-page")).ManagerCapMonitoringPage })),
  "manager-checklist-management": lazy(async () => ({ default: (await import("../features/checklists/checklist-management-page")).ChecklistManagementPage })),
  "organization-detail": lazy(async () => ({ default: (await import("../features/organizations/organization-detail-page")).OrganizationDetailPage })),
  "manager-preliminary-report-review": lazy(async () => ({ default: (await import("../features/reports/manager-preliminary-review-page")).ManagerPreliminaryReviewPage })),
  "manager-cap-closure-review": lazy(async () => ({ default: (await import("../features/caps/department-closure-review-page")).DepartmentClosureReviewPage })),
  "manager-risk-dashboard": lazy(async () => ({ default: (await import("../features/risk/manager-risk-workspaces")).ManagerRiskDashboardPage })),
  "manager-safety-intelligence": lazy(async () => ({ default: (await import("../features/risk/manager-risk-workspaces")).ManagerSafetyIntelligencePage })),
  "organization-risk-profile": lazy(async () => ({ default: (await import("../features/risk/manager-risk-workspaces")).OrganizationRiskProfilePage })),
  "manager-ssp-nasp": lazy(async () => ({ default: (await import("../features/risk/manager-risk-workspaces")).ManagerSspNaspPage })),
  "manager-usoap-readiness": lazy(async () => ({ default: (await import("../features/risk/manager-risk-workspaces")).ManagerUsoapReadinessPage })),
  "manager-cap-effectiveness": lazy(async () => ({ default: (await import("../features/risk/manager-risk-workspaces")).ManagerCapEffectivenessPage })),
  "new-audit-wizard-1": lazy(async () => ({ default: (await import("../features/planning/new-audit-wizard")).NewAuditWizardPage })),
  "new-audit-wizard-2": lazy(async () => ({ default: (await import("../features/planning/new-audit-wizard")).NewAuditWizardPage })),
  "new-audit-wizard-3": lazy(async () => ({ default: (await import("../features/planning/new-audit-wizard")).NewAuditWizardPage })),
  "new-audit-wizard-4": lazy(async () => ({ default: (await import("../features/planning/new-audit-wizard")).NewAuditWizardPage })),
  "new-audit-wizard-5": lazy(async () => ({ default: (await import("../features/planning/new-audit-wizard")).NewAuditWizardPage })),
  "gm-planning": lazy(async () => ({ default: (await import("../features/planning/planning-workspaces")).GeneralManagerPlanningPage })),
  "gm-report-approvals": lazy(async () => ({ default: (await import("../features/reports/executive-report-workspaces")).GeneralManagerReportApprovalsPage })),
  "gm-departments": lazy(async () => ({ default: (await import("../features/management/department-comparison-page")).DepartmentComparisonPage })),
  "gm-risk-dashboard": lazy(async () => ({ default: (await import("../features/risk/executive-risk-page")).ExecutiveRiskPage })),
  "gm-settings": lazy(async () => ({ default: (await import("../features/profile/profile-page")).GeneralManagerSettingsPage })),
  "executive-planning": lazy(async () => ({ default: (await import("../features/planning/planning-workspaces")).ExecutivePlanningPage })),
  "executive-preliminary-reports": lazy(async () => ({ default: (await import("../features/reports/executive-report-workspaces")).ExecutivePreliminaryReportsPage })),
  "executive-final-reports": lazy(async () => ({ default: (await import("../features/reports/executive-report-workspaces")).ExecutiveFinalReportsPage })),
  "executive-report-preview": lazy(async () => ({ default: (await import("../features/reports/executive-report-workspaces")).ExecutiveReportPreviewPage })),
  "executive-notifications": lazy(async () => ({ default: (await import("../features/notifications/executive-notifications-page")).ExecutiveNotificationsPage })),
  "executive-settings": lazy(async () => ({ default: (await import("../features/profile/profile-page")).ExecutiveDirectorSettingsPage })),
  "auditee-inspection-coordination": lazy(async () => ({ default: (await import("../features/auditee/inspection-coordination-page")).AuditeeInspectionCoordinationPage })),
  "auditee-preliminary-reports": lazy(async () => ({ default: (await import("../features/reports/auditee-report-pages")).AuditeePreliminaryReportsPage })),
  "auditee-final-reports": lazy(async () => ({ default: (await import("../features/reports/auditee-report-pages")).AuditeeFinalReportsPage })),
  "auditee-report-preview": lazy(async () => ({ default: (await import("../features/reports/auditee-report-pages")).AuditeeReportPreviewPage })),
  "auditee-messages": lazy(async () => ({ default: (await import("../features/communications/message-center-page")).AuditeeMessageCenterPage })),
  "auditee-documents": lazy(async () => ({ default: (await import("../features/documents/auditee-documents-page")).AuditeeDocumentsPage })),
  "auditee-settings": lazy(async () => ({ default: (await import("../features/profile/profile-page")).AuditeeSettingsPage })),
  "admin-regulatory-library": lazy(async () => ({ default: (await import("../features/admin/regulatory-library-page")).RegulatoryLibraryPage })),
  "admin-template-list": lazy(async () => ({ default: (await import("../features/admin/template-list-page")).TemplateListPage })),
  "admin-question-bank": lazy(async () => ({ default: (await import("../features/admin/question-bank-page")).QuestionBankPage })),
  "admin-checklist-builder": lazy(async () => ({ default: (await import("../features/admin/checklist-builder-page")).ChecklistBuilderRoute })),
  "admin-version-history": lazy(async () => ({ default: (await import("../features/admin/template-version-history-page")).TemplateVersionHistoryPage })),
  "admin-inspection-package-builder": lazy(async () => ({ default: (await import("../features/admin/inspection-package-admin-page")).InspectionPackageAdminPage })),
  "admin-reports": lazy(async () => ({ default: (await import("../features/admin/admin-reports-page")).AdminReportsPage })),
  "admin-users-roles": lazy(async () => ({ default: (await import("../features/admin/users-roles-page")).UsersRolesPage })),
  "admin-configurations": lazy(async () => ({ default: (await import("../features/admin/admin-configuration-page")).AdminConfigurationsPage })),
  "admin-organization-master-data": lazy(async () => ({ default: (await import("../features/admin/organization-master-data-page")).OrganizationMasterDataPage })),
  "admin-organization-detail": lazy(async () => ({ default: (await import("../features/admin/organization-master-data-page")).AdminOrganizationDetailPage })),
  "admin-audit-log": lazy(async () => ({ default: (await import("../features/admin/admin-audit-log-page")).AdminAuditLogPage })),
};

export const SCREEN_COMPONENT_REGISTRY: Readonly<Record<ScreenComponentKey, ScreenComponentEntry>> =
  Object.fromEntries(
    REACT_ROUTE_CONTRACTS.map((contract) => {
      const component = implementedScreens[contract.id];
      if (component) return [contract.componentKey, { status: "implemented", component }];
      if (contract.id === "role-select") return [contract.componentKey, { status: "router-owned" }];
      return [contract.componentKey, { status: "pending" }];
    }),
  ) as Record<ScreenComponentKey, ScreenComponentEntry>;
