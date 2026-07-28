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
  EvidenceReviewState,
  FindingStatus,
  FindingView,
  InspectionPackage,
  InspectionTeamAuditView,
  PlanningIntakeDraftView,
  PotentialFindingView,
} from "../backend/backend";
import {
  BackendAuthorizationInvariantError,
  BackendConflictError,
  BackendInvariantError,
  requireNonEmpty,
  requireDemoCapability,
  requireRevision,
  requireRole,
} from "../backend/backend-contracts";
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

export class MockBackendEngine implements DemoBackend {
  readonly mode = "mock" as const;

  constructor(
    private readonly store: MemoryMockStore,
    private readonly principal: BackendPrincipal,
  ) {}

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

  readonly adminWorkspace: DemoBackend["adminWorkspace"] = {
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
        if (state.planningItems[input.planningItemId]) {
          throw new BackendConflictError(`Planning item ${input.planningItemId} already exists.`);
        }
        const planningItem = {
          id: input.planningItemId,
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
    getPackage: async ({ packageId }) =>
      this.store.read((state) => {
        const packageView = getPackage(state, packageId);
        if (this.principal.role === "auditee") {
          throw new BackendAuthorizationInvariantError(
            "Inspection execution packages are not available to Auditee users.",
          );
        }
        return packageView;
      }),

    checkout: async (input) => {
      requireRole(this.principal, ["inspector", "leadInspector"], "Inspector authority is required.");
      return this.store.execute(input.operationId, input, (state) => {
        const packageView = getPackage(state, input.packageId);
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
        if (!Object.values(state.checklistResponses).some((response) => response.questionId === "CAB-EMEQ-PBE-001")) {
          throw new BackendInvariantError("The canonical PBE response is required before submission.");
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
