# AGA Hybrid Question Classification Contract

Status: frozen version 1 contract for a local, synthetic, `candidate-only`
workspace. It does not establish source authority, technical approval,
publication, compliance, enforcement, release, or production readiness.

Machine-readable authorities:

- taxonomy: `aga-question-classification-taxonomy.v1.json`;
- schema: `aga-question-classification.schema.json`;
- row prompt: `AGA_HYBRID_CLASSIFICATION_PROMPT.md`; and
- discovery receipts: `deliverables/aga-question-classification-contract-v1/`.

The immutable accepted AGA package remains the question-body authority. This
contract never alters that package or the sealed Admin-only AGA overlay.

## Fixed scope and identity

The base run contains exactly 1,310 rows in the accepted 52-form package order.
Every row is identified only by the complete tuple `packageVersion`,
`packageJsonSha256`, `formCode`, `proposalId`, `ordinal`, and `textDigest`.
Duplicate body/text digests across distinct tuples are valid. A digest,
proposal ID, ordinal, form code, or later workspace `rootSequence` is never a
substitute for complete identity.

Every base row receives exactly one controlled main domain and set-like
controlled topics. Taxonomy version `AGA_QUESTION_CLASSIFICATION_V1` freezes 18
main domains, topics, inspection profiles/types, the seven canonical target
kinds, target profiles and compatibility, qualifier keys/values, Evidence
expectations, applicability dispositions, external-involvement semantics,
evidence/rationale/blocker/source/disagreement codes, statuses, reasons, and
lifecycle codes. Classification cannot add a code. A new code requires a
successor taxonomy and run.

Canonical target kinds are `ORGANIZATION`, `PERSON`, `FACILITY`, `DEVICE`,
`SYSTEM`, `ASSET`, and `LOCATION`. Target profile is separate and must be
compatible. `PERSON` is deliberately ineligible for V1 proposal output because
V1 freezes no compatible controlled Person target profile; the other six kinds
each have at least one compatible target profile. The current inspected provider scope is only the synthetic
`AERODROME_OPERATOR` scope. The other connected provider codes can occur only
as an independently evidenced external-involvement edge. No field may model a
primary, secondary, supporting, ownership, or provider hierarchy.

## Two-pass sealed run

One `CANDIDATE` and one blind `CHALLENGE` pass process the identical 25-batch
classification manifest in separately isolated contexts. The accepted 24-batch
vocabulary-discovery manifest remains a separate immutable discovery receipt;
its input digests are prohibited from classification reuse. Each classification
batch is package-order greedy over the same 1,310 source items and is sized
against the complete private pass-input envelope at maximum valid ASCII run-ID
lengths for both roles, with canonical JSON only and fixed-width SHA string
widths. The retained `sizingContract` is a non-authoritative byte-width
template: it has no taxonomy/prompt/manifest/model digest or model metadata
pin. Each
private batch conforms to the
closed `aga-hybrid-classification-pass-input/v1` schema and hashes the exact
body/fact/run payload with `AGA-CLASSIFICATION-PASS-INPUT-V1`; a discovery-only
input digest is never reused as a classification digest. Each pass retains its
closed tool-reported model-descriptor preimage and independently seals exactly one
complete text-free `passProposalRecord` for every full identity, for 2,620
records total. Neither pass can see the other pass, and neither repository code
nor the runtime application calls an LLM.

For supplied sealed-pass ZIPs, the prompt and model descriptor digests carried
by each immutable receipt remain source provenance. The validator checks their
lexical SHA-256 form, prompt equality across roles, and the complete
receipt/batch/record/pass-seal graph without rebinding either digest to the
former repository prompt hash or to a locally recomputed descriptor digest.
Truthful unavailable platform metadata remains candidate-only provenance and
is never filled or promoted into an exact model identity.

Model provenance is availability-aware. Every platform-unavailable scalar is
stored as literal `null` with its exact field name in sorted
`unavailableFields`; every available scalar is non-null and absent from that
array. A displayed model label is retained separately for user-visible
provenance but never establishes an exact `modelId`. Truthful unavailable
platform metadata is accepted for this local synthetic `candidate-only` demo
and does not block deterministic Task 2 validation. It is not a production
model-provenance, release, or production-readiness claim.

`fixedInputs.discoveryBatchManifestDigest` identifies only the retained
vocabulary-discovery receipt. `fixedInputs.classificationBatchManifestDigest`
identifies the 25-batch manifest and is the only manifest digest permitted in
each classification pass, pass seal, or classification run.

The classification manifest itself directly binds
`discoveryBatchManifestDigest` to the accepted discovery manifest root digest,
in addition to its ordered prohibited discovery input-digest list.

Every pass record carries its full identity and run/pass/prompt/model/input
pins, one complete proposal projection, controlled rationale/evidence/sources,
and `passResultDigest`. The sealed final item resolves both records by complete
identity plus run and result digest. The final editable proposal is the exact
normalized candidate-pass projection; challenge-only values remain in the
immutable challenge record. The validator derives disagreements and confidence
without rewriting either pass.

Unknown fields/codes, missing or duplicate identities, incomplete passes,
mismatched facts/rules/provenance/digests, malformed edge evidence, body/text
leakage, or aggregate mismatch reject the whole run. Such defects never become
a row-level blocker. Only `SEALED` runs are readable; failed runs are
`REJECTED` and expose no classification rows.

## Proposal and evidence contract

A complete proposal has one main domain, set-like topic/profile/type arrays,
one compatible target kind/profile, closed qualifier pairs, one applicability
disposition, set-like Evidence expectations, and zero or more independent
external-involvement edges.

For a fully evidenced proposal and therefore for `HIGH`, every emitted non-edge
scalar, set member, or qualifier pair has an independent closed evidence tuple
containing:

- the exact lower-camel `proposalField`;
- the digest of that normalized exact proposal value;
- a controlled rationale code;
- an exact input-fact selector and fact digest; and
- a frozen `signalRuleId` only when the validator recomputes that rule.

Evidence omission is structurally valid but deterministically lowers confidence;
invented or mismatched evidence is fatal. An evidence tuple cannot support
another field or value. Each involvement edge
contains exactly provider type, involvement role, condition, applicability,
edge-specific rationales/evidence/sources/blockers, and binds evidence to the
edge's complete semantic tuple. Empty involvement is valid. Edges never create
ownership, assignment, inspected scope, or transfer of AGA responsibility.

Set-like arrays reject semantic duplicates before UTF-8 bytewise tuple sorting.
Package forms, question identities, batches, aggregate item digests, Draft
revisions, lifecycle revisions, and event streams preserve defined order.

## Confidence and recommendation

The core proposal set is main domain, canonical target kind, target profile,
applicability, and each inspection-profile member. The auxiliary set is every
topic, inspection type, operation/activity qualifier, Evidence expectation,
and external-involvement edge.

Confidence is deterministic and total:

1. `LOW` when either pass disagrees on a core proposal or lacks valid
   field/value evidence for any emitted core proposal.
2. `MEDIUM` when core proposals agree and are fully evidenced, but an auxiliary
   proposal disagrees or lacks its own evidence.
3. `HIGH` only when both complete projections agree and every emitted proposal
   in both passes has valid independent evidence.

Governance blockers remain immutable regardless of agreement. In particular,
all rows retain source mapping, non-attestation, candidate interpretation, and
decision/extraction states supplied by the package. The 49 source-proposal gaps
and 51 external-applicability unresolved identities remain explicit.

Recommendation precedence is also total: a question source-proposal gap gives
`BLOCKED_SOURCE_GAP`; otherwise unresolved external applicability, any
disagreement, `MEDIUM`, or `LOW` gives `MANAGER_REVIEW_REQUIRED`; otherwise
`HIGH` gives `AUTO_PROPOSED_HIGH_CONFIDENCE`. Auto-proposed means only a working
Draft default. It is never approval or publication.

## Question references and immutable Drafts

`questionOrigin` discriminates the closed `questionRef` union.

- `SEALED_BASE` contains exactly the discriminator and six Base identity
  fields.
- `WORKSPACE` contains exactly generation, server-issued root/version/proposal
  IDs, append-only `rootSequence`, `bodyDigest`, closed `parentQuestionKey`,
  actor, canonical UTC creation time, and controlled reason.

The closed nullable `parentQuestionKey` union is null only for a first
`ADD_CANDIDATE`; a first Base reword has the exact Base key; a later reword has
the exact earlier Workspace key without copied provenance. Parent chains are
same-generation/root/sequence, current-leaf, acyclic, and immutable.

`ADD_CANDIDATE` accepts no client identity and allocates a new root, version,
proposal, and generation-unique sequence. A first Base reword allocates a new
Workspace root/version/proposal while retaining the Base package position. A
Workspace reword preserves root and sequence, allocates fresh version/proposal
IDs, and replaces only the current leaf. New or changed wording always starts a
new source gap and cannot inherit a parent `HIGH` result.

Every Manager change creates one immutable successor Draft under exact
generation/revision/content-digest CAS. Semantic edits demote the successor to
null Draft confidence, `PENDING_MANAGER_REVIEW`, null disposition, and either
`MANAGER_REVIEW_REQUIRED` or `BLOCKED_SOURCE_GAP`. A separate `INCLUDE`,
`EXCLUDE`, or `DEFER` is required. Only the current `INCLUDE` leaf can enter a
recommendation.

`RESOLVE_CLASSIFICATION_PROPOSALS` has exactly three modes:

- `ACCEPT_CANDIDATE_PASS` server-copies the immutable candidate projection;
- `ACCEPT_CHALLENGE_PASS` server-copies the immutable challenge projection; or
- `SET_EXACT` supplies one complete closed taxonomy-valid projection.

All require a controlled reason and a later disposition. Batch preview and
execution are atomic, limited to 500 items, and bind the canonical filter,
sorted full-identity digest, current Draft pin, count, expiry, and preview
digest. Execution recomputes the set in one transaction.

Readiness is only `READY_FOR_DEMO_SIMULATION`. It requires one active generation
pinned to the exact 1,310-row sealed run, a current closed Draft with no pending
review, explicit dispositions for every non-default or edited item, complete
profile/target/qualifier compatibility, and preserved blockers. No technical
approval or publication command exists.

## Deterministic recommendation

The Manager supplies exact organization, provider-scope root/ID/version,
provider type ID, target ID/kind/profile, inspection profile/type, department,
unit, operation/activity qualifiers, effective time, taxonomy/run/Draft pins,
readiness pin, and generic operation/idempotency/generation envelope. The server
derives provider code, checks exact compatibility and qualifier completeness,
reconstructs one current leaf per root, and selects only current included
leaves. Base order is preserved; added roots follow append-only sequence;
rewords replace the leaf without changing root position.

Missing, extra, inactive, ambiguous, stale, mismatched, or under-qualified
input fails closed and creates no recommendation. Inspection creation pins the
exact recommendation and current exact Inspector/Lead workspace bindings. It
creates no canonical assignment.

## Synthetic lifecycle invariants

The lifecycle is synthetic and append-only:

`DEMO_PROVIDER_SCOPE -> DEMO_INSPECTION -> DEMO_CHECKLIST_RESPONSE ->`
`DEMO_POTENTIAL_FINDING -> DEMO_FINDING -> DEMO_CAP_REVISION ->`
`DEMO_EVIDENCE_VERSION -> DEMO_VERIFICATION_DECISION -> DEMO_CLOSURE`.

Only the assigned Inspector executes and creates a Potential Finding. Only the
exact Lead returns, dismisses, or converts it. Conversion records independent
human CAP, Evidence, and due-date requirements. Auditee CAP/Evidence versions
are append-only. CAA review keeps `commentToAuditee` separate from
`internalCaaNote`, and Auditee projections structurally omit the latter.
The taxonomy's operation registry freezes source and target states separately
for every affected entity; pooled entity/state cross-products are invalid.
A returned Potential Finding is corrected only through
`CREATE_POTENTIAL_FINDING`: after a corrected `NON_COMPLIANT` or `OBSERVATION`
response, it preserves the existing root ID, appends the next immutable version
from `RETURNED` to `PENDING_LEAD_REVIEW`, retains the prior returned version,
and pins the exact current corrected response revision and semantic digest. The
corrected revision must be strictly newer and its semantic digest must differ
from the response bound to the returned version. `SUBMIT_CHECKLIST` is denied
with `RETURNED_ROOT_SUCCESSOR_REQUIRED` while any latest root remains
`RETURNED`; it is permitted only after the successor is current. An initial
creation instead allocates a new root from `ABSENT`; the two branches cannot be
conflated.

CAP acceptance is not Finding closure. CAP acceptance keeps the Finding open
and moves it to `EVIDENCE_REQUIRED` or `PENDING_CLOSURE`; the synthetic Finding
does not expose a `CAP_ACCEPTED` projection state. Evidence `CLOSE` records
accepted Evidence verification and closes with `EVIDENCE_VERIFIED`. Other
Evidence outcomes remain open. `AUTHORIZED_CLOSE` is separately reasoned,
Manager-only, and allowed only from `PENDING_CLOSURE`, producing
`AUTHORIZED_CLOSURE` without impersonating Evidence verification.

Generation reset is forward-only. It requires exact Admin authority, reason,
generation revision and seal digest, and fully terminal lifecycle state. It
appends a reset tombstone and new active generation referencing the immutable
run and fixture; it never reloads or copies the 1,310 classifications and never
reactivates history.

## Digest graph

Canonical JSON uses the `AVIASURVEIL360_CANONICAL_JSON_V1` profile: UTF-8,
recursively UTF-8-bytewise sorted object keys, pre-normalized arrays, minimal
base-10 finite integers with no negative zero, JSON control/quote/backslash
escaping, no HTML escaping, no solidus escaping, literal UTF-8 for non-ASCII
scalars, no Unicode normalization, rejection of lone surrogates, and no
insignificant whitespace. Both validator-signal `identityDigest` and aggregate
exception `orderedIdentityDigests` use
`AGA-CLASSIFICATION-BASE-IDENTITY-V1` over the complete six-field identity.
Model descriptors are keyed and sorted by their normalized descriptor shape;
the supplied sealed-pass receipt digest set is retained as source provenance
and must be unique and role-complete, but it is not rebound to a former local
descriptor-digest pin. Every null model-metadata scalar requires its exact
field in `unavailableFields`, while every supplied scalar forbids that
unavailable marker. `displayedModelLabel` cannot substitute for `modelId`; an
exact model ID is accepted only when separately platform-reported.
The taxonomy self-digest uses
`AGA-QUESTION-CLASSIFICATION-TAXONOMY-V1` over the complete taxonomy object
excluding only `taxonomyDigest`. Pass proposal records and their enclosing
pass-batch output bind `inputDigest` to that batch's
`AGA-CLASSIFICATION-PASS-INPUT-V1` digest, while every sealed item and all run
states bind `inputDigest` to the one shared
`AGA-CLASSIFICATION-RUN-INPUT-V1` digest reconstructed from
`runInputPayloadFields`; pass seals retain exactly 25 ordered pass-input
digests. For a supplied ZIP, those immutable receipt input digests are the
source evidence retained after private-input schema, identity, fact, and
receipt-graph validation; the validator does not silently replace them with a
different locally derived digest. Pass proposal, semantic item, aggregate, and
run domains are
respectively `AGA-CLASSIFICATION-PASS-PROPOSAL-V1`,
`AGA-CLASSIFICATION-ITEM-V1`, `AGA-CLASSIFICATION-AGGREGATE-V1`, and
`AGA-CLASSIFICATION-RUN-V1`.

The pass digest excludes only itself. The semantic item digest excludes its own
digest and enclosing pass/run/aggregate back-references. The aggregate hashes
ordered semantic item digests plus its canonical payload. The run hashes both
pass seals and the aggregate while excluding only its own digest. Stored
back-references are populated after sealing and verified, never included in a
child digest.

## Privacy and authority

Accepted-package question bodies never enter classification or workspace
artifacts; they exist only in the immutable sealed overlay and transient
authorized memory. Workspace-authored bodies are append-only only for genuinely
new or reworded workspace question versions and are never copied from the
accepted package. Base identities and runtime Draft/lifecycle IDs never enter diagnostics, URLs, browser storage,
caches, service workers, offline outboxes, analytics, telemetry, retained media,
or test artifacts. Authorized bodies exist only transiently in memory and are
cleared on logout, principal change, denial, and BFCache restore.

The sibling workspace writes only its synthetic schema through least-privilege
reader/command roles. It cannot write canonical Provider, Identity, Membership,
Assignment, Publication, Planning, Audit, Finding, CAP, Evidence, notification,
outbox, delivery, release, or production data. Local verification remains
`candidate-only`; release is pending; production readiness is not established.
