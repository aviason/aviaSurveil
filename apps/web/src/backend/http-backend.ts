import type { components } from "../generated/transport/api-types";
import type { PublicHttpConfig } from "../app/public-http-config";
import type {
  Backend,
  BackendRequestOptions,
  DecidePotentialFindingInput,
  FieldSyncOperation,
  GovernedValidationIssue,
  GovernedChecklistIntakeBackend,
  CreateChecklistImportBatchReceiptInput,
  CreateChecklistImportFileExtractionReviewInput,
  ResolveChecklistImportFileIdentityInput,
  CreateExistingChecklistCandidateInput,
  GovernedSourceAuthorityAttestationInput,
  GovernedChecklistReviewCommentInput,
  GovernedSourceMappingAttestationInput,
  GovernedAuditPackageEligibilityInput,
  CanonicalQuestionCatalogEntry,
  CanonicalQuestionCatalogPage,
  CanonicalAuditScopeOptionPage,
  CanonicalSelectionPreview,
  CanonicalSelectionReceipt,
  CanonicalAuditWorkflowBackend,
} from "./backend";
import { GovernedValidationError } from "./backend-contracts";
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

function mapCanonicalCatalogEntry(value: Schemas["CanonicalQuestionCatalogEntry"]): CanonicalQuestionCatalogEntry {
  return {
    catalogVersion: value.catalogVersion,
    usageClass: value.usageClass,
    questionVersionId: value.questionVersionId,
    formCode: value.formCode,
    proposalId: value.proposalId,
    ordinal: value.ordinal,
    questionDigest: value.questionDigest,
    prompt: value.prompt ?? null,
    configuredReference: value.configuredReference ?? null,
    expectedEvidence: value.expectedEvidence ?? null,
    sourceLocator: value.sourceLocator ?? null,
    sourceGapState: value.sourceGapState,
    proposedDomain: value.proposedDomain ?? null,
    proposedTopic: value.proposedTopic ?? null,
    proposedRiskBand: value.proposedRiskBand ?? null,
    canSelect: value.canSelect,
    canPublish: value.canPublish,
    governedCandidateId: value.governedCandidateId ?? null,
    governedCandidateRevision: value.governedCandidateRevision ?? null,
    governedCandidateContentDigest: value.governedCandidateContentDigest ?? null,
    governedCandidateStatus: value.governedCandidateStatus ?? null,
    reviewRevision: value.reviewRevision ?? 0,
    reviewDisposition: value.reviewDisposition ?? null,
    reviewDigest: value.reviewDigest ?? null,
    reviewHistory: value.reviewHistory ?? [],
  };
}

function mapCanonicalCatalogPage(value: Schemas["CanonicalQuestionCatalogPage"]): CanonicalQuestionCatalogPage {
  return { ...value, items: value.items.map(mapCanonicalCatalogEntry) };
}

function mapCanonicalScopeOptionPage(value: Schemas["CanonicalAuditScopeOptionPage"]): CanonicalAuditScopeOptionPage {
  return { ...value, items: value.items.map((item) => ({ ...item, inspectionTypes: [...item.inspectionTypes] })) };
}

function mapCanonicalSelectionPreview(value: Schemas["CanonicalAuditScopeSelectionPreview"]): CanonicalSelectionPreview {
  return value;
}

function mapCanonicalSelectionReceipt(value: Schemas["CanonicalAuditScopeSelectionReceipt"]): CanonicalSelectionReceipt {
  return value;
}

export interface BackendProblem {
  type: string;
  title: string;
  status: number;
  detail: string | null;
  code: string | null;
  requestId: string | null;
  issues: GovernedValidationIssue[];
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
  /** Candidate package data is never browser-cached or telemetry-correlated. */
  cache?: RequestCache;
  suppressTelemetry?: boolean;
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
    issues: Array.isArray(candidate.issues)
      ? candidate.issues.filter((issue): issue is GovernedValidationIssue =>
        Boolean(issue) &&
        typeof issue === "object" &&
        typeof (issue as Record<string, unknown>).fieldPath === "string" &&
        typeof (issue as Record<string, unknown>).code === "string" &&
        typeof (issue as Record<string, unknown>).message === "string" &&
        typeof (issue as Record<string, unknown>).sourceIdentity === "string" &&
        typeof (issue as Record<string, unknown>).sourceHash === "string" &&
        typeof (issue as Record<string, unknown>).clauseId === "string" &&
        typeof (issue as Record<string, unknown>).locator === "string")
      : [],
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
    const recordOutcome = (outcome: "succeeded" | "failed" | "canceled") => {
      if (!requestInput.suppressTelemetry) recordActiveBrowserAPIOutcome(telemetryOperation, outcome);
    };
    const headers = new Headers({ Accept: "application/json" });
    for (const [name, value] of Object.entries(requestInput.headers ?? {})) {
      headers.set(name, value);
    }
    for (const [name, value] of Object.entries(activeBrowserRequestHeaders())) {
      headers.set(name, value);
    }
    const multipart = typeof FormData !== "undefined" && requestInput.body instanceof FormData;
    if (requestInput.body !== undefined && !multipart) {
      headers.set("content-type", "application/json");
      const token = csrfToken();
      if (token) headers.set("x-csrf-token", token);
    }
    if (multipart) {
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
        cache: requestInput.cache,
        headers,
        body: requestInput.body === undefined ? undefined : multipart ? requestInput.body as FormData : JSON.stringify(requestInput.body),
        signal,
      });
    } catch (error) {
      if (error instanceof DOMException && error.name === "AbortError") {
        if (timeoutController?.signal.aborted && !options.signal?.aborted) {
          recordOutcome("failed");
          throw new BackendTimeoutError();
        }
        recordOutcome("canceled");
        throw new BackendCancelledError();
      }
      recordOutcome("failed");
      throw error;
    } finally {
      if (timeoutHandle) clearTimeout(timeoutHandle);
    }

    const requestId = response.headers.get("x-request-id");
    const contentType = response.headers.get("content-type")?.toLowerCase() ?? "";
    if (!contentType.includes("application/json") && !contentType.includes("application/problem+json")) {
      recordOutcome("failed");
      if (!response.ok) {
        if (response.status === 401) {
          const error = new BackendAuthenticationError(null, requestId);
          if (!authenticationLostNotified) {
            authenticationLostNotified = true;
            dependencies.onAuthenticationLost?.(error);
          }
          throw error;
        }
        if (response.status === 403) {
          throw new BackendAuthorizationError(null, requestId);
        }
        throw new BackendHttpError(
          `Backend request failed with status ${response.status}`,
          response.status,
          null,
          requestId,
          null,
        );
      }
      throw new BackendProtocolError(
        `Backend response ${response.status} did not use a JSON content type.`,
        requestId,
      );
    }

    let body: unknown;
    try {
      body = await response.json();
    } catch {
      recordOutcome("failed");
      throw new BackendProtocolError("Backend response contained invalid JSON.", requestId);
    }
    if (!response.ok) {
      recordOutcome("failed");
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
      if (response.status === 422 && problem?.issues.length) {
        throw new GovernedValidationError(problem.issues);
      }
      throw new BackendHttpError(
        problem?.title ?? `Backend request failed with status ${response.status}`,
        response.status,
        problem?.code ?? null,
        correlatedRequestId,
        problem,
      );
    }
    recordOutcome("succeeded");
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
      start: async (input, options) =>
        await request<Schemas["StartInspectionOutput"]>(
          `/v1/audits/${encodeURIComponent(input.auditId)}/start`,
          {
            method: "POST",
            body: { operationId: input.operationId, expectedInspectionRevision: input.expectedInspectionRevision },
            headers: revisionCommandHeaders({
              idempotencyKey: input.operationId,
              expectedRevision: input.expectedInspectionRevision,
            }),
          },
          options,
        ),
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
      create: async (input, options) =>
        mapReportVersion(
          await request<Schemas["ReportVersionView"]>(
            "/v1/report-versions",
            {
              method: "POST",
              body: input,
              headers: revisionCommandHeaders({ idempotencyKey: input.idempotencyKey, expectedRevision: null }),
            },
            options,
          ),
        ),
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
      createDraft: async (input, options) =>
        mapPlanningIntakeDraft(
          await request<Schemas["PlanningIntakeDraftView"]>(
            "/v1/planning/intake-drafts",
            {
              method: "POST",
              body: {
                operationId: input.operationId,
                idempotencyKey: input.idempotencyKey,
                expectedRevision: input.expectedRevision ?? null,
                ...(input.draftId ? { draftId: input.draftId } : {}),
                values: input.values,
              },
              headers: revisionCommandHeaders({
                idempotencyKey: input.idempotencyKey,
                expectedRevision: null,
              }),
            },
            options,
          ),
        ),
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
    canonicalCatalog: {
      listScopeOptions: async (input, options) => mapCanonicalScopeOptionPage(await request<Schemas["CanonicalAuditScopeOptionPage"]>(
        appendQuery("/v1/audit-scope-options", {
          cursor: input?.cursor,
          limit: input?.limit,
          catalogVersion: input?.catalogVersion,
          usageClass: input?.usageClass,
          review: input?.forReview ? "true" : undefined,
        }), {}, options)),
      listCatalog: async (input, options) => mapCanonicalCatalogPage(await request<Schemas["CanonicalQuestionCatalogPage"]>(
        appendQuery(`/v1/question-catalogs/${encodeURIComponent(input.catalogVersion)}/questions`, {
          usageClass: input.usageClass, search: input.search, formCode: input.formCode,
          domain: input.domain, topic: input.topic, riskBand: input.riskBand,
          sourceGapState: input.sourceGapState, selected: input.selected, scopeId: input.scopeId,
          cursor: input.cursor, limit: input.limit,
        }), {}, options)),
      getQuestion: async (input, options) => mapCanonicalCatalogEntry(await request<Schemas["CanonicalQuestionCatalogEntry"]>(
        appendQuery(`/v1/question-catalogs/${encodeURIComponent(input.catalogVersion)}/questions/${encodeURIComponent(input.questionVersionId)}`, { usageClass: input.usageClass, scopeId: input.scopeId }), {}, options)),
      previewSelection: async (input, options) => {
        const { scopeId, ...body } = input;
        return mapCanonicalSelectionPreview(await request<Schemas["CanonicalAuditScopeSelectionPreview"]>(
          `/v1/audit-scopes/${encodeURIComponent(scopeId)}/preview`, { method: "POST", body }, options));
      },
      commitSelection: async (input, options) => {
        const { scopeId, ...body } = input;
        return mapCanonicalSelectionReceipt(await request<Schemas["CanonicalAuditScopeSelectionReceipt"]>(
          `/v1/audit-scopes/${encodeURIComponent(scopeId)}/selection`, { method: "PUT", body, headers: revisionCommandHeaders({ idempotencyKey: input.idempotencyKey ?? input.operationId, expectedRevision: null }) }, options));
      },
    },
    canonicalAuditWorkflow: {
      getPreparation: async (input, options) => request<Schemas["CanonicalAssignmentView"]>(
        appendQuery("/v1/audit-assignments/preparations/current", { assignmentId: input?.assignmentId, planningItemId: input?.planningItemId }),
        { cache: "no-store" },
        options,
      ),
      prepare: async (planningItemId, input, options) => request<Schemas["PreparationView"]>(
        `/v1/planning/items/${encodeURIComponent(planningItemId)}/preparations`,
        { method: "POST", body: input, headers: revisionCommandHeaders({ idempotencyKey: input.idempotencyKey, expectedRevision: input.expectedPlanningRevision }) },
        options,
      ),
      assignLead: async (assignmentId, input, options) => request<Schemas["CanonicalAssignmentView"]>(
        `/v1/audit-assignments/${encodeURIComponent(assignmentId)}/lead`,
        { method: "POST", body: input, headers: revisionCommandHeaders({ idempotencyKey: input.idempotencyKey, expectedRevision: input.expectedInspectionRevision }) },
        options,
      ),
      previewTeam: async (assignmentId, input, options) => request<Schemas["PreparationEditPreviewView"]>(
        `/v1/audit-assignments/${encodeURIComponent(assignmentId)}/team-previews`,
        { method: "POST", body: input, headers: revisionCommandHeaders({ idempotencyKey: input.idempotencyKey, expectedRevision: input.expectedRevision }) },
        options,
      ),
      assignTeam: async (assignmentId, input, options) => request<Schemas["CanonicalAssignmentView"]>(
        `/v1/audit-assignments/${encodeURIComponent(assignmentId)}/team`,
        { method: "POST", body: input, headers: revisionCommandHeaders({ idempotencyKey: input.idempotencyKey, expectedRevision: input.expectedRevision }) },
        options,
      ),
      previewQuestionCoverage: async (assignmentId, input, options) => request<Schemas["PreparationEditPreviewView"]>(
        `/v1/audit-assignments/${encodeURIComponent(assignmentId)}/question-coverage-previews`,
        { method: "POST", body: input, headers: revisionCommandHeaders({ idempotencyKey: input.idempotencyKey, expectedRevision: input.expectedRevision }) },
        options,
      ),
      assignQuestionCoverage: async (assignmentId, input, options) => request<Schemas["CanonicalAssignmentView"]>(
        `/v1/audit-assignments/${encodeURIComponent(assignmentId)}/question-coverage`,
        { method: "POST", body: input, headers: revisionCommandHeaders({ idempotencyKey: input.idempotencyKey, expectedRevision: input.expectedRevision }) },
        options,
      ),
      confirmPreparation: async (assignmentId, input, options) => request<Schemas["PreparationConfirmationView"]>(
        `/v1/audit-assignments/${encodeURIComponent(assignmentId)}/preparation-confirmations`,
        { method: "POST", body: input, headers: revisionCommandHeaders({ idempotencyKey: input.idempotencyKey, expectedRevision: input.expectedAssignmentRevision }) },
        options,
      ),
      materialize: async (assignmentId, input, options) => request<Schemas["CanonicalMaterializedAuditView"]>(
        `/v1/audit-assignments/${encodeURIComponent(assignmentId)}/materializations`,
        { method: "POST", body: input, headers: revisionCommandHeaders({ idempotencyKey: input.idempotencyKey, expectedRevision: input.expectedAssignmentRevision }) },
        options,
      ),
      start: async (auditId, input, options) => request<Schemas["StartInspectionOutput"]>(
        `/v1/audits/${encodeURIComponent(auditId)}/start`,
        { method: "POST", body: input, headers: revisionCommandHeaders({ idempotencyKey: input.operationId, expectedRevision: input.expectedInspectionRevision }) },
        options,
      ),
    } satisfies CanonicalAuditWorkflowBackend,
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
      review: async (input, options) =>
        mapAuditeeCoordination(
          await request<Schemas["AuditeeCoordinationView"]>(
            `/v1/auditee/coordination/${encodeURIComponent(input.auditId)}/reviews`,
            {
              method: "POST",
              body: input,
              headers: revisionCommandHeaders({ idempotencyKey: input.idempotencyKey, expectedRevision: input.expectedRevision }),
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
    governedChecklistReview: {
      validateBlockedGeneration: async (input, options) =>
        request<Schemas["GovernedBlockedGenerationResult"]>(
          "/v1/department-manager/governed-checklist/blocked-generation-validations",
          { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input },
          options,
        ),
      listQueue: async (_input, options) =>
        request<Schemas["DepartmentManagerGovernedReviewQueue"]>(
          "/v1/department-manager/governed-checklist/review-queue",
          {},
          options,
        ),
      getCandidate: async ({ candidateId }, options) =>
        request<Schemas["DepartmentManagerGovernedReviewItem"]>(
          `/v1/department-manager/governed-checklist/candidates/${encodeURIComponent(candidateId)}`,
          {},
          options,
        ),
      return: async (input, options) =>
        request<Schemas["GovernedCandidateView"]>(
          `/v1/department-manager/governed-checklist/candidates/${encodeURIComponent(input.candidateId)}/returns`,
          { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input },
          options,
        ),
      reject: async (input, options) =>
        request<Schemas["GovernedCandidateView"]>(
          `/v1/department-manager/governed-checklist/candidates/${encodeURIComponent(input.candidateId)}/rejections`,
          { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input },
          options,
        ),
      approve: async (input, options) =>
        request<Schemas["GovernedCandidateView"]>(
          `/v1/department-manager/governed-checklist/candidates/${encodeURIComponent(input.candidateId)}/technical-approvals`,
          { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input },
          options,
        ),
      publish: async (input, options) =>
        request<Schemas["GovernedPublicationView"]>(
          `/v1/department-manager/governed-checklist/candidates/${encodeURIComponent(input.candidateId)}/publications`,
          { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input },
          options,
        ),
      getPublishedVersion: async ({ templateVersionId }, options) =>
        request<Schemas["GovernedPublishedVersionView"]>(
          `/v1/department-manager/governed-checklist/published-versions/${encodeURIComponent(templateVersionId)}`,
          {},
          options,
        ),
    },
    governedChecklistIntake: {
      receiveBatch: async (input, options) => {
        const form = new FormData();
        const archive = input.archive instanceof Blob ? input.archive : new Blob([input.archive as unknown as BlobPart]);
        form.append("archive", archive, "archive.zip");
        const receipt: CreateChecklistImportBatchReceiptInput = {
          operationId: input.operationId,
          idempotencyKey: input.idempotencyKey,
          expectedArchiveSha256: input.expectedArchiveSha256,
          reason: input.reason,
        };
        form.append("receipt", new Blob([JSON.stringify(receipt)], { type: "application/json" }), "receipt.json");
        return request<Schemas["ChecklistImportBatchReceiptView"]>("/v1/admin/governed-checklist/import-batches", { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: form }, options);
      },
      getBatch: async ({ importBatchId }, options) => request<Schemas["ChecklistImportBatchView"]>(`/v1/admin/governed-checklist/import-batches/${encodeURIComponent(importBatchId)}`, {}, options),
      listFiles: async ({ importBatchId, cursor, limit }, options) => request<Schemas["ChecklistImportFilePage"]>(appendQuery(`/v1/admin/governed-checklist/import-batches/${encodeURIComponent(importBatchId)}/files`, { cursor, limit }), {}, options),
      listReceipts: async ({ importBatchId, cursor, limit }, options) => request<Schemas["ChecklistImportReceiptPage"]>(appendQuery(`/v1/admin/governed-checklist/import-batches/${encodeURIComponent(importBatchId)}/receipts`, { cursor, limit }), {}, options),
      createExtractionReview: async (input: CreateChecklistImportFileExtractionReviewInput, options) => request<Schemas["ChecklistImportExtractionReviewSummaryView"]>(`/v1/admin/governed-checklist/import-batches/${encodeURIComponent(input.importBatchId)}/files/${encodeURIComponent(input.importFileId)}/extraction-reviews`, { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input }, options),
      getExtractionReview: async ({ importBatchId, importFileId, cursor, limit }, options) => request<Schemas["ChecklistImportExtractionReviewPage"]>(appendQuery(`/v1/admin/governed-checklist/import-batches/${encodeURIComponent(importBatchId)}/files/${encodeURIComponent(importFileId)}/extraction-review`, { cursor, limit }), {}, options),
      resolveIdentity: async (input: ResolveChecklistImportFileIdentityInput, options) => request<Schemas["ChecklistImportFileView"]>(`/v1/admin/governed-checklist/import-batches/${encodeURIComponent(input.importBatchId)}/files/${encodeURIComponent(input.importFileId)}/identity-resolutions`, { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input }, options),
      importCandidate: async (input: CreateExistingChecklistCandidateInput, options) => request<Schemas["GovernedBackendCommandResult"]>(`/v1/admin/governed-checklist/import-batches/${encodeURIComponent(input.importBatchId)}/files/${encodeURIComponent(input.importFileId)}/candidate-imports`, { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input }, options),
      listSourceReviewQueue: async ({ cursor, limit }, options) => request<Schemas["GovernedSourceReviewQueuePage"]>(appendQuery("/v1/governed-checklist/source-review-queue", { cursor, limit }), {}, options),
      getSourceReviewItem: async ({ reviewItemId }, options) => request<Schemas["GovernedSourceReviewQueuePage"]["items"][number]>(`/v1/governed-checklist/source-review-items/${encodeURIComponent(reviewItemId)}`, {}, options),
      listReviewerQueue: async ({ cursor, limit }, options) => request<Schemas["GovernedReviewerQueuePage"]>(appendQuery("/v1/governed-checklist/reviewer-queue", { cursor, limit }), {}, options),
      attestSourceAuthority: async (input: GovernedSourceAuthorityAttestationInput, options) => request<Schemas["GovernedSourceAuthorityAttestationView"]>(`/v1/governed-checklist/source-versions/${encodeURIComponent(input.sourceVersionId)}/authority-attestations`, { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input }, options),
      getExistingCandidate: async ({ existingCandidateId }, options) => request<Schemas["ExistingChecklistCandidateView"]>(`/v1/governed-checklist/existing-candidates/${encodeURIComponent(existingCandidateId)}`, {}, options),
      createDraftFromExisting: async (input, options) => request<Schemas["GovernedBackendCommandResult"]>(`/v1/governed-checklist/existing-candidates/${encodeURIComponent(input.existingCandidateId)}/drafts`, { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input }, options),
      createOfficialSourceDraft: async (input, options) => request<Schemas["GovernedBackendCommandResult"]>("/v1/governed-checklist/official-source-drafts", { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input }, options),
      getDraft: async ({ candidateId }, options) => request<Schemas["GovernedCandidateDetailView"]>(`/v1/governed-checklist/candidates/${encodeURIComponent(candidateId)}`, {}, options),
      createHybridReconciliation: async (input, options) => request<Schemas["GovernedBackendCommandResult"]>(`/v1/governed-checklist/candidates/${encodeURIComponent(input.candidateId)}/hybrid-reconciliations`, { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input }, options),
      listReviewComments: async ({ candidateId, cursor, limit }, options) => request<Schemas["GovernedChecklistReviewCommentPage"]>(appendQuery(`/v1/governed-checklist/candidates/${encodeURIComponent(candidateId)}/review-comments`, { cursor, limit }), {}, options),
      createReviewComment: async (input: GovernedChecklistReviewCommentInput, options) => request<Schemas["GovernedChecklistReviewCommentView"]>(`/v1/governed-checklist/candidates/${encodeURIComponent(input.candidateId)}/review-comments`, { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input }, options),
      attestSourceMapping: async (input: GovernedSourceMappingAttestationInput, options) => request<Schemas["GovernedSourceMappingAttestationView"]>(`/v1/governed-checklist/candidates/${encodeURIComponent(input.candidateId)}/source-mapping-attestations`, { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input }, options),
      evaluateAuditPackageEligibility: async (input: GovernedAuditPackageEligibilityInput, options) => request<Schemas["GovernedAuditPackageEligibilityView"]>(`/v1/governed-checklist/published-versions/${encodeURIComponent(input.publishedVersionId)}/audit-package-eligibility-evaluations`, { method: "POST", headers: { "Idempotency-Key": input.operationId }, body: input }, options),
    },
    adminWorkspace: {
      listGovernedSources: async (_input, options) => {
        const output = await request<Schemas["GovernedSourceSnapshotPage"]>("/v1/admin/governed-checklist/sources", {}, options);
        return output as unknown as import("./backend").PageOutput<import("./backend").GovernedSourceSnapshotView>;
      },
      activateGovernedSourceCurrentness: async (input, options) => request<Schemas["GovernedSourceCurrentnessActivationView"]>("/v1/admin/governed-checklist/source-currentness-activations", { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input }, options) as Promise<import("./backend").GovernedSourceCurrentnessActivationView>,
      importGovernedGenerationRun: async (input, options) => request<Schemas["GovernedGenerationRunView"]>("/v1/admin/governed-checklist/generation-runs", { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input }, options) as Promise<import("./backend").GovernedGenerationRunView>,
      getGovernedGenerationRun: async ({ generationRunId }, options) => request<Schemas["GovernedGenerationRunView"]>(`/v1/admin/governed-checklist/generation-runs/${encodeURIComponent(generationRunId)}`, {}, options) as Promise<import("./backend").GovernedGenerationRunView>,
      getGovernedCandidate: async ({ candidateId }, options) => request<Schemas["GovernedCandidateView"]>(`/v1/admin/governed-checklist/candidates/${encodeURIComponent(candidateId)}`, {}, options) as Promise<import("./backend").GovernedCandidateView>,
      createGovernedCandidateRevision: async (input, options) => request<Schemas["GovernedCandidateView"]>(`/v1/admin/governed-checklist/candidates/${encodeURIComponent(input.candidateId)}/revisions`, { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input }, options) as Promise<import("./backend").GovernedCandidateView>,
      submitGovernedCandidateReview: async (input, options) => request<Schemas["GovernedCandidateView"]>(`/v1/admin/governed-checklist/candidates/${encodeURIComponent(input.candidateId)}/submissions`, { method: "POST", headers: { "Idempotency-Key": input.idempotencyKey }, body: input }, options) as Promise<import("./backend").GovernedCandidateView>,
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
