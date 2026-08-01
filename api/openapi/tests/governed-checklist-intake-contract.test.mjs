import assert from "node:assert/strict";
import test from "node:test";

import { assembleOpenApi } from "../../../scripts/bundle-openapi.mjs";

const dereference = (schemas, schema) => {
  if (!schema?.$ref) return schema;
  return schemas[schema.$ref.split("/").at(-1)];
};

const operation = (document, method, route) => document.paths[route]?.[method];

test("intake transport defines bounded multipart receipt and admin inventory routes", () => {
  const document = assembleOpenApi();
  const schemas = document.components.schemas;
  const multipart = operation(
    document,
    "post",
    "/v1/admin/governed-checklist/import-batches",
  );

  assert.ok(multipart, "the archive receipt route is required");
  assert.equal(multipart.operationId, "createAdminChecklistImportBatch");
  assert.ok(multipart.requestBody?.content?.["multipart/form-data"]);
  assert.equal(multipart.requestBody.content["multipart/form-data"].schema.type, "object");
  const multipartSchema = multipart.requestBody.content["multipart/form-data"].schema;
  assert.deepEqual(multipartSchema.required, ["archive", "receipt"]);
  assert.equal(multipartSchema.additionalProperties, false);
  assert.equal(multipartSchema.properties.archive.format, "binary");
  assert.equal(multipartSchema.properties.receipt.$ref, "#/components/schemas/CreateChecklistImportBatchReceiptInput");

  for (const [method, route, operationId] of [
    ["get", "/v1/admin/governed-checklist/import-batches/{importBatchId}", "getAdminChecklistImportBatch"],
    ["get", "/v1/admin/governed-checklist/import-batches/{importBatchId}/files", "listAdminChecklistImportFiles"],
    ["get", "/v1/admin/governed-checklist/import-batches/{importBatchId}/receipts", "listAdminChecklistImportReceipts"],
    ["post", "/v1/admin/governed-checklist/import-batches/{importBatchId}/files/{importFileId}/extraction-reviews", "createAdminChecklistImportFileExtractionReview"],
    ["get", "/v1/admin/governed-checklist/import-batches/{importBatchId}/files/{importFileId}/extraction-review", "getAdminChecklistImportFileExtractionReview"],
    ["post", "/v1/admin/governed-checklist/import-batches/{importBatchId}/files/{importFileId}/identity-resolutions", "resolveAdminChecklistImportFileIdentity"],
    ["post", "/v1/admin/governed-checklist/import-batches/{importBatchId}/files/{importFileId}/candidate-imports", "createAdminExistingChecklistCandidate"],
  ]) {
    assert.equal(operation(document, method, route)?.operationId, operationId, `${method} ${route}`);
  }

  const page = schemas.ChecklistImportExtractionReviewPage;
  assert.ok(page, "the private extraction page is a first-class contract");
  assert.equal(page.additionalProperties, false);
  assert.ok(page.required.includes("proposals"));
  assert.ok(page.required.includes("currentDecisionSet"));
  assert.ok(page.properties.proposals.maxItems <= 100);
  assert.ok(page.properties.proposals.items.$ref);
  assert.ok(schemas.ChecklistImportExtractionReviewProposalView);
  assert.ok(schemas.ChecklistImportExtractionDecisionSetView);

  const decisionSet = dereference(schemas, page.properties.currentDecisionSet);
  assert.equal(decisionSet.discriminator.propertyName, "decisionState");
  assert.equal(decisionSet.oneOf.length, 2);
  for (const variant of decisionSet.oneOf.map((member) => dereference(schemas, member))) {
    assert.equal(variant.additionalProperties, false);
  }
});

test("intake identity and extraction actions are strict discriminated unions", () => {
  const schemas = assembleOpenApi().components.schemas;
  const resolution = schemas.ResolveChecklistImportFileIdentityInput;
  assert.equal(resolution.additionalProperties, false);
  assert.ok(resolution.required.includes("importBatchId"));
  assert.ok(resolution.required.includes("importFileId"));
  assert.ok(resolution.required.includes("expectedResolutionState"));
  assert.equal(
    dereference(schemas, resolution.properties.expectedResolutionState).discriminator.propertyName,
    "resolutionState",
  );

  const candidate = schemas.CreateExistingChecklistCandidateInput;
  assert.equal(candidate.additionalProperties, false);
  assert.equal(
    dereference(schemas, candidate.properties.candidateLineageAction).discriminator.propertyName,
    "action",
  );
  assert.equal(
    dereference(schemas, candidate.properties.extractionDecisionAction).discriminator.propertyName,
    "action",
  );

  const decisions = schemas.ExtractionDecisionInput;
  assert.equal(decisions.discriminator.propertyName, "decisionKind");
  assert.deepEqual(
    decisions.oneOf.map((member) => member.$ref.split("/").at(-1)),
    [
      "AcceptExtractionDecisionInput",
      "SplitExtractionDecisionInput",
      "MergeExtractionDecisionInput",
      "TranscribeExtractionDecisionInput",
      "ExcludeExtractionDecisionInput",
    ],
  );
  for (const member of decisions.oneOf) {
    assert.equal(schemas[member.$ref.split("/").at(-1)].additionalProperties, false);
  }
});
