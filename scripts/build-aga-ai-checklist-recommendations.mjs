import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const approvedPackagePath = path.join(repositoryRoot, "deliverables", "AGA_ALL_FORMS_APPROVED_SOURCE_V2.zip");
const classificationDirectory = path.join(repositoryRoot, "deliverables", "aga-question-classification-candidate-2026-08-03");
const classificationManifestPath = path.join(classificationDirectory, "manifest.json");
const classificationPath = path.join(repositoryRoot, "deliverables", "aga-question-classification-candidate-2026-08-03", "question-classifications.json");
const taxonomyPath = path.join(repositoryRoot, "docs", "product-specs", "data-and-rules", "aga-question-classification-taxonomy.v1.json");
const outputDirectory = path.join(repositoryRoot, "deliverables", "aga-ai-checklist-recommendations-v1");
const outputPath = path.join(outputDirectory, "AGA_AI_CHECKLIST_RECOMMENDATIONS_V1.json");
const outputManifestPath = path.join(outputDirectory, "MANIFEST.sha256");

const APPROVED_PACKAGE_JSON_ENTRY = "aga-approved-source-v2/AGA_ALL_FORMS_APPROVED_SOURCE_V2.json";
const EXPECTED_CATALOG_ROOT_DIGEST = "sha256:972f06005ba7befecb480d477334ea8cee542555d39d2604607f082deeee6e48";
const EXPECTED_PACKAGE_JSON_SHA256 = "sha256:57abb7b87ce91dc7383fb3b24426ccd542811ce979f33c6b17bcf938c3907973";
const EXPECTED_SOURCE_MANIFEST_SHA256 = "sha256:53679bd6eccb77b2d4bf1c909cb16a3b925ca1433584d206daeaf212031877f8";
const EXPECTED_QUESTION_COUNT = 1310;

const RISK_POLICY = Object.freeze({
  PROPOSED_SAFETY_CRITICAL: Object.freeze({ tier: "HIGH", recurrenceMonths: 12, defaultBucket: "SUGGESTED_NOW" }),
  PROPOSED_HIGH_OPERATIONAL: Object.freeze({ tier: "MEDIUM", recurrenceMonths: 18, defaultBucket: "MATCHING_OPTIONAL" }),
  PROPOSED_CONTROL_ASSURANCE: Object.freeze({ tier: "LOW", recurrenceMonths: 24, defaultBucket: "MATCHING_OPTIONAL" }),
  PROPOSED_REVIEW_REQUIRED: Object.freeze({ tier: "UNKNOWN", recurrenceMonths: 12, defaultBucket: "UNCERTAIN_SIGNAL" }),
});

const ADVISORY_STATES = new Set(["SUGGESTED_NOW", "MATCHING_OPTIONAL", "RECENTLY_VERIFIED", "OUTSIDE_FOCUS", "UNCERTAIN_SIGNAL"]);
const ADVISORY_BUCKETS = new Set(["SUGGESTED_NOW", "MATCHING_OPTIONAL", "UNCERTAIN_SIGNAL"]);

function sha256Bytes(bytes) {
  return `sha256:${crypto.createHash("sha256").update(bytes).digest("hex")}`;
}

function canonicalize(value) {
  if (Array.isArray(value)) return value.map(canonicalize);
  if (value && typeof value === "object") {
    return Object.fromEntries(Object.entries(value).sort(([left], [right]) => Buffer.from(left).compare(Buffer.from(right))).map(([key, child]) => [key, canonicalize(child)]));
  }
  return value;
}

function canonicalJson(value) {
  return `${JSON.stringify(canonicalize(value))}\n`;
}

function canonicalJsonWithoutDelimiter(value) {
  return JSON.stringify(canonicalize(value));
}

function digestWithout(value, field) {
  const copy = { ...value };
  delete copy[field];
  return sha256Bytes(Buffer.from(canonicalJson(copy), "utf8"));
}

function readApprovedPackage() {
  const zipBytes = fs.readFileSync(approvedPackagePath);
  const packageJsonBytes = execFileSync("unzip", ["-p", approvedPackagePath, APPROVED_PACKAGE_JSON_ENTRY], { maxBuffer: 16 * 1024 * 1024 });
  if (sha256Bytes(packageJsonBytes) !== EXPECTED_PACKAGE_JSON_SHA256) throw new Error("approved source package JSON digest changed");
  const packageJson = JSON.parse(packageJsonBytes.toString("utf8"));
  if (packageJson.packageVersion !== "AGA_ALL_FORMS_APPROVED_SOURCE_V2" || packageJson.catalogRootDigest !== EXPECTED_CATALOG_ROOT_DIGEST || packageJson.sourceManifestSha256 !== EXPECTED_SOURCE_MANIFEST_SHA256) {
    throw new Error("approved source package identity changed");
  }
  const questions = packageJson.forms.flatMap((form) => form.questions.map((question) => ({
    formCode: form.formCode,
    ordinal: question.ordinal,
    proposalId: question.proposalId,
    questionVersionId: question.immutableQuestionVersionId,
    questionDigest: question.textDigest,
    sourceLocator: question.sourceLocator,
    optionalEnrichment: question.optionalEnrichment ?? {},
  })));
  if (packageJson.totals?.questions !== EXPECTED_QUESTION_COUNT || questions.length !== EXPECTED_QUESTION_COUNT) throw new Error("approved source package question count changed");
  return {
    packageVersion: packageJson.packageVersion,
    packageJsonSha256: sha256Bytes(packageJsonBytes),
    packageZipSha256: sha256Bytes(zipBytes),
    sourceManifestSha256: packageJson.sourceManifestSha256,
    catalogRootDigest: packageJson.catalogRootDigest,
    questions,
  };
}

function readClassification() {
  const manifest = JSON.parse(fs.readFileSync(classificationManifestPath, "utf8"));
  const classification = JSON.parse(fs.readFileSync(classificationPath, "utf8"));
  const classificationManifestEntry = manifest.files?.find((entry) => entry.filename === "question-classifications.json");
  if (!classificationManifestEntry || classificationManifestEntry.sha256 !== sha256Bytes(fs.readFileSync(classificationPath))) {
    throw new Error("offline AI classification file digest does not match its manifest");
  }
  if (manifest.schemaVersion !== "aga-hybrid-classification-candidate/v1" || manifest.status !== "SEALED" || manifest.packageVersion !== "AGA_ALL_FORMS_SOURCE_RISK_DRAFT_V1" || manifest.itemCount !== EXPECTED_QUESTION_COUNT || !manifest.classificationRunDigest || !manifest.taxonomyDigest || !manifest.inputDigest) {
    throw new Error("offline AI classification manifest is not the expected sealed run");
  }
  if (classification.schemaVersion !== "aga-hybrid-classification-question-classifications/v1" || classification.status !== "SEALED" || classification.itemCount !== EXPECTED_QUESTION_COUNT || classification.items.length !== EXPECTED_QUESTION_COUNT) {
    throw new Error("offline AI classification artifact is not the expected sealed 1,310-item run");
  }
  if (classification.classificationRunId !== manifest.classificationRunId || classification.promptDigest !== manifest.promptDigest) {
    throw new Error("offline AI classification manifest and item file identity drifted");
  }
  return { manifest, classification };
}

function identityKey(value) {
  return `${value.formCode}\u0000${value.proposalId}\u0000${value.ordinal}\u0000${value.questionDigest ?? value.textDigest}`;
}

function buildItems(source, classification) {
  const byIdentity = new Map();
  for (const item of classification.items) {
    const key = identityKey(item);
    if (byIdentity.has(key)) throw new Error(`classification contains duplicate identity ${key}`);
    byIdentity.set(key, item);
  }
  const seen = new Set();
  const items = source.questions.map((question) => {
    const item = byIdentity.get(identityKey(question));
    if (!item) throw new Error(`classification is missing approved question ${question.formCode}/${question.ordinal}`);
    if (item.textDigest !== question.questionDigest) throw new Error(`classification text digest mismatch for ${question.formCode}/${question.ordinal}`);
    if (seen.has(question.questionVersionId)) throw new Error(`duplicate approved question version ${question.questionVersionId}`);
    seen.add(question.questionVersionId);

    const proposedRisk = question.optionalEnrichment.proposedRisk ?? {};
    const riskBand = RISK_POLICY[proposedRisk.band] ? proposedRisk.band : "PROPOSED_REVIEW_REQUIRED";
    const riskPolicy = RISK_POLICY[riskBand];
    const inspectionTypeCodes = [...new Set(item.inspectionTypeCodes ?? [])].sort();
    const inspectionProfileCodes = [...new Set(item.inspectionProfileCodes ?? [])].sort();
    const topicCodes = [...new Set(item.topicCodes ?? [])].sort();
    const rationaleCodes = [...new Set(item.rationaleCodes ?? [])].sort();
    return {
      questionVersionId: question.questionVersionId,
      formCode: question.formCode,
      proposalId: question.proposalId,
      ordinal: question.ordinal,
      questionDigest: question.questionDigest,
      sourceLocator: question.sourceLocator,
      domainCode: item.mainDomainCode,
      topicCodes,
      inspectionTypeCodes,
      inspectionProfileCodes,
      applicabilityDisposition: item.applicabilityDisposition,
      riskBand,
      riskTier: riskPolicy.tier,
      safetyCritical: proposedRisk.safetyCritical === true || riskBand === "PROPOSED_SAFETY_CRITICAL",
      agreementConfidence: item.agreementConfidence,
      sourceClassificationRecommendationState: item.recommendationState,
      advisoryState: riskPolicy.defaultBucket,
      defaultRecommendationBucket: riskPolicy.defaultBucket,
      recurrenceMonths: riskPolicy.recurrenceMonths,
      rationaleCodes,
      externalApplicabilityUnresolved: item.externalApplicabilityUnresolved === true,
      riskSignalSource: proposedRisk.band ? "APPROVED_SOURCE_OPTIONAL_SIGNAL_RECONCILED_OFFLINE" : "OFFLINE_CLASSIFICATION_UNCERTAIN",
    };
  });
  if (byIdentity.size !== EXPECTED_QUESTION_COUNT || items.length !== EXPECTED_QUESTION_COUNT || seen.size !== EXPECTED_QUESTION_COUNT) throw new Error("AI recommendation item bijection is incomplete");
  return items;
}

export function buildArtifact() {
  const source = readApprovedPackage();
  const { manifest, classification } = readClassification();
  const taxonomy = JSON.parse(fs.readFileSync(taxonomyPath, "utf8"));
  const taxonomyForDigest = { ...taxonomy };
  delete taxonomyForDigest.taxonomyDigest;
  const taxonomyDigest = sha256Bytes(Buffer.from(`AGA-QUESTION-CLASSIFICATION-TAXONOMY-V1${canonicalJsonWithoutDelimiter(taxonomyForDigest)}`, "utf8"));
  if (taxonomy.taxonomyVersion !== manifest.taxonomyVersion || taxonomyDigest !== manifest.taxonomyDigest) {
    throw new Error("classification taxonomy digest does not match the sealed manifest");
  }
  const allowedDomains = new Set(taxonomy.mainDomainCodes);
  const allowedTopics = new Set(taxonomy.topicCodes);
  const allowedInspectionTypes = new Set(taxonomy.inspectionTypeCodes);
  const allowedInspectionProfiles = new Set(taxonomy.inspectionProfileCodes);
  for (const item of classification.items) {
    if (!allowedDomains.has(item.mainDomainCode) || item.topicCodes.some((code) => !allowedTopics.has(code)) || item.inspectionTypeCodes.some((code) => !allowedInspectionTypes.has(code)) || item.inspectionProfileCodes.some((code) => !allowedInspectionProfiles.has(code))) {
      throw new Error(`classification contains an unknown taxonomy code for ${item.formCode}/${item.ordinal}`);
    }
  }
  const items = buildItems(source, classification);
  const artifact = {
    schemaVersion: "aga-ai-checklist-recommendations/v1",
    status: "SEALED",
    advisoryOnly: true,
    generatedBy: "CODEX_OFFLINE_AI",
    sourceCatalog: {
      catalogVersion: "aga-approved-source@2.0.0",
      packageVersion: source.packageVersion,
      packageZipSha256: source.packageZipSha256,
      packageJsonSha256: source.packageJsonSha256,
      sourceManifestSha256: source.sourceManifestSha256,
      catalogRootDigest: source.catalogRootDigest,
      questionCount: source.questions.length,
    },
    modelRun: {
      classificationRunId: manifest.classificationRunId,
      classificationRunDigest: manifest.classificationRunDigest,
      promptDigest: manifest.promptDigest,
      taxonomyVersion: manifest.taxonomyVersion,
      taxonomyDigest: manifest.taxonomyDigest,
      inputDigest: manifest.inputDigest,
      modelDescriptorDigests: [...new Set(classification.items.flatMap((item) => item.modelDescriptorDigests ?? []))].sort(),
    },
    recommendationPolicy: {
      version: "AGA_AI_RECOMMENDATION_POLICY_V1",
      unknownRiskNeverSuppressed: true,
      priorAuditEvidenceRequiredForDeferral: true,
      runtimeModelCalls: false,
      riskBands: RISK_POLICY,
      explanation: "Offline AI advisory classification is reconciled with the source risk signal before import. Runtime recommendations use this sealed artifact and immutable audit history only; every suggestion remains optional.",
    },
    items,
  };
  artifact.itemCount = items.length;
  artifact.artifactDigest = digestWithout(artifact, "artifactDigest");
  return artifact;
}

export function validateArtifact(artifact) {
  if (!artifact || artifact.schemaVersion !== "aga-ai-checklist-recommendations/v1" || artifact.status !== "SEALED" || artifact.advisoryOnly !== true || artifact.generatedBy !== "CODEX_OFFLINE_AI") throw new Error("AI recommendation artifact envelope is invalid");
  if (artifact.sourceCatalog?.catalogRootDigest !== EXPECTED_CATALOG_ROOT_DIGEST || artifact.sourceCatalog?.packageJsonSha256 !== EXPECTED_PACKAGE_JSON_SHA256 || artifact.sourceCatalog?.sourceManifestSha256 !== EXPECTED_SOURCE_MANIFEST_SHA256 || artifact.sourceCatalog?.questionCount !== EXPECTED_QUESTION_COUNT) throw new Error("AI recommendation source binding is invalid");
  if (!Array.isArray(artifact.items) || artifact.itemCount !== EXPECTED_QUESTION_COUNT || artifact.items.length !== EXPECTED_QUESTION_COUNT) throw new Error("AI recommendation count is invalid");
  const ids = new Set();
  for (const item of artifact.items) {
    if (!item.questionVersionId || ids.has(item.questionVersionId) || !RISK_POLICY[item.riskBand] || !["HIGH", "MEDIUM", "LOW", "UNKNOWN"].includes(item.riskTier) || !Array.isArray(item.topicCodes) || !Array.isArray(item.inspectionTypeCodes) || !Array.isArray(item.inspectionProfileCodes) || !ADVISORY_STATES.has(item.advisoryState) || !ADVISORY_BUCKETS.has(item.defaultRecommendationBucket) || !item.riskSignalSource) throw new Error("AI recommendation item identity or controlled values are invalid");
    ids.add(item.questionVersionId);
  }
  if (artifact.artifactDigest !== digestWithout(artifact, "artifactDigest")) throw new Error("AI recommendation artifact digest mismatch");
  return artifact;
}

export function writeArtifact() {
  const artifact = validateArtifact(buildArtifact());
  fs.mkdirSync(outputDirectory, { recursive: true });
  const artifactText = canonicalJson(artifact);
  fs.writeFileSync(outputPath, artifactText, { mode: 0o644 });
  const artifactDigest = sha256Bytes(Buffer.from(artifactText, "utf8"));
  fs.writeFileSync(outputManifestPath, `${artifactDigest.slice("sha256:".length)}  ${path.basename(outputPath)}\n`, { mode: 0o644 });
  return { outputPath, outputManifestPath, itemCount: artifact.itemCount, artifactDigest: artifact.artifactDigest, fileDigest: artifactDigest };
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  process.stdout.write(`${JSON.stringify(writeArtifact())}\n`);
}
