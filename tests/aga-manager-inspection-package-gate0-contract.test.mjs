import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const schemas = JSON.parse(readFileSync("api/openapi/source/schemas/platform.json", "utf8"));
const paths = JSON.parse(readFileSync("api/openapi/source/paths/platform.json", "utf8"));
const routeSource = readFileSync("apps/web/src/app/aga-demo-workspace-routes.tsx", "utf8");
const plan = readFileSync("docs/exec-plans/active/2026-08-05-department-manager-aga-inspection-package-demo-plan.md", "utf8");
const prefix = "/v1/preprod/aga-demo-workspace";

test("Gate 0 freezes the bounded text, setup, current-object, and lifecycle page operations", () => {
  const queryOperations = schemas.AGADemoWorkspaceQuery.properties.operationId.enum;
  for (const operation of ["GET_SIMULATION_SETUP", "GET_CURRENT_RECOMMENDATION", "GET_CURRENT_INSPECTION", "GET_INSPECTION_QUESTION_PAGE"]) {
    assert.ok(queryOperations.includes(operation), operation);
  }
  assert.ok(paths[`${prefix}/recommendations/query`]?.post, "recommendations query route");
  assert.equal(paths[`${prefix}/recommendations/query`].post["x-operation-kind"], "query");
  assert.match(routeSource, /\/department-manager\/aga-demo-workspace\/inspection-package/u);
  assert.ok(schemas.AGADemoWorkspaceClassificationReviewItem, "transport-only review item");
  assert.ok(schemas.AGADemoWorkspaceBatchPreview, "closed server-issued batch preview");
  assert.ok(schemas.AGADemoWorkspaceSimulationSetup, "read-only simulation setup");
  assert.ok(schemas.AGADemoWorkspaceLifecycleQuestionPageItem, "transient lifecycle text page");
});

test("Gate 0 freezes the no-persistence and fail-closed contract", () => {
  const reviewItem = schemas.AGADemoWorkspaceClassificationReviewItem;
  assert.equal(reviewItem.additionalProperties, false);
  assert.ok(reviewItem.properties.questionText);
  assert.ok(reviewItem.properties.questionTextDigest);
  assert.equal(reviewItem.properties.questionTextDigest.readOnly, true);
  const queryResponse = schemas.AGADemoWorkspaceQueryResponse;
  assert.ok(queryResponse.properties.reviewItems, "review items are a response projection");
  assert.ok(queryResponse.properties.questionPage, "lifecycle text is a separate query projection");
  assert.equal(schemas.AGADemoWorkspaceLifecycleQuestionSnapshot.properties.questionText, undefined);
  assert.equal(schemas.AGADemoWorkspaceLifecycleQuestionSnapshot.properties.questionTextDigest, undefined);
  assert.equal(schemas.AGADemoWorkspaceCommand.properties.readinessEventId.readOnly, true);
  assert.ok(schemas.AGADemoWorkspaceCommand.properties.previewId, "server-issued preview is consumed by ID and digest");
  assert.match(plan, /at most 25/u);
  assert.match(plan, /fails the complete response/u);
  assert.match(plan, /no client-generated preview ID/u);
});

test("Gate 0 freezes the exact Manager package route without URL state", () => {
  assert.match(routeSource, /path: "\/department-manager\/aga-demo-workspace"/u);
  assert.doesNotMatch(routeSource, /inspection-package:\w|inspection-package\/:[^}]+/u);
  assert.match(plan, /Do not place question identities, text, search terms, or object IDs in query\s+strings or route parameters/u);
});
