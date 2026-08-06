import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const source = JSON.parse(readFileSync("api/openapi/source/paths/platform.json", "utf8"));
const schemas = JSON.parse(readFileSync("api/openapi/source/schemas/platform.json", "utf8"));
const prefix = "/v1/preprod/aga-demo-workspace";
const expected = {
  [`${prefix}/capability`]: ["get"],
  [`${prefix}/classification/query`]: ["post"],
  [`${prefix}/classification/commands`]: ["post"],
  [`${prefix}/recommendations/commands`]: ["post"],
  [`${prefix}/recommendations/query`]: ["post"],
  [`${prefix}/lifecycle/query`]: ["post"],
  [`${prefix}/lifecycle/commands`]: ["post"],
  [`${prefix}/admin/commands`]: ["post"],
};

test("workspace source exposes the fixed route matrix and explicit role declarations", () => {
  assert.deepEqual(Object.fromEntries(Object.entries(source).filter(([path]) => path.startsWith(prefix)).map(([path, item]) => [path, Object.keys(item)])), expected);
  for (const [path, methods] of Object.entries(expected)) {
    const operation = source[path][methods[0]];
    assert.equal(operation["x-neutral-denial"], true, path);
    assert.ok(Array.isArray(operation["x-authorized-roles"]), `${path} must declare roles`);
    assert.ok(operation["x-authorized-roles"].length > 0, `${path} must not use an empty role declaration`);
  }
});

test("the frozen legacy AGA prefix remains five Admin-only reads", () => {
  const legacyPrefix = "/v1/admin/governed-checklist/aga-candidate-demo";
  const routes = Object.entries(source).filter(([path]) => path.startsWith(legacyPrefix));
  assert.equal(routes.length, 5);
  for (const [, item] of routes) assert.deepEqual(Object.keys(item), ["get"]);
});

test("lifecycle routes and closed bodies expose the append-only contract", () => {
  const lifecycleQuery = source[`${prefix}/lifecycle/query`].post;
  const lifecycleCommands = source[`${prefix}/lifecycle/commands`].post;
  assert.equal(lifecycleQuery["x-lifecycle-projection"], "organization-scoped-public-or-CAA-projection");
  assert.equal(lifecycleCommands["x-lifecycle-cas"], "expectedLifecycleRevision-and-expectedLifecycleDigest");
  const query = schemas.AGADemoWorkspaceQuery;
  for (const property of ["inspectionId", "findingId", "capId", "evidenceId"]) assert.ok(query.properties[property], property);
  const command = schemas.AGADemoWorkspaceCommand;
  for (const property of ["inspectionId", "expectedLifecycleRevision", "expectedLifecycleDigest", "potentialFindingId", "findingId", "answer", "commentToAuditee", "internalCaaNote", "capRequired", "evidenceRequired", "dueDateRequired", "dueDate", "outcome"]) assert.ok(command.properties[property], property);
  assert.deepEqual(command.properties.answer.enum, ["COMPLIANT", "NON_COMPLIANT", "OBSERVATION", "NOT_APPLICABLE", "NOT_CHECKED"]);
  assert.deepEqual(command.properties.outcome.enum, ["ACCEPT", "REJECT", "MORE_INFORMATION_REQUESTED", "CLOSE", "PARTIALLY_CLOSE", "NOT_CLOSE", "REQUEST_MORE_INFORMATION"]);
  for (const name of ["AGADemoWorkspaceLifecycleProjection", "AGADemoWorkspaceLifecycleCAAProjection", "AGADemoWorkspaceLifecycleAuditeeProjection"]) {
    assert.equal(schemas[name].additionalProperties, false, `${name} must be closed`);
  }
  assert.equal(schemas.AGADemoWorkspaceQueryResponse.properties.lifecycle.$ref, "#/components/schemas/AGADemoWorkspaceLifecycleProjection");
  assert.equal(schemas.AGADemoWorkspaceQueryResponse.properties.lifecycleCaa.$ref, "#/components/schemas/AGADemoWorkspaceLifecycleCAAProjection");
  assert.equal(schemas.AGADemoWorkspaceQueryResponse.properties.lifecycleAuditee.$ref, "#/components/schemas/AGADemoWorkspaceLifecycleAuditeeProjection");
});

test("successor manager package contract is explicit and bounded", () => {
  const query = schemas.AGADemoWorkspaceQuery;
  for (const operation of ["GET_SIMULATION_SETUP", "GET_CURRENT_RECOMMENDATION", "GET_CURRENT_INSPECTION", "GET_INSPECTION_QUESTION_PAGE"]) {
    assert.ok(query.properties.operationId.enum.includes(operation), operation);
  }
  for (const name of ["AGADemoWorkspaceClassificationReviewItem", "AGADemoWorkspaceBatchPreview", "AGADemoWorkspaceSimulationSetup", "AGADemoWorkspaceLifecycleQuestionPageItem"]) {
    assert.equal(schemas[name].additionalProperties, false, name);
  }
  assert.equal(schemas.AGADemoWorkspaceQueryResponse.properties.items.items.$ref, "#/components/schemas/AGADemoWorkspaceClassificationReviewItem");
  for (const property of ["includeEligible", "includeEligibilityReason"]) {
    assert.ok(schemas.AGADemoWorkspaceClassificationReviewItem.required.includes(property), property);
    assert.ok(schemas.AGADemoWorkspaceClassificationReviewItem.properties[property], property);
  }
  assert.ok(schemas.AGADemoWorkspaceQueryResponse.properties.questionPage);
  assert.equal(schemas.AGADemoWorkspaceCommand.properties.readinessEventId.readOnly, true);
  assert.ok(schemas.AGADemoWorkspaceCommand.properties.simulationSetupDigest);
  assert.ok(schemas.AGADemoWorkspaceCommand.properties.inspectorSelectionPin);
  assert.ok(schemas.AGADemoWorkspaceCommand.properties.leadSelectionPin);
});
