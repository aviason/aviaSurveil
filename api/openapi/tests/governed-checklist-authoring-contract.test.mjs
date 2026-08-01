import assert from "node:assert/strict";
import test from "node:test";

import { assembleOpenApi } from "../../../scripts/bundle-openapi.mjs";

const schema = (schemas, name) => {
  const value = schemas[name];
  assert.ok(value, `missing schema ${name}`);
  return value;
};

test("regulatory traces separate immutable content from decision projections", () => {
  const schemas = assembleOpenApi().components.schemas;
  const trace = schema(schemas, "GovernedRegulatoryTraceView");
  const content = schema(schemas, "GovernedRegulatoryTraceContent");

  assert.equal(content.discriminator.propertyName, "state");
  assert.equal(content.oneOf.length, 2);
  const variants = content.oneOf.map((member) => schema(schemas, member.$ref.split("/").at(-1)));
  assert.deepEqual(variants.map((variant) => variant.properties.state.const).sort(), [
    "RESOLVED",
    "SOURCE_MAPPING_REQUIRED",
  ]);
  for (const variant of variants) assert.equal(variant.additionalProperties, false);

  assert.equal(trace.type, "object");
  assert.equal(trace.additionalProperties, false);
  assert.ok(trace.properties.content);
  assert.ok(trace.properties.mappingReviewState);
  assert.ok(trace.properties.technicalReviewState);
  assert.ok(!trace.properties.decisionId, "decision IDs cannot be immutable trace content");

  const resolved = variants.find((variant) => variant.properties.state.const === "RESOLVED");
  assert.ok(resolved.required.includes("sourceChain"));
  assert.ok(resolved.required.includes("sourceChainDigest"));
  assert.ok(resolved.required.includes("currentnessState"));
  assert.equal(resolved.properties.currentnessState.const, "CURRENT");
  assert.equal(resolved.properties.sourceChain.items.$ref, "#/components/schemas/GovernedSourceChainLink");

  const gap = variants.find((variant) => variant.properties.state.const === "SOURCE_MAPPING_REQUIRED");
  assert.ok(gap.required.includes("gapReason"));
  assert.ok(gap.required.includes("missingFields"));
  assert.ok(gap.required.includes("lastReviewedAt"));
  for (const prohibited of ["sourceChain", "sourceChainDigest", "applicabilityDisposition", "verificationObjective"]) {
    assert.equal(gap.properties[prohibited], undefined, `gap variant must prohibit ${prohibited}`);
  }
});

test("candidate lineage and source-review detail use closed variant unions", () => {
  const schemas = assembleOpenApi().components.schemas;
  const candidate = schema(schemas, "GovernedCandidateView");
  const lineage = schema(schemas, candidate.properties.lineage.$ref.split("/").at(-1));
  assert.equal(lineage.discriminator.propertyName, "lineageType");
  assert.deepEqual(
    lineage.oneOf.map((member) => member.$ref.split("/").at(-1)),
    [
      "GovernedPreV28GenerationRunLineage",
      "GovernedGenerationRunLineage",
      "GovernedExistingCandidateLineage",
      "GovernedDirectOfficialSourceLineage",
    ],
  );
  for (const member of lineage.oneOf) {
    assert.equal(schema(schemas, member.$ref.split("/").at(-1)).additionalProperties, false);
  }

  const detail = schema(schemas, "GovernedSourceReviewDetailView");
  assert.equal(detail.discriminator.propertyName, "reviewItemKind");
  assert.deepEqual(
    detail.oneOf.map((member) => member.$ref.split("/").at(-1)),
    ["GovernedSourceAuthorityReviewDetail", "GovernedCandidateMappingReviewDetail"],
  );
  for (const member of detail.oneOf) {
    const variant = schema(schemas, member.$ref.split("/").at(-1));
    assert.equal(variant.additionalProperties, false);
    assert.ok(variant.properties.currentDecision);
    assert.equal(
      schema(schemas, variant.properties.currentDecision.$ref.split("/").at(-1)).discriminator.propertyName,
      "decisionState",
    );
  }
});

test("capability-scoped authoring routes preserve distinct authority boundaries", () => {
  const document = assembleOpenApi();
  const expected = {
    "/v1/governed-checklist/source-review-queue": "listGovernedChecklistSourceReviewQueue",
    "/v1/governed-checklist/source-review-items/{reviewItemId}": "getGovernedChecklistSourceReviewItem",
    "/v1/governed-checklist/reviewer-queue": "listGovernedChecklistReviewerQueue",
    "/v1/governed-checklist/source-versions/{sourceVersionId}/authority-attestations": "attestRegulatorySourceAuthority",
    "/v1/governed-checklist/existing-candidates/{existingCandidateId}": "getExistingChecklistCandidate",
    "/v1/governed-checklist/existing-candidates/{existingCandidateId}/drafts": "createDraftFromExistingChecklistCandidate",
    "/v1/governed-checklist/official-source-drafts": "createOfficialSourceChecklistDraft",
    "/v1/governed-checklist/candidates/{candidateId}": "getGovernedChecklistDraft",
    "/v1/governed-checklist/candidates/{candidateId}/hybrid-reconciliations": "createHybridReconciledChecklistDraft",
    "/v1/governed-checklist/candidates/{candidateId}/review-comments": "listGovernedChecklistReviewComments",
    "/v1/governed-checklist/candidates/{candidateId}/source-mapping-attestations": "attestGovernedChecklistSourceMapping",
    "/v1/governed-checklist/published-versions/{publishedVersionId}/audit-package-eligibility-evaluations": "evaluateGovernedChecklistAuditPackageEligibility",
  };
  for (const [route, operationId] of Object.entries(expected)) {
    const pathItem = document.paths[route];
    assert.ok(pathItem, `missing ${route}`);
    assert.ok(Object.values(pathItem).some((operation) => operation.operationId === operationId), operationId);
  }

  assert.equal(
    Object.values(document.paths).flatMap((pathItem) => Object.values(pathItem)).some(
      (operation) => operation.operationId === "createChecklistTemplateVersion",
    ),
    false,
    "Admin cannot gain a direct publication alias",
  );
});
