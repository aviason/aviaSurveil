import assert from "node:assert/strict";
import test from "node:test";

import { assembleOpenApi } from "../../../scripts/bundle-openapi.mjs";

const document = assembleOpenApi();

const required = new Map([
  ["/v1/admin/governed-checklist/sources", "listAdminGovernedSources"],
  ["/v1/admin/governed-checklist/generation-runs", "importAdminGovernedGenerationRun"],
  ["/v1/admin/governed-checklist/generation-runs/{generationRunId}", "getAdminGovernedGenerationRun"],
  ["/v1/admin/governed-checklist/candidates/{candidateId}", "getAdminGovernedCandidate"],
  ["/v1/admin/governed-checklist/candidates/{candidateId}/revisions", "createAdminGovernedCandidateRevision"],
  ["/v1/admin/governed-checklist/candidates/{candidateId}/submissions", "submitAdminGovernedCandidateReview"],
]);

test("Task 5 exposes only Admin candidate inspection, immutable editing, and review submission", () => {
  for (const [path, operationId] of required) {
    const method = path.endsWith("sources") || path.includes("/{generationRunId}") || path.endsWith("{candidateId}") ? "get" : "post";
    const operation = document.paths[path]?.[method];
    assert.ok(operation, `${method.toUpperCase()} ${path} is required`);
    assert.equal(operation.operationId, operationId);
  }
  const serialized = JSON.stringify(Object.fromEntries(
    Object.entries(document.paths).filter(([path]) => path.startsWith("/v1/admin/")),
  )).toLowerCase();
  for (const forbidden of ["approvegoverned", "publishgoverned", "technical-approval", "governed-checklist/publication"]) {
    assert.equal(serialized.includes(forbidden), false, `${forbidden} must remain absent`);
  }
});

test("Task 5 command schemas carry exact concurrency and idempotency fields", () => {
  for (const schemaName of [
    "ImportAdminGovernedGenerationRunInput",
    "CreateAdminGovernedCandidateRevisionInput",
    "SubmitAdminGovernedCandidateReviewInput",
  ]) {
    const schema = document.components.schemas[schemaName];
    assert.ok(schema, `${schemaName} is required`);
    assert.equal(schema.additionalProperties, false);
    for (const field of ["operationId", "idempotencyKey"]) {
      assert.ok(schema.required.includes(field), `${schemaName}.${field} is required`);
    }
  }
  assert.ok(document.components.schemas.CreateAdminGovernedCandidateRevisionInput.required.includes("expectedRevision"));
  assert.ok(document.components.schemas.CreateAdminGovernedCandidateRevisionInput.required.includes("expectedContentDigest"));
  assert.ok(document.components.schemas.SubmitAdminGovernedCandidateReviewInput.required.includes("expectedRevision"));
  assert.ok(document.components.schemas.SubmitAdminGovernedCandidateReviewInput.required.includes("expectedContentDigest"));
});

test("Task 5 schemas preserve typed source facts and structured validation issues", () => {
  const source = document.components.schemas.GovernedSourceSnapshotView;
  assert.deepEqual(source.required, [
    "sourceId", "sourceIdentity", "versionIdentity", "title", "sourceHash",
    "locator", "clauseId", "clauseLocator", "partitions",
    "applicabilityFacts", "unresolvedGaps", "generationRunIds", "candidateIds",
  ]);
  assert.equal(source.properties.partitions.items.$ref, "#/components/schemas/GovernedSourcePartitionFactView");
  assert.equal(source.properties.applicabilityFacts.items.$ref, "#/components/schemas/GovernedSourceApplicabilityFactView");
  assert.equal(source.properties.unresolvedGaps.items.$ref, "#/components/schemas/GovernedUnresolvedSourceGapView");
  for (const removedProjection of ["crosswalkRole", "applicability", "unresolvedGapIds", "partitionIds"]) {
    assert.equal(source.properties[removedProjection], undefined, `${removedProjection} must not be inferred or conflated`);
  }

  const issue = document.components.schemas.GovernedValidationIssue;
  assert.equal(issue.additionalProperties, false);
  assert.deepEqual(issue.required, [
    "fieldPath", "code", "message", "sourceIdentity", "sourceHash",
    "clauseId", "locator",
  ]);
  assert.equal(
    document.components.schemas.GovernedValidationProblem.properties.issues.items.$ref,
    "#/components/schemas/GovernedValidationIssue",
  );

  const revision = document.components.schemas.CreateAdminGovernedCandidateRevisionInput;
  assert.equal(revision.properties.mappings.items.$ref, "#/components/schemas/GovernedMappingView");
  assert.equal(revision.properties.questions.items.$ref, "#/components/schemas/GovernedQuestionView");
  assert.equal(revision.properties.requiredOwners.items.$ref, "#/components/schemas/GovernedRequiredOwnerView");
  assert.equal(
    document.components.schemas.ImportAdminGovernedGenerationRunInput.properties.candidateBundle.$ref,
    "#/components/schemas/GovernedCandidateBundleInput",
  );
  const run = document.components.schemas.GovernedGenerationRunView;
  for (const field of [
    "inputSchemaVersion", "generationPolicyVersion", "providerCatalogVersion",
    "providerId", "providerAdapterVersion", "inspectionType", "targetId", "requestId",
  ]) {
    assert.ok(run.required.includes(field), `GovernedGenerationRunView.${field} is required`);
  }
  assert.equal(run.properties.failure.anyOf[0].$ref, "#/components/schemas/GovernedGenerationFailureView");
});
