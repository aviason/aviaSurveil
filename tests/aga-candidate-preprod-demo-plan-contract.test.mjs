import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const planPath = "docs/exec-plans/active/2026-08-01-preprod-only-aga-candidate-demo-intake-plan.md";
const specificationPaths = [
  "docs/product-specs/data-and-rules/PREPROD_IDENTITY_AND_DATA_PROFILE.md",
  "docs/product-specs/modules/CHECKLIST_BUILDER_AND_RUNNER.md",
  "docs/product-specs/modules/ADMIN_CONFIGURATION.md",
  "docs/product-specs/data-and-rules/CONCEPTUAL_DATA_MODEL.md",
  "docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md",
  "docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.md",
];

async function readContract() {
  const plan = await readFile(planPath, "utf8");
  const match = plan.match(/<!-- PREPROD_AGA_CANDIDATE_DEMO_CONTRACT:BEGIN -->\s*```json\s*([\s\S]*?)\s*```\s*<!-- PREPROD_AGA_CANDIDATE_DEMO_CONTRACT:END -->/);
  assert.ok(match, "the AGA candidate demo contract must remain fenced and machine-readable");
  return JSON.parse(match[1]);
}

test("Gate 0 freezes every immutable AGA candidate demo contract boundary", async () => {
  const contract = await readContract();

  assert.equal(contract.schemaVersion, "preprod-aga-candidate-demo-contract/v1");
  assert.equal(contract.contractId, "aviasurveil360-preprod-aga-candidate-demo");
  assert.equal(contract.contractVersion, "1.1.0");
  assert.deepEqual(contract.target, {
    environment: "local-preprod",
    databaseName: "aviasurveil360_local_preprod",
    databaseOwner: "aviasurveil360_preprod_loader",
    composeProject: "aviasurveil360-local-preprod",
    overlaySchema: "preprod_aga_demo",
    operation: "LOAD_AGA_CANDIDATE_DEMO_OVERLAY",
    cleanup: "DROP_RECREATE_EXACT_WHOLE_NAMESPACE_ONLY",
    requiredRuntimeBindings: ["POSTGRES_SYSTEM_IDENTIFIER", "POSTGRES_HOST", "POSTGRES_PORT", "BASE_TARGET_FINGERPRINT_DIGEST", "BASE_PROFILE_VERSION", "BASE_RUN_ID", "BASE_INTENT_DIGEST", "BASE_RESULT_DIGEST"],
    databaseAccess: {
      normalApi: "NO_OVERLAY_PRIVILEGE",
      taggedReader: "SELECT_SEALED_VIEWS_ONLY",
      oneShotWriter: "OVERLAY_DDL_DML_ONLY",
      public: "NO_OVERLAY_PRIVILEGE",
    },
  });
  assert.deepEqual(contract.input, {
    zipFile: "AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip",
    zipBytes: 336524,
    zipSha256: "sha256:30700a88aeb5b26514bf7eb76bef050deb08b96294db94117d185de5c9f163b2",
    packageVersion: "AGA_ALL_FORMS_SOURCE_RISK_DRAFT_V1",
    packageStatus: "PENDING_ADMIN_AND_SOURCE_OWNER_REVIEW",
    candidateOnly: true,
    jsonBytes: 3370312,
    jsonSha256: "sha256:5ebcce2d70ee22fef4165b490cb6e4b276ad776f40dbaf12e5cea85c9da91b15",
    manifestSha256: "sha256:1be7b37e78a320da51cf7069b033240f1ad032b045d3e3cd5746c4b2115c19dc",
    sourceArchiveSha256: "sha256:dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32",
    sourceArchiveBytes: 12227415,
    registerSha256: "sha256:29ed8384693b615926fc42a0ca4654be2ea9a36b0946f217975571ca0ad9564f",
  });
  assert.deepEqual(contract.expected, {
    formIdentities: 52,
    formsWithCandidateBoundaries: 31,
    questionExtractionReviewRequiredForms: 21,
    candidateQuestions: 1310,
    questionsWithSourceProposals: 1261,
    questionsExplicitlyUnmapped: 49,
    questionSourceProposalLinks: 2329,
    formSourceProposalLinks: 274,
    uniqueSourceReferences: 174,
    proposedRiskBands: { PROPOSED_CONTROL_ASSURANCE: 50, PROPOSED_HIGH_OPERATIONAL: 457, PROPOSED_REVIEW_REQUIRED: 14, PROPOSED_SAFETY_CRITICAL: 789 },
    expertRiskReviewBlockers: 14,
    formProposedRiskBands: { PROPOSED_CONTROL_ASSURANCE: 11, PROPOSED_HIGH_OPERATIONAL: 23, PROPOSED_REVIEW_REQUIRED: 4, PROPOSED_SAFETY_CRITICAL: 14 },
    proposedSafetyCritical: { true: 789, false: 521 },
    packageExtractionStates: { EXACT_SOURCE_BACKED: 28, EXTRACTED_CANDIDATE: 1282 },
    packageRiskReviewStates: { CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW: 1310 },
  });
  assert.deepEqual(contract.zeroBoundaryFormCodes, [
    "FSS-AGA-FORM-001", "FSS-AGA-FORM-003", "FSS-AGA-FORM-004", "FSS-AGA-FORM-005", "FSS-AGA-FORM-007", "FSS-AGA-FORM-008", "FSS-AGA-FORM-025", "FSS-AGA-FORM-026", "FSS-AGA-FORM-029", "FSS-AGA-FORM-032", "FSS-AGA-FORM-033", "FSS-AGA-FORM-035A", "FSS-AGA-FORM-036", "FSS-AGA-FORM-038", "FSS-AGA-FORM-039", "FSS-AGA-FORM-042", "FSS-AGA-FORM-043", "FSS-AGA-FORM-044", "FSS-AGA-FORM-045", "FSS-AGA-FORM-046", "FSS-AGA-FORM-052",
  ]);
  assert.deepEqual(contract.fixedStates, {
    formIdentity: "NON_AUTHORITATIVE_FORM_IDENTITY",
    zeroBoundaryForm: "QUESTION_EXTRACTION_REVIEW_REQUIRED",
    question: "NON_AUTHORITATIVE_CANDIDATE",
    sourceMapping: "SOURCE_MAPPING_REQUIRED",
    proposalPresent: "PROPOSAL_PRESENT_REVIEW_REQUIRED",
    noQuestionProposal: "UNMAPPED_NO_QUESTION_LEVEL_SOURCE_PROPOSAL",
    risk: "PROVISIONAL_RISK_PROPOSAL",
    packageRiskReview: "CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW",
    exactSourceBackedProvenance: "EXACT_SOURCE_BACKED",
    extractedCandidateProvenance: "EXTRACTED_CANDIDATE",
    reviewRiskBlocker: "EXPERT_RISK_REVIEW_REQUIRED",
  });
  assert.deepEqual(contract.sourceResolutionRequirements, ["EXACT_SOURCE_BYTES", "EXACT_SOURCE_BYTES_SHA256", "EFFECTIVE_DATE", "CLAUSE_OR_PAGE_LOCATOR", "APPLICABILITY", "NAMED_SOURCE_OWNER_ATTESTATION"]);
  assert.equal(contract.sourceResolutionBoundary, "PACKAGE_LOCAL_MINIMUM_NECESSARY_NOT_SUFFICIENT_FOR_GOVERNED_RESOLUTION");
  assert.deepEqual(contract.reconciliation, {
    canonicalizationContract: "AVIA_AGA_CANDIDATE_DEMO_CANONICAL_V1",
    allPersistedAndExposedFieldsDigestBound: true,
    sealInsertedOnlyAfterInTransactionReconciliation: true,
    committedSealIsDatabaseReadabilityReceipt: true,
  });
  assert.deepEqual(contract.privacy, {
    allowedTopLevelRoles: ["admin"],
    requiredOrganizationScope: "exact-CAA",
    nonAdminOutcome: "NOT_FOUND_WITHOUT_EXISTENCE_SIGNAL",
    browserPersistence: "FORBIDDEN",
    httpCache: "NO_STORE_ALL_OUTCOMES",
    denial: "AUTHORIZATION_BEFORE_PARSE_OR_LOOKUP_NEUTRAL_LABEL_FREE_NOT_FOUND",
    candidateObservability: "FORBIDDEN",
    retainedRealPackageBrowserArtifacts: "FORBIDDEN",
  });
  assert.deepEqual(contract.forbiddenEffects, ["ADMIN_PRODUCT_DECISION", "EXTRACTION_DECISION", "REAL_IMPORT_BATCH", "REAL_IMPORT_FILE_OR_RECEIPT", "EXTRACTION_REVIEW_PACKET", "REAL_CHECKLIST_CANDIDATE", "GOVERNED_DRAFT", "SOURCE_CURRENTNESS_ACTIVATION", "SOURCE_AUTHORITY_ATTESTATION", "SOURCE_MAPPING_ATTESTATION", "FUNCTIONAL_ASSIGNMENT", "TECHNICAL_APPROVAL", "PUBLICATION", "TEMPLATE_VERSION", "AUDIT_PACKAGE_ELIGIBILITY", "AUDIT_RECORD_OR_PACKAGE", "FINDING_RECORD", "NOTIFICATION", "OUTBOX_ITEM", "CAA_OR_PROVIDER_DELIVERY", "PRODUCTION_RECORD", "FINDING_SEVERITY", "AUTOMATIC_SAFETY_CRITICAL_APPROVAL", "PRODUCTION_READINESS_CLAIM"]);
  assert.deepEqual(contract.labels, ["candidate-only", "release pending", "production-ready: not established"]);
});

test("Gate 0 makes the overlay a separate immutable preprod-only specification", async () => {
  const specifications = await Promise.all(specificationPaths.map((path) => readFile(path, "utf8")));
  const combined = specifications.join("\n");

  for (const phrase of [
    "aga-candidate-demo@1.1.0",
    "preprod_aga_demo",
    "READ_ONLY_PREPROD_DEMO",
    "SEALED_PREPROD_DEMO_PROJECTION",
    "QUESTION_EXTRACTION_REVIEW_REQUIRED",
    "NON_AUTHORITATIVE_CANDIDATE",
    "PROPOSAL_PRESENT_REVIEW_REQUIRED",
    "UNMAPPED_NO_QUESTION_LEVEL_SOURCE_PROPOSAL",
    "PROVISIONAL_RISK_PROPOSAL",
    "EXPERT_RISK_REVIEW_REQUIRED",
    "CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW",
    "EXACT_SOURCE_BACKED",
    "EXTRACTED_CANDIDATE",
    "candidate-only",
    "release pending",
    "production-ready: not established",
  ]) {
    assert.match(combined, new RegExp(phrase.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "i"));
  }

  for (const specification of specifications) {
    assert.match(specification, /aga-candidate-demo@1\.1\.0[\s\S]*read-only[\s\S]*immutable[\s\S]*preprod-only[\s\S]*Admin-only/i);
    assert.match(specification, /cannot\s+satisfy Task\s+9/i);
  }
});
