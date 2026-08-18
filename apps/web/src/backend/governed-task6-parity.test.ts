import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

import type {
  AuditEventView,
  GovernedCandidateView,
  GovernedPublicationView,
  GovernedReviewDecisionView,
} from "./backend";
import {
  BackendAuthorizationInvariantError,
  GovernedValidationError,
} from "./backend-contracts";
import {
  createMockBackend,
  createMockBackendRuntime,
} from "../mock/create-mock-backend";
import { MemoryMockStore } from "../mock/memory-mock-store";
import {
  EXACT_BLOCKED_REAL_OPS_AOC_REQUEST,
  SYNTHETIC_GOVERNED_BUNDLE,
  SYNTHETIC_HYBRID_RECONCILED_BUNDLE,
  SYNTHETIC_LEGACY_CHECKLIST_CANDIDATE_BUNDLE,
} from "./governed-synthetic-profile";
import { governedCanonicalSHA256 } from "./governed-canonical";

type Task6ReviewDecisionArtifact = GovernedReviewDecisionView & {
  actorMembershipIsCurrent: boolean;
};

type Task6AuditArtifact = {
  eventId: string;
  candidateRootId: string;
  candidateId: string;
  candidateRevision: number;
  candidateContentDigest: string;
  actorSubjectId: string;
  actorRole: string;
  actorDepartmentMembershipId: string;
  actorMembershipIsCurrent: boolean;
  actorDepartmentId: string;
  actorOrganizationalUnitId: string;
  action: string;
  entityType: string;
  entityId: string;
  beforeStatus: string;
  afterStatus: string;
  reason: string;
  occurredAt: string;
  operationId: string;
  idempotencyKey: string;
  semanticPayloadDigest: string;
  linkedDecisionId: string;
};

type Task6PublicationDecisionArtifact = GovernedPublicationView & {
  actorMembershipIsCurrent: boolean;
  createdAt: string;
};

type Task6ChecklistVersionArtifact = {
  templateVersionId: string;
  templateId: string;
  version: number;
  title: string;
  publishedAt: string;
  candidateRootId: string;
  candidateId: string;
  candidateRevision: number;
  candidateContentDigest: string;
  publicationDecisionId: string;
  auditEventId: string;
  questionVersionOrder: Array<{
    questionVersionId: string;
    position: number;
  }>;
  immutableSnapshot: {
    candidateId: string;
    candidateRevision: number;
    candidateContentDigest: string;
    complianceMappings: GovernedCandidateView["mappings"];
    questions: GovernedCandidateView["questions"];
  };
};

type Task6ArtifactRows = {
  reviewDecisions: Task6ReviewDecisionArtifact[];
  auditEvents: Task6AuditArtifact[];
  publicationDecisions: Task6PublicationDecisionArtifact[];
  checklistVersions: Task6ChecklistVersionArtifact[];
};

type Task6ArtifactCheckpoint = {
  reviewDecisionIds: string[];
  auditEventIds: string[];
  publicationDecisionIds: string[];
  checklistVersionIds: string[];
};

type Task6ArtifactParityContract = {
  contractId: string;
  clock: string;
  candidate: {
    candidateId: string;
    candidateRootId: string;
    revision: number;
    contentDigest: string;
    absentTemplateVersionId: string;
    tamperedDigest: string;
  };
  actors: {
    foi: {
      subjectId: string;
      membershipId: string;
      departmentId: string;
      organizationalUnitId: string;
    };
    air: {
      subjectId: string;
      membershipId: string;
      departmentId: string;
      organizationalUnitId: string;
    };
  };
  requiredOwners: Array<{
    departmentId: string;
    organizationalUnitId: string;
    approvalRequired: boolean;
  }>;
  operations: {
    import: { operationId: string; idempotencyKey: string };
    submit: { operationId: string; idempotencyKey: string; reason: string };
    foiApproval: { operationId: string; idempotencyKey: string; reason: string };
    airApproval: { operationId: string; idempotencyKey: string; reason: string };
    publication: { operationId: string; idempotencyKey: string; reason: string };
    digestTamper: { operationId: string; idempotencyKey: string; reason: string };
    conflictingReplayReason: string;
  };
  expected: {
    partialStatus: string;
    finalStatus: string;
    publishedStatus: string;
    artifactCheckpoints: {
      approvalOnly: Task6ArtifactCheckpoint;
      jointComplete: Task6ArtifactCheckpoint;
      published: Task6ArtifactCheckpoint;
    };
    artifactRows: Task6ArtifactRows;
  };
  publishedArtifact: {
    mappings: GovernedCandidateView["mappings"];
    questions: GovernedCandidateView["questions"];
  };
};

function loadTask6ArtifactParityContract(): Task6ArtifactParityContract {
  return JSON.parse(readFileSync(new URL(
    "../../../api/tests/fixtures/task6/manager-artifact-parity-contract.json",
    import.meta.url,
  ), "utf8")) as Task6ArtifactParityContract;
}

function task6ExpectedArtifactCheckpoint(
  contract: Task6ArtifactParityContract,
  checkpoint: keyof Task6ArtifactParityContract["expected"]["artifactCheckpoints"],
): Task6ArtifactRows {
  const identities = contract.expected.artifactCheckpoints[checkpoint];
  const exactRows = <Row,>(
    rows: Row[],
    expectedIds: string[],
    identity: (row: Row) => string,
  ) => {
    const byId = new Map(rows.map((row) => [identity(row), row]));
    const selected = expectedIds.map((id) => {
      const row = byId.get(id);
      if (!row) throw new Error(`Task 6 artifact contract is missing exact row ${id}.`);
      return row;
    });
    if (new Set(expectedIds).size !== expectedIds.length) {
      throw new Error(`Task 6 artifact checkpoint ${checkpoint} repeats a row identity.`);
    }
    return selected;
  };
  return {
    reviewDecisions: exactRows(
      contract.expected.artifactRows.reviewDecisions,
      identities.reviewDecisionIds,
      (row) => row.decisionId,
    ),
    auditEvents: exactRows(
      contract.expected.artifactRows.auditEvents,
      identities.auditEventIds,
      (row) => row.eventId,
    ),
    publicationDecisions: exactRows(
      contract.expected.artifactRows.publicationDecisions,
      identities.publicationDecisionIds,
      (row) => row.publicationDecisionId,
    ),
    checklistVersions: exactRows(
      contract.expected.artifactRows.checklistVersions,
      identities.checklistVersionIds,
      (row) => row.templateVersionId,
    ),
  };
}

function task6MockAuditArtifacts(
  events: AuditEventView[],
  candidate: GovernedCandidateView,
  reviewDecisions: Task6ReviewDecisionArtifact[],
  publicationDecisions: Task6PublicationDecisionArtifact[],
): Task6AuditArtifact[] {
  const linked = new Map<string, Task6ReviewDecisionArtifact | Task6PublicationDecisionArtifact>([
    ...reviewDecisions.map((decision) => [decision.auditEventId, decision] as const),
    ...publicationDecisions.map((decision) => [decision.auditEventId, decision] as const),
  ]);
  return events.map((event) => {
    const decision = linked.get(event.eventId);
    return {
      eventId: event.eventId,
      candidateRootId: candidate.candidateRootId,
      candidateId: candidate.candidateId,
      candidateRevision: event.entityRevision ?? candidate.revision,
      candidateContentDigest: candidate.contentDigest,
      actorSubjectId: event.actorSubjectId ?? "",
      actorRole: event.actorRole ?? "",
      actorDepartmentMembershipId: decision?.actorDepartmentMembershipId ?? "",
      actorMembershipIsCurrent: decision?.actorMembershipIsCurrent ?? false,
      actorDepartmentId: decision?.actorDepartmentId ?? "",
      actorOrganizationalUnitId: decision?.actorOrganizationalUnitId ?? "",
      action: event.action,
      entityType: event.entityType,
      entityId: event.entityId,
      beforeStatus: event.beforeStatus ?? "",
      afterStatus: event.afterStatus ?? "",
      reason: event.reason ?? "",
      occurredAt: event.occurredAt,
      operationId: decision?.operationId ?? "",
      idempotencyKey: decision?.idempotencyKey ?? "",
      semanticPayloadDigest: decision?.semanticPayloadDigest ?? "",
      linkedDecisionId: decision
        ? "decisionId" in decision
          ? decision.decisionId
          : decision.publicationDecisionId
        : "",
    };
  }).sort((left, right) => left.eventId.localeCompare(right.eventId));
}

async function submittedRuntime(label: string) {
  const runtime = createMockBackendRuntime();
  const admin = runtime.backendForRole("admin").adminWorkspace;
  const imported = await admin.importGovernedGenerationRun({
    operationId: `TASK6-${label}-IMPORT`,
    idempotencyKey: `TASK6-${label}-IMPORT`,
    candidateBundle: SYNTHETIC_GOVERNED_BUNDLE,
  });
  const candidate = imported.candidate!;
  const submitted = await admin.submitGovernedCandidateReview({
    operationId: `TASK6-${label}-SUBMIT`,
    idempotencyKey: `TASK6-${label}-SUBMIT`,
    candidateId: candidate.candidateId,
    expectedRevision: candidate.revision,
    expectedContentDigest: candidate.contentDigest,
    reason: "Submit exact synthetic candidate for department review.",
  });
  return { runtime, submitted };
}

function command(label: string, candidate: GovernedCandidateView) {
  return {
    operationId: `TASK6-${label}`,
    idempotencyKey: `TASK6-${label}`,
    candidateId: candidate.candidateId,
    expectedRevision: candidate.revision,
    expectedContentDigest: candidate.contentDigest,
    reason: `Task 6 ${label.toLowerCase()} decision.`,
  };
}

function sourceImpactActivation(label: string) {
  const binding = SYNTHETIC_HYBRID_RECONCILED_BUNDLE.sourceCurrentness;
  if (!binding || !binding.previousSourceSnapshotId || !binding.previousSourceHash) {
    throw new Error("The synthetic impact fixture must declare an exact predecessor/current binding.");
  }
  return {
    operationId: `TASK6-${label}-SOURCE-ACTIVATION`,
    idempotencyKey: `TASK6-${label}-SOURCE-ACTIVATION`,
    currentSourceSnapshotId: binding.currentSourceSnapshotId,
    currentSourceHash: binding.currentSourceHash,
    previousSourceSnapshotId: binding.previousSourceSnapshotId,
    previousSourceHash: binding.previousSourceHash,
    reason: "Activate the exact current synthetic source before importing a separate reconciliation Draft.",
  };
}

describe("Task 6 manager governed-checklist backend parity", () => {
  it("keeps a legacy source-gap Draft immutable while a current-source hybrid reconciliation follows the separate review path", async () => {
    const runtime = createMockBackendRuntime();
    const admin = runtime.backendForRole("admin").adminWorkspace;

    const legacyRun = await admin.importGovernedGenerationRun({
      operationId: "TASK6-LEGACY-CANDIDATE-IMPORT",
      idempotencyKey: "TASK6-LEGACY-CANDIDATE-IMPORT",
      candidateBundle: SYNTHETIC_LEGACY_CHECKLIST_CANDIDATE_BUNDLE,
    });
    const legacy = legacyRun.candidate!;
    expect(legacy.questions[0]).toEqual(expect.objectContaining({
      origin: "EXISTING_CHECKLIST_CANDIDATE",
      regulatoryTrace: expect.objectContaining({
        state: "SOURCE_MAPPING_REQUIRED",
        currentnessState: "SOURCE_MAPPING_REQUIRED",
        technicalReviewState: "NOT_AVAILABLE",
      }),
    }));
    await expect(admin.submitGovernedCandidateReview({
      operationId: "TASK6-LEGACY-CANDIDATE-SUBMIT",
      idempotencyKey: "TASK6-LEGACY-CANDIDATE-SUBMIT",
      candidateId: legacy.candidateId,
      expectedRevision: legacy.revision,
      expectedContentDigest: legacy.contentDigest,
      reason: "A source-gap candidate cannot enter department review.",
    })).rejects.toMatchObject({
      issues: [expect.objectContaining({ code: "SOURCE_MAPPING_REQUIRED" })],
    });

    await expect(admin.importGovernedGenerationRun({
      operationId: "TASK6-HYBRID-RECONCILIATION-RAW-V2-IMPORT",
      idempotencyKey: "TASK6-HYBRID-RECONCILIATION-RAW-V2-IMPORT",
      candidateBundle: SYNTHETIC_HYBRID_RECONCILED_BUNDLE,
    })).rejects.toMatchObject({
      issues: [expect.objectContaining({ code: "SOURCE_CURRENTNESS_REQUIRED" })],
    });
    const activation = await admin.activateGovernedSourceCurrentness(sourceImpactActivation("HYBRID-RECONCILIATION"));
    expect(activation).toMatchObject({ status: "IMPACT_REVIEW_DRAFT", impactReviewDraftId: expect.any(String) });

    const hybridRun = await admin.importGovernedGenerationRun({
      operationId: "TASK6-HYBRID-RECONCILIATION-IMPORT",
      idempotencyKey: "TASK6-HYBRID-RECONCILIATION-IMPORT",
      candidateBundle: SYNTHETIC_HYBRID_RECONCILED_BUNDLE,
    });
    const hybrid = hybridRun.candidate!;
    expect(hybrid.questions).toEqual(SYNTHETIC_HYBRID_RECONCILED_BUNDLE.inspectionChecklist.questions);
    expect(hybrid.questions[0]).toMatchObject({
      origin: "HYBRID_RECONCILED",
      reconciliation: {
        legacyQuestionId: "Q-SYNTHETIC-LEGACY-CANDIDATE-003",
        wordingChanged: true,
      },
    });
    expect(await admin.getGovernedCandidate({ candidateId: legacy.candidateId }))
      .toEqual(expect.objectContaining({
        status: "GENERATED_DRAFT",
        questions: [expect.objectContaining({
          origin: "EXISTING_CHECKLIST_CANDIDATE",
          regulatoryTrace: expect.objectContaining({
            state: "SOURCE_MAPPING_REQUIRED",
            currentnessState: "SOURCE_MAPPING_REQUIRED",
            technicalReviewState: "NOT_AVAILABLE",
          }),
        })],
      }));

    const submitted = await admin.submitGovernedCandidateReview({
      operationId: "TASK6-HYBRID-RECONCILIATION-SUBMIT",
      idempotencyKey: "TASK6-HYBRID-RECONCILIATION-SUBMIT",
      candidateId: hybrid.candidateId,
      expectedRevision: hybrid.revision,
      expectedContentDigest: hybrid.contentDigest,
      reason: "Submit the current-source reconciliation for technical review.",
    });
    const manager = runtime.backendForRole("manager").governedChecklistReview;
    const approved = await manager.approve(command("HYBRID-RECONCILIATION-APPROVE", submitted));
    const publication = await manager.publish(command("HYBRID-RECONCILIATION-PUBLISH", approved));
    expect(publication.candidateId).toBe(hybrid.candidateId);
    expect(await admin.getGovernedCandidate({ candidateId: legacy.candidateId }))
      .toEqual(expect.objectContaining({ status: "GENERATED_DRAFT" }));
  });

  it("projects attributed approval and source-impact currentness without rewriting immutable snapshots", async () => {
    const { runtime, submitted } = await submittedRuntime("SOURCE-IMPACT-PROJECTION");
    const admin = runtime.backendForRole("admin").adminWorkspace;
    const manager = runtime.backendForRole("manager").governedChecklistReview;
    const approved = await manager.approve(command("SOURCE-IMPACT-PROJECTION-APPROVE", submitted));

    expect(approved.questions[0]).toMatchObject({
      scopeRecommendation: { approvalReviewState: "TECHNICALLY_APPROVED" },
      regulatoryTrace: { currentnessState: "CURRENT", technicalReviewState: "TECHNICALLY_APPROVED" },
    });
    expect((await manager.getCandidate({ candidateId: approved.candidateId })).candidate.questions[0])
      .toMatchObject({
        scopeRecommendation: { approvalReviewState: "TECHNICALLY_APPROVED" },
        regulatoryTrace: { currentnessState: "CURRENT", technicalReviewState: "TECHNICALLY_APPROVED" },
      });
    expect((await admin.getGovernedGenerationRun({ generationRunId: approved.generationRunId! })).candidate)
      .toMatchObject({
        status: "TECHNICALLY_APPROVED",
        questions: [expect.objectContaining({
          regulatoryTrace: expect.objectContaining({ technicalReviewState: "TECHNICALLY_APPROVED" }),
        })],
      });

    const activation = await admin.activateGovernedSourceCurrentness(sourceImpactActivation("SOURCE-IMPACT-PROJECTION"));
    expect(activation).toMatchObject({ status: "IMPACT_REVIEW_DRAFT", impactReviewDraftId: expect.any(String) });
    const impactRun = await admin.importGovernedGenerationRun({
      operationId: "TASK6-SOURCE-IMPACT-PROJECTION-V2-IMPORT",
      idempotencyKey: "TASK6-SOURCE-IMPACT-PROJECTION-V2-IMPORT",
      candidateBundle: SYNTHETIC_HYBRID_RECONCILED_BUNDLE,
    });
    expect(impactRun.candidate?.questions[0]).toMatchObject({
      regulatoryTrace: { immutableVersion: "2", currentnessState: "CURRENT", technicalReviewState: "TECHNICAL_REVIEW_REQUIRED" },
    });
    const stalePrior = await admin.getGovernedCandidate({ candidateId: approved.candidateId });
    expect(stalePrior).toMatchObject({
      status: "TECHNICALLY_APPROVED",
      questions: [expect.objectContaining({
        scopeRecommendation: expect.objectContaining({
          approvalReviewState: "TECHNICALLY_APPROVED",
          guardrails: expect.objectContaining({ sourceChanged: true }),
        }),
        regulatoryTrace: expect.objectContaining({
          currentnessState: "STALE",
          technicalReviewState: "TECHNICALLY_APPROVED",
        }),
      })],
    });
    expect((await manager.getCandidate({ candidateId: approved.candidateId })).candidate)
      .toMatchObject({
        status: "TECHNICALLY_APPROVED",
        questions: [expect.objectContaining({
          regulatoryTrace: expect.objectContaining({ currentnessState: "STALE" }),
        })],
      });
    expect((await admin.getGovernedGenerationRun({ generationRunId: approved.generationRunId! })).candidate)
      .toMatchObject({
        status: "TECHNICALLY_APPROVED",
        questions: [expect.objectContaining({
          regulatoryTrace: expect.objectContaining({ currentnessState: "STALE" }),
        })],
      });
    expect((await manager.listQueue({})).items).toEqual(expect.arrayContaining([
      expect.objectContaining({
        candidate: expect.objectContaining({
          candidateId: approved.candidateId,
          questions: [expect.objectContaining({
            regulatoryTrace: expect.objectContaining({ currentnessState: "STALE" }),
          })],
        }),
        blockingIssues: expect.arrayContaining([expect.objectContaining({ code: "STALE_SOURCE_TRACE" })]),
      }),
    ]));
    await expect(manager.publish(command("SOURCE-IMPACT-PROJECTION-PUBLISH", approved)))
      .rejects.toMatchObject({ issues: [expect.objectContaining({ code: "STALE_SOURCE_TRACE" })] });
    expect((await admin.listAuditEvents({ entity: impactRun.candidate!.candidateId })).items)
      .toEqual(expect.arrayContaining([
        expect.objectContaining({ action: "regulatory.source_impact_candidate_bound" }),
      ]));
  });

  it("rejects a self-transition so mock source activation remains identical to the Go and HTTP boundary", async () => {
    const runtime = createMockBackendRuntime();
    const admin = runtime.backendForRole("admin").adminWorkspace;
    await expect(admin.activateGovernedSourceCurrentness({
      operationId: "TASK6-SOURCE-CURRENTNESS-SELF-TRANSITION",
      idempotencyKey: "TASK6-SOURCE-CURRENTNESS-SELF-TRANSITION",
      currentSourceSnapshotId: "SOURCE-SYNTHETIC-OPS-AOC",
      currentSourceHash: SYNTHETIC_GOVERNED_BUNDLE.sourceCurrentness!.currentSourceHash,
      previousSourceSnapshotId: "SOURCE-SYNTHETIC-OPS-AOC",
      previousSourceHash: SYNTHETIC_GOVERNED_BUNDLE.sourceCurrentness!.currentSourceHash,
      reason: "A self-transition must be rejected before any impact Draft can exist.",
    })).rejects.toMatchObject({
      issues: [expect.objectContaining({ code: "SOURCE_CURRENTNESS_INVALID" })],
    });

    const missingPredecessor = {
      operationId: "TASK6-SOURCE-CURRENTNESS-MISSING-PREDECESSOR",
      idempotencyKey: "TASK6-SOURCE-CURRENTNESS-MISSING-PREDECESSOR",
      currentSourceSnapshotId: "SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2",
      currentSourceHash: SYNTHETIC_HYBRID_RECONCILED_BUNDLE.sourceCurrentness!.currentSourceHash,
      reason: "The canonical contract requires explicit predecessor fields.",
    };
    await expect(admin.activateGovernedSourceCurrentness(missingPredecessor as never)).rejects.toMatchObject({
      issues: [expect.objectContaining({ code: "SOURCE_CURRENTNESS_INVALID" })],
    });
  });

  it("replays the pre-seeded V1 source-currentness activation with its immutable receipt", async () => {
    const runtime = createMockBackendRuntime();
    const admin = runtime.backendForRole("admin").adminWorkspace;
    const input = {
      operationId: "TESTPROFILE-SOURCE-CURRENTNESS-BASELINE-V1",
      idempotencyKey: "TESTPROFILE-SOURCE-CURRENTNESS-BASELINE-V1",
      currentSourceSnapshotId: "SOURCE-SYNTHETIC-OPS-AOC",
      currentSourceHash: SYNTHETIC_GOVERNED_BUNDLE.sourceCurrentness!.currentSourceHash,
      previousSourceSnapshotId: null,
      previousSourceHash: null,
      reason: "Synthetic internal test-profile baseline currentness declaration.",
    };

    const replay = await admin.activateGovernedSourceCurrentness(input);
    expect(replay).toEqual(expect.objectContaining({
      eventId: "SRC-CURRENTNESS-TESTPROFILE-BASELINE-V1",
      impactReviewDraftId: null,
      status: "BASELINE_ACTIVATED",
      currentSourceSnapshotId: input.currentSourceSnapshotId,
      currentSourceHash: input.currentSourceHash,
    }));
  });

  it("keeps V1 and V2 Regulatory Library links bound to their exact source/version identities", async () => {
    const runtime = createMockBackendRuntime();
    const admin = runtime.backendForRole("admin").adminWorkspace;
    const baselineSources = await admin.listGovernedSources({});
    const baselineV1 = baselineSources.items.find((item) => item.clauseId === "CLAUSE-SYNTHETIC-OPS-AOC-1");
    expect(baselineV1).toEqual(expect.objectContaining({
      sourceId: "SOURCE-SYNTHETIC-OPS-AOC",
      versionIdentity: "1",
      sourceHash: SYNTHETIC_GOVERNED_BUNDLE.generationRequest.sourceSnapshots[0]!.sourceHash,
      generationRunIds: [SYNTHETIC_GOVERNED_BUNDLE.generationRunId],
      candidateIds: [SYNTHETIC_GOVERNED_BUNDLE.candidateBundleId],
    }));

    await admin.activateGovernedSourceCurrentness(sourceImpactActivation("SOURCE-LIBRARY-V2"));
    const impact = await admin.importGovernedGenerationRun({
      operationId: "TASK6-SOURCE-LIBRARY-V2-IMPORT",
      idempotencyKey: "TASK6-SOURCE-LIBRARY-V2-IMPORT",
      candidateBundle: SYNTHETIC_HYBRID_RECONCILED_BUNDLE,
    });
    const sources = await admin.listGovernedSources({});
    const v1 = sources.items.find((item) => item.clauseId === "CLAUSE-SYNTHETIC-OPS-AOC-1");
    const v2 = sources.items.find((item) => item.clauseId === "CLAUSE-SYNTHETIC-OPS-AOC-IMPACT-2");
    expect(v1).toEqual(expect.objectContaining({
      sourceId: "SOURCE-SYNTHETIC-OPS-AOC",
      versionIdentity: "1",
      generationRunIds: [SYNTHETIC_GOVERNED_BUNDLE.generationRunId],
      candidateIds: [SYNTHETIC_GOVERNED_BUNDLE.candidateBundleId],
    }));
    expect(v2).toEqual(expect.objectContaining({
      sourceId: "SOURCE-SYNTHETIC-OPS-AOC-IMPACT-V2",
      versionIdentity: "2",
      sourceHash: SYNTHETIC_HYBRID_RECONCILED_BUNDLE.generationRequest.sourceSnapshots[0]!.sourceHash,
      clauseLocator: "Synthetic OPS/AOC impact 2",
      partitions: [expect.objectContaining({ partitionId: "PARTITION-SYNTHETIC-IMPACT-INPUT", role: "GENERATION_INPUT" })],
      generationRunIds: [impact.generationRunId],
      candidateIds: [impact.candidate!.candidateId],
      applicabilityFacts: [expect.objectContaining({ candidateId: impact.candidate!.candidateId, mappingId: "MAP-SYNTHETIC-HYBRID-RECONCILED-004" })],
    }));
  });

  it("executes the shared exact artifact parity contract against the semantic mock", async () => {
    const contract = loadTask6ArtifactParityContract();
    expect(contract.contractId).toBe("task6-manager-artifact-parity-v1");
    const store = MemoryMockStore.createCanonical({ clock: () => contract.clock });
    const admin = createMockBackend({
      store,
      principal: { subjectId: "USR-ADMIN-ADA", role: "admin", organizationId: null },
      governedRequiredOwners: contract.requiredOwners,
    }).adminWorkspace;
    const imported = await admin.importGovernedGenerationRun({
      ...contract.operations.import,
      candidateBundle: SYNTHETIC_GOVERNED_BUNDLE,
    });
    const submitted = await admin.submitGovernedCandidateReview({
      ...contract.operations.submit,
      candidateId: imported.candidate!.candidateId,
      expectedRevision: imported.candidate!.revision,
      expectedContentDigest: imported.candidate!.contentDigest,
    });
    expect({
      candidateId: submitted.candidateId,
      candidateRootId: submitted.candidateRootId,
      revision: submitted.revision,
      contentDigest: submitted.contentDigest,
    }).toEqual({
      candidateId: contract.candidate.candidateId,
      candidateRootId: contract.candidate.candidateRootId,
      revision: contract.candidate.revision,
      contentDigest: contract.candidate.contentDigest,
    });
    const foi = createMockBackend({
      store,
      principal: {
        subjectId: contract.actors.foi.subjectId,
        role: "manager",
        organizationId: null,
      },
    }).governedChecklistReview;
    const air = createMockBackend({
      store,
      principal: {
        subjectId: contract.actors.air.subjectId,
        role: "manager",
        organizationId: null,
      },
    }).governedChecklistReview;
    const reviewInput = (
      operation: Task6ArtifactParityContract["operations"]["foiApproval"],
      candidate: GovernedCandidateView,
    ) => ({
      ...operation,
      candidateId: candidate.candidateId,
      expectedRevision: candidate.revision,
      expectedContentDigest: candidate.contentDigest,
    });
    const baselineAuditEventIds = new Set(
      (await admin.listAuditEvents({})).items.map((event) => event.eventId),
    );
    const publicationRows: Task6PublicationDecisionArtifact[] = [];
    const checklistVersionRows: Task6ChecklistVersionArtifact[] = [];
    const observedArtifacts = async (
      decisions: GovernedReviewDecisionView[],
    ): Promise<Task6ArtifactRows> => {
      const reviewDecisions = decisions.map((decision) => ({
        ...decision,
        actorMembershipIsCurrent: true,
      })).sort((left, right) => left.decisionId.localeCompare(right.decisionId));
      const auditEvents = (await admin.listAuditEvents({})).items
        .filter((event) => !baselineAuditEventIds.has(event.eventId));
      return {
        reviewDecisions,
        auditEvents: task6MockAuditArtifacts(
          auditEvents,
          submitted,
          reviewDecisions,
          publicationRows,
        ),
        publicationDecisions: structuredClone(publicationRows)
          .sort((left, right) =>
            left.publicationDecisionId.localeCompare(right.publicationDecisionId)),
        checklistVersions: structuredClone(checklistVersionRows)
          .sort((left, right) =>
            left.templateVersionId.localeCompare(right.templateVersionId)),
      };
    };

    const foiInput = reviewInput(contract.operations.foiApproval, submitted);
    const partial = await foi.approve(foiInput);
    expect(partial.status).toBe(contract.expected.partialStatus);
    await expect(foi.approve(foiInput)).resolves.toEqual(partial);
    const approvalOnly = await foi.getCandidate({ candidateId: partial.candidateId });
    await expect(foi.getPublishedVersion({
      templateVersionId: contract.candidate.absentTemplateVersionId,
    })).rejects.toThrow(/not found/i);
    expect(await observedArtifacts(approvalOnly.decisions)).toEqual(
      task6ExpectedArtifactCheckpoint(contract, "approvalOnly"),
    );
    await expect(foi.approve({
      ...foiInput,
      reason: contract.operations.conflictingReplayReason,
    })).rejects.toThrow(/different semantics/i);
    expect(await observedArtifacts(approvalOnly.decisions)).toEqual(
      task6ExpectedArtifactCheckpoint(contract, "approvalOnly"),
    );

    const airInput = reviewInput(contract.operations.airApproval, partial);
    const complete = await air.approve(airInput);
    expect(complete.status).toBe(contract.expected.finalStatus);
    const completeDetail = await air.getCandidate({ candidateId: complete.candidateId });
    expect(await observedArtifacts(completeDetail.decisions)).toEqual(
      task6ExpectedArtifactCheckpoint(contract, "jointComplete"),
    );

    await expect(foi.publish({
      ...contract.operations.digestTamper,
      candidateId: complete.candidateId,
      expectedRevision: complete.revision,
      expectedContentDigest: contract.candidate.tamperedDigest,
    })).rejects.toThrow(/stale governed candidate/i);
    expect(await observedArtifacts(completeDetail.decisions)).toEqual(
      task6ExpectedArtifactCheckpoint(contract, "jointComplete"),
    );
    const publishInput = reviewInput(contract.operations.publication, complete);
    const publication = await foi.publish(publishInput);
    const [expectedPublicationRow] =
      contract.expected.artifactRows.publicationDecisions;
    if (!expectedPublicationRow) throw new Error("Task 6 publication row is absent.");
    const {
      actorMembershipIsCurrent: expectedMembershipIsCurrent,
      createdAt: expectedCreatedAt,
      ...expectedPublication
    } = expectedPublicationRow;
    expect(expectedMembershipIsCurrent).toBe(true);
    expect(expectedCreatedAt).toBe(publication.decidedAt);
    expect(publication).toEqual(expectedPublication);
    publicationRows.push({
      ...publication,
      actorMembershipIsCurrent: true,
      createdAt: publication.decidedAt,
    });
    expect((await foi.getCandidate({
      candidateId: complete.candidateId,
    })).candidate.status).toBe(contract.expected.publishedStatus);
    await expect(foi.publish(publishInput)).resolves.toEqual(publication);
    const artifact = await foi.getPublishedVersion({
      templateVersionId: publication.templateVersionId,
    });
    expect({
      mappings: artifact.mappings,
      questions: artifact.questions,
    }).toEqual(contract.publishedArtifact);
    expect(JSON.stringify({
      mappings: artifact.mappings,
      questions: artifact.questions,
    })).toBe(JSON.stringify(contract.publishedArtifact));
    checklistVersionRows.push({
      templateVersionId: publication.templateVersionId,
      templateId: complete.templateId,
      version: checklistVersionRows.length + 1,
      title: `Governed ${complete.templateId}`,
      publishedAt: publication.publishedAt,
      candidateRootId: publication.candidateRootId,
      candidateId: publication.candidateId,
      candidateRevision: publication.candidateRevision,
      candidateContentDigest: publication.candidateContentDigest,
      publicationDecisionId: publication.publicationDecisionId,
      auditEventId: publication.auditEventId,
      questionVersionOrder: artifact.questions.map((question, position) => ({
        questionVersionId: `QV-${question.questionId}-V${complete.revision}`,
        position,
      })),
      immutableSnapshot: {
        candidateId: publication.candidateId,
        candidateRevision: publication.candidateRevision,
        candidateContentDigest: publication.candidateContentDigest,
        complianceMappings: structuredClone(artifact.mappings),
        questions: structuredClone(artifact.questions),
      },
    });
    artifact.mappings[0]!.mappingId = "TAMPERED-CLONE";
    await expect(foi.getPublishedVersion({
      templateVersionId: publication.templateVersionId,
    })).resolves.toEqual({
      publication,
      ...contract.publishedArtifact,
    });
    await expect(foi.publish({
      ...publishInput,
      reason: contract.operations.conflictingReplayReason,
    })).rejects.toThrow(/different semantics/i);
    expect(await observedArtifacts(completeDetail.decisions)).toEqual(
      task6ExpectedArtifactCheckpoint(contract, "published"),
    );
    expect("updatePublishedVersion" in foi).toBe(false);
    expect("deletePublishedVersion" in foi).toBe(false);
  });

  it("exposes exact immutable review/publication identities and ordered published snapshots", async () => {
    const { runtime, submitted } = await submittedRuntime("SEMANTIC-ARTIFACT");
    const manager = runtime.backendForRole("manager").governedChecklistReview;
    const approveInput = command("SEMANTIC-ARTIFACT-APPROVE", submitted);
    const approved = await manager.approve(approveInput);
    const approvalSemantic = await governedCanonicalSHA256({
      command: "TECHNICALLY_APPROVED",
      operationId: approveInput.operationId,
      candidateId: approveInput.candidateId,
      expectedRevision: approveInput.expectedRevision,
      expectedContentDigest: approveInput.expectedContentDigest,
      reason: approveInput.reason,
    });
    await expect(manager.getCandidate({ candidateId: approved.candidateId })).resolves
      .toEqual(expect.objectContaining({
        decisions: [{
          decisionId: `DRD-${approveInput.operationId}`,
          decision: "TECHNICALLY_APPROVED",
          candidateRootId: approved.candidateRootId,
          candidateId: approved.candidateId,
          candidateRevision: approved.revision,
          candidateContentDigest: approved.contentDigest,
          actorSubjectId: "USR-MANAGER-NORA",
          actorDepartmentMembershipId: "MEM-TASK6-NORA",
          actorDepartmentId: "FLIGHT_OPERATIONS_INSPECTORATE",
          actorOrganizationalUnitId: "FLIGHT_OPERATIONS_INSPECTORATE",
          reason: approveInput.reason,
          decidedAt: "2026-06-15T09:00:00.000Z",
          operationId: approveInput.operationId,
          idempotencyKey: approveInput.idempotencyKey,
          semanticPayloadDigest: approvalSemantic,
          auditEventId: `AE-${approveInput.operationId}`,
        }],
      }));
    await expect(manager.getPublishedVersion({
      templateVersionId: "CTV-GOV-APPROVAL-ONLY-ABSENT",
    })).rejects.toThrow(/not found/i);

    const publishInput = command("SEMANTIC-ARTIFACT-PUBLISH", approved);
    const publication = await manager.publish(publishInput);
    const publicationSemantic = await governedCanonicalSHA256({
      command: "PUBLISHED",
      operationId: publishInput.operationId,
      candidateId: publishInput.candidateId,
      expectedRevision: publishInput.expectedRevision,
      expectedContentDigest: publishInput.expectedContentDigest,
      reason: publishInput.reason,
    });
    expect(publication).toEqual({
      templateVersionId: expect.stringMatching(/^CTV-GOV-/),
      publicationDecisionId: `PUBDEC-${publishInput.operationId}`,
      candidateRootId: approved.candidateRootId,
      candidateId: approved.candidateId,
      candidateRevision: approved.revision,
      candidateContentDigest: approved.contentDigest,
      actorSubjectId: "USR-MANAGER-NORA",
      actorDepartmentMembershipId: "MEM-TASK6-NORA",
      actorDepartmentId: "FLIGHT_OPERATIONS_INSPECTORATE",
      actorOrganizationalUnitId: "FLIGHT_OPERATIONS_INSPECTORATE",
      reason: publishInput.reason,
      decidedAt: "2026-06-15T09:00:00.000Z",
      publishedAt: "2026-06-15T09:00:00.000Z",
      operationId: publishInput.operationId,
      idempotencyKey: publishInput.idempotencyKey,
      semanticPayloadDigest: publicationSemantic,
      auditEventId: `AE-${publishInput.operationId}`,
    });
    await expect(manager.publish(publishInput)).resolves.toEqual(publication);
    const artifact = await manager.getPublishedVersion({
      templateVersionId: publication.templateVersionId,
    });
    const immutableQuestions = structuredClone(submitted.questions);
    for (const question of immutableQuestions) {
      Reflect.deleteProperty(question.regulatoryTrace, "mappingReviewState");
    }
    expect(artifact).toEqual({
      publication,
      mappings: submitted.mappings,
      questions: immutableQuestions,
    });
    const exactBytes = JSON.stringify({
      mappings: artifact.mappings,
      questions: artifact.questions,
    });
    expect(exactBytes).toBe(JSON.stringify({
      mappings: submitted.mappings,
      questions: immutableQuestions,
    }));
    artifact.mappings[0]!.mappingId = "TAMPERED-CLONE";
    expect((await manager.getPublishedVersion({
      templateVersionId: publication.templateVersionId,
    })).mappings).toEqual(approved.mappings);
    await expect(manager.publish({
      ...publishInput,
      reason: "Conflicting command-level tamper must have no effect.",
    })).rejects.toThrow(/different semantics/i);
    await expect(manager.getPublishedVersion({
      templateVersionId: publication.templateVersionId,
    })).resolves.toEqual({
      publication,
      mappings: submitted.mappings,
      questions: immutableQuestions,
    });
    expect("updatePublishedVersion" in manager).toBe(false);
    expect("deletePublishedVersion" in manager).toBe(false);
  });

  it("keeps the exact real controlled-procedure OPS/AOC profile blocked before candidate creation", async () => {
    const runtime = createMockBackendRuntime();
    const admin = runtime.backendForRole("admin").adminWorkspace;
    const manager = runtime.backendForRole("manager").governedChecklistReview;
    const blocked = await manager.validateBlockedGeneration({
      operationId: "TASK6-MOCK-BLOCKED-VALIDATE",
      idempotencyKey: "TASK6-MOCK-BLOCKED-VALIDATE",
      generationRequest: EXACT_BLOCKED_REAL_OPS_AOC_REQUEST,
    });
    expect(blocked).toEqual({
      status: "BLOCKED",
      requestId: "GENREQ-OPS-AOC-0001",
      blockingIssues: EXACT_BLOCKED_REAL_OPS_AOC_REQUEST.unresolvedSourceGaps,
      effectCounts: {
        generationRuns: 0,
        candidates: 0,
        reviewDecisions: 0,
        publicationDecisions: 0,
        checklistVersions: 0,
        auditEvents: 0,
      },
    });
    await expect(admin.getGovernedGenerationRun({
      generationRunId: "GENRUN-REAL-OPS-AOC-BLOCKED",
    })).rejects.toThrow(/not found/i);
    await expect(manager.listQueue({})).resolves.toEqual({ items: [] });
    const absentCandidate = {
      operationId: "TASK6-MOCK-REAL-APPROVE",
      idempotencyKey: "TASK6-MOCK-REAL-APPROVE",
      candidateId: "CAND-REAL-OPS-AOC-BLOCKED",
      expectedRevision: 1,
      expectedContentDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      reason: blocked.blockingIssues.map((issue) => issue.gapId).join(","),
    };
    await expect(manager.approve(absentCandidate)).rejects.toThrow(/stale governed candidate/i);
    await expect(manager.publish({
      ...absentCandidate,
      operationId: "TASK6-MOCK-REAL-PUBLISH",
      idempotencyKey: "TASK6-MOCK-REAL-PUBLISH",
    })).rejects.toThrow(/stale governed candidate/i);
    await expect(admin.getGovernedGenerationRun({
      generationRunId: "GENRUN-REAL-OPS-AOC-BLOCKED",
    })).rejects.toThrow(/not found/i);
  });

  it("resolves the global latest membership successor before active filtering", async () => {
    const store = MemoryMockStore.createCanonical({
      clock: () => "2026-06-15T09:00:00.000Z",
    });
    const revoked = createMockBackend({
      store,
      principal: { subjectId: "USR-MANAGER-REVOKED", role: "manager", organizationId: null },
    }).governedChecklistReview;
    await expect(revoked.listQueue({}))
      .rejects.toBeInstanceOf(BackendAuthorizationInvariantError);
    await expect(revoked.validateBlockedGeneration({
      operationId: "TASK6-MOCK-REVOKED-BLOCKED-VALIDATE",
      idempotencyKey: "TASK6-MOCK-REVOKED-BLOCKED-VALIDATE",
      generationRequest: EXACT_BLOCKED_REAL_OPS_AOC_REQUEST,
    }))
      .rejects.toBeInstanceOf(BackendAuthorizationInvariantError);
  });

  it("denies role-only and cross-department managers before queue, detail, or action effects", async () => {
    const store = MemoryMockStore.createCanonical({
      clock: () => "2026-06-15T09:00:00.000Z",
    });
    const admin = createMockBackend({
      store,
      principal: { subjectId: "USR-ADMIN-ADA", role: "admin", organizationId: null },
    }).adminWorkspace;
    const imported = await admin.importGovernedGenerationRun({
      operationId: "TASK6-MOCK-AUTH-IMPORT",
      idempotencyKey: "TASK6-MOCK-AUTH-IMPORT",
      candidateBundle: SYNTHETIC_GOVERNED_BUNDLE,
    });
    const submitted = await admin.submitGovernedCandidateReview({
      operationId: "TASK6-MOCK-AUTH-SUBMIT",
      idempotencyKey: "TASK6-MOCK-AUTH-SUBMIT",
      candidateId: imported.candidate!.candidateId,
      expectedRevision: imported.candidate!.revision,
      expectedContentDigest: imported.candidate!.contentDigest,
      reason: "Submit exact synthetic candidate for authority denial.",
    });
    const roleOnly = createMockBackend({
      store,
      principal: { subjectId: "USR-MANAGER-ROLE-ONLY", role: "manager", organizationId: null },
    }).governedChecklistReview;
    await expect(roleOnly.listQueue({})).rejects.toBeInstanceOf(BackendAuthorizationInvariantError);
    await expect(roleOnly.getCandidate({ candidateId: submitted.candidateId }))
      .rejects.toBeInstanceOf(BackendAuthorizationInvariantError);
    await expect(roleOnly.approve(command("DENIED-ROLE-ONLY", submitted)))
      .rejects.toBeInstanceOf(BackendAuthorizationInvariantError);

    const crossDepartment = createMockBackend({
      store,
      principal: { subjectId: "USR-MANAGER-AIR", role: "manager", organizationId: null },
    }).governedChecklistReview;
    await expect(crossDepartment.listQueue({})).resolves.toEqual({ items: [] });
    await expect(crossDepartment.getCandidate({ candidateId: submitted.candidateId }))
      .rejects.toBeInstanceOf(BackendAuthorizationInvariantError);
    await expect(crossDepartment.approve(command("DENIED-CROSS-DEPARTMENT", submitted)))
      .rejects.toBeInstanceOf(BackendAuthorizationInvariantError);
    const authorized = createMockBackend({
      store,
      principal: { subjectId: "USR-MANAGER-NORA", role: "manager", organizationId: null },
    }).governedChecklistReview;
    await expect(authorized.listQueue({})).resolves.toEqual({
      items: [expect.objectContaining({
        candidate: expect.objectContaining({ candidateId: submitted.candidateId }),
      })],
    });
  });

  it("keeps exact joint-owner approvals partial until every current owner approves", async () => {
    const store = MemoryMockStore.createCanonical({
      clock: () => "2026-06-15T09:00:00.000Z",
    });
    const requiredOwners = [
      {
        departmentId: "FLIGHT_OPERATIONS_INSPECTORATE",
        organizationalUnitId: "FLIGHT_OPERATIONS_INSPECTORATE",
        approvalRequired: true,
      },
      {
        departmentId: "AIRWORTHINESS_INSPECTORATE",
        organizationalUnitId: "AIRWORTHINESS_INSPECTORATE",
        approvalRequired: true,
      },
    ];
    const admin = createMockBackend({
      store,
      principal: { subjectId: "USR-ADMIN-ADA", role: "admin", organizationId: null },
      governedRequiredOwners: requiredOwners,
    }).adminWorkspace;
    const imported = await admin.importGovernedGenerationRun({
      operationId: "TASK6-MOCK-JOINT-IMPORT",
      idempotencyKey: "TASK6-MOCK-JOINT-IMPORT",
      candidateBundle: SYNTHETIC_GOVERNED_BUNDLE,
    });
    const submitted = await admin.submitGovernedCandidateReview({
      operationId: "TASK6-MOCK-JOINT-SUBMIT",
      idempotencyKey: "TASK6-MOCK-JOINT-SUBMIT",
      candidateId: imported.candidate!.candidateId,
      expectedRevision: imported.candidate!.revision,
      expectedContentDigest: imported.candidate!.contentDigest,
      reason: "Submit exact joint-owner candidate.",
    });
    const foi = createMockBackend({
      store,
      principal: { subjectId: "USR-MANAGER-NORA", role: "manager", organizationId: null },
    }).governedChecklistReview;
    const air = createMockBackend({
      store,
      principal: { subjectId: "USR-MANAGER-AIR", role: "manager", organizationId: null },
    }).governedChecklistReview;
    const partial = await foi.approve(command("JOINT-FOI", submitted));
    expect(partial.status).toBe("DEPARTMENT_REVIEW");
    const partialDetail = await foi.getCandidate({ candidateId: submitted.candidateId });
    expect(partialDetail.requiredOwners).toEqual(requiredOwners);
    expect(partialDetail.decisions).toEqual([
      expect.objectContaining({
        actorDepartmentId: "FLIGHT_OPERATIONS_INSPECTORATE",
        decision: "TECHNICALLY_APPROVED",
      }),
    ]);
    const complete = await air.approve(command("JOINT-AIR", partial));
    expect(complete.status).toBe("TECHNICALLY_APPROVED");
    expect((await air.getCandidate({ candidateId: complete.candidateId })).decisions)
      .toHaveLength(2);
  });

  it("fails closed before review submission when persisted governance blockers remain", async () => {
    const store = MemoryMockStore.createCanonical({
      clock: () => "2026-06-15T09:00:00.000Z",
    });
    const blockingIssues = [
      {
        fieldPath: "sourceSnapshots[0]",
        code: "UNRESOLVED_SOURCE_GAP",
        message: "Controlled procedure remains unresolved.",
        sourceIdentity: "SYNTHETIC-OPS-AOC",
        sourceHash: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
        clauseId: "CLAUSE-SYNTHETIC-OPS-AOC-1",
        locator: "Synthetic OPS/AOC 1",
      },
      {
        fieldPath: "sourceSnapshots[0].sourceHash",
        code: "SOURCE_HASH_MISMATCH",
        message: "Persisted source hash does not match.",
        sourceIdentity: "SYNTHETIC-OPS-AOC",
        sourceHash: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
        clauseId: "CLAUSE-SYNTHETIC-OPS-AOC-1",
        locator: "Synthetic OPS/AOC 1",
      },
      {
        fieldPath: "generationRunId",
        code: "MISSING_GENERATION_LINEAGE",
        message: "Exact generation lineage is incomplete.",
        sourceIdentity: "SYNTHETIC-OPS-AOC",
        sourceHash: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
        clauseId: "CLAUSE-SYNTHETIC-OPS-AOC-1",
        locator: "Synthetic OPS/AOC 1",
      },
      {
        fieldPath: "requiredOwners",
        code: "OWNER_SET_MISMATCH",
        message: "Exact required owner set is incomplete.",
        sourceIdentity: "SYNTHETIC-OPS-AOC",
        sourceHash: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
        clauseId: "CLAUSE-SYNTHETIC-OPS-AOC-1",
        locator: "Synthetic OPS/AOC 1",
      },
    ];
    const admin = createMockBackend({
      store,
      principal: { subjectId: "USR-ADMIN-ADA", role: "admin", organizationId: null },
      governedBlockingIssues: blockingIssues,
    }).adminWorkspace;
    const imported = await admin.importGovernedGenerationRun({
      operationId: "TASK6-MOCK-BLOCKER-IMPORT",
      idempotencyKey: "TASK6-MOCK-BLOCKER-IMPORT",
      candidateBundle: SYNTHETIC_GOVERNED_BUNDLE,
    });
    const draft = imported.candidate!;
    await expect(admin.submitGovernedCandidateReview({
      operationId: "TASK6-MOCK-BLOCKER-SUBMIT",
      idempotencyKey: "TASK6-MOCK-BLOCKER-SUBMIT",
      candidateId: draft.candidateId,
      expectedRevision: draft.revision,
      expectedContentDigest: draft.contentDigest,
      reason: "Submit blocked synthetic candidate.",
    })).rejects.toBeInstanceOf(GovernedValidationError);
    await expect(admin.submitGovernedCandidateReview({
      operationId: "TASK6-MOCK-BLOCKER-SUBMIT-ISSUES",
      idempotencyKey: "TASK6-MOCK-BLOCKER-SUBMIT-ISSUES",
      candidateId: draft.candidateId,
      expectedRevision: draft.revision,
      expectedContentDigest: draft.contentDigest,
      reason: "Submit blocked synthetic candidate with inspectable denials.",
    })).rejects.toMatchObject({
      issues: blockingIssues,
    });
    const unchanged = await admin.getGovernedCandidate({ candidateId: draft.candidateId });
    expect(unchanged).toEqual(expect.objectContaining({ status: "GENERATED_DRAFT" }));
    expect((await admin.listAuditEvents({ entity: draft.candidateId })).items).toEqual([]);
  });

  it("keeps manager queue, technical approval, and publication as separate exact steps", async () => {
    const { runtime, submitted } = await submittedRuntime("APPROVAL");
    const manager = runtime.backendForRole("manager").governedChecklistReview;

    await expect(manager.listQueue({})).resolves.toEqual({
      items: [expect.objectContaining({
        candidate: expect.objectContaining({
          candidateId: submitted.candidateId,
          status: "DEPARTMENT_REVIEW",
        }),
        requiredOwners: submitted.requiredOwners,
        decisions: [],
        blockingIssues: [],
      })],
    });
    const approved = await manager.approve(command("APPROVE", submitted));
    expect(approved.status).toBe("TECHNICALLY_APPROVED");
    expect(approved.questions[0]).toMatchObject({
      scopeRecommendation: { approvalReviewState: "TECHNICALLY_APPROVED" },
      regulatoryTrace: { technicalReviewState: "TECHNICALLY_APPROVED" },
    });
    expect(await manager.getCandidate({ candidateId: submitted.candidateId }))
      .toEqual(expect.objectContaining({
        candidate: expect.objectContaining({
          status: "TECHNICALLY_APPROVED",
          questions: [expect.objectContaining({
            scopeRecommendation: expect.objectContaining({ approvalReviewState: "TECHNICALLY_APPROVED" }),
            regulatoryTrace: expect.objectContaining({ technicalReviewState: "TECHNICALLY_APPROVED" }),
          })],
        }),
        decisions: [expect.objectContaining({
          decision: "TECHNICALLY_APPROVED",
          actorSubjectId: "USR-MANAGER-NORA",
        })],
      }));

    const publication = await manager.publish(command("PUBLISH", approved));
    expect(publication).toEqual(expect.objectContaining({
      candidateId: approved.candidateId,
      candidateRevision: approved.revision,
      candidateContentDigest: approved.contentDigest,
      templateVersionId: expect.stringMatching(/^CTV-GOV-/),
      publicationDecisionId: expect.stringMatching(/^PUBDEC-/),
    }));
    await expect(manager.publish(command("PUBLISH", approved))).resolves.toEqual(publication);
    await expect(manager.listQueue({})).resolves.toEqual({ items: [] });
    const audits = await runtime.backendForRole("admin").adminWorkspace.listAuditEvents({
      entity: approved.candidateId,
    });
    expect(audits.items).toEqual(expect.arrayContaining([
      expect.objectContaining({
        actorSubjectId: "USR-MANAGER-NORA",
        action: "TECHNICAL_APPROVAL_RECORDED",
        entityId: approved.candidateId,
        reason: "Task 6 approve decision.",
        entityRevision: approved.revision,
      }),
      expect.objectContaining({
        actorSubjectId: "USR-MANAGER-NORA",
        action: "CHECKLIST_PUBLISHED",
        entityId: approved.candidateId,
        reason: "Task 6 publish decision.",
        entityRevision: approved.revision,
      }),
    ]));
  });

  it("keeps synthetic test-profile package materialization separate from publication", async () => {
    const { runtime, submitted } = await submittedRuntime("SYNTHETIC-MATERIALIZATION");
    const manager = runtime.backendForRole("manager").governedChecklistReview;
    const approved = await manager.approve(command("SYNTHETIC-MATERIALIZATION-APPROVE", submitted));
    const publication = await manager.publish(command("SYNTHETIC-MATERIALIZATION-PUBLISH", approved));
    const inspector = runtime.backendForRole("inspector");

    await expect(inspector.inspections.getPackage({
      packageId: "PKG-SYNTHETIC-OPS-AOC-001",
    })).rejects.toThrow(/was not found/i);

    expect(runtime.materializeSyntheticGovernedPackageForTest()).toMatchObject({
      inspectionId: "AUD-SYNTHETIC-OPS-AOC-001",
      packageId: "PKG-SYNTHETIC-OPS-AOC-001",
      templateVersionId: publication.templateVersionId,
      packageDigest: publication.candidateContentDigest,
      selection: {
        organizationId: "ORG-SYNTHETIC-AOC",
        inspectionType: "RAMP_INSPECTION",
        targetId: "TARGET-SYNTHETIC-AOC",
        targetKind: "ORGANIZATION",
        departmentId: "FLIGHT_OPERATIONS_INSPECTORATE",
      },
    });
    await expect(inspector.inspections.getPackage({
      packageId: "PKG-SYNTHETIC-OPS-AOC-001",
    })).resolves.toMatchObject({
      auditId: "AUD-SYNTHETIC-OPS-AOC-001",
      templateVersionId: publication.templateVersionId,
      packageDigest: publication.candidateContentDigest,
      questions: [{ id: "Q-SYNTHETIC-OPS-AOC-001" }],
    });
  });

  it.each([
    ["return", "RETURNED"],
    ["reject", "REJECTED"],
  ] as const)("persists an attributed %s transition", async (action, status) => {
    const { runtime, submitted } = await submittedRuntime(action.toUpperCase());
    const manager = runtime.backendForRole("manager").governedChecklistReview;
    const result = await manager[action](command(action.toUpperCase(), submitted));
    expect(result.status).toBe(status);
    const detail = await manager.getCandidate({ candidateId: submitted.candidateId });
    expect(detail.decisions).toEqual([
      expect.objectContaining({
        decision: status,
        actorSubjectId: "USR-MANAGER-NORA",
        actorDepartmentId: "FLIGHT_OPERATIONS_INSPECTORATE",
        actorOrganizationalUnitId: "FLIGHT_OPERATIONS_INSPECTORATE",
      }),
    ]);
    if (status === "RETURNED") {
      await expect(manager.listQueue({})).resolves.toEqual(expect.objectContaining({
        items: [expect.objectContaining({ candidate: expect.objectContaining({ status: "RETURNED" }) })],
      }));
    } else {
      await expect(manager.listQueue({})).resolves.toEqual({ items: [] });
    }
  });
});
