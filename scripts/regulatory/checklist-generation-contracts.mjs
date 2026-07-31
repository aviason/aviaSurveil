import { createHash } from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import Ajv2020 from "../../apps/web/node_modules/@redocly/ajv/dist/2020.js";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const schemaDirectory = path.join(repositoryRoot, "docs/regulatory-sources/schemas");
const schemaNames = [
  "regulatory-generation-request.schema.json",
  "compliance-mapping-candidate.schema.json",
  "inspection-checklist-candidate.schema.json",
  "regulatory-generation-candidate-bundle.schema.json",
];

export const allowedAnswers = ["COMPLIANT", "NON_COMPLIANT", "OBSERVATION", "NOT_APPLICABLE", "NOT_CHECKED"];

export function canonicalJSON(value) {
  if (value === null || typeof value !== "object") return JSON.stringify(value);
  if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(", ")}]`;
  return `{${Object.keys(value).sort((left, right) => left.length - right.length || left.localeCompare(right)).map((key) => `${JSON.stringify(key)}: ${canonicalJSON(value[key])}`).join(", ")}}`;
}

export function sha256(value) {
  return `sha256:${createHash("sha256").update(canonicalJSON(value)).digest("hex")}`;
}

export function requestDigest(request) {
  const unsigned = structuredClone(request);
  delete unsigned.canonicalInputDigest;
  return sha256(unsigned);
}

export function outputDigest(bundle) {
  return sha256({
    complianceMappings: bundle.complianceMappings,
    inspectionChecklist: bundle.inspectionChecklist,
  });
}

function loadSchema(name) {
  return JSON.parse(fs.readFileSync(path.join(schemaDirectory, name), "utf8"));
}

export function schemaValidator() {
  const ajv = new Ajv2020({ allErrors: true, strict: false });
  for (const name of schemaNames) ajv.addSchema(loadSchema(name));
  const validate = ajv.getSchema("https://aviasurveil360.local/schemas/regulatory-generation-candidate-bundle.schema.json");
  if (!validate) throw new Error("candidate-bundle schema was not registered");
  return validate;
}

function requestSchemaValidator() {
  const ajv = new Ajv2020({ allErrors: true, strict: false });
  const validate = ajv.compile(loadSchema("regulatory-generation-request.schema.json"));
  return validate;
}

const syntheticRegistry = {
  requestId: "GENREQ-SYNTHETIC-OPS-AOC-0001",
  organizationId: "ORG-SYNTHETIC-AOC",
  scopeFactId: "SCOPE-SYNTHETIC-AOC",
  targetId: "TARGET-SYNTHETIC-AOC",
  sourceSnapshotId: "SOURCE-SYNTHETIC-OPS-AOC",
  sourceHash: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
  clauseId: "CLAUSE-SYNTHETIC-OPS-AOC-1",
  clauseLocator: "Synthetic OPS/AOC 1",
  partitionId: "PARTITION-SYNTHETIC-INPUT",
  stableRowId: "CC:SYNTHETIC:OPS:AOC:1",
};

const syntheticSupportedClaims = {
  requirement: "Synthetic controlled training requirement for verifying ramp safety evidence.",
  rationale: "This synthetic source is test-profile-only and proves the bounded import workflow without asserting real authority.",
  prompt: "Can the inspector reconcile the synthetic ramp-safety evidence with the controlled synthetic requirement?",
  verificationMethod: "Physical observation and controlled-record reconciliation",
  expectedEvidence: ["Synthetic inspection observation", "Synthetic controlled record"],
};

function sameArray(left, right) {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

function validateKnownRequest(request) {
  assert(request.requestId === syntheticRegistry.requestId, "unknown or unresolved generation request identity");
  assert(request.organizationId === syntheticRegistry.organizationId, "unknown organization identity");
  assert(sameArray(request.serviceProviderScopeFactIds, [syntheticRegistry.scopeFactId]), "unknown provider-scope fact identity");
  assert(sameArray(request.serviceProviderTypes, ["AIR_OPERATOR"]), "unknown provider type identity");
  assert(request.target.targetId === syntheticRegistry.targetId && request.target.kind === "ORGANIZATION", "unknown or incompatible target identity");
  assert(request.sourceSnapshots.length === 1, "unknown source snapshot identity");
  const source = request.sourceSnapshots[0];
  assert(source.sourceSnapshotId === syntheticRegistry.sourceSnapshotId && source.sourceHash === syntheticRegistry.sourceHash && sameArray(source.clauseIds, [syntheticRegistry.clauseId]) && sameArray(source.clauseLocators, [syntheticRegistry.clauseLocator]), "unknown source, source hash, clause identity, or clause locator");
  assert(request.secondaryCrosswalkPartition.partitionId === syntheticRegistry.partitionId && sameArray(request.secondaryCrosswalkPartition.stableRowIds, [syntheticRegistry.stableRowId]), "unknown or non-input crosswalk partition identity");
  assert(sameArray(request.unresolvedSourceGaps, []), "synthetic request cannot claim unresolved real-source gaps");
  assert(request.generationPolicyVersion === "regulatory-checklist-v1" && request.providerCatalogVersion === "1.0.0" && request.providerId === "deterministic-regulatory-fixture" && request.providerVersion === "1.0.0", "synthetic request has an unpinned policy or provider identity");
  assert(request.canonicalInputDigest === requestDigest(request), "canonical input digest does not match bounded request content");
}

export function validateRealOPSAOCRequest(request) {
  const validate = requestSchemaValidator();
  assert(validate(request), `real request does not validate against canonical request schema: ${JSON.stringify(validate.errors)}`);
  const gaps = [
    ["CONTROLLED_PROCEDURE", "The controlled NCAA Operations surveillance/ramp-inspection procedure has not been supplied."],
    ["PART_140_AUTHORITY", "Current Part 140 authority and supersession require source-owner confirmation."],
    ["PART_127_APPLICABILITY", "Exact Part 127 operation/configuration applicability requires Department Manager confirmation."],
  ];
  const source = request.sourceSnapshots[0];
  assert(request.requestId === "GENREQ-OPS-AOC-0001" && request.organizationId === "ORG-FLY-NAMIBIA" && sameArray(request.serviceProviderScopeFactIds, ["SCOPE-OPS-AOC-SOURCE-BOUND"]) && sameArray(request.serviceProviderTypes, ["AIR_OPERATOR"]) && request.providerCatalogVersion === "1.0.0" && request.inspectionType === "RAMP_INSPECTION" && request.target.targetId === "TARGET-OPS-AOC-SOURCE-BOUND" && request.target.kind === "ORGANIZATION", "real request has an untracked scope or target identity");
  assert(request.sourceSnapshots.length === 1 && source.sourceSnapshotId === "NCAA-CC-ANNEX6-PARTI-A610-SUPPLIED-2026-07-28" && source.sourceHash === "sha256:13fe82d1767320443f91ed61cf7d3b4bba0ea24f217fad45bbd9cae5fc682af2" && sameArray(source.clauseIds, ["NCAA-CC-A610-4.2.2.2"]) && sameArray(source.clauseLocators, ["Annex 6 Part I 4.2.2.2"]), "real request has an untracked source, hash, clause, or locator");
  assert(request.secondaryCrosswalkPartition.partitionId === "CC-OPS-TRAIN-1" && sameArray(request.secondaryCrosswalkPartition.stableRowIds, ["CC:NAMB:ANNEX6:4.2.2.2"]) && request.unresolvedSourceGaps.length === gaps.length && request.unresolvedSourceGaps.every((gap, index) => gap.gapId === gaps[index][0] && gap.reason === gaps[index][1]) && request.generationPolicyVersion === "regulatory-checklist-v1" && request.providerId === "imported-result-only" && request.providerVersion === "1.0.0" && sameArray(request.requestedOutputs, ["COMPLIANCE_MAPPING", "INSPECTION_CHECKLIST"]), "real request has an untracked crosswalk, source-gap, or provider binding");
  assert(request.canonicalInputDigest === requestDigest(request), "real request canonical digest does not match its complete bounded content");
  return request;
}

function validateCitations(citations, request) {
  for (const citation of citations) {
    const source = request.sourceSnapshots.find((snapshot) => snapshot.sourceSnapshotId === citation.sourceSnapshotId);
    assert(source && source.sourceHash === citation.sourceHash && source.clauseIds.includes(citation.clauseId) && source.clauseLocators.includes(citation.locator), "citation does not resolve to the exact bounded source snapshot, clause, and locator");
  }
}

// expectedRequest is the independently validated request held by the caller.
// Keeping it separate prevents a candidate from self-authenticating its own
// embedded request through a tautological comparison.
export function validateCandidateBundle(bundle, expectedRequest = bundle.generationRequest) {
	const validate = schemaValidator();
	assert(validate(bundle), `candidate does not validate against canonical schemas: ${JSON.stringify(validate.errors)}`);
	validateKnownRequest(expectedRequest);
	assert(canonicalJSON(bundle.generationRequest) === canonicalJSON(expectedRequest), "candidate embedded request differs from the validated bounded request");
	assert(bundle.inputDigest === expectedRequest.canonicalInputDigest, "candidate input digest differs from request digest");
  assert(bundle.outputDigest === outputDigest(bundle), "candidate output digest does not match content");

  const mappingIds = new Set();
  for (const mapping of bundle.complianceMappings) {
    assert(!mappingIds.has(mapping.mappingId), "duplicate mapping identity");
    mappingIds.add(mapping.mappingId);
		assert(["ADDRESSES", "PARTIALLY_ADDRESSES", "NOT_ADDRESSED", "CONTEXT_ONLY"].includes(mapping.relationship), "unsupported mapping relationship");
		assert(["DIRECT", "CONDITIONAL", "CONTEXTUAL"].includes(mapping.applicability), "unsupported mapping applicability");
		validateCitations(mapping.citations, expectedRequest);
    assert(mapping.sourceGap === null, "synthetic registry does not support an unregistered source-gap claim");
    assert(mapping.requirement === syntheticSupportedClaims.requirement && mapping.rationale === syntheticSupportedClaims.rationale, "unsupported synthetic mapping claim text");
  }
  const questionIds = new Set();
  for (const question of bundle.inspectionChecklist.questions) {
    assert(!questionIds.has(question.questionId), "duplicate question identity");
    questionIds.add(question.questionId);
    assert(new Set(question.mappingIds).size === question.mappingIds.length && question.mappingIds.every((mappingId) => mappingIds.has(mappingId)), "question references an unknown or duplicate mapping identity");
		validateCitations(question.citations, expectedRequest);
    assert(question.expectedEvidence.every((evidence) => evidence.trim() !== "") && sameArray(question.allowedAnswers, allowedAnswers), "question expected evidence or allowed answers are incomplete");
    assert(question.prompt === syntheticSupportedClaims.prompt && question.verificationMethod === syntheticSupportedClaims.verificationMethod && sameArray(question.expectedEvidence, syntheticSupportedClaims.expectedEvidence), "unsupported synthetic question claim text");
  }
  return bundle;
}

export function readTrackedJSON(relativePath) {
  return JSON.parse(fs.readFileSync(path.join(repositoryRoot, relativePath), "utf8"));
}

export { repositoryRoot, syntheticRegistry };
