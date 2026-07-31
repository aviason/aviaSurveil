import { describe, expect, it, vi } from "vitest";

import {
  BackendAuthenticationError,
  BackendAuthorizationError,
  BackendCancelledError,
  BackendHttpError,
  BackendProtocolError,
  BackendTimeoutError,
  createHttpBackend,
} from "./http-backend";
import { BACKEND_CAPABILITY_KEYS } from "./backend";
import {
  activateBrowserTelemetry,
  createBrowserTelemetry,
} from "../telemetry/browser-telemetry";

function jsonResponse(value: unknown, init: ResponseInit = {}): Response {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: { "content-type": "application/json", "x-request-id": "REQ-TEST-001" },
    ...init,
  });
}

describe("HttpBackend", () => {
  it("exposes the exact complete aggregate capability registry", () => {
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation: vi.fn<typeof fetch>() },
    );

    expect(BACKEND_CAPABILITY_KEYS).toHaveLength(29);
    expect(BACKEND_CAPABILITY_KEYS.every((key) => backend[key] !== undefined)).toBe(true);
  });

  it("maps an assignment response with same-origin credentials", async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse({
        items: [
          {
            auditId: "AUD-2026-001",
            organizationId: "ORG-FLY-NAMIBIA",
            organizationName: "Fly Namibia",
            title: "2026 Cabin Inspection - Fly Namibia",
            status: "IN_PROGRESS",
            dueDate: "2026-06-18",
            dueState: "DUE_SOON",
            nextAction: "Continue Cabin Inspection checklist",
          },
        ],
        nextCursor: null,
      }),
    );
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation, csrfToken: () => "csrf-test" },
    );
    const result = await backend.assignments.list({ limit: 20 });
    expect(result.items[0]?.auditId).toBe("AUD-2026-001");
    expect(fetchImplementation).toHaveBeenCalledTimes(1);
    const [url, init] = fetchImplementation.mock.calls[0]!;
    expect(url).toBe("/v1/assignments?limit=20");
    expect(init?.credentials).toBe("same-origin");
    expect(new Headers(init?.headers).get("accept")).toBe("application/json");
  });

  it("propagates the active browser W3C trace and bounded correlation ID", async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse({ items: [], nextCursor: null }),
    );
    const telemetry = createBrowserTelemetry({
      buildProfile: "http",
      serviceVersion: "test",
      correlationID: "CORRELATION-HTTP-001",
      transport: { send: async () => undefined },
    });
    telemetry.recordNavigation("inspector-findings", "load");
    const deactivate = activateBrowserTelemetry(
      telemetry,
      () => "inspector-findings",
    );
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation },
    );

    try {
      await backend.assignments.list({ limit: 20 });
    } finally {
      deactivate();
    }

    const headers = new Headers(fetchImplementation.mock.calls[0]?.[1]?.headers);
    expect(headers.get("traceparent")).toMatch(
      /^00-[0-9a-f]{32}-[0-9a-f]{16}-01$/,
    );
    expect(headers.get("x-correlation-id")).toBe("CORRELATION-HTTP-001");
  });

  it("injects CSRF and operation ID once without hidden command replay", async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse({
        capRevisionId: "CAP-CAB-2026-001-R1",
        capRevision: 1,
        capStatus: "SUBMITTED",
        findingStatus: "CAP_SUBMITTED",
        findingRevision: 2,
      }, { status: 201 }),
    );
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation, csrfToken: () => "csrf-test" },
    );
    await backend.caps.submit({
      operationId: "OP-CAP-HTTP-001",
      findingId: "FND-CAB-2026-001",
      expectedFindingRevision: 1,
      rootCause: "Root cause",
      correctiveAction: "Corrective action",
      preventiveAction: "Preventive action",
      responsiblePerson: "Responsible person",
      targetCompletionDate: "2026-07-15",
      commentToCaa: "CAA comment",
    });
    expect(fetchImplementation).toHaveBeenCalledTimes(1);
    const [, init] = fetchImplementation.mock.calls[0]!;
    const headers = new Headers(init?.headers);
    expect(headers.get("x-csrf-token")).toBe("csrf-test");
    expect(JSON.parse(String(init?.body)).operationId).toBe("OP-CAP-HTTP-001");
  });

  it("preserves exact Potential Finding attachments and Evidence review comments", async () => {
    const fetchImplementation = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(
        jsonResponse({
          id: "PF-2026-001",
          auditId: "AUD-2026-001",
          questionId: "CAB-EMEQ-PBE-001",
          organizationId: "ORG-FLY-NAMIBIA",
          title: "PBE record gap",
          description: "Configured PBE evidence was unavailable.",
          status: "PENDING_LEAD_REVIEW",
          revision: 1,
          convertedFindingId: null,
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          reviewDecisionId: "EVD-REVIEW-0001",
          evidenceVersionId: "EVD-CAB-2026-001-V2",
          evidenceVersionRevision: 3,
          findingStatus: "CLOSED",
          findingRevision: 8,
        }),
      );
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation, csrfToken: () => "csrf-test" },
    );

    await backend.potentialFindings.create({
      operationId: "OP-PF-HTTP-ATTACHMENTS",
      auditId: "AUD-2026-001",
      questionId: "CAB-EMEQ-PBE-001",
      checklistResponseId: "RESP-CAB-EMEQ-PBE-001",
      expectedChecklistResponseRevision: 2,
      title: "PBE record gap",
      description: "Configured PBE evidence was unavailable.",
      requiredComment: "Provide the exact configured record.",
      inspectionAttachmentIds: ["ATT-PBE-001", "ATT-PBE-002"],
    });
    await backend.evidence.review({
      operationId: "OP-EVIDENCE-HTTP-CLOSE",
      evidenceVersionId: "EVD-CAB-2026-001-V2",
      expectedEvidenceVersionRevision: 2,
      findingId: "FND-CAB-2026-001",
      expectedFindingRevision: 7,
      decision: "CLOSE",
      commentToAuditee: "Evidence accepted and verified.",
      internalCaaNote: "Exact latest scan-clean version reviewed.",
    });

    expect(fetchImplementation.mock.calls[0]?.[0]).toBe("/v1/potential-findings");
    expect(JSON.parse(String(fetchImplementation.mock.calls[0]?.[1]?.body))).toMatchObject({
      inspectionAttachmentIds: ["ATT-PBE-001", "ATT-PBE-002"],
    });
    expect(fetchImplementation.mock.calls[1]?.[0]).toBe(
      "/v1/evidence/EVD-CAB-2026-001-V2/reviews",
    );
    expect(JSON.parse(String(fetchImplementation.mock.calls[1]?.[1]?.body))).toMatchObject({
      commentToAuditee: "Evidence accepted and verified.",
      internalCaaNote: "Exact latest scan-clean version reviewed.",
    });
  });

  it.each([
    [401, BackendAuthenticationError],
    [403, BackendAuthorizationError],
  ] as const)("maps %s problem responses without exposing transport DTOs", async (status, ErrorType) => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse(
        {
          type: "about:blank",
          title: status === 401 ? "Session expired" : "Forbidden",
          status,
          detail: "Deterministic fake response",
          code: status === 401 ? "SESSION_EXPIRED" : "FORBIDDEN",
          requestId: "REQ-TEST-001",
        },
        { status },
      ),
    );
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation, csrfToken: () => "csrf-test" },
    );
    await expect(backend.findings.get({ findingId: "FND-CAB-2026-001" })).rejects.toBeInstanceOf(
      ErrorType,
    );
    expect(fetchImplementation).toHaveBeenCalledTimes(1);
  });

  it("notifies authentication loss once for both protected reads and mutations before page handling", async () => {
    const authenticationLost = vi.fn();
    const fetchImplementation = vi.fn<typeof fetch>().mockImplementation(async () =>
      jsonResponse(
        {
          type: "about:blank",
          title: "Session expired",
          status: 401,
          detail: "The browser session has expired.",
          code: "SESSION_EXPIRED",
          requestId: "REQ-AUTH-LOST",
        },
        { status: 401 },
      ),
    );
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      {
        fetchImplementation,
        csrfToken: () => "csrf-test",
        onAuthenticationLost: authenticationLost,
      },
    );

    await expect(backend.assignments.list({ limit: 20 })).rejects.toBeInstanceOf(
      BackendAuthenticationError,
    );
    await expect(
      backend.caps.submit({
        operationId: "OP-CAP-AUTH-LOST",
        findingId: "FND-CAB-2026-001",
        expectedFindingRevision: 1,
        rootCause: "Root cause",
        correctiveAction: "Corrective action",
        preventiveAction: "Preventive action",
        responsiblePerson: "Responsible person",
        targetCompletionDate: "2026-07-15",
        commentToCaa: "CAA comment",
      }),
    ).rejects.toBeInstanceOf(BackendAuthenticationError);
    expect(authenticationLost).toHaveBeenCalledTimes(1);
    expect(authenticationLost.mock.calls[0]?.[0]).toBeInstanceOf(BackendAuthenticationError);
  });

  it("rejects a successful non-JSON response", async () => {
    const fetchImplementation = vi
      .fn<typeof fetch>()
      .mockResolvedValue(new Response("not-json", { status: 200, headers: { "content-type": "text/plain" } }));
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation, csrfToken: () => "csrf-test" },
    );
    await expect(backend.findings.list({})).rejects.toBeInstanceOf(BackendProtocolError);
  });

  it("propagates an AbortSignal and maps cancellation", async () => {
    const controller = new AbortController();
    const fetchImplementation = vi.fn<typeof fetch>().mockImplementation(async (_url, init) => {
      expect(init?.signal).toBe(controller.signal);
      throw new DOMException("The operation was aborted", "AbortError");
    });
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation, csrfToken: () => "csrf-test" },
    );
    await expect(
      backend.assignments.list({}, { signal: controller.signal }),
    ).rejects.toBeInstanceOf(BackendCancelledError);
    expect(fetchImplementation).toHaveBeenCalledTimes(1);
  });

  it("maps request timeouts separately from caller cancellation", async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockImplementation(
      async (_url, init) => new Promise<Response>((_resolve, reject) => {
        init?.signal?.addEventListener("abort", () => {
          reject(new DOMException("The operation was aborted", "AbortError"));
        }, { once: true });
      }),
    );
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation, requestTimeoutMs: 1 },
    );

    await expect(backend.assignments.list({})).rejects.toBeInstanceOf(
      BackendTimeoutError,
    );
  });

  it("preserves conflict problem code and request identity without retry", async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse({
        type: "about:blank",
        title: "Revision conflict",
        status: 409,
        detail: "Expected revision no longer matches.",
        code: "REVISION_CONFLICT",
        requestId: "REQ-CONFLICT-001",
      }, { status: 409 }),
    );
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation },
    );

    const error = await backend.findings.get({
      findingId: "FND-CAB-2026-001",
    }).catch((cause: unknown) => cause);
    expect(error).toBeInstanceOf(BackendHttpError);
    expect(error).toMatchObject({
      status: 409,
      code: "REVISION_CONFLICT",
      requestId: "REQ-CONFLICT-001",
    });
    expect(fetchImplementation).toHaveBeenCalledTimes(1);
  });

  it("preserves opaque list cursors", async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse({ items: [], nextCursor: "opaque:assignment:cursor" }),
    );
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation },
    );

    await expect(backend.assignments.list({ limit: 20 })).resolves.toEqual({
      items: [],
      nextCursor: "opaque:assignment:cursor",
    });
  });

  it("maps lifecycle read routes and preserves AbortSignal for direct loads", async () => {
    const controller = new AbortController();
    const fetchImplementation = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(
        jsonResponse({
          items: [{
            id: "PF-2026-001",
            auditId: "AUD-2026-001",
            questionId: "CAB-EMEQ-PBE-001",
            organizationId: "ORG-FLY-NAMIBIA",
            title: "PBE serviceability and accessibility not confirmed",
            description: "Configured check exception.",
            status: "PENDING_LEAD_REVIEW",
            revision: 1,
            convertedFindingId: null,
          }],
          nextCursor: null,
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          audience: "AUDITEE",
          id: "CAP-CAB-2026-001-R1",
          capId: "CAP-CAB-2026-001",
          findingId: "FND-CAB-2026-001",
          organizationId: "ORG-FLY-NAMIBIA",
          revision: 1,
          status: "ACCEPTED",
          rootCause: "Root cause",
          correctiveAction: "Corrective action",
          preventiveAction: "Preventive action",
          responsiblePerson: "Fly Namibia Cabin Safety Manager",
          targetCompletionDate: "2026-07-15",
          commentToCaa: "CAP submitted for CAA review.",
          submittedAt: "2026-06-15T09:00:00.000Z",
          latestReview: {
            decision: "ACCEPT",
            commentToAuditee: "CAP accepted.",
            decidedAt: "2026-06-15T09:00:00.000Z",
          },
        }),
      );
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation, csrfToken: () => "csrf-test" },
    );

    const queue = await backend.potentialFindings.list(
      { status: "PENDING_LEAD_REVIEW", limit: 50 },
      { signal: controller.signal },
    );
    expect(queue.items[0]?.id).toBe("PF-2026-001");
    const cap = await backend.caps.getRevision(
      { capRevisionId: "CAP-CAB-2026-001-R1" },
      { signal: controller.signal },
    );
    expect(cap.audience).toBe("AUDITEE");
    expect(JSON.stringify(cap)).not.toMatch(/internalCaaNote/i);

    expect(fetchImplementation.mock.calls[0]?.[0]).toBe(
      "/v1/potential-findings?status=PENDING_LEAD_REVIEW&limit=50",
    );
    expect(fetchImplementation.mock.calls[0]?.[1]?.signal).toBe(controller.signal);
    expect(fetchImplementation.mock.calls[1]?.[0]).toBe(
      "/v1/cap-revisions/CAP-CAB-2026-001-R1",
    );
    expect(fetchImplementation.mock.calls[1]?.[1]?.signal).toBe(controller.signal);
  });

  it("loads checklist template version detail through the Admin configuration route", async () => {
    const controller = new AbortController();
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse({
        id: "CTV-CABIN-1",
        templateId: "CABIN",
        title: "Cabin Inspection checklist",
        version: 1,
        status: "PUBLISHED",
        publishedAt: "2026-06-15T09:00:00.000Z",
        questionCount: 6,
        questions: [
          {
            id: "CAB-EMEQ-PBE-001",
            sectionId: "EM EQ / PBE",
            prompt:
              "Is the PBE installed, serviceable, accessible, and in compliance with configured cabin emergency equipment requirements?",
            regulatoryReference: "Configured Cabin Inspection reference - EM EQ / PBE",
            expectedEvidence: "PBE serviceability record and cabin position confirmation",
            allowedAnswers: [
              "COMPLIANT",
              "NON_COMPLIANT",
              "OBSERVATION",
              "NOT_APPLICABLE",
              "NOT_CHECKED",
            ],
            commentRequiredFor: ["NON_COMPLIANT", "OBSERVATION"],
          },
        ],
      }),
    );
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation, csrfToken: () => "csrf-test" },
    );

    const detail = await backend.configuration.getChecklistTemplateVersion(
      { templateVersionId: "CTV-CABIN-1" },
      { signal: controller.signal },
    );

    expect(detail.questions[0]?.allowedAnswers).toEqual([
      "COMPLIANT",
      "NON_COMPLIANT",
      "OBSERVATION",
      "NOT_APPLICABLE",
      "NOT_CHECKED",
    ]);
    expect(detail.questions[0]?.commentRequiredFor).toEqual(["NON_COMPLIANT", "OBSERVATION"]);
    expect(JSON.stringify(detail)).not.toMatch(/assignedInspectorUserIds|currentResponse/i);
    expect(fetchImplementation.mock.calls[0]?.[0]).toBe(
      "/v1/configuration/checklist-template-versions/CTV-CABIN-1",
    );
    expect(fetchImplementation.mock.calls[0]?.[1]?.signal).toBe(controller.signal);
  });

  it("maps first-production registry and planning requests to exact versioned routes", async () => {
    const fetchImplementation = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(
        jsonResponse({
          items: [{
            id: "ORG-FLY-NAMIBIA",
            legalName: "Fly Namibia",
            organizationType: "OPERATOR",
            status: "ACTIVE",
            openFindingCount: 0,
            lastAuditDate: null,
            nextAuditDate: "2026-07-15",
            revision: 1,
          }],
          nextCursor: null,
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          id: "PLAN-2026-CAB-001",
          title: "2026 Cabin Surveillance — Fly Namibia",
          planYear: 2026,
          organizationId: "ORG-FLY-NAMIBIA",
          organizationName: "Fly Namibia",
          inspectionType: "CABIN",
          scheduledDate: "2026-07-15",
          estimatedBudget: 48000,
          status: "GM_REVIEW",
          currentOwnerRole: "gm",
          nextAction: "General Manager to review operational scope",
          revision: 2,
        }),
      );
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation, csrfToken: () => "csrf-test" },
    );

    const registry = await backend.organizations.list({ limit: 20 });
    expect(registry.items[0]?.legalName).toBe("Fly Namibia");
    const updated = await backend.planning.decide({
      operationId: "OP-PLAN-FINANCE-APPROVE",
      planningItemId: "PLAN-2026-CAB-001",
      expectedPlanningRevision: 1,
      decision: "APPROVE_BUDGET",
      reason: "Budget envelope confirmed.",
    });
    expect(updated).toMatchObject({ status: "GM_REVIEW", currentOwnerRole: "gm", revision: 2 });

    expect(fetchImplementation.mock.calls[0]?.[0]).toBe("/v1/organizations?limit=20");
    const [decisionURL, decisionInit] = fetchImplementation.mock.calls[1]!;
    expect(decisionURL).toBe("/v1/planning/items/PLAN-2026-CAB-001/decisions");
    expect(decisionInit?.method).toBe("POST");
    expect(JSON.parse(String(decisionInit?.body))).toMatchObject({
      operationId: "OP-PLAN-FINANCE-APPROVE",
      decision: "APPROVE_BUDGET",
      expectedPlanningRevision: 1,
    });
  });

  it("maps the Task 4 planning, package, team, and Auditee notice capability slice", async () => {
    const draft = {
      id: "PLAN-DRAFT-2026-001",
      organizationId: "ORG-FLY-NAMIBIA",
      organizationName: "Fly Namibia",
      applicationType: "Air Operator Certificate",
      domain: "Cabin Safety",
      inspectionCategory: "Routine / Announced" as const,
      noticePolicy: "ADVANCE" as const,
      purpose: "Annual routine oversight",
      triggerType: "Annual Plan",
      riskCategory: "Cabin Safety",
      plannedDate: "2026-07-15",
      mode: "On-site" as const,
      location: "Windhoek",
      templateVersionId: "CTV-CABIN-1",
      scope: "Cabin safety and emergency equipment.",
      requestedBudget: 0,
      currency: "NAD" as const,
      revision: 1,
      submittedPlanningItemId: null,
      updatedAt: "2026-07-23T09:00:00Z",
    };
    const packageDraft = {
      id: "PKG-AUD-2026-001-CABIN",
      sourceAuditId: "AUD-2026-001",
      organizationId: "ORG-FLY-NAMIBIA",
      organizationName: "Fly Namibia",
      applicationType: "Air Operator Certificate",
      domain: "Cabin Safety",
      status: "DRAFT" as const,
      packageVersion: 1,
      revision: 1,
      riskFocus: ["Emergency equipment"],
      questions: [{
        id: "CAB-EMEQ-PBE-001",
        prompt: "Is the PBE serviceable?",
        whyIncluded: "Configured Cabin safety risk focus",
        expectedEvidence: ["PBE serviceability record"],
        configuredReference: "Configured Cabin Inspection reference",
      }],
      updatedAt: "2026-07-23T09:00:00Z",
    };
    const member = {
      subjectId: "USR-LEAD-CANER",
      displayName: "Caner Lead Inspector",
      role: "leadInspector" as const,
      organizationId: "CAA",
      revision: 1,
    };
    const auditTeam = {
      auditId: "AUD-2026-001",
      organizationId: "ORG-FLY-NAMIBIA",
      organizationName: "Fly Namibia",
      title: "2026 Cabin Inspection - Fly Namibia",
      status: "AWAITING_AUDITEE_CONFIRMATION",
      scheduledStartDate: "2026-07-15",
      scheduledEndDate: "2026-07-16",
      leadInspector: member,
      members: [member],
      assignments: [{
        questionId: "CAB-EMEQ-PBE-001",
        assignedMemberSubjectIds: ["USR-LEAD-CANER"],
      }],
      documents: [],
      history: [],
      revision: 4,
    };
    const coordination = {
      auditId: "AUD-2026-001",
      organizationId: "ORG-FLY-NAMIBIA",
      organizationName: "Fly Namibia",
      title: "2026 Cabin Inspection - Fly Namibia",
      inspectionCategory: "Routine / Announced" as const,
      scheduledStartDate: "2026-07-15",
      status: "AWAITING_AUDITEE_CONFIRMATION" as const,
      alternativeDate: null,
      nextAction: "Confirm proposed date or provide an alternative date",
      revision: 4,
    };
    const fetchImplementation = vi.fn<typeof fetch>().mockImplementation(async (input, init) => {
      const url = String(input);
      if (url.includes("/v1/planning/intake-drafts/") && url.endsWith("/submissions")) {
        return jsonResponse({
          draft: { ...draft, revision: 2, submittedPlanningItemId: "PLAN-2026-CAB-001" },
          planningItem: {
            id: "PLAN-2026-CAB-001",
            title: "Routine / Announced — Fly Namibia",
            planYear: 2026,
            organizationId: "ORG-FLY-NAMIBIA",
            organizationName: "Fly Namibia",
            inspectionType: "Air Operator Certificate · Cabin Safety",
            scheduledDate: "2026-07-15",
            estimatedBudget: 0,
            status: "FINANCE_REVIEW",
            currentOwnerRole: "finance",
            nextAction: "Finance to review budget and resources",
            revision: 1,
          },
        });
      }
      if (url.includes("/v1/planning/intake-drafts/")) {
        return jsonResponse(init?.method === "PUT" ? { ...draft, revision: 2 } : draft);
      }
      if (url.includes("/v1/inspection-package-drafts/")) {
        return jsonResponse(init?.method === "PUT" ? { ...packageDraft, revision: 2 } : packageDraft);
      }
      if (url === "/v1/team-members?role=leadInspector") {
        return jsonResponse({ items: [member], nextCursor: null });
      }
      if (url === "/v1/audit-teams?limit=20") {
        return jsonResponse({ items: [auditTeam], nextCursor: null });
      }
      if (url === "/v1/auditee/coordination") {
        return jsonResponse({ items: [coordination], nextCursor: null });
      }
      if (url.endsWith("/responses")) {
        return jsonResponse({ ...coordination, status: "CONFIRMED", revision: 5 });
      }
      throw new Error(`Unexpected Task 4 request: ${url}`);
    });
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation, csrfToken: () => "csrf-task-4" },
    );

    const loadedDraft = await backend.planningIntake!.getDraft({ draftId: draft.id });
    const savedDraft = await backend.planningIntake!.saveDraft({
      idempotencyKey: "IDEM-TASK-4-SAVE",
      expectedRevision: loadedDraft.revision,
      draftId: loadedDraft.id,
      values: loadedDraft,
    });
    const submitted = await backend.planningIntake!.submit({
      idempotencyKey: "IDEM-TASK-4-SUBMIT",
      expectedRevision: savedDraft.revision,
      draftId: savedDraft.id,
      planningItemId: "PLAN-2026-CAB-001",
    });
    const loadedPackage = await backend.packageDrafts!.get({ packageDraftId: packageDraft.id });
    const savedPackage = await backend.packageDrafts!.save({
      idempotencyKey: "IDEM-TASK-4-PACKAGE",
      expectedRevision: loadedPackage.revision,
      packageDraftId: loadedPackage.id,
      riskFocus: ["Emergency equipment"],
    });
    const members = await backend.teams!.list({ role: "leadInspector" });
    const teams = await backend.teams!.listAuditTeams({ limit: 20 });
    const notices = await backend.auditeeCoordination!.list({});
    const confirmed = await backend.auditeeCoordination!.respond({
      idempotencyKey: "IDEM-TASK-4-CONFIRM",
      expectedRevision: notices.items[0]!.revision,
      auditId: notices.items[0]!.auditId,
      organizationId: notices.items[0]!.organizationId,
      decision: "CONFIRM",
      alternativeDate: null,
    });

    expect(submitted.planningItem).toMatchObject({
      status: "FINANCE_REVIEW",
      estimatedBudget: 0,
    });
    expect(savedPackage).toMatchObject({ packageVersion: 1, revision: 2 });
    expect(members.items).toEqual([member]);
    expect(teams.items[0]).toMatchObject({
      auditId: "AUD-2026-001",
      assignments: [{
        questionId: "CAB-EMEQ-PBE-001",
        assignedMemberSubjectIds: ["USR-LEAD-CANER"],
      }],
    });
    expect(confirmed).toMatchObject({ status: "CONFIRMED", revision: 5 });
    const mutationHeaders = fetchImplementation.mock.calls
      .filter(([, init]) => init?.method === "PUT" || init?.method === "POST")
      .map(([, init]) => new Headers(init?.headers));
    expect(mutationHeaders.map((headers) => headers.get("idempotency-key"))).toEqual([
      "IDEM-TASK-4-SAVE",
      "IDEM-TASK-4-SUBMIT",
      "IDEM-TASK-4-PACKAGE",
      "IDEM-TASK-4-CONFIRM",
    ]);
    expect(mutationHeaders.map((headers) => headers.get("if-match"))).toEqual([
      '"rev-1"',
      '"rev-2"',
      '"rev-1"',
      '"rev-4"',
    ]);
    expect(fetchImplementation.mock.calls
      .filter(([, init]) => init?.method === "PUT" || init?.method === "POST")
      .map(([, init]) => JSON.parse(String(init?.body)).operationId)).toEqual([
      "IDEM-TASK-4-SAVE",
      "IDEM-TASK-4-SUBMIT",
      "IDEM-TASK-4-PACKAGE",
      "IDEM-TASK-4-CONFIRM",
    ]);
  });

  it("maps the Task 5 Admin Question, Template Draft, reorder, and immutable Package slice", async () => {
    const published = {
      id: "CTV-CABIN-1",
      templateId: "TPL-CABIN-2026",
      version: 1,
      status: "PUBLISHED" as const,
      owner: "Department Manager" as const,
      creatorSubjectId: "USR-MANAGER-NORA",
      changeReason: "Initial immutable published Cabin Inspection version.",
      questionIds: ["CAB-GALLEY-001", "CAB-EMEQ-PBE-001"],
      revision: 1,
      createdAt: "2026-06-15T09:00:00Z",
    };
    const createdQuestion = {
      id: "Q-ADMIN-2026-007",
      prompt: "Is the multiline record complete?\nDoes it identify the cabin position?",
      configuredReference: "Configured Cabin Inspection reference — EM EQ / PBE",
      expectedEvidence: "PBE record\nCabin position",
      revision: 1,
    };
    const draft = {
      ...published,
      id: "CTV-CABIN-DRAFT-2",
      version: 2,
      status: "DRAFT" as const,
      owner: "Admin Preview" as const,
      creatorSubjectId: "USR-ADMIN-ADA",
      changeReason: "Add multiline Question.",
      createdAt: "2026-06-15T09:01:00Z",
    };
    const added = {
      ...draft,
      questionIds: [...draft.questionIds, createdQuestion.id],
      revision: 2,
    };
    const moved = {
      ...added,
      questionIds: ["CAB-GALLEY-001", createdQuestion.id, "CAB-EMEQ-PBE-001"],
      revision: 3,
    };
    const immutablePackage = {
      id: "PKG-CAB-2026-001",
      auditId: "AUD-2026-001",
      organizationId: "ORG-FLY-NAMIBIA",
      organizationName: "Fly Namibia",
      questionIds: ["CAB-GALLEY-001", "CAB-EMEQ-PBE-001"],
      configuredReferences: ["Configured GALLEY", "Configured EM EQ / PBE"],
      expectedEvidence: ["Galley record", "PBE record"],
      riskFocus: ["Emergency equipment serviceability"],
    };
    const fetchImplementation = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse({
        items: [{
          id: "CAB-EMEQ-PBE-001",
          prompt: "Is the PBE serviceable?",
          configuredReference: "Configured EM EQ / PBE",
          expectedEvidence: "PBE record",
          revision: 1,
        }],
        nextCursor: null,
      }))
      .mockResolvedValueOnce(jsonResponse(createdQuestion, { status: 201 }))
      .mockResolvedValueOnce(jsonResponse({
        id: "TPL-CABIN-2026",
        publishedVersionId: "CTV-CABIN-1",
        versions: [published],
        revision: 1,
      }))
      .mockResolvedValueOnce(jsonResponse(draft, { status: 201 }))
      .mockResolvedValueOnce(jsonResponse(added))
      .mockResolvedValueOnce(jsonResponse(moved))
      .mockResolvedValueOnce(jsonResponse(immutablePackage));
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation, csrfToken: () => "csrf-task-5" },
    );
    const admin = backend.adminWorkspace!;

    const questions = await admin.listQuestions({ search: "PBE" });
    const question = await admin.createQuestion({
      idempotencyKey: "IDEM-TASK-5-QUESTION",
      expectedRevision: null,
      prompt: createdQuestion.prompt,
      configuredReference: createdQuestion.configuredReference,
      expectedEvidence: createdQuestion.expectedEvidence,
    });
    const template = await admin.getTemplate({ templateId: "TPL-CABIN-2026" });
    const draftResult = await admin.createDraft({
      idempotencyKey: "IDEM-TASK-5-DRAFT",
      expectedRevision: template.revision,
      templateId: template.id,
      changeReason: "Add multiline Question.",
    });
    const addedResult = await admin.addDraftQuestion({
      idempotencyKey: "IDEM-TASK-5-ADD",
      expectedRevision: draftResult.revision,
      templateId: template.id,
      draftVersionId: draftResult.id,
      questionId: question.id,
    });
    const movedResult = await admin.moveDraftQuestion({
      idempotencyKey: "IDEM-TASK-5-MOVE",
      expectedRevision: addedResult.revision,
      templateId: template.id,
      draftVersionId: addedResult.id,
      questionId: question.id,
      direction: "UP",
    });
    const packageResult = await admin.getInspectionPackage({
      packageId: "PKG-CAB-2026-001",
    });

    expect(questions.items[0]?.id).toBe("CAB-EMEQ-PBE-001");
    expect(question.prompt).toContain("\n");
    expect(movedResult).toMatchObject({
      id: "CTV-CABIN-DRAFT-2",
      revision: 3,
      questionIds: ["CAB-GALLEY-001", question.id, "CAB-EMEQ-PBE-001"],
    });
    expect(packageResult.questionIds).not.toContain(question.id);
    expect(fetchImplementation.mock.calls.map(([url]) => url)).toEqual([
      "/v1/admin/questions?search=PBE",
      "/v1/admin/questions",
      "/v1/admin/templates/TPL-CABIN-2026",
      "/v1/admin/templates/TPL-CABIN-2026/drafts",
      "/v1/admin/templates/TPL-CABIN-2026/drafts/CTV-CABIN-DRAFT-2/questions",
      "/v1/admin/templates/TPL-CABIN-2026/drafts/CTV-CABIN-DRAFT-2/questions/Q-ADMIN-2026-007/moves",
      "/v1/admin/inspection-packages/PKG-CAB-2026-001",
    ]);
    const mutationHeaders = fetchImplementation.mock.calls
      .filter(([, init]) => init?.method === "POST")
      .map(([, init]) => new Headers(init?.headers));
    expect(mutationHeaders.map((headers) => headers.get("idempotency-key"))).toEqual([
      "IDEM-TASK-5-QUESTION",
      "IDEM-TASK-5-DRAFT",
      "IDEM-TASK-5-ADD",
      "IDEM-TASK-5-MOVE",
    ]);
    expect(mutationHeaders.map((headers) => headers.get("if-match"))).toEqual([
      null,
      '"rev-1"',
      '"rev-1"',
      '"rev-2"',
    ]);
    expect(fetchImplementation.mock.calls
      .filter(([, init]) => init?.method === "POST")
      .map(([, init]) => JSON.parse(String(init?.body)).operationId)).toEqual([
      "IDEM-TASK-5-QUESTION",
      "IDEM-TASK-5-DRAFT",
      "IDEM-TASK-5-ADD",
      "IDEM-TASK-5-MOVE",
    ]);
  });

  it("maps the Task 7 immutable document and Auditee-safe report capability slice", async () => {
    const document = {
      id: "RPT-CAB-2026-001-V1",
      organizationId: "ORG-FLY-NAMIBIA",
      title: "Report RPT-CAB-2026-001",
      kind: "REPORT" as const,
      version: 1,
      revision: 2,
      createdAt: "2026-06-15T09:00:00Z",
      publicReviewResult: "RELEASED",
      downloadFileName: "RPT-CAB-2026-001.pdf",
    };
    const report = {
      reportVersionId: "RPT-CAB-2026-001-V1",
      reportId: "RPT-CAB-2026-001",
      kind: "FINAL" as const,
      organizationId: "ORG-FLY-NAMIBIA",
      auditId: "AUD-2026-001",
      findingIds: ["FND-CAB-2026-001"],
      version: 1,
      status: "LOCKED" as const,
      revision: 2,
      issuedAt: "2026-06-15T09:00:00Z",
      responseDueDate: null,
      caaVisibleCommentState: "NO_COMMENT_RECORDED" as const,
      caaVisibleComment: null,
    };
    const fetchImplementation = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse({ items: [document], nextCursor: null }))
      .mockResolvedValueOnce(jsonResponse(document))
      .mockResolvedValueOnce(jsonResponse({ items: [report], nextCursor: null }))
      .mockResolvedValueOnce(jsonResponse(report));
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Task 7 Test" },
      { fetchImplementation, csrfToken: () => "csrf-task-7" },
    );

    const documents = await backend.documents!.list({ organizationId: "ORG-FLY-NAMIBIA" });
    const opened = await backend.documents!.open({ documentId: document.id });
    const reports = await backend.auditeeReports!.listReleased({ kind: "FINAL" });
    const released = await backend.auditeeReports!.getReleased({
      reportVersionId: report.reportVersionId,
    });

    expect(documents.items).toEqual([document]);
    expect(opened).toEqual(document);
    expect(reports.items).toEqual([report]);
    expect(released).toEqual(report);
    expect(fetchImplementation.mock.calls.map(([url]) => url)).toEqual([
      "/v1/documents?organizationId=ORG-FLY-NAMIBIA",
      "/v1/documents/RPT-CAB-2026-001-V1",
      "/v1/auditee/report-versions?kind=FINAL",
      "/v1/auditee/report-versions/RPT-CAB-2026-001-V1",
    ]);
  });

  it("maps the Task 8 communications, calendar, and notification capability slices", async () => {
    const communication = {
      id: "COMM-2026-001",
      organizationId: "ORG-FLY-NAMIBIA",
      subject: "Cabin Inspection follow-up",
      body: "Please provide the requested public training record.",
      audience: "AUDITEE" as const,
      direction: "CAA_TO_AUDITEE" as const,
      revision: 1,
      createdAt: "2026-06-15T09:00:00Z",
    };
    const calendarItem = {
      id: "CAL-AUD-2026-001",
      auditId: "AUD-2026-001",
      organizationId: "ORG-FLY-NAMIBIA",
      organizationName: "Fly Namibia",
      title: "2026 Cabin Inspection - Fly Namibia",
      nextAction: "Continue Cabin Inspection checklist",
      scheduledDate: "2026-06-15",
      dueState: "DUE_TODAY" as const,
    };
    const notification = {
      id: "NOTIFICATION-2026-001",
      subjectId: "USR-AUDITEE-FLY",
      title: "New CAA communication",
      body: "Cabin Inspection follow-up — open the authorized message record for details.",
      readAt: null,
      revision: 1,
    };
    const readNotification = {
      ...notification,
      readAt: "2026-06-15T09:05:00Z",
      revision: 2,
    };
    const fetchImplementation = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse({ items: [communication], nextCursor: null }))
      .mockResolvedValueOnce(jsonResponse(communication, { status: 201 }))
      .mockResolvedValueOnce(jsonResponse({ items: [calendarItem], nextCursor: null }))
      .mockResolvedValueOnce(jsonResponse(calendarItem))
      .mockResolvedValueOnce(jsonResponse({ items: [notification], nextCursor: null }))
      .mockResolvedValueOnce(jsonResponse(readNotification));
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Task 8 Test" },
      { fetchImplementation, csrfToken: () => "csrf-task-8" },
    );

    const messages = await backend.communications!.list({
      organizationId: "ORG-FLY-NAMIBIA",
    });
    const sent = await backend.communications!.send({
      idempotencyKey: "IDEM-TASK-8-MESSAGE",
      expectedRevision: null,
      organizationId: "ORG-FLY-NAMIBIA",
      subject: communication.subject,
      body: communication.body,
      audience: "AUDITEE",
    });
    const calendar = await backend.calendar!.list({
      organizationId: "ORG-FLY-NAMIBIA",
    });
    const opened = await backend.calendar!.openItem({
      calendarItemId: calendarItem.id,
    });
    const notifications = await backend.notifications!.list({});
    const markedRead = await backend.notifications!.markRead({
      idempotencyKey: "IDEM-TASK-8-READ",
      expectedRevision: notification.revision,
      notificationId: notification.id,
    });

    expect(messages.items).toEqual([communication]);
    expect(sent).toEqual(communication);
    expect(calendar.items).toEqual([calendarItem]);
    expect(opened).toEqual(calendarItem);
    expect(notifications.items).toEqual([notification]);
    expect(markedRead).toEqual(readNotification);
    expect(fetchImplementation.mock.calls.map(([url]) => url)).toEqual([
      "/v1/communications?organizationId=ORG-FLY-NAMIBIA",
      "/v1/communications",
      "/v1/calendar-items?organizationId=ORG-FLY-NAMIBIA",
      "/v1/calendar-items/CAL-AUD-2026-001",
      "/v1/notifications",
      "/v1/notifications/NOTIFICATION-2026-001/read",
    ]);
    const mutationCalls = fetchImplementation.mock.calls.filter(
      ([, init]) => init?.method === "POST",
    );
    expect(mutationCalls.map(([, init]) => new Headers(init?.headers).get("idempotency-key"))).toEqual([
      "IDEM-TASK-8-MESSAGE",
      "IDEM-TASK-8-READ",
    ]);
    expect(mutationCalls.map(([, init]) => JSON.parse(String(init?.body)))).toEqual([
      {
        idempotencyKey: "IDEM-TASK-8-MESSAGE",
        expectedRevision: null,
        organizationId: "ORG-FLY-NAMIBIA",
        subject: communication.subject,
        body: communication.body,
        audience: "AUDITEE",
        operationId: "IDEM-TASK-8-MESSAGE",
      },
      {
        idempotencyKey: "IDEM-TASK-8-READ",
        expectedRevision: 1,
        notificationId: "NOTIFICATION-2026-001",
        operationId: "IDEM-TASK-8-READ",
      },
    ]);
  });

  it("maps governed Task 9 Risk, Administration, Admin, and advisory capability slices", async () => {
    const riskOverview = {
      organizationId: "ORG-FLY-NAMIBIA",
      overdueFindingCount: 1,
      openFindingCount: 2,
      repeatFindingCount: 0,
      revision: 1,
    };
    const riskManagement = {
      findings: [{
        findingId: "FND-CAB-2026-001",
        findingNumber: "CAR-2026-001",
        organizationId: "ORG-FLY-NAMIBIA",
        organizationName: "Fly Namibia",
        inspectionId: "AUD-2026-001",
        inspectionTitle: "2026 Cabin Inspection - Fly Namibia",
        department: null,
        title: "Cabin record gap",
        severity: "LEVEL_2_MAJOR",
        riskLevel: "MEDIUM",
        status: "WAITING_FOR_CAP",
        issuedAt: "2026-06-15T09:00:00Z",
        dueState: "DUE_SOON",
        capRequired: true,
      }],
      capEffectiveness: [],
      generatedAt: "2026-06-15T09:00:00Z",
      revision: 1,
    };
    const adminScreen = {
      screenId: "admin-reports",
      organizationId: null,
      directRecordId: null,
      state: "ready",
      overdue: false,
      versionHistory: false,
      visibleActions: [{
        id: "download-admin-report",
        label: "Download report",
        kind: "fileDownload",
        effect: { type: "fileDownload", file: "admin-report.csv" },
      }],
    };
    const reportDefinition = {
      id: "CAPS_BY_DUE_STATE",
      title: "CAPs by Due State",
      description: "Governed CAP due-state management projection.",
      packageFields: ["findingId", "dueState"],
      actionReason: "Review only; no automatic closure.",
    };
    const accessEntry = {
      subjectId: "USR-INSPECTOR-DAVID",
      displayName: "David Inspector",
      roles: ["inspector"],
      organizationId: "CAA",
      email: "david.inspector@example.test",
      mfaEnrolled: true,
      mfaState: "enrolled",
      requiredActions: [],
      invitationState: "required-actions-complete",
      accountStatus: "enabled",
      applicationProfileState: "linked",
      membershipId: "membership-david",
      membershipState: "active",
      membershipRevision: 3,
      membershipDrift: "in-sync",
      lastSuccessfulSessionAt: "2026-07-21T12:00:00Z",
      providerObservedAt: "2026-07-21T12:00:00Z",
    };
    const organization = {
      id: "ORG-FLY-NAMIBIA",
      legalName: "Fly Namibia",
      organizationType: "OPERATOR",
      status: "ACTIVE",
      scope: "CAA oversight",
      detailAvailable: true,
      disabledReason: null,
    };
    const guidance = {
      advisoryOnly: true as const,
      prohibitedActions: ["create Finding", "set severity", "close Finding", "enforcement action"],
    };
    const draft = {
      id: "DRAFT-FND-SKYCARGO-2026-099",
      findingId: "FND-SKYCARGO-2026-099",
      prompt: "Draft an evidence request only.",
      draft: "Advisory draft for CAR-2026-099: review the configured finding basis and request only the expected evidence.",
      advisoryOnly: true as const,
      canCreateFinding: false as const,
      canSetSeverity: false as const,
      canCloseFinding: false as const,
    };
    const fetchImplementation = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse(riskOverview))
      .mockResolvedValueOnce(jsonResponse(riskManagement))
      .mockResolvedValueOnce(jsonResponse(adminScreen))
      .mockResolvedValueOnce(jsonResponse([adminScreen]))
      .mockResolvedValueOnce(jsonResponse({
        screenId: adminScreen.screenId,
        actionId: "download-admin-report",
        effect: adminScreen.visibleActions[0]!.effect,
      }))
      .mockResolvedValueOnce(jsonResponse({ items: [reportDefinition], nextCursor: null }))
      .mockResolvedValueOnce(jsonResponse({
        items: [accessEntry],
        nextCursor: null,
        consistencyToken: "2026-07-21T12:00:00Z",
        providerCalls: 2,
      }))
      .mockResolvedValueOnce(jsonResponse({ items: [organization], nextCursor: null }))
      .mockResolvedValueOnce(jsonResponse(organization))
      .mockResolvedValueOnce(jsonResponse({ items: [], nextCursor: null }))
      .mockResolvedValueOnce(jsonResponse(guidance))
      .mockResolvedValueOnce(jsonResponse(draft));
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Task 9 Test" },
      { fetchImplementation, csrfToken: () => "csrf-task-9" },
    );

    expect(await backend.risk!.getOverview({
      organizationId: "ORG-FLY-NAMIBIA",
    })).toEqual(riskOverview);
    expect(await backend.risk!.getManagementProjection({})).toEqual(riskManagement);
    expect(await backend.administration!.getScreenProjection({
      screenId: "admin-reports",
    })).toEqual(adminScreen);
    expect(await backend.administration!.listScreenProjections({})).toEqual([adminScreen]);
    expect(await backend.administration!.invokeVisibleAction({
      screenId: "admin-reports",
      actionId: "download-admin-report",
    })).toEqual({
      screenId: "admin-reports",
      actionId: "download-admin-report",
      effect: { type: "fileDownload", file: "admin-report.csv" },
    });
    expect((await backend.adminWorkspace!.listReportDefinitions({
      search: "due",
    })).items).toEqual([reportDefinition]);
    expect((await backend.adminWorkspace!.listAccessDirectory({
      search: "David",
      role: "inspector",
    })).items).toEqual([accessEntry]);
    expect((await backend.adminWorkspace!.listOrganizations({
      search: "Fly",
      organizationType: "OPERATOR",
      status: "ACTIVE",
      scope: "CAA oversight",
    })).items).toEqual([organization]);
    expect(await backend.adminWorkspace!.getOrganization({
      organizationId: "ORG-FLY-NAMIBIA",
    })).toEqual(organization);
    expect((await backend.adminWorkspace!.listAuditEvents({
      actor: "David",
      action: "assistant.advisory",
      entity: "ASSISTANT_DRAFT",
      system: "MANUAL",
      dateText: "2026-06-15",
    })).items).toEqual([]);
    expect(await backend.assistantDrafts!.getGuidance({})).toEqual(guidance);
    expect(await backend.assistantDrafts!.createDraft({
      findingId: draft.findingId,
      prompt: draft.prompt,
    })).toEqual(draft);

    expect(fetchImplementation.mock.calls.map(([url]) => url)).toEqual([
      "/v1/risk/overview?organizationId=ORG-FLY-NAMIBIA",
      "/v1/risk/management",
      "/v1/administration/screens/admin-reports",
      "/v1/administration/screens",
      "/v1/administration/screens/admin-reports/actions/download-admin-report",
      "/v1/admin/report-definitions?search=due",
      "/v1/admin/access-directory?search=David&role=inspector",
      "/v1/admin/organizations?search=Fly&organizationType=OPERATOR&status=ACTIVE&scope=CAA+oversight",
      "/v1/admin/organizations/ORG-FLY-NAMIBIA",
      "/v1/admin/audit-events?actor=David&action=assistant.advisory&entity=ASSISTANT_DRAFT&system=MANUAL&dateText=2026-06-15",
      "/v1/assistant/guidance",
      "/v1/assistant/drafts",
    ]);
    const mutations = fetchImplementation.mock.calls.filter(
      ([, init]) => init?.method === "POST",
    );
    expect(mutations).toHaveLength(2);
    const actionBody = JSON.parse(String(mutations[0]![1]?.body));
    expect(actionBody).toMatchObject({
      operationId: expect.any(String),
      expectedRevision: null,
      idempotencyKey: expect.any(String),
      screenId: "admin-reports",
      actionId: "download-admin-report",
    });
    const assistantBody = JSON.parse(String(mutations[1]![1]?.body));
    expect(assistantBody).toEqual({
      operationId: assistantBody.idempotencyKey,
      expectedRevision: null,
      idempotencyKey: assistantBody.idempotencyKey,
      findingId: draft.findingId,
      prompt: draft.prompt,
    });
    expect(assistantBody.idempotencyKey).toMatch(/^assistant-draft:/);
    expect(new Headers(mutations[1]![1]?.headers).get("idempotency-key"))
      .toBe(assistantBody.idempotencyKey);
  });

  it("requests and reconciles Keycloak user provisioning through the exact admin transport", async () => {
    const pending = {
      id: "user-lifecycle-001",
      subjectId: null,
      action: "PROVISION" as const,
      roles: ["inspector" as const],
      organizationId: "ORG-FLY-NAMIBIA",
      email: "new.inspector@example.test",
      displayName: "New Inspector",
      status: "PENDING" as const,
      idempotencyKey: "provision:new.inspector@example.test",
      expectedMembershipRevision: 0,
      resultingMembershipRevision: 0,
      membershipId: null,
      reason: "Approved HTTP provisioning proof.",
      effectiveAt: null,
      providerFailureClass: null,
      providerAcknowledgedAt: null,
      attemptCount: 0,
      requestedBySubjectId: "USR-ADMIN-ADA",
      outboxMessageId: "outbox-user-lifecycle-001",
      failureReason: null,
      createdAt: "2026-07-24T08:00:00Z",
      updatedAt: "2026-07-24T08:00:00Z",
    };
    const succeeded = {
      ...pending,
      subjectId: "kc-subject-001",
      status: "SUCCEEDED" as const,
      updatedAt: "2026-07-24T08:00:01Z",
    };
    const fetchImplementation = vi
      .fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse(pending, { status: 202 }))
      .mockResolvedValueOnce(jsonResponse(succeeded));
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Plan 3 Test" },
      { fetchImplementation, csrfToken: () => "csrf-plan-3" },
    );

    expect(await backend.adminWorkspace.requestUserLifecycle({
      idempotencyKey: pending.idempotencyKey,
      action: "PROVISION",
      roles: ["inspector"],
      organizationId: pending.organizationId,
      email: pending.email,
      displayName: pending.displayName,
      reason: pending.reason,
      expectedMembershipRevision: 0,
      effectiveAt: null,
    })).toEqual(pending);
    expect(await backend.adminWorkspace.getUserLifecycleRequest({
      requestId: pending.id,
    })).toEqual(succeeded);

    const [url, init] = fetchImplementation.mock.calls[0]!;
    expect(url).toBe("/v1/admin/user-lifecycle-requests");
    expect(init?.method).toBe("POST");
    expect(new Headers(init?.headers).get("idempotency-key")).toBe(pending.idempotencyKey);
    expect(new Headers(init?.headers).get("x-csrf-token")).toBe("csrf-plan-3");
    expect(JSON.parse(String(init?.body))).toEqual({
      operationId: pending.idempotencyKey,
      idempotencyKey: pending.idempotencyKey,
      subjectId: null,
      action: "PROVISION",
      roles: ["inspector"],
      organizationId: pending.organizationId,
      email: pending.email,
      displayName: pending.displayName,
      reason: pending.reason,
      expectedMembershipRevision: 0,
      effectiveAt: null,
    });
    expect(fetchImplementation.mock.calls[1]?.[0]).toBe(
      "/v1/admin/user-lifecycle-requests/user-lifecycle-001",
    );
  });
});
