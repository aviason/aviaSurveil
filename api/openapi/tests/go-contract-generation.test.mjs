import assert from "node:assert/strict";
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..", "..");
const specificationPath = path.join(repositoryRoot, "api/openapi/aviasurveil360.yaml");
const generatedPath = path.join(repositoryRoot, "apps/api/internal/httpapi/generated/api.gen.go");
const generatorPath = path.join(repositoryRoot, "scripts/generate-go-contracts.mjs");
const specificationBytes = fs.readFileSync(specificationPath);
const specification = JSON.parse(specificationBytes);

assert.ok(fs.existsSync(generatorPath), "the repository-owned Go contract generator must be checked in");
assert.ok(fs.existsSync(generatedPath), "the generated Go handler contract must be checked in");

const generator = fs.readFileSync(generatorPath, "utf8");
assert.match(generator, /GENERATOR_VERSION = "1\.0\.0"/);

const generated = fs.readFileSync(generatedPath, "utf8");
const specificationHash = crypto.createHash("sha256").update(specificationBytes).digest("hex");
assert.match(generated, new RegExp(`OpenAPI-SHA256: ${specificationHash}`));
assert.match(generated, /type StrictServerInterface interface/);
for (const unionName of [
  "GovernedRegulatoryTraceContent",
  "GovernedCandidateLineage",
  "GovernedSourceReviewDetailView",
  "ExtractionDecisionInput",
]) {
  assert.match(generated, new RegExp(`type ${unionName} struct`));
  assert.doesNotMatch(generated, new RegExp(`type ${unionName} = json\\.RawMessage`));
}
assert.match(generated, /Archive\s+io\.Reader\s+`multipart:"archive"`/);
assert.match(generated, /Receipt\s+CreateChecklistImportBatchReceiptInput\s+`json:"receipt"`/);

for (const pathItem of Object.values(specification.paths)) {
  for (const operation of Object.values(pathItem)) {
    if (!operation?.operationId) continue;
    assert.match(generated, new RegExp(`operationId: ${operation.operationId}\\b`));
  }
}

console.log("go-contract-generation: ok");
