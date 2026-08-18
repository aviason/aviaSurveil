import type { CanonicalQuestionCatalogEntry, CanonicalQuestionUsageClass } from "../backend/backend";

export type PriorAuditRecommendationProfile = "prior-audit-multi-history" | "prior-audit-single-history";
export type MockPriorAuditRecommendation = CanonicalQuestionCatalogEntry["recommendation"];

const catalogVersion = "aga-approved-source@2.0.0";
const questionId = (ordinal: number) => `qv:aga-approved-source-v2:FSS-AGA-FORM-002:all-forms-preview-002-${String(ordinal).padStart(4, "0")}`;

function recommendation(
  recommendationState: MockPriorAuditRecommendation["recommendationState"],
  classification: MockPriorAuditRecommendation["classification"],
  includedByDefault: boolean,
  canDefer: boolean,
  historyCount: number,
  signalCodes: string[],
  rationale: string,
): MockPriorAuditRecommendation {
  return {
    recommendationState,
    classification,
    includedByDefault,
    canDefer,
    historyCount,
    comparableAuditCount: historyCount,
    validatedCleanAuditCount: recommendationState === "RECENTLY_VERIFIED" ? historyCount : 0,
    lastComparableResult: historyCount ? "COMPLIANT" : null,
    lastComparableAuditId: historyCount ? `AUD-PRIOR-${historyCount === 1 ? "SINGLE-001" : "MULTI-003"}` : null,
    lastValidatedCleanAt: recommendationState === "RECENTLY_VERIFIED" ? "2026-08-15T00:00:00.000Z" : null,
    lastVerifiedAt: null,
    recurrenceDueAt: null,
    signalCodes,
    rationale,
    guardrails: ["MANDATORY_FLOOR_ENFORCED", "FULL_CATALOG_OVERRIDE_ALLOWED", ...(canDefer ? ["EXPLICIT_MANAGER_DEVIATION_REQUIRED"] : [])],
  };
}

const multiRecommendations: Record<string, MockPriorAuditRecommendation> = {
  [questionId(1)]: recommendation("SUGGESTED_NOW", "MANDATORY_CORE", true, false, 3, ["MANDATORY_CONTROL"], "Mandatory controls remain in every comparable Audit scope."),
  [questionId(2)]: recommendation("SUGGESTED_NOW", "MANDATORY_CORE", true, false, 3, ["SAFETY_CRITICAL_CONTROL"], "Safety-critical controls remain in every comparable Audit scope."),
  [questionId(3)]: recommendation("SUGGESTED_NOW", "FOCUSED_FULL", true, false, 3, ["OPEN_FINDING"], "Open Finding work keeps this question in the suggested scope."),
  [questionId(4)]: recommendation("SUGGESTED_NOW", "FOCUSED_FULL", true, false, 3, ["REPEAT_FINDING"], "Repeat Finding history keeps this question in the suggested scope."),
  [questionId(5)]: recommendation("RECENTLY_VERIFIED", "DEFER_ELIGIBLE", false, true, 3, ["RECENTLY_VERIFIED", "DEFER_ELIGIBLE"], "Repeated validated-clean history is within its recurrence interval; the question is safe to defer by default."),
  [questionId(6)]: recommendation("SUGGESTED_NOW", "ROTATIONAL_SAMPLE", true, false, 3, ["RECURRENCE_DUE"], "The configured recurrence interval is due; keep this optional control in the suggested scope."),
  [questionId(7)]: recommendation("SUGGESTED_NOW", "FOCUSED_FULL", true, false, 3, ["SOURCE_OR_MAPPING_CHANGED"], "A source, mapping, successor, or accepted-remediation change requires full review."),
  [questionId(8)]: recommendation("UNCERTAIN_SIGNAL", "ROTATIONAL_SAMPLE", true, false, 3, ["UNKNOWN_HISTORY"], "History is incomplete or non-validating; the question remains suggested."),
};

const singleRecommendations: Record<string, MockPriorAuditRecommendation> = {
  [questionId(11)]: recommendation("SUGGESTED_NOW", "MANDATORY_CORE", true, false, 1, ["MANDATORY_CONTROL"], "Mandatory controls remain in every comparable Audit scope."),
  [questionId(12)]: recommendation("SUGGESTED_NOW", "FOCUSED_FULL", true, false, 1, ["OPEN_FINDING"], "Open Finding work keeps this question in the suggested scope."),
  [questionId(13)]: recommendation("UNCERTAIN_SIGNAL", "ROTATIONAL_SAMPLE", true, false, 1, ["UNKNOWN_HISTORY"], "History is incomplete or non-validating; the question remains suggested."),
  [questionId(14)]: recommendation("SUGGESTED_NOW", "FOCUSED_FULL", true, false, 1, ["SOURCE_OR_MAPPING_CHANGED"], "A source, mapping, successor, or accepted-remediation change requires full review."),
  [questionId(15)]: recommendation("UNCERTAIN_SIGNAL", "ROTATIONAL_SAMPLE", true, false, 1, ["INSUFFICIENT_LONGITUDINAL_HISTORY"], "One clean Audit is not sufficient longitudinal evidence for omission."),
};

export const priorAuditRecommendationFixtures: Record<PriorAuditRecommendationProfile, {
  profile: PriorAuditRecommendationProfile;
  organizationId: string;
  providerScopeId: string;
  regulatedTargetId: string;
  location: string;
  auditType: string;
  catalogVersion: string;
  priorAuditIds: string[];
  recommendations: Record<string, MockPriorAuditRecommendation>;
}> = {
  "prior-audit-multi-history": {
    profile: "prior-audit-multi-history",
    organizationId: "ORG-PRIOR-AUDIT-QUALIFICATION",
    providerScopeId: "SCOPE-PRIOR-AUDIT-001",
    regulatedTargetId: "TARGET-PRIOR-AUDIT-001",
    location: "Windhoek International Airport",
    auditType: "RAMP_INSPECTION",
    catalogVersion,
    priorAuditIds: ["AUD-PRIOR-MULTI-001", "AUD-PRIOR-MULTI-002", "AUD-PRIOR-MULTI-003"],
    recommendations: multiRecommendations,
  },
  "prior-audit-single-history": {
    profile: "prior-audit-single-history",
    organizationId: "ORG-PRIOR-AUDIT-QUALIFICATION",
    providerScopeId: "SCOPE-PRIOR-AUDIT-001",
    regulatedTargetId: "TARGET-PRIOR-AUDIT-001",
    location: "Windhoek International Airport",
    auditType: "RAMP_INSPECTION",
    catalogVersion,
    priorAuditIds: ["AUD-PRIOR-SINGLE-001"],
    recommendations: singleRecommendations,
  },
};

export function evaluatePriorAuditRecommendations(profile: PriorAuditRecommendationProfile): MockPriorAuditRecommendation[] {
  return Object.values(priorAuditRecommendationFixtures[profile].recommendations);
}

export function suggestedPriorAuditQuestionIds(profile: PriorAuditRecommendationProfile): string[] {
  return evaluatePriorAuditRecommendations(profile).filter((item) => item.includedByDefault).map((item) => Object.entries(priorAuditRecommendationFixtures[profile].recommendations).find(([, value]) => value === item)?.[0] ?? "");
}

export function fullCatalogPriorAuditQuestionIds(profile: PriorAuditRecommendationProfile): string[] {
  return Object.keys(priorAuditRecommendationFixtures[profile].recommendations);
}

export function priorAuditCatalogEntries(profile: PriorAuditRecommendationProfile, usageClass: CanonicalQuestionUsageClass, requestedCatalogVersion: string): CanonicalQuestionCatalogEntry[] {
  return Object.entries(priorAuditRecommendationFixtures[profile].recommendations).map(([id, item], index) => ({
    catalogVersion: requestedCatalogVersion,
    usageClass,
    questionVersionId: id,
    formCode: "FSS-AGA-FORM-002",
    proposalId: `prior-audit-${profile}-${String(index + 1).padStart(4, "0")}`,
    ordinal: index + 1,
    questionDigest: `sha256:${String(index + 1).padStart(64, "0")}`,
    prompt: `Prior-audit recommendation question ${index + 1}.`,
    configuredReference: "Approved source fixture reference",
    expectedEvidence: "Exact inspection evidence for the selected question.",
    sourceLocator: `approved://FSS-AGA-FORM-002/${index + 1}`,
    sourceGapState: "APPROVED",
    proposedDomain: "AERODROME_OPERATIONAL_SAFETY",
    proposedTopic: "SAFETY_ASSURANCE",
    proposedRiskBand: item.classification === "MANDATORY_CORE" ? "PROPOSED_SAFETY_CRITICAL" : "PROPOSED_HIGH_OPERATIONAL",
    aiAdvisory: {
      domainCode: "AERODROME_OPERATIONAL_SAFETY",
      topicCodes: ["SAFETY_ASSURANCE"],
      inspectionTypeCodes: ["RAMP_INSPECTION"],
      inspectionProfileCodes: ["AERODROME_OPERATIONAL_SAFETY"],
      applicabilityDisposition: "APPLICABLE",
      riskTier: item.classification === "MANDATORY_CORE" ? "HIGH" : item.recommendationState === "UNCERTAIN_SIGNAL" ? "UNKNOWN" : "MEDIUM",
      safetyCritical: item.signalCodes.includes("SAFETY_CRITICAL_CONTROL"),
      agreementConfidence: "HIGH",
      advisoryState: item.recommendationState,
      recommendationReasonCodes: [...item.signalCodes],
      recurrenceMonths: 12,
      previouslyVerifiedAt: item.lastVerifiedAt,
      recurrenceDueAt: item.recurrenceDueAt,
      externalApplicabilityUnresolved: false,
    },
    recommendation: structuredClone(item),
    canSelect: true,
    canPublish: false,
    governedCandidateId: null,
    governedCandidateRevision: null,
    governedCandidateContentDigest: null,
    governedCandidateStatus: null,
    reviewRevision: 0,
    reviewDisposition: null,
    reviewDigest: null,
  }));
}

export function recommendationProfileForScope(scope: { organizationId?: string; providerScopeId?: string; regulatedTargetId?: string; location?: string; auditType?: string }): PriorAuditRecommendationProfile | null {
  return Object.values(priorAuditRecommendationFixtures).find((fixture) =>
    fixture.organizationId === scope.organizationId && fixture.providerScopeId === scope.providerScopeId &&
    fixture.regulatedTargetId === scope.regulatedTargetId && fixture.location === scope.location && fixture.auditType === scope.auditType,
  )?.profile ?? null;
}
