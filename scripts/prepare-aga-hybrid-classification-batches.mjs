import { createHash } from "node:crypto";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const expected = Object.freeze({
  package: Object.freeze({
    bytes: 3370312,
    sha256: "5ebcce2d70ee22fef4165b490cb6e4b276ad776f40dbaf12e5cea85c9da91b15",
  }),
  researchZip: Object.freeze({
    bytes: 76750,
    sha256: "137592c739bc22f6be026f5bad94c5b200bb983132017d026b7e39634ab392c7",
  }),
  workbook: Object.freeze({
    bytes: 12228,
    sha256: "e4d054f741b11ca9d848842a891d6f811f2e644aba29a7ffda970bfe6abb931e",
  }),
  researchQuestionCSV: Object.freeze({
    bytes: 296840,
    sha256: "e39685d467c9c66220b20e998deab366a138148f4d532db7fac07e58e64e7a7c",
  }),
});

class ControlledError extends Error {
  constructor(code) {
    super(code);
    this.code = code;
  }
}

function fail(code) {
  throw new ControlledError(code);
}

function digestBytes(bytes) {
  return `sha256:${createHash("sha256").update(bytes).digest("hex")}`;
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
  return digestBytes(Buffer.from(`${domain}${canonicalJSON(value)}`, "utf8"));
}

function fileFact(path, expectedFact, code) {
  let bytes;
  try {
    bytes = readFileSync(path);
  } catch {
    fail(`${code}_UNAVAILABLE`);
  }
  const fact = {
    bytes: bytes.byteLength,
    sha256: digestBytes(bytes).slice("sha256:".length),
  };
  if (fact.bytes !== expectedFact.bytes || fact.sha256 !== expectedFact.sha256) {
    fail(`${code}_MISMATCH`);
  }
  return fact;
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
  if (!headers || headers.length === 0) fail("RESEARCH_HEADER_MISSING");
  const objects = [];
  for (const candidate of rows) {
    if (candidate.length === 1 && candidate[0] === "") continue;
    if (candidate.length !== headers.length) fail("RESEARCH_CSV_MALFORMED");
    objects.push(
      Object.fromEntries(headers.map((header, index) => [header, candidate[index]])),
    );
  }
  return objects;
}

function readZipEntry(zipPath, entry, expectedFact) {
  const result = spawnSync("unzip", ["-p", zipPath, entry], {
    encoding: null,
    maxBuffer: 4 * 1024 * 1024,
  });
  if (result.status !== 0 || !Buffer.isBuffer(result.stdout)) {
    fail("RESEARCH_ENTRY_UNAVAILABLE");
  }
  if (
    result.stdout.byteLength !== expectedFact.bytes ||
    digestBytes(result.stdout).slice("sha256:".length) !== expectedFact.sha256
  ) {
    fail("RESEARCH_ENTRY_MISMATCH");
  }
  return result.stdout;
}

function parsePackage(path) {
  let document;
  try {
    document = JSON.parse(readFileSync(path, "utf8"));
  } catch {
    fail("PACKAGE_JSON_INVALID");
  }
  if (
    document?.packageVersion !== "AGA_ALL_FORMS_SOURCE_RISK_DRAFT_V1" ||
    document?.totals?.forms !== 52 ||
    document?.totals?.questions !== 1310 ||
    !Array.isArray(document.forms) ||
    !document.forms.every((form) => Array.isArray(form.questions))
  ) {
    fail("PACKAGE_CONTRACT_MISMATCH");
  }
  const formCodes = document.forms.map((form) => form.formCode);
  if (canonicalJSON(formCodes) !== canonicalJSON(expectedFormCodes())) {
    fail("PACKAGE_ORDER_MISMATCH");
  }
  return document;
}

function baseIdentity(document, form, question) {
  return {
    packageVersion: document.packageVersion,
    packageJsonSha256: `sha256:${expected.package.sha256}`,
    formCode: form.formCode,
    proposalId: question.proposalId,
    ordinal: question.ordinal,
    textDigest: question.textDigest,
  };
}

function researchIdentity(row) {
  return [row.form_code, row.proposal_id, row.ordinal, row.text_digest].join("\u0000");
}

function identityKey(identity) {
  return [
    identity.formCode,
    identity.proposalId,
    String(identity.ordinal),
    identity.textDigest,
  ].join("\u0000");
}

const omissionSignalRules = Object.freeze([
  Object.freeze({
    signalRuleId: "SOURCE_PROPOSAL_GAP_V1",
    cueCode: "SOURCE_REFERENCE_MISSING",
    inputFactSelector: "SOURCE_PROPOSAL_DIGEST",
    matcher: Object.freeze({ kind: "SOURCE_PROPOSALS_EMPTY" }),
  }),
  Object.freeze({
    signalRuleId: "PROVIDER_APPLICABILITY_UNRESOLVED_V1",
    cueCode: "PROVIDER_APPLICABILITY_UNRESOLVED",
    inputFactSelector: "RESEARCH_ROW_DIGEST",
    matcher: Object.freeze({
      kind: "RESEARCH_FIELD_EQUALS",
      field: "provider_applicability_unresolved",
      value: "true",
    }),
  }),
  Object.freeze({
    signalRuleId: "PRIOR_RESPONSE_DEPENDENCY_V1",
    cueCode: "CONDITIONAL_ON_PRIOR_RESPONSE",
    inputFactSelector: "QUESTION_BODY_DIGEST",
    matcher: Object.freeze({
      kind: "QUESTION_BODY_REGEX",
      pattern:
        "\\b(?:previous|preceding|above)\\s+(?:answer|item|question|response)\\b|\\bif\\s+(?:the\\s+)?(?:answer|response)\\s+is\\b",
      flags: "iu",
    }),
  }),
  Object.freeze({
    signalRuleId: "CROSS_QUESTION_DEPENDENCY_V1",
    cueCode: "CROSS_QUESTION_DEPENDENCY",
    inputFactSelector: "QUESTION_BODY_DIGEST",
    matcher: Object.freeze({
      kind: "QUESTION_BODY_REGEX",
      pattern:
        "\\b(?:refer|subject)\\s+to\\s+(?:item|question|response)\\b|\\bdepending\\s+on\\s+(?:item|question|response)\\b",
      flags: "iu",
    }),
  }),
  Object.freeze({
    signalRuleId: "EMBEDDED_FORM_TEMPLATE_TEXT_V1",
    cueCode: "EMBEDDED_FORM_TEMPLATE_TEXT",
    inputFactSelector: "QUESTION_BODY_DIGEST",
    matcher: Object.freeze({
      kind: "QUESTION_BODY_REGEX",
      pattern:
        "(?:^|\\n)\\s*(?:name|signature|date|remarks?|satisfactory|unsatisfactory|yes\\s*\\/\\s*no|n\\s*\\/\\s*a)\\s*[:_]",
      flags: "imu",
    }),
  }),
  Object.freeze({
    signalRuleId: "EXPLICIT_EXTERNAL_ACTOR_V1",
    cueCode: "EXTERNAL_INVOLVEMENT_CANDIDATE",
    inputFactSelector: "QUESTION_BODY_DIGEST",
    matcher: Object.freeze({
      kind: "QUESTION_BODY_REGEX",
      pattern:
        "\\b(?:ANSP|AIS|AIM|CNS|air\\s+operator|security\\s+agency|survey\\s+organization|registration\\s+holder)\\b",
      flags: "iu",
    }),
  }),
  Object.freeze({
    signalRuleId: "CONTRACTED_PERSONNEL_V1",
    cueCode: "CONTRACTED_PERSONNEL_SERVICE",
    inputFactSelector: "QUESTION_BODY_DIGEST",
    matcher: Object.freeze({
      kind: "QUESTION_BODY_REGEX",
      pattern:
        "\\b(?:contracted|contractor|third[- ]party)\\s+(?:personnel|service|provider|organization|activity|work)\\b",
      flags: "iu",
    }),
  }),
  Object.freeze({
    signalRuleId: "EXTERNAL_STAKEHOLDER_V1",
    cueCode: "EXTERNAL_STAKEHOLDER_PARTICIPATION",
    inputFactSelector: "QUESTION_BODY_DIGEST",
    matcher: Object.freeze({
      kind: "QUESTION_BODY_REGEX",
      pattern:
        "\\b(?:external|aviation|affected)\\s+stakeholders?\\b|\\bstakeholder\\s+(?:consultation|participation|collaboration)\\b",
      flags: "iu",
    }),
  }),
  Object.freeze({
    signalRuleId: "EMERGENCY_AGENCY_PARTICIPATION_V1",
    cueCode: "EMERGENCY_AGENCY_PARTICIPATION",
    inputFactSelector: "QUESTION_BODY_DIGEST",
    matcher: Object.freeze({
      kind: "QUESTION_BODY_REGEX",
      pattern: "\\b(?:emergency|rescue|fire)\\s+(?:agency|service|organization)\\b",
      flags: "iu",
    }),
  }),
  Object.freeze({
    signalRuleId: "EXTERNAL_REVIEW_PROVIDER_V1",
    cueCode: "EXTERNAL_REVIEW_PROVIDER",
    inputFactSelector: "QUESTION_BODY_DIGEST",
    matcher: Object.freeze({
      kind: "QUESTION_BODY_REGEX",
      pattern: "\\b(?:external|independent)\\s+(?:review|audit|auditor|assessor)\\b",
      flags: "iu",
    }),
  }),
  Object.freeze({
    signalRuleId: "EXTERNAL_APPROVAL_AUTHORITY_V1",
    cueCode: "EXTERNAL_APPROVAL_AUTHORITY",
    inputFactSelector: "QUESTION_BODY_DIGEST",
    matcher: Object.freeze({
      kind: "QUESTION_BODY_REGEX",
      pattern:
        "\\b(?:local|government|regulatory)\\s+(?:authority|institution)\\s+(?:approval|authorization)\\b",
      flags: "iu",
    }),
  }),
  Object.freeze({
    signalRuleId: "SPECIALIST_RESCUE_SERVICE_V1",
    cueCode: "SPECIALIST_RESCUE_SERVICE",
    inputFactSelector: "QUESTION_BODY_DIGEST",
    matcher: Object.freeze({
      kind: "QUESTION_BODY_REGEX",
      pattern: "\\bspecialist\\s+rescue\\s+service\\b",
      flags: "iu",
    }),
  }),
  Object.freeze({
    signalRuleId: "THIRD_PARTY_SIGNOFF_V1",
    cueCode: "THIRD_PARTY_SIGNOFF",
    inputFactSelector: "QUESTION_BODY_DIGEST",
    matcher: Object.freeze({
      kind: "QUESTION_BODY_REGEX",
      pattern: "\\bthird[- ]party\\s+sign[- ]?off\\b",
      flags: "iu",
    }),
  }),
]);

function freezeModelSignalMapping(
  candidateSignalRuleId,
  candidateCueCode,
  inputFactSelector,
  frozenSignalRuleId,
) {
  return Object.freeze({
    candidateSignalRuleId,
    candidateCueCode,
    inputFactSelector,
    frozenSignalRuleId,
  });
}

const modelSignalMappings = Object.freeze([
  freezeModelSignalMapping(
    "CONTRACTED_PERSONNEL_CUE_DETECTED",
    "CONTRACTED_PERSONNEL_SERVICE",
    "QUESTION_BODY_DIGEST",
    "CONTRACTED_PERSONNEL_V1",
  ),
  freezeModelSignalMapping(
    "CROSS_QUESTION_CONDITION_CUE_DETECTED",
    "CROSS_QUESTION_DEPENDENCY",
    "QUESTION_BODY_DIGEST",
    "CROSS_QUESTION_DEPENDENCY_V1",
  ),
  freezeModelSignalMapping(
    "DETECT_EMBEDDED_FORM_METADATA",
    "EMBEDDED_FORM_METADATA",
    "QUESTION_BODY_DIGEST",
    "EMBEDDED_FORM_TEMPLATE_TEXT_V1",
  ),
  freezeModelSignalMapping(
    "DETECT_EXPLICIT_EXTERNAL_ACTOR",
    "EXTERNAL_INVOLVEMENT_CANDIDATE",
    "QUESTION_BODY_DIGEST",
    "EXPLICIT_EXTERNAL_ACTOR_V1",
  ),
  freezeModelSignalMapping(
    "DETECT_PRIOR_RESPONSE_DEPENDENCY",
    "CONDITIONAL_ON_PRIOR_RESPONSE",
    "QUESTION_BODY_DIGEST",
    "PRIOR_RESPONSE_DEPENDENCY_V1",
  ),
  freezeModelSignalMapping(
    "EMBEDDED_FORM_TEMPLATE_TEXT_DETECTED",
    "EMBEDDED_FORM_TEMPLATE_TEXT",
    "QUESTION_BODY_DIGEST",
    "EMBEDDED_FORM_TEMPLATE_TEXT_V1",
  ),
  freezeModelSignalMapping(
    "EMERGENCY_AGENCY_PARTICIPATION_CUE_DETECTED",
    "EMERGENCY_AGENCY_PARTICIPATION",
    "QUESTION_BODY_DIGEST",
    "EMERGENCY_AGENCY_PARTICIPATION_V1",
  ),
  freezeModelSignalMapping(
    "EXTERNAL_APPROVAL_AUTHORITY_CUE_DETECTED",
    "GOVERNMENT_INSTITUTION_APPROVAL",
    "QUESTION_BODY_DIGEST",
    "EXTERNAL_APPROVAL_AUTHORITY_V1",
  ),
  freezeModelSignalMapping(
    "EXTERNAL_APPROVAL_AUTHORITY_CUE_DETECTED",
    "LOCAL_AUTHORITY_APPROVAL",
    "QUESTION_BODY_DIGEST",
    "EXTERNAL_APPROVAL_AUTHORITY_V1",
  ),
  freezeModelSignalMapping(
    "EXTERNAL_AUDIT_CUE_DETECTED",
    "INDEPENDENT_EXTERNAL_AUDITOR",
    "QUESTION_BODY_DIGEST",
    "EXTERNAL_REVIEW_PROVIDER_V1",
  ),
  freezeModelSignalMapping(
    "EXTERNAL_REVIEW_CUE_DETECTED",
    "EXTERNAL_REVIEW_PROVIDER",
    "QUESTION_BODY_DIGEST",
    "EXTERNAL_REVIEW_PROVIDER_V1",
  ),
  freezeModelSignalMapping(
    "EXTERNAL_STAKEHOLDER_CUE_DETECTED",
    "AVIATION_SAFETY_ORGANIZATION",
    "QUESTION_BODY_DIGEST",
    "EXTERNAL_STAKEHOLDER_V1",
  ),
  freezeModelSignalMapping(
    "EXTERNAL_STAKEHOLDER_CUE_DETECTED",
    "AVIATION_STAKEHOLDER_CONSULTATION",
    "QUESTION_BODY_DIGEST",
    "EXTERNAL_STAKEHOLDER_V1",
  ),
  freezeModelSignalMapping(
    "EXTERNAL_STAKEHOLDER_CUE_DETECTED",
    "CROSS_AERODROME_RST_COLLABORATION",
    "QUESTION_BODY_DIGEST",
    "EXTERNAL_STAKEHOLDER_V1",
  ),
  freezeModelSignalMapping(
    "EXTERNAL_STAKEHOLDER_CUE_DETECTED",
    "EXTERNAL_STAKEHOLDER_PARTICIPATION",
    "QUESTION_BODY_DIGEST",
    "EXTERNAL_STAKEHOLDER_V1",
  ),
  freezeModelSignalMapping(
    "LOCAL_AUTHORITY_INVOLVEMENT_CUE_DETECTED",
    "LOCAL_AUTHORITY_FIRE_PREVENTION",
    "QUESTION_BODY_DIGEST",
    "EMERGENCY_AGENCY_PARTICIPATION_V1",
  ),
  freezeModelSignalMapping(
    "MISSING_SOURCE_REFERENCE_DETECTED",
    "SOURCE_REFERENCE_MISSING",
    "RESEARCH_ROW_DIGEST",
    "SOURCE_PROPOSAL_GAP_V1",
  ),
  freezeModelSignalMapping(
    "QUESTION_BODY_EMBEDDED_FORM_BOILERPLATE",
    "EMBEDDED_FORM_BOILERPLATE",
    "QUESTION_BODY_DIGEST",
    "EMBEDDED_FORM_TEMPLATE_TEXT_V1",
  ),
  freezeModelSignalMapping(
    "RESEARCH_MISSING_SOURCE_REFERENCE",
    "SOURCE_REFERENCE_MISSING",
    "RESEARCH_ROW_DIGEST",
    "SOURCE_PROPOSAL_GAP_V1",
  ),
  freezeModelSignalMapping(
    "RESEARCH_PROVIDER_APPLICABILITY_UNRESOLVED",
    "POSSIBLE_EXTERNAL_ACTOR_AS_GRAMMATICAL_SUBJECT",
    "RESEARCH_ROW_DIGEST",
    "PROVIDER_APPLICABILITY_UNRESOLVED_V1",
  ),
  freezeModelSignalMapping(
    "RESEARCH_PROVIDER_UNRESOLVED",
    "PROVIDER_APPLICABILITY_UNRESOLVED",
    "RESEARCH_ROW_DIGEST",
    "PROVIDER_APPLICABILITY_UNRESOLVED_V1",
  ),
  freezeModelSignalMapping(
    "SPECIALIST_RESCUE_SERVICE_CUE_DETECTED",
    "SPECIALIST_RESCUE_SERVICE",
    "QUESTION_BODY_DIGEST",
    "SPECIALIST_RESCUE_SERVICE_V1",
  ),
  freezeModelSignalMapping(
    "THIRD_PARTY_SIGNOFF_CUE_DETECTED",
    "THIRD_PARTY_SIGNOFF",
    "QUESTION_BODY_DIGEST",
    "THIRD_PARTY_SIGNOFF_V1",
  ),
]);

function matchesOmissionSignalRule(rule, item) {
  if (rule.matcher.kind === "SOURCE_PROPOSALS_EMPTY") {
    return item.packageFacts.sourceProposalDigests.length === 0;
  }
  if (rule.matcher.kind === "RESEARCH_FIELD_EQUALS") {
    return item.researchCandidateFacts[rule.matcher.field] === rule.matcher.value;
  }
  if (rule.matcher.kind === "QUESTION_BODY_REGEX") {
    return new RegExp(rule.matcher.pattern, rule.matcher.flags).test(item.questionBody);
  }
  fail("OMISSION_SIGNAL_MATCHER_UNKNOWN");
}

function completeIdentityValidation(expectedIdentity, observedIdentity) {
  return {
    matches: canonicalJSON(expectedIdentity) === canonicalJSON(observedIdentity),
    expectedIdentityDigest: identityDigest(expectedIdentity),
    observedIdentityDigest: identityDigest(observedIdentity),
  };
}

function omissionInputFactValueDigest(selector, item) {
  if (selector === "QUESTION_BODY_DIGEST") {
    return digestBytes(Buffer.from(item.questionBody, "utf8"));
  }
  if (selector === "FORM_METADATA_DIGEST") {
    return digestValue("AGA-FORM-METADATA-FACT-V1", {
      formKind: item.packageFacts.formKind,
      formRiskBand: item.packageFacts.formRiskBand,
    });
  }
  if (selector === "SOURCE_PROPOSAL_DIGEST") {
    return digestValue(
      "AGA-SOURCE-PROPOSAL-SET-V1",
      item.packageFacts.sourceProposalDigests,
    );
  }
  if (selector === "RESEARCH_ROW_DIGEST") {
    return digestValue("AGA-RESEARCH-ROW-FACT-V1", item.researchCandidateFacts);
  }
  fail("OMISSION_INPUT_FACT_SELECTOR_UNKNOWN");
}

function identityDigest(identity) {
  return digestValue("AGA-HYBRID-DISCOVERY-IDENTITY-V1", identity);
}

function buildOmissionReviewInventory(privateItems, modelSignalBatches = []) {
  const items = [];
  for (const item of privateItems) {
    for (const rule of omissionSignalRules) {
      if (!matchesOmissionSignalRule(rule, item)) continue;
      const itemPayload = {
        identity: item.identity,
        identityDigest: identityDigest(item.identity),
        signalRuleId: rule.signalRuleId,
        cueCode: rule.cueCode,
        inputFactSelector: rule.inputFactSelector,
        inputFactValueDigest: omissionInputFactValueDigest(
          rule.inputFactSelector,
          item,
        ),
      };
      items.push({
        ...itemPayload,
        itemDigest: digestValue("AGA-HYBRID-OMISSION-INVENTORY-ITEM-V1", itemPayload),
      });
    }
  }
  const privateItemByIdentityDigest = new Map(
    privateItems.map((item) => [identityDigest(item.identity), item]),
  );
  const ruleById = new Map(
    omissionSignalRules.map((rule) => [rule.signalRuleId, rule]),
  );
  const inventoryItemByIdentityAndRule = new Map(
    items.map((item) => [
      [item.identityDigest, item.signalRuleId].join("\u0000"),
      item,
    ]),
  );
  const mappingByCandidate = new Map(
    modelSignalMappings.map((mapping) => [
      [
        mapping.candidateSignalRuleId,
        mapping.candidateCueCode,
        mapping.inputFactSelector,
      ].join("\u0000"),
      mapping,
    ]),
  );
  const modelSignalDispositions = [];
  const seenRawSignalDigests = new Set();
  let accepted = 0;
  let rejected = 0;
  for (const batch of modelSignalBatches) {
    if (!Number.isSafeInteger(batch.batchOrdinal) || !Array.isArray(batch.signals)) {
      fail("MODEL_SIGNAL_BATCH_INVALID");
    }
    for (const signal of batch.signals) {
      const rawSignalDigest = digestValue("AGA-HYBRID-MODEL-OMISSION-SIGNAL-V1", {
        batchOrdinal: batch.batchOrdinal,
        signal,
      });
      if (seenRawSignalDigests.has(rawSignalDigest)) fail("MODEL_SIGNAL_DUPLICATE");
      seenRawSignalDigests.add(rawSignalDigest);
      const item = privateItemByIdentityDigest.get(signal.identityDigest);
      if (!item) fail("MODEL_SIGNAL_IDENTITY_UNKNOWN");
      const mapping = mappingByCandidate.get(
        [signal.signalRuleId, signal.cueCode, signal.inputFactSelector].join("\u0000"),
      );
      if (!mapping) fail("MODEL_SIGNAL_MAPPING_UNKNOWN");
      const frozenRule = ruleById.get(mapping.frozenSignalRuleId);
      if (!frozenRule) fail("MODEL_SIGNAL_FROZEN_RULE_UNKNOWN");
      let disposition;
      let rejectionCode;
      let inventoryItemDigestValue = null;
      if (frozenRule.inputFactSelector !== signal.inputFactSelector) {
        disposition = "REJECTED_CANDIDATE_SIGNAL";
        rejectionCode = "INPUT_SELECTOR_MISMATCH";
        rejected += 1;
      } else if (!matchesOmissionSignalRule(frozenRule, item)) {
        disposition = "REJECTED_CANDIDATE_SIGNAL";
        rejectionCode = "FROZEN_RULE_NOT_MATCHED";
        rejected += 1;
      } else {
        const inventoryItem = inventoryItemByIdentityAndRule.get(
          [signal.identityDigest, frozenRule.signalRuleId].join("\u0000"),
        );
        if (!inventoryItem) fail("MODEL_SIGNAL_INVENTORY_ITEM_MISSING");
        disposition = "ACCEPTED_FROZEN_RULE";
        rejectionCode = null;
        inventoryItemDigestValue = inventoryItem.itemDigest;
        accepted += 1;
      }
      modelSignalDispositions.push({
        batchOrdinal: batch.batchOrdinal,
        rawSignalDigest,
        identity: item.identity,
        identityDigest: signal.identityDigest,
        candidateSignalRuleId: signal.signalRuleId,
        candidateCueCode: signal.cueCode,
        inputFactSelector: signal.inputFactSelector,
        inputFactValueDigest: omissionInputFactValueDigest(
          signal.inputFactSelector,
          item,
        ),
        disposition,
        frozenSignalRuleId: frozenRule.signalRuleId,
        inventoryItemDigest: inventoryItemDigestValue,
        rejectionCode,
      });
    }
  }
  const payload = {
    schemaVersion: "aga-hybrid-omission-review-inventory/v1",
    identityCount: privateItems.length,
    signalRuleCount: omissionSignalRules.length,
    signalRules: omissionSignalRules,
    signalRuleDigest: digestValue(
      "AGA-HYBRID-OMISSION-SIGNAL-RULES-V1",
      omissionSignalRules,
    ),
    modelSignalMappingCount: modelSignalMappings.length,
    modelSignalMappings,
    modelSignalMappingDigest: digestValue(
      "AGA-HYBRID-MODEL-SIGNAL-MAPPINGS-V1",
      modelSignalMappings,
    ),
    modelSignalCount: modelSignalDispositions.length,
    modelSignalDispositionCounts: { accepted, rejected },
    modelSignalDispositions,
    modelSignalDispositionDigest: digestValue(
      "AGA-HYBRID-MODEL-SIGNAL-DISPOSITIONS-V1",
      modelSignalDispositions,
    ),
    itemCount: items.length,
    items,
  };
  return {
    ...payload,
    inventoryDigest: digestValue("AGA-HYBRID-OMISSION-INVENTORY-V1", payload),
  };
}

function privateInputItem(document, form, question, researchRow) {
  const identity = baseIdentity(document, form, question);
  return {
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
    researchCandidateFacts: researchRow,
  };
}

function materializePrivateBatch(items, batchOrdinal) {
  return {
    schemaVersion: "aga-hybrid-vocabulary-discovery-input/v1",
    purpose: "VOCABULARY_DISCOVERY_ONLY_NO_ROW_CLASSIFICATION",
    batchOrdinal,
    packageJsonSha256: `sha256:${expected.package.sha256}`,
    researchZipSha256: `sha256:${expected.researchZip.sha256}`,
    workbookSha256: `sha256:${expected.workbook.sha256}`,
    items,
  };
}

function closeBatch(privateItems, batchOrdinal) {
  const payload = materializePrivateBatch(privateItems, batchOrdinal);
  const serialized = canonicalJSON(payload);
  const redacted = {
    batchOrdinal,
    itemCount: privateItems.length,
    inputBytes: Buffer.byteLength(serialized, "utf8"),
    inputDigest: digestValue("AGA-HYBRID-DISCOVERY-INPUT-V1", payload),
    identities: privateItems.map((item) => item.identity),
  };
  return {
    privateItems,
    manifest: {
      ...redacted,
      batchDigest: digestValue("AGA-HYBRID-BATCH-MANIFEST-V1", redacted),
    },
  };
}

const classificationFixedInputDigests = Object.freeze({
  packageJsonSha256: `sha256:${expected.package.sha256}`,
  sealedOverlayLoaderZipSha256:
    "sha256:30700a88aeb5b26514bf7eb76bef050deb08b96294db94117d185de5c9f163b2",
  providerCatalogSha256:
    "sha256:42079b4046542e392c393fe6de1052d84f96938ea163cf63deed5ae9c4b6a789",
  researchZipSha256: `sha256:${expected.researchZip.sha256}`,
  researchQuestionCsvSha256: `sha256:${expected.researchQuestionCSV.sha256}`,
  providerClassificationCsvSha256:
    "sha256:d52a98739db61828c16aa734154be18b11e6ebb358eeeb7f84c3d92a4a5430de",
  ambiguityCsvSha256:
    "sha256:6e97a193f5e12dbe81f87d44d4b22c36ce446a40be7ef0f9fc939e8fbf1e654d",
  workbookSha256: `sha256:${expected.workbook.sha256}`,
  auditChecklistWorkflowSha256:
    "sha256:7dee737c7c5e47e996857e956514a8d46d1a4444234b021cac77cd6cff6b30a2",
  findingCapEvidenceWorkflowSha256:
    "sha256:896f9fa7d498fdc20c582134a15ed6acdc11b78926e655854c43e49fbb24815c",
  productionContractVocabularySha256:
    "sha256:3ef3349d738feb9789aaab6e92246f55948053604a8304706fc1bbd0cd786769",
});

const classificationSizingContract = Object.freeze({
  taxonomyVersion: "AGA_QUESTION_CLASSIFICATION_V1",
  canonicalJSON: "AVIASURVEIL360_CANONICAL_JSON_V1",
  passInputDigestDomain: "AGA-CLASSIFICATION-PASS-INPUT-V1",
  fixedSha256StringBytes: 71,
  passRoles: ["CANDIDATE", "CHALLENGE"],
  classificationRunId: Object.freeze({
    maximumValidAsciiValue: `aga-classification-run-${"a".repeat(64)}`,
    maximumValidAsciiBytes: 87,
  }),
  candidatePassRunId: Object.freeze({
    maximumValidAsciiValue: `aga-classification-pass-candidate-${"a".repeat(64)}`,
    maximumValidAsciiBytes: 98,
  }),
  challengePassRunId: Object.freeze({
    maximumValidAsciiValue: `aga-classification-pass-challenge-${"a".repeat(64)}`,
    maximumValidAsciiBytes: 98,
  }),
});

function sizingDigestPlaceholder() {
  return `sha256:${"0".repeat(classificationSizingContract.fixedSha256StringBytes - 7)}`;
}

function materializeClassificationPassInput(items, batchOrdinal, passRole) {
  return {
    schemaVersion: "aga-hybrid-classification-pass-input/v1",
    purpose: "ROW_CLASSIFICATION_PRIVATE_INPUT",
    classificationRunId:
      classificationSizingContract.classificationRunId.maximumValidAsciiValue,
    passRole,
    passRunId:
      passRole === "CANDIDATE"
        ? classificationSizingContract.candidatePassRunId.maximumValidAsciiValue
        : classificationSizingContract.challengePassRunId.maximumValidAsciiValue,
    batchOrdinal,
    taxonomyVersion: classificationSizingContract.taxonomyVersion,
    taxonomyDigest: sizingDigestPlaceholder(),
    promptDigest: sizingDigestPlaceholder(),
    modelDescriptorDigest: sizingDigestPlaceholder(),
    batchManifestDigest: sizingDigestPlaceholder(),
    fixedInputDigests: classificationFixedInputDigests,
    items,
  };
}

function closeClassificationBatch(privateItems, batchOrdinal) {
  const sourceSnapshot = {
    batchOrdinal,
    fixedInputDigests: classificationFixedInputDigests,
    items: privateItems,
  };
  const candidateCanonicalBytes = Buffer.byteLength(
    canonicalJSON(
      materializeClassificationPassInput(privateItems, batchOrdinal, "CANDIDATE"),
    ),
    "utf8",
  );
  const challengeCanonicalBytes = Buffer.byteLength(
    canonicalJSON(
      materializeClassificationPassInput(privateItems, batchOrdinal, "CHALLENGE"),
    ),
    "utf8",
  );
  const entry = {
    batchOrdinal,
    itemCount: privateItems.length,
    candidateCanonicalBytes,
    challengeCanonicalBytes,
    worstCaseCanonicalBytes: Math.max(candidateCanonicalBytes, challengeCanonicalBytes),
    sourceSnapshotDigest: digestValue(
      "AGA-CLASSIFICATION-SOURCE-SNAPSHOT-V1",
      sourceSnapshot,
    ),
    orderedIdentityDigest: digestValue(
      "AGA-CLASSIFICATION-ORDERED-IDENTITIES-V1",
      privateItems.map((item) => item.identity),
    ),
    identities: privateItems.map((item) => item.identity),
  };
  return {
    privateItems,
    manifest: {
      ...entry,
      batchEntryDigest: digestValue("AGA-CLASSIFICATION-BATCH-ENTRY-V1", entry),
    },
  };
}

function buildClassificationManifest(privateItems, discoveryManifest, maxItems, maxInputBytes) {
  const batches = [];
  let current = [];
  for (const item of privateItems) {
    const candidate = [...current, item];
    const batchOrdinal = batches.length + 1;
    const candidateBytes = Math.max(
      Buffer.byteLength(
        canonicalJSON(
          materializeClassificationPassInput(candidate, batchOrdinal, "CANDIDATE"),
        ),
        "utf8",
      ),
      Buffer.byteLength(
        canonicalJSON(
          materializeClassificationPassInput(candidate, batchOrdinal, "CHALLENGE"),
        ),
        "utf8",
      ),
    );
    if (candidate.length > maxItems || candidateBytes > maxInputBytes) {
      if (current.length === 0) fail("CLASSIFICATION_SINGLE_ITEM_EXCEEDS_LIMIT");
      batches.push(closeClassificationBatch(current, batchOrdinal));
      current = [item];
    } else {
      current = candidate;
    }
  }
  if (current.length > 0) batches.push(closeClassificationBatch(current, batches.length + 1));
  if (
    batches.some(
      (batch) =>
        batch.manifest.itemCount > maxItems ||
        batch.manifest.worstCaseCanonicalBytes > maxInputBytes,
    )
  ) {
    fail("CLASSIFICATION_SINGLE_ITEM_EXCEEDS_LIMIT");
  }
  const manifestPayload = {
    schemaVersion: "aga-hybrid-classification-batch-manifest/v2",
    itemCount: privateItems.length,
    batchCount: batches.length,
    maxItems,
    maxCanonicalBytes: maxInputBytes,
    sizingContract: classificationSizingContract,
    fixedInputDigests: classificationFixedInputDigests,
    discoveryBatchManifestDigest: discoveryManifest.manifestDigest,
    prohibitedDiscoveryInputDigests: discoveryManifest.batches.map(
      (batch) => batch.inputDigest,
    ),
    orderedIdentityDigest: digestValue(
      "AGA-CLASSIFICATION-ORDERED-IDENTITIES-V1",
      privateItems.map((item) => item.identity),
    ),
    batches: batches.map((batch) => batch.manifest),
  };
  return {
    ...manifestPayload,
    manifestDigest: digestValue(
      "AGA-CLASSIFICATION-BATCH-MANIFEST-SET-V2",
      manifestPayload,
    ),
  };
}

function hasExactKeys(value, expectedKeys) {
  return (
    value &&
    typeof value === "object" &&
    !Array.isArray(value) &&
    Object.keys(value).sort().join("\u0000") === [...expectedKeys].sort().join("\u0000")
  );
}

export function validateClassificationManifestOrderedUnion(manifest, expectedIdentities) {
  const manifestKeys = [
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
  ];
  const batchKeys = [
    "batchOrdinal",
    "itemCount",
    "candidateCanonicalBytes",
    "challengeCanonicalBytes",
    "worstCaseCanonicalBytes",
    "sourceSnapshotDigest",
    "orderedIdentityDigest",
    "identities",
    "batchEntryDigest",
  ];
  if (
    !hasExactKeys(manifest, manifestKeys) ||
    !Array.isArray(manifest.batches) ||
    !Array.isArray(expectedIdentities) ||
    expectedIdentities.length !== 1310
  ) {
    fail("CLASSIFICATION_MANIFEST_ORDERED_UNION_CONTRACT_INVALID");
  }
  const expectedIdentityKeys = new Set(expectedIdentities.map((identity) => canonicalJSON(identity)));
  if (expectedIdentityKeys.size !== 1310) {
    fail("CLASSIFICATION_MANIFEST_EXPECTED_IDENTITIES_INVALID");
  }
  const flattened = [];
  const seen = new Set();
  for (let index = 0; index < manifest.batches.length; index += 1) {
    const batch = manifest.batches[index];
    if (!hasExactKeys(batch, batchKeys) || !Array.isArray(batch.identities)) {
      fail("CLASSIFICATION_MANIFEST_ORDERED_UNION_CONTRACT_INVALID");
    }
    if (batch.batchOrdinal !== index + 1) {
      fail("CLASSIFICATION_MANIFEST_BATCH_ORDINAL_INVALID");
    }
    for (const identity of batch.identities) {
      const key = canonicalJSON(identity);
      if (seen.has(key)) fail("CLASSIFICATION_MANIFEST_IDENTITY_DUPLICATE");
      seen.add(key);
      flattened.push(identity);
    }
  }
  if (flattened.length !== 1310 || seen.size !== 1310) {
    fail("CLASSIFICATION_MANIFEST_IDENTITY_COUNT_INVALID");
  }
  if (canonicalJSON(flattened) !== canonicalJSON(expectedIdentities)) {
    fail("CLASSIFICATION_MANIFEST_ORDERED_UNION_MISMATCH");
  }
}

export function prepareClassificationBatches({
  packagePath,
  researchZipPath,
  workbookPath,
  maxItems,
  maxInputBytes,
}) {
  if (!Number.isSafeInteger(maxItems) || maxItems < 1 || maxItems > 64) {
    fail("MAX_ITEMS_INVALID");
  }
  if (
    !Number.isSafeInteger(maxInputBytes) ||
    maxInputBytes < 1024 ||
    maxInputBytes > 98304
  ) {
    fail("MAX_INPUT_BYTES_INVALID");
  }
  const packageFact = fileFact(packagePath, expected.package, "PACKAGE");
  const researchFact = fileFact(researchZipPath, expected.researchZip, "RESEARCH_ZIP");
  const workbookFact = fileFact(workbookPath, expected.workbook, "WORKBOOK");
  const document = parsePackage(packagePath);
  const researchRows = parseCSV(
    readZipEntry(
      researchZipPath,
      "question_level_review.csv",
      expected.researchQuestionCSV,
    ).toString("utf8"),
  );
  if (researchRows.length !== 1310) fail("RESEARCH_COUNT_MISMATCH");
  const researchByIdentity = new Map();
  for (const row of researchRows) {
    const key = researchIdentity(row);
    if (researchByIdentity.has(key)) fail("RESEARCH_IDENTITY_DUPLICATE");
    researchByIdentity.set(key, row);
  }

  const privateItems = [];
  const seenIdentities = new Set();
  for (const form of document.forms) {
    for (const question of form.questions) {
      const identity = baseIdentity(document, form, question);
      const key = identityKey(identity);
      if (seenIdentities.has(key)) fail("PACKAGE_IDENTITY_DUPLICATE");
      seenIdentities.add(key);
      const researchRow = researchByIdentity.get(key);
      if (!researchRow) fail("RESEARCH_IDENTITY_MISSING");
      const observedIdentity = {
        packageVersion: document.packageVersion,
        packageJsonSha256: `sha256:${expected.package.sha256}`,
        formCode: researchRow.form_code,
        proposalId: researchRow.proposal_id,
        ordinal: Number(researchRow.ordinal),
        textDigest: researchRow.text_digest,
      };
      if (!completeIdentityValidation(identity, observedIdentity).matches) {
        fail("RESEARCH_IDENTITY_MISMATCH");
      }
      privateItems.push(privateInputItem(document, form, question, researchRow));
    }
  }
  if (privateItems.length !== 1310 || seenIdentities.size !== 1310) {
    fail("PACKAGE_IDENTITY_COUNT_MISMATCH");
  }

  const batches = [];
  let current = [];
  for (const item of privateItems) {
    const candidate = [...current, item];
    const batchOrdinal = batches.length + 1;
    const bytes = Buffer.byteLength(
      canonicalJSON(materializePrivateBatch(candidate, batchOrdinal)),
      "utf8",
    );
    if (candidate.length > maxItems || bytes > maxInputBytes) {
      if (current.length === 0) fail("SINGLE_ITEM_EXCEEDS_LIMIT");
      batches.push(closeBatch(current, batchOrdinal));
      current = [item];
      const singleBytes = Buffer.byteLength(
        canonicalJSON(materializePrivateBatch(current, batches.length + 1)),
        "utf8",
      );
      if (singleBytes > maxInputBytes) fail("SINGLE_ITEM_EXCEEDS_LIMIT");
    } else {
      current = candidate;
    }
  }
  if (current.length > 0) batches.push(closeBatch(current, batches.length + 1));

  const manifestPayload = {
    schemaVersion: "aga-hybrid-classification-batch-manifest/v1",
    packageFact,
    researchZipFact: researchFact,
    workbookFact,
    packageOrder: expectedFormCodes(),
    itemCount: privateItems.length,
    maxItems,
    maxInputBytes,
    orderedIdentityDigest: digestValue(
      "AGA-HYBRID-ORDERED-IDENTITIES-V1",
      privateItems.map((item) => item.identity),
    ),
    batches: batches.map((batch) => batch.manifest),
  };
  const manifest = {
    ...manifestPayload,
    manifestDigest: digestValue("AGA-HYBRID-BATCH-MANIFEST-SET-V1", manifestPayload),
  };
  const classificationManifest = buildClassificationManifest(
    privateItems,
    manifest,
    maxItems,
    maxInputBytes,
  );
  validateClassificationManifestOrderedUnion(
    classificationManifest,
    privateItems.map((item) => item.identity),
  );
  return {
    manifest,
    classificationManifest,
    batches,
    document,
    privateItems,
  };
}

function parseArguments(argv) {
  const values = new Map();
  for (let index = 0; index < argv.length; index += 2) {
    const name = argv[index];
    const value = argv[index + 1];
    if (!name?.startsWith("--") || value === undefined) fail("ARGUMENTS_INVALID");
    if (values.has(name)) fail("ARGUMENT_DUPLICATE");
    values.set(name, value);
  }
  const allowed = new Set([
    "--package",
    "--research-zip",
    "--workbook",
    "--max-items",
    "--max-input-bytes",
    "--output",
    "--classification-output",
    "--diagnostic-probe",
    "--discovery-run",
  ]);
  if ([...values.keys()].some((key) => !allowed.has(key))) fail("ARGUMENT_UNKNOWN");
  for (const required of [
    "--package",
    "--research-zip",
    "--workbook",
    "--max-items",
    "--max-input-bytes",
    "--output",
  ]) {
    if (!values.has(required)) fail("ARGUMENT_REQUIRED");
  }
  return values;
}

function run(argv) {
  const argumentsMap = parseArguments(argv);
  const result = prepareClassificationBatches({
    packagePath: argumentsMap.get("--package"),
    researchZipPath: argumentsMap.get("--research-zip"),
    workbookPath: argumentsMap.get("--workbook"),
    maxItems: Number(argumentsMap.get("--max-items")),
    maxInputBytes: Number(argumentsMap.get("--max-input-bytes")),
  });
  if (argumentsMap.get("--diagnostic-probe") === "real-derived-identity-mismatch") {
    const expectedIdentity = result.privateItems[0].identity;
    const observedIdentity = {
      ...expectedIdentity,
      proposalId: `${expectedIdentity.proposalId}-diagnostic-mismatch`,
    };
    const validation = completeIdentityValidation(expectedIdentity, observedIdentity);
    if (validation.matches) fail("DIAGNOSTIC_PROBE_DID_NOT_MISMATCH");
    const probeDigest = digestValue("AGA-HYBRID-DIAGNOSTIC-PROBE-V1", {
      manifestDigest: result.manifest.manifestDigest,
      mismatchCount: 1,
      expectedIdentityDigest: validation.expectedIdentityDigest,
      observedIdentityDigest: validation.observedIdentityDigest,
    });
    process.stderr.write(
      `ERR_IDENTITY_MISMATCH batch=1 count=1 digest=${probeDigest}\n`,
    );
    process.exitCode = 2;
    return;
  }
  if (argumentsMap.has("--diagnostic-probe")) fail("DIAGNOSTIC_PROBE_UNKNOWN");
  const output = resolve(argumentsMap.get("--output"));
  mkdirSync(dirname(output), { recursive: true });
  writeFileSync(output, `${JSON.stringify(result.manifest, null, 2)}\n`, {
    encoding: "utf8",
    mode: 0o600,
  });
  if (argumentsMap.has("--classification-output")) {
    const classificationOutput = resolve(argumentsMap.get("--classification-output"));
    mkdirSync(dirname(classificationOutput), { recursive: true });
    writeFileSync(
      classificationOutput,
      `${JSON.stringify(result.classificationManifest, null, 2)}\n`,
      { encoding: "utf8", mode: 0o600 },
    );
  }
  const omissionOutput = resolve(dirname(output), "omission-review-inventory.json");
  let modelSignalBatches = [];
  if (argumentsMap.has("--discovery-run")) {
    let discovery;
    try {
      discovery = JSON.parse(readFileSync(argumentsMap.get("--discovery-run"), "utf8"));
    } catch {
      fail("DISCOVERY_RUN_INVALID");
    }
    if (
      discovery?.schemaVersion !== "aga-hybrid-vocabulary-discovery-run/v1" ||
      !Array.isArray(discovery.batches)
    ) {
      fail("DISCOVERY_RUN_CONTRACT_MISMATCH");
    }
    modelSignalBatches = discovery.batches.map((batch) => {
      if (!Array.isArray(batch?.output?.omissionSignals)) {
        fail("DISCOVERY_RUN_SIGNALS_MISSING");
      }
      return {
        batchOrdinal: batch.batchOrdinal,
        signals: batch.output.omissionSignals,
      };
    });
  }
  writeFileSync(
    omissionOutput,
    `${JSON.stringify(
      buildOmissionReviewInventory(result.privateItems, modelSignalBatches),
      null,
      2,
    )}\n`,
    { encoding: "utf8", mode: 0o600 },
  );
  if (argumentsMap.has("--classification-output")) {
    process.stdout.write(
      `aga-hybrid-batches: ok discoveryBatches=${result.manifest.batches.length} classificationBatches=${result.classificationManifest.batches.length} items=${result.manifest.itemCount}\n`,
    );
  } else {
    process.stdout.write(
      `aga-hybrid-batches: ok batches=${result.manifest.batches.length} items=${result.manifest.itemCount}\n`,
    );
  }
}

if (
  process.argv[1] &&
  fileURLToPath(import.meta.url) === resolve(process.argv[1])
) {
  try {
    run(process.argv.slice(2));
  } catch (error) {
    const code = error instanceof ControlledError ? error.code : "UNEXPECTED";
    process.stderr.write(`ERR_GATE0_PREPARE code=${code}\n`);
    process.exitCode = 1;
  }
}
