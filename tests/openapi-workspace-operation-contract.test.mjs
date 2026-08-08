import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const document = JSON.parse(readFileSync("api/openapi/aviasurveil360.yaml", "utf8"));
const prefix = "/v1/preprod/aga-demo-workspace";
const queryPaths = [
  `${prefix}/classification/query`,
  `${prefix}/recommendations/query`,
  `${prefix}/lifecycle/query`,
];
const commandPaths = [
  `${prefix}/classification/commands`,
  `${prefix}/recommendations/commands`,
  `${prefix}/lifecycle/commands`,
  `${prefix}/admin/commands`,
];

test("workspace donor paths are absent from the normal OpenAPI contract", () => {
  for (const path of [...queryPaths, ...commandPaths]) {
    assert.equal(document.paths[path], undefined, `${path} must remain donor-only`);
  }
});

test("workspace request schemas are closed and carry the generic command envelope", () => {
  const query = document.components.schemas.AGADemoWorkspaceQuery;
  const command = document.components.schemas.AGADemoWorkspaceCommand;
  assert.equal(query.additionalProperties, false);
  assert.equal(command.additionalProperties, false);
  assert.deepEqual(command.required, ["operationId", "idempotencyKey", "expectedGenerationId"]);
  assert.ok(command.properties.expectedLifecycleRevision);
  assert.ok(command.properties.expectedLifecycleDigest);
  assert.ok(command.properties.expectedGenerationSealDigest);
});
