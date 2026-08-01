import { describe, expect, it } from "vitest";

import type {
  CreateAdminGovernedCandidateRevisionInput,
  GovernedCandidateView,
} from "./backend";
import { GovernedValidationError } from "./backend-contracts";
import {
  SYNTHETIC_EDITED_RATIONALE,
  SYNTHETIC_GOVERNED_BUNDLE,
} from "./governed-synthetic-profile";
import {
  governedCandidateContentDigest,
  governedEditSemanticDigest,
  governedImportSemanticDigest,
  governedRequestDigest,
  governedSubmitSemanticDigest,
} from "./governed-canonical";
import { createMockBackendRuntime } from "../mock/create-mock-backend";
import task5ParityFixture from "../../../../docs/regulatory-sources/fixtures/task5-admin-parity.v1.json";

interface InvalidEditCase {
  name: string;
  fieldPath: string;
  code: string;
  message: string;
  mutate(input: CreateAdminGovernedCandidateRevisionInput): void;
}

const invalidEditCases: InvalidEditCase[] = [
  {
    name: "changed relationship",
    fieldPath: "mappings[0].relationship",
    code: "RELATIONSHIP_MISMATCH",
    message: "relationship must preserve the exact supported mapping",
    mutate: (input) => { input.mappings[0]!.relationship = "PARTIALLY_ADDRESSES"; },
  },
  {
    name: "changed applicability",
    fieldPath: "mappings[0].applicability",
    code: "APPLICABILITY_MISMATCH",
    message: "applicability must preserve the exact supported mapping",
    mutate: (input) => { input.mappings[0]!.applicability = "CONDITIONAL"; },
  },
  {
    name: "changed source hash",
    fieldPath: "mappings[0].citations[0].sourceHash",
    code: "SOURCE_HASH_MISMATCH",
    message: "citation source hash is immutable",
    mutate: (input) => { input.mappings[0]!.citations[0]!.sourceHash = "sha256:changed"; },
  },
  {
    name: "changed clause locator",
    fieldPath: "mappings[0].citations[0].locator",
    code: "LOCATOR_MISMATCH",
    message: "citation locator is immutable",
    mutate: (input) => { input.mappings[0]!.citations[0]!.locator = "Unknown clause"; },
  },
  {
    name: "invented source gap",
    fieldPath: "mappings[0].sourceGap",
    code: "SOURCE_GAP_MISMATCH",
    message: "source gaps may not be inferred or fabricated",
    mutate: (input) => {
      input.mappings[0]!.sourceGap = {
        status: "UNRESOLVED",
        reason: "Invented gap",
      };
    },
  },
  {
    name: "duplicate mapping identity",
    fieldPath: "mappings[1].mappingId",
    code: "MAPPING_IDENTITY_MISMATCH",
    message: "the edit must preserve the complete mapping identity set",
    mutate: (input) => { input.mappings.push(structuredClone(input.mappings[0]!)); },
  },
  {
    name: "changed question prompt",
    fieldPath: "questions[0].prompt",
    code: "UNSUPPORTED_CLAIM",
    message: "question text is outside the controlled synthetic registry",
    mutate: (input) => { input.questions[0]!.prompt = "Unsupported question"; },
  },
  {
    name: "blank expected evidence",
    fieldPath: "questions[0].expectedEvidence[0]",
    code: "BLANK_EVIDENCE",
    message: "expected Evidence entries must be nonblank",
    mutate: (input) => { input.questions[0]!.expectedEvidence[0] = ""; },
  },
  {
    name: "invalid allowed answers",
    fieldPath: "questions[0].allowedAnswers",
    code: "INVALID_ALLOWED_ANSWERS",
    message: "allowed answers must preserve the exact governed set",
    mutate: (input) => { input.questions[0]!.allowedAnswers = ["COMPLIANT"]; },
  },
  {
    name: "changed mandatory flag",
    fieldPath: "questions[0].mandatoryCore",
    code: "MANDATORY_FLAG_MISMATCH",
    message: "mandatory-core classification is immutable",
    mutate: (input) => { input.questions[0]!.mandatoryCore = false; },
  },
  {
    name: "changed safety flag",
    fieldPath: "questions[0].safetyCritical",
    code: "SAFETY_FLAG_MISMATCH",
    message: "safety-critical classification is immutable",
    mutate: (input) => { input.questions[0]!.safetyCritical = false; },
  },
  {
    name: "unknown owner",
    fieldPath: "requiredOwners[0].departmentId",
    code: "UNKNOWN_OWNER",
    message: "required owner department is unknown or changed",
    mutate: (input) => { input.requiredOwners[0]!.departmentId = "UNKNOWN_DEPARTMENT"; },
  },
];

function validEdit(candidate: GovernedCandidateView): CreateAdminGovernedCandidateRevisionInput {
  return {
    operationId: task5ParityFixture.edit.operationId,
    idempotencyKey: task5ParityFixture.edit.idempotencyKey,
    candidateId: candidate.candidateId,
    expectedRevision: candidate.revision,
    expectedContentDigest: candidate.contentDigest,
    changeReason: task5ParityFixture.edit.changeReason,
    mappings: candidate.mappings.map((mapping, index) =>
      index === 0 ? { ...mapping, rationale: SYNTHETIC_EDITED_RATIONALE } : mapping),
    questions: structuredClone(candidate.questions),
    requiredOwners: structuredClone(candidate.requiredOwners),
  };
}

describe("Task 5 semantic mock and HTTP parity", () => {
  it("runs the exact invalid edit matrix through mock validation using the real-handler shared contract", async () => {
    for (const invalidCase of invalidEditCases) {
      const runtime = createMockBackendRuntime();
      const mock = runtime.backendForRole("admin").adminWorkspace;
      const candidate = await mock.getGovernedCandidate({
        candidateId: SYNTHETIC_GOVERNED_BUNDLE.candidateBundleId,
      });
      const input = validEdit(candidate);
      invalidCase.mutate(input);

      const mockError = await mock.createGovernedCandidateRevision(input).catch((error: unknown) => error);
      expect(mockError, invalidCase.name).toBeInstanceOf(GovernedValidationError);
      const expectedIssue = {
        fieldPath: invalidCase.fieldPath,
        code: invalidCase.code,
        message: invalidCase.message,
        sourceIdentity: "SYNTHETIC-OPS-AOC",
        sourceHash: SYNTHETIC_GOVERNED_BUNDLE.generationRequest.sourceSnapshots[0]!.sourceHash,
        clauseId: "CLAUSE-SYNTHETIC-OPS-AOC-1",
        locator: "Synthetic OPS/AOC 1",
      };
      expect((mockError as GovernedValidationError).issues, invalidCase.name).toEqual([expectedIssue]);
      if (invalidCase.name === "changed source hash") {
        expect((mockError as GovernedValidationError).issues[0]).toEqual(task5ParityFixture.validation.issue);
      }
    }
  });

  it("pins request, candidate, edited-candidate, and command semantics to shared canonical SHA-256 vectors", async () => {
    const editedMappings = structuredClone(SYNTHETIC_GOVERNED_BUNDLE.complianceMappings);
    editedMappings[0]!.rationale = SYNTHETIC_EDITED_RATIONALE;
    const owners = [{
      departmentId: "FLIGHT_OPERATIONS_INSPECTORATE",
      organizationalUnitId: "FLIGHT_OPERATIONS_INSPECTORATE",
      approvalRequired: true,
    }];
    const editedDigest = await governedCandidateContentDigest({
      complianceMappings: editedMappings,
      inspectionChecklist: {
        checklistId: `TPL-${SYNTHETIC_GOVERNED_BUNDLE.inspectionChecklist.checklistId}`,
        questions: SYNTHETIC_GOVERNED_BUNDLE.inspectionChecklist.questions,
      },
    });
    expect(await governedRequestDigest(SYNTHETIC_GOVERNED_BUNDLE.generationRequest))
      .toBe("sha256:16263c98748063fc7e29dbb7744189cf9a81fd635bed641579772bafa70a6d64");
    expect(await governedCandidateContentDigest({
      complianceMappings: SYNTHETIC_GOVERNED_BUNDLE.complianceMappings,
      inspectionChecklist: SYNTHETIC_GOVERNED_BUNDLE.inspectionChecklist,
    })).toBe("sha256:377598cb1bee5388b19c9d7d4de34f1ff9f6b16b7ac1d2ff6cc5d96af798ad19");
    expect(editedDigest).toBe("sha256:31554d0293d724c6ececb947d3479d306e6281e1a010fd380c5fe2bf626561de");
    expect(await governedImportSemanticDigest("TASK5-GOLDEN-IMPORT", SYNTHETIC_GOVERNED_BUNDLE))
      .toBe("sha256:c956df2ea2b43049e976aab22f79ab1d1526112b55a4b4bcea5e288b279b4ce1");
    expect(await governedEditSemanticDigest({
      candidateId: SYNTHETIC_GOVERNED_BUNDLE.candidateBundleId,
      expectedRevision: 1,
      expectedContentDigest: SYNTHETIC_GOVERNED_BUNDLE.outputDigest,
      changeReason: "Apply the single controlled synthetic alternative.",
      mappings: editedMappings,
      questions: SYNTHETIC_GOVERNED_BUNDLE.inspectionChecklist.questions,
      requiredOwners: owners,
    })).toBe("sha256:e512051bbf68b320c910183b467ddd6ac255419b2c112e95fb21492470ca7515");
    expect(await governedSubmitSemanticDigest({
      operationId: "TASK5-GOLDEN-SUBMIT",
      idempotencyKey: "TASK5-GOLDEN-SUBMIT-KEY",
      candidateId: "CAND-EDIT-GOLDEN",
      expectedContentDigest: editedDigest,
      reason: "Submit exact leaf.",
      expectedRevision: 2,
    })).toBe("sha256:c000a5149e35d1aac3985e779d9b51c231c74d1aa1c9cb1321cacdbfea92f697");

    const runtime = createMockBackendRuntime();
    const admin = runtime.backendForRole("admin").adminWorkspace;
    const candidate = await admin.getGovernedCandidate({
      candidateId: SYNTHETIC_GOVERNED_BUNDLE.candidateBundleId,
    });
    const successor = await admin.createGovernedCandidateRevision(validEdit(candidate));
    expect(successor.contentDigest).toBe(editedDigest);
  });

  it("keeps exact import identity, failed-run projection, immutable successor, and current-leaf submission", async () => {
    const runtime = createMockBackendRuntime();
    const admin = runtime.backendForRole("admin").adminWorkspace;
    const imported = await admin.importGovernedGenerationRun({
      operationId: task5ParityFixture.import.operationId,
      idempotencyKey: task5ParityFixture.import.idempotencyKey,
      candidateBundle: SYNTHETIC_GOVERNED_BUNDLE,
    });
    await expect(admin.importGovernedGenerationRun({
      operationId: "TASK5-PARITY-IMPORT",
      idempotencyKey: "TASK5-PARITY-IMPORT-DIFFERENT-KEY",
      candidateBundle: SYNTHETIC_GOVERNED_BUNDLE,
    })).rejects.toThrow(/identity|semantics/i);
    expect((await admin.getGovernedGenerationRun({
      generationRunId: "GENRUN-FAILED-SYNTHETIC-INSPECTION",
    }))).toEqual(expect.objectContaining({
      status: "FAILED",
      candidate: null,
      failure: expect.objectContaining({
        code: "VALIDATION_FAILED",
        requestId: "GENREQ-FAILED-SYNTHETIC-INSPECTION",
      }),
    }));
    const successor = await admin.createGovernedCandidateRevision(validEdit(imported.candidate!));
    await expect(admin.submitGovernedCandidateReview({
      operationId: "TASK5-PARITY-SUBMIT-ANCESTOR",
      idempotencyKey: "TASK5-PARITY-SUBMIT-ANCESTOR",
      candidateId: imported.candidate!.candidateId,
      expectedRevision: imported.candidate!.revision,
      expectedContentDigest: imported.candidate!.contentDigest,
      reason: "Ancestor must fail.",
    })).rejects.toThrow(/stale/i);
    const submitted = await admin.submitGovernedCandidateReview({
      operationId: task5ParityFixture.submit.operationId,
      idempotencyKey: task5ParityFixture.submit.idempotencyKey,
      candidateId: successor.candidateId,
      expectedRevision: successor.revision,
      expectedContentDigest: successor.contentDigest,
      reason: task5ParityFixture.submit.reason,
    });
    expect(submitted).toEqual(expect.objectContaining(task5ParityFixture.submitted));
    expect(successor).toEqual(expect.objectContaining(task5ParityFixture.successor));

    const rereadRun = await admin.getGovernedGenerationRun({
      generationRunId: task5ParityFixture.run.generationRunId,
    });
    const rereadCandidate = await admin.getGovernedCandidate({
      candidateId: successor.candidateId,
    });
    expect(rereadRun).toEqual(expect.objectContaining({
      generationRunId: task5ParityFixture.run.generationRunId,
      status: task5ParityFixture.run.status,
      inputDigest: task5ParityFixture.run.inputDigest,
      outputDigest: task5ParityFixture.run.outputDigest,
      requestId: task5ParityFixture.run.requestId,
      providerId: task5ParityFixture.run.providerId,
    }));
    expect(rereadRun.candidate).toEqual(expect.objectContaining({
      candidateId: rereadCandidate.candidateId,
      revision: rereadCandidate.revision,
      contentDigest: rereadCandidate.contentDigest,
      status: rereadCandidate.status,
    }));
    expect(rereadCandidate).toEqual(expect.objectContaining({
      candidateId: task5ParityFixture.successor.candidateId,
      revision: task5ParityFixture.successor.revision,
      contentDigest: task5ParityFixture.successor.contentDigest,
      status: task5ParityFixture.submitted.status,
    }));
  });
});
