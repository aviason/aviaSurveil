# AGA all-form source and risk review draft

**Package status:** `PENDING_ADMIN_AND_SOURCE_OWNER_REVIEW`
**Product status:** `candidate-only`
**Release status:** `release pending`
**Production readiness:** `production-ready: not established`

This is a bounded review package for all AGA forms present in the supplied
archive. It is not an import, a regulatory opinion, a source-owner attestation,
or a publication request. The package contains derived inventory/provenance,
printed reference strings, and parser-derived candidate question strings.
It does not contain raw AGA ZIP/PDF byte artifacts, page images, or full PDF
text dumps.

## Inventory receipt

- Archive: `53` PDF entries (`1` register and `52` forms).
- Archive SHA-256:
  `sha256:dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32`
- Archive size: `12,227,415` bytes.
- Register SHA-256:
  `sha256:29ed8384693b615926fc42a0ca4654be2ea9a36b0946f217975571ca0ad9564f`.
- The register has no `FSS-AGA-FORM-049`; it contains `FSS-AGA-FORM-035A`.
- All `52` form identities and file hashes are in
  `AGA_ALL_FORMS_REGISTER.csv` and the JSON package.
- The parser found `1,310` question-shaped candidate boundaries across `31`
  forms. The other `21` forms remain in the inventory as application,
  appointment, information, assessment, or other form records without a
  detected protocol-question boundary.
- Form 048 is the only exact source-backed vertical slice in this package: it
  retains its `28` questions from the previously hash-bound Form 048 packet.
  All other question rows are parser candidates and must be reviewed.
- The package contains `174` unique form-level or question-level
  NAMCAR/NAMCATS reference strings. These are references printed in the AGA
  material, not proof that a source mapping is correct.

## What each file means

- `AGA_ALL_FORMS_SOURCE_RISK_DRAFT.json` is the complete machine-readable
  review object: inventory, question candidates, source proposals, proposed
  risk bands, state boundaries, and future-change rules.
- `AGA_ALL_FORMS_REGISTER.csv` is the 52-form inventory, including immutable
  archive/file hashes and the candidate question count.
- `AGA_ALL_FORMS_QUESTION_MAPPING_RISK_QUEUE.csv` is the review queue. Every
  row is `SOURCE_MAPPING_REQUIRED`, `NOT_ATTESTED`,
  `CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW`, and `NOT_SUPPLIED` for
  the decision state.
- `AGA_ALL_FORMS_SOURCE_COVERAGE.csv` deduplicates all source references and
  records the proposed source document, clause locator, local hash/page when
  available, and the remaining authority gap.
- `TASK9_ALL_FORMS_EXPANSION_AUTHORIZATION_TEMPLATE.json` is intentionally
  blank. It records the fields a real Admin must supply before any bounded
  candidate import may be considered.
- `FURKAN_MESSAGE_DRAFT.tr.md` is the Turkish handoff message to send with
  this ZIP.
- `MANIFEST.sha256` is the package-file integrity manifest.

## Source-authority boundary

The local source-vault manifest identifies the current NAMCATS Part 139
candidate source as `NCAA-NAMCATS-P1-PDF-04` with its recorded immutable hash.
NAMCARs Part 139 (2023) is represented only by an official URL proposal in
this package because its exact bytes are not in the local/tracked source
manifest. Its `sourceSha256` is therefore `null` and its state is
`SOURCE_BYTES_NOT_LOCALLY_HASHED_SOURCE_OWNER_ATTESTATION_REQUIRED`. A source
owner must supply and attest the exact current bytes, effective date, and
clause/page before the queue can become source-backed.

An AGA reference such as `NAMCARs 139.17.2` or `NAMCATS 139.17.2` is kept
separate from regulatory authority. The proposal does not claim that the
referenced clause is current, applicable, complete, or legally controlling.

## Risk boundary

The four `PROPOSED_*` bands are a transparent review aid derived from wording
signals only:

1. `PROPOSED_SAFETY_CRITICAL`
2. `PROPOSED_HIGH_OPERATIONAL`
3. `PROPOSED_CONTROL_ASSURANCE`
4. `PROPOSED_REVIEW_REQUIRED`

They are not an approved product taxonomy, finding severity, legal
classification, enforcement decision, or automatic `safetyCritical` fact.
Every risk row remains
`CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW`; a human must confirm or
replace the band and rationale.

## Required review before any import

The current Admin/source owner should, in order:

1. Confirm the identity, title, form code, version/date, and scope of each of
   the 52 forms. Confirm that a zero-question row is truly a non-protocol form
   rather than an extraction miss.
2. Review every question boundary. Supply an actor-bound `ACCEPT`, `SPLIT`,
   `MERGE`, `TRANSCRIBE`, or `EXCLUDE` decision tied to the exact proposal ID,
   text digest, PDF page/locator, and reason.
3. Bind each reference to the exact current official document, immutable
   source hash, effective date, clause, page/locator, applicability, and named
   source owner. Resolve the NAMCAR Part 139 source-byte gap.
4. Confirm or correct the proposed risk band and safety-critical flag with a
   reason. No row may be treated as an approved finding severity.
5. Provide a named, bounded Phase 2 authorization: form IDs, batch limit,
   actor, decision ID, and explicit candidate-only scope. Functional
   assignment provisioning, Department Manager publication, and production
   release remain separate gates.

The future-change rule is immutable source/effective-date snapshots plus an
impact-review Draft. A changed source must not silently mutate a published
checklist or a pinned Audit snapshot.

## Explicit non-authorizations

This package does not authorize or establish source authority, regulatory
applicability, functional assignment, Admin decisions, Department Manager
technical approval, publication, deployment, release, external upload, or
production readiness. These remain `blocked` until the responsible human gate
is completed.
