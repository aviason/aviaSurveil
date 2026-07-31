import assert from "node:assert/strict";
import test from "node:test";

import { assembleOpenApi } from "../../../scripts/bundle-openapi.mjs";

test("governed checklist OpenAPI keeps the separate import, review, and publication contract", () => {
  const document = assembleOpenApi();
  const paths = document.paths;
  for (const [path, operationId] of [
    ["/v1/admin/governed-checklist/generation-runs", "importAdminGovernedGenerationRun"],
    ["/v1/department-manager/governed-checklist/candidates/{candidateId}/technical-approvals", "approveDepartmentManagerGovernedCandidate"],
    ["/v1/department-manager/governed-checklist/candidates/{candidateId}/publications", "publishDepartmentManagerGovernedCandidate"],
  ]) {
    const operations = Object.values(paths[path] ?? {});
    assert.equal(operations.some((operation) => operation.operationId === operationId), true, `${path} must expose ${operationId}`);
  }
  assert.equal(paths["/v1/admin/governed-checklist/candidates/{candidateId}/publications"], undefined);
  const schemas = document.components.schemas;
  for (const name of ["GovernedGenerationRequestInput", "GovernedCandidateBundleInput", "GovernedCandidateView", "GovernedPublicationView"]) {
    assert.ok(schemas[name], `missing governed contract schema ${name}`);
  }
});
