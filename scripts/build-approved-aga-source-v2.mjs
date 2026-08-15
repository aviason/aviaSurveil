#!/usr/bin/env node

/**
 * Build the approved AGA source package from the frozen V1 package.
 *
 * The V1 archive is a historical input and is never rewritten.  This builder
 * creates a new, deterministic package whose operational authority is the
 * project-provided direct-Aviation classification.  Parser suggestions are
 * retained only under explicitly optional, not-approved enrichment fields.
 */

import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { mkdtempSync, mkdirSync, readFileSync, rmSync, utimesSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";

const surveilRoot = resolve(new URL("..", import.meta.url).pathname);
const inputZip = join(surveilRoot, "deliverables", "AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip");
const outputZip = join(surveilRoot, "deliverables", "AGA_ALL_FORMS_APPROVED_SOURCE_V2.zip");
const jsonEntry = "aga-all-forms-source-risk-draft-2026-08-01/AGA_ALL_FORMS_SOURCE_RISK_DRAFT.json";
const registerEntry = "aga-all-forms-source-risk-draft-2026-08-01/AGA_ALL_FORMS_REGISTER.csv";

const ORIGINAL_SOURCE_SHA256 = "sha256:dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32";
const V1_ZIP_SHA256 = "sha256:30700a88aeb5b26514bf7eb76bef050deb08b96294db94117d185de5c9f163b2";
const V1_JSON_SHA256 = "sha256:5ebcce2d70ee22fef4165b490cb6e4b276ad776f40dbaf12e5cea85c9da91b15";
const V1_REGISTER_SHA256 = "sha256:29ed8384693b615926fc42a0ca4654be2ea9a36b0946f217975571ca0ad9564f";
const V1_DERIVED_REGISTER_SHA256 = "sha256:e9f26bf8085de8851f89877b878f61932d45b4b7069c7083efe00e0cd42cb6cf";
const FIXED_TIME = new Date("2000-01-01T00:00:00.000Z");

function sha256(bytes) {
  return `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
}

function sha256Text(value) {
  return sha256(Buffer.from(value, "utf8"));
}

function unzipEntry(entry) {
  return execFileSync("unzip", ["-p", inputZip, entry], { maxBuffer: 64 * 1024 * 1024 });
}

function canonicalJson(value) {
  return `${JSON.stringify(value, null, 2)}\n`;
}

function required(condition, message) {
  if (!condition) throw new Error(message);
}

function normalizeQuestion(form, question) {
  required(Number.isInteger(question.ordinal) && question.ordinal > 0, `invalid ordinal in ${form.formCode}`);
  required(typeof question.proposalId === "string" && question.proposalId.trim() !== "", `missing proposal id in ${form.formCode}`);
  required(typeof question.originalText === "string" && question.originalText.trim() !== "", `missing question text in ${form.formCode}/${question.proposalId}`);
  required(question.textDigest === sha256Text(question.originalText), `text digest mismatch in ${form.formCode}/${question.proposalId}`);
  const questionId = `aga:${form.formCode}:${question.proposalId}`;
  const questionVersionId = `qv:aga-approved-source-v2:${form.formCode}:${question.proposalId}`;
  return {
    immutableQuestionId: questionId,
    immutableQuestionVersionId: questionVersionId,
    version: 1,
    ordinal: question.ordinal,
    protocolCode: question.protocolCode ?? null,
    page: question.page ?? null,
    text: question.originalText,
    textDigest: question.textDigest,
    sourceLocator: question.sourceLocator,
    proposalId: question.proposalId,
    optionalEnrichment: {
      status: "OPTIONAL_NOT_APPROVED",
      sourceMappingState: "OPTIONAL_ENRICHMENT_NOT_PROVIDED",
      proposedSourceRefs: Array.isArray(question.sourceRefs) ? question.sourceRefs : [],
      proposedSourceProposals: Array.isArray(question.sourceProposals) ? question.sourceProposals : [],
      proposedRisk: question.proposedRisk ?? null,
      riskClassificationState: "OPTIONAL_ENRICHMENT_NOT_PROVIDED",
    },
  };
}

function build() {
  const inputZipBytes = readFileSync(inputZip);
  required(sha256(inputZipBytes) === V1_ZIP_SHA256, "historical V1 ZIP digest changed");
  const v1JsonBytes = unzipEntry(jsonEntry);
  required(sha256(v1JsonBytes) === V1_JSON_SHA256, "historical V1 JSON digest changed");
  const registerBytes = unzipEntry(registerEntry);
  required(sha256(registerBytes) === V1_DERIVED_REGISTER_SHA256, "historical V1 derived register changed");
  const v1 = JSON.parse(v1JsonBytes.toString("utf8"));
  required(v1.packageVersion === "AGA_ALL_FORMS_SOURCE_RISK_DRAFT_V1", "unexpected V1 package version");
  required(v1.status === "PENDING_ADMIN_AND_SOURCE_OWNER_REVIEW", "unexpected V1 package status");
  required(v1.archive?.registerSha256 === V1_REGISTER_SHA256, "historical V1 source register provenance changed");
  required(v1.totals?.forms === 52 && v1.totals?.questions === 1310, "unexpected V1 totals");
  required(Array.isArray(v1.forms) && v1.forms.length === 52, "V1 must contain exactly 52 forms");

  const forms = [];
  const orderedRows = [];
  const questionIds = new Set();
  const questionVersionIds = new Set();
  let questionCount = 0;
  let boundaryForms = 0;
  for (const form of v1.forms) {
    required(typeof form.formCode === "string" && form.formCode !== "", "form code is required");
    required(Array.isArray(form.questions), `questions missing for ${form.formCode}`);
    if (form.questions.length > 0) boundaryForms += 1;
    const questions = form.questions.map((question) => normalizeQuestion(form, question));
    const ordinals = questions.map((question) => question.ordinal);
    required(ordinals.every((value, index) => value === index + 1), `question ordering changed in ${form.formCode}`);
    for (const question of questions) {
      required(!questionIds.has(question.immutableQuestionId), `duplicate immutable question id ${question.immutableQuestionId}`);
      required(!questionVersionIds.has(question.immutableQuestionVersionId), `duplicate immutable question version ${question.immutableQuestionVersionId}`);
      questionIds.add(question.immutableQuestionId);
      questionVersionIds.add(question.immutableQuestionVersionId);
      orderedRows.push({
        formCode: form.formCode,
        ordinal: question.ordinal,
        questionId: question.immutableQuestionId,
        questionVersionId: question.immutableQuestionVersionId,
        text: question.text,
        textDigest: question.textDigest,
        sourceLocator: question.sourceLocator,
      });
    }
    questionCount += questions.length;
    forms.push({
      formCode: form.formCode,
      documentTitle: form.documentTitle ?? null,
      formKind: form.formKind ?? null,
      pageCount: form.pageCount ?? null,
      sourceArchivePath: form.archivePath ?? null,
      sourceArchiveSha256: form.archiveSha256 ?? null,
      sourceFormSha256: form.formSha256 ?? null,
      questionCount: questions.length,
      questionBoundaryState: questions.length > 0 ? "IMMUTABLE_SOURCE_BOUNDARY" : "NO_QUESTION_BOUNDARY_IN_SOURCE",
      questions,
    });
  }
  required(forms.length === 52, `expected 52 forms, got ${forms.length}`);
  required(boundaryForms === 31, `expected 31 question-boundary forms, got ${boundaryForms}`);
  required(questionCount === 1310, `expected 1310 questions, got ${questionCount}`);

  const rootLines = [];
  for (const form of forms) {
    rootLines.push(`form\x00${form.formCode}\x00${form.sourceFormSha256 ?? ""}\x00${form.sourceArchiveSha256 ?? ""}\x00${form.questionCount}\n`);
    for (const question of form.questions) {
      rootLines.push(`question\x00${form.formCode}\x00${question.ordinal}\x00${question.immutableQuestionId}\x00${question.immutableQuestionVersionId}\x00${question.textDigest}\x00${question.sourceLocator ?? ""}\n`);
    }
  }
  const catalogRootDigest = sha256Text(rootLines.join(""));
  const sourceManifest = {
    schemaVersion: 1,
    manifestKind: "AGA_APPROVED_SOURCE_TECHNICAL_MANIFEST",
    sourceClassification: {
      origin: "IMPORTED_APPROVED_SOURCE",
      authority: "DIRECT_AVIATION_SOURCE_APPROVED_BY_PROJECT_OWNER",
      humanApprovalRequired: false,
      deploymentBlocking: false,
    },
    originalSourceArchive: {
      sha256: ORIGINAL_SOURCE_SHA256,
      pdfCount: 53,
      bytes: 12227415,
    },
    historicalV1: {
      zipSha256: V1_ZIP_SHA256,
      jsonSha256: V1_JSON_SHA256,
      registerSha256: V1_REGISTER_SHA256,
      derivedRegisterSha256: V1_DERIVED_REGISTER_SHA256,
      packageVersion: "AGA_ALL_FORMS_SOURCE_RISK_DRAFT_V1",
    },
    packageVersion: "AGA_ALL_FORMS_APPROVED_SOURCE_V2",
    catalogUsageClass: "GOVERNED_OPERATIONAL",
    catalogOrigin: "IMPORTED_APPROVED_SOURCE",
    formCount: forms.length,
    questionBoundaryFormCount: boundaryForms,
    questionCount,
    catalogRootDigest,
    forms: forms.map((form) => ({
      formCode: form.formCode,
      sourceFormSha256: form.sourceFormSha256,
      sourceArchiveSha256: form.sourceArchiveSha256,
      questionCount: form.questionCount,
      questionBoundaryState: form.questionBoundaryState,
    })),
    orderedQuestions: orderedRows,
  };
  const sourceManifestText = canonicalJson(sourceManifest);
  const sourceManifestSha256 = sha256Text(sourceManifestText);
  const packageJson = {
    schemaVersion: 2,
    packageVersion: "AGA_ALL_FORMS_APPROVED_SOURCE_V2",
    status: "IMPORTED_APPROVED_SOURCE",
    candidateOnly: false,
    sourceClassification: sourceManifest.sourceClassification,
    sourceManifestSha256,
    catalogRootDigest,
    catalogUsageClass: "GOVERNED_OPERATIONAL",
    catalogOrigin: "IMPORTED_APPROVED_SOURCE",
    optionalEnrichmentPolicy: {
      status: "OPTIONAL_NOT_APPROVED",
      riskAndRegulatoryMappingMayBeAbsent: true,
      sourceRefsAreNotApprovedConclusions: true,
      configuredReferenceMustRemainEmptyWhenUnverified: true,
    },
    originalSourceArchive: sourceManifest.originalSourceArchive,
    historicalV1: sourceManifest.historicalV1,
    totals: { forms: forms.length, questionBoundaryForms: boundaryForms, questions: questionCount },
    forms,
  };
  const packageJsonText = canonicalJson(packageJson);
  const readme = [
    "# AGA Approved Source V2",
    "",
    "This package is an immutable technical import of the project-provided Aviation-approved AGA source.",
    "It is operationally governed by the source classification, not by a second human approval or publication step.",
    "",
    "Optional risk, regulatory, and currentness enrichment is explicitly not approved and is never required for import.",
    "The complete question inventory is available for Department Manager subset selection; import does not preselect it for an Audit.",
    "",
  ].join("\n");

  const stage = mkdtempSync(join(tmpdir(), "avia-aga-v2-"));
  try {
    const packageRoot = join(stage, "aga-approved-source-v2");
    mkdirSync(packageRoot, { recursive: true });
    const files = new Map([
      ["AGA_ALL_FORMS_APPROVED_SOURCE_V2.json", packageJsonText],
      ["AGA_APPROVED_SOURCE_MANIFEST.json", sourceManifestText],
      ["README.md", readme],
    ]);
    const manifestLines = [];
    for (const [name, text] of [...files.entries()].sort(([left], [right]) => left.localeCompare(right))) {
      const path = join(packageRoot, name);
      writeFileSync(path, text, { mode: 0o644 });
      utimesSync(path, FIXED_TIME, FIXED_TIME);
      manifestLines.push(`${sha256Text(text).slice("sha256:".length)}  ${name}\n`);
    }
    const manifestText = manifestLines.join("");
    const manifestPath = join(packageRoot, "MANIFEST.sha256");
    writeFileSync(manifestPath, manifestText, { mode: 0o644 });
    utimesSync(manifestPath, FIXED_TIME, FIXED_TIME);
    rmSync(outputZip, { force: true });
    execFileSync("zip", ["-X", "-q", "-r", outputZip, "aga-approved-source-v2"], { cwd: stage });
    const zipBytes = readFileSync(outputZip);
    const result = {
      package: outputZip,
      packageVersion: packageJson.packageVersion,
      packageSha256: sha256(zipBytes),
      packageBytes: zipBytes.length,
      packageJsonSha256: sha256Text(packageJsonText),
      sourceManifestSha256,
      catalogRootDigest,
      formCount: forms.length,
      questionBoundaryFormCount: boundaryForms,
      questionCount,
    };
    writeFileSync(join(surveilRoot, "deliverables", "AGA_ALL_FORMS_APPROVED_SOURCE_V2.build.json"), `${JSON.stringify(result, null, 2)}\n`);
    process.stdout.write(`${JSON.stringify(result)}\n`);
  } finally {
    rmSync(stage, { recursive: true, force: true });
  }
}

build();
