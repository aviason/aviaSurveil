import assert from "node:assert/strict";
import test from "node:test";

import { assembleOpenApi } from "../../../scripts/bundle-openapi.mjs";

const document = assembleOpenApi();

const requiredMutations = new Map([
  ["/v1/admin/organizations", "createAdminOrganization"],
  ["/v1/admin/reminder-rules", "createReminderRule"],
  ["/v1/planning/intake-drafts", "createPlanningIntakeDraft"],
  ["/v1/planning/items/{planningItemId}/preparations", "prepareAudit"],
  ["/v1/report-versions", "createReportVersion"],
]);

test("normal profiles expose only non-publication clean-state mutations", () => {
  for (const [path, operationId] of requiredMutations) {
    const operation = document.paths[path]?.post;
    assert.ok(operation, `POST ${path} is required`);
    assert.equal(operation.operationId, operationId);
    assert.ok(operation.requestBody?.content?.["application/json"]?.schema);
    assert.ok(operation.responses?.["201"]?.content?.["application/json"]?.schema);
    assert.ok(operation.responses?.default, `${operationId} must expose Problem responses`);
  }
});

test("clean-state operations cannot expose test, reset, fixture, or seed semantics", () => {
  const serialized = JSON.stringify(
    [...requiredMutations.keys()].map((path) => document.paths[path]),
  ).toLowerCase();
  for (const forbidden of ["__test", "reset", "fixture", "seed", "canonical-header"]) {
    assert.equal(serialized.includes(forbidden), false, `${forbidden} is forbidden`);
  }
});

test("clean-state request schemas require idempotency and expected-version boundaries", () => {
  const schemaNames = [
    "CreateAdminOrganizationInput",
    "CreateReminderRuleInput",
    "CreatePlanningIntakeDraftInput",
    "PrepareAuditInput",
    "CreateReportVersionInput",
  ];
  for (const name of schemaNames) {
    const schema = document.components.schemas[name];
    assert.ok(schema, `${name} is required`);
    assert.equal(schema.additionalProperties, false);
    assert.ok(schema.required.includes("operationId"));
    assert.ok(schema.required.includes("idempotencyKey"));
  }
  assert.ok(
    document.components.schemas.PrepareAuditInput.required.includes(
      "expectedPlanningRevision",
    ),
  );
});

test("normal profiles forbid direct published checklist creation", () => {
  // Production break: restoring the ordinary Admin direct-publication command
  // would make this assertion fail.
  assert.equal(
    document.paths["/v1/admin/checklist-template-versions"],
    undefined,
    "normal profiles must not expose an Admin command that creates PUBLISHED checklist versions",
  );
  assert.equal(document.components.schemas.CreateChecklistTemplateVersionInput, undefined);
  assert.equal(document.components.schemas.PublishedChecklistQuestionInput, undefined);
});
