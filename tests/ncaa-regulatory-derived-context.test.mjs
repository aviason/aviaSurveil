import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const assessmentPath = path.join(
  repositoryRoot,
  "docs/regulatory-sources/derived/ncaa-namcats-part-127-140-applicability.json",
);
const assessmentNotePath = path.join(
  repositoryRoot,
  "docs/regulatory-sources/derived/ncaa-namcats-part-127-140-applicability.md",
);
const regulatoryReadmePath = path.join(
  repositoryRoot,
  "docs/regulatory-sources/README.md",
);
const harnessRegistryPath = path.join(
  repositoryRoot,
  "docs/agent-harness/registry.md",
);
const conceptualModelPath = path.join(
  repositoryRoot,
  "docs/product-specs/data-and-rules/CONCEPTUAL_DATA_MODEL.md",
);
const debtTrackerPath = path.join(
  repositoryRoot,
  "docs/exec-plans/tech-debt-tracker.md",
);
const manifestPath = path.join(
  repositoryRoot,
  "docs/regulatory-sources/ncaa-namcats-manifest.json",
);

const assessment = JSON.parse(readFileSync(assessmentPath, "utf8"));
const manifest = JSON.parse(readFileSync(manifestPath, "utf8"));
const assessmentNote = readFileSync(assessmentNotePath, "utf8");
const regulatoryReadme = readFileSync(regulatoryReadmePath, "utf8");
const harnessRegistry = readFileSync(harnessRegistryPath, "utf8");
const conceptualModel = readFileSync(conceptualModelPath, "utf8");
const debtTracker = readFileSync(debtTrackerPath, "utf8");
const manifestById = new Map(
  manifest.documents.map((document) => [document.id, document]),
);
const sourceById = new Map(
  assessment.sourceVersions.map((source) => [source.sourceVersionId, source]),
);
const evidenceById = new Map(
  assessment.evidenceRecords.map((evidence) => [evidence.evidenceId, evidence]),
);

const expectedQuestionIds = [
  "CAB-GALLEY-001",
  "CAB-LAV-001",
  "CAB-PAX-SEAT-001",
  "CAB-EMEQ-PBE-001",
  "CAB-VID-CREW-SEAT-001",
  "CAB-COCKPIT-GEN-001",
];

test("keeps the Part 127 / Part 140 assessment explicitly candidate and human-gated", () => {
  assert.equal(assessment.schemaVersion, 1);
  assert.equal(
    assessment.status.assessmentStatus,
    "CANDIDATE_DERIVED_CONTEXT",
  );
  assert.equal(assessment.status.evidenceStatus, "SOURCE_BOUND");
  assert.equal(assessment.status.reviewStatus, "EXPERT_REVIEW_REQUIRED");
  assert.equal(
    assessment.status.publicationStatus,
    "NOT_PUBLISHED_BY_THIS_ASSESSMENT",
  );
  assert.equal(assessment.status.productionReadiness, "NOT_CLAIMED");
  assert.equal(assessment.scope.sourceCollectionId, manifest.libraryId);
  assert.equal(
    assessment.scope.sourceManifest,
    "docs/regulatory-sources/ncaa-namcats-manifest.json",
  );

  assert.deepEqual(
    assessment.applicabilityDecisions.map(({ part, classification }) => ({
      part,
      classification,
    })),
    [
      { part: "127", classification: "OPERATION_TYPE_CONDITIONAL" },
      { part: "140", classification: "SYSTEM_LEVEL_APPLICABLE" },
    ],
  );

  const gates = assessment.governanceGates.map(({ gate }) => gate);
  assert.deepEqual(gates, [
    "CURRENT_SOURCE_AUTHORITY",
    "APPLICABILITY_AND_INTERPRETATION",
    "CONTROLLED_PROCEDURE",
    "QUESTION_DECOMPOSITION_AND_EVIDENCE",
    "CHECKLIST_PUBLICATION",
  ]);
  assert.match(
    assessment.guardrails.join("\n"),
    /cannot automatically decide compliance[\s\S]*Finding closure/u,
  );
});

test("binds every reviewed source to the exact tracked manifest identity", () => {
  assert.deepEqual(
    assessment.sourceVersions.map(({ sourceVersionId }) => sourceVersionId),
    [
      "NCAA-NAMCATS-P2-PDF-16",
      "NCAA-NAMCATS-P3-PDF-07",
      "NCAA-NAMCATS-P3-PDF-16",
    ],
  );

  for (const source of assessment.sourceVersions) {
    const manifestSource = manifestById.get(source.sourceVersionId);
    assert.ok(
      manifestSource,
      `Missing manifest source ${source.sourceVersionId}`,
    );
    assert.equal(source.sourceUrl, manifestSource.sourceUrl);
    assert.equal(source.sourcePage, manifestSource.sourcePage);
    assert.equal(source.observedUpdatedAt, manifestSource.observedUpdatedAt);
    assert.equal(source.byteSize, manifestSource.byteSize);
    assert.equal(source.sha256, manifestSource.sha256);
    assert.equal(source.extractionStatus, manifestSource.extractionStatus);
    assert.equal(
      source.extractionEngine,
      manifestSource.extraction.engine,
    );
    assert.equal(source.pageCount, manifestSource.extraction.pageCount);
    assert.equal(
      source.pagesWithContextText,
      manifestSource.extraction.pagesWithContextText,
    );
    assert.equal(
      source.characterCount,
      manifestSource.extraction.characterCount,
    );
    assert.equal(
      source.extractedTextLocator,
      manifestSource.extractedTextRelativePath,
    );
    assert.match(
      source.extractedTextLocator,
      /^\.local\/aviasurveil360\/regulatory-sources\/ncaa\/namcats\/all-pages\/text\/.+\.pdf\.txt$/u,
    );
    assert.ok(source.versionCaveat.length > 30);
  }
});

test("keeps every evidence citation source-bound and within the cited PDF", () => {
  assert.equal(assessment.evidenceRecords.length, 13);
  assert.equal(
    evidenceById.size,
    assessment.evidenceRecords.length,
    "Evidence identifiers must be unique",
  );

  for (const evidence of assessment.evidenceRecords) {
    const source = sourceById.get(evidence.sourceVersionId);
    assert.ok(
      source,
      `Unknown source for evidence ${evidence.evidenceId}`,
    );
    assert.ok(evidence.section.length > 3);
    assert.ok(evidence.paraphrasedEvidence.length > 30);
    assert.ok(evidence.candidateUse.length > 20);
    assert.ok(evidence.limitation.length > 20);
    assert.ok(evidence.pdfPages.length > 0);
    for (const pdfPage of evidence.pdfPages) {
      assert.ok(Number.isInteger(pdfPage));
      assert.ok(pdfPage >= 1 && pdfPage <= source.pageCount);
    }
  }

  assert.equal(
    sourceById.get("NCAA-NAMCATS-P3-PDF-16").documentRole,
    "COMPARATOR_SOURCE",
  );
  assert.equal(
    sourceById.get("NCAA-NAMCATS-P3-PDF-07").authorityStatus,
    "CANDIDATE_CURRENT_PUBLIC_REFERENCE",
  );
});

test("records the six pilot implications without promoting Part 140 to an item-specific rule", () => {
  assert.deepEqual(
    assessment.checklistImplications.map(({ questionId }) => questionId),
    expectedQuestionIds,
  );

  for (const implication of assessment.checklistImplications) {
    assert.equal(
      implication.part140Disposition,
      "CONTEXTUAL_RISK_AND_ASSURANCE_ONLY",
    );
    assert.ok(implication.candidateConclusion.length > 40);
    assert.ok(implication.requiredExpertAction.length > 30);
    for (const evidenceId of implication.part127EvidenceIds) {
      assert.ok(
        evidenceById.has(evidenceId),
        `Unknown evidence ${evidenceId} for ${implication.questionId}`,
      );
      assert.equal(
        evidenceById.get(evidenceId).sourceVersionId,
        "NCAA-NAMCATS-P2-PDF-16",
      );
    }
  }

  const pbe = assessment.checklistImplications.find(
    ({ questionId }) => questionId === "CAB-EMEQ-PBE-001",
  );
  const lavatory = assessment.checklistImplications.find(
    ({ questionId }) => questionId === "CAB-LAV-001",
  );
  assert.equal(
    pbe.part127Disposition,
    "GENERIC_EMERGENCY_EQUIPMENT_ONLY_NO_PBE_SPECIFIC_MATCH",
  );
  assert.equal(lavatory.part127Disposition, "NO_DIRECT_SOURCE_MATCH");
  assert.equal(lavatory.part127EvidenceIds.length, 0);
});

test("routes the human-readable note to the machine record and preserves source gaps", () => {
  assert.match(
    assessmentNote,
    /\[.*ncaa-namcats-part-127-140-applicability\.json.*\]\(ncaa-namcats-part-127-140-applicability\.json\)/u,
  );
  assert.match(assessmentNote, /`OPERATION_TYPE_CONDITIONAL`/u);
  assert.match(assessmentNote, /`SYSTEM_LEVEL_APPLICABLE`/u);
  assert.match(assessmentNote, /No direct extracted-text match was found/u);
  assert.match(assessmentNote, /not\s+legal advice/u);
  assert.match(assessmentNote, /Finding closure/u);
});

test("routes the derived context through the regulatory and harness documentation", () => {
  for (const fileName of [
    "ncaa-namcats-part-127-140-applicability.md",
    "ncaa-namcats-part-127-140-applicability.json",
  ]) {
    assert.match(regulatoryReadme, new RegExp(fileName, "u"));
  }
  assert.match(
    harnessRegistry,
    /tests\/ncaa-regulatory-derived-context\.test\.mjs/u,
  );
  assert.match(
    harnessRegistry,
    /Regulatory source assessment[\s\S]*ignored local vault/u,
  );
  assert.match(conceptualModel, /### DerivedRegulatoryAssessment/u);
  assert.match(conceptualModel, /`OPERATION_TYPE_CONDITIONAL`/u);
  assert.match(conceptualModel, /`SYSTEM_LEVEL_APPLICABLE`/u);
  assert.match(
    debtTracker,
    /source-bound Part 127 \/ Part 140 assessment/u,
  );
});
