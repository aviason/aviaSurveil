import { describe, expect, it, vi } from "vitest";

import { createHttpBackend } from "./http-backend";

function jsonResponse(value: unknown): Response {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: { "content-type": "application/json", "x-request-id": "REQ-AGA-001" },
  });
}

describe("AGA demo workspace HTTP client", () => {
  it("uses fixed POST query bodies for search and never serializes filters into the URL", async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse({ operation: "SEARCH_ITEMS", generation: {}, items: [], itemCount: 0, lifecycleAvailable: false }),
    );
    const backend = createHttpBackend(
      { apiBaseUrl: "/api/", environmentLabel: "Test" },
      { fetchImplementation, csrfToken: () => "csrf-aga" },
    );

    await backend.agaDemoWorkspace?.classificationQuery({
      operationId: "SEARCH_ITEMS",
      search: "FSS-AGA-FORM-001",
      domainCode: "DOMAIN_A",
      topicCode: "TOPIC_A",
      confidence: "HIGH",
      blocker: "true",
      sourceGap: "false",
      externalInvolvement: "true",
      formCode: "FSS-AGA-FORM-001",
      page: 2,
      pageSize: 25,
    });

    const [url, init] = fetchImplementation.mock.calls[0]!;
    expect(url).toBe("/api/v1/preprod/aga-demo-workspace/classification/query");
    expect(String(url)).not.toContain("?");
    expect(init?.method).toBe("POST");
    expect(new Headers(init?.headers).get("x-csrf-token")).toBe("csrf-aga");
    expect(JSON.parse(String(init?.body))).toMatchObject({
      operationId: "SEARCH_ITEMS",
      search: "FSS-AGA-FORM-001",
      domainCode: "DOMAIN_A",
      blocker: "true",
      page: 2,
    });
  });

  it("keeps capability reads no-store and commands on the shared idempotency/CAS header path", async () => {
    const fetchImplementation = vi.fn<typeof fetch>()
      .mockResolvedValueOnce(jsonResponse({ available: true, projection: "CAA_ADMIN", classificationEnabled: true, recommendationEnabled: true, lifecycleEnabled: false, resetEnabled: true }))
      .mockResolvedValueOnce(jsonResponse({ operationId: "INCLUDE", replayed: false, lifecycleAvailable: false }));
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation, csrfToken: () => "csrf-aga" },
    );

    await backend.agaDemoWorkspace?.capability();
    await backend.agaDemoWorkspace?.classificationCommand({
      operationId: "INCLUDE",
      idempotencyKey: "AGA-IDEMPOTENCY-001",
      expectedGenerationId: "aga-ws-generation-0001",
      expectedDraftRevision: 7,
      expectedDraftContentDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
      targetQuestionKey: "server-returned-question-key",
      reasonCode: "MANAGER_SCOPE_DECISION",
    });

    const capabilityInit = fetchImplementation.mock.calls[0]?.[1];
    expect(capabilityInit?.cache).toBe("no-store");
    const commandInit = fetchImplementation.mock.calls[1]?.[1];
    const headers = new Headers(commandInit?.headers);
    expect(headers.get("idempotency-key")).toBe("AGA-IDEMPOTENCY-001");
    expect(headers.get("if-match")).toBe('"rev-7"');
    expect(JSON.parse(String(commandInit?.body))).toMatchObject({ operationId: "INCLUDE", expectedGenerationId: "aga-ws-generation-0001" });
  });

  it("sends all three exact generation CAS fields to the admin reset route", async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(
      jsonResponse({ operationId: "RESET_GENERATION", replayed: false, lifecycleAvailable: false }),
    );
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "Test" },
      { fetchImplementation, csrfToken: () => "csrf-aga" },
    );
    await backend.agaDemoWorkspace?.adminCommand({
      operationId: "RESET_GENERATION",
      idempotencyKey: "AGA-RESET-001",
      expectedGenerationId: "aga-ws-generation-0001",
      expectedGenerationRevision: 3,
      expectedGenerationSealDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
      reasonCode: "SYNTHETIC_RESET",
    });
    const [url, init] = fetchImplementation.mock.calls[0]!;
    expect(url).toBe("/v1/preprod/aga-demo-workspace/admin/commands");
    expect(new Headers(init?.headers).get("if-match")).toBe('"rev-3"');
    expect(JSON.parse(String(init?.body))).toMatchObject({
      expectedGenerationId: "aga-ws-generation-0001",
      expectedGenerationRevision: 3,
      expectedGenerationSealDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
    });
  });
});
