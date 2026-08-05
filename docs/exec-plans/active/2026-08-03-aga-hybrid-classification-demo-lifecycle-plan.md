# AGA Hybrid Question Classification And Synthetic Demo Lifecycle ExecPlan

> **For agentic workers:** Execute this plan only after the user explicitly
> authorizes implementation. Do not commit, push, deploy, call an external
> system, or change a real database without separate exact authorization.

**Goal:** Classify every one of the 1,310 immutable AGA candidate question
identities with a controlled, versioned domain model; give CAA Admin and an
exactly scoped Department Manager a shared, editable candidate workspace; and
run a complete synthetic inspection lifecycle without changing the sealed AGA
overlay or creating any real governed record.

**Architecture:** Keep `preprod_aga_demo` as the immutable Admin-only source
projection. Add a sibling `preprod_aga_demo_workspace` companion that stores
classification provenance, immutable successor Drafts, synthetic scope facts,
deterministic recommendations, and `DEMO_*` lifecycle events. The original
1,310 question bodies remain in the immutable accepted package and sealed
overlay, never in the workspace/classification artifacts, and are composed
server-side after authorization. A genuinely new or reworded synthetic
candidate is the sole exception: its append-only workspace question version
stores a new body and digest with parent identity, actor, time, and reason; it
never copies or overwrites an original body and never enters a sealed AI run
without a successor run.

**Tech Stack:** Existing Go packages and PostgreSQL local-preprod patterns,
OpenAPI 3.1 with generated Go/TypeScript transport, React/Vite, Node contract
validators, Vitest, Playwright with isolated profiles, existing preprod OIDC
fixtures, and Codex AI analysis recorded through immutable prompt/model/run
digests. No runtime LLM or external research API is added.

## Global Constraints

- Preserve byte-for-byte the accepted
  `AGA_ALL_FORMS_SOURCE_RISK_DRAFT.json`, its 1,310 question identities, and the
  existing sealed `preprod_aga_demo` schema, loader, five GET-only routes, and
  Admin-only raw-package behavior.
- Do not add `primaryProvider`, `secondaryProvider`, `supportingProvider`, or
  equivalent ownership columns to question classifications. Questions belong
  to checklist/inspection profiles; provider involvement is an optional,
  reasoned relationship.
- Every question receives exactly one `mainDomainCode` and zero or more
  controlled `topicCodes`. No AI-generated free-text category may enter a
  sealed run.
- AI may propose candidate classification, applicability, evidence expectation,
  and external involvement. It cannot establish source authority, technical
  approval, publication, compliance, Finding severity, enforcement, release,
  or production readiness.
- High-confidence AI results are preselected only as
  `AUTO_PROPOSED_HIGH_CONFIDENCE`. They remain editable candidate input and are
  never called approved.
- A Department Manager edit creates an immutable successor Draft. Exclude and
  defer commands require a non-empty reason. A wording change creates an
  append-only workspace-owned question version with a new proposal identity
  and digest; the base classification run stays exactly 1,310 rows while a
  Manager Draft may contain more or fewer items.
- Technical approval and publication remain separate real governance
  decisions. Because every supplied question remains `SOURCE_MAPPING_REQUIRED`
  and `NOT_ATTESTED`, this plan may create only
  `READY_FOR_DEMO_SIMULATION`, never a real or implied approval/publication.
- Keep all supplied blocker states, including the 49 questions without a
  question-level source proposal, the 51 externally identified unresolved
  provider-applicability rows, and every expert-risk/source-currentness gap.
- Never infer applicability or involvement from `AGA`, form code, risk band,
  source proposal, organization type, lexical match, or model confidence alone.
- Store canonical target kind separately from target profile. Canonical target
  kinds remain `ORGANIZATION`, `PERSON`, `FACILITY`, `DEVICE`, `SYSTEM`,
  `ASSET`, and `LOCATION`; labels such as `AERODROME_RFFS_FUNCTION` are target
  profiles, not target kinds.
- The complete AGA-connected demo provider universe contains 14 catalog codes,
  but the current 1,310-question AGA profile is eligible for an inspected
  `AERODROME_OPERATOR` scope only. The other 13 codes may appear only in an
  explicit involvement relationship until a separately governed inspection
  profile establishes otherwise.
- Use synthetic, namespaced organization, provider-scope, target, authority,
  inspection, response, Potential Finding, Finding, CAP, Evidence, review, and
  closure identities. The workspace data plane must not write canonical
  provider, identity, assignment, source-attestation, decision, publication,
  Planning, Audit, Finding, CAP, Evidence, notification, outbox, delivery,
  release, or production tables.
- Admin may see the complete classification workspace. Department Manager
  access requires the current authenticated CAA `manager` role, an active exact
  membership, and an effective workspace-only
  `AGA_DEMO_CLASSIFICATION_MANAGER` binding for the same subject/membership and
  synthetic department/organizational-unit scope. It does not require or
  manufacture `Principal.DepartmentAssignments`, is not a canonical functional
  assignment, and grants neither `CHECKLIST_REVIEWER` nor technical approval/
  publication authority.
- Inspector, Lead Inspector, CAA reviewer, and Service Provider/Auditee access
  applies only to the lifecycle operations and projections explicitly listed
  below. Auditee and unrelated roles receive no classification, count, search,
  direct-ID, blocker, cross-organization, or existence signal.
- Candidate question text, accepted-package question identities/digests,
  runtime Draft filters, and runtime lifecycle identifiers must not enter URL/
  query state, browser history,
  referrers, Web Storage, IndexedDB, Cache Storage, service-worker caches,
  offline outbox, analytics, telemetry, logs, traces, metrics, retained
  screenshots, video, or test artifacts. Authorized response bodies and
  transient React memory are cleared on logout, principal change, denial, and
  BFCache restore. This new-workspace rule does not modify the preserved five-
  route Admin overlay prefix, whose existing form path and cursor query remain
  part of the accepted predecessor contract. Bounded invented IDs may appear
  only in unit-test source; no accepted or runtime-derived ID may be retained in
  test output or media.
- Preserve unrelated working-tree changes. The existing untracked
  `apps/api/internal/agaapplicability/applicability_test.go` is directly in
  scope, but its earlier provider-profile assumptions must be replaced without
  deleting its identity, provenance, exact-selection, and immutable-successor
  test intent.
- Work on the current branch. Do not create, switch, rename, or delete branches
  or worktrees. Do not stage or commit without current user authorization.
- Use English for code, schemas, statuses, tests, plan updates, and UI copy.
- Every implementation task requires focused regression coverage, a broader
  verification gate, and an update to this plan with literal evidence.
- Final claims remain `candidate-only`, `release pending`, and
  `production-ready: not established`.

---

## Status

- Plan status: `ready-for-verification`.
- Design: approved by the user on 2026-08-03.
- Independent plan review: first pass found 6 Critical, 24 Important, and 4
  Minor plan-quality findings. All were corrected in this artifact; the second
  adversarial pass and fresh docs-only evidence are recorded in Progress.
- ExecPlan artifact: `verified locally` only for the fresh commands and
  observations recorded in Progress.
- Implementation: Gate 0A and the original reopened 25-batch Gate 0B are
  independently accepted. The user-authorized Gate 0B platform-metadata
  availability amendment and the Task 2 supplied-receipt reconciliation are
  `verified locally` and the independent review boundary is recorded below.
  Task 1 remains independently accepted and `verified locally`; Task 2 is
  accepted and `verified locally`.
  Truthful platform-unavailable metadata is valid `candidate-only` demo
  provenance and is not a Task 2 blocker. The replacement candidate and
  challenge ZIPs independently seal at 25 batches and 1,310 records each;
  their immutable receipt prompt/model digests are retained as source evidence
  without rebinding to the repository's former fixed prompt or descriptor
  digest. Tasks 3–8 are `verified locally`. Task 9's offline inventory,
  authorization, recovery, and evidence gates are `verified locally`; the
  disposable predecessor setup and exact nine-account OIDC qualification
  subgate are also `verified locally`. Task 9 connected F1/F2/F3 evidence is
  `verified locally` on fresh task-owned disposable targets: the receipt-bound
  workspace qualification recorded zero forbidden/overlay delta, the two
  sibling-schema load/seal barriers passed, the exact 17-test browser set
  passed, credential revocation was receipt-backed, and the four-case connected
  fault matrix reached zero residue. The successor privacy-safe summary pins
  happy ledger `sha256:5b2f03652eaef75aa6cb33a2d22789f927bf1e3b2e62b5094c47d21a098c06ec`
  and fault ledger
  `sha256:0cec9fc66a074725297ddb95a9f61f6b1c152da061ab951464777f7d4311de3c`.
  Task 10's exact aggregate contract ladder is `verified locally` against the
  corrected successor evidence.
- AI classification run: repository runtime LLM `not run`; supplied candidate
  and challenge pass receipts reconciled locally.
- PostgreSQL workspace: `verified locally` on the task-owned disposable
  PostgreSQL namespace; the result remains candidate-only and is not a real
  database claim.
- Browser and connected role verification: `verified locally` for the isolated
  disposable run; no production or external-system claim is made.
- Product status: `candidate-only`.
- Release: `release pending`.
- Production-ready: not established.

## Objective And User-Visible Outcome

The delivered local-preprod demo must provide:

1. A versioned 18-domain taxonomy and a sealed base classification row for all
   1,310 immutable question identities, with exactly one main domain, controlled
   topic tags, inspection profile/type candidates, canonical target kind,
   target profile, operation/activity qualifiers, applicability, candidate
   Evidence expectation, controlled rationale codes, confidence evidence,
   blockers, sources,
   and AI run provenance.
2. A complete aggregate and exception inventory proving 1,310/1,310 coverage,
   zero missing or duplicate identities, exact digest reconciliation, domain
   and topic distributions, disagreement counts, source gaps, and all manual
   review queues.
3. A shared CAA Admin/Department Manager workspace. High-confidence rows are
   preselected; medium, low, conflicted, source-gap, and extraction-risk rows
   are prioritized for review. The Manager can retain, reclassify, add tags,
   remove tags, include, exclude, defer, or add a new candidate under the exact
   action-specific reason rules and immutable successor history.
4. A deterministic checklist recommendation using exact synthetic
   `organizationId`, provider scope root/ID/version, `providerTypeId`,
   `targetId`, `canonicalTargetKind`, `targetProfileCode`,
   `inspectionProfileCode`, `inspectionTypeCode`, department, organizational
   unit, operation/activity qualifiers, `effectiveAt`, and exact
   taxonomy/run/Draft/readiness pins.
5. A complete synthetic lifecycle:

   `DEMO_PROVIDER_SCOPE -> DEMO_INSPECTION -> DEMO_CHECKLIST_RESPONSE ->`
   `DEMO_POTENTIAL_FINDING -> DEMO_FINDING -> DEMO_CAP_REVISION ->`
   `DEMO_EVIDENCE_VERSION -> DEMO_VERIFICATION_DECISION -> DEMO_CLOSURE`

   CAP acceptance is not Finding closure. Closure requires accepted Evidence
   plus verification or an explicitly authorized synthetic closure path.
6. A visible separation between `READY_FOR_DEMO_SIMULATION`, technical
   approval, and publication. Unresolved source/authority facts keep the latter
   controls disabled with an exact reason.
7. An Admin-only generation reset that appends a reset tombstone and opens a
   new workspace generation without selectively deleting or mutating earlier
   Draft or lifecycle history.

## Scope

### Included

- Full-package, no-sampling classification of all 1,310 question boundaries.
- Controlled taxonomy, prompt contract, analysis-run provenance, deterministic
  validation, and machine-readable candidate deliverables without question
  text duplication.
- Optional external-involvement edges with explicit role, controlled condition,
  edge-specific sources, controlled rationale codes, confidence evidence, and
  blockers; an empty edge array is valid.
- Shared local-preprod persistence and exact synthetic role/functional-scope
  authorization.
- Department Manager review, batch actions, immutable successor Drafts, and
  conflict-safe idempotent commands.
- Deterministic candidate checklist recommendation and synthetic lifecycle.
- OpenAPI, tagged Go API, React UI, privacy, security, recovery, and connected
  local-preprod verification.

### Explicitly Excluded

- Editing or resealing the accepted AGA package or its immutable overlay.
- Runtime calls to OpenAI or any external model/research service.
- Real source download, source-currentness activation, source-owner
  attestation, technical approval, publication, provider provisioning,
  functional-assignment provisioning, Audit, Finding, CAP, Evidence, delivery,
  notification, release, deployment, or production records.
- Automatic Finding type, severity, safety-critical status, enforcement,
  certificate, compliance, or closure decisions.
- Treating the external researcher package, the System Design Matrix workbook,
  a lexical match, or AI agreement as regulatory authority.
- Changing the root legacy HTML/CSS/JavaScript demo oracle.

## Fixed Inputs And Verified Planning Facts

| Input | Exact planning identity and permitted use |
|---|---|
| Accepted question package | `deliverables/aga-all-forms-source-risk-draft-2026-08-01/AGA_ALL_FORMS_SOURCE_RISK_DRAFT.json`; 3,370,312 bytes; SHA-256 `5ebcce2d70ee22fef4165b490cb6e4b276ad776f40dbaf12e5cea85c9da91b15`; immutable question/source/risk candidate input |
| Sealed-overlay loader ZIP | `deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip`; 336,524 bytes; SHA-256 `30700a88aeb5b26514bf7eb76bef050deb08b96294db94117d185de5c9f163b2`; exact predecessor input required to recreate a disposable overlay for Task 9 |
| Service-provider catalog | `docs/regulatory-sources/catalogs/service-provider-catalog.v1.json`; 11,325 bytes; SHA-256 `42079b4046542e392c393fe6de1052d84f96938ea163cf63deed5ae9c4b6a789`; exact 20 provider codes |
| Independent research ZIP | `/Users/marlonjd/.codex/attachments/a2fa9639-5e9a-4e5d-a68d-0b38ef797b75/AGA_INDEPENDENT_RESEARCH_DELIVERABLES_2026-08-02.zip`; 76,750 bytes; SHA-256 `137592c739bc22f6be026f5bad94c5b200bb983132017d026b7e39634ab392c7`; adversarial candidate input only |
| Research question CSV | ZIP entry `question_level_review.csv`; 296,840 bytes; SHA-256 `e39685d467c9c66220b20e998deab366a138148f4d532db7fac07e58e64e7a7c`; 1,310 identities, not a final taxonomy |
| Provider classification CSV | ZIP entry `provider_classification_matrix.csv`; 12,808 bytes; SHA-256 `d52a98739db61828c16aa734154be18b11e6ebb358eeeb7f84c3d92a4a5430de`; 20 provider-level candidate classifications |
| Ambiguity CSV | ZIP entry `ambiguous_unmapped_inventory.csv`; 7,489 bytes; SHA-256 `6e97a193f5e12dbe81f87d44d4b22c36ce446a40be7ef0f9fc939e8fbf1e654d`; 51 explicit unresolved identities |
| System Design Matrix | `/tmp/codex-remote-attachments/019fcd4b-4cdb-7672-bf84-c703b0a24a58/39DD3E5A-E6A8-483B-AF11-706021BCEE53/1-AVIASURVEIL360_System_Design_Matrix.xlsx`; 12,228 bytes; SHA-256 `e4d054f741b11ca9d848842a891d6f811f2e644aba29a7ffda970bfe6abb931e`; form/module/screen/workflow reference only |
| Lifecycle behavior authorities | `docs/product-specs/workflows/AUDIT_CHECKLIST_WORKFLOW.md` SHA-256 `7dee737c7c5e47e996857e956514a8d46d1a4444234b021cac77cd6cff6b30a2`; `docs/product-specs/workflows/FINDING_CAP_EVIDENCE_WORKFLOW.md` SHA-256 `896f9fa7d498fdc20c582134a15ed6acdc11b78926e655854c43e49fbb24815c`; `docs/product-specs/data-and-rules/PRODUCTION_CONTRACT_VOCABULARY.md` SHA-256 `3ef3349d738feb9789aaab6e92246f55948053604a8304706fc1bbd0cd786769`; semantic inputs to freeze at Gate 0 |

The accepted package contains exactly 52 forms, 31 question-bearing forms,
1,310 question boundaries, 1,310 unique proposal IDs, and 1,310 unique
form/proposal/ordinal/digest identities. `candidateOnly` is globally `true`;
`NON_AUTHORITATIVE_CANDIDATE` is not a supplied per-row state and must not be
invented. All 1,310 rows are literally `SOURCE_MAPPING_REQUIRED`,
`NOT_ATTESTED`, `CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW`, and
`NOT_SUPPLIED`. Extraction is 1,282 `EXTRACTED_CANDIDATE` plus 28
`EXACT_SOURCE_BACKED`. Forty-nine questions have neither question-level source
proposals nor source references.

Identity is the full tuple, never text digest alone. There are only 1,278 unique
text digests: 30 duplicate-digest groups account for 32 repeated occurrences,
with a maximum group size of three. Tests must accept identical wording while
rejecting duplicate full identities.

The external research package proposes `AERODROME_OPERATOR` as candidate
subject context for all 1,310 rows and contains 30 unique candidate
question/provider pairs: `CNS_PROVIDER` 16, `AIS_AIM_PROVIDER` 7, `ANSP` 6,
and `AIR_OPERATOR` 1. Every pair is duplicated across the research CSV's
operational-interface and Evidence-contributor fields, so it is not yet a
role-specific edge and must not be mechanically imported twice. Those 30 rows
are not classification coverage. The full taxonomy must classify 1,310/1,310
rows.

The earlier 2/4/5/3 lexical-omission counts had no pinned matching rules or
identity inventory and are not verified planning facts. Gate 0 replaces them
with a text-free, hashed omission-review inventory whose every entry records
the full immutable identity and a controlled `signalRuleId`. A lexical signal
never creates an involvement edge by itself.

The 51 research-unresolved identities are not additive to the 49 source gaps:
all 49 source-gap identities are inside the 51, leaving exactly two additional
external-applicability unresolved identities.

The workbook contains two worksheets. Its main `A1:K28` range has 27 grouped
data rows covering all 52 forms and no question-level or provider-mapping
columns. It is useful only for form-level module, screen, trigger, actor,
workflow, and output hints. The 14 connected/6 no-default provider partition is
a research-derived candidate decision frozen by this plan, not a field in the
provider catalog.

## Controlled Taxonomy V1

Every classification has exactly one of these main domain codes:

1. `GOVERNANCE_ORGANIZATION_PERSONNEL`
2. `CERTIFICATION_LICENSING_CHANGE`
3. `AERODROME_MANUAL_DOCUMENT_CONTROL`
4. `QUALITY_MANAGEMENT`
5. `SAFETY_MANAGEMENT_RISK_ASSESSMENT`
6. `AERODROME_DATA_INFORMATION_PUBLICATION`
7. `PHYSICAL_CHARACTERISTICS_MOVEMENT_AREA`
8. `OBSTACLES_OLS_WORKS`
9. `VISUAL_AIDS_MARKINGS_SIGNS_LIGHTING`
10. `ELECTRICAL_SYSTEMS_POWER`
11. `APRON_GROUND_OPERATIONS`
12. `RESCUE_FIRE_FIGHTING_FIRE_SAFETY`
13. `EMERGENCY_PLANNING`
14. `MAINTENANCE_OPERATIONAL_INSPECTION`
15. `RUNWAY_SAFETY_FRICTION_SURFACE_CONDITIONS`
16. `WILDLIFE_HAZARD_MANAGEMENT`
17. `ENVIRONMENTAL_MANAGEMENT`
18. `NIGHT_OPERATIONS_FACILITIES`

Gate 0 performs a read-only vocabulary-discovery pass across all 1,310 rows,
then freezes the complete `topicCodes`, inspection-profile codes, inspection
types, target profiles, qualifier keys, Evidence expectation codes, and
applicability dispositions in taxonomy version 1 before the classification
pass starts. Once frozen, a new code requires a successor taxonomy version;
the classifier cannot extend version 1 while classifying.

Allowed external-involvement roles are:

- `TECHNICAL_INTERFACE`
- `COORDINATION`
- `DATA_ORIGINATION`
- `DATA_PUBLICATION`
- `EVIDENCE_CONTRIBUTION`
- `OPERATIONAL_PARTICIPATION`

The 14 AGA-connected demo provider codes are:

- `AERODROME_OPERATOR`
- `ANSP`
- `CNS_PROVIDER`
- `AIS_AIM_PROVIDER`
- `MET_PROVIDER`
- `SAR_ORGANIZATION`
- `AVSEC_PROVIDER`
- `AIR_OPERATOR`
- `AMO`
- `ATO`
- `GROUND_HANDLING`
- `FUEL_PROVIDER`
- `CARGO_REGULATED_AGENT`
- `RPAS_UAS_OPERATOR`

The remaining catalog codes `CAMO`, `FSTD`, `DOA`, `POA`, `AEMC`, and `AME`
retain `NO_DEFAULT_AGA_RELATIONSHIP` and never enter the demo-connected set
without a successor taxonomy/provider-classification decision.

## Classification And Confidence Contract

For each immutable base question identity, the sealed item contains only these
lower-camel JSON fields: `packageVersion`, `packageJsonSha256`, `formCode`,
`proposalId`, `ordinal`, `textDigest`, `taxonomyVersion`, `mainDomainCode`,
`topicCodes`, `inspectionProfileCodes`, `inspectionTypeCodes`,
`canonicalTargetKind`, `targetProfileCode`, `operationQualifiers`,
`activityQualifiers`, `applicabilityDisposition`,
`evidenceExpectationCodes`, `externalInvolvements`, `agreementConfidence`,
`recommendationState`, `rationaleCodes`, `confidenceEvidence`, `sourceRefs`,
`sourceMappingState`, `sourceAuthorityState`, `riskClassificationState`,
`decisionState`, `extractionState`, `questionSourceProposalGap`,
`externalApplicabilityUnresolved`, `passDisagreementCodes`,
`passOneResultDigest`, `passTwoResultDigest`, `passOneRunId`, `passTwoRunId`,
`promptDigest`, `modelDescriptorDigests`, `taxonomyDigest`, `inputDigest`,
`itemSemanticDigest`, `classificationRunDigest`, and `aggregateDigest`.
`externalInvolvements` is a set-like array of zero or more closed edge objects.
Each edge contains exactly `providerTypeCode`, `involvementRoleCode`,
`conditionCode`, `applicabilityDisposition`, edge-specific `rationaleCodes`,
`confidenceEvidence`, `sourceRefs`, and `blockerCodes`. Its nested
`confidenceEvidence` uses the same closed evidence object defined below and
binds to the canonical digest of that edge's controlled semantic tuple. Empty
is valid; no edge implies ownership, checklist assignment, or inspected scope.
The two duplicated research role columns are candidate input facts for one
semantic analysis, never instructions to emit two edges. Neither an edge nor
any other field may contain Primary, Secondary, Supporting, owner, ownership,
or hierarchy semantics.

`rationaleCodes` are controlled taxonomy values, never free narrative.
Top-level `confidenceEvidence` is a bounded array of closed objects containing
exactly `proposalField`, `proposalValueDigest`, `rationaleCode`,
`inputFactSelector`, `inputFactValueDigest`, and optional `signalRuleId`.
`proposalField` is a frozen proposal-bearing field name, and
`proposalValueDigest` binds the evidence to the exact normalized scalar,
set-member, qualifier pair, or external-involvement semantic tuple emitted by
that pass. One evidence tuple cannot be reused to satisfy a different field or
value. Gate 0 freezes the permitted selectors, proposal shapes, and allowed
field/value/rationale/selector/rule combinations. Selectors are limited to
exact supplied facts such as question-body digest, form-metadata digest,
source-proposal digest, research-row digest, or a validator-recomputed
signal-rule match; no body fragment or model chain-of-thought is emitted. The
UI renders explanations from controlled codes. Edge-specific evidence remains
nested on its edge; top-level evidence covers every non-edge proposal.

Question text and future Manager decisions are not sealed classification-item
fields. Manager decisions, Draft successors, and workspace question versions
are separate append-only records referencing the exact closed `questionRef`
union below. A body digest, proposal ID, root sequence, or other single field
is never a question identity.

The sealed run also owns exactly 2,620 immutable text-free per-pass proposal
records: one `CANDIDATE` and one `CHALLENGE` record for each full base identity.
Each closed record contains the six-field base identity,
`classificationRunId`, `passRole`, `passRunId`, `promptDigest`,
`modelDescriptorDigest`, `inputDigest`, one closed `proposalProjection` with
the complete normalized editable proposal fields (including edge-specific
involvement provenance/evidence), pass-level `rationaleCodes`, top-level
`confidenceEvidence`, `sourceRefs`, and `passResultDigest`. The final item does
not rely on a digest as hidden data: its `passOneRunId`/`passOneResultDigest`
and `passTwoRunId`/`passTwoResultDigest` must resolve exactly those two records
under the same full identity. Both records and the final item seal together;
none can be replaced or appended afterward. `ACCEPT_CANDIDATE_PASS` and
`ACCEPT_CHALLENGE_PASS` server-copy the selected immutable
`proposalProjection`, never reconstruct it from disagreement codes or accept a
client projection.
The final reconciled item's editable proposal values, `rationaleCodes`,
`confidenceEvidence`, `sourceRefs`, and edge provenance are exact normalized
copies of the `CANDIDATE` record; `passDisagreementCodes` is validator-derived.
Challenge-only values/evidence remain in the resolvable `CHALLENGE` record.
Confidence is recomputed from both records, so this candidate precedence does
not discard or misattribute challenge evidence.

### Exact field and status registry

| Concept | Exact contract |
|---|---|
| Base identity | The six-field tuple `packageVersion`, `packageJsonSha256`, `formCode`, `proposalId`, `ordinal`, `textDigest`; digest alone is not identity |
| Question reference | Closed union discriminated by `questionOrigin`. `SEALED_BASE` contains exactly the discriminator plus the six-field Base identity. `WORKSPACE` contains exactly the discriminator, `generationId`, server-issued `questionRootId`, server-issued `questionVersionId`, server-issued `proposalId`, `rootSequence`, `bodyDigest`, `parentQuestionKey`, `createdBySubjectId`, canonical UTC `createdAt`, and controlled `reasonCode`. Gate 0 freezes ID and timestamp lexical forms. No client may issue an ID or substitute a digest/sequence for this reference |
| Parent question key | Closed nullable union. Null is valid only for the first `ADD_CANDIDATE` version. `SEALED_BASE` contains exactly its discriminator plus the six-field Base identity and is valid only on the first Workspace reword of that Base leaf. `WORKSPACE` contains exactly its discriminator plus `generationId`, `questionRootId`, `questionVersionId`, `proposalId`, `rootSequence`, and `bodyDigest`; it must resolve an earlier version with the child's same generation/root/sequence. Parent chains are acyclic and provenance is resolved from the immutable parent rather than duplicated in the key |
| Workspace identity transition | `ADD_CANDIDATE` allocates a fresh root, version, proposal, and generation-unique `rootSequence`, with null parent. A first `REWORD_CANDIDATE` of a Base leaf allocates those Workspace fields and names the Base key as parent; its logical order remains the Base package position. Rewording a Workspace leaf preserves generation, root, and sequence, allocates fresh version and proposal IDs, and names the exact current Workspace key as parent. A root ID cannot name two root chains, and version/proposal IDs cannot repeat within a generation; `rootSequence` is order-only, never identity |
| Target | `canonicalTargetKind` is one of the seven canonical kinds; `targetProfileCode` is a taxonomy code and is never accepted as a kind |
| Provider | `providerTypeId` identifies a synthetic scope record; the server derives its catalog `providerTypeCode`; involvement rows store only the code |
| Sealed agreement confidence | Base `agreementConfidence` is `HIGH`, `MEDIUM`, or `LOW`; there is no base value named `BLOCKED` or null |
| Recommendation state | `AUTO_PROPOSED_HIGH_CONFIDENCE`, `MANAGER_REVIEW_REQUIRED`, `BLOCKED_SOURCE_GAP` |
| Classification run | `LOADING`, `SEALED`, `REJECTED`; only `SEALED` is readable |
| Pass proposal role | `CANDIDATE` or `CHALLENGE`; exactly one of each per full identity/run |
| Draft state | `WORKING`, `READY_FOR_DEMO_SIMULATION`, `SUPERSEDED` |
| Draft item origin | `SEALED_BASE`, `MANAGER_AUTHORED`, `MANAGER_REWORDED` |
| Workspace root order | `rootSequence` is append-only and unique within a generation; rewording preserves it, but no lookup, CAS, replay, or history projection may use it without the full Workspace question reference |
| Draft item confidence field | `draftAgreementConfidence` copies sealed `agreementConfidence` only for an unedited base item; every semantic/authored/reworded successor is null |
| Draft item recommendation field | `draftRecommendationState`: `AUTO_PROPOSED_HIGH_CONFIDENCE`, `MANAGER_REVIEW_REQUIRED`, or `BLOCKED_SOURCE_GAP` |
| Draft item review field | `draftReviewState`: `AUTO_PRESELECTED`, `PENDING_MANAGER_REVIEW`, or `MANAGER_DISPOSED` |
| Draft item disposition field | `draftDisposition`: null only for `PENDING_MANAGER_REVIEW`; otherwise exactly `INCLUDE`, `EXCLUDE`, or `DEFER` |
| Manager action | `RETAIN`, `RECLASSIFY_MAIN_DOMAIN`, `ADD_TOPIC`, `REMOVE_TOPIC`, `RESOLVE_CLASSIFICATION_PROPOSALS`, `INCLUDE`, `EXCLUDE`, `DEFER`, `ADD_CANDIDATE`, `REWORD_CANDIDATE`, `MARK_READY_FOR_DEMO_SIMULATION` |
| Workspace generation | `ACTIVE`, `RESET`; a reset generation cannot be reactivated |
| Generic command envelope | Every command POST body contains exact `operationId`, `idempotencyKey`, and current `expectedGenerationId` in addition to its discriminator/payload |
| Reset CAS body | Exact `expectedGenerationId`, `expectedGenerationRevision`, and `expectedGenerationSealDigest`; all three must match the current active generation |
| Lifecycle CAS body | Exact `expectedLifecycleRevision` and `expectedLifecycleDigest`; both must match the command's current aggregate projection |

Gate 0 freezes every remaining controlled array/code set: topic, inspection profile,
inspection type, target profile, qualifier-key/value, Evidence expectation,
applicability disposition, involvement condition, blocker, source-reference
kind, disagreement, signal-rule, lifecycle event, and reason codes. Every governing JSON object is closed with
`additionalProperties: false`; every controlled array has enum-constrained
items, `uniqueItems: true`, and an explicit maximum where the contract bounds
cardinality. There is no open or untyped code array, and the classification
passes cannot add a code.

Gate 0 also classifies every array as ordered or set-like. Package forms,
question identities, bounded batches, aggregate item digests, Draft/lifecycle
revisions, and event streams are genuinely ordered and preserve their defined
order. Every classification code/member collection, qualifier collection,
`rationaleCodes`, `confidenceEvidence`, `sourceRefs`, `blockerCodes`, and
`externalInvolvements` is set-like: before hashing it is normalized by the
frozen UTF-8 bytewise
tuple for its scalar or object fields; nested evidence sorts by
`proposalField`, `proposalValueDigest`, `rationaleCode`,
`inputFactSelector`, `inputFactValueDigest`, then `signalRuleId`. Validation
rejects any duplicate semantic key before normalization; it never silently
drops one. Semantically equivalent unique input order
therefore produces the same normalized payload and digest. Gate 0 tests
reorder-equivalence, duplicate rejection, and the distinct ordered-array
cases in both Go and Node.

### Non-circular digest graph

Gate 0 freezes one cross-language canonical JSON algorithm: UTF-8, recursively
lexicographically sorted object keys, the already normalized array order above,
minimal base-10 integers, and no insignificant whitespace. Each per-pass
proposal digest hashes `AGA-CLASSIFICATION-PASS-PROPOSAL-V1` plus its canonical
record excluding only `passResultDigest`. A pass item semantic digest hashes
the domain separator `AGA-CLASSIFICATION-ITEM-V1` plus its canonical payload,
excluding `itemSemanticDigest`, pass-result/run/aggregate digests, and all
enclosing back-references. The aggregate digest hashes
`AGA-CLASSIFICATION-AGGREGATE-V1` plus the ordered semantic-item digests and
canonical aggregate payload. The run digest hashes
`AGA-CLASSIFICATION-RUN-V1` plus the immutable run receipt, both pass seals,
and aggregate digest, excluding only its own digest. Stored item back-references
to pass, aggregate, and run digests are populated after sealing and verified
against the enclosing records but never enter the child digest. Go and Node
tests reconstruct each layer independently and reject any alternate ordering,
domain separator, omitted field, or circular inclusion.

### Fatal run validation versus row review

Identity/digest mismatch, duplicate or missing full identity, incomplete pass,
missing/duplicate/mismatched candidate or challenge proposal record, unknown
rationale/selector/rule, input-fact digest or recomputed-rule mismatch,
malformed provenance, schema error, forbidden field, aggregate mismatch, or
non-text-free output is fatal to the whole run. The loader records `REJECTED`
and writes no readable classification rows or seal. These defects never become
a `BLOCKED` row. A structurally valid pass that simply omits permitted evidence
for a semantic proposal remains valid but deterministically lowers that row's
confidence; the validator never judges free prose.

Every Gate 0/Task 2 CLI, test, and validator diagnostic is privacy-redacted.
Stdout/stderr may contain only a controlled error code, pass/batch ordinal,
bounded count, and aggregate/receipt digest; it may not contain a form code,
proposal ID, ordinal, text digest, full identity, source reference, question
body/fragment, private path, or raw response. Text-free detailed exception
inventories belong only in the authorized deliverable. Tests derive a real
identity mismatch in memory, capture both streams, and assert both the allowed
shape and absence of every sensitive input token/body.

Pass one is the candidate pass and pass two is a blind challenge pass. If both
valid passes disagree, the sealed candidate retains pass one's controlled value
and records pass two's challenge plus a disagreement code for Manager review.
This deterministic precedence always yields one candidate `mainDomainCode`
without pretending the disagreement is resolved.

`agreementConfidence` is rule-derived and independent of immutable governance
states. The exact core proposal set is `mainDomainCode`,
`canonicalTargetKind`, `targetProfileCode`, `applicabilityDisposition`, and
every normalized member of `inspectionProfileCodes`. The exact auxiliary set
is every normalized member of `topicCodes`, `inspectionTypeCodes`,
`operationQualifiers`, `activityQualifiers`, `evidenceExpectationCodes`, and
`externalInvolvements`. No undefined "actor responsibility" proposal exists.
For a scalar or set member to be evidenced, its field and exact normalized
value digest must have a validator-confirmed permitted evidence object in that
pass; an optional empty set emits no proposal.

Confidence precedence is total and evaluated `LOW`, then `MEDIUM`, then
`HIGH`:

- `LOW`: the passes disagree on any core proposal, or either pass lacks valid
  field/value-bound evidence for any emitted core proposal.
- `MEDIUM`: the core proposal sets agree and are fully evidenced, but the
  passes disagree on any auxiliary proposal or either pass lacks valid
  field/value-bound evidence for an emitted auxiliary proposal.
- `HIGH`: both valid passes agree on the complete normalized core and auxiliary
  proposal sets and every emitted proposal in both passes has its own valid
  field/value-bound evidence.

All 1,310 rows still retain `SOURCE_MAPPING_REQUIRED`, `NOT_ATTESTED`, and
`CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW`. Those governance facts do
not make semantic agreement impossible and never get cleared by confidence.
`EXTRACTED_CANDIDATE` is also preserved rather than silently converted into a
fatal extraction error.

Recommendation precedence is total and evaluated in this order:

1. `questionSourceProposalGap=true` yields `BLOCKED_SOURCE_GAP`, even if the
   passes semantically agree.
2. Otherwise, an external-applicability unresolved flag, any pass disagreement,
   or `MEDIUM`/`LOW` yields `MANAGER_REVIEW_REQUIRED`.
3. Otherwise, `HIGH` yields `AUTO_PROPOSED_HIGH_CONFIDENCE`.

`AUTO_PROPOSED_HIGH_CONFIDENCE` means included in the working Draft by default;
it never means technical approval or publication. A Manager-authored workspace
question has `questionSourceProposalGap=true` and therefore starts with
`draftAgreementConfidence=null`, effective `BLOCKED_SOURCE_GAP`,
`SOURCE_MAPPING_REQUIRED`, `NOT_ATTESTED`, and
`CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW`. A reworded body also starts
with its new version's `questionSourceProposalGap=true` and effective
`BLOCKED_SOURCE_GAP`; any parent source proposal/reference remains immutable
parent history and is not mapping for changed wording. It cannot inherit
`HIGH`. This plan contains no source-proposal-resolution action that could clear
the new-version flag.

## Draft, Batch, And Simulation-Readiness Contract

Every successful classification Manager action or batch execution uses the
generic command envelope and carries
`operationId`, `idempotencyKey`, `expectedGenerationId`,
`expectedDraftRevision`, and `expectedDraftContentDigest` and creates one
immutable successor Draft. Recommendation and lifecycle commands do not mutate
the Draft; they pin its readiness/digest and use their own exact entity
revision/digest CAS while appending only their snapshot or event.
`RECLASSIFY_MAIN_DOMAIN`, `ADD_TOPIC`, `REMOVE_TOPIC`,
`RESOLVE_CLASSIFICATION_PROPOSALS`, `EXCLUDE`, `DEFER`, `ADD_CANDIDATE`,
`REWORD_CANDIDATE`, and
`MARK_READY_FOR_DEMO_SIMULATION` require a reason. `INCLUDE` also requires a
reason when the row is `BLOCKED_SOURCE_GAP`; `RETAIN` does not.

`ADD_CANDIDATE` accepts no client root/version/proposal ID. In one transaction
the server allocates all three IDs and the next generation-unique
`rootSequence`, appends the initial Workspace question reference with null
parent, and appends the Draft successor. `REWORD_CANDIDATE` supplies the full
current `questionRef` and Draft CAS, never a bare logical root, version ID, or
body digest. The server requires that reference to be the sole current leaf,
then handles its discriminator exactly. For a Base leaf it allocates the first
Workspace root/version/proposal and
sequence while retaining Base package order through the parent key; for a
Workspace leaf it preserves generation/root/sequence and issues fresh
version/proposal IDs. Both forms store the exact current key as parent. The
successor body digest must differ from its parent. In one transaction the
command appends the new body/version
and a Draft successor that replaces the parent reference with the new leaf;
the same Draft can never include both versions. An original overlay question
may be the referenced base parent, but its body is never copied into workspace
storage and a byte-identical “reword” is rejected. Repeated body digests on
different roots remain distinct references; an exact duplicate reference, a
reused server-issued root/version/proposal ID within a generation, a
non-current parent, or a missing, cyclic, cross-generation, or cross-root
parent chain fails with no write.

Every unedited initial Draft item copies sealed `recommendationState` exactly
into `draftRecommendationState` and sealed `agreementConfidence` into
`draftAgreementConfidence`. Initial sealed-base rows with
`AUTO_PROPOSED_HIGH_CONFIDENCE` are
`draftReviewState=AUTO_PRESELECTED`/`draftDisposition=INCLUDE`. Every `MANAGER_REVIEW_REQUIRED` or
`BLOCKED_SOURCE_GAP` row and every authored/reworded row starts
`draftReviewState=PENDING_MANAGER_REVIEW` with
`draftDisposition=null`. `RETAIN` confirms the current controlled
classification but does not itself dispose a pending item or resolve a pass
disagreement;
`INCLUDE`, `EXCLUDE`, or `DEFER` moves `draftReviewState` to
`MANAGER_DISPOSED` under the reason rules. No Draft containing
`draftReviewState=PENDING_MANAGER_REVIEW` can become ready.

Any classification-semantic Manager edit—`RECLASSIFY_MAIN_DOMAIN`, `ADD_TOPIC`,
`REMOVE_TOPIC`, or `RESOLVE_CLASSIFICATION_PROPOSALS`—creates a Draft-only successor item with
effective `draftRecommendationState=BLOCKED_SOURCE_GAP` when its immutable
`questionSourceProposalGap` remains true and otherwise
`draftRecommendationState=MANAGER_REVIEW_REQUIRED`,
`draftAgreementConfidence=null`, `draftReviewState=PENDING_MANAGER_REVIEW`, and
`draftDisposition=null`. It preserves the
referenced sealed base's original `agreementConfidence`,
`recommendationState`, pass values, evidence, and run provenance as immutable
history, but those sealed values cannot auto-dispose the edited successor.
`ADD_CANDIDATE` and `REWORD_CANDIDATE` start with a new version-level source
gap and effective `BLOCKED_SOURCE_GAP` under the same Draft-only review/
disposition state. The exact Manager must
subsequently issue `INCLUDE`, `EXCLUDE`, or `DEFER`; only `INCLUDE` makes the
current leaf selectable. Tests cover edits to an originally
`AUTO_PROPOSED_HIGH_CONFIDENCE` row and prove it cannot reach readiness through
its former auto-preselection.

`RESOLVE_CLASSIFICATION_PROPOSALS` has exactly one `resolutionMode`:
`ACCEPT_CANDIDATE_PASS`, `ACCEPT_CHALLENGE_PASS`, or `SET_EXACT`. The first two
copy the named sealed pass's complete normalized editable projection on the
server; `SET_EXACT` supplies one closed complete projection containing main
domain, topics, inspection profiles/types, target kind/profile,
operation/activity qualifiers, applicability, Evidence expectations, and
zero-or-more fully provenanced external-involvement edges. All modes require a
controlled reason, validate taxonomy/compatibility, append a Draft-only manual
provenance record, and never rewrite either pass. Partial convenience domain/
topic actions follow the same demotion rule. After any resolution, a separate
`INCLUDE`, `EXCLUDE`, or `DEFER` is still required. Tests and UI cover each
field family, both sealed-pass modes, exact custom resolution, invalid mixed or
partial payloads, and subsequent disposition.

A batch operation is limited to `RETAIN`, `RECLASSIFY_MAIN_DOMAIN`,
`ADD_TOPIC`, or `REMOVE_TOPIC` and at most 500 items. The server first returns a
preview containing the exact generation, run, Draft revision/digest,
canonicalized filter digest, sorted affected-item identity digest, count, and
expiry. Execution supplies that preview ID/digest; the server recomputes the
set in the same transaction and either writes one whole successor or writes
nothing. Pagination state or a client count is never authority.

`RESET_GENERATION` never invokes the removed loader and never copies 1,310
classification rows. In one transaction it resets the old generation, appends
its tombstone, creates the new `ACTIVE` generation referencing the already
immutable sealed taxonomy/classification run and sealed fixture profile, and
creates a deterministic revision-1 `WORKING` Draft over those base identities.
Each unedited base row again copies sealed `recommendationState` into
`draftRecommendationState` and sealed `agreementConfidence` into
`draftAgreementConfidence`. Only `AUTO_PROPOSED_HIGH_CONFIDENCE` rows start
`draftReviewState=AUTO_PRESELECTED`/`draftDisposition=INCLUDE`; every other
base row starts `draftReviewState=PENDING_MANAGER_REVIEW` with
`draftDisposition=null` and requires a later explicit decision. Workspace-added/
reworded questions and all prior mutable lifecycle state remain immutable in
the old reset generation and are not copied into the new revision-1 Draft. A
failed transaction exposes neither tombstone nor new generation;
a successful reset cannot be rolled back or reactivate the old generation.
Reset is allowed only when every inspection in the current generation is
`COMPLETED`, every Potential Finding root is terminal, every converted Finding
is `CLOSED`, and no CAP/Evidence review or resubmission is pending. Any
nonterminal inspection, Finding, CAP, or Evidence branch makes the whole reset
conflict with no write; tests cover each named nonterminal lifecycle state.
Ordinary routes return neutral unavailable for old-generation IDs, while Admin
history may read their immutable projections by their complete historical
question references and reconstruct every parent/current-leaf snapshot without
digest- or sequence-based aliasing. The new generation is immediately usable
for Manager review and can reach readiness only through a new event.

An Admin cannot manufacture Manager readiness. The exact scoped Manager may
record `MARK_READY_FOR_DEMO_SIMULATION` only when:

- the workspace generation is `ACTIVE` and references exactly one 1,310-row
  classification run whose state is `SEALED` and whose package/taxonomy/input
  digests match;
- the expected Draft revision/content digest is current, every included item
  has a controlled domain/profile/target/applicability shape, and every item
  whose effective `draftRecommendationState` is not the unchanged base
  `AUTO_PROPOSED_HIGH_CONFIDENCE` default has an explicit Manager disposition;
  no item remains `draftReviewState=PENDING_MANAGER_REVIEW`;
- a source-gap inclusion carries a simulation-only reason and retains every
  source/attestation/expert-review blocker;
- each new/reworded workspace item has its own body digest, provenance,
  controlled manual classification, and explicit Manager inclusion; and
- no fatal identity, schema, taxonomy, provenance, seal, generation, authority,
  or unresolved Draft conflict remains.

The readiness event pins generation, run, taxonomy, Draft, provider-scope
profile, actor/scope, reason, time, and their digests. A later Draft successor
does not mutate it and requires a new readiness event. Source mapping,
attestation, and expert review may remain unresolved for simulation; the UI
must show them. No technical-approval or publication command/route exists.

## Deterministic Demo Recommendation Contract

The request must supply all of:

- `operationId`
- `idempotencyKey`
- current `expectedGenerationId`
- `organizationId`
- `providerScopeRootId`
- `providerScopeId`
- `providerScopeVersion`
- `providerTypeId`
- `targetId`
- `canonicalTargetKind`
- `targetProfileCode`
- `inspectionProfileCode`
- `inspectionTypeCode`
- `departmentId`
- `organizationalUnitId`
- exact `operationQualifiers`
- exact `activityQualifiers`
- `effectiveAt`
- `taxonomyVersion` and `taxonomyDigest`
- `classificationRunId` and `classificationRunDigest`
- exact immutable `draftId`, `draftRevision`, and `draftContentDigest`
- `expectedDraftRevision`, exactly equal to `draftRevision`
- `readinessEventId` and `readinessEventDigest`

For `CREATE_RECOMMENDATION`, body `draftRevision` is the immutable selection
pin and body `expectedDraftRevision` is the command CAS mirrored by
`If-Match`; they must be equal, and `draftContentDigest` must match that exact
revision. For `CREATE_INSPECTION`, body `expectedRecommendationRevision` is
both the current recommendation CAS mirrored by `If-Match` and the revision
whose ID/digest is supplied; any inequality fails before store use.

The server resolves one exact active synthetic scope version, one typed target,
the scope's catalog `providerTypeCode`, one target kind/profile-compatible
inspection profile, one taxonomy version, one sealed run, one immutable Draft,
and one readiness event. Required qualifier keys and allowed values come from
the frozen inspection profile; the server compares exact maps and never accepts
a client `complete` flag. From the pinned Draft it resolves the immutable
reference/parent graph, rejects zero or multiple leaves for any root, selects
only the one current leaf for each exact question root, and selects only leaves
whose exact disposition is
`INCLUDE`; `EXCLUDE`, `DEFER`, null disposition, superseded wording, and the
replaced base version are ineligible. It then applies the edited leaf's
controlled profile/applicability facts and the explicit profile rules in stable
order: Base roots and Workspace roots whose ancestry terminates in Base retain
accepted package form/question order; null-parent Workspace-added roots follow
by their append-only `rootSequence`; and a reword retains its root position
while replacing only the leaf. The snapshot stores
the full discriminated reference for every selected leaf; it never stores only
a digest, sequence, or client reconstruction. Missing, extra, inactive,
expired, ambiguous, stale, mismatched, or under-qualified input returns the
same neutral no-store result and creates no recommendation or lifecycle record.
A successful recommendation is an immutable synthetic snapshot pinned to all
resolved IDs, versions, times, and digests. The word `AGA`, form code, risk
band, source proposal, organization type, target profile, provider code, and AI
confidence are never sufficient selectors.

`CREATE_INSPECTION` accepts the exact `recommendationId` and
`recommendationDigest`, exact Manager-selected `inspectorBindingId`/
`inspectorBindingRevision` and `leadBindingId`/`leadBindingRevision`, plus
the generic `operationId`/`idempotencyKey`/current `expectedGenerationId`
envelope and exact `expectedRecommendationRevision`. The server must resolve that immutable
recommendation, require that its generation, run,
Draft, readiness, scope, target, qualifiers, and effective interval are still
eligible for a new inspection, require exactly one current Inspector and one
current Lead workspace binding for the same synthetic scope, and pin their
binding IDs/revisions/subjects/memberships while server-copying the ordered
selected full question references into an immutable inspection snapshot. This
creates no canonical Assignment. A missing, ambiguous, revoked, stale, cross-
scope, or subject/membership-mismatched binding, or a stale,
superseded, wrong-scope, client-reconstructed, or digest-mismatched
recommendation creates nothing. An already-created inspection retains its
historical recommendation and question pins after a later Draft. A generation
cannot reset until that inspection's complete lifecycle is terminal; after
reset it is immutable Admin history and ordinary actors receive neutral denial.
Only new inspections require the current eligible recommendation.

## Repository Orientation And Planned Interfaces

- Immutable overlay source and reader:
  `apps/api/internal/preproddata/agacandidatedemo/`,
  `apps/api/internal/agacandidatedemo/`, and
  `apps/api/internal/httpapi/aga_candidate_demo_api.go`.
- Existing incomplete candidate-domain tests:
  `apps/api/internal/agaapplicability/applicability_test.go`.
- New pure classification/recommendation domain:
  `apps/api/internal/agaapplicability/`.
- New shared synthetic persistence:
  `apps/api/internal/preproddata/agademoworkspace/` and PostgreSQL schema
  `preprod_aga_demo_workspace`.
- New service/API boundary:
  `apps/api/internal/agademoworkspace/` and
  `apps/api/internal/httpapi/aga_demo_workspace_api.go`.
- Tagged API wiring: `apps/api/cmd/api/profile_preproddemo.go`,
  `apps/api/cmd/api/profile_runtime.go`, `apps/api/cmd/api/main.go`,
  `apps/api/cmd/api/profile_preproddemo_test.go`, and
  `apps/api/internal/platform/config/config.go`. The existing five-route
  `NewAGACandidateDemoHandler` and `ProtectAGACandidateDemo` remain unchanged;
  the tagged main mounts a separate workspace handler/protector.
- One-shot workspace provisioning/loading:
  `apps/api/cmd/preprod-aga-demo-workspace-role-provisioner/main.go`,
  `apps/api/cmd/preprod-aga-demo-workspace-fixture-exporter/main.go`,
  `apps/api/cmd/preprod-aga-demo-workspace-loader/main.go`,
  `apps/api/Dockerfile`, `deploy/local/compose.yaml`,
  `scripts/init-local-preprod-namespace.sh`,
  and `deploy/local/secrets/README.md`.
- OpenAPI source and generated transport:
  `api/openapi/source/paths/platform.json`,
  `api/openapi/source/schemas/platform.json`,
  `api/openapi/aviasurveil360.yaml`,
  `apps/api/internal/httpapi/generated/api.gen.go`, and
  `apps/web/src/generated/transport/api-types.ts`.
- Web transport and supplemental preprod routes:
  `apps/web/src/backend/aga-demo-workspace.ts`,
  `apps/web/src/app/aga-demo-workspace-routes.tsx`,
  `apps/web/src/auth/aga-demo-workspace-guard.tsx`,
  `apps/web/src/app/router.tsx`, and `apps/web/src/ui/role-navigation.tsx`.
  These capability-gated routes stay outside the frozen 86-route parity
  registry and are available only in the HTTP build after an authorized server
  capability response.
- Admin/Manager classification UI:
  `apps/web/src/features/admin/aga-candidate-demo-panel.tsx` and
  `apps/web/src/features/checklists/aga-classification-workspace-page.tsx`.
- Planning/recommendation entry:
  `apps/web/src/features/planning/new-audit-wizard.tsx`.
- Synthetic execution UI:
  `apps/web/src/features/inspections/aga-demo-inspection-page.tsx`,
  `apps/web/src/features/findings/aga-demo-potential-finding-page.tsx`, and
  `apps/web/src/features/caps/aga-demo-cap-evidence-page.tsx`, with the exact
  `*.test.tsx` files listed in Task 8.
- Boundary and connected qualification:
  `api/openapi/tests/aga-demo-workspace-contract.test.mjs`,
  `tests/aga-hybrid-demo-workspace-boundary.test.mjs`,
  `scripts/build-aga-hybrid-forbidden-inventory.mjs`,
  `scripts/test-aga-hybrid-demo-workspace-connected.sh`,
  `apps/web/playwright.config.ts`, and the three exact Task 8 E2E specs.

## Tagged API, Authorization, And Projection Contract

The old prefix stays exactly five Admin-only GET operations. New operations use
a separate fixed prefix and never put a question identity, digest, filter,
cursor, Draft ID, lifecycle ID, or organization ID in a URL or query string.
Every POST uses a closed discriminated body whose operation enum is frozen at
Gate 0.

| Method and path | Operations | Authorized actor and projection |
|---|---|---|
| Existing five GETs under `/v1/admin/governed-checklist/aga-candidate-demo` | Existing capability, summary, forms, form detail, questions | Current CAA Admin only; behavior and contract unchanged |
| `GET /v1/preprod/aga-demo-workspace/capability` | Role-scoped availability only; no counts or IDs | Current Admin role/membership, or an effective scoped workspace binding for Manager, Inspector, Lead/reviewer, or matching Auditee; all others receive neutral denial |
| `POST /v1/preprod/aga-demo-workspace/classification/query` | `GET_SUMMARY`, `GET_TAXONOMY`, `GET_PROVIDER_CONFIGURATION`, `SEARCH_ITEMS`, `GET_DRAFT`, `GET_HISTORY` | CAA Admin read-all or exact scoped Manager; no lifecycle-only role |
| `POST /v1/preprod/aga-demo-workspace/classification/commands` | `PREVIEW_BATCH`, `EXECUTE_BATCH`, and all Manager actions in the status registry | Exact scoped Manager only; Admin is read-only for classification decisions |
| `POST /v1/preprod/aga-demo-workspace/recommendations/commands` | `CREATE_RECOMMENDATION`, `CREATE_INSPECTION` | Exact scoped Manager with current readiness pin |
| `POST /v1/preprod/aga-demo-workspace/lifecycle/query` | `GET_INSPECTION`, `GET_FINDING`, `GET_CAP_EVIDENCE` | Admin full synthetic history; Manager department/unit scope; assigned Inspector/Lead; reviewer binding; Auditee matching exact organization/provider-scope projection |
| `POST /v1/preprod/aga-demo-workspace/lifecycle/query` | `GET_ROLE_HISTORY` | CAA-only: Admin full history; Manager department/unit scope; assigned Inspector/Lead/reviewer only for their exact synthetic inspection; never Auditee |
| `POST /v1/preprod/aga-demo-workspace/lifecycle/commands` | `START_INSPECTION`, `RECORD_RESPONSE`, `CREATE_POTENTIAL_FINDING`, `SUBMIT_CHECKLIST`, `REOPEN_CHECKLIST`, `RETURN_POTENTIAL_FINDING`, `DISMISS_POTENTIAL_FINDING`, `CONVERT_POTENTIAL_FINDING`, `SUBMIT_CAP_REVISION`, `REVIEW_CAP`, `SUBMIT_EVIDENCE_VERSION`, `VERIFY_EVIDENCE`, `AUTHORIZED_CLOSE` | Exact operation-role matrix below; never broad role-only access |
| `POST /v1/preprod/aga-demo-workspace/admin/commands` | `RESET_GENERATION` | Current CAA Admin only; reason plus exact generation ID/revision/seal-digest CAS |

Every workspace POST requires the existing CSRF header. Source OpenAPI marks
each operation with `x-operation-kind: query` or `command` and
`x-neutral-denial: true`. Query POSTs have neither `Idempotency-Key` nor
`If-Match`. Every command POST has the generic closed body envelope
`operationId`, `idempotencyKey`, and `expectedGenerationId`; the generation ID
is checked under the replay ordering below. Command POSTs require `Idempotency-Key`
equal to body `idempotencyKey`, and `If-Match: "rev-N"` equal to the operation-specific body
revision: `expectedDraftRevision` for classification/recommendation creation,
`expectedRecommendationRevision` for inspection creation,
`expectedLifecycleRevision` for lifecycle commands, or
`expectedGenerationRevision` for reset. A mismatch fails before service/store
use. Every lifecycle command additionally requires body
`expectedLifecycleDigest`; `If-Match` mirrors only the revision while the
server equality-checks both lifecycle values before store use.
`RESET_GENERATION` additionally requires body `expectedGenerationId` and
`expectedGenerationSealDigest`; `If-Match` mirrors only
`expectedGenerationRevision`, while the server equality-checks all three before
any reset write. The OpenAPI bundler must not auto-advertise distinguishable 401/403
responses for these neutral operations; their broad/scope/missing-object
denials are the same no-store 404, while an already authorized malformed body
may be 400 and CAS/idempotency conflicts may be 409/412.

After neutral authentication/CSRF, current membership, closed-body/header
equality validation, and current exact operation/scope authorization, a command
first queries only the neutral
idempotency binding store for `(expectedGenerationId, actorSubjectId,
operationId, idempotencyKey)`. An exact existing binding with the same command
and canonical semantic hash returns its stored response without generation or
domain-object lookup only when the binding's stored closed authorization-scope
digest still matches the actor's current membership, organization, workspace
binding, operation role, and request scope. Thus an exact current-Admin reset
replay may bypass only current-generation CAS after the generation becomes
`RESET`; revoked/stale membership or binding, changed organization, or wrong
operation role receives neutral denial and no stored response. Any token/
counterpart/operation/hash mismatch conflicts with no domain lookup. Only an
unseen pair proceeds to require `expectedGenerationId` be the current `ACTIVE`
generation and then performs operation-specific CAS/object lookup/write. Tests
cover response loss and exact reset replay after commit, revoked/stale/wrong-
scope replay denial, plus unseen or nonmatching stale-generation requests
writing nothing.

`ProtectAGADemoWorkspace` is a new mutation-aware neutral boundary. It applies
no-store headers before authentication, authenticates to a discard sink,
validates CSRF on every POST, establishes that the principal has at least one
effective workspace authority before parsing the body, then performs exact
scope authorization before any domain-object lookup. Authentication, CSRF,
role, stale-membership, wrong-scope, cross-organization, missing-object, and
direct-ID denials use the same no-store 404 body and length. An authorized
actor's malformed closed body may return 400 only after this pre-authorization.

The React application adds fixed supplemental paths
`/admin/aga-demo-workspace`, `/department-manager/aga-demo-workspace`,
`/inspector/aga-demo-workspace`, `/lead-inspector/aga-demo-workspace`, and
`/auditee/aga-demo-workspace`. Server capability and operation authorization,
not `buildProfile` or a client role selector, controls visibility and data.
The demo build has no workspace backend and shows no route/link; the HTTP build
does not show one until capability succeeds.

## Database Role And Cross-Schema Contract

| Connection/role | Allowed | Forbidden |
|---|---|---|
| Existing normal session pool / `preprod_normal_api` | Existing OIDC/session operations only | Any overlay or workspace schema grant |
| Existing `preprod_aga_demo_reader` | Existing sealed overlay views only | Workspace access, base tables, DML, role change |
| `preprod_aga_demo_workspace_owner` | One-shot schema/role/function ownership; `NOLOGIN NOINHERIT` | Runtime credential |
| `preprod_aga_demo_workspace_fixture_exporter` | One-shot, read-only exact predecessor scenario/current identity-membership projection used to create/verify the private fixture | DML; workspace/overlay access; broad canonical reads; runtime use |
| `preprod_aga_demo_workspace_loader` | One-shot taxonomy/classification plus exact synthetic fixture load and seal before runtime | Draft/lifecycle commands after seal; overlay/canonical access |
| `preprod_aga_demo_workspace_reader` | `SELECT` on explicit sealed/workspace projection views; default read-only | Base-table DML/DDL, overlay/canonical access, role change |
| `preprod_aga_demo_workspace_command` | `EXECUTE` only on enumerated workspace-owned command functions | Direct table DML/DDL, loader/seal functions, overlay/canonical access, role change |

All roles are least-privilege `NOSUPERUSER NOCREATEDB NOCREATEROLE
NOINHERIT`; PUBLIC/default privileges and database/schema `CREATE` and `TEMP`
are revoked as applicable. Workspace command functions are owner-held
`SECURITY DEFINER` functions with exact signatures granted only to the command
role, schema-qualified relation references, and fixed safe
`search_path=pg_catalog,preprod_aga_demo_workspace,pg_temp`; the owner remains
`NOLOGIN`, and the command role cannot set role or search path. No cross-schema
view, foreign key, function, trigger, or SQL join is permitted. Task 9 probes
direct DML, unlisted function execution, temp-object shadowing, role/search-
path changes, and PUBLIC/default privilege drift. The tagged service uses four
separate pools: normal session, overlay reader, workspace reader, and workspace
command. It authorizes first, reads exact identity/digest from both read pools
in separate transactions, fails closed on mismatch, and only then composes a
response in Go.

The exporter role has column-limited `SELECT` only on
`preprod_loader.scenario_records` and the current identity-reference, desired-
membership-version, and membership-observation columns needed to prove
subject/membership/organization/role equality. The disposable namespace must
contain exactly one predecessor run, every exporter query must include that
bound run/subject set, and static SQL tests reject a missing predicate or extra
column. The role has no sequence, DML, workspace, overlay, session, department,
functional-assignment, or business-domain grant; the exporter composes separate
reads in Go and rejects any extra account or role family.

The existing smoke OIDC fixture supplies current synthetic subjects,
memberships, organizations, and broad application roles, but it supplies no
`caa_department_memberships`. This workspace does not create one and does not
populate `Principal.DepartmentAssignments` or call
`identity.CanTechnicallyReviewUnit` as authority. Instead, after the broad
current role/membership check, the workspace service requires a current
workspace-only binding for the same subject and membership to an explicitly
synthetic department and organizational unit. Those bindings authorize only
the `DEMO_*` workspace operations in this plan and are never represented as
canonical Department Manager, checklist-review, technical-approval, or
publication authority.

The existing OIDC/session middleware necessarily updates bounded authentication
control-plane rows on authenticated requests: `session_references` idle/
observation fields and, when a fresh provider observation is required,
observation columns of `desired_membership_sync`; login/logout may also append
an authentication audit event. Task 9 classifies those exact tables/columns as
`AUTH_CONTROL_PLANE`, measures their permitted deltas separately per request,
and never treats them as workspace writes. Canonical identity references,
membership versions/requests, department/functional assignments, Provider,
Publication, Audit, Finding, CAP, Evidence, and their sequences remain in
`FORBIDDEN_BUSINESS` with exact zero delta. It must not claim that the entire
database is byte-identical across authenticated requests or logout. No
canonical provider, identity, membership version, department, functional
assignment, or business record may be provisioned by a workspace command; the
one-shot loader creates only the separately labeled synthetic fixture objects
inside its own schema.

## Synthetic Lifecycle State Contract

Every lifecycle command uses the generic command envelope plus exact
`expectedLifecycleRevision` and `expectedLifecycleDigest`; reset uses that
generic envelope plus its generation revision/seal fields. No command may omit
`operationId`, substitute the header key for body `idempotencyKey`, or infer a
generation from the object ID.

| Command operations | Required effective workspace authority |
|---|---|
| `CREATE_RECOMMENDATION`, `CREATE_INSPECTION` | Exact scoped Department Manager |
| `START_INSPECTION`, `RECORD_RESPONSE`, `CREATE_POTENTIAL_FINDING`, `SUBMIT_CHECKLIST` | Inspector assigned to the exact synthetic inspection/questions |
| `REOPEN_CHECKLIST` | Assigned Inspector or exact Lead; checklist `SUBMITTED`, inspection `SUBMITTED`/`COMPLETED`, non-empty reason, current CAS |
| `RETURN_POTENTIAL_FINDING`, `DISMISS_POTENTIAL_FINDING`, `CONVERT_POTENTIAL_FINDING` | Exact Lead for the synthetic inspection |
| `SUBMIT_CAP_REVISION`, `SUBMIT_EVIDENCE_VERSION` | Auditee with exact matching organization/provider-scope binding |
| `REVIEW_CAP` | Exact `AGA_DEMO_CAA_REVIEWER` binding held by Lead or scoped Manager |
| `VERIFY_EVIDENCE` | Inspector or Lead bound to the exact synthetic inspection |
| `AUTHORIZED_CLOSE` | Exact scoped Department Manager plus reason |
| `RESET_GENERATION` | Current CAA Admin plus reason and exact `expectedGenerationId`/`expectedGenerationRevision`/`expectedGenerationSealDigest` |

- A created `DEMO_INSPECTION` starts `READY`, moves to `IN_PROGRESS` only when
  the assigned Inspector starts it, to `SUBMITTED` when its checklist is
  submitted with at least one nonterminal Potential Finding root, and to
  `COMPLETED` when submission has zero Potential Finding roots or when the last
  root reaches terminal latest `DISMISSED` or `CONVERTED`. The zero-root
  submission and last-terminal-root transition atomically update checklist and
  inspection projections. `RETURNED` is not
  terminal: the Lead's reasoned return atomically moves the checklist and
  inspection back to `IN_PROGRESS`. The assigned Inspector corrects the
  response, appends a successor Potential Finding under the same root, and
  resubmits before the Lead can dismiss, convert, or return it again. Prior
  versions remain immutable. Open converted Findings may remain in follow-up
  after inspection completion. Checklist execution uses exactly `IN_PROGRESS`
  and `SUBMITTED`. `REOPEN_CHECKLIST` is allowed only when the checklist is
  `SUBMITTED` and its inspection is `SUBMITTED` or `COMPLETED`; an assigned
  Inspector or exact Lead supplies a non-empty controlled reason and current
  lifecycle CAS. It atomically returns both checklist and inspection to
  `IN_PROGRESS`, preserves the pinned question snapshot and every existing
  Finding/root version, and permits corrected responses or new successor roots.
  It is denied from `READY`/`IN_PROGRESS`, a reset generation, or stale CAS.
- Answers are exactly `COMPLIANT`, `NON_COMPLIANT`, `OBSERVATION`,
  `NOT_APPLICABLE`, and `NOT_CHECKED`. `NON_COMPLIANT` or `OBSERVATION` plus a
  required `commentToAuditee` makes a Potential Finding eligible.
- An Inspector creates `DEMO_POTENTIAL_FINDING` in
  `PENDING_LEAD_REVIEW`. Only the exact Lead may move it to `RETURNED`,
  `DISMISSED`, or `CONVERTED`. Conversion records a human-selected severity and
  explicit CAP, Evidence, and due-date requirements; Inspector execution never
  creates a Finding directly.
- Lead conversion records three independent human choices: exact booleans
  `capRequired`, `evidenceRequired`, and `dueDateRequired`, with `dueDate`
  present and valid if and only if `dueDateRequired=true`. The due-date choice
  does not change the state branch. It creates `DEMO_FINDING` in exactly one
  initial state from the CAP/Evidence choices: CAP required/Evidence required and CAP
  required/no Evidence both start `WAITING_FOR_CAP`; no CAP/Evidence required
  starts `EVIDENCE_REQUIRED`; no CAP/no Evidence starts `PENDING_CLOSURE` with
  next action `AUTHORIZED_CLOSE`. The no-CAP branches expose no CAP
  command, and the no-Evidence branches expose no Evidence submission command.
- `DEMO_FINDING` uses the applicable canonical candidate states
  `WAITING_FOR_CAP`, `CAP_SUBMITTED`, `CAP_REJECTED`,
  `CAP_MORE_INFORMATION_REQUESTED`, `EVIDENCE_REQUIRED`,
  `EVIDENCE_SUBMITTED`, `PENDING_CAA_REVIEW`,
  `EVIDENCE_MORE_INFORMATION_REQUESTED`, `PENDING_CLOSURE`, and `CLOSED`.
  The canonical vocabulary also permits `CAP_ACCEPTED`, but this synthetic
  projection deliberately uses the listed subset: CAP acceptance records the
  CAP revision as `ACCEPTED` and maps the still-open Finding directly to
  `EVIDENCE_REQUIRED` or `PENDING_CLOSURE`. Omitting that intermediate
  projection state never turns CAP acceptance into Finding closure.
- `DEMO_CAP_REVISION` uses `SUBMITTED`, `PENDING_CAA_REVIEW`,
  `ACCEPTED`, `REJECTED`, `MORE_INFORMATION_REQUESTED`, and `SUPERSEDED`.
  The matching Auditee submits root cause, corrective action, preventive
  action, responsible person, target date, and reasoned revisions. An exact
  `AGA_DEMO_CAA_REVIEWER` binding held by a Lead or scoped Manager reviews it.
  `SUBMIT_CAP_REVISION` is valid only from `WAITING_FOR_CAP`, `CAP_REJECTED`, or
  `CAP_MORE_INFORMATION_REQUESTED`; it appends a new immutable revision and,
  when a prior non-accepted revision exists, appends a supersession link/event
  whose derived historical projection is `SUPERSEDED` without updating that
  prior revision's body, state event, or digest. It then appends
  ordered `SUBMITTED`/`CAP_SUBMITTED` then
  `PENDING_CAA_REVIEW`/`PENDING_CAA_REVIEW` events in the same transaction.
  The committed latest CAP and Finding projections are both
  `PENDING_CAA_REVIEW`; no intermediate projection is externally writable.
  `REVIEW_CAP` is valid only there. Reject and more-information outcomes map
  the CAP and Finding to their corresponding states with next action
  `SUBMIT_CAP_REVISION`; accept maps as defined below.
  Every `REVIEW_CAP` outcome requires separate `commentToAuditee` and
  `internalCaaNote`; Auditee projections expose only the former. CAP acceptance
  leaves the Finding open.
- `DEMO_EVIDENCE_VERSION` is append-only metadata and a mock filename only; no
  `uploadState` or `scanState`, file bytes, upload, malware scan, clean result,
  or chain-of-custody claim is created. The matching Auditee submission starts
  exact `reviewState=PENDING_CAA_REVIEW`. Inspector or Lead records `CLOSE`,
  `PARTIALLY_CLOSE`, `NOT_CLOSE`, or `REQUEST_MORE_INFORMATION` with separate
  required `commentToAuditee` and `internalCaaNote`.
- `SUBMIT_EVIDENCE_VERSION` is valid only from `EVIDENCE_REQUIRED` or
  `EVIDENCE_MORE_INFORMATION_REQUESTED`. It appends a new immutable metadata
  version and ordered Finding events `EVIDENCE_SUBMITTED` then
  `PENDING_CAA_REVIEW` in one transaction; its committed latest Evidence
  review state and Finding projection are `PENDING_CAA_REVIEW`. A prior
  Evidence version remains immutable and is never overwritten. The exact next
  action is `VERIFY_EVIDENCE`; every non-close outcome below returns the next
  action to `SUBMIT_EVIDENCE_VERSION`.
- CAP acceptance maps a Finding with Evidence required to `EVIDENCE_REQUIRED`;
  when conversion explicitly required no Evidence, it maps to
  `PENDING_CLOSURE`. That latter state remains open with exact next action
  `AUTHORIZED_CLOSE`; only the scoped Manager's reasoned command may close it
  with `AUTHORIZED_CLOSURE` basis.
- `VERIFY_EVIDENCE` atomically appends the verification and moves the latest
  Evidence review state plus Finding: `CLOSE` maps directly to `ACCEPTED` and
  `CLOSED` with closure basis `EVIDENCE_VERIFIED`; `PARTIALLY_CLOSE` maps to
  `PARTIALLY_ACCEPTED` and
  `EVIDENCE_MORE_INFORMATION_REQUESTED`; `NOT_CLOSE` maps to `REJECTED` and
  `EVIDENCE_MORE_INFORMATION_REQUESTED`; `REQUEST_MORE_INFORMATION` maps to
  `MORE_INFORMATION_REQUESTED` and
  `EVIDENCE_MORE_INFORMATION_REQUESTED`. Only the close path closes the
  Finding. Exact scoped Department Manager `AUTHORIZED_CLOSE` is a separate
  reason-required path allowed only from exact Finding state `PENDING_CLOSURE`;
  it is denied from `WAITING_FOR_CAP`, every CAP review state,
  `EVIDENCE_REQUIRED`, `EVIDENCE_SUBMITTED`, `PENDING_CAA_REVIEW`, and
  `EVIDENCE_MORE_INFORMATION_REQUESTED`. It never changes an Evidence review
  state or impersonates Evidence verification.
- Auditee projections structurally omit `internalCaaNote`, classification
  rationale/confidence/blockers, CAA workload, other organizations, private
  risk, enforcement deliberation, subject/membership/binding IDs or revisions,
  assignment chronology, and internal CAA role history. `GET_ROLE_HISTORY`
  returns the same neutral denial to every Auditee. Every Auditee projection
  shows only a public role label/current action owner, due date when applicable,
  and next action.

## Umbrella Decomposition And Stop Gates

This cross-cutting umbrella is not executable safely as one uninterrupted
change. It remains the sole active goal and defines these ordered child-plan
boundaries inside this file; this review creates no competing ExecPlan. Each
slice has one writer, stops after its acceptance gate, updates this umbrella,
and requires review before the next slice starts.

| Slice | Included work | Required stop |
|---|---|---|
| A | Gate 0A discovery receipt and Gate 0B taxonomy/prompt/schema/status freeze | Independent contract review; no classification output |
| B | Tasks 1–2 pure domain, bounded two-pass outputs, reconciliation, sealed text-free candidate | 1,310/1,310 per-pass and final bijection review |
| C | Tasks 3–4 isolated schema, roles, one-shot loader, tagged API/auth/OpenAPI | Static/unit/tagged-build review; no UI or connected claim |
| D | Tasks 5–6 capability-gated classification UI and exact recommendation | React/pure-service review; no lifecycle implementation |
| E | Tasks 7–8 product-correct synthetic lifecycle backend/UI and nonzero browser discovery | Unit/contract review; browser execution still `not run` |
| F1 | Task 9 inventory/auth contracts plus exhaustive deterministic journal/commit/publication fault tests | Static/unit/integration review; no container/browser claim |
| F2 | Task 9 one fresh happy-path prepare/qualify/cleanup run | Connected isolation/privacy evidence review before recovery targets |
| F3 | Task 9 exact four-case connected recovery matrix on separate fresh targets | Recovery/authority/residue evidence review |
| F4 | Task 10 offline aggregate verification and handoff | Full evidence review; no release/production claim |

## Ordered Work Packages

### Gate 0 — Freeze The Classification Contract

**Files:**

- Create `docs/product-specs/data-and-rules/AGA_HYBRID_QUESTION_CLASSIFICATION.md`.
- Create `docs/product-specs/data-and-rules/aga-question-classification-taxonomy.v1.json`.
- Create `docs/product-specs/data-and-rules/aga-question-classification.schema.json`.
- Create
  `docs/product-specs/data-and-rules/AGA_HYBRID_VOCABULARY_DISCOVERY_PROMPT.md`.
- Create `docs/product-specs/data-and-rules/AGA_HYBRID_CLASSIFICATION_PROMPT.md`.
- Create `scripts/prepare-aga-hybrid-classification-batches.mjs`.
- Create `scripts/check-aga-hybrid-created-files.mjs`.
- Create `tests/aga-hybrid-classification-plan-contract.test.mjs`.
- Create `tests/fixtures/aga-hybrid-created-file-inventory.v1.json` containing
  every create-path owned by Gate 0 and Tasks 1–10.
- Create text-free Gate 0 receipts under
  `deliverables/aga-question-classification-contract-v1/`:
  `discovery-run.json`, `batch-manifest.json`, `taxonomy-discovery.json`, and
  `omission-review-inventory.json`. Gate 0B additionally creates
  `classification-batch-manifest.json`; this second artifact never replaces or
  rewrites the accepted discovery manifest.
- Modify `docs/product-specs/index.md` only to add the new specification link.

**Gate 0A — discovery RED/GREEN:**

1. Write `TestGate0DiscoveryRequiresExactTextFreeCoverage` in the Node contract
   test. It requires the fixed byte/hash facts, exact 52-form package order
   including `035A` and excluding nonexistent `035`/`049`, 1,310 full
   identities, the 49/51 overlap, digest non-uniqueness, bounded batches, and
   no question-body field. Run it and record the expected missing-script/
   missing-receipt failure.
   Add `TestGate0DiagnosticsAreIdentityRedacted`; it intentionally mismatches a
   real-derived identity, captures stdout/stderr, and fails if any body, form,
   proposal ID, ordinal, text digest, source reference, or private path appears.
2. Implement the preparer with stable package order and greedy batches capped
   at both 64 items and 98,304 serialized input bytes. The manifest records
   only identity/digest, batch ordinal, count/byte count, and batch/input
   digests; its ordered union must equal the package exactly.
3. Write and freeze the separate vocabulary-discovery prompt before any model
   call. The Gate 0A test records its tracked-file SHA-256 and rejects the later
   row-classification prompt as a substitute. Run vocabulary-only Codex
   discovery over every manifest batch with no row classification. Record a
   new immutable discovery run with that discovery prompt digest, exact tool-
   reported model descriptor/config, per-batch input/output digests, and text-
   free candidate vocabularies. If the runtime does not expose an exact model
   identifier/snapshot descriptor, stop `blocked`; never invent it or claim a
   model-weights digest.
4. Convert every lexical/semantic omission cue into a full identity plus frozen
   `signalRuleId`; reject body text and unruled numeric counts. Re-run the test
   and require the discovery subtest to pass, then stop for primary-writer
   review before Gate 0B.

**Gate 0B — freeze RED/GREEN:**

1. Extend the same test to require the exact 18 domains, six involvement roles,
   seven canonical target kinds, three recommendation states, 14/6 research-
   derived provider partition, field/status registry, fatal-run rules,
   confidence precedence, controlled rationale codes/fact selectors/rule
   combinations, exact proposal-field/value evidence binding, closed
   edge-specific external-involvement provenance, ordered-versus-set array
   normalization, Draft-item transitions and full proposal resolution,
   the closed two-record-per-identity pass-proposal schema/reference/digest,
   the closed Base-versus-Workspace `questionRef`/`parentQuestionKey` unions
   and server-issued add/reword identity transitions,
   profile/target compatibility, qualifier completeness, lifecycle transition
   enums, digest canonicalization, prohibited provider-hierarchy fields, and
   fixed prompt/schema/spec digests, and distinct discovery-versus-
   classification manifest roles. The classification manifest has exactly 25
   ordered batches, at most 64 items each, and a stored worst-case full-envelope
   byte count at most 98,304. Run it and record the expected missing/unfinished
   freeze assertion.
2. Review the discovery receipts, freeze all controlled arrays and compatibility
   tables in taxonomy v1, write the row-classification prompt/spec/schema, and
   make every object and array closed under the rules above. Preserve all four
   existing Gate 0A artifacts byte-for-byte. Deterministically create the
   separate closed classification manifest over the same ordered 1,310 source
   items. Its package-order greedy partition is sized from canonical JSON only,
   excluding digest-domain framing, with maximum-valid ASCII classification and
   pass run IDs, both roles, and fixed-width SHA pins. Freeze separate domains
   and exact preimages for the role-neutral source snapshot, ordered identities,
   batch entry, and manifest root; batch/root digests exclude only their own
   digest fields. The source snapshot contains only batch ordinal, the 11 source
   fixed-input digests, and full private items, so it precedes and cannot depend
   on taxonomy, prompt, schema, or the manifest itself. Bind the 24 discovery
   input digests in ordinal order as prohibited classification digests. Re-run
   the same test and require pass. A new code after this point requires taxonomy
   v2; no classification run may start before this gate is independently
   accepted.
3. Freeze the complete planned-create inventory. The checker rejects a missing,
   extra, duplicate, or out-of-scope inventory path, records each owning slice,
   and with `--through <slice>` directly scans every file due through that
   slice for trailing whitespace; it also parses JSON and checks Markdown fence
   balance. This closes the untracked-file gap that `git diff --check` cannot
   see while allowing later-slice paths to remain absent and leaving unrelated
   user files outside the inventory.

**Commands:**

```bash
node --test tests/aga-hybrid-classification-plan-contract.test.mjs
node scripts/prepare-aga-hybrid-classification-batches.mjs --package deliverables/aga-all-forms-source-risk-draft-2026-08-01/AGA_ALL_FORMS_SOURCE_RISK_DRAFT.json --research-zip /Users/marlonjd/.codex/attachments/a2fa9639-5e9a-4e5d-a68d-0b38ef797b75/AGA_INDEPENDENT_RESEARCH_DELIVERABLES_2026-08-02.zip --workbook /tmp/codex-remote-attachments/019fc297-2e52-7403-a817-337ba1432877/1BC79425-401B-4F87-BD0B-7C543BC1E6F0/1-AVIASURVEIL360_System_Design_Matrix.xlsx --max-items 64 --max-input-bytes 98304 --discovery-run deliverables/aga-question-classification-contract-v1/discovery-run.json --output deliverables/aga-question-classification-contract-v1/batch-manifest.json --classification-output deliverables/aga-question-classification-contract-v1/classification-batch-manifest.json
node --test tests/aga-hybrid-classification-plan-contract.test.mjs
node scripts/check-aga-hybrid-created-files.mjs --inventory tests/fixtures/aga-hybrid-created-file-inventory.v1.json --through gate0
node tests/harness-docs-smoke.test.js
node -e 'const fs=require("node:fs");const p="docs/exec-plans/active/2026-08-03-aga-hybrid-classification-demo-lifecycle-plan.md";const s=fs.readFileSync(p,"utf8");if(s.split(/\n/u).some((line)=>/[ \t]+$/u.test(line)))throw new Error("trailing whitespace");if(((s.match(/^```/gmu)||[]).length%2)!==0)throw new Error("unbalanced fences");console.log("aga-hybrid-plan-file-scan: ok")'
git diff --check
```

**Acceptance:** Gate 0A proves text-free 1,310/1,310 discovery coverage and Gate
0B closes the separate 25-batch classification manifest, taxonomy, prompt,
schema, statuses, transitions, compatibility, and digest rules before any
classification output. Runtime validation must prove `actual canonical bytes <=
stored worst-case bytes <= 98,304`, while every accepted Gate 0A artifact, the
accepted package, and the sealed overlay remain unchanged. Record both red
failures and green passes literally.

### Task 1 — Implement The Pure Classification, Draft, And Recommendation Domain

**Files:**

- Create `apps/api/internal/agaapplicability/types.go`.
- Create `apps/api/internal/agaapplicability/taxonomy.go`.
- Create `apps/api/internal/agaapplicability/classification.go`.
- Create `apps/api/internal/agaapplicability/draft.go`.
- Create `apps/api/internal/agaapplicability/recommendation.go`.
- Refactor `apps/api/internal/agaapplicability/applicability_test.go` into the
  classification-contract tests while preserving its identity, provenance,
  exact-selection, and immutable-successor intent.
- Create `apps/api/internal/agaapplicability/draft_test.go` and
  `apps/api/internal/agaapplicability/recommendation_test.go`.

**Focused verification sequence:**

1. Add focused tests named `TestClassifySealedBaseContract`,
   `TestClassificationFatalErrorsAbortRun`,
   `TestConfidenceRecommendationPrecedence`,
   `TestConfidenceEvidenceBindsEveryProposal`,
   `TestExternalInvolvementEdgesAreIndependentAndOptional`,
   `TestPassProposalRecordsAreCompleteAndResolvable`,
   `TestClassificationDigestGraphIsNonCircular`,
   `TestDraftCommandsCreateImmutableSuccessors`,
   `TestDraftSemanticEditDemotesAutoPreselection`,
   `TestDraftResolvesEveryProposalFamily`,
   `TestQuestionReferenceUnionIsClosed`,
   `TestAddAllocatesFreshWorkspaceRootVersionAndProposal`,
   `TestDraftRewordReplacesCurrentLeaf`,
   `TestWorkspaceQuestionIdentityRejectsAliases`,
   `TestQuestionSnapshotReconstructsExactLeaves`,
   `TestDraftBatchPreviewIsAtomic`, and
   `TestRecommendRequiresCurrentIncludedLeaf`. Run the focused regex below;
   record the focused result for undefined new types/functions; do not skip the
   test.
2. Cover one main domain, optional involvement, exact provenance, accepted
   duplicate text digests, rejected duplicate full identities, unknown codes,
   canonical-kind/profile mismatch, validator-recomputed field/value-bound
   evidence references,
   fatal unknown/mismatched selectors, LOW on missing core evidence, MEDIUM on
   missing auxiliary evidence, HIGH only when every emitted proposal is
   independently supported, reorder-equivalent set arrays, duplicate
   rejection, zero/one/multiple edge-specific involvement objects, duplicated
   research-role input not becoming duplicated edges, total confidence mapping,
   exactly 1,310 candidate plus 1,310 challenge proposal records whose digests
   resolve from every final item, and preservation of all governance blockers.
3. Cover immutable successor commands, the exact reason table, source-gap
   inclusion, 500-item batch limit, filter/identity preview digest, stale
   revision/content conflict, semantic-edit recommendation/review demotion and
   null Draft confidence while sealed confidence remains immutable, candidate/challenge/exact
   resolution of every editable proposal family, subsequent disposition,
   new/reworded workspace versions always starting a new source gap even when
   the parent had a source proposal, server-issued add/reword IDs, exact parent
   keys, duplicate root/version/proposal rejection, repeated body digests on
   distinct roots remaining distinct, byte-identical reword rejection,
   non-current/missing/cyclic/cross-generation/cross-root parent rejection,
   current-leaf/snapshot reconstruction, exact
   `ACTIVE` generation reference to one `SEALED` 1,310-row run for readiness,
   and base count remaining 1,310.
4. Cover every recommendation request field, equality of `draftRevision` and
   `expectedDraftRevision`, server-derived provider code and qualifier
   completeness, exact readiness pin, current-leaf-only selection, inclusion
   of only `draftDisposition=INCLUDE`, exclusion of null/`EXCLUDE`/`DEFER` and
   superseded leaves, stable order, and every fail-closed mismatch.
5. Implement the smallest pure Go domain with no PostgreSQL, HTTP, identity, or
   model dependency. Re-run the focused regex for GREEN, then the full package.

**Commands:**

```bash
go -C apps/api test -count=1 ./internal/agaapplicability -run 'Test(ClassifySealedBaseContract|ClassificationFatalErrorsAbortRun|ConfidenceRecommendationPrecedence|ConfidenceEvidenceBindsEveryProposal|ExternalInvolvementEdgesAreIndependentAndOptional|PassProposalRecordsAreCompleteAndResolvable|ClassificationDigestGraphIsNonCircular|DraftCommandsCreateImmutableSuccessors|DraftSemanticEditDemotesAutoPreselection|DraftResolvesEveryProposalFamily|QuestionReferenceUnionIsClosed|AddAllocatesFreshWorkspaceRootVersionAndProposal|DraftRewordReplacesCurrentLeaf|WorkspaceQuestionIdentityRejectsAliases|QuestionSnapshotReconstructsExactLeaves|DraftBatchPreviewIsAtomic|RecommendRequiresCurrentIncludedLeaf)$'
go -C apps/api test -count=1 ./internal/agaapplicability
```

**Acceptance:** Pure domain tests prove identity, taxonomy, provenance,
confidence, Draft immutability, new wording isolation, exact selection, and
blocker preservation. Both RED and GREEN outputs are recorded in Progress.

### Task 2 — Produce And Validate The Complete Two-Pass AI Candidate Artifact

**Files:**

- Create `apps/api/cmd/aga-question-classification-validator/main.go`.
- Create `apps/api/cmd/aga-question-classification-validator/validator.go`,
  `validator_test.go`, and bounded synthetic fixtures
  `testdata/exact-two-item-pass.json`,
  `testdata/duplicate-identity-pass.json`,
  `testdata/question-text-leak-pass.json`, and
  `testdata/unknown-code-pass.json`.
- Create `deliverables/aga-question-classification-candidate-2026-08-03/`
  containing `manifest.json`, `batch-manifest.json`, `pass-one-run.json`,
  `pass-one-results.json`, `pass-two-run.json`, `pass-two-results.json`,
  `reconciliation.json`, `question-classifications.json`,
  `question-classifications.csv`, `ambiguity-review.csv`, and
  `aggregates.json`, plus text-free `pass-isolation-cleanup.json`.
- Create `tests/aga-question-classification-candidate.test.mjs`.

**Focused verification and AI sequence:**

1. Add focused tests named `TestValidatePassRequiresExactBijection`,
   `TestValidatePassRejectsTextAndUnknownCodes`,
   `TestValidatePassRecomputesConfidenceEvidence`,
   `TestValidatorDiagnosticsAreIdentityRedacted`,
   `TestReconcileUsesCandidateChallengePrecedence`,
   `TestReconcilePersistsBothPassProjections`, and
   `TestCandidateReconstructsAggregates`. Run the focused Go command and record
   the result for the missing validator/subcommands.
2. Implement `validate-pass`, `reconcile`, and `validate-candidate`. The tool
   consumes Gate 0B's identical 25-batch classification manifest for both
   passes and never calls a model. It reconstructs every source-snapshot,
   selected fact digest, and signal rule in memory from fixed inputs; requires
   actual canonical bytes not to exceed the manifest's stored worst-case bound;
   and applies the exact fatal-versus-LOW rules above. Re-run the focused
   command for GREEN.
3. Pin package, loader ZIP, taxonomy, prompt, provider catalog, research,
   workbook, lifecycle-authority, and batch-manifest digests in each pass/run
   receipt. Require lexical SHA-256 prompt and model receipt digests and exact
   prompt equality across the two supplied passes, but retain those immutable
   ZIP receipt digests as source provenance instead of rebinding them to the
   repository's former fixed prompt or locally recomputed descriptor digest.
   `modelDescriptorDigest` is still a source receipt digest, not a model-weights
   hash. Missing platform metadata remains literal `null` plus its exact
   `unavailableFields` marker and is accepted only as candidate-only demo
   provenance; no value is invented.
4. Launch both passes from the same frozen input snapshot into separately
   isolated read roots under one caller-owned empty 0700 private root, with
   0700 pass directories and 0600 files, before the primary writer materializes either pass
   output in the shared working tree. Each read root contains only fixed
   inputs, prompt, taxonomy, validator, and its assigned empty private output
   sink; neither contains the other pass's sink, transcript, receipt, or shared
   deliverable directory. Complete and seal both private pass outputs before
   copying either normalized result into tracked deliverables. Record per-
   batch blind-context receipts with read-root manifest digest, start/end time,
   context ID, and an explicit negative visibility scan. Reject the run if a
   pass-one artifact/transcript was visible to pass two (or vice versa), if
   either pass began after the other was materialized in the shared tree, or if
   isolation cannot be demonstrated. No external system or runtime LLM is
   called by repository code.
5. Normalize each batch response in memory into text-free controlled JSON.
   Reject any output that contains a question body or unknown field. Store no
   raw transcript/log. Validate and seal each pass only after its independent
   ordered union is exactly 1,310/1,310 with zero missing/extra/duplicate
   identities, exactly one immutable complete proposal record per identity,
   and an internally consistent source receipt graph for all 25 batch input,
   batch-output, record, and pass-seal digests. The ZIP's immutable input
   receipt digests are retained as source evidence for the supplied pass.
6. After both validated normalized seals are durably materialized, and on every
   failure path, remove each whole task-owned isolated read/output root and
   verify zero file/directory/process residue. A tracked text-free cleanup
   receipt contains only root-manifest digests, counts, and literal result; it
   contains no path, question body, identity, proposal ID, text digest, or raw
   response. `tests/aga-question-classification-candidate.test.mjs` requires
   this receipt and tests success/failure cleanup plus absence of leaked text.
7. Reconcile only two independently sealed passes. Preserve pass-one candidate
   values in the final item plus the complete pass-two challenge record, compute the total confidence/
   recommendation precedence, and validate all aggregate and exception files
   against the accepted package. Go and Node validators independently rebuild
   every item/aggregate/run digest using the frozen non-circular graph.

**Commands:**

```bash
go -C apps/api test -count=1 ./cmd/aga-question-classification-validator -run 'Test(ValidatePassRequiresExactBijection|ValidatePassRejectsTextAndUnknownCodes|ValidatePassRecomputesConfidenceEvidence|ValidatorDiagnosticsAreIdentityRedacted|ReconcileUsesCandidateChallengePrecedence|ReconcilePersistsBothPassProjections|CandidateReconstructsAggregates)$'
go -C apps/api test -count=1 ./cmd/aga-question-classification-validator
go -C apps/api run ./cmd/aga-question-classification-validator validate-pass --zip <caller-controlled-candidate-pass-zip> --private-root <new-empty-0700-candidate-root-outside-repository> --expected-pass pass-one
go -C apps/api run ./cmd/aga-question-classification-validator validate-pass --zip <caller-controlled-challenge-pass-zip> --private-root <new-empty-0700-challenge-root-outside-repository> --expected-pass pass-two
go -C apps/api run ./cmd/aga-question-classification-validator reconcile --package ../../deliverables/aga-all-forms-source-risk-draft-2026-08-01/AGA_ALL_FORMS_SOURCE_RISK_DRAFT.json --taxonomy ../../docs/product-specs/data-and-rules/aga-question-classification-taxonomy.v1.json --research-zip <caller-controlled-research-zip> --batch-manifest ../../deliverables/aga-question-classification-contract-v1/classification-batch-manifest.json --candidate ../../deliverables/aga-question-classification-candidate-2026-08-03 --candidate-pass <caller-controlled-candidate-pass-zip> --challenge-pass <caller-controlled-challenge-pass-zip>
go -C apps/api run ./cmd/aga-question-classification-validator validate-candidate --package ../../deliverables/aga-all-forms-source-risk-draft-2026-08-01/AGA_ALL_FORMS_SOURCE_RISK_DRAFT.json --taxonomy ../../docs/product-specs/data-and-rules/aga-question-classification-taxonomy.v1.json --research-zip <caller-controlled-research-zip> --batch-manifest ../../deliverables/aga-question-classification-contract-v1/classification-batch-manifest.json --candidate ../../deliverables/aga-question-classification-candidate-2026-08-03
node --test tests/aga-question-classification-candidate.test.mjs
```

**Acceptance:** Each pass independently proves exactly 1,310 unique full
identities and seals one complete proposal record per identity before
reconciliation; every final item resolves both immutable records. Supplied
receipt prompt/model digests remain immutable source evidence, while identity,
batch, record, batch-output, pass-seal, isolation, and privacy checks remain
independently validated. The candidate has one main domain per base
identity; all codes are in taxonomy v1; all 49 source gaps, 51 external
unresolved identities with their exact overlap, governance blockers,
disagreements, and manual-review states remain visible; aggregates reconstruct
from detail. Every tracked deliverable is text-free.

### Task 3 — Add The Isolated Shared Workspace Store

**Files:**

- Create `apps/api/internal/preproddata/agademoworkspace/contract.go`.
- Create `apps/api/internal/preproddata/agademoworkspace/postgres_roles.go`.
- Create `apps/api/internal/preproddata/agademoworkspace/postgres_roles_test.go`.
- Create `apps/api/internal/preproddata/agademoworkspace/postgres_provision.go`.
- Create
  `apps/api/internal/preproddata/agademoworkspace/postgres_provision_test.go`.
- Create `apps/api/internal/preproddata/agademoworkspace/postgres_store.go`.
- Create `apps/api/internal/preproddata/agademoworkspace/postgres_store_test.go`.
- Create `apps/api/internal/preproddata/agademoworkspace/store.go` and
  `store_test.go`.
- Create `apps/api/internal/preproddata/agademoworkspace/fixture.go` and
  `fixture_test.go`.
- Create `apps/api/cmd/preprod-aga-demo-workspace-role-provisioner/main.go` and
  `main_test.go`.
- Create
  `apps/api/cmd/preprod-aga-demo-workspace-fixture-exporter/main.go` and
  `main_test.go`.
- Create `apps/api/cmd/preprod-aga-demo-workspace-loader/main.go` and
  `main_test.go`.
- Create
  `docs/product-specs/data-and-rules/aga-demo-workspace-fixture.schema.json`
  and
  `tests/fixtures/aga-demo-workspace-authority-fixture-template.v1.json`.
- Create `tests/aga-hybrid-demo-workspace-boundary.test.mjs` with the Task 3
  schema/role/command/Compose closure, then extend it in later owning tasks.
- Modify `apps/api/Dockerfile`, `deploy/local/compose.yaml`,
  `scripts/init-local-preprod-namespace.sh`,
  and `deploy/local/secrets/README.md` for the exact role/credential matrix and
  one-shot services.

The preprod initializer alone creates 0600 secrets named
`preprod_aga_demo_workspace_fixture_exporter_database_password`,
`preprod_aga_demo_workspace_loader_database_password`,
`preprod_aga_demo_workspace_reader_database_password`, and
`preprod_aga_demo_workspace_command_database_password`. The tagged API accepts
only `AVIA_AGA_DEMO_WORKSPACE_READER_DATABASE_URL` and
`AVIA_AGA_DEMO_WORKSPACE_COMMAND_DATABASE_URL`; the loader URL/secret is mounted
only into its one-shot service and is removed after sealing. The exporter
secret is mounted only for `export`/`verify` and removed immediately after the
qualify-time verification. The ordinary
`scripts/init-local-secrets.sh` namespace is unchanged and receives a negative
boundary assertion.

Compose freezes profile `aga-demo-workspace-loader` and services
`preprod-aga-demo-workspace-role-provisioner`,
`preprod-aga-demo-workspace-fixture-exporter`, and
`preprod-aga-demo-workspace-loader`; the existing `preprod-aga-demo-api` stays
under `preproddemo`. Static verification renders the exact combined
`local-preprod-loader`, `aga-candidate-demo`,
`aga-candidate-demo-oidc-fixture`, `aga-demo-workspace-loader`, and
`preproddemo` profiles without starting them and checks every service, mount,
secret, network, read-only flag, capability, and dependency.

**Required schema families:** workspace generations/reset tombstones, taxonomy
versions, classification runs, immutable per-pass proposal records/edges/
evidence, reconciled classification items/involvement edges, Draft roots/revisions/
items, append-only workspace question versions, Manager decisions, batch
previews, synthetic authority bindings, provider scopes/targets,
recommendation/readiness snapshots, lifecycle streams/events, and idempotency
responses. Classification tables seal independently from mutable append-only
workspace event families.

The tracked fixture template contains only role slots, one explicitly
synthetic CAA department/unit, the catalog-backed 14/6 provider configuration,
one matching and one other-organization synthetic `AERODROME_OPERATOR` scope/
target, and Manager/Inspector/Lead/reviewer/Auditee binding rules. It contains
no live subject or membership. After the predecessor OIDC fixture, the one-shot
exporter reads only the exact smoke-profile provider-account/current-membership
rows, requires the fixed nine-account/eight-role-family matrix, and writes a
create-only 0600 private manifest binding those slots to subject, membership,
organization, base-run, target-fingerprint, template, and provider-catalog
digests. Its `export` mode creates that file; its later `verify` mode rereads
the current matrix and fails on any manifest/authority drift without writing.
It performs no database write. The workspace loader has no canonical
or overlay access; in one transaction it validates and loads the text-free
classification candidate, provider catalog, tracked template, and private
manifest, creates only synthetic workspace bindings/scopes/targets, and seals
a receipt over every input and row aggregate.

**Focused verification sequence:**

1. Add focused tests named `TestWorkspaceRoleMatrixIsClosed`,
   `TestWorkspaceLoaderReconcilesAndSeals`,
   `TestWorkspaceLoaderPersistsBothPassProjections`,
   `TestWorkspaceFixtureExporterIsReadOnlyAndExact`,
   `TestWorkspaceFixtureBindsSyntheticAuthorityOnly`,
   `TestWorkspaceCommandsAreAppendOnly`, and
   `TestWorkspaceQuestionReferencesRoundTripExactly`,
   `TestWorkspaceQuestionIdentityConstraintsAreClosed`,
   `TestWorkspaceResetIsForwardOnly`. Add static boundary assertions for the
   exact Docker/Compose/services/secrets. The focused result should identify
   missing packages, commands, services, and grants.
2. Implement the five new workspace-role contract above, including the bounded
   read-only fixture-exporter role. The loader accepts only the exact text-free
   candidate directory, fixed provider catalog, tracked fixture template, and
   private exported manifest; it requires exactly 1,310 candidate and 1,310
   challenge proposal records plus 1,310 final items, validates every
   full-identity/run/digest reference and `ACCEPT_*` server-copy source, uses
   the loader credential, and seals after
   reconciling package/run/taxonomy/fixture/aggregate digests. The runtime
   command credential has EXECUTE-only access to enumerated functions. After a
   successful seal, revoke the loader's login, remove its secret mount and
   one-shot containers, and write text-free target-bound revocation receipts;
   replay, wrong target/input, partial load, or post-seal load fails closed.
3. Add unit/transaction-double tests for append-only question versions and
   events; the complete discriminated reference and parent key; independent
   uniqueness of server-issued root/version/proposal IDs per generation;
   repeated body digests that remain separate roots; duplicate full-reference,
   ID-reuse, missing/cyclic/cross-generation/cross-root parent, and
   non-current-leaf rejection; exact immutable
   snapshot reconstruction; old-generation Admin history; new-generation
   revision-1 base-only reconstruction; stored idempotent response in the same transaction; independent
   operation-ID/idempotency-key uniqueness and cross-pair reuse conflicts, CAS,
   reset reason plus exact `expectedGenerationId`,
   `expectedGenerationRevision`, and `expectedGenerationSealDigest`, including
   every single-field mismatch, every nonterminal lifecycle branch, and
   rejection of loader calls after seal.
4. Re-run focused tests for GREEN, then the full package/static boundary. Do not
   claim live PostgreSQL persistence, live grants, or canonical zero delta here;
   those connected claims belong only to Task 9.

**Commands:**

```bash
go -C apps/api test -count=1 ./internal/preproddata/agademoworkspace/... ./cmd/preprod-aga-demo-workspace-role-provisioner ./cmd/preprod-aga-demo-workspace-fixture-exporter ./cmd/preprod-aga-demo-workspace-loader
node --test tests/aga-candidate-preprod-demo-boundary.test.mjs tests/aga-hybrid-classification-plan-contract.test.mjs tests/aga-hybrid-demo-workspace-boundary.test.mjs
docker compose --project-name aviasurveil360-local-preprod --file deploy/local/compose.yaml --profile local-preprod-loader --profile aga-candidate-demo --profile aga-candidate-demo-oidc-fixture --profile aga-demo-workspace-loader --profile preproddemo config >/dev/null
bash scripts/check-compose-policy.sh
git diff --check
```

**Acceptance:** Unit/static evidence proves the schema, role, one-shot loader,
fixture binding, append-only/CAS/idempotency/reset design is closed and cannot
import canonical packages or authority. Live shared persistence, privilege
probes, and zero-delta evidence remain `not run` until Task 9.

### Task 4 — Add Exact Authorization And The Tagged Workspace API

**Files:**

- Create `apps/api/internal/agademoworkspace/types.go`, `authorization.go`,
  `authorization_test.go`, `service.go`, and `service_test.go`.
- Create `apps/api/internal/httpapi/aga_demo_workspace_api.go` and
  `aga_demo_workspace_api_test.go`.
- Modify `apps/api/cmd/api/profile_preproddemo.go`, `profile_runtime.go`,
  `main.go`, and `profile_preproddemo_test.go`.
- Modify `apps/api/internal/platform/config/config.go` for separate workspace
  read/command URLs.
- Modify `api/openapi/source/paths/platform.json`,
  `api/openapi/source/schemas/platform.json`, and the bundled/generated
  `api/openapi/aviasurveil360.yaml`,
  `apps/api/internal/httpapi/generated/api.gen.go`, and
  `apps/web/src/generated/transport/api-types.ts`.
- Modify `scripts/bundle-openapi.mjs` and create
  `tests/openapi-workspace-operation-contract.test.mjs` for neutral-denial and
  query-versus-command header generation.
- Create `api/openapi/tests/aga-demo-workspace-contract.test.mjs`.

**Route families:** exactly the fixed method/path/operation matrix above. Every
OpenAPI operation declares explicit `x-authorized-roles`; never rely on the
bundler's broad path defaults. The existing contract test must still see
exactly five old GET operations under its old prefix.
Task 4 freezes the lifecycle envelopes/protector and exercises authorization
with explicit no-write domain-service doubles, but the tagged runtime reports
the lifecycle capability unavailable with a specific neutral reason until Task
7's real append-only state machine passes and is wired. Task 4 never returns a
fake lifecycle success or appends a dummy event.

**Focused verification sequence:**

1. Add focused tests named `TestWorkspaceProtectorAuthenticatesCSRFFirst`,
   `TestWorkspaceClassificationAuthorizationMatrix`,
   `TestWorkspaceLifecycleAuthorizationMatrix`, and
   `TestWorkspaceDirectIDDenialIsNeutral`. The direct-ID test covers complete
   Base and Workspace references plus bare/mixed root, version, proposal,
   sequence, and digest guesses in current and reset generations. Add the
   failing Node assertions for
   exact `x-operation-kind`, neutral responses, header/body revision equality,
   required generic `operationId`/`idempotencyKey`/`expectedGenerationId`,
   body/header idempotency equality, lifecycle revision/digest CAS, and reset
   ID/revision/seal CAS. Run the focused Go and OpenAPI tests and record the
   result for missing handler/routes/protector and generated contract/bundler
   behavior.
2. Cover anonymous, Auditee-on-classification, unrelated CAA roles, stale
   session/membership, wrong department/unit/organization, invalid CSRF,
   cross-organization ID guessing, Auditee `GET_ROLE_HISTORY` neutral denial,
   client-issued workspace identity fields, malformed/mixed discriminators,
   current-leaf aliases, old-generation Workspace references on ordinary
   routes, exact Admin history reconstruction,
   missing/mismatched generic command-envelope fields, and invalid bodies.
   Assert no domain-object
   reader before broad authority and no scoped object lookup before exact
   authorization; body parsing may occur only after broad workspace authority.
   Cover idempotency binding lookup before current-generation/domain lookup,
   exact post-reset stored-response replay for a still-current Admin, neutral
   replay denial after revoked/stale membership or workspace binding, changed
   organization or wrong operation-role, and nonmatching/unseen stale-
   generation no-write conflicts.
3. Cover positive Admin read/reset with exact generation ID/revision/seal body
   equality and header-revision equality, every reset CAS mismatch, exact Manager classification/readiness,
   assigned Inspector/Lead, exact reviewer binding, matching Auditee lifecycle
   projections, and every lifecycle revision/digest mismatch without canonical
   assignment writes.
4. Implement the separate handler/protector and four-pool tagged wiring. Keep
   `NewAGACandidateDemoHandler` and its protector unchanged. Generate contracts,
   re-run for GREEN, then run both default and `preproddemo` tagged tests/build.

**Commands:**

```bash
go -C apps/api test -count=1 ./internal/agademoworkspace ./internal/httpapi
go -C apps/api test -count=1 ./cmd/api
go -C apps/api test -count=1 -tags=preproddemo ./cmd/api
build_dir="$(mktemp -d "${TMPDIR:-/tmp}/avia-aga-workspace-build.XXXXXX")"
trap 'rm -rf "$build_dir"' EXIT
go -C apps/api build -tags=preproddemo -o "$build_dir/preprod-aga-demo-api" ./cmd/api
./scripts/generate-contracts.sh
npm --prefix apps/web run contracts:check
node --test tests/openapi-workspace-operation-contract.test.mjs api/openapi/tests/aga-candidate-demo-contract.test.mjs api/openapi/tests/aga-demo-workspace-contract.test.mjs api/openapi/tests/contract-examples.test.mjs
```

**Acceptance:** Classification, lifecycle, and reset surfaces enforce the exact
operation matrix with neutral CSRF-safe denials. Generated contracts are clean,
the tagged artifact is actually compiled/tested, the normal API has no
workspace credential, and the old five Admin-only reads remain unchanged.

### Task 5 — Build The Shared Classification Review UI

**Files:**

- Create `apps/web/src/backend/aga-demo-workspace.ts` and
  `aga-demo-workspace.test.ts`.
- Modify `apps/web/src/backend/backend.ts`,
  `apps/web/src/backend/http-backend.ts`, and
  `apps/web/src/backend/http-backend.test.ts`.
- Create `apps/web/src/auth/aga-demo-workspace-guard.tsx` and
  `aga-demo-workspace-guard.test.tsx`.
- Create `apps/web/src/app/aga-demo-workspace-routes.tsx` and
  `aga-demo-workspace-routes.test.tsx`.
- Create `apps/web/src/features/checklists/aga-classification-workspace-page.tsx`.
- Create `apps/web/src/features/checklists/aga-classification-workspace-page.css`
  and `aga-classification-workspace-page.test.tsx`.
- Create `apps/web/scripts/assert-aga-workspace-artifact-boundary.mjs` and
  `assert-aga-workspace-artifact-boundary.test.mjs`.
- Modify `apps/web/src/app/router.tsx`, `apps/web/src/ui/role-navigation.tsx`,
  and `apps/web/src/features/admin/aga-candidate-demo-panel.tsx` to mount/link
  the capability-gated supplemental route without changing the old raw-package
  presentation or its eager-load behavior.

**Required behavior:** server-paginated/filterable 1,310-row base inventory;
domain/topic, confidence, blocker, source-gap, external-involvement, and form
filters sent only in POST bodies; exact batch preview/execute semantics;
individual retain/reclassify/add-topic/remove-topic/full-proposal-resolution/
include/exclude/defer/add/reword/mark-ready commands; exact reason table; stale
conflict recovery; immutable revision/question-version timeline; separate base
and Draft counts; sealed AI confidence/provenance displayed separately from
nullable Draft-effective confidence/review state; aggregate reconciliation; all 14 connected providers marked
as `INSPECTED_SCOPE_ELIGIBLE` only for `AERODROME_OPERATOR` or
`INVOLVEMENT_ONLY`; explicit simulation/approval/publication separation; and an
Admin-only history/reset control requiring reason, exact
`expectedGenerationId`, `expectedGenerationRevision`, and
`expectedGenerationSealDigest`, explicit destructive confirmation, neutral conflict recovery,
and refresh to the new current generation. Manager and lifecycle-only
projections never receive that control.
`Backend` exposes an optional workspace capability/client only when the server
capability succeeds. `createHttpBackend` implements it through its existing
private request closure so API-base joining, CSRF injection, idempotency/
revision headers, authentication-loss handling, no-store behavior, and disabled
browser telemetry remain single-path; the feature must not create a parallel
raw `fetch` stack.
The artifact scanner recursively inspects the just-built artifact before the
next profile overwrites `dist`: demo mode rejects every supplemental workspace
route/API prefix, workspace client/chunk marker, embedded classification field,
and candidate data; HTTP mode requires the fixed supplemental route/API client
markers but rejects embedded question/classification rows, static capability
success, or a body-bearing source map. Fixture tests prove positive and
negative modes. Runtime capability/neutral authorization remains covered by
route/backend tests rather than inferred from string presence.

**Focused verification sequence:**

1. Add focused tests named `renders authorized supplemental workspace`,
   `executes exact batch preview`, `creates immutable wording successor`,
   `resolves every controlled proposal family`,
   `shows provider eligibility`, `admin resets generation with exact CAS`, and
   `purges sensitive state`. The focused result should identify missing
   modules/routes; do not
   satisfy it with a nonfunctional shell.
   Write the Node artifact-scanner fixture test first and record RED for the
   missing scanner.
2. Cover every visible control, preview expiry/digest/count, required reasons,
   source-gap inclusion, semantic-edit demotion, candidate/challenge/exact
   proposal resolution, subsequent disposition, stale conflict, server-returned
   complete Base/Workspace references for add/reword, rejection of client ID
   synthesis, durable successor reload, base versus Draft counts, and disabled technical
   approval/publication reasons. Reset UI tests send all three exact CAS fields
   and cover ID, revision/header, seal-digest mismatch, and each nonterminal
   lifecycle conflict separately.
3. Cover logout, subject/organization change, denial, BFCache, URL/history/
   referrer, Cache Storage, Web Storage, IndexedDB, service worker, React-query
   memory, and telemetry/log payload absence. Never snapshot accepted question
   content.
4. Implement accessible keyboard/mobile behavior with server pagination and a
   bounded page cache; the new page never calls the old panel's all-question
   loading loop. Re-run focused tests for GREEN, then typecheck/build/artifact
   regressions. The demo artifact must contain no active workspace route/data.

**Commands:**

```bash
npm --prefix apps/web test -- src/backend/aga-demo-workspace.test.ts src/backend/http-backend.test.ts src/auth/aga-demo-workspace-guard.test.tsx src/app/aga-demo-workspace-routes.test.tsx src/features/checklists/aga-classification-workspace-page.test.tsx src/features/admin/aga-candidate-demo-panel.test.tsx
node --test apps/web/scripts/assert-aga-workspace-artifact-boundary.test.mjs
npm --prefix apps/web run typecheck
npm --prefix apps/web run build:demo
node apps/web/scripts/assert-aga-workspace-artifact-boundary.mjs --profile demo --artifact apps/web/dist
npm --prefix apps/web run build:http
node apps/web/scripts/assert-aga-workspace-artifact-boundary.mjs --profile http --artifact apps/web/dist
node apps/web/scripts/assert-http-artifact.mjs apps/web/dist
```

**Acceptance:** UI/backend-double tests prove Admin and scoped Department
Manager render the same Draft contract, reload successor responses, and give
every control an exact command result or a specific disabled reason; candidate
text has no browser persistence or observability residue. Live shared
persistence/durability remains `not run` until Task 9.

### Task 6 — Integrate Deterministic Recommendation Into Planning

**Files:**

- Modify `apps/api/internal/agaapplicability/recommendation.go` and
  `recommendation_test.go`.
- Create `apps/api/internal/agademoworkspace/recommendation.go` and
  `recommendation_test.go`; modify
  `apps/api/internal/preproddata/agademoworkspace/postgres_store.go` and its
  `postgres_store_test.go` for immutable recommendation snapshots.
- Modify `apps/api/internal/httpapi/aga_demo_workspace_api.go` and
  `aga_demo_workspace_api_test.go`.
- Modify `apps/web/src/features/planning/new-audit-wizard.tsx` and
  `new-audit-wizard.test.tsx`.

**Focused verification sequence:**

1. Add focused tests named `TestRecommendationRequiresServerDerivedFacts`,
   `TestRecommendationRejectsKindProfileMismatch`,
   `TestRecommendationRejectsAmbiguousQuestionLeafGraph`,
   `TestRecommendationSnapshotPinsReadiness`, and
   `shows neutral fail-closed AGA recommendation`. The focused result should
   identify the missing complete contract/snapshot UI.
2. Cover every required field, missing/extra qualifier key/value, active
   interval, provider type ID-to-code resolution, target kind/profile
   compatibility, `draftRevision`/`expectedDraftRevision` equality,
   taxonomy/run/Draft/readiness staleness, current included-leaf selection,
   exclusion of every other disposition/version, duplicate/missing leaf,
   broken parent or mixed-generation reference, exact full-reference snapshot
   reconstruction, Base-first reword retaining its accepted package position,
   null-parent added-root ordering, and ambiguous facts.
3. Prove only synthetic `AERODROME_OPERATOR` scope receives the current profile;
   optional involvement never transfers ownership or creates another
   checklist.
4. Implement stable full-reference ordering and an immutable snapshot only on
   success. Base roots and Workspace roots with Base ancestry use accepted
   package order; only null-parent Workspace-added roots use `rootSequence`
   after exact identity resolution; every reword preserves its logical root
   position. A neutral failure writes nothing. The wizard shows exact reasons
   only after authorized workspace capability, while every ordinary demo/HTTP
   planning path remains unchanged. It creates no canonical Planning or Audit
   record. Re-run focused tests for GREEN.

**Commands:**

```bash
go -C apps/api test -count=1 ./internal/agaapplicability ./internal/agademoworkspace ./internal/preproddata/agademoworkspace ./internal/httpapi
npm --prefix apps/web test -- src/features/planning/new-audit-wizard.test.tsx
npm --prefix apps/web run typecheck
npm --prefix apps/web run build:demo
node apps/web/scripts/assert-aga-workspace-artifact-boundary.mjs --profile demo --artifact apps/web/dist
npm --prefix apps/web run build:http
node apps/web/scripts/assert-aga-workspace-artifact-boundary.mjs --profile http --artifact apps/web/dist
node apps/web/scripts/assert-http-artifact.mjs apps/web/dist
```

**Acceptance:** A complete exact synthetic selection returns a reproducible
question set; any missing or ambiguous fact returns none and causes no write.

### Task 7 — Implement The Synthetic Inspection Lifecycle Backend

**Files:**

- Create `apps/api/internal/agademoworkspace/lifecycle_types.go`,
  `lifecycle.go`, `lifecycle_test.go`, `lifecycle_projection.go`, and
  `lifecycle_projection_test.go`.
- Create
  `apps/api/internal/preproddata/agademoworkspace/postgres_lifecycle_store.go`
  and `postgres_lifecycle_store_test.go`.
- Modify `apps/api/internal/httpapi/aga_demo_workspace_api.go` and
  `aga_demo_workspace_api_test.go`.
- Modify `api/openapi/source/paths/platform.json`,
  `api/openapi/source/schemas/platform.json`, bundled/generated contracts, and
  `api/openapi/tests/aga-demo-workspace-contract.test.mjs`.

**Focused verification sequence:**

1. Add focused tests named `TestLifecycleRequiresPotentialFindingConversion`,
   `TestInspectionRequiresExactCurrentRecommendation`,
   `TestFindingInitialStateCoversEveryCAPEvidenceChoice`,
   `TestDueDateChoiceIsIndependentAndExact`,
   `TestReopenAndInspectionCompletionTransitionsAreTotal`,
   `TestCAPAndEvidenceResubmissionTransitionsAreTotal`,
   `TestCAPAcceptanceLeavesFindingOpen`,
   `TestEvidenceReviewOutcomeMappingIsAtomic`,
   `TestEvidenceVerificationAndAuthorizedClosureAreSeparate`, and
   `TestAuditeeProjectionIsOrganizationScoped`. The focused result should
   identify missing state machine/events; do not weaken the assertion to current
   behavior.
2. Cover the exact answer eligibility, required Auditee comment, Lead
   return/correct/successor/resubmit/dismiss-or-convert cycle, human severity/
   CAP/Evidence/due choices, CAP fields
   and review with separate CAP comments/notes, the Evidence-required versus
   no-Evidence `PENDING_CLOSURE` branches, all four conversion initial states/
   next actions, no-CAP and no-Evidence command denial, CAP revision
   submit/reject/more-information/resubmit ordering, Evidence submit/
   more-information/resubmit ordering, metadata-only Evidence versions,
   exact four-outcome Evidence/Finding mapping, due-date true/false across the
   CAP/Evidence branches, zero-root and last-root completion, exact reopen
   source/result states, append-only CAP supersession, `AUTHORIZED_CLOSE`
   denial from every non-`PENDING_CLOSURE` state, required separate Evidence
   comments/notes, CAA-only role-history identity/binding fields, Auditee
   structural omission and neutral role-history denial, and owner/next action.
3. Cover every operation-role/binding, generation and readiness pin,
   recommendation-to-inspection digest and server-copied question snapshot,
   exact Inspector/Lead binding pins, binding revocation/staleness/cross-scope
   rejection, stale recommendation rejection, missing/mismatched generic
   command-envelope fields, exact
   `expectedLifecycleRevision`/`expectedLifecycleDigest` mismatch and header
   equality, idempotency/CAS, append-only
   event, and cross-organization denial. Prove CAP acceptance never closes and
   authorized closure never creates a verification result.
4. Implement the smallest append-only state machine and structurally separate
   CAA/Auditee projections. Regenerate/check contracts and re-run for GREEN.

**Commands:**

```bash
go -C apps/api test -count=1 ./internal/agademoworkspace ./internal/httpapi
go -C apps/api test -count=1 ./internal/preproddata/agademoworkspace
./scripts/generate-contracts.sh
npm --prefix apps/web run contracts:check
node --test api/openapi/tests/aga-demo-workspace-contract.test.mjs
```

**Acceptance:** Unit/static/service-double evidence proves the complete
synthetic state machine and preserves product closure semantics. Canonical
Audit/Finding/CAP/Evidence zero-delta remains `not run` until the connected
Task 9 gate.

### Task 8 — Implement The Synthetic Multi-Role Lifecycle UI

**Files:**

- Create `apps/web/src/features/inspections/aga-demo-inspection-page.tsx` and
  `aga-demo-inspection-page.test.tsx`.
- Create `apps/web/src/features/inspections/aga-demo-lifecycle.css` for the
  responsive lifecycle surface shared by the supplemental role pages.
- Create
  `apps/web/src/features/findings/aga-demo-potential-finding-page.tsx` and
  `aga-demo-potential-finding-page.test.tsx`.
- Create `apps/web/src/features/caps/aga-demo-cap-evidence-page.tsx` and
  `aga-demo-cap-evidence-page.test.tsx`.
- Modify `apps/web/src/app/aga-demo-workspace-routes.tsx` and
  `aga-demo-workspace-routes.test.tsx`.
- Modify `apps/web/playwright.config.ts` to add all three exact specs to the
  `preprod-aga-demo` project's `testMatch`.
- Create `apps/web/tests/e2e/aga-hybrid-classification-workspace.http.spec.ts`,
  `apps/web/tests/e2e/aga-synthetic-lifecycle.http.spec.ts`, and
  `apps/web/tests/e2e/aga-hybrid-privacy.http.spec.ts`.

**Required paths:** Department Manager creates a recommendation and releases a
simulation; Inspector answers and proposes a Potential Finding; Lead returns,
dismisses, or converts; matching Service Provider submits CAP/Evidence; exact
CAA reviewer reviews CAP; Inspector/Lead verifies Evidence; scoped Manager may
use the separate authorized-closure path; Admin inspects history and resets a
generation. Actors use existing synthetic preprod subjects plus workspace-only
bindings; no UI impersonation or unauthorized role switching is introduced.

**Focused verification sequence:**

1. Add focused Vitest paths for every role/transition, comment/note privacy,
   CAA-only role history, Auditee public owner labels with structural omission
   of subject/membership/binding/assignment history and neutral role-history
   denial, CAP-versus-closure semantics, disabled controls, logout/principal/
   BFCache purge, keyboard, and mobile behavior. The focused result should
   identify missing pages/routes.
2. Add the three Playwright files and first run list-only discovery; expected
   RED before config change is zero selected new tests or project mismatch.
3. Implement the pages and supplemental route registration, extend `testMatch`,
   and re-run unit tests and `--list` for GREEN with a recorded nonzero count.
   The preprod project keeps trace, screenshot, and video `off`.
4. Do not run connected Playwright here: that project has no `webServer` and
   requires Task 9's isolated services. Record browser execution `not run` at
   this slice rather than claiming the list command exercised behavior.

**Commands:**

```bash
npm --prefix apps/web test -- src/features/inspections/aga-demo-inspection-page.test.tsx src/features/findings/aga-demo-potential-finding-page.test.tsx src/features/caps/aga-demo-cap-evidence-page.test.tsx src/app/aga-demo-workspace-routes.test.tsx
npm --prefix apps/web run typecheck
npm --prefix apps/web run build:demo
node apps/web/scripts/assert-aga-workspace-artifact-boundary.mjs --profile demo --artifact apps/web/dist
npm --prefix apps/web run build:http
node apps/web/scripts/assert-aga-workspace-artifact-boundary.mjs --profile http --artifact apps/web/dist
node apps/web/scripts/assert-http-artifact.mjs apps/web/dist
npm --prefix apps/web run test:e2e:aga-preprod -- --list aga-hybrid-classification-workspace.http.spec.ts aga-synthetic-lifecycle.http.spec.ts aga-hybrid-privacy.http.spec.ts
```

**Acceptance:** Every visible lifecycle control creates the correct synthetic
artifact or is disabled with a precise governance reason; desktop/mobile,
keyboard, role, privacy, and logout unit behavior pass with no retained media.
The three connected specs have nonzero discovery; actual browser behavior stays
`not run` until Task 9.

### Task 9 — Connected Isolation, Recovery, And Reset Qualification

**Files:**

- Create `scripts/test-aga-hybrid-demo-workspace-connected.sh`.
- Create `scripts/verify-aga-hybrid-demo-workspace-evidence.mjs`.
- Create `scripts/build-aga-hybrid-forbidden-inventory.mjs`.
- Create `scripts/issue-aga-hybrid-connected-authorization.mjs` as a separate
  operator utility; the connected harness must never import or invoke it.
- Extend `tests/aga-hybrid-demo-workspace-boundary.test.mjs` and create
  `tests/fixtures/aga-hybrid-forbidden-object-inventory.v1.json`.
- Create
  `tests/fixtures/aga-hybrid-connected-authorization.schema.json`.
- Create the offline-gate evidence record
  `docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_TASK9_OFFLINE_2026-08-04.md`.
- Create the privacy-safe tracked evidence summary
  `docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_DEMO_LIFECYCLE_2026-08-04-v2.md`
  only after private evidence passes the verifier; never copy private raw logs,
  subjects, memberships, IDs, question text, secrets, or browser artifacts
  into it.

**Focused verification sequence:**

1. Add the boundary test. It requires an inventory class for every
   table and sequence parsed from `apps/api/migrations/*.up.sql`, every new
   workspace object, every exact predecessor-overlay object/sequence from its
   loader contract/seal receipt, every role/grant, all Compose services/secrets, static Go
   import/SQL closure, exact connected-authorization schema, tagged test/build
   commands, E2E nonzero discovery, and no-skip harness markers. Its exact
   subtests include `authorization envelopes are closed and atomically
   consumed`, `forbidden baseline predates workspace provisioning`, `loader
   barriers precede credential revocation`, `phase journal recovers every
   commit publication boundary`, and `fault matrix manifest has the exact four
   connected cases`. The matrix subtest also kills before/after every outer
   case-intent/target/receipt/aggregate boundary, covers status/resume/partial-
   cleanup, and proves the concurrency case token was not pre-reserved. Add the
   named subtest `evidence finalizer is privacy-safe
   and check-only`; with bounded synthetic happy/fault ledgers it rejects a
   missing/extra phase or case, broken hash chain/digest, nonzero residue,
   sensitive path/subject/membership/ID/question token, one-ledger substitution,
   pre-existing summary on finalize, or changed summary on check, and proves
   `--check-summary` leaves summary bytes, size, inode, and mtime unchanged.
   Expected RED is
   missing inventory/script/services/contracts and recovery/evidence protocol,
   never a skipped or weakened assertion.
2. Implement the inventory generator. Any newly discovered object absent from
   the closed `FORBIDDEN_BUSINESS`, `AUTH_CONTROL_PLANE`,
   `SEALED_OVERLAY_ALLOWED`, or `WORKSPACE_ALLOWED` class fails; no wildcard
   family silently passes. `SEALED_OVERLAY_ALLOWED` contains only the exact
   predecessor schema objects/sequences and grants reconstructed by its
   unchanged loader: creation/load is allowed only before `OVERLAY_SEALED`, and
   afterward every row/sequence/grant/seal digest has zero delta and no
   workspace role access. It is never merged into `WORKSPACE_ALLOWED`. The
   forbidden class includes canonical Provider/scope/target, identity
   references/profiles/membership versions and requests, department/functional
   assignments, regulatory/source/decision/publication, Planning/inspection/
   checklist/response, Potential Finding/Finding/CAP/Evidence/report,
   notification/outbox/delivery, business idempotency/audit events, and every
   sequence. `AUTH_CONTROL_PLANE` names exact objects and columns, including
   session rows and `desired_membership_sync` observation fields, with allowed
   insert/update/event shapes per authentication request and logout; role,
   organization, membership version, and non-observation changes remain
   forbidden.
   The existing full synthetic smoke/OIDC predecessor setup necessarily writes
   its own disposable canonical fixture rows before this baseline. Record those
   exact predecessor receipts/counts as `PREDECESSOR_SETUP`, never as workspace
   zero-write evidence. From the post-fixture baseline through workspace load,
   API commands, lifecycle, and reset, every forbidden object/sequence is
   immutable; only whole-namespace cleanup later removes the disposable target.
3. Every prepare, qualification, recovery, and cleanup authorization document
   is a closed instance of the tracked common envelope. It contains exact
   `schemaVersion`, `authorizationId`, `issuer`, at least 128-bit CSPRNG
   `nonce`, `issuedAt`,
   `expiresAt` no more than 15 minutes after issuance, fixed-input/code-contract
   digests, and the phase-conditional target/intent fields. Only the prepare
   envelope uses `targetMode=CREATE_FRESH_DISPOSABLE` with no fingerprint;
   every later envelope requires the exact target fingerprint plus recovery/
   qualification/bundle digests applicable to that phase. A matrix-outer
   recovery envelope instead binds the outer prepare intent and the exact
   ordered set of case target fingerprints known at interruption, which may be
   empty only before the first target receipt,
   and a closed operation-to-one-shot-token map. All equality is byte/digest
   equality; no prefix or wildcard operation exists. Consumption is an atomic
   create-if-absent ledger transaction keyed separately by authorization ID,
   nonce, and token hash, followed by file and parent-directory `fsync`.
   Concurrent consumers have one winner. At `prepare`, `qualify`,
   `recover-prepare`, `recover-qualify`, or `cleanup-prepared` branch entry, all
   tokens in that document are atomically reserved while the envelope is
   unexpired and bound to that exact journal. A reservation remains usable only
   for the original journal's closed operation sequence after wall-clock
   expiry; it cannot start another branch or target. Thus a long connected run
   retains its reserved cleanup capability, while any new recovery or cleanup
   branch still requires a fresh unexpired document. Before each side effect the harness
   fsyncs an intent record; database effects store the authorization/token hash
   and deterministic result receipt in the same target transaction; afterward
   the harness fsyncs the receipt to the external ledger. On a crash between
   target commit and ledger publication, recovery reads and republishes the
   stored receipt rather than repeating the effect. Missing, expired, reused,
   concurrently consumed, drifted, or unpublishable authority stops before the
   associated effect or branch reservation.
   That same-transaction receipt rule applies to newly owned workspace effects.
   Inherited `CREATE_BASE` and
   `LOAD_AGA_CANDIDATE_DEMO_OVERLAY` code/transactions remain unchanged: the
   hybrid phase intent authorizes invoking them, binds their existing target-
   scoped authorization/control-store/result receipts into its journal, and
   recovers by exact target fingerprint, seal, input digest, and predecessor-
   receipt reconciliation. It never adds a hybrid token column or write to the
   accepted predecessor schema/loader.
4. `prepare` first validates a caller-issued 0600
   `aga-hybrid-demo-connected-prepare-authorization/v1` and requires operation
   `PREPARE_FRESH_AGA_HYBRID_DEMO_TARGET`,
   `targetMode=CREATE_FRESH_DISPOSABLE`, and exactly the closed suboperations
   `CREATE_BASE`, `QUALIFY_EXISTING_SYNTHETIC_OIDC`,
   `PIN_PRE_WORKSPACE_FORBIDDEN_BASELINE`,
   `PROVISION_EMPTY_WORKSPACE_CONTRACT`, `EXPORT_WORKSPACE_FIXTURE`, and
   `PREPARE_LOAD_INTENTS`, plus bounded `CLEANUP_ON_PREPARE_FAILURE`.
   `prepare` requires no existing Compose-project residue and an empty caller-
   owned 0700 private root. It creates a 0600 append-only, hash-chained,
   file-and-directory-fsynced private phase journal mirrored by digest/status
   receipts in the external 0700 ledger. Immediately after the fresh target
   fingerprint exists it seals `recovery-intent.json`; that intent permits a
   later fresh target-bound recovery or cleanup authorization but is not
   authority by itself. A kill between inherited target commit and that seal is
   recoverable from the fsynced pre-create ledger intent plus the predecessor's
   target/control receipt; recovery may seal the same deterministic recovery
   intent but may not create a second target.

   Invoke the existing predecessor setup as
   `AVIA_PREPROD_RETAIN_SUCCESSFUL_BASE_HANDOFF_DIR=<private>/base-handoff bash scripts/test-preprod-connected-scenarios.sh smoke`.
   Validate its 0600 base result/handoff, state directory, control store, run,
   and target fingerprint; then qualify the already-created nine synthetic
   OIDC accounts. Immediately after base plus OIDC qualification—and before
   any workspace provisioner or exporter—snapshot and fsync the exact
   `FORBIDDEN_BUSINESS` table/content/sequence baseline. Provision the empty
   workspace schema/role/function contract, compare the baseline unchanged,
   export through the dedicated read-only role the exact private subject/
   membership/organization fixture manifest, and compare it unchanged again.
   Build but do not execute the current overlay/workspace load intents. Seal
   `qualification-intent.json` with the pre-provision baseline digest, both
   post-step comparisons, target/recovery intent, role-provision/OIDC/fixture
   receipts, manifest digest, and both load-intent digests; then report literal
   `pending external authority`. Cleaned run 47 is never an input.

   An operator may issue a separate 0600
   `aga-hybrid-demo-connected-qualification-bundle/v1`, under the same common
   envelope and bound to the exact target/recovery/qualification intents, with
   distinct one-shot tokens for `LOAD_AGA_CANDIDATE_DEMO_OVERLAY`,
   `RUN_WORKSPACE_LOAD_SEAL_BARRIERS`,
   `LOAD_AND_SEAL_AGA_DEMO_WORKSPACE`,
   `RUN_CONNECTED_AGA_HYBRID_QUALIFICATION`,
   `CLEANUP_AGA_CANDIDATE_DEMO_OVERLAY`, and
   `CLEANUP_WHOLE_DISPOSABLE_NAMESPACE`. The harness never creates or derives
   a token. For this authorized local task the primary writer acts separately
   as the operator and invokes
   `scripts/issue-aga-hybrid-connected-authorization.mjs` outside the harness.
   That utility accepts one explicit closed envelope template plus the exact
   target/intent receipt inputs, uses `crypto.randomBytes` for each nonce and
   one-shot token, writes one create-only 0600 document under a caller-owned
   0700 private root, prints only a controlled receipt digest, and cannot be
   imported by or share a process with the harness. Its unit tests prove
   create-only behavior, at least 128 bits of entropy, exact target binding,
   expiry bounds, closed operation sets, safe diagnostics, and absence from
   every production/tagged artifact. If qualification is declined or any retained phase must be
   abandoned, an operator instead issues a fresh 0600
   `aga-hybrid-demo-connected-cleanup-authorization/v1` under that envelope,
   bound to the exact recovery intent, target, qualification intent when it
   exists, and operation `CLEANUP_WHOLE_DISPOSABLE_NAMESPACE`.
   `cleanup-prepared` may consume it at any retained phase, appends cleanup/
   replay/zero-residue receipts, and never needs load or run authority.

   Ordinary TERM/ERR traps use `CLEANUP_ON_PREPARE_FAILURE` only before a
   complete qualification intent is sealed. SIGKILL/host loss is recovered
   from the journal: `recover-status` reports only text-free phase/receipt
   digests, and a fresh
   `aga-hybrid-demo-connected-recovery-authorization/v1` may authorize exactly
   `RESUME_PREPARE`, `RESUME_QUALIFICATION`, or
   `CLEANUP_WHOLE_DISPOSABLE_NAMESPACE` for the original recovery intent and,
   when qualification started, the consumed bundle digest. Resume may execute
   only the original closed not-yet-completed operations; a completed operation
   returns its stored receipt. It cannot broaden or restart the run.
5. `qualify` validates and atomically consumes the bundle against the intents,
   journal, ledger, and still-identical target. Before each effect and after
   each receipt it rechecks the prepare-time forbidden baseline; a mismatch
   aborts to authorized cleanup. It hash-checks/loads the exact predecessor ZIP
   and requires a newly sealed `preprod_aga_demo`. Before workspace loading,
   the exporter runs read-only `verify` and requires the current nine-account
   membership/organization matrix and private manifest digest to equal the
   qualification intent. While the loader credentials still exist and before
   the real workspace seal, run `load-versus-seal` barriers against two
   independently named task-owned throwaway sibling schemas using the exact
   schema/functions: one forces load-then-seal and one seal-then-rejected-load;
   each proves one deterministic winner, loser zero delta, complete rollback
   or usable seal. Drop each whole sibling schema under the barrier authority,
   prove zero sibling-role/schema residue and the real append-only workspace
   schema/role contract untouched, then load the real workspace. No probe
   generation is inserted into or selectively deleted from the real workspace.
   This connected barrier evidence is recorded at this pre-revocation phase
   and is not deferred to runtime.

   Load the candidate/catalog/template/private fixture through the isolated
   loader, independently reconcile overlay and workspace identity/digests via
   separate read pools, seal, revoke loader/exporter login, unmount/delete both
   secrets, remove all one-shot containers, and require target-bound seal and
   privilege-removal receipts before API startup. Recompare the original
   pre-provision forbidden baseline after provision, export, each overlay/
   workspace load and seal, credential revocation, and final pre-API state.
   Cleanup later destroys the target under its own receipt; no impossible
   post-destruction database comparison is claimed.
6. Start the tagged API and ordinary HTTP artifact and take pre-authentication
   forbidden/auth-control snapshots. Authenticate each of the nine predecessor
   subjects separately; around every login/fresh-provider observation require
   zero forbidden delta and the exact allowed auth-control row/column/event/
   sequence delta. Then establish post-login baselines for workspace requests.
   Run Admin, exact Manager, Inspector, Lead/reviewer, matching and other-
   organization Auditee, anonymous, unrelated-role, CSRF, direct-ID, logout-
   state, and BFCache matrices at the required desktop/mobile viewports with
   trace/screenshot/video off. Around every authenticated POST, require exact
   zero `FORBIDDEN_BUSINESS` content/sequence delta, zero post-seal
   `SEALED_OVERLAY_ALLOWED` row/sequence/grant/seal delta, and only the
   enumerated `AUTH_CONTROL_PLANE` column/event delta.
7. Complete one full Potential Finding-to-closure lifecycle and independently
   reconstruct every generation/run/Draft/readiness/recommendation/event/body
   digest. Verify `commentToAuditee`/`internalCaaNote` separation and zero
   question/body/ID/filter leakage in URL/history/referrer, browser stores,
   response caches, telemetry/logs, and artifacts.
8. Use barrier-controlled tests for same-pair/same-payload replay, same-pair/
   different-payload conflict, operation-ID reuse with a new key, key reuse
   with a new operation ID, concurrent cross-pair reuse, parallel Draft CAS,
   parallel lifecycle transitions with exact lifecycle revision/digest CAS,
   reset-versus-command,
   reset replay, and response loss after commit. A reset requires reason,
   `expectedGenerationId`, `expectedGenerationRevision`, and
   `expectedGenerationSealDigest`; it atomically bootstraps the referenced sealed run and
   revision-1 Draft, never rolls back/reactivates the old one, and remains
   usable for a new Manager decision/recommendation after reset. Assert failed-
   reset rollback for every nonterminal Inspection/Finding/CAP/Evidence state,
   successful-reset replay, old-ID neutral denial/Admin-only
   history, replay denial after membership/binding revocation, organization or
   operation-role change, unseen/mismatched old-generation no-write conflict,
   and no loader credential or row copy.
9. The journal state machine is exact and monotonic:
   `TARGET_CREATED`, `OIDC_QUALIFIED`, `FORBIDDEN_BASELINE_PINNED`,
   `WORKSPACE_CONTRACT_PROVISIONED`, `FIXTURE_EXPORTED`, `INTENTS_SEALED`,
   `OVERLAY_SEALED`, `LOAD_SEAL_BARRIERS_COMPLETE`, `WORKSPACE_SEALED`,
   `CREDENTIALS_REVOKED`, `API_STARTED`, `AUTH_VERIFIED`, `E2E_COMPLETE`, and
   `CLEANED`. Task 9 has three internal stop gates: F1 runs exhaustive
   deterministic unit/integration fault injection before/after every target
   effect and before/after every ledger publication; F2 runs and reviews the
   single happy-path connected qualification; only then may F3 run the bounded
   representative connected recovery matrix. F1 derives every applicable
   phase crossed with `BEFORE_EFFECT`,
   `AFTER_EFFECT_BEFORE_TARGET_RECEIPT`,
   `AFTER_TARGET_RECEIPT_BEFORE_LEDGER_PUBLICATION`, and
   `AFTER_LEDGER_PUBLICATION` and rejects missing/skipped cases without
   rebuilding a full connected target per permutation.

   F3 is itself two-phase because target-bound documents cannot exist before
   their targets. `fault-matrix-prepare` consumes a caller-owned 0700 root with
   a closed `fault-matrix-prepare-manifest.json` containing exactly four case
   IDs—`INHERITED_BASE_RECEIPT_GAP`,
   `WORKSPACE_TRANSACTION_RECEIPT_GAP`, `CONCURRENT_TOKEN_RESERVATION`, and
   `CLEANUP_RECEIPT_GAP`—plus distinct prepare documents, Compose projects, and
   empty private roots. It reserves all prepare documents while fresh, creates
   a separate outer 0600 hash-chained/file-and-directory-fsynced matrix journal
   before the first case, then creates and prepares the four targets. Before
   and after each case intent, target effect, per-case receipt, and aggregate
   publication it fsyncs the outer journal/ledger. The aggregate target/intent
   digest is deterministically reconstructible from the four per-case
   predecessor/journal receipts, so a publication crash never requires another
   target. It writes that aggregate digest,
   and stops `pending external authority`. It rejects a missing, extra,
   duplicate, or shared target/project/private-root.

   `fault-matrix-recover-status` reports only the outer/per-case phase and
   receipt digests. After any prepare interruption, the operator issues a fresh
   `fault-matrix-recovery-manifest.json` bound to every case whose target receipt
   exists; `fault-matrix-recover-prepare` resumes exact incomplete per-case
   prepare phases and republishes the deterministic aggregate, while
   `fault-matrix-cleanup-partial` consumes fresh target-bound cleanup documents
   for exactly the created targets and proves zero residue. Neither mode creates
   an additional target for an existing case.

   The operator then issues per-case target/intent-bound qualification,
   recovery, and cleanup documents and a closed
   `fault-matrix-run-manifest.json` bound to that aggregate digest. The run
   manifest has a distinct outer matrix authorization plus per-case operation
   documents. `fault-matrix-run` reserves the outer authorization and all
   non-concurrency operation/cleanup tokens while unexpired at branch start.
   It executes `CONCURRENT_TOKEN_RESERVATION` first: two barrier-released case
   consumers race to reserve its separate still-unconsumed shared case token,
   yielding exactly one winner and one no-effect loser; the outer runner never
   pre-reserves that shared token. It then executes the other three cases from
   their reserved tokens. If the operator declines,
   `fault-matrix-cleanup-prepared` instead consumes all four fresh cleanup
   documents and disposes the prepared targets. After a run interruption,
   `fault-matrix-recover-run` requires a fresh outer recovery manifest, returns
   stored receipts for completed effects/concurrency, and resumes only the
   first incomplete case; `fault-matrix-cleanup-partial` remains available with
   fresh exact cleanup documents. Across the four run cases the
   matrix proves inherited-receipt reconciliation, new-workspace stored-receipt
   publication, one atomic winner/no-effect loser, resume after cleanup target
   disposal but before ledger publication, and final zero residue. It never
   uses the happy-path target. The harness never generates fault-matrix
   authority; inability to complete both F3 phases leaves F3 `blocked`, never
   silently skipped or inferred from F1/F2.
10. Run logout as a separate auth-control phase, recheck the exact allowed
   control-plane delta and zero forbidden delta, exercise overlay cleanup/
   authorization replay rejection, and dispose the whole namespace. Prove the
   caller-owned private root is empty plus zero task-owned container,
   volume, network, browser, Vite, Go-test, temporary-build, secret, and
   database residue.
11. Only after F2 and all four F3 cases pass, run `finalize-evidence`. It
   privacy-scans both immutable ledgers/private receipt sets, requires their
   independent zero-residue terminals, and creates the tracked summary once
   with only aggregate ledger/receipt hashes and literal results. It never
   copies a path, subject, membership, ID, question text/digest, secret, or raw
   log. Task 10 checks that finalized summary offline and must not replay a
   connected run. The independent synthetic-ledger boundary test above must be
   GREEN before this writer/checker is trusted with connected receipts.

**Commands:**

Run `prepare`, stop for the target-bound authority bundle, and then run exactly
one terminal branch: `qualify` for Task 9 acceptance or `cleanup-prepared` to
abort. `/absolute/private/aga-hybrid-run-0700` below is the same directory while
empty at prepare entry and populated afterward; the earlier
"empty"/"prepared" labels do not denote different roots. Recovery commands are
used only after an interrupted corresponding branch and continue that same
journal; they are not additional happy-path branches.

```bash
node scripts/build-aga-hybrid-forbidden-inventory.mjs --check tests/fixtures/aga-hybrid-forbidden-object-inventory.v1.json
node --test tests/aga-hybrid-demo-workspace-boundary.test.mjs tests/aga-question-classification-candidate.test.mjs
AVIA_AGA_HYBRID_PRIVATE_ROOT=/absolute/private/aga-hybrid-run-0700 \
AVIA_AGA_HYBRID_PREPARE_AUTHORIZATION_FILE=/absolute/private/prepare-authorization-0600.json \
AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR=/absolute/private/authorization-ledger-0700 \
AVIA_AGA_DEMO_PACKAGE_FILE="$PWD/deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip" \
AVIA_AGA_CLASSIFICATION_CANDIDATE_DIR="$PWD/deliverables/aga-question-classification-candidate-2026-08-03" \
AVIA_AGA_PROVIDER_CATALOG_FILE="$PWD/docs/regulatory-sources/catalogs/service-provider-catalog.v1.json" \
  bash scripts/test-aga-hybrid-demo-workspace-connected.sh prepare
```

After a process/host interruption, inspect only text-free state:

```bash
AVIA_AGA_HYBRID_PRIVATE_ROOT=/absolute/private/aga-hybrid-run-0700 \
AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR=/absolute/private/authorization-ledger-0700 \
  bash scripts/test-aga-hybrid-demo-workspace-connected.sh recover-status
```

If `prepare` was interrupted, a fresh recovery document bound to
`recovery-intent.json` resumes it to literal `pending external authority`:

```bash
AVIA_AGA_HYBRID_PRIVATE_ROOT=/absolute/private/aga-hybrid-run-0700 \
AVIA_AGA_HYBRID_PREPARE_AUTHORIZATION_FILE=/absolute/private/prepare-authorization-0600.json \
AVIA_AGA_HYBRID_RECOVERY_AUTHORIZATION_FILE=/absolute/private/recover-prepare-authorization-0600.json \
AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR=/absolute/private/authorization-ledger-0700 \
AVIA_AGA_DEMO_PACKAGE_FILE="$PWD/deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip" \
AVIA_AGA_CLASSIFICATION_CANDIDATE_DIR="$PWD/deliverables/aga-question-classification-candidate-2026-08-03" \
AVIA_AGA_PROVIDER_CATALOG_FILE="$PWD/docs/regulatory-sources/catalogs/service-provider-catalog.v1.json" \
  bash scripts/test-aga-hybrid-demo-workspace-connected.sh recover-prepare
```

After an operator issues the bundle from `qualification-intent.json`, the
qualification branch is:

```bash
AVIA_AGA_HYBRID_PRIVATE_ROOT=/absolute/private/aga-hybrid-run-0700 \
AVIA_AGA_HYBRID_QUALIFICATION_AUTHORIZATION_BUNDLE_FILE=/absolute/private/qualification-bundle-0600.json \
AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR=/absolute/private/authorization-ledger-0700 \
AVIA_AGA_DEMO_PACKAGE_FILE="$PWD/deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip" \
AVIA_AGA_CLASSIFICATION_CANDIDATE_DIR="$PWD/deliverables/aga-question-classification-candidate-2026-08-03" \
AVIA_AGA_PROVIDER_CATALOG_FILE="$PWD/docs/regulatory-sources/catalogs/service-provider-catalog.v1.json" \
  bash scripts/test-aga-hybrid-demo-workspace-connected.sh qualify
git diff --check
```

If `qualify` was interrupted, a fresh recovery document plus the original
bundle resumes its exact journal to the same verified `CLEANED` terminal result
without repeating a completed effect:

```bash
AVIA_AGA_HYBRID_PRIVATE_ROOT=/absolute/private/aga-hybrid-run-0700 \
AVIA_AGA_HYBRID_QUALIFICATION_AUTHORIZATION_BUNDLE_FILE=/absolute/private/qualification-bundle-0600.json \
AVIA_AGA_HYBRID_RECOVERY_AUTHORIZATION_FILE=/absolute/private/recover-qualify-authorization-0600.json \
AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR=/absolute/private/authorization-ledger-0700 \
AVIA_AGA_DEMO_PACKAGE_FILE="$PWD/deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip" \
AVIA_AGA_CLASSIFICATION_CANDIDATE_DIR="$PWD/deliverables/aga-question-classification-candidate-2026-08-03" \
AVIA_AGA_PROVIDER_CATALOG_FILE="$PWD/docs/regulatory-sources/catalogs/service-provider-catalog.v1.json" \
  bash scripts/test-aga-hybrid-demo-workspace-connected.sh recover-qualify
```

If qualification is declined, run this mutually exclusive recovery branch:

```bash
AVIA_AGA_HYBRID_PRIVATE_ROOT=/absolute/private/aga-hybrid-run-0700 \
AVIA_AGA_HYBRID_CLEANUP_AUTHORIZATION_FILE=/absolute/private/cleanup-authorization-0600.json \
AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR=/absolute/private/authorization-ledger-0700 \
  bash scripts/test-aga-hybrid-demo-workspace-connected.sh cleanup-prepared
git diff --check
```

The independently authorized fault matrix uses only its manifest-owned fresh
targets and roots; it is separate from the happy-path/declined target above.
First prepare the four cases:

```bash
AVIA_AGA_HYBRID_PRIVATE_ROOT=/absolute/private/fault-authorizations-0700 \
AVIA_AGA_HYBRID_FAULT_PREPARE_MANIFEST=/absolute/private/fault-authorizations-0700/fault-matrix-prepare-manifest.json \
AVIA_AGA_HYBRID_FAULT_PREPARE_AUTHORIZATION_FILE=/absolute/private/fault-authorizations-0700/fault-matrix-prepare-authorization.json \
AVIA_AGA_HYBRID_FAULT_AUTHORIZATION_ROOT=/absolute/private/fault-authorizations-0700 \
AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR=/absolute/private/fault-ledger-0700 \
AVIA_AGA_DEMO_PACKAGE_FILE="$PWD/deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip" \
AVIA_AGA_CLASSIFICATION_CANDIDATE_DIR="$PWD/deliverables/aga-question-classification-candidate-2026-08-03" \
AVIA_AGA_PROVIDER_CATALOG_FILE="$PWD/docs/regulatory-sources/catalogs/service-provider-catalog.v1.json" \
  bash scripts/test-aga-hybrid-demo-workspace-connected.sh fault-matrix-prepare
```

After an outer/per-case interruption, inspect the text-free matrix journal:

```bash
AVIA_AGA_HYBRID_PRIVATE_ROOT=/absolute/private/fault-authorizations-0700 \
AVIA_AGA_HYBRID_FAULT_AUTHORIZATION_ROOT=/absolute/private/fault-authorizations-0700 \
AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR=/absolute/private/fault-ledger-0700 \
  bash scripts/test-aga-hybrid-demo-workspace-connected.sh fault-matrix-recover-status
```

Resume an interrupted matrix prepare with a fresh target-bound recovery
manifest:

```bash
AVIA_AGA_HYBRID_PRIVATE_ROOT=/absolute/private/fault-authorizations-0700 \
AVIA_AGA_HYBRID_FAULT_PREPARE_MANIFEST=/absolute/private/fault-authorizations-0700/fault-matrix-prepare-manifest.json \
AVIA_AGA_HYBRID_FAULT_RECOVERY_MANIFEST=/absolute/private/fault-authorizations-0700/fault-matrix-recovery-manifest.json \
AVIA_AGA_HYBRID_FAULT_RECOVERY_AUTHORIZATION_FILE=/absolute/private/fault-authorizations-0700/fault-matrix-recovery-authorization.json \
AVIA_AGA_HYBRID_FAULT_AUTHORIZATION_ROOT=/absolute/private/fault-authorizations-0700 \
AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR=/absolute/private/fault-ledger-0700 \
AVIA_AGA_DEMO_PACKAGE_FILE="$PWD/deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip" \
AVIA_AGA_CLASSIFICATION_CANDIDATE_DIR="$PWD/deliverables/aga-question-classification-candidate-2026-08-03" \
AVIA_AGA_PROVIDER_CATALOG_FILE="$PWD/docs/regulatory-sources/catalogs/service-provider-catalog.v1.json" \
  bash scripts/test-aga-hybrid-demo-workspace-connected.sh fault-matrix-recover-prepare
```

After the operator issues the target-bound run manifest, execute the matrix:

```bash
AVIA_AGA_HYBRID_PRIVATE_ROOT=/absolute/private/fault-authorizations-0700 \
AVIA_AGA_HYBRID_FAULT_RUN_MANIFEST=/absolute/private/fault-authorizations-0700/fault-matrix-run-manifest.json \
AVIA_AGA_HYBRID_FAULT_RUN_AUTHORIZATION_FILE=/absolute/private/fault-authorizations-0700/fault-matrix-run-authorization.json \
AVIA_AGA_HYBRID_FAULT_AUTHORIZATION_ROOT=/absolute/private/fault-authorizations-0700 \
AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR=/absolute/private/fault-ledger-0700 \
AVIA_AGA_DEMO_PACKAGE_FILE="$PWD/deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip" \
AVIA_AGA_CLASSIFICATION_CANDIDATE_DIR="$PWD/deliverables/aga-question-classification-candidate-2026-08-03" \
AVIA_AGA_PROVIDER_CATALOG_FILE="$PWD/docs/regulatory-sources/catalogs/service-provider-catalog.v1.json" \
  bash scripts/test-aga-hybrid-demo-workspace-connected.sh fault-matrix-run
```

Resume an interrupted matrix run without repeating a completed case:

```bash
AVIA_AGA_HYBRID_PRIVATE_ROOT=/absolute/private/fault-authorizations-0700 \
AVIA_AGA_HYBRID_FAULT_RUN_MANIFEST=/absolute/private/fault-authorizations-0700/fault-matrix-run-manifest.json \
AVIA_AGA_HYBRID_FAULT_RECOVERY_MANIFEST=/absolute/private/fault-authorizations-0700/fault-matrix-recovery-manifest.json \
AVIA_AGA_HYBRID_FAULT_RUN_AUTHORIZATION_FILE=/absolute/private/fault-authorizations-0700/fault-matrix-run-authorization.json \
AVIA_AGA_HYBRID_FAULT_RECOVERY_AUTHORIZATION_FILE=/absolute/private/fault-authorizations-0700/fault-matrix-recovery-authorization.json \
AVIA_AGA_HYBRID_FAULT_AUTHORIZATION_ROOT=/absolute/private/fault-authorizations-0700 \
AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR=/absolute/private/fault-ledger-0700 \
AVIA_AGA_DEMO_PACKAGE_FILE="$PWD/deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip" \
AVIA_AGA_CLASSIFICATION_CANDIDATE_DIR="$PWD/deliverables/aga-question-classification-candidate-2026-08-03" \
AVIA_AGA_PROVIDER_CATALOG_FILE="$PWD/docs/regulatory-sources/catalogs/service-provider-catalog.v1.json" \
  bash scripts/test-aga-hybrid-demo-workspace-connected.sh fault-matrix-recover-run
```

If F3 run authority is declined, the same run manifest must instead contain
only four target-bound cleanup documents and is consumed by the mutually
exclusive `fault-matrix-cleanup-prepared` mode:

```bash
AVIA_AGA_HYBRID_PRIVATE_ROOT=/absolute/private/fault-authorizations-0700 \
AVIA_AGA_HYBRID_FAULT_RUN_MANIFEST=/absolute/private/fault-authorizations-0700/fault-matrix-run-manifest.json \
AVIA_AGA_HYBRID_FAULT_CLEANUP_AUTHORIZATION_FILE=/absolute/private/fault-authorizations-0700/fault-matrix-cleanup-authorization.json \
AVIA_AGA_HYBRID_FAULT_AUTHORIZATION_ROOT=/absolute/private/fault-authorizations-0700 \
AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR=/absolute/private/fault-ledger-0700 \
  bash scripts/test-aga-hybrid-demo-workspace-connected.sh fault-matrix-cleanup-prepared
```

For an interrupted partial prepare/run, a newly issued recovery manifest whose
case entries contain only exact cleanup documents uses:

```bash
AVIA_AGA_HYBRID_PRIVATE_ROOT=/absolute/private/fault-authorizations-0700 \
AVIA_AGA_HYBRID_FAULT_RECOVERY_MANIFEST=/absolute/private/fault-authorizations-0700/fault-matrix-recovery-manifest.json \
AVIA_AGA_HYBRID_FAULT_CLEANUP_AUTHORIZATION_FILE=/absolute/private/fault-authorizations-0700/fault-matrix-cleanup-authorization.json \
AVIA_AGA_HYBRID_FAULT_AUTHORIZATION_ROOT=/absolute/private/fault-authorizations-0700 \
AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR=/absolute/private/fault-ledger-0700 \
  bash scripts/test-aga-hybrid-demo-workspace-connected.sh fault-matrix-cleanup-partial
```

The run succeeds only after
every case reaches its expected resumed or
cleaned terminal state, every per-case token replay/concurrent-consumption
assertion passes, and every case-owned target/project/private root has zero
residue.

After F2 and F3 evidence both exist, finalize the privacy-safe tracked summary:

```bash
AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR=/absolute/private/authorization-ledger-0700 \
AVIA_AGA_HYBRID_FAULT_AUTHORIZATION_LEDGER_DIR=/absolute/private/fault-ledger-0700 \
  node scripts/verify-aga-hybrid-demo-workspace-evidence.mjs --finalize-summary docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_DEMO_LIFECYCLE_2026-08-04-v2.md
```

**Acceptance:** Connected shared behavior and recovery pass with zero forbidden
business/content/sequence delta, only exact per-authentication/logout control-
plane deltas, zero original-overlay mutation, sealed fixture/load and privilege-
revocation receipts, complete crash/publication and concurrent-consumption fault
matrix receipts, zero privacy leakage, and zero task-owned residue.

### Task 10 — Aggregate Verification And Handoff

**Files:**

- Update this ExecPlan and only its existing row in
  `docs/exec-plans/index.md`.
- Verify, but do not update, the frozen Gate 0 product spec/taxonomy/schema/
  discovery and classification prompts or their pinned hashes. A defect stops
  Task 10 `blocked` and returns to the owning slice under new authorization; a
  taxonomy/code change requires a successor version/run and rerunning every
  downstream seal/evidence gate.
- Verify, but do not update, Task 9's create-once finalized
  `docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_DEMO_LIFECYCLE_2026-08-04-v2.md`.
- Modify `docs/exec-plans/tech-debt-tracker.md` only if a durable unresolved
  fact remains; do not create an empty tracker entry.

**Commands:**

```bash
go -C apps/api test -count=1 ./internal/agaapplicability ./internal/agademoworkspace ./internal/preproddata/agademoworkspace ./internal/httpapi ./cmd/aga-question-classification-validator ./cmd/preprod-aga-demo-workspace-role-provisioner ./cmd/preprod-aga-demo-workspace-fixture-exporter ./cmd/preprod-aga-demo-workspace-loader ./cmd/api
go -C apps/api test -count=1 -tags=preproddemo ./cmd/api
build_dir="$(mktemp -d "${TMPDIR:-/tmp}/avia-aga-workspace-final-build.XXXXXX")"
trap 'rm -rf "$build_dir"' EXIT
go -C apps/api build -tags=preproddemo -o "$build_dir/preprod-aga-demo-api" ./cmd/api
./scripts/generate-contracts.sh
npm --prefix apps/web run contracts:check
npm --prefix apps/web run typecheck
npm --prefix apps/web test
npm --prefix apps/web run build:demo
node apps/web/scripts/assert-aga-workspace-artifact-boundary.mjs --profile demo --artifact apps/web/dist
npm --prefix apps/web run build:http
node apps/web/scripts/assert-aga-workspace-artifact-boundary.mjs --profile http --artifact apps/web/dist
node apps/web/scripts/assert-http-artifact.mjs apps/web/dist
node --test tests/openapi-workspace-operation-contract.test.mjs api/openapi/tests/aga-candidate-demo-contract.test.mjs api/openapi/tests/aga-demo-workspace-contract.test.mjs api/openapi/tests/contract-examples.test.mjs tests/aga-candidate-preprod-demo-boundary.test.mjs tests/aga-hybrid-classification-plan-contract.test.mjs tests/aga-question-classification-candidate.test.mjs tests/aga-hybrid-demo-workspace-boundary.test.mjs tests/demo-boundary-smoke.test.js
docker compose --project-name aviasurveil360-local-preprod --file deploy/local/compose.yaml --profile local-preprod-loader --profile aga-candidate-demo --profile aga-candidate-demo-oidc-fixture --profile aga-demo-workspace-loader --profile preproddemo config >/dev/null
bash scripts/check-compose-policy.sh
npm --prefix apps/web run test:e2e:aga-preprod -- --list aga-hybrid-classification-workspace.http.spec.ts aga-synthetic-lifecycle.http.spec.ts aga-hybrid-privacy.http.spec.ts
node scripts/build-aga-hybrid-forbidden-inventory.mjs --check tests/fixtures/aga-hybrid-forbidden-object-inventory.v1.json
node scripts/check-aga-hybrid-created-files.mjs --inventory tests/fixtures/aga-hybrid-created-file-inventory.v1.json --through task10
AVIA_AGA_HYBRID_AUTHORIZATION_LEDGER_DIR=/absolute/private/authorization-ledger-0700 AVIA_AGA_HYBRID_FAULT_AUTHORIZATION_LEDGER_DIR=/absolute/private/fault-ledger-0700 node scripts/verify-aga-hybrid-demo-workspace-evidence.mjs --check-summary docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_DEMO_LIFECYCLE_2026-08-04-v2.md
node tests/harness-docs-smoke.test.js
rg -n "docs/agent-harness|agent-harness/index|output-contract|verification-matrix|entropy-cleanup" AGENTS.md MANIFEST.md docs
node -e 'const fs=require("node:fs");const p="docs/exec-plans/active/2026-08-03-aga-hybrid-classification-demo-lifecycle-plan.md";const s=fs.readFileSync(p,"utf8");if(s.split(/\n/u).some((line)=>/[ \t]+$/u.test(line)))throw new Error("trailing whitespace");if(((s.match(/^```/gmu)||[]).length%2)!==0)throw new Error("unbalanced fences");console.log("aga-hybrid-plan-file-scan: ok")'
git diff --check
```

The Playwright list must report a nonzero count for each named spec and Task
9's happy-path connected run must execute that exact discovered set with no
skip. Task 10 validates Task 9's consumed-authorization, target, pre-provision
forbidden baseline, per-phase journal, seal, revocation, privacy, cleanup,
complete derived fault-matrix cases, concurrent token consumption, resume, and
zero-residue receipts offline. The tracked privacy-safe summary pins separate
happy-path and fault-ledger aggregate digests; it never reuses a one-shot authorization or
recreates a target. Because
an untracked plan is invisible to ordinary `git diff --check`, also run the
explicit target-file whitespace/fence scan defined in the Gate 0 contract test,
the closed created-file inventory checker, and inspect
`git diff --no-index /dev/null <plan>` while treating its normal exit 1 as
“different,” not as a verification failure. Inspect and clean
task-owned Chrome, helper, Playwright, Vite, Go-test, Docker, temp-build, and
database residue.

**Acceptance:** Plan, index, evidence, taxonomy, candidate artifact, source
counts, API/UI behavior, connected isolation, and literal status claims agree.
Stakeholder review remains required before lifecycle status can move to
`completed`.

## API And UI Boundary Summary

| Surface | Allowed | Forbidden |
|---|---|---|
| Immutable AGA overlay | Existing Admin-only sealed reads | Any mutation, new grant, Manager decision, classification write |
| Classification workspace | Admin complete read/reset; exact Manager scoped reads/successor commands | Auditee/lifecycle-only existence signal; original body copies; sealed-item mutation |
| Workspace question versions | Append-only new/reworded synthetic body, digest, parent, actor, time, reason | Original-body copy/overwrite; automatic HIGH; silent entry into sealed run |
| AI candidate artifact | Identity/digest, controlled classifications, provenance, blockers | Question text, legal/technical approval, arbitrary taxonomy code |
| Recommendation | Exact synthetic scope/target/type/profile/qualifiers/taxonomy/run/Draft/readiness pin | Client completeness flag; inference from AGA/form/risk/source/org type/confidence alone |
| Synthetic lifecycle | Operation-specific Manager/Inspector/Lead/reviewer/Auditee events and projections | Direct Inspector Finding; cross-organization/internal-note leak; canonical Planning/Audit/Finding/CAP/Evidence writes |
| Approval/publication UI | Separate visible blocked states and reasons | Fake approval, fake publication, cleared source blockers |
| Browser | Fixed routes, no-store bounded authorized views, transient memory | IDs/digests/filters in URLs/history/referrer; persistence/offline/telemetry/retained media |
| Database composition | Four separate least-privilege pools and Go composition after authorization | Cross-schema grants, joins, views, functions, triggers, or runtime loader credential |

## Verification And Acceptance Criteria

- Each AI pass and the final base artifact independently prove exactly 1,310
  classifications and 1,310 unique full immutable identities; the 1,278
  distinct text digests are not misused as identity.
- Exactly one allowed main domain per question; no missing domain.
- Zero unknown topic, profile, inspection-type, Evidence, target, qualifier,
  applicability, provider, or involvement codes.
- Zero `primaryProvider`, `secondaryProvider`, or `supportingProvider` fields in
  schema, artifacts, API, UI, or database.
- All identity/digest/extraction/source/authority/risk/decision facts reconstruct
  from the accepted package, including the exact 49/51 overlap.
- Fatal identity/schema/taxonomy/provenance failures abort sealing; every valid
  row maps through the total confidence/recommendation precedence.
- Every AI result records exact prompt/model-descriptor/run/taxonomy/batch/input/
  output digests; no model-weight claim or invented runtime identity appears.
- Pass disagreements and the hashed omission-review inventory remain
  reviewable; no lexical candidate or duplicated research role field is
  silently promoted.
- Admin and Department Manager share durable state; stale writers fail without
  losing either revision.
- Exact reason rules, atomic batch previews, immutable new/reworded versions,
  response replay, and forward-only reset are enforced under CAS/concurrency.
- Every Draft, recommendation, inspection snapshot, and Admin history view uses
  the closed Base-versus-Workspace question reference. Server-issued Workspace
  IDs cannot alias by digest or sequence; add/reword parent chains reconstruct
  one exact current leaf per root, and reset never migrates an old Workspace
  reference into a new generation.
- Recommendation fails closed on every missing/mismatched exact input.
- All 14 connected provider types are visible in demo configuration, while the
  current AGA profile remains `AERODROME_OPERATOR`-scope-only.
- Full synthetic Inspection/Response/Potential Finding/Lead conversion/Finding/
  CAP/Evidence/verification/Closure works and preserves CAP-versus-closure,
  verified-versus-authorized closure, and comment/note semantics.
- Technical approval/publication remain separate and blocked by current facts.
- Original overlay and the closed canonical business-table/sequence inventory
  have zero data-plane delta. Existing authentication control-plane changes are
  measured separately and never mislabeled as whole-database identity.
- Denied actors receive neutral no-store responses and no existence signal.
- Browser/test/observability artifacts and URL/history/referrer state contain
  no candidate body, identity, digest, filter, or lifecycle identifier.
- Reset isolates generations and preserves an append-only tombstone.
- Every task records its exact RED failure and GREEN pass. Focused, aggregate,
  tagged-build, nonzero discovery, connected, privacy, recovery, and residue
  gates have fresh literal results.

## Risks And Mitigations

| Risk | Mitigation |
|---|---|
| Provider analysis is mistaken for full question categorization | Require one domain for 1,310/1,310; keep external involvement optional |
| AI category drift or hallucination | Closed taxonomy/prompt before either pass, blind bounded passes in isolated read roots before shared materialization, per-pass bijection, rule-derived reconciliation, Manager successors |
| Pass-two blindness is asserted but the shared tree exposes pass one | Isolated read roots/private sinks, concurrent pre-materialization launch, negative visibility receipts, and fatal cross-pass visibility detection |
| Isolated pass roots or failing diagnostics leak bodies/identities | Caller-owned 0700 roots/0600 files, whole-root success/failure cleanup receipt, controlled aggregate-only diagnostics, and captured-stream leak tests |
| Discovery-sized boundaries exceed the complete classification envelope limit | Preserve the accepted discovery manifest and outputs; generate a separate text-free classification manifest whose worst-case closed pass envelope is at most 98,304 canonical UTF-8 bytes, bind its exact role-neutral source snapshot, and require both passes to use it |
| Free-text rationale or reusable evidence makes confidence non-deterministic | Controlled rationale/fact/rule references, exact proposal-field/value binding, validator-recomputed digests, fatal malformed provenance, LOW/MEDIUM missing-evidence rules |
| Set-like output order or duplicates change semantic digests | Freeze set/ordered arrays, normalize by cross-language tuple, reject duplicates, and test reorder-equivalence |
| Governance blockers make confidence unreachable | Keep agreement confidence independent; apply the total source-gap/review/high precedence without clearing blockers |
| A structural failure becomes an apparently reviewable row | Abort and reject the entire run on identity/digest/schema/taxonomy/provenance failure |
| Model agreement is mistaken for authority | Preserve source/attestation blockers and prohibit technical/publication transitions |
| Form metadata over-classifies broad forms such as Form 010/048 | Question-level semantic pass plus adversarial review; form is one signal only |
| External research misses, duplicates, or overstates involvement | Treat 30 duplicated-role pairs as candidate facts; require closed edge-specific condition/source/rationale/evidence/blocker objects; test empty/multiple edges and no mechanical duplication |
| Research target profiles corrupt canonical target kind | Separate canonical kind and target profile with schema enforcement |
| New/reworded text contradicts sealed-source storage | Keep all 1,310 original bodies in the accepted package/overlay only; store only append-only workspace-owned synthetic successors with parent/digest/provenance |
| Workspace question identity aliases by digest, sequence, or a client ID | Closed discriminated Base/Workspace references; server-issued per-generation root/version/proposal IDs; exact parent keys; duplicate/broken/current-leaf/snapshot/reset-history/direct-ID tests |
| Manager batch action changes unseen rows | Server-generated batch preview digest, explicit filter snapshot, CAS, immutable successor |
| A semantic edit retains HIGH auto-disposition or cannot correct an auxiliary proposal | Draft-only demotion with source-gap precedence, complete controlled proposal-resolution command, explicit later disposition, current-leaf recommendation tests |
| Shared writer weakens sealed overlay | Owner/loader/reader/EXECUTE-only command split, separate pools, no cross-schema object, privilege probes |
| Department Manager access leaks Admin-only raw package | Mutation-aware neutral CSRF protector, workspace-only exact binding, no direct overlay grant, POST-body pagination |
| Synthetic actors/scopes are absent or mistaken for canonical authority | Prepare-authorized read-only fixture export, target-bound private digest, workspace-only binding/load, no canonical assignment write |
| Lifecycle role access leaks classification or another organization | Fixed operation-role/projection matrix and neutral scope denial before object lookup |
| Demo lifecycle is mistaken for real governance | `DEMO_*` identities, simulation-only state, separate blocked approval/publication, no canonical bridge |
| Finding/CAP semantics are simplified incorrectly | Require Potential Finding/Lead conversion, human severity, distinct CAP/Evidence/verification, and separate authorized closure |
| Reset races an in-flight command or strands a lifecycle | Barrier-tested generation lock/CAS, reject every nonterminal lifecycle branch, append tombstone/new generation, no rollback/reactivation/selective delete |
| Reset succeeds into an empty unusable generation | Atomic reference to the existing sealed run/fixture plus deterministic revision-1 Draft; no loader or row copy; post-reset usability tests |
| OIDC writes invalidate a whole-database zero-delta claim | Freeze exact auth-control objects/columns per authenticated request/logout; keep canonical business and non-auth audit content/sequences at zero delta |
| Tagged/profile/browser gates false-green | Compile/test `preproddemo`, regenerate contracts, require all three Playwright specs in `testMatch` and nonzero list before connected run |
| Demo build silently contains workspace routes/data | Scan each just-built demo/HTTP artifact before overwrite; reject demo workspace markers/data and body-bearing maps, then run the broad HTTP artifact gate |
| Connected authority is issued before a target/fixture exists, expires mid-run, crashes between commit/receipt, or strands a declined run | Common target-bound envelopes reserve closed tokens while fresh; fsynced journal/ledger, predecessor receipt adaptation, recover/status/resume modes, exhaustive F1 fault tests plus the separately authorized four-case F3 matrix, and cleanup from every retained phase prove replay rejection/residue zero |
| 1,310-row UI leaks or stalls | Fixed URLs, POST-body filters, server pagination, bounded browser memory, purge on session transitions, no persistence/media |
| Umbrella size hides ordering/dependency defects | Ordered A–E slices plus stop-gated F1–F4 substops (only F2–F3 are connected), one writer, mandatory review gates, and no overlapping implementation |
| Dirty worktree loses user work | One writer, focused patches, pre/post diff inspection, no cleanup/reset/checkout |

## Dependencies

- The fixed input paths and hashes above must remain available and exact.
- The external research ZIP is candidate evidence only; if unavailable or
  changed, Gate 0/Task 2 is `blocked` rather than reconstructed from memory.
- If the workbook is unavailable, record that narrow limitation and continue
  with package/research facts; do not reconstruct its content. A changed
  workbook hash blocks using it as a frozen model input.
- The sealed overlay must be recreated in Task 9 from the exact 336,524-byte
  predecessor ZIP and pass current reconciliation; prior run 47 was cleaned up
  and is not a live dependency. Stakeholder review state is never advanced by
  this plan.
- Gate 0 must be independently accepted before model work. Each later child
  slice requires its predecessor stop gate and separate current implementation
  authorization; this plan artifact alone grants no execution authority.
- Both AI passes require an exact tool-reported model descriptor and isolated
  contexts. Missing metadata or inability to prevent pass-two access to
  pass-one output blocks sealing rather than weakening provenance.
- Existing synthetic preprod role fixtures must be available for connected
  multi-role qualification before the business baseline. Any missing actor is
  a workspace-fixture blocker, not permission for a workspace command to create
  a real identity, membership, department, or assignment.
- Exact source-owner attestation and real Department Manager governance facts
  are external blockers for technical approval/publication and are not required
  for simulation-only classification review.
- Local container/browser startup and whole disposable namespace cleanup
  authority must exist before connected qualification. No remote/external
  system is a dependency.
- Task 9 has a real authority stop: `prepare` may retain only the newly created
  target and exact intent; `qualify` cannot begin without the separately issued
  target-bound load/cleanup bundle. Missing authority leaves Task 9 pending and
  requires its owner to obtain cleanup authority; it is never reported as a
  successful or residue-free run.
- Task 9 F3 recovery acceptance additionally depends on the caller supplying the
  exact four-case prepare manifest followed by its target-bound run or cleanup
  manifest and distinct fresh
  prepare/qualification/recovery/cleanup authorizations. Missing cases leave
  the recovery matrix `blocked`; they do not weaken the happy-path result or
  permit a recovery claim.

## Idempotence And Recovery

- Deterministic batch preparation over unchanged inputs reproduces the same
  manifest digest. Repeated vocabulary/AI work gets a new immutable run ID and
  complete input/prompt/model-descriptor/taxonomy/batch/output digests.
- A sealed classification run cannot accept new child rows. Changed output
  requires a successor run and aggregate digest.
- Within `(generationId, actorSubjectId)`, `operationId` and `idempotencyKey`
  each have separate uniqueness constraints and are permanently bound to the
  same counterpart token, operation, canonical semantic hash, and stored
  response. The event/successor and both bindings commit in one transaction.
  Reusing the exact pair with the same payload returns the stored response;
  reusing either token with a different counterpart, operation, or payload
  conflicts and writes nothing, including under concurrent cross-pair races.
  This binding lookup precedes current-generation/domain lookup after neutral
  authentication and current exact operation/scope authorization. The binding
  stores that closed authorization-scope digest. An exact reset replay returns
  its committed response only for a still-current exact Admin; revoked/stale/
  wrong-scope actors receive neutral denial, while an unseen or mismatched
  request naming the reset generation writes nothing.
- A crash before workspace transaction commit leaves no partial run/Draft/event.
  A crash after commit but before response delivery is recovered only by exact
  replay of the stored response; it never reconstructs a second event.
- Stale Manager edits return conflict with the latest revision metadata; they
  never overwrite a successor.
- Barrier tests make parallel same/different-key commands, Draft CAS, lifecycle
  transitions, load/seal, and reset/command ordering deterministic. The
  generation lock and expected seal decide the single winner; a loser writes
  nothing.
- Recommendation and inspection snapshots pin exact taxonomy, classification
  run, Draft revision/content digest, readiness event, scope version, target
  kind/profile, qualifiers, and effective time.
- Reset requires reason, exact `expectedGenerationId`,
  `expectedGenerationRevision`, and `expectedGenerationSealDigest`. It appends
  a tombstone, references the existing immutable
  sealed taxonomy/run/fixture, and creates the deterministic revision-1 Draft
  in one transaction without a loader or 1,310-row copy, and it is rejected
  while any lifecycle root is nonterminal. Old rows remain
  immutable and unavailable to ordinary routes; a failed reset rolls back
  fully, while rollback/reactivation after a committed reset is forbidden.
- Partial/unsealed/corrupt generations are unreadable. Recovery uses a new
  generation or whole disposable sibling-schema recreation. Product reset is
  distinct from Task 9 teardown; neither repairs the sealed overlay or a
  canonical table.
- Task 9 operational recovery is separate from product generation recovery.
  Its hash-chained journal/ledger and fixed phase machine resume only the exact
  target/intent: completed new-workspace effects return atomically stored
  receipts; inherited base/overlay effects reconcile their unchanged
  predecessor receipts; a fresh recovery authorization continues the first
  incomplete reserved phase; and a fresh cleanup authorization may dispose any
  retained phase. Kill/publication and concurrent-consumption cases are
  required connected evidence, not inferred from transaction unit tests.

## Progress

- [x] 2026-08-03: Independently confirmed the accepted package has 52 forms,
  31 question-bearing forms, 1,310 unique full boundaries/identities, 1,278
  unique text digests, exact per-form counts/ordinals/digests/states, and the
  exact package/loader-ZIP hashes without sampling or question-text output.
- [x] 2026-08-03: Validated the external research ZIP, 20-code provider matrix,
  1,310-row question CSV, 51-row ambiguity inventory, aggregates, and exact
  reconciliation to the accepted package. The 30 provider pairs duplicate two
  role fields, and the 51 unresolved set contains all 49 source-gap identities
  plus two others.
- [x] 2026-08-03: Inspected the System Design Matrix and established that it is
  available at the fixed path/hash with two worksheets and a 27-row `A1:K28`
  grouped 52-form workflow map, not a question/provider classification matrix.
- [x] 2026-08-03: User approved one mandatory main domain plus optional topic
  tags, AI high-confidence preselection without approval, immutable Manager
  successor Drafts, the hybrid two-pass approach, the 18-domain taxonomy,
  the sibling shared workspace, the complete synthetic lifecycle, and the
  fail-closed security/testing design.
- [x] 2026-08-03: Created this active ExecPlan and synchronized the plan index.
- [x] 2026-08-03: Independent adversarial first pass found 6 Critical, 24
  Important, and 4 Minor plan-quality findings. Corrections separated semantic
  confidence from governance blockers; restored Potential Finding/Lead
  conversion; defined body, route/role, CSRF, database-role, loader,
  recommendation, readiness, concurrency/reset, privacy, TDD, tagged-build,
  Playwright, and ordered child-slice contracts.
- [x] 2026-08-03: Completed the iterative second adversarial pass after
  correcting the remaining connected-recovery, exact Workspace-question
  reference, Base-first reword ordering, snapshot, reset-history, and direct-ID
  contracts. The final live-file verdict has zero unresolved Critical or
  Important plan-quality findings; the F4 handoff is explicitly offline.
- [x] 2026-08-03: Revalidated every one of the 1,310 accepted boundaries and
  its ordered research-row match without sampling or question-text output,
  all fixed bytes/SHA-256 claims, the 49/51 overlap, 20-provider 14/6
  partition, and the available 2-sheet/27-group/52-form workbook map. Fresh
  docs-only checks passed: `git diff --check` exit 0;
  `node tests/harness-docs-smoke.test.js` printed `harness-docs-smoke: ok`;
  the required harness-reference scan exited 0; the direct target-file and
  plan/index consistency scans printed their `ok` markers; and the
  unfinished-marker scan returned no matches.
- [x] 2026-08-03: Re-read the complete governing authorities and routed
  predecessor/product contracts, preserved the 46-file tracked dirty baseline
  plus unrelated untracked predecessor evidence, and independently reviewed
  the implementation plan. The review corrected the canonical
  `CAP_ACCEPTED` discovery, the four omitted Task 1 focused-test names, and the
  missing separate local operator-issuance procedure without weakening any
  technical stop gate.
- [x] 2026-08-03: Gate 0A TDD RED ran first. Both required Node tests failed
  with the controlled `ERR_GATE0_PREPARER_MISSING` result (0 pass, 2 fail), as
  expected before the preparer or receipts existed. After the smallest
  deterministic implementation, the diagnostic privacy test passed and the
  coverage test advanced to controlled `ERR_GATE0_RECEIPT_MISSING` (1 pass,
  1 fail). This is not a Gate 0A pass.
- [x] 2026-08-03: Gate 0A discovery receipt and text-free coverage are
  independently accepted. This acceptance remains current for the immutable
  24-batch discovery artifacts.
- [ ] 2026-08-03: Gate 0B classification-manifest scope is reopened. Earlier
  Gate 0B/Slice A acceptance evidence remains historical but is superseded for
  this scope pending fresh focused re-review.
- [x] 2026-08-03: The first unsealed vocabulary-discovery attempt was stopped
  and all partial outputs were discarded after an independent read-only agent
  found that the prompt said “domain-separated” without freezing the exact
  identity domain separator. No discovery receipt or taxonomy output was
  created from that attempt. Prompt V2 now freezes the identity and output
  digest algorithms; every batch restarts from ordinal 1 under the new digest.
- [x] 2026-08-03: Three isolated Prompt V2 contexts completed all 24 bounded
  vocabulary-only batches with the explicitly configured `gpt-5.6-sol` model,
  `xhigh` reasoning effort, and no row classification. The text-free outputs
  cover all 1,310 ordered complete identities. Runtime snapshot/version and
  service-tier fields were not exposed and are recorded as unavailable; no
  model-weights or invented metadata claim is present.
- [x] 2026-08-03: The first Gate 0A implementation review rejected advancement
  with 1 Critical and 4 Important findings after the reviewer withdrew its
  separate model-metadata concern. Focused correction tests first produced the
  expected RED result (1 pass, 3 fail): the all-identity diagnostic mode was
  missing, context paths were not opaque, and omission rules were not frozen as
  declarative data. Corrections retained every complete normalized per-batch
  output preimage, closed and independently reconstructed the full digest
  graph, removed unverifiable timestamps and private context paths, froze and
  independently rebuilt all 106 omission-review items, exhaustively constrained
  diagnostics, and recursively rejected body/private-path/provider-hierarchy
  leakage. The temporary 28-file private output sink was removed after sealing.
- [x] 2026-08-03: The corrected focused command
  `node --test tests/aga-hybrid-classification-plan-contract.test.mjs` passed
  all 4 tests with 0 failures in 3.2 seconds. This is fresh local Gate 0A
  technical evidence; independent acceptance is still in progress, so Gate 0B
  has not started.
- [x] 2026-08-03: The first Gate 0A re-review rejected advancement with 1
  Critical orphan-accounting finding and 1 Important diagnostic finding. All
  157 retained model omission signals now have a one-to-one digest-bound
  disposition, resolve through the manifest to a complete identity, use one of
  23 closed candidate-to-frozen-rule mappings, and carry a recomputed input-fact
  digest. Five signals match a deterministic frozen rule and link to the exact
  106-item inventory; 152 remain explicitly rejected with controlled
  input-selector or rule-match reasons rather than being silently normalized.
  The diagnostic now perturbs one real-derived complete identity through the
  normal validator and renders only batch/count/digest, and the ordered identity
  comparison can no longer dump raw tuples on failure. The focused test first
  recorded the expected 0-pass/2-fail RED, then passed all 5 tests with 0
  failures in 3.3 seconds. A fresh independent final re-review is in progress;
  Gate 0B has not started.
- [x] 2026-08-03: The final Gate 0A review found one additional Important
  failure-diagnostic issue: `assert.match` could echo a captured private stream.
  A focused static regression first failed with the controlled
  `ERR_GATE0_DIAGNOSTIC_ASSERTION_CAN_ECHO_CAPTURED_STREAM`, then the assertion
  was replaced with a boolean-only shape check. The fresh exact Gate 0A suite
  passed 6 tests with 0 failures in 3.3 seconds, and the independent reviewer
  returned `ACCEPT` with zero Critical, Important, or Minor findings. Gate 0A
  is complete; Gate 0B is next.
- [x] 2026-08-03: Gate 0B recorded the intended unfinished-freeze RED, then
  froze the exact 18-domain taxonomy, controlled provider partition, row
  prompt, closed output/private-input schema, status and transition registry,
  digest graph, and 105-path planned-create inventory. A read-only pre-audit
  found no Critical issue and identified three Important plus two Minor
  ambiguities: cross-language canonical string bytes/model-descriptor order/
  identity domains, an impossible V1 `PERSON` target pairing, pooled lifecycle
  state cross-products, snapshot availability inconsistency, an unmodeled
  rejection response, and an implicit taxonomy self-digest. Focused tests
  recorded RED before each correction. All findings are corrected: canonical
  JSON has a non-ASCII/HTML known answer, descriptor and identity domains are
  explicit, `PERSON` is intentionally excluded from V1 proposals, lifecycle
  transitions are entity-specific, metadata availability is conditional,
  rejection is closed, and the taxonomy self-digest is machine-readable.
- [x] 2026-08-03: Historical/superseded Gate 0B technical evidence: the exact Node
  suite passed 14/14 with 0 failures; the preparer printed
  `aga-hybrid-batches: ok batches=24 items=1310`; the rerun suite again passed
  14/14; the inventory checker printed
  `aga-hybrid-created-files: ok through=gate0 due=13 planned=105`; harness docs
  printed `harness-docs-smoke: ok`; the plan scan printed
  `aga-hybrid-plan-file-scan: ok`; and `git diff --check` exited 0. Gate 0B is
  technically GREEN but not yet independently accepted; fresh contract review
  is the current stop gate before Task 1.
- [x] 2026-08-03: The first independent Gate 0B review rejected advancement
  with 2 Important findings and no Critical or Minor findings. It found that
  final-item `inputDigest` could be interpreted as either a per-pass batch
  digest or the common run-input digest, and that a returned Potential Finding
  had no reachable immutable same-root successor path before checklist
  resubmission. The focused correction test recorded the expected RED, then
  passed. The taxonomy and schema now distinguish typed pass-input versus
  run-input digest roles and bind every occurrence; returned roots use the
  controlled `CREATE_POTENTIAL_FINDING` successor branch from `RETURNED` to
  `PENDING_LEAD_REVIEW`, preserving root identity and prior immutable version.
  The refrozen exact suite passed 15/15 with 0 failures. Fresh independent
  re-review is the current stop gate; Task 1 has not started.
- [x] 2026-08-03: The fresh Gate 0B re-review accepted the input-digest repair
  but rejected advancement with 1 Important finding and no Critical or Minor
  findings: checklist submission could still occur with a latest `RETURNED`
  root, and the successor did not prove a post-return corrected response. The
  focused test recorded RED, then added strict response-revision ordering,
  changed semantic digest, exact successor response binding, a controlled
  `RETURNED_ROOT_SUCCESSOR_REQUIRED` submission guard, and negative tests for
  premature submit and pre-return response reuse. The refrozen exact suite
  again passed 15/15 with 0 failures. Focused independent confirmation is the
  current stop gate; Task 1 has not started.
- [x] 2026-08-03: Historical/superseded focused independent Gate 0B confirmation returned `ACCEPT`
  with 0 Critical, 0 Important, and 0 Minor findings. It verified the newer
  response revision/digest binding, exact successor pin, returned-root submit
  denial, and all negative tests. The exact suite passed 15/15 and
  `git diff --check` exited 0. Slice A is complete; Task 1 TDD RED is next.
- [ ] 2026-08-03: Task 1 implementation is in progress. Its focused TDD set,
  pure classification/Draft/recommendation domain, and two correction rounds
  exist locally. The implementation writer reported green focused/full Go and
  vet runs after the latest round, but the fresh independent Sol Max review was
  interrupted before a verdict and primary-agent acceptance has not run.
  Therefore Task 1 is not yet `verified locally`.
- [ ] 2026-08-03: The resumed independent Sol Max Task 1 review returned
  `REJECT` with 3 Critical, 5 Important, and 0 Minor findings. The Critical
  defects are missing question-body/frozen-fact authority binding, missing
  candidate/challenge role-neutral snapshot equality, and forbidden post-seal
  retention of private bodies coupled to a non-round-trippable Draft handoff.
  Important defects cover runtime discovery-digest exclusion, strict sealed-item
  semantic decoding, private-input/model bounds, controlled diagnostics, and
  five remaining required-test regressions. The exact focused command failed
  with 4 failures in 164.529 seconds and the full package failed with 5 in
  200.167 seconds; vet, gofmt, scoped diff, whitespace, provider-hierarchy, and
  direct sensitive-field logging checks passed. Correcting every Critical and
  Important finding remains mandatory.
- [ ] 2026-08-03: A subsequent read-only full-envelope calculation found a
  predecessor defect before Task 1 correction could continue. Reconstructing
  the exact closed `classificationPassInput/v1` envelope over the accepted
  private items produced 21/24 batches above the frozen 98,304 canonical-byte
  limit; the maximum was 99,653 bytes and the per-pass total was 2,338,919.
  The accepted discovery manifest measured its smaller discovery envelope, so
  its boundaries cannot serve Task 2 unchanged. Gate 0A discovery outputs stay
  immutable; Gate 0B is reopened to freeze a separate digest-bound,
  full-envelope-safe classification manifest. Task 1 is paused at its rejected
  review state until that predecessor correction is independently accepted.
- [ ] 2026-08-03: Gate 0B correction facts: exact reconstruction found 21/24
  discovery boundaries above 98,304 bytes (maximum 99,653); maximum-valid
  sizing requires 25 batches. The first focused RED was
  `ERR_CLASSIFICATION_MANIFEST_MISSING`; the discovery-root correction RED was
  `ERR_CLASSIFICATION_DISCOVERY_ROOT_UNBOUND`. The refrozen preparer reports
  `aga-hybrid-batches: ok discoveryBatches=24 classificationBatches=25 items=1310`;
  the exact Node suite is 17/17, the inventory is 14/106, and all four Gate 0A
  artifact hashes remain byte-identical. A focused Sol review returned `REJECT`
  with 0 Critical and 2 Important findings (0C/2I): independent ordered-union
  validation and plan/index truth were required.
- [ ] 2026-08-03: The ordered-union correction recorded
  `ERR_CLASSIFICATION_ORDERED_UNION_VALIDATOR_MISSING` RED, then added a
  producer-bound validator that rejects reordered, duplicate/omitted, and
  ordinal-drift manifests even after self fields are recomputed. Focused GREEN
  and the full Gate 0B gates are recorded locally: Node 18/18, preparer
  `aga-hybrid-batches: ok discoveryBatches=24 classificationBatches=25 items=1310`,
  inventory 14/106, harness-docs smoke, plan scan, and diff/direct scans pass.
  The preserved Gate 0A file SHA-256 values are
  `aef10701332f833651ec28783cd2f2965cc618c796300d010a02e757880e72b2`,
  `593d18bbeee297339c302213c889581a7a3f958103a57bddc4ab5a91253bc7cf`,
  `229692632dfb1f58dcd1c0f60298c0c4535076bc0b8b9cb8d70eae36b5fc8453`, and
  `336654cdabbd9fde1ef88e4314bfb4e2474fe1f7a3480f65c9389ff05a420859`.
  Fresh focused Sol re-review is next. Gate 0B is not independently accepted
  and Task 1 remains paused.
- [ ] 2026-08-03: The subsequent focused Sol Gate 0B re-review returned
  `REJECT` with 0 Critical and 2 Important findings (0C/2I). The contract test
  could render full identity tuples through `assert.deepEqual`, and its
  duplicate-plus-omission mutation did not independently prove the controlled
  count-error branch. The correction first recorded the focused RED
  `ERR_CLASSIFICATION_ORDERED_UNION_DIAGNOSTIC_LEAK` (18 pass, 1 fail), then
  replaced the assertion with boolean equality of canonical ordered-identity
  digests and added a self-field-recomputed pure omission mutation requiring
  `CLASSIFICATION_MANIFEST_IDENTITY_COUNT_INVALID`. The focused Node suite is
  green at 19/19. The exact preparer reports
  `aga-hybrid-batches: ok discoveryBatches=24 classificationBatches=25 items=1310`;
  inventory reports 14/106; harness-docs smoke, plan scan, `git diff --check`,
  and the direct public-artifact privacy scan all pass. A fresh focused Sol
  re-review remains pending. Gate 0B is not independently accepted and Task 1
  remains paused.
- [x] 2026-08-03: Sol Max independently accepted the corrected Gate 0B with 0
  Critical, 0 Important, and 0 Minor findings. The fresh exact suite passed
  19/19 and the 25-batch classification-manifest facts passed. Gate 0B is now
  the accepted predecessor; Task 1 correction is in progress and Tasks 2–10
  remain `not run`.
- [ ] 2026-08-03: Task 1 correction recorded RED for the missing role-neutral
  private-input digest, then corrected the 25-batch runtime pin, body SHA-256
  binding, derived evidence facts, candidate/challenge snapshot equality,
  text-free result receipt handoff, sealed-item semantic/canonical decoding,
  bounded private/model inputs, and controlled diagnostics. The exact focused
  Go command and the full `./internal/agaapplicability` package both exited 0
  with `GOCACHE=/tmp/avia-aga-go-cache`; `go vet ./internal/agaapplicability`,
  harness-docs smoke, and `git diff --check` also exited 0. The requested
  `--through task1` inventory is blocked with
  `ERR_AGA_HYBRID_CREATED_FILES code=DUE_FILE_MISSING` because it also marks
  Task 2 validator/candidate artifacts due; those Task 2 paths remain
  intentionally uncreated. Task 1 is locally corrected but not independently
  accepted. The next action is focused Sol review of Task 1; Tasks 2–10 remain
  `not run`.
- [ ] 2026-08-03: A fresh root exact 17-test Task 1 command disproved the
  earlier local green (exit 1 after 163.843 seconds): record-order receipt
  validation returned `ErrDigestMismatch` before `ErrPassBijection`, and draft
  semantic/resolution validation returned `ErrInvalidResolution` before the
  required pass-bijection branch. The correction adds the ordered-record
  bijection check before seal reconciliation and returns `ErrPassBijection`
  before generic Draft resolution validation when a sealed pass has been
  replaced. The rerun exact focused command and full
  `GOCACHE=/tmp/avia-aga-go-cache go test -count=1 ./internal/agaapplicability`
  both exited 0. Task 1 remains locally verified only and awaits focused Sol
  review; Tasks 2–10 remain `not run`.
- [ ] 2026-08-03: A third root exact command again failed (exit 1 after
  164.306 seconds). The correction preserves `ErrPrivateInputMismatch` while
  joining the compatible controlled `ErrDigestMismatch`, restores duplicate,
  ordered-record, selector, role/run, governance, and aggregate error
  precedence before receipt-output reconciliation, and creates the
  final-topic-removal test precondition through the public first-topic-removal
  command rather than an invalid direct mutation. The two-test RED was
  `private classification input mismatch` and `proposal resolution is
  incomplete or invalid`; its GREEN printed
  `ok github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability`.
  The final verbose full package rerun passed its listed tests, including the
  corrected 25-batch receipt assertion. Task 1 remains locally verified only
  and awaits focused Sol review; Tasks 2–10 remain `not run`.
- [ ] 2026-08-03: Root parallel verification confirmed the exact required
  17-test command passed, but the full package failed after 223.477 seconds
  only at `TestDraftRequiresResolvableSealedPasses`: a blank recommendation
  code was obscured by derivative aggregate/run digest validation. Task 1 now
  checks the controlled recommendation enum before those derivative checks and
  checks the required sealed row/pass cardinality before unrelated taxonomy
  pins. The isolated RED was `blank recommendation state error = digest
  mismatch`; the isolated GREEN printed
  `ok github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability 8.650s`.
  The full `GOCACHE=/tmp/avia-aga-go-cache go test -count=1
  ./internal/agaapplicability` rerun exited 0. Task 1 remains locally verified
  only and awaits focused Sol review; Tasks 2–10 remain `not run`.
- [x] 2026-08-04: The final Task 1 correction recorded RED for a reachable
  unknown `DraftAction` diagnostic that echoed caller-controlled input, then
  GREEN after replacing it with a controlled non-echoing error and adding the
  focused regression. Fresh independent local evidence after that correction
  passed the exact 17-test command, full owning package, `go vet`, docs smoke,
  `git diff --check`, privacy regressions, unfinished-marker scan,
  direct-sensitive-logging scan, and provider-hierarchy scan. Exactly one
  bounded read-only GPT-5.6 Sol xhigh adversarial review returned `ACCEPT` with
  0 Critical, 0 Important, and 0 Minor findings. It confirmed the diagnostic
  correction at `apps/api/internal/agaapplicability/draft.go:692` and its
  regression at `draft_test.go:20`, then rejected the earlier provenance
  concern at the exact Gate 0B boundary: `fixedInputDigests` is a closed
  11-field object,
  `sourceSnapshotDigest` is a separate classification-batch-entry field owned
  by Task 2 reconstruction, and Task 1 pins the accepted classification-
  manifest root. The requested `--through task1` inventory remains `blocked`
  with `ERR_AGA_HYBRID_CREATED_FILES code=DUE_FILE_MISSING` because the mapping
  also requires intentionally uncreated Task 2 validator/candidate artifacts.
  Task 1 is independently accepted and `verified locally`; Tasks 2–10 remain
  `not run`.
- [x] Task 1 implementation and verification — independently accepted and
  `verified locally`.
- [x] 2026-08-04: Task 2 recorded the required validator TDD RED/GREEN.
  The focused Go command initially failed because the validator symbols were
  absent; after the smallest offline ZIP/schema/isolation implementation, the
  focused seven-test command exited 0 in 0.344s, the full validator package
  exited 0 in 0.545s, `go vet` exited 0, and the Node receipt test passed 1/1
  in 0.092s. Both untrusted ZIPs safely scanned at 25/25 batches and
  1,310/1,310 records, with 27 semantic and 29 AppleDouble/`__MACOSX`
  transport-noise entries each. The exact fresh `validate-pass` command for
  pass one exited 1 in 0.751s with `ERR_AGA_PASS_SCHEMA`; pass two exited 1 in
  0.830s with the same controlled marker. The exact `reconcile` and
  `validate-candidate` commands exited 1 in 0.323s and 0.537s respectively
  with `ERR_AGA_CANDIDATE_INVALID`. These are required blocks, not passes:
  each raw response record omits `classificationRunId`, `passRunId`,
  `promptDigest`, `modelDescriptorDigest`, `inputDigest`, and
  `passResultDigest`; each batch omits `batchOutputDigest`; supplied metadata
  also leaves candidate `modelId` and challenge `modelId`, `service`, and
  `interface` unavailable, with neither pass supplying requested reasoning
  effort or fork configuration. The validator did not infer or synthesize any
  field. The private root, ZIPs, extraction directories, pointer file, and
  task-named process residue were removed after the blocked validation; the
  text-free receipt records zero remaining files, directories, and matching
  processes. No candidate pass sealed, no reconciliation/artifact was
  manufactured, and no review is requested while the required Task 2 gates are
  blocked.
- [x] 2026-08-04: User-authorized Gate 0B metadata-availability amendment
  recorded RED before implementation: the focused Go metadata tests rejected
  truthful `null`/`unavailableFields` platform receipts and accepted a displayed
  label as an exact model ID. GREEN changed only the closed metadata contract:
  every unavailable scalar is literal `null` with an exact sorted marker,
  every available scalar is non-null and unmarked, and a displayed model label
  cannot establish an exact model ID. The amendment accepts truthful missing
  platform fields as `candidate-only` demo provenance without fabricating a
  model ID, reasoning effort, fork setting, or platform claim. Fresh focused
  Go tests for `TestModelDescriptorAcceptsTruthfulPlatformUnavailableMetadata`
  and the two Task 2 metadata validator tests passed; the 19-test Gate 0B Node
  contract suite passed. The changed taxonomy, schema, prompt, specification,
  embedded pure-Go taxonomy authority, validator, tests, and frozen file
  digests are synchronized. Independent Gate 0B amendment review is next.
- [x] Task 2 implementation and local acceptance — `verified locally`. The
  replacement candidate ZIP (`sha256:36da15f34be44f883372aec588eca8413d2e370ed482dcdd84c50f2344b45a9c`,
  509,331 bytes) and challenge ZIP
  (`sha256:6c415cbd77189b7ab08db108bda87dae8adcecf46bae5db93b7d69130a04493e`,
  517,459 bytes) each passed the closed archive, metadata-availability,
  identity, 25-batch, and 1,310-record checks. The validator retained each
  ZIP's immutable prompt/model/input/batch/record/pass receipt digests as
  source evidence and did not rebind the former fixed prompt or descriptor
  digest. Both private roots were removed with zero file/directory/process
  residue. Reconciliation emitted `AGA_CANDIDATE_RECONCILED`; the artifact
  validator reread and rebuilt it with `AGA_CANDIDATE_VALIDATED`. The sealed
  artifact has 1,310 items and 2,620 pass records, and its cleanup receipt is
  text-free. This is local candidate evidence only; bounded independent Task 2
  acceptance is pending. Tasks 3–4 are `verified locally`; Task 5 is the next
  implementation slice. Tasks 6–10 remain `not run`.
- [ ] 2026-08-04: Replacement pass ZIP intake remains `blocked` before either
  pass can seal. The candidate immutable input receipt is
  `sha256:36da15f34be44f883372aec588eca8413d2e370ed482dcdd84c50f2344b45a9c`
  at 509331 bytes; the challenge immutable input receipt is
  `sha256:6c415cbd77189b7ab08db108bda87dae8adcecf46bae5db93b7d69130a04493e`
  at 517459 bytes. Fresh isolated `validate-pass` runs copied one input at a
  time into a new 0700 role-specific root with a 0600 file, scanned without
  extraction, and removed the entire root on return; direct post-run residue
  checks found neither root. Both runs exited 1 with the controlled marker
  `ERR_AGA_PASS_METADATA`. The subsequent user-authorized supplied-provenance
  compatibility correction recorded RED because the normal record constructor
  still required the repository prompt pin; GREEN adds a ZIP-ingestion-only
  constructor that requires a lexical canonical SHA-256 prompt and preserves
  the supplied immutable prompt/model digests through receipt, batch, record,
  and seal reconstruction. It does not invent a model value or treat a
  displayed label as an exact model ID. The candidate independently completed
  this closed validation with `AGA_PASS_VALIDATED` at 25 batches and 1,310
  identities. The separately produced challenge then required the exact
  optional closed `metadataAcceptanceStatus` value
  `BLOCKED_MISSING_PLATFORM_METADATA` (a value its producer was instructed to
  emit) and the controlled `batch-NN.response.json` transport filename; RED
  rejected both before implementation and GREEN permits neither arbitrary
  statuses nor arbitrary filenames. The challenge also independently completed
  `AGA_PASS_VALIDATED` at 25 batches and 1,310 identities. Both private roots
  were removed after success. At the time of this intake note, no normalized
  pass had yet been copied into the tracked deliverable directory; the
  subsequent local reconciliation and artifact validation are recorded in the
  Task 2 acceptance entry below.
  The focused private-ingest RED initially failed because
  `validatePassZIPInPrivateRoot` was absent; its GREEN passed after adding
  isolated copy/no-extraction/cleanup behavior. Fresh focused ingestion and
  compatibility tests, the full validator package, and `go vet` exited 0.
  Reconciliation is now authorized only over these two independently validated
  sealed passes.
- [x] 2026-08-04: Final Task 2 local verification — `verified locally`. The
  focused Go domain/validator tests, `go vet ./...`, candidate artifact Node
  test, harness-docs smoke test, and `git diff --check` passed. Full API
  `go test ./... -count=1` ran with task-owned pinned PostgreSQL 17.6 and
  MinIO services; every package, including `apps/api/tests/integration`,
  reported `ok`. The initial wrapper exit was caused only by a zsh cleanup
  variable collision; the exact task-owned containers, volumes, network, and
  generated runtime directory were then removed and verified absent. The full
  19-test Gate 0B contract suite reached 14 passing tests; five remain
  `blocked` because the fixed external System Design Matrix workbook is
  absent. That workbook gate is outside the Task 2 candidate artifact and
  does not invalidate its scoped reconciliation evidence.

### 2026-08-04 — Tasks 3–5 local implementation evidence

- [x] Task 3 — The isolated sibling workspace schema/roles, fixture template,
  one-shot commands, loader barrier, append-only memory/PostgreSQL seams,
  Compose services/secrets, and static boundary were implemented. Focused Go
  tests passed for the workspace package and all three commands; the boundary
  test passed 3/3; `go vet`, selected Compose config, Compose policy 21/21,
  and `git diff --check` passed. The combined Task 3 Node command passed 21
  tests and retained five pre-existing Gate 0/Task 2 failures: missing fixed
  external workbook path and predecessor reconstruction diagnostics. Those
  failures were not changed because the workbook is external and the Task 2
  artifact is accepted. Live persistence, grants, and zero-delta evidence are
  `not run` until Task 9.
- [x] Task 4 — The exact workspace operation matrix, neutral CSRF-aware
  protector, closed service/query/command envelopes, authorization scope
  digest, idempotency-before-domain lookup, direct-ID neutral denial,
  separate reader/command tagged pools, fixed routes, OpenAPI source/bundler,
  generated Go/TypeScript contracts, and legacy five-route preservation were
  implemented. Focused Go tests passed for the service and HTTP packages,
  default and `preproddemo` API tests passed, the tagged artifact built, the
  generated contract check passed, and the focused OpenAPI suite passed 20/20.
  Lifecycle commands remain an explicit unavailable capability until Task 7;
  no fake lifecycle success is returned. Browser and connected evidence remain
  `not run`. Scoped evidence is recorded in
  `docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_TASK3_TASK4_2026-08-04.md`.
- [x] Task 5 — The shared HTTP-only classification workspace, capability-gated
  role routes, bounded server-paginated/filterable inventory, exact Draft
  actions, provider eligibility, immutable successor controls, stale recovery,
  and Admin-only generation history/reset were implemented without importing
  the workspace into the demo entry or changing the old raw-package panel.
  The scanner TDD RED/GREEN, focused Vitest 41/41, scanner fixtures 4/4,
  typecheck, demo build/scan, HTTP build/scan, and HTTP artifact scan all passed.
  Evidence is recorded in
  `docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_TASK5_2026-08-04.md`.
  Browser and connected evidence remain `not run`; Task 6 was next in order
  and is now locally complete.
- [x] Task 6 — Deterministic recommendation request pins, server-derived-fact
  fail-closed handling, immutable recommendation/snapshot digests and full
  question-reference validation, append-only memory/PostgreSQL snapshot seams,
  HTTP neutral no-store behavior, and the authorized Planning-only neutral AGA
  status surface were implemented. Focused recommendation and HTTP tests,
  Planning Vitest 15/15, typecheck, demo/HTTP builds, both artifact-boundary
  scans, and the HTTP artifact scan passed. OpenAPI and generated Go/TypeScript
  contracts were regenerated. Evidence is recorded in
  `docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_TASK6_2026-08-04.md`.
  The append-only lifecycle state machine, projections, PostgreSQL command-store
  seam, HTTP behavior, and regenerated contracts passed the Task 7 focused and
  package gates. Evidence is recorded in
  `docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_TASK7_2026-08-04.md`. The
  multi-role lifecycle UI, fixed supplemental suffix routes, responsive
  controls, privacy/purge tests, and three nonzero preprod Playwright specs
  passed the Task 8 focused, build, artifact-boundary, and discovery gates;
  evidence is recorded in
  `docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_TASK8_2026-08-04.md`.
  PostgreSQL/grant/zero-delta and connected role/browser evidence are
  `verified locally` on the disposable Task 9 target. Task 9's offline and
  connected evidence is recorded in
  `docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_TASK9_OFFLINE_2026-08-04.md`
  and the create-once privacy-safe lifecycle summary. Task 10 is
  `verified locally` after its full aggregate ladder and summary check.

- [x] Task 7 — The synthetic lifecycle aggregate now enforces the exact
  recommendation, generation/readiness, organization/provider-scope, pinned
  Inspector/Lead/Auditee bindings, operation-role, idempotency/CAS, required
  comments, answer, CAP/Evidence/due-date, and append-only event rules. The
  complete state machine preserves Potential Finding/Lead conversion, CAP
  acceptance versus Finding closure, Evidence review versus verification, and
  separate authorized closure. Public, CAA, and Auditee projections are
  structurally separated and redacted. The exact ten-test focused lifecycle
  command, all three owning Go packages, contract generation/check, and the
  three-test workspace contract suite passed. Evidence is recorded in
  `docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_TASK7_2026-08-04.md`.
  Canonical zero-delta, connected PostgreSQL/grant, and browser behavior are
  `verified locally` in Task 9's isolated disposable qualification; Task 8 is
  now verified locally.

- [x] Task 8 — The synthetic multi-role lifecycle UI now provides the
  Inspector response/Potential Finding path, Lead review/conversion path,
  Service Provider CAP/Evidence path, CAA review/verification path, separate
  Manager authorized closure, CAA-only history/public Auditee projections,
  fixed identifier-free route suffixes, responsive controls, and purge/role
  boundary coverage. Manager recommendation and simulation-release controls
  remain visibly fail-closed until exact server-derived pins are returned; the
  UI never invents them. The three required preprod specs discover 7 tests with
  trace, screenshot, and video disabled. Focused UI tests passed 16/16,
  typecheck, demo/HTTP builds, both artifact-boundary scans, and the HTTP
  artifact scan passed. Evidence is recorded in
  `docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_TASK8_2026-08-04.md`.
  Connected browser execution and PostgreSQL/grant/zero-delta evidence are
  `verified locally` in the Task 9 receipt-bound disposable run; production
  evidence remains not established.

- [x] Task 9 — The closed forbidden-object inventory (129 parsed migration
  objects), exact role/grant and Compose coverage, separate CSPRNG
  authorization issuance, target/operation validation, receipt-bound
  fail-closed connected harness, hash-chain ledgers, and create-once
  privacy-safe finalizer are `verified locally`. F1 passed the expanded
  12-test boundary suite and both sibling-schema load/seal barriers. F2
  qualified the task-owned disposable predecessor with 9 synthetic OIDC
  accounts across 8 role families, 1,310 workspace items, one workspace seal,
  receipt-backed exporter/loader revocation, zero forbidden/overlay delta,
  isolated browser `17/17`, and zero whole-namespace residue. F3 completed
  the exact four-case connected recovery matrix with one concurrent-token
  winner, no loser effect, target receipt recovery, cleanup receipt replay
  rejection, and zero residue. The new successor summary
  `docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_DEMO_LIFECYCLE_2026-08-04-v2.md`
  was finalized from the real happy/fault ledgers and passed the immutable
  check-only verifier. The older offline summary remains historical and was
  not rewritten as connected evidence.

- [x] Task 10 — The corrected successor aggregate ladder is `verified locally`.
  The full Go package list passed, tagged `preproddemo` API tests passed, the
  explicit temporary tagged API build passed, contract generation/check and
  frontend typecheck passed, Vitest passed `89` files and `748/748` tests,
  demo/HTTP builds and both artifact boundary scans passed, and the HTTP
  artifact scan passed. The exact Node contract/boundary bundle passed
  `57/57`, Compose config and policy passed `21/21`, E2E discovery reported
  the required `7` tests in `3` files, forbidden inventory passed `129`
  objects, created-file inventory passed `through=task10 due=108 planned=108`,
  the successor summary passed its non-mutating check, docs smoke passed,
  instruction scan passed, the literal plan whitespace/fence scan passed,
  the untracked-plan no-index diff was inspected with expected exit `1`, and
  `git diff --check` passed. Task-owned browser, Vite, Go-test, Compose,
  temporary-build, and database residue checks are clean. The result remains
  `candidate-only`, `release pending`, and `production-ready: not established`.

### 2026-08-05 — Task 9 corrected connected F2/F3 verification log

- [x] F2 happy path — `verified locally` from the retained disposable
  PostgreSQL/OIDC target. The exact connected command returned
  `aga-hybrid-connected: verified locally happy-path phases=14 browser=17 residue=0`.
  The connected happy ledger is
  `sha256:5b2f03652eaef75aa6cb33a2d22789f927bf1e3b2e62b5094c47d21a098c06ec`.
- [x] F3 prepare — `node scripts/prepare-aga-hybrid-f3-manifest.mjs` created
  the closed four-case manifest from the current package, connected-config
  code digest, and connected-config contract digest. The target-bound prepare
  command stopped at the required `pending external authority` gate with exit
  `2` after creating four distinct Compose/PostgreSQL targets.
- [x] F3 deliberate stop — the exact
  `AVIA_AGA_HYBRID_F3_STOP_AFTER_CASE=CONCURRENT_TOKEN_RESERVATION` protocol
  produced child exit `73` with
  `INTERRUPTED_AFTER_CONCURRENT_TOKEN_RESERVATION`; the harness recorded
  `aga-hybrid-connected: fault-matrix interrupted-after=CONCURRENT_TOKEN_RESERVATION`.
- [x] F3 recovery — a fresh target-bound recovery manifest and authorization
  resumed the same outer journal. The exact recovery command returned
  `aga-hybrid-connected: verified locally fault-matrix cases=4 residue=0 resume=verified`.
  The independent status command returned
  `aga-hybrid-connected: fault-matrix-status=receipt-journal-present`.
  The connected fault ledger is
  `sha256:0cec9fc66a074725297ddb95a9f61f6b1c152da061ab951464777f7d4311de3c`.
  All four cases have four journal phases, `effectCount=1`,
  `duplicateEffectCount=0`, cleanup receipts, `terminalState=CLEANED`, and
  `residueCount=0`; `CONCURRENT_TOKEN_RESERVATION` has
  `winnerCount=1` and `loserEffectCount=0`.
- [x] Evidence finalizer —
  `node scripts/verify-aga-hybrid-demo-workspace-evidence.mjs --finalize-summary
  docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_DEMO_LIFECYCLE_2026-08-04-v2.md`
  returned `aga-hybrid-evidence: finalized privacy-safe summary`; the exact
  `--check-summary` command returned
  `aga-hybrid-evidence: summary check passed without mutation`. Both ledgers
  reported `sourceKind=connected-receipt`; no synthetic ledger was used.
- [x] The F3 target-mode dispatch, transaction schema binding, and stored
  target-receipt recovery idempotency findings were fixed in
  `scripts/run-aga-hybrid-f3-target-protocol.mjs`, then syntax and whitespace
  checks passed. Failed disposable Compose projects/volumes were removed by
  exact case-bound cleanup before the successful fresh root; the successful
  root ended with zero target, container, volume, and network residue.

### 2026-08-05 — Independent review remediation and final connected evidence

- [x] Sol xhigh and Luna max review findings were addressed locally. F3 now
  validates the exact outer case/namespace/fingerprint/manifest/seal and
  input/code/contract binding before any target effect, stores the
  authorization identity/digest with the PostgreSQL effect, and uses a
  create-only shared-token reservation with one recorded winner and one
  recorded no-effect loser. A losing reservation never removes the winning
  receipt.
- [x] `INHERITED_BASE_RECEIPT_GAP` and
  `WORKSPACE_TRANSACTION_RECEIPT_GAP` recover through an append-only
  `recovery_receipts` table; the original effect row is never updated.
  `CLEANUP_RECEIPT_GAP` records an actual cleanup replay rejection. The
  deliberate stop is after target disposal and case-receipt fsync but before
  outer-journal publication; recovery accepts the empty outer journal when
  the disposed case receipt exists and appends the missing publication once.
- [x] Prepare failure cleanup covers the currently-starting target, F3
  cleanup/recover-prepare branches are explicit and fresh-authorized, and
  evidence derives terminal/residue/case facts from retained target receipts.
  Provenance now binds execution, target receipts, authority-consumption
  receipts, and raw-file privacy scanning; the verifier rejects synthetic
  connected ledgers.
- [x] The corrected fresh F3 root was
  `/private/tmp/avia-aga-hybrid-f3-final-review.ZaK4xo`; its final connected
  fault ledger is
  `sha256:0cec9fc66a074725297ddb95a9f61f6b1c152da061ab951464777f7d4311de3c`.
  The exact interrupted run returned child exit `73`, recovery returned
  `aga-hybrid-connected: verified locally fault-matrix cases=4 residue=0
  resume=verified`, and the four target-owned cleanup receipts ended with
  `terminalState=CLEANED` and `residueCount=0`.
- [x] The finalizer was rerun against the retained happy ledger and this
  corrected fault ledger. The create-once successor
  `docs/demo-evidence/AGA_HYBRID_CLASSIFICATION_DEMO_LIFECYCLE_2026-08-04-v2.md`
  was finalized and its check-only verification returned
  `aga-hybrid-evidence: summary check passed without mutation`.

### 2026-08-05 — Task 10 aggregate verification log

- [x] `go -C apps/api test -count=1` with the exact nine-package list passed;
  the tagged `go -C apps/api test -count=1 -tags=preproddemo ./cmd/api` also
  passed. The first unmodified Go invocation was blocked by the sandbox’s
  global cache permission; the same exact package/test commands passed with
  the disposable `GOCACHE=/tmp/avia-aga-gocache`, which was removed afterward.
- [x] The explicit `mktemp` tagged API build passed and its exact temporary
  build directory was removed by the command trap. Contract generation,
  `contracts:check`, and `typecheck` passed. `npm --prefix apps/web test`
  passed `89` files and `748/748` tests. `build:demo` passed with the demo
  boundary scan (`82 files, 184 inputs`); `build:http` passed with the HTTP
  boundary scan (`82 files, 186 inputs`) and `http-artifact-scan` passed.
- [x] After the final META formatting pass, the exact focused command
  `env GOCACHE=/tmp/avia-aga-gocache go test
  ./internal/agaapplicability ./internal/agademoworkspace
  ./internal/preproddata/agademoworkspace -count=1` passed from the
  `apps/api` module root. The exact forbidden-inventory `--write` and
  `--check` commands both passed with `objects=129`, and the disposable Go
  cache was removed.
- [x] The exact Node bundle passed `57/57` tests. Compose config passed;
  `bash scripts/check-compose-policy.sh` passed `21/21`. The required E2E
  list reported `7 tests in 3 files`. Forbidden inventory passed
  `objects=129`; created-file inventory passed
  `through=task10 due=108 planned=108`.
- [x] The first post-build created-file inventory check exposed one stale
  historical evidence path. The tracked inventory was corrected with
  `apply_patch` to point at the create-once successor summary, and the exact
  command `node scripts/check-aga-hybrid-created-files.mjs --inventory
  tests/fixtures/aga-hybrid-created-file-inventory.v1.json --through task10`
  was rerun successfully with `due=108 planned=108`.
- [x] The exact successor `--check-summary` returned
  `aga-hybrid-evidence: summary check passed without mutation`, docs smoke
  returned `harness-docs-smoke: ok`, instruction scan returned the canonical
  harness references, and both literal plan-file scans returned
  `aga-hybrid-plan-file-scan: ok`. `git diff --check` passed. A final
  read-only residue scan found no task-owned Compose, browser, Vite, Go-test,
  temporary-build, or database processes/resources; the retained successful
  private F2/F3 ledgers are evidence only.

## Decision Log

### 2026-08-04 — Preserve supplied ZIP receipt digests as demo source evidence

**Decision:** For the demo-first Task 2 lane, remove only the former fixed
repository prompt-hash and local model-descriptor-digest equality gates from
supplied pass-ZIP reconciliation. Retain the prompt/model/input/batch/record/
pass receipt digests carried by each immutable ZIP as source evidence. Require
lexical digest validity, equal prompt digests across roles, complete identity
and count coverage, record/batch/pass receipt-graph integrity, private-pass
isolation, and text-free cleanup exactly as before.

**Reason:** The replacement candidate and challenge passes are semantically
complete and independently bounded at 25 batches and 1,310 records, while the
platform did not expose the former prompt/model pins in the same form expected
by the repository. Treating the immutable producer receipt as evidence keeps
the demo truthful without fabricating metadata or weakening identity,
cardinality, digest-graph, privacy, or isolation controls. This is
`candidate-only`; it does not establish source authority, release, or
production readiness.

### 2026-08-03 — Categorize questions by domain, not provider hierarchy

**Decision:** Remove primary/supporting provider fields. Give every question
one main domain and optional controlled topic tags; model external provider
involvement only when evidence supports an explicit role.

**Reason:** Only 30 of 1,310 rows have conservative external-provider candidates.
Provider hierarchy therefore cannot serve as useful full-package taxonomy.

### 2026-08-03 — Use a hybrid two-pass AI pipeline

**Decision:** Combine deterministic extraction/validation, a complete semantic
AI pass, an independent adversarial pass, rule-derived confidence, and Manager
review.

**Reason:** LLM-only classification is inconsistent and rules-only
classification cannot reliably capture question meaning.

### 2026-08-04 — Accept truthful platform-unavailable model metadata for the local demo

**Decision:** Reopen only the Gate 0B model-metadata availability rule by user
authorization. A platform-unavailable scalar is `null` with its exact field
name in `unavailableFields`; a displayed label remains separate and never
establishes an exact model ID. Such a descriptor is valid `candidate-only` demo
provenance and does not block deterministic Task 2 validation.

**Reason:** The ChatGPT Pro surface can truthfully expose a displayed label,
service, and interface while withholding an exact model ID, reasoning effort,
fork setting, or snapshot. Requiring unavailable platform values would either
stop a local demo unnecessarily or invite fabrication. The amendment retains
closed-schema, canonical-digest, privacy, isolation, and all non-metadata
provenance gates, and makes no release or production claim.

### 2026-08-03 — Separate discovery and classification batch manifests

**Decision:** Preserve the accepted vocabulary-discovery manifest and all model
outputs unchanged. Freeze a second text-free classification manifest over the
same ordered 1,310 authoritative private items. Size its boundaries against the
complete closed pass-input envelope at the maximum permitted run-ID lengths,
bind each role-neutral private snapshot by digest, and pin both classification
passes to this new manifest.

**Reason:** The discovery envelope is smaller than the classification envelope.
A fresh exact calculation showed that reusing its boundaries would exceed the
98,304-byte classification limit in 21 of 24 batches. Rewriting accepted model
outputs or weakening the byte limit would invalidate stronger evidence; a
separate deterministic manifest preserves discovery receipts and corrects the
downstream contract.

### 2026-08-03 — Preserve and explicitly disposition discovery-only omission cues

**Decision:** Retain the exact text-free model output preimages, but never treat
a candidate omission cue as a frozen rule match merely because the model emitted
it. Resolve every candidate to a complete identity and frozen mapping; accept it
only when its selector and deterministic matcher reconstruct, otherwise retain
an explicit digest-bound rejection reason.

**Reason:** Rewriting raw model outputs would destroy receipt integrity, while
silently dropping or auto-accepting unmatched semantic cues would manufacture
evidence. The disposition ledger preserves discovery coverage without granting
unverified candidates controlled-taxonomy authority.

### 2026-08-03 — Preselect high confidence without automatic approval

**Decision:** Use `AUTO_PROPOSED_HIGH_CONFIDENCE` only as working-Draft default.

**Reason:** Model agreement and metadata consistency can reduce review effort
but cannot create regulatory, technical, or publication authority.

### 2026-08-03 — Keep the sealed overlay and add a sibling workspace

**Decision:** Do not alter `preprod_aga_demo`; store classifications, decisions,
and synthetic lifecycle state in `preprod_aga_demo_workspace`.

**Reason:** The original accepted projection must remain byte/behavior stable,
while shared Manager edits require durable, isolated state.

### 2026-08-03 — Run the demo from simulation readiness

**Decision:** Use `READY_FOR_DEMO_SIMULATION` as the only bridge from a Manager
Draft to synthetic execution. Keep technical approval and publication separate
and blocked by current source/authority facts.

**Reason:** A fully working demo must not manufacture authority or clear the
package's explicit blockers.

### 2026-08-03 — Separate agreement confidence from governance truth

**Decision:** Use only `HIGH`, `MEDIUM`, and `LOW` for two-pass agreement;
preserve source/authority/risk states separately; apply the total recommendation
precedence; reject an entire run on structural failure.

**Reason:** Every supplied row has source and expert-review blockers. Treating
those as confidence blockers made high-confidence preselection unreachable and
left valid rows without a status.

### 2026-08-03 — Store only genuinely new workspace wording

**Decision:** Original bodies remain accepted-package/overlay-only. A Manager
add/reword command creates a separate append-only workspace-owned synthetic body/version with
parent/digest/provenance and manual-review states.

**Reason:** Immutable wording successors cannot survive reload if all workspace
text is forbidden, but copying any of the 1,310 originals would weaken the
sealed-overlay boundary.

### 2026-08-03 — Give every Draft leaf one closed question reference

**Decision:** Discriminate immutable Base references from Workspace references.
Keep the accepted six-field tuple for Base rows; give Workspace rows exact
generation/root/version/proposal IDs, order, digest, parent, actor, time, and
reason fields. Allocate add/reword IDs only on the server, preserve root/order
across rewording, and use `rootSequence` only after identity resolution.

**Reason:** A body digest, proposal ID, or ordering sequence cannot distinguish
all accepted duplicate wording or reconstruct append-only Manager successors,
reset history, CAS, snapshots, and neutral direct-ID behavior.

### 2026-08-03 — Make lifecycle authority operation-specific

**Decision:** Classification remains Admin-read/scoped-Manager-command only;
synthetic lifecycle operations use exact Manager, Inspector, Lead, reviewer,
and organization-scoped Auditee bindings. Inspector proposes a Potential
Finding; Lead conversion alone creates the synthetic Finding.

**Reason:** A shared “workspace role” both contradicted the required multi-role
lifecycle and omitted the product's Potential Finding authority boundary.

### 2026-08-03 — Use separate pools and measure auth honestly

**Decision:** Add owner/loader/reader/EXECUTE-only command roles and compose
overlay/workspace data in Go after authorization. Measure business-table and
non-auth audit-event zero delta separately from exact, column-bounded existing
OIDC/session control-plane updates on every authenticated request and logout.

**Reason:** A broad workspace writer or whole-database byte-identity claim is
not compatible with least privilege or the current session middleware.

### 2026-08-03 — Bound both AI passes and make them independently auditable

**Decision:** Both passes use the same deterministic 64-item/98,304-byte maximum
batch manifest, separate blind contexts, independent 1,310-row seals, exact
tool-reported model descriptors, and text-free normalized outputs.

**Reason:** The previous 1,087/223 numeric form split was ambiguous around
`035A`/missing `049`, operationally unbounded, and did not prove pass
independence.

### 2026-08-03 — Keep one umbrella with six top-level slices and F substops

**Decision:** Retain this file as the sole active goal and execute Gate 0,
Tasks 1–2, Tasks 3–4, Tasks 5–6, Tasks 7–8, and Tasks 9–10 as ordered internal
slices with review stops; connected Slice F is further stop-gated as F1
fault/static, F2 happy path, F3 four-case recovery, and F4 offline handoff.

**Reason:** Classification, isolated persistence/API, UI/recommendation,
lifecycle, and connected qualification are independently rejectable systems;
one uninterrupted implementation batch is not safely reviewable.

### 2026-08-03 — Keep demo authority synthetic and target-bound

**Decision:** Bind the existing synthetic OIDC subjects/memberships to a
digest-pinned workspace-only department/unit, provider scope, target, and
operation-role fixture. Do not create or impersonate canonical department or
functional assignments.

**Reason:** The predecessor fixture contains no `caa_department_memberships`,
while the demo still needs exact Department Manager, Inspector, Lead/reviewer,
and organization-scoped Auditee authorization.

### 2026-08-03 — Make confidence and sealing mechanically reconstructible

**Decision:** Replace free rationale with controlled rationale/fact/rule
references, persist both complete immutable pass projections, and use a domain-
separated, non-circular proposal/item/aggregate/run digest graph.

**Reason:** A validator cannot derive confidence from narrative or verify a
digest that recursively includes its enclosing run; a Manager also cannot
accept a challenge projection that was stored only as a digest/disagreement
code.

### 2026-08-03 — Split connected preparation from target-bound authority

**Decision:** Task 9 uses `prepare`, an external authority stop, then exactly
one of `qualify` or cleanup-only `cleanup-prepared`. OIDC fixture export and
load intents are pinned before the operator issues the qualification bundle.

**Reason:** The target fingerprint and actual subject/membership fixture do not
exist soon enough for a safe one-command authorization, and declined authority
must still have a bounded zero-residue recovery path.

### 2026-08-03 — Make Manager resolution and selection leaf-exact

**Decision:** Preserve sealed AI provenance, but demote every semantic edit to
explicit Draft review under source-gap precedence. Provide a closed complete
proposal-resolution command, require a later disposition, and select only the
current included leaf for recommendations.

**Reason:** A HIGH base row must not remain automatically disposed after a
Manager changes it, and an actionable review queue must support every proposal
family that can disagree rather than only domain/topic edits.

### 2026-08-03 — Make synthetic lifecycle branches total

**Decision:** Freeze all CAP/Evidence combinations, independent due-date
choice, append-only CAP supersession, Evidence resubmission, exact reopen/
completion transitions, and `AUTHORIZED_CLOSE` only from `PENDING_CLOSURE`.

**Reason:** CAP acceptance must never become implicit closure, and no Lead or
Auditee choice may enter an unreachable state or bypass required CAP/Evidence.

### 2026-08-03 — Journal connected authority through crashes

**Decision:** Reserve fresh target-bound phase tokens into a monotonic fsynced
journal, adapt inherited predecessor receipts without modifying their loaders,
support exact recover/status/resume/cleanup modes, and require a separately
authorized complete commit/publication fault matrix.

**Reason:** Shell traps do not survive SIGKILL or host loss, and a target effect
committed before ledger publication must be reconciled without a second effect
or a stranded disposable namespace.

### 2026-08-04 — Keep Task 1 provenance at the accepted manifest-root boundary

**Decision:** Treat Gate 0B's closed 11-field `fixedInputDigests` object as the
exact fixed-input authority, keep `sourceSnapshotDigest` on each
classification-batch entry for Task 2 reconstruction, and require the pure
Task 1 domain to pin the accepted classification-manifest root.

**Reason:** Adding `sourceSnapshotDigest` to the closed fixed-input object would
contradict the accepted schema and move Task 2 batch reconstruction into the
pure Task 1 domain. The final independent review therefore rejected that prior
provenance concern rather than weakening or expanding either boundary.

## Discoveries

- The accepted 24-batch discovery manifest bounded the discovery-only JSON
  envelope, not the larger closed classification-pass envelope. Exact
  reconstruction with valid pass metadata measured 21 over-limit batches, a
  99,653-byte maximum, and 2,338,919 bytes total per pass. Maximum-valid run-ID
  sizing then proved the deterministic greedy classification partition requires
  25 batches (maximum 98,239 bytes); the existing 24 discovery boundaries
  cannot be reused for Task 2.

- The existing untracked `agaapplicability` test suite has no implementation
  file and encodes an earlier form-wide provider-profile design. Its useful
  identity, provenance, exact-selection, and immutable-successor behaviors can
  be preserved while replacing that taxonomy.
- The existing `new-audit-wizard.tsx` creates only a governed Planning item and
  currently contains Cabin Safety/Fly Namibia mock selections. The AGA demo
  path must remain profile-gated and must not turn that real planning workflow
  into a synthetic Audit bridge.
- The existing AGA overlay has distinct reader/writer/normal-API database roles
  and immutable triggers. Reusing its writer or granting Department Manager
  direct access would invalidate the accepted boundary.
- Fresh Codex contexts alone do not prove blind classification because agents
  share the working tree. Both passes need isolated read roots/private sinks
  and must finish before either output is materialized in that shared tree.
- The canonical Finding vocabulary includes `CAP_ACCEPTED`, while the workflow
  still makes CAP acceptance non-closing and treats CAP, Evidence, and due-date
  requirements as independent Lead choices. The synthetic projection uses an
  explicit canonical subset and maps accepted CAP directly to the appropriate
  still-open Evidence/closure state; this is a projection choice, not a claim
  that the canonical value is absent.
- The inherited predecessor base and overlay loaders already own their target-
  bound authorization/control receipts. Crash-safe hybrid orchestration must
  reconcile those receipts rather than modify the accepted loader transaction.
- Pass result digests and disagreement codes cannot support a later Manager
  choice of the challenge projection. Both complete text-free pass projections
  must remain immutable and resolvable by full identity/run/digest.
- Exact idempotent replay may bypass a now-reset generation CAS, but it cannot
  bypass current membership, organization, operation-role, or workspace-scope
  authorization. The binding therefore needs a closed authorization-scope
  digest independent of the domain object.
- Resetting a generation with an open Inspection/Finding/CAP/Evidence branch
  would strand ordinary actors because old-generation objects are intentionally
  neutral-denied. Reset must reject until every lifecycle root is terminal.
- `GET_ROLE_HISTORY` contains CAA binding/assignment chronology and cannot share
  the otherwise Auditee-readable lifecycle-query projection.
- The `preproddemo` API is a build-tagged five-route-only artifact. Normal Go
  tests skip its profile file, and its `ProtectReadOnlyNeutral` middleware does
  not validate CSRF because it is intentionally GET-only. Workspace mutations
  therefore require a separate protector plus explicit tagged tests/build.
- The current Admin AGA panel eventually loads all 1,310 bodies into memory.
  The accepted raw panel remains unchanged, but the new Manager workspace must
  use its own bounded server-pagination client and cannot reuse that loop.
- The React app has only `demo` and `http` build profiles and a frozen 86-route
  parity registry; there is no preprod build-profile discriminator or
  multi-role `RoleGuard`. The workspace needs a server-capability-gated
  supplemental registry rather than a false `preprod` route flag.
- The `preprod-aga-demo` Playwright project starts no web server and currently
  matches only two old specs. New specs must be added to `testMatch`, proven
  nonzero with `--list`, and executed only inside the connected harness.
- The normal session middleware updates `session_references` during
  every authentication and POST protection forces fresh provider observation,
  which can update observation columns in `desired_membership_sync` as well as
  login/logout audit state. A whole-database byte-identical browser claim is
  impossible; the corrected plan closes and measures business versus exact
  auth-control-plane objects/columns.
- The smoke OIDC fixture creates nine current role memberships but no canonical
  CAA department membership. Exact demo department/unit authority must
  therefore remain a separately sealed workspace-only binding, not a fabricated
  `Principal.DepartmentAssignments` value.
- The current OpenAPI bundler treats every POST as a mutation and automatically
  adds idempotency/revision headers plus 401/403 responses. The workspace needs
  explicit query/command and neutral-denial metadata plus a bundler regression
  test.
- The React HTTP request closure is private to `createHttpBackend`; safely
  preserving API-base, CSRF, auth-loss, no-store, and telemetry behavior
  requires extending `Backend`/`http-backend.ts`, not an independent fetch
  client.
- The current Compose policy script does not render the candidate/workspace
  preprod profiles. The corrected static gate renders the exact combined
  profile set and checks the workspace services/secrets without starting them.
- No workspace provisioner/loader command or Docker/Compose target currently
  exists. Prior overlay run 47 was cleaned, so connected qualification must
  rebuild both sealed schemas from the exact fixed inputs.
- The external question CSV currently duplicates the same candidate provider
  in operational-interface and Evidence-contributor fields. Precise involvement
  role requires new question-level analysis rather than mechanical import.
- The supplied JSON has no per-row `NON_AUTHORITATIVE_CANDIDATE` literal, and
  text digests are not unique. The corrected contract uses the global
  `candidateOnly` fact and the complete six-field identity.
- The six-field Base identity does not type a Manager-added or reworded row.
  The corrected plan now owns a closed Base/Workspace reference union, exact
  add/reword transitions, parent-chain/current-leaf reconstruction, reset
  history, and neutral direct-ID tests; no digest or `rootSequence` is a
  substitute identity.
- The target ExecPlan is untracked. Ordinary `git diff`/`git diff --check`
  inspect the tracked index change but not this file, so final review requires a
  direct target-file scan and a no-index diff inspection.
- All current questions remain source-mapping-required/not-attested, so a
  truly published executable checklist cannot be produced from this package.
  Simulation readiness is the only truthful full-demo boundary.

## Outcome

The intended result is a shared, fully functioning local-preprod simulation
whose complete 1,310-question classification is reproducible, reviewable, and
immutable by revision; whose checklist selection is exact and fail-closed; and
whose full lifecycle remains isolated from every real governed table and
authority decision. Local success does not establish source authority,
release, deployment, or production readiness.

candidate-only
release pending
production-ready: not established

## Execution Prompt

Use this prompt only after the user explicitly authorizes implementation:

```text
Execute only the currently authorized child slice in
docs/exec-plans/active/2026-08-03-aga-hybrid-classification-demo-lifecycle-plan.md.
Gate 0A, Gate 0B, Task 1, Task 2, Tasks 3–8, Task 9 F1/F2/F3, and Task 10 are
already recorded. The current implementation scope is review remediation and
final local handoff; do not replay the connected happy path or fault matrix
unless a fresh disposable target and fresh target-bound authorization are
explicitly created. Record exact
focused-test and verification output. Preserve every
unrelated dirty change, the accepted package, the sealed preprod_aga_demo
schema and five Admin-only GET routes, all source/authority/risk blockers, and
the exact accepted fixed-input hashes plus immutable supplied receipt hashes.
Do not use digest alone as identity, invent model metadata, rebind a supplied
ZIP receipt to a former prompt/model pin, expose pass one to pass two, create
provider ownership fields, or extend a frozen taxonomy during a run. When separately authorized, execute
ordered A–E slices and stop-gated F1–F4 substops (only F2–F3 are connected) in
order and never begin a later boundary before review of its predecessor. The
sibling workspace may store original identities/digests and only genuinely
new/reworded synthetic bodies; it must use separate least-privilege pools,
exact operation-role projections, Potential Finding/Lead conversion,
deterministic recommendation/readiness pins, append-only events, and
forward-only reset. Never write canonical provider, identity, membership,
assignment, attestation, decision, publication, Planning, Audit, Finding, CAP,
Evidence, notification, outbox, delivery, release, or production records
through the workspace data plane. Stop on any identity, digest, count,
taxonomy, provenance, authority, CSRF, privacy, forbidden-object, concurrency,
replay, recovery, discovery, or residue mismatch. Do not commit, push, deploy,
call external systems, change branches, or change a real database without
separate exact authorization. Keep this umbrella and its single index row
current and use only literal verified locally, not run, blocked,
candidate-only, release pending, and production-ready: not established
evidence.
```
