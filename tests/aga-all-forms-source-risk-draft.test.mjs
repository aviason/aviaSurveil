import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readdir, readFile } from "node:fs/promises";
import { join } from "node:path";
import { test } from "node:test";

const packageDir = join(process.cwd(), "deliverables/aga-all-forms-source-risk-draft-2026-08-01");
const packagePath = join(packageDir, "AGA_ALL_FORMS_SOURCE_RISK_DRAFT.json");

const expectedFormCodes = [
  ...Array.from({ length: 34 }, (_, index) => String(index + 1).padStart(3, "0")),
  ...Array.from({ length: 13 }, (_, index) => String(index + 36).padStart(3, "0")),
  "035A",
  "050",
  "051",
  "052",
  "053",
].sort((a, b) => a.localeCompare(b, undefined, { numeric: true }));

function csvDataRows(text) {
  return String(text).trimEnd().split("\n").slice(1);
}

test("all-form source/risk draft is a bounded, candidate-only review package", async () => {
  const draft = JSON.parse(await readFile(packagePath, "utf8"));
  assert.equal(draft.packageVersion, "AGA_ALL_FORMS_SOURCE_RISK_DRAFT_V1");
  assert.equal(draft.status, "PENDING_ADMIN_AND_SOURCE_OWNER_REVIEW");
  assert.equal(draft.candidateOnly, true);
  assert.equal(draft.archive.sha256, "sha256:dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32");
  assert.equal(draft.archive.bytes, 12227415);
  assert.equal(draft.archive.pdfEntryCount, 53);
  assert.equal(draft.archive.formCount, 52);
  assert.equal(draft.archive.rawBytesPersisted, false);
  assert.equal(draft.archive.registerSha256, "sha256:29ed8384693b615926fc42a0ca4654be2ea9a36b0946f217975571ca0ad9564f");

  assert.deepEqual(draft.forms.map((form) => form.formCode.replace("FSS-AGA-FORM-", "")), expectedFormCodes);
  assert.equal(draft.forms.length, 52);
  assert.equal(draft.totals.forms, 52);
  assert.equal(draft.totals.questions, 1310);
  assert.equal(draft.totals.sourceRefs, 174);

  const form048 = draft.forms.find((form) => form.formCode === "FSS-AGA-FORM-048");
  assert.equal(form048.questionCount, 28);
  assert.equal(form048.questionExtractionState, "EXTRACTED_CANDIDATE_BOUNDARIES");
  assert.equal(form048.formSourceRefs.includes("NAMCARs 139.17.2"), true);
  assert.equal(form048.documentTitle, "Check-list for the surveillance of an aerodrome");

  const questions = draft.forms.flatMap((form) => form.questions);
  assert.equal(questions.length, 1310);
  assert.equal(new Set(questions.map((question) => question.proposalId)).size, questions.length);
  for (const form of draft.forms) {
    assert.equal(form.sourceMappingState, "SOURCE_MAPPING_REQUIRED");
    assert.equal(form.candidateState, "NOT_IMPORTED");
    assert.equal(form.publicationState, "NOT_AUTHORIZED");
    assert.ok(Array.isArray(form.formSourceRefs));
    assert.ok(Array.isArray(form.formSourceProposals));
    for (const question of form.questions) {
      assert.ok(question.originalText.endsWith("?"));
      assert.ok(question.sourceLocator.includes("Protocol question column"));
      assert.equal(question.sourceMappingState, "SOURCE_MAPPING_REQUIRED");
      assert.equal(question.sourceAuthorityState, "NOT_ATTESTED");
      assert.equal(question.riskClassificationState, "CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW");
      assert.equal(question.decisionState, "NOT_SUPPLIED");
      assert.match(question.proposedRisk.band, /^PROPOSED_/);
      assert.equal(typeof question.proposedRisk.safetyCritical, "boolean");
    }
  }

  assert.equal(draft.sourceCoverage.length, 174);
  assert.ok(draft.sourceCoverage.some((source) => source.status === "PROPOSED_LOCAL_SOURCE" && source.sourceSha256));
  assert.ok(draft.sourceCoverage.some((source) => source.status === "PROPOSED_EXTERNAL_OFFICIAL_SOURCE" && source.sourceSha256 === null));
  assert.equal(draft.officialSourcePolicy.noAutomaticAuthority, true);
  assert.equal(draft.riskPolicy.safetyCriticalIsAdvisory, true);
  assert.equal(draft.riskPolicy.noAutomaticFindingSeverity, true);
  assert.equal(draft.futureChangePolicy.changeAction.includes("never mutate published checklist"), true);

  const files = await readdir(packageDir);
  assert.equal(files.some((file) => /\.(pdf|zip)$/i.test(file)), false, "raw PDF/ZIP bytes must not be in the review package");
  assert.equal(csvDataRows(await readFile(join(packageDir, "AGA_ALL_FORMS_REGISTER.csv"))).length, 52);
  assert.equal(csvDataRows(await readFile(join(packageDir, "AGA_ALL_FORMS_QUESTION_MAPPING_RISK_QUEUE.csv"))).length, 1310);
  assert.equal(csvDataRows(await readFile(join(packageDir, "AGA_ALL_FORMS_SOURCE_COVERAGE.csv"))).length, 174);

  const manifestLines = (await readFile(join(packageDir, "MANIFEST.sha256"), "utf8")).trimEnd().split("\n");
  assert.equal(manifestLines.length, 7);
  for (const line of manifestLines) {
    const [, expectedDigest, file] = line.match(/^([a-f0-9]{64}) {2}(.+)$/u) ?? [];
    assert.ok(expectedDigest && file, `invalid package manifest line: ${line}`);
    const actualDigest = createHash("sha256").update(await readFile(join(packageDir, file))).digest("hex");
    assert.equal(actualDigest, expectedDigest, `manifest mismatch for ${file}`);
  }
});

test("all-form review handoff contains explicit non-authorizations", async () => {
  const readme = await readFile(join(packageDir, "README.md"), "utf8");
  const turkishMessage = await readFile(join(packageDir, "FURKAN_MESSAGE_DRAFT.tr.md"), "utf8");
  const authorization = JSON.parse(await readFile(join(packageDir, "TASK9_ALL_FORMS_EXPANSION_AUTHORIZATION_TEMPLATE.json"), "utf8"));
  for (const text of [readme, turkishMessage]) {
    assert.match(text, /candidate-only/);
    assert.match(text, /source mapping/i);
    assert.match(text, /publication/i);
    assert.match(text, /production-ready/);
  }
  assert.match(readme, /candidate question strings/i);
  assert.match(readme, /raw AGA ZIP\/PDF byte artifacts/i);
  assert.doesNotMatch(readme, /contains derived metadata only/i);
  assert.equal(authorization.status, "NOT_SUPPLIED");
  assert.equal(authorization.candidateOnly, true);
  assert.equal(authorization.sourceMappingAuthorized, false);
  assert.equal(authorization.publicationAuthorized, false);
  assert.equal(authorization.formCodes.length, 52);
});
