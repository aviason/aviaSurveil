import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const paths = JSON.parse(await readFile(new URL("../source/paths/platform.json", import.meta.url), "utf8"));
const schemas = JSON.parse(await readFile(new URL("../source/schemas/platform.json", import.meta.url), "utf8"));

const managerOperations = [
  ["/v1/department-manager/governed-checklist/blocked-generation-validations", "post", "validateDepartmentManagerBlockedGeneration"],
  ["/v1/department-manager/governed-checklist/review-queue", "get", "listDepartmentManagerGovernedReviewQueue"],
  ["/v1/department-manager/governed-checklist/candidates/{candidateId}", "get", "getDepartmentManagerGovernedCandidate"],
  ["/v1/department-manager/governed-checklist/candidates/{candidateId}/returns", "post", "returnDepartmentManagerGovernedCandidate"],
  ["/v1/department-manager/governed-checklist/candidates/{candidateId}/rejections", "post", "rejectDepartmentManagerGovernedCandidate"],
  ["/v1/department-manager/governed-checklist/candidates/{candidateId}/technical-approvals", "post", "approveDepartmentManagerGovernedCandidate"],
  ["/v1/department-manager/governed-checklist/candidates/{candidateId}/publications", "post", "publishDepartmentManagerGovernedCandidate"],
  ["/v1/department-manager/governed-checklist/published-versions/{templateVersionId}", "get", "getDepartmentManagerGovernedPublishedVersion"],
];

test("Task 6 exposes queue/read and separate manager lifecycle commands without an Admin publication route", () => {
  for (const [path, method, operationId] of managerOperations) {
    assert.equal(paths[path]?.[method]?.operationId, operationId, `${method.toUpperCase()} ${path}`);
  }
  assert.equal(paths["/v1/admin/governed-checklist/candidates/{candidateId}/technical-approvals"], undefined);
  assert.equal(paths["/v1/admin/governed-checklist/candidates/{candidateId}/publications"], undefined);
});

test("Task 6 schemas close exact command, queue, decision, blocker, and publication shapes", () => {
  assert.deepEqual(schemas.GovernedCandidateView.properties.status.enum, [
    "GENERATED_DRAFT", "DEPARTMENT_REVIEW", "RETURNED", "REJECTED", "TECHNICALLY_APPROVED", "PUBLISHED",
  ]);
  for (const schemaName of [
    "DepartmentManagerGovernedReviewCommandInput",
    "DepartmentManagerGovernedReviewItem",
    "DepartmentManagerGovernedReviewQueue",
    "GovernedReviewDecisionView",
    "GovernedPublicationView",
    "GovernedPublishedVersionView",
    "GovernedBlockedGenerationEffectCounts",
    "ValidateDepartmentManagerBlockedGenerationInput",
    "GovernedBlockedGenerationResult",
  ]) {
    assert.equal(schemas[schemaName]?.additionalProperties, false, schemaName);
    assert.ok(schemas[schemaName]?.required?.length > 0, schemaName);
  }
  assert.deepEqual(schemas.DepartmentManagerGovernedReviewCommandInput.required, [
    "operationId", "idempotencyKey", "candidateId", "expectedRevision", "expectedContentDigest", "reason",
  ]);
  assert.ok(schemas.DepartmentManagerGovernedReviewItem.required.includes("blockingIssues"));
  assert.ok(schemas.DepartmentManagerGovernedReviewItem.required.includes("decisions"));
  assert.deepEqual(Object.keys(paths[
    "/v1/department-manager/governed-checklist/published-versions/{templateVersionId}"
  ]), ["get"]);
  assert.ok(schemas.GovernedReviewDecisionView.required.includes("semanticPayloadDigest"));
  assert.ok(schemas.GovernedReviewDecisionView.required.includes("auditEventId"));
  assert.ok(schemas.GovernedPublicationView.required.includes("semanticPayloadDigest"));
  assert.ok(schemas.GovernedPublicationView.required.includes("auditEventId"));
});
