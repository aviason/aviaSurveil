import type { components as GeneratedComponents } from "../generated/transport/api-types";

type GeneratedSchemas = GeneratedComponents["schemas"];

export type BackendMode = "mock" | "http";
export type LocalDate = string;
export type Instant = string;

export type Role =
  | "inspector"
  | "leadInspector"
  | "manager"
  | "finance"
  | "gm"
  | "executiveDirector"
  | "auditee"
  | "admin";

export interface BackendPrincipal {
  subjectId: string;
  role: Role;
  organizationId: string | null;
}

export interface BackendRequestOptions {
  signal?: AbortSignal;
}

export type ChecklistAnswer =
  | "COMPLIANT"
  | "NON_COMPLIANT"
  | "OBSERVATION"
  | "NOT_APPLICABLE"
  | "NOT_CHECKED";

export type DueState = "NONE" | "NOT_DUE" | "DUE_SOON" | "DUE_TODAY" | "OVERDUE";

export type PotentialFindingStatus =
  | "PENDING_LEAD_REVIEW"
  | "RETURNED"
  | "DISMISSED"
  | "CONVERTED";

export type FindingStatus =
  | "DRAFT"
  | "OPEN"
  | "WAITING_FOR_CAP"
  | "CAP_SUBMITTED"
  | "CAP_ACCEPTED"
  | "CAP_REJECTED"
  | "CAP_MORE_INFORMATION_REQUESTED"
  | "EVIDENCE_REQUIRED"
  | "EVIDENCE_SUBMITTED"
  | "PENDING_CAA_REVIEW"
  | "EVIDENCE_MORE_INFORMATION_REQUESTED"
  | "PENDING_CLOSURE"
  | "CLOSED"
  | "ESCALATED";

export type CapStatus =
  | "DRAFT"
  | "SUBMITTED"
  | "PENDING_CAA_REVIEW"
  | "ACCEPTED"
  | "REJECTED"
  | "MORE_INFORMATION_REQUESTED"
  | "SUPERSEDED";

export type EvidenceUploadState = "PENDING" | "UPLOADING" | "UPLOADED" | "FAILED";
export type EvidenceScanState = "PENDING" | "CLEAN" | "QUARANTINED" | "FAILED";
export type EvidenceReviewState =
  | "NOT_READY"
  | "PENDING_CAA_REVIEW"
  | "ACCEPTED"
  | "PARTIALLY_ACCEPTED"
  | "REJECTED"
  | "MORE_INFORMATION_REQUESTED";

export type FindingSeverity =
  | "LEVEL_1_CRITICAL"
  | "LEVEL_2_MAJOR"
  | "LEVEL_3_MINOR"
  | "OBSERVATION";

export type ReportApprovalStatus =
  | "DRAFT"
  | "DEPARTMENT_REVIEW"
  | "GM_REVIEW"
  | "EXECUTIVE_DIRECTOR_REVIEW"
  | "RETURNED"
  | "ISSUED"
  | "LOCKED";

export interface CommandMeta {
  operationId: string;
}

/** Command metadata for the composed demo capabilities added for the 86-screen migration. */
export interface RevisionedCommandMeta {
  expectedRevision: number | null;
  idempotencyKey: string;
}

export interface OptionalRevisionedCommandMeta {
  expectedRevision?: number | null;
  idempotencyKey: string;
}

export interface CommunicationView {
  id: string;
  organizationId: string | null;
  subject: string;
  body: string;
  audience: "CAA" | "AUDITEE";
  direction: "CAA_TO_AUDITEE" | "AUDITEE_TO_CAA" | "CAA_INTERNAL";
  revision: number;
  createdAt: Instant;
}

export interface CalendarItemView {
  id: string;
  auditId: string;
  organizationId: string;
  organizationName?: string;
  title: string;
  nextAction?: string;
  scheduledDate: LocalDate;
  dueState: DueState;
}

export interface ProfileView {
  subjectId: string;
  role: Role;
  organizationId: string | null;
  displayName: string;
  revision: number;
}

export interface TeamMemberView {
  subjectId: string;
  displayName: string;
  role: Role;
  organizationId: string | null;
  revision: number;
}

export interface AuditTeamAssignmentView {
  questionId: string;
  assignedMemberSubjectIds: string[];
}

export interface AuditTeamHistoryView {
  eventId: string;
  occurredAt: Instant;
  actorSubjectId: string;
  action: string;
  detail: string;
}

export interface InspectionTeamAuditView {
  auditId: string;
  organizationId: string;
  organizationName: string;
  title: string;
  status: string;
  scheduledStartDate: LocalDate | null;
  scheduledEndDate: LocalDate | null;
  leadInspector: TeamMemberView;
  members: TeamMemberView[];
  assignments: AuditTeamAssignmentView[];
  documents: DocumentMetadataView[];
  history: AuditTeamHistoryView[];
  revision: number;
}

export interface RiskOverviewView {
  organizationId: string | null;
  overdueFindingCount: number;
  openFindingCount: number;
  repeatFindingCount: number;
  advisoryHealth?: {
    score: number;
    band: string;
    basis: "CONFIGURED_DEMO_SCENARIO";
    recommendedAction: string;
  } | null;
  revision: number;
}

export type ManagementRiskLevel = "HIGH" | "MEDIUM" | "LOW" | "VERY_LOW";

export interface RiskFindingProjectionView {
  findingId: string;
  findingNumber: string;
  organizationId: string;
  organizationName: string;
  inspectionId: string;
  inspectionTitle: string | null;
  department: string | null;
  title: string;
  severity: FindingSeverity;
  riskLevel: ManagementRiskLevel;
  status: FindingStatus;
  issuedAt: Instant | null;
  dueState: DueState;
  capRequired: boolean;
}

export interface CapEffectivenessProjectionView {
  findingId: string;
  findingNumber: string;
  organizationId: string;
  organizationName: string;
  findingStatus: FindingStatus;
  closureBasis: FindingView["closureBasis"];
  capId: string | null;
  capRevisionId: string | null;
  capRevision: number | null;
  capStatus: CapStatus | null;
  state: "NOT_ELIGIBLE" | "PENDING_POST_CLOSURE_VERIFICATION";
  reason: string;
}

export interface RiskManagementProjectionView {
  findings: RiskFindingProjectionView[];
  capEffectiveness: CapEffectivenessProjectionView[];
  generatedAt: Instant;
  revision: number;
}

export interface DocumentMetadataView {
  id: string;
  organizationId: string;
  title: string;
  kind: "REPORT" | "EVIDENCE" | "CHECKLIST_TEMPLATE";
  version: number;
  revision: number;
  createdAt: Instant;
  publicReviewResult?: EvidenceReviewState | "RELEASED";
  downloadFileName?: string;
  renderStatus?: "PENDING" | "RUNNING" | "FAILED" | "SUCCEEDED";
  documentVersionId?: string;
  downloadUrl?: string;
  downloadExpiresAt?: Instant;
  sha256?: string;
  rendererHash?: string;
  templateHash?: string;
  sourceHash?: string;
}

export interface AuditeeCoordinationView {
  auditId: string;
  organizationId: string;
  organizationName: string;
  title: string;
  inspectionCategory: "Routine / Announced";
  scheduledStartDate: LocalDate;
  status: "AWAITING_AUDITEE_CONFIRMATION" | "CONFIRMED" | "ALTERNATIVE_PROPOSED";
  alternativeDate: LocalDate | null;
  nextAction: string;
  revision: number;
}

export interface RespondToAuditeeCoordinationInput extends RevisionedCommandMeta {
  auditId: string;
  organizationId: string;
  decision: "CONFIRM" | "PROPOSE_ALTERNATIVE";
  alternativeDate: LocalDate | null;
}

export interface AuditeeReleasedReportView {
  reportVersionId: string;
  reportId: string;
  kind: "PRELIMINARY" | "FINAL";
  organizationId: string;
  auditId: string;
  findingIds: string[];
  version: number;
  status: "LOCKED";
  revision: number;
  issuedAt: Instant;
  responseDueDate: LocalDate | null;
  caaVisibleCommentState: "NO_COMMENT_RECORDED" | "RECORDED";
  caaVisibleComment: string | null;
}

export interface NotificationView {
  id: string;
  subjectId: string;
  title: string;
  body: string;
  readAt: Instant | null;
  emailDeliveryStatus:
    | "NOT_CONFIGURED"
    | "PENDING"
    | "RETRYING"
    | "DELIVERED"
    | "FAILED";
  emailDeliveryAttempts: number;
  emailAcceptedAt: Instant | null;
  emailNextAttemptAt: Instant | null;
  revision: number;
}

export interface AdministrationScreenProjection {
  screenId: string;
  organizationId: string | null;
  directRecordId: string | null;
  state: "ready" | "empty" | "denied" | "returned";
  overdue: boolean;
  versionHistory: boolean;
  visibleActions: readonly VisibleScreenAction[];
}

export interface VisibleScreenAction {
  id: string;
  label: string;
  kind: VisibleActionEffect["type"];
  effect: VisibleActionEffect;
}

export type VisibleActionEffect =
  | { type: "navigation"; target: string }
  | { type: "modal"; dialog: string; confirmCommand?: ConfirmCommandBinding }
  | { type: "filePreview"; file: string }
  | { type: "fileDownload"; file: string }
  | { type: "localProjection"; projection: string }
  | { type: "capabilityDispatch"; capability: string };

export interface ConfirmCommandBinding {
  owner: "caps.review" | "evidence.review" | "findings.authorizedClose" | "planning.decide" | "reports.decide";
  requiresRevision: boolean;
  requiresIdempotency: boolean;
  requiresOperationMetadata: boolean;
}

export interface VisibleActionResult {
  screenId: string;
  actionId: string;
  effect: VisibleActionEffect;
}

export interface AssistantDraftView {
  id: string;
  findingId: string;
  prompt: string;
  draft: string;
  advisoryOnly: true;
  canCreateFinding: false;
  canSetSeverity: false;
  canCloseFinding: false;
}

export interface AssignmentSummary {
  auditId: string;
  organizationId: string;
  organizationName: string;
  title: string;
  status: string;
  dueDate: LocalDate | null;
  dueState: DueState;
  nextAction: string;
  scheduledStartDate: LocalDate | null;
  currentOwnerId: string | null;
  currentOwnerRole: Role | null;
  currentOwnerDisplayName: string | null;
  inspectionNotice?: "ROUTINE" | "ANNOUNCED" | "AD_HOC" | "UNANNOUNCED";
  caaReleasedToAuditee?: boolean;
  /** Server-resolved executable package; absent before canonical materialization. */
  packageId?: string | null;
  assignmentId?: string | null;
  revision?: number | null;
  inspectionRevision?: number | null;
  noticeWithheld?: boolean;
}

export interface ListAssignmentsInput {
  cursor?: string;
  limit?: number;
  status?: string;
}

export interface ListAssignmentsOutput {
  items: AssignmentSummary[];
  nextCursor: string | null;
}

export interface ChecklistResponseView {
  id: string;
  questionId: string;
  answer: ChecklistAnswer;
  comment: string;
  revision: number;
  updatedAt: Instant;
}

export interface InspectionQuestion {
  id: string;
  sectionId: string;
  prompt: string;
  regulatoryReference: string | null;
  expectedEvidence: string | null;
  allowedAnswers: ChecklistAnswer[];
  commentRequiredFor: ChecklistAnswer[];
  assignedInspectorUserIds: string[];
  currentResponse: ChecklistResponseView | null;
}

export interface InspectionPackage {
  id: string;
  auditId: string;
  organizationId: string;
  organizationName: string;
  title: string;
  packageVersion: number;
  schemaVersion: number;
  protocolVersion: number;
  templateVersionId: string;
  packageDigest: string;
  expiresAt: Instant;
  checklistStatus: "NOT_STARTED" | "IN_PROGRESS" | "SUBMITTED";
  checklistRevision: number;
  questions: InspectionQuestion[];
}

export interface UpsertChecklistResponseInput extends CommandMeta {
  responseId: string;
  auditId: string;
  questionId: string;
  expectedResponseRevision: number | null;
  answer: ChecklistAnswer;
  comment: string;
}

export interface SubmitChecklistInput extends CommandMeta {
  auditId: string;
  expectedChecklistRevision: number;
}

export interface ReopenChecklistInput extends CommandMeta {
  auditId: string;
  expectedChecklistRevision: number;
  reason: string;
}

export interface SubmitChecklistOutput {
  auditId: string;
  checklistStatus: "NOT_STARTED" | "IN_PROGRESS" | "SUBMITTED";
  checklistRevision: number;
}

export interface OfflineGrant {
  grantId: string;
  subjectId: string;
  organizationId: string;
  packageId: string;
  packageVersion: number;
  packageDigest: string;
  allowedCommandTypes: FieldCommandType[];
  assignmentScope: { questionIds: string[] };
  deviceInstanceId: string;
  issuedAt: Instant;
  expiresAt: Instant;
  protocolVersion: number;
}

export interface CheckoutInspectionPackageInput extends CommandMeta {
  packageId: string;
  expectedPackageVersion: number;
  deviceInstanceId: string;
}

export interface CheckoutInspectionPackageOutput {
  inspectionPackage: InspectionPackage;
  offlineGrant: OfflineGrant;
}

export interface CreatePotentialFindingInput extends CommandMeta {
  auditId: string;
  questionId: string;
  checklistResponseId: string;
  expectedChecklistResponseRevision: number;
  title: string;
  description: string;
  requiredComment: string;
  inspectionAttachmentIds: string[];
}

export interface PotentialFindingView {
  id: string;
  auditId: string;
  questionId: string;
  organizationId: string;
  title: string;
  description: string;
  status: PotentialFindingStatus;
  revision: number;
  convertedFindingId: string | null;
}

export interface ListPotentialFindingsInput {
  status?: PotentialFindingStatus;
  limit?: number;
}

export type ListPotentialFindingsOutput = PageOutput<PotentialFindingView>;

export type DecidePotentialFindingInput =
  | (CommandMeta & {
      potentialFindingId: string;
      expectedPotentialFindingRevision: number;
      decision: "RETURN" | "DISMISS";
      reason: string;
    })
  | (CommandMeta & {
      potentialFindingId: string;
      expectedPotentialFindingRevision: number;
      decision: "CONVERT";
      severity: FindingSeverity;
      capRequired: boolean;
      evidenceRequired: boolean;
      dueDate: LocalDate | null;
    });

export interface PotentialFindingDecisionOutput {
  potentialFinding: PotentialFindingView;
  finding: FindingView | null;
}

export interface FindingView {
  id: string;
  findingNumber: string;
  auditId: string;
  organizationId: string;
  organizationName: string;
  title: string;
  description: string;
  regulatoryReference: string | null;
  findingBasis: string;
  severity: FindingSeverity;
  status: FindingStatus;
  dueDate: LocalDate | null;
  dueState: DueState;
  currentOwnerType: "CAA" | "AUDITEE";
  currentOwnerId: string;
  currentOwnerRole: Role;
  nextAction: string;
  capRequired: boolean;
  evidenceRequired: boolean;
  repeatFinding: boolean;
  createdAt: Instant;
  issuedAt: Instant | null;
  closedAt: Instant | null;
  closureBasis: "EVIDENCE_VERIFIED" | "AUTHORIZED" | null;
  revision: number;
}

export interface ListFindingsInput {
  cursor?: string;
  limit?: number;
  status?: FindingStatus;
}

export interface ListFindingsOutput {
  items: FindingView[];
  nextCursor: string | null;
}

export interface AuthorizedCloseInput extends CommandMeta {
  findingId: string;
  expectedFindingRevision: number;
  reason: string;
}

export interface SubmitCapInput extends CommandMeta {
  findingId: string;
  expectedFindingRevision: number;
  rootCause: string;
  correctiveAction: string;
  preventiveAction: string;
  responsiblePerson: string;
  targetCompletionDate: LocalDate;
  commentToCaa: string;
}

export interface SubmitCapOutput {
  capRevisionId: string;
  capRevision: number;
  capStatus: "SUBMITTED" | "PENDING_CAA_REVIEW";
  findingStatus: FindingStatus;
  findingRevision: number;
}

export interface ReviewCapInput extends CommandMeta {
  capRevisionId: string;
  expectedCapRevision: number;
  findingId: string;
  expectedFindingRevision: number;
  decision: "ACCEPT" | "REJECT" | "REQUEST_MORE_INFORMATION";
  commentToAuditee: string;
  internalCaaNote: string;
}

export interface ReviewCapOutput {
  capRevisionId: string;
  capRevision: number;
  capStatus: CapStatus;
  findingStatus: FindingStatus;
  findingRevision: number;
}

export interface CapRevisionSubmission {
  id: string;
  capId: string;
  findingId: string;
  organizationId: string;
  revision: number;
  status: CapStatus;
  rootCause: string;
  correctiveAction: string;
  preventiveAction: string;
  responsiblePerson: string;
  targetCompletionDate: LocalDate;
  commentToCaa: string;
  submittedAt: Instant;
}

export interface CaaCapRevisionView extends CapRevisionSubmission {
  audience: "CAA";
  latestReview: null | {
    decision: "ACCEPT" | "REJECT" | "REQUEST_MORE_INFORMATION";
    commentToAuditee: string;
    internalCaaNote: string;
    decidedAt: Instant;
  };
}

export interface AuditeeCapRevisionView extends CapRevisionSubmission {
  audience: "AUDITEE";
  latestReview: null | {
    decision: "ACCEPT" | "REJECT" | "REQUEST_MORE_INFORMATION";
    commentToAuditee: string;
    decidedAt: Instant;
  };
}

export type CapRevisionView = CaaCapRevisionView | AuditeeCapRevisionView;

export interface ListCapRevisionsOutput {
  items: CapRevisionView[];
  nextCursor: null;
}

export interface BeginInspectionAttachmentUploadInput extends CommandMeta {
  inspectionAttachmentId: string;
  packageId: string;
  byteSize: number;
  sha256: string;
  fileName: string;
  declaredMediaType: string;
}

export interface BeginInspectionAttachmentUploadOutput {
  uploadId: string;
  stagingObjectKey: string;
  uploadUrl: string;
  requiredHeaders: Record<string, string>;
  expiresAt: Instant;
  maximumByteSize: number;
}

export interface CompleteInspectionAttachmentUploadInput extends CommandMeta {
  uploadId: string;
  sha256: string;
  byteSize: number;
}

export interface CompleteInspectionAttachmentUploadOutput {
  inspectionAttachmentId: string;
  uploadState: "UPLOADED";
  scanState: "PENDING";
}

export interface BeginEvidenceUploadInput extends CommandMeta {
  findingId: string;
  expectedFindingRevision: number;
  fileName: string;
  declaredMediaType: string;
  byteSize: number;
  sha256: string;
}

export interface BeginEvidenceUploadOutput {
  uploadId: string;
  stagingObjectKey: string;
  uploadUrl: string;
  requiredHeaders: Record<string, string>;
  expiresAt: Instant;
  maximumByteSize: number;
}

export interface CompleteEvidenceUploadInput extends CommandMeta {
  uploadId: string;
  sha256: string;
  byteSize: number;
}

export interface CompleteEvidenceUploadOutput {
  evidenceVersionId: string;
  version: number;
  uploadState: "UPLOADED";
  scanState: "PENDING" | "CLEAN";
  reviewState: EvidenceReviewState;
}

export interface EvidenceVersionView {
  id: string;
  findingId: string;
  organizationId: string;
  version: number;
  fileName: string;
  submittedAt: Instant;
  uploadState: EvidenceUploadState;
  scanState: EvidenceScanState;
  reviewState: EvidenceReviewState;
  revision: number;
}

export interface ReviewEvidenceInput extends CommandMeta {
  evidenceVersionId: string;
  expectedEvidenceVersionRevision: number;
  findingId: string;
  expectedFindingRevision: number;
  decision: "CLOSE" | "PARTIALLY_CLOSE" | "NOT_CLOSE" | "REQUEST_MORE_INFORMATION";
  commentToAuditee: string;
  internalCaaNote: string;
}

export interface ReviewEvidenceOutput {
  reviewDecisionId: string;
  evidenceVersionId: string;
  evidenceVersionRevision: number;
  findingStatus: FindingStatus;
  findingRevision: number;
}

export interface ReportVersionView {
  reportVersionId: string;
  reportId: string;
  organizationId: string;
  auditId: string;
  findingIds: string[];
  contentHash: string;
  version: number;
  status: ReportApprovalStatus;
  revision: number;
  issuedAt: Instant | null;
}

export interface DecideReportInput extends CommandMeta {
  reportVersionId: string;
  expectedReportVersionRevision: number;
  decision: "RETURN" | "FORWARD" | "ISSUE_AND_LOCK";
  reason: string;
}

export interface ManagerDashboardProjection {
  generatedAt: Instant;
  openFindings: number;
  closedFindings: number;
  overdueFindings: number;
  pendingCapReviews: number;
  pendingEvidenceReviews: number;
  pendingReportReviews: number;
  recentFindingNumbers: string[];
}

export interface OrganizationSummary {
  id: string;
  legalName: string;
  organizationType: string;
  status: string;
  openFindingCount: number;
  lastAuditDate: LocalDate | null;
  nextAuditDate: LocalDate | null;
  revision: number;
}

export interface ListOrganizationsOutput {
  items: OrganizationSummary[];
  nextCursor: string | null;
}

export type PlanningStatus =
  | "FINANCE_REVIEW"
  | "GM_REVIEW"
  | "EXECUTIVE_DIRECTOR_REVIEW"
  | "GM_RELEASE"
  | "RELEASED"
  | "RETURNED";

export type PlanningDecision =
  | "APPROVE_BUDGET"
  | "FORWARD_FOR_FINAL_APPROVAL"
  | "APPROVE_PLAN"
  | "RELEASE_PLAN"
  | "RETURN_FOR_REVISION";

export interface PlanningItemView {
  id: string;
  title: string;
  planYear: number;
  organizationId: string;
  organizationName: string;
  inspectionType: string;
  scheduledDate: LocalDate;
  estimatedBudget: number;
  status: PlanningStatus;
  currentOwnerRole: Role;
  nextAction: string;
  revision: number;
  submittedScopeSnapshotId?: string;
  planningSnapshotDigest?: string;
}

export type PlanningIntakeInspectionCategory = "Routine / Announced" | "Ad Hoc / Unannounced";
export type PlanningIntakeNoticePolicy = "ADVANCE" | "WITHHELD";

export interface PlanningIntakeDraftValues {
  organizationId: string;
  organizationName: string;
  applicationType: string;
  domain: string;
  inspectionCategory: PlanningIntakeInspectionCategory;
  noticePolicy: PlanningIntakeNoticePolicy;
  purpose: string;
  triggerType: string;
  riskCategory: string;
  plannedDate: LocalDate;
  mode: "On-site" | "Remote";
  location: string;
  /** Legacy fields are retained only for reading historical returned drafts. */
  templateVersionId?: string;
  scope?: string;
  catalogVersion?: string;
  scopeDraftId?: string;
  selectionDigest?: string;
  selectedQuestionVersionIds?: readonly string[];
  estimatedResourceRequirement?: number;
  formDistribution?: Readonly<Record<string, number>>;
  domainDistribution?: Readonly<Record<string, number>>;
  providerScopeId?: string;
  regulatedTargetId?: string;
  requestedBudget: number;
  currency: "USD" | "EUR" | "NAD";
}

export interface PlanningIntakeDraftView extends PlanningIntakeDraftValues {
  id: string;
  revision: number;
  submittedPlanningItemId: string | null;
  updatedAt: Instant;
}

export interface SavePlanningIntakeDraftInput extends RevisionedCommandMeta {
  draftId: string;
  values: PlanningIntakeDraftValues;
}

export interface SubmitPlanningIntakeInput extends RevisionedCommandMeta {
  draftId: string;
  planningItemId?: string;
}

export interface SubmitPlanningIntakeOutput {
  draft: PlanningIntakeDraftView;
  planningItem: PlanningItemView;
}

export interface ListPlanningItemsOutput {
  items: PlanningItemView[];
  nextCursor: string | null;
}

export interface PlanningDecisionInput extends CommandMeta {
  planningItemId: string;
  expectedPlanningRevision: number;
  decision: PlanningDecision;
  reason: string;
  expectedSubmittedScopeSnapshotId?: string;
  expectedPlanningSnapshotDigest?: string;
}

export interface ChecklistTemplateVersionView {
  id: string;
  templateId: string;
  title: string;
  version: number;
  status: "PUBLISHED";
  publishedAt: Instant;
  questionCount: number;
}

export interface ChecklistTemplateQuestionView {
  id: string;
  sectionId: string;
  prompt: string;
  regulatoryReference: string | null;
  expectedEvidence: string | null;
  allowedAnswers: ChecklistAnswer[];
  commentRequiredFor: ChecklistAnswer[];
}

export interface ChecklistTemplateVersionDetailView extends ChecklistTemplateVersionView {
  questions: ChecklistTemplateQuestionView[];
}

export interface ReminderRuleView {
  id: string;
  label: string;
  offsetDays: number;
  channel: "IN_APP";
  status: "ACTIVE";
  revision: number;
}

export interface AuditEventView {
  eventId: string;
  occurredAt: Instant;
  actorRole: Role | null;
  actorSubjectId: string | null;
  action: string;
  entityType: string;
  entityId: string;
  beforeStatus: string | null;
  afterStatus: string | null;
  reason: string | null;
  entityRevision: number | null;
}

export interface AdminRegulatorySourceView {
  id: string;
  title: string;
  sourceType: "ICAO_PQ" | "ANNEX_CROSSWALK" | "AUDIT_AREA_TAXONOMY" | "NATIONAL_LIBRARY" | "CAA_PROCEDURE";
  version: string;
  status: "SUPPLIED_WORKING_COPY" | "PUBLIC_REFERENCE" | "SOURCE_GAP";
  locator: string;
  url: string | null;
}

export interface AdminProposedInspectionQuestionView {
  id: string;
  prompt: string;
  verificationMethod: string;
  evidenceExamples: string[];
  whyIncluded: string;
}

export interface AdminRegulatoryRefreshPolicyView {
  sourceCollectionId: string;
  lastCheckedAt: Instant;
  nextReconciliationDate: LocalDate;
  nextExpertValidationDate: LocalDate;
  eventDrivenReview: true;
  reconciliationIntervalMonths: 6;
  expertValidationIntervalMonths: 12;
  sourceChangeState: "BASELINE_CAPTURED" | "NO_CHANGE" | "CHANGE_DETECTED" | "REVIEW_REQUIRED";
  updateMode: "PROPOSE_DRAFT_ONLY";
  documentCount: number;
  manifestPath: string;
  guardrails: string[];
}

export interface AdminChecklistQuestionScopeRecommendationView {
  questionId: string;
  classification: "MANDATORY_CORE" | "FOCUSED_FULL" | "ROTATIONAL_SAMPLE" | "DEFER_ELIGIBLE";
  rationale: string;
  historyBasis: string;
  requiresManagerApproval: true;
}

export interface AdminChecklistScopeRecommendationView {
  id: string;
  status: "ADVISORY_ONLY";
  historyState: "INSUFFICIENT_FOR_DEFERRAL" | "VALIDATED_HISTORY_AVAILABLE";
  generatedAt: Instant;
  signals: string[];
  guardrails: string[];
  questionRecommendations: AdminChecklistQuestionScopeRecommendationView[];
}

export interface AdminRegulatoryMappingView {
  id: string;
  auditArea: string;
  serviceProviderTypes: string[];
  applicableRegulations: string[];
  criticalElement: string;
  protocolQuestionId: string;
  protocolQuestion: string;
  annexReferences: string[];
  nationalReferences: string[];
  caaImplementationReference: string;
  requirement: string;
  verificationObjective: string;
  expectedEvidence: string[];
  whyIncluded: string;
  reviewStatus:
    | "TECHNICAL_REVIEW_REQUIRED"
    | "EXPERT_REVIEW_REQUIRED"
    | "VALIDATED"
    | "REJECTED";
  sourceGap: string | null;
  refreshPolicy: AdminRegulatoryRefreshPolicyView;
  scopeRecommendation: AdminChecklistScopeRecommendationView;
  sources: AdminRegulatorySourceView[];
  proposedQuestions: AdminProposedInspectionQuestionView[];
}

export interface AdminRegulatoryReferenceView {
  id: string;
  title: string;
  version: string;
  status: "ACTIVE" | "SUPERSEDED";
  effectiveDate: LocalDate;
  configuredRules: string[];
  changeHistory: string[];
  mappings: AdminRegulatoryMappingView[];
}

export type GovernedSourceSnapshotView = GeneratedSchemas["GovernedSourceSnapshotView"];
export type GovernedCandidateView = GeneratedSchemas["GovernedCandidateView"];
export type GovernedGenerationRunView = GeneratedSchemas["GovernedGenerationRunView"];
export type GovernedGenerationRequestInput = GeneratedSchemas["GovernedGenerationRequestInput"];
export type ValidateDepartmentManagerBlockedGenerationInput = GeneratedSchemas["ValidateDepartmentManagerBlockedGenerationInput"];
export type GovernedBlockedGenerationResult = GeneratedSchemas["GovernedBlockedGenerationResult"];
export type GovernedCandidateBundleInput = GeneratedSchemas["GovernedCandidateBundleInput"];
export type GovernedSourceCurrentnessActivationInput = GeneratedSchemas["GovernedSourceCurrentnessActivationInput"];
export type GovernedSourceCurrentnessActivationView = GeneratedSchemas["GovernedSourceCurrentnessActivationView"];
export type GovernedMappingView = GeneratedSchemas["GovernedMappingView"];
export type GovernedQuestionView = GeneratedSchemas["GovernedQuestionView"];
export type GovernedQuestionReconciliationView = GeneratedSchemas["GovernedQuestionReconciliationView"];
export type GovernedRequiredOwnerView = GeneratedSchemas["GovernedRequiredOwnerView"];
export type GovernedValidationIssue = GeneratedSchemas["GovernedValidationIssue"];
export type ImportAdminGovernedGenerationRunInput = GeneratedSchemas["ImportAdminGovernedGenerationRunInput"];
export type CreateAdminGovernedCandidateRevisionInput = GeneratedSchemas["CreateAdminGovernedCandidateRevisionInput"];
export type SubmitAdminGovernedCandidateReviewInput = GeneratedSchemas["SubmitAdminGovernedCandidateReviewInput"];
export type DepartmentManagerGovernedReviewCommandInput = GeneratedSchemas["DepartmentManagerGovernedReviewCommandInput"];
export type GovernedReviewDecisionView = GeneratedSchemas["GovernedReviewDecisionView"];
export type DepartmentManagerGovernedReviewItem = GeneratedSchemas["DepartmentManagerGovernedReviewItem"];
export type DepartmentManagerGovernedReviewQueue = GeneratedSchemas["DepartmentManagerGovernedReviewQueue"];
export type GovernedPublicationView = GeneratedSchemas["GovernedPublicationView"];
export type GovernedPublishedVersionView = GeneratedSchemas["GovernedPublishedVersionView"];
export type CreateChecklistImportBatchReceiptInput = GeneratedSchemas["CreateChecklistImportBatchReceiptInput"];
export type ChecklistImportBatchView = GeneratedSchemas["ChecklistImportBatchView"];
export type ChecklistImportBatchReceiptView = GeneratedSchemas["ChecklistImportBatchReceiptView"];
export type ChecklistImportFileView = GeneratedSchemas["ChecklistImportFileView"];
export type ChecklistImportFilePage = GeneratedSchemas["ChecklistImportFilePage"];
export type ChecklistImportReceiptPage = GeneratedSchemas["ChecklistImportReceiptPage"];
export type CreateChecklistImportFileExtractionReviewInput = GeneratedSchemas["CreateChecklistImportFileExtractionReviewInput"];
export type ChecklistImportExtractionReviewPage = GeneratedSchemas["ChecklistImportExtractionReviewPage"];
export type ChecklistImportExtractionReviewSummaryView = GeneratedSchemas["ChecklistImportExtractionReviewSummaryView"];
export type ResolveChecklistImportFileIdentityInput = GeneratedSchemas["ResolveChecklistImportFileIdentityInput"];
export type ExistingChecklistCandidateView = GeneratedSchemas["ExistingChecklistCandidateView"];
export type CreateExistingChecklistCandidateInput = GeneratedSchemas["CreateExistingChecklistCandidateInput"];
export type CreateDraftFromExistingChecklistCandidateInput = GeneratedSchemas["CreateDraftFromExistingChecklistCandidateInput"];
export type CreateOfficialSourceChecklistDraftInput = GeneratedSchemas["CreateOfficialSourceChecklistDraftInput"];
export type CreateHybridReconciledChecklistDraftInput = GeneratedSchemas["CreateHybridReconciledChecklistDraftInput"];
export type GovernedSourceReviewQueuePage = GeneratedSchemas["GovernedSourceReviewQueuePage"];
export type GovernedReviewerQueuePage = GeneratedSchemas["GovernedReviewerQueuePage"];
export type GovernedSourceAuthorityAttestationInput = GeneratedSchemas["GovernedSourceAuthorityAttestationInput"];
export type GovernedSourceAuthorityAttestationView = GeneratedSchemas["GovernedSourceAuthorityAttestationView"];
export type GovernedChecklistReviewCommentInput = GeneratedSchemas["GovernedChecklistReviewCommentInput"];
export type GovernedChecklistReviewCommentPage = GeneratedSchemas["GovernedChecklistReviewCommentPage"];
export type GovernedSourceMappingAttestationInput = GeneratedSchemas["GovernedSourceMappingAttestationInput"];
export type GovernedAuditPackageEligibilityInput = GeneratedSchemas["GovernedAuditPackageEligibilityInput"];
export type GovernedAuditPackageEligibilityView = GeneratedSchemas["GovernedAuditPackageEligibilityView"];
export type GovernedBackendCommandResult = GeneratedSchemas["GovernedBackendCommandResult"];

export interface AdminTemplateMasterView {
  id: string;
  title: string;
  publishedVersionId: string;
  status: "PUBLISHED";
  owner: "Department Manager";
  itemCount: number;
  previewPath: string | null;
  disabledReason: string | null;
  revision: number;
}

export interface AdminQuestionView {
  id: string;
  prompt: string;
  configuredReference: string;
  expectedEvidence: string;
  revision: number;
}

export interface AdminTemplateVersionView {
  id: string;
  templateId: "TPL-CABIN-2026";
  version: number;
  status: "PUBLISHED" | "DRAFT";
  owner: "Department Manager" | "Admin Preview";
  creatorSubjectId: string;
  changeReason: string;
  questionIds: string[];
  revision: number;
  createdAt: Instant;
}

export interface AdminTemplateView {
  id: "TPL-CABIN-2026";
  publishedVersionId: "CTV-CABIN-1";
  versions: AdminTemplateVersionView[];
  revision: number;
}

export interface AdminInspectionPackageView {
  id: "PKG-CAB-2026-001";
  auditId: "AUD-2026-001";
  organizationId: "ORG-FLY-NAMIBIA";
  organizationName: "Fly Namibia";
  questionIds: string[];
  configuredReferences: string[];
  expectedEvidence: string[];
  riskFocus: string[];
}

export interface AdminReportDefinitionView {
  id: string;
  title: string;
  description: string;
  packageFields: string[];
  actionReason: string;
}

export interface AdminAccessDirectoryEntryView {
  subjectId: string;
  displayName: string;
  roles: Role[];
  organizationId: string | null;
  email: string;
  mfaEnrolled: boolean;
  mfaState: string;
  requiredActions: string[];
  invitationState: string;
  accountStatus: string;
  applicationProfileState: string;
  membershipId: string | null;
  membershipState: string;
  membershipRevision: number;
  membershipDrift: string;
  lastSuccessfulSessionAt: string | null;
  providerObservedAt: string;
}

export type UserLifecycleAction =
  | "PROVISION"
  | "UPDATE_ROLES"
  | "SUSPEND"
  | "REACTIVATE"
  | "DEACTIVATE"
  | "TRANSFER_ORGANIZATION"
  | "RESEND_INVITATION"
  | "RESET_PASSWORD"
  | "RESET_MFA"
  | "FORCE_LOGOUT";

export type UserLifecycleStatus =
  | "PENDING"
  | "RUNNING"
  | "SUCCEEDED"
  | "FAILED"
  | "FAILED_RETRYABLE"
  | "FAILED_PERMANENT"
  | "MANUAL_REVIEW";

export interface RequestUserLifecycleInput {
  idempotencyKey: string;
  subjectId?: string | null;
  action: UserLifecycleAction;
  roles: Role[];
  organizationId: string;
  email?: string | null;
  displayName?: string | null;
  reason: string;
  expectedMembershipRevision: number;
  effectiveAt?: Instant | null;
}

export interface UserLifecycleRequestView {
  id: string;
  subjectId: string | null;
  action: UserLifecycleAction;
  roles: Role[];
  organizationId: string;
  email: string | null;
  displayName: string | null;
  status: UserLifecycleStatus;
  idempotencyKey: string;
  expectedMembershipRevision: number;
  resultingMembershipRevision: number;
  membershipId: string | null;
  reason: string;
  effectiveAt: Instant | null;
  providerFailureClass: "RETRYABLE" | "PERMANENT" | "MANUAL_REVIEW" | null;
  providerAcknowledgedAt: Instant | null;
  attemptCount: number;
  requestedBySubjectId: string;
  outboxMessageId: string;
  failureReason: string | null;
  createdAt: Instant;
  updatedAt: Instant;
}

export interface AdminOrganizationView {
  id: string;
  legalName: string;
  organizationType: string;
  status: string;
  scope: "CAA oversight";
  detailAvailable: boolean;
  disabledReason: string | null;
}

export interface PageOutput<T> {
  items: T[];
  nextCursor: string | null;
}

export type FieldCommandType =
  | "UPSERT_CHECKLIST_RESPONSE"
  | "CREATE_POTENTIAL_FINDING"
  | "SUBMIT_CHECKLIST"
  | "REGISTER_INSPECTION_ATTACHMENT";

export interface FieldOperationBase<TType extends FieldCommandType, TPayload> {
  operationId: string;
  protocolVersion: number;
  offlineGrantId: string;
  packageId: string;
  packageVersion: number;
  entityId: string;
  commandType: TType;
  baseRevision: number | null;
  deviceInstanceId: string;
  clientOccurredAt: Instant;
  payload: TPayload;
}

export type FieldSyncOperation =
  | FieldOperationBase<
      "UPSERT_CHECKLIST_RESPONSE",
      { auditId: string; questionId: string; answer: ChecklistAnswer; comment: string }
    >
  | FieldOperationBase<
      "CREATE_POTENTIAL_FINDING",
      {
        auditId: string;
        questionId: string;
        checklistResponseId: string;
        expectedChecklistResponseRevision: number | null;
        title: string;
        description: string;
        requiredComment: string;
        inspectionAttachmentIds: string[];
      }
    >
  | FieldOperationBase<"SUBMIT_CHECKLIST", { auditId: string }>
  | FieldOperationBase<
      "REGISTER_INSPECTION_ATTACHMENT",
      {
        auditId: string;
        checklistResponseId: string;
        potentialFindingOperationId: string | null;
        fileName: string;
        mediaType: string;
        byteSize: number;
        sha256: string;
      }
    >;

export interface PushFieldOperationRequest {
  operation: FieldSyncOperation;
}

export type PushFieldOperationStatus =
  | "accepted"
  | "already_applied"
  | "conflict"
  | "forbidden"
  | "invalid"
  | "retryable";

export interface AuthorizedConflictDescriptor {
  code: "STALE_REVISION" | "PACKAGE_REVOKED" | "ASSIGNMENT_CHANGED";
  entityId: string;
  authoritativeRevision: number | null;
  authoritativeStatus: string | null;
  changedAt: Instant | null;
}

export interface PushFieldOperationResult {
  operationId: string;
  status: PushFieldOperationStatus;
  authoritativeEntityId: string | null;
  authoritativeRevision: number | null;
  errorCode: string | null;
  conflict: AuthorizedConflictDescriptor | null;
  acknowledgedAt: Instant;
}

export interface SyncPullRequest {
  packageId: string;
  offlineGrantId: string;
  cursor: string | null;
  limit?: number;
}

export type AuthorizedSyncChange =
  | { kind: "checklist_response"; value: ChecklistResponseView }
  | { kind: "potential_finding"; value: PotentialFindingView }
  | { kind: "package_revoked"; packageId: string; reasonCode: string; revokedAt: Instant }
  | {
      kind: "tombstone";
      entityType: "checklist_response" | "potential_finding";
      entityId: string;
      revision: number;
    };

export interface SyncPullResponse {
  changes: AuthorizedSyncChange[];
  nextCursor: string | null;
  hasMore: boolean;
  resnapshotRequired: boolean;
  projectionVersion: number;
}

export interface AssignmentBackend {
  list(input: ListAssignmentsInput, options?: BackendRequestOptions): Promise<ListAssignmentsOutput>;
}

export interface InspectionBackend {
  start(
    input: { auditId: string; operationId: string; expectedInspectionRevision: number },
    options?: BackendRequestOptions,
  ): Promise<StartInspectionOutput>;
  getPackage(
    input: { packageId: string },
    options?: BackendRequestOptions,
  ): Promise<InspectionPackage>;
  checkout(
    input: CheckoutInspectionPackageInput,
    options?: BackendRequestOptions,
  ): Promise<CheckoutInspectionPackageOutput>;
  upsertChecklistResponse(
    input: UpsertChecklistResponseInput,
    options?: BackendRequestOptions,
  ): Promise<ChecklistResponseView>;
  submitChecklist(
    input: SubmitChecklistInput,
    options?: BackendRequestOptions,
  ): Promise<SubmitChecklistOutput>;
  reopenChecklist(
    input: ReopenChecklistInput,
    options?: BackendRequestOptions,
  ): Promise<SubmitChecklistOutput>;
}

export interface StartInspectionOutput {
  inspectionId: string;
  assignmentId: string;
  inspectionStatus: "IN_PROGRESS";
  assignmentStatus: string;
  inspectionRevision: number;
  checklistRevision: number;
  startedAt: Instant;
}

export interface PotentialFindingBackend {
  list(
    input: ListPotentialFindingsInput,
    options?: BackendRequestOptions,
  ): Promise<ListPotentialFindingsOutput>;
  get(
    input: { potentialFindingId: string },
    options?: BackendRequestOptions,
  ): Promise<PotentialFindingView>;
  create(
    input: CreatePotentialFindingInput,
    options?: BackendRequestOptions,
  ): Promise<PotentialFindingView>;
  decide(
    input: DecidePotentialFindingInput,
    options?: BackendRequestOptions,
  ): Promise<PotentialFindingDecisionOutput>;
}

export interface FindingBackend {
  list(input: ListFindingsInput, options?: BackendRequestOptions): Promise<ListFindingsOutput>;
  get(
    input: { findingId: string },
    options?: BackendRequestOptions,
  ): Promise<FindingView>;
  authorizedClose(
    input: AuthorizedCloseInput,
    options?: BackendRequestOptions,
  ): Promise<FindingView>;
}

export interface CapBackend {
  listRevisions(
    input: { findingId: string },
    options?: BackendRequestOptions,
  ): Promise<ListCapRevisionsOutput>;
  getRevision(
    input: { capRevisionId: string },
    options?: BackendRequestOptions,
  ): Promise<CapRevisionView>;
  submit(input: SubmitCapInput, options?: BackendRequestOptions): Promise<SubmitCapOutput>;
  review(input: ReviewCapInput, options?: BackendRequestOptions): Promise<ReviewCapOutput>;
}

export interface InspectionAttachmentBackend {
  beginUpload(
    input: BeginInspectionAttachmentUploadInput,
    options?: BackendRequestOptions,
  ): Promise<BeginInspectionAttachmentUploadOutput>;
  completeUpload(
    input: CompleteInspectionAttachmentUploadInput,
    options?: BackendRequestOptions,
  ): Promise<CompleteInspectionAttachmentUploadOutput>;
}

export interface EvidenceBackend {
  beginUpload(
    input: BeginEvidenceUploadInput,
    options?: BackendRequestOptions,
  ): Promise<BeginEvidenceUploadOutput>;
  completeUpload(
    input: CompleteEvidenceUploadInput,
    options?: BackendRequestOptions,
  ): Promise<CompleteEvidenceUploadOutput>;
  listVersions(
    input: { findingId: string },
    options?: BackendRequestOptions,
  ): Promise<EvidenceVersionView[]>;
  review(
    input: ReviewEvidenceInput,
    options?: BackendRequestOptions,
  ): Promise<ReviewEvidenceOutput>;
}

export interface ReportBackend {
  getVersion(
    input: { reportVersionId: string },
    options?: BackendRequestOptions,
  ): Promise<ReportVersionView>;
  decide(input: DecideReportInput, options?: BackendRequestOptions): Promise<ReportVersionView>;
}

export interface DashboardBackend {
  getManagerProjection(
    input: { organizationId?: string },
    options?: BackendRequestOptions,
  ): Promise<ManagerDashboardProjection>;
}

export interface OrganizationBackend {
  list(
    input: { limit?: number },
    options?: BackendRequestOptions,
  ): Promise<ListOrganizationsOutput>;
}

export interface PlanningBackend {
  list(
    input: { limit?: number },
    options?: BackendRequestOptions,
  ): Promise<ListPlanningItemsOutput>;
  decide(
    input: PlanningDecisionInput,
    options?: BackendRequestOptions,
  ): Promise<PlanningItemView>;
}

export interface PlanningIntakeBackend {
  createDraft(
    input: {
      values: PlanningIntakeDraftValues;
      draftId?: string;
    } & CommandMeta & OptionalRevisionedCommandMeta,
    options?: BackendRequestOptions,
  ): Promise<PlanningIntakeDraftView>;
  getDraft(
    input: { draftId: string },
    options?: BackendRequestOptions,
  ): Promise<PlanningIntakeDraftView>;
  saveDraft(
    input: SavePlanningIntakeDraftInput,
    options?: BackendRequestOptions,
  ): Promise<PlanningIntakeDraftView>;
  submit(
    input: SubmitPlanningIntakeInput,
    options?: BackendRequestOptions,
  ): Promise<SubmitPlanningIntakeOutput>;
}

export type CanonicalQuestionUsageClass = "GOVERNED_OPERATIONAL" | "PREPROD_EXERCISE";
export type CanonicalQuestionReviewAction =
  | "RETAIN" | "INCLUDE" | "EXCLUDE" | "DEFER"
  | "DOMAIN_RECLASSIFIED" | "TOPIC_RECLASSIFIED"
  | "TECHNICAL_APPROVE" | "PUBLISH";

export interface CanonicalAuditScopeOption {
  organizationId: string;
  organizationName: string;
  providerScopeId: string;
  regulatedTargetId: string;
  providerTypeId: string;
  providerTypeLabel: string;
  scopeStatus: "ACTIVE";
  targetLabel: string;
  catalogVersion: string;
  usageClass: CanonicalQuestionUsageClass;
  inspectionTypes: Array<"RAMP" | "CABIN" | "RAMP_INSPECTION" | "CABIN_INSPECTION">;
}

export interface CanonicalAuditScopeOptionPage {
  items: CanonicalAuditScopeOption[];
  nextCursor: string | null;
}

export interface CanonicalQuestionCatalogEntry {
  catalogVersion: string;
  usageClass: CanonicalQuestionUsageClass;
  questionVersionId: string;
  formCode: string;
  proposalId: string;
  ordinal: number;
  questionDigest: string;
  prompt: string | null;
  configuredReference: string | null;
  expectedEvidence: string | null;
  sourceLocator: string | null;
  sourceGapState: string;
  proposedDomain: string | null;
  proposedTopic: string | null;
  proposedRiskBand: string | null;
  canSelect: boolean;
  canPublish: boolean;
  governedCandidateId: string | null;
  governedCandidateRevision: number | null;
  governedCandidateContentDigest: string | null;
  governedCandidateStatus: string | null;
  reviewRevision: number;
  reviewDisposition: string | null;
  reviewDigest: string | null;
  reviewHistory?: GeneratedSchemas["QuestionReviewHistoryItem"][];
}

export interface CanonicalQuestionCatalogPage {
  items: CanonicalQuestionCatalogEntry[];
  nextCursor: string | null;
  catalogVersion: string;
  usageClass: CanonicalQuestionUsageClass;
  totalCount: number;
}

export interface CanonicalSelectionDigest {
  selectionDigest: string;
  selectedQuestionVersionIds: string[];
  selectedCount: number;
  catalogVersion: string;
  usageClass: CanonicalQuestionUsageClass;
  formDistribution: Readonly<Record<string, number>>;
  domainDistribution: Readonly<Record<string, number>>;
  estimatedResourceRequirement: number;
}

export interface CanonicalSelectionPreview {
  preview: CanonicalSelectionDigest;
  affectedCount: number;
  valid: boolean;
  reason: string;
}

export interface CanonicalSelectionReceipt {
  operationId: string;
  replayed: boolean;
  selection: CanonicalSelectionDigest;
}

export interface CanonicalQuestionReviewQueue {
  mode: CanonicalQuestionUsageClass;
  items: CanonicalQuestionCatalogEntry[];
  nextCursor: string | null;
  totalCount: number;
  capabilities: {
    canTechnicalApprove: boolean;
    canPublish: boolean;
    disabledReason: string | null;
  };
}

export interface CanonicalExerciseQuestionReviewCommandInput extends CommandMeta {
  idempotencyKey?: string;
  mode: "PREPROD_EXERCISE";
  catalogVersion: string;
  scopeId: string;
  questionVersionId: string;
  expectedRevision: number;
  expectedReviewDigest: string;
  action: Exclude<CanonicalQuestionReviewAction, "TECHNICAL_APPROVE" | "PUBLISH">;
  reason: string;
  domain?: string | null;
  topic?: string | null;
}

export interface CanonicalGovernedQuestionReviewCommandInput extends CommandMeta {
  idempotencyKey?: string;
  mode: "GOVERNED_OPERATIONAL";
  catalogVersion: string;
  questionVersionId: string;
  candidateId: string;
  expectedCandidateRevision: number;
  expectedCandidateContentDigest: string;
  action: CanonicalQuestionReviewAction;
  reason: string;
  domain?: string | null;
  topic?: string | null;
}

export type CanonicalQuestionReviewCommandInput =
  | CanonicalExerciseQuestionReviewCommandInput
  | CanonicalGovernedQuestionReviewCommandInput;

export interface CanonicalQuestionReviewCommandOutput {
  operationId: string;
  mode: CanonicalQuestionUsageClass;
  questionVersionId: string;
  action: string;
  replayed: boolean;
  canPublish: boolean;
  currentCandidateId?: string | null;
  currentCandidateRevision?: number | null;
  currentCandidateContentDigest?: string | null;
}

export interface CanonicalQuestionReviewBackend {
  listScopeOptions(input?: { cursor?: string; limit?: number; catalogVersion?: string; usageClass?: CanonicalQuestionUsageClass; forReview?: boolean }, options?: BackendRequestOptions): Promise<CanonicalAuditScopeOptionPage>;
  listCatalog(input: { catalogVersion: string; usageClass: CanonicalQuestionUsageClass; search?: string; formCode?: string; domain?: string; topic?: string; riskBand?: string; sourceGapState?: string; selected?: "all" | "selected" | "unselected"; scopeId?: string; cursor?: string; limit?: number }, options?: BackendRequestOptions): Promise<CanonicalQuestionCatalogPage>;
  getQuestion(input: { catalogVersion: string; usageClass: CanonicalQuestionUsageClass; questionVersionId: string; scopeId?: string }, options?: BackendRequestOptions): Promise<CanonicalQuestionCatalogEntry>;
  previewSelection(input: { scopeId: string; operationId: string; idempotencyKey?: string; expectedSelectionDigest: string; questionVersionIds: string[]; operationKind?: "ADD" | "REMOVE" | "REPLACE"; usageClass: CanonicalQuestionUsageClass; filter?: Record<string, never> }, options?: BackendRequestOptions): Promise<CanonicalSelectionPreview>;
  commitSelection(input: { scopeId: string; operationId: string; previewOperationId: string; idempotencyKey?: string; expectedSelectionDigest: string; questionVersionIds: string[]; operationKind?: "ADD" | "REMOVE" | "REPLACE"; usageClass: CanonicalQuestionUsageClass; filter?: Record<string, never> }, options?: BackendRequestOptions): Promise<CanonicalSelectionReceipt>;
  reviewQueue(input: { mode: CanonicalQuestionUsageClass; catalogVersion: string; search?: string; formCode?: string; domain?: string; topic?: string; riskBand?: string; sourceGapState?: string; selected?: "all" | "selected" | "unselected"; scopeId?: string; cursor?: string; limit?: number }, options?: BackendRequestOptions): Promise<CanonicalQuestionReviewQueue>;
  command(input: CanonicalQuestionReviewCommandInput, options?: BackendRequestOptions): Promise<CanonicalQuestionReviewCommandOutput>;
}

/**
 * The post-release canonical preparation boundary.  UI surfaces must use
 * these server-owned commands instead of constructing audit/package IDs or
 * keeping assignment state in a local demo store.
 */
export type CanonicalPreparationView = GeneratedSchemas["PreparationView"];
export type CanonicalAssignmentView = GeneratedSchemas["CanonicalAssignmentView"];
export type CanonicalPreparationConfirmationView = GeneratedSchemas["PreparationConfirmationView"];
export type CanonicalMaterializedAuditView = GeneratedSchemas["CanonicalMaterializedAuditView"];
export type CanonicalPreparationEditPreviewView = GeneratedSchemas["PreparationEditPreviewView"];

export interface CanonicalAuditWorkflowBackend {
  getPreparation(input?: { assignmentId?: string; planningItemId?: string }, options?: BackendRequestOptions): Promise<CanonicalAssignmentView>;
  prepare(
    planningItemId: string,
    input: GeneratedSchemas["PrepareAuditInput"],
    options?: BackendRequestOptions,
  ): Promise<CanonicalPreparationView>;
  assignLead(
    assignmentId: string,
    input: GeneratedSchemas["AssignLeadInput"],
    options?: BackendRequestOptions,
  ): Promise<CanonicalAssignmentView>;
  previewTeam(
    assignmentId: string,
    input: GeneratedSchemas["PreviewAssignTeamInput"],
    options?: BackendRequestOptions,
  ): Promise<CanonicalPreparationEditPreviewView>;
  assignTeam(
    assignmentId: string,
    input: GeneratedSchemas["AssignTeamInput"],
    options?: BackendRequestOptions,
  ): Promise<CanonicalAssignmentView>;
  previewQuestionCoverage(
    assignmentId: string,
    input: GeneratedSchemas["PreviewAssignQuestionsInput"],
    options?: BackendRequestOptions,
  ): Promise<CanonicalPreparationEditPreviewView>;
  assignQuestionCoverage(
    assignmentId: string,
    input: GeneratedSchemas["AssignQuestionsInput"],
    options?: BackendRequestOptions,
  ): Promise<CanonicalAssignmentView>;
  confirmPreparation(
    assignmentId: string,
    input: GeneratedSchemas["ConfirmAuditPreparationInput"],
    options?: BackendRequestOptions,
  ): Promise<CanonicalPreparationConfirmationView>;
  materialize(
    assignmentId: string,
    input: GeneratedSchemas["MaterializeCanonicalAuditInput"],
    options?: BackendRequestOptions,
  ): Promise<CanonicalMaterializedAuditView>;
  start(
    auditId: string,
    input: GeneratedSchemas["StartInspectionInput"],
    options?: BackendRequestOptions,
  ): Promise<GeneratedSchemas["StartInspectionOutput"]>;
}

export interface ConfigurationBackend {
  listChecklistTemplateVersions(
    input: { limit?: number },
    options?: BackendRequestOptions,
  ): Promise<PageOutput<ChecklistTemplateVersionView>>;
  getChecklistTemplateVersion(
    input: { templateVersionId: string },
    options?: BackendRequestOptions,
  ): Promise<ChecklistTemplateVersionDetailView>;
  listReminderRules(
    input: { limit?: number },
    options?: BackendRequestOptions,
  ): Promise<PageOutput<ReminderRuleView>>;
}

export interface AuditTrailBackend {
  list(
    input: { entityType?: string; entityId?: string; limit?: number },
    options?: BackendRequestOptions,
  ): Promise<PageOutput<AuditEventView>>;
}

export interface SyncBackend {
  pushOperation(
    input: PushFieldOperationRequest,
    options?: BackendRequestOptions,
  ): Promise<PushFieldOperationResult>;
  pull(input: SyncPullRequest, options?: BackendRequestOptions): Promise<SyncPullResponse>;
}

export interface CommunicationsBackend {
  list(input: { organizationId?: string }, options?: BackendRequestOptions): Promise<PageOutput<CommunicationView>>;
  send(input: RevisionedCommandMeta & { organizationId: string | null; subject: string; body: string; audience: "CAA" | "AUDITEE" }, options?: BackendRequestOptions): Promise<CommunicationView>;
}

export interface CalendarBackend {
  list(input: { organizationId?: string }, options?: BackendRequestOptions): Promise<PageOutput<CalendarItemView>>;
  openItem(input: { calendarItemId: string }, options?: BackendRequestOptions): Promise<CalendarItemView>;
}

export interface ProfilesBackend {
  getMine(input: Record<string, never>, options?: BackendRequestOptions): Promise<ProfileView>;
  updateMine(input: RevisionedCommandMeta & { displayName: string }, options?: BackendRequestOptions): Promise<ProfileView>;
}

export interface TeamsBackend {
  list(input: { role?: Role }, options?: BackendRequestOptions): Promise<PageOutput<TeamMemberView>>;
  openMember(input: { subjectId: string }, options?: BackendRequestOptions): Promise<TeamMemberView>;
  listAuditTeams(input: { limit?: number }, options?: BackendRequestOptions): Promise<PageOutput<InspectionTeamAuditView>>;
  openAuditTeam(input: { auditId: string }, options?: BackendRequestOptions): Promise<InspectionTeamAuditView>;
}

export interface RiskBackend {
  getOverview(input: { organizationId?: string }, options?: BackendRequestOptions): Promise<RiskOverviewView>;
  getManagementProjection(input: Record<string, never>, options?: BackendRequestOptions): Promise<RiskManagementProjectionView>;
  openFinding(input: { findingId: string }, options?: BackendRequestOptions): Promise<FindingView>;
}

export interface DocumentsBackend {
  list(input: { organizationId?: string }, options?: BackendRequestOptions): Promise<PageOutput<DocumentMetadataView>>;
  open(input: { documentId: string }, options?: BackendRequestOptions): Promise<DocumentMetadataView>;
}

export interface AuditeeCoordinationBackend {
  list(
    input: Record<string, never>,
    options?: BackendRequestOptions,
  ): Promise<PageOutput<AuditeeCoordinationView>>;
  respond(
    input: RespondToAuditeeCoordinationInput,
    options?: BackendRequestOptions,
  ): Promise<AuditeeCoordinationView>;
  review?(
    input: GeneratedSchemas["ReviewAuditeeCoordinationInput"],
    options?: BackendRequestOptions,
  ): Promise<AuditeeCoordinationView>;
}

export interface AuditeeReportsBackend {
  listReleased(
    input: { kind?: AuditeeReleasedReportView["kind"] },
    options?: BackendRequestOptions,
  ): Promise<PageOutput<AuditeeReleasedReportView>>;
  getReleased(
    input: { reportVersionId: string },
    options?: BackendRequestOptions,
  ): Promise<AuditeeReleasedReportView>;
}

export interface NotificationsBackend {
  list(input: Record<string, never>, options?: BackendRequestOptions): Promise<PageOutput<NotificationView>>;
  markRead(input: RevisionedCommandMeta & { notificationId: string }, options?: BackendRequestOptions): Promise<NotificationView>;
}

export interface AdministrationBackend {
  getScreenProjection(input: { screenId: string }, options?: BackendRequestOptions): Promise<AdministrationScreenProjection>;
  listScreenProjections(input: Record<string, never>, options?: BackendRequestOptions): Promise<AdministrationScreenProjection[]>;
  invokeVisibleAction(input: { screenId: string; actionId: string }, options?: BackendRequestOptions): Promise<VisibleActionResult>;
}

export interface AdminWorkspaceBackend {
  listGovernedSources(input: Record<string, never>, options?: BackendRequestOptions): Promise<PageOutput<GovernedSourceSnapshotView>>;
  activateGovernedSourceCurrentness(input: GovernedSourceCurrentnessActivationInput, options?: BackendRequestOptions): Promise<GovernedSourceCurrentnessActivationView>;
  importGovernedGenerationRun(input: ImportAdminGovernedGenerationRunInput, options?: BackendRequestOptions): Promise<GovernedGenerationRunView>;
  getGovernedGenerationRun(input: { generationRunId: string }, options?: BackendRequestOptions): Promise<GovernedGenerationRunView>;
  getGovernedCandidate(input: { candidateId: string }, options?: BackendRequestOptions): Promise<GovernedCandidateView>;
  createGovernedCandidateRevision(input: CreateAdminGovernedCandidateRevisionInput, options?: BackendRequestOptions): Promise<GovernedCandidateView>;
  submitGovernedCandidateReview(input: SubmitAdminGovernedCandidateReviewInput, options?: BackendRequestOptions): Promise<GovernedCandidateView>;
  listRegulatoryReferences(input: { search?: string; status?: string }, options?: BackendRequestOptions): Promise<PageOutput<AdminRegulatoryReferenceView>>;
  listTemplateMasters(input: Record<string, never>, options?: BackendRequestOptions): Promise<PageOutput<AdminTemplateMasterView>>;
  listQuestions(input: { search?: string }, options?: BackendRequestOptions): Promise<PageOutput<AdminQuestionView>>;
  createQuestion(input: RevisionedCommandMeta & { prompt: string; configuredReference: string; expectedEvidence: string }, options?: BackendRequestOptions): Promise<AdminQuestionView>;
  getTemplate(input: { templateId: string }, options?: BackendRequestOptions): Promise<AdminTemplateView>;
  createDraft(input: RevisionedCommandMeta & { templateId: string; changeReason: string }, options?: BackendRequestOptions): Promise<AdminTemplateVersionView>;
  addDraftQuestion(input: RevisionedCommandMeta & { templateId: string; draftVersionId: string; questionId: string }, options?: BackendRequestOptions): Promise<AdminTemplateVersionView>;
  moveDraftQuestion(input: RevisionedCommandMeta & { templateId: string; draftVersionId: string; questionId: string; direction: "UP" | "DOWN" }, options?: BackendRequestOptions): Promise<AdminTemplateVersionView>;
  getInspectionPackage(input: { packageId: string }, options?: BackendRequestOptions): Promise<AdminInspectionPackageView>;
  listReportDefinitions(input: { search?: string }, options?: BackendRequestOptions): Promise<PageOutput<AdminReportDefinitionView>>;
  listAccessDirectory(input: { search?: string; role?: Role | string; organizationId?: string; accountStatus?: "enabled" | "disabled"; cursor?: string; limit?: number }, options?: BackendRequestOptions): Promise<PageOutput<AdminAccessDirectoryEntryView>>;
  requestUserLifecycle(input: RequestUserLifecycleInput, options?: BackendRequestOptions): Promise<UserLifecycleRequestView>;
  getUserLifecycleRequest(input: { requestId: string }, options?: BackendRequestOptions): Promise<UserLifecycleRequestView>;
  listOrganizations(input: { search?: string; organizationType?: string; status?: string; scope?: string }, options?: BackendRequestOptions): Promise<PageOutput<AdminOrganizationView>>;
  getOrganization(input: { organizationId: string }, options?: BackendRequestOptions): Promise<AdminOrganizationView>;
  listAuditEvents(input: { actor?: string; action?: string; entity?: string; system?: string; dateText?: string }, options?: BackendRequestOptions): Promise<PageOutput<AuditEventView>>;
}

export interface GovernedChecklistReviewBackend {
  validateBlockedGeneration(input: ValidateDepartmentManagerBlockedGenerationInput, options?: BackendRequestOptions): Promise<GovernedBlockedGenerationResult>;
  listQueue(input: Record<string, never>, options?: BackendRequestOptions): Promise<DepartmentManagerGovernedReviewQueue>;
  getCandidate(input: { candidateId: string }, options?: BackendRequestOptions): Promise<DepartmentManagerGovernedReviewItem>;
  return(input: DepartmentManagerGovernedReviewCommandInput, options?: BackendRequestOptions): Promise<GovernedCandidateView>;
  reject(input: DepartmentManagerGovernedReviewCommandInput, options?: BackendRequestOptions): Promise<GovernedCandidateView>;
  approve(input: DepartmentManagerGovernedReviewCommandInput, options?: BackendRequestOptions): Promise<GovernedCandidateView>;
  publish(input: DepartmentManagerGovernedReviewCommandInput, options?: BackendRequestOptions): Promise<GovernedPublicationView>;
  getPublishedVersion(input: { templateVersionId: string }, options?: BackendRequestOptions): Promise<GovernedPublishedVersionView>;
}

/** Candidate-only intake and source-review surface. Admin inventory data never
 * appears in auditee projections; every command is server-authorized. */
export interface GovernedChecklistIntakeBackend {
  receiveBatch(input: CreateChecklistImportBatchReceiptInput & { archive: Blob | Uint8Array }, options?: BackendRequestOptions): Promise<ChecklistImportBatchReceiptView>;
  getBatch(input: { importBatchId: string }, options?: BackendRequestOptions): Promise<ChecklistImportBatchView>;
  listFiles(input: { importBatchId: string; cursor?: string; limit?: number }, options?: BackendRequestOptions): Promise<ChecklistImportFilePage>;
  listReceipts(input: { importBatchId: string; cursor?: string; limit?: number }, options?: BackendRequestOptions): Promise<ChecklistImportReceiptPage>;
  createExtractionReview(input: CreateChecklistImportFileExtractionReviewInput, options?: BackendRequestOptions): Promise<ChecklistImportExtractionReviewSummaryView>;
  getExtractionReview(input: { importBatchId: string; importFileId: string; cursor?: string; limit?: number }, options?: BackendRequestOptions): Promise<ChecklistImportExtractionReviewPage>;
  resolveIdentity(input: ResolveChecklistImportFileIdentityInput, options?: BackendRequestOptions): Promise<ChecklistImportFileView>;
  importCandidate(input: CreateExistingChecklistCandidateInput, options?: BackendRequestOptions): Promise<GovernedBackendCommandResult>;
  listSourceReviewQueue(input: { cursor?: string; limit?: number }, options?: BackendRequestOptions): Promise<GovernedSourceReviewQueuePage>;
  getSourceReviewItem(input: { reviewItemId: string }, options?: BackendRequestOptions): Promise<GovernedSourceReviewQueuePage["items"][number]>;
  listReviewerQueue(input: { cursor?: string; limit?: number }, options?: BackendRequestOptions): Promise<GovernedReviewerQueuePage>;
  attestSourceAuthority(input: GovernedSourceAuthorityAttestationInput, options?: BackendRequestOptions): Promise<GovernedSourceAuthorityAttestationView>;
  getExistingCandidate(input: { existingCandidateId: string }, options?: BackendRequestOptions): Promise<ExistingChecklistCandidateView>;
  createDraftFromExisting(input: CreateDraftFromExistingChecklistCandidateInput, options?: BackendRequestOptions): Promise<GovernedBackendCommandResult>;
  createOfficialSourceDraft(input: CreateOfficialSourceChecklistDraftInput, options?: BackendRequestOptions): Promise<GovernedBackendCommandResult>;
  getDraft(input: { candidateId: string }, options?: BackendRequestOptions): Promise<GeneratedSchemas["GovernedCandidateDetailView"]>;
  createHybridReconciliation(input: CreateHybridReconciledChecklistDraftInput, options?: BackendRequestOptions): Promise<GovernedBackendCommandResult>;
  listReviewComments(input: { candidateId: string; cursor?: string; limit?: number }, options?: BackendRequestOptions): Promise<GovernedChecklistReviewCommentPage>;
  createReviewComment(input: GovernedChecklistReviewCommentInput, options?: BackendRequestOptions): Promise<GeneratedSchemas["GovernedChecklistReviewCommentView"]>;
  attestSourceMapping(input: GovernedSourceMappingAttestationInput, options?: BackendRequestOptions): Promise<GeneratedSchemas["GovernedSourceMappingAttestationView"]>;
  evaluateAuditPackageEligibility(input: GovernedAuditPackageEligibilityInput, options?: BackendRequestOptions): Promise<GovernedAuditPackageEligibilityView>;
}

export interface AssistantDraftsBackend {
  getGuidance(input: Record<string, never>, options?: BackendRequestOptions): Promise<{ advisoryOnly: true; prohibitedActions: readonly string[] }>;
  createDraft(input: { findingId: string; prompt: string }, options?: BackendRequestOptions): Promise<AssistantDraftView>;
}

export interface Backend {
  readonly mode: BackendMode;
  readonly assignments: AssignmentBackend;
  readonly inspections: InspectionBackend;
  readonly potentialFindings: PotentialFindingBackend;
  readonly findings: FindingBackend;
  readonly caps: CapBackend;
  readonly inspectionAttachments: InspectionAttachmentBackend;
  readonly evidence: EvidenceBackend;
  readonly reports: ReportBackend;
  readonly dashboards: DashboardBackend;
  readonly organizations: OrganizationBackend;
  readonly planning: PlanningBackend;
  readonly configuration: ConfigurationBackend;
  readonly auditTrail: AuditTrailBackend;
  readonly sync: SyncBackend;
  readonly communications: CommunicationsBackend;
  readonly calendar: CalendarBackend;
  readonly profiles: ProfilesBackend;
  readonly teams: TeamsBackend;
  readonly risk: RiskBackend;
  readonly documents: DocumentsBackend;
  readonly notifications: NotificationsBackend;
  readonly administration: AdministrationBackend;
  readonly adminWorkspace: AdminWorkspaceBackend;
  readonly governedChecklistReview: GovernedChecklistReviewBackend;
  readonly governedChecklistIntake: GovernedChecklistIntakeBackend;
  readonly assistantDrafts: AssistantDraftsBackend;
  readonly planningIntake: PlanningIntakeBackend;
  readonly canonicalQuestionReview?: CanonicalQuestionReviewBackend;
  readonly canonicalAuditWorkflow?: CanonicalAuditWorkflowBackend;
  /** Fail-closed Auditee coordination projection and command boundary. */
  readonly auditeeCoordination: AuditeeCoordinationBackend;
  /** LOCKED-only Auditee report projection. */
  readonly auditeeReports: AuditeeReportsBackend;
}

export const BACKEND_CAPABILITY_REGISTRY = {
  assignments: true,
  inspections: true,
  potentialFindings: true,
  findings: true,
  caps: true,
  inspectionAttachments: true,
  evidence: true,
  reports: true,
  dashboards: true,
  organizations: true,
  planning: true,
  configuration: true,
  auditTrail: true,
  sync: true,
  communications: true,
  calendar: true,
  profiles: true,
  teams: true,
  risk: true,
  documents: true,
  notifications: true,
  administration: true,
  adminWorkspace: true,
  governedChecklistReview: true,
  governedChecklistIntake: true,
  assistantDrafts: true,
  planningIntake: true,
  canonicalQuestionReview: true,
  canonicalAuditWorkflow: true,
  auditeeCoordination: true,
  auditeeReports: true,
} as const satisfies Record<Exclude<keyof Backend, "mode">, true>;

export const BACKEND_CAPABILITY_KEYS = Object.keys(
  BACKEND_CAPABILITY_REGISTRY,
) as Array<keyof typeof BACKEND_CAPABILITY_REGISTRY>;

export type DemoBackend = Backend;
