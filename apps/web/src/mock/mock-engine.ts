import type {
  Backend,
  BackendPrincipal,
  DemoBackend,
  AuditeeCoordinationView,
  AuditeeReleasedReportView,
  CommunicationView,
  DocumentMetadataView,
  NotificationView,
  ProfileView,
  RiskManagementProjectionView,
  RiskOverviewView,
  TeamMemberView,
  CapRevisionView,
  ChecklistResponseView,
  ChecklistAnswer,
  EvidenceReviewState,
  FindingStatus,
  FindingView,
  InspectionPackage,
  InspectionTeamAuditView,
  PlanningIntakeDraftView,
  PotentialFindingView,
  GovernedCandidateBundleInput,
  GovernedSourceCurrentnessActivationInput,
  GovernedSourceCurrentnessActivationView,
  GovernedSourceSnapshotView,
  GovernedBlockedGenerationResult,
  GovernedCandidateView,
  GovernedGenerationRunView,
  GovernedMappingView,
  GovernedQuestionView,
  GovernedRequiredOwnerView,
  GovernedValidationIssue,
  GovernedReviewDecisionView,
  GovernedPublicationView,
  GovernedPublishedVersionView,
  DepartmentManagerGovernedReviewCommandInput,
  GovernedChecklistIntakeBackend,
  CreateChecklistImportBatchReceiptInput,
  ChecklistImportBatchReceiptView,
  ChecklistImportBatchView,
  ChecklistImportFilePage,
  ChecklistImportReceiptPage,
  CreateChecklistImportFileExtractionReviewInput,
  ChecklistImportExtractionReviewPage,
  ChecklistImportExtractionReviewSummaryView,
  ResolveChecklistImportFileIdentityInput,
  CreateExistingChecklistCandidateInput,
  GovernedBackendCommandResult,
  GovernedSourceReviewQueuePage,
  GovernedReviewerQueuePage,
  GovernedSourceAuthorityAttestationInput,
  GovernedSourceAuthorityAttestationView,
  ExistingChecklistCandidateView,
  CreateDraftFromExistingChecklistCandidateInput,
  CreateOfficialSourceChecklistDraftInput,
  CreateHybridReconciledChecklistDraftInput,
  GovernedChecklistReviewCommentInput,
  GovernedChecklistReviewCommentPage,
  GovernedSourceMappingAttestationInput,
  GovernedAuditPackageEligibilityInput,
  GovernedAuditPackageEligibilityView,
  CanonicalQuestionReviewBackend,
  CanonicalQuestionCatalogEntry,
  CanonicalQuestionUsageClass,
  CanonicalQuestionReviewCommandInput,
} from "../backend/backend";
import {
  BackendAuthorizationInvariantError,
  BackendConflictError,
  BackendInvariantError,
  GovernedValidationError,
  OperationIdReuseError,
  requireNonEmpty,
  requireDemoCapability,
  requireRevision,
  requireRole,
} from "../backend/backend-contracts";
import {
  SYNTHETIC_EDITED_RATIONALE,
  EXACT_BLOCKED_REAL_OPS_AOC_REQUEST,
  SYNTHETIC_FAILED_REQUEST_ID,
  SYNTHETIC_FAILED_RUN_ID,
  SYNTHETIC_GOVERNED_BUNDLE,
	SYNTHETIC_HYBRID_RECONCILED_BUNDLE,
  SYNTHETIC_INPUT_DIGEST,
	SYNTHETIC_LEGACY_CHECKLIST_CANDIDATE_BUNDLE,
  SYNTHETIC_OUTPUT_DIGEST,
  SYNTHETIC_SOURCE_HASH,
} from "../backend/governed-synthetic-profile";
import {
  governedCandidateContentDigest,
  governedCanonicalJSON,
  governedCanonicalSHA256,
  governedEditSemanticDigest,
  governedImportSemanticDigest,
  governedSubmitSemanticDigest,
} from "../backend/governed-canonical";
import { MemoryMockStore } from "./memory-mock-store";
import { REACT_ROUTE_CONTRACTS } from "../app/route-contracts";
import {
  publicEvidenceVersion,
  type MockEvidenceUpload,
  type MockEvidenceVersion,
  type MockInspectionAttachmentUpload,
  type MockCommunication,
  type MockState,
} from "./seed-data";

function pad(value: number, width = 3): string {
  return String(value).padStart(width, "0");
}

function addHours(instant: string, hours: number): string {
  return new Date(new Date(instant).getTime() + hours * 60 * 60 * 1000).toISOString();
}

function managementRiskLevel(severity: FindingView["severity"]): RiskManagementProjectionView["findings"][number]["riskLevel"] {
  if (severity === "LEVEL_1_CRITICAL") return "HIGH";
  if (severity === "LEVEL_2_MAJOR") return "MEDIUM";
  if (severity === "LEVEL_3_MINOR") return "LOW";
  return "VERY_LOW";
}

function getPackage(state: Readonly<MockState>, packageId: string): InspectionPackage {
  const packageView = state.packages[packageId];
  if (!packageView) throw new BackendInvariantError(`Inspection package ${packageId} was not found.`);
  return {
    ...packageView,
    questions: packageView.questions.map((question) => ({
      ...question,
      currentResponse:
        Object.values(state.checklistResponses).find(
          (response) => response.questionId === question.id,
        ) ?? null,
    })),
  };
}

function packageForAudit(state: Readonly<MockState>, auditId: string): InspectionPackage {
  const packageView = Object.values(state.packages).find((candidate) => candidate.auditId === auditId);
  if (!packageView) throw new BackendInvariantError(`Audit ${auditId} has no inspection package.`);
  return packageView;
}

function findingForPrincipal(
  state: Readonly<MockState>,
  principal: BackendPrincipal,
  findingId: string,
): FindingView {
  const finding = state.findings[findingId];
  if (!finding) throw new BackendInvariantError(`Finding ${findingId} was not found.`);
  if (principal.role === "auditee" && finding.organizationId !== principal.organizationId) {
    throw new BackendAuthorizationInvariantError("Finding is unavailable to this Auditee organization.");
  }
  return finding;
}

function potentialFindingForPrincipal(
  state: Readonly<MockState>,
  principal: BackendPrincipal,
  potentialFindingId: string,
): PotentialFindingView {
  const potential = state.potentialFindings[potentialFindingId];
  if (!potential) throw new BackendInvariantError(`Potential Finding ${potentialFindingId} was not found.`);
  if (principal.role === "leadInspector") return potential;
  if (principal.role === "inspector") {
    const packageView = packageForAudit(state, potential.auditId);
    const question = packageView.questions.find((candidate) => candidate.id === potential.questionId);
    if (question?.assignedInspectorUserIds.includes(principal.subjectId)) return potential;
  }
  throw new BackendAuthorizationInvariantError("Potential Finding read authority is unavailable.");
}

function mutableFinding(state: MockState, findingId: string): FindingView {
  const finding = state.findings[findingId];
  if (!finding) throw new BackendInvariantError(`Finding ${findingId} was not found.`);
  return finding;
}

function stableMockCommandKey(prefix: string, fields: readonly string[]): string {
  return `${prefix}-${fields.join("-").replace(/[^A-Za-z0-9-]+/g, "-").slice(0, 72)}`;
}

function inspectionTeamForAudit(state: Readonly<MockState>, auditId: string): InspectionTeamAuditView {
  const assignment = state.assignments.find((candidate) => candidate.auditId === auditId);
  if (!assignment) throw new BackendInvariantError(`Audit ${auditId} was not found.`);
  const packageView = Object.values(state.packages).find((candidate) => candidate.auditId === auditId);
  const memberSubjectIds = packageView
    ? [...new Set(packageView.questions.flatMap((question) => question.assignedInspectorUserIds))]
    : assignment.currentOwnerRole === "inspector" && assignment.currentOwnerId
      ? [assignment.currentOwnerId]
      : [];
  const members = memberSubjectIds.map((subjectId) => {
    const member = state.profiles[subjectId];
    if (!member || member.role !== "inspector") {
      throw new BackendInvariantError(`Audit ${auditId} references unavailable Inspector ${subjectId}.`);
    }
    return { ...member };
  });
  const leadInspector = state.profiles["USR-LEAD-CANER"];
  if (!leadInspector || leadInspector.role !== "leadInspector") {
    throw new BackendInvariantError(`Audit ${auditId} has no typed Lead Inspector projection.`);
  }
  const findingIds = Object.values(state.findings)
    .filter((finding) => finding.auditId === auditId)
    .map((finding) => finding.id);
  const reportDocuments: DocumentMetadataView[] = Object.values(state.reportVersions)
    .filter((report) => report.auditId === auditId)
    .map((report) => ({
      id: report.reportVersionId,
      organizationId: report.organizationId,
      title: `Report ${report.reportId}`,
      kind: "REPORT",
      version: report.version,
      revision: report.revision,
      createdAt: report.issuedAt ?? "2026-01-15T08:00:00.000Z",
    }));
  const evidenceDocuments: DocumentMetadataView[] = state.evidenceVersions
    .filter((evidence) => findingIds.includes(evidence.findingId))
    .map((evidence) => ({
      id: evidence.id,
      organizationId: evidence.organizationId,
      title: evidence.fileName,
      kind: "EVIDENCE",
      version: evidence.version,
      revision: evidence.revision,
      createdAt: evidence.submittedAt,
    }));
  return {
    auditId: assignment.auditId,
    organizationId: assignment.organizationId,
    organizationName: assignment.organizationName,
    title: assignment.title,
    status: assignment.status,
    scheduledStartDate: assignment.scheduledStartDate,
    scheduledEndDate: assignment.dueDate,
    leadInspector: { ...leadInspector },
    members,
    assignments: packageView?.questions.map((question) => ({
      questionId: question.id,
      assignedMemberSubjectIds: [...question.assignedInspectorUserIds],
    })) ?? [],
    documents: [...reportDocuments, ...evidenceDocuments],
    history: [{
      eventId: `AUDIT-TEAM-${assignment.auditId}-001`,
      occurredAt: `${assignment.scheduledStartDate ?? assignment.dueDate ?? "2026-01-15"}T08:00:00.000Z`,
      actorSubjectId: leadInspector.subjectId,
      action: "AUDIT_TEAM_REGISTERED",
      detail: `Audit team projection opened for ${assignment.auditId}.`,
    }],
    revision: 1,
  };
}

function requireScreenAuthority(principal: BackendPrincipal, screenId: string) {
  const route = REACT_ROUTE_CONTRACTS.find((candidate) => candidate.id === screenId);
  if (!route) throw new BackendInvariantError(`Screen ${screenId} was not found in the route contract.`);
  if (route.requiredRole !== null && route.requiredRole !== principal.role) {
    throw new BackendAuthorizationInvariantError("Screen projection is unavailable to this role.");
  }
  return route;
}

function screenProjectionFor(
  state: Readonly<MockState>,
  principal: BackendPrincipal,
  screenId: string,
): import("../backend/backend").AdministrationScreenProjection {
  const route = requireScreenAuthority(principal, screenId);
  const seed = state.screenProjectionSeeds[screenId];
  if (!seed || seed.requiredRole !== route.requiredRole) {
    throw new BackendInvariantError(`Screen ${screenId} has no valid deterministic projection seed.`);
  }
  const directRecordId = route.path.match(/\/(?:AUD|FND|ORG|RPT|PR|CR|TPL)-[A-Z0-9-]+(?:-V\d+)?/)?.[0].slice(1) ?? null;
  const finding = directRecordId ? state.findings[directRecordId] : undefined;
  const report = directRecordId ? state.reportVersions[directRecordId] : undefined;
  const reportHistory = finding
    ? Object.values(state.reportVersions).filter((candidate) => candidate.findingIds.includes(finding.id))
    : report
      ? Object.values(state.reportVersions).filter((candidate) => candidate.reportId === report.reportId)
      : [];
  const evidenceHistory = finding
    ? state.evidenceVersions.filter((candidate) => candidate.findingId === finding.id)
    : [];
  const capHistory = finding
    ? state.capRevisions.filter((candidate) => candidate.findingId === finding.id)
    : [];
  const messageEmpty = route.id.includes("messages") && state.communications.length === 0;
  const returned = finding?.status === "EVIDENCE_MORE_INFORMATION_REQUESTED" || report?.status === "RETURNED";
  return {
    screenId: seed.screenId,
    organizationId: principal.role === "auditee" ? principal.organizationId : null,
    directRecordId,
    state: returned ? "returned" : messageEmpty ? "empty" : "ready",
    overdue: finding?.dueState === "OVERDUE",
    versionHistory: capHistory.length > 1 || evidenceHistory.length > 1 || reportHistory.length > 1,
    visibleActions: seed.visibleActions,
  };
}

function profileForPrincipal(state: Readonly<MockState>, principal: BackendPrincipal): ProfileView {
  const profile = state.profiles[principal.subjectId];
  if (!profile) throw new BackendInvariantError(`Profile ${principal.subjectId} was not found.`);
  return profile;
}

function auditeeCanViewCalendarAssignment(
  principal: BackendPrincipal,
  assignment: Readonly<MockState>["assignments"][number],
): boolean {
  return principal.role === "auditee"
    && assignment.organizationId === principal.organizationId
    && (assignment.inspectionNotice === "ROUTINE" || assignment.inspectionNotice === "ANNOUNCED")
    && assignment.caaReleasedToAuditee === true
    && assignment.noticeWithheld !== true;
}

function publicCommunication(message: MockCommunication): CommunicationView {
  const {
    id,
    organizationId,
    subject,
    body,
    audience,
    direction,
    revision,
    createdAt,
  } = message;
  return { id, organizationId, subject, body, audience, direction, revision, createdAt };
}

function auditeeCoordinationProjection(
  state: Readonly<MockState>,
  assignment: Readonly<MockState>["assignments"][number],
): AuditeeCoordinationView {
  if (!assignment.scheduledStartDate) {
    throw new BackendInvariantError(`Audit ${assignment.auditId} has no proposed scheduled start date.`);
  }
  const response = state.auditeeCoordinationResponses?.[assignment.auditId];
  return {
    auditId: assignment.auditId,
    organizationId: assignment.organizationId,
    organizationName: assignment.organizationName,
    title: assignment.title,
    inspectionCategory: "Routine / Announced",
    scheduledStartDate: assignment.scheduledStartDate,
    status: response?.status ?? "AWAITING_AUDITEE_CONFIRMATION",
    alternativeDate: response?.alternativeDate ?? null,
    nextAction: response?.status === "CONFIRMED"
      ? `Prepare for the CAA inspection scheduled on ${assignment.scheduledStartDate}.`
      : response?.status === "ALTERNATIVE_PROPOSED"
        ? `Wait for CAA acceptance of the proposed alternative date ${response.alternativeDate}.`
        : "Confirm the proposed inspection date or propose an alternative date to the CAA.",
    revision: response?.revision ?? 1,
  };
}

function auditeeCanViewReleasedReport(
  state: Readonly<MockState>,
  principal: BackendPrincipal,
  reportVersionId: string,
): boolean {
  const report = state.reportVersions[reportVersionId];
  const metadata = state.reportPublicMetadata?.[reportVersionId];
  if (principal.role !== "auditee" || !principal.organizationId || !report || !metadata) return false;
  return report.organizationId === principal.organizationId
    && report.status === "LOCKED"
    && Boolean(report.issuedAt)
    && report.findingIds.every((findingId) => state.findings[findingId]?.organizationId === principal.organizationId);
}

function auditeeReleasedReportProjection(
  state: Readonly<MockState>,
  principal: BackendPrincipal,
  reportVersionId: string,
): AuditeeReleasedReportView {
  requireRole(principal, ["auditee"], "Auditee authority is required for released report projections.");
  const report = state.reportVersions[reportVersionId];
  const metadata = state.reportPublicMetadata?.[reportVersionId];
  if (!report || !metadata || !report.issuedAt || !auditeeCanViewReleasedReport(state, principal, reportVersionId)) {
    throw new BackendAuthorizationInvariantError("Released report version is unavailable to this Auditee.");
  }
  return {
    reportVersionId: report.reportVersionId,
    reportId: report.reportId,
    kind: metadata.kind,
    organizationId: report.organizationId,
    auditId: report.auditId,
    findingIds: [...report.findingIds],
    version: report.version,
    status: "LOCKED",
    revision: report.revision,
    issuedAt: report.issuedAt,
    responseDueDate: metadata.responseDueDate,
    caaVisibleCommentState: metadata.caaVisibleComment ? "RECORDED" : "NO_COMMENT_RECORDED",
    caaVisibleComment: metadata.caaVisibleComment,
  };
}

function auditeeReportDocumentMetadata(
  state: Readonly<MockState>,
  principal: BackendPrincipal,
  report: Readonly<MockState>["reportVersions"][string],
): DocumentMetadataView {
  const released = auditeeReleasedReportProjection(state, principal, report.reportVersionId);
  return {
    id: released.reportVersionId,
    organizationId: released.organizationId,
    title: `Report ${released.reportId}`,
    kind: "REPORT",
    version: released.version,
    revision: released.revision,
    createdAt: released.issuedAt,
    publicReviewResult: "RELEASED",
    downloadFileName: `${released.reportId}.pdf`,
  };
}

function calendarAssignmentProjection(
  state: Readonly<MockState>,
  principal: BackendPrincipal,
  assignment: Readonly<MockState>["assignments"][number],
) {
  const auditeeCoordination = principal.role === "auditee"
    ? auditeeCoordinationProjection(state, assignment)
    : null;
  return {
    id: `CAL-${assignment.auditId}`,
    auditId: assignment.auditId,
    organizationId: assignment.organizationId,
    organizationName: assignment.organizationName,
    title: assignment.title,
    nextAction: auditeeCoordination?.nextAction ?? assignment.nextAction,
    scheduledDate: auditeeCoordination?.scheduledStartDate ?? assignment.dueDate ?? "2026-06-15",
    dueState: assignment.dueState,
  };
}

function inspectorCanViewCalendarAssignment(
  state: Readonly<MockState>,
  principal: BackendPrincipal,
  assignment: Readonly<MockState>["assignments"][number],
): boolean {
  if (principal.role !== "inspector") return false;
  const packageView = Object.values(state.packages).find((candidate) => candidate.auditId === assignment.auditId);
  return packageView?.questions.some((question) => question.assignedInspectorUserIds.includes(principal.subjectId)) ?? false;
}

function canViewCalendarAssignment(
  state: Readonly<MockState>,
  principal: BackendPrincipal,
  assignment: Readonly<MockState>["assignments"][number],
): boolean {
  if (principal.role === "inspector") return inspectorCanViewCalendarAssignment(state, principal, assignment);
  if (principal.role === "auditee") return auditeeCanViewCalendarAssignment(principal, assignment);
  return true;
}

function requireAuditeeOrganization(principal: BackendPrincipal, organizationId: string): void {
  requireRole(principal, ["auditee"], "Auditee authority is required.");
  if (!principal.organizationId || principal.organizationId !== organizationId) {
    throw new BackendAuthorizationInvariantError("Auditee organization scope does not match this record.");
  }
}

function requireSeparateReviewComments(commentToAuditee: string, internalCaaNote: string): void {
  requireNonEmpty(commentToAuditee, "Comment to Auditee");
  requireNonEmpty(internalCaaNote, "Internal CAA Note");
}

function capReadAudience(principal: BackendPrincipal, organizationId: string): "CAA" | "AUDITEE" {
  if (principal.role === "inspector" || principal.role === "leadInspector" || principal.role === "manager") {
    return "CAA";
  }
  if (principal.role === "auditee" && principal.organizationId === organizationId) {
    return "AUDITEE";
  }
  throw new BackendAuthorizationInvariantError("CAP revision read authority is unavailable.");
}

function capRevisionView(
  cap: Readonly<MockState>["capRevisions"][number],
  audience: "CAA" | "AUDITEE",
): CapRevisionView {
  const latestReview =
    cap.reviewDecision && cap.reviewedAt
      ? audience === "CAA"
        ? {
            decision: cap.reviewDecision,
            commentToAuditee: cap.commentToAuditee,
            internalCaaNote: cap.internalCaaNote,
            decidedAt: cap.reviewedAt,
          }
        : {
            decision: cap.reviewDecision,
            commentToAuditee: cap.commentToAuditee,
            decidedAt: cap.reviewedAt,
          }
      : null;
  return {
    audience,
    id: cap.id,
    capId: cap.capId,
    findingId: cap.findingId,
    organizationId: cap.organizationId,
    revision: cap.version,
    status: cap.status,
    rootCause: cap.rootCause,
    correctiveAction: cap.correctiveAction,
    preventiveAction: cap.preventiveAction,
    responsiblePerson: cap.responsiblePerson,
    targetCompletionDate: cap.targetCompletionDate,
    commentToCaa: cap.commentToCaa,
    submittedAt: cap.submittedAt,
    latestReview,
  } as CapRevisionView;
}

const SYNTHETIC_OWNER: GovernedRequiredOwnerView = {
  departmentId: "FLIGHT_OPERATIONS_INSPECTORATE",
  organizationalUnitId: "FLIGHT_OPERATIONS_INSPECTORATE",
  approvalRequired: true,
};

interface MockGovernedManagerAssignment {
  membershipId: string;
  membershipRootId: string;
  supersedesMembershipId: string | null;
  revision: number;
  departmentId: string;
  organizationalUnitId: string;
  status: "ACTIVE" | "REVOKED" | "EXPIRED";
  effectiveFrom: string;
  effectiveTo: string | null;
  departmentActive: boolean;
  organizationalUnitActive: boolean;
}

const MOCK_GOVERNED_MANAGER_ASSIGNMENTS: Record<string, MockGovernedManagerAssignment[]> = {
  "USR-MANAGER-NORA": [{
    membershipId: "MEM-TASK6-NORA",
    membershipRootId: "MEM-TASK6-NORA-ROOT",
    supersedesMembershipId: null,
    revision: 1,
    departmentId: "FLIGHT_OPERATIONS_INSPECTORATE",
    organizationalUnitId: "FLIGHT_OPERATIONS_INSPECTORATE",
    status: "ACTIVE",
    effectiveFrom: "2025-01-01",
    effectiveTo: null,
    departmentActive: true,
    organizationalUnitActive: true,
  }],
  "USR-MANAGER-AIR": [{
    membershipId: "MEM-TASK6-AIR",
    membershipRootId: "MEM-TASK6-AIR-ROOT",
    supersedesMembershipId: null,
    revision: 1,
    departmentId: "AIRWORTHINESS_INSPECTORATE",
    organizationalUnitId: "AIRWORTHINESS_INSPECTORATE",
    status: "ACTIVE",
    effectiveFrom: "2025-01-01",
    effectiveTo: null,
    departmentActive: true,
    organizationalUnitActive: true,
  }],
  "USR-TASK6-AIR-MANAGER": [{
    membershipId: "MEM-TASK6-AIR",
    membershipRootId: "MEM-TASK6-AIR-ROOT",
    supersedesMembershipId: null,
    revision: 1,
    departmentId: "AIRWORTHINESS_INSPECTORATE",
    organizationalUnitId: "AIRWORTHINESS_INSPECTORATE",
    status: "ACTIVE",
    effectiveFrom: "2025-01-01",
    effectiveTo: null,
    departmentActive: true,
    organizationalUnitActive: true,
  }],
  "USR-MANAGER-REVOKED": [{
    membershipId: "MEM-TASK6-REVOKED-PREDECESSOR",
    membershipRootId: "MEM-TASK6-REVOKED-ROOT",
    supersedesMembershipId: null,
    revision: 1,
    departmentId: "FLIGHT_OPERATIONS_INSPECTORATE",
    organizationalUnitId: "FLIGHT_OPERATIONS_INSPECTORATE",
    status: "ACTIVE",
    effectiveFrom: "2025-01-01",
    effectiveTo: null,
    departmentActive: true,
    organizationalUnitActive: true,
  }, {
    membershipId: "MEM-TASK6-REVOKED-LATEST",
    membershipRootId: "MEM-TASK6-REVOKED-ROOT",
    supersedesMembershipId: "MEM-TASK6-REVOKED-PREDECESSOR",
    revision: 2,
    departmentId: "FLIGHT_OPERATIONS_INSPECTORATE",
    organizationalUnitId: "FLIGHT_OPERATIONS_INSPECTORATE",
    status: "REVOKED",
    effectiveFrom: "2026-01-01",
    effectiveTo: null,
    departmentActive: true,
    organizationalUnitActive: true,
  }],
};

function mockCandidateChangeReason(bundle: GovernedCandidateBundleInput): string {
  if (bundle.candidateBundleId === SYNTHETIC_LEGACY_CHECKLIST_CANDIDATE_BUNDLE.candidateBundleId) {
    return "Imported candidate-only legacy checklist Draft; SOURCE_MAPPING_REQUIRED must be repaired through the current controlled source chain.";
  }
  if (bundle.candidateBundleId === SYNTHETIC_HYBRID_RECONCILED_BUNDLE.candidateBundleId) {
    return "Imported current-source impact-review reconciliation Draft; Department Manager technical approval and publication remain separate.";
  }
  return "Imported deterministic synthetic governed candidate.";
}

function mockGovernedCandidateForBundle(bundle: GovernedCandidateBundleInput): GovernedCandidateView {
  const source = bundle.generationRequest.sourceSnapshots[0]!;
  return {
  candidateId: bundle.candidateBundleId,
  candidateRootId: bundle.candidateBundleId,
  supersedesCandidateId: null,
  generationRunId: bundle.generationRunId,
  templateId: `TPL-${bundle.inspectionChecklist.checklistId}`,
  version: 1,
  revision: 1,
  status: "GENERATED_DRAFT",
  contentDigest: bundle.outputDigest,
  schemaVersion: bundle.schemaVersion,
  changeReason: mockCandidateChangeReason(bundle),
  sourceSnapshots: [{
    sourceId: source.sourceSnapshotId,
    sourceIdentity: "SYNTHETIC-OPS-AOC",
    versionIdentity: source.sourceSnapshotId === "SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2" ? "2" : "1",
    sourceHash: source.sourceHash,
    clauseId: source.clauseIds[0]!,
    locator: source.clauseLocators[0]!,
  }],
  scopeFactIds: structuredClone(bundle.generationRequest.serviceProviderScopeFactIds),
  crosswalkPartitionIds: [bundle.generationRequest.secondaryCrosswalkPartition.partitionId],
  mappings: structuredClone(bundle.complianceMappings),
  questions: structuredClone(bundle.inspectionChecklist.questions),
  requiredOwners: [SYNTHETIC_OWNER],
  lineage: {
    lineageType: "GENERATION_RUN",
    entryPath: "GENERATION_RUN",
    lineageKind: "GENERATION_RUN",
    candidateRootId: bundle.candidateBundleId,
    supersedesCandidateId: null,
    supersedesContentDigest: null,
    generationRunId: bundle.generationRunId,
    existingCandidateId: null,
    legacyAuthorityState: null,
    bindingSetId: null,
  },
  };
}

function mockGovernedRunForBundle(
  bundle: GovernedCandidateBundleInput,
  candidate: GovernedCandidateView,
): GovernedGenerationRunView {
  return {
    generationRunId: bundle.generationRunId,
    status: "GENERATED",
    inputDigest: bundle.inputDigest,
    outputDigest: bundle.outputDigest,
    inputSchemaVersion: bundle.generationRequest.schemaVersion,
    generationPolicyVersion: bundle.generationRequest.generationPolicyVersion,
    providerCatalogVersion: bundle.generationRequest.providerCatalogVersion,
    providerId: bundle.generationRequest.providerId,
    providerAdapterVersion: bundle.generationRequest.providerVersion,
    inspectionType: bundle.generationRequest.inspectionType,
    targetId: bundle.generationRequest.target.targetId,
    requestId: bundle.generationRequest.requestId,
    failure: null,
    candidate,
  };
}

const SYNTHETIC_GOVERNED_CANDIDATE = mockGovernedCandidateForBundle(SYNTHETIC_GOVERNED_BUNDLE);

const SYNTHETIC_GOVERNED_RUN = mockGovernedRunForBundle(
  SYNTHETIC_GOVERNED_BUNDLE,
  SYNTHETIC_GOVERNED_CANDIDATE,
);

const SYNTHETIC_FAILED_GOVERNED_RUN: GovernedGenerationRunView = {
  generationRunId: SYNTHETIC_FAILED_RUN_ID,
  status: "FAILED",
  inputDigest: SYNTHETIC_INPUT_DIGEST,
  outputDigest: null,
  inputSchemaVersion: "1.0.0",
  generationPolicyVersion: "regulatory-checklist-v1",
  providerCatalogVersion: "1.0.0",
  providerId: "deterministic-regulatory-fixture",
  providerAdapterVersion: "1.0.0",
  inspectionType: "RAMP_INSPECTION",
  targetId: "TARGET-SYNTHETIC-AOC",
  requestId: SYNTHETIC_FAILED_REQUEST_ID,
  failure: {
    code: "VALIDATION_FAILED",
    reason: "Exact synthetic failed-run inspection fixture",
    requestId: SYNTHETIC_FAILED_REQUEST_ID,
    operationId: "ADMIN-SYNTHETIC-GOVERNED-FAILED",
    idempotencyKey: "ADMIN-SYNTHETIC-GOVERNED-FAILED",
  },
  candidate: null,
};

function governedValidationIssue(fieldPath: string, code: string, message: string): GovernedValidationIssue {
  return {
    fieldPath, code, message, sourceIdentity: "SYNTHETIC-OPS-AOC",
    sourceHash: SYNTHETIC_SOURCE_HASH, clauseId: "CLAUSE-SYNTHETIC-OPS-AOC-1",
    locator: "Synthetic OPS/AOC 1",
  };
}

// The browser-local profile follows the same fail-closed question boundary as
// the Go candidate service. It intentionally validates only the bounded
// generated transport shape; it does not treat mock/seed wording as a
// regulatory authority source.
export function governedQuestionGovernanceIssuesForTest(question: GovernedQuestionView): GovernedValidationIssue[] {
  const issue = (fieldPath: string, code: string, message: string) => governedValidationIssue(fieldPath, code, message);
  const issues: GovernedValidationIssue[] = [];
  const scope = question.scopeRecommendation;
  const trace = question.regulatoryTrace;
  const guarded = scope.guardrails;
  if (![
    "REGULATORY_TRACE",
    "EXISTING_CHECKLIST_CANDIDATE",
    "HYBRID_RECONCILED",
  ].includes(question.origin)) {
    issues.push(issue(`questions[${question.questionId}].origin`, "QUESTION_ORIGIN_REQUIRED", "every generated or published question must name one exact origin"));
  }
  if (![
    "MANDATORY_CORE",
    "FOCUSED_FULL",
    "ROTATIONAL_SAMPLE",
    "DEFER_ELIGIBLE",
  ].includes(scope.classification)) {
    issues.push(issue(`questions[${question.questionId}].scopeRecommendation.classification`, "SCOPE_CLASSIFICATION_REQUIRED", "scope recommendation requires a visible classification"));
  }
  if (!scope.inputSignals.length || scope.inputSignals.some((signal) => !signal.trim()) || !scope.operationalHistoryBasis.trim()) {
    issues.push(issue(`questions[${question.questionId}].scopeRecommendation`, "SCOPE_RECOMMENDATION_REQUIRED", "scope recommendation requires input signals and an operational-history basis"));
  }
  if (!scope.rationale.trim()) {
    issues.push(issue(`questions[${question.questionId}].scopeRecommendation.rationale`, "SCOPE_RATIONALE_REQUIRED", "scope recommendation requires a visible inclusion or deferral rationale"));
  }
  if (scope.approvalReviewState !== "TECHNICAL_REVIEW_REQUIRED") {
    issues.push(issue(`questions[${question.questionId}].scopeRecommendation.approvalReviewState`, "SCOPE_REVIEW_STATE_REQUIRED", "immutable Draft scope state must require technical review; approval is projected only from an attributed decision"));
  }
  if (question.mandatoryCore !== guarded.mandatoryControl || question.safetyCritical !== guarded.safetyCritical) {
    issues.push(issue(`questions[${question.questionId}].scopeRecommendation.guardrails`, "SCOPE_GUARDRAIL_MISMATCH", "scope guardrails must preserve the question's mandatory and safety-critical controls"));
  }
  if (scope.automaticDeferral && (guarded.mandatoryControl || guarded.safetyCritical || guarded.unknownHistory || guarded.sourceChanged || guarded.overdueControl || !guarded.automaticDeferralPermitted)) {
    issues.push(issue(`questions[${question.questionId}].scopeRecommendation.automaticDeferral`, "AUTOMATIC_DEFERRAL_DENIED", "mandatory, safety-critical, changed, overdue, or unknown-history controls cannot be automatically deferred"));
  }
  if (question.origin === "HYBRID_RECONCILED") {
    const reconciliation = question.reconciliation;
    if (!reconciliation || !reconciliation.legacyQuestionId.trim() || !reconciliation.legacyWording.trim() ||
      !reconciliation.legacyOperationalIntent.trim() || !reconciliation.legacyResultHistory.trim() ||
      !reconciliation.legacyExpectedEvidence.length || !reconciliation.legacyApplicability.trim() ||
      !reconciliation.legacyScopeClassification.trim() || reconciliation.currentWording !== question.prompt ||
      JSON.stringify(reconciliation.currentExpectedEvidence) !== JSON.stringify(question.expectedEvidence) ||
      reconciliation.currentApplicability !== trace.applicability ||
      reconciliation.currentScopeClassification !== scope.classification) {
      issues.push(issue(`questions[${question.questionId}].reconciliation`, "HYBRID_RECONCILIATION_REQUIRED", "hybrid reconciliation requires a complete candidate-only legacy/current comparison"));
    }
  } else if (question.reconciliation !== null) {
    issues.push(issue(`questions[${question.questionId}].reconciliation`, "HYBRID_RECONCILIATION_REQUIRED", "only HYBRID_RECONCILED questions may carry a legacy/current comparison"));
  }
  if (question.origin === "EXISTING_CHECKLIST_CANDIDATE" && trace.state !== "SOURCE_MAPPING_REQUIRED") {
    issues.push(issue(`questions[${question.questionId}].origin`, "QUESTION_ORIGIN_TRACE_MISMATCH", "EXISTING_CHECKLIST_CANDIDATE remains a non-authoritative source-gap Draft until a HYBRID_RECONCILED current-source trace is created"));
  }
  if (question.origin !== "EXISTING_CHECKLIST_CANDIDATE" && trace.state === "SOURCE_MAPPING_REQUIRED") {
    issues.push(issue(`questions[${question.questionId}].origin`, "QUESTION_ORIGIN_TRACE_MISMATCH", "SOURCE_MAPPING_REQUIRED is reserved for an explicit EXISTING_CHECKLIST_CANDIDATE repair Draft"));
  }
  if (trace.state === "SOURCE_MAPPING_REQUIRED") {
    const partialSourceGapTrace = question.citations.length !== 0 || [
      trace.sourceIdentity,
      trace.sourceTitle,
      trace.immutableVersion,
      trace.sha256,
      trace.locator,
      trace.page,
      trace.section,
      trace.clause,
      trace.sourceType,
      trace.applicability,
      trace.nationalReference,
      trace.controlledCaaProcedureMapping,
      trace.verificationObjective,
    ].some((value) => value?.trim()) || (trace.expectedEvidence?.length ?? 0) !== 0 ||
      ![undefined, "SOURCE_MAPPING_REQUIRED"].includes(trace.currentnessState) ||
      ![undefined, "NOT_AVAILABLE"].includes(trace.technicalReviewState);
    if (partialSourceGapTrace) {
      issues.push(issue(`questions[${question.questionId}].regulatoryTrace`, "REGULATORY_TRACE_REQUIRED", "SOURCE_MAPPING_REQUIRED must remain a literal repair state without a partial citation or trace"));
    }
    issues.push(issue(`questions[${question.questionId}].regulatoryTrace`, "SOURCE_MAPPING_REQUIRED", "SOURCE_MAPPING_REQUIRED must be repaired before validation, publication, deferral, or executable Audit use"));
    return issues;
  }
  if (trace.state !== "RESOLVED") {
    issues.push(issue(`questions[${question.questionId}].regulatoryTrace`, "REGULATORY_TRACE_REQUIRED", "every question requires a resolved regulatory trace or the literal SOURCE_MAPPING_REQUIRED state"));
    return issues;
  }
  if (!trace.sourceIdentity?.trim() || !trace.sourceTitle?.trim() || !trace.immutableVersion?.trim() ||
    !trace.sha256?.startsWith("sha256:") || !trace.locator?.trim() || !trace.page?.trim() ||
    !trace.section?.trim() || !trace.clause?.trim() || !trace.sourceType?.trim() ||
    !trace.applicability?.trim() || !trace.nationalReference?.trim() ||
    !trace.controlledCaaProcedureMapping?.trim() || !trace.verificationObjective?.trim() ||
    !trace.expectedEvidence?.length || trace.expectedEvidence.some((evidence) => !evidence.trim())) {
    issues.push(issue(`questions[${question.questionId}].regulatoryTrace`, "REGULATORY_TRACE_REQUIRED", "regulatory trace requires source identity, immutable version/hash, locator, applicability, procedure mapping, objective, and expected Evidence"));
    return issues;
  }
  const citation = question.citations[0];
  if (question.citations.length !== 1 || !citation || citation.sourceHash !== trace.sha256 || citation.clauseId !== trace.clause || citation.locator !== trace.locator ||
    JSON.stringify(trace.expectedEvidence) !== JSON.stringify(question.expectedEvidence)) {
    issues.push(issue(`questions[${question.questionId}].regulatoryTrace`, "REGULATORY_TRACE_MISMATCH", "regulatory trace must exactly match the persisted citation and expected Evidence"));
  }
  if (trace.currentnessState === "STALE") {
    issues.push(issue(`questions[${question.questionId}].regulatoryTrace.currentnessState`, "STALE_SOURCE_TRACE", "a stale source version or hash blocks publication until a new impact-review Draft is approved"));
  } else if (trace.currentnessState !== "CURRENT") {
    issues.push(issue(`questions[${question.questionId}].regulatoryTrace.currentnessState`, "SOURCE_CURRENTNESS_REQUIRED", "regulatory trace requires a currentness result"));
  }
  if (trace.technicalReviewState !== "TECHNICAL_REVIEW_REQUIRED") {
    issues.push(issue(`questions[${question.questionId}].regulatoryTrace.technicalReviewState`, "TRACE_TECHNICAL_REVIEW_REQUIRED", "immutable Draft trace state must require technical review; approval is projected only from an attributed decision"));
  }
  return issues;
}

function validateMockGovernedEdit(
  expectedCandidate: GovernedCandidateView,
  mappings: GovernedMappingView[],
  questions: GovernedQuestionView[],
  owners: GovernedRequiredOwnerView[],
): void {
  const same = (left: unknown, right: unknown) =>
    governedCanonicalJSON(left) === governedCanonicalJSON(right);
  const expectedMapping = expectedCandidate.mappings[0];
  const mapping = mappings[0];
  if (mappings.length !== 1) {
    const index = Math.max(0, mappings.length - 1);
    throw new GovernedValidationError([governedValidationIssue(`mappings[${index}].mappingId`, "MAPPING_IDENTITY_MISMATCH", "the edit must preserve the complete mapping identity set")]);
  }
  if (!mapping || !mapping.mappingId.trim()) {
    throw new GovernedValidationError([governedValidationIssue("mappings[0].mappingId", "DUPLICATE_OR_BLANK_MAPPING_ID", "mapping IDs must be nonblank and unique")]);
  }
  if (mapping.mappingId !== expectedMapping.mappingId) {
    throw new GovernedValidationError([governedValidationIssue("mappings[0].mappingId", "MAPPING_IDENTITY_MISMATCH", "mapping identity is immutable")]);
  }
  if (mapping.requirement !== expectedMapping.requirement) {
    throw new GovernedValidationError([governedValidationIssue("mappings[0].requirement", "UNSUPPORTED_CLAIM", "requirement is outside the controlled synthetic registry")]);
  }
  if (mapping.relationship !== expectedMapping.relationship) {
    throw new GovernedValidationError([governedValidationIssue("mappings[0].relationship", "RELATIONSHIP_MISMATCH", "relationship must preserve the exact supported mapping")]);
  }
  if (mapping.applicability !== expectedMapping.applicability) {
    throw new GovernedValidationError([governedValidationIssue("mappings[0].applicability", "APPLICABILITY_MISMATCH", "applicability must preserve the exact supported mapping")]);
  }
  if (mapping.sourceGap !== null) {
    throw new GovernedValidationError([governedValidationIssue("mappings[0].sourceGap", "SOURCE_GAP_MISMATCH", "source gaps may not be inferred or fabricated")]);
  }
  const expectedCitation = expectedMapping.citations[0]!;
  const citation = mapping.citations[0];
  if (!citation || mapping.citations.length !== 1) {
    throw new GovernedValidationError([governedValidationIssue("mappings[0].citations", "CITATION_MISMATCH", "one exact persisted citation is required")]);
  }
  if (citation.sourceSnapshotId !== expectedCitation.sourceSnapshotId) throw new GovernedValidationError([governedValidationIssue("mappings[0].citations[0].sourceSnapshotId", "SOURCE_IDENTITY_MISMATCH", "citation source identity is immutable")]);
  if (citation.sourceHash !== expectedCitation.sourceHash) throw new GovernedValidationError([governedValidationIssue("mappings[0].citations[0].sourceHash", "SOURCE_HASH_MISMATCH", "citation source hash is immutable")]);
  if (citation.clauseId !== expectedCitation.clauseId) throw new GovernedValidationError([governedValidationIssue("mappings[0].citations[0].clauseId", "CLAUSE_IDENTITY_MISMATCH", "citation clause identity is immutable")]);
  if (citation.locator !== expectedCitation.locator) throw new GovernedValidationError([governedValidationIssue("mappings[0].citations[0].locator", "LOCATOR_MISMATCH", "citation locator is immutable")]);
  if (mapping.rationale !== expectedMapping.rationale && mapping.rationale !== SYNTHETIC_EDITED_RATIONALE) {
    throw new GovernedValidationError([governedValidationIssue("mappings[0].rationale", "UNSUPPORTED_CLAIM", "rationale is outside the controlled synthetic registry")]);
  }
  const expectedQuestion = expectedCandidate.questions[0];
  const question = questions[0];
  if (questions.length !== 1) {
    const index = Math.max(0, questions.length - 1);
    throw new GovernedValidationError([governedValidationIssue(`questions[${index}].questionId`, "QUESTION_IDENTITY_MISMATCH", "the edit must preserve the complete question identity set")]);
  }
  if (!question || !question.questionId.trim()) {
    throw new GovernedValidationError([governedValidationIssue("questions[0].questionId", "DUPLICATE_OR_BLANK_QUESTION_ID", "question IDs must be nonblank and unique")]);
  }
  if (question.questionId !== expectedQuestion.questionId) {
    throw new GovernedValidationError([governedValidationIssue("questions[0].questionId", "QUESTION_IDENTITY_MISMATCH", "question identity is immutable")]);
  }
  if (!same(question.mappingIds, [expectedMapping.mappingId])) {
    throw new GovernedValidationError([governedValidationIssue("questions[0].mappingIds[0]", "MAPPING_REFERENCE_MISMATCH", "question mapping references must resolve to the exact preserved mapping")]);
  }
  if (question.prompt !== expectedQuestion.prompt) {
    throw new GovernedValidationError([governedValidationIssue("questions[0].prompt", "UNSUPPORTED_CLAIM", "question text is outside the controlled synthetic registry")]);
  }
  if (question.verificationMethod !== expectedQuestion.verificationMethod) {
    throw new GovernedValidationError([governedValidationIssue("questions[0].verificationMethod", "UNSUPPORTED_CLAIM", "verification method is outside the controlled synthetic registry")]);
  }
  if (question.expectedEvidence.length === 0 || question.expectedEvidence.some((value) => !value.trim())) {
    throw new GovernedValidationError([governedValidationIssue("questions[0].expectedEvidence[0]", "BLANK_EVIDENCE", "expected Evidence entries must be nonblank")]);
  }
  if (!same(question.expectedEvidence, expectedQuestion.expectedEvidence)) {
    throw new GovernedValidationError([governedValidationIssue("questions[0].expectedEvidence", "UNSUPPORTED_CLAIM", "expected Evidence is outside the controlled synthetic registry")]);
  }
  if (!same(question.allowedAnswers, expectedQuestion.allowedAnswers)) throw new GovernedValidationError([governedValidationIssue("questions[0].allowedAnswers", "INVALID_ALLOWED_ANSWERS", "allowed answers must preserve the exact governed set")]);
  if (question.mandatoryCore !== expectedQuestion.mandatoryCore) throw new GovernedValidationError([governedValidationIssue("questions[0].mandatoryCore", "MANDATORY_FLAG_MISMATCH", "mandatory-core classification is immutable")]);
  if (question.safetyCritical !== expectedQuestion.safetyCritical) throw new GovernedValidationError([governedValidationIssue("questions[0].safetyCritical", "SAFETY_FLAG_MISMATCH", "safety-critical classification is immutable")]);
  const questionCitation = question.citations[0];
  if (!questionCitation || question.citations.length !== 1) {
    throw new GovernedValidationError([governedValidationIssue("questions[0].citations", "CITATION_MISMATCH", "one exact persisted citation is required")]);
  }
  if (questionCitation.sourceSnapshotId !== expectedCitation.sourceSnapshotId) throw new GovernedValidationError([governedValidationIssue("questions[0].citations[0].sourceSnapshotId", "SOURCE_IDENTITY_MISMATCH", "citation source identity is immutable")]);
  if (questionCitation.sourceHash !== expectedCitation.sourceHash) throw new GovernedValidationError([governedValidationIssue("questions[0].citations[0].sourceHash", "SOURCE_HASH_MISMATCH", "citation source hash is immutable")]);
  if (questionCitation.clauseId !== expectedCitation.clauseId) throw new GovernedValidationError([governedValidationIssue("questions[0].citations[0].clauseId", "CLAUSE_IDENTITY_MISMATCH", "citation clause identity is immutable")]);
  if (questionCitation.locator !== expectedCitation.locator) throw new GovernedValidationError([governedValidationIssue("questions[0].citations[0].locator", "LOCATOR_MISMATCH", "citation locator is immutable")]);
  const owner = owners[0];
  if (!owner || owners.length !== 1) {
    throw new GovernedValidationError([governedValidationIssue("requiredOwners", "OWNER_SET_MISMATCH", "the complete required-owner set is immutable")]);
  }
  if (owner.departmentId !== SYNTHETIC_OWNER.departmentId) throw new GovernedValidationError([governedValidationIssue("requiredOwners[0].departmentId", "UNKNOWN_OWNER", "required owner department is unknown or changed")]);
  if (owner.organizationalUnitId !== SYNTHETIC_OWNER.organizationalUnitId) throw new GovernedValidationError([governedValidationIssue("requiredOwners[0].organizationalUnitId", "UNKNOWN_OWNER", "required owner organizational unit is unknown or changed")]);
  if (owner.approvalRequired !== SYNTHETIC_OWNER.approvalRequired) throw new GovernedValidationError([governedValidationIssue("requiredOwners[0].approvalRequired", "OWNER_APPROVAL_MISMATCH", "required owner approval classification is immutable")]);
}

interface MockGovernedCommand {
  operationId: string;
  idempotencyKey: string;
  semantic: string;
  candidateId: string;
	generationRunId?: string;
  actorSubjectId?: string;
  actorDepartmentMembershipId?: string;
  candidateRevision?: number;
  candidateContentDigest?: string;
  reason?: string;
  occurredAt?: string;
  committedCandidate?: GovernedCandidateView;
}

interface MockGovernedPublishedVersion {
  publication: GovernedPublicationView;
  mappings: GovernedMappingView[];
  questions: GovernedQuestionView[];
}

// The synthetic source catalog deliberately contains both the supplied V1
// baseline and the supplied-but-inert V2 source. Currentness activation
// changes eligibility; it never rewrites source identity or makes a V2
// candidate appear under V1 in the Regulatory Library.
type MockGovernedSourceDefinition = Omit<
  GovernedSourceSnapshotView,
  "applicabilityFacts" | "generationRunIds" | "candidateIds"
>;

const SYNTHETIC_IMPACT_SOURCE = SYNTHETIC_HYBRID_RECONCILED_BUNDLE.generationRequest.sourceSnapshots[0]!;

const MOCK_GOVERNED_SOURCE_DEFINITIONS: readonly MockGovernedSourceDefinition[] = [
  {
    sourceId: "SOURCE-SYNTHETIC-OPS-AOC",
    sourceIdentity: "SYNTHETIC-OPS-AOC",
    versionIdentity: "1",
    title: "Synthetic test-profile source",
    sourceHash: SYNTHETIC_SOURCE_HASH,
    locator: "Synthetic OPS/AOC source",
    clauseId: "CLAUSE-SYNTHETIC-OPS-AOC-1",
    clauseLocator: "Synthetic OPS/AOC 1",
    partitions: [{
      evaluationId: "EVAL-SYNTHETIC-OPS-AOC",
      partitionId: "PARTITION-SYNTHETIC-INPUT",
      role: "GENERATION_INPUT",
      crosswalkRowId: "CCROW-SYNTHETIC-OPS-AOC-1",
      stableRowIdentity: "CC:SYNTHETIC:OPS:AOC:1",
    }],
    unresolvedGaps: [],
  },
  {
    sourceId: "SOURCE-SYNTHETIC-OPS-AOC",
    sourceIdentity: "SYNTHETIC-OPS-AOC",
    versionIdentity: "1",
    title: "Synthetic test-profile source",
    sourceHash: SYNTHETIC_SOURCE_HASH,
    locator: "Synthetic OPS/AOC source",
    clauseId: "CLAUSE-SYNTHETIC-OPS-AOC-HOLDOUT-1",
    clauseLocator: "Synthetic OPS/AOC holdout 1",
    partitions: [{
      evaluationId: "EVAL-SYNTHETIC-OPS-AOC",
      partitionId: "PARTITION-SYNTHETIC-HOLDOUT",
      role: "BLIND_HOLDOUT",
      crosswalkRowId: "CCROW-SYNTHETIC-OPS-AOC-HOLDOUT-1",
      stableRowIdentity: "CC:SYNTHETIC:OPS:AOC:HOLDOUT:1",
    }],
    unresolvedGaps: [],
  },
  {
    sourceId: SYNTHETIC_IMPACT_SOURCE.sourceSnapshotId,
    sourceIdentity: "SYNTHETIC-OPS-AOC",
    versionIdentity: "2",
    title: "Synthetic test-profile impact source",
    sourceHash: SYNTHETIC_IMPACT_SOURCE.sourceHash,
    locator: "Synthetic OPS/AOC impact source",
    clauseId: "CLAUSE-SYNTHETIC-OPS-AOC-IMPACT-2",
    clauseLocator: "Synthetic OPS/AOC impact 2",
    partitions: [{
      evaluationId: "EVAL-SYNTHETIC-OPS-AOC-IMPACT-V2",
      partitionId: "PARTITION-SYNTHETIC-IMPACT-INPUT",
      role: "GENERATION_INPUT",
      crosswalkRowId: "CCROW-SYNTHETIC-OPS-AOC-IMPACT-2",
      stableRowIdentity: "CC:SYNTHETIC:OPS:AOC:IMPACT:2",
    }],
    unresolvedGaps: [],
  },
  {
    sourceId: SYNTHETIC_IMPACT_SOURCE.sourceSnapshotId,
    sourceIdentity: "SYNTHETIC-OPS-AOC",
    versionIdentity: "2",
    title: "Synthetic test-profile impact source",
    sourceHash: SYNTHETIC_IMPACT_SOURCE.sourceHash,
    locator: "Synthetic OPS/AOC impact source",
    clauseId: "CLAUSE-SYNTHETIC-OPS-AOC-IMPACT-HOLDOUT-2",
    clauseLocator: "Synthetic OPS/AOC impact holdout 2",
    partitions: [{
      evaluationId: "EVAL-SYNTHETIC-OPS-AOC-IMPACT-V2",
      partitionId: "PARTITION-SYNTHETIC-IMPACT-HOLDOUT",
      role: "BLIND_HOLDOUT",
      crosswalkRowId: "CCROW-SYNTHETIC-OPS-AOC-IMPACT-HOLDOUT-2",
      stableRowIdentity: "CC:SYNTHETIC:OPS:AOC:IMPACT:HOLDOUT:2",
    }],
    unresolvedGaps: [],
  },
];

// The mock keeps explicit source-currentness activations separately from
// candidate and published snapshots. A supplied V2 source is inert until this
// append-only ledger records the exact predecessor/current pair; candidate
// import only binds to that record and can never create it implicitly.
interface MockSourceCurrentnessEvent {
  eventId: string;
  impactReviewDraftId: string | null;
  sourceIdentity: string;
  previousSourceSnapshotId: string | null;
  previousSourceHash: string | null;
  currentSourceSnapshotId: string;
  currentSourceHash: string;
  sequence: number;
  operationId: string;
  idempotencyKey: string;
  semantic: string;
  reason: string;
  activatedAt: string;
}

interface MockSourceCurrentnessCommand {
  operationId: string;
  idempotencyKey: string;
  semantic: string;
  view: GovernedSourceCurrentnessActivationView;
}

interface MockGovernedState {
  candidate: GovernedCandidateView;
  candidates: Map<string, GovernedCandidateView>;
  runs: Map<string, GovernedGenerationRunView>;
  commands: Map<string, MockGovernedCommand>;
  decisions: Map<string, GovernedReviewDecisionView[]>;
  blockers: Map<string, GovernedValidationIssue[]>;
  publicationDecisions: Map<string, GovernedPublicationView>;
  publishedVersions: Map<string, MockGovernedPublishedVersion>;
  sourceCurrentnessEvents: Map<string, MockSourceCurrentnessEvent>;
  sourceCurrentnessCommands: Map<string, MockSourceCurrentnessCommand>;
  sourceImpactCandidateBindings: Map<string, string>;
}

const governedStateByStore = new WeakMap<MemoryMockStore, MockGovernedState>();

export class MockBackendEngine implements DemoBackend {
  readonly mode = "mock" as const;
  private readonly governedState: MockGovernedState;
  private readonly governedIntakeBatches = new Map<string, ChecklistImportBatchReceiptView>();
  private readonly canonicalSelections = new Map<string, { digest: string; ids: string[] }>();
  private readonly canonicalReviewOperations = new Map<string, string>();

  constructor(
    private readonly store: MemoryMockStore,
    private readonly principal: BackendPrincipal,
    governedRequiredOwners?: GovernedRequiredOwnerView[],
    governedBlockingIssues?: GovernedValidationIssue[],
  ) {
    const existing = governedStateByStore.get(store);
    if (existing) {
      this.governedState = existing;
    } else {
      const candidate = {
        ...structuredClone(SYNTHETIC_GOVERNED_CANDIDATE),
        requiredOwners: structuredClone(governedRequiredOwners ?? [SYNTHETIC_OWNER]),
      };
      this.governedState = {
        candidate,
        candidates: new Map([[candidate.candidateId, candidate]]),
        runs: new Map([
          [SYNTHETIC_GOVERNED_RUN.generationRunId, { ...structuredClone(SYNTHETIC_GOVERNED_RUN), candidate }],
          [SYNTHETIC_FAILED_GOVERNED_RUN.generationRunId, structuredClone(SYNTHETIC_FAILED_GOVERNED_RUN)],
        ]),
        commands: new Map(),
        decisions: new Map(),
        blockers: new Map([[
          candidate.candidateId,
          structuredClone(governedBlockingIssues ?? []),
        ]]),
        publicationDecisions: new Map(),
        publishedVersions: new Map(),
        sourceCurrentnessEvents: new Map([["SRC-CURRENTNESS-TESTPROFILE-BASELINE-V1", {
          eventId: "SRC-CURRENTNESS-TESTPROFILE-BASELINE-V1",
          impactReviewDraftId: null,
          sourceIdentity: "SYNTHETIC-OPS-AOC",
          previousSourceSnapshotId: null,
          previousSourceHash: null,
          currentSourceSnapshotId: "SOURCE-SYNTHETIC-OPS-AOC",
          currentSourceHash: SYNTHETIC_SOURCE_HASH,
          sequence: 1,
          operationId: "TESTPROFILE-SOURCE-CURRENTNESS-BASELINE-V1",
          idempotencyKey: "TESTPROFILE-SOURCE-CURRENTNESS-BASELINE-V1",
          semantic: "test-profile-baseline",
          reason: "Synthetic internal test-profile baseline currentness declaration.",
          activatedAt: store.clock(),
        }]]),
        sourceCurrentnessCommands: new Map(),
        sourceImpactCandidateBindings: new Map(),
      };
      governedStateByStore.set(store, this.governedState);
    }
  }

  private get governedCandidate() { return this.governedState.candidate; }
  private set governedCandidate(candidate: GovernedCandidateView) { this.governedState.candidate = candidate; }
  private get governedCandidates() { return this.governedState.candidates; }
  private get governedRuns() { return this.governedState.runs; }
  private get governedCommands() { return this.governedState.commands; }

  private mockGovernedSourceSnapshots(): GovernedSourceSnapshotView[] {
    const candidates = [...this.governedCandidates.values()];
    const runs = [...this.governedRuns.values()];
    return MOCK_GOVERNED_SOURCE_DEFINITIONS.map((definition) => {
      const isExactSourceClause = (source: { sourceId: string; sourceHash: string; clauseId: string }) =>
        source.sourceId === definition.sourceId &&
        source.sourceHash === definition.sourceHash &&
        source.clauseId === definition.clauseId;
      const applicabilityFacts = candidates.flatMap((candidate) =>
        candidate.mappings
          .filter((mapping) => mapping.citations.some((citation) => isExactSourceClause({
            sourceId: citation.sourceSnapshotId,
            sourceHash: citation.sourceHash,
            clauseId: citation.clauseId,
          })))
          .map((mapping) => ({
            candidateId: candidate.candidateId,
            mappingId: mapping.mappingId,
            relationship: mapping.relationship,
            applicability: mapping.applicability,
            sourceGap: mapping.sourceGap,
          })),
      ).sort((left, right) =>
        left.candidateId.localeCompare(right.candidateId) || left.mappingId.localeCompare(right.mappingId),
      );
      const generationRunIds = runs
        .filter((run) => run.candidate?.sourceSnapshots.some(isExactSourceClause))
        .map((run) => run.generationRunId)
        .sort();
      const candidateIds = candidates
        .filter((candidate) => candidate.sourceSnapshots.some(isExactSourceClause))
        .map((candidate) => candidate.candidateId)
        .sort();
      return {
        ...structuredClone(definition),
        applicabilityFacts,
        generationRunIds,
        candidateIds,
      };
    });
  }

  private mockCandidateSourceCurrentness(candidate: GovernedCandidateView): GovernedCandidateView {
    const projected = structuredClone(candidate);
    projected.questions = projected.questions.map((question) => {
      if (question.regulatoryTrace.state === "SOURCE_MAPPING_REQUIRED") {
        return {
          ...question,
          regulatoryTrace: {
            ...question.regulatoryTrace,
            currentnessState: "SOURCE_MAPPING_REQUIRED",
            technicalReviewState: "NOT_AVAILABLE",
          },
        };
      }
      const citation = question.citations[0];
      const stale = !!citation && [...this.governedState.sourceCurrentnessEvents.values()].some((event) =>
        event.previousSourceSnapshotId === citation.sourceSnapshotId &&
        event.previousSourceHash === question.regulatoryTrace.sha256,
      );
      if (!stale) return question;
      return {
        ...question,
        scopeRecommendation: {
          ...question.scopeRecommendation,
          guardrails: { ...question.scopeRecommendation.guardrails, sourceChanged: true },
        },
        regulatoryTrace: { ...question.regulatoryTrace, currentnessState: "STALE" },
      };
    });
    return projected;
  }

  // Match the Go read model: the immutable persisted question remains in the
  // TECHNICAL_REVIEW_REQUIRED state, while an attributed approval decorates
  // only the returned candidate projection.
  private projectMockGovernedCandidate(candidate: GovernedCandidateView): GovernedCandidateView {
    const projected = this.mockCandidateSourceCurrentness(candidate);
    if (!["TECHNICALLY_APPROVED", "PUBLISHED"].includes(projected.status)) {
      return projected;
    }
    projected.questions = projected.questions.map((question) => {
      if (question.regulatoryTrace.state === "SOURCE_MAPPING_REQUIRED") return question;
      return {
        ...question,
        scopeRecommendation: {
          ...question.scopeRecommendation,
          approvalReviewState: "TECHNICALLY_APPROVED",
        },
        regulatoryTrace: {
          ...question.regulatoryTrace,
          technicalReviewState: "TECHNICALLY_APPROVED",
        },
      };
    });
    return projected;
  }

  private projectMockGovernedRun(run: GovernedGenerationRunView): GovernedGenerationRunView {
    if (!run.candidate) return structuredClone(run);
    const current = this.governedCandidates.get(run.candidate.candidateId) ?? run.candidate;
    return { ...structuredClone(run), candidate: this.projectMockGovernedCandidate(current) };
  }

  private sourceCurrentnessEventForBundle(bundle: GovernedCandidateBundleInput): MockSourceCurrentnessEvent | null {
    const resolved = bundle.inspectionChecklist.questions.some((question) => question.regulatoryTrace.state === "RESOLVED");
    const binding = bundle.sourceCurrentness;
    if (!resolved) {
      if (binding) {
        throw new GovernedValidationError([governedValidationIssue("candidateBundle.sourceCurrentness", "SOURCE_CURRENTNESS_UNEXPECTED", "a literal SOURCE_MAPPING_REQUIRED Draft cannot claim source currentness")]);
      }
      return null;
    }
    if (!binding) {
      throw new GovernedValidationError([governedValidationIssue("candidateBundle.sourceCurrentness", "SOURCE_CURRENTNESS_REQUIRED", "a traced candidate must bind an explicitly activated source currentness record")]);
    }
    const current = bundle.generationRequest.sourceSnapshots[0];
    const previousSourceSnapshotId = binding.previousSourceSnapshotId ?? null;
    const previousSourceHash = binding.previousSourceHash ?? null;
    if (!current || binding.currentSourceSnapshotId !== current.sourceSnapshotId || binding.currentSourceHash !== current.sourceHash ||
      (previousSourceSnapshotId === null) !== (previousSourceHash === null)) {
      throw new GovernedValidationError([governedValidationIssue("candidateBundle.sourceCurrentness", "SOURCE_CURRENTNESS_BINDING_MISMATCH", "candidate currentness binding must exactly match its frozen source snapshot and predecessor pair")]);
    }
    const event = [...this.governedState.sourceCurrentnessEvents.values()].find((candidate) =>
      candidate.currentSourceSnapshotId === binding.currentSourceSnapshotId &&
      candidate.currentSourceHash === binding.currentSourceHash &&
      candidate.previousSourceSnapshotId === previousSourceSnapshotId &&
      candidate.previousSourceHash === previousSourceHash,
    );
    if (!event) {
      throw new GovernedValidationError([governedValidationIssue("candidateBundle.sourceCurrentness", "SOURCE_CURRENTNESS_REQUIRED", "source currentness must be activated before this traced candidate can be imported")]);
    }
    if (previousSourceSnapshotId !== null && !event.impactReviewDraftId) {
      throw new GovernedValidationError([governedValidationIssue("candidateBundle.sourceCurrentness", "IMPACT_REVIEW_DRAFT_REQUIRED", "a source-change candidate must bind an immutable impact-review Draft")]);
    }
    return event;
  }

  private recordMockSourceImpactCandidateBinding(candidate: GovernedCandidateView, event: MockSourceCurrentnessEvent | null): void {
    if (!event?.impactReviewDraftId) return;
    const existing = this.governedState.sourceImpactCandidateBindings.get(candidate.candidateId);
    if (existing === event.impactReviewDraftId) return;
    if (existing) throw new BackendConflictError("Governed candidate is already bound to a different immutable source impact-review Draft.");
    this.governedState.sourceImpactCandidateBindings.set(candidate.candidateId, event.impactReviewDraftId);
    const eventId = `AE-SOURCE-IMPACT-${candidate.candidateId}`;
    const occurredAt = this.store.clock();
    this.store.execute(`GOVERNED-SOURCE-IMPACT-BIND-${candidate.candidateId}`, event, (state) => {
      state.auditEvents.push({
        eventId,
        occurredAt,
        actorRole: "admin",
        actorSubjectId: this.principal.subjectId,
        action: "regulatory.source_impact_candidate_bound",
        entityType: "REGULATORY_SOURCE_IMPACT",
        entityId: candidate.candidateId,
        beforeStatus: event.previousSourceHash,
        afterStatus: event.currentSourceHash,
        reason: "Candidate was bound to an already activated immutable source impact-review Draft.",
        entityRevision: candidate.revision,
      });
      return true;
    });
  }

  private async activateMockSourceCurrentness(input: GovernedSourceCurrentnessActivationInput): Promise<GovernedSourceCurrentnessActivationView> {
    const hasPreviousSourceSnapshotId = Object.prototype.hasOwnProperty.call(input, "previousSourceSnapshotId");
    const hasPreviousSourceHash = Object.prototype.hasOwnProperty.call(input, "previousSourceHash");
    const previousSourceSnapshotId = input.previousSourceSnapshotId ?? null;
    const previousSourceHash = input.previousSourceHash ?? null;
    const semantic = await governedCanonicalSHA256({
      operationId: input.operationId,
      idempotencyKey: input.idempotencyKey,
      currentSourceSnapshotId: input.currentSourceSnapshotId,
      currentSourceHash: input.currentSourceHash,
      // The Go command represents an explicit JSON null as its canonical empty
      // predecessor value. Keep mock replay semantics byte-for-byte compatible
      // while the transport itself still requires the two explicit null fields.
      previousSourceSnapshotId: previousSourceSnapshotId ?? "",
      previousSourceHash: previousSourceHash ?? "",
      reason: input.reason,
    });
    const replay = this.governedState.sourceCurrentnessCommands.get(input.operationId) ?? this.governedState.sourceCurrentnessCommands.get(input.idempotencyKey);
    if (replay) {
      if (replay.operationId !== input.operationId || replay.idempotencyKey !== input.idempotencyKey || replay.semantic !== semantic) {
        throw new BackendConflictError("Source-currentness activation command identity was reused with different semantics.");
      }
      return structuredClone(replay.view);
    }
    const seededReplayEvent = [...this.governedState.sourceCurrentnessEvents.values()].find((event) =>
      event.operationId === input.operationId || event.idempotencyKey === input.idempotencyKey,
    );
    if (seededReplayEvent) {
      const seededSemantic = await governedCanonicalSHA256({
        operationId: seededReplayEvent.operationId,
        idempotencyKey: seededReplayEvent.idempotencyKey,
        currentSourceSnapshotId: seededReplayEvent.currentSourceSnapshotId,
        currentSourceHash: seededReplayEvent.currentSourceHash,
        previousSourceSnapshotId: seededReplayEvent.previousSourceSnapshotId ?? "",
        previousSourceHash: seededReplayEvent.previousSourceHash ?? "",
        reason: seededReplayEvent.reason,
      });
      if (seededReplayEvent.operationId !== input.operationId || seededReplayEvent.idempotencyKey !== input.idempotencyKey || seededSemantic !== semantic) {
        throw new BackendConflictError("Source-currentness activation command identity was reused with different semantics.");
      }
      const view: GovernedSourceCurrentnessActivationView = {
        eventId: seededReplayEvent.eventId,
        impactReviewDraftId: seededReplayEvent.impactReviewDraftId,
        sourceIdentity: seededReplayEvent.sourceIdentity,
        previousSourceSnapshotId: seededReplayEvent.previousSourceSnapshotId,
        previousSourceHash: seededReplayEvent.previousSourceHash,
        currentSourceSnapshotId: seededReplayEvent.currentSourceSnapshotId,
        currentSourceHash: seededReplayEvent.currentSourceHash,
        status: seededReplayEvent.impactReviewDraftId ? "IMPACT_REVIEW_DRAFT" : "BASELINE_ACTIVATED",
        activatedAt: seededReplayEvent.activatedAt,
      };
      const command: MockSourceCurrentnessCommand = { operationId: input.operationId, idempotencyKey: input.idempotencyKey, semantic, view };
      this.governedState.sourceCurrentnessCommands.set(input.operationId, command);
      this.governedState.sourceCurrentnessCommands.set(input.idempotencyKey, command);
      return structuredClone(view);
    }
    if (!hasPreviousSourceSnapshotId || !hasPreviousSourceHash ||
      !input.operationId || !input.idempotencyKey || !input.currentSourceSnapshotId || !input.currentSourceHash || !input.reason ||
      (previousSourceSnapshotId === null) !== (previousSourceHash === null) ||
      (previousSourceSnapshotId !== null && previousSourceSnapshotId === input.currentSourceSnapshotId)) {
      throw new GovernedValidationError([governedValidationIssue("sourceCurrentness", "SOURCE_CURRENTNESS_INVALID", "activation requires a complete current snapshot and either both predecessor fields or neither")]);
    }
    const sourceIdentity = input.currentSourceSnapshotId === "SOURCE-SYNTHETIC-OPS-AOC" && input.currentSourceHash === SYNTHETIC_SOURCE_HASH
      ? "SYNTHETIC-OPS-AOC"
      : input.currentSourceSnapshotId === "SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2" && input.currentSourceHash === SYNTHETIC_HYBRID_RECONCILED_BUNDLE.generationRequest.sourceSnapshots[0]!.sourceHash
        ? "SYNTHETIC-OPS-AOC"
        : null;
    if (!sourceIdentity || (previousSourceSnapshotId !== null && !(previousSourceSnapshotId === "SOURCE-SYNTHETIC-OPS-AOC" && previousSourceHash === SYNTHETIC_SOURCE_HASH))) {
      throw new GovernedValidationError([governedValidationIssue("sourceCurrentness", "SOURCE_CURRENTNESS_BINDING_MISMATCH", "activation must use an exact known source snapshot/hash predecessor-current chain")]);
    }
    const chain = [...this.governedState.sourceCurrentnessEvents.values()]
      .filter((event) => event.sourceIdentity === sourceIdentity)
      .sort((left, right) => left.sequence - right.sequence);
    const head = chain.at(-1);
    if (!head && previousSourceSnapshotId !== null) {
      throw new GovernedValidationError([governedValidationIssue("sourceCurrentness", "SOURCE_CURRENTNESS_BASELINE_REQUIRED", "a source change requires an explicit activated baseline")]);
    }
    if (head && previousSourceSnapshotId === null) {
      throw new BackendConflictError("Source identity already has an activated baseline/current head.");
    }
    if (head && (head.currentSourceSnapshotId !== previousSourceSnapshotId || head.currentSourceHash !== previousSourceHash)) {
      throw new GovernedValidationError([governedValidationIssue("sourceCurrentness", "SOURCE_CURRENTNESS_PREDECESSOR_MISMATCH", "activation predecessor must exactly match the latest activated source snapshot/hash")]);
    }
    const duplicate = chain.find((event) =>
      event.currentSourceSnapshotId === input.currentSourceSnapshotId && event.currentSourceHash === input.currentSourceHash &&
      event.previousSourceSnapshotId === previousSourceSnapshotId && event.previousSourceHash === previousSourceHash,
    );
    if (duplicate) throw new BackendConflictError("Exact source-currentness transition is already activated.");
    const transitionDigest = await governedCanonicalSHA256({
      sourceIdentity,
      currentSourceSnapshotId: input.currentSourceSnapshotId,
      currentSourceHash: input.currentSourceHash,
      previousSourceSnapshotId: previousSourceSnapshotId ?? "",
      previousSourceHash: previousSourceHash ?? "",
    });
    const suffix = transitionDigest.slice("sha256:".length, "sha256:".length + 24);
    const impactReviewDraftId = previousSourceSnapshotId === null ? null : `SRC-IMPACT-DRAFT-${suffix}`;
    const activatedAt = this.store.clock();
    const event: MockSourceCurrentnessEvent = {
      eventId: `SRC-CURRENTNESS-${suffix}`,
      impactReviewDraftId,
      sourceIdentity,
      previousSourceSnapshotId,
      previousSourceHash,
      currentSourceSnapshotId: input.currentSourceSnapshotId,
      currentSourceHash: input.currentSourceHash,
      sequence: (head?.sequence ?? 0) + 1,
      operationId: input.operationId,
      idempotencyKey: input.idempotencyKey,
      semantic,
      reason: input.reason,
      activatedAt,
    };
    const view: GovernedSourceCurrentnessActivationView = {
      eventId: event.eventId,
      impactReviewDraftId,
      sourceIdentity,
      previousSourceSnapshotId,
      previousSourceHash,
      currentSourceSnapshotId: event.currentSourceSnapshotId,
      currentSourceHash: event.currentSourceHash,
      status: impactReviewDraftId ? "IMPACT_REVIEW_DRAFT" : "BASELINE_ACTIVATED",
      activatedAt,
    };
    this.governedState.sourceCurrentnessEvents.set(event.eventId, event);
    const command: MockSourceCurrentnessCommand = { operationId: input.operationId, idempotencyKey: input.idempotencyKey, semantic, view };
    this.governedState.sourceCurrentnessCommands.set(input.operationId, command);
    this.governedState.sourceCurrentnessCommands.set(input.idempotencyKey, command);
    this.store.execute(`GOVERNED-SOURCE-CURRENTNESS-${event.eventId}`, event, (state) => {
      state.auditEvents.push({
        eventId: `AE-SOURCE-CURRENTNESS-${suffix}`,
        occurredAt: activatedAt,
        actorRole: "admin",
        actorSubjectId: this.principal.subjectId,
        action: "regulatory.source_currentness_activated",
        entityType: "REGULATORY_SOURCE_CURRENTNESS",
        entityId: event.eventId,
        beforeStatus: previousSourceHash,
        afterStatus: input.currentSourceHash,
        reason: input.reason,
        entityRevision: 1,
      });
      return true;
    });
    return structuredClone(view);
  }

  private mockGovernedBlockers(candidate: GovernedCandidateView): GovernedValidationIssue[] {
    const currentness = this.mockCandidateSourceCurrentness(candidate);
    return [
      ...structuredClone(this.governedState.blockers.get(candidate.candidateId) ?? []),
      ...currentness.questions.flatMap((question) => governedQuestionGovernanceIssuesForTest(question)),
    ];
  }

  // This internal mock-only test-profile seam mirrors the separate
  // applicability/materialization command used by the canonical HTTP test
  // boundary. Publication must never create an executable inspection package.
  materializeSyntheticGovernedPackageForTest() {
    const candidate = this.governedCandidate;
    this.mockGovernedAssignmentFor(candidate);
    if (candidate.status !== "PUBLISHED") {
      throw new BackendConflictError("Only an exact published governed candidate can be materialized.");
    }
    const publication = this.governedState.publicationDecisions.get(candidate.candidateId);
    if (!publication) {
      throw new BackendInvariantError("Published governed candidate is missing its immutable publication decision.");
    }
    const selection = {
      organizationId: "ORG-SYNTHETIC-AOC",
      inspectionType: "RAMP_INSPECTION",
      targetId: "TARGET-SYNTHETIC-AOC",
      targetKind: "ORGANIZATION",
      departmentId: "FLIGHT_OPERATIONS_INSPECTORATE",
    } as const;
    const packageId = "PKG-SYNTHETIC-OPS-AOC-001";
    const inspectionId = "AUD-SYNTHETIC-OPS-AOC-001";
    return this.store.execute("TEST-PROFILE-MATERIALIZE-SYNTHETIC", {
      candidateId: candidate.candidateId,
      candidateContentDigest: candidate.contentDigest,
      templateVersionId: publication.templateVersionId,
      selection,
    }, (state) => {
      state.packages[packageId] = {
        id: packageId,
        auditId: inspectionId,
        organizationId: selection.organizationId,
        organizationName: "Synthetic Internal AOC Test Profile",
        title: "Synthetic governed ramp inspection package",
        packageVersion: 1,
        schemaVersion: 1,
        protocolVersion: 1,
        templateVersionId: publication.templateVersionId,
        packageDigest: publication.candidateContentDigest,
        expiresAt: "2026-08-29T00:00:00.000Z",
        checklistStatus: "IN_PROGRESS",
        checklistRevision: 1,
        questions: candidate.questions.map((question) => ({
          id: question.questionId,
          sectionId: "SYNTHETIC-OPS-AOC",
          prompt: question.prompt,
          regulatoryReference: question.citations.map((citation) => citation.locator).join("; "),
          expectedEvidence: question.expectedEvidence.join("; "),
          allowedAnswers: question.allowedAnswers as ChecklistAnswer[],
          commentRequiredFor: ["NON_COMPLIANT", "OBSERVATION"],
          assignedInspectorUserIds: ["USR-INSPECTOR-AMINA"],
          currentResponse: null,
        })),
      };
      state.auditEvents.push({
        eventId: "AE-TASK9-MATERIALIZE-SYNTHETIC",
        occurredAt: this.store.clock(),
        actorRole: "manager",
        actorSubjectId: this.principal.subjectId,
        action: "CHECKLIST_PACKAGE_MATERIALIZED",
        entityType: "INSPECTION_PACKAGE",
        entityId: packageId,
        beforeStatus: "PUBLISHED",
        afterStatus: "IN_PROGRESS",
        reason: "Internal synthetic test-profile applicability materialization.",
        entityRevision: 1,
      });
      return {
        inspectionId,
        packageId,
        templateVersionId: publication.templateVersionId,
        packageDigest: publication.candidateContentDigest,
        selection,
      };
    });
  }

  private mockGovernedAssignments(): MockGovernedManagerAssignment[] {
    requireRole(this.principal, ["manager"], "Department Manager governed-checklist authority is required.");
    const asOf = this.store.clock().slice(0, 10);
    const latestByRoot = new Map<string, MockGovernedManagerAssignment>();
    for (const assignment of MOCK_GOVERNED_MANAGER_ASSIGNMENTS[this.principal.subjectId] ?? []) {
      const current = latestByRoot.get(assignment.membershipRootId);
      if (!current || assignment.revision > current.revision) {
        latestByRoot.set(assignment.membershipRootId, assignment);
      }
    }
    const assignments = [...latestByRoot.values()]
      .filter((assignment) =>
        assignment.status === "ACTIVE" &&
        assignment.departmentActive &&
        assignment.organizationalUnitActive &&
        assignment.effectiveFrom <= asOf &&
        (assignment.effectiveTo === null || assignment.effectiveTo > asOf));
    if (assignments.length === 0) {
      throw new BackendAuthorizationInvariantError(
        "A current active Department Manager assignment is required.",
      );
    }
    return assignments;
  }

  private mockGovernedAssignmentFor(candidate: GovernedCandidateView): MockGovernedManagerAssignment {
    const assignment = this.mockGovernedAssignments().find((current) =>
      candidate.requiredOwners.some((owner) =>
        owner.approvalRequired &&
        owner.departmentId === current.departmentId &&
        owner.organizationalUnitId === current.organizationalUnitId));
    if (!assignment) {
      throw new BackendAuthorizationInvariantError(
        "The candidate is outside the manager's current exact department and unit.",
      );
    }
    return assignment;
  }

  readonly communications: DemoBackend["communications"] = {
    list: async (input) => {
      requireDemoCapability(this.principal, "communications");
      return this.store.read((state) => {
        let items = state.communications;
        if (this.principal.role === "auditee") {
          items = items.filter(
            (item) => item.organizationId === this.principal.organizationId && (
              (item.direction === "CAA_TO_AUDITEE" && item.audience === "AUDITEE") ||
              (
                item.direction === "AUDITEE_TO_CAA" &&
                item.audience === "CAA" &&
                item.senderSubjectId === this.principal.subjectId
              )
            ),
          );
        } else if (input.organizationId) {
          items = items.filter((item) => item.organizationId === input.organizationId);
        }
        return { items: items.map(publicCommunication), nextCursor: null };
      });
    },
    send: async (input) => {
      requireDemoCapability(this.principal, "communications");
      requireNonEmpty(input.idempotencyKey, "Idempotency key");
      requireNonEmpty(input.subject, "Communication subject");
      requireNonEmpty(input.body, "Communication body");
      return this.store.execute(input.idempotencyKey, input, (state) => {
        if (this.principal.role === "auditee") {
          if (input.organizationId !== this.principal.organizationId || input.audience !== "CAA") {
            throw new BackendAuthorizationInvariantError("Auditee messages may only be sent to CAA for its own organization.");
          }
        } else if (input.audience === "CAA") {
          requireRole(this.principal, ["inspector", "leadInspector", "manager", "gm", "executiveDirector", "admin"], "CAA communication authority is required.");
        }
        const existing = state.communications.find((item) => item.id === input.idempotencyKey);
        if (existing) {
          requireRevision(existing.revision, input.expectedRevision, "Communication");
        } else if (input.expectedRevision !== null) {
          throw new BackendConflictError(
            `Communication revision conflict: expected ${input.expectedRevision}, received null.`,
          );
        }
        const message: MockCommunication = {
          id: input.idempotencyKey,
          organizationId: input.organizationId,
          subject: input.subject.trim(),
          body: input.body.trim(),
          audience: input.audience,
          direction: this.principal.role === "auditee"
            ? "AUDITEE_TO_CAA"
            : input.audience === "AUDITEE"
              ? "CAA_TO_AUDITEE"
              : "CAA_INTERNAL",
          senderSubjectId: this.principal.subjectId,
          revision: 1,
          createdAt: this.store.clock(),
        };
        state.communications.push(message);
        const recipientProfiles = Object.values(state.profiles).filter((profile) => {
          if (message.audience === "AUDITEE") {
            return profile.role === "auditee" &&
              profile.organizationId === message.organizationId;
          }
          return profile.role !== "auditee" &&
            profile.subjectId !== this.principal.subjectId;
        });
        const title = message.direction === "AUDITEE_TO_CAA"
          ? "New Auditee communication"
          : message.direction === "CAA_INTERNAL"
            ? "New Internal CAA Note"
            : "New CAA communication";
        for (const profile of recipientProfiles) {
          const notificationSequence = state.notifications.filter(
            ({ id }) => id.startsWith("notification-candidate-"),
          ).length + 1;
          const notificationId = `notification-candidate-${pad(notificationSequence)}`;
          if (state.notifications.some((notification) => notification.id === notificationId)) {
            continue;
          }
          state.notifications.push({
            id: notificationId,
            subjectId: profile.subjectId,
            title,
            body: `${message.subject} — open the authorized message record for details.`,
            readAt: null,
            emailDeliveryStatus: "NOT_CONFIGURED",
            emailDeliveryAttempts: 0,
            emailAcceptedAt: null,
            emailNextAttemptAt: null,
            revision: 1,
          });
        }
        return publicCommunication(message);
      });
    },
  };

  readonly auditeeCoordination: DemoBackend["auditeeCoordination"] = {
    list: async () => {
      requireDemoCapability(this.principal, "auditeeCoordination");
      requireRole(this.principal, ["auditee"], "Auditee authority is required for inspection coordination.");
      return this.store.read((state) => ({
        items: state.assignments
          .filter((assignment) => auditeeCanViewCalendarAssignment(this.principal, assignment))
          .map((assignment) => auditeeCoordinationProjection(state, assignment)),
        nextCursor: null,
      }));
    },
    respond: async (input) => {
      requireDemoCapability(this.principal, "auditeeCoordination");
      requireRole(this.principal, ["auditee"], "Auditee authority is required for inspection coordination.");
      requireNonEmpty(input.idempotencyKey, "Idempotency key");
      return this.store.execute(input.idempotencyKey, input, (state) => {
        requireAuditeeOrganization(this.principal, input.organizationId);
        const assignment = state.assignments.find((candidate) => candidate.auditId === input.auditId);
        if (
          !assignment ||
          assignment.organizationId !== input.organizationId ||
          !auditeeCanViewCalendarAssignment(this.principal, assignment)
        ) {
          throw new BackendAuthorizationInvariantError("Inspection coordination record is unavailable to this Auditee.");
        }
        const current = auditeeCoordinationProjection(state, assignment);
        requireRevision(current.revision, input.expectedRevision, "Inspection coordination");
        if (input.decision === "PROPOSE_ALTERNATIVE") {
          requireNonEmpty(input.alternativeDate ?? "", "Alternative date");
        } else if (input.alternativeDate !== null) {
          throw new BackendInvariantError("Confirming the proposed date cannot include an alternative date.");
        }
        state.auditeeCoordinationResponses[input.auditId] = {
          auditId: input.auditId,
          organizationId: input.organizationId,
          status: input.decision === "CONFIRM" ? "CONFIRMED" : "ALTERNATIVE_PROPOSED",
          alternativeDate: input.decision === "PROPOSE_ALTERNATIVE" ? input.alternativeDate : null,
          revision: current.revision + 1,
        };
        return auditeeCoordinationProjection(state, assignment);
      });
    },
  };

  readonly auditeeReports: DemoBackend["auditeeReports"] = {
    listReleased: async (input) => {
      requireDemoCapability(this.principal, "auditeeReports");
      requireRole(this.principal, ["auditee"], "Auditee authority is required for released report projections.");
      return this.store.read((state) => {
        const items = Object.values(state.reportVersions)
          .filter((report) => auditeeCanViewReleasedReport(state, this.principal, report.reportVersionId))
          .map((report) => auditeeReleasedReportProjection(state, this.principal, report.reportVersionId))
          .filter((report) => !input.kind || report.kind === input.kind)
          .sort((left, right) => (left.kind === right.kind ? left.reportVersionId.localeCompare(right.reportVersionId) : left.kind === "PRELIMINARY" ? -1 : 1));
        return { items, nextCursor: null };
      });
    },
    getReleased: async ({ reportVersionId }) => {
      requireDemoCapability(this.principal, "auditeeReports");
      return this.store.read((state) => auditeeReleasedReportProjection(state, this.principal, reportVersionId));
    },
  };

  readonly calendar: DemoBackend["calendar"] = {
    list: async (input) => {
      requireDemoCapability(this.principal, "calendar");
      return this.store.read((state) => {
        let assignments = state.assignments;
        if (this.principal.role === "inspector" || this.principal.role === "auditee") {
          assignments = assignments.filter((assignment) => canViewCalendarAssignment(state, this.principal, assignment));
        }
        if (input.organizationId) assignments = assignments.filter((assignment) => assignment.organizationId === input.organizationId);
        return {
          items: assignments.map((assignment) => calendarAssignmentProjection(state, this.principal, assignment)),
          nextCursor: null,
        };
      });
    },
    openItem: async ({ calendarItemId }) =>
      (requireDemoCapability(this.principal, "calendar"), this.store.read((state) => {
        const visibleAssignments = state.assignments.filter((assignment) => canViewCalendarAssignment(state, this.principal, assignment));
        const items = visibleAssignments.map((assignment) => calendarAssignmentProjection(state, this.principal, assignment));
        const item = items.find((candidate) => candidate.id === calendarItemId);
        if (!item) {
          throw new BackendAuthorizationInvariantError("Calendar item is unavailable to this principal.");
        }
        return item;
      })),
  };

  readonly profiles: DemoBackend["profiles"] = {
    getMine: async () => (requireDemoCapability(this.principal, "profiles"), this.store.read((state) => profileForPrincipal(state, this.principal))),
    updateMine: async (input) => {
      requireDemoCapability(this.principal, "profiles");
      requireNonEmpty(input.idempotencyKey, "Idempotency key");
      const displayName = requireNonEmpty(input.displayName, "Display name");
      return this.store.execute(input.idempotencyKey, input, (state) => {
        const profile = profileForPrincipal(state, this.principal);
        requireRevision(profile.revision, input.expectedRevision, "Profile");
        profile.displayName = displayName;
        profile.revision += 1;
        return profile;
      });
    },
  };

  readonly teams: DemoBackend["teams"] = {
    list: async (input) => {
      requireDemoCapability(this.principal, "teams");
      return this.store.read((state) => {
        if (this.principal.role === "auditee") {
          throw new BackendAuthorizationInvariantError("CAA team assignments are not available to Auditee users.");
        }
        let items: TeamMemberView[] = Object.values(state.profiles).map((profile) => ({ ...profile }));
        if (input.role) items = items.filter((member) => member.role === input.role);
        return { items, nextCursor: null };
      });
    },
    openMember: async ({ subjectId }) =>
      (requireDemoCapability(this.principal, "teams"), this.store.read((state) => {
        if (this.principal.role === "auditee") throw new BackendAuthorizationInvariantError("CAA team assignments are not available to Auditee users.");
        const profile = state.profiles[subjectId];
        if (!profile) throw new BackendInvariantError(`Team member ${subjectId} was not found.`);
        return profile;
      })),
    listAuditTeams: async ({ limit }) => {
      requireDemoCapability(this.principal, "teams");
      requireRole(this.principal, ["manager"], "Department Manager authority is required for Audit team projections.");
      return this.store.read((state) => ({
        items: state.assignments.slice(0, limit ?? state.assignments.length).map((assignment) => inspectionTeamForAudit(state, assignment.auditId)),
        nextCursor: null,
      }));
    },
    openAuditTeam: async ({ auditId }) => {
      requireDemoCapability(this.principal, "teams");
      requireRole(this.principal, ["manager"], "Department Manager authority is required for Audit team projections.");
      return this.store.read((state) => inspectionTeamForAudit(state, auditId));
    },
  };

  readonly risk: DemoBackend["risk"] = {
    getOverview: async (input) =>
      (requireDemoCapability(this.principal, "risk"), this.store.read((state) => {
        if (this.principal.role === "auditee") {
          throw new BackendAuthorizationInvariantError("Internal CAA risk scoring is unavailable to Auditee users.");
        }
        const organizationId = input.organizationId ?? null;
        const findings = Object.values(state.findings).filter((finding) => !organizationId || finding.organizationId === organizationId);
        const overdueFindingCount = findings.filter((finding) => finding.dueState === "OVERDUE").length;
        const output: RiskOverviewView = {
          organizationId,
          overdueFindingCount,
          openFindingCount: findings.filter((finding) => finding.status !== "CLOSED").length,
          repeatFindingCount: findings.filter((finding) => finding.repeatFinding).length,
          advisoryHealth: organizationId === "ORG-FLY-NAMIBIA"
            ? {
                score: 74,
                band: "Needs Attention",
                basis: "CONFIGURED_DEMO_SCENARIO",
                recommendedAction: "Prioritize Cabin Inspection focus on emergency-equipment serviceability and CAP effectiveness.",
              }
            : null,
          revision: 1,
        };
        return output;
      })),
    getManagementProjection: async () =>
      (requireDemoCapability(this.principal, "risk"), this.store.read((state) => {
        requireRole(
          this.principal,
          ["manager"],
          "Department Manager authority is required for the management risk projection.",
        );
        const findings = Object.values(state.findings).map((finding) => {
          const assignment = state.assignments.find((candidate) => candidate.auditId === finding.auditId) ?? null;
          return {
            findingId: finding.id,
            findingNumber: finding.findingNumber,
            organizationId: finding.organizationId,
            organizationName: finding.organizationName,
            inspectionId: finding.auditId,
            inspectionTitle: assignment?.title ?? null,
            department: null,
            title: finding.title,
            severity: finding.severity,
            riskLevel: managementRiskLevel(finding.severity),
            status: finding.status,
            issuedAt: finding.issuedAt,
            dueState: finding.dueState,
            capRequired: finding.capRequired,
          };
        });
        const capEffectiveness = Object.values(state.findings)
          .map((finding) => {
            const latestCap = state.capRevisions
              .filter((candidate) => candidate.findingId === finding.id)
              .sort((left, right) => right.version - left.version)[0] ?? null;
            const identity = {
              findingId: finding.id,
              findingNumber: finding.findingNumber,
              organizationId: finding.organizationId,
              organizationName: finding.organizationName,
              findingStatus: finding.status,
              closureBasis: finding.closureBasis,
              capId: latestCap?.capId ?? null,
              capRevisionId: latestCap?.id ?? null,
              capRevision: latestCap?.version ?? null,
              capStatus: latestCap?.status ?? null,
            };
            if (!latestCap) {
              return {
                ...identity,
                state: "NOT_ELIGIBLE" as const,
                reason: `Finding ${finding.id} has no CAP revision; effectiveness is unavailable.`,
              };
            }
            if (finding.status !== "CLOSED") {
              return {
                ...identity,
                state: "NOT_ELIGIBLE" as const,
                reason: `Finding ${finding.id} is ${finding.status}; effectiveness requires a CLOSED Finding with a closure or verification basis.`,
              };
            }
            if (!finding.closureBasis) {
              return {
                ...identity,
                state: "NOT_ELIGIBLE" as const,
                reason: `Finding ${finding.id} is CLOSED without a typed closure or verification basis; effectiveness is unavailable.`,
              };
            }
            return {
              ...identity,
              state: "PENDING_POST_CLOSURE_VERIFICATION" as const,
              reason: `Finding ${finding.id} closed with ${finding.closureBasis}; no typed post-closure effectiveness verification record is available.`,
            };
          })
          .filter((record) => record.capRevisionId !== null);
        return {
          findings,
          capEffectiveness,
          generatedAt: this.store.clock(),
          revision: 1,
        } satisfies RiskManagementProjectionView;
      })),
    openFinding: async ({ findingId }) =>
      (requireDemoCapability(this.principal, "risk"), this.store.read((state) => {
        if (this.principal.role === "auditee") throw new BackendAuthorizationInvariantError("Internal CAA risk scoring is unavailable to Auditee users.");
        return findingForPrincipal(state, this.principal, findingId);
      })),
  };

  readonly documents: DemoBackend["documents"] = {
    list: async (input) => {
      requireDemoCapability(this.principal, "documents");
      return this.store.read((state) => {
        const organizationId = this.principal.role === "auditee" ? this.principal.organizationId : input.organizationId;
        const reports: DocumentMetadataView[] = Object.values(state.reportVersions)
          .filter((report) => !organizationId || report.organizationId === organizationId)
          .filter((report) => this.principal.role !== "auditee" || auditeeCanViewReleasedReport(state, this.principal, report.reportVersionId))
          .map((report) => this.principal.role === "auditee"
            ? auditeeReportDocumentMetadata(state, this.principal, report)
            : {
                id: report.reportVersionId,
                organizationId: report.organizationId,
                title: `Report ${report.reportId}`,
                kind: "REPORT" as const,
                version: report.version,
                revision: report.revision,
                createdAt: report.issuedAt ?? this.store.clock(),
              });
        const evidence: DocumentMetadataView[] = state.evidenceVersions
          .filter((version) => !organizationId || version.organizationId === organizationId)
          .map((version) => ({
            id: version.id,
            organizationId: version.organizationId,
            title: version.fileName,
            kind: "EVIDENCE",
            version: version.version,
            revision: version.revision,
            createdAt: version.submittedAt,
            ...(this.principal.role === "auditee" ? {
              publicReviewResult: version.reviewState,
              downloadFileName: version.fileName,
            } : {}),
          }));
        return { items: [...reports, ...evidence], nextCursor: null };
      });
    },
    open: async ({ documentId }) =>
      (requireDemoCapability(this.principal, "documents"), this.store.read((state) => {
        const organizationId = this.principal.role === "auditee" ? this.principal.organizationId : undefined;
        const documents: DocumentMetadataView[] = [
          ...Object.values(state.reportVersions)
            .filter((report) => this.principal.role !== "auditee" || auditeeCanViewReleasedReport(state, this.principal, report.reportVersionId))
            .map((report) => this.principal.role === "auditee"
              ? auditeeReportDocumentMetadata(state, this.principal, report)
              : {
                  id: report.reportVersionId,
                  organizationId: report.organizationId,
                  title: `Report ${report.reportId}`,
                  kind: "REPORT" as const,
                  version: report.version,
                  revision: report.revision,
                  createdAt: report.issuedAt ?? this.store.clock(),
                }),
          ...state.evidenceVersions.map((version) => ({
            id: version.id,
            organizationId: version.organizationId,
            title: version.fileName,
            kind: "EVIDENCE" as const,
            version: version.version,
            revision: version.revision,
            createdAt: version.submittedAt,
            ...(this.principal.role === "auditee" ? {
              publicReviewResult: version.reviewState,
              downloadFileName: version.fileName,
            } : {}),
          })),
        ];
        const document = documents.find((candidate) => candidate.id === documentId);
        if (!document || (organizationId && document.organizationId !== organizationId)) throw new BackendAuthorizationInvariantError("Document is unavailable to this principal.");
        return document;
      })),
  };

  readonly notifications: DemoBackend["notifications"] = {
    list: async () => (requireDemoCapability(this.principal, "notifications"), this.store.read((state) => ({
      items: state.notifications.filter((notification) => notification.subjectId === this.principal.subjectId),
      nextCursor: null,
    }))),
    markRead: async (input) => {
      requireDemoCapability(this.principal, "notifications");
      requireNonEmpty(input.idempotencyKey, "Idempotency key");
      return this.store.execute(input.idempotencyKey, input, (state) => {
        const notification = state.notifications.find((item) => item.id === input.notificationId);
        if (!notification || notification.subjectId !== this.principal.subjectId) {
          throw new BackendAuthorizationInvariantError("Notification is unavailable to this principal.");
        }
        requireRevision(notification.revision, input.expectedRevision, "Notification");
        notification.readAt = this.store.clock();
        notification.revision += 1;
        return notification as NotificationView;
      });
    },
  };

  readonly administration: DemoBackend["administration"] = {
    getScreenProjection: async ({ screenId }) => (requireDemoCapability(this.principal, "administration"), this.store.read((state) => screenProjectionFor(state, this.principal, screenId))),
    listScreenProjections: async () => (requireDemoCapability(this.principal, "administration"), this.store.read((state) =>
      REACT_ROUTE_CONTRACTS
        .filter((route) => route.requiredRole === null || route.requiredRole === this.principal.role)
        .map((route) => screenProjectionFor(state, this.principal, route.id)),
    )),
    invokeVisibleAction: async ({ screenId, actionId }) =>
      (requireDemoCapability(this.principal, "administration"), this.store.read((state) => {
        const projection = screenProjectionFor(state, this.principal, screenId);
        const action = projection.visibleActions.find((candidate) => candidate.id === actionId);
        if (!action) {
          throw new BackendInvariantError(`Action ${actionId} is not declared for screen ${screenId}.`);
        }
        return { screenId, actionId, effect: action.effect };
      })),
  };

  private requireMockGovernedCurrent(
    input: DepartmentManagerGovernedReviewCommandInput,
    status: GovernedCandidateView["status"],
    candidate = this.governedCandidate,
  ): void {
    requireNonEmpty(input.operationId, "Operation ID");
    requireNonEmpty(input.idempotencyKey, "Idempotency key");
    requireNonEmpty(input.reason, "Decision reason");
    if (input.candidateId !== candidate.candidateId ||
      input.expectedRevision !== candidate.revision ||
      input.expectedContentDigest !== candidate.contentDigest ||
      candidate.status !== status) {
      throw new BackendConflictError("Stale governed candidate revision, digest, or status.");
    }
  }

  private async mockGovernedReviewDecision(
    input: DepartmentManagerGovernedReviewCommandInput,
    decision: GovernedReviewDecisionView["decision"],
  ): Promise<GovernedCandidateView> {
    requireDemoCapability(this.principal, "governedChecklistReview");
    const assignment = this.mockGovernedAssignmentFor(this.governedCandidate);
    const semantic = await governedCanonicalSHA256({
      command: decision,
      operationId: input.operationId,
      candidateId: input.candidateId,
      expectedRevision: input.expectedRevision,
      expectedContentDigest: input.expectedContentDigest,
      reason: input.reason,
    });
    const replay = this.governedCommands.get(input.operationId) ?? this.governedCommands.get(input.idempotencyKey);
    if (replay) {
      if (replay.operationId !== input.operationId || replay.idempotencyKey !== input.idempotencyKey ||
        replay.semantic !== semantic) {
        throw new BackendConflictError("Governed review command identity was reused with different semantics.");
      }
      return this.projectMockGovernedCandidate(replay.committedCandidate!);
    }
    this.requireMockGovernedCurrent(input, "DEPARTMENT_REVIEW");
    const blockers = this.mockGovernedBlockers(this.governedCandidate);
    if (decision === "TECHNICALLY_APPROVED" && blockers.length > 0) {
      throw new GovernedValidationError(structuredClone(blockers));
    }
    const decisions = this.governedState.decisions.get(input.candidateId) ?? [];
    if (decision === "TECHNICALLY_APPROVED" &&
      decisions.some((item) => item.decision === decision &&
        item.actorDepartmentId === assignment.departmentId &&
        item.actorOrganizationalUnitId === assignment.organizationalUnitId)) {
      throw new BackendConflictError("This exact department owner already approved the candidate.");
    }
    decisions.push({
      decisionId: `DRD-${input.operationId}`,
      decision,
      candidateRootId: this.governedCandidate.candidateRootId,
      candidateId: input.candidateId,
      candidateRevision: input.expectedRevision,
      candidateContentDigest: input.expectedContentDigest,
      actorSubjectId: this.principal.subjectId,
      actorDepartmentMembershipId: assignment.membershipId,
      actorDepartmentId: assignment.departmentId,
      actorOrganizationalUnitId: assignment.organizationalUnitId,
      reason: input.reason,
      decidedAt: this.store.clock(),
      operationId: input.operationId,
      idempotencyKey: input.idempotencyKey,
      semanticPayloadDigest: semantic,
      auditEventId: `AE-${input.operationId}`,
    });
    this.governedState.decisions.set(input.candidateId, decisions);
    const allOwnersApproved = decision === "TECHNICALLY_APPROVED" &&
      this.governedCandidate.requiredOwners
        .filter((owner) => owner.approvalRequired)
        .every((owner) => decisions.some((item) =>
          item.decision === "TECHNICALLY_APPROVED" &&
          item.actorDepartmentId === owner.departmentId &&
          item.actorOrganizationalUnitId === owner.organizationalUnitId));
    this.governedCandidate = {
      ...this.governedCandidate,
      status: decision === "TECHNICALLY_APPROVED" && !allOwnersApproved
        ? "DEPARTMENT_REVIEW"
        : decision,
    };
    this.governedCandidates.set(input.candidateId, this.governedCandidate);
    const occurredAt = this.store.clock();
    const command: MockGovernedCommand = {
      operationId: input.operationId,
      idempotencyKey: input.idempotencyKey,
      semantic,
      candidateId: input.candidateId,
      actorSubjectId: this.principal.subjectId,
      actorDepartmentMembershipId: assignment.membershipId,
      candidateRevision: input.expectedRevision,
      candidateContentDigest: input.expectedContentDigest,
      reason: input.reason,
      occurredAt,
      committedCandidate: structuredClone(this.governedCandidate),
    };
    this.governedCommands.set(input.operationId, command);
    this.governedCommands.set(input.idempotencyKey, command);
    this.store.execute(`GOVERNED-AUDIT-${input.operationId}`, command, (state) => {
      state.auditEvents.push({
        eventId: `AE-${input.operationId}`,
        occurredAt,
        actorRole: "manager",
        actorSubjectId: this.principal.subjectId,
        action: decision === "TECHNICALLY_APPROVED"
          ? "TECHNICAL_APPROVAL_RECORDED"
          : decision,
        entityType: "GOVERNED_CANDIDATE",
        entityId: input.candidateId,
        beforeStatus: "DEPARTMENT_REVIEW",
        afterStatus: this.governedCandidate.status,
        reason: input.reason,
        entityRevision: input.expectedRevision,
      });
      return true;
    });
    return this.projectMockGovernedCandidate(this.governedCandidate);
  }

  readonly governedChecklistReview: DemoBackend["governedChecklistReview"] = {
    validateBlockedGeneration: async (input): Promise<GovernedBlockedGenerationResult> => {
      requireDemoCapability(this.principal, "governedChecklistReview");
      this.mockGovernedAssignments();
      requireNonEmpty(input.operationId, "Operation ID");
      requireNonEmpty(input.idempotencyKey, "Idempotency key");
      const request = input.generationRequest;
      if (governedCanonicalJSON(request) !== governedCanonicalJSON(EXACT_BLOCKED_REAL_OPS_AOC_REQUEST)) {
        throw new GovernedValidationError([{
          fieldPath: "requestId",
          code: "BLOCKED_REQUEST_IDENTITY_MISMATCH",
          message: "The exact source-bound OPS/AOC request identity is required.",
          sourceIdentity: request.sourceSnapshots[0]?.sourceSnapshotId,
          sourceHash: request.sourceSnapshots[0]?.sourceHash,
          clauseId: request.sourceSnapshots[0]?.clauseIds[0],
          locator: request.sourceSnapshots[0]?.clauseLocators[0],
        }]);
      }
      return {
        status: "BLOCKED",
        requestId: request.requestId,
        blockingIssues: structuredClone(request.unresolvedSourceGaps),
        effectCounts: {
          generationRuns: 0,
          candidates: 0,
          reviewDecisions: 0,
          publicationDecisions: 0,
          checklistVersions: 0,
          auditEvents: 0,
        },
      };
    },
    listQueue: async () => {
      requireDemoCapability(this.principal, "governedChecklistReview");
      const assignments = this.mockGovernedAssignments();
      return {
        items: [...this.governedCandidates.values()]
          .filter((persistedCandidate) =>
            ["DEPARTMENT_REVIEW", "RETURNED", "TECHNICALLY_APPROVED"].includes(persistedCandidate.status) &&
            ![...this.governedCandidates.values()].some((successor) =>
              successor.supersedesCandidateId === persistedCandidate.candidateId,
            ) &&
            assignments.some((assignment) => persistedCandidate.requiredOwners.some((owner) =>
              owner.approvalRequired &&
              owner.departmentId === assignment.departmentId &&
              owner.organizationalUnitId === assignment.organizationalUnitId,
            )),
          )
          .sort((left, right) => left.candidateId.localeCompare(right.candidateId))
          .map((persistedCandidate) => ({
            candidate: this.projectMockGovernedCandidate(persistedCandidate),
            requiredOwners: structuredClone(persistedCandidate.requiredOwners),
            decisions: structuredClone(this.governedState.decisions.get(persistedCandidate.candidateId) ?? []),
            blockingIssues: this.mockGovernedBlockers(persistedCandidate),
          })),
      };
    },
    getCandidate: async ({ candidateId }) => {
      requireDemoCapability(this.principal, "governedChecklistReview");
      const candidate = this.governedCandidates.get(candidateId);
      if (!candidate) {
        throw new BackendInvariantError("Governed candidate was not found.");
      }
      this.mockGovernedAssignmentFor(candidate);
      const projected = this.projectMockGovernedCandidate(candidate);
      return {
        candidate: projected,
        requiredOwners: structuredClone(candidate.requiredOwners),
        decisions: structuredClone(this.governedState.decisions.get(candidateId) ?? []),
        blockingIssues: this.mockGovernedBlockers(candidate),
      };
    },
    return: async (input) => this.mockGovernedReviewDecision(input, "RETURNED"),
    reject: async (input) => this.mockGovernedReviewDecision(input, "REJECTED"),
    approve: async (input) => this.mockGovernedReviewDecision(input, "TECHNICALLY_APPROVED"),
    publish: async (input) => {
      requireDemoCapability(this.principal, "governedChecklistReview");
      const candidate = this.governedCandidates.get(input.candidateId);
      if (!candidate) {
        throw new BackendConflictError("Stale governed candidate revision, digest, or status.");
      }
      this.mockGovernedAssignmentFor(candidate);
      const semantic = await governedCanonicalSHA256({
        command: "PUBLISHED",
        operationId: input.operationId,
        candidateId: input.candidateId,
        expectedRevision: input.expectedRevision,
        expectedContentDigest: input.expectedContentDigest,
        reason: input.reason,
      });
      const replay = this.governedCommands.get(input.operationId) ?? this.governedCommands.get(input.idempotencyKey);
      if (replay) {
        if (replay.operationId !== input.operationId || replay.idempotencyKey !== input.idempotencyKey ||
          replay.semantic !== semantic || !this.governedState.publicationDecisions.has(input.candidateId)) {
          throw new BackendConflictError("Governed publication command identity was reused with different semantics.");
        }
        return structuredClone(this.governedState.publicationDecisions.get(input.candidateId)!);
      }
      this.requireMockGovernedCurrent(input, "TECHNICALLY_APPROVED", candidate);
      const blockers = this.mockGovernedBlockers(candidate);
      if (blockers.length > 0) {
        throw new GovernedValidationError(structuredClone(blockers));
      }
      const technicalDecisions = this.governedState.decisions.get(input.candidateId) ?? [];
      if (!candidate.requiredOwners
        .filter((owner) => owner.approvalRequired)
        .every((owner) => technicalDecisions.some((decision) =>
          decision.decision === "TECHNICALLY_APPROVED" &&
          decision.actorDepartmentId === owner.departmentId &&
          decision.actorOrganizationalUnitId === owner.organizationalUnitId))) {
        throw new BackendConflictError("Every exact required owner must technically approve before publication.");
      }
      const recomputedDigest = await governedCandidateContentDigest({
        complianceMappings: candidate.mappings,
        inspectionChecklist: {
          checklistId: candidate.templateId.replace(/^TPL-/, ""),
          questions: candidate.questions,
        },
      });
      if (recomputedDigest !== candidate.contentDigest) {
        throw new GovernedValidationError([governedValidationIssue(
          "contentDigest",
          "CANDIDATE_DIGEST_MISMATCH",
          "Persisted ordered mapping and question snapshots no longer match the approved digest.",
        )]);
      }
      const assignment = this.mockGovernedAssignmentFor(candidate);
      const occurredAt = this.store.clock();
      const publication: GovernedPublicationView = {
        templateVersionId: `CTV-GOV-${semantic.slice("sha256:".length, "sha256:".length + 20)}`,
        publicationDecisionId: `PUBDEC-${input.operationId}`,
        candidateRootId: candidate.candidateRootId,
        candidateId: input.candidateId,
        candidateRevision: input.expectedRevision,
        candidateContentDigest: input.expectedContentDigest,
        actorSubjectId: this.principal.subjectId,
        actorDepartmentMembershipId: assignment.membershipId,
        actorDepartmentId: assignment.departmentId,
        actorOrganizationalUnitId: assignment.organizationalUnitId,
        reason: input.reason,
        decidedAt: occurredAt,
        publishedAt: occurredAt,
        operationId: input.operationId,
        idempotencyKey: input.idempotencyKey,
        semanticPayloadDigest: semantic,
        auditEventId: `AE-${input.operationId}`,
      };
      const publishedCandidate = { ...candidate, status: "PUBLISHED" as const };
      this.governedCandidates.set(input.candidateId, publishedCandidate);
      if (this.governedCandidate.candidateId === input.candidateId) {
        this.governedCandidate = publishedCandidate;
      }
      this.governedState.publicationDecisions.set(input.candidateId, publication);
      this.governedState.publishedVersions.set(publication.templateVersionId, {
        publication: structuredClone(publication),
        mappings: structuredClone(candidate.mappings),
        questions: structuredClone(candidate.questions),
      });
      const command: MockGovernedCommand = {
        operationId: input.operationId,
        idempotencyKey: input.idempotencyKey,
        semantic,
        candidateId: input.candidateId,
        actorSubjectId: this.principal.subjectId,
        actorDepartmentMembershipId: assignment.membershipId,
        candidateRevision: input.expectedRevision,
        candidateContentDigest: input.expectedContentDigest,
        reason: input.reason,
        occurredAt,
      };
      this.governedCommands.set(input.operationId, command);
      this.governedCommands.set(input.idempotencyKey, command);
      this.store.execute(`GOVERNED-AUDIT-${input.operationId}`, command, (state) => {
        state.auditEvents.push({
          eventId: `AE-${input.operationId}`,
          occurredAt,
          actorRole: "manager",
          actorSubjectId: this.principal.subjectId,
          action: "CHECKLIST_PUBLISHED",
          entityType: "GOVERNED_CANDIDATE",
          entityId: input.candidateId,
          beforeStatus: "TECHNICALLY_APPROVED",
          afterStatus: "PUBLISHED",
          reason: input.reason,
          entityRevision: input.expectedRevision,
        });
        return true;
      });
      return structuredClone(publication);
    },
    getPublishedVersion: async ({ templateVersionId }): Promise<GovernedPublishedVersionView> => {
      requireDemoCapability(this.principal, "governedChecklistReview");
      const published = this.governedState.publishedVersions.get(templateVersionId);
      if (!published) {
        throw new BackendInvariantError("Governed published version was not found.");
      }
      this.mockGovernedAssignmentFor(this.governedCandidates.get(
        published.publication.candidateId,
      )!);
      return structuredClone(published);
    },
  };

  readonly governedChecklistIntake: GovernedChecklistIntakeBackend = {
    receiveBatch: async (input: CreateChecklistImportBatchReceiptInput & { archive: Blob | Uint8Array }) => {
      requireDemoCapability(this.principal, "governedChecklistIntake");
      requireRole(this.principal, ["admin"], "Admin authority is required for candidate-only archive intake.");
      void input.archive;
      const existing = this.governedIntakeBatches.get(input.operationId) ?? this.governedIntakeBatches.get(input.idempotencyKey);
      if (existing) return { ...structuredClone(existing), replayed: true };
      const batch: ChecklistImportBatchView = {
        importBatchId: `IMPORT-${input.operationId}`,
        expectedArchiveSha256: input.expectedArchiveSha256,
        status: "RECEIVED",
        manifestDigest: null,
        fileCount: 0,
        registerCount: 0,
        blockingIssues: ["PARSER_PENDING", "CANDIDATE_ONLY"],
      };
      const receipt = { batch, replayed: false };
      this.governedIntakeBatches.set(input.operationId, receipt);
      this.governedIntakeBatches.set(input.idempotencyKey, receipt);
      return structuredClone(receipt);
    },
    getBatch: async ({ importBatchId }) => {
      requireDemoCapability(this.principal, "governedChecklistIntake");
      requireRole(this.principal, ["admin"], "Admin authority is required for candidate-only inventory.");
      const receipt = [...this.governedIntakeBatches.values()].find(({ batch }) => batch.importBatchId === importBatchId);
      if (!receipt) throw new BackendInvariantError("Import batch was not found.");
      return structuredClone(receipt.batch);
    },
    listFiles: async () => {
      requireDemoCapability(this.principal, "governedChecklistIntake");
      requireRole(this.principal, ["admin"], "Admin authority is required for candidate-only inventory.");
      return { items: [], nextCursor: null } as ChecklistImportFilePage;
    },
    listReceipts: async () => {
      requireDemoCapability(this.principal, "governedChecklistIntake");
      requireRole(this.principal, ["admin"], "Admin authority is required for candidate-only receipts.");
      return { items: [], nextCursor: null } as ChecklistImportReceiptPage;
    },
    createExtractionReview: async (_input: CreateChecklistImportFileExtractionReviewInput) => {
      requireDemoCapability(this.principal, "governedChecklistIntake");
      requireRole(this.principal, ["admin"], "Admin authority is required for private extraction review.");
      throw new BackendInvariantError("Synthetic mock has no parser-output object; extraction review remains blocked.");
    },
    getExtractionReview: async ({ importBatchId, importFileId }) => {
      requireDemoCapability(this.principal, "governedChecklistIntake");
      requireRole(this.principal, ["admin"], "Admin authority is required for private extraction review.");
      const page: ChecklistImportExtractionReviewPage = {
        importBatchId,
        importFileId,
        terminalManifestDigest: "",
        parserReceiptId: "",
        parserOutputDigest: "",
        packetId: "",
        packetDigest: "",
        competingIdentities: [],
        currentDecisionSet: { decisionState: "NO_DECISION" },
        proposals: [],
        nextCursor: null,
      };
      return page;
    },
    resolveIdentity: async (_input: ResolveChecklistImportFileIdentityInput) => {
      requireDemoCapability(this.principal, "governedChecklistIntake");
      requireRole(this.principal, ["admin"], "Admin authority is required for identity resolution.");
      throw new BackendInvariantError("Synthetic mock has no conflicting inventory identity to resolve.");
    },
    importCandidate: async (_input: CreateExistingChecklistCandidateInput): Promise<GovernedBackendCommandResult> => {
      requireDemoCapability(this.principal, "governedChecklistIntake");
      requireRole(this.principal, ["admin"], "Admin authority is required for candidate import.");
      throw new BackendInvariantError("Candidate import remains blocked until the private extraction packet is present.");
    },
    listSourceReviewQueue: async () => {
      requireRole(this.principal, ["admin", "manager"], "Scoped source-review authority is required.");
      return { items: [], nextCursor: null } as GovernedSourceReviewQueuePage;
    },
    getSourceReviewItem: async () => {
      requireRole(this.principal, ["admin", "manager"], "Source review item is not in the current assignment scope.");
      throw new BackendAuthorizationInvariantError("Source review item is not in the current assignment scope.");
    },
    listReviewerQueue: async () => {
      requireRole(this.principal, ["manager"], "Current Department Manager assignment is required for reviewer queue.");
      return { items: [], nextCursor: null } as GovernedReviewerQueuePage;
    },
    attestSourceAuthority: async (_input: GovernedSourceAuthorityAttestationInput): Promise<GovernedSourceAuthorityAttestationView> => {
      throw new BackendAuthorizationInvariantError("No synthetic source-owner assignment can attest regulatory authority.");
    },
    getExistingCandidate: async (_input: { existingCandidateId: string }): Promise<ExistingChecklistCandidateView> => {
      requireRole(this.principal, ["admin", "manager"], "Existing candidate is not in the current assignment scope.");
      throw new BackendAuthorizationInvariantError("Existing candidate is not in the current assignment scope.");
    },
    createDraftFromExisting: async (_input: CreateDraftFromExistingChecklistCandidateInput): Promise<GovernedBackendCommandResult> => {
      throw new BackendInvariantError("Candidate Draft creation remains blocked until server-derived owner resolution.");
    },
    createOfficialSourceDraft: async (_input: CreateOfficialSourceChecklistDraftInput): Promise<GovernedBackendCommandResult> => {
      throw new BackendInvariantError("Official-source Draft creation requires persisted accepted source links.");
    },
    getDraft: async (_input: { candidateId: string }) => {
      requireRole(this.principal, ["admin", "manager"], "Governed Draft is not in the current assignment scope.");
      throw new BackendAuthorizationInvariantError("Governed Draft is not in the current assignment scope.");
    },
    createHybridReconciliation: async (_input: CreateHybridReconciledChecklistDraftInput): Promise<GovernedBackendCommandResult> => {
      throw new BackendInvariantError("Hybrid reconciliation remains blocked until accepted source binding.");
    },
    listReviewComments: async (_input: { candidateId: string; cursor?: string; limit?: number }): Promise<GovernedChecklistReviewCommentPage> => {
      requireRole(this.principal, ["manager"], "Scoped reviewer authority is required for internal comments.");
      return { items: [], nextCursor: null };
    },
    createReviewComment: async (_input: GovernedChecklistReviewCommentInput) => {
      throw new BackendAuthorizationInvariantError("Reviewer comment scope is not established in the synthetic profile.");
    },
    attestSourceMapping: async (_input: GovernedSourceMappingAttestationInput) => {
      throw new BackendAuthorizationInvariantError("No reviewed-source-set assignment can attest candidate mapping.");
    },
    evaluateAuditPackageEligibility: async (input: GovernedAuditPackageEligibilityInput): Promise<GovernedAuditPackageEligibilityView> => {
      requireRole(this.principal, ["manager"], "Current Department Manager assignment is required for eligibility evaluation.");
      return { publishedVersionId: input.publishedVersionId, eligible: false, blockerCodes: ["SOURCE_AUTHORITY_OR_MAPPING_REQUIRED"] };
    },
  };

  readonly adminWorkspace: DemoBackend["adminWorkspace"] = {
    listGovernedSources: async () => {
      requireDemoCapability(this.principal, "adminWorkspace"); requireRole(this.principal, ["admin"], "Admin Preview authority is required.");
      return {
        items: this.mockGovernedSourceSnapshots(),
        nextCursor: null,
      };
    },
    activateGovernedSourceCurrentness: async (input) => {
      requireDemoCapability(this.principal, "adminWorkspace"); requireRole(this.principal, ["admin"], "Admin Preview authority is required.");
      return this.activateMockSourceCurrentness(input);
    },
    importGovernedGenerationRun: async (input) => {
      requireDemoCapability(this.principal, "adminWorkspace"); requireRole(this.principal, ["admin"], "Admin Preview authority is required.");
      const semantic = await governedImportSemanticDigest(input.operationId, input.candidateBundle);
      const replay = this.governedCommands.get(input.operationId) ?? this.governedCommands.get(input.idempotencyKey);
      if (replay) {
        if (replay.operationId !== input.operationId || replay.idempotencyKey !== input.idempotencyKey || replay.semantic !== semantic) throw new BackendConflictError("Governed import command identity was reused with different semantics.");
        return this.projectMockGovernedRun(this.governedRuns.get(replay.generationRunId ?? SYNTHETIC_GOVERNED_RUN.generationRunId)!);
      }
      const bundle = [
        SYNTHETIC_GOVERNED_BUNDLE,
        SYNTHETIC_LEGACY_CHECKLIST_CANDIDATE_BUNDLE,
        SYNTHETIC_HYBRID_RECONCILED_BUNDLE,
      ].find((candidateBundle) =>
        governedCanonicalJSON(input.candidateBundle) === governedCanonicalJSON(candidateBundle),
      );
      if (!bundle) {
        const issue = governedValidationIssue("candidateBundle", "CANDIDATE_BUNDLE_MISMATCH", "only the exact synthetic internal test-profile bundle is supported");
        throw new GovernedValidationError([issue]);
      }
      const sourceCurrentnessEvent = this.sourceCurrentnessEventForBundle(bundle);
      let candidate = this.governedCandidates.get(bundle.candidateBundleId);
      if (!candidate) {
        candidate = {
          ...mockGovernedCandidateForBundle(bundle),
          requiredOwners: structuredClone(this.governedCandidate.requiredOwners),
        };
        this.governedCandidates.set(candidate.candidateId, candidate);
        this.governedState.blockers.set(candidate.candidateId, []);
      }
      this.recordMockSourceImpactCandidateBinding(candidate, sourceCurrentnessEvent);
      this.governedCandidate = candidate;
      const run = {
        ...mockGovernedRunForBundle(bundle, candidate),
        candidate,
      };
      this.governedRuns.set(bundle.generationRunId, run);
      const command: MockGovernedCommand = {
        operationId: input.operationId,
        idempotencyKey: input.idempotencyKey,
        semantic,
        candidateId: candidate.candidateId,
        generationRunId: bundle.generationRunId,
      };
      this.governedCommands.set(input.operationId, command);
      this.governedCommands.set(input.idempotencyKey, command);
      return this.projectMockGovernedRun(run);
    },
    getGovernedGenerationRun: async ({ generationRunId }) => {
      requireDemoCapability(this.principal, "adminWorkspace"); requireRole(this.principal, ["admin"], "Admin Preview authority is required.");
      const run = this.governedRuns.get(generationRunId);
      if (!run) throw new BackendInvariantError("Governed generation run was not found.");
      return this.projectMockGovernedRun(run);
    },
    getGovernedCandidate: async ({ candidateId }) => {
      requireDemoCapability(this.principal, "adminWorkspace"); requireRole(this.principal, ["admin"], "Admin Preview authority is required.");
      const candidate = this.governedCandidates.get(candidateId);
      if (!candidate) throw new BackendInvariantError("Governed candidate was not found.");
      return this.projectMockGovernedCandidate(candidate);
    },
    createGovernedCandidateRevision: async (input) => {
      requireDemoCapability(this.principal, "adminWorkspace"); requireRole(this.principal, ["admin"], "Admin Preview authority is required.");
      const semantic = await governedEditSemanticDigest(input);
      const replay = this.governedCommands.get(input.operationId) ?? this.governedCommands.get(input.idempotencyKey);
      if (replay) {
        if (replay.operationId !== input.operationId || replay.idempotencyKey !== input.idempotencyKey || replay.semantic !== semantic) throw new BackendConflictError("Governed edit command identity was reused with different semantics.");
        return this.projectMockGovernedCandidate(this.governedCandidates.get(replay.candidateId)!);
      }
      if (input.candidateId !== this.governedCandidate.candidateId || input.expectedRevision !== this.governedCandidate.revision || input.expectedContentDigest !== this.governedCandidate.contentDigest || this.governedCandidate.status !== "GENERATED_DRAFT") throw new BackendConflictError("Stale governed candidate revision or digest.");
      validateMockGovernedEdit(this.governedCandidate, input.mappings, input.questions, input.requiredOwners);
      const digest = await governedCandidateContentDigest({ complianceMappings: input.mappings, inspectionChecklist: { checklistId: this.governedCandidate.templateId, questions: input.questions } });
      const candidateId = `CAND-EDIT-${semantic.slice(7, 27)}`;
      const successor: GovernedCandidateView = {
        ...structuredClone(this.governedCandidate), candidateId,
        supersedesCandidateId: this.governedCandidate.candidateId,
        version: this.governedCandidate.version + 1, revision: this.governedCandidate.revision + 1,
        contentDigest: digest, changeReason: input.changeReason,
        mappings: structuredClone(input.mappings), questions: structuredClone(input.questions),
        requiredOwners: structuredClone(input.requiredOwners),
      };
      this.governedCandidate = successor;
      this.governedCandidates.set(candidateId, successor);
      const generationRunId = successor.generationRunId;
      if (!generationRunId) throw new BackendInvariantError("Governed successor lost its generation-run lineage.");
      const run = this.governedRuns.get(generationRunId)!;
      this.governedRuns.set(generationRunId, { ...run, candidate: successor });
      const command = { operationId: input.operationId, idempotencyKey: input.idempotencyKey, semantic, candidateId };
      this.governedCommands.set(input.operationId, command);
      this.governedCommands.set(input.idempotencyKey, command);
      return this.projectMockGovernedCandidate(successor);
    },
    submitGovernedCandidateReview: async (input) => {
      requireDemoCapability(this.principal, "adminWorkspace"); requireRole(this.principal, ["admin"], "Admin Preview authority is required.");
      const semantic = await governedSubmitSemanticDigest(input);
      const replay = this.governedCommands.get(input.operationId) ?? this.governedCommands.get(input.idempotencyKey);
      if (replay) {
        if (replay.operationId !== input.operationId || replay.idempotencyKey !== input.idempotencyKey || replay.semantic !== semantic) throw new BackendConflictError("Governed submission command identity was reused with different semantics.");
        return this.projectMockGovernedCandidate(this.governedCandidates.get(replay.candidateId)!);
      }
      if (input.candidateId !== this.governedCandidate.candidateId || input.expectedRevision !== this.governedCandidate.revision || input.expectedContentDigest !== this.governedCandidate.contentDigest || this.governedCandidate.status !== "GENERATED_DRAFT") throw new BackendConflictError("Stale governed candidate revision or digest.");
      const blockers = this.mockGovernedBlockers(this.governedCandidate);
      if (blockers.length > 0) {
        throw new GovernedValidationError(structuredClone(blockers));
      }
      this.governedCandidate = { ...this.governedCandidate, status: "DEPARTMENT_REVIEW" };
      this.governedCandidates.set(this.governedCandidate.candidateId, this.governedCandidate);
      const generationRunId = this.governedCandidate.generationRunId;
      if (!generationRunId) throw new BackendInvariantError("Governed candidate lost its generation-run lineage.");
      const run = this.governedRuns.get(generationRunId)!;
      this.governedRuns.set(generationRunId, { ...run, candidate: this.governedCandidate });
      const command = { operationId: input.operationId, idempotencyKey: input.idempotencyKey, semantic, candidateId: this.governedCandidate.candidateId };
      this.governedCommands.set(input.operationId, command);
      this.governedCommands.set(input.idempotencyKey, command);
      return this.projectMockGovernedCandidate(this.governedCandidate);
    },
    listRegulatoryReferences: async ({ search = "", status = "" }) => {
      requireDemoCapability(this.principal, "adminWorkspace");
      requireRole(this.principal, ["admin"], "Admin Preview authority is required for Administration workspace data.");
      return this.store.read((state) => {
        const needle = search.trim().toLocaleLowerCase();
        return {
          items: state.adminWorkspace.regulatoryReferences.filter((reference) =>
            (!needle || `${reference.id} ${reference.title} ${reference.version}`.toLocaleLowerCase().includes(needle)) &&
            (!status || reference.status === status),
          ),
          nextCursor: null,
        };
      });
    },
    listTemplateMasters: async () => {
      requireDemoCapability(this.principal, "adminWorkspace");
      requireRole(this.principal, ["admin"], "Admin Preview authority is required for Administration workspace data.");
      return this.store.read((state) => ({ items: state.adminWorkspace.templateMasters, nextCursor: null }));
    },
    listQuestions: async ({ search = "" }) => {
      requireDemoCapability(this.principal, "adminWorkspace");
      requireRole(this.principal, ["admin"], "Admin Preview authority is required for Administration workspace data.");
      return this.store.read((state) => {
        const needle = search.trim().toLocaleLowerCase();
        return {
          items: Object.values(state.adminWorkspace.questions).filter((question) =>
            !needle || `${question.id} ${question.prompt} ${question.configuredReference} ${question.expectedEvidence}`.toLocaleLowerCase().includes(needle),
          ),
          nextCursor: null,
        };
      });
    },
    createQuestion: async (input) => {
      requireDemoCapability(this.principal, "adminWorkspace");
      requireRole(this.principal, ["admin"], "Admin Preview authority is required to create demo questions.");
      const prompt = requireNonEmpty(input.prompt, "Question text");
      if (prompt.length > 500) throw new BackendInvariantError("Question text must be 500 characters or fewer.");
      const configuredReference = requireNonEmpty(input.configuredReference, "Configured reference");
      const expectedEvidence = requireNonEmpty(input.expectedEvidence, "Expected Evidence");
      if (input.expectedRevision !== null) throw new BackendConflictError(`Question collection revision conflict: expected null, received ${input.expectedRevision}.`);
      requireNonEmpty(input.idempotencyKey, "Idempotency key");
      const normalizedInput = { ...input, prompt, configuredReference, expectedEvidence };
      return this.store.execute(input.idempotencyKey, normalizedInput, (state) => {
        const id = `Q-ADMIN-2026-${pad(++state.adminWorkspace.questionCounter)}`;
        const question = { id, prompt, configuredReference, expectedEvidence, revision: 1 };
        state.adminWorkspace.questions[id] = question;
        state.auditEvents.push({
          eventId: `AUDIT-ADMIN-${pad(state.counters.auditEvent++)}`,
          occurredAt: this.store.clock(),
          actorRole: "admin",
          actorSubjectId: this.principal.subjectId,
          action: "admin.question_created",
          entityType: "checklist_question",
          entityId: id,
          beforeStatus: null,
          afterStatus: "DRAFT",
          reason: "Created browser-local demo Question Bank record.",
          entityRevision: 1,
        });
        return question;
      });
    },
    getTemplate: async ({ templateId }) => {
      requireDemoCapability(this.principal, "adminWorkspace");
      requireRole(this.principal, ["admin"], "Admin Preview authority is required for Administration workspace data.");
      return this.store.read((state) => {
        if (templateId !== state.adminWorkspace.template.id) throw new BackendInvariantError(`Template ${templateId} was not found.`);
        return state.adminWorkspace.template;
      });
    },
    createDraft: async (input) => {
      requireDemoCapability(this.principal, "adminWorkspace");
      requireRole(this.principal, ["admin"], "Admin Preview authority is required to create a working checklist Draft.");
      requireNonEmpty(input.idempotencyKey, "Idempotency key");
      const changeReason = requireNonEmpty(input.changeReason, "Change reason");
      return this.store.execute(input.idempotencyKey, input, (state) => {
        const template = state.adminWorkspace.template;
        if (input.templateId !== template.id) throw new BackendInvariantError(`Template ${input.templateId} was not found.`);
        requireRevision(template.revision, input.expectedRevision, "Template master");
        if (template.versions.some((version) => version.status === "DRAFT")) throw new BackendConflictError(`${template.id} already has a working Draft version.`);
        const published = template.versions.find((version) => version.id === template.publishedVersionId);
        if (!published || published.status !== "PUBLISHED") throw new BackendInvariantError(`${template.publishedVersionId} is not the immutable published version.`);
        const versionNumber = Math.max(...template.versions.map((version) => version.version)) + 1;
        const draft = {
          id: `CTV-CABIN-DRAFT-${versionNumber}`,
          templateId: template.id,
          version: versionNumber,
          status: "DRAFT" as const,
          owner: "Admin Preview" as const,
          creatorSubjectId: this.principal.subjectId,
          changeReason,
          questionIds: [...published.questionIds],
          revision: 1,
          createdAt: this.store.clock(),
        };
        template.versions.push(draft);
        template.revision += 1;
        state.auditEvents.push({
          eventId: `AUDIT-ADMIN-${pad(state.counters.auditEvent++)}`,
          occurredAt: this.store.clock(), actorRole: "admin", actorSubjectId: this.principal.subjectId,
          action: "admin.template_draft_created", entityType: "checklist_template_version", entityId: draft.id,
          beforeStatus: null, afterStatus: "DRAFT", reason: changeReason, entityRevision: draft.revision,
        });
        return draft;
      });
    },
    addDraftQuestion: async (input) => {
      requireDemoCapability(this.principal, "adminWorkspace");
      requireRole(this.principal, ["admin"], "Admin Preview authority is required to configure a working checklist Draft.");
      requireNonEmpty(input.idempotencyKey, "Idempotency key");
      return this.store.execute(input.idempotencyKey, input, (state) => {
        const template = state.adminWorkspace.template;
        if (input.templateId !== template.id) throw new BackendInvariantError(`Template ${input.templateId} was not found.`);
        const draft = template.versions.find((version) => version.id === input.draftVersionId);
        if (!draft || draft.status !== "DRAFT") throw new BackendInvariantError(`${input.draftVersionId} is not an editable Draft.`);
        requireRevision(draft.revision, input.expectedRevision, "Checklist Draft");
        if (!state.adminWorkspace.questions[input.questionId]) throw new BackendInvariantError(`Question ${input.questionId} was not found.`);
        if (draft.questionIds.includes(input.questionId)) throw new BackendConflictError(`${input.questionId} is already in ${draft.id}.`);
        draft.questionIds.push(input.questionId);
        draft.revision += 1;
        state.auditEvents.push({
          eventId: `AUDIT-ADMIN-${pad(state.counters.auditEvent++)}`, occurredAt: this.store.clock(), actorRole: "admin", actorSubjectId: this.principal.subjectId,
          action: "admin.template_question_added", entityType: "checklist_template_version", entityId: draft.id,
          beforeStatus: "DRAFT", afterStatus: "DRAFT", reason: `Added exact question ${input.questionId}.`, entityRevision: draft.revision,
        });
        return draft;
      });
    },
    moveDraftQuestion: async (input) => {
      requireDemoCapability(this.principal, "adminWorkspace");
      requireRole(this.principal, ["admin"], "Admin Preview authority is required to configure a working checklist Draft.");
      requireNonEmpty(input.idempotencyKey, "Idempotency key");
      return this.store.execute(input.idempotencyKey, input, (state) => {
        const template = state.adminWorkspace.template;
        if (input.templateId !== template.id) throw new BackendInvariantError(`Template ${input.templateId} was not found.`);
        const draft = template.versions.find((version) => version.id === input.draftVersionId);
        if (!draft || draft.status !== "DRAFT") throw new BackendInvariantError(`${input.draftVersionId} is not an editable Draft.`);
        requireRevision(draft.revision, input.expectedRevision, "Checklist Draft");
        const index = draft.questionIds.indexOf(input.questionId);
        if (index < 0) throw new BackendInvariantError(`Question ${input.questionId} is not in ${draft.id}.`);
        const target = input.direction === "UP" ? index - 1 : index + 1;
        if (target < 0 || target >= draft.questionIds.length) throw new BackendInvariantError(`${input.questionId} cannot move ${input.direction.toLocaleLowerCase()} in ${draft.id}.`);
        [draft.questionIds[index], draft.questionIds[target]] = [draft.questionIds[target]!, draft.questionIds[index]!];
        draft.revision += 1;
        state.auditEvents.push({
          eventId: `AUDIT-ADMIN-${pad(state.counters.auditEvent++)}`, occurredAt: this.store.clock(), actorRole: "admin", actorSubjectId: this.principal.subjectId,
          action: "admin.template_question_reordered", entityType: "checklist_template_version", entityId: draft.id,
          beforeStatus: "DRAFT", afterStatus: "DRAFT", reason: `Moved exact question ${input.questionId} ${input.direction}.`, entityRevision: draft.revision,
        });
        return draft;
      });
    },
    getInspectionPackage: async ({ packageId }) => {
      requireDemoCapability(this.principal, "adminWorkspace");
      requireRole(this.principal, ["admin"], "Admin Preview authority is required for Administration workspace data.");
      return this.store.read((state) => {
        if (packageId !== state.adminWorkspace.inspectionPackage.id) throw new BackendInvariantError(`Admin inspection package ${packageId} was not found.`);
        return state.adminWorkspace.inspectionPackage;
      });
    },
    listReportDefinitions: async ({ search = "" }) => {
      requireDemoCapability(this.principal, "adminWorkspace");
      requireRole(this.principal, ["admin"], "Admin Preview authority is required for Administration workspace data.");
      return this.store.read((state) => {
        const needle = search.trim().toLocaleLowerCase();
        return { items: state.adminWorkspace.reportDefinitions.filter((report) => !needle || `${report.id} ${report.title} ${report.description}`.toLocaleLowerCase().includes(needle)), nextCursor: null };
      });
    },
    listAccessDirectory: async ({ search = "", role = "" }) => {
      requireDemoCapability(this.principal, "adminWorkspace");
      requireRole(this.principal, ["admin"], "Admin Preview authority is required for Administration workspace data.");
      return this.store.read((state) => {
        const needle = search.trim().toLocaleLowerCase();
        return {
          items: Object.values(state.profiles)
            .filter((profile) => (!needle || `${profile.subjectId} ${profile.displayName} ${profile.organizationId ?? "CAA"}`.toLocaleLowerCase().includes(needle)) && (!role || profile.role === role))
            .map((profile) => ({
              subjectId: profile.subjectId,
              displayName: profile.displayName,
              roles: [profile.role],
              organizationId: profile.organizationId,
              email: "Not configured in demo" as const,
              mfaEnrolled: false,
              mfaState: "Not configured in demo",
              requiredActions: [],
              invitationState: "Not configured in demo",
              accountStatus: "Not configured in demo" as const,
              applicationProfileState: "linked",
              membershipId: null,
              membershipState: "Not configured in demo",
              membershipRevision: 0,
              membershipDrift: "Not configured in demo",
              lastSuccessfulSessionAt: null,
              providerObservedAt: "",
            })),
          nextCursor: null,
        };
      });
    },
    requestUserLifecycle: async () => {
      requireDemoCapability(this.principal, "adminWorkspace");
      requireRole(this.principal, ["admin"], "Admin Preview authority is required for Administration workspace data.");
      throw new BackendInvariantError(
        "Keycloak user lifecycle commands are unavailable in the browser-local demo profile.",
      );
    },
    getUserLifecycleRequest: async () => {
      requireDemoCapability(this.principal, "adminWorkspace");
      requireRole(this.principal, ["admin"], "Admin Preview authority is required for Administration workspace data.");
      throw new BackendInvariantError(
        "Keycloak user lifecycle status is unavailable in the browser-local demo profile.",
      );
    },
    listOrganizations: async ({ search = "", organizationType = "", status = "", scope = "" }) => {
      requireDemoCapability(this.principal, "adminWorkspace");
      requireRole(this.principal, ["admin"], "Admin Preview authority is required for Administration workspace data.");
      return this.store.read((state) => {
        const needle = search.trim().toLocaleLowerCase();
        const items = state.organizations.map((organization) => ({
          id: organization.id,
          legalName: organization.legalName,
          organizationType: organization.organizationType,
          status: organization.status,
          scope: "CAA oversight" as const,
          detailAvailable: organization.id === "ORG-FLY-NAMIBIA",
          disabledReason: organization.id === "ORG-FLY-NAMIBIA" ? null : `${organization.id} has no declared contextual detail route in Task 10.`,
        })).filter((organization) =>
          (!needle || `${organization.id} ${organization.legalName}`.toLocaleLowerCase().includes(needle)) &&
          (!organizationType || organization.organizationType === organizationType) &&
          (!status || organization.status === status) &&
          (!scope || organization.scope === scope),
        );
        return { items, nextCursor: null };
      });
    },
    getOrganization: async ({ organizationId }) => {
      requireDemoCapability(this.principal, "adminWorkspace");
      requireRole(this.principal, ["admin"], "Admin Preview authority is required for Administration workspace data.");
      return this.store.read((state) => {
        if (organizationId !== "ORG-FLY-NAMIBIA") throw new BackendAuthorizationInvariantError(`${organizationId} has no declared contextual Admin detail route.`);
        const organization = state.organizations.find((candidate) => candidate.id === organizationId);
        if (!organization) throw new BackendInvariantError(`Organization ${organizationId} was not found.`);
        return { id: organization.id, legalName: organization.legalName, organizationType: organization.organizationType, status: organization.status, scope: "CAA oversight" as const, detailAvailable: true, disabledReason: null };
      });
    },
    listAuditEvents: async ({ actor = "", action = "", entity = "", system = "", dateText = "" }) => {
      requireDemoCapability(this.principal, "adminWorkspace");
      requireRole(this.principal, ["admin"], "Admin Preview authority is required for Administration workspace data.");
      return this.store.read((state) => ({
        items: state.auditEvents.filter((event) =>
          (!actor || `${event.actorSubjectId ?? ""} ${event.actorRole ?? "SYSTEM"}`.toLocaleLowerCase().includes(actor.toLocaleLowerCase())) &&
          (!action || event.action.toLocaleLowerCase().includes(action.toLocaleLowerCase())) &&
          (!entity || `${event.entityType} ${event.entityId}`.toLocaleLowerCase().includes(entity.toLocaleLowerCase())) &&
          (!system || ((!event.actorRole && !event.actorSubjectId) ? "SYSTEM" : "MANUAL") === system.toLocaleUpperCase()) &&
          (!dateText || event.occurredAt.includes(dateText)),
        ),
        nextCursor: null,
      }));
    },
  };

  readonly assistantDrafts: DemoBackend["assistantDrafts"] = {
    getGuidance: async () => {
      requireDemoCapability(this.principal, "assistantDrafts");
      requireRole(this.principal, ["inspector", "leadInspector"], "Inspector advisory authority is required.");
      return { advisoryOnly: true, prohibitedActions: ["create Finding", "set severity", "close Finding", "enforcement action"] };
    },
    createDraft: async (input) => this.store.read((state) => {
      requireDemoCapability(this.principal, "assistantDrafts");
      requireRole(this.principal, ["inspector", "leadInspector"], "Inspector advisory authority is required.");
      const finding = findingForPrincipal(state, this.principal, input.findingId);
      const prompt = requireNonEmpty(input.prompt, "Assistant prompt");
      return {
        id: `DRAFT-${finding.id}`,
        findingId: finding.id,
        prompt,
        draft: `Advisory draft for ${finding.findingNumber}: review the configured finding basis and request only the expected evidence.`,
        advisoryOnly: true,
        canCreateFinding: false,
        canSetSeverity: false,
        canCloseFinding: false,
      };
    }),
  };

  readonly planningIntake: DemoBackend["planningIntake"] = {
    createDraft: async (input) => {
      requireDemoCapability(this.principal, "planningIntake");
      requireRole(this.principal, ["manager"], "Department Manager authority is required to create Planning intake drafts.");
      requireNonEmpty(input.operationId, "Operation id");
      requireNonEmpty(input.idempotencyKey, "Idempotency key");
      return this.store.execute(input.operationId, input, (state) => {
        const organization = state.organizations.find((item) => item.id === input.values.organizationId);
        if (!organization) throw new BackendInvariantError(`Organization ${input.values.organizationId} was not found.`);
        const draftId = input.draftId ?? stableMockCommandKey("PLAN-DRAFT", [input.operationId, input.values.organizationId]);
        const existing = state.planningIntakeDrafts[draftId];
        if (existing) throw new BackendConflictError(`Planning intake draft ${draftId} already exists.`);
        const draft: PlanningIntakeDraftView = {
          id: draftId,
          organizationId: organization.id,
          organizationName: organization.legalName,
          applicationType: input.values.applicationType ?? "",
          domain: input.values.domain ?? "",
          inspectionCategory: input.values.inspectionCategory ?? "Routine / Announced",
          noticePolicy: input.values.noticePolicy ?? "ADVANCE",
          purpose: input.values.purpose ?? "",
          triggerType: input.values.triggerType ?? "Department Manager initiated",
          riskCategory: input.values.riskCategory ?? "",
          plannedDate: input.values.plannedDate ?? "",
          mode: input.values.mode ?? "On-site",
          location: input.values.location ?? "",
          templateVersionId: input.values.templateVersionId,
          scope: input.values.scope,
          catalogVersion: input.values.catalogVersion,
          scopeDraftId: input.values.scopeDraftId,
          selectionDigest: input.values.selectionDigest,
          selectedQuestionVersionIds: input.values.selectedQuestionVersionIds ? [...input.values.selectedQuestionVersionIds] : [],
          estimatedResourceRequirement: input.values.estimatedResourceRequirement,
          providerScopeId: input.values.providerScopeId,
          regulatedTargetId: input.values.regulatedTargetId,
          requestedBudget: input.values.requestedBudget ?? 0,
          currency: input.values.currency ?? "USD",
          revision: 1,
          submittedPlanningItemId: null,
          updatedAt: this.store.clock(),
        };
        state.planningIntakeDrafts[draftId] = draft;
        return draft;
      });
    },
    getDraft: async ({ draftId }) => {
      requireDemoCapability(this.principal, "planningIntake");
      requireRole(this.principal, ["manager"], "Department Manager authority is required for Planning intake drafts.");
      return this.store.read((state) => {
        const draft = state.planningIntakeDrafts[draftId];
        if (!draft) throw new BackendInvariantError(`Planning intake draft ${draftId} was not found.`);
        return draft;
      });
    },
    saveDraft: async (input) => {
      requireDemoCapability(this.principal, "planningIntake");
      requireRole(this.principal, ["manager"], "Department Manager authority is required for Planning intake drafts.");
      requireNonEmpty(input.idempotencyKey, "Idempotency key");
      return this.store.execute(input.idempotencyKey, input, (state) => {
        const draft = state.planningIntakeDrafts[input.draftId];
        if (!draft) throw new BackendInvariantError(`Planning intake draft ${input.draftId} was not found.`);
        requireRevision(draft.revision, input.expectedRevision, "Planning intake draft");
        if (draft.submittedPlanningItemId) {
          throw new BackendConflictError(`Planning intake draft ${draft.id} is already submitted.`);
        }
        const organization = state.organizations.find((item) => item.id === input.values.organizationId);
        if (!organization) throw new BackendInvariantError(`Organization ${input.values.organizationId} was not found.`);
        const noticePolicy: PlanningIntakeDraftView["noticePolicy"] = input.values.inspectionCategory === "Ad Hoc / Unannounced" ? "WITHHELD" : "ADVANCE";
        const saved = {
          ...draft,
          ...input.values,
          organizationName: organization.legalName,
          noticePolicy,
          revision: draft.revision + 1,
          updatedAt: this.store.clock(),
        };
        state.planningIntakeDrafts[input.draftId] = saved;
        return saved;
      });
    },
    submit: async (input) => {
      requireDemoCapability(this.principal, "planningIntake");
      requireRole(this.principal, ["manager"], "Department Manager authority is required to submit a Planning intake.");
      requireNonEmpty(input.idempotencyKey, "Idempotency key");
      return this.store.execute(input.idempotencyKey, input, (state) => {
        const draft = state.planningIntakeDrafts[input.draftId];
        if (!draft) throw new BackendInvariantError(`Planning intake draft ${input.draftId} was not found.`);
        requireRevision(draft.revision, input.expectedRevision, "Planning intake draft");
        if (draft.submittedPlanningItemId) {
          throw new BackendConflictError(`Planning intake draft ${draft.id} is already submitted.`);
        }
        requireNonEmpty(draft.purpose, "Planning intake purpose");
        requireNonEmpty(draft.location, "Planning intake location");
        if (!Number.isFinite(draft.requestedBudget) || draft.requestedBudget < 0) {
          throw new BackendInvariantError("Requested budget must be zero or greater.");
        }
        const planningItemId = input.planningItemId ?? stableMockCommandKey("PLAN-INTAKE", [input.idempotencyKey, input.draftId]);
        if (state.planningItems[planningItemId]) {
          throw new BackendConflictError(`Planning item ${planningItemId} already exists.`);
        }
        const planningItem = {
          id: planningItemId,
          title: `${draft.inspectionCategory} — ${draft.organizationName}`,
          planYear: Number(draft.plannedDate.slice(0, 4)),
          organizationId: draft.organizationId,
          organizationName: draft.organizationName,
          inspectionType: `${draft.applicationType} · ${draft.domain}`,
          scheduledDate: draft.plannedDate,
          estimatedBudget: draft.requestedBudget,
          status: "FINANCE_REVIEW" as const,
          currentOwnerRole: "finance" as const,
          nextAction: "Finance to review budget and resources",
          revision: 1,
        };
        state.planningItems[planningItem.id] = planningItem;
        draft.submittedPlanningItemId = planningItem.id;
        draft.revision += 1;
        draft.updatedAt = this.store.clock();
        state.auditEvents.push({
          eventId: `AUDIT-PLAN-${pad(state.counters.auditEvent++)}`,
          occurredAt: this.store.clock(),
          actorRole: this.principal.role,
          actorSubjectId: this.principal.subjectId,
          action: "planning.intake_submitted",
          entityType: "SURVEILLANCE_PLAN",
          entityId: planningItem.id,
          beforeStatus: "DRAFT",
          afterStatus: planningItem.status,
          reason: `${draft.inspectionCategory}; notice ${draft.noticePolicy.toLowerCase()}`,
          entityRevision: planningItem.revision,
        });
        return { draft, planningItem };
      });
    },
  };

  /** Canonical catalog/review boundary used by the New Audit and Checklist
   * Management workbenches. The fixture text is intentionally invented and
   * never represents the sealed AGA source package. */
  readonly canonicalQuestionReview: CanonicalQuestionReviewBackend = {
    listCatalog: async (input) => {
      requireDemoCapability(this.principal, "canonicalQuestionReview");
      requireRole(this.principal, ["manager", "admin"], "Canonical catalog access is restricted to CAA managers.");
      const limit = Math.min(25, Math.max(1, input.limit ?? 25));
      const offset = Number(input.cursor ?? 0) || 0;
      const selectedIds = input.scopeId ? new Set(this.canonicalSelections.get(input.scopeId)?.ids ?? []) : null;
      const rows = this.syntheticCanonicalRows(input.usageClass, input.catalogVersion).filter((row) => {
        const needle = input.search?.trim().toLocaleLowerCase() ?? "";
        const selected = selectedIds?.has(row.questionVersionId) ?? false;
        return (!needle || `${row.formCode} ${row.proposalId} ${row.questionVersionId}`.toLocaleLowerCase().includes(needle))
          && (!input.formCode || row.formCode.includes(input.formCode))
          && (!input.domain || row.proposedDomain === input.domain)
          && (!input.topic || row.proposedTopic === input.topic)
          && (!input.riskBand || row.proposedRiskBand === input.riskBand)
          && (!input.sourceGapState || row.sourceGapState === input.sourceGapState)
          && (!input.selected || input.selected === "all" || (input.selected === "selected" ? selected : !selected));
      });
      const page = rows.slice(offset, offset + limit);
      return { items: page.map((row) => ({ ...row, prompt: null, configuredReference: null, expectedEvidence: null })), nextCursor: offset + limit < rows.length ? String(offset + limit) : null, catalogVersion: input.catalogVersion, usageClass: input.usageClass, totalCount: rows.length };
    },
    getQuestion: async (input) => {
      requireDemoCapability(this.principal, "canonicalQuestionReview");
      requireRole(this.principal, ["manager", "admin"], "Canonical catalog access is restricted to CAA managers.");
      const row = this.syntheticCanonicalRows(input.usageClass, input.catalogVersion).find((candidate) => candidate.questionVersionId === input.questionVersionId);
      if (!row) throw new BackendInvariantError(`Question version ${input.questionVersionId} was not found.`);
      return { ...row, prompt: `Synthetic privacy-safe question ${row.formCode} item ${row.ordinal}.`, configuredReference: "Synthetic controlled reference", expectedEvidence: "Synthetic evidence record" };
    },
    previewSelection: async (input) => {
      requireDemoCapability(this.principal, "canonicalQuestionReview");
      requireRole(this.principal, ["manager"], "Department Manager authority is required for scope selection.");
      const state = this.canonicalSelections.get(input.scopeId) ?? { digest: await governedCanonicalSHA256([]), ids: [] };
      const ids = [...new Set(input.questionVersionIds)].sort();
      if (ids.length === 0 || ids.length > 500 || ids.length !== input.questionVersionIds.length) throw new BackendInvariantError("Selection must contain 1–500 unique question versions.");
      if (input.expectedSelectionDigest && input.expectedSelectionDigest !== state.digest) throw new BackendConflictError("Selection digest is stale.");
      return { preview: { selectionDigest: await governedCanonicalSHA256(ids), selectedQuestionVersionIds: ids, selectedCount: ids.length, catalogVersion: "aga-preprod@1.0.0", usageClass: input.usageClass }, affectedCount: ids.length, valid: true, reason: "Selection is ready to commit." };
    },
    commitSelection: async (input) => {
      requireDemoCapability(this.principal, "canonicalQuestionReview");
      requireRole(this.principal, ["manager"], "Department Manager authority is required for scope selection.");
      requireNonEmpty(input.operationId, "Selection operation");
      const state = this.canonicalSelections.get(input.scopeId) ?? { digest: await governedCanonicalSHA256([]), ids: [] };
      const ids = [...new Set(input.questionVersionIds)].sort();
      if (ids.length === 0 || ids.length > 500 || ids.length !== input.questionVersionIds.length) throw new BackendInvariantError("Selection must contain 1–500 unique question versions.");
      const prior = this.canonicalReviewOperations.get(`selection:${input.operationId}`);
      const digest = await governedCanonicalSHA256(ids);
      if (prior) {
        if (prior !== digest) throw new OperationIdReuseError(input.operationId);
        return { operationId: input.operationId, replayed: true, selection: { selectionDigest: digest, selectedQuestionVersionIds: ids, selectedCount: ids.length, catalogVersion: "aga-preprod@1.0.0", usageClass: input.usageClass } };
      }
      const expected = input.expectedSelectionDigest || await governedCanonicalSHA256([]);
      if (expected !== state.digest) throw new BackendConflictError("Selection digest is stale.");
      this.canonicalSelections.set(input.scopeId, { digest, ids });
      this.canonicalReviewOperations.set(`selection:${input.operationId}`, digest);
      return { operationId: input.operationId, replayed: false, selection: { selectionDigest: digest, selectedQuestionVersionIds: ids, selectedCount: ids.length, catalogVersion: "aga-preprod@1.0.0", usageClass: input.usageClass } };
    },
    reviewQueue: async (input) => {
      requireDemoCapability(this.principal, "canonicalQuestionReview");
      requireRole(this.principal, ["manager", "admin"], "Department Manager authority is required for Question Review.");
      const page = await this.canonicalQuestionReview.listCatalog({ catalogVersion: input.catalogVersion, usageClass: input.mode, search: input.search, formCode: input.formCode, domain: input.domain, topic: input.topic, riskBand: input.riskBand, sourceGapState: input.sourceGapState, selected: input.selected, scopeId: input.scopeId, cursor: input.cursor, limit: input.limit });
      return { mode: input.mode, items: page.items, nextCursor: page.nextCursor, totalCount: page.totalCount, capabilities: { canTechnicalApprove: false, canPublish: false, disabledReason: input.mode === "PREPROD_EXERCISE" ? "PREPROD_EXERCISE review cannot invoke technical approval or publication." : "Governed technical approval and publication remain on the candidate authority route." } };
    },
    command: async (input: CanonicalQuestionReviewCommandInput) => {
      requireDemoCapability(this.principal, "canonicalQuestionReview");
      requireRole(this.principal, ["manager"], "Department Manager authority is required for Question Review.");
      if (input.mode === "PREPROD_EXERCISE" && (input.action === "TECHNICAL_APPROVE" || input.action === "PUBLISH")) throw new BackendAuthorizationInvariantError("Exercise review cannot invoke technical approval or publication.");
      requireNonEmpty(input.reason, "Controlled reason");
      const prior = this.canonicalReviewOperations.get(`review:${input.operationId}`);
      if (prior) return { operationId: input.operationId, mode: input.mode, questionVersionId: input.questionVersionId, action: input.action, replayed: true, canPublish: input.mode === "GOVERNED_OPERATIONAL" };
      this.canonicalReviewOperations.set(`review:${input.operationId}`, await governedCanonicalSHA256({ mode: input.mode, questionVersionId: input.questionVersionId, action: input.action, reason: input.reason, domain: input.domain, topic: input.topic }));
      return { operationId: input.operationId, mode: input.mode, questionVersionId: input.questionVersionId, action: input.action, replayed: false, canPublish: input.mode === "GOVERNED_OPERATIONAL" };
    },
  };

  private syntheticCanonicalRows(usageClass: CanonicalQuestionUsageClass, catalogVersion: string): CanonicalQuestionCatalogEntry[] {
    return Array.from({ length: 1310 }, (_, index) => {
      const ordinal = (index % 25) + 1;
      const form = String(index < 1275 ? Math.floor(index / 25) + 1 : 52).padStart(3, "0");
      return {
        catalogVersion,
        usageClass,
        questionVersionId: `qv:synthetic:${catalogVersion}:${form}:${ordinal}`,
        formCode: `SYNTH-FORM-${form}`,
        proposalId: `SYNTH-PROPOSAL-${String(index + 1).padStart(4, "0")}`,
        ordinal,
        questionDigest: `sha256:${String(index + 1).padStart(64, "0")}`,
        prompt: `Synthetic privacy-safe question SYNTH-FORM-${form} item ${ordinal}.`,
        configuredReference: "Synthetic controlled reference",
        expectedEvidence: "Synthetic evidence record",
        sourceLocator: `synthetic://form/${form}/item/${ordinal}`,
        sourceGapState: usageClass === "PREPROD_EXERCISE" ? "SOURCE_MAPPING_REQUIRED" : "RESOLVED",
        proposedDomain: "SYNTHETIC_DOMAIN",
        proposedTopic: "SYNTHETIC_TOPIC",
        proposedRiskBand: "PROPOSED_REVIEW_REQUIRED",
        canSelect: true,
        canPublish: usageClass === "GOVERNED_OPERATIONAL",
      };
    });
  }

  readonly packageDrafts: DemoBackend["packageDrafts"] = {
    get: async ({ packageDraftId }) => {
      requireDemoCapability(this.principal, "packageDrafts");
      requireRole(this.principal, ["manager"], "Department Manager authority is required for Inspection Package drafts.");
      return this.store.read((state) => {
        const draft = state.inspectionPackageDrafts[packageDraftId];
        if (!draft) throw new BackendInvariantError(`Inspection Package draft ${packageDraftId} was not found.`);
        return draft;
      });
    },
    save: async (input) => {
      requireDemoCapability(this.principal, "packageDrafts");
      requireRole(this.principal, ["manager"], "Department Manager authority is required for Inspection Package drafts.");
      requireNonEmpty(input.idempotencyKey, "Idempotency key");
      return this.store.execute(input.idempotencyKey, input, (state) => {
        const draft = state.inspectionPackageDrafts[input.packageDraftId];
        if (!draft) throw new BackendInvariantError(`Inspection Package draft ${input.packageDraftId} was not found.`);
        requireRevision(draft.revision, input.expectedRevision, "Inspection Package draft");
        const riskFocus = input.riskFocus.map((item) => item.trim()).filter(Boolean);
        if (riskFocus.length === 0) throw new BackendInvariantError("Inspection Package risk focus is required.");
        const saved = {
          ...draft,
          riskFocus,
          revision: draft.revision + 1,
          updatedAt: this.store.clock(),
        };
        state.inspectionPackageDrafts[input.packageDraftId] = saved;
        return saved;
      });
    },
  };

  readonly assignments: Backend["assignments"] = {
    list: async (input) => {
      if (this.principal.role === "auditee") {
        throw new BackendAuthorizationInvariantError(
          "Legacy assignment projections are unavailable to Auditee users; use Auditee Coordination.",
        );
      }
      return this.store.read((state) => {
        let items = state.assignments;
        if (this.principal.role === "inspector") {
          items = items.filter((assignment) => {
            const packageView = Object.values(state.packages).find(
              (candidate) => candidate.auditId === assignment.auditId,
            );
            return packageView?.questions.some((question) =>
              question.assignedInspectorUserIds.includes(this.principal.subjectId),
            );
          });
        }
        if (input.status) items = items.filter((assignment) => assignment.status === input.status);
        const limit = input.limit ?? items.length;
        return { items: items.slice(0, limit), nextCursor: null };
      });
    },
  };

  readonly inspections: Backend["inspections"] = {
    start: async (input) => {
      requireRole(this.principal, ["inspector"], "Assigned Inspector authority is required to start an Audit.");
      return this.store.execute(input.operationId, input, (state) => {
        const packageView = getPackage(state, input.auditId);
        if (packageView.checklistStatus !== "NOT_STARTED") {
          throw new BackendConflictError("Audit is not awaiting the separate Inspector start transition.");
        }
        const questionIds = packageView.questions.filter((question) => question.assignedInspectorUserIds.includes(this.principal.subjectId));
        if (!questionIds.length) throw new BackendAuthorizationInvariantError("Inspector is not assigned to this Audit.");
        const assignment = state.assignments.find((candidate) => candidate.auditId === input.auditId);
        if (!assignment || !["CONFIRMED", "SCHEDULED", "READY"].includes(assignment.status)) {
          throw new BackendConflictError("Audit readiness has not been satisfied.");
        }
        const mutablePackage = state.packages[packageView.id];
        mutablePackage.checklistStatus = "IN_PROGRESS";
        mutablePackage.checklistRevision += 1;
        assignment.status = "READY";
        return {
          inspectionId: input.auditId,
          assignmentId: `ASSIGN-${input.auditId}`,
          inspectionStatus: "IN_PROGRESS" as const,
          assignmentStatus: assignment.status,
          inspectionRevision: input.expectedInspectionRevision + 1,
          checklistRevision: mutablePackage.checklistRevision,
          startedAt: this.store.clock(),
        };
      });
    },
    getPackage: async ({ packageId }) =>
      this.store.read((state) => {
        const packageView = getPackage(state, packageId);
        if (this.principal.role === "auditee") {
          throw new BackendAuthorizationInvariantError(
            "Inspection execution packages are not available to Auditee users.",
          );
        }
        if ((this.principal.role === "inspector" || this.principal.role === "leadInspector") && packageView.checklistStatus === "NOT_STARTED") {
          throw new BackendConflictError("Execution package is unavailable before Inspector start.");
        }
        return packageView;
      }),

    checkout: async (input) => {
      requireRole(this.principal, ["inspector", "leadInspector"], "Inspector authority is required.");
      return this.store.execute(input.operationId, input, (state) => {
        const packageView = getPackage(state, input.packageId);
        if (packageView.checklistStatus === "NOT_STARTED") {
          throw new BackendConflictError("Offline execution grants are unavailable before Inspector start.");
        }
        requireRevision(packageView.packageVersion, input.expectedPackageVersion, "Package");
        const questionIds = packageView.questions
          .filter(
            (question) =>
              this.principal.role === "leadInspector" ||
              question.assignedInspectorUserIds.includes(this.principal.subjectId),
          )
          .map((question) => question.id);
        return {
          inspectionPackage: packageView,
          offlineGrant: {
            grantId: "GRANT-CANDIDATE-001",
            subjectId: this.principal.subjectId,
            organizationId: packageView.organizationId,
            packageId: packageView.id,
            packageVersion: packageView.packageVersion,
            packageDigest: packageView.packageDigest,
            allowedCommandTypes: [
              "UPSERT_CHECKLIST_RESPONSE",
              "CREATE_POTENTIAL_FINDING",
              "SUBMIT_CHECKLIST",
              "REGISTER_INSPECTION_ATTACHMENT",
            ],
            assignmentScope: { questionIds },
            deviceInstanceId: input.deviceInstanceId,
            issuedAt: this.store.clock(),
            expiresAt: "2026-07-15T23:59:59.000Z",
            protocolVersion: packageView.protocolVersion,
          },
        };
      });
    },

    upsertChecklistResponse: async (input) => {
      requireRole(
        this.principal,
        ["inspector", "leadInspector"],
        "Inspector or Lead Inspector authority is required.",
      );
      return this.store.execute(input.operationId, input, (state) => {
        const packageView = packageForAudit(state, input.auditId);
        if (packageView.checklistStatus === "SUBMITTED") {
          throw new BackendInvariantError("Submitted checklist is read-only until a reasoned reopen.");
        }
        const question = packageView.questions.find((candidate) => candidate.id === input.questionId);
        if (!question) {
          throw new BackendInvariantError("Checklist response must target an exact Audit question.");
        }
        if (
          this.principal.role === "inspector" &&
          !question.assignedInspectorUserIds.includes(this.principal.subjectId)
        ) {
          throw new BackendAuthorizationInvariantError(
            "This question is read-only because it belongs to another assigned Inspector.",
          );
        }
        if (!question.allowedAnswers.includes(input.answer)) {
          throw new BackendInvariantError("Checklist answer is not allowed for this question.");
        }
        if (question.commentRequiredFor.includes(input.answer)) {
          requireNonEmpty(input.comment, "Required checklist comment");
        }
        const existing = state.checklistResponses[input.responseId];
        requireRevision(existing?.revision ?? null, input.expectedResponseRevision, "Checklist response");
        if (existing && existing.questionId !== input.questionId) {
          throw new BackendInvariantError("Checklist response identity cannot move to another question.");
        }
        const response: ChecklistResponseView = {
          id: input.responseId,
          questionId: input.questionId,
          answer: input.answer,
          comment: input.comment.trim(),
          revision: (existing?.revision ?? 0) + 1,
          updatedAt: this.store.clock(),
        };
        state.checklistResponses[response.id] = response;
        return response;
      });
    },

    submitChecklist: async (input) => {
      requireRole(
        this.principal,
        ["inspector", "leadInspector"],
        "Inspector or Lead Inspector authority is required.",
      );
      return this.store.execute(input.operationId, input, (state) => {
        const packageView = packageForAudit(state, input.auditId);
        requireRevision(packageView.checklistRevision, input.expectedChecklistRevision, "Checklist");
        if (!packageView.questions.every((question) =>
          Object.values(state.checklistResponses).some((response) => response.questionId === question.id))) {
          throw new BackendInvariantError("Every assigned checklist question requires an exact response before submission.");
        }
        packageView.checklistStatus = "SUBMITTED";
        packageView.checklistRevision += 1;
        return {
          auditId: packageView.auditId,
          checklistStatus: packageView.checklistStatus,
          checklistRevision: packageView.checklistRevision,
        };
      });
    },

    reopenChecklist: async (input) => {
      requireRole(
        this.principal,
        ["inspector", "leadInspector"],
        "Inspector or Lead Inspector authority is required.",
      );
      requireNonEmpty(input.reason, "Reopen reason");
      return this.store.execute(input.operationId, input, (state) => {
        const packageView = packageForAudit(state, input.auditId);
        requireRevision(packageView.checklistRevision, input.expectedChecklistRevision, "Checklist");
        if (packageView.checklistStatus !== "SUBMITTED") {
          throw new BackendInvariantError("Only a submitted checklist can be reopened.");
        }
        packageView.checklistStatus = "IN_PROGRESS";
        packageView.checklistRevision += 1;
        return {
          auditId: packageView.auditId,
          checklistStatus: packageView.checklistStatus,
          checklistRevision: packageView.checklistRevision,
        };
      });
    },
  };

  readonly potentialFindings: Backend["potentialFindings"] = {
    list: async (input) =>
      this.store.read((state) => {
        requireRole(this.principal, ["leadInspector"], "Lead Inspector authority is required.");
        let items = Object.values(state.potentialFindings);
        if (input.status) items = items.filter((potential) => potential.status === input.status);
        items = items.sort((left, right) => left.id.localeCompare(right.id));
        const limit = input.limit ?? items.length;
        return { items: items.slice(0, limit), nextCursor: null };
      }),

    get: async ({ potentialFindingId }) =>
      this.store.read((state) =>
        potentialFindingForPrincipal(state, this.principal, potentialFindingId),
      ),

    create: async (input) => {
      requireRole(this.principal, ["inspector"], "CAA Inspector authority is required.");
      return this.store.execute(input.operationId, input, (state) => {
        const packageView = packageForAudit(state, input.auditId);
        const question = packageView.questions.find((candidate) => candidate.id === input.questionId);
        if (!question) throw new BackendInvariantError("Potential Finding must target an exact Audit question.");
        if (!question.assignedInspectorUserIds.includes(this.principal.subjectId)) {
          throw new BackendAuthorizationInvariantError(
            "Potential Finding requires the question's assigned Inspector.",
          );
        }
        const response = state.checklistResponses[input.checklistResponseId];
        if (!response || response.questionId !== input.questionId) {
          throw new BackendInvariantError(
            "Potential Finding must target the exact checklist response and question.",
          );
        }
        requireRevision(
          response.revision,
          input.expectedChecklistResponseRevision,
          "Checklist response",
        );
        if (!(["NON_COMPLIANT", "OBSERVATION"] as const).includes(response.answer as never)) {
          throw new BackendInvariantError(
            "Only Non-Compliant or Observation responses may create a Potential Finding.",
          );
        }
        requireNonEmpty(input.requiredComment, "Required Potential Finding comment");
        if (
          Object.values(state.potentialFindings).some(
            (candidate) =>
              candidate.auditId === input.auditId &&
              candidate.questionId === input.questionId &&
              candidate.status !== "DISMISSED",
          )
        ) {
          throw new BackendConflictError("An active Potential Finding already exists for this response.");
        }
        const sequence = state.counters.potentialFinding++;
        const potential: PotentialFindingView = {
          id: `PF-2026-${pad(sequence)}`,
          auditId: input.auditId,
          questionId: input.questionId,
          organizationId: packageView.organizationId,
          title: input.title.trim(),
          description: input.description.trim(),
          status: "PENDING_LEAD_REVIEW",
          revision: 1,
          convertedFindingId: null,
        };
        state.potentialFindings[potential.id] = potential;
        return potential;
      });
    },

    decide: async (input) => {
      requireRole(this.principal, ["leadInspector"], "Lead Inspector authority is required.");
      return this.store.execute(input.operationId, input, (state) => {
        const potential = state.potentialFindings[input.potentialFindingId];
        if (!potential) throw new BackendInvariantError("Potential Finding was not found.");
        requireRevision(
          potential.revision,
          input.expectedPotentialFindingRevision,
          "Potential Finding",
        );
        if (potential.status !== "PENDING_LEAD_REVIEW" && potential.status !== "RETURNED") {
          throw new BackendInvariantError("Potential Finding is not available for a Lead decision.");
        }
        if (input.decision === "RETURN" || input.decision === "DISMISS") {
          requireNonEmpty(input.reason, "Lead decision reason");
          potential.status = input.decision === "RETURN" ? "RETURNED" : "DISMISSED";
          potential.revision += 1;
          return { potentialFinding: potential, finding: null };
        }

        const conversion = input as Extract<typeof input, { decision: "CONVERT" }>;

        const packageView = packageForAudit(state, potential.auditId);
        const question = packageView.questions.find(
          (candidate) => candidate.id === potential.questionId,
        );
        if (!question) throw new BackendInvariantError("Potential Finding question is unavailable.");
        const sequence = state.counters.finding++;
        const findingId =
          sequence === 1 ? "FND-CAB-2026-001" : `FND-CAB-2026-${pad(sequence)}`;
        const findingNumber = `CAB-2026-${pad(sequence)}`;
        if (state.findings[findingId]) {
          throw new BackendConflictError(`Finding identity ${findingId} already exists and cannot be overwritten.`);
        }
        const findingStatus = conversion.capRequired
          ? "WAITING_FOR_CAP"
          : conversion.evidenceRequired
            ? "EVIDENCE_REQUIRED"
            : "PENDING_CLOSURE";
        const auditeeOwnsNextAction = conversion.capRequired || conversion.evidenceRequired;
        const finding: FindingView = {
          id: findingId,
          findingNumber,
          auditId: potential.auditId,
          organizationId: potential.organizationId,
          organizationName: packageView.organizationName,
          title: potential.title,
          description: potential.description,
          regulatoryReference: question.regulatoryReference,
          findingBasis: `Non-Compliant response and required Inspector comment for ${question.id}`,
          severity: conversion.severity,
          status: findingStatus,
          dueDate: conversion.dueDate,
          dueState: conversion.dueDate ? "NOT_DUE" : "NONE",
          currentOwnerType: auditeeOwnsNextAction ? "AUDITEE" : "CAA",
          currentOwnerId: auditeeOwnsNextAction
            ? potential.organizationId
            : this.principal.subjectId,
          currentOwnerRole: auditeeOwnsNextAction ? "auditee" : "leadInspector",
          nextAction: conversion.capRequired
            ? "Auditee to submit CAP"
            : conversion.evidenceRequired
              ? "Auditee to submit required Evidence"
            : "CAA to verify closure path",
          capRequired: conversion.capRequired,
          evidenceRequired: conversion.evidenceRequired,
          repeatFinding: false,
          createdAt: this.store.clock(),
          issuedAt: this.store.clock(),
          closedAt: null,
          closureBasis: null,
          revision: 1,
        };
        state.findings[finding.id] = finding;
        potential.status = "CONVERTED";
        potential.convertedFindingId = finding.id;
        potential.revision += 1;
        return { potentialFinding: potential, finding };
      });
    },
  };

  readonly findings: Backend["findings"] = {
    list: async (input) =>
      this.store.read((state) => {
        let items = Object.values(state.findings);
        if (this.principal.role === "auditee") {
          items = items.filter((finding) => finding.organizationId === this.principal.organizationId);
        }
        if (input.status) items = items.filter((finding) => finding.status === input.status);
        items = items.sort((left, right) => left.findingNumber.localeCompare(right.findingNumber));
        const limit = input.limit ?? items.length;
        return { items: items.slice(0, limit), nextCursor: null };
      }),

    get: async ({ findingId }) =>
      this.store.read((state) => findingForPrincipal(state, this.principal, findingId)),

    authorizedClose: async (input) => {
      requireRole(
        this.principal,
        ["manager"],
        "Department Manager authority is required for authorized closure.",
      );
      requireNonEmpty(input.reason, "Authorized closure reason");
      return this.store.execute(input.operationId, input, (state) => {
        const finding = mutableFinding(state, input.findingId);
        requireRevision(finding.revision, input.expectedFindingRevision, "Finding");
        if (finding.status === "CLOSED") throw new BackendInvariantError("Finding is already closed.");
        const beforeStatus = finding.status;
        finding.status = "CLOSED";
        finding.currentOwnerType = "CAA";
        finding.currentOwnerId = this.principal.subjectId;
        finding.currentOwnerRole = "manager";
        finding.nextAction = "No action — Finding closed through authorized path";
        finding.closedAt = this.store.clock();
        finding.closureBasis = "AUTHORIZED";
        finding.revision += 1;
        state.auditEvents.push({
          eventId: `AUDIT-FINDING-${pad(state.counters.auditEvent++)}`,
          occurredAt: this.store.clock(),
          actorRole: this.principal.role,
          actorSubjectId: this.principal.subjectId,
          action: "finding.authorized_closure",
          entityType: "finding",
          entityId: finding.id,
          beforeStatus,
          afterStatus: finding.status,
          reason: input.reason.trim(),
          entityRevision: finding.revision,
        });
        return finding;
      });
    },
  };

  readonly caps: Backend["caps"] = {
    listRevisions: async ({ findingId }) =>
      this.store.read((state) => {
        const finding = state.findings[findingId];
        if (!finding) throw new BackendInvariantError(`Finding ${findingId} was not found.`);
        const audience = capReadAudience(this.principal, finding.organizationId);
        const items = state.capRevisions
          .filter((cap) => cap.findingId === finding.id)
          .sort((left, right) => left.version - right.version)
          .map((cap) => capRevisionView(cap, audience));
        return { items, nextCursor: null };
      }),

    getRevision: async ({ capRevisionId }) =>
      this.store.read((state) => {
        const cap = state.capRevisions.find((revision) => revision.id === capRevisionId);
        if (!cap) throw new BackendInvariantError("CAP revision was not found.");
        const audience = capReadAudience(this.principal, cap.organizationId);
        return capRevisionView(cap, audience);
      }),

    submit: async (input) => {
      requireRole(this.principal, ["auditee"], "Auditee authority is required to submit CAP.");
      for (const [value, label] of [
        [input.rootCause, "Root cause"],
        [input.correctiveAction, "Corrective action"],
        [input.preventiveAction, "Preventive action"],
        [input.responsiblePerson, "Responsible person"],
        [input.targetCompletionDate, "Target completion date"],
      ] as const) {
        requireNonEmpty(value, label);
      }
      return this.store.execute(input.operationId, input, (state) => {
        const finding = mutableFinding(state, input.findingId);
        requireAuditeeOrganization(this.principal, finding.organizationId);
        requireRevision(finding.revision, input.expectedFindingRevision, "Finding");
        if (
          finding.status !== "WAITING_FOR_CAP" &&
          finding.status !== "CAP_MORE_INFORMATION_REQUESTED" &&
          finding.status !== "CAP_REJECTED"
        ) {
          throw new BackendInvariantError("Finding is not accepting a CAP submission.");
        }
        const existingVersions = state.capRevisions.filter(
          (revision) => revision.findingId === finding.id,
        );
        for (const prior of existingVersions) {
          if (prior.status !== "SUPERSEDED") prior.status = "SUPERSEDED";
        }
        const version = existingVersions.length + 1;
        const capRevisionId = `CAP-${finding.findingNumber}-R${version}`;
        const capId = existingVersions[0]?.capId ?? `CAP-${finding.findingNumber}`;
        state.capRevisions.push({
          id: capRevisionId,
          capId,
          findingId: finding.id,
          organizationId: finding.organizationId,
          version,
          revision: version,
          status: "SUBMITTED",
          rootCause: input.rootCause.trim(),
          correctiveAction: input.correctiveAction.trim(),
          preventiveAction: input.preventiveAction.trim(),
          responsiblePerson: input.responsiblePerson.trim(),
          targetCompletionDate: input.targetCompletionDate,
          commentToCaa: input.commentToCaa.trim(),
          commentToAuditee: "",
          internalCaaNote: "",
          reviewDecision: null,
          submittedAt: this.store.clock(),
          reviewedAt: null,
        });
        finding.status = "CAP_SUBMITTED";
        finding.currentOwnerType = "CAA";
        finding.currentOwnerId = "USR-LEAD-CANER";
        finding.currentOwnerRole = "leadInspector";
        finding.nextAction = "CAA to review submitted CAP";
        finding.revision += 1;
        return {
          capRevisionId,
          capRevision: version,
          capStatus: "SUBMITTED",
          findingStatus: finding.status,
          findingRevision: finding.revision,
        };
      });
    },

    review: async (input) => {
      requireRole(
        this.principal,
        ["inspector", "leadInspector"],
        "CAA Inspector or Lead Inspector authority is required to review CAP.",
      );
      requireSeparateReviewComments(input.commentToAuditee, input.internalCaaNote);
      return this.store.execute(input.operationId, input, (state) => {
        const finding = mutableFinding(state, input.findingId);
        requireRevision(finding.revision, input.expectedFindingRevision, "Finding");
        const cap = state.capRevisions.find((revision) => revision.id === input.capRevisionId);
        if (!cap || cap.findingId !== finding.id) {
          throw new BackendInvariantError("CAP review must target the exact Finding revision.");
        }
        requireRevision(cap.revision, input.expectedCapRevision, "CAP");
        if (cap.status !== "SUBMITTED" && cap.status !== "PENDING_CAA_REVIEW") {
          throw new BackendInvariantError("CAP revision is not pending CAA review.");
        }
        cap.commentToAuditee = input.commentToAuditee.trim();
        cap.internalCaaNote = input.internalCaaNote.trim();
        cap.reviewDecision = input.decision;
        cap.reviewedAt = this.store.clock();
        if (input.decision === "ACCEPT") {
          cap.status = "ACCEPTED";
          finding.status = finding.evidenceRequired ? "EVIDENCE_REQUIRED" : "PENDING_CLOSURE";
          finding.currentOwnerType = finding.evidenceRequired ? "AUDITEE" : "CAA";
          finding.currentOwnerId = finding.evidenceRequired
            ? finding.organizationId
            : this.principal.subjectId;
          finding.currentOwnerRole = finding.evidenceRequired ? "auditee" : this.principal.role;
          finding.nextAction = finding.evidenceRequired
            ? "Auditee to submit required Evidence"
            : "CAA to verify closure";
        } else if (input.decision === "REJECT") {
          cap.status = "REJECTED";
          finding.status = "CAP_REJECTED";
          finding.currentOwnerType = "AUDITEE";
          finding.currentOwnerId = finding.organizationId;
          finding.currentOwnerRole = "auditee";
          finding.nextAction = "Auditee to revise CAP";
        } else {
          cap.status = "MORE_INFORMATION_REQUESTED";
          finding.status = "CAP_MORE_INFORMATION_REQUESTED";
          finding.currentOwnerType = "AUDITEE";
          finding.currentOwnerId = finding.organizationId;
          finding.currentOwnerRole = "auditee";
          finding.nextAction = "Auditee to provide more CAP information";
        }
        finding.revision += 1;
        return {
          capRevisionId: cap.id,
          capRevision: cap.revision,
          capStatus: cap.status,
          findingStatus: finding.status,
          findingRevision: finding.revision,
        };
      });
    },
  };

  readonly inspectionAttachments: Backend["inspectionAttachments"] = {
    beginUpload: async (input) => {
      requireRole(
        this.principal,
        ["inspector", "leadInspector"],
        "Inspector authority is required for Inspection Attachment upload.",
      );
      return this.store.execute(input.operationId, input, (state) => {
        getPackage(state, input.packageId);
        const sequence = state.counters.upload++;
        const uploadId = `UP-ATT-${pad(sequence, 4)}`;
        const upload: MockInspectionAttachmentUpload = {
          kind: "inspection-attachment",
          uploadId,
          inspectionAttachmentId: input.inspectionAttachmentId,
          packageId: input.packageId,
          fileName: input.fileName,
          byteSize: input.byteSize,
          sha256: input.sha256,
        };
        state.uploads[uploadId] = upload;
        return {
          uploadId,
          stagingObjectKey: `candidate/inspection-attachments/${input.inspectionAttachmentId}`,
          uploadUrl: `mock://inspection-attachments/${uploadId}`,
          requiredHeaders: { "x-candidate-sha256": input.sha256 },
          expiresAt: addHours(this.store.clock(), 1),
          maximumByteSize: 10_000_000,
        };
      });
    },

    completeUpload: async (input) =>
      this.store.execute(input.operationId, input, (state) => {
        const upload = state.uploads[input.uploadId];
        if (!upload || upload.kind !== "inspection-attachment") {
          throw new BackendInvariantError("Inspection Attachment upload was not found.");
        }
        if (upload.sha256 !== input.sha256 || upload.byteSize !== input.byteSize) {
          throw new BackendInvariantError("Inspection Attachment checksum or byte size does not match.");
        }
        return {
          inspectionAttachmentId: upload.inspectionAttachmentId,
          uploadState: "UPLOADED",
          scanState: "PENDING",
        };
      }),
  };

  readonly evidence: Backend["evidence"] = {
    beginUpload: async (input) => {
      requireRole(this.principal, ["auditee"], "Auditee authority is required to submit Evidence.");
      requireNonEmpty(input.fileName, "Evidence filename");
      return this.store.execute(input.operationId, input, (state) => {
        const finding = mutableFinding(state, input.findingId);
        requireAuditeeOrganization(this.principal, finding.organizationId);
        requireRevision(finding.revision, input.expectedFindingRevision, "Finding");
        if (
          finding.status !== "EVIDENCE_REQUIRED" &&
          finding.status !== "EVIDENCE_MORE_INFORMATION_REQUESTED"
        ) {
          throw new BackendInvariantError("Finding is not accepting Evidence.");
        }
        const sequence = state.counters.upload++;
        const uploadId = `UP-EV-${pad(sequence, 4)}`;
        const upload: MockEvidenceUpload = {
          kind: "evidence",
          uploadId,
          findingId: finding.id,
          organizationId: finding.organizationId,
          fileName: input.fileName.trim(),
          declaredMediaType: input.declaredMediaType,
          byteSize: input.byteSize,
          sha256: input.sha256,
        };
        state.uploads[uploadId] = upload;
        return {
          uploadId,
          stagingObjectKey: `candidate/evidence/${uploadId}`,
          uploadUrl: `mock://evidence/${uploadId}`,
          requiredHeaders: { "x-candidate-sha256": input.sha256 },
          expiresAt: addHours(this.store.clock(), 1),
          maximumByteSize: 10_000_000,
        };
      });
    },

    completeUpload: async (input) =>
      this.store.execute(input.operationId, input, (state) => {
        const upload = state.uploads[input.uploadId];
        if (!upload || upload.kind !== "evidence") {
          throw new BackendInvariantError("Evidence upload was not found.");
        }
        requireAuditeeOrganization(this.principal, upload.organizationId);
        if (upload.sha256 !== input.sha256 || upload.byteSize !== input.byteSize) {
          throw new BackendInvariantError("Evidence checksum or byte size does not match.");
        }
        const finding = mutableFinding(state, upload.findingId);
        const version =
          state.evidenceVersions.filter((candidate) => candidate.findingId === finding.id).length + 1;
        const evidenceVersion: MockEvidenceVersion = {
          id: `EV-${finding.findingNumber}-V${version}`,
          findingId: finding.id,
          organizationId: finding.organizationId,
          version,
          fileName: upload.fileName,
          submittedAt: this.store.clock(),
          uploadState: "UPLOADED",
          scanState: "CLEAN",
          reviewState: "PENDING_CAA_REVIEW",
          revision: 2,
          commentToAuditee: "",
        };
        state.evidenceVersions.push(evidenceVersion);
        finding.status = "PENDING_CAA_REVIEW";
        finding.currentOwnerType = "CAA";
        finding.currentOwnerId = "USR-LEAD-CANER";
        finding.currentOwnerRole = "leadInspector";
        finding.nextAction = "CAA reviews Evidence";
        finding.revision += 2;
        return {
          evidenceVersionId: evidenceVersion.id,
          version,
          uploadState: "UPLOADED",
          scanState: "CLEAN",
          reviewState: "PENDING_CAA_REVIEW",
        };
      }),

    listVersions: async ({ findingId }) =>
      this.store.read((state) => {
        findingForPrincipal(state, this.principal, findingId);
        return state.evidenceVersions
          .filter((version) => version.findingId === findingId)
          .sort((left, right) => left.version - right.version)
          .map(publicEvidenceVersion);
      }),

    review: async (input) => {
      requireRole(
        this.principal,
        ["inspector", "leadInspector", "manager"],
        "CAA Inspector, Lead Inspector, or Department Manager authority is required to review Evidence.",
      );
      requireSeparateReviewComments(input.commentToAuditee, input.internalCaaNote);
      return this.store.execute(input.operationId, input, (state) => {
        const finding = mutableFinding(state, input.findingId);
        requireRevision(finding.revision, input.expectedFindingRevision, "Finding");
        const evidenceVersion = state.evidenceVersions.find(
          (version) => version.id === input.evidenceVersionId,
        );
        if (!evidenceVersion || evidenceVersion.findingId !== finding.id) {
          throw new BackendInvariantError("Evidence review must target the exact Finding version.");
        }
        const latestVersion = state.evidenceVersions
          .filter((version) => version.findingId === finding.id)
          .sort((left, right) => right.version - left.version)[0];
        if (latestVersion?.id !== evidenceVersion.id) {
          throw new BackendInvariantError("Evidence review must target the exact latest version.");
        }
        requireRevision(
          evidenceVersion.revision,
          input.expectedEvidenceVersionRevision,
          "Evidence version",
        );
        if (evidenceVersion.scanState !== "CLEAN") {
          throw new BackendInvariantError("Only scan-clean Evidence can be reviewed.");
        }
        if (evidenceVersion.reviewState !== "PENDING_CAA_REVIEW") {
          throw new BackendConflictError("Evidence version is not pending CAA review.");
        }
        const beforeFindingStatus = finding.status;
        let reviewState: EvidenceReviewState;
        if (input.decision === "CLOSE") reviewState = "ACCEPTED";
        else if (input.decision === "PARTIALLY_CLOSE") reviewState = "PARTIALLY_ACCEPTED";
        else if (input.decision === "NOT_CLOSE") reviewState = "REJECTED";
        else reviewState = "MORE_INFORMATION_REQUESTED";
        evidenceVersion.reviewState = reviewState;
        evidenceVersion.commentToAuditee = input.commentToAuditee.trim();
        evidenceVersion.revision += 1;

        const reviewSequence = state.counters.evidenceReview++;
        const reviewDecisionId = `EVD-REVIEW-${pad(reviewSequence, 4)}`;
        state.evidenceReviews.push({
          id: reviewDecisionId,
          findingId: finding.id,
          evidenceVersionId: evidenceVersion.id,
          decision: input.decision,
          commentToAuditee: input.commentToAuditee.trim(),
          internalCaaNote: input.internalCaaNote.trim(),
          reviewedAt: this.store.clock(),
        });

        if (input.decision === "CLOSE") {
          finding.status = "CLOSED";
          finding.currentOwnerType = "CAA";
          finding.currentOwnerId = this.principal.subjectId;
          finding.currentOwnerRole = this.principal.role;
          finding.nextAction = "No action — Finding closed";
          finding.closedAt = this.store.clock();
          finding.closureBasis = "EVIDENCE_VERIFIED";
        } else {
          finding.status = "EVIDENCE_MORE_INFORMATION_REQUESTED";
          finding.currentOwnerType = "AUDITEE";
          finding.currentOwnerId = finding.organizationId;
          finding.currentOwnerRole = "auditee";
          finding.nextAction = "Auditee to provide remaining Evidence or information";
          finding.closedAt = null;
          finding.closureBasis = null;
        }
        finding.revision += 1;
        state.auditEvents.push({
          eventId: `AUDIT-EVIDENCE-${pad(state.counters.auditEvent++)}`,
          occurredAt: this.store.clock(),
          actorRole: this.principal.role,
          actorSubjectId: this.principal.subjectId,
          action: "evidence.reviewed",
          entityType: "finding",
          entityId: finding.id,
          beforeStatus: beforeFindingStatus,
          afterStatus: finding.status,
          reason: input.commentToAuditee.trim(),
          entityRevision: finding.revision,
        });
        return {
          reviewDecisionId,
          evidenceVersionId: evidenceVersion.id,
          evidenceVersionRevision: evidenceVersion.revision,
          findingStatus: finding.status,
          findingRevision: finding.revision,
        };
      });
    },
  };

  readonly reports: Backend["reports"] = {
    getVersion: async ({ reportVersionId }) => {
      if (this.principal.role === "auditee") {
        throw new BackendAuthorizationInvariantError(
          "Report version is unavailable to this Auditee; use released Auditee Reports.",
        );
      }
      return this.store.read((state) => {
        const report = state.reportVersions[reportVersionId];
        if (!report) throw new BackendInvariantError("Report version was not found.");
        return report;
      });
    },

    decide: async (input) =>
      this.store.execute(input.operationId, input, (state) => {
        const report = state.reportVersions[input.reportVersionId];
        if (!report) throw new BackendInvariantError("Report version was not found.");
        requireRevision(report.revision, input.expectedReportVersionRevision, "Report version");
        requireNonEmpty(input.reason, "Report decision reason");
        const beforeStatus = report.status;

        if (this.principal.role === "manager" && report.status === "DEPARTMENT_REVIEW") {
          if (input.decision === "ISSUE_AND_LOCK") {
            throw new BackendAuthorizationInvariantError("Department Manager cannot issue or lock reports.");
          }
          report.status = input.decision === "FORWARD" ? "GM_REVIEW" : "RETURNED";
        } else if (this.principal.role === "gm" && report.status === "GM_REVIEW") {
          if (input.decision === "ISSUE_AND_LOCK") {
            throw new BackendAuthorizationInvariantError("General Manager cannot issue or lock reports.");
          }
          report.status =
            input.decision === "FORWARD" ? "EXECUTIVE_DIRECTOR_REVIEW" : "RETURNED";
        } else if (
          this.principal.role === "executiveDirector" &&
          report.status === "EXECUTIVE_DIRECTOR_REVIEW" &&
          input.decision === "ISSUE_AND_LOCK"
        ) {
          report.status = "LOCKED";
          report.issuedAt = this.store.clock();
        } else {
          throw new BackendAuthorizationInvariantError(
            "This role or report stage cannot perform the requested report decision.",
          );
        }
        report.revision += 1;
        state.auditEvents.push({
          eventId: `AUDIT-REPORT-${pad(state.counters.auditEvent++)}`,
          occurredAt: this.store.clock(),
          actorRole: this.principal.role,
          actorSubjectId: this.principal.subjectId,
          action: "report.decision_recorded",
          entityType: "report_version",
          entityId: report.reportVersionId,
          beforeStatus,
          afterStatus: report.status,
          reason: input.reason.trim(),
          entityRevision: report.revision,
        });
        return report;
      }),
  };

  readonly dashboards: Backend["dashboards"] = {
    getManagerProjection: async ({ organizationId }) => {
      requireRole(
        this.principal,
        ["manager", "gm", "executiveDirector"],
        "CAA management authority is required for the manager dashboard.",
      );
      return this.store.read((state) => {
        const findings = Object.values(state.findings).filter(
          (finding) => !organizationId || finding.organizationId === organizationId,
        );
        return {
          generatedAt: this.store.clock(),
          openFindings: findings.filter((finding) => finding.status !== "CLOSED").length,
          closedFindings: findings.filter((finding) => finding.status === "CLOSED").length,
          overdueFindings: findings.filter(
            (finding) => finding.status !== "CLOSED" && finding.dueState === "OVERDUE",
          ).length,
          pendingCapReviews: findings.filter((finding) => finding.status === "CAP_SUBMITTED").length,
          pendingEvidenceReviews: findings.filter(
            (finding) => finding.status === "PENDING_CAA_REVIEW",
          ).length,
          recentFindingNumbers: findings
            .slice()
            .sort((left, right) => right.revision - left.revision)
            .map((finding) => finding.findingNumber),
        };
      });
    },
  };

  readonly organizations: Backend["organizations"] = {
    list: async ({ limit }) => {
      requireRole(
        this.principal,
        ["inspector", "leadInspector", "manager", "gm", "executiveDirector", "auditee", "admin"],
        "Organization Registry access is not available to this role.",
      );
      return this.store.read((state) => {
        let organizations = state.organizations;
        if (this.principal.role === "auditee") {
          organizations = organizations.filter(
            (organization) => organization.id === this.principal.organizationId,
          );
        }
        const items = organizations
          .map((organization) => {
            const openFindingCount = Object.values(state.findings).filter(
              (finding) =>
                finding.organizationId === organization.id && finding.status !== "CLOSED",
            ).length;
            return {
              ...organization,
              openFindingCount,
            };
          })
          .sort((left, right) => left.legalName.localeCompare(right.legalName));
        return { items: items.slice(0, limit ?? items.length), nextCursor: null };
      });
    },
  };

  readonly planning: Backend["planning"] = {
    list: async ({ limit }) => {
      requireRole(
        this.principal,
        ["inspector", "leadInspector", "manager", "finance", "gm", "executiveDirector", "admin"],
        "CAA planning access is required.",
      );
      return this.store.read((state) => {
        const items = Object.values(state.planningItems).sort((left, right) =>
          left.scheduledDate.localeCompare(right.scheduledDate),
        );
        return { items: items.slice(0, limit ?? items.length), nextCursor: null };
      });
    },

    decide: async (input) => {
      requireNonEmpty(input.reason, "Planning decision reason");
      return this.store.execute(input.operationId, input, (state) => {
        const item = state.planningItems[input.planningItemId];
        if (!item) throw new BackendInvariantError("Planning item was not found.");
        requireRevision(item.revision, input.expectedPlanningRevision, "Planning item");
        const beforeStatus = item.status;
        let action: string;

        if (input.decision === "APPROVE_BUDGET") {
          requireRole(this.principal, ["finance"], "Finance Review authority is required.");
          if (item.status !== "FINANCE_REVIEW") {
            throw new BackendConflictError("Planning item is not at Finance Review.");
          }
          item.status = "GM_REVIEW";
          item.currentOwnerRole = "gm";
          item.nextAction = "General Manager to review operational scope";
          action = "PLANNING_BUDGET_APPROVED";
        } else if (input.decision === "FORWARD_FOR_FINAL_APPROVAL") {
          requireRole(this.principal, ["gm"], "General Manager authority is required.");
          if (item.status !== "GM_REVIEW") {
            throw new BackendConflictError("Planning item is not at General Manager review.");
          }
          item.status = "EXECUTIVE_DIRECTOR_REVIEW";
          item.currentOwnerRole = "executiveDirector";
          item.nextAction = "Executive Director to approve or return plan";
          action = "PLANNING_FORWARDED_FOR_FINAL_APPROVAL";
        } else if (input.decision === "APPROVE_PLAN") {
          requireRole(
            this.principal,
            ["executiveDirector"],
            "Executive Director authority is required.",
          );
          if (item.status !== "EXECUTIVE_DIRECTOR_REVIEW") {
            throw new BackendConflictError("Planning item is not at Executive Director review.");
          }
          item.status = "GM_RELEASE";
          item.currentOwnerRole = "gm";
          item.nextAction = "General Manager to release approved plan";
          action = "PLANNING_APPROVED";
        } else if (input.decision === "RELEASE_PLAN") {
          requireRole(this.principal, ["gm"], "General Manager authority is required.");
          if (item.status !== "GM_RELEASE") {
            throw new BackendConflictError("Planning item is not ready for General Manager release.");
          }
          item.status = "RELEASED";
          item.currentOwnerRole = "manager";
          item.nextAction = "Department Manager to prepare the scheduled Audit";
          action = "PLANNING_RELEASED";
        } else {
          const allowed =
            (this.principal.role === "finance" && item.status === "FINANCE_REVIEW") ||
            (this.principal.role === "gm" && ["GM_REVIEW", "GM_RELEASE"].includes(item.status)) ||
            (this.principal.role === "executiveDirector" &&
              item.status === "EXECUTIVE_DIRECTOR_REVIEW");
          if (!allowed) {
            throw new BackendAuthorizationInvariantError(
              "The current role and planning stage cannot return this item.",
            );
          }
          item.status = "RETURNED";
          item.currentOwnerRole = "manager";
          item.nextAction = "Department Manager to revise and resubmit plan";
          action = "PLANNING_RETURNED_FOR_REVISION";
        }
        item.revision += 1;
        state.auditEvents.push({
          eventId: `AUDIT-PLAN-${pad(state.counters.auditEvent++)}`,
          occurredAt: this.store.clock(),
          actorRole: this.principal.role,
          actorSubjectId: this.principal.subjectId,
          action,
          entityType: "SURVEILLANCE_PLAN",
          entityId: item.id,
          beforeStatus,
          afterStatus: item.status,
          reason: input.reason.trim(),
          entityRevision: item.revision,
        });
        return item;
      });
    },
  };

  readonly configuration: Backend["configuration"] = {
    listChecklistTemplateVersions: async ({ limit }) => {
      requireRole(this.principal, ["admin"], "Admin configuration authority is required.");
      return this.store.read((state) => ({
        items: state.checklistTemplateVersions.slice(0, limit ?? state.checklistTemplateVersions.length),
        nextCursor: null,
      }));
    },
    getChecklistTemplateVersion: async ({ templateVersionId }) => {
      requireRole(this.principal, ["admin"], "Admin configuration authority is required.");
      return this.store.read((state) => {
        const detail = state.checklistTemplateVersionDetails[templateVersionId];
        if (!detail) {
          throw new BackendInvariantError(`Checklist Template Version ${templateVersionId} was not found.`);
        }
        return detail;
      });
    },
    listReminderRules: async ({ limit }) => {
      requireRole(this.principal, ["admin"], "Admin configuration authority is required.");
      return this.store.read((state) => ({
        items: state.reminderRules.slice(0, limit ?? state.reminderRules.length),
        nextCursor: null,
      }));
    },
  };

  readonly auditTrail: Backend["auditTrail"] = {
    list: async ({ entityType, entityId, limit }) => {
      requireRole(
        this.principal,
        ["inspector", "leadInspector", "manager", "gm", "executiveDirector", "admin"],
        "Internal CAA audit-trail authority is required.",
      );
      return this.store.read((state) => {
        const items = state.auditEvents.filter(
          (event) =>
            (!entityType || event.entityType === entityType) &&
            (!entityId || event.entityId === entityId),
        );
        return { items: items.slice(0, limit ?? items.length), nextCursor: null };
      });
    },
  };

  readonly sync: Backend["sync"] = {
    pushOperation: async ({ operation }) => {
      requireRole(this.principal, ["inspector"], "CAA Inspector authority is required for sync.");
      return this.store.execute(operation.operationId, operation, (state) => {
        const packageView = getPackage(state, operation.packageId);
        if (
          operation.packageVersion !== packageView.packageVersion ||
          operation.protocolVersion !== packageView.protocolVersion
        ) {
          throw new BackendConflictError("Sync package or protocol version does not match.");
        }
        if (
          operation.offlineGrantId !== "GRANT-CANDIDATE-001" ||
          operation.deviceInstanceId !== "DEVICE-CANDIDATE-001"
        ) {
          return {
            operationId: operation.operationId,
            status: "forbidden" as const,
            authoritativeEntityId: null,
            authoritativeRevision: null,
            errorCode: "OFFLINE_GRANT_DEVICE_MISMATCH",
            conflict: null,
            acknowledgedAt: this.store.clock(),
          };
        }
        if (operation.commandType === "UPSERT_CHECKLIST_RESPONSE") {
          const current = state.checklistResponses[operation.entityId];
          if ((current?.revision ?? null) !== operation.baseRevision) {
            return {
              operationId: operation.operationId,
              status: "conflict" as const,
              authoritativeEntityId: operation.entityId,
              authoritativeRevision: current?.revision ?? null,
              errorCode: "STALE_REVISION",
              conflict: {
                code: "STALE_REVISION" as const,
                entityId: operation.entityId,
                authoritativeRevision: current?.revision ?? null,
                authoritativeStatus: current?.answer ?? null,
                changedAt: current?.updatedAt ?? null,
              },
              acknowledgedAt: this.store.clock(),
            };
          }
          const question = packageView.questions.find(
            ({ id }) => id === operation.payload.questionId,
          );
          if (!question?.assignedInspectorUserIds.includes(this.principal.subjectId)) {
            return {
              operationId: operation.operationId,
              status: "forbidden" as const,
              authoritativeEntityId: null,
              authoritativeRevision: null,
              errorCode: "QUESTION_ASSIGNMENT_FORBIDDEN",
              conflict: null,
              acknowledgedAt: this.store.clock(),
            };
          }
          const response: ChecklistResponseView = {
            id: operation.entityId,
            questionId: operation.payload.questionId,
            answer: operation.payload.answer,
            comment: operation.payload.comment,
            revision: (current?.revision ?? 0) + 1,
            updatedAt: this.store.clock(),
          };
          state.checklistResponses[response.id] = response;
          state.authorizedSyncChanges.push({ kind: "checklist_response", value: response });
          return {
            operationId: operation.operationId,
            status: "accepted" as const,
            authoritativeEntityId: response.id,
            authoritativeRevision: response.revision,
            errorCode: null,
            conflict: null,
            acknowledgedAt: this.store.clock(),
          };
        }
        return {
          operationId: operation.operationId,
          status: "accepted" as const,
          authoritativeEntityId: operation.entityId,
          authoritativeRevision: 1,
          errorCode: null,
          conflict: null,
          acknowledgedAt: this.store.clock(),
        };
      });
    },

    pull: async (input) => {
      requireRole(this.principal, ["inspector"], "CAA Inspector authority is required for sync.");
      if (input.offlineGrantId !== "GRANT-CANDIDATE-001") {
        throw new BackendAuthorizationInvariantError(
          "Offline grant is unavailable to this device session.",
        );
      }
      return this.store.read((state) => {
        getPackage(state, input.packageId);
        return {
          changes: state.authorizedSyncChanges,
          nextCursor: input.cursor,
          hasMore: false,
          resnapshotRequired: false,
          projectionVersion: 1,
        };
      });
    },
  };
}
