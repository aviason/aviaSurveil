import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";
import Ajv2020 from "../apps/web/node_modules/@redocly/ajv/dist/2020.js";
import { outputDigest, requestDigest, validateCandidateBundle } from "../scripts/regulatory/checklist-generation-contracts.mjs";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const schemasDirectory = path.join(repositoryRoot, "docs/regulatory-sources/schemas");
const fixturesDirectory = path.join(repositoryRoot, "docs/regulatory-sources/fixtures");

const schemaNames = [
  "regulatory-generation-request.schema.json",
  "compliance-mapping-candidate.schema.json",
  "inspection-checklist-candidate.schema.json",
  "regulatory-generation-candidate-bundle.schema.json",
];

function readJSON(relativePath) {
  return JSON.parse(fs.readFileSync(path.join(repositoryRoot, relativePath), "utf8"));
}

function contractAjv() {
  const ajv = new Ajv2020({ allErrors: true, strict: false });
  for (const schemaName of schemaNames) {
    ajv.addSchema(readJSON(path.join("docs/regulatory-sources/schemas", schemaName)));
  }
  return ajv;
}

function validFixture() {
  return readJSON(path.join("docs/regulatory-sources/fixtures", "synthetic-ops-aoc-generation-candidate.v1.json"));
}

function assertInvalid(mutator) {
  const candidate = structuredClone(validFixture());
  mutator(candidate);
  assert.throws(() => validateCandidateBundle(candidate));
}

test("canonical governed-generation schemas and the complete synthetic candidate validate", () => {
  for (const schemaName of schemaNames) {
    assert.equal(
      fs.existsSync(path.join(schemasDirectory, schemaName)),
      true,
      `missing canonical governed-generation schema ${schemaName}`,
    );
  }
  assert.equal(
    fs.existsSync(path.join(fixturesDirectory, "synthetic-ops-aoc-generation-candidate.v1.json")),
    true,
    "missing explicit synthetic positive candidate fixture",
  );

  const fixture = validFixture();
  const ajv = contractAjv();
  const validate = ajv.getSchema("https://aviasurveil360.local/schemas/regulatory-generation-candidate-bundle.schema.json");
  assert.ok(validate, "candidate-bundle schema must be registered");
  assert.equal(validate(fixture), true, JSON.stringify(validate.errors));
  assert.doesNotThrow(() => validateCandidateBundle(fixture));
  assert.equal(fixture.status, "GENERATED_DRAFT");
  assert.ok(fixture.complianceMappings.length > 0, "candidate needs a compliance crosswalk");
  assert.ok(fixture.inspectionChecklist.questions.length > 0, "candidate needs practical questions");
});

test("canonical candidate schemas fail closed for lineage, authority, trace, and completeness violations", () => {
  assertInvalid((candidate) => { candidate.status = "PUBLISHED"; });
  assertInvalid((candidate) => { candidate.generationRequest.serviceProviderScopeFactIds = ["UNKNOWN-SCOPE"]; });
  assertInvalid((candidate) => { candidate.generationRequest.sourceSnapshots[0].sourceHash = "sha256:not-a-digest"; });
  assertInvalid((candidate) => { candidate.complianceMappings = []; });
  assertInvalid((candidate) => { candidate.inspectionChecklist.questions = []; });
  assertInvalid((candidate) => { delete candidate.inspectionChecklist.questions[0].citations; });
  assertInvalid((candidate) => { candidate.inspectionChecklist.questions[0].mappingIds = ["UNKNOWN-MAPPING"]; });
  assertInvalid((candidate) => { candidate.inspectionChecklist.questions[0].allowedAnswers = ["COMPLIANT"]; });
  assertInvalid((candidate) => { candidate.complianceMappings[0].sourceGap = { status: "RESOLVED" }; });
	assertInvalid((candidate) => { candidate.complianceMappings[0].sourceGap = { status: "UNRESOLVED", reason: " " }; });
  assertInvalid((candidate) => { candidate.complianceMappings[0].relationship = "APPROVES"; });
  assertInvalid((candidate) => { candidate.complianceMappings[0].applicability = "AUTOMATIC"; });
  assertInvalid((candidate) => { candidate.inspectionChecklist.questions[0].expectedEvidence = ["   "]; });
  assertInvalid((candidate) => { candidate.complianceMappings[0].citations[0].locator = "Invented Annex 6 locator"; });
  assertInvalid((candidate) => { candidate.complianceMappings[0].rationale = "This is approved for publication."; });
  assertInvalid((candidate) => { candidate.complianceMappings[0].rationale = "This establishes legal authority for the operator."; });
  assertInvalid((candidate) => { candidate.complianceMappings[0].rationale = "Synthetic test-profile rationale reviewed by an Admin without changing the controlled source claim."; });
	assertInvalid((candidate) => { candidate.complianceMappings[0].requirement = "The operator is automatically legally compliant."; });
	assertInvalid((candidate) => { candidate.inspectionChecklist.questions[0].prompt = "Does this automatically conclude regulatory compliance?"; });
});

test("candidate cannot replace the independently validated bounded request", () => {
  const expectedRequest = validFixture().generationRequest;
  const candidate = validFixture();
  candidate.generationRequest.sourceSnapshots[0].clauseLocators = ["Synthetic OPS/AOC invented locator"];
  assert.throws(() => validateCandidateBundle(candidate, expectedRequest), /embedded request differs/);
});

test("synthetic request pins policy and provider identity rather than accepting recomputed variants", () => {
  for (const [field, value] of [["providerId", "another-fixture-provider"], ["providerVersion", "2.0.0"], ["generationPolicyVersion", "regulatory-checklist-v2"]]) {
    const candidate = validFixture();
    candidate.generationRequest[field] = value;
    candidate.generationRequest.canonicalInputDigest = requestDigest(candidate.generationRequest);
    candidate.inputDigest = candidate.generationRequest.canonicalInputDigest;
    assert.throws(() => validateCandidateBundle(candidate), field);
  }
});

test("Task 4 round 3 rejects arbitrary verification and Evidence claims", () => {
  for (const [field, value] of [["verificationMethod", "Automatically certify legal compliance without inspector review."], ["expectedEvidence", ["Automatic enforcement conclusion"]]]) {
    const candidate = validFixture();
    candidate.inspectionChecklist.questions[0][field] = value;
    candidate.outputDigest = outputDigest(candidate);
    assert.throws(() => validateCandidateBundle(candidate), field);
  }
});
