# AGA Hybrid Row Classification Prompt V1

Status: frozen `candidate-only` input for the two local classification passes.
This prompt cannot establish source authority, technical approval, publication,
compliance, a Finding, enforcement, release, or production readiness.

Taxonomy version: `AGA_QUESTION_CLASSIFICATION_V1`

Taxonomy digest:
`sha256:40517b48d0820db221501f89ff7fe58b120c6674e905cd722231d0b35ba18222`

## Pass isolation

`passRole is exactly CANDIDATE or CHALLENGE`. A run receives one role and one
bounded batch. The `CANDIDATE` pass proposes the complete controlled projection.
The blind `CHALLENGE` pass independently proposes the same shape. Neither pass
may see the other pass's input root, output sink, transcript, receipt, result,
digest, disagreement, or reconciled candidate. Do not infer a missing value
from the role or from any earlier batch.

## Input

The input is one closed private classification-manifest batch. The separate
24-batch vocabulary-discovery manifest is immutable and its discovery input
digests are prohibited here. The classification manifest has exactly 25 ordered
batches over the same 1,310 identities. `batchManifestDigest` is exactly
`fixedInputs.classificationBatchManifestDigest`; the distinct
`fixedInputs.discoveryBatchManifestDigest` is never valid for a classification
pass. The input is one closed private
`aga-hybrid-classification-pass-input/v1` batch with at most 64 items and at
most 98,304 canonical UTF-8 bytes. Its `inputDigest` is reconstructed as
SHA-256 over `AGA-CLASSIFICATION-PASS-INPUT-V1` immediately followed by the
canonical JSON of the complete input. This is the per-pass-batch input digest;
every record in the batch and the enclosing pass-batch output repeat that exact
digest. It is not the shared `AGA-CLASSIFICATION-RUN-INPUT-V1` digest later
stored on sealed items and run records. Every item supplies:

- one complete six-field Base identity;
- the question body in isolated model context only;
- bounded form and question facts;
- digests of supplied source proposals and references;
- the matching independent-research row facts; and
- the taxonomy and the prompt/run configuration declared by the pass receipt.

The public classification manifest directly binds the accepted discovery
manifest root and separately retains a non-authoritative sizing contract. That
sizing contract specifies byte widths, canonicalization, roles, and maximum
valid ASCII run IDs only; it never presents placeholder digest strings as real
taxonomy, prompt, manifest, model-descriptor, or runtime pins.

## Platform metadata provenance

`CHAT_METADATA.json` is a closed, text-free platform-availability receipt. It
contains `modelId`, `service`, `interface`, `snapshotBuildLabel`,
`displayedModelLabel`, `requestedReasoningEffort`, `forkTurns`, and
`unavailableFields`. Every unavailable scalar is literal JSON `null` and its
exact field name is in the UTF-8-bytewise sorted `unavailableFields` array;
every available scalar is non-null and absent from that array. Do not infer,
fill, transform, or omit unavailable metadata. `displayedModelLabel` is only a
displayed platform label and never establishes an exact `modelId`.

Truthful platform-unavailable metadata is valid `candidate-only` demo
provenance. It does not establish production model provenance, release, or
production readiness. The validator derives `modelIdSource` as
`platform-unavailable` when `modelId` is null and
`platform-reported-exact` only when the platform actually supplies a distinct
exact model ID.

## Immutable receipt provenance

For a supplied sealed-pass ZIP, `promptDigest` and `modelDescriptorDigest` in
`PASS_RUN_RECEIPT.json`, every batch, every record, and the pass seal are
immutable source-receipt facts. The validator requires canonical SHA-256
syntax, equality of the two pass prompt digests, and the complete
identity/batch/record/batch-output/pass-seal graph. It does not rebind a
supplied receipt to the former repository prompt hash or require the supplied
model receipt digest to equal a locally recomputed descriptor digest. Missing
platform metadata remains literal `null` with its exact availability marker;
the validator never invents a model ID, reasoning setting, fork setting, or
platform claim.

The complete identity is the tuple `packageVersion`, `packageJsonSha256`,
`formCode`, `proposalId`, `ordinal`, and `textDigest`. A text digest, proposal
ID, ordinal, or form code alone is never identity. Input facts are evidence,
not authority.

## Required controlled proposal

For every supplied identity, return one complete proposal projection containing:

- exactly one `mainDomainCode`;
- set-like `topicCodes`, `inspectionProfileCodes`, and
  `inspectionTypeCodes`;
- one V1-eligible `canonicalTargetKind` and one compatible
  `targetProfileCode`; `PERSON` remains in the seven-code vocabulary but is
  excluded from V1 proposals because V1 has no controlled compatible Person
  target profile;
- set-like closed `operationQualifiers` and `activityQualifiers`;
- one `applicabilityDisposition`;
- set-like `evidenceExpectationCodes`;
- zero or more independent `externalInvolvements`;
- controlled top-level `rationaleCodes`, `confidenceEvidence`, and `sourceRefs`;
  and no governance field or blocker.

Treat supplied governance states and blockers as immutable validator input.
Do not output governance fields, clear them, upgrade them, or infer replacements
inside a pass record; the deterministic validator copies the supplied values
into the final sealed item.

Use only codes in the frozen taxonomy file. Do not mint, approximate, alias, or
extend a code. Do not create a provider ownership, primary, secondary,
supporting, assignment, responsibility-transfer, or inspected-scope field.
External involvement is optional and independent. Duplicated research role
columns are one candidate fact, never two edges.
`providerPartition.externalInvolvementAllowed` is the exact provider-code set
permitted on an edge. The inspected `AERODROME_OPERATOR` scope and every
`NO_DEFAULT_AGA_RELATIONSHIP` provider code are forbidden on an external edge.

## Evidence binding

For `HIGH`, every non-edge proposal emitted by the pass must have its own closed
evidence tuple:

- `proposalField` names the exact proposal-bearing field;
- `proposalValueDigest` binds the normalized scalar, set member, or qualifier
  pair;
- `rationaleCode` is controlled;
- `inputFactSelector` is controlled;
- `inputFactValueDigest` binds the exact supplied fact; and
- `signalRuleId` is present only for a validator-recomputed frozen rule.

Each external-involvement edge carries its own rationale, evidence, sources,
and blockers. Edge evidence binds the canonical digest of that edge's full
semantic tuple. Evidence cannot be reused for another field, value, or edge.
Evidence omission is structurally valid and deterministically lowers confidence;
it is not permission to invent a tuple. Invented, unknown, mismatched, or
cross-field evidence is a fatal pass error. A lexical signal cannot create or
fully evidence an external edge by itself.

## Output

Return one JSON object with schema
`aga-hybrid-classification-pass-batch/v1` and exactly:

- `classificationRunId`;
- `schemaVersion`, exactly `aga-hybrid-classification-pass-batch/v1`;
- `passRole`;
- `passRunId`;
- `batchOrdinal`;
- `promptDigest`;
- `modelDescriptorDigest`;
- `inputDigest`;
- `records`; and
- `batchOutputDigest`.

`records` is ordered by the input batch and contains exactly one closed
`passProposalRecord` per complete identity. Each record contains the complete
identity, run/pass provenance, one complete `proposalProjection`, controlled
top-level rationale/evidence/sources, and `passResultDigest`. Do not return a
final confidence, recommendation, reconciliation decision, Manager decision,
approval, or publication state. Those are validator-derived or outside the
classification pass.

Normalize every set-like array by the frozen UTF-8 bytewise tuple and reject
duplicates before hashing. Preserve ordered identity and batch arrays exactly.
Canonical JSON uses the `AVIASURVEIL360_CANONICAL_JSON_V1` profile: UTF-8,
recursively UTF-8-bytewise key-sorted objects, compact minimal base-10 finite
integers with no negative zero, JSON control/quote/backslash escaping,
no HTML escaping, no solidus escaping, literal UTF-8 for non-ASCII scalars,
no Unicode normalization, and rejection of lone surrogates. A validator signal's
`identityDigest` always uses `AGA-CLASSIFICATION-BASE-IDENTITY-V1` over the
complete six-field Base identity. `passResultDigest` is
SHA-256 over `AGA-CLASSIFICATION-PASS-PROPOSAL-V1` followed by canonical JSON
of the complete record excluding only `passResultDigest`.
`batchOutputDigest` is SHA-256 over `AGA-CLASSIFICATION-PASS-BATCH-V1`
immediately followed, with no delimiter, by canonical JSON of the complete
batch output excluding only `batchOutputDigest`.

## Fatal conditions

Return only `{"errorCode":"CLASSIFICATION_INPUT_REJECTED"}` for malformed or
over-bound input, unknown taxonomy content, duplicate or incomplete identity,
an unsupported role, or a request to see another pass. The repository
validator rejects the whole pass for missing/extra/duplicate identities,
unknown or forbidden fields/codes, malformed edge provenance, evidence/fact or
signal-rule mismatch, text leakage, aggregate mismatch, or digest mismatch.
These conditions never become a row-level blocker.

## Privacy

Never output question text, a body fragment, or chain of thought. Never output
a private path, source title, URL, raw response, transcript, hidden reasoning,
or free narrative. The complete Base identity is permitted only in the sealed
text-free result record required for exact 1,310-row reconciliation. It must not
appear in stdout, stderr, URLs, telemetry, screenshots, videos, or retained
test artifacts.
