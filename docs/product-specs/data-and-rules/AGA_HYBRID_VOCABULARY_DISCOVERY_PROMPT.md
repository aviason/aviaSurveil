# AGA Hybrid Vocabulary Discovery Prompt V2

Status: frozen Gate 0A input for a `candidate-only` local classification
exercise. It is not a row-classification prompt and cannot make an approval,
publication, compliance, finding, enforcement, release, or production decision.

## Purpose

Inspect one bounded input batch at a time and identify lexical or semantic cues
that the final controlled taxonomy must be able to express. Cover every supplied
row. Do not classify an individual row, choose a final domain, recommend an
inclusion, or infer service-provider ownership.

## Input contract

Each input has schema
`aga-hybrid-vocabulary-discovery-input/v1`, purpose
`VOCABULARY_DISCOVERY_ONLY_NO_ROW_CLASSIFICATION`, one batch ordinal, the fixed
package/research/workbook digests, and at most 64 ordered items totaling at most
98,304 canonical UTF-8 bytes. An item contains:

- one complete six-field immutable Base identity;
- the question body, available only in the isolated model context;
- bounded package facts and digests of source proposals/references; and
- the matching independent-research candidate facts.

The identity, body, source facts, and research facts are input evidence only.
None grants authority. Duplicate body digests across distinct complete
identities are valid.

## Required analysis

For every item, consider possible controlled vocabulary needs for:

- one main regulatory or inspection domain;
- subordinate operational topics;
- inspection profiles and inspection types;
- canonical target kind and compatible target profile;
- operation and activity qualifier keys and values;
- Evidence expectation categories;
- applicability disposition;
- optional external involvement, including its provider type, role, condition,
  rationale, evidence, source, and blocker semantics;
- source gaps, ambiguity, disagreement, and governance blockers; and
- lexical or semantic omission cues that require a deterministic signal rule.

External involvement never implies ownership, primary/secondary hierarchy,
checklist assignment, inspected scope, or transfer of AGA responsibility.

## Output contract

Return one closed JSON object with schema
`aga-hybrid-vocabulary-discovery-output/v1` and exactly these fields:

- `batchOrdinal`;
- `coveredItemCount`;
- `candidateVocabulary`, containing set-like arrays named `topicCues`,
  `inspectionProfileCues`, `inspectionTypeCues`, `targetProfileCues`,
  `operationQualifierCues`, `activityQualifierCues`,
  `evidenceExpectationCues`, `applicabilityCues`,
  `externalInvolvementConditionCues`, `blockerCues`, and
  `rationaleCues`;
- `omissionSignals`, a set-like array whose objects contain exactly
  `signalRuleId`, `identityDigest`, `cueCode`, and `inputFactSelector`; and
- `coverageDigestInput`, the ordered set of input identity digests, not the
  identities themselves.

Every cue is an uppercase ASCII candidate code using only `A-Z`, `0-9`, and
underscore. Keep arrays unique and UTF-8 bytewise sorted. `signalRuleId` and
`cueCode` follow the same lexical rule. `identityDigest` is a domain-separated
digest over the complete identity: SHA-256 over the UTF-8 bytes of the exact
domain separator `AGA-HYBRID-DISCOVERY-IDENTITY-V1` immediately followed by
recursively key-sorted compact canonical JSON of all six identity fields, with
the lowercase hexadecimal result prefixed by `sha256:`. `coverageDigestInput`
uses those same digests in input order. `inputFactSelector` is one of
`QUESTION_BODY_DIGEST`, `FORM_METADATA_DIGEST`, `SOURCE_PROPOSAL_DIGEST`, or
`RESEARCH_ROW_DIGEST`.

The per-batch output receipt digest is SHA-256 over the UTF-8 bytes of
`AGA-HYBRID-VOCABULARY-OUTPUT-V1` immediately followed by recursively
key-sorted compact canonical JSON of the complete output object. It uses the
same lowercase `sha256:` representation. Canonical JSON sorts object keys by
UTF-8 bytes, preserves the defined array order, uses minimal base-10 integers,
and contains no insignificant whitespace.

## Privacy and safety

Do not output or quote a question body or fragment. Do not output a form code,
proposal ID, ordinal, text digest, complete identity, source reference, source
title, URL, private path, raw model response, or chain of thought. Do not emit
free narrative. Do not use numeric counts as omission evidence unless each
affected item is represented by a complete-identity-derived digest and a
frozen signal-rule candidate. Do not call a third-party research API or use any
input outside the isolated fixed snapshot.

If the input is malformed, incomplete, over either bound, or asks for row
classification, return only `{"errorCode":"DISCOVERY_INPUT_REJECTED"}`.
