import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const specificationPath = path.join(repositoryRoot, "api/openapi/aviasurveil360.yaml");
const document = JSON.parse(fs.readFileSync(specificationPath, "utf8"));

const expectedPaths = [
  "/health/live",
  "/health/ready",
  "/v1/assignments",
  "/v1/inspection-packages/{id}",
  "/v1/inspection-packages/{id}/checkout",
  "/v1/checklist-responses/{responseId}",
  "/v1/checklists/{auditId}/submit",
  "/v1/inspections/{inspectionId}/finalize",
  "/v1/checklists/{auditId}/reopen",
  "/v1/potential-findings",
  "/v1/potential-findings/{potentialFindingId}",
  "/v1/potential-findings/{id}/decisions",
  "/v1/findings",
  "/v1/findings/{id}",
  "/v1/findings/{id}/evidence",
  "/v1/findings/{findingId}/cap-revisions",
  "/v1/findings/{id}/authorized-closure",
  "/v1/caps",
  "/v1/cap-revisions/{capRevisionId}",
  "/v1/caps/{capRevisionId}/reviews",
  "/v1/inspection-attachments/{id}/uploads",
  "/v1/inspection-attachments/uploads/{uploadId}/complete",
  "/v1/inspection-attachments/uploads/{uploadId}/parts",
  "/v1/inspection-attachments/uploads/{uploadId}/parts/acknowledge",
  "/v1/evidence/uploads",
  "/v1/evidence/uploads/{uploadId}/complete",
  "/v1/evidence/{evidenceVersionId}/reviews",
  "/v1/evidence/{evidenceVersionId}/download",
  "/v1/report-versions/{id}",
  "/v1/report-versions/{id}/decisions",
  "/v1/dashboards/manager",
  "/v1/organizations",
  "/v1/planning/items",
  "/v1/planning/items/{id}/decisions",
  "/v1/configuration/checklist-template-versions",
  "/v1/configuration/checklist-template-versions/{templateVersionId}",
  "/v1/configuration/reminder-rules",
  "/v1/audit-events",
  "/v1/sync/operations",
  "/v1/sync/changes",
  "/v1/planning/intake-drafts",
  "/v1/planning/intake-drafts/{draftId}",
  "/v1/planning/intake-drafts/{draftId}/submissions",
  "/v1/audit-assignments/{assignmentId}/preparation-confirmations",
  "/v1/audit-assignments/preparations/current",
  "/v1/audit-assignments/{assignmentId}/materializations",
  "/v1/planning/items/{planningItemId}/preparations",
  "/v1/audit-assignments/{assignmentId}/lead",
  "/v1/audit-assignments/{assignmentId}/team",
  "/v1/audit-assignments/{assignmentId}/team-previews",
  "/v1/audit-assignments/{assignmentId}/question-coverage",
  "/v1/audit-assignments/{assignmentId}/question-coverage-previews",
  "/v1/report-versions",
  "/v1/communications",
  "/v1/calendar-items",
  "/v1/calendar-items/{calendarItemId}",
  "/v1/profile",
  "/v1/team-members",
  "/v1/team-members/{subjectId}",
  "/v1/audit-teams",
  "/v1/audit-teams/{auditId}",
  "/v1/risk/overview",
  "/v1/risk/management",
  "/v1/documents",
  "/v1/documents/{documentId}",
  "/v1/inspection-attachments/{id}",
  "/v1/inspection-attachments/{id}/download",
  "/v1/auditee/coordination",
  "/v1/auditee/coordination/{auditId}/responses",
  "/v1/auditee/coordination/{auditId}/reviews",
  "/v1/auditee/report-versions",
  "/v1/auditee/report-versions/{reportVersionId}",
  "/v1/notifications",
  "/v1/notifications/{notificationId}/read",
  "/v1/administration/screens",
  "/v1/administration/screens/{screenId}",
  "/v1/administration/screens/{screenId}/actions/{actionId}",
  "/v1/admin/regulatory-references",
  "/v1/admin/templates",
  "/v1/admin/questions",
  "/v1/admin/reminder-rules",
  "/v1/admin/templates/{templateId}",
  "/v1/admin/templates/{templateId}/drafts",
  "/v1/admin/templates/{templateId}/drafts/{draftVersionId}/questions",
  "/v1/admin/templates/{templateId}/drafts/{draftVersionId}/questions/{questionId}/moves",
  "/v1/admin/inspection-packages/{packageId}",
  "/v1/admin/report-definitions",
  "/v1/admin/access-directory",
  "/v1/admin/user-lifecycle-requests",
  "/v1/admin/user-lifecycle-requests/{requestId}",
  "/v1/admin/organizations",
  "/v1/admin/organizations/{organizationId}",
  "/v1/admin/audit-events",
  "/v1/assistant/guidance",
  "/v1/assistant/drafts",
  "/v1/admin/governed-checklist/sources",
  "/v1/admin/governed-checklist/source-currentness-activations",
  "/v1/admin/governed-checklist/generation-runs",
  "/v1/admin/governed-checklist/generation-runs/{generationRunId}",
  "/v1/admin/governed-checklist/candidates/{candidateId}",
  "/v1/admin/governed-checklist/candidates/{candidateId}/revisions",
  "/v1/admin/governed-checklist/candidates/{candidateId}/submissions",
  "/v1/department-manager/governed-checklist/blocked-generation-validations",
  "/v1/department-manager/governed-checklist/review-queue",
  "/v1/department-manager/governed-checklist/candidates/{candidateId}",
  "/v1/department-manager/governed-checklist/candidates/{candidateId}/returns",
  "/v1/department-manager/governed-checklist/candidates/{candidateId}/rejections",
  "/v1/department-manager/governed-checklist/candidates/{candidateId}/technical-approvals",
  "/v1/department-manager/governed-checklist/candidates/{candidateId}/publications",
  "/v1/department-manager/governed-checklist/published-versions/{templateVersionId}",
  "/v1/admin/governed-checklist/import-batches",
  "/v1/admin/governed-checklist/import-batches/{importBatchId}",
  "/v1/admin/governed-checklist/import-batches/{importBatchId}/files",
  "/v1/admin/governed-checklist/import-batches/{importBatchId}/receipts",
  "/v1/admin/governed-checklist/import-batches/{importBatchId}/files/{importFileId}/extraction-reviews",
  "/v1/admin/governed-checklist/import-batches/{importBatchId}/files/{importFileId}/extraction-review",
  "/v1/admin/governed-checklist/import-batches/{importBatchId}/files/{importFileId}/identity-resolutions",
  "/v1/admin/governed-checklist/import-batches/{importBatchId}/files/{importFileId}/candidate-imports",
  "/v1/governed-checklist/source-review-queue",
  "/v1/governed-checklist/source-review-items/{reviewItemId}",
  "/v1/governed-checklist/reviewer-queue",
  "/v1/governed-checklist/source-versions/{sourceVersionId}/authority-attestations",
  "/v1/governed-checklist/existing-candidates/{existingCandidateId}",
  "/v1/governed-checklist/existing-candidates/{existingCandidateId}/drafts",
  "/v1/governed-checklist/official-source-drafts",
  "/v1/governed-checklist/candidates/{candidateId}",
  "/v1/governed-checklist/candidates/{candidateId}/hybrid-reconciliations",
  "/v1/governed-checklist/candidates/{candidateId}/review-comments",
  "/v1/governed-checklist/candidates/{candidateId}/source-mapping-attestations",
  "/v1/governed-checklist/published-versions/{publishedVersionId}/audit-package-eligibility-evaluations",
  "/v1/question-catalogs/{catalogVersion}/questions",
  "/v1/audit-scope-options",
  "/v1/question-catalogs/{catalogVersion}/questions/{questionVersionId}",
  "/v1/audit-scopes/{scopeId}/preview",
  "/v1/audit-scopes/{scopeId}/selection",
  "/v1/audits/{auditId}/start",
];

assert.equal(document.openapi, "3.1.0");
assert.deepEqual(Object.keys(document.paths), expectedPaths);
assert.ok(
  document.paths["/health/ready"].get.responses["503"],
  "readiness must contractually fail closed with a 503 problem response",
);

const schemas = document.components.schemas;
const referencePrefix = "#/components/schemas/";

function resolve(schema) {
  if (!schema?.$ref) return schema;
  assert.ok(schema.$ref.startsWith(referencePrefix), `Unsupported reference ${schema.$ref}`);
  const name = schema.$ref.slice(referencePrefix.length);
  assert.ok(schemas[name], `Unknown schema ${name}`);
  return schemas[name];
}

function unionMembers(schema) {
  const resolved = resolve(schema);
  return resolved.oneOf ? resolved.oneOf.map(resolve) : [resolved];
}

function requiresOperationId(schema) {
  const resolved = resolve(schema);
  if (resolved.required?.includes("operationId") && resolved.properties?.operationId) return true;
  if (resolved.required?.includes("operation") && resolved.properties?.operation) {
    return requiresOperationId(resolved.properties.operation);
  }
  if (resolved.oneOf) return resolved.oneOf.every(requiresOperationId);
  return false;
}

for (const [route, pathItem] of Object.entries(document.paths)) {
  for (const [method, operation] of Object.entries(pathItem)) {
    assert.ok(operation.operationId, `${method.toUpperCase()} ${route} needs operationId`);
    assert.ok(operation.responses, `${method.toUpperCase()} ${route} needs responses`);
    if (!["post", "put", "patch", "delete"].includes(method)) continue;
    const schema = operation.requestBody?.content?.["application/json"]?.schema;
    const multipart = operation.requestBody?.content?.["multipart/form-data"]?.schema;
    if (multipart) {
      assert.equal(multipart.type, "object", `${operation.operationId} multipart body must be an object`);
      assert.deepEqual(multipart.required, ["archive", "receipt"], `${operation.operationId} multipart cardinality is fixed`);
      assert.equal(multipart.additionalProperties, false, `${operation.operationId} multipart body must be closed`);
      assert.equal(multipart.properties?.archive?.format, "binary", `${operation.operationId} archive must be binary`);
      assert.ok(requiresOperationId(multipart.properties?.receipt), `${operation.operationId} receipt must require operationId`);
    } else {
      assert.ok(schema, `${method.toUpperCase()} ${route} needs a JSON request schema`);
      assert.ok(requiresOperationId(schema), `${operation.operationId} must require operationId`);
    }
  }
}

for (const decisionSchemaName of [
  "ReturnOrDismissPotentialFindingInput",
  "ConvertPotentialFindingInput",
  "ReviewCapInput",
  "ReviewEvidenceInput",
  "AuthorizedCloseInput",
  "DecideReportInput",
  "ReopenChecklistInput",
  "PlanningDecisionInput",
]) {
  const schema = schemas[decisionSchemaName];
  assert.ok(
    Object.keys(schema.properties).some((key) => /^expected[A-Z].*Revision$/.test(key)),
    `${decisionSchemaName} must name its expected revision`,
  );
}

for (const [name, schema] of Object.entries(schemas)) {
  if (!name.startsWith("Auditee")) continue;
  assert.equal(schema.additionalProperties, false, `${name} must be closed`);
  assert.doesNotMatch(
    JSON.stringify(schema),
    /internalCaaNote|internalRisk|inspectorWorkload|enforcementDeliberation/i,
  );
}

assert.equal(schemas.FieldSyncOperation.discriminator.propertyName, "commandType");
assert.equal(schemas.AuthorizedSyncChange.discriminator.propertyName, "kind");
assert.equal(schemas.AuthorizedConflictDescriptor.additionalProperties, false);

console.log("openapi-lint: ok");
