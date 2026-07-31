import { describe, expect, it, vi } from "vitest";

import { SYNTHETIC_GOVERNED_BUNDLE, SYNTHETIC_HYBRID_RECONCILED_BUNDLE } from "./governed-synthetic-profile";
import { createHttpBackend } from "./http-backend";

const governedCandidateResponse = {
  candidateId: "CAND-SYNTHETIC-OPS-AOC-0001",
  candidateRootId: "CAND-SYNTHETIC-OPS-AOC-0001",
  supersedesCandidateId: null,
  generationRunId: "GENRUN-SYNTHETIC-OPS-AOC-0001",
  templateId: "TPL-CHK-SYNTHETIC-OPS-AOC-001",
  version: 1,
  revision: 1,
  status: "GENERATED_DRAFT",
  contentDigest: SYNTHETIC_GOVERNED_BUNDLE.outputDigest,
  schemaVersion: "1.0.0",
  changeReason: "Imported deterministic synthetic governed candidate.",
  sourceSnapshots: [{
    sourceId: "SOURCE-SYNTHETIC-OPS-AOC",
    sourceIdentity: "SYNTHETIC-OPS-AOC",
    versionIdentity: "1",
    sourceHash: SYNTHETIC_GOVERNED_BUNDLE.generationRequest.sourceSnapshots[0]!.sourceHash,
    clauseId: "CLAUSE-SYNTHETIC-OPS-AOC-1",
    locator: "Synthetic OPS/AOC 1",
  }],
  scopeFactIds: ["SCOPE-SYNTHETIC-AOC"],
  crosswalkPartitionIds: ["PARTITION-SYNTHETIC-INPUT"],
  mappings: SYNTHETIC_GOVERNED_BUNDLE.complianceMappings,
  questions: SYNTHETIC_GOVERNED_BUNDLE.inspectionChecklist.questions,
  requiredOwners: [{
    departmentId: "FLIGHT_OPERATIONS_INSPECTORATE",
    organizationalUnitId: "FLIGHT_OPERATIONS_INSPECTORATE",
    approvalRequired: true,
  }],
} as const;

describe("governed checklist HTTP transport", () => {
  it("sends the exact synthetic import command once with its idempotency key", async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(new Response(JSON.stringify({
      generationRunId: "GENRUN-SYNTHETIC-OPS-AOC-0001", status: "GENERATED_DRAFT", requestId: "GENREQ-SYNTHETIC-OPS-AOC-0001", providerId: "deterministic-regulatory-fixture", candidate: null,
    }), { status: 200, headers: { "content-type": "application/json", "x-request-id": "REQ-GOVERNED-001" } }));
    const backend = createHttpBackend({ apiBaseUrl: "/", environmentLabel: "test" }, { fetchImplementation, csrfToken: () => "csrf-governed" });
    await backend.adminWorkspace.importGovernedGenerationRun({ operationId: "TASK9-HTTP-IMPORT", idempotencyKey: "TASK9-HTTP-IMPORT", candidateBundle: SYNTHETIC_GOVERNED_BUNDLE });
    expect(fetchImplementation).toHaveBeenCalledTimes(1);
    const [url, init] = fetchImplementation.mock.calls[0]!;
    expect(url).toBe("/v1/admin/governed-checklist/generation-runs");
    expect(init?.method).toBe("POST");
    const headers = new Headers(init?.headers);
    expect(headers.get("idempotency-key")).toBe("TASK9-HTTP-IMPORT");
    expect(headers.get("x-csrf-token")).toBe("csrf-governed");
    expect(JSON.parse(String(init?.body))).toEqual({ operationId: "TASK9-HTTP-IMPORT", idempotencyKey: "TASK9-HTTP-IMPORT", candidateBundle: SYNTHETIC_GOVERNED_BUNDLE });
  });

  it("preserves both required per-question governance views from HTTP transport", async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(new Response(JSON.stringify(
      governedCandidateResponse,
    ), { status: 200, headers: { "content-type": "application/json", "x-request-id": "REQ-GOVERNED-TRACE-001" } }));
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "test" },
      { fetchImplementation },
    );

    const candidate = await backend.adminWorkspace.getGovernedCandidate({
      candidateId: governedCandidateResponse.candidateId,
    });
    const [url, init] = fetchImplementation.mock.calls[0]!;
    expect(url).toBe("/v1/admin/governed-checklist/candidates/CAND-SYNTHETIC-OPS-AOC-0001");
    expect(init?.method).toBe("GET");
    expect(candidate.questions[0]).toEqual(
      SYNTHETIC_GOVERNED_BUNDLE.inspectionChecklist.questions[0],
    );
    expect(candidate.questions[0]!.scopeRecommendation).toEqual(expect.objectContaining({
      classification: "MANDATORY_CORE",
      rationale: expect.any(String),
      guardrails: expect.objectContaining({ mandatoryControl: true, safetyCritical: true }),
    }));
    expect(candidate.questions[0]!.regulatoryTrace).toEqual(expect.objectContaining({
      state: "RESOLVED",
      sourceTitle: "Synthetic test-profile source",
      immutableVersion: "1",
      sha256: SYNTHETIC_GOVERNED_BUNDLE.generationRequest.sourceSnapshots[0]!.sourceHash,
      currentnessState: "CURRENT",
    }));
  });

  it("sends one exact explicit source-currentness activation before a source-change import", async () => {
    const binding = SYNTHETIC_HYBRID_RECONCILED_BUNDLE.sourceCurrentness;
    if (!binding || !binding.previousSourceSnapshotId || !binding.previousSourceHash) {
      throw new Error("The source-change fixture must declare an exact predecessor/current binding.");
    }
    const input = {
      operationId: "TASK6-HTTP-SOURCE-ACTIVATION",
      idempotencyKey: "TASK6-HTTP-SOURCE-ACTIVATION",
      currentSourceSnapshotId: binding.currentSourceSnapshotId,
      currentSourceHash: binding.currentSourceHash,
      previousSourceSnapshotId: binding.previousSourceSnapshotId,
      previousSourceHash: binding.previousSourceHash,
      reason: "Activate the exact synthetic source change before importing a candidate.",
    };
    const receipt = {
      eventId: "SRC-CURRENTNESS-TEST-IMPACT",
      impactReviewDraftId: "SRC-IMPACT-DRAFT-TEST-IMPACT",
      sourceIdentity: "SYNTHETIC-OPS-AOC",
      previousSourceSnapshotId: input.previousSourceSnapshotId,
      previousSourceHash: input.previousSourceHash,
      currentSourceSnapshotId: input.currentSourceSnapshotId,
      currentSourceHash: input.currentSourceHash,
      status: "IMPACT_REVIEW_DRAFT",
      activatedAt: "2026-07-31T00:00:00Z",
    } as const;
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(new Response(JSON.stringify(receipt), {
      status: 201,
      headers: { "content-type": "application/json", "x-request-id": "REQ-GOVERNED-SOURCE-ACTIVATION" },
    }));
    const backend = createHttpBackend(
      { apiBaseUrl: "/", environmentLabel: "test" },
      { fetchImplementation, csrfToken: () => "csrf-currentness" },
    );

    await expect(backend.adminWorkspace.activateGovernedSourceCurrentness(input)).resolves.toEqual(receipt);
    expect(fetchImplementation).toHaveBeenCalledTimes(1);
    const [url, init] = fetchImplementation.mock.calls[0]!;
    expect(url).toBe("/v1/admin/governed-checklist/source-currentness-activations");
    expect(init?.method).toBe("POST");
    const headers = new Headers(init?.headers);
    expect(headers.get("idempotency-key")).toBe(input.idempotencyKey);
    expect(headers.get("x-csrf-token")).toBe("csrf-currentness");
    expect(JSON.parse(String(init?.body))).toEqual(input);
  });
});
