import type { components } from "../generated/transport/api-types";
import type {
  AdminAccessDirectoryEntryView,
  AdminInspectionPackageView,
  AdminOrganizationView,
  AdminQuestionView,
  AdminRegulatoryReferenceView,
  AdminReportDefinitionView,
  AdminTemplateMasterView,
  AdminTemplateVersionView,
  AdminTemplateView,
  AdministrationScreenProjection,
  AssistantDraftView,
  AssignmentSummary,
  AuditeeReleasedReportView,
  AuditeeCoordinationView,
  AuditEventView,
  CalendarItemView,
  CommunicationView,
  InspectionTeamAuditView,
  CapRevisionView,
  ChecklistTemplateQuestionView,
  ChecklistTemplateVersionDetailView,
  ChecklistTemplateVersionView,
  ChecklistResponseView,
  CheckoutInspectionPackageOutput,
  CompleteEvidenceUploadOutput,
  CompleteInspectionAttachmentUploadOutput,
  DocumentMetadataView,
  EvidenceVersionView,
  FindingView,
  InspectionPackage,
  ListOrganizationsOutput,
  ListPlanningItemsOutput,
  ListAssignmentsOutput,
  ListCapRevisionsOutput,
  ListFindingsOutput,
  ListPotentialFindingsOutput,
  ManagerDashboardProjection,
  NotificationView,
  OrganizationSummary,
  PageOutput,
  PlanningIntakeDraftView,
  PlanningItemView,
  PotentialFindingDecisionOutput,
  PotentialFindingView,
  ProfileView,
  PushFieldOperationResult,
  ReportVersionView,
  ReminderRuleView,
  RiskManagementProjectionView,
  RiskOverviewView,
  Role,
  ReviewCapOutput,
  ReviewEvidenceOutput,
  SubmitCapOutput,
  SubmitChecklistOutput,
  SubmitPlanningIntakeOutput,
  SyncPullResponse,
  TeamMemberView,
  UserLifecycleRequestView,
  VisibleActionResult,
} from "./backend";

type Schemas = components["schemas"];

export function mapAssignment(value: Schemas["AssignmentSummary"]): AssignmentSummary {
  return {
    auditId: value.auditId,
    organizationId: value.organizationId,
    organizationName: value.organizationName,
    title: value.title,
    status: value.status,
    dueDate: value.dueDate,
    dueState: value.dueState,
    nextAction: value.nextAction,
    packageId: value.packageId ?? null,
    assignmentId: value.assignmentId ?? null,
    revision: value.revision ?? null,
    inspectionRevision: value.inspectionRevision ?? null,
    scheduledStartDate: null,
    currentOwnerId: null,
    currentOwnerRole: null,
    currentOwnerDisplayName: null,
  };
}

export function mapAssignments(value: Schemas["ListAssignmentsOutput"]): ListAssignmentsOutput {
  return { items: value.items.map(mapAssignment), nextCursor: value.nextCursor };
}

export function mapChecklistResponse(
  value: Schemas["ChecklistResponseView"],
): ChecklistResponseView {
  return {
    id: value.id,
    questionId: value.questionId,
    answer: value.answer,
    comment: value.comment,
    revision: value.revision,
    updatedAt: value.updatedAt,
  };
}

export function mapInspectionPackage(value: Schemas["InspectionPackage"]): InspectionPackage {
  return {
    id: value.id,
    auditId: value.auditId,
    organizationId: value.organizationId,
    organizationName: value.organizationName,
    title: value.title,
    packageVersion: value.packageVersion,
    schemaVersion: value.schemaVersion,
    protocolVersion: value.protocolVersion,
    templateVersionId: value.templateVersionId,
    packageDigest: value.packageDigest,
    expiresAt: value.expiresAt,
    checklistStatus: value.checklistStatus,
    checklistRevision: value.checklistRevision,
    questions: value.questions.map((question) => ({
      id: question.id,
      sectionId: question.sectionId,
      prompt: question.prompt,
      regulatoryReference: question.regulatoryReference,
      expectedEvidence: question.expectedEvidence,
      allowedAnswers: [...question.allowedAnswers],
      commentRequiredFor: [...question.commentRequiredFor],
      assignedInspectorUserIds: [...question.assignedInspectorUserIds],
      currentResponse: question.currentResponse
        ? mapChecklistResponse(question.currentResponse)
        : null,
    })),
  };
}

export function mapCheckout(
  value: Schemas["CheckoutInspectionPackageOutput"],
): CheckoutInspectionPackageOutput {
  return {
    inspectionPackage: mapInspectionPackage(value.inspectionPackage),
    offlineGrant: {
      ...value.offlineGrant,
      allowedCommandTypes: [...value.offlineGrant.allowedCommandTypes],
      assignmentScope: { questionIds: [...value.offlineGrant.assignmentScope.questionIds] },
    },
  };
}

export function mapSubmitChecklist(
  value: Schemas["SubmitChecklistOutput"],
): SubmitChecklistOutput {
  return { ...value };
}

export function mapPotentialFinding(
  value: Schemas["PotentialFindingView"],
): PotentialFindingView {
  return { ...value };
}

export function mapPotentialFindings(
  value: Schemas["ListPotentialFindingsOutput"],
): ListPotentialFindingsOutput {
  return { items: value.items.map(mapPotentialFinding), nextCursor: value.nextCursor };
}

export function mapFinding(value: Schemas["FindingView"]): FindingView {
  return {
    id: value.id,
    findingNumber: value.findingNumber,
    auditId: value.auditId,
    organizationId: value.organizationId,
    organizationName: value.organizationName,
    title: value.title,
    description: value.description,
    regulatoryReference: value.regulatoryReference,
    findingBasis: value.findingBasis,
    severity: value.severity,
    status: value.status,
    dueDate: value.dueDate,
    dueState: value.dueState,
    currentOwnerType: value.currentOwnerType,
    currentOwnerId: value.currentOwnerId,
    currentOwnerRole: value.currentOwnerRole,
    nextAction: value.nextAction,
    capRequired: value.capRequired,
    evidenceRequired: value.evidenceRequired,
    repeatFinding: value.repeatFinding,
    createdAt: value.createdAt,
    issuedAt: value.issuedAt,
    closedAt: value.closedAt,
    closureBasis: value.closureBasis,
    revision: value.revision,
  };
}

export function mapFindings(value: Schemas["ListFindingsOutput"]): ListFindingsOutput {
  return { items: value.items.map(mapFinding), nextCursor: value.nextCursor };
}

export function mapPotentialFindingDecision(
  value: Schemas["PotentialFindingDecisionOutput"],
): PotentialFindingDecisionOutput {
  return {
    potentialFinding: mapPotentialFinding(value.potentialFinding),
    finding: value.finding ? mapFinding(value.finding) : null,
  };
}

export function mapSubmitCap(value: Schemas["SubmitCapOutput"]): SubmitCapOutput {
  return { ...value };
}

export function mapReviewCap(value: Schemas["ReviewCapOutput"]): ReviewCapOutput {
  return { ...value };
}

export function mapCapRevision(value: Schemas["CapRevisionView"]): CapRevisionView {
  if (value.audience === "AUDITEE") {
    return {
      ...value,
      latestReview: value.latestReview ? { ...value.latestReview } : null,
    };
  }
  return {
    ...value,
    latestReview: value.latestReview ? { ...value.latestReview } : null,
  };
}

export function mapCapRevisions(value: Schemas["ListCapRevisionsOutput"]): ListCapRevisionsOutput {
  return { items: value.items.map(mapCapRevision), nextCursor: null };
}

export function mapCompleteInspectionAttachment(
  value: Schemas["CompleteInspectionAttachmentUploadOutput"],
): CompleteInspectionAttachmentUploadOutput {
  return { ...value };
}

export function mapCompleteEvidence(
  value: Schemas["CompleteEvidenceUploadOutput"],
): CompleteEvidenceUploadOutput {
  return { ...value };
}

export function mapEvidenceVersion(value: Schemas["EvidenceVersionView"]): EvidenceVersionView {
  return { ...value };
}

export function mapReviewEvidence(
  value: Schemas["ReviewEvidenceOutput"],
): ReviewEvidenceOutput {
  return { ...value };
}

export function mapReportVersion(value: Schemas["ReportVersionView"]): ReportVersionView {
  return { ...value, findingIds: [...value.findingIds] };
}

export function mapManagerDashboard(
  value: Schemas["ManagerDashboardProjection"],
): ManagerDashboardProjection {
  return { ...value, recentFindingNumbers: [...value.recentFindingNumbers] };
}

export function mapOrganization(value: Schemas["OrganizationSummary"]): OrganizationSummary {
  return { ...value };
}

export function mapOrganizations(
  value: Schemas["ListOrganizationsOutput"],
): ListOrganizationsOutput {
  return { items: value.items.map(mapOrganization), nextCursor: value.nextCursor };
}

export function mapPlanningItem(value: Schemas["PlanningItemView"]): PlanningItemView {
  return { ...value };
}

export function mapPlanningItems(
  value: Schemas["ListPlanningItemsOutput"],
): ListPlanningItemsOutput {
  return { items: value.items.map(mapPlanningItem), nextCursor: value.nextCursor };
}

export function mapPlanningIntakeDraft(
  value: Schemas["PlanningIntakeDraftView"],
): PlanningIntakeDraftView {
  return { ...value };
}

export function mapSubmitPlanningIntake(
  value: Schemas["SubmitPlanningIntakeOutput"],
): SubmitPlanningIntakeOutput {
  return {
    draft: mapPlanningIntakeDraft(value.draft),
    planningItem: mapPlanningItem(value.planningItem),
  };
}

export function mapTeamMember(value: Schemas["TeamMemberView"]): TeamMemberView {
  return { ...value };
}

export function mapInspectionTeamAudit(
  value: Schemas["InspectionTeamAuditView"],
): InspectionTeamAuditView {
  return {
    ...value,
    leadInspector: mapTeamMember(value.leadInspector),
    members: value.members.map(mapTeamMember),
    assignments: value.assignments.map((assignment) => ({
      questionId: assignment.questionId,
      assignedMemberSubjectIds: [...assignment.assignedMemberSubjectIds],
    })),
    documents: value.documents.map(mapDocumentMetadata),
    history: value.history.map((event) => ({ ...event })),
  };
}

export function mapDocumentMetadata(
  value: Schemas["DocumentMetadataView"],
): DocumentMetadataView {
  const publicReviewResult = value.publicReviewResult;
  if (
    publicReviewResult !== undefined &&
    ![
      "NOT_READY",
      "PENDING_CAA_REVIEW",
      "ACCEPTED",
      "PARTIALLY_ACCEPTED",
      "REJECTED",
      "MORE_INFORMATION_REQUESTED",
      "RELEASED",
    ].includes(publicReviewResult)
  ) {
    throw new Error(`Unsupported document public review result: ${publicReviewResult}`);
  }
  return {
    ...value,
    publicReviewResult: publicReviewResult as DocumentMetadataView["publicReviewResult"],
  };
}

export function mapDocuments(
  value: Schemas["ListDocumentsOutput"],
): PageOutput<DocumentMetadataView> {
  return {
    items: value.items.map(mapDocumentMetadata),
    nextCursor: value.nextCursor,
  };
}

export function mapAuditeeReleasedReport(
  value: Schemas["AuditeeReleasedReportView"],
): AuditeeReleasedReportView {
  if (
    (value.kind !== "PRELIMINARY" && value.kind !== "FINAL") ||
    value.status !== "LOCKED" ||
    (value.caaVisibleCommentState !== "NO_COMMENT_RECORDED" &&
      value.caaVisibleCommentState !== "RECORDED")
  ) {
    throw new Error("Unsupported Auditee released Report projection.");
  }
  return {
    ...value,
    kind: value.kind,
    status: value.status,
    findingIds: [...value.findingIds],
    caaVisibleCommentState: value.caaVisibleCommentState,
  };
}

export function mapAuditeeReleasedReports(
  value: Schemas["AuditeeReleasedReportPage"],
): PageOutput<AuditeeReleasedReportView> {
  return {
    items: value.items.map(mapAuditeeReleasedReport),
    nextCursor: value.nextCursor,
  };
}

export function mapAuditeeCoordination(
  value: Schemas["AuditeeCoordinationView"],
): AuditeeCoordinationView {
  return { ...value };
}

export function mapChecklistTemplateVersions(
  value: Schemas["ListChecklistTemplateVersionsOutput"],
): PageOutput<ChecklistTemplateVersionView> {
  return { items: value.items.map((item) => ({ ...item })), nextCursor: value.nextCursor };
}

export function mapChecklistTemplateQuestion(
  value: Schemas["ChecklistTemplateQuestionView"],
): ChecklistTemplateQuestionView {
  return {
    ...value,
    allowedAnswers: [...value.allowedAnswers],
    commentRequiredFor: [...value.commentRequiredFor],
  };
}

export function mapChecklistTemplateVersionDetail(
  value: Schemas["ChecklistTemplateVersionDetailView"],
): ChecklistTemplateVersionDetailView {
  return {
    ...value,
    questions: value.questions.map(mapChecklistTemplateQuestion),
  };
}

export function mapReminderRules(
  value: Schemas["ListReminderRulesOutput"],
): PageOutput<ReminderRuleView> {
  return { items: value.items.map((item) => ({ ...item })), nextCursor: value.nextCursor };
}

export function mapCommunication(
  value: Schemas["CommunicationView"],
): CommunicationView {
  const valid =
    (value.direction === "CAA_TO_AUDITEE" &&
      value.audience === "AUDITEE" &&
      value.organizationId !== null) ||
    (value.direction === "AUDITEE_TO_CAA" &&
      value.audience === "CAA" &&
      value.organizationId !== null) ||
    (value.direction === "CAA_INTERNAL" && value.audience === "CAA");
  if (!valid) {
    throw new Error("Unsupported Communication audience/direction projection.");
  }
  return {
    ...value,
    audience: value.audience as CommunicationView["audience"],
    direction: value.direction as CommunicationView["direction"],
  };
}

export function mapCommunications(
  value: Schemas["ListCommunicationsOutput"],
): PageOutput<CommunicationView> {
  return {
    items: value.items.map(mapCommunication),
    nextCursor: value.nextCursor,
  };
}

export function mapCalendarItem(
  value: Schemas["CalendarItemView"],
): CalendarItemView {
  return { ...value };
}

export function mapCalendarItems(
  value: Schemas["ListCalendarItemsOutput"],
): PageOutput<CalendarItemView> {
  return {
    items: value.items.map(mapCalendarItem),
    nextCursor: value.nextCursor,
  };
}

export function mapProfile(value: Schemas["ProfileView"]): ProfileView {
  return { ...value };
}

export function mapNotification(
  value: Schemas["NotificationView"],
): NotificationView {
  return { ...value };
}

export function mapNotifications(
  value: Schemas["ListNotificationsOutput"],
): PageOutput<NotificationView> {
  return {
    items: value.items.map(mapNotification),
    nextCursor: value.nextCursor,
  };
}

export function mapAuditEvents(
  value: Schemas["ListAuditEventsOutput"],
): PageOutput<AuditEventView> {
  return {
    items: value.items.map((item) => ({
      ...item,
      actorRole: item.actorRole as Role | null,
      actorSubjectId: null,
      entityRevision: null,
    })),
    nextCursor: value.nextCursor,
  };
}

export function mapAdminRegulatoryReference(
  value: Schemas["AdminRegulatoryReferenceView"],
): AdminRegulatoryReferenceView {
  return {
    ...value,
    configuredRules: [...value.configuredRules],
    changeHistory: [...value.changeHistory],
    mappings: value.mappings.map((mapping) => ({
      ...mapping,
      serviceProviderTypes: [...mapping.serviceProviderTypes],
      applicableRegulations: [...mapping.applicableRegulations],
      annexReferences: [...mapping.annexReferences],
      nationalReferences: [...mapping.nationalReferences],
      expectedEvidence: [...mapping.expectedEvidence],
      refreshPolicy: {
        ...mapping.refreshPolicy,
        guardrails: [...mapping.refreshPolicy.guardrails],
      },
      scopeRecommendation: {
        ...mapping.scopeRecommendation,
        signals: [...mapping.scopeRecommendation.signals],
        guardrails: [...mapping.scopeRecommendation.guardrails],
        questionRecommendations: mapping.scopeRecommendation.questionRecommendations.map(
          (recommendation) => ({ ...recommendation }),
        ),
      },
      sources: mapping.sources.map((source) => ({ ...source })),
      proposedQuestions: mapping.proposedQuestions.map((question) => ({
        ...question,
        evidenceExamples: [...question.evidenceExamples],
      })),
    })),
  };
}

export function mapAdminTemplateMaster(
  value: Schemas["AdminTemplateMasterView"],
): AdminTemplateMasterView {
  return { ...value };
}

export function mapAdminQuestion(
  value: Schemas["AdminQuestionView"],
): AdminQuestionView {
  return { ...value };
}

export function mapAdminTemplateVersion(
  value: Schemas["AdminTemplateVersionView"],
): AdminTemplateVersionView {
  return {
    ...value,
    templateId: value.templateId as AdminTemplateVersionView["templateId"],
    questionIds: [...value.questionIds],
  };
}

export function mapAdminTemplate(
  value: Schemas["AdminTemplateView"],
): AdminTemplateView {
  return {
    id: value.id as AdminTemplateView["id"],
    publishedVersionId:
      value.publishedVersionId as AdminTemplateView["publishedVersionId"],
    versions: value.versions.map(mapAdminTemplateVersion),
    revision: value.revision,
  };
}

export function mapAdminInspectionPackage(
  value: Schemas["AdminInspectionPackageView"],
): AdminInspectionPackageView {
  return {
    ...value,
    id: value.id as AdminInspectionPackageView["id"],
    auditId: value.auditId as AdminInspectionPackageView["auditId"],
    organizationId:
      value.organizationId as AdminInspectionPackageView["organizationId"],
    organizationName:
      value.organizationName as AdminInspectionPackageView["organizationName"],
    questionIds: [...value.questionIds],
    configuredReferences: [...value.configuredReferences],
    expectedEvidence: [...value.expectedEvidence],
    riskFocus: [...value.riskFocus],
  };
}

export function mapAdminReportDefinition(
  value: Schemas["AdminReportDefinitionView"],
): AdminReportDefinitionView {
  return { ...value, packageFields: [...value.packageFields] };
}

export function mapAdminAccessDirectoryEntry(
  value: Schemas["AdminAccessDirectoryEntryView"],
): AdminAccessDirectoryEntryView {
  return {
    ...value,
    roles: [...value.roles],
    requiredActions: [...value.requiredActions],
  };
}

export function mapUserLifecycleRequest(
  value: Schemas["UserLifecycleRequestView"],
): UserLifecycleRequestView {
  return {
    ...value,
    roles: [...value.roles],
  };
}

export function mapAdminOrganization(
  value: Schemas["AdminOrganizationView"],
): AdminOrganizationView {
  return { ...value };
}

export function mapRiskOverview(
  value: Schemas["RiskOverviewView"],
): RiskOverviewView {
  return { ...value };
}

export function mapRiskManagementProjection(
  value: Schemas["RiskManagementProjectionView"],
): RiskManagementProjectionView {
  return {
    findings: value.findings.map((finding) => ({
      findingId: finding.findingId,
      findingNumber: finding.findingNumber,
      organizationId: finding.organizationId,
      organizationName: finding.organizationName,
      inspectionId: finding.inspectionId,
      inspectionTitle: finding.inspectionTitle,
      department: finding.department,
      title: finding.title,
      severity: finding.severity,
      riskLevel: finding.riskLevel,
      status: finding.status,
      issuedAt: finding.issuedAt,
      dueState: finding.dueState,
      capRequired: finding.capRequired,
    })),
    capEffectiveness: value.capEffectiveness.map((cap) => ({
      findingId: cap.findingId,
      findingNumber: cap.findingNumber,
      organizationId: cap.organizationId,
      organizationName: cap.organizationName,
      findingStatus: cap.findingStatus,
      closureBasis: cap.closureBasis,
      capId: cap.capId,
      capRevisionId: cap.capRevisionId,
      capRevision: cap.capRevision,
      capStatus: cap.capStatus,
      state: cap.state,
      reason: cap.reason,
    })),
    generatedAt: value.generatedAt,
    revision: value.revision,
  };
}

export function mapAdministrationScreenProjection(
  value: Schemas["AdministrationScreenProjection"],
): AdministrationScreenProjection {
  return structuredClone(value) as AdministrationScreenProjection;
}

export function mapVisibleActionResult(
  value: Schemas["VisibleActionResult"],
): VisibleActionResult {
  return structuredClone(value) as VisibleActionResult;
}

export function mapAssistantDraft(
  value: Schemas["AssistantDraftView"],
): AssistantDraftView {
  return { ...value, advisoryOnly: true };
}

export function mapPushResult(
  value: Schemas["PushFieldOperationResult"],
): PushFieldOperationResult {
  return {
    ...value,
    conflict: value.conflict ? { ...value.conflict } : null,
  };
}

export function mapSyncPull(value: Schemas["SyncPullResponse"]): SyncPullResponse {
  return structuredClone(value) as unknown as SyncPullResponse;
}
