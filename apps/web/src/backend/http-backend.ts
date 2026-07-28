import type { components } from "../generated/transport/api-types";
import type { PublicHttpConfig } from "../app/public-http-config";
import type {
  Backend,
  BackendRequestOptions,
  DecidePotentialFindingInput,
  FieldSyncOperation,
} from "./backend";
import {
  activeBrowserRequestHeaders,
  recordActiveBrowserAPIOutcome,
} from "../telemetry/browser-telemetry";
import {
  mapAdminAccessDirectoryEntry,
  mapAdminInspectionPackage,
  mapAdminOrganization,
  mapAdminQuestion,
  mapAdminRegulatoryReference,
  mapAdminReportDefinition,
  mapAdminTemplate,
  mapAdminTemplateMaster,
  mapAdminTemplateVersion,
  mapAdministrationScreenProjection,
  mapAssistantDraft,
  mapAssignments,
  mapAuditeeCoordination,
  mapAuditeeReleasedReport,
  mapAuditeeReleasedReports,
  mapAuditEvents,
  mapCalendarItem,
  mapCalendarItems,
  mapChecklistTemplateVersions,
  mapChecklistTemplateVersionDetail,
  mapChecklistResponse,
  mapCheckout,
  mapCapRevision,
  mapCapRevisions,
  mapCompleteEvidence,
  mapCompleteInspectionAttachment,
  mapCommunication,
  mapCommunications,
  mapDocumentMetadata,
  mapDocuments,
  mapEvidenceVersion,
  mapFinding,
  mapFindings,
  mapInspectionPackage,
  mapInspectionPackageDraft,
  mapInspectionTeamAudit,
  mapManagerDashboard,
  mapNotification,
  mapNotifications,
  mapOrganizations,
  mapPlanningItem,
  mapPlanningIntakeDraft,
  mapPlanningItems,
  mapPotentialFinding,
  mapPotentialFindingDecision,
  mapPotentialFindings,
  mapProfile,
  mapPushResult,
  mapReportVersion,
  mapReminderRules,
  mapRiskManagementProjection,
  mapRiskOverview,
  mapReviewCap,
  mapReviewEvidence,
  mapSubmitCap,
  mapSubmitChecklist,
  mapSubmitPlanningIntake,
  mapSyncPull,
  mapTeamMember,
  mapUserLifecycleRequest,
  mapVisibleActionResult,
} from "./transport-mappers";

type Schemas = components["schemas"];

export interface BackendProblem {
  type: string;
  title: string;
  status: number;
  detail: string | null;
  code: string | null;
  requestId: string | null;
}

export class BackendHttpError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code: string | null,
    readonly requestId: string | null,
    readonly problem: BackendProblem | null,
  ) {
    super(message);
    this.name = "BackendHttpError";
  }
}

export class BackendAuthenticationError extends BackendHttpError {
  constructor(problem: BackendProblem | null, requestId: string | null) {
    super(problem?.title ?? "Authentication required", 401, problem?.code ?? null, requestId, problem);
    this.name = "BackendAuthenticationError";
  }
}

export class BackendAuthorizationError extends BackendHttpError {
  constructor(problem: BackendProblem | null, requestId: string | null) {
    super(problem?.title ?? "Forbidden", 403, problem?.code ?? null, requestId, problem);
    this.name = "BackendAuthorizationError";
  }
}

export class BackendProtocolError extends Error {
  constructor(message: string, readonly requestId: string | null = null) {
    super(message);
    this.name = "BackendProtocolError";
  }
}

export class BackendCancelledError extends Error {
  constructor() {
    super("Backend request was cancelled.");
    this.name = "BackendCancelledError";
  }
}

export class BackendTimeoutError extends Error {
  constructor() {
    super("Backend request timed out.");
    this.name = "BackendTimeoutError";
  }
}

export interface HttpBackendDependencies {
  fetchImplementation?: typeof fetch;
  csrfToken?: () => string | null;
  requestTimeoutMs?: number;
  onAuthenticationLost?: (error: BackendAuthenticationError) => void;
}

interface RequestInput {
  method?: "GET" | "POST" | "PUT";
  body?: unknown;
  headers?: Record<string, string>;
}

function joinApiPath(apiBaseUrl: string, path: string): string {
  const prefix = apiBaseUrl === "/" ? "" : apiBaseUrl.replace(/\/$/, "");
  return `${prefix}${path}`;
}

function appendQuery<T extends object>(path: string, values: T): string {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(values as Record<string, unknown>)) {
    if (typeof value !== "string" && typeof value !== "number" && value != null) continue;
    if (value !== undefined && value !== null && value !== "") query.set(key, String(value));
  }
  const encoded = query.toString();
  return encoded ? `${path}?${encoded}` : path;
}

function revisionCommandHeaders(input: {
  idempotencyKey: string;
  expectedRevision: number | null;
}): Record<string, string> {
  const headers: Record<string, string> = {
    "Idempotency-Key": input.idempotencyKey,
  };
  if (input.expectedRevision !== null) {
    headers["If-Match"] = `"rev-${input.expectedRevision}"`;
  }
  return headers;
}

function revisionCommandBody<T extends {
  idempotencyKey: string;
}>(input: T): T & { operationId: string } {
  return { ...input, operationId: input.idempotencyKey };
}

function stableCommandKey(prefix: string, fields: readonly string[]): string {
  let hash = 0x811c9dc5;
  for (const character of fields.join("\u001f")) {
    hash ^= character.codePointAt(0) ?? 0;
    hash = Math.imul(hash, 0x01000193);
  }
  return `${prefix}:${(hash >>> 0).toString(16).padStart(8, "0")}`;
}

function parseProblem(value: unknown, fallbackStatus: number): BackendProblem | null {
  if (!value || typeof value !== "object") return null;
  const candidate = value as Record<string, unknown>;
  if (typeof candidate.title !== "string") return null;
  return {
    type: typeof candidate.type === "string" ? candidate.type : "about:blank",
    title: candidate.title,
    status: typeof candidate.status === "number" ? candidate.status : fallbackStatus,
    detail: typeof candidate.detail === "string" ? candidate.detail : null,
    code: typeof candidate.code === "string" ? candidate.code : null,
    requestId: typeof candidate.requestId === "string" ? candidate.requestId : null,
  };
}

export function createHttpBackend(
  config: PublicHttpConfig,
  dependencies: HttpBackendDependencies = {},
): Backend {
  const fetchImplementation = dependencies.fetchImplementation ?? fetch;
  const csrfToken = dependencies.csrfToken ?? (() => null);
  let authenticationLostNotified = false;

  async function request<T>(
    path: string,
    requestInput: RequestInput = {},
    options: BackendRequestOptions = {},
  ): Promise<T> {
    const method = requestInput.method ?? "GET";
    const telemetryOperation =
      method === "GET" ? "read" : "command";
    const headers = new Headers({ Accept: "application/json" });
    for (const [name, value] of Object.entries(requestInput.headers ?? {})) {
      headers.set(name, value);
    }
    for (const [name, value] of Object.entries(activeBrowserRequestHeaders())) {
      headers.set(name, value);
    }
    if (requestInput.body !== undefined) {
      headers.set("content-type", "application/json");
      const token = csrfToken();
      if (token) headers.set("x-csrf-token", token);
    }

    let timeoutController: AbortController | null = null;
    let timeoutHandle: ReturnType<typeof setTimeout> | null = null;
    let signal = options.signal;
    if (dependencies.requestTimeoutMs !== undefined) {
      timeoutController = new AbortController();
      timeoutHandle = setTimeout(() => timeoutController?.abort(), dependencies.requestTimeoutMs);
      signal = options.signal
        ? AbortSignal.any([options.signal, timeoutController.signal])
        : timeoutController.signal;
    }

    let response: Response;
    try {
      response = await fetchImplementation(joinApiPath(config.apiBaseUrl, path), {
        method,
        credentials: "same-origin",
        headers,
        body: requestInput.body === undefined ? undefined : JSON.stringify(requestInput.body),
        signal,
      });
    } catch (error) {
      if (error instanceof DOMException && error.name === "AbortError") {
        if (timeoutController?.signal.aborted && !options.signal?.aborted) {
          recordActiveBrowserAPIOutcome(telemetryOperation, "failed");
          throw new BackendTimeoutError();
        }
        recordActiveBrowserAPIOutcome(telemetryOperation, "canceled");
        throw new BackendCancelledError();
      }
      recordActiveBrowserAPIOutcome(telemetryOperation, "failed");
      throw error;
    } finally {
      if (timeoutHandle) clearTimeout(timeoutHandle);
    }

    const requestId = response.headers.get("x-request-id");
    const contentType = response.headers.get("content-type")?.toLowerCase() ?? "";
    if (!contentType.includes("application/json") && !contentType.includes("application/problem+json")) {
      recordActiveBrowserAPIOutcome(telemetryOperation, "failed");
      throw new BackendProtocolError(
        `Backend response ${response.status} did not use a JSON content type.`,
        requestId,
      );
    }

    let body: unknown;
    try {
      body = await response.json();
    } catch {
      recordActiveBrowserAPIOutcome(telemetryOperation, "failed");
      throw new BackendProtocolError("Backend response contained invalid JSON.", requestId);
    }
    if (!response.ok) {
      recordActiveBrowserAPIOutcome(telemetryOperation, "failed");
      const problem = parseProblem(body, response.status);
      const correlatedRequestId = problem?.requestId ?? requestId;
      if (response.status === 401) {
        const error = new BackendAuthenticationError(problem, correlatedRequestId);
        if (!authenticationLostNotified) {
          authenticationLostNotified = true;
          dependencies.onAuthenticationLost?.(error);
        }
        throw error;
      }
      if (response.status === 403) throw new BackendAuthorizationError(problem, correlatedRequestId);
      throw new BackendHttpError(
        problem?.title ?? `Backend request failed with status ${response.status}`,
        response.status,
        problem?.code ?? null,
        correlatedRequestId,
        problem,
      );
    }
    recordActiveBrowserAPIOutcome(telemetryOperation, "succeeded");
    return body as T;
  }

  const backend: Backend = {
    mode: "http",
    assignments: {
      list: async (input, options) =>
        mapAssignments(
          await request<Schemas["ListAssignmentsOutput"]>(
            appendQuery("/v1/assignments", input),
            {},
            options,
          ),
        ),
    },
    inspections: {
      getPackage: async ({ packageId }, options) =>
        mapInspectionPackage(
          await request<Schemas["InspectionPackage"]>(
            `/v1/inspection-packages/${encodeURIComponent(packageId)}`,
            {},
            options,
          ),
        ),
      checkout: async (input, options) =>
        mapCheckout(
          await request<Schemas["CheckoutInspectionPackageOutput"]>(
            `/v1/inspection-packages/${encodeURIComponent(input.packageId)}/checkout`,
            { method: "POST", body: input },
            options,
          ),
        ),
      upsertChecklistResponse: async (input, options) =>
        mapChecklistResponse(
          await request<Schemas["ChecklistResponseView"]>(
            `/v1/checklist-responses/${encodeURIComponent(input.responseId)}`,
            { method: "PUT", body: input },
            options,
          ),
        ),
      submitChecklist: async (input, options) =>
        mapSubmitChecklist(
          await request<Schemas["SubmitChecklistOutput"]>(
            `/v1/checklists/${encodeURIComponent(input.auditId)}/submit`,
            { method: "POST", body: input },
            options,
          ),
        ),
      reopenChecklist: async (input, options) =>
        mapSubmitChecklist(
          await request<Schemas["SubmitChecklistOutput"]>(
            `/v1/checklists/${encodeURIComponent(input.auditId)}/reopen`,
            { method: "POST", body: input },
            options,
          ),
        ),
    },
    potentialFindings: {
      list: async (input, options) =>
        mapPotentialFindings(
          await request<Schemas["ListPotentialFindingsOutput"]>(
            appendQuery("/v1/potential-findings", input),
            {},
            options,
          ),
        ),
      get: async ({ potentialFindingId }, options) =>
        mapPotentialFinding(
          await request<Schemas["PotentialFindingView"]>(
            `/v1/potential-findings/${encodeURIComponent(potentialFindingId)}`,
            {},
            options,
          ),
        ),
      create: async (input, options) =>
        mapPotentialFinding(
          await request<Schemas["PotentialFindingView"]>(
            "/v1/potential-findings",
            { method: "POST", body: input },
            options,
          ),
        ),
      decide: async (input, options) =>
        mapPotentialFindingDecision(
          await request<Schemas["PotentialFindingDecisionOutput"]>(
            `/v1/potential-findings/${encodeURIComponent(input.potentialFindingId)}/decisions`,
            { method: "POST", body: input as DecidePotentialFindingInput },
            options,
          ),
        ),
    },
    findings: {
      list: async (input, options) =>
        mapFindings(
          await request<Schemas["ListFindingsOutput"]>(
            appendQuery("/v1/findings", input),
            {},
            options,
          ),
        ),
      get: async ({ findingId }, options) =>
        mapFinding(
          await request<Schemas["FindingView"]>(
            `/v1/findings/${encodeURIComponent(findingId)}`,
            {},
            options,
          ),
        ),
      authorizedClose: async (input, options) =>
        mapFinding(
          await request<Schemas["FindingView"]>(
            `/v1/findings/${encodeURIComponent(input.findingId)}/authorized-closure`,
            { method: "POST", body: input },
            options,
          ),
        ),
    },
    caps: {
      listRevisions: async ({ findingId }, options) =>
        mapCapRevisions(
          await request<Schemas["ListCapRevisionsOutput"]>(
            `/v1/findings/${encodeURIComponent(findingId)}/cap-revisions`,
            {},
            options,
          ),
        ),
      getRevision: async ({ capRevisionId }, options) =>
        mapCapRevision(
          await request<Schemas["CapRevisionView"]>(
            `/v1/cap-revisions/${encodeURIComponent(capRevisionId)}`,
            {},
            options,
          ),
        ),
      submit: async (input, options) =>
        mapSubmitCap(
          await request<Schemas["SubmitCapOutput"]>(
            "/v1/caps",
            { method: "POST", body: input },
            options,
          ),
        ),
      review: async (input, options) =>
        mapReviewCap(
          await request<Schemas["ReviewCapOutput"]>(
            `/v1/caps/${encodeURIComponent(input.capRevisionId)}/reviews`,
            { method: "POST", body: input },
            options,
          ),
        ),
    },
    inspectionAttachments: {
      beginUpload: async (input, options) =>
        await request<Schemas["BeginInspectionAttachmentUploadOutput"]>(
          `/v1/inspection-attachments/${encodeURIComponent(input.inspectionAttachmentId)}/uploads`,
          { method: "POST", body: input },
          options,
        ),
      completeUpload: async (input, options) =>
        mapCompleteInspectionAttachment(
          await request<Schemas["CompleteInspectionAttachmentUploadOutput"]>(
            `/v1/inspection-attachments/uploads/${encodeURIComponent(input.uploadId)}/complete`,
            { method: "POST", body: input },
            options,
          ),
        ),
    },
    evidence: {
      beginUpload: async (input, options) =>
        await request<Schemas["BeginEvidenceUploadOutput"]>(
          "/v1/evidence/uploads",
          { method: "POST", body: input },
          options,
        ),
      completeUpload: async (input, options) =>
        mapCompleteEvidence(
          await request<Schemas["CompleteEvidenceUploadOutput"]>(
            `/v1/evidence/uploads/${encodeURIComponent(input.uploadId)}/complete`,
            { method: "POST", body: input },
            options,
          ),
        ),
      listVersions: async ({ findingId }, options) => {
        const output = await request<Schemas["ListEvidenceVersionsOutput"]>(
          `/v1/findings/${encodeURIComponent(findingId)}/evidence`,
          {},
          options,
        );
        return output.items.map(mapEvidenceVersion);
      },
      review: async (input, options) =>
        mapReviewEvidence(
          await request<Schemas["ReviewEvidenceOutput"]>(
            `/v1/evidence/${encodeURIComponent(input.evidenceVersionId)}/reviews`,
            { method: "POST", body: input },
            options,
          ),
        ),
    },
    reports: {
      getVersion: async ({ reportVersionId }, options) =>
        mapReportVersion(
          await request<Schemas["ReportVersionView"]>(
            `/v1/report-versions/${encodeURIComponent(reportVersionId)}`,
            {},
            options,
          ),
        ),
      decide: async (input, options) =>
        mapReportVersion(
          await request<Schemas["ReportVersionView"]>(
            `/v1/report-versions/${encodeURIComponent(input.reportVersionId)}/decisions`,
            { method: "POST", body: input },
            options,
          ),
        ),
    },
    documents: {
      list: async (input, options) =>
        mapDocuments(
          await request<Schemas["ListDocumentsOutput"]>(
            appendQuery("/v1/documents", input),
            {},
            options,
          ),
        ),
      open: async ({ documentId }, options) =>
        mapDocumentMetadata(
          await request<Schemas["DocumentMetadataView"]>(
            `/v1/documents/${encodeURIComponent(documentId)}`,
            {},
            options,
          ),
        ),
    },
    auditeeReports: {
      listReleased: async (input, options) =>
        mapAuditeeReleasedReports(
          await request<Schemas["AuditeeReleasedReportPage"]>(
            appendQuery("/v1/auditee/report-versions", input),
            {},
            options,
          ),
        ),
      getReleased: async ({ reportVersionId }, options) =>
        mapAuditeeReleasedReport(
          await request<Schemas["AuditeeReleasedReportView"]>(
            `/v1/auditee/report-versions/${encodeURIComponent(reportVersionId)}`,
            {},
            options,
          ),
        ),
    },
    dashboards: {
      getManagerProjection: async (input, options) =>
        mapManagerDashboard(
          await request<Schemas["ManagerDashboardProjection"]>(
            appendQuery("/v1/dashboards/manager", input),
            {},
            options,
          ),
        ),
    },
    organizations: {
      list: async (input, options) =>
        mapOrganizations(
          await request<Schemas["ListOrganizationsOutput"]>(
            appendQuery("/v1/organizations", input),
            {},
            options,
          ),
        ),
    },
    planning: {
      list: async (input, options) =>
        mapPlanningItems(
          await request<Schemas["ListPlanningItemsOutput"]>(
            appendQuery("/v1/planning/items", input),
            {},
            options,
          ),
        ),
      decide: async (input, options) =>
        mapPlanningItem(
          await request<Schemas["PlanningItemView"]>(
            `/v1/planning/items/${encodeURIComponent(input.planningItemId)}/decisions`,
            { method: "POST", body: input },
            options,
          ),
        ),
    },
    planningIntake: {
      getDraft: async ({ draftId }, options) =>
        mapPlanningIntakeDraft(
          await request<Schemas["PlanningIntakeDraftView"]>(
            `/v1/planning/intake-drafts/${encodeURIComponent(draftId)}`,
            {},
            options,
          ),
        ),
      saveDraft: async (input, options) =>
        mapPlanningIntakeDraft(
          await request<Schemas["PlanningIntakeDraftView"]>(
            `/v1/planning/intake-drafts/${encodeURIComponent(input.draftId)}`,
            {
              method: "PUT",
              body: revisionCommandBody(input),
              headers: revisionCommandHeaders(input),
            },
            options,
          ),
        ),
      submit: async (input, options) =>
        mapSubmitPlanningIntake(
          await request<Schemas["SubmitPlanningIntakeOutput"]>(
            `/v1/planning/intake-drafts/${encodeURIComponent(input.draftId)}/submissions`,
            {
              method: "POST",
              body: revisionCommandBody(input),
              headers: revisionCommandHeaders(input),
            },
            options,
          ),
        ),
    },
    packageDrafts: {
      get: async ({ packageDraftId }, options) =>
        mapInspectionPackageDraft(
          await request<Schemas["InspectionPackageDraftView"]>(
            `/v1/inspection-package-drafts/${encodeURIComponent(packageDraftId)}`,
            {},
            options,
          ),
        ),
      save: async (input, options) =>
        mapInspectionPackageDraft(
          await request<Schemas["InspectionPackageDraftView"]>(
            `/v1/inspection-package-drafts/${encodeURIComponent(input.packageDraftId)}`,
            {
              method: "PUT",
              body: revisionCommandBody(input),
              headers: revisionCommandHeaders(input),
            },
            options,
          ),
        ),
    },
    profiles: {
      getMine: async (_input, options) =>
        mapProfile(
          await request<Schemas["ProfileView"]>(
            "/v1/profile",
            {},
            options,
          ),
        ),
      updateMine: async (input, options) =>
        mapProfile(
          await request<Schemas["ProfileView"]>(
            "/v1/profile",
            {
              method: "PUT",
              body: revisionCommandBody(input),
              headers: revisionCommandHeaders(input),
            },
            options,
          ),
        ),
    },
    teams: {
      list: async (input, options) => {
        const output = await request<Schemas["ListTeamMembersOutput"]>(
          appendQuery("/v1/team-members", input),
          {},
          options,
        );
        return { items: output.items.map(mapTeamMember), nextCursor: output.nextCursor };
      },
      openMember: async ({ subjectId }, options) =>
        mapTeamMember(
          await request<Schemas["TeamMemberView"]>(
            `/v1/team-members/${encodeURIComponent(subjectId)}`,
            {},
            options,
          ),
        ),
      listAuditTeams: async (input, options) => {
        const output = await request<Schemas["ListInspectionTeamAuditsOutput"]>(
          appendQuery("/v1/audit-teams", input),
          {},
          options,
        );
        return {
          items: output.items.map(mapInspectionTeamAudit),
          nextCursor: output.nextCursor,
        };
      },
      openAuditTeam: async ({ auditId }, options) =>
        mapInspectionTeamAudit(
          await request<Schemas["InspectionTeamAuditView"]>(
            `/v1/audit-teams/${encodeURIComponent(auditId)}`,
            {},
            options,
          ),
        ),
    },
    risk: {
      getOverview: async (input, options) =>
        mapRiskOverview(
          await request<Schemas["RiskOverviewView"]>(
            appendQuery("/v1/risk/overview", input),
            {},
            options,
          ),
        ),
      getManagementProjection: async (_input, options) =>
        mapRiskManagementProjection(
          await request<Schemas["RiskManagementProjectionView"]>(
            "/v1/risk/management",
            {},
            options,
          ),
        ),
      openFinding: async ({ findingId }, options) =>
        mapFinding(
          await request<Schemas["FindingView"]>(
            `/v1/findings/${encodeURIComponent(findingId)}`,
            {},
            options,
          ),
        ),
    },
    administration: {
      getScreenProjection: async ({ screenId }, options) =>
        mapAdministrationScreenProjection(
          await request<Schemas["AdministrationScreenProjection"]>(
            `/v1/administration/screens/${encodeURIComponent(screenId)}`,
            {},
            options,
          ),
        ),
      listScreenProjections: async (_input, options) => {
        const output = await request<Schemas["AdministrationScreenProjectionList"]>(
          "/v1/administration/screens",
          {},
          options,
        );
        return output.map(mapAdministrationScreenProjection);
      },
      invokeVisibleAction: async ({ screenId, actionId }, options) => {
        const idempotencyKey = stableCommandKey("administration-action", [
          screenId,
          actionId,
        ]);
        return mapVisibleActionResult(
          await request<Schemas["VisibleActionResult"]>(
            `/v1/administration/screens/${encodeURIComponent(screenId)}/actions/${encodeURIComponent(actionId)}`,
            {
              method: "POST",
              body: {
                operationId: idempotencyKey,
                expectedRevision: null,
                idempotencyKey,
                screenId,
                actionId,
              },
              headers: revisionCommandHeaders({
                idempotencyKey,
                expectedRevision: null,
              }),
            },
            options,
          ),
        );
      },
    },
    auditeeCoordination: {
      list: async (_input, options) => {
        const output = await request<Schemas["AuditeeCoordinationPage"]>(
          "/v1/auditee/coordination",
          {},
          options,
        );
        return {
          items: output.items.map(mapAuditeeCoordination),
          nextCursor: output.nextCursor,
        };
      },
      respond: async (input, options) =>
        mapAuditeeCoordination(
          await request<Schemas["AuditeeCoordinationView"]>(
            `/v1/auditee/coordination/${encodeURIComponent(input.auditId)}/responses`,
            {
              method: "POST",
              body: revisionCommandBody(input),
              headers: revisionCommandHeaders(input),
            },
            options,
          ),
        ),
    },
    configuration: {
      listChecklistTemplateVersions: async (input, options) =>
        mapChecklistTemplateVersions(
          await request<Schemas["ListChecklistTemplateVersionsOutput"]>(
            appendQuery("/v1/configuration/checklist-template-versions", input),
            {},
            options,
          ),
        ),
      getChecklistTemplateVersion: async ({ templateVersionId }, options) =>
        mapChecklistTemplateVersionDetail(
          await request<Schemas["ChecklistTemplateVersionDetailView"]>(
            `/v1/configuration/checklist-template-versions/${encodeURIComponent(templateVersionId)}`,
            {},
            options,
          ),
        ),
      listReminderRules: async (input, options) =>
        mapReminderRules(
          await request<Schemas["ListReminderRulesOutput"]>(
            appendQuery("/v1/configuration/reminder-rules", input),
            {},
            options,
          ),
        ),
    },
    communications: {
      list: async (input, options) =>
        mapCommunications(
          await request<Schemas["ListCommunicationsOutput"]>(
            appendQuery("/v1/communications", input),
            {},
            options,
          ),
        ),
      send: async (input, options) =>
        mapCommunication(
          await request<Schemas["CommunicationView"]>(
            "/v1/communications",
            {
              method: "POST",
              body: revisionCommandBody(input),
              headers: revisionCommandHeaders(input),
            },
            options,
          ),
        ),
    },
    calendar: {
      list: async (input, options) =>
        mapCalendarItems(
          await request<Schemas["ListCalendarItemsOutput"]>(
            appendQuery("/v1/calendar-items", input),
            {},
            options,
          ),
        ),
      openItem: async ({ calendarItemId }, options) =>
        mapCalendarItem(
          await request<Schemas["CalendarItemView"]>(
            `/v1/calendar-items/${encodeURIComponent(calendarItemId)}`,
            {},
            options,
          ),
        ),
    },
    notifications: {
      list: async (_input, options) =>
        mapNotifications(
          await request<Schemas["ListNotificationsOutput"]>(
            "/v1/notifications",
            {},
            options,
          ),
        ),
      markRead: async (input, options) =>
        mapNotification(
          await request<Schemas["NotificationView"]>(
            `/v1/notifications/${encodeURIComponent(input.notificationId)}/read`,
            {
              method: "POST",
              body: revisionCommandBody(input),
              headers: revisionCommandHeaders(input),
            },
            options,
          ),
        ),
    },
    auditTrail: {
      list: async (input, options) =>
        mapAuditEvents(
          await request<Schemas["ListAuditEventsOutput"]>(
            appendQuery("/v1/audit-events", input),
            {},
            options,
          ),
        ),
    },
    adminWorkspace: {
      listRegulatoryReferences: async (input, options) => {
        const output = await request<Schemas["AdminRegulatoryReferencePage"]>(
          appendQuery("/v1/admin/regulatory-references", input),
          {},
          options,
        );
        return {
          items: output.items.map(mapAdminRegulatoryReference),
          nextCursor: output.nextCursor,
        };
      },
      listTemplateMasters: async (_input, options) => {
        const output = await request<Schemas["AdminTemplateMasterPage"]>(
          "/v1/admin/templates",
          {},
          options,
        );
        return {
          items: output.items.map(mapAdminTemplateMaster),
          nextCursor: output.nextCursor,
        };
      },
      listQuestions: async (input, options) => {
        const output = await request<Schemas["AdminQuestionPage"]>(
          appendQuery("/v1/admin/questions", input),
          {},
          options,
        );
        return {
          items: output.items.map(mapAdminQuestion),
          nextCursor: output.nextCursor,
        };
      },
      createQuestion: async (input, options) =>
        mapAdminQuestion(
          await request<Schemas["AdminQuestionView"]>(
            "/v1/admin/questions",
            {
              method: "POST",
              body: revisionCommandBody(input),
              headers: revisionCommandHeaders(input),
            },
            options,
          ),
        ),
      getTemplate: async ({ templateId }, options) =>
        mapAdminTemplate(
          await request<Schemas["AdminTemplateView"]>(
            `/v1/admin/templates/${encodeURIComponent(templateId)}`,
            {},
            options,
          ),
        ),
      createDraft: async (input, options) =>
        mapAdminTemplateVersion(
          await request<Schemas["AdminTemplateVersionView"]>(
            `/v1/admin/templates/${encodeURIComponent(input.templateId)}/drafts`,
            {
              method: "POST",
              body: revisionCommandBody(input),
              headers: revisionCommandHeaders(input),
            },
            options,
          ),
        ),
      addDraftQuestion: async (input, options) =>
        mapAdminTemplateVersion(
          await request<Schemas["AdminTemplateVersionView"]>(
            `/v1/admin/templates/${encodeURIComponent(input.templateId)}/drafts/${encodeURIComponent(input.draftVersionId)}/questions`,
            {
              method: "POST",
              body: revisionCommandBody(input),
              headers: revisionCommandHeaders(input),
            },
            options,
          ),
        ),
      moveDraftQuestion: async (input, options) =>
        mapAdminTemplateVersion(
          await request<Schemas["AdminTemplateVersionView"]>(
            `/v1/admin/templates/${encodeURIComponent(input.templateId)}/drafts/${encodeURIComponent(input.draftVersionId)}/questions/${encodeURIComponent(input.questionId)}/moves`,
            {
              method: "POST",
              body: revisionCommandBody(input),
              headers: revisionCommandHeaders(input),
            },
            options,
          ),
        ),
      getInspectionPackage: async ({ packageId }, options) =>
        mapAdminInspectionPackage(
          await request<Schemas["AdminInspectionPackageView"]>(
            `/v1/admin/inspection-packages/${encodeURIComponent(packageId)}`,
            {},
            options,
          ),
        ),
      listReportDefinitions: async (input, options) => {
        const output = await request<Schemas["AdminReportDefinitionPage"]>(
          appendQuery("/v1/admin/report-definitions", input),
          {},
          options,
        );
        return {
          items: output.items.map(mapAdminReportDefinition),
          nextCursor: output.nextCursor,
        };
      },
      listAccessDirectory: async (input, options) => {
        const output = await request<Schemas["AdminAccessDirectoryPage"]>(
          appendQuery("/v1/admin/access-directory", input),
          {},
          options,
        );
        return {
          items: output.items.map(mapAdminAccessDirectoryEntry),
          nextCursor: output.nextCursor,
        };
      },
      requestUserLifecycle: async (input, options) =>
        mapUserLifecycleRequest(
          await request<Schemas["UserLifecycleRequestView"]>(
            "/v1/admin/user-lifecycle-requests",
            {
              method: "POST",
              headers: { "Idempotency-Key": input.idempotencyKey },
              body: {
                operationId: input.idempotencyKey,
                idempotencyKey: input.idempotencyKey,
                subjectId: input.subjectId ?? null,
                action: input.action,
                roles: input.roles,
                organizationId: input.organizationId,
                email: input.email ?? null,
                displayName: input.displayName ?? null,
                reason: input.reason,
                expectedMembershipRevision: input.expectedMembershipRevision,
                effectiveAt: input.effectiveAt ?? null,
              } satisfies Schemas["RequestUserLifecycleInput"],
            },
            options,
          ),
        ),
      getUserLifecycleRequest: async ({ requestId }, options) =>
        mapUserLifecycleRequest(
          await request<Schemas["UserLifecycleRequestView"]>(
            `/v1/admin/user-lifecycle-requests/${encodeURIComponent(requestId)}`,
            {},
            options,
          ),
        ),
      listOrganizations: async (input, options) => {
        const output = await request<Schemas["AdminOrganizationPage"]>(
          appendQuery("/v1/admin/organizations", input),
          {},
          options,
        );
        return {
          items: output.items.map(mapAdminOrganization),
          nextCursor: output.nextCursor,
        };
      },
      getOrganization: async ({ organizationId }, options) =>
        mapAdminOrganization(
          await request<Schemas["AdminOrganizationView"]>(
            `/v1/admin/organizations/${encodeURIComponent(organizationId)}`,
            {},
            options,
          ),
        ),
      listAuditEvents: async (input, options) =>
        mapAuditEvents(
          await request<Schemas["ListAuditEventsOutput"]>(
            appendQuery("/v1/admin/audit-events", input),
            {},
            options,
          ),
        ),
    },
    assistantDrafts: {
      getGuidance: async (_input, options) => {
        const output = await request<Schemas["AssistantGuidance"]>(
          "/v1/assistant/guidance",
          {},
          options,
        );
        return {
          advisoryOnly: true,
          prohibitedActions: [...output.prohibitedActions],
        };
      },
      createDraft: async (input, options) => {
        const idempotencyKey = stableCommandKey("assistant-draft", [
          input.findingId,
          input.prompt.trim(),
        ]);
        return mapAssistantDraft(
          await request<Schemas["AssistantDraftView"]>(
            "/v1/assistant/drafts",
            {
              method: "POST",
              body: {
                operationId: idempotencyKey,
                expectedRevision: null,
                idempotencyKey,
                findingId: input.findingId,
                prompt: input.prompt,
              },
              headers: revisionCommandHeaders({
                idempotencyKey,
                expectedRevision: null,
              }),
            },
            options,
          ),
        );
      },
    },
    sync: {
      pushOperation: async (input, options) =>
        mapPushResult(
          await request<Schemas["PushFieldOperationResult"]>(
            "/v1/sync/operations",
            { method: "POST", body: { operation: input.operation as FieldSyncOperation } },
            options,
          ),
        ),
      pull: async (input, options) =>
        mapSyncPull(
          await request<Schemas["SyncPullResponse"]>(
            appendQuery("/v1/sync/changes", input),
            {},
            options,
          ),
        ),
    },
  };
  return backend;
}
