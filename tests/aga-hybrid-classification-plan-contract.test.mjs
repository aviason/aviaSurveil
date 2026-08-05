import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import Ajv2020 from "../apps/web/node_modules/@redocly/ajv/dist/2020.js";
import { throughOwnerSlice } from "../scripts/check-aga-hybrid-created-files.mjs";
import * as classificationBatchPreparer from "../scripts/prepare-aga-hybrid-classification-batches.mjs";

const packagePath =
  "deliverables/aga-all-forms-source-risk-draft-2026-08-01/AGA_ALL_FORMS_SOURCE_RISK_DRAFT.json";
const researchZipPath =
  "/Users/marlonjd/.codex/attachments/a2fa9639-5e9a-4e5d-a68d-0b38ef797b75/AGA_INDEPENDENT_RESEARCH_DELIVERABLES_2026-08-02.zip";
const workbookPath =
  "/tmp/codex-remote-attachments/019fcd4b-4cdb-7672-bf84-c703b0a24a58/39DD3E5A-E6A8-483B-AF11-706021BCEE53/1-AVIASURVEIL360_System_Design_Matrix.xlsx";
const preparerPath = "scripts/prepare-aga-hybrid-classification-batches.mjs";
const discoveryPromptPath =
  "docs/product-specs/data-and-rules/AGA_HYBRID_VOCABULARY_DISCOVERY_PROMPT.md";
const discoveryDirectory = "deliverables/aga-question-classification-contract-v1";
const classificationSpecPath =
  "docs/product-specs/data-and-rules/AGA_HYBRID_QUESTION_CLASSIFICATION.md";
const taxonomyPath =
  "docs/product-specs/data-and-rules/aga-question-classification-taxonomy.v1.json";
const classificationSchemaPath =
  "docs/product-specs/data-and-rules/aga-question-classification.schema.json";
const classificationPromptPath =
  "docs/product-specs/data-and-rules/AGA_HYBRID_CLASSIFICATION_PROMPT.md";
const createdFileCheckerPath = "scripts/check-aga-hybrid-created-files.mjs";
const createdFileInventoryPath =
  "tests/fixtures/aga-hybrid-created-file-inventory.v1.json";

const frozenGate0BFileDigests = {
  specification: "1122cab81f1b7ab2648ae9fbbfbcd6dfe3378dd628e07c85864c9bd50617bf37",
  taxonomy: "6ed9d19755b7bdd4891409b6de8bb4ec367ade84073ca80428e4914591aec558",
  schema: "756f47e40ba762ef45d7fec8c35e23248632d96ad243bbdb655ca3b5581ff462",
  prompt: "2fb92dcebf366cbfdcaf5bebdb72f4b01a2b9f8adf41512fd257df9faa039f80",
};

const sealedClassificationItemFields = [
  "packageVersion",
  "packageJsonSha256",
  "formCode",
  "proposalId",
  "ordinal",
  "textDigest",
  "taxonomyVersion",
  "mainDomainCode",
  "topicCodes",
  "inspectionProfileCodes",
  "inspectionTypeCodes",
  "canonicalTargetKind",
  "targetProfileCode",
  "operationQualifiers",
  "activityQualifiers",
  "applicabilityDisposition",
  "evidenceExpectationCodes",
  "externalInvolvements",
  "agreementConfidence",
  "recommendationState",
  "rationaleCodes",
  "confidenceEvidence",
  "sourceRefs",
  "sourceMappingState",
  "sourceAuthorityState",
  "riskClassificationState",
  "decisionState",
  "extractionState",
  "questionSourceProposalGap",
  "externalApplicabilityUnresolved",
  "passDisagreementCodes",
  "passOneResultDigest",
  "passTwoResultDigest",
  "passOneRunId",
  "passTwoRunId",
  "promptDigest",
  "modelDescriptorDigests",
  "taxonomyDigest",
  "inputDigest",
  "itemSemanticDigest",
  "classificationRunDigest",
  "aggregateDigest",
];

const fixedInputs = {
  package: {
    bytes: 3370312,
    sha256: "5ebcce2d70ee22fef4165b490cb6e4b276ad776f40dbaf12e5cea85c9da91b15",
  },
  research: {
    bytes: 76750,
    sha256: "137592c739bc22f6be026f5bad94c5b200bb983132017d026b7e39634ab392c7",
  },
  workbook: {
    bytes: 12228,
    sha256: "e4d054f741b11ca9d848842a891d6f811f2e644aba29a7ffda970bfe6abb931e",
  },
};

function sha256(bytes) {
  return createHash("sha256").update(bytes).digest("hex");
}

function canonicalize(value) {
  if (Array.isArray(value)) return value.map(canonicalize);
  if (value && typeof value === "object") {
    return Object.fromEntries(
      Object.keys(value)
        .sort((left, right) => Buffer.from(left).compare(Buffer.from(right)))
        .map((key) => [key, canonicalize(value[key])]),
    );
  }
  return value;
}

function canonicalJSON(value) {
  return JSON.stringify(canonicalize(value));
}

function digestValue(domain, value) {
  return `sha256:${sha256(Buffer.from(`${domain}${canonicalJSON(value)}`, "utf8"))}`;
}

function withoutKey(value, key) {
  const copy = { ...value };
  delete copy[key];
  return copy;
}

function assertExactKeys(value, expected, code) {
  assert.ok(
    value &&
      typeof value === "object" &&
      !Array.isArray(value) &&
      canonicalJSON(Object.keys(value).sort()) === canonicalJSON([...expected].sort()),
    code,
  );
}

function assertSortedUniqueCodes(values, code) {
  assert.ok(Array.isArray(values), code);
  assert.deepEqual(values, [...new Set(values)].sort((left, right) =>
    Buffer.from(left).compare(Buffer.from(right))), code);
  assert.ok(values.every((value) => /^[A-Z0-9_]+$/u.test(value)), code);
}

function normalizeSetLikeByTuple(values, fields) {
  const keyed = values.map((value) => ({
    value,
    key: canonicalJSON(fields.map((field) =>
      Object.hasOwn(value, field) ? value[field] : "")),
  }));
  if (new Set(keyed.map((entry) => entry.key)).size !== keyed.length) {
    throw new Error("ERR_DUPLICATE_SEMANTIC_KEY");
  }
  return keyed
    .sort((left, right) => Buffer.from(left.key).compare(Buffer.from(right.key)))
    .map((entry) => entry.value);
}

function readJSON(path) {
  return JSON.parse(readFileSync(path, "utf8"));
}

function fileFact(path) {
  const bytes = readFileSync(path);
  return { bytes: bytes.byteLength, sha256: sha256(bytes) };
}

function expectedFormCodes() {
  const codes = [];
  for (let number = 1; number <= 34; number += 1) {
    codes.push(`FSS-AGA-FORM-${String(number).padStart(3, "0")}`);
  }
  codes.push("FSS-AGA-FORM-035A");
  for (let number = 36; number <= 48; number += 1) {
    codes.push(`FSS-AGA-FORM-${String(number).padStart(3, "0")}`);
  }
  for (let number = 50; number <= 53; number += 1) {
    codes.push(`FSS-AGA-FORM-${String(number).padStart(3, "0")}`);
  }
  return codes;
}

function fullIdentity(packageDocument, form, question) {
  return {
    packageVersion: packageDocument.packageVersion,
    packageJsonSha256: `sha256:${fixedInputs.package.sha256}`,
    formCode: form.formCode,
    proposalId: question.proposalId,
    ordinal: question.ordinal,
    textDigest: question.textDigest,
  };
}

function identityKey(identity) {
  return [
    identity.packageVersion,
    identity.packageJsonSha256,
    identity.formCode,
    identity.proposalId,
    identity.ordinal,
    identity.textDigest,
  ].join("\u0000");
}

function assertExactOrderedIdentityUnion(actual, expected) {
  assert.equal(
    digestValue("AGA-CLASSIFICATION-ORDERED-IDENTITIES-V1", actual) ===
      digestValue("AGA-CLASSIFICATION-ORDERED-IDENTITIES-V1", expected),
    true,
    "ERR_CLASSIFICATION_ORDERED_UNION_NOT_INDEPENDENTLY_PROVEN",
  );
}

function parseCSV(input) {
  const rows = [];
  let row = [];
  let cell = "";
  let quoted = false;
  for (let index = 0; index < input.length; index += 1) {
    const character = input[index];
    if (quoted) {
      if (character === '"' && input[index + 1] === '"') {
        cell += '"';
        index += 1;
      } else if (character === '"') {
        quoted = false;
      } else {
        cell += character;
      }
    } else if (character === '"') {
      quoted = true;
    } else if (character === ",") {
      row.push(cell);
      cell = "";
    } else if (character === "\n") {
      row.push(cell.replace(/\r$/u, ""));
      rows.push(row);
      row = [];
      cell = "";
    } else {
      cell += character;
    }
  }
  if (cell !== "" || row.length > 0) {
    row.push(cell);
    rows.push(row);
  }
  const headers = rows.shift();
  assert.ok(headers, "ERR_RESEARCH_HEADER_MISSING");
  return rows
    .filter((candidate) => candidate.length === headers.length)
    .map((candidate) =>
      Object.fromEntries(headers.map((header, index) => [header, candidate[index]])),
    );
}

function readZipCSV(entry) {
  const result = spawnSync("unzip", ["-p", researchZipPath, entry], {
    encoding: "utf8",
    maxBuffer: 4 * 1024 * 1024,
  });
  assert.equal(result.status, 0, "ERR_RESEARCH_ENTRY_UNAVAILABLE");
  return parseCSV(result.stdout);
}

function runPreparer(output, extraArguments = []) {
  return spawnSync(
    process.execPath,
    [
      preparerPath,
      "--package",
      packagePath,
      "--research-zip",
      researchZipPath,
      "--workbook",
      workbookPath,
      "--max-items",
      "64",
      "--max-input-bytes",
      "98304",
      "--output",
      output,
      ...extraArguments,
    ],
    { encoding: "utf8", maxBuffer: 4 * 1024 * 1024 },
  );
}

test("TestGate0BClassificationManifestOrderedUnionRejectsProducerRecomputedMutations", () => {
  const validator = classificationBatchPreparer.validateClassificationManifestOrderedUnion;
  assert.equal(typeof validator, "function", "ERR_CLASSIFICATION_ORDERED_UNION_VALIDATOR_MISSING");
  const packageDocument = readJSON(packagePath);
  const expectedIdentities = packageDocument.forms.flatMap((form) =>
    form.questions.map((question) => fullIdentity(packageDocument, form, question)),
  );
  const manifest = readJSON(
    join(discoveryDirectory, "classification-batch-manifest.json"),
  );
  assert.doesNotThrow(() => validator(manifest, expectedIdentities));
  assertExactOrderedIdentityUnion(
    manifest.batches.flatMap((batch) => batch.identities),
    expectedIdentities,
  );
  assert.equal(
    new Set(manifest.batches.flatMap((batch) => batch.identities.map(identityKey))).size,
    1310,
    "ERR_CLASSIFICATION_ORDERED_UNION_NOT_UNIQUE",
  );
  assert.ok(
    manifest.batches.every((batch, index) => batch.batchOrdinal === index + 1),
    "ERR_CLASSIFICATION_BATCH_ORDINAL_NOT_EXACT",
  );

  const recomputeSelfFields = (candidate) => {
    const batches = candidate.batches.map((batch) => {
      const withItemCount = {
        ...batch,
        itemCount: batch.identities.length,
      };
      const withOrderedIdentityDigest = {
        ...withItemCount,
        orderedIdentityDigest: digestValue(
          "AGA-CLASSIFICATION-ORDERED-IDENTITIES-V1",
          withItemCount.identities,
        ),
      };
      return {
        ...withOrderedIdentityDigest,
      batchEntryDigest: digestValue(
        "AGA-CLASSIFICATION-BATCH-ENTRY-V1",
          withoutKey(withOrderedIdentityDigest, "batchEntryDigest"),
      ),
      };
    });
    const identities = batches.flatMap((batch) => batch.identities);
    const payload = {
      ...candidate,
      batches,
      itemCount: identities.length,
      batchCount: batches.length,
      orderedIdentityDigest: digestValue(
        "AGA-CLASSIFICATION-ORDERED-IDENTITIES-V1",
        identities,
      ),
    };
    return {
      ...payload,
      manifestDigest: digestValue(
        "AGA-CLASSIFICATION-BATCH-MANIFEST-SET-V2",
        withoutKey(payload, "manifestDigest"),
      ),
    };
  };
  const reordered = recomputeSelfFields({
    ...manifest,
    batches: [
      { ...manifest.batches[1], batchOrdinal: 1 },
      { ...manifest.batches[0], batchOrdinal: 2 },
      ...manifest.batches.slice(2),
    ],
  });
  assert.throws(
    () => validator(reordered, expectedIdentities),
    (error) => error?.code === "CLASSIFICATION_MANIFEST_ORDERED_UNION_MISMATCH",
  );
  const duplicateAndOmission = recomputeSelfFields({
    ...manifest,
    batches: manifest.batches.map((batch, index) =>
      index === 1
        ? {
            ...batch,
            identities: [manifest.batches[0].identities[0], ...batch.identities.slice(1)],
          }
        : batch,
    ),
  });
  assert.throws(
    () => validator(duplicateAndOmission, expectedIdentities),
    (error) => error?.code === "CLASSIFICATION_MANIFEST_IDENTITY_DUPLICATE",
  );
  const ordinalDrift = recomputeSelfFields({
    ...manifest,
    batches: manifest.batches.map((batch, index) =>
      index === 1 ? { ...batch, batchOrdinal: 99 } : batch,
    ),
  });
  assert.throws(
    () => validator(ordinalDrift, expectedIdentities),
    (error) => error?.code === "CLASSIFICATION_MANIFEST_BATCH_ORDINAL_INVALID",
  );
  const pureOmission = recomputeSelfFields({
    ...manifest,
    batches: manifest.batches.map((batch, index) =>
      index === 1
        ? { ...batch, identities: batch.identities.slice(1) }
        : batch,
    ),
  });
  assert.throws(
    () => validator(pureOmission, expectedIdentities),
    (error) => error?.code === "CLASSIFICATION_MANIFEST_IDENTITY_COUNT_INVALID",
  );
});

test("TestGate0BOrderedUnionComparisonDiagnosticsNeverRenderIdentities", () => {
  const packageDocument = readJSON(packagePath);
  const sourceForm = packageDocument.forms.find((form) => form.questions.length > 0);
  const expectedIdentity = fullIdentity(
    packageDocument,
    sourceForm,
    sourceForm.questions[0],
  );
  const actualIdentity = {
    ...expectedIdentity,
    textDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  };
  let diagnostic = "";
  try {
    assertExactOrderedIdentityUnion([actualIdentity], [expectedIdentity]);
  } catch (error) {
    diagnostic = String(error);
  }
  const diagnosticTokens = (identity) => [
    identity.packageVersion,
    identity.packageJsonSha256,
    identity.formCode,
    identity.proposalId,
    `ordinal: ${identity.ordinal}`,
    identity.textDigest,
  ];
  const forbiddenIdentityValues = [
    ...diagnosticTokens(expectedIdentity),
    ...diagnosticTokens(actualIdentity),
  ];
  if (forbiddenIdentityValues.some((value) => diagnostic.includes(value))) {
    throw new Error("ERR_CLASSIFICATION_ORDERED_UNION_DIAGNOSTIC_LEAK");
  }
});

test("TestGate0BClassificationManifestBindsDiscoveryRootAndUsesNonAuthoritativeSizing", () => {
  const manifest = readJSON(
    join(discoveryDirectory, "classification-batch-manifest.json"),
  );
  const discoveryManifest = readJSON(join(discoveryDirectory, "batch-manifest.json"));
  assert.equal(
    manifest.discoveryBatchManifestDigest,
    discoveryManifest.manifestDigest,
    "ERR_CLASSIFICATION_DISCOVERY_ROOT_UNBOUND",
  );
  assert.ok(!Object.hasOwn(manifest, "sizingPins"), "ERR_CLASSIFICATION_AUTHORITATIVE_SIZING_PINS");
  assertExactKeys(
    manifest.sizingContract,
    [
      "taxonomyVersion",
      "canonicalJSON",
      "passInputDigestDomain",
      "fixedSha256StringBytes",
      "passRoles",
      "classificationRunId",
      "candidatePassRunId",
      "challengePassRunId",
    ],
    "ERR_CLASSIFICATION_SIZING_CONTRACT_OPEN",
  );
  const temporaryRoot = mkdtempSync(join(tmpdir(), "avia-aga-gate0b-cli."));
  try {
    const legacyOutput = join(temporaryRoot, "legacy.json");
    const classificationOutput = join(temporaryRoot, "classification.json");
    const legacy = runPreparer(legacyOutput);
    assert.equal(legacy.status, 0, "ERR_CLASSIFICATION_LEGACY_PREPARER_FAILED");
    assert.equal(legacy.stdout, "aga-hybrid-batches: ok batches=24 items=1310\n");
    const classification = runPreparer(classificationOutput, [
      "--classification-output",
      classificationOutput,
    ]);
    assert.equal(classification.status, 0, "ERR_CLASSIFICATION_PREPARER_FAILED");
    assert.equal(
      classification.stdout,
      "aga-hybrid-batches: ok discoveryBatches=24 classificationBatches=25 items=1310\n",
    );
  } finally {
    rmSync(temporaryRoot, { recursive: true, force: true });
  }
});

test("TestGate0BClassificationManifestSeparatesFullPrivateEnvelope", () => {
  const legacyFacts = {
    "batch-manifest.json": "aef10701332f833651ec28783cd2f2965cc618c796300d010a02e757880e72b2",
    "discovery-run.json": "593d18bbeee297339c302213c889581a7a3f958103a57bddc4ab5a91253bc7cf",
    "taxonomy-discovery.json": "229692632dfb1f58dcd1c0f60298c0c4535076bc0b8b9cb8d70eae36b5fc8453",
    "omission-review-inventory.json": "336654cdabbd9fde1ef88e4314bfb4e2474fe1f7a3480f65c9389ff05a420859",
  };
  for (const [name, expectedDigest] of Object.entries(legacyFacts)) {
    assert.equal(
      sha256(readFileSync(join(discoveryDirectory, name))),
      expectedDigest,
      "ERR_GATE0A_ARTIFACT_REWRITTEN",
    );
  }

  const classificationManifestPath = join(
    discoveryDirectory,
    "classification-batch-manifest.json",
  );
  assert.ok(existsSync(classificationManifestPath), "ERR_CLASSIFICATION_MANIFEST_MISSING");
  const manifest = readJSON(classificationManifestPath);
  const taxonomy = readJSON(taxonomyPath);
  const schema = readJSON(classificationSchemaPath);
  assert.ok(schema.$defs.classificationBatchManifest, "ERR_CLASSIFICATION_MANIFEST_SCHEMA_MISSING");
  assert.ok(schema.$defs.classificationBatchEntry, "ERR_CLASSIFICATION_BATCH_ENTRY_SCHEMA_MISSING");
  const manifestAjv = new Ajv2020({ allErrors: true, strict: false });
  manifestAjv.addSchema(schema);
  const validateManifest = manifestAjv.getSchema(
    `${schema.$id}#/$defs/classificationBatchManifest`,
  );
  assert.equal(validateManifest(manifest), true, "ERR_CLASSIFICATION_MANIFEST_SCHEMA_INVALID");
  assertExactKeys(
    manifest,
    [
      "schemaVersion",
      "itemCount",
      "batchCount",
      "maxItems",
      "maxCanonicalBytes",
      "sizingContract",
      "fixedInputDigests",
      "discoveryBatchManifestDigest",
      "prohibitedDiscoveryInputDigests",
      "orderedIdentityDigest",
      "batches",
      "manifestDigest",
    ],
    "ERR_CLASSIFICATION_MANIFEST_OPEN",
  );
  assert.equal(manifest.schemaVersion, "aga-hybrid-classification-batch-manifest/v2");
  assert.equal(manifest.itemCount, 1310);
  assert.equal(manifest.batchCount, 25);
  assert.equal(manifest.maxItems, 64);
  assert.equal(manifest.maxCanonicalBytes, 98304);
  assert.deepEqual(manifest.batches.map((batch) => batch.itemCount), [
    50, 49, 55, 54, 54, 53, 52, 54, 53, 52, 52, 57, 58, 58, 58, 61, 59, 53,
    51, 52, 56, 52, 56, 57, 4,
  ]);
  assert.equal(
    Math.max(...manifest.batches.map((batch) => batch.worstCaseCanonicalBytes)),
    98239,
  );
  assert.ok(manifest.batches.every((batch) => batch.itemCount <= manifest.maxItems));
  assert.ok(
    manifest.batches.every(
      (batch) => batch.worstCaseCanonicalBytes <= manifest.maxCanonicalBytes,
    ),
  );
  assertExactKeys(
    manifest.sizingContract,
    [
      "taxonomyVersion",
      "canonicalJSON",
      "passInputDigestDomain",
      "fixedSha256StringBytes",
      "passRoles",
      "classificationRunId",
      "candidatePassRunId",
      "challengePassRunId",
    ],
    "ERR_CLASSIFICATION_SIZING_CONTRACT_OPEN",
  );
  assert.equal(manifest.sizingContract.taxonomyVersion, taxonomy.taxonomyVersion);
  assert.equal(manifest.sizingContract.canonicalJSON, "AVIASURVEIL360_CANONICAL_JSON_V1");
  assert.equal(manifest.sizingContract.passInputDigestDomain, "AGA-CLASSIFICATION-PASS-INPUT-V1");
  assert.equal(manifest.sizingContract.fixedSha256StringBytes, 71);
  assert.deepEqual(manifest.sizingContract.passRoles, ["CANDIDATE", "CHALLENGE"]);
  assert.match(
    manifest.sizingContract.classificationRunId.maximumValidAsciiValue,
    /^aga-classification-run-a{64}$/u,
  );
  assert.match(
    manifest.sizingContract.candidatePassRunId.maximumValidAsciiValue,
    /^aga-classification-pass-candidate-a{64}$/u,
  );
  assert.match(
    manifest.sizingContract.challengePassRunId.maximumValidAsciiValue,
    /^aga-classification-pass-challenge-a{64}$/u,
  );
  for (const name of ["classificationRunId", "candidatePassRunId", "challengePassRunId"]) {
    assert.equal(
      Buffer.byteLength(manifest.sizingContract[name].maximumValidAsciiValue, "utf8"),
      manifest.sizingContract[name].maximumValidAsciiBytes,
    );
  }
  const sourceDigestNames = [
    "packageJsonSha256",
    "sealedOverlayLoaderZipSha256",
    "providerCatalogSha256",
    "researchZipSha256",
    "researchQuestionCsvSha256",
    "providerClassificationCsvSha256",
    "ambiguityCsvSha256",
    "workbookSha256",
    "auditChecklistWorkflowSha256",
    "findingCapEvidenceWorkflowSha256",
    "productionContractVocabularySha256",
  ];
  assert.deepEqual(Object.keys(manifest.fixedInputDigests), sourceDigestNames);
  assert.deepEqual(
    manifest.fixedInputDigests,
    Object.fromEntries(sourceDigestNames.map((name) => [name, taxonomy.fixedInputs[name]])),
  );
  for (const path of [classificationSpecPath, classificationPromptPath]) {
    const text = readFileSync(path, "utf8");
    assert.ok(text.includes("discoveryBatchManifestDigest"));
    assert.ok(text.includes("classificationBatchManifestDigest"));
  }
  assert.equal(manifest.discoveryBatchManifestDigest, taxonomy.fixedInputs.discoveryBatchManifestDigest);
  assert.deepEqual(
    taxonomy.digestContract.classificationBatchManifestContract,
    {
      schemaVersion: "aga-hybrid-classification-batch-manifest/v2",
      sourceSnapshotPreimageFields: ["batchOrdinal", "fixedInputDigests", "items"],
      sourceSnapshotExcludedFields: [],
      sourceSnapshotProhibits: [
        "taxonomyVersion",
        "taxonomyDigest",
        "promptDigest",
        "modelDescriptorDigest",
        "batchManifestDigest",
      ],
      orderedIdentityPreimage: "ORDERED_COMPLETE_BASE_IDENTITIES",
      batchEntryExcludedFields: ["batchEntryDigest"],
      manifestExcludedFields: ["manifestDigest"],
      discoveryBatchManifestDigest: "REQUIRED_EXACT_ACCEPTED_DISCOVERY_MANIFEST_ROOT_DIGEST",
      sizingContractFields: ["taxonomyVersion", "canonicalJSON", "passInputDigestDomain", "fixedSha256StringBytes", "passRoles", "classificationRunId", "candidatePassRunId", "challengePassRunId"],
      sizingContractAuthority: "NON_AUTHORITATIVE_BYTE_WIDTH_TEMPLATE_NO_DIGEST_OR_MODEL_METADATA_PIN",
      prohibitedDiscoveryInputDigestCount: 24,
      classificationBatchCount: 25,
    },
  );
  const discoveryManifest = readJSON(join(discoveryDirectory, "batch-manifest.json"));
  assert.deepEqual(
    manifest.prohibitedDiscoveryInputDigests,
    discoveryManifest.batches.map((batch) => batch.inputDigest),
  );
  assert.equal(new Set(manifest.prohibitedDiscoveryInputDigests).size, 24);

  const packageDocument = readJSON(packagePath);
  const researchByIdentity = new Map(
    readZipCSV("question_level_review.csv").map((row) => [
      [row.form_code, row.proposal_id, row.ordinal, row.text_digest].join("\u0000"),
      row,
    ]),
  );
  const privateItemsByIdentity = new Map();
  for (const form of packageDocument.forms) {
    for (const question of form.questions) {
      const identity = fullIdentity(packageDocument, form, question);
      const researchCandidateFacts = researchByIdentity.get(
        [form.formCode, question.proposalId, String(question.ordinal), question.textDigest].join(
          "\u0000",
        ),
      );
      assert.ok(researchCandidateFacts, "ERR_CLASSIFICATION_RESEARCH_IDENTITY_MISSING");
      privateItemsByIdentity.set(identityKey(identity), {
        identity,
        questionBody: question.originalText,
        packageFacts: {
          formKind: form.formKind,
          formRiskBand: form.proposedRisk?.band ?? null,
          questionRiskBand: question.proposedRisk?.band ?? null,
          questionRiskDomain: question.proposedRisk?.domain ?? null,
          sourceMappingState: question.sourceMappingState,
          sourceAuthorityState: question.sourceAuthorityState,
          extractionState: question.extractionState,
          riskClassificationState: question.riskClassificationState,
          decisionState: question.decisionState,
          sourceProposalDigests: (question.sourceProposals ?? []).map((proposal) =>
            digestValue("AGA-SOURCE-PROPOSAL-FACT-V1", proposal),
          ),
          sourceReferenceDigests: (question.sourceRefs ?? []).map((reference) =>
            digestValue("AGA-SOURCE-REFERENCE-FACT-V1", reference),
          ),
        },
        researchCandidateFacts,
      });
    }
  }
  assert.equal(privateItemsByIdentity.size, 1310);
  const passInputFields = [
    "schemaVersion",
    "purpose",
    "classificationRunId",
    "passRole",
    "passRunId",
    "batchOrdinal",
    "taxonomyVersion",
    "taxonomyDigest",
    "promptDigest",
    "modelDescriptorDigest",
    "batchManifestDigest",
    "fixedInputDigests",
    "items",
  ];
  assert.deepEqual(schema.$defs.classificationPassInput.required, passInputFields);
  assert.deepEqual(
    Object.keys(schema.$defs.classificationPassInput.properties),
    passInputFields,
    "ERR_CLASSIFICATION_SIZING_IGNORES_NEW_INPUT_FIELD",
  );
  const retainedTextForbidden = new Set([
    "questionBody",
    "originalText",
    "questionText",
    "body",
    "researchCandidateFacts",
    "packageFacts",
  ]);
  const visitPublic = (value) => {
    if (Array.isArray(value)) value.forEach(visitPublic);
    else if (value && typeof value === "object") {
      for (const [key, child] of Object.entries(value)) {
        assert.ok(!retainedTextForbidden.has(key), "ERR_CLASSIFICATION_MANIFEST_PRIVATE_LEAK");
        visitPublic(child);
      }
    }
  };
  visitPublic(manifest);
  const orderedIdentities = [];
  for (const batch of manifest.batches) {
    assertExactKeys(
      batch,
      [
        "batchOrdinal",
        "itemCount",
        "candidateCanonicalBytes",
        "challengeCanonicalBytes",
        "worstCaseCanonicalBytes",
        "sourceSnapshotDigest",
        "orderedIdentityDigest",
        "identities",
        "batchEntryDigest",
      ],
      "ERR_CLASSIFICATION_BATCH_ENTRY_OPEN",
    );
    const items = batch.identities.map((identity) => {
      const item = privateItemsByIdentity.get(identityKey(identity));
      assert.ok(item, "ERR_CLASSIFICATION_IDENTITY_UNKNOWN");
      return item;
    });
    orderedIdentities.push(...batch.identities);
    const sourceSnapshot = {
      batchOrdinal: batch.batchOrdinal,
      fixedInputDigests: manifest.fixedInputDigests,
      items,
    };
    assert.equal(
      batch.sourceSnapshotDigest,
      digestValue("AGA-CLASSIFICATION-SOURCE-SNAPSHOT-V1", sourceSnapshot),
      "ERR_CLASSIFICATION_SOURCE_SNAPSHOT_DIGEST",
    );
    assert.notEqual(
      batch.sourceSnapshotDigest,
      digestValue("AGA-CLASSIFICATION-SOURCE-SNAPSHOT-V1", {
        ...sourceSnapshot,
        fixedInputDigests: { ...sourceSnapshot.fixedInputDigests, packageJsonSha256: `sha256:${"0".repeat(64)}` },
      }),
      "ERR_CLASSIFICATION_SOURCE_SNAPSHOT_FIXED_INPUT_UNBOUND",
    );
    assert.notEqual(
      batch.sourceSnapshotDigest,
      digestValue("AGA-CLASSIFICATION-SOURCE-SNAPSHOT-V1", {
        ...sourceSnapshot,
        items: [
          {
            ...items[0],
            questionBody: `${items[0].questionBody} `,
          },
          ...items.slice(1),
        ],
      }),
      "ERR_CLASSIFICATION_PRIVATE_ITEM_UNBOUND",
    );
    assert.equal(
      batch.orderedIdentityDigest,
      digestValue("AGA-CLASSIFICATION-ORDERED-IDENTITIES-V1", batch.identities),
      "ERR_CLASSIFICATION_BATCH_IDENTITIES_DIGEST",
    );
    assert.notEqual(
      batch.orderedIdentityDigest,
      digestValue(
        "AGA-CLASSIFICATION-ORDERED-IDENTITIES-V1",
        [...batch.identities].reverse(),
      ),
      "ERR_CLASSIFICATION_IDENTITY_REORDER_UNBOUND",
    );
    for (const passRole of ["CANDIDATE", "CHALLENGE"]) {
      const input = {
        schemaVersion: "aga-hybrid-classification-pass-input/v1",
        purpose: "ROW_CLASSIFICATION_PRIVATE_INPUT",
        classificationRunId: manifest.sizingContract.classificationRunId.maximumValidAsciiValue,
        passRole,
        passRunId:
          passRole === "CANDIDATE"
            ? manifest.sizingContract.candidatePassRunId.maximumValidAsciiValue
            : manifest.sizingContract.challengePassRunId.maximumValidAsciiValue,
        batchOrdinal: batch.batchOrdinal,
        taxonomyVersion: manifest.sizingContract.taxonomyVersion,
        taxonomyDigest: `sha256:${"0".repeat(manifest.sizingContract.fixedSha256StringBytes - 7)}`,
        promptDigest: `sha256:${"0".repeat(manifest.sizingContract.fixedSha256StringBytes - 7)}`,
        modelDescriptorDigest: `sha256:${"0".repeat(manifest.sizingContract.fixedSha256StringBytes - 7)}`,
        batchManifestDigest: `sha256:${"0".repeat(manifest.sizingContract.fixedSha256StringBytes - 7)}`,
        fixedInputDigests: manifest.fixedInputDigests,
        items,
      };
      const bytes = Buffer.byteLength(canonicalJSON(input), "utf8");
      assert.equal(
        bytes,
        passRole === "CANDIDATE"
          ? batch.candidateCanonicalBytes
          : batch.challengeCanonicalBytes,
        "ERR_CLASSIFICATION_CANONICAL_SIZE_MISMATCH",
      );
      assert.ok(bytes <= manifest.maxCanonicalBytes, "ERR_CLASSIFICATION_SIZE_LIMIT");
    }
    assert.equal(
      batch.worstCaseCanonicalBytes,
      Math.max(batch.candidateCanonicalBytes, batch.challengeCanonicalBytes),
    );
    assert.equal(
      batch.batchEntryDigest,
      digestValue("AGA-CLASSIFICATION-BATCH-ENTRY-V1", withoutKey(batch, "batchEntryDigest")),
      "ERR_CLASSIFICATION_BATCH_ENTRY_DIGEST",
    );
    assert.ok(
      !manifest.prohibitedDiscoveryInputDigests.includes(batch.sourceSnapshotDigest),
      "ERR_DISCOVERY_DIGEST_REUSED_FOR_CLASSIFICATION",
    );
  }
  assert.equal(orderedIdentities.length, 1310);
  assert.equal(
    manifest.orderedIdentityDigest,
    digestValue("AGA-CLASSIFICATION-ORDERED-IDENTITIES-V1", orderedIdentities),
  );
  assert.equal(
    manifest.manifestDigest,
    digestValue("AGA-CLASSIFICATION-BATCH-MANIFEST-SET-V2", withoutKey(manifest, "manifestDigest")),
    "ERR_CLASSIFICATION_MANIFEST_ROOT_DIGEST",
  );

  const temporaryRoot = mkdtempSync(join(tmpdir(), "avia-aga-gate0b-rebuild."));
  try {
    const generatedLegacy = join(temporaryRoot, "batch-manifest.json");
    const generatedClassification = join(temporaryRoot, "classification-batch-manifest.json");
    const result = runPreparer(generatedLegacy, [
      "--discovery-run",
      join(discoveryDirectory, "discovery-run.json"),
      "--classification-output",
      generatedClassification,
    ]);
    assert.equal(result.status, 0, "ERR_CLASSIFICATION_REBUILD_FAILED");
    assert.equal(
      result.stdout,
      "aga-hybrid-batches: ok discoveryBatches=24 classificationBatches=25 items=1310\n",
    );
    assert.equal(readFileSync(generatedLegacy).compare(readFileSync(join(discoveryDirectory, "batch-manifest.json"))), 0);
    assert.equal(readFileSync(generatedClassification).compare(readFileSync(classificationManifestPath)), 0);
  } finally {
    rmSync(temporaryRoot, { recursive: true, force: true });
  }
});

test("TestGate0DiscoveryRequiresExactTextFreeCoverage", () => {
  assert.ok(existsSync(preparerPath), "ERR_GATE0_PREPARER_MISSING");
  assert.ok(existsSync(discoveryPromptPath), "ERR_DISCOVERY_PROMPT_MISSING");
  assert.deepEqual(fileFact(packagePath), fixedInputs.package);
  assert.deepEqual(fileFact(researchZipPath), fixedInputs.research);
  assert.deepEqual(fileFact(workbookPath), fixedInputs.workbook);

  const packageDocument = readJSON(packagePath);
  const formCodes = packageDocument.forms.map((form) => form.formCode);
  assert.deepEqual(formCodes, expectedFormCodes());
  assert.ok(formCodes.includes("FSS-AGA-FORM-035A"));
  assert.ok(!formCodes.includes("FSS-AGA-FORM-035"));
  assert.ok(!formCodes.includes("FSS-AGA-FORM-049"));

  const questions = packageDocument.forms.flatMap((form) =>
    form.questions.map((question) => ({
      identity: fullIdentity(packageDocument, form, question),
      question,
    })),
  );
  assert.equal(questions.length, 1310);
  assert.equal(new Set(questions.map(({ identity }) => identityKey(identity))).size, 1310);
  assert.ok(
    new Set(questions.map(({ question }) => question.textDigest)).size < questions.length,
    "ERR_DIGEST_NON_UNIQUENESS_NOT_PROVEN",
  );

  const researchRows = readZipCSV("question_level_review.csv");
  assert.equal(researchRows.length, 1310);
  const unresolved = researchRows.filter(
    (row) => row.provider_applicability_unresolved === "true",
  );
  assert.equal(unresolved.length, 51);
  const unresolvedKeys = new Set(
    unresolved.map((row) =>
      [row.form_code, row.proposal_id, row.ordinal, row.text_digest].join("\u0000"),
    ),
  );
  const sourceGaps = packageDocument.forms.flatMap((form) =>
    form.questions
      .filter((question) => question.sourceProposals.length === 0)
      .map((question) =>
        [form.formCode, question.proposalId, String(question.ordinal), question.textDigest].join(
          "\u0000",
        ),
      ),
  );
  assert.equal(sourceGaps.length, 49);
  assert.equal(sourceGaps.filter((key) => unresolvedKeys.has(key)).length, 49);

  const temporaryRoot = mkdtempSync(join(tmpdir(), "avia-aga-gate0-test."));
  try {
    const generatedPath = join(temporaryRoot, "batch-manifest.json");
    const result = runPreparer(generatedPath);
    assert.equal(result.status, 0, "ERR_GATE0_PREPARER_FAILED");
    assert.equal(result.stdout, "aga-hybrid-batches: ok batches=24 items=1310\n");
    assert.equal(result.stderr, "");
    const manifest = readJSON(generatedPath);
    assert.equal(manifest.schemaVersion, "aga-hybrid-classification-batch-manifest/v1");
    assert.equal(manifest.itemCount, 1310);
    assert.equal(manifest.maxItems, 64);
    assert.equal(manifest.maxInputBytes, 98304);
    assert.deepEqual(manifest.packageOrder, expectedFormCodes());
    assert.equal(manifest.batches.length, 24);
    assert.ok(manifest.batches.every((batch) => batch.itemCount <= 64));
    assert.ok(manifest.batches.every((batch) => batch.inputBytes <= 98304));
    assert.equal(
      digestValue(
        "AGA-HYBRID-TEST-ORDERED-IDENTITY-COMPARISON-V1",
        manifest.batches.flatMap((batch) => batch.identities),
      ),
      digestValue(
        "AGA-HYBRID-TEST-ORDERED-IDENTITY-COMPARISON-V1",
        questions.map(({ identity }) => identity),
      ),
      "ERR_GATE0_ORDERED_IDENTITY_MISMATCH",
    );
    const forbiddenKeys = new Set([
      "originalText",
      "questionText",
      "text",
      "body",
      "bodyFragment",
      "sourceLocator",
      "sourceTitle",
      "sourceUrl",
      "rawResponse",
    ]);
    const visit = (value) => {
      if (Array.isArray(value)) {
        value.forEach(visit);
      } else if (value && typeof value === "object") {
        for (const [key, child] of Object.entries(value)) {
          assert.ok(!forbiddenKeys.has(key), "ERR_TEXT_BEARING_MANIFEST_FIELD");
          visit(child);
        }
      }
    };
    visit(manifest);
  } finally {
    rmSync(temporaryRoot, { recursive: true, force: true });
  }

  for (const receipt of [
    "batch-manifest.json",
    "discovery-run.json",
    "taxonomy-discovery.json",
    "omission-review-inventory.json",
  ]) {
    assert.ok(existsSync(join(discoveryDirectory, receipt)), "ERR_GATE0_RECEIPT_MISSING");
  }
  const discovery = readJSON(join(discoveryDirectory, "discovery-run.json"));
  assert.equal(discovery.schemaVersion, "aga-hybrid-vocabulary-discovery-run/v1");
  assert.equal(discovery.status, "SEALED");
  assert.equal(discovery.itemCount, 1310);
  assert.equal(discovery.batchCount, 24);
  assert.equal(discovery.batches.length, 24);
  assert.equal(discovery.promptDigest, `sha256:${sha256(readFileSync(discoveryPromptPath))}`);
  assert.match(discovery.modelDescriptor?.modelId ?? "", /^[-a-zA-Z0-9._:]+$/u);
  assert.match(discovery.modelDescriptorDigest ?? "", /^sha256:[a-f0-9]{64}$/u);
  assert.ok(
    discovery.batches.every(
      (batch) =>
        /^sha256:[a-f0-9]{64}$/u.test(batch.inputDigest) &&
        /^sha256:[a-f0-9]{64}$/u.test(batch.outputDigest),
    ),
  );

  const omissionReview = readJSON(
    join(discoveryDirectory, "omission-review-inventory.json"),
  );
  assert.equal(omissionReview.identityCount, 1310);
  assert.equal(omissionReview.itemCount, omissionReview.items.length);
  assert.ok(omissionReview.itemCount > 0);
  assert.equal(
    new Set(omissionReview.signalRules.map((rule) => rule.signalRuleId)).size,
    omissionReview.signalRuleCount,
  );
  assert.ok(
    omissionReview.items.every(
      (item) =>
        Object.keys(item.identity ?? {}).sort().join(",") ===
          "formCode,ordinal,packageJsonSha256,packageVersion,proposalId,textDigest" &&
        typeof item.signalRuleId === "string" &&
        /^sha256:[a-f0-9]{64}$/u.test(item.inputFactValueDigest) &&
        !Object.hasOwn(item, "count"),
    ),
  );
});

test("TestGate0DiagnosticsAreIdentityRedacted", () => {
  assert.ok(existsSync(preparerPath), "ERR_GATE0_PREPARER_MISSING");
  const packageDocument = readJSON(packagePath);
  const questions = packageDocument.forms.flatMap((form) =>
    form.questions.map((question) => ({ form, question })),
  );
  assert.equal(questions.length, 1310, "ERR_TEST_INPUT_UNAVAILABLE");
  const temporaryRoot = mkdtempSync(join(tmpdir(), "avia-aga-gate0-probe."));
  try {
    const result = runPreparer(join(temporaryRoot, "unused.json"), [
      "--diagnostic-probe",
      "real-derived-identity-mismatch",
    ]);
    assert.notEqual(result.status, 0, "ERR_DIAGNOSTIC_PROBE_DID_NOT_FAIL");
    const combined = `${result.stdout}${result.stderr}`;
    assert.ok(
      /^ERR_IDENTITY_MISMATCH batch=1 count=1 digest=sha256:[a-f0-9]{64}\n$/u.test(
        combined,
      ),
      "ERR_DIAGNOSTIC_SHAPE",
    );
    const protectedValues = new Set([packagePath, researchZipPath, workbookPath]);
    const protect = (value) => {
      if (typeof value === "string" && value.length >= 8) protectedValues.add(value);
    };
    for (const { form, question } of questions) {
      protect(form.formCode);
      protect(question.proposalId);
      protect(question.textDigest);
      protect(question.originalText);
      protect(question.sourceLocator);
      for (const reference of question.sourceRefs ?? []) {
        for (const value of Object.values(reference)) protect(value);
      }
    }
    for (const sensitive of protectedValues) {
      if (sensitive) assert.ok(!combined.includes(String(sensitive)), "ERR_DIAGNOSTIC_LEAK");
    }
    for (const forbiddenLabel of [
      "formCode",
      "proposalId",
      "ordinal",
      "textDigest",
      "sourceRef",
      "identity",
      "path",
    ]) {
      assert.ok(!combined.includes(forbiddenLabel), "ERR_DIAGNOSTIC_LABEL_LEAK");
    }
  } finally {
    rmSync(temporaryRoot, { recursive: true, force: true });
  }
});

test("TestGate0DiscoveryReceiptsReconstructClosedDigestGraph", () => {
  const packageDocument = readJSON(packagePath);
  const questions = packageDocument.forms.flatMap((form) =>
    form.questions.map((question) => ({
      identity: fullIdentity(packageDocument, form, question),
      body: question.originalText,
    })),
  );
  const manifest = readJSON(join(discoveryDirectory, "batch-manifest.json"));
  const omissionReview = readJSON(
    join(discoveryDirectory, "omission-review-inventory.json"),
  );
  const discovery = readJSON(join(discoveryDirectory, "discovery-run.json"));
  const taxonomyDiscovery = readJSON(
    join(discoveryDirectory, "taxonomy-discovery.json"),
  );

  const temporaryRoot = mkdtempSync(join(tmpdir(), "avia-aga-gate0-rebuild."));
  try {
    const generatedManifestPath = join(temporaryRoot, "batch-manifest.json");
    const result = runPreparer(generatedManifestPath, [
      "--discovery-run",
      join(discoveryDirectory, "discovery-run.json"),
    ]);
    assert.equal(result.status, 0, "ERR_GATE0_REBUILD_FAILED");
    assert.ok(
      canonicalJSON(readJSON(generatedManifestPath)) === canonicalJSON(manifest),
      "ERR_GATE0_MANIFEST_NOT_REPRODUCIBLE",
    );
    assert.ok(
      canonicalJSON(readJSON(join(temporaryRoot, "omission-review-inventory.json"))) ===
        canonicalJSON(omissionReview),
      "ERR_GATE0_OMISSION_INVENTORY_NOT_REPRODUCIBLE",
    );
  } finally {
    rmSync(temporaryRoot, { recursive: true, force: true });
  }

  for (const batch of manifest.batches) {
    assert.equal(
      batch.batchDigest,
      digestValue("AGA-HYBRID-BATCH-MANIFEST-V1", withoutKey(batch, "batchDigest")),
      "ERR_GATE0_BATCH_DIGEST_MISMATCH",
    );
  }
  assert.equal(
    manifest.manifestDigest,
    digestValue(
      "AGA-HYBRID-BATCH-MANIFEST-SET-V1",
      withoutKey(manifest, "manifestDigest"),
    ),
    "ERR_GATE0_MANIFEST_DIGEST_MISMATCH",
  );

  assertExactKeys(
    discovery,
    [
      "schemaVersion",
      "status",
      "candidateOnly",
      "itemCount",
      "batchCount",
      "packageJsonSha256",
      "researchZipSha256",
      "workbookSha256",
      "promptDigest",
      "batchManifestDigest",
      "omissionInventoryDigest",
      "modelDescriptor",
      "modelDescriptorDigest",
      "contexts",
      "batches",
      "orderedOutputDigest",
      "orderedCandidateVocabularyDigest",
      "labels",
      "discoveryRunDigest",
    ],
    "ERR_GATE0_DISCOVERY_RUN_SCHEMA_OPEN",
  );
  assert.equal(discovery.batchManifestDigest, manifest.manifestDigest);
  assert.equal(discovery.omissionInventoryDigest, omissionReview.inventoryDigest);
  assert.equal(discovery.modelDescriptor.modelId, "gpt-5.6-sol");
  assert.equal(
    discovery.modelDescriptorDigest,
    digestValue("AGA-HYBRID-MODEL-DESCRIPTOR-V1", discovery.modelDescriptor),
    "ERR_GATE0_MODEL_DESCRIPTOR_DIGEST_MISMATCH",
  );
  assert.ok(
    discovery.contexts.every(
      (context) =>
        /^discovery-context-[0-9]{2}$/u.test(context.contextId) &&
        !context.contextId.includes("/") &&
        !context.contextId.includes("\\"),
    ),
    "ERR_GATE0_PRIVATE_CONTEXT_PATH",
  );

  const candidateVocabularyKeys = [
    "topicCues",
    "inspectionProfileCues",
    "inspectionTypeCues",
    "targetProfileCues",
    "operationQualifierCues",
    "activityQualifierCues",
    "evidenceExpectationCues",
    "applicabilityCues",
    "externalInvolvementConditionCues",
    "blockerCues",
    "rationaleCues",
  ];
  const identityDigests = manifest.batches.map((batch) =>
    batch.identities.map((identity) =>
      digestValue("AGA-HYBRID-DISCOVERY-IDENTITY-V1", identity),
    ),
  );
  assert.equal(discovery.batches.length, manifest.batches.length);
  for (let index = 0; index < discovery.batches.length; index += 1) {
    const batch = discovery.batches[index];
    const manifestBatch = manifest.batches[index];
    assertExactKeys(
      batch,
      [
        "batchOrdinal",
        "contextId",
        "itemCount",
        "inputBytes",
        "inputDigest",
        "output",
        "outputDigest",
        "candidateVocabularyDigest",
        "omissionSignalCount",
        "negativeVisibilityScan",
      ],
      "ERR_GATE0_BATCH_RECEIPT_SCHEMA_OPEN",
    );
    assert.equal(batch.batchOrdinal, manifestBatch.batchOrdinal);
    assert.equal(batch.itemCount, manifestBatch.itemCount);
    assert.equal(batch.inputBytes, manifestBatch.inputBytes);
    assert.equal(batch.inputDigest, manifestBatch.inputDigest);
    assert.equal(batch.negativeVisibilityScan, "PASS");
    assertExactKeys(
      batch.output,
      [
        "schemaVersion",
        "batchOrdinal",
        "coveredItemCount",
        "candidateVocabulary",
        "omissionSignals",
        "coverageDigestInput",
      ],
      "ERR_GATE0_MODEL_OUTPUT_SCHEMA_OPEN",
    );
    assert.equal(batch.output.schemaVersion, "aga-hybrid-vocabulary-discovery-output/v1");
    assert.equal(batch.output.batchOrdinal, batch.batchOrdinal);
    assert.equal(batch.output.coveredItemCount, batch.itemCount);
    assert.deepEqual(batch.output.coverageDigestInput, identityDigests[index]);
    assertExactKeys(
      batch.output.candidateVocabulary,
      candidateVocabularyKeys,
      "ERR_GATE0_CANDIDATE_VOCABULARY_SCHEMA_OPEN",
    );
    for (const key of candidateVocabularyKeys) {
      assertSortedUniqueCodes(
        batch.output.candidateVocabulary[key],
        "ERR_GATE0_CANDIDATE_VOCABULARY_NOT_CLOSED",
      );
    }
    assert.ok(Array.isArray(batch.output.omissionSignals));
    for (const signal of batch.output.omissionSignals) {
      assertExactKeys(
        signal,
        ["signalRuleId", "identityDigest", "cueCode", "inputFactSelector"],
        "ERR_GATE0_OMISSION_SIGNAL_SCHEMA_OPEN",
      );
      assert.match(signal.signalRuleId, /^[A-Z0-9_]+$/u);
      assert.match(signal.cueCode, /^[A-Z0-9_]+$/u);
      assert.ok(identityDigests[index].includes(signal.identityDigest));
      assert.ok(
        [
          "QUESTION_BODY_DIGEST",
          "FORM_METADATA_DIGEST",
          "SOURCE_PROPOSAL_DIGEST",
          "RESEARCH_ROW_DIGEST",
        ].includes(signal.inputFactSelector),
      );
    }
    assert.equal(batch.omissionSignalCount, batch.output.omissionSignals.length);
    assert.equal(
      batch.outputDigest,
      digestValue("AGA-HYBRID-VOCABULARY-OUTPUT-V1", batch.output),
      "ERR_GATE0_MODEL_OUTPUT_DIGEST_MISMATCH",
    );
    assert.equal(
      batch.candidateVocabularyDigest,
      digestValue(
        "AGA-HYBRID-CANDIDATE-VOCABULARY-V1",
        batch.output.candidateVocabulary,
      ),
      "ERR_GATE0_CANDIDATE_VOCABULARY_DIGEST_MISMATCH",
    );
  }
  assert.equal(
    discovery.orderedOutputDigest,
    digestValue(
      "AGA-HYBRID-ORDERED-OUTPUT-DIGESTS-V1",
      discovery.batches.map((batch) => batch.outputDigest),
    ),
    "ERR_GATE0_ORDERED_OUTPUT_DIGEST_MISMATCH",
  );
  assert.equal(
    discovery.orderedCandidateVocabularyDigest,
    digestValue(
      "AGA-HYBRID-ORDERED-CANDIDATE-VOCABULARY-DIGESTS-V1",
      discovery.batches.map((batch) => batch.candidateVocabularyDigest),
    ),
    "ERR_GATE0_ORDERED_CANDIDATE_DIGEST_MISMATCH",
  );
  assert.equal(
    discovery.discoveryRunDigest,
    digestValue(
      "AGA-HYBRID-VOCABULARY-DISCOVERY-RUN-V1",
      withoutKey(discovery, "discoveryRunDigest"),
    ),
    "ERR_GATE0_DISCOVERY_RUN_DIGEST_MISMATCH",
  );

  assert.equal(taxonomyDiscovery.promptDigest, discovery.promptDigest);
  assert.equal(taxonomyDiscovery.batchManifestDigest, manifest.manifestDigest);
  assert.equal(taxonomyDiscovery.modelDescriptorDigest, discovery.modelDescriptorDigest);
  assert.equal(
    taxonomyDiscovery.orderedCandidateVocabularyDigest,
    discovery.orderedCandidateVocabularyDigest,
  );
  assert.deepEqual(
    taxonomyDiscovery.normalizedSignalRuleIds,
    omissionReview.signalRules.map((rule) => rule.signalRuleId).sort(),
  );
  assert.deepEqual(taxonomyDiscovery.prohibitedSemantics, [
    "OWNER",
    "OWNERSHIP",
    "PRIMARY_PROVIDER",
    "SECONDARY_PROVIDER",
    "SUPPORTING_PROVIDER",
  ]);
  assert.equal(
    taxonomyDiscovery.taxonomyDiscoveryDigest,
    digestValue(
      "AGA-HYBRID-TAXONOMY-DISCOVERY-V1",
      withoutKey(taxonomyDiscovery, "taxonomyDiscoveryDigest"),
    ),
    "ERR_GATE0_TAXONOMY_DISCOVERY_DIGEST_MISMATCH",
  );

  const bodies = new Set(questions.map(({ body }) => body).filter(Boolean));
  const forbiddenKeys = /^(?:originalText|questionText|text|body|bodyFragment|sourceLocator|sourceRefs?|sourceTitle|sourceUrl|rawResponse|startedAt|endedAt|owner|ownership|primaryProvider|secondaryProvider|supportingProvider)$/iu;
  const forbiddenHierarchyValues = new Set(taxonomyDiscovery.prohibitedSemantics);
  const visit = (value, path = []) => {
    if (Array.isArray(value)) {
      value.forEach((child, index) => visit(child, [...path, String(index)]));
      return;
    }
    if (value && typeof value === "object") {
      for (const [key, child] of Object.entries(value)) {
        assert.ok(!forbiddenKeys.test(key), "ERR_GATE0_FORBIDDEN_RECEIPT_FIELD");
        visit(child, [...path, key]);
      }
      return;
    }
    if (typeof value !== "string") return;
    assert.ok(!bodies.has(value), "ERR_GATE0_QUESTION_BODY_LEAK");
    assert.ok(!/^\/(?:Users|private|tmp)\//u.test(value), "ERR_GATE0_PRIVATE_PATH_LEAK");
    const isProhibitionDeclaration = path.at(-2) === "prohibitedSemantics";
    assert.ok(
      isProhibitionDeclaration || !forbiddenHierarchyValues.has(value),
      "ERR_GATE0_PROVIDER_HIERARCHY_VALUE",
    );
  };
  visit(manifest);
  visit(omissionReview);
  visit(discovery);
  visit(taxonomyDiscovery);
});

test("TestGate0OmissionRulesAreFrozenAndReproducible", () => {
  const omissionReview = readJSON(
    join(discoveryDirectory, "omission-review-inventory.json"),
  );
  assertExactKeys(
    omissionReview,
    [
      "schemaVersion",
      "identityCount",
      "signalRuleCount",
      "signalRules",
      "signalRuleDigest",
      "modelSignalMappingCount",
      "modelSignalMappings",
      "modelSignalMappingDigest",
      "modelSignalCount",
      "modelSignalDispositionCounts",
      "modelSignalDispositions",
      "modelSignalDispositionDigest",
      "itemCount",
      "items",
      "inventoryDigest",
    ],
    "ERR_GATE0_OMISSION_INVENTORY_SCHEMA_OPEN",
  );
  assert.equal(omissionReview.signalRules.length, omissionReview.signalRuleCount);
  assert.equal(
    omissionReview.signalRuleDigest,
    digestValue("AGA-HYBRID-OMISSION-SIGNAL-RULES-V1", omissionReview.signalRules),
    "ERR_GATE0_OMISSION_SIGNAL_RULE_DIGEST_MISMATCH",
  );
  assert.equal(
    omissionReview.inventoryDigest,
    digestValue(
      "AGA-HYBRID-OMISSION-INVENTORY-V1",
      withoutKey(omissionReview, "inventoryDigest"),
    ),
    "ERR_GATE0_OMISSION_INVENTORY_DIGEST_MISMATCH",
  );
  for (const rule of omissionReview.signalRules) {
    assertExactKeys(
      rule,
      ["signalRuleId", "cueCode", "inputFactSelector", "matcher"],
      "ERR_GATE0_OMISSION_RULE_SCHEMA_OPEN",
    );
    assertExactKeys(
      rule.matcher,
      rule.matcher.kind === "QUESTION_BODY_REGEX"
        ? ["kind", "pattern", "flags"]
        : rule.matcher.kind === "RESEARCH_FIELD_EQUALS"
          ? ["kind", "field", "value"]
          : ["kind"],
      "ERR_GATE0_OMISSION_MATCHER_SCHEMA_OPEN",
    );
    if (rule.matcher.kind === "QUESTION_BODY_REGEX") {
      assert.doesNotThrow(() => new RegExp(rule.matcher.pattern, rule.matcher.flags));
    } else {
      assert.ok(
        ["RESEARCH_FIELD_EQUALS", "SOURCE_PROPOSALS_EMPTY"].includes(rule.matcher.kind),
        "ERR_GATE0_OMISSION_MATCHER_UNKNOWN",
      );
    }
  }

  const researchByIdentity = new Map(
    readZipCSV("question_level_review.csv").map((row) => [
      [row.form_code, row.proposal_id, row.ordinal, row.text_digest].join("\u0000"),
      row,
    ]),
  );
  const packageDocument = readJSON(packagePath);
  const expectedItems = [];
  for (const form of packageDocument.forms) {
    for (const question of form.questions) {
      const identity = fullIdentity(packageDocument, form, question);
      const researchRow = researchByIdentity.get(
        [form.formCode, question.proposalId, String(question.ordinal), question.textDigest].join(
          "\u0000",
        ),
      );
      assert.ok(researchRow, "ERR_GATE0_OMISSION_RESEARCH_IDENTITY_MISSING");
      for (const rule of omissionReview.signalRules) {
        let matches = false;
        if (rule.matcher.kind === "SOURCE_PROPOSALS_EMPTY") {
          matches = (question.sourceProposals ?? []).length === 0;
        } else if (rule.matcher.kind === "RESEARCH_FIELD_EQUALS") {
          matches = researchRow[rule.matcher.field] === rule.matcher.value;
        } else {
          matches = new RegExp(rule.matcher.pattern, rule.matcher.flags).test(
            question.originalText,
          );
        }
        if (!matches) continue;
        let inputFactValueDigest;
        if (rule.inputFactSelector === "QUESTION_BODY_DIGEST") {
          inputFactValueDigest = `sha256:${sha256(Buffer.from(question.originalText, "utf8"))}`;
        } else if (rule.inputFactSelector === "SOURCE_PROPOSAL_DIGEST") {
          inputFactValueDigest = digestValue(
            "AGA-SOURCE-PROPOSAL-SET-V1",
            (question.sourceProposals ?? []).map((proposal) =>
              digestValue("AGA-SOURCE-PROPOSAL-FACT-V1", proposal),
            ),
          );
        } else {
          inputFactValueDigest = digestValue("AGA-RESEARCH-ROW-FACT-V1", researchRow);
        }
        const itemPayload = {
          identity,
          identityDigest: digestValue("AGA-HYBRID-DISCOVERY-IDENTITY-V1", identity),
          signalRuleId: rule.signalRuleId,
          cueCode: rule.cueCode,
          inputFactSelector: rule.inputFactSelector,
          inputFactValueDigest,
        };
        expectedItems.push({
          ...itemPayload,
          itemDigest: digestValue(
            "AGA-HYBRID-OMISSION-INVENTORY-ITEM-V1",
            itemPayload,
          ),
        });
      }
    }
  }
  assert.equal(expectedItems.length, omissionReview.itemCount);
  assert.equal(
    digestValue("AGA-HYBRID-INDEPENDENT-OMISSION-ITEMS-V1", expectedItems),
    digestValue("AGA-HYBRID-INDEPENDENT-OMISSION-ITEMS-V1", omissionReview.items),
    "ERR_GATE0_OMISSION_ITEMS_NOT_INDEPENDENTLY_RECONSTRUCTIBLE",
  );
});

test("TestGate0ModelOmissionSignalsAreExhaustivelyDispositioned", () => {
  const manifest = readJSON(join(discoveryDirectory, "batch-manifest.json"));
  const discovery = readJSON(join(discoveryDirectory, "discovery-run.json"));
  const omissionReview = readJSON(
    join(discoveryDirectory, "omission-review-inventory.json"),
  );
  const rawSignals = discovery.batches.flatMap((batch) =>
    batch.output.omissionSignals.map((signal) => ({
      batchOrdinal: batch.batchOrdinal,
      signal,
      rawSignalDigest: digestValue("AGA-HYBRID-MODEL-OMISSION-SIGNAL-V1", {
        batchOrdinal: batch.batchOrdinal,
        signal,
      }),
    })),
  );
  assert.equal(rawSignals.length, 157);
  assert.equal(new Set(rawSignals.map((entry) => entry.rawSignalDigest)).size, 157);

  for (const required of [
    "modelSignalMappingCount",
    "modelSignalMappings",
    "modelSignalMappingDigest",
    "modelSignalCount",
    "modelSignalDispositionCounts",
    "modelSignalDispositions",
    "modelSignalDispositionDigest",
  ]) {
    assert.ok(Object.hasOwn(omissionReview, required), "ERR_GATE0_MODEL_SIGNAL_LEDGER_MISSING");
  }
  assert.equal(
    omissionReview.modelSignalMappingCount,
    omissionReview.modelSignalMappings.length,
  );
  for (const mapping of omissionReview.modelSignalMappings) {
    assertExactKeys(
      mapping,
      [
        "candidateSignalRuleId",
        "candidateCueCode",
        "inputFactSelector",
        "frozenSignalRuleId",
      ],
      "ERR_GATE0_MODEL_SIGNAL_MAPPING_SCHEMA_OPEN",
    );
  }
  assert.equal(
    omissionReview.modelSignalMappingDigest,
    digestValue(
      "AGA-HYBRID-MODEL-SIGNAL-MAPPINGS-V1",
      omissionReview.modelSignalMappings,
    ),
    "ERR_GATE0_MODEL_SIGNAL_MAPPING_DIGEST_MISMATCH",
  );
  assert.equal(omissionReview.modelSignalCount, rawSignals.length);
  assert.equal(omissionReview.modelSignalDispositions.length, rawSignals.length);
  assertExactKeys(
    omissionReview.modelSignalDispositionCounts,
    ["accepted", "rejected"],
    "ERR_GATE0_MODEL_SIGNAL_DISPOSITION_COUNTS_SCHEMA_OPEN",
  );
  assert.equal(
    omissionReview.modelSignalDispositionDigest,
    digestValue(
      "AGA-HYBRID-MODEL-SIGNAL-DISPOSITIONS-V1",
      omissionReview.modelSignalDispositions,
    ),
    "ERR_GATE0_MODEL_SIGNAL_DISPOSITION_DIGEST_MISMATCH",
  );
  assert.equal(
    omissionReview.modelSignalDispositionCounts.accepted +
      omissionReview.modelSignalDispositionCounts.rejected,
    rawSignals.length,
  );

  const dispositionByDigest = new Map(
    omissionReview.modelSignalDispositions.map((disposition) => [
      disposition.rawSignalDigest,
      disposition,
    ]),
  );
  assert.equal(dispositionByDigest.size, rawSignals.length);
  const mappingByCandidate = new Map(
    omissionReview.modelSignalMappings.map((mapping) => [
      [
        mapping.candidateSignalRuleId,
        mapping.candidateCueCode,
        mapping.inputFactSelector,
      ].join("\u0000"),
      mapping,
    ]),
  );
  assert.equal(mappingByCandidate.size, omissionReview.modelSignalMappings.length);
  assert.equal(
    new Set(
      rawSignals.map(({ signal }) =>
        [signal.signalRuleId, signal.cueCode, signal.inputFactSelector].join("\u0000"),
      ),
    ).size,
    mappingByCandidate.size,
    "ERR_GATE0_MODEL_SIGNAL_MAPPING_ORPHAN",
  );
  const ruleById = new Map(
    omissionReview.signalRules.map((rule) => [rule.signalRuleId, rule]),
  );
  const itemByDigest = new Map(
    omissionReview.items.map((item) => [item.itemDigest, item]),
  );
  assert.equal(itemByDigest.size, omissionReview.items.length);

  const packageDocument = readJSON(packagePath);
  const researchByIdentity = new Map(
    readZipCSV("question_level_review.csv").map((row) => [
      [row.form_code, row.proposal_id, row.ordinal, row.text_digest].join("\u0000"),
      row,
    ]),
  );
  const privateFactsByIdentityDigest = new Map();
  for (const form of packageDocument.forms) {
    for (const question of form.questions) {
      const identity = fullIdentity(packageDocument, form, question);
      const identityDigest = digestValue("AGA-HYBRID-DISCOVERY-IDENTITY-V1", identity);
      const researchRow = researchByIdentity.get(
        [form.formCode, question.proposalId, String(question.ordinal), question.textDigest].join(
          "\u0000",
        ),
      );
      assert.ok(researchRow, "ERR_GATE0_MODEL_SIGNAL_RESEARCH_IDENTITY_MISSING");
      privateFactsByIdentityDigest.set(identityDigest, {
        identity,
        form,
        question,
        researchRow,
      });
    }
  }
  assert.equal(privateFactsByIdentityDigest.size, 1310);

  const factDigest = (selector, facts) => {
    if (selector === "QUESTION_BODY_DIGEST") {
      return `sha256:${sha256(Buffer.from(facts.question.originalText, "utf8"))}`;
    }
    if (selector === "FORM_METADATA_DIGEST") {
      return digestValue("AGA-FORM-METADATA-FACT-V1", {
        formKind: facts.form.formKind,
        formRiskBand: facts.form.proposedRisk?.band ?? null,
      });
    }
    if (selector === "SOURCE_PROPOSAL_DIGEST") {
      return digestValue(
        "AGA-SOURCE-PROPOSAL-SET-V1",
        (facts.question.sourceProposals ?? []).map((proposal) =>
          digestValue("AGA-SOURCE-PROPOSAL-FACT-V1", proposal),
        ),
      );
    }
    assert.equal(selector, "RESEARCH_ROW_DIGEST", "ERR_GATE0_MODEL_SIGNAL_SELECTOR_UNKNOWN");
    return digestValue("AGA-RESEARCH-ROW-FACT-V1", facts.researchRow);
  };
  const ruleMatches = (rule, facts) => {
    if (rule.matcher.kind === "SOURCE_PROPOSALS_EMPTY") {
      return (facts.question.sourceProposals ?? []).length === 0;
    }
    if (rule.matcher.kind === "RESEARCH_FIELD_EQUALS") {
      return facts.researchRow[rule.matcher.field] === rule.matcher.value;
    }
    return new RegExp(rule.matcher.pattern, rule.matcher.flags).test(
      facts.question.originalText,
    );
  };

  for (const raw of rawSignals) {
    const disposition = dispositionByDigest.get(raw.rawSignalDigest);
    assert.ok(disposition, "ERR_GATE0_MODEL_SIGNAL_ORPHAN");
    assertExactKeys(
      disposition,
      [
        "batchOrdinal",
        "rawSignalDigest",
        "identity",
        "identityDigest",
        "candidateSignalRuleId",
        "candidateCueCode",
        "inputFactSelector",
        "inputFactValueDigest",
        "disposition",
        "frozenSignalRuleId",
        "inventoryItemDigest",
        "rejectionCode",
      ],
      "ERR_GATE0_MODEL_SIGNAL_DISPOSITION_SCHEMA_OPEN",
    );
    assert.equal(disposition.batchOrdinal, raw.batchOrdinal);
    assert.equal(disposition.identityDigest, raw.signal.identityDigest);
    assert.equal(disposition.candidateSignalRuleId, raw.signal.signalRuleId);
    assert.equal(disposition.candidateCueCode, raw.signal.cueCode);
    assert.equal(disposition.inputFactSelector, raw.signal.inputFactSelector);
    const facts = privateFactsByIdentityDigest.get(disposition.identityDigest);
    assert.ok(facts, "ERR_GATE0_MODEL_SIGNAL_IDENTITY_UNRESOLVED");
    assert.equal(
      digestValue("AGA-HYBRID-DISCOVERY-IDENTITY-V1", disposition.identity),
      disposition.identityDigest,
    );
    assert.equal(
      digestValue("AGA-HYBRID-TEST-IDENTITY-RESOLUTION-V1", disposition.identity),
      digestValue("AGA-HYBRID-TEST-IDENTITY-RESOLUTION-V1", facts.identity),
      "ERR_GATE0_MODEL_SIGNAL_IDENTITY_MISMATCH",
    );
    assert.equal(disposition.inputFactValueDigest, factDigest(disposition.inputFactSelector, facts));
    const mapping = mappingByCandidate.get(
      [
        disposition.candidateSignalRuleId,
        disposition.candidateCueCode,
        disposition.inputFactSelector,
      ].join("\u0000"),
    );
    assert.ok(mapping, "ERR_GATE0_MODEL_SIGNAL_MAPPING_MISSING");
    assert.equal(mapping.frozenSignalRuleId, disposition.frozenSignalRuleId);
    const frozenRule = ruleById.get(disposition.frozenSignalRuleId);
    assert.ok(frozenRule, "ERR_GATE0_MODEL_SIGNAL_FROZEN_RULE_UNKNOWN");
    if (disposition.disposition === "ACCEPTED_FROZEN_RULE") {
      assert.equal(disposition.rejectionCode, null);
      assert.equal(frozenRule.inputFactSelector, disposition.inputFactSelector);
      assert.ok(ruleMatches(frozenRule, facts), "ERR_GATE0_MODEL_SIGNAL_RULE_NOT_MATCHED");
      const item = itemByDigest.get(disposition.inventoryItemDigest);
      assert.ok(item, "ERR_GATE0_MODEL_SIGNAL_INVENTORY_ITEM_MISSING");
      assert.equal(item.identityDigest, disposition.identityDigest);
      assert.equal(item.signalRuleId, disposition.frozenSignalRuleId);
      assert.equal(item.inputFactValueDigest, disposition.inputFactValueDigest);
    } else {
      assert.equal(disposition.disposition, "REJECTED_CANDIDATE_SIGNAL");
      assert.equal(disposition.inventoryItemDigest, null);
      assert.ok(
        ["INPUT_SELECTOR_MISMATCH", "FROZEN_RULE_NOT_MATCHED"].includes(
          disposition.rejectionCode,
        ),
        "ERR_GATE0_MODEL_SIGNAL_REJECTION_UNKNOWN",
      );
      if (disposition.rejectionCode === "INPUT_SELECTOR_MISMATCH") {
        assert.notEqual(frozenRule.inputFactSelector, disposition.inputFactSelector);
      } else {
        assert.equal(frozenRule.inputFactSelector, disposition.inputFactSelector);
        assert.ok(!ruleMatches(frozenRule, facts));
      }
    }
  }

  assert.equal(
    new Set(
      manifest.batches
        .flatMap((batch) => batch.identities)
        .map((identity) => digestValue("AGA-HYBRID-DISCOVERY-IDENTITY-V1", identity)),
    ).size,
    1310,
  );
});

test("TestGate0DiagnosticAssertionsCannotEchoCapturedStreams", () => {
  const source = readFileSync(import.meta.filename, "utf8");
  assert.ok(
    !/assert\.match\(\s*combined\s*,/u.test(source),
    "ERR_GATE0_DIAGNOSTIC_ASSERTION_CAN_ECHO_CAPTURED_STREAM",
  );
  assert.ok(
    !/assert\.(?:equal|deepEqual|strictEqual)\(\s*combined\s*,/u.test(source),
    "ERR_GATE0_DIAGNOSTIC_ASSERTION_CAN_ECHO_CAPTURED_STREAM",
  );
});

test("TestGate0PreparerIsImportSafe", () => {
  const temporaryRoot = mkdtempSync(join(tmpdir(), "aga-gate0-import-"));
  try {
    const importerPath = join(temporaryRoot, "import-preparer.mjs");
    writeFileSync(
      importerPath,
      `await import(${JSON.stringify(`file://${resolve(preparerPath)}`)});\nprocess.stdout.write("IMPORT_OK\\n");\n`,
      { encoding: "utf8", mode: 0o600 },
    );
    const result = spawnSync(process.execPath, [importerPath], {
      cwd: process.cwd(),
      encoding: "utf8",
    });
    assert.equal(result.status === 0, true, "ERR_GATE0_PREPARER_IMPORT_EXECUTED_CLI");
    assert.equal(result.stdout === "IMPORT_OK\n", true, "ERR_GATE0_PREPARER_IMPORT_OUTPUT");
    assert.equal(result.stderr === "", true, "ERR_GATE0_PREPARER_IMPORT_ERROR");
  } finally {
    rmSync(temporaryRoot, { recursive: true, force: true });
  }
});

test("TestGate0BFreezesExactTaxonomyAndProviderPartition", () => {
  for (const path of [
    classificationSpecPath,
    taxonomyPath,
    classificationSchemaPath,
    classificationPromptPath,
  ]) {
    assert.ok(existsSync(path), "ERR_GATE0B_FREEZE_ARTIFACT_MISSING");
  }
  const taxonomy = readJSON(taxonomyPath);
  assertExactKeys(
    taxonomy,
    [
      "schemaVersion",
      "taxonomyVersion",
      "status",
      "fixedInputs",
      "mainDomainCodes",
      "topicCodes",
      "inspectionProfileCodes",
      "inspectionProfileDefinitions",
      "inspectionTypeCodes",
      "canonicalTargetKinds",
      "targetProfileCodes",
      "targetCompatibility",
      "canonicalTargetKindEligibility",
      "operationQualifierDefinitions",
      "activityQualifierDefinitions",
      "evidenceExpectationCodes",
      "applicabilityDispositions",
      "externalInvolvementRoles",
      "externalInvolvementConditionCodes",
      "blockerCodes",
      "rationaleCodes",
      "inputFactSelectors",
      "signalRuleIds",
      "proposalFields",
      "sourceReferenceKinds",
      "disagreementCodes",
      "providerPartition",
      "recommendationStates",
      "agreementConfidenceStates",
      "passRoles",
      "classificationRunStates",
      "draftStates",
      "draftItemOrigins",
      "draftRecommendationStates",
      "draftReviewStates",
      "draftDispositions",
      "managerActions",
      "workspaceGenerationStates",
      "reasonCodes",
      "lifecycleCodes",
      "evidenceBindingContract",
      "confidenceContract",
      "recommendationPrecedence",
      "fatalRunErrorCodes",
      "proposalResolutionModes",
      "draftTransitionContract",
      "questionReferenceContract",
      "commandContract",
      "aggregateContract",
      "normalization",
      "digestContract",
      "forbiddenProviderHierarchyFields",
      "taxonomyDigest",
    ],
    "ERR_GATE0B_TAXONOMY_SCHEMA_OPEN",
  );
  assert.equal(taxonomy.schemaVersion, "aga-question-classification-taxonomy/v1");
  assert.equal(taxonomy.taxonomyVersion, "AGA_QUESTION_CLASSIFICATION_V1");
  assert.equal(taxonomy.status, "FROZEN");
  assert.deepEqual(taxonomy.fixedInputs, {
    packageJsonSha256:
      "sha256:5ebcce2d70ee22fef4165b490cb6e4b276ad776f40dbaf12e5cea85c9da91b15",
    sealedOverlayLoaderZipSha256:
      "sha256:30700a88aeb5b26514bf7eb76bef050deb08b96294db94117d185de5c9f163b2",
    providerCatalogSha256:
      "sha256:42079b4046542e392c393fe6de1052d84f96938ea163cf63deed5ae9c4b6a789",
    researchZipSha256:
      "sha256:137592c739bc22f6be026f5bad94c5b200bb983132017d026b7e39634ab392c7",
    researchQuestionCsvSha256:
      "sha256:e39685d467c9c66220b20e998deab366a138148f4d532db7fac07e58e64e7a7c",
    providerClassificationCsvSha256:
      "sha256:d52a98739db61828c16aa734154be18b11e6ebb358eeeb7f84c3d92a4a5430de",
    ambiguityCsvSha256:
      "sha256:6e97a193f5e12dbe81f87d44d4b22c36ce446a40be7ef0f9fc939e8fbf1e654d",
    workbookSha256:
      "sha256:e4d054f741b11ca9d848842a891d6f811f2e644aba29a7ffda970bfe6abb931e",
    auditChecklistWorkflowSha256:
      "sha256:7dee737c7c5e47e996857e956514a8d46d1a4444234b021cac77cd6cff6b30a2",
    findingCapEvidenceWorkflowSha256:
      "sha256:896f9fa7d498fdc20c582134a15ed6acdc11b78926e655854c43e49fbb24815c",
    productionContractVocabularySha256:
      "sha256:3ef3349d738feb9789aaab6e92246f55948053604a8304706fc1bbd0cd786769",
    vocabularyDiscoveryPromptDigest:
      "sha256:5fa38fea1990c0c493af502f45fc6895d00e7b55e53fff534dc2123ab0865b1d",
    discoveryBatchManifestDigest:
      "sha256:000bb8c32076a74c9468c19e9d8b35901c83279e0635a0e84dc109801977873c",
    classificationBatchManifestDigest:
      "sha256:dee3a0101dcfdeaef9dbb8c3f53d7e4a99de9499eaa7d82a039eb6cac077c96b",
    discoveryRunDigest:
      "sha256:9e2ed56ac9eaf250949b53e7a63d00cf5a3ae6fabf224e33400340d87cf16c3f",
    taxonomyDiscoveryDigest:
      "sha256:5714a6c0cbe198f3e7608163d508b7dbefe1b30a8b8cd7a04268ede03690b4ed",
    omissionInventoryDigest:
      "sha256:1347b64a24adfaf10df4514f1317f96d9d2e5f1fc39ddd647c73289029bb28df",
  });
  assert.deepEqual(taxonomy.mainDomainCodes, [
    "GOVERNANCE_ORGANIZATION_PERSONNEL",
    "CERTIFICATION_LICENSING_CHANGE",
    "AERODROME_MANUAL_DOCUMENT_CONTROL",
    "QUALITY_MANAGEMENT",
    "SAFETY_MANAGEMENT_RISK_ASSESSMENT",
    "AERODROME_DATA_INFORMATION_PUBLICATION",
    "PHYSICAL_CHARACTERISTICS_MOVEMENT_AREA",
    "OBSTACLES_OLS_WORKS",
    "VISUAL_AIDS_MARKINGS_SIGNS_LIGHTING",
    "ELECTRICAL_SYSTEMS_POWER",
    "APRON_GROUND_OPERATIONS",
    "RESCUE_FIRE_FIGHTING_FIRE_SAFETY",
    "EMERGENCY_PLANNING",
    "MAINTENANCE_OPERATIONAL_INSPECTION",
    "RUNWAY_SAFETY_FRICTION_SURFACE_CONDITIONS",
    "WILDLIFE_HAZARD_MANAGEMENT",
    "ENVIRONMENTAL_MANAGEMENT",
    "NIGHT_OPERATIONS_FACILITIES",
  ]);
  assert.deepEqual(taxonomy.externalInvolvementRoles, [
    "TECHNICAL_INTERFACE",
    "COORDINATION",
    "DATA_ORIGINATION",
    "DATA_PUBLICATION",
    "EVIDENCE_CONTRIBUTION",
    "OPERATIONAL_PARTICIPATION",
  ]);
  assert.deepEqual(taxonomy.canonicalTargetKinds, [
    "ORGANIZATION",
    "PERSON",
    "FACILITY",
    "DEVICE",
    "SYSTEM",
    "ASSET",
    "LOCATION",
  ]);
  assert.deepEqual(taxonomy.recommendationStates, [
    "AUTO_PROPOSED_HIGH_CONFIDENCE",
    "MANAGER_REVIEW_REQUIRED",
    "BLOCKED_SOURCE_GAP",
  ]);
  assert.deepEqual(taxonomy.providerPartition.connected, [
    "AERODROME_OPERATOR",
    "ANSP",
    "CNS_PROVIDER",
    "AIS_AIM_PROVIDER",
    "MET_PROVIDER",
    "SAR_ORGANIZATION",
    "AVSEC_PROVIDER",
    "AIR_OPERATOR",
    "AMO",
    "ATO",
    "GROUND_HANDLING",
    "FUEL_PROVIDER",
    "CARGO_REGULATED_AGENT",
    "RPAS_UAS_OPERATOR",
  ]);
  assert.deepEqual(taxonomy.providerPartition.noDefaultRelationship, [
    "CAMO",
    "FSTD",
    "DOA",
    "POA",
    "AEMC",
    "AME",
  ]);
  assert.equal(taxonomy.providerPartition.inspectedScopeCode, "AERODROME_OPERATOR");
  assert.deepEqual(
    taxonomy.providerPartition.externalInvolvementAllowed,
    taxonomy.providerPartition.connected.filter(
      (providerCode) => providerCode !== taxonomy.providerPartition.inspectedScopeCode,
    ),
  );
  assert.deepEqual(taxonomy.forbiddenProviderHierarchyFields, [
    "primaryProvider",
    "secondaryProvider",
    "supportingProvider",
    "owner",
    "ownership",
    "providerHierarchy",
  ]);
  const discovery = readJSON(join(discoveryDirectory, "taxonomy-discovery.json"));
  const omissionReview = readJSON(
    join(discoveryDirectory, "omission-review-inventory.json"),
  );
  assert.deepEqual(
    taxonomy.signalRuleIds,
    omissionReview.signalRules.map((rule) => rule.signalRuleId).sort(),
  );
  assert.deepEqual(taxonomy.topicCodes, discovery.reviewedCandidateVocabulary.topicCues);
  assert.deepEqual(
    taxonomy.inspectionProfileCodes,
    discovery.reviewedCandidateVocabulary.inspectionProfileCues,
  );
  assert.equal(
    taxonomy.inspectionProfileDefinitions.length,
    taxonomy.inspectionProfileCodes.length,
  );
  assert.deepEqual(
    taxonomy.inspectionProfileDefinitions.map((profile) => profile.code),
    taxonomy.inspectionProfileCodes,
  );
  const operationQualifierKeys = taxonomy.operationQualifierDefinitions.map(
    (definition) => definition.key,
  );
  const activityQualifierKeys = taxonomy.activityQualifierDefinitions.map(
    (definition) => definition.key,
  );
  assertSortedUniqueCodes(operationQualifierKeys, "ERR_GATE0B_OPERATION_QUALIFIER_KEYS_OPEN");
  assertSortedUniqueCodes(activityQualifierKeys, "ERR_GATE0B_ACTIVITY_QUALIFIER_KEYS_OPEN");
  assert.equal(
    new Set([...operationQualifierKeys, ...activityQualifierKeys]).size,
    operationQualifierKeys.length + activityQualifierKeys.length,
    "ERR_GATE0B_QUALIFIER_KEY_FAMILY_OVERLAP",
  );
  for (const definition of [
    ...taxonomy.operationQualifierDefinitions,
    ...taxonomy.activityQualifierDefinitions,
  ]) {
    assertExactKeys(
      definition,
      ["key", "allowedValues"],
      "ERR_GATE0B_QUALIFIER_DEFINITION_OPEN",
    );
    assertSortedUniqueCodes(definition.allowedValues, "ERR_GATE0B_QUALIFIER_VALUES_OPEN");
  }
  assert.deepEqual(Object.keys(taxonomy.targetCompatibility), taxonomy.canonicalTargetKinds);
  for (const profileCodes of Object.values(taxonomy.targetCompatibility)) {
    assert.deepEqual(
      profileCodes,
      [...new Set(profileCodes)].sort((left, right) =>
        Buffer.from(left).compare(Buffer.from(right))),
      "ERR_GATE0B_TARGET_COMPATIBILITY_NOT_NORMALIZED",
    );
    assert.ok(
      profileCodes.every((code) => taxonomy.targetProfileCodes.includes(code)),
      "ERR_GATE0B_TARGET_COMPATIBILITY_UNKNOWN_PROFILE",
    );
  }
  for (const profile of taxonomy.inspectionProfileDefinitions) {
    assertExactKeys(
      profile,
      [
        "code",
        "allowedTargetKinds",
        "allowedTargetProfileCodes",
        "allowedInspectionTypeCodes",
        "requiredOperationQualifierKeys",
        "requiredActivityQualifierKeys",
      ],
      "ERR_GATE0B_INSPECTION_PROFILE_DEFINITION_OPEN",
    );
    assert.ok(
      profile.allowedTargetKinds.every((code) => taxonomy.canonicalTargetKinds.includes(code)),
      "ERR_GATE0B_INSPECTION_PROFILE_UNKNOWN_TARGET_KIND",
    );
    assert.ok(
      profile.allowedTargetProfileCodes.every((code) =>
        taxonomy.targetProfileCodes.includes(code)),
      "ERR_GATE0B_INSPECTION_PROFILE_UNKNOWN_TARGET_PROFILE",
    );
    assert.ok(
      profile.allowedInspectionTypeCodes.every((code) =>
        taxonomy.inspectionTypeCodes.includes(code)),
      "ERR_GATE0B_INSPECTION_PROFILE_UNKNOWN_TYPE",
    );
    assert.ok(
      profile.requiredOperationQualifierKeys.every((code) =>
        operationQualifierKeys.includes(code)),
      "ERR_GATE0B_INSPECTION_PROFILE_UNKNOWN_OPERATION_QUALIFIER",
    );
    assert.ok(
      profile.requiredActivityQualifierKeys.every((code) =>
        activityQualifierKeys.includes(code)),
      "ERR_GATE0B_INSPECTION_PROFILE_UNKNOWN_ACTIVITY_QUALIFIER",
    );
    assert.ok(
      profile.allowedTargetKinds.every((kind) =>
        profile.allowedTargetProfileCodes.some((profileCode) =>
          taxonomy.targetCompatibility[kind].includes(profileCode))),
      "ERR_GATE0B_INSPECTION_PROFILE_HAS_DEAD_TARGET_KIND",
    );
    assert.ok(
      profile.allowedTargetProfileCodes.every((profileCode) =>
        profile.allowedTargetKinds.some((kind) =>
          taxonomy.targetCompatibility[kind].includes(profileCode))),
      "ERR_GATE0B_INSPECTION_PROFILE_HAS_DEAD_TARGET_PROFILE",
    );
  }
  assert.deepEqual(
    taxonomy.inspectionTypeCodes,
    discovery.reviewedCandidateVocabulary.inspectionTypeCues,
  );
  assert.deepEqual(
    taxonomy.targetProfileCodes,
    discovery.reviewedCandidateVocabulary.targetProfileCues,
  );
  assert.deepEqual(
    taxonomy.evidenceExpectationCodes,
    discovery.reviewedCandidateVocabulary.evidenceExpectationCues,
  );
  assert.deepEqual(
    taxonomy.applicabilityDispositions,
    discovery.reviewedCandidateVocabulary.applicabilityCues,
  );
  assert.deepEqual(
    taxonomy.externalInvolvementConditionCodes,
    discovery.reviewedCandidateVocabulary.externalInvolvementConditionCues,
  );
  for (const values of [
    taxonomy.topicCodes,
    taxonomy.inspectionProfileCodes,
    taxonomy.inspectionTypeCodes,
    taxonomy.targetProfileCodes,
    taxonomy.evidenceExpectationCodes,
    taxonomy.applicabilityDispositions,
    taxonomy.externalInvolvementConditionCodes,
    taxonomy.blockerCodes,
    taxonomy.rationaleCodes,
    taxonomy.inputFactSelectors,
    taxonomy.signalRuleIds,
    taxonomy.sourceReferenceKinds,
    taxonomy.disagreementCodes,
  ]) {
    assertSortedUniqueCodes(values, "ERR_GATE0B_CONTROLLED_CODES_NOT_CLOSED");
  }
  assert.deepEqual(taxonomy.proposalFields, [
    "activityQualifiers",
    "applicabilityDisposition",
    "canonicalTargetKind",
    "evidenceExpectationCodes",
    "externalInvolvements",
    "inspectionProfileCodes",
    "inspectionTypeCodes",
    "mainDomainCode",
    "operationQualifiers",
    "targetProfileCode",
    "topicCodes",
  ]);
  assert.deepEqual(
    Object.keys(taxonomy.evidenceBindingContract.fieldRules).sort(),
    [...taxonomy.proposalFields].sort(),
  );
  assert.deepEqual(
    Object.keys(taxonomy.evidenceBindingContract.signalRuleFieldRules).sort(),
    taxonomy.signalRuleIds,
  );
  for (const [signalRuleId, rules] of Object.entries(
    taxonomy.evidenceBindingContract.signalRuleFieldRules,
  )) {
    assert.ok(rules.length > 0, `ERR_GATE0B_SIGNAL_RULE_WITHOUT_FIELD_RULE ${signalRuleId}`);
    for (const rule of rules) {
      assertExactKeys(
        rule,
        [
          "proposalField",
          "valueShape",
          "allowedValues",
          "allowedRationaleCodes",
          "signalAloneSatisfiesEvidence",
        ],
        "ERR_GATE0B_SIGNAL_FIELD_RULE_OPEN",
      );
      assert.ok(taxonomy.proposalFields.includes(rule.proposalField));
      assert.notEqual(rule.proposalField, "mainDomainCode");
      assert.ok(rule.allowedValues.length > 0);
      assert.ok(rule.allowedRationaleCodes.every((code) => taxonomy.rationaleCodes.includes(code)));
    }
  }
  for (const rules of Object.values(taxonomy.evidenceBindingContract.signalRuleFieldRules)) {
    for (const rule of rules.filter((entry) => entry.proposalField === "externalInvolvements")) {
      assert.equal(rule.signalAloneSatisfiesEvidence, false);
    }
  }
  assert.deepEqual(taxonomy.evidenceBindingContract.requiredEvidenceFields, [
    "proposalField",
    "proposalValueDigest",
    "rationaleCode",
    "inputFactSelector",
    "inputFactValueDigest",
  ]);
  assert.deepEqual(taxonomy.evidenceBindingContract.edgeSemanticTupleFields, [
    "providerTypeCode",
    "involvementRoleCode",
    "conditionCode",
    "applicabilityDisposition",
  ]);
  assert.deepEqual(taxonomy.commandContract.genericCommandEnvelopeFields, [
    "operationId",
    "idempotencyKey",
    "expectedGenerationId",
  ]);
  assert.deepEqual(taxonomy.commandContract.classificationCasFields, [
    "expectedDraftRevision",
    "expectedDraftContentDigest",
  ]);
  assert.deepEqual(taxonomy.commandContract.lifecycleCasFields, [
    "expectedLifecycleRevision",
    "expectedLifecycleDigest",
  ]);
  assert.deepEqual(taxonomy.commandContract.resetCasFields, [
    "expectedGenerationId",
    "expectedGenerationRevision",
    "expectedGenerationSealDigest",
  ]);
  assert.deepEqual(taxonomy.aggregateContract.exactCounts, {
    sealedItems: 1310,
    passProposalRecords: 2620,
    sourceProposalGaps: 49,
    externalApplicabilityUnresolved: 51,
    sourceGapExternalUnresolvedOverlap: 49,
    extractedCandidate: 1282,
    exactSourceBacked: 28,
  });
  assert.deepEqual(taxonomy.aggregateContract.distributionDimensions, [
    "agreementConfidence",
    "applicabilityDisposition",
    "canonicalTargetKind",
    "disagreementCode",
    "evidenceExpectationCode",
    "externalProviderTypeCode",
    "extractionState",
    "inspectionProfileCode",
    "inspectionTypeCode",
    "mainDomainCode",
    "recommendationState",
    "targetProfileCode",
    "topicCode",
  ]);
  assert.equal(
    taxonomy.taxonomyDigest,
    digestValue(
      "AGA-QUESTION-CLASSIFICATION-TAXONOMY-V1",
      withoutKey(taxonomy, "taxonomyDigest"),
    ),
    "ERR_GATE0B_TAXONOMY_DIGEST_MISMATCH",
  );
});

test("TestGate0BClosesClassificationSchemaPromptAndDigestFreeze", () => {
  for (const path of [
    classificationSpecPath,
    taxonomyPath,
    classificationSchemaPath,
    classificationPromptPath,
  ]) {
    assert.ok(existsSync(path), "ERR_GATE0B_FREEZE_ARTIFACT_MISSING");
  }
  const taxonomy = readJSON(taxonomyPath);
  const schema = readJSON(classificationSchemaPath);
  const prompt = readFileSync(classificationPromptPath, "utf8");
  const specification = readFileSync(classificationSpecPath, "utf8");
  assert.equal(schema.$schema, "https://json-schema.org/draft/2020-12/schema");
  assert.equal(schema.$id, "urn:aviasurveil360:aga-question-classification:v1");
  assert.equal(schema.title, "AGA Hybrid Question Classification V1");
  for (const definition of [
    "baseIdentity",
    "modelDescriptor",
    "classificationPackageFacts",
    "classificationResearchCandidateFacts",
    "classificationPassInputItem",
    "classificationPassInput",
    "sealedBaseQuestionRef",
    "workspaceQuestionRef",
    "questionRef",
    "sealedBaseParentQuestionKey",
    "workspaceParentQuestionKey",
    "parentQuestionKey",
    "operationQualifier",
    "activityQualifier",
    "confidenceEvidence",
    "externalInvolvementConfidenceEvidence",
    "sourceReference",
    "externalInvolvement",
    "proposalProjection",
    "passProposalRecord",
    "passBatchOutput",
    "passSealReceipt",
    "sealedClassificationItem",
    "proposalResolution",
    "draftItem",
    "classificationDistributions",
    "classificationExceptions",
    "classificationAggregate",
    "fixedInputDigests",
    "classificationRunReceipt",
    "loadingClassificationRunRecord",
    "rejectedClassificationRunRecord",
  ]) {
    assert.ok(schema.$defs?.[definition], "ERR_GATE0B_SCHEMA_DEFINITION_MISSING");
  }
  const objectSchemas = [];
  const visitSchema = (value) => {
    if (Array.isArray(value)) {
      value.forEach(visitSchema);
    } else if (value && typeof value === "object") {
      if (value.type === "object") objectSchemas.push(value);
      Object.values(value).forEach(visitSchema);
    }
  };
  visitSchema(schema);
  assert.ok(objectSchemas.length >= 12);
  assert.ok(
    objectSchemas.every((value) => value.additionalProperties === false),
    "ERR_GATE0B_SCHEMA_OBJECT_NOT_CLOSED",
  );
  const schemaArrayFieldNames = new Set();
  const collectSchemaArrayFields = (value) => {
    if (!value || typeof value !== "object") return;
    if (value.properties && typeof value.properties === "object") {
      for (const [field, fieldSchema] of Object.entries(value.properties)) {
        if (fieldSchema?.type === "array") schemaArrayFieldNames.add(field);
      }
    }
    Object.values(value).forEach(collectSchemaArrayFields);
  };
  collectSchemaArrayFields(schema.$defs);
  assert.deepEqual(
    [...schemaArrayFieldNames].sort(),
    Object.keys(taxonomy.normalization.schemaArrayFieldClasses).sort(),
    "ERR_GATE0B_SCHEMA_ARRAY_CLASSIFICATION_INCOMPLETE",
  );
  assert.ok(
    Object.values(taxonomy.normalization.schemaArrayFieldClasses).every(
      (classification) => classification === "ORDERED" || classification === "SET_LIKE",
    ),
  );
  assert.ok(
    schema.oneOf.some((entry) => entry.$ref === "#/$defs/passBatchOutput"),
    "ERR_GATE0B_PASS_BATCH_NOT_TOP_LEVEL_SCHEMA",
  );
  assert.ok(!schema.oneOf.some((entry) => entry.$ref === "#/$defs/classificationPassInput"));
  assert.ok(schema.oneOf.some((entry) => entry.$ref === "#/$defs/passSealReceipt"));
  assert.equal(schema["x-taxonomy-version"], taxonomy.taxonomyVersion);
  assert.equal(schema["x-taxonomy-digest"], taxonomy.taxonomyDigest);
  const taxonomyVersionPins = [];
  const taxonomyDigestPins = [];
  const collectTaxonomyPins = (value) => {
    if (!value || typeof value !== "object") return;
    if (value.taxonomyVersion?.const && typeof value.taxonomyVersion === "object") {
      taxonomyVersionPins.push(value.taxonomyVersion.const);
    }
    if (value.taxonomyDigest?.const && typeof value.taxonomyDigest === "object") {
      taxonomyDigestPins.push(value.taxonomyDigest.const);
    }
    Object.values(value).forEach(collectTaxonomyPins);
  };
  collectTaxonomyPins(schema.$defs);
  assert.ok(taxonomyVersionPins.length >= 3);
  assert.ok(taxonomyDigestPins.length >= 3);
  assert.ok(taxonomyVersionPins.every((value) => value === taxonomy.taxonomyVersion));
  assert.ok(taxonomyDigestPins.every((value) => value === taxonomy.taxonomyDigest));
  const expectedBatchManifestDigest =
    "sha256:dee3a0101dcfdeaef9dbb8c3f53d7e4a99de9499eaa7d82a039eb6cac077c96b";
  const promptDigestSchemas = [];
  const batchManifestDigestSchemas = [];
  const collectFrozenInputPins = (value) => {
    if (!value || typeof value !== "object") return;
    if (value.properties?.promptDigest) {
      promptDigestSchemas.push(value.properties.promptDigest);
    }
    if (value.properties?.batchManifestDigest?.const) {
      batchManifestDigestSchemas.push(value.properties.batchManifestDigest);
    }
    Object.values(value).forEach(collectFrozenInputPins);
  };
  collectFrozenInputPins(schema.$defs);
  assert.equal(promptDigestSchemas.length, 8);
  assert.ok(promptDigestSchemas.every((value) => value.$ref === "#/$defs/sha256"));
  assert.equal(batchManifestDigestSchemas.length, 5);
  assert.ok(
    batchManifestDigestSchemas.every((value) => value.const === expectedBatchManifestDigest),
  );
  assert.deepEqual(
    Object.keys(schema.$defs.proposalProjection.properties).sort(),
    [...taxonomy.proposalFields].sort(),
  );
  assert.deepEqual(
    schema.$defs.sealedBaseQuestionRef.required,
    ["questionOrigin", ...taxonomy.questionReferenceContract.baseIdentityFields],
  );
  assert.deepEqual(
    schema.$defs.workspaceQuestionRef.required,
    taxonomy.questionReferenceContract.workspaceQuestionFields,
  );
  assert.equal(schema.$defs.parentQuestionKey.oneOf.length, 3);
  assert.deepEqual(
    schema.$defs.passProposalRecord.required,
    [
      "identity",
      "classificationRunId",
      "passRole",
      "passRunId",
      "promptDigest",
      "modelDescriptorDigest",
      "inputDigest",
      "proposalProjection",
      "rationaleCodes",
      "confidenceEvidence",
      "sourceRefs",
      "passResultDigest",
    ],
  );
  assert.deepEqual(
    schema.$defs.confidenceEvidence.properties.signalRuleId.enum,
    taxonomy.signalRuleIds,
  );
  assert.deepEqual(
    schema.$defs.confidenceEvidence.properties.inputFactSelector.enum,
    taxonomy.inputFactSelectors,
  );
  assert.equal(
    schema.$defs.confidenceEvidence.properties.proposalField.enum.includes(
      "externalInvolvements",
    ),
    false,
  );
  assert.equal(
    schema.$defs.confidenceEvidence.allOf[0].then.properties.inputFactSelector.const,
    "VALIDATOR_SIGNAL_RULE_MATCH_DIGEST",
  );
  assert.equal(
    schema.$defs.externalInvolvementConfidenceEvidence.properties.proposalField.const,
    "externalInvolvements",
  );
  assert.deepEqual(
    schema.$defs.externalInvolvementConfidenceEvidence.properties.signalRuleId.enum,
    taxonomy.signalRuleIds.filter((signalRuleId) =>
      taxonomy.evidenceBindingContract.signalRuleFieldRules[signalRuleId].some(
        (rule) => rule.proposalField === "externalInvolvements",
      )),
  );
  assert.deepEqual(
    schema.$defs.externalInvolvement.properties.providerTypeCode.enum,
    taxonomy.providerPartition.externalInvolvementAllowed,
  );
  assert.equal(
    schema.$defs.proposalProjection.properties.externalInvolvements.maxItems,
    taxonomy.providerPartition.externalInvolvementAllowed.length,
  );
  assert.equal(
    schema.$defs.externalInvolvement.properties.confidenceEvidence.items.$ref,
    "#/$defs/externalInvolvementConfidenceEvidence",
  );
  assert.deepEqual(
    schema.$defs.externalInvolvementRationaleCode.enum,
    taxonomy.evidenceBindingContract.combinationProfiles.EXTERNAL_EDGE.allowedRationaleCodes,
  );
  assert.equal(
    schema.$defs.proposalProjection.properties.operationQualifiers.items.$ref,
    "#/$defs/operationQualifier",
  );
  assert.equal(
    schema.$defs.proposalProjection.properties.activityQualifiers.items.$ref,
    "#/$defs/activityQualifier",
  );
  assert.deepEqual(
    schema.$defs.operationQualifier.oneOf.map((entry) => entry.properties.key.const),
    taxonomy.operationQualifierDefinitions.map((entry) => entry.key),
  );
  assert.deepEqual(
    schema.$defs.activityQualifier.oneOf.map((entry) => entry.properties.key.const),
    taxonomy.activityQualifierDefinitions.map((entry) => entry.key),
  );
  for (const [schemaDefinition, taxonomyDefinitions] of [
    [schema.$defs.operationQualifier, taxonomy.operationQualifierDefinitions],
    [schema.$defs.activityQualifier, taxonomy.activityQualifierDefinitions],
  ]) {
    schemaDefinition.oneOf.forEach((entry, index) => {
      assert.deepEqual(entry.properties.value.enum, taxonomyDefinitions[index].allowedValues);
    });
  }
  const exactFormPattern = new RegExp(schema.$defs.formCode.pattern, "u");
  assert.ok(expectedFormCodes().every((code) => exactFormPattern.test(code)));
  assert.ok(
    ["FSS-AGA-FORM-000", "FSS-AGA-FORM-035", "FSS-AGA-FORM-049", "FSS-AGA-FORM-054"]
      .every((code) => !exactFormPattern.test(code)),
    "ERR_GATE0B_FORM_CODE_PATTERN_OPEN",
  );
  assert.equal(schema.$defs.sealedClassificationItem.properties.formCode.$ref, "#/$defs/formCode");
  assert.deepEqual(
    schema.$defs.sealedClassificationItem.required,
    sealedClassificationItemFields,
  );
  assert.deepEqual(
    Object.keys(schema.$defs.sealedClassificationItem.properties),
    sealedClassificationItemFields,
  );
  assert.deepEqual(
    schema.$defs.sealedClassificationItem.properties.extractionState.enum,
    ["EXTRACTED_CANDIDATE", "EXACT_SOURCE_BACKED"],
  );
  assert.equal(
    schema.$defs.sealedClassificationItem.properties.modelDescriptorDigests.minItems,
    1,
  );
  assert.equal(
    schema.$defs.sealedClassificationItem.properties.modelDescriptorDigests.maxItems,
    2,
  );
  assert.equal(schema.$defs.passBatchOutput.properties.records.maxItems, 64);
  assert.equal(
    schema.$defs.passBatchOutput.properties.schemaVersion.const,
    "aga-hybrid-classification-pass-batch/v1",
  );
  assert.equal(schema.$defs.draftItem.allOf.length, 7);
  const draftSchemaText = canonicalJSON(schema.$defs.draftItem);
  for (const invariantValue of [
    "AUTO_PRESELECTED",
    "PENDING_MANAGER_REVIEW",
    "MANAGER_DISPOSED",
    "AUTO_PROPOSED_HIGH_CONFIDENCE",
    "BLOCKED_SOURCE_GAP",
  ]) {
    assert.ok(draftSchemaText.includes(invariantValue));
  }
  assert.equal(schema.$defs.passSealReceipt.properties.itemCount.const, 1310);
  assert.equal(schema.$defs.passSealReceipt.properties.batchCount.const, 25);
  assert.equal(
    schema.$defs.passSealReceipt.properties.orderedPassResultDigests.maxItems,
    1310,
  );
  assert.equal(schema.$defs.classificationPassInput.properties.schemaVersion.const,
    "aga-hybrid-classification-pass-input/v1");
  assert.equal(schema.$defs.classificationPassInput.properties.items.maxItems, 64);
  assert.equal(schema.$defs.classificationPassInputItem.properties.questionBody.maxLength, 2048);
  assert.equal(schema.$defs.classificationRunReceipt.properties.state.const, "SEALED");
  assert.equal(schema.$defs.loadingClassificationRunRecord.properties.state.const, "LOADING");
  assert.equal(schema.$defs.rejectedClassificationRunRecord.properties.state.const, "REJECTED");
  assert.ok(!Object.hasOwn(schema.$defs.rejectedClassificationRunRecord.properties, "passOneSealDigest"));
  assert.ok(!Object.hasOwn(schema.$defs.rejectedClassificationRunRecord.properties, "aggregateDigest"));
  assert.ok(schema.$defs.classificationRunReceipt.required.includes("modelDescriptors"));
  assert.equal(
    schema.$defs.classificationAggregate.properties.passProposalRecordCount.const,
    2620,
  );
  assert.deepEqual(
    schema.$defs.classificationAggregate.required,
    [
      "itemCount",
      "passProposalRecordCount",
      "orderedItemSemanticDigests",
      "distributions",
      "exceptions",
      "distributionDigest",
      "aggregateDigest",
    ],
  );
  assert.deepEqual(
    Object.keys(schema.$defs.classificationDistributions.properties).sort(),
    taxonomy.aggregateContract.distributionDimensions.map((dimension) => `${dimension}Counts`).sort(),
  );
  assert.deepEqual(
    schema.$defs.classificationDistributions.properties.extractionStateCounts.allOf
      .map((constraint) => [
        constraint.contains.properties.code.const,
        constraint.contains.properties.count.const,
      ]),
    [["EXTRACTED_CANDIDATE", 1282], ["EXACT_SOURCE_BACKED", 28]],
  );
  assert.deepEqual(schema.$defs.exceptionInventory.required, [
    "count",
    "orderedIdentityDigests",
    "orderedIdentityDigest",
  ]);
  assert.equal(
    schema.$defs.classificationExceptions.properties.blockedSourceGap
      .allOf[1].properties.count.const,
    49,
  );
  assert.equal(
    schema.$defs.classificationExceptions.properties.externalApplicabilityUnresolved
      .allOf[1].properties.count.const,
    51,
  );
  const schemaText = canonicalJSON(schema);
  for (const forbidden of taxonomy.forbiddenProviderHierarchyFields) {
    assert.ok(!schemaText.includes(`\"${forbidden}\"`), "ERR_GATE0B_PROVIDER_HIERARCHY_FIELD");
  }
  assert.ok(prompt.includes("passRole is exactly CANDIDATE or CHALLENGE"));
  assert.ok(prompt.includes("aga-hybrid-classification-pass-batch/v1"));
  assert.ok(prompt.includes("externalInvolvementAllowed"));
  assert.ok(prompt.includes("NO_DEFAULT_AGA_RELATIONSHIP"));
  assert.ok(prompt.includes("Do not output governance fields"));
  assert.ok(prompt.includes("Evidence omission is structurally valid"));
  assert.ok(prompt.includes("Never output question text, a body fragment, or chain of thought"));
  assert.ok(prompt.includes(taxonomy.taxonomyVersion));
  assert.ok(prompt.includes(taxonomy.taxonomyDigest));
  assert.ok(specification.includes("CAP acceptance is not Finding closure."));
  assert.ok(specification.includes("READY_FOR_DEMO_SIMULATION"));
  assert.ok(specification.includes("questionOrigin"));
  assert.ok(specification.includes("parentQuestionKey"));
  assert.ok(specification.includes("ACCEPT_CANDIDATE_PASS"));
  assert.ok(specification.includes("ACCEPT_CHALLENGE_PASS"));
  assert.ok(specification.includes("SET_EXACT"));
  assert.ok(specification.includes("Accepted-package question bodies never enter"));
  assert.ok(specification.includes("Workspace-authored bodies are append-only"));
  assert.deepEqual(
    {
      specification: sha256(readFileSync(classificationSpecPath)),
      taxonomy: sha256(readFileSync(taxonomyPath)),
      schema: sha256(readFileSync(classificationSchemaPath)),
      prompt: sha256(readFileSync(classificationPromptPath)),
    },
    frozenGate0BFileDigests,
    "ERR_GATE0B_FILE_DIGEST_FREEZE_MISMATCH",
  );
});

test("TestGate0BFreezesStatusesTransitionsNormalizationAndEvidenceBinding", () => {
  assert.ok(existsSync(taxonomyPath), "ERR_GATE0B_FREEZE_ARTIFACT_MISSING");
  const taxonomy = readJSON(taxonomyPath);
  assert.deepEqual(taxonomy.agreementConfidenceStates, ["HIGH", "MEDIUM", "LOW"]);
  assert.deepEqual(taxonomy.passRoles, ["CANDIDATE", "CHALLENGE"]);
  assert.deepEqual(taxonomy.classificationRunStates, ["LOADING", "SEALED", "REJECTED"]);
  assert.deepEqual(taxonomy.draftStates, [
    "WORKING",
    "READY_FOR_DEMO_SIMULATION",
    "SUPERSEDED",
  ]);
  assert.deepEqual(taxonomy.draftItemOrigins, [
    "SEALED_BASE",
    "MANAGER_AUTHORED",
    "MANAGER_REWORDED",
  ]);
  assert.deepEqual(taxonomy.draftReviewStates, [
    "AUTO_PRESELECTED",
    "PENDING_MANAGER_REVIEW",
    "MANAGER_DISPOSED",
  ]);
  assert.deepEqual(taxonomy.draftDispositions, ["INCLUDE", "EXCLUDE", "DEFER"]);
  assert.deepEqual(taxonomy.workspaceGenerationStates, ["ACTIVE", "RESET"]);
  assert.deepEqual(taxonomy.questionReferenceContract.baseIdentityFields, [
    "packageVersion",
    "packageJsonSha256",
    "formCode",
    "proposalId",
    "ordinal",
    "textDigest",
  ]);
  assert.deepEqual(taxonomy.questionReferenceContract.questionOrigins, [
    "SEALED_BASE",
    "WORKSPACE",
  ]);
  assert.deepEqual(taxonomy.questionReferenceContract.workspaceIdentityTransitions, {
    ADD_CANDIDATE: {
      parentKind: "NULL",
      root: "ALLOCATE",
      version: "ALLOCATE",
      proposal: "ALLOCATE",
      rootSequence: "ALLOCATE",
    },
    REWORD_BASE: {
      parentKind: "SEALED_BASE",
      root: "ALLOCATE",
      version: "ALLOCATE",
      proposal: "ALLOCATE",
      rootSequence: "ALLOCATE_AT_BASE_POSITION",
    },
    REWORD_WORKSPACE: {
      parentKind: "WORKSPACE",
      root: "PRESERVE",
      version: "ALLOCATE",
      proposal: "ALLOCATE",
      rootSequence: "PRESERVE",
    },
  });
  assert.deepEqual(taxonomy.normalization.evidenceSortFields, [
    "proposalField",
    "proposalValueDigest",
    "rationaleCode",
    "inputFactSelector",
    "inputFactValueDigest",
    "signalRuleId",
  ]);
  assert.deepEqual(taxonomy.normalization.orderedFields, [
    "boundedBatches",
    "draftRevisions",
    "eventStream",
    "items",
    "lifecycleRevisions",
    "orderedBatchOutputDigests",
    "orderedIdentityDigests",
    "orderedInputDigests",
    "orderedItemSemanticDigests",
    "orderedPassResultDigests",
    "packageForms",
    "questionIdentities",
    "records",
    "sourceProposalDigests",
    "sourceReferenceDigests",
  ]);
  assert.deepEqual(taxonomy.normalization.setLikeFields, [
    "activityQualifiers",
    "agreementConfidenceCounts",
    "applicabilityDispositionCounts",
    "blockerCodes",
    "canonicalTargetKindCounts",
    "confidenceEvidence",
    "disagreementCodeCounts",
    "evidenceExpectationCodeCounts",
    "evidenceExpectationCodes",
    "externalInvolvements",
    "externalProviderTypeCodeCounts",
    "extractionStateCounts",
    "inspectionProfileCodeCounts",
    "inspectionProfileCodes",
    "inspectionTypeCodeCounts",
    "inspectionTypeCodes",
    "mainDomainCodeCounts",
    "modelDescriptorDigests",
    "modelDescriptors",
    "operationQualifiers",
    "passDisagreementCodes",
    "rationaleCodes",
    "recommendationStateCounts",
    "sourceRefs",
    "targetProfileCodeCounts",
    "topicCodeCounts",
    "topicCodes",
    "unavailableFields",
  ]);
  assert.deepEqual(taxonomy.normalization.sourceReferenceSortFields, [
    "kind",
    "referenceDigest",
  ]);
  assert.deepEqual(taxonomy.normalization.distributionCountSemanticKeyFields, ["code"]);
  assert.deepEqual(taxonomy.normalization.distributionCountSortFields, ["code", "count"]);
  assert.deepEqual(taxonomy.normalization.qualifierSemanticKeyFields, ["key"]);
  assert.equal(
    taxonomy.normalization.inspectionProfileQualifierKeyPolicy,
    "REQUIRED_KEYS_ARE_EXACT_ALLOWED_KEYS_EXTRAS_REJECTED",
  );
  assert.equal(
    taxonomy.normalization.optionalSortFieldAbsenceEncoding.signalRuleId,
    "EMPTY_UTF8_STRING_SORTS_BEFORE_CODE",
  );
  assert.deepEqual(taxonomy.digestContract.domainSeparators, {
    canonicalJSONConformance: "AGA-CANONICAL-JSON-CONFORMANCE-V1",
    passProposal: "AGA-CLASSIFICATION-PASS-PROPOSAL-V1",
    passBatchOutput: "AGA-CLASSIFICATION-PASS-BATCH-V1",
    passSeal: "AGA-CLASSIFICATION-PASS-SEAL-V1",
    proposalScalar: "AGA-PROPOSAL-VALUE-SCALAR-V1",
    proposalSetMember: "AGA-PROPOSAL-VALUE-SET-MEMBER-V1",
    proposalQualifier: "AGA-PROPOSAL-VALUE-QUALIFIER-V1",
    proposalExternalInvolvement: "AGA-PROPOSAL-VALUE-EXTERNAL-INVOLVEMENT-V1",
    baseIdentity: "AGA-CLASSIFICATION-BASE-IDENTITY-V1",
    classificationSourceSnapshot: "AGA-CLASSIFICATION-SOURCE-SNAPSHOT-V1",
    classificationOrderedIdentities: "AGA-CLASSIFICATION-ORDERED-IDENTITIES-V1",
    classificationBatchEntry: "AGA-CLASSIFICATION-BATCH-ENTRY-V1",
    classificationBatchManifest: "AGA-CLASSIFICATION-BATCH-MANIFEST-SET-V2",
    classificationPassInput: "AGA-CLASSIFICATION-PASS-INPUT-V1",
    classificationPassInputSet: "AGA-CLASSIFICATION-PASS-INPUT-SET-V1",
    classificationRunInput: "AGA-CLASSIFICATION-RUN-INPUT-V1",
    modelDescriptor: "AGA-MODEL-DESCRIPTOR-V1",
    formMetadataFact: "AGA-FORM-METADATA-FACT-V1",
    sourceProposalFact: "AGA-SOURCE-PROPOSAL-FACT-V1",
    sourceProposalSetFact: "AGA-SOURCE-PROPOSAL-SET-V1",
    sourceReferenceFact: "AGA-SOURCE-REFERENCE-FACT-V1",
    sourceReferenceSetFact: "AGA-SOURCE-REFERENCE-SET-V1",
    researchRowFact: "AGA-RESEARCH-ROW-FACT-V1",
    validatorSignalRuleMatchFact: "AGA-VALIDATOR-SIGNAL-RULE-MATCH-V1",
    semanticItem: "AGA-CLASSIFICATION-ITEM-V1",
    distributions: "AGA-CLASSIFICATION-DISTRIBUTIONS-V1",
    exceptionIdentityInventory: "AGA-CLASSIFICATION-EXCEPTION-IDENTITY-INVENTORY-V1",
    aggregate: "AGA-CLASSIFICATION-AGGREGATE-V1",
    run: "AGA-CLASSIFICATION-RUN-V1",
  });
  assert.deepEqual(taxonomy.digestContract.semanticItemExcludedFields, [
    "itemSemanticDigest",
    "passOneResultDigest",
    "passTwoResultDigest",
    "classificationRunDigest",
    "aggregateDigest",
  ]);
  assert.deepEqual(taxonomy.digestContract.aggregateExcludedFields, ["aggregateDigest"]);
  assert.deepEqual(taxonomy.digestContract.passProposalExcludedFields, ["passResultDigest"]);
  assert.deepEqual(taxonomy.digestContract.passBatchOutputExcludedFields, ["batchOutputDigest"]);
  assert.deepEqual(taxonomy.digestContract.passSealExcludedFields, ["passSealDigest"]);
  assert.deepEqual(taxonomy.digestContract.distributionExcludedFields, ["distributionDigest"]);
  assert.deepEqual(taxonomy.digestContract.runExcludedFields, ["classificationRunDigest"]);
  assert.equal(taxonomy.digestContract.hashAlgorithm, "SHA-256");
  assert.equal(
    taxonomy.digestContract.hashFraming,
    "UTF8_DOMAIN_IMMEDIATELY_FOLLOWED_BY_UTF8_CANONICAL_JSON_NO_DELIMITER",
  );
  assert.deepEqual(
    taxonomy.digestContract.knownAnswerVectors.map((vector) => vector.digest),
    [
      "sha256:6d45f8e40c2b1cc30de9f61afc9c8f64210bcf323979fe3fe28f4e0805ba21c8",
      "sha256:6566c6ece811ee091cde2b4ae7a7fd918c6fec479c56dcc1347280cc0adf61ee",
      "sha256:63ed7d387f03727c0dd5cdfb6c8db60ad8aaa13711e49d2c2a6960e782b90ec9",
      "sha256:8e107c581635b3af72e3d1e4870123ac85ea0caa3e06ef71396ab4d3f0c70e0f",
      "sha256:b4cb2a2343aa92fdb0326f5934b7bd2e6a5347fed01685f92917db1e8ec20a21",
    ],
  );
  for (const vector of taxonomy.digestContract.knownAnswerVectors) {
    assert.equal(
      digestValue(vector.domainSeparator, vector.payload),
      vector.digest,
      "ERR_GATE0B_DIGEST_KNOWN_ANSWER_MISMATCH",
    );
  }
  assert.deepEqual(taxonomy.digestContract.proposalValuePreimages, {
    SCALAR: ["proposalField", "value"],
    SET_MEMBER: ["proposalField", "value"],
    QUALIFIER_PAIR: ["proposalField", "key", "value"],
    EXTERNAL_EDGE_TUPLE: [
      "proposalField",
      "providerTypeCode",
      "involvementRoleCode",
      "conditionCode",
      "applicabilityDisposition",
    ],
  });
  assert.deepEqual(
    Object.keys(taxonomy.digestContract.inputFactDigestRules).sort(),
    taxonomy.inputFactSelectors,
  );
  assert.deepEqual(taxonomy.digestContract.sourceElementDigestRules, {
    SOURCE_PROPOSAL_FACT: {
      domainSeparator: "AGA-SOURCE-PROPOSAL-FACT-V1",
      preimage: "COMPLETE_SUPPLIED_SOURCE_PROPOSAL_OBJECT_FROM_PINNED_PACKAGE",
    },
    SOURCE_REFERENCE_FACT: {
      domainSeparator: "AGA-SOURCE-REFERENCE-FACT-V1",
      preimage: "COMPLETE_SUPPLIED_SOURCE_REFERENCE_VALUE_FROM_PINNED_PACKAGE",
    },
  });
  assert.equal(
    taxonomy.digestContract.inputFactDigestRules.QUESTION_BODY_DIGEST.hashInput,
    "RAW_UTF8_QUESTION_BODY_NO_DOMAIN_SEPARATOR",
  );
  assert.deepEqual(
    taxonomy.digestContract.inputFactDigestRules.FORM_METADATA_DIGEST.preimageFields,
    ["formKind", "formRiskBand"],
  );
  assert.equal(
    taxonomy.digestContract.inputFactDigestRules.SOURCE_PROPOSAL_DIGEST.arrayNormalization,
    "ACCEPTED_PACKAGE_ORDER_PRESERVED_DUPLICATES_REJECTED",
  );
  assert.equal(
    taxonomy.digestContract.inputFactDigestRules.VALIDATOR_SIGNAL_RULE_MATCH_DIGEST.matched,
    true,
  );
  assert.deepEqual(taxonomy.digestContract.passSealPayloadFields, [
    "classificationRunId",
    "passRole",
    "passRunId",
    "promptDigest",
    "modelDescriptorDigest",
    "batchManifestDigest",
    "batchCount",
    "itemCount",
    "orderedInputDigests",
    "passInputSetDigest",
    "orderedBatchOutputDigests",
    "orderedPassResultDigests",
  ]);
  assert.deepEqual(taxonomy.digestContract.passInputSetPayloadFields, [
    "classificationRunId",
    "passRole",
    "passRunId",
    "batchManifestDigest",
    "orderedInputDigests",
  ]);
  assert.deepEqual(taxonomy.digestContract.runInputPayloadFields, [
    "taxonomyVersion",
    "taxonomyDigest",
    "fixedInputDigests",
    "promptDigest",
    "batchManifestDigest",
  ]);
  assert.deepEqual(taxonomy.digestContract.distributionPayloadFields, ["distributions"]);
  assert.deepEqual(taxonomy.digestContract.exceptionIdentityInventoryPayloadFields, [
    "count",
    "orderedIdentityDigests",
  ]);
  assert.equal(
    taxonomy.digestContract.promptDigestContract.preimage,
    "EXACT_TRACKED_FILE_BYTES",
  );
  assert.deepEqual(taxonomy.confidenceContract.coreProposalFields, [
    "mainDomainCode",
    "canonicalTargetKind",
    "targetProfileCode",
    "applicabilityDisposition",
    "inspectionProfileCodes"
  ]);
  assert.deepEqual(taxonomy.confidenceContract.orderedPrecedence, ["LOW", "MEDIUM", "HIGH"]);
  assert.deepEqual(taxonomy.recommendationPrecedence.map((rule) => rule.result), [
    "BLOCKED_SOURCE_GAP",
    "MANAGER_REVIEW_REQUIRED",
    "AUTO_PROPOSED_HIGH_CONFIDENCE",
  ]);
  assert.ok(taxonomy.fatalRunErrorCodes.includes("IDENTITY_MISMATCH"));
  assert.ok(taxonomy.fatalRunErrorCodes.includes("QUESTION_TEXT_LEAK"));
  assert.ok(taxonomy.fatalRunErrorCodes.includes("UNKNOWN_SIGNAL_RULE"));
  assert.deepEqual(taxonomy.proposalResolutionModes, [
    "ACCEPT_CANDIDATE_PASS",
    "ACCEPT_CHALLENGE_PASS",
    "SET_EXACT",
  ]);
  assert.equal(taxonomy.draftTransitionContract.batchMaximumItems, 500);
  assert.deepEqual(
    taxonomy.draftTransitionContract.initialSealedBaseByRecommendation,
    {
      AUTO_PROPOSED_HIGH_CONFIDENCE: {
        draftAgreementConfidence: "COPY_SEALED",
        draftReviewState: "AUTO_PRESELECTED",
        draftDisposition: "INCLUDE",
      },
      MANAGER_REVIEW_REQUIRED: {
        draftAgreementConfidence: "COPY_SEALED",
        draftReviewState: "PENDING_MANAGER_REVIEW",
        draftDisposition: null,
      },
      BLOCKED_SOURCE_GAP: {
        draftAgreementConfidence: "COPY_SEALED",
        draftReviewState: "PENDING_MANAGER_REVIEW",
        draftDisposition: null,
      },
    },
  );
  assert.deepEqual(taxonomy.draftTransitionContract.dispositionInvariant, {
    AUTO_PRESELECTED: ["INCLUDE"],
    PENDING_MANAGER_REVIEW: [null],
    MANAGER_DISPOSED: ["INCLUDE", "EXCLUDE", "DEFER"],
  });
  assert.equal(
    taxonomy.draftTransitionContract.postResolutionRequiresSeparateDisposition,
    true,
  );
  assert.equal(
    taxonomy.draftTransitionContract.semanticEditSuccessor.draftAgreementConfidence,
    null,
  );
  assert.equal(
    taxonomy.draftTransitionContract.semanticEditSuccessor.draftReviewState,
    "PENDING_MANAGER_REVIEW",
  );
  assert.equal(taxonomy.lifecycleCodes.findingStates.includes("CAP_ACCEPTED"), false);
  assert.ok(taxonomy.lifecycleCodes.findingStates.includes("PENDING_CLOSURE"));
  assert.ok(taxonomy.lifecycleCodes.findingStates.includes("CLOSED"));
  assert.ok(taxonomy.lifecycleCodes.capRevisionStates.includes("ACCEPTED"));
  assert.deepEqual(taxonomy.lifecycleCodes.evidenceDecisionToFindingState.CLOSE, {
    evidenceReviewState: "ACCEPTED",
    findingState: "CLOSED",
    closureBasis: "EVIDENCE_VERIFIED",
  });
  assert.equal(
    taxonomy.lifecycleCodes.capAcceptedFindingStates.evidenceRequired,
    "EVIDENCE_REQUIRED",
  );
  assert.equal(
    taxonomy.lifecycleCodes.capAcceptedFindingStates.noEvidenceRequired,
    "PENDING_CLOSURE",
  );
  assert.deepEqual(taxonomy.lifecycleCodes.classificationQueryOperations, [
    "GET_DRAFT",
    "GET_HISTORY",
    "GET_PROVIDER_CONFIGURATION",
    "GET_SUMMARY",
    "GET_TAXONOMY",
    "SEARCH_ITEMS",
  ]);
  assert.deepEqual(
    taxonomy.lifecycleCodes.classificationCommandOperations,
    [...taxonomy.managerActions, "EXECUTE_BATCH", "PREVIEW_BATCH"].sort((left, right) =>
      Buffer.from(left).compare(Buffer.from(right))),
  );
  assert.deepEqual(taxonomy.lifecycleCodes.recommendationCommandOperations, [
    "CREATE_INSPECTION",
    "CREATE_RECOMMENDATION",
  ]);
  assert.deepEqual(taxonomy.lifecycleCodes.lifecycleQueryOperations, [
    "GET_CAP_EVIDENCE",
    "GET_FINDING",
    "GET_INSPECTION",
    "GET_ROLE_HISTORY",
  ]);
  assert.deepEqual(taxonomy.lifecycleCodes.lifecycleCommandOperations, [
    "AUTHORIZED_CLOSE",
    "CONVERT_POTENTIAL_FINDING",
    "CREATE_POTENTIAL_FINDING",
    "DISMISS_POTENTIAL_FINDING",
    "RECORD_RESPONSE",
    "REOPEN_CHECKLIST",
    "RETURN_POTENTIAL_FINDING",
    "REVIEW_CAP",
    "START_INSPECTION",
    "SUBMIT_CAP_REVISION",
    "SUBMIT_CHECKLIST",
    "SUBMIT_EVIDENCE_VERSION",
    "VERIFY_EVIDENCE",
  ]);
  assert.deepEqual(taxonomy.lifecycleCodes.adminCommandOperations, ["RESET_GENERATION"]);
  assert.deepEqual(taxonomy.lifecycleCodes.findingInitialStateByRequirements, {
    CAP_REQUIRED_EVIDENCE_REQUIRED: "WAITING_FOR_CAP",
    CAP_REQUIRED_NO_EVIDENCE: "WAITING_FOR_CAP",
    NO_CAP_EVIDENCE_REQUIRED: "EVIDENCE_REQUIRED",
    NO_CAP_NO_EVIDENCE: "PENDING_CLOSURE",
  });
  assert.equal(taxonomy.lifecycleCodes.dueDateRequirementAffectsStateBranch, false);
  assert.deepEqual(taxonomy.lifecycleCodes.capReviewOutcomeMappings, {
    REJECT: { capRevisionState: "REJECTED", findingState: "CAP_REJECTED" },
    REQUEST_MORE_INFORMATION: {
      capRevisionState: "MORE_INFORMATION_REQUESTED",
      findingState: "CAP_MORE_INFORMATION_REQUESTED",
    },
    ACCEPT_EVIDENCE_REQUIRED: {
      capRevisionState: "ACCEPTED",
      findingState: "EVIDENCE_REQUIRED",
    },
    ACCEPT_NO_EVIDENCE: {
      capRevisionState: "ACCEPTED",
      findingState: "PENDING_CLOSURE",
    },
  });
  assert.deepEqual(
    taxonomy.lifecycleCodes.lifecycleEventCodes,
    taxonomy.lifecycleCodes.lifecycleCommandOperations,
  );
  assert.deepEqual(
    Object.keys(taxonomy.lifecycleCodes.operationTransitionRegistry).sort(),
    taxonomy.lifecycleCodes.lifecycleCommandOperations,
  );
  for (const transition of Object.values(
    taxonomy.lifecycleCodes.operationTransitionRegistry,
  )) {
    assertExactKeys(
      transition,
      ["entityTransitions", "transitionRule"],
      "ERR_GATE0B_LIFECYCLE_TRANSITION_OPEN",
    );
    assert.ok(Object.keys(transition.entityTransitions).length > 0);
    assert.ok(
      Object.values(transition.entityTransitions).every(
        (entityTransition) =>
          entityTransition.sourceStates.length > 0 &&
          entityTransition.targetStates.length > 0 &&
          entityTransition.targetSelectionRule.length > 0,
      ),
    );
    assert.ok(typeof transition.transitionRule === "string" && transition.transitionRule !== "");
  }
});

test("TestGate0BCanonicalJSONAndIdentityDigestDomainsAreUnambiguous", () => {
  const taxonomy = readJSON(taxonomyPath);
  const prompt = readFileSync(classificationPromptPath, "utf8");
  const specification = readFileSync(classificationSpecPath, "utf8");
  assert.deepEqual(taxonomy.digestContract.canonicalJSON, {
    profile: "AVIASURVEIL360_CANONICAL_JSON_V1",
    encoding: "UTF-8",
    objectKeyOrder: "UTF8_BYTEWISE_RECURSIVE",
    numberEncoding: "FINITE_INTEGER_MINIMAL_BASE10_NO_NEGATIVE_ZERO",
    whitespace: "NONE",
    arrays: "PRENORMALIZED_BY_FIELD_CONTRACT",
    stringEscaping:
      "JSON_QUOTE_BACKSLASH_AND_U0000_TO_U001F_ONLY_SHORT_ESCAPES_OTHERWISE_LOWERCASE_HEX",
    htmlEscaping: "DISABLED_LITERAL_UTF8_FOR_AMPERSAND_LESS_THAN_GREATER_THAN",
    solidusEscaping: "DISABLED",
    nonAsciiEncoding: "LITERAL_UTF8_NOT_U_ESCAPE",
    unicodeNormalization: "NONE_PRESERVE_SCALAR_SEQUENCE",
    loneSurrogatePolicy: "REJECT",
  });
  const conformanceVector = taxonomy.digestContract.knownAnswerVectors.find(
    (vector) => vector.name === "CANONICAL_JSON_UNICODE_AND_HTML",
  );
  assert.deepEqual(conformanceVector, {
    name: "CANONICAL_JSON_UNICODE_AND_HTML",
    domainSeparator: "AGA-CANONICAL-JSON-CONFORMANCE-V1",
    payload: {
      ampersand: "A&B<>",
      controls: "\b\t\n\f\r\u0000",
      unicode: "İstanbul—çığ",
    },
    canonicalJSON:
      '{"ampersand":"A&B<>","controls":"\\b\\t\\n\\f\\r\\u0000","unicode":"İstanbul—çığ"}',
    digest: "sha256:b4cb2a2343aa92fdb0326f5934b7bd2e6a5347fed01685f92917db1e8ec20a21",
  });
  assert.equal(canonicalJSON(conformanceVector.payload), conformanceVector.canonicalJSON);
  assert.equal(
    digestValue(conformanceVector.domainSeparator, conformanceVector.payload),
    conformanceVector.digest,
  );
  assert.equal(
    taxonomy.normalization.modelDescriptorSemanticKey,
    "RECOMPUTED_MODEL_DESCRIPTOR_DIGEST",
  );
  assert.equal(
    taxonomy.normalization.modelDescriptorSortRule,
    "ASCENDING_UTF8_BY_RECOMPUTED_MODEL_DESCRIPTOR_DIGEST",
  );
  assert.deepEqual(taxonomy.digestContract.modelDescriptorSetBinding, {
    descriptorDigestDomainSeparator: "AGA-MODEL-DESCRIPTOR-V1",
    descriptorDigestAlgorithm: "RECOMPUTE_EACH_DESCRIPTOR",
    descriptorUniqueness: "REJECT_DUPLICATE_DESCRIPTOR_DIGESTS",
    descriptorOrdering: "ASCENDING_UTF8_BY_RECOMPUTED_MODEL_DESCRIPTOR_DIGEST",
    digestArrayOrdering: "ASCENDING_UTF8",
    setEquality: "MODEL_DESCRIPTOR_DIGESTS_EXACTLY_EQUAL_RECOMPUTED_DESCRIPTOR_DIGEST_SET",
    platformAvailabilityPolicy:
      "NULL_FIELD_REQUIRES_EXACT_UNAVAILABLE_FIELD_MARKER_AND_NON_NULL_FIELD_FORBIDS_IT",
    modelIdPolicy: "DISPLAYED_MODEL_LABEL_NEVER_ESTABLISHES_EXACT_MODEL_ID",
    demoAvailabilityAcceptance:
      "TRUTHFUL_PLATFORM_UNAVAILABLE_METADATA_IS_VALID_CANDIDATE_ONLY_PROVENANCE",
  });
  assert.deepEqual(taxonomy.digestContract.identityDigestDomains, {
    validatorSignalRuleMatch: "AGA-CLASSIFICATION-BASE-IDENTITY-V1",
    exceptionIdentityInventory: "AGA-CLASSIFICATION-BASE-IDENTITY-V1",
  });
  assert.equal(
    taxonomy.digestContract.inputFactDigestRules.VALIDATOR_SIGNAL_RULE_MATCH_DIGEST
      .identityDigestDomainSeparator,
    taxonomy.digestContract.domainSeparators.baseIdentity,
  );
  assert.equal(
    taxonomy.digestContract.exceptionIdentityDigestDomainSeparator,
    taxonomy.digestContract.domainSeparators.baseIdentity,
  );
  for (const text of [prompt, specification]) {
    assert.ok(text.includes("AVIASURVEIL360_CANONICAL_JSON_V1"));
    assert.ok(text.includes("no HTML escaping"));
    assert.ok(text.includes("no Unicode normalization"));
    assert.ok(text.includes("AGA-CLASSIFICATION-BASE-IDENTITY-V1"));
  }
});

test("TestGate0BTargetLifecycleModelAndRejectionBranchesAreClosed", () => {
  const taxonomy = readJSON(taxonomyPath);
  const schema = readJSON(classificationSchemaPath);
  assert.deepEqual(taxonomy.canonicalTargetKindEligibility, {
    eligibleForV1Proposal: [
      "ORGANIZATION",
      "FACILITY",
      "DEVICE",
      "SYSTEM",
      "ASSET",
      "LOCATION",
    ],
    excludedFromV1Proposal: [
      {
        code: "PERSON",
        exclusionCode: "NO_CONTROLLED_TARGET_PROFILE_IN_V1",
      },
    ],
  });
  const dispositionedKinds = [
    ...taxonomy.canonicalTargetKindEligibility.eligibleForV1Proposal,
    ...taxonomy.canonicalTargetKindEligibility.excludedFromV1Proposal.map(({ code }) => code),
  ];
  assert.deepEqual([...dispositionedKinds].sort(), [...taxonomy.canonicalTargetKinds].sort());
  assert.ok(
    taxonomy.canonicalTargetKindEligibility.eligibleForV1Proposal.every(
      (kind) => taxonomy.targetCompatibility[kind].length > 0,
    ),
  );
  assert.deepEqual(
    schema.$defs.proposalProjection.properties.canonicalTargetKind.enum,
    taxonomy.canonicalTargetKindEligibility.eligibleForV1Proposal,
  );
  assert.deepEqual(
    schema.$defs.sealedClassificationItem.properties.canonicalTargetKind.enum,
    taxonomy.canonicalTargetKindEligibility.eligibleForV1Proposal,
  );

  const transitionRegistry = taxonomy.lifecycleCodes.operationTransitionRegistry;
  assert.equal(taxonomy.lifecycleCodes.transitionAbsenceSentinel, "ABSENT");
  for (const transition of Object.values(transitionRegistry)) {
    assert.ok(!Object.hasOwn(transition, "entities"));
    assert.ok(!Object.hasOwn(transition, "sourceStates"));
    assert.ok(!Object.hasOwn(transition, "targetStates"));
    assert.ok(Object.keys(transition.entityTransitions).length > 0);
    for (const entityTransition of Object.values(transition.entityTransitions)) {
      assertExactKeys(
        entityTransition,
        ["sourceStates", "targetStates", "targetSelectionRule"],
        "ERR_GATE0B_ENTITY_TRANSITION_OPEN",
      );
      assert.ok(entityTransition.sourceStates.length > 0);
      assert.ok(entityTransition.targetStates.length > 0);
      assert.equal(typeof entityTransition.targetSelectionRule, "string");
    }
  }
  assert.deepEqual(transitionRegistry.SUBMIT_CHECKLIST.entityTransitions, {
    DEMO_CHECKLIST: {
      sourceStates: ["IN_PROGRESS"],
      targetStates: ["SUBMITTED"],
      targetSelectionRule: "EXACT",
    },
    DEMO_INSPECTION: {
      sourceStates: ["IN_PROGRESS"],
      targetStates: ["SUBMITTED", "COMPLETED"],
      targetSelectionRule:
        "COMPLETED_IFF_ZERO_ROOTS_OR_ALL_LATEST_ROOTS_TERMINAL_ELSE_SUBMITTED",
    },
  });
  assert.deepEqual(transitionRegistry.REOPEN_CHECKLIST.entityTransitions, {
    DEMO_CHECKLIST: {
      sourceStates: ["SUBMITTED"],
      targetStates: ["IN_PROGRESS"],
      targetSelectionRule: "EXACT",
    },
    DEMO_INSPECTION: {
      sourceStates: ["SUBMITTED", "COMPLETED"],
      targetStates: ["IN_PROGRESS"],
      targetSelectionRule: "EXACT",
    },
  });
  assert.deepEqual(transitionRegistry.RETURN_POTENTIAL_FINDING.entityTransitions, {
    DEMO_POTENTIAL_FINDING: {
      sourceStates: ["PENDING_LEAD_REVIEW"],
      targetStates: ["RETURNED"],
      targetSelectionRule: "EXACT",
    },
    DEMO_CHECKLIST: {
      sourceStates: ["SUBMITTED"],
      targetStates: ["IN_PROGRESS"],
      targetSelectionRule: "EXACT",
    },
    DEMO_INSPECTION: {
      sourceStates: ["SUBMITTED", "COMPLETED"],
      targetStates: ["IN_PROGRESS"],
      targetSelectionRule: "EXACT",
    },
  });
  assert.deepEqual(transitionRegistry.DISMISS_POTENTIAL_FINDING.entityTransitions, {
    DEMO_POTENTIAL_FINDING: {
      sourceStates: ["PENDING_LEAD_REVIEW"],
      targetStates: ["DISMISSED"],
      targetSelectionRule: "EXACT",
    },
    DEMO_INSPECTION: {
      sourceStates: ["SUBMITTED"],
      targetStates: ["SUBMITTED", "COMPLETED"],
      targetSelectionRule:
        "COMPLETED_IFF_ALL_LATEST_ROOTS_TERMINAL_ELSE_SUBMITTED",
    },
  });

  const ajv = new Ajv2020({ allErrors: true, strict: false });
  ajv.addSchema(schema);
  const validateModelDescriptor = ajv.getSchema(`${schema.$id}#/$defs/modelDescriptor`);
  const modelDescriptor = {
    modelId: null,
    modelIdSource: "platform-unavailable",
    displayedModelLabel: "GPT-5.6 Pro",
    service: "ChatGPT",
    interface: "native app on a desktop computer",
    requestedReasoningEffort: null,
    forkTurns: null,
    snapshotBuildLabel: null,
    unavailableFields: ["forkTurns", "modelId", "requestedReasoningEffort", "snapshotBuildLabel"],
  };
  assert.equal(validateModelDescriptor(modelDescriptor), true);
  assert.equal(
    validateModelDescriptor({ ...modelDescriptor, unavailableFields: ["modelId"] }),
    false,
  );
  assert.equal(
    validateModelDescriptor({
      ...modelDescriptor,
      snapshotBuildLabel: "build-1",
      unavailableFields: ["snapshotBuildLabel"],
    }),
    false,
  );
  assert.equal(
    validateModelDescriptor({
      ...modelDescriptor,
      modelId: "gpt-5.6-pro-202608",
      modelIdSource: "platform-reported-exact",
      unavailableFields: ["forkTurns", "requestedReasoningEffort", "snapshotBuildLabel"],
    }),
    true,
  );
  assert.ok(schema.$defs.classificationInputRejected);
  assert.ok(
    schema.oneOf.some((entry) => entry.$ref === "#/$defs/classificationInputRejected"),
  );
  const validateRejection = ajv.getSchema(
    `${schema.$id}#/$defs/classificationInputRejected`,
  );
  assert.equal(validateRejection({ errorCode: "CLASSIFICATION_INPUT_REJECTED" }), true);
  assert.equal(
    validateRejection({
      errorCode: "CLASSIFICATION_INPUT_REJECTED",
      diagnostic: "forbidden",
    }),
    false,
  );
  assert.deepEqual(taxonomy.digestContract.taxonomySelfDigest, {
    domainSeparator: "AGA-QUESTION-CLASSIFICATION-TAXONOMY-V1",
    preimage: "COMPLETE_TAXONOMY_OBJECT_EXCLUDING_ONLY_TAXONOMY_DIGEST",
    excludedFields: ["taxonomyDigest"],
  });
});

test("TestGate0BInputDigestRolesAndReturnedRootPathAreTotal", () => {
  const taxonomy = readJSON(taxonomyPath);
  const schema = readJSON(classificationSchemaPath);
  assert.deepEqual(taxonomy.digestContract.inputDigestFieldSemantics, {
    passProposalRecord: {
      field: "inputDigest",
      digestRole: "PASS_BATCH_INPUT_DIGEST",
      domainSeparator: "AGA-CLASSIFICATION-PASS-INPUT-V1",
      preimageContract: "classificationPassInputPreimage",
      cardinality: "ONE_PER_PASS_BATCH_SHARED_BY_ALL_BATCH_RECORDS",
    },
    passBatchOutput: {
      field: "inputDigest",
      digestRole: "PASS_BATCH_INPUT_DIGEST",
      domainSeparator: "AGA-CLASSIFICATION-PASS-INPUT-V1",
      preimageContract: "classificationPassInputPreimage",
      cardinality: "ONE_PER_PASS_BATCH",
    },
    passSealReceipt: {
      field: "orderedInputDigests",
      digestRole: "ORDERED_PASS_BATCH_INPUT_DIGESTS",
      domainSeparator: "AGA-CLASSIFICATION-PASS-INPUT-V1",
      preimageContract: "classificationPassInputPreimage",
      cardinality: "EXACTLY_25_IN_BATCH_ORDINAL_ORDER",
    },
    sealedClassificationItem: {
      field: "inputDigest",
      digestRole: "CLASSIFICATION_RUN_INPUT_DIGEST",
      domainSeparator: "AGA-CLASSIFICATION-RUN-INPUT-V1",
      preimageContract: "runInputPayloadFields",
      cardinality: "ONE_COMMON_DIGEST_FOR_ALL_1310_ITEMS",
    },
    classificationRunReceipt: {
      field: "inputDigest",
      digestRole: "CLASSIFICATION_RUN_INPUT_DIGEST",
      domainSeparator: "AGA-CLASSIFICATION-RUN-INPUT-V1",
      preimageContract: "runInputPayloadFields",
      cardinality: "ONE_PER_CLASSIFICATION_RUN",
    },
    loadingClassificationRunRecord: {
      field: "inputDigest",
      digestRole: "CLASSIFICATION_RUN_INPUT_DIGEST",
      domainSeparator: "AGA-CLASSIFICATION-RUN-INPUT-V1",
      preimageContract: "runInputPayloadFields",
      cardinality: "ONE_PER_CLASSIFICATION_RUN",
    },
    rejectedClassificationRunRecord: {
      field: "inputDigest",
      digestRole: "CLASSIFICATION_RUN_INPUT_DIGEST",
      domainSeparator: "AGA-CLASSIFICATION-RUN-INPUT-V1",
      preimageContract: "runInputPayloadFields",
      cardinality: "ONE_PER_CLASSIFICATION_RUN",
    },
  });
  assert.deepEqual(schema.$defs.passInputDigest, {
    type: "string",
    pattern: "^sha256:[a-f0-9]{64}$",
    "x-digest-role": "PASS_BATCH_INPUT_DIGEST",
    "x-digest-domain": "AGA-CLASSIFICATION-PASS-INPUT-V1",
    "x-preimage-contract": "classificationPassInputPreimage",
  });
  assert.deepEqual(schema.$defs.runInputDigest, {
    type: "string",
    pattern: "^sha256:[a-f0-9]{64}$",
    "x-digest-role": "CLASSIFICATION_RUN_INPUT_DIGEST",
    "x-digest-domain": "AGA-CLASSIFICATION-RUN-INPUT-V1",
    "x-preimage-contract": "runInputPayloadFields",
  });
  for (const definition of ["passProposalRecord", "passBatchOutput"]) {
    assert.equal(
      schema.$defs[definition].properties.inputDigest.$ref,
      "#/$defs/passInputDigest",
    );
  }
  assert.equal(
    schema.$defs.passSealReceipt.properties.orderedInputDigests.items.$ref,
    "#/$defs/passInputDigest",
  );
  for (const definition of [
    "sealedClassificationItem",
    "classificationRunReceipt",
    "loadingClassificationRunRecord",
    "rejectedClassificationRunRecord",
  ]) {
    assert.equal(
      schema.$defs[definition].properties.inputDigest.$ref,
      "#/$defs/runInputDigest",
    );
  }

  const lifecycle = taxonomy.lifecycleCodes;
  assert.deepEqual(lifecycle.returnedPotentialFindingSuccessorContract, {
    operation: "CREATE_POTENTIAL_FINDING",
    sourceState: "RETURNED",
    targetState: "PENDING_LEAD_REVIEW",
    correctedResponseRequired: true,
    correctedResponseStates: ["NON_COMPLIANT", "OBSERVATION"],
    returnedVersionResponseBindingFields: ["responseRevision", "responseSemanticDigest"],
    correctedResponseRevisionRule:
      "STRICTLY_GREATER_THAN_RETURNED_BOUND_RESPONSE_REVISION",
    correctedResponseDigestRule:
      "MUST_DIFFER_FROM_RETURNED_BOUND_RESPONSE_SEMANTIC_DIGEST",
    successorResponseBindingRule:
      "EXACT_CURRENT_CORRECTED_RESPONSE_REVISION_AND_SEMANTIC_DIGEST",
    rootIdentity: "PRESERVE_EXISTING_ROOT_ID",
    successorVersion: "ALLOCATE_NEXT_IMMUTABLE_VERSION",
    priorVersion: "RETAIN_IMMUTABLE_RETURNED_VERSION",
    checklistState: "IN_PROGRESS",
    inspectionState: "IN_PROGRESS",
    resubmissionOperation: "SUBMIT_CHECKLIST",
    submissionGuard: {
      denyWhenAnyLatestRootState: ["RETURNED"],
      denialCode: "RETURNED_ROOT_SUCCESSOR_REQUIRED",
      allowAfterSuccessorLatestState: "PENDING_LEAD_REVIEW",
    },
  });
  assert.deepEqual(
    lifecycle.operationTransitionRegistry.CREATE_POTENTIAL_FINDING.entityTransitions
      .DEMO_POTENTIAL_FINDING,
    {
      sourceStates: ["ABSENT", "RETURNED"],
      targetStates: ["PENDING_LEAD_REVIEW"],
      targetSelectionRule:
        "ABSENT_ALLOCATES_NEW_ROOT_RETURNED_APPENDS_SAME_ROOT_IMMUTABLE_SUCCESSOR",
    },
  );
  assert.ok(
    lifecycle.operationTransitionRegistry.RETURN_POTENTIAL_FINDING.entityTransitions
      .DEMO_POTENTIAL_FINDING.targetStates.includes("RETURNED"),
  );
  assert.ok(
    lifecycle.operationTransitionRegistry.SUBMIT_CHECKLIST.entityTransitions
      .DEMO_CHECKLIST.sourceStates.includes("IN_PROGRESS"),
  );
  assert.ok(
    lifecycle.lifecycleCommandOperations.includes(
      lifecycle.returnedPotentialFindingSuccessorContract.operation,
    ),
  );
  assert.ok(
    lifecycle.lifecycleCommandOperations.includes(
      lifecycle.returnedPotentialFindingSuccessorContract.resubmissionOperation,
    ),
  );
  assert.equal(
    lifecycle.operationTransitionRegistry.SUBMIT_CHECKLIST.transitionRule,
    "ASSIGNED_INSPECTOR_TERMINAL_ROOT_RECOMPUTATION_DENY_LATEST_RETURNED_ROOTS",
  );
  const successorContract = lifecycle.returnedPotentialFindingSuccessorContract;
  const canAppendReturnedRootSuccessor = ({
    returnedResponseRevision,
    returnedResponseDigest,
    currentResponseRevision,
    currentResponseDigest,
    successorResponseRevision,
    successorResponseDigest,
  }) =>
    currentResponseRevision > returnedResponseRevision &&
    currentResponseDigest !== returnedResponseDigest &&
    successorResponseRevision === currentResponseRevision &&
    successorResponseDigest === currentResponseDigest;
  const digestA = `sha256:${"a".repeat(64)}`;
  const digestB = `sha256:${"b".repeat(64)}`;
  assert.equal(
    canAppendReturnedRootSuccessor({
      returnedResponseRevision: 4,
      returnedResponseDigest: digestA,
      currentResponseRevision: 5,
      currentResponseDigest: digestB,
      successorResponseRevision: 5,
      successorResponseDigest: digestB,
    }),
    true,
  );
  assert.equal(
    canAppendReturnedRootSuccessor({
      returnedResponseRevision: 4,
      returnedResponseDigest: digestA,
      currentResponseRevision: 4,
      currentResponseDigest: digestA,
      successorResponseRevision: 4,
      successorResponseDigest: digestA,
    }),
    false,
    "ERR_GATE0B_RETURNED_ROOT_PRE_RETURN_RESPONSE_REUSE",
  );
  assert.equal(
    canAppendReturnedRootSuccessor({
      returnedResponseRevision: 4,
      returnedResponseDigest: digestA,
      currentResponseRevision: 5,
      currentResponseDigest: digestB,
      successorResponseRevision: 4,
      successorResponseDigest: digestA,
    }),
    false,
    "ERR_GATE0B_RETURNED_ROOT_SUCCESSOR_BINDING_MISMATCH",
  );
  const canSubmitChecklist = (latestRootStates) =>
    !latestRootStates.some((state) =>
      successorContract.submissionGuard.denyWhenAnyLatestRootState.includes(state),
    );
  assert.equal(
    canSubmitChecklist(["RETURNED", "CONVERTED"]),
    false,
    "ERR_GATE0B_RETURNED_ROOT_PREMATURE_SUBMISSION",
  );
  assert.equal(canSubmitChecklist(["PENDING_LEAD_REVIEW", "CONVERTED"]), true);
});

test("TestGate0BNormalizationKnownAnswers", () => {
  const taxonomy = readJSON(taxonomyPath);
  const digestA = `sha256:${"a".repeat(64)}`;
  const digestB = `sha256:${"b".repeat(64)}`;
  const evidenceA = {
    proposalField: "mainDomainCode",
    proposalValueDigest: digestA,
    rationaleCode: "GOVERNANCE_CUE",
    inputFactSelector: "QUESTION_BODY_DIGEST",
    inputFactValueDigest: digestA,
  };
  const evidenceB = {
    proposalField: "topicCodes",
    proposalValueDigest: digestB,
    rationaleCode: "OPERATIONAL_SAFETY_CUE",
    inputFactSelector: "FORM_METADATA_DIGEST",
    inputFactValueDigest: digestB,
  };
  const normalizedEvidence = normalizeSetLikeByTuple(
    [evidenceB, evidenceA],
    taxonomy.normalization.evidenceSortFields,
  );
  assert.deepEqual(
    normalizedEvidence,
    normalizeSetLikeByTuple(
      [evidenceA, evidenceB],
      taxonomy.normalization.evidenceSortFields,
    ),
  );
  assert.throws(
    () => normalizeSetLikeByTuple(
      [evidenceA, { ...evidenceA }],
      taxonomy.normalization.evidenceSortFields,
    ),
    /ERR_DUPLICATE_SEMANTIC_KEY/u,
  );
  const edgeA = {
    providerTypeCode: "ANSP",
    involvementRoleCode: "COORDINATION",
    conditionCode: "ANSP_COORDINATION_REQUIRED",
    applicabilityDisposition: "CONDITIONAL_ON_SERVICE_ARRANGEMENT",
    rationaleCodes: ["EXTERNAL_INTERFACE_CUE"],
  };
  assert.throws(
    () => normalizeSetLikeByTuple(
      [edgeA, { ...edgeA, rationaleCodes: ["OPERATIONAL_SAFETY_CUE"] }],
      taxonomy.normalization.externalInvolvementSortFields,
    ),
    /ERR_DUPLICATE_SEMANTIC_KEY/u,
  );
  assert.throws(
    () => normalizeSetLikeByTuple(
      [
        { key: "OPERATION_STATUS", value: "ACTIVE" },
        { key: "OPERATION_STATUS", value: "CLOSED" },
      ],
      taxonomy.normalization.qualifierSemanticKeyFields,
    ),
    /ERR_DUPLICATE_SEMANTIC_KEY/u,
  );
  const sourceA = { kind: "PACKAGE_SOURCE_REFERENCE", referenceDigest: digestA };
  const sourceB = { kind: "RESEARCH_ROW", referenceDigest: digestB };
  assert.deepEqual(
    normalizeSetLikeByTuple(
      [sourceB, sourceA],
      taxonomy.normalization.sourceReferenceSortFields,
    ),
    normalizeSetLikeByTuple(
      [sourceA, sourceB],
      taxonomy.normalization.sourceReferenceSortFields,
    ),
  );
  assert.notEqual(
    digestValue("AGA-ORDERED-KNOWN-ANSWER-V1", [digestA, digestB]),
    digestValue("AGA-ORDERED-KNOWN-ANSWER-V1", [digestB, digestA]),
    "ERR_GATE0B_ORDERED_ARRAY_WAS_NORMALIZED_AS_SET",
  );
});

test("TestGate0BPlannedCreateInventoryIsCompleteAndScoped", () => {
  assert.ok(existsSync(createdFileCheckerPath), "ERR_GATE0B_CREATED_FILE_CHECKER_MISSING");
  assert.ok(existsSync(createdFileInventoryPath), "ERR_GATE0B_CREATED_FILE_INVENTORY_MISSING");
  assert.equal(throughOwnerSlice("gate0"), "gate0");
  assert.equal(throughOwnerSlice("task10"), "slice-f");
  const inventory = readJSON(createdFileInventoryPath);
  assert.equal(inventory.schemaVersion, "aga-hybrid-created-file-inventory/v1");
  assert.equal(
    inventory.planPath,
    "docs/exec-plans/active/2026-08-03-aga-hybrid-classification-demo-lifecycle-plan.md",
  );
  assert.ok(Array.isArray(inventory.entries));
  assert.equal(
    new Set(inventory.entries.map((entry) => entry.path)).size,
    inventory.entries.length,
    "ERR_GATE0B_CREATED_FILE_INVENTORY_DUPLICATE",
  );
  assert.ok(inventory.entries.length >= 80, "ERR_GATE0B_CREATED_FILE_INVENTORY_INCOMPLETE");
  assert.ok(
    inventory.entries.every(
      (entry) =>
        /^[a-zA-Z0-9._/-]+$/u.test(entry.path) &&
        !entry.path.startsWith("/") &&
        !entry.path.includes("..") &&
        ["gate0", "slice-b", "slice-c", "slice-d", "slice-e", "slice-f"].includes(
          entry.ownerSlice,
        ),
    ),
    "ERR_GATE0B_CREATED_FILE_INVENTORY_OUT_OF_SCOPE",
  );
  for (const required of [
    taxonomyPath,
    classificationSchemaPath,
    classificationPromptPath,
    "apps/api/internal/agaapplicability/types.go",
    "apps/api/cmd/aga-question-classification-validator/validator_test.go",
    "apps/api/internal/preproddata/agademoworkspace/postgres_store_test.go",
    "api/openapi/tests/aga-demo-workspace-contract.test.mjs",
    "apps/web/tests/e2e/aga-hybrid-privacy.http.spec.ts",
    "scripts/test-aga-hybrid-demo-workspace-connected.sh",
    "docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_DEMO_LIFECYCLE_2026-08-03.md",
  ]) {
    assert.ok(
      inventory.entries.some((entry) => entry.path === required),
      "ERR_GATE0B_CREATED_FILE_INVENTORY_REQUIRED_PATH_MISSING",
    );
  }
});
