import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import { assembleOpenApi } from "../../../scripts/bundle-openapi.mjs";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..", "..");
const generatedTypeScriptPath = path.join(
  repositoryRoot,
  "apps/web/src/generated/transport/api-types.ts",
);
const generatedGoPath = path.join(
  repositoryRoot,
  "apps/api/internal/httpapi/generated/api.gen.go",
);

test("generated normal clients cannot invoke direct checklist publication", () => {
  // Production break: exposing this Admin operation in OpenAPI regeneration
  // would make one or both generated client assertions fail.
  const document = assembleOpenApi();
  const operationIds = Object.values(document.paths).flatMap((pathItem) =>
    Object.values(pathItem).map((operation) => operation.operationId),
  );
  const generatedTypeScript = fs.readFileSync(generatedTypeScriptPath, "utf8");
  const generatedGo = fs.readFileSync(generatedGoPath, "utf8");

  assert.equal(operationIds.includes("createChecklistTemplateVersion"), false);
  assert.equal(generatedTypeScript.includes('"/v1/admin/checklist-template-versions"'), false);
  assert.equal(generatedTypeScript.includes("CreateChecklistTemplateVersionInput"), false);
  assert.equal(generatedGo.includes("CreateChecklistTemplateVersionInput"), false);
});
