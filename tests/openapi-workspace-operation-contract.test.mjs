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

function parameterNames(operation) {
  return (operation.parameters ?? []).map((parameter) => parameter.$ref?.split("/").at(-1) ?? parameter.name);
}

test("workspace OpenAPI marks query and command operation kinds exactly", () => {
  for (const path of queryPaths) {
    const operation = document.paths[path].post;
    assert.equal(operation["x-operation-kind"], "query", path);
    assert.equal(operation["x-neutral-denial"], true, path);
    assert.ok(parameterNames(operation).includes("CsrfToken"), `${path} requires CSRF`);
    assert.ok(!parameterNames(operation).includes("IdempotencyKey"), `${path} must not advertise idempotency`);
    assert.ok(!parameterNames(operation).includes("ExpectedRevision"), `${path} must not advertise If-Match`);
    assert.equal(operation.responses["401"], undefined, `${path} must not distinguish 401`);
    assert.equal(operation.responses["403"], undefined, `${path} must not distinguish 403`);
    assert.ok(operation.responses["404"], `${path} must provide neutral 404`);
  }
  for (const path of commandPaths) {
    const operation = document.paths[path].post;
    assert.equal(operation["x-operation-kind"], "command", path);
    assert.equal(operation["x-neutral-denial"], true, path);
    assert.deepEqual(new Set(parameterNames(operation)), new Set(["IdempotencyKey", "CsrfToken", "ExpectedRevision"]), path);
    assert.equal(operation.responses["401"], undefined, `${path} must not distinguish 401`);
    assert.equal(operation.responses["403"], undefined, `${path} must not distinguish 403`);
    assert.ok(operation.responses["404"], `${path} must provide neutral 404`);
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
