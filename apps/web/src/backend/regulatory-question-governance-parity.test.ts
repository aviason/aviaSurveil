import { describe, expect, it } from "vitest";

import type { GovernedQuestionView } from "./backend";
import {
  SYNTHETIC_GOVERNED_BUNDLE,
  SYNTHETIC_HYBRID_RECONCILED_BUNDLE,
  SYNTHETIC_LEGACY_CHECKLIST_CANDIDATE_BUNDLE,
} from "./governed-synthetic-profile";
import {
  governedCandidateContentDigest,
  governedEditSemanticDigest,
  governedImportSemanticDigest,
  governedRequestDigest,
} from "./governed-canonical";
import { governedQuestionGovernanceIssuesForTest } from "../mock/mock-engine";

function completeQuestion(): GovernedQuestionView {
  return structuredClone(SYNTHETIC_GOVERNED_BUNDLE.inspectionChecklist.questions[0]!);
}

function scopedQuestion(classification: "FOCUSED_FULL" | "ROTATIONAL_SAMPLE"): GovernedQuestionView {
  const question = completeQuestion();
  question.mandatoryCore = false;
  question.safetyCritical = false;
  question.scopeRecommendation = {
    ...question.scopeRecommendation,
    classification,
    inputSignals: ["Current source applicability", "Reviewed operational history"],
    operationalHistoryBasis: "SYNTHETIC_REVIEWED_HISTORY",
    rationale: "A reviewed current source trace supports this scope recommendation.",
    guardrails: {
      mandatoryControl: false,
      safetyCritical: false,
      unknownHistory: false,
      sourceChanged: false,
      overdueControl: false,
      automaticDeferralPermitted: true,
    },
    automaticDeferral: false,
  };
  return question;
}

describe("Task 6 question-governance mock parity", () => {
  it("excludes server-derived mapping review projections from immutable content digests", async () => {
    const baseline = structuredClone(SYNTHETIC_GOVERNED_BUNDLE);
    const accepted = structuredClone(baseline);
    accepted.inspectionChecklist.questions[0]!.regulatoryTrace.mappingReviewState = "ACCEPTED";
    expect(await governedCandidateContentDigest({
      complianceMappings: baseline.complianceMappings,
      inspectionChecklist: baseline.inspectionChecklist,
    })).toBe(await governedCandidateContentDigest({
      complianceMappings: accepted.complianceMappings,
      inspectionChecklist: accepted.inspectionChecklist,
    }));
  });

  it("excludes server-derived mapping review projections from import and edit digests", async () => {
    const baseline = structuredClone(SYNTHETIC_GOVERNED_BUNDLE);
    const projected = structuredClone(baseline);
    projected.inspectionChecklist.questions[0]!.regulatoryTrace.mappingReviewState = "ACCEPTED";

    expect(await governedImportSemanticDigest("TASK6-MAPPING-PROJECTION", baseline))
      .toBe(await governedImportSemanticDigest("TASK6-MAPPING-PROJECTION", projected));

    const baseCommand = {
      candidateId: baseline.candidateBundleId,
      expectedRevision: 1,
      expectedContentDigest: baseline.outputDigest,
      changeReason: "Apply the controlled synthetic mapping review projection.",
      mappings: structuredClone(baseline.complianceMappings),
      questions: structuredClone(baseline.inspectionChecklist.questions),
      requiredOwners: [{
        departmentId: "FLIGHT_OPERATIONS_INSPECTORATE",
        organizationalUnitId: "FLIGHT_OPERATIONS_INSPECTORATE",
        approvalRequired: true,
      }],
    };
    const projectedCommand = {
      ...baseCommand,
      questions: structuredClone(projected.inspectionChecklist.questions),
    };
    expect(await governedEditSemanticDigest(baseCommand))
      .toBe(await governedEditSemanticDigest(projectedCommand));
  });

  it("excludes the literal source-gap technical projection from candidate, import, and edit digests", async () => {
    const baseline = structuredClone(SYNTHETIC_LEGACY_CHECKLIST_CANDIDATE_BUNDLE);
    const projected = structuredClone(baseline);
    projected.inspectionChecklist.questions[0]!.regulatoryTrace.technicalReviewState = "TECHNICAL_REVIEW_REQUIRED";

    expect(await governedCandidateContentDigest({
      complianceMappings: baseline.complianceMappings,
      inspectionChecklist: baseline.inspectionChecklist,
    })).toBe(await governedCandidateContentDigest({
      complianceMappings: projected.complianceMappings,
      inspectionChecklist: projected.inspectionChecklist,
    }));
    expect(await governedImportSemanticDigest("TASK6-SOURCE-GAP-PROJECTION", baseline))
      .toBe(await governedImportSemanticDigest("TASK6-SOURCE-GAP-PROJECTION", projected));

    const baseCommand = {
      candidateId: baseline.candidateBundleId,
      expectedRevision: 1,
      expectedContentDigest: baseline.outputDigest,
      changeReason: "Keep the source gap visible while awaiting mapping.",
      mappings: structuredClone(baseline.complianceMappings),
      questions: structuredClone(baseline.inspectionChecklist.questions),
      requiredOwners: [{
        departmentId: "FLIGHT_OPERATIONS_INSPECTORATE",
        organizationalUnitId: "FLIGHT_OPERATIONS_INSPECTORATE",
        approvalRequired: true,
      }],
    };
    const projectedCommand = {
      ...baseCommand,
      questions: structuredClone(projected.inspectionChecklist.questions),
    };
    expect(await governedEditSemanticDigest(baseCommand))
      .toBe(await governedEditSemanticDigest(projectedCommand));
  });

  it("pins candidate-only legacy and hybrid fixtures to the Go digest vectors", async () => {
    for (const bundle of [
      SYNTHETIC_LEGACY_CHECKLIST_CANDIDATE_BUNDLE,
      SYNTHETIC_HYBRID_RECONCILED_BUNDLE,
    ]) {
      expect(await governedRequestDigest(bundle.generationRequest)).toBe(bundle.inputDigest);
      expect(await governedCandidateContentDigest({
        complianceMappings: bundle.complianceMappings,
        inspectionChecklist: bundle.inspectionChecklist,
      })).toBe(bundle.outputDigest);
    }
  });

  it("accepts complete traced mandatory, focused, and rotational-sample questions", () => {
    expect(governedQuestionGovernanceIssuesForTest(completeQuestion())).toEqual([]);
    expect(governedQuestionGovernanceIssuesForTest(scopedQuestion("FOCUSED_FULL"))).toEqual([]);
    expect(governedQuestionGovernanceIssuesForTest(scopedQuestion("ROTATIONAL_SAMPLE"))).toEqual([]);
  });

  it("keeps an explicit legacy source gap visible but fail-closed", () => {
    const question = scopedQuestion("FOCUSED_FULL");
    question.origin = "EXISTING_CHECKLIST_CANDIDATE";
    question.citations = [];
    question.scopeRecommendation = {
      ...question.scopeRecommendation,
      guardrails: {
        ...question.scopeRecommendation.guardrails,
        unknownHistory: true,
        automaticDeferralPermitted: false,
      },
    };
    question.regulatoryTrace = {
      state: "SOURCE_MAPPING_REQUIRED",
      mappingReviewState: "SOURCE_MAPPING_REQUIRED",
      technicalReviewState: "NOT_AVAILABLE",
    };

    expect(governedQuestionGovernanceIssuesForTest(question).map((issue) => issue.code))
      .toEqual(["SOURCE_MAPPING_REQUIRED"]);

    question.regulatoryTrace = {
      state: "SOURCE_MAPPING_REQUIRED",
      mappingReviewState: "SOURCE_MAPPING_REQUIRED",
      technicalReviewState: "NOT_AVAILABLE",
      sourceTitle: "Unverified partial trace",
    };
    expect(governedQuestionGovernanceIssuesForTest(question).map((issue) => issue.code))
      .toContain("REGULATORY_TRACE_REQUIRED");
  });

  it("does not elevate an existing checklist candidate to a resolved authority trace without hybrid reconciliation", () => {
    const question = completeQuestion();
    question.origin = "EXISTING_CHECKLIST_CANDIDATE";

    expect(governedQuestionGovernanceIssuesForTest(question).map((issue) => issue.code))
      .toContain("QUESTION_ORIGIN_TRACE_MISMATCH");
  });

  it("fails closed for missing trace/scope, stale trace, and guarded automatic deferral", () => {
    const cases: Array<{ name: string; question: GovernedQuestionView; code: string }> = [];

    const missingTrace = completeQuestion();
    missingTrace.regulatoryTrace = {
      state: "RESOLVED",
      mappingReviewState: "SOURCE_OWNER_REVIEW_REQUIRED",
      technicalReviewState: "TECHNICAL_REVIEW_REQUIRED",
    };
    cases.push({ name: "missing trace", question: missingTrace, code: "REGULATORY_TRACE_REQUIRED" });

    const missingClassification = completeQuestion();
    missingClassification.scopeRecommendation = {
      ...missingClassification.scopeRecommendation,
      classification: "" as never,
    };
    cases.push({ name: "missing classification", question: missingClassification, code: "SCOPE_CLASSIFICATION_REQUIRED" });

    const missingRationale = completeQuestion();
    missingRationale.scopeRecommendation = {
      ...missingRationale.scopeRecommendation,
      rationale: "",
    };
    cases.push({ name: "missing rationale", question: missingRationale, code: "SCOPE_RATIONALE_REQUIRED" });

    const stale = completeQuestion();
    stale.regulatoryTrace = { ...stale.regulatoryTrace, currentnessState: "STALE" };
    cases.push({ name: "stale trace", question: stale, code: "STALE_SOURCE_TRACE" });

    const missingTechnicalReview = completeQuestion();
    missingTechnicalReview.regulatoryTrace = {
      ...missingTechnicalReview.regulatoryTrace,
      technicalReviewState: "" as never,
    };
    cases.push({ name: "missing technical review", question: missingTechnicalReview, code: "TRACE_TECHNICAL_REVIEW_REQUIRED" });

    const preclaimedApproval = completeQuestion();
    preclaimedApproval.scopeRecommendation = {
      ...preclaimedApproval.scopeRecommendation,
      approvalReviewState: "TECHNICALLY_APPROVED",
    };
    preclaimedApproval.regulatoryTrace = {
      ...preclaimedApproval.regulatoryTrace,
      technicalReviewState: "TECHNICALLY_APPROVED",
    };
    cases.push({ name: "preclaimed technical approval", question: preclaimedApproval, code: "SCOPE_REVIEW_STATE_REQUIRED" });
    cases.push({ name: "preclaimed trace approval", question: preclaimedApproval, code: "TRACE_TECHNICAL_REVIEW_REQUIRED" });

    const automaticDeferral = completeQuestion();
    automaticDeferral.scopeRecommendation = {
      ...automaticDeferral.scopeRecommendation,
      automaticDeferral: true,
    };
    cases.push({ name: "mandatory automatic deferral", question: automaticDeferral, code: "AUTOMATIC_DEFERRAL_DENIED" });

    for (const testCase of cases) {
      expect(
        governedQuestionGovernanceIssuesForTest(testCase.question).map((issue) => issue.code),
        testCase.name,
      ).toContain(testCase.code);
    }
  });

  it("requires the hybrid comparison to bind candidate-only legacy values to the current trace", () => {
    const question = scopedQuestion("ROTATIONAL_SAMPLE");
    question.origin = "HYBRID_RECONCILED";
    question.scopeRecommendation = {
      ...question.scopeRecommendation,
      guardrails: {
        ...question.scopeRecommendation.guardrails,
        sourceChanged: true,
        automaticDeferralPermitted: false,
      },
    };
    question.reconciliation = {
      legacyQuestionId: "Q-SYNTHETIC-LEGACY-CANDIDATE-003",
      legacyWording: "Historical checklist wording retained only as candidate input.",
      legacyOperationalIntent: "Candidate-only operational observation.",
      legacyResultHistory: "Unverified historical result input.",
      legacyExpectedEvidence: ["Historical checklist note"],
      legacyApplicability: "UNKNOWN_CANDIDATE_INPUT",
      legacyScopeClassification: "UNKNOWN_CANDIDATE_INPUT",
      currentWording: question.prompt,
      currentExpectedEvidence: [...question.expectedEvidence],
      currentApplicability: question.regulatoryTrace.applicability!,
      currentScopeClassification: "ROTATIONAL_SAMPLE",
      wordingChanged: true,
      evidenceChanged: true,
      applicabilityChanged: true,
      scopeChanged: true,
    };
    expect(governedQuestionGovernanceIssuesForTest(question)).toEqual([]);

    question.reconciliation.currentApplicability = "";
    expect(governedQuestionGovernanceIssuesForTest(question).map((issue) => issue.code))
      .toContain("HYBRID_RECONCILIATION_REQUIRED");
  });
});
